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
- **页面**：webhooks 读 `error?.error` 的三处（`web-hooks/create-webhook-modal.tsx`、`web-hooks/form/secret-key.tsx`、`app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/webhooks/[webhookId]/page.tsx`）改经文案表（`errorMessageKey`，M3/P9 spec 第 5 节）；密钥的复制（`web-hooks/form/secret-key.tsx`）改经 `useCopyText`（`core/hooks/use-copy-link.ts`），关键词规则 `copy-hook` 的这个例外（`until: "M8/closeout"`）随之删除；只读工作区的 3 个文件（webhooks 的两个页面和 `secret-key.tsx`）改用 `useRouteWorkspace()` 或路由的 slug，加进关键词规则 `workspace-address-route`（M3 收尾，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 2.12）；加进之前先把列表页传给 `CreateWebhookModal` 的 prop `currentWorkspace`（`app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/webhooks/page.tsx:71`，`web-hooks/create-webhook-modal.tsx`）和 `getCurrentHookAsCSV` 的同名参数（`web-hooks/utils.ts:13`）改名（例如 `workspace`）：规则认的是 `currentWorkspace` 这个名字，值改读路由之后这个 prop 照样命中；页面上写死的英文经 `t()`；`web/apps/web/core/store/workspace/webhook.store.ts` 的 1 条警告（`no-accumulating-spread`）清零。
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
- `@nerve/editor` 不依赖 i18n，它的控件的名字和提示是写死的英文（图片工具栏的"Download image""Zoom in"，表格的"Insert columns""Insert rows"等）。M3 收尾让 `@nerve/propel` 的 `EmojiPicker` 的 `ariaLabel` 必传，编辑器标注块的图标按钮照这种写法名为"Callout icon"（`web/packages/editor/src/extensions/callout/logo-selector.tsx`，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 3 节第 1 条）。编辑器的文字怎样经 `t()`（由应用传入，或编辑器依赖 i18n）一次定，这些名字随之改。
- **关闭条件**：日期的决定写进 M8 的设计并实现；`web/` 的页面上没有写死的英文（M8 的扫描）。

## 4. 可访问性

- Base UI 的 `Toast.Close`（`@nerve/propel` 的 `toast/toast.tsx`）渲染成 `aria-hidden="true"` 而 `tabindex="0"`：Tab 能到、读屏读不到（M3 收尾找到）。
- 页头（`navigation/top-navigation-root.tsx`，每个工作区页面都有）：帮助菜单（`workspace/sidebar/help-section/root.tsx`）和账户菜单（`workspace/sidebar/user-menu-root.tsx`）的按钮里又是按钮，帮助菜单的没有名字；改法照 M3 收尾的项目标签栏的菜单（`navigation/project-actions-menu.tsx`：`CustomMenu` 的 `customButtonClassName` 给菜单自己的按钮样式，内容不再是按钮）。页头的通知链接在 M7 的交接。
- 页头的搜索（`navigation/top-nav-power-k.tsx`）的 4 条 oxlint 警告：`exhaustive-deps` 2 条，`click-events-have-key-events`、`prefer-tag-over-role` 各 1 条。M3 收尾只把它的占位文字改经 `t()`（控制者的裁定，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 3 节第 2 条）；后两条是它的点击区不是按钮，与上面帮助、账户菜单的按钮是同一类改法、同在页头，一起改，改时清零。
- 焦点和输入法只在 Chromium 中逐个核对过：发布之前在 Firefox、Safari 中看一遍焦点的轮廓、弹窗的 Tab 循环和输入法组字时的 Escape、Enter（M3/P11 评审第 6 节、M3/P10 spec 第 5 节；M3 收尾的 C11 看过的部分写在它的评审里）。
- 命令面板打开之后约 200 毫秒之内输入的字会被清空（`power-k/ui/modal/wrapper.tsx` 打开时的重置，Plane 原有；之后的输入留着，M3 收尾的探查）。
- 命令面板的弹窗没有名字：它经 `ModalCore`，里面没有 `DialogTitle`，`ModalCore` 也不收名字；搜索框的清除按钮（`power-k/ui/modal/header.tsx:53`）只有一个图标，没有名字（M3 收尾 Task 2 的评审，Plane 原有）。面板经 `t()` 取一个名字（例如面板里一个 `sr-only` 的 `DialogTitle`，或让 `ModalCore` 收一个名字），清除按钮一个经 `t()` 的 `aria-label`，en 与 zh-CN 同时加。
- **客户端导航之后的焦点**（M3 收尾）：工作区的内容以 slug 为 key 之后，经菜单换工作区时焦点落到 `body`，像一次页面的加载（原来回到菜单的按钮；控制者裁定接受，[收尾 spec](../../M3-workspace-project/specs/closeout.md) 附录 A.8）。全应用的客户端导航之后焦点去哪里（`body`、页面的标题、主区域）是一个全应用的决定：发布之前定下并实现，读屏核对换页之后读出什么。
- **关闭条件**：M8 的评审写明每一项的修法或结论。

## 5. 嵌入的文件加 `ETag`

- `server/internal/platform/webui` 提供的嵌入文件没有 `ETag`，浏览器每次都整个下载（M3/P10 spec 第 5 节，表情的数据最大）。发布的性能核对时决定加不加（M3 收尾的建议）。
- **关闭条件**：加的话，`webui` 的测试核对 `ETag` 和 304；不加的话写明理由。

来源：[M3 设计](../../M3-workspace-project/M3-design.md) 13.2 的 M8 一行；M3/P7b、P9、P10、P11 的 spec 第 5 节和评审第 6 节中交给 M8 的各条；[M3 收尾 spec](../../M3-workspace-project/specs/closeout.md) 第 2、3 节。
