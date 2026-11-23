# M7 / P5b：网页的收集箱——新建弹窗、文件与嵌入

> 状态：草稿，预检 [re-95](../../../reviews/re-95-M7P5b预检.md)（修正后通过）之后照主控的裁定修正。基线 main `ab86965f`（P5a 的 noff，`ab86965fc615fce57bfa4197515e58205e939222`）。设计见 [M7 设计](../M7-design.md) §2（H8、H10、H11、H20）、§3.5、§3.6（`:108-113`）、§3.11、§7（P5 一条 `:245`）、§11、§12（P5b 一行 `:307`）、§14、§15 A、B；网页的长期规则见[总体设计 §7.2、§7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)（请求体轮到时才算 `:638`）。上一段见 [P5a spec](P5a-web-intake.md)（§2 的数据层、写的接缝、详情、分诊与焦点的落点 R35 照旧约束本段）与它的[阶段记录](../reviews/P5a-web-intake-review.md)。接口合同见 [P2a spec](P2a-intake-submit-triage.md)、[P2b spec](P2b-intake-list-edit-delete.md)、[P2c spec](P2c-intake-assets.md)（资产的写，§2.2）。现状调研在台账 `.superpowers/sdd/P5b-web-intake-files-embed/research/scope.md` 与 `research/questions.md`（B-n 的选项与建议）；主控裁定在台账 `rulings.md`（下文 B-n 指它）；P5a 的裁定副本在 `research/p5a-rulings.md`（Q-n）。下文 `C/` = `web/apps/web/core/`，`A/` = `web/apps/web/app/(all)/[workspaceSlug]/(projects)/`，`IN/` = `C/components/inbox/`，`WN/` = `C/components/workspace-notifications/`，`K` = `tools/keywords.json`，`IY` = `api/modules/issue.yaml`，`HO/` = `docs/v0/M7-collaboration/handoffs/`，`D` = `docs/v0/M7-collaboration/M7-design.md`，`S5a` = P5a spec，`R5a` = P5a 阶段记录，`E/` = `e2e/`，`pw/` = 只读的 `plane/apps/web/`，`pa/` = 只读的 `plane/apps/api/plane/`。行号都按基线读出；第三方库的行为按主目录 `node_modules` 里装着的 `@headlessui/react@2.2.10`、`swr@2.4.2` 的源码（预检 §1）。本段不改 `server/`、`api/`、`deploy/` 与 CI。

## 1. 目标与边界

P5 在实施前划为两段（S5a §1，Q1）。P5a 已合入（noff `ab86965f`，main CI `45197451256`）；P5b 是它在 S5a §5“后续”里列下的其余，加上 R5a §7 与 P5a 台账里留给本段的几件（台账 `progress.md` 的范围一行）：

- **新建弹窗的其余**：关闭政策（M6-M7-01：编辑器建成之前能关空表单，提交仍要求就绪；有改动时先确认放弃；在途时 Escape 与点背景不动焦点，R28）；迟到的成功不跳出已离开的页面（R29）；弹窗以 `ModalTitle` 为名；名字的提示是 `FieldError`、被拒取得焦点；去掉文档级的 Escape；四个文件离开 `getTabIndex`；弹窗里的英文字面量改走已有的键；“Create more” 是开关本身；死代码删除（P5 剩下的 6 行死成员归零）。
- **收集箱详情的文件**：描述图片与附件（M5-M7-01），前端的写权限照收集箱的规则（R9，动 M5 的共用件）；`[local]` 与 `[s3]` 三引擎的页面证据（Q21，范围见 R12）。
- **收集箱详情的链接**（Q14），照合同的 `issue.update`（R10）。
- **通用详情与浮层**在条目未被接受时隐去状态（Q12）；**浏览页**把未被接受的条目以 `replace` 送到收集箱页（Q12、R14）；收集箱页的“打开工作项”只给已接受的（R15）。
- **通知预览嵌入收集箱的内容**（Q13）：读完工作项之后判定、判定锁定（R16、R27）；P4 的两个死 prop 以“选中的端口”的形式随读者回来（R32、R18）；通知页自己的一份焦点落点（R26），浮层的关闭也走它。
- **负责人多选的连点**（Q34）：按代码必然重现（预检 I5），修在写的层（T8c，R20 (b)），故事作证。
- **页面核对**：两个负责人下拉框（`HO/M3-closeout.md:51` 的收集箱一半，R21）。
- **P5a 的 A9 回填**（S5a `:387`）与设计 §14 的 P5a 一行（T1）。
- H8 (page)；H11 (page) 的资产、链接与连点；嵌入的故事；H2–H4 (page) 的收集箱一步；浏览页的重定向一步。

**P5b 不做：**
- `getTabIndex`、`ETabIndices.PROJECT_VIEW`、两份 `tab-indices.ts` 与桶文件导出的最终删除：去掉收集箱的四个读者之后只剩视图表单 `C/components/views/form.tsx:75`，删除随 P6a（Q36、S5a R23）；`projectMemberIds` 同。
- 视图的弹窗、右键菜单（`HO/M3-closeout.md:51` 的另一半，P6a）；共用组件的焦点轮廓是否被 `overflow-hidden` 裁掉（交 M8 的全应用核对，R21）。
- 服务端、合同与 CI：G1–G3 照 S5a R26 不改；计数的实时性照旧进 M7 收官报告（S5a R8）；CI 余量只报告（R2）。
- 通用详情与浮层上别的会被拒的控件（开始日、迭代与模块、父工作项、子工作项与关联、归档与删除）不另加分支（R13）。
- 资产的写被拒（`issue.intake_closed`、`issue.intake_disabled`）之后不重读条目（§2.2 末，交 M8 的候选）。

**P5b 之前 main 的样子**（R5a §4 `:80-87`）：收集箱页经 `/api/v0` 可用；新建弹窗经新接口提交，关闭政策仍是 Plane 的（文档级 Escape、编辑器未就绪时拒绝关闭，空表单在编辑器建成之前关不掉）；收集箱详情没有图片上传、附件与链接；收集箱条目的通知以普通浮层打开，属性里 “State” 一栏是占位；浏览页把收集箱条目显示成普通详情，七个字段之外的修改在本地以 `issue.in_triage` 拒绝；负责人多选在第一次修改的回答到之前再点，第二次修改丢掉第一位。

**P5b 之后 main 的样子**：新建弹窗的 Escape、点背景与 “Discard” 走同一条关闭规则（在途不关、焦点留在弹窗里；编辑器没建成可关；有改动先确认）；访客提交者能在自己的待处理、暂缓条目上加图片与附件，不是提交者的成员、已处理的条目、收集箱关着时没有上传的控件；成员能在任何状态的条目上加链接；浏览页打开一个未被接受的条目时换成收集箱页并选中它；通知页里未被接受的条目嵌着收集箱的详情与页头（管理员在那里分诊），已接受的仍是浮层，关掉任何一种之后焦点回到那张卡片；负责人多选的每次修改在轮到时并到所持的集合上；`getTabIndex` 只剩视图一个读者。收集箱一侧不再有留给 P5 的项。

R14 的两个后果（预检 O3，与 B-14 一致，§7 第 3 条登记）：收集箱关着时，未被接受的条目经地址到不了任何详情（落到 P5a R19 的停用页），只剩通知的嵌入能看见它；到期之前的暂缓条目被送到“未处理”页签，而这个页签默认只列待处理（`C/store/issue/intake/filters.ts:50`），条目选中了、列表里没有它那一行。

规模：14 个 Task（B-1 的 (b)；B-1 的 T3 并进 T2，H11 一条拆为 T8a、T8b，预检 I5 加 T8c；plan 写出对照）。

## 2. 交付合同

### 2.1 新建弹窗（R3–R8、R24、R28、R29）

文件 `IN/modals/create-modal/`：`modal.tsx`（52 行）、`create-root.tsx`（240）、`issue-title.tsx`（56）、`issue-description.tsx`（62）、`issue-properties.tsx`（90）；打开它的只有 `IN/page-header.tsx:72-80`（`canSubmit` `:46-48`）。

**结构**：
- 关闭规则 `requestClose` 写在持有表单状态的 `create-root.tsx` 里；`modal.tsx` 的 `ModalCore` 以 `handleClose` 调它（Escape 与点背景都经 `handleClose`，`web/packages/ui/src/modals/modal-core.tsx:38-42`；最内层的对话框认领 Escape，`:96-100`）。`create-root.tsx` 以 ref 交出 `requestClose`（`useImperativeHandle`，先例 `DescriptionInputHandle`），“Discard” 按钮也调它。不改 M3 的 `ModalCore`（R4）。
- **在途不给 `handleClose`**（R28，预检 m2）：`create-root.tsx` 把在途告诉 `modal.tsx`（一个 `onSubmittingChange` prop，随 `formSubmitting` 变时调），`modal.tsx` 在途时给 `ModalCore` 的 `handleClose` 是 `undefined`（同 `AlertModalCore`，`alert-modal.tsx:76`）。实际的结果：
  - 在途按 Escape：`ModalCore` 阻止它的默认（`modal-core.tsx:96-100`），Headless 的窗口监听见到已阻止的事件就什么都不做（2.2.10 的 `hooks/use-escape.js`：既不 `blur` 也不关）；焦点不动——在字段里的留在字段里，提交按钮停用时已交给对话框的（`keep-focus.ts`）照旧等按钮恢复可用时还给它。
  - 在途点背景：Headless 在 `pointerup` 判为点外面、调 `onClose`（`hooks/use-outside-click.js`），`handleClose` 是 `undefined`，什么都不关；按下时浏览器把焦点给背景最近的可聚焦祖先，即 Headless 的对话框元素（`tabIndex=-1`），焦点仍在弹窗里，`keepFocusIn` 兜底，绝不落到 `body`。
  - 被拒之后（nerve 的拒绝）：提交按钮恢复可用，焦点回到它（`keepFocusIn` 的等待没被 Escape 换掉）；名字被拒时 `useRefusedFocus` 先给名字。
  - `requestClose` 仍看在途（Discard 在途已停用；两层同查，变异两层一起改、留一层的对照）。
  - H8 (page) 在途的一步三引擎断言：Escape 之后、点背景之后、被拒之后焦点都在弹窗里（每个引擎落在哪个控件记台账）。
- 删 `modal.tsx:11`、`:29-34` 与 `create-root.tsx:26`、`:86-99` 两个文档级的 `useKeypress("Escape")`（`HO/M3-closeout.md:63`）。输入法组字时的 Escape 由 `ModalCore` 阻止（`modal-core.tsx:99` 的 `isComposing`），不关、不弹确认框；一格钉住（R29）。
- **每次打开从初值起**（预检 m4）：`ModalCore` 关上、离场的过渡（200 毫秒，`modal-core.tsx:104`、`:113`）结束之后卸下内容，再开是新的表单；一格（填字、确认放弃、等关上、再开是空的）与一个变异钉住。过渡当中再打开是同一份表单（Headless 把过渡倒回去），写明为已知的行为：确认放弃之后 200 毫秒内再按 “Add work item” 才碰得到，不另加 key。

**关闭规则**（R4，同 M4 `C/components/issues/issue-modal/draft-issue-layout.tsx:68-90` 去掉草稿）：

| 情形 | 结果 |
|---|---|
| 提交在途 | 不关：`ModalCore` 没有 `handleClose`（R28）；“Discard” 与 “Create more” 停用；`requestClose` 也不关 |
| 描述编辑器的引用为 `null`（编辑器的代码块还在路上、没建成） | 按下面的“有改动”两行 |
| 引用存在且 `isEditorReadyToDiscard()` 为假 | 提示 `t("editor_is_not_ready_to_discard_changes")`，不关（这个表单不收图片，编辑器建成之后按代码恒为真，`web/packages/editor/src/helpers/editor-ref.ts:144`；一条规则照 M4 留着，单测以替身的引用造出它） |
| 有改动 | 弹确认放弃的 `AlertModalCore`（下面），不关 |
| 没有改动 | 直接关 |

