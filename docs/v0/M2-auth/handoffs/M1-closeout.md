---
status: closed
from: M1/closeout
to: M2
created: 2026-09-25
---

# M1 收尾留下的清理：死成员和死 prop、oxlint、主题下拉框的位置

M1 收尾把 knip、tsc 看得见的死代码和包导出都删完了（[收尾 spec](../../M1-frontend-trim/specs/closeout.md)）。下面几项留给后续各 M，做法都是"谁改谁清"：本 M 重写或修改到的代码，在本 M 结束时不再带着它们。

## 死成员和死 prop

类型检查器能找出两类 knip 和 tsc 都不报的死代码：没有代码读取的对象成员（store、service 的方法和属性，接口和类型的字段），和没有调用方传入的组件 prop。M1 收尾结束时，web 里一共 1350 个（成员 898、prop 452），属于本 M 领域（认证、账户、新手引导、实例配置、个人设置、API 令牌，以及 `packages/services`（M2、M5 对接后删除））的有 **66 个**（成员 59、prop 7）。

M1 没有删它们（[收尾 spec](../../M1-frontend-trim/specs/closeout.md) 第 4 节）：
- 本 M 按总体设计 7.2 重写这一领域的 types、services、stores 和相关组件，大部分会随重写消失，M1 先删一遍等于做两遍；
- 每一处都要判断：对象经展开传入、按另一个类型写入、交给第三方库回调的成员，脚本也会列出（例如 `ui` 表格的列对象 `thRender`、`tdRender`，由 `useProjectColumns` 写入）；删掉一个 prop 还要收掉读它的分支。

- **怎样列出**：在仓库根目录先 `pnpm --filter web exec react-router typegen`，再 `node deadsym.mjs --tsv > dead.tsv`、`node domains.mjs dead.tsv --rows M2`。三个脚本的全文在 [M1 收尾计划](../../M1-frontend-trim/plans/closeout.md)的附录 A（`lib/program.mjs` 放在 `deadsym.mjs` 旁边的 `lib/` 下）。每行是 `member` 或 `prop`、文件和行号、名称；领域按文件路径划分（附录 A 中 `domains.mjs` 的 `DOMAINS`，取第一个匹配）。
- **怎样处理**：本 M 改到一个文件时，删掉其中列出的死成员和死 prop，连同只为它们存在的代码；确认是误报的不删。
- **关闭条件**：本 M 合并时，`domains.mjs … --rows M2` 列出的每一行都已消失，或者写进本 M 的 review（文件、名称、谁在读或传它）。

## oxlint 的清理（M1 设计 7.3）

M1 结束时 oxlint 警告共 694 个（web 565、editor 65、ui 25、utils 18、propel 16、hooks 3、constants 1、i18n 1），按包、按规则的表在 [收尾 spec](../../M1-frontend-trim/specs/closeout.md) 3.3 和 [收尾 review](../../M1-frontend-trim/reviews/closeout-review.md)。
- **谁改谁清**：本 M 改到的文件，在本 M 结束时没有 oxlint 警告；
- **按规则清一类**：另外按规则集中清掉至少一类，优先能机械修复的（`eslint(no-shadow)`、`eslint-plugin-promise(always-return)`、`eslint(no-unneeded-ternary)`）；
- 上限随之调低（`tools/lint-cap.mjs` 要求警告数等于上限）。
- **关闭条件**：本 M 的 review 写明改到的文件的警告数（为 0）、清掉的规则和各包上限的变化。

## 个人设置的主题下拉框

M1 收尾的浏览器核对看到：界面语言为 zh-CN 时，个人设置里主题的下拉框画在页面的左上角，不在它的按钮旁（[收尾 spec](../../M1-frontend-trim/specs/closeout.md) 2.10 的 I1，基线上相同）。主题选择器是 ui 的 `CustomSelect`（`web/packages/ui/src/dropdowns/custom-select.tsx`，react-popper 定位，`createPortal` 到 `document.body`），整个 M1 里只改过包名，是 Plane 原有的行为。个人设置归本 M，改写这一页时一起查定位。
- **关闭条件**：本 M 的浏览器核对写明个人设置的主题下拉框在它的按钮旁展开（zh-CN 和 en 各一次）。

来源：[M1 收尾 spec](../../M1-frontend-trim/specs/closeout.md)第 4 节、第 8 节，2.10 的 I1。

