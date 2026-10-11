# M3/P11 状态和标签的设置页、清理 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** M3 的最后一批页面和 M3 的清理。页面：项目设置的 states、labels 两页接上 P8b 的数据层，行为按 M3 设计第 2 节的故事 P6、P7、W11：项目管理员和是工作区管理员的项目成员增删改、拖动、设默认（状态）、拖进拖出标签组（标签），项目的其他成员只能看，访客看到没有权限；每个修改只在发出它的会话里跟进（`followInSession`），名字已被占用显示在名字下方，被拒绝时表单和弹窗留着，请求在路上时表单忙、弹窗关不掉；控件 Tab 能到、有名字、Tab 到达时看得见。访客看到的个人主页（P9 的页面版本，决策点 3 的 A：页面不变）。M2 的账户页面（新手引导的资料一步、个人设置的 general、preferences、security）的提示和字段错误只在发出修改的会话里（P9 的裁定 P8）。清理：M3 的死成员和死 prop（`domains.mjs --rows M3`）、P8a 和 P8b 留下的只经机械步骤到达的文件的 oxlint 警告、`promise(always-return)` 全仓清零、关键词守卫的最终核对。P6、P7、P9、W11 的页面版本，S2 的挂载清单加上 P11 的页面。

**Architecture:** 只改 web、`e2e/`、根目录 `.oxlintrc.json`（非空断言的范围：Task 1–5、7 各加它新建的模块和测试）、`tools/keywords.json`（`project-address-route` 加 P11 的五个文件：Task 3、5）和四份文档（README、前端改动清单、M1 收尾交接和 M1-P2 交接的处理结果）。页面调用 store 的方法，不调 service；会话的每个取数经 `useSessionSWR`。共用的新部分：`core/lib/form-refusal.ts`（`formRefusal`、`useFormRefusal`：表单的拒绝落到字段下或提示，P10 的 `projectRefusal` 改为以它写成，状态和标签的表单也用它；Task 1）；`@nerve/ui` 的 `AlertModalCore` 在请求路上时关不掉（P9 的 F-1；它在 M4–M8 的 11 个调用方一样；Task 1）；`useStateChanges`（Task 2）、`useLabelChanges`（Task 4）：每个修改经 store、经 `followInSession`，拒绝经 `useFormRefusal` 或 `useRefusalToast`，标签放到第三层的拒绝说标签只有两层。store 的两个 getter 改为以项目的 id 读（`getGroupedProjectStates(projectId)`、`getProjectLabelsTree(projectId)`，Task 3、5；原来读 router store 的项目）。e2e 的共用部分：P8 故事的 `dragOnto` 移进 `e2e/fixtures/project-pages.ts`，`tabCycle` 收按 Tab 的次数（Task 3）。清理（Task 8、9）只删没有读者、没有调用方传入的成员和 prop，只改 oxlint 指出的写法；每一处改动的行为照旧，或写进前端改动清单（spec 附录 A.5、A.6）。契约、Go 代码和生成的文件都不变。

**Tech Stack:** React 19.2.8、React Router 8.3.0、MobX 6.12.0、SWR 2.4.2、openapi-fetch 0.17.0、Headless UI 2.2.10、react-hook-form 7.84、pragmatic-drag-and-drop 1.7.4、TypeScript 5.8.3、vitest 4.1.11、oxlint 1.51.0、oxfmt 0.35.0、knip 6.37.0、turbo 2.10.11；Node 24、pnpm 11.10.0、Playwright 1.63.0；Go 1.27.1（Go 代码不变）。版本都由 `pnpm-lock.yaml` 固定。

**Spec:** `docs/v0/M3-workspace-project/specs/P11-web-states-labels-and-cleanup.md`（上级：`docs/v0/M3-workspace-project/M3-design.md`）

## Global Constraints

- **依赖**：不加任何 npm 包或 Go 模块，不执行 `go get`、`pnpm add`，`pnpm-lock.yaml` 不变。`git diff --stat 7d6a416b -- server/ api/ pnpm-lock.yaml` 在本 plan 的任何时刻都没有输出。
- **每个 Task 提交前**：`make lint-web`（关键词守卫、`tsc`、oxlint 在上限、格式、`en` 与 `zh-CN` 的键一致）、`make knip`、`make test-web`、`make e2e` 都通过。本 plan 不改 Go 代码和接口描述，不生成任何文件，不执行 `make gen`（`make gen-check` 照旧没有差异），没有要钉住 SHA-256 的生成文件。
- **e2e 的数目**：在 worktree 中 `make e2e` 必须全部通过：开始时 106 个；Task 3 起 108、Task 5 起 109、Task 6 起 110。只在不是 git 仓库的副本里，S3 因为构建读不到提交而失败（P4b spec F4），其余照上面的数目减一全部通过。
- **容器**：`make e2e` 用自己的 testcontainers；机器忙时偶尔起不来，等 Docker 空闲之后重跑一次再当作失败。容器测试一次只跑一套。开发库 `nerve-dev-db-1` 不要启动、停止或重建，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`、`nervewiki-*`）和留下的 testcontainers。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合，只读的命令也不行。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。不把副本的 `node_modules` 链接到 worktree 的。
- **规则**（总体设计 7.7，P8a 的裁定 F-1–F-10，P8b 的 PF-M2，P9 的 F-1–F-4，P10 的 A-M2、F-2、F-15）：会话的每个取数经 `useSessionSWR`（不传配置）；页面只取它的角色能读的；修改经 store 的方法，一个接一个，store 写入 nerve 的回答，请求体取决于 store 所持的值时轮到它发出时从 nerve 最近一次回答算出（状态、标签的移动，P7a、P7b 的 store 已是这样）；`SessionChangedError` 不是认证失败；`core/lib/reconciled.ts` 只扩展，不复制（P11 不改它）。页面读地址只有一种写法：路由的参数（route module 的 `params`），`allowPermissions` 写明工作区和项目（`project-address-route`）。页面发出的每个修改（状态、标签的增删改、移动、设默认，账户页面的修改）之后的提示、字段错误和界面变化经 `followInSession` 只在发出它的会话里进行；每一处由它自己的 vitest 守着（换账户之后兑现的 `lateSettlings`、`lateSettlingsAnswering`）。请求在路上时表单忙（react-hook-form 的 `isSubmitting`，取消也按不了），弹窗关不掉（`AlertModalCore` 的取消禁用、`handleClose` 为空），由故事扣住请求核对（`sentHeld`、`closedByEscape`、`enabledWithin`）。被拒绝时表单和弹窗留着，原因在字段下或共用的拒绝提示里；只在成功的跟进中关上（A-M2）。请求体由表单编辑的字段构成，类型是生成的 `StateCreate`、`StateUpdate`、`LabelCreate`、`LabelUpdate`、`UserUpdate`、`ProfileUpdate`；每个请求体有一个测试钉住它（vitest 或端到端的 `sentTo`、`sentHeld`）。每个控件 Tab 能到、按次序（`tabCycle`），不加正的 `tabIndex`（P10 的 F-15、F-18；`AlertModalCore` 的 `tabIndex={1}` 属于 M3 收尾的 `ModalCore` 重写，P11 不动它）。类型只来自生成的客户端：不写重述契约的类型，没有新的 `as`、`any`、`!`（spec 附录 A.7）。删除的代码删干净（组件、类型、两种语言的文案、常量），不加 knip 的忽略、开关或桩。本 plan 写或重写的文件都在约 400 行以内（spec 附录 A.8；原来就超过的 `power-k` 的 `work-item/commands.ts` 和 `tools/keywords.json` 各多三行，spec 第 3 节第 14 条）；只为使用方改到的 Plane 文件不变长。
- **oxlint**（M3 设计 7.9）：有手改的文件在它的 Task 提交时没有 oxlint 警告；例外两处：`@nerve/ui` 的 `alert-modal.tsx` 留着 `tabIndex={1}` 的一条（M3 收尾的 `ModalCore` 重写，spec 第 3 节第 3 条），Task 9 只按 `promise(always-return)` 改到的 26 个文件的其余 14 条留给它们的 M（spec 第 3 节第 8 条）。上限：web 348 → 346（Task 2：状态的组标题是按钮）→ 337（Task 8：P8a、P8b 留下的文件和死 prop 的删除带走的）→ 303（Task 9：`promise(always-return)` 34 条）；editor 65 → 64（Task 9：1 条）；ui 15、propel 11、utils 7、hooks 3、constants 1 不变（spec 第 3 节第 9 条）。没有新的 oxlint 抑制。
- **注释**：TS 代码、测试、JSON 的说明用英文；中文文档照本 plan 原样。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的）。

  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从 `7d6a416b` 起按顺序核对过全部块：每个 `old` 恰好出现一次（在它之前的块应用之后的文件中），每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同（spec 附录 A.10）。可以用 `node <planapply.mjs> <本 plan> apply <仓库根> <n>` 写入第 n 个 Task 的块，也可以手工照抄。Task 9 的 35 处 `return;` 是一个一次性的脚本在原型中写出的（spec 附录 A.6），这里照样写成块。
- **过渡版本**：一些文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改：根目录 `.oxlintrc.json`（Task 1–5、7，每个 Task 加它的文件）、`web/apps/web/package.json`（Task 2、8、9 的上限）、`tools/keywords.json`（Task 3、5）、`states-labels.test.tsx`（Task 3、5）、状态的 `group-item.tsx`、`root.tsx`、`state-item-title.tsx`、`options/delete.tsx`（Task 2、3）、标签的 `project-setting-label-group.tsx`、`project-setting-label-list.tsx`（Task 4、5），以及 S2（Task 3、5、6）、W11（Task 3、5）。每个过渡版本都在逐 Task 复现中运行过。
- **变异**：每个 Task 末尾的"变异"表列出：把代码改坏的方式、必须因此失败的检查和它所在的层（静态：`make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest：`make test-web`；端到端：`make e2e` 的故事）。它们在最终的原型上逐个跑过（`$M3TMP/p11tools/mutants_p11.py`，`mut.py` 在它写的每个检查上各跑一次），也在各自 Task 的树上跑过（spec 附录 A.2）；表中标"（Task N 起）"的检查只在第 N 个 Task 加入的测试中才让它失败（在变异自己的 Task 的树上它还活着，在第 N 个 Task 的树上核对过）。**会话、权限或取数的性质只由评审才能发现的，算缺口**（brief）：表中每一条这类性质都有一个会失败的检查。
- **评审敏感**（M3 设计 12 节 P11 的评审重点）：每个修改的跟进只在发出它的会话里（`useStateChanges`、`useLabelChanges` 和账户页面的 vitest 的 `lateSettlings`）；页面只提供 nerve 允许的（改状态、标签的控件只给项目管理员和是工作区管理员的项目成员：`states-labels.test.tsx` 在 store 自己的权限上核对，P6、P7 的页面版本以成员和工作区管理员各看一次）；访客打开个人主页不取工作项（P9 的页面版本、S2）；请求在路上时弹窗关不掉、表单忙（故事扣住请求）；清理的每一处行为照旧（spec 附录 A.5、A.6）。改动这些之前，先照"变异"表确认它在所说的性质去掉之后失败。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录。一个文件由多个 Task 修改时，"Task"一列都列出。

| 文件 | 职责 | Task |
|---|---|---|
| `.oxlintrc.json` | P11 的新模块和测试加进非空断言的范围，各在加入它们的 Task（P9 spec 2.14 的做法） | 1、2、3、4、5、7 |
| `web/apps/web/core/lib/form-refusal.ts` | `formRefusal`、`useFormRefusal`：表单的拒绝，已被占用的码在它的字段下，字段错误在表单的字段下，其余提示（新文件） | 1 |
| `web/apps/web/core/components/project/project-refusal.ts` | `projectRefusal`、`useProjectRefusal` 以 `formRefusal`、`useFormRefusal` 写成，显示的照旧 | 1 |
| `web/packages/ui/src/modals/alert-modal.tsx` | `AlertModalCore`：`isSubmitting` 时取消禁用，Escape 和点外面不关（P9 的 F-1） | 1 |
| `web/apps/web/core/components/project-states/use-state-changes.ts`、`web/apps/web/core/components/project-states/use-state-changes.test.ts` | `useStateChanges`：状态的创建、修改、移动、删除、设默认经 store、经 `followInSession`，拒绝在字段下或提示；它的 vitest（新文件） | 2 |
| `web/apps/web/core/components/project-states/create-update/form.tsx`、`web/apps/web/core/components/project-states/create-update/create.tsx`、`web/apps/web/core/components/project-states/create-update/update.tsx`、`web/apps/web/core/components/project-states/create-update/create-update.test.tsx` | 状态的表单用 react-hook-form：忙到页面跟进完，取消也按不了，nerve 指名的字段错误在字段下；创建、修改经 `useStateChanges`；它们的 vitest | 2 |
| `web/apps/web/core/components/project-states/options/delete.tsx` | 删除经 `useStateChanges`，确认框在跟进做完之前忙、关不掉，只在成功的跟进中关上（Task 2）；删除按钮有名字（Task 3） | 2、3 |
| `web/apps/web/core/components/project-states/options/delete.test.tsx` | 哪些状态提供删除，确认之后发出什么，拒绝的提示（新文件） | 2 |
| `web/apps/web/core/components/project-states/options/mark-as-default.tsx` | 设默认经 `useStateChanges`，忙到跟进做完 | 2 |
| `web/apps/web/core/components/project-states/root.tsx` | 不再拼回调（Task 2）；项目的状态按路由的项目读（`getGroupedProjectStates`，Task 3） | 2、3 |
| `web/apps/web/core/components/project-states/group-list.tsx`、`web/apps/web/core/components/project-states/state-list.tsx`、`web/apps/web/core/components/project-states/state-item.tsx` | 不再传回调和没人传的 `className`；状态的一行以 id 为 key；移动经 `useStateChanges` | 2 |
| `web/apps/web/core/components/project-states/group-item.tsx` | 组标题是按钮（`aria-expanded`），不再传回调（Task 2）；加状态的按钮有名字（Task 3） | 2、3 |
| `web/apps/web/core/components/project-states/state-item-title.tsx` | 一个类型（`disabled` 为布尔），没人传的 `shouldShowDescription` 删除（Task 2）；一行的控件 Tab 能到、Tab 到达时看得见，编辑按钮有名字（Task 3） | 2、3 |
| `web/packages/types/src/state.ts`、`web/packages/types/src/index.ts` | `TStateOperationsCallbacks`（只有状态的组件读它）删除 | 2 |
| `web/apps/web/core/lib/fake-controls.ts` | `AlertModalCore` 的替身留下它收到的 props | 2 |
| `web/apps/web/package.json` | web 的 oxlint 上限 348 → 346（Task 2）→ 337（Task 8）→ 303（Task 9） | 2、8、9 |
| `web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx` | 状态设置页从路由读项目，`allowPermissions` 写明工作区和项目 | 3 |
| `web/apps/web/core/store/state.store.ts`、`web/apps/web/core/store/state.store.test.ts` | `getGroupedProjectStates(projectId)` 代替读 router store 的 `groupedProjectStates` | 3 |
| `web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx` | 状态（Task 3）、标签（Task 5）的设置页给谁看什么：在 store 自己的权限上核对（新文件） | 3、5 |
| `tools/keywords.json` | `project-address-route` 加状态设置页和它的根组件（Task 3）、标签设置页、标签列表和它的拖放组件（Task 5） | 3、5 |
| `e2e/fixtures/project-pages.ts` | `dragOnto`（从 P8 的故事移来）；`tabCycle` 收按 Tab 的次数 | 3 |
| `e2e/stories/project/p6-states-page.spec.ts` | P6 的页面版本（新文件） | 3 |
| `e2e/stories/project/p6-states.spec.ts`、`e2e/stories/project/p8-project-preferences.spec.ts` | P6 的 API 版本的说明指向页面版本；P8 的拖动用 fixture 的 `dragOnto` | 3 |
| `e2e/stories/workspace/w11-guest-bounds.spec.ts` | W11 的页面版本：访客打开状态（Task 3）、标签（Task 5）的设置页看到没有权限 | 3、5 |
| `e2e/stories/smoke/s2-web-app.spec.ts` | S2 的挂载清单加上状态（Task 3）、标签（Task 5）的设置页和访客打开的两个个人主页（Task 6） | 3、5、6 |
| `web/apps/web/core/components/labels/use-label-changes.ts`、`web/apps/web/core/components/labels/use-label-changes.test.ts` | `useLabelChanges`：标签的创建、修改、移动、移出组、删除经 store、经 `followInSession`，拒绝在名字下或提示，第三层说标签只有两层；它的 vitest（新文件） | 4 |
| `web/apps/web/core/components/labels/create-update-label-inline.tsx`、`web/apps/web/core/components/labels/create-update-label-inline.test.tsx` | 标签的表单忙到页面跟进完，取消也按不了，名字已被占用在名字下；经 `useLabelChanges`；它的 vitest | 4 |
| `web/apps/web/core/components/labels/delete-label-modal.tsx` | 删除经 `useLabelChanges`，确认框只在成功的跟进中关上 | 4 |
| `web/apps/web/core/components/labels/project-setting-label-item.tsx` | 不再收回调，移出组经 `useLabelChanges`，表单收项目 | 4 |
| `web/apps/web/core/components/labels/project-setting-label-group.tsx` | 不再收、不再传回调，表单收项目（Task 4）；组的那一行的菜单和手柄只给能改标签的人（Task 5） | 4、5 |
| `web/apps/web/core/components/labels/project-setting-label-list.tsx` | 移动经 `useLabelChanges`（Task 4）；工作区和项目由页面给，标签按项目的 id 读，没有标签时的按钮只给能改标签的人（Task 5） | 4、5 |
| `web/packages/i18n/src/locales/en/project-settings.json`、`web/packages/i18n/src/locales/zh-CN/project-settings.json` | `project_settings.labels.parent_not_allowed`：标签只有两层 | 4 |
| `web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx` | 标签设置页从路由读项目，`allowPermissions` 写明工作区和项目 | 5 |
| `web/apps/web/core/store/label.store.ts`、`web/apps/web/core/store/label.store.test.ts`、`web/apps/web/core/store/label.store.changes.test.ts` | `getProjectLabelsTree(projectId)` 代替读 router store 的 `projectLabelsTree` | 5 |
| `web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx` | 拖动只由此刻渲染的手柄开始（回调的 ref），手柄只给能改标签的人 | 5 |
| `web/apps/web/core/components/labels/label-block/label-item-block.tsx` | 没有调用方传的 `draggable` 删除；菜单和删除按钮有名字，Tab 到达时看得见 | 5 |
| `web/apps/web/core/components/labels/project-setting-label-group.test.tsx` | 组的那一行的控件给谁（新文件） | 5 |
| `e2e/stories/project/p7-labels-page.spec.ts` | P7 的页面版本（新文件） | 5 |
| `e2e/stories/project/p7-labels.spec.ts` | P7 的 API 版本的说明指向页面版本 | 5 |
| `e2e/stories/project/p9-guest-profile.spec.ts` | P9 的页面版本：访客打开别人的和自己的个人主页（新文件） | 6 |
| `web/apps/web/core/components/onboarding/steps/profile/root.tsx`、`web/apps/web/core/components/onboarding/steps/profile/root.test.tsx` | 新手引导的资料一步：修改的跟进经 `followInSession`；它的 vitest | 7 |
| `web/apps/web/core/components/settings/profile/content/pages/general/form.tsx`、`web/apps/web/core/components/settings/profile/content/pages/security.tsx`、`web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx`、`web/apps/web/core/components/profile/start-of-week-preference.tsx` | 个人设置的 general、security、时区、语言、每周第一天：跟进经 `followInSession` | 7 |
| `web/apps/web/core/components/settings/profile/content/pages/general/form.test.tsx`、`web/apps/web/core/components/settings/profile/content/pages/security.test.tsx`、`web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.test.tsx` | 它们的 vitest（新文件） | 7 |
| `web/apps/web/core/components/dropdowns/project/dropdown.tsx`、`web/apps/web/core/components/dropdowns/project/base.tsx` | 没有调用方传的六个 prop 删除；`getProjectById` 的类型是生成的 `Project`（P8b 的裁定 P1）；oxlint 清零 | 8 |
| `web/apps/web/core/components/dropdowns/member/avatar.tsx`、`web/apps/web/core/components/dropdowns/state/state-option.tsx`、`web/apps/web/core/components/power-k/menus/members.tsx`、`web/apps/web/core/components/project/create/common-attributes.tsx`、`web/apps/web/core/components/settings/workspace/sidebar/root.tsx`、`web/apps/web/core/components/workspace/invite-modal/actions.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/header.tsx`、`web/packages/propel/src/icons/state/dashed-circle.tsx` | 没有调用方传入的 prop 删除，组件按原来的默认值显示 | 8 |
| `web/apps/web/core/store/member/workspace/workspace-member.store.ts`、`web/packages/constants/src/members.ts` | 没有代码读的 `memberMap`、`iconName` 删除 | 8 |
| `web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx`、`web/apps/web/core/components/core/sidebar/progress-stats/state_group.tsx` | P8a、P8b 留下的 oxlint 警告清零（`no-shadow`、`no-array-index-key`） | 8 |
| `web/apps/web/app/entry.client.tsx`、`web/apps/web/core/components/cycles/delete-modal.tsx`、`web/apps/web/core/components/cycles/modal.tsx`、`web/apps/web/core/components/cycles/quick-actions.tsx`、`web/apps/web/core/components/cycles/transfer-issues-modal.tsx`、`web/apps/web/core/components/inbox/modals/delete-issue-modal.tsx`、`web/apps/web/core/components/issues/delete-issue-modal.tsx`、`web/apps/web/core/components/issues/issue-detail/issue-activity/helper.tsx`、`web/apps/web/core/components/issues/peek-overview/header.tsx`、`web/apps/web/core/components/issues/workspace-draft/delete-modal.tsx`、`web/apps/web/core/components/modules/delete-module-modal.tsx`、`web/apps/web/core/components/modules/module-card-item.tsx`、`web/apps/web/core/components/modules/module-list-item-action.tsx`、`web/apps/web/core/components/modules/quick-actions.tsx`、`web/apps/web/core/components/power-k/config/miscellaneous-commands.ts`、`web/apps/web/core/components/power-k/ui/pages/context-based/cycle/commands.ts`、`web/apps/web/core/components/power-k/ui/pages/context-based/module/commands.tsx`、`web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts`、`web/apps/web/core/components/web-hooks/create-webhook-modal.tsx`、`web/apps/web/core/components/web-hooks/form/secret-key.tsx`、`web/apps/web/core/components/work-item-filters/filters-hoc/project-level.tsx`、`web/apps/web/core/components/workspace-notifications/sidebar/notification-card/options/snooze/modal.tsx`、`web/apps/web/core/components/workspace/views/default-view-quick-action.tsx`、`web/apps/web/core/store/global-view.store.ts`、`web/apps/web/core/store/workspace/webhook.store.ts`、`web/packages/editor/src/helpers/floating-ui.ts` | `then()` 的回调末尾加 `return;`（35 处，`promise(always-return)` 清零） | 9 |
| `web/packages/editor/package.json` | editor 的 oxlint 上限 65 → 64 | 9 |
| `README.md`、`docs/v0/frontend-changes.md`、`docs/v0/M3-workspace-project/handoffs/M1-closeout.md`、`docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md` | README 的"前端"一节、前端改动清单、两份交接的处理结果 | 10 |

---

### Task 1: 表单的拒绝（与项目的表单共用）和请求在路上时关不掉的确认框

**Files:**
- Create: `web/apps/web/core/lib/form-refusal.ts`
- Modify: `.oxlintrc.json`、`web/apps/web/core/components/project/project-refusal.ts`、`web/packages/ui/src/modals/alert-modal.tsx`

**Interfaces:**
- Produces（spec 2.1；总体设计 7.7；P9 的 F-1；P10 spec 2 的 `project-refusal.ts`）：
  - `core/lib/form-refusal.ts`（新文件）：`FormRefusal<F>`（`{ kind: "fields"; fields } | { kind: "toast" }`）；`formRefusal(error, fields, taken)`：表示"值已被占用"的码在 `taken` 给它的字段下（不论问题指名哪些字段），只指名表单的字段的字段错误在那些字段下（`needsErrorBanner`），其余提示；`useFormRefusal(fields, taken)`：照它显示，字段下的经表单的 `underField`（它的 `setError`），按 `fields` 的次序，提示经 `useRefusalToast`。
  - `project-refusal.ts`：`projectRefusal`、`useProjectRefusal` 以它们写成（字段 `name`、`identifier`，码 `project.name_taken`、`project.identifier_taken`）；`ProjectRefusal` 是 `FormRefusal<ProjectFormField>`。显示的照旧（P10 的 `project-refusal.test.ts` 不改，照旧通过）。状态（Task 2）和标签（Task 4）的表单用同一份。
  - `@nerve/ui` 的 `AlertModalCore`：`isSubmitting` 时取消按钮禁用，`ModalCore` 收到的 `handleClose` 为空（Escape、点外面都不关）。它的其余 11 个调用方同样（M4–M8 的 10 个，个人设置的 API 令牌的撤销；spec 第 3 节第 2 条，W17）；它的确认按钮的 `tabIndex={1}` 不动（M3 收尾的 `ModalCore` 重写）：这个文件留着它的一条 oxlint 警告（`jsx-a11y/tabindex-no-positive`，spec 第 3 节第 3 条），`@nerve/ui` 的上限 15 不变。
  - 静态检查：`.oxlintrc.json` 的非空断言的范围加上 `form-refusal.ts`。

**Tests:** 没有新的测试文件：`formRefusal` 由 `project-refusal.test.ts`（P10，不改）和 Task 2、4 的 hook 的 vitest 守着；`AlertModalCore` 在请求路上时关不掉由 Task 3 的 P6 页面版本扣住删除请求核对（Task 5 的 P7 页面版本也是），见本 Task 的变异表。

- [ ] **Step 1: 表单的拒绝**

`web/apps/web/core/components/project/project-refusal.ts`（整个文件，42 行）：

````whole web/apps/web/core/components/project/project-refusal.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// lib
import { formRefusal, useFormRefusal, type FormRefusal } from "@/lib/form-refusal";

/** The fields of a project's forms (its creation, its general settings) under which nerve's refusals show. */
export type ProjectFormField = "name" | "identifier";

/** What a project's form shows of nerve's refusal: under the fields it names (their messages' i18n keys), or a toast. */
export type ProjectRefusal = FormRefusal<ProjectFormField>;

/** The fields, in the order their messages show. */
const FIELDS: readonly ProjectFormField[] = ["name", "identifier"];

/** nerve's codes for a name or an identifier another project of the workspace has, each under its field. */
const TAKEN: Readonly<Record<string, ProjectFormField>> = {
  "project.name_taken": "name",
  "project.identifier_taken": "identifier",
};

/**
 * nerve's refusal of a project's creation or change (M3 design 2 P1, P3), as its form shows it (formRefusal): a name
 * or an identifier taken, under its field; field errors under the form's fields when they name only those; else
 * nerve's reason in a toast.
 */
export function projectRefusal(error: unknown): ProjectRefusal {
  return formRefusal(error, FIELDS, TAKEN);
}

/**
 * Shows nerve's refusal of a project's form as projectRefusal decides (useFormRefusal): each message under its field,
 * by the form's underField (its setError), else nerve's reason in a toast.
 */
export function useProjectRefusal(): (
  error: unknown,
  underField: (field: ProjectFormField, message: string) => void
) => void {
  return useFormRefusal(FIELDS, TAKEN);
}
````

`web/apps/web/core/lib/form-refusal.ts`（新文件，59 行）：

````file web/apps/web/core/lib/form-refusal.ts
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

/** What a form shows of nerve's refusal: under the fields it names (their messages' i18n keys), or a toast. */
export type FormRefusal<F extends string> = { kind: "fields"; fields: Partial<Record<F, string>> } | { kind: "toast" };

/**
 * nerve's refusal of what a form sent (a project's, a state's, a label's), as the form shows it, among its fields: a
 * code of a value another resource already has, under the field taken names for it, whatever fields the problem names;
 * field errors under the form's fields when they name only those (needsErrorBanner: a field the form shows none under
 * is not one of them); else nerve's reason in a toast.
 */
export function formRefusal<F extends string>(
  error: unknown,
  fields: readonly F[],
  taken: Readonly<Record<string, F>>
): FormRefusal<F> {
  const code = error instanceof ApiError ? (error.problem?.code ?? "") : "";
  const field = taken[code];
  if (field) {
    // by the field's name, as fieldErrorKeys gives a problem's fields
    const named: Partial<Record<string, string>> = { [field]: PROBLEM_MESSAGES[code] };
    return { kind: "fields", fields: named };
  }
  if (needsErrorBanner(error, fields)) return { kind: "toast" };
  return { kind: "fields", fields: fieldErrorKeys(error) };
}

/**
 * Shows nerve's refusal of what a form sent as formRefusal decides: each message under its field, by the form's
 * underField (its setError), in the order of fields; else nerve's reason in a toast (useRefusalToast).
 */
export function useFormRefusal<F extends string>(
  fields: readonly F[],
  taken: Readonly<Record<string, F>>
): (error: unknown, underField: (field: F, message: string) => void) => void {
  const { t } = useTranslation();
  const toastRefusal = useRefusalToast();
  return (error, underField) => {
    const refusal = formRefusal(error, fields, taken);
    if (refusal.kind === "toast") {
      toastRefusal(error);
      return;
    }
    for (const field of fields) {
      const message = refusal.fields[field];
      if (message) underField(field, t(message));
    }
  };
}
````

- [ ] **Step 2: 确认框**

`web/packages/ui/src/modals/alert-modal.tsx`（修改，3 处）：

````old web/packages/ui/src/modals/alert-modal.tsx
  content: React.ReactNode | string;
````
````new web/packages/ui/src/modals/alert-modal.tsx
  content: React.ReactNode | string;
  /** Dismisses the dialog: by Cancel, Escape or the backdrop, none of which does while isSubmitting. */
````
````old web/packages/ui/src/modals/alert-modal.tsx
    <ModalCore isOpen={isOpen} handleClose={handleClose} position={position} width={width}>
````
````new web/packages/ui/src/modals/alert-modal.tsx
    <ModalCore isOpen={isOpen} handleClose={isSubmitting ? undefined : handleClose} position={position} width={width}>
````
````old web/packages/ui/src/modals/alert-modal.tsx
        <Button variant="secondary" onClick={handleClose}>
````
````new web/packages/ui/src/modals/alert-modal.tsx
        <Button variant="secondary" onClick={handleClose} disabled={isSubmitting}>
````

- [ ] **Step 3: 静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/app/(all)/\\[workspaceSlug\\]/(settings)/settings/projects/\\[projectId\\]/admin-pages.test.tsx"
````
````new .oxlintrc.json
        "web/apps/web/app/(all)/\\[workspaceSlug\\]/(settings)/settings/projects/\\[projectId\\]/admin-pages.test.tsx",
        "web/apps/web/core/lib/form-refusal.ts"
````

- [ ] **Step 4: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 69 条规则、3 个例外，没有命中；web 的 oxlint 348 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 106 个全部通过。

- [ ] **Step 5: 提交**

```bash
git add .oxlintrc.json web/apps/web/core/components/project/project-refusal.ts web/apps/web/core/lib/form-refusal.ts web/packages/ui/src/modals/alert-modal.tsx
```
```bash
git commit -m "feat(M3/P11): a form's refusal is one module, the project's forms its first user, and a confirmation's dialog stays while its request is out

formRefusal and useFormRefusal put a code of a value taken under its
field, field errors under the form's fields and anything else in a toast;
projectRefusal and useProjectRefusal are now written with them, showing
what they showed. AlertModalCore disables Cancel and ignores Escape and
the backdrop while isSubmitting (P9's F-1), for every dialog built on it.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p11.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T1.1` | `formRefusal` 不认已被占用的码：名字已被占用照问题指名的字段或提示显示 | `project-refusal.test.ts` | vitest |
| `T1.2` | 确认框在请求路上时按 Escape 关上（F-1） | 故事 P6（Task 3 起） | 端到端 |
| `T1.3` | 确认框在请求路上时取消仍可用 | 故事 P6（Task 3 起） | 端到端 |
| `T1.4` | 在 `form-refusal.ts` 里写一个非空断言 | oxlint（`check:lint`） | 静态 |

---

### Task 2: 状态的修改在会话里跟进；表单和删除的确认框忙到跟进做完

**Files:**
- Create: `web/apps/web/core/components/project-states/options/delete.test.tsx`、`web/apps/web/core/components/project-states/use-state-changes.test.ts`、`web/apps/web/core/components/project-states/use-state-changes.ts`
- Modify: `.oxlintrc.json`、`web/apps/web/core/components/project-states/create-update/create-update.test.tsx`、`web/apps/web/core/components/project-states/create-update/create.tsx`、`web/apps/web/core/components/project-states/create-update/form.tsx`、`web/apps/web/core/components/project-states/create-update/update.tsx`、`web/apps/web/core/components/project-states/group-item.tsx`、`web/apps/web/core/components/project-states/group-list.tsx`、`web/apps/web/core/components/project-states/options/delete.tsx`、`web/apps/web/core/components/project-states/options/mark-as-default.tsx`、`web/apps/web/core/components/project-states/root.tsx`、`web/apps/web/core/components/project-states/state-item-title.tsx`、`web/apps/web/core/components/project-states/state-item.tsx`、`web/apps/web/core/components/project-states/state-list.tsx`、`web/apps/web/core/lib/fake-controls.ts`、`web/apps/web/package.json`、`web/packages/types/src/index.ts`
- Delete: `web/packages/types/src/state.ts`

**Interfaces:**
- Produces（spec 2.2；M3 设计 2 的 P6，3.17，7.1、7.6；总体设计 7.7；P7a spec 第 5 节的 P11 一行）：
  - `useStateChanges()`（新文件）：`create(projectId, data, underField, done)`、`update(stateId, data, underField, done)`、`move(stateId, group, droppedOnId, after)`、`remove(stateId, done)`、`markDefault(stateId)`，各经 store 的方法和 `followInSession`；创建、修改的拒绝经 `useFormRefusal`（`project.state_name_taken` 在名字下，字段错误在 `name`、`description` 下，其余提示），移动、删除、设默认的拒绝经 `useRefusalToast`；每个都在页面跟进之后才兑现。
  - 状态的表单（`create-update/form.tsx`，整个文件）用 react-hook-form：`onSubmit(formData, underField)` 返回的 promise 兑现之前表单忙（`isSubmitting`：按钮显示"Creating"/"Updating"、按不了，取消也按不了），nerve 指名的字段错误经 `setError` 在字段下；`create.tsx`、`update.tsx` 经 `useStateChanges`，成功的跟进里提示、关上表单，被拒绝时表单留着。请求体是 `StateCreate`（名字、说明、颜色和表单的组）、`StateUpdate`（表单编辑的名字、颜色、说明）。
  - `options/delete.tsx`：删除经 `remove`，确认框在跟进做完之前忙（Task 1 的 `AlertModalCore` 关不掉），只在成功的跟进中关上；默认的状态和组里唯一的状态不提供删除（按钮禁用、确认也不发）。`options/mark-as-default.tsx`：经 `markDefault`，忙到跟进做完。
  - 组件不再收回调：`TStateOperationsCallbacks`（`@nerve/types`，只有这几个组件读它）删除；`root.tsx` 不再拼回调，`GroupList`、`GroupItem` 收 `projectId`（创建用）；没有调用方传的 `groupListClassName`、`groupItemClassName`、`stateItemClassName` 和 `shouldShowDescription` 删除（P8b A.9 的行）；`state-item-title.tsx` 的两种 props 合成一种（`disabled: boolean`）；一组的状态以 `state.id` 为 key（原来是名字）；移动经 `move`。
  - 组标题（`group-item.tsx`）是按钮（`type="button"`、`aria-expanded`），里面的 `div` 换成 `span`：Tab 能到，Enter、空格展开、收起（原来是可点击的 `div`；它的两条 oxlint 警告随之没有了，web 的上限 348 → 346）。按钮里没有别的按钮（加状态的按钮在它旁边）；样式不变（spec 附录 A.5 的 C-a）。
  - `core/lib/fake-controls.ts`：`AlertModalCore` 的替身留下它收到的 props（`shown.alerts`）。
  - 静态检查：`.oxlintrc.json` 的非空断言的范围加上 `use-state-changes.ts` 和它的测试、`options/delete.test.tsx`。

**Tests:**
- vitest：`use-state-changes.test.ts`（新文件）：`sends $change to the store as the page asks`；`follows $change nerve makes with done, and says nothing`；`shows $refused under its field, of $change, and does not follow it`（创建和修改各三种：名字已被占用、名字太长、nerve 不接受的说明）；`shows nerve's reason for $refused, of $change, in a toast`（表单下没有它的字段的颜色、不再是管理员的人的修改）；`shows nerve's reason when it refuses $refused, in a toast, and does not follow it`（删除默认的、删除组里唯一的、移出组里唯一的、给已删除的状态设默认）；`does nothing on the page when it $settles after another tab moved this one`（五个修改 × `lateSettlingsAnswering` 的两种）。`create-update.test.tsx`：`creates a state in the page's project and the form's group, with what the form holds`；`says the $change made and closes the form, the form busy until it has`；`shows nerve's refusal of a $change under the name, and stays open`（原来的"在提示里说名字已被占用"一条删除）。`options/delete.test.tsx`（新文件）：`offers no deletion of $state, and sends none`；`deletes a state its group has another of, once its dialog is confirmed, the dialog closed until then`；`says nerve's reason when it refuses the deletion`。
- 端到端：表单和确认框扣住请求的核对在 Task 3 的 P6 页面版本（状态设置页从那里起按路由读项目）；本 Task 的 `make e2e` 照旧通过（P6、P8 的 API 版本，S2）。

- [ ] **Step 1: 状态的修改**

`web/apps/web/core/components/project-states/use-state-changes.test.ts`（新文件，175 行）：

````file web/apps/web/core/components/project-states/use-state-changes.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { State } from "@nerve/api-client";
import { heldChange, lateSettlingsAnswering, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal, underFields } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf, stateOf } from "@/store/project/fake-projects";
import { useStateChanges } from "./use-state-changes";

// How the state settings change a project's states (M3 design 2 P6, 7.1, 7.6): the hook runs as a plain function,
// with stand-ins for the state store's changes, which nerve answers when the test says, and for the form's setError.
// Its session is fake-tab.ts's.

const store = vi.hoisted(() => ({
  createState: vi.fn(),
  updateState: vi.fn(),
  moveState: vi.fn(),
  deleteState: vi.fn(),
  markStateAsDefault: vi.fn(),
}));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => store }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const web = projectOf("WEB", "w-acme");
const review = stateOf(web, "Review", "started", 70000);
const setError = vi.fn();
const done = vi.fn();
const underField = (field: string, message: string) => setError(field, { message });

