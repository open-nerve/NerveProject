# M7 / P5b 预检

> 2026-11-24：结论 **修正后通过**（Ready after fixes）。严重 0、重要 5、次要 14、观察 7。合同在 `ab86965f` 上都撑得住：资产的四个写对收集箱里的工作项按 `intake.update` 加提交者、只在待处理与暂缓且收集箱开着时（P2c spec §1、§2.2，`api/modules/asset.yaml:59`、`:118`、`:150`、`:192`），三个链接操作的码里没有 `issue.in_triage`、`issue.intake_closed`、`issue.intake_disabled`（`IY:1751`、`:1789`、`:1816`），提交的端点与请求体同 spec 写的（`IY:858-903`、`:2353-2388`）。spec 与 plan 引的行号核了约 150 处（Task 依赖的全部），只有两处差一两行。两条高风险裁定方向都对：R9 只动 `canWrite` 的 `issue` 宿主一支、谓词一处，读者里只漏列了一个（行为随之，不出错）；R20 (b) 的“轮到时算”在写的层已有现成的接缝。没有发现裁定与代码或合同相悖。要先改的五件事：① T2 的允许清单做不出——两份既有测试把表单直接渲染在 `ModalCore` 之外，标题一改成 `ModalTitle` 就在渲染时抛错；② H8 的“扣住编辑器代码块”在有选中条目的收集箱页上扣不住，照 plan 写出的故事会是空证据；③ O-2 的落点只有一句话，通知列表的区域从预览够不着，落点链在列表没画出来时断在 `body`；④ 判定的读与浮层共用一个键，这个键的任何一次再验证（网络恢复、浮层自己的重试）都会让照“`WN/root.tsx:72-86` 的写法”写出的预览回到“加载”、撤下浮层或嵌入连同没发出的输入，这正是 P5a 整审 I1 的形状，而 jsdom 里读的 hook 是替身，没有一格在真实的 SWR 上；⑤ Q34 按代码是必然重现的，R20 的修复应当写成计划内的 Task。
>
> - **目标**：[P5b spec](../v0/M7-collaboration/specs/P5b-web-intake-files-embed.md)、[P5b plan](../v0/M7-collaboration/plans/P5b-web-intake-files-embed.md)（未提交的草稿），次要目标台账的 `implementer-brief.md` 与 `final.sh`。依据台账 `rulings.md`（B-1–B-23 与 O-1–O-4 的裁定）、`research/scope.md`、`research/questions.md`，[M7 设计](../v0/M7-collaboration/M7-design.md) §3.6、§12，[总体设计](../v0/v0-design.md) §7.7，[P2c spec](../v0/M7-collaboration/specs/P2c-intake-assets.md)，先例 [re-94](re-94-M7P5a预检.md)（含 §6）与 [P5a 阶段记录](../v0/M7-collaboration/reviews/P5a-web-intake-review.md) §3、§7、§9。
> - **基线**：main `ab86965f`（P5a 的 noff，`ab86965fc615fce57bfa4197515e58205e939222`），worktree `codex/m7-p5b`。

下文 `C/` = `web/apps/web/core/`，`A/` = `web/apps/web/app/(all)/[workspaceSlug]/(projects)/`，`IN/` = `C/components/inbox/`，`WN/` = `C/components/workspace-notifications/`，`IY` = `api/modules/issue.yaml`，`HO/` = `docs/v0/M7-collaboration/handoffs/`，`E/` = `e2e/`，`K` = `tools/keywords.json`。

## 1. 方法

一次 fresh 只读审查（`fable`）：只写本文件，不动 git 状态（只用了 `git status`、`git grep`），不起容器，不跑测试、构建与安装。跑过的只有 `grep`、`sed`、`wc`、`ls`、`cat` 与只读的 `python3 -I`（读 `K` 的规则；台账的 `doc-links.py`：235 个文档、2355 个链接、0 断，两份草稿在内）。`node tools/keywords.mjs`、oxlint、knip、deadsym、vitest、Playwright 都没有运行。

worktree 没有装依赖。两处第三方库的行为是读主目录 `node_modules` 里装着的源码核的（只读，版本同 `pnpm-workspace.yaml` 的 catalog）：`@headlessui/react@2.2.10` 的 `dialog.js`（`Dialog.Title` 在 `Dialog` 之外抛错；Escape 时先 `blur` 再 `close`）与 `swr@2.4.2` 的 `index.mjs`、`_internal`（不带数据的 `mutate()`、去重窗口、重连、挂载时是否再验证）。`plane/` 只用 `sed` 读了 10 处行号，没有在里面跑 git。

对照代码核了：

- 新建弹窗：`IN/modals/create-modal/` 五个产品文件与两份测试全文、`IN/page-header.tsx`、`web/packages/ui/src/modals/{modal-core,alert-modal,keep-focus}.tsx`、M4 的 `issue-modal/{draft-issue-layout,form-footer,base}.tsx`、`components/title-input.tsx`、`close-policy.test.tsx`、`fake-dialog.tsx`、`C/lib/{refused-focus,form-refusal,in-session}.ts`、两份 `tab-indices.ts` 与 `getTabIndex` 的读者、三份文案（en 的 `inbox.json`、`common.json`、`work-item.json`）。
- 写权限与资产：`C/store/asset/{content-view,root.store,upload-policy,content-deletion}.ts` 与 `canWrite`、`canRemoveAttachment` 的全部读者（`git grep`）、`C/hooks/{use-content-assets,editor/use-issue-images,editor/use-editor-assets,editor/editor-assets}.ts`、`C/store/issue/intake/status.ts`、`IN/content/{detail,root}.tsx`、`C/components/issues/detail-host.tsx`、`C/components/assets/attachments/root.tsx`、`issue-detail-widgets/` 的附件与链接、`permissions.store.ts:68-75`、服务端 `access/domain/decide.go:85`、`rules.go:143`、`:167`。
- 嵌入：`WN/{root,page-state}.tsx`、`WN/sidebar/{root,list,use-leaving-focus}.ts(x)`、`WN/root.test.tsx` 全文、通知页的布局、`C/components/issues/peek-overview/{root,view,error,index}.ts(x)`、`issue-detail-widgets/{resource-error,use-detail-scope}.ts(x)`、`C/lib/{use-session-swr,session-key}.ts`、`web/packages/constants/src/swr.ts`、`IN/content/header/{use-intake-actions,root,mobile,detail-focus}.ts(x)`、`IN/{root,landing,page-state}.ts(x)`、`C/store/issue/issue-details/issue.store.ts:40-152`。
- 重定向与“打开”：`A/browse/[workItem]/{layout,page}.tsx`、工作项 id 的入口页、`C/store/issue/intake/filters.ts`、收集箱的布局与页面、`header.test.tsx:100-196`、`content/root.test.tsx:155-178`、H10 的 `expectNoTriage` 与它的三个读者。
- Q34：`IN/content/issue-properties.tsx`、`custom-search-select.tsx:268-290`、`member/base.tsx`、`C/store/issue/{issue-writes,issue-write-fields,issue-write-intake}.ts`、`helpers/base-issues.store.ts:73-75`、`@nerve/types` 的 `issue-change.ts`、命令面板的 `work-item/{writes,mutation-commands}.ts`、写负责人集合的全部调用方（`git grep`）。
- 门禁与 e2e：`.oxlintrc.json:120-210`、`web/apps/web/package.json:10`、`K` 的 `project-address-route`、`workspace-address-route`、`old-intake-urls`、`copy-hook` 与 phase、`knip.jsonc`、`Makefile` 的 `e2e`、台账的 `final.sh`、`gates.sh`、`run-check.py`、`mutate.py`、`E/playwright.{config,m7-pages,m5-content,m4-detail}.config.ts`、`E/global-setup.ts`、`E/fixtures/{file-story,intake-pages,content-files-page,settings-pages,cycle-pages}.ts` 的点名处、`E/stories/issue/editor-loading.spec.ts:1-150`、`h2-h4-notification-cards-page.spec.ts:40-56`、`:155-185`、`:236-298`。
- 交接与文档：`HO/M3-closeout.md`、`M4-closeout.md`、`M5-closeout.md`、`M6-closeout.md` 被引的每一行，P4 记录 `:148`、P4 spec `:270`、`:478`、P2c spec 全文、P2c 记录 `:65`、P5a spec `:387`、`:422`、P5a 记录全文、设计 `:106-113`、`:245`、`:291`、`:307`、`:351`、v0 `:623-644`、P5a 归档台账的 `progress.md:683-687`。

