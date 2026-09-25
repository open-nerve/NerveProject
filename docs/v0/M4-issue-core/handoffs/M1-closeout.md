---
status: open
from: M1/closeout
to: M4
created: 2026-09-25
---

# M1 收尾留下的清理：死成员和死 prop、oxlint、一处类型断言、弹窗描述的 ID 迁移、标注块的 Markdown 序列化

M1 收尾把 knip、tsc 看得见的死代码和包导出都删完了（[收尾 spec](../../M1-frontend-trim/specs/closeout.md)）。下面几项留给后续各 M，做法都是"谁改谁清"：本 M 重写或修改到的代码，在本 M 结束时不再带着它们。

## 死成员和死 prop

类型检查器能找出两类 knip 和 tsc 都不报的死代码：没有代码读取的对象成员（store、service 的方法和属性，接口和类型的字段），和没有调用方传入的组件 prop。M1 收尾结束时，web 里一共 1346 个（成员 894、prop 452），属于本 M 领域（工作项、草稿、评论、表情、关联、筛选、布局、动态、搜索、描述，以及 `packages/editor`、`packages/shared-state`）的有 **558 个**（成员 435、prop 123）。

M1 没有删它们（[收尾 spec](../../M1-frontend-trim/specs/closeout.md) 第 4 节）：
- 本 M 按总体设计 7.2 重写这一领域的 types、services、stores 和相关组件，大部分会随重写消失，M1 先删一遍等于做两遍；
- 每一处都要判断：对象经展开传入、按另一个类型写入、交给第三方库回调的成员，脚本也会列出（例如 `ui` 表格的列对象 `thRender`、`tdRender`，由 `useProjectColumns` 写入）；删掉一个 prop 还要收掉读它的分支。

- **怎样列出**：在仓库根目录先 `pnpm --filter web exec react-router typegen`，再 `node deadsym.mjs --tsv > dead.tsv`、`node domains.mjs dead.tsv --rows M4`。三个脚本的全文在 [M1 收尾计划](../../M1-frontend-trim/plans/closeout.md)的附录 A（`lib/program.mjs` 放在 `deadsym.mjs` 旁边的 `lib/` 下）。每行是 `member` 或 `prop`、文件和行号、名称；领域按文件路径划分（附录 A 中 `domains.mjs` 的 `DOMAINS`，取第一个匹配）。
- **怎样处理**：本 M 改到一个文件时，删掉其中列出的死成员和死 prop，连同只为它们存在的代码；确认是误报的不删。
- **关闭条件**：本 M 合并时，`domains.mjs … --rows M4` 列出的每一行都已消失，或者写进本 M 的 review（文件、名称、谁在读或传它）。

## `viewId as TProfileViews`

`web/apps/web/core/hooks/use-issues-actions.tsx` 的个人主页一支把 `viewId` 断言为 `TProfileViews`：共享的 `IssueActions` 接口把 `viewId` 定为 `string`（M1/P4 修复轮留下的基线写法，P4 评审第 7 节）。根治是在路由处把个人主页的视图段按 profile views 收窄一次，再把 `TProfileViews` 往下传，这要改各种工作项 store 共用的 hook 接口，本 M 重写工作项 store 时一并做。

- **关闭条件**：`git grep -n "as TProfileViews" -- web` 没有输出。

## 工作项弹窗把描述的 ID 迁移当作改动

编辑器给没有块 ID 的旧描述补上 ID 时，也会触发 `onChange`，第三个参数带 `isMigrationUpdate: true`（`packages/editor/src/hooks/use-editor.ts`，事务的 `uniqueIdOnlyChange` 标记）。详情页的描述（`web/apps/web/core/components/editor/rich-text/description-input/root.tsx`）读取它，这次保存带 `skip_activity`，不记动态；工作项弹窗的描述（`web/apps/web/core/components/issues/issue-modal/components/description-editor.tsx`）的 `onChange` 不读，照常写回表单的 `description_html` 并调用 `handleFormChange`。所以在弹窗里打开一条描述是旧格式的工作项，什么都不改，表单也被标为已改。M1 收尾的评审发现；工作项弹窗归本 M。

- **关闭条件**：`git grep -n isMigrationUpdate -- web/apps/web/core/components/issues/issue-modal` 有输出，并且本 M 的测试或浏览器核对写明：在弹窗里打开描述没有块 ID 的工作项、不做改动，表单没有被标为已改。

## 标注块的 Markdown 序列化不转义

`web/packages/editor/src/extensions/callout/extension-config.ts` 给 tiptap-markdown 的节点序列化（`addStorage` 的 `markdown.serialize`）把属性原样拼进 HTML：`` `> <img src="${attrs["data-emoji-url"]}" alt="${attrs["data-emoji-unicode"]}" width="30px" />` ``，图标一支同样把 `data-icon-name` 拼进 `<icon>…</icon>`。这些属性来自描述的 HTML 和本地存储：`data-emoji-url` 里的 `"` 会提前结束属性，`data-icon-name` 里的 `<` 会写出新的标签。

它现在不会运行：tiptap-markdown 的 `getMarkdown()` 在应用里没有调用方（编辑器 ref 的 `getMarkDown` 用的是 `@nerve/utils` 的 `convertHTMLToMarkdown`），`Markdown.configure` 的 `transformCopiedText` 是 `false`（复制时不序列化；`transformPastedText` 是粘贴时的解析，不用节点序列化）。所以各扩展的节点序列化（标注块、`custom-color`、`custom-image`、`emoji`、`mentions`）看起来都是死代码。编辑器归本 M，由本 M 决定删掉它们，还是保留并转义。

- **关闭条件**：`git grep -n "markdown: {" -- web/packages/editor/src` 没有输出（删除）；或者保留，并且本 M 的测试证明标注块写出的属性经过转义（`data-emoji-url` 里的 `"` 写出后仍在属性值里）。

## oxlint 的清理（M1 设计 7.3）

M1 结束时 oxlint 警告共 696 个（web 566、editor 65、ui 25、utils 19、propel 16、hooks 3、constants 1、i18n 1），按包、按规则的表在 [收尾 spec](../../M1-frontend-trim/specs/closeout.md) 3.3 和 [收尾 review](../../M1-frontend-trim/reviews/closeout-review.md)。
- **谁改谁清**：本 M 改到的文件，在本 M 结束时没有 oxlint 警告；
- **按规则清一类**：另外按规则集中清掉至少一类，优先能机械修复的（`eslint(no-shadow)`、`eslint-plugin-promise(always-return)`、`eslint(no-unneeded-ternary)`）；
- 上限随之调低（`tools/lint-cap.mjs` 要求警告数等于上限）。
- **关闭条件**：本 M 的 review 写明改到的文件的警告数（为 0）、清掉的规则和各包上限的变化。

来源：[M1 收尾 spec](../../M1-frontend-trim/specs/closeout.md)第 4 节、第 8 节；弹窗描述和标注块两节来自收尾各 Task 的评审。