- **有改动**：名字去掉两端空白之后非空，或描述 `!isEmptyHtmlString(description_html)`（`web/packages/utils/src/string.ts:93`，同 M4 的 `draft-issue-layout.tsx:58-63`：编辑器建成时会把描述写成空段落，不能按字面比），或优先级不是 `none`，或负责人、标签非空，或截止日非空。与 M4 的差别：M4 把只有空格的名字算作改动（`draft-issue-layout.tsx:53` 只删空串），本段不算（名字的规则同合同，去空白之后判）。
- **确认放弃**：`AlertModalCore`（以 `ModalTitle` 为名，`web/packages/ui/src/modals/alert-modal.tsx:53`），`variant="danger"`；标题与正文是新键 `inbox_issue.modals.discard.title`、`inbox_issue.modals.discard.content`（en 与 zh-CN；找过没有可用的已有键：`issue.draft.save_prompt.*` 说的是存草稿）；主按钮 `t("discard")`、次按钮 `t("common.cancel")` 显式传入（`AlertModalCore` 的默认文字是英文，`:62-66`，P5a 整审 m4 的教训）。确认 → 两个弹窗都关、焦点回到 “Add work item”；取消或 Escape → 只关确认框，表单原样，焦点回到打开确认框的控件（Escape 由最内层认领）。
- **提交成功之后**直接关（不经“有改动”）：跳到新条目、关弹窗（`create-root.tsx:145-147`；地址在 T5 改读 `intakeAddress`，字符串不变，§2.4）；“Create more” 开着时清空表单、不关。成功提示改走 `t("issue_created_successfully")` 与 `t("toast.success")`。

**跟进只在表单还挂着时**（R29，预检 m1）：提交的跟进今天只核会话（`create-root.tsx:143-159` 经 `followInSession`，`C/lib/in-session.ts:40-53`）。提交在途时弹窗关不掉，但后退、前进不受弹窗约束（v0 `:642`），项目的布局随之重挂、弹窗卸下，迟到的成功照样 `navigate` 回原项目的收集箱并提示成功。改为：成功与失败的跟进（跳转、关、清空、提示）只在表单的内容还挂着时做——弹窗的内容在按 id 键着的项目布局之下，卸下即离开了这一页或这个项目；会话照旧由 `followInSession` 管。写法是 `create-root.tsx` 自己的一个挂载范围（一个 ref，挂上为真、卸下为假），不用 M4 的 `useDetailScope`（`issue-modal/base.tsx:93-98`）：它读 `rootIssueStore`，而 `create-root.request.test.tsx:19-23` 的 store 替身没有 `StoreContext`。条目在 nerve 已建成；收集箱的列表与计数照旧在 store 一层重取（P5a R30）。一格（扣住提交、卸下、放开：没有跳转、没有提示）与一个变异。

**提交**：仍要求编辑器就绪：引用为 `null` 或 `isEditorReadyToDiscard()` 为假时提示 `editor_is_not_ready_to_discard_changes`、不发（`create-root.tsx:122-129` 的英文提示改键）。在途时 “Discard” 与 “Create more” 停用，提交按钮 `loading`（今天 Discard 在途不停用，research §1(a)）。

**名字**（R5）：
- `issue-title.tsx` 照 M4 `C/components/issues/issue-modal/components/title-input.tsx:56-81`：propel 的 `Field` 包着输入框，提示是 `FieldError`，控件恰在提示显示时 `aria-invalid`，以提示为描述；输入框有名字（`aria-label` 经 `t("title")`）。**删 `issue-title.tsx:48` 的 `required`**（预检 m3）：留着它，空名字由浏览器的原生校验拦下，`submit` 事件不发，`FieldError`、`aria-invalid` 与 `useRefusedFocus` 都不走（M4 的输入框没有它，`title-input.tsx:58-74`；`HO/M3-closeout.md:62`）。表单状态仍是 `useState`（不迁 react-hook-form）。
- 规则**同合同**（`IY:2365-2367`，服务端 `server/internal/modules/issue/domain/issue.go:184` 先 `TrimSpace`）：`Array.from(name.trim()).length` 为 0 → `t("title_is_required")`；大于 255 → `t("errors.field.too_long")`。与 M4 的差别：M4 的超长按没去空白的字符数（`title-input.tsx:43`），本段按去了空白的。超长的提示边打边显示；空白的提示只在一次提交被拒之后显示，到名字不再空白为止。单测与 H8 的“空白”分两格：空的与只有空格的。
- 提交不停用（删 `create-root.tsx:163`、`:228` 的按原始 `length > 255` 停用）；提交时本地判两条，被拒则显示提示、不发，焦点经 `useRefusedFocus`（`C/lib/refused-focus.ts:29`，它交回的 ref 挂在 `<form>` 上，取代 `:61` 的 `formRef`）给名字；每次被拒的提交是一次新的 attempt。
- nerve 的拒绝经 `useFormRefusal(["name"], {})`（`C/lib/form-refusal.ts:58-75`）：`validation_failed` 点名 `name` 时提示放到名字下并取得焦点；别的（`issue.intake_disabled`、`project.archived`、`server_busy` 等，`IY:888`）经 `useRefusalToast`，表单留着。

**名字、开关与英文**：
- 标题经 `ModalTitle`（`create-root.tsx:172` 的 `h3`，`HO/M3-closeout.md:44`），弹窗以它为名（“Create intake work item”，`inbox_issue.modal.title`）。`ModalTitle` 是 Headless 的 `DialogTitle`，在 `Dialog` 之外渲染即抛错（2.2.10 的 `dialog.js`，预检 I1）：两份既有测试把表单直接渲染在 `ModalCore` 之外，照 M4 的先例（`issue-modal/base.clone.test.tsx:94-95`）部分替身 `@nerve/ui`、只把 `ModalTitle` 换成普通的 `h3`（plan 的允许清单）；要真实 `ModalCore` 的格（弹窗的名字、点背景、Escape、在途）都在新文件里，经 `InboxIssueCreateModalRoot` 渲染、从 `document` 查（`ModalCore` 渲染进 portal）。
- “Create more” 是 `Switch` 本身（`aria-label` 经 `t("create_more")`、`onCheckedChange`、在途停用），照 M4 `C/components/issues/issue-modal/form-footer.tsx:66-77`；删 `create-root.tsx:194-202` 的 `div role="button"`（按钮里套开关）与只为它的 `create-root.tsx:2-3` 两条 `oxlint-disable`（R24）。
- 英文字面量改走已有的键（R6）：`create-root.tsx:90-95`、`:124-128`、`:212-216` → `editor_is_not_ready_to_discard_changes`（`common.json:304`）与 `toast.error`；`:152-156` → `issue_created_successfully`（`:293`）与 `toast.success`；`issue-properties.tsx:60` → `issue.add.assignee`（“Add assignees”，`work-item.json:23`）；`:84` → `issue.add.due_date`（“Add due date”，`:25`）。可见文字随之变，H8 (page) 按新文字定位。

**`getTabIndex`**：四个文件去掉 `getTabIndex`、`ETabIndices` 的导入（`create-root.tsx:14`、`issue-title.tsx:12`、`issue-description.tsx:12`、`issue-properties.tsx:10`）与只为它读的 `usePlatformOS`（`create-root.tsx:21`、`:27`、`:68`、`:84` 与三处 `tabIndex`；`issue-title.tsx:16`、`:18`、`:32`、`:34`、`:47`；`issue-description.tsx:16`、`:20`、`:41`、`:43`、`:59`；`issue-properties.tsx:11`、`:18`、`:32-33` 与四处 `tabIndex`），Tab 走 DOM 的自然次序：名字 → 描述 → 优先级 → 负责人 → 标签 → 截止日 → “Create more” → “Discard” → 提交（今天的正 `tabIndex` 次序本就是 DOM 的次序，去掉之后只少了开关停两次；编辑器是否多出停点由 H8 的 `tabStops` 核）。`web/packages/constants/src/tab-indices.ts` 删收集箱一组：`INTAKE_ISSUE_CREATE_FORM_TAB_INDICES`（`:7-21`）、枚举成员 `INTAKE_ISSUE_FORM`（`:26`）与映射一行（`:31`）；函数、`PROJECT_VIEW` 与桶文件留给 P6a（`git grep getTabIndex -- web` 之后只剩 `views/form.tsx:75`）。

**死代码**（R3）：删 `create-root.tsx:62` 的 `modalContainerRef` 与 `:101-116` 进门即返回的 `ResizeObserver` effect；删 `modal.tsx:25-27`、`:40`、`:47-48` 与 `create-root.tsx:40-41` 的 `isDuplicateModalOpen`/`handleDuplicateIssueModal`（宽度固定 `EModalWidth.XXXXL`）。两份测试里的 `NoResize` 替身随之删去（plan 的允许清单），P5 剩下的 6 行死成员（R5a `:116`）归零。

**测试**（预检 m4）：新的格约 35 个，按关注点拆成三个新文件加一份共用的夹具（先例 `IN/fake-intake-page.tsx`）：`IN/modals/create-modal/fake-create-modal.tsx`（夹具：真实的 store、替身的编辑器引用 `null`/就绪/未就绪、扣住的提交）；`create-root.close.test.tsx`（关闭规则、确认框、在途、IME、再开从初值起，经 `InboxIssueCreateModalRoot` 与真实的 `ModalCore`）；`create-root.name.test.tsx`（名字的规则、提示、被拒与焦点、nerve 的拒绝）；`create-root.form.test.tsx`（提交与成功、迟到的跟进、开关、Tab、英文字面量）。每份不过 400 行。`create-root.request.test.tsx`（191）、`create-root.files.test.tsx`（143）只按 plan 的允许清单改。点背景在 jsdom 里以背景上的 `pointerdown`、`pointerup` 派发（Headless 2.2.10 的点外面只听这两个，`hooks/use-outside-click.js`）；触发不了时这一格留给 H8 (page)，单测只钉 `handleClose` 的接线，记台账。

### 2.2 写权限的共用谓词与描述图片（R9，高风险；R11 的一半）

**谓词**：`C/store/issue/intake/status.ts`（63 行）加 `intakeEditableBy(issue, project, role, me)`，即 P5a 的 `editableBy`（`IN/content/detail.tsx:44-57`）搬过来：项目在、未归档、`intake_view` 开着；条目存的状态是待处理或暂缓（只看存的状态，同服务端的 `CheckOpen`，P2c spec §2.2 `:44`；到期的暂缓仍可写）；调用者有项目角色，且是管理员（`EUserPermissions.ADMIN`，含是项目成员的工作区管理员，权限 store 给 ADMIN，`permissions.store.ts:68-75`，同服务端 `access/domain/decide.go:85`）或提交者（`created_by_id === me`，访客也算）。“有项目角色”是合同的 `intake.update` 三种角色（`server/internal/modules/access/domain/rules.go:167`）；`editableBy` 今天不查它，读不到详情的人看不到这页，所以页面上没有变化，单测加一格。`detail.tsx` 删去自己的 `editableBy`，读这个谓词；“只改名字与描述”的访客规则照旧（`:171` 的 `isEditable={editable && role !== GUEST}`）。

**`ContentView.canWrite`**（`C/store/asset/content-view.ts:62-76`，M5 的共用件，L!）：
- 只改 `issue` 宿主一支：所持详情 `inIntake`（`status.ts:34`，存在且未被接受）时，返回 `intakeEditableBy(...)` 的结果；其余（`intake` 为 `null`、已接受、只持有行）照旧按角色 15/20（`:72-75`）。
- `new_comment`、`comment`、`draft` 宿主不变（在 `issue` 一支之前返回；评论的两个表单 `comments/comment-create.tsx:108-183`、`card/edit-form.tsx:95-168` 读的是评论宿主一支）；`canRemoveAttachment`（`:77-92`）先看 `canWrite`，随之认收集箱的规则（拒绝、重复的条目上管理员也不能删附件，合同 409 `issue.intake_closed`，P2c spec §1 第 3 条）。
- 不加新的宿主种类（M5-M7-01 的 P2c 处理结果，`HO/M5-closeout.md:20`）。
- 后果：上传与删除的每一步核对它（`C/store/asset/root.store.ts:55`、`:97-100`；`C/hooks/editor/editor-assets.ts:36`、`:117`；`C/store/asset/upload-policy.ts:63`；`C/hooks/use-content-assets.ts:37`；`C/components/assets/attachments/root.tsx:51`、`:103`；删除附件的命令经 `canRemoveAttachment`，`C/store/asset/content-deletion.ts:45`，预检 m6），所以访客提交者从此能传，不是提交者的成员、已处理的条目、收集箱关着时控件不给，与合同（P2c spec §1 的四条）一致。
- 两层的关系（预检 m6）：编辑器的上传能力在 `EditorAssets` 建成时定一次（`editor-assets.ts:36`）；之后条目的状态在本页变了（本页分诊、重读），靠详情的 `disabled={!editable}` 与传输每一步的 `canWrite` 兜着。
- 单测：`C/store/asset/content-view.test.ts`（197 行）追加收集箱的格；`status.test.ts`（144）追加谓词的格；`detail.test.tsx`（181）追加“详情与内容视图同答”的格——**只对条目在收集箱里的格**（待处理、暂缓、拒绝、重复四种状态，预检 m5）；已接受与 `intake` 为 `null` 时详情给假（收集箱的详情对已接受的条目只读）、`canWrite` 按角色给真，是设计如此，两边分别断言。