/** Each change of the page, sent as the page sends it, with the store's change it makes and what nerve answers. */
const changes = {
  creation: {
    change: "a creation",
    send: () =>
      useStateChanges().create(web.id, { name: "Review", color: "#8B5CF6", group: "started" }, underField, done),
    sent: () => store.createState,
    call: [web.id, { name: "Review", color: "#8B5CF6", group: "started" }],
    answer: review,
  },
  change: {
    change: "a change",
    send: () => useStateChanges().update(review.id, { name: "Checked" }, underField, done),
    sent: () => store.updateState,
    call: [review.id, { name: "Checked" }],
    answer: review,
  },
  move: {
    change: "a move",
    send: () => useStateChanges().move(review.id, "started", "s-WEB-Todo", true),
    sent: () => store.moveState,
    call: [review.id, "started", "s-WEB-Todo", true],
    answer: review,
  },
  deletion: {
    change: "a deletion",
    send: () => useStateChanges().remove(review.id, done),
    sent: () => store.deleteState,
    call: [review.id],
    answer: undefined,
  },
  newDefault: {
    change: "a new default",
    send: () => useStateChanges().markDefault(review.id),
    sent: () => store.markStateAsDefault,
    call: [review.id],
    answer: undefined,
  },
};
const { creation, change, move, deletion, newDefault } = changes;

beforeEach(() => {
  signedIn();
  for (const sent of Object.values(store)) {
    sent.mockReset();
    sent.mockResolvedValue(review);
  }
  setError.mockReset();
  done.mockReset();
  toasts.length = 0;
});

describe("useStateChanges", () => {
  it.each(Object.values(changes))("sends $change to the store as the page asks", async ({ send, sent, call }) => {
    await send();
    expect(sent().mock.calls).toEqual([call]);
  });

  it.each([creation, change, deletion])("follows $change nerve makes with done, and says nothing", async ({ send }) => {
    await send();
    expect([done.mock.calls.length, underFields(setError), toasts]).toEqual([1, [], []]);
  });

  /** Each row for a creation and for a change alike, the form's two changes. */
  const ofBoth = <R extends object>(rows: R[]) =>
    rows.flatMap((row) => [creation, change].map((by) => Object.assign({ by, change: by.change }, row)));

  it.each(
    ofBoth([
      {
        refused: "a name taken",
        error: refusal(409, "project.state_name_taken"),
        fields: [["name", "errors.project_state_name_taken"]],
      },
      {
        refused: "a name too long",
        error: refusal(422, "validation_failed", [{ field: "name", code: "too_long" }]),
        fields: [["name", "errors.field.too_long"]],
      },
      {
        refused: "a description nerve does not allow",
        error: refusal(422, "validation_failed", [{ field: "description", code: "not_allowed" }]),
        fields: [["description", "errors.field.not_allowed"]],
      },
    ])
  )("shows $refused under its field, of $change, and does not follow it", async ({ by, error, fields }) => {
    by.sent().mockRejectedValueOnce(error);
    await by.send();
    expect([underFields(setError), toasts, done.mock.calls]).toEqual([fields, [], []]);
  });

  it.each(
    ofBoth([
      {
        refused: "a colour nerve does not allow, which the form shows nothing under",
        error: refusal(422, "validation_failed", [{ field: "color", code: "not_allowed" }]),
        message: "errors.validation_failed",
      },
      { refused: "a change by one no longer its admin", error: refusal(403, "forbidden"), message: "errors.forbidden" },
    ])
  )("shows nerve's reason for $refused, of $change, in a toast", async ({ by, error, message }) => {
    by.sent().mockRejectedValueOnce(error);
    await by.send();
    expect([underFields(setError), toasts]).toEqual([[], [{ type: "error", title: "toast.error", message }]]);
  });

  it.each([
    { refused: "the deletion of the default", by: deletion, status: 409, code: "project.state_default" },
    { refused: "the deletion of a group's only state", by: deletion, status: 409, code: "project.state_last_in_group" },
    { refused: "the move of a group's only state", by: move, status: 409, code: "project.state_last_in_group" },
    {
      refused: "a new default of a state deleted meanwhile",
      by: newDefault,
      status: 404,
      code: "project.state_not_found",
    },
  ])("shows nerve's reason when it refuses $refused, in a toast, and does not follow it", async (row) => {
    const { by, status, code } = row;
    by.sent().mockRejectedValueOnce(refusal(status, code));
    await by.send();
    const message = `errors.${code.replace(".", "_")}`;
    expect([toasts, done.mock.calls]).toEqual([[{ type: "error", title: "toast.error", message }], []]);
  });

  describe.each(Object.values(changes))("$change", ({ send, sent, answer }) => {
    it.each(lateSettlingsAnswering(answer))(
      "does nothing on the page when it $settles after another tab moved this one",
      async ({ settle }) => {
        const held = heldChange<State | undefined>();
        sent().mockReturnValueOnce(held.sent);
        const followed = send();
        switchAccount();
        settle(held);
        await followed;
        await pageSettled();
        expect([done.mock.calls, underFields(setError), toasts]).toEqual([[], [], []]);
      }
    );
  });
});
````

`web/apps/web/core/components/project-states/use-state-changes.ts`（新文件，51 行）：

````file web/apps/web/core/components/project-states/use-state-changes.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { StateCreate, StateGroup, StateUpdate } from "@nerve/api-client";
// hooks
import { useProjectState } from "@/hooks/store/use-project-state";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { useFormRefusal } from "@/lib/form-refusal";
import { followInSession } from "@/lib/in-session";

/** The fields of a state's form under which nerve's refusals show, in the order their messages show. */
type StateFormField = "name" | "description";
const FIELDS: readonly StateFormField[] = ["name", "description"];
/** nerve's code for a name another state of the project has, under the name (M3 design 3.17). */
const TAKEN: Readonly<Record<string, StateFormField>> = { "project.state_name_taken": "name" };

/** Where a state's form says a message of nerve's refusal: under the field (its setError). */
export type StateUnderField = (field: StateFormField, message: string) => void;

/**
 * The changes of a project's states that its settings page makes (M3 design 2 P6, 7.6), each through the state store,
 * one at a time (v0 design 7.7). The page follows each only in the session it was sent in (M3 design 7.1): done, the
 * caller's follow-up of success, once nerve has made the change; nerve's refusal of a creation or a change under the
 * form's fields it names (a name taken, under the name), else in a toast, as every refusal of a move, a deletion or a
 * new default is (a default or a group's only state, which the page offers no deletion of, refused all the same when
 * another tab changed them meanwhile). Each settles once the page has followed the change.
 */
export function useStateChanges() {
  const { createState, updateState, moveState, deleteState, markStateAsDefault } = useProjectState();
  const failed = useRefusalToast();
  const showRefusal = useFormRefusal(FIELDS, TAKEN);

  return {
    /** Creates a state in the project of projectId, last of its group. */
    create: (projectId: string, data: StateCreate, underField: StateUnderField, done: () => void) =>
      followInSession(() => createState(projectId, data), { done, failed: (error) => showRefusal(error, underField) }),
    /** Changes the state of stateId to what data names. */
    update: (stateId: string, data: StateUpdate, underField: StateUnderField, done: () => void) =>
      followInSession(() => updateState(stateId, data), { done, failed: (error) => showRefusal(error, underField) }),
    /** Moves the state of stateId into group, before the state of droppedOnId, or after it (StateStore.moveState). */
    move: (stateId: string, group: StateGroup, droppedOnId: string, after: boolean) =>
      followInSession(() => moveState(stateId, group, droppedOnId, after), { failed }),
    /** Deletes the state of stateId. */
    remove: (stateId: string, done: () => void) => followInSession(() => deleteState(stateId), { done, failed }),
    /** Makes the state of stateId its project's default. */
    markDefault: (stateId: string) => followInSession(() => markStateAsDefault(stateId), { failed }),
  };
}
````

- [ ] **Step 2: 表单、创建和修改**

`web/apps/web/core/components/project-states/create-update/create-update.test.tsx`（整个文件，127 行）：

````whole web/apps/web/core/components/project-states/create-update/create-update.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { State } from "@nerve/api-client";
import { heldChange, settledYet, signedIn } from "@/lib/auth/fake-tab";
import { refusal, underFields } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf, stateOf } from "@/store/project/fake-projects";
import type { StateUnderField } from "../use-state-changes";
import { StateCreate } from "./create";
import type { TStateFormData } from "./form";
import { StateUpdate } from "./update";

// What the state settings' creation and change send, and show when nerve refuses them (M3 design 3.17, 7.6; v0 design
// 7.7). The page renders on the server with a stand-in for StateForm, which keeps the props it was given: the test
// submits the form's data as the form would, with the edits it says, and the form's setError; the state store's
// changes are stand-ins nerve answers when the test says. Its session is fake-tab.ts's.

type Form = {
  data: TStateFormData;
  onSubmit: (formData: TStateFormData, underField: StateUnderField) => Promise<void>;
};
const shown = vi.hoisted((): { forms: Form[] } => ({ forms: [] }));
const store = vi.hoisted(() => ({ createState: vi.fn(), updateState: vi.fn() }));
vi.mock("@/components/project-states", () => ({
  StateForm: (props: Form) => {
    shown.forms.push(props);
    return null;
  },
}));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => store }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const web = projectOf("WEB", "w-acme");
const doing = stateOf(web, "Doing", "started", 30000, { description: "Being worked on" });
const setError = vi.fn();
const underField: StateUnderField = (field, message) => setError(field, { message });
const handleClose = vi.fn();

/** Renders the page, and gives the form it showed. */
function formOf(page: ReactElement): Form {
  shown.forms.length = 0;
  renderToStaticMarkup(page);
  const [form] = shown.forms;
  if (!form) throw new Error("the page showed no form");
  return form;
}

const pages = [
  {
    change: "creation",
    page: <StateCreate projectId={web.id} groupKey="started" handleClose={handleClose} />,
    sent: () => store.createState,
    said: "State created successfully.",
  },
  {
    change: "change",
    page: <StateUpdate state={doing} handleClose={handleClose} />,
    sent: () => store.updateState,
    said: "State updated successfully.",
  },
];

beforeEach(() => {
  signedIn();
  store.createState.mockReset();
  store.createState.mockResolvedValue(doing);
  store.updateState.mockReset();
  store.updateState.mockResolvedValue(doing);
  setError.mockReset();
  handleClose.mockReset();
  toasts.length = 0;
});

describe("the state settings' creation and change", () => {
  it("creates a state in the page's project and the form's group, with what the form holds", async () => {
    const form = formOf(<StateCreate projectId={web.id} groupKey="started" handleClose={handleClose} />);
    expect(form.data).toEqual({ name: "", description: "", color: "#f59e0b" });
    await form.onSubmit({ ...form.data, name: "Review" }, underField);
    expect(store.createState.mock.calls).toEqual([
      [web.id, { name: "Review", description: "", color: "#f59e0b", group: "started" }],
    ]);
  });

  it("changes a state with the fields its form edits, the only ones nerve's StateUpdate takes", async () => {
    const form = formOf(<StateUpdate state={doing} handleClose={handleClose} />);
    expect(form.data).toEqual({ name: "Doing", color: "#60646C", description: "Being worked on" });
    await form.onSubmit({ ...form.data, name: "In progress" }, underField);
    expect(store.updateState.mock.calls).toEqual([
      [doing.id, { name: "In progress", color: "#60646C", description: "Being worked on" }],
    ]);
  });

  it.each(pages)("says the $change made and closes the form, the form busy until it has", async (row) => {
    const answer = heldChange<State>();
    row.sent().mockReturnValueOnce(answer.sent);
    const form = formOf(row.page);
    const submitted = form.onSubmit({ ...form.data, name: "Review" }, underField);
    const settled = settledYet(submitted);
    await Promise.resolve();
    expect([settled(), handleClose.mock.calls.length, toasts]).toEqual([false, 0, []]);
    answer.answer(doing);
    await submitted;
    expect([handleClose.mock.calls.length, toasts]).toEqual([
      1,
      [{ type: "success", title: "Success!", message: row.said }],
    ]);
  });

  it.each(pages)("shows nerve's refusal of a $change under the name, and stays open", async ({ page, sent }) => {
    sent().mockRejectedValueOnce(refusal(409, "project.state_name_taken"));
    const form = formOf(page);
    await form.onSubmit({ ...form.data, name: "Todo" }, underField);
    expect([underFields(setError), toasts, handleClose.mock.calls]).toEqual([
      [["name", "errors.project_state_name_taken"]],
      [],
      [],
    ]);
  });
});
````

`web/apps/web/core/components/project-states/create-update/create.tsx`（整个文件，47 行）：

````whole web/apps/web/core/components/project-states/create-update/create.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { STATE_GROUPS } from "@nerve/constants";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { StateGroup } from "@nerve/api-client";
// components
import { StateForm } from "@/components/project-states";
// local imports
import { useStateChanges, type StateUnderField } from "../use-state-changes";
import type { TStateFormData } from "./form";

type TStateCreate = {
  projectId: string;
  groupKey: StateGroup;
  handleClose: () => void;
};

export const StateCreate = observer(function StateCreate(props: TStateCreate) {
  const { projectId, groupKey, handleClose } = props;
  const { create } = useStateChanges();

  // the form closes once nerve has made the state, and stays with what it holds when nerve refuses it
  const onSubmit = (formData: TStateFormData, underField: StateUnderField) =>
    create(projectId, { ...formData, group: groupKey }, underField, () => {
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: "State created successfully.",
      });
      handleClose();
    });

  return (
    <StateForm
      data={{ name: "", description: "", color: STATE_GROUPS[groupKey].color }}
      onSubmit={onSubmit}
      onCancel={handleClose}
      buttonTitle="Create"
      busyButtonTitle="Creating"
    />
  );
});
````

`web/apps/web/core/components/project-states/create-update/form.tsx`（整个文件，139 行）：

````whole web/apps/web/core/components/project-states/create-update/form.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { TwitterPicker } from "react-color";
import { Controller, useForm } from "react-hook-form";
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { TextArea, TextAreaGroup } from "@makeplane/propel/components/text-area";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import type { StateCreate } from "@nerve/api-client";
import { Popover } from "@nerve/ui";
// local imports
import type { StateUnderField } from "../use-state-changes";

/** What the form edits of a state, as a creation sends it. */
export type TStateFormData = Omit<StateCreate, "group">;

type TStateForm = {
  data: TStateFormData;
  /**
   * Sends what the form holds; settles once the page has followed nerve's answer, the form busy until then. nerve's
   * refusal shows under the fields it names, by underField.
   */
  onSubmit: (formData: TStateFormData, underField: StateUnderField) => Promise<void>;
  onCancel: () => void;
  /** What the form's button says, and says while the form is busy. */
  buttonTitle: string;
  busyButtonTitle: string;
};

function PopoverButton({ color }: { color?: string }) {
  return (
    <div
      className="group inline-flex h-5 w-5 items-center rounded-sm text-14 font-medium transition-all focus:outline-none"
      style={{
        backgroundColor: color ?? "black",
      }}
    />
  );
}

export function StateForm(props: TStateForm) {
  const { data, onSubmit, onCancel, buttonTitle, busyButtonTitle } = props;
  const { t } = useTranslation();
  const {
    control,
    handleSubmit,
    setError,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<TStateFormData>({ defaultValues: data });

  // busy from the click until the page has followed nerve's answer: a second click finds the button disabled
  const submit = handleSubmit((formData) =>
    onSubmit(formData, (field, message) => setError(field, { type: "server", message }))
  );

  return (
    <div className="relative flex space-x-2 rounded-sm bg-surface-1 p-3">
      {/* color */}
      <div className="mt-2 h-full flex-shrink-0">
        <Popover button={<PopoverButton color={watch("color")} />} panelClassName="mt-4 -ml-3">
          <Controller
            name="color"
            control={control}
            render={({ field: { value, onChange } }) => (
              <TwitterPicker color={value} onChange={(picked) => onChange(picked.hex)} />
            )}
          />
        </Popover>
      </div>

      <div className="w-full space-y-2">
        {/* title */}
        <Controller
          name="name"
          control={control}
          rules={{ required: t("errors.field.required") }}
          render={({ field: { value, onChange } }) => (
            <Field name="name" invalid={Boolean(errors.name)}>
              <InputGroup size="2xl">
                <Input
                  size="2xl"
                  id="name"
                  type="text"
                  name="name"
                  placeholder="Name"
                  value={value}
                  onChange={onChange}
                  maxLength={100}
                />
              </InputGroup>
            </Field>
          )}
        />
        {errors.name?.message && <span className="text-11 text-danger-primary">{errors.name.message}</span>}

        {/* description */}
        <Controller
          name="description"
          control={control}
          render={({ field: { value, onChange } }) => (
            <Field name="description" invalid={Boolean(errors.description)}>
              <TextAreaGroup resize="none">
                <TextArea
                  size="lg"
                  surface="field"
                  autoResize
                  maxRows={8}
                  id="description"
                  name="description"
                  placeholder="Describe this state for your members."
                  value={value ?? ""}
                  onChange={onChange}
                />
              </TextAreaGroup>
            </Field>
          )}
        />
        {errors.description?.message && (
          <span className="text-11 text-danger-primary">{errors.description.message}</span>
        )}

        <div className="flex items-center space-x-2">
          <Button onClick={submit} variant="primary" size="lg" loading={isSubmitting}>
            {isSubmitting ? busyButtonTitle : buttonTitle}
          </Button>
          <Button type="button" variant="secondary" size="lg" disabled={isSubmitting} onClick={onCancel}>
            Cancel
          </Button>
        </div>
      </div>
    </div>
  );
}
````

`web/apps/web/core/components/project-states/create-update/update.tsx`（整个文件，46 行）：

````whole web/apps/web/core/components/project-states/create-update/update.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { State } from "@nerve/api-client";
// components
import { StateForm } from "@/components/project-states";
// local imports
import { useStateChanges, type StateUnderField } from "../use-state-changes";
import type { TStateFormData } from "./form";

type TStateUpdate = {
  state: State;
  handleClose: () => void;
};

export const StateUpdate = observer(function StateUpdate(props: TStateUpdate) {
  const { state, handleClose } = props;
  const { update } = useStateChanges();

  // the form closes once nerve has made the change, and stays with what it holds when nerve refuses it
  const onSubmit = (formData: TStateFormData, underField: StateUnderField) =>
    update(state.id, formData, underField, () => {
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: "Success!",
        message: "State updated successfully.",
      });
      handleClose();
    });

  return (
    <StateForm
      // the fields the form edits, all a change sends: nerve's StateUpdate takes no others
      data={{ name: state.name, color: state.color, description: state.description }}
      onSubmit={onSubmit}
      onCancel={handleClose}
      buttonTitle="Update"
      busyButtonTitle="Updating"
    />
  );
});
````

- [ ] **Step 3: 删除和设默认**

`web/apps/web/core/components/project-states/options/delete.test.tsx`（新文件，75 行）：

````file web/apps/web/core/components/project-states/options/delete.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactNode } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { pageSettled, signedIn } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { projectOf, stateOf } from "@/store/project/fake-projects";
import { StateDelete } from "./delete";

// Which states the state settings offer to delete, and what confirming it sends (M3 design 2 P6, 3.17, 7.6): the
// button renders on the server, with a stand-in for its dialog (fake-controls.ts), which keeps the props it was given;
// the test presses the dialog's button, as a person would. A default and a group's only state cannot be deleted, as
// nerve refuses (409), and their button is disabled; nerve's refusal all the same (another tab changed them meanwhile)
// is said in a toast. Its session is fake-tab.ts's.

const store = vi.hoisted(() => ({ deleteState: vi.fn() }));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => store }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@makeplane/propel/components/tooltip", () => ({
  Tooltip: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const web = projectOf("WEB", "w-acme");
const review = stateOf(web, "Review", "started", 70000);

/** Renders the deletion of state among totalStates of its group: whether its button is disabled, and its dialog. */
function deletionOf(state = review, totalStates = 2) {
  emptyShown();
  const markup = renderToStaticMarkup(<StateDelete state={state} totalStates={totalStates} />);
  const [alert] = shown.alerts;
  if (!alert) throw new Error("no dialog");
  return { disabled: /<button[^>]* disabled=""/.test(markup), alert };
}

beforeEach(() => {
  signedIn();
  store.deleteState.mockReset();
  store.deleteState.mockResolvedValue(undefined);
  toasts.length = 0;
});

describe("StateDelete", () => {
  it.each([
    { state: "the default", deleted: { ...review, default: true }, totalStates: 2 },
    { state: "its group's only state", deleted: review, totalStates: 1 },
  ])("offers no deletion of $state, and sends none", async ({ deleted, totalStates }) => {
    const { disabled, alert } = deletionOf(deleted, totalStates);
    await alert.handleSubmit();
    expect([disabled, store.deleteState.mock.calls]).toEqual([true, []]);
  });

  it("deletes a state its group has another of, once its dialog is confirmed, the dialog closed until then", async () => {
    const { disabled, alert } = deletionOf();
    expect([disabled, alert.isOpen, store.deleteState.mock.calls]).toEqual([false, false, []]);
    await alert.handleSubmit();
    expect([store.deleteState.mock.calls, toasts]).toEqual([[[review.id]], []]);
  });

  it("says nerve's reason when it refuses the deletion", async () => {
    store.deleteState.mockRejectedValueOnce(refusal(409, "project.state_default"));
    await deletionOf().alert.handleSubmit();
    await pageSettled();
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.project_state_default" }]);
  });
});
````

`web/apps/web/core/components/project-states/options/delete.tsx`（修改，8 处）：

````old web/apps/web/core/components/project-states/options/delete.tsx
import type { State } from "@nerve/api-client";
import type { TStateOperationsCallbacks } from "@nerve/types";
````
````new web/apps/web/core/components/project-states/options/delete.tsx
import type { State } from "@nerve/api-client";
````
````old web/apps/web/core/components/project-states/options/delete.tsx
import { useRefusalToast } from "@/hooks/use-refusal-toast";
````
````new web/apps/web/core/components/project-states/options/delete.tsx
// local imports
import { useStateChanges } from "../use-state-changes";
````
````old web/apps/web/core/components/project-states/options/delete.tsx
  state: State;
  deleteStateCallback: TStateOperationsCallbacks["deleteState"];
````
````new web/apps/web/core/components/project-states/options/delete.tsx
  state: State;
````
````old web/apps/web/core/components/project-states/options/delete.tsx
  const { totalStates, state, deleteStateCallback } = props;
````
````new web/apps/web/core/components/project-states/options/delete.tsx
  const { totalStates, state } = props;
````
````old web/apps/web/core/components/project-states/options/delete.tsx
  const toastRefusal = useRefusalToast();
````
````new web/apps/web/core/components/project-states/options/delete.tsx
  const { remove } = useStateChanges();
````
````old web/apps/web/core/components/project-states/options/delete.tsx
  const isDeleteDisabled = state.default || totalStates === 1;

````
````new web/apps/web/core/components/project-states/options/delete.tsx
  const isDeleteDisabled = state.default || totalStates === 1;

  // Busy, and the dialog not to be dismissed, until the page has followed nerve's answer: it closes once nerve has
  // deleted the state, and stays, its buttons enabled again, when nerve refuses, the reason in a toast.
````
````old web/apps/web/core/components/project-states/options/delete.tsx
    if (isDeleteDisabled) return;

````
````new web/apps/web/core/components/project-states/options/delete.tsx
    if (isDeleteDisabled) return;
````
````old web/apps/web/core/components/project-states/options/delete.tsx

    try {
      await deleteStateCallback(state.id);
      setIsDelete(false);
    } catch (error) {
      toastRefusal(error);
      setIsDelete(false);
    }
````
````new web/apps/web/core/components/project-states/options/delete.tsx
    await remove(state.id, () => setIsDeleteModal(false));
    setIsDelete(false);
````

`web/apps/web/core/components/project-states/options/mark-as-default.tsx`（整个文件，46 行）：

````whole web/apps/web/core/components/project-states/options/mark-as-default.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import { cn } from "@nerve/utils";
// local imports
import { useStateChanges } from "../use-state-changes";

type TStateMarksAsDefault = {
  stateId: string;
  isDefault: boolean;
};

export const StateMarksAsDefault = observer(function StateMarksAsDefault(props: TStateMarksAsDefault) {
  const { stateId, isDefault } = props;
  const { markDefault } = useStateChanges();
  // states
  const [isLoading, setIsLoading] = useState(false);

  // busy until the page has followed nerve's answer; a refusal is said in a toast
  const handleMarkAsDefault = async () => {
    if (isDefault) return;
    setIsLoading(true);
    await markDefault(stateId);
    setIsLoading(false);
  };

  return (
    <button
      type="button"
      className={cn(
        "text-11 whitespace-nowrap transition-colors",
        isDefault ? "text-tertiary" : "text-secondary hover:text-primary"
      )}
      disabled={isDefault || isLoading}
      onClick={handleMarkAsDefault}
    >
      {isLoading ? "Marking as default" : isDefault ? `Default` : `Mark as default`}
    </button>
  );
});
````

- [ ] **Step 4: 组件不再收回调；组标题是按钮**

`web/apps/web/core/components/project-states/group-item.tsx`（整个文件，107 行）：

````whole web/apps/web/core/components/project-states/group-item.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState, useRef } from "react";
import { observer } from "mobx-react";

// nerve imports
import { EIconSize } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { StateGroupIcon } from "@nerve/propel/icons";
import { AddOutline, ChevronDownOutline } from "@makeplane/propel/icons";
import type { State, StateGroup } from "@nerve/api-client";
import { cn } from "@nerve/utils";
// components
import { StateList, StateCreate } from "@/components/project-states";

type TGroupItem = {
  projectId: string;
  groupKey: StateGroup;
  groupsExpanded: Partial<StateGroup>[];
  groupedStates: Record<string, State[]>;
  states: State[];
  isEditable: boolean;
  handleGroupCollapse: (groupKey: StateGroup) => void;
  handleExpand: (groupKey: StateGroup) => void;
};

export const GroupItem = observer(function GroupItem(props: TGroupItem) {
  const { projectId, groupKey, groupedStates, states, groupsExpanded, isEditable, handleExpand, handleGroupCollapse } =
    props;
  // refs
  const dropElementRef = useRef<HTMLDivElement | null>(null);
  // nerve hooks
  const { t } = useTranslation();
  // state
  const [createState, setCreateState] = useState(false);
  // derived values
  const currentStateExpanded = groupsExpanded.includes(groupKey);
  const shouldShowEmptyState = states.length === 0 && currentStateExpanded && !createState;

  return (
    <div className="space-y-1 rounded-sm border border-subtle bg-surface-2 p-2 transition-all" ref={dropElementRef}>
      <div className="flex items-center justify-between gap-2">
        <button
          type="button"
          className="flex w-full cursor-pointer items-center py-1"
          aria-expanded={currentStateExpanded}
          onClick={() => (!currentStateExpanded ? handleExpand(groupKey) : handleGroupCollapse(groupKey))}
        >
          <span
            className={cn(
              "flex h-5 w-5 flex-shrink-0 items-center justify-center overflow-hidden rounded-sm transition-all",
              {
                "rotate-0": currentStateExpanded,
                "-rotate-90": !currentStateExpanded,
              }
            )}
          >
            <ChevronDownOutline className="h-4 w-4" />
          </span>
          <span className="flex h-6 w-6 flex-shrink-0 items-center justify-center overflow-hidden rounded-sm">
            <StateGroupIcon stateGroup={groupKey} size={EIconSize.XL} />
          </span>
          <span className="px-1 text-14 font-medium text-secondary capitalize">{groupKey}</span>
        </button>
        <button
          type="button"
          className={cn(
            "flex h-6 w-6 flex-shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-sm text-accent-primary/80 transition-colors hover:bg-layer-1 hover:text-accent-primary",
            (!isEditable || createState) && "cursor-not-allowed text-placeholder hover:text-placeholder"
          )}
          onClick={() => {
            if (!createState) {
              handleExpand(groupKey);
              setCreateState(true);
            }
          }}
          disabled={!isEditable || createState}
        >
          <AddOutline className="h-4 w-4" />
        </button>
      </div>

      {shouldShowEmptyState && (
        <div className="flex h-full flex-col items-center justify-center py-4 text-13 text-tertiary">
          <div>{t("project_settings.states.empty_state.title", { groupKey })}</div>
          {isEditable && <div>{t("project_settings.states.empty_state.description")}</div>}
        </div>
      )}

      {currentStateExpanded && (
        <div id="group-droppable-container">
          <StateList groupKey={groupKey} groupedStates={groupedStates} states={states} disabled={!isEditable} />
        </div>
      )}

      {isEditable && createState && (
        <div className="">
          <StateCreate projectId={projectId} groupKey={groupKey} handleClose={() => setCreateState(false)} />
        </div>
      )}
    </div>
  );
});
````

`web/apps/web/core/components/project-states/group-list.tsx`（修改，7 处）：

````old web/apps/web/core/components/project-states/group-list.tsx
import type { State, StateGroup } from "@nerve/api-client";
import type { TStateOperationsCallbacks } from "@nerve/types";
import { cn } from "@nerve/utils";
````
````new web/apps/web/core/components/project-states/group-list.tsx
import type { State, StateGroup } from "@nerve/api-client";
````
````old web/apps/web/core/components/project-states/group-list.tsx
type TGroupList = {
  groupedStates: Record<string, State[]>;
  stateOperationsCallbacks: TStateOperationsCallbacks;
  isEditable: boolean;
  groupListClassName?: string;
  groupItemClassName?: string;
  stateItemClassName?: string;
````
````new web/apps/web/core/components/project-states/group-list.tsx
type TGroupList = {
  projectId: string;
  groupedStates: Record<string, State[]>;
  isEditable: boolean;
````
````old web/apps/web/core/components/project-states/group-list.tsx
  const {
    groupedStates,
    stateOperationsCallbacks,
    isEditable,
    groupListClassName,
    groupItemClassName,
    stateItemClassName,
  } = props;
````
````new web/apps/web/core/components/project-states/group-list.tsx
  const { projectId, groupedStates, isEditable } = props;
````
````old web/apps/web/core/components/project-states/group-list.tsx
    <div className={cn("space-y-5", groupListClassName)}>
````
````new web/apps/web/core/components/project-states/group-list.tsx
    <div className="space-y-5">
````
````old web/apps/web/core/components/project-states/group-list.tsx
            key={groupKey}
````
````new web/apps/web/core/components/project-states/group-list.tsx
            key={groupKey}
            projectId={projectId}
````
````old web/apps/web/core/components/project-states/group-list.tsx
            groupsExpanded={groupsExpanded}
            stateOperationsCallbacks={stateOperationsCallbacks}
````
````new web/apps/web/core/components/project-states/group-list.tsx
            groupsExpanded={groupsExpanded}
````
````old web/apps/web/core/components/project-states/group-list.tsx
            handleExpand={handleExpand}
            groupItemClassName={groupItemClassName}
            stateItemClassName={stateItemClassName}
````
````new web/apps/web/core/components/project-states/group-list.tsx
            handleExpand={handleExpand}
````

`web/apps/web/core/components/project-states/root.tsx`（整个文件，42 行）：

````whole web/apps/web/core/components/project-states/root.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// components
import { EUserPermissionsLevel } from "@nerve/constants";
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
  const { groupedProjectStates, fetchProjectStates } = useProjectState();
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

  // Loader
  if (!groupedProjectStates) return <ProjectStateLoader />;

  return <GroupList projectId={projectId} groupedStates={groupedProjectStates} isEditable={isEditable} />;
});
````

`web/apps/web/core/components/project-states/state-item-title.tsx`（整个文件，73 行）：

````whole web/apps/web/core/components/project-states/state-item-title.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { SetStateAction } from "react";
import { observer } from "mobx-react";
import { DragDropOutline, EditOutline } from "@makeplane/propel/icons";
import { EIconSize } from "@nerve/constants";
// nerve imports
import { StateGroupIcon } from "@nerve/propel/icons";
import type { State } from "@nerve/api-client";
// local imports
import { useProjectState } from "@/hooks/store/use-project-state";
import { StateDelete, StateMarksAsDefault } from "./options";

type TStateItemTitleProps = {
  stateCount: number;
  state: State;
  setUpdateStateModal: (value: SetStateAction<boolean>) => void;
  /** Whether the state is shown alone, with none of the controls that change it. */
  disabled: boolean;
};

export const StateItemTitle = observer(function StateItemTitle(props: TStateItemTitleProps) {
  const { stateCount, setUpdateStateModal, disabled, state } = props;
  // store hooks
  const { getStatePercentageInGroup } = useProjectState();
  // derived values
  const statePercentage = getStatePercentageInGroup(state.id);
  const percentage = statePercentage ? statePercentage / 100 : undefined;

  return (
    <div className="flex w-full items-center justify-between gap-2">
      <div className="flex items-center gap-1 px-1">
        {/* draggable indicator */}
        {!disabled && stateCount != 1 && (
          <div className="absolute -left-1.5 hidden h-3 w-3 flex-shrink-0 cursor-pointer items-center justify-center rounded-xs bg-surface-2 text-secondary transition-colors group-hover:flex hover:text-primary">
            <DragDropOutline className="h-3 w-3" />
          </div>
        )}
        {/* state icon */}
        <div className="flex-shrink-0">
          <StateGroupIcon stateGroup={state.group} color={state.color} size={EIconSize.XL} percentage={percentage} />
        </div>
        {/* state title and description */}
        <div className="min-h-5 px-2 text-13">
          <h6 className="text-13 font-medium">{state.name}</h6>
          <p className="text-11 text-secondary">{state.description}</p>
        </div>
      </div>
      {!disabled && (
        <div className="hidden items-center gap-2 group-hover:flex">
          {/* state mark as default option */}
          <div className="flex-shrink-0 text-11 transition-all">
            <StateMarksAsDefault stateId={state.id} isDefault={state.default} />
          </div>
          {/* state edit options */}
          <div className="flex items-center gap-1 transition-all">
            <button
              className="flex h-5 w-5 flex-shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-sm text-secondary transition-colors hover:bg-layer-1 hover:text-primary"
              onClick={() => setUpdateStateModal(true)}
            >
              <EditOutline className="h-3 w-3" />
            </button>
            <StateDelete totalStates={stateCount} state={state} />
          </div>
        </div>
      )}
    </div>
  );
});
````

`web/apps/web/core/components/project-states/state-item.tsx`（整个文件，116 行）：

````whole web/apps/web/core/components/project-states/state-item.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment, useEffect, useRef, useState } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { draggable, dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/element/adapter";
import { attachClosestEdge, extractClosestEdge } from "@atlaskit/pragmatic-drag-and-drop-hitbox/closest-edge";
import { observer } from "mobx-react";
// Nerve
import type { TDraggableData } from "@nerve/constants";
import type { State, StateGroup } from "@nerve/api-client";
import { DropIndicator } from "@nerve/ui";
import { cn } from "@nerve/utils";
// components
import { StateItemTitle, StateUpdate } from "@/components/project-states";
// local imports
import { useStateChanges } from "./use-state-changes";

type TStateItem = {
  groupKey: StateGroup;
  groupedStates: Record<string, State[]>;
  totalStates: number;
  state: State;
  disabled?: boolean;
};

export const StateItem = observer(function StateItem(props: TStateItem) {
  const { groupKey, groupedStates, totalStates, state, disabled = false } = props;
  const { move } = useStateChanges();
  // ref
  const draggableElementRef = useRef<HTMLDivElement | null>(null);
  // states
  const [updateStateModal, setUpdateStateModal] = useState(false);
  const [isDragging, setIsDragging] = useState(false);
  const [isDraggedOver, setIsDraggedOver] = useState(false);
  const [closestEdge, setClosestEdge] = useState<string | null>(null);
  // derived values
  const isDraggable = totalStates !== 1;
  const commonStateItemListProps = {
    stateCount: totalStates,
    state: state,
    setUpdateStateModal: setUpdateStateModal,
  };

  useEffect(() => {
    const elementRef = draggableElementRef.current;
    const initialData: TDraggableData = { groupKey: groupKey, id: state.id };

    if (!elementRef) return;

    // the cleanup: each run registers the element again, with the handlers of this render
    return combine(
      draggable({
        element: elementRef,
        getInitialData: () => initialData,
        onDragStart: () => setIsDragging(true),
        onDrop: () => setIsDragging(false),
        canDrag: () => isDraggable && !disabled,
      }),
      dropTargetForElements({
        element: elementRef,
        getData: ({ input, element }) =>
          attachClosestEdge(initialData, {
            input,
            element,
            allowedEdges: ["top", "bottom"],
          }),
        onDragEnter: (args) => {
          setIsDraggedOver(true);
          setClosestEdge(extractClosestEdge(args.self.data));
        },
        onDragLeave: () => {
          setIsDraggedOver(false);
          setClosestEdge(null);
        },
        onDrop: (data) => {
          setIsDraggedOver(false);
          const { self, source } = data;
          const sourceData = source.data as TDraggableData;
          const destinationData = self.data as TDraggableData;

          if (sourceData && destinationData && sourceData.id) {
            const destinationGroupKey = destinationData.groupKey;
            const edge = extractClosestEdge(destinationData) || undefined;
            void move(sourceData.id, destinationGroupKey, destinationData.id, edge === "bottom");
          }
        },
      })
    );
  }, [draggableElementRef, state, groupKey, isDraggable, groupedStates, move, disabled]);
  // DND ends

  if (updateStateModal) return <StateUpdate state={state} handleClose={() => setUpdateStateModal(false)} />;

  return (
    <Fragment>
      {/* draggable drop top indicator */}
      <DropIndicator isVisible={isDraggedOver && closestEdge === "top"} />
      <div
        ref={draggableElementRef}
        className={cn(
          "group relative rounded-sm border border-subtle bg-surface-1 px-3.5 py-3",
          isDragging ? `opacity-50` : `opacity-100`,
          totalStates === 1 ? `cursor-auto` : `cursor-grab`
        )}
      >
        <StateItemTitle {...commonStateItemListProps} disabled={disabled} />
      </div>
      {/* draggable drop bottom indicator */}
      <DropIndicator isVisible={isDraggedOver && closestEdge === "bottom"} />
    </Fragment>
  );
});
````

`web/apps/web/core/components/project-states/state-list.tsx`（整个文件，36 行）：

````whole web/apps/web/core/components/project-states/state-list.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import type { State, StateGroup } from "@nerve/api-client";
// components
import { StateItem } from "@/components/project-states";

type TStateList = {
  groupKey: StateGroup;
  groupedStates: Record<string, State[]>;
  states: State[];
  disabled?: boolean;
};

export const StateList = observer(function StateList(props: TStateList) {
  const { groupKey, groupedStates, states, disabled = false } = props;

  return (
    <>
      {states.map((state: State) => (
        <StateItem
          key={state.id}
          groupKey={groupKey}
          groupedStates={groupedStates}
          totalStates={states.length || 0}
          state={state}
          disabled={disabled}
        />
      ))}
    </>
  );
});
````

- [ ] **Step 5: 回调的类型、替身、上限和静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/lib/form-refusal.ts"
````
````new .oxlintrc.json
        "web/apps/web/core/lib/form-refusal.ts",
        "web/apps/web/core/components/project-states/use-state-changes.ts",
        "web/apps/web/core/components/project-states/use-state-changes.test.ts",
        "web/apps/web/core/components/project-states/options/delete.test.tsx"
````

`web/apps/web/core/lib/fake-controls.ts`（修改，6 处）：

````old web/apps/web/core/lib/fake-controls.ts
// A stand-in for the UI kit's controls a page renders (@nerve/ui's selects, modal and context menu, propel's input,
````
````new web/apps/web/core/lib/fake-controls.ts
// A stand-in for the UI kit's controls a page renders (@nerve/ui's selects, modals and context menu, propel's input,
````
````old web/apps/web/core/lib/fake-controls.ts
type Toggle = { onCheckedChange: (checked: boolean) => unknown; disabled?: boolean };
````
````new web/apps/web/core/lib/fake-controls.ts
type Toggle = { onCheckedChange: (checked: boolean) => unknown; disabled?: boolean };
/** A confirmation's dialog (AlertModalCore): whether it shows, its button's press, and whether it is busy. */
type Alert = { isOpen: boolean; handleSubmit: () => unknown; isSubmitting: boolean };
````
````old web/apps/web/core/lib/fake-controls.ts
  modals: { children?: ReactNode }[];
````
````new web/apps/web/core/lib/fake-controls.ts
  modals: { children?: ReactNode }[];
  alerts: Alert[];
````
````old web/apps/web/core/lib/fake-controls.ts
} = { inputs: [], buttons: [], selects: [], searchSelects: [], switches: [], modals: [], menus: [], menuItems: [] };
````
````new web/apps/web/core/lib/fake-controls.ts
} = {
  inputs: [],
  buttons: [],
  selects: [],
  searchSelects: [],
  switches: [],
  modals: [],
  alerts: [],
  menus: [],
  menuItems: [],
};
````
````old web/apps/web/core/lib/fake-controls.ts
    modals: [],