没有核实的：

- oxlint 的 18 条、死成员的 1598 行与 88 行、knip 在各 Task 之后的结果、任何 vitest 与页面故事的时长（都没有运行）。
- `[s3]` 在 Firefox、WebKit 下能否跑通（M5 只在 chromium 跑过）。
- Q34 在页面上是否重现（I5 是按代码推断的“必然”，没有在浏览器里看）。
- Headless 的 Combobox 与 Popover 在 `ModalCore` 里按 Escape 时是否都只关自己（R7 留给 H8 三引擎）。
- 自然 Tab 次序里编辑器是否多出停点（留给 H8 的 `tabStops`）。
- jsdom 里点背景能否触发 Headless 的点外面关闭（T2 的那一格要用，M4 的 `close-policy.test.tsx` 只测了 Escape）。
- `plane/` 的行号只抽了 10 处（都对）；spec §7 的七条差异没有逐条对 Plane 的源码。

## 2. 发现

### 严重

无。

### 重要

| # | 位置 | 发现 | 证据 | 须做的修正 |
|---|---|---|---|---|
| I1 | plan 允许清单的 T2 两条；spec §2.1“标题经 `ModalTitle`” | **两份既有测试照清单改完，每一格都在渲染时抛错。** 两份都把 `InboxIssueCreateRoot` 直接渲染进测试的容器，不在 `ModalCore` 之内。标题一改成 `ModalTitle`，它就是 Headless 的 `DialogTitle`，在 `Dialog` 之外渲染时抛 “`<Dialog.Title />` is missing a parent `<Dialog />` component.”。清单只许删四处（`use-platform-os` 替身、`NoResize`、`stubGlobal`、两个 prop），又规定“这里没写的先停下报告”，所以 T2 一动手就停。换成包进真实的 `ModalCore` 也不是一行：它渲染进 portal，而两份测试都从自己的容器里查（`container.querySelector`、`host.querySelector`）。 | `IN/modals/create-modal/create-root.request.test.tsx:93-103`、`:110`、`:115`、`:124`；`create-root.files.test.tsx:52`、`:105-110`、`:116`、`:124`；`web/packages/ui/src/modals/modal-core.tsx:136-143`；`@headlessui/react@2.2.10` 的 `dialog.js`（`Dialog.Title` 取不到 `DialogContext` 即抛）；M4 的先例是替身：`C/components/issues/issue-modal/fake-dialog.tsx:10-14`、`base.clone.test.tsx:94-95` | 清单的两条各加一行编译跟随，写出原文：对 `@nerve/ui` 的部分替身，只把 `ModalTitle` 换成普通的 `h3`（其余照原模块），断言不改。“弹窗以 `ModalTitle` 为名”“点背景”“Escape”这些要真实 `ModalCore` 的格都在新文件里，经 `InboxIssueCreateModalRoot` 渲染、从 `document` 查。brief 同改。 |
| I2 | plan T7 Step 2“关闭：先登记 `holdEditorImplementation` 再打开弹窗”；spec §2.8 的 `h8-intake-close-page` 一行、§5 的 M6-M7-01 一行 | **在有选中条目的收集箱页上扣不住，写出来的是空证据。** 帮手靠拦住编辑器代码块的请求来扣住；收集箱页只要列表有行就自动选中第一行，详情的 `DescriptionInput` 用的是同一个惰性入口，代码块在弹窗打开之前已经载入，之后弹窗的 `import()` 不再发请求，`held` 一直是空的，编辑器立刻建成。这时“Escape 关空表单”“Discard 关空表单”照样通过（编辑器已就绪、表单没改动），而“提交被拒”那一步会失败——若故事没有断言 `held.length > 0`，前两步就在没扣住的情况下通过，M6-M7-01 的关闭证据是假的。plan 要的“先跑一个期望错误的版本”也查不出这一点。 | `IN/root.tsx:74-85`（自动选中）；`IN/content/detail.tsx:15`、`:115`；`C/components/editor/rich-text/editor.tsx:11-12`（`lazy(() => import("./editor-implementation"))`，新建弹窗的 `issue-description.tsx:18` 同一个）；`E/stories/issue/editor-loading.spec.ts:35-53`（按请求扣住）；C10 的写法：打开页面之前登记（`:59-64`、`:107-110`），再断言 `held.length > 0` 与表单里没有 `[contenteditable]`（`:116-117`） | spec §2.8 与 plan T7 写明：关闭的故事在页面载入之前登记扣住（C10 的次序），并在每个关闭、提交的步骤之前断言 `held.length > 0` 且弹窗里没有 `[contenteditable="true"]`；说明页面上不能先有编辑器建成（空的收集箱，或详情的编辑器同样被扣着）。这一步的红色版本定为“去掉扣住之后，‘提交被拒、没有 `POST`’一步失败”。 |
| I3 | spec §2.5“嵌入的生命周期”、§8 的 O-2；plan T6b 的 Files 与允许清单 | **O-2 的落点没有机制，落点链不完整。** ① “卡片不在了则通知列表的区域（`useLeavingFocus` 的 `region`）”：这个 ref 在列表组件里面，元素没有名字也没有标记，预览（`WN/root.tsx`，列表的兄弟）够不着；要够着就得改 `WN/sidebar/list.tsx` 或 `page-state.tsx`，两者都不在 T6b 的文件清单与 spec §2.9 的表里，它们的测试也不在允许清单。② 列表读失败或还在读时根本不画这个区域，落点链到这里断在 `body`；预览自己的 section 今天不可聚焦。③ 选中跨页签与筛选保留，所以“卡片不在了”是常见的情形，不是边角。④ 时序没有写：删除弹窗卸下时 Headless 在一个微任务之后把焦点还给打开它的菜单按钮，而那个按钮随嵌入一起卸下，落点必须排在它之后（P5a 的 D2 与 `IN/landing.ts` 处理的正是这个次序）。⑤ O-1 乙新加的失败界面也有“关闭”（清选中），不在 O-2 列的三种里，按下之后焦点同样到 `body`。 | `WN/sidebar/list.tsx:46`、`:48-50`、`:53`、`:65`、`:70`；`WN/sidebar/use-leaving-focus.ts:54-55`、`:74-96`（只在卡片离开列表时落焦点）；`WN/root.tsx:41`；`WN/page-state.tsx:51-53`；`IN/landing.ts:47-48`、`:71-82`；`IN/content/header/detail-focus.ts:44-45`；`C/components/issues/peek-overview/error.tsx:36-43`；plan `:242`；P5a 记录 §3 的 D1、D2 | spec §2.5 写出嵌入清掉选中之后的落点怎么实现（建议通知页自己的一份落点，同 `IN/landing.ts` 的形状）：由一个不随预览卸下的组件在“选中被清掉”的那次提交之后一个微任务里落焦点；链是那张卡片的主按钮（`[data-notification-id] button`，列表的既有约定）→ 列表的区域 → 列表没画出来时预览的区域（给它 `tabIndex={-1}`，它已有名字）。失败界面的关闭也走它。把要动的 P4 文件与它们测试的编译跟随写进 T6b 的 Files 与允许清单；单测每一环一格、各一个变异（现在只有 `t6b-mut-landing-none` 一个）；T9 的故事在键盘、指针、800 宽之外加“先换页签再关闭”（卡片不在）。 |
| I4 | spec §2.5“判定”；plan T6b Step 1、允许清单的 `WN/root.test.tsx` 一条、性质表、“页面故事的教训” | **判定的读被再验证时，预览会回到“加载”、撤下它下面的浮层或嵌入；jsdom 里也没有一格能看见。** ① spec 让预览照 `WN/root.tsx:72-86` 的写法读、“读落定之前预览说在加载”；那个写法的落定是 `answered && !read.isValidating`。预览与浮层共用一个键，SWR 的 `isValidating` 是按键共享的：网络恢复时浮层自己的 hook 再验证（会话取数默认随重连重读），或用户按浮层的重试（`mutate()` 走这个键第一个登记的 hook，即预览的），预览都会看到 `isValidating` 为真。照字面实现，这时预览画回加载，浮层连同没发出的评论被卸下重挂——P4 今天没有这个问题，是对每一张通知的回归，也正是 P5a 整审 I1 的形状。嵌入一支只要预览自己的读不关重连重读，同样。“判定一次”写了，但性质表的 `t6b-mut-rejudge` 只钉“随 canonical 改判”，plan 的教训一节也没有 I1。② 允许清单让 `WN/root.test.tsx` 把读的 hook 连同浮层一起替身（hook 从 `peek-overview` 的桶文件导出，替身只能整模块）。于是“以这次的读为准”“非收集箱只一个读”“重连不重读”三条在 jsdom 里都只对着手写的假 SWR 成立；P4 的 m1 一格是在真实的 SWR 上钉的。“只一个读”只剩检查点 1 的页面核对，那时 T6b 已经提交。 | `C/lib/use-session-swr.ts:26`、`:41-46`；`swr@2.4.2`（默认 `revalidateOnReconnect: true`、`dedupingInterval: 2000`；重连时每个 hook `softRevalidate`；不带数据的 `mutate()` 调 `revalidators[0]`）；`WN/root.tsx:72-89`；`C/components/issues/peek-overview/root.tsx:93-96`、`:118`；`IN/content/root.tsx:106-110`、`:128`（P5a 的修法是关重连重读）；P5a 记录 §9 的 I1（`d673c9f3`）与 `IN/content/root.after-read.test.tsx`；`WN/root.test.tsx:21`（“SWR is real”）、`:45-57`、`:182-203` | spec §2.5 改三处：预览的那次读以 `{ revalidateOnMount: false, revalidateOnReconnect: false }` 加 `mutate()` 发出；判定在第一次成功落定时记下，之后不再看这个键的 `isValidating` 与 `error`（浮层与嵌入两支都是）；读的 hook 放进 `peek-overview/` 下它自己的文件（收入 swr 名单，读者仍是 `PeekContent` 与预览），这样通知的测试照旧替身浮层组件、用真实的 hook 与真实的 SWR，只替身取数。T6b 加格：SWR 留着较早回答时以这次的读判定；非收集箱的预览加一个调真实 hook 的浮层替身，取数恰一次；越过去重窗口之后发 `online`，预览不再取数，浮层（或嵌入）是同一个元素、输入还在；浮层的重试不卸下浮层。变异 `t6b-mut-reconnect-rereads`、`t6b-mut-validating-unmounts`。教训一节补 I1。 |
| I5 | spec §2.6（R19、R20）、§1、§6；plan T8b 的 Step 2–4、检查点 1 的定义、性质表末行 | **Q34 按代码是必然重现的，修复不是“或有”的一步。** T8b 的故事扣住第一次 `PATCH` 的回答再点第二位成员。多选是受控的，新集合从渲染出的值算；canonical 只在确认时变；调用方把整集原样交给写的层，写的层的 `changeBody` 把 `assignee_ids` 原样带过；队列一次发一个。所以第二次的请求体必然是只有第二位成员的集合，最终集合丢掉第一位。这不靠时序。于是：R20 (b) 的修复是确定要做的产品改动，落在检查点 1（“产品代码完成”）之后；Task 数是 14；它要动的文件与调用方现在就能列出，不必等重现。另：“一处改好所有调用方”做不到——写的层只能在轮到时把增删合到所持集合上，增删相对“被点时显示的值”，只能由每个调用方算；有下拉框的调用方是五个，其中两个在列表上，不在检查点的五个页面配置里。 | `IN/content/issue-properties.tsx:47-48`、`:81-83`；`web/packages/ui/src/dropdowns/custom-search-select.tsx:277-286`；`C/store/issue/helpers/base-issues.store.ts:73-75`；`C/store/issue/issue-writes.ts:102-134`（`makeBody(held)` 已在轮次里算）、`:142-144`；`C/store/issue/issue-write-fields.ts:70-84`（`assignee_ids: change.assignee_ids`）；`web/packages/types/src/issues/issue-change.ts:17`；已有的先例：命令面板的指派在轮到时从所持集合算（`C/components/power-k/ui/pages/context-based/work-item/mutation-commands.ts:36-39`、`writes.ts:19-39`）；其余四个调用方：`peek-overview/properties.tsx:93`、`issue-detail/sidebar.tsx:97`、`issue-layouts/properties/all-properties.tsx:73`、`issue-layouts/spreadsheet/columns/assignee-column.tsx:32` | plan 把修复写成 T8c（主控派、方案先看的流程照旧），列出文件：`issue-change.ts`（加负责人的增删）、`issue-write-fields.ts` 的 `changeBody`（收所持的工作项）、`issue-writes.ts:143`、命令面板的 `writes.ts:45`、五个调用方；请求体仍是整集。修复之后另跑 chromium 的 m4-detail 与 m4-list（列表行的负责人下拉框）。检查点 1 的说明改为“除 T8c 之外的产品代码”，或把 T8b 的连点一步与 T8c 提到检查点 1 之前。spec §2.6 的“一处改好所有调用方”改成实际的形状。第二个“不扣住、快速连点”的版本只能靠时序，写明它不作证据。 |

