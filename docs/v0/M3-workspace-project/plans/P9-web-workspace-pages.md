# M3/P9 工作区的页面 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 工作区的页面接上 P8a、P8b 的数据层，行为按 M3 设计第 2 节的故事 W1–W9：落点的页面一侧（切换工作区、创建工作区、接受邀请之后写 `last_workspace_id`，删除、离开之后回到落点）、新手引导（资料；还没有工作区的人创建工作区、邀请成员并复制链接；已经有工作区的人资料一步之后完成）、邀请页和带邀请的注册、工作区首页和侧边栏的工作区部分、项目导航对话框、工作区设置 general 和 members（成员、邀请）、停用弹窗的 409、`WorkspaceAuthWrapper` 的界面。页面发出工作区一侧的修改之后的跳转、提示和界面变化只在发出它的会话里进行；每个请求体由表单编辑的字段构成，类型取自生成的客户端，数字就是数字；每个错误经 `errorMessageKey` 显示，没有吞进 `console.error` 的拒绝；取决于 store 所持的值的修改在轮到它发出时才算出（E5）。W1–W9 的页面版本、W3 的会话切换、S2 的挂载清单。

**Architecture:** 只改 web、`e2e/`、两处静态检查（根目录 `.oxlintrc.json` 的两条 `overrides`、`tools/keywords.json` 的一条规则）和三份文档（README、前端改动清单、M2 收尾交接的 P9 处理结果）。页面调用 store 的方法，不调 service；会话的每个取数经 `useSessionSWR`。共用的新部分：`core/lib/in-session.ts` 的 `followInSession(change, { done?, failed })`（`sessionGuard` 旁边：发出修改，只在发出它的会话里跟进结果；Task 1）；测试的共用部分 `core/lib/auth/fake-tab.ts`（标签页的会话、换账户、由测试决定何时兑现的修改）和 `core/lib/fake-refusal.ts`（nerve 的拒绝）；纯函数的决定都返回 `kind` 的联合，用 `it.each` 测：`invitation-view.ts`（邀请页显示什么，Task 4）、`onboarding-place.ts`（新手引导的位置，Task 6）、`invite-modal/refusal.ts`（批量邀请的拒绝落到哪一行，Task 3）、`use-create-workspace.ts` 的 `creationRefusal`（Task 5）。页面一侧的 hook：`useMembershipChanges`（Task 2）、`useInvitationChanges`（Task 3）、`useInvitationAnswer`、`useOpenWorkspace`（Task 4）、`useCreateWorkspace`（Task 5，`/create-workspace` 和新手引导共用）、`useCopyInvitationLink`（Task 3，Task 6 也用）、`useProjectNavigationPreferences`（Task 8 重写）、`useWorkspacesFetch`（Task 10，调用者的工作区列表的唯一取数）。store 改两处：`WorkspacePreferencesStore.updatePreferences(slug, change)` 收 `(held) => WorkspacePreferencesUpdate`，轮到它时作用于 nerve 最近一次的回答（Task 8，照 P8b 的 `ProjectPreferencesStore`）；资料 store 的 `finishUserOnboarding` 不再把列表中的第一个工作区写成上次打开的（Task 6）；M2 的 `UserStore.deactivateAccount` 交回它是否结束了标签页的会话（Task 9）。静态检查：P9 的新模块和它们的测试加进 `typescript/no-non-null-assertion` 的范围，取工作区列表的三个文件加进 `no-restricted-imports` 的范围，都在加入它们的 Task（总体设计 7.7）；关键词守卫的规则 `workspaces-list-fetch` 让 `useWorkspacesFetch` 是列表唯一的取数（Task 10）。契约、Go 代码和生成的文件都不变；不加 npm 包，锁文件不变。

**Tech Stack:** React 19.2.8、React Router 8.3.0、MobX 6.12.0、SWR 2.4.2、openapi-fetch 0.17.0、TypeScript 5.8.3、vitest 4.1.11、oxlint 1.51.0、oxfmt 0.35.0、knip 6.37.0、turbo 2.10.11；Node 24、pnpm 11.10.0、Playwright 1.63.0；Go 1.27.1（Go 代码不变）。版本都由 `pnpm-lock.yaml` 固定。

**Spec:** `docs/v0/M3-workspace-project/specs/P9-web-workspace-pages.md`（上级：`docs/v0/M3-workspace-project/M3-design.md`）

## Global Constraints

- **依赖**：不加任何 npm 包或 Go 模块，不执行 `go get`、`pnpm add`。`git diff --stat 4b1334a5 -- server/ api/ pnpm-lock.yaml` 在本 plan 的任何时刻都没有输出。
- **每个 Task 提交前**：`make lint-web`（关键词守卫、`tsc`、oxlint 在上限、格式、`en` 与 `zh-CN` 的键一致）、`make knip`、`make test-web`、`make e2e` 都通过。本 plan 不改 Go 代码和接口描述，不生成任何文件，不执行 `make gen`。
- **e2e 的数目**：在 worktree 中 `make e2e` 必须全部通过：开始时 75 个；Task 1 起 77、Task 2 起 79、Task 3 起 80、Task 4 起 82、Task 5 起 83、Task 6 起 88、Task 7 起 90、Task 8 起 91、Task 9 起 92、Task 10 起 93。只在不是 git 仓库的副本里，S3 因为构建读不到提交而失败（P4b spec F4），其余照上面的数目减一全部通过。
- **容器**：`make e2e` 用自己的 testcontainers；机器忙时偶尔起不来，等 Docker 空闲之后重跑一次再当作失败。容器测试一次只跑一套。开发库 `nerve-dev-db-1` 可以用，但不要停止或重建它，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`、`nervewiki-*`）。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合，只读的命令也不行。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。不把副本的 `node_modules` 链接到 worktree 的。
- **规则**（总体设计 7.7，P8a 的裁定 F-1–F-10，P8b 的一条）：会话的每个取数经 `useSessionSWR`（不传配置），没有权限时键为 `null`；页面只取它的角色能读的（成员页的邀请只为管理员取）；修改经 store 的方法，一个接一个，store 写入 nerve 的回答；请求体取决于 store 所持的值时，轮到它发出时从 nerve 最近一次回答算出，页面不按它此刻显示的值算请求体（Task 8）；`SessionChangedError` 不是认证失败。页面发出工作区一侧的修改（删除、改名、成员、邀请、接受和忽略、创建、新手引导的步骤、导航设置、导览的结束）之后的跳转、提示和界面变化经 `followInSession` 只在发出它的会话里进行（M3 设计 7.1；另一个标签页换了账户之后，页面已是那个账户的）；每一处跟进由它自己的 vitest 守着（换账户之后兑现的 `lateSettlings`）；新手引导的资料一步只在发出名字的会话里把引导交给根（Task 6），它自己的提示和个人设置的三页（M2 的账户修改）不在本 plan 中，见 spec 第 5 节；会话变化时随页面卸载的组件状态（停用弹窗的原因，Task 9；确认框的关闭、按钮的加载）不另加核对，理由见 spec 第 3 节。请求体由表单编辑的字段构成，类型是生成的 `…Create`、`…Update`，角色是数字（`CustomSelect` 的值是 `any`，P8a 的 F-9）；每个请求体有一个测试钉住它（vitest 或端到端的 `sentTo`）。每个错误经 `errorMessageKey`（提示）或 `FIELD_ERROR_MESSAGES`（字段下方）显示，被拒绝的修改结束它的加载状态。类型只来自生成的客户端：不写重述契约的类型，没有新的 `as`、`any`、`!`（spec 附录 A.7）。删除的代码删干净（组件、两种语言的文案、工具函数、常量），不加 knip 的忽略、开关或桩。本 plan 写或重写的文件都在约 400 行以内（最终原型上量的：最长的是 `e2e/stories/workspace/w3-workspace-settings.spec.ts` 395 行、`e2e/fixtures/assert/workspace.ts` 390 行、`e2e/stories/smoke/s2-web-app.spec.ts` 379 行；web 中最长的新文件是 `use-create-workspace.test.ts` 165 行）；只为使用方改到的 Plane 文件不变长，例外一个：`workspace/settings/members-list.tsx` 89 → 91（Task 2：成员行收地址的工作区，格式化把一行拆成三行）。
- **oxlint**（M3 设计 7.9，裁定 R3）：有手改的文件（本 plan 的每个 TS 文件都是手改的，没有机械步骤）在它的 Task 提交时没有 oxlint 警告。上限：web 360 → 359（Task 3：`use-workspace-invitation.tsx` 的一条清零），其余包不变。没有新的 oxlint 抑制，删掉两条（Task 6：新手引导的根原来以 `useEffect` 决定第一步，抑制 `react-hooks/exhaustive-deps`；Task 8：对话框的 `unicorn/consistent-function-scoping`）。
- **注释**：TS 代码、测试、JSON 的说明用英文；中文文档照本 plan 原样。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的）。

  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从 `4b1334a5` 起按顺序核对过全部块：每个 `old` 恰好出现一次（在它之前的块应用之后的文件中），每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同（spec 附录 A.10）。可以用 `node <planapply.mjs> <本 plan> apply <仓库根> <n>` 写入第 n 个 Task 的块，也可以手工照抄。
- **过渡版本**：一些文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改：根目录 `.oxlintrc.json`（Task 1–6、8–10，每个 Task 加它的文件）、`core/lib/in-session.ts`（Task 1、2）、`core/lib/invitation-link.ts` 和它的测试（Task 3、4）、`onboarding/root.tsx`（Task 6、10）、两种语言的 `workspace-settings.json`（Task 1、3）和 `workspace.json`（Task 4、5、6、10），以及 `e2e/fixtures/workspace-pages.ts`（Task 1、2、3）、`e2e/fixtures/api.ts`（Task 1、3）、`e2e/fixtures/assert/workspace.ts`（Task 2、3）、S2（Task 1、2、6、10）、W1（Task 5、6）、W2（Task 1、2）。每个过渡版本都在逐 Task 复现中运行过。
- **变异**：每个 Task 末尾的"变异"表列出：把代码改坏的方式、必须因此失败的检查和它所在的层（静态：`make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫；vitest：`make test-web`；端到端：`make e2e` 的故事）。它们在最终的原型上逐个跑过（`$M3TMP/p9tools/mutants_p9.py`，`mut.py` 在它写的每个检查上各跑一次），也在各自 Task 的树上跑过（spec 附录 A.2）；表中标"（之后的 Task 起）"的检查只在之后的 Task 加入的测试中才让它失败。**会话、权限或取数的性质只由评审才能发现的，算缺口**（brief）：表中每一条这类性质都有一个会失败的检查。
- **评审敏感**（M3 设计 12 节 P9 的评审重点）：页面级的副作用只在发出修改的会话里（Task 1 的 W3 会话切换：`route.fetch()` 在换账户之前，`route.fulfill()` 在之后；去掉 `followInSession` 的核对时它失败，见 Task 1 的变异 `T1.1`）；成员页的邀请只为管理员取（Task 2，S2）；角色是数字（Task 2、3）；导航设置的修改在轮到它时算，开关也是（Task 8）；邀请页不显示被邀请的邮箱（Task 4）。改动这些之前，先照"变异"表确认它在所说的性质去掉之后失败。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录。一个文件由多个 Task 修改时，"Task"一列都列出。

| 文件 | 职责 | Task |
|---|---|---|
| `web/apps/web/core/lib/in-session.ts` | `followInSession`：发出修改，只在发出它的会话里跟进结果（Task 1）；`done` 可省（Task 2） | 1、2 |
| `web/apps/web/core/lib/in-session.test.ts` | `sessionGuard`、`followInSession` 的 vitest，改用 `fake-tab.ts` | 1 |
| `.oxlintrc.json` | P9 的新模块和它们的测试加进非空断言的范围，取工作区列表的文件加进 `no-restricted-imports` 的范围，各在加入它们的 Task（总体设计 7.7） | 1、2、3、4、5、6、8、9、10 |
| `web/apps/web/core/lib/auth/fake-tab.ts` | 测试的共用部分：标签页的会话、换账户、由测试决定何时兑现的修改（`fake-browser.ts` 的 `gate()`）、换账户之后兑现的两种方式 | 1 |
| `web/apps/web/core/lib/fake-refusal.ts` | 测试的共用部分：nerve 的拒绝（`ApiError`，状态、`code`、字段错误） | 1 |
| `web/apps/web/core/components/workspace/delete-workspace-modal.tsx`、`web/apps/web/core/components/workspace/delete-workspace-modal.test.tsx` | 删除工作区的弹窗自己带表单（整个文件）：按提交的值确认，在会话里跟进，拒绝按 `code` 提示；它的 vitest（9.5） | 1 |
| `web/apps/web/core/components/workspace/delete-workspace-form.tsx` | 删除工作区的表单（并入弹窗，删除） | 1 |
| `web/apps/web/core/components/workspace/delete-workspace-section.tsx` | 弹窗收地址的工作区 | 1 |
| `web/apps/web/core/components/workspace/settings/workspace-details.tsx`、`web/apps/web/core/components/workspace/settings/workspace-details.test.tsx` | general 页：工作区按地址找，修改只发表单的三个字段，规模没有时为 `null` 的受控值，拒绝按 `code` 提示，在会话里跟进，复制地址失败有提示；它的 vitest | 1 |
| `web/packages/i18n/src/locales/en/workspace-settings.json`、`web/packages/i18n/src/locales/zh-CN/workspace-settings.json` | 删除工作区之后的说明，删除失败的通用文案删除（Task 1）；已是成员、已有邀请、已忽略的文案（Task 3） | 1、3 |
| `web/apps/web/core/components/workspace/settings/use-membership-changes.ts`、`web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts` | 成员页的三个修改（改角色、移出、离开）：拒绝按 `code` 提示，在会话里跟进；它的 vitest | 2 |
| `web/apps/web/core/components/workspace/workspace-roles.ts` | 工作区角色的选项（编号、文案），成员页和邀请共用 | 2 |
| `web/apps/web/core/components/workspace/settings/member-columns.tsx`、`web/apps/web/core/components/workspace/settings/useMemberColumns.tsx`、`web/apps/web/core/components/workspace/settings/members-list.tsx`、`web/apps/web/core/components/workspace/settings/members-list-item.tsx` | 角色列不再用表单、不再 `as`；成员行经 `useMembershipChanges`，地址的工作区由列表传入 | 2 |
| `web/apps/web/core/lib/invitation-link.ts`、`web/apps/web/core/lib/invitation-link.test.ts` | 邀请链接的拼法（Task 3，9.5）；从链接去登录、注册的地址（Task 4） | 3、4 |
| `web/apps/web/core/components/workspace/invite-modal/refusal.ts`、`web/apps/web/core/components/workspace/invite-modal/refusal.test.ts` | 批量邀请被拒绝时，每个字段错误落到哪一行、说什么 | 3 |
| `web/apps/web/core/components/workspace/members/invite-modal.test.tsx` | 邀请弹窗的 vitest：请求体、行的错误、拒绝、会话 | 3 |
| `web/apps/web/core/components/workspace/invite-modal/fields.tsx` | 邀请的一行：角色选择给出编号，错误来自表单 | 3 |
| `web/apps/web/core/components/workspace/members/invite-modal.tsx` | 弹窗收 `invite`，`onSent` 是它的关闭，关闭之后自己清空表单 | 3 |
| `web/apps/web/core/hooks/use-workspace-invitation.tsx` | 生成的 `WorkspaceInvitationsCreate`，在会话里跟进，行的错误经 `setError`，发出的邀请交给 `onSent`（整个文件） | 3 |
| `web/apps/web/core/components/workspace/settings/use-invitation-changes.ts`、`web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts` | 邀请列表的两个修改（改角色、删除）：拒绝按 `code` 提示，在会话里跟进；它的 vitest | 3 |
| `web/apps/web/core/components/workspace/settings/invitations-list-item.tsx` | 已忽略的邀请标"已忽略"、只能删除；修改经 `useInvitationChanges`；复制经 `useCopyInvitationLink` | 3 |
| `web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx` | 成员页把 `invite` 交给邀请弹窗 | 3 |
| `web/apps/web/package.json` | web 的 oxlint 上限 360 → 359 | 3 |
| `web/apps/web/core/hooks/use-open-workspace.ts`、`web/apps/web/core/hooks/use-open-workspace.test.ts` | 进入刚加入的工作区：写上次打开的工作区（尽力而为），再跳转，在会话里跟进；它的 vitest | 4 |
| `web/apps/web/core/components/workspace/invitation-view.ts`、`web/apps/web/core/components/workspace/invitation-view.test.ts` | 邀请页显示什么：纯函数，`kind` 的联合；它的 vitest | 4 |
| `web/apps/web/core/components/workspace/use-invitation-answer.ts`、`web/apps/web/core/components/workspace/use-invitation-answer.test.ts` | 邀请页的接受、忽略：在会话里跟进（打开工作区、重读邀请、邮箱不一致），拒绝按 `code` 提示；它的 vitest | 4 |
| `web/apps/web/app/(all)/workspace-invitations/page.tsx` | 邀请页（整个文件）：查看、接受、忽略（经 `useInvitationAnswer`）、邮箱不一致、已忽略、链接无效 | 4 |
| `web/packages/i18n/src/locales/en/workspace.json`、`web/packages/i18n/src/locales/zh-CN/workspace.json` | 邀请页的文案（Task 4）；slug 的两条文案和创建关闭的文案（Task 5）；创建失败的提示删除（Task 6）；"找不到工作区"的文案（Task 10） | 4、5、6、10 |
| `web/apps/web/core/components/workspace/use-create-workspace.ts`、`web/apps/web/core/components/workspace/use-create-workspace.test.ts` | 创建工作区：nerve 的 slug 检查决定发不发，请求体由表单构成，拒绝落到字段或提示，在会话里跟进；它的 vitest | 5 |
| `web/apps/web/core/components/workspace/create-workspace-form.tsx`、`web/apps/web/app/(all)/create-workspace/page.tsx` | 表单和页面改用 `useCreateWorkspace`、`useOpenWorkspace`；slug 字段显示的就是发出的值；创建关闭时页面说明已关闭，给实例管理员写信的按钮删除 | 5 |
| `web/apps/web/core/components/onboarding/onboarding-place.ts`、`web/apps/web/core/components/onboarding/onboarding-place.test.ts` | 新手引导从哪一步接着做、资料一步之后去哪：纯函数；它的 vitest（9.5） | 6 |
| `web/apps/web/core/components/onboarding/root.test.tsx` | 新手引导写什么、何时完成的 vitest | 6 |
| `web/apps/web/core/components/onboarding/root.tsx` | 新手引导取工作区列表，按它决定步骤，修改在会话里跟进（Task 6，整个文件）；列表的取数改用共用的 hook（Task 10） | 6、10 |
| `web/apps/web/core/components/onboarding/steps/root.tsx`、`web/apps/web/core/components/onboarding/steps/team/root.tsx` | 步骤按位置显示；邀请成员一步用成员页的邀请表单（整个文件） | 6 |
| `web/apps/web/core/components/onboarding/steps/team/links.tsx` | 发出邀请之后列出每个邀请的链接供复制 | 6 |
| `web/apps/web/core/components/onboarding/steps/workspace/create.tsx`、`web/apps/web/core/components/onboarding/steps/workspace/index.ts`、`web/apps/web/app/(all)/onboarding/page.tsx` | 创建一步改用 `useCreateWorkspace`；页面的说明 | 6 |
| `web/apps/web/core/components/onboarding/steps/profile/root.tsx`、`web/apps/web/core/components/onboarding/steps/profile/root.test.tsx` | 资料一步只在发出名字的会话里把引导交给根；它的 vitest | 6 |
| `web/packages/constants/src/workspace.ts` | 角色的说明字段删除（没有读者） | 6 |
| `web/apps/web/core/components/onboarding/steps/workspace/root.tsx` | "创建或加入工作区"的包装组件（删除） | 6 |
| `web/apps/web/core/hooks/use-copy-invitation-link.ts` | 复制邀请链接，成功、失败都有提示：邀请列表和新手引导共用 | 3 |
| `web/apps/web/core/store/user/profile.store.ts` | 完成引导不再写列表中第一个工作区 | 6 |
| `web/packages/utils/src/validation.ts` | `validateSlug`、`SLUG_REGEX` 删除 | 6 |
| `web/packages/i18n/src/locales/en/common.json`、`web/packages/i18n/src/locales/zh-CN/common.json` | 新手引导的文案：创建关闭（P7）、邀请的链接；邀请一步的两条旧文案和角色的三条说明删除 | 6 |
| `web/apps/web/core/components/account/auth-forms/auth-root.tsx`、`web/apps/web/core/components/account/auth-forms/password.tsx`、`web/apps/web/core/components/auth-screens/header.tsx` | 注册带上链接的邀请；登录页和注册页之间的链接保留查询参数 | 7 |
| `web/apps/web/core/store/workspace/preferences.store.ts`、`web/apps/web/core/store/workspace/preferences.store.test.ts` | 导航设置的修改是 `(held) => update`，轮到它时作用于 nerve 最近的回答；没有设置时不发出 | 8 |
| `web/apps/web/core/hooks/navigation-preferences.ts`、`web/apps/web/core/hooks/navigation-preferences.test.ts` | nerve 的默认值；修改在轮到它时算出，开关在那时决定打开或关闭；数量框的草稿（整个文件） | 8 |
| `web/apps/web/core/hooks/use-navigation-preferences.ts`、`web/apps/web/core/hooks/use-navigation-preferences.test.ts` | 侧边栏读设置，对话框改设置：在会话里跟进，拒绝按 `code` 提示（整个文件）；它的 vitest | 8 |
| `web/apps/web/core/components/navigation/project-navigation-dialog.tsx` | 数量在失去焦点时发出，框里显示设置的数量，标签关联输入框 | 8 |
| `web/apps/web/core/components/sidebar/sidebar-wrapper.tsx`、`web/packages/i18n/src/locales/en/accessibility.json`、`web/packages/i18n/src/locales/zh-CN/accessibility.json` | 打开项目导航对话框的按钮有名字 | 8 |
| `web/packages/types/src/navigation-preferences.ts` | `DEFAULT_PROJECT_PREFERENCES` 删除 | 8 |
| `web/apps/web/core/components/home/root.tsx`、`web/apps/web/core/components/home/root.test.tsx` | 首页的导览结束在会话里跟进，拒绝按 `code` 提示；它的 vitest | 8 |
| `web/apps/web/core/components/account/deactivate-account-modal.tsx`、`web/apps/web/core/components/account/deactivate-account-modal.test.tsx` | 停用被拒绝的原因显示在弹窗里；停用结束了标签页的会话时才说已停用、关闭；它的 vitest | 9 |
| `web/apps/web/core/store/user/index.ts`、`web/apps/web/core/store/user/index.test.ts` | 停用交回它是否结束了标签页的会话（`endSession` 的回答）；它的 vitest | 9 |
| `web/apps/web/core/hooks/use-workspaces-fetch.ts` | 调用者的工作区列表的取数，落点、工作区的页面、新手引导、个人设置共用 | 10 |
| `tools/keywords.json` | 规则 `workspaces-list-fetch`：`useWorkspacesFetch` 是列表唯一的取数 | 10 |
| `web/apps/web/core/lib/use-landing.ts`、`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts`、`web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx` | 列表的取数改用共用的 hook；个人设置自己取列表（P8a 评审的 M7） | 10 |
| `web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`、`web/apps/web/core/layouts/auth-layout/workspace-wrapper.test.tsx` | "找不到工作区"的界面和退出失败的提示经 `t()`；测试读键 | 10 |
| `README.md`、`docs/v0/frontend-changes.md`、`docs/v0/M3-workspace-project/handoffs/M2-closeout.md` | "前端"一节的能用的页面；前端改动清单 3.1、3.2；M2 收尾交接的 P9 处理结果 | 11 |
| `e2e/fixtures/settings-pages.ts` | `sentTo`：请求离开页面时读它的请求体，路由一直留着 | 1 |
| `e2e/fixtures/api.ts` | `changeRole`、`furnishWorkspace`（Task 1）；`invitationTo`（Task 3） | 1、3 |
| `e2e/fixtures/workspace-pages.ts` | 工作区页面的操作：另一个浏览器、删除（Task 1，新文件）；重新登录、切换工作区、成员行、改角色、结束成员关系（Task 2，整个文件）；邀请链接、邀请行、发出邀请、删除邀请（Task 3） | 1、2、3 |
| `e2e/fixtures/assert/workspace.ts` | 成员关系的断言（Task 2）；邀请的断言，同一邮箱的行按创建时间排（Task 3） | 2、3 |
| `e2e/stories/smoke/s2-web-app.spec.ts` | 挂载清单：general 页（Task 1）；成员页，邀请只对管理员（Task 2）；新人的新手引导（Task 6）；个人设置（Task 10） | 1、2、6、10 |
| `e2e/stories/workspace/w2-landing.spec.ts` | W2 的页面版本 | 1、2 |
| `e2e/stories/workspace/w3-workspace-settings.spec.ts` | W3 的页面版本和会话切换 | 1 |
| `e2e/stories/workspace/w7-member-management.spec.ts` | W7 的页面版本 | 2 |
| `e2e/stories/workspace/w4-invite-members.spec.ts` | W4 的页面版本 | 3 |
| `e2e/stories/workspace/w5-invitation-link.spec.ts` | W5 的页面版本 | 4 |
| `e2e/stories/workspace/w1-create-workspace.spec.ts` | W1 的页面版本：`/create-workspace`（Task 5）；新手引导（Task 6） | 5、6 |
| `e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts` | W6 的页面版本 | 7 |
| `e2e/stories/workspace/w8-navigation-preferences.spec.ts` | W8 的页面版本（连按两下限制的开关） | 8 |
| `e2e/stories/identity/a12-deactivate.spec.ts`、`e2e/stories/workspace/w9-deactivation.spec.ts` | 停用被拒绝的原因在弹窗里；W9 的页面版本 | 9 |

---

### Task 1: 页面在会话里跟进修改；工作区设置 general 和删除工作区；W3 的页面版本和会话切换

**Files:**
- Create: `e2e/fixtures/workspace-pages.ts`、`web/apps/web/core/components/workspace/delete-workspace-modal.test.tsx`、`web/apps/web/core/components/workspace/settings/workspace-details.test.tsx`、`web/apps/web/core/lib/auth/fake-tab.ts`、`web/apps/web/core/lib/fake-refusal.ts`
- Modify: `.oxlintrc.json`、`e2e/fixtures/api.ts`、`e2e/fixtures/settings-pages.ts`、`e2e/stories/smoke/s2-web-app.spec.ts`、`e2e/stories/workspace/w2-landing.spec.ts`、`e2e/stories/workspace/w3-workspace-settings.spec.ts`、`web/apps/web/core/components/workspace/delete-workspace-modal.tsx`、`web/apps/web/core/components/workspace/delete-workspace-section.tsx`、`web/apps/web/core/components/workspace/settings/workspace-details.tsx`、`web/apps/web/core/lib/in-session.test.ts`、`web/apps/web/core/lib/in-session.ts`、`web/packages/i18n/src/locales/en/workspace-settings.json`、`web/packages/i18n/src/locales/zh-CN/workspace-settings.json`
- Delete: `web/apps/web/core/components/workspace/delete-workspace-form.tsx`

**Interfaces:**
- Produces（spec 2.1；M3 设计 7.1、7.5、9.5、9.6；P8a 的 P12、P26）：
  - `core/lib/in-session.ts`：`followInSession<T>(change: () => Promise<T>, followers: ChangeFollowers<T>): Promise<void>`，`ChangeFollowers<T> = { done: (answer: T) => void; failed: (error: unknown) => void }`（Task 2 起 `done` 可省）：发出修改时取标签页的会话（`sessionGuard()`），修改兑现之后只在标签页仍在那个会话里时跟进：成功调 `done`，失败调 `failed`；另一个标签页把这个标签页换到别的账户之后两者都不调。自己从不拒绝。
  - `DeleteWorkspaceModal`（`core/components/workspace/delete-workspace-modal.tsx`，整个文件）：收 `workspace: Workspace`，表单并入弹窗（`delete-workspace-form.tsx` 删除）；按提交的值核对名称和"delete my workspace"，`deleteWorkspace(workspace)` 经 `followInSession`：成功时提示、关闭、回到 `/`（落点按剩下的工作区决定，3.14），失败时按 `code` 提示并停在弹窗。`delete-workspace-section.tsx` 传地址的工作区。
  - i18n（en、zh-CN 的 `workspace-settings.json`）：删除被拒绝时说 nerve 的原因，`delete_modal.error_message`（"Try again, please."）随之没有读者，删除；`delete_modal.success_message` 改写：删除之后根路径把调用者落到他的另一个工作区或 `/create-workspace`（3.14），不是 Plane 说的个人资料页。
  - `WorkspaceDetails`（general 页）：地址的工作区经 `useParams` + `getWorkspaceBySlug`（不读 `currentWorkspace`）；修改只发表单编辑的 `name`、`organization_size`（没有时不发）、`timezone`（生成的 `WorkspaceUpdate`）；规模没有时表单的值是 `null`（受控，原来的 `undefined` 让 React 在第一次选择时报"uncontrolled to controlled"）；结果经 `followInSession`，拒绝按 `code` 提示（P12：原来只有 `console.error`），被拒绝之后按钮结束加载。复制地址时浏览器不让写剪贴板，提示失败（原来什么都不说；与邀请链接的复制同一句，M2 交接第 14 节）。
- 测试的共用部分：`core/lib/auth/fake-tab.ts`（`api-client.ts` 的替身：`tokenManager`、`signedIn`、`switchAccount`、`heldChange`（M2 的 `fake-browser.ts` 的 `gate()`）、`lateSettlings`、`pageSettled`），`core/lib/fake-refusal.ts`（`refusal(status, code, fields?)`：nerve 的拒绝，`ApiError`）。
- 静态检查（总体设计 7.7）：根目录 `.oxlintrc.json` 的 `typescript/no-non-null-assertion` 的范围加上本 Task 的新模块 `fake-tab.ts`、`fake-refusal.ts` 和 `delete-workspace-modal.test.tsx`。
- e2e 的共用部分：`e2e/fixtures/settings-pages.ts` 的 `sentTo(page, method, path, act)`（请求离开页面时经路由读它的请求体：页面的客户端发出的 `fetch`，`response.request().postDataJSON()` 读不到；路由按路径的模式匹配，一直留着：在页面发出下一个请求时移除路由，可能让那个请求一直挂着）；`e2e/fixtures/api.ts` 的 `changeRole`、`furnishWorkspace`（W2 的 API 版本改用它）；新文件 `e2e/fixtures/workspace-pages.ts`：`anotherBrowser`、`deleteFromGeneralPage`。

**Tests:**
- vitest：`in-session.test.ts` 改用 `fake-tab.ts`，加 `followInSession` 的三条：`follows nerve's answer, and its refusal, while the tab is in the session the change was sent in`；`follows nothing when the change $settles after another tab moved this one to another account`（`it.each(lateSettlings)`）；`takes the session as the change is sent: a change sent after the switch is followed`。`delete-workspace-modal.test.tsx`（9.5：删除工作区的组件，修改在 `loginId` 改变之后才兑现时不跳转、不提示）：`deletes the workspace once its name and the words are typed, then lands at the root and says so`；`deletes nothing when the form has $typed`；`shows nerve's reason when it refuses, and stays`；换账户之后兑现的两种。`workspace-details.test.tsx`：`sends the fields its form edits, the size as nerve names it, and says the workspace is updated`；`sends no size while the workspace has none and none was picked`；`shows nerve's reason when it refuses`；换账户之后兑现的两种。
- 端到端：W3 的两个页面版本：`W3 (page): the admin changes the name, size and time zone, which hold after a reload; a member sees them and can change nothing; …`（请求体经 `sentTo`；被降级的管理员的修改被拒绝，页面说明原因；删除之后到另一个工作区），和会话切换 `W3 (page): a deletion nerve made before another tab signed another account in, and answered after, neither moves the page nor says so: the page is that account's`（9.6：`holdAnswer` 先 `route.fetch()` 把删除发到 nerve，另一个标签页以另一个账户登录之后才 `route.fulfill()`）。S2 加每个角色的 `/{slug}/settings`（管理员和成员取时区，访客没有 general 页）。

- [ ] **Step 1: 会话里的跟进和测试的共用部分**

`web/apps/web/core/lib/auth/fake-tab.ts`（新文件，48 行）：

````file web/apps/web/core/lib/auth/fake-tab.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// A stand-in for api-client.ts, for the tests of what checks the tab's session (in-session.ts, and the pages that
// follow a change with it): a test file mocks that module with this one,
// vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab")), moves the tab from one session to another,
// and has nerve settle a change when it says (heldChange).

import { gate } from "./fake-browser";
import type { SessionState } from "./token-manager";

/** The token manager, as far as the session's check reads it: the tab's session. */
export const tokenManager: { state: SessionState } = { state: { status: "signed-in", loginId: "x" } };

/** Puts the tab in a session of its own, as a test starts: signed in, as login x. */
export function signedIn() {
  tokenManager.state = { status: "signed-in", loginId: "x" };
}

/** Another tab signs another account in, and this one follows it: a change sent before is the previous session's. */
export function switchAccount() {
  tokenManager.state = { status: "signed-in", loginId: `${tokenManager.state.loginId ?? ""}+` };
}

/** A change sent to nerve, and the two ways the test settles it: nerve answers it, or refuses it. */
export type HeldChange<T> = { sent: Promise<T>; answer: (value: T) => void; refuse: (error: unknown) => void };

/** A change nerve settles when the test says (a store's change, as a page's test mocks it): fake-browser.ts's gate. */
export function heldChange<T>(): HeldChange<T> {
  const { promise, open, fail } = gate<T>();
  return { sent: promise, answer: open, refuse: fail };
}

/** The ways a change sent before the tab switched settles after it, for it.each: answered, or refused. */
export const lateSettlings = [
  { settles: "is answered", settle: (change: HeldChange<undefined>) => change.answer(undefined) },
  { settles: "is refused", settle: (change: HeldChange<undefined>) => change.refuse(new Error("refused")) },
];

/**
 * Resolves once a page has followed a change that settled: what it does then runs in promises' steps, and a timer's
 * turn comes after them all.
 */
export function pageSettled(): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, 0));
}
````

`web/apps/web/core/lib/fake-refusal.ts`（新文件，19 行）：

````file web/apps/web/core/lib/fake-refusal.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// nerve's refusal as the web app's client gives it, for the tests of what a page shows of one.

import type { FieldError } from "@nerve/api-client";
import { ApiError } from "./api-error";

/** A refusal of status and code, naming fields, each by its path in the request's body and the code of its fault. */
export function refusal(status: number, code: string, fields: Pick<FieldError, "field" | "code">[] = []): ApiError {
  return new ApiError(status, {
    status,
    code,
    title: "",
    errors: fields.map(({ field, code: fault }) => ({ field, code: fault, message: "" })),
  });
}
````

`web/apps/web/core/lib/in-session.test.ts`（修改，4 处）：

````old web/apps/web/core/lib/in-session.test.ts
import { describe, expect, it, vi } from "vitest";
import type { SessionState } from "@/lib/auth/token-manager";
````
````new web/apps/web/core/lib/in-session.test.ts
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  heldChange,
  lateSettlings,
  signedIn,
  switchAccount,
  tokenManager as tab,
  type HeldChange,
} from "@/lib/auth/fake-tab";
````
````old web/apps/web/core/lib/in-session.test.ts
// The tab's session as the token manager has it, which the test moves from one account to another.
const tab = vi.hoisted((): { state: SessionState } => ({ state: { status: "signed-in", loginId: "x" } }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: tab }));
````
````new web/apps/web/core/lib/in-session.test.ts
// The tab's session as the token manager has it, which the test moves from one account to another (fake-tab.ts).
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
````
````old web/apps/web/core/lib/in-session.test.ts
const { sessionGuard } = await import("./in-session");
````
````new web/apps/web/core/lib/in-session.test.ts
const { followInSession, sessionGuard } = await import("./in-session");

beforeEach(() => {
  signedIn();
});
````
````old web/apps/web/core/lib/in-session.test.ts
  });
});

````
````new web/apps/web/core/lib/in-session.test.ts
  });
});

describe("followInSession", () => {
  /** Follows a change nerve settles when the test says (heldChange): what the page did once it settled. */
  function follow<T>(change: HeldChange<T>) {
    const followed: string[] = [];
    const following = followInSession(() => change.sent, {
      done: (answer) => followed.push(`done: ${String(answer)}`),
      failed: (error) => followed.push(`failed: ${String(error)}`),
    });
    return { followed, following };
  }

  it("follows nerve's answer, and its refusal, while the tab is in the session the change was sent in", async () => {
    const answered = heldChange<string>();
    const afterAnswer = follow(answered);
    answered.answer("acme");
    await afterAnswer.following;
    const refused = heldChange<string>();
    const afterRefusal = follow(refused);
    refused.refuse("refused");
    await afterRefusal.following;
    expect([afterAnswer.followed, afterRefusal.followed]).toEqual([["done: acme"], ["failed: refused"]]);
  });

  it.each(lateSettlings)(
    "follows nothing when the change $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const change = heldChange<undefined>();
      const after = follow(change);
      switchAccount();
      settle(change);
      await after.following;
      expect(after.followed).toEqual([]);
    }
  );

  it("takes the session as the change is sent: a change sent after the switch is followed", async () => {
    switchAccount();
    const change = heldChange<string>();
    const after = follow(change);
    change.answer("acme");
    await after.following;
    expect(after.followed).toEqual(["done: acme"]);
  });
});

````

`web/apps/web/core/lib/in-session.ts`（修改，1 处）：

````old web/apps/web/core/lib/in-session.ts
}

````
````new web/apps/web/core/lib/in-session.ts
}

/** What a page does once a change it sent has settled: with nerve's answer, or with why it failed. */
export type ChangeFollowers<T> = { done: (answer: T) => void; failed: (error: unknown) => void };

/**
 * Sends a change, and follows how it settles on the page only while the tab is in the session the change was sent in
 * (sessionGuard): done with nerve's answer, failed with the error; neither once another tab has moved this one to
 * another account, whose page it then is, whatever the answer (M3 design 7.1). Settles once the change has, and never
 * rejects: its failure is the followers' to show.
 */
export async function followInSession<T>(change: () => Promise<T>, followers: ChangeFollowers<T>): Promise<void> {
  const inSession = sessionGuard();
  const outcome = await change().then(
    (answer) => ({ settled: "done" as const, answer }),
    (error: unknown) => ({ settled: "failed" as const, error })
  );
  if (!inSession()) return;
  if (outcome.settled === "done") followers.done(outcome.answer);
  else followers.failed(outcome.error);
}

````

- [ ] **Step 2: 删除工作区、general 页和文案**

`web/apps/web/core/components/workspace/delete-workspace-form.tsx`（删除）：

````delete web/apps/web/core/components/workspace/delete-workspace-form.tsx
````

`web/apps/web/core/components/workspace/delete-workspace-modal.test.tsx`（新文件，105 行）：

````file web/apps/web/core/components/workspace/delete-workspace-modal.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, shown, submitModalForm } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { DeleteWorkspaceModal } from "./delete-workspace-modal";

// What the deletion of a workspace sends and does on the page (M3 design 7.1, 9.5): the modal renders on the server,
// with stand-ins for its inputs and buttons (fake-controls.ts), which keep the props they were given; the test types
// as a person would, through the inputs' onChange, and submits the form. Its session is fake-tab.ts's.

const store = vi.hoisted(() => ({ deleteWorkspace: vi.fn() }));
const navigate = vi.hoisted(() => vi.fn());
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => ({ deleteWorkspace: store.deleteWorkspace }) }));
vi.mock("react-router", () => ({ useNavigate: () => navigate }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@makeplane/propel/components/input", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const acme = workspaceOf("acme", { name: "Acme" });

/** Renders the modal of Acme, types name and words into its two inputs, and submits it; gives onClose. */
async function confirm(name: string, words: string) {
  const onClose = vi.fn();
  renderToStaticMarkup(<DeleteWorkspaceModal isOpen workspace={acme} onClose={onClose} />);
  const [nameInput, wordsInput] = shown.inputs;
  nameInput?.onChange({ target: { value: name } });
  wordsInput?.onChange({ target: { value: words } });
  await submitModalForm();
  return onClose;
}

beforeEach(() => {
  signedIn();
  store.deleteWorkspace.mockReset();
  store.deleteWorkspace.mockResolvedValue(undefined);
  navigate.mockClear();
  toasts.length = 0;
  emptyShown();
});

describe("DeleteWorkspaceModal", () => {
  it("deletes the workspace once its name and the words are typed, then lands at the root and says so", async () => {
    const onClose = await confirm("Acme", "delete my workspace");
    expect(store.deleteWorkspace.mock.calls).toEqual([[acme]]);
    expect([onClose.mock.calls.length, navigate.mock.calls]).toEqual([1, [["/"]]]);
    expect(toasts).toEqual([
      {
        type: "success",
        title: "workspace_settings.settings.general.delete_modal.success_title",
        message: "workspace_settings.settings.general.delete_modal.success_message",
      },
    ]);
  });

  it.each([
    { typed: "another name", name: "Acme Corp", words: "delete my workspace" },
    { typed: "other words", name: "Acme", words: "delete it" },
  ])("deletes nothing when the form has $typed", async ({ name, words }) => {
    const onClose = await confirm(name, words);
    expect([store.deleteWorkspace.mock.calls, onClose.mock.calls, navigate.mock.calls, toasts]).toEqual([
      [],
      [],
      [],
      [],
    ]);
  });

  it("shows nerve's reason when it refuses, and stays", async () => {
    store.deleteWorkspace.mockRejectedValueOnce(refusal(403, "forbidden"));
    const onClose = await confirm("Acme", "delete my workspace");
    expect([onClose.mock.calls, navigate.mock.calls]).toEqual([[], []]);
    expect(toasts).toEqual([
      {
        type: "error",
        title: "workspace_settings.settings.general.delete_modal.error_title",
        message: "errors.forbidden",
      },
    ]);
  });

  it.each(lateSettlings)(
    "neither moves nor speaks when the deletion $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const deletion = heldChange<undefined>();
      store.deleteWorkspace.mockReturnValueOnce(deletion.sent);
      const submitted = confirm("Acme", "delete my workspace");
      await vi.waitFor(() => expect(store.deleteWorkspace).toHaveBeenCalledTimes(1));
      switchAccount();
      settle(deletion);
      const onClose = await submitted;
      expect([onClose.mock.calls, navigate.mock.calls, toasts]).toEqual([[], [], []]);
    }
  );
});
````

`web/apps/web/core/components/workspace/delete-workspace-modal.tsx`（整个文件，179 行）：

````whole web/apps/web/core/components/workspace/delete-workspace-modal.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { Controller, useForm } from "react-hook-form";
import { useNavigate } from "react-router";
import { WarningTriangleOutline } from "@makeplane/propel/icons";
// Nerve Imports
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import type { Workspace } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
import { cn } from "@nerve/utils";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
// lib
import { errorMessageKey } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";

type Props = {
  isOpen: boolean;
  workspace: Workspace;
  onClose: () => void;
};

/** What the form asks before the deletion: the workspace's name, and the words that confirm it. */
type TDeleteWorkspaceForm = { workspaceName: string; confirmDelete: string };

const defaultValues: TDeleteWorkspaceForm = { workspaceName: "", confirmDelete: "" };

/** The words that confirm the deletion. */
const CONFIRMATION = "delete my workspace";

export const DeleteWorkspaceModal = observer(function DeleteWorkspaceModal(props: Props) {
  const { isOpen, workspace, onClose } = props;
  // router
  const navigate = useNavigate();
  // store hooks
  const { deleteWorkspace } = useWorkspace();
  const { t } = useTranslation();
  // form info
  const {
    control,
    formState: { errors, isSubmitting },
    handleSubmit,
    reset,
    watch,
  } = useForm<TDeleteWorkspaceForm>({ defaultValues });

  const confirmed = (values: TDeleteWorkspaceForm) =>
    values.workspaceName === workspace.name && values.confirmDelete === CONFIRMATION;

  const handleClose = () => {
    const timer = setTimeout(() => {
      reset(defaultValues);
      clearTimeout(timer);
    }, 350);

    onClose();
  };

  // The values submitted decide, as typed; the page follows the deletion only in the session it was sent in (M3
  // design 7.1): once another tab has moved this one to another account, the page is that account's.
  const onSubmit = (values: TDeleteWorkspaceForm) => {
    if (!confirmed(values)) return;
    return followInSession(() => deleteWorkspace(workspace), {
      done: () => {
        handleClose();
        // the root lands the caller where his workspaces, as they are now, say (M3 design 3.14)
        void navigate("/");
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("workspace_settings.settings.general.delete_modal.success_title"),
          message: t("workspace_settings.settings.general.delete_modal.success_message"),
        });
      },
      failed: (error) =>
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("workspace_settings.settings.general.delete_modal.error_title"),
          message: t(errorMessageKey(error)),
        }),
    });
  };

  return (
    <ModalCore isOpen={isOpen} handleClose={() => onClose()} position={EModalPosition.CENTER} width={EModalWidth.XL}>
      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-6 p-6">
        <div className="flex flex-col items-center gap-4 sm:flex-row sm:items-start">
          <span
            className={cn(
              "grid size-12 shrink-0 place-items-center rounded-full bg-danger-subtle text-danger-primary sm:size-10"
            )}
          >
            <WarningTriangleOutline className="size-5 text-danger-primary" aria-hidden="true" />
          </span>
          <div>
            <div className="text-center sm:text-left">
              <h3 className="text-h5-medium">{t("workspace_settings.settings.general.delete_modal.title")}</h3>
              <p className="mt-1 text-body-xs-regular text-secondary">
                You are about to delete the workspace{" "}
                <span className="text-body-xs-semibold break-words">{workspace.name}</span>. If you confirm, you will
                lose access to all your work data in this workspace without any way to restore it. Tread very carefully.
              </p>
            </div>

            <div className="mt-4 text-secondary">
              <p className="text-body-xs-regular break-words">Type in this workspace&apos;s name to continue.</p>
              <Controller
                control={control}
                name="workspaceName"
                render={({ field: { value, onChange, ref } }) => (
                  <Field name="workspaceName" invalid={Boolean(errors.workspaceName)}>
                    <InputGroup size="2xl">
                      <Input
                        size="2xl"
                        id="workspaceName"
                        name="workspaceName"
                        type="text"
                        value={value}
                        onChange={onChange}
                        ref={ref}
                        placeholder={workspace.name}
                        autoComplete="off"
                      />
                    </InputGroup>
                  </Field>
                )}
              />
            </div>

            <div className="mt-4 text-secondary">
              <p className="text-body-xs-regular">
                For final confirmation, type <span className="text-body-xs-medium text-primary">{CONFIRMATION} </span>
                below.
              </p>
              <Controller
                control={control}
                name="confirmDelete"
                render={({ field: { value, onChange, ref } }) => (
                  <Field name="confirmDelete" invalid={Boolean(errors.confirmDelete)}>
                    <InputGroup size="2xl">
                      <Input
                        size="2xl"
                        id="confirmDelete"
                        name="confirmDelete"
                        type="text"
                        value={value}
                        onChange={onChange}
                        ref={ref}
                        placeholder=""
                        autoComplete="off"
                      />
                    </InputGroup>
                  </Field>
                )}
              />
            </div>
          </div>
        </div>

        <div className="flex justify-end gap-2">
          <Button variant="secondary" size="lg" onClick={handleClose}>
            {t("cancel")}
          </Button>
          <Button variant="error-fill" size="lg" type="submit" disabled={!confirmed(watch())} loading={isSubmitting}>
            {isSubmitting ? t("deleting") : t("confirm")}
          </Button>
        </div>
      </form>
    </ModalCore>
  );
});
````

`web/apps/web/core/components/workspace/delete-workspace-section.tsx`（修改，2 处）：

````old web/apps/web/core/components/workspace/delete-workspace-section.tsx
  workspace: Workspace | null;
````
````new web/apps/web/core/components/workspace/delete-workspace-section.tsx
  workspace: Workspace;
````
````old web/apps/web/core/components/workspace/delete-workspace-section.tsx
        data={workspace}
````
````new web/apps/web/core/components/workspace/delete-workspace-section.tsx
        workspace={workspace}
````

`web/apps/web/core/components/workspace/settings/workspace-details.test.tsx`（新文件，108 行）：

````file web/apps/web/core/components/workspace/settings/workspace-details.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, pick, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { WorkspaceDetails } from "./workspace-details";

// What the workspace's general page sends, and what it shows of nerve's answer (M3 design 7.1, 7.5; P8a review P12):
// the page renders on the server, for the workspace at its address, with stand-ins for its inputs, selects and button
// (fake-controls.ts) and for the time zone's select, which keep the props they were given; the test edits as an admin
// would, through their onChange, and clicks the update. Its session is fake-tab.ts's.

const page = vi.hoisted(() => {
  const zones: { onChange: (zone: string) => void }[] = [];
  return { updateWorkspace: vi.fn(), zones };
});
// Acme has no size: the form shows none, and sends none until one is picked.
const acme = workspaceOf("acme", { name: "Acme", role: 20 });
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "acme" }) }));
vi.mock("@/hooks/store/use-workspace", () => ({
  useWorkspace: () => ({
    getWorkspaceBySlug: (slug: string) => (slug === acme.slug ? acme : null),
    updateWorkspace: page.updateWorkspace,
  }),
}));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => ({ allowPermissions: () => true }) }));
vi.mock("@/components/global/timezone-select", () => ({
  TimezoneSelect: (props: { onChange: (zone: string) => void }) => {
    page.zones.push(props);
    return null;
  },
}));
vi.mock("@/components/workspace/delete-workspace-section", () => ({ DeleteWorkspaceSection: () => null }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@makeplane/propel/components/input", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Renders the page, has edit change its fields, and clicks the update; settles once the page has followed nerve. */
async function update(edit: (form: { name: string; size?: string; zone?: string }) => void = () => {}) {
  renderToStaticMarkup(<WorkspaceDetails />);
  const [name] = shown.inputs;
  const [size] = shown.selects;
  const [zone] = page.zones;
  const button = shown.buttons.at(-1);
  if (!name || !size || !zone || !button?.onClick) throw new Error("the page showed no form to update");
  const form: { name: string; size?: string; zone?: string } = { name: acme.name };
  edit(form);
  name.onChange({ target: { value: form.name } });
  if (form.size) pick(size, form.size);
  if (form.zone) zone.onChange(form.zone);
  button.onClick({ preventDefault: () => {} });
  await pageSettled();
}

beforeEach(() => {
  signedIn();
  page.updateWorkspace.mockReset();
  page.updateWorkspace.mockResolvedValue(acme);
  page.zones.length = 0;
  toasts.length = 0;
  emptyShown();
});

describe("WorkspaceDetails", () => {
  it("sends the fields its form edits, the size as nerve names it, and says the workspace is updated", async () => {
    await update((form) => Object.assign(form, { name: "Acme Corp", size: "11-50", zone: "Asia/Shanghai" }));
    expect(page.updateWorkspace.mock.calls).toEqual([
      ["acme", { name: "Acme Corp", organization_size: "11-50", timezone: "Asia/Shanghai" }],
    ]);
    expect(toasts).toEqual([{ type: "success", title: "Success!", message: "Workspace updated successfully" }]);
  });

  // nerve cannot set the size to none (WorkspaceUpdate): a null would be refused
  it("sends no size while the workspace has none and none was picked", async () => {
    await update();
    expect(page.updateWorkspace.mock.calls).toEqual([["acme", { name: "Acme", timezone: "UTC" }]]);
  });

  it("shows nerve's reason when it refuses", async () => {
    page.updateWorkspace.mockRejectedValueOnce(refusal(403, "forbidden"));
    await update();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.forbidden" }]);
  });

  it.each(lateSettlings)(
    "says nothing when the update $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const change = heldChange<undefined>();
      page.updateWorkspace.mockReturnValueOnce(change.sent);
      await update();
      expect(page.updateWorkspace).toHaveBeenCalledTimes(1);
      switchAccount();
      settle(change);
      await pageSettled();
      expect(toasts).toEqual([]);
    }
  );
});
````

`web/apps/web/core/components/workspace/settings/workspace-details.tsx`（修改，11 处）：

````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
import { Controller, useForm } from "react-hook-form";
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
import { Controller, useForm } from "react-hook-form";
import { useParams } from "react-router";
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
import type { Workspace, WorkspaceUpdate } from "@nerve/api-client";
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
import type { OrganizationSize, Workspace, WorkspaceUpdate } from "@nerve/api-client";
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
import { DeleteWorkspaceSection } from "@/components/workspace/delete-workspace-section";
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
import { DeleteWorkspaceSection } from "@/components/workspace/delete-workspace-section";
// lib
import { errorMessageKey } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
/** The form's values: what an admin may change of the workspace. */
type TWorkspaceForm = Required<Pick<WorkspaceUpdate, "name" | "timezone">> & Pick<WorkspaceUpdate, "organization_size">;
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
/**
 * The form's values: what an admin may change of the workspace. The size is null while the workspace has none, which
 * nerve's WorkspaceUpdate cannot set: the select holds a value from the start (undefined would leave it uncontrolled).
 */
type TWorkspaceForm = Required<Pick<WorkspaceUpdate, "name" | "timezone">> & {
  organization_size: OrganizationSize | null;
};
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
  organization_size: ORGANIZATION_SIZE.find((size) => size === workspace.organization_size),
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
  organization_size: ORGANIZATION_SIZE.find((size) => size === workspace.organization_size) ?? null,
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
  const [isLoading, setIsLoading] = useState(false);
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
  const [isLoading, setIsLoading] = useState(false);
  // router: the address's workspace, as the caller's list has it (the wrapper shows the page once it has)
  const { workspaceSlug } = useParams();
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
  const { currentWorkspace, updateWorkspace } = useWorkspace();
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
  const { getWorkspaceBySlug, updateWorkspace } = useWorkspace();
  const currentWorkspace = workspaceSlug ? getWorkspaceBySlug(workspaceSlug) : null;
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
  });

  const onSubmit = async (formData: TWorkspaceForm) => {
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
  });

  // the fields the form edits, which nerve's WorkspaceUpdate takes; the page follows the change only in the session it
  // was sent in (M3 design 7.1)
  const onSubmit = async (formData: TWorkspaceForm) => {
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
      organization_size: formData.organization_size,
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
      organization_size: formData.organization_size ?? undefined,
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
    try {
      await updateWorkspace(currentWorkspace.slug, payload);
      setToast({
        title: "Success!",
        type: TOAST_TYPE.SUCCESS,
        message: "Workspace updated successfully",
      });
    } catch (err: unknown) {
      console.error(err);
    } finally {
      setTimeout(() => {
        setIsLoading(false);
      }, 300);
    }
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
    await followInSession(() => updateWorkspace(currentWorkspace.slug, payload), {
      done: () =>
        setToast({
          title: "Success!",
          type: TOAST_TYPE.SUCCESS,
          message: "Workspace updated successfully",
        }),
      failed: (error) =>
        setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) }),
    });
    setIsLoading(false);
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
      .catch(() => {
        // Silently handle clipboard errors
      });
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
      // the browser did not let the page write the clipboard
      .catch(() =>
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("toast.error"),
          message: t("something_went_wrong_please_try_again"),
        })
      );
````

`web/packages/i18n/src/locales/en/workspace-settings.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/workspace-settings.json
          "success_message": "You will soon go to your profile page.",
          "error_title": "That didn't work.",
          "error_message": "Try again, please."
````
````new web/packages/i18n/src/locales/en/workspace-settings.json
          "success_message": "You will go to another of your workspaces, or create one if you have none left.",
          "error_title": "That didn't work."
````

`web/packages/i18n/src/locales/zh-CN/workspace-settings.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/workspace-settings.json
          "success_message": "即将跳转到您的个人资料页面。",
          "error_title": "操作失败。",
          "error_message": "请重试。"
````
````new web/packages/i18n/src/locales/zh-CN/workspace-settings.json
          "success_message": "即将跳转到您的另一个工作区；没有其他工作区时，跳转到创建工作区。",
          "error_title": "操作失败。"
````

- [ ] **Step 3: 静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/components/projects/use-archived-projects-fetch.test.ts",
        "web/apps/web/core/components/projects/page.tsx"
````
````new .oxlintrc.json
        "web/apps/web/core/components/projects/use-archived-projects-fetch.test.ts",
        "web/apps/web/core/components/projects/page.tsx",
        "web/apps/web/core/lib/auth/fake-tab.ts",
        "web/apps/web/core/lib/fake-refusal.ts",
        "web/apps/web/core/components/workspace/delete-workspace-modal.test.tsx"
````

- [ ] **Step 4: 端到端**

`e2e/fixtures/api.ts`（修改，2 处）：

````old e2e/fixtures/api.ts
export type WorkspaceMember = components["schemas"]["WorkspaceMember"];
````
````new e2e/fixtures/api.ts
export type WorkspaceMember = components["schemas"]["WorkspaceMember"];
export type WorkspaceMemberUpdate = components["schemas"]["WorkspaceMemberUpdate"];
````
````old e2e/fixtures/api.ts
}

/** Creates a project in the workspace of slug with the bearer token given, an admin's or a member's, and returns it. */
````
````new e2e/fixtures/api.ts
}

/**
 * Gives the workspace of slug, as its admin of adminToken, a row of each table its deletion writes
 * (expectWorkspaceDeleted): the membership of member, by an invitation he accepts (deleted as he accepts it); a pending
 * invitation of inviteeEmail; the admin's display settings; and the project Web, with its own rows, its label Bug
 * among them.
 */
export async function furnishWorkspace(
  api: Api,
  adminToken: string,
  slug: string,
  member: { email: string; token: string },
  inviteeEmail: string
): Promise<void> {
  await inviteAndAccept(api, adminToken, slug, member, 15);
  await invite(api, adminToken, slug, [{ email: inviteeEmail, role: 15 }]);
  const { error, response } = await api.PATCH("/api/v0/me/workspaces/{slug}/preferences", {
    params: { path: { slug } },
    body: { navigation_project_limit: 3 },
    headers: bearer(adminToken),
  });
  expect(response.status, `the admin's settings in ${slug}: ${JSON.stringify(error)}`).toBe(200);
  const web = await createProject(api, adminToken, slug, { name: "Web", identifier: "WEB" });
  await createLabel(api, adminToken, web.id, { name: "Bug" });
}

/** Gives the account of memberId the role in the workspace of slug, with the bearer token given, an admin's. */
export async function changeRole(
  api: Api,
  token: string,
  slug: string,
  memberId: string,
  role: WorkspaceMemberUpdate["role"]
): Promise<void> {
  const { error, response } = await api.PATCH("/api/v0/workspace-members/{workspace_member_id}", {
    params: { path: { workspace_member_id: await membershipOf(api, token, slug, memberId) } },
    body: { role },
    headers: bearer(token),
  });
  expect(response.status, `the role of ${memberId} in ${slug}: ${JSON.stringify(error)}`).toBe(200);
}

/** Creates a project in the workspace of slug with the bearer token given, an admin's or a member's, and returns it. */
````

`e2e/fixtures/settings-pages.ts`（修改，1 处）：

````old e2e/fixtures/settings-pages.ts
  return response;
````
````new e2e/fixtures/settings-pages.ts
  return response;
}

/**
 * Resolves with the body, as JSON, of the request of method to path (no query) that act makes page send, and nerve's
 * answer to it. The body is read as the request leaves page, on its way (a route): the request of a response does not
 * have the bodies the web app's client sends. The route matches path by a pattern, so it stops no other request, and
 * stays for the page's life: a route removed while the page sends its next request can leave that request waiting for
 * good.
 */
export async function sentTo(
  page: Page,
  method: string,
  path: string,
  act: () => Promise<void>
): Promise<{ body: unknown; answer: Response }> {
  const bodies: unknown[] = [];
  await page.route(`**${path}`, async (route: Route) => {
    if (route.request().method() === method) bodies.push(route.request().postDataJSON());
    await route.fallback();
  });
  const answer = await answerTo(page, method, path, act);
  return { body: bodies[0], answer };
````

`e2e/fixtures/workspace-pages.ts`（新文件，33 行）：

````file e2e/fixtures/workspace-pages.ts
import type { Browser, Page, Response } from "@playwright/test";

import { signInContext, type AuthTokens } from "./auth";
import { answerTo } from "./settings-pages";

// The workspace's pages (M3 design 7.5), as a person uses them.

/**
 * A page of a browser of its own, signed in with tokens at the nerve of baseURL: another person's, beside the test's
 * page. Its context closes when the test's function returned by it is called.
 */
export async function anotherBrowser(
  browser: Browser,
  baseURL: string,
  tokens: AuthTokens
): Promise<{ page: Page; close: () => Promise<void> }> {
  const context = await browser.newContext({ baseURL });
  await signInContext(context, baseURL, tokens);
  return { page: await context.newPage(), close: () => context.close() };
}

/**
 * Deletes the workspace of slug, whose name is name, from its general page, which page shows: the admin opens the
 * deletion, types the name and the words that confirm it, and confirms. Resolves with nerve's answer to the deletion.
 */
export async function deleteFromGeneralPage(page: Page, slug: string, name: string): Promise<Response> {
  await page.getByRole("button", { name: "Delete", exact: true }).click();
  await page.locator("#workspaceName").fill(name);
  await page.locator("#confirmDelete").fill("delete my workspace");
  return answerTo(page, "DELETE", `/api/v0/workspaces/${slug}`, () =>
    page.getByRole("button", { name: "Confirm", exact: true }).click()
  );
}
````

`e2e/stories/smoke/s2-web-app.spec.ts`（修改，6 处）：

````old e2e/stories/smoke/s2-web-app.spec.ts
/** The project's general settings, the page S2 opens, list the time zones for its member (M2 design 5.3). */
````
````new e2e/stories/smoke/s2-web-app.spec.ts
/**
 * A general settings page lists the time zones (M2 design 5.3): the project's, for its members; the workspace's, for
 * its admins and members (the workspace's settings show a guest no general page, M3 design 9.2).
 */
````
````old e2e/stories/smoke/s2-web-app.spec.ts
 * call of its own fetch no other check holds; then the project's settings.
````
````new e2e/stories/smoke/s2-web-app.spec.ts
 * call of its own fetch no other check holds; the workspace's general settings; then the project's settings.
````
````old e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/projects", [...APP, ...WORKSPACE, ...ARCHIVED]],
````
````new e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/projects", [...APP, ...WORKSPACE, ...ARCHIVED]],
    ["/{slug}/settings", [...APP, ...WORKSPACE, ...GENERAL]],
````
````old e2e/stories/smoke/s2-web-app.spec.ts
  member: [
    ["/", [...APP, ...WORKSPACE]],
````
````new e2e/stories/smoke/s2-web-app.spec.ts
  member: [
    ["/", [...APP, ...WORKSPACE]],
    ["/{slug}/settings", [...APP, ...WORKSPACE, ...GENERAL]],
````
````old e2e/stories/smoke/s2-web-app.spec.ts
  guest: [
    ["/", [...APP, ...WORKSPACE]],
````
````new e2e/stories/smoke/s2-web-app.spec.ts
  guest: [
    ["/", [...APP, ...WORKSPACE]],
    ["/{slug}/settings", [...APP, ...WORKSPACE]],
````
````old e2e/stories/smoke/s2-web-app.spec.ts
  "project non-member": [
    ["/", [...APP, ...WORKSPACE]],
````
````new e2e/stories/smoke/s2-web-app.spec.ts
  "project non-member": [
    ["/", [...APP, ...WORKSPACE]],
    ["/{slug}/settings", [...APP, ...WORKSPACE, ...GENERAL]],
````

`e2e/stories/workspace/w2-landing.spec.ts`（修改，3 处）：

````old e2e/stories/workspace/w2-landing.spec.ts
  addProjectMembers,
  createLabel,
````
````new e2e/stories/workspace/w2-landing.spec.ts
  addProjectMembers,
````
````old e2e/stories/workspace/w2-landing.spec.ts
  invite,
````
````new e2e/stories/workspace/w2-landing.spec.ts
  furnishWorkspace,
````
````old e2e/stories/workspace/w2-landing.spec.ts
  await inviteAndAccept(api, alice, first, { email: bobEmail, token: bob }, 15);
  await invite(api, alice, first, [{ email: emailFor(testInfo, "invitee"), role: 15 }]);
  const settings = await api.PATCH("/api/v0/me/workspaces/{slug}/preferences", {
    params: { path: { slug: first } },
    body: { navigation_project_limit: 3 },
    headers: bearer(alice),
  });
  expect(settings.response.status).toBe(200);
  const web = await createProject(api, alice, first, { name: "Web", identifier: "WEB" });
  await createLabel(api, alice, web.id, { name: "Bug" });
````
````new e2e/stories/workspace/w2-landing.spec.ts
  await furnishWorkspace(api, alice, first, { email: bobEmail, token: bob }, emailFor(testInfo, "invitee"));
````

`e2e/stories/workspace/w3-workspace-settings.spec.ts`（修改，5 处）：

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
import {
  createLabel,
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
import {
  changeRole,
  createLabel,
````
````old e2e/stories/workspace/w3-workspace-settings.spec.ts
  createWorkspace,
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
  createWorkspace,
  furnishWorkspace,
````
````old e2e/stories/workspace/w3-workspace-settings.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
import { accountId, bearer, createPAT, emailFor, login, newRecord, register, writeRecord } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { answerTo, holdAnswer, registerOnboarded, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser, deleteFromGeneralPage } from "../../fixtures/workspace-pages";
````
````old e2e/stories/workspace/w3-workspace-settings.spec.ts
// W3, the workspace's settings (M3 design 2). The page version, with the
// session switch of 7.1, comes with the general page (P9).
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
// W3, the workspace's settings (M3 design 2), with the session switch of 7.1.
````
````old e2e/stories/workspace/w3-workspace-settings.spec.ts
  ]);
});

````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
  ]);
});

test("W3 (page): the admin changes the name, size and time zone, which hold after a reload; a member sees them and can change nothing; demoted, the admin is refused and told why; the admin deletes the workspace by its name and lands on his other one", async ({
  api,
  baseURL,
  browser,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const memberEmail = emailFor(testInfo, "member");
  const member = await registerOnboarded(api, memberEmail);
  const slug = slugFor(testInfo);
  const other = slugFor(testInfo, "other");
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  await createWorkspace(api, admin.access_token, { name: "Zeta", slug: other });
  const memberAccount = { email: memberEmail, token: member.access_token };
  await furnishWorkspace(api, admin.access_token, slug, memberAccount, emailFor(testInfo, "invitee"));
  const memberId = await accountId(api, member.access_token);

  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/settings`);
  await expect(page.locator("#name")).toHaveValue("Acme");
  await page.locator("#name").fill("Acme Corp");
  await page.getByRole("button", { name: "Select organization size" }).click();
  await page.getByRole("option", { name: "11-50" }).click();
  await page.getByRole("button", { name: "UTC" }).click();
  await page.getByRole("combobox", { name: "Search" }).fill("Asia/Shanghai");
  await page.getByRole("option", { name: "Beijing" }).click();
  // The page sends the fields its form edits, the size as nerve's OrganizationSize names it.
  const updated = await sentTo(page, "PATCH", `/api/v0/workspaces/${slug}`, () =>
    page.getByRole("button", { name: "Update workspace" }).click()
  );
  expect([updated.answer.status(), updated.body]).toEqual([
    200,
    { name: "Acme Corp", organization_size: "11-50", timezone: "Asia/Shanghai" },
  ]);
  await expect(page.getByText("Workspace updated successfully")).toBeVisible();
  expect(
    await db.query(
      `SELECT w.name, w.organization_size, w.timezone, w.updated_by_id = u.id AS by_the_admin
         FROM workspaces w JOIN users u ON u.email = $2 WHERE w.slug = $1`,
      [slug, adminEmail]
    )
  ).toEqual([{ name: "Acme Corp", organization_size: "11-50", timezone: "Asia/Shanghai", by_the_admin: true }]);
  await page.reload();
  await expect(page.locator("#name")).toHaveValue("Acme Corp");
  await expect(page.getByRole("button", { name: "11-50" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Beijing" })).toBeVisible();

  // The member sees what nerve holds, in a form he cannot change; he has no update and no deletion.
  const theMember = await anotherBrowser(browser, baseURL ?? "", member);
  const memberWatch = await watchPage(theMember.page);
  await theMember.page.goto(`/${slug}/settings`);
  await expect(theMember.page.locator("#name")).toHaveValue("Acme Corp");
  await expect(theMember.page.locator("#name")).toBeDisabled();
  await expect(theMember.page.getByRole("button", { name: "Update workspace" })).toHaveCount(0);
  await expect(theMember.page.getByRole("button", { name: "Delete", exact: true })).toHaveCount(0);
  expect([memberWatch.apiFailures, memberWatch.oldApiRequests, memberWatch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(theMember.page, memberWatch, { warnings: [EMOJI_CHECK_WARNING] });
  await theMember.close();

  // Made an admin, the member demotes the admin while his page is open: nerve refuses his next update, and the page
  // says why and lets him try again. Made an admin again, he goes on.
  const adminId = await accountId(api, admin.access_token);
  await changeRole(api, admin.access_token, slug, memberId, 20);
  await changeRole(api, member.access_token, slug, adminId, 15);
  const refused = await answerTo(page, "PATCH", `/api/v0/workspaces/${slug}`, () =>
    page.getByRole("button", { name: "Update workspace" }).click()
  );
  expect(refused.status()).toBe(403);
  await expect(page.getByText("Your role does not allow this.")).toBeVisible();
  await expect(page.getByRole("button", { name: "Update workspace" })).toBeEnabled();
  await changeRole(api, member.access_token, slug, adminId, 20);

  // The admin deletes it; the root lands him on his other workspace, and the page says so.
  expect((await deleteFromGeneralPage(page, slug, "Acme Corp")).status()).toBe(204);
  await expect(page).toHaveURL(`/${other}`);
  await expect(page.getByText("Workspace deleted.")).toBeVisible();
  await expectWorkspaceDeleted(db, slug, adminEmail, ["workspace_member_invites"]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`403 PATCH /api/v0/workspaces/${slug}`],
    [],
    [],
  ]);
  // The settings' two loads, and the workspace's home; the browser's report of the refusal.
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING],
    errors: ["Failed to load resource: the server responded with a status of 403 (Forbidden)"],
  });
});

// The session switch (M3 design 7.1, 9.6): a change the page sent before another tab moved it to another account may
// still succeed after, and the page is that account's then. The request reaches nerve before the switch, as X's: sent
// after it, it would be Y's, refused, and the page would stay put without any check of the session.
test("W3 (page): a deletion nerve made before another tab signed another account in, and answered after, neither moves the page nor says so: the page is that account's", async ({
  api,
  context,
  db,
  signedInPage,
}, testInfo) => {
  const x = emailFor(testInfo, "x");
  const y = emailFor(testInfo, "y");
  const xTokens = await registerOnboarded(api, x);
  const yTokens = await registerOnboarded(api, y);
  const slug = slugFor(testInfo);
  await createWorkspace(api, xTokens.access_token, { name: "Doomed", slug });
  await createWorkspace(api, xTokens.access_token, { name: "Kept", slug: slugFor(testInfo, "kept") });
  await createWorkspace(api, yTokens.access_token, { name: "Yours", slug: slugFor(testInfo, "yours") });
  const tabA = await signedInPage(xTokens);
  const watch = await watchPage(tabA);
  await tabA.goto(`/${slug}/settings`);
  await expect(tabA.locator("#name")).toHaveValue("Doomed");

  // X deletes Doomed: nerve deletes it at once, and its answer waits (holdAnswer: route.fetch, later route.fulfill).
  const release = await holdAnswer(tabA, "DELETE", `/api/v0/workspaces/${slug}`);
  const deleted = deleteFromGeneralPage(tabA, slug, "Doomed");
  const deletedAt = async () =>
    (await db.query<{ deleted_at: Date | null }>(`SELECT deleted_at FROM workspaces WHERE slug = $1`, [slug]))[0]
      ?.deleted_at ?? null;
  await expect.poll(deletedAt).toBeInstanceOf(Date);

  // Tab B keeps a sign-in of Y as the token manager does; tab A follows, and Doomed is not one of Y's.
  const tabB = await context.newPage();
  await tabB.goto("/site.webmanifest.json");
  await writeRecord(tabB, newRecord(await login(api, y)));
  await expect(tabA.getByText("Workspace not found")).toBeVisible();

  // nerve's answer reaches tab A. A move would come with the deletion's continuation, as its answer settles it: the
  // window of 2 s after the release is orders of magnitude longer than the moment that takes.
  const moved = tabA
    .waitForURL((url) => url.pathname !== `/${slug}/settings`, { timeout: 2_000 })
    .then(
      () => true,
      () => false
    );
  await release();
  expect((await deleted).status()).toBe(204);
  expect(await moved).toBe(false);
  // counted at once, as a retrying check would pass once a toast had gone
  expect(await tabA.getByText("Workspace deleted.").count()).toBe(0);
  await expect(tabA.getByText("Workspace not found")).toBeVisible();
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  // One load of tab A.
  await expectQuietConsole(tabA, watch, { warnings: [EMOJI_CHECK_WARNING] });
});

````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、3 个例外，没有命中；web 的 oxlint 360 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 77 个全部通过。

- [ ] **Step 6: 提交**

```bash
git add .oxlintrc.json e2e/fixtures/api.ts e2e/fixtures/settings-pages.ts e2e/fixtures/workspace-pages.ts e2e/stories/smoke/s2-web-app.spec.ts e2e/stories/workspace/w2-landing.spec.ts e2e/stories/workspace/w3-workspace-settings.spec.ts web/apps/web/core/components/workspace/delete-workspace-form.tsx web/apps/web/core/components/workspace/delete-workspace-modal.test.tsx web/apps/web/core/components/workspace/delete-workspace-modal.tsx web/apps/web/core/components/workspace/delete-workspace-section.tsx web/apps/web/core/components/workspace/settings/workspace-details.test.tsx web/apps/web/core/components/workspace/settings/workspace-details.tsx web/apps/web/core/lib/auth/fake-tab.ts web/apps/web/core/lib/fake-refusal.ts web/apps/web/core/lib/in-session.test.ts web/apps/web/core/lib/in-session.ts web/packages/i18n/src/locales/en/workspace-settings.json web/packages/i18n/src/locales/zh-CN/workspace-settings.json
```
```bash
git commit -m "feat(M3/P9): pages follow a change only in the session they sent it in; the general page and the deletion

followInSession sends a change and follows how it settles only while the
tab is in the session it was sent in: once another tab has moved this one
to another account, the page is that account's and hears nothing of it.
The deletion modal owns its form, checks the values submitted, says
nerve's reason for a refusal, and lands at the root only in its
session, saying the page goes to another workspace or to create one,
not to a profile page. The general page finds the address's workspace,
sends the three fields its form edits, says nerve's reason for a
refusal, keeps the size controlled, and says when the browser does not
let it copy the address. The new modules join the non-null override.
W3's page versions, the session switch among them, and S2's general
pages.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p9.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T1.1` | `followInSession` 不论会话都跟进修改的结果 | `in-session.test.ts`、`delete-workspace-modal.test.tsx`、`workspace-details.test.tsx`、故事 W3 | vitest；端到端 |
| `T1.2` | general 页丢掉 nerve 对修改的拒绝 | `workspace-details.test.tsx`、故事 W3 | vitest；端到端 |
| `T1.3` | 删除工作区丢掉 nerve 的拒绝 | `delete-workspace-modal.test.tsx` | vitest |
| `T1.4` | 修改发出整个工作区 | `workspace-details.test.tsx`、故事 W3 | vitest；端到端 |
| `T1.5` | 没有规模的工作区，修改发出规模为 `null` | `workspace-details.test.tsx` | vitest |
| `T1.6` | 没有规模的工作区，规模选择开始时不受控（React 的控制台错误） | 故事 W3 | 端到端 |
| `T1.7` | 不论输入的名称和确认的话，删除都照样进行 | `delete-workspace-modal.test.tsx` | vitest |
| `T1.8` | 删除之后不回到根路径（落点） | `delete-workspace-modal.test.tsx`、故事 W3 | vitest；端到端 |
| `T1.9` | 修改被拒绝之后按钮一直在加载 | 故事 W3 | 端到端 |
| `T1.10` | 标签页的替身（`fake-tab.ts`，非空断言的范围内）用非空断言 | oxlint（`check:lint`） | 静态 |

---

### Task 2: 成员页的三个修改；W7、W2 的页面版本

**Files:**
- Create: `web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts`、`web/apps/web/core/components/workspace/settings/use-membership-changes.ts`、`web/apps/web/core/components/workspace/workspace-roles.ts`
- Modify: `.oxlintrc.json`、`e2e/fixtures/assert/workspace.ts`、`e2e/fixtures/workspace-pages.ts`、`e2e/stories/smoke/s2-web-app.spec.ts`、`e2e/stories/workspace/w2-landing.spec.ts`、`e2e/stories/workspace/w7-member-management.spec.ts`、`web/apps/web/core/components/workspace/settings/member-columns.tsx`、`web/apps/web/core/components/workspace/settings/members-list-item.tsx`、`web/apps/web/core/components/workspace/settings/members-list.tsx`、`web/apps/web/core/components/workspace/settings/useMemberColumns.tsx`、`web/apps/web/core/lib/in-session.ts`

**Interfaces:**
- Produces（spec 2.2；M3 设计 2 的 W7、W2，3.14，7.1，7.5；P8a 的 P21、P22）：
  - `core/components/workspace/settings/use-membership-changes.ts`：`useMembershipChanges(workspaceSlug)` 给出 `{ changeRole(userId, role: WorkspaceRole), remove(userId), leave() }`，都经 `followInSession`，拒绝按 `code` 提示（`errorMessageKey`）。`changeRole` 发 `{ role }`（生成的 `WorkspaceMemberUpdate`，编号）；`leave` 在会话里成功之后回到 `/`（落点按剩下的工作区决定，3.14）。原来三处各自的 `try/catch` 读 `err.error`（P21），换账户之后仍跳转（9.5）。
  - `core/components/workspace/workspace-roles.ts`：`WORKSPACE_ROLES: WorkspaceRole[] = [5, 15, 20]`，角色选择的选项（成员页的角色列，Task 3 的邀请行和邀请列表也用）。
  - `member-columns.tsx` 的 `AccountTypeColumn`：不再包一层 `useForm` 的 `Controller`（它从没有提交过，`value as EUserPermissions` 去掉，`errors.role` 永远为空）；`CustomSelect` 的值是行的角色，选项是 `WORKSPACE_ROLES`，`onChange` 调 `changeRole`。
  - `members-list-item.tsx`（整个文件）：收 `workspaceSlug`（地址的工作区，`members-list.tsx` 传入，不读 `currentWorkspace`），自己一行是"离开"，别人的一行是"移出"，都经 `useMembershipChanges`。
  - `core/lib/in-session.ts`：`ChangeFollowers.done` 可省（store 写入回答、页面不另做什么的修改）。
  - 静态检查（总体设计 7.7）：`.oxlintrc.json` 的非空断言的范围加上 `workspace-roles.ts`（`use-membership-changes.ts` 和它的测试已在 `workspace/settings/**` 之内）。
- e2e 的共用部分：`e2e/fixtures/workspace-pages.ts`（整个文件）加 `signInAnew`（在自己的浏览器里经登录表单登录，看着它的第一页）、`switchWorkspace`（经工作区菜单打开，交回页面写的"上次打开的工作区"和 nerve 的回答）、`memberRow`、`pickRole`、`endMembership`；`e2e/fixtures/assert/workspace.ts` 加 `lastWorkspaceOf(db, email)`。

**Tests:**
- vitest：`use-membership-changes.test.ts`：`sends a member's new role as its number, to the membership's workspace`；`removes a member of the workspace, and stays`；`leaves the workspace of the address, then lands at the root`；`shows nerve's reason when it refuses $change, and stays`（三种修改各一行）；`does nothing on the page when it $settles after another tab moved this one to another account`（`lateSettlings`）。
- 端到端：`W7 (page): the admin makes a member a guest, in the workspace and in each of his projects, and removes another, who comes back by a new invitation; …`（请求体 `{ role: 5 }` 经 `sentTo`；只有一个管理员时离开被拒绝，页面说明原因；成员看得到角色，改不了）；`W2 (page): signed in, an account lands on the workspace it opened last; once that is deleted, on its other one; …`（落点、切换工作区写的 `last_workspace_id`、删除和离开之后的落点，没有失败的请求）。S2 加每个角色的 `/{slug}/settings/members`：工作区的邀请只为管理员取（`INVITATIONS`），成员和访客的挂载不请求它。

- [ ] **Step 1: 成员页的修改**

`web/apps/web/core/components/workspace/settings/member-columns.tsx`（修改，6 处）：

````old web/apps/web/core/components/workspace/settings/member-columns.tsx
import { Link } from "react-router";
import { Controller, useForm } from "react-hook-form";
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
import { Link } from "react-router";
````
````old web/apps/web/core/components/workspace/settings/member-columns.tsx
import { Pill, EPillVariant, EPillSize } from "@nerve/propel/pill";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
import { Pill, EPillVariant, EPillSize } from "@nerve/propel/pill";
````
````old web/apps/web/core/components/workspace/settings/member-columns.tsx
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useUser, useUserPermissions } from "@/hooks/store/user";
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
// hooks
import { useUser, useUserPermissions } from "@/hooks/store/user";
// local imports
import { WORKSPACE_ROLES } from "../workspace-roles";
import { useMembershipChanges } from "./use-membership-changes";
````
````old web/apps/web/core/components/workspace/settings/member-columns.tsx
  const { rowData, workspaceSlug } = props;
  // form info
  const {
    control,
    formState: { errors },
  } = useForm();
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
  const { rowData, workspaceSlug } = props;
````
````old web/apps/web/core/components/workspace/settings/member-columns.tsx
  const { allowPermissions } = useUserPermissions();

  const {
    workspace: { updateMember },
  } = useMember();
  const { data: currentUser } = useUser();
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
  const { allowPermissions } = useUserPermissions();
  const { data: currentUser } = useUser();
  const { changeRole } = useMembershipChanges(workspaceSlug);
````
````old web/apps/web/core/components/workspace/settings/member-columns.tsx
        <Controller
          name="role"
          control={control}
          rules={{ required: "Role is required." }}
          render={({ field: { value } }) => (
            <CustomSelect
              value={value as EUserPermissions}
              // the select gives the chosen option's value: the role's number, which nerve decodes as a WorkspaceRole
              onChange={async (role: WorkspaceRole) => {
                if (!workspaceSlug) return;
                try {
                  await updateMember(workspaceSlug, rowData.member.id, { role });
                } catch (err: unknown) {
                  const error = err as { error?: string | string[] };
                  const errorString = Array.isArray(error?.error) ? error.error[0] : error?.error;

                  setToast({
                    type: TOAST_TYPE.ERROR,
                    title: "Error!",
                    message: errorString ?? "An error occurred while updating member role. Please try again.",
                  });
                }
              }}
              label={
                <div className="flex">
                  <span>{ROLE[rowData.role]}</span>
                </div>
              }
              buttonClassName={`!px-0 !justify-start hover:bg-surface-1 ${errors.role ? "border-danger-strong" : "border-none"}`}
              className="w-32 rounded-md p-0"
              input
            >
              {Object.entries(ROLE).map(([role, label]) => (
                <CustomSelect.Option key={role} value={Number(role)}>
                  {label}
                </CustomSelect.Option>
              ))}
            </CustomSelect>
          )}
        />
````
````new web/apps/web/core/components/workspace/settings/member-columns.tsx
        <CustomSelect
          value={rowData.role}
          // the select gives the picked option's value: a role's number, of WORKSPACE_ROLES
          onChange={(role: WorkspaceRole) => void changeRole(rowData.member.id, role)}
          label={
            <div className="flex">
              <span>{ROLE[rowData.role]}</span>
            </div>
          }
          buttonClassName="!px-0 !justify-start hover:bg-surface-1 border-none"
          className="w-32 rounded-md p-0"
          input
        >
          {WORKSPACE_ROLES.map((role) => (
            <CustomSelect.Option key={role} value={role}>
              {ROLE[role]}
            </CustomSelect.Option>
          ))}
        </CustomSelect>
````

`web/apps/web/core/components/workspace/settings/members-list-item.tsx`（整个文件，65 行）：

````whole web/apps/web/core/components/workspace/settings/members-list-item.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { isEmpty } from "lodash-es";
import { observer } from "mobx-react";
// nerve imports
import type { WorkspaceMember } from "@nerve/api-client";
import { Table } from "@nerve/ui";
// components
import { MembersLayoutLoader } from "@/components/ui/loader/layouts/members-layout-loader";
import { ConfirmWorkspaceMemberRemove } from "@/components/workspace/confirm-workspace-member-remove";
// hooks
import { useUser } from "@/hooks/store/user";
// components
import { useMemberColumns } from "@/components/workspace/settings/useMemberColumns";
// local imports
import { useMembershipChanges } from "./use-membership-changes";

type Props = {
  /** The workspace of the page's address, whose members these are. */
  workspaceSlug: string;
  memberDetails: (WorkspaceMember | null)[];
};

export const WorkspaceMembersListItem = observer(function WorkspaceMembersListItem(props: Props) {
  const { workspaceSlug, memberDetails } = props;
  const { columns, removeMemberModal, setRemoveMemberModal } = useMemberColumns();
  // store hooks
  const { data: currentUser } = useUser();
  const { leave, remove } = useMembershipChanges(workspaceSlug);

  // the caller's own row offers him to leave; another's, to remove that member (an admin's)
  const handleRemove = (memberId: string) => (memberId === currentUser?.id ? leave() : remove(memberId));

  if (isEmpty(columns)) return <MembersLayoutLoader />;

  return (
    <div className="grid border-t border-subtle">
      {removeMemberModal && (
        <ConfirmWorkspaceMemberRemove
          isOpen={removeMemberModal.member.id.length > 0}
          onClose={() => setRemoveMemberModal(null)}
          userDetails={{
            id: removeMemberModal.member.id,
            display_name: removeMemberModal.member.display_name || "",
          }}
          onSubmit={() => handleRemove(removeMemberModal.member.id)}
        />
      )}
      <Table<WorkspaceMember>
        columns={columns ?? []}
        data={memberDetails.filter((member): member is WorkspaceMember => member !== null)}
        keyExtractor={(rowData) => rowData?.member.id ?? ""}
        tHeadClassName="border-b border-subtle"
        thClassName="text-left font-medium divide-x-0 text-placeholder"
        tBodyClassName="divide-y-0"
        tBodyTrClassName="divide-x-0 p-4 h-10 text-secondary"
        tHeadTrClassName="divide-x-0"
      />
    </div>
  );
});
````

`web/apps/web/core/components/workspace/settings/members-list.tsx`（修改，1 处）：

````old web/apps/web/core/components/workspace/settings/members-list.tsx
        {searchedMemberIds?.length !== 0 && <WorkspaceMembersListItem memberDetails={memberDetails} />}
````
````new web/apps/web/core/components/workspace/settings/members-list.tsx
        {workspaceSlug && searchedMemberIds?.length !== 0 && (
          <WorkspaceMembersListItem workspaceSlug={workspaceSlug} memberDetails={memberDetails} />
        )}
````

`web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts`（新文件，100 行）：

````file web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { useMembershipChanges } from "./use-membership-changes";

// What the members page sends for each change of a membership, and what it does with nerve's answer (M3 design 7.1,
// 7.5): the hook runs as a plain function, with stand-ins for the stores' changes, which nerve answers when the test
// says. Its session is fake-tab.ts's.

const page = vi.hoisted(() => ({
  updateMember: vi.fn(),
  removeMemberFromWorkspace: vi.fn(),
  leaveWorkspace: vi.fn(),
  navigate: vi.fn(),
}));
const acme = workspaceOf("acme", { role: 20 });
vi.mock("react-router", () => ({ useNavigate: () => page.navigate }));
vi.mock("@/hooks/store/use-workspace", () => ({
  useWorkspace: () => ({
    getWorkspaceBySlug: (slug: string) => (slug === acme.slug ? acme : null),
    leaveWorkspace: page.leaveWorkspace,
  }),
}));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    workspace: { updateMember: page.updateMember, removeMemberFromWorkspace: page.removeMemberFromWorkspace },
  }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Each change the page makes, by its store change: the role's, the removal's and the leaving's. */
const changes = [
  { change: "a role", store: page.updateMember, make: () => useMembershipChanges("acme").changeRole("u-bob", 5) },
  {
    change: "a removal",
    store: page.removeMemberFromWorkspace,
    make: () => useMembershipChanges("acme").remove("u-bob"),
  },
  { change: "the leaving", store: page.leaveWorkspace, make: () => useMembershipChanges("acme").leave() },
];

beforeEach(() => {
  signedIn();
  for (const store of [page.updateMember, page.removeMemberFromWorkspace, page.leaveWorkspace]) {
    store.mockReset();
    store.mockResolvedValue(undefined);
  }
  page.navigate.mockClear();
  toasts.length = 0;
});

describe("useMembershipChanges", () => {
  it("sends a member's new role as its number, to the membership's workspace", async () => {
    await useMembershipChanges("acme").changeRole("u-bob", 5);
    expect(page.updateMember.mock.calls).toEqual([["acme", "u-bob", { role: 5 }]]);
    expect([page.navigate.mock.calls, toasts]).toEqual([[], []]);
  });

  it("removes a member of the workspace, and stays", async () => {
    await useMembershipChanges("acme").remove("u-bob");
    expect(page.removeMemberFromWorkspace.mock.calls).toEqual([["acme", "u-bob"]]);
    expect([page.navigate.mock.calls, toasts]).toEqual([[], []]);
  });

  it("leaves the workspace of the address, then lands at the root", async () => {
    await useMembershipChanges("acme").leave();
    expect([page.leaveWorkspace.mock.calls, page.navigate.mock.calls]).toEqual([[[acme]], [["/"]]]);
  });

  it.each(changes)("shows nerve's reason when it refuses $change, and stays", async ({ store, make }) => {
    store.mockRejectedValueOnce(refusal(409, "workspace.sole_admin"));
    await make();
    expect(page.navigate.mock.calls).toEqual([]);
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.workspace_sole_admin" }]);
  });

  describe.each(changes)("$change", ({ store, make }) => {
    it.each(lateSettlings)(
      "does nothing on the page when it $settles after another tab moved this one to another account",
      async ({ settle }) => {
        const change = heldChange<undefined>();
        store.mockReturnValueOnce(change.sent);
        const made = make();
        switchAccount();
        settle(change);
        await made;
        expect([page.navigate.mock.calls, toasts]).toEqual([[], []]);
      }
    );
  });
});
````

`web/apps/web/core/components/workspace/settings/use-membership-changes.ts`（新文件，49 行）：

````file web/apps/web/core/components/workspace/settings/use-membership-changes.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useNavigate } from "react-router";
import type { WorkspaceRole } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useWorkspace } from "@/hooks/store/use-workspace";
// lib
import { errorMessageKey } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";

/**
 * The changes the members page makes to the memberships of the workspace of workspaceSlug (M3 design 7.5): a member's
 * role, a member's removal, and the caller's leaving. Each shows nerve's refusal as its reason; the page follows each
 * only in the session it was sent in (M3 design 7.1): once another tab has moved this one to another account, the
 * page is that account's, and says nothing of the change.
 */
export function useMembershipChanges(workspaceSlug: string) {
  const navigate = useNavigate();
  const { getWorkspaceBySlug, leaveWorkspace } = useWorkspace();
  const {
    workspace: { updateMember, removeMemberFromWorkspace },
  } = useMember();
  const { t } = useTranslation();
  const failed = (error: unknown) =>
    setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) });

  return {
    /** Gives the member of userId the role: its number, as nerve's WorkspaceMemberUpdate takes it. */
    changeRole: (userId: string, role: WorkspaceRole) =>
      followInSession(() => updateMember(workspaceSlug, userId, { role }), { failed }),
    /** Ends the membership of the member of userId. */
    remove: (userId: string) => followInSession(() => removeMemberFromWorkspace(workspaceSlug, userId), { failed }),
    /**
     * Ends the caller's own membership; then the root lands him where his workspaces, as they are now, say (M3 design
     * 3.14).
     */
    leave: async () => {
      const workspace = getWorkspaceBySlug(workspaceSlug);
      if (!workspace) return;
      await followInSession(() => leaveWorkspace(workspace), { done: () => void navigate("/"), failed });
    },
  };
}
````

`web/apps/web/core/components/workspace/settings/useMemberColumns.tsx`（修改，1 处）：

````old web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
  return { columns, workspaceSlug, removeMemberModal, setRemoveMemberModal };
````
````new web/apps/web/core/components/workspace/settings/useMemberColumns.tsx
  return { columns, removeMemberModal, setRemoveMemberModal };
````

`web/apps/web/core/components/workspace/workspace-roles.ts`（新文件，13 行）：

````file web/apps/web/core/components/workspace/workspace-roles.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { WorkspaceRole } from "@nerve/api-client";

/**
 * A workspace's roles, the lowest first, as its role selects offer them, each labelled by ROLE: a select gives the
 * picked role's number, the WorkspaceRole nerve's WorkspaceMemberUpdate, InvitationCreate and WorkspaceInvitationUpdate
 * take (a string is refused).
 */
export const WORKSPACE_ROLES: WorkspaceRole[] = [5, 15, 20];
````

`web/apps/web/core/lib/in-session.ts`（修改，2 处）：

````old web/apps/web/core/lib/in-session.ts
/** What a page does once a change it sent has settled: with nerve's answer, or with why it failed. */
export type ChangeFollowers<T> = { done: (answer: T) => void; failed: (error: unknown) => void };
````
````new web/apps/web/core/lib/in-session.ts
/**
 * What a page does once a change it sent has settled: with nerve's answer (nothing, for a change whose answer the
 * stores show), or with why it failed.
 */
export type ChangeFollowers<T> = { done?: (answer: T) => void; failed: (error: unknown) => void };
````
````old web/apps/web/core/lib/in-session.ts
  if (outcome.settled === "done") followers.done(outcome.answer);
````
````new web/apps/web/core/lib/in-session.ts
  if (outcome.settled === "done") followers.done?.(outcome.answer);
````

- [ ] **Step 2: 静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/components/workspace/delete-workspace-modal.test.tsx"
````
````new .oxlintrc.json
        "web/apps/web/core/components/workspace/delete-workspace-modal.test.tsx",
        "web/apps/web/core/components/workspace/workspace-roles.ts"
````

- [ ] **Step 3: 端到端**

`e2e/fixtures/assert/workspace.ts`（修改，1 处）：

````old e2e/fixtures/assert/workspace.ts
  return id;
````
````new e2e/fixtures/assert/workspace.ts
  return id;
}

/** The workspace the account of email opened last, as the web app wrote it in the profile (M3 design 3.14), or null. */
export async function lastWorkspaceOf(db: Database, email: string): Promise<string | null> {
  const [profile] = await db.query<{ last_workspace_id: string | null }>(
    "SELECT p.last_workspace_id FROM profiles p JOIN users u ON u.id = p.user_id WHERE u.email = $1",
    [email]
  );
  return profile?.last_workspace_id ?? null;
````

`e2e/fixtures/workspace-pages.ts`（整个文件，108 行）：

````whole e2e/fixtures/workspace-pages.ts
import { expect, type Browser, type Locator, type Page, type Response } from "@playwright/test";

import { password, signInContext, type AuthTokens } from "./auth";
import { submitSignIn } from "./auth-pages";
import { watchPage, type PageWatch } from "./browser";
import { answerTo, sentTo } from "./settings-pages";

// The workspace's pages (M3 design 7.5), as a person uses them.

/**
 * A page of a browser of its own, signed in with tokens at the nerve of baseURL: another person's, beside the test's
 * page. Its context closes when the test's function returned by it is called.
 */
export async function anotherBrowser(
  browser: Browser,
  baseURL: string,
  tokens: AuthTokens
): Promise<{ page: Page; close: () => Promise<void> }> {
  const context = await browser.newContext({ baseURL });
  await signInContext(context, baseURL, tokens);
  return { page: await context.newPage(), close: () => context.close() };
}

/**
 * Signs the account of email in through the sign-in form, in a browser of its own at the nerve of baseURL, watched
 * from its first page: resolves once nerve has let it in. Its context closes when the function returned is called.
 */
export async function signInAnew(
  browser: Browser,
  baseURL: string,
  email: string
): Promise<{ page: Page; watch: PageWatch; close: () => Promise<void> }> {
  const context = await browser.newContext({ baseURL });
  const page = await context.newPage();
  const watch = await watchPage(page);
  await page.goto("/");
  expect(await submitSignIn(page, email, password), `the sign-in of ${email}`).toBe(200);
  return { page, watch, close: () => context.close() };
}

/**
 * Opens the workspace of id through the workspace menu of page, which shows a workspace: resolves with what the page
 * sent as the workspace opened last, and nerve's answer.
 */
export async function switchWorkspace(page: Page, id: string): Promise<{ body: unknown; answer: Response }> {
  await page.getByRole("button", { name: "Open workspace switcher" }).click();
  return sentTo(page, "PATCH", "/api/v0/me/profile", () => page.locator(`a[id="${id}"]`).click());
}

/**
 * Confirms the deletion of the workspace whose name is name on its general page, which page shows: the admin opens the
 * deletion, types the name and the words that confirm it, and confirms. Resolves once the confirmation is clicked, not
 * waiting for nerve's answer.
 */
export async function confirmDeletion(page: Page, name: string): Promise<void> {
  await page.getByRole("button", { name: "Delete", exact: true }).click();
  await page.locator("#workspaceName").fill(name);
  await page.locator("#confirmDelete").fill("delete my workspace");
  await page.getByRole("button", { name: "Confirm", exact: true }).click();
}

/**
 * Deletes the workspace of slug, whose name is name, from its general page, which page shows (confirmDeletion).
 * Resolves with nerve's answer to the deletion.
 */
export function deleteFromGeneralPage(page: Page, slug: string, name: string): Promise<Response> {
  return answerTo(page, "DELETE", `/api/v0/workspaces/${slug}`, () => confirmDeletion(page, name));
}

/** The row of the members page, which page shows, of the member whose address is email. */
export function memberRow(page: Page, email: string): Locator {
  return page.locator("tr", { hasText: email });
}

/**
 * Picks the role named to in the role select of the row of email on the members page, which page shows and where the
 * select shows the role named from: resolves with what the page sent to the membership of id, and nerve's answer.
 */
export async function pickRole(
  page: Page,
  email: string,
  id: string,
  role: { from: string; to: string }
): Promise<{ body: unknown; answer: Response }> {
  await memberRow(page, email).getByRole("button", { name: role.from, exact: true }).click();
  return sentTo(page, "PATCH", `/api/v0/workspace-members/${id}`, () =>
    page.getByRole("option", { name: role.to, exact: true }).click()
  );
}

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
  return answerTo(page, request.method, request.path, () =>
    page.getByRole("dialog").getByRole("button", { name: entry, exact: true }).click()
  );
}
````

`e2e/stories/smoke/s2-web-app.spec.ts`（修改，6 处）：

````old e2e/stories/smoke/s2-web-app.spec.ts
const ARCHIVED = ["GET /api/v0/workspaces/{slug}/projects?archived=true"];
````
````new e2e/stories/smoke/s2-web-app.spec.ts
const ARCHIVED = ["GET /api/v0/workspaces/{slug}/projects?archived=true"];
/** The members page lists the workspace's invitations for an admin alone, as nerve shows them to no one else. */
const INVITATIONS = ["GET /api/v0/workspaces/{slug}/invitations"];
````
````old e2e/stories/smoke/s2-web-app.spec.ts
 * call of its own fetch no other check holds; the workspace's general settings; then the project's settings.
````
````new e2e/stories/smoke/s2-web-app.spec.ts
 * call of its own fetch no other check holds; the workspace's general settings and its members; then the project's
 * settings.
````
````old e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/projects", [...APP, ...WORKSPACE, ...ARCHIVED]],
    ["/{slug}/settings", [...APP, ...WORKSPACE, ...GENERAL]],
````
````new e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/projects", [...APP, ...WORKSPACE, ...ARCHIVED]],
    ["/{slug}/settings", [...APP, ...WORKSPACE, ...GENERAL]],
    ["/{slug}/settings/members", [...APP, ...WORKSPACE, ...INVITATIONS]],
````
````old e2e/stories/smoke/s2-web-app.spec.ts
    ["/", [...APP, ...WORKSPACE]],
    ["/{slug}/settings", [...APP, ...WORKSPACE, ...GENERAL]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL]],
````
````new e2e/stories/smoke/s2-web-app.spec.ts
    ["/", [...APP, ...WORKSPACE]],
    ["/{slug}/settings", [...APP, ...WORKSPACE, ...GENERAL]],
    ["/{slug}/settings/members", [...APP, ...WORKSPACE]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL]],
````
````old e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings", [...APP, ...WORKSPACE]],
````
````new e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings", [...APP, ...WORKSPACE]],
    ["/{slug}/settings/members", [...APP, ...WORKSPACE]],
````
````old e2e/stories/smoke/s2-web-app.spec.ts
    ["/", [...APP, ...WORKSPACE]],
    ["/{slug}/settings", [...APP, ...WORKSPACE, ...GENERAL]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT]],
````
````new e2e/stories/smoke/s2-web-app.spec.ts
    ["/", [...APP, ...WORKSPACE]],
    ["/{slug}/settings", [...APP, ...WORKSPACE, ...GENERAL]],
    ["/{slug}/settings/members", [...APP, ...WORKSPACE]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT]],
````

`e2e/stories/workspace/w2-landing.spec.ts`（修改，3 处）：

````old e2e/stories/workspace/w2-landing.spec.ts
import { expectMembershipEnded, expectWorkspaceDeleted, expectWrittenLastBy } from "../../fixtures/assert/workspace";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
````
````new e2e/stories/workspace/w2-landing.spec.ts
import {
  expectMembershipEnded,
  expectWorkspaceDeleted,
  expectWrittenLastBy,
  lastWorkspaceOf,
} from "../../fixtures/assert/workspace";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole } from "../../fixtures/browser";
import { registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import {
  deleteFromGeneralPage,
  endMembership,
  pickRole,
  signInAnew,
  switchWorkspace,
} from "../../fixtures/workspace-pages";
````
````old e2e/stories/workspace/w2-landing.spec.ts
// W2, where an account lands once signed in (M3 design 2). The landing rule is the web app's (P9); the API version
// checks the list it rests on.
````
````new e2e/stories/workspace/w2-landing.spec.ts
// W2, where an account lands once signed in (M3 design 2). The landing rule is the web app's: the page version
// follows it; the API version checks the list it rests on.
````
````old e2e/stories/workspace/w2-landing.spec.ts
  expect(await listed(bob)).toEqual([entry(secondWorkspace, 20, 1)]);
});

````
````new e2e/stories/workspace/w2-landing.spec.ts
  expect(await listed(bob)).toEqual([entry(secondWorkspace, 20, 1)]);
});

test("W2 (page): signed in, an account lands on the workspace it opened last; once that is deleted, on its other one; once it has made a member that one's admin and left it, on the page that creates one; no request fails", async ({
  api,
  baseURL,
  browser,
  db,
}, testInfo) => {
  const aliceEmail = emailFor(testInfo, "alice");
  const alice = await registerOnboarded(api, aliceEmail);
  const bobEmail = emailFor(testInfo, "bob");
  const bob = await registerOnboarded(api, bobEmail);
  const first = slugFor(testInfo, "first");
  const second = slugFor(testInfo, "second");
  await createWorkspace(api, alice.access_token, { name: "First", slug: first });
  await inviteAndAccept(api, alice.access_token, first, { email: bobEmail, token: bob.access_token }, 15);
  const secondWorkspace = await createWorkspace(api, alice.access_token, { name: "Second", slug: second });
  const bobsMembership = await membershipOf(api, alice.access_token, first, await accountId(api, bob.access_token));
  /** Signs alice in, in a browser of her own, and checks where she lands. */
  const signIn = async (landing: string) => {
    const signedIn = await signInAnew(browser, baseURL ?? "", aliceEmail);
    await expect(signedIn.page).toHaveURL(landing);
    return signedIn;
  };
  /**
   * Nothing failed in the browser of signedIn, where loads documents opened a workspace's page (each logs
   * EMOJI_CHECK_WARNING); then it closes.
   */
  const nothingFailed = async ({ page, watch, close }: Awaited<ReturnType<typeof signIn>>, loads: number) => {
    expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
    await expectQuietConsole(page, watch, { warnings: Array.from({ length: loads }, () => EMOJI_CHECK_WARNING) });
    await close();
  };

  // She has opened neither: she lands on First, created first. She opens Second through the workspace menu, and the
  // web app writes it down as the one she opened last.
  const one = await signIn(`/${first}`);
  const switched = await switchWorkspace(one.page, secondWorkspace.id);
  expect([switched.answer.status(), switched.body]).toEqual([200, { last_workspace_id: secondWorkspace.id }]);
  await expect(one.page).toHaveURL(`/${second}`);
  expect(await lastWorkspaceOf(db, aliceEmail)).toBe(secondWorkspace.id);
  await nothingFailed(one, 1);

  // Signed in again, she lands on Second, and deletes it from its general page: the root lands her on First.
  const two = await signIn(`/${second}`);
  await two.page.goto(`/${second}/settings`);
  expect((await deleteFromGeneralPage(two.page, second, "Second")).status()).toBe(204);
  await expect(two.page).toHaveURL(`/${first}`);
  await nothingFailed(two, 2);

  // Signed in again, she lands on First. She makes bob its admin, and leaves it: the root lands her on the page that
  // creates a workspace, as it does when she signs in again.
  const three = await signIn(`/${first}`);
  await three.page.goto(`/${first}/settings/members`);
  const promoted = await pickRole(three.page, bobEmail, bobsMembership, { from: "Member", to: "Admin" });
  expect([promoted.answer.status(), promoted.body]).toEqual([200, { role: 20 }]);
  const leaving = { method: "POST", path: `/api/v0/workspaces/${first}/leave` };
  expect((await endMembership(three.page, aliceEmail, "Leave", leaving)).status()).toBe(204);
  await expect(three.page).toHaveURL("/create-workspace");
  await nothingFailed(three, 2);
  await nothingFailed(await signIn("/create-workspace"), 0);
});

````

`e2e/stories/workspace/w7-member-management.spec.ts`（修改，7 处）：

````old e2e/stories/workspace/w7-member-management.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
````
````new e2e/stories/workspace/w7-member-management.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import type { Database } from "../../fixtures/db";
import { registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser, endMembership, memberRow, pickRole } from "../../fixtures/workspace-pages";
````
````old e2e/stories/workspace/w7-member-management.spec.ts
// W7, the members' management (M3 design 2). The page version comes with the members' page (P9).
````
````new e2e/stories/workspace/w7-member-management.spec.ts
// W7, the members' management (M3 design 2).

/**
 * The memberships of the account of email of the projects of the workspace of slug, ended ones too: each one's role,
 * whether it is active, and who wrote it last.
 */
const projectRoles = (db: Database, slug: string, email: string) =>
  db.query(
    `SELECT p.identifier, m.role, m.is_active, b.email AS by
       FROM project_members m JOIN projects p ON p.id = m.project_id JOIN workspaces w ON w.id = p.workspace_id
       JOIN users u ON u.id = m.member_id JOIN users b ON b.id = m.updated_by_id
      WHERE w.slug = $1 AND u.email = $2 AND m.deleted_at IS NULL ORDER BY p.identifier COLLATE "C"`,
    [slug, email]
  );
````
````old e2e/stories/workspace/w7-member-management.spec.ts
  // The account of email's memberships of acme's projects, ended ones too: each one's role, whether it is active,
  // and who wrote it last.
  const projectRoles = (email: string) =>
    db.query(
      `SELECT p.identifier, m.role, m.is_active, b.email AS by
         FROM project_members m JOIN projects p ON p.id = m.project_id JOIN workspaces w ON w.id = p.workspace_id
         JOIN users u ON u.id = m.member_id JOIN users b ON b.id = m.updated_by_id
        WHERE w.slug = $1 AND u.email = $2 AND m.deleted_at IS NULL ORDER BY p.identifier COLLATE "C"`,
      [slug, email]
    );
  expect(await projectRoles(erin.email), "erin's membership of Docs, an admin's, ended by her leaving").toEqual([
    { identifier: "DOCS", role: 20, is_active: false, by: erin.email },
  ]);
````
````new e2e/stories/workspace/w7-member-management.spec.ts
  expect(
    await projectRoles(db, slug, erin.email),
    "erin's membership of Docs, an admin's, ended by her leaving"
  ).toEqual([{ identifier: "DOCS", role: 20, is_active: false, by: erin.email }]);
````
````old e2e/stories/workspace/w7-member-management.spec.ts
  expect(await projectRoles(bob.email), "bob's memberships of the projects, Ops's admin's").toEqual([
````
````new e2e/stories/workspace/w7-member-management.spec.ts
  expect(await projectRoles(db, slug, bob.email), "bob's memberships of the projects, Ops's admin's").toEqual([
````
````old e2e/stories/workspace/w7-member-management.spec.ts
  expect(await projectRoles(bob.email), "bob's memberships of the projects").toEqual([
````
````new e2e/stories/workspace/w7-member-management.spec.ts
  expect(await projectRoles(db, slug, bob.email), "bob's memberships of the projects").toEqual([
````
````old e2e/stories/workspace/w7-member-management.spec.ts
  expect(await projectRoles(carol.email), "carol's memberships of the projects").toEqual([
````
````new e2e/stories/workspace/w7-member-management.spec.ts
  expect(await projectRoles(db, slug, carol.email), "carol's memberships of the projects").toEqual([
````
````old e2e/stories/workspace/w7-member-management.spec.ts
  ]);
});

````
````new e2e/stories/workspace/w7-member-management.spec.ts
  ]);
});

test("W7 (page): the admin makes a member a guest, in the workspace and in each of his projects, and removes another, who comes back by a new invitation; his own row offers no role and no removal, and his leaving, as the only admin, is refused with the reason; a member sees the roles and changes none", async ({
  api,
  baseURL,
  browser,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  const join = async (name: string) => {
    const email = emailFor(testInfo, name);
    const tokens = await registerOnboarded(api, email);
    await inviteAndAccept(api, admin.access_token, slug, { email, token: tokens.access_token }, 15);
    return { email, tokens, id: await accountId(api, tokens.access_token) };
  };
  const bob = await join("bob");
  const carol = await join("carol");
  // bob is a member of Web, and the admin of Ops, his own.
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.access_token, web.id, [{ member_id: bob.id, role: 15 }]);
  await createProject(api, bob.tokens.access_token, slug, { name: "Ops", identifier: "OPS" });
  const membership = (id: string) => membershipOf(api, admin.access_token, slug, id);
  const members = `/${slug}/settings/members`;

  // carol, a member, sees each role as text; her own row's menu offers her leaving.
  const theMember = await anotherBrowser(browser, baseURL ?? "", carol.tokens);
  const memberWatch = await watchPage(theMember.page);
  await theMember.page.goto(members);
  await expect(memberRow(theMember.page, bob.email)).toContainText("Member");
  await expect(memberRow(theMember.page, bob.email).locator("button")).toHaveCount(0);
  await expect(memberRow(theMember.page, adminEmail).locator("button")).toHaveCount(0);
  await memberRow(theMember.page, carol.email).locator("button").first().click();
  await expect(theMember.page.getByRole("button", { name: "Leave", exact: true })).toBeVisible();
  expect([memberWatch.apiFailures, memberWatch.oldApiRequests, memberWatch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(theMember.page, memberWatch, { warnings: [EMOJI_CHECK_WARNING] });
  await theMember.close();

  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  await page.goto(members);
  // His own row: his role as text, nothing to pick.
  await expect(memberRow(page, adminEmail)).toContainText("Admin");
  await expect(memberRow(page, adminEmail).getByRole("button", { name: "Admin" })).toHaveCount(0);

  // bob becomes a guest: the page sends the role's number (nerve refuses a string), and so is he in each project.
  const demoted = await pickRole(page, bob.email, await membership(bob.id), { from: "Member", to: "Guest" });
  expect([demoted.answer.status(), demoted.body]).toEqual([200, { role: 5 }]);
  await expect(memberRow(page, bob.email).getByRole("button", { name: "Guest", exact: true })).toBeVisible();
  await expectMembership(db, slug, bob.email, { role: 5, is_active: true });
  expect(await projectRoles(db, slug, bob.email)).toEqual([
    { identifier: "OPS", role: 5, is_active: true, by: adminEmail },
    { identifier: "WEB", role: 5, is_active: true, by: adminEmail },
  ]);

  // carol is removed: her membership ends, its row kept; a new invitation brings her back, with its role.
  const carols = await membership(carol.id);
  const removal = { method: "DELETE", path: `/api/v0/workspace-members/${carols}` };
  expect((await endMembership(page, carol.email, "Remove", removal)).status()).toBe(204);
  await expect(memberRow(page, carol.email)).toContainText("Suspended");
  await expectMembershipEnded(db, slug, carol.email, adminEmail, 15, []);
  await inviteAndAccept(api, admin.access_token, slug, { email: carol.email, token: carol.tokens.access_token }, 5);
  await expectMembership(db, slug, carol.email, { role: 5, is_active: true });

  // His leaving, as the only admin: nerve refuses it, and the page says why and stays.
  const leaving = { method: "POST", path: `/api/v0/workspaces/${slug}/leave` };
  expect((await endMembership(page, adminEmail, "Leave", leaving)).status()).toBe(409);
  await expect(
    page.getByText("The workspace would be left without an admin. Make another member an admin first.")
  ).toBeVisible();
  await expect(page).toHaveURL(members);
  await expectMembership(db, slug, adminEmail, { role: 20, is_active: true });
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`409 ${leaving.method} ${leaving.path}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING],
    errors: ["Failed to load resource: the server responded with a status of 409 (Conflict)"],
  });
});

````

- [ ] **Step 4: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、3 个例外，没有命中；web 的 oxlint 360 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 79 个全部通过。

- [ ] **Step 5: 提交**

```bash
git add .oxlintrc.json e2e/fixtures/assert/workspace.ts e2e/fixtures/workspace-pages.ts e2e/stories/smoke/s2-web-app.spec.ts e2e/stories/workspace/w2-landing.spec.ts e2e/stories/workspace/w7-member-management.spec.ts web/apps/web/core/components/workspace/settings/member-columns.tsx web/apps/web/core/components/workspace/settings/members-list-item.tsx web/apps/web/core/components/workspace/settings/members-list.tsx web/apps/web/core/components/workspace/settings/use-membership-changes.test.ts web/apps/web/core/components/workspace/settings/use-membership-changes.ts web/apps/web/core/components/workspace/settings/useMemberColumns.tsx web/apps/web/core/components/workspace/workspace-roles.ts web/apps/web/core/lib/in-session.ts
```
```bash
git commit -m "feat(M3/P9): the members page changes a role, removes and leaves through one hook, in its session

useMembershipChanges sends a member's role as its number, a removal and
the caller's leaving to the address's workspace, says nerve's reason for
a refusal, and follows each only in the session it was sent in; leaving
lands at the root. The role column drops the form it never submitted and
its cast; the roles offered are WORKSPACE_ROLES. W7's and W2's page
versions, and S2's members pages, where only an admin asks for the
workspace's invitations. The roles' module joins the non-null override.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p9.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T2.1` | 角色选择给出字符串形式的角色编号 | 故事 W7、故事 W2 | 端到端 |
| `T2.2` | 成员页丢掉 nerve 对成员关系修改的拒绝 | `use-membership-changes.test.ts`、故事 W7 | vitest；端到端 |
| `T2.3` | 离开工作区之后不论会话都回到根路径 | `use-membership-changes.test.ts` | vitest |
| `T2.4` | 管理员自己的一行可以改自己的角色 | 故事 W7 | 端到端 |
| `T2.5` | 成员看到的页面可以改别人的角色 | 故事 W7 | 端到端 |
| `T2.6` | 自己的一行显示"移出"而不是"离开" | 故事 W7、故事 W2 | 端到端 |
| `T2.7` | 离开工作区之后停在原页 | `use-membership-changes.test.ts`、故事 W2 | vitest；端到端 |
| `T2.8` | 成员页对每个角色都取邀请 | 故事 S2 | 端到端 |
| `T2.9` | 工作区的角色（`workspace-roles.ts`，范围内）用非空断言 | oxlint（`check:lint`） | 静态 |

---

### Task 3: 邀请的表单和列表；W4 的页面版本

**Files:**
- Create: `web/apps/web/core/components/workspace/invite-modal/refusal.test.ts`、`web/apps/web/core/components/workspace/invite-modal/refusal.ts`、`web/apps/web/core/components/workspace/members/invite-modal.test.tsx`、`web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts`、`web/apps/web/core/components/workspace/settings/use-invitation-changes.ts`、`web/apps/web/core/hooks/use-copy-invitation-link.ts`、`web/apps/web/core/lib/invitation-link.test.ts`、`web/apps/web/core/lib/invitation-link.ts`
- Modify: `.oxlintrc.json`、`e2e/fixtures/api.ts`、`e2e/fixtures/assert/workspace.ts`、`e2e/fixtures/workspace-pages.ts`、`e2e/stories/workspace/w4-invite-members.spec.ts`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx`、`web/apps/web/core/components/workspace/invite-modal/fields.tsx`、`web/apps/web/core/components/workspace/members/invite-modal.tsx`、`web/apps/web/core/components/workspace/settings/invitations-list-item.tsx`、`web/apps/web/core/hooks/use-workspace-invitation.tsx`、`web/apps/web/package.json`、`web/packages/i18n/src/locales/en/workspace-settings.json`、`web/packages/i18n/src/locales/zh-CN/workspace-settings.json`

**Interfaces:**
- Produces（spec 2.3；M3 设计 2 的 W4，7.1，7.4，7.5；P8a 的 P19、P21、P22）：
  - `core/hooks/use-workspace-invitation.tsx`（整个文件）：表单的值就是生成的 `WorkspaceInvitationsCreate`（`{ invitations: InvitationCreate[] }`；Plane 的 `InvitationFormValues`、`EmailRole` 和 `role: EUserPermissions` 删除，页面原来把 `data.emails` 改名发出）。`useWorkspaceInvitationActions({ invite, onSent })`：提交经 `followInSession(() => invite(data), …)`；成功时清空、提示，把 nerve 建的邀请交给 `onSent(invitations)`（表单所在的地方决定之后做什么：弹窗关闭，Task 6 的新手引导列出链接）；`clear()` 清空表单（弹窗在它离场的过渡之后调；原来 hook 的 `handleClose` 知道弹窗的 350 毫秒）；被拒绝时按 `invitationRefusal` 把原因放在 nerve 指出的行下（`setError("invitations.<i>.email", …)`），否则提示。页面原来在 `catch` 里读 `err.error`（P21）再重新抛出（没有处理的拒绝）。`watch` 不再给出（没有读者）；oxlint 的一条警告随之清零。
  - `core/components/workspace/invite-modal/refusal.ts`：`invitationRefusal(error, rows): InvitationRefusal`，`{ kind: "rows"; rows: { index; message }[] } | { kind: "toast"; message }`：nerve 的字段错误都指向表单里的行（`invitations[<i>].email`）时落到行下：`not_allowed` 是"已是成员"，`duplicate` 是"已有邀请"（待接受、已忽略或表单里重复），其余按 `FIELD_ERROR_MESSAGES`；有任何一条不在行上时，整个拒绝按 `errorMessageKey` 提示。
  - `core/lib/invitation-link.ts`：`invitationLink(origin, invitation)`，邀请链接（`/workspace-invitations?invitation_id=…&token=…`，查询参数经 `URLSearchParams` 编码；原来在列表项里拼字符串，令牌不编码）。
  - `SendWorkspaceInvitationModal`（`members/invite-modal.tsx`）收 `invite: (data: WorkspaceInvitationsCreate) => Promise<WorkspaceInvitation[]>`（原来是 `onSubmit`），`onSent` 是它的 `onClose`，关闭时自己在过渡之后清空表单；成员页传 `(data) => inviteMembersToWorkspace(workspaceSlug, data)`，它自己的 `handleWorkspaceInvite` 删除。`invite-modal/fields.tsx` 的行名是 `invitations.<i>.*`，角色的选项是 `WORKSPACE_ROLES`。
  - `core/components/workspace/settings/use-invitation-changes.ts`：`useInvitationChanges()` 给出 `{ changeRole(workspaceSlug, invitationId, role: WorkspaceRole), remove(workspaceSlug, invitationId) }`（参数照 store 的修改：列表项在地址没有工作区时不渲染，hook 先于这个判断调用）：改角色发 `{ role }`（编号），删除之后提示；都经 `followInSession`，拒绝按 `code` 提示（原来读 `err.error`）。与 Task 2 的 `useMembershipChanges` 同一形状，跟进由它自己的 vitest 守着（预检 M1）。
  - `settings/invitations-list-item.tsx`：已忽略的邀请（`responded_at` 不为空）标"已忽略"，没有角色选择和复制链接，只能删除（nerve 不让改、不让接受它）；删除和改角色经 `useInvitationChanges`；复制链接经 `useCopyInvitationLink`。
  - `core/hooks/use-copy-invitation-link.ts`：`useCopyInvitationLink()`，复制邀请链接（`invitationLink`，本页的 origin），成功、失败都提示（原来失败只有 `console.error`）；Task 6 的邀请一步也用它。
  - i18n（en、zh-CN 的 `workspace-settings.json`）：`members.declined`，`members.modal.errors.already_member`、`already_invited`。
  - `web/apps/web/package.json`：web 的 oxlint 上限 360 → 359。
  - 静态检查（总体设计 7.7）：`.oxlintrc.json` 的非空断言的范围加上 `invitation-link.ts`、`invite-modal/refusal.ts` 和它们的测试、`members/invite-modal.test.tsx`、`use-copy-invitation-link.ts`（`use-invitation-changes.ts` 和它的测试在 `workspace/settings/**` 之内）。
- e2e 的共用部分：`e2e/fixtures/api.ts` 加 `decline`、`invitationTo`（管理员列出的某个邮箱的邀请，带链接的令牌）；`workspace-pages.ts` 加 `invitationLinkOf`（链接的路径；W4、W5、W1、W6 共用）、`invitationRow`、`sendInvitations`、`removeInvitation`；`assert/workspace.ts` 的 `expectInvitations` 让同一邮箱的两条邀请按创建时间排（W4 的页面版本里同一邮箱先后两条，原来的顺序不定，偶尔失败）。

**Tests:**
- vitest：`refusal.test.ts`：`invitationRefusal` 的 `shows $refusal`（`it.each`，两行的表单：行上的已是成员和已有邀请、行上的格式不对、一行的角色错误（表单没有它的文案，整个按原因提示）、表单没有的行、没有字段的拒绝、没有回答的失败）。`invitation-link.test.ts`：`opens the invitation page at the origin, with the invitation's id and token`；`gives back the token %s as it was`（`a+b`、`a&token=b`、`a b/c=?#d`、`雪` 经 `URL` 的查询读回原样）。`members/invite-modal.test.tsx`（`SendWorkspaceInvitationModal`）：`sends its rows as nerve's WorkspaceInvitationsCreate, each role a number; then closes, and says so`；`sends nothing for an address that is none`；`stays open, with no toast, when nerve refuses a row`（行下的原因由 W4 核对：服务端渲染看不到之后才设的错误）；`stays open, and shows nerve's reason, when it refuses the invitations`；`neither closes nor speaks when the invitations' answer $settles after another tab moved this one to another account`。`settings/use-invitation-changes.test.ts`：`sends a pending invitation's new role as its number, to the invitation's workspace`；`deletes an invitation of the workspace, and says so`；`shows nerve's reason when it refuses $change`（两种修改各一行）；`does nothing on the page when it $settles after another tab moved this one to another account`（两种修改各两行）。
- 端到端：`W4 (page): the admin invites a member and a guest, changes a role and copies a link; an invitation deleted meanwhile is refused with the reason; …`（成员的页面不显示邀请，也不请求它；已忽略的邀请标"Declined"，没有角色选择；两行一次发出，请求体经 `sentTo`，角色是编号；改角色的请求体 `{ role: 15 }`；复制的链接与 `invitationLinkOf` 相同；经 API 先删掉的邀请，页面的删除得到 404，提示原因；已是成员的邮箱、已忽略的邀请的邮箱被拒绝，原因在各自的行下，表单不关；删除已忽略的那条之后可以再邀请）。

- [ ] **Step 1: 链接、拒绝和表单**

`web/apps/web/core/components/workspace/invite-modal/fields.tsx`（修改，10 处）：

````old web/apps/web/core/components/workspace/invite-modal/fields.tsx
import { Input, InputGroup } from "@makeplane/propel/components/input";
````
````new web/apps/web/core/components/workspace/invite-modal/fields.tsx
import { Input, InputGroup } from "@makeplane/propel/components/input";
import type { WorkspaceInvitationsCreate } from "@nerve/api-client";
````
````old web/apps/web/core/components/workspace/invite-modal/fields.tsx
// hooks
import type { InvitationFormValues } from "@/hooks/use-workspace-invitation";
````
````new web/apps/web/core/components/workspace/invite-modal/fields.tsx
// local imports
import { WORKSPACE_ROLES } from "../workspace-roles";
````
````old web/apps/web/core/components/workspace/invite-modal/fields.tsx
  fields: FieldArrayWithId<InvitationFormValues, "emails", "id">[];
  control: Control<InvitationFormValues>;
  formState: FormState<InvitationFormValues>;
````
````new web/apps/web/core/components/workspace/invite-modal/fields.tsx
  fields: FieldArrayWithId<WorkspaceInvitationsCreate, "invitations", "id">[];
  control: Control<WorkspaceInvitationsCreate>;
  formState: FormState<WorkspaceInvitationsCreate>;
````
````old web/apps/web/core/components/workspace/invite-modal/fields.tsx
              control={control}
              name={`emails.${index}.email`}
````
````new web/apps/web/core/components/workspace/invite-modal/fields.tsx
              control={control}
              name={`invitations.${index}.email`}
````
````old web/apps/web/core/components/workspace/invite-modal/fields.tsx
                  <Field name="input" invalid={Boolean(errors.emails?.[index]?.email)}>
````
````new web/apps/web/core/components/workspace/invite-modal/fields.tsx
                  <Field name="input" invalid={Boolean(errors.invitations?.[index]?.email)}>
````
````old web/apps/web/core/components/workspace/invite-modal/fields.tsx
                        id={`emails.${index}.email`}
                        name={`emails.${index}.email`}
````
````new web/apps/web/core/components/workspace/invite-modal/fields.tsx
                        id={`invitations.${index}.email`}
                        name={`invitations.${index}.email`}
````
````old web/apps/web/core/components/workspace/invite-modal/fields.tsx
                  {errors.emails?.[index]?.email && (
````
````new web/apps/web/core/components/workspace/invite-modal/fields.tsx
                  {errors.invitations?.[index]?.email && (
````
````old web/apps/web/core/components/workspace/invite-modal/fields.tsx
                      {errors.emails?.[index]?.email?.message}
````
````new web/apps/web/core/components/workspace/invite-modal/fields.tsx
                      {errors.invitations?.[index]?.email?.message}
````
````old web/apps/web/core/components/workspace/invite-modal/fields.tsx
                name={`emails.${index}.role`}
````
````new web/apps/web/core/components/workspace/invite-modal/fields.tsx
                name={`invitations.${index}.role`}
````
````old web/apps/web/core/components/workspace/invite-modal/fields.tsx
                    {/* every role: only an admin invites (the members page's gate), and an admin may give any */}
                    {Object.entries(ROLE).map(([key, label]) => (
                      <CustomSelect.Option key={key} value={parseInt(key)}>
                        {label}
````
````new web/apps/web/core/components/workspace/invite-modal/fields.tsx
                    {/* every role: only an admin invites (the members page's gate), and an admin may give any; the select
                    gives the picked option's value, a role's number */}
                    {WORKSPACE_ROLES.map((role) => (
                      <CustomSelect.Option key={role} value={role}>
                        {ROLE[role]}
````

`web/apps/web/core/components/workspace/invite-modal/refusal.test.ts`（新文件，64 行）：

````file web/apps/web/core/components/workspace/invite-modal/refusal.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import type { FieldError } from "@nerve/api-client";
import { refusal } from "@/lib/fake-refusal";
import { invitationRefusal, type InvitationRefusal } from "./refusal";

// What the invitation form shows of nerve's refusal (M3 design 2, W4), for a form of two rows.

/** nerve's refusal of a batch, 422, naming errors. */
const refused = (...errors: Pick<FieldError, "field" | "code">[]) => refusal(422, "validation_failed", errors);
const toast = (message: string): InvitationRefusal => ({ kind: "toast", message });

describe("invitationRefusal", () => {
  it.each<{ refusal: string; error: unknown; shown: InvitationRefusal }>([
    {
      refusal: "a member's address and one invited already, each under its row",
      error: refused(
        { field: "invitations[1].email", code: "duplicate" },
        { field: "invitations[0].email", code: "not_allowed" }
      ),
      shown: {
        kind: "rows",
        rows: [
          { index: 1, message: "workspace_settings.settings.members.modal.errors.already_invited" },
          { index: 0, message: "workspace_settings.settings.members.modal.errors.already_member" },
        ],
      },
    },
    {
      refusal: "another fault of an address, under its row",
      error: refused({ field: "invitations[0].email", code: "invalid_format" }),
      shown: { kind: "rows", rows: [{ index: 0, message: "errors.field.invalid_format" }] },
    },
    {
      refusal: "a fault of a row's role, which the form has no message for, as nerve's reason",
      error: refused(
        { field: "invitations[0].email", code: "duplicate" },
        { field: "invitations[1].role", code: "invalid_format" }
      ),
      shown: toast("errors.validation_failed"),
    },
    {
      refusal: "a row the form does not have, as nerve's reason",
      error: refused({ field: "invitations[2].email", code: "duplicate" }),
      shown: toast("errors.validation_failed"),
    },
    {
      refusal: "a refusal that names no field, as nerve's reason",
      error: refusal(403, "forbidden"),
      shown: toast("errors.forbidden"),
    },
    {
      refusal: "a failure without an answer, as unknown",
      error: new TypeError("offline"),
      shown: toast("errors.unknown"),
    },
  ])("shows $refusal", ({ error, shown }) => {
    expect(invitationRefusal(error, 2)).toEqual(shown);
  });
});
````

`web/apps/web/core/components/workspace/invite-modal/refusal.ts`（新文件，37 行）：

````file web/apps/web/core/components/workspace/invite-modal/refusal.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { FieldError } from "@nerve/api-client";
// lib
import { ApiError } from "@/lib/api-error";
import { FIELD_ERROR_MESSAGES, errorMessageKey } from "@/lib/error-messages";

/**
 * What the invitation form shows of nerve's refusal of its invitations (M3 design 2, W4): the message under each row
 * nerve names, when it names rows of the form alone; else nerve's reason, in a toast. Each message is an i18n key.
 */
export type InvitationRefusal =
  | { kind: "rows"; rows: { index: number; message: string }[] }
  | { kind: "toast"; message: string };

/**
 * Why an address of a row cannot be invited, by nerve's code for its email: it is an active member's; or it is
 * invited already, pending or declined, or twice in the form.
 */
const ROW_MESSAGES: Partial<Record<FieldError["code"], string>> = {
  not_allowed: "workspace_settings.settings.members.modal.errors.already_member",
  duplicate: "workspace_settings.settings.members.modal.errors.already_invited",
};

/** The refusal of a form of rows rows, as the form shows it. */
export function invitationRefusal(error: unknown, rows: number): InvitationRefusal {
  const named = error instanceof ApiError ? (error.problem?.errors ?? []) : [];
  const onRows = named.flatMap(({ field, code }) => {
    const index = Number(/^invitations\[(\d+)\]\.email$/.exec(field)?.[1] ?? -1);
    return index >= 0 && index < rows ? [{ index, message: ROW_MESSAGES[code] ?? FIELD_ERROR_MESSAGES[code] }] : [];
  });
  if (onRows.length > 0 && onRows.length === named.length) return { kind: "rows", rows: onRows };
  return { kind: "toast", message: errorMessageKey(error) };
}
````

`web/apps/web/core/hooks/use-copy-invitation-link.ts`（新文件，33 行）：

````file web/apps/web/core/hooks/use-copy-invitation-link.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { WorkspaceInvitation } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { copyTextToClipboard } from "@nerve/utils";
// lib
import { invitationLink } from "@/lib/invitation-link";

/** Copies an invitation's link (invitationLink, at this origin) to the clipboard, and says whether it could. */
export function useCopyInvitationLink(): (invitation: Pick<WorkspaceInvitation, "id" | "token">) => Promise<void> {
  const { t } = useTranslation();
  return async (invitation) => {
    try {
      await copyTextToClipboard(invitationLink(window.location.origin, invitation));
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("common.link_copied"),
        message: t("entity.link_copied_to_clipboard", { entity: t("common.invite") }),
      });
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
````

`web/apps/web/core/hooks/use-workspace-invitation.tsx`（整个文件，101 行）：

````whole web/apps/web/core/hooks/use-workspace-invitation.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect } from "react";
import type { Control, FieldArrayWithId, FormState } from "react-hook-form";
import { useFieldArray, useForm } from "react-hook-form";
// nerve imports
import type { InvitationCreate, WorkspaceInvitation, WorkspaceInvitationsCreate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// components
import { invitationRefusal } from "@/components/workspace/invite-modal/refusal";
// lib
import { followInSession } from "@/lib/in-session";

/** A row as the form adds it: an address to type, invited as a member. */
const newRow = (): InvitationCreate => ({ email: "", role: 15 });

/** The form as it opens: one row. */
const opened = (): WorkspaceInvitationsCreate => ({ invitations: [newRow()] });

type TUseWorkspaceInvitationProps = {
  /** Sends the invitations of the form, nerve's WorkspaceInvitationsCreate, and gives the invitations nerve made. */
  invite: (data: WorkspaceInvitationsCreate) => Promise<WorkspaceInvitation[]>;
  /** What the form's place does with the invitations sent: the modal closes, the onboarding shows their links. */
  onSent: (invitations: WorkspaceInvitation[]) => void;
};

type TUseWorkspaceInvitationReturn = {
  control: Control<WorkspaceInvitationsCreate>;
  fields: FieldArrayWithId<WorkspaceInvitationsCreate, "invitations", "id">[];
  formState: FormState<WorkspaceInvitationsCreate>;
  remove: (index: number) => void;
  onFormSubmit: () => void;
  /** Empties the form: one row again. */
  clear: () => void;
  appendField: () => void;
};

/**
 * The invitation form (M3 design 2, W4): its rows, each an address and a role, which it sends together. Sent, the form
 * empties, says so and gives nerve's invitations to onSent; refused, it shows why under each row nerve names, else in a
 * toast (invitationRefusal). It follows the answer only in the session the invitations were sent in (M3 design 7.1).
 */
export const useWorkspaceInvitationActions = (props: TUseWorkspaceInvitationProps): TUseWorkspaceInvitationReturn => {
  const { invite, onSent } = props;
  const { t } = useTranslation();
  // form info
  const { control, reset, handleSubmit, formState, setError } = useForm<WorkspaceInvitationsCreate>({
    defaultValues: opened(),
  });

  const { fields, append, remove } = useFieldArray({
    control,
    name: "invitations",
  });

  const appendField = () => {
    append(newRow());
  };

  const onSubmitForm = (data: WorkspaceInvitationsCreate) =>
    followInSession(() => invite(data), {
      done: (invitations) => {
        reset(opened());
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("toast.success"),
          message: t("workspace_settings.settings.members.invitations_sent_successfully"),
        });
        onSent(invitations);
      },
      failed: (error) => {
        const refusal = invitationRefusal(error, data.invitations.length);
        if (refusal.kind === "toast") {
          setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(refusal.message) });
          return;
        }
        for (const { index, message } of refusal.rows) {
          setError(`invitations.${index}.email`, { type: "server", message: t(message) });
        }
      },
    });

  useEffect(() => {
    if (fields.length === 0) append([newRow()]);
  }, [fields, append]);

  return {
    control,
    fields,
    formState,
    remove,
    onFormSubmit: handleSubmit(onSubmitForm),
    clear: () => reset(opened()),
    appendField,
  };
};
````

`web/apps/web/core/lib/invitation-link.test.ts`（新文件，27 行）：

````file web/apps/web/core/lib/invitation-link.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { invitationLink } from "./invitation-link";

// The link an admin copies for an invitation (M3 design 7.4, 9.5): the invitation page reads its id and its token
// back from the query as they were.

describe("invitationLink", () => {
  it("opens the invitation page at the origin, with the invitation's id and token", () => {
    expect(invitationLink("https://nerve.example", { id: "i-1", token: "nrv_inv_abc-_Z9" })).toBe(
      "https://nerve.example/workspace-invitations?invitation_id=i-1&token=nrv_inv_abc-_Z9"
    );
  });

  it.each(["a+b", "a&token=b", "a b/c=?#d", "雪"])("gives back the token %s as it was", (token) => {
    const link = new URL(invitationLink("https://nerve.example", { id: "i-1", token }));
    expect([link.pathname, link.searchParams.get("invitation_id"), link.searchParams.get("token")]).toEqual([
      "/workspace-invitations",
      "i-1",
      token,
    ]);
  });
});
````

`web/apps/web/core/lib/invitation-link.ts`（新文件，15 行）：

````file web/apps/web/core/lib/invitation-link.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { WorkspaceInvitation } from "@nerve/api-client";

/**
 * The link of an invitation (M3 design 7.4): the invitation page at origin, with the invitation's id and the token
 * nerve gives an admin with it, each a value of the query, encoded as one.
 */
export function invitationLink(origin: string, invitation: Pick<WorkspaceInvitation, "id" | "token">): string {
  const query = new URLSearchParams({ invitation_id: invitation.id, token: invitation.token });
  return `${origin}/workspace-invitations?${query.toString()}`;
}
````

- [ ] **Step 2: 弹窗、成员页、邀请列表、文案和上限**

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx`（修改，4 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
import { SearchOutline } from "@makeplane/propel/icons";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
import { SearchOutline } from "@makeplane/propel/icons";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
import { useUserPermissions } from "@/hooks/store/user";
import type { InvitationFormValues } from "@/hooks/use-workspace-invitation";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
import { useUserPermissions } from "@/hooks/store/user";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
  );

  const handleWorkspaceInvite = async (data: InvitationFormValues) => {
    try {
      await inviteMembersToWorkspace(workspaceSlug, { invitations: data.emails });

      setInviteModal(false);

      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: t("workspace_settings.settings.members.invitations_sent_successfully"),
      });
    } catch (error: unknown) {
      let message = undefined;
      if (error instanceof Error) {
        const err = error as Error & { error?: string };
        message = err.error;
      }
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: `${message ?? t("something_went_wrong_please_try_again")}`,
      });

      throw error;
    }
  };
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
  );
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
        onSubmit={handleWorkspaceInvite}
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx
        invite={(data) => inviteMembersToWorkspace(workspaceSlug, data)}
````

`web/apps/web/core/components/workspace/members/invite-modal.test.tsx`（新文件，101 行）：

````file web/apps/web/core/components/workspace/members/invite-modal.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactNode } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, pick, shown, submitModalForm } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { SendWorkspaceInvitationModal } from "./invite-modal";

// What the invitation form sends, and what it does with nerve's answer (M3 design 2 W4, 7.1): the modal renders on
// the server, with stand-ins for its inputs, selects and buttons (fake-controls.ts), which keep the props they were
// given; the test types an address and picks a role as a person would, and submits the form. Its session is
// fake-tab.ts's.

const page = vi.hoisted(() => ({ invite: vi.fn(), onClose: vi.fn() }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@makeplane/propel/components/input", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
// The form's title is headless UI's Dialog.Title, which renders only inside its Dialog: the modal's stand-in has none.
vi.mock("@headlessui/react", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@headlessui/react")>()),
  Dialog: { Title: ({ children }: { children?: ReactNode }) => children },
}));

/** Renders the form, types email in its one row, picks the role labelled role, and submits it. */
async function invite(email: string, role: string) {
  renderToStaticMarkup(<SendWorkspaceInvitationModal isOpen onClose={page.onClose} invite={page.invite} />);
  const [address] = shown.inputs;
  const [roles] = shown.selects;
  if (!address || !roles) throw new Error("the form showed no row");
  address.onChange({ target: { value: email } });
  pick(roles, role);
  await submitModalForm();
}

beforeEach(() => {
  signedIn();
  page.invite.mockReset();
  page.invite.mockResolvedValue([]);
  page.onClose.mockClear();
  toasts.length = 0;
  emptyShown();
});

describe("SendWorkspaceInvitationModal", () => {
  it("sends its rows as nerve's WorkspaceInvitationsCreate, each role a number; then closes, and says so", async () => {
    await invite("dave@example.com", "Guest");
    expect(page.invite.mock.calls).toEqual([[{ invitations: [{ email: "dave@example.com", role: 5 }] }]]);
    expect(page.onClose).toHaveBeenCalledTimes(1);
    expect(toasts).toEqual([
      {
        type: "success",
        title: "toast.success",
        message: "workspace_settings.settings.members.invitations_sent_successfully",
      },
    ]);
  });

  it("sends nothing for an address that is none", async () => {
    await invite("dave", "Member");
    expect([page.invite.mock.calls, page.onClose.mock.calls, toasts]).toEqual([[], [], []]);
  });

  // the message the form then shows under the row is W4's to check: the render does not show what is set after it
  it("stays open, with no toast, when nerve refuses a row", async () => {
    page.invite.mockRejectedValueOnce(
      refusal(422, "validation_failed", [{ field: "invitations[0].email", code: "duplicate" }])
    );
    await invite("dave@example.com", "Member");
    expect([page.onClose.mock.calls, toasts]).toEqual([[], []]);
  });

  it("stays open, and shows nerve's reason, when it refuses the invitations", async () => {
    page.invite.mockRejectedValueOnce(refusal(403, "forbidden"));
    await invite("dave@example.com", "Member");
    expect(page.onClose.mock.calls).toEqual([]);
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.forbidden" }]);
  });

  it.each(lateSettlings)(
    "neither closes nor speaks when the invitations' answer $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const sent = heldChange<undefined>();
      page.invite.mockReturnValueOnce(sent.sent);
      const submitted = invite("dave@example.com", "Member");
      await vi.waitFor(() => expect(page.invite).toHaveBeenCalledTimes(1));
      switchAccount();
      settle(sent);
      await submitted;
      expect([page.onClose.mock.calls, toasts]).toEqual([[], []]);
    }
  );
});
````

`web/apps/web/core/components/workspace/members/invite-modal.tsx`（修改，5 处）：

````old web/apps/web/core/components/workspace/members/invite-modal.tsx
// nerve imports
````
````new web/apps/web/core/components/workspace/members/invite-modal.tsx
// nerve imports
import type { WorkspaceInvitation, WorkspaceInvitationsCreate } from "@nerve/api-client";
````
````old web/apps/web/core/components/workspace/members/invite-modal.tsx
// hooks
import type { InvitationFormValues } from "@/hooks/use-workspace-invitation";
````
````new web/apps/web/core/components/workspace/members/invite-modal.tsx
// hooks
````
````old web/apps/web/core/components/workspace/members/invite-modal.tsx
  onSubmit: (data: InvitationFormValues) => Promise<void> | undefined;
````
````new web/apps/web/core/components/workspace/members/invite-modal.tsx
  /** Sends the invitations of the form (the workspace's store, for the page's workspace). */
  invite: (data: WorkspaceInvitationsCreate) => Promise<WorkspaceInvitation[]>;
````
````old web/apps/web/core/components/workspace/members/invite-modal.tsx
  const { isOpen, onClose, onSubmit } = props;
````
````new web/apps/web/core/components/workspace/members/invite-modal.tsx
  const { isOpen, onClose, invite } = props;
````
````old web/apps/web/core/components/workspace/members/invite-modal.tsx
  const { control, fields, formState, remove, onFormSubmit, handleClose, appendField } = useWorkspaceInvitationActions({
    onSubmit,
    onClose,
  });
````
````new web/apps/web/core/components/workspace/members/invite-modal.tsx
  const { control, fields, formState, remove, onFormSubmit, clear, appendField } = useWorkspaceInvitationActions({
    invite,
    onSent: onClose,
  });

  // the form empties once the modal has gone (its leave transition)
  const handleClose = () => {
    onClose();
    const timeout = setTimeout(() => {
      clear();
      clearTimeout(timeout);
    }, 350);
  };
````

`web/apps/web/core/components/workspace/settings/invitations-list-item.tsx`（整个文件，159 行）：

````whole web/apps/web/core/components/workspace/settings/invitations-list-item.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
// nerve imports
import type { WorkspaceRole } from "@nerve/api-client";
import { ROLE } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { ChevronDownOutline, DeleteOutline, LinkOutline } from "@makeplane/propel/icons";
import type { TContextMenuItem } from "@nerve/ui";
import { CustomSelect, CustomMenu } from "@nerve/ui";
import { cn } from "@nerve/utils";
// components
import { ConfirmWorkspaceMemberRemove } from "@/components/workspace/confirm-workspace-member-remove";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useCopyInvitationLink } from "@/hooks/use-copy-invitation-link";
// local imports
import { WORKSPACE_ROLES } from "../workspace-roles";
import { useInvitationChanges } from "./use-invitation-changes";

type Props = {
  invitationId: string;
};

export const WorkspaceInvitationsListItem = observer(function WorkspaceInvitationsListItem(props: Props) {
  const { invitationId } = props;
  // router
  const { workspaceSlug } = useParams();
  // states
  const [removeMemberModal, setRemoveMemberModal] = useState(false);
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const {
    workspace: { getWorkspaceInvitationDetails },
  } = useMember();
  const { changeRole, remove } = useInvitationChanges();
  const copyLink = useCopyInvitationLink();
  // derived values: the row shows only to an admin (the members page's gate, decision 4), who may change, copy and
  // delete any pending invitation, and delete a declined one, which nerve keeps from being changed or accepted
  const invitationDetails = getWorkspaceInvitationDetails(invitationId);

  if (!workspaceSlug || !invitationDetails) return null;

  const declined = invitationDetails.responded_at !== null;

  const MENU_ITEMS: TContextMenuItem[] = [
    {
      key: "copy-link",
      action: () => void copyLink(invitationDetails),
      title: t("common.actions.copy_link"),
      icon: LinkOutline,
      shouldRender: !declined,
    },
    {
      key: "remove",
      action: () => {
        setRemoveMemberModal(true);
      },
      title: t("common.remove"),
      icon: DeleteOutline,
      className: "text-danger-primary",
      iconClassName: "text-danger-primary",
    },
  ];

  return (
    <>
      <ConfirmWorkspaceMemberRemove
        isOpen={removeMemberModal}
        onClose={() => setRemoveMemberModal(false)}
        userDetails={{
          id: invitationDetails.id,
          display_name: `${invitationDetails.email}`,
        }}
        onSubmit={() => remove(workspaceSlug, invitationDetails.id)}
      />
      <div className="group flex h-full w-full items-center justify-between px-3 py-4 hover:bg-layer-transparent-hover">
        <div className="flex items-center gap-x-4 gap-y-2">
          <span className="relative flex h-10 w-10 items-center justify-center rounded-sm bg-layer-3 p-4 text-tertiary capitalize">
            {(invitationDetails.email ?? "?")[0]}
          </span>
          <div>
            <h4 className="cursor-default text-body-xs-regular">{invitationDetails.email}</h4>
          </div>
        </div>
        <div className="flex items-center gap-2 text-11">
          <div className="flex items-center justify-center rounded-sm bg-label-yellow-bg-strong/20 px-2.5 py-1 text-center text-caption-sm-medium text-label-yellow-text">
            <p>{declined ? t("workspace_settings.settings.members.declined") : t("common.pending")}</p>
          </div>
          {declined ? (
            <span className="px-2 py-0.5 text-caption-sm-medium">{ROLE[invitationDetails.role]}</span>
          ) : (
            <CustomSelect
              customButton={
                <div className="item-center flex gap-1 rounded-sm px-2 py-0.5">
                  <span className="flex items-center rounded-sm text-caption-sm-medium">
                    {ROLE[invitationDetails.role]}
                  </span>
                  <span className="grid place-items-center">
                    <ChevronDownOutline className="h-3 w-3" />
                  </span>
                </div>
              }
              value={invitationDetails.role}
              // the select gives the picked option's value: a role's number, of WORKSPACE_ROLES
              onChange={(role: WorkspaceRole) => void changeRole(workspaceSlug, invitationDetails.id, role)}
              placement="bottom-end"
            >
              {WORKSPACE_ROLES.map((role) => (
                <CustomSelect.Option key={role} value={role}>
                  {ROLE[role]}
                </CustomSelect.Option>
              ))}
            </CustomSelect>
          )}
          <CustomMenu ellipsis placement="bottom-end" closeOnSelect>
            {MENU_ITEMS.filter((item) => item.shouldRender !== false).map((item) => (
              <CustomMenu.MenuItem
                key={item.key}
                onClick={() => {
                  item.action();
                }}
                className={cn(
                  "flex items-center gap-2",
                  {
                    "text-placeholder": item.disabled,
                  },
                  item.className
                )}
                disabled={item.disabled}
              >
                {item.icon && <item.icon className={cn("h-3 w-3", item.iconClassName)} />}
                <div>
                  <h5>{item.title}</h5>
                  {item.description && (
                    <p
                      className={cn("whitespace-pre-line text-tertiary", {
                        "text-placeholder": item.disabled,
                      })}
                    >
                      {item.description}
                    </p>
                  )}
                </div>
              </CustomMenu.MenuItem>
            ))}
          </CustomMenu>
        </div>
      </div>
    </>
  );
});
````

`web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts`（新文件，84 行）：

````file web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { useInvitationChanges } from "./use-invitation-changes";

// What the members page sends for each change of an invitation, and what it does with nerve's answer (M3 design 7.1,
// 7.5): the hook runs as a plain function, with stand-ins for the store's changes, which nerve answers when the test
// says. Its session is fake-tab.ts's.

const page = vi.hoisted(() => ({ updateMemberInvitation: vi.fn(), deleteMemberInvitation: vi.fn() }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    workspace: {
      updateMemberInvitation: page.updateMemberInvitation,
      deleteMemberInvitation: page.deleteMemberInvitation,
    },
  }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Each change the page makes, by its store change: the role's and the deletion's. */
const changes = [
  {
    change: "a role",
    store: page.updateMemberInvitation,
    make: () => useInvitationChanges().changeRole("acme", "i-dan", 5),
  },
  {
    change: "a deletion",
    store: page.deleteMemberInvitation,
    make: () => useInvitationChanges().remove("acme", "i-dan"),
  },
];

beforeEach(() => {
  signedIn();
  for (const store of [page.updateMemberInvitation, page.deleteMemberInvitation]) {
    store.mockReset();
    store.mockResolvedValue(undefined);
  }
  toasts.length = 0;
});

describe("useInvitationChanges", () => {
  it("sends a pending invitation's new role as its number, to the invitation's workspace", async () => {
    await useInvitationChanges().changeRole("acme", "i-dan", 5);
    expect([page.updateMemberInvitation.mock.calls, toasts]).toEqual([[["acme", "i-dan", { role: 5 }]], []]);
  });

  it("deletes an invitation of the workspace, and says so", async () => {
    await useInvitationChanges().remove("acme", "i-dan");
    expect(page.deleteMemberInvitation.mock.calls).toEqual([["acme", "i-dan"]]);
    expect(toasts).toEqual([{ type: "success", title: "Success!", message: "Invitation removed successfully." }]);
  });

  it.each(changes)("shows nerve's reason when it refuses $change", async ({ store, make }) => {
    store.mockRejectedValueOnce(refusal(409, "workspace.invitation_responded"));
    await make();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.workspace_invitation_responded" }]);
  });

  describe.each(changes)("$change", ({ store, make }) => {
    it.each(lateSettlings)(
      "does nothing on the page when it $settles after another tab moved this one to another account",
      async ({ settle }) => {
        const change = heldChange<undefined>();
        store.mockReturnValueOnce(change.sent);
        const made = make();
        switchAccount();
        settle(change);
        await made;
        expect(toasts).toEqual([]);
      }
    );
  });
});
````

`web/apps/web/core/components/workspace/settings/use-invitation-changes.ts`（新文件，42 行）：

````file web/apps/web/core/components/workspace/settings/use-invitation-changes.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { WorkspaceRole } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useMember } from "@/hooks/store/use-member";
// lib
import { errorMessageKey } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";

/**
 * The changes the members page makes to a workspace's invitations (M3 design 7.5), each to the invitation of
 * invitationId in the workspace of workspaceSlug, as the store's changes take them: a pending invitation's role, and
 * the deletion of an invitation, pending or declined. Each shows nerve's refusal as its reason; the page follows each
 * only in the session it was sent in (M3 design 7.1): once another tab has moved this one to another account, the
 * page is that account's, and says nothing of the change.
 */
export function useInvitationChanges() {
  const {
    workspace: { updateMemberInvitation, deleteMemberInvitation },
  } = useMember();
  const { t } = useTranslation();
  const failed = (error: unknown) =>
    setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) });

  return {
    /** Gives the pending invitation the role: its number, as nerve's invitation update takes it. */
    changeRole: (workspaceSlug: string, invitationId: string, role: WorkspaceRole) =>
      followInSession(() => updateMemberInvitation(workspaceSlug, invitationId, { role }), { failed }),
    /** Deletes the invitation, and says so. */
    remove: (workspaceSlug: string, invitationId: string) =>
      followInSession(() => deleteMemberInvitation(workspaceSlug, invitationId), {
        done: () =>
          setToast({ type: TOAST_TYPE.SUCCESS, title: "Success!", message: "Invitation removed successfully." }),
        failed,
      }),
  };
}
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 360",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 359",
````

`web/packages/i18n/src/locales/en/workspace-settings.json`（修改，2 处）：

````old web/packages/i18n/src/locales/en/workspace-settings.json
        "pending_invites": "Pending invites",
````
````new web/packages/i18n/src/locales/en/workspace-settings.json
        "pending_invites": "Pending invites",
        "declined": "Declined",
````
````old web/packages/i18n/src/locales/en/workspace-settings.json
            "invalid": "Email is invalid"
````
````new web/packages/i18n/src/locales/en/workspace-settings.json
            "invalid": "Email is invalid",
            "already_member": "Already a member of this workspace.",
            "already_invited": "This address is invited already (delete that invitation first), or given twice here."
````

`web/packages/i18n/src/locales/zh-CN/workspace-settings.json`（修改，2 处）：

````old web/packages/i18n/src/locales/zh-CN/workspace-settings.json
        "pending_invites": "待处理邀请",
````
````new web/packages/i18n/src/locales/zh-CN/workspace-settings.json
        "pending_invites": "待处理邀请",
        "declined": "已忽略",
````
````old web/packages/i18n/src/locales/zh-CN/workspace-settings.json
            "invalid": "电子邮件无效"
````
````new web/packages/i18n/src/locales/zh-CN/workspace-settings.json
            "invalid": "电子邮件无效",
            "already_member": "已是这个工作区的成员。",
            "already_invited": "这个邮箱已有一份邀请（先删除它），或在这里填了两次。"
````

- [ ] **Step 3: 静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/components/workspace/workspace-roles.ts"
````
````new .oxlintrc.json
        "web/apps/web/core/components/workspace/workspace-roles.ts",
        "web/apps/web/core/lib/invitation-link.ts",
        "web/apps/web/core/lib/invitation-link.test.ts",
        "web/apps/web/core/components/workspace/invite-modal/refusal.ts",
        "web/apps/web/core/components/workspace/invite-modal/refusal.test.ts",
        "web/apps/web/core/components/workspace/members/invite-modal.test.tsx",
        "web/apps/web/core/hooks/use-copy-invitation-link.ts"
````

- [ ] **Step 4: 端到端**

`e2e/fixtures/api.ts`（修改，1 处）：

````old e2e/fixtures/api.ts
    throw new Error(`accept answered 200 without the workspace`);
  }
  return data;
}
````
````new e2e/fixtures/api.ts
    throw new Error(`accept answered 200 without the workspace`);
  }
  return data;
}

/** Declines invitation with the bearer token given, the invitee's. */
export async function decline(api: Api, token: string, invitation: WorkspaceInvitation): Promise<void> {
  const { error, response } = await api.POST("/api/v0/workspace-invitations/{invitation_id}/decline", {
    params: { path: { invitation_id: invitation.id } },
    body: { token: invitation.token },
    headers: bearer(token),
  });
  expect(response.status, `decline the invitation of ${invitation.email}: ${JSON.stringify(error)}`).toBe(204);
}

/**
 * The invitation of email to the workspace of slug, pending or declined, with its link's token, as its admin of token
 * lists it.
 */
export async function invitationTo(api: Api, token: string, slug: string, email: string): Promise<WorkspaceInvitation> {
  const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/invitations", {
    params: { path: { slug } },
    headers: bearer(token),
  });
  expect(response.status, `list the invitations of ${slug}: ${JSON.stringify(error)}`).toBe(200);
  const invitation = data?.data.find((i) => i.email === email);
  if (!invitation) {
    throw new Error(`no invitation of ${email} to ${slug}`);
  }
  return invitation;
}
````

`e2e/fixtures/assert/workspace.ts`（修改，3 处）：

````old e2e/fixtures/assert/workspace.ts
 * W4, W5, W6: the invitations of the workspace of slug, deleted ones too, are want, in any order. The table
````
````new e2e/fixtures/assert/workspace.ts
 * W4, W5, W6: the invitations of the workspace of slug, deleted ones too, are want, in any order of their addresses
 * (of two to one address, the older first). The table
````
````old e2e/fixtures/assert/workspace.ts
      WHERE w.slug = $1 ORDER BY i.email COLLATE "C"`,
````
````new e2e/fixtures/assert/workspace.ts
      WHERE w.slug = $1 ORDER BY i.email COLLATE "C", i.created_at`,
````
````old e2e/fixtures/assert/workspace.ts
  ).toEqual(want.toSorted((a, b) => (a.email < b.email ? -1 : 1)));
````
````new e2e/fixtures/assert/workspace.ts
  ).toEqual(want.toSorted((a, b) => (a.email < b.email ? -1 : a.email > b.email ? 1 : 0)));
````

`e2e/fixtures/workspace-pages.ts`（修改，2 处）：

````old e2e/fixtures/workspace-pages.ts
import { expect, type Browser, type Locator, type Page, type Response } from "@playwright/test";

````
````new e2e/fixtures/workspace-pages.ts
import { expect, type Browser, type Locator, type Page, type Response } from "@playwright/test";

import type { WorkspaceInvitation } from "./api";
````
````old e2e/fixtures/workspace-pages.ts
    page.getByRole("dialog").getByRole("button", { name: entry, exact: true }).click()
  );
}

````
````new e2e/fixtures/workspace-pages.ts
    page.getByRole("dialog").getByRole("button", { name: entry, exact: true }).click()
  );
}

/** The path of an invitation's link (M3 design 7.4): the invitation page, with its id and its token in the query. */
export function invitationLinkOf(invitation: Pick<WorkspaceInvitation, "id" | "token">): string {
  return `/workspace-invitations?invitation_id=${invitation.id}&token=${invitation.token}`;
}

/** The row of the members page's invitations, which page shows, of the invitation to email. */
export function invitationRow(page: Page, email: string): Locator {
  // the innermost element that holds the address and the invitation's state: rows come after the lists that hold them
  return page
    .locator("div")
    .filter({ has: page.getByRole("heading", { name: email, exact: true }) })
    .filter({ hasText: /Pending|Declined/ })
    .last();
}

/**
 * Sends invitations from the members page, which page shows to an admin of the workspace of slug: opens the form, types
 * each address in a row of its own and picks its role by its label, and sends them. Resolves with what the page sent,
 * and nerve's answer.
 */
export async function sendInvitations(
  page: Page,
  slug: string,
  rows: { email: string; role: string }[]
): Promise<{ body: unknown; answer: Response }> {
  await page.getByRole("button", { name: "Add member" }).click();
  const form = page.getByRole("dialog");
  // the rows one after another: the form adds a row once the one before it is typed
  await rows.reduce(async (before, { email, role }, index) => {
    await before;
    if (index > 0) await form.getByRole("button", { name: "Add more" }).click();
    await form.locator(`[id="invitations.${index}.email"]`).fill(email);
    // a row's role select shows the role it holds, Member as the row is added
    if (role !== "Member") {
      await form.getByRole("button", { name: "Member", exact: true }).nth(index).click();
      await page.getByRole("option", { name: role, exact: true }).click();
    }
  }, Promise.resolve());
  return sentTo(page, "POST", `/api/v0/workspaces/${slug}/invitations`, () =>
    form.getByRole("button", { name: "Send invitations" }).click()
  );
}

/**
 * Deletes the invitation to email from the members page, which page shows: its row's menu, Remove, and the dialog that
 * asks. Resolves with nerve's answer to the request of method to path that the page sends.
 */
export async function removeInvitation(
  page: Page,
  email: string,
  request: { method: string; path: string }
): Promise<Response> {
  // the row's menu is its last button
  await invitationRow(page, email).getByRole("button").last().click();
  await page.getByText("Remove", { exact: true }).click();
  return answerTo(page, request.method, request.path, () =>
    page.getByRole("dialog").getByRole("button", { name: "Remove", exact: true }).click()
  );
}

````

`e2e/stories/workspace/w4-invite-members.spec.ts`（修改，4 处）：

````old e2e/stories/workspace/w4-invite-members.spec.ts
import { createWorkspace, invite, inviteAndAccept, slugFor, type InvitationCreate } from "../../fixtures/api";
````
````new e2e/stories/workspace/w4-invite-members.spec.ts
import {
  createWorkspace,
  decline,
  invitationTo,
  invite,
  inviteAndAccept,
  slugFor,
  type InvitationCreate,
} from "../../fixtures/api";
````
````old e2e/stories/workspace/w4-invite-members.spec.ts
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
````
````new e2e/stories/workspace/w4-invite-members.spec.ts
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { registerOnboarded, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import {
  anotherBrowser,
  invitationLinkOf,
  invitationRow,
  memberRow,
  removeInvitation,
  sendInvitations,
} from "../../fixtures/workspace-pages";
````
````old e2e/stories/workspace/w4-invite-members.spec.ts
// W4, invite members (M3 design 2, 3.8; decision 4: admins only). The page
// version comes with the members page (P9).
````
````new e2e/stories/workspace/w4-invite-members.spec.ts
// W4, invite members (M3 design 2, 3.8; decision 4: admins only).
````
````old e2e/stories/workspace/w4-invite-members.spec.ts
  );
});

````
````new e2e/stories/workspace/w4-invite-members.spec.ts
  );
});

test("W4 (page): the admin invites a member and a guest, changes a role and copies a link; an invitation deleted meanwhile is refused with the reason; an active member's address and a declined invitation's are refused under their rows; the declined one deleted, its address is invited again; a member sees no invitation and asks for none", async ({
  api,
  baseURL,
  browser,
  context,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const memberEmail = emailFor(testInfo, "member");
  const member = await registerOnboarded(api, memberEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  await inviteAndAccept(api, admin.access_token, slug, { email: memberEmail, token: member.access_token }, 15);
  // erin declined an invitation to Acme.
  const erinEmail = emailFor(testInfo, "erin");
  await invite(api, admin.access_token, slug, [{ email: erinEmail, role: 15 }]);
  const toErin = await invitationTo(api, admin.access_token, slug, erinEmail);
  await decline(api, (await registerOnboarded(api, erinEmail)).access_token, toErin);
  const [dave, frank] = [emailFor(testInfo, "dave"), emailFor(testInfo, "frank")];
  const invitations = `/api/v0/workspaces/${slug}/invitations`;

  // The member's page shows no invitation, and asks nerve for none.
  const theMember = await anotherBrowser(browser, baseURL ?? "", member);
  const memberWatch = await watchPage(theMember.page);
  await theMember.page.goto(`/${slug}/settings/members`);
  await expect(memberRow(theMember.page, adminEmail)).toBeVisible();
  await expect(theMember.page.getByText("Pending invites")).toHaveCount(0);
  await expect(theMember.page.getByRole("button", { name: "Add member" })).toHaveCount(0);
  expect(memberWatch.apiRequests.filter((request) => request.endsWith(invitations))).toEqual([]);
  expect([memberWatch.apiFailures, memberWatch.oldApiRequests, memberWatch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(theMember.page, memberWatch, { warnings: [EMOJI_CHECK_WARNING] });
  await theMember.close();

  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/settings/members`);
  // erin's invitation says it was declined, and offers neither a role nor a link.
  await expect(invitationRow(page, erinEmail)).toContainText("Declined");
  await expect(invitationRow(page, erinEmail).getByRole("button", { name: "Member" })).toHaveCount(0);

  // dave as a member, frank as a guest: one request, nerve's WorkspaceInvitationsCreate, each role a number.
  const sent = await sendInvitations(page, slug, [
    { email: dave, role: "Member" },
    { email: frank, role: "Guest" },
  ]);
  expect([sent.answer.status(), sent.body]).toEqual([
    201,
    {
      invitations: [
        { email: dave, role: 15 },
        { email: frank, role: 5 },
      ],
    },
  ]);
  await expect(page.getByText("Invitations sent successfully")).toBeVisible();
  const davesInvitation = await invitationTo(api, admin.access_token, slug, dave);
  const franksInvitation = await invitationTo(api, admin.access_token, slug, frank);
  await expect(invitationRow(page, dave)).toContainText("Pending");

  // frank becomes a member; dave's link goes to the clipboard.
  await invitationRow(page, frank).getByRole("button", { name: "Guest" }).click();
  const changed = await sentTo(page, "PATCH", `/api/v0/workspace-invitations/${franksInvitation.id}`, () =>
    page.getByRole("option", { name: "Member", exact: true }).click()
  );
  expect([changed.answer.status(), changed.body]).toEqual([200, { role: 15 }]);
  await invitationRow(page, dave).getByRole("button").last().click();
  await page.getByText("Copy link", { exact: true }).click();
  await expect(page.getByText("Invite link copied to clipboard")).toBeVisible();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(
    `${baseURL}${invitationLinkOf(davesInvitation)}`
  );
  // frank's invitation, deleted meanwhile through the API: the page's removal is refused, and says why.
  const removal = { method: "DELETE", path: `/api/v0/workspace-invitations/${franksInvitation.id}` };
  const meanwhile = await api.DELETE("/api/v0/workspace-invitations/{invitation_id}", {
    params: { path: { invitation_id: franksInvitation.id } },
    headers: bearer(admin.access_token),
  });
  expect(meanwhile.response.status).toBe(204);
  expect((await removeInvitation(page, frank, removal)).status()).toBe(404);
  await expect(page.getByText("The invitation does not exist, or its link is not valid.")).toBeVisible();

  // The member's address and erin's: refused, each under its row, and the form stays open.
  const refused = await sendInvitations(page, slug, [
    { email: memberEmail, role: "Member" },
    { email: erinEmail, role: "Member" },
  ]);
  expect(refused.answer.status()).toBe(422);
  const form = page.getByRole("dialog");
  await expect(form.getByText("Already a member of this workspace.")).toBeVisible();
  await expect(
    form.getByText("This address is invited already (delete that invitation first), or given twice here.")
  ).toBeVisible();
  await form.getByRole("button", { name: "Cancel" }).click();
  // erin's declined invitation deleted, her address is invited again.
  const erins = { method: "DELETE", path: `/api/v0/workspace-invitations/${toErin.id}` };
  expect((await removeInvitation(page, erinEmail, erins)).status()).toBe(204);
  const again = await sendInvitations(page, slug, [{ email: erinEmail, role: "Guest" }]);
  expect([again.answer.status(), again.body]).toEqual([201, { invitations: [{ email: erinEmail, role: 5 }] }]);
  await expect(invitationRow(page, erinEmail)).toContainText("Pending");

  const accepted = { email: memberEmail, role: 15, accepted: true, responded: true, deleted: true };
  await expectInvitations(db, slug, adminEmail, [
    accepted,
    { email: dave, role: 15, accepted: false, responded: false, deleted: false },
    { email: erinEmail, role: 15, accepted: false, responded: true, deleted: true },
    { email: erinEmail, role: 5, accepted: false, responded: false, deleted: false },
    { email: frank, role: 15, accepted: false, responded: false, deleted: true },
  ]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`404 ${removal.method} ${removal.path}`, `422 POST ${invitations}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING],
    errors: [
      "Failed to load resource: the server responded with a status of 404 (Not Found)",
      "Failed to load resource: the server responded with a status of 422 (Unprocessable Entity)",
    ],
  });
});

````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、3 个例外，没有命中；web 的 oxlint 359 条，等于新的上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 80 个全部通过。

- [ ] **Step 6: 提交**

```bash
git add .oxlintrc.json e2e/fixtures/api.ts e2e/fixtures/assert/workspace.ts e2e/fixtures/workspace-pages.ts e2e/stories/workspace/w4-invite-members.spec.ts 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/members/page.tsx' web/apps/web/core/components/workspace/invite-modal/fields.tsx web/apps/web/core/components/workspace/invite-modal/refusal.test.ts web/apps/web/core/components/workspace/invite-modal/refusal.ts web/apps/web/core/components/workspace/members/invite-modal.test.tsx web/apps/web/core/components/workspace/members/invite-modal.tsx web/apps/web/core/components/workspace/settings/invitations-list-item.tsx web/apps/web/core/components/workspace/settings/use-invitation-changes.test.ts web/apps/web/core/components/workspace/settings/use-invitation-changes.ts web/apps/web/core/hooks/use-copy-invitation-link.ts web/apps/web/core/hooks/use-workspace-invitation.tsx web/apps/web/core/lib/invitation-link.test.ts web/apps/web/core/lib/invitation-link.ts web/apps/web/package.json web/packages/i18n/src/locales/en/workspace-settings.json web/packages/i18n/src/locales/zh-CN/workspace-settings.json
```
```bash
git commit -m "feat(M3/P9): the invitation form sends nerve's WorkspaceInvitationsCreate and shows its refusals under the rows

The form's values are the generated WorkspaceInvitationsCreate, each role
a number; a refusal that names rows of the form is shown under them (an
active member's address, one invited already), any other as nerve's
reason; the answer is followed only in the session it was sent in. The
invitation list labels a declined invitation, offers only its removal,
makes its changes through useInvitationChanges, which follows each only
in its session and says why nerve refused it, and says when the
clipboard was not written; the link is built in one place, its query
encoded. The new modules join the non-null override. W4's page version.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p9.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T3.1` | 邀请表单的角色选择给出字符串形式的角色编号 | `invite-modal.test.tsx`、故事 W4 | vitest；端到端 |
| `T3.2` | 邀请表单丢掉 nerve 的拒绝 | `invite-modal.test.tsx`、故事 W4 | vitest；端到端 |
| `T3.3` | 邀请表单不论会话都跟进回答 | `invite-modal.test.tsx` | vitest |
| `T3.4` | 邀请列表的修改（`useInvitationChanges`）丢掉 nerve 的拒绝 | `use-invitation-changes.test.ts`、故事 W4 | vitest；端到端 |
| `T3.5` | 已忽略的邀请显示为待接受，带角色和链接 | 故事 W4 | 端到端 |
| `T3.6` | 一行的拒绝显示字段的通用文案 | `refusal.test.ts`、故事 W4 | vitest；端到端 |
| `T3.7` | 邀请链接不编码其中的值 | `invitation-link.test.ts` | vitest |
| `T3.8` | 拒绝指名的字段不属于任何一行时，显示在它指名的行下，其余不说 | `refusal.test.ts` | vitest |
| `T3.9` | 邀请的改角色不论会话都跟进（`try/catch`，预检的 PF3c） | `use-invitation-changes.test.ts` | vitest |
| `T3.10` | 删除邀请不论会话都跟进（`try/catch`，预检的 PF3c） | `use-invitation-changes.test.ts` | vitest |
| `T3.11` | 邀请链接（`invitation-link.ts`，范围内）用非空断言 | oxlint（`check:lint`） | 静态 |
| `T3.12` | 邀请弹窗发出邀请之后不关闭 | `invite-modal.test.tsx`、故事 W4 | vitest；端到端 |

---

### Task 4: 邀请页；打开刚加入的工作区；W5 的页面版本

**Files:**
- Create: `web/apps/web/core/components/workspace/invitation-view.test.ts`、`web/apps/web/core/components/workspace/invitation-view.ts`、`web/apps/web/core/components/workspace/use-invitation-answer.test.ts`、`web/apps/web/core/components/workspace/use-invitation-answer.ts`、`web/apps/web/core/hooks/use-open-workspace.test.ts`、`web/apps/web/core/hooks/use-open-workspace.ts`
- Modify: `.oxlintrc.json`、`e2e/stories/workspace/w5-invitation-link.spec.ts`、`web/apps/web/app/(all)/workspace-invitations/page.tsx`、`web/apps/web/core/lib/invitation-link.test.ts`、`web/apps/web/core/lib/invitation-link.ts`、`web/packages/i18n/src/locales/en/workspace.json`、`web/packages/i18n/src/locales/zh-CN/workspace.json`

**Interfaces:**
- Produces（spec 2.4；M3 设计 2 的 W5，3.8，3.14，7.1，7.4；决定 1；P8a 的 P12、P21）：
  - `core/components/workspace/invitation-view.ts`：`invitationView({ link, preview, signedIn, mismatched }): InvitationView`，纯函数，返回 `kind` 的联合：链接缺 id 或令牌、nerve 按链接找不到邀请（400：id 不是 id；404）是 `invalid`；nerve 没有回答是 `unavailable`（可以重试）；预览还在路上是 `loading`；然后按预览：已忽略是 `declined`，没有登录是 `sign-in`，nerve 对调用者的回答说邀请是别的邮箱的是 `mismatch`，其余是 `answer`。后四种带邀请和链接的令牌。
  - `core/hooks/use-open-workspace.ts`：`useOpenWorkspace()` 给出 `(workspace: Pick<Workspace, "id" | "slug">) => Promise<void>`：把工作区写成调用者上次打开的（`updateUserProfile({ last_workspace_id })`，尽力而为：nerve 不保存时照样打开），再去 `/{slug}`；经 `followInSession`，换账户之后不跳转。接受邀请之后用它（Task 5 的创建也用）。
  - `core/components/workspace/use-invitation-answer.ts`：`useInvitationAnswer({ reread, mismatched })` 给出 `{ accept(invitationId, token), decline(invitationId, token) }`，都经 `followInSession`：接受之后 `useOpenWorkspace`，忽略之后 `reread()`（重读预览，它说已忽略）；nerve 拒绝时：邮箱不一致（`workspace.invitation_email_mismatch`）调 `mismatched()`，其余按 `code` 提示并 `reread()`（邀请可能已被删除或回答）。页面的跟进由它自己的 vitest 守着（预检 M1）。
  - `core/lib/invitation-link.ts` 加 `invitationAuthPath("/" | "/sign-up", invitation)`：从链接去登录或注册的地址，带 `invitation_id`、`token`（页头据此显示要加入的工作区，Task 7 的注册据此带上邀请）和 `next_path`（登录之后回到链接，M2 设计 3.18）。
  - 邀请页（`app/(all)/workspace-invitations/page.tsx`，整个文件）：按 `invitationView` 显示；文案都经 `t()`（`workspace_invitation.*`，en、zh-CN 的 `workspace.json`），标题写工作区和角色，不写被邀请的邮箱（预览不给它，页面也不显示）；接受、忽略经 `useInvitationAnswer`：`reread` 重读预览，`mismatched` 转到 `mismatch`，只给"退出登录"。原来接受之后直接跳转，不写上次打开的工作区；两种失败都只有 `console.error`；没有登录的人也看到"接受"。
- 静态检查（总体设计 7.7）：`.oxlintrc.json` 的非空断言的范围加上 `invitation-view.ts`、`use-invitation-answer.ts`、`use-open-workspace.ts` 和它们的测试（邀请页在 P8a 起已在 `no-restricted-imports` 的范围内）。
- e2e：W5 的两个页面版本用 Task 3 的 `invitationLinkOf`、`invitationTo`、`decline`。

**Tests:**
- vitest：`invitation-view.test.ts`：`invitationView` 的 `shows $view.kind when $when`（`it.each` 十行：链接没有令牌、没有 id，nerve 找不到（404）、id 不是 id（400），nerve 没有回答，还在路上，已忽略，没有登录，邮箱不一致，可以回答）。`use-open-workspace.test.ts`：`writes the workspace as the one opened last, then goes there`；`goes there when nerve does not save it`；`stays when the write $settles after another tab moved this one to another account`（`lateSettlings`）。`invitation-link.test.ts` 加 `opens %s for the link's invitation, and comes back to the link`（`/`、`/sign-up`）。`use-invitation-answer.test.ts`：`accepts the invitation with the link's token, then opens the workspace nerve answers`；`declines the invitation with the link's token, then reads it again`；`says the invitation is another address's when nerve refuses $answer so`；`shows nerve's reason when it refuses $answer, and reads the invitation again`（接受、忽略各一行）；`does nothing on the page when it $settles after another tab moved this one to another account`（接受、忽略各两行）。
- 端到端：`W5 (page): signed out, the link shows the workspace and the role and sends the invitee to sign in and back; she accepts and lands in the workspace, written as the one she opened last; …`（另一条邀请忽略之后页面说已忽略；链接打开时被删除的邀请，接受被拒绝，页面说明原因，之后链接无效）；`W5 (page): another address's invitation, once nerve refuses the answer, says it was sent to another address, offers no answer but signing out, and changes nothing; a link with a changed token, or none, is not valid`。

- [ ] **Step 1: 邀请页的决定、打开工作区和登录的地址**

`web/apps/web/core/components/workspace/invitation-view.test.ts`（新文件，77 行）：

````file web/apps/web/core/components/workspace/invitation-view.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import type { InvitationPreview } from "@nerve/api-client";
import { refusal } from "@/lib/fake-refusal";
import { invitationView, type InvitationView } from "./invitation-view";

// What the invitation page shows (M3 design 7.4, W5), each case one step further down the decision than the one
// before: a page that showed a later view on an earlier case's inputs fails it.

const acme: InvitationPreview = {
  id: "i-1",
  role: 15,
  declined: false,
  workspace_name: "Acme",
  workspace_slug: "acme",
  workspace_logo_url: null,
};
const link = { invitationId: "i-1", token: "nrv_inv_x" };
const shown = { data: acme, error: undefined };
const page = { link, preview: shown, signedIn: true, mismatched: false };

describe("invitationView", () => {
  it.each<{ when: string; inputs: Parameters<typeof invitationView>[0]; view: InvitationView }>([
    {
      when: "the link has no token",
      inputs: { ...page, link: { invitationId: "i-1", token: null } },
      view: { kind: "invalid" },
    },
    {
      when: "the link has no invitation's id",
      inputs: { ...page, link: { invitationId: null, token: "nrv_inv_x" } },
      view: { kind: "invalid" },
    },
    {
      when: "nerve knows no invitation of the link",
      inputs: { ...page, preview: { data: undefined, error: refusal(404, "workspace.invitation_not_found") } },
      view: { kind: "invalid" },
    },
    {
      when: "the link's id is none",
      inputs: { ...page, preview: { data: undefined, error: refusal(400, "bad_request") } },
      view: { kind: "invalid" },
    },
    {
      when: "nerve could not be asked",
      inputs: { ...page, preview: { data: undefined, error: new TypeError("offline") } },
      view: { kind: "unavailable" },
    },
    {
      when: "the link's invitation is on its way",
      inputs: { ...page, preview: { data: undefined, error: undefined } },
      view: { kind: "loading" },
    },
    {
      when: "the invitation was declined",
      inputs: { ...page, preview: { data: { ...acme, declined: true }, error: undefined }, signedIn: false },
      view: { kind: "declined", invitation: { ...acme, declined: true }, token: "nrv_inv_x" },
    },
    {
      when: "no one is signed in",
      inputs: { ...page, signedIn: false, mismatched: true },
      view: { kind: "sign-in", invitation: acme, token: "nrv_inv_x" },
    },
    {
      when: "nerve found the caller's address not the invitation's",
      inputs: { ...page, mismatched: true },
      view: { kind: "mismatch", invitation: acme, token: "nrv_inv_x" },
    },
    { when: "the caller may answer it", inputs: page, view: { kind: "answer", invitation: acme, token: "nrv_inv_x" } },
  ])("shows $view.kind when $when", ({ inputs, view }) => {
    expect(invitationView(inputs)).toEqual(view);
  });
});
````

`web/apps/web/core/components/workspace/invitation-view.ts`（新文件，44 行）：

````file web/apps/web/core/components/workspace/invitation-view.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { InvitationPreview } from "@nerve/api-client";
// lib
import { ApiError } from "@/lib/api-error";

/**
 * What the invitation page shows (M3 design 7.4, W5): a link without its id or its token, or one nerve finds no
 * invitation by (400 for an id that is none, 404), is not valid; nerve not answering, a retry; then, as the link shows
 * the invitation, that it was declined; to one signed out, the ways to sign in or up and accept; to one whose address
 * nerve found not the invitation's as he answered, that it was sent to another (decision 1); else, accepting or
 * ignoring it.
 */
export type InvitationView =
  | { kind: "invalid" }
  | { kind: "loading" }
  | { kind: "unavailable" }
  | { kind: "declined" | "sign-in" | "mismatch" | "answer"; invitation: InvitationPreview; token: string };

/** Whether nerve found no invitation by the link: an id that is none (400), or none of it and its token (404). */
const notFound = (error: unknown) => error instanceof ApiError && (error.status === 400 || error.status === 404);

/** The page's view of the link's invitation, from the link, the invitation's preview and the caller. */
export function invitationView(page: {
  link: { invitationId: string | null; token: string | null };
  preview: { data: InvitationPreview | undefined; error: unknown };
  signedIn: boolean;
  /** Whether nerve answered the caller's acceptance or decline that the invitation is another address's. */
  mismatched: boolean;
}): InvitationView {
  const { link, preview, signedIn, mismatched } = page;
  const { token } = link;
  if (!link.invitationId || !token) return { kind: "invalid" };
  if (preview.error) return notFound(preview.error) ? { kind: "invalid" } : { kind: "unavailable" };
  const invitation = preview.data;
  if (!invitation) return { kind: "loading" };
  if (invitation.declined) return { kind: "declined", invitation, token };
  if (!signedIn) return { kind: "sign-in", invitation, token };
  if (mismatched) return { kind: "mismatch", invitation, token };
  return { kind: "answer", invitation, token };
}
````

`web/apps/web/core/components/workspace/use-invitation-answer.test.ts`（新文件，102 行）：

````file web/apps/web/core/components/workspace/use-invitation-answer.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { useInvitationAnswer } from "./use-invitation-answer";

// What the invitation page sends for an answer to the invitation of its link, and what it does with nerve's word on it
// (M3 design 7.1, 7.4; W5): the hook runs as a plain function, with stand-ins for the store's answers, which nerve
// settles when the test says, and for the opening of a workspace. Its session is fake-tab.ts's.

const page = vi.hoisted(() => ({
  acceptInvitation: vi.fn(),
  declineInvitation: vi.fn(),
  openWorkspace: vi.fn(),
  reread: vi.fn(),
  mismatched: vi.fn(),
}));
vi.mock("@/hooks/store/use-workspace", () => ({
  useWorkspace: () => ({ acceptInvitation: page.acceptInvitation, declineInvitation: page.declineInvitation }),
}));
vi.mock("@/hooks/use-open-workspace", () => ({ useOpenWorkspace: () => page.openWorkspace }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const acme = workspaceOf("acme", { role: 15 });
/** The page's hook, with the page's follow-ups: reading the invitation again, and saying it is another address's. */
const answers = () => useInvitationAnswer({ reread: page.reread, mismatched: page.mismatched });

/** Each answer the page gives, by its store change. */
const given = [
  { answer: "an acceptance", store: page.acceptInvitation, make: () => answers().accept("i-ada", "nrv_inv_x") },
  { answer: "a decline", store: page.declineInvitation, make: () => answers().decline("i-ada", "nrv_inv_x") },
];

/** What the page did after nerve's word: opened, read again, said another address's, toasted. */
const followed = () => [page.openWorkspace.mock.calls, page.reread.mock.calls, page.mismatched.mock.calls, toasts];

beforeEach(() => {
  signedIn();
  page.acceptInvitation.mockReset();
  page.acceptInvitation.mockResolvedValue(acme);
  page.declineInvitation.mockReset();
  page.declineInvitation.mockResolvedValue(undefined);
  for (const followUp of [page.openWorkspace, page.reread, page.mismatched]) followUp.mockReset();
  toasts.length = 0;
});

describe("useInvitationAnswer", () => {
  it("accepts the invitation with the link's token, then opens the workspace nerve answers", async () => {
    await answers().accept("i-ada", "nrv_inv_x");
    expect(page.acceptInvitation.mock.calls).toEqual([["i-ada", "nrv_inv_x"]]);
    expect(followed()).toEqual([[[acme]], [], [], []]);
  });

  it("declines the invitation with the link's token, then reads it again", async () => {
    await answers().decline("i-ada", "nrv_inv_x");
    expect(page.declineInvitation.mock.calls).toEqual([["i-ada", "nrv_inv_x"]]);
    expect(followed()).toEqual([[], [[]], [], []]);
  });

  it.each(given)("says the invitation is another address's when nerve refuses $answer so", async ({ store, make }) => {
    store.mockRejectedValueOnce(refusal(403, "workspace.invitation_email_mismatch"));
    await make();
    expect(followed()).toEqual([[], [], [[]], []]);
  });

  it.each(given)(
    "shows nerve's reason when it refuses $answer, and reads the invitation again",
    async ({ store, make }) => {
      store.mockRejectedValueOnce(refusal(404, "workspace.invitation_not_found"));
      await make();
      expect(followed()).toEqual([
        [],
        [[]],
        [],
        [{ type: "error", title: "toast.error", message: "errors.workspace_invitation_not_found" }],
      ]);
    }
  );

  describe.each(given)("$answer", ({ store, make }) => {
    it.each(lateSettlings)(
      "does nothing on the page when it $settles after another tab moved this one to another account",
      async ({ settle }) => {
        const answer = heldChange<undefined>();
        store.mockReturnValueOnce(answer.sent);
        const made = make();
        switchAccount();
        settle(answer);
        await made;
        expect(followed()).toEqual([[], [], [], []]);
      }
    );
  });
});
````

`web/apps/web/core/components/workspace/use-invitation-answer.ts`（新文件，52 行）：

````file web/apps/web/core/components/workspace/use-invitation-answer.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useOpenWorkspace } from "@/hooks/use-open-workspace";
// lib
import { ApiError } from "@/lib/api-error";
import { errorMessageKey } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";

/**
 * What the invitation page does with nerve's word on an answer: it reads the invitation again, or it says the
 * invitation is another address's (decision 1).
 */
type AnswerFollowUps = { reread: () => void; mismatched: () => void };

/**
 * The answers the invitation page gives to the invitation of invitationId, with its link's token (M3 design 7.4, W5):
 * accepted, the page opens the workspace, its membership's or the caller's own already (M3 design 3.8); declined, it
 * reads the invitation again, which says so. nerve's refusal: another address's invitation, which the page then says;
 * else its reason, and the invitation read again, as nerve has it now (answered meanwhile, or deleted). The page
 * follows an answer only in the session it was sent in (M3 design 7.1): once another tab has moved this one to
 * another account, the page is that account's, and does nothing of the answer.
 */
export function useInvitationAnswer({ reread, mismatched }: AnswerFollowUps) {
  const { acceptInvitation, declineInvitation } = useWorkspace();
  const openWorkspace = useOpenWorkspace();
  const { t } = useTranslation();
  const refused = (error: unknown) => {
    if (error instanceof ApiError && error.problem?.code === "workspace.invitation_email_mismatch") {
      mismatched();
      return;
    }
    setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) });
    reread();
  };

  return {
    accept: (invitationId: string, token: string) =>
      followInSession(() => acceptInvitation(invitationId, token), {
        done: (workspace) => void openWorkspace(workspace),
        failed: refused,
      }),
    decline: (invitationId: string, token: string) =>
      followInSession(() => declineInvitation(invitationId, token), { done: () => reread(), failed: refused }),
  };
}
````

`web/apps/web/core/hooks/use-open-workspace.test.ts`（新文件，54 行）：

````file web/apps/web/core/hooks/use-open-workspace.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { useOpenWorkspace } from "./use-open-workspace";

// How a page opens a workspace the caller has just joined (M3 design 3.14, 7.1): the hook runs as a plain function,
// with stand-ins for the router and the profile's store, whose change nerve answers when the test says. Its session is
// fake-tab.ts's.

const page = vi.hoisted(() => ({ updateUserProfile: vi.fn(), navigate: vi.fn() }));
vi.mock("react-router", () => ({ useNavigate: () => page.navigate }));
vi.mock("@/hooks/store/user", () => ({ useUserProfile: () => ({ updateUserProfile: page.updateUserProfile }) }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));

const acme = { id: "id-acme", slug: "acme" };

beforeEach(() => {
  signedIn();
  page.updateUserProfile.mockReset();
  page.updateUserProfile.mockResolvedValue({});
  page.navigate.mockClear();
});

describe("useOpenWorkspace", () => {
  it("writes the workspace as the one opened last, then goes there", async () => {
    await useOpenWorkspace()(acme);
    expect(page.updateUserProfile.mock.calls).toEqual([[{ last_workspace_id: "id-acme" }]]);
    expect(page.navigate.mock.calls).toEqual([["/acme"]]);
  });

  it("goes there when nerve does not save it", async () => {
    page.updateUserProfile.mockRejectedValueOnce(refusal(500, "internal_error"));
    await useOpenWorkspace()(acme);
    expect(page.navigate.mock.calls).toEqual([["/acme"]]);
  });

  it.each(lateSettlings)(
    "stays when the write $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const written = heldChange<undefined>();
      page.updateUserProfile.mockReturnValueOnce(written.sent);
      const opened = useOpenWorkspace()(acme);
      switchAccount();
      settle(written);
      await opened;
      expect(page.navigate.mock.calls).toEqual([]);
    }
  );
});
````

`web/apps/web/core/hooks/use-open-workspace.ts`（新文件，25 行）：

````file web/apps/web/core/hooks/use-open-workspace.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useNavigate } from "react-router";
import type { Workspace } from "@nerve/api-client";
// hooks
import { useUserProfile } from "@/hooks/store/user";
// lib
import { followInSession } from "@/lib/in-session";

/**
 * Opens a workspace the caller has just joined (M3 design 3.14, 7.4): writes it in his profile as the one he opened
 * last, then goes there. The hint is best-effort: the workspace opens whether nerve saves it or not. Sent in the
 * session the page is in, and followed only while the tab stays in it (M3 design 7.1).
 */
export function useOpenWorkspace(): (workspace: Pick<Workspace, "id" | "slug">) => Promise<void> {
  const navigate = useNavigate();
  const { updateUserProfile } = useUserProfile();
  return (workspace) => {
    const open = () => void navigate(`/${workspace.slug}`);
    return followInSession(() => updateUserProfile({ last_workspace_id: workspace.id }), { done: open, failed: open });
  };
}
````

`web/apps/web/core/lib/invitation-link.test.ts`（修改，2 处）：

````old web/apps/web/core/lib/invitation-link.test.ts
import { invitationLink } from "./invitation-link";
````
````new web/apps/web/core/lib/invitation-link.test.ts
import { invitationAuthPath, invitationLink } from "./invitation-link";
````
````old web/apps/web/core/lib/invitation-link.test.ts
  });
});

````
````new web/apps/web/core/lib/invitation-link.test.ts
  });
});

describe("invitationAuthPath", () => {
  it.each(["/", "/sign-up"] as const)("opens %s for the link's invitation, and comes back to the link", (path) => {
    const page = new URL(invitationAuthPath(path, { id: "i-1", token: "a&b" }), "https://nerve.example");
    expect([
      page.pathname,
      page.searchParams.get("invitation_id"),
      page.searchParams.get("token"),
      page.searchParams.get("next_path"),
    ]).toEqual([path, "i-1", "a&b", "/workspace-invitations?invitation_id=i-1&token=a%26b"]);
  });
});

````

`web/apps/web/core/lib/invitation-link.ts`（修改，1 处）：

````old web/apps/web/core/lib/invitation-link.ts
}

````
````new web/apps/web/core/lib/invitation-link.ts
}

/**
 * The sign-in page ("/") or the sign-up page for an invitation's link (M3 design 7.4): its header names the
 * invitation's workspace, and once signed in, the account comes back to the link (next_path, M2 design 3.18).
 */
export function invitationAuthPath(
  path: "/" | "/sign-up",
  invitation: Pick<WorkspaceInvitation, "id" | "token">
): string {
  const query = new URLSearchParams({
    invitation_id: invitation.id,
    token: invitation.token,
    next_path: invitationLink("", invitation),
  });
  return `${path}?${query.toString()}`;
}

````

- [ ] **Step 2: 邀请页和文案**

`web/apps/web/app/(all)/workspace-invitations/page.tsx`（整个文件，153 行）：

````whole web/apps/web/app/(all)/workspace-invitations/page.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { useSearchParams } from "react-router";
import { ROLE } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { BoxesOutline, CloseOutline, LogOutOutline, TickOutline, UserOutline } from "@makeplane/propel/icons";
// components
import { LogoSpinner } from "@/components/common/logo-spinner";
import { EmptySpace, EmptySpaceItem } from "@/components/ui/empty-space";
import { invitationView } from "@/components/workspace/invitation-view";
import { useInvitationAnswer } from "@/components/workspace/use-invitation-answer";
// helpers
import { EPageTypes } from "@/helpers/authentication.helper";
// hooks
import { useUser } from "@/hooks/store/user";
import { useInvitationPreview } from "@/hooks/use-invitation-preview";
// lib
import { invitationAuthPath } from "@/lib/invitation-link";
// wrappers
import { AuthenticationWrapper } from "@/lib/wrappers/authentication-wrapper";

/**
 * The page an invitation's link opens (M3 design 7.4, W5): what invitationView decides, from the link, its invitation
 * as the link shows it, and the caller.
 */
function WorkspaceInvitationPage() {
  // query params: the invitation's link
  const [searchParams] = useSearchParams();
  const link = { invitationId: searchParams.get("invitation_id"), token: searchParams.get("token") };
  // store hooks
  const { data: currentUser, signOut } = useUser();
  const { t } = useTranslation();
  // whether nerve answered the caller's answer that the invitation is another address's (decision 1)
  const [mismatched, setMismatched] = useState(false);

  const preview = useInvitationPreview(link.invitationId, link.token);
  const view = invitationView({
    link,
    preview: { data: preview.data, error: preview.error },
    signedIn: currentUser !== undefined,
    mismatched,
  });

  // the caller's answers, followed only in the session they were sent in (use-invitation-answer.ts)
  const { accept, decline } = useInvitationAnswer({
    reread: () => void preview.mutate(),
    mismatched: () => setMismatched(true),
  });

  const home = currentUser ? (
    <EmptySpaceItem Icon={BoxesOutline} title={t("workspace_invitation.home")} href="/" />
  ) : (
    <EmptySpaceItem Icon={UserOutline} title={t("workspace_invitation.sign_in")} href="/" />
  );

  const content = () => {
    switch (view.kind) {
      case "loading":
        return <LogoSpinner />;
      case "invalid":
        return (
          <EmptySpace
            title={t("workspace_invitation.invalid.title")}
            description={t("workspace_invitation.invalid.description")}
          >
            {home}
          </EmptySpace>
        );
      case "unavailable":
        return (
          <EmptySpace title={t("errors.unreachable")} description="">
            <EmptySpaceItem Icon={BoxesOutline} title={t("common.retry")} action={() => void preview.mutate()} />
          </EmptySpace>
        );
      case "declined":
        return (
          <EmptySpace
            title={t("workspace_invitation.declined.title")}
            description={t("workspace_invitation.declined.description", { workspace: view.invitation.workspace_name })}
          >
            {home}
          </EmptySpace>
        );
      case "sign-in":
        return (
          <EmptySpace
            title={t("workspace_invitation.invited", {
              workspace: view.invitation.workspace_name,
              role: ROLE[view.invitation.role],
            })}
            description={t("workspace_invitation.addressed")}
          >
            <EmptySpaceItem
              Icon={UserOutline}
              title={t("workspace_invitation.sign_in_to_accept")}
              href={invitationAuthPath("/", { id: view.invitation.id, token: view.token })}
            />
            <EmptySpaceItem
              Icon={UserOutline}
              title={t("workspace_invitation.sign_up_to_accept")}
              href={invitationAuthPath("/sign-up", { id: view.invitation.id, token: view.token })}
            />
          </EmptySpace>
        );
      case "mismatch":
        return (
          <EmptySpace
            title={t("errors.workspace_invitation_email_mismatch")}
            description={t("workspace_invitation.mismatch")}
          >
            <EmptySpaceItem Icon={LogOutOutline} title={t("sign_out")} action={() => void signOut()} />
          </EmptySpace>
        );
      case "answer": {
        const { invitation, token } = view;
        return (
          <EmptySpace
            title={t("workspace_invitation.invited", {
              workspace: invitation.workspace_name,
              role: ROLE[invitation.role],
            })}
            description={t("workspace_invitation.description")}
          >
            <EmptySpaceItem
              Icon={TickOutline}
              title={t("workspace_invitation.accept")}
              action={() => void accept(invitation.id, token)}
            />
            <EmptySpaceItem
              Icon={CloseOutline}
              title={t("workspace_invitation.ignore")}
              action={() => void decline(invitation.id, token)}
            />
          </EmptySpace>
        );
      }
    }
  };

  return (
    <AuthenticationWrapper pageType={EPageTypes.PUBLIC}>
      <div className="flex h-full w-full flex-col items-center justify-center px-3">{content()}</div>
    </AuthenticationWrapper>
  );
}

export default observer(WorkspaceInvitationPage);
````

`web/packages/i18n/src/locales/en/workspace.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/workspace.json
        "message": "Workspace could not be created. Please try again."
      }
    }
  },
````
````new web/packages/i18n/src/locales/en/workspace.json
        "message": "Workspace could not be created. Please try again."
      }
    }
  },
  "workspace_invitation": {
    "invited": "You have been invited to {workspace} as {role}.",
    "description": "Your workspace is where you'll create projects, collaborate on your work items, and organize different streams of work in your Nerve account.",
    "accept": "Accept",
    "ignore": "Ignore",
    "addressed": "The invitation was sent to one email address: sign in or sign up with it to accept.",
    "sign_in_to_accept": "Sign in to accept",
    "sign_up_to_accept": "Sign up to accept",
    "mismatch": "Sign out, and sign in with the address it was sent to.",
    "declined": {
      "title": "You declined this invitation.",
      "description": "Ask an admin of {workspace} for a new one."
    },
    "invalid": {
      "title": "This invitation link is not valid.",
      "description": "Ask an admin of the workspace for a new link."
    },
    "home": "Continue to home",
    "sign_in": "Sign in"
  },
````

`web/packages/i18n/src/locales/zh-CN/workspace.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/workspace.json
        "message": "工作区创建失败。请重试。"
      }
    }
  },
````
````new web/packages/i18n/src/locales/zh-CN/workspace.json
        "message": "工作区创建失败。请重试。"
      }
    }
  },
  "workspace_invitation": {
    "invited": "您受邀以 {role} 的身份加入 {workspace}。",
    "description": "工作区是您创建项目、协作处理工作项、组织不同工作流的地方。",
    "accept": "接受",
    "ignore": "忽略",
    "addressed": "这份邀请发给了一个指定的邮箱：用它登录或注册即可接受。",
    "sign_in_to_accept": "登录以接受",
    "sign_up_to_accept": "注册以接受",
    "mismatch": "请退出登录，再用收到邀请的邮箱登录。",
    "declined": {
      "title": "您已忽略这份邀请。",
      "description": "请向 {workspace} 的管理员要一份新的邀请。"
    },
    "invalid": {
      "title": "这个邀请链接无效。",
      "description": "请向工作区的管理员要一个新的链接。"
    },
    "home": "回到首页",
    "sign_in": "登录"
  },
````

- [ ] **Step 3: 静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/hooks/use-copy-invitation-link.ts"
````
````new .oxlintrc.json
        "web/apps/web/core/hooks/use-copy-invitation-link.ts",
        "web/apps/web/core/components/workspace/invitation-view.ts",
        "web/apps/web/core/components/workspace/invitation-view.test.ts",
        "web/apps/web/core/components/workspace/use-invitation-answer.ts",
        "web/apps/web/core/components/workspace/use-invitation-answer.test.ts",
        "web/apps/web/core/hooks/use-open-workspace.ts",
        "web/apps/web/core/hooks/use-open-workspace.test.ts"
````

- [ ] **Step 4: 端到端**

`e2e/stories/workspace/w5-invitation-link.spec.ts`（修改，3 处）：

````old e2e/stories/workspace/w5-invitation-link.spec.ts
import { createWorkspace, invite, slugFor, type Api, type WorkspaceInvitation } from "../../fixtures/api";
import { expectInvitations, expectMembership } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
````
````new e2e/stories/workspace/w5-invitation-link.spec.ts
import { createWorkspace, invitationTo, invite, slugFor, type Api, type WorkspaceInvitation } from "../../fixtures/api";
import { expectInvitations, expectMembership, lastWorkspaceOf } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, password, register } from "../../fixtures/auth";
import { submitSignIn } from "../../fixtures/auth-pages";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { answerTo, registerOnboarded, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { invitationLinkOf } from "../../fixtures/workspace-pages";
````
````old e2e/stories/workspace/w5-invitation-link.spec.ts
// W5, accept or decline an invitation by its link (M3 design 2, 3.8; decision 1). The page version comes with
// /workspace-invitations (P9).
````
````new e2e/stories/workspace/w5-invitation-link.spec.ts
// W5, accept or decline an invitation by its link (M3 design 2, 3.8; decision 1).
````
````old e2e/stories/workspace/w5-invitation-link.spec.ts
  await expectMembership(db, slug, daveEmail, null);
});

````
````new e2e/stories/workspace/w5-invitation-link.spec.ts
  await expectMembership(db, slug, daveEmail, null);
});

test("W5 (page): signed out, the link shows the workspace and the role and sends the invitee to sign in and back; she accepts and lands in the workspace, written as the one she opened last; another invitation she ignores says she declined it; one deleted while its link is open is refused with the reason, and is no longer valid", async ({
  api,
  baseURL,
  browser,
  db,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const slug = slugFor(testInfo);
  const other = slugFor(testInfo, "other");
  const acme = await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  await createWorkspace(api, admin.access_token, { name: "Other", slug: other });
  const carolEmail = emailFor(testInfo, "carol");
  await registerOnboarded(api, carolEmail);
  await invite(api, admin.access_token, slug, [{ email: carolEmail, role: 5 }]);
  await invite(api, admin.access_token, other, [{ email: carolEmail, role: 15 }]);
  const toAcme = await invitationTo(api, admin.access_token, slug, carolEmail);
  const toOther = await invitationTo(api, admin.access_token, other, carolEmail);
  const third = slugFor(testInfo, "third");
  await createWorkspace(api, admin.access_token, { name: "Third", slug: third });
  await invite(api, admin.access_token, third, [{ email: carolEmail, role: 15 }]);
  const toThird = await invitationTo(api, admin.access_token, third, carolEmail);

  const context = await browser.newContext({ baseURL });
  const page = await context.newPage();
  const watch = await watchPage(page);
  await page.goto(invitationLinkOf(toAcme));
  await expect(page.getByText("You have been invited to Acme as Guest.")).toBeVisible();
  await expect(page.getByText(carolEmail)).toHaveCount(0);
  await page.getByRole("button", { name: "Sign in to accept" }).click();
  expect(await submitSignIn(page, carolEmail, password)).toBe(200);
  await expect(page).toHaveURL(invitationLinkOf(toAcme));

  // She accepts: the page sends the link's token, and opens Acme, which it writes as the one she opened last.
  const accepted = await sentTo(page, "POST", `/api/v0/workspace-invitations/${toAcme.id}/accept`, () =>
    page.getByRole("button", { name: "Accept" }).click()
  );
  expect([accepted.answer.status(), accepted.body]).toEqual([200, { token: toAcme.token }]);
  await expect(page).toHaveURL(`/${slug}`);
  await expectMembership(db, slug, carolEmail, { role: 5, is_active: true });
  await expect.poll(() => lastWorkspaceOf(db, carolEmail)).toBe(acme.id);

  // She ignores Other's: the page reads the invitation again, which says she declined it.
  await page.goto(invitationLinkOf(toOther));
  const declined = await sentTo(page, "POST", `/api/v0/workspace-invitations/${toOther.id}/decline`, () =>
    page.getByRole("button", { name: "Ignore" }).click()
  );
  expect([declined.answer.status(), declined.body]).toEqual([204, { token: toOther.token }]);
  await expect(page.getByText("You declined this invitation.")).toBeVisible();
  await expectMembership(db, other, carolEmail, null);
  await expectInvitations(db, other, adminEmail, [
    { email: carolEmail, role: 15, accepted: false, responded: true, deleted: false },
  ]);

  // Third's, deleted while its link is open: nerve refuses the acceptance and says why; the page reads the link again.
  await page.goto(invitationLinkOf(toThird));
  await expect(page.getByRole("button", { name: "Accept" })).toBeVisible();
  const deleted = await api.DELETE("/api/v0/workspace-invitations/{invitation_id}", {
    params: { path: { invitation_id: toThird.id } },
    headers: bearer(admin.access_token),
  });
  expect(deleted.response.status).toBe(204);
  const accept = `/api/v0/workspace-invitations/${toThird.id}/accept`;
  const gone = await answerTo(page, "POST", accept, () => page.getByRole("button", { name: "Accept" }).click());
  expect(gone.status()).toBe(404);
  await expect(page.getByText("The invitation does not exist, or its link is not valid.")).toBeVisible();
  await expect(page.getByText("This invitation link is not valid.")).toBeVisible();
  await expectMembership(db, third, carolEmail, null);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`404 POST ${accept}`, `404 GET /api/v0/workspace-invitations/${toThird.id}`],
    [],
    [],
  ]);
  // Acme's home logs the hint once; the browser reports the two refusals.
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING],
    errors: [
      "Failed to load resource: the server responded with a status of 404 (Not Found)",
      "Failed to load resource: the server responded with a status of 404 (Not Found)",
    ],
  });
  await context.close();
});

test("W5 (page): another address's invitation, once nerve refuses the answer, says it was sent to another address, offers no answer but signing out, and changes nothing; a link with a changed token, or none, is not valid", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  const carolEmail = emailFor(testInfo, "carol");
  await invite(api, admin.access_token, slug, [{ email: carolEmail, role: 15 }]);
  const toCarol = await invitationTo(api, admin.access_token, slug, carolEmail);
  const daveEmail = emailFor(testInfo, "dave");
  const page = await signedInPage(await registerOnboarded(api, daveEmail));
  const watch = await watchPage(page);

  await page.goto(invitationLinkOf(toCarol));
  const accept = `/api/v0/workspace-invitations/${toCarol.id}/accept`;
  const refused = await answerTo(page, "POST", accept, () => page.getByRole("button", { name: "Accept" }).click());
  expect(refused.status()).toBe(403);
  await expect(page.getByText("This invitation was sent to another email address.")).toBeVisible();
  await expect(page.getByText(carolEmail)).toHaveCount(0);
  await expect(page.getByRole("button", { name: /^(Accept|Ignore)$/ })).toHaveCount(0);
  await expectMembership(db, slug, daveEmail, null);
  await expectInvitations(db, slug, adminEmail, [
    { email: carolEmail, role: 15, accepted: false, responded: false, deleted: false },
  ]);
  // Signed out, the link offers to sign in as the invitee.
  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page.getByRole("button", { name: "Sign in to accept" })).toBeVisible();

  // A token changed in one character: nerve finds no invitation. No token: the page does not ask.
  const changed = toCarol.token.slice(0, 10) + (toCarol.token[10] === "A" ? "B" : "A") + toCarol.token.slice(11);
  await page.goto(invitationLinkOf({ id: toCarol.id, token: changed }));
  await expect(page.getByText("This invitation link is not valid.")).toBeVisible();
  await page.goto(`/workspace-invitations?invitation_id=${toCarol.id}`);
  await expect(page.getByText("This invitation link is not valid.")).toBeVisible();
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`403 POST ${accept}`, `404 GET /api/v0/workspace-invitations/${toCarol.id}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, {
    errors: [
      "Failed to load resource: the server responded with a status of 403 (Forbidden)",
      "Failed to load resource: the server responded with a status of 404 (Not Found)",
    ],
  });
});

````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、3 个例外，没有命中；web 的 oxlint 359 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 82 个全部通过。

- [ ] **Step 6: 提交**

```bash
git add .oxlintrc.json e2e/stories/workspace/w5-invitation-link.spec.ts 'web/apps/web/app/(all)/workspace-invitations/page.tsx' web/apps/web/core/components/workspace/invitation-view.test.ts web/apps/web/core/components/workspace/invitation-view.ts web/apps/web/core/components/workspace/use-invitation-answer.test.ts web/apps/web/core/components/workspace/use-invitation-answer.ts web/apps/web/core/hooks/use-open-workspace.test.ts web/apps/web/core/hooks/use-open-workspace.ts web/apps/web/core/lib/invitation-link.test.ts web/apps/web/core/lib/invitation-link.ts web/packages/i18n/src/locales/en/workspace.json web/packages/i18n/src/locales/zh-CN/workspace.json
```
```bash
git commit -m "feat(M3/P9): the invitation page shows what invitationView decides, and opens the workspace joined

invitationView decides the page from the link, its invitation's preview
and the caller: not valid, unreachable, declined, sign in or up to
accept, sent to another address, or accept and ignore. Accepted, the
page writes the workspace as the one opened last (best effort) and goes
there; a decline reads the invitation again; a refusal says nerve's
reason and reads the invitation again; another address's invitation
offers signing out alone. useInvitationAnswer gives the answers and
follows each only in its session. The page never shows the invited
address. The new modules join the non-null override. W5's page versions.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p9.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T4.1` | 接受邀请之后不论会话都进入工作区 | `use-open-workspace.test.ts` | vitest |
| `T4.2` | 发给另一个邮箱的邀请把 nerve 的原因放在提示里，仍可接受和忽略（`useInvitationAnswer`） | `use-invitation-answer.test.ts`、故事 W5 | vitest；端到端 |
| `T4.3` | 忽略之后不重新读取邀请 | `use-invitation-answer.test.ts`、故事 W5 | vitest；端到端 |
| `T4.4` | 接受的工作区不写成上次打开的工作区 | `use-open-workspace.test.ts`、故事 W5 | vitest；端到端 |
| `T4.5` | 已忽略的邀请让未登录的人去登录 | `invitation-view.test.ts` | vitest |
| `T4.6` | `invitation_id` 不是 id 的链接（nerve 答 400）显示重试 | `invitation-view.test.ts` | vitest |
| `T4.7` | 没有令牌的链接一直转圈 | `invitation-view.test.ts`、故事 W5 | vitest；端到端 |
| `T4.8` | 从链接去登录之后不回到链接 | `invitation-link.test.ts`、故事 W5 | vitest；端到端 |
| `T4.9` | 邀请页丢掉 nerve 对接受、忽略的拒绝（`useInvitationAnswer`） | `use-invitation-answer.test.ts`、故事 W5 | vitest；端到端 |
| `T4.10` | 接受邀请不论会话都跟进（预检的 PF2） | `use-invitation-answer.test.ts` | vitest |
| `T4.11` | 忽略邀请不论会话都跟进（预检的 PF2b） | `use-invitation-answer.test.ts` | vitest |
| `T4.12` | 邀请页的回答（`use-invitation-answer.ts`，范围内）用非空断言 | oxlint（`check:lint`） | 静态 |

---

### Task 5: 创建工作区的共用 hook；`/create-workspace`；W1 的页面版本（一）

**Files:**
- Create: `web/apps/web/core/components/workspace/use-create-workspace.test.ts`、`web/apps/web/core/components/workspace/use-create-workspace.ts`
- Modify: `.oxlintrc.json`、`e2e/stories/workspace/w1-create-workspace.spec.ts`、`web/apps/web/app/(all)/create-workspace/page.tsx`、`web/apps/web/core/components/workspace/create-workspace-form.tsx`、`web/packages/i18n/src/locales/en/workspace.json`、`web/packages/i18n/src/locales/zh-CN/workspace.json`

**Interfaces:**
- Produces（spec 2.5；M3 设计 2 的 W1，3.10，3.11，3.14，7.1；P8a 的 P9、P21 和评审的 M7）：
  - `core/components/workspace/use-create-workspace.ts`：
    - `CreationForm = Pick<WorkspaceCreate, "name" | "slug"> & { organization_size: OrganizationSize | null }`（没有选规模时为 `null`，规模的选择保持受控；Task 1 的 general 页同理）。
    - `slugFrom(text)`：输入的文字变成 slug（小写，空格换成 `-`）；字段显示的就是表单发出的（原来字段显示 `value.toLocaleLowerCase().trim().replace(/ /g, "-")`，表单里存、发出的却只是小写：显示 `acme-up`，发出 `acme up`）。
    - `creationRefusal(error): CreationRefusal`，`{ kind: "fields"; fields } | { kind: "toast"; message }`：slug 在检查之后被别人占用（`workspace.slug_taken`）落到 slug 下；nerve 的字段错误只指向 `name`、`slug` 时落到字段下（slug 的 `not_allowed` 是保留的，`invalid_format` 是格式不对，其余按 `FIELD_ERROR_MESSAGES`），否则按 `errorMessageKey` 提示（原来两种失败都提示同一句"创建失败"，P21）。
    - `useCreateWorkspace()` 给出 `(form, setError) => Promise<Workspace | undefined>`：先问 nerve slug 能不能用（`checkWorkspaceSlug`），不能时不发出，原因（已被占用、保留、格式不对）放在 slug 下；能用时发 `{ name, slug, organization_size? }`（生成的 `WorkspaceCreate`，规模没有时不发）；成功时提示并交回工作区；整个过程经 `followInSession`，换账户之后既不提示也不交回（页面已是那个账户的）。`/create-workspace` 和 Task 6 的新手引导共用它。
  - `CreateWorkspaceForm`：收 `onCreated(workspace)`（原来是 `onSubmit`），值是 `CreationForm`，经 `useCreateWorkspace` 和表单的 `setError`；表单自己的 slug 检查（`validateSlug`、`invalidSlug`、`slugError` 和它们的两段文字）删除：nerve 的检查是唯一的检查（原来的检查允许 Unicode 字母、文案与 nerve 的不同，还可能和 nerve 的原因一起显示两遍）。
  - `/create-workspace` 页：`defaultValues` 是 `CreationForm`；创建之后 `useOpenWorkspace`（Task 4：写上次打开的工作区，再打开它，只在会话里）。创建关闭时（`workspace_creation_enabled` 为假）说明创建已关闭、请工作区的管理员给一个邀请链接，与新手引导的说法一致（第 2 节 W1："两个入口都显示'创建工作区已关闭'"）；Plane 的"只有实例管理员能创建工作区"和给实例管理员写信的按钮（`mailto:`）删除：Nerve 没有实例管理员（M2 设计 3.16）。
  - i18n（en、zh-CN 的 `workspace.json`）：`url_alphanumeric` 改成 nerve 的规则（小写字母、数字、`-` 和 `_`），加 `url_reserved`；`creation_disabled` 的标题和说明改写，`creation_disabled.request_button` 和 `request_email` 的两条删除。

- 静态检查（总体设计 7.7）：`.oxlintrc.json` 的非空断言的范围加上 `use-create-workspace.ts` 和它的测试。

**Tests:**
- vitest：`use-create-workspace.test.ts`：`useCreateWorkspace` 的 `asks whether the slug is free, creates the workspace from the form, says so and gives it`；`sends no size when the form has none chosen`；`sends nothing for a slug nerve says is $reason, and says so under it`（已被占用、保留、格式不对）；`shows nerve's refusal under the fields it names`；`shows nerve's reason when it names no field of the form`；`neither speaks nor gives the workspace when the creation $settles after another tab moved this one to another account`（`lateSettlings`）。`creationRefusal` 的 `shows $refused`、`slugFrom` 的 `makes '$typed' into '$slug'`（`it.each`）。
- 端到端：`W1 (page): at /create-workspace an onboarded account is told under the field of a taken, a reserved or an invalid slug, and nothing is created; then it creates a workspace with the slug the field shows, which opens, written as the one opened last; on a nerve with creation switched off the page says so`（三种 slug 都没有 `POST`，"含大写"由字段转成小写，格式不对的样例是 `café`，spec 第 3 节；请求体经 `sentTo`；创建关闭的 nerve 上页面说明已关闭，没有表单，也没有 `mailto:` 链接）。

- [ ] **Step 1: 共用的 hook**

`web/apps/web/core/components/workspace/use-create-workspace.test.ts`（新文件，165 行）：

````file web/apps/web/core/components/workspace/use-create-workspace.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import {
  creationRefusal,
  slugFrom,
  useCreateWorkspace,
  type CreationForm,
  type CreationRefusal,
} from "./use-create-workspace";

// How a creation form creates a workspace (M3 design 2 W1, 7.1): the hook runs as a plain function, with stand-ins
// for the workspace store's check of the slug and its creation, which nerve answers when the test says, and for the
// form's setError. Its session is fake-tab.ts's.

const page = vi.hoisted(() => ({ checkWorkspaceSlug: vi.fn(), createWorkspace: vi.fn(), setError: vi.fn() }));
vi.mock("@/hooks/store/use-workspace", () => ({
  useWorkspace: () => ({ checkWorkspaceSlug: page.checkWorkspaceSlug, createWorkspace: page.createWorkspace }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const form: CreationForm = { name: "Acme", slug: "acme", organization_size: "2-10" };
const acme = workspaceOf("acme", { name: "Acme", role: 20 });
const create = (values = form) => useCreateWorkspace()(values, page.setError);

beforeEach(() => {
  signedIn();
  page.checkWorkspaceSlug.mockReset();
  page.checkWorkspaceSlug.mockResolvedValue({ available: true });
  page.createWorkspace.mockReset();
  page.createWorkspace.mockResolvedValue(acme);
  page.setError.mockReset();
  toasts.length = 0;
});

describe("useCreateWorkspace", () => {
  it("asks whether the slug is free, creates the workspace from the form, says so and gives it", async () => {
    expect(await create()).toBe(acme);
    expect([page.checkWorkspaceSlug.mock.calls, page.createWorkspace.mock.calls]).toEqual([[["acme"]], [[form]]]);
    expect(page.setError.mock.calls).toEqual([]);
    expect(toasts).toEqual([
      {
        type: "success",
        title: "workspace_creation.toast.success.title",
        message: "workspace_creation.toast.success.message",
      },
    ]);
  });

  it("sends no size when the form has none chosen", async () => {
    await create({ ...form, organization_size: null });
    expect(page.createWorkspace.mock.calls).toEqual([[{ name: "Acme", slug: "acme" }]]);
  });

  it.each([
    { reason: "taken", message: "workspace_creation.errors.validation.url_already_taken" },
    { reason: "reserved", message: "workspace_creation.errors.validation.url_reserved" },
    { reason: "invalid", message: "workspace_creation.errors.validation.url_alphanumeric" },
  ])("sends nothing for a slug nerve says is $reason, and says so under it", async ({ reason, message }) => {
    page.checkWorkspaceSlug.mockResolvedValueOnce({ available: false, reason });
    expect(await create()).toBeUndefined();
    expect([page.createWorkspace.mock.calls, page.setError.mock.calls, toasts]).toEqual([
      [],
      [["slug", { type: "server", message }]],
      [],
    ]);
  });

  it("shows nerve's refusal under the fields it names", async () => {
    page.createWorkspace.mockRejectedValueOnce(
      refusal(422, "validation_failed", [
        { field: "name", code: "contains_url" },
        { field: "slug", code: "not_allowed" },
      ])
    );
    expect(await create()).toBeUndefined();
    expect([page.setError.mock.calls, toasts]).toEqual([
      [
        ["name", { type: "server", message: "errors.field.contains_url" }],
        ["slug", { type: "server", message: "workspace_creation.errors.validation.url_reserved" }],
      ],
      [],
    ]);
  });

  it("shows nerve's reason when it names no field of the form", async () => {
    page.createWorkspace.mockRejectedValueOnce(refusal(403, "workspace.creation_disabled"));
    expect(await create()).toBeUndefined();
    expect(page.setError.mock.calls).toEqual([]);
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.workspace_creation_disabled" }]);
  });

  it.each(lateSettlings)(
    "neither speaks nor gives the workspace when the creation $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const created = heldChange<undefined>();
      page.createWorkspace.mockReturnValueOnce(created.sent);
      const creating = create();
      await vi.waitFor(() => expect(page.createWorkspace).toHaveBeenCalledTimes(1));
      switchAccount();
      settle(created);
      expect(await creating).toBeUndefined();
      expect([page.setError.mock.calls, toasts]).toEqual([[], []]);
    }
  );
});

describe("creationRefusal", () => {
  it.each<{ refused: string; error: unknown; shown: CreationRefusal }>([
    {
      refused: "a taken slug",
      error: refusal(409, "workspace.slug_taken"),
      shown: { kind: "fields", fields: { slug: "workspace_creation.errors.validation.url_already_taken" } },
    },
    {
      refused: "a reserved slug and a name with a web address",
      error: refusal(422, "validation_failed", [
        { field: "slug", code: "not_allowed" },
        { field: "name", code: "contains_url" },
      ]),
      shown: {
        kind: "fields",
        fields: { slug: "workspace_creation.errors.validation.url_reserved", name: "errors.field.contains_url" },
      },
    },
    {
      refused: "a field the form does not have",
      error: refusal(422, "validation_failed", [
        { field: "name", code: "too_long" },
        { field: "timezone", code: "invalid_format" },
      ]),
      shown: { kind: "toast", message: "errors.validation_failed" },
    },
    {
      refused: "creation switched off",
      error: refusal(403, "workspace.creation_disabled"),
      shown: { kind: "toast", message: "errors.workspace_creation_disabled" },
    },
    {
      refused: "a failure without an answer",
      error: new TypeError("offline"),
      shown: { kind: "toast", message: "errors.unknown" },
    },
  ])("shows $refused", ({ error, shown }) => {
    expect(creationRefusal(error)).toEqual(shown);
  });
});

describe("slugFrom", () => {
  it.each([
    { typed: "Acme Two", slug: "acme-two" },
    { typed: "acme ", slug: "acme-" },
  ])("makes '$typed' into '$slug'", ({ typed, slug }) => {
    expect(slugFrom(typed)).toBe(slug);
  });
});
````

`web/apps/web/core/components/workspace/use-create-workspace.ts`（新文件，118 行）：

````file web/apps/web/core/components/workspace/use-create-workspace.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { UseFormSetError } from "react-hook-form";
import type { FieldError, OrganizationSize, SlugAvailability, Workspace, WorkspaceCreate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
// lib
import { ApiError } from "@/lib/api-error";
import { FIELD_ERROR_MESSAGES, errorMessageKey } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";

/** A creation form's values: no size until one is chosen (null, which keeps a select of sizes controlled). */
export type CreationForm = Pick<WorkspaceCreate, "name" | "slug"> & { organization_size: OrganizationSize | null };

/** The slug that text typed into a slug's field makes, as the field shows it and the form sends it. */
export const slugFrom = (text: string): string => text.toLowerCase().replace(/ /g, "-");

/** The fields of the creation's form that nerve may refuse, each with the i18n key of the message under it. */
type CreationFields = Partial<Record<"name" | "slug", string>>;

/** What the creation's form shows of nerve's refusal: under the fields it names, or its reason in a toast. */
export type CreationRefusal = { kind: "fields"; fields: CreationFields } | { kind: "toast"; message: string };

/** Why nerve says a slug cannot name a new workspace (SlugAvailability.reason). */
type SlugUnavailable = NonNullable<SlugAvailability["reason"]>;

/** How a creation ends: the workspace created, or none, its slug unavailable. */
type Attempt = { kind: "created"; workspace: Workspace } | { kind: "unavailable"; reason: SlugUnavailable };

/** The message under the slug, by why nerve says it cannot name a new workspace. */
const SLUG_MESSAGES: Record<SlugUnavailable, string> = {
  taken: "workspace_creation.errors.validation.url_already_taken",
  reserved: "workspace_creation.errors.validation.url_reserved",
  invalid: "workspace_creation.errors.validation.url_alphanumeric",
};

/** A field error of nerve's on the slug: a reserved slug is not allowed; one of other characters, not of the format. */
const SLUG_CODES: Partial<Record<FieldError["code"], string>> = {
  not_allowed: SLUG_MESSAGES.reserved,
  invalid_format: SLUG_MESSAGES.invalid,
};

/** nerve's refusal of a creation (M3 design 2 W1), as the form shows it. */
export function creationRefusal(error: unknown): CreationRefusal {
  if (!(error instanceof ApiError)) return { kind: "toast", message: errorMessageKey(error) };
  if (error.problem?.code === "workspace.slug_taken") return { kind: "fields", fields: { slug: SLUG_MESSAGES.taken } };
  const named = error.problem?.errors ?? [];
  const fields: CreationFields = {};
  for (const { field, code } of named) {
    if (field === "slug") fields.slug = SLUG_CODES[code] ?? FIELD_ERROR_MESSAGES[code];
    if (field === "name") fields.name = FIELD_ERROR_MESSAGES[code];
  }
  const shown = Object.keys(fields).length;
  return shown > 0 && shown === named.length
    ? { kind: "fields", fields }
    : { kind: "toast", message: errorMessageKey(error) };
}

/**
 * Creates a workspace from a creation form's values (M3 design 2 W1, 3.10): asks nerve first whether the slug can name
 * it, and creates it only then, with the fields of WorkspaceCreate the form has; shows nerve's refusal under the fields
 * it names (the form's setError), else in a toast; says so once created. Gives the workspace created, or undefined
 * when it was not, or when the tab moved to another account meanwhile: the page is that account's then, and goes no
 * further (M3 design 7.1).
 */
export function useCreateWorkspace(): (
  form: CreationForm,
  setError: UseFormSetError<CreationForm>
) => Promise<Workspace | undefined> {
  const { checkWorkspaceSlug, createWorkspace } = useWorkspace();
  const { t } = useTranslation();

  // the slug's check decides: an unavailable slug is not sent
  const attempt = async (form: CreationForm): Promise<Attempt> => {
    const slug = await checkWorkspaceSlug(form.slug);
    if (!slug.available) return { kind: "unavailable", reason: slug.reason ?? "taken" };
    const data: WorkspaceCreate = {
      name: form.name,
      slug: form.slug,
      organization_size: form.organization_size ?? undefined,
    };
    return { kind: "created", workspace: await createWorkspace(data) };
  };

  return async (form, setError) => {
    // nerve's reasons, each under the field it is about
    const refused = (fields: CreationFields) => {
      if (fields.name) setError("name", { type: "server", message: t(fields.name) });
      if (fields.slug) setError("slug", { type: "server", message: t(fields.slug) });
    };
    let workspace: Workspace | undefined;
    await followInSession(() => attempt(form), {
      done: (outcome) => {
        if (outcome.kind === "unavailable") {
          refused({ slug: SLUG_MESSAGES[outcome.reason] });
          return;
        }
        workspace = outcome.workspace;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("workspace_creation.toast.success.title"),
          message: t("workspace_creation.toast.success.message"),
        });
      },
      failed: (error) => {
        const refusal = creationRefusal(error);
        if (refusal.kind === "fields") refused(refusal.fields);
        else setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(refusal.message) });
      },
    });
    return workspace;
  };
}
````

- [ ] **Step 2: 表单、页面和文案**

`web/apps/web/app/(all)/create-workspace/page.tsx`（修改，8 处）：

````old web/apps/web/app/(all)/create-workspace/page.tsx
import { Button, getButtonStyling } from "@nerve/propel/button";
import type { Workspace, WorkspaceCreate } from "@nerve/api-client";
````
````new web/apps/web/app/(all)/create-workspace/page.tsx
import { Button } from "@nerve/propel/button";
````
````old web/apps/web/app/(all)/create-workspace/page.tsx
import { CreateWorkspaceForm } from "@/components/workspace/create-workspace-form";
````
````new web/apps/web/app/(all)/create-workspace/page.tsx
import { CreateWorkspaceForm } from "@/components/workspace/create-workspace-form";
import type { CreationForm } from "@/components/workspace/use-create-workspace";
````
````old web/apps/web/app/(all)/create-workspace/page.tsx
import { useUser, useUserProfile } from "@/hooks/store/user";
import { useInstance } from "@/hooks/store/use-instance";
````
````new web/apps/web/app/(all)/create-workspace/page.tsx
import { useUser } from "@/hooks/store/user";
import { useInstance } from "@/hooks/store/use-instance";
import { useOpenWorkspace } from "@/hooks/use-open-workspace";
````
````old web/apps/web/app/(all)/create-workspace/page.tsx
  const { updateUserProfile } = useUserProfile();
````
````new web/apps/web/app/(all)/create-workspace/page.tsx
  const openWorkspace = useOpenWorkspace();
````
````old web/apps/web/app/(all)/create-workspace/page.tsx
  const [defaultValues, setDefaultValues] = useState<Pick<WorkspaceCreate, "name" | "slug" | "organization_size">>({
    name: "",
    slug: "",
  });
````
````new web/apps/web/app/(all)/create-workspace/page.tsx
  const [defaultValues, setDefaultValues] = useState<CreationForm>({ name: "", slug: "", organization_size: null });
````
````old web/apps/web/app/(all)/create-workspace/page.tsx
  const isWorkspaceCreationDisabled = config?.workspace_creation_enabled === false;

  // methods
  const getMailtoHref = () => {
    const subject = t("workspace_creation.request_email.subject");
    const body = t("workspace_creation.request_email.body", {
      firstName: currentUser?.first_name || "",
      lastName: currentUser?.last_name || "",
      email: currentUser?.email || "",
    });

    return `mailto:?subject=${encodeURIComponent(subject)}&body=${encodeURIComponent(body)}`;
  };

  const onSubmit = async (workspace: Workspace) => {
    // the workspace opened last is a best-effort preference: the new workspace opens whether nerve saves it or not
    await updateUserProfile({ last_workspace_id: workspace.id }).catch(() => undefined);
    await navigate(`/${workspace.slug}`);
  };
````
````new web/apps/web/app/(all)/create-workspace/page.tsx
  const isWorkspaceCreationDisabled = config?.workspace_creation_enabled === false;
````
````old web/apps/web/app/(all)/create-workspace/page.tsx
                </Button>
                <a href={getMailtoHref()} className={getButtonStyling("secondary", "base")}>
                  {t("workspace_creation.errors.creation_disabled.request_button")}
                </a>
````
````new web/apps/web/app/(all)/create-workspace/page.tsx
                </Button>
````
````old web/apps/web/app/(all)/create-workspace/page.tsx
                  onSubmit={onSubmit}
````
````new web/apps/web/app/(all)/create-workspace/page.tsx
                  onCreated={(workspace) => void openWorkspace(workspace)}
````

`web/apps/web/core/components/workspace/create-workspace-form.tsx`（修改，13 处）：

````old web/apps/web/core/components/workspace/create-workspace-form.tsx
import { useEffect, useState } from "react";
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
import { useEffect } from "react";
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { Workspace, WorkspaceCreate } from "@nerve/api-client";
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
import type { Workspace } from "@nerve/api-client";
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
import { validateWorkspaceName, validateSlug } from "@nerve/utils";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useNavigate } from "react-router";
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
import { validateWorkspaceName } from "@nerve/utils";
import { useNavigate } from "react-router";
// local imports
import { slugFrom, useCreateWorkspace, type CreationForm } from "./use-create-workspace";
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
  onSubmit?: (res: Workspace) => Promise<void>;
  defaultValues: Pick<WorkspaceCreate, "name" | "slug" | "organization_size">;
  setDefaultValues: Dispatch<SetStateAction<Pick<WorkspaceCreate, "name" | "slug" | "organization_size">>>;
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
  /** What the page does with the workspace created (in the session it was created in). */
  onCreated: (workspace: Workspace) => void;
  defaultValues: CreationForm;
  setDefaultValues: Dispatch<SetStateAction<CreationForm>>;
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
  const { onSubmit, defaultValues, setDefaultValues } = props;
  // states
  const [slugError, setSlugError] = useState(false);
  const [invalidSlug, setInvalidSlug] = useState(false);
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
  const { onCreated, defaultValues, setDefaultValues } = props;
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
  const { createWorkspace, checkWorkspaceSlug } = useWorkspace();
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
  const create = useCreateWorkspace();
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
    getValues,
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
    getValues,
    setError,
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
  } = useForm<WorkspaceCreate>({ defaultValues, mode: "onChange" });
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
  } = useForm<CreationForm>({ defaultValues, mode: "onChange" });
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
  const handleCreateWorkspace = async (formData: WorkspaceCreate) => {
    try {
      const { available } = await checkWorkspaceSlug(formData.slug);
      if (available) {
        setSlugError(false);
        try {
          const workspaceResponse = await createWorkspace(formData);
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: t("workspace_creation.toast.success.title"),
            message: t("workspace_creation.toast.success.message"),
          });

          if (onSubmit) await onSubmit(workspaceResponse);
        } catch {
          setToast({
            type: TOAST_TYPE.ERROR,
            title: t("workspace_creation.toast.error.title"),
            message: t("workspace_creation.toast.error.message"),
          });
        }
      } else {
        setSlugError(true);
      }
    } catch {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("workspace_creation.toast.error.title"),
        message: t("workspace_creation.toast.error.message"),
      });
    }
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
  const handleCreateWorkspace = async (formData: CreationForm) => {
    const created = await create(formData, setError);
    if (created) onCreated(created);
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
                        setValue("slug", e.target.value.toLocaleLowerCase().trim().replace(/ /g, "-"), {
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
                        setValue("slug", slugFrom(e.target.value.trim()), {
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
              <Field name="workspaceUrl" invalid={invalidSlug || Boolean(errors.slug)}>
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
              <Field name="workspaceUrl" invalid={Boolean(errors.slug)}>
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
                    value={value.toLocaleLowerCase().trim().replace(/ /g, "-")}
                    onChange={(e) => {
                      const validation = validateSlug(e.target.value);
                      if (validation === true) setInvalidSlug(false);
                      else setInvalidSlug(true);
                      onChange(e.target.value.toLowerCase());
                    }}
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
                    value={value}
                    onChange={(e) => onChange(slugFrom(e.target.value))}
````
````old web/apps/web/core/components/workspace/create-workspace-form.tsx
          />
          {slugError && (
            <p className="-mt-3 text-13 text-danger-primary">
              {t("workspace_creation.errors.validation.url_already_taken")}
            </p>
          )}
          {invalidSlug && (
            <p className="text-13 text-danger-primary">{t("workspace_creation.errors.validation.url_alphanumeric")}</p>
          )}
````
````new web/apps/web/core/components/workspace/create-workspace-form.tsx
          />
````

`web/packages/i18n/src/locales/en/workspace.json`（修改，3 处）：

````old web/packages/i18n/src/locales/en/workspace.json
        "title": "Only your instance admin can create workspaces",
        "description": "If you know your instance admin's email address, click the button below to get in touch with them.",
        "request_button": "Request instance admin"
````
````new web/packages/i18n/src/locales/en/workspace.json
        "title": "Creating workspaces is switched off",
        "description": "Ask a workspace's admin for an invitation link."
````
````old web/packages/i18n/src/locales/en/workspace.json
        "url_alphanumeric": "URLs can contain only ('-') and alphanumeric characters.",
````
````new web/packages/i18n/src/locales/en/workspace.json
        "url_alphanumeric": "URLs can contain only lower-case letters, digits, '-' and '_'.",
````
````old web/packages/i18n/src/locales/en/workspace.json
        "url_already_taken": "Workspace URL is already taken!"
      }
    },
    "request_email": {
      "subject": "Requesting a new workspace",
      "body": "Hi instance admin(s),\n\nPlease create a new workspace with the URL [/workspace-name] for [purpose of creating the workspace].\n\nThanks,\n{firstName} {lastName}\n{email}"
````
````new web/packages/i18n/src/locales/en/workspace.json
        "url_already_taken": "Workspace URL is already taken!",
        "url_reserved": "This URL is reserved: choose another."
      }
````

`web/packages/i18n/src/locales/zh-CN/workspace.json`（修改，3 处）：

````old web/packages/i18n/src/locales/zh-CN/workspace.json
        "title": "只有您的实例管理员可以创建工作区",
        "description": "如果您知道实例管理员的电子邮件地址，请点击下方按钮与他们联系。",
        "request_button": "请求实例管理员"
````
````new web/packages/i18n/src/locales/zh-CN/workspace.json
        "title": "创建工作区已关闭",
        "description": "请向工作区的管理员要一个邀请链接。"
````
````old web/packages/i18n/src/locales/zh-CN/workspace.json
        "url_alphanumeric": "URL 只能包含 ('-') 和字母数字字符。",
````
````new web/packages/i18n/src/locales/zh-CN/workspace.json
        "url_alphanumeric": "URL 只能包含小写字母、数字、'-' 和 '_'。",
````
````old web/packages/i18n/src/locales/zh-CN/workspace.json
        "url_already_taken": "工作区 URL 已被占用！"
      }
    },
    "request_email": {
      "subject": "请求新工作区",
      "body": "您好，实例管理员：\n\n请为 [创建工作区的目的] 创建一个 URL 为 [/workspace-name] 的新工作区。\n\n谢谢，\n{firstName} {lastName}\n{email}"
````
````new web/packages/i18n/src/locales/zh-CN/workspace.json
        "url_already_taken": "工作区 URL 已被占用！",
        "url_reserved": "这个 URL 是保留的，请换一个。"
      }
````

- [ ] **Step 3: 静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/hooks/use-open-workspace.test.ts"
````
````new .oxlintrc.json
        "web/apps/web/core/hooks/use-open-workspace.test.ts",
        "web/apps/web/core/components/workspace/use-create-workspace.ts",
        "web/apps/web/core/components/workspace/use-create-workspace.test.ts"
````

- [ ] **Step 4: 端到端**

`e2e/stories/workspace/w1-create-workspace.spec.ts`（修改，3 处）：

````old e2e/stories/workspace/w1-create-workspace.spec.ts
import { countWorkspaces, expectNoWorkspaceAdded, expectWorkspaceCreated } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
````
````new e2e/stories/workspace/w1-create-workspace.spec.ts
import {
  countWorkspaces,
  expectNoWorkspaceAdded,
  expectWorkspaceCreated,
  lastWorkspaceOf,
} from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { answerTo, registerOnboarded, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser } from "../../fixtures/workspace-pages";
````
````old e2e/stories/workspace/w1-create-workspace.spec.ts
// W1, create a workspace (M3 design 2, 3.10, 3.11). The page version comes
// with the onboarding and /create-workspace pages.
````
````new e2e/stories/workspace/w1-create-workspace.spec.ts
// W1, create a workspace (M3 design 2, 3.10, 3.11).
````
````old e2e/stories/workspace/w1-create-workspace.spec.ts
  expect(await expectWorkspaceCreated(db, email, body)).toBe(created.id);
});

````
````new e2e/stories/workspace/w1-create-workspace.spec.ts
  expect(await expectWorkspaceCreated(db, email, body)).toBe(created.id);
});

test("W1 (page): at /create-workspace an onboarded account is told under the field of a taken, a reserved or an invalid slug, and nothing is created; then it creates a workspace with the slug the field shows, which opens, written as the one opened last; on a nerve with creation switched off the page says so", async ({
  api,
  browser,
  db,
  nerveWith,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const tokens = await registerOnboarded(api, email);
  const taken = slugFor(testInfo, "taken");
  await createWorkspace(api, tokens.access_token, { name: "Taken", slug: taken });
  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  await page.goto("/create-workspace");
  await page.locator("#workspaceName").fill("Acme Two");
  await page.getByRole("button", { name: "Select a range" }).click();
  await page.getByRole("option", { name: "2-10" }).click();
  const url = page.locator("#workspaceUrl");
  // The name gives the slug.
  await expect(url).toHaveValue("acme-two");

  // nerve's check of the slug decides: a refused one is said under the field, and the form is not sent.
  const refused = async (slug: string, message: string) => {
    await url.fill(slug);
    const checked = await answerTo(page, "GET", `/api/v0/workspace-slugs/${encodeURIComponent(slug)}`, () =>
      page.getByRole("button", { name: "Create workspace" }).click()
    );
    expect(checked.status()).toBe(200);
    await expect(page.getByText(message)).toBeVisible();
  };
  await refused(taken, "Workspace URL is already taken!");
  await refused("settings", "This URL is reserved: choose another.");
  await refused("café", "URLs can contain only lower-case letters, digits, '-' and '_'.");

  // The field shows the slug it sends: in lower case, a space as "-".
  const slug = slugFor(testInfo);
  await url.fill(slug.replace("-", " ").toUpperCase());
  await expect(url).toHaveValue(slug);
  const sent = await sentTo(page, "POST", "/api/v0/workspaces", () =>
    page.getByRole("button", { name: "Create workspace" }).click()
  );
  expect([sent.answer.status(), sent.body]).toEqual([201, { name: "Acme Two", slug, organization_size: "2-10" }]);
  await expect(page).toHaveURL(`/${slug}`);
  await expect(page.getByText("Workspace created successfully")).toBeVisible();
  const id = await expectWorkspaceCreated(db, email, {
    name: "Acme Two",
    slug,
    organization_size: "2-10",
    timezone: "UTC",
  });
  await expect.poll(() => lastWorkspaceOf(db, email)).toBe(id);
  expect(watch.apiRequests.filter((request) => request === "POST /api/v0/workspaces")).toHaveLength(1);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  // The new workspace's home.
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });

  // A nerve with creation switched off, on the same database: the page says so, and offers no form, nor a letter to
  // an instance admin, whom nerve does not have.
  const closed = await nerveWith({ NERVE_WORKSPACE__CREATION_ENABLED: "false" });
  const closedTokens = await registerOnboarded(createApi(closed.baseURL), emailFor(testInfo, "closed"));
  const there = await anotherBrowser(browser, closed.baseURL, closedTokens);
  await there.page.goto("/create-workspace");
  await expect(there.page.getByText("Creating workspaces is switched off", { exact: true })).toBeVisible();
  await expect(there.page.getByText("Ask a workspace's admin for an invitation link.")).toBeVisible();
  await expect(there.page.locator("#workspaceName")).toHaveCount(0);
  await expect(there.page.locator('a[href^="mailto:"]')).toHaveCount(0);
  await there.close();
});

````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、3 个例外，没有命中；web 的 oxlint 359 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 83 个全部通过。

- [ ] **Step 6: 提交**

```bash
git add .oxlintrc.json e2e/stories/workspace/w1-create-workspace.spec.ts 'web/apps/web/app/(all)/create-workspace/page.tsx' web/apps/web/core/components/workspace/create-workspace-form.tsx web/apps/web/core/components/workspace/use-create-workspace.test.ts web/apps/web/core/components/workspace/use-create-workspace.ts web/packages/i18n/src/locales/en/workspace.json web/packages/i18n/src/locales/zh-CN/workspace.json
```
```bash
git commit -m "feat(M3/P9): workspaces are created through one hook: nerve's slug check decides, its refusals land on the fields

useCreateWorkspace asks nerve whether the slug can name a workspace and
sends nothing when it cannot, saying why under the slug; it sends the
form's WorkspaceCreate, no size when none was chosen, shows a refusal
under the fields it names or as nerve's reason, and follows the creation
only in its session. The form drops its own slug check, and its slug
field shows what it sends. /create-workspace opens the workspace created
through useOpenWorkspace; with creation switched off it says so, as the
onboarding does, without Plane's letter to an instance admin nerve does
not have. W1's /create-workspace page version.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p9.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T5.1` | 创建不等 nerve 对 slug 的检查（P8a 的 t5-slug-ignored） | `use-create-workspace.test.ts`、故事 W1 | vitest；端到端 |
| `T5.2` | 创建工作区不论会话都跟进 | `use-create-workspace.test.ts` | vitest |
| `T5.3` | 没有选规模的表单发出规模为 `null` | `use-create-workspace.test.ts` | vitest |
| `T5.4` | nerve 对字段的拒绝显示在提示里 | `use-create-workspace.test.ts` | vitest |
| `T5.5` | slug 字段显示的和发出的不是同一个（Plane） | 故事 W1 | 端到端 |
| `T5.6` | 工作区建好之后停在表单上 | oxlint（`check:lint`）、故事 W1 | 静态；端到端 |
| `T5.7` | 检查之后才被占用的 slug 显示在提示里，不在字段下方 | `use-create-workspace.test.ts` | vitest |
| `T5.8` | nerve 对 slug 的说明哪里都不显示 | `use-create-workspace.test.ts`、故事 W1 | vitest；端到端 |
| `T5.9` | slug 字段保留输入的空格 | `use-create-workspace.test.ts`、故事 W1 | vitest；端到端 |
| `T5.10` | 创建工作区的 hook（范围内）用非空断言 | oxlint（`check:lint`） | 静态 |

---

### Task 6: 新手引导：按工作区列表决定步骤，创建一步共用 hook，邀请一步用成员页的表单；W1 的页面版本（二）

**Files:**
- Create: `web/apps/web/core/components/onboarding/onboarding-place.test.ts`、`web/apps/web/core/components/onboarding/onboarding-place.ts`、`web/apps/web/core/components/onboarding/root.test.tsx`、`web/apps/web/core/components/onboarding/steps/profile/root.test.tsx`、`web/apps/web/core/components/onboarding/steps/team/links.tsx`
- Modify: `.oxlintrc.json`、`e2e/stories/smoke/s2-web-app.spec.ts`、`e2e/stories/workspace/w1-create-workspace.spec.ts`、`web/apps/web/app/(all)/onboarding/page.tsx`、`web/apps/web/core/components/onboarding/root.tsx`、`web/apps/web/core/components/onboarding/steps/profile/root.tsx`、`web/apps/web/core/components/onboarding/steps/root.tsx`、`web/apps/web/core/components/onboarding/steps/team/root.tsx`、`web/apps/web/core/components/onboarding/steps/workspace/create.tsx`、`web/apps/web/core/components/onboarding/steps/workspace/index.ts`、`web/apps/web/core/store/user/profile.store.ts`、`web/packages/constants/src/workspace.ts`、`web/packages/i18n/src/locales/en/common.json`、`web/packages/i18n/src/locales/en/workspace.json`、`web/packages/i18n/src/locales/zh-CN/common.json`、`web/packages/i18n/src/locales/zh-CN/workspace.json`、`web/packages/utils/src/validation.ts`
- Delete: `web/apps/web/core/components/onboarding/steps/workspace/root.tsx`

**Interfaces:**
- Produces（spec 2.6；M3 设计 2 的 W1，7.1，7.4；决定 2；P8a 的 P7、P21 和评审的 M4、M7）：
  - `core/components/onboarding/onboarding-place.ts`：`OnboardingPlace`（`kind` 是 `EOnboardingSteps` 的三步，邀请一步带它邀请到的工作区）；`resumedPlace(profile, workspaces)`：资料、创建都已完成、还没邀请、创建的工作区（写成上次打开的那个）仍在列表中时，接着在邀请一步（邀请到它，不是列表的第一个，M4）；资料已完成、没有创建、没有工作区时，在创建一步；其余在资料一步。`afterProfile(workspaces)`：有工作区的人（先接受了邀请的）资料一步之后就完成（`"finish"`），没有的去创建（`"create"`，决定 2）。原来接着做时，被邀请的人也停在创建一步。
  - `core/components/onboarding/root.tsx`（整个文件）：`OnboardingRoot` 经 `useSessionSWR(["WORKSPACES"], …)` 取调用者的工作区列表（M7：原来新手引导读一份没人取过的列表），列表到之前显示加载，取不到时显示 `SessionUnavailable`；`OnboardingSteps` 按 `onboarding-place.ts` 决定步骤，资料一步之后、创建之后、完成时的修改都经 `followInSession`，拒绝按 `code` 提示；创建的工作区与它的步骤一起写成上次打开的（`last_workspace_id`）；只给创建者自己用的工作区（规模选"Just myself"）没有邀请一步。
  - `steps/root.tsx`（整个文件）：按 `place` 显示，收 `onNamed`、`onCreated(workspace, alone)`、`onDone`；`steps/profile/root.tsx` 的 `ProfileSetupStep` 收 `onDone`（原来的 `handleStepChange(step, skipInvites?)` 删除），只在发出名字的会话里调它：名字的保存经 `followInSession`，`done` 在保存成功时调 `onDone()`（预检 M2：原来在 `await` 之后直接交给根，另一个标签页换了账户之后，根的完成或下一步发生在新会话里，旧 store 的请求被 `SessionChangedError` 截断，"出错了"显示在那个账户的页面上）；资料一步自己的提示仍是 M2 的（spec 第 5 节 P11）。`steps/workspace/root.tsx`（"创建或加入"的包装）删除，`index.ts` 改为导出创建一步。
  - `steps/workspace/create.tsx`：创建一步改用 Task 5 的 `useCreateWorkspace`、`CreationForm`、`slugFrom`（它自己的 slug 检查、`validateSlug` 和 `try/catch` 删除；slug 框的红边原来读的是 `errors.name`，改为 `errors.slug`）。
  - `steps/team/root.tsx`（整个文件）：邀请一步用成员页的表单（`useWorkspaceInvitationActions` 和 `InvitationFields`：一行起，角色是 Guest、Member、Admin 的编号，行的错误来自 nerve），邀请到这一步的工作区（原来 Plane 的三行表单，邀请到列表的第一个工作区，拒绝读 `err.error`，P21）；表单的 `onSent` 交来的邀请就是 `steps/team/links.tsx` 列出的：每个邀请的链接和"复制链接"（Task 3 的 `useCopyInvitationLink`），再"继续"（v0 不发邮件）。
  - `profile.store.ts`：`finishUserOnboarding` 不再把列表中的第一个工作区写成上次打开的。
  - `@nerve/utils` 的 `validateSlug`、`SLUG_REGEX` 删除（不再有读者）。`@nerve/constants` 的 `ROLE_DETAILS` 的 `i18n_description` 和两种语言 `common.json` 的 `role_details.{guest,member,admin}.description` 删除：它们唯一的读者是旧的邀请一步（预检 L2）。i18n：`onboarding.workspace.creation_disabled` 改成 nerve 的情形（创建已关闭：请工作区的管理员给一个邀请链接，P7）；`onboarding.invite.links.*` 新增；`onboarding.invite.role`、`not_an_email` 和 `workspace_creation.toast.error` 删除（en、zh-CN）。
  - 静态检查（总体设计 7.7）：`.oxlintrc.json` 的 `no-restricted-imports` 的范围加上 `onboarding/root.tsx`（它取工作区列表）；非空断言的范围加上 `onboarding-place.ts` 和它的测试、`onboarding/root.test.tsx`、`steps/profile/root.test.tsx`、`steps/team/links.tsx`。

**Tests:**
- vitest：`onboarding-place.test.ts`：`resumedPlace` 的 `resumes $account at $place.kind`、`afterProfile` 的 `leads $account to $next`（`it.each`）。`root.test.tsx`（`OnboardingRoot`）：`lists the caller's workspaces, and shows no step before nerve has`；`ends the onboarding of one who has a workspace with the profile step`；`writes the profile step done for one who has none, whom the creation step follows`；`writes a workspace created $workspace as the one opened last, with its step`；`says why nerve refused a step's change`；`says nothing when it $settles after another tab moved this one to another account`（一步的资料修改、完成引导各两行，预检 M1）。`steps/profile/root.test.tsx`（`ProfileSetupStep`；`react-hook-form` 的 `useForm` 换成留下提交函数的替身）：`sends the names, then hands the onboarding on`；`stays when nerve refuses the names`；`hands nothing on when nerve has the names after another tab moved this one to another account`。
- 端到端：`W1 (page): a newcomer's onboarding creates a workspace after the profile step, written as the one opened last; invites to it and shows the link to copy; then lands in it`（请求体经 `sentTo`：创建、写进资料的步骤和上次打开的工作区、邀请、完成；复制的链接）；`W1 (page): a newcomer who puts off the invitations at their step lands in the workspace he created, having invited no one`（第 2 节 W1 的"点跳过"）；`W1 (page): one who joined a workspace before onboarding is done after the profile step; one who comes back after creating a workspace invites to that one, whatever comes first in his list`；`W1 (page): on a nerve with creation switched off, a newcomer's onboarding says to ask for an invitation link after the profile step`。S2 加 `S2: a newcomer opens /, which sends him to the onboarding: it asks for his workspaces, as the app does, and no more`（像其余的 S2 测试，安静的控制台之后再读一次请求：之后没有别的请求）。

- [ ] **Step 1: 新手引导的位置和根**

`web/apps/web/core/components/onboarding/onboarding-place.test.ts`（新文件，68 行）：

````file web/apps/web/core/components/onboarding/onboarding-place.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import type { OnboardingSteps, Workspace } from "@nerve/api-client";
import { EOnboardingSteps } from "@nerve/types";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { afterProfile, resumedPlace, type OnboardingPlace } from "./onboarding-place";

// Where the onboarding resumes and where its profile step leads (M3 design 7.4, decision 2), of the steps nerve keeps
// done and the caller's workspaces.

const none: OnboardingSteps = {
  profile_complete: false,
  workspace_create: false,
  workspace_invite: false,
  workspace_join: false,
};
const named = { ...none, profile_complete: true };
const created = { ...named, workspace_create: true };
// alpha comes first in the list, as nerve orders it by name: zeta is the one the caller created
const alpha = workspaceOf("alpha");
const zeta = workspaceOf("zeta", { role: 20 });
const { PROFILE_SETUP, WORKSPACE_CREATE_OR_JOIN, INVITE_MEMBERS } = EOnboardingSteps;

describe("resumedPlace", () => {
  it.each<{ account: string; steps: OnboardingSteps; workspaces: Workspace[]; place: OnboardingPlace }>([
    { account: "a new account", steps: none, workspaces: [], place: { kind: PROFILE_SETUP } },
    {
      account: "one named, with no workspace",
      steps: named,
      workspaces: [],
      place: { kind: WORKSPACE_CREATE_OR_JOIN },
    },
    { account: "one named, invited since", steps: named, workspaces: [alpha], place: { kind: PROFILE_SETUP } },
    {
      account: "one that created a workspace and invited no one",
      steps: created,
      workspaces: [alpha, zeta],
      place: { kind: INVITE_MEMBERS, workspace: zeta },
    },
    {
      account: "one that created a workspace and invited",
      steps: { ...created, workspace_invite: true },
      workspaces: [alpha, zeta],
      place: { kind: PROFILE_SETUP },
    },
    {
      account: "one whose workspace created is gone",
      steps: created,
      workspaces: [alpha],
      place: { kind: PROFILE_SETUP },
    },
  ])("resumes $account at $place.kind", ({ steps, workspaces, place }) => {
    expect(resumedPlace({ onboarding_step: steps, last_workspace_id: zeta.id }, workspaces)).toEqual(place);
  });
});

describe("afterProfile", () => {
  it.each<{ account: string; workspaces: Workspace[]; next: "finish" | "create" }>([
    { account: "one with no workspace", workspaces: [], next: "create" },
    { account: "one with a workspace", workspaces: [alpha], next: "finish" },
  ])("leads $account to $next", ({ workspaces, next }) => {
    expect(afterProfile(workspaces)).toBe(next);
  });
});
````

`web/apps/web/core/components/onboarding/onboarding-place.ts`（新文件，39 行）：

````file web/apps/web/core/components/onboarding/onboarding-place.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Profile, Workspace } from "@nerve/api-client";
import { EOnboardingSteps } from "@nerve/types";

/** Where the onboarding is (M3 design 7.4): a step; the invitation step's with the workspace it invites to. */
export type OnboardingPlace =
  | { kind: EOnboardingSteps.PROFILE_SETUP }
  | { kind: EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN }
  | { kind: EOnboardingSteps.INVITE_MEMBERS; workspace: Workspace };

/**
 * Where the onboarding of profile resumes, by the steps nerve keeps done (M3 design 7.4): at the invitation step, for
 * the workspace it created (the one it opened last, written with the step), while the caller has it and has invited
 * no one yet; at the creation's, for one who has no workspace and created none; else at the profile step, which is
 * the last for one who has a workspace (afterProfile).
 */
export function resumedPlace(
  profile: Pick<Profile, "onboarding_step" | "last_workspace_id">,
  workspaces: readonly Workspace[]
): OnboardingPlace {
  const steps = profile.onboarding_step;
  const created = workspaces.find((workspace) => workspace.id === profile.last_workspace_id);
  if (steps.profile_complete && steps.workspace_create && !steps.workspace_invite && created)
    return { kind: EOnboardingSteps.INVITE_MEMBERS, workspace: created };
  if (steps.profile_complete && !steps.workspace_create && workspaces.length === 0)
    return { kind: EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN };
  return { kind: EOnboardingSteps.PROFILE_SETUP };
}

/**
 * What follows the profile step (M3 design 7.4, decision 2): the end, for one who has a workspace (the invited, who
 * accepted first); else the creation of one.
 */
export const afterProfile = (workspaces: readonly Workspace[]): "finish" | "create" =>
  workspaces.length > 0 ? "finish" : "create";
````

`web/apps/web/core/components/onboarding/root.test.tsx`（新文件，137 行）：

````file web/apps/web/core/components/onboarding/root.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Workspace } from "@nerve/api-client";
import { emptyStores, stores } from "@/hooks/store/fake-store-hooks";
import { heldChange, lateSettlings, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { fetchHanded, handed } from "@/lib/fake-session-swr";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { OnboardingRoot } from "./root";

// What the onboarding writes as its steps end (M3 design 7.4, 7.1): the root renders on the server, its steps a
// stand-in that keeps what it was given, and the test ends a step as the step would; the profile's store is a
// stand-in too, which nerve answers when the test says. The workspaces are fake-store-hooks.ts's, the fetch
// fake-session-swr.ts's, the session fake-tab.ts's.

/** How the steps tell the root they are done (OnboardingStepRoot's props). */
type Ends = { onNamed: () => void; onCreated: (workspace: Workspace, alone: boolean) => void; onDone: () => void };

const page = vi.hoisted(
  (): {
    ends: Ends | undefined;
    updateUserProfile: ReturnType<typeof vi.fn>;
    finishUserOnboarding: ReturnType<typeof vi.fn>;
  } => ({
    ends: undefined,
    updateUserProfile: vi.fn(),
    finishUserOnboarding: vi.fn(),
  })
);
vi.mock("./steps", () => ({
  OnboardingStepRoot: (props: Ends) => {
    page.ends = props;
    return null;
  },
}));
vi.mock("./header", () => ({ OnboardingHeader: () => null }));
vi.mock("@/hooks/store/user", () => ({
  useUserProfile: () => ({
    data: undefined,
    updateUserProfile: page.updateUserProfile,
    finishUserOnboarding: page.finishUserOnboarding,
  }),
}));
vi.mock("@/hooks/store/use-workspace", () => import("@/hooks/store/fake-store-hooks"));
vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const alpha = workspaceOf("alpha");
const zeta = workspaceOf("zeta", { role: 20 });
/** Each change the onboarding follows, by the profile's store change: a step's, and the end's. */
const changes = [
  { change: "a step's change", store: page.updateUserProfile, make: () => opened([]).onNamed() },
  { change: "the end of the onboarding", store: page.finishUserOnboarding, make: () => opened([alpha]).onDone() },
];

/** Opens the onboarding of one whose workspaces nerve listed as workspaces; gives how its steps end. */
function opened(workspaces: Workspace[]): Ends {
  stores.workspaces = workspaces;
  renderToStaticMarkup(<OnboardingRoot />);
  if (!page.ends) throw new Error("the onboarding showed no step");
  return page.ends;
}

beforeEach(() => {
  signedIn();
  emptyStores();
  handed.length = 0;
  page.ends = undefined;
  page.updateUserProfile.mockReset();
  page.updateUserProfile.mockResolvedValue({});
  page.finishUserOnboarding.mockReset();
  page.finishUserOnboarding.mockResolvedValue(undefined);
  toasts.length = 0;
});

describe("OnboardingRoot", () => {
  it("lists the caller's workspaces, and shows no step before nerve has", async () => {
    renderToStaticMarkup(<OnboardingRoot />);
    expect(page.ends).toBeUndefined();
    await fetchHanded();
    expect([handed.map(([fetch]) => fetch), stores.fetched]).toEqual([[["WORKSPACES"]], ["the workspaces"]]);
  });

  it("ends the onboarding of one who has a workspace with the profile step", () => {
    opened([alpha]).onNamed();
    expect([page.finishUserOnboarding.mock.calls, page.updateUserProfile.mock.calls]).toEqual([[[]], []]);
  });

  it("writes the profile step done for one who has none, whom the creation step follows", () => {
    opened([]).onNamed();
    expect([page.finishUserOnboarding.mock.calls, page.updateUserProfile.mock.calls]).toEqual([
      [],
      [[{ onboarding_step: { profile_complete: true } }]],
    ]);
  });

  it.each([
    { workspace: "for others too", alone: false, finished: [] },
    { workspace: "for its creator alone", alone: true, finished: [[]] },
  ])("writes a workspace created $workspace as the one opened last, with its step", ({ alone, finished }) => {
    opened([alpha, zeta]).onCreated(zeta, alone);
    expect([page.updateUserProfile.mock.calls, page.finishUserOnboarding.mock.calls]).toEqual([
      [[{ onboarding_step: { workspace_create: true }, last_workspace_id: zeta.id }]],
      finished,
    ]);
  });

  it("says why nerve refused a step's change", async () => {
    page.updateUserProfile.mockRejectedValueOnce(refusal(422, "validation_failed"));
    opened([]).onNamed();
    await pageSettled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.validation_failed" }]);
  });

  describe.each(changes)("$change", ({ store, make }) => {
    it.each(lateSettlings)(
      "says nothing when it $settles after another tab moved this one to another account",
      async ({ settle }) => {
        const change = heldChange<undefined>();
        store.mockReturnValueOnce(change.sent);
        make();
        switchAccount();
        settle(change);
        await pageSettled();
        expect(toasts).toEqual([]);
      }
    );
  });
});
````

`web/apps/web/core/components/onboarding/root.tsx`（整个文件，97 行）：

````whole web/apps/web/core/components/onboarding/root.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import type { ProfileUpdate, Workspace } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { EOnboardingSteps } from "@nerve/types";
// components
import { SessionUnavailable } from "@/components/account/session-unavailable";
import { LogoSpinner } from "@/components/common/logo-spinner";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUserProfile } from "@/hooks/store/user";
// lib
import { errorMessageKey } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";
import { useSessionSWR } from "@/lib/use-session-swr";
// local components
import { OnboardingHeader } from "./header";
import { afterProfile, resumedPlace, type OnboardingPlace } from "./onboarding-place";
import { OnboardingStepRoot } from "./steps";

type Props = {
  /** The caller's workspaces, as nerve listed them. */
  workspaces: Workspace[];
};

const OnboardingSteps = observer(function OnboardingSteps({ workspaces }: Props) {
  const { t } = useTranslation();
  // store hooks
  const { data: profile, updateUserProfile, finishUserOnboarding } = useUserProfile();
  // where the onboarding is: where it was left, as it opens
  const [place, setPlace] = useState<OnboardingPlace>(() =>
    profile ? resumedPlace(profile, workspaces) : { kind: EOnboardingSteps.PROFILE_SETUP }
  );

  // a change of the profile; nerve's refusal says why, in the session it was sent in alone (M3 design 7.1)
  const failed = (error: unknown) =>
    setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) });
  const change = (data: ProfileUpdate) => void followInSession(() => updateUserProfile(data), { failed });
  const finish = () => void followInSession(() => finishUserOnboarding(), { failed });

  // one who has a workspace is done after the profile step; one who has none creates one (M3 design 7.4)
  const named = () => {
    if (afterProfile(workspaces) === "finish") {
      finish();
      return;
    }
    change({ onboarding_step: { profile_complete: true } });
    setPlace({ kind: EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN });
  };

  // the workspace created is the one opened last, and the one the invitation step invites to; a workspace for its
  // creator alone has no invitation step
  const created = (workspace: Workspace, alone: boolean) => {
    change({ onboarding_step: { workspace_create: true }, last_workspace_id: workspace.id });
    if (alone) finish();
    else setPlace({ kind: EOnboardingSteps.INVITE_MEMBERS, workspace });
  };

  return (
    <div className="flex h-full flex-col">
      {/* Header with progress: its one way back is from the creation to the profile step */}
      <OnboardingHeader
        currentStep={place.kind}
        updateCurrentStep={() => setPlace({ kind: EOnboardingSteps.PROFILE_SETUP })}
      />

      {/* Main content area */}
      <OnboardingStepRoot place={place} onNamed={named} onCreated={created} onDone={finish} />
    </div>
  );
});

/**
 * The onboarding (M3 design 7.4): its steps once nerve has listed the caller's workspaces, which decide them. The list
 * is the session's, the one the landing reads once the onboarding is done.
 */
export const OnboardingRoot = observer(function OnboardingRoot() {
  // store hooks
  const { workspaces, fetchWorkspaces } = useWorkspace();
  const listed = useSessionSWR(["WORKSPACES"], () => fetchWorkspaces());

  if (workspaces) return <OnboardingSteps workspaces={workspaces} />;
  if (listed.error) return <SessionUnavailable autoRetry={false} onRetry={() => void listed.mutate()} />;
  return (
    <div className="grid h-full w-full place-items-center">
      <LogoSpinner />
    </div>
  );
});
````

- [ ] **Step 2: 步骤、邀请的链接、store、工具函数和文案**

`web/apps/web/app/(all)/onboarding/page.tsx`（修改，1 处）：

````old web/apps/web/app/(all)/onboarding/page.tsx
// The page asks for nothing of the caller's workspaces when it opens (M2 design 3.1): P9 adds the fetch its steps
// decide by (M3 design 7.4).
````
````new web/apps/web/app/(all)/onboarding/page.tsx
// The onboarding lists the caller's workspaces, which decide its steps (OnboardingRoot, M3 design 7.4).
````

`web/apps/web/core/components/onboarding/steps/profile/root.test.tsx`（新文件，88 行）：

````file web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { ProfileSetupStep } from "./root";

// When the onboarding's profile step hands the onboarding on (M3 design 7.1, 7.4): the step renders on the server with
// a stand-in for its form (react-hook-form's useForm), which keeps the step's submit; the test submits names as the
// form would, and nerve answers them when the test says. Its session is fake-tab.ts's. What the step itself says of a
// refusal is M2's (M2 design 7.3).

/** The names the step's form gives its submit. */
type Names = { first_name: string; last_name: string };

const page = vi.hoisted(
  (): { submit: ((names: Names) => Promise<void>) | undefined; updateCurrentUser: ReturnType<typeof vi.fn> } => ({
    submit: undefined,
    updateCurrentUser: vi.fn(),
  })
);
/** The step hands the onboarding on: the root's part. */
const onDone = vi.fn<() => void>();
vi.mock("react-hook-form", () => ({
  useForm: () => ({
    handleSubmit: (submit: (names: Names) => Promise<void>) => {
      page.submit = submit;
      return () => undefined;
    },
    control: {},
    watch: () => "",
    setError: () => undefined,
    formState: { errors: {}, isSubmitting: false, isValid: true },
  }),
  Controller: () => null,
}));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => ({ data: { display_name: "ada" }, updateCurrentUser: page.updateCurrentUser }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Renders the step, and submits the names as its form would; settles once the step has. */
async function named() {
  renderToStaticMarkup(<ProfileSetupStep onDone={onDone} />);
  if (!page.submit) throw new Error("the step showed no form");
  await page.submit({ first_name: "Ada", last_name: "Lovelace" });
}

beforeEach(() => {
  signedIn();
  page.submit = undefined;
  page.updateCurrentUser.mockReset();
  page.updateCurrentUser.mockResolvedValue(undefined);
  onDone.mockReset();
});

describe("ProfileSetupStep", () => {
  it("sends the names, then hands the onboarding on", async () => {
    await named();
    expect(page.updateCurrentUser.mock.calls).toEqual([[{ first_name: "Ada", last_name: "Lovelace" }]]);
    expect(onDone.mock.calls).toEqual([[]]);
  });

  it("stays when nerve refuses the names", async () => {
    page.updateCurrentUser.mockRejectedValueOnce(
      refusal(422, "validation_failed", [{ field: "first_name", code: "required" }])
    );
    await named();
    expect(onDone.mock.calls).toEqual([]);
  });

  it("hands nothing on when nerve has the names after another tab moved this one to another account", async () => {
    const names = heldChange<undefined>();
    page.updateCurrentUser.mockReturnValueOnce(names.sent);
    const submitted = named();
    switchAccount();
    names.answer(undefined);
    await submitted;
    expect(onDone.mock.calls).toEqual([]);
  });
});
````

`web/apps/web/core/components/onboarding/steps/profile/root.tsx`（修改，5 处）：

````old web/apps/web/core/components/onboarding/steps/profile/root.tsx
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { EOnboardingSteps } from "@nerve/types";
````
````new web/apps/web/core/components/onboarding/steps/profile/root.tsx
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
````
````old web/apps/web/core/components/onboarding/steps/profile/root.tsx
import { errorMessageKey, fieldErrorKeys, needsErrorBanner } from "@/lib/error-messages";
````
````new web/apps/web/core/components/onboarding/steps/profile/root.tsx
import { errorMessageKey, fieldErrorKeys, needsErrorBanner } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";
````
````old web/apps/web/core/components/onboarding/steps/profile/root.tsx
  handleStepChange: (step: EOnboardingSteps, skipInvites?: boolean) => void;
````
````new web/apps/web/core/components/onboarding/steps/profile/root.tsx
  /** The step is done: nerve has the names. */
  onDone: () => void;
````
````old web/apps/web/core/components/onboarding/steps/profile/root.tsx
export const ProfileSetupStep = observer(function ProfileSetupStep({ handleStepChange }: Props) {
````
````new web/apps/web/core/components/onboarding/steps/profile/root.tsx
export const ProfileSetupStep = observer(function ProfileSetupStep({ onDone }: Props) {
````
````old web/apps/web/core/components/onboarding/steps/profile/root.tsx
    if (await handleSubmitUserDetail(formData)) handleStepChange(EOnboardingSteps.PROFILE_SETUP);
````
````new web/apps/web/core/components/onboarding/steps/profile/root.tsx
    // the onboarding goes on only in the session the names were sent in (M3 design 7.1); what the step says of a
    // refusal is M2's (M3/P9 spec 5: P11)
    await followInSession(() => handleSubmitUserDetail(formData), {
      done: (named) => {
        if (named) onDone();
      },
      // handleSubmitUserDetail says why itself, and answers false: it does not reject
      failed: () => undefined,
    });
````

`web/apps/web/core/components/onboarding/steps/root.tsx`（整个文件，62 行）：

````whole web/apps/web/core/components/onboarding/steps/root.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef } from "react";
// nerve imports
import type { Workspace } from "@nerve/api-client";
import { EOnboardingSteps } from "@nerve/types";
// local components
import type { OnboardingPlace } from "../onboarding-place";
import { ProfileSetupStep } from "./profile";
import { InviteTeamStep } from "./team";
import { WorkspaceCreateStep } from "./workspace";

type Props = {
  place: OnboardingPlace;
  /** The profile step is done. */
  onNamed: () => void;
  /** The creation step created workspace; alone when it is for its creator alone. */
  onCreated: (workspace: Workspace, alone: boolean) => void;
  /** The invitation step is done, or put off. */
  onDone: () => void;
};

function OnboardingStepContent({ place, onNamed, onCreated, onDone }: Props) {
  switch (place.kind) {
    case EOnboardingSteps.PROFILE_SETUP:
      return <ProfileSetupStep onDone={onNamed} />;
    case EOnboardingSteps.WORKSPACE_CREATE_OR_JOIN:
      return <WorkspaceCreateStep onCreated={onCreated} />;
    case EOnboardingSteps.INVITE_MEMBERS:
      return <InviteTeamStep workspace={place.workspace} onDone={onDone} />;
  }
}

export function OnboardingStepRoot(props: Props) {
  const { place } = props;
  // ref for the scrollable container
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  // scroll to top when step changes
  useEffect(() => {
    if (scrollContainerRef.current) {
      scrollContainerRef.current.scrollTo({
        top: 0,
        behavior: "smooth",
      });
    }
  }, [place.kind]);

  return (
    <div ref={scrollContainerRef} className="flex-1 overflow-y-auto">
      <div className="flex min-h-full items-center justify-center p-8">
        <div className="w-full max-w-[24rem]">
          <OnboardingStepContent {...props} />
        </div>
      </div>
    </div>
  );
}
````

`web/apps/web/core/components/onboarding/steps/team/links.tsx`（新文件，47 行）：

````file web/apps/web/core/components/onboarding/steps/team/links.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { WorkspaceInvitation } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
// hooks
import { useCopyInvitationLink } from "@/hooks/use-copy-invitation-link";
// local components
import { CommonOnboardingHeader } from "../common";

type Props = {
  /** The invitations nerve made of the step's form. */
  invitations: WorkspaceInvitation[];
  /** Ends the onboarding. */
  onDone: () => void;
};

/** The links of the invitations the step sent, each to copy (M3 design 7.4: v0 sends no email). */
export function InvitationLinks({ invitations, onDone }: Props) {
  const { t } = useTranslation();
  const copyLink = useCopyInvitationLink();

  return (
    <div className="flex flex-col gap-10">
      <CommonOnboardingHeader
        title={t("onboarding.invite.links.title")}
        description={t("onboarding.invite.links.description")}
      />
      <ul className="flex flex-col gap-3">
        {invitations.map((invitation) => (
          <li key={invitation.id} className="flex items-center justify-between gap-4 text-13">
            <span className="truncate text-secondary">{invitation.email}</span>
            <Button variant="secondary" size="sm" onClick={() => void copyLink(invitation)}>
              {t("common.actions.copy_link")}
            </Button>
          </li>
        ))}
      </ul>
      <Button variant="primary" size="xl" className="w-full" onClick={onDone}>
        {t("continue")}
      </Button>
    </div>
  );
}
````

`web/apps/web/core/components/onboarding/steps/team/root.tsx`（整个文件，80 行）：

````whole web/apps/web/core/components/onboarding/steps/team/root.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { AddOutline } from "@makeplane/propel/icons";
// nerve imports
import type { Workspace, WorkspaceInvitation } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { Spinner } from "@nerve/ui";
// components
import { InvitationFields } from "@/components/workspace/invite-modal/fields";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useWorkspaceInvitationActions } from "@/hooks/use-workspace-invitation";
// local components
import { CommonOnboardingHeader } from "../common";
import { InvitationLinks } from "./links";

type Props = {
  /** The workspace the onboarding created, which the step invites to (M3 design 7.4). */
  workspace: Workspace;
  /** Ends the onboarding. */
  onDone: () => void;
};

/**
 * The invitation step (M3 design 7.4): the members page's invitation form, for the workspace the onboarding created;
 * once nerve has the invitations, their links, which the caller gives the people invited (v0 sends no email).
 */
export const InviteTeamStep = observer(function InviteTeamStep(props: Props) {
  const { workspace, onDone } = props;
  const { t } = useTranslation();
  // states
  const [sent, setSent] = useState<WorkspaceInvitation[]>();
  // store hooks
  const {
    workspace: { inviteMembersToWorkspace },
  } = useMember();
  const { control, fields, formState, remove, onFormSubmit, appendField } = useWorkspaceInvitationActions({
    invite: (data) => inviteMembersToWorkspace(workspace.slug, data),
    onSent: setSent,
  });

  if (sent) return <InvitationLinks invitations={sent} onDone={onDone} />;
  return (
    <form
      className="flex flex-col gap-10"
      onSubmit={onFormSubmit}
      onKeyDown={(e) => {
        if (e.code === "Enter") e.preventDefault();
      }}
    >
      <CommonOnboardingHeader title={t("onboarding.invite.title")} description={t("onboarding.invite.description")} />
      <div className="w-full text-13">
        <InvitationFields fields={fields} control={control} formState={formState} remove={remove} />
        <button
          type="button"
          className="flex items-center gap-1.5 bg-transparent text-13 font-medium text-accent-primary outline-accent-strong"
          onClick={appendField}
        >
          <AddOutline className="h-4 w-4" />
          {t("onboarding.invite.add_another")}
        </button>
      </div>
      <div className="mx-auto flex w-full flex-col items-center justify-center gap-4 px-8 sm:px-2">
        <Button variant="primary" type="submit" size="xl" className="w-full" disabled={formState.isSubmitting}>
          {formState.isSubmitting ? <Spinner height="20px" width="20px" /> : t("continue")}
        </Button>
        <Button variant="ghost" size="xl" className="w-full" onClick={onDone}>
          {t("onboarding.invite.later")}
        </Button>
      </div>
    </form>
  );
});
````

`web/apps/web/core/components/onboarding/steps/workspace/create.tsx`（修改，16 处）：

````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx

import { useState } from "react";
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx

````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { User, WorkspaceCreate } from "@nerve/api-client";
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
import type { Workspace } from "@nerve/api-client";
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
import { cn, validateWorkspaceName, validateSlug } from "@nerve/utils";
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
import { cn, validateWorkspaceName } from "@nerve/utils";
// components
import { slugFrom, useCreateWorkspace, type CreationForm } from "@/components/workspace/use-create-workspace";
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
import { useInstance } from "@/hooks/store/use-instance";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUserProfile } from "@/hooks/store/user";
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
import { useInstance } from "@/hooks/store/use-instance";
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
  user: User | undefined;
  onComplete: (skipInvites?: boolean) => void;
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
  /** What the onboarding does with the workspace created; alone when it is for its creator alone. */
  onCreated: (workspace: Workspace, alone: boolean) => void;
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
export const WorkspaceCreateStep = observer(function WorkspaceCreateStep({ user, onComplete }: Props) {
  // states
  const [slugError, setSlugError] = useState(false);
  const [invalidSlug, setInvalidSlug] = useState(false);
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
export const WorkspaceCreateStep = observer(function WorkspaceCreateStep({ onCreated }: Props) {
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
  const { updateUserProfile } = useUserProfile();
  const { createWorkspace, fetchWorkspaces, checkWorkspaceSlug } = useWorkspace();
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
  const create = useCreateWorkspace();
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
    setValue,
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
    setValue,
    setError,
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
  } = useForm<WorkspaceCreate>({
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
  } = useForm<CreationForm>({
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
      slug: "",
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
      slug: "",
      organization_size: null,
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
  const handleCreateWorkspace = async (formData: WorkspaceCreate) => {
    if (isSubmitting) return;

    try {
      const { available } = await checkWorkspaceSlug(formData.slug);
      if (available) {
        setSlugError(false);
        try {
          const workspaceResponse = await createWorkspace(formData);
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: t("workspace_creation.toast.success.title"),
            message: t("workspace_creation.toast.success.message"),
          });
          await fetchWorkspaces();
          await completeStep(workspaceResponse.id);
          onComplete(formData.organization_size === "Just myself");
        } catch {
          setToast({
            type: TOAST_TYPE.ERROR,
            title: t("workspace_creation.toast.error.title"),
            message: t("workspace_creation.toast.error.message"),
          });
        }
      } else {
        setSlugError(true);
      }
    } catch {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("workspace_creation.toast.error.title"),
        message: t("workspace_creation.toast.error.message"),
      });
    }
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
  const handleCreateWorkspace = async (formData: CreationForm) => {
    const created = await create(formData, setError);
    if (created) onCreated(created, formData.organization_size === "Just myself");
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
  const completeStep = async (workspaceId: string) => {
    if (!user) return;
    // the workspace opened last is a best-effort preference: the onboarding goes on whether nerve saves it or not
    await updateUserProfile({ last_workspace_id: workspaceId }).catch(() => undefined);
  };

  const isButtonDisabled = !isValid || invalidSlug || isSubmitting;
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
  const isButtonDisabled = !isValid || isSubmitting;
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
                    setValue("slug", event.target.value.toLocaleLowerCase().trim().replace(/ /g, "-"), {
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
                    setValue("slug", slugFrom(event.target.value.trim()), {
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
                    "border-strong": !errors.name,
                    "border-danger-strong": errors.name,
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
                    "border-strong": !errors.slug,
                    "border-danger-strong": errors.slug,
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
                  value={value.toLocaleLowerCase().trim().replace(/ /g, "-")}
                  onChange={(e) => {
                    const validation = validateSlug(e.target.value);
                    if (validation === true) setInvalidSlug(false);
                    else setInvalidSlug(true);
                    onChange(e.target.value.toLowerCase());
                  }}
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
                  value={value}
                  onChange={(e) => onChange(slugFrom(e.target.value))}
````
````old web/apps/web/core/components/onboarding/steps/workspace/create.tsx
          <p className="text-13 text-tertiary">{t("workspace_creation.form.url.edit_slug")}</p>
          {slugError && (
            <p className="-mt-3 text-13 text-danger-primary">
              {t("workspace_creation.errors.validation.url_already_taken")}
            </p>
          )}
          {invalidSlug && (
            <p className="text-13 text-danger-primary">{t("workspace_creation.errors.validation.url_alphanumeric")}</p>
          )}
````
````new web/apps/web/core/components/onboarding/steps/workspace/create.tsx
          <p className="text-13 text-tertiary">{t("workspace_creation.form.url.edit_slug")}</p>
````

`web/apps/web/core/components/onboarding/steps/workspace/index.ts`（修改，1 处）：

````old web/apps/web/core/components/onboarding/steps/workspace/index.ts
export * from "./create";
export * from "./root";
````
````new web/apps/web/core/components/onboarding/steps/workspace/index.ts
export * from "./create";
````

`web/apps/web/core/components/onboarding/steps/workspace/root.tsx`（删除）：

````delete web/apps/web/core/components/onboarding/steps/workspace/root.tsx
````

`web/apps/web/core/store/user/profile.store.ts`（修改，2 处）：

````old web/apps/web/core/store/user/profile.store.ts
  finishUserOnboarding = async (): Promise<void> => {
    const firstWorkspace = this.store.workspaceRoot.workspaces?.[0];
````
````new web/apps/web/core/store/user/profile.store.ts
  finishUserOnboarding = async (): Promise<void> => {
````
````old web/apps/web/core/store/user/profile.store.ts
      is_onboarded: true,
      ...(firstWorkspace ? { last_workspace_id: firstWorkspace.id } : {}),
````
````new web/apps/web/core/store/user/profile.store.ts
      is_onboarded: true,
````

`web/packages/constants/src/workspace.ts`（修改，3 处）：

````old web/packages/constants/src/workspace.ts
    i18n_title: "role_details.guest.title",
    i18n_description: "role_details.guest.description",
````
````new web/packages/constants/src/workspace.ts
    i18n_title: "role_details.guest.title",
````
````old web/packages/constants/src/workspace.ts
    i18n_title: "role_details.member.title",
    i18n_description: "role_details.member.description",
````
````new web/packages/constants/src/workspace.ts
    i18n_title: "role_details.member.title",
````
````old web/packages/constants/src/workspace.ts
    i18n_title: "role_details.admin.title",
    i18n_description: "role_details.admin.description",
````
````new web/packages/constants/src/workspace.ts
    i18n_title: "role_details.admin.title",
````

`web/packages/i18n/src/locales/en/common.json`（修改，6 处）：

````old web/packages/i18n/src/locales/en/common.json
      "title": "Guest",
      "description": "External members of organizations can be invited as guests."
````
````new web/packages/i18n/src/locales/en/common.json
      "title": "Guest"
````
````old web/packages/i18n/src/locales/en/common.json
      "title": "Member",
      "description": "Ability to read, write, edit, and delete entities inside projects, cycles, and modules"
````
````new web/packages/i18n/src/locales/en/common.json
      "title": "Member"
````
````old web/packages/i18n/src/locales/en/common.json
      "title": "Admin",
      "description": "All permissions set to true within the workspace."
````
````new web/packages/i18n/src/locales/en/common.json
      "title": "Admin"
````
````old web/packages/i18n/src/locales/en/common.json
      "creation_disabled": "You don't seem to have any invites to a workspace and your instance admin has restricted creation of new workspaces. Please ask a workspace owner or admin to invite you to a workspace first and come back to this screen to join."
````
````new web/packages/i18n/src/locales/en/common.json
      "creation_disabled": "Creating workspaces is switched off on this instance: ask a workspace's admin for an invitation link."
````
````old web/packages/i18n/src/locales/en/common.json
      "description": "Work in Nerve happens best with your team. Invite them now to use Nerve to its potential.",
      "role": "Role",
      "not_an_email": "That doesn't look like an email address.",
````
````new web/packages/i18n/src/locales/en/common.json
      "description": "Work in Nerve happens best with your team. Invite them now to use Nerve to its potential.",
````
````old web/packages/i18n/src/locales/en/common.json
      "later": "I’ll do it later"
````
````new web/packages/i18n/src/locales/en/common.json
      "later": "I’ll do it later",
      "links": {
        "title": "Your invitations",
        "description": "Nerve sends no email: copy each link and give it to the person it is for."
      }
````

`web/packages/i18n/src/locales/en/workspace.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/workspace.json
        "message": "Workspace created successfully"
      },
      "error": {
        "title": "Error",
        "message": "Workspace could not be created. Please try again."
````
````new web/packages/i18n/src/locales/en/workspace.json
        "message": "Workspace created successfully"
````

`web/packages/i18n/src/locales/zh-CN/common.json`（修改，6 处）：

````old web/packages/i18n/src/locales/zh-CN/common.json
      "title": "访客",
      "description": "组织的外部成员可以被邀请为访客。"
````
````new web/packages/i18n/src/locales/zh-CN/common.json
      "title": "访客"
````
````old web/packages/i18n/src/locales/zh-CN/common.json
      "title": "成员",
      "description": "可以在项目、周期和模块内读取、写入、编辑和删除实体"
````
````new web/packages/i18n/src/locales/zh-CN/common.json
      "title": "成员"
````
````old web/packages/i18n/src/locales/zh-CN/common.json
      "title": "管理员",
      "description": "在工作区内所有权限均设置为允许。"
````
````new web/packages/i18n/src/locales/zh-CN/common.json
      "title": "管理员"
````
````old web/packages/i18n/src/locales/zh-CN/common.json
      "creation_disabled": "您似乎还没有收到任何工作区的邀请，而实例管理员限制了创建新工作区。请先请工作区的所有者或管理员邀请您加入工作区，再回到这里加入。"
````
````new web/packages/i18n/src/locales/zh-CN/common.json
      "creation_disabled": "这个实例关闭了创建工作区：请向某个工作区的管理员要一个邀请链接。"
````
````old web/packages/i18n/src/locales/zh-CN/common.json
      "description": "在 Nerve 中，与团队一起工作效果最好。现在就邀请他们，充分发挥 Nerve 的作用。",
      "role": "角色",
      "not_an_email": "这看起来不像邮箱地址。",
````
````new web/packages/i18n/src/locales/zh-CN/common.json
      "description": "在 Nerve 中，与团队一起工作效果最好。现在就邀请他们，充分发挥 Nerve 的作用。",
````
````old web/packages/i18n/src/locales/zh-CN/common.json
      "later": "稍后再说"
````
````new web/packages/i18n/src/locales/zh-CN/common.json
      "later": "稍后再说",
      "links": {
        "title": "你的邀请",
        "description": "Nerve 不发邮件：复制每个链接，交给它邀请的人。"
      }
````

`web/packages/i18n/src/locales/zh-CN/workspace.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/workspace.json
        "message": "工作区创建成功"
      },
      "error": {
        "title": "错误",
        "message": "工作区创建失败。请重试。"
````
````new web/packages/i18n/src/locales/zh-CN/workspace.json
        "message": "工作区创建成功"
````

`web/packages/utils/src/validation.ts`（修改，2 处）：

````old web/packages/utils/src/validation.ts
const HAS_ALPHANUMERIC_REGEX = /[\p{L}\p{N}]/u;

/**
 * URL Slug Pattern (for workspace slugs, URL-safe identifiers)
 * Allows: Unicode letters (\p{L}), numbers (\p{N}), underscores, hyphens
 * Use case: International URL-safe identifiers like "josé-workspace", "李明-project"
 * Blocks: Spaces and special characters (URL encoding will handle Unicode in actual URLs)
 */
const SLUG_REGEX = /^[\p{L}\p{N}_-]+$/u;
````
````new web/packages/utils/src/validation.ts
const HAS_ALPHANUMERIC_REGEX = /[\p{L}\p{N}]/u;
````
````old web/packages/utils/src/validation.ts
/**
 * @description Validates URL slugs and identifiers
 * @param {string} slug - Slug to validate
 * @returns {boolean | string} true if valid, error message if invalid
 * @example
 * validateSlug("my-workspace") // returns true
 * validateSlug("my_workspace_123") // returns true
 * validateSlug("my workspace") // returns error message (spaces not allowed)
 */
export const validateSlug = (slug: string): boolean | string => {
  if (!slug || slug.trim() === "") {
    return "Slug is required";
  }

  if (slug.length > 48) {
    return "Slug must be 48 characters or less";
  }

  if (hasInjectionRiskChars(slug)) {
    return "Slug cannot contain special characters like < > ' \" { } [ ] * ^ ! # %";
  }

  if (!SLUG_REGEX.test(slug)) {
    return "Slug can only contain letters, numbers, hyphens, and underscores";
  }

  return true;
};

/**
````
````new web/packages/utils/src/validation.ts
/**
````

- [ ] **Step 3: 静态检查的范围**

`.oxlintrc.json`（修改，2 处）：

````old .oxlintrc.json
        "web/apps/web/core/components/project/project-settings-member-defaults.tsx"
````
````new .oxlintrc.json
        "web/apps/web/core/components/project/project-settings-member-defaults.tsx",
        "web/apps/web/core/components/onboarding/root.tsx"
````
````old .oxlintrc.json
        "web/apps/web/core/components/workspace/use-create-workspace.test.ts"
````
````new .oxlintrc.json
        "web/apps/web/core/components/workspace/use-create-workspace.test.ts",
        "web/apps/web/core/components/onboarding/onboarding-place.ts",
        "web/apps/web/core/components/onboarding/onboarding-place.test.ts",
        "web/apps/web/core/components/onboarding/root.test.tsx",
        "web/apps/web/core/components/onboarding/steps/profile/root.test.tsx",
        "web/apps/web/core/components/onboarding/steps/team/links.tsx"
````

- [ ] **Step 4: 端到端**

`e2e/stories/smoke/s2-web-app.spec.ts`（修改，2 处）：

````old e2e/stories/smoke/s2-web-app.spec.ts
import { accountId, emailFor, type AuthTokens } from "../../fixtures/auth";
````
````new e2e/stories/smoke/s2-web-app.spec.ts
import { accountId, emailFor, register, type AuthTokens } from "../../fixtures/auth";
````
````old e2e/stories/smoke/s2-web-app.spec.ts
  expect(requests.between(0)).toEqual({ pending: 0, requests: expected });
});

````
````new e2e/stories/smoke/s2-web-app.spec.ts
  expect(requests.between(0)).toEqual({ pending: 0, requests: expected });
});

test("S2: a newcomer opens /, which sends him to the onboarding: it asks for his workspaces, as the app does, and no more", async ({
  api,
  signedInPage,
}, testInfo) => {
  const page = await signedInPage(await register(api, emailFor(testInfo)));
  const watch = await watchPage(page);
  const requests = followRequests(page, {});
  await page.goto("/");
  await expect(page).toHaveURL("/onboarding");
  await expect(page.getByText("Create your profile.")).toBeVisible();
  await expect.poll(() => requests.between(0)).toEqual({ pending: 0, requests: APP.toSorted() });
  expect([watch.apiFailures, watch.oldApiRequests, watch.cspViolations, watch.pageErrors]).toEqual([[], [], [], []]);
  await expectQuietConsole(page, watch);
  // Nothing came after the list was whole.
  expect(requests.between(0)).toEqual({ pending: 0, requests: APP.toSorted() });
});

````

`e2e/stories/workspace/w1-create-workspace.spec.ts`（修改，6 处）：

````old e2e/stories/workspace/w1-create-workspace.spec.ts
import { createApi, createWorkspace, slugFor, type Api } from "../../fixtures/api";
````
````new e2e/stories/workspace/w1-create-workspace.spec.ts
import type { Response } from "@playwright/test";

import { createApi, createWorkspace, invitationTo, inviteAndAccept, slugFor, type Api } from "../../fixtures/api";
````
````old e2e/stories/workspace/w1-create-workspace.spec.ts
  countWorkspaces,
````
````new e2e/stories/workspace/w1-create-workspace.spec.ts
  countWorkspaces,
  expectInvitations,
````
````old e2e/stories/workspace/w1-create-workspace.spec.ts
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
````
````new e2e/stories/workspace/w1-create-workspace.spec.ts
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { saveProfileStep } from "../../fixtures/onboarding-pages";
````
````old e2e/stories/workspace/w1-create-workspace.spec.ts
import { anotherBrowser } from "../../fixtures/workspace-pages";
````
````new e2e/stories/workspace/w1-create-workspace.spec.ts
import { anotherBrowser, invitationLinkOf } from "../../fixtures/workspace-pages";
````
````old e2e/stories/workspace/w1-create-workspace.spec.ts
// W1, create a workspace (M3 design 2, 3.10, 3.11).
````
````new e2e/stories/workspace/w1-create-workspace.spec.ts
// W1, create a workspace (M3 design 2, 3.10, 3.11), at /create-workspace and in the onboarding (7.4).
````
````old e2e/stories/workspace/w1-create-workspace.spec.ts
  await there.close();
});

````
````new e2e/stories/workspace/w1-create-workspace.spec.ts
  await there.close();
});

/** What finishing the onboarding writes: every step done, and the account onboarded. */
const FINISHED = {
  onboarding_step: { profile_complete: true, workspace_join: true, workspace_create: true, workspace_invite: true },
  is_onboarded: true,
};

test("W1 (page): a newcomer's onboarding creates a workspace after the profile step, written as the one opened last; invites to it and shows the link to copy; then lands in it", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const invitee = emailFor(testInfo, "invitee");
  const tokens = await register(api, email);
  const page = await signedInPage(tokens);
  await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
  const watch = await watchPage(page);
  await page.goto("/onboarding");
  await page.getByLabel("Name", { exact: true }).fill("Ada");
  expect(await saveProfileStep(page)).toBe(200);

  // The creation step: nerve's check, the creation, and the workspace created written with the step.
  const slug = slugFor(testInfo);
  await page.locator("#name").fill("Acme");
  await page.locator("#slug").fill(slug);
  await page.getByRole("button", { name: "2-10", exact: true }).click();
  let created: { body: unknown; answer: Response } | undefined;
  const stepped = await sentTo(page, "PATCH", "/api/v0/me/profile", async () => {
    created = await sentTo(page, "POST", "/api/v0/workspaces", () =>
      page.getByRole("button", { name: "Create workspace" }).click()
    );
  });
  expect([created?.answer.status(), created?.body]).toEqual([201, { name: "Acme", slug, organization_size: "2-10" }]);
  const id = await expectWorkspaceCreated(db, email, {
    name: "Acme",
    slug,
    organization_size: "2-10",
    timezone: "UTC",
  });
  expect([stepped.answer.status(), stepped.body]).toEqual([
    200,
    { onboarding_step: { workspace_create: true }, last_workspace_id: id },
  ]);

  // The invitation step invites to it, and shows the link of each invitation.
  await page.locator('[id="invitations.0.email"]').fill(invitee);
  const invited = await sentTo(page, "POST", `/api/v0/workspaces/${slug}/invitations`, () =>
    page.getByRole("button", { name: "Continue", exact: true }).click()
  );
  expect([invited.answer.status(), invited.body]).toEqual([201, { invitations: [{ email: invitee, role: 15 }] }]);
  await expect(page.getByText("Your invitations")).toBeVisible();
  await page.getByRole("button", { name: "Copy link" }).click();
  await expect(page.getByText("Invite link copied to clipboard")).toBeVisible();
  const invitation = await invitationTo(api, tokens.access_token, slug, invitee);
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(
    new URL(invitationLinkOf(invitation), page.url()).href
  );
  await expectInvitations(db, slug, email, [
    { email: invitee, role: 15, accepted: false, responded: false, deleted: false },
  ]);

  // Done: the landing is the workspace created.
  const finished = await sentTo(page, "PATCH", "/api/v0/me/profile", () =>
    page.getByRole("button", { name: "Continue", exact: true }).click()
  );
  expect([finished.answer.status(), finished.body]).toEqual([200, FINISHED]);
  await expect(page).toHaveURL(`/${slug}`);
  expect(await lastWorkspaceOf(db, email)).toBe(id);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  // The workspace's home.
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});

test("W1 (page): a newcomer who puts off the invitations at their step lands in the workspace he created, having invited no one", async ({
  api,
  signedInPage,
}, testInfo) => {
  const page = await signedInPage(await register(api, emailFor(testInfo)));
  const watch = await watchPage(page);
  await page.goto("/onboarding");
  await page.getByLabel("Name", { exact: true }).fill("Ada");
  expect(await saveProfileStep(page)).toBe(200);
  const slug = slugFor(testInfo);
  await page.locator("#name").fill("Acme");
  await page.locator("#slug").fill(slug);
  await page.getByRole("button", { name: "2-10", exact: true }).click();
  const stepped = await sentTo(page, "PATCH", "/api/v0/me/profile", () =>
    page.getByRole("button", { name: "Create workspace" }).click()
  );
  expect(stepped.answer.status()).toBe(200);

  // The invitation step, put off: the onboarding ends, and he lands in the workspace he created.
  const finished = await sentTo(page, "PATCH", "/api/v0/me/profile", () =>
    page.getByRole("button", { name: "I’ll do it later", exact: true }).click()
  );
  expect([finished.answer.status(), finished.body]).toEqual([200, FINISHED]);
  await expect(page).toHaveURL(`/${slug}`);
  expect(watch.apiRequests.filter((request) => request.endsWith(`/api/v0/workspaces/${slug}/invitations`))).toEqual([]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  // The workspace's home.
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});

test("W1 (page): one who joined a workspace before onboarding is done after the profile step; one who comes back after creating a workspace invites to that one, whatever comes first in his list", async ({
  api,
  baseURL,
  browser,
  signedInPage,
}, testInfo) => {
  const admin = await registerOnboarded(api, emailFor(testInfo, "admin"));
  const alpha = await createWorkspace(api, admin.access_token, { name: "Alpha", slug: slugFor(testInfo, "alpha") });
  const newcomer = async (label: string) => {
    const email = emailFor(testInfo, label);
    const tokens = await register(api, email);
    await inviteAndAccept(api, admin.access_token, alpha.slug, { email, token: tokens.access_token }, 15);
    return tokens;
  };

  // Ada joined Alpha: her profile step is her last, and she lands there.
  const page = await signedInPage(await newcomer("ada"));
  const watch = await watchPage(page);
  await page.goto("/onboarding");
  await page.getByLabel("Name", { exact: true }).fill("Ada");
  const finished = await sentTo(page, "PATCH", "/api/v0/me/profile", () =>
    page.getByRole("button", { name: "Continue", exact: true }).click()
  );
  expect([finished.answer.status(), finished.body]).toEqual([200, FINISHED]);
  await expect(page).toHaveURL(`/${alpha.slug}`);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });

  // Bob joined Alpha too, and created Zeta, which comes after it, and left the onboarding at its invitation step.
  const bob = await newcomer("bob");
  const zeta = await createWorkspace(api, bob.access_token, { name: "Zeta", slug: slugFor(testInfo, "zeta") });
  const left = await api.PATCH("/api/v0/me/profile", {
    body: { onboarding_step: { profile_complete: true, workspace_create: true }, last_workspace_id: zeta.id },
    headers: bearer(bob.access_token),
  });
  expect(left.response.status).toBe(200);
  const there = await anotherBrowser(browser, baseURL ?? "", bob);
  await there.page.goto("/onboarding");
  await expect(there.page.getByText("Invite your teammates")).toBeVisible();
  await there.page.locator('[id="invitations.0.email"]').fill(emailFor(testInfo, "carol"));
  const invited = await answerTo(there.page, "POST", `/api/v0/workspaces/${zeta.slug}/invitations`, () =>
    there.page.getByRole("button", { name: "Continue", exact: true }).click()
  );
  expect(invited.status()).toBe(201);
  await there.close();
});

test("W1 (page): on a nerve with creation switched off, a newcomer's onboarding says to ask for an invitation link after the profile step", async ({
  browser,
  nerveWith,
}, testInfo) => {
  const closed = await nerveWith({ NERVE_WORKSPACE__CREATION_ENABLED: "false" });
  const tokens = await register(createApi(closed.baseURL), emailFor(testInfo));
  const there = await anotherBrowser(browser, closed.baseURL, tokens);
  await there.page.goto("/onboarding");
  await there.page.getByLabel("Name", { exact: true }).fill("Ada");
  expect(await saveProfileStep(there.page)).toBe(200);
  await expect(
    there.page.getByText(
      "Creating workspaces is switched off on this instance: ask a workspace's admin for an invitation link."
    )
  ).toBeVisible();
  await expect(there.page.locator("#name")).toHaveCount(0);
  await there.close();
});

````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、3 个例外，没有命中；web 的 oxlint 359 条，等于上限。

Run: `make knip`
Expected: 通过（`validateSlug`、`SLUG_REGEX`、删除的步骤和文案都不再被引用）。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 88 个全部通过。

- [ ] **Step 6: 提交**

```bash
git add .oxlintrc.json e2e/stories/smoke/s2-web-app.spec.ts e2e/stories/workspace/w1-create-workspace.spec.ts 'web/apps/web/app/(all)/onboarding/page.tsx' web/apps/web/core/components/onboarding/onboarding-place.test.ts web/apps/web/core/components/onboarding/onboarding-place.ts web/apps/web/core/components/onboarding/root.test.tsx web/apps/web/core/components/onboarding/root.tsx web/apps/web/core/components/onboarding/steps/profile/root.test.tsx web/apps/web/core/components/onboarding/steps/profile/root.tsx web/apps/web/core/components/onboarding/steps/root.tsx web/apps/web/core/components/onboarding/steps/team/links.tsx web/apps/web/core/components/onboarding/steps/team/root.tsx web/apps/web/core/components/onboarding/steps/workspace/create.tsx web/apps/web/core/components/onboarding/steps/workspace/index.ts web/apps/web/core/components/onboarding/steps/workspace/root.tsx web/apps/web/core/store/user/profile.store.ts web/packages/constants/src/workspace.ts web/packages/i18n/src/locales/en/common.json web/packages/i18n/src/locales/en/workspace.json web/packages/i18n/src/locales/zh-CN/common.json web/packages/i18n/src/locales/zh-CN/workspace.json web/packages/utils/src/validation.ts
```
```bash
git commit -m "feat(M3/P9): the onboarding lists the caller's workspaces and decides its steps by them

The onboarding fetches the caller's workspaces and decides by them: one
who has a workspace is done after the profile step; one who has none
creates one, written as the one opened last with its step, and invites
to it with the members page's form, then copies each invitation's link,
or puts the invitations off.
Resumed, it invites to the workspace it created, not the list's first.
Each change is followed only in its session and a refusal says nerve's
reason; the profile step hands the onboarding on only in the session
the names were sent in; finishing no longer writes the list's first
workspace. The create step shares useCreateWorkspace; the slug helpers
and the roles' descriptions it no longer needs are deleted. The root
joins the swr override, the new modules the non-null one. W1's
onboarding page versions and S2's newcomer.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p9.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T6.1` | 新手引导不取调用者的工作区列表（P8a 的 M7） | `root.test.tsx`、故事 S2 | vitest；端到端 |
| `T6.2` | 已经加入工作区的人在资料一步之后被要求创建工作区（决策点 2） | `onboarding-place.test.ts`、`root.test.tsx`、故事 W1 | vitest；端到端 |
| `T6.3` | 接着做的邀请一步邀请到列表中的第一个工作区（P8a 的 M4） | `onboarding-place.test.ts`、故事 W1 | vitest；端到端 |
| `T6.4` | 建好的工作区不写成上次打开的工作区 | `root.test.tsx`、故事 W1 | vitest；端到端 |
| `T6.5` | 完成引导时把列表中的第一个工作区写成上次打开的（Plane） | 故事 W1 | 端到端 |
| `T6.6` | 新手引导的修改不论会话都跟进 | `root.test.tsx` | vitest |
| `T6.7` | 邀请一步发出邀请之后不显示链接 | oxlint（`check:lint`）、故事 W1 | 静态；端到端 |
| `T6.8` | 新手引导的修改被拒绝时什么都不说 | `root.test.tsx` | vitest |
| `T6.9` | 邀请成员一步的"以后再说"不结束引导 | 故事 W1 | 端到端 |
| `T6.10` | 新手引导一步的资料修改不论会话都跟进（预检的 PF19） | `root.test.tsx` | vitest |
| `T6.11` | 资料一步不论会话都把引导交给根（预检的 M2） | `root.test.tsx` | vitest |
| `T6.12` | 新手引导的根经 `swr` 取列表，不经 `useSessionSWR` | oxlint（`check:lint`） | 静态 |
| `T6.13` | 新手引导的位置（`onboarding-place.ts`，范围内）用非空断言 | oxlint（`check:lint`） | 静态 |

---

### Task 7: 带邀请的注册；登录页和注册页之间保留链接；W6 的页面版本

**Files:**
- Modify: `e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts`、`web/apps/web/core/components/account/auth-forms/auth-root.tsx`、`web/apps/web/core/components/account/auth-forms/password.tsx`、`web/apps/web/core/components/auth-screens/header.tsx`

**Interfaces:**
- Produces（spec 2.7；M3 设计 2 的 W6，3.8，7.4；决定 1；M2 设计 3.18）：
  - `auth-forms/auth-root.tsx`：地址带 `invitation_id` 和 `token`（Task 4 的 `invitationAuthPath` 给出）时，把 `{ id, token }` 交给密码表单。
  - `auth-forms/password.tsx`：`AuthPasswordForm` 收 `invitation?: RegisterInvitation`（生成的类型）；注册发 `{ email, password, invitation }`（生成的 `RegisterRequest`；注册关闭时 nerve 凭它放行，决定 1）。登录不带它。原来注册页的页头显示要加入的工作区，注册本身却不带邀请，注册关闭时被拒绝。
  - `auth-screens/header.tsx`：登录页和注册页之间的链接带上当前页面的查询参数（`useLocation().search`：邀请链接和回到它的 `next_path`），原来切换之后两者都丢了。

**Tests:** 没有新的 vitest：注册的请求体由 W6 的页面版本经 `sentTo` 钉住，查询参数的保留由第二个页面版本核对。
- 端到端：`W6 (page): with sign-up off, the sign-up page says so; the invitee opens her link signed out and signs up to accept it, under the workspace's name, with the link's invitation; back at the link she accepts, takes the profile step alone and lands in the workspace`（请求体 `{ email, password, invitation: { id, token } }`；接受之后新手引导只有资料一步，落在工作区，写成上次打开的）；`W6 (page): with sign-up on, one who has an account goes from the sign-up page of her link to the sign-in page, which keeps the link and comes back to it`。

- [ ] **Step 1: 注册带上邀请，链接保留查询参数**

`web/apps/web/core/components/account/auth-forms/auth-root.tsx`（修改，2 处）：

````old web/apps/web/core/components/account/auth-forms/auth-root.tsx
  // query params: a workspace invitation's link, for M3's "join the workspace" title (M2 design 7.3)
````
````new web/apps/web/core/components/account/auth-forms/auth-root.tsx
  // query params: a workspace invitation's link, for the "join the workspace" title and the registration it lets
  // through (M3 design 7.4)
````
````old web/apps/web/core/components/account/auth-forms/auth-root.tsx
        <AuthPasswordForm mode={authMode} />
````
````new web/apps/web/core/components/account/auth-forms/auth-root.tsx
        <AuthPasswordForm
          mode={authMode}
          invitation={invitation_id && token ? { id: invitation_id, token } : undefined}
        />
````

`web/apps/web/core/components/account/auth-forms/password.tsx`（修改，4 处）：

````old web/apps/web/core/components/account/auth-forms/password.tsx
import { Button } from "@nerve/propel/button";
````
````new web/apps/web/core/components/account/auth-forms/password.tsx
import { Button } from "@nerve/propel/button";
import type { RegisterInvitation } from "@nerve/api-client";
````
````old web/apps/web/core/components/account/auth-forms/password.tsx
  mode: EAuthModes;
````
````new web/apps/web/core/components/account/auth-forms/password.tsx
  mode: EAuthModes;
  /** The invitation of the link the page was opened with: a registration sends it (M3 design 7.4, decision 1). */
  invitation?: RegisterInvitation;
````
````old web/apps/web/core/components/account/auth-forms/password.tsx
  const { mode } = props;
````
````new web/apps/web/core/components/account/auth-forms/password.tsx
  const { mode, invitation } = props;
````
````old web/apps/web/core/components/account/auth-forms/password.tsx
      await (mode === EAuthModes.SIGN_IN ? signIn(credentials) : signUp(credentials));
````
````new web/apps/web/core/components/account/auth-forms/password.tsx
      await (mode === EAuthModes.SIGN_IN ? signIn(credentials) : signUp({ ...credentials, invitation }));
````

`web/apps/web/core/components/auth-screens/header.tsx`（修改，3 处）：

````old web/apps/web/core/components/auth-screens/header.tsx
import { Link } from "react-router";
````
````new web/apps/web/core/components/auth-screens/header.tsx
import { Link, useLocation } from "react-router";
````
````old web/apps/web/core/components/auth-screens/header.tsx
  const { t } = useTranslation();
````
````new web/apps/web/core/components/auth-screens/header.tsx
  const { t } = useTranslation();
  // router: the other page gets this one's query, an invitation's link and the way back to it (M3 design 7.4)
  const { search } = useLocation();
````
````old web/apps/web/core/components/auth-screens/header.tsx
              to={authContentMap[type].linkHref}
````
````new web/apps/web/core/components/auth-screens/header.tsx
              to={{ pathname: authContentMap[type].linkHref, search }}
````

- [ ] **Step 2: 端到端**

`e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts`（修改，4 处）：

````old e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts
import { accept, createApi, createWorkspace, invite, slugFor } from "../../fixtures/api";
````
````new e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts
import { accept, createApi, createWorkspace, invite, slugFor, type WorkspaceInvitation } from "../../fixtures/api";
````
````old e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts
import { expectInvitations, expectMembership } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, password, register, type RegisterInvitation } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
````
````new e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts
import { expectInvitations, expectMembership, lastWorkspaceOf } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, password, register, type RegisterInvitation } from "../../fixtures/auth";
import { fillSignUp, formAlert, submitSignIn, submitSignUp } from "../../fixtures/auth-pages";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { registerOnboarded, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { invitationLinkOf } from "../../fixtures/workspace-pages";
````
````old e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts
// W6, sign up by an invitation while sign-up is off (M3 design 2, 3.8; decision 1). The page version comes with
// /workspace-invitations (P9).
````
````new e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts
// W6, sign up by an invitation while sign-up is off (M3 design 2, 3.8, 7.4; decision 1).
````
````old e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts
  );
});

````
````new e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts
  );
});

/** The sign-in or sign-up page for an invitation's link, which comes back to it. */
const authPathOf = (path: "/" | "/sign-up", invitation: Pick<WorkspaceInvitation, "id" | "token">) => {
  const query = { invitation_id: invitation.id, token: invitation.token, next_path: invitationLinkOf(invitation) };
  return `${path}?${new URLSearchParams(query).toString()}`;
};

test("W6 (page): with sign-up off, the sign-up page says so; the invitee opens her link signed out and signs up to accept it, under the workspace's name, with the link's invitation; back at the link she accepts, takes the profile step alone and lands in the workspace", async ({
  api,
  browser,
  db,
  nerveWith,
}, testInfo) => {
  // The workspace and the invitation are the closed nerve's, as in W6's API version.
  const closed = await nerveWith({ NERVE_AUTH__SIGNUP_ENABLED: "false" });
  const closedApi = createApi(closed.baseURL);
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await createPAT(api, (await register(api, adminEmail)).access_token)).token;
  const slug = slugFor(testInfo);
  const acme = await createWorkspace(closedApi, admin, { name: "Acme", slug });
  const carolEmail = emailFor(testInfo, "carol");
  const [toCarol] = await invite(closedApi, admin, slug, [{ email: carolEmail, role: 15 }]);
  if (!toCarol) {
    throw new Error("the invitation was not created");
  }
  const context = await browser.newContext({ baseURL: closed.baseURL });
  const page = await context.newPage();
  const watch = await watchPage(page);

  // Without a link, M2's sign-up page: nerve says sign-up is closed.
  await page.goto("/sign-up");
  expect(await submitSignUp(page, carolEmail, password)).toBe(403);
  await expect(formAlert(page)).toHaveText("Sign-up is closed.");

  // With her link: she signs up to accept it, and the registration carries the invitation.
  await page.goto(invitationLinkOf(toCarol));
  await page.getByRole("link", { name: "Sign up to accept" }).click();
  await expect(page).toHaveURL(authPathOf("/sign-up", toCarol));
  // the heading: "Join", the workspace's logo (its initial while it has no image), its name
  await expect(page.getByText(/^Join\s+A\s+Acme$/)).toBeVisible();
  await fillSignUp(page, carolEmail, password);
  const registered = await sentTo(page, "POST", "/api/v0/auth/register", () =>
    page.getByRole("button", { name: "Create account", exact: true }).click()
  );
  expect([registered.answer.status(), registered.body]).toEqual([
    201,
    { email: carolEmail, password, invitation: { id: toCarol.id, token: toCarol.token } },
  ]);

  // Back at the link she accepts; her onboarding is the profile step alone, and she lands in Acme.
  await expect(page).toHaveURL(invitationLinkOf(toCarol));
  await page.getByRole("button", { name: "Accept" }).click();
  await expect(page).toHaveURL("/onboarding");
  await page.getByLabel("Name", { exact: true }).fill("Carol");
  await page.getByRole("button", { name: "Continue", exact: true }).click();
  await expect(page).toHaveURL(`/${slug}`);
  await expectMembership(db, slug, carolEmail, { role: 15, is_active: true });
  expect(await lastWorkspaceOf(db, carolEmail)).toBe(acme.id);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    ["403 POST /api/v0/auth/register"],
    [],
    [],
  ]);
  // Acme's home logs the hint once; the browser reports the refusal.
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING],
    errors: ["Failed to load resource: the server responded with a status of 403 (Forbidden)"],
  });
  await context.close();
});

test("W6 (page): with sign-up on, one who has an account goes from the sign-up page of her link to the sign-in page, which keeps the link and comes back to it", async ({
  api,
  baseURL,
  browser,
}, testInfo) => {
  const admin = await registerOnboarded(api, emailFor(testInfo, "admin"));
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  const daveEmail = emailFor(testInfo, "dave");
  await registerOnboarded(api, daveEmail);
  const [toDave] = await invite(api, admin.access_token, slug, [{ email: daveEmail, role: 15 }]);
  if (!toDave) {
    throw new Error("the invitation was not created");
  }
  const context = await browser.newContext({ baseURL });
  const page = await context.newPage();
  await page.goto(authPathOf("/sign-up", toDave));
  await page.getByRole("link", { name: "Sign in", exact: true }).click();
  await expect(page).toHaveURL(authPathOf("/", toDave));
  await expect(page.getByText(/^Join\s+A\s+Acme$/)).toBeVisible();
  expect(await submitSignIn(page, daveEmail, password)).toBe(200);
  await expect(page).toHaveURL(invitationLinkOf(toDave));
  await context.close();
});

````

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、3 个例外，没有命中；web 的 oxlint 359 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 90 个全部通过。

- [ ] **Step 4: 提交**

```bash
git add e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts web/apps/web/core/components/account/auth-forms/auth-root.tsx web/apps/web/core/components/account/auth-forms/password.tsx web/apps/web/core/components/auth-screens/header.tsx
```
```bash
git commit -m "feat(M3/P9): a sign-up from an invitation's link carries the invitation, and the auth pages keep the link

The sign-up page opened from an invitation's link sends the link's
invitation with the registration, which nerve lets through while sign-up
is closed (decision 1); the link between the sign-in and sign-up pages
keeps the page's query, the invitation and the way back to it. W6's page
versions.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p9.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T7.1` | 从邀请链接来的注册不带邀请 | oxlint（`check:lint`）、故事 W6 | 静态；端到端 |
| `T7.2` | 从邀请链接来的登录也带邀请 | `tsc` | 静态 |
| `T7.3` | 登录页和注册页之间的链接丢掉邀请链接的参数 | 故事 W6 | 端到端 |

---

### Task 8: 导航设置的修改在轮到它时算出；项目导航对话框；首页的导览；W8 的页面版本

**Files:**
- Create: `web/apps/web/core/components/home/root.test.tsx`、`web/apps/web/core/hooks/use-navigation-preferences.test.ts`
- Modify: `.oxlintrc.json`、`e2e/stories/workspace/w8-navigation-preferences.spec.ts`、`web/apps/web/core/components/home/root.tsx`、`web/apps/web/core/components/navigation/project-navigation-dialog.tsx`、`web/apps/web/core/components/sidebar/sidebar-wrapper.tsx`、`web/apps/web/core/hooks/navigation-preferences.test.ts`、`web/apps/web/core/hooks/navigation-preferences.ts`、`web/apps/web/core/hooks/use-navigation-preferences.ts`、`web/apps/web/core/store/workspace/preferences.store.test.ts`、`web/apps/web/core/store/workspace/preferences.store.ts`、`web/packages/i18n/src/locales/en/accessibility.json`、`web/packages/i18n/src/locales/zh-CN/accessibility.json`、`web/packages/types/src/navigation-preferences.ts`

**Interfaces:**
- Produces（spec 2.8；M3 设计 2 的 W8，3.18，7.1；总体设计 7.7 的 P8b 一条；P8b 评审的 E5；P8a 的 P12）：
  - `core/store/workspace/preferences.store.ts`：`PreferencesChange = (held: WorkspacePreferences) => WorkspacePreferencesUpdate`；`updatePreferences(workspaceSlug, change: PreferencesChange)` 在队列轮到它时，把 `change` 作用于 store 此刻所持的、nerve 最近一次回答的设置，再发出（照 P8b 的 `ProjectPreferencesStore`）；还没有这个工作区的设置时不发出，以 `Workspace settings not found` 失败。原来收现成的请求体，页面按调用时侧边栏显示的值算出它（E5）。
  - `core/hooks/navigation-preferences.ts`（整个文件）：`NERVE_DEFAULTS`（`ACCORDION`、10，nerve 的默认值）是 `navigationOf` 的缺省参数：设置到达之前侧边栏按默认值显示，不再在到达时从"全部项目"跳到 10 个（原来缺省是 `@nerve/types` 的 `DEFAULT_PROJECT_PREFERENCES`，它随之删除）；`TProjectNavigationChange` 是模式、限制的一次开关（`{ limitToggled: true }`）或数量之一；`preferencesChangeOf(change): PreferencesChange`，修改在轮到它时算出：开关按所作用的设置决定打开还是关闭（设置限制项目数时关闭，是 0；不限制时打开，取 `navigationOf(held)` 的数量，即 nerve 的默认数量），连按两下回到原样（预检 M3：原来对话框发出点击时显示的开或关，nerve 回答第一下之前的第二下仍发同一个值；P8b 的 `toggleProject` 的先例）；`countOf(draft)`：数量框的草稿限制到的数量（数字组成的数，至少 1，否则没有）。
  - `core/hooks/use-navigation-preferences.ts`（整个文件）：`useProjectNavigationPreferences()` 给出 `{ preferences, changeNavigation(change) }`：`preferences` 是地址的工作区的设置（`navigationOf`），`changeNavigation` 经 `followInSession` 发出 `preferencesChangeOf(change)`，拒绝按 `code` 提示。原来是三个更新函数，被拒绝时成为没有处理的拒绝。
  - `ProjectNavigationDialog`：开关发出 `{ limitToggled: true }`；数量框的文字在编辑时是草稿（`null` 时显示设置的数量，随设置到达更新），失去焦点时发出一次（原来每敲一个数字发一次：输入 30 发出 3 和 30）；标签经 `htmlFor`/`useId` 关联输入框；`blockNonDigits` 移到模块级（一条 `oxlint-disable-next-line` 删除）。`sidebar-wrapper.tsx` 打开对话框的按钮有名字（`aria_labels.projects_sidebar.project_navigation`，en "Project navigation"、zh-CN）。
  - `home/root.tsx`：导览结束（`updateTourCompleted`）经 `followInSession`，拒绝按 `code` 提示（原来只有 `console.error`）。
  - 静态检查（总体设计 7.7）：`.oxlintrc.json` 的非空断言的范围加上 `use-navigation-preferences.test.ts`、`home/root.test.tsx`（hook 和 `navigation-preferences.test.ts` 在 P8a 起已在范围内）。

**Tests:**
- vitest：`preferences.store.test.ts` 的修改都写成函数（`to(data)`），加 `makes each change to the settings nerve answered the one before it, not to those the store had as it was asked for`；`makes two quick turns of the limit each to nerve's answer to the one before it: off, then on again`；`fails without sending a change while it has no settings of the workspace`。`navigation-preferences.test.ts`（整个文件）：`navigationOf` 的三条；`preferencesChangeOf` 的 `sends only the setting a change changes, made to the settings nerve last answered: $does`（四行：模式、限制项目数时的一次开关、不限制时的一次开关、数量）；`countOf` 的 `limits the sidebar to the number of a draft's digits, when 1 or more: '$draft'`（四行）。`use-navigation-preferences.test.ts`：`shows the caller's settings in the address's workspace, and nerve's defaults until they arrive`；`sends a change to the address's workspace, made in its turn to the settings nerve last answered`；`shows nerve's reason when it refuses a change`；`says nothing of a change that $settles after another tab moved this one to another account`。`home/root.test.tsx`（`WorkspaceHomeView`）：`writes the tour's end in the caller's profile`；`shows nerve's reason when it refuses the tour's end`；`says nothing of the tour's end when it $settles after another tab moved this one to another account`。
- 端到端：`W8 (page): in his workspace's sidebar the caller makes the project navigation tabs and limits it to 3 projects, the count sent once, as he leaves its field; so it stays after a refresh; a count nerve refuses is said, and the sidebar keeps its 3`（跳过导览的请求体 `{ is_tour_completed: true }`；四个项目按默认值全部显示；改成标签页 `{ navigation_control_preference: "TABBED" }`；连按两下开关，第二下在 nerve 回答第一下之前（`holdAnswer`）：先关闭 `{ navigation_project_limit: 0 }`，再在那个回答之上打开 `{ navigation_project_limit: 10 }`，回到原样；输入 30、删一位、离开，只发一次 `{ navigation_project_limit: 3 }`；刷新之后仍是 3 个，对话框显示标签页和 3；3000000000 被拒绝（422），页面说明，侧边栏仍是 3 个；一共五个 `PATCH`。项目行只在"Main sidebar"里数：侧边栏的项目列表渲染了三份）。

- [ ] **Step 1: store 的修改是函数**

`web/apps/web/core/store/workspace/preferences.store.test.ts`（修改，10 处）：

````old web/apps/web/core/store/workspace/preferences.store.test.ts
import type { WorkspacePreferences } from "@nerve/api-client";
````
````new web/apps/web/core/store/workspace/preferences.store.test.ts
import type { WorkspacePreferences, WorkspacePreferencesUpdate } from "@nerve/api-client";
import { preferencesChangeOf } from "@/hooks/navigation-preferences";
````
````old web/apps/web/core/store/workspace/preferences.store.test.ts
import type { IWorkspacePreferencesStore } from "@/store/workspace/preferences.store";
````
````new web/apps/web/core/store/workspace/preferences.store.test.ts
import type { IWorkspacePreferencesStore, PreferencesChange } from "@/store/workspace/preferences.store";
````
````old web/apps/web/core/store/workspace/preferences.store.test.ts
const all: WorkspacePreferences = { navigation_control_preference: "ACCORDION", navigation_project_limit: 0 };
````
````new web/apps/web/core/store/workspace/preferences.store.test.ts
const all: WorkspacePreferences = { navigation_control_preference: "ACCORDION", navigation_project_limit: 0 };
/** The change to data, whatever the settings it is made to. */
const to =
  (data: WorkspacePreferencesUpdate): PreferencesChange =>
  () =>
    data;
````
````old web/apps/web/core/store/workspace/preferences.store.test.ts
    const { nerve, store } = await loaded();
    const changed = track(store.updatePreferences("acme", { navigation_control_preference: "TABBED" }));
````
````new web/apps/web/core/store/workspace/preferences.store.test.ts
    const { nerve, store } = await loaded();
    const changed = track(store.updatePreferences("acme", to({ navigation_control_preference: "TABBED" })));
````
````old web/apps/web/core/store/workspace/preferences.store.test.ts
    const refused = track(store.updatePreferences("acme", { navigation_project_limit: -1 }));
````
````new web/apps/web/core/store/workspace/preferences.store.test.ts
    const refused = track(store.updatePreferences("acme", to({ navigation_project_limit: -1 })));
````
````old web/apps/web/core/store/workspace/preferences.store.test.ts
    await load(nerve, store, all, "beta");
    const changed = track(store.updatePreferences("acme", { navigation_control_preference: "TABBED" }));
````
````new web/apps/web/core/store/workspace/preferences.store.test.ts
    await load(nerve, store, all, "beta");
    const changed = track(store.updatePreferences("acme", to({ navigation_control_preference: "TABBED" })));
````
````old web/apps/web/core/store/workspace/preferences.store.test.ts
    const first = track(store.updatePreferences("acme", { navigation_project_limit: 5 }));
    const second = track(store.updatePreferences("acme", { navigation_control_preference: "TABBED" }));
````
````new web/apps/web/core/store/workspace/preferences.store.test.ts
    const first = track(store.updatePreferences("acme", to({ navigation_project_limit: 5 })));
    const second = track(store.updatePreferences("acme", to({ navigation_control_preference: "TABBED" })));
````
````old web/apps/web/core/store/workspace/preferences.store.test.ts
  });

  it("fetches the settings while a change is out: a fetch does not wait for it", async () => {
````
````new web/apps/web/core/store/workspace/preferences.store.test.ts
  });

  it("makes each change to the settings nerve answered the one before it, not to those the store had as it was asked for", async () => {
    const { nerve, store } = await loaded();
    const first = track(store.updatePreferences("acme", to({ navigation_control_preference: "TABBED" })));
    // turning the limit on: a limit of the count of the settings it is made to
    const second = track(
      store.updatePreferences("acme", (held) => ({ navigation_project_limit: held.navigation_project_limit }))
    );
    // nerve's answer to the first: another tab of his has set the limit to 3 meanwhile
    await inTurn(nerve, 1, ["PATCH", PREFERENCES], json(200, elsewhere));
    await inTurn(nerve, 2, ["PATCH", PREFERENCES], json(200, elsewhere));
    await until(() => second.settled, "the second change");
    expect(first.error).toBeUndefined();
    expect(nerve.calls[2]?.body).toEqual({ navigation_project_limit: 3 });
  });

  it("makes two quick turns of the limit each to nerve's answer to the one before it: off, then on again", async () => {
    const { nerve, store } = await loaded();
    const turn = preferencesChangeOf({ limitToggled: true });
    track(store.updatePreferences("acme", turn));
    const second = track(store.updatePreferences("acme", turn));
    await inTurn(nerve, 1, ["PATCH", PREFERENCES], json(200, { ...defaults, navigation_project_limit: 0 }));
    await inTurn(nerve, 2, ["PATCH", PREFERENCES], json(200, defaults));
    await until(() => second.settled, "the second turn");
    expect([nerve.calls[1]?.body, nerve.calls[2]?.body]).toEqual([
      { navigation_project_limit: 0 },
      { navigation_project_limit: 10 },
    ]);
  });

  it("fails without sending a change while it has no settings of the workspace", async () => {
    const { nerve, store } = await preferencesStore();
    const changed = track(store.updatePreferences("acme", to({ navigation_control_preference: "TABBED" })));
    await until(() => changed.settled, "the change");
    expect(changed).toMatchObject({ settled: true, error: new Error("Workspace settings not found") });
    expect(nerve.calls).toHaveLength(0);
  });

  it("fetches the settings while a change is out: a fetch does not wait for it", async () => {
````
````old web/apps/web/core/store/workspace/preferences.store.test.ts
      { send: () => store.updatePreferences("acme", { navigation_project_limit: 3 }), request: ["PATCH", PREFERENCES] },
````
````new web/apps/web/core/store/workspace/preferences.store.test.ts
      {
        send: () => store.updatePreferences("acme", to({ navigation_project_limit: 3 })),
        request: ["PATCH", PREFERENCES],
      },
````
````old web/apps/web/core/store/workspace/preferences.store.test.ts
    await until(() => nerve.calls.length === 2, "the refetch");
    const changed = track(store.updatePreferences("acme", { navigation_control_preference: "TABBED" }));
````
````new web/apps/web/core/store/workspace/preferences.store.test.ts
    await until(() => nerve.calls.length === 2, "the refetch");
    const changed = track(store.updatePreferences("acme", to({ navigation_control_preference: "TABBED" })));
````

`web/apps/web/core/store/workspace/preferences.store.ts`（修改，6 处）：

````old web/apps/web/core/store/workspace/preferences.store.ts
import { WorkspacePreferencesService } from "@/services/workspace/workspace-preferences.service";

````
````new web/apps/web/core/store/workspace/preferences.store.ts
import { WorkspacePreferencesService } from "@/services/workspace/workspace-preferences.service";

/** A change of the caller's settings in a workspace: what it changes of the settings it is made to. */
export type PreferencesChange = (held: WorkspacePreferences) => WorkspacePreferencesUpdate;

````
````old web/apps/web/core/store/workspace/preferences.store.ts
  updatePreferences: (workspaceSlug: string, data: WorkspacePreferencesUpdate) => Promise<WorkspacePreferences>;
````
````new web/apps/web/core/store/workspace/preferences.store.ts
  updatePreferences: (workspaceSlug: string, change: PreferencesChange) => Promise<WorkspacePreferences>;
````
````old web/apps/web/core/store/workspace/preferences.store.ts
   * @description changes the settings data names; the store then has nerve's answer, all of them, once it has
   * fetched them. Fails, changing nothing, when nerve refuses.
````
````new web/apps/web/core/store/workspace/preferences.store.ts
   * @description changes the caller's settings in a workspace: change is made, in the change's turn, to the settings
   * nerve last answered, so that a change asked for before the one before it is answered builds on that one's answer
   * (v0 design 7.7); the store then has nerve's answer. Fails, changing nothing, when nerve refuses; and without
   * sending, when the store has no settings of the workspace (none fetched yet, or of a workspace not on his list).
````
````old web/apps/web/core/store/workspace/preferences.store.ts
  updatePreferences = (workspaceSlug: string, data: WorkspacePreferencesUpdate): Promise<WorkspacePreferences> =>
````
````new web/apps/web/core/store/workspace/preferences.store.ts
  updatePreferences = (workspaceSlug: string, change: PreferencesChange): Promise<WorkspacePreferences> =>
````
````old web/apps/web/core/store/workspace/preferences.store.ts
      const preferences = await this.service.update(workspaceSlug, data);
````
````new web/apps/web/core/store/workspace/preferences.store.ts
      const preferences = await this.service.update(workspaceSlug, change(this.held(workspaceSlug)));
````
````old web/apps/web/core/store/workspace/preferences.store.ts
      return preferences;
    });
````
````new web/apps/web/core/store/workspace/preferences.store.ts
      return preferences;
    });

  /** The caller's settings in the workspace as the store has them; fails when it has none. */
  private held(workspaceSlug: string): WorkspacePreferences {
    const preferences = this.getPreferences(workspaceSlug);
    if (!preferences) throw new Error("Workspace settings not found");
    return preferences;
  }
````

- [ ] **Step 2: 导航设置、对话框和按钮**

`web/apps/web/core/components/navigation/project-navigation-dialog.tsx`（修改，14 处）：

````old web/apps/web/core/components/navigation/project-navigation-dialog.tsx
import { useState } from "react";
````
````new web/apps/web/core/components/navigation/project-navigation-dialog.tsx
import { useId, useState } from "react";
````
````old web/apps/web/core/components/navigation/project-navigation-dialog.tsx
// hooks
````
````new web/apps/web/core/components/navigation/project-navigation-dialog.tsx
// hooks
import { countOf } from "@/hooks/navigation-preferences";
````
````old web/apps/web/core/components/navigation/project-navigation-dialog.tsx
};

export const ProjectNavigationDialog = observer(function ProjectNavigationDialog(props: TProjectNavigationDialogProps) {
````
````new web/apps/web/core/components/navigation/project-navigation-dialog.tsx
};

/** Keeps out of the count field the keys a number's other characters are typed with: e, E, +, - and the point. */
function blockNonDigits(e: React.KeyboardEvent<HTMLInputElement>) {
  if (["e", "E", "+", "-", "."].includes(e.key)) e.preventDefault();
}

export const ProjectNavigationDialog = observer(function ProjectNavigationDialog(props: TProjectNavigationDialogProps) {
````
````old web/apps/web/core/components/navigation/project-navigation-dialog.tsx
  // store hooks
  const {
    preferences: projectPreferences,
    updateNavigationMode,
    updateShowLimitedProjects,
    updateLimitedProjectsCount,
  } = useProjectNavigationPreferences();
````
````new web/apps/web/core/components/navigation/project-navigation-dialog.tsx
  const countId = useId();
````
````old web/apps/web/core/components/navigation/project-navigation-dialog.tsx
  // local state for limited projects count input
  const [projectCountInput, setProjectCountInput] = useState(projectPreferences.limitedProjectsCount.toString());
````
````new web/apps/web/core/components/navigation/project-navigation-dialog.tsx
  // store hooks: the caller's settings as nerve gave them, and their change (use-navigation-preferences.ts)
  const { preferences: projectPreferences, changeNavigation } = useProjectNavigationPreferences();
````
````old web/apps/web/core/components/navigation/project-navigation-dialog.tsx
  // Prevent typing invalid characters in number input
  // oxlint-disable-next-line unicorn/consistent-function-scoping
  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    // Block: e, E, +, -, .
    if (["e", "E", "+", "-", "."].includes(e.key)) {
      e.preventDefault();
    }
  };
````
````new web/apps/web/core/components/navigation/project-navigation-dialog.tsx
  // the count field's text while the caller edits it, sent once he leaves the field; otherwise the field shows the
  // settings' count, as they arrive
  const [countDraft, setCountDraft] = useState<string | null>(null);
  const countText = countDraft ?? projectPreferences.limitedProjectsCount.toString();
  const countValid = countOf(countText) !== undefined;
````
````old web/apps/web/core/components/navigation/project-navigation-dialog.tsx
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
````
````new web/apps/web/core/components/navigation/project-navigation-dialog.tsx
  const sendCount = () => {
    if (countDraft === null) return;
    const count = countOf(countDraft);
    setCountDraft(null);
    if (count !== undefined) void changeNavigation({ limitedProjectsCount: count });
````
````old web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                    onChange={() => updateNavigationMode("ACCORDION")}
````
````new web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                    onChange={() => void changeNavigation({ navigationMode: "ACCORDION" })}
````
````old web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                    onChange={() => updateNavigationMode("TABBED")}
````
````new web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                    onChange={() => void changeNavigation({ navigationMode: "TABBED" })}
````
````old web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                    onCheckedChange={updateShowLimitedProjects}
````
````new web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                    onCheckedChange={() => void changeNavigation({ limitToggled: true })}
````
````old web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                        <label className="w-full text-11 text-secondary">{t("enter_number_of_projects")}</label>
                        <input
````
````new web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                        <label htmlFor={countId} className="w-full text-11 text-secondary">
                          {t("enter_number_of_projects")}
                        </label>
                        <input
                          id={countId}
````
````old web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                          value={projectCountInput}
                          onKeyDown={handleKeyDown}
                          onChange={(e) => handleProjectCountChange(e.target.value)}
````
````new web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                          value={countText}
                          onKeyDown={blockNonDigits}
                          onChange={(e) => setCountDraft(e.target.value.replace(/\D/g, ""))}
                          onBlur={sendCount}
````
````old web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                            parseInt(projectCountInput) >= 1
````
````new web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                            countValid
````
````old web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                      {parseInt(projectCountInput) < 1 && projectCountInput !== "" && (
````
````new web/apps/web/core/components/navigation/project-navigation-dialog.tsx
                      {!countValid && countText !== "" && (
````

`web/apps/web/core/components/sidebar/sidebar-wrapper.tsx`（修改，3 处）：

````old web/apps/web/core/components/sidebar/sidebar-wrapper.tsx
import { useOutsideClickDetector } from "@nerve/hooks";
````
````new web/apps/web/core/components/sidebar/sidebar-wrapper.tsx
import { useOutsideClickDetector } from "@nerve/hooks";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/sidebar/sidebar-wrapper.tsx
  const [isProjectNavDialogOpen, setIsProjectNavDialogOpen] = useState(false);
````
````new web/apps/web/core/components/sidebar/sidebar-wrapper.tsx
  const [isProjectNavDialogOpen, setIsProjectNavDialogOpen] = useState(false);
  const { t } = useTranslation();
````
````old web/apps/web/core/components/sidebar/sidebar-wrapper.tsx
                  onClick={() => setIsProjectNavDialogOpen(true)}
````
````new web/apps/web/core/components/sidebar/sidebar-wrapper.tsx
                  onClick={() => setIsProjectNavDialogOpen(true)}
                  aria-label={t("aria_labels.projects_sidebar.project_navigation")}
````

`web/apps/web/core/hooks/navigation-preferences.test.ts`（整个文件，92 行）：

````whole web/apps/web/core/hooks/navigation-preferences.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import type { WorkspacePreferences, WorkspacePreferencesUpdate } from "@nerve/api-client";
import type { TProjectNavigationPreferences } from "@nerve/types";
import type { TProjectNavigationChange } from "./navigation-preferences";
import { countOf, navigationOf, preferencesChangeOf } from "./navigation-preferences";

// The sidebar's project navigation and the caller's settings in a workspace (M3 design 3.18), both ways, and the
// count the dialog's field gives.

const views: { settings: string; preferences?: WorkspacePreferences; shown: TProjectNavigationPreferences }[] = [
  {
    settings: "a limit of 3, as an accordion",
    preferences: { navigation_control_preference: "ACCORDION", navigation_project_limit: 3 },
    shown: { navigationMode: "ACCORDION", limitedProjectsCount: 3, showLimitedProjects: true },
  },
  {
    settings: "a limit of 0, in tabs",
    preferences: { navigation_control_preference: "TABBED", navigation_project_limit: 0 },
    shown: { navigationMode: "TABBED", limitedProjectsCount: 10, showLimitedProjects: false },
  },
  {
    settings: "none yet, taken to be nerve's defaults",
    shown: { navigationMode: "ACCORDION", limitedProjectsCount: 10, showLimitedProjects: true },
  },
];

/** The settings nerve last answered: a limit of 7, in tabs. */
const sevenTabbed: WorkspacePreferences = { navigation_control_preference: "TABBED", navigation_project_limit: 7 };
/** The settings nerve last answered: every project, as an accordion. */
const all: WorkspacePreferences = { navigation_control_preference: "ACCORDION", navigation_project_limit: 0 };
const changes: {
  does: string;
  change: TProjectNavigationChange;
  held: WorkspacePreferences;
  sent: WorkspacePreferencesUpdate;
}[] = [
  {
    does: "the mode",
    change: { navigationMode: "ACCORDION" },
    held: sevenTabbed,
    sent: { navigation_control_preference: "ACCORDION" },
  },
  {
    does: "a turn of the limit, where the settings limit the projects: off, a limit of 0",
    change: { limitToggled: true },
    held: sevenTabbed,
    sent: { navigation_project_limit: 0 },
  },
  {
    does: "a turn of the limit, where the settings show every project: on, nerve's default count",
    change: { limitToggled: true },
    held: all,
    sent: { navigation_project_limit: 10 },
  },
  {
    does: "the count the caller set",
    change: { limitedProjectsCount: 4 },
    held: sevenTabbed,
    sent: { navigation_project_limit: 4 },
  },
];

describe("navigationOf", () => {
  it.each(views)("shows what the caller's settings say: $settings", ({ preferences, shown }) => {
    expect(navigationOf(preferences)).toEqual(shown);
  });
});

describe("preferencesChangeOf", () => {
  it.each(changes)(
    "sends only the setting a change changes, made to the settings nerve last answered: $does",
    ({ change, held, sent }) => {
      expect(preferencesChangeOf(change)(held)).toEqual(sent);
    }
  );
});

describe("countOf", () => {
  it.each([
    { draft: "3", count: 3 },
    { draft: "007", count: 7 },
    { draft: "0", count: undefined },
    { draft: "", count: undefined },
  ])("limits the sidebar to the number of a draft's digits, when 1 or more: '$draft'", ({ draft, count }) => {
    expect(countOf(draft)).toBe(count);
  });
});
````

`web/apps/web/core/hooks/navigation-preferences.ts`（整个文件，59 行）：

````whole web/apps/web/core/hooks/navigation-preferences.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { NavigationControlPreference, WorkspacePreferences } from "@nerve/api-client";
import type { TProjectNavigationPreferences } from "@nerve/types";
// store
import type { PreferencesChange } from "@/store/workspace/preferences.store";

/** One change of the sidebar's project navigation: its mode, a turn of its limit, or the count it limits to. */
export type TProjectNavigationChange =
  | { navigationMode: NavigationControlPreference }
  | { limitToggled: true }
  | { limitedProjectsCount: number };

/** nerve's defaults: the caller's settings in a workspace until he changes one (M3 design 3.18). */
const NERVE_DEFAULTS: WorkspacePreferences = {
  navigation_control_preference: "ACCORDION",
  navigation_project_limit: 10,
};

/**
 * The sidebar's project navigation as the caller's settings in a workspace say (M3 design 3.18): a limit of 0 shows
 * every project, and the count is then nerve's default one, which turning the limit on starts from. Until the
 * settings arrive they are taken to be nerve's defaults, which are the settings of one who has changed none: his
 * sidebar does not change as they arrive.
 */
export function navigationOf(preferences: WorkspacePreferences = NERVE_DEFAULTS): TProjectNavigationPreferences {
  const limit = preferences.navigation_project_limit;
  return {
    navigationMode: preferences.navigation_control_preference,
    limitedProjectsCount: limit > 0 ? limit : NERVE_DEFAULTS.navigation_project_limit,
    showLimitedProjects: limit > 0,
  };
}

/**
 * The change of the caller's settings that a change of the sidebar's project navigation is, made in its turn to the
 * settings nerve last answered (v0 design 7.7), not to those the sidebar shows as it is asked for. A turn of the limit
 * is decided by those settings too: where they limit the projects it turns the limit off, a limit of 0; where they do
 * not, it turns it on, to the count they give (nerve's default one). Two quick turns end where they began.
 */
export function preferencesChangeOf(change: TProjectNavigationChange): PreferencesChange {
  return (held) => {
    if ("navigationMode" in change) return { navigation_control_preference: change.navigationMode };
    if ("limitToggled" in change) {
      const shown = navigationOf(held);
      return { navigation_project_limit: shown.showLimitedProjects ? 0 : shown.limitedProjectsCount };
    }
    return { navigation_project_limit: change.limitedProjectsCount };
  };
}

/** The count a draft of the dialog's count field limits the sidebar to: the number its digits make, when 1 or more. */
export function countOf(draft: string): number | undefined {
  const count = Number.parseInt(draft, 10);
  return count >= 1 ? count : undefined;
}
````

`web/apps/web/core/hooks/use-navigation-preferences.test.ts`（新文件，85 行）：

````file web/apps/web/core/hooks/use-navigation-preferences.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { WorkspacePreferences } from "@nerve/api-client";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { useProjectNavigationPreferences } from "./use-navigation-preferences";

// What the sidebar shows of the caller's settings in the address's workspace, and what its project navigation dialog
// sends (M3 design 3.18, 7.5): the hook runs as a plain function, with stand-ins for the store's settings and their
// change, which nerve answers when the test says. Its session is fake-tab.ts's.

const page = vi.hoisted(() => ({ settings: new Map<string, WorkspacePreferences>(), updatePreferences: vi.fn() }));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "acme" }) }));
vi.mock("./store/use-workspace", () => ({
  useWorkspace: () => ({
    preferences: {
      getPreferences: (slug: string) => page.settings.get(slug),
      updatePreferences: page.updatePreferences,
    },
  }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** The caller's settings in acme: a limit of 3, in tabs. */
const tabbed: WorkspacePreferences = { navigation_control_preference: "TABBED", navigation_project_limit: 3 };

beforeEach(() => {
  signedIn();
  page.settings.clear();
  page.updatePreferences.mockReset();
  page.updatePreferences.mockResolvedValue(tabbed);
  toasts.length = 0;
});

describe("useProjectNavigationPreferences", () => {
  it("shows the caller's settings in the address's workspace, and nerve's defaults until they arrive", () => {
    expect(useProjectNavigationPreferences().preferences).toEqual({
      navigationMode: "ACCORDION",
      limitedProjectsCount: 10,
      showLimitedProjects: true,
    });
    page.settings.set("acme", tabbed);
    expect(useProjectNavigationPreferences().preferences).toEqual({
      navigationMode: "TABBED",
      limitedProjectsCount: 3,
      showLimitedProjects: true,
    });
  });

  it("sends a change to the address's workspace, made in its turn to the settings nerve last answered", async () => {
    page.settings.set("acme", tabbed);
    await useProjectNavigationPreferences().changeNavigation({ limitToggled: true });
    expect(page.updatePreferences.mock.calls).toEqual([["acme", expect.any(Function)]]);
    // nerve answered every project to the turn before it: the sidebar showed a limit of 3 as this one was asked for
    const answered: WorkspacePreferences = { navigation_control_preference: "TABBED", navigation_project_limit: 0 };
    expect(page.updatePreferences.mock.calls[0]?.[1](answered)).toEqual({ navigation_project_limit: 10 });
    expect(toasts).toEqual([]);
  });

  it("shows nerve's reason when it refuses a change", async () => {
    page.updatePreferences.mockRejectedValueOnce(refusal(422, "validation_failed"));
    await useProjectNavigationPreferences().changeNavigation({ limitedProjectsCount: 3_000_000_000 });
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.validation_failed" }]);
  });

  it.each(lateSettlings)(
    "says nothing of a change that $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const change = heldChange<undefined>();
      page.updatePreferences.mockReturnValueOnce(change.sent);
      const made = useProjectNavigationPreferences().changeNavigation({ navigationMode: "TABBED" });
      switchAccount();
      settle(change);
      await made;
      expect(toasts).toEqual([]);
    }
  );
});
````

`web/apps/web/core/hooks/use-navigation-preferences.ts`（整个文件，42 行）：

````whole web/apps/web/core/hooks/use-navigation-preferences.ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "react-router";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { TProjectNavigationPreferences } from "@nerve/types";
// lib
import { errorMessageKey } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";
// local imports
import type { TProjectNavigationChange } from "./navigation-preferences";
import { navigationOf, preferencesChangeOf } from "./navigation-preferences";
import { useWorkspace } from "./store/use-workspace";

/**
 * The sidebar's project navigation in the address's workspace, as the caller's settings there say (M3 design 3.18,
 * 7.5), and its change: made in its turn to the settings nerve last answered (navigation-preferences.ts), and followed
 * only while the tab stays in the session it was sent in (M3 design 7.1): a refusal's reason is said in a toast, and
 * nothing once another tab has moved this one to another account.
 */
export function useProjectNavigationPreferences(): {
  preferences: TProjectNavigationPreferences;
  changeNavigation: (change: TProjectNavigationChange) => Promise<void>;
} {
  const { workspaceSlug = "" } = useParams();
  const { t } = useTranslation();
  const {
    preferences: { getPreferences, updatePreferences },
  } = useWorkspace();
  return {
    preferences: navigationOf(getPreferences(workspaceSlug)),
    changeNavigation: (change) =>
      followInSession(() => updatePreferences(workspaceSlug, preferencesChangeOf(change)), {
        failed: (error) =>
          setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) }),
      }),
  };
}
````

- [ ] **Step 3: 首页的导览、类型和文案**

`web/apps/web/core/components/home/root.test.tsx`（新文件，78 行）：

````file web/apps/web/core/components/home/root.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { WorkspaceHomeView } from "./root";

// What the workspace's home writes as its tour ends (M3 design 7.1, 7.5): the home renders on the server, the tour a
// stand-in that keeps how it ends, and the test ends it as the tour would; the profile's store is a stand-in too,
// which nerve answers when the test says. The session is fake-tab.ts's.

const page = vi.hoisted((): { complete: (() => void) | undefined; updateTourCompleted: ReturnType<typeof vi.fn> } => ({
  complete: undefined,
  updateTourCompleted: vi.fn(),
}));
vi.mock("@/components/onboarding/tour/root", () => ({
  TourRoot: (props: { onComplete: () => void }) => {
    page.complete = props.onComplete;
    return null;
  },
}));
vi.mock("./home-body", () => ({ HomeBody: () => null }));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => ({ data: undefined }),
  useUserProfile: () => ({ data: { is_tour_completed: false }, updateTourCompleted: page.updateTourCompleted }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Opens the home of one who has not ended the tour, and gives how the tour ends. */
function opened(): () => void {
  renderToStaticMarkup(<WorkspaceHomeView />);
  if (!page.complete) throw new Error("the home showed no tour");
  return page.complete;
}

beforeEach(() => {
  signedIn();
  page.complete = undefined;
  page.updateTourCompleted.mockReset();
  page.updateTourCompleted.mockResolvedValue({});
  toasts.length = 0;
});

describe("WorkspaceHomeView", () => {
  it("writes the tour's end in the caller's profile", async () => {
    opened()();
    await pageSettled();
    expect(page.updateTourCompleted).toHaveBeenCalledTimes(1);
    expect(toasts).toEqual([]);
  });

  it("shows nerve's reason when it refuses the tour's end", async () => {
    page.updateTourCompleted.mockRejectedValueOnce(refusal(503, "server_busy"));
    opened()();
    await pageSettled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.server_busy" }]);
  });

  it.each(lateSettlings)(
    "says nothing of the tour's end when it $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const change = heldChange<undefined>();
      page.updateTourCompleted.mockReturnValueOnce(change.sent);
      opened()();
      switchAccount();
      settle(change);
      await pageSettled();
      expect(toasts).toEqual([]);
    }
  );
});
````

`web/apps/web/core/components/home/root.tsx`（修改，5 处）：

````old web/apps/web/core/components/home/root.tsx
// nerve imports
````
````new web/apps/web/core/components/home/root.tsx
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
````
````old web/apps/web/core/components/home/root.tsx
import { TourRoot } from "@/components/onboarding/tour/root";
````
````new web/apps/web/core/components/home/root.tsx
import { TourRoot } from "@/components/onboarding/tour/root";
// lib
import { errorMessageKey } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";
````
````old web/apps/web/core/components/home/root.tsx
  const { data: currentUserProfile, updateTourCompleted } = useUserProfile();
````
````new web/apps/web/core/components/home/root.tsx
  const { data: currentUserProfile, updateTourCompleted } = useUserProfile();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/home/root.tsx
  const handleTourCompleted = async () => {
    try {
      await updateTourCompleted();
    } catch (error) {
      console.error("Error updating tour completed", error);
    }
  };
````
````new web/apps/web/core/components/home/root.tsx
  // the tour ends once nerve has its end in the profile; a refusal's reason is said, and the tour stays. Followed only
  // in the session the end was sent in (M3 design 7.1)
  const completeTour = () =>
    void followInSession(() => updateTourCompleted(), {
      failed: (error) =>
        setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) }),
    });
````
````old web/apps/web/core/components/home/root.tsx
          <TourRoot onComplete={handleTourCompleted} />
````
````new web/apps/web/core/components/home/root.tsx
          <TourRoot onComplete={completeTour} />
````

`web/packages/i18n/src/locales/en/accessibility.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/accessibility.json
      "close_project_menu": "Close project menu"
````
````new web/packages/i18n/src/locales/en/accessibility.json
      "close_project_menu": "Close project menu",
      "project_navigation": "Project navigation"
````

`web/packages/i18n/src/locales/zh-CN/accessibility.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/accessibility.json
      "close_project_menu": "关闭项目菜单"
````
````new web/packages/i18n/src/locales/zh-CN/accessibility.json
      "close_project_menu": "关闭项目菜单",
      "project_navigation": "项目导航"
````

`web/packages/types/src/navigation-preferences.ts`（修改，1 处）：

````old web/packages/types/src/navigation-preferences.ts

export const DEFAULT_PROJECT_PREFERENCES: TProjectNavigationPreferences = {
  navigationMode: "ACCORDION",
  showLimitedProjects: false,
  limitedProjectsCount: 10,
};

````
````new web/packages/types/src/navigation-preferences.ts

````

- [ ] **Step 4: 静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/components/onboarding/steps/team/links.tsx"
````
````new .oxlintrc.json
        "web/apps/web/core/components/onboarding/steps/team/links.tsx",
        "web/apps/web/core/hooks/use-navigation-preferences.test.ts",
        "web/apps/web/core/components/home/root.test.tsx"
````

- [ ] **Step 5: 端到端**

`e2e/stories/workspace/w8-navigation-preferences.spec.ts`（修改，4 处）：

````old e2e/stories/workspace/w8-navigation-preferences.spec.ts
import {
  createWorkspace,
````
````new e2e/stories/workspace/w8-navigation-preferences.spec.ts
import {
  createProject,
  createWorkspace,
````
````old e2e/stories/workspace/w8-navigation-preferences.spec.ts
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/workspace/w8-navigation-preferences.spec.ts
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { answerTo, holdAnswer, registerOnboarded, sentTo } from "../../fixtures/settings-pages";
````
````old e2e/stories/workspace/w8-navigation-preferences.spec.ts
// W8, the project navigation's settings (M3 design 2, 3.18). The page
// version comes with the sidebar's "project navigation" dialog (P9).
````
````new e2e/stories/workspace/w8-navigation-preferences.spec.ts
// W8, the project navigation's settings (M3 design 2, 3.18): through the API, and through the sidebar's "project
// navigation" dialog.
````
````old e2e/stories/workspace/w8-navigation-preferences.spec.ts
  await expectPreferences(db, slug, strangerEmail, null);
});

````
````new e2e/stories/workspace/w8-navigation-preferences.spec.ts
  await expectPreferences(db, slug, strangerEmail, null);
});

test("W8 (page): in his workspace's sidebar the caller makes the project navigation tabs and limits it to 3 projects, the count sent once, as he leaves its field; so it stays after a refresh; a count nerve refuses is said, and the sidebar keeps its 3", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo);
  const tokens = await registerOnboarded(api, email);
  const slug = slugFor(testInfo);
  await createWorkspace(api, tokens.access_token, { name: "Acme", slug });
  await Promise.all(
    ["ALPHA", "BETA", "GAMMA", "DELTA"].map((identifier) =>
      createProject(api, tokens.access_token, slug, { name: identifier, identifier })
    )
  );
  const path = `/api/v0/me/workspaces/${slug}/preferences`;
  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  await page.goto(`/${slug}`);

  // The home's tour comes first, which he skips: the page writes its end in his profile.
  const skipped = await sentTo(page, "PATCH", "/api/v0/me/profile", () =>
    page.getByRole("button", { name: "No thanks, I will explore it myself" }).click()
  );
  expect([skipped.answer.status(), skipped.body]).toEqual([200, { is_tour_completed: true }]);
  await expect(page.getByRole("button", { name: "Take a Product Tour" })).toHaveCount(0);

  // nerve's defaults: up to 10 projects, as an accordion, so the sidebar shows the 4.
  const sidebar = page.getByRole("complementary", { name: "Main sidebar" });
  const projects = sidebar.locator('[id^="sidebar-"][id$="-JOINED"]');
  await expect(projects).toHaveCount(4);
  const dialog = page.getByRole("dialog");
  const tabs = dialog.getByRole("radio", { name: /Tabbed Navigation/ });
  const limited = dialog.getByRole("checkbox", { name: "Show limited projects on sidebar" });
  const count = dialog.getByLabel("Enter number of projects");
  await sidebar.getByRole("button", { name: "Project navigation" }).click();
  await expect(limited).toBeChecked();
  await expect(count).toHaveValue("10");

  const tabbed = await sentTo(page, "PATCH", path, () => tabs.click());
  expect([tabbed.answer.status(), tabbed.body]).toEqual([200, { navigation_control_preference: "TABBED" }]);
  await expect(tabs).toBeChecked();
  // Every project, then a limit again, in two quick turns: the second is asked for before nerve answers the first, and
  // made to that answer (v0 design 7.7), so they end where they began, at nerve's default count.
  const release = await holdAnswer(page, "PATCH", path);
  const turns: unknown[] = [];
  await page.route(`**${path}`, async (route) => {
    if (route.request().method() === "PATCH") turns.push(route.request().postDataJSON());
    await route.fallback();
  });
  await limited.click();
  await expect.poll(() => turns.length).toBe(1);
  await limited.click();
  await release();
  await expect.poll(() => turns).toEqual([{ navigation_project_limit: 0 }, { navigation_project_limit: 10 }]);
  await expect(limited).toBeChecked();
  await expect(count).toHaveValue("10");
  // He types 30 and corrects it to 3: the count goes once, as he leaves the field.
  const limit = await sentTo(page, "PATCH", path, async () => {
    await count.fill("30");
    await count.press("Backspace");
    await count.blur();
  });
  expect([limit.answer.status(), limit.body]).toEqual([200, { navigation_project_limit: 3 }]);
  await expect(count).toHaveValue("3");
  await dialog.getByRole("button", { name: "Close" }).click();
  await expect(projects).toHaveCount(3);
  const settings: WorkspacePreferences = { navigation_control_preference: "TABBED", navigation_project_limit: 3 };
  await expectPreferences(db, slug, email, settings);

  // After a refresh: 3 projects, and the dialog shows his settings, its count nerve's, not one taken as it mounted.
  await page.reload();
  await expect(projects).toHaveCount(3);
  await sidebar.getByRole("button", { name: "Project navigation" }).click();
  await expect(tabs).toBeChecked();
  await expect(count).toHaveValue("3");

  // A count beyond nerve's bounds: nerve refuses it, the page says why, and the field and the sidebar keep 3.
  const beyond = await answerTo(page, "PATCH", path, async () => {
    await count.fill("3000000000");
    await count.blur();
  });
  expect(beyond.status()).toBe(422);
  await expect(page.getByText("Some fields are not valid.")).toBeVisible();
  await expect(count).toHaveValue("3");
  await expect(projects).toHaveCount(3);
  await expectPreferences(db, slug, email, settings);
  expect(watch.apiRequests.filter((request) => request === `PATCH ${path}`)).toHaveLength(5);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[`422 PATCH ${path}`], [], []]);
  // Two loads: the first and the refresh.
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING],
    errors: ["Failed to load resource: the server responded with a status of 422 (Unprocessable Entity)"],
  });
});

````

- [ ] **Step 6: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、3 个例外，没有命中；web 的 oxlint 359 条，等于上限。

Run: `make knip`
Expected: 通过（`DEFAULT_PROJECT_PREFERENCES` 不再被引用）。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 91 个全部通过。

- [ ] **Step 7: 提交**

```bash
git add .oxlintrc.json e2e/stories/workspace/w8-navigation-preferences.spec.ts web/apps/web/core/components/home/root.test.tsx web/apps/web/core/components/home/root.tsx web/apps/web/core/components/navigation/project-navigation-dialog.tsx web/apps/web/core/components/sidebar/sidebar-wrapper.tsx web/apps/web/core/hooks/navigation-preferences.test.ts web/apps/web/core/hooks/navigation-preferences.ts web/apps/web/core/hooks/use-navigation-preferences.test.ts web/apps/web/core/hooks/use-navigation-preferences.ts web/apps/web/core/store/workspace/preferences.store.test.ts web/apps/web/core/store/workspace/preferences.store.ts web/packages/i18n/src/locales/en/accessibility.json web/packages/i18n/src/locales/zh-CN/accessibility.json web/packages/types/src/navigation-preferences.ts
```
```bash
git commit -m "feat(M3/P9): a navigation change is made in its turn to nerve's latest settings; the dialog sends the count once

The workspace preferences store takes a change as a function of the
settings it holds, applied in the queue's turn to nerve's latest answer;
the sidebar's changes are made that way: the limit's switch is a turn,
off where those settings limit the projects and on, to their count,
where they do not, so two quick turns end where they began. Until the settings arrive the sidebar shows
nerve's defaults. The dialog sends the count once, as its field is left,
shows the settings' count otherwise, and labels its field; the button
that opens it has a name. Navigation changes and the tour's end are
followed in their session and say nerve's reason for a refusal. The
new tests join the non-null override. W8's page version, two quick
turns of the switch among it.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p9.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T8.1` | store 把修改作用于提出修改时它持有的设置（E5） | `preferences.store.test.ts` | vitest |
| `T8.2` | store 没有这个工作区的设置时照样发出修改，作用于 nerve 的默认值 | `preferences.store.test.ts` | vitest |
| `T8.3` | 限制项目数的开关按 nerve 的默认值决定打开或关闭，而不是按它作用的设置（E5） | `navigation-preferences.test.ts`、`preferences.store.test.ts`、故事 W8 | vitest；端到端 |
| `T8.4` | 对话框把修改（开关的一次也在其中）作用于提出修改时侧边栏显示的设置（页面上的 E5） | `use-navigation-preferences.test.ts` | vitest |
| `T8.5` | 设置到达之前侧边栏显示全部项目（P28，Plane） | `navigation-preferences.test.ts`、`use-navigation-preferences.test.ts` | vitest |
| `T8.6` | 导航设置的修改被拒绝时进了控制台，页面不说（P28） | `use-navigation-preferences.test.ts`、故事 W8 | vitest；端到端 |
| `T8.7` | 对话框的修改不论会话都跟进（P28） | `use-navigation-preferences.test.ts` | vitest |
| `T8.8` | 数量随每次按键发出（P28，Plane） | 故事 W8 | 端到端 |
| `T8.9` | 数量框显示挂载时的数量，那时设置还没到（P28，Plane） | 故事 W8 | 端到端 |
| `T8.10` | 数量 0 也发出 | `navigation-preferences.test.ts` | vitest |
| `T8.11` | 首页的导览结束而 nerve 不知道 | `root.test.tsx`、故事 W8 | vitest；端到端 |
| `T8.12` | 首页导览的结束被拒绝时什么都不说（Plane：`console.error`） | `root.test.tsx` | vitest |
| `T8.13` | 首页导览的结束不论会话都跟进 | `root.test.tsx` | vitest |
| `T8.14` | 对话框的开关发出点击时侧边栏显示的开或关（预检的 M3，P8b 的 P14 在页面上） | 故事 W8 | 端到端 |
| `T8.15` | 限制项目数的开关每次都打开 | `navigation-preferences.test.ts`、`preferences.store.test.ts`、故事 W8 | vitest；端到端 |
| `T8.16` | 导航 hook 的测试（范围内）用非空断言 | oxlint（`check:lint`） | 静态 |

---

### Task 9: 停用被拒绝的原因在弹窗里，停用的成功只在它结束了标签页的会话时跟进；W9 的页面版本

**Files:**
- Create: `web/apps/web/core/components/account/deactivate-account-modal.test.tsx`
- Modify: `.oxlintrc.json`、`e2e/stories/identity/a12-deactivate.spec.ts`、`e2e/stories/workspace/w9-deactivation.spec.ts`、`web/apps/web/core/components/account/deactivate-account-modal.tsx`、`web/apps/web/core/store/user/index.test.ts`、`web/apps/web/core/store/user/index.ts`

**Interfaces:**
- Produces（spec 2.9；M3 设计 2 的 W9，3.7，3.9，7.1，7.5；P6 评审留给页面的一侧）：
  - `DeactivateAccountModal`：nerve 拒绝停用时（只剩一个管理员的工作区或项目，409），原因（`errorMessageKey`）写在弹窗里（`role="alert"`），弹窗不关，可以再确认或取消；关闭、再次确认时清掉。原来是提示（toast），它不属于弹窗。原因是弹窗自己的状态，不另加会话的核对：另一个标签页以另一个账户登录时，认证包装层在新账户到达之前显示加载，页面连同弹窗卸载（spec 第 3 节）。停用成功时，只在它结束了标签页的会话时提示"账户已停用"并关闭；另一个标签页在它发出之后把这个标签页换到了别的账户时，什么都不说（原来照样提示，显示在那个账户的页面上）。这里不能用 `followInSession`：停用成功时自己结束标签页的会话，`sessionGuard()` 的核对随之为假。
  - `UserStore.deactivateAccount(): Promise<boolean>`（M2 的 store，原来是 `Promise<void>`）：发出时读标签页的会话，nerve 停用之后结束它（`tokenManager.endSession(loginId)`），交回 `endSession` 的回答：结束了它为真；标签页的记录已不是那个会话的（另一个标签页退出了，或换到了别的账户）为假。
- e2e：A12 被拒绝的那一段改读弹窗里的原因（原来读提示）。

- 静态检查（总体设计 7.7）：`.oxlintrc.json` 的非空断言的范围加上 `deactivate-account-modal.test.tsx`。

**Tests:**
- vitest：`deactivate-account-modal.test.tsx`（新文件；弹窗在服务端渲染，按钮和弹窗是 `fake-controls.ts` 的替身，store 的停用是替身）：`says the account is deactivated, and closes, once the deactivation ended the tab's session`；`says nothing, and stays, when it ended no session of the tab's: another tab had moved it to another account`；`stays open, with no toast, when nerve refuses`（弹窗里的原因由 W9 核对：服务端渲染看不到之后才设的状态）。`core/store/user/index.test.ts` 加 `UserStore.deactivateAccount` 的 `ends the session it was sent in, and resolves $ended when $when`（`it.each` 两行；`api-client` 的替身改为记下 `endSession` 的调用；停用在外时标签页换了会话，store 结束的仍是发出时的那个）。
- 端到端：`W9 (page): the only admin of a workspace with another member deactivates from the general page: the dialog says to make another admin first, and nothing changes; once there is another, the deactivation ends his membership and the page is back at sign-in`（409 的原因在弹窗里，账户的状态和成员关系不变；取消再打开，原因不在了；把成员改成管理员之后 204，页面在登录页，说账户已停用，成员关系结束）。

- [ ] **Step 1: store 的停用交回它是否结束了标签页的会话**

`web/apps/web/core/store/user/index.test.ts`（修改，3 处）：

````old web/apps/web/core/store/user/index.test.ts
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
````
````new web/apps/web/core/store/user/index.test.ts
import { FakeNerve, json, noContent, problem } from "@/lib/auth/fake-nerve";
````
````old web/apps/web/core/store/user/index.test.ts
// last answer is what nerve holds. The store gets its session's client from RootStore; the token manager it
// imports is not used here.
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
````
````new web/apps/web/core/store/user/index.test.ts
// last answer is what nerve holds. The store gets its session's client from RootStore; of the token manager it
// imports, the deactivation reads the tab's session and ends it (tab).
const tab = vi.hoisted(() => ({
  state: { loginId: "x" },
  endSession: vi.fn<(loginId: string | undefined) => Promise<boolean>>(),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: tab, publicClient: {} }));
````
````old web/apps/web/core/store/user/index.test.ts
});

// The projects the pages offer for a new work item, cycle or module: those of the address's workspace in which the
````
````new web/apps/web/core/store/user/index.test.ts
});

// The deactivation ends the session it was sent in, read as it is sent: the tab may follow another tab's sign-in while
// it is out (M3 design 7.1). Its answer says whether it ended it, which the dialog follows.
describe("UserStore.deactivateAccount", () => {
  it.each([
    { ended: true, when: "the tab's record is still that session's" },
    { ended: false, when: "another tab has moved this one to another account" },
  ])("ends the session it was sent in, and resolves $ended when $when", async ({ ended }) => {
    tab.state.loginId = "x";
    tab.endSession.mockReset();
    tab.endSession.mockResolvedValue(ended);
    const nerve = new FakeNerve();
    const deactivated = track(new UserStore(fakeRoot({}), nerve.client()).deactivateAccount());
    await until(() => nerve.calls.length === 1, "the deactivation");
    expect([nerve.calls[0]?.method, nerve.calls[0]?.path]).toEqual(["POST", "/api/v0/me/deactivate"]);
    // the tab moves while the deactivation is out: the store ends the session it was sent in
    tab.state.loginId = "y";
    nerve.calls[0]?.answer(noContent());
    await until(() => deactivated.settled, "the deactivation's answer");

    expect([tab.endSession.mock.calls, deactivated.value]).toEqual([[["x"]], ended]);
  });
});

// The projects the pages offer for a new work item, cycle or module: those of the address's workspace in which the
````

`web/apps/web/core/store/user/index.ts`（修改，4 处）：

````old web/apps/web/core/store/user/index.ts
  deactivateAccount: () => Promise<void>;
````
````new web/apps/web/core/store/user/index.ts
  deactivateAccount: () => Promise<boolean>;
````
````old web/apps/web/core/store/user/index.ts
   * tab's sign-in while the request is out
   * @returns {Promise<void>}
````
````new web/apps/web/core/store/user/index.ts
   * tab's sign-in while the request is out. Resolves whether it ended that session: false when the tab's record was
   * no longer that session's (another tab signed out, or moved this one to another account, whose page it then is)
   * @returns {Promise<boolean>}
````
````old web/apps/web/core/store/user/index.ts
  deactivateAccount = async (): Promise<void> => {
````
````new web/apps/web/core/store/user/index.ts
  deactivateAccount = async (): Promise<boolean> => {
````
````old web/apps/web/core/store/user/index.ts
    await tokenManager.endSession(loginId);
````
````new web/apps/web/core/store/user/index.ts
    return tokenManager.endSession(loginId);
````

- [ ] **Step 2: 弹窗**

`web/apps/web/core/components/account/deactivate-account-modal.test.tsx`（新文件，60 行）：

````file web/apps/web/core/components/account/deactivate-account-modal.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { pageSettled } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { DeactivateAccountModal } from "./deactivate-account-modal";

// What the deactivation dialog does with nerve's answer (M3 design 7.1, 7.5, W9): the dialog renders on the server,
// with stand-ins for its modal and buttons (fake-controls.ts), and the test confirms through the confirm button. The
// user store's deactivation is a stand-in, which resolves whether it ended the tab's session.

const store = vi.hoisted(() => ({ deactivateAccount: vi.fn<() => Promise<boolean>>() }));
vi.mock("@/hooks/store/user", () => ({ useUser: () => ({ deactivateAccount: store.deactivateAccount }) }));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Renders the open dialog and confirms it, as its confirm button would; gives onClose once the dialog has followed. */
async function confirm() {
  const onClose = vi.fn();
  renderToStaticMarkup(<DeactivateAccountModal isOpen onClose={onClose} />);
  shown.buttons.find((button) => button.variant === "error-fill")?.onClick?.({ preventDefault: () => {} });
  await pageSettled();
  return onClose;
}

beforeEach(() => {
  store.deactivateAccount.mockReset();
  toasts.length = 0;
  emptyShown();
});

describe("DeactivateAccountModal", () => {
  it("says the account is deactivated, and closes, once the deactivation ended the tab's session", async () => {
    store.deactivateAccount.mockResolvedValueOnce(true);
    const onClose = await confirm();
    expect(toasts).toEqual([{ type: "success", title: "toast.success", message: "account_deactivated" }]);
    expect(onClose.mock.calls).toHaveLength(1);
  });

  it("says nothing, and stays, when it ended no session of the tab's: another tab had moved it to another account", async () => {
    store.deactivateAccount.mockResolvedValueOnce(false);
    const onClose = await confirm();
    expect([toasts, onClose.mock.calls]).toEqual([[], []]);
  });

  // the reason the dialog then shows is W9's to check: the render does not show what is set after it
  it("stays open, with no toast, when nerve refuses", async () => {
    store.deactivateAccount.mockRejectedValueOnce(refusal(409, "workspace.sole_admin"));
    const onClose = await confirm();
    expect([toasts, onClose.mock.calls]).toEqual([[], []]);
  });
});
````

`web/apps/web/core/components/account/deactivate-account-modal.tsx`（修改，6 处）：

````old web/apps/web/core/components/account/deactivate-account-modal.tsx
  const [isDeactivating, setIsDeactivating] = useState(false);
````
````new web/apps/web/core/components/account/deactivate-account-modal.tsx
  const [isDeactivating, setIsDeactivating] = useState(false);
  // nerve's reason for refusing the deactivation (an i18n key), which the dialog says until it closes or confirms again
  const [refusal, setRefusal] = useState<string | undefined>(undefined);
````
````old web/apps/web/core/components/account/deactivate-account-modal.tsx
    setIsDeactivating(false);
````
````new web/apps/web/core/components/account/deactivate-account-modal.tsx
    setIsDeactivating(false);
    setRefusal(undefined);
````
````old web/apps/web/core/components/account/deactivate-account-modal.tsx
    setIsDeactivating(true);
````
````new web/apps/web/core/components/account/deactivate-account-modal.tsx
    setIsDeactivating(true);
    setRefusal(undefined);
````
````old web/apps/web/core/components/account/deactivate-account-modal.tsx
      .then(() => {
        // The session has ended: the sign-in page takes over (AuthenticationWrapper), and shows this.
````
````new web/apps/web/core/components/account/deactivate-account-modal.tsx
      .then((endedHere) => {
        // The deactivation ended the tab's session: the sign-in page takes over (AuthenticationWrapper), and shows
        // this. It ended none when another tab had moved this one to another account meanwhile, whose page this is
        // then, and hears nothing of it (M3 design 7.1).
        if (!endedHere) return;
````
````old web/apps/web/core/components/account/deactivate-account-modal.tsx
      .catch((error: unknown) => {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("toast.error"),
          message: t(errorMessageKey(error)),
        });
      })
````
````new web/apps/web/core/components/account/deactivate-account-modal.tsx
      // nerve's reason, such as the only admin's of a workspace or a project (W9). The dialog is a page's of the
      // session the deactivation was sent in: another tab's sign-in of another account unmounts it, the wrapper
      // waiting for that account (authentication-wrapper.tsx), so a refusal answered after says nothing there.
      .catch((error: unknown) => setRefusal(errorMessageKey(error)))
````
````old web/apps/web/core/components/account/deactivate-account-modal.tsx
              </p>
````
````new web/apps/web/core/components/account/deactivate-account-modal.tsx
              </p>
              {refusal && (
                <p role="alert" className="mt-4 pr-4 text-14 font-medium text-danger-primary">
                  {t(refusal)}
                </p>
              )}
````

- [ ] **Step 3: 静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/components/home/root.test.tsx"
````
````new .oxlintrc.json
        "web/apps/web/core/components/home/root.test.tsx",
        "web/apps/web/core/components/account/deactivate-account-modal.test.tsx"
````

- [ ] **Step 4: 端到端**

`e2e/stories/identity/a12-deactivate.spec.ts`（修改，1 处）：

````old e2e/stories/identity/a12-deactivate.spec.ts
  // The toast says why, by the problem's code; the confirmation stays open, to confirm again or cancel; the page
  // stays signed in, on the general page; nothing changed.
  await expect(page.getByText("Something went wrong on the server. Please try again.")).toBeVisible();
````
````new e2e/stories/identity/a12-deactivate.spec.ts
  // The confirmation says why, by the problem's code, and stays open, to confirm again or cancel; the page stays
  // signed in, on the general page; nothing changed.
  await expect(page.getByRole("dialog").getByRole("alert")).toHaveText(
    "Something went wrong on the server. Please try again."
  );
````

`e2e/stories/workspace/w9-deactivation.spec.ts`（修改，4 处）：

````old e2e/stories/workspace/w9-deactivation.spec.ts
  addProjectMembers,
````
````new e2e/stories/workspace/w9-deactivation.spec.ts
  addProjectMembers,
  changeRole,
````
````old e2e/stories/workspace/w9-deactivation.spec.ts
import { accountOf, expectDeactivated, tokensOf } from "../../fixtures/assert/identity";
import { expectMembership } from "../../fixtures/assert/workspace";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/workspace/w9-deactivation.spec.ts
import { accountOf, accountStateOf, expectDeactivated, tokensOf } from "../../fixtures/assert/identity";
import { expectMembership } from "../../fixtures/assert/workspace";
import { signInPath } from "../../fixtures/auth-pages";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { answerTo, registerOnboarded } from "../../fixtures/settings-pages";
````
````old e2e/stories/workspace/w9-deactivation.spec.ts
// command. The page's is P9's.
````
````new e2e/stories/workspace/w9-deactivation.spec.ts
// command, and the page's, from the general page's dialog.
````
````old e2e/stories/workspace/w9-deactivation.spec.ts
  expect(await others(), "every row of anyone else").toEqual(othersBefore);
});

````
````new e2e/stories/workspace/w9-deactivation.spec.ts
  expect(await others(), "every row of anyone else").toEqual(othersBefore);
});

test("W9 (page): the only admin of a workspace with another member deactivates from the general page: the dialog says to make another admin first, and nothing changes; once there is another, the deactivation ends his membership and the page is back at sign-in", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const email = emailFor(testInfo, "admin");
  const tokens = await registerOnboarded(api, email);
  const memberEmail = emailFor(testInfo, "member");
  const member = await registerOnboarded(api, memberEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, tokens.access_token, { name: "Acme", slug });
  await inviteAndAccept(api, tokens.access_token, slug, { email: memberEmail, token: member.access_token }, 15);
  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  await page.goto("/settings/profile/general");
  await expect(page.getByRole("button", { name: "Deactivate account" })).toBeVisible();
  // once the page has its session: its first refresh writes the session's row
  const before = await accountStateOf(db, email);
  const dialog = page.getByRole("dialog");
  const confirmed = () =>
    answerTo(page, "POST", "/api/v0/me/deactivate", () => dialog.getByRole("button", { name: "Confirm" }).click());

  // The only admin of Acme, which has a member: nerve refuses, the dialog says why, and nothing changes.
  await page.getByRole("button", { name: "Deactivate account" }).click();
  expect((await confirmed()).status()).toBe(409);
  await expect(dialog.getByRole("alert")).toHaveText(
    "The workspace would be left without an admin. Make another member an admin first."
  );
  expect(await accountStateOf(db, email)).toEqual(before);
  await expectMembership(db, slug, email, { role: 20, is_active: true });
  // Cancelled and opened again, the dialog no longer says it.
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await page.getByRole("button", { name: "Deactivate account" }).click();
  await expect(dialog.getByRole("button", { name: "Confirm" })).toBeVisible();
  await expect(dialog.getByRole("alert")).toHaveCount(0);

  // He makes the member an admin: the deactivation goes through, his membership ends, and the page is at sign-in.
  await changeRole(api, tokens.access_token, slug, await accountId(api, member.access_token), 20);
  expect((await confirmed()).status()).toBe(204);
  await expect(page).toHaveURL(signInPath("/settings/profile/general"));
  await expect(page.getByText("Your account is deactivated.")).toBeVisible();
  await expectMembership(db, slug, email, { role: 20, is_active: false });
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    ["409 POST /api/v0/me/deactivate"],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING],
    errors: ["Failed to load resource: the server responded with a status of 409 (Conflict)"],
  });
});

````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 64 条规则、3 个例外，没有命中；web 的 oxlint 359 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 92 个全部通过。

- [ ] **Step 6: 提交**

```bash
git add .oxlintrc.json e2e/stories/identity/a12-deactivate.spec.ts e2e/stories/workspace/w9-deactivation.spec.ts web/apps/web/core/components/account/deactivate-account-modal.test.tsx web/apps/web/core/components/account/deactivate-account-modal.tsx web/apps/web/core/store/user/index.test.ts web/apps/web/core/store/user/index.ts
```
```bash
git commit -m "feat(M3/P9): the deactivation dialog says why nerve refused it, and follows a success only in the session it ended

A refused deactivation, such as a workspace's only admin's, is said in
the dialog, which stays open to confirm again or cancel, and forgets it
as it closes or confirms again. The user store's deactivation resolves
whether it ended the tab's session; the dialog says the account is
deactivated, and closes, only then: once another tab has moved this one
to another account, that account's page hears nothing of it. W9's page
version; A12 reads the reason in the dialog.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p9.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T9.1` | nerve 对停用的拒绝显示在弹窗外的提示里（M2） | `deactivate-account-modal.test.tsx`、故事 W9、故事 A12 | vitest；端到端 |
| `T9.2` | 弹窗关闭再打开，仍显示上一次的拒绝 | 故事 W9 | 端到端 |
| `T9.3` | 不论标签页此时在哪个会话里，弹窗都说账户已停用并关闭（M2） | oxlint（`check:lint`）、`deactivate-account-modal.test.tsx` | 静态；vitest |
| `T9.4` | store 的停用不论是否结束了标签页的会话，都说结束了 | `index.test.ts` | vitest |
| `T9.5` | 停用弹窗的测试（范围内）用非空断言 | oxlint（`check:lint`） | 静态 |

---

### Task 10: 工作区列表的唯一取数；个人设置自己取列表；`WorkspaceAuthWrapper` 的界面

**Files:**
- Create: `web/apps/web/core/hooks/use-workspaces-fetch.ts`
- Modify: `.oxlintrc.json`、`e2e/stories/smoke/s2-web-app.spec.ts`、`tools/keywords.json`、`web/apps/web/core/components/onboarding/root.tsx`、`web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx`、`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts`、`web/apps/web/core/layouts/auth-layout/workspace-wrapper.test.tsx`、`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`、`web/apps/web/core/lib/use-landing.ts`、`web/packages/i18n/src/locales/en/workspace.json`、`web/packages/i18n/src/locales/zh-CN/workspace.json`

**Interfaces:**
- Produces（spec 2.10；M3 设计 7.1，7.5；P8a 评审的 M7；M2 收尾第 13 节）：
  - `core/hooks/use-workspaces-fetch.ts`：`useWorkspacesFetch(wanted = true)`，调用者的工作区列表的唯一取数（`useSessionSWR(wanted ? ["WORKSPACES"] : null, () => fetchWorkspaces())`）。落点（`use-landing.ts`，只在要决定落点时取）、工作区的页面（`use-workspace-fetch.ts`）、新手引导（`onboarding/root.tsx`，Task 6 的取数）都改用它；原来三处各写一遍键和取数。
  - 个人设置的侧边栏（`settings/profile/sidebar/workspace-options.tsx`）挂载时经 `useWorkspacesFetch()` 取列表（M7：工作区之外没有包装层取它，直接打开 `/settings/profile/*` 时侧边栏的工作区列表和命令面板的工作区菜单是空的）。
  - 静态检查：`useWorkspacesFetch` 是唯一的取数，由关键词守卫的新规则 `workspaces-list-fetch` 看住（`tools/keywords.json`：web 的源文件中，store 自己、测试和测试的替身之外，只有 `use-workspaces-fetch.ts` 能写 `fetchWorkspaces`；预检 L10）；`.oxlintrc.json` 的 `no-restricted-imports` 的范围加上 `use-workspaces-fetch.ts`、`workspace-options.tsx`，非空断言的范围加上 `use-workspaces-fetch.ts`（总体设计 7.7）。
  - `WorkspaceAuthWrapper`：退出登录失败的提示经 `t("auth.sign_out.toast.error.*")`，"找不到工作区"的界面经 `t("workspace_not_found.*")`（en、zh-CN 的 `workspace.json` 新增五个键；原来是写死的英文）。`workspace-wrapper.test.tsx` 改为核对键（测试的 `t` 交回键）。

**Tests:**
- vitest：`workspace-wrapper.test.tsx` 的"找不到工作区"一条（`it.each` 的两行：有别的工作区、没有工作区）改读键；退出失败的提示只是换成已有的一对键（其余几处退出登录的失败提示用的同一对），没有新的测试。`use-landing.test.ts` 不改：不需要决定落点时不取列表（`wanted` 为假）由它的既有用例守着（变异 `T10.2`）。
- 端到端：S2 加 `S2: a member of a workspace opens his profile's settings directly: they ask for his workspaces, as the app does, which their sidebar lists, and no more`（安静的控制台之后再读一次请求）。

- [ ] **Step 1: 列表的取数**

`web/apps/web/core/components/onboarding/root.tsx`（修改，3 处）：

````old web/apps/web/core/components/onboarding/root.tsx
import { useUserProfile } from "@/hooks/store/user";
````
````new web/apps/web/core/components/onboarding/root.tsx
import { useUserProfile } from "@/hooks/store/user";
import { useWorkspacesFetch } from "@/hooks/use-workspaces-fetch";
````
````old web/apps/web/core/components/onboarding/root.tsx
import { followInSession } from "@/lib/in-session";
import { useSessionSWR } from "@/lib/use-session-swr";
````
````new web/apps/web/core/components/onboarding/root.tsx
import { followInSession } from "@/lib/in-session";
````
````old web/apps/web/core/components/onboarding/root.tsx
  const { workspaces, fetchWorkspaces } = useWorkspace();
  const listed = useSessionSWR(["WORKSPACES"], () => fetchWorkspaces());
````
````new web/apps/web/core/components/onboarding/root.tsx
  const { workspaces } = useWorkspace();
  const listed = useWorkspacesFetch();
````

`web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx`（修改，2 处）：

````old web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx
import { useWorkspace } from "@/hooks/store/use-workspace";
````
````new web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useWorkspacesFetch } from "@/hooks/use-workspaces-fetch";
````
````old web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx
  const { workspaces } = useWorkspace();
````
````new web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx
  const { workspaces } = useWorkspace();
  // outside a workspace no wrapper has fetched the caller's workspaces: the settings fetch them as they mount (M3
  // design 7.1), for this list and power-K's
  useWorkspacesFetch();
````

`web/apps/web/core/hooks/use-workspaces-fetch.ts`（新文件，20 行）：

````file web/apps/web/core/hooks/use-workspaces-fetch.ts
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

/**
 * The one fetch of the caller's workspaces (M3 design 7.1), which the landing, a workspace's pages, the onboarding and
 * the profile's settings share: the session's list, fetched as each mounts; nothing while wanted is false.
 */
export function useWorkspacesFetch(wanted = true): SWRResponse<Workspace[] | undefined> {
  const { fetchWorkspaces } = useWorkspace();
  return useSessionSWR(wanted ? ["WORKSPACES"] : null, () => fetchWorkspaces());
}
````

`web/apps/web/core/lib/use-landing.ts`（修改，4 处）：

````old web/apps/web/core/lib/use-landing.ts
import { useUserProfile } from "@/hooks/store/user";
````
````new web/apps/web/core/lib/use-landing.ts
import { useUserProfile } from "@/hooks/store/user";
import { useWorkspacesFetch } from "@/hooks/use-workspaces-fetch";
````
````old web/apps/web/core/lib/use-landing.ts
import { landingPath } from "@/lib/landing";
import { useSessionSWR } from "@/lib/use-session-swr";
````
````new web/apps/web/core/lib/use-landing.ts
import { landingPath } from "@/lib/landing";
````
````old web/apps/web/core/lib/use-landing.ts
  const { workspaces, fetchWorkspaces } = useWorkspace();
````
````new web/apps/web/core/lib/use-landing.ts
  const { workspaces } = useWorkspace();
````
````old web/apps/web/core/lib/use-landing.ts
  const listed = useSessionSWR(lands ? ["WORKSPACES"] : null, () => fetchWorkspaces());
````
````new web/apps/web/core/lib/use-landing.ts
  const listed = useWorkspacesFetch(lands);
````

- [ ] **Step 2: 包装层的界面和文案**

`web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts`（修改，3 处）：

````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
import { useWorkspaceMembersFetch } from "@/hooks/use-workspace-members-fetch";
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
import { useWorkspaceMembersFetch } from "@/hooks/use-workspace-members-fetch";
import { useWorkspacesFetch } from "@/hooks/use-workspaces-fetch";
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
    workspaces,
    fetchWorkspaces,
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
    workspaces,
````
````old web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
  const listed = useSessionSWR(["WORKSPACES"], () => fetchWorkspaces());
````
````new web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts
  const listed = useWorkspacesFetch();
````

`web/apps/web/core/layouts/auth-layout/workspace-wrapper.test.tsx`（修改，2 处）：

````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.test.tsx
    { has: "others", hasWorkspaces: true, links: ["Go Home", "Visit Profile"], not: "Create new workspace" },
    { has: "none", hasWorkspaces: false, links: ["Create new workspace"], not: "Go Home" },
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.test.tsx
    {
      has: "others",
      hasWorkspaces: true,
      links: ["workspace_not_found.go_home", "workspace_not_found.visit_profile"],
      not: "workspace_not_found.create_workspace",
    },
    {
      has: "none",
      hasWorkspaces: false,
      links: ["workspace_not_found.create_workspace"],
      not: "workspace_not_found.go_home",
    },
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.test.tsx
    expect(markup).toContain("Workspace not found");
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.test.tsx
    expect(markup).toContain("workspace_not_found.title");
````

`web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx`（修改，5 处）：

````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
        title: "Error!",
        message: "Failed to sign out. Please try again.",
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
        title: t("auth.sign_out.toast.error.title"),
        message: t("auth.sign_out.toast.error.message"),
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
            <h3 className="text-center text-16 font-semibold">Workspace not found</h3>
            <p className="text-center text-13 text-secondary">
              No workspace found with the URL. It may not exist or you lack authorization to view it.
            </p>
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
            <h3 className="text-center text-16 font-semibold">{t("workspace_not_found.title")}</h3>
            <p className="text-center text-13 text-secondary">{t("workspace_not_found.description")}</p>
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
                  Go Home
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
                  {t("workspace_not_found.go_home")}
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
                  Visit Profile
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
                  {t("workspace_not_found.visit_profile")}
````
````old web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
                  Create new workspace
````
````new web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx
                  {t("workspace_not_found.create_workspace")}
````

`web/packages/i18n/src/locales/en/workspace.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/workspace.json
    "sign_in": "Sign in"
````
````new web/packages/i18n/src/locales/en/workspace.json
    "sign_in": "Sign in"
  },
  "workspace_not_found": {
    "title": "Workspace not found",
    "description": "No workspace found with the URL. It may not exist or you lack authorization to view it.",
    "go_home": "Go Home",
    "visit_profile": "Visit Profile",
    "create_workspace": "Create new workspace"
````

`web/packages/i18n/src/locales/zh-CN/workspace.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/workspace.json
    "sign_in": "登录"
````
````new web/packages/i18n/src/locales/zh-CN/workspace.json
    "sign_in": "登录"
  },
  "workspace_not_found": {
    "title": "找不到工作区",
    "description": "这个地址没有对应的工作区：它可能不存在，或者您没有权限查看。",
    "go_home": "回到首页",
    "visit_profile": "个人设置",
    "create_workspace": "创建新工作区"
````

- [ ] **Step 3: 静态检查：关键词守卫和 oxlint 的范围**

`.oxlintrc.json`（修改，2 处）：

````old .oxlintrc.json
        "web/apps/web/core/components/onboarding/root.tsx"
````
````new .oxlintrc.json
        "web/apps/web/core/components/onboarding/root.tsx",
        "web/apps/web/core/hooks/use-workspaces-fetch.ts",
        "web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx"
````
````old .oxlintrc.json
        "web/apps/web/core/components/account/deactivate-account-modal.test.tsx"
````
````new .oxlintrc.json
        "web/apps/web/core/components/account/deactivate-account-modal.test.tsx",
        "web/apps/web/core/hooks/use-workspaces-fetch.ts"
````

`tools/keywords.json`（修改，1 处）：

````old tools/keywords.json
          "miss": ["docs/v0/M3-workspace-project/M3-design.md", "e2e/stories/identity/a3-sign-in.spec.ts"]
        }
      }
    }
````
````new tools/keywords.json
          "miss": ["docs/v0/M3-workspace-project/M3-design.md", "e2e/stories/identity/a3-sign-in.spec.ts"]
        }
      }
    },
    {
      "id": "workspaces-list-fetch",
      "phase": "M3/P9",
      "why": "调用者的工作区列表只有一个取数：useWorkspacesFetch（web/apps/web/core/hooks/use-workspaces-fetch.ts，键 [\"WORKSPACES\", loginId]），登录后的落点、工作区包装层、新手引导和个人设置的侧边栏都经它（M3/P9 spec 第 3 节第 14 条）；别处调 store 的 fetchWorkspaces 就是另一个键、另一份取数。store 自己（core/store/workspace/）、测试和测试的替身不算",
      "files": {
        "source": "^web/apps/web/(?:app|core)/(?!hooks/use-workspaces-fetch\\.ts$|store/workspace/|hooks/store/fake-store-hooks\\.ts$)(?!.*\\.test\\.[jt]sx?$).*\\.[jt]sx?$",
        "flags": ""
      },
      "content": {
        "source": "\\bfetchWorkspaces\\b",
        "flags": ""
      },
      "samples": {
        "hit": [
          "  const { workspaces, fetchWorkspaces } = useWorkspace();",
          "  useSessionSWR([\"PROFILE_WORKSPACES\"], () => fetchWorkspaces());"
        ],
        "miss": ["  useWorkspacesFetch();", "  const listed = useWorkspacesFetch(false);"],
        "files": {
          "hit": [
            "web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx",
            "web/apps/web/core/components/onboarding/root.tsx"
          ],
          "miss": [
            "web/apps/web/core/hooks/use-workspaces-fetch.ts",
            "web/apps/web/core/store/workspace/index.ts",
            "web/apps/web/core/hooks/store/fake-store-hooks.ts",
            "web/apps/web/core/lib/store-context.test.ts"
          ]
        }
      }
    }
````

- [ ] **Step 4: 端到端**

`e2e/stories/smoke/s2-web-app.spec.ts`（修改，1 处）：

````old e2e/stories/smoke/s2-web-app.spec.ts
  expect(requests.between(0)).toEqual({ pending: 0, requests: APP.toSorted() });
});

````
````new e2e/stories/smoke/s2-web-app.spec.ts
  expect(requests.between(0)).toEqual({ pending: 0, requests: APP.toSorted() });
});

test("S2: a member of a workspace opens his profile's settings directly: they ask for his workspaces, as the app does, which their sidebar lists, and no more", async ({
  api,
  signedInPage,
}, testInfo) => {
  const tokens = await registerOnboarded(api, emailFor(testInfo));
  await createWorkspace(api, tokens.access_token, { name: "Acme", slug: slugFor(testInfo) });
  const page = await signedInPage(tokens);
  const watch = await watchPage(page);
  const requests = followRequests(page, {});
  await page.goto("/settings/profile/general");
  await expect(page.getByRole("link", { name: "Acme" })).toBeVisible();
  await expect.poll(() => requests.between(0)).toEqual({ pending: 0, requests: APP.toSorted() });
  expect([watch.apiFailures, watch.oldApiRequests, watch.cspViolations, watch.pageErrors]).toEqual([[], [], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
  // Nothing came after the list was whole.
  expect(requests.between(0)).toEqual({ pending: 0, requests: APP.toSorted() });
});

````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 65 条规则、3 个例外，没有命中；web 的 oxlint 359 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 93 个全部通过。

- [ ] **Step 6: 提交**

```bash
git add .oxlintrc.json e2e/stories/smoke/s2-web-app.spec.ts tools/keywords.json web/apps/web/core/components/onboarding/root.tsx web/apps/web/core/components/settings/profile/sidebar/workspace-options.tsx web/apps/web/core/hooks/use-workspaces-fetch.ts web/apps/web/core/layouts/auth-layout/use-workspace-fetch.ts web/apps/web/core/layouts/auth-layout/workspace-wrapper.test.tsx web/apps/web/core/layouts/auth-layout/workspace-wrapper.tsx web/apps/web/core/lib/use-landing.ts web/packages/i18n/src/locales/en/workspace.json web/packages/i18n/src/locales/zh-CN/workspace.json
```
```bash
git commit -m "feat(M3/P9): one fetch of the caller's workspaces, which the profile's settings use too; the wrapper's screens speak through t()

useWorkspacesFetch is the one fetch of the caller's workspaces: the
landing, a workspace's pages and the onboarding share it, and the
profile's settings, which no wrapper serves, fetch the list as they
mount, so their sidebar and power-K's menu list the caller's workspaces.
The workspace wrapper's not-found screen and its sign-out failure speak
through t(). The keyword guard holds the one fetch; its files join the
swr and non-null overrides. S2 opens the profile's settings directly.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p9.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T10.1` | 个人设置显示调用者的工作区而不取列表（P8a 评审的 M7） | 故事 S2 | 端到端 |
| `T10.2` | 落点在不落到任何地方的页面上也取工作区列表 | `use-landing.test.ts` | vitest |
| `T10.3` | 包装层"找不到工作区"的界面不论语言都是英文（Plane） | `workspace-wrapper.test.tsx` | vitest |
| `T10.4` | 工作区列表唯一的取数经 `swr`，不经 `useSessionSWR` | oxlint（`check:lint`） | 静态 |
| `T10.5` | 个人设置的侧边栏经 `swr` 取列表，不经 `useSessionSWR` | oxlint（`check:lint`） | 静态 |
| `T10.6` | 个人设置的侧边栏以自己的键取列表（预检的 PF11） | 关键词守卫 | 静态 |
| `T10.7` | 列表的取数（`use-workspaces-fetch.ts`，范围内）用非空断言 | oxlint（`check:lint`） | 静态 |

---

### Task 11: 文档：README 的"前端"一节和前端改动清单

**Files:**
- Modify: `README.md`、`docs/v0/M3-workspace-project/handoffs/M2-closeout.md`、`docs/v0/frontend-changes.md`

**Interfaces:**
- Produces（spec 2.11；M3 设计 3.20 中 P9 的行；M2 收尾第 2 节的关闭条件之一）：
  - `README.md` 的"前端"一节："M2 中看到的页面"一条改写为"能用的页面（M3/P9 起）"：登录、注册（凭邀请链接）、邀请链接的页面、新手引导、落点、创建工作区、工作区首页、侧边栏、工作区设置的 general 和 members、个人设置；项目的页面在 P10、P11，其余的由 M4–M7 对接。
  - `docs/v0/frontend-changes.md`：3.1 中 M3 一行的状态（工作区的页面已对接）；3.2 "所有处理接口错误的地方"一行写明 P9 改到的页面；新增五行（页面跟进修改的结果、创建工作区、新手引导的邀请成员一步、侧边栏的项目导航设置、停用账户的弹窗），都是"已完成，M3/P9"。
  - `docs/v0/M3-workspace-project/handoffs/M2-closeout.md`：加"处理结果（M3/P9）"：第 1、2 节的页面一侧完成，第 11 节的页面一侧中工作区的部分，第 13 节 P9 的部分（浏览器核对是 C4）；M2 的 `UserStore.deactivateAccount` 改为交回 `Promise<boolean>`（`endSession` 的回答），停用弹窗只在为 `true` 时说已停用（裁定 P3）。
- 控制者的浏览器核对 C1–C5 和评审不在本 plan 中（M3 设计 9.7、12 节 P9 的第 14 个任务）。

**Tests:** 没有新的测试：本 Task 只改文档。

- [ ] **Step 1: 文档**

`README.md`（修改，1 处）：

````old README.md
- **M2 中看到的页面**：登录、注册；新手引导（三步中只有资料一步能用，工作区、邀请两步要用 M3 的接口）；完成了新手引导的账户登录后的落点 `/create-workspace`（创建工作区在 M3）；个人设置的四个标签页（general、preferences、security、api-tokens）。工作区相关的页面和操作还在调用 Plane 的接口，Nerve 返回 404，从 M3 起对接。
````
````new README.md
- **能用的页面**（M3/P9 起）：登录、注册（注册关闭时，凭邀请链接仍可注册）；邀请链接的页面 `/workspace-invitations`（没登录时显示工作区和角色，登录后接受或忽略）；新手引导（资料一步；还没有工作区的人接着创建工作区、邀请成员，发出邀请之后列出每个邀请的链接供复制，v0 不发邮件；已经有工作区的人资料一步之后就完成）；登录后的落点（上次打开的工作区；它不在列表中时，最早创建的那个；一个都没有时 `/create-workspace`）；创建工作区；工作区首页（问候、"快速开始"的引导，第一次进入时的导览）；侧边栏的工作区菜单（切换、创建工作区、退出）和"项目导航"对话框；工作区设置的 general（名称、规模、时区，删除工作区）和 members（成员、改角色、移出、离开；管理员还有邀请：批量邀请、复制链接、改角色、删除）；个人设置的四个标签页（general、preferences、security、api-tokens，停用账户在 general 页）。项目的页面在 M3/P10、P11 对接；工作项、迭代、模块、视图、收藏、通知等页面还在调用 Plane 的接口，Nerve 返回 404，由 M4–M7 对接。
````

`docs/v0/M3-workspace-project/handoffs/M2-closeout.md`（修改，1 处）：

````old docs/v0/M3-workspace-project/handoffs/M2-closeout.md
来源：[M3/P8b spec](../specs/P8b-web-project-data.md) 第 7 节；[M3/P8a spec](../specs/P8a-web-workspace-data.md) 第 7 节。

````
````new docs/v0/M3-workspace-project/handoffs/M2-closeout.md
来源：[M3/P8b spec](../specs/P8b-web-project-data.md) 第 7 节；[M3/P8a spec](../specs/P8a-web-workspace-data.md) 第 7 节。

## 处理结果（M3/P9）

- **第 1 节 邀请与注册**（页面一侧完成）：邀请链接的页面由 `invitationView`（`web/apps/web/core/components/workspace/invitation-view.ts`）决定显示什么：链接缺 id 或令牌、nerve 按链接找不到时无效；没有登录时给登录、注册，两者都带着链接回来；登录之后接受或忽略（`use-invitation-answer.ts`），邮箱不一致时只给退出登录；接受之后写上次打开的工作区再进入。注册凭链接中的邀请（注册关闭时也能注册），登录页和注册页之间的链接保留链接的参数。W5、W6 的页面版本守着（[M3/P9 spec](../specs/P9-web-workspace-pages.md) 2.4、2.7）。
- **第 2 节 登录后的落点与新手引导的取数**（页面一侧完成）：新手引导取调用者的工作区列表，由它决定步骤，已有工作区的人资料一步之后就完成；创建一步与 `/create-workspace` 共用 `useCreateWorkspace`，邀请一步发往刚建的工作区；完成引导不再把列表中的第一个工作区写成上次打开的。W2 的页面版本守着切换、创建、接受之后写的 `last_workspace_id` 和删除、离开之后的落点；README 的"前端"一节写明能用的页面（spec 2.2、2.5、2.6、2.11）。
- **第 11 节 M2 留下的 M3 调用和类型**（页面一侧中工作区的部分完成）：新手引导的创建、邀请两步（spec 2.6）；工作区 general 页的时区随修改发出（spec 2.1）。项目的部分随 P10。
- **第 13 节**（P9 的部分）：`WorkspaceAuthWrapper` 的"找不到工作区"界面和退出登录失败的提示经 `t()`（spec 2.10）；新手引导的邀请一步换成成员页的表单（spec 2.6）。逐条的浏览器核对是 M3/P9 评审的 C4。
- **M2 的 `UserStore.deactivateAccount`**（`web/apps/web/core/store/user/index.ts`，M3/P9 改了签名）：交回 `Promise<boolean>`，即 `tokenManager.endSession(loginId)` 的回答：停用应答时标签页的记录是否仍是发出停用的那个会话（是时结束它）；另一个标签页先退出、或把这个标签页换到别的账户时为 `false`。它唯一的调用方停用弹窗只在为 `true` 时说已停用并关闭；nerve 拒绝时它照旧拒绝（spec 2.9，第 3 节第 3 条）。

仍未处理，状态保持 `open`：第 11 节的页面一侧（P10）；第 13 节等 C4 写进 M3/P9 的评审；第 10、14 节（P10，M3 设计 7.7）；第 12 节等 M3 的收尾。

来源：[M3/P9 spec](../specs/P9-web-workspace-pages.md) 第 7 节。

````

`docs/v0/frontend-changes.md`（修改，3 处）：

````old docs/v0/frontend-changes.md
| 工作区、成员、邀请、项目、项目成员、项目归档、状态、标签、显示设置 | M3 | 进行中：数据层已对接（工作区一侧 M3/P8a，项目一侧 M3/P8b）；页面在 M3/P9–P11 |
````
````new docs/v0/frontend-changes.md
| 工作区、成员、邀请、项目、项目成员、项目归档、状态、标签、显示设置 | M3 | 进行中：数据层已对接（工作区一侧 M3/P8a，项目一侧 M3/P8b）；工作区的页面 M3/P9 已对接，项目的页面在 M3/P10–P11 |
````
````old docs/v0/frontend-changes.md
| 所有处理接口错误的地方 | 统一按 RFC 9457 的 problem+json 读取 `code`、`title`、`errors`。M2/P4 已改：`ApiError` 和 `unwrap`（`core/lib/api-error.ts`）按生成的 `Problem` 读取；登录页、注册页和安全页的修改密码按它的 `code`、`errors` 显示错误。M2/P5 改完个人设置的其余部分（[M2 设计](M2-auth/M2-design.md) 7.7）和新手引导的资料步骤：general 页的保存、资料步骤和 api-tokens 页的创建把字段错误显示在字段下方（名字的规则只在 nerve，页面只查必填，Plane 的名字校验从 `@nerve/utils` 删除），其余的错误和 preferences 的主题、时区、语言、每周第一天、PAT 的撤销、停用账户、新手引导换步骤时更新资料（`onboarding/root.tsx`）的失败都在提示中，文案按 `code` 取（`core/lib/error-messages.ts` 的 `PROBLEM_MESSAGES`、`fieldErrorKeys`、`errorMessageKey`，M3/P8a 从 `helpers/authentication.helper.ts` 移来）。M3/P8a 起工作区、成员、邀请、工作区的显示设置，M3/P8b 起项目、项目成员、状态、标签、项目的显示设置经生成的客户端，nerve 的错误应答是 `ApiError`：状态和标签的设置页、工作项的三个标签选择器和项目包装层的加入界面在 M3/P8b 已按 `code` 显示错误（`errorMessageKey`），其余页面在 M3/P9–P11；M4–M8 的领域在各自的 M，它们现在还经 Plane 的 axios 基类按 Plane 的错误格式读取 | 错误格式统一 | 进行中 | |
````
````new docs/v0/frontend-changes.md
| 所有处理接口错误的地方 | 统一按 RFC 9457 的 problem+json 读取 `code`、`title`、`errors`。M2/P4 已改：`ApiError` 和 `unwrap`（`core/lib/api-error.ts`）按生成的 `Problem` 读取；登录页、注册页和安全页的修改密码按它的 `code`、`errors` 显示错误。M2/P5 改完个人设置的其余部分（[M2 设计](M2-auth/M2-design.md) 7.7）和新手引导的资料步骤：general 页的保存、资料步骤和 api-tokens 页的创建把字段错误显示在字段下方（名字的规则只在 nerve，页面只查必填，Plane 的名字校验从 `@nerve/utils` 删除），其余的错误和 preferences 的主题、时区、语言、每周第一天、PAT 的撤销、停用账户、新手引导换步骤时更新资料（`onboarding/root.tsx`）的失败都在提示中，文案按 `code` 取（`core/lib/error-messages.ts` 的 `PROBLEM_MESSAGES`、`fieldErrorKeys`、`errorMessageKey`，M3/P8a 从 `helpers/authentication.helper.ts` 移来）。M3/P8a 起工作区、成员、邀请、工作区的显示设置，M3/P8b 起项目、项目成员、状态、标签、项目的显示设置经生成的客户端，nerve 的错误应答是 `ApiError`：状态和标签的设置页、工作项的三个标签选择器和项目包装层的加入界面在 M3/P8b 已按 `code` 显示错误（`errorMessageKey`），工作区的页面在 M3/P9（落点、新手引导、邀请页、注册、创建工作区、首页、侧边栏的项目导航、工作区设置和停用账户的弹窗；nerve 指名的字段错误显示在对应的字段下方），其余页面在 M3/P10–P11；M4–M8 的领域在各自的 M，它们现在还经 Plane 的 axios 基类按 Plane 的错误格式读取 | 错误格式统一 | 进行中 | |
````
````old docs/v0/frontend-changes.md
| 项目一侧的挂载时取数 | 项目包装层的取数由 `useProjectFetch`（`core/layouts/auth-layout/use-project-fetch.ts`）决定：先取项目，nerve 说调用者是有效成员之后才取显示设置、标签、成员和状态；不是成员的看得到的项目显示"加入项目"，看不到的和在另一个工作区的地址下打开的显示"找不到项目"（nerve 对看不到的项目答 404，Plane 的 403 界面删除；项目只在地址的工作区里算数）；工作区包装层取项目列表和工作区的状态，项目角色取自 `Project.member_role`，`project-roles` 的取数删除 | M3 设计 7.1、7.3 | 已完成 | M3/P8b |
````
````new docs/v0/frontend-changes.md
| 项目一侧的挂载时取数 | 项目包装层的取数由 `useProjectFetch`（`core/layouts/auth-layout/use-project-fetch.ts`）决定：先取项目，nerve 说调用者是有效成员之后才取显示设置、标签、成员和状态；不是成员的看得到的项目显示"加入项目"，看不到的和在另一个工作区的地址下打开的显示"找不到项目"（nerve 对看不到的项目答 404，Plane 的 403 界面删除；项目只在地址的工作区里算数）；工作区包装层取项目列表和工作区的状态，项目角色取自 `Project.member_role`，`project-roles` 的取数删除 | M3 设计 7.1、7.3 | 已完成 | M3/P8b |
| 页面跟进修改的结果 | 工作区的页面发出修改之后的跳转、提示和界面变化，只在发出它的会话里进行（`core/lib/in-session.ts` 的 `followInSession`）：另一个标签页换了账户之后，页面已是那个账户的，旧会话的修改不论成败都不再跟进 | M3 设计 7.1 | 已完成 | M3/P9 |
| 创建工作区 | 表单自己的 slug 校验（`@nerve/utils` 的 `validateSlug`、`SLUG_REGEX`）删除，规则只在 nerve：提交时 nerve 说明不能用的原因（已占用、保留、格式不对），显示在 slug 字段下方；slug 字段显示的就是要发出的值（Plane 显示转换后的文字，发出的却是原文）。`/create-workspace` 和新手引导的创建一步共用 `useCreateWorkspace`。创建关闭时两处都说明已关闭、请工作区的管理员给邀请链接；Plane 给实例管理员写信的按钮和信的文案删除（Nerve 没有实例管理员，M2 设计 3.16） | M3 设计 3.10、3.11、7.4 | 已完成 | M3/P9 |
| 新手引导的邀请成员一步 | Plane 的三行邀请表单换成成员页的邀请表单（`InvitationFields`、`useWorkspaceInvitationActions`），发往刚建的工作区；发出之后列出每个邀请的链接供复制（v0 不发邮件）。"创建或加入工作区"的包装组件（`onboarding/steps/workspace/root.tsx`）删除；引导从工作区列表决定步骤（`onboarding-place.ts`），完成引导不再写 `last_workspace_id` | M3 设计 7.4、决策点 2 | 已完成 | M3/P9 |
| 侧边栏的项目导航设置 | `@nerve/types` 的 `DEFAULT_PROJECT_PREFERENCES` 删除：设置到达之前按 nerve 的默认值显示（最多 10 个项目，折叠式）；对话框的修改在轮到它时按 nerve 最近一次的回答算出（总体设计 7.7）：限制项目数的开关按那个回答打开或关闭，连按两下回到原样；数量在输入框失去焦点时发出 | M3 设计 3.18、7.5 | 已完成 | M3/P9 |
| 停用账户的弹窗 | nerve 拒绝的原因（如唯一的管理员）显示在确认弹窗里，不再是提示，弹窗关闭时清掉；停用只在它结束了标签页的会话时才说已停用（另一个标签页先换了账户时，页面已是那个账户的） | M3 设计 7.1、7.5 | 已完成 | M3/P9 |
````

- [ ] **Step 2: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 65 条规则、3 个例外，没有命中；web 的 oxlint 359 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 93 个全部通过。

- [ ] **Step 3: 提交**

```bash
git add README.md docs/v0/M3-workspace-project/handoffs/M2-closeout.md docs/v0/frontend-changes.md
```
```bash
git commit -m "docs(M3/P9): README's frontend section lists the pages that work, the frontend changes list P9's, and the M2 handoff its results

README's frontend section says which pages work from M3/P9 on, where the
project pages and the rest are connected; the frontend changes list says
the workspace pages are connected, which pages show nerve's errors by
their code, and P9's five changes. The M2 handoff records what P9 did
for its sections, and that UserStore.deactivateAccount now says whether
it ended the tab's session.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 只改文档，没有变异。