````
````new web/apps/web/core/lib/fake-controls.ts
    modals: [],
    alerts: [],
````
````old web/apps/web/core/lib/fake-controls.ts
}

export const EModalPosition = { CENTER: "center" };
````
````new web/apps/web/core/lib/fake-controls.ts
}

export function AlertModalCore(props: Alert) {
  shown.alerts.push(props);
  return null;
}

export const EModalPosition = { CENTER: "center" };
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 348",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 346",
````

`web/packages/types/src/index.ts`（修改，1 处）：

````old web/packages/types/src/index.ts
export * from "./settings";
export * from "./state";
````
````new web/packages/types/src/index.ts
export * from "./settings";
````

`web/packages/types/src/state.ts`（删除）：

````delete web/packages/types/src/state.ts
````

- [ ] **Step 6: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 69 条规则、3 个例外，没有命中；web 的 oxlint 346 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 106 个全部通过。

- [ ] **Step 7: 提交**

```bash
git add .oxlintrc.json web/apps/web/core/components/project-states/create-update/create-update.test.tsx web/apps/web/core/components/project-states/create-update/create.tsx web/apps/web/core/components/project-states/create-update/form.tsx web/apps/web/core/components/project-states/create-update/update.tsx web/apps/web/core/components/project-states/group-item.tsx web/apps/web/core/components/project-states/group-list.tsx web/apps/web/core/components/project-states/options/delete.test.tsx web/apps/web/core/components/project-states/options/delete.tsx web/apps/web/core/components/project-states/options/mark-as-default.tsx web/apps/web/core/components/project-states/root.tsx web/apps/web/core/components/project-states/state-item-title.tsx web/apps/web/core/components/project-states/state-item.tsx web/apps/web/core/components/project-states/state-list.tsx web/apps/web/core/components/project-states/use-state-changes.test.ts web/apps/web/core/components/project-states/use-state-changes.ts web/apps/web/core/lib/fake-controls.ts web/apps/web/package.json web/packages/types/src/index.ts web/packages/types/src/state.ts
```
```bash
git commit -m "feat(M3/P11): the state settings follow each change in the session it was sent in, and their form and dialog wait for it

useStateChanges sends each change of a project's states through the store
and follows it with followInSession; a creation's or a change's refusal
shows under the field it names, the others in a toast. The state form
stays busy, Cancel too, until the page has followed nerve's answer, and
stays open on a refusal; the deletion's dialog closes only once the state
is deleted. The components take no callbacks, a state's row is keyed by
its id, and a group's heading is a button.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p11.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T2.1` | 状态的创建不论会话都跟进 | `use-state-changes.test.ts` | vitest |
| `T2.2` | 状态的修改不论会话都跟进 | `use-state-changes.test.ts` | vitest |
| `T2.3` | 状态的移动不论会话都跟进 | `use-state-changes.test.ts` | vitest |
| `T2.4` | 状态的删除不论会话都跟进 | `use-state-changes.test.ts` | vitest |
| `T2.5` | 设默认不论会话都跟进 | `use-state-changes.test.ts` | vitest |
| `T2.6` | 状态的名字已被占用在提示里，不在名字下 | `use-state-changes.test.ts`、故事 P6（Task 3 起） | vitest；端到端 |
| `T2.7` | 状态的表单在 nerve 回答之前不忙 | 故事 P6（Task 3 起） | 端到端 |
| `T2.8` | 状态的表单在字段下什么都不显示 | 故事 P6（Task 3 起） | 端到端 |
| `T2.9` | 状态的表单在 nerve 回答之前取消仍可用 | 故事 P6（Task 3 起） | 端到端 |
| `T2.10` | 默认的状态提供删除 | `delete.test.tsx`、故事 P6（Task 3 起） | vitest；端到端 |
| `T2.11` | 删除状态的确认框在 nerve 拒绝时关上（A-M2） | 故事 P6（Task 3 起） | 端到端 |
| `T2.12` | 组标题又是 `div`：Tab 到不了，没有展开的状态 | 故事 P6（Task 3 起） | 端到端 |
| `T2.13` | 在 `use-state-changes.ts` 里写一个非空断言 | oxlint（`check:lint`） | 静态 |

---

### Task 3: 状态设置页从路由读项目，控件给谁和怎样到达；P6、W11 的页面版本

**Files:**
- Create: `e2e/stories/project/p6-states-page.spec.ts`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx`
- Modify: `.oxlintrc.json`、`e2e/fixtures/project-pages.ts`、`e2e/stories/project/p6-states.spec.ts`、`e2e/stories/project/p8-project-preferences.spec.ts`、`e2e/stories/smoke/s2-web-app.spec.ts`、`e2e/stories/workspace/w11-guest-bounds.spec.ts`、`tools/keywords.json`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx`、`web/apps/web/core/components/project-states/group-item.tsx`、`web/apps/web/core/components/project-states/options/delete.tsx`、`web/apps/web/core/components/project-states/root.tsx`、`web/apps/web/core/components/project-states/state-item-title.tsx`、`web/apps/web/core/store/state.store.test.ts`、`web/apps/web/core/store/state.store.ts`

**Interfaces:**
- Produces（spec 2.3；M3 设计 2 的 P6、W11，7.6，9.2；P10 的 F-2 和 `project-address-route`；P10 review 第 6 节 P11 一行）：
  - 状态设置页（`states/page.tsx`）：工作区和项目取自路由的参数（`Route.ComponentProps` 的 `params`），标题用 `getProjectById(projectId)` 的名字；门是 `allowPermissions([ADMIN, MEMBER], PROJECT, workspaceSlug, projectId)`：项目的成员（不论角色）看得到状态，访客看到没有权限（`state.list`，W11）。原来读 router store 的 `currentProjectDetails`，`allowPermissions` 不写工作区和项目。
  - `ProjectStateRoot`（`root.tsx`）：项目的状态按路由的项目读，`getGroupedProjectStates(projectId)`（state store 的 `computedFn`，每个组都在，组里按顺序；原来是读 router store 的项目的 `groupedProjectStates` getter，删除）；改状态的控件只给 `allowPermissions([ADMIN], PROJECT, workspaceSlug, projectId)`：项目管理员和是工作区管理员的项目成员（`state.create` 等，nerve 的规则）。挂载时照旧取项目的状态（P8b 的选择）。
  - 一行的控件（`state-item-title.tsx`）：设默认、编辑、删除的按钮 Tab 能到，焦点在这一行时和指针经过时一样显示（`opacity-0 group-focus-within:opacity-100 group-hover:opacity-100`；原来 `hidden group-hover:flex`，键盘到不了）；编辑按钮（"Edit"）、删除按钮（"Delete"，`delete.tsx`）、加状态的按钮（"Add"，`group-item.tsx`）有名字。
  - `tools/keywords.json`：`project-address-route` 的文件加上状态设置页和 `project-states/root.tsx`（69 条规则不变）。
  - e2e：P8 故事的拖动（按手柄、等过渡结束、经过项目本身再到目标）移进 `e2e/fixtures/project-pages.ts` 的 `dragOnto(page, item, grip, onto, at?)`，P6、P7 的页面版本也用它；P8 故事里找侧边栏的项目、拖动、读回答的一步改名 `dragInSidebar`，拖动经 `dragOnto`；`tabCycle(page, within, presses = 40)`：页面上的表单不是弹窗，Tab 会离开它，P6、P7 只看前几站的次序。
  - 静态检查：`.oxlintrc.json` 的非空断言的范围加上 `states-labels.test.tsx`（方括号转义）。

**Tests:**
- vitest：`[projectId]/states-labels.test.tsx`（新文件；P10 的 `admin-pages.test.tsx` 的写法：页面经 React Router 的 stub 在服务端渲染，权限和项目是 store 自己的，`callerInWeb`、`its`，router store 的地址里没有项目）：`shows $who the states and their controls`（项目管理员、是工作区管理员的项目成员、是工作区管理员的项目访客）；`shows its member the states, and no control`；`tells its guest he may not see them`。`state.store.test.ts` 的三处读 `getGroupedProjectStates(web.id)`。
- 端到端：`P6 (page): the admin creates Review in the started group, its form gone through by Tab and held until nerve answers; a name taken is said under the name; he recolours Review, drags it before In Progress and makes it the default, Backlog the default no longer; the default and a group's only state offer no deletion; a deletion nerve refuses leaves its dialog open, one under way cannot be dismissed, and In Progress is deleted; its member sees the states and can change none, a workspace admin who is its member can`（新文件：五个组的标题 `aria-expanded`；表单的 Tab 次序 `name`、`description`、`Create`、`Cancel`；创建扣住时"Creating"和"Cancel"都按不了，请求体是 `StateCreate`；名字已被占用在表单里显示；Review 的一行从"Mark as default"按 Tab 到"Edit"，这一行的控件在焦点进来之前看不见、之后看得见；修改的请求体；拖到 In Progress 的上方：`{ group, sequence: 20000 }`；设默认；默认的、组里唯一的状态的删除按钮禁用；另一处把 In Progress 设为默认之后删除它被拒绝：提示，确认框留着；重新载入之后删除扣住时 Escape 不关、"Cancel"按不了；`expectStates` 核对每一行；成员看不到改状态的控件，是工作区管理员的成员看得到）。`W11 (page): a guest signs in: his sidebar lists Web, the project he is a guest of, not Secret; the workspace's members page and Web's state settings tell him he may not see them; no page he opens shows an address, his own neither`（新的一条）。S2 的挂载清单加上状态设置页（spec 2.11）。

- [ ] **Step 1: 状态的 getter 按项目的 id**

`web/apps/web/core/store/state.store.test.ts`（修改，3 处）：

````old web/apps/web/core/store/state.store.test.ts
    expect(store.groupedProjectStates).toEqual({
````
````new web/apps/web/core/store/state.store.test.ts
    expect(store.getGroupedProjectStates(web.id)).toEqual({
````
````old web/apps/web/core/store/state.store.test.ts
    expect(store.groupedProjectStates?.started).toEqual([doing, review, blocked]);
````
````new web/apps/web/core/store/state.store.test.ts
    expect(store.getGroupedProjectStates(web.id)?.started).toEqual([doing, review, blocked]);
````
````old web/apps/web/core/store/state.store.test.ts
    expect(store.groupedProjectStates?.started).toEqual([reviewFirst, doingAfter]);
````
````new web/apps/web/core/store/state.store.test.ts
    expect(store.getGroupedProjectStates(web.id)?.started).toEqual([reviewFirst, doingAfter]);
````

`web/apps/web/core/store/state.store.ts`（修改，5 处）：

````old web/apps/web/core/store/state.store.ts
  projectStates: State[] | undefined;
  /** The address's project's states in each group, every group there; undefined until fetched. */
  groupedProjectStates: Record<string, State[]> | undefined;
````
````new web/apps/web/core/store/state.store.ts
  projectStates: State[] | undefined;
````
````old web/apps/web/core/store/state.store.ts
  getProjectStates: (projectId: string | null | undefined) => State[] | undefined;
````
````new web/apps/web/core/store/state.store.ts
  getProjectStates: (projectId: string | null | undefined) => State[] | undefined;
  getGroupedProjectStates: (projectId: string | null | undefined) => Record<string, State[]> | undefined;
````
````old web/apps/web/core/store/state.store.ts
      projectStates: computed,
      groupedProjectStates: computed,
````
````new web/apps/web/core/store/state.store.ts
      projectStates: computed,
````
````old web/apps/web/core/store/state.store.ts

  get groupedProjectStates() {
    const states = this.projectStates;
    if (!states) return undefined;
    return Object.fromEntries(
      Object.keys(STATE_GROUPS).map((group) => [group, states.filter((state) => state.group === group)])
    );
  }

````
````new web/apps/web/core/store/state.store.ts

````
````old web/apps/web/core/store/state.store.ts
    return listed && sortStates(listed);
````
````new web/apps/web/core/store/state.store.ts
    return listed && sortStates(listed);
  });

  /** @description the project's states in each group, every group there, in their order; undefined as getProjectStates */
  getGroupedProjectStates = computedFn((projectId: string | null | undefined): Record<string, State[]> | undefined => {
    const states = this.getProjectStates(projectId);
    if (!states) return undefined;
    return Object.fromEntries(
      Object.keys(STATE_GROUPS).map((group) => [group, states.filter((state) => state.group === group)])
    );
````

- [ ] **Step 2: 状态设置页和它的根组件**

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx`（修改，4 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx
function StatesSettingsPage({ params }: Route.ComponentProps) {
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx
function StatesSettingsPage({ params }: Route.ComponentProps) {
  // router: the address's project, as the store gives it (the wrapper shows the page once it has)
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx
  const { currentProjectDetails } = useProject();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx
  const { getProjectById } = useProject();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx
  const pageTitle = currentProjectDetails?.name ? `${currentProjectDetails?.name} - States` : undefined;
  // derived values
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx
  const project = getProjectById(projectId);
  const pageTitle = project ? `${project.name} - States` : undefined;
  // its members see its states; its guests are told they may not (M3 design 2 W11)
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx
    EUserPermissionsLevel.PROJECT
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx
    EUserPermissionsLevel.PROJECT,
    workspaceSlug,
    projectId
````

`web/apps/web/core/components/project-states/root.tsx`（修改，4 处）：

````old web/apps/web/core/components/project-states/root.tsx
  const { groupedProjectStates, fetchProjectStates } = useProjectState();
````
````new web/apps/web/core/components/project-states/root.tsx
  const { getGroupedProjectStates, fetchProjectStates } = useProjectState();
````
````old web/apps/web/core/components/project-states/root.tsx
  // derived values
````
````new web/apps/web/core/components/project-states/root.tsx
  // derived values
  const groupedStates = getGroupedProjectStates(projectId);
````
````old web/apps/web/core/components/project-states/root.tsx
  if (!groupedProjectStates) return <ProjectStateLoader />;
````
````new web/apps/web/core/components/project-states/root.tsx
  if (!groupedStates) return <ProjectStateLoader />;
````
````old web/apps/web/core/components/project-states/root.tsx
  return <GroupList projectId={projectId} groupedStates={groupedProjectStates} isEditable={isEditable} />;
````
````new web/apps/web/core/components/project-states/root.tsx
  return <GroupList projectId={projectId} groupedStates={groupedStates} isEditable={isEditable} />;
````

- [ ] **Step 3: 一行的控件**

`web/apps/web/core/components/project-states/group-item.tsx`（修改，1 处）：

````old web/apps/web/core/components/project-states/group-item.tsx
        <button
          type="button"
          className={cn(
````
````new web/apps/web/core/components/project-states/group-item.tsx
        <button
          type="button"
          aria-label={t("common.add")}
          className={cn(
````

`web/apps/web/core/components/project-states/options/delete.tsx`（修改，3 处）：

````old web/apps/web/core/components/project-states/options/delete.tsx
import type { State } from "@nerve/api-client";
````
````new web/apps/web/core/components/project-states/options/delete.tsx
import type { State } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/project-states/options/delete.tsx
  const { isMobile } = usePlatformOS();
````
````new web/apps/web/core/components/project-states/options/delete.tsx
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();
````
````old web/apps/web/core/components/project-states/options/delete.tsx
        type="button"
````
````new web/apps/web/core/components/project-states/options/delete.tsx
        type="button"
        aria-label={t("common.actions.delete")}
````

`web/apps/web/core/components/project-states/state-item-title.tsx`（修改，4 处）：

````old web/apps/web/core/components/project-states/state-item-title.tsx
// nerve imports
````
````new web/apps/web/core/components/project-states/state-item-title.tsx
// nerve imports
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/project-states/state-item-title.tsx
  const { stateCount, setUpdateStateModal, disabled, state } = props;
````
````new web/apps/web/core/components/project-states/state-item-title.tsx
  const { stateCount, setUpdateStateModal, disabled, state } = props;
  const { t } = useTranslation();
````
````old web/apps/web/core/components/project-states/state-item-title.tsx
        <div className="hidden items-center gap-2 group-hover:flex">
````
````new web/apps/web/core/components/project-states/state-item-title.tsx
        <div className="flex items-center gap-2 opacity-0 group-focus-within:opacity-100 group-hover:opacity-100">
````
````old web/apps/web/core/components/project-states/state-item-title.tsx
            <button
````
````new web/apps/web/core/components/project-states/state-item-title.tsx
            <button
              type="button"
              aria-label={t("common.actions.edit")}
````

- [ ] **Step 4: 权限的测试、关键词规则和静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/components/project-states/options/delete.test.tsx"
````
````new .oxlintrc.json
        "web/apps/web/core/components/project-states/options/delete.test.tsx",
        "web/apps/web/app/(all)/\\[workspaceSlug\\]/(settings)/settings/projects/\\[projectId\\]/states-labels.test.tsx"
````

`tools/keywords.json`（修改，4 处）：

````old tools/keywords.json
      "why": "项目页面从路由参数读地址（项目、工作区），不从 router store 读：project store 的 currentProjectDetails 和 member store 的 projectMemberIds 读 router store 的 projectId，workspace store 的 currentWorkspace 读它的 workspaceSlug，allowPermissions 不传 slug 和项目时也退回 router store（permissions.store.ts）。router store 由 StoreWrapper 的 effect 在渲染之后写入，同一路由换了项目时晚一次提交，那一次页面按上一个项目显示、按上一个项目判断权限（M3/P10 修复轮，裁定 F-2；P10 规格 §5）。本规则只管 P10 已经改过来的文件：常规设置页和它的控制区、四个功能页、自动化页、成员页、成员列表、成员选择、项目设置侧栏的头部；后面的阶段（P11 的状态和标签页，M4、M6、M7 各自的页面）改它们的读法时，把那些文件加进本规则的 files，收尾删掉 projectMemberIds 这个 getter。测试不算",
````
````new tools/keywords.json
      "why": "项目页面从路由参数读地址（项目、工作区），不从 router store 读：project store 的 currentProjectDetails 和 member store 的 projectMemberIds 读 router store 的 projectId，workspace store 的 currentWorkspace 读它的 workspaceSlug，allowPermissions 不传 slug 和项目时也退回 router store（permissions.store.ts）。router store 由 StoreWrapper 的 effect 在渲染之后写入，同一路由换了项目时晚一次提交，那一次页面按上一个项目显示、按上一个项目判断权限（M3/P10 修复轮，裁定 F-2；P10 规格 §5）。本规则只管 P10、P11 已经改过来的文件：常规设置页和它的控制区、四个功能页、自动化页、成员页、成员列表、成员选择、项目设置侧栏的头部（P10），状态设置页和它的根组件（P11）；后面的阶段（P11 的标签页，M4、M6、M7 各自的页面）改它们的读法时，把那些文件加进本规则的 files，收尾删掉 projectMemberIds 这个 getter。测试不算",
````
````old tools/keywords.json
        "source": "^web/apps/web/(?:app/\\(all\\)/\\[workspaceSlug\\]/\\(settings\\)/settings/projects/\\[projectId\\]/(?:page|members/page|automations/page|features/(?:cycles|intake|modules|views)/page)|core/components/(?:project/(?:settings/control-section|member-list|member-select)|settings/project/sidebar/header))\\.tsx$",
````
````new tools/keywords.json
        "source": "^web/apps/web/(?:app/\\(all\\)/\\[workspaceSlug\\]/\\(settings\\)/settings/projects/\\[projectId\\]/(?:page|members/page|states/page|automations/page|features/(?:cycles|intake|modules|views)/page)|core/components/(?:project/(?:settings/control-section|member-list|member-select)|project-states/root|settings/project/sidebar/header))\\.tsx$",
````
````old tools/keywords.json
            "web/apps/web/core/components/settings/project/sidebar/header.tsx"
````
````new tools/keywords.json
            "web/apps/web/core/components/settings/project/sidebar/header.tsx",
            "web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx",
            "web/apps/web/core/components/project-states/root.tsx"
````
````old tools/keywords.json
          "miss": [
            "web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx",
````
````new tools/keywords.json
          "miss": [
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx`（新文件，75 行）：

````file web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { createRoutesStub } from "react-router";
import { describe, expect, it, vi } from "vitest";
import type { IProjectStore } from "@/store/project/project.store";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import StatesPage from "./states/page";

// Whom a project's state settings show what (M3 design 2 P6, W11, 9.2): its states to those nerve lets list them, but
// its guests, whom the page tells they may not (state.list: its members, whatever their role; design 2 W11); the
// controls that change them to those nerve lets change them (state.create and the others: its admins, and its members
// who are the workspace's admins, its guests among them). The page renders on the server at its route, through React
// Router's stub, which gives it the route's params, with stand-ins for its parts that write what they show: the list
// of groups says whether it has its controls. The caller's permissions and projects are the stores' own, over acme and
// its project web as nerve lists them to him (fake-permissions.ts), while the address the router store holds names no
// project: the page reads its project from its route.

const caller = vi.hoisted((): { permissions?: IUserPermissionStore; projects?: IProjectStore } => ({}));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => caller.permissions }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => caller.projects }));
vi.mock("@/hooks/store/use-project-state", () => ({
  useProjectState: () => ({ getGroupedProjectStates: () => ({}), fetchProjectStates: vi.fn() }),
}));
vi.mock("@/lib/use-session-swr", () => ({ useSessionSWR: () => ({}) }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/components/project-states/group-list", () => ({
  GroupList: ({ isEditable }: { isEditable: boolean }) => `states${isEditable ? " and their controls" : ""}`,
}));
vi.mock("@/components/auth-screens/not-authorized-view", () => ({ NotAuthorizedView: () => "not authorized" }));
vi.mock("@/components/core/page-title", () => ({ PageHead: ({ title }: { title?: string }) => `[${title}]` }));
vi.mock("@/components/settings/content-wrapper", () => ({
  SettingsContentWrapper: ({ children }: { children: unknown }) => children,
}));
vi.mock("@/components/settings/heading", () => ({ SettingsHeading: () => null }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** A route of React Router's stub, whose component gets the route's props as a route module's does. */
type StubRoute = Parameters<typeof createRoutesStub>[0][number];

/**
 * Signs the caller in as nerve lists acme and web to him (callerInWeb), moves the router store's address to acme with
 * no project, and opens the page at its route under web's settings: the text it shows him, and web's name.
 */
async function openedBy({ workspaceRole, projectRole }: Caller, route: string, Component: StubRoute["Component"]) {
  const { permissions, router, projects, acme, web } = await callerInWeb(workspaceRole, projectRole);
  Object.assign(caller, { permissions, projects });
  router.setQuery({ workspaceSlug: acme.slug });
  const Routes = createRoutesStub([{ path: `/:workspaceSlug/settings/projects/:projectId/${route}`, Component }]);
  const markup = renderToStaticMarkup(
    <Routes initialEntries={[`/${acme.slug}/settings/projects/${web.id}/${route}`]} />
  );
  return { text: markup.replace(/<[^>]*>/g, ""), name: web.name };
}

describe("the state settings page", () => {
  it.each([its.admin, its.memberAdmin, its.guestAdmin])("shows $who the states and their controls", async (row) => {
    const { text, name } = await openedBy(row, "states", StatesPage);
    expect(text).toBe(`[${name} - States]states and their controls`);
  });

  it("shows its member the states, and no control", async () => {
    const { text, name } = await openedBy(its.member, "states", StatesPage);
    expect(text).toBe(`[${name} - States]states`);
  });

  it("tells its guest he may not see them", async () => {
    const { text } = await openedBy(its.guest, "states", StatesPage);
    expect(text).toBe("not authorized");
  });
});
````

- [ ] **Step 5: 端到端**

`e2e/fixtures/project-pages.ts`（修改，5 处）：

````old e2e/fixtures/project-pages.ts
import { answerTo } from "./settings-pages";
````
````new e2e/fixtures/project-pages.ts
import { answerTo, transitionsEnded } from "./settings-pages";
````
````old e2e/fixtures/project-pages.ts
 * Presses Tab on page until the focus is back where it was, at most 40 times, and gives the stops in order: each the
 * focused element's label, a field's id or a button's text, marked "outside: " when within does not hold it.
````
````new e2e/fixtures/project-pages.ts
 * Presses Tab on page until the focus is back where it was, at most presses times, and gives the stops in order: each
 * the focused element's label, a field's id or a button's text, marked "outside: " when within does not hold it.
````
````old e2e/fixtures/project-pages.ts
export async function tabCycle(page: Page, within: Locator): Promise<string[]> {
````
````new e2e/fixtures/project-pages.ts
export async function tabCycle(page: Page, within: Locator, presses = 40): Promise<string[]> {
````
````old e2e/fixtures/project-pages.ts
    return back || stops.length === 39 ? [...stops, stop] : from([...stops, stop]);
````
````new e2e/fixtures/project-pages.ts
    return back || stops.length === presses - 1 ? [...stops, stop] : from([...stops, stop]);
````
````old e2e/fixtures/project-pages.ts
  return from([]);
}

````
````new e2e/fixtures/project-pages.ts
  return from([]);
}

/**
 * Drags item onto onto by its grip, which may show only while the pointer is over item, as a hand does: once page's
 * transitions have ended (one still under way, as when a toast comes in, can scroll the document, and a scroll between
 * the press and the move starts no drag), it presses on the grip, moves within item, which starts the drag, then over
 * onto, at its point at (its middle when left out), and lets go. Playwright's dragTo moves at once, which pragmatic-dnd
 * does not take for a drag.
 */
export async function dragOnto(
  page: Page,
  item: Locator,
  grip: Locator,
  onto: Locator,
  at?: { x: number; y: number }
): Promise<void> {
  await transitionsEnded(page.locator("body"));
  await item.hover();
  await grip.hover();
  await page.mouse.down();
  await item.hover({ position: { x: 60, y: 8 } });
  await onto.hover(at && { position: at });
  await page.mouse.up();
}

````

`e2e/stories/project/p6-states-page.spec.ts`（新文件，262 行）：

````file e2e/stories/project/p6-states-page.spec.ts
import type { Locator, Page } from "@playwright/test";

import {
  addProjectMembers,
  changeRole,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
} from "../../fixtures/api";
import { expectStates, statesOfANewProject, type StateRow } from "../../fixtures/assert/project";
import { accountId, bearer, emailFor } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { dragOnto, tabCycle } from "../../fixtures/project-pages";
import {
  closedByEscape,
  closedWithin,
  enabledWithin,
  registerOnboarded,
  sentHeld,
  sentTo,
} from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser } from "../../fixtures/workspace-pages";

// P6's page version (M3 design 2, 3.17, 7.6): a project's states on its settings page, as its admin changes them, and
// as its member and a workspace admin who is its member see them. The API version is p6-states.spec.ts.

/** The row of the state of name on page's state settings. */
function stateRow(page: Page, name: string): Locator {
  return page.locator("div.group", { has: page.getByRole("heading", { name, exact: true, level: 6 }) });
}

/** The heading of the group of key on page's state settings, which opens and closes its list. */
function groupHeading(page: Page, key: string): Locator {
  return page.getByRole("button", { name: key, exact: true });
}

/** The button beside the heading of the group of key that adds a state to it. */
function addTo(page: Page, key: string): Locator {
  return groupHeading(page, key).locator("xpath=..").getByRole("button", { name: "Add" });
}

/** The form page shows to create or change a state: one at a time. */
function stateForm(page: Page): Locator {
  return page.locator("div.rounded-sm.p-3", { has: page.locator("#name") });
}

/** The ids of the states of the project of projectId, by name, as the caller of token lists them. */
async function stateIds(api: Api, token: string, projectId: string): Promise<Record<string, string>> {
  const { data } = await api.GET("/api/v0/projects/{project_id}/states", {
    params: { path: { project_id: projectId } },
    headers: bearer(token),
  });
  return Object.fromEntries((data?.data ?? []).map((s) => [s.name, s.id]));
}

/** rows with the state of name changed as to says. */
function changed(rows: StateRow[], name: string, to: Partial<StateRow>): StateRow[] {
  return rows.map((s) => (s.name === name ? { ...s, ...to } : s));
}

test("P6 (page): the admin creates Review in the started group, its form gone through by Tab and held until nerve answers; a name taken is said under the name; he recolours Review, drags it before In Progress and makes it the default, Backlog the default no longer; the default and a group's only state offer no deletion; a deletion nerve refuses leaves its dialog open, one under way cannot be dismissed, and In Progress is deleted; its member sees the states and can change none, a workspace admin who is its member can", async ({
  api,
  baseURL,
  browser,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const memEmail = emailFor(testInfo, "mem");
  const wandaEmail = emailFor(testInfo, "wanda");
  const [admin, mem, wanda] = await Promise.all([
    registerOnboarded(api, adminEmail),
    registerOnboarded(api, memEmail),
    registerOnboarded(api, wandaEmail),
  ]);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  // mem and wanda members of acme and of Web; wanda then acme's admin, Web's member still
  await inviteAndAccept(api, admin.access_token, slug, { email: memEmail, token: mem.access_token }, 15);
  await inviteAndAccept(api, admin.access_token, slug, { email: wandaEmail, token: wanda.access_token }, 15);
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  const wandaId = await accountId(api, wanda.access_token);
  await addProjectMembers(api, admin.access_token, web.id, [
    { member_id: await accountId(api, mem.access_token), role: 15 },
    { member_id: wandaId, role: 15 },
  ]);
  await changeRole(api, admin.access_token, slug, wandaId, 20);
  const listed = `/api/v0/projects/${web.id}/states`;
  const inProgressId = (await stateIds(api, admin.access_token, web.id))["In Progress"];
  if (!inProgressId) throw new Error("Web was made without In Progress");
  let rows = statesOfANewProject(adminEmail);

  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/settings/projects/${web.id}/states`);
  await expect(stateRow(page, "In Progress")).toBeVisible();
  await Promise.all(
    ["backlog", "unstarted", "started", "completed", "cancelled"].map((key) =>
      expect(groupHeading(page, key), key).toHaveAttribute("aria-expanded", "true")
    )
  );

  // Review, in the started group: Tab goes through the form in its order; the form waits for nerve, its buttons
  // disabled, Cancel too; it closes once nerve has made the state, last of its project's.
  await addTo(page, "started").click();
  await page.locator("#name").fill("Review");
  await stateForm(page).getByRole("button").first().focus();
  expect(await tabCycle(page, stateForm(page), 4)).toEqual(["name", "description", "Create", "Cancel"]);
  const creation = await sentHeld(page, "POST", listed, () =>
    stateForm(page).getByRole("button", { name: "Create" }).click()
  );
  expect(creation.body).toEqual({ name: "Review", description: "", color: "#f59e0b", group: "started" });
  expect(await enabledWithin(stateForm(page).getByRole("button", { name: "Creating" }))).toBe(false);
  expect(await enabledWithin(stateForm(page).getByRole("button", { name: "Cancel" }))).toBe(false);
  expect((await creation.release()).status()).toBe(201);
  const reviewId = (await stateIds(api, admin.access_token, web.id)).Review;
  if (!reviewId) throw new Error("nerve made no Review");
  await expect(stateRow(page, "Review")).toBeVisible();
  await expect(stateForm(page)).toHaveCount(0);
  rows = [
    ...rows,
    {
      name: "Review",
      color: "#f59e0b",
      group: "started",
      sequence: 70000,
      default: false,
      deleted: false,
      by: adminEmail,
    },
  ];
  await expectStates(db, web.id, rows);

  // Todo again, in the unstarted group: said under the name, the form open with what it holds.
  await addTo(page, "unstarted").click();
  await page.locator("#name").fill("Todo");
  const taken = await sentTo(page, "POST", listed, () =>
    stateForm(page).getByRole("button", { name: "Create" }).click()
  );
  expect(taken.answer.status()).toBe(409);
  await expect(stateForm(page).getByText("A state of this project already has this name.")).toBeVisible();
  await expect(page.locator("#name")).toHaveValue("Todo");
  await stateForm(page).getByRole("button", { name: "Cancel" }).click();

  // Review recoloured, its row's Edit reached by Tab, its row's controls shown while the focus is in it as while the
  // pointer is over it (unseen while neither is): the fields its form edits.
  const review = stateRow(page, "Review");
  const reviewEdit = review.getByRole("button", { name: "Edit" });
  await page.mouse.move(0, 0);
  await expect(reviewEdit.locator("xpath=../..")).toHaveCSS("opacity", "0");
  await review.getByRole("button", { name: "Mark as default" }).focus();
  await page.keyboard.press("Tab");
  await expect(reviewEdit).toBeFocused();
  await expect(reviewEdit.locator("xpath=../..")).toHaveCSS("opacity", "1");
  await page.keyboard.press("Enter");
  await stateForm(page).getByRole("button").first().click();
  await page.locator('[title="#0693E3"]').click();
  const recoloured = await sentTo(page, "PATCH", `/api/v0/states/${reviewId}`, () =>
    stateForm(page).getByRole("button", { name: "Update" }).click()
  );
  expect([recoloured.answer.status(), recoloured.body]).toEqual([
    200,
    { name: "Review", color: "#0693e3", description: "" },
  ]);
  rows = changed(rows, "Review", { color: "#0693e3" });

  // Review dragged onto In Progress's top: before it, the first of the started group, a step before In Progress.
  const moved = await sentTo(page, "PATCH", `/api/v0/states/${reviewId}`, () =>
    dragOnto(page, review, review, stateRow(page, "In Progress"), { x: 60, y: 4 })
  );
  expect([moved.answer.status(), moved.body]).toEqual([200, { group: "started", sequence: 20000 }]);
  await expect(page.locator("div.group h6")).toHaveText([
    "Backlog",
    "Todo",
    "Review",
    "In Progress",
    "Done",
    "Cancelled",
  ]);
  rows = changed(rows, "Review", { sequence: 20000 });

  // Review the default, Backlog no longer; neither offers its deletion, nor Todo, its group's only state.
  const markReview = () =>
    sentTo(page, "POST", `/api/v0/states/${reviewId}/mark-default`, () =>
      review.getByRole("button", { name: "Mark as default" }).click()
    );
  expect((await markReview()).answer.status()).toBe(204);
  await expect(review.getByRole("button", { name: "Default" })).toBeDisabled();
  await expect(stateRow(page, "Backlog").getByRole("button", { name: "Mark as default" })).toBeEnabled();
  await Promise.all(
    ["Review", "Backlog", "Todo"].map((name) =>
      expect(stateRow(page, name).getByRole("button", { name: "Delete" }), name).toBeDisabled()
    )
  );
  rows = changed(changed(rows, "Backlog", { default: false }), "Review", { default: true });
  await expectStates(db, web.id, rows);

  // In Progress made the default meanwhile, elsewhere: nerve refuses its deletion, said in a toast, the dialog open.
  const elsewhere = await api.POST("/api/v0/states/{state_id}/mark-default", {
    params: { path: { state_id: inProgressId } },
    headers: bearer(admin.access_token),
  });
  expect(elsewhere.response.status).toBe(204);
  const inProgress = stateRow(page, "In Progress");
  const deleted = `/api/v0/states/${inProgressId}`;
  const dialog = page.getByRole("dialog");
  await inProgress.getByRole("button", { name: "Delete" }).click();
  const refused = await sentTo(page, "DELETE", deleted, () => dialog.getByRole("button", { name: "Delete" }).click());
  expect(refused.answer.status()).toBe(409);
  await expect(
    page.getByText("The default state cannot be deleted. Make another state the default first.")
  ).toBeVisible();
  expect(await closedWithin(page)).toBe(false);
  await dialog.getByRole("button", { name: "Cancel" }).click();
  expect(await closedWithin(page)).toBe(true);

  // Reloaded, the page shows In Progress the default, as nerve holds it; Review made the default again. In Progress
  // deleted, its dialog held while nerve answers: Escape does not dismiss it, Cancel is disabled; it closes once nerve
  // has, Review left in the started group.
  await page.reload();
  await expect(inProgress.getByRole("button", { name: "Default" })).toBeDisabled();
  expect((await markReview()).answer.status()).toBe(204);
  await inProgress.getByRole("button", { name: "Delete" }).click();
  const deletion = await sentHeld(page, "DELETE", deleted, () =>
    dialog.getByRole("button", { name: "Delete" }).click()
  );
  expect(await closedByEscape(page)).toBe(false);
  expect(await enabledWithin(dialog.getByRole("button", { name: "Cancel" }))).toBe(false);
  expect((await deletion.release()).status()).toBe(204);
  expect(await closedWithin(page)).toBe(true);
  await expect(inProgress).toHaveCount(0);
  rows = changed(rows, "In Progress", { deleted: true });
  await expectStates(db, web.id, rows);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`409 POST ${listed}`, `409 DELETE ${deleted}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING],
    errors: Array(2).fill("Failed to load resource: the server responded with a status of 409 (Conflict)"),
  });

  // Its member sees the states and no control that changes them; a workspace admin who is its member has them all.
  const seenBy = async (who: string, tokens: typeof mem, controls: boolean) => {
    const other = await anotherBrowser(browser, baseURL ?? "", tokens);
    const otherWatch = await watchPage(other.page);
    await other.page.goto(`/${slug}/settings/projects/${web.id}/states`);
    await expect(stateRow(other.page, "Review"), who).toBeVisible();
    await expect(addTo(other.page, "started"), who).toBeEnabled({ enabled: controls });
    await expect(other.page.getByRole("button", { name: "Mark as default" }), who).toHaveCount(controls ? 4 : 0);
    await expect(other.page.getByRole("button", { name: "Edit" }), who).toHaveCount(controls ? 5 : 0);
    expect([otherWatch.apiFailures, otherWatch.pageErrors], who).toEqual([[], []]);
    await expectQuietConsole(other.page, otherWatch, { warnings: [EMOJI_CHECK_WARNING] });
    await other.close();
  };
  await seenBy("its member", mem, false);
  await seenBy("the workspace admin who is its member", wanda, true);
});
````