### 次要

| # | 位置 | 发现 | 须做的修正 |
|---|---|---|---|
| m1 | spec §2.1“提交成功之后”、§5 的 M7-04 一行；plan T2 | **M7-04 点名的“换宿主迟到答”与 IME 没有处理，也没有写不处理。** `HO/M4-closeout.md:69` 要统一的有“嵌套 Escape/IME、拒绝与换宿主迟到答”。提交的跟进只核会话（`create-root.tsx:143-159` 经 `followInSession`，`C/lib/in-session.ts:40-53`）：提交在途时弹窗关不掉，但后退、前进不受弹窗约束（v0 `:642`），项目的布局随之重挂、弹窗卸下，迟到的成功照样 `navigate` 回原项目的收集箱并提示成功。M4 的弹窗以表单的范围挡住（`issue-modal/base.tsx:93-98`、`:211`）。IME：文档级的 Escape 去掉之后由 `ModalCore` 管（`modal-core.tsx:99`），没有一格。 | T2 二选一并写进 spec：给跟进加表单的范围（卸下或换了项目之后不跳转、不提示；一格、一个变异），或在 §5 写明这一项不随 P5b 关闭、交 M8。IME 加一格（输入法的 Escape 不关、不弹确认框），或在 M7-04 的处理结果里写明由 `ModalCore` 保证。 |
| m2 | spec §2.1 关闭规则的表第一行“Escape 与点背景什么都不做” | 提交在途时 `ModalCore` 仍带着 `handleClose`，Headless 在调它之前先把有焦点的控件 `blur`（`modal-core.tsx:80-100` 的注释说的就是这件事，`AlertModalCore` 因此在途时传 `undefined`，`alert-modal.tsx:76`）。之后 `keepFocusIn` 把焦点给对话框本身，原来“提交按钮恢复可用时把焦点还给它”的等待被换掉（`keep-focus.ts:32-48`）：被拒之后焦点留在对话框上，不在任何字段里。不落到 `body`，M4 的 `draft-issue-layout.tsx:108` 同形。 | 取一种并写明：在途时不给 `ModalCore` `handleClose`（表单把在途告诉 `modal.tsx`），表里那句话才成立；或照 M4 的形状，把表改成实际的结果，H8 在途的一步断言 Escape 之后与被拒之后焦点在弹窗里。 |
| m3 | spec §2.1“名字” | ① `issue-title.tsx:48` 的 `required` 要删，spec 没有写：留着它，空名字由浏览器的原生校验拦下，`submit` 事件不发，`FieldError`、`aria-invalid` 与 `useRefusedFocus` 都不走（M4 的输入框没有它，`title-input.tsx:58-74`；`HO/M3-closeout.md:62` 点了它）。② “规则同 M4（`title-input.tsx:39-45`）”不准：M4 的超长按没去空白的字符数（`:43`），本段按去了空白的（同服务端 `domain/issue.go:184`）；“有改动”里 M4 把只有空格的名字算作改动（`draft-issue-layout.tsx:53`），本段不算。规则本身是对的。 | ① 写明删 `required`，H8 与单测的“空白”分空与只有空格两格。② 两处“同 M4”改成“同合同”，写明与 M4 的差别。 |
| m4 | spec §2.1“测试”；plan T2 Step 1、Review Focus 2 | ① 新文件 `create-root.close.test.tsx` 要装约 35 格（关闭规则、确认框、结构、提交、名字、开关、Tab、字面量），加上与 `create-root.request.test.tsx:16-108` 同量的夹具，过 400 行是大概率的。② Review Focus 2 的“每次打开从初值起”没有格也没有变异；它只在离场的过渡（200 毫秒，`modal-core.tsx:104`、`:113`）结束之后成立，过渡当中再打开是同一份表单。 | ① 按关注点先拆成两三个文件加一份共用的夹具（先例 `IN/fake-intake-page.tsx`）。② 加一格（填字、确认放弃、再打开是空的）与一个变异；过渡当中的那一种写一句或给内容一个随每次打开变的 key。 |
| m5 | spec §2.8 的 `h11-intake-images-page` 一行；plan T8a Step 1、T3 Step 1 | ① “接受之后照 `issue.update`：访客提交者只读、成员可传”在收集箱页上做不出：已接受的条目在收集箱的详情里对谁都只读（`IN/content/detail.tsx:50-56` 只认待处理与暂缓，`:131`）。“成员可传”只能在浏览页的通用详情上看到（已接受的不重定向）。② T3 的“详情与内容视图对矩阵的每一格给同一答案”只对在收集箱里的格成立：已接受与 `intake` 为 `null` 时详情给假、`canWrite` 按角色给真，这是设计如此；把这两格也放进“同答”的表，`t3-mut-accepted-intake-rule` 反而让它变绿。 | ① 故事写明这一步在浏览页（`/browse/…`）上做。② spec §2.2 与 plan 把“同答”限定为待处理、暂缓、拒绝、重复四种状态的格，已接受一格分别断言两边。 |
| m6 | spec §2.2“后果”一条；plan 性质表的 T3 | `canWrite` 的读者列漏了一个：删除附件的命令经 `canRemoveAttachment` 把关（`C/store/asset/content-deletion.ts:45`）。行为随 `canWrite`，不出错。另：“`canRemoveAttachment` 随之”有格、没有变异；编辑器的上传能力在 `EditorAssets` 建成时定一次（`C/hooks/editor/editor-assets.ts:36`），之后状态变了靠详情的 `disabled` 与每一步的 `canWrite` 兜着——这两层的关系值得写一句。 | 把 `content-deletion.ts:45` 补进列表；加变异（管理员在已拒绝的条目上仍可删）。 |
| m7 | spec §2.2 末句、§2.8 的两份文件故事 | ① 资产的写被拒为 `issue.intake_closed`、`issue.intake_disabled` 之后页面不重读，所持的详情仍是待处理，上传的控件还在，再传再被拒；内容修改遇到同一个码是重读的（`C/store/issue/issue-write-intake.ts:9`）。② “在途的图片以 `issue.intake_closed` 被拒”要写明扣的是哪一步：`holdAnswer` 扣的是回答，`createAsset` 在服务端已经做成，被拒的是之后的 `completeAsset`。 | ① 写进 spec 的已知行为与 M8 的候选（或说明为什么不重读）。② 故事写明扣住 `createAsset` 的回答、另一会话拒绝、放开之后 `completeAsset` 答 409，并把这个 409 写进 `watch.apiFailures` 的期望。 |
| m8 | plan T6a Step 1、T7 Step 1（两处“逐字搬”） | ① `offeredOn`（`use-intake-actions.ts:111`）与 `OpenModal`（`:61`）今天都没有导出，搬到新文件之后 hook 要导入它们，必须加 `export`；`t6a-move-identical` 的比对要容这两处。② `editorChunks` 在故事里另有两个读者（`editor-loading.spec.ts:160`、`:215`），要一起导出；它按 `import.meta.url` 找构建产物（`:17` 的 `../../../web/…`），从 `E/fixtures/` 起要少一层（`../../web/…`），逐字搬过去读不到目录。 | 两处都把“逐字”之外允许的改动写明。 |
| m9 | spec §2.5“打开”、R25 (3)；plan T5、T6b | ① 嵌入里“打开”与“复制”的地址不同了，而动作模型只有一个 `link`，注释写着两者共用（`use-intake-actions.ts:69-70`、`:183-189`），“打开”有两个读者（`:255` 的 `take("open")`、`header/root.tsx:104-112`）。spec 没有写第二个地址放在哪。② “收集箱的地址由 `IN/address.ts` 一处给出”：新建弹窗成功之后的跳转手写着同一个地址（`create-root.tsx:146`）。 | ① 写明动作模型多一个“打开的地址”，两个读者都读它，复制仍读 `link`。② T5 把 `create-root.tsx:146` 改为读 `intakeAddress`（字符串不变，`create-root.request.test.tsx:160-162` 不用改），或把那句话限定为“选中某一条的地址有两个新读者”。 |
| m10 | spec §2.5“判定”“嵌入的生命周期”；plan Review Focus 7 | 预览只以通知 id 为 key（`WN/root.tsx:47`），收集箱的选中以会话与权限身份为 key（`IN/content/root.tsx:77`）。“判定一次”的状态若只按通知记，换了会话或权限身份之后沿用上一代的判定；Review Focus 7 列了“换代之后旧的回答不写进新的预览”，Step 1 没有格，性质表没有行。 | spec 写明判定的状态按（通知、权限身份、`loginId`）记，身份变了重新读、重新判定；T6b 加一格与一个变异。 |
| m11 | plan 允许清单的 `WN/root.test.tsx` 一条、性质表“读失败的一支”；spec §2.5“读失败时” | ① 清单只点了 `:45-57`；判定要读 canonical 的详情，`use-issue-detail` 的替身只有 `peekIssue` 与 `setPeekIssue`（`:23-27`、`:44`、`:73-75`），也要跟随。② `IssuePeekOverviewError` 不从桶文件导出（`peek-overview/index.ts:7` 只导出 `root`），预览得按文件路径导入；它里面的 `DetailResourceError` 经 `useDetailScope` 读 `rootIssueStore`（`resource-error.tsx:23`、`use-detail-scope.ts:12`），通知的测试要给得出。③ 性质表说变异“失败时渲染浮层”由“读两次的一格”杀死：浮层是替身时读不到第二次；而且 `swr@2.4.2` 在一个键已有 hook 挂着且带着错误时，新挂的 hook 不在挂载时再读（`shouldDoInitialRevalidation`）。spec 的推断“失败之后浮层挂上会再发一次读”在这个版本下不成立；裁定乙的理由不靠它，不受影响。 | ① ② 写进清单与 T6b 的 Files。③ 那一行改为由“浮层没有挂上、失败界面是预览自己的”杀死，另加一个变异（重试落定之后不再判定）；spec 那句推断照源码改写。 |
| m12 | plan 允许清单的 H2–H4 一条；plan T9；spec §2.8 | ① H2–H4 改成嵌入之后，紧接着的一步是 bob 在另一个登录里离开项目（`h2-h4-notification-cards-page.spec.ts:270`）。嵌入要读两整包（判定的读与嵌入自己的读，每包是工作项加八个附属读，`issue.store.ts:71-106`）再加描述版本；这些读没答完就离开项目，会多出 403 或 404，而故事末尾对 `watch.apiFailures` 是精确比对（`:292-296`）。这是 I9、H6 那一类竞态，清单是“只照写的改”，要把等待写进去。② 嵌入的关闭按钮名为 “Close”，toast 的关闭按钮也是（`web/apps/web/app/provider.tsx:35` 的 `closeLabel={t("close")}`）：按名字在整页定位，有 toast 时会撞。 | ① 清单的那一条加：点卡片之前登记嵌入最后几个读的 `waitForResponse`，离开项目之前等它们答完。② H2–H4 与 T9 的故事把 “Close” 限在嵌入的区域里定位（教训一节加一句）。 |
| m13 | plan“守卫”一节；spec §2.3、brief `:55` | ① `IN/content/widgets.tsx` 读项目与角色（R10 的规则写在它里面），照规矩要收入 `project-address-route` 并带样例；plan 只写了 T6a 搬出的文件。若采纳 I4 的新文件，收入 swr 名单。② 行号的小偏差：`detailReads` 在 `E/fixtures/intake-pages.ts:194-199`（spec 写 `:193-198`，brief 写 `:193`）；`custom-search-select.tsx` 的多选在 `:277-286`（写 `:276-284`）。 | 补进 plan 的守卫一节；改行号。 |
| m14 | plan T6b | 一个提交里有判定与失败一支、嵌入的页头与关闭按钮、动作模型（分诊之后不移、打开的地址）、通知页的落点、读的 hook 的导出、两份新测试、两份既有测试与一份故事的改动、15 个以上的变异，是本段最大的 Task；按 knip 的第一个读者规则它确实拆不成两个提交。 | 同 re-94 m11：保持一个提交，plan 里写成两个可分别审查的步骤（收集箱一侧：端口的嵌入一支；通知一侧：判定、失败与落点），各有自己的 RED 收据，审查分两次看。 |

