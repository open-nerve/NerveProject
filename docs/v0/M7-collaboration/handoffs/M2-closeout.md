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