`e2e/stories/project/p6-states.spec.ts`（修改，1 处）：

````old e2e/stories/project/p6-states.spec.ts
// group keep. The page version comes with the states' settings page (P11).
````
````new e2e/stories/project/p6-states.spec.ts
// group keep. The page version is p6-states-page.spec.ts.
````

`e2e/stories/project/p8-project-preferences.spec.ts`（修改，5 处）：

````old e2e/stories/project/p8-project-preferences.spec.ts
import {
  answerTo,
  holdAnswer,
  registerOnboarded,
  sentTo,
  shownWithin,
  transitionsEnded,
} from "../../fixtures/settings-pages";
````
````new e2e/stories/project/p8-project-preferences.spec.ts
import { dragOnto } from "../../fixtures/project-pages";
import { answerTo, holdAnswer, registerOnboarded, sentTo, shownWithin } from "../../fixtures/settings-pages";
````
````old e2e/stories/project/p8-project-preferences.spec.ts
 * Drags the project of id in page's sidebar onto the one of onto by its handle, which shows while the pointer is over
 * the project: the drag starts there, as a hand's would, once page's transitions have ended. One still under way, as
 * when a toast comes in, can scroll the document, and a scroll between the press and the move starts no drag. Resolves
 * with the body page sends and nerve's answer.
````
````new e2e/stories/project/p8-project-preferences.spec.ts
 * Drags the project of id in page's sidebar onto the one of onto by its handle, its first button, which shows while
 * the pointer is over the project (dragOnto). Resolves with the body page sends and nerve's answer.
````
````old e2e/stories/project/p8-project-preferences.spec.ts
async function dragOnto(page: Page, id: string, onto: string): Promise<{ body: unknown; answer: Response }> {
  await transitionsEnded(page.locator("body"));
  await inSidebar(page, id).hover();
  await inSidebar(page, id).locator("button").first().hover();
  await page.mouse.down();
  await inSidebar(page, id).hover({ position: { x: 60, y: 8 } });
  return sentTo(page, "PATCH", `/api/v0/me/projects/${id}/preferences`, async () => {
    await inSidebar(page, onto).hover();
    await page.mouse.up();
  });
````
````new e2e/stories/project/p8-project-preferences.spec.ts
function dragInSidebar(page: Page, id: string, onto: string): Promise<{ body: unknown; answer: Response }> {
  const project = inSidebar(page, id);
  return sentTo(page, "PATCH", `/api/v0/me/projects/${id}/preferences`, () =>
    dragOnto(page, project, project.locator("button").first(), inSidebar(page, onto))
  );
````
````old e2e/stories/project/p8-project-preferences.spec.ts
  const moved = await dragOnto(page, ops.id, docs.id);
````
````new e2e/stories/project/p8-project-preferences.spec.ts
  const moved = await dragInSidebar(page, ops.id, docs.id);
````
````old e2e/stories/project/p8-project-preferences.spec.ts
  const gusMoved = await dragOnto(theGuest.page, web.id, docs.id);
````
````new e2e/stories/project/p8-project-preferences.spec.ts
  const gusMoved = await dragInSidebar(theGuest.page, web.id, docs.id);
````

`e2e/stories/smoke/s2-web-app.spec.ts`（修改，5 处）：

````old e2e/stories/smoke/s2-web-app.spec.ts
 * fetch no other check holds; the workspace's general settings and its members; then the project's settings.
````
````new e2e/stories/smoke/s2-web-app.spec.ts
 * fetch no other check holds; the workspace's general settings and its members; then the project's settings, its
 * general page and its states.
````
````old e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings/members", [...APP, ...WORKSPACE, ...INVITATIONS]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL]],
````
````new e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings/members", [...APP, ...WORKSPACE, ...INVITATIONS]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL]],
    ["/{slug}/settings/projects/{project}/states", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER]],
````
````old e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings", [...APP, ...WORKSPACE, ...GENERAL]],
    ["/{slug}/settings/members", [...APP, ...WORKSPACE]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL]],
  ],
````
````new e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings", [...APP, ...WORKSPACE, ...GENERAL]],
    ["/{slug}/settings/members", [...APP, ...WORKSPACE]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL]],
    ["/{slug}/settings/projects/{project}/states", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER]],
  ],
````
````old e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings", [...APP, ...WORKSPACE]],
    ["/{slug}/settings/members", [...APP, ...WORKSPACE]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL]],
  ],
````
````new e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings", [...APP, ...WORKSPACE]],
    ["/{slug}/settings/members", [...APP, ...WORKSPACE]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL]],
    ["/{slug}/settings/projects/{project}/states", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER]],
  ],
````
````old e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT]],
````
````new e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT]],
    ["/{slug}/settings/projects/{project}/states", [...APP, ...WORKSPACE, ...PROJECT]],
````

`e2e/stories/workspace/w11-guest-bounds.spec.ts`（修改，3 处）：

````old e2e/stories/workspace/w11-guest-bounds.spec.ts
import { bearer, emailFor, newAccount } from "../../fixtures/auth";
````
````new e2e/stories/workspace/w11-guest-bounds.spec.ts
import { accountId, bearer, emailFor, newAccount } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { registerOnboarded } from "../../fixtures/settings-pages";
````
````old e2e/stories/workspace/w11-guest-bounds.spec.ts
// test holds every cell; this story samples four of them through the API.
// The page version is P11's.
````
````new e2e/stories/workspace/w11-guest-bounds.spec.ts
// test holds every cell; this story samples four of them through the API, and
// its page version what a guest's pages show him.
````
````old e2e/stories/workspace/w11-guest-bounds.spec.ts
  expect(await tables(), "the tables after gus's calls").toEqual(before);
});

````
````new e2e/stories/workspace/w11-guest-bounds.spec.ts
  expect(await tables(), "the tables after gus's calls").toEqual(before);
});

test("W11 (page): a guest signs in: his sidebar lists Web, the project he is a guest of, not Secret; the workspace's members page and Web's state settings tell him he may not see them; no page he opens shows an address, his own neither", async ({
  api,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const gusEmail = emailFor(testInfo, "gus");
  const gus = await registerOnboarded(api, gusEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  await inviteAndAccept(api, admin.access_token, slug, { email: gusEmail, token: gus.access_token }, 5);
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.access_token, web.id, [
    { member_id: await accountId(api, gus.access_token), role: 5 },
  ]);
  await createProject(api, admin.access_token, slug, { name: "Secret", identifier: "SECRET", network: 0 });

  const page = await signedInPage(gus);
  const watch = await watchPage(page);
  /** Opens path, and checks that once it shows what is expected no member's address is in the page. */
  const opened = async (path: string, shown: string) => {
    await page.goto(path);
    await expect(page.getByText(shown, { exact: true }).first(), path).toBeVisible();
    const html = await page.content();
    expect([html.includes(adminEmail), html.includes(gusEmail)], `the addresses on ${path}`).toEqual([false, false]);
  };
  await opened(`/${slug}`, "Web");
  const joined = page.getByRole("complementary", { name: "Main sidebar" }).locator('[id^="sidebar-"][id$="-JOINED"]');
  await expect(joined).toHaveText([/Web/]);
  const denied = "Oops! You are not authorized to view this page";
  await opened(`/${slug}/settings/members`, denied);
  await opened(`/${slug}/settings/projects/${web.id}/states`, denied);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: Array(3).fill(EMOJI_CHECK_WARNING) });
});

````

- [ ] **Step 6: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 69 条规则、3 个例外，没有命中；web 的 oxlint 346 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 108 个全部通过。

- [ ] **Step 7: 提交**

```bash
git add .oxlintrc.json e2e/fixtures/project-pages.ts e2e/stories/project/p6-states-page.spec.ts e2e/stories/project/p6-states.spec.ts e2e/stories/project/p8-project-preferences.spec.ts e2e/stories/smoke/s2-web-app.spec.ts e2e/stories/workspace/w11-guest-bounds.spec.ts tools/keywords.json 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states/page.tsx' web/apps/web/core/components/project-states/group-item.tsx web/apps/web/core/components/project-states/options/delete.tsx web/apps/web/core/components/project-states/root.tsx web/apps/web/core/components/project-states/state-item-title.tsx web/apps/web/core/store/state.store.test.ts web/apps/web/core/store/state.store.ts
```
```bash
git commit -m "feat(M3/P11): the state settings page reads its project from its route, offers the changes to whom nerve allows them, and its controls are reached by Tab

The page and its root read the workspace and the project from the route's
params, allowPermissions naming them, and the root reads the project's
states by its id; the keyword rule project-address-route takes both
files. A row's controls show while the focus is in the row, and the edit,
delete and add buttons have names. P6's and W11's page versions, S2's
mount list for the page.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p11.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T3.1` | 状态设置页的门读 router store 的地址 | 关键词守卫、`states-labels.test.tsx` | 静态；vitest |
| `T3.2` | 项目的访客看得到状态 | `states-labels.test.tsx`、故事 W11 | vitest；端到端 |
| `T3.3` | 项目的成员得到改状态的控件 | `states-labels.test.tsx`、故事 P6 | vitest；端到端 |
| `T3.4` | 一行的控件在指针经过之前 Tab 到不了（Plane 的写法） | 故事 P6 | 端到端 |
| `T3.5` | 一行的控件在 Tab 进到这一行时看不见 | 故事 P6 | 端到端 |
| `T3.6` | 一行的编辑按钮没有名字 | 故事 P6 | 端到端 |
| `T3.7` | 状态的表单的按钮有正的 `tabIndex` | 故事 P6 | 端到端 |
| `T3.8` | 项目的状态按组分时少一个组 | `state.store.test.ts` | vitest |
| `T3.9` | 在 `states-labels.test.tsx` 里写一个非空断言 | oxlint（`check:lint`） | 静态 |

---

### Task 4: 标签的修改在会话里跟进；名字已被占用在名字下；第三层说标签只有两层

**Files:**
- Create: `web/apps/web/core/components/labels/use-label-changes.test.ts`、`web/apps/web/core/components/labels/use-label-changes.ts`
- Modify: `.oxlintrc.json`、`web/apps/web/core/components/labels/create-update-label-inline.test.tsx`、`web/apps/web/core/components/labels/create-update-label-inline.tsx`、`web/apps/web/core/components/labels/delete-label-modal.tsx`、`web/apps/web/core/components/labels/project-setting-label-group.tsx`、`web/apps/web/core/components/labels/project-setting-label-item.tsx`、`web/apps/web/core/components/labels/project-setting-label-list.tsx`、`web/packages/i18n/src/locales/en/project-settings.json`、`web/packages/i18n/src/locales/zh-CN/project-settings.json`

**Interfaces:**
- Produces（spec 2.4；M3 设计 2 的 P7，3.16，7.1、7.6；总体设计 7.7；P7b spec 第 5 节的 P11 一行；P10 review 第 6 节的 T11-a）：
  - `useLabelChanges()`（新文件）：`create(projectId, data, underField, done)`、`update(labelId, data, underField, done)`、`move(labelId, parentId, droppedOnId, dropAtEndOfList)`、`removeFromGroup(labelId)`、`remove(labelId, done)`，各经 store 的方法和 `followInSession`；创建、修改的拒绝经 `useFormRefusal`（`project.label_name_taken` 在名字下，不论大小写；名字的字段错误在名字下，其余提示）；移动、移出组被拒绝时 nerve 指名 `parent_id`（第三层、别的项目的标签）说标签只有两层（`project_settings.labels.parent_not_allowed`，两种语言），其余经 `useRefusalToast`；删除的拒绝经 `useRefusalToast`。每个都在页面跟进之后才兑现。移动的请求体由 store 在轮到它时从 nerve 最近的回答算出（P7b：放进没有子标签的组只发 `parent_id`），页面不改。
  - 标签的行内表单（`create-update-label-inline.tsx`）：经 `create`、`update`，`handleSubmit` 返回的 promise 兑现之前表单忙（按钮显示"Adding"/"Updating"、按不了，取消也按不了），nerve 指名的名字的错误经 `setError` 在名字下（原来名字已被占用在提示里），被拒绝时表单留着，成功的跟进里关上；请求体是 `LabelCreate`/`LabelUpdate` 中表单编辑的名字和颜色。`TLabelOperationsCallbacks` 删除：表单收 `projectId`，列表、组、一行不再收、不再传回调。
  - 删除的确认框（`delete-label-modal.tsx`）：经 `remove`，在跟进做完之前忙、关不掉（Task 1 的 `AlertModalCore`），只在成功的跟进中关上；删除连同子标签由 store 写入（P7b）。
  - 一行和组的"移出组"经 `removeFromGroup`；列表的拖放经 `move`。
  - 静态检查：`.oxlintrc.json` 的非空断言的范围加上 `use-label-changes.ts` 和它的测试。

**Tests:**
- vitest：`use-label-changes.test.ts`（新文件）：`sends $change to the store as the page asks`（创建、修改、移动、移出组、删除）；`follows $change nerve makes with done, and says nothing`；`shows $refused under the name, of $by.change, and does not follow it`（另一种大小写的名字已被占用）；`shows a name too long under the name`；`shows nerve's reason for a colour it does not allow, which the form shows nothing under, in a toast`；`says a place nerve does not allow for $change: labels have two levels`（移动、移出组）；`shows nerve's reason when it refuses $by.change otherwise, in a toast`；`does nothing on the page when it $settles after another tab moved this one`（五个修改 × `lateSettlingsAnswering` 的两种）。`create-update-label-inline.test.tsx`（整个文件）：`creates a label in the page's project, at the top, with the name and colour it holds`；`changes a label with the fields its form edits, the only ones nerve's LabelUpdate takes`；`closes once nerve has made the change, not before`；`stays open when nerve refuses a $change, its name taken said under the name, no toast`（原来的"在提示里说名字已被占用"一条删除）。
- 端到端：表单和确认框扣住请求的核对、名字已被占用在名字下、第三层的提示在 Task 5 的 P7 页面版本（标签设置页从那里起按路由读项目）；本 Task 的 `make e2e` 照旧通过（P7 的 API 版本，S2）。

- [ ] **Step 1: 标签的修改**

`web/apps/web/core/components/labels/use-label-changes.test.ts`（新文件，160 行）：

````file web/apps/web/core/components/labels/use-label-changes.test.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Label } from "@nerve/api-client";
import { heldChange, lateSettlingsAnswering, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal, underFields } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { labelOf, projectOf } from "@/store/project/fake-projects";
import { useLabelChanges } from "./use-label-changes";

// How the label settings change a project's labels (M3 design 2 P7, 3.16, 7.1, 7.6): the hook runs as a plain
// function, with stand-ins for the label store's changes, which nerve answers when the test says, and for the form's
// setError. Its session is fake-tab.ts's.

const store = vi.hoisted(() => ({
  createLabel: vi.fn(),
  updateLabel: vi.fn(),
  updateLabelPosition: vi.fn(),
  deleteLabel: vi.fn(),
}));
vi.mock("@/hooks/store/use-label", () => ({ useLabel: () => store }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const web = projectOf("WEB", "w-acme");
const bug = labelOf(web, "Bug", 10000);
const ui = labelOf(web, "UI", 20000);
const setError = vi.fn();
const done = vi.fn();
const underField = (field: string, message: string) => setError(field, { message });

/** Each change of the page, sent as the page sends it, with the store's change it makes and what nerve answers. */
const changes = {
  creation: {
    change: "a creation",
    send: () => useLabelChanges().create(web.id, { name: "Bug", color: "#F59E0B" }, underField, done),
    sent: () => store.createLabel,
    call: [web.id, { name: "Bug", color: "#F59E0B" }],
    answer: bug,
  },
  change: {
    change: "a change",
    send: () => useLabelChanges().update(bug.id, { name: "Defect", color: "#F59E0B" }, underField, done),
    sent: () => store.updateLabel,
    call: [bug.id, { name: "Defect", color: "#F59E0B" }],
    answer: bug,
  },
  move: {
    change: "a move",
    send: () => useLabelChanges().move(ui.id, bug.id, undefined, false),
    sent: () => store.updateLabelPosition,
    call: [ui.id, bug.id, undefined, false],
    answer: ui,
  },
  removal: {
    change: "a removal from its group",
    send: () => useLabelChanges().removeFromGroup(ui.id),
    sent: () => store.updateLabel,
    call: [ui.id, { parent_id: null }],
    answer: ui,
  },
  deletion: {
    change: "a deletion",
    send: () => useLabelChanges().remove(bug.id, done),
    sent: () => store.deleteLabel,
    call: [bug.id],
    answer: undefined,
  },
};
const { creation, change, move, removal, deletion } = changes;

beforeEach(() => {
  signedIn();
  for (const sent of Object.values(store)) {
    sent.mockReset();
    sent.mockResolvedValue(bug);
  }
  setError.mockReset();
  done.mockReset();
  toasts.length = 0;
});

describe("useLabelChanges", () => {
  it.each(Object.values(changes))("sends $change to the store as the page asks", async ({ send, sent, call }) => {
    await send();
    expect(sent().mock.calls).toEqual([call]);
  });

  it.each([creation, change, deletion])("follows $change nerve makes with done, and says nothing", async ({ send }) => {
    await send();
    expect([done.mock.calls.length, underFields(setError), toasts]).toEqual([1, [], []]);
  });

  it.each([
    { by: creation, refused: "a name another label has, in another case", code: "project.label_name_taken" },
    { by: change, refused: "a name another label has, in another case", code: "project.label_name_taken" },
  ])("shows $refused under the name, of $by.change, and does not follow it", async ({ by, code }) => {
    by.sent().mockRejectedValueOnce(refusal(409, code));
    await by.send();
    expect([underFields(setError), toasts, done.mock.calls]).toEqual([
      [["name", "errors.project_label_name_taken"]],
      [],
      [],
    ]);
  });

  it("shows a name too long under the name", async () => {
    store.createLabel.mockRejectedValueOnce(refusal(422, "validation_failed", [{ field: "name", code: "too_long" }]));
    await creation.send();
    expect([underFields(setError), toasts]).toEqual([[["name", "errors.field.too_long"]], []]);
  });

  it("shows nerve's reason for a colour it does not allow, which the form shows nothing under, in a toast", async () => {
    store.updateLabel.mockRejectedValueOnce(refusal(422, "validation_failed", [{ field: "color", code: "too_long" }]));
    await change.send();
    expect([underFields(setError), toasts]).toEqual([
      [],
      [{ type: "error", title: "toast.error", message: "errors.validation_failed" }],
    ]);
  });

  it.each([move, removal])("says a place nerve does not allow for $change: labels have two levels", async (row) => {
    row.sent().mockRejectedValueOnce(refusal(422, "validation_failed", [{ field: "parent_id", code: "not_allowed" }]));
    await row.send();
    expect(toasts).toEqual([
      { type: "error", title: "toast.error", message: "project_settings.labels.parent_not_allowed" },
    ]);
  });

  it.each([
    { by: move, status: 404, code: "project.label_not_found" },
    { by: removal, status: 403, code: "forbidden" },
    { by: deletion, status: 404, code: "project.label_not_found" },
  ])("shows nerve's reason when it refuses $by.change otherwise, in a toast", async ({ by, status, code }) => {
    by.sent().mockRejectedValueOnce(refusal(status, code));
    await by.send();
    const message = `errors.${code.replace(".", "_")}`;
    expect([toasts, done.mock.calls]).toEqual([[{ type: "error", title: "toast.error", message }], []]);
  });

  describe.each(Object.values(changes))("$change", ({ send, sent, answer }) => {
    it.each(lateSettlingsAnswering(answer))(
      "does nothing on the page when it $settles after another tab moved this one",
      async ({ settle }) => {
        const held = heldChange<Label | undefined>();
        sent().mockReturnValueOnce(held.sent);
        const followed = send();
        switchAccount();
        settle(held);
        await followed;
        await pageSettled();
        expect([done.mock.calls, underFields(setError), toasts]).toEqual([[], [], []]);
      }
    );
  });
});
````

`web/apps/web/core/components/labels/use-label-changes.ts`（新文件，70 行）：

````file web/apps/web/core/components/labels/use-label-changes.ts
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { LabelCreate, LabelUpdate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useLabel } from "@/hooks/store/use-label";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { fieldErrorKeys } from "@/lib/error-messages";
import { useFormRefusal } from "@/lib/form-refusal";
import { followInSession } from "@/lib/in-session";

/** The field of a label's form under which nerve's refusals show. */
type LabelFormField = "name";
const FIELDS: readonly LabelFormField[] = ["name"];
/** nerve's code for a name another label of the project has, in any case, under the name (M3 design 3.16). */
const TAKEN: Readonly<Record<string, LabelFormField>> = { "project.label_name_taken": "name" };

/** Where a label's form says a message of nerve's refusal: under the field (its setError). */
export type LabelUnderField = (field: LabelFormField, message: string) => void;

/**
 * The changes of a project's labels that its settings page makes (M3 design 2 P7, 7.6), each through the label store,
 * one at a time (v0 design 7.7). The page follows each only in the session it was sent in (M3 design 7.1): done, the
 * caller's follow-up of success, once nerve has made the change; nerve's refusal of a creation or a change under the
 * form's name it names (a name taken, in any case), else in a toast, as every refusal of a move or a deletion is. A
 * move to a place nerve does not allow (its parent_id: labels have two levels, a label goes under one of its project
 * at the top; 3.16), which the page's drop rules keep from but another tab's change can make, is said as such. Each
 * settles once the page has followed the change.
 */
export function useLabelChanges() {
  const { createLabel, updateLabel, updateLabelPosition, deleteLabel } = useLabel();
  const { t } = useTranslation();
  const failed = useRefusalToast();
  const showRefusal = useFormRefusal(FIELDS, TAKEN);
  const placeRefused = (error: unknown) => {
    if (fieldErrorKeys(error).parent_id === undefined) {
      failed(error);
      return;
    }
    setToast({
      type: TOAST_TYPE.ERROR,
      title: t("toast.error"),
      message: t("project_settings.labels.parent_not_allowed"),
    });
  };

  return {
    /** Creates a label at the top of the project of projectId, last of its labels. */
    create: (projectId: string, data: LabelCreate, underField: LabelUnderField, done: () => void) =>
      followInSession(() => createLabel(projectId, data), { done, failed: (error) => showRefusal(error, underField) }),
    /** Changes the label of labelId to what data names. */
    update: (labelId: string, data: LabelUpdate, underField: LabelUnderField, done: () => void) =>
      followInSession(() => updateLabel(labelId, data), { done, failed: (error) => showRefusal(error, underField) }),
    /** Moves the label of labelId where it was dropped (LabelStore.updateLabelPosition). */
    move: (labelId: string, parentId: string | null, droppedOnId: string | undefined, dropAtEndOfList: boolean) =>
      followInSession(() => updateLabelPosition(labelId, parentId, droppedOnId, dropAtEndOfList), {
        failed: placeRefused,
      }),
    /** Moves the label of labelId to the top of its project's labels. */
    removeFromGroup: (labelId: string) =>
      followInSession(() => updateLabel(labelId, { parent_id: null }), { failed: placeRefused }),
    /** Deletes the label of labelId, and the labels under it. */
    remove: (labelId: string, done: () => void) => followInSession(() => deleteLabel(labelId), { done, failed }),
  };
}
````

- [ ] **Step 2: 行内表单和删除的确认框**

`web/apps/web/core/components/labels/create-update-label-inline.test.tsx`（整个文件，100 行）：

````whole web/apps/web/core/components/labels/create-update-label-inline.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Label } from "@nerve/api-client";
import { heldChange, pageSettled, signedIn } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { labelOf, projectOf } from "@/store/project/fake-projects";
import { CreateUpdateLabelInline } from "./create-update-label-inline";

// What the label settings' inline form sends to create or change a label, and how it follows nerve's answer (M3
// design 3.16, 7.6; v0 design 7.7). The form renders on the server with stand-ins for the name's input and the buttons,
// which keep the props they were given: the test types the name as the input would, and clicks the form's button; the
// label store's changes are stand-ins nerve answers when the test says. Its session is fake-tab.ts's.

const store = vi.hoisted(() => ({ createLabel: vi.fn(), updateLabel: vi.fn() }));
vi.mock("@/hooks/store/use-label", () => ({ useLabel: () => store }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@makeplane/propel/components/input", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const web = projectOf("WEB", "w-acme");
const bug = labelOf(web, "bug", 65535);
const setLabelForm = vi.fn();

/** Renders the form, types the name, and clicks its button, which sends what the form holds. */
function submit(form: ReactElement, name: string) {
  emptyShown();
  renderToStaticMarkup(form);
  const [input] = shown.inputs;
  const button = shown.buttons.find((props) => props.variant === "primary");
  if (!input || !button?.onClick) throw new Error("the form showed no name or no button");
  input.onChange(name);
  button.onClick({ preventDefault: () => {} });
}

/** The form for a change of a label, or for a creation with none. */
const formOf = (label?: Label) => (
  <CreateUpdateLabelInline
    projectId={web.id}
    labelForm
    setLabelForm={setLabelForm}
    isUpdating={label !== undefined}
    labelToUpdate={label}
  />
);

beforeEach(() => {
  signedIn();
  store.createLabel.mockReset();
  store.createLabel.mockResolvedValue(bug);
  store.updateLabel.mockReset();
  store.updateLabel.mockResolvedValue(bug);
  setLabelForm.mockReset();
  toasts.length = 0;
});

describe("the label settings' inline form", () => {
  it("creates a label in the page's project, at the top, with the name and colour it holds", async () => {
    submit(formOf(), "Feature");
    await vi.waitFor(() => expect(store.createLabel).toHaveBeenCalled());
    expect(store.createLabel.mock.calls).toEqual([[web.id, { name: "Feature", color: expect.any(String) }]]);
  });

  it("changes a label with the fields its form edits, the only ones nerve's LabelUpdate takes", async () => {
    submit(formOf(bug), "defect");
    await vi.waitFor(() => expect(store.updateLabel).toHaveBeenCalled());
    expect(store.updateLabel.mock.calls).toEqual([[bug.id, { name: "defect", color: "#F59E0B" }]]);
  });

  it("closes once nerve has made the change, not before", async () => {
    const answer = heldChange<Label>();
    store.updateLabel.mockReturnValueOnce(answer.sent);
    submit(formOf(bug), "defect");
    await vi.waitFor(() => expect(store.updateLabel).toHaveBeenCalled());
    await pageSettled();
    expect(setLabelForm.mock.calls).toEqual([]);
    answer.answer(bug);
    await vi.waitFor(() => expect(setLabelForm.mock.calls).toEqual([[false]]));
  });

  it.each([
    { change: "creation", form: () => formOf(), sent: () => store.createLabel },
    { change: "change", form: () => formOf(bug), sent: () => store.updateLabel },
  ])("stays open when nerve refuses a $change, its name taken said under the name, no toast", async (row) => {
    row.sent().mockRejectedValueOnce(refusal(409, "project.label_name_taken"));
    submit(row.form(), "BUG");
    await vi.waitFor(() => expect(row.sent()).toHaveBeenCalled());
    await pageSettled();
    expect([setLabelForm.mock.calls, toasts]).toEqual([[], []]);
  });
});
````

`web/apps/web/core/components/labels/create-update-label-inline.tsx`（修改，11 处）：

````old web/apps/web/core/components/labels/create-update-label-inline.tsx
import { TwitterPicker } from "react-color";
import type { SubmitHandler } from "react-hook-form";
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
import { TwitterPicker } from "react-color";
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
import type { Label, LabelCreate, LabelUpdate } from "@nerve/api-client";
// hooks
import { useRefusalToast } from "@/hooks/use-refusal-toast";

export type TLabelOperationsCallbacks = {
  createLabel: (data: LabelCreate) => Promise<Label>;
  updateLabel: (labelId: string, data: LabelUpdate) => Promise<Label>;
};
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
import type { Label } from "@nerve/api-client";
// local imports
import { useLabelChanges, type LabelUnderField } from "./use-label-changes";
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
type TCreateUpdateLabelInlineProps = {
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
type TCreateUpdateLabelInlineProps = {
  /** The project a label created goes in. */
  projectId: string;
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
  isUpdating: boolean;
  labelOperationsCallbacks: TLabelOperationsCallbacks;
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
  isUpdating: boolean;
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
    const { labelForm, setLabelForm, isUpdating, labelOperationsCallbacks, labelToUpdate, onClose } = props;
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
    const { projectId, labelForm, setLabelForm, isUpdating, labelToUpdate, onClose } = props;
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
      reset,
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
      reset,
      setError,
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
    const toastRefusal = useRefusalToast();
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
    const { create, update } = useLabelChanges();
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
    /** Shows nerve's reason for a refused creation or change, and keeps what the form holds. */
    const handleRefusal = (error: unknown, formData: TLabelFormValues) => {
      toastRefusal(error);
      reset(formData);
    };

    const handleLabelCreate: SubmitHandler<TLabelFormValues> = async (formData) => {
      if (isSubmitting) return;

      await labelOperationsCallbacks
        .createLabel(formData)
        .then(() => handleClose())
        .catch((error: unknown) => handleRefusal(error, formData));
    };

    const handleLabelUpdate: SubmitHandler<TLabelFormValues> = async (formData) => {
      if (!labelToUpdate?.id || isSubmitting) return;

      await labelOperationsCallbacks
        .updateLabel(labelToUpdate.id, formData)
        .then(() => handleClose())
        .catch((error: unknown) => handleRefusal(error, formData));
    };

````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
    // Busy until the page has followed nerve's answer: the form closes once nerve has made the label or the change,
    // and stays with what it holds when nerve refuses it, the reason under the name or in a toast.
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
      if (isUpdating) {
        handleLabelUpdate(formData);
      } else {
        handleLabelCreate(formData);
      }
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
      const underField: LabelUnderField = (field, message) => setError(field, { type: "server", message });
      return labelToUpdate
        ? update(labelToUpdate.id, formData, underField, handleClose)
        : create(projectId, formData, underField, handleClose);
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
          <Button variant="secondary" onClick={() => handleClose()}>
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
          <Button variant="secondary" onClick={() => handleClose()} disabled={isSubmitting}>
````
````old web/apps/web/core/components/labels/create-update-label-inline.tsx
              handleSubmit(handleFormSubmit)();
````
````new web/apps/web/core/components/labels/create-update-label-inline.tsx
              void handleSubmit(handleFormSubmit)();
````

`web/apps/web/core/components/labels/delete-label-modal.tsx`（修改，4 处）：

````old web/apps/web/core/components/labels/delete-label-modal.tsx
// hooks
import { useLabel } from "@/hooks/store/use-label";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
````
````new web/apps/web/core/components/labels/delete-label-modal.tsx
// local imports
import { useLabelChanges } from "./use-label-changes";
````
````old web/apps/web/core/components/labels/delete-label-modal.tsx
  // store hooks
  const { deleteLabel } = useLabel();
  const toastRefusal = useRefusalToast();
````
````new web/apps/web/core/components/labels/delete-label-modal.tsx
  const { remove } = useLabelChanges();
````
````old web/apps/web/core/components/labels/delete-label-modal.tsx
  };

  const handleDeletion = async () => {
    if (!data) return;

````
````new web/apps/web/core/components/labels/delete-label-modal.tsx
  };

  // Busy, and the dialog not to be dismissed, until the page has followed nerve's answer: it closes once nerve has
  // deleted the label, and stays, its buttons enabled again, when nerve refuses, the reason in a toast.
  const handleDeletion = async () => {
    if (!data) return;
````
````old web/apps/web/core/components/labels/delete-label-modal.tsx

    await deleteLabel(data.id)
      .then(() => handleClose())
      .catch((error: unknown) => {
        setIsDeleteLoading(false);
        toastRefusal(error);
      });
````
````new web/apps/web/core/components/labels/delete-label-modal.tsx
    await remove(data.id, handleClose);
    setIsDeleteLoading(false);
````

- [ ] **Step 3: 列表、组和一行不再传回调**

`web/apps/web/core/components/labels/project-setting-label-group.tsx`（修改，6 处）：

````old web/apps/web/core/components/labels/project-setting-label-group.tsx
// components
import type { TLabelOperationsCallbacks } from "./create-update-label-inline";
````
````new web/apps/web/core/components/labels/project-setting-label-group.tsx
// components
````
````old web/apps/web/core/components/labels/project-setting-label-group.tsx
  ) => void;
  labelOperationsCallbacks: TLabelOperationsCallbacks;
````
````new web/apps/web/core/components/labels/project-setting-label-group.tsx
  ) => void;
````
````old web/apps/web/core/components/labels/project-setting-label-group.tsx
    isEditable = false,
    labelOperationsCallbacks,
````
````new web/apps/web/core/components/labels/project-setting-label-group.tsx
    isEditable = false,
````
````old web/apps/web/core/components/labels/project-setting-label-group.tsx
                        <CreateUpdateLabelInline
````
````new web/apps/web/core/components/labels/project-setting-label-group.tsx
                        <CreateUpdateLabelInline
                          projectId={label.project_id}
````
````old web/apps/web/core/components/labels/project-setting-label-group.tsx
                          labelToUpdate={label}
                          labelOperationsCallbacks={labelOperationsCallbacks}
````
````new web/apps/web/core/components/labels/project-setting-label-group.tsx
                          labelToUpdate={label}
````
````old web/apps/web/core/components/labels/project-setting-label-group.tsx
                                  isEditable={isEditable}
                                  labelOperationsCallbacks={labelOperationsCallbacks}
````
````new web/apps/web/core/components/labels/project-setting-label-group.tsx
                                  isEditable={isEditable}
````

`web/apps/web/core/components/labels/project-setting-label-item.tsx`（修改，8 处）：

````old web/apps/web/core/components/labels/project-setting-label-item.tsx
import type { Label } from "@nerve/api-client";
// hooks
import { useLabel } from "@/hooks/store/use-label";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// components
import type { TLabelOperationsCallbacks } from "./create-update-label-inline";
````
````new web/apps/web/core/components/labels/project-setting-label-item.tsx
import type { Label } from "@nerve/api-client";
// components
````
````old web/apps/web/core/components/labels/project-setting-label-item.tsx
import { LabelDndHOC } from "./label-drag-n-drop-HOC";
````
````new web/apps/web/core/components/labels/project-setting-label-item.tsx
import { LabelDndHOC } from "./label-drag-n-drop-HOC";
import { useLabelChanges } from "./use-label-changes";
````
````old web/apps/web/core/components/labels/project-setting-label-item.tsx
  ) => void;
  labelOperationsCallbacks: TLabelOperationsCallbacks;
````
````new web/apps/web/core/components/labels/project-setting-label-item.tsx
  ) => void;
````
````old web/apps/web/core/components/labels/project-setting-label-item.tsx
    onDrop,
    labelOperationsCallbacks,
````
````new web/apps/web/core/components/labels/project-setting-label-item.tsx
    onDrop,
````
````old web/apps/web/core/components/labels/project-setting-label-item.tsx
  // nerve hooks
  const toastRefusal = useRefusalToast();
  // store hooks
  const { updateLabel } = useLabel();

  const removeFromGroup = () => {
    updateLabel(label.id, { parent_id: null }).catch((error: unknown) => {
      toastRefusal(error);
    });
  };
````
````new web/apps/web/core/components/labels/project-setting-label-item.tsx
  const { removeFromGroup } = useLabelChanges();
````
````old web/apps/web/core/components/labels/project-setting-label-item.tsx
      onClick: removeFromGroup,
````
````new web/apps/web/core/components/labels/project-setting-label-item.tsx
      onClick: () => void removeFromGroup(label.id),
````
````old web/apps/web/core/components/labels/project-setting-label-item.tsx
              <CreateUpdateLabelInline
````
````new web/apps/web/core/components/labels/project-setting-label-item.tsx
              <CreateUpdateLabelInline
                projectId={label.project_id}
````
````old web/apps/web/core/components/labels/project-setting-label-item.tsx
                labelToUpdate={label}
                labelOperationsCallbacks={labelOperationsCallbacks}
````
````new web/apps/web/core/components/labels/project-setting-label-item.tsx
                labelToUpdate={label}
````

`web/apps/web/core/components/labels/project-setting-label-list.tsx`（修改，11 处）：

````old web/apps/web/core/components/labels/project-setting-label-list.tsx
import type { Label, LabelCreate } from "@nerve/api-client";
import { Loader } from "@nerve/ui";
import type { TLabelOperationsCallbacks } from "@/components/labels";
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
import type { Label } from "@nerve/api-client";
import { Loader } from "@nerve/ui";
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
import { useUserPermissions } from "@/hooks/store/user";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
import { useUserPermissions } from "@/hooks/store/user";
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
import { SettingsHeading } from "../settings/heading";
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
import { SettingsHeading } from "../settings/heading";
import { useLabelChanges } from "./use-label-changes";
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
  const { t } = useTranslation();
  const toastRefusal = useRefusalToast();
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
  const { t } = useTranslation();
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
  const { projectLabels, updateLabelPosition, projectLabelsTree, createLabel, updateLabel } = useLabel();
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
  const { projectLabels, projectLabelsTree } = useLabel();
  const { move } = useLabelChanges();
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
  const isEditable = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.PROJECT);
  const labelOperationsCallbacks: TLabelOperationsCallbacks = {
    createLabel: (data: LabelCreate) => createLabel(projectId, data),
    updateLabel,
  };
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
  const isEditable = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.PROJECT);
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
    updateLabelPosition(draggingLabelId, droppedParentId, droppedLabelId, dropAtEndOfList).catch((error: unknown) => {
      toastRefusal(error);
    });
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
    void move(draggingLabelId, droppedParentId, droppedLabelId, dropAtEndOfList);
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
            <CreateUpdateLabelInline
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
            <CreateUpdateLabelInline
              projectId={projectId}
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
              isUpdating={isUpdating}
              labelOperationsCallbacks={labelOperationsCallbacks}
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
              isUpdating={isUpdating}
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
                    isEditable={isEditable}
                    labelOperationsCallbacks={labelOperationsCallbacks}
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
                    isEditable={isEditable}
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
                  isEditable={isEditable}
                  labelOperationsCallbacks={labelOperationsCallbacks}
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
                  isEditable={isEditable}
````

- [ ] **Step 4: 文案和静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/app/(all)/\\[workspaceSlug\\]/(settings)/settings/projects/\\[projectId\\]/states-labels.test.tsx"
````
````new .oxlintrc.json
        "web/apps/web/app/(all)/\\[workspaceSlug\\]/(settings)/settings/projects/\\[projectId\\]/states-labels.test.tsx",
        "web/apps/web/core/components/labels/use-label-changes.ts",
        "web/apps/web/core/components/labels/use-label-changes.test.ts"
````

`web/packages/i18n/src/locales/en/project-settings.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/project-settings.json
      "label_max_char": "Label name should not exceed 255 characters"
````
````new web/packages/i18n/src/locales/en/project-settings.json
      "label_max_char": "Label name should not exceed 255 characters",
      "parent_not_allowed": "A label can go only under a label at the top of its project: labels have two levels."
````

`web/packages/i18n/src/locales/zh-CN/project-settings.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/project-settings.json
      "label_max_char": "标签名称不应超过255个字符"