### 观察

- **O1 嵌入的读是两整包，不是“多一个 GET”。** 判定的读就是浮层的取数（`fetchIssue`：工作项、表情、链接、动态、评论、子工作项、关联、项目的状态，`issue.store.ts:71-106`），落定之后嵌入再以自己的 `mutate()` 读一遍。收集箱条目的通知从点开到看见内容约是今天的两倍，读的数目也是两倍。这是 B-16 (a) 与“每次挂载读自己的”两条规则合起来的代价，建议写进 spec §2.5 与阶段记录；T9 的“恰两个 `GET /api/v0/issues/{id}`”照旧可断言。
- **O2 “非收集箱只一个读”在 `swr@2.4.2` 上成立，靠的是时间窗。** 不带数据的 `mutate()` 先删去重标记、发新请求，回答之后标记再留 `dedupingInterval`（2000 毫秒）；浮层挂上时的再验证带去重，在窗口里并入已答的那一次，不发请求。窗口从回答算起，正常只隔几帧；主线程卡两秒以上才会多发一次。页面上的“恰一个”因此是带时间窗的断言，偶发时先看这里，不要去调去重间隔（v0 §7.7 只许列表缩短它）。
- **O3 R14 的两个后果，建议写进 spec 与 plane-diff。** 收集箱关着时，未被接受的条目经地址到不了任何详情（落到停用页），只剩通知的嵌入能看见它；到期之前的暂缓条目被送到“未处理”页签，而这个页签默认只列待处理（`C/store/issue/intake/filters.ts:50`），条目选中了、列表里没有它那一行。都与裁定一致（B-14），不是错。
- **O4 链接折叠栏里的加号按钮没有名字**（`issue-detail-widgets/links/quick-action-button.tsx:31-33`、`links/root.tsx:37`）：M4 的共用件，收集箱详情从此也显示它。本段的“链接按钮有名字”只管新加的那一个；这个交 M8 的全应用核对。
- **O5 嵌入第一次把收集箱的详情放到项目的布局之外。** 标签、成员、状态的下拉框都自己读目录（`label-select.tsx:34`、`member/dropdown.tsx:40`、`state/dropdown.tsx:30`），接受弹窗的迭代与模块也是，按代码不依赖项目包装层的取数。建议 T9 的故事在嵌入里各断言一次已有的标签与负责人显示出来、接受弹窗的状态有默认值。
- **O6 web 作业的报告线。** main CI 45197451256 的 web 作业在 `.376` 上已是 959 秒；本段净增 vitest，慢 runner 上越过 960 秒几乎是确定的。R2 的判别法可用（基线本身是慢 runner 的，快 runner 上三个比值都小于 1，照样比三者是否相近）。
- **O7 通知页上浮层关闭之后焦点在 `body`**（`peek-overview/view.tsx:97-102`：嵌在页面里时不还焦点），O-2 明说不改。本段之后同一页上“关嵌入落在卡片、关浮层落在 `body`”不一致；I3 的落点做成通知页自己的一份之后，浮层的关闭接上它只是一行。是否顺手做由主控定，不做则记给 M8。