**被拒之后不重读**（预检 m7，已知的行为）：页面上别人在本会话之外改了状态（另一会话拒绝、关掉收集箱），所持的详情不知道，上传的控件还在，传输照常发出，由 nerve 以 `issue.intake_closed` 或 `issue.intake_disabled` 拒绝、经提示说出（R12 的页面证据）；页面不重读条目，控件留着，再传再被拒。内容修改遇到同一个码是重读的（`C/store/issue/issue-write-intake.ts:9`），资产的写不经写队列（M5 的上传链）。本段不改 M5 的上传链：写进 §5 的 M8 候选。

**描述图片**：`detail.tsx` 读 `useIssueImages(workspaceSlug, projectId, issueId)`（`C/hooks/editor/use-issue-images.ts:7-17`，宿主 `issue`，挂载时不发请求）并把 `images` 交给 `DescriptionInput`（`:115-145`，`disabled={!editable}` 照旧）。绑定经 P5a 的收集箱写路（`PATCH /api/v0/intake-issues/{id}`，`C/store/issue/issue-writes.ts:118-124` → `issue-write-intake.ts`；合同已声明 `asset.not_found`、`asset.not_uploaded`，`IY:1007`），不加新写路。描述版本的图片照旧只读（`C/components/issues/issue-detail/description-versions.tsx:22`）。

**`ConfirmedContentNotice`**（`C/components/issues/detail-host.tsx:64-73`）：画在 `IN/content/root.tsx` 的 `IssueDetailHost` 之内、页头之前（它读宿主的输入 context，画在宿主之外读不到；同 M4 详情 `C/components/issues/issue-detail/root.tsx:91` 与浮层 `peek-overview/view.tsx:171`），确认的上传或删除之后说出结果。

### 2.3 附件与链接（R10、R11）

新文件 `IN/content/widgets.tsx`（约 70–90 行），由 `detail.tsx` 在属性之后渲染，在 `IssueDetailHost` 之内（弹窗的 context 现成，`IN/content/root.tsx:163`）。它读项目与角色（R10 的规则写在它里面），收入 `K` 的 `project-address-route` 并带样例（预检 m13）。
- **附件**：`IssueAttachments`（`C/components/issues/issue-detail-widgets/attachments/root.tsx:7-33`，可折叠、默认收起：`C/store/issue/issue-details/root.store.ts:84` 的 `openWidgets` 默认只有 `sub-work-items`、`links`；展开时才建资源、读列表，`attachments/content.tsx:12-46`），`disabled={!editable}`（R9 的谓词；`canWrite` 是第二层）。
- **链接**：一个有名字的 “Add link” 按钮（`IssueLinksActionButton` 带 `customButton`，同 M4 `issue-detail-widgets/action-buttons.tsx:49-57`，名字 `t("issue.add.link")`）；`link_count > 0` 时 `LinksCollapsible`（`links/root.tsx:25-48`，M4 同条件，`issue-detail-widget-collapsibles.tsx:41`）；`IssueLinkCreateUpdateModal`（`C/components/issues/issue-detail/links/create-update-link-modal.tsx`）以 `useLinkOperations`（`issue-detail-widgets/links/helper`）接线，关闭时同 M4 `issue-detail-widget-modals.tsx:87-91`。写经 `link.store.ts`（逐工作项的队列，确认之后在轮次里重读）。折叠栏里的加号按钮没有名字（M4 的共用件，`links/quick-action-button.tsx:31-33`、`links/root.tsx:37`），收集箱详情从此也显示它；本段的“链接按钮有名字”只管新加的那一个，加号交 M8（§5，预检 O4）。
- **链接谁能写**（R10，照合同 `IY:1735-1812`）：项目未归档，调用者是管理员或成员（`issue.update`，`rules.go:143`）；**任何状态**（待处理、暂缓、拒绝、重复、已接受）、**收集箱开着与否**都能写（三个链接操作的码里没有 `issue.in_triage`、`issue.intake_closed`、`issue.intake_disabled`，`IY:1751`、`:1789`、`:1816`；P2a 有意不给链接加守卫，`specs/P2a-intake-submit-triage.md:90`、`:125`；设计 `D:109`）；访客只读（看得到列表，没有按钮与编辑、删除）。这条规则写在 `widgets.tsx` 一处。
- **不渲染** M4 的 `IssueDetailWidgets`（`issue-detail-widgets/root.tsx:21-38`）：它的子工作项与关联的按钮对收集箱条目 nerve 答 409 `issue.in_triage`（`IY:1404`、`:1486`）；也不给它加收集箱分支（R11）。
- 读：链接已在详情的读里（H9 (page) ② 的 `detailReads` 含 `/links`，`E/fixtures/intake-pages.ts:194-199`）；附件默认收起、不多读，H9 的读表只数 `/api/v0/issues/{id}` 之下的读，加图片与附件不改它（预检 §4）。
- 单测：新 `IN/content/widgets.test.tsx`。

### 2.4 通用详情与浮层、浏览页的重定向、“打开”（R13–R15）

**隐去状态**（R13）：浮层 `C/components/issues/peek-overview/properties.tsx:75-88` 与详情侧栏 `C/components/issues/issue-detail/sidebar.tsx:79-92` 的状态一栏在 `inIntake(getIssueDetailById(issueId))` 时不渲染；已接受、`intake` 为 `null`、只持有行时照旧。其余控件不动：开始日、迭代与模块、父工作项在本地以 `ForeseenRefusal("issue.in_triage")` 说 nerve 的同一句（P5a 整审 m3，`C/store/issue/issue-write-fields.ts:60-64`），子工作项、关联、归档、删除由 nerve 答 409。`peek-overview/view.tsx`（303 行）不碰。单测进新文件（`properties.test.tsx` 351 行、`sidebar.test.tsx` 346 行不加长）。

**浏览页的重定向**（R14）：在布局 `A/browse/[workItem]/layout.tsx`（93 行）的 `IssueRouteVisit`（`:50-91`）里，这次访问的读落定（`pending` 为假，`:76`；读用的是这次访问自己的回答，`:65-76`）且回答 `inIntake(data)` 时：
- 不渲染页头与 `Outlet`（`:81-88`），在 effect 里 `navigate(intakeAddress(…), { replace: true })`；详情宿主从不为收集箱条目挂上；
- 地址 `/${slug}/projects/${project_id}/intake?currentTab=<tab>&inboxIssueId=${id}`：待处理、暂缓（存的状态，也含到期的暂缓）→ `open`；拒绝、重复 → `closed`；
- 收集箱关着照样送（落到 P5a R19 的停用页，同 Plane 的 `is_intake` 不看开关）；后果见 §1 末（预检 O3）；
- 选中某一条的收集箱地址由新文件 `IN/address.ts` 的 `intakeAddress(workspaceSlug, projectId, issueId, intake)` 给出，读者三个：这里、新建弹窗成功之后的跳转（`create-root.tsx:146` 改读它，字符串不变，`create-root.request.test.tsx:160-162` 不用改；预检 m9）、嵌入的“打开”（§2.5）。
- 工作项 id 的入口（`A/projects/(detail)/[projectId]/issues/(detail)/[issueId]/page.tsx:31-44`）读到之后 `replace` 到浏览页，再由这里送走：两次 `replace`，历史里只有收集箱页。
- 单测进新文件 `C/components/browse/intake-redirect.test.tsx`（`detail-route.test.tsx` 480 行不加长）。

**“打开工作项”**（R15）：
- 收集箱页上只给已接受的条目：动作模型的 `offeredOn`（`IN/content/header/use-intake-actions.ts:111-133`）对待处理、暂缓不再给 `open`；已处理一支里只有已接受的给 `copy`、`open`，拒绝、重复只给 `copy`；已归档的项目只给 `copy`（已接受的给 `copy`、`open`）。
- 宽页头的 “Open work item” 今天无条件渲染（`header/root.tsx:104-112`），改为 `offered.includes("open")` 时才渲染；窄页头的菜单本来按 `offered`（`mobile.tsx:70-74`）。
- 复制链接不变（浏览页的地址；在别处打开时由重定向送到收集箱页）。
- `use-intake-actions.ts` 已 300 行：本条只改 `offeredOn` 的返回，不让它变长（`wc -l` 收据），纯搬移在 T6a。
- 跟随：`header.test.tsx` 的动作矩阵与 H10 (page) 的 `expectNoTriage`（`E/stories/collaboration/h10-intake-triage-page.spec.ts:52-58`）照 plan 的允许清单改。

### 2.5 通知嵌入、选中的端口与通知页的落点（R16–R18、R26、R27、R30）

**先纯搬移**（R18；`use-intake-actions.ts` 300 行，P5a 定为不再加长，R5a §9）：类型与文案表（`IntakeAction`、`INTAKE_ACTION_LABELS`、`IntakeModal`、`OpenModal`、`IntakeActions`，`:32-102`）与纯函数 `offeredOn`（`:104-133`）搬到新文件 `IN/content/header/intake-actions.ts`，`use-intake-actions.ts` 只留 hook；读者（`header/root.tsx:30`、`mobile.tsx:21`、`content/root.tsx:37`、`header.test.tsx:17`）只改导入；一个 `refactor` 提交，既有测试一行不改。“逐字”之外只许一处（预检 m8）：今天没有导出的 `OpenModal`（`:61`）、`offeredOn`（`:111`）与 hook 读到的 `IntakeModal`（`:58`）搬过去之后 hook 要导入，加 `export`；逐行比对的收据容这几处。

**选中的端口**（R18）：`InboxContentRoot`（`IN/content/root.tsx:39-52`）的 `onLeave`、`region`、`isMobileSidebar`、`setIsMobileSidebar` 四个 prop 换成一个 `selection` 端口（类型在新文件 `IN/content/selection-port.ts`），两种：

| 端口 | 成员 | 谁给 |
|---|---|---|
| `{ kind: "page", leave, neighbour, region, navigation }` | `leave`：页面的落点（`IN/landing.ts:83-98`）；`neighbour()`：动作开始时的邻行（今天 `use-intake-actions.ts:196-200` 读 `useIntakePageState()` 与列表算出）；`region`：`landing.detail`；`navigation`：页头的上一条、下一条（`IntakeNavigation`）与窄屏的列表开关（今天 `mobile.tsx:50-58`，读 `IN/root.tsx:51` 的 `isMobileSidebar`） | 收集箱页 `IN/root.tsx:134-142` |
| `{ kind: "embed", leave, region, close }` | `leave`：经通知页的落点清选中（R26）；`region`：嵌入自己的 ref；`close`：关闭按钮的动作（同 `leave`） | 通知预览 `WN/root.tsx` |

- 动作模型 `useIntakeActions` 今天的第四个参数已叫 `selection`（`{ leave, leaving, region }`，`use-intake-actions.ts:152-156`，由 `IN/content/root.tsx:126` 拼出；`leaving` 是详情区域自己的 ref，`:125`，不进端口）；它改为收端口的 `leave`、`neighbour`、`region` 加上 `leaving`：邻行只从 `selection.neighbour` 来，不再读 `useIntakePageState()`（`:167`；它在 provider 之外抛错，`IN/page-state.tsx:51-54`，通知页没有这个 provider；`useIntakePageState` 的另一个读者 `IntakeNavigation`，`navigation.tsx:28`，嵌入时不画）；页头（`header/root.tsx:72`、`mobile.tsx:47-59`）的导航从端口来，嵌入时换成关闭按钮。
- `IN/content/root.tsx:132-158` 的“读不到”“不在收集箱”（`intake === null`）两支照旧调 `leave`：嵌入里由嵌入的 `leave` 清选中。
- 先做端口、行为不变（`refactor` 提交，既有测试只为端口的形状编译跟随，断言不改），再做嵌入。
- **与 P4、Plane 两个布尔 prop 的对应**（R32）：`isNotificationEmbed` ↔ `selection.kind === "embed"`；`embedRemoveCurrentNotification` ↔ `selection.close`（Plane `pw/core/components/workspace-notifications/root.tsx:101-110`）。`InboxContentRoot` 仍从 `@/components/inbox/content` 导出，`WN/root.test.tsx:58` 的替身照旧。

**打开的地址**（R30，预检 m9）：动作模型多一个“打开的地址” `openLink`，“打开”的两个读者（`take("open")`，`use-intake-actions.ts:255`；宽页头的链接，`header/root.tsx:104-112`）都读它，复制仍读 `link`（`:69-70`、`:183-189` 的注释随之改）。页面上“打开”只给已接受的，`openLink` 即 `link`；嵌入里，未被接受的是 `intakeAddress(…)`（收集箱页、选中它），已接受的是 `link`（浏览页）。`offeredOn` 以端口的 `kind` 区分页面与嵌入：嵌入里一直给 `open`。