````
````new web/packages/i18n/src/locales/zh-CN/project-settings.json
      "label_max_char": "标签名称不应超过255个字符",
      "parent_not_allowed": "标签只能放在本项目的顶层标签之下：标签只有两层。"
````

- [ ] **Step 5: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 69 条规则、3 个例外，没有命中；web 的 oxlint 346 条，等于上限；`en` 与 `zh-CN` 的键一致。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 108 个全部通过。

- [ ] **Step 6: 提交**

```bash
git add .oxlintrc.json web/apps/web/core/components/labels/create-update-label-inline.test.tsx web/apps/web/core/components/labels/create-update-label-inline.tsx web/apps/web/core/components/labels/delete-label-modal.tsx web/apps/web/core/components/labels/project-setting-label-group.tsx web/apps/web/core/components/labels/project-setting-label-item.tsx web/apps/web/core/components/labels/project-setting-label-list.tsx web/apps/web/core/components/labels/use-label-changes.test.ts web/apps/web/core/components/labels/use-label-changes.ts web/packages/i18n/src/locales/en/project-settings.json web/packages/i18n/src/locales/zh-CN/project-settings.json
```
```bash
git commit -m "feat(M3/P11): the label settings follow each change in the session it was sent in; a name taken shows under the name, a third level is said as such

useLabelChanges sends each change of a project's labels through the store
and follows it with followInSession; a creation's or a change's refusal
shows under the name when nerve names it, a move nerve refuses for its
parent says labels have two levels, the others are toasts. The inline
form stays busy, Cancel too, until the page has followed nerve's answer;
the deletion's dialog closes only once the label is deleted. The
components take no callbacks.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p11.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T4.1` | 标签的创建不论会话都跟进 | `use-label-changes.test.ts` | vitest |
| `T4.2` | 标签的修改不论会话都跟进 | `use-label-changes.test.ts` | vitest |
| `T4.3` | 标签的移动不论会话都跟进 | `use-label-changes.test.ts` | vitest |
| `T4.4` | 标签的移出组不论会话都跟进 | `use-label-changes.test.ts` | vitest |
| `T4.5` | 标签的删除不论会话都跟进 | `use-label-changes.test.ts` | vitest |
| `T4.6` | 标签的名字已被占用在提示里，不在名字下 | `use-label-changes.test.ts`、故事 P7（Task 5 起） | vitest；端到端 |
| `T4.7` | 第三层按普通的拒绝说，不说标签只有两层 | `use-label-changes.test.ts`、故事 P7（Task 5 起） | vitest；端到端 |
| `T4.8` | 标签的表单在 nerve 回答之前不忙 | 故事 P7（Task 5 起） | 端到端 |
| `T4.9` | 标签的表单在 nerve 回答之前取消仍可用 | 故事 P7（Task 5 起） | 端到端 |
| `T4.10` | 删除标签的确认框在 nerve 回答之前关上 | 故事 P7（Task 5 起） | 端到端 |
| `T4.11` | 标签的表单在名字下什么都不显示 | 故事 P7（Task 5 起） | 端到端 |
| `T4.12` | 在 `use-label-changes.ts` 里写一个非空断言 | oxlint（`check:lint`） | 静态 |

---

### Task 5: 标签设置页从路由读项目，控件只给能改标签的人，拖动由此刻的手柄开始；P7 的页面版本

**Files:**
- Create: `e2e/stories/project/p7-labels-page.spec.ts`、`web/apps/web/core/components/labels/project-setting-label-group.test.tsx`
- Modify: `.oxlintrc.json`、`e2e/stories/project/p7-labels.spec.ts`、`e2e/stories/smoke/s2-web-app.spec.ts`、`e2e/stories/workspace/w11-guest-bounds.spec.ts`、`tools/keywords.json`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx`、`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx`、`web/apps/web/core/components/labels/label-block/label-item-block.tsx`、`web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx`、`web/apps/web/core/components/labels/project-setting-label-group.tsx`、`web/apps/web/core/components/labels/project-setting-label-list.tsx`、`web/apps/web/core/store/label.store.changes.test.ts`、`web/apps/web/core/store/label.store.test.ts`、`web/apps/web/core/store/label.store.ts`

**Interfaces:**
- Produces（spec 2.5；M3 设计 2 的 P7、W11，3.16，7.6，9.2；P10 的 F-2 和 `project-address-route`；P10 review 第 6 节 P11 一行；P8b 的 P7）：
  - 标签设置页（`labels/page.tsx`）：工作区和项目取自路由的参数，标题用 `getProjectById(projectId)` 的名字；门是 `allowPermissions([ADMIN, MEMBER], PROJECT, workspaceSlug, projectId)`（`label.list`，访客看到没有权限）；列表收工作区和项目。原来读 router store 的 `currentProjectDetails`，`allowPermissions` 不写工作区和项目。
  - 标签列表（`project-setting-label-list.tsx`）：`TProjectSettingsLabelList`（`workspaceSlug`、`projectId`，不再读 `useParams`）；项目的标签按项目的 id 读（`getProjectLabels(projectId)`、`getProjectLabelsTree(projectId)`：label store 的 `computedFn`，原来是读 router store 的项目的 `projectLabelsTree` getter，删除）；改标签的控件只给 `allowPermissions([ADMIN], PROJECT, workspaceSlug, projectId)`（`label.create` 等）：标题旁的"Add label"和没有标签时的按钮（原来没有标签时每个成员都看得到）。
  - 组（`project-setting-label-group.tsx`）：组的那一行收 `disabled={!isEditable}`：菜单和拖动手柄只给能改标签的人（原来项目的每个成员都看得到组的菜单和手柄，nerve 拒绝他们的修改）。
  - 拖动（`label-drag-n-drop-HOC.tsx`）：手柄是此刻渲染的那一个（回调的 ref 写进 state，effect 以它为依赖）：一行重新渲染出新的手柄（编辑关上之后）时重新登记，不再靠列表每次渲染都新建的 `onDrop`（P8b 的 P7）；没有手柄就不登记拖动和放下：一行只给能改标签的人渲染手柄、编辑时不渲染，所以 HOC 不再自己判断权限（原来的 `allowPermissions` 不写工作区和项目，删除）。
  - 一行（`label-item-block.tsx`）：没有调用方传的 `draggable` 删除（P8b A.9 的行）；菜单（"Options"）和删除按钮（"Delete"，`type="button"`）有名字；菜单和删除按钮在焦点进到这一行时和指针经过时一样显示（`group-focus-within:opacity-100`；原来键盘到达时看不见）。
  - `tools/keywords.json`：`project-address-route` 的文件加上标签设置页、`project-setting-label-list.tsx`、`label-drag-n-drop-HOC.tsx`（规则的说明随之改）。
  - 静态检查：`.oxlintrc.json` 的非空断言的范围加上 `project-setting-label-group.test.tsx`。

**Tests:**
- vitest：`states-labels.test.tsx` 加标签设置页：`shows $who the labels, their controls and its button`（项目管理员、是工作区管理员的项目成员、是工作区管理员的项目访客）；`shows its member the labels, and no control`；`tells its guest he may not see them`；`shows the button of its empty state, with no label, to those who may add one alone`（管理员看到两个按钮，成员一个也没有）。`project-setting-label-group.test.tsx`（新文件）：`shows one who may change the labels the group's controls and those of the labels under it`；`shows any other member the group and the labels under it, with no control`。`label.store.test.ts`、`label.store.changes.test.ts` 读 `getProjectLabelsTree(web.id)`。
- 端到端：`P7 (page): the admin creates Bug, its form gone through by Tab and held until nerve answers, and Feature; bug is said taken under the name; he drags UI under Bug, which sends its parent alone, renames Bug Defect and drags Feature first; Feature put under Defect meanwhile, a third level the page asks for is said refused; its member sees the labels and can change none, a group's neither, a workspace admin who is its member can; Defect is deleted, its dialog held, with the labels under it`（新文件：表单打开时焦点在名字上，Tab 次序 `labelName`、`Cancel`、`Add`；创建扣住时"Adding"和"Cancel"都按不了，请求体只有名字和颜色；`bug` 的 409 在表单下面一行；拖 UI 到 Bug 的中间：只发 `parent_id`；从组的菜单改名：请求体是名字和颜色；拖 Feature 到 Docs 的上方：`{ parent_id: null, sort_order: 65535 }`；另一处把 Feature 放进 Defect 之后，把 Docs 拖到 Feature 上：422，提示说标签只有两层；点别处之后 Docs 的菜单和删除按钮看不见，从菜单按 Tab 到删除按钮时看得见；成员看不到"Add label"、手柄、菜单、删除按钮，也没有可拖的标签（`[draggable="true"]`），是工作区管理员的成员各有四个（删除按钮三个：组没有）；重新载入之后从菜单删除 Defect，扣住时 Escape 不关、"Cancel"按不了，子标签一起删除；`expectLabels` 核对每一行）。W11 的页面版本加标签设置页；S2 的挂载清单加上标签设置页（spec 2.11）。

- [ ] **Step 1: 标签的 getter 按项目的 id**

`web/apps/web/core/store/label.store.changes.test.ts`（修改，1 处）：