## 3. 逐项回答

### 3.1 行号与事实

spec、plan 与 brief 引的代码、合同、交接与文档的行号核了约 150 处，Task 依赖的都核了，全部存在、说的是所称的内容；偏差只有 m13 的两处（各差一两行）。`plane/` 抽的 10 处都对。两处“事实”与所称不符：“规则同 M4”（m3）、“失败之后浮层挂上会再读”（m11）。

### 3.2 合同

- 谁能在收集箱条目上传图片与附件、在哪些状态：同 spec §2.2（P2c spec `:13-20`、`:41-52`）。管理员是 `Grant.ProjectAdmin`，含是项目成员的工作区管理员，与网页的角色读法一致（`decide.go:85`、`permissions.store.ts:68-75`）。拒绝、重复的条目上删除已绑定的附件对管理员也是 409，`canRemoveAttachment` 随 `canWrite` 是对的。评论宿主不变，合同也是这样。
- 链接（R10）：三个操作都是 `issue.update`，没有收集箱的三个码，已归档的项目 409；P2a 有意不加守卫（P2a spec `:90`、`:125`），设计 `:109` 同。
- 码：`issue.in_triage`、`issue.intake_closed`、`issue.intake_disabled` 在 `C/lib/error-messages.ts:87-89`；提交的码里有 `issue.intake_disabled`、没有 `issue.intake_closed`（`IY:888`）；内容修改的码含 `asset.not_found`、`asset.not_uploaded`（`IY:1007`）。
- 提交的端点与请求体：P5b 不改；名字的规则是去空白之后 1–255 个字符（`IY:2365-2367`）。

