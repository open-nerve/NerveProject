# M3/P10 项目的页面 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 项目的页面接上 P8b 的数据层，行为按 M3 设计第 2 节的故事 P1–P5、P8：项目列表和卡片（复制链接、加入）、`ProjectAuthWrapper` 的界面（加入、已归档、找不到）、创建项目、项目设置的 general、features、automations 和 members（添加、改角色、移出、离开；负责人、默认负责人、访客可见全部工作项），归档、恢复和删除（项目设置、已归档的项目页），侧边栏拖动项目的顺序和项目页头的标签栏；M2 收尾交接第 10 节（表情选择器的数据由 Nerve 自己提供）和第 14 节（下拉框和复制到剪贴板）的 M3 部分。页面发出项目一侧的修改之后的跳转、提示和界面变化只在发出它的会话里进行（`followInSession`）；每个请求体由表单编辑的字段构成，类型取自生成的客户端，数字就是数字；每个错误经 `errorMessageKey` 显示（字段下方或提示），没有吞进 `console.error` 的拒绝；请求在路上时弹窗关不掉，按钮忙到跟进做完；被拒绝时弹窗留着，只在成功的跟进中关上。P1–P5、P8 的页面版本，S2 的挂载清单。

**Architecture:** 只改 web、`e2e/`、根目录 `.oxlintrc.json`（非空断言的范围，Task 1–10 各加它的文件）、`tools/keywords.json`（Task 7 的规则 `project-leave`：离开只有一处）、依赖的声明（Task 1：web 应用声明锁文件中已有的 `emojibase-data`）和四份文档（README、前端改动清单、M2 收尾交接和 M1-P4 交接的处理结果）。页面调用 store 的方法，不调 service；会话的每个取数经 `useSessionSWR`。共用的新部分：`web/apps/web/emojibase.ts`（构建把表情选择器的数据写进资源，`EMOJIBASE_URL` 指向它；Task 1）；`@nerve/ui` 的 `CustomSearchSelect` 改成 Popover 的按钮、面板里的 `Combobox`，成员下拉框以它为底，对 M4–M6 的调用方照旧（推迟到指针经过才渲染、手机上不取焦点、列表离窗口边缘 12 像素），面包屑一节一个 Tab 停留点（Task 2）；`core/hooks/use-copy-link.ts`（`useCopyLink`、`useCopyProjectLink`：复制、成功和失败的提示，Task 3）；`core/components/project/use-join-project.ts`（Task 3）；`core/components/project/project-refusal.ts`（`projectRefusal`、`useProjectRefusal`：项目表单的拒绝落到字段下或提示，创建和 general 页共用，Task 4）和 `logo-props.ts`（Task 4）；页面一侧的 hook `useCreateProject`（Task 4）、`useUpdateProjectDetails`（Task 5）、`useProjectMembershipChanges`（Task 7）；`project-roles.ts` 的 `roleChoices`、`canRemove`（Task 7）、`addableRoles`（Task 8）：页面只提供 nerve 允许的（M3 设计 3.5）。数据层改两处：`core/lib/reconciled.ts` 的 `answeredAt`（扩展这份实现，不复制），项目 store 的 `getProjectById` 给出 nerve 最后回答的那一份（P8b 的 F-3，Task 3）；`useProjectFetch` 对已归档的项目给出 `archived`，不取成员才读的（Task 3）。`useTabPreferences` 交回 `{ navigation, changes }`，设置取到之前没有修改（Task 10）。e2e 的共用部分：`watchPage` 记下每个请求的地址，`requestsElsewhere` 给出发往别处的（Task 1）；S2 的挂载清单的列表移进 `e2e/fixtures/mounts.ts`，P2 读同一份（Task 3）。契约、Go 代码和生成的文件都不变。

**Tech Stack:** React 19.2.8、React Router 8.3.0、MobX 6.12.0、SWR 2.4.2、openapi-fetch 0.17.0、Headless UI 2.2、TypeScript 5.8.3、vitest 4.1.11、oxlint 1.51.0、oxfmt 0.35.0、knip 6.37.0、turbo 2.10.11；Node 24、pnpm 11.10.0、Playwright 1.63.0；Go 1.27.1（Go 代码不变）。版本都由 `pnpm-lock.yaml` 固定。

**Spec:** `docs/v0/M3-workspace-project/specs/P10-web-project-pages.md`（上级：`docs/v0/M3-workspace-project/M3-design.md`）

## Global Constraints

- **依赖**：不加任何 npm 包或 Go 模块，不执行 `go get`、`pnpm add`。Task 1 让 web 应用声明 `emojibase-data` 15.3.2：它已在锁文件中（`@tiptap/extension-emoji` 依赖它），web 应用在构建时读它的文件（M2 收尾交接第 10 节）；`pnpm-workspace.yaml` 的 catalog 一行、`web/apps/web/package.json` 的 `devDependencies` 一行、`pnpm-lock.yaml` 的两处 importer 的条目（块中给出），没有新的包，之后执行 `pnpm install --frozen-lockfile`。`git diff --stat 6c1a090a -- server/ api/` 在本 plan 的任何时刻都没有输出；`pnpm-lock.yaml` 只有 Task 1 的两处。
- **每个 Task 提交前**：`make lint-web`（关键词守卫、`tsc`、oxlint 在上限、格式、`en` 与 `zh-CN` 的键一致）、`make knip`、`make test-web`、`make e2e` 都通过。本 plan 不改 Go 代码和接口描述，不生成任何文件，不执行 `make gen`（`make gen-check` 照旧没有差异）。
- **e2e 的数目**：在 worktree 中 `make e2e` 必须全部通过：开始时 93 个；Task 1 起 94、Task 2 起 95、Task 3 起 98、Task 4 起 99、Task 5 起 101、Task 6 起 102、Task 7 起 103、Task 8 起 104、Task 9 起 105、Task 10 起 106。只在不是 git 仓库的副本里，S3 因为构建读不到提交而失败（P4b spec F4），其余照上面的数目减一全部通过。
- **容器**：`make e2e` 用自己的 testcontainers；机器忙时偶尔起不来，等 Docker 空闲之后重跑一次再当作失败。容器测试一次只跑一套。开发库 `nerve-dev-db-1` 不要启动、停止或重建，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`、`nervewiki-*`）和留下的 testcontainers。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合，只读的命令也不行。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。不把副本的 `node_modules` 链接到 worktree 的。
- **规则**（总体设计 7.7，P8a 的裁定 F-1–F-10，P8b 的 PF-M2，P9 的 F-1–F-4）：会话的每个取数经 `useSessionSWR`（不传配置），没有权限时键为 `null`；页面只取它的角色能读的（已归档的项目只读项目本身，Task 3）；修改经 store 的方法，一个接一个，store 写入 nerve 的回答；请求体取决于 store 所持的值时，轮到它发出时从 nerve 最近一次回答算出（侧边栏的移动、标签栏的修改、功能和自动归档的开关）；`SessionChangedError` 不是认证失败；`core/lib/reconciled.ts` 只扩展，不复制（Task 3 的 `answeredAt`）。页面读地址只有一种写法：路由的参数（`useParams`）。角色的名字经 `ROLE_DETAILS[...].i18n_title`（翻译），不用英文的 `ROLE`。页面发出项目一侧的修改（加入、创建、general 的修改、功能和自动归档、改角色、移出、离开、添加成员、成员默认值、归档、恢复、删除、侧边栏的移动、标签栏）之后的跳转、提示和界面变化经 `followInSession` 只在发出它的会话里进行；每一处跟进由它自己的 vitest 守着（换账户之后兑现的 `lateSettlings`）。请求在路上时弹窗关不掉（取消按钮禁用，`ModalCore` 的 `handleClose` 为空，P9 的 F-1），按钮忙到跟进做完（P9 的 F-2），由故事扣住请求核对（`sentHeld`、`closedByEscape`、`enabledWithin`）。每个弹窗一条规则：被拒绝时留着、按钮重新可用，原因在共用的拒绝提示里；只在成功的跟进中关上（经 `followInSession`，spec 第 3 节第 6 条和"预检之后"的 A-M2）。请求体由表单编辑的字段构成，类型是生成的 `ProjectCreate`、`ProjectUpdate`、`ProjectMemberUpdate`、`ProjectMembersAdd`，角色是数字；每个请求体有一个测试钉住它（vitest 或端到端的 `sentTo`、`sentHeld`）。每个错误经 `errorMessageKey`（`useRefusalToast` 的提示）或 `FIELD_ERROR_MESSAGES`（字段下方，`projectRefusal`）显示，被拒绝的修改结束它的加载状态。类型只来自生成的客户端：不写重述契约的类型，没有新的 `as`、`any`、`!`（spec 附录 A.7）。删除的代码删干净（组件、两种语言的文案、工具函数、常量），不加 knip 的忽略、开关或桩。本 plan 写或重写的文件都在约 400 行以内（最终原型上量的：最长的是 `e2e/fixtures/assert/project.ts` 397 行、`e2e/stories/workspace/w3-workspace-settings.spec.ts` 397 行、`e2e/fixtures/api.ts` 390 行、`e2e/stories/project/p1-create-project.spec.ts` 397 行、`e2e/stories/project/p3-project-settings.spec.ts` 399 行、`e2e/stories/project/p5-project-members.spec.ts` 396 行；web 中最长的是 `project/form.tsx` 292 行，新文件中最长的是 `use-update-project-details.test.ts` 143 行）；只为使用方改到的 Plane 文件不变长，一处例外（`@nerve/propel` 的 `emoji-icon-picker/index.ts` 多一行导出，spec 第 3 节第 5 条，裁定 R5；附录 A.6）。
- **oxlint**（M3 设计 7.9，裁定 R3）：有手改的文件（本 plan 的每个 TS 文件都是手改的，没有机械步骤）在它的 Task 提交时没有 oxlint 警告。上限：web 356 → 350（Task 2：`dropdowns/member/` 的六条随 `member-options.tsx` 和旧的写法一起没有了）→ 349（Task 7：确认框的正的 `tabIndex`）；`propel` 16 → 11（Task 1：表情选择器的两个文件的五条，`no-shadow` 四条、`consistent-function-scoping` 一条）；其余包不变。没有新的 oxlint 抑制，删掉两条（Task 5：`project/form.tsx` 的 `react-hooks/exhaustive-deps`；Task 7：`member-list-item.tsx` 的 `promise/always-return`）。
- **注释**：TS 代码、测试、JSON 的说明用英文；中文文档照本 plan 原样。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的）。

  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从 `6c1a090a` 起按顺序核对过全部块：每个 `old` 恰好出现一次（在它之前的块应用之后的文件中），每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同（spec 附录 A.10）。可以用 `node <planapply.mjs> <本 plan> apply <仓库根> <n>` 写入第 n 个 Task 的块，也可以手工照抄。
- **过渡版本**：一些文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改：根目录 `.oxlintrc.json`（Task 1–10，每个 Task 加它的文件）、`web/apps/web/package.json`（Task 1 的依赖，Task 2、7 的上限）、`propel` 的 `emoji-icon-picker/helper.tsx`（Task 1、4）、`project/create/header.tsx`（Task 1、4）、`project/card.tsx`（Task 3、9）、`project-roles.ts` 和它的测试（Task 7、8）、`core/lib/fake-controls.ts`（Task 2、8），以及 `e2e/fixtures/api.ts`（Task 3、8）、`e2e/fixtures/settings-pages.ts`（Task 2、3、5、10）、`e2e/fixtures/project-pages.ts`（Task 7、8）、S2（Task 1、3）、P1（Task 1、2、4）、P3（Task 5、6、8）、P5（Task 7、8）。每个过渡版本都在逐 Task 复现中运行过。
- **变异**：每个 Task 末尾的"变异"表列出：把代码改坏的方式、必须因此失败的检查和它所在的层（静态：`make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest：`make test-web`；端到端：`make e2e` 的故事）。它们在最终的原型上逐个跑过（`$M3TMP/p10tools/mutants_p10.py`，`mut.py` 在它写的每个检查上各跑一次），也在各自 Task 的树上跑过（spec 附录 A.2）；表中标"（Task N 起）"的检查只在第 N 个 Task 加入的测试中才让它失败（在变异自己的 Task 的树上它还活着）。**会话、权限或取数的性质只由评审才能发现的，算缺口**（brief）：表中每一条这类性质都有一个会失败的检查。
- **评审敏感**（M3 设计 12 节 P10 的评审重点）：页面级的副作用只在发出修改的会话里（每个 hook 的 vitest 的 `lateSettlings`；删除项目的组件，9.5）；页面只取角色能读的（已归档的项目、看得到而不是成员的项目只读项目本身：S2、P2）；页面只提供 nerve 允许的角色和移出（Task 7、8 的 `project-roles.ts`）；请求在路上时弹窗关不掉（每个弹窗一个故事的扣住）；表情的数据不发往别处（P1、P3 的 `requestsElsewhere`）。改动这些之前，先照"变异"表确认它在所说的性质去掉之后失败。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录。一个文件由多个 Task 修改时，"Task"一列都列出。

| 文件 | 职责 | Task |
|---|---|---|
| `.oxlintrc.json` | P10 的新模块和测试、改到的 nerve 写的模块和测试加进非空断言的范围，各在加入它们的 Task（P9 spec 2.14 的做法） | 1、2、3、4、5、6、7、8、9、10 |
| `pnpm-workspace.yaml`、`pnpm-lock.yaml` | catalog 的 `emojibase-data` 15.3.2；锁文件中 web 应用的两处 importer 条目（包已在锁文件中） | 1 |
| `web/apps/web/package.json` | `emojibase-data` 的开发依赖（Task 1）；web 的 oxlint 上限 356 → 350（Task 2）→ 349（Task 7） | 1、2、7 |
| `web/apps/web/emojibase.ts`、`web/apps/web/emojibase.test.ts` | Vite 插件：构建把 `emojibase-data` 的 `en/data.json`、`en/messages.json` 写到 `/assets/emojibase/<版本>/`，开发服务器答同样的路径；它的 vitest（地址、版本、内容与表情选择器读的一致） | 1 |
| `web/apps/web/vite.config.ts` | 用这个插件 | 1 |
| `web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx`、`web/packages/propel/src/emoji-icon-picker/index.ts` | `EMOJIBASE_URL`：表情选择器从 Nerve 读数据；导出它；三个插槽的 `...rest`（oxlint） | 1 |
| `web/packages/propel/src/emoji-icon-picker/emoji-picker.tsx`、`web/packages/propel/src/emoji-icon-picker/helper.tsx` | 选择器的按钮收 `ariaLabel`（Task 1）；`toHex` 移到模块（oxlint，Task 1）；`TChangeHandlerProps` 导出（Task 4） | 1、4 |
| `web/packages/propel/package.json` | propel 的 oxlint 上限 16 → 11 | 1 |
| `web/packages/i18n/src/locales/en/accessibility.json`、`web/packages/i18n/src/locales/zh-CN/accessibility.json` | 项目图标的按钮的名字 | 1 |
| `web/apps/web/core/components/project/create/header.tsx` | 图标选择器有名字（Task 1）；图标经 `logoPropsOf`，创建在途时关闭按钮禁用、有名字（Task 4） | 1、4 |
| `web/packages/ui/src/dropdowns/custom-search-select.tsx` | Popover 的按钮（Tab 能到）、面板里的 `Combobox`；打开时搜索框取得焦点（调用方可以不要），Escape 从搜索框也关（输入法组字时不关）；单选选了就关；关上时清空搜索；popper 定位面板，调用方可以给它修饰（整个文件） | 2 |
| `web/packages/ui/src/dropdowns/helper.tsx` | `Placement` 用 popper 的类型；`focusSearchOnOpen`、`popperModifiers` | 2 |
| `web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx` | 一节一个 Tab 停留点：最后一节的标题是 `<span>`，之前一节的标题按钮不在 Tab 顺序里 | 2 |
| `web/apps/web/core/components/dropdowns/member/base.tsx`、`web/apps/web/core/components/dropdowns/member/types.d.ts` | 成员下拉框以 `CustomSearchSelect` 为底（整个文件）；Tab 只停在 `DropdownButton`；调用方推迟时指针经过之前只有按钮，手机上打开时搜索框不取得焦点，列表离窗口边缘 12 像素 | 2 |
| `web/apps/web/core/components/dropdowns/member/base.test.tsx` | 成员下拉框对调用方的约定（新文件） | 2 |
| `web/apps/web/core/components/dropdowns/buttons.tsx` | `DropdownButton` 收 `tabIndex`：成员下拉框在调用方给的位置停一次 | 2 |
| `web/apps/web/core/components/dropdowns/member/member-options.tsx` | 成员下拉框原来的列表（删除） | 2 |
| `web/apps/web/core/hooks/use-copy-link.ts`、`web/apps/web/core/hooks/use-copy-link.test.ts` | `useCopyLink`（复制，成功和失败的提示）、`useCopyProjectLink`（地址的工作区中项目的链接）；它的 vitest | 3 |
| `web/apps/web/core/hooks/use-copy-invitation-link.ts`、`web/apps/web/core/components/workspace/settings/workspace-details.tsx` | 邀请链接、工作区地址的复制经 `useCopyLink` | 3 |
| `web/apps/web/core/components/workspace/sidebar/projects-list.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx`、`web/apps/web/core/components/navigation/use-project-actions.ts` | 侧边栏、扩展侧边栏、项目页头的菜单复制项目的链接经 `useCopyProjectLink` | 3 |
| `web/apps/web/core/components/project/use-join-project.ts`、`web/apps/web/core/components/project/use-join-project.test.ts` | 加入项目：在会话里跟进，拒绝按 `code` 提示；它的 vitest | 3 |
| `web/apps/web/core/components/project/join-project-modal.tsx` | 卡片的加入对话框：加入在途时关不掉（整个文件） | 3 |
| `web/apps/web/core/components/project/card.tsx` | 复制经 `useCopyProjectLink`（Task 3）；已归档的卡片对项目管理员和是工作区管理员的项目成员提供恢复、删除，删除按钮有名字（Task 9） | 3、9 |
| `web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx` | 包装层的三种界面：加入（按钮忙到跟进做完）、已归档（按钮去已归档的项目）、找不到 | 3 |
| `web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`、`web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx` | 包装层按 `useProjectFetch` 的 `kind` 渲染，加入经 `useJoinProject`（整个文件）；它的渲染测试 | 3 |
| `web/apps/web/core/layouts/auth-layout/use-project-fetch.ts`、`web/apps/web/core/layouts/auth-layout/use-project-fetch.test.ts` | 已归档的项目是 `archived`，不取成员才读的 | 3 |
| `web/packages/i18n/src/locales/en/empty-state.json`、`web/packages/i18n/src/locales/zh-CN/empty-state.json` | 已归档项目的界面的文案 | 3 |
| `web/apps/web/core/lib/reconciled.ts` | `answeredAt`：一个值是第几个写入的回答（`Reconciled`、`ReconciledByKey`） | 3 |
| `web/apps/web/core/store/project/project.store.ts`、`web/apps/web/core/store/project/project.store.test.ts` | `getProjectById` 给出 nerve 最后回答的那一份（自己的读取或列表，F-3）；它的测试 | 3 |
| `web/apps/web/core/components/project/project-refusal.ts`、`web/apps/web/core/components/project/project-refusal.test.ts` | 项目表单的拒绝：名称、标识已被占用和字段错误在字段下，其余提示；创建和 general 页共用 | 4 |
| `web/apps/web/core/components/project/logo-props.ts`、`web/apps/web/core/components/project/logo-props.test.ts` | 选择器选的表情或图标成为 nerve 的 `LogoProps` | 4 |
| `web/apps/web/core/components/projects/create/use-create-project.ts`、`web/apps/web/core/components/projects/create/use-create-project.test.ts` | 创建项目：表单的字段构成 `ProjectCreate`，没选负责人不发，在会话里跟进；它的 vitest | 4 |
| `web/apps/web/core/components/projects/create/root.tsx` | 创建表单经 `useCreateProject`，在途时 Escape 不关（整个文件） | 4 |
| `web/apps/web/core/components/projects/create/attributes.tsx`、`web/apps/web/core/components/projects/create/utils.ts` | 负责人从工作区的管理员和成员中选；表单的值的类型、默认值 | 4 |
| `web/apps/web/core/components/project/create-project-modal.tsx`、`web/apps/web/core/components/project/create/common-attributes.tsx`、`web/apps/web/core/components/project/create/project-create-buttons.tsx` | 弹窗的 Escape 只在功能一步；表单的值的类型；创建在途时取消禁用 | 4 |
| `web/packages/utils/src/project.ts`、`web/packages/utils/src/project.test.ts` | 标识符转大写、只留允许的字符（不再有 `.`、`..`）；它的 vitest | 4 |
| `web/apps/web/core/components/project/use-update-project-details.ts`、`web/apps/web/core/components/project/use-update-project-details.test.ts` | general 页的修改：改了的标识先问 nerve，被占用时说在它下面、不发；字段构成 `ProjectUpdate`；在会话里跟进；它的 vitest | 5 |
| `web/apps/web/core/components/project/form.tsx` | general 页的表单（整个文件）：经 `useUpdateProjectDetails`，忙到跟进做完，成员只读 | 5 |
| `web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx` | general 页读地址的项目，每个项目一个表单（`key`） | 5 |
| `web/packages/i18n/src/locales/en/common.json`、`web/packages/i18n/src/locales/zh-CN/common.json` | general 页不再读的四条文案删除 | 5 |
| `web/apps/web/core/components/project/settings/use-feature-toggle.ts`、`web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx` | 功能的开关在会话里跟进，拒绝按 `code` 提示（不再用 `setPromiseToast`）；它的 vitest | 6 |
| `web/apps/web/core/components/automation/auto-archive-automation.tsx`、`web/apps/web/core/components/automation/auto-archive-automation.test.tsx` | 自动归档读地址的项目，开关、时长在会话里跟进；它的 vitest | 6 |
| `web/apps/web/core/components/automation/select-month-modal.tsx`、`web/apps/web/core/components/automation/select-month-modal.test.tsx` | 自定义时长等 nerve 做完才关，在途时关不掉；它的 vitest | 6 |
| `web/apps/web/core/components/project/settings/use-project-membership-changes.ts`、`web/apps/web/core/components/project/settings/use-project-membership-changes.test.ts` | 成员页的改角色、移出、离开：在会话里跟进，离开之后才显示工作区的项目；它的 vitest | 7 |
| `web/apps/web/core/components/project/project-roles.ts`、`web/apps/web/core/components/project/project-roles.test.tsx` | nerve 允许的角色和移出（`roleChoices`、`canRemove`，Task 7）、添加时的角色（`addableRoles`，Task 8）；它们的表 | 7、8 |
| `web/apps/web/core/components/project/settings/member-columns.tsx`、`web/apps/web/core/components/projects/settings/useProjectColumns.tsx` | 成员页的列：角色只提供允许的，名字经 `ROLE_DETAILS` 翻译，自己一行是离开（整个文件）；列算出调用者和每一行 | 7 |
| `web/apps/web/core/components/project/member-list-item.tsx` | 成员行经 `useProjectMembershipChanges`（整个文件） | 7 |
| `web/apps/web/core/components/project/confirm-project-member-remove.tsx`、`web/apps/web/core/components/project/leave-project-modal.tsx` | 移出、离开在途时关不掉；离开弹窗先离开、再跳转 | 7 |
| `tools/keywords.json` | 规则 `project-leave`：store 的 `leaveProject` 只由离开的 hook 调 | 7 |
| `e2e/fixtures/project-pages.ts` | 项目页面的操作：请求、显示的名字、移出和离开（Task 7，新文件）；`shownNameOf`（Task 8） | 7、8 |
| `web/apps/web/core/components/project/add-project-members-modal.tsx`、`web/apps/web/core/components/project/add-project-members-modal.test.tsx` | 添加成员：工作区的有效成员中还不是项目成员的，角色按 `addableRoles`（名字同成员页），在会话里跟进，在途时关不掉；它的 vitest | 8 |
| `web/apps/web/core/components/project/member-list.tsx` | 添加弹窗不再收 `workspaceSlug` | 8 |
| `web/apps/web/core/components/project/project-settings-member-defaults.tsx`、`web/apps/web/core/components/project/project-settings-member-defaults.test.tsx` | 负责人、默认负责人、访客可见：显示 nerve 的回答，在会话里跟进（整个文件）；它的 vitest | 8 |
| `web/apps/web/core/lib/fake-controls.ts` | 搜索选择的替身留下它收到的焦点、popper 的修饰（Task 2）和选项（Task 8） | 2、8 |
| `web/apps/web/core/components/project/delete-project-modal.tsx`、`web/apps/web/core/components/project/delete-project-modal.test.tsx` | 删除项目：按提交的值核对，在会话里跟进，项目自己的页面才回到项目列表，在途时关不掉；它的 vitest（9.5） | 9 |
| `web/apps/web/core/components/project/archive-restore-modal.tsx`、`web/apps/web/core/components/project/archive-restore-modal.test.tsx` | 归档、恢复：在会话里跟进，在途时关不掉（整个文件）；它的 vitest | 9 |
| `web/apps/web/core/components/workspace/sidebar/use-project-drop.ts`、`web/apps/web/core/components/workspace/sidebar/use-project-drop.test.ts` | 侧边栏的移动失败在会话里按 `code` 提示（整个文件）；它的 vitest | 10 |
| `web/apps/web/core/components/navigation/use-tab-preferences.ts`、`web/apps/web/core/components/navigation/use-tab-preferences.test.ts` | 标签栏交回 `{ navigation, changes }`：设置取到之前没有修改；修改在会话里跟进（整个文件）；它的 vitest | 10 |
| `web/apps/web/core/components/navigation/tab-navigation-root.tsx`、`web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx`、`web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx` | 页头的标签和"更多"只在有修改时提供菜单 | 10 |
| `README.md`、`docs/v0/frontend-changes.md`、`docs/v0/M3-workspace-project/handoffs/M2-closeout.md`、`docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md` | "前端"一节的项目页面；前端改动清单；M2 收尾交接、M1-P4 交接的 P10 处理结果 | 11 |
| `e2e/fixtures/browser.ts` | `watchPage` 记下每个请求的地址；`requestsElsewhere`：发往页面之外的源的请求 | 1 |
| `e2e/fixtures/mounts.ts` | 页面加载时的请求清单的列表和 `valued`，从 S2 移来（新文件）；S2、P2 读它 | 3 |
| `e2e/fixtures/settings-pages.ts` | `transitionsEnded`（Task 2）；`closedWithin`（Task 3 的修正轮，`closedByEscape` 经它）；`moveWithinApp`（Task 5）；`shownWithin`（Task 10） | 2、3、5、10 |
| `e2e/fixtures/workspace-pages.ts` | `pickTimeZone`：只用键盘选时区 | 2 |
| `e2e/fixtures/api.ts` | `archiveProject`（Task 3）；`projectMembershipOf`、`changeProject`、`projectMemberWrites`（Task 8） | 3、8 |
| `e2e/fixtures/assert/project.ts` | `projectSettingsOf`：项目存下的设置 | 8 |
| `e2e/stories/project/p1-create-project.spec.ts` | P1 的页面版本：表情选择器（Task 1）、负责人的列表（Task 2）、创建（Task 4） | 1、2、4 |
| `e2e/stories/workspace/w3-workspace-settings.spec.ts` | W3 的页面版本只用键盘选时区 | 2 |
| `e2e/stories/project/p2-visibility.spec.ts` | P2 的页面版本：地址的加入界面、已归档、找不到；卡片的复制和加入；页头的项目面包屑一个 Tab 停留点 | 3 |
| `e2e/stories/smoke/s2-web-app.spec.ts` | 发往别处的请求经 `requestsElsewhere`（Task 1）；挂载清单的列表读 `mounts.ts`，成员、访客、不是成员的人打开项目列表，已归档的项目只读项目本身（Task 3） | 1、3 |
| `e2e/stories/project/p3-project-settings.spec.ts` | P3 的页面版本：general 和不离开路由到另一个项目的 general（Task 5）、功能和自动归档（Task 6）、成员默认值（Task 8，经 API 的被拒绝的添加经 `projectMemberWrites`） | 5、6、8 |
| `e2e/stories/project/p5-project-members.spec.ts` | P5 的页面版本：改角色、移出、离开，被拒绝时确认框、离开弹窗留着（Task 7）；添加成员，成员的选择用 Tab 到达（Task 8） | 7、8 |
| `e2e/stories/project/p4-archive.spec.ts` | P4 的页面版本：归档、恢复、删除 | 9 |
| `e2e/stories/project/p8-project-preferences.spec.ts` | P8 的页面版本：标签栏的默认和"更多"、侧边栏的拖动 | 10 |

---

### Task 1: 表情选择器的数据由 Nerve 提供；P1 的表情选择器

**Files:**
- Create: `web/apps/web/emojibase.test.ts`、`web/apps/web/emojibase.ts`
- Modify: `.oxlintrc.json`、`e2e/fixtures/browser.ts`、`e2e/stories/project/p1-create-project.spec.ts`、`e2e/stories/smoke/s2-web-app.spec.ts`、`pnpm-lock.yaml`、`pnpm-workspace.yaml`、`web/apps/web/core/components/project/create/header.tsx`、`web/apps/web/package.json`、`web/apps/web/vite.config.ts`、`web/packages/i18n/src/locales/en/accessibility.json`、`web/packages/i18n/src/locales/zh-CN/accessibility.json`、`web/packages/propel/package.json`、`web/packages/propel/src/emoji-icon-picker/emoji-picker.tsx`、`web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx`、`web/packages/propel/src/emoji-icon-picker/helper.tsx`、`web/packages/propel/src/emoji-icon-picker/index.ts`

**Interfaces:**
- Produces（spec 2.1；M3 设计 7.7；M2 收尾交接第 10 节）：
  - 依赖：web 应用声明 `emojibase-data`（catalog 的 `15.3.2`，`devDependencies`）。它已在锁文件中（`@tiptap/extension-emoji` 依赖它），锁文件只多 web 应用的两处 importer 条目；写入块之后执行 `pnpm install --frozen-lockfile`。
  - `web/apps/web/emojibase.ts`：`emojibaseUrl`（`/assets/emojibase/<安装的版本>`，版本读 `emojibase-data/package.json`）、`emojibaseFiles()`（要写出的两个文件：`en/data.json`、`en/messages.json`，frimousse 只读这两个）和 Vite 插件 `emojibase()`：客户端的构建把它们写进资源（`generateBundle` 的 `emitFile`），Nerve 嵌入、提供它们，同其余的资源（webui：`/assets/` 长期缓存，新版本是新路径）；开发服务器答同样的路径。`vite.config.ts` 用它。
  - `@nerve/propel`：`EMOJIBASE_URL = "/assets/emojibase/15.3.2"`（`emoji/emoji.tsx`，由 `emoji-icon-picker` 导出），`EmojiPicker.Root` 传 `emojibaseUrl`：表情选择器不再请求 frimousse 默认的 CDN（页面的 CSP `connect-src 'self'` 挡住它）。选择器的按钮收 `ariaLabel`（`TCustomEmojiPicker`），创建项目的图标按钮名为"Project icon"（`aria_labels.project_icon`，两种语言）。
  - oxlint（7.9）：本 Task 改到的两个 propel 文件清零：`emoji.tsx` 三个插槽把传下去的属性叫 `rest`（不再遮住组件的 `props`），`helper.tsx` 的 `toHex` 移到模块（`padStart`，结果相同）；propel 的上限 16 → 11。
  - 静态检查：`.oxlintrc.json` 的非空断言的范围加上 `emojibase.ts` 和它的测试。
- e2e 的共用部分：`e2e/fixtures/browser.ts` 的 `watchPage` 记下每个请求的地址（`PageWatch.requestUrls`），`requestsElsewhere(page, watch)` 给出发往页面所在的源之外的；S2 打开页面时原来自己记的一份删除，经它（spec 第 3 节第 16 条）。

**Tests:**
- vitest：`emojibase.test.ts`（9.5 中 P10 的表情数据）：`the emoji picker's data > is read from nerve where the build writes it, at the version pinned`（`EMOJIBASE_URL` 与插件的地址相同，版本是锁定的）；`… > is emojibase-data's en emoji and their groups' names, which frimousse reads`（两个文件的名字和内容）。
- 端到端：`P1 (page): a new project's icon picker shows the emoji nerve serves itself, nothing asked of another address, nothing blocked`（成员打开创建项目的弹窗、经图标按钮打开表情一页：选择器显示表情，没有发往别处的请求（`requestsElsewhere`）、没有 CSP 违规；表情的数据由 Nerve 答 `GET`，也答 frimousse 的 `HEAD`）。

- [ ] **Step 1: 依赖和构建**

`pnpm-lock.yaml`（修改，2 处）：

````old pnpm-lock.yaml
      specifier: ^10.3.0
      version: 10.5.0
````
````new pnpm-lock.yaml
      specifier: ^10.3.0
      version: 10.5.0
    emojibase-data:
      specifier: 15.3.2
      version: 15.3.2
````
````old pnpm-lock.yaml
        specifier: 19.2.3
        version: 19.2.3(@types/react@19.2.17)
      typescript:
````
````new pnpm-lock.yaml
        specifier: 19.2.3
        version: 19.2.3(@types/react@19.2.17)
      emojibase-data:
        specifier: 'catalog:'
        version: 15.3.2(emojibase@16.0.0)
      typescript:
````

`pnpm-workspace.yaml`（修改，1 处）：

````old pnpm-workspace.yaml
  "emoji-regex": "^10.3.0"
````
````new pnpm-workspace.yaml
  "emoji-regex": "^10.3.0"
  "emojibase-data": "15.3.2"
````

`web/apps/web/emojibase.test.ts`（新文件，28 行）：

````file web/apps/web/emojibase.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { EMOJIBASE_URL } from "@nerve/propel/emoji-icon-picker";
import { emojibaseFiles, emojibaseUrl } from "./emojibase";

// Where the emoji picker reads its data, and what the build writes there (M3 design 7.7, 9.5): nerve serves
// emojibase-data's files itself, at the version the web app pins, and the picker reads them there, not from a CDN.

describe("the emoji picker's data", () => {
  it("is read from nerve where the build writes it, at the version pinned", () => {
    expect([EMOJIBASE_URL, emojibaseUrl]).toEqual(["/assets/emojibase/15.3.2", "/assets/emojibase/15.3.2"]);
  });

  it("is emojibase-data's en emoji and their groups' names, which frimousse reads", () => {
    const files = emojibaseFiles().map(({ fileName, source }) => ({ fileName, read: JSON.parse(source) }));
    expect(files.map(({ fileName }) => fileName)).toEqual([
      "assets/emojibase/15.3.2/en/data.json",
      "assets/emojibase/15.3.2/en/messages.json",
    ]);
    const [data, messages] = files.map(({ read }) => read);
    expect(data).toEqual(expect.arrayContaining([expect.objectContaining({ emoji: "🚀", label: "rocket" })]));
    expect(messages).toEqual(expect.objectContaining({ groups: expect.any(Array), skinTones: expect.any(Array) }));
  });
});
````

`web/apps/web/emojibase.ts`（新文件，71 行）：

````file web/apps/web/emojibase.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import type { Plugin } from "vite";

const require = createRequire(import.meta.url);

/** The emoji picker's data: its emoji, and the names of their groups, in the one locale it reads (frimousse's en). */
const FILES = ["en/data.json", "en/messages.json"];

/** The version of emojibase-data the web app has: its package.json pins it (M3 design 7.7). */
function installedVersion(): string {
  const manifest: unknown = JSON.parse(readFileSync(require.resolve("emojibase-data/package.json"), "utf8"));
  if (
    typeof manifest !== "object" ||
    manifest === null ||
    !("version" in manifest) ||
    typeof manifest.version !== "string"
  ) {
    throw new Error("emojibase-data's package.json names no version");
  }
  return manifest.version;
}

/**
 * Where nerve serves the emoji picker's data: among the build's assets, at a path that names the version, which the
 * picker reads (propel's EMOJIBASE_URL, frimousse's emojibaseUrl) instead of the CDN of frimousse's default, which the
 * page's Content-Security-Policy blocks (connect-src 'self'; M2-closeout §10).
 */
export const emojibaseUrl = `/assets/emojibase/${installedVersion()}`;

/** The picker's data files, each as the build writes it: its name under the client's build, and what it holds. */
export function emojibaseFiles(): { fileName: string; source: string }[] {
  return FILES.map((file) => ({
    fileName: `${emojibaseUrl.slice(1)}/${file}`,
    source: readFileSync(require.resolve(`emojibase-data/${file}`), "utf8"),
  }));
}

/**
 * Serves the emoji picker's data from nerve: the client's build writes emojibaseFiles among its assets, which nerve
 * embeds and serves as it does every asset (webui: cached for good, a new version having a new path); the development
 * server answers the same paths.
 */
export function emojibase(): Plugin {
  return {
    name: "nerve:emojibase",
    applyToEnvironment: (environment) => environment.name === "client",
    generateBundle() {
      for (const { fileName, source } of emojibaseFiles()) {
        this.emitFile({ type: "asset", fileName, source });
      }
    },
    configureServer(server) {
      const files = new Map(emojibaseFiles().map(({ fileName, source }) => [`/${fileName}`, source]));
      server.middlewares.use((request, response, next) => {
        const source = request.url === undefined ? undefined : files.get(request.url);
        if (source === undefined) {
          next();
          return;
        }
        response.setHeader("Content-Type", "application/json");
        response.end(source);
      });
    },
  };
}
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "@types/react-dom": "catalog:",
````
````new web/apps/web/package.json
    "@types/react-dom": "catalog:",
    "emojibase-data": "catalog:",
````

`web/apps/web/vite.config.ts`（修改，2 处）：

````old web/apps/web/vite.config.ts
import { defineConfig } from "vite";
````
````new web/apps/web/vite.config.ts
import { defineConfig } from "vite";
import { emojibase } from "./emojibase";
````
````old web/apps/web/vite.config.ts
  plugins: [reactRouter()],
````
````new web/apps/web/vite.config.ts
  plugins: [reactRouter(), emojibase()],
````

Run: `pnpm install --frozen-lockfile`
Expected: 通过，没有下载新的包（`emojibase-data` 15.3.2 已在 store 中），`pnpm-lock.yaml` 不变。

- [ ] **Step 2: 表情选择器读 Nerve 的数据，按钮有名字**

`web/apps/web/core/components/project/create/header.tsx`（修改，1 处）：

````old web/apps/web/core/components/project/create/header.tsx
            <EmojiPicker
````
````new web/apps/web/core/components/project/create/header.tsx
            <EmojiPicker
              ariaLabel={t("aria_labels.project_icon")}
````

`web/packages/i18n/src/locales/en/accessibility.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/accessibility.json
  "aria_labels": {
````
````new web/packages/i18n/src/locales/en/accessibility.json
  "aria_labels": {
    "project_icon": "Project icon",
````

`web/packages/i18n/src/locales/zh-CN/accessibility.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/accessibility.json
  "aria_labels": {
````
````new web/packages/i18n/src/locales/zh-CN/accessibility.json
  "aria_labels": {
    "project_icon": "项目图标",
````

`web/packages/propel/package.json`（修改，1 处）：

````old web/packages/propel/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 16",
````
````new web/packages/propel/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 11",
````

`web/packages/propel/src/emoji-icon-picker/emoji-picker.tsx`（修改，2 处）：

````old web/packages/propel/src/emoji-icon-picker/emoji-picker.tsx
export function EmojiPicker(props: TCustomEmojiPicker) {
  const {
````
````new web/packages/propel/src/emoji-icon-picker/emoji-picker.tsx
export function EmojiPicker(props: TCustomEmojiPicker) {
  const {
    ariaLabel,
````
````old web/packages/propel/src/emoji-icon-picker/emoji-picker.tsx
      <Popover.Button className={cn("outline-none", buttonClassName)} disabled={disabled}>
````
````new web/packages/propel/src/emoji-icon-picker/emoji-picker.tsx
      <Popover.Button className={cn("outline-none", buttonClassName)} disabled={disabled} aria-label={ariaLabel}>
````

`web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx`（修改，7 处）：

````old web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx
import { cn } from "../../utils";
````
````new web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx
import { cn } from "../../utils";

/**
 * Where the picker reads its data (frimousse's emojibaseUrl): nerve serves emojibase-data's en emoji and messages
 * itself, which the web app's build writes at this path (web/apps/web/emojibase.ts), the version pinned (M3 design
 * 7.7). frimousse's default is a CDN, which the page's Content-Security-Policy blocks and which would learn who uses it.
 */
export const EMOJIBASE_URL = "/assets/emojibase/15.3.2";
````
````old web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx
      className="isolate flex h-full w-full flex-col rounded-md border-none p-2"
````
````new web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx
      className="isolate flex h-full w-full flex-col rounded-md border-none p-2"
      emojibaseUrl={EMOJIBASE_URL}
````
````old web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx
            CategoryHeader: ({ category, ...props }) => (
````
````new web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx
            CategoryHeader: ({ category, ...rest }) => (
````
````old web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx
                className="bg-surface-1 px-3 pb-1.5 text-11 font-medium text-tertiary"
                {...props}
````
````new web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx
                className="bg-surface-1 px-3 pb-1.5 text-11 font-medium text-tertiary"
                {...rest}
````
````old web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx
            Row: ({ children, ...props }) => (
              <div data-slot="emoji-picker-list-row" className="scroll-my-1.5 px-1.5" {...props}>
````
````new web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx
            Row: ({ children, ...rest }) => (
              <div data-slot="emoji-picker-list-row" className="scroll-my-1.5 px-1.5" {...rest}>
````
````old web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx
            Emoji: ({ emoji, ...props }) => (
````
````new web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx
            Emoji: ({ emoji, ...rest }) => (
````
````old web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx
                className="data-active:bg-accent flex size-8 items-center justify-center rounded-md text-16"
                {...props}
````
````new web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx
                className="data-active:bg-accent flex size-8 items-center justify-center rounded-md text-16"
                {...rest}
````

`web/packages/propel/src/emoji-icon-picker/helper.tsx`（修改，3 处）：

````old web/packages/propel/src/emoji-icon-picker/helper.tsx
export type TCustomEmojiPicker = {
````
````new web/packages/propel/src/emoji-icon-picker/helper.tsx
export type TCustomEmojiPicker = {
  /** The name of the picker's button, which shows the icon picked: what the icon is of. */
  ariaLabel?: string;
````
````old web/packages/propel/src/emoji-icon-picker/helper.tsx
  align?: TAlign;
};
````
````new web/packages/propel/src/emoji-icon-picker/helper.tsx
  align?: TAlign;
};

/** A colour channel's value, 0 to 255, as two hex digits. */
const toHex = (value: number): string => value.toString(16).padStart(2, "0");
````
````old web/packages/propel/src/emoji-icon-picker/helper.tsx
  // Convert RGB back to hex
  const toHex = (value: number): string => {
    const hex = value.toString(16);
    return hex.length === 1 ? "0" + hex : hex;
  };

````
````new web/packages/propel/src/emoji-icon-picker/helper.tsx
  // Convert RGB back to hex
````

`web/packages/propel/src/emoji-icon-picker/index.ts`（修改，1 处）：

````old web/packages/propel/src/emoji-icon-picker/index.ts
 */

````
````new web/packages/propel/src/emoji-icon-picker/index.ts
 */

export { EMOJIBASE_URL } from "./emoji/emoji";
````

- [ ] **Step 3: 静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/hooks/use-sign-out.test.ts"
````
````new .oxlintrc.json
        "web/apps/web/core/hooks/use-sign-out.test.ts",
        "web/apps/web/emojibase.ts",
        "web/apps/web/emojibase.test.ts"
````

- [ ] **Step 4: 端到端**

`e2e/fixtures/browser.ts`（修改，4 处）：

````old e2e/fixtures/browser.ts
  readonly apiRequests: string[];
````
````new e2e/fixtures/browser.ts
  readonly apiRequests: string[];
  /** Every request, by its URL, of any origin: nerve's, or another (requestsElsewhere). */
  readonly requestUrls: string[];
````
````old e2e/fixtures/browser.ts
    apiRequests: [],
````
````new e2e/fixtures/browser.ts
    apiRequests: [],
    requestUrls: [],
````
````old e2e/fixtures/browser.ts
  };
  page.on("request", (request) => {
````
````new e2e/fixtures/browser.ts
  };
  page.on("request", (request) => {
    watch.requestUrls.push(request.url());
````
````old e2e/fixtures/browser.ts
  return watch;
````
````new e2e/fixtures/browser.ts
  return watch;
}

/**
 * The requests watch saw page send to an origin other than the one it shows now: nerve serves the app and all it
 * loads, the emoji picker's data among them (M3 design 7.7), and the page's CSP allows no other (M2 design 8.3).
 */
export function requestsElsewhere(page: Page, watch: PageWatch): string[] {
  const { origin } = new URL(page.url());
  return watch.requestUrls.filter((url) => new URL(url).origin !== origin);
````

`e2e/stories/project/p1-create-project.spec.ts`（修改，3 处）：

````old e2e/stories/project/p1-create-project.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/project/p1-create-project.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, requestsElsewhere, watchPage } from "../../fixtures/browser";
import { registerOnboarded } from "../../fixtures/settings-pages";
````
````old e2e/stories/project/p1-create-project.spec.ts
// P1, create a project (M3 design 2, 3.17, 3.18). The page version comes
// with the projects' pages (P10).
````
````new e2e/stories/project/p1-create-project.spec.ts
// P1, create a project (M3 design 2, 3.17, 3.18), and its page (P10).

/** Where nerve serves the emoji picker's data (M3 design 7.7). */
const EMOJIBASE = "/assets/emojibase/15.3.2/en";
````
````old e2e/stories/project/p1-create-project.spec.ts
  expect(await read(api, member, ops.id)).toEqual({ status: 404, code: "project.not_found" });
});

````
````new e2e/stories/project/p1-create-project.spec.ts
  expect(await read(api, member, ops.id)).toEqual({ status: 404, code: "project.not_found" });
});

test("P1 (page): a new project's icon picker shows the emoji nerve serves itself, nothing asked of another address, nothing blocked", async ({
  api,
  signedInPage,
}, testInfo) => {
  const member = await registerOnboarded(api, emailFor(testInfo, "member"));
  const slug = slugFor(testInfo);
  await createWorkspace(api, member.access_token, { name: "Acme", slug });
  const page = await signedInPage(member);
  const watch = await watchPage(page);

  await page.goto(`/${slug}/projects`);
  await page.getByRole("button", { name: "Add Project", exact: true }).click();
  const data = page.waitForResponse((answer) => new URL(answer.url()).pathname === `${EMOJIBASE}/data.json`);
  const messages = page.waitForResponse((answer) => new URL(answer.url()).pathname === `${EMOJIBASE}/messages.json`);
  await page.getByRole("button", { name: "Project icon" }).click();
  expect([(await data).status(), (await messages).status()]).toEqual([200, 200]);
  await page.getByRole("searchbox").fill("rocket");
  await page.getByRole("gridcell", { name: "Rocket" }).click();
  await expect(page.getByRole("button", { name: "Project icon" })).toContainText("🚀");

  // The picker asks nerve, which answers its HEAD too (frimousse compares the files' ETags before it reads its cache).
  expect((await page.request.head(`${EMOJIBASE}/data.json`)).status()).toBe(200);
  expect(requestsElsewhere(page, watch)).toEqual([]);
  expect([watch.cspViolations, watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});

````

`e2e/stories/smoke/s2-web-app.spec.ts`（修改，4 处）：

````old e2e/stories/smoke/s2-web-app.spec.ts
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage, type PageWatch } from "../../fixtures/browser";
````
````new e2e/stories/smoke/s2-web-app.spec.ts
import {
  EMOJI_CHECK_WARNING,
  expectQuietConsole,
  requestsElsewhere,
  watchPage,
  type PageWatch,
} from "../../fixtures/browser";
````
````old e2e/stories/smoke/s2-web-app.spec.ts
  const watch = await watchPage(page);
  const requested: string[] = [];
````
````new e2e/stories/smoke/s2-web-app.spec.ts
  const watch = await watchPage(page);
````
````old e2e/stories/smoke/s2-web-app.spec.ts
  const failed: string[] = [];
  page.on("request", (req) => {
    requested.push(req.url());
  });
````
````new e2e/stories/smoke/s2-web-app.spec.ts
  const failed: string[] = [];
````
````old e2e/stories/smoke/s2-web-app.spec.ts
  const origin = new URL(document.url()).origin;
  const elsewhere = requested.filter((url) => new URL(url).origin !== origin);
  return { document, watch, loaded, failed, elsewhere };
````
````new e2e/stories/smoke/s2-web-app.spec.ts
  return { document, watch, loaded, failed, elsewhere: requestsElsewhere(page, watch) };
````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 67 条规则、3 个例外，没有命中；web 的 oxlint 356 条，`propel` 11 条，各等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 94 个全部通过。

- [ ] **Step 6: 提交**

```bash
git add .oxlintrc.json e2e/fixtures/browser.ts e2e/stories/project/p1-create-project.spec.ts e2e/stories/smoke/s2-web-app.spec.ts pnpm-lock.yaml pnpm-workspace.yaml web/apps/web/core/components/project/create/header.tsx web/apps/web/emojibase.test.ts web/apps/web/emojibase.ts web/apps/web/package.json web/apps/web/vite.config.ts web/packages/i18n/src/locales/en/accessibility.json web/packages/i18n/src/locales/zh-CN/accessibility.json web/packages/propel/package.json web/packages/propel/src/emoji-icon-picker/emoji-picker.tsx web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx web/packages/propel/src/emoji-icon-picker/helper.tsx web/packages/propel/src/emoji-icon-picker/index.ts
```
```bash
git commit -m "feat(M3/P10): the emoji picker reads its data from nerve, which the build writes among its assets

The web app declares emojibase-data, which the lockfile has already, and a
Vite plugin writes its en data and messages under /assets/emojibase/<version>,
which nerve embeds and serves as every asset; the picker's emojibaseUrl
points there, so it asks no CDN, which the page's CSP blocks. The project
icon's button has a name. The two propel files the task edits end with no
oxlint warning, the cap lowered. P1's page version opens the picker;
watchPage keeps every request's URL, and requestsElsewhere gives those of
another origin, which S2's visit reads too.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p10.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T1.1` | 表情选择器从 frimousse 默认的 CDN 读数据 | 故事 P1 | 端到端 |
| `T1.2` | 构建不写出表情选择器的数据 | 故事 P1 | 端到端 |
| `T1.3` | 表情选择器读的版本不是构建写出的版本 | `emojibase.test.ts`、故事 P1 | vitest；端到端 |
| `T1.4` | 表情选择器的按钮没有名字 | 故事 P1 | 端到端 |
| `T1.5` | 在表情数据的插件里写一个非空断言 | oxlint（`check:lint`） | 静态 |

---

### Task 2: 下拉框：带搜索的选择 Tab 能到、在按钮旁展开；成员下拉框以它为底，照旧对调用方；面包屑一个 Tab 停留点

**Files:**
- Create: `web/apps/web/core/components/dropdowns/member/base.test.tsx`
- Modify: `.oxlintrc.json`、`e2e/fixtures/settings-pages.ts`、`e2e/fixtures/workspace-pages.ts`、`e2e/stories/project/p1-create-project.spec.ts`、`e2e/stories/workspace/w3-workspace-settings.spec.ts`、`web/apps/web/core/components/dropdowns/buttons.tsx`、`web/apps/web/core/components/dropdowns/member/base.tsx`、`web/apps/web/core/components/dropdowns/member/types.d.ts`、`web/apps/web/core/lib/fake-controls.ts`、`web/apps/web/package.json`、`web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx`、`web/packages/ui/src/dropdowns/custom-search-select.tsx`、`web/packages/ui/src/dropdowns/helper.tsx`
- Delete: `web/apps/web/core/components/dropdowns/member/member-options.tsx`

**Interfaces:**
- Produces（spec 2.2；M3 设计 7.7；M2 收尾交接第 14 节，根因和修法见 M2/P5 spec 2.3）：
  - `@nerve/ui` 的 `CustomSearchSelect`（整个文件）：按钮是 Headless UI 的 `Popover.Button`，Tab 能到，点击、Enter、空格打开（原来是 `Combobox.Button`，Headless UI 2.2 固定给它 `tabIndex: -1`）；搜索框和选项是面板里的 `Combobox`（`static`），打开状态只有 Popover 的一份。列表打开时搜索框取得焦点（不滚动：popper 那时还没定位）；Escape 从按钮或搜索框都关上列表，焦点回到按钮，不传给外面的弹窗（输入打开 Combobox 之后它的输入框先处理 Escape 并 `preventDefault`，Headless UI 随之跳过面板自己的处理，所以面板在捕获阶段处理）；输入法组字时的 Escape 是输入法的，不关列表（`event.nativeEvent.isComposing`）；单选选了一项就关；清空搜索不改选中的值；列表关上时清空搜索（`WhileOpen` 在列表挂载、卸载时调 `onOpen`、`onClose`，并清空搜索；它用的是列表打开那一刻的两个回调）；`defaultOpen` 开一次，像点了按钮；popper 的 ref 在面板（门户里）本身，列表在按钮旁展开。`helper.tsx`：`placement` 用 popper 的 `Placement` 类型（原来手写的联合删除）；两个可选的 props：`focusSearchOnOpen`（列表打开时搜索框是否取得焦点，默认是）、`popperModifiers`（popper 的修饰，与 `placement` 一起交给 `usePopper`）。其余 props 不变。
  - 成员下拉框 `dropdowns/member/base.tsx`（整个文件）：以 `CustomSearchSelect` 为底（选项是工作区的成员，按 `memberIds` 筛选，单选、多选），`member-options.tsx`（popper 的 ref 放在列表唯一的子元素上，列表停在页面左上角；两份打开状态）删除；Tab 只停在里面的 `DropdownButton`（每个 `dropdowns/` 下拉框都这样嵌一个按钮，嵌套的按钮是 Plane 的，M4 随其余的下拉框处理），它的 Enter、空格交给选择的按钮打开列表；调用方给的 `tabIndex` 给这个按钮（`dropdowns/buttons.tsx` 的 `DropdownButton` 收 `tabIndex`，交给三种样式的按钮；原来在选择自己的按钮上，那个按钮现在不在 Tab 顺序里）。M4–M6 的调用方靠的照旧（spec 第 3 节"预检之后"的 A-M5）：调用方推迟的（`renderByDefault` 为假：桌面上列表的每一行）在指针第一次经过之前只是它的按钮，没有选择、没有 popper（与 `ComboDropDown` 相同）；手机上打开时搜索框不取得焦点（`focusSearchOnOpen={!isMobile}`）；列表离窗口边缘 12 像素（popper 的 `preventOverflow`）；按钮的点击不冒泡到行、不做默认动作：Headless UI 的 Popover 按钮自己在打开之前 `preventDefault`、`stopPropagation`（点击和按下都是），所以不另写。`types.d.ts` 删去 `button`（只有旧的写法读它）。
  - 面包屑的 `BreadcrumbNavigationSearchDropdown`（`@nerve/ui`）：一节只有一个 Tab 停留点，选择的按钮，它打开列表。最后一节的标题是 `<span>`（点它也打开列表）；之前一节的标题点击时去那一节的页面，是选择的按钮里的按钮，不在 Tab 顺序里（`tabIndex={-1}`；键盘从侧边栏到那个页面）。
  - `web/apps/web/package.json`：web 的 oxlint 上限 356 → 350（`dropdowns/member/` 的六条随旧的写法删除）。
  - 测试的替身：`core/lib/fake-controls.ts` 的搜索选择留下它收到的 `focusSearchOnOpen`、`popperModifiers`（`SearchSelect`）。根目录 `.oxlintrc.json` 的非空断言范围加 `dropdowns/member/base.test.tsx`、`core/lib/fake-controls.ts`（spec 2.14）。
- e2e 的共用部分：`e2e/fixtures/settings-pages.ts` 的 `transitionsEnded(locator)`（等定位到的元素和它里面的过渡结束：弹窗的进场过渡移动按钮，popper 在列表打开时按按钮此刻的位置定位）；`e2e/fixtures/workspace-pages.ts` 的 `pickTimeZone(page, current, typed)`（只用键盘在工作区的 general 页选时区：Tab 到选择的按钮、Enter 打开、列表在按钮旁、搜索框有焦点、Escape 关上、再打开、输入、方向键和 Enter 选第一项）。

**Tests:**
- vitest：`core/components/dropdowns/member/base.test.tsx`（新文件，4 个；`@nerve/ui` 换成 `fake-controls.ts`，服务端渲染）：`is its button alone, no select, while its caller defers it`；`is a select, its button in the select's, unless its caller defers it`；`focuses the search as the list opens on a desktop, not on a phone`；`keeps its list 12 pixels inside the window`。
- 端到端：`P1 (page): the lead's list opens beside its button with its search focused; Escape closes it`（负责人的按钮外面那个选择自己的按钮不在 Tab 顺序中（`tabindex="-1"`），Tab 只停一次；创建弹窗的进场过渡结束之后点负责人：列表在按钮旁展开（`expectListBesideButton`），搜索框有焦点；Escape 关上列表、弹窗还开着；在按钮上按 Enter 再打开，输入之后，组字中的 Escape（`isComposing`）不关，再输入仍在搜索框里；输入打开 Combobox 之后的 Escape 关上；再打开时没有搜索）。W3 的页面版本中时区改用 `pickTimeZone`，选中的"Beijing"按钮保有焦点。个人设置的时区（A8、A9）也是 `CustomSearchSelect`，照旧通过。面包屑的一个停留点由 Task 3 的 P2 页面版本核对（项目的工作项页的页头），添加成员的选择用 Tab 到达由 Task 8 的 P5 核对。

- [ ] **Step 1: 带搜索的选择和面包屑**

`web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx`（修改，4 处）：

````old web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
````
````new web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const titleClassName = cn(
    "group flex h-full cursor-pointer items-center gap-2 rounded-sm rounded-r-none px-1.5 py-1 text-13 font-medium text-tertiary",
    {
      "hover:bg-layer-1 hover:text-primary": !isLast,
    }
  );
  const label = (
    <>
      {shouldTruncate && <div className="flex text-tertiary @4xl:hidden">...</div>}
      <div
        className={cn("flex gap-2", {
          "hidden items-center gap-2 @4xl:flex": shouldTruncate,
        })}
      >
        {icon && <Breadcrumbs.Icon>{icon}</Breadcrumbs.Icon>}
        <Breadcrumbs.Label>{title}</Breadcrumbs.Label>
      </div>
    </>
  );
````
````old web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx
            <button
              onClick={(e) => {
                if (!isLast) {
````
````new web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx
            {/* The crumb is one Tab stop, the select's button, which opens the list; the last crumb's title opens it
                too. An earlier crumb's title goes to its page by a click: a button in the select's button, out of the
                Tab order (the keyboard reaches that page from the sidebar). */}
            {isLast ? (
              <span className={titleClassName}>{label}</span>
            ) : (
              <button
                type="button"
                tabIndex={-1}
                onClick={(e) => {
````
````old web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx
                }
              }}
              className={cn(
                "group flex h-full cursor-pointer items-center gap-2 rounded-sm rounded-r-none px-1.5 py-1 text-13 font-medium text-tertiary",
                {
                  "hover:bg-layer-1 hover:text-primary": !isLast,
                }
              )}
            >
              {shouldTruncate && <div className="flex text-tertiary @4xl:hidden">...</div>}
              <div
                className={cn("flex gap-2", {
                  "hidden items-center gap-2 @4xl:flex": shouldTruncate,
                })}
````
````new web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx
                }}
                className={titleClassName}
````
````old web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx
                {icon && <Breadcrumbs.Icon>{icon}</Breadcrumbs.Icon>}
                <Breadcrumbs.Label>{title}</Breadcrumbs.Label>
              </div>
            </button>
````
````new web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx
                {label}
              </button>
            )}
````

`web/packages/ui/src/dropdowns/custom-search-select.tsx`（整个文件，272 行）：

````whole web/packages/ui/src/dropdowns/custom-search-select.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Combobox, Popover } from "@headlessui/react";
import { ChevronDownOutline, InfoOutline, SearchOutline, TickOutline } from "@makeplane/propel/icons";
import React, { useCallback, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { usePopper } from "react-popper";
// local imports
import { Tooltip } from "@nerve/propel/tooltip";
import { cn } from "../utils";
import type { ICustomSearchSelectProps } from "./helper";

/**
 * Calls onOpen as it mounts, with the list, and onClose as it unmounts, when the list closes: Headless UI's Popover
 * has no callback for either.
 */
function WhileOpen(props: { onOpen?: () => void; onClose?: () => void }) {
  // the callbacks of the moment the list opened: a caller's new functions on each render do not call them again
  const [{ onOpen, onClose }] = useState(() => props);
  useEffect(() => {
    onOpen?.();
    return () => onClose?.();
  }, [onOpen, onClose]);
  return null;
}

export function CustomSearchSelect(props: ICustomSearchSelectProps) {
  const {
    customButtonClassName = "",
    buttonClassName = "",
    className = "",
    chevronClassName = "",
    customButton,
    placement,
    disabled = false,
    footerOption,
    input = false,
    label,
    maxHeight = "md",
    multiple = false,
    noChevron = false,
    onChange,
    options,
    onOpen,
    onClose,
    optionsClassName = "",
    value,
    tabIndex,
    searchPlaceholder = "Search",
    noResultsMessage = "No matches found",
    loadingMessage = "Loading...",
    defaultOpen = false,
    focusSearchOnOpen = true,
    popperModifiers,
  } = props;
  const [query, setQuery] = useState("");

  const [referenceElement, setReferenceElement] = useState<HTMLButtonElement | null>(null);
  const [popperElement, setPopperElement] = useState<HTMLElement | null>(null);
  const openedByDefault = useRef(false);
  // The search input is in the list: it takes the focus as the list opens (unless the caller says not), so that
  // typing, the arrows, Enter and Escape reach Headless UI's input (a click on the button leaves the focus on the
  // button). Not scrolled to: popper has not placed the list yet.
  const focusSearch = useCallback((search: HTMLInputElement | null) => {
    search?.focus({ preventScroll: true });
  }, []);

  const { styles, attributes } = usePopper(referenceElement, popperElement, {
    placement: placement ?? "bottom-start",
    modifiers: popperModifiers,
  });

  // Headless UI's Popover has no defaultOpen: a list asked to start open opens once, as a click on its button does.
  useEffect(() => {
    if (!defaultOpen || openedByDefault.current || !referenceElement) return;
    openedByDefault.current = true;
    referenceElement.click();
  }, [defaultOpen, referenceElement]);

  const filteredOptions =
    query === "" ? options : options?.filter((option) => option.query.toLowerCase().includes(query.toLowerCase()));

  const comboboxProps: any = {
    value,
    disabled,
  };

  if (multiple) comboboxProps.multiple = true;

  return (
    // The button is Headless UI's Popover's, which Tab reaches (a Combobox's button is out of the Tab order: Headless
    // UI 2.2 gives it tabIndex -1 whatever its caller says); the search and the options are a Combobox in its panel.
    // The Popover's open state is the only one: its button opens the list (a click, Enter, Space); it closes it
    // (Escape, a pick of a single value, a click outside, the focus leaving it), and the focus goes back to the
    // button.
    <Popover className={cn("relative flex-shrink-0 text-left", className)}>
      {({ open, close }) => (
        <>
          {customButton ? (
            <Popover.Button
              ref={setReferenceElement}
              type="button"
              disabled={disabled}
              tabIndex={tabIndex}
              className={cn(
                "flex w-full items-center justify-between gap-1 text-11",
                {
                  "cursor-not-allowed text-secondary": disabled,
                  "cursor-pointer hover:bg-layer-transparent-hover": !disabled,
                },
                customButtonClassName
              )}
            >
              {customButton}
            </Popover.Button>
          ) : (
            <Popover.Button
              ref={setReferenceElement}
              type="button"
              disabled={disabled}
              tabIndex={tabIndex}
              className={cn(
                "flex w-full items-center justify-between gap-1 rounded-sm border-[0.5px] border-strong",
                {
                  "px-3 py-2 text-13": input,
                  "px-2 py-1 text-11": !input,
                  "cursor-not-allowed text-secondary": disabled,
                  "cursor-pointer hover:bg-layer-transparent-hover": !disabled,
                },
                buttonClassName
              )}
            >
              {label}
              {!noChevron && !disabled && (
                <ChevronDownOutline className={cn("h-3 w-3 flex-shrink-0", chevronClassName)} aria-hidden="true" />
              )}
            </Popover.Button>
          )}
          {open && (
            <WhileOpen
              onOpen={onOpen}
              onClose={() => {
                // the list opens again with no search
                setQuery("");
                onClose?.();
              }}
            />
          )}
          {createPortal(
            // Popper places the panel, which holds the list and nothing else.
            <Popover.Panel
              data-prevent-outside-click
              ref={setPopperElement}
              className="z-30"
              style={styles.popper}
              {...attributes.popper}
              // Escape closes the list, from the search too: once typing has opened the Combobox, its input takes
              // Escape first and prevents its default, and Headless UI then skips the panel's own handler. Not
              // passed on: a modal around the select stays open. An Escape that ends an input method's composition
              // (Chinese, say) is the input method's.
              onKeyDownCapture={(event: React.KeyboardEvent) => {
                if (event.key !== "Escape" || event.nativeEvent.isComposing) return;
                event.preventDefault();
                event.stopPropagation();
                close();
              }}
            >
              <Combobox
                {...comboboxProps}
                // A pick of a single value closes the list. Headless UI's Combobox picks null when its input is
                // emptied (single mode): the input here only filters the options, so emptying the search leaves the
                // value as it is.
                onChange={(picked: unknown) => {
                  if (multiple) {
                    onChange(picked);
                  } else if (picked !== null) {
                    onChange(picked);
                    close();
                  }
                }}
              >
                {/* Static: the list shows while the panel does, whatever the Combobox's own state. Not modal: the
                    panel is the list's, and a modal list makes the rest of the page inert (Headless UI 2.2). */}
                <Combobox.Options
                  as="ul"
                  static
                  modal={false}
                  className={cn(
                    "my-1 min-w-48 overflow-y-scroll rounded-md border-[0.5px] border-subtle-1 bg-surface-1 py-2.5 text-11 whitespace-nowrap focus:outline-none",
                    optionsClassName
                  )}
                >
                  <div className="mx-2 flex items-center gap-1.5 rounded-sm border border-subtle px-2">
                    <SearchOutline className="h-3.5 w-3.5 text-placeholder" />
                    <Combobox.Input
                      ref={focusSearchOnOpen ? focusSearch : undefined}
                      className="w-full bg-transparent py-1 text-11 text-secondary placeholder:text-placeholder focus:outline-none"
                      value={query}
                      onChange={(e) => setQuery(e.target.value)}
                      placeholder={searchPlaceholder}
                      displayValue={(assigned: any) => assigned?.name}
                    />
                  </div>
                  <div
                    className={cn("vertical-scrollbar mt-2 scrollbar-xs space-y-1 overflow-y-scroll px-2", {
                      "max-h-96": maxHeight === "2xl",
                      "max-h-80": maxHeight === "xl",
                      "max-h-60": maxHeight === "lg",
                      "max-h-48": maxHeight === "md",
                      "max-h-36": maxHeight === "rg",
                      "max-h-28": maxHeight === "sm",
                    })}
                  >
                    {filteredOptions ? (
                      filteredOptions.length > 0 ? (
                        filteredOptions.map((option) => (
                          <Combobox.Option
                            as="li"
                            key={option.value}
                            value={option.value}
                            className={({ active }) =>
                              cn(
                                "flex w-full cursor-pointer items-center justify-between gap-2 truncate rounded-sm px-1 py-1.5 select-none",
                                {
                                  "bg-layer-transparent-hover": active,
                                  "cursor-not-allowed text-placeholder opacity-60": option.disabled,
                                }
                              )
                            }
                            disabled={option.disabled}
                          >
                            {({ selected }) => (
                              <>
                                <span className="flex-grow truncate">{option.content}</span>
                                {selected && <TickOutline className="h-3.5 w-3.5 flex-shrink-0" />}
                                {option.tooltip && (
                                  <>
                                    {typeof option.tooltip === "string" ? (
                                      <Tooltip tooltipContent={option.tooltip}>
                                        <InfoOutline className="h-3.5 w-3.5 flex-shrink-0 cursor-pointer text-secondary" />
                                      </Tooltip>
                                    ) : (
                                      option.tooltip
                                    )}
                                  </>
                                )}
                              </>
                            )}
                          </Combobox.Option>
                        ))
                      ) : (
                        <p className="px-1.5 py-1 text-placeholder italic">{noResultsMessage}</p>
                      )
                    ) : (
                      <p className="px-1.5 py-1 text-placeholder italic">{loadingMessage}</p>
                    )}
                  </div>
                  {footerOption}
                </Combobox.Options>
              </Combobox>
            </Popover.Panel>,
            document.body
          )}
        </>
      )}
    </Popover>
  );
}
````

`web/packages/ui/src/dropdowns/helper.tsx`（修改，2 处）：

````old web/packages/ui/src/dropdowns/helper.tsx
// FIXME: fix this!!!
import type { ICustomSearchSelectOption } from "@nerve/types";

type Placement =
  | "top"
  | "top-start"
  | "top-end"
  | "bottom"
  | "bottom-start"
  | "bottom-end"
  | "left"
  | "left-start"
  | "left-end"
  | "right"
  | "right-start"
  | "right-end";
````
````new web/packages/ui/src/dropdowns/helper.tsx
// FIXME: fix this!!!
import type { Modifier, Placement } from "@popperjs/core";
import type { ICustomSearchSelectOption } from "@nerve/types";
````
````old web/packages/ui/src/dropdowns/helper.tsx
  options?: ICustomSearchSelectOption[];
````
````new web/packages/ui/src/dropdowns/helper.tsx
  options?: ICustomSearchSelectOption[];
  /** Whether the search takes the focus as the list opens: it does unless the caller says not. */
  focusSearchOnOpen?: boolean;
  /** Popper's modifiers for the list, beside its placement. */
  popperModifiers?: Partial<Modifier<string, object>>[];
````

- [ ] **Step 2: 成员下拉框、它的测试和上限**

`web/apps/web/core/components/dropdowns/buttons.tsx`（修改，10 处）：

````old web/apps/web/core/components/dropdowns/buttons.tsx
  variant: TButtonVariants;
  renderToolTipByDefault?: boolean;
````
````new web/apps/web/core/components/dropdowns/buttons.tsx
  variant: TButtonVariants;
  renderToolTipByDefault?: boolean;
  /** The button's place in the Tab order, where the dropdown's caller gives one. */
  tabIndex?: number;
````
````old web/apps/web/core/components/dropdowns/buttons.tsx
  showTooltip: boolean;
  renderToolTipByDefault?: boolean;
````
````new web/apps/web/core/components/dropdowns/buttons.tsx
  showTooltip: boolean;
  renderToolTipByDefault?: boolean;
  tabIndex?: number;
````
````old web/apps/web/core/components/dropdowns/buttons.tsx
    variant,
````
````new web/apps/web/core/components/dropdowns/buttons.tsx
    variant,
    tabIndex,
````
````old web/apps/web/core/components/dropdowns/buttons.tsx
      renderToolTipByDefault={renderToolTipByDefault}
````
````new web/apps/web/core/components/dropdowns/buttons.tsx
      renderToolTipByDefault={renderToolTipByDefault}
      tabIndex={tabIndex}
````
````old web/apps/web/core/components/dropdowns/buttons.tsx
function BorderButton(props: ButtonProps) {
  const { children, className, isActive, tooltipContent, tooltipHeading, showTooltip } = props;
````
````new web/apps/web/core/components/dropdowns/buttons.tsx
function BorderButton(props: ButtonProps) {
  const { children, className, isActive, tooltipContent, tooltipHeading, showTooltip, tabIndex } = props;
````
````old web/apps/web/core/components/dropdowns/buttons.tsx
      <Button
        variant="ghost"
        size="sm"
        className={cn(
          "flex h-full w-full items-center justify-start gap-1.5 border-[0.5px] border-strong",
````
````new web/apps/web/core/components/dropdowns/buttons.tsx
      <Button
        variant="ghost"
        size="sm"
        tabIndex={tabIndex}
        className={cn(
          "flex h-full w-full items-center justify-start gap-1.5 border-[0.5px] border-strong",
````
````old web/apps/web/core/components/dropdowns/buttons.tsx
  const { children, className, tooltipContent, tooltipHeading, showTooltip } = props;
````
````new web/apps/web/core/components/dropdowns/buttons.tsx
  const { children, className, tooltipContent, tooltipHeading, showTooltip, tabIndex } = props;
````
````old web/apps/web/core/components/dropdowns/buttons.tsx
      <Button
        variant="ghost"
        size="sm"
        className={cn(
          "flex h-full w-full items-center justify-between gap-1.5 bg-layer-3 hover:bg-layer-1-hover",
````
````new web/apps/web/core/components/dropdowns/buttons.tsx
      <Button
        variant="ghost"
        size="sm"
        tabIndex={tabIndex}
        className={cn(
          "flex h-full w-full items-center justify-between gap-1.5 bg-layer-3 hover:bg-layer-1-hover",
````
````old web/apps/web/core/components/dropdowns/buttons.tsx
function TransparentButton(props: ButtonProps) {
  const { children, className, isActive, tooltipContent, tooltipHeading, showTooltip } = props;
````
````new web/apps/web/core/components/dropdowns/buttons.tsx
function TransparentButton(props: ButtonProps) {
  const { children, className, isActive, tooltipContent, tooltipHeading, showTooltip, tabIndex } = props;
````
````old web/apps/web/core/components/dropdowns/buttons.tsx
      <Button
        variant="ghost"
        size="sm"
        className={cn(
          "flex h-full w-full items-center justify-between gap-1.5",
````
````new web/apps/web/core/components/dropdowns/buttons.tsx
      <Button
        variant="ghost"
        size="sm"
        tabIndex={tabIndex}
        className={cn(
          "flex h-full w-full items-center justify-between gap-1.5",
````

`web/apps/web/core/components/dropdowns/member/base.test.tsx`（新文件，68 行）：

````file web/apps/web/core/components/dropdowns/member/base.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { emptyShown, shown } from "@/lib/fake-controls";
import { MemberDropdownBase } from "./base";

// What the member dropdown keeps for its callers (M4–M6's rows among them) now that a CustomSearchSelect is under it
// (M3 design 7.7, P10 spec 3): it renders on the server with a stand-in for the select (fake-controls.ts), which keeps
// the props it was given, on a desktop unless the test says a phone.

const device = vi.hoisted(() => ({ isMobile: false }));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => device }));
vi.mock("@/hooks/store/user", () => ({ useUser: () => ({ data: { id: "u-me" } }) }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({ getUserDetails: () => undefined, workspace: { isUserSuspended: () => false } }),
}));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "acme" }) }));

/** Renders the lead's dropdown, as a caller that defers it or not; gives its markup. */
function render(renderByDefault?: boolean) {
  emptyShown();
  return renderToStaticMarkup(
    <MemberDropdownBase
      getUserDetails={() => undefined}
      memberIds={["u-me"]}
      multiple={false}
      value={null}
      onChange={() => {}}
      buttonVariant="border-with-text"
      placeholder="Lead"
      renderByDefault={renderByDefault}
    />
  );
}

beforeEach(() => {
  device.isMobile = false;
});

describe("MemberDropdownBase", () => {
  it("is its button alone, no select, while its caller defers it", () => {
    const markup = render(false);
    expect([shown.searchSelects.length, markup.includes(">Lead</span></button>")]).toEqual([0, true]);
  });

  it("is a select, its button in the select's, unless its caller defers it", () => {
    expect([render(), shown.searchSelects.length]).toEqual(["", 1]);
  });

  it("focuses the search as the list opens on a desktop, not on a phone", () => {
    render();
    const desktop = shown.searchSelects[0]?.focusSearchOnOpen;
    device.isMobile = true;
    render();
    expect([desktop, shown.searchSelects[0]?.focusSearchOnOpen]).toEqual([true, false]);
  });

  it("keeps its list 12 pixels inside the window", () => {
    render();
    expect(shown.searchSelects[0]?.popperModifiers).toEqual([{ name: "preventOverflow", options: { padding: 12 } }]);
  });
});
````

`web/apps/web/core/components/dropdowns/member/base.tsx`（整个文件，195 行）：

````whole web/apps/web/core/components/dropdowns/member/base.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import type { ComponentType, SVGProps } from "react";
import { useState } from "react";
import { useParams } from "react-router";
import { useTranslation } from "@nerve/i18n";
import { Avatar } from "@makeplane/propel/components/avatar";
import { ChevronDownOutline, DeactivatedUserOutline } from "@makeplane/propel/icons";
// nerve imports
import { EPillSize, EPillVariant, Pill } from "@nerve/propel/pill";
import type { MemberUser } from "@nerve/api-client";
import type { ICustomSearchSelectOption } from "@nerve/types";
import { CustomSearchSelect } from "@nerve/ui";
// helpers
import { cn, getFileURL, sortByCurrentUserThenSelected } from "@nerve/utils";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useUser } from "@/hooks/store/user";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { DropdownButton } from "../buttons";
import { BUTTON_VARIANTS_WITH_TEXT } from "../constants";
import { ButtonAvatars } from "./avatar";
import type { MemberDropdownProps } from "./types";

type TMemberDropdownBaseProps = {
  getUserDetails: (userId: string) => MemberUser | undefined;
  icon?: ComponentType<SVGProps<SVGSVGElement>>;
  memberIds?: string[];
  onClose?: () => void;
  onDropdownOpen?: () => void;
  optionsClassName?: string;
  renderByDefault?: boolean;
} & MemberDropdownProps;

/**
 * Picks members: a CustomSearchSelect (@nerve/ui), whose list, with its search, opens beside its button, the open
 * state Headless UI's alone, the search taking the focus as the list opens on a desktop (M3 design 7.7). A dropdown
 * its caller defers (renderByDefault false: a row of a list on a desktop) is its button alone until the pointer
 * first comes over it, as ComboDropDown (@nerve/ui) does: no select and no popper for each row.
 */
export const MemberDropdownBase = observer(function MemberDropdownBase(props: TMemberDropdownBaseProps) {
  const { t } = useTranslation();
  const {
    buttonClassName,
    buttonContainerClassName,
    buttonVariant,
    className = "",
    disabled = false,
    dropdownArrow = false,
    dropdownArrowClassName = "",
    getUserDetails,
    hideIcon = false,
    icon,
    memberIds,
    onClose,
    onDropdownOpen,
    optionsClassName = "",
    placeholder = t("members"),
    placement,
    renderByDefault = true,
    showTooltip = false,
    showUserDetails = false,
    tabIndex,
    tooltipContent,
    value,
  } = props;
  // router
  const { workspaceSlug } = useParams();
  // store hooks
  const { data: currentUser } = useUser();
  const {
    workspace: { isUserSuspended },
  } = useMember();
  const { isMobile } = usePlatformOS();
  const [rendered, setRendered] = useState(renderByDefault);

  // what the button says: the member picked, how many are, or the placeholder
  const getDisplayName = () => {
    if (Array.isArray(value)) {
      if (value.length === 0) return placeholder;
      if (value.length === 1) return getUserDetails(value[0])?.display_name || placeholder;
      return showUserDetails ? `${value.length} ${t("members").toLocaleLowerCase()}` : "";
    }
    return showUserDetails && value ? getUserDetails(value)?.display_name || placeholder : placeholder;
  };

  // the button: alone while the dropdown is deferred, then in the select's button
  const button = (
    <DropdownButton
      // shown as active while the list is open (Headless UI's data-open on the select's button)
      className={cn("text-11 group-data-open:bg-layer-transparent-active", buttonClassName)}
      isActive={false}
      tooltipHeading={placeholder}
      tooltipContent={tooltipContent ?? `${value?.length ?? 0} ${value?.length !== 1 ? t("assignees") : t("assignee")}`}
      showTooltip={showTooltip}
      variant={buttonVariant}
      renderToolTipByDefault={renderByDefault}
      tabIndex={tabIndex}
    >
      {!hideIcon && <ButtonAvatars showTooltip={showTooltip} userIds={value} icon={icon} />}
      {BUTTON_VARIANTS_WITH_TEXT.includes(buttonVariant) && (
        <span className="flex-grow truncate text-left text-body-xs-medium leading-5">{getDisplayName()}</span>
      )}
      {dropdownArrow && (
        <ChevronDownOutline className={cn("h-2.5 w-2.5 flex-shrink-0", dropdownArrowClassName)} aria-hidden="true" />
      )}
    </DropdownButton>
  );
  if (!rendered) {
    return (
      <div className="flex h-full items-center" onMouseEnter={() => setRendered(true)}>
        {button}
      </div>
    );
  }

  // the value and its handler as the select takes them: one member, or many
  const selection:
    | { multiple: false; value: string | null; onChange: (val: string | null) => void }
    | { multiple: true; value: string[]; onChange: (val: string[]) => void } = props.multiple
    ? { multiple: true, value: props.value, onChange: props.onChange }
    : { multiple: false, value: props.value, onChange: props.onChange };

  const options = sortByCurrentUserThenSelected<ICustomSearchSelectOption>(
    memberIds?.map((userId) => {
      const userDetails = getUserDetails(userId);
      const suspended = isUserSuspended(userId, workspaceSlug);
      return {
        value: userId,
        query: `${userDetails?.display_name} ${userDetails?.first_name} ${userDetails?.last_name}`,
        content: (
          <div className="flex items-center gap-2">
            <div className="w-4">
              {suspended ? (
                <DeactivatedUserOutline className="h-3.5 w-3.5 text-placeholder" />
              ) : (
                <Avatar
                  alt={userDetails?.display_name}
                  fallback={userDetails?.display_name?.[0]?.toUpperCase()}
                  src={getFileURL(userDetails?.avatar_url ?? "")}
                  size="xs"
                />
              )}
            </div>
            <span className={cn("flex-grow truncate", suspended ? "text-placeholder" : "")}>
              {currentUser?.id === userId ? t("you") : userDetails?.display_name}
            </span>
            {suspended && (
              <Pill variant={EPillVariant.DEFAULT} size={EPillSize.XS} className="border-none">
                Suspended
              </Pill>
            )}
          </div>
        ),
        disabled: suspended,
      };
    }),
    value,
    currentUser?.id
  );

  return (
    <CustomSearchSelect
      {...selection}
      options={options}
      onOpen={onDropdownOpen}
      onClose={onClose}
      disabled={disabled}
      // Tab stops once, at the DropdownButton in it, a button of its own, as in every dropdown of dropdowns/, where
      // the caller's tabIndex puts it: its Enter and Space reach the select's button, which opens the list
      tabIndex={-1}
      placement={placement}
      className={cn("h-full", className)}
      customButtonClassName={cn(
        "clickable group block h-full w-auto max-w-full outline-none",
        buttonContainerClassName
      )}
      optionsClassName={cn("w-48", optionsClassName)}
      searchPlaceholder={t("search")}
      noResultsMessage={t("no_matching_results")}
      loadingMessage={t("loading")}
      // on a phone the search takes no focus as the list opens: the keyboard would cover the list
      focusSearchOnOpen={!isMobile}
      // the list keeps 12 pixels inside the window
      popperModifiers={[{ name: "preventOverflow", options: { padding: 12 } }]}
      customButton={button}
    />
  );
});
````

`web/apps/web/core/components/dropdowns/member/member-options.tsx`（删除）：

````delete web/apps/web/core/components/dropdowns/member/member-options.tsx
````

`web/apps/web/core/components/dropdowns/member/types.d.ts`（修改，1 处）：

````old web/apps/web/core/components/dropdowns/member/types.d.ts
export type MemberDropdownProps = TDropdownProps & {
  button?: React.ReactNode;
````
````new web/apps/web/core/components/dropdowns/member/types.d.ts
export type MemberDropdownProps = TDropdownProps & {
````

`web/apps/web/core/lib/fake-controls.ts`（修改，3 处）：

````old web/apps/web/core/lib/fake-controls.ts
type Select = Field & { children?: ReactNode };
````
````new web/apps/web/core/lib/fake-controls.ts
type Select = Field & { children?: ReactNode };
/** A search select: what it gives, and how its list opens. */
type SearchSelect = Field & { focusSearchOnOpen?: boolean; popperModifiers?: object[] };
````
````old web/apps/web/core/lib/fake-controls.ts
  searchSelects: Field[];
````
````new web/apps/web/core/lib/fake-controls.ts
  searchSelects: SearchSelect[];
````
````old web/apps/web/core/lib/fake-controls.ts
export function CustomSearchSelect(props: Field) {
````
````new web/apps/web/core/lib/fake-controls.ts
export function CustomSearchSelect(props: SearchSelect) {
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 356",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 350",
````

- [ ] **Step 3: 端到端和非空断言的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/emojibase.test.ts"
````
````new .oxlintrc.json
        "web/apps/web/emojibase.test.ts",
        "web/apps/web/core/components/dropdowns/member/base.test.tsx",
        "web/apps/web/core/lib/fake-controls.ts"
````

`e2e/fixtures/settings-pages.ts`（修改，1 处）：

````old e2e/fixtures/settings-pages.ts
  expect(Math.min(left, right), "between the left edges or the right edges").toBeLessThan(2);
}

/**
````
````new e2e/fixtures/settings-pages.ts
  expect(Math.min(left, right), "between the left edges or the right edges").toBeLessThan(2);
}

/** Waits for the transitions of what locator finds, and of what it holds, to end (a modal's enter transition). */
export async function transitionsEnded(locator: Locator): Promise<void> {
  await locator.evaluate((element) =>
    Promise.all(element.getAnimations({ subtree: true }).map((animation) => animation.finished))
  );
}

/**
````

`e2e/fixtures/workspace-pages.ts`（修改，2 处）：

````old e2e/fixtures/workspace-pages.ts
import { answerTo, sentTo } from "./settings-pages";
````
````new e2e/fixtures/workspace-pages.ts
import { answerTo, expectListBesideButton, sentTo } from "./settings-pages";
````
````old e2e/fixtures/workspace-pages.ts
  return answerTo(page, "DELETE", `/api/v0/workspaces/${slug}`, () => confirmDeletion(page, name));
````
````new e2e/fixtures/workspace-pages.ts
  return answerTo(page, "DELETE", `/api/v0/workspaces/${slug}`, () => confirmDeletion(page, name));
}

/**
 * Picks a time zone on the workspace's general page, which page shows, by the keyboard alone (M3 design 7.7, the M2
 * closeout's section 14): Tab reaches the select's button, named by the zone current, from the size's (the address
 * between them is disabled); Enter opens the list beside it with its search focused; Escape closes it and gives the
 * button the focus back; then typed filters the list, and the arrows and Enter pick its first option, which closes it.
 */
export async function pickTimeZone(page: Page, current: string, typed: string): Promise<void> {
  await page.keyboard.press("Tab");
  const timezone = page.getByRole("button", { name: current });
  await expect(timezone).toBeFocused();
  await page.keyboard.press("Enter");
  await expectListBesideButton(page, timezone);
  await expect(page.getByRole("combobox", { name: "Search" })).toBeFocused();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await expect(timezone).toBeFocused();
  await page.keyboard.press("Enter");
  await page.keyboard.type(typed);
  await page.keyboard.press("ArrowDown");
  await page.keyboard.press("Enter");
  await expect(page.getByRole("listbox")).toHaveCount(0);
````

`e2e/stories/project/p1-create-project.spec.ts`（修改，2 处）：

````old e2e/stories/project/p1-create-project.spec.ts
import { registerOnboarded } from "../../fixtures/settings-pages";
````
````new e2e/stories/project/p1-create-project.spec.ts
import { expectListBesideButton, registerOnboarded, transitionsEnded } from "../../fixtures/settings-pages";
````
````old e2e/stories/project/p1-create-project.spec.ts
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});

````
````new e2e/stories/project/p1-create-project.spec.ts
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});

test("P1 (page): the lead's list opens beside its button with its search focused; Escape closes it", async ({
  api,
  signedInPage,
}, testInfo) => {
  const member = await registerOnboarded(api, emailFor(testInfo, "member"));
  const slug = slugFor(testInfo);
  await createWorkspace(api, member.access_token, { name: "Acme", slug });
  const page = await signedInPage(member);
  const watch = await watchPage(page);

  await page.goto(`/${slug}/projects`);
  await page.getByRole("button", { name: "Add Project", exact: true }).click();
  // Popper places the list where the button is as it opens: the modal's enter transition moves the button.
  await transitionsEnded(page.getByRole("dialog"));
  // The button Tab reaches is the DropdownButton's, in the select's (each dropdown of dropdowns/ nests one), which is
  // out of the Tab order: Tab stops once.
  const lead = page.getByRole("button", { name: "Lead", exact: true }).last();
  await expect(lead.locator("xpath=ancestor::button")).toHaveAttribute("tabindex", "-1");
  await lead.click();
  await expectListBesideButton(page, lead);
  const search = page.getByRole("combobox", { name: "Search" });
  await expect(search).toBeFocused();
  // Escape closes the list and leaves the modal open; so does it once a search has opened the Combobox.
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await lead.press("Enter");
  await expectListBesideButton(page, lead);
  await page.keyboard.type("nobody");
  // An Escape that ends an input method's composition (Chinese, say) is the input method's: the list stays open.
  await search.dispatchEvent("keydown", { key: "Escape", isComposing: true });
  await page.keyboard.type("!");
  await expect(search).toHaveValue("nobody!");
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Create project" })).toBeVisible();
  // opened again, the list has no search
  await lead.press("Enter");
  await expect(search).toHaveValue("");
  await expect(page.getByRole("option", { name: "You" })).toBeVisible();

  expect([watch.cspViolations, watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});

````

`e2e/stories/workspace/w3-workspace-settings.spec.ts`（修改，2 处）：

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
import { anotherBrowser, confirmDeletion, deleteFromGeneralPage } from "../../fixtures/workspace-pages";
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
import { anotherBrowser, confirmDeletion, deleteFromGeneralPage, pickTimeZone } from "../../fixtures/workspace-pages";
````
````old e2e/stories/workspace/w3-workspace-settings.spec.ts
  await page.getByRole("button", { name: "UTC" }).click();
  await page.getByRole("combobox", { name: "Search" }).fill("Asia/Shanghai");
  await page.getByRole("option", { name: "Beijing" }).click();
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
  // The time zone by the keyboard alone (M3 design 7.7): the button then names the zone picked, and keeps the focus.
  await pickTimeZone(page, "UTC", "Asia/Shanghai");
  await expect(page.getByRole("button", { name: "Beijing" })).toBeFocused();
````

- [ ] **Step 4: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 67 条规则、3 个例外，没有命中；web 的 oxlint 350 条，等于新的上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 95 个全部通过。

- [ ] **Step 5: 提交**

```bash
git add .oxlintrc.json e2e/fixtures/settings-pages.ts e2e/fixtures/workspace-pages.ts e2e/stories/project/p1-create-project.spec.ts e2e/stories/workspace/w3-workspace-settings.spec.ts web/apps/web/core/components/dropdowns/buttons.tsx web/apps/web/core/components/dropdowns/member/base.test.tsx web/apps/web/core/components/dropdowns/member/base.tsx web/apps/web/core/components/dropdowns/member/member-options.tsx web/apps/web/core/components/dropdowns/member/types.d.ts web/apps/web/core/lib/fake-controls.ts web/apps/web/package.json web/packages/ui/src/breadcrumbs/navigation-search-dropdown.tsx web/packages/ui/src/dropdowns/custom-search-select.tsx web/packages/ui/src/dropdowns/helper.tsx
```
```bash
git commit -m "fix(M3/P10): the search select is a popover's button Tab reaches, its list beside it; the member dropdown is built on it

CustomSearchSelect's button is a Popover's, which Tab reaches and the
keyboard opens; the search and the options are a Combobox in its panel,
whose open state is the popover's alone. The search takes the focus as
the list opens, Escape closes it from the button or the search (not one
that ends an input method's composition), a pick of a single value closes
it, and it opens again with no search; popper places the panel itself.
The member dropdown is built on it: member-options, whose list stayed at
the page's corner, is deleted, and Tab stops once, at the DropdownButton,
where the caller's tabIndex puts it. What its callers relied on stays: a
deferred dropdown is its button alone until hovered, a phone's search
takes no focus, popper keeps the list 12 pixels inside the window. A
breadcrumb is one Tab stop.
P1's lead list and W3's time zone by the keyboard alone.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p10.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T2.1` | 带搜索的下拉框的按钮像 Combobox 的按钮一样不在 Tab 顺序里 | 故事 W3 | 端到端 |
| `T2.2` | 输入打开 Combobox 之后，在搜索框按 Escape，列表仍开着 | 故事 A9、故事 P1 | 端到端 |
| `T2.3` | 单选下拉框选了一项之后列表仍开着 | 故事 A8、故事 W3 | 端到端 |
| `T2.4` | popper 不定位：列表留在页面开头 | 故事 A8、故事 A9、故事 W3、故事 P1 | 端到端 |
| `T2.5` | 列表打开时搜索框不取得焦点 | 故事 W3、故事 P1 | 端到端 |
| `T2.6` | 成员下拉框里的选择按钮也在 Tab 顺序里：负责人一格要按两次 Tab | 故事 P1 | 端到端 |
| `T2.7` | 列表再打开时还留着上次输入的搜索 | 故事 P1 | 端到端 |
| `T2.8` | 成员下拉框丢掉调用方给的 `tabIndex` | 故事 P1（Task 4 起） | 端到端 |
| `T2.9` | 带搜索的下拉框用自定义按钮时不在 Tab 顺序里（添加成员的选择） | 故事 P5（Task 8 起） | 端到端 |
| `T2.10` | 调用方推迟的成员下拉框立即渲染选择和 popper（每一行都有） | `base.test.tsx` | vitest |
| `T2.11` | 手机上成员下拉框打开时搜索框也取得焦点 | `base.test.tsx` | vitest |
| `T2.12` | 成员下拉框的列表离窗口边缘没有 12 像素 | `base.test.tsx` | vitest |
| `T2.13` | 输入法组字时的 Escape 关上列表 | 故事 P1 | 端到端 |
| `T2.14` | 面包屑中不是最后一节的标题也在 Tab 顺序里：一节有两个停留点 | 故事 P2（Task 3 起） | 端到端 |
| `T2.15` | 在成员下拉框的测试里写一个非空断言 | oxlint（`check:lint`） | 静态 |
| `T2.16` | 输入法组字时的 Escape 传到列表之外：`document` 上听 Escape 的弹窗关上（Task 2 的修正轮，裁定 T2-f） | 故事 P1 | 端到端 |
| `T2.17` | 选择的按钮由键盘取得焦点时没有样式：调用方去掉了轮廓的（成员下拉框、面包屑、项目页头）看不出焦点（修正轮，裁定 T2-b） | 故事 P1 | 端到端 |
| `T2.18` | 成员下拉框的按钮不在选择的按钮里（修正轮，裁定 T2-e） | `base.test.tsx` | vitest |

修正轮的这三行之外，修正轮改的几处没有会失败的检查，只由评审看住（spec 附录 A.2）：调用方的 `optionsClassName` 到达 popper 定位的面板（M3 没有给列表 z-index 的调用方，vitest 的替身不渲染）；成员下拉框的根在行中可以收缩、它的选择按钮没有自己的悬停底色（类名）；`transitionsEnded` 不等无限的动画（测试的辅助函数）。

---

### Task 3: 项目列表、卡片和加入；包装层的界面；复制链接；P2 的页面版本

**Files:**
- Create: `e2e/fixtures/mounts.ts`、`web/apps/web/core/components/project/use-join-project.test.ts`、`web/apps/web/core/components/project/use-join-project.ts`、`web/apps/web/core/hooks/use-copy-link.test.ts`、`web/apps/web/core/hooks/use-copy-link.ts`
- Modify: `.oxlintrc.json`、`e2e/fixtures/api.ts`、`e2e/stories/project/p2-visibility.spec.ts`、`e2e/stories/smoke/s2-web-app.spec.ts`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx`、`web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx`、`web/apps/web/core/components/navigation/use-project-actions.ts`、`web/apps/web/core/components/project/card.tsx`、`web/apps/web/core/components/project/join-project-modal.tsx`、`web/apps/web/core/components/workspace/settings/workspace-details.tsx`、`web/apps/web/core/components/workspace/sidebar/projects-list.tsx`、`web/apps/web/core/hooks/use-copy-invitation-link.ts`、`web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx`、`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`、`web/apps/web/core/layouts/auth-layout/use-project-fetch.test.ts`、`web/apps/web/core/layouts/auth-layout/use-project-fetch.ts`、`web/apps/web/core/lib/reconciled.ts`、`web/apps/web/core/store/project/project.store.test.ts`、`web/apps/web/core/store/project/project.store.ts`、`web/packages/i18n/src/locales/en/empty-state.json`、`web/packages/i18n/src/locales/zh-CN/empty-state.json`

**Interfaces:**
- Produces（spec 2.3；M3 设计 2 的 P2，3.4、3.5、3.19，7.1、7.6、7.7；M2 收尾交接第 14 节；P8b 的 F-3；Codex 4.3 第 1 条）：
  - `core/hooks/use-copy-link.ts`：`useCopyLink(): (link, copied) => Promise<void>`：`copyUrlToClipboard` 写剪贴板，成功提示 `common.link_copied` 和 `copied`，浏览器不让写（没有权限、纯 http）时提示失败（`toast.error`、`something_went_wrong_please_try_again`），自己不拒绝；`useCopyProjectLink(): (projectId) => Promise<void>`：地址的工作区中项目的工作项页的链接，成功的说明是 `project_link_copied_to_clipboard`。M2 交接第 14 节的三处（侧边栏 `projects-list.tsx`、扩展侧边栏、卡片）和项目页头的菜单（`use-project-actions.ts`）经它；P9 的邀请行（`use-copy-invitation-link.ts`）和工作区 general 页的地址经 `useCopyLink`，各自的成功、失败的写法删除（一处复制，不是几份）。卡片的复制按钮有名字（`copy_link`）。
  - `core/components/project/use-join-project.ts`：`useJoinProject(): (projectId, done?) => Promise<void>`：`joinProject(projectId)` 经 `followInSession`：加入之后做页面的下一步（`done`：卡片的对话框打开项目；项目自己的页面在 store 把调用者算作成员之后显示），拒绝按 `code` 提示；忙到跟进做完，自己不拒绝。
  - `JoinProjectModal`（整个文件）：经 `useJoinProject`，加入之后关闭、打开项目的工作项页；加入在途时关不掉（取消禁用，`handleClose` 为空：Escape、背景不关），按钮加载。
  - `useProjectFetch`：`ProjectAccess` 加 `{ kind: "archived"; project }`：项目已归档时（不论角色）是它，不取成员才读的（显示设置、标签、成员、状态）：已归档项目的页面不显示，只读项目本身（W2、Codex 4.3 第 1 条：页面只取角色能读的）。
  - `ProjectAuthWrapper`（整个文件）照 `kind` 渲染：`not-member` 是加入界面（经 `useJoinProject`，按钮忙到跟进做完），`archived` 是已归档的界面（`project_empty_state.archived.*`，按钮去 `/{slug}/projects/archives`），`not-found` 照旧。`project-access-restriction.tsx` 给出这三种。
  - `core/lib/reconciled.ts`：`Reconciled.answeredAt`（一个值显示的是第几个写入的回答；所有 `Reconciled` 共用一个计数，0 是还没取到）和 `ReconciledByKey.answeredAt(key)`：扩展这份实现，不复制（总体设计 7.7）。
  - 项目 store 的 `getProjectById`：在项目自己的读取和它的工作区的两个列表中，给出 nerve 最后回答的那一份（P8b 的 F-3：自己的读取之后再取的列表显示其间的修改，例如角色）；store 确认的修改照旧写进每一份；不在调用者的工作区里的照旧不给。
  - i18n（en、zh-CN 的 `empty-state.json`）：`project_empty_state.archived.title`、`description`、`cta_primary`。
  - 静态检查：`.oxlintrc.json` 的非空断言的范围加上 `use-join-project.ts`、`use-copy-link.ts` 和它们的测试。
- e2e 的共用部分：`e2e/fixtures/api.ts` 的 `archiveProject`（经 API 归档）；新文件 `e2e/fixtures/mounts.ts`：S2 的 `REQUESTS` 由它的列表组成（`APP`、`WORKSPACE`、`PROJECT`、`PROJECT_MEMBER`、`GENERAL`、`ARCHIVED`、`INVITATIONS` 和 `valued` 从 S2 移来），P2 的页面版本读同一份 `PROJECT_MEMBER`，不另写清单和记录器（spec 第 3 节第 16 条）。

**Tests:**
- vitest：`use-copy-link.test.ts`：`useCopyLink > writes the link to the clipboard and says what was copied`；`… > says it could not when the browser does not let the page write the clipboard, and does not reject`；`useCopyProjectLink > copies the link of the project's work items in the address's workspace`。`use-join-project.test.ts`：`joins the project, then does what the page does next`；`joins the project with nothing next where the store's answer is the page's`；`shows nerve's reason when it refuses, and does nothing next`；`does nothing on the page when the join $settles after another tab moved this one to another account`（`lateSettlings`）。`project-wrapper.test.tsx` 加 `shows that the project is archived, whose button opens the archived projects, to a member or anyone`。`use-project-fetch.test.ts`：取数的表加一行（已归档、调用者是成员时只取项目本身），决定的表加两行（已归档压过成员关系，也压过加入）。`project.store.test.ts`：`gives a project's own read nerve answered after its list, and holds nothing when nerve refuses it`（改写原来的"自己的读取在先"）；`gives the copy of a list nerve answered after the project's own read: a change made meanwhile shows (F-3)`。
- 端到端：`P2 (page): a member opens a public project he is no member of by its address: the join screen, which reads the project alone; joined, its pages show and read the rest; a private project is not found, an archived one says so`；`P2 (page): the projects page shows a member the public project to join and not the private one; its card copies its link, or says it could not, and joins it, the dialog held until nerve answers`（`sentHeld` 扣住加入：取消禁用、Escape 不关，只发一次；不是成员时和已归档的项目都不取 `PROJECT_MEMBER`，`watchPage` 的 `apiRequests`；加入之后工作项页的页头中项目的面包屑是一个 Tab 停留点：从它按 Tab 到下一节"Work Items"，Task 2 的面包屑）。S2：成员、访客、不是项目成员的成员打开项目列表（`ARCHIVED` 的取数对每个角色），`S2: the admin of a project opens it archived: the archived screen, and the project's read alone, not what its members read`。

- [ ] **Step 1: 复制链接**

`web/apps/web/core/hooks/use-copy-invitation-link.ts`（整个文件，22 行）：

````whole web/apps/web/core/hooks/use-copy-invitation-link.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { WorkspaceInvitation } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
// hooks
import { useCopyLink } from "@/hooks/use-copy-link";
// lib
import { invitationLink } from "@/lib/invitation-link";

/** Copies an invitation's link (invitationLink, at this origin) to the clipboard, and says whether it could. */
export function useCopyInvitationLink(): (invitation: Pick<WorkspaceInvitation, "id" | "token">) => Promise<void> {
  const { t } = useTranslation();
  const copyLink = useCopyLink();
  return (invitation) =>
    copyLink(
      invitationLink(window.location.origin, invitation),
      t("entity.link_copied_to_clipboard", { entity: t("common.invite") })
    );
}
````

`web/apps/web/core/hooks/use-copy-link.test.ts`（新文件，47 行）：

````file web/apps/web/core/hooks/use-copy-link.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { toasts } from "@/lib/fake-toast";
import { useCopyLink, useCopyProjectLink } from "./use-copy-link";

// Copying a link (M3 design 7.7): the hooks run as plain functions, with stand-ins for the clipboard's write, which
// the browser allows or refuses as the test says, and for the address's workspace.

const browser = vi.hoisted(() => ({ copyUrlToClipboard: vi.fn() }));
vi.mock("@nerve/utils", () => ({ copyUrlToClipboard: browser.copyUrlToClipboard }));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "acme" }) }));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

beforeEach(() => {
  browser.copyUrlToClipboard.mockReset();
  browser.copyUrlToClipboard.mockResolvedValue(undefined);
  toasts.length = 0;
});

describe("useCopyLink", () => {
  it("writes the link to the clipboard and says what was copied", async () => {
    await useCopyLink()("/acme/projects/p-web/issues", "the copied");
    expect(browser.copyUrlToClipboard.mock.calls).toEqual([["/acme/projects/p-web/issues"]]);
    expect(toasts).toEqual([{ type: "success", title: "common.link_copied", message: "the copied" }]);
  });

  it("says it could not when the browser does not let the page write the clipboard, and does not reject", async () => {
    browser.copyUrlToClipboard.mockRejectedValueOnce(new DOMException("Write permission denied.", "NotAllowedError"));
    await expect(useCopyLink()("/acme/projects/p-web/issues", "the copied")).resolves.toBeUndefined();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "something_went_wrong_please_try_again" }]);
  });
});

describe("useCopyProjectLink", () => {
  it("copies the link of the project's work items in the address's workspace", async () => {
    await useCopyProjectLink()("p-web");
    expect(browser.copyUrlToClipboard.mock.calls).toEqual([["/acme/projects/p-web/issues"]]);
    expect(toasts).toEqual([
      { type: "success", title: "common.link_copied", message: "project_link_copied_to_clipboard" },
    ]);
  });
});
````

`web/apps/web/core/hooks/use-copy-link.ts`（新文件，40 行）：

````file web/apps/web/core/hooks/use-copy-link.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useParams } from "react-router";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { copyUrlToClipboard } from "@nerve/utils";

/**
 * Copies a link to the clipboard, and says whether it could (M3 design 7.7): copied names what was copied, the
 * success's message; the browser may not let the page write the clipboard (no permission, or plain http), which the
 * failure says. The link is a path of this origin, or a whole address.
 */
export function useCopyLink(): (link: string, copied: string) => Promise<void> {
  const { t } = useTranslation();
  return async (link, copied) => {
    try {
      await copyUrlToClipboard(link);
      setToast({ type: TOAST_TYPE.SUCCESS, title: t("common.link_copied"), message: copied });
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

/** Copies the link of a project of the address's workspace, its work items' page, as useCopyLink. */
export function useCopyProjectLink(): (projectId: string) => Promise<void> {
  const { workspaceSlug } = useParams();
  const { t } = useTranslation();
  const copyLink = useCopyLink();
  return (projectId) =>
    copyLink(`/${workspaceSlug}/projects/${projectId}/issues`, t("project_link_copied_to_clipboard"));
}
````

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx`（修改，4 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
import { AddOutline, SearchOutline } from "@makeplane/propel/icons";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import { copyUrlToClipboard } from "@nerve/utils";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
import { AddOutline, SearchOutline } from "@makeplane/propel/icons";
import { Tooltip } from "@makeplane/propel/components/tooltip";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
import { useUserPermissions } from "@/hooks/store/user";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
import { useUserPermissions } from "@/hooks/store/user";
import { useCopyProjectLink } from "@/hooks/use-copy-link";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
  const handleCopyText = (projectId: string) => {
    copyUrlToClipboard(`${workspaceSlug}/projects/${projectId}/issues`).then(() =>
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("link_copied"),
        message: t("project_link_copied_to_clipboard"),
      })
    );
  };
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
  const copyProjectLink = useCopyProjectLink();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
                  handleCopyText={() => handleCopyText(projectId)}
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx
                  handleCopyText={() => void copyProjectLink(projectId)}
````

`web/apps/web/core/components/navigation/use-project-actions.ts`（整个文件，44 行）：

````whole web/apps/web/core/components/navigation/use-project-actions.ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useState } from "react";
import { useTranslation } from "@nerve/i18n";
import type { TNavigationItem } from "@/components/navigation/tab-navigation-root";
import { useCopyLink } from "@/hooks/use-copy-link";

type UseProjectActionsProps = {
  workspaceSlug: string;
  projectId: string;
  activeItem?: TNavigationItem;
};

export const useProjectActions = ({ workspaceSlug, projectId, activeItem }: UseProjectActionsProps) => {
  const [leaveProjectModalOpen, setLeaveProjectModalOpen] = useState(false);
  const { t } = useTranslation();
  const copyLink = useCopyLink();

  const handleLeaveProject = useCallback(() => {
    setLeaveProjectModalOpen(true);
  }, []);

  // the link of the tab open, else of the project's work items
  const handleCopyText = () =>
    copyLink(
      activeItem?.href ?? `/${workspaceSlug}/projects/${projectId}/issues`,
      t("project_link_copied_to_clipboard")
    );

  const handleLeaveProjectModal = useCallback((open: boolean) => {
    setLeaveProjectModalOpen(open);
  }, []);

  return {
    leaveProjectModalOpen,
    handleLeaveProject,
    handleCopyText,
    handleLeaveProjectModal,
  };
};
````

`web/apps/web/core/components/workspace/settings/workspace-details.tsx`（修改，4 处）：

````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
import { cn, copyUrlToClipboard, getFileURL, validateWorkspaceName } from "@nerve/utils";
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
import { cn, getFileURL, validateWorkspaceName } from "@nerve/utils";
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
import { useUserPermissions } from "@/hooks/store/user";
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
import { useUserPermissions } from "@/hooks/store/user";
import { useCopyLink } from "@/hooks/use-copy-link";
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx
  const toastRefusal = useRefusalToast();
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
  const toastRefusal = useRefusalToast();
  const copyLink = useCopyLink();
````
````old web/apps/web/core/components/workspace/settings/workspace-details.tsx

    void copyUrlToClipboard(`${workspace.slug}`)
      .then(() => {
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: "Workspace URL copied to the clipboard.",
        });
        return undefined;
      })
      // the browser did not let the page write the clipboard
      .catch(() =>
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("toast.error"),
          message: t("something_went_wrong_please_try_again"),
        })
      );
````
````new web/apps/web/core/components/workspace/settings/workspace-details.tsx
    void copyLink(workspace.slug, "Workspace URL copied to the clipboard.");
````

`web/apps/web/core/components/workspace/sidebar/projects-list.tsx`（修改，5 处）：

````old web/apps/web/core/components/workspace/sidebar/projects-list.tsx
import { IconButton } from "@nerve/propel/icon-button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
````
````new web/apps/web/core/components/workspace/sidebar/projects-list.tsx
import { IconButton } from "@nerve/propel/icon-button";
````
````old web/apps/web/core/components/workspace/sidebar/projects-list.tsx
import { copyUrlToClipboard, cn } from "@nerve/utils";
````
````new web/apps/web/core/components/workspace/sidebar/projects-list.tsx
import { cn } from "@nerve/utils";
````
````old web/apps/web/core/components/workspace/sidebar/projects-list.tsx
import { useUserPermissions } from "@/hooks/store/user";
````
````new web/apps/web/core/components/workspace/sidebar/projects-list.tsx
import { useUserPermissions } from "@/hooks/store/user";
import { useCopyProjectLink } from "@/hooks/use-copy-link";
````
````old web/apps/web/core/components/workspace/sidebar/projects-list.tsx
  const handleCopyText = (projectId: string) => {
    copyUrlToClipboard(`${workspaceSlug}/projects/${projectId}/issues`).then(() =>
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("link_copied"),
        message: t("project_link_copied_to_clipboard"),
      })
    );
  };
````
````new web/apps/web/core/components/workspace/sidebar/projects-list.tsx
  const copyProjectLink = useCopyProjectLink();
````
````old web/apps/web/core/components/workspace/sidebar/projects-list.tsx
                        handleCopyText={() => handleCopyText(projectId)}
````
````new web/apps/web/core/components/workspace/sidebar/projects-list.tsx
                        handleCopyText={() => void copyProjectLink(projectId)}
````

- [ ] **Step 2: 加入和包装层**

`web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx`（修改，6 处）：

````old web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
import { observer } from "mobx-react";
````
````new web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
import { observer } from "mobx-react";
import { useNavigate, useParams } from "react-router";
````
````old web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
type TProps = {
  /** Whether the caller sees the project and is no member of it (M3 design 3.19): he may join it. */
  canJoin: boolean;
  handleJoinProject: () => void;
  isJoinButtonDisabled: boolean;
};
````
````new web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
/**
 * Why the project's pages do not show (M3 design 3.19, 7.6): the caller sees the project and is no member of it,
 * which he may join; it is archived; or it is not found to him (it does not exist, is deleted, or he does not see it).
 */
type TProps =
  | { kind: "not-member"; handleJoinProject: () => void; isJoinButtonDisabled: boolean }
  | { kind: "archived" }
  | { kind: "not-found" };
````
````old web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
  const { canJoin, handleJoinProject, isJoinButtonDisabled } = props;
````
````new web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
  // router
  const navigate = useNavigate();
  const { workspaceSlug } = useParams();
````
````old web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
  if (canJoin)
````
````new web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
  if (props.kind === "not-member")
````
````old web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
              label: isJoinButtonDisabled
````
````new web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
              label: props.isJoinButtonDisabled
````
````old web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
              onClick: handleJoinProject,
              disabled: isJoinButtonDisabled,
````
````new web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx
              onClick: props.handleJoinProject,
              disabled: props.isJoinButtonDisabled,
            },
          ]}
        />
      </div>
    );

  // the project is archived: its pages show again once restored, from the workspace's archived projects
  if (props.kind === "archived")
    return (
      <div className="grid h-full w-full place-items-center bg-surface-1">
        <EmptyStateDetailed
          title={t("project_empty_state.archived.title")}
          description={t("project_empty_state.archived.description")}
          assetKey="project"
          assetClassName="size-40"
          actions={[
            {
              label: t("project_empty_state.archived.cta_primary"),
              onClick: () => void navigate(`/${workspaceSlug}/projects/archives`),
````

`web/apps/web/core/components/project/card.tsx`（修改，6 处）：

````old web/apps/web/core/components/project/card.tsx
import { Logo } from "@nerve/propel/emoji-icon-picker";
import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
````
````new web/apps/web/core/components/project/card.tsx
import { Logo } from "@nerve/propel/emoji-icon-picker";
````
````old web/apps/web/core/components/project/card.tsx
import { copyUrlToClipboard, cn, getFileURL, renderFormattedDate } from "@nerve/utils";
````
````new web/apps/web/core/components/project/card.tsx
import { useTranslation } from "@nerve/i18n";
import { cn, getFileURL, renderFormattedDate } from "@nerve/utils";
````
````old web/apps/web/core/components/project/card.tsx
import { useMember } from "@/hooks/store/use-member";
````
````new web/apps/web/core/components/project/card.tsx
import { useMember } from "@/hooks/store/use-member";
import { useCopyProjectLink } from "@/hooks/use-copy-link";
````
````old web/apps/web/core/components/project/card.tsx
  const { isMobile } = usePlatformOS();
````
````new web/apps/web/core/components/project/card.tsx
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();
  const copyProjectLink = useCopyProjectLink();
````
````old web/apps/web/core/components/project/card.tsx
  const handleCopyText = () =>
    copyUrlToClipboard(projectLink).then(() =>
      setToast({
        type: TOAST_TYPE.INFO,
        title: "Link Copied!",
        message: "Project link copied to clipboard.",
      })
    );
````
````new web/apps/web/core/components/project/card.tsx
  const handleCopyText = () => void copyProjectLink(project.id);
````
````old web/apps/web/core/components/project/card.tsx
                  className="flex h-6 w-6 items-center justify-center rounded-sm bg-white/10"
````
````new web/apps/web/core/components/project/card.tsx
                  className="flex h-6 w-6 items-center justify-center rounded-sm bg-white/10"
                  aria-label={t("copy_link")}
````

`web/apps/web/core/components/project/join-project-modal.tsx`（整个文件，72 行）：

````whole web/apps/web/core/components/project/join-project-modal.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useNavigate } from "react-router";
// types
import { Button } from "@nerve/propel/button";
import type { Project } from "@nerve/api-client";
// ui
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// local imports
import { useJoinProject } from "./use-join-project";

// type
type TJoinProjectModalProps = {
  isOpen: boolean;
  workspaceSlug: string;
  project: Project;
  handleClose: () => void;
};

export function JoinProjectModal(props: TJoinProjectModalProps) {
  const { handleClose, isOpen, project, workspaceSlug } = props;
  // states
  const [isJoiningLoading, setIsJoiningLoading] = useState(false);
  // hooks
  const join = useJoinProject();
  // router
  const navigate = useNavigate();

  // One join at a time: the dialog is busy, its button loading, until nerve has answered and the page has followed
  // (useJoinProject); joined, the dialog closes and the page opens the project.
  const handleJoin = async () => {
    setIsJoiningLoading(true);
    await join(project.id, async () => {
      handleClose();
      await navigate(`/${workspaceSlug}/projects/${project.id}/issues`);
    });
    setIsJoiningLoading(false);
  };

  // While the join is out the dialog cannot be dismissed (Cancel, Escape, the backdrop): nerve's answer is followed by
  // the dialog that sent it.
  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={isJoiningLoading ? undefined : handleClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.XL}
    >
      <div className="space-y-5 px-5 py-8 sm:p-6">
        <h3 className="text-16 leading-6 font-medium text-primary">Join Project?</h3>
        <p>
          Are you sure you want to join the project <span className="font-semibold break-words">{project?.name}</span>?
          Please click the &apos;Join Project&apos; button below to continue.
        </p>
        <div className="space-y-3" />
      </div>
      <div className="mt-5 flex justify-end gap-2 px-5 pb-8 sm:px-6 sm:pb-6">
        <Button variant="secondary" size="lg" onClick={handleClose} disabled={isJoiningLoading}>
          Cancel
        </Button>
        <Button variant="primary" size="lg" type="submit" onClick={handleJoin} loading={isJoiningLoading}>
          {isJoiningLoading ? "Joining..." : "Join Project"}
        </Button>
      </div>
    </ModalCore>
  );
}
````

`web/apps/web/core/components/project/use-join-project.test.ts`（新文件，64 行）：

````file web/apps/web/core/components/project/use-join-project.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf } from "@/store/project/fake-projects";
import { useJoinProject } from "./use-join-project";

// Joining a project from its card or its own page (M3 design 3.5, 7.1, 7.6): the hook runs as a plain function, with
// a stand-in for the store's join, which nerve settles when the test says. Its session is fake-tab.ts's.

const page = vi.hoisted(() => ({ joinProject: vi.fn(), done: vi.fn() }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => ({ joinProject: page.joinProject }) }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const web = projectOf("WEB", "w-acme", { member_role: 15 });

beforeEach(() => {
  signedIn();
  page.joinProject.mockReset();
  page.joinProject.mockResolvedValue(web);
  page.done.mockReset();
  toasts.length = 0;
});

describe("useJoinProject", () => {
  it("joins the project, then does what the page does next", async () => {
    await useJoinProject()(web.id, page.done);
    expect([page.joinProject.mock.calls, page.done.mock.calls, toasts]).toEqual([[[web.id]], [[web]], []]);
  });

  it("joins the project with nothing next where the store's answer is the page's", async () => {
    await useJoinProject()(web.id);
    expect([page.joinProject.mock.calls, toasts]).toEqual([[[web.id]], []]);
  });

  it("shows nerve's reason when it refuses, and does nothing next", async () => {
    page.joinProject.mockRejectedValueOnce(refusal(404, "project.not_found"));
    await useJoinProject()(web.id, page.done);
    expect([page.done.mock.calls, toasts]).toEqual([
      [],
      [{ type: "error", title: "toast.error", message: "errors.project_not_found" }],
    ]);
  });

  it.each(lateSettlings)(
    "does nothing on the page when the join $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const answer = heldChange<undefined>();
      page.joinProject.mockReturnValueOnce(answer.sent);
      const joined = useJoinProject()(web.id, page.done);
      switchAccount();
      settle(answer);
      await joined;
      expect([page.done.mock.calls, toasts]).toEqual([[], []]);
    }
  );
});
````

`web/apps/web/core/components/project/use-join-project.ts`（新文件，22 行）：

````file web/apps/web/core/components/project/use-join-project.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// hooks
import { useProject } from "@/hooks/store/use-project";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";

/**
 * Joins a project the caller sees and is no member of (M3 design 3.5, 7.6): the store then shows it as he now sees
 * it, a member. The page follows nerve's answer only in the session it was sent in (M3 design 7.1): joined, what the
 * page does next (done: the card's modal opens the project; the project's own pages show once the store has him a
 * member); refused, nerve's reason in a toast. Settles once the follow-up has; never rejects.
 */
export function useJoinProject(): (projectId: string, done?: () => void | Promise<void>) => Promise<void> {
  const { joinProject } = useProject();
  const toastRefusal = useRefusalToast();
  return (projectId, done) => followInSession(() => joinProject(projectId), { done, failed: toastRefusal });
}
````

`web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx`（修改，8 处）：

````old web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx
import { beforeEach, describe, expect, it, vi } from "vitest";
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx
import { beforeEach, describe, expect, it, vi } from "vitest";
import { signedIn } from "@/lib/auth/fake-tab";
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx
// join and not-found screens, which keep the props they were given, and for the toasts.
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx
// join, archived and not-found screens, which keep the props they were given, for the router and for the toasts. Its
// session is fake-tab.ts's.
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx
  const joinProject = vi.fn((_projectId: string) => Promise.resolve());
  return { access, unavailable, screens, joinProject };
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx
  const joinProject = vi.fn((_projectId: string): Promise<unknown> => Promise.resolve());
  const navigate = vi.fn();
  return { access, unavailable, screens, joinProject, navigate };
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx
  },
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx
  },
}));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "acme" }), useNavigate: () => shown.navigate }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx
const seen = projectOf("WEB", "w-acme", { member_role: null });
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx
const seen = projectOf("WEB", "w-acme", { member_role: null });
const archived = projectOf("WEB", "w-acme", { archived_at: "2026-10-02T09:00:00Z" });
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx
beforeEach(() => {
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx
beforeEach(() => {
  signedIn();
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx
  shown.joinProject.mockClear();
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx
  shown.joinProject.mockClear();
  shown.navigate.mockClear();
````
````old web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx
  });

  it("renders the project's pages to its member", () => {
````
````new web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx
  });

  it("shows that the project is archived, whose button opens the archived projects, to a member or anyone", () => {
    expect(render({ kind: "archived", project: archived })).toContain("project_empty_state.archived.title");
    expect(shown.screens).toMatchObject([
      {
        title: "project_empty_state.archived.title",
        description: "project_empty_state.archived.description",
        actions: [{ label: "project_empty_state.archived.cta_primary" }],
      },
    ]);
    shown.screens[0]?.actions?.[0]?.onClick();
    expect(shown.navigate.mock.calls).toEqual([["/acme/projects/archives"]]);
    expect(shown.joinProject).not.toHaveBeenCalled();
  });

  it("renders the project's pages to its member", () => {
````

`web/apps/web/core/layouts/auth-layout/project-wrapper.tsx`（整个文件，65 行）：

````whole web/apps/web/core/layouts/auth-layout/project-wrapper.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { useState } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
// components
import { SessionUnavailable } from "@/components/account/session-unavailable";
import { ProjectAccessRestriction } from "@/components/auth-screens/project/project-access-restriction";
import { useJoinProject } from "@/components/project/use-join-project";
// local imports
import { useProjectFetch } from "./use-project-fetch";

interface IProjectAuthWrapper {
  projectId: string;
  children: ReactNode;
}

export const ProjectAuthWrapper = observer(function ProjectAuthWrapper(props: IProjectAuthWrapper) {
  const { projectId, children } = props;
  // router params
  const { workspaceSlug } = useParams();
  // states
  const [isJoiningProject, setIsJoiningProject] = useState(false);
  // hooks
  const join = useJoinProject();

  // the project side of what every page of a project fetches (M3 design 7.1), and what nerve's read of the project
  // decides it is to the caller (3.19)
  const access = useProjectFetch(workspaceSlug, projectId);

  // One join at a time, the button disabled until nerve has answered and the page has followed (useJoinProject):
  // joined, the store has the caller a member, and the project's pages show.
  const handleJoinProject = async () => {
    setIsJoiningProject(true);
    await join(projectId);
    setIsJoiningProject(false);
  };

  // nerve's read of the project has not answered yet
  if (access.kind === "loading") return null;

  // nerve could not be reached: the page says so, and tries again when asked (M2 design 7.1)
  if (access.kind === "unavailable") return <SessionUnavailable autoRetry={false} onRetry={access.retry} />;

  // a project the caller sees and is no member of, which he may join
  if (access.kind === "not-member") {
    return (
      <ProjectAccessRestriction
        kind="not-member"
        handleJoinProject={() => void handleJoinProject()}
        isJoinButtonDisabled={isJoiningProject}
      />
    );
  }

  // an archived project, or one not found to him
  if (access.kind !== "member") return <ProjectAccessRestriction kind={access.kind} />;

  return <>{children}</>;
});
````

`web/apps/web/core/layouts/auth-layout/use-project-fetch.test.ts`（修改，3 处）：

````old web/apps/web/core/layouts/auth-layout/use-project-fetch.test.ts
const seen = projectOf("WEB", acme.id, { member_role: null });
````
````new web/apps/web/core/layouts/auth-layout/use-project-fetch.test.ts
const seen = projectOf("WEB", acme.id, { member_role: null });
const archived = projectOf("WEB", acme.id, { archived_at: "2026-10-02T09:00:00Z" });
const archivedSeen = { ...archived, member_role: null };
````
````old web/apps/web/core/layouts/auth-layout/use-project-fetch.test.ts
    { when: "it is a project of another of his workspaces", projects: [elsewhere], data: elsewhere },
````
````new web/apps/web/core/layouts/auth-layout/use-project-fetch.test.ts
    { when: "it is a project of another of his workspaces", projects: [elsewhere], data: elsewhere },
    { when: "it is archived, though he is its member", projects: [archived], data: archived },
````
````old web/apps/web/core/layouts/auth-layout/use-project-fetch.test.ts
      shows: { kind: "not-found" },
    },
    { when: "he sees it, no member", projects: [seen], data: seen, shows: { kind: "not-member", project: seen } },
````
````new web/apps/web/core/layouts/auth-layout/use-project-fetch.test.ts
      shows: { kind: "not-found" },
    },
    {
      when: "it is archived: archived wins over his membership",
      projects: [archived],
      data: archived,
      shows: { kind: "archived", project: archived },
    },
    {
      when: "it is archived and he sees it, no member: archived wins over the join",
      projects: [archivedSeen],
      data: archivedSeen,
      shows: { kind: "archived", project: archivedSeen },
    },
    { when: "he sees it, no member", projects: [seen], data: seen, shows: { kind: "not-member", project: seen } },
````

`web/apps/web/core/layouts/auth-layout/use-project-fetch.ts`（修改，6 处）：

````old web/apps/web/core/layouts/auth-layout/use-project-fetch.ts
 * caller does not see it); that he sees it and is no member of it, which he may join; or its pages, a member's.
````
````new web/apps/web/core/layouts/auth-layout/use-project-fetch.ts
 * caller does not see it); that it is archived; that he sees it and is no member of it, which he may join; or its
 * pages, a member's.
````
````old web/apps/web/core/layouts/auth-layout/use-project-fetch.ts
  | { kind: "not-found" }
````
````new web/apps/web/core/layouts/auth-layout/use-project-fetch.ts
  | { kind: "not-found" }
  | { kind: "archived"; project: Project }
````
````old web/apps/web/core/layouts/auth-layout/use-project-fetch.ts
 * the project is to the caller: nerve's read of the project, which decides it; once it says he is a member, his tab
 * bar in the project, its labels, its members and its states, which nerve gives its members alone. A project counts
````
````new web/apps/web/core/layouts/auth-layout/use-project-fetch.ts
 * the project is to the caller: nerve's read of the project, which decides it; once it says he is a member of a
 * project that is not archived, his tab bar in the project, its labels, its members and its states, which nerve gives
 * its members alone (an archived project's pages do not show, so they need none). A project counts
````
````old web/apps/web/core/layouts/auth-layout/use-project-fetch.ts
  // the project's own reads, a member's alone: for anyone else they are nothing to fetch
````
````new web/apps/web/core/layouts/auth-layout/use-project-fetch.ts
  // the project's own reads, a member's alone, of a project not archived: for anything else they are nothing to fetch
````
````old web/apps/web/core/layouts/auth-layout/use-project-fetch.ts
 * another workspace than the address's), whose member_role says whether he is a member.
````
````new web/apps/web/core/layouts/auth-layout/use-project-fetch.ts
 * another workspace than the address's), archived whatever his role, else member_role says whether he is a member.
````
````old web/apps/web/core/layouts/auth-layout/use-project-fetch.ts
  if (project === undefined) return { kind: "not-found" };
````
````new web/apps/web/core/layouts/auth-layout/use-project-fetch.ts
  if (project === undefined) return { kind: "not-found" };
  if (project.archived_at !== null) return { kind: "archived", project };
````

`web/packages/i18n/src/locales/en/empty-state.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/empty-state.json
      "description": "The project you are looking for does not exist."
````
````new web/packages/i18n/src/locales/en/empty-state.json
      "description": "The project you are looking for does not exist."
    },
    "archived": {
      "title": "This project is archived",
      "description": "Its pages show again once an admin of the project restores it from the archived projects.",
      "cta_primary": "Archived projects"
````

`web/packages/i18n/src/locales/zh-CN/empty-state.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/empty-state.json
      "description": "您查找的项目不存在。"
````
````new web/packages/i18n/src/locales/zh-CN/empty-state.json
      "description": "您查找的项目不存在。"
    },
    "archived": {
      "title": "该项目已归档",
      "description": "项目管理员在已归档的项目中恢复它之后，它的页面会重新显示。",
      "cta_primary": "已归档的项目"
````

- [ ] **Step 3: nerve 最后回答的那一份**

`web/apps/web/core/lib/reconciled.ts`（修改，5 处）：

````old web/apps/web/core/lib/reconciled.ts
export type Change<V> = (value: V) => V;

````
````new web/apps/web/core/lib/reconciled.ts
export type Change<V> = (value: V) => V;

/** The answers written so far, of every Reconciled value: each fetch's answer written takes the next number. */
let answersWritten = 0;

````
````old web/apps/web/core/lib/reconciled.ts
  value: V | undefined = undefined;
````
````new web/apps/web/core/lib/reconciled.ts
  value: V | undefined = undefined;
  /**
   * When nerve's answer the value shows was written, as the order of the answers written (0 until fetched): of two
   * values holding the same thing (a project in its list and in its own read), the greater was answered last.
   */
  answeredAt = 0;
````
````old web/apps/web/core/lib/reconciled.ts
    makeObservable(this, { value: observable.ref });
````
````new web/apps/web/core/lib/reconciled.ts
    makeObservable(this, { value: observable.ref, answeredAt: observable });
````
````old web/apps/web/core/lib/reconciled.ts
        this.value = value;
````
````new web/apps/web/core/lib/reconciled.ts
        this.value = value;
        this.answeredAt = ++answersWritten;
````
````old web/apps/web/core/lib/reconciled.ts
    return key === undefined ? undefined : this.entries.get(key)?.value;
````
````new web/apps/web/core/lib/reconciled.ts
    return key === undefined ? undefined : this.entries.get(key)?.value;
  }

  /** When the key's value was answered (Reconciled.answeredAt); 0 for a key not fetched. */
  answeredAt(key: string): number {
    return this.entries.get(key)?.answeredAt ?? 0;
````

`web/apps/web/core/store/project/project.store.test.ts`（修改，2 处）：

````old web/apps/web/core/store/project/project.store.test.ts
  it("gives a project's own read before its list's, and holds nothing when nerve refuses it", async () => {
````
````new web/apps/web/core/store/project/project.store.test.ts
  it("gives a project's own read nerve answered after its list, and holds nothing when nerve refuses it", async () => {
````
````old web/apps/web/core/store/project/project.store.test.ts
    expect(store.getProjectById("p-gone")).toBeUndefined();
````
````new web/apps/web/core/store/project/project.store.test.ts
    expect(store.getProjectById("p-gone")).toBeUndefined();
  });

  it("gives the copy of a list nerve answered after the project's own read: a change made meanwhile shows (F-3)", async () => {
    const { nerve, store } = await loaded();
    await loadProject(nerve, store, web);
    const demoted: Project = { ...web, member_role: 5 };
    await loadProjects(nerve, store, acme, [demoted, ops, docs]);
    expect(store.getProjectById(web.id)).toEqual(demoted);
    // and the project's own read, answered after that list, again
    await loadProject(nerve, store, web, renamed);
    expect(store.getProjectById(web.id)).toEqual(renamed);
````

`web/apps/web/core/store/project/project.store.ts`（修改，6 处）：

````old web/apps/web/core/store/project/project.store.ts
import { sortBy } from "lodash-es";
````
````new web/apps/web/core/store/project/project.store.ts
import { maxBy, sortBy } from "lodash-es";
````
````old web/apps/web/core/store/project/project.store.ts
   * Each project as nerve last read it alone; null once deleted or left: then only a list read after gives it (a fetch
   * out meanwhile drops it from its answer).
````
````new web/apps/web/core/store/project/project.store.ts
   * Each project as nerve last read it alone; null once deleted or left: then only a list gives it (a fetch out
   * meanwhile drops it from its answer).
````
````old web/apps/web/core/store/project/project.store.ts
   * The project as the store last had it from nerve, its own read first, else from its workspace's lists; nothing
   * once deleted or left until a list has it again (a public project, to one no longer its member), or when its
   * workspace is no longer among the caller's.
````
````new web/apps/web/core/store/project/project.store.ts
   * The project as nerve last answered it, of the copies the store holds: its own read's and its workspace's lists'
   * (a list fetched after the read shows a change made meanwhile, a role for one; P8b's F-3); the changes nerve
   * confirmed are made on every copy. Nothing once deleted or left until a list has it again (a public project, to
   * one no longer its member), or when its workspace is no longer among the caller's.
````
````old web/apps/web/core/store/project/project.store.ts
    const workspaces = this.rootStore.workspaceRoot.workspaces ?? [];
````
````new web/apps/web/core/store/project/project.store.ts
    const workspaces = this.rootStore.workspaceRoot.workspaces ?? [];
    const copies: { project: Project; answeredAt: number }[] = [];
````
````old web/apps/web/core/store/project/project.store.ts
    if (read) {
      return workspaces.some((workspace) => workspace.id === read.workspace_id) ? read : undefined;
````
````new web/apps/web/core/store/project/project.store.ts
    if (read) copies.push({ project: read, answeredAt: this.details.answeredAt(projectId) });
    for (const { id } of workspaces) {
      for (const lists of [this.unarchived, this.archived]) {
        const project = lists.get(id)?.find((held) => held.id === projectId);
        if (project) copies.push({ project, answeredAt: lists.answeredAt(id) });
      }
````
````old web/apps/web/core/store/project/project.store.ts
    for (const { id } of workspaces) {
      const listed = [...(this.unarchived.get(id) ?? []), ...(this.archived.get(id) ?? [])];
      const project = listed.find((held) => held.id === projectId);
      if (project) return project;
    }
    return undefined;
````
````new web/apps/web/core/store/project/project.store.ts
    const newest = maxBy(copies, "answeredAt")?.project;
    return newest && workspaces.some((workspace) => workspace.id === newest.workspace_id) ? newest : undefined;
````

- [ ] **Step 4: 静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/lib/fake-controls.ts"
````
````new .oxlintrc.json
        "web/apps/web/core/lib/fake-controls.ts",
        "web/apps/web/core/components/project/use-join-project.ts",
        "web/apps/web/core/components/project/use-join-project.test.ts",
        "web/apps/web/core/hooks/use-copy-link.ts",
        "web/apps/web/core/hooks/use-copy-link.test.ts"
````

- [ ] **Step 5: 端到端**

`e2e/fixtures/api.ts`（修改，1 处）：

````old e2e/fixtures/api.ts
}

/** Creates a state in the project of projectId with the bearer token given, an admin's of the project, and returns it. */
````
````new e2e/fixtures/api.ts
}

/** Archives the project of projectId with the bearer token given, an admin's of the project, and returns it. */
export async function archiveProject(api: Api, token: string, projectId: string): Promise<Project> {
  const { data, error, response } = await api.POST("/api/v0/projects/{project_id}/archive", {
    params: { path: { project_id: projectId } },
    headers: bearer(token),
  });
  expect(response.status, `archive ${projectId}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) {
    throw new Error(`archiveProject ${projectId} answered 200 without the project`);
  }
  return data;
}

/** Creates a state in the project of projectId with the bearer token given, an admin's of the project, and returns it. */
````

`e2e/fixtures/mounts.ts`（新文件，47 行）：

````file e2e/fixtures/mounts.ts
// What the pages ask nerve for as they load: S2's REQUESTS is built of these lists, and a page's story that checks what
// it loads reads them here rather than writing its own.

/**
 * What a page asks nerve for as it loads, signed in, as "<method> <path>" with the query when there is one, {slug} for
 * the workspace's slug and {project} for the project's id: the app's start, on every page (M2 design 7.1, M3 design
 * 7.4); the workspace wrapper's, on every page of a workspace, whatever the role; the project's read, on every page of
 * a project; and the project's own resources, only once that read says the caller is a member of the project (M3
 * design 7.1). No list has an address of M6's or M7's (cycles, modules, views, the intake's triage state, favourites,
 * the unread notifications, recents; M3 design 3.1), nor one outside /api/v0 (M2 design 3.1). These lists are the
 * record of what the pages load: a phase that adds a fetch to a page adds it here (the P8b spec's appendix A.5 is a
 * copy, as of P8b).
 */
export const APP = [
  "POST /api/v0/auth/refresh",
  "GET /api/v0/instance",
  "GET /api/v0/me",
  "GET /api/v0/me/profile",
  "GET /api/v0/workspaces",
];
export const WORKSPACE = [
  "GET /api/v0/me/workspaces/{slug}/preferences",
  "GET /api/v0/workspaces/{slug}/members",
  "GET /api/v0/workspaces/{slug}/projects?archived=false",
  "GET /api/v0/workspaces/{slug}/states",
];
export const PROJECT = ["GET /api/v0/projects/{project}"];
export const PROJECT_MEMBER = [
  "GET /api/v0/me/projects/{project}/preferences",
  "GET /api/v0/projects/{project}/labels",
  "GET /api/v0/projects/{project}/members",
  "GET /api/v0/projects/{project}/states",
];
/**
 * A general settings page lists the time zones (M2 design 5.3): the project's, for its members; the workspace's, for
 * its admins and members (the workspace's settings show a guest no general page, M3 design 9.2).
 */
export const GENERAL = ["GET /api/v0/timezones"];
/** The workspace's projects page lists its archived projects too, its own fetch (useArchivedProjectsFetch), for every role. */
export const ARCHIVED = ["GET /api/v0/workspaces/{slug}/projects?archived=true"];
/** The members page lists the workspace's invitations for an admin alone, as nerve shows them to no one else. */
export const INVITATIONS = ["GET /api/v0/workspaces/{slug}/invitations"];

/** path with the values of names in place of their names: the address of a page of REQUESTS, or a request of a list. */
export function valued(path: string, names: Record<string, string>): string {
  return Object.entries(names).reduce((shown, [value, name]) => shown.replaceAll(name, value), path);
}
````

`e2e/stories/project/p2-visibility.spec.ts`（修改，4 处）：

````old e2e/stories/project/p2-visibility.spec.ts
  amidAnotherWorkspace,
````
````new e2e/stories/project/p2-visibility.spec.ts
  amidAnotherWorkspace,
  archiveProject,
````
````old e2e/stories/project/p2-visibility.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/project/p2-visibility.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, refuseClipboardWrites, watchPage } from "../../fixtures/browser";
import { PROJECT_MEMBER, valued } from "../../fixtures/mounts";
import { closedByEscape, enabledWithin, registerOnboarded, sentHeld } from "../../fixtures/settings-pages";
````
````old e2e/stories/project/p2-visibility.spec.ts
// P2, the projects' list and who sees them, and joining (M3 design 2, 3.4,
// 3.5, 3.19). The page version, with the "join the project" screen, comes
// with the projects' pages (P10).
````
````new e2e/stories/project/p2-visibility.spec.ts
// P2, the projects' list and who sees them, and joining (M3 design 2, 3.4, 3.5, 3.19); the page versions, with the
// "join the project" screen and the projects page's cards (7.6, P10).
````
````old e2e/stories/project/p2-visibility.spec.ts
  await expectMember(db, docs.id, guestEmail, { role: 5, is_active: true, sort_order: 65535, by: adminEmail });
});

````
````new e2e/stories/project/p2-visibility.spec.ts
  await expectMember(db, docs.id, guestEmail, { role: 5, is_active: true, sort_order: 65535, by: adminEmail });
});

/**
 * The reads of the project of projectId that nerve gives its members alone, which its pages make once the caller is one
 * (M3 design 7.1), as watchPage records them (none has a query).
 */
const membersReads = (projectId: string) => PROJECT_MEMBER.map((read) => valued(read, { [projectId]: "{project}" }));

test("P2 (page): a member opens a public project he is no member of by its address: the join screen, which reads the project alone; joined, its pages show and read the rest; a private project is not found, an archived one says so", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = (await registerOnboarded(api, adminEmail)).access_token;
  const memberEmail = emailFor(testInfo, "member");
  const member = await registerOnboarded(api, memberEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug });
  await inviteAndAccept(api, admin, slug, { email: memberEmail, token: member.access_token }, 15);
  const web = await createProject(api, admin, slug, { name: "Web", identifier: "WEB", network: 2 });
  const secret = await createProject(api, admin, slug, { name: "Secret", identifier: "SEC", network: 0 });
  const old = await createProject(api, admin, slug, { name: "Old", identifier: "OLD", network: 2 });
  await archiveProject(api, admin, old.id);

  const page = await signedInPage(member);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/settings/projects/${web.id}`);
  const joinButton = page.getByRole("button", { name: "Join project" });
  await expect(joinButton).toBeVisible();
  // The page read the project, and none of what nerve gives its members alone.
  expect(watch.apiRequests).toContain(`GET /api/v0/projects/${web.id}`);
  expect(watch.apiRequests.filter((request) => membersReads(web.id).includes(request))).toEqual([]);

  // Joining: one request, the button busy until nerve answers; then the project's general page, which reads the rest.
  const { release } = await sentHeld(page, "POST", `/api/v0/projects/${web.id}/join`, () => joinButton.click());
  const joining = page.getByRole("button", { name: "Joining project" });
  expect(await enabledWithin(joining)).toBe(false);
  expect((await release()).status()).toBe(200);
  await expect(page.locator("#name")).toHaveValue("Web");
  expect(watch.apiRequests.filter((request) => request === `POST /api/v0/projects/${web.id}/join`)).toHaveLength(1);
  await expect.poll(() => membersReads(web.id).filter((read) => !watch.apiRequests.includes(read))).toEqual([]);
  await expectMember(db, web.id, memberEmail, { role: 15, is_active: true, sort_order: 65535, by: memberEmail });

  // A private project he does not see is not found; an archived one says so, and its button opens the archived ones.
  await page.goto(`/${slug}/projects/${secret.id}/issues`);
  await expect(page.getByText("Project not found")).toBeVisible();
  await page.goto(`/${slug}/projects/${old.id}/issues`);
  await expect(page.getByText("This project is archived")).toBeVisible();
  expect(watch.apiRequests).toContain(`GET /api/v0/projects/${old.id}`);
  expect(watch.apiRequests.filter((request) => membersReads(old.id).includes(request))).toEqual([]);
  await page.getByRole("button", { name: "Archived projects" }).click();
  await expect(page).toHaveURL(`/${slug}/projects/archives`);

  // The private project's 404, the browser's report of it
  expect([watch.cspViolations, watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [],
    [`404 GET /api/v0/projects/${secret.id}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, {
    // the three loads: the settings page, the private project's, the archived one's
    warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING],
    errors: ["Failed to load resource: the server responded with a status of 404 (Not Found)"],
  });
});

test("P2 (page): the projects page shows a member the public project to join and not the private one; its card copies its link, or says it could not, and joins it, the dialog held until nerve answers", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const admin = (await registerOnboarded(api, emailFor(testInfo, "admin"))).access_token;
  const memberEmail = emailFor(testInfo, "member");
  const member = await registerOnboarded(api, memberEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin, { name: "Acme", slug });
  await inviteAndAccept(api, admin, slug, { email: memberEmail, token: member.access_token }, 15);
  const web = await createProject(api, admin, slug, { name: "Web", identifier: "WEB", network: 2 });
  await createProject(api, admin, slug, { name: "Secret", identifier: "SEC", network: 0 });

  const page = await signedInPage(member);
  await page.context().grantPermissions(["clipboard-read", "clipboard-write"]);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/projects`);
  const card = page.getByRole("link", { name: /Web/ });
  await expect(card).toBeVisible();
  await expect(page.getByRole("link", { name: /Secret/ })).toHaveCount(0);

  // The card copies the project's link; a browser that refuses the page the clipboard: the card says it could not.
  await card.getByRole("button", { name: "Copy link" }).click();
  await expect(page.getByText("Project link copied to clipboard")).toBeVisible();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(
    new URL(`/${slug}/projects/${web.id}/issues`, page.url()).toString()
  );
  await refuseClipboardWrites(page);
  await card.getByRole("button", { name: "Copy link" }).click();
  await expect(page.getByText("Something went wrong. Please try again.")).toBeVisible();

  // Joining from the card: the dialog cannot be dismissed while nerve has not answered, and sends one join.
  await card.getByRole("button", { name: "Join", exact: true }).click();
  const { release } = await sentHeld(page, "POST", `/api/v0/projects/${web.id}/join`, () =>
    page.getByRole("button", { name: "Join Project" }).click()
  );
  await expect(page.getByRole("dialog").getByRole("button", { name: "Cancel" })).toBeDisabled();
  expect(await closedByEscape(page)).toBe(false);
  expect((await release()).status()).toBe(200);
  // joined, the page opens the project's work items
  await expect(page).toHaveURL(`/${slug}/projects/${web.id}/issues`);
  expect(watch.apiRequests.filter((request) => request === `POST /api/v0/projects/${web.id}/join`)).toHaveLength(1);
  await expectMember(db, web.id, memberEmail, { role: 15, is_active: true, sort_order: 65535, by: memberEmail });
  // The project's crumb in the header is one Tab stop, the button of its list (M3 design 7.7): its title, which opens
  // the project's work items by a click, is not another. Tab goes on to the next crumb.
  await page.getByRole("button", { name: "Web", exact: true }).first().focus();
  await page.keyboard.press("Tab");
  await expect(page.getByRole("link", { name: "Work Items", exact: true })).toBeFocused();

  // The work items' page asks Plane's address of the filters, M4's (P8b spec §5)
  const filters = `GET /api/workspaces/${slug}/projects/${web.id}/user-properties/`;
  await expect.poll(() => watch.apiFailures).toEqual([`404 ${filters}`]);
  expect([watch.cspViolations, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [filters], []]);
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING],
    errors: ["Failed to load resource: the server responded with a status of 404 (Not Found)"],
  });
});

````

`e2e/stories/smoke/s2-web-app.spec.ts`（修改，9 处）：

````old e2e/stories/smoke/s2-web-app.spec.ts
  addProjectMembers,
````
````new e2e/stories/smoke/s2-web-app.spec.ts
  addProjectMembers,
  archiveProject,
````
````old e2e/stories/smoke/s2-web-app.spec.ts
} from "../../fixtures/browser";
````
````new e2e/stories/smoke/s2-web-app.spec.ts
} from "../../fixtures/browser";
import { APP, ARCHIVED, GENERAL, INVITATIONS, PROJECT, PROJECT_MEMBER, WORKSPACE, valued } from "../../fixtures/mounts";
````
````old e2e/stories/smoke/s2-web-app.spec.ts
/**
 * What a page asks nerve for as it loads, signed in, as "<method> <path>" with the query when there is one, {slug} for
 * the workspace's slug and {project} for the project's id: the app's start, on every page (M2 design 7.1, M3 design
 * 7.4); the workspace wrapper's, on every page of a workspace, whatever the role; the project's read, on every page of
 * a project; and the project's own resources, only once that read says the caller is a member of the project (M3
 * design 7.1). No list has an address of M6's or M7's (cycles, modules, views, the intake's triage state, favourites,
 * the unread notifications, recents; M3 design 3.1), nor one outside /api/v0 (M2 design 3.1). These lists are the
 * record of what the pages load: a phase that adds a fetch to a page adds it here (the P8b spec's appendix A.5 is a
 * copy, as of P8b).
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
  "GET /api/v0/workspaces/{slug}/projects?archived=false",
  "GET /api/v0/workspaces/{slug}/states",
];
const PROJECT = ["GET /api/v0/projects/{project}"];
const PROJECT_MEMBER = [
  "GET /api/v0/me/projects/{project}/preferences",
  "GET /api/v0/projects/{project}/labels",
  "GET /api/v0/projects/{project}/members",
  "GET /api/v0/projects/{project}/states",
];
/**
 * A general settings page lists the time zones (M2 design 5.3): the project's, for its members; the workspace's, for
 * its admins and members (the workspace's settings show a guest no general page, M3 design 9.2).
 */
const GENERAL = ["GET /api/v0/timezones"];
/** The workspace's projects page lists its archived projects too, its own fetch (useArchivedProjectsFetch). */
const ARCHIVED = ["GET /api/v0/workspaces/{slug}/projects?archived=true"];
/** The members page lists the workspace's invitations for an admin alone, as nerve shows them to no one else. */
const INVITATIONS = ["GET /api/v0/workspaces/{slug}/invitations"];

/**
````
````new e2e/stories/smoke/s2-web-app.spec.ts
/**
````
````old e2e/stories/smoke/s2-web-app.spec.ts
 * makes as it loads: "/", which lands on the workspace's home; for the admin, the workspace's projects page, whose
 * call of its own fetch no other check holds; the workspace's general settings and its members; then the project's
 * settings.
````
````new e2e/stories/smoke/s2-web-app.spec.ts
 * makes as it loads: "/", which lands on the workspace's home; the workspace's projects page, whose call of its own
 * fetch no other check holds; the workspace's general settings and its members; then the project's settings.
````
````old e2e/stories/smoke/s2-web-app.spec.ts
  member: [
    ["/", [...APP, ...WORKSPACE]],
````
````new e2e/stories/smoke/s2-web-app.spec.ts
  member: [
    ["/", [...APP, ...WORKSPACE]],
    ["/{slug}/projects", [...APP, ...WORKSPACE, ...ARCHIVED]],
````
````old e2e/stories/smoke/s2-web-app.spec.ts
  guest: [
    ["/", [...APP, ...WORKSPACE]],
````
````new e2e/stories/smoke/s2-web-app.spec.ts
  guest: [
    ["/", [...APP, ...WORKSPACE]],
    ["/{slug}/projects", [...APP, ...WORKSPACE, ...ARCHIVED]],
````
````old e2e/stories/smoke/s2-web-app.spec.ts
  "project non-member": [
    ["/", [...APP, ...WORKSPACE]],
````
````new e2e/stories/smoke/s2-web-app.spec.ts
  "project non-member": [
    ["/", [...APP, ...WORKSPACE]],
    ["/{slug}/projects", [...APP, ...WORKSPACE, ...ARCHIVED]],
````
````old e2e/stories/smoke/s2-web-app.spec.ts
  return pathname.startsWith("/api/") ? `${pathname}${search}` : undefined;
}

/** path with the values of names in place of their names: the address of a page of REQUESTS. */
function valued(path: string, names: Record<string, string>): string {
  return Object.entries(names).reduce((shown, [value, name]) => shown.replaceAll(name, value), path);
````
````new e2e/stories/smoke/s2-web-app.spec.ts
  return pathname.startsWith("/api/") ? `${pathname}${search}` : undefined;
````
````old e2e/stories/smoke/s2-web-app.spec.ts
});

test("S2: a newcomer opens /, which sends him to the onboarding: it asks for his workspaces, as the app does, and no more", async ({
````
````new e2e/stories/smoke/s2-web-app.spec.ts
});

test("S2: the admin of a project opens it archived: the archived screen, and the project's read alone, not what its members read", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { slug, project, tokens } = await acme(api, testInfo);
  await archiveProject(api, tokens.admin.access_token, project.id);
  const page = await signedInPage(tokens.admin);
  const watch = await watchPage(page);
  const requests = followRequests(page, { [slug]: "{slug}", [project.id]: "{project}" });
  await page.goto(`/${slug}/projects/${project.id}/issues`);
  await expect(page.getByText("This project is archived")).toBeVisible();
  await expectExactly(page, watch, requests, [...APP, ...WORKSPACE, ...PROJECT], [EMOJI_CHECK_WARNING]);
});

test("S2: a newcomer opens /, which sends him to the onboarding: it asks for his workspaces, as the app does, and no more", async ({
````

- [ ] **Step 6: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 67 条规则、3 个例外，没有命中；web 的 oxlint 350 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 98 个全部通过。

- [ ] **Step 7: 提交**

```bash
git add .oxlintrc.json e2e/fixtures/api.ts e2e/fixtures/mounts.ts e2e/stories/project/p2-visibility.spec.ts e2e/stories/smoke/s2-web-app.spec.ts 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/extended-project-sidebar.tsx' web/apps/web/core/components/auth-screens/project/project-access-restriction.tsx web/apps/web/core/components/navigation/use-project-actions.ts web/apps/web/core/components/project/card.tsx web/apps/web/core/components/project/join-project-modal.tsx web/apps/web/core/components/project/use-join-project.test.ts web/apps/web/core/components/project/use-join-project.ts web/apps/web/core/components/workspace/settings/workspace-details.tsx web/apps/web/core/components/workspace/sidebar/projects-list.tsx web/apps/web/core/hooks/use-copy-invitation-link.ts web/apps/web/core/hooks/use-copy-link.test.ts web/apps/web/core/hooks/use-copy-link.ts web/apps/web/core/layouts/auth-layout/project-wrapper.test.tsx web/apps/web/core/layouts/auth-layout/project-wrapper.tsx web/apps/web/core/layouts/auth-layout/use-project-fetch.test.ts web/apps/web/core/layouts/auth-layout/use-project-fetch.ts web/apps/web/core/lib/reconciled.ts web/apps/web/core/store/project/project.store.test.ts web/apps/web/core/store/project/project.store.ts web/packages/i18n/src/locales/en/empty-state.json web/packages/i18n/src/locales/zh-CN/empty-state.json
```
```bash
git commit -m "feat(M3/P10): the projects page's cards copy and join, the wrapper says a project is archived, and a project is nerve's last answer

Copying a link goes through one hook, which says what was copied and
says when the browser would not write the clipboard: the sidebar, the
extended sidebar, the card, the project header's menu, the invitation
row and the workspace's address use it. Joining a project goes through
useJoinProject, followed only in the session it was sent in; the card's
dialog cannot be dismissed while the join is out. The project wrapper
shows an archived project's screen, whose button opens the archived
projects, and reads nothing its members read; a project's own read and
its lists give the copy nerve answered last (reconciled.ts's answeredAt).
P2's page versions and S2's lists, which move to a fixture P2 reads too.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p10.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T3.1` | 加入项目不论会话都跟进 | `use-join-project.test.ts` | vitest |
| `T3.2` | 丢掉 nerve 对加入的拒绝 | `use-join-project.test.ts`、`project-wrapper.test.tsx` | vitest |
| `T3.3` | 加入在途时加入的对话框可以关掉 | 故事 P2 | 端到端 |
| `T3.4` | 加入在途时对话框的取消按钮仍可用 | 故事 P2 | 端到端 |
| `T3.5` | 加入在途时加入界面的按钮不显示忙 | 故事 P2 | 端到端 |
| `T3.6` | 已归档的项目像未归档的一样向成员显示它的页面 | `use-project-fetch.test.ts`、故事 P2、故事 S2 | vitest；端到端 |
| `T3.7` | 已归档项目的页面照成员的取数去取 | `use-project-fetch.test.ts`、故事 S2 | vitest；端到端 |
| `T3.8` | 浏览器拒绝复制时成为未处理的拒绝，不提示 | `use-copy-link.test.ts`、故事 P2、故事 W4 | vitest；端到端 |
| `T3.9` | 项目的链接是另一个地址（少了工作区的 slug） | `use-copy-link.test.ts`、故事 P2 | vitest；端到端 |
| `T3.10` | 项目自己的读取压过 nerve 在它之后回答的列表（F-3） | `project.store.test.ts` | vitest |
| `T3.11` | 不按回答的先后：在自己的读取之后回答的列表不算较新的 | `project.store.test.ts` | vitest |
| `T3.12` | 在加入项目的 hook 里写一个非空断言 | oxlint（`check:lint`） | 静态 |
| `T3.13` | 加入在途时对话框的按钮不显示忙：再点一次又发一次加入（Task 3 的修正轮，裁定 T3-c） | 故事 P2 | 端到端 |
| `T3.14` | 加入的对话框在 nerve 回答之后不论结果都关上，被拒绝时也关（修正轮，裁定 T3-c） | 故事 P2 | 端到端 |
| `T3.15` | 已归档在 nerve 的读取之前决定：删除了的已归档项目显示已归档而不是找不到，nerve 不可达时没有重试，读取回答之前就显示（修正轮，裁定 T3-d） | `use-project-fetch.test.ts` | vitest |
| `T3.16` | 每个值自己计回答的数：取了两次的列表压过在它之后回答的自己的读取（修正轮，裁定 T3-e） | `project.store.test.ts` | vitest |
| `T3.17` | 已归档的列表中的一份只在没有别的副本时才算（修正轮，裁定 T3-e） | `project.store.test.ts` | vitest |

修正轮的这五行之外，修正轮改的 `getProjectById` 的注释（裁定 T3-f）和 spec 第 3 节第 8、15 条复制的提示（裁定 T3-g）是文字，没有变异。修正轮的测试：P2 的卡片的故事中被拒绝的加入（管理员在对话框开着时删除项目，原因提示、对话框留着、按钮重新可用；`settings-pages.ts` 的 `closedWithin`，`closedByEscape` 改为经它）和在途时按钮忙（`enabledWithin`）；`use-project-fetch.test.ts` 决定的表加三行（已归档的项目：nerve 找不到时找不到、不可达时不可达、没回答时加载）；`project.store.test.ts` 加两个测试（列表取了两次之后回答的自己的读取；自己的读取之后回答的已归档的列表）。

---

### Task 4: 创建项目；P1 的页面版本

**Files:**
- Create: `web/apps/web/core/components/project/logo-props.test.ts`、`web/apps/web/core/components/project/logo-props.ts`、`web/apps/web/core/components/project/project-refusal.test.ts`、`web/apps/web/core/components/project/project-refusal.ts`、`web/apps/web/core/components/projects/create/use-create-project.test.ts`、`web/apps/web/core/components/projects/create/use-create-project.ts`、`web/packages/utils/src/project.test.ts`
- Modify: `.oxlintrc.json`、`e2e/stories/project/p1-create-project.spec.ts`、`web/apps/web/core/components/project/create-project-modal.tsx`、`web/apps/web/core/components/project/create/common-attributes.tsx`、`web/apps/web/core/components/project/create/header.tsx`、`web/apps/web/core/components/project/create/project-create-buttons.tsx`、`web/apps/web/core/components/projects/create/attributes.tsx`、`web/apps/web/core/components/projects/create/root.tsx`、`web/apps/web/core/components/projects/create/utils.ts`、`web/packages/propel/src/emoji-icon-picker/helper.tsx`、`web/packages/utils/src/project.ts`

**Interfaces:**
- Produces（spec 2.4；M3 设计 2 的 P1，3.19，7.1、7.6；P8b spec 第 5 节 P10 一行；P4a 评审的标识符检查）：
  - `core/components/project/project-refusal.ts`：`projectRefusal(error): ProjectRefusal`，`{ kind: "fields"; fields: Partial<Record<"name" | "identifier", string>> } | { kind: "toast" }`：名称、标识已被占用（`project.name_taken`、`project.identifier_taken`）落到各自的字段下；nerve 的字段错误只指向名称、标识时落到字段下（`FIELD_ERROR_MESSAGES`）；其余（负责人不被允许等，`needsErrorBanner`）按原因提示。`useProjectRefusal()` 照它显示：字段的经表单的 `setError`，提示的经 `useRefusalToast`。创建和 Task 5 的 general 页共用。
  - `core/components/project/logo-props.ts`：`logoPropsOf(picked: TChangeHandlerProps): LogoProps`：表情只留它的值，图标只留名字和颜色（nerve 的 `LogoProps` 是封闭的）；`@nerve/propel` 导出 `TChangeHandlerProps`。创建的页头和 Task 5 的 general 页都经它。
  - `core/components/projects/create/use-create-project.ts`：`ProjectCreationForm`（`ProjectCreate` 的 `name`、`identifier`、`description`、`network`、`logo_props`，加 `project_lead_id: string | null`）；`useCreateProject(): (workspaceSlug, form, setError) => Promise<Project | undefined>`：请求体由这些字段构成，没选负责人时不发 `project_lead_id`；经 `followInSession`：创建之后提示、交回项目，被拒绝时经 `useProjectRefusal`，换账户之后交回 `undefined`。
  - `CreateProjectForm`（`projects/create/root.tsx`，整个文件）：经 `useCreateProject`，交回项目时进入功能一步；创建在途时 Escape 不关；页头的关闭按钮（有名字）、取消在途时禁用；`create-project-modal.tsx` 的 Escape 只在功能一步关弹窗。`ProjectCreationForm` 是表单的值的类型（`utils.ts` 的默认值没有负责人，`common-attributes.tsx`、`project-create-buttons.tsx` 就地换掉导入，行数不变）。
  - 负责人（`projects/create/attributes.tsx`）：`MemberDropdownBase`，候选是工作区的有效管理员和成员（不含访客，M3 设计 3.19），再选一次已选的就是没有负责人；它在表单的 Tab 顺序中的位置照旧（`tabIndex={getIndex("lead")}`，Task 2 的成员下拉框把它给 Tab 到达的按钮：表单用正的 `tabIndex`，没有它时 Tab 在表单的按钮之后才到负责人）。
  - `@nerve/utils` 的 `projectIdentifierSanitizer`：先转大写，再只留 A–Z、0–9 和 ÇŞĞİÖÜ（原来保留小写，nerve 只收大写；也不再有 `.`、`..`：标识符的检查把它放在地址里，P4a 评审）。
  - 静态检查：`.oxlintrc.json` 的非空断言的范围加上 `project-refusal.ts`、`logo-props.ts`、`use-create-project.ts` 和它们的测试。

**Tests:**
- vitest：`project-refusal.test.ts`：`projectRefusal shows $when`（`it.each`，决定的顺序：名称已被占用（不论问题还指出哪些字段）、标识已被占用、只指向表单的字段的字段错误、也指向表单没有的字段（负责人）的、没有字段的拒绝、没有到达 nerve）。`logo-props.test.ts`：`keeps an $picked.type picked as nerve keeps it`。`use-create-project.test.ts`：`creates the project from the form's fields, no lead until one is picked, says so and gives it`；`sends the lead picked`；`shows $refused under its field, gives nothing and says no success`；`shows nerve's reason for $refused in a toast`；`does nothing on the page and gives nothing when the creation $settles after another tab moved this one`。`web/packages/utils/src/project.test.ts`：`projectIdentifierSanitizer makes $typed $identifier`（转大写、去掉别的字符，土耳其字母照留；`..` 成为空，`w.e.b` 成为 `WEB`）。
- 端到端：`P1 (page): a member creates a project from the projects page, its identifier upper case as typed and its lead an admin, not a guest; a taken identifier and a forbidden name show under their fields; the form waits for nerve; the project shows in the list and the sidebar`（表单的顺序中负责人紧在取消之前：从取消按 Shift+Tab 到它，Enter 打开它的列表：有管理员和成员、没有访客；被占用的标识和不允许的名称各在字段下；创建的请求体经 `sentHeld`：只有表单的字段，扣住时取消和页头的关闭按钮禁用、Escape 不关；之后列表和侧边栏有它）。

- [ ] **Step 1: 拒绝、图标和创建的 hook**

`web/apps/web/core/components/project/logo-props.test.ts`（新文件，26 行）：

````file web/apps/web/core/components/project/logo-props.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import type { LogoProps } from "@nerve/api-client";
import type { TChangeHandlerProps } from "@nerve/propel/emoji-icon-picker";
import { logoPropsOf } from "./logo-props";

// A project's icon as the picker gives it and as nerve keeps it (M3 design 3.19): LogoProps's closed shape.

describe("logoPropsOf", () => {
  it.each<{ picked: TChangeHandlerProps; kept: LogoProps }>([
    {
      picked: { type: "emoji", value: "128640" },
      kept: { in_use: "emoji", emoji: { value: "128640" } },
    },
    {
      picked: { type: "icon", value: { name: "rocket", color: "#5e6ad2" } },
      kept: { in_use: "icon", icon: { name: "rocket", color: "#5e6ad2" } },
    },
  ])("keeps an $picked.type picked as nerve keeps it", ({ picked, kept }) => {
    expect(logoPropsOf(picked)).toEqual(kept);
  });
});
````

`web/apps/web/core/components/project/logo-props.ts`（新文件，16 行）：

````file web/apps/web/core/components/project/logo-props.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { LogoProps } from "@nerve/api-client";
import type { TChangeHandlerProps } from "@nerve/propel/emoji-icon-picker";

/**
 * A project's icon as nerve keeps it (LogoProps, M3 design 3.19) from what the emoji picker picked: an emoji, by its
 * code, or an icon, by its name and colour. The creation's form and the general settings' both set it so.
 */
export function logoPropsOf(picked: TChangeHandlerProps): LogoProps {
  if (picked.type === "emoji") return { in_use: "emoji", emoji: { value: picked.value } };
  return { in_use: "icon", icon: { name: picked.value.name, color: picked.value.color } };
}
````

`web/apps/web/core/components/project/project-refusal.test.ts`（新文件，45 行）：

````file web/apps/web/core/components/project/project-refusal.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { refusal } from "@/lib/fake-refusal";
import { projectRefusal, type ProjectRefusal } from "./project-refusal";

// What a project's form shows of nerve's refusal (M3 design 2 P1, P3): the decision, in its order.

describe("projectRefusal", () => {
  it.each<{ when: string; error: unknown; shows: ProjectRefusal }>([
    {
      when: "a name taken: under the name, whatever fields the problem names",
      error: refusal(409, "project.name_taken", [{ field: "identifier", code: "too_long" }]),
      shows: { kind: "fields", fields: { name: "errors.project_name_taken" } },
    },
    {
      when: "an identifier taken: under the identifier",
      error: refusal(409, "project.identifier_taken"),
      shows: { kind: "fields", fields: { identifier: "errors.project_identifier_taken" } },
    },
    {
      when: "field errors of the form's fields alone: under them",
      error: refusal(422, "validation_failed", [
        { field: "name", code: "not_allowed" },
        { field: "identifier", code: "too_long" },
      ]),
      shows: { kind: "fields", fields: { name: "errors.field.not_allowed", identifier: "errors.field.too_long" } },
    },
    {
      when: "a field error of a field the form shows none under, with one it has: a toast",
      error: refusal(422, "validation_failed", [
        { field: "name", code: "not_allowed" },
        { field: "project_lead_id", code: "not_allowed" },
      ]),
      shows: { kind: "toast" },
    },
    { when: "a refusal of no field: a toast", error: refusal(409, "project.archived"), shows: { kind: "toast" } },
    { when: "nerve not reached: a toast", error: new Error("offline"), shows: { kind: "toast" } },
  ])("shows $when", ({ error, shows }) => {
    expect(projectRefusal(error)).toEqual(shows);
  });
});
````

`web/apps/web/core/components/project/project-refusal.ts`（新文件，63 行）：

````file web/apps/web/core/components/project/project-refusal.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useTranslation } from "@nerve/i18n";
// hooks
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { ApiError } from "@/lib/api-error";
import { PROBLEM_MESSAGES, fieldErrorKeys, needsErrorBanner } from "@/lib/error-messages";

/** The fields of a project's forms (its creation, its general settings) under which nerve's refusals show. */
export type ProjectFormField = "name" | "identifier";

/** What a project's form shows of nerve's refusal: under the fields it names (their messages' i18n keys), or a toast. */
export type ProjectRefusal = { kind: "fields"; fields: Partial<Record<ProjectFormField, string>> } | { kind: "toast" };

/** The fields, in the order their messages show. */
const FIELDS: readonly ProjectFormField[] = ["name", "identifier"];

/** nerve's codes for a name or an identifier another project of the workspace has, each under its field. */
const TAKEN: Readonly<Record<string, ProjectFormField>> = {
  "project.name_taken": "name",
  "project.identifier_taken": "identifier",
};

/**
 * nerve's refusal of a project's creation or change (M3 design 2 P1, P3), as its form shows it: a name or an
 * identifier taken, under its field; field errors under the form's fields when they name only those
 * (needsErrorBanner: a lead nerve does not allow is not one of them); else nerve's reason in a toast.
 */
export function projectRefusal(error: unknown): ProjectRefusal {
  const code = error instanceof ApiError ? (error.problem?.code ?? "") : "";
  const taken = TAKEN[code];
  if (taken) return { kind: "fields", fields: { [taken]: PROBLEM_MESSAGES[code] } };
  if (needsErrorBanner(error, FIELDS)) return { kind: "toast" };
  const { name, identifier } = fieldErrorKeys(error);
  return { kind: "fields", fields: { name, identifier } };
}

/**
 * Shows nerve's refusal of a project's form as projectRefusal decides: each message under its field, by the form's
 * underField (its setError), else nerve's reason in a toast.
 */
export function useProjectRefusal(): (
  error: unknown,
  underField: (field: ProjectFormField, message: string) => void
) => void {
  const { t } = useTranslation();
  const toastRefusal = useRefusalToast();
  return (error, underField) => {
    const refusal = projectRefusal(error);
    if (refusal.kind === "toast") {
      toastRefusal(error);
      return;
    }
    for (const field of FIELDS) {
      const message = refusal.fields[field];
      if (message) underField(field, t(message));
    }
  };
}
````

`web/apps/web/core/components/projects/create/use-create-project.test.ts`（新文件，114 行）：

````file web/apps/web/core/components/projects/create/use-create-project.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf } from "@/store/project/fake-projects";
import { useCreateProject, type ProjectCreationForm } from "./use-create-project";

// How the creation's form creates a project (M3 design 2 P1, 7.1, 7.6): the hook runs as a plain function, with a
// stand-in for the project store's creation, which nerve answers when the test says, and for the form's setError. Its
// session is fake-tab.ts's.

const page = vi.hoisted(() => ({ createProject: vi.fn(), setError: vi.fn() }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => ({ createProject: page.createProject }) }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const form: ProjectCreationForm = {
  name: "Web",
  identifier: "WEB",
  description: "The site",
  network: 2,
  logo_props: { in_use: "emoji", emoji: { value: "128640" } },
  project_lead_id: null,
};
const web = projectOf("WEB", "w-acme", { member_role: 20 });
const create = (values = form) => useCreateProject()("acme", values, page.setError);
/** What nerve's refusal named, field by field, through setError. */
const underFields = () => page.setError.mock.calls.map(([field, { message }]) => [field, message]);

beforeEach(() => {
  signedIn();
  page.createProject.mockReset();
  page.createProject.mockResolvedValue(web);
  page.setError.mockReset();
  toasts.length = 0;
});

describe("useCreateProject", () => {
  it("creates the project from the form's fields, no lead until one is picked, says so and gives it", async () => {
    expect(await create()).toBe(web);
    expect(page.createProject.mock.calls).toEqual([
      [
        "acme",
        {
          name: "Web",
          identifier: "WEB",
          description: "The site",
          network: 2,
          logo_props: { in_use: "emoji", emoji: { value: "128640" } },
        },
      ],
    ]);
    expect(toasts).toEqual([{ type: "success", title: "success", message: "project_created_successfully" }]);
  });

  it("sends the lead picked", async () => {
    await create({ ...form, project_lead_id: "u-ada" });
    expect(page.createProject.mock.calls[0]?.[1]).toMatchObject({ project_lead_id: "u-ada" });
  });

  it.each([
    {
      refused: "a name taken",
      error: refusal(409, "project.name_taken"),
      fields: [["name", "errors.project_name_taken"]],
    },
    {
      refused: "an identifier taken",
      error: refusal(409, "project.identifier_taken"),
      fields: [["identifier", "errors.project_identifier_taken"]],
    },
    {
      refused: "a name of characters nerve does not allow",
      error: refusal(422, "validation_failed", [{ field: "name", code: "not_allowed" }]),
      fields: [["name", "errors.field.not_allowed"]],
    },
  ])("shows $refused under its field, gives nothing and says no success", async ({ error, fields }) => {
    page.createProject.mockRejectedValueOnce(error);
    expect(await create()).toBeUndefined();
    expect([underFields(), toasts]).toEqual([fields, []]);
  });

  it.each([
    {
      refused: "a lead nerve does not allow",
      error: refusal(422, "validation_failed", [{ field: "project_lead_id", code: "not_allowed" }]),
      message: "errors.validation_failed",
    },
    { refused: "a guest's creation", error: refusal(403, "forbidden"), message: "errors.forbidden" },
  ])("shows nerve's reason for $refused in a toast", async ({ error, message }) => {
    page.createProject.mockRejectedValueOnce(error);
    expect(await create()).toBeUndefined();
    expect([underFields(), toasts]).toEqual([[], [{ type: "error", title: "toast.error", message }]]);
  });

  it.each(lateSettlings)(
    "does nothing on the page and gives nothing when the creation $settles after another tab moved this one",
    async ({ settle }) => {
      const answer = heldChange<undefined>();
      page.createProject.mockReturnValueOnce(answer.sent);
      const created = create();
      switchAccount();
      settle(answer);
      expect(await created).toBeUndefined();
      expect([underFields(), toasts]).toEqual([[], []]);
    }
  );
});
````

`web/apps/web/core/components/projects/create/use-create-project.ts`（新文件，60 行）：

````file web/apps/web/core/components/projects/create/use-create-project.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { UseFormSetError } from "react-hook-form";
import type { Project, ProjectCreate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// components
import { useProjectRefusal } from "@/components/project/project-refusal";
// hooks
import { useProject } from "@/hooks/store/use-project";
// lib
import { followInSession } from "@/lib/in-session";

/** A creation form's values: the fields of ProjectCreate the form asks for, no lead until one is picked. */
export type ProjectCreationForm = Pick<
  ProjectCreate,
  "name" | "identifier" | "description" | "network" | "logo_props"
> & {
  project_lead_id: string | null;
};

/**
 * Creates a project of the workspace from a creation form's values (M3 design 2 P1, 7.6): the fields of ProjectCreate
 * the form has, a lead only once picked; shows nerve's refusal under the fields it names (the form's setError), else
 * in a toast (useProjectRefusal); says so once created. Gives the project created, or undefined when it was not, or
 * when the tab moved to another account meanwhile: the page is that account's then, and goes no further (M3 design
 * 7.1).
 */
export function useCreateProject(): (
  workspaceSlug: string,
  form: ProjectCreationForm,
  setError: UseFormSetError<ProjectCreationForm>
) => Promise<Project | undefined> {
  const { createProject } = useProject();
  const { t } = useTranslation();
  const showRefusal = useProjectRefusal();

  return async (workspaceSlug, form, setError) => {
    const data: ProjectCreate = {
      name: form.name,
      identifier: form.identifier,
      description: form.description,
      network: form.network,
      logo_props: form.logo_props,
      project_lead_id: form.project_lead_id ?? undefined,
    };
    let created: Project | undefined;
    await followInSession(() => createProject(workspaceSlug, data), {
      done: (project) => {
        created = project;
        setToast({ type: TOAST_TYPE.SUCCESS, title: t("success"), message: t("project_created_successfully") });
      },
      failed: (error) => showRefusal(error, (field, message) => setError(field, { type: "server", message })),
    });
    return created;
  };
}
````

`web/packages/propel/src/emoji-icon-picker/helper.tsx`（修改，1 处）：

````old web/packages/propel/src/emoji-icon-picker/helper.tsx
type TChangeHandlerProps =
````
````new web/packages/propel/src/emoji-icon-picker/helper.tsx
/** What the picker picked: an emoji, by its code, or an icon, by its name and colour. */
export type TChangeHandlerProps =
````

`web/packages/utils/src/project.test.ts`（新文件，23 行）：

````file web/packages/utils/src/project.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { projectIdentifierSanitizer } from "./project";

// A project's identifier as typed (M3 design 3.19, 7.6): upper case, of A-Z, 0-9 and ÇŞĞİÖÜ alone.

describe("projectIdentifierSanitizer", () => {
  it.each([
    { typed: "web", identifier: "WEB" },
    { typed: "We b-2", identifier: "WEB2" },
    { typed: "ç ş ğ i ö ü", identifier: "ÇŞĞIÖÜ" },
    { typed: "İzmir", identifier: "İZMIR" },
    // nothing of a path segment's "." or "..", which a check of the identifier would send in its address
    { typed: "..", identifier: "" },
    { typed: "w.e.b", identifier: "WEB" },
  ])("makes $typed $identifier", ({ typed, identifier }) => {
    expect(projectIdentifierSanitizer(typed)).toBe(identifier);
  });
});
````

`web/packages/utils/src/project.ts`（修改，2 处）：

````old web/packages/utils/src/project.ts
import { satisfiesDateFilter } from "./filter";

````
````new web/packages/utils/src/project.ts
import { satisfiesDateFilter } from "./filter";

/**
 * The identifier text typed makes (M3 design 3.19, 7.6): in upper case, of the characters an identifier may have
 * (A-Z, 0-9 and ÇŞĞİÖÜ), the others dropped as they are typed. A check of it never sends "." or "..", which a URL's
 * parser would take for path segments (P4a review).
 */
````
````old web/packages/utils/src/project.ts
  identifier.replace(/[^ÇŞĞIİÖÜA-Za-z0-9]/g, "");
````
````new web/packages/utils/src/project.ts
  identifier.toUpperCase().replace(/[^ÇŞĞİÖÜA-Z0-9]/g, "");
````

- [ ] **Step 2: 创建的表单**

`web/apps/web/core/components/project/create-project-modal.tsx`（修改，2 处）：

````old web/apps/web/core/components/project/create-project-modal.tsx
  };

````
````new web/apps/web/core/components/project/create-project-modal.tsx
  };

  // the creation's form closes itself on Escape, but while it creates; the next step closes on it
````
````old web/apps/web/core/components/project/create-project-modal.tsx
    if (isOpen) onClose();
````
````new web/apps/web/core/components/project/create-project-modal.tsx
    if (isOpen && currentStep === EProjectCreationSteps.FEATURE_SELECTION) onClose();
````

`web/apps/web/core/components/project/create/common-attributes.tsx`（修改，3 处）：

````old web/apps/web/core/components/project/create/common-attributes.tsx
import type { ProjectCreate } from "@nerve/api-client";
````
````new web/apps/web/core/components/project/create/common-attributes.tsx
import type { ProjectCreationForm } from "@/components/projects/create/use-create-project";
````
````old web/apps/web/core/components/project/create/common-attributes.tsx
  setValue: UseFormSetValue<ProjectCreate>;
````
````new web/apps/web/core/components/project/create/common-attributes.tsx
  setValue: UseFormSetValue<ProjectCreationForm>;
````
````old web/apps/web/core/components/project/create/common-attributes.tsx
  } = useFormContext<ProjectCreate>();
````
````new web/apps/web/core/components/project/create/common-attributes.tsx
  } = useFormContext<ProjectCreationForm>();
````

`web/apps/web/core/components/project/create/header.tsx`（修改，5 处）：

````old web/apps/web/core/components/project/create/header.tsx
import { CloseOutline } from "@makeplane/propel/icons";
// nerve types
import type { ProjectCreate } from "@nerve/api-client";
````
````new web/apps/web/core/components/project/create/header.tsx
import { CloseOutline } from "@makeplane/propel/icons";
````
````old web/apps/web/core/components/project/create/header.tsx
import { CoverImage } from "@/components/common/cover-image";
````
````new web/apps/web/core/components/project/create/header.tsx
import { CoverImage } from "@/components/common/cover-image";
import { logoPropsOf } from "@/components/project/logo-props";
import type { ProjectCreationForm } from "@/components/projects/create/use-create-project";
````
````old web/apps/web/core/components/project/create/header.tsx
  const { control, setValue } = useFormContext<ProjectCreate>();
````
````new web/apps/web/core/components/project/create/header.tsx
  const {
    control,
    setValue,
    formState: { isSubmitting },
  } = useFormContext<ProjectCreationForm>();
````
````old web/apps/web/core/components/project/create/header.tsx
        <button type="button" onClick={handleClose} tabIndex={getIndex("close")}>
````
````new web/apps/web/core/components/project/create/header.tsx
        <button
          type="button"
          onClick={handleClose}
          disabled={isSubmitting}
          aria-label={t("close")}
          tabIndex={getIndex("close")}
        >
````
````old web/apps/web/core/components/project/create/header.tsx
              onChange={(val: any) => {
                let logoValue = {};

                if (val?.type === "emoji")
                  logoValue = {
                    value: val.value,
                  };
                else if (val?.type === "icon") logoValue = val.value;

                const newLogoProps = {
                  in_use: val?.type,
                  [val?.type]: logoValue,
                };
                setValue("logo_props", newLogoProps, {
                  shouldDirty: true,
                });
                onChange(newLogoProps);
````
````new web/apps/web/core/components/project/create/header.tsx
              onChange={(picked) => {
                const logo = logoPropsOf(picked);
                setValue("logo_props", logo, { shouldDirty: true });
                onChange(logo);
````

`web/apps/web/core/components/project/create/project-create-buttons.tsx`（修改，3 处）：

````old web/apps/web/core/components/project/create/project-create-buttons.tsx
import type { ProjectCreate } from "@nerve/api-client";
````
````new web/apps/web/core/components/project/create/project-create-buttons.tsx
import type { ProjectCreationForm } from "@/components/projects/create/use-create-project";
````
````old web/apps/web/core/components/project/create/project-create-buttons.tsx
  } = useFormContext<ProjectCreate>();
````
````new web/apps/web/core/components/project/create/project-create-buttons.tsx
  } = useFormContext<ProjectCreationForm>();
````
````old web/apps/web/core/components/project/create/project-create-buttons.tsx
      <Button variant="secondary" size="lg" onClick={handleClose} tabIndex={getIndex("cancel")}>
````
````new web/apps/web/core/components/project/create/project-create-buttons.tsx
      <Button variant="secondary" size="lg" onClick={handleClose} disabled={isSubmitting} tabIndex={getIndex("cancel")}>
````

`web/apps/web/core/components/projects/create/attributes.tsx`（修改，7 处）：

````old web/apps/web/core/components/projects/create/attributes.tsx
 */

````
````new web/apps/web/core/components/projects/create/attributes.tsx
 */

import { observer } from "mobx-react";
````
````old web/apps/web/core/components/projects/create/attributes.tsx
import { NETWORK_CHOICES, ETabIndices } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import type { ProjectCreate } from "@nerve/api-client";
````
````new web/apps/web/core/components/projects/create/attributes.tsx
import { EUserPermissions, NETWORK_CHOICES, ETabIndices } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/projects/create/attributes.tsx
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { ProjectNetworkIcon } from "@/components/project/project-network-icon";
````
````new web/apps/web/core/components/projects/create/attributes.tsx
import { MemberDropdownBase } from "@/components/dropdowns/member/base";
import { ProjectNetworkIcon } from "@/components/project/project-network-icon";
// hooks
import { useMember } from "@/hooks/store/use-member";
// local imports
import type { ProjectCreationForm } from "./use-create-project";
````
````old web/apps/web/core/components/projects/create/attributes.tsx
function ProjectAttributes(props: Props) {
````
````new web/apps/web/core/components/projects/create/attributes.tsx
const ProjectAttributes = observer(function ProjectAttributes(props: Props) {
````
````old web/apps/web/core/components/projects/create/attributes.tsx
  const { control } = useFormContext<ProjectCreate>();
  const { getIndex } = getTabIndex(ETabIndices.PROJECT_CREATE, isMobile);
````
````new web/apps/web/core/components/projects/create/attributes.tsx
  const { control } = useFormContext<ProjectCreationForm>();
  const { getIndex } = getTabIndex(ETabIndices.PROJECT_CREATE, isMobile);
  const {
    getUserDetails,
    workspace: { workspaceMemberIds, getWorkspaceMemberDetails },
  } = useMember();
  // the lead is one of the workspace's admins and members (M3 design 3.19): no guest, no one whose membership ended
  const leadCandidates = (workspaceMemberIds ?? []).filter((id) => {
    const membership = getWorkspaceMemberDetails(id);
    return membership !== null && membership.is_active && membership.role >= EUserPermissions.MEMBER;
  });
````
````old web/apps/web/core/components/projects/create/attributes.tsx
          <div className="h-7 flex-shrink-0" tabIndex={getIndex("lead")}>
            <MemberDropdown
              value={value ?? null}
              onChange={(lead) => onChange(lead === value ? undefined : lead)}
````
````new web/apps/web/core/components/projects/create/attributes.tsx
          <div className="h-7 flex-shrink-0">
            <MemberDropdownBase
              getUserDetails={getUserDetails}
              memberIds={leadCandidates}
              value={value}
              // the lead picked again is no lead
              onChange={(lead) => onChange(lead === value ? null : lead)}
````
````old web/apps/web/core/components/projects/create/attributes.tsx
  );
}
````
````new web/apps/web/core/components/projects/create/attributes.tsx
  );
});
````

`web/apps/web/core/components/projects/create/root.tsx`（整个文件，86 行）：

````whole web/apps/web/core/components/projects/create/root.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { FormProvider, useForm } from "react-hook-form";
// components
import ProjectCommonAttributes from "@/components/project/create/common-attributes";
import ProjectCreateHeader from "@/components/project/create/header";
import ProjectCreateButtons from "@/components/project/create/project-create-buttons";
// hooks
import useKeypress from "@/hooks/use-keypress";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { ProjectAttributes } from "./attributes";
import { useCreateProject, type ProjectCreationForm } from "./use-create-project";
import { getProjectFormValues } from "./utils";

export type TCreateProjectFormProps = {
  workspaceSlug: string;
  onClose: () => void;
  handleNextStep: (projectId: string) => void;
};

export const CreateProjectForm = observer(function CreateProjectForm(props: TCreateProjectFormProps) {
  const { workspaceSlug, onClose, handleNextStep } = props;
  // hooks
  const create = useCreateProject();
  // states
  const [shouldAutoSyncIdentifier, setShouldAutoSyncIdentifier] = useState(true);
  // form info
  const methods = useForm<ProjectCreationForm>({
    defaultValues: getProjectFormValues(),
    reValidateMode: "onChange",
  });
  const {
    handleSubmit,
    reset,
    setError,
    setValue,
    formState: { isSubmitting },
  } = methods;
  const { isMobile } = usePlatformOS();

  // The form is busy until nerve has answered and the page has followed: created, the modal's next step.
  const onSubmit = async (values: ProjectCreationForm) => {
    const project = await create(workspaceSlug, values, setError);
    if (project) handleNextStep(project.id);
  };

  const handleClose = () => {
    onClose();
    setShouldAutoSyncIdentifier(true);
    setTimeout(() => {
      reset();
    }, 300);
  };

  // While the creation is out the form cannot be closed (Escape, its close button, Cancel): nerve's answer is followed
  // by the form that sent it.
  useKeypress("Escape", () => {
    if (!isSubmitting) handleClose();
  });

  return (
    <FormProvider {...methods}>
      <ProjectCreateHeader handleClose={handleClose} isMobile={isMobile} />

      <form onSubmit={handleSubmit(onSubmit)} className="px-3">
        <div className="mt-9 space-y-6 pb-5">
          <ProjectCommonAttributes
            setValue={setValue}
            isMobile={isMobile}
            shouldAutoSyncIdentifier={shouldAutoSyncIdentifier}
            setShouldAutoSyncIdentifier={setShouldAutoSyncIdentifier}
          />
          <ProjectAttributes isMobile={isMobile} />
        </div>
        <ProjectCreateButtons handleClose={handleClose} />
      </form>
    </FormProvider>
  );
});
````

`web/apps/web/core/components/projects/create/utils.ts`（修改，3 处）：

````old web/apps/web/core/components/projects/create/utils.ts
import type { ProjectCreate } from "@nerve/api-client";
````
````new web/apps/web/core/components/projects/create/utils.ts
// local imports
import type { ProjectCreationForm } from "./use-create-project";
````
````old web/apps/web/core/components/projects/create/utils.ts
export const getProjectFormValues = (): ProjectCreate => ({
````
````new web/apps/web/core/components/projects/create/utils.ts
/** A new project's form as it opens: public, no lead, a random emoji its icon (M3 design 3.19). */
export const getProjectFormValues = (): ProjectCreationForm => ({
````
````old web/apps/web/core/components/projects/create/utils.ts
  network: 2,
````
````new web/apps/web/core/components/projects/create/utils.ts
  network: 2,
  project_lead_id: null,
````

- [ ] **Step 3: 静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/hooks/use-copy-link.test.ts"
````
````new .oxlintrc.json
        "web/apps/web/core/hooks/use-copy-link.test.ts",
        "web/apps/web/core/components/project/project-refusal.ts",
        "web/apps/web/core/components/project/project-refusal.test.ts",
        "web/apps/web/core/components/project/logo-props.ts",
        "web/apps/web/core/components/project/logo-props.test.ts",
        "web/apps/web/core/components/projects/create/use-create-project.ts",
        "web/apps/web/core/components/projects/create/use-create-project.test.ts"
````

- [ ] **Step 4: 端到端**

`e2e/stories/project/p1-create-project.spec.ts`（修改，3 处）：

````old e2e/stories/project/p1-create-project.spec.ts
import {
  expectListBesideButton,
  keydownsReachingDocument,
  registerOnboarded,
  transitionsEnded,
} from "../../fixtures/settings-pages";
````
````new e2e/stories/project/p1-create-project.spec.ts
import {
  closedByEscape,
  expectListBesideButton,
  keydownsReachingDocument,
  registerOnboarded,
  sentHeld,
  sentTo,
  transitionsEnded,
} from "../../fixtures/settings-pages";
````
````old e2e/stories/project/p1-create-project.spec.ts
  return data ? { status: response.status, project: data } : { status: response.status, code: error?.code };
````
````new e2e/stories/project/p1-create-project.spec.ts
  return data ? { status: response.status, project: data } : { status: response.status, code: error?.code };
}

/** Gives the account of token the display name name, by which the pages show it. */
async function named(api: Api, token: string, name: string): Promise<void> {
  const { response } = await api.PATCH("/api/v0/me", { body: { display_name: name }, headers: bearer(token) });
  expect(response.status, `name ${name}`).toBe(200);
````
````old e2e/stories/project/p1-create-project.spec.ts
});

test("P1 (page): the lead's list opens beside its button with its search focused; Escape closes it", async ({
````
````new e2e/stories/project/p1-create-project.spec.ts
});

test("P1 (page): a member creates a project from the projects page, its identifier upper case as typed and its lead an admin, not a guest; a taken identifier and a forbidden name show under their fields; the form waits for nerve; the project shows in the list and the sidebar", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const memberEmail = emailFor(testInfo, "member");
  const member = await registerOnboarded(api, memberEmail);
  const guestEmail = emailFor(testInfo, "guest");
  const guest = await registerOnboarded(api, guestEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug, timezone: "Asia/Shanghai" });
  await inviteAndAccept(api, admin.access_token, slug, { email: memberEmail, token: member.access_token }, 15);
  await inviteAndAccept(api, admin.access_token, slug, { email: guestEmail, token: guest.access_token }, 5);
  await named(api, admin.access_token, "ada");
  await named(api, guest.access_token, "gus");
  // WEB is taken by the time the member sends it
  await createProject(api, admin.access_token, slug, { name: "Site", identifier: "WEB" });
  const adminId = await accountId(api, admin.access_token);

  const page = await signedInPage(member);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/projects`);
  await page.getByRole("button", { name: "Add Project", exact: true }).click();
  await transitionsEnded(page.getByRole("dialog"));
  // The name gives the identifier, as typed; the identifier typed is upper case, of the characters it may have.
  await page.locator("#name").fill("Web-App");
  await expect(page.locator("#identifier")).toHaveValue("WEBAPP");
  await page.locator("#identifier").fill("we.b");
  await expect(page.locator("#identifier")).toHaveValue("WEB");
  await page.locator("#description").fill("The site");
  // The lead: the form's order puts it right before Cancel, which Shift+Tab leaves for it; the workspace's admin or a
  // member, not its guest.
  await page.getByRole("button", { name: "Cancel" }).focus();
  await page.keyboard.press("Shift+Tab");
  await expect(page.getByRole("button", { name: "Lead", exact: true }).last()).toBeFocused();
  await page.keyboard.press("Enter");
  // each after its avatar's initial
  await expect(page.getByRole("option")).toHaveText([/You$/, /ada$/]);
  await page.getByRole("option", { name: "ada" }).click();
  // The icon: an emoji from the picker.
  await page.getByRole("button", { name: "Project icon" }).click();
  await page.getByRole("searchbox").fill("rocket");
  await page.getByRole("gridcell", { name: "Rocket" }).click();

  // Refused: the name nerve does not allow and the identifier taken, each under its field.
  const create = page.getByRole("button", { name: "Create project" });
  const refusedName = await sentTo(page, "POST", `/api/v0/workspaces/${slug}/projects`, () => create.click());
  expect(refusedName.answer.status()).toBe(422);
  await expect(page.getByText("Not allowed")).toBeVisible();
  await page.locator("#name").fill("Web App");
  await page.locator("#identifier").fill("web");
  const refusedIdentifier = await sentTo(page, "POST", `/api/v0/workspaces/${slug}/projects`, () => create.click());
  expect(refusedIdentifier.answer.status()).toBe(409);
  await expect(page.getByText("A project of this workspace already has this identifier.")).toBeVisible();

  // Created: the form's fields alone, sent once; the form cannot be closed until nerve answers.
  await page.locator("#identifier").fill("webapp");
  const { body, release } = await sentHeld(page, "POST", `/api/v0/workspaces/${slug}/projects`, () => create.click());
  expect(body).toEqual({
    name: "Web App",
    identifier: "WEBAPP",
    description: "The site",
    network: 2,
    logo_props: { in_use: "emoji", emoji: { value: "128640" } },
    project_lead_id: adminId,
  });
  // neither by Cancel, by the header's Close nor by Escape
  await expect(page.getByRole("button", { name: "Cancel" })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Close", exact: true })).toBeDisabled();
  expect(await closedByEscape(page)).toBe(false);
  expect((await release()).status()).toBe(201);
  await expect(page.getByText("Project created successfully")).toBeVisible();
  await page.getByRole("button", { name: "Close", exact: true }).click();
  await expect(page.getByRole("link", { name: /Web App/ })).toBeVisible();
  await expect(page.getByRole("complementary", { name: "Main sidebar" }).getByText("Web App")).toBeVisible();

  const created = {
    name: "Web App",
    identifier: "WEBAPP",
    description: "The site",
    network: 2,
    timezone: "Asia/Shanghai",
    logo_props: { in_use: "emoji", emoji: { value: "128640" } },
  };
  await expectProjectCreated(db, slug, created, memberEmail, adminEmail, [
    { email: memberEmail, sort_order: 65535 },
    { email: adminEmail, sort_order: 55535 },
  ]);

  // the two refusals, the browser's reports of them
  expect([watch.cspViolations, watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [],
    [`422 POST /api/v0/workspaces/${slug}/projects`, `409 POST /api/v0/workspaces/${slug}/projects`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING],
    errors: [
      "Failed to load resource: the server responded with a status of 422 (Unprocessable Entity)",
      "Failed to load resource: the server responded with a status of 409 (Conflict)",
    ],
  });
});

test("P1 (page): the lead's list opens beside its button with its search focused; Escape closes it", async ({
````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 67 条规则、3 个例外，没有命中；web 的 oxlint 350 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过（`@nerve/utils` 的 vitest 在其中）。

Run: `make e2e`
Expected: 99 个全部通过。

- [ ] **Step 6: 提交**

```bash
git add .oxlintrc.json e2e/stories/project/p1-create-project.spec.ts web/apps/web/core/components/project/create-project-modal.tsx web/apps/web/core/components/project/create/common-attributes.tsx web/apps/web/core/components/project/create/header.tsx web/apps/web/core/components/project/create/project-create-buttons.tsx web/apps/web/core/components/project/logo-props.test.ts web/apps/web/core/components/project/logo-props.ts web/apps/web/core/components/project/project-refusal.test.ts web/apps/web/core/components/project/project-refusal.ts web/apps/web/core/components/projects/create/attributes.tsx web/apps/web/core/components/projects/create/root.tsx web/apps/web/core/components/projects/create/use-create-project.test.ts web/apps/web/core/components/projects/create/use-create-project.ts web/apps/web/core/components/projects/create/utils.ts web/packages/propel/src/emoji-icon-picker/helper.tsx web/packages/utils/src/project.test.ts web/packages/utils/src/project.ts
```
```bash
git commit -m "feat(M3/P10): creating a project sends its form's fields, shows nerve's refusals under them and waits for nerve

useCreateProject builds ProjectCreate from the form's fields, a lead only
once picked, follows the creation only in the session it was sent in, and
shows a taken name or identifier, or a field nerve names, under its field,
anything else as nerve's reason. The form cannot be closed while the
creation is out; the lead is one of the workspace's admins and members,
at its place in the form's Tab order;
the identifier is upper case as typed, with nothing a URL would take for
a path segment; the icon is nerve's closed LogoProps. P1's page version.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p10.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T4.1` | 没有负责人时发出 `null`，而不是不发 | `tsc`、`use-create-project.test.ts` | 静态；vitest |
| `T4.2` | 名称已被占用在提示里说，不在名称下方说（创建和 general 设置） | `project-refusal.test.ts`、`use-create-project.test.ts` | vitest |
| `T4.3` | 创建项目不论会话都跟进 | `use-create-project.test.ts` | vitest |
| `T4.4` | 创建在途时 Escape 关掉表单 | 故事 P1 | 端到端 |
| `T4.5` | 创建在途时表单的取消按钮仍可用 | 故事 P1 | 端到端 |
| `T4.6` | 负责人的候选包括工作区的访客 | 故事 P1 | 端到端 |
| `T4.7` | 标识符保留输入的大小写 | `project.test.ts`、故事 P1 | vitest；端到端 |
| `T4.8` | 图标丢掉颜色 | `logo-props.test.ts` | vitest |
| `T4.9` | 在创建项目的 hook 里写一个非空断言 | oxlint（`check:lint`） | 静态 |
| `T4.10` | 负责人不在创建表单的 Tab 顺序里：Tab 在表单的按钮之后才到它 | 故事 P1 | 端到端 |
| `T4.11` | 创建在途时页头的关闭按钮仍可用 | 故事 P1 | 端到端 |

---

### Task 5: 项目设置的 general；P3 的页面版本（一）

**Files:**
- Create: `web/apps/web/core/components/project/use-update-project-details.test.ts`、`web/apps/web/core/components/project/use-update-project-details.ts`
- Modify: `.oxlintrc.json`、`e2e/fixtures/settings-pages.ts`、`e2e/stories/project/p3-project-settings.spec.ts`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx`、`web/apps/web/core/components/project/form.tsx`、`web/packages/i18n/src/locales/en/common.json`、`web/packages/i18n/src/locales/zh-CN/common.json`

**Interfaces:**
- Produces（spec 2.5；M3 设计 2 的 P3，3.19，7.1、7.6；P8b spec 第 5 节 P10 一行和 P6；M2 收尾交接第 11 节的项目时区）：
  - `core/components/project/use-update-project-details.ts`：`ProjectDetails`（`ProjectUpdate` 中 general 页编辑的 `name`、`identifier`、`description`、`network`、`logo_props`、`timezone`）、`projectDetailsOf(project)`；`useUpdateProjectDetails(): (project, workspaceSlug, details, setError) => Promise<void>`：经 `followInSession` 发出一个修改：标识改了时先问 nerve（`checkProjectIdentifier`），被占用就交回 `identifier_taken`，在标识下说明，什么都不发（P8b 的 P6：原来的检查在 `try` 之外，`!available` 的一支没有测试）；否则发这些字段，成功提示；被拒绝时经 `useProjectRefusal`（检查本身被拒绝也是），不论哪一步都只在发出它的会话里跟进。
  - `ProjectDetailsForm`（`project/form.tsx`，整个文件，411 → 292 行）：默认值是 `projectDetailsOf(project)`；提交经 `useUpdateProjectDetails`，按钮忙到跟进做完（`isSubmitting`）；标识照 Task 4 的 `projectIdentifierSanitizer`；图标经 `logoPropsOf`，选择器的按钮有名字；时区是 `GET /api/v0/timezones` 的列表（M2 交接第 11 节）；成员看到 nerve 所持的值，改不了（没有归档、删除）；原来按 Plane 的错误形状读拒绝（`err.name`、`err.identifier` 中的码，读不到 nerve 的 `problem.code`，只剩"something went wrong"）、`console.error`、提交之后 300 毫秒才结束的加载和 `useEffect` 的 `reset`（和它的 `react-hooks/exhaustive-deps` 抑制）都删除：页面按项目挂载一个表单。
  - general 页（`settings/projects/[projectId]/page.tsx`）：地址的项目经 `getProjectById(projectId)`（不读 `currentProjectDetails`），表单 `key={project.id}`：每个项目一个表单（spec 第 3 节第 7 条）。应用中还没有不离开路由就从一个项目的设置页到另一个的路，P3 的页面版本用路由自己的移动代替以后的切换。
  - i18n（en、zh-CN 的 `common.json`）：没有读者的四条删除（`project_name_already_taken`、`project_name_cannot_contain_special_characters`、`project_identifier_already_taken`、`common.identifier_already_exists`；拒绝的文案是 `errors` 的）。
  - 静态检查：`.oxlintrc.json` 的非空断言的范围加上 `use-update-project-details.ts` 和它的测试。
- e2e 的共用部分：`e2e/fixtures/settings-pages.ts` 的 `moveWithinApp(page, path)`：像应用的链接那样不重新载入就到 `path`（`history.pushState`，再发 `popstate`，React Router 跟着它），故事用它代替应用还没有的链接。

**Tests:**
- vitest：`use-update-project-details.test.ts`：`sends the fields the page edits, asks nothing of an identifier unchanged, and says so`；`asks nerve of an identifier changed, then sends it`；`says under the identifier that another project has it, and sends nothing`；`shows $refused under its field and says no success`；`shows nerve's reason for $refused in a toast`；`shows nerve's reason in a toast when it refuses the identifier's check, sends nothing and settles`；`does nothing on the page when the change $settles after another tab moved this one`；`says nothing of an identifier nerve answers taken after another tab moved this one`。
- 端到端：`P3 (page): the project's admin changes its name, identifier, description, visibility, time zone and icon on its general page, which hold after a reload; an identifier another project has is said under it and nothing is sent; the page waits for nerve; its member sees them and can change nothing`（输入 `o.ps`，标识显示 `OPS`，被占用，没有 `PATCH`；修改的请求体经 `sentHeld`，扣住时按钮不可用；重新载入之后都在；成员的页面只读；没有发往别处的请求，`requestsElsewhere`）。`P3 (page): another project's general page, reached without leaving the route, shows that project's values`（从 Web 的 general 页经 `moveWithinApp` 到 Ops 的，显示 Ops；再回到 Web 的：它的读取已在会话的缓存里，包装层立即显示页面，路由一直挂载着，显示的是 WEB 的标识）。

- [ ] **Step 1: general 页的修改**

`web/apps/web/core/components/project/use-update-project-details.test.ts`（新文件，143 行）：

````file web/apps/web/core/components/project/use-update-project-details.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf } from "@/store/project/fake-projects";
import { projectDetailsOf, useUpdateProjectDetails, type ProjectDetails } from "./use-update-project-details";

// How a project's general page changes it (M3 design 2 P3, 7.1, 7.6): the hook runs as a plain function, with
// stand-ins for the project store's change and identifier check, which nerve answers when the test says, and for the
// form's setError. Its session is fake-tab.ts's.

const page = vi.hoisted(() => ({ updateProject: vi.fn(), checkProjectIdentifier: vi.fn(), setError: vi.fn() }));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({ updateProject: page.updateProject, checkProjectIdentifier: page.checkProjectIdentifier }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const web = projectOf("WEB", "w-acme", { member_role: 20, description: "The site", timezone: "Asia/Shanghai" });
const renamed: ProjectDetails = { ...projectDetailsOf(web), name: "Site", network: 0 };
const update = (details = renamed) => useUpdateProjectDetails()(web, "acme", details, page.setError);
/** What the page said under the form's fields, field by field, through setError. */
const underFields = () => page.setError.mock.calls.map(([field, { message }]) => [field, message]);

beforeEach(() => {
  signedIn();
  page.updateProject.mockReset();
  page.updateProject.mockResolvedValue(web);
  page.checkProjectIdentifier.mockReset();
  page.checkProjectIdentifier.mockResolvedValue({ available: true });
  page.setError.mockReset();
  toasts.length = 0;
});

describe("useUpdateProjectDetails", () => {
  it("sends the fields the page edits, asks nothing of an identifier unchanged, and says so", async () => {
    await update();
    expect(page.updateProject.mock.calls).toEqual([
      [
        "p-web",
        {
          name: "Site",
          identifier: "WEB",
          description: "The site",
          network: 0,
          logo_props: {},
          timezone: "Asia/Shanghai",
        },
      ],
    ]);
    expect(page.checkProjectIdentifier).not.toHaveBeenCalled();
    expect(toasts).toEqual([
      { type: "success", title: "toast.success", message: "project_settings.general.toast.success" },
    ]);
  });

  it("asks nerve of an identifier changed, then sends it", async () => {
    await update({ ...renamed, identifier: "SITE" });
    expect(page.checkProjectIdentifier.mock.calls).toEqual([["acme", "SITE"]]);
    expect(page.updateProject.mock.calls[0]?.[1]).toMatchObject({ identifier: "SITE" });
  });

  it("says under the identifier that another project has it, and sends nothing", async () => {
    page.checkProjectIdentifier.mockResolvedValueOnce({ available: false });
    await update({ ...renamed, identifier: "OPS" });
    expect(page.updateProject).not.toHaveBeenCalled();
    expect([underFields(), toasts]).toEqual([[["identifier", "errors.project_identifier_taken"]], []]);
  });

  it.each([
    {
      refused: "a name taken",
      error: refusal(409, "project.name_taken"),
      fields: [["name", "errors.project_name_taken"]],
    },
    {
      refused: "an identifier taken meanwhile",
      error: refusal(409, "project.identifier_taken"),
      fields: [["identifier", "errors.project_identifier_taken"]],
    },
    {
      refused: "a name of characters nerve does not allow",
      error: refusal(422, "validation_failed", [{ field: "name", code: "not_allowed" }]),
      fields: [["name", "errors.field.not_allowed"]],
    },
  ])("shows $refused under its field and says no success", async ({ error, fields }) => {
    page.updateProject.mockRejectedValueOnce(error);
    await update();
    expect([underFields(), toasts]).toEqual([fields, []]);
  });

  it.each([
    { refused: "a change by one no longer its admin", error: refusal(403, "forbidden"), message: "errors.forbidden" },
    {
      refused: "a change of a project archived meanwhile",
      error: refusal(409, "project.archived"),
      message: "errors.project_archived",
    },
  ])("shows nerve's reason for $refused in a toast", async ({ error, message }) => {
    page.updateProject.mockRejectedValueOnce(error);
    await update();
    expect([underFields(), toasts]).toEqual([[], [{ type: "error", title: "toast.error", message }]]);
  });

  it("shows nerve's reason in a toast when it refuses the identifier's check, sends nothing and settles", async () => {
    page.checkProjectIdentifier.mockRejectedValueOnce(refusal(403, "forbidden"));
    await update({ ...renamed, identifier: "SITE" });
    expect(page.updateProject).not.toHaveBeenCalled();
    expect([underFields(), toasts]).toEqual([
      [],
      [{ type: "error", title: "toast.error", message: "errors.forbidden" }],
    ]);
  });

  it.each(lateSettlings)(
    "does nothing on the page when the change $settles after another tab moved this one",
    async ({ settle }) => {
      const answer = heldChange<undefined>();
      page.updateProject.mockReturnValueOnce(answer.sent);
      const updated = update();
      switchAccount();
      settle(answer);
      await updated;
      expect([underFields(), toasts]).toEqual([[], []]);
    }
  );

  it("says nothing of an identifier nerve answers taken after another tab moved this one", async () => {
    const answer = heldChange<{ available: boolean }>();
    page.checkProjectIdentifier.mockReturnValueOnce(answer.sent);
    const updated = update({ ...renamed, identifier: "OPS" });
    switchAccount();
    answer.answer({ available: false });
    await updated;
    expect([page.updateProject.mock.calls, underFields(), toasts]).toEqual([[], [], []]);
  });
});
````

`web/apps/web/core/components/project/use-update-project-details.ts`（新文件，84 行）：

````file web/apps/web/core/components/project/use-update-project-details.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { UseFormSetError } from "react-hook-form";
import type { Project, ProjectUpdate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useProject } from "@/hooks/store/use-project";
// lib
import { followInSession } from "@/lib/in-session";
// local imports
import { useProjectRefusal, type ProjectFormField } from "./project-refusal";

/** A project's general settings: the fields of ProjectUpdate its general page edits (M3 design 7.6). */
export type ProjectDetails = Required<
  Pick<ProjectUpdate, "name" | "identifier" | "description" | "network" | "logo_props" | "timezone">
>;

/** A project's general settings as nerve last answered them. */
export const projectDetailsOf = (project: Project): ProjectDetails => ({
  name: project.name,
  identifier: project.identifier,
  description: project.description,
  network: project.network,
  logo_props: project.logo_props,
  timezone: project.timezone,
});

/**
 * Changes a project's general settings to the values its page holds (M3 design 2 P3, 7.6). An identifier changed is
 * asked of nerve first (checkProjectIdentifier): one another project of the workspace has is said under it, and
 * nothing is sent. Else the page's fields are sent; nerve's refusal shows under the fields it names (the form's
 * setError), else in a toast (useProjectRefusal), and a change made is said so. Settles once the page has followed
 * the change, which it does only in the session the change was sent in (M3 design 7.1).
 */
export function useUpdateProjectDetails(): (
  project: Project,
  workspaceSlug: string,
  details: ProjectDetails,
  setError: UseFormSetError<ProjectDetails>
) => Promise<void> {
  const { updateProject, checkProjectIdentifier } = useProject();
  const { t } = useTranslation();
  const showRefusal = useProjectRefusal();

  return (project, workspaceSlug, details, setError) => {
    const underField = (field: ProjectFormField, message: string) => setError(field, { type: "server", message });
    const data: ProjectUpdate = {
      name: details.name,
      identifier: details.identifier,
      description: details.description,
      network: details.network,
      logo_props: details.logo_props,
      timezone: details.timezone,
    };
    return followInSession(
      async (): Promise<"updated" | "identifier_taken"> => {
        if (details.identifier !== project.identifier) {
          const { available } = await checkProjectIdentifier(workspaceSlug, details.identifier);
          if (!available) return "identifier_taken";
        }
        await updateProject(project.id, data);
        return "updated";
      },
      {
        done: (outcome) => {
          if (outcome === "identifier_taken") {
            underField("identifier", t("errors.project_identifier_taken"));
            return;
          }
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: t("toast.success"),
            message: t("project_settings.general.toast.success"),
          });
        },
        failed: (error) => showRefusal(error, underField),
      }
    );
  };
}
````

- [ ] **Step 2: 表单、页面和文案**

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx`（修改，4 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
  // router
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
  // router: the address's project, as the store gives it (the wrapper shows the page once it has)
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
  const { currentProjectDetails } = useProject();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
  const { getProjectById } = useProject();
  const project = getProjectById(projectId);
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
  const pageTitle = currentProjectDetails?.name ? `${currentProjectDetails?.name} - General Settings` : undefined;
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
  const pageTitle = project ? `${project.name} - General Settings` : undefined;
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
        {currentProjectDetails ? (
          <ProjectDetailsForm
            project={currentProjectDetails}
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            isAdmin={isAdmin}
          />
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx
        {project ? (
          // one form per project: another project's page opens with its own values
          <ProjectDetailsForm key={project.id} project={project} workspaceSlug={workspaceSlug} isAdmin={isAdmin} />
````

`web/apps/web/core/components/project/form.tsx`（整个文件，292 行）：

````whole web/apps/web/core/components/project/form.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { InfoOutline, LockOutline } from "@makeplane/propel/icons";
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { TextArea, TextAreaGroup } from "@makeplane/propel/components/text-area";
import { NETWORK_CHOICES } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
// nerve imports
import { Button } from "@nerve/propel/button";
import { EmojiPicker, EmojiIconPickerTypes, Logo } from "@nerve/propel/emoji-icon-picker";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import type { Project } from "@nerve/api-client";
import { CustomSelect } from "@nerve/ui";
import { projectIdentifierSanitizer, renderFormattedDate } from "@nerve/utils";
import { CoverImage } from "@/components/common/cover-image";
import { TimezoneSelect } from "@/components/global";
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { logoPropsOf } from "./logo-props";
import { ProjectNetworkIcon } from "./project-network-icon";
import { projectDetailsOf, useUpdateProjectDetails, type ProjectDetails } from "./use-update-project-details";

export interface IProjectDetailsForm {
  project: Project;
  workspaceSlug: string;
  isAdmin: boolean;
}

/**
 * A project's general settings (M3 design 7.6), as nerve last answered them when the page opened them: its page mounts
 * one per project. Its admin changes them; anyone else sees them, in a form he cannot change.
 */
export function ProjectDetailsForm(props: IProjectDetailsForm) {
  const { project, workspaceSlug, isAdmin } = props;
  const { t } = useTranslation();
  // states
  const [isOpen, setIsOpen] = useState(false);
  // store hooks
  const updateDetails = useUpdateProjectDetails();
  const { isMobile } = usePlatformOS();

  // form info
  const {
    handleSubmit,
    watch,
    control,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<ProjectDetails>({ defaultValues: projectDetailsOf(project) });
  // derived values
  const currentNetwork = NETWORK_CHOICES.find((n) => n.key === project.network);

  // the page is busy until nerve has answered and the page has followed the answer
  const onSubmit = (details: ProjectDetails) => updateDetails(project, workspaceSlug, details, setError);

  return (
    <form onSubmit={(event) => void handleSubmit(onSubmit)(event)}>
      <div className="relative h-44 w-full">
        <div className="absolute inset-0 bg-gradient-to-t from-black/50 to-transparent" />
        <CoverImage
          src={project.cover_image_url ?? undefined}
          showDefaultWhenEmpty
          alt="Project cover image"
          className="h-44 w-full rounded-md"
        />
        <div className="absolute bottom-4 z-5 flex w-full items-end justify-between gap-3 px-4">
          <div className="flex flex-grow gap-3 truncate">
            <Controller
              control={control}
              name="logo_props"
              render={({ field: { value, onChange } }) => (
                <EmojiPicker
                  ariaLabel={t("aria_labels.project_icon")}
                  iconType="material"
                  closeOnSelect={false}
                  isOpen={isOpen}
                  handleToggle={(val: boolean) => setIsOpen(val)}
                  className="flex items-center justify-center"
                  buttonClassName="flex h-[52px] w-[52px] flex-shrink-0 items-center justify-center rounded-lg bg-white/10"
                  label={<Logo logo={value} size={28} />}
                  onChange={(picked) => {
                    onChange(logoPropsOf(picked));
                    setIsOpen(false);
                  }}
                  defaultIconColor={value.in_use === "icon" ? value.icon?.color : undefined}
                  defaultOpen={value.in_use === "emoji" ? EmojiIconPickerTypes.EMOJI : EmojiIconPickerTypes.ICON}
                  disabled={!isAdmin}
                />
              )}
            />
            <div className="flex flex-col gap-1 truncate text-on-color">
              <span className="truncate text-16 font-semibold">{watch("name")}</span>
              <span className="flex items-center gap-2 text-13">
                <span>{watch("identifier")} .</span>
                <span className="flex items-center gap-1.5">
                  {project.network === 0 && <LockOutline className="h-2.5 w-2.5 text-on-color" />}
                  {currentNetwork && t(currentNetwork.i18n_label)}
                </span>
              </span>
            </div>
          </div>
        </div>
      </div>
      <div className="mt-8 flex flex-col gap-8">
        <div className="flex flex-col gap-1">
          <h4 className="text-13">{t("common.project_name")}</h4>
          <Controller
            control={control}
            name="name"
            rules={{
              required: t("name_is_required"),
              maxLength: {
                value: 255,
                message: "Project name should be less than 255 characters",
              },
            }}
            render={({ field: { value, onChange, ref } }) => (
              <Field name="name" invalid={Boolean(errors.name)}>
                <InputGroup size="2xl">
                  <Input
                    size="2xl"
                    id="name"
                    name="name"
                    type="text"
                    ref={ref}
                    value={value}
                    onChange={onChange}
                    placeholder={t("common.project_name")}
                    disabled={!isAdmin}
                  />
                </InputGroup>
              </Field>
            )}
          />
          <span className="text-11 text-danger-primary">{errors.name?.message}</span>
        </div>
        <div className="flex flex-col gap-1">
          <h4 className="text-13">{t("description")}</h4>
          <Controller
            name="description"
            control={control}
            render={({ field: { value, onChange } }) => (
              <Field name="description" invalid={Boolean(errors.description)} disabled={!isAdmin}>
                <TextAreaGroup resize="none">
                  <TextArea
                    size="lg"
                    surface="field"
                    autoResize
                    maxRows={8}
                    id="description"
                    name="description"
                    value={value}
                    placeholder={t("project_description_placeholder")}
                    onChange={onChange}
                  />
                </TextAreaGroup>
              </Field>
            )}
          />
        </div>
        <div className="grid grid-cols-1 gap-6 md:grid-cols-2">
          <div className="flex flex-col gap-1">
            <h4 className="text-13">Project ID</h4>
            <div className="relative">
              <Controller
                control={control}
                name="identifier"
                rules={{
                  required: t("project_id_is_required"),
                  validate: (value) => /^[ÇŞĞIİÖÜA-Z0-9]+$/.test(value.toUpperCase()) || t("project_id_allowed_char"),
                  minLength: {
                    value: 1,
                    message: t("project_id_min_char"),
                  },
                  maxLength: {
                    value: 10,
                    message: t("project_id_max_char"),
                  },
                }}
                render={({ field: { value, onChange, ref } }) => (
                  <Field name="identifier" invalid={Boolean(errors.identifier)}>
                    <InputGroup size="2xl">
                      <Input
                        size="2xl"
                        id="identifier"
                        name="identifier"
                        type="text"
                        value={value}
                        // upper case as typed, of the characters an identifier may have, as the creation's form
                        onChange={(event) => onChange(projectIdentifierSanitizer(event.target.value))}
                        ref={ref}
                        placeholder={t("project_settings.general.enter_project_id")}
                        disabled={!isAdmin}
                      />
                    </InputGroup>
                  </Field>
                )}
              />
              <Tooltip
                label={t("project_id_tooltip_content")}
                layout="stacked"
                side="right"
                align="start"
                disabled={isMobile}
              >
                <InfoOutline className="absolute top-2.5 right-2 h-4 w-4 text-placeholder" />
              </Tooltip>
            </div>
            <span className="text-11 text-danger-primary">{errors.identifier?.message}</span>
          </div>
          <div className="flex flex-col gap-1">
            <h4 className="text-13">{t("workspace_projects.network.label")}</h4>
            <Controller
              name="network"
              control={control}
              render={({ field: { value, onChange } }) => {
                const selectedNetwork = NETWORK_CHOICES.find((n) => n.key === value);
                return (
                  <CustomSelect
                    value={value}
                    onChange={onChange}
                    label={
                      <div className="flex items-center gap-1">
                        {selectedNetwork ? (
                          <>
                            <ProjectNetworkIcon iconKey={selectedNetwork.iconKey} className="h-3.5 w-3.5" />
                            {t(selectedNetwork.i18n_label)}
                          </>
                        ) : (
                          <span className="text-placeholder">{t("select_network")}</span>
                        )}
                      </div>
                    }
                    buttonClassName="!border-subtle !shadow-none font-medium rounded-md"
                    input
                    disabled={!isAdmin}
                  >
                    {NETWORK_CHOICES.map((network) => (
                      <CustomSelect.Option key={network.key} value={network.key}>
                        <div className="flex items-start gap-2">
                          <ProjectNetworkIcon iconKey={network.iconKey} className="h-3.5 w-3.5" />
                          <div className="-mt-1">
                            <p>{t(network.i18n_label)}</p>
                            <p className="text-11 text-placeholder">{t(network.description)}</p>
                          </div>
                        </div>
                      </CustomSelect.Option>
                    ))}
                  </CustomSelect>
                );
              }}
            />
          </div>
          <div className="col-span-1 flex flex-col gap-1 sm:col-span-2 xl:col-span-1">
            <h4 className="text-13">{t("common.project_timezone")}</h4>
            <Controller
              name="timezone"
              control={control}
              rules={{ required: t("project_settings.general.please_select_a_timezone") }}
              render={({ field: { value, onChange } }) => (
                <TimezoneSelect
                  value={value}
                  onChange={onChange}
                  error={Boolean(errors.timezone)}
                  buttonClassName="!border-subtle !shadow-none font-medium rounded-md"
                  disabled={!isAdmin}
                />
              )}
            />
            {errors.timezone && <span className="text-11 text-danger-primary">{errors.timezone.message}</span>}
          </div>
        </div>
        <div className="flex items-center justify-between py-2">
          <Button variant="primary" size="lg" type="submit" loading={isSubmitting} disabled={!isAdmin}>
            {isSubmitting ? t("updating") : t("common.update_project")}
          </Button>
          <span className="text-13 text-placeholder italic">
            {t("common.created_on")} {renderFormattedDate(project.created_at)}
          </span>
        </div>
      </div>
    </form>
  );
}
````

`web/packages/i18n/src/locales/en/common.json`（修改，2 处）：

````old web/packages/i18n/src/locales/en/common.json
  "project_created_successfully": "Project created successfully",
  "project_name_already_taken": "The project name is already taken.",
  "project_name_cannot_contain_special_characters": "The project name cannot contain special characters.",
  "project_identifier_already_taken": "The project identifier is already taken.",
````
````new web/packages/i18n/src/locales/en/common.json
  "project_created_successfully": "Project created successfully",
````
````old web/packages/i18n/src/locales/en/common.json
    "update_project": "Update project",
    "identifier_already_exists": "Identifier already exists",
````
````new web/packages/i18n/src/locales/en/common.json
    "update_project": "Update project",
````

`web/packages/i18n/src/locales/zh-CN/common.json`（修改，2 处）：

````old web/packages/i18n/src/locales/zh-CN/common.json
  "project_created_successfully": "项目创建成功",
  "project_name_already_taken": "项目名称已被使用。",
  "project_name_cannot_contain_special_characters": "项目名称不能包含特殊字符。",
  "project_identifier_already_taken": "项目标识符已被使用。",
````
````new web/packages/i18n/src/locales/zh-CN/common.json
  "project_created_successfully": "项目创建成功",
````
````old web/packages/i18n/src/locales/zh-CN/common.json
    "update_project": "更新项目",
    "identifier_already_exists": "标识符已存在",
````
````new web/packages/i18n/src/locales/zh-CN/common.json
    "update_project": "更新项目",
````

- [ ] **Step 3: 静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/components/projects/create/use-create-project.test.ts"
````
````new .oxlintrc.json
        "web/apps/web/core/components/projects/create/use-create-project.test.ts",
        "web/apps/web/core/components/project/use-update-project-details.ts",
        "web/apps/web/core/components/project/use-update-project-details.test.ts"
````

- [ ] **Step 4: 端到端**

`e2e/fixtures/settings-pages.ts`（修改，1 处）：

````old e2e/fixtures/settings-pages.ts

/**
 * Fills the security page's form, which page shows, with the current password and a new one typed twice, and
````
````new e2e/fixtures/settings-pages.ts

/**
 * Moves page to path as a link of the app would, without a load: React Router follows the history's popstate. A
 * story's stand-in for a link the app does not have yet.
 */
export async function moveWithinApp(page: Page, path: string): Promise<void> {
  await page.evaluate((to) => {
    window.history.pushState(null, "", to);
    window.dispatchEvent(new PopStateEvent("popstate"));
  }, path);
}

/**
 * Fills the security page's form, which page shows, with the current password and a new one typed twice, and
````

`e2e/stories/project/p3-project-settings.spec.ts`（修改，3 处）：

````old e2e/stories/project/p3-project-settings.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import type { Database } from "../../fixtures/db";
import { expect, test } from "../../fixtures/test";
````
````new e2e/stories/project/p3-project-settings.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, requestsElsewhere, watchPage } from "../../fixtures/browser";
import type { Database } from "../../fixtures/db";
import {
  answerTo,
  bodiesSentTo,
  enabledWithin,
  moveWithinApp,
  registerOnboarded,
  sentHeld,
} from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser } from "../../fixtures/workspace-pages";
````
````old e2e/stories/project/p3-project-settings.spec.ts
// P3, a project's settings (M3 design 2, 3.4, 3.5, 3.19). The page version
// comes with the project's settings pages (P10).
````
````new e2e/stories/project/p3-project-settings.spec.ts
// P3, a project's settings (M3 design 2, 3.4, 3.5, 3.19, 7.6).
````
````old e2e/stories/project/p3-project-settings.spec.ts
  expect(await stored(db, web.id)).toEqual({ ...changed, lead: null, default_assignee: null });
});

````
````new e2e/stories/project/p3-project-settings.spec.ts
  expect(await stored(db, web.id)).toEqual({ ...changed, lead: null, default_assignee: null });
});

test("P3 (page): the project's admin changes its name, identifier, description, visibility, time zone and icon on its general page, which hold after a reload; an identifier another project has is said under it and nothing is sent; the page waits for nerve; its member sees them and can change nothing", async ({
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
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  await inviteAndAccept(api, admin.access_token, slug, { email: memberEmail, token: member.access_token }, 15);
  await createProject(api, admin.access_token, slug, { name: "Ops", identifier: "OPS" });
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  const memberId = await accountId(api, member.access_token);
  await addProjectMembers(api, admin.access_token, web.id, [{ member_id: memberId, role: 15 }]);

  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/settings/projects/${web.id}`);
  await expect(page.locator("#name")).toHaveValue("Web");
  await page.locator("#name").fill("Site");
  await page.locator("#description").fill("The site");
  await page.getByRole("button", { name: "Public" }).click();
  await page.getByRole("option", { name: /Private/ }).click();
  // the workspace's time zone, which the project took, for Shanghai's
  await page.getByRole("button", { name: "UTC" }).click();
  await page.getByRole("combobox", { name: "Search" }).fill("Asia/Shanghai");
  await page.keyboard.press("ArrowDown");
  await page.keyboard.press("Enter");
  // The icon: the project has none, so the picker opens on its icons; an emoji from the emoji nerve serves.
  await page.getByRole("button", { name: "Project icon" }).click();
  await page.getByRole("tab", { name: "Emoji" }).click();
  await page.getByRole("searchbox").fill("rocket");
  await page.getByRole("gridcell", { name: "Rocket" }).click();
  await expect(page.getByRole("button", { name: "Project icon" })).toContainText("🚀");

  // An identifier another project has: the page asks nerve, says so under it, and sends no change.
  const changes = await bodiesSentTo(page, "PATCH", `/api/v0/projects/${web.id}`);
  const update = page.getByRole("button", { name: "Update project" });
  await page.locator("#identifier").fill("o.ps");
  await expect(page.locator("#identifier")).toHaveValue("OPS");
  const checked = await answerTo(page, "GET", `/api/v0/workspaces/${slug}/project-identifiers/OPS`, () =>
    update.click()
  );
  expect(await checked.json()).toEqual({ available: false });
  await expect(page.getByText("A project of this workspace already has this identifier.")).toBeVisible();
  await expect(update).toBeEnabled();
  expect(changes).toEqual([]);

  // Changed: the fields the page edits, the button busy until nerve answers.
  await page.locator("#identifier").fill("site");
  const { body, release } = await sentHeld(page, "PATCH", `/api/v0/projects/${web.id}`, () => update.click());
  const logo = { in_use: "emoji", emoji: { value: "128640" } };
  const settings = {
    name: "Site",
    identifier: "SITE",
    description: "The site",
    network: 0,
    logo_props: logo,
    timezone: "Asia/Shanghai",
  };
  expect(body).toEqual(settings);
  expect(await enabledWithin(page.getByRole("button", { name: "Updating" }))).toBe(false);
  expect((await release()).status()).toBe(200);
  await expect(page.getByText("Project updated successfully")).toBeVisible();
  await expect(update).toBeEnabled();
  expect(await stored(db, web.id)).toMatchObject({ ...settings, by: adminEmail });
  await page.reload();
  await expect(page.locator("#name")).toHaveValue("Site");
  await expect(page.locator("#identifier")).toHaveValue("SITE");
  await expect(page.locator("#description")).toHaveValue("The site");
  await expect(page.getByRole("button", { name: "Private" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Beijing" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Project icon" })).toContainText("🚀");

  // Its member sees what nerve holds, in a form he cannot change, and no archiving or deletion.
  const theMember = await anotherBrowser(browser, baseURL ?? "", member);
  const memberWatch = await watchPage(theMember.page);
  await theMember.page.goto(`/${slug}/settings/projects/${web.id}`);
  await expect(theMember.page.locator("#name")).toHaveValue("Site");
  await Promise.all(
    ["#name", "#identifier", "#description"].map((field) => expect(theMember.page.locator(field), field).toBeDisabled())
  );
  await Promise.all(
    ["Project icon", "Private", "Beijing", "Update project"].map((name) =>
      expect(theMember.page.getByRole("button", { name }), name).toBeDisabled()
    )
  );
  await expect(theMember.page.getByRole("button", { name: "Archive" })).toHaveCount(0);
  await expect(theMember.page.getByRole("button", { name: "Delete" })).toHaveCount(0);
  expect([memberWatch.apiFailures, memberWatch.oldApiRequests, memberWatch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(theMember.page, memberWatch, { warnings: [EMOJI_CHECK_WARNING] });
  await theMember.close();

  // The emoji came from nerve: nothing asked of another address, nothing blocked.
  expect(requestsElsewhere(page, watch)).toEqual([]);
  expect([watch.cspViolations, watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], [], []]);
  // the first load, and the reload
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING] });
});

test("P3 (page): another project's general page, reached without leaving the route, shows that project's values", async ({
  api,
  signedInPage,
}, testInfo) => {
  const admin = await registerOnboarded(api, emailFor(testInfo, "admin"));
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  const ops = await createProject(api, admin.access_token, slug, { name: "Ops", identifier: "OPS" });
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  // the router's own move, as a switcher on the settings pages would make it: no link of the app does it yet
  await page.goto(`/${slug}/settings/projects/${web.id}`);
  await expect(page.locator("#name")).toHaveValue("Web");
  await moveWithinApp(page, `/${slug}/settings/projects/${ops.id}`);
  await expect(page.locator("#name")).toHaveValue("Ops");
  // Web's read is in the session's cache: the wrapper shows its page at once, the route mounted throughout
  await moveWithinApp(page, `/${slug}/settings/projects/${web.id}`);
  await expect(page.locator("#identifier")).toHaveValue("WEB");

  expect([watch.cspViolations, watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});

````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 67 条规则、3 个例外，没有命中；web 的 oxlint 350 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 101 个全部通过。

- [ ] **Step 6: 提交**

```bash
git add .oxlintrc.json e2e/fixtures/settings-pages.ts e2e/stories/project/p3-project-settings.spec.ts 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/page.tsx' web/apps/web/core/components/project/form.tsx web/apps/web/core/components/project/use-update-project-details.test.ts web/apps/web/core/components/project/use-update-project-details.ts web/packages/i18n/src/locales/en/common.json web/packages/i18n/src/locales/zh-CN/common.json
```
```bash
git commit -m "feat(M3/P10): a project's general page asks nerve of a changed identifier first, sends its fields and waits for nerve

useUpdateProjectDetails sends one change, followed only in the session it
was sent in: an identifier changed is asked of nerve first, and one taken
is said under it with nothing sent; else the page's fields go out as
ProjectUpdate, and nerve's refusal shows under the fields it names or as
its reason. The form is busy until the page has followed nerve's answer,
the page mounts one form per project, and the strings no page reads are
deleted. P3's page version of the general page.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p10.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T5.1` | nerve 回答可用的标识符被说成已被占用，已被占用的却发出去（P8b 的 P6） | `use-update-project-details.test.ts`、故事 P3 | vitest；端到端 |
| `T5.2` | 没有改的标识符也去问 nerve | `use-update-project-details.test.ts` | vitest |
| `T5.3` | 发出项目原来的标识符，不是输入的 | `use-update-project-details.test.ts`、故事 P3 | vitest；端到端 |
| `T5.4` | 修改不论会话都跟进 | `use-update-project-details.test.ts` | vitest |
| `T5.5` | 修改在途时更新按钮仍可用 | 故事 P3 | 端到端 |
| `T5.6` | 输入的标识符保留大小写和其他字符 | 故事 P3 | 端到端 |
| `T5.7` | general 页不随项目重建表单：路由移到另一个项目时仍是第一个项目的值（去掉 `key`） | 故事 P3 | 端到端 |
| `T5.8` | 在 general 页的 hook 里写一个非空断言 | oxlint（`check:lint`） | 静态 |

---

### Task 6: 项目设置的 features 和 automations；P3 的页面版本（二）

**Files:**
- Modify: `.oxlintrc.json`、`e2e/stories/project/p3-project-settings.spec.ts`、`web/apps/web/core/components/automation/auto-archive-automation.test.tsx`、`web/apps/web/core/components/automation/auto-archive-automation.tsx`、`web/apps/web/core/components/automation/select-month-modal.test.tsx`、`web/apps/web/core/components/automation/select-month-modal.tsx`、`web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx`、`web/apps/web/core/components/project/settings/use-feature-toggle.ts`

**Interfaces:**
- Produces（spec 2.6；M3 设计 2 的 P3，3.19，7.1、7.6；总体设计 7.7 的 PF-M2：开关在轮到它时由 store 决定，P8b 已做）：
  - `useFeatureToggle(workspaceSlug, projectId)`（`project/settings/use-feature-toggle.ts`，整个文件）：一下开关经 `followInSession(() => toggleProject(projectId, field), …)`：完成时提示（Plane 的"Project feature updated successfully."），被拒绝时经 `useRefusalToast` 说 nerve 的原因；原来的 `setPromiseToast` 不论会话都提示，失败的文案也不是 nerve 的原因。
  - `AutoArchiveAutomation`：项目是地址的（`useParams` 的 `projectId`，经 `getProjectById`；原来读路由 store 的 `currentProjectDetails`）；`send(change, done?)` 经 `followInSession`，拒绝经 `useRefusalToast`（原来的 `try/catch` 提示固定的"Something went wrong"）；`handleChange(formData, done?)` 把 `done` 交给自定义时长的弹窗。
  - `SelectMonthModal`：`handleChange(formData, done)`，提交时 `await` 它，nerve 做完之后才关（原来发出就关，被拒绝之后弹窗已不在）；在途时关不掉：取消禁用、`handleClose` 为空（Escape、背景不关）。
  - 静态检查：`.oxlintrc.json` 的非空断言的范围加上 `use-feature-toggle.ts` 和它的测试、两个自动化组件的测试。

**Tests:**
- vitest：`use-feature-toggle.test.tsx`：`says a turn is done, and nerve's reason for refusing one`（改写原来的"拒绝的提示"）；`says nothing when a turn $settles after another tab moved this one`。`auto-archive-automation.test.tsx`：`shows nerve's reason in a toast when it refuses the turn`（改写）；`says nothing when the turn $settles after another tab moved this one`。`select-month-modal.test.tsx`：`closes when the change says nerve made the range, not when it is sent`。
- 端到端：`P3 (page): the project's admin turns its cycles, modules, views and intake on, each on its feature's page, and has its closed work items archived, after a range of his own that the page holds until nerve answers, then after 3 months`（nerve 建的项目四个功能都关着；每页的开关只发它的字段（`sentTo`），开关随之打开；自动归档打开发 `{ archive_in: 1 }`；自定义 6 个月的修改被扣住（`sentHeld`）：取消禁用、Escape 不关，放行之后弹窗关上；再从列表选 3 个月）。

- [ ] **Step 1: 功能的开关**

`web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx`（修改，7 处）：

````old web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
import { ProjectSettingsFeatureControlItem } from "@/components/settings/project/content/feature-control-item";
import { emptyShown, shown } from "@/lib/fake-controls";
````
````new web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
import { ProjectSettingsFeatureControlItem } from "@/components/settings/project/content/feature-control-item";
import { heldChange, lateSettlings, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
````
````old web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
// store holds is built in its turn): the hook runs as a plain function, and the two pages that flip a feature render
// on the server, with stand-ins for their switches (fake-controls.ts), which keep the props they were given, and for
// the project store, whose changes record their arguments.
````
````new web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
// store holds is built in its turn), and what the page says of it (M3 design 7.1): the hook runs as a plain function,
// and the two pages that flip a feature render on the server, with stand-ins for their switches (fake-controls.ts),
// which keep the props they were given, and for the project store, whose changes record their arguments. Its session
// is fake-tab.ts's.
````
````old web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
vi.mock("@makeplane/propel/components/switch", () => import("@/lib/fake-controls"));
````
````new web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
vi.mock("@makeplane/propel/components/switch", () => import("@/lib/fake-controls"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
````
````old web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
beforeEach(() => {
````
````new web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
beforeEach(() => {
  signedIn();
````
````old web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
  it("shows nerve's refusal of a turn in an error toast", async () => {
    store.toggleProject.mockImplementationOnce(() => Promise.reject(new Error("refused")));
````
````new web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
  it("says a turn is done, and nerve's reason for refusing one", async () => {
````
````old web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
    await vi.waitFor(() => expect(toasts).toHaveLength(1));
````
````new web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
    store.toggleProject.mockImplementationOnce(() => Promise.reject(refusal(403, "forbidden")));
    useFeatureToggle("acme", web.id)("cycle_view");
    await pageSettled();
````
````old web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
      {
        type: "error",
        title: "Error!",
        message: "Something went wrong while updating project feature. Please try again.",
      },
    ]);
````
````new web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx
      { type: "success", title: "Success!", message: "Project feature updated successfully." },
      { type: "error", title: "toast.error", message: "errors.forbidden" },
    ]);
  });

  it.each(lateSettlings)("says nothing when a turn $settles after another tab moved this one", async ({ settle }) => {
    const turn = heldChange<undefined>();
    store.toggleProject.mockReturnValueOnce(turn.sent);
    useFeatureToggle("acme", web.id)("module_view");
    switchAccount();
    settle(turn);
    await pageSettled();
    expect(toasts).toEqual([]);
````

`web/apps/web/core/components/project/settings/use-feature-toggle.ts`（整个文件，38 行）：

````whole web/apps/web/core/components/project/settings/use-feature-toggle.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// nerve imports
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
import type { ProjectToggleField } from "@/store/project/project.store";

/**
 * What a flip of a feature's switch does in the project's settings, in the features list and on a feature's own page:
 * the project store turns the feature the other way from nerve's last answer, in the change's turn
 * (ProjectStore.toggleProject; v0 design 7.7), not from what the switch shows. A toast says it is done, or nerve's
 * reason for refusing it, only in the session the flip was sent in (M3 design 7.1).
 */
export function useFeatureToggle(workspaceSlug: string, projectId: string) {
  const { toggleProject } = useProject();
  const toastRefusal = useRefusalToast();

  return (field: ProjectToggleField) => {
    if (!workspaceSlug || !projectId) return;

    void followInSession(() => toggleProject(projectId, field), {
      done: () =>
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: "Success!",
          message: "Project feature updated successfully.",
        }),
      failed: toastRefusal,
    });
  };
}
````

- [ ] **Step 2: 自动归档和自定义时长**

`web/apps/web/core/components/automation/auto-archive-automation.test.tsx`（修改，7 处）：

````old web/apps/web/core/components/automation/auto-archive-automation.test.tsx
import type { Project, ProjectUpdate } from "@nerve/api-client";
import { emptyShown, shown } from "@/lib/fake-controls";
````
````new web/apps/web/core/components/automation/auto-archive-automation.test.tsx
import type { Project, ProjectUpdate } from "@nerve/api-client";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
````
````old web/apps/web/core/components/automation/auto-archive-automation.test.tsx
    currentProjectDetails: web,
````
````new web/apps/web/core/components/automation/auto-archive-automation.test.tsx
    getProjectById: (projectId: string) => (projectId === web.id ? web : undefined),
````
````old web/apps/web/core/components/automation/auto-archive-automation.test.tsx
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "acme" }) }));
````
````new web/apps/web/core/components/automation/auto-archive-automation.test.tsx
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "acme", projectId: web.id }) }));
````
````old web/apps/web/core/components/automation/auto-archive-automation.test.tsx
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
````
````new web/apps/web/core/components/automation/auto-archive-automation.test.tsx
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
````
````old web/apps/web/core/components/automation/auto-archive-automation.test.tsx
beforeEach(() => {
````
````new web/apps/web/core/components/automation/auto-archive-automation.test.tsx
beforeEach(() => {
  signedIn();
````
````old web/apps/web/core/components/automation/auto-archive-automation.test.tsx
  it("shows a toast when nerve refuses the turn", async () => {
    store.toggleAutoArchive.mockImplementationOnce(() => Promise.reject(new Error("refused")));
````
````new web/apps/web/core/components/automation/auto-archive-automation.test.tsx
  it("shows nerve's reason in a toast when it refuses the turn", async () => {
    store.toggleAutoArchive.mockImplementationOnce(() => Promise.reject(refusal(403, "forbidden")));
````
````old web/apps/web/core/components/automation/auto-archive-automation.test.tsx
    expect(toasts).toEqual([{ type: "error", title: "Error!", message: "Something went wrong. Please try again." }]);
````
````new web/apps/web/core/components/automation/auto-archive-automation.test.tsx
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.forbidden" }]);
  });

  it.each(lateSettlings)("says nothing when the turn $settles after another tab moved this one", async ({ settle }) => {
    const turn = heldChange<undefined>();
    store.toggleAutoArchive.mockReturnValueOnce(turn.sent);
    const { toggle } = render();
    const turned = toggle.onCheckedChange(false);
    switchAccount();
    settle(turn);
    await turned;
    expect(toasts).toEqual([]);
````

`web/apps/web/core/components/automation/auto-archive-automation.tsx`（修改，13 处）：

````old web/apps/web/core/components/automation/auto-archive-automation.tsx
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
````
````new web/apps/web/core/components/automation/auto-archive-automation.tsx
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/automation/auto-archive-automation.tsx
import { useUserPermissions } from "@/hooks/store/user";
````
````new web/apps/web/core/components/automation/auto-archive-automation.tsx
import { useUserPermissions } from "@/hooks/store/user";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
````
````old web/apps/web/core/components/automation/auto-archive-automation.tsx
 * from what the switch shows; its select and the custom range send the months picked. A refusal shows a toast.
````
````new web/apps/web/core/components/automation/auto-archive-automation.tsx
 * from what the switch shows; its select and the custom range send the months picked. The page follows each only in
 * the session it was sent in (M3 design 7.1), a refusal with nerve's reason in a toast.
````
````old web/apps/web/core/components/automation/auto-archive-automation.tsx
  const { workspaceSlug } = useParams();
````
````new web/apps/web/core/components/automation/auto-archive-automation.tsx
  const { workspaceSlug, projectId } = useParams();
````
````old web/apps/web/core/components/automation/auto-archive-automation.tsx
  const { t } = useTranslation();
````
````new web/apps/web/core/components/automation/auto-archive-automation.tsx
  const { t } = useTranslation();
  const toastRefusal = useRefusalToast();
````
````old web/apps/web/core/components/automation/auto-archive-automation.tsx
  const { currentProjectDetails, updateProject, toggleAutoArchive } = useProject();
````
````new web/apps/web/core/components/automation/auto-archive-automation.tsx
  const { getProjectById, updateProject, toggleAutoArchive } = useProject();
  const project = projectId ? getProjectById(projectId) : undefined;
````
````old web/apps/web/core/components/automation/auto-archive-automation.tsx
  const isAdmin = allowPermissions(
    [EUserPermissions.ADMIN],
    EUserPermissionsLevel.PROJECT,
    workspaceSlug,
    currentProjectDetails?.id
  );
````
````new web/apps/web/core/components/automation/auto-archive-automation.tsx
  const isAdmin = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.PROJECT, workspaceSlug, project?.id);
````
````old web/apps/web/core/components/automation/auto-archive-automation.tsx
  /** Sends a change of the current project's auto-archiving; a refusal shows a toast. */
  const send = async (change: (projectId: string) => Promise<unknown>) => {
    if (!currentProjectDetails) return;
````
````new web/apps/web/core/components/automation/auto-archive-automation.tsx
  /**
   * Sends a change of the current project's auto-archiving, and follows it: done, once nerve has made it; a refusal,
   * with nerve's reason in a toast. Settles once the page has followed it.
   */
  const send = async (change: (id: string) => Promise<unknown>, done?: () => void) => {
    if (!project) return;
````
````old web/apps/web/core/components/automation/auto-archive-automation.tsx
    try {
      await change(currentProjectDetails.id);
    } catch {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Something went wrong. Please try again.",
      });
    }
````
````new web/apps/web/core/components/automation/auto-archive-automation.tsx
    await followInSession(() => change(project.id), { done, failed: toastRefusal });
````
````old web/apps/web/core/components/automation/auto-archive-automation.tsx
  const handleChange = (formData: Pick<ProjectUpdate, "archive_in">) =>
    send((projectId) => updateProject(projectId, formData));
  const handleToggle = () => send((projectId) => toggleAutoArchive(projectId));
````
````new web/apps/web/core/components/automation/auto-archive-automation.tsx
  const handleChange = (formData: Pick<ProjectUpdate, "archive_in">, done?: () => void) =>
    send((id) => updateProject(id, formData), done);
  const handleToggle = () => send((id) => toggleAutoArchive(id));
````
````old web/apps/web/core/components/automation/auto-archive-automation.tsx
    if (currentProjectDetails?.archive_in === undefined) return false;
    return currentProjectDetails.archive_in !== 0;
  }, [currentProjectDetails]);
````
````new web/apps/web/core/components/automation/auto-archive-automation.tsx
    if (project?.archive_in === undefined) return false;
    return project.archive_in !== 0;
  }, [project]);
````
````old web/apps/web/core/components/automation/auto-archive-automation.tsx
        {currentProjectDetails ? (
````
````new web/apps/web/core/components/automation/auto-archive-automation.tsx
        {project ? (
````
````old web/apps/web/core/components/automation/auto-archive-automation.tsx
                    value={currentProjectDetails?.archive_in}
                    label={`${currentProjectDetails?.archive_in} ${
                      currentProjectDetails?.archive_in === 1 ? "month" : "months"
                    }`}
````
````new web/apps/web/core/components/automation/auto-archive-automation.tsx
                    value={project?.archive_in}
                    label={`${project?.archive_in} ${project?.archive_in === 1 ? "month" : "months"}`}
````

`web/apps/web/core/components/automation/select-month-modal.test.tsx`（修改，2 处）：

````old web/apps/web/core/components/automation/select-month-modal.test.tsx
});

describe("the auto-archiving's custom range", () => {
````
````new web/apps/web/core/components/automation/select-month-modal.test.tsx
});

/** Renders the custom range with handleChange, types 6 months in it and submits it; gives its handleClose. */
async function submitSix(
  handleChange: (formData: Pick<ProjectUpdate, "archive_in">, done: () => void) => Promise<void>
) {
  const handleClose = vi.fn();
  renderToStaticMarkup(
    <SelectMonthModal isOpen initialValues={{ archive_in: 1 }} handleClose={handleClose} handleChange={handleChange} />
  );
  const [months] = shown.inputs;
  if (!months) throw new Error("the modal showed no months");
  months.onChange({ target: { value: "6" } });
  await submitModalForm();
  return handleClose;
}

describe("the auto-archiving's custom range", () => {
````
````old web/apps/web/core/components/automation/select-month-modal.test.tsx
    const handleChange = vi.fn(async (_formData: Pick<ProjectUpdate, "archive_in">) => {});
    renderToStaticMarkup(
      <SelectMonthModal isOpen initialValues={{ archive_in: 1 }} handleClose={vi.fn()} handleChange={handleChange} />
    );
    const [months] = shown.inputs;
    if (!months) throw new Error("the modal showed no months");
    months.onChange({ target: { value: "6" } });
    await submitModalForm();
    expect(handleChange.mock.calls).toEqual([[{ archive_in: 6 }]]);
````
````new web/apps/web/core/components/automation/select-month-modal.test.tsx
    const handleChange = vi.fn(async (_formData: Pick<ProjectUpdate, "archive_in">, _done: () => void) => {});
    await submitSix(handleChange);
    expect(handleChange.mock.calls.map(([formData]) => formData)).toEqual([{ archive_in: 6 }]);
  });

  it("closes when the change says nerve made the range, not when it is sent", async () => {
    let made: (() => void) | undefined;
    const handleClose = await submitSix(async (_formData, done) => {
      made = done;
    });
    expect(handleClose).not.toHaveBeenCalled();
    made?.();
    expect(handleClose).toHaveBeenCalledTimes(1);
````

`web/apps/web/core/components/automation/select-month-modal.tsx`（修改，5 处）：

````old web/apps/web/core/components/automation/select-month-modal.tsx
  handleChange: (formData: TArchiveIn) => Promise<void>;
````
````new web/apps/web/core/components/automation/select-month-modal.tsx
  /** Sends the range; calls done once nerve has made it, and settles once the page has followed it. */
  handleChange: (formData: TArchiveIn, done: () => void) => Promise<void>;
````
````old web/apps/web/core/components/automation/select-month-modal.tsx
  const onSubmit = (formData: TArchiveIn) => {
````
````new web/apps/web/core/components/automation/select-month-modal.tsx
  // the modal waits for nerve, and cannot be closed meanwhile: it closes once nerve has made the range
  const onSubmit = async (formData: TArchiveIn) => {
````
````old web/apps/web/core/components/automation/select-month-modal.tsx
    handleChange(formData);
    onClose();
````
````new web/apps/web/core/components/automation/select-month-modal.tsx
    await handleChange(formData, onClose);
````
````old web/apps/web/core/components/automation/select-month-modal.tsx
    <ModalCore isOpen={isOpen} handleClose={onClose} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
````
````new web/apps/web/core/components/automation/select-month-modal.tsx
    <ModalCore
      isOpen={isOpen}
      handleClose={isSubmitting ? undefined : onClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.XXL}
    >
````
````old web/apps/web/core/components/automation/select-month-modal.tsx
          <Button variant="secondary" size="lg" onClick={onClose}>
````
````new web/apps/web/core/components/automation/select-month-modal.tsx
          <Button variant="secondary" size="lg" onClick={onClose} disabled={isSubmitting}>
````

- [ ] **Step 3: 静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/components/project/use-update-project-details.test.ts"
````
````new .oxlintrc.json
        "web/apps/web/core/components/project/use-update-project-details.test.ts",
        "web/apps/web/core/components/project/settings/use-feature-toggle.ts",
        "web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx",
        "web/apps/web/core/components/automation/auto-archive-automation.test.tsx",
        "web/apps/web/core/components/automation/select-month-modal.test.tsx"
````

- [ ] **Step 4: 端到端**

`e2e/stories/project/p3-project-settings.spec.ts`（修改，3 处）：

````old e2e/stories/project/p3-project-settings.spec.ts
  bodiesSentTo,
````
````new e2e/stories/project/p3-project-settings.spec.ts
  bodiesSentTo,
  closedByEscape,
````
````old e2e/stories/project/p3-project-settings.spec.ts
  sentHeld,
````
````new e2e/stories/project/p3-project-settings.spec.ts
  sentHeld,
  sentTo,
````
````old e2e/stories/project/p3-project-settings.spec.ts
});

test("P3 (page): another project's general page, reached without leaving the route, shows that project's values", async ({
````
````new e2e/stories/project/p3-project-settings.spec.ts
});

test("P3 (page): the project's admin turns its cycles, modules, views and intake on, each on its feature's page, and has its closed work items archived, after a range of his own that the page holds until nerve answers, then after 3 months", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  const settings = `/${slug}/settings/projects/${web.id}`;
  const projectApi = `/api/v0/projects/${web.id}`;

  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  /** Opens the page of the feature of path and turns it on: the feature alone is sent, the switch then shows it. */
  const flip = async (path: string, name: string) => {
    await page.goto(`${settings}/features/${path}`);
    const toggle = page.getByRole("switch", { name });
    await expect(toggle).not.toBeChecked();
    const flipped = await sentTo(page, "PATCH", projectApi, () => toggle.click());
    await expect(toggle).toBeChecked();
    return [flipped.answer.status(), flipped.body];
  };
  expect(await flip("cycles", "Enable cycles")).toEqual([200, { cycle_view: true }]);
  expect(await flip("modules", "Enable modules")).toEqual([200, { module_view: true }]);
  expect(await flip("views", "Enable views")).toEqual([200, { issue_views_view: true }]);
  expect(await flip("intake", "Enable intake")).toEqual([200, { intake_view: true }]);

  // The auto-archiving: on, after a month.
  await page.goto(`${settings}/automations`);
  const turnedOn = await sentTo(page, "PATCH", projectApi, () =>
    page.getByRole("switch", { name: "Auto-archive closed work items" }).click()
  );
  expect([turnedOn.answer.status(), turnedOn.body]).toEqual([200, { archive_in: 1 }]);
  // A range of his own: the modal waits for nerve, and cannot be closed meanwhile; it closes once nerve has made it.
  await page.getByRole("button", { name: "1 month" }).click();
  await page.getByRole("button", { name: "Customize time range" }).click();
  await page.locator("#archive_in").fill("6");
  const { body, release } = await sentHeld(page, "PATCH", projectApi, () =>
    page.getByRole("button", { name: "Submit" }).click()
  );
  expect(body).toEqual({ archive_in: 6 });
  await expect(page.getByRole("button", { name: "Cancel" })).toBeDisabled();
  expect(await closedByEscape(page)).toBe(false);
  expect((await release()).status()).toBe(200);
  await expect(page.locator("#archive_in")).toHaveCount(0);
  await expect(page.getByRole("button", { name: "6 months" })).toBeVisible();
  // Then 3 months, from the list.
  await page.getByRole("button", { name: "6 months" }).click();
  const three = await sentTo(page, "PATCH", projectApi, () => page.getByRole("option", { name: "3 months" }).click());
  expect([three.answer.status(), three.body]).toEqual([200, { archive_in: 3 }]);
  await expect(page.getByRole("button", { name: "3 months" })).toBeVisible();

  expect(await stored(db, web.id)).toMatchObject({
    cycle_view: true,
    module_view: true,
    issue_views_view: true,
    intake_view: true,
    archive_in: 3,
    by: adminEmail,
  });
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  // a load of each feature's page, and of the automations page
  await expectQuietConsole(page, watch, { warnings: Array.from({ length: 5 }, () => EMOJI_CHECK_WARNING) });
});

test("P3 (page): another project's general page, reached without leaving the route, shows that project's values", async ({
````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 67 条规则、3 个例外，没有命中；web 的 oxlint 350 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 102 个全部通过。

- [ ] **Step 6: 提交**

```bash
git add .oxlintrc.json e2e/stories/project/p3-project-settings.spec.ts web/apps/web/core/components/automation/auto-archive-automation.test.tsx web/apps/web/core/components/automation/auto-archive-automation.tsx web/apps/web/core/components/automation/select-month-modal.test.tsx web/apps/web/core/components/automation/select-month-modal.tsx web/apps/web/core/components/project/settings/use-feature-toggle.test.tsx web/apps/web/core/components/project/settings/use-feature-toggle.ts
```
```bash
git commit -m "feat(M3/P10): the features' switches and the auto-archiving follow nerve's answer in their session, and the custom range waits for it

A flip of a feature and every change of the auto-archiving go through
followInSession: done says so, a refusal says nerve's reason, and neither
is said once another tab has moved this one to another account. The
custom range closes once nerve has made it, not as it sends, and cannot
be closed while the change is out. P3's page version of the features and
the automations.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p10.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T6.1` | 功能的开关不论会话都跟进 | `use-feature-toggle.test.tsx` | vitest |
| `T6.2` | 功能的开关被拒绝时什么都不说 | `use-feature-toggle.test.tsx` | vitest |
| `T6.3` | 自动归档的修改不论会话都跟进 | `auto-archive-automation.test.tsx` | vitest |
| `T6.4` | 自定义时长的修改在途时 Escape 关掉它 | 故事 P3 | 端到端 |
| `T6.5` | 自定义时长的修改在途时取消按钮仍可用 | 故事 P3 | 端到端 |
| `T6.6` | 自定义时长在发出时就关闭，不等 nerve 做完 | `select-month-modal.test.tsx`、故事 P3 | vitest；端到端 |
| `T6.7` | 在功能开关的 hook 里写一个非空断言 | oxlint（`check:lint`） | 静态 |

---

### Task 7: 成员页的改角色、移出、离开；P5 的页面版本（一）

**Files:**
- Create: `e2e/fixtures/project-pages.ts`、`web/apps/web/core/components/project/settings/use-project-membership-changes.test.ts`、`web/apps/web/core/components/project/settings/use-project-membership-changes.ts`
- Modify: `.oxlintrc.json`、`e2e/stories/project/p5-project-members.spec.ts`、`tools/keywords.json`、`web/apps/web/core/components/project/confirm-project-member-remove.tsx`、`web/apps/web/core/components/project/leave-project-modal.tsx`、`web/apps/web/core/components/project/member-list-item.tsx`、`web/apps/web/core/components/project/project-roles.test.tsx`、`web/apps/web/core/components/project/project-roles.ts`、`web/apps/web/core/components/project/settings/member-columns.tsx`、`web/apps/web/core/components/projects/settings/useProjectColumns.tsx`、`web/apps/web/package.json`

**Interfaces:**
- Produces（spec 2.7；M3 设计 2 的 P5，3.5、3.7，7.1、7.6；M1-P4 交接的离开的顺序；P8b spec 第 5 节 P10 一行：`err.error` 和 `console.log`）：
  - `core/components/project/settings/use-project-membership-changes.ts`：`useProjectMembershipChanges(workspaceSlug, projectId)` 给出 `changeRole(userId, role: ProjectRole)`（发编号，生成的 `ProjectMemberUpdate`）、`remove(userId, done?)`、`leave(done?)`，都经 `followInSession`，拒绝经 `useRefusalToast`；`done` 是发出移出、离开的弹窗的关上，只在 nerve 做完之后（经 `followInSession`，在发出它的会话里），被拒绝时弹窗留着（spec 第 3 节第 6 条，预检之后的 A-M2：每个弹窗一条规则）；`leave` 在 nerve 做完之后才关上弹窗、去 `/{slug}/projects`（M1-P4：原来先跳转再离开，nerve 拒绝时人已离开页面；唯一的管理员离开被拒绝，停在原处），忙到跳转做完。
  - `project-roles.ts`（整个文件）：`MembershipCaller`（调用者在工作区和项目中的角色）、`ShownMembership`（是否他自己的、角色、成员在工作区的角色）；`roleChoices(caller, membership)`：nerve 允许的角色（M3 设计 3.5：工作区管理员不受相对规则限制；别人只改低于自己的、只给低于自己的，不改自己的；工作区访客只能是访客；没有权限时没有选项）；`canRemove(caller, membership)`：不是自己的，角色不高于自己的（工作区管理员同样）。成员页只提供这些，不提供 nerve 会拒绝的（P5b spec 第 5 节）。
  - 成员页：`member-columns.tsx`（整个文件）去掉从没有提交过的 `useForm`、`console.log` 和 `err.error`；角色列给出 `roleChoices`，没有选项时只显示角色，角色的名字经 `ROLE_DETAILS[role].i18n_title` 翻译（原来是英文的 `ROLE`）；名字列的菜单在自己一行是"离开"，别人一行在 `canRemove` 时是"移出"。`useProjectColumns` 算出调用者（`MembershipCaller`）和每一行的 `ShownMembership`。`member-list-item.tsx`（整个文件）经 `useProjectMembershipChanges`（原来读 `err.error`、自己跳转），确认框的关上交给 `remove`、`leave` 作 `done`。
  - 确认框 `confirm-project-member-remove.tsx`：移出、离开在途时关不掉（取消禁用、`handleClose` 为空）；它自己不再在 `await onSubmit()` 之后关上，只结束忙碌：关上是成功的跟进（`onSubmit` 的 `done`），被拒绝时留着、按钮重新可用，原因在共用的拒绝提示里；确认按钮的 `tabIndex={1}` 删除（oxlint：正的 `tabIndex`），web 的上限 350 → 349。
  - 离开弹窗 `leave-project-modal.tsx`：表单的值有类型（原来是 `any`），核对名称和"Leave Project"之后 `await leave(handleClose)`：nerve 做完之后关上、去项目列表，被拒绝时留着；在途时关不掉。侧边栏和项目页头的离开都经它，两处都只对项目的访客提供离开。
  - 静态检查：`.oxlintrc.json` 的非空断言的范围加上 `use-project-membership-changes.ts`、`project-roles.ts` 和它们的测试；`tools/keywords.json` 加规则 `project-leave`：web 应用的 `app`、`core` 中只有这个 hook 调 store 的 `leaveProject`（store 自己的定义和测试不算），离开的顺序和拒绝只写在 `leave` 里（spec 2.14）。
- e2e 的共用部分：新文件 `e2e/fixtures/project-pages.ts`：`Sent`（请求的方法和路径）、`removalOf(id)`、`leavingOf(id)`、`endProjectMembership(page, email, …)`（在成员页打开某一行的菜单，选"离开"或"移出"，在确认框中确认，交回 nerve 的回答）。

**Tests:**
- vitest：`use-project-membership-changes.test.ts`：`sends a member's new role as its number, to the membership's project`；`removes a member of the project, and stays, its dialog closed once nerve has`；`leaves the project, and closes its dialog and shows the workspace's projects only once nerve has made it, settling once they show`；`shows nerve's reason when it refuses $change, and stays, its dialog open`（三种修改各一行）；`does nothing on the page when it $settles after another tab moved this one to another account`（三种修改各两行：弹窗不关、不跳转、不提示）。`project-roles.test.tsx` 的角色表加上每个角色显示的键（`role_details.*.title`），成员页的选择按键选；加一张表 `what the members page offers > $who`：项目管理员、工作区管理员（是或不是项目的成员）、项目成员对成员、另一位管理员、自己、工作区的访客，各自可给的角色（`roleChoices`）和能否移出（`canRemove`）。
- 端到端：`P5 (page): a project admin who is no workspace admin is offered only the roles below his own, none for another admin; he makes a member a guest, removes the other admin and a guest, the dialog held until nerve answers, and, its only admin now, is told why he may not leave, its dialog open; a member leaves, and the workspace's projects show once nerve has made it, not before; the guest leaves by the sidebar, its modal open after a refusal and held too`（改角色的请求体 `{ role: 5 }`；移出被扣住时取消禁用、Escape 不关；唯一的管理员离开被拒绝（409），页面说明原因，确认框还开着、按钮可用；成员离开被扣住时页面不跳转；访客从侧边栏离开：弹窗开着时管理员经 API 结束了她的成员关系，nerve 拒绝（403），弹窗留着、按钮可用；管理员把她加回来，她再离开，弹窗被扣住时关不掉）。

- [ ] **Step 1: 角色的规则和成员关系的修改**

`web/apps/web/core/components/project/project-roles.test.tsx`（修改，10 处）：

````old web/apps/web/core/components/project/project-roles.test.tsx
import type { ProjectMembersAdd, ProjectRole } from "@nerve/api-client";
````
````new web/apps/web/core/components/project/project-roles.test.tsx
import type { ProjectMembersAdd, ProjectRole, WorkspaceRole } from "@nerve/api-client";
````
````old web/apps/web/core/components/project/project-roles.test.tsx
import { AddProjectMembersModal } from "./add-project-members-modal";
````
````new web/apps/web/core/components/project/project-roles.test.tsx
import { AddProjectMembersModal } from "./add-project-members-modal";
import { PROJECT_ROLES, canRemove, roleChoices, type MembershipCaller, type ShownMembership } from "./project-roles";
````
````old web/apps/web/core/components/project/project-roles.test.tsx
// selects would, and submits the modal's form. The caller is an admin of web; ann is a member of its workspace.
````
````new web/apps/web/core/components/project/project-roles.test.tsx
// selects would, and submits the modal's form. The caller is an admin of web; ann is a member of its workspace. Then
// which roles and removals the members page offers whom (M3 design 3.5).
````
````old web/apps/web/core/components/project/project-roles.test.tsx
  }),
}));
````
````new web/apps/web/core/components/project/project-roles.test.tsx
  }),
}));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => ({ getProjectById: () => web }) }));
vi.mock("react-router", () => ({ useNavigate: () => vi.fn() }));
````
````old web/apps/web/core/components/project/project-roles.test.tsx
// the workspace members' fakes build the account's store, which imports the tab's session
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
````
````new web/apps/web/core/components/project/project-roles.test.tsx
// the workspace members' fakes build the account's store, which imports the tab's session; a role's change is
// followed in it
vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: { state: { status: "signed-in", loginId: "x" } },
  publicClient: {},
}));
````
````old web/apps/web/core/components/project/project-roles.test.tsx
/** Each role a select offers, by its label, and the number nerve takes. */
const roles: { label: string; role: ProjectRole }[] = [
  { label: "Guest", role: 5 },
  { label: "Member", role: 15 },
  { label: "Admin", role: 20 },
````
````new web/apps/web/core/components/project/project-roles.test.tsx
/**
 * Each role a select offers: its name, as the add modal shows it; the key the members page shows it by (ROLE_DETAILS's,
 * translated); the number nerve takes.
 */
const roles: { label: string; key: string; role: ProjectRole }[] = [
  { label: "Guest", key: "role_details.guest.title", role: 5 },
  { label: "Member", key: "role_details.member.title", role: 15 },
  { label: "Admin", key: "role_details.admin.title", role: 20 },
````
````old web/apps/web/core/components/project/project-roles.test.tsx
  it.each(roles)("make ann $label with that role's number", ({ label, role }) => {
````
````new web/apps/web/core/components/project/project-roles.test.tsx
  it.each(roles)("make ann $label with that role's number", ({ key, role }) => {
````
````old web/apps/web/core/components/project/project-roles.test.tsx
        currentProjectRole={20}
````
````new web/apps/web/core/components/project/project-roles.test.tsx
        choices={PROJECT_ROLES}
````
````old web/apps/web/core/components/project/project-roles.test.tsx
    if (!roleSelect) throw new Error("the column showed no role select");
    pick(roleSelect, label);
````
````new web/apps/web/core/components/project/project-roles.test.tsx
    if (!roleSelect) throw new Error("the column showed no role select");
    pick(roleSelect, key);
````
````old web/apps/web/core/components/project/project-roles.test.tsx
  });
});

````
````new web/apps/web/core/components/project/project-roles.test.tsx
  });
});

/** A membership the page shows: its role, its member's role in the workspace, the caller's own or not. */
const shownAs = (role: ProjectRole, workspaceRole: WorkspaceRole, own = false): ShownMembership => ({
  own,
  role,
  workspaceRole,
});
const projectAdmin: MembershipCaller = { workspaceRole: 15, projectRole: 20 };
const workspaceAdmin: MembershipCaller = { workspaceRole: 20, projectRole: 20 };

describe("what the members page offers", () => {
  it.each<{
    who: string;
    caller: MembershipCaller;
    membership: ShownMembership;
    choices: ProjectRole[];
    removable: boolean;
  }>([
    {
      who: "a project admin, for a member: the roles below his own, and the removal",
      caller: projectAdmin,
      membership: shownAs(15, 15),
      choices: [5, 15],
      removable: true,
    },
    {
      who: "a project admin, for another admin: no role, the removal",
      caller: projectAdmin,
      membership: shownAs(20, 15),
      choices: [],
      removable: true,
    },
    {
      who: "a project admin, for himself: neither",
      caller: projectAdmin,
      membership: shownAs(20, 15, true),
      choices: [],
      removable: false,
    },
    {
      who: "a project admin, for a guest of the workspace: a guest's role, and the removal",
      caller: projectAdmin,
      membership: shownAs(5, 5),
      choices: [5],
      removable: true,
    },
    {
      who: "the workspace's admin, for another admin: every role, and the removal",
      caller: workspaceAdmin,
      membership: shownAs(20, 15),
      choices: [5, 15, 20],
      removable: true,
    },
    {
      who: "the workspace's admin, for himself: every role, no removal",
      caller: workspaceAdmin,
      membership: shownAs(20, 20, true),
      choices: [5, 15, 20],
      removable: false,
    },
    {
      who: "the workspace's admin who is the project's member, for its admin: every role, no removal",
      caller: { workspaceRole: 20, projectRole: 15 },
      membership: shownAs(20, 15),
      choices: [5, 15, 20],
      removable: false,
    },
    {
      who: "a project member, for a guest: neither",
      caller: { workspaceRole: 15, projectRole: 15 },
      membership: shownAs(5, 15),
      choices: [],
      removable: false,
    },
    {
      who: "the workspace's admin who is not the project's member: neither",
      caller: { workspaceRole: 20, projectRole: null },
      membership: shownAs(15, 15),
      choices: [],
      removable: false,
    },
  ])("$who", ({ caller, membership, choices, removable }) => {
    expect([roleChoices(caller, membership), canRemove(caller, membership)]).toEqual([choices, removable]);
  });
});

````

`web/apps/web/core/components/project/project-roles.ts`（整个文件，61 行）：

````whole web/apps/web/core/components/project/project-roles.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ProjectRole, WorkspaceRole } from "@nerve/api-client";

/**
 * A project's roles, the lowest first, as its role selects offer them, each labelled by ROLE: a select gives the
 * picked role's number, the ProjectRole nerve's ProjectMemberNew and ProjectMemberUpdate take (a string is refused).
 */
export const PROJECT_ROLES: ProjectRole[] = [5, 15, 20];

/** The caller as nerve decides his changes of a project's memberships: his role in its workspace, and in it. */
export type MembershipCaller = {
  workspaceRole: WorkspaceRole | undefined;
  projectRole: ProjectRole | null | undefined;
};

/**
 * A membership of the project as its members page shows it: whether it is the caller's own, its role, and its
 * member's role in the workspace.
 */
export type ShownMembership = { own: boolean; role: ProjectRole; workspaceRole: WorkspaceRole | undefined };

/** Whether role comes below the role than in PROJECT_ROLES' order, not by their numbers (as nerve's roleOrder). */
const isBelow = (role: ProjectRole, than: ProjectRole) => PROJECT_ROLES.indexOf(role) < PROJECT_ROLES.indexOf(than);

/**
 * The caller's role in the project when he changes its memberships at all, as nerve decides it (M3 design 3.5): its
 * admins, and its members who are the workspace's admins; undefined for anyone else.
 */
function managerRole({ workspaceRole, projectRole }: MembershipCaller): ProjectRole | undefined {
  if (projectRole === null || projectRole === undefined) return undefined;
  return projectRole === 20 || workspaceRole === 20 ? projectRole : undefined;
}

/**
 * The roles the caller may give a membership of the project, as nerve allows them (M3 design 3.5): none when he may
 * not change its role. One who is not the workspace's admin changes neither his own role nor one that is not below
 * his own, and gives no role that is not below his own; a workspace guest's role stays a guest's. The members page
 * offers these alone, so as not to offer what nerve refuses; hiding the others protects nothing (an admin may still
 * add a member as an admin, P5b spec §5).
 */
export function roleChoices(caller: MembershipCaller, membership: ShownMembership): ProjectRole[] {
  const callerRole = managerRole(caller);
  if (callerRole === undefined) return [];
  const assignable = membership.workspaceRole === 5 ? PROJECT_ROLES.filter((role) => role === 5) : PROJECT_ROLES;
  if (caller.workspaceRole === 20) return assignable;
  if (membership.own || !isBelow(membership.role, callerRole)) return [];
  return assignable.filter((role) => isBelow(role, callerRole));
}

/**
 * Whether the caller may remove a membership of the project, as nerve allows it (M3 design 3.5): not his own (he
 * leaves), nor one whose role is above his own role in the project, the workspace's admins neither.
 */
export function canRemove(caller: MembershipCaller, membership: ShownMembership): boolean {
  const callerRole = managerRole(caller);
  return callerRole !== undefined && !membership.own && !isBelow(callerRole, membership.role);
}
````

`web/apps/web/core/components/project/settings/use-project-membership-changes.test.ts`（新文件，116 行）：

````file web/apps/web/core/components/project/settings/use-project-membership-changes.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, settledYet, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf } from "@/store/project/fake-projects";
import { useProjectMembershipChanges } from "./use-project-membership-changes";

// What a project's pages send for each change of a membership, and what they do with nerve's answer (M3 design 7.1,
// 7.6, M1-P4): the hook runs as a plain function, with stand-ins for the stores' changes, which nerve answers when the
// test says, for the router's navigate, and for the closing of the dialog that sent a removal or the leaving. Its
// session is fake-tab.ts's.

const page = vi.hoisted(() => ({
  updateMemberRole: vi.fn(),
  removeMemberFromProject: vi.fn(),
  leaveProject: vi.fn(),
  navigate: vi.fn(),
  closeDialog: vi.fn(),
}));
const web = projectOf("WEB", "w-acme", { member_role: 20 });
vi.mock("react-router", () => ({ useNavigate: () => page.navigate }));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    getProjectById: (projectId: string) => (projectId === web.id ? web : undefined),
    leaveProject: page.leaveProject,
  }),
}));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    project: { updateMemberRole: page.updateMemberRole, removeMemberFromProject: page.removeMemberFromProject },
  }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const changes = () => useProjectMembershipChanges("acme", web.id);
/** Each change the pages make, by its store change: the role's, the removal's and the leaving's. */
const made = [
  { change: "a role", store: page.updateMemberRole, make: () => changes().changeRole("u-bob", 5) },
  { change: "a removal", store: page.removeMemberFromProject, make: () => changes().remove("u-bob", page.closeDialog) },
  { change: "the leaving", store: page.leaveProject, make: () => changes().leave(page.closeDialog) },
];

beforeEach(() => {
  signedIn();
  for (const store of [page.updateMemberRole, page.removeMemberFromProject, page.leaveProject]) {
    store.mockReset();
    store.mockResolvedValue(undefined);
  }
  page.navigate.mockReset();
  page.closeDialog.mockReset();
  toasts.length = 0;
});

describe("useProjectMembershipChanges", () => {
  it("sends a member's new role as its number, to the membership's project", async () => {
    await changes().changeRole("u-bob", 5);
    expect(page.updateMemberRole.mock.calls).toEqual([[web.id, "u-bob", 5]]);
    expect([page.navigate.mock.calls, toasts]).toEqual([[], []]);
  });

  it("removes a member of the project, and stays, its dialog closed once nerve has", async () => {
    const removal = heldChange<undefined>();
    page.removeMemberFromProject.mockReturnValueOnce(removal.sent);
    const removed = changes().remove("u-bob", page.closeDialog);
    expect(page.removeMemberFromProject.mock.calls).toEqual([[web.id, "u-bob"]]);
    expect(page.closeDialog).not.toHaveBeenCalled();
    removal.answer(undefined);
    await removed;
    expect([page.closeDialog.mock.calls.length, page.navigate.mock.calls, toasts]).toEqual([1, [], []]);
  });

  it("leaves the project, and closes its dialog and shows the workspace's projects only once nerve has made it, settling once they show", async () => {
    const leaving = heldChange<undefined>();
    page.leaveProject.mockReturnValueOnce(leaving.sent);
    const shown = heldChange<undefined>();
    page.navigate.mockReturnValueOnce(shown.sent);
    const left = changes().leave(page.closeDialog);
    const settled = settledYet(left);
    expect(page.leaveProject.mock.calls).toEqual([[web]]);
    expect([page.closeDialog.mock.calls, page.navigate.mock.calls]).toEqual([[], []]);
    leaving.answer(undefined);
    await vi.waitFor(() => expect(page.navigate.mock.calls).toEqual([["/acme/projects"]]));
    expect([page.closeDialog.mock.calls.length, settled()]).toEqual([1, false]);
    shown.answer(undefined);
    await left;
  });

  it.each(made)("shows nerve's reason when it refuses $change, and stays, its dialog open", async ({ store, make }) => {
    store.mockRejectedValueOnce(refusal(409, "project.sole_admin"));
    await make();
    expect([page.closeDialog.mock.calls, page.navigate.mock.calls]).toEqual([[], []]);
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.project_sole_admin" }]);
  });

  describe.each(made)("$change", ({ store, make }) => {
    it.each(lateSettlings)(
      "does nothing on the page when it $settles after another tab moved this one to another account",
      async ({ settle }) => {
        const change = heldChange<undefined>();
        store.mockReturnValueOnce(change.sent);
        const making = make();
        switchAccount();
        settle(change);
        await making;
        expect([page.closeDialog.mock.calls, page.navigate.mock.calls, toasts]).toEqual([[], [], []]);
      }
    );
  });
});
````

`web/apps/web/core/components/project/settings/use-project-membership-changes.ts`（新文件，53 行）：

````file web/apps/web/core/components/project/settings/use-project-membership-changes.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useNavigate } from "react-router";
import type { ProjectRole } from "@nerve/api-client";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useProject } from "@/hooks/store/use-project";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";

/**
 * The changes of the memberships of the project of projectId, in the workspace of workspaceSlug, that its pages make
 * (M3 design 7.6): a member's role, a member's removal, and the caller's leaving. Each shows nerve's refusal as its
 * reason; the page follows each only in the session it was sent in (M3 design 7.1): once another tab has moved this
 * one to another account, the page is that account's, and says nothing of the change. A dialog that sent the removal
 * or the leaving gives done, its closing, which follows nerve's having made it alone: one nerve refuses leaves it open.
 */
export function useProjectMembershipChanges(workspaceSlug: string, projectId: string) {
  const navigate = useNavigate();
  const { getProjectById, leaveProject } = useProject();
  const {
    project: { updateMemberRole, removeMemberFromProject },
  } = useMember();
  const failed = useRefusalToast();

  return {
    /** Gives the member of userId the role: its number, as nerve's ProjectMemberUpdate takes it. */
    changeRole: (userId: string, role: ProjectRole) =>
      followInSession(() => updateMemberRole(projectId, userId, role), { failed }),
    /** Ends the membership of the member of userId; once nerve has, done. */
    remove: (userId: string, done?: () => void) =>
      followInSession(() => removeMemberFromProject(projectId, userId), { done, failed }),
    /**
     * Ends the caller's own membership; once nerve has, and only then, done, and the workspace's projects show (M1-P4:
     * a leaving nerve refuses, its only admin's, leaves him where he was, its member still). Settles once they show.
     */
    leave: async (done?: () => void) => {
      const project = getProjectById(projectId);
      if (!project) return;
      await followInSession(() => leaveProject(project), {
        done: () => {
          done?.();
          return navigate(`/${workspaceSlug}/projects`);
        },
        failed,
      });
    },
  };
}
````

- [ ] **Step 2: 成员页、确认框、离开弹窗和上限**

`web/apps/web/core/components/project/confirm-project-member-remove.tsx`（修改，7 处）：

````old web/apps/web/core/components/project/confirm-project-member-remove.tsx
  data: Partial<MemberUser>;
````
````new web/apps/web/core/components/project/confirm-project-member-remove.tsx
  data: Partial<MemberUser>;
  /** Makes the removal or the leaving; settles once the page has followed it: its follow-up of success closes this. */
````
````old web/apps/web/core/components/project/confirm-project-member-remove.tsx
    setIsDeleteLoading(true);

````
````new web/apps/web/core/components/project/confirm-project-member-remove.tsx
    setIsDeleteLoading(true);
````
````old web/apps/web/core/components/project/confirm-project-member-remove.tsx

    handleClose();
````
````new web/apps/web/core/components/project/confirm-project-member-remove.tsx
    setIsDeleteLoading(false);
````
````old web/apps/web/core/components/project/confirm-project-member-remove.tsx
  const currentProjectDetails = getProjectById(projectId);

````
````new web/apps/web/core/components/project/confirm-project-member-remove.tsx
  const currentProjectDetails = getProjectById(projectId);

  // While the removal or the leaving is out the dialog cannot be dismissed (Cancel, Escape, the backdrop): it closes
  // once nerve has made the change, and no dialog opened again offers it while the request is out. One nerve refuses
  // leaves it open, its buttons enabled again, the reason in the refusal's toast.
````
````old web/apps/web/core/components/project/confirm-project-member-remove.tsx
    <ModalCore isOpen={isOpen} handleClose={handleClose} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
````
````new web/apps/web/core/components/project/confirm-project-member-remove.tsx
    <ModalCore
      isOpen={isOpen}
      handleClose={isDeleteLoading ? undefined : handleClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.XXL}
    >
````
````old web/apps/web/core/components/project/confirm-project-member-remove.tsx
        <Button variant="secondary" size="lg" onClick={handleClose}>
````
````new web/apps/web/core/components/project/confirm-project-member-remove.tsx
        <Button variant="secondary" size="lg" onClick={handleClose} disabled={isDeleteLoading}>
````
````old web/apps/web/core/components/project/confirm-project-member-remove.tsx
        <Button variant="error-fill" size="lg" tabIndex={1} onClick={handleDeletion} loading={isDeleteLoading}>
````
````new web/apps/web/core/components/project/confirm-project-member-remove.tsx
        <Button variant="error-fill" size="lg" onClick={handleDeletion} loading={isDeleteLoading}>
````

`web/apps/web/core/components/project/leave-project-modal.tsx`（修改，8 处）：

````old web/apps/web/core/components/project/leave-project-modal.tsx
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useParams, useNavigate } from "react-router";
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
import { useParams } from "react-router";
// components
import { useProjectMembershipChanges } from "@/components/project/settings/use-project-membership-changes";
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
  // router
  const navigate = useNavigate();
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
  // router
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
  const { leaveProject } = useProject();
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
  const { leave } = useProjectMembershipChanges(workspaceSlug ?? "", project.id);
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
  const onSubmit = async (data: any) => {
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
  const onSubmit = async (data: FormData) => {
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
    if (data) {
      if (data.projectName === project?.name) {
        if (data.confirmLeave === "Leave Project") {
          navigate(`/${workspaceSlug}/projects`);
          return leaveProject(project)
            .then(() => handleClose())
            .catch((_err) => {
              setToast({
                type: TOAST_TYPE.ERROR,
                title: "Error!",
                message: "Something went wrong please try again later.",
              });
            });
        } else {
          setToast({
            type: TOAST_TYPE.ERROR,
            title: "Error!",
            message: "Please confirm leaving the project by typing the 'Leave Project'.",
          });
        }
      } else {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: "Please enter the project name as shown in the description.",
        });
      }
    } else {
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
    if (data.projectName !== project.name) {
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
        message: "Please fill all fields.",
      });
    }
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
        message: "Please enter the project name as shown in the description.",
      });
      return;
    }
    if (data.confirmLeave !== "Leave Project") {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Please confirm leaving the project by typing the 'Leave Project'.",
      });
      return;
    }
    // the leaving first: once nerve has made it the modal closes and the workspace's projects show; one it refuses
    // leaves the modal open
    await leave(handleClose);
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
    <ModalCore isOpen={isOpen} handleClose={handleClose} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
    <ModalCore
      isOpen={isOpen}
      handleClose={isSubmitting ? undefined : handleClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.XXL}
    >
````
````old web/apps/web/core/components/project/leave-project-modal.tsx
          <Button variant="secondary" size="lg" onClick={handleClose}>
````
````new web/apps/web/core/components/project/leave-project-modal.tsx
          <Button variant="secondary" size="lg" onClick={handleClose} disabled={isSubmitting}>
````

`web/apps/web/core/components/project/member-list-item.tsx`（整个文件，66 行）：

````whole web/apps/web/core/components/project/member-list-item.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { Table } from "@nerve/ui";
// hooks
import { useUser } from "@/hooks/store/user";
// components
import { useProjectColumns } from "@/components/projects/settings/useProjectColumns";
// store
import type { IProjectMemberDetails } from "@/store/member/project/project-member.store";
// local imports
import { ConfirmProjectMemberRemove } from "./confirm-project-member-remove";
import { useProjectMembershipChanges } from "./settings/use-project-membership-changes";

type Props = {
  memberDetails: (IProjectMemberDetails | null)[];
  projectId: string;
  workspaceSlug: string;
};

export const ProjectMemberListItem = observer(function ProjectMemberListItem(props: Props) {
  const { memberDetails, projectId, workspaceSlug } = props;
  // store hooks
  const { data: currentUser } = useUser();
  const { remove, leave } = useProjectMembershipChanges(workspaceSlug, projectId);
  // helper hooks
  const { columns, removeMemberModal, setRemoveMemberModal } = useProjectColumns({
    projectId,
    workspaceSlug,
  });

  // the caller's own membership he leaves; another's he removes; the dialog that asked closes once nerve has done it
  const handleRemove = (memberId: string) => {
    const closeDialog = () => setRemoveMemberModal(null);
    return memberId === currentUser?.id ? leave(closeDialog) : remove(memberId, closeDialog);
  };

  if (!memberDetails) return null;
  return (
    <>
      {removeMemberModal && (
        <ConfirmProjectMemberRemove
          isOpen={removeMemberModal !== null}
          onClose={() => setRemoveMemberModal(null)}
          data={{ id: removeMemberModal.member.id, display_name: removeMemberModal.member.display_name || "" }}
          onSubmit={() => handleRemove(removeMemberModal.member.id)}
        />
      )}
      <Table
        columns={columns}
        data={memberDetails.filter((member): member is IProjectMemberDetails => member !== null)}
        keyExtractor={(rowData) => rowData?.member.id ?? ""}
        tHeadClassName="border-b border-subtle"
        thClassName="text-left font-medium divide-x-0 text-placeholder"
        tBodyClassName="divide-y-0"
        tBodyTrClassName="divide-x-0 p-4 h-[40px] text-secondary"
        tHeadTrClassName="divide-x-0"
      />
    </>
  );
});
````

`web/apps/web/core/components/project/settings/member-columns.tsx`（整个文件，125 行）：

````whole web/apps/web/core/components/project/settings/member-columns.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { Link } from "react-router";
import { CircleMinus } from "lucide-react";
import { Disclosure } from "@headlessui/react";
// nerve imports
import { ROLE_DETAILS } from "@nerve/constants";
import type { ProjectRole } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { CustomMenu, CustomSelect } from "@nerve/ui";
import { getFileURL } from "@nerve/utils";
import type { IProjectMemberDetails } from "@/store/member/project/project-member.store";
// local imports
import { useProjectMembershipChanges } from "./use-project-membership-changes";

type NameProps = {
  rowData: IProjectMemberDetails;
  workspaceSlug: string;
  /** Whether the membership is the caller's own, which he may leave. */
  own: boolean;
  /** Whether the caller may remove the membership (canRemove). */
  removable: boolean;
  setRemoveMemberModal: (rowData: IProjectMemberDetails) => void;
};

type AccountTypeProps = {
  rowData: IProjectMemberDetails;
  /** The roles the caller may give the membership (roleChoices): none shows its role alone. */
  choices: ProjectRole[];
  workspaceSlug: string;
  projectId: string;
};

export function NameColumn(props: NameProps) {
  const { rowData, workspaceSlug, own, removable, setRemoveMemberModal } = props;
  // derived values
  const { avatar_url, display_name, email, first_name, id, last_name } = rowData.member;

  return (
    <Disclosure>
      {() => (
        <div className="group relative">
          <div className="flex w-72 items-center gap-2">
            <div className="flex flex-1 items-center gap-x-2 gap-y-2">
              {avatar_url && avatar_url.trim() !== "" ? (
                <Link to={`/${workspaceSlug}/profile/${id}`}>
                  <span className="relative flex size-6 items-center justify-center rounded-full text-on-color capitalize">
                    <img
                      src={getFileURL(avatar_url)}
                      className="absolute top-0 left-0 h-full w-full rounded-full object-cover"
                      alt={display_name || (email ?? undefined)}
                    />
                  </span>
                </Link>
              ) : (
                <Link to={`/${workspaceSlug}/profile/${id}`}>
                  <span className="relative flex size-6 items-center justify-center rounded-full bg-layer-3 text-11 text-on-color capitalize">
                    {(email ?? display_name ?? "?")[0]}
                  </span>
                </Link>
              )}
              {first_name} {last_name}
            </div>
            {(own || removable) && (
              <CustomMenu
                ellipsis
                buttonClassName="p-0.5 opacity-0 group-hover:opacity-100 transition-opacity"
                optionsClassName="p-1.5"
                placement="bottom-end"
              >
                <CustomMenu.MenuItem onClick={() => setRemoveMemberModal(rowData)}>
                  <div className="flex items-center gap-x-1 font-medium text-danger-primary">
                    <CircleMinus className="size-3.5 flex-shrink-0" />
                    {own ? "Leave " : "Remove "}
                  </div>
                </CustomMenu.MenuItem>
              </CustomMenu>
            )}
          </div>
        </div>
      )}
    </Disclosure>
  );
}

export const AccountTypeColumn = observer(function AccountTypeColumn(props: AccountTypeProps) {
  const { rowData, choices, projectId, workspaceSlug } = props;
  // store hooks
  const { changeRole } = useProjectMembershipChanges(workspaceSlug, projectId);
  // translation
  const { t } = useTranslation();
  // derived values
  const roleLabel = t(ROLE_DETAILS[rowData.role].i18n_title);

  return choices.length > 0 ? (
    <CustomSelect
      value={rowData.role}
      // the select gives the picked option's value: a role's number, of the choices
      onChange={(role: ProjectRole) => void changeRole(rowData.member.id, role)}
      label={
        <div className="flex">
          <span>{roleLabel}</span>
        </div>
      }
      buttonClassName="!px-0 !justify-start hover:bg-surface-1 border-none"
      className="w-32 rounded-md p-0"
      input
    >
      {choices.map((role) => (
        <CustomSelect.Option key={role} value={role}>
          {t(ROLE_DETAILS[role].i18n_title)}
        </CustomSelect.Option>
      ))}
    </CustomSelect>
  ) : (
    <div className="flex w-32">
      <span>{roleLabel}</span>
    </div>
  );
});
````

`web/apps/web/core/components/projects/settings/useProjectColumns.tsx`（修改，8 处）：

````old web/apps/web/core/components/projects/settings/useProjectColumns.tsx
// nerve imports
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
````
````new web/apps/web/core/components/projects/settings/useProjectColumns.tsx
// nerve imports
````
````old web/apps/web/core/components/projects/settings/useProjectColumns.tsx
import { MemberHeaderColumn } from "@/components/project/member-header-column";
````
````new web/apps/web/core/components/projects/settings/useProjectColumns.tsx
import { MemberHeaderColumn } from "@/components/project/member-header-column";
import {
  canRemove,
  roleChoices,
  type MembershipCaller,
  type ShownMembership,
} from "@/components/project/project-roles";
````
````old web/apps/web/core/components/projects/settings/useProjectColumns.tsx
import { useMember } from "@/hooks/store/use-member";
````
````new web/apps/web/core/components/projects/settings/useProjectColumns.tsx
import { useMember } from "@/hooks/store/use-member";
import { useProject } from "@/hooks/store/use-project";
````
````old web/apps/web/core/components/projects/settings/useProjectColumns.tsx
  const { allowPermissions, getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
````
````new web/apps/web/core/components/projects/settings/useProjectColumns.tsx
  const { getWorkspaceRoleByWorkspaceSlug } = useUserPermissions();
  const { getProjectById } = useProject();
````
````old web/apps/web/core/components/projects/settings/useProjectColumns.tsx
      filters: { getFilters, updateFilters },
    },
````
````new web/apps/web/core/components/projects/settings/useProjectColumns.tsx
      filters: { getFilters, updateFilters },
    },
    workspace: { getWorkspaceMemberDetails },
````
````old web/apps/web/core/components/projects/settings/useProjectColumns.tsx
  // derived values
  const isAdmin = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.PROJECT, workspaceSlug, projectId);
  const currentProjectRole =
    getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId) ?? EUserPermissions.GUEST;
````
````new web/apps/web/core/components/projects/settings/useProjectColumns.tsx
  // derived values: the caller and each membership, as nerve decides a change of a membership (M3 design 3.5)
  const caller: MembershipCaller = {
    workspaceRole: getWorkspaceRoleByWorkspaceSlug(workspaceSlug),
    projectRole: getProjectById(projectId)?.member_role,
  };
  const shown = (rowData: IProjectMemberDetails): ShownMembership => ({
    own: rowData.member.id === currentUser?.id,
    role: rowData.role,
    workspaceRole: getWorkspaceMemberDetails(rowData.member.id)?.role,
  });
````
````old web/apps/web/core/components/projects/settings/useProjectColumns.tsx
          isAdmin={isAdmin}
          currentUser={currentUser}
````
````new web/apps/web/core/components/projects/settings/useProjectColumns.tsx
          own={shown(rowData).own}
          removable={canRemove(caller, shown(rowData))}
````
````old web/apps/web/core/components/projects/settings/useProjectColumns.tsx
          currentProjectRole={currentProjectRole}
````
````new web/apps/web/core/components/projects/settings/useProjectColumns.tsx
          choices={roleChoices(caller, shown(rowData))}
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 350",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 349",
````

- [ ] **Step 3: 静态检查的范围和离开的关键词规则**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/components/automation/select-month-modal.test.tsx"
````
````new .oxlintrc.json
        "web/apps/web/core/components/automation/select-month-modal.test.tsx",
        "web/apps/web/core/components/project/settings/use-project-membership-changes.ts",
        "web/apps/web/core/components/project/settings/use-project-membership-changes.test.ts",
        "web/apps/web/core/components/project/project-roles.ts",
        "web/apps/web/core/components/project/project-roles.test.tsx"
````

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
      "id": "project-leave",
      "phase": "M3/P10",
      "why": "页面离开项目只有一处：useProjectMembershipChanges 的 leave（web/apps/web/core/components/project/settings/use-project-membership-changes.ts），只在发出它的会话里跟进，nerve 做完离开之后才显示工作区的项目，拒绝时给出 nerve 的原因（M1-P4 交接，M3/P10 spec 2.7）；成员页和离开对话框都经它。别处再调 store 的 leaveProject 就是又一份，顺序和拒绝要再写一遍。store 自己的定义不算；测试不算",
      "files": {
        "source": "^web/apps/web/(?:app|core)/(?!components/project/settings/use-project-membership-changes\\.ts$|store/project/project\\.store\\.ts$)(?!.*\\.test\\.[jt]sx?$).*\\.[jt]sx?$",
        "flags": ""
      },
      "content": {
        "source": "\\bleaveProject\\b",
        "flags": ""
      },
      "samples": {
        "hit": ["  const { getProjectById, leaveProject } = useProject();", "      return leaveProject(project)"],
        "miss": [
          "  const { remove, leave } = useProjectMembershipChanges(workspaceSlug, projectId);",
          "  const [leaveProjectModalOpen, setLeaveProjectModalOpen] = useState(false);"
        ],
        "files": {
          "hit": [
            "web/apps/web/core/components/project/leave-project-modal.tsx",
            "web/apps/web/core/components/project/member-list-item.tsx"
          ],
          "miss": [
            "web/apps/web/core/components/project/settings/use-project-membership-changes.ts",
            "web/apps/web/core/components/project/settings/use-project-membership-changes.test.ts",
            "web/apps/web/core/store/project/project.store.ts"
          ]
        }
      }
    }
  ],
````

- [ ] **Step 4: 端到端**

`e2e/fixtures/project-pages.ts`（新文件，38 行）：

````file e2e/fixtures/project-pages.ts
import type { Page, Response } from "@playwright/test";

import { answerTo } from "./settings-pages";
import { memberRow } from "./workspace-pages";

// A project's pages (M3 design 7.6), as a person uses them.

/** A request the page sends: its method and its path, as answerTo and holdAnswer take them. */
type Sent = { method: string; path: string };

/** The request that removes the project membership of id. */
export function removalOf(id: string): Sent {
  return { method: "DELETE", path: `/api/v0/project-members/${id}` };
}

/** The request that ends the caller's membership of the project of id. */
export function leavingOf(id: string): Sent {
  return { method: "POST", path: `/api/v0/projects/${id}/leave` };
}

/**
 * Ends a membership from a project's members page, which page shows: opens the menu of the row of email, picks its
 * entry (Leave on the caller's own row, Remove on another's), and confirms it in the dialog that asks. Resolves with
 * nerve's answer to the request the page sends.
 */
export async function endProjectMembership(
  page: Page,
  email: string,
  entry: "Leave" | "Remove",
  request: Sent
): Promise<Response> {
  // the row's menu is its first button: the role's select, when there is one, comes after it
  await memberRow(page, email).locator("button").first().click();
  await page.getByRole("menuitem", { name: entry }).click();
  return answerTo(page, request.method, request.path, () =>
    page.getByRole("dialog").getByRole("button", { name: entry, exact: true }).click()
  );
}
````

`e2e/stories/project/p5-project-members.spec.ts`（修改，3 处）：

````old e2e/stories/project/p5-project-members.spec.ts
import { bearer, newAccount } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
````
````new e2e/stories/project/p5-project-members.spec.ts
import { accountId, bearer, emailFor, newAccount } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import {
  answerTo,
  closedByEscape,
  enabledWithin,
  holdAnswer,
  registerOnboarded,
  sentTo,
} from "../../fixtures/settings-pages";
import { endProjectMembership, leavingOf, removalOf } from "../../fixtures/project-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser, memberRow } from "../../fixtures/workspace-pages";
````
````old e2e/stories/project/p5-project-members.spec.ts
// P5, a project's members (M3 design 2, 3.5, 3.7): adding them, changing a
// role, removing a member, leaving. The page version comes with the
// project's members page (P10).
````
````new e2e/stories/project/p5-project-members.spec.ts
// P5, a project's members (M3 design 2, 3.5, 3.7, 7.6): adding them, changing a role, removing a member, leaving.
````
````old e2e/stories/project/p5-project-members.spec.ts
  );
});

````
````new e2e/stories/project/p5-project-members.spec.ts
  );
});

test("P5 (page): a project admin who is no workspace admin is offered only the roles below his own, none for another admin; he makes a member a guest, removes the other admin and a guest, the dialog held until nerve answers, and, its only admin now, is told why he may not leave, its dialog open; a member leaves, and the workspace's projects show once nerve has made it, not before; the guest leaves by the sidebar, its modal open after a refusal and held too", async ({
  api,
  baseURL,
  browser,
  db,
  signedInPage,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  // pat, bob and ann open the pages; max and gus do not
  const [pat, bob, ann] = await Promise.all(
    ["pat", "bob", "ann"].map(async (label) => {
      const email = emailFor(testInfo, label);
      const tokens = await registerOnboarded(api, email);
      await inviteAndAccept(api, admin.token, slug, { email, token: tokens.access_token }, 15);
      return { email, tokens, id: await accountId(api, tokens.access_token) };
    })
  );
  const [max, gus] = await Promise.all(["max", "gus"].map((label) => newAccount(api, testInfo, label)));
  if (!pat || !bob || !ann || !max || !gus) throw new Error("the accounts were not registered");
  await inviteAndAccept(api, admin.token, slug, max, 15);
  await inviteAndAccept(api, admin.token, slug, gus, 5);
  // pat, a member of acme, makes Web, its admin; max its other admin, ann and bob its members, gus its guest.
  const web = await createProject(api, pat.tokens.access_token, slug, { name: "Web", identifier: "WEB" });
  const [maxs, anns, , guss] = await addProjectMembers(api, pat.tokens.access_token, web.id, [
    { member_id: max.id, role: 20 },
    { member_id: ann.id, role: 15 },
    { member_id: bob.id, role: 15 },
    { member_id: gus.id, role: 5 },
  ]);
  if (!maxs || !anns || !guss) throw new Error("the members were not added");
  const members = `/${slug}/settings/projects/${web.id}/members`;
  const leaving = leavingOf(web.id);

  const page = await signedInPage(pat.tokens);
  const watch = await watchPage(page);
  await page.goto(members);
  // His own row and max's, another admin's: the role as text, nothing to pick.
  await expect(memberRow(page, pat.email)).toContainText("Admin");
  await expect(memberRow(page, max.email)).toContainText("Admin");
  await expect(memberRow(page, max.email).getByRole("button", { name: "Admin" })).toHaveCount(0);
  // ann, a member: the roles below his own, not an admin's; the page sends the role's number.
  await memberRow(page, ann.email).getByRole("button", { name: "Member", exact: true }).click();
  await expect(page.getByRole("option")).toHaveText(["Guest", "Member"]);
  const demoted = await sentTo(page, "PATCH", `/api/v0/project-members/${anns.id}`, () =>
    page.getByRole("option", { name: "Guest", exact: true }).click()
  );
  expect([demoted.answer.status(), demoted.body]).toEqual([200, { role: 5 }]);
  await expect(memberRow(page, ann.email).getByRole("button", { name: "Guest", exact: true })).toBeVisible();

  // max is removed; then gus, while the removal is out the dialog that asked cannot be dismissed.
  expect((await endProjectMembership(page, max.email, "Remove", removalOf(maxs.id))).status()).toBe(204);
  await expect(memberRow(page, max.email)).toHaveCount(0);
  const release = await holdAnswer(page, "DELETE", removalOf(guss.id).path);
  const removed = endProjectMembership(page, gus.email, "Remove", removalOf(guss.id));
  await expect(page.getByRole("dialog").getByRole("button", { name: "Cancel" })).toBeDisabled();
  expect(await closedByEscape(page)).toBe(false);
  await release();
  expect((await removed).status()).toBe(204);
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(memberRow(page, gus.email)).toHaveCount(0);

  // His leaving, as the only admin: nerve refuses it, and the page says why and stays, the dialog open.
  expect((await endProjectMembership(page, pat.email, "Leave", leaving)).status()).toBe(409);
  await expect(page.getByText("The project would be left without an admin")).toBeVisible();
  await expect(page.getByRole("dialog").getByRole("button", { name: "Leave", exact: true })).toBeEnabled();
  await expect(page).toHaveURL(members);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`409 ${leaving.method} ${leaving.path}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING],
    errors: ["Failed to load resource: the server responded with a status of 409 (Conflict)"],
  });

  // bob leaves: the page stays while nerve has not answered, the dialog busy; then the workspace's projects show.
  const theMember = await anotherBrowser(browser, baseURL ?? "", bob.tokens);
  const memberWatch = await watchPage(theMember.page);
  await theMember.page.goto(members);
  const releaseLeaving = await holdAnswer(theMember.page, leaving.method, leaving.path);
  const left = endProjectMembership(theMember.page, bob.email, "Leave", leaving);
  const busy = theMember.page.getByRole("dialog").getByRole("button", { name: "Leaving..." });
  expect(await enabledWithin(busy)).toBe(false);
  await expect(theMember.page).toHaveURL(members);
  await releaseLeaving();
  expect((await left).status()).toBe(204);
  await expect(theMember.page).toHaveURL(`/${slug}/projects`);
  expect([memberWatch.apiFailures, memberWatch.oldApiRequests, memberWatch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(theMember.page, memberWatch, { warnings: [EMOJI_CHECK_WARNING] });
  await theMember.close();

  // ann, a guest now, leaves by the sidebar, which offers it to guests alone: its modal asks Web's name and "Leave
  // Project". pat has ended her membership meanwhile: nerve refuses the leaving, and the modal stays, to try again.
  // He adds her back; the modal cannot be closed while the leaving is out; once nerve has made it, Web leaves the
  // sidebar.
  const theGuest = await anotherBrowser(browser, baseURL ?? "", ann.tokens);
  const guestWatch = await watchPage(theGuest.page);
  await theGuest.page.goto(`/${slug}/projects`);
  const sidebar = theGuest.page.getByRole("complementary", { name: "Main sidebar" });
  await sidebar.getByText("Web", { exact: true }).hover();
  await sidebar.getByRole("button", { name: "Toggle quick actions menu" }).last().click();
  await theGuest.page.getByRole("menuitem", { name: "Leave project" }).click();
  await theGuest.page.locator("#projectName").fill("Web");
  await theGuest.page.locator("#confirmLeave").fill("Leave Project");
  expect((await writes(api, web.id).remove(pat.tokens.access_token, anns.id)).status).toBe(204);
  const leaveProject = theGuest.page.getByRole("dialog").getByRole("button", { name: "Leave Project" });
  expect((await answerTo(theGuest.page, leaving.method, leaving.path, () => leaveProject.click())).status()).toBe(403);
  await expect(leaveProject).toBeEnabled();
  await addProjectMembers(api, pat.tokens.access_token, web.id, [{ member_id: ann.id, role: 5 }]);
  const releaseGuest = await holdAnswer(theGuest.page, leaving.method, leaving.path);
  const guestLeft = answerTo(theGuest.page, leaving.method, leaving.path, () => leaveProject.click());
  await expect(theGuest.page.getByRole("dialog").getByRole("button", { name: "Cancel" })).toBeDisabled();
  expect(await closedByEscape(theGuest.page)).toBe(false);
  await releaseGuest();
  expect((await guestLeft).status()).toBe(204);
  await expect(sidebar.getByText("Web", { exact: true })).toHaveCount(0);
  expect([guestWatch.apiFailures, guestWatch.oldApiRequests, guestWatch.pageErrors]).toEqual([
    [`403 ${leaving.method} ${leaving.path}`],
    [],
    [],
  ]);
  await expectQuietConsole(theGuest.page, guestWatch, {
    warnings: [EMOJI_CHECK_WARNING],
    errors: ["Failed to load resource: the server responded with a status of 403 (Forbidden)"],
  });
  await theGuest.close();

  const row = (email: string, role: number, is_active: boolean, by: string): MemberRow => ({
    email,
    role,
    is_active,
    by,
    sort_order: 65535,
    settings_by: pat.email,
  });
  await expectMembers(db, web.id, [
    row(pat.email, 20, true, pat.email),
    row(max.email, 20, false, pat.email),
    row(ann.email, 5, false, ann.email),
    row(bob.email, 15, false, bob.email),
    row(gus.email, 5, false, pat.email),
  ]);
});

````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 68 条规则、3 个例外，没有命中（`project-leave` 是新的）；web 的 oxlint 349 条，等于新的上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 103 个全部通过。

- [ ] **Step 6: 提交**

```bash
git add .oxlintrc.json e2e/fixtures/project-pages.ts e2e/stories/project/p5-project-members.spec.ts tools/keywords.json web/apps/web/core/components/project/confirm-project-member-remove.tsx web/apps/web/core/components/project/leave-project-modal.tsx web/apps/web/core/components/project/member-list-item.tsx web/apps/web/core/components/project/project-roles.test.tsx web/apps/web/core/components/project/project-roles.ts web/apps/web/core/components/project/settings/member-columns.tsx web/apps/web/core/components/project/settings/use-project-membership-changes.test.ts web/apps/web/core/components/project/settings/use-project-membership-changes.ts web/apps/web/core/components/projects/settings/useProjectColumns.tsx web/apps/web/package.json
```
```bash
git commit -m "feat(M3/P10): the members page offers what nerve allows, follows each change in its session, and leaves once nerve has

useProjectMembershipChanges makes a role's change, a removal and the
caller's leaving, each followed only in the session it was sent in and
refused with nerve's reason; leaving shows the workspace's projects once
nerve has made it, not before (M1-P4). The page offers the roles and the
removals nerve's rules allow the caller (roleChoices, canRemove), the
caller's own row offers leaving, a role is named as the workspace's
pages name it, and the columns hold no form, console.log or err.error. The
confirmation and the leave modal cannot be dismissed while their request
is out, and close in the follow-up of success alone: a refusal leaves
them open, the reason in the shared refusal toast. A keyword rule keeps
the leaving in its one place. P5's page version.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p10.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T7.1` | 离开在发出时就显示工作区的项目列表，不等 nerve 做完 | `use-project-membership-changes.test.ts`、故事 P5 | vitest；端到端 |
| `T7.2` | 移出被拒绝时什么都不说 | `use-project-membership-changes.test.ts` | vitest |
| `T7.3` | 改角色不论会话都跟进 | `use-project-membership-changes.test.ts` | vitest |
| `T7.4` | 工作区管理员也受相对规则的限制 | `project-roles.test.tsx` | vitest |
| `T7.5` | 项目管理员可以选管理员 | `project-roles.test.tsx`、故事 P5 | vitest；端到端 |
| `T7.6` | 项目管理员可以改另一位管理员的角色 | `project-roles.test.tsx`、故事 P5 | vitest；端到端 |
| `T7.7` | 移出在途时 Escape 关掉确认框 | 故事 P5 | 端到端 |
| `T7.8` | 移出在途时确认框的取消按钮仍可用 | 故事 P5 | 端到端 |
| `T7.9` | 离开在途时 Escape 关掉离开的弹窗 | 故事 P5 | 端到端 |
| `T7.10` | 离开在途时离开弹窗的取消按钮仍可用 | 故事 P5 | 端到端 |
| `T7.11` | 自己那一行的"离开"是移出，别人那一行的"移出"是离开 | 故事 P5 | 端到端 |
| `T7.12` | 在成员关系修改的 hook 里写一个非空断言 | oxlint（`check:lint`） | 静态 |
| `T7.13` | 离开弹窗直接调 store 的 leaveProject：第二条离开的路 | 关键词守卫 | 静态 |
| `T7.14` | 成员页的角色下拉框用英文的 ROLE 称呼角色 | `project-roles.test.tsx` | vitest |
| `T7.15` | 离开弹窗在 nerve 回答之后就关上，拒绝也关 | 故事 P5 | 端到端 |
| `T7.16` | 成员页的确认框在 nerve 回答之后就关上，拒绝也关 | 故事 P5 | 端到端 |
| `T7.17` | nerve 移出之后确认框不关 | `use-project-membership-changes.test.ts`、故事 P5 | vitest；端到端 |

---

### Task 8: 添加成员和成员的默认值；P5、P3 的页面版本（二）

**Files:**
- Create: `web/apps/web/core/components/project/add-project-members-modal.test.tsx`
- Modify: `.oxlintrc.json`、`e2e/fixtures/api.ts`、`e2e/fixtures/assert/project.ts`、`e2e/fixtures/project-pages.ts`、`e2e/stories/project/p3-project-settings.spec.ts`、`e2e/stories/project/p5-project-members.spec.ts`、`web/apps/web/core/components/project/add-project-members-modal.tsx`、`web/apps/web/core/components/project/member-list.tsx`、`web/apps/web/core/components/project/project-roles.test.tsx`、`web/apps/web/core/components/project/project-roles.ts`、`web/apps/web/core/components/project/project-settings-member-defaults.test.tsx`、`web/apps/web/core/components/project/project-settings-member-defaults.tsx`、`web/apps/web/core/lib/fake-controls.ts`

**Interfaces:**
- Produces（spec 2.8；M3 设计 2 的 P3、P5，3.5、3.19，7.1、7.6；P8b spec 第 5 节 P10 一行：添加成员的错误只进控制台）：
  - `project-roles.ts` 的 `addableRoles(workspaceRole)`：添加时 nerve 允许的角色（工作区管理员只能是管理员，工作区访客只能是访客，成员任意；还没选人时任意）。添加没有相对规则：调用者不论角色都按它提供（P5b spec 第 5 节：隐藏不保护任何东西）。
  - `AddProjectMembersModal`：候选是工作区的有效成员中还不是项目成员的（原来也列出成员关系已结束的人）；每行的角色按 `addableRoles`（原来还按调用者的项目角色筛），名字同成员页（`ROLE_DETAILS`）；成员的选择用 `customButton` 的 `<span>`（不在按钮里再放按钮）；提交经 `followInSession`（生成的 `ProjectMembersAdd`）：成功时关闭、提示，被拒绝时经 `useRefusalToast` 说原因、表单留着（原来只有 `console.error`，表单随之清空）；在途时关不掉（取消禁用、`handleClose` 为空）。不再收 `workspaceSlug`（`member-list.tsx` 不传）。
  - `ProjectSettingsMemberDefaults`（整个文件）：负责人、默认负责人（项目中不是访客的成员）、访客可见全部工作项，显示 nerve 所持的值（`getProjectById(projectId)`），一次改一个字段（`ProjectUpdate` 的部分修改），经 `followInSession`：成功提示，被拒绝时说原因；原来的 `useForm` 和先改显示再发（被拒绝之后仍显示被拒绝的值）删除：值在 nerve 做完之后才变（W17）。
  - `core/lib/fake-controls.ts`：搜索选择的替身（`SearchSelect`，Task 2）还留下它的选项（`shown.searchSelects[i].options`）。
  - 静态检查：`.oxlintrc.json` 的非空断言的范围加上两个 vitest 文件（`fake-controls.ts` 在 Task 2 已加）。
- e2e 的共用部分（P3、P5 的故事和 `assert/project.ts` 保持在约 400 行：P3、P5 原来各写的经 API 的修改和读取移到共用的 fixture）：`e2e/fixtures/api.ts` 的 `projectMembershipOf`、`changeProject`、`projectMemberWrites`；`e2e/fixtures/assert/project.ts` 的 `projectSettingsOf`；`e2e/fixtures/project-pages.ts` 的 `shownNameOf`（页面显示的没有名字的账户：邮箱 `@` 之前的部分）。P3 的经 API 的故事中成员被拒绝的添加也经 `projectMemberWrites`；P5 的页面版本中访客的成员关系的结束（Task 7）同样。

**Tests:**
- vitest：`add-project-members-modal.test.tsx`：`offers the workspace's active members who are not the project's`；`adds them, then closes and says so`；`stays open and shows nerve's reason when it refuses them`；`neither closes nor says anything when the adding $settles after another tab moved this one`。`project-roles.test.tsx` 加 `the roles a member is added with > $who`（`addableRoles` 的表），添加的选择也按键选。`project-settings-member-defaults.test.tsx`：`says a change is made, and nerve's reason for refusing one`；`says nothing when a change $settles after another tab moved this one`。
- 端到端：P5 的页面版本在开头加上添加（`P5 (page): a project admin who is no workspace admin adds a member and a guest from the workspace's members who are not the project's, the modal held until nerve answers; …`：成员的选择用键盘：Tab 到它，Enter 打开，搜索框有焦点（Task 2 的 `CustomSearchSelect` 的自定义按钮也在 Tab 顺序里）；候选只有工作区的管理员、ann 和 gus；gus 只能是访客；请求体经 `sentHeld`，扣住时取消禁用、Escape 不关；成功的提示在角色列表之前关上，它会遮住列表）。`P3 (page): the project's admin makes a member its lead and its default assignee, each picked from its members who are not its guests, and lets its guests see every work item`（每个修改只发它的字段；数据库中的设置）。

- [ ] **Step 1: 添加时的角色、添加的弹窗和测试的替身**

`web/apps/web/core/components/project/add-project-members-modal.test.tsx`（新文件，112 行）：

````file web/apps/web/core/components/project/add-project-members-modal.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectMembersAdd } from "@nerve/api-client";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, shown, submitModalForm } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { membershipOf } from "@/store/member/workspace/fake-members";
import { projectOf } from "@/store/project/fake-projects";
import { AddProjectMembersModal } from "./add-project-members-modal";

// Whom the project's add-members modal offers, and what it does with nerve's answer (M3 design 2 P5, 3.5, 7.1, 7.6):
// the modal renders on the server with stand-ins for the UI kit's controls (fake-controls.ts), which keep the props
// they were given, and for the member stores: of acme's members, bob is web's already and sid's membership ended;
// ann, wes (its admin) and gus (its guest) are not web's. nerve answers the adding when the test says. Its session is
// fake-tab.ts's.

const web = projectOf("WEB", "w-acme");
const acme = [
  membershipOf("ann"),
  membershipOf("bob"),
  membershipOf("sid", { is_active: false }),
  membershipOf("wes", { role: 20 }),
  membershipOf("gus", { role: 5 }),
];
const page = vi.hoisted(() => ({ bulkAddMembersToProject: vi.fn(), onClose: vi.fn() }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({
    project: {
      getProjectMemberDetails: (userId: string) => (userId === "u-bob" ? {} : null),
      bulkAddMembersToProject: page.bulkAddMembersToProject,
    },
    workspace: {
      workspaceMemberIds: acme.map((membership) => membership.member.id),
      getWorkspaceMemberDetails: (userId: string) => acme.find((membership) => membership.member.id === userId) ?? null,
    },
  }),
}));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));

/** Renders the modal and gives its member select. */
function render() {
  emptyShown();
  renderToStaticMarkup(<AddProjectMembersModal isOpen onClose={page.onClose} projectId={web.id} />);
  const [members] = shown.searchSelects;
  if (!members) throw new Error("the modal showed no member select");
  return members;
}

/** Renders the modal, picks ann, and submits it: settles once the modal has followed nerve's answer. */
function addAnn() {
  render().onChange("u-ann");
  return submitModalForm();
}

beforeEach(() => {
  signedIn();
  page.bulkAddMembersToProject.mockReset();
  page.bulkAddMembersToProject.mockImplementation((_projectId: string, _data: ProjectMembersAdd) =>
    Promise.resolve([])
  );
  page.onClose.mockReset();
  toasts.length = 0;
});

describe("AddProjectMembersModal", () => {
  it("offers the workspace's active members who are not the project's", () => {
    expect(render().options?.map((option) => option.value)).toEqual(["u-ann", "u-wes", "u-gus"]);
  });

  it("adds them, then closes and says so", async () => {
    await addAnn();
    expect(page.bulkAddMembersToProject.mock.calls).toEqual([
      [web.id, { members: [{ member_id: "u-ann", role: 15 }] }],
    ]);
    expect(page.onClose).toHaveBeenCalledTimes(1);
    expect(toasts).toEqual([{ type: "success", title: "Success!", message: "Members added successfully." }]);
  });

  it("stays open and shows nerve's reason when it refuses them", async () => {
    page.bulkAddMembersToProject.mockRejectedValueOnce(
      refusal(422, "validation_failed", [{ field: "members[0].member_id", code: "duplicate" }])
    );
    await addAnn();
    expect(page.onClose).not.toHaveBeenCalled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.validation_failed" }]);
  });

  it.each(lateSettlings)(
    "neither closes nor says anything when the adding $settles after another tab moved this one",
    async ({ settle }) => {
      const adding = heldChange<undefined>();
      page.bulkAddMembersToProject.mockReturnValueOnce(adding.sent);
      const added = addAnn();
      // the form's checks come first: the adding leaves once they pass
      await vi.waitFor(() => expect(page.bulkAddMembersToProject).toHaveBeenCalled());
      switchAccount();
      settle(adding);
      await added;
      expect([page.onClose.mock.calls, toasts]).toEqual([[], []]);
    }
  );
});
````

`web/apps/web/core/components/project/add-project-members-modal.tsx`（修改，19 处）：

````old web/apps/web/core/components/project/add-project-members-modal.tsx
import type { ProjectMembersAdd, ProjectRole } from "@nerve/api-client";
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
import type { ProjectMembersAdd } from "@nerve/api-client";
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
import { ROLE, EUserPermissions } from "@nerve/constants";
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
import { ROLE_DETAILS } from "@nerve/constants";
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
import { useUserPermissions } from "@/hooks/store/user";
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
import { PROJECT_ROLES } from "./project-roles";
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
import { addableRoles } from "./project-roles";
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
  projectId: string;
  workspaceSlug: string;
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
  projectId: string;
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
  const { isOpen, onClose, projectId, workspaceSlug } = props;
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
  const { isOpen, onClose, projectId } = props;
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
  const toastRefusal = useRefusalToast();
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
  // derived values
  const currentProjectRole = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
  // derived values: the workspace's active members who are not the project's (M3 design 7.6), whom nerve adds
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
    (userId) => getProjectMemberDetails(userId, projectId) === null
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
    (userId) => getWorkspaceMemberDetails(userId)?.is_active && getProjectMemberDetails(userId, projectId) === null
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
  const onSubmit = async (formData: FormValues) => {
    if (!workspaceSlug || !projectId || isSubmitting) return;

    try {
      await bulkAddMembersToProject(projectId, formData);
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
  };
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
  // The modal waits for nerve, and cannot be closed meanwhile: it closes once nerve has added them; a refusal shows
  // nerve's reason and keeps the form. The page follows the adding only in the session that sent it (M3 design 7.1).
  const onSubmit = (formData: FormValues) =>
    followInSession(() => bulkAddMembersToProject(projectId, formData), {
      done: () => {
        handleClose();
        setToast({
          title: "Success!",
          type: TOAST_TYPE.SUCCESS,
          message: "Members added successfully.",
        });
      },
      failed: toastRefusal,
    });
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx

  const checkCurrentOptionWorkspaceRole = (value: string): ProjectRole[] => {
    const currentMemberWorkspaceRole = getWorkspaceMemberDetails(value)?.role;
    if (!value || !currentMemberWorkspaceRole) return PROJECT_ROLES;

    const isGuestOROwner = [EUserPermissions.ADMIN, EUserPermissions.GUEST].includes(
      currentMemberWorkspaceRole as EUserPermissions
    );

    return PROJECT_ROLES.filter((role) => !isGuestOROwner || role === currentMemberWorkspaceRole);
  };

````
````new web/apps/web/core/components/project/add-project-members-modal.tsx

````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
    <ModalCore isOpen={isOpen} handleClose={handleClose} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
    <ModalCore
      isOpen={isOpen}
      handleClose={isSubmitting ? undefined : handleClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.XXL}
    >
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
                            <button className="shadow-sm flex w-full items-center justify-between gap-1 rounded-md border border-subtle px-3 py-2 text-left text-13 text-secondary duration-300 hover:bg-layer-1 hover:text-primary focus:outline-none">
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
                            <span className="shadow-sm flex w-full items-center justify-between gap-1 rounded-md border border-subtle px-3 py-2 text-left text-13 text-secondary duration-300 hover:bg-layer-1 hover:text-primary focus:outline-none">
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
                            </button>
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
                            </span>
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
                                {roleField.value ? ROLE[roleField.value] : "Select role"}
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
                                {roleField.value ? t(ROLE_DETAILS[roleField.value].i18n_title) : "Select role"}
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
                          {checkCurrentOptionWorkspaceRole(watch(`members.${index}.member_id`)).map((role) => {
                            if (role > (currentProjectRole ?? EUserPermissions.GUEST)) return null;

                            return (
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
                          {addableRoles(getWorkspaceMemberDetails(watch(`members.${index}.member_id`))?.role).map(
                            (role) => (
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
                                {ROLE[role]}
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
                                {t(ROLE_DETAILS[role].i18n_title)}
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
                            );
                          })}
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
                            )
                          )}
````
````old web/apps/web/core/components/project/add-project-members-modal.tsx
            <Button variant="secondary" size="lg" onClick={handleClose}>
````
````new web/apps/web/core/components/project/add-project-members-modal.tsx
            <Button variant="secondary" size="lg" onClick={handleClose} disabled={isSubmitting}>
````

`web/apps/web/core/components/project/member-list.tsx`（修改，1 处）：

````old web/apps/web/core/components/project/member-list.tsx
        projectId={projectId}
        workspaceSlug={workspaceSlug}
````
````new web/apps/web/core/components/project/member-list.tsx
        projectId={projectId}
````

`web/apps/web/core/components/project/project-roles.test.tsx`（修改，6 处）：

````old web/apps/web/core/components/project/project-roles.test.tsx
import { PROJECT_ROLES, canRemove, roleChoices, type MembershipCaller, type ShownMembership } from "./project-roles";
````
````new web/apps/web/core/components/project/project-roles.test.tsx
import {
  PROJECT_ROLES,
  addableRoles,
  canRemove,
  roleChoices,
  type MembershipCaller,
  type ShownMembership,
} from "./project-roles";
````
````old web/apps/web/core/components/project/project-roles.test.tsx
vi.mock("react-router", () => ({ useNavigate: () => vi.fn() }));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => ({ data: { id: "u-me" } }),
  useUserPermissions: () => ({ getProjectRoleByWorkspaceSlugAndProjectId: () => 20 }),
}));
````
````new web/apps/web/core/components/project/project-roles.test.tsx
vi.mock("react-router", () => ({ useNavigate: () => vi.fn() }));
````
````old web/apps/web/core/components/project/project-roles.test.tsx
/**
 * Each role a select offers: its name, as the add modal shows it; the key the members page shows it by (ROLE_DETAILS's,
 * translated); the number nerve takes.
 */
````
````new web/apps/web/core/components/project/project-roles.test.tsx
/** Each role a select offers: its name; the key the selects show it by (ROLE_DETAILS's, translated); its number. */
````
````old web/apps/web/core/components/project/project-roles.test.tsx
  it.each(roles)("add ann as $label with that role's number", async ({ label, role }) => {
    renderToStaticMarkup(<AddProjectMembersModal isOpen onClose={vi.fn()} projectId={web.id} workspaceSlug="acme" />);
````
````new web/apps/web/core/components/project/project-roles.test.tsx
  it.each(roles)("add ann as $label with that role's number", async ({ key, role }) => {
    renderToStaticMarkup(<AddProjectMembersModal isOpen onClose={vi.fn()} projectId={web.id} />);
````
````old web/apps/web/core/components/project/project-roles.test.tsx
    pick(roleSelect, label);
````
````new web/apps/web/core/components/project/project-roles.test.tsx
    pick(roleSelect, key);
````
````old web/apps/web/core/components/project/project-roles.test.tsx
const workspaceAdmin: MembershipCaller = { workspaceRole: 20, projectRole: 20 };
````
````new web/apps/web/core/components/project/project-roles.test.tsx
const workspaceAdmin: MembershipCaller = { workspaceRole: 20, projectRole: 20 };

describe("the roles a member is added with", () => {
  it.each<{ who: string; workspaceRole: WorkspaceRole | undefined; offered: ProjectRole[] }>([
    { who: "the workspace's admin: an admin's", workspaceRole: 20, offered: [20] },
    { who: "its member: any", workspaceRole: 15, offered: [5, 15, 20] },
    { who: "its guest: a guest's", workspaceRole: 5, offered: [5] },
    { who: "no one picked yet: any", workspaceRole: undefined, offered: [5, 15, 20] },
  ])("$who", ({ workspaceRole, offered }) => {
    expect(addableRoles(workspaceRole)).toEqual(offered);
  });
});
````

`web/apps/web/core/components/project/project-roles.ts`（修改，1 处）：

````old web/apps/web/core/components/project/project-roles.ts
export const PROJECT_ROLES: ProjectRole[] = [5, 15, 20];
````
````new web/apps/web/core/components/project/project-roles.ts
export const PROJECT_ROLES: ProjectRole[] = [5, 15, 20];

/**
 * The roles a member of the workspace whose role in it is workspaceRole may be added to a project with, as nerve allows
 * them (M3 design 3.5): a workspace admin as an admin alone, a workspace guest as a guest alone, a member as any; any
 * while no member is picked. Adding has no relative rule.
 */
export function addableRoles(workspaceRole: WorkspaceRole | undefined): ProjectRole[] {
  if (workspaceRole === 20) return [20];
  if (workspaceRole === 5) return [5];
  return PROJECT_ROLES;
}
````

`web/apps/web/core/lib/fake-controls.ts`（修改，1 处）：

````old web/apps/web/core/lib/fake-controls.ts
/** A search select: what it gives, how its list opens, and the button it shows. */
type SearchSelect = Field & { focusSearchOnOpen?: boolean; popperModifiers?: object[]; customButton?: ReactNode };
````
````new web/apps/web/core/lib/fake-controls.ts
/** A search select: what it gives, how its list opens, the button it shows, and its options, each by its value. */
type SearchSelect = Field & {
  focusSearchOnOpen?: boolean;
  popperModifiers?: object[];
  customButton?: ReactNode;
  options?: { value: string }[];
};
````

- [ ] **Step 2: 成员的默认值**

`web/apps/web/core/components/project/project-settings-member-defaults.test.tsx`（修改，8 处）：

````old web/apps/web/core/components/project/project-settings-member-defaults.test.tsx
import type { Project, ProjectUpdate } from "@nerve/api-client";
import { emptyShown, shown } from "@/lib/fake-controls";
````
````new web/apps/web/core/components/project/project-settings-member-defaults.test.tsx
import type { Project, ProjectUpdate } from "@nerve/api-client";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
````
````old web/apps/web/core/components/project/project-settings-member-defaults.test.tsx
// built in its turn): the page renders on the server with stand-ins for the member selects and the guests' switch
// (fake-controls.ts), which keep the props they were given, and for the project store, whose changes nerve has not
// answered yet.
````
````new web/apps/web/core/components/project/project-settings-member-defaults.test.tsx
// built in its turn), and what the page says of them (M3 design 7.1): the page renders on the server with stand-ins
// for the member selects and the guests' switch (fake-controls.ts), which keep the props they were given, and for the
// project store, whose changes nerve has not answered yet unless the test says. Its session is fake-tab.ts's.
````
````old web/apps/web/core/components/project/project-settings-member-defaults.test.tsx
  // nerve has not answered: a change stays out
  const updateProject = vi.fn((_projectId: string, _data: ProjectUpdate) => new Promise<never>(() => {}));
  const toggleProject = vi.fn((_projectId: string, _field: ProjectToggleField) => new Promise<never>(() => {}));
````
````new web/apps/web/core/components/project/project-settings-member-defaults.test.tsx
  // nerve has not answered: a change stays out, unless a test answers it
  const updateProject = vi.fn((_projectId: string, _data: ProjectUpdate): Promise<unknown> => new Promise(() => {}));
  const toggleProject = vi.fn(
    (_projectId: string, _field: ProjectToggleField): Promise<unknown> => new Promise(() => {})
  );
````
````old web/apps/web/core/components/project/project-settings-member-defaults.test.tsx
    currentProjectDetails: web,
````
````new web/apps/web/core/components/project/project-settings-member-defaults.test.tsx
    getProjectById: () => web,
````
````old web/apps/web/core/components/project/project-settings-member-defaults.test.tsx
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
````
````new web/apps/web/core/components/project/project-settings-member-defaults.test.tsx
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
````
````old web/apps/web/core/components/project/project-settings-member-defaults.test.tsx
beforeEach(() => {
````
````new web/apps/web/core/components/project/project-settings-member-defaults.test.tsx
beforeEach(() => {
  signedIn();
````
````old web/apps/web/core/components/project/project-settings-member-defaults.test.tsx
  page.toggleProject.mockClear();
````
````new web/apps/web/core/components/project/project-settings-member-defaults.test.tsx
  page.toggleProject.mockClear();
  toasts.length = 0;
````
````old web/apps/web/core/components/project/project-settings-member-defaults.test.tsx
    expect(page.updateProject).not.toHaveBeenCalled();
  });
````
````new web/apps/web/core/components/project/project-settings-member-defaults.test.tsx
    expect(page.updateProject).not.toHaveBeenCalled();
  });

  it("says a change is made, and nerve's reason for refusing one", async () => {
    page.updateProject.mockResolvedValueOnce(web);
    page.updateProject.mockRejectedValueOnce(
      refusal(422, "validation_failed", [{ field: "project_lead_id", code: "not_allowed" }])
    );
    const { lead } = render();
    lead.onChange("u-ann");
    lead.onChange("u-gus");
    await vi.waitFor(() => expect(toasts).toHaveLength(2));
    expect(toasts).toEqual([
      { type: "success", title: "success!", message: "project_settings.general.toast.success" },
      { type: "error", title: "toast.error", message: "errors.validation_failed" },
    ]);
  });

  it.each(lateSettlings)("says nothing when a change $settles after another tab moved this one", async ({ settle }) => {
    const change = heldChange<undefined>();
    page.toggleProject.mockReturnValueOnce(change.sent);
    const { guests } = render();
    const turned = guests.onCheckedChange(true);
    switchAccount();
    settle(change);
    await turned;
    expect(toasts).toEqual([]);
  });
````

`web/apps/web/core/components/project/project-settings-member-defaults.tsx`（整个文件，139 行）：

````whole web/apps/web/core/components/project/project-settings-member-defaults.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { observer } from "mobx-react";
// nerve imports
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { ProjectUpdate } from "@nerve/api-client";
import { Switch } from "@makeplane/propel/components/switch";
import { Loader } from "@nerve/ui";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
// local imports
import { MemberSelect } from "./member-select";

/** A change of one of the project's member defaults: the one field changed (ProjectUpdate is a partial change). */
type TMemberDefault = Pick<ProjectUpdate, "project_lead_id"> | Pick<ProjectUpdate, "default_assignee_id">;

/** The member a select names, or none for its "none". */
const chosen = (value: string) => (value === "none" ? null : value);

type TDefaultSettingItemProps = {
  title: string;
  description: string;
  children: ReactNode;
};

function DefaultSettingItem({ title, description, children }: TDefaultSettingItemProps) {
  return (
    <div className="flex items-center justify-between gap-x-2">
      <div className="flex flex-col gap-0.5">
        <h4 className="text-13 font-medium">{title}</h4>
        <p className="text-11 text-tertiary">{description}</p>
      </div>
      <div className="w-full max-w-48 sm:max-w-64">{children}</div>
    </div>
  );
}

type TProjectSettingsMemberDefaultsProps = {
  workspaceSlug: string;
  projectId: string;
};

/**
 * The project's lead, its default assignee (each one of its members who is not its guest, M3 design 7.6) and its
 * guests' view of every work item, as nerve last answered them: a change shows once nerve has made it. The page
 * follows each change only in the session it was sent in (M3 design 7.1), a refusal with nerve's reason in a toast.
 */
export const ProjectSettingsMemberDefaults = observer(function ProjectSettingsMemberDefaults(
  props: TProjectSettingsMemberDefaultsProps
) {
  const { workspaceSlug, projectId } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { allowPermissions } = useUserPermissions();
  const { getProjectById, updateProject, toggleProject } = useProject();
  const toastRefusal = useRefusalToast();
  // derived values
  const project = getProjectById(projectId);
  const isAdmin = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.PROJECT, workspaceSlug, projectId);

  const followers = {
    done: () =>
      setToast({
        title: `${t("success")}!`,
        type: TOAST_TYPE.SUCCESS,
        message: t("project_settings.general.toast.success"),
      }),
    failed: toastRefusal,
  };

  /**
   * Changes the project's lead or its default assignee: data is the one field changed, as ProjectUpdate is a partial
   * change and nerve keeps the other as it has it (v0 design 7.7).
   */
  const submitChanges = (data: TMemberDefault) => followInSession(() => updateProject(projectId, data), followers);

  /** Turns the guests' view of every work item the other way, from nerve's last answer, in the change's turn. */
  const toggleGuestViewAllIssues = () =>
    followInSession(() => toggleProject(projectId, "guest_view_all_features"), followers);

  return (
    <div className="my-6 flex flex-col gap-y-6">
      <DefaultSettingItem title="Project Lead" description="Select the project lead for the project.">
        {project ? (
          <MemberSelect
            value={project.project_lead_id}
            onChange={(val: string) => void submitChanges({ project_lead_id: chosen(val) })}
            isDisabled={!isAdmin}
          />
        ) : (
          <Loader className="h-9 w-full">
            <Loader.Item width="100%" height="100%" />
          </Loader>
        )}
      </DefaultSettingItem>
      <DefaultSettingItem title="Default Assignee" description="Select the default assignee for the project.">
        {project ? (
          <MemberSelect
            value={project.default_assignee_id}
            onChange={(val: string) => void submitChanges({ default_assignee_id: chosen(val) })}
            isDisabled={!isAdmin}
          />
        ) : (
          <Loader className="h-9 w-full">
            <Loader.Item width="100%" height="100%" />
          </Loader>
        )}
      </DefaultSettingItem>
      {project && (
        <DefaultSettingItem
          title="Guest access"
          description="This will allow guests to have view access to all the project work items."
        >
          <div className="flex items-center justify-end">
            <Switch
              size="sm"
              checked={project.guest_view_all_features}
              onCheckedChange={toggleGuestViewAllIssues}
              disabled={!isAdmin}
              aria-label="Guest access"
            />
          </div>
        </DefaultSettingItem>
      )}
    </div>
  );
});
````

- [ ] **Step 3: 静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/components/project/project-roles.test.tsx"
````
````new .oxlintrc.json
        "web/apps/web/core/components/project/project-roles.test.tsx",
        "web/apps/web/core/components/project/add-project-members-modal.test.tsx",
        "web/apps/web/core/components/project/project-settings-member-defaults.test.tsx"
````

- [ ] **Step 4: 端到端**

`e2e/fixtures/api.ts`（修改，1 处）：

````old e2e/fixtures/api.ts
}

/** Archives the project of projectId with the bearer token given, an admin's of the project, and returns it. */
````
````new e2e/fixtures/api.ts
}

/** The id of the membership of the account of memberId in the project of projectId, as the caller of token lists it. */
export async function projectMembershipOf(
  api: Api,
  token: string,
  projectId: string,
  memberId: string
): Promise<string> {
  const { data, error, response } = await api.GET("/api/v0/projects/{project_id}/members", {
    params: { path: { project_id: projectId } },
    headers: bearer(token),
  });
  expect(response.status, `the members of ${projectId}: ${JSON.stringify(error)}`).toBe(200);
  const membership = data?.data.find((m) => m.member_id === memberId);
  if (!membership) {
    throw new Error(`no membership of ${memberId} in ${projectId}`);
  }
  return membership.id;
}

/** The answer of PATCH /api/v0/projects/{project_id}: its status, the project, or the problem's code and fields. */
export async function changeProject(api: Api, token: string, id: string, body: ProjectUpdate) {
  const { data, error, response } = await api.PATCH("/api/v0/projects/{project_id}", {
    params: { path: { project_id: id } },
    body,
    headers: bearer(token),
  });
  return data
    ? { status: response.status, project: data }
    : {
        status: response.status,
        code: error?.code,
        errors: error?.errors?.map((e) => ({ field: e.field, code: e.code })),
      };
}

/** The writes on a project's members, each by the caller of token. */
export function projectMemberWrites(api: Api, projectId: string) {
  return {
    add: async (token: string, members: ProjectMemberNew[]) => {
      const { error, response } = await api.POST("/api/v0/projects/{project_id}/members", {
        params: { path: { project_id: projectId } },
        body: { members },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    change: async (token: string, membership: string, role: 5 | 15 | 20) => {
      const { error, response } = await api.PATCH("/api/v0/project-members/{project_member_id}", {
        params: { path: { project_member_id: membership } },
        body: { role },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    remove: async (token: string, membership: string) => {
      const { error, response } = await api.DELETE("/api/v0/project-members/{project_member_id}", {
        params: { path: { project_member_id: membership } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    leave: async (token: string) => {
      const { error, response } = await api.POST("/api/v0/projects/{project_id}/leave", {
        params: { path: { project_id: projectId } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
  };
}

/** Archives the project of projectId with the bearer token given, an admin's of the project, and returns it. */
````

`e2e/fixtures/assert/project.ts`（修改，1 处）：

````old e2e/fixtures/assert/project.ts
        in_its_project: true,
        deleted_with_its_last_write: true,
      }))
  );
}

````
````new e2e/fixtures/assert/project.ts
        in_its_project: true,
        deleted_with_its_last_write: true,
      }))
  );
}

/** The project's settings as stored, its lead and default assignee by address, and who changed it last. */
export async function projectSettingsOf(db: Database, id: string): Promise<unknown> {
  const [row] = await db.query(
    `SELECT p.name, p.identifier, p.description, p.network, p.timezone, p.logo_props, p.cycle_view, p.module_view,
            p.issue_views_view, p.intake_view, p.guest_view_all_features, p.archive_in, l.email AS lead,
            a.email AS default_assignee, u.email AS by
       FROM projects p
       JOIN users u ON u.id = p.updated_by_id
       LEFT JOIN users l ON l.id = p.project_lead_id
       LEFT JOIN users a ON a.id = p.default_assignee_id
      WHERE p.id = $1`,
    [id]
  );
  return row;
}

````

`e2e/fixtures/project-pages.ts`（修改，1 处）：

````old e2e/fixtures/project-pages.ts
type Sent = { method: string; path: string };
````
````new e2e/fixtures/project-pages.ts
type Sent = { method: string; path: string };

/** The name the pages show of an account nobody named: nerve's display name for it, its address before the @. */
export function shownNameOf(email: string): string {
  return email.slice(0, email.indexOf("@"));
}
````

`e2e/stories/project/p3-project-settings.spec.ts`（修改，15 处）：

````old e2e/stories/project/p3-project-settings.spec.ts
  amidAnotherWorkspace,
````
````new e2e/stories/project/p3-project-settings.spec.ts
  amidAnotherWorkspace,
  changeProject,
````
````old e2e/stories/project/p3-project-settings.spec.ts
  inviteAndAccept,
  slugFor,
  type Api,
  type ProjectUpdate,
````
````new e2e/stories/project/p3-project-settings.spec.ts
  inviteAndAccept,
  projectMemberWrites,
  slugFor,
````
````old e2e/stories/project/p3-project-settings.spec.ts
import { expectMember } from "../../fixtures/assert/project";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
````
````new e2e/stories/project/p3-project-settings.spec.ts
import { expectMember, projectSettingsOf } from "../../fixtures/assert/project";
import { accountId, bearer, createPAT, emailFor, newAccount, register } from "../../fixtures/auth";
````
````old e2e/stories/project/p3-project-settings.spec.ts
import type { Database } from "../../fixtures/db";
````
````new e2e/stories/project/p3-project-settings.spec.ts
import { shownNameOf } from "../../fixtures/project-pages";
````
````old e2e/stories/project/p3-project-settings.spec.ts
// P3, a project's settings (M3 design 2, 3.4, 3.5, 3.19, 7.6).

/** The answer of PATCH /api/v0/projects/{project_id}: its status, the project, or the problem's code and fields. */
async function change(api: Api, token: string, id: string, body: ProjectUpdate) {
  const { data, error, response } = await api.PATCH("/api/v0/projects/{project_id}", {
    params: { path: { project_id: id } },
    body,
    headers: bearer(token),
  });
  return data
    ? { status: response.status, project: data }
    : {
        status: response.status,
        code: error?.code,
        errors: error?.errors?.map((e) => ({ field: e.field, code: e.code })),
      };
}

/** The project's settings as stored, its lead and default assignee by address, and who changed it last. */
async function stored(db: Database, id: string): Promise<unknown> {
  const [row] = await db.query(
    `SELECT p.name, p.identifier, p.description, p.network, p.timezone, p.logo_props, p.cycle_view, p.module_view,
            p.issue_views_view, p.intake_view, p.guest_view_all_features, p.archive_in, l.email AS lead,
            a.email AS default_assignee, u.email AS by
       FROM projects p
       JOIN users u ON u.id = p.updated_by_id
       LEFT JOIN users l ON l.id = p.project_lead_id
       LEFT JOIN users a ON a.id = p.default_assignee_id
      WHERE p.id = $1`,
    [id]
  );
  return row;
}
````
````new e2e/stories/project/p3-project-settings.spec.ts
// P3, a project's settings (M3 design 2, 3.4, 3.5, 3.19, 7.6).
````
````old e2e/stories/project/p3-project-settings.spec.ts
  const addedByMember = await api.POST("/api/v0/projects/{project_id}/members", {
    params: { path: { project_id: web.id } },
    body: { members: [{ member_id: otherId, role: 15 }] },
    headers: bearer(member),
  });
  expect({ status: addedByMember.response.status, code: addedByMember.error?.code }).toEqual({
````
````new e2e/stories/project/p3-project-settings.spec.ts
  expect(await projectMemberWrites(api, web.id).add(member, [{ member_id: otherId, role: 15 }])).toEqual({
````
````old e2e/stories/project/p3-project-settings.spec.ts
    await change(api, admin, web.id, {
````
````new e2e/stories/project/p3-project-settings.spec.ts
    await changeProject(api, admin, web.id, {
````
````old e2e/stories/project/p3-project-settings.spec.ts
  const changed = { ...settings, lead: memberEmail, default_assignee: memberEmail, by: adminEmail };
  expect(await stored(db, web.id)).toEqual(changed);
````
````new e2e/stories/project/p3-project-settings.spec.ts
  const changed = { ...settings, lead: memberEmail, default_assignee: memberEmail, by: adminEmail };
  expect(await projectSettingsOf(db, web.id)).toEqual(changed);
````
````old e2e/stories/project/p3-project-settings.spec.ts
    await Promise.all(refusals.map(({ token, body }) => change(api, token, web.id, body))),
````
````new e2e/stories/project/p3-project-settings.spec.ts
    await Promise.all(refusals.map(({ token, body }) => changeProject(api, token, web.id, body))),
````
````old e2e/stories/project/p3-project-settings.spec.ts
  ).toEqual(refusals.map((r) => r.want));
  expect(await stored(db, web.id)).toEqual(changed);
````
````new e2e/stories/project/p3-project-settings.spec.ts
  ).toEqual(refusals.map((r) => r.want));
  expect(await projectSettingsOf(db, web.id)).toEqual(changed);
````
````old e2e/stories/project/p3-project-settings.spec.ts
  expect(await change(api, admin, web.id, { project_lead_id: null, default_assignee_id: null })).toMatchObject({
````
````new e2e/stories/project/p3-project-settings.spec.ts
  expect(await changeProject(api, admin, web.id, { project_lead_id: null, default_assignee_id: null })).toMatchObject({
````
````old e2e/stories/project/p3-project-settings.spec.ts
  expect(await stored(db, web.id)).toEqual({ ...changed, lead: null, default_assignee: null });
````
````new e2e/stories/project/p3-project-settings.spec.ts
  expect(await projectSettingsOf(db, web.id)).toEqual({ ...changed, lead: null, default_assignee: null });
````
````old e2e/stories/project/p3-project-settings.spec.ts
  expect(await stored(db, web.id)).toMatchObject({ ...settings, by: adminEmail });
````
````new e2e/stories/project/p3-project-settings.spec.ts
  expect(await projectSettingsOf(db, web.id)).toMatchObject({ ...settings, by: adminEmail });
````
````old e2e/stories/project/p3-project-settings.spec.ts

  expect(await stored(db, web.id)).toMatchObject({
    cycle_view: true,
````
````new e2e/stories/project/p3-project-settings.spec.ts

  expect(await projectSettingsOf(db, web.id)).toMatchObject({
    cycle_view: true,
````
````old e2e/stories/project/p3-project-settings.spec.ts
  await expectQuietConsole(page, watch, { warnings: Array.from({ length: 5 }, () => EMOJI_CHECK_WARNING) });
````
````new e2e/stories/project/p3-project-settings.spec.ts
  await expectQuietConsole(page, watch, { warnings: Array.from({ length: 5 }, () => EMOJI_CHECK_WARNING) });
});

test("P3 (page): the project's admin makes a member its lead and its default assignee, each picked from its members who are not its guests, and lets its guests see every work item", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  // ann is Web's member, gus its guest; otto is a member of acme, not of Web.
  const [ann, gus, otto] = await Promise.all(["ann", "gus", "otto"].map((label) => newAccount(api, testInfo, label)));
  if (!ann || !gus || !otto) throw new Error("the accounts were not registered");
  await inviteAndAccept(api, admin.access_token, slug, ann, 15);
  await inviteAndAccept(api, admin.access_token, slug, gus, 5);
  await inviteAndAccept(api, admin.access_token, slug, otto, 15);
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.access_token, web.id, [
    { member_id: ann.id, role: 15 },
    { member_id: gus.id, role: 5 },
  ]);

  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/settings/projects/${web.id}/members`);
  /** Picks ann in the member select under title: the select offers the admin and ann, and none; the page sends field. */
  const pickAnn = async (title: string) => {
    await page.getByRole("heading", { name: title }).locator("xpath=following::button[1]").click();
    await expect(page.getByRole("option")).toHaveCount(3);
    await expect(page.getByRole("option", { name: shownNameOf(adminEmail) })).toBeVisible();
    await expect(page.getByRole("option", { name: "None" })).toBeVisible();
    const picked = await sentTo(page, "PATCH", `/api/v0/projects/${web.id}`, () =>
      page.getByRole("option", { name: shownNameOf(ann.email) }).click()
    );
    await expect(page.getByRole("heading", { name: title }).locator("xpath=following::button[1]")).toHaveText(
      new RegExp(shownNameOf(ann.email))
    );
    return [picked.answer.status(), picked.body];
  };
  expect(await pickAnn("Project Lead")).toEqual([200, { project_lead_id: ann.id }]);
  expect(await pickAnn("Default Assignee")).toEqual([200, { default_assignee_id: ann.id }]);
  const guests = await sentTo(page, "PATCH", `/api/v0/projects/${web.id}`, () =>
    page.getByRole("switch", { name: "Guest access" }).click()
  );
  expect([guests.answer.status(), guests.body]).toEqual([200, { guest_view_all_features: true }]);
  await expect(page.getByRole("switch", { name: "Guest access" })).toBeChecked();

  expect(await projectSettingsOf(db, web.id)).toMatchObject({
    lead: ann.email,
    default_assignee: ann.email,
    guest_view_all_features: true,
    by: adminEmail,
  });
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
````

`e2e/stories/project/p5-project-members.spec.ts`（修改，14 处）：

````old e2e/stories/project/p5-project-members.spec.ts
  amidAnotherWorkspace,
  answer,
````
````new e2e/stories/project/p5-project-members.spec.ts
  amidAnotherWorkspace,
````
````old e2e/stories/project/p5-project-members.spec.ts
  membershipOf,
  slugFor,
  type Api,
  type ProjectMemberNew,
````
````new e2e/stories/project/p5-project-members.spec.ts
  membershipOf,
  projectMemberWrites,
  projectMembershipOf,
  slugFor,
````
````old e2e/stories/project/p5-project-members.spec.ts
  registerOnboarded,
````
````new e2e/stories/project/p5-project-members.spec.ts
  registerOnboarded,
  sentHeld,
````
````old e2e/stories/project/p5-project-members.spec.ts
import { endProjectMembership, leavingOf, removalOf } from "../../fixtures/project-pages";
````
````new e2e/stories/project/p5-project-members.spec.ts
import { endProjectMembership, leavingOf, removalOf, shownNameOf } from "../../fixtures/project-pages";
````
````old e2e/stories/project/p5-project-members.spec.ts
// P5, a project's members (M3 design 2, 3.5, 3.7, 7.6): adding them, changing a role, removing a member, leaving.

/** The writes on a project's members, each by the caller of token. */
function writes(api: Api, projectId: string) {
  return {
    add: async (token: string, members: ProjectMemberNew[]) => {
      const { error, response } = await api.POST("/api/v0/projects/{project_id}/members", {
        params: { path: { project_id: projectId } },
        body: { members },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    change: async (token: string, membership: string, role: 5 | 15 | 20) => {
      const { error, response } = await api.PATCH("/api/v0/project-members/{project_member_id}", {
        params: { path: { project_member_id: membership } },
        body: { role },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    remove: async (token: string, membership: string) => {
      const { error, response } = await api.DELETE("/api/v0/project-members/{project_member_id}", {
        params: { path: { project_member_id: membership } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    leave: async (token: string) => {
      const { error, response } = await api.POST("/api/v0/projects/{project_id}/leave", {
        params: { path: { project_id: projectId } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
  };
}
````
````new e2e/stories/project/p5-project-members.spec.ts
// P5, a project's members (M3 design 2, 3.5, 3.7, 7.6): adding them, changing a role, removing a member, leaving.
````
````old e2e/stories/project/p5-project-members.spec.ts
  const { add, change, remove, leave } = writes(api, web.id);
````
````new e2e/stories/project/p5-project-members.spec.ts
  const { add, change, remove, leave } = projectMemberWrites(api, web.id);
````
````old e2e/stories/project/p5-project-members.spec.ts
test("P5 (page): a project admin who is no workspace admin is offered only the roles below his own, none for another admin; he makes a member a guest, removes the other admin and a guest, the dialog held until nerve answers, and, its only admin now, is told why he may not leave, its dialog open; a member leaves, and the workspace's projects show once nerve has made it, not before; the guest leaves by the sidebar, its modal open after a refusal and held too", async ({
````
````new e2e/stories/project/p5-project-members.spec.ts
test("P5 (page): a project admin who is no workspace admin adds a member and a guest from the workspace's members who are not the project's, the modal held until nerve answers; he is offered only the roles below his own, none for another admin; he makes a member a guest, removes the other admin and a guest, the dialog held until nerve answers, and, its only admin now, is told why he may not leave, its dialog open; a member leaves, and the workspace's projects show once nerve has made it, not before; the guest leaves by the sidebar, its modal open after a refusal and held too", async ({
````
````old e2e/stories/project/p5-project-members.spec.ts
  // pat, a member of acme, makes Web, its admin; max its other admin, ann and bob its members, gus its guest.
````
````new e2e/stories/project/p5-project-members.spec.ts
  // pat, a member of acme, makes Web, its admin; max its other admin, bob its member.
````
````old e2e/stories/project/p5-project-members.spec.ts
  const [maxs, anns, , guss] = await addProjectMembers(api, pat.tokens.access_token, web.id, [
    { member_id: max.id, role: 20 },
    { member_id: ann.id, role: 15 },
    { member_id: bob.id, role: 15 },
    { member_id: gus.id, role: 5 },
````
````new e2e/stories/project/p5-project-members.spec.ts
  const [maxs] = await addProjectMembers(api, pat.tokens.access_token, web.id, [
    { member_id: max.id, role: 20 },
    { member_id: bob.id, role: 15 },
````
````old e2e/stories/project/p5-project-members.spec.ts
  if (!maxs || !anns || !guss) throw new Error("the members were not added");
````
````new e2e/stories/project/p5-project-members.spec.ts
  if (!maxs) throw new Error("the members were not added");
````
````old e2e/stories/project/p5-project-members.spec.ts
  await page.goto(members);
````
````new e2e/stories/project/p5-project-members.spec.ts
  await page.goto(members);
  // He adds ann as a member and gus as a guest, from acme's active members who are not Web's (the admin, ann and gus),
  // gus with a guest's role alone; the modal cannot be closed until nerve has added them.
  await page.getByRole("button", { name: "Add member" }).click();
  // the member select by the keyboard: Tab reaches it, Enter opens its list with the search focused
  const coWorker = page.getByRole("dialog").getByRole("button", { name: "Select co-worker" });
  await expect
    .poll(
      async () => {
        await page.keyboard.press("Tab");
        return coWorker.evaluate((button) => button === document.activeElement);
      },
      { timeout: 5_000 }
    )
    .toBe(true);
  await page.keyboard.press("Enter");
  await expect(page.getByRole("combobox", { name: "Search" })).toBeFocused();
  await expect(page.getByRole("option")).toHaveCount(3);
  await expect(page.getByRole("option", { name: shownNameOf(admin.email) })).toBeVisible();
  await page.getByRole("option", { name: shownNameOf(ann.email) }).click();
  await page.getByRole("button", { name: "Add more" }).click();
  await page.getByRole("button", { name: "Select co-worker" }).click();
  await page.getByRole("option", { name: shownNameOf(gus.email) }).click();
  await page.getByRole("button", { name: "Guest", exact: true }).click();
  await expect(page.getByRole("option")).toHaveText(["Guest"]);
  await page.getByRole("option", { name: "Guest" }).click();
  const adding = await sentHeld(page, "POST", `/api/v0/projects/${web.id}/members`, () =>
    page.getByRole("button", { name: "Add members" }).click()
  );
  expect(adding.body).toEqual({
    members: [
      { member_id: ann.id, role: 15 },
      { member_id: gus.id, role: 5 },
    ],
  });
  await expect(page.getByRole("dialog").getByRole("button", { name: "Cancel" })).toBeDisabled();
  expect(await closedByEscape(page)).toBe(false);
  expect((await adding.release()).status()).toBe(201);
  await expect(page.getByRole("heading", { name: "Add members" })).toHaveCount(0);
  // the page says so; the toast is closed, as it would cover the role lists below it
  const saidSo = page.getByRole("dialog").filter({ hasText: "Members added successfully." });
  await saidSo.locator("button").click();
  await expect(saidSo).toHaveCount(0);
  await expect(memberRow(page, gus.email)).toContainText("Guest");
  const membershipOfWeb = (id: string) => projectMembershipOf(api, pat.tokens.access_token, web.id, id);
  const [anns, guss] = await Promise.all([membershipOfWeb(ann.id), membershipOfWeb(gus.id)]);
````
````old e2e/stories/project/p5-project-members.spec.ts
  const demoted = await sentTo(page, "PATCH", `/api/v0/project-members/${anns.id}`, () =>
````
````new e2e/stories/project/p5-project-members.spec.ts
  const demoted = await sentTo(page, "PATCH", `/api/v0/project-members/${anns}`, () =>
````
````old e2e/stories/project/p5-project-members.spec.ts
  const release = await holdAnswer(page, "DELETE", removalOf(guss.id).path);
  const removed = endProjectMembership(page, gus.email, "Remove", removalOf(guss.id));
````
````new e2e/stories/project/p5-project-members.spec.ts
  const release = await holdAnswer(page, "DELETE", removalOf(guss).path);
  const removed = endProjectMembership(page, gus.email, "Remove", removalOf(guss));
````
````old e2e/stories/project/p5-project-members.spec.ts
  expect((await writes(api, web.id).remove(pat.tokens.access_token, anns.id)).status).toBe(204);
````
````new e2e/stories/project/p5-project-members.spec.ts
  expect((await projectMemberWrites(api, web.id).remove(pat.tokens.access_token, anns)).status).toBe(204);
````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 68 条规则、3 个例外，没有命中；web 的 oxlint 349 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 104 个全部通过。

- [ ] **Step 6: 提交**

```bash
git add .oxlintrc.json e2e/fixtures/api.ts e2e/fixtures/assert/project.ts e2e/fixtures/project-pages.ts e2e/stories/project/p3-project-settings.spec.ts e2e/stories/project/p5-project-members.spec.ts web/apps/web/core/components/project/add-project-members-modal.test.tsx web/apps/web/core/components/project/add-project-members-modal.tsx web/apps/web/core/components/project/member-list.tsx web/apps/web/core/components/project/project-roles.test.tsx web/apps/web/core/components/project/project-roles.ts web/apps/web/core/components/project/project-settings-member-defaults.test.tsx web/apps/web/core/components/project/project-settings-member-defaults.tsx web/apps/web/core/lib/fake-controls.ts
```
```bash
git commit -m "feat(M3/P10): adding members offers whom nerve would add and waits for it; the member defaults show nerve's answer

The add modal offers the workspace's active members who are not the
project's, each with the roles nerve allows him (addableRoles), sends
ProjectMembersAdd through followInSession, closes once nerve has added
them and shows its reason when it refuses; it cannot be dismissed while
the adding is out. The lead, the default assignee and the guests' view
are nerve's values, changed one field at a time and shown once nerve has
made the change, its refusal said. P5's adding and P3's member defaults.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p10.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T8.1` | 添加成员的弹窗列出工作区成员关系已结束的人 | `add-project-members-modal.test.tsx` | vitest |
| `T8.2` | 添加成员的弹窗也列出项目已有的成员 | `add-project-members-modal.test.tsx`、故事 P5 | vitest；端到端 |
| `T8.3` | 工作区访客可以选成员和管理员的角色 | `project-roles.test.tsx`、故事 P5 | vitest；端到端 |
| `T8.4` | 添加在途时 Escape 关掉添加的弹窗 | 故事 P5 | 端到端 |
| `T8.5` | 添加在途时添加弹窗的取消按钮仍可用 | 故事 P5 | 端到端 |
| `T8.6` | 添加成员不论会话都跟进 | `add-project-members-modal.test.tsx` | vitest |
| `T8.7` | 添加被拒绝时什么都不说 | `add-project-members-modal.test.tsx` | vitest |
| `T8.8` | 成员默认值的修改不论会话都跟进 | `project-settings-member-defaults.test.tsx` | vitest |
| `T8.9` | 成员默认值的修改被拒绝时什么都不说 | `project-settings-member-defaults.test.tsx` | vitest |
| `T8.10` | 在添加成员弹窗的测试里写一个非空断言 | oxlint（`check:lint`） | 静态 |
| `T8.11` | 添加成员弹窗的角色下拉框用英文的 ROLE 称呼角色 | `project-roles.test.tsx` | vitest |

---

### Task 9: 归档、恢复和删除项目；P4 的页面版本

**Files:**
- Create: `web/apps/web/core/components/project/archive-restore-modal.test.tsx`、`web/apps/web/core/components/project/delete-project-modal.test.tsx`
- Modify: `.oxlintrc.json`、`e2e/stories/project/p4-archive.spec.ts`、`web/apps/web/core/components/project/archive-restore-modal.tsx`、`web/apps/web/core/components/project/card.tsx`、`web/apps/web/core/components/project/delete-project-modal.tsx`

**Interfaces:**
- Produces（spec 2.9；M3 设计 2 的 P4，3.4、3.6、3.19，7.1、7.6，9.5：删除项目的组件；P9 的 `DeleteWorkspaceModal` 的写法）：
  - `DeleteProjectModal`：`TDeleteProjectForm`、`CONFIRMATION = "delete my project"`；按提交的值核对名称和确认的话（`confirmed(values)`，提交按钮在两者都对之前不可用）；`deleteProject(project)` 经 `followInSession`：成功时关闭，只在页面是这个项目的（地址的 `projectId` 是它）时回到 `/{slug}/projects`（从已归档的项目页删除时停在原页），提示；被拒绝时经 `useRefusalToast` 说原因、停在弹窗；在途时关不掉（取消禁用、`handleClose` 为空）。原来不论会话都跳转、提示，被拒绝时提示固定的文字，在途时弹窗可以关。
  - `ArchiveRestoreProjectModal`（整个文件）：归档或恢复经 `followInSession`：成功时提示、关闭、回到 `/{slug}/projects`，忙到跳转做完；被拒绝时说原因、停在弹窗；在途时关不掉。
  - 已归档的卡片（`card.tsx`）：恢复、删除提供给项目的管理员，也提供给是工作区管理员的项目成员（`allowPermissions([ADMIN], PROJECT, …)`，与项目设置提供归档的条件相同，nerve 3.4 也允许他们；原来只看项目角色）；删除的图标按钮有名字（"Delete"）。
  - 静态检查：`.oxlintrc.json` 的非空断言的范围加上两个 vitest 文件。

**Tests:**
- vitest：`delete-project-modal.test.tsx`（9.5：删除项目的组件，修改在 `loginId` 改变之后才兑现时不跳转、不提示）：`deletes the project once its name and the words are typed; its own page gives way to the projects`；`stays on a page that is not the project's, the archived projects', and says so`；`deletes nothing when the form has $typed`（名称不对、确认的话不对）；`shows nerve's reason when it refuses, and stays`；`neither moves nor speaks when the deletion $settles after another tab moved this one to another account`。`archive-restore-modal.test.tsx`：`$change the project, says so and gives way to the projects`（归档、恢复）；`shows nerve's reason when it refuses, and stays`；`neither moves nor speaks when the archiving $settles after another tab moved this one to another account`。
- 端到端：`P4 (page): the admin archives a project from its settings, the dialog held until nerve answers, and it leaves his sidebar; a member of it who is the workspace's admin restores it from the archived projects; archived again, the admin deletes it there by its name, the dialog held too, and it is gone`（归档、删除被扣住时取消禁用、Escape 不关、地址不变；恢复的按钮在页面的主体里找（卡片的菜单在自己的门户里也有一个）；删除之后停在已归档的项目页；`expectProjectDeleted`：项目的每张表都先有一行）。

- [ ] **Step 1: 删除、归档和恢复**

`web/apps/web/core/components/project/archive-restore-modal.test.tsx`（新文件，99 行）：

````file web/apps/web/core/components/project/archive-restore-modal.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf } from "@/store/project/fake-projects";
import { ArchiveRestoreProjectModal } from "./archive-restore-modal";

// What archiving and restoring a project send and do on the page (M3 design 7.1): the modal renders on the server,
// with stand-ins for its buttons (fake-controls.ts), which keep the props they were given; the test presses its
// second, as a person would. Its session is fake-tab.ts's.

const web = projectOf("WEB", "w-acme", { name: "Web" });
const store = vi.hoisted(() => ({ archiveProject: vi.fn(), restoreProject: vi.fn() }));
const navigate = vi.hoisted(() => vi.fn());
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({
    getProjectById: () => web,
    archiveProject: store.archiveProject,
    restoreProject: store.restoreProject,
  }),
}));
vi.mock("react-router", () => ({ useNavigate: () => navigate }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Renders the modal of Web, to archive it or to restore it, and presses Archive or Restore; gives onClose. */
function press(archive: boolean) {
  const onClose = vi.fn();
  renderToStaticMarkup(
    <ArchiveRestoreProjectModal workspaceSlug="acme" projectId={web.id} isOpen onClose={onClose} archive={archive} />
  );
  shown.buttons[1]?.onClick?.({ preventDefault: () => {} });
  return onClose;
}

beforeEach(() => {
  signedIn();
  store.archiveProject.mockReset();
  store.archiveProject.mockResolvedValue(web);
  store.restoreProject.mockReset();
  store.restoreProject.mockResolvedValue(web);
  navigate.mockClear();
  toasts.length = 0;
  emptyShown();
});

describe("ArchiveRestoreProjectModal", () => {
  it.each([
    {
      change: "archives",
      archive: true,
      sent: () => store.archiveProject,
      said: { title: "Archive success", message: "Web has been archived successfully" },
    },
    {
      change: "restores",
      archive: false,
      sent: () => store.restoreProject,
      said: { title: "Restore success", message: "You can find Web in your projects." },
    },
  ])("$change the project, says so and gives way to the projects", async ({ archive, sent, said }) => {
    const onClose = press(archive);
    await pageSettled();
    expect(sent().mock.calls).toEqual([[web.id]]);
    expect(toasts).toEqual([{ type: "success", ...said }]);
    expect([onClose.mock.calls.length, navigate.mock.calls]).toEqual([1, [["/acme/projects"]]]);
  });

  it("shows nerve's reason when it refuses, and stays", async () => {
    store.archiveProject.mockRejectedValueOnce(refusal(403, "forbidden"));
    const onClose = press(true);
    await pageSettled();
    expect([onClose.mock.calls, navigate.mock.calls]).toEqual([[], []]);
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.forbidden" }]);
  });

  it.each(lateSettlings)(
    "neither moves nor speaks when the archiving $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const archiving = heldChange<undefined>();
      store.archiveProject.mockReturnValueOnce(archiving.sent);
      const onClose = press(true);
      switchAccount();
      settle(archiving);
      await pageSettled();
      expect([onClose.mock.calls, navigate.mock.calls, toasts]).toEqual([[], [], []]);
    }
  );
});
````

`web/apps/web/core/components/project/archive-restore-modal.tsx`（整个文件，97 行）：

````whole web/apps/web/core/components/project/archive-restore-modal.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
// ui
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { useNavigate } from "react-router";
// lib
import { followInSession } from "@/lib/in-session";

type Props = {
  workspaceSlug: string;

  projectId: string;
  isOpen: boolean;
  onClose: () => void;
  archive: boolean;
};

export function ArchiveRestoreProjectModal(props: Props) {
  const { workspaceSlug, projectId, isOpen, onClose, archive } = props;
  // router
  const navigate = useNavigate();
  // states
  const [isLoading, setIsLoading] = useState(false);
  // store hooks
  const { getProjectById, archiveProject, restoreProject } = useProject();
  const toastRefusal = useRefusalToast();

  const projectDetails = getProjectById(projectId);
  if (!projectDetails) return null;

  // The page follows the change only in the session it was sent in (M3 design 7.1): it says so, and gives way to the
  // workspace's projects; busy until it has. A refusal is said by nerve's reason, and the dialog stays.
  const handleChange = async () => {
    setIsLoading(true);
    await followInSession(() => (archive ? archiveProject(projectId) : restoreProject(projectId)), {
      done: () => {
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
        onClose();
        return navigate(`/${workspaceSlug}/projects`);
      },
      failed: toastRefusal,
    });
    setIsLoading(false);
  };

  // While the change is out the dialog cannot be dismissed (Cancel, Escape, the backdrop): nerve's answer is followed
  // by the dialog that sent it.
  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={isLoading ? undefined : onClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.LG}
    >
      <div className="px-5 py-4">
        <h3 className="text-18 font-medium 2xl:text-20">
          {archive ? "Archive" : "Restore"} {projectDetails.name}
        </h3>
        <p className="mt-3 text-13 text-secondary">
          {archive
            ? "This project and its work items, cycles and modules will be archived. Its work items won't appear in search. Only project admins can restore the project."
            : "Restoring a project will activate it and make it visible to all members of the project. Are you sure you want to continue?"}
        </p>
        <div className="mt-3 flex justify-end gap-2">
          <Button variant="secondary" size="lg" onClick={onClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button variant="primary" size="lg" onClick={() => void handleChange()} loading={isLoading}>
            {archive ? (isLoading ? "Archiving" : "Archive") : isLoading ? "Restoring" : "Restore"}
          </Button>
        </div>
      </div>
    </ModalCore>
  );
}
````

`web/apps/web/core/components/project/delete-project-modal.test.tsx`（新文件，101 行）：

````file web/apps/web/core/components/project/delete-project-modal.test.tsx
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
import { projectOf } from "@/store/project/fake-projects";
import { DeleteProjectModal } from "./delete-project-modal";

// What the deletion of a project sends and does on the page (M3 design 7.1, 9.5): the modal renders on the server,
// with stand-ins for its inputs and buttons (fake-controls.ts), which keep the props they were given; the test types
// as a person would, through the inputs' onChange, and submits the form. Its session is fake-tab.ts's; the address's
// params are params'.

const store = vi.hoisted(() => ({ deleteProject: vi.fn() }));
const navigate = vi.hoisted(() => vi.fn());
const params = vi.hoisted((): { workspaceSlug?: string; projectId?: string } => ({}));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => ({ deleteProject: store.deleteProject }) }));
vi.mock("react-router", () => ({ useNavigate: () => navigate, useParams: () => params }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@makeplane/propel/components/input", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const web = projectOf("WEB", "w-acme", { name: "Web" });

/** Renders the modal of Web, types name and words into its two inputs, and submits it; gives onClose. */
async function confirm(name: string, words: string) {
  const onClose = vi.fn();
  renderToStaticMarkup(<DeleteProjectModal isOpen project={web} onClose={onClose} />);
  const [nameInput, wordsInput] = shown.inputs;
  nameInput?.onChange({ target: { value: name } });
  wordsInput?.onChange({ target: { value: words } });
  await submitModalForm();
  return onClose;
}

beforeEach(() => {
  signedIn();
  store.deleteProject.mockReset();
  store.deleteProject.mockResolvedValue(undefined);
  navigate.mockClear();
  Object.assign(params, { workspaceSlug: "acme", projectId: web.id });
  toasts.length = 0;
  emptyShown();
});

const deleted = { type: "success", title: "Success!", message: "Project deleted successfully." };

describe("DeleteProjectModal", () => {
  it("deletes the project once its name and the words are typed; its own page gives way to the projects", async () => {
    const onClose = await confirm("Web", "delete my project");
    expect(store.deleteProject.mock.calls).toEqual([[web]]);
    expect([onClose.mock.calls.length, navigate.mock.calls]).toEqual([1, [["/acme/projects"]]]);
    expect(toasts).toEqual([deleted]);
  });

  it("stays on a page that is not the project's, the archived projects', and says so", async () => {
    params.projectId = undefined;
    const onClose = await confirm("Web", "delete my project");
    expect(store.deleteProject.mock.calls).toEqual([[web]]);
    expect([onClose.mock.calls.length, navigate.mock.calls]).toEqual([1, []]);
    expect(toasts).toEqual([deleted]);
  });

  it.each([
    { typed: "another name", name: "Web app", words: "delete my project" },
    { typed: "other words", name: "Web", words: "delete it" },
  ])("deletes nothing when the form has $typed", async ({ name, words }) => {
    const onClose = await confirm(name, words);
    expect([store.deleteProject.mock.calls, onClose.mock.calls, navigate.mock.calls, toasts]).toEqual([[], [], [], []]);
  });

  it("shows nerve's reason when it refuses, and stays", async () => {
    store.deleteProject.mockRejectedValueOnce(refusal(403, "forbidden"));
    const onClose = await confirm("Web", "delete my project");
    expect([onClose.mock.calls, navigate.mock.calls]).toEqual([[], []]);
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.forbidden" }]);
  });

  it.each(lateSettlings)(
    "neither moves nor speaks when the deletion $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const deletion = heldChange<undefined>();
      store.deleteProject.mockReturnValueOnce(deletion.sent);
      const submitted = confirm("Web", "delete my project");
      await vi.waitFor(() => expect(store.deleteProject).toHaveBeenCalledTimes(1));
      switchAccount();
      settle(deletion);
      const onClose = await submitted;
      expect([onClose.mock.calls, navigate.mock.calls, toasts]).toEqual([[], [], []]);
    }
  );
});
````

`web/apps/web/core/components/project/delete-project-modal.tsx`（修改，12 处）：

````old web/apps/web/core/components/project/delete-project-modal.tsx
import { useProject } from "@/hooks/store/use-project";
import { useParams, useNavigate } from "react-router";
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
import { useProject } from "@/hooks/store/use-project";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { useParams, useNavigate } from "react-router";
// lib
import { followInSession } from "@/lib/in-session";
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
const defaultValues = {
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
/** What the form asks before the deletion: the project's name, and the words that confirm it. */
type TDeleteProjectForm = { projectName: string; confirmDelete: string };

const defaultValues: TDeleteProjectForm = {
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
  confirmDelete: "",
};
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
  confirmDelete: "",
};

/** The words that confirm the deletion. */
const CONFIRMATION = "delete my project";
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
  const { deleteProject } = useProject();
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
  const { deleteProject } = useProject();
  const toastRefusal = useRefusalToast();
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
  } = useForm({ defaultValues });
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
  } = useForm<TDeleteProjectForm>({ defaultValues });
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
  const canDelete = watch("projectName") === project?.name && watch("confirmDelete") === "delete my project";
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
  const confirmed = (values: TDeleteProjectForm) =>
    values.projectName === project.name && values.confirmDelete === CONFIRMATION;
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
  const onSubmit = async () => {
    if (!workspaceSlug || !canDelete) return;

    try {
      await deleteProject(project);
      if (projectId && projectId === project.id) navigate(`/${workspaceSlug}/projects`);
      handleClose();
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: "Project deleted successfully.",
      });
    } catch (_error) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Something went wrong. Please try again later.",
      });
    }
  };

  return (
    <ModalCore isOpen={isOpen} handleClose={handleClose} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
  // The values submitted decide, as typed; the page follows the deletion only in the session it was sent in (M3
  // design 7.1): once another tab has moved this one to another account, the page is that account's. A page of the
  // project deleted gives way to the workspace's projects.
  const onSubmit = (values: TDeleteProjectForm) => {
    if (!workspaceSlug || !confirmed(values)) return;
    return followInSession(() => deleteProject(project), {
      done: () => {
        handleClose();
        if (projectId === project.id) void navigate(`/${workspaceSlug}/projects`);
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: "Success!",
          message: "Project deleted successfully.",
        });
      },
      failed: toastRefusal,
    });
  };

  // While the deletion is out the dialog cannot be dismissed (Cancel, Escape, the backdrop): nerve's answer is followed
  // by the dialog that sent it, and no dialog opened again offers the deletion while the request is out.
  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={isSubmitting ? undefined : handleClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.XXL}
    >
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
            Are you sure you want to delete project <span className="font-semibold break-words">{project?.name}</span>?
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
            Are you sure you want to delete project <span className="font-semibold break-words">{project.name}</span>?
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
            Enter the project name <span className="font-medium text-primary">{project?.name}</span> to continue:
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
            Enter the project name <span className="font-medium text-primary">{project.name}</span> to continue:
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
            To confirm, type <span className="font-medium text-primary">delete my project</span> below:
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
            To confirm, type <span className="font-medium text-primary">{CONFIRMATION}</span> below:
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
          <Button variant="secondary" size="lg" onClick={handleClose}>
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
          <Button variant="secondary" size="lg" onClick={handleClose} disabled={isSubmitting}>
````
````old web/apps/web/core/components/project/delete-project-modal.tsx
          <Button variant="error-fill" size="lg" type="submit" disabled={!canDelete} loading={isSubmitting}>
````
````new web/apps/web/core/components/project/delete-project-modal.tsx
          <Button variant="error-fill" size="lg" type="submit" disabled={!confirmed(watch())} loading={isSubmitting}>
````

- [ ] **Step 2: 已归档的卡片**

`web/apps/web/core/components/project/card.tsx`（修改，5 处）：

````old web/apps/web/core/components/project/card.tsx
import { EUserPermissions } from "@nerve/constants";
````
````new web/apps/web/core/components/project/card.tsx
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
````
````old web/apps/web/core/components/project/card.tsx
import { useMember } from "@/hooks/store/use-member";
````
````new web/apps/web/core/components/project/card.tsx
import { useMember } from "@/hooks/store/use-member";
import { useUserPermissions } from "@/hooks/store/user";
````
````old web/apps/web/core/components/project/card.tsx
  const { getUserDetails } = useMember();
````
````new web/apps/web/core/components/project/card.tsx
  const { getUserDetails } = useMember();
  const { allowPermissions } = useUserPermissions();
````
````old web/apps/web/core/components/project/card.tsx
  const hasAdminRole = project.member_role === EUserPermissions.ADMIN;
````
````new web/apps/web/core/components/project/card.tsx
  // its admins, and its members who are the workspace's admins (M3 design 3.4), as its settings take them
  const hasAdminRole = allowPermissions(
    [EUserPermissions.ADMIN],
    EUserPermissionsLevel.PROJECT,
    workspaceSlug,
    project.id
  );
````
````old web/apps/web/core/components/project/card.tsx
                      setDeleteProjectModal(true);
                    }}
````
````new web/apps/web/core/components/project/card.tsx
                      setDeleteProjectModal(true);
                    }}
                    aria-label="Delete"
````

- [ ] **Step 3: 静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/components/project/project-settings-member-defaults.test.tsx"
````
````new .oxlintrc.json
        "web/apps/web/core/components/project/project-settings-member-defaults.test.tsx",
        "web/apps/web/core/components/project/delete-project-modal.test.tsx",
        "web/apps/web/core/components/project/archive-restore-modal.test.tsx"
````

- [ ] **Step 4: 端到端**

`e2e/stories/project/p4-archive.spec.ts`（修改，4 处）：

````old e2e/stories/project/p4-archive.spec.ts
import {
  amidAnotherWorkspace,
````
````new e2e/stories/project/p4-archive.spec.ts
import {
  addProjectMembers,
  amidAnotherWorkspace,
  archiveProject,
  changeRole,
````
````old e2e/stories/project/p4-archive.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import type { Database } from "../../fixtures/db";
import { expect, test } from "../../fixtures/test";
````
````new e2e/stories/project/p4-archive.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import type { Database } from "../../fixtures/db";
import { answerTo, closedByEscape, registerOnboarded, sentHeld } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser } from "../../fixtures/workspace-pages";
````
````old e2e/stories/project/p4-archive.spec.ts
// P4, archiving, unarchiving and deleting a project (M3 design 2, 3.6,
// 3.19). The page version comes with the project's settings pages (P10).
````
````new e2e/stories/project/p4-archive.spec.ts
// P4, archiving, unarchiving and deleting a project (M3 design 2, 3.6, 3.19): through the API, and through the
// project's settings and the archived projects (7.6).
````
````old e2e/stories/project/p4-archive.spec.ts
  ).toBe(ops.id);
});

````
````new e2e/stories/project/p4-archive.spec.ts
  ).toBe(ops.id);
});

test("P4 (page): the admin archives a project from its settings, the dialog held until nerve answers, and it leaves his sidebar; a member of it who is the workspace's admin restores it from the archived projects; archived again, the admin deletes it there by its name, the dialog held too, and it is gone", async ({
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
  // wes is Web's member, and then acme's admin: nerve lets him archive, restore and delete Web (M3 design 3.4).
  const wesEmail = emailFor(testInfo, "wes");
  const wes = await registerOnboarded(api, wesEmail);
  await inviteAndAccept(api, admin.access_token, slug, { email: wesEmail, token: wes.access_token }, 15);
  const wesId = await accountId(api, wes.access_token);
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  await createProject(api, admin.access_token, slug, { name: "Ops", identifier: "OPS" });
  await addProjectMembers(api, admin.access_token, web.id, [{ member_id: wesId, role: 15 }]);
  await changeRole(api, admin.access_token, slug, wesId, 20);
  // Web has a row in each table its deletion writes: its label Bug too (expectProjectDeleted).
  await createLabel(api, admin.access_token, web.id, { name: "Bug" });

  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  const general = `/${slug}/settings/projects/${web.id}`;
  await page.goto(general);
  // The admin archives Web: the dialog cannot be closed, and the page stays, until nerve has archived it.
  await page.getByRole("button", { name: "Archive", exact: true }).click();
  const archive = await sentHeld(page, "POST", `/api/v0/projects/${web.id}/archive`, () =>
    page.getByRole("dialog").getByRole("button", { name: "Archive" }).click()
  );
  await expect(page.getByRole("dialog").getByRole("button", { name: "Cancel" })).toBeDisabled();
  expect(await closedByEscape(page)).toBe(false);
  expect(new URL(page.url()).pathname).toBe(general);
  expect((await archive.release()).status()).toBe(200);
  await expect(page).toHaveURL(`/${slug}/projects`);
  await expect(page.getByText("Web has been archived successfully")).toBeVisible();
  const sidebar = page.getByRole("complementary", { name: "Main sidebar" });
  await expect(sidebar.getByText("Ops", { exact: true })).toBeVisible();
  await expect(sidebar.getByText("Web", { exact: true })).toHaveCount(0);
  expect(await archiving(db, web.id)).toEqual({ archived: true, at_its_change: true, by: adminEmail });

  // wes restores it from the archived projects, which offer it him as they do its admins.
  const theOtherAdmin = await anotherBrowser(browser, baseURL ?? "", wes);
  const wesWatch = await watchPage(theOtherAdmin.page);
  await theOtherAdmin.page.goto(`/${slug}/projects/archives`);
  // the card's button: the card's menu, in a portal of its own, has one too
  await theOtherAdmin.page.getByRole("main").getByRole("button", { name: "Restore" }).click();
  const restored = await answerTo(theOtherAdmin.page, "POST", `/api/v0/projects/${web.id}/unarchive`, () =>
    theOtherAdmin.page.getByRole("dialog").getByRole("button", { name: "Restore" }).click()
  );
  expect(restored.status()).toBe(200);
  await expect(theOtherAdmin.page).toHaveURL(`/${slug}/projects`);
  await expect(theOtherAdmin.page.getByText("You can find Web in your projects.")).toBeVisible();
  expect(await archiving(db, web.id)).toEqual({ archived: false, at_its_change: true, by: wesEmail });
  expect([wesWatch.apiFailures, wesWatch.oldApiRequests, wesWatch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(theOtherAdmin.page, wesWatch, { warnings: [EMOJI_CHECK_WARNING] });
  await theOtherAdmin.close();

  // Archived again, Web is deleted from the archived projects, by its name and the words that confirm it; the dialog
  // cannot be closed until nerve has deleted it, and the page stays where it is, the archived projects, without Web.
  await archiveProject(api, admin.access_token, web.id);
  const archives = `/${slug}/projects/archives`;
  await page.goto(archives);
  await page.getByRole("main").getByRole("button", { name: "Delete" }).click();
  await page.locator("#projectName").fill("Web");
  await page.locator("#confirmDelete").fill("delete my project");
  const deleting = await sentHeld(page, "DELETE", `/api/v0/projects/${web.id}`, () =>
    page.getByRole("button", { name: "Delete project" }).click()
  );
  await expect(page.getByRole("dialog").getByRole("button", { name: "Cancel" })).toBeDisabled();
  expect(await closedByEscape(page)).toBe(false);
  expect((await deleting.release()).status()).toBe(204);
  await expect(page.getByRole("heading", { name: "Delete project" })).toHaveCount(0);
  await expect(page.getByText("Project deleted successfully.")).toBeVisible();
  await expect(page.getByRole("main").getByRole("button", { name: "Restore" })).toHaveCount(0);
  expect(new URL(page.url()).pathname).toBe(archives);
  await expectProjectDeleted(db, web.id, adminEmail);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  // Two loads: the settings, and the archived projects; the archiving's landing navigates within the app.
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING] });
});

````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 68 条规则、3 个例外，没有命中；web 的 oxlint 349 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 105 个全部通过。

- [ ] **Step 6: 提交**

```bash
git add .oxlintrc.json e2e/stories/project/p4-archive.spec.ts web/apps/web/core/components/project/archive-restore-modal.test.tsx web/apps/web/core/components/project/archive-restore-modal.tsx web/apps/web/core/components/project/card.tsx web/apps/web/core/components/project/delete-project-modal.test.tsx web/apps/web/core/components/project/delete-project-modal.tsx
```
```bash
git commit -m "feat(M3/P10): archiving, restoring and deleting a project follow nerve's answer in their session and wait for it

The deletion checks the values submitted, deletes through followInSession,
leaves only the deleted project's own page for the projects, and shows
nerve's reason when it refuses; archiving and restoring do the same. None
of the dialogs can be dismissed while its request is out. The archived
projects offer restoring and deleting to whom the settings offer
archiving: the project's admins and its members who are the workspace's
admins, as nerve allows. P4's page version.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p10.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T9.1` | 删除项目不论会话都跟进 | `delete-project-modal.test.tsx` | vitest |
| `T9.2` | 删除被拒绝时什么都不说 | `delete-project-modal.test.tsx` | vitest |
| `T9.3` | 不论表单填的是什么，删除都照样发出 | `delete-project-modal.test.tsx` | vitest |
| `T9.4` | 删除把不是这个项目的页面也带到项目列表 | `delete-project-modal.test.tsx`、故事 P4 | vitest；端到端 |
| `T9.5` | 删除在途时 Escape 关掉它 | 故事 P4 | 端到端 |
| `T9.6` | 删除在途时取消按钮仍可用 | 故事 P4 | 端到端 |
| `T9.7` | 归档和恢复不论会话都跟进 | `archive-restore-modal.test.tsx` | vitest |
| `T9.8` | 归档被拒绝时什么都不说 | `archive-restore-modal.test.tsx` | vitest |
| `T9.9` | 归档在途时 Escape 关掉它 | 故事 P4 | 端到端 |
| `T9.10` | 归档在途时取消按钮仍可用 | 故事 P4 | 端到端 |
| `T9.11` | 已归档的项目只向项目管理员提供恢复和删除，不向是工作区管理员的项目成员提供 | 故事 P4 | 端到端 |
| `T9.12` | 已归档卡片的删除按钮没有名字 | 故事 P4 | 端到端 |
| `T9.13` | 在删除项目的测试里写一个非空断言 | oxlint（`check:lint`） | 静态 |

---

### Task 10: 侧边栏拖动项目和项目页头的标签栏；P8 的页面版本

**Files:**
- Modify: `.oxlintrc.json`、`e2e/fixtures/settings-pages.ts`、`e2e/stories/project/p8-project-preferences.spec.ts`、`web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx`、`web/apps/web/core/components/navigation/tab-navigation-root.tsx`、`web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx`、`web/apps/web/core/components/navigation/use-tab-preferences.test.ts`、`web/apps/web/core/components/navigation/use-tab-preferences.ts`、`web/apps/web/core/components/workspace/sidebar/use-project-drop.test.ts`、`web/apps/web/core/components/workspace/sidebar/use-project-drop.ts`

**Interfaces:**
- Produces（spec 2.10；M3 设计 2 的 P8，3.18，7.1、7.6；总体设计 7.7 的 PF-M2：移动的位置和标签栏的修改在轮到它时由 store 算，P8b 已做；P8b review 第 6 节的 T5-a；P9 spec 第 5 节 P10 一行：`use-tab-preferences.ts` 的拒绝进 `console.error`）：
  - `useProjectDrop`（`workspace/sidebar/use-project-drop.ts`，整个文件）：移动经 `followInSession`，失败时经 `useRefusalToast` 说原因（原来是固定的"something went wrong"，不论会话）；没有到达 nerve 的失败说 `errors.unknown`。
  - `useTabPreferences(projectId): TTabPreferencesHook`（整个文件）：交回 `{ navigation, changes }`；`navigation` 在设置取到之前是 nerve 的默认（工作项，没有收起的），`changes`（`TTabChanges`：`toggleDefault`、`hide`、`show`）在那之前是 `undefined`：store 对它没有的设置不发修改，页头也就不提供（T5-a：原来的菜单在设置到达之前就可用，点了什么都不发）。每个修改经 `followInSession`：设为默认时提示（Plane 的"Default tab updated successfully."），被拒绝时经 `useRefusalToast`（原来提示固定的文字，再进 `console.error`）。
  - 页头：`tab-navigation-root.tsx` 把 `changes` 交给每个标签和"更多"；`tab-navigation-visible-item.tsx`（整个文件）只在有 `changes` 时有右键菜单（设为默认、收进"更多"）；`tab-navigation-overflow-menu.tsx` 只在有 `changes` 时有"显示"和"设为默认"的按钮。
  - 静态检查：`.oxlintrc.json` 的非空断言的范围加上 `use-project-drop.ts` 和它的测试、`use-tab-preferences.test.ts`（`use-tab-preferences.ts` 已在其中）。
- e2e 的共用部分：`e2e/fixtures/settings-pages.ts` 的 `shownWithin(locator)`（一秒之内是否显示：扣住页面等的东西时，一直不该显示的东西在整个期间都不显示；只把一秒用完算作没显示，定位到几个元素等其余的失败照常抛出）。

**Tests:**
- vitest：`use-project-drop.test.ts`：`says why a move failed: nerve's reason, or none for one the store did not send`（改写原来的"失败时提示"）；`says nothing when a move $settles after another tab moved this one`。`use-tab-preferences.test.ts`（整个文件）：`gives nerve's default tab bar and no change until the caller's is fetched, then his and its changes`；`makes each change to the tab bar nerve last answered, in the change's turn`；`says a new default once nerve has it, and nerve's reason for refusing a change`；`says nothing when a change $settles after another tab moved this one`。
- 端到端：`P8 (page): the admin makes modules the tab Web opens on and moves its views under more, by the menus of its header's tabs, which offer neither until nerve has given his tab bar; he drags his third project first in his sidebar; each lasts after a refresh`（工作区的导航设为标签式，Web 打开模块和视图；扣住设置的取数时右键"Modules"没有菜单（`shownWithin`），放行之后有（右键再看，五秒为限的 `toPass`）；设为默认、收起视图的请求体经 `sentTo`；按住 Ops 的拖动柄拖到 Docs 上（`dragTo` 拖不动：指针离开时柄就隐藏，改用逐步的鼠标操作），发 `{ sort_order: 35535 }`；刷新之后顺序、默认标签、收起的视图都在，数据库中两个项目的显示设置；工作项页请求 M4 的筛选的旧地址，两次 404 是预期的）。

- [ ] **Step 1: 侧边栏的移动**

`web/apps/web/core/components/workspace/sidebar/use-project-drop.test.ts`（修改，7 处）：

````old web/apps/web/core/components/workspace/sidebar/use-project-drop.test.ts
import type { Project } from "@nerve/api-client";
````
````new web/apps/web/core/components/workspace/sidebar/use-project-drop.test.ts
import type { Project } from "@nerve/api-client";
import { heldChange, lateSettlings, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
````
````old web/apps/web/core/components/workspace/sidebar/use-project-drop.test.ts
// What a drop of a project in the caller's sidebar asks the project store (M3 design 3.18): the hook runs as a plain
// function, outside React, with a stand-in for the project store, whose moves record their arguments.
````
````new web/apps/web/core/components/workspace/sidebar/use-project-drop.test.ts
// What a drop of a project in the caller's sidebar asks the project store (M3 design 3.18), and what the page says of
// it (7.1): the hook runs as a plain function, outside React, with a stand-in for the project store, whose moves record
// their arguments. Its session is fake-tab.ts's.
````
````old web/apps/web/core/components/workspace/sidebar/use-project-drop.test.ts
  updateProjectSortOrder: vi.fn((_project: Project, _droppedOnId: string | undefined, _atEnd: boolean) =>
    Promise.resolve()
````
````new web/apps/web/core/components/workspace/sidebar/use-project-drop.test.ts
  updateProjectSortOrder: vi.fn(
    (_project: Project, _droppedOnId: string | undefined, _atEnd: boolean): Promise<unknown> => Promise.resolve()
````
````old web/apps/web/core/components/workspace/sidebar/use-project-drop.test.ts
  }),
}));
````
````new web/apps/web/core/components/workspace/sidebar/use-project-drop.test.ts
  }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
````
````old web/apps/web/core/components/workspace/sidebar/use-project-drop.test.ts
beforeEach(() => {
````
````new web/apps/web/core/components/workspace/sidebar/use-project-drop.test.ts
beforeEach(() => {
  signedIn();
````
````old web/apps/web/core/components/workspace/sidebar/use-project-drop.test.ts
  it("shows a toast when the move fails", async () => {
````
````new web/apps/web/core/components/workspace/sidebar/use-project-drop.test.ts
  it("says why a move failed: nerve's reason, or none for one the store did not send", async () => {
    store.updateProjectSortOrder.mockImplementationOnce(() => Promise.reject(refusal(404, "project.not_found")));
````
````old web/apps/web/core/components/workspace/sidebar/use-project-drop.test.ts
    await vi.waitFor(() => expect(toasts).toHaveLength(1));
    expect(toasts).toEqual([{ type: "error", title: "error", message: "something_went_wrong" }]);
````
````new web/apps/web/core/components/workspace/sidebar/use-project-drop.test.ts
    useProjectDrop()(web.id, docs.id, false);
    await pageSettled();
    expect(toasts).toEqual([
      { type: "error", title: "toast.error", message: "errors.project_not_found" },
      { type: "error", title: "toast.error", message: "errors.unknown" },
    ]);
  });

  it.each(lateSettlings)("says nothing when a move $settles after another tab moved this one", async ({ settle }) => {
    const move = heldChange<undefined>();
    store.updateProjectSortOrder.mockReturnValueOnce(move.sent);
    useProjectDrop()(docs.id, web.id, false);
    switchAccount();
    settle(move);
    await pageSettled();
    expect(toasts).toEqual([]);
````

`web/apps/web/core/components/workspace/sidebar/use-project-drop.ts`（整个文件，34 行）：

````whole web/apps/web/core/components/workspace/sidebar/use-project-drop.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useParams } from "react-router";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";

/**
 * What a drop of a project in the caller's sidebar does, in its list and in the extended one: the project dropped
 * moves before the one it was dropped on, or after his last at the end of the list (ProjectStore.updateProjectSortOrder
 * reckons the place in the change's turn); a drop on itself or on nothing moves nothing. A move that fails says why,
 * followed in the session that sent it (M3 design 7.1).
 */
export function useProjectDrop() {
  const { workspaceSlug } = useParams();
  const toastRefusal = useRefusalToast();
  const { getProjectById, updateProjectSortOrder } = useProject();

  return (sourceId: string | undefined, destinationId: string | undefined, shouldDropAtEnd: boolean) => {
    if (!sourceId || !destinationId || !workspaceSlug) return;
    if (sourceId === destinationId) return;

    const source = getProjectById(sourceId);
    if (source)
      void followInSession(() => updateProjectSortOrder(source, destinationId, shouldDropAtEnd), {
        failed: toastRefusal,
      });
  };
}
````

- [ ] **Step 2: 标签栏**

`web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx`（修改，7 处）：

````old web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
import type { ProjectNavigation, ProjectTab } from "@nerve/api-client";
````
````new web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
import type { ProjectNavigation } from "@nerve/api-client";
````
````old web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
import type { TNavigationItem } from "./tab-navigation-root";
````
````new web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
import type { TNavigationItem } from "./tab-navigation-root";
import type { TTabChanges } from "./use-tab-preferences";
````
````old web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
  onToggleDefault: (tabKey: ProjectTab) => void;
  onShow: (tabKey: ProjectTab) => void;
````
````new web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
  /** The tab bar's changes; none until the caller's tab bar is fetched, and the menu then offers none. */
  changes: TTabChanges | undefined;
````
````old web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
export function TabNavigationOverflowMenu({ overflowItems, isActive, navigation, onToggleDefault, onShow }: Props) {
````
````new web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
export function TabNavigationOverflowMenu({ overflowItems, isActive, navigation, changes }: Props) {
````
````old web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
                {isHidden && (
````
````new web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
                {changes && isHidden && (
````
````old web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
                      onShow(item.key);
````
````new web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
                      changes.show(item.key);
````
````old web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
                <Tooltip label={isDefault ? "Clear default" : "Set as default"}>
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      e.preventDefault();
                      onToggleDefault(item.key);
                    }}
                    className={cn(
                      "invisible rounded-sm p-1 text-tertiary transition-colors group-hover/menu-item:visible hover:text-primary",
                      {
                        visible: isDefault,
                      }
                    )}
                    title={isDefault ? "Clear default" : "Set as default"}
                  >
                    <DefaultTabOutline className="size-3" />
                  </button>
                </Tooltip>
````
````new web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx
                {changes && (
                  <Tooltip label={isDefault ? "Clear default" : "Set as default"}>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        e.preventDefault();
                        changes.toggleDefault(item.key);
                      }}
                      className={cn(
                        "invisible rounded-sm p-1 text-tertiary transition-colors group-hover/menu-item:visible hover:text-primary",
                        {
                          visible: isDefault,
                        }
                      )}
                      title={isDefault ? "Clear default" : "Set as default"}
                    >
                      <DefaultTabOutline className="size-3" />
                    </button>
                  </Tooltip>
                )}
````

`web/apps/web/core/components/navigation/tab-navigation-root.tsx`（修改，3 处）：

````old web/apps/web/core/components/navigation/tab-navigation-root.tsx
  const { navigation, handleToggleDefaultTab, handleHideTab, handleShowTab } = useTabPreferences(projectId);
````
````new web/apps/web/core/components/navigation/tab-navigation-root.tsx
  const { navigation, changes } = useTabPreferences(projectId);
````
````old web/apps/web/core/components/navigation/tab-navigation-root.tsx
                  onToggleDefault={handleToggleDefaultTab}
                  onHide={handleHideTab}
````
````new web/apps/web/core/components/navigation/tab-navigation-root.tsx
                  changes={changes}
````
````old web/apps/web/core/components/navigation/tab-navigation-root.tsx
                onToggleDefault={handleToggleDefaultTab}
                onShow={handleShowTab}
````
````new web/apps/web/core/components/navigation/tab-navigation-root.tsx
                changes={changes}
````

`web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx`（整个文件，88 行）：

````whole web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link } from "react-router";
import type { ProjectNavigation } from "@nerve/api-client";
import { DefaultTabOutline, UnpinOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { ContextMenu } from "@nerve/propel/context-menu";
import { TabNavigationItem } from "@nerve/propel/tab-navigation";
// local imports
import type { TNavigationItem } from "./tab-navigation-root";
import type { TTabChanges } from "./use-tab-preferences";

export type TTabNavigationVisibleItemProps = {
  item: TNavigationItem;
  isActive: boolean;
  navigation: ProjectNavigation;
  /** The tab bar's changes; none until the caller's tab bar is fetched, and the tab then has no menu. */
  changes: TTabChanges | undefined;
  itemRef?: (el: HTMLDivElement | null) => void;
};

/**
 * Individual visible tab navigation item with context menu
 * Handles right-click actions for setting default and hiding tabs
 */
export function TabNavigationVisibleItem({
  item,
  isActive,
  navigation,
  changes,
  itemRef,
}: TTabNavigationVisibleItemProps) {
  const { t } = useTranslation();
  const isDefault = item.key === navigation.default_tab;
  const link = (
    <Link key={`${item.key}-${isActive ? "active" : "inactive"}`} to={item.href}>
      <TabNavigationItem isActive={isActive}>
        <span>{t(item.i18n_key)}</span>
      </TabNavigationItem>
    </Link>
  );

  return (
    <div className="relative flex h-full items-center transition-all duration-300">
      {isActive && (
        <span className="absolute bottom-0 left-1/2 h-0.5 w-[80%] -translate-x-1/2 rounded-t-md bg-(--text-color-icon-primary) transition-all duration-300" />
      )}
      <div key={`${item.key}-measure`} ref={itemRef}>
        {changes ? (
          <ContextMenu>
            <ContextMenu.Trigger>{link}</ContextMenu.Trigger>
            <ContextMenu.Portal>
              <ContextMenu.Content positionerClassName="z-30">
                <ContextMenu.Item
                  onClick={(e) => {
                    e.stopPropagation();
                    changes.toggleDefault(item.key);
                  }}
                  className="flex cursor-pointer items-center gap-2 text-secondary transition-colors"
                >
                  <DefaultTabOutline className="size-3 shrink-0" />
                  <span className="text-11">{isDefault ? "Clear default" : "Set as default"}</span>
                </ContextMenu.Item>
                <ContextMenu.Item
                  onClick={(e) => {
                    e.stopPropagation();
                    changes.hide(item.key);
                  }}
                  className="flex cursor-pointer items-center gap-2 text-secondary transition-colors"
                >
                  <UnpinOutline className="size-3 shrink-0" />
                  <span className="text-11">Hide in more menu</span>
                </ContextMenu.Item>
              </ContextMenu.Content>
            </ContextMenu.Portal>
          </ContextMenu>
        ) : (
          link
        )}
      </div>
    </div>
  );
}
````

`web/apps/web/core/components/navigation/use-tab-preferences.test.ts`（整个文件，93 行）：

````whole web/apps/web/core/components/navigation/use-tab-preferences.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectNavigation } from "@nerve/api-client";
import { emptyStores, stores } from "@/hooks/store/fake-store-hooks";
import { heldChange, lateSettlings, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import type { NavigationChange } from "@/store/project/preferences.store";
import { useTabPreferences } from "./use-tab-preferences";

// The tab bar the project's header and the sidebar show (M3 design 3.18), and its changes (7.1): nerve's
// ProjectNavigation, the caller's once the project wrapper has fetched it. The hook runs as a plain function, outside
// React, with fake-store-hooks.ts for the tab bar store, whose changes record their arguments. Its session is
// fake-tab.ts's.

const preferences = vi.hoisted(() => ({
  updateNavigation: vi.fn((_projectId: string, _change: NavigationChange): Promise<unknown> => Promise.resolve()),
}));
vi.mock("@/hooks/store/use-project-preferences", async () => {
  const fake = await import("@/hooks/store/fake-store-hooks");
  return {
    useProjectPreferences: () => ({ ...fake.useProjectPreferences(), updateNavigation: preferences.updateNavigation }),
  };
});
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** The caller's tab bar in Web as nerve last answered it. */
const held: ProjectNavigation = { default_tab: "cycles", hide_in_more_menu: ["views"] };

/** Each change the store was asked for: its project, and what it makes of the tab bar held. */
const made = () => preferences.updateNavigation.mock.calls.map(([projectId, change]) => [projectId, change(held)]);

beforeEach(() => {
  signedIn();
  emptyStores();
  stores.navigations = { "p-web": held };
  preferences.updateNavigation.mockClear();
  toasts.length = 0;
});

describe("useTabPreferences", () => {
  it("gives nerve's default tab bar and no change until the caller's is fetched, then his and its changes", () => {
    stores.navigations = {};
    expect(useTabPreferences("p-web")).toEqual({
      navigation: { default_tab: "work_items", hide_in_more_menu: [] },
      changes: undefined,
    });
    stores.navigations = { "p-web": held };
    const { navigation, changes } = useTabPreferences("p-web");
    expect([navigation, changes && Object.keys(changes)]).toEqual([held, ["toggleDefault", "hide", "show"]]);
  });

  it("makes each change to the tab bar nerve last answered, in the change's turn", () => {
    const { changes } = useTabPreferences("p-web");
    changes?.toggleDefault("modules");
    changes?.hide("modules");
    changes?.show("views");
    expect(made()).toEqual([
      ["p-web", { default_tab: "modules", hide_in_more_menu: ["views"] }],
      ["p-web", { default_tab: "cycles", hide_in_more_menu: ["views", "modules"] }],
      ["p-web", { default_tab: "cycles", hide_in_more_menu: [] }],
    ]);
  });

  it("says a new default once nerve has it, and nerve's reason for refusing a change", async () => {
    const { changes } = useTabPreferences("p-web");
    changes?.toggleDefault("modules");
    changes?.hide("modules");
    preferences.updateNavigation.mockImplementationOnce(() => Promise.reject(refusal(422, "validation_failed")));
    changes?.show("views");
    await pageSettled();
    expect(toasts).toEqual([
      { type: "success", title: "Success!", message: "Default tab updated successfully." },
      { type: "error", title: "toast.error", message: "errors.validation_failed" },
    ]);
  });

  it.each(lateSettlings)("says nothing when a change $settles after another tab moved this one", async ({ settle }) => {
    const change = heldChange<undefined>();
    preferences.updateNavigation.mockReturnValueOnce(change.sent);
    useTabPreferences("p-web").changes?.toggleDefault("modules");
    switchAccount();
    settle(change);
    await pageSettled();
    expect(toasts).toEqual([]);
  });
});
````

`web/apps/web/core/components/navigation/use-tab-preferences.ts`（整个文件，63 行）：

````whole web/apps/web/core/components/navigation/use-tab-preferences.ts
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ProjectNavigation, ProjectTab } from "@nerve/api-client";
import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
import { useProjectPreferences } from "@/hooks/store/use-project-preferences";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { followInSession } from "@/lib/in-session";
import type { NavigationChange } from "@/store/project/preferences.store";
import { DEFAULT_NAVIGATION, hideTab, showTab, toggleDefaultTab } from "./tab-navigation-utils";

/** The changes of the caller's tab bar that the header's controls make, each of the tab it is given. */
export type TTabChanges = {
  toggleDefault: (tabKey: ProjectTab) => void;
  hide: (tabKey: ProjectTab) => void;
  show: (tabKey: ProjectTab) => void;
};

export type TTabPreferencesHook = {
  navigation: ProjectNavigation;
  changes: TTabChanges | undefined;
};

/**
 * The caller's tab bar in the project's header (ProjectPreferences.navigation): the tab the project opens on and the
 * tabs under "more", shown as nerve's default (work items, none hidden) until the project wrapper has fetched his; and
 * its changes, none until then, so that the header offers none: the store makes a change to the tab bar nerve last
 * answered, and sends none before it has one (made to the default, it would replace his whole). A change shows once
 * nerve has answered it, and the page follows it in the session that sent it (M3 design 7.1): a new default says so; a
 * refusal says nerve's reason, the tab bar as it was.
 *
 * @param projectId - The project ID
 * @returns The caller's tab bar, nerve's default until it is fetched, and its changes once it is
 */
export const useTabPreferences = (projectId: string): TTabPreferencesHook => {
  const { getNavigation, updateNavigation } = useProjectPreferences();
  const toastRefusal = useRefusalToast();
  const fetched = getNavigation(projectId);

  const send = (change: NavigationChange, done?: () => void) =>
    void followInSession(() => updateNavigation(projectId, change), { done, failed: toastRefusal });

  return {
    navigation: fetched ?? DEFAULT_NAVIGATION,
    changes: fetched
      ? {
          toggleDefault: (tabKey) =>
            send(toggleDefaultTab(tabKey), () =>
              setToast({
                type: TOAST_TYPE.SUCCESS,
                title: "Success!",
                message: "Default tab updated successfully.",
              })
            ),
          hide: (tabKey) => send(hideTab(tabKey)),
          show: (tabKey) => send(showTab(tabKey)),
        }
      : undefined,
  };
};
````

- [ ] **Step 3: 静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/components/project/archive-restore-modal.test.tsx"
````
````new .oxlintrc.json
        "web/apps/web/core/components/project/archive-restore-modal.test.tsx",
        "web/apps/web/core/components/navigation/use-tab-preferences.test.ts",
        "web/apps/web/core/components/workspace/sidebar/use-project-drop.ts",
        "web/apps/web/core/components/workspace/sidebar/use-project-drop.test.ts"
````

- [ ] **Step 4: 端到端**

`e2e/fixtures/settings-pages.ts`（修改，1 处）：

````old e2e/fixtures/settings-pages.ts
    .waitFor({ state: "attached", timeout: 1_000 })
    .then(() => true, timedOut);
````
````new e2e/fixtures/settings-pages.ts
    .waitFor({ state: "attached", timeout: 1_000 })
    .then(() => true, timedOut);
}

/**
 * Resolves with whether locator, one element, shows within a second: what must not show while the test holds what its
 * page waits for stays hidden the whole time, where a check made at once could pass before the page has rendered it.
 * Only the second running out means it did not show (timedOut); any other failure, such as a locator that matches
 * several elements, rejects.
 */
export async function shownWithin(locator: Locator): Promise<boolean> {
  return locator.waitFor({ state: "visible", timeout: 1_000 }).then(() => true, timedOut);
````

`e2e/stories/project/p8-project-preferences.spec.ts`（修改，4 处）：

````old e2e/stories/project/p8-project-preferences.spec.ts
  amidAnotherWorkspace,
````
````new e2e/stories/project/p8-project-preferences.spec.ts
  amidAnotherWorkspace,
  changeProject,
````
````old e2e/stories/project/p8-project-preferences.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import type { Database } from "../../fixtures/db";
````
````new e2e/stories/project/p8-project-preferences.spec.ts
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import type { Database } from "../../fixtures/db";
import { answerTo, holdAnswer, registerOnboarded, sentTo, shownWithin } from "../../fixtures/settings-pages";
````
````old e2e/stories/project/p8-project-preferences.spec.ts
// P8, a member's display settings in a project (M3 design 2, 3.18). The
// page version comes with the project's header and the sidebar (P10).
````
````new e2e/stories/project/p8-project-preferences.spec.ts
// P8, a member's display settings in a project (M3 design 2, 3.18): through the API, and through the project's header
// and the sidebar (7.6).
````
````old e2e/stories/project/p8-project-preferences.spec.ts
  ]);
});

````
````new e2e/stories/project/p8-project-preferences.spec.ts
  ]);
});

test("P8 (page): the admin makes modules the tab Web opens on and moves its views under more, by the menus of its header's tabs, which offer neither until nerve has given his tab bar; he drags his third project first in his sidebar; each lasts after a refresh", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  // His sidebar: Docs 45535, Web 55535, Ops 65535. Web shows its modules and its views: its header has their tabs.
  const ops = await createProject(api, admin.access_token, slug, { name: "Ops", identifier: "OPS" });
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  const docs = await createProject(api, admin.access_token, slug, { name: "Docs", identifier: "DOCS" });
  const shown = await changeProject(api, admin.access_token, web.id, { module_view: true, issue_views_view: true });
  expect(shown.status).toBe(200);
  // His projects' navigation is the tabbed one, whose project pages have a header of tabs (W8).
  const tabbed = await api.PATCH("/api/v0/me/workspaces/{slug}/preferences", {
    params: { path: { slug } },
    body: { navigation_control_preference: "TABBED" },
    headers: bearer(admin.access_token),
  });
  expect(tabbed.response.status, `tabbed: ${JSON.stringify(tabbed.error)}`).toBe(200);

  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  const settings = `/api/v0/me/projects/${web.id}/preferences`;
  /** The tab of name in Web's header: a link of the page outside the sidebar, which has links of the same names. */
  const tab = (name: string) =>
    page.locator("xpath=//a[not(ancestor::aside)]").filter({ hasText: new RegExp(`^${name}$`) });
  const setAsDefault = page.getByRole("menuitem", { name: "Set as default" });
  // Until nerve has given his tab bar a tab has no menu: a change made to nerve's default would replace his tab bar.
  const release = await holdAnswer(page, "GET", settings);
  await page.goto(`/${slug}/projects/${web.id}/issues`);
  await tab("Modules").click({ button: "right" });
  expect(await shownWithin(setAsDefault)).toBe(false);
  expect((await answerTo(page, "GET", settings, release)).status()).toBe(200);
  await expect(async () => {
    await tab("Modules").click({ button: "right" });
    await expect(setAsDefault).toBeVisible({ timeout: 1_000 });
  }).toPass({ timeout: 5_000 });
  const defaulted = await sentTo(page, "PATCH", settings, () => setAsDefault.click());
  expect([defaulted.answer.status(), defaulted.body]).toEqual([
    200,
    { navigation: { default_tab: "modules", hide_in_more_menu: [] } },
  ]);
  await expect(page.getByText("Default tab updated successfully.")).toBeVisible();
  // The second change is made to the tab bar nerve answered the first with.
  await tab("Views").click({ button: "right" });
  const hidden = await sentTo(page, "PATCH", settings, () =>
    page.getByRole("menuitem", { name: "Hide in more menu" }).click()
  );
  expect([hidden.answer.status(), hidden.body]).toEqual([
    200,
    { navigation: { default_tab: "modules", hide_in_more_menu: ["views"] } },
  ]);
  await expect(tab("Views")).toHaveCount(0);

  // He drags Ops, third in his sidebar, onto Docs, the first, by its handle: Ops goes a step before Docs.
  const sidebar = page.getByRole("complementary", { name: "Main sidebar" });
  const item = (id: string) => sidebar.locator(`[id="sidebar-${id}-JOINED"]`);
  const order = sidebar.locator('[id^="sidebar-"][id$="-JOINED"]');
  await expect(order).toHaveText([/Docs/, /Web/, /Ops/]);
  await item(ops.id).hover();
  // the handle shows while the pointer is over Ops: the drag starts there, as a hand's would, then goes to Docs
  await item(ops.id).locator("button").first().hover();
  await page.mouse.down();
  await item(ops.id).hover({ position: { x: 60, y: 8 } });
  const moved = await sentTo(page, "PATCH", `/api/v0/me/projects/${ops.id}/preferences`, async () => {
    await item(docs.id).hover();
    await page.mouse.up();
  });
  expect([moved.answer.status(), moved.body]).toEqual([200, { sort_order: 35535 }]);
  await expect(order).toHaveText([/Ops/, /Docs/, /Web/]);

  // After a refresh: the same order; Web opens on its modules, and its views are under more.
  await page.reload();
  await expect(order).toHaveText([/Ops/, /Docs/, /Web/]);
  await expect(item(web.id).locator("a").first()).toHaveAttribute("href", `/${slug}/projects/${web.id}/modules`);
  // the header's tabs show (the first Modules: the header measures the tabs it shows by hidden copies of them)
  await expect(tab("Modules").first()).toBeVisible();
  await expect(tab("Views")).toHaveCount(0);
  expect(await stored(db, web.id, adminEmail)).toEqual([
    {
      preferences: { navigation: { default_tab: "modules", hide_in_more_menu: ["views"] } },
      sort_order: 55535,
      by: adminEmail,
    },
  ]);
  expect(await stored(db, ops.id, adminEmail)).toEqual([
    {
      preferences: { navigation: { default_tab: "work_items", hide_in_more_menu: [] } },
      sort_order: 35535,
      by: adminEmail,
    },
  ]);

  // The work items' page asks Plane's address of the filters, M4's (P8b spec §5), at each of its two loads.
  const filters = `GET /api/workspaces/${slug}/projects/${web.id}/user-properties/`;
  await expect.poll(() => watch.apiFailures).toEqual([`404 ${filters}`, `404 ${filters}`]);
  expect([watch.cspViolations, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [filters, filters], []]);
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING],
    errors: [
      "Failed to load resource: the server responded with a status of 404 (Not Found)",
      "Failed to load resource: the server responded with a status of 404 (Not Found)",
    ],
  });
});

````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 68 条规则、3 个例外，没有命中；web 的 oxlint 349 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 106 个全部通过。

- [ ] **Step 6: 提交**

```bash
git add .oxlintrc.json e2e/fixtures/settings-pages.ts e2e/stories/project/p8-project-preferences.spec.ts web/apps/web/core/components/navigation/tab-navigation-overflow-menu.tsx web/apps/web/core/components/navigation/tab-navigation-root.tsx web/apps/web/core/components/navigation/tab-navigation-visible-item.tsx web/apps/web/core/components/navigation/use-tab-preferences.test.ts web/apps/web/core/components/navigation/use-tab-preferences.ts web/apps/web/core/components/workspace/sidebar/use-project-drop.test.ts web/apps/web/core/components/workspace/sidebar/use-project-drop.ts
```
```bash
git commit -m "feat(M3/P10): the sidebar's moves and the tab bar's changes follow nerve's answer in their session; the header offers none before nerve has the tab bar

A move of a project in the sidebar and each change of the header's tab
bar go through followInSession, and a refusal says nerve's reason instead
of a fixed string or a console.error. useTabPreferences gives the tab
bar's changes only once nerve has given the caller's tab bar, and the
header's tabs and its more menu offer none until then. P8's page version.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p10.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T10.1` | 侧边栏移动的失败不论会话都提示 | `use-project-drop.test.ts` | vitest |
| `T10.2` | 侧边栏的移动被拒绝时什么都不说 | `use-project-drop.test.ts` | vitest |
| `T10.3` | 标签栏的修改不论会话都跟进 | `use-tab-preferences.test.ts` | vitest |
| `T10.4` | 页头在 nerve 给出标签栏的设置之前就提供修改 | `use-tab-preferences.test.ts`、故事 P8 | vitest；端到端 |
| `T10.5` | 标签栏的修改被拒绝时什么都不说 | `use-tab-preferences.test.ts` | vitest |
| `T10.6` | 新的默认标签不提示 | `use-tab-preferences.test.ts`、故事 P8 | vitest；端到端 |
| `T10.7` | 在侧边栏拖放的 hook 里写一个非空断言 | oxlint（`check:lint`） | 静态 |

---

### Task 11: 文档：README 的"前端"一节、前端改动清单和两份交接的处理结果

**Files:**
- Modify: `README.md`、`docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md`、`docs/v0/M3-workspace-project/handoffs/M2-closeout.md`、`docs/v0/frontend-changes.md`

**Interfaces:**
- Produces（spec 2.11；M3 设计 3.20 中 P10 的行；M2 收尾交接第 10、11、14 节；M1-P4 交接）：
  - `README.md` 的"前端"一节：项目的页面一句改写为能用的项目页面（M3/P10 起）：项目列表和已归档的项目、创建项目、包装层的三种界面、项目设置的 general、members、features、automations、侧边栏的拖动和项目页头的标签栏；状态、标签的设置页和个人主页在 M3/P11，其余的由 M4–M8 对接。
  - `docs/v0/frontend-changes.md`：3.1 中 M3 一行的状态（工作区的页面 M3/P9、项目的页面 M3/P10 已对接，状态、标签的设置页在 M3/P11）；3.2 "所有处理接口错误的地方"一行写明 P10 改到的页面；"页面跟进修改的结果"一行写 P9、P10；新增八行（表情选择器的数据、下拉框、复制到剪贴板、创建项目、已归档的项目、项目成员页、项目的成员默认值、项目页头的标签栏），都是"已完成，M3/P10"。
  - `docs/v0/M3-workspace-project/handoffs/M2-closeout.md`：加"处理结果（M3/P10）"：第 10 节关闭，第 11 节关闭（项目 general 页的时区），第 14 节代码一侧完成（关闭条件中的 `member-options` 一项改为成员下拉框：`member-options.tsx` 已删除），逐个调用方的浏览器核对是 P10 评审的 C7、C8，之后关闭；状态保持 `open`（第 12 节等 M3 的收尾）。
  - `docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md`：加"处理结果（M3/P10）"：离开项目的顺序（nerve 做完之后才显示工作区的项目，spec 2.7）；保留名在 P8a spec 第 7 节；状态保持 `open` 到 M3 的收尾。
- 控制者的浏览器核对和评审不在本 plan 中（M3 设计 9.7、12 节 P10 的最后一个任务）。

**Tests:** 没有新的测试：本 Task 只改文档。

- [ ] **Step 1: 文档**

`README.md`（修改，1 处）：

````old README.md
- **能用的页面**（M3/P9 起）：登录、注册（注册关闭时，凭邀请链接仍可注册）；邀请链接的页面 `/workspace-invitations`（没登录时显示工作区和角色，登录后接受或忽略）；新手引导（资料一步；还没有工作区的人接着创建工作区，再邀请成员（规模选"Just myself"时创建之后就完成），发出邀请之后列出每个邀请的链接供复制，v0 不发邮件；已经有工作区的人资料一步之后就完成）；登录后的落点（上次打开的工作区；它不在列表中时，最早创建的那个；一个都没有时 `/create-workspace`）；创建工作区；工作区首页（问候、"快速入门指南"，第一次进入时的导览）；侧边栏的工作区菜单（切换、创建工作区、退出）和"项目导航"对话框；工作区设置的 general（名称、规模、时区，删除工作区）和 members（成员、改角色、移出、离开；管理员还有邀请：批量邀请、复制链接、改角色、删除）；个人设置的四个标签页（general、preferences、security、api-tokens，停用账户在 general 页）。项目的页面和个人主页在 M3/P10、P11 对接；工作项、迭代、模块、视图、收藏、通知、Webhook 等页面还在调用 Plane 的接口，Nerve 返回 404，由 M4–M8 对接。
````
````new README.md
- **能用的页面**（M3/P9 起）：登录、注册（注册关闭时，凭邀请链接仍可注册）；邀请链接的页面 `/workspace-invitations`（没登录时显示工作区和角色，登录后接受或忽略）；新手引导（资料一步；还没有工作区的人接着创建工作区，再邀请成员（规模选"Just myself"时创建之后就完成），发出邀请之后列出每个邀请的链接供复制，v0 不发邮件；已经有工作区的人资料一步之后就完成）；登录后的落点（上次打开的工作区；它不在列表中时，最早创建的那个；一个都没有时 `/create-workspace`）；创建工作区；工作区首页（问候、"快速入门指南"，第一次进入时的导览）；侧边栏的工作区菜单（切换、创建工作区、退出）和"项目导航"对话框；工作区设置的 general（名称、规模、时区，删除工作区）和 members（成员、改角色、移出、离开；管理员还有邀请：批量邀请、复制链接、改角色、删除）；个人设置的四个标签页（general、preferences、security、api-tokens，停用账户在 general 页）。项目的页面（M3/P10 起）：项目列表（卡片复制项目的链接；看得到而还不是成员的项目，卡片上加入）和已归档的项目（恢复、删除）；创建项目（负责人从工作区的管理员和成员中选；表情和图标的选择器，表情的数据由 Nerve 自己提供）；打开项目的地址时，不是成员的看到"加入项目"，看不到的看到"找不到项目"，已归档的看到它已归档；项目设置的 general（名称、标识、说明、可见性、时区、图标；归档、删除）、members（添加、改角色、移出、离开；负责人、默认负责人、访客可见全部）、features（迭代、模块、视图、收集箱的开关）和 automations（自动归档）；侧边栏拖动项目的顺序；标签式导航下项目页头的标签栏（默认打开的标签页、收进"更多"）。状态、标签的设置页和个人主页在 M3/P11 对接；工作项、迭代、模块、视图、收藏、通知、Webhook 等页面还在调用 Plane 的接口，Nerve 返回 404，由 M4–M8 对接。
````

`docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md`（修改，1 处）：

````old docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md
来源：[M3/P1 spec](../specs/P1-platform.md) 第 7 节。

````
````new docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md
来源：[M3/P1 spec](../specs/P1-platform.md) 第 7 节。

## 处理结果（M3/P10）

- **离开项目的顺序**（完成）：先等接口成功，再回到项目列表。项目的三个离开入口（成员页的离开、侧边栏和项目页头菜单的离开弹窗）都经 `useProjectMembershipChanges`（`web/apps/web/core/components/project/settings/use-project-membership-changes.ts`）的 `leave`：nerve 成功之后才关上确认框或弹窗、跳转，失败时留在原页，确认框或弹窗留着，原因在共用的拒绝提示里（如唯一的管理员）；关上和跳转只在发出离开的会话里进行。它的 vitest 和故事 P5 的页面版本（nerve 的回答扣住时页面不动；被拒绝时成员页的确认框和侧边栏的弹窗都留着）守着（[M3/P10 spec](../specs/P10-web-project-pages.md) 2.7）。
- **保留的工作区名，前后端同一份**：M3/P8a 已完成（[M3/P8a spec](../specs/P8a-web-workspace-data.md) 第 7 节）。

两项都已处理；逐项的结论随 M3 的收尾写进评审，然后 `status` 改为 `closed`。

来源：[M3/P10 spec](../specs/P10-web-project-pages.md) 第 7 节。

````

`docs/v0/M3-workspace-project/handoffs/M2-closeout.md`（修改，1 处）：

````old docs/v0/M3-workspace-project/handoffs/M2-closeout.md
来源：[M3/P9 spec](../specs/P9-web-workspace-pages.md) 第 7 节。

````
````new docs/v0/M3-workspace-project/handoffs/M2-closeout.md
来源：[M3/P9 spec](../specs/P9-web-workspace-pages.md) 第 7 节。

## 处理结果（M3/P10）

- **第 10 节 CSP：表情选择器的数据从本站提供**（关闭）：`emojibase-data` 15.3.2 是 web 应用的直接依赖；构建时一个 Vite 插件（`web/apps/web/emojibase.ts`）把 `en/data.json`、`en/messages.json` 写到 `/assets/emojibase/15.3.2/en/`，`EmojiPicker.Root` 传 `emojibaseUrl`（`EMOJIBASE_URL`，`web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx`），一个 vitest 核对它与构建写出的地址和版本一致；CSP 没有为它放开外部来源。P1、P3 的页面版本打开表情选择器，`watchPage` 没有 CSP 违规、没有发往别处的请求（[M3/P10 spec](../specs/P10-web-project-pages.md) 2.1）。
- **第 11 节 M2 留下的 M3 调用和类型**（关闭）：项目 general 页的时区选择用这个接口，随修改发出，P3 的页面版本核对（spec 2.5）；其余各项在 P8a、P8b、P9 已完成（见上）。
- **第 14 节 下拉框和复制到剪贴板**（代码一侧完成）：`CustomSearchSelect` 改成像 Popover 那样的按钮，`Combobox` 在它的面板里，Tab 能到、键盘打开，列表关上时清空搜索；成员下拉框以它为底，`member-options.tsx` 删除，列表在按钮旁展开（spec 2.2）；关闭条件中的 `member-options` 一项改为成员下拉框（`member-options.tsx` 已删除）。3 处复制、邀请行和工作区 general 页的复制经 `useCopyLink`（`web/apps/web/core/hooks/use-copy-link.ts`），失败时经 `t()` 提示（spec 2.3）。逐个调用方的浏览器核对是 M3/P10 评审的 C7、C8。

仍未处理，状态保持 `open`：第 14 节等 C7、C8 写进 M3/P10 的评审；第 12 节等 M3 的收尾。

来源：[M3/P10 spec](../specs/P10-web-project-pages.md) 第 7 节。

````

`docs/v0/frontend-changes.md`（修改，4 处）：

````old docs/v0/frontend-changes.md
| 工作区、成员、邀请、项目、项目成员、项目归档、状态、标签、显示设置 | M3 | 进行中：数据层已对接（工作区一侧 M3/P8a，项目一侧 M3/P8b）；工作区的页面 M3/P9 已对接，项目的页面在 M3/P10–P11 |
````
````new docs/v0/frontend-changes.md
| 工作区、成员、邀请、项目、项目成员、项目归档、状态、标签、显示设置 | M3 | 进行中：数据层已对接（工作区一侧 M3/P8a，项目一侧 M3/P8b）；工作区的页面 M3/P9、项目的页面 M3/P10 已对接，状态、标签的设置页在 M3/P11 |
````
````old docs/v0/frontend-changes.md
| 所有处理接口错误的地方 | 统一按 RFC 9457 的 problem+json 读取 `code`、`title`、`errors`。M2/P4 已改：`ApiError` 和 `unwrap`（`core/lib/api-error.ts`）按生成的 `Problem` 读取；登录页、注册页和安全页的修改密码按它的 `code`、`errors` 显示错误。M2/P5 改完个人设置的其余部分（[M2 设计](M2-auth/M2-design.md) 7.7）和新手引导的资料步骤：general 页的保存、资料步骤和 api-tokens 页的创建把字段错误显示在字段下方（名字的规则只在 nerve，页面只查必填，Plane 的名字校验从 `@nerve/utils` 删除），其余的错误和 preferences 的主题、时区、语言、每周第一天、PAT 的撤销、停用账户、新手引导换步骤时更新资料（`onboarding/root.tsx`）的失败都在提示中，文案按 `code` 取（`core/lib/error-messages.ts` 的 `PROBLEM_MESSAGES`、`fieldErrorKeys`、`errorMessageKey`，M3/P8a 从 `helpers/authentication.helper.ts` 移来）。M3/P8a 起工作区、成员、邀请、工作区的显示设置，M3/P8b 起项目、项目成员、状态、标签、项目的显示设置经生成的客户端，nerve 的错误应答是 `ApiError`：状态和标签的设置页、工作项的三个标签选择器和项目包装层的加入界面在 M3/P8b 已按 `code` 显示错误（`errorMessageKey`），工作区的页面在 M3/P9（新手引导、邀请页、注册、创建工作区、首页、侧边栏的项目导航、工作区设置和停用账户的弹窗；创建工作区和邀请的表单把 nerve 指名的字段错误显示在对应的字段下方），其余页面在 M3/P10–P11；M4–M8 的领域在各自的 M，它们现在还经 Plane 的 axios 基类按 Plane 的错误格式读取 | 错误格式统一 | 进行中 | |
````
````new docs/v0/frontend-changes.md
| 所有处理接口错误的地方 | 统一按 RFC 9457 的 problem+json 读取 `code`、`title`、`errors`。M2/P4 已改：`ApiError` 和 `unwrap`（`core/lib/api-error.ts`）按生成的 `Problem` 读取；登录页、注册页和安全页的修改密码按它的 `code`、`errors` 显示错误。M2/P5 改完个人设置的其余部分（[M2 设计](M2-auth/M2-design.md) 7.7）和新手引导的资料步骤：general 页的保存、资料步骤和 api-tokens 页的创建把字段错误显示在字段下方（名字的规则只在 nerve，页面只查必填，Plane 的名字校验从 `@nerve/utils` 删除），其余的错误和 preferences 的主题、时区、语言、每周第一天、PAT 的撤销、停用账户、新手引导换步骤时更新资料（`onboarding/root.tsx`）的失败都在提示中，文案按 `code` 取（`core/lib/error-messages.ts` 的 `PROBLEM_MESSAGES`、`fieldErrorKeys`、`errorMessageKey`，M3/P8a 从 `helpers/authentication.helper.ts` 移来）。M3/P8a 起工作区、成员、邀请、工作区的显示设置，M3/P8b 起项目、项目成员、状态、标签、项目的显示设置经生成的客户端，nerve 的错误应答是 `ApiError`：状态和标签的设置页、工作项的三个标签选择器和项目包装层的加入界面在 M3/P8b 已按 `code` 显示错误（`errorMessageKey`），工作区的页面在 M3/P9（新手引导、邀请页、注册、创建工作区、首页、侧边栏的项目导航、工作区设置和停用账户的弹窗；创建工作区和邀请的表单把 nerve 指名的字段错误显示在对应的字段下方），项目的页面在 M3/P10（项目列表的加入、创建项目、项目设置的 general、members、features、automations，归档、恢复、删除，侧边栏的项目顺序和页头的标签栏；创建项目和 general 页把 nerve 指名的字段错误显示在对应的字段下方）；M4–M8 的领域在各自的 M，它们现在还经 Plane 的 axios 基类按 Plane 的错误格式读取 | 错误格式统一 | 进行中 | |
````
````old docs/v0/frontend-changes.md
| 页面跟进修改的结果 | 工作区的页面发出修改之后的跳转、提示和界面变化，只在发出它的会话里进行（`core/lib/in-session.ts` 的 `followInSession`）：另一个标签页换了账户之后，页面已是那个账户的，旧会话的修改不论成败都不再跟进 | M3 设计 7.1 | 已完成 | M3/P9 |
````
````new docs/v0/frontend-changes.md
| 页面跟进修改的结果 | 工作区的页面（M3/P9）和项目的页面（M3/P10）发出修改之后的跳转、提示和界面变化，只在发出它的会话里进行（`core/lib/in-session.ts` 的 `followInSession`）：另一个标签页换了账户之后，页面已是那个账户的，旧会话的修改不论成败都不再跟进 | M3 设计 7.1 | 已完成 | M3/P9、P10 |
````
````old docs/v0/frontend-changes.md
| 停用账户的弹窗 | nerve 拒绝的原因（如唯一的管理员）显示在确认弹窗里，不再是提示，弹窗关闭时清掉；停用只在它结束了标签页的会话时才说已停用（另一个标签页先换了账户时，页面已是那个账户的） | M3 设计 7.1、7.5 | 已完成 | M3/P9 |
````
````new docs/v0/frontend-changes.md
| 停用账户的弹窗 | nerve 拒绝的原因（如唯一的管理员）显示在确认弹窗里，不再是提示，弹窗关闭时清掉；停用只在它结束了标签页的会话时才说已停用（另一个标签页先换了账户时，页面已是那个账户的） | M3 设计 7.1、7.5 | 已完成 | M3/P9 |
| 表情选择器的数据 | 表情选择器（`frimousse`）的数据由 Nerve 自己提供：`emojibase-data` 15.3.2 随前端构建，写到 `/assets/emojibase/15.3.2/en/`，选择器从那里读，不再请求 `cdn.jsdelivr.net`（页面的 CSP 本来就挡住它）；表情名称的搜索只有英文（与 Plane 相同） | M3 设计 7.7；M2 收尾交接第 10 节 | 已完成 | M3/P10 |
| 下拉框 | `CustomSearchSelect`（`@nerve/ui`）改成可以用 Tab 到达的按钮，`Combobox` 在它的弹出面板里：键盘打开，Escape 关上（输入法组字时的 Escape 不关），列表关上时清空搜索；成员下拉框（创建项目的负责人；M4–M6 的指派人、负责人）以它为底，`member-options.tsx` 删除：列表在按钮旁展开，打开时搜索框取得焦点（手机上不取），其余照旧（调用方推迟的，指针经过之前只有按钮；列表离窗口边缘 12 像素）；面包屑的下拉框是一个 Tab 停留点 | M3 设计 7.7；M2 收尾交接第 14 节 | 已完成 | M3/P10 |
| 复制到剪贴板 | 项目卡片、侧边栏、扩展侧边栏和项目页头的菜单复制项目的链接，邀请行复制邀请的链接，工作区设置的 general 页复制工作区的地址，都经 `useCopyLink`（`core/hooks/use-copy-link.ts`）：复制失败（浏览器拒绝时）有提示，不再是未处理的 Promise 拒绝 | M3 设计 7.7；M2 收尾交接第 14 节 | 已完成 | M3/P10 |
| 创建项目 | 请求体只含表单的字段（生成的 `ProjectCreate`）；负责人从工作区的管理员和成员中选（Plane 也列访客）；标识输入时转成大写、只留 nerve 接受的字符；nerve 指名的字段错误显示在字段下方；发出之后表单关不掉，直到 nerve 回答 | M3 设计 3.19、7.6 | 已完成 | M3/P10 |
| 已归档的项目 | 打开已归档项目的地址显示它已归档，按钮去已归档的项目页，不取它的成员、状态、标签和显示设置；已归档的项目页对项目管理员和是工作区管理员的项目成员都给恢复、删除（Plane 只给项目管理员，nerve 两者都允许，项目设置的归档也给两者） | M3 设计 3.4、3.19、7.6 | 已完成 | M3/P10 |
| 项目成员页 | 角色的选项只有 nerve 允许调用者给的（3.5 的相对规则：不是工作区管理员的只改比自己低的角色、只给比自己低的角色；工作区的访客只能是访客），移出只对 nerve 允许的行显示；添加成员只列出工作区里有效、还不是项目成员的人，角色按他的工作区角色（工作区管理员只能是管理员，访客只能是访客）；离开项目先等 nerve 成功再回到项目列表（M1-P4，Plane 先跳后调）；唯一的管理员离开时提示 nerve 的原因，确认框或离开弹窗留着（成员页、侧边栏、项目页头相同，成功之后才关上）；角色的名字随界面的语言（Plane 写死英文） | M3 设计 3.5、7.6 | 已完成 | M3/P10 |
| 项目的成员默认值 | 负责人、默认负责人、"访客可见全部"显示 nerve 的回答，不再先显示选中的值（Plane 被拒绝时仍显示选中的值），拒绝时提示 nerve 的原因 | M3 设计 7.6；总体设计 7.7 | 已完成 | M3/P10 |
| 项目页头的标签栏 | 调用者的标签栏（默认打开的标签页、收进"更多"的标签页）取到之前，标签不给修改它的菜单：修改按 nerve 最近一次的回答算出，取到之前按默认值算会整份替换他的标签栏 | M3 设计 3.18、7.6 | 已完成 | M3/P10 |
````

- [ ] **Step 2: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 68 条规则、3 个例外，没有命中；web 的 oxlint 349 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 106 个全部通过。

- [ ] **Step 3: 提交**

```bash
git add README.md docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md docs/v0/M3-workspace-project/handoffs/M2-closeout.md docs/v0/frontend-changes.md
```
```bash
git commit -m "docs(M3/P10): README's frontend section lists the project pages that work, the frontend changes list P10's, and two handoffs their results

README's frontend section says which project pages work from M3/P10 on and
where the rest are connected; the frontend changes list says the project
pages are connected, which pages show nerve's errors by their code, and
P10's eight changes. The M2 closeout handoff records what P10 did for its
sections 10, 11 and 14, the M1-P4 handoff the leaving's order.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 只改文档，没有变异。