**判定的读**（R16、R27，预检 I4、m10、m11，O2）：`WN/root.tsx` 的 `NotificationPreview`（`:64-114`，以通知 id 为 key，`:47`）在调用者对项目有角色之后（`:71`，没有角色时照旧先 `useProjectRead`）：
- **读的 hook 在它自己的文件里**：新 `C/components/issues/peek-overview/use-peek-issue-read.ts`（收入 swr 名单），键与取数同今天的 `PeekContent`（`peek-overview/root.tsx:91-96`，键 `["peek-issue", identity, ws, project, id]`，取数 `useIssueOperations(…).fetch`）；`PeekContent` 与预览都调它，按文件路径导入（不经桶文件 `peek-overview/index.ts:7`）；这是 M4 浮层唯一的改动。这样通知的测试照旧替身浮层组件，读的 hook 与 SWR 是真的，只替身取数。
- **预览的那次读**以 `{ revalidateOnMount: false, revalidateOnReconnect: false }` 加每次挂载一个 `mutate()` 发出（P5a 整审 I1 的修法，`IN/content/root.tsx:106-110`、`:128`）；`PeekContent` 照旧用默认的选项（浮层在别的页面上不变）。判定以这次挂载之后落定的读为准，SWR 留着的较早回答不算（P4 整审 m1）；读落定之前预览说在加载（`NotificationsLoader`）。
- **判定锁定**：判定在这次读第一次成功落定时记下，之后不再看这个键的 `isValidating` 与 `error`（浮层与嵌入两支都是）。理由：预览与浮层共用这个键，SWR 的 `isValidating` 按键共享；网络恢复时浮层自己的 hook 再验证（会话取数默认随重连重读，`use-session-swr.ts:26`、`:41-46`），用户按浮层的重试时不带数据的 `mutate()` 走这个键第一个登记的 hook（预览的）——照“落定 = 已答且不在验证”写，预览会画回加载、把浮层或嵌入连同没发出的输入卸下重挂，这是对每一张通知的回归，也是 P5a 整审 I1 的形状。
- **判定的代**（预检 m10）：判定连同它所属的代记下——权限身份（`root.permissionReads.identity(workspace.id, [projectId])`）与 `loginId`；预览以通知 id 为 key，所以代变了（换了会话、权限变了）记下的判定作废，以新代的读（同样 `mutate()`）重新判定。
- 读落定后看 canonical 的 `getIssueDetailById(issueId)`：`inIntake` 为假（不在收集箱、`intake` 为 `null`、已接受）→ `setPeekIssue` 并渲染浮层（今天的 `:93-95`、`:107-112` 挪到判定之后）；为真（待处理、暂缓、拒绝、重复）→ 渲染嵌入：`InboxContentRoot` 带嵌入的端口，不 `setPeekIssue`。
- **读的次数**：非收集箱的通知只有一个 `GET /api/v0/issues/{id}`。它在 `swr@2.4.2` 上成立，靠的是时间窗（预检 O2）：不带数据的 `mutate()` 先删去重标记、发新请求，回答之后标记再留 `dedupingInterval`（默认 2000 毫秒，`WEB_SWR_CONFIG` 不设它）；浮层挂上时的再验证带去重，在窗口里并入已答的那一次。窗口从回答算起，正常只隔几帧，主线程卡两秒以上才会多发一次。页面上的“恰一个”因此是带时间窗的断言，偶发时先看这里，不调去重间隔（v0 §7.7 只许列表缩短它）；页面上不成立先报主控（B-16）。收集箱条目是**两整包**（预检 O1）：判定的读就是浮层的取数（工作项，再加表情、链接、动态、评论、子工作项、关联与项目的状态，`issue.store.ts:71-106`），落定之后嵌入以自己的 `mutate()` 再读一遍（`IN/content/root.tsx:106-123`，P5a 整审 I1 的规则）；从点开到看见内容约是浮层的两倍，代价见 §6。
- **读失败**（O-1 乙）：预览自己画 `IssuePeekOverviewError`（按文件路径 `peek-overview/error.tsx` 导入，它不从桶文件导出；它里面的 `DetailResourceError` 经 `useDetailScope` 读 `rootIssueStore`，`resource-error.tsx:23`、`use-detail-scope.ts:12`，通知的测试要给得出），“关闭”经通知页的落点（R26）清选中；重试再读、落定之后再判定；浮层没有挂上。起草时“失败之后浮层挂上会再发一次读”的推断在 `swr@2.4.2` 下不成立（一个键已有 hook 挂着且带着错误时，新挂的 hook 不在挂载时再读，`shouldDoInitialRevalidation`，预检 m11）；O-1 乙的理由不靠它：判定以预览自己的读为准，失败的界面是预览的。

**嵌入的生命周期**（R17）：
- 每个选中（与它的代）判定一次，之后留在嵌入里；分诊的结果在嵌入里显示（接受之后是“已接受”、只读、“打开”到浏览页）。预览以通知 id 为 key，换选中即重新判定。
- 分诊（接受、拒绝、暂缓、改暂缓、取消暂缓、标重复）确认之后：不移选中、不动通知列表；焦点照 P5a R35 的“选中没有移”一支：新页头里这个状态的主动作，没有（已处理、非管理员）则嵌入的区域（`useDetailFocus(region).land("primary")`，`IN/content/header/detail-focus.ts:44-45`、`:62`）。接受弹窗关上之后同样（嵌入没有邻行）。
- 删除确认之后、条目读不到（`IN/content/root.tsx:145-158` 的那一支）、冲突的 `issue.intake_not_found` 之后：嵌入的 `leave` 经通知页的落点清选中（R26）。
- 关闭：页头左侧一个有名字的按钮（`t("close")`，“Close”，`common.json:185`），调 `selection.close`。toast 的关闭按钮也叫 “Close”（`web/apps/web/app/provider.tsx:35`）：故事把 “Close” 限在名为 “Preview” 的预览区域里定位（预检 m12）。
- 不给：上一条、下一条、↑↓（`useIntakeArrowKeys` 只在页面，`IN/root.tsx:62`）、窄屏的列表开关（同 Plane `pw/core/components/inbox/content/inbox-issue-header.tsx:231-235`、`:311`）。
- 已归档的项目（P5a R27 只读）、访客、只看自己的访客照收集箱详情既有的规则。嵌入第一次把收集箱的详情放到项目的布局之外：标签、成员、状态的下拉框都自己读目录（`label-select.tsx:34`、`member/dropdown.tsx:40`、`state/dropdown.tsx:30`），接受弹窗的迭代与模块也是，按代码不依赖项目包装层的取数；T9 的故事各断言一次（预检 O5）。
- 单测：嵌入的格进新文件 `IN/content/root.embed.test.tsx`（`root.test.tsx` 已 404 行）与 `WN/root.embed.test.tsx`；`WN/root.test.tsx` 按 plan 的允许清单改。

**通知页的落点**（R26，预检 I3、O7；细化 O-2 甲）：
- 新文件 `WN/landing.ts` 的 `useNotificationLanding()`，由 `NotificationsRoot`（`WN/root.tsx:33-51`，不随预览卸下）调，交出预览区域的 ref 与 `leave(notificationId)`。`leave` 只在选中仍是这条通知时记下要落的通知并 `clearSelection(notificationId)`（`WN/page-state.tsx:68-69`）；选中被清掉的那次提交之后一个微任务里落焦点——排在 Headless 把焦点还给随嵌入卸下的菜单按钮之后（它在卸下的 effect 之后一个微任务里做），同 `IN/landing.ts:47-48`、`:71-82` 的次序（P5a 的 D2）。
- **链**：那张卡片的主按钮（`[data-notification-id="<id>"] button`，列表的既有约定，`WN/sidebar/list.tsx:70`，`use-leaving-focus.ts` 同）→ 通知列表的区域（`WN/sidebar/list.tsx` 的两个区域元素 `:53`、`:65` 加 `data-notification-list` 标记；列表读失败或还在读时不画它，`:49-50`）→ 预览的区域（`WN/root.tsx:41` 的 section 加 ref 与 `tabIndex={-1}`，它已名 “Preview”）。绝不落到 `body`。
- **谁用它**：嵌入的 `leave` 与 `close`（删除、关闭、读不到、`issue.intake_not_found`）；预览失败界面的关闭（O-1 乙）；**普通浮层的关闭**（预检 O7）：浮层嵌在通知页里时关闭调 `embedRemoveCurrentNotification`（`peek-overview/view.tsx:96`），今天不还焦点（`:97` 的 `!embedIssue`），焦点落在 `body`；预览把 `landing.leave` 交给它，关浮层与关嵌入落在同一处。这是 P4 页面行为的改变（§7 第 8 条）。
- 选中跨页签与筛选保留，“卡片不在”是常见的情形：T9 的故事先换页签再关。
- P4 的 `useLeavingFocus`（卡片离开列表时的落点）不变：那时焦点在卡片上；这里焦点在预览里，两者不相交。

### 2.6 负责人多选的连点与下拉框（R19–R21）

- **Q34 按代码必然重现**（预检 I5）：多选是受控的，新集合从渲染出的值算（`web/packages/ui/src/dropdowns/custom-search-select.tsx:277-286`，`C/components/dropdowns/member/base.tsx:116-122`）；调用方把整集原样交给写的层（`IN/content/issue-properties.tsx:81-83` 等）；写的层的 `changeBody` 把 `assignee_ids` 原样带过（`issue-write-fields.ts:75`）；队列一次发一个，canonical 只在确认时变（`C/store/issue/issue-details/issue.store.ts:149-152`）。所以扣住第一次 `PATCH` 的回答再点第二位成员，第二次的请求体必然只有第二位，最终集合丢掉第一位，不靠时序。这违背 v0 §7.7“请求体轮到时才算”（`docs/v0/v0-design.md:638`）。
- **修在写的层**（T8c，R20 (b)，**高风险的局部裁定，动 M4 的共用写层**）：
  - `IssueChange`（`web/packages/types/src/issues/issue-change.ts:17`）加负责人的增删，形状同模块集合（`ModuleSetChange`，`:9`），与整集的 `assignee_ids` 类型上互斥；
  - `changeBody`（`issue-write-fields.ts:70-84`）的第二个参数由所持的模块集合改为所持的工作项，增删在轮到时并到所持的负责人集合上（同 `modulesAfter`，`C/lib/planning-directory.ts:67`）；请求体仍是整集，合同不变；
  - `changeBody` 的三个调用方传所持的工作项：`issue-writes.ts:143`（`changeIssue`）、`helpers/base-issues.store.ts:94`（拖放：它的 `makeChange(held)` 已在轮次里从所持的工作项算出整集，照旧用 `assignee_ids`）、命令面板的 `work-item/writes.ts:45`（它的指派本来就在轮次里算，`mutation-commands.ts:36-39`）；
  - 有负责人下拉框、改已有工作项的**七个调用方**改为送增删，增删相对“被点时显示的值”（那次渲染所持的 `assignee_ids`）算：`IN/content/issue-properties.tsx:83`、`C/components/issues/peek-overview/properties.tsx:93`、`issue-detail/sidebar.tsx:97`、`issue-layouts/properties/all-properties.tsx:73`、`issue-layouts/spreadsheet/columns/assignee-column.tsx:32`、`issue-detail-widgets/sub-issues/issues-list/properties.tsx:216-219`、`relations/properties.tsx:52-56`。预检列了前五个；后两个是 `git grep MemberDropdown` 找到的，同样经 `projectIssues.updateIssue` → `changeIssue`（`helpers/base-issues.store.ts:73-75`、`issue-details/sub_issues.store.ts:177-185`）。写的层只能在轮到时把增删合到所持的集合上，增删只能由每个调用方算：起草时“一处改好所有调用方”做不到。
  - 不改：新建与编辑的表单（`issue-modal/components/default-properties.tsx:112`，提交时整集是用户的选择）、草稿（`workspace-draft/draft-issue-properties.tsx:125`，另一条写路）、分组的新建载荷与拖放（`issue-layouts/group-columns.tsx:314`）。
  - 类型与帮手的名字（同形的两份，或合成一份按 id 集合的改动）在 T8c 的方案里定；方案先给主控看，再写代码（L! 的流程）。
  - 证据：单测（轮次里的合并、排在一个负责人写之后的第二次点击的请求体、每个调用方送增删）与变异；H11 (page) 的连点一步先在没修的构建上跑红（重现的收据），修好之后三引擎绿；chromium 的 m4-detail 与 m4-list（列表行与表格的负责人下拉框在 m4-list，`E/playwright.m4-list.config.ts`）。不扣住的“快速连点”只能靠时序，不作证据，不写。