### 3.3 R9 与 R20 (b)

- **R9**：方向成立，没有发现会坏全应用上传的地方。`inIntake` 对缺 `intake` 键的旧夹具为假，既有的格走原来的一支；谓词放在 `status.ts` 不成环。缓解基本够（矩阵、七个变异、T3 之后的 m5-content、每个检查点三引擎的 m5-content），要补的是 m5（“同答”的范围、接受之后的故事在哪一页）、m6（漏列的读者与一个变异）、m7（被拒之后不重读）。漏掉的读者只有 `content-deletion.ts:45`；评论的两个表单（`comments/comment-create.tsx:108-183`、`card/edit-form.tsx:95-168`）读的是评论宿主的一支，不受影响。
- **R20 (b)**：方向成立，而且写的层已经在轮次里算请求体（`makeBody(held)`），模块集合与命令面板的指派是现成的先例。问题在计划：它按代码必然重现（I5）。变异 `t8b-mut-assignees-at-call` 用假的队列能确定地杀死，不靠竞态。

### 3.4 嵌入

- **端口取代两个布尔 prop**：做得出。`useIntakePageState` 的读者里只有动作模型与 `IntakeNavigation` 会进嵌入（`use-intake-actions.ts:167`、`navigation.tsx:28`），前者改读端口、后者嵌入时不画；`fake-intake-page.tsx` 渲染整页，不用跟随；T6a 的允许清单两处是全的。
- **去重**：见 O2，成功时成立。`WEB_SWR_CONFIG` 设 `revalidateOnMount: true`、`revalidateIfStale: true`、`revalidateOnFocus: true`、`errorRetryCount: 3`；`useSessionSWR` 盖掉聚焦重取与出错重试，只许三个选项（`dedupingInterval`、`revalidateOnMount: false`、`revalidateOnReconnect: false`），重连重读默认开着——I4 的来源。
- **焦点不落到 `body`**：分诊之后的落点（`detail-focus.ts`）现成；删除、关闭、读不到之后的落点没有机制（I3）；新建弹窗在途时的 Escape 落在对话框上（m2）。
- **会话分代**：键经 `sessionKey` 带 `loginId`、浮层的键带权限身份，都对；嵌入里的跟进经动作模型的 `command`（范围与会话都核）。缺的是判定状态的代（m10）与新建弹窗的范围（m1）。

### 3.5 新建弹窗

关闭规则与 M4 的 `requestClose` 逐条对得上（在途、引用存在而未就绪、有改动才确认、提交仍要求就绪、Discard 在途停用），去掉的只有草稿。“有改动”的规范化对（`isEmptyHtmlString`，`start_date` 的默认值不算）。名字的规则对合同。Tab：今天的正 `tabIndex` 次序本来就是 DOM 的次序，去掉之后只少了“开关停两次”。要改的是 I1、m1–m4。