````old web/apps/web/core/store/label.store.changes.test.ts
    expect(store.projectLabelsTree).toEqual([
````
````new web/apps/web/core/store/label.store.changes.test.ts
    expect(store.getProjectLabelsTree(web.id)).toEqual([
````

`web/apps/web/core/store/label.store.test.ts`（修改，4 处）：

````old web/apps/web/core/store/label.store.test.ts
    expect(store.getProjectLabelIds(web.id)).toEqual(ids(ordered));
    expect(store.projectLabelsTree).toEqual([
````
````new web/apps/web/core/store/label.store.test.ts
    expect(store.getProjectLabelIds(web.id)).toEqual(ids(ordered));
    expect(store.getProjectLabelsTree(web.id)).toEqual([
````
````old web/apps/web/core/store/label.store.test.ts
    expect(store.projectLabelsTree?.find((top) => top.id === feature.id)?.children).toEqual([frontend, backend, infra]);
````
````new web/apps/web/core/store/label.store.test.ts
    expect(store.getProjectLabelsTree(web.id)?.find((top) => top.id === feature.id)?.children).toEqual([
      frontend,
      backend,
      infra,
    ]);
````
````old web/apps/web/core/store/label.store.test.ts
    await until(() => freed.settled, "the answer");
    expect(store.projectLabelsTree).toEqual([
````
````new web/apps/web/core/store/label.store.test.ts
    await until(() => freed.settled, "the answer");
    expect(store.getProjectLabelsTree(web.id)).toEqual([
````
````old web/apps/web/core/store/label.store.test.ts
    expect(store.projectLabelsTree?.[0]).toEqual({ ...feature, children: [backendFirst, bugBetween, frontend] });
````
````new web/apps/web/core/store/label.store.test.ts
    expect(store.getProjectLabelsTree(web.id)?.[0]).toEqual({
      ...feature,
      children: [backendFirst, bugBetween, frontend],
    });
````

`web/apps/web/core/store/label.store.ts`（修改，5 处）：

````old web/apps/web/core/store/label.store.ts
  projectLabels: Label[] | undefined;
  /** The address's project's labels at the top, each with the labels under it; undefined until fetched. */
  projectLabelsTree: LabelTree[] | undefined;
````
````new web/apps/web/core/store/label.store.ts
  projectLabels: Label[] | undefined;
````
````old web/apps/web/core/store/label.store.ts
  getProjectLabels: (projectId: string | null | undefined) => Label[] | undefined;
````
````new web/apps/web/core/store/label.store.ts
  getProjectLabels: (projectId: string | null | undefined) => Label[] | undefined;
  getProjectLabelsTree: (projectId: string | null | undefined) => LabelTree[] | undefined;
````
````old web/apps/web/core/store/label.store.ts
      projectLabels: computed,
      projectLabelsTree: computed,
````
````new web/apps/web/core/store/label.store.ts
      projectLabels: computed,
````
````old web/apps/web/core/store/label.store.ts

  get projectLabelsTree() {
    const labels = this.projectLabels;
    return labels?.filter((label) => label.parent_id === null).map((top) => treeOf(top, labels));
  }

````
````new web/apps/web/core/store/label.store.ts

````
````old web/apps/web/core/store/label.store.ts
    return labels && inOrder(labels);
````
````new web/apps/web/core/store/label.store.ts
    return labels && inOrder(labels);
  });

  /** @description the project's labels at the top, each with the labels under it; undefined as getProjectLabels */
  getProjectLabelsTree = computedFn((projectId: string | null | undefined): LabelTree[] | undefined => {
    const labels = this.getProjectLabels(projectId);
    return labels?.filter((label) => label.parent_id === null).map((top) => treeOf(top, labels));
````

- [ ] **Step 2: 标签设置页和列表**

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx`（修改，7 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
// local imports
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
// local imports
import type { Route } from "./+types/page";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
function LabelsSettingsPage() {
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
function LabelsSettingsPage({ params }: Route.ComponentProps) {
  // router: the address's project, as the store gives it (the wrapper shows the page once it has)
  const { workspaceSlug, projectId } = params;
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
  const { currentProjectDetails } = useProject();
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
  const { getProjectById } = useProject();
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
  const pageTitle = currentProjectDetails?.name ? `${currentProjectDetails?.name} - Labels` : undefined;
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
  const project = getProjectById(projectId);
  const pageTitle = project ? `${project.name} - Labels` : undefined;
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
  // derived values
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
  // its members see its labels; its guests are told they may not, as on its states (M3 design 2 W11)
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
    EUserPermissionsLevel.PROJECT
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
    EUserPermissionsLevel.PROJECT,
    workspaceSlug,
    projectId
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
        <ProjectSettingsLabelList />
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx
        <ProjectSettingsLabelList workspaceSlug={workspaceSlug} projectId={projectId} />
````

`web/apps/web/core/components/labels/project-setting-label-list.tsx`（修改，6 处）：

````old web/apps/web/core/components/labels/project-setting-label-list.tsx
import { observer } from "mobx-react";
import { useParams } from "react-router";
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
import { observer } from "mobx-react";
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
export const ProjectSettingsLabelList = observer(function ProjectSettingsLabelList() {
  // router
  const { projectId } = useParams();
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
type TProjectSettingsLabelList = {
  workspaceSlug: string;
  projectId: string;
};

export const ProjectSettingsLabelList = observer(function ProjectSettingsLabelList(props: TProjectSettingsLabelList) {
  const { workspaceSlug, projectId } = props;
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
  const { projectLabels, projectLabelsTree } = useLabel();
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
  const { getProjectLabels, getProjectLabelsTree } = useLabel();
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx

  if (!projectId) return null;

````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx

````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
  const isEditable = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.PROJECT);
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
  const projectLabels = getProjectLabels(projectId);
  const projectLabelsTree = getProjectLabelsTree(projectId);
  // who may change the labels (label.create and the others): its admins, and its members who are the workspace's
  const isEditable = allowPermissions(
    [EUserPermissions.ADMIN],
    EUserPermissionsLevel.PROJECT,
    workspaceSlug,
    projectId
  );
````
````old web/apps/web/core/components/labels/project-setting-label-list.tsx
              actions={[
                {
                  label: t("settings_empty_state.labels.cta_primary"),
                  onClick: () => {
                    newLabel();
                  },
                },
              ]}
````
````new web/apps/web/core/components/labels/project-setting-label-list.tsx
              actions={
                isEditable
                  ? [
                      {
                        label: t("settings_empty_state.labels.cta_primary"),
                        onClick: () => {
                          newLabel();
                        },
                      },
                    ]
                  : undefined
              }
````

- [ ] **Step 3: 组、拖动和一行**

`web/apps/web/core/components/labels/label-block/label-item-block.tsx`（修改，8 处）：

````old web/apps/web/core/components/labels/label-block/label-item-block.tsx

import type { MutableRefObject } from "react";
````
````new web/apps/web/core/components/labels/label-block/label-item-block.tsx

````
````old web/apps/web/core/components/labels/label-block/label-item-block.tsx
import { useOutsideClickDetector } from "@nerve/hooks";
````
````new web/apps/web/core/components/labels/label-block/label-item-block.tsx
import { useOutsideClickDetector } from "@nerve/hooks";
import { useTranslation } from "@nerve/i18n";
````
````old web/apps/web/core/components/labels/label-block/label-item-block.tsx
  dragHandleRef: MutableRefObject<HTMLButtonElement | null>;
  disabled?: boolean;
  draggable?: boolean;
````
````new web/apps/web/core/components/labels/label-block/label-item-block.tsx
  dragHandleRef: (element: HTMLButtonElement | null) => void;
  /** Whether the label is shown alone, with no grip and no control that changes it. */
  disabled?: boolean;
````
````old web/apps/web/core/components/labels/label-block/label-item-block.tsx
    disabled = false,
    draggable = true,
  } = props;
````
````new web/apps/web/core/components/labels/label-block/label-item-block.tsx
    disabled = false,
  } = props;
  const { t } = useTranslation();
````
````old web/apps/web/core/components/labels/label-block/label-item-block.tsx
        {!disabled && draggable && (
````
````new web/apps/web/core/components/labels/label-block/label-item-block.tsx
        {!disabled && (
````
````old web/apps/web/core/components/labels/label-block/label-item-block.tsx
              : "opacity-0 group-hover:pointer-events-auto group-hover:opacity-100"
````
````new web/apps/web/core/components/labels/label-block/label-item-block.tsx
              : "opacity-0 group-focus-within:opacity-100 group-hover:pointer-events-auto group-hover:opacity-100"
````
````old web/apps/web/core/components/labels/label-block/label-item-block.tsx
          <CustomMenu ellipsis menuButtonOnClick={() => setIsMenuActive(!isMenuActive)} useCaptureForOutsideClick>
````
````new web/apps/web/core/components/labels/label-block/label-item-block.tsx
          <CustomMenu
            ellipsis
            ariaLabel={t("common.options")}
            menuButtonOnClick={() => setIsMenuActive(!isMenuActive)}
            useCaptureForOutsideClick
          >
````
````old web/apps/web/core/components/labels/label-block/label-item-block.tsx
              <button
````
````new web/apps/web/core/components/labels/label-block/label-item-block.tsx
              <button
                type="button"
                aria-label={t("common.actions.delete")}
````

`web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx`（修改，10 处）：

````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx

import type { MutableRefObject } from "react";
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx

````
````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
// types
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
// types
````
````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
// components
import { useUserPermissions } from "@/hooks/store/user";
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
// components
````
````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
    dragHandleRef: MutableRefObject<HTMLButtonElement | null>
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
    dragHandleRef: (element: HTMLButtonElement | null) => void
````
````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
  const dragHandleRef = useRef<HTMLButtonElement | null>(null);

  const { allowPermissions } = useUserPermissions();
  const isEditable = allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.PROJECT);
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
  // the grip as the row renders it now: a row rendered anew (once its edit closes, say) has a new one, registered anew
  const [dragHandle, setDragHandle] = useState<HTMLButtonElement | null>(null);
````
````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
    const labelElement = labelRef.current;
    const dragHandleElement = dragHandleRef.current;
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
    const labelElement = labelRef.current;
````
````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
    if (!labelElement || !isEditable) return;
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
    // a label is dragged by its grip alone, which its row renders only for those who may change the labels, and not
    // while the label is edited
    if (!labelElement || !dragHandle) return;
````
````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
        dragHandle: dragHandleElement ?? undefined,
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
        dragHandle,
````
````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
  }, [label, isChild, isGroup, isLastChild, onDrop, isEditable]);
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
  }, [label, isChild, isGroup, isLastChild, onDrop, dragHandle]);
````
````old web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
      {children(isDragging, isMakeChild, dragHandleRef)}
````
````new web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx
      {children(isDragging, isMakeChild, setDragHandle)}
````

`web/apps/web/core/components/labels/project-setting-label-group.tsx`（修改，1 处）：

````old web/apps/web/core/components/labels/project-setting-label-group.tsx
                          dragHandleRef={dragHandleRef}
````
````new web/apps/web/core/components/labels/project-setting-label-group.tsx
                          dragHandleRef={dragHandleRef}
                          disabled={!isEditable}
````

- [ ] **Step 4: 测试、关键词规则和静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/components/labels/use-label-changes.test.ts"
````
````new .oxlintrc.json
        "web/apps/web/core/components/labels/use-label-changes.test.ts",
        "web/apps/web/core/components/labels/project-setting-label-group.test.tsx"
````

`tools/keywords.json`（修改，4 处）：

````old tools/keywords.json
      "why": "项目页面从路由参数读地址（项目、工作区），不从 router store 读：project store 的 currentProjectDetails 和 member store 的 projectMemberIds 读 router store 的 projectId，workspace store 的 currentWorkspace 读它的 workspaceSlug，allowPermissions 不传 slug 和项目时也退回 router store（permissions.store.ts）。router store 由 StoreWrapper 的 effect 在渲染之后写入，同一路由换了项目时晚一次提交，那一次页面按上一个项目显示、按上一个项目判断权限（M3/P10 修复轮，裁定 F-2；P10 规格 §5）。本规则只管 P10、P11 已经改过来的文件：常规设置页和它的控制区、四个功能页、自动化页、成员页、成员列表、成员选择、项目设置侧栏的头部（P10），状态设置页和它的根组件（P11）；后面的阶段（P11 的标签页，M4、M6、M7 各自的页面）改它们的读法时，把那些文件加进本规则的 files，收尾删掉 projectMemberIds 这个 getter。测试不算",
````
````new tools/keywords.json
      "why": "项目页面从路由参数读地址（项目、工作区），不从 router store 读：project store 的 currentProjectDetails 和 member store 的 projectMemberIds 读 router store 的 projectId，workspace store 的 currentWorkspace 读它的 workspaceSlug，allowPermissions 不传 slug 和项目时也退回 router store（permissions.store.ts）。router store 由 StoreWrapper 的 effect 在渲染之后写入，同一路由换了项目时晚一次提交，那一次页面按上一个项目显示、按上一个项目判断权限（M3/P10 修复轮，裁定 F-2；P10 规格 §5）。本规则只管 P10、P11 已经改过来的文件：常规设置页和它的控制区、四个功能页、自动化页、成员页、成员列表、成员选择、项目设置侧栏的头部（P10），状态设置页和它的根组件、标签设置页、标签列表和它的拖放组件（P11）；后面的阶段（M4、M6、M7 各自的页面）改它们的读法时，把那些文件加进本规则的 files，收尾删掉 projectMemberIds 这个 getter。测试不算",
````
````old tools/keywords.json
        "source": "^web/apps/web/(?:app/\\(all\\)/\\[workspaceSlug\\]/\\(settings\\)/settings/projects/\\[projectId\\]/(?:page|members/page|states/page|automations/page|features/(?:cycles|intake|modules|views)/page)|core/components/(?:project/(?:settings/control-section|member-list|member-select)|project-states/root|settings/project/sidebar/header))\\.tsx$",
````
````new tools/keywords.json
        "source": "^web/apps/web/(?:app/\\(all\\)/\\[workspaceSlug\\]/\\(settings\\)/settings/projects/\\[projectId\\]/(?:page|members/page|states/page|labels/page|automations/page|features/(?:cycles|intake|modules|views)/page)|core/components/(?:project/(?:settings/control-section|member-list|member-select)|project-states/root|labels/(?:project-setting-label-list|label-drag-n-drop-HOC)|settings/project/sidebar/header))\\.tsx$",
````
````old tools/keywords.json
            "web/apps/web/core/components/project-states/root.tsx"
````
````new tools/keywords.json
            "web/apps/web/core/components/project-states/root.tsx",
            "web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx",
            "web/apps/web/core/components/labels/project-setting-label-list.tsx",
            "web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx"
````
````old tools/keywords.json
          "miss": [
            "web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx",
````
````new tools/keywords.json
          "miss": [
````

`web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx`（修改，7 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
import { describe, expect, it, vi } from "vitest";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
import { beforeEach, describe, expect, it, vi } from "vitest";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
import type { IUserPermissionStore } from "@/store/user/permissions.store";
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import LabelsPage from "./labels/page";
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
// Whom a project's state settings show what (M3 design 2 P6, W11, 9.2): its states to those nerve lets list them, but
// its guests, whom the page tells they may not (state.list: its members, whatever their role; design 2 W11); the
// controls that change them to those nerve lets change them (state.create and the others: its admins, and its members
// who are the workspace's admins, its guests among them). The page renders on the server at its route, through React
// Router's stub, which gives it the route's params, with stand-ins for its parts that write what they show: the list
// of groups says whether it has its controls. The caller's permissions and projects are the stores' own, over acme and
// its project web as nerve lists them to him (fake-permissions.ts), while the address the router store holds names no
// project: the page reads its project from its route.
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
// Whom a project's state and label settings show what (M3 design 2 P6, P7, W11, 9.2): its states and its labels to
// those nerve lets list them, but its guests, whom each page tells they may not (state.list and label.list: its
// members, whatever their role; design 2 W11); the controls that change them to those nerve lets change them
// (state.create, label.create and the others: its admins, and its members who are the workspace's admins, its guests
// among them). A page renders on the server at its route, through React Router's stub, which gives it the route's
// params, with stand-ins for its parts that write what they show: the list of groups and a label say whether they have
// their controls, the label list's heading and its empty state show their buttons. The caller's permissions and
// projects are the stores' own, over acme and its project web as nerve lists them to him (fake-permissions.ts), while
// the address the router store holds names no project: the page reads its project from its route.
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
const caller = vi.hoisted((): { permissions?: IUserPermissionStore; projects?: IProjectStore } => ({}));
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
const caller = vi.hoisted((): { permissions?: IUserPermissionStore; projects?: IProjectStore } => ({}));
/** web's one label, as the label store gives it. */
const bug = vi.hoisted(() => ({ id: "l-bug", name: "Bug", parent_id: null }));
/** web's labels as the label store lists them: Bug, but in the test of a project with none. */
const listed = vi.hoisted(() => ({ labels: [bug] }));
beforeEach(() => {
  listed.labels = [bug];
});
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
  useProjectState: () => ({ getGroupedProjectStates: () => ({}), fetchProjectStates: vi.fn() }),
}));
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
  useProjectState: () => ({ getGroupedProjectStates: () => ({}), fetchProjectStates: vi.fn() }),
}));
vi.mock("@/hooks/store/use-label", () => ({
  useLabel: () => ({
    getProjectLabels: () => listed.labels,
    getProjectLabelsTree: () =>
      listed.labels.map((label) => ({ id: label.id, name: label.name, parent_id: label.parent_id, children: [] })),
  }),
}));
vi.mock("@/components/labels/project-setting-label-item", () => ({
  ProjectSettingLabelItem: ({ label, isEditable }: { label: { name: string }; isEditable: boolean }) =>
    `${label.name}${isEditable ? " and its controls" : ""}`,
}));
vi.mock("@/components/labels/use-label-changes", () => ({ useLabelChanges: () => ({}) }));
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
vi.mock("@/components/settings/heading", () => ({ SettingsHeading: () => null }));
````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
vi.mock("@/components/settings/heading", () => ({ SettingsHeading: ({ control }: { control?: unknown }) => control }));
vi.mock("@nerve/propel/button", () => ({ Button: ({ children }: { children: unknown }) => `[${children}]` }));
vi.mock("@nerve/propel/empty-state", () => ({
  EmptyStateCompact: ({ actions }: { actions?: { label: string }[] }) =>
    (actions ?? []).map((action) => `[${action.label}]`).join(""),
}));
````
````old web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
  });
});

````
````new web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx
  });
});

describe("the label settings page", () => {
  it.each([its.admin, its.memberAdmin, its.guestAdmin])(
    "shows $who the labels, their controls and its button",
    async (row) => {
      const { text, name } = await openedBy(row, "labels", LabelsPage);
      expect(text).toBe(`[${name} - Labels][common.add_label]Bug and its controls`);
    }
  );

  it("shows its member the labels, and no control", async () => {
    const { text, name } = await openedBy(its.member, "labels", LabelsPage);
    expect(text).toBe(`[${name} - Labels]Bug`);
  });

  it("tells its guest he may not see them", async () => {
    const { text } = await openedBy(its.guest, "labels", LabelsPage);
    expect(text).toBe("not authorized");
  });

  it("shows the button of its empty state, with no label, to those who may add one alone", async () => {
    listed.labels = [];
    const admin = await openedBy(its.admin, "labels", LabelsPage);
    const member = await openedBy(its.member, "labels", LabelsPage);
    expect([admin.text, member.text]).toEqual([
      `[${admin.name} - Labels][common.add_label][settings_empty_state.labels.cta_primary]`,
      `[${member.name} - Labels]`,
    ]);
  });
});

````

`web/apps/web/core/components/labels/project-setting-label-group.test.tsx`（新文件，55 行）：

````file web/apps/web/core/components/labels/project-setting-label-group.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { labelOf, projectOf } from "@/store/project/fake-projects";
import { ProjectSettingLabelGroup } from "./project-setting-label-group";

// A label with labels under it, on the label settings page (M3 design 2 P7, 3.16, 9.2): its row and theirs show the
// controls that change them (a grip, a menu) to those nerve lets change the labels alone (label.update and the others:
// the project's admins, and its members who are the workspace's admins), as the page says by isEditable. The group
// renders on the server with stand-ins for its row and for the rows under it, which say whether they have their
// controls. Its session is fake-tab.ts's; the label store, which it does not read, a stand-in.

vi.mock("@/hooks/store/use-label", () => ({ useLabel: () => ({}) }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/components/labels/label-block/label-item-block", () => ({
  LabelItemBlock: ({ label, disabled }: { label: { name: string }; disabled?: boolean }) =>
    `${label.name}${disabled ? "" : " and its controls"};`,
}));
vi.mock("@/components/labels/project-setting-label-item", () => ({
  ProjectSettingLabelItem: ({ label, isEditable }: { label: { name: string }; isEditable: boolean }) =>
    `${label.name}${isEditable ? " and its controls" : ""};`,
}));

const web = projectOf("WEB", "w-acme");
const bug = labelOf(web, "Bug", 10000);
const ui = labelOf(web, "UI", 20000, { parent_id: bug.id });

/** The text the group of Bug, with UI under it, shows, for one who may change the labels or not. */
const shownWith = (isEditable: boolean) =>
  renderToStaticMarkup(
    <ProjectSettingLabelGroup
      label={bug}
      labelChildren={[ui]}
      handleLabelDelete={vi.fn()}
      isUpdating={false}
      setIsUpdating={vi.fn()}
      isLastChild
      onDrop={vi.fn()}
      isEditable={isEditable}
    />
  ).replace(/<[^>]*>/g, "");

describe("ProjectSettingLabelGroup", () => {
  it("shows one who may change the labels the group's controls and those of the labels under it", () => {
    expect(shownWith(true)).toBe("Bug and its controls;UI and its controls;");
  });

  it("shows any other member the group and the labels under it, with no control", () => {
    expect(shownWith(false)).toBe("Bug;UI;");
  });
});
````

- [ ] **Step 5: 端到端**

`e2e/stories/project/p7-labels-page.spec.ts`（新文件，249 行）：

````file e2e/stories/project/p7-labels-page.spec.ts
import type { Locator, Page } from "@playwright/test";

import {
  addProjectMembers,
  changeRole,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type Label,
} from "../../fixtures/api";
import { expectLabels } from "../../fixtures/assert/project";
import { accountId, bearer, emailFor } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { dragOnto, tabCycle } from "../../fixtures/project-pages";
import {
  closedByEscape,
  closedWithin,
  enabledWithin,
  registerOnboarded,
  sentHeld,
  sentTo,
} from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser } from "../../fixtures/workspace-pages";

// P7's page version (M3 design 2, 3.16, 7.6): a project's labels on its settings page, as its admin changes them, and
// as its member and a workspace admin who is its member see them. The API version is p7-labels.spec.ts.

/** The row of the label of name on page's label settings: its grip, its name, its menu; the innermost row with it. */
function labelRow(page: Page, name: string): Locator {
  return page.locator("div.group", { has: page.getByRole("heading", { name, exact: true, level: 6 }) }).last();
}

/**
 * The label of name on page as a place to drop another on, which tells where by the point: its row, the labels under
 * it and the lines that show where a label dropped goes; its top quarter puts the label dropped before it, its middle
 * under it. That is the nearest div around its row with no class.
 */
function dropZone(page: Page, name: string): Locator {
  return labelRow(page, name).locator("xpath=ancestor::div[not(@class)][1]");
}

/** Drags the label of name on page by its grip onto the label of onto, at its point at (its middle when left out). */
function dragLabel(page: Page, name: string, onto: string, at?: { x: number; y: number }): Promise<void> {
  const row = labelRow(page, name);
  return dragOnto(page, row, row.locator("button.cursor-grab"), dropZone(page, onto), at);
}

/** The inline form page shows to create or change a label: one at a time; the page scrolls to it. */
function labelForm(page: Page): Locator {
  return page.locator("div.scroll-m-8", { has: page.locator("#labelName") });
}

/** The labels of the project of projectId, as the caller of token lists them: the label of a name. */
async function labelsOf(api: Api, token: string, projectId: string): Promise<(name: string) => Label> {
  const { data } = await api.GET("/api/v0/projects/{project_id}/labels", {
    params: { path: { project_id: projectId } },
    headers: bearer(token),
  });
  const labels = data?.data ?? [];
  return (name) => {
    const label = labels.find((l) => l.name === name);
    if (!label) throw new Error(`${projectId} has no label ${name}`);
    return label;
  };
}

test("P7 (page): the admin creates Bug, its form gone through by Tab and held until nerve answers, and Feature; bug is said taken under the name; he drags UI under Bug, which sends its parent alone, renames Bug Defect and drags Feature first; Feature put under Defect meanwhile, a third level the page asks for is said refused; its member sees the labels and can change none, a group's neither, a workspace admin who is its member can; Defect is deleted, its dialog held, with the labels under it", async ({
  api,
  baseURL,
  browser,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const memEmail = emailFor(testInfo, "mem");
  const wandaEmail = emailFor(testInfo, "wanda");
  const [admin, mem, wanda] = await Promise.all([
    registerOnboarded(api, adminEmail),
    registerOnboarded(api, memEmail),
    registerOnboarded(api, wandaEmail),
  ]);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug });
  // mem and wanda members of acme and of Web; wanda then acme's admin, Web's member still
  await inviteAndAccept(api, admin.access_token, slug, { email: memEmail, token: mem.access_token }, 15);
  await inviteAndAccept(api, admin.access_token, slug, { email: wandaEmail, token: wanda.access_token }, 15);
  const web = await createProject(api, admin.access_token, slug, { name: "Web", identifier: "WEB" });
  const wandaId = await accountId(api, wanda.access_token);
  await addProjectMembers(api, admin.access_token, web.id, [
    { member_id: await accountId(api, mem.access_token), role: 15 },
    { member_id: wandaId, role: 15 },
  ]);
  await changeRole(api, admin.access_token, slug, wandaId, 20);
  // UI and Docs at the top, at 65535 and 75535, without a colour
  const ui = await createLabel(api, admin.access_token, web.id, { name: "UI" });
  const docs = await createLabel(api, admin.access_token, web.id, { name: "Docs" });
  const listed = `/api/v0/projects/${web.id}/labels`;

  const page = await signedInPage(admin);
  const watch = await watchPage(page);
  await page.goto(`/${slug}/settings/projects/${web.id}/labels`);
  await expect(page.locator("h6")).toHaveText(["UI", "Docs"]);

  // Bug: the form opens on the name (its effect, which may run once the form shows), and Tab goes through it in its
  // order; the form waits for nerve, its buttons disabled, Cancel too; it closes once nerve has made the label, at the
  // top, last of the project's. Feature after it.
  await page.getByRole("button", { name: "Add label" }).click();
  await expect(page.locator("#labelName")).toBeFocused();
  await page.locator("#labelName").fill("Bug");
  await labelForm(page).getByRole("button").first().focus();
  expect(await tabCycle(page, labelForm(page), 3)).toEqual(["labelName", "Cancel", "Add"]);
  const creation = await sentHeld(page, "POST", listed, () =>
    labelForm(page).getByRole("button", { name: "Add" }).click()
  );
  expect(creation.body).toEqual({ name: "Bug", color: expect.stringMatching(/^#/) });
  expect(await enabledWithin(labelForm(page).getByRole("button", { name: "Adding" }))).toBe(false);
  expect(await enabledWithin(labelForm(page).getByRole("button", { name: "Cancel" }))).toBe(false);
  expect((await creation.release()).status()).toBe(201);
  await expect(page.locator("h6")).toHaveText(["UI", "Docs", "Bug"]);
  await expect(labelForm(page)).toHaveCount(0);
  await page.getByRole("button", { name: "Add label" }).click();
  await page.locator("#labelName").fill("Feature");
  const created = await sentTo(page, "POST", listed, () =>
    labelForm(page).getByRole("button", { name: "Add" }).click()
  );
  expect(created.answer.status()).toBe(201);

  // bug: said under the name (the line after the form), the form open with what it holds.
  await page.getByRole("button", { name: "Add label" }).click();
  await page.locator("#labelName").fill("bug");
  const taken = await sentTo(page, "POST", listed, () => labelForm(page).getByRole("button", { name: "Add" }).click());
  expect(taken.answer.status()).toBe(409);
  await expect(labelForm(page).locator("xpath=following-sibling::p[1]")).toHaveText(
    "A label of this project already has this name, in this case or another."
  );
  await expect(page.locator("#labelName")).toHaveValue("bug");
  await labelForm(page).getByRole("button", { name: "Cancel" }).click();
  const named = await labelsOf(api, admin.access_token, web.id);
  const [bug, feature] = [named("Bug"), named("Feature")];

  // UI dragged onto Bug's middle: under it, which has no label under it, so the page sends UI's parent alone.
  const under = await sentTo(page, "PATCH", `/api/v0/labels/${ui.id}`, () => dragLabel(page, "UI", "Bug"));
  expect([under.answer.status(), under.body]).toEqual([200, { parent_id: bug.id }]);
  await expect(page.locator("h6")).toHaveText(["Docs", "Bug", "UI", "Feature"]);

  // Bug renamed Defect, from its menu as a group: the fields its form edits.
  await labelRow(page, "Bug").getByRole("button", { name: "Options" }).click();
  await page.getByRole("menuitem", { name: "Edit label" }).click();
  await page.locator("#labelName").fill("Defect");
  const renamed = await sentTo(page, "PATCH", `/api/v0/labels/${bug.id}`, () =>
    labelForm(page).getByRole("button", { name: "Update" }).click()
  );
  expect([renamed.answer.status(), renamed.body]).toEqual([200, { name: "Defect", color: bug.color }]);
  await expect(page.locator("h6")).toHaveText(["Docs", "Defect", "UI", "Feature"]);

  // Feature dragged onto Docs's top: first of the labels at the top, a step before Docs.
  const first = await sentTo(page, "PATCH", `/api/v0/labels/${feature.id}`, () =>
    dragLabel(page, "Feature", "Docs", { x: 60, y: 2 })
  );
  expect([first.answer.status(), first.body]).toEqual([200, { parent_id: null, sort_order: 65535 }]);
  await expect(page.locator("h6")).toHaveText(["Feature", "Docs", "Defect", "UI"]);

  // Feature put under Defect meanwhile, elsewhere: Docs dropped onto it would make a third level, which nerve refuses
  // and the page says; Docs stays where it was.
  const elsewhere = await api.PATCH("/api/v0/labels/{label_id}", {
    params: { path: { label_id: feature.id } },
    body: { parent_id: bug.id },
    headers: bearer(admin.access_token),
  });
  expect(elsewhere.response.status).toBe(200);
  const third = await sentTo(page, "PATCH", `/api/v0/labels/${docs.id}`, () => dragLabel(page, "Docs", "Feature"));
  expect([third.answer.status(), third.body]).toEqual([422, { parent_id: feature.id }]);
  await expect(
    page.getByText("A label can go only under a label at the top of its project: labels have two levels.")
  ).toBeVisible();
  await expect(page.locator("h6")).toHaveText(["Feature", "Docs", "Defect", "UI"]);

  // Docs's controls, unseen while neither the pointer nor the focus is in its row (a click elsewhere hides them), are
  // shown while Tab goes through them, as while the pointer is over it.
  const docsRow = labelRow(page, "Docs");
  const docsControls = docsRow.locator(":scope > div.absolute");
  await docsRow.locator("h6").click();
  await page.mouse.move(0, 0);
  await expect(docsControls).toHaveCSS("opacity", "0");
  await docsRow.getByRole("button", { name: "Options" }).focus();
  await page.keyboard.press("Tab");
  await expect(docsRow.getByRole("button", { name: "Delete" })).toBeFocused();
  await expect(docsControls).toHaveCSS("opacity", "1");

  // Its member sees the labels, Defect's group among them, and no control that changes them, no grip, no label to drag;
  // a workspace admin who is its member has them all: a grip, a menu and a drag each, a delete button each but the
  // group.
  const shown = ["Docs", "Defect", "UI", "Feature"];
  const seenBy = async (who: string, tokens: typeof mem, controls: boolean) => {
    const other = await anotherBrowser(browser, baseURL ?? "", tokens);
    const otherWatch = await watchPage(other.page);
    await other.page.goto(`/${slug}/settings/projects/${web.id}/labels`);
    await expect(other.page.locator("h6"), who).toHaveText(shown);
    await expect(other.page.getByRole("button", { name: "Add label" }), who).toHaveCount(controls ? 1 : 0);
    await expect(other.page.locator("button.cursor-grab"), who).toHaveCount(controls ? 4 : 0);
    await expect(other.page.locator('[draggable="true"]:has(h6)'), who).toHaveCount(controls ? 4 : 0);
    await expect(other.page.getByRole("button", { name: "Options" }), who).toHaveCount(controls ? 4 : 0);
    await expect(other.page.getByRole("button", { name: "Delete" }), who).toHaveCount(controls ? 3 : 0);
    expect([otherWatch.apiFailures, otherWatch.pageErrors], who).toEqual([[], []]);
    await expectQuietConsole(other.page, otherWatch, { warnings: [EMOJI_CHECK_WARNING] });
    await other.close();
  };
  await seenBy("its member", mem, false);
  await seenBy("the workspace admin who is its member", wanda, true);

  // Reloaded, the page shows Feature under Defect, after UI, of the same place and made before it. Defect deleted from
  // its menu, its dialog held while nerve answers: Escape does not dismiss it, Cancel is disabled; it closes once nerve
  // has deleted Defect with the labels under it.
  await page.reload();
  await expect(page.locator("h6")).toHaveText(shown);
  await labelRow(page, "Defect").getByRole("button", { name: "Options" }).click();
  await page.getByRole("menuitem", { name: "Delete label" }).click();
  const dialog = page.getByRole("dialog");
  const deletion = await sentHeld(page, "DELETE", `/api/v0/labels/${bug.id}`, () =>
    dialog.getByRole("button", { name: "Delete" }).click()
  );
  expect(await closedByEscape(page)).toBe(false);
  expect(await enabledWithin(dialog.getByRole("button", { name: "Cancel" }))).toBe(false);
  expect((await deletion.release()).status()).toBe(204);
  expect(await closedWithin(page)).toBe(true);
  await expect(page.locator("h6")).toHaveText(["Docs"]);
  await expectLabels(db, web.id, [
    { name: "Feature", color: feature.color, parent: "Defect", sort_order: 65535, deleted: true, by: adminEmail },
    { name: "UI", color: "", parent: "Defect", sort_order: 65535, deleted: true, by: adminEmail },
    { name: "Docs", color: "", parent: null, sort_order: 75535, deleted: false, by: adminEmail },
    { name: "Defect", color: bug.color, parent: null, sort_order: 85535, deleted: true, by: adminEmail },
  ]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [`409 POST ${listed}`, `422 PATCH /api/v0/labels/${docs.id}`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING],
    errors: [
      "Failed to load resource: the server responded with a status of 409 (Conflict)",
      "Failed to load resource: the server responded with a status of 422 (Unprocessable Entity)",
    ],
  });
});
````

`e2e/stories/project/p7-labels.spec.ts`（修改，1 处）：

````old e2e/stories/project/p7-labels.spec.ts
// it, with what the names and the two levels keep. The page version comes
// with the labels' settings page (P11).
````
````new e2e/stories/project/p7-labels.spec.ts
// it, with what the names and the two levels keep. The page version is
// p7-labels-page.spec.ts.
````

`e2e/stories/smoke/s2-web-app.spec.ts`（修改，5 处）：

````old e2e/stories/smoke/s2-web-app.spec.ts
 * general page and its states.
````
````new e2e/stories/smoke/s2-web-app.spec.ts
 * general page, its states and its labels.
````
````old e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings/members", [...APP, ...WORKSPACE, ...INVITATIONS]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL]],
    ["/{slug}/settings/projects/{project}/states", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER]],
  ],
````
````new e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings/members", [...APP, ...WORKSPACE, ...INVITATIONS]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL]],
    ["/{slug}/settings/projects/{project}/states", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER]],
    ["/{slug}/settings/projects/{project}/labels", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER]],
  ],
````
````old e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings/members", [...APP, ...WORKSPACE]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL]],
    ["/{slug}/settings/projects/{project}/states", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER]],
  ],
  guest: [
````
````new e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings/members", [...APP, ...WORKSPACE]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL]],
    ["/{slug}/settings/projects/{project}/states", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER]],
    ["/{slug}/settings/projects/{project}/labels", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER]],
  ],
  guest: [
````
````old e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings/members", [...APP, ...WORKSPACE]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL]],
    ["/{slug}/settings/projects/{project}/states", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER]],
  ],
  "project non-member": [
````
````new e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings/members", [...APP, ...WORKSPACE]],
    ["/{slug}/settings/projects/{project}", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER, ...GENERAL]],
    ["/{slug}/settings/projects/{project}/states", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER]],
    ["/{slug}/settings/projects/{project}/labels", [...APP, ...WORKSPACE, ...PROJECT, ...PROJECT_MEMBER]],
  ],
  "project non-member": [
````
````old e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings/projects/{project}/states", [...APP, ...WORKSPACE, ...PROJECT]],
````
````new e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings/projects/{project}/states", [...APP, ...WORKSPACE, ...PROJECT]],
    ["/{slug}/settings/projects/{project}/labels", [...APP, ...WORKSPACE, ...PROJECT]],
````

`e2e/stories/workspace/w11-guest-bounds.spec.ts`（修改，3 处）：

````old e2e/stories/workspace/w11-guest-bounds.spec.ts
test("W11 (page): a guest signs in: his sidebar lists Web, the project he is a guest of, not Secret; the workspace's members page and Web's state settings tell him he may not see them; no page he opens shows an address, his own neither", async ({
````
````new e2e/stories/workspace/w11-guest-bounds.spec.ts
test("W11 (page): a guest signs in: his sidebar lists Web, the project he is a guest of, not Secret; the workspace's members page and Web's state and label settings tell him he may not see them; no page he opens shows an address, his own neither", async ({
````
````old e2e/stories/workspace/w11-guest-bounds.spec.ts
  await opened(`/${slug}/settings/projects/${web.id}/states`, denied);
````
````new e2e/stories/workspace/w11-guest-bounds.spec.ts
  await opened(`/${slug}/settings/projects/${web.id}/states`, denied);
  await opened(`/${slug}/settings/projects/${web.id}/labels`, denied);
````
````old e2e/stories/workspace/w11-guest-bounds.spec.ts
  await expectQuietConsole(page, watch, { warnings: Array(3).fill(EMOJI_CHECK_WARNING) });
````
````new e2e/stories/workspace/w11-guest-bounds.spec.ts
  await expectQuietConsole(page, watch, { warnings: Array(4).fill(EMOJI_CHECK_WARNING) });
````

- [ ] **Step 6: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 69 条规则、3 个例外，没有命中；web 的 oxlint 346 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 109 个全部通过。

- [ ] **Step 7: 提交**

```bash
git add .oxlintrc.json e2e/stories/project/p7-labels-page.spec.ts e2e/stories/project/p7-labels.spec.ts e2e/stories/smoke/s2-web-app.spec.ts e2e/stories/workspace/w11-guest-bounds.spec.ts tools/keywords.json 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/labels/page.tsx' 'web/apps/web/app/(all)/[workspaceSlug]/(settings)/settings/projects/[projectId]/states-labels.test.tsx' web/apps/web/core/components/labels/label-block/label-item-block.tsx web/apps/web/core/components/labels/label-drag-n-drop-HOC.tsx web/apps/web/core/components/labels/project-setting-label-group.test.tsx web/apps/web/core/components/labels/project-setting-label-group.tsx web/apps/web/core/components/labels/project-setting-label-list.tsx web/apps/web/core/store/label.store.changes.test.ts web/apps/web/core/store/label.store.test.ts web/apps/web/core/store/label.store.ts
```
```bash
git commit -m "feat(M3/P11): the label settings page reads its project from its route, offers the changes to whom nerve allows them, and a label is dragged by the grip it shows now

The page reads the workspace and the project from the route's params and
hands them to the list, which reads the project's labels by its id; the
keyword rule project-address-route takes the page, the list and the drag.
A group's menu and grip, and the empty state's button, show only to whom
may change the labels. A label's drag is registered by the grip its row
renders now, so a row rendered anew is draggable again. The row's menu
and delete button have names and show while the focus is in the row.
P7's page version; W11 and S2 open the page.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p11.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T5.1` | 标签设置页的门读 router store 的地址 | 关键词守卫、`states-labels.test.tsx` | 静态；vitest |
| `T5.2` | 项目的访客看得到标签 | `states-labels.test.tsx`、故事 W11 | vitest；端到端 |
| `T5.3` | 标签列表的控件按 router store 的地址判断 | 关键词守卫、`states-labels.test.tsx` | 静态；vitest |
| `T5.4` | 项目的成员得到改标签的控件 | `states-labels.test.tsx`、故事 P7 | vitest；端到端 |
| `T5.5` | 组的菜单和手柄给项目的每个成员（Plane 的写法） | `project-setting-label-group.test.tsx`、故事 P7 | vitest；端到端 |
| `T5.6` | 没有标签时的按钮给项目的每个成员 | `states-labels.test.tsx` | vitest |
| `T5.7` | 标签由整行开始拖动，没有手柄：每个成员都能拖 | 故事 P7 | 端到端 |
| `T5.8` | 标签的删除按钮没有名字 | 故事 P7 | 端到端 |
| `T5.9` | 标签的菜单没有名字 | 故事 P7 | 端到端 |
| `T5.10` | 标签一行的控件在 Tab 进到这一行时看不见 | 故事 P7 | 端到端 |
| `T5.11` | 标签的表单的按钮有正的 `tabIndex` | 故事 P7 | 端到端 |
| `T5.12` | 项目的标签树把组下的标签也放在顶层 | `label.store.test.ts` | vitest |
| `T5.13` | 在 `project-setting-label-group.test.tsx` 里写一个非空断言 | oxlint（`check:lint`） | 静态 |

---

### Task 6: 访客打开的个人主页；P9 的页面版本

**Files:**
- Create: `e2e/stories/project/p9-guest-profile.spec.ts`
- Modify: `e2e/stories/smoke/s2-web-app.spec.ts`

**Interfaces:**
- Produces（spec 2.6；M3 设计 2 的 P9，决策点 3 的 A，9.2 的那一格，3.1；M1-P2 交接）：
  - 页面不改（决策点 3 的 A："页面不变"）：访客打开工作区成员的个人主页（`/{slug}/profile/{id}`）或自己的，看到用户卡片（数据来自工作区的成员列表，P9 的 `useProfileMember`），工作项一栏说他没有权限（布局的门 `allowPermissions([ADMIN, MEMBER], WORKSPACE)`，访客不渲染 `Outlet`），页面不取工作项（M4 的旧地址 `user-issues`）。管理员和成员看到的工作项一栏还请求 Plane 的旧地址（404，M4 对接，3.1；S2 不打开它）。
  - S2：访客的挂载清单加上 `/{slug}/profile/{admin}`、`/{slug}/profile/{guest}`，都是 `APP`、`WORKSPACE`（spec 2.11）；`acme` 交回管理员和访客的 id，`{admin}`、`{guest}` 是它们的名字。
  - 访客离开工作区的界面、访客的个人主页上工作项的筛选栏：spec 第 3 节第 5、6 条（第 5 条待裁定）。

**Tests:**
- 端到端：`P9 (page): a guest opens a member's profile: its card, without the address, and in place of the work items that he may not see them, which the page does not ask for; his own profile the same`（新文件：Ada Lovelace（显示名 ada）是管理员，Gus Grant（gus）是访客；两张主页的地址都转到 `/assigned`，卡片的名字和显示名，"You do not have the permission to access this page."，没有 Assigned、Created、Subscribed 的链接，页面里没有任何人的邮箱；没有一个请求的地址含 `issues`，没有失败、没有 Plane 的旧地址、没有页面错误）。S2 的访客一条加两页。

- [ ] **Step 1: P9 的页面版本**

`e2e/stories/project/p9-guest-profile.spec.ts`（新文件，52 行）：

````file e2e/stories/project/p9-guest-profile.spec.ts
import { createWorkspace, inviteAndAccept, slugFor } from "../../fixtures/api";
import { accountId, bearer, emailFor } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import { registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// P9, a guest on a member's profile (M3 design 2, decision point 3, A: Plane's page): the card, and of the work items
// only that he may not see them, on anyone's profile, his own too. Page only: the work items' list is M4's, which tests
// its rule (13.2); the card's data is the members' list, which W11's API version covers. S2 holds what the page loads.

test("P9 (page): a guest opens a member's profile: its card, without the address, and in place of the work items that he may not see them, which the page does not ask for; his own profile the same", async ({
  api,
  signedInPage,
}, testInfo) => {
  const adaEmail = emailFor(testInfo, "ada");
  const gusEmail = emailFor(testInfo, "gus");
  const [ada, gus] = await Promise.all([registerOnboarded(api, adaEmail), registerOnboarded(api, gusEmail)]);
  const named = async (token: string, body: { first_name: string; last_name: string; display_name: string }) => {
    const { response } = await api.PATCH("/api/v0/me", { body, headers: bearer(token) });
    expect(response.status, `named ${body.display_name}`).toBe(200);
  };
  await named(ada.access_token, { first_name: "Ada", last_name: "Lovelace", display_name: "ada" });
  await named(gus.access_token, { first_name: "Gus", last_name: "Grant", display_name: "gus" });
  const slug = slugFor(testInfo);
  await createWorkspace(api, ada.access_token, { name: "Acme", slug });
  await inviteAndAccept(api, ada.access_token, slug, { email: gusEmail, token: gus.access_token }, 5);
  const [adaId, gusId] = await Promise.all([accountId(api, ada.access_token), accountId(api, gus.access_token)]);

  const page = await signedInPage(gus);
  const watch = await watchPage(page);
  /** Opens the profile of id, which goes to its first tab, and checks what it shows of him named name. */
  const profileOf = async (id: string, name: string, display: string) => {
    await page.goto(`/${slug}/profile/${id}`);
    await expect(page).toHaveURL(`/${slug}/profile/${id}/assigned`);
    await expect(page.getByRole("heading", { level: 4, name }), name).toBeVisible();
    await expect(page.getByText(`(${display})`, { exact: true }), name).toBeVisible();
    await expect(page.getByText("You do not have the permission to access this page."), name).toBeVisible();
    // no tab of the work items: the assigned, the created, the subscribed
    await expect(page.getByRole("link", { name: /^(Assigned|Created|Subscribed)$/ }), name).toHaveCount(0);
    const html = await page.content();
    expect([html.includes(adaEmail), html.includes(gusEmail)], `the addresses on ${name}'s profile`).toEqual([
      false,
      false,
    ]);
  };
  await profileOf(adaId, "Ada Lovelace", "ada");
  await profileOf(gusId, "Gus Grant", "gus");
  // Nothing asked of the work items, by nerve's API or by the old frontend's.
  expect(watch.apiRequests.filter((request) => request.includes("issues"))).toEqual([]);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING] });
});
````

- [ ] **Step 2: S2 的挂载清单**

`e2e/stories/smoke/s2-web-app.spec.ts`（修改，9 处）：

````old e2e/stories/smoke/s2-web-app.spec.ts
 * The pages each account opens, in order, each as its path ({slug} and {project} for their values) and the requests it
 * makes as it loads: "/", which lands on the workspace's home; the workspace's projects page, whose call of its own
 * fetch no other check holds; the workspace's general settings and its members; then the project's settings, its
 * general page, its states and its labels.
````
````new e2e/stories/smoke/s2-web-app.spec.ts
 * The pages each account opens, in order, each as its path ({slug}, {project}, {admin} and {guest} for their values:
 * the workspace's, the project's, the ids of its admin and its guest) and the requests it makes as it loads: "/", which
 * lands on the workspace's home; the workspace's projects page, whose call of its own fetch no other check holds; the
 * workspace's general settings and its members; the guest the admin's profile and his own, the card, from the members
 * the workspace's wrapper lists, and no work item (M3 design decision point 3, A; a member's profile lists his work
 * items, which are M4's); then the project's settings, its general page, its states and its labels.
````
````old e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings", [...APP, ...WORKSPACE]],
    ["/{slug}/settings/members", [...APP, ...WORKSPACE]],
````
````new e2e/stories/smoke/s2-web-app.spec.ts
    ["/{slug}/settings", [...APP, ...WORKSPACE]],
    ["/{slug}/settings/members", [...APP, ...WORKSPACE]],
    ["/{slug}/profile/{admin}", [...APP, ...WORKSPACE]],
    ["/{slug}/profile/{guest}", [...APP, ...WORKSPACE]],
````
````old e2e/stories/smoke/s2-web-app.spec.ts
 * each account, which have not been used in a browser yet.
````
````new e2e/stories/smoke/s2-web-app.spec.ts
 * each account, which have not been used in a browser yet, and the ids of its admin and its guest.
````
````old e2e/stories/smoke/s2-web-app.spec.ts
): Promise<{ slug: string; project: Project; tokens: Record<Account, AuthTokens> }> {
````
````new e2e/stories/smoke/s2-web-app.spec.ts
): Promise<{
  slug: string;
  project: Project;
  tokens: Record<Account, AuthTokens>;
  ids: { admin: string; guest: string };
}> {
````
````old e2e/stories/smoke/s2-web-app.spec.ts
  await join(outsider, 15);
````
````new e2e/stories/smoke/s2-web-app.spec.ts
  await join(outsider, 15);
  const ids = { admin: await accountId(api, token), guest: await accountId(api, guest.tokens.access_token) };
````
````old e2e/stories/smoke/s2-web-app.spec.ts
    { member_id: await accountId(api, guest.tokens.access_token), role: 5 },
````
````new e2e/stories/smoke/s2-web-app.spec.ts
    { member_id: ids.guest, role: 5 },
````
````old e2e/stories/smoke/s2-web-app.spec.ts
  return { slug, project, tokens };
````
````new e2e/stories/smoke/s2-web-app.spec.ts
  return { slug, project, tokens, ids };
````
````old e2e/stories/smoke/s2-web-app.spec.ts
    const { slug, project, tokens } = await acme(api, testInfo);
````
````new e2e/stories/smoke/s2-web-app.spec.ts
    const { slug, project, tokens, ids } = await acme(api, testInfo);
````
````old e2e/stories/smoke/s2-web-app.spec.ts
    const names = { [slug]: "{slug}", [project.id]: "{project}" };
````
````new e2e/stories/smoke/s2-web-app.spec.ts
    const names = { [slug]: "{slug}", [project.id]: "{project}", [ids.admin]: "{admin}", [ids.guest]: "{guest}" };
````

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 69 条规则、3 个例外，没有命中；web 的 oxlint 346 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 110 个全部通过。

- [ ] **Step 4: 提交**

```bash
git add e2e/stories/project/p9-guest-profile.spec.ts e2e/stories/smoke/s2-web-app.spec.ts
```
```bash
git commit -m "test(M3/P11): a guest opens a member's profile and his own: the card, no work items and no request for them

P9's page version: on a member's profile and his own, a guest sees the
card without anyone's address and, in place of the work items, that he
may not see them; the page asks for no work item. S2's guest opens both
profiles, which fetch what the app and the workspace fetch, and no more.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p11.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T6.1` | 访客得到个人主页的工作项一栏，它去取工作项 | 故事 P9、故事 S2 | 端到端 |
| `T6.2` | 访客算在看得到个人主页的工作项的人里 | 故事 P9 | 端到端 |

---

### Task 7: M2 的账户页面只在发出修改的会话里跟进

**Files:**
- Create: `web/apps/web/core/components/settings/profile/content/pages/general/form.test.tsx`、`web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.test.tsx`、`web/apps/web/core/components/settings/profile/content/pages/security.test.tsx`
- Modify: `.oxlintrc.json`、`web/apps/web/core/components/onboarding/steps/profile/root.test.tsx`、`web/apps/web/core/components/onboarding/steps/profile/root.tsx`、`web/apps/web/core/components/profile/start-of-week-preference.tsx`、`web/apps/web/core/components/settings/profile/content/pages/general/form.tsx`、`web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx`、`web/apps/web/core/components/settings/profile/content/pages/security.tsx`

**Interfaces:**
- Produces（spec 2.7；M3 设计 7.1；P9 的裁定 P8、P9 spec 第 5 节和 review 第 6 节的 P11 一行；M2 设计 7.3）：
  - 五个文件的修改经 `followInSession`：`done` 里提示成功（或交给下一步、清空表单），`failed` 里显示拒绝；说什么、怎样说照 M2（字段下的错误由各文件自己的映射决定，不改用 `useFormRefusal`：spec 第 3 节第 7 条）。
    - 新手引导的资料一步（`onboarding/steps/profile/root.tsx`）：名字的修改兑现之后交给下一步（`onDone`）；被拒绝时 `first_name` 的字段错误在名字下，其余提示（`needsErrorBanner`）；原来的 `handleSubmitUserDetail` 删除。
    - general（`general/form.tsx`）：成功提示"profile_updated"；被拒绝时 nerve 指名的字段错误在字段下，其余提示；按钮的加载在跟进之后结束。
    - preferences：时区和语言（`language-and-timezone-list.tsx`）、每周第一天（`start-of-week-preference.tsx`）：成功提示，被拒绝时提示。
    - security（`security.tsx`）：密码改了之后清空表单、隐藏密码、提示；`current_password_incorrect` 在旧密码下，`new_password` 的字段错误在新密码下，其余提示。
  - 另一个标签页换了账户之后才兑现的修改，页面什么都不做（不提示、不交给下一步、不清空表单、不在字段下写东西）。
  - 静态检查：`.oxlintrc.json` 的非空断言的范围加上三个新的测试文件（新手引导的测试已在里面）。

**Tests:**
- vitest：`onboarding/steps/profile/root.test.tsx`：`stays when nerve refuses the names, its refusal of the name said under it`；`says in a toast a refusal that is not of the name`；`hands nothing on and says nothing when the names' change $settles after another tab moved this one to another account`（替换原来的两条）。`general/form.test.tsx`（新文件）：`sends the names, the address not among them, and says nerve made the change`；`says a refusal under the field it names, and no toast`；`says in a toast a refusal that names no field of the form`；`says nothing when the change $settles after another tab moved this one to another account`。`security.test.tsx`（新文件）：`sends the passwords; once nerve has changed it, empties the form and says so`；`says nerve's refusal of $refused under its field, the form as it was`；`says in a toast a refusal of neither password`；`says nothing, nor empties the form, when the change $settles after another tab moved this one to another account`。`preferences/language-and-timezone-list.test.tsx`（新文件；列表渲染每周第一天）：`sends $preference picked, and says nerve made the change`；`says nerve's refusal of $preference in a toast`；`says nothing when the change of $preference $settles after another tab moved this one to another account`（时区、语言、每周第一天）。
- 端到端：没有新的；M2 的账户故事（A 系列）照旧通过。

- [ ] **Step 1: 新手引导的资料一步**

`web/apps/web/core/components/onboarding/steps/profile/root.test.tsx`（修改，10 处）：

````old web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
import { renderToStaticMarkup } from "react-dom/server";
````
````new web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
import { renderToStaticMarkup } from "react-dom/server";
import type { Mock } from "vitest";
````
````old web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
import { heldChange, pageSettled, settledYet, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
````
````new web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
import { heldChange, lateSettlings, pageSettled, settledYet, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal, underFields } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
````
````old web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
// When the onboarding's profile step hands the onboarding on (M3 design 7.1, 7.4): the step renders on the server with
// a stand-in for its form (react-hook-form's useForm), which keeps the step's submit; the test submits names as the
// form would, and nerve answers them when the test says. Its session is fake-tab.ts's. What the step itself says of a
// refusal is M2's (M2 design 7.3).
````
````new web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
// When the onboarding's profile step hands the onboarding on, and what it says of nerve's refusal of the names (M2
// design 7.3, M3 design 7.1, 7.4): the step renders on the server with a stand-in for its form (react-hook-form's
// useForm), which keeps the step's submit and what the step says under the name; the test submits names as the form
// would, and nerve answers them when the test says. Its session is fake-tab.ts's.
````
````old web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
  (): { submit: ((names: Names) => Promise<void>) | undefined; updateCurrentUser: ReturnType<typeof vi.fn> } => ({
````
````new web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
  (): {
    submit: ((names: Names) => Promise<void>) | undefined;
    updateCurrentUser: ReturnType<typeof vi.fn>;
    setError: Mock;
  } => ({
````
````old web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
    updateCurrentUser: vi.fn(),
````
````new web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
    updateCurrentUser: vi.fn(),
    setError: vi.fn(),
````
````old web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
    setError: () => undefined,
````
````new web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
    setError: page.setError,
````
````old web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
  page.updateCurrentUser.mockResolvedValue(undefined);
  onDone.mockReset();
````
````new web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
  page.updateCurrentUser.mockResolvedValue(undefined);
  page.setError.mockReset();
  onDone.mockReset();
  toasts.length = 0;
````
````old web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
  it("stays when nerve refuses the names", async () => {
````
````new web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
  it("stays when nerve refuses the names, its refusal of the name said under it", async () => {
````
````old web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
    await named();
    expect(onDone.mock.calls).toEqual([]);
````
````new web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
    await named();
    expect([onDone.mock.calls, underFields(page.setError), toasts]).toEqual([
      [],
      [["first_name", "errors.field.required"]],
      [],
    ]);
````
````old web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
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
````new web/apps/web/core/components/onboarding/steps/profile/root.test.tsx
  it("says in a toast a refusal that is not of the name", async () => {
    page.updateCurrentUser.mockRejectedValueOnce(refusal(403, "forbidden"));
    await named();
    expect([onDone.mock.calls, underFields(page.setError), toasts.map((toast) => toast.type)]).toEqual([
      [],
      [],
      ["error"],
    ]);
  });

  it.each(lateSettlings)(
    "hands nothing on and says nothing when the names' change $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const names = heldChange<undefined>();
      page.updateCurrentUser.mockReturnValueOnce(names.sent);
      const submitted = named();
      switchAccount();
      settle(names);
      await submitted;
      expect([onDone.mock.calls, page.setError.mock.calls, toasts]).toEqual([[], [], []]);
    }
  );
});
````

`web/apps/web/core/components/onboarding/steps/profile/root.tsx`（修改，2 处）：

````old web/apps/web/core/components/onboarding/steps/profile/root.tsx
  /** Saves the names; false when nerve did not, so the step stays (M2 design 7.1: nor for another account). */
  const handleSubmitUserDetail = async (formData: TProfileSetupFormValues): Promise<boolean> => {
````
````new web/apps/web/core/components/onboarding/steps/profile/root.tsx
  const onSubmit = async (formData: TProfileSetupFormValues) => {
    if (!user) return;
````
````old web/apps/web/core/components/onboarding/steps/profile/root.tsx
    try {
      await updateCurrentUser(userDetailsPayload);
      return true;
    } catch (error) {
      // A refusal of the name shows under it, anything else in a toast (M2 design 7.3).
      const key = fieldErrorKeys(error).first_name;
      if (key !== undefined) setError("first_name", { type: "manual", message: t(key) });
      if (needsErrorBanner(error, FIELDS)) toastRefusal(error);
      return false;
    }
  };

  const onSubmit = async (formData: TProfileSetupFormValues) => {
    if (!user) return;
    // the onboarding goes on only in the session the names were sent in (M3 design 7.1); what the step says of a
    // refusal is M2's (M3/P9 spec 5: P11)
    await followInSession(() => handleSubmitUserDetail(formData), {
      done: (named) => (named ? onDone() : undefined),
      // handleSubmitUserDetail says why itself, and answers false: it does not reject
      failed: () => undefined,
````
````new web/apps/web/core/components/onboarding/steps/profile/root.tsx
    // Only in the session the names were sent in (M3 design 7.1): the onboarding goes on once nerve has them; the step
    // stays when nerve refuses them, a refusal of the name shown under it, anything else in a toast (M2 design 7.3).
    await followInSession(() => updateCurrentUser(userDetailsPayload), {
      done: () => onDone(),
      failed: (error) => {
        const key = fieldErrorKeys(error).first_name;
        if (key !== undefined) setError("first_name", { type: "manual", message: t(key) });
        if (needsErrorBanner(error, FIELDS)) toastRefusal(error);
      },
````

- [ ] **Step 2: 个人设置的 general、preferences、security**

`web/apps/web/core/components/profile/start-of-week-preference.tsx`（修改，2 处）：

````old web/apps/web/core/components/profile/start-of-week-preference.tsx
import { useRefusalToast } from "@/hooks/use-refusal-toast";
````
````new web/apps/web/core/components/profile/start-of-week-preference.tsx
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
````
````old web/apps/web/core/components/profile/start-of-week-preference.tsx
  const handleStartOfWeekChange = async (val: StartOfTheWeek) => {
    try {
      await updateUserProfile({ start_of_the_week: val });
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("toast.success"),
        message: t("power_k.preferences_actions.toast.generic.success"),
      });
    } catch (error) {
      toastRefusal(error);
    }
  };
````
````new web/apps/web/core/components/profile/start-of-week-preference.tsx
  // It says how nerve settled the change only in the session it was sent in (M3 design 7.1).
  const handleStartOfWeekChange = (val: StartOfTheWeek) =>
    followInSession(() => updateUserProfile({ start_of_the_week: val }), {
      done: () =>
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("toast.success"),
          message: t("power_k.preferences_actions.toast.generic.success"),
        }),
      failed: toastRefusal,
    });
````

`web/apps/web/core/components/settings/profile/content/pages/general/form.test.tsx`（新文件，117 行）：

````file web/apps/web/core/components/settings/profile/content/pages/general/form.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import type { Mock } from "vitest";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { User } from "@nerve/api-client";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal, underFields } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { GeneralProfileSettingsForm } from "./form";

// What the profile settings' general form says of how nerve settled the change of the names (M2 design 7.3, M3 design
// 7.1): the form renders on the server with a stand-in for its form (react-hook-form's useForm), which keeps the form's
// submit and what it says under its fields; the test submits the names as the form would, and nerve answers them when
// the test says. Its session is fake-tab.ts's.

/** What the form's fields hold, as it gives its submit. */
type Names = { first_name: string; last_name: string; display_name: string; email: string };

const form = vi.hoisted(
  (): {
    submit: ((names: Names) => Promise<void>) | undefined;
    setError: Mock;
    updateCurrentUser: Mock;
  } => ({ submit: undefined, setError: vi.fn(), updateCurrentUser: vi.fn() })
);
vi.mock("react-hook-form", () => ({
  useForm: () => ({
    handleSubmit: (submit: (names: Names) => Promise<void>) => {
      form.submit = submit;
      return () => undefined;
    },
    control: {},
    watch: () => "",
    setError: form.setError,
    formState: { errors: {} },
  }),
  Controller: () => null,
}));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => ({ data: undefined, updateCurrentUser: form.updateCurrentUser }),
}));
vi.mock("@/components/account/deactivate-account-modal", () => ({ DeactivateAccountModal: () => null }));
vi.mock("@/components/common/cover-image", () => ({ CoverImage: () => null }));
vi.mock("@/components/settings/boxed-control-item", () => ({ SettingsBoxedControlItem: () => null }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

const ada: User = {
  id: "u-ada",
  email: "ada@example.com",
  first_name: "Ada",
  last_name: "Lovelace",
  display_name: "ada",
  user_timezone: "UTC",
  avatar_url: null,
  cover_image_url: null,
  created_at: "2026-01-01T00:00:00Z",
};

/** Renders the form, and submits new names as it would; settles once the form has. */
async function renamed() {
  renderToStaticMarkup(<GeneralProfileSettingsForm user={ada} />);
  if (!form.submit) throw new Error("the page showed no form");
  await form.submit({ first_name: "Augusta", last_name: "King", display_name: "augusta", email: ada.email });
}

beforeEach(() => {
  signedIn();
  form.submit = undefined;
  form.setError.mockReset();
  form.updateCurrentUser.mockReset();
  form.updateCurrentUser.mockResolvedValue(ada);
  toasts.length = 0;
});

describe("GeneralProfileSettingsForm", () => {
  it("sends the names, the address not among them, and says nerve made the change", async () => {
    await renamed();
    expect(form.updateCurrentUser.mock.calls).toEqual([
      [{ first_name: "Augusta", last_name: "King", display_name: "augusta" }],
    ]);
    expect(toasts).toEqual([{ type: "success", title: "toast.success", message: "profile_updated" }]);
  });

  it("says a refusal under the field it names, and no toast", async () => {
    form.updateCurrentUser.mockRejectedValueOnce(
      refusal(422, "validation_failed", [{ field: "display_name", code: "too_long" }])
    );
    await renamed();
    expect([underFields(form.setError), toasts]).toEqual([[["display_name", "errors.field.too_long"]], []]);
  });

  it("says in a toast a refusal that names no field of the form", async () => {
    form.updateCurrentUser.mockRejectedValueOnce(refusal(403, "forbidden"));
    await renamed();
    expect([underFields(form.setError), toasts.map((toast) => toast.type)]).toEqual([[], ["error"]]);
  });

  it.each(lateSettlings)(
    "says nothing when the change $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const change = heldChange<undefined>();
      form.updateCurrentUser.mockReturnValueOnce(change.sent);
      const submitted = renamed();
      switchAccount();
      settle(change);
      await submitted;
      expect([form.setError.mock.calls, toasts]).toEqual([[], []]);
    }
  );
});
````

`web/apps/web/core/components/settings/profile/content/pages/general/form.tsx`（修改，2 处）：

````old web/apps/web/core/components/settings/profile/content/pages/general/form.tsx
import { fieldErrorKeys, needsErrorBanner } from "@/lib/error-messages";
````
````new web/apps/web/core/components/settings/profile/content/pages/general/form.tsx
import { fieldErrorKeys, needsErrorBanner } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";
````
````old web/apps/web/core/components/settings/profile/content/pages/general/form.tsx
    try {
      await updateCurrentUser(userPayload);
      setToast({ type: TOAST_TYPE.SUCCESS, title: t("toast.success"), message: t("profile_updated") });
    } catch (error) {
      // A refusal shows under the field it is about, anything else in a toast (M2 design 7.3).
      const fields = fieldErrorKeys(error);
      for (const field of FIELDS) {
        const key = fields[field];
        if (key !== undefined) setError(field, { type: "manual", message: t(key) });
      }
      if (needsErrorBanner(error, FIELDS)) toastRefusal(error);
    } finally {
      setIsLoading(false);
    }
````
````new web/apps/web/core/components/settings/profile/content/pages/general/form.tsx
    // Only in the session the change was sent in (M3 design 7.1): the form says nerve made it; a refusal shows under
    // the field it is about, anything else in a toast (M2 design 7.3).
    await followInSession(() => updateCurrentUser(userPayload), {
      done: () => setToast({ type: TOAST_TYPE.SUCCESS, title: t("toast.success"), message: t("profile_updated") }),
      failed: (error) => {
        const fields = fieldErrorKeys(error);
        for (const field of FIELDS) {
          const key = fields[field];
          if (key !== undefined) setError(field, { type: "manual", message: t(key) });
        }
        if (needsErrorBanner(error, FIELDS)) toastRefusal(error);
      },
    });
    setIsLoading(false);
````

`web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.test.tsx`（新文件，113 行）：

````file web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ReactNode } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import type { Mock } from "vitest";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, pageSettled, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { emptyShown, shown } from "@/lib/fake-controls";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { ProfileSettingsLanguageAndTimezonePreferencesList } from "./language-and-timezone-list";

// What the profile settings' preferences say of how nerve settled a change (M2 design 7.3, M3 design 7.1): the time
// zone, the language and the start of the week (StartOfWeekPreference, which the list renders). The list renders on
// the server with stand-ins for its selects, which keep what each gives the page; the test picks a value as a select
// would, and nerve answers when the test says. Its session is fake-tab.ts's.

const page = vi.hoisted(
  (): {
    timezone: ((value: string) => unknown) | undefined;
    updateCurrentUser: Mock;
    updateUserProfile: Mock;
  } => ({ timezone: undefined, updateCurrentUser: vi.fn(), updateUserProfile: vi.fn() })
);
vi.mock("@/components/global", () => ({
  TimezoneSelect: ({ onChange }: { onChange: (value: string) => unknown }) => {
    page.timezone = onChange;
    return null;
  },
}));
vi.mock("@/components/settings/control-item", () => ({
  SettingsControlItem: ({ control }: { control: ReactNode }) => control,
}));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => ({ data: undefined, updateCurrentUser: page.updateCurrentUser, userProfile: { data: undefined } }),
  useUserProfile: () => ({ data: undefined, updateUserProfile: page.updateUserProfile }),
}));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/ui", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", async (importOriginal) => ({
  ...(await importOriginal<object>()),
  ...(await import("@/lib/fake-i18n")),
}));

/** The list's three preferences: how a select picks a value, what the page sends, and what it says of success. */
const preferences = [
  {
    preference: "the time zone",
    pick: () => page.timezone?.("Asia/Shanghai"),
    sent: () => page.updateCurrentUser,
    body: { user_timezone: "Asia/Shanghai" },
    said: "power_k.preferences_actions.toast.timezone.success",
  },
  {
    preference: "the language",
    pick: () => shown.selects[0]?.onChange("zh-CN"),
    sent: () => page.updateUserProfile,
    body: { language: "zh-CN" },
    said: "power_k.preferences_actions.toast.generic.success",
  },
  {
    preference: "the start of the week",
    pick: () => shown.selects[1]?.onChange(1),
    sent: () => page.updateUserProfile,
    body: { start_of_the_week: 1 },
    said: "power_k.preferences_actions.toast.generic.success",
  },
];

beforeEach(() => {
  signedIn();
  emptyShown();
  page.timezone = undefined;
  page.updateCurrentUser.mockReset();
  page.updateCurrentUser.mockResolvedValue(undefined);
  page.updateUserProfile.mockReset();
  page.updateUserProfile.mockResolvedValue(undefined);
  toasts.length = 0;
  renderToStaticMarkup(<ProfileSettingsLanguageAndTimezonePreferencesList />);
});

describe("ProfileSettingsLanguageAndTimezonePreferencesList", () => {
  it.each(preferences)("sends $preference picked, and says nerve made the change", async (row) => {
    row.pick();
    await pageSettled();
    expect(row.sent().mock.calls).toEqual([[row.body]]);
    expect(toasts).toEqual([{ type: "success", title: "toast.success", message: row.said }]);
  });

  it.each(preferences)("says nerve's refusal of $preference in a toast", async (row) => {
    row.sent().mockRejectedValueOnce(refusal(403, "forbidden"));
    row.pick();
    await pageSettled();
    expect(toasts.map((toast) => toast.type)).toEqual(["error"]);
  });

  it.each(preferences.flatMap((row) => lateSettlings.map((late) => Object.assign({}, row, late))))(
    "says nothing when the change of $preference $settles after another tab moved this one to another account",
    async (row) => {
      const change = heldChange<undefined>();
      row.sent().mockReturnValueOnce(change.sent);
      row.pick();
      switchAccount();
      row.settle(change);
      await pageSettled();
      expect([row.sent().mock.calls.length, toasts]).toEqual([1, []]);
    }
  );
});
````

`web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx`（修改，3 处）：

````old web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx
import { useRefusalToast } from "@/hooks/use-refusal-toast";
````
````new web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
````
````old web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx
    const handleTimezoneChange = async (value: string) => {
      try {
        await updateCurrentUser({ user_timezone: value });
        setToast({
          title: t("toast.success"),
          message: t("power_k.preferences_actions.toast.timezone.success"),
          type: TOAST_TYPE.SUCCESS,
        });
      } catch (error) {
        toastRefusal(error);
      }
    };
````
````new web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx
    // Each says how nerve settled the change only in the session it was sent in (M3 design 7.1).
    const handleTimezoneChange = (value: string) =>
      followInSession(() => updateCurrentUser({ user_timezone: value }), {
        done: () =>
          setToast({
            title: t("toast.success"),
            message: t("power_k.preferences_actions.toast.timezone.success"),
            type: TOAST_TYPE.SUCCESS,
          }),
        failed: toastRefusal,
      });
````
````old web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx
    const handleLanguageChange = async (value: Language) => {
      try {
        await updateUserProfile({ language: value });
        setToast({
          title: t("toast.success"),
          message: t("power_k.preferences_actions.toast.generic.success"),
          type: TOAST_TYPE.SUCCESS,
        });
      } catch (error) {
        toastRefusal(error);
      }
    };
````
````new web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx
    const handleLanguageChange = (value: Language) =>
      followInSession(() => updateUserProfile({ language: value }), {
        done: () =>
          setToast({
            title: t("toast.success"),
            message: t("power_k.preferences_actions.toast.generic.success"),
            type: TOAST_TYPE.SUCCESS,
          }),
        failed: toastRefusal,
      });
````

`web/apps/web/core/components/settings/profile/content/pages/security.test.tsx`（新文件，114 行）：

````file web/apps/web/core/components/settings/profile/content/pages/security.test.tsx
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { renderToStaticMarkup } from "react-dom/server";
import type { Mock } from "vitest";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { heldChange, lateSettlings, signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal, underFields } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { SecurityProfileSettings } from "./security";

// What the security page says of how nerve settled the change of the password (M2 design 7.3, M3 design 7.1): the
// page renders on the server with a stand-in for its form (react-hook-form's useForm), which keeps the form's submit,
// what it says under its fields and its emptying; the test submits the passwords as the form would, and nerve answers
// them when the test says. Its session is fake-tab.ts's.

/** What the form's fields hold, as it gives its submit. */
type Passwords = { old_password: string; new_password: string; confirm_password: string };

const form = vi.hoisted(
  (): {
    submit: ((passwords: Passwords) => Promise<void>) | undefined;
    setError: Mock;
    reset: Mock;
    changePassword: Mock;
  } => ({ submit: undefined, setError: vi.fn(), reset: vi.fn(), changePassword: vi.fn() })
);
vi.mock("react-hook-form", () => ({
  useForm: () => ({
    handleSubmit: (submit: (passwords: Passwords) => Promise<void>) => {
      form.submit = submit;
      return () => undefined;
    },
    control: {},
    watch: () => "",
    setError: form.setError,
    reset: form.reset,
    formState: { errors: {}, isSubmitting: false },
  }),
  Controller: () => null,
}));
vi.mock("@/hooks/store/user", () => ({ useUser: () => ({ changePassword: form.changePassword }) }));
vi.mock("@/components/api-token/token-list", () => ({ ApiTokenList: () => null }));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@nerve/ui", () => ({ PasswordStrengthIndicator: () => null }));
vi.mock("@nerve/propel/button", () => import("@/lib/fake-controls"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));

/** Renders the page, and submits the passwords as its form would; settles once the page has. */
async function changed() {
  renderToStaticMarkup(<SecurityProfileSettings />);
  if (!form.submit) throw new Error("the page showed no form");
  await form.submit({ old_password: "old secret", new_password: "new secret", confirm_password: "new secret" });
}

beforeEach(() => {
  signedIn();
  form.submit = undefined;
  form.setError.mockReset();
  form.reset.mockReset();
  form.changePassword.mockReset();
  form.changePassword.mockResolvedValue(undefined);
  toasts.length = 0;
});

describe("SecurityProfileSettings", () => {
  it("sends the passwords; once nerve has changed it, empties the form and says so", async () => {
    await changed();
    expect(form.changePassword.mock.calls).toEqual([[{ current_password: "old secret", new_password: "new secret" }]]);
    expect([form.reset.mock.calls.length, toasts.map((toast) => toast.message)]).toEqual([
      1,
      ["auth.common.password.toast.change_password.success.message"],
    ]);
  });

  it.each([
    {
      refused: "the current password, wrong",
      error: refusal(403, "identity.current_password_incorrect"),
      under: [["old_password", "errors.current_password_incorrect"]],
    },
    {
      refused: "the new password",
      error: refusal(422, "validation_failed", [{ field: "new_password", code: "weak_password" }]),
      under: [["new_password", "errors.field.weak_password"]],
    },
  ])("says nerve's refusal of $refused under its field, the form as it was", async ({ error, under }) => {
    form.changePassword.mockRejectedValueOnce(error);
    await changed();
    expect([underFields(form.setError), form.reset.mock.calls, toasts]).toEqual([under, [], []]);
  });

  it("says in a toast a refusal of neither password", async () => {
    form.changePassword.mockRejectedValueOnce(refusal(403, "forbidden"));
    await changed();
    expect([underFields(form.setError), toasts.map((toast) => toast.type)]).toEqual([[], ["error"]]);
  });

  it.each(lateSettlings)(
    "says nothing, nor empties the form, when the change $settles after another tab moved this one to another account",
    async ({ settle }) => {
      const change = heldChange<undefined>();
      form.changePassword.mockReturnValueOnce(change.sent);
      const submitted = changed();
      switchAccount();
      settle(change);
      await submitted;
      expect([form.setError.mock.calls, form.reset.mock.calls, toasts]).toEqual([[], [], []]);
    }
  );
});
````

`web/apps/web/core/components/settings/profile/content/pages/security.tsx`（修改，2 处）：

````old web/apps/web/core/components/settings/profile/content/pages/security.tsx
import { errorMessageKey, fieldErrorKeys } from "@/lib/error-messages";
````
````new web/apps/web/core/components/settings/profile/content/pages/security.tsx
import { errorMessageKey, fieldErrorKeys } from "@/lib/error-messages";
import { followInSession } from "@/lib/in-session";
````
````old web/apps/web/core/components/settings/profile/content/pages/security.tsx
    try {
      await changePassword({ current_password: old_password, new_password });

      reset(defaultValues);
      setShowPassword(defaultShowPassword);
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("auth.common.password.toast.change_password.success.title"),
        message: t("auth.common.password.toast.change_password.success.message"),
      });
    } catch (error: unknown) {
      // A refusal shows under the field it is about, anything else in a toast (M2 design 7.3).
      if (error instanceof ApiError && error.problem?.code === "identity.current_password_incorrect") {
        setError("old_password", { type: "manual", message: t(errorMessageKey(error)) });
        return;
      }
      const newPassword = fieldErrorKeys(error).new_password;
      if (newPassword !== undefined) {
        setError("new_password", { type: "manual", message: t(newPassword) });
        return;
      }
      toastRefusal(error);
    }
````
````new web/apps/web/core/components/settings/profile/content/pages/security.tsx
    // Only in the session the change was sent in (M3 design 7.1): the form empties and says nerve made it; a refusal
    // shows under the field it is about, anything else in a toast (M2 design 7.3).
    await followInSession(() => changePassword({ current_password: old_password, new_password }), {
      done: () => {
        reset(defaultValues);
        setShowPassword(defaultShowPassword);
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("auth.common.password.toast.change_password.success.title"),
          message: t("auth.common.password.toast.change_password.success.message"),
        });
      },
      failed: (error) => {
        if (error instanceof ApiError && error.problem?.code === "identity.current_password_incorrect") {
          setError("old_password", { type: "manual", message: t(errorMessageKey(error)) });
          return;
        }
        const newPassword = fieldErrorKeys(error).new_password;
        if (newPassword !== undefined) {
          setError("new_password", { type: "manual", message: t(newPassword) });
          return;
        }
        toastRefusal(error);
      },
    });
````

- [ ] **Step 3: 静态检查的范围**

`.oxlintrc.json`（修改，1 处）：

````old .oxlintrc.json
        "web/apps/web/core/components/labels/project-setting-label-group.test.tsx"
````
````new .oxlintrc.json
        "web/apps/web/core/components/labels/project-setting-label-group.test.tsx",
        "web/apps/web/core/components/settings/profile/content/pages/general/form.test.tsx",
        "web/apps/web/core/components/settings/profile/content/pages/security.test.tsx",
        "web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.test.tsx"
````

- [ ] **Step 4: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 69 条规则、3 个例外，没有命中；web 的 oxlint 346 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 110 个全部通过。

- [ ] **Step 5: 提交**

```bash
git add .oxlintrc.json web/apps/web/core/components/onboarding/steps/profile/root.test.tsx web/apps/web/core/components/onboarding/steps/profile/root.tsx web/apps/web/core/components/profile/start-of-week-preference.tsx web/apps/web/core/components/settings/profile/content/pages/general/form.test.tsx web/apps/web/core/components/settings/profile/content/pages/general/form.tsx web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.test.tsx web/apps/web/core/components/settings/profile/content/pages/preferences/language-and-timezone-list.tsx web/apps/web/core/components/settings/profile/content/pages/security.test.tsx web/apps/web/core/components/settings/profile/content/pages/security.tsx
```
```bash
git commit -m "feat(M3/P11): M2's account pages say how nerve settled a change only in the session it was sent in

The onboarding's profile step, the profile settings' general form, time
zone, language, start of the week and password follow each change with
followInSession: what they say and where stays M2's, and a change that
settles after another tab moved this one to another account says nothing,
hands nothing on and empties no form.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p11.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T7.1` | 新手引导的资料一步不论会话都跟进 | `root.test.tsx` | vitest |
| `T7.2` | 个人设置的 general 不论会话都跟进 | `form.test.tsx` | vitest |
| `T7.3` | 时区和语言不论会话都跟进 | `language-and-timezone-list.test.tsx` | vitest |
| `T7.4` | 每周第一天不论会话都跟进 | `language-and-timezone-list.test.tsx` | vitest |
| `T7.5` | 修改密码不论会话都跟进 | `security.test.tsx` | vitest |
| `T7.6` | 在 `general/form.test.tsx` 里写一个非空断言 | oxlint（`check:lint`） | 静态 |

---

### Task 8: M3 的死成员和死 prop；P8a、P8b 留下的文件的 oxlint 警告

**Files:**
- Modify: `web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/header.tsx`、`web/apps/web/core/components/core/sidebar/progress-stats/state_group.tsx`、`web/apps/web/core/components/dropdowns/member/avatar.tsx`、`web/apps/web/core/components/dropdowns/project/base.tsx`、`web/apps/web/core/components/dropdowns/project/dropdown.tsx`、`web/apps/web/core/components/dropdowns/state/state-option.tsx`、`web/apps/web/core/components/power-k/menus/members.tsx`、`web/apps/web/core/components/project/create/common-attributes.tsx`、`web/apps/web/core/components/settings/workspace/sidebar/root.tsx`、`web/apps/web/core/components/workspace/invite-modal/actions.tsx`、`web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx`、`web/apps/web/core/store/member/workspace/workspace-member.store.ts`、`web/apps/web/package.json`、`web/packages/constants/src/members.ts`、`web/packages/propel/src/icons/state/dashed-circle.tsx`

**Interfaces:**
- Produces（spec 2.8、附录 A.5、A.9；M3 设计 7.9；M1 收尾交接；P8a、P8b spec 附录 A.8、A.9 和 review 第 6 节的 P11 一行；P8b 的裁定 P1）：
  - 删掉的都没有读者、没有调用方传入（spec 附录 A.5 的 C-b：`tsc`、knip 通过，`?url` 资源、包的再导出、文案、测试中都没有它们）；没有调用方传的 prop 原来取默认值，组件现在把默认值写成常量，显示的照旧：
    - 项目下拉框（`dropdowns/project/dropdown.tsx`、`base.tsx`）：`button`、`currentProjectId`、`dropdownArrow`、`dropdownArrowClassName`、`onClose`、`renderByDefault`；`getProjectById` 的类型是生成的 `Project`（返回 `Project | undefined`，P8b 的裁定 P1）；`base.tsx` 的五条 oxlint 警告清零（`no-shadow` 三条：取显示名和图标的函数不再收参数；`consistent-function-scoping`：画图标的函数移到模块；`ComboDropDown` 照 `@nerve/ui` 的说明收 `role="presentation"`：它只收集里面的部分的按键）。
    - 成员头像（`dropdowns/member/avatar.tsx`）的 `size`（固定 `"xs"`）；状态选项（`dropdowns/state/state-option.tsx`）的 `query`；命令面板的成员菜单（`power-k/menus/members.tsx`）的 `heading`；创建项目的公共字段（`project/create/common-attributes.tsx`）的 `handleFormOnChange`；工作区设置的侧边栏（`settings/workspace/sidebar/root.tsx`）的 props（只有 `className`，没有人传）；邀请弹窗的按钮行（`workspace/invite-modal/actions.tsx`）只留 `isSubmitting`、`handleClose`、`appendField`；状态图标的虚线圆（`propel` 的 `dashed-circle.tsx`）的 `totalSegments`（固定 15）；归档页头（`archives/header.tsx`）每个标签的 `href`（图标的类型写成 `React.FC<React.SVGAttributes<SVGElement>>`）。
    - 工作区成员 store 的 `memberMap`（接口、`computed`、getter）；成员列（`@nerve/constants` 的 `members.ts`）的 `iconName`（五处）。
  - P8a、P8b 留下的只经机械步骤到达的文件（spec 附录 A.5）：`use-work-item-filters-config.tsx` 的三条 `no-shadow`（回调的参数改名 `id`、`listed`、`option`）；`progress-stats/state_group.tsx` 的 `no-array-index-key`（`key={group.state}`：每组一行，`state` 不重复）。它们的 `promise(always-return)` 在 Task 9。
  - web 的 oxlint 上限 346 → 337。
  - `--rows M3` 留下的行（spec 附录 A.9）：每一行写明谁在读或传它（测试中 `it.each` 的列和 `vi.hoisted` 的桩、经展开或第三方库读取的成员、M4 的搜索类型），按路径归到 M3、实际属于 M4、M6、M7 的 41 行列给收尾。

**Tests:** 没有新的测试：删除的代码没有读者（`tsc`、knip），组件按原来的默认值显示（spec 附录 A.5 逐个说明）；oxlint 的改法由上限守着（本 Task 的变异表）。

- [ ] **Step 1: 项目下拉框**

`web/apps/web/core/components/dropdowns/project/base.tsx`（修改，18 处）：

````old web/apps/web/core/components/dropdowns/project/base.tsx

import type { ReactNode } from "react";
````
````new web/apps/web/core/components/dropdowns/project/base.tsx

````
````old web/apps/web/core/components/dropdowns/project/base.tsx
import { ChevronDownOutline, ProjectsOutline, SearchOutline, TickOutline } from "@makeplane/propel/icons";
````
````new web/apps/web/core/components/dropdowns/project/base.tsx
import { ProjectsOutline, SearchOutline, TickOutline } from "@makeplane/propel/icons";
````
````old web/apps/web/core/components/dropdowns/project/base.tsx
  button?: ReactNode;
  currentProjectId?: string;
  dropdownArrow?: boolean;
  dropdownArrowClassName?: string;
  getProjectById: (projectId: string | null | undefined) => Partial<Project> | undefined;
  onClose?: () => void;
  projectIds: string[];
  renderByDefault?: boolean;
````
````new web/apps/web/core/components/dropdowns/project/base.tsx
  getProjectById: (projectId: string | null | undefined) => Project | undefined;
  projectIds: string[];
````
````old web/apps/web/core/components/dropdowns/project/base.tsx
  );

export const ProjectDropdownBase = observer(function ProjectDropdownBase(props: Props) {
  const {
    button,
````
````new web/apps/web/core/components/dropdowns/project/base.tsx
  );

/** A project's logo, as the dropdown's button shows it. */
const renderProjectLogo = (logoProps: Project["logo_props"]) => (
  <span className="grid h-4 w-4 flex-shrink-0 place-items-center">
    <Logo logo={logoProps} size={14} />
  </span>
);

export const ProjectDropdownBase = observer(function ProjectDropdownBase(props: Props) {
  const {
````
````old web/apps/web/core/components/dropdowns/project/base.tsx
    className = "",
    currentProjectId,
    disabled = false,
    dropdownArrow = false,
    dropdownArrowClassName = "",
````
````new web/apps/web/core/components/dropdowns/project/base.tsx
    className = "",
    disabled = false,
````
````old web/apps/web/core/components/dropdowns/project/base.tsx
    onChange,
    onClose,
````
````new web/apps/web/core/components/dropdowns/project/base.tsx
    onChange,
````
````old web/apps/web/core/components/dropdowns/project/base.tsx
    projectIds,
    renderByDefault = true,
````
````new web/apps/web/core/components/dropdowns/project/base.tsx
    projectIds,
````
````old web/apps/web/core/components/dropdowns/project/base.tsx
    (query === ""
      ? options?.filter((o) => o?.value !== currentProjectId)
      : options?.filter((o) => o?.value !== currentProjectId && o?.query.toLowerCase().includes(query.toLowerCase()))
    )?.filter((o): o is NonNullable<typeof o> => o !== undefined),
````
````new web/apps/web/core/components/dropdowns/project/base.tsx
    (query === "" ? options : options?.filter((o) => o?.query.toLowerCase().includes(query.toLowerCase())))?.filter(
      (o): o is NonNullable<typeof o> => o !== undefined
    ),
````
````old web/apps/web/core/components/dropdowns/project/base.tsx
    isOpen,
    onClose,
````
````new web/apps/web/core/components/dropdowns/project/base.tsx
    isOpen,
````
````old web/apps/web/core/components/dropdowns/project/base.tsx
  const getDisplayName = (value: string | string[] | null, placeholder: string = "") => {
````
````new web/apps/web/core/components/dropdowns/project/base.tsx
  // what the button says: the project picked, how many are, or the placeholder
  const getDisplayName = () => {
````
````old web/apps/web/core/components/dropdowns/project/base.tsx
  const getProjectIcon = (value: string | string[] | null) => {
    const renderIcon = (logoProps: Project["logo_props"]) => (
      <span className="grid h-4 w-4 flex-shrink-0 place-items-center">
        <Logo logo={logoProps} size={14} />
      </span>
    );

````
````new web/apps/web/core/components/dropdowns/project/base.tsx
  // the logos of the projects picked
  const getProjectIcon = () => {
````
````old web/apps/web/core/components/dropdowns/project/base.tsx
              return projectDetails?.logo_props ? renderIcon(projectDetails.logo_props) : null;
````
````new web/apps/web/core/components/dropdowns/project/base.tsx
              return projectDetails?.logo_props ? renderProjectLogo(projectDetails.logo_props) : null;
````
````old web/apps/web/core/components/dropdowns/project/base.tsx
      const projectDetails = getProjectById(value);
      return projectDetails?.logo_props ? renderIcon(projectDetails.logo_props) : null;
````
````new web/apps/web/core/components/dropdowns/project/base.tsx
      const projectDetails = getProjectById(value);
      return projectDetails?.logo_props ? renderProjectLogo(projectDetails.logo_props) : null;
````
````old web/apps/web/core/components/dropdowns/project/base.tsx
  const comboButton = button ? (
    <button
      ref={setReferenceElement}
      type="button"
      className={cn("clickable block h-full w-full outline-none", buttonContainerClassName)}
      onClick={handleOnClick}
      disabled={disabled}
    >
      {button}
    </button>
  ) : (
````
````new web/apps/web/core/components/dropdowns/project/base.tsx
  const comboButton = (
````
````old web/apps/web/core/components/dropdowns/project/base.tsx
        variant={buttonVariant}
        renderToolTipByDefault={renderByDefault}
````
````new web/apps/web/core/components/dropdowns/project/base.tsx
        variant={buttonVariant}
````
````old web/apps/web/core/components/dropdowns/project/base.tsx
        {!hideIcon && getProjectIcon(value)}
````
````new web/apps/web/core/components/dropdowns/project/base.tsx
        {!hideIcon && getProjectIcon()}
````
````old web/apps/web/core/components/dropdowns/project/base.tsx
          <span className="max-w-40 truncate">{getDisplayName(value, placeholder)}</span>
        )}
        {dropdownArrow && (
          <ChevronDownOutline className={cn("h-2.5 w-2.5 flex-shrink-0", dropdownArrowClassName)} aria-hidden="true" />
````
````new web/apps/web/core/components/dropdowns/project/base.tsx
          <span className="max-w-40 truncate">{getDisplayName()}</span>
````
````old web/apps/web/core/components/dropdowns/project/base.tsx
      onKeyDown={handleKeyDown}
      button={comboButton}
      renderByDefault={renderByDefault}
````
````new web/apps/web/core/components/dropdowns/project/base.tsx
      onKeyDown={handleKeyDown}
      role="presentation"
      button={comboButton}
````

`web/apps/web/core/components/dropdowns/project/dropdown.tsx`（修改，2 处）：

````old web/apps/web/core/components/dropdowns/project/dropdown.tsx

import type { ReactNode } from "react";
````
````new web/apps/web/core/components/dropdowns/project/dropdown.tsx

````
````old web/apps/web/core/components/dropdowns/project/dropdown.tsx
type Props = TDropdownProps & {
  button?: ReactNode;
  dropdownArrow?: boolean;
  dropdownArrowClassName?: string;
  onClose?: () => void;
  renderCondition?: (projectId: string) => boolean;
  renderByDefault?: boolean;
  currentProjectId?: string;
````
````new web/apps/web/core/components/dropdowns/project/dropdown.tsx
type Props = TDropdownProps & {
  renderCondition?: (projectId: string) => boolean;
````

- [ ] **Step 2: 其余的死 prop 和死成员**

`web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/header.tsx`（修改，4 处）：

````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/header.tsx
    href: string;
    icon: React.FC<React.SVGAttributes<SVGElement> & { className?: string }>;
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/header.tsx
    icon: React.FC<React.SVGAttributes<SVGElement>>;
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/header.tsx
    label: "Work items",
    href: "/issues",
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/header.tsx
    label: "Work items",
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/header.tsx
    label: "Cycles",
    href: "/cycles",
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/header.tsx
    label: "Cycles",
````
````old web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/header.tsx
    label: "Modules",
    href: "/modules",
````
````new web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/header.tsx
    label: "Modules",
````

`web/apps/web/core/components/dropdowns/member/avatar.tsx`（修改，5 处）：

````old web/apps/web/core/components/dropdowns/member/avatar.tsx
import { Avatar } from "@makeplane/propel/components/avatar";
import type { AvatarGroupSize } from "@makeplane/propel/components/avatar-group";
````
````new web/apps/web/core/components/dropdowns/member/avatar.tsx
import { Avatar } from "@makeplane/propel/components/avatar";
````
````old web/apps/web/core/components/dropdowns/member/avatar.tsx
  icon?: ComponentType<SVGProps<SVGSVGElement>>;
  size?: AvatarGroupSize;
````
````new web/apps/web/core/components/dropdowns/member/avatar.tsx
  icon?: ComponentType<SVGProps<SVGSVGElement>>;
````
````old web/apps/web/core/components/dropdowns/member/avatar.tsx
  const { userIds, icon: Icon, size = "xs" } = props;
````
````new web/apps/web/core/components/dropdowns/member/avatar.tsx
  const { userIds, icon: Icon } = props;
````
````old web/apps/web/core/components/dropdowns/member/avatar.tsx
        <AvatarGroupOverflow size={size}>
````
````new web/apps/web/core/components/dropdowns/member/avatar.tsx
        <AvatarGroupOverflow size="xs">
````
````old web/apps/web/core/components/dropdowns/member/avatar.tsx
          size={size}
````
````new web/apps/web/core/components/dropdowns/member/avatar.tsx
          size="xs"
````

`web/apps/web/core/components/dropdowns/state/state-option.tsx`（修改，1 处）：

````old web/apps/web/core/components/dropdowns/state/state-option.tsx
    value: string | undefined;
    query: string;
````
````new web/apps/web/core/components/dropdowns/state/state-option.tsx
    value: string | undefined;
````

`web/apps/web/core/components/power-k/menus/members.tsx`（修改，3 处）：

````old web/apps/web/core/components/power-k/menus/members.tsx
  handleSelect: (assigneeId: string) => void;
  heading?: string;
````
````new web/apps/web/core/components/power-k/menus/members.tsx
  handleSelect: (assigneeId: string) => void;
````
````old web/apps/web/core/components/power-k/menus/members.tsx
  const { handleSelect, heading, userIds, value } = props;
````
````new web/apps/web/core/components/power-k/menus/members.tsx
  const { handleSelect, userIds, value } = props;
````
````old web/apps/web/core/components/power-k/menus/members.tsx
    <Command.Group heading={heading}>
````
````new web/apps/web/core/components/power-k/menus/members.tsx
    <Command.Group>
````

`web/apps/web/core/components/project/create/common-attributes.tsx`（修改，5 处）：

````old web/apps/web/core/components/project/create/common-attributes.tsx
  setShouldAutoSyncIdentifier: (value: boolean) => void;
  handleFormOnChange?: () => void;
````
````new web/apps/web/core/components/project/create/common-attributes.tsx
  setShouldAutoSyncIdentifier: (value: boolean) => void;
````
````old web/apps/web/core/components/project/create/common-attributes.tsx
  const { setValue, isMobile, shouldAutoSyncIdentifier, setShouldAutoSyncIdentifier, handleFormOnChange } = props;
````
````new web/apps/web/core/components/project/create/common-attributes.tsx
  const { setValue, isMobile, shouldAutoSyncIdentifier, setShouldAutoSyncIdentifier } = props;
````
````old web/apps/web/core/components/project/create/common-attributes.tsx
      onChange(e);
      handleFormOnChange?.();
````
````new web/apps/web/core/components/project/create/common-attributes.tsx
      onChange(e);
````
````old web/apps/web/core/components/project/create/common-attributes.tsx
    onChange(alphanumericValue);
    handleFormOnChange?.();
````
````new web/apps/web/core/components/project/create/common-attributes.tsx
    onChange(alphanumericValue);
````
````old web/apps/web/core/components/project/create/common-attributes.tsx
                  onChange={(e) => {
                    onChange(e);
                    handleFormOnChange?.();
                  }}
````
````new web/apps/web/core/components/project/create/common-attributes.tsx
                  onChange={onChange}
````

`web/apps/web/core/components/settings/workspace/sidebar/root.tsx`（整个文件，25 行）：

````whole web/apps/web/core/components/settings/workspace/sidebar/root.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// nerve imports
import { ScrollArea } from "@nerve/propel/scrollarea";
// local imports
import { WorkspaceSettingsSidebarHeader } from "./header";
import { WorkspaceSettingsSidebarItemCategories } from "./item-categories";

export function WorkspaceSettingsSidebarRoot() {
  return (
    <ScrollArea
      scrollType="hover"
      orientation="vertical"
      size="sm"
      rootClassName="h-full w-[250px] shrink-0 animate-fade-in overflow-y-scroll border-r border-r-subtle bg-surface-1"
    >
      <WorkspaceSettingsSidebarHeader />
      <WorkspaceSettingsSidebarItemCategories />
    </ScrollArea>
  );
}
````

`web/apps/web/core/components/workspace/invite-modal/actions.tsx`（整个文件，48 行）：

````whole web/apps/web/core/components/workspace/invite-modal/actions.tsx
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";

// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { AddOutline } from "@makeplane/propel/icons";

type TInvitationModalActionsProps = {
  isSubmitting?: boolean;
  handleClose: () => void;
  appendField: () => void;
};

export const InvitationModalActions = observer(function InvitationModalActions(props: TInvitationModalActionsProps) {
  const { isSubmitting = false, handleClose, appendField } = props;
  // store hooks
  const { t } = useTranslation();

  return (
    <div className="mt-5 flex items-center justify-between gap-2">
      <button
        type="button"
        className="flex items-center gap-1 bg-transparent py-2 pr-3 text-caption-md-medium text-accent-primary outline-accent-strong"
        onClick={appendField}
      >
        <AddOutline className="h-3.5 w-3.5" />
        {t("common.add_more")}
      </button>
      <div className="flex items-center gap-2">
        {/* Disabled while the invitations are out: nerve's answer is said in the form that sent them */}
        <Button variant="secondary" size="lg" onClick={handleClose} disabled={isSubmitting}>
          {t("cancel")}
        </Button>
        <Button variant="primary" size="lg" type="submit" loading={isSubmitting}>
          {isSubmitting
            ? t("workspace_settings.settings.members.modal.button_loading")
            : t("workspace_settings.settings.members.modal.button")}
        </Button>
      </div>
    </div>
  );
});
````

`web/apps/web/core/store/member/workspace/workspace-member.store.ts`（修改，3 处）：

````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
  workspaceMemberInvitationIds: string[] | null;
  memberMap: Memberships | null;
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
  workspaceMemberInvitationIds: string[] | null;
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
      workspaceMemberInvitationIds: computed,
      memberMap: computed,
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
      workspaceMemberInvitationIds: computed,
````
````old web/apps/web/core/store/member/workspace/workspace-member.store.ts
    return this.getWorkspaceMemberIds(workspaceSlug);
  }

  get memberMap() {
    const workspaceSlug = this.routerStore.workspaceSlug;
    if (!workspaceSlug) return null;
    return this.getMemberships(workspaceSlug) ?? {};
````
````new web/apps/web/core/store/member/workspace/workspace-member.store.ts
    return this.getWorkspaceMemberIds(workspaceSlug);
````

`web/packages/constants/src/members.ts`（修改，6 处）：

````old web/packages/constants/src/members.ts
    descendingOrderTitle: string;
    iconName: string;
````
````new web/packages/constants/src/members.ts
    descendingOrderTitle: string;
````
````old web/packages/constants/src/members.ts
    descendingOrderKey: "-full_name",
    descendingOrderTitle: "Z",
    iconName: "User",
````
````new web/packages/constants/src/members.ts
    descendingOrderKey: "-full_name",
    descendingOrderTitle: "Z",
````
````old web/packages/constants/src/members.ts
    descendingOrderKey: "-display_name",
    descendingOrderTitle: "Z",
    iconName: "User",
````
````new web/packages/constants/src/members.ts
    descendingOrderKey: "-display_name",
    descendingOrderTitle: "Z",
````
````old web/packages/constants/src/members.ts
    descendingOrderTitle: "Z",
    iconName: "Mail",
````
````new web/packages/constants/src/members.ts
    descendingOrderTitle: "Z",
````
````old web/packages/constants/src/members.ts
    descendingOrderTitle: "New",
    iconName: "Calendar",
````
````new web/packages/constants/src/members.ts
    descendingOrderTitle: "New",
````
````old web/packages/constants/src/members.ts
    descendingOrderTitle: "Admin",
    iconName: "Shield",
````
````new web/packages/constants/src/members.ts
    descendingOrderTitle: "Admin",
````

`web/packages/propel/src/icons/state/dashed-circle.tsx`（修改，2 处）：

````old web/packages/propel/src/icons/state/dashed-circle.tsx
  percentage: number;
  totalSegments?: number;
````
````new web/packages/propel/src/icons/state/dashed-circle.tsx
  percentage: number;
````
````old web/packages/propel/src/icons/state/dashed-circle.tsx
export function DashedCircle({ center, color, percentage, totalSegments = 15 }: DashedCircleProps) {
````
````new web/packages/propel/src/icons/state/dashed-circle.tsx
/** How many dashes the circle is drawn with. */
const totalSegments = 15;

export function DashedCircle({ center, color, percentage }: DashedCircleProps) {
````

- [ ] **Step 3: P8a、P8b 留下的文件和 web 的上限**

`web/apps/web/core/components/core/sidebar/progress-stats/state_group.tsx`（修改，2 处）：

````old web/apps/web/core/components/core/sidebar/progress-stats/state_group.tsx
      {distribution.map((group, index) => (
````
````new web/apps/web/core/components/core/sidebar/progress-stats/state_group.tsx
      {distribution.map((group) => (
````
````old web/apps/web/core/components/core/sidebar/progress-stats/state_group.tsx
          key={index}
````
````new web/apps/web/core/components/core/sidebar/progress-stats/state_group.tsx
          key={group.state}
````

`web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx`（修改，2 处）：

````old web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx
    () =>
      projectIds
        ? (projectIds.map((projectId) => getProjectById(projectId)).filter((project) => project) as Project[])
        : [],
````
````new web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx
    () => (projectIds ? (projectIds.map((id) => getProjectById(id)).filter((listed) => listed) as Project[]) : []),
````
````old web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx
        getOptionIcon: (project) => <Logo logo={project.logo_props} size={12} />,
````
````new web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx
        getOptionIcon: (option) => <Logo logo={option.logo_props} size={12} />,
````

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 346",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 337",
````

- [ ] **Step 4: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 69 条规则、3 个例外，没有命中；web 的 oxlint 337 条，等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 110 个全部通过。

- [ ] **Step 5: 提交**

```bash
git add 'web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/header.tsx' web/apps/web/core/components/core/sidebar/progress-stats/state_group.tsx web/apps/web/core/components/dropdowns/member/avatar.tsx web/apps/web/core/components/dropdowns/project/base.tsx web/apps/web/core/components/dropdowns/project/dropdown.tsx web/apps/web/core/components/dropdowns/state/state-option.tsx web/apps/web/core/components/power-k/menus/members.tsx web/apps/web/core/components/project/create/common-attributes.tsx web/apps/web/core/components/settings/workspace/sidebar/root.tsx web/apps/web/core/components/workspace/invite-modal/actions.tsx web/apps/web/core/hooks/work-item-filters/use-work-item-filters-config.tsx web/apps/web/core/store/member/workspace/workspace-member.store.ts web/apps/web/package.json web/packages/constants/src/members.ts web/packages/propel/src/icons/state/dashed-circle.tsx
```
```bash
git commit -m "refactor(M3/P11): M3's dead members and props go, and the files P8a and P8b left keep no oxlint warning

The props no caller passes (the project dropdown's six, the member
avatar's size, the state option's query, the command palette's heading,
the project form's handleFormOnChange, the workspace sidebar's className,
the invite buttons' five, the dashed circle's totalSegments, the archives
header's href) and the members nothing reads (the workspace members'
memberMap, the member columns' iconName) are deleted; each component
shows what its default showed. The project dropdown's getter is nerve's
Project. The work-item filters' and the state groups' warnings go; the
web app's cap is 337.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p11.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T8.1` | 项目下拉框的 `ComboDropDown` 没有 `role` | oxlint（`check:lint`） | 静态 |
| `T8.2` | 状态组的进度以下标为 key | oxlint（`check:lint`） | 静态 |

---

### Task 9: `promise(always-return)` 全仓清零

**Files:**
- Modify: `web/apps/web/app/entry.client.tsx`、`web/apps/web/core/components/cycles/delete-modal.tsx`、`web/apps/web/core/components/cycles/modal.tsx`、`web/apps/web/core/components/cycles/quick-actions.tsx`、`web/apps/web/core/components/cycles/transfer-issues-modal.tsx`、`web/apps/web/core/components/inbox/modals/delete-issue-modal.tsx`、`web/apps/web/core/components/issues/delete-issue-modal.tsx`、`web/apps/web/core/components/issues/issue-detail/issue-activity/helper.tsx`、`web/apps/web/core/components/issues/peek-overview/header.tsx`、`web/apps/web/core/components/issues/workspace-draft/delete-modal.tsx`、`web/apps/web/core/components/modules/delete-module-modal.tsx`、`web/apps/web/core/components/modules/module-card-item.tsx`、`web/apps/web/core/components/modules/module-list-item-action.tsx`、`web/apps/web/core/components/modules/quick-actions.tsx`、`web/apps/web/core/components/power-k/config/miscellaneous-commands.ts`、`web/apps/web/core/components/power-k/ui/pages/context-based/cycle/commands.ts`、`web/apps/web/core/components/power-k/ui/pages/context-based/module/commands.tsx`、`web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts`、`web/apps/web/core/components/web-hooks/create-webhook-modal.tsx`、`web/apps/web/core/components/web-hooks/form/secret-key.tsx`、`web/apps/web/core/components/work-item-filters/filters-hoc/project-level.tsx`、`web/apps/web/core/components/workspace-notifications/sidebar/notification-card/options/snooze/modal.tsx`、`web/apps/web/core/components/workspace/views/default-view-quick-action.tsx`、`web/apps/web/core/store/global-view.store.ts`、`web/apps/web/core/store/workspace/webhook.store.ts`、`web/apps/web/package.json`、`web/packages/editor/package.json`、`web/packages/editor/src/helpers/floating-ui.ts`

**Interfaces:**
- Produces（spec 2.9、附录 A.6；M3 设计 7.9；M1 收尾交接；P8a spec 第 5 节的 P11 一行）：
  - oxlint 指出的 35 个 `then()` 回调（web 34、editor 1）各在最后一个语句之后加 `return;`（原型中由一次性的脚本 `alwaysreturn.mjs` 照 oxlint 的 JSON 写出，见 spec 附录 A.6；下面的块就是它写出的）。回调原来的完成值是 `undefined`，加了 `return;` 仍是 `undefined`：`then()` 给出的 promise 以同样的值、在同样的时刻兑现，拒绝的路径一处都不变（每一处的拒绝怎样处理：后面有 `.catch` 或第二个回调 21 处，交给调用方 9 处，没有处理 5 处，见 spec 附录 A.6；没有处理的照旧没有处理，列给收尾抄进 M4、M6 的交接）。
  - 上限：web 337 → 303，editor 65 → 64。`promise(always-return)` 全仓 0 条。
  - 这些文件中 5 个的其余 14 条警告不改（`no-shadow`、可点击的 `div`、`exhaustive-deps` 等，都在 M4、M6、M7、M8 的页面上：spec 第 3 节第 8 条）。

**Tests:** 没有新的测试：行为不变（spec 附录 A.6 逐处的说明）；规则由上限守着（本 Task 的变异表）。

- [ ] **Step 1: 回调的 `return;`**

`web/apps/web/app/entry.client.tsx`（修改，1 处）：

````old web/apps/web/app/entry.client.tsx
    });
````
````new web/apps/web/app/entry.client.tsx
    });
    return;
````

`web/apps/web/core/components/cycles/delete-modal.tsx`（修改，1 处）：

````old web/apps/web/core/components/cycles/delete-modal.tsx
            message: "Cycle deleted successfully.",
          });
````
````new web/apps/web/core/components/cycles/delete-modal.tsx
            message: "Cycle deleted successfully.",
          });
          return;
````

`web/apps/web/core/components/cycles/modal.tsx`（修改，4 处）：

````old web/apps/web/core/components/cycles/modal.tsx
          message: "Cycle created successfully.",
        });
````
````new web/apps/web/core/components/cycles/modal.tsx
          message: "Cycle created successfully.",
        });
        return;
````
````old web/apps/web/core/components/cycles/modal.tsx
          message: "Cycle updated successfully.",
        });
````
````new web/apps/web/core/components/cycles/modal.tsx
          message: "Cycle updated successfully.",
        });
        return;
````
````old web/apps/web/core/components/cycles/modal.tsx
      status = res.status;
````
````new web/apps/web/core/components/cycles/modal.tsx
      status = res.status;
      return;
````
````old web/apps/web/core/components/cycles/modal.tsx
          setCycleTab("all");
````
````new web/apps/web/core/components/cycles/modal.tsx
          setCycleTab("all");
          return;
````

`web/apps/web/core/components/cycles/quick-actions.tsx`（修改，2 处）：

````old web/apps/web/core/components/cycles/quick-actions.tsx
        message: t("common.link_copied_to_clipboard"),
      });
````
````new web/apps/web/core/components/cycles/quick-actions.tsx
        message: t("common.link_copied_to_clipboard"),
      });
      return;
````
````old web/apps/web/core/components/cycles/quick-actions.tsx
        navigate(`/${workspaceSlug}/projects/${projectId}/archives/cycles`);
````
````new web/apps/web/core/components/cycles/quick-actions.tsx
        navigate(`/${workspaceSlug}/projects/${projectId}/archives/cycles`);
        return;
````

`web/apps/web/core/components/cycles/transfer-issues-modal.tsx`（修改，1 处）：

````old web/apps/web/core/components/cycles/transfer-issues-modal.tsx
        await getCycleDetails(payload.new_cycle_id);
````
````new web/apps/web/core/components/cycles/transfer-issues-modal.tsx
        await getCycleDetails(payload.new_cycle_id);
        return;
````

`web/apps/web/core/components/inbox/modals/delete-issue-modal.tsx`（修改，1 处）：

````old web/apps/web/core/components/inbox/modals/delete-issue-modal.tsx
          message: `${t("inbox_issue.modals.delete.success")}`,
        });
````
````new web/apps/web/core/components/inbox/modals/delete-issue-modal.tsx
          message: `${t("inbox_issue.modals.delete.success")}`,
        });
        return;
````

`web/apps/web/core/components/issues/delete-issue-modal.tsx`（修改，1 处）：

````old web/apps/web/core/components/issues/delete-issue-modal.tsx
          onClose();
````
````new web/apps/web/core/components/issues/delete-issue-modal.tsx
          onClose();
          return;
````

`web/apps/web/core/components/issues/issue-detail/issue-activity/helper.tsx`（修改，1 处）：

````old web/apps/web/core/components/issues/issue-detail/issue-activity/helper.tsx
            });