- **连点放在哪**（R19）：收集箱详情上（管理员、待处理的条目），与 R21 的下拉框核对同一个 `test()`，放进 H11 的一个短的新文件（§2.8）。
- **下拉框的页面核对**（R21）：详情的负责人下拉框以键盘核：Enter 打开、列表在按钮旁（`expectPopupBesideButton`）、搜索取得焦点、Escape 关上并回到按钮；新建弹窗的那一个在 H8 (page)。收集箱页没有右键菜单（`git grep ContextMenu -- web/apps/web/core/components/inbox` 为空），右键一条写明不适用、留给 P6a；焦点轮廓交 M8。核过即关闭 `HO/M3-closeout.md:51` 的收集箱一半。

### 2.7 守卫与告警

- **oxlint**：web 的上限 18（`web/apps/web/package.json:10`）是等式；新文件零告警；删掉的写法若带告警（`create-root.tsx:2-3` 的两条 `oxlint-disable` 管的 `div role="button"`），同一提交照实改上限并留计数的收据。e2e 的上限 0。
- **knip 与第一个读者**：新文件、新导出在第一个读者出现的 Task 里建立（plan 列出）；`tab-indices.ts` 去掉收集箱一组之后 `ETabIndices` 与函数仍有视图一个读者，knip 不报。
- **swr 名单**（`.oxlintrc.json:120-210`）：新的读 hook `peek-overview/use-peek-issue-read.ts` 收入；`WN/root.tsx`（`:198`）、`peek-overview/root.tsx`（`:162`）已在。
- **地址规则**：`IN/content/widgets.tsx` 读项目与角色，收入 `project-address-route`（T4）；T6a 搬出的新文件若读项目同样收入（`use-intake-actions.ts` 在它的范围里）；`IN/address.ts` 只拼地址、`WN/landing.ts` 只落焦点，都不读 store。每次改 `K` 跑 `node tools/keywords.mjs`。
- **文案**：新键只有 `inbox_issue.modals.discard.title`、`.content`（en、zh-CN 同一提交；与 `inbox_issue.modals.*` 的既有键是兄弟，不成前缀）；别的一律用已有的键。
- **死成员**：T1 的基线（rulings.md 末条）、T2 之后的整份比对（P5 的 6 行归零；R32 的两个 prop 随读者回来、不新出死行），T6a、T6b、T8c 之后各一次，T10 的终表。

### 2.8 e2e

- **配置**：新文件都在 `E/stories/collaboration/`，名字合 `E/playwright.m7-pages.config.ts:10-11` 的 `testMatch` 与 `grep`（每个 `test()` 的标题带 `H<n> (page)`），三引擎；`pageFileStory`（`E/fixtures/file-story.ts:36-43`）给 `[local]` 与 `[s3]` 两个变体，m7-pages 不排除 `[s3]`（它的 S3 由 `E/global-setup.ts:15-28` 起），所以 `[s3]` 也三引擎（Q21；M5 的先例只在 chromium 跑 `[s3]`，`docs/v0/M5-files/reviews/P7b-content-assets-review.md:126`）。CI 以默认配置在 chromium 跑一次（两个变体）。
- **检查点与终检跑六个页面配置**（R2）：m7-pages、m6-pages、m6-acceptance、m4-detail、m5-content（`E/playwright.m5-content.config.ts`，F8–F12、F16 三引擎，`[s3]` 被 `grepInvert`）、m4-list（`E/playwright.m4-list.config.ts`，T8c 动了列表行与表格的负责人下拉框；检查点 0、1 的那一次作 T8c 之前的基线）。
- **夹具**：编辑器代码块的扣住从 M4 的故事搬到新 `E/fixtures/editor-chunks.ts`（R8：`E/stories/issue/editor-loading.spec.ts:14-53` 的 `editorChunks` 与 `holdEditorImplementation`，M4 的故事改为导入，断言不改）。“逐字”之外只许两处（预检 m8）：`editorChunks` 在故事里另有两个读者（`:160`、`:215`），一起导出；它按 `import.meta.url` 找构建产物（`:17` 的 `../../../web/…`），从 `E/fixtures/` 起少一层（`../../web/…`）。新建弹窗的帮手进新 `E/fixtures/intake-create.ts`；收集箱宿主上的文件故事的登录与传输核对进新 `E/fixtures/intake-files.ts`（以 `intakeWorld(world.api, info)`，`E/fixtures/intake-pages.ts:32`，从 `E/fixtures/content-files-page.ts` 导入 `pngFile`、`chooseAttachment`、`pasteInto`、`downloadedExactly`）；`intake-pages.ts`（344 行）不加长。
- **编辑器代码块的扣住**（预检 I2）：帮手靠拦住代码块的请求来扣住；收集箱页只要列表有行就自动选中第一行（`IN/root.tsx:74-85`），详情的 `DescriptionInput` 用的是同一个惰性入口（`C/components/editor/rich-text/editor.tsx:11`，新建弹窗的 `issue-description.tsx:18` 同一个），代码块在弹窗打开之前已经载入，之后弹窗的 `import()` 不再发请求，`held` 一直是空的，编辑器立刻建成——“Escape 关空表单”“Discard 关空表单”照样通过，关闭的证据就是假的。所以关闭的故事照 C10 的次序（`editor-loading.spec.ts:59-64`、`:107-110`）在**页面载入之前**登记扣住，页面载入时收集箱是**空的**（没有行，详情不挂编辑器）；每个关闭与提交的步骤之前断言 `held.length > 0` 且弹窗里没有 `[contenteditable="true"]`（`:116-117`）。这一步的红色版本：去掉扣住之后，“提交被拒、没有 `POST`”一步失败。
- **故事**：

| 文件 | 标题 | 覆盖 |
|---|---|---|
| `h8-intake-submit-page.spec.ts` | `H8 (page)` | 管理员、成员、访客经弹窗提交：请求体精确、数据库的条目行与工作项、跳到新条目并选中、列表与计数的重取（`expect.poll` 内容）；弹窗以 “Create intake work item” 为名；名字超长（`FieldError` “Too long”、`aria-invalid`、边打边显示）与空白（空的与只有空格的各一次，提交之后 “Title is required”）：提交被拒、不发请求、焦点在名字上；收集箱在弹窗开着时被关掉 → 提交答 `issue.intake_disabled` 的提示、表单留着；“Create more”（开关、成功之后清空不关） |
| `h8-intake-close-page.spec.ts`（若与上一份合起来不过 400 行可合并） | `H8 (page)` | 页面载入之前扣住编辑器代码块、收集箱是空的（I2）：每步之前 `held.length > 0`、没有 `[contenteditable="true"]`；Escape 关空表单，再开一次 “Discard” 关空表单，提交被拒（提示、没有 `POST`）；有改动时 Escape、点背景、Discard 各一次弹确认框（以标题为名），取消回到表单且输入还在，确认之后关上、焦点回到 “Add work item”，再开是空的；没有改动时点背景直接关；提交在途（`holdAnswer`）时 Escape、点背景关不掉、Discard 停用，Escape 之后、点背景之后与被拒之后焦点都在弹窗里（R28）；嵌套 Escape：负责人（Combobox）与截止日（Popover）开着时 Escape 只关它们（R7）；负责人下拉框以键盘核（R21）；Tab 的自然次序（名字 → … → 提交）；焦点在弹窗里 |
| `h11-intake-images-page.spec.ts` | `H11 (page)`，`pageFileStory` 的 `[local]`/`[s3]` | 访客提交者在描述里加图片：上传、`PATCH /api/v0/intake-issues/{id}` 带资产 id、本机文件或桶里的对象字节相同；不是提交者的成员：描述只读、没有上传；**在途被拒**：扣住 `POST /api/v0/assets`（`createAsset`）的回答，另一会话经接口拒绝条目，放开之后 `POST /api/v0/assets/{id}/complete`（`completeAsset`）答 409 `issue.intake_closed` 并提示，这个 409 写进 `watch.apiFailures` 的期望（预检 m7）；**接受之后照 `issue.update`**：在浏览页（`/browse/…`，已接受的不重定向）上，访客提交者只读、成员可传（收集箱的详情对已接受的条目对谁都只读，预检 m5） |
| `h11-intake-attachments-page.spec.ts` | 同上 | 访客提交者加附件、下载字节相同、`ConfirmedContentNotice` 说出；不是提交者的成员没有附件的选择器；收集箱关掉：扣住 `createAsset` 的回答，经接口关掉收集箱，放开之后 `completeAsset` 答 409 `issue.intake_disabled` 并提示（写进期望）；传输中取消；传输在途时换选中，迟到的结局不写进新条目；删除附件，本机文件与桶里的对象随之没了（清理任务若在页面故事里触发不了，写明由 M5 的清理故事 `E/stories/files/f15-cleanup.spec.ts` 覆盖） |
| `h11-intake-links-assignees-page.spec.ts` | `H11 (page)` | 链接（R10）：不是提交者的成员在已拒绝的条目上加、改、删链接；访客只读（T8b）；详情的负责人下拉框以键盘核（R21，T8b），同一个 `test()` 里接着连点：扣住第一次 `PATCH /api/v0/intake-issues/{id}` 的回答、点第二位、放开，两次请求体与数据库里的最终集合含两位（T8c；先在没修的构建上跑红） |
| `h10-intake-embed-page.spec.ts` | `H10 (page)`（重定向一步 `H20 (page)`） | 通知页嵌入待处理条目的收集箱内容：没有 “Close the peek view”，有收集箱的页头（状态、复制、打开）与名为 “Close” 的按钮（在 “Preview” 区域里定位）；已有的标签与负责人显示出来，接受弹窗的状态有默认值（O5）；管理员在嵌入里接受、拒绝、暂缓各一次之后的样子（选中不移、通知列表不变、焦点落在主动作或嵌入的区域）；成员与访客只读的页头；已拒绝的也嵌入；已接受的开普通浮层，关上之后焦点在那张卡片的主按钮（O7）；嵌入里没有上一条、下一条与 ↑↓；“打开”到收集箱页并选中它；关闭与删除之后清选中、焦点落在卡片的主按钮，先换页签再关则落在列表的区域（R26）；800 宽一次；收集箱条目恰两个 `GET /api/v0/issues/{id}`、普通的恰一个；**重定向**：打开待处理条目的浏览地址 → 换成收集箱页并选中它，拒绝的到“已处理”页签，后退不回浏览页 |
| `h2-h4-notification-cards-page.spec.ts`（既有，298 行；随 T6b 改，O-4） | `H2–H4 (page)` | `:257-264` 的一步改为断言嵌入（`:49` 的注释与 `:162` 的标题里 “opens a plain peek” 随之改写）：bob（成员，不是提交者）点开 Crash on save 的卡片，出现收集箱的页头（“Pending”、复制、打开）与 “Close”（限在 “Preview” 区域里），没有 “Close the peek view”，没有 Accept/Decline；点卡片之前登记嵌入最后几个读的 `waitForResponse`，离开项目（`:270`）之前等它们答完（预检 m12：末尾对 `watch.apiFailures` 是精确比对，`:292-296`） |

- 不改：`h9-intake-list-page.spec.ts`（398）、`h11-intake-content-page.spec.ts`（397）、`h20-intake-*.spec.ts`。H10 (page) 的 `expectNoTriage` 照 R15 改一处（允许清单）。
- **时长**：本机 chromium 每个文件约 8 秒是先例的上限（P4、P5a）；`[s3]` 的测试每个自起一个 nerve 与桶（`E/fixtures/s3-world.ts`），会超，写明原因；任何一个 `test()` 单引擎超过 30 秒先报主控。CP2 之前照 P4 spec §2.11 `:270` 的方法估出 e2e 两片的时长。

### 2.9 文件与职责