### 3.6 重定向与“打开”

没有循环：重定向只在浏览页、只对未被接受的；收集箱页对未被接受的不再给“打开”，复制的地址经重定向落回收集箱页，嵌入的“打开”直接到收集箱页。`replace` 之后后退回到来处；工作项 id 的入口是两次 `replace`。读用的是这次访问自己的回答（`layout.tsx:65-76`），不会拿较早的回答判定。`offeredOn` 的改动一行换一行，不变长。H20 的故事不断言 “Open work item”，只有 H10 `:58` 与 `header.test.tsx` 的矩阵跟随，清单是全的。后果见 O3。

### 3.7 plan

- **次序与大小**：次序成立（谓词在图片与附件之前，纯搬移、端口、嵌入三步）。偏大的是 T2（m4）与 T6b（m14）；T8c 要补（I5）。
- **不许变长的文件**：19 项的行数逐个量过，都对。
- **knip 的第一个读者**：成立；m8、m9、m13 各有一处要补。
- **允许清单**：T5、T6a 的是全的；T2（I1）、T6b（m11、m12）不全。
- **性质与变异**：没有只能靠竞态杀死的行。缺行：重连与再验证（I4）、落点链的每一环（I3）、每次打开从初值起（m4）、`canRemoveAttachment`（m6）、判定状态的代（m10）、失败之后的重试（m11）。
- **检查点**：五个配置在 spec、plan、brief 与 `final.sh` 里一致；`final.sh` 的循环对，配置名 `playwright.m5-content.config.ts` 对，它自己排除 `[s3]`；`gates.sh` 经 `make e2e` 构建 S3 的辅助程序，之后的 m7-pages 能跑 `[s3]`。检查点 1 的定义见 I5。
- **故事**：六个新文件的名字都合 m7-pages 的 `testMatch`，标题合 `grep`，`pageFileStory` 加的后缀不碍事；`[s3]` 在 m7-pages 里三引擎、在默认配置里 chromium 一次。要改的是 I2、m5、m7、m12。
- **P5a 的教训**：真实引擎才查得出的焦点（已写，I3 补机制）、动作之前先等在途的读（已写，m12 补 H2–H4）、持有详情宿主的读不随重连重读（没写，I4）。

### 3.8 交接

spec §5 的表逐行核过，行号都是今天的。写“关闭”的各项由 Task 覆盖，有三处要改措辞或补证据：M6-M7-01 的页面证据要先修 I2；M7-04 的“收集箱部分关闭”不含它点名的“换宿主迟到答”与 IME（m1）；M5-M7-01 的“接受之后”一步要换到浏览页（m5）。M7-05、M6-M7-05 保持 open 是对的（去掉四个读者之后 `getTabIndex` 只剩 `C/components/views/form.tsx:75`）。

## 4. 核实为正确的事实（摘要）

- 新建弹窗：两个文档级的 Escape（`modal.tsx:29-34`、`create-root.tsx:86-99`）、死的 `modalContainerRef` 与 `ResizeObserver`（`:62`、`:101-116`）、没有读者的宽度状态（`modal.tsx:25-27`、`:40`）、`div role="button"` 套开关与两条 `oxlint-disable`（`create-root.tsx:2-3`、`:194-202`）、四处英文字面量、`getTabIndex` 的四个读者与只为它读的 `usePlatformOS`，都在 spec 说的位置；`page-header.tsx:72-80` 是唯一的打开者。
- `ModalCore` 的 `handleClose` 同时管 Escape 与点背景（`modal-core.tsx:80`），最内层的对话框认领 Escape（`:96-100`），关着时不渲染内容；`AlertModalCore` 以 `ModalTitle` 为名、默认按钮文字是英文（`alert-modal.tsx:62-66`）。
- M4 的先例：`draft-issue-layout.tsx:50-66`、`:68-90`，`title-input.tsx:56-81`，`form-footer.tsx:66-77`，`useRefusedFocus`（`refused-focus.ts:29`），`useFormRefusal`（`form-refusal.ts:58-75`）。
- 文案：要用的已有键都在（`discard`、`common.cancel`、`close`、`title`、`create_more`、`title_is_required`、`errors.field.too_long`、`editor_is_not_ready_to_discard_changes` `common.json:304`、`issue_created_successfully` `:293`、`toast.success`、`toast.error`、`issue.add.assignee`、`.due_date`、`.link`、`inbox_issue.modal.title`）；两个新键与 `inbox_issue.modals.*` 的既有键是兄弟，不成前缀。
- `tab-indices.ts`：`:7-21`、`:26`、`:31` 是收集箱一组；删去之后枚举与函数还有视图一个读者。
- `ContentView.canWrite`（`content-view.ts:62-76`）对 `issue` 宿主只看角色 15、20；`new_comment`、`comment`、`draft` 在它之前返回；`canRemoveAttachment`（`:77-92`）先看 `canWrite`。`editableBy` 在 `IN/content/detail.tsx:44-57`，访客只改名字与描述在 `:171`。`ConfirmedContentNotice` 读宿主的输入 context（`detail-host.tsx:64-73`），M4 的详情与浮层都画在宿主之内。
- 附件默认收起（`issue-details/root.store.ts:84`），展开才读；`useIssueImages` 挂载时不发请求；H9 的读表只数 `/api/v0/issues/{id}` 之下的读，加图片与附件不改它。链接的三个组件、弹窗与 `useLinkOperations` 都只要宿主的弹窗 context，收集箱的正文本来就在宿主里（`IN/content/root.tsx:163`）。
- 浏览页：这次访问自己的读与 `pending`（`layout.tsx:59-76`）、页头与 `Outlet`（`:81-88`）；工作项 id 的入口读到之后 `replace` 到浏览页（`issues/(detail)/[issueId]/page.tsx:31-44`）。
- 通知预览：以通知 id 为 key（`WN/root.tsx:47`）、有角色才设浮层（`:93-95`）、`clearSelection`（`page-state.tsx:68-69`）；浮层的键与取数（`peek-overview/root.tsx:91-96`）；`InboxContentRoot` 的四个 prop（`IN/content/root.tsx:39-52`）、自己的读（`:106-123`）、读不到与不在收集箱两支（`:132-158`）；动作模型的第四个参数（`use-intake-actions.ts:152-156`）、读页面状态（`:167`）、邻行（`:196-200`）；`useIntakePageState` 在 provider 之外抛错（`page-state.tsx:51-54`）。
- 写的层：`updateIssue(id, makeBody(held), …)` 在轮次里算请求体，收集箱里的条目在轮到时改道（`issue-writes.ts:102-134`）；`changeBody` 以增删算模块集合（`issue-write-fields.ts:70-84`）；v0 `:638` 是“请求体轮到时才算”。
- 门禁：web 的上限 18（`package.json:10`），e2e 与 `@nerve/constants` 为 0；swr 名单 `:162` 是浮层、`:198` 是通知预览；`K` 的 phase 是 `M7/P5`；`old-intake-urls` 不碰网页内的 `/intake` 链接。
- e2e：`playwright.m7-pages.config.ts:10-11` 的 `testMatch` 与 `grep`；`pageFileStory`（`file-story.ts:36-43`）；`intakeWorld`（`intake-pages.ts:32`）、`browserDay`（`:104`）；`holdAnswer`（`settings-pages.ts:74`）、`expectPopupBesideButton`（`:240`）、`tabStops`（`cycle-pages.ts:315`）；`expectNoTriage` 的三个读者看的都是待处理的条目（`h10-intake-triage-page.spec.ts:217`、`:220`、`:226`）。
- T1 的回填：P5a 归档台账的末四行与 plan T1 写的数字一致（终检 `9983ed46`、分支 CI 45189969560 的 968.9 秒与 80.7%、noff `ab86965f`、main CI 45197451256 的 93 / 818 / 17 秒）；设计 `:307`、`:351` 是要改的两行。
- Plane：浏览页只送待处理与暂缓、以 `push`（`page.tsx:88-92`、`base.py:1317-1325`）；通知的 `is_inbox_issue` 不含拒绝（`notification/base.py:58-62`）；嵌入时没有上一条下一条与键盘（`inbox-issue-header.tsx:231-235`、`:311`）；新建弹窗的文档 Escape 与超长停用（`modal.tsx:29-34`、`create-root.tsx:238-248`）。