````
````new web/apps/web/core/components/issues/issue-detail/issue-activity/helper.tsx
            });
            return;
````

`web/apps/web/core/components/issues/peek-overview/header.tsx`（修改，2 处）：

````old web/apps/web/core/components/issues/peek-overview/header.tsx
        message: t("common.link_copied_to_clipboard"),
      });
````
````new web/apps/web/core/components/issues/peek-overview/header.tsx
        message: t("common.link_copied_to_clipboard"),
      });
      return;
````
````old web/apps/web/core/components/issues/peek-overview/header.tsx
        setPeekIssue(undefined);
````
````new web/apps/web/core/components/issues/peek-overview/header.tsx
        setPeekIssue(undefined);
        return;
````

`web/apps/web/core/components/issues/workspace-draft/delete-modal.tsx`（修改，1 处）：

````old web/apps/web/core/components/issues/workspace-draft/delete-modal.tsx
          onClose();
````
````new web/apps/web/core/components/issues/workspace-draft/delete-modal.tsx
          onClose();
          return;
````

`web/apps/web/core/components/modules/delete-module-modal.tsx`（修改，1 处）：

````old web/apps/web/core/components/modules/delete-module-modal.tsx
          message: "Module deleted successfully.",
        });
````
````new web/apps/web/core/components/modules/delete-module-modal.tsx
          message: "Module deleted successfully.",
        });
        return;