## 处理结果（M2/P5）

- **死成员和死 prop**（完成）：`domains.mjs … --rows M2` 在 M1 收尾时 66 行，P5 开始时 64 行，结束时 24 行。删掉的有 PAT 的旧 store（`store/workspace/api-token.store.ts`，随 PAT store 重写）、`@nerve/services` 的 axios 基类、`settings.store` 的 `isLoading`、`error`、`isScrolled`、`toggleIsScrolled`，`issue/profile` 两个 store 的 `userId`、`issueFilterService`、`currentView`、`quickAddIssue`，`IUserPermissionStore` 的 `fetchWorkspaceLevelProjectEntities`，`IBaseIssueFilterStore` 和实现它的八个过滤 store 的 `appliedFilters`（M2 的一行只能连同接口一起删），个人设置两张表各写一遍的成员类型（合成一个类型），以及 5 个没人传的 prop（`NotAuthorizedView` 的 `actionButton`、`ProfileSidebar` 和 `ProfileSettingsHeading` 的 `className`、两个认证包装的 `isLoading`）。剩下的 24 行都不是死代码：23 行是测试替身和测试里的字面量（`fake-browser.ts`、`fake-nerve.ts`、`fake-time.ts` 的成员只由测试读取；`token-manager.test.ts` 的记录字面量由 `JSON.stringify` 读取），`OnboardingRoot` 的 `invitations` 由 M3 加回取数时传入（M2 设计 3.1、13.2）。逐行的理由在 [M2/P5 spec](../specs/P5-web-account.md) 附录 A。
- **oxlint**（完成）：P5 改到的 157 个 web 文件和 P4 改到的文件都没有警告；`eslint(no-unneeded-ternary)` 全仓清零（54 条）。各包上限：web 551 → 452，ui 25 → 19，utils 18 → 12，i18n 1 → 0（逐个任务的数值见 [M2/P5 spec](../specs/P5-web-account.md) 2.10）。
- **主题下拉框**（完成）：根因在 Headless UI 2.2 的 `Combobox.Options`，与语言无关：它克隆自己唯一的子元素时换上自己的 ref，放在子元素上的 react-popper 的 ref 从来没有被设置，列表停在 `position: absolute; left: 0; top: 0`，即页面左上角。`CustomSelect`、`CustomSearchSelect` 和 `DateDropdown` 改为把 popper 的 ref、样式和属性放在列表元素本身（`Combobox.Options`，`CustomSelect` 是 `Listbox.Options`；Headless UI 转发这个 ref）；`CustomSearchSelect` 另外设 `modal={false}`，否则搜索框所在的列表里的选项被设为 inert，点不了。`CustomSelect` 改用 `Listbox`，`CustomSearchSelect` 仍用 `Combobox`（列表打开时搜索框取得焦点），两者的打开状态只有 Headless UI 的一份；`DateDropdown` 保持两份，经 `onClose` 同步，键盘打不开日历，交给 M4（M2 设计 13.2）。A9 的页面版本核对主题列表在按钮旁展开（英文、中文各一次），中文下 Escape 之后一次点击就再打开，每周第一天的列表（星期名是中文）从键盘打开、选择、关闭；A8 核对时区列表；A11 用鼠标核对自定义有效期的日历。浏览器核对 C2 在中英文下核对主题、时区、每周第一天三个列表（中文下每周第一天的标题和星期名也是中文），C3 核对日历。

来源：[M2/P5 spec](../specs/P5-web-account.md) 第 7 节。

## 处理结果（M2/收尾）

三节都在 M2/P5 完成，收尾在 `d97c513` 上核对：`make lint-web` 通过，各包的 oxlint 警告数等于上限（web 452、ui 19、utils 12、i18n 0）；`eslint(no-unneeded-ternary)` 全仓 0 条警告（[M2/P5 评审](../reviews/P5-web-account-review.md)第 2 节第 4 行；Plane 原有的两处禁用注释在 `issue-layouts/utils.tsx`、`issue-layouts/quick-add/root.tsx`）；剩下的 24 行死成员和死 prop 都不是死代码，逐行的理由在 [M2/P5 spec](../specs/P5-web-account.md) 附录 A；主题下拉框在按钮旁展开，由 A9 的页面版本在英文、中文下各核对一次，随 `make e2e` 通过。状态改为 `closed`。

来源：[M2 收尾 spec](../specs/closeout.md) 2.1。