## 5. 需要负责人的事

无新的。五条重要发现都是网页一侧的写法与计划，属主控的局部裁定；没有一条要改合同、服务端或 CI 的预算与结构。照已有的约定，R9 与 R20 (b)（按 I5，它会发生）是动了别的 M 网页共用代码的高风险局部裁定，阶段记录与 M7 收官报告里向负责人列出。O6 的 web 作业报告线照 R2 只报告；要动 20 分钟的期限或作业的结构才需要负责人。

## 6. 处理（主控，2026-11-24）

全部发现都改进了 spec、plan、实施者须知与 `final.sh`（起草者照主控的裁定改；R1–R25 编号不变，新增 R26–R30），没有需要负责人的事。R9 与 R20 (b) 是动了 M5、M4 网页共用代码的高风险局部裁定，阶段记录与 M7 收官报告列出。裁定：

- **I1**：照建议。T2 的允许清单给两份既有的新建弹窗测试一个字面的 `@nerve/ui` 部分替身（`ModalTitle` → `h3`，先例 M4 的 `base.clone.test.tsx:94-95`），要真 `ModalCore` 的格渲染 `InboxIssueCreateModalRoot`、在 `document` 上查。
- **I2**：照建议。H8 的关闭故事在页面载入之前扣住编辑器的代码块、用空的收集箱，每个关闭与提交的步骤之前断言 `held.length > 0` 且没有可编辑的编辑器；期望错误的版本是“不扣住时，提交被拒、没有 POST 的那一步失败”。
- **I3**：照建议，定为 R26：通知页的落点链（卡片的主按钮 → 通知列表的区域 → 预览的区域），新 `WN/landing.ts` 在清空选中的那次提交之后的微任务里落，读失败的界面的“关闭”也走它；链的每一环一格一变异。O-2 的裁定由此细化。
- **I4**：照建议，定为 R27：判定的读放进自己的文件，关掉重连重读（同 P5a 整分支审查 I1 的修法）、每次挂载 `mutate()` 一次；判定在第一次读成功落定时锁住，此后不看 `isValidating` 与 `error`；格用真的 hook 与真的 SWR（只替身取数），两个变异。
- **I5**：照建议，Q34 改为计划内的 T8c（R20 (b)，高风险的局部裁定）：`IssueChange` 加负责人的增删（与整集合在类型上互斥），`changeBody` 在写的轮次里以所持集合合并，请求体仍是整个集合；`changeBody` 的三个调用者与七个负责人下拉框的调用者（起草者核出七个，比预检的五个多子工作项与关联两行）；先在未修的构建上看页面的一步变红；T8c 写修法的计划进台账、主控审过再写代码；chromium 跑 m4-detail 与 m4-list，检查点 2 与终检跑六个页面配置。检查点 1 改称“除 T8c 之外的产品代码完成”。拖放仍以整集合在轮次里算（`assignee_ids` 留在 `IssueChange` 里给拖放），不改拖放的代码与测试：接受起草者的取舍。
- **m1**：R29，新建之后的跟进只在表单挂着时做（挂载的范围），IME 一格；M7-04 逐项写明 P5b 做的与交给 P6a/P6b 的（命令面板的迭代、视图结果，筛选与视图的写后关闭）。
- **m2**：R28，提交在途时 `modal.tsx` 不给 `handleClose`（Headless 2.2.10 的 `use-escape.js` 对被阻止的 Escape 什么也不做，焦点不动、等提交按钮的 `keepFocusIn` 仍在；点背景什么也不关、焦点到对话框元素）；H8 三引擎断言焦点在弹窗里。
- **m3–m14**：照建议全改（m3 删 `required`、空与全空白分两格、“同合同”并写明与 M4 的两处不同；m4 新测试拆成替身与三份测试、重新打开是空的一格；m5 接受之后的一步改在浏览页；m6 `content-deletion.ts:45` 进读者清单、`canRemoveAttachment` 的变异；m7 资产被拒之后不重读详情记为已知的行为与 M8 的候选，故事扣住 `createAsset` 的回答、期望 `completeAsset` 的 409；m8 两处纯搬移允许加 `export`，`editorChunks` 的相对路径；m9 R30 的 `openLink`；m10 判定按通知、权限身份与 `loginId` 为键；m11 `WN/root.test.tsx` 的允许清单、失败组件按文件路径导入、变异的杀法改为“浮层没有挂上”；m12 H2–H4 那一步在 bob 离开之前等嵌入的读、“Close”限定在预览区域里；m13 `widgets.tsx` 进 `project-address-route`、两处行号；m14 T6b 一个提交、两个可单独审查的步骤，各有 RED）。
- **观察**：O1 嵌入读两遍详情的读写进 spec §2.5 与 §6 的代价；O2 去重窗口写进 R16 与实施者须知；O3 两个后果写进 spec §1 与 §7；O4 链接的 “+” 按钮没有名字记为 M8 的候选；O5 T9 的嵌入故事断言标签、负责人与接受弹窗状态的默认值；O6 只报告（R2）；O7 通知页上普通浮层的关闭也走 R26 的落点链（一格一变异），记为 P4 页面行为的改变（spec §7）。
- **规模**：14 个 Task（T1、T2、T3、T4、T5、T6a、T6b、T7、T8a、T8b、T8c、T9、T10、T11），检查点在 T2、T6b、T9 之后。