````

`web/apps/web/core/components/modules/module-card-item.tsx`（修改，2 处）：

````old web/apps/web/core/components/modules/module-card-item.tsx
      if (!storedValue) toggleFavoriteMenu(true);
````
````new web/apps/web/core/components/modules/module-card-item.tsx
      if (!storedValue) toggleFavoriteMenu(true);
      return;
````
````old web/apps/web/core/components/modules/module-card-item.tsx
          message: "Module updated successfully.",
        });
````
````new web/apps/web/core/components/modules/module-card-item.tsx
          message: "Module updated successfully.",
        });
        return;
````

`web/apps/web/core/components/modules/module-list-item-action.tsx`（修改，2 处）：

````old web/apps/web/core/components/modules/module-list-item-action.tsx
      if (!storedValue) toggleFavoriteMenu(true);
````
````new web/apps/web/core/components/modules/module-list-item-action.tsx
      if (!storedValue) toggleFavoriteMenu(true);
      return;
````
````old web/apps/web/core/components/modules/module-list-item-action.tsx
          message: "Module updated successfully.",
        });
````
````new web/apps/web/core/components/modules/module-list-item-action.tsx
          message: "Module updated successfully.",
        });
        return;
````

`web/apps/web/core/components/modules/quick-actions.tsx`（修改，1 处）：

````old web/apps/web/core/components/modules/quick-actions.tsx
        message: "Module link copied to clipboard.",
      });
````
````new web/apps/web/core/components/modules/quick-actions.tsx
        message: "Module link copied to clipboard.",
      });
      return;
````

`web/apps/web/core/components/power-k/config/miscellaneous-commands.ts`（修改，1 处）：

````old web/apps/web/core/components/power-k/config/miscellaneous-commands.ts
          title: t("power_k.miscellaneous_actions.copy_current_page_url_toast_success"),
        });
````
````new web/apps/web/core/components/power-k/config/miscellaneous-commands.ts
          title: t("power_k.miscellaneous_actions.copy_current_page_url_toast_success"),
        });
        return;
````

`web/apps/web/core/components/power-k/ui/pages/context-based/cycle/commands.ts`（修改，1 处）：

````old web/apps/web/core/components/power-k/ui/pages/context-based/cycle/commands.ts
          title: t("power_k.contextual_actions.cycle.copy_url_toast_success"),
        });
````
````new web/apps/web/core/components/power-k/ui/pages/context-based/cycle/commands.ts
          title: t("power_k.contextual_actions.cycle.copy_url_toast_success"),
        });
        return;
````

`web/apps/web/core/components/power-k/ui/pages/context-based/module/commands.tsx`（修改，1 处）：

````old web/apps/web/core/components/power-k/ui/pages/context-based/module/commands.tsx
          title: t("power_k.contextual_actions.module.copy_url_toast_success"),
        });
````
````new web/apps/web/core/components/power-k/ui/pages/context-based/module/commands.tsx
          title: t("power_k.contextual_actions.module.copy_url_toast_success"),
        });
        return;
````

`web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts`（修改，3 处）：

````old web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts
          title: t("power_k.contextual_actions.work_item.copy_id_toast_success"),
        });
````
````new web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts
          title: t("power_k.contextual_actions.work_item.copy_id_toast_success"),
        });
        return;
````
````old web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts
          title: t("power_k.contextual_actions.work_item.copy_title_toast_success"),
        });
````
````new web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts
          title: t("power_k.contextual_actions.work_item.copy_title_toast_success"),
        });
        return;
````
````old web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts
          title: t("power_k.contextual_actions.work_item.copy_url_toast_success"),
        });
````
````new web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts
          title: t("power_k.contextual_actions.work_item.copy_url_toast_success"),
        });
        return;
````

`web/apps/web/core/components/web-hooks/create-webhook-modal.tsx`（修改，1 处）：

````old web/apps/web/core/components/web-hooks/create-webhook-modal.tsx
        csvDownload(csvData, `webhook-secret-key-${Date.now()}`);
````
````new web/apps/web/core/components/web-hooks/create-webhook-modal.tsx
        csvDownload(csvData, `webhook-secret-key-${Date.now()}`);
        return;
````

`web/apps/web/core/components/web-hooks/form/secret-key.tsx`（修改，1 处）：

````old web/apps/web/core/components/web-hooks/form/secret-key.tsx
          csvDownload(csvData, `webhook-secret-key-${Date.now()}`);
        }
````
````new web/apps/web/core/components/web-hooks/form/secret-key.tsx
          csvDownload(csvData, `webhook-secret-key-${Date.now()}`);
        }
        return;
````

`web/apps/web/core/components/work-item-filters/filters-hoc/project-level.tsx`（修改，1 处）：

````old web/apps/web/core/components/work-item-filters/filters-hoc/project-level.tsx
            message: "Your view has been updated successfully.",
          });
````
````new web/apps/web/core/components/work-item-filters/filters-hoc/project-level.tsx
            message: "Your view has been updated successfully.",
          });
          return;
````

`web/apps/web/core/components/workspace-notifications/sidebar/notification-card/options/snooze/modal.tsx`（修改，1 处）：

````old web/apps/web/core/components/workspace-notifications/sidebar/notification-card/options/snooze/modal.tsx
      handleClose();
````
````new web/apps/web/core/components/workspace-notifications/sidebar/notification-card/options/snooze/modal.tsx
      handleClose();
      return;
````

`web/apps/web/core/components/workspace/views/default-view-quick-action.tsx`（修改，1 处）：

````old web/apps/web/core/components/workspace/views/default-view-quick-action.tsx
      });
````
````new web/apps/web/core/components/workspace/views/default-view-quick-action.tsx
      });
      return;
````

`web/apps/web/core/store/global-view.store.ts`（修改，1 处）：

````old web/apps/web/core/store/global-view.store.ts
        delete this.globalViewMap[viewId];
      });
````
````new web/apps/web/core/store/global-view.store.ts
        delete this.globalViewMap[viewId];
      });
      return;
````

`web/apps/web/core/store/workspace/webhook.store.ts`（修改，1 处）：

````old web/apps/web/core/store/workspace/webhook.store.ts
        this.webhooks = _webhooks;
      });
    });
````
````new web/apps/web/core/store/workspace/webhook.store.ts
        this.webhooks = _webhooks;
      });
      return;
    });
````

`web/packages/editor/src/helpers/floating-ui.ts`（修改，1 处）：

````old web/packages/editor/src/helpers/floating-ui.ts
        });
````
````new web/packages/editor/src/helpers/floating-ui.ts
        });
        return;
````

- [ ] **Step 2: 两个包的上限**

`web/apps/web/package.json`（修改，1 处）：

````old web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 337",
````
````new web/apps/web/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 303",
````

`web/packages/editor/package.json`（修改，1 处）：

````old web/packages/editor/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 65",
````
````new web/packages/editor/package.json
    "check:lint": "node ../../../tools/lint-cap.mjs 64",
````

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 69 条规则、3 个例外，没有命中；web 的 oxlint 303 条、editor 64 条，各等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 110 个全部通过。

- [ ] **Step 4: 提交**

```bash
git add web/apps/web/app/entry.client.tsx web/apps/web/core/components/cycles/delete-modal.tsx web/apps/web/core/components/cycles/modal.tsx web/apps/web/core/components/cycles/quick-actions.tsx web/apps/web/core/components/cycles/transfer-issues-modal.tsx web/apps/web/core/components/inbox/modals/delete-issue-modal.tsx web/apps/web/core/components/issues/delete-issue-modal.tsx web/apps/web/core/components/issues/issue-detail/issue-activity/helper.tsx web/apps/web/core/components/issues/peek-overview/header.tsx web/apps/web/core/components/issues/workspace-draft/delete-modal.tsx web/apps/web/core/components/modules/delete-module-modal.tsx web/apps/web/core/components/modules/module-card-item.tsx web/apps/web/core/components/modules/module-list-item-action.tsx web/apps/web/core/components/modules/quick-actions.tsx web/apps/web/core/components/power-k/config/miscellaneous-commands.ts web/apps/web/core/components/power-k/ui/pages/context-based/cycle/commands.ts web/apps/web/core/components/power-k/ui/pages/context-based/module/commands.tsx web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts web/apps/web/core/components/web-hooks/create-webhook-modal.tsx web/apps/web/core/components/web-hooks/form/secret-key.tsx web/apps/web/core/components/work-item-filters/filters-hoc/project-level.tsx web/apps/web/core/components/workspace-notifications/sidebar/notification-card/options/snooze/modal.tsx web/apps/web/core/components/workspace/views/default-view-quick-action.tsx web/apps/web/core/store/global-view.store.ts web/apps/web/core/store/workspace/webhook.store.ts web/apps/web/package.json web/packages/editor/package.json web/packages/editor/src/helpers/floating-ui.ts
```
```bash
git commit -m "refactor(M3/P11): every then() callback ends with return, and promise(always-return) is at zero across the repository

The 35 callbacks oxlint names, 34 in the web app and one in the editor,
end with return;: each completed with undefined and still does, so the
promise then() gives settles with the same value at the same moment and
no rejection is handled differently. The web app's cap is 303, the
editor's 64.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A.2；`mutants_p11.py` 的编号；"层"是发现它的检查所在的层：静态是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的错误级规则）、关键词守卫，以及 `make knip`；vitest 是 `make test-web`；端到端是 `make e2e` 的故事）：

| 变异 | 改坏 | 必须失败的检查 | 层 |
|---|---|---|---|
| `T9.1` | editor 的浮层定位的回调不返回 | `@nerve/editor` 的 oxlint（`check:lint`） | 静态 |
| `T9.2` | 删除迭代的回调不返回 | oxlint（`check:lint`） | 静态 |

---

### Task 10: 文档：README 的"前端"一节、前端改动清单和两份交接的处理结果

**Files:**
- Modify: `README.md`、`docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md`、`docs/v0/M3-workspace-project/handoffs/M1-closeout.md`、`docs/v0/frontend-changes.md`

**Interfaces:**
- Produces（spec 2.10；M3 设计 3.20、13.1 中 P11 的行；M1 收尾交接；M1-P2 交接）：
  - `README.md` 的"前端"一节：状态、标签的设置页一句改写为 M3/P11 起能用的（谁能改、谁只能看、访客看到没有权限），个人主页（访客看到用户卡片和没有权限；管理员和成员看到的工作项一栏与其余工作项页面一样由 M4–M8 对接）。
  - `docs/v0/frontend-changes.md`：3.1 中 M3 一行的状态（页面已对接：工作区的 M3/P9，项目的 M3/P10，状态和标签的设置页、访客看到的个人主页 M3/P11；M3 的收尾之后改为已完成）；3.2 的错误格式一行写明状态和标签的设置页的字段错误（M3/P11），"页面跟进修改的结果"一行加上状态、标签的设置页；3.2 新增五行，都是"已完成，M3/P11"：状态和标签的设置页（W17 的每一条，spec 第 3 节第 12 条）、确认框（`AlertModalCore` 在请求路上时关不掉，它的其余 11 个调用方同样）、M2 的账户页面跟进修改的结果、死成员和死 prop、`then()` 的回调。
  - `docs/v0/M3-workspace-project/handoffs/M1-closeout.md`：加"处理结果（M3/P11）"：死成员和死 prop（spec 2.8，留下的行在 spec 附录 A.9），oxlint（`promise(always-return)` 全仓清零，P8a、P8b 留下的文件清零，上限 web 348 → 303、editor 65 → 64，其余的包条数没有变；留着的是 `alert-modal.tsx` 的一条和只按 `always-return` 改到的文件中 5 个的 14 条，spec 第 3 节第 3、8 条）；状态保持 `open`，P11 的评审写明之后由 M3 的收尾关闭。
  - `docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md`：加"处理结果（M3/P11）"：访客看别人的个人主页（页面一侧完成，决策点 3 的 A），P9 的页面版本和 S2 核对，浏览器核对 C10 由控制者做；状态保持 `open`。
- 关键词守卫的最终状态和旧的 `ProjectService` 只核对、不改（spec 2.10）：69 条规则、3 个例外（`analytics` 到 M6、`brand` 到 M9、`plane-workspace-urls` 的两处 `/user-properties/` 到 M4），没有 `until: "M3"`；`ProjectService` 只剩 7.3 的三个方法（`getProjectUserProperties`、`updateProjectUserProperties`、`projectIssuesSearch`）。
- 控制者的浏览器核对 C9、C10 和评审不在本 plan 中（M3 设计 9.7、12 节 P11 的最后一个任务）。

**Tests:** 没有新的测试：本 Task 只改文档。

- [ ] **Step 1: 文档**

`README.md`（修改，1 处）：

````old README.md
- **能用的页面**（M3/P9 起）：登录、注册（注册关闭时，凭邀请链接仍可注册）；邀请链接的页面 `/workspace-invitations`（没登录时显示工作区和角色，登录后接受或忽略）；新手引导（资料一步；还没有工作区的人接着创建工作区，再邀请成员（规模选"Just myself"时创建之后就完成），发出邀请之后列出每个邀请的链接供复制，v0 不发邮件；已经有工作区的人资料一步之后就完成）；登录后的落点（上次打开的工作区；它不在列表中时，最早创建的那个；一个都没有时 `/create-workspace`）；创建工作区；工作区首页（问候、"快速入门指南"，第一次进入时的导览）；侧边栏的工作区菜单（切换、创建工作区、退出）和"项目导航"对话框；工作区设置的 general（名称、规模、时区，删除工作区）和 members（成员、改角色、移出、离开；管理员还有邀请：批量邀请、复制链接、改角色、删除）；个人设置的四个标签页（general、preferences、security、api-tokens，停用账户在 general 页）。项目的页面（M3/P10 起）：项目列表（卡片复制项目的链接；看得到而还不是成员的项目，卡片上加入）和已归档的项目（恢复、删除）；创建项目（负责人从工作区的管理员和成员中选；表情和图标的选择器，表情的数据由 Nerve 自己提供）；打开项目的地址时，不是成员的看到"加入项目"，看不到的看到"找不到项目"，已归档的看到它已归档；项目设置的 general（名称、标识、说明、可见性、时区、图标；归档、删除）、members（添加、改角色、移出、离开；负责人、默认负责人、访客可见全部）、features（迭代、模块、视图、收集箱的开关）和 automations（自动归档）；侧边栏拖动项目的顺序；标签式导航下项目页头的标签栏（默认打开的标签页、收进"更多"）。状态、标签的设置页和个人主页在 M3/P11 对接；工作项、迭代、模块、视图、收藏、通知、Webhook 等页面还在调用 Plane 的接口，Nerve 返回 404，由 M4–M8 对接。
````
````new README.md
- **能用的页面**（M3/P9 起）：登录、注册（注册关闭时，凭邀请链接仍可注册）；邀请链接的页面 `/workspace-invitations`（没登录时显示工作区和角色，登录后接受或忽略）；新手引导（资料一步；还没有工作区的人接着创建工作区，再邀请成员（规模选"Just myself"时创建之后就完成），发出邀请之后列出每个邀请的链接供复制，v0 不发邮件；已经有工作区的人资料一步之后就完成）；登录后的落点（上次打开的工作区；它不在列表中时，最早创建的那个；一个都没有时 `/create-workspace`）；创建工作区；工作区首页（问候、"快速入门指南"，第一次进入时的导览）；侧边栏的工作区菜单（切换、创建工作区、退出）和"项目导航"对话框；工作区设置的 general（名称、规模、时区，删除工作区）和 members（成员、改角色、移出、离开；管理员还有邀请：批量邀请、复制链接、改角色、删除）；个人设置的四个标签页（general、preferences、security、api-tokens，停用账户在 general 页）。项目的页面（M3/P10 起）：项目列表（卡片复制项目的链接；看得到而还不是成员的项目，卡片上加入）和已归档的项目（恢复、删除）；创建项目（负责人从工作区的管理员和成员中选；表情和图标的选择器，表情的数据由 Nerve 自己提供）；打开项目的地址时，不是成员的看到"加入项目"，看不到的看到"找不到项目"，已归档的看到它已归档；项目设置的 general（名称、标识、说明、可见性、时区、图标；归档、删除）、members（添加、改角色、移出、离开；负责人、默认负责人、访客可见全部）、features（迭代、模块、视图、收集箱的开关）和 automations（自动归档）；侧边栏拖动项目的顺序；标签式导航下项目页头的标签栏（默认打开的标签页、收进"更多"）。项目设置的 states 和 labels（M3/P11 起）：项目管理员和是工作区管理员的项目成员增删改、拖动排序，状态还能设为默认，标签能拖进、拖出标签组；项目的其他成员只能看，访客看到没有权限。个人主页（M3/P11 起）：访客打开别人的或自己的主页，看到用户卡片，工作项一栏说他没有权限；管理员和成员看到的工作项列表，与工作项、迭代、模块、视图、收藏、通知、Webhook 等页面一样还在调用 Plane 的接口，Nerve 返回 404，由 M4–M8 对接。
````

`docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md`（修改，1 处）：

````old docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md
来源：[M3/P9 spec](../specs/P9-web-workspace-pages.md) 第 7 节。

````
````new docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md
来源：[M3/P9 spec](../specs/P9-web-workspace-pages.md) 第 7 节。

## 处理结果（M3/P11）

- **访客看别人的个人主页**（页面一侧完成）：照决策点 3 的 A，页面不改：访客打开别人的或自己的个人主页，看到用户卡片（数据来自成员列表），工作项一栏说他没有权限，页面不取工作项列表。故事 P9 的页面版本核对（`e2e/stories/project/p9-guest-profile.spec.ts`），S2 核对访客打开这两页时只取应用和工作区的数据。浏览器核对 C10 由控制者做，写进 P11 的评审。

状态保持 `open`：C10 写进评审之后，M3 的收尾关闭。

````

`docs/v0/M3-workspace-project/handoffs/M1-closeout.md`（修改，1 处）：

````old docs/v0/M3-workspace-project/handoffs/M1-closeout.md
来源：[M1 收尾 spec](../../M1-frontend-trim/specs/closeout.md)第 4 节、第 8 节。

````
````new docs/v0/M3-workspace-project/handoffs/M1-closeout.md
来源：[M1 收尾 spec](../../M1-frontend-trim/specs/closeout.md)第 4 节、第 8 节。

## 处理结果（M3/P11）

- **死成员和死 prop**：P11 删掉它改到的文件和只经机械步骤到达的文件中真正没有读者、没有调用方传入的行（[P11 spec](../specs/P11-web-states-labels-and-cleanup.md) 第 2.8 节）。`--rows M3` 剩下的每一行写进 P11 spec 的附录 A.9：文件、名称、谁在读或传它（测试里 `it.each` 的列和 `vi.hoisted` 的桩、经展开或第三方库读取的成员、M4 的搜索类型），以及按路径归到 M3、实际属于 M4、M6、M7 的行（M3 设计第 1 节；收尾抄进它们的交接）。
- **oxlint**：`promise(always-return)` 全仓清零（35 条：web 34、editor 1）；P8a、P8b 留给 P11 的只经机械步骤到达的文件清零；各包的上限随之调到新的条数（web 348 → 303、editor 65 → 64，其余的包条数没有变）。P11 改到的文件中留着的警告：`@nerve/ui` 的 `alert-modal.tsx` 的一条（`tabIndex={1}`，M3 收尾的 `ModalCore` 重写），以及只按 `always-return` 改到的文件中 5 个的 14 条（M4、M6、M7、M8 的组件；P11 spec 第 3 节第 3、8 条）。

状态保持 `open`：P11 的评审写明剩下的行和上限之后，M3 的收尾关闭。

````

`docs/v0/frontend-changes.md`（修改，4 处）：

````old docs/v0/frontend-changes.md
| 工作区、成员、邀请、项目、项目成员、项目归档、状态、标签、显示设置 | M3 | 进行中：数据层已对接（工作区一侧 M3/P8a，项目一侧 M3/P8b）；工作区的页面 M3/P9、项目的页面 M3/P10 已对接，状态、标签的设置页在 M3/P11 |
````
````new docs/v0/frontend-changes.md
| 工作区、成员、邀请、项目、项目成员、项目归档、状态、标签、显示设置 | M3 | 进行中：数据层已对接（工作区一侧 M3/P8a，项目一侧 M3/P8b）；页面已对接（工作区的 M3/P9，项目的 M3/P10，状态和标签的设置页、访客看到的个人主页 M3/P11），M3 的收尾之后改为已完成 |
````
````old docs/v0/frontend-changes.md
| 所有处理接口错误的地方 | 统一按 RFC 9457 的 problem+json 读取 `code`、`title`、`errors`。M2/P4 已改：`ApiError` 和 `unwrap`（`core/lib/api-error.ts`）按生成的 `Problem` 读取；登录页、注册页和安全页的修改密码按它的 `code`、`errors` 显示错误。M2/P5 改完个人设置的其余部分（[M2 设计](M2-auth/M2-design.md) 7.7）和新手引导的资料步骤：general 页的保存、资料步骤和 api-tokens 页的创建把字段错误显示在字段下方（名字的规则只在 nerve，页面只查必填，Plane 的名字校验从 `@nerve/utils` 删除），其余的错误和 preferences 的主题、时区、语言、每周第一天、PAT 的撤销、停用账户、新手引导换步骤时更新资料（`onboarding/root.tsx`）的失败都在提示中，文案按 `code` 取（`core/lib/error-messages.ts` 的 `PROBLEM_MESSAGES`、`fieldErrorKeys`、`errorMessageKey`，M3/P8a 从 `helpers/authentication.helper.ts` 移来）。M3/P8a 起工作区、成员、邀请、工作区的显示设置，M3/P8b 起项目、项目成员、状态、标签、项目的显示设置经生成的客户端，nerve 的错误应答是 `ApiError`：状态和标签的设置页、工作项的三个标签选择器和项目包装层的加入界面在 M3/P8b 已按 `code` 显示错误（`errorMessageKey`），工作区的页面在 M3/P9（新手引导、邀请页、注册、创建工作区、首页、侧边栏的项目导航、工作区设置和停用账户的弹窗；创建工作区和邀请的表单把 nerve 指名的字段错误显示在对应的字段下方），项目的页面在 M3/P10（项目列表的加入、创建项目、项目设置的 general、members、features、automations，归档、恢复、删除，侧边栏的项目顺序和页头的标签栏；创建项目和 general 页把 nerve 指名的字段错误显示在对应的字段下方；general 页标识的检查被拒绝时说 nerve 的原因、按钮结束加载，原来是未处理的拒绝、一直加载）；M4–M8 的领域在各自的 M，它们现在还经 Plane 的 axios 基类按 Plane 的错误格式读取 | 错误格式统一 | 进行中 | |
````
````new docs/v0/frontend-changes.md
| 所有处理接口错误的地方 | 统一按 RFC 9457 的 problem+json 读取 `code`、`title`、`errors`。M2/P4 已改：`ApiError` 和 `unwrap`（`core/lib/api-error.ts`）按生成的 `Problem` 读取；登录页、注册页和安全页的修改密码按它的 `code`、`errors` 显示错误。M2/P5 改完个人设置的其余部分（[M2 设计](M2-auth/M2-design.md) 7.7）和新手引导的资料步骤：general 页的保存、资料步骤和 api-tokens 页的创建把字段错误显示在字段下方（名字的规则只在 nerve，页面只查必填，Plane 的名字校验从 `@nerve/utils` 删除），其余的错误和 preferences 的主题、时区、语言、每周第一天、PAT 的撤销、停用账户、新手引导换步骤时更新资料（`onboarding/root.tsx`）的失败都在提示中，文案按 `code` 取（`core/lib/error-messages.ts` 的 `PROBLEM_MESSAGES`、`fieldErrorKeys`、`errorMessageKey`，M3/P8a 从 `helpers/authentication.helper.ts` 移来）。M3/P8a 起工作区、成员、邀请、工作区的显示设置，M3/P8b 起项目、项目成员、状态、标签、项目的显示设置经生成的客户端，nerve 的错误应答是 `ApiError`：状态和标签的设置页、工作项的三个标签选择器和项目包装层的加入界面在 M3/P8b 已按 `code` 显示错误（`errorMessageKey`），工作区的页面在 M3/P9（新手引导、邀请页、注册、创建工作区、首页、侧边栏的项目导航、工作区设置和停用账户的弹窗；创建工作区和邀请的表单把 nerve 指名的字段错误显示在对应的字段下方），项目的页面在 M3/P10（项目列表的加入、创建项目、项目设置的 general、members、features、automations，归档、恢复、删除，侧边栏的项目顺序和页头的标签栏；创建项目和 general 页把 nerve 指名的字段错误显示在对应的字段下方；general 页标识的检查被拒绝时说 nerve 的原因、按钮结束加载，原来是未处理的拒绝、一直加载），状态和标签的设置页在 M3/P11（名字已被占用和 nerve 指名的字段错误显示在对应的字段下方，原来在提示里）；M4–M8 的领域在各自的 M，它们现在还经 Plane 的 axios 基类按 Plane 的错误格式读取 | 错误格式统一 | 进行中 | |
````
````old docs/v0/frontend-changes.md
| 页面跟进修改的结果 | 工作区的页面（M3/P9）和项目的页面（M3/P10）发出修改之后的跳转、提示和界面变化，只在发出它的会话里进行（`core/lib/in-session.ts` 的 `followInSession`）：另一个标签页换了账户之后，页面已是那个账户的，旧会话的修改不论成败都不再跟进。项目的弹窗（加入、移出和离开的确认、离开、添加成员、自定义时长、删除、归档和恢复）请求在路上时关不掉（取消禁用，Escape 和点弹窗外面都不关；原来可以），被拒绝时留着、按钮重新可用，原因在提示里，只在成功的跟进中关上（原来成员页的确认框被拒绝时也关上）；自定义时长 nerve 做完才关（原来发出就关）。功能的开关不再显示"Updating project feature..."的加载提示，完成时提示，被拒绝时说 nerve 的原因 | M3 设计 7.1、7.6 | 已完成 | M3/P9、P10 |
````
````new docs/v0/frontend-changes.md
| 页面跟进修改的结果 | 工作区的页面（M3/P9）、项目的页面（M3/P10）和状态、标签的设置页（M3/P11）发出修改之后的跳转、提示和界面变化，只在发出它的会话里进行（`core/lib/in-session.ts` 的 `followInSession`）：另一个标签页换了账户之后，页面已是那个账户的，旧会话的修改不论成败都不再跟进。项目的弹窗（加入、移出和离开的确认、离开、添加成员、自定义时长、删除、归档和恢复）请求在路上时关不掉（取消禁用，Escape 和点弹窗外面都不关；原来可以），被拒绝时留着、按钮重新可用，原因在提示里，只在成功的跟进中关上（原来成员页的确认框被拒绝时也关上）；自定义时长 nerve 做完才关（原来发出就关）。功能的开关不再显示"Updating project feature..."的加载提示，完成时提示，被拒绝时说 nerve 的原因 | M3 设计 7.1、7.6 | 已完成 | M3/P9、P10、P11 |
````
````old docs/v0/frontend-changes.md
| 创建项目的 Tab 顺序 | 创建项目的弹窗不再有正的 `tabIndex`：表单不再给字段、网络、负责人、按钮和关闭按钮（`ETabIndices.PROJECT_CREATE` 删除），创建之后的功能一步不再给关闭和打开项目（两处 oxlint 的抑制随之删除）。Tab 按页面中的顺序走，绕着弹窗转；原来正的 `tabIndex` 排在弹窗的焦点陷阱之前，Tab 在表单的关闭按钮、功能一步的打开项目之后离开弹窗，走到后面的页面（侧边栏、工作区切换；功能一步时先是创建成功的提示） | 弹窗里的 Tab 不离开弹窗（M3/P10 的浏览器核对 C7；功能一步是之后的裁定 F-18） | 已完成 | M3/P10 |
````
````new docs/v0/frontend-changes.md
| 创建项目的 Tab 顺序 | 创建项目的弹窗不再有正的 `tabIndex`：表单不再给字段、网络、负责人、按钮和关闭按钮（`ETabIndices.PROJECT_CREATE` 删除），创建之后的功能一步不再给关闭和打开项目（两处 oxlint 的抑制随之删除）。Tab 按页面中的顺序走，绕着弹窗转；原来正的 `tabIndex` 排在弹窗的焦点陷阱之前，Tab 在表单的关闭按钮、功能一步的打开项目之后离开弹窗，走到后面的页面（侧边栏、工作区切换；功能一步时先是创建成功的提示） | 弹窗里的 Tab 不离开弹窗（M3/P10 的浏览器核对 C7；功能一步是之后的裁定 F-18） | 已完成 | M3/P10 |
| 状态和标签的设置页 | 页面的工作区和项目取自路由的参数（不再读路由 store），权限的判断写明工作区和项目；增删改、拖动和设默认只在发出它的会话里跟进；表单在 nerve 回答、页面跟进之前一直忙（按钮显示"Creating""Adding"一类、按不了，取消也按不了；状态的表单原来马上就能再点），名字已被占用显示在名字下方（标签原来在提示里），被拒绝时表单留着；删除的确认框请求在路上时关不掉，被拒绝时留着、原因在提示里；标签拖到 nerve 不允许的地方（第三层）时说标签只有两层，原来说"出错了"一类的通用原因；状态的组标题是按钮（Tab 能到，`aria-expanded`），每一行的编辑、删除、设默认的按钮 Tab 能到、有名字（原来只在指针经过时显示）；标签每一行的菜单和删除按钮有名字，Tab 到达时显示（原来只在指针经过时显示）；标签组的菜单和拖动手柄、没有标签时的"添加"按钮只给能改标签的人（原来项目的每个成员都看得到组的菜单）；正在编辑的标签不能拖动 | M3 设计 7.6、9.2；总体设计 7.7 | 已完成 | M3/P11 |
| 确认框 | `AlertModalCore`（`@nerve/ui`）在 `isSubmitting` 时关不掉：取消按不了，Escape 和点外面都不关（M3/P9 的 F-1）；它的其余 11 个调用方（M4–M8 的 10 处确认框、个人设置的 API 令牌的撤销）同样 | 请求在路上的弹窗关不掉（总体设计 7.7） | 已完成 | M3/P11 |
| M2 的账户页面跟进修改的结果 | 新手引导的资料一步、个人设置的 general、preferences（时区、语言、每周第一天）和 security 页，修改之后的提示、字段下方的错误和清空表单，只在发出修改的会话里进行（`followInSession`）；说什么、怎样说照旧（M2 设计 7.3） | M3 设计 7.1（M3/P9 的裁定 P8） | 已完成 | M3/P11 |
| 死成员和死 prop | 没有调用方传入的 prop（项目下拉框的 `button`、`dropdownArrow` 等六个，邀请弹窗按钮行的五个，成员头像的 `size`、命令面板成员菜单的 `heading`、创建项目的 `handleFormOnChange`、工作区设置侧边栏的 `className`、状态图标虚线圆的 `totalSegments`）和没有代码读取的成员（工作区成员 store 的 `memberMap`、成员列的 `iconName`、归档页头的 `href`、状态选项的 `query`）删除，组件按原来的默认值显示；项目下拉框的取项目函数的类型是生成的 `Project` | M1 收尾交接（M3 设计 7.9） | 已完成 | M3/P11 |
| `then()` 的回调 | 35 处 `then()` 回调的末尾加 `return;`（`promise(always-return)` 全仓清零），回调的结果照旧是 `undefined`，行为不变 | M3 设计 7.9 | 已完成 | M3/P11 |
````

- [ ] **Step 2: 核对关键词守卫和旧的 `ProjectService`**

Run: `node -e 'const k=require("./tools/keywords.json");console.log(k.rules.length,JSON.stringify(k.exceptions.map((e)=>[e.rule,e.until,e.count??1])))'`
Expected: `69 [["analytics","M6",1],["brand","M9",3],["plane-workspace-urls","M4",2]]`：没有 `until` 为 `M3` 的例外，M3 领域只剩两处 `/user-properties/`（到 M4）。

Run: `grep -nE '^  (async )?[a-zA-Z]+\(' web/apps/web/core/services/project/project.service.ts`
Expected: 三行：`getProjectUserProperties`（第 18 行）、`updateProjectUserProperties`（第 26 行）、`projectIssuesSearch`（第 38 行）。

- [ ] **Step 3: 运行检查**

Run: `make lint-web`
Expected: 通过；关键词守卫 69 条规则、3 个例外，没有命中；web 的 oxlint 303 条、editor 64 条，各等于上限。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 110 个全部通过。

- [ ] **Step 4: 提交**

```bash
git add README.md docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md docs/v0/M3-workspace-project/handoffs/M1-closeout.md docs/v0/frontend-changes.md
```
```bash
git commit -m "docs(M3/P11): README's frontend section lists the state and label settings and the profile, the frontend changes list P11's, and two handoffs their results

README's frontend section says who may change a project's states and
labels from M3/P11 on, who sees them and what a guest sees on a profile;
the frontend changes list says the pages are connected and records P11's
five changes. The M1 closeout handoff records the dead rows and the caps,
the M1-P2 handoff the guest's profile.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 只改文档，没有变异。