| 文件 | 职责 | Task |
|---|---|---|
| `IN/modals/create-modal/{modal,create-root,issue-title,issue-description,issue-properties}.tsx` | 关闭规则、在途不给 `handleClose`、确认放弃、迟到的跟进、名字的提示与被拒、`ModalTitle`、开关、英文改键、离开 `getTabIndex`、删死代码 | T2 |
| `IN/modals/create-modal/fake-create-modal.tsx`（新） | 三份新测试的共用夹具 | T2 |
| `web/packages/constants/src/tab-indices.ts` | 删收集箱一组 | T2 |
| `C/store/issue/intake/status.ts` | `intakeEditableBy` | T3 |
| `C/store/asset/content-view.ts` | `canWrite` 的 `issue` 宿主认收集箱的规则（M5 共用件，L!） | T3 |
| `IN/content/detail.tsx`、`IN/content/root.tsx` | 读谓词与图片；`ConfirmedContentNotice`；渲染 `widgets.tsx` | T3、T4 |
| `IN/content/widgets.tsx` | 附件与链接 | T4 |
| `C/components/issues/peek-overview/properties.tsx`、`issue-detail/sidebar.tsx` | 隐去状态 | T5 |
| `A/browse/[workItem]/layout.tsx`、`IN/address.ts`、`create-root.tsx:146` | 重定向与收集箱的地址 | T5 |
| `IN/content/header/{use-intake-actions.ts,root.tsx}` | “打开”只给已接受的 | T5 |
| `IN/content/header/intake-actions.ts`（新）、`IN/content/selection-port.ts`（新）、`IN/content/root.tsx`、`IN/content/header/{root,mobile,use-intake-actions}.ts(x)`、`IN/root.tsx` | 纯搬移；选中的端口 | T6a |
| `IN/content/header/*`、`IN/content/root.tsx` | 嵌入的页头与关闭、分诊之后不移、`openLink` | T6b（收集箱一侧） |
| `C/components/issues/peek-overview/use-peek-issue-read.ts`（新）、`peek-overview/root.tsx`、`WN/root.tsx`、`WN/landing.ts`（新）、`WN/sidebar/list.tsx` | 读的 hook；判定、锁定与代；失败；通知页的落点（嵌入、失败、浮层的关闭） | T6b（通知一侧） |
| `E/fixtures/editor-chunks.ts`（新）、`E/fixtures/intake-create.ts`（新）、H8 的故事 | H8 (page) | T7 |
| `E/fixtures/intake-files.ts`（新）、H11 的两份文件故事 | H11 (page) 的资产 | T8a |
| H11 的链接与下拉框的故事 | R10、R21 | T8b |
| `web/packages/types/src/issues/issue-change.ts`、`C/store/issue/issue-write-fields.ts`、`issue-writes.ts`、`helpers/base-issues.store.ts`、命令面板的 `writes.ts`、七个调用方；H11 的连点一步 | 负责人集合在轮到时由增删算（M4 共用写层，L!） | T8c |
| H10 的嵌入故事 | 嵌入、落点、重定向 | T9 |

## 3. 局部裁定

主控的裁定在台账 `rulings.md`（B-1–B-23 与起草时的补充），预检 re-95 之后的裁定写在本节 R26–R30 与各条的修正里；都是局部裁定。两条动了别的 M 的网页共用代码，是**高风险的局部裁定**，阶段记录写明，收官报告列出：**R9**（B-9，M5 的 `ContentView.canWrite`）与 **R20**（B-20 的 (b)，按预检 I5 确定要做，M4 的共用写层）。P5a 的裁定（Q12、Q13、Q14、Q21、Q34、Q36，S5a 的 R19、R23、R25、R27、R32、R35）照旧约束本段。R24 是起草时照 `research/p5a-split.md` §3 的 T2 落的细节，R25 是起草时为让 B-16、B-17、B-22 能落地而定的细节；§8 是主控对起草者 O-1–O-4 的裁定。R26–R30 是主控对预检 re-95 的裁定（2026-11-24）。

| B | R | B | R |
|---|---|---|---|
| B-1（与 T1 的回填） | R1 | B-13 | R13 |
| B-2 | R2 | B-14 | R14 |
| B-3 | R3 | B-15 | R15 |
| B-4 | R4 | B-16 | R16 |
| B-5 | R5 | B-17 | R17 |
| B-6 | R6 | B-18 | R18 |
| B-7 | R7 | B-19 | R19 |
| B-8 | R8 | B-20 | R20（(b) 为 L!） |
| B-9 | R9（L!） | B-21 | R21 |
| B-10 | R10 | B-22 | R22 |
| B-11 | R11 | B-23 | R23 |
| B-12 | R12 | 起草时 | R24、R25 |
| re-95 I3、O7 | R26 | re-95 I4、m10 | R27 |
| re-95 m2 | R28 | re-95 m1 | R29 |
| re-95 m9 | R30 | | |

- **R1 形态与回填**（B-1 的 (b)）：14 个 Task：T1 回填与文档 → T2 新建弹窗（含 `getTabIndex` 与常量的收集箱一组，**检查点 0**）→ T3 共用谓词、`canWrite`、描述图片与 `ConfirmedContentNotice` → T4 附件与链接 → T5 隐去状态、重定向、“打开” → T6a 纯搬移与端口 → T6b 嵌入与通知页的落点（**检查点 1**：除 T8c 之外的产品代码完成）→ T7 H8 (page)（先搬编辑器代码块的帮手）→ T8a H11 (page) 资产 → T8b 链接与详情的下拉框 → T8c 负责人集合在轮到时算（产品代码，在检查点 1 之后，L!）与连点的一步 → T9 嵌入的故事、H2–H4、重定向（**检查点 2**）→ T10 文档（主控）→ T11 门禁、整审与集成（主控）。B-1 的 T3 并进 T2：同四个文件，分开时 T2 的开关已离开 `getTabIndex` 而别的控件还带正的 `tabIndex`，Tab 次序有一段是乱的；代价是 T2 大一些。B-1 的 H11 资产一条拆为 T8a、T8b：文件故事两个变体三引擎、证据多，链接与下拉框不是文件，另成一个短文件。T8c 是预检 I5 加的：连点按代码必然重现，修复是计划内的产品改动；它放在检查点 1 之后，因为它的页面证据（连点的一步）要先在没修的构建上跑红，而那一步与 T8b 的下拉框核对同一个 `test()`；检查点 2 与终检跑 m4-detail 与 m4-list 兜住它。仍 ≤ 16（`D:291`）。T1 回填 P5a 的 A9 与阶段记录（数字取自归档台账 `/Users/xiaoruan/project/nerve-project/.superpowers/sdd/M7-P5a-integration-20261123/payload/worktree/.superpowers/sdd/P5a-web-intake/progress.md:683-687`）；设计 §12、§14 的行由主控改。
- **R2 检查点与页面配置**（B-2 的 (b)）：三个检查点在 T2、T6b、T9 之后：本机七项、六个页面配置（m7-pages、m6-pages、m6-acceptance、m4-detail、m5-content、m4-list）、分支 CI；新故事文件 chromium `--repeat-each=3`。第 1 路以 P5b 检查点 0 的本机值为基线、+30 秒停查；CI 的八成、web 作业 960 秒、e2e 每片 30 分钟只报告（P5a R25）。web 作业越线时对照 main CI `45197451256` 的三步（Lint 93 / Unit 818 / Build 17 秒，`.376` 映像；它自己 959 秒，贴着报告线）：三步同比例变慢是 runner，只有 Unit tests 变长才是本段的增长，先报告再继续；基线本身是慢 runner 的，快 runner 上三个比值都小于 1，照样看三者是否相近（预检 O6）。推送之前照 P4 spec §2.11 估 e2e 两片。理由：动了 M5 的共用件就跑 M5 的页面故事（同 P5a 动 M4 的写队列就跑 m4-detail）；T8c 动了列表行与表格的负责人下拉框，就跑 m4-list，检查点 0、1 的那一次是它的基线。代价：每个检查点多 m5-content 与 m4-list 两个配置（估约 2 分钟与几分钟，未测）。
- **R3 新建弹窗里的死代码**（B-3 的 (a)）：§2.1。负责人的规则（不用的代码删掉，不留开关与桩）；P5 的死成员归零。
- **R4 关闭政策**（B-4 的 (b)）：§2.1 的表。理由：`handleClose` 让点背景也能关，是新的丢输入的路，交接明确要“统一 dirty/pending/busy、输入保留、嵌套 Escape”（`HO/M4-closeout.md:69`，`HO/M3-closeout.md:90`）；不改 M3 的 `ModalCore`；规则与 M4 同形（只是没有存草稿）。在途时的实际结果见 R28。代价：一个确认框、两个新键、H8 多几步；登记 plane-diff（Plane 点背景不关、文档 Escape 不问就关，`pw/core/components/inbox/modals/create-modal/modal.tsx:29-42`）。
- **R5 名字的提示与被拒**（B-5 的 (b)）：§2.1。规则同合同（去空白之后的字符数），与 M4 的两处差别写明（预检 m3）；删 `required`。理由：停用的按钮表达不了“被拒”，也收不回焦点（P4 的教训）；迁 react-hook-form 改动大、收益只在一个字段。登记 plane-diff。
- **R6 弹窗里的英文**（B-6）：全部改走已有的键，不加新键；H8 按新文字定位。
- **R7 嵌套 Escape 核到哪**（B-7 的 (b)）：H8 (page) 三引擎核负责人（Combobox）与截止日（Popover）各一次；编辑器菜单在 `ModalCore` 里的 Escape 是 M4 的共用行为，不核。代价：优先级与标签两个下拉框只靠同构推断（Headless 的四个原件只有两种）。
- **R8 编辑器代码块的帮手**（B-8）：从 M4 的故事搬到 `E/fixtures/editor-chunks.ts`，一个提交、断言不改，H8 复用；“逐字”之外的两处见 §2.8（预检 m8）；H8 的用法照 C10 的次序（预检 I2）。代价：动 M4 的故事文件（只改导入），以 m4-detail 配置三引擎跑 `editor-loading.spec.ts` 一次作证。
- **R9 前端的写权限认收集箱的规则**（B-9 的 (a)，**L!**）：§2.2。理由：只在详情传 `readOnly` 不够，访客提交者仍过不了 `canWrite`（`root.store.ts:55` 以“换了会话”中止）；新宿主种类违背 M5-M7-01 的 P2c 处理结果。代价：改 M5 的共用件一支；`content-view.test.ts` 加收集箱的格与变异（含 `canRemoveAttachment`）；T3 之后另跑 chromium 的 m5-content，每个检查点跑三引擎的 m5-content。
- **R10 链接谁能写**（B-10 的 (a)）：§2.3。理由：与合同、设计 `D:109` 和 P5a 对评论、表情的做法一致；照收集箱的内容规则会比服务端严、要另写一条前端规则。登记 plane-diff（Plane 的收集箱详情没有链接，Q14）。
- **R11 附件与链接的组合**（B-11 的 (b)）：§2.3；`ConfirmedContentNotice` 画在宿主之内（§2.2）。理由：不给 M4 的组件加收集箱分支。
- **R12 H11 资产的证据范围**（B-12 的 (b)）：M5-M7-01 的“同等的证据”（`HO/M5-closeout.md:22`）按类别在收集箱的宿主上各取有代表性的步骤（§2.8 的两份文件），`[local]` 与 `[s3]` 三引擎；“接受之后”一步在浏览页上（预检 m5）；在途被拒扣的是 `createAsset` 的回答、被拒的是 `completeAsset`（预检 m7）。理由：P5b 对共用的上传件只改“谁能写”一支，分页、拖放、复制、签名续期等 M5 已有三引擎证据，逐条重做 F8–F16 是重复；m5-content 在每个检查点跑，保住 M5 自己的证据。`[s3]` 在 Firefox、WebKit 里若因 M5 的上传件失败，是跨模块的发现，停下报告。
- **R13 通用详情与浮层只隐去状态**（B-13 的 (a)）：§2.4。理由：窄缝里其余的写已说 nerve 的同一句（P5a 整审 m3）；全隐要动 `view.tsx`、页头与 `IssueDetailWidgets` 多处，为一条几乎走不到的路加分支。
- **R14 浏览页的重定向**（B-14）：§2.4。在布局、`replace`、按状态选页签、关着照样送。两个后果写在 §1 末与 §7 第 3 条（预检 O3）。理由：放在页面的 effect 里，详情宿主会为收集箱条目挂一下；`push` 时后退回到浏览页又被送走。登记 plane-diff（Plane 只送待处理、暂缓，以 `push`，在页面的 effect 里，`pw/app/(all)/[workspaceSlug]/(projects)/browse/[workItem]/page.tsx:88-92`、`pa/app/views/issue/base.py:1317-1325`）。
- **R15 “打开工作项”只给已接受的**（B-15 的 (a)）：§2.4、§2.5。理由：重定向之后，对未被接受的条目按“打开”会回到同一个收集箱页。代价：P5a 的 `header.test.tsx` 矩阵与 H10 (page) `:58` 跟随；登记 plane-diff。
- **R16 嵌入的判定**（B-16 的 (a)）：§2.5。理由：判定以这个预览自己的读为准（P4 m1），不闪，非收集箱的通知不多读；动 M4 浮层的只是把读的 hook 搬进它自己的文件。“只一个读”靠去重的时间窗（预检 O2，§2.5 写明）；页面上不成立先报。失败一支照 O-1 乙（§8）。代价：收集箱条目的通知是两整包读（预检 O1，§6）。
- **R17 嵌入的生命周期与界面**（B-17）：§2.5。判定一次（R27）、分诊之后不移、删除之后清选中、一个有名字的关闭按钮、不给导航；落点照 R26。理由：同 Plane；随 canonical 换成浮层会重挂、焦点跳走。
- **R18 选中的端口**（B-18 的 (b)）：§2.5。先纯搬移，再端口（行为不变），再嵌入。理由：去掉“组件内部读页面 provider”的耦合，比到处判断布尔值干净；R32 只要求加回的东西有读者。
- **R19 连点放在哪**（B-19 的 (a)）：收集箱详情上连点两位成员，与 R21 的下拉框核对同一个 `test()`，放进 H11 的一个短的新文件（§2.8）；下拉框的一半在 T8b，连点的一半在 T8c。
- **R20 修在写的层**（B-20，(b) 为 **L!**；按预检 I5 确定要做）：§2.6。负责人集合改为增删、轮到时并到所持的集合上；增删由七个调用方相对被点时显示的值算，`changeBody` 与它的三个调用方传所持的工作项。理由：v0 §7.7 的“请求体轮到时才算”，模块集合与命令面板的指派是现成的先例。代价：M4 每个从下拉框改负责人的写都经改动的路径；方案先给主控看；单独的 `fix` 提交先红、变异、chromium 的 m4-detail 与 m4-list，检查点 2 与终检跑六个配置与整套默认 e2e。
- **R21 下拉框的页面核对**（B-21）：§2.6。核过即关闭 `HO/M3-closeout.md:51` 的收集箱一半；右键菜单不适用（P6a），焦点轮廓交 M8。
- **R22 交接**（B-22）：§5 的表。引用 `HO/M3-closeout.md` 今天的行号（S5a `:422` 与 R5a `:146` 引的 `:40-41`、`:47`、`:55`、`:56` 是 P4 基线 `89be91e5` 时的行号）。M7-04 只关收集箱部分、保持 open；收集箱部分逐项写明本段处理的与留下的（预检 m1，§5）。M7-05 与 M6-M7-05 写“收集箱的四个读者已去掉，最终删除随 P6a”。`:47` 的收集箱一半照 O-3 关闭。
- **R23 plane-diff**（B-23）：§7 的七行，加预检 O7 的第 8 行（P4 页面行为的改变，Plane 同 P4）。
- **R24 “Create more” 是开关本身**（起草时，照 `research/p5a-split.md` §3 的 T2 与 M4 `form-footer.tsx:66-77`）：今天的 `div role="button"` 包着 `Switch`（`create-root.tsx:194-202`）是按钮里套开关，读屏读出两个控件、Tab 停两次（`HO/M3-closeout.md` §3 的嵌套按钮规则）。代价：H8 按开关定位；不登记 plane-diff（可访问性的修正，同 P5a 不登记浮层位置的修复）。
- **R25 起草时补的细节**：(1) 关闭规则的“有改动”按 M4 的 sanitize 规范化（描述用 `isEmptyHtmlString`），否则编辑器建成时写下的空段落会让空表单也要确认；(2) 共用谓词要求调用者有项目角色（合同 `intake.update` 的三种角色）；(3) 选中某一条的收集箱地址由 `IN/address.ts` 一处给出，重定向、新建弹窗的跳转与嵌入的“打开”共用（预检 m9 的第二处）；(4) `offeredOn` 以端口的 `kind` 区分页面与嵌入的“打开”（不另加端口成员；打开的地址见 R30）；(5) 成功的提交直接关弹窗、不经“有改动”。
- **R26 通知页的落点**（re-95 I3、O7；细化 O-2 甲）：§2.5。一份不随预览卸下的落点，链是卡片的主按钮 → 列表的区域 → 预览的区域，在清选中的那次提交之后一个微任务里落；嵌入的删除、关闭、读不到，预览失败界面的关闭，普通浮层的关闭都走它。理由：只说“卡片，否则列表的区域”时，预览够不着列表的 ref，列表没画出来时断在 `body`，时序也没写；浮层的关闭顺手接上，同一页上不出现“关嵌入落在卡片、关浮层落在 `body`”。代价：动 P4 的 `WN/root.tsx`、`WN/sidebar/list.tsx`（一个标记）与浮层在通知页上的关闭行为（§7 第 8 条）；每一环一格一个变异。
- **R27 判定的读与锁定**（re-95 I4、m10）：§2.5。读的 hook 在它自己的文件；预览的读关挂载读与重连重读、每次挂载一个 `mutate()`；判定在第一次成功落定时锁定，之后不看这个键的 `isValidating` 与 `error`；判定按（通知、权限身份、`loginId`）记，代变了重新判定。理由：共用的键上任何一次再验证都会让照字面实现的预览撤下浮层或嵌入连同没发出的输入（P5a 整审 I1 的形状）；把读的 hook 放进自己的文件，通知的测试才能在真实的 SWR 上钉住它。代价：多一个文件；`WN/root.test.tsx` 的替身跟随（允许清单）；真实 SWR 上的格与两个变异。
- **R28 新建弹窗在途不给 `handleClose`**（re-95 m2）：§2.1。理由：留着 `handleClose` 时 Headless 在调它之前先 `blur`，`keepFocusIn` 把焦点给对话框本身，原来“提交按钮恢复可用时还给它”的等待被换掉，被拒之后焦点停在对话框上、不在任何字段；不给 `handleClose`（同 `AlertModalCore`）时 Escape 不动焦点，表里“在途不关”才是实际的结果。代价：`create-root.tsx` 多一个 `onSubmittingChange` prop，两份既有测试编译跟随；H8 三引擎断言焦点。
- **R29 迟到的跟进与 IME**（re-95 m1）：§2.1。新建弹窗的跟进只在表单的内容还挂着时做（卸下即离开了这一页或这个项目），以一个挂载的范围写；IME 组字时的 Escape 由 `ModalCore` 阻止，一格钉住。M7-04 里本段处理与留下的逐项见 §5。理由：后退、前进不受弹窗约束，迟到的成功会跳回原项目；`useDetailScope` 读 `rootIssueStore`，既有测试的 store 替身给不出。代价：一格一个变异；条目在 nerve 已建成而页面不说（同 M4 弹窗的表单范围）。
- **R30 打开的地址**（re-95 m9）：§2.5。动作模型多一个 `openLink`，“打开”的两个读者读它，复制仍读 `link`。理由：嵌入里“打开”与“复制”的地址不同了，动作模型今天只有一个 `link`、注释写着两者共用。代价：`IntakeActions` 多一个成员。

