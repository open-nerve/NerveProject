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