## 4. 验收标准

- [ ] **A1 新建弹窗的关闭**：Escape、点背景与 Discard 只经 `requestClose`；在途不给 `handleClose`、`requestClose` 也不关，Escape、点背景与被拒之后焦点都在弹窗里（H8 三引擎）；编辑器没建成可关；已建成且未就绪拒绝（单测）；有改动先确认（名字去空白、描述按 `isEmptyHtmlString`、优先级、负责人、标签、截止日），没有改动直接关；确认放弃之后再开是空的；没有文档级的 Escape；IME 组字的 Escape 不关；嵌套 Escape 只关最内层；提交仍要求就绪；迟到的跟进在卸下之后不跳转、不提示；每条有变异；H8 (page) 三引擎（页面载入之前扣住编辑器代码块、空的收集箱，每步之前 `held.length > 0`）。
- [ ] **A2 新建弹窗的名字与其余**：弹窗以 `ModalTitle` 为名；`required` 删去；名字的 `FieldError` 与 `aria-invalid`、按 `Array.from(name.trim())` 判空白（空的与只有空格的）与超长、提交不停用、被拒取得焦点、nerve 的 `validation_failed` 在字段下；“Create more” 是开关本身、在途停用；英文全部改键、只有两个新键（en、zh-CN）；四个文件离开 `getTabIndex`、自然 Tab 次序、常量的收集箱一组删除；死代码删除、P5 的死成员归零；新的格分三个文件、每份不过 400 行。
- [ ] **A3 写权限与图片**：`intakeEditableBy` 一处，详情与 `canWrite` 都读它，在收集箱里的四种状态同答、已接受分别断言；`issue` 宿主在收集箱里按收集箱的规则、接受之后照旧；`new_comment`、`comment`、`draft` 不变；`canRemoveAttachment` 随之（有变异）；描述有图片能力（访客提交者能传）；`ConfirmedContentNotice` 在宿主之内；m5-content 三引擎不回归。
- [ ] **A4 附件与链接**：`widgets.tsx` 只组合附件与链接、收入 `project-address-route`；附件按谓词；链接照 `issue.update`、任何状态、访客只读；没有子工作项与关联的按钮；H9 (page) ② 的读表不变。
- [ ] **A5 通用面与重定向**：浮层与侧栏在未被接受时没有状态一栏；浏览页在页头与详情挂上之前以 `replace` 送到收集箱页，页签按状态、关着照样送；收集箱的地址一处给出（三个读者）；收集箱页的“打开”只给已接受的、复制不变。
- [ ] **A6 嵌入**：纯搬移的提交既有测试一行不改；端口的提交行为不变；读的 hook 在自己的文件、预览的读关重连重读、判定锁定（重连与浮层的重试不卸下浮层或嵌入，真实 SWR 上的格）；判定按代记；非收集箱的通知只一个 `GET`（页面故事核）；`setPeekIssue` 只给浮层；失败界面是预览的、重试再判定；嵌入里分诊之后不移选中、焦点照 R35 的“选中没有移”；删除、关闭、读不到与浮层的关闭经通知页的落点，链的每一环有格有变异，焦点不落到 `body`；没有导航与 ↑↓；“打开”读 `openLink`、按状态链到收集箱页或浏览页；R32 的对应写明；`InboxContentRoot` 的导出与 `WN/root.test.tsx:58` 的替身照旧。
- [ ] **A7 Q34 与下拉框**：负责人集合在轮到时由增删算，七个调用方送增删、相对被点时显示的值；先红、变异；连点的一步先在没修的构建上红、修好之后三引擎绿；chromium 的 m4-detail 与 m4-list；两个负责人下拉框在页面上以键盘核过。
- [ ] **A8 e2e**：§2.8 的新故事三引擎各一次、chromium `--repeat-each=3`；`[s3]` 三引擎；H2–H4 (page) 的一步（等嵌入的读答完再离开项目）；既有故事不回归（检查点的六个页面配置与整套默认 e2e）。
- [ ] **A9 守卫与文档**：oxlint 等于上限、新文件零告警；knip 与第一个读者；swr 名单与 `project-address-route`；死成员整份比对；plane-diff 八行；前端改动清单；交接的处理结果（§5）；P5a 的 A9 与设计 §14 的回填（T1）。
- [ ] **A10 收尾**：终检、最终分支 CI、合并、main CI 与归档，由下一段（P6a）的 T1 补记。

## 5. 交接与后续责任

**本段处理的输入交接**：都写“处理结果（M7/P5b）”；写关闭的照写关闭，其余照设计 §11 保持 open 至收尾。T10 写。

| 交接 | 条款 | 本段 | Task |
|---|---|---|---|
| `HO/M6-closeout.md:12-19`（M6-M7-01） | 新建弹窗在编辑器建成之前拒绝关闭；可仿 C10 扣住编辑器代码块 | 关闭：§2.1 的关闭规则；H8 (page) 照 C10 的次序在页面载入之前扣住、空的收集箱、每步之前断言 `held.length > 0`（预检 I2），三引擎核 Escape 与 Discard 关空表单、提交被拒（“有未完成图片时仍拒绝”对这个不收图片的表单不适用，规则照 M4 留着、单测造出） | T2、T7 |
| `HO/M5-closeout.md:13-22`（M5-M7-01） | 收集箱的图片与附件；上传、拒绝、取消、换宿主与迟到、权限与访客、删除与清理，三浏览器、local 与 S3 | 关闭：R9 的写权限、§2.2 的图片、§2.3 的附件；H11 (page) 两份文件故事 `[local]`/`[s3]` 三引擎，范围照 R12（“接受之后”在浏览页上） | T3、T4、T8a |
| `HO/M4-closeout.md:14-29`（M7-01） | 新建弹窗的其余（P5a 写“本条在 P5b 关闭”） | 关闭 | T2、T7 |
| `HO/M4-closeout.md:62-75`（M7-04） | 待 lazy editor 就绪期间的 discard 与文档 Escape 的关闭政策；统一 dirty/pending/busy、输入保留、嵌套 Escape/IME、拒绝与换宿主迟到答（`:69`） | 收集箱部分关闭，逐项：discard 与文档 Escape 的关闭政策、dirty/pending/busy（§2.1 的表与 R28）、输入保留（取消确认框之后输入还在）、嵌套 Escape（R7）、IME（`ModalCore` 阻止，一格，R29）、拒绝（R5）、换宿主迟到答（新建弹窗的跟进限在表单还挂着时，R29）。留下、保持 open：迭代、视图的命令面板结果（`:67`）随 P6a/P6b；筛选、视图写后关闭（`:69`）随 P6a/P6b | T2 |
| `HO/M4-closeout.md:77-94`（M7-05） | `getTabIndex` 的四个收集箱文件；新页面的弹窗名、字段提示、两级 Escape | 收集箱的四个读者去掉，只剩 `views/form.tsx:75`；最终删除随 P6a；保持 open | T2 |
| `HO/M6-closeout.md:62-68`（M6-M7-05） | `getTabIndex` 与两份 `tab-indices.ts` 的最后删除 | 同上，保持 open | T2 |
| `HO/M3-closeout.md:43-44` | 弹窗的 Tab、Escape、点外面、焦点、portal；弹窗的名字（点名 `create-root.tsx:184`，今天 `:172`） | 新建弹窗一半关闭（`ModalTitle`；在途的焦点，R28；H8 (page) 三引擎核）；视图的随 P6a | T2、T7 |
| `HO/M3-closeout.md:47` | 收集箱创建表单的正的 `tabIndex` | 收集箱一半关闭（O-3）：四个文件离开 `getTabIndex`、自然 Tab 次序；函数的删除随 P6a | T2 |
| `HO/M3-closeout.md:51` | 两个负责人下拉框在页面上核对；右键菜单；焦点轮廓 | 收集箱一半关闭（详情的在 T8b，新建弹窗的在 T7）；右键菜单写明收集箱页不适用、留给 P6a；焦点轮廓交 M8 | T7、T8b |
| `HO/M3-closeout.md:62` | 标题长度提示照 `FieldError` 写（点了 `required`） | 关闭（`required` 删去） | T2、T7 |
| `HO/M3-closeout.md:63` | 新建弹窗自己的文档级 Escape | 关闭 | T2、T7 |
| `HO/M3-closeout.md:86-90`（§5，M4/P14 留下） | 统一 Escape/dirty 的关闭策略，加 lazy pending、busy、内层 Escape、输入保留的回归 | 写处理结果：§2.1 的规则与 H8 (page) 的四种回归；本 handoff 顶层保持 open | T2、T7 |
| `reviews/P4-web-notifications-review.md:148`（P4 记录 §7） | 收集箱条目以普通浮层打开，“State” 一栏是占位 | 关闭：嵌入（§2.5）与通用面隐去状态（§2.4） | T5、T6b |
| `specs/P4-web-notifications.md:478`（P4 spec §5） | 嵌入分诊；两个 prop 接回 | 关闭：§2.5，R32 的对应写明 | T6b |
| `specs/P2c-intake-assets.md:97`（P2c spec §5） | 网页按状态开放图片与附件 | 关闭 | T3、T4 |
| `reviews/P2c-intake-assets-review.md:65`（P2c 记录 §7） | 负责人多选的忙时守卫 | 关闭：按代码必然重现（预检 I5），修在写的层（R20），连点的一步先红后绿 | T8c |
| S5a `:387`（A9）、`D:351`（§14 的 P5a 一行）、R5a `:3`、`:50`、`:179` | P5a 的收尾记录 | 回填 | T1 |

**后续**：
- **P6a**：`getTabIndex`、`ETabIndices`、两份 `tab-indices.ts` 与桶文件导出的最终删除（最后的读者 `views/form.tsx:75`）；`projectMemberIds`；M6-M7-07 的视图一半；`HO/M3-closeout.md:43-44` 的视图弹窗、`:51` 的右键菜单；M7-04 的命令面板结果与筛选、视图写后关闭。
- **M7 收官报告**：R9、R20 是动了别的 M 的网页共用代码的局部裁定，与 P5a 的 R3、R4、R33、R34 同列；R26 改了 P4 通知页上浮层关闭的焦点（§7 第 8 条）；S5a R8 与 P4 R6（计数与未读点不实时）、G1–G3 照旧。
- **M8**：共用组件的焦点轮廓是否被 `overflow-hidden` 裁掉（`HO/M3-closeout.md:51`）；链接折叠栏里的加号按钮没有名字（M4 的共用件，`issue-detail-widgets/links/quick-action-button.tsx:31-33`、`links/root.tsx:37`，预检 O4）；资产的写被拒（`issue.intake_closed`、`issue.intake_disabled`）之后页面不重读条目、上传的控件留着（§2.2，预检 m7）；R5a §7 的候选照旧；本段发现的、不属本段的共用件问题。

## 6. 风险

- **高风险的局部裁定 R9：M5 的 `canWrite` 认收集箱的规则。** 图片、附件与评论的上传都经 `ContentView`，改错就坏了全应用的上传。缓解：只改 `issue` 宿主、只在 `inIntake` 时；`new_comment`、`comment`、`draft` 的格是回归；矩阵逐格与变异（含“评论宿主不受影响”“管理员在已拒绝的条目上仍可删附件”）；T3 之后 chromium 的 m5-content，每个检查点三引擎的 m5-content。
- **高风险的局部裁定 R20：M4 的负责人写。** 浮层、详情侧栏、列表行、表格、子工作项与关联的行、收集箱详情都经它。缓解：方案先给主控看；单独的修复提交先红；变异钉“轮到时算”与“调用方送增删”；请求体仍是整集（合同与既有的请求体断言不变）；chromium 的 m4-detail 与 m4-list；检查点 2 与终检跑六个配置与整套默认 e2e。
- **判定的读被再验证**（R27）：共用的键上网络恢复与浮层的重试都会再验证；锁定写错就撤下浮层或嵌入连同输入。缓解：真实 SWR 上的格（越过去重窗口之后发 `online`、浮层的重试）与两个变异。
- **嵌入的读是两整包**（预检 O1）：收集箱条目的通知从点开到看见内容约是浮层的两倍，读的数目也是两倍（判定的读 = 浮层的整包取数，嵌入再以自己的 `mutate()` 读一遍）。这是 B-16 (a) 与 P5a“每次挂载读自己的”两条规则合起来的代价，阶段记录写明；T9 断言恰两个 `GET /api/v0/issues/{id}`。
- **“只一个读”靠时间窗**（预检 O2）：主线程卡两秒以上会多一个读；页面上偶发时先看这里，不调去重间隔。
- **`[s3]` 在 Firefox、WebKit 下从未跑过**（M5 只在 chromium 跑，`P7b-content-assets-review.md:126`）：若因 M5 的上传件失败，停下报告（跨模块）。
- **点背景从此能关新建弹窗**：有改动的确认框兜住；H8 (page) 核三条路。
- **嵌套 Escape**：Headless 的下拉框在 `ModalCore` 里是否都只关自己未在这个弹窗里核实，以 H8 (page) 三引擎为准。
- **编辑器代码块扣不住**（预检 I2）：收集箱页有行时代码块已载入，扣住是空的；缓解：页面载入之前扣住、空的收集箱、每步断言 `held.length > 0`。
- **嵌入复用收集箱的详情**：动作模型、页头与落点要从收集箱页解开；先纯搬移、再端口（行为不变、既有测试只编译跟随）、再嵌入。jsdom 查不出真实的焦点问题（P5a 的 D1、D2 都是 T13 的页面故事才查出，R5a §3）：嵌入与通知页的落点以 H10 (page) 三引擎为准，键盘、指针、800 宽分别看。
- **重定向在 M4 的浏览路由里**：只加“读到之后、挂上之前”的一支；单测钉“详情从未挂上”；T5 之后 chromium 的 m4-detail。
- **故事的时序竞态**（P5a 的 I9、H6）：跳转或移出之前先等在途的读答完（登记 `waitForResponse` 在动作之前）；嵌入的故事先等判定的读与嵌入的读答完再分诊；H2–H4 等嵌入的读答完再离开项目（预检 m12）。
- **共用浮层的包含块**（P5a T10 的教训，`b1aeccb1`）：嵌入的祖先若带 `transform`，浮层会开到视口之外；嵌入的页头菜单与负责人下拉框以 `expectPopupBesideButton` 核。
- **M6-M8-07 的偶发**：H11 的图片故事会遇到 F11 的 `net::ERR_ABORTED` 资产内容读的签名；重跑该文件一次并记下，第二次失败停下。
- **CI 时长**：main CI `45197451256` 的 web 作业已 959 秒（慢 runner），本段净增 vitest，越过 960 秒几乎是确定的（预检 O6）：照 R2 的判别法只报告。e2e 加约 12–16 个 chromium 测试（含 `[s3]`），估约 +150–250 秒（估算，未测）；P5a 的最终分支 CI 两片 1331/1171 秒，离 30 分钟远。

## 7. 与 Plane 的差异（T10 登记进 `docs/v0/plane-diff.md`，收集箱一节 `:358-384`）

1. 收集箱详情有链接（任何状态、管理员与成员可写、访客只读）；Plane 的收集箱详情没有链接（`pw/core/components/inbox/content/issue-root.tsx:126-224`）（Q14、R10）。
2. 通知预览也嵌入已拒绝的条目；Plane 的 `is_inbox_issue` 只对待处理、暂缓、重复为真（`pa/app/views/notification/base.py:58-62`）；判定读工作项的 `intake`，不读通知的字段（Q13、R16）。
3. 浏览页把每个未被接受的条目送到收集箱页：待处理、暂缓到“未处理”，拒绝、重复到“已处理”，以 `replace`、在详情挂上之前；Plane 只送待处理、暂缓，一律 `currentTab=open`，以 `push`（R14）。两个后果（预检 O3）：收集箱关着时未被接受的条目经地址到不了任何详情（停用页），只剩通知的嵌入能看见；到期之前的暂缓条目被送到“未处理”页签而它默认只列待处理，条目选中了、列表里没有它那一行。
4. 新建弹窗的 Escape、点背景与 Discard 走同一条关闭规则：在途不关、焦点留在弹窗里，编辑器没建成可关，有改动先确认放弃；Plane 点背景不关、文档 Escape 不问就关，编辑器未就绪时空表单也关不掉（R4、R28）。
5. 新建弹窗的名字：按去空白之后的字符数判空白与超长，提示在字段下、提交被拒时焦点给名字，提交不停用、没有原生的 `required`；Plane 只在超长时显示红字并停用提交（`pw/core/components/inbox/modals/create-modal/create-root.tsx:238-248`、`issue-title.tsx:47-48`）（R5）。
6. 收集箱页的“打开工作项”只给已接受的条目（嵌入里一直给，未被接受的链到收集箱页）；Plane 一直给，链到浏览页（`pw/core/components/inbox/content/inbox-issue-header.tsx:375`）（R15、R30）。
7. 资产的写在网页上照 `intake.update`：访客提交者能加图片与附件，不是提交者的成员、已处理的条目、收集箱关着时没有上传的控件；补进 P2c 的一行（`:369`）的网页一半（R9）。
8. 通知页上关掉浮层或嵌入之后，焦点落在那张通知卡片的主按钮，卡片不在了落在列表的区域，再不然落在预览的区域；P4 与 Plane 嵌在通知页里关浮层都不还焦点、焦点落在 `body`（`C/components/issues/peek-overview/view.tsx:93-97`；`pw/core/components/issues/peek-overview/view.tsx:75-78`）（R26，P4 页面行为的改变，预检 O7）。

## 8. 起草时的补充裁定（2026-11-23，主控对起草者的 O-1–O-4）

- **O-1 判定的读失败时**：取乙。预览自己以 `IssuePeekOverviewError` 显示失败与重试（“关闭”经 `clearSelection`），重试再读、落定之后再判定；任何结局都只有一个读，P4 的 H2–H4 (page) 末步不变。理由：判定以这个预览自己的读为准（P4 m1），失败时不该为了显示失败再挂一个会再读的浮层。（预检 m11 之后：失败界面的“关闭”经通知页的落点，R26；“浮层会再读”的推断在 `swr@2.4.2` 下不成立，裁定的理由不靠它，§2.5。）
- **O-2 嵌入清掉选中之后的焦点**：取甲。那张通知卡片的主按钮，卡片不在了则通知列表的区域（`useLeavingFocus` 的 `region`）；只用于嵌入的删除、关闭与读不到，P4 浮层的关闭不改。理由：同 P4 卡片离开之后的落点，焦点留在用户来的那张卡片上，不落到 body（R35）。（预检 I3、O7 之后由 R26 细化：一份通知页自己的落点，列表的区域以标记找到，链末加上预览的区域，时序写明；失败界面与普通浮层的关闭也走它，主控 2026-11-24 裁定。）
- **O-3 `HO/M3-closeout.md:47` 的收集箱一半**：同意建议，T2 之后写处理结果、关闭收集箱一半，函数的删除随 P6a（§5 已照此写）。
- **O-4 H2–H4 (page) 的那一步**：取甲。与 T6b 的产品改动同一提交（允许清单），在 chromium 跑一次（`t6b-h2-h4-follow`）；T9 只加新的嵌入故事。理由：没有红的提交。
- **起草者自定的**：R24（“Create more” 就是 `Switch` 本身）与 R25 的五处细节同意；B-1 的 T3 并进 T2 同意（分开时 T2 的开关会让 Tab 次序在两个 Task 之间是乱的）；B-1 的 T9 拆为 T8a、T8b 同意。
