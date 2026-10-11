# M3/P11 状态、标签的页面与清理：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M3/P11 `web-states-labels-and-cleanup` |
| 日期 | 2026-10-11 |
| 状态 | 已完成（[评审记录](../reviews/P11-web-states-labels-and-cleanup-review.md)）：第 3 节原来待裁定的 8 条由控制者的裁定 Q1–Q8 定下（2026-10-11，第 5 条由负责人答复"菜单加入口，收尾做"），预检的发现（1 高、1 中、9 低）由修订一轮（`18b5ce41`；M3 设计的收尾一节在它之前的 `1fa8f921`）改入，各条的落点在第 3 节末"预检之后的修订"；其余是本 Phase 在设计之内做的决定，写明理由。各 Task 的修正轮和最终评审（0 严重、0 重要、8 次要）之后的修复轮照控制者的裁定 F-1–F-13 改了代码、本稿和 plan（附录 A.10 的修复轮一段）；修复的复审（0 严重、0 重要、3 次要）之后，裁定 F-14 的四处文字改在评审的提交里（2.3、第 3 节第 12 条、第 5 节收尾、附录 A.10），F-15 交给收尾（评审记录第 6 节）。执行开始时的裁定和执行中的裁定见评审记录第 3 节，修复轮的裁定 F-1–F-15 见第 3、4 节；控制者的浏览器核对 C9、C10 见第 2 节 |
| 上级文档 | [M3 设计文档](../M3-design.md) 第 1、2（P6、P7、P9、W11）、3.1、3.16、3.17、3.20、7.1、7.3、7.6、7.9、7.10、9.2、9.5、9.6、9.7（C9、C10）、12（P11）、13.1 节和决策点 3；[总体设计](../../v0-design.md) 7.7 |
| 前置交接 | [P10 spec](P10-web-project-pages.md) 第 5 节 P11 一行；[P10 review](../reviews/P10-web-project-pages-review.md) 第 6 节 P11 一行；[P9 spec](P9-web-workspace-pages.md) 第 5 节 P11 一行和第 3 节第 16 条（裁定 P8）；[P9 review](../reviews/P9-web-workspace-pages-review.md) 第 6 节 P11 一行和收尾一行；[P8b spec](P8b-web-project-data.md) 第 5 节 P11 一行、附录 A.8、A.9；[P8b review](../reviews/P8b-web-project-data-review.md) 第 6 节 P11 一行和裁定 P1；[P8a spec](P8a-web-workspace-data.md) 第 5 节 P11 一行、附录 A.8、A.9；[P8a review](../reviews/P8a-web-workspace-data-review.md) 第 6 节；[P7a spec](P7a-states.md) 第 5 节 P11 一行；[P7b spec](P7b-labels.md) 第 5 节 P11 一行；[P7b review](../reviews/P7b-labels-review.md) 第 6 节"P11（页面）"；[P4a spec](P4a-projects.md) 的 M1-P2 一行；[M1 收尾交接](../handoffs/M1-closeout.md)；[M1-P2 交接](../handoffs/M1-P2-trim-content.md)；[M2 收尾交接](../handoffs/M2-closeout.md) 的状态行；[Codex 设计评审](../reviews/M3-design-codex-adversarial-review.md) 的 M1-closeout 一行 |
| 计划 | [P11 plan](../plans/P11-web-states-labels-and-cleanup.md) |

本 spec 只写 M3 设计交给 P11 决定的东西：名字、签名、文件的位置、测试名，以及原型证明了什么。规则本身以 M3 设计和总体设计 7.7 为准，这里引用节号，不重述。P11 依赖 P8a、P8b、P9、P10（`7d6a416b` 的 `main`），不改契约、Go 代码和生成的文件。

P11 的每一处页面改动都按安全测试看待（brief）：一个角色不能发的请求在页面上是 403；换账户之后才落地的写以错误的身份写入；页面不能显示角色读不到的东西（项目的访客、不是项目管理员的项目成员、访客打开别的成员的个人主页）。所以附录 A 的变异逐个核对：状态、标签的每个修改和账户页面的每个修改之后的提示、字段错误和界面变化只在发出它的会话里进行（2.12 的每一处由它自己的 vitest 守着）；改状态、标签的控件只给 nerve 允许改的人（项目管理员和是工作区管理员的项目成员，权限的 vitest 在 store 自己的权限上，P6、P7 的页面版本以成员和工作区管理员各看一次），访客看到没有权限；访客打开个人主页只取应用和工作区的（P9 的页面版本、S2）；每个请求体只有表单编辑的字段；每个拒绝经 `errorMessageKey`（`useFormRefusal`、`useRefusalToast`），名字已被占用在名字下；请求在路上时表单忙、弹窗关不掉，被拒绝时留着；控件 Tab 能到、有名字、Tab 到达时看得见。清理的每一处按行为改动看待，直到证明它不改行为（附录 A.5、A.6）。只有评审才能发现的会话、权限或取数的性质算缺口：预检找到一处（security 页的 API 令牌的两个弹窗不经会话的核对，第 3 节第 18 条），本稿并进 Task 7；之后没有（plan 的 72 个变异和任务评审加的 26 行都被发现，附录 A.2）。没有会失败的检查的只有一处，不是会话、权限或取数的性质：删除确认框的 Tab 循环（`AlertModalCore` 的 `tabIndex={1}` 属于 M3 收尾的 `ModalCore` 重写，第 3 节第 3 条）；标签的拖动把手以状态登记由 `label-drag-n-drop-HOC.test.tsx` 看住（第 3 节第 10 条）。

## 1. 目标

按 M3 设计 12 节 P11 和第 2 节的故事：

- 状态设置页（P6，3.17，7.6）：列表按组、`sequence`；新建、改名改色、拖动改 `sequence`、设为默认、删除；默认状态和一组中唯一的状态不提供删除，服务端并发下仍拒绝时说明 409 的原因；同名的 409 落到名称下（P7a spec 第 5 节）。改状态的只有项目管理员和是工作区管理员的项目成员（他在项目里是访客也算）；项目的其他成员只能看，访客看到没有权限（W11）。
- 标签设置页（P7，3.16，7.6）：两层的拖放；新建、改名、调顺序、删除（子标签随它）；同名（不分大小写）的 409 落到名称下；放到第三层的 422（`parent_id`）说明标签只有两层；拖进空的子列表只发 `parent_id`。改标签的人同上，组的菜单和拖动把手也只给他们；拖动把手随行的渲染重新登记（P8b 的 P7）。
- 访客看到的个人主页（P9，决策点 3 的 A，9.2 的一格）：用户卡片，工作项区域"没有权限"，不请求工作项列表；管理员和成员看到的工作项列表是 M4 的（3.1）。W11 的页面版本：访客的侧边栏、成员页和项目的状态、标签设置看到没有权限，页面上没有邮箱。
- M2 的账户页面（新手引导的资料一步，个人设置的 general、preferences、security，security 页的 API 令牌的新建和撤销在内）在修改兑现之后的提示、字段错误和新令牌只在发出修改的会话里（P9 的裁定 P8；预检的 M1）；新建令牌的弹窗在请求路上时关不掉（F-1，第 3 节第 18 条）。
- 清理（7.9、7.10、7.3）：`--rows M3` 剩下的死成员和死 prop；P8a、P8b 留下的只经机械步骤到达的文件的 oxlint 警告清零；`promise(always-return)` 全仓清零；各包的上限照新的条数；关键词守卫的最终核对（M3 的例外只剩 `/user-properties/` 的两处，`until: "M4"`）；旧 `ProjectService` 只剩 7.3 的三个方法。
- 3.20 中 P11 的行：README 的"前端"一节、前端改动清单；M1 收尾交接和 M1-P2 交接的处理结果。控制者的浏览器核对 C9、C10 和评审不在 plan 中（Task 10 只改文档）。

## 2. 交付物

**文件总览**：plan 的"文件结构"一节逐个列出 Task 1–10 改到的 141 个文件和每个文件所属的 Task（任务评审的修正轮和最终评审的修复轮改到的在内，附录 A.10；另有本 spec：Task 10 和它的修正轮改了它，它与 plan 在同一个提交里，不写成块）：新文件 19 个（web 应用的 15 个：`form-refusal.ts`、`use-state-changes.ts`、`use-label-changes.ts`、测试的 `fake-form.ts` 和 11 个测试文件；e2e 的 4 个：`e2e/fixtures/dialogs.ts` 和 P6、P7、P9 的页面版本的 3 个故事文件），删除 1 个（`@nerve/types` 的 `state.ts`），修改 121 个。web 的 TS 文件 105 个（删除的一个在内）：web 应用 90 个（其中测试 17 个），`@nerve/propel` 6 个，`@nerve/ui` 5 个，`@nerve/types` 2 个，`@nerve/constants`、`@nerve/editor` 各 1 个；都是手改，Task 9 的 35 处 `return;` 由一次性的脚本照 oxlint 的输出写出，照样写成 plan 的块（附录 A.6）。web 的其余 6 个是两个 `package.json`（web 应用、editor 的上限）和两个文案文件（`project-settings.json`、`common.json`）的两种语言；e2e 24 个（fixture 5 个、故事 19 个，其中 9 个只改导入：弹窗的检查移进 `dialogs.ts`）；根目录 `.oxlintrc.json`（2.13 的范围）和 `tools/keywords.json`（2.13 的规则）；文档 4 个（`README.md`、`docs/v0/frontend-changes.md`、M1 收尾交接、M1-P2 交接）。契约、Go 代码和生成的文件都不变。

**依赖**：不加新的 npm 包，`pnpm-lock.yaml` 不变。

**共用的新部分**：
- `core/lib/form-refusal.ts`（Task 1）：`FormRefusal<F>`（`{ kind: "fields"; fields } | { kind: "toast" }`）、`formRefusal(error, fields, taken)` 和 `useFormRefusal(fields, taken)`：表单的拒绝落到它的字段下（已被占用的码落到 `taken` 指的字段下，不论问题指名哪些字段；只指名表单字段的字段错误落到各自的字段下，`needsErrorBanner`）或提示（`useRefusalToast`）。P10 的 `projectRefusal`、`useProjectRefusal` 改为以它写成（项目的两个字段、两个码），显示的照旧，P10 的 `project-refusal.test.ts` 不变、照样通过（它经 `projectRefusal` 钉住决定的顺序，`T1.1`）；状态（Task 2）、标签（Task 4）的表单也经它。
- `@nerve/ui` 的 `AlertModalCore`（Task 1）：`isSubmitting` 时取消禁用，交给 `ModalCore` 的 `handleClose` 为空（Escape、点外面都不关；P9 的 F-1，总体设计 7.7）。它的其余 11 个调用方同样（第 3 节第 2 条）。
- `useStateChanges`（Task 2）、`useLabelChanges`（Task 4）：页面的每个修改经 store 的方法、经 `followInSession`；拒绝经 `useFormRefusal`（新建、修改）或 `useRefusalToast`（其余）；每个都在页面跟进之后才兑现，所以调用方 `await` 它就是"忙到跟进做完"。
- store 的两个 getter 以项目的 id 读（Task 3、5）：`getGroupedProjectStates(projectId)`、`getProjectLabelsTree(projectId)`（`computedFn`），代替读 router store 的项目的 `groupedProjectStates`、`projectLabelsTree`（第 3 节第 16 条）。
- 共用组件的焦点和名字（Task 5，任务评审的裁定 T4-a、T5-a、T5-b、T5-e、T5-f；W17 在第 3 节第 12 条）：propel 的 `Button`、`IconButton` 的基础类去掉 `focus-visible:outline-none`、加 `focus-visible:outline-offset-2`（键盘取得焦点时有浏览器的轮廓，离按钮 2 像素，实心的按钮也看得见；`getButtonStyling`、`getIconButtonStyling` 的调用方同样）；`@nerve/ui` 的 `CustomMenu` 的省略号按钮去掉 `outline-none`；`@nerve/ui` 的 `Popover` 收 `buttonAriaLabel`，交给它的按钮。两个表单的颜色按钮名为"Color"（新文案 `common.color`，两种语言），项目设置侧边栏的返回箭头名为"Go back"（已有的 `common.go_back`）。
- 测试的共用部分：`core/lib/fake-controls.ts` 加 `AlertModalCore` 的替身（`shown.alerts`：是否打开、确认的按下、是否忙；Task 2）；`core/lib/fake-form.ts`（Task 7，新文件）：react-hook-form 的替身 `fakeForm(form)` 和它留下的东西的类型 `KeptForm<V>`，新手引导、general、security 的三个测试都经它（裁定 T7-b；三个测试的 `vi.hoisted` 对象以 `KeptForm` 为类型，不重述它的形状，修复轮的 F-7）；其余用 P8b–P10 的（`fake-tab.ts` 的 `lateSettlings`、`lateSettlingsAnswering`，`fake-refusal.ts`、`fake-toast.ts`、`fake-i18n.ts`、`fake-store-hooks.ts`、`fake-permissions.ts` 的 `callerInWeb`、`its`），不另写一份。
- e2e：P8 故事的 `dragOnto` 移进 `e2e/fixtures/project-pages.ts`（照按住把手、在行内移动、移到目标的点上、放开，先等过渡结束），P6、P7、P8 的故事都经它；`tabCycle(page, within, presses = 40)` 收按 Tab 的次数（行内的表单不是弹窗，Tab 走到表单之后就离开它；Task 3）。`e2e/fixtures/dialogs.ts`（Task 7，新文件；修复轮的 F-4）：P9、P10 写在 `settings-pages.ts` 里的弹窗的检查和"一秒之内"的检查（`enabledWithin`、`shownWithin`、`hiddenWithin`、`closedWithin`、`closedByEscape`、`composingEscape`、`closedByComposingEscape`）移来，导入它们的故事随之改；`closedByBackdropClick`（点弹窗外面，`closedByEscape` 的孪生，裁定 T7-a）；一个不导出的 `happenedWithin(page, event, matches, act)`：在 `act` 之前就听页面的 `request` 或 `download`，`act` 结束之后再等一秒，事件先到时结束计时器，之后不再听；`sentWithin`（请求，Task 3：项目成员的拖动不发请求）和 `downloadedWithin`（下载，Task 7：迟到的回答不下载）是它的两个名字。`e2e/fixtures/assert/identity.ts` 的 `expectNoAddress(page, emails, where)`（Task 6，修复轮的 F-5）：页面此刻的文档里没有这些邮箱，P9、W11 的页面版本都经它。`e2e/fixtures/auth.ts` 的 `named` 收要设的名字（`UserUpdate` 的 `first_name`、`last_name`、`display_name`；Task 6，裁定 T6-a）：P9 设名字和显示名，P1 照旧只设显示名。P9、P10 的 `sentTo`、`sentHeld`、`watchPage`、`expectQuietConsole`、`anotherBrowser` 照用。

### 2.1 表单的拒绝；确认框在请求路上时关不掉（Task 1；总体设计 7.7 的 F-1；P10 的 `projectRefusal`）

- `form-refusal.ts`、`project-refusal.ts`（上面的共用部分）。`formRefusal` 决定的顺序：已被占用的码（`taken[code]`）先于字段错误，字段错误只指名表单的字段时落到字段下，其余提示。已被占用的码的文案也经 `errorMessageKey`（读错误的唯一的函数，没有文案的码说 `errors.unknown`，不会无声地丢掉；Task 1 评审的 m1，裁定 T1-b）。
- `AlertModalCore`：`handleClose` 的说明写明取消、Escape、点外面在 `isSubmitting` 时都不关；`ModalCore` 收 `isSubmitting ? undefined : handleClose`；取消 `disabled={isSubmitting}`。确认按钮的 `tabIndex={1}` 不动（第 3 节第 3 条）。
- 两个工作项的删除确认框（`issues/delete-issue-modal.tsx`、`issues/workspace-draft/delete-modal.tsx`，M4 的页面）的 `onSubmit` 改为必传（裁定 Q2 的加固，附录 A.4）：不传时确认框一直忙，有了上一条就再也关不掉；今天 11 个调用处都传，`tsc` 核对。两个文件各有的一条 `always-return` 在 Task 9 清零，P11 结束时 0 条警告。
- 测试：没有新的 vitest（`project-refusal.test.ts` 经 `projectRefusal` 照旧核对整个决定；`AlertModalCore` 的关不掉在服务端渲染中看不到）。端到端：P6 的页面版本（Task 3 起）扣住状态的删除：Escape 不关、取消禁用（`T1.2`、`T1.3`）；P7 的页面版本（Task 5 起）扣住标签的删除。
- 静态检查：`form-refusal.ts` 进非空断言的范围（2.13）。

### 2.2 状态的修改（Task 2；P6，3.17，7.6；总体设计 7.7；P8b spec 附录 A.8、A.9）

- `core/components/project-states/use-state-changes.ts`：`useStateChanges()` 给出 `create(projectId, data: StateCreate, underField, done)`、`update(stateId, data: StateUpdate, underField, done)`、`move(stateId, group, droppedOnId, after)`、`remove(stateId, done)`、`markDefault(stateId)`；新建、修改的拒绝经 `useFormRefusal`（`name`、`description`；`project.state_name_taken` 落到名称下），其余经 `useRefusalToast`；`StateUnderField` 是表单的 `setError`。
- 表单 `create-update/form.tsx`（整个文件）：react-hook-form（`TStateFormData = Omit<StateCreate, "group">`），`onSubmit(formData, underField)` 交回的 `Promise` 兑现之前忙（`isSubmitting`：按钮显示"Creating"/"Updating"、加载，取消禁用），字段的错误在字段下（`errors.name`、`errors.description`）；原来的 `loader` 从不置真（连按两下发两次），被拒绝时一律提示（名称已被占用也是），表单自己的 `catch` 只写 `console.log`（接不到拒绝：`create.tsx`、`update.tsx` 已在自己的 `catch` 中提示）。`create.tsx`、`update.tsx`（整个文件）经 `useStateChanges`，`done` 关上表单；请求体是表单的字段（新建另加组，修改只发 `StateUpdate` 的字段）。
- 删除 `options/delete.tsx`（整个文件）：`remove(state.id, () => setIsDeleteModal(false))`，确认框忙到跟进做完，只在 nerve 删除之后、只在发出它的会话里关上，被拒绝时留着（P10 的 A-M2），原因在提示里；默认状态、一组中唯一的状态不提供删除（`isDeleteDisabled`，原来照旧）。设为默认 `options/mark-as-default.tsx`（整个文件）、拖动 `state-item.tsx` 经 `markDefault`、`move`。
- 组件不再收回调：`TStateOperationsCallbacks`（`@nerve/types` 的 `state.ts`，只有这些组件读它）删除；`root.tsx`、`group-list.tsx` 把 `projectId` 传到 `group-item.tsx`（新建状态的表单要它），没有调用方传的 `groupListClassName`、`groupItemClassName`、`stateItemClassName`（P8b spec 附录 A.9）和 `state-item-title.tsx` 的 `shouldShowDescription` 删除；一行以状态的 id 为 key（原来是名称：改名之后整行重新挂载）。
- 组标题（`group-item.tsx`）是按钮（`type="button"`、`aria-expanded`），里面的 `div` 换成 `span`：Tab 能到，Enter、空格展开、收起（原来是可点击的 `div`；它的两条 oxlint 警告随之没有了，P8b 附录 A.8 中这个文件的两条，web 的上限 348 → 346）。加状态的按钮在它旁边，不在它里面（附录 A.5 的 C-a）。
- 测试：`use-state-changes.test.ts`（新文件，32 个：五个修改各发给 store 的、做完时跟进 `done` 不提示、新建和修改的拒绝在字段下或提示、其余的拒绝提示且不跟进、换账户之后兑现的两种）；`create-update.test.tsx`（3 → 6 个：新建发页面的项目和表单的组；修改只发 `StateUpdate` 的字段；做完时说、关上表单，忙到那时；名称的拒绝在名称下、表单留着）；`options/delete.test.tsx`（新文件，4 个：默认状态和组中唯一的状态不提供删除、不发；确认之后才删，之前确认框关着；拒绝说 nerve 的原因）。`fake-controls.ts` 加 `AlertModalCore` 的替身。
- 静态检查：`use-state-changes.ts`、它的测试和 `options/delete.test.tsx` 进非空断言的范围。

### 2.3 状态设置页；P6、W11 的页面版本（Task 3；P6、W11，7.6、9.2；P10 的 F-2、F-12；P7a spec 第 5 节）

- 页面 `settings/projects/[projectId]/states/page.tsx`：地址只从路由参数读：标题的项目经 `getProjectById(projectId)`（不读 `currentProjectDetails`），`allowPermissions([ADMIN, MEMBER], PROJECT, workspaceSlug, projectId)`（项目的成员看到状态，访客看到没有权限）。根组件 `root.tsx` 的项目的状态经 `getGroupedProjectStates(projectId)`，它自己不取：原来的 `useSessionSWR(["PROJECT_STATES", projectId])` 与项目的包装层取的是同一个，删除（修复轮的 F-3）。所以在一个项目的设置页之间移动、进状态页时不再重取状态：项目的包装层一直挂载，持有那一个取数，标签页原来就是这样（修复的复审的 m1，裁定 F-14）；一组的状态交给它的行，整个分组（`groupedStates`）不再从组列表一层层传到每一行（在 `group-item.tsx` 之下没有读者，修复轮的 F-1）；改状态的控件给 `allowPermissions([ADMIN], PROJECT, workspaceSlug, projectId)` 的人（项目管理员和是工作区管理员的项目成员，他在项目里是访客也算：nerve 的 `state.create` 等，P7a）。关键词规则 `project-address-route` 的文件加上页面和根组件（2.13）。
- 一行的控件（`state-item-title.tsx`）：原来 `hidden group-hover:flex`，指针经过之前不在 Tab 顺序里；现在一直在，透明（`opacity-0`），行里有焦点或指针经过时显示（`group-focus-within:opacity-100`）；编辑按钮名为"Edit"（`common.actions.edit`），删除按钮名为"Delete"，组的加状态按钮名为"Add"（`common.add`）；删除按钮去掉 Plane 的 `focus:outline-none`，Tab 到达时有焦点的轮廓（Task 3 评审的 I1）。
- 测试：`states-labels.test.tsx`（新文件，Task 3 起 5 个、Task 5 起 11 个；服务端渲染页面的路由，`createRoutesStub` 给出参数，router store 的地址不指任何项目；调用者的权限和项目是 store 自己的，`fake-permissions.ts`；状态和标签的 store 的替身只对页面的项目回答，状态的替身是 web 的两组状态，组列表的替身写出它们：读了别的 id 的页面显示加载，测试失败，`T3.10`，预检的 L5）：项目管理员、是工作区管理员的项目成员（他在项目里是成员或访客，`its.memberAdmin`、`its.guestAdmin`）看到状态和控件；项目成员看到状态、没有控件；访客看到没有权限。`state.store.test.ts` 读 `getGroupedProjectStates(web.id)`。
- 端到端：P6 的页面版本（新文件 `p6-states-page.spec.ts`）：管理员在 started 组新建 Review，表单的 Tab 顺序是名称、说明、Create、Cancel，请求体只有表单的字段和组，`sentHeld` 扣住时 Create（显示"Creating"）和 Cancel 都不可用，nerve 建好之后表单关上，`sequence` 是 70000；unstarted 组再建 Todo：409 在名称下，表单留着、值不变；Review 的 Edit 由 Mark as default 按 Tab 到达，到达之前行的控件透明、到达之后显示，改色只发 `name`、`color`、`description`；Review 拖到 In Progress 之前（`dragOnto`）只发 `group`、`sequence`；Review 设为默认，Backlog 不再是；Review、Backlog（默认）、Todo（组中唯一的）的删除不可用；另一处把 In Progress 设为默认之后删它：409 的原因在提示里，确认框留着，取消关上；重新载入之后再删，`sentHeld` 扣住时 Escape 不关、取消禁用，nerve 删除之后关上；数据库的每一步（`expectStates`）；项目成员看到状态、没有控件，是工作区管理员的项目成员看到全部控件。任务评审之后另核对（裁定 T2-a、T2-b，Task 3 评审的 I1、I2，裁定 T4-a）：组标题收起、展开（`aria-expanded`，`T2.16`）；Review 第一次设默认被扣住时按钮忙、显示"Marking as default"（`T2.14`、`T2.15`）；Edit、Delete 由 Tab、Shift+Tab 到达时有焦点的轮廓（`T3.11`）；状态表单的颜色按钮按名字"Color"找到（`T4.14b`）；项目成员把 In Progress 拖到 Review 上：一秒之内不发请求，顺序不变（`sentWithin`，`T3.12`）。W11 的页面版本（`w11-guest-bounds.spec.ts` 加一个）：访客登录，侧边栏只列他是访客的 Web、不列私密的 Secret；工作区成员页、Web 的状态设置都说没有权限；他打开的页面上没有任何邮箱。S2 的四个账户各打开状态设置（2.11）。P8 的拖动经 `dragOnto`（`dragInSidebar`）。

### 2.4 标签的修改（Task 4；P7，3.16，7.6；P7b review 第 6 节；P10 的 T11-a）

- `core/components/labels/use-label-changes.ts`：`useLabelChanges()` 给出 `create(projectId, data: LabelCreate, underField, done)`、`update(labelId, data: LabelUpdate, underField, done)`、`move(labelId, parentId, droppedOnId, dropAtEndOfList)`、`removeFromGroup(labelId)`、`remove(labelId, done)`。新建、修改的拒绝经 `useFormRefusal`（`name`；`project.label_name_taken`，不分大小写，落到名称下，不再是提示：M3 设计 7.6）；移动的拒绝指名 `parent_id`（422 `not_allowed`，nerve 的四种：父标签不存在或在别的项目、自己做自己的父标签、父标签已在别的标签之下、下面有标签的标签放到别的标签之下，3.16）时说"标签只能放在本项目的顶层标签之下，下面有标签的标签留在顶层：标签只有两层"（新文案 `project_settings.labels.parent_not_allowed`，两种语言；第 3 节第 4 条），其余经 `useRefusalToast`；移出组只发 `parent_id: null`，nerve 不因 `parent_id` 拒绝它，它和删除的拒绝都经 `useRefusalToast`。
- 行内表单 `create-update-label-inline.tsx`（整个文件）：提交经 `create`/`update`，`underField` 是 react-hook-form 的 `setError`，`done` 关上表单；忙到跟进做完（`isSubmitting`：按钮"Adding"/"Updating"，取消禁用），名称下显示 nerve 的原因。新建还是修改、按钮说什么，都只由 `labelToUpdate` 决定，`isUpdating` prop 删除（预检的 L8：顶部的表单在编辑别的标签时原来显示"Update"；列表和组照旧以自己的 `isUpdating` 排版）。修改从 nerve 存的颜色开始：没有颜色的标签改名时颜色照旧是空的，不再发出 `#000`（Task 4 评审的 M1，裁定 T4-c，改在 Task 5；表单中它的色块照列表画成黑色）。Task 5 再改这个文件（2.5）。删除 `delete-label-modal.tsx`（整个文件）：确认框忙到跟进做完，只在 nerve 删除之后、只在发出它的会话里关上，被拒绝时留着（原来请求在路上时关得掉，回答之后不经会话的核对就关上）。`project-setting-label-item.tsx`、`project-setting-label-group.tsx`、`project-setting-label-list.tsx` 经 `useLabelChanges`，不再收回调（`TLabelOperationsCallbacks` 删除）。
- 测试：`use-label-changes.test.ts`（新文件，26 个：五个修改各发给 store 的、做完时跟进、名称的拒绝在名称下、名字太长在名称下、颜色的拒绝提示、移动到 nerve 不允许的地方说两层、其余的拒绝提示且不跟进、换账户之后兑现的两种）；`create-update-label-inline.test.tsx`（3 → 5 个：新建发页面的项目、在顶层，按钮是 Add；修改只发 `LabelUpdate` 的字段，按钮是 Update；nerve 做完才关；被拒绝时留着，名称已被占用在名称下、不提示：表单是 react-hook-form 自己的，它的 `setError` 由测试一并记下，预检的 L6）。
- 静态检查：`use-label-changes.ts` 和它的测试进非空断言的范围。

### 2.5 标签设置页；P7 的页面版本（Task 5；P7、W11，3.16、7.6、9.2；P8b 的 P7；P10 的 F-2、F-12）

- 页面 `labels/page.tsx`：路由参数，标题的项目经 `getProjectById(projectId)`，`allowPermissions` 写明工作区和项目（项目的成员看到标签，访客看到没有权限），把工作区和项目交给列表。列表 `project-setting-label-list.tsx` 收 `workspaceSlug`、`projectId`（不再 `useParams`），标签经 `getProjectLabels(projectId)`、`getProjectLabelsTree(projectId)`，改标签的人是 `allowPermissions([ADMIN], PROJECT, workspaceSlug, projectId)`；没有标签时的"添加标签"按钮只给他们（原来给每个成员）。关键词规则加上页面、列表和拖放组件（2.13）。
- 拖动（`label-drag-n-drop-HOC.tsx`）：拖动由把手开始，把手只在改标签的人的行上渲染（行被编辑时也不渲染）；组件把行此刻渲染的把手存在状态里（回调 ref `setDragHandle`），把手换了（编辑关上之后行重新渲染）就重新登记。原来经 ref 读把手、`useEffect` 的依赖里没有它，只因列表每次渲染重建 `onDrop` 才重新登记（P8b 的 P7，`project-setting-label-list.tsx:62`、`project-setting-label-item.tsx:77-78`、`:105-106`；第 3 节第 10 条）。组件自己的 `allowPermissions`（不传工作区和项目）删除：谁能拖由列表决定。`@nerve/ui` 的 `DragHandle` 只给指针用：`tabIndex={-1}`、`aria-hidden`（第 3 节第 19 条，它的另 3 个调用方同样；侧边栏项目的行和收藏夹的外层手柄按钮照旧是显示时的停留点，第 5 节收尾）。没有把手的行（正在编辑的标签的）不登记拖动，也不是放下的目标（组件在没有把手时直接返回）。
- 组的行（`project-setting-label-group.tsx`）给 `LabelItemBlock` 传 `disabled={!isEditable}`：组的菜单和把手只给改标签的人（原来给每个成员，nerve 拒绝他们的修改）。一行（`label-item-block.tsx`）：没有调用方传的 `draggable` 删除；菜单名为"Options"（`common.options`），删除按钮名为"Delete"；控件在行里有焦点时也显示（`group-focus-within:opacity-100`）。组的展开按钮（`Disclosure.Button`）以标签的名字为名。
- 行内表单（`create-update-label-inline.tsx`，整个文件）：颜色按钮名为"Color"（`common.color`），去掉 Plane 的 `focus:outline-none`（Task 4 评审的 I1，裁定 T4-a）；修改从 nerve 存的颜色开始（裁定 T4-c，2.4）。Plane 的显示和隐藏的接线删除：`labelForm`（调用方只在它为真时渲染表单）、`setLabelForm`（调用方的 `onClose` 再关一次）、`forwardRef` 和列表的 `scrollToRef`（没有读者）；`onClose` 必传，是表单唯一的关法（取消，或 nerve 做完新建、修改之后），新建的颜色是 `getRandomLabelColor()`（最终评审的 M2，修复轮的 F-2）。状态表单的颜色按钮同样名为"Color"（`Popover` 的 `buttonAriaLabel`）。
- 共用组件的焦点（上面的共用部分）：标签一行的菜单（`CustomMenu` 的省略号按钮）、表单的"Add"（propel 的 `Button`）、项目设置侧边栏的返回箭头（propel 的 `IconButton`，名为"Go back"）Tab 到达时有焦点的轮廓，`Button`、`IconButton` 的离按钮 2 像素。
- 测试：`states-labels.test.tsx` 加 6 个（项目管理员、是工作区管理员的项目成员（成员或访客）看到标签、控件和按钮；项目成员看到标签、没有控件；访客看到没有权限；没有标签时只有能加的人看到空状态的按钮）；`project-setting-label-group.test.tsx`（新文件，2 个：能改的人看到组的控件和组下标签的控件；其他成员看到组和组下的标签，没有控件）；`label-drag-n-drop-HOC.test.tsx`（新文件，2 个：以行此刻渲染的把手登记拖动，行换了把手时以新的登记；行没有把手时不登记；测试自己照 React 的语义写三个 hook，第 3 节第 10 条）；`label.store.test.ts`、`label.store.changes.test.ts` 读 `getProjectLabelsTree(web.id)`；`label.store.test.ts` 的"各项目的标签分开"一条再读 `ops` 的树，`state.store.test.ts` 的同一条再读 `ops` 的按组分的状态：读路由 store 的项目的 getter 让它们失败（Task 5 评审的 I1，`T5.19`、`T3.13`）；`create-update-label-inline.test.tsx` 加 1 个（5 → 6）：没有颜色的标签改名，请求体的颜色是空的（`T4.16`），表单收调用方的 `onClose`，测试看它是否被调用（修复轮的 F-2）。
- 端到端：P7 的页面版本（新文件 `p7-labels-page.spec.ts`）：管理员加 Bug：表单打开时焦点在名称上（它的副作用在表单显示之后才可能运行，故事先等名称有焦点），Tab 顺序是 `labelName`、Cancel、Add，`sentHeld` 扣住时 Add（显示"Adding"）和 Cancel 都不可用，请求体只有名称和颜色；再加 Feature；加 `bug`：409 在表单下面一行的名称下；UI 拖到 Bug 的中间只发 `parent_id`；从组的菜单把 Bug 改名 Defect，请求体是名称和颜色，组的展开按钮名为 Defect、展开着；Feature 拖到 Docs 的上方：`{ parent_id: null, sort_order: 65535 }`；另一处把 Feature 放进 Defect 之后，页面把 Docs 拖到 Feature 上：422，提示说标签只有两层；点别处之后 Docs 的菜单和删除按钮透明，由 Feature 的删除按钮按 Tab 直接到 Docs 的菜单（经过把手而不停），再到删除按钮，这时显示；项目成员看不到"Add label"、把手、菜单、删除按钮，也没有可拖的标签（`[draggable="true"]` 0 个），是工作区管理员的项目成员各有四个（删除按钮三个：组没有）；重新载入之后从菜单删除 Defect：`sentHeld` 扣住时 Escape 不关、Cancel 不可用，nerve 删除之后关上，它下面的标签一起不见；`expectLabels` 核对每一行。任务评审之后另核对（裁定 T4-a、T4-b、T4-d、T5-b、T5-e、T5-f）：侧边栏的返回箭头由 Shift+Tab 按名字"Go back"到达，有离开它的焦点轮廓（`T5.21`、`T5.23`、`T5.24`）；颜色按钮按名字"Color"找到，Tab 到达时有焦点的轮廓（`T4.13`、`T4.14`）；"Add"的焦点轮廓离开实心的按钮（`T5.20`、`T5.22`）；一行的菜单 Tab 到达时有轮廓（`T5.18`）；另一处删掉 Docs 之后在这一页删除它：nerve 拒绝（404），提示，确认框留着，取消关上（`T4.15`）；重新载入之后从 UI 的菜单把它移出 Defect 的组：只发 `{ parent_id: null }`（`T4.17`）。P6 的页面版本按名字"Color"找状态表单的颜色按钮（`T4.14b`）。W11 的页面版本加上标签设置；S2 的四个账户各打开标签设置。
- 静态检查：`project-setting-label-group.test.tsx`、`label-drag-n-drop-HOC.test.tsx` 进非空断言的范围。

### 2.6 访客看到的个人主页；P9 的页面版本（Task 6；P9，决策点 3 的 A，3.1，9.2；M1-P2 交接）

- web 的代码不变：决策点 3 的 A 是 Plane 的页面原样（`profile/[userId]/layout.tsx` 的 `isAuthorized` 只许工作区管理员和成员看工作项部分，访客看到"没有权限"、不挂载 `Outlet`，不请求工作项）。每个角色挂载和取的由设计定下（3.1："访客看到的个人主页（决策点 3，卡片之外不挂载工作项列表）"；"成员和管理员看到的个人主页的工作项列表（M4）"，点进去之后旧接口 404 是已知的过渡），P11 只核对（附录 A.3；第 3 节第 6 条）。
- 端到端：P9 的页面版本（新文件 `p9-guest-profile.spec.ts`）：Ada（管理员）和 Gus（访客）；Gus 打开 Ada 的个人主页和他自己的：地址都转到 `/assigned`，卡片有名字和显示名（数据来自工作区的成员列表，P9 的 `useProfileMember`），工作项区域说"You do not have the permission to access this page."，没有 Assigned、Created、Subscribed 的链接，页面里没有任何人的邮箱（`expectNoAddress`，W11 的页面版本同样经它，修复轮的 F-5）；两人的名字和显示名由 `named` 设（裁定 T6-a）；没有一个请求的地址含 `issues`，没有失败的请求、没有 Plane 的旧地址、没有页面错误。S2 的访客加两页：`/{slug}/profile/{admin}`、`/{slug}/profile/{guest}`，都是 `APP`、`WORKSPACE`（2.11）。

### 2.7 M2 的账户页面在会话里跟进（Task 7；P9 的裁定 P8，P9 spec 第 5 节；总体设计 7.7；M2 设计 7.3）

- 五个文件的修改经 `followInSession`：`done` 里提示成功（或交给下一步、清空表单），`failed` 里显示拒绝；说什么、在哪里说照 M2（第 3 节第 7 条）：
  - 新手引导的资料一步（`onboarding/steps/profile/root.tsx`）：名字被拒绝时 `first_name` 的字段错误在名字下，其余提示（`needsErrorBanner`）；交给下一步的 `onDone` 是 P9 的，照旧。
  - general（`settings/profile/content/pages/general/form.tsx`）：成功提示"profile_updated"；被拒绝时 nerve 指名的字段错误在字段下，其余提示；按钮的加载在跟进之后结束。
  - preferences 的时区和语言（`language-and-timezone-list.tsx`）、每周第一天（`profile/start-of-week-preference.tsx`）：成功提示，被拒绝时提示。
  - security（`security.tsx`）：密码改了之后清空表单、隐藏密码、提示；`current_password_incorrect` 在旧密码下，`new_password` 的字段错误在新密码下，其余提示。
  - security 页的 API 令牌（`api-token/modal/create-token-modal.tsx`、`api-token/delete-token-modal.tsx`，预检的 M1）：新建在 `done` 里显示新令牌、下载它的 CSV（只在发出它的那一次打开里，`phase` 照旧），`failed` 里表单照 M2 显示拒绝（表单的 `onSubmit` 把显示拒绝的函数交给弹窗）；新建在请求路上时取消禁用（F-1），表单忙到跟进做完（第 3 节第 18 条）。撤销在 `done` 里提示、关上，`failed` 里提示原因，确认框忙到跟进做完。
- 另一个标签页换了账户之后才兑现的修改，页面什么都不做（不提示、不交给下一步、不清空表单、不在字段下写东西、不显示也不下载新令牌）。
- 测试：`onboarding/steps/profile/root.test.tsx`（4 → 6 个，原来的两个换成名字的拒绝在名字下、不是名字的拒绝提示、换账户之后兑现的两种）；`general/form.test.tsx`（新文件，5 个：发名字、地址不在其中，说 nerve 改了；字段的拒绝在字段下、不提示；不指名表单字段的拒绝提示；换账户之后兑现的两种）；`security.test.tsx`（新文件，6 个：发两个密码，nerve 改了之后清空、说；两种字段的拒绝各在它的字段下，表单照旧；两者都不是的拒绝提示；换账户之后兑现的两种）；`preferences/language-and-timezone-list.test.tsx`（新文件，12 个：时区、语言、每周第一天各一组：发选的、说 nerve 改了；拒绝提示；换账户之后兑现的两种）；`api-token/modal/create-token-modal.test.tsx`（新文件，4 个：发表单给的，nerve 建好之后下载密钥；拒绝交给表单、不下载；换账户之后兑现的两种）；`api-token/delete-token-modal.test.tsx`（新文件，4 个：确认之后撤销，nerve 做完之后提示、关上；拒绝提示原因、不关；换账户之后兑现的两种）。新手引导、general、security 的三个测试的 react-hook-form 替身是 `core/lib/fake-form.ts` 的（裁定 T7-b，修复轮的 F-7）。
- 端到端：没有新的故事。A11（裁定 T7-a，`T7.10`–`T7.12`）：第一个故事的撤销被扣住（`sentHeld`）时确认框关不掉、删除按不了；第二个故事核对新建的请求被扣住时"Generating"和取消都按不了，Escape 和点外面都不关（`closedByEscape`、`closedByBackdropClick`），再经浏览器的后退离开页面（原来按取消，第 3 节第 18 条），回答到达之后的一秒之内不下载（`downloadedWithin`，Task 7 评审的 m1）。M2 的其余账户故事（A 系列）和 P9 的新手引导照旧通过；P1–P8、W1、W3、W4、W7、A12 只改导入（`dialogs.ts`）。
- 静态检查：五个新的测试文件和 `fake-form.ts` 进非空断言的范围（新手引导的测试已在里面）。

### 2.8 死成员和死 prop；P8a、P8b 留下的文件（Task 8；7.9；M1 收尾交接；P8a、P8b spec 附录 A.8、A.9；P8b 的裁定 P1）

- 删掉的都没有读者、没有调用方传入（附录 A.5 的 C-b）；没有调用方传的 prop 原来取默认值，组件现在把默认值写成常量，显示的照旧：
  - 项目下拉框（`dropdowns/project/dropdown.tsx`、`base.tsx`）：`button`、`currentProjectId`、`dropdownArrow`、`dropdownArrowClassName`、`onClose`、`renderByDefault`；`getProjectById` 的类型是生成的 `Project`（P8b 的裁定 P1）；`base.tsx` 的五条 oxlint 警告清零（`no-shadow` 三条：取显示名和图标的函数不再收参数；`consistent-function-scoping`：画图标的函数移到模块；`ComboDropDown` 照 `@nerve/ui` 的说明收 `role="presentation"`，根可以聚焦的地方浏览器不理会它，第 3 节第 12 条）。
  - 成员头像（`dropdowns/member/avatar.tsx`）的 `size`；状态选项（`dropdowns/state/state-option.tsx`）的 `query`；命令面板的成员菜单（`power-k/menus/members.tsx`）的 `heading`；创建项目的公共字段（`project/create/common-attributes.tsx`）的 `handleFormOnChange`；邀请弹窗的按钮行（`workspace/invite-modal/actions.tsx`）的五个；状态图标的虚线圆（`@nerve/propel` 的 `dashed-circle.tsx`）的 `totalSegments` 和 `radius`（三个调用方都传，它从不读；修复轮的 F-6，`backlog-group-icon.tsx` 的 `radius` 常量随之删除，另两个图标的照旧由进度圆读）；归档页头（`archives/header.tsx`）每个标签的 `href`（和图标类型中与 `SVGAttributes` 重复的 `className`）。
  - 工作区成员 store 的 `memberMap`（接口、`computed`、getter）；`@nerve/constants` 的 `members.ts` 的 `iconName`（五处）。
- P8a、P8b 留下的只经机械步骤到达的文件（附录 A.5）：`use-work-item-filters-config.tsx` 的三条 `no-shadow`；`progress-stats/state_group.tsx` 的 `no-array-index-key`（`key={group.state}`）；`dropdowns/project/base.tsx` 的五条（上一条）；`project-states/group-item.tsx` 的两条（Task 2）；`create-webhook-modal.tsx`、`work-item/commands.ts` 的 `always-return`（Task 9）。web 的上限 346 → 337。
- 测试：没有新的（删除的代码没有读者，`tsc`、knip；组件按原来的默认值显示，附录 A.5 逐个说明）；oxlint 的改法由上限守着（`T8.1`、`T8.2`）。

### 2.9 `promise(always-return)` 全仓清零（Task 9；7.9；M1 收尾交接；P8a spec 第 5 节）

- oxlint 指出的 35 个 `then()` 回调（web 34 个、在 25 个文件中，editor 1 个）各在最后一个语句之后加 `return;`。回调原来的完成值是 `undefined`，加了之后仍是：`then()` 给出的 promise 以同样的值、在同样的时刻兑现，拒绝的路径一处都不变（附录 A.6 逐处列出它的拒绝由谁处理）。
- 上限：web 337 → 303，editor 65 → 64；`promise(always-return)` 全仓 0 条。这些文件的其余警告留着（第 3 节第 8 条）。
- 根目录 `.oxlintrc.json` 的 `no-unreachable` 设为 error（预检的 L9）：一处 `return;` 写在回调最后一个语句之前，那个语句就走不到，这条规则让 `check:lint` 失败（`T9.3`）；今天全仓 0 处，上限不变。

### 2.10 文档；关键词守卫和旧 `ProjectService` 的最终核对（Task 10；3.20、13.1 中 P11 的行；7.3、7.10）

- `README.md` 的"前端"一节：状态、标签的设置页（谁能改、谁只能看、访客看到没有权限）和个人主页（访客看到卡片和没有权限；工作项一栏由 M4 对接）。
- `docs/v0/frontend-changes.md`：3.1 中 M3 一行的状态；3.2 新增六行，都是"已完成，M3/P11"：状态和标签的设置页（第 3 节第 12 条的每一项，拖动把手和组的展开按钮在内）、确认框（`AlertModalCore`，它的其余 11 个调用方同样）、M2 的账户页面跟进修改的结果（API 令牌的新建和撤销在内，新建在请求路上时取消按不了）、死成员和死 prop、`then()` 的回调、共用组件的键盘焦点（propel 的 `Button`、`IconButton`，`CustomMenu` 的省略号按钮，`getButtonStyling` 的调用方）。
- M1 收尾交接、M1-P2 交接各加"处理结果（M3/P11）"，状态保持 `open`（M3 的收尾关闭）。
- 只核对、不改（C-c，附录 A.5）：`tools/keywords.json` 69 条规则、3 个例外（`analytics` 到 M6、`brand` 到 M9、`plane-workspace-urls` 的两处 `/user-properties/` 到 M4），没有 `until: "M3"`；旧 `ProjectService`（`core/services/project/project.service.ts`）只有 7.3 的三个方法：`getProjectUserProperties`、`updateProjectUserProperties`、`projectIssuesSearch`。

### 2.11 挂载时的取数（W2；7.1；S2）

S2 的 `REQUESTS` 是每个账户每一页的挂载清单唯一的地方（P8b 的裁定 D5），列表在 `e2e/fixtures/mounts.ts`（P10）。P11 加的：

| 页面 | 管理员 | 成员 | 访客 | 不是项目成员的成员 | Task |
|---|---|---|---|---|---|
| 项目的状态设置 | `APP`、`WORKSPACE`、`PROJECT`、`PROJECT_MEMBER` | 同管理员 | 同管理员（页面说没有权限） | `APP`、`WORKSPACE`、`PROJECT` | 3 |
| 项目的标签设置 | 同上 | 同上 | 同上 | 同上 | 5 |
| 管理员的个人主页、访客自己的 | | | `APP`、`WORKSPACE` | | 6 |

访客取 `PROJECT_MEMBER`（显示设置、标签、成员、状态）是项目的包装层的（P8b）：他是项目成员，nerve 答 200。个人主页的卡片来自工作区的包装层取的成员列表，不另取。另在 `7d6a416b` 和最终原型上各跑一次一次性的清单（附录 A.3）：之前之后相同。

### 2.12 页面级副作用的会话核对（7.1）

页面在修改兑现之后提示、改界面、关表单和弹窗的每一处都经 `followInSession`，由它自己的 vitest 守着：另一个标签页换了账户之后才兑现（`lateSettlings` 的两种：换了账户之后 nerve 才回答、换了账户之后 nerve 才拒绝；回答和跟进之间不会换，`followInSession` 在 `await` 之后马上核对），页面什么都不做；那一处不经 `followInSession` 时这个测试失败（附录 A.2 的会话一类）：

| 组件 | 修改 | 跟进 | 守着它的 vitest | Task |
|---|---|---|---|---|
| `use-state-changes.ts`（状态的表单、删除、设默认、拖动） | 新建、修改、移动、删除、设默认 | `done`（关上表单、确认框）；字段下或提示 | `use-state-changes.test.ts` | 2 |
| `use-label-changes.ts`（标签的表单、删除、拖动、移出组） | 新建、修改、移动、移出组、删除 | `done`；名称下或提示；两层的说明 | `use-label-changes.test.ts` | 4 |
| `onboarding/steps/profile/root.tsx` | 名字 | 交给下一步（P9）；名字下或提示 | `root.test.tsx` | 7 |
| `general/form.tsx` | 名字等 | 提示；字段下或提示 | `general/form.test.tsx` | 7 |
| `language-and-timezone-list.tsx`、`start-of-week-preference.tsx` | 时区、语言、每周第一天 | 提示；拒绝提示 | `language-and-timezone-list.test.tsx` | 7 |
| `security.tsx` | 密码 | 清空表单、提示；字段下或提示 | `security.test.tsx` | 7 |
| `api-token/modal/create-token-modal.tsx` | API 令牌的新建 | 显示新令牌、下载 CSV；表单照 M2 显示拒绝 | `create-token-modal.test.tsx` | 7 |
| `api-token/delete-token-modal.tsx` | API 令牌的撤销 | 提示、关上；提示原因 | `delete-token-modal.test.tsx` | 7 |

不跟进的：组件自己的忙碌（表单的 `isSubmitting`、确认框的 `isSubmitting`），换账户时随页面卸载（P9 的裁定 P4）。会话切换的端到端在 P9 的 W3（M3 设计 9.6）；P11 的每一处由上表的 vitest 守着。

### 2.13 静态检查的范围（总体设计 7.7；P9 review 的 M4；P10 的 F-2）

- 根目录 `.oxlintrc.json` 的 `typescript/no-non-null-assertion` 的范围：Task 1 `form-refusal.ts`；Task 2 `use-state-changes.ts`、它的测试、`options/delete.test.tsx`；Task 3 `states-labels.test.tsx`（路径中的方括号照已有的写法转义）；Task 4 `use-label-changes.ts`、它的测试；Task 5 `project-setting-label-group.test.tsx`、`label-drag-n-drop-HOC.test.tsx`；Task 7 `general/form.test.tsx`、`security.test.tsx`、`language-and-timezone-list.test.tsx`、`create-token-modal.test.tsx`、`delete-token-modal.test.tsx` 和测试的 `fake-form.ts`。它们都没有非空断言（W12 的六个变异，附录 A.2）。`no-restricted-imports` 不变：P11 没有新的取数文件。Task 9 把 `no-unreachable` 设为 error（2.9）。
- `tools/keywords.json` 的 `project-address-route`：Task 3 加状态设置页和 `project-states/root.tsx`，Task 5 加标签设置页、`labels/project-setting-label-list.tsx`、`labels/label-drag-n-drop-HOC.tsx`（规则的样例里状态、标签的设置页从"不命中"移到"命中"；`why` 写明 P11 的文件）。规则仍是 69 条、3 个例外。不加新规则（第 3 节第 13 条）。

## 3. 与设计的差异、补充和需要裁定的

原标"待裁定"的 8 条已由控制者裁定（2026-10-11；第 5 条是产品决定，由负责人答复），各条写明裁定；其余是本 Phase 在设计之内做的决定和理由。第 18、19 条和节末的"预检之后的修订"是裁定和预检（H1、M1、L1–L9）之后加的。

1. **Task 的切分**（已裁定 Q1：接受，在上限之内，P7–P10 的先例）：设计 12 节 P11 的草稿是 9 个任务，plan 是 10 个（P7–P10 的先例：在上限之内重新切分）。对应：设计 1（状态设置页）→ Task 1（与标签共用的表单的拒绝、确认框）、2（状态的修改）、3（页面、权限、键盘）；2（标签设置页）→ Task 4（标签的修改）、5（页面、拖动、权限、键盘）；3（死成员和死 prop）→ Task 8；4（oxlint：M3 改到的文件清零，P8a、P8b 留下的只经机械步骤到达的文件）→ 每个 Task 和 Task 8；5（`always-return`、上限）→ Task 9；6（关键词守卫的最终核对、旧 `ProjectService`）→ Task 10 的核对（只核对、不改：两件事在 P8b 已做到）；7（P6、P7、P9、W11 的页面版本）并进各页面的 Task（3、5、6；测试与代码同一个提交，每个 Task 之后 `make e2e` 通过）；8（C9、C10）和 9 的评审是控制者的，9 的 3.20 是 Task 10（只改文档）。另有两件交接来的：M2 的账户页面在会话里跟进（P9 的裁定 P8）→ Task 7；个人主页的页面版本 → Task 6。原型中原来的第一个 Task（状态的全部修改，连同共用的表单的拒绝和确认框）量出 1,458 行块，加上说明会超过约 1,500 行，所以在本 Phase 之内把共用的部分分出来成为 Task 1。10 个 Task 都在约 16 个之内；写 plan 时每个在约 1,500 行之内（最长的 Task 2 是 1,458 行）。任务评审的修正轮和最终评审的修复轮之后，块由提交的代码重新写出，Task 5、7 超过约 1,500 行（1,674、1,832 行：Task 5 加了共用组件的焦点和名字、行内表单的改写和 P7 的核对，Task 7 加了 `fake-form.ts`、`dialogs.ts` 和改导入的 14 个故事）；两个 Task 都已提交，plan 是它们的记录（附录 A.8）。
2. **`AlertModalCore` 在请求路上时关不掉，它的其余 11 个调用方同样**（已裁定 Q2：接受，这是 P9 的 F-1 在组件里的根因修复，P10 修共用组件的先例；条件是逐个核对其余 11 个调用方在每条被拒绝的路上都把 `isSubmitting` 复位，附录 A.4；W17；总体设计 7.7 的 F-1）：状态、标签的删除确认框是 `@nerve/ui` 的 `AlertModalCore`。它原来在 `isSubmitting` 时只让确认按钮加载，取消、Escape、点外面照样关上，回答之后照样跟进（P9 的第一类缺陷）。在 P11 的两个调用方各自绕开（自己包一层、自己忽略关闭）就是同一个规则的两份实现，而且共用组件照旧有这个缺陷；根在组件里，所以改组件（Task 1）。它的其余 11 个调用方随之一样：M4 的工作项删除、草稿删除，M5 的附件删除，M6 的迭代、模块删除，M7 的视图删除、工作区视图删除、收集箱的拒绝和删除，M8 的 Webhook 删除，M2 个人设置的 API 令牌的撤销。`isSubmitting` 是必传的 prop，每个调用方都传：请求在路上时取消禁用、Escape 和点外面不关（与 P9、P10 的每个弹窗相同）。它们的页面不在 M3 的故事里，没有端到端看它们；组件的这一点由 P6、P7 的页面版本看住（`T1.2`、`T1.3`）。前端改动清单的确认框一行写明（Task 10）。核对的结果（附录 A.4）：没有一个调用方在被拒绝之后卡在忙里，不用改调用方；两个工作项的删除确认框的 `onSubmit` 可以不传，不传时一直忙、有了 F-1 就再也关不掉，今天 11 个调用处都传，Task 1 把它改为必传（加固）。收尾的 `ModalCore` 重写保持这一点（第 5 节收尾）。
3. **`AlertModalCore` 的 `tabIndex={1}` 和它的那条 oxlint 警告留着；删除确认框的 Tab 循环不核对**（已裁定 Q3：接受，负责人 2026-10-10 的裁定）：确认按钮的正的 `tabIndex`（`jsx-a11y/tabindex-no-positive`，`@nerve/ui` 的上限中的一条）属于 M3 收尾的 `ModalCore` 重写（brief：P11 不做、不抢先）。所以 Task 1 手改的 `alert-modal.tsx` 结束时仍有这一条警告（M3 设计 7.9 要手改的文件 0 条，这是唯一的例外），`@nerve/ui` 的上限 15 不变；P6、P7 的页面版本核对删除确认框的 Escape、取消、被拒绝时留着，不核对它的 Tab 循环（brief 要每个 P11 的弹窗有键盘的端到端核对：表单有，确认框只到这里）。收尾的重写之后，确认框的 Tab 循环加进 P6、P7 的页面版本（第 5 节收尾）。
4. **放到第三层的拒绝：页面说标签只有两层，不是 nerve 的原话**（已裁定 Q4：接受，照预检的 L3 改正为四种、用更完整的文案；P7b review 第 6 节写"422 `parent_id not_allowed` 显示服务端的说明"）：nerve 的 422 是 `validation_failed`，`errors` 中一项 `{ field: "parent_id", code: "not_allowed" }`，有四种：父标签不存在或在别的项目、自己做自己的父标签、父标签已在别的标签之下、下面有标签的标签放到别的标签之下（`domain/label.go` 的 `checkParent`）；按 `errorMessageKey` 读出的是"有字段不合要求"一类的通用说明，标签的设置页没有 `parent_id` 字段可以把它写在下面。页面的拖放规则（`label-drag-n-drop-HOC.tsx` 的两层规则）本来不让人放到第三层，只有另一个标签页同时改了层级才会被拒绝；这时说"标签只能放在本项目的顶层标签之下，下面有标签的标签留在顶层：标签只有两层"（新文案 `project_settings.labels.parent_not_allowed`，两种语言；四种都照实说，第四种由后半句说明）比通用说明有用。读法照旧经 `fieldErrorKeys`（问题指名 `parent_id` 时是这一句，其余经 `useRefusalToast`）：nerve 先查请求体的字段、再判权限、最后查父标签，`parent_id` 的问题总是单独来的（预检的核对）。只有移动会得到它：移出组只发 `parent_id: null`，nerve 不查父标签，它的拒绝经 `useRefusalToast`，没有走不到的分支和测试。P7 的页面版本走这一条（`T4.7`）。
5. **访客在界面上没有离开工作区的路**（已裁定 Q5：产品决定，负责人 2026-10-11 答复"菜单加入口，收尾做"；P9 review 第 6 节）：nerve 让每个有效成员离开工作区（`POST /api/v0/workspaces/{slug}/leave`，访客在内，9.2），页面上唯一的入口在成员页，而成员页对访客说没有权限（W11 的页面一半，Plane 相同）。负责人的答复：侧边栏的工作区菜单加"离开工作区"，每个角色都有；唯一的管理员离开时 nerve 拒绝，页面说明它的原因。这是 M3 收尾的一个小任务（M3 设计第 12 节收尾，本提交之前的设计提交已写进），P11 不改。在那之前访客经 PAT 离开（第 7 节）。
6. **个人主页：页面不变，每个角色挂载和取的由设计定下**（决策点 3 的 A；3.1；9.2 的一格；M1-P2 交接）：设计已定（3.1：访客看到的个人主页"卡片之外不挂载工作项列表"；成员和管理员看到的工作项列表是 M4 的，旧接口的 404 是已知的过渡），A 就是 Plane 的页面原样，所以 web 不改（Task 6 只加 P9 的页面版本和 S2 的两行）。附录 A.3 的清单：管理员、成员、不是项目成员的成员打开任何人的主页都请求 Plane 的 `/api/workspaces/{slug}/user-issues/{id}/`（404），访客只取应用和工作区的，之前之后相同。另两处留给后面：个人主页的地址转到工作项的标签页（`/profile/{id}/assigned`），页头照 `isIssuesTab` 渲染工作项的筛选工具栏（`ProfileIssuesFilter`，Plane 相同），所以访客也看到它：它的按钮什么都不做（`updateFilters` 在这个用户没有取过筛选时直接返回），只有同一个会话里他先以成员的身份打开过这个主页（之后被降为访客）时，换布局会以旧的筛选请求工作项（`fetchIssuesWithExistingPagination`），nerve 答 404（M4 之后按矩阵的这一格拒绝）：都在 M4 对接工作项列表时处理（第 5 节 M4）；`layout.tsx` 的 `allowPermissions` 不传工作区（只读工作区的 router store，P10 的 36 个之一，第 5 节收尾）。
7. **M2 的账户页面照它们自己的拒绝映射**（Task 7）：七个文件（API 令牌的两个弹窗在内，第 18 条）只改成经 `followInSession` 跟进，字段错误落在哪里、提示说什么照 M2 的写法（general 的 `fieldErrorKeys` 和 `needsErrorBanner`，security 的 `identity.current_password_incorrect` 和 `new_password`，新手引导的 `first_name`，新建令牌的 `label`、`expired_at`），不改用 Task 1 的 `useFormRefusal`：它们的字段和码与状态、标签的不同（security 的 `current_password_incorrect` 不是"已被占用"一类，而它的问题不指名字段），改用会改到 M2 已核对过的显示（W17），不是 P9 的裁定 P8 要的（只要会话的核对）。
8. **Task 9 的文件其余的 14 条警告留着**（已裁定 Q6：接受，Task 9 是照 oxlint 的输出写出的一个机械步骤，正是 R3；M3 设计 7.9 的 R3）：Task 9 只按 `promise(always-return)` 这一条规则改 26 个文件（脚本照 oxlint 的输出写 `return;`，附录 A.6）；其中 5 个文件还有别的警告：`cycles/modal.tsx` 1 条（`no-shadow`）、`issue-activity/helper.tsx` 2 条（`preserve-caught-error`、`react-hooks/exhaustive-deps`）、`modules/module-card-item.tsx` 5 条（`consistent-function-scoping`，可点击的 `div` 的两条规则各 2 条）、`snooze/modal.tsx` 5 条（`no-array-index-key`，可点击的 `div` 的两条规则各 2 条）、`webhook.store.ts` 1 条（`no-accumulating-spread`）。它们是 M4、M6、M7、M8 的组件，在 M3 的故事走不到的页面上；修它们会改那些页面的行为（`div` 变按钮改 DOM、Tab 顺序和样式，`exhaustive-deps` 改副作用运行的时机）而无从核对，正是 R3 不让机械步骤到达的文件在那个 Phase 修的理由。7.9 的"按规则清一类"只要这条规则全仓清零，所以这 26 个文件按只经机械步骤到达的看待：它们的其余警告列给各自的 M（第 5 节）。第 5 节把它们列给 M4、M6、M7、M8。
9. **各包的上限：只有 web、editor 变**（已裁定 Q7：接受，7.9 写的是每次合并时调到新的条数；完成线"各包的 oxlint 上限已调低"的读法）：M3 设计 7.9 写"各包的上限照旧在每个 Phase 合并时调到新的条数"，13.1 写"各包上限调低，review 写明"。P11 改到的包中，web 348 → 303、editor 65 → 64；`@nerve/ui`（15，P11 改到的 5 个文件中只有 `alert-modal.tsx` 的一条，在第 3 条）、`@nerve/propel`（11，P11 改到的 6 个文件原来就是 0 条）、`@nerve/constants`（1，`members.ts` 原来就是 0 条）、`@nerve/types`（0）的条数不变，上限照条数不变；utils 7、hooks 3 P11 没有改到。按"调到新的条数"读，完成线已满足（裁定的读法）。
10. **标签的拖动把手以状态登记，由一个 vitest 看住**（已裁定 Q8：(b)；P8b 的 P7）：`label-drag-n-drop-HOC.tsx` 把行此刻渲染的把手存在状态里（回调 ref），`useEffect` 的依赖有它，把手换了就重新登记。Plane 的写法（`useRef`，依赖里没有把手）只因列表每次渲染都重建 `onDrop`、副作用每次都重新运行才登记到新的把手，今天两种写法登记的结果相同（P7 的页面版本拖了三次、改名之后再拖，都通过），将来有人把 `onDrop` 包进 `useCallback`，编辑之后重新渲染的行就拖不动。Task 5 加 `label-drag-n-drop-HOC.test.tsx`（预检的原型，删短）：web 应用的 vitest 在 node 中运行、没有 DOM，测试自己照 React 的语义写 `useState`、`useRef`、`useEffect`（依赖按 `Object.is` 比较、setter 不变、提交时调用把手的 ref），以 pragmatic-dnd 的替身记下登记的把手：行换了把手时以新的登记、旧的撤销；行没有把手时不登记。每次渲染给同一个 `onDrop`（列表把它包进 `useCallback` 时的样子），所以靠 `onDrop` 重建才重新登记的写法登记不到新的把手（修订中发现：原型的测试原来每次渲染给新的 `onDrop`，`T5.14` 因此存活，W18 的运行看出了它）。回到 ref 的变异 `T5.14`（原型中的 `T5.ref`）和把手不在依赖里的 `T5.15` 都由它发现（后者也由 oxlint 的上限）。没有新的依赖、测试环境，没有写 `as`、`any`、`!`（替身元素是 `Object.create(null)`）。
11. **`confirm-workspace-member-remove.tsx` 的英文归收尾**（P9 review 第 6 节"归 P11 或收尾"）：P11 不改这个文件（它是 P9 的成员页的），也不改 P11 自己的页面上的 Plane 英文（"Delete State"、"State created successfully." 等，P10 的先例：写死的英文是收尾的一张清单）。收尾的清单加上这个文件的"Leave workspace?"、"Remove {名字}?"和移出的正文（第 5 节收尾）。
12. **页面上看得到的不同**（W17；前端改动清单的六行，Task 10）：
    - 会话：状态的新建、修改、移动、删除、设默认，标签的新建、修改、移动、移出组、删除，新手引导的资料一步、个人设置的 general、时区、语言、每周第一天、密码、API 令牌的新建和撤销，在另一个标签页换了账户之后兑现时，页面不提示、不关表单和确认框、不清空表单、不在字段下写东西、不显示也不下载新令牌（原来照样做，显示在那个账户的页面上；新令牌的 CSV 下载到那个账户的页面上）。
    - 错误：状态、标签的名称已被占用在名称下，名字太长、说明不被允许在各自的字段下（原来都是提示）；其余的拒绝和移动、删除、设默认的拒绝在提示里说 nerve 的原因；标签放到第三层说只有两层（第 4 条）。
    - 忙碌：状态、标签的表单在 nerve 回答、页面跟进之前忙，按钮显示"Creating"、"Updating"、"Adding"，取消禁用（原来状态的表单从不忙，连按两下发两次；标签的取消照样可按）；删除确认框在请求路上时关不掉（第 2 条），被拒绝时留着，nerve 删除之后才关（原来请求在路上时可按 Escape、取消关掉，回答之后不经会话的核对就关上）；新建 API 令牌的弹窗在请求路上时取消按不了（第 18 条）。
    - 权限：标签的组的菜单和把手只给能改标签的人（原来给每个成员，点了 nerve 拒绝）；没有标签时的"添加标签"按钮同样；标签的一行只由把手开始拖动，把手只给能改的人、正在编辑的行没有把手：不能改标签的人照旧拖不动（原来拖放组件自己判断权限），正在编辑的标签现在拖不动，它的一行也不再是放下的目标（原来没有把手时整行就是把手，能改的人正在编辑的那一行整行都能拖、也接得住别的标签）；项目的其他成员拖不动状态（照旧，P6 的页面版本现在核对它）。
    - 表单：顶部的新建表单在编辑别的标签时照旧显示"Add"、提交的是新建（原来显示"Update"，按了什么都不做；表单是新建还是修改只由 `labelToUpdate` 决定，预检的 L8）；改名一个没有颜色的标签时照它原有的颜色（没有）发出，不再发出 `#000`（原来顺带把它改成黑色；表单中它的色块照旧显示黑色，与列表相同；裁定 T4-c）。
    - 键盘和名字：状态一行的编辑、删除、设默认在 Tab 顺序里（原来指针经过之前不在），行里有焦点时显示；编辑、删除、组的加状态按钮有名字，删除按钮 Tab 到达时有焦点的轮廓（原来 `focus:outline-none`）；两个表单的颜色按钮名为"Color"（`common.color`），标签表单的颜色按钮 Tab 到达时有焦点的轮廓；项目设置侧边栏的返回箭头名为"Go back"（`common.go_back`）；组标题是按钮（Tab 能到，Enter、空格展开收起，`aria-expanded`）；标签一行的菜单和删除按钮有名字，Tab 到达时显示（原来只在指针经过时显示）；标签组的展开按钮以标签的名字为名（Headless UI 给它 `aria-expanded`）；标签一行的拖动把手不再是没有名字、看不见的 Tab 停留点；侧边栏项目的行和收藏夹的外层手柄按钮照旧是显示时的停留点，里面只剩 `aria-hidden` 的图标，现在没有名字（第 19 条，第 5 节收尾的键盘清单）。
    - 共用组件的焦点（裁定 T5-a、T5-b、T5-e）：propel 的 `Button`、`IconButton` 在键盘取得焦点时有浏览器的轮廓，离按钮 2 像素（原来 `focus-visible:outline-none`，看不出焦点）：应用中的每个 `Button`，`IconButton` 的 32 个文件 37 处（其中五处照旧看不到或 Tab 到不了，第 5 节收尾）；`getButtonStyling` 的 9 个文件 12 处中能取得焦点的 4 个文件 6 处：M3 的 4 个链接现在有轮廓，M4 的两处由它们自己的样式照旧（工作项筛选的"添加筛选"按钮的轮廓是 `CustomSearchSelect` 在 `data-focus` 上画的那一条，布局下拉框的按钮自带 `outline-none`）；`@nerve/ui` 的 `CustomMenu` 的省略号按钮 Tab 到达时有轮廓（17 个文件 17 处；成员页的 `member-columns.tsx` 照旧看不见：它自己的 `opacity-0`）。用指针点的人看不到不同。
    - 看不出的：两页读地址只从路由参数（router store 晚路由一次提交的那一刻，原来可能显示上一个项目的标题和权限）；在一个项目的设置页之间移动、进状态页时不再重取状态（修复轮的 F-3 删去根组件自己的取数：项目的包装层一直挂载，持有那一个取数，标签页原来就是这样；裁定 F-14）；项目下拉框的 `ComboDropDown` 收 `role="presentation"`（清掉一条 oxlint 警告），辅助技术读到的照旧：它的三个调用方在桌面上都给根 `tabIndex`（工作项的表单给正数，迭代、模块的表单给 0），浏览器不理会可以聚焦的元素上的这个角色，根照旧是没有名字的 Tab 停留点（M4、M6 的表单上；裁定 T8-a，改法在第 5 节收尾的键盘清单）；状态组的进度以组为 key（每组一行，显示不变；M6 的页面上）；死 prop 删除之后组件取原来的默认值；`then()` 的回调加 `return;`（附录 A.6）。
    - 个人主页、W11：没有（页面不变，第 6 条）。
13. **不加新的关键词规则**：`useStateChanges`、`useLabelChanges` 是设置页上的修改唯一的地方（它们的 vitest 守着会话的核对），但 store 的 `createLabel` 另有 M4 的五个调用方（工作项页的标签选择器里新建标签：`issues/issue-detail/label/root.tsx`、`select/root.tsx`、`issue-layouts/properties/label-dropdown.tsx`、`issues/select/dropdown.tsx`、`base.tsx`），规则要给它们各写一个例外。M4 对接这些页面时改经 `useLabelChanges().create`（第 5 节 M4），之后可以加规则。
14. **超过约 400 行的文件**：变长的是 `power-k/ui/pages/context-based/work-item/commands.ts` 415 → 418（Task 9 的三处 `return;`；它是 M4 的工作项命令，原来就超过，拆它是 M4 的，第 5 节 M4）；`tools/keywords.json` 2,387 → 2,390（规则的数据，P8a 起就是例外）；两种语言的 `common.json` 527 → 528（文案的数据，Task 5 的 `common.color`）；`e2e/fixtures/assert/identity.ts` 472 → 485（P9、W11 共用的 `expectNoAddress`：Task 6 评审的 m1，修复轮的 F-5 把它放在这里；这个文件原来就超过，它的拆分是 P10 第 5 节收尾的一项）。行数不变的：`@nerve/ui` 的 `custom-menu.tsx` 492 行（Task 5 只改省略号按钮的类名）。其余 P11 写或改的文件都在约 400 行以内（附录 A.8）。
15. **e2e 的共用部分**：P6、P7 要拖动，P8 的 `dragOnto` 移进 `e2e/fixtures/project-pages.ts`（brief：移进 fixture，不复制），签名改为拖动的项、把手、目标和目标中的点；P8 经 `dragInSidebar` 用它，行为照旧（P8 照旧通过，反复 3 次）。`tabCycle` 加 `presses`（默认 40，P10 的调用方照旧）：状态、标签的表单在页面上、不是弹窗，Tab 走过表单就离开它，只按表单的停留点的个数。
16. **store 的两个 getter 以项目的 id 读**（P10 的 F-2 的延伸）：`groupedProjectStates`、`projectLabelsTree` 是读 router store 的项目的 `computed`，状态、标签的设置页是它们唯一的读者；改为 `getGroupedProjectStates(projectId)`、`getProjectLabelsTree(projectId)`（`computedFn`，与 `getProjectStates`、`getProjectLabels` 一样），页面从路由参数传项目。store 的测试改读它们（`T3.8`、`T5.12`）。`projectStates`、`projectLabels`（还读 router store）由 M4 的页面读，不改。
17. **`state_group.tsx` 的三行留在 41 行里**：`--rows M3` 中 `progress-stats/state_group.tsx` 的 `state`、`completed`、`total`（`TStateGroupData` 的列）是设计第 1 节按路径误归到 M3 的 41 行中的 3 行（M6 的迭代、模块的进度），P11 只改了这个文件的 key（Task 8）；`completed`、`state` 有读者（`domains.mjs` 以字段名判断，读不到 `distribution.map` 里的读），只有 `total` 真的没有读者，随 M6 对接进度时处理（第 5 节 M6）。
18. **新建 API 令牌的弹窗在请求路上时关不掉；A11 的第二个故事改为离开页面**（预检的 M1，裁定的修法 (a)：Task 7 也改 security 页的两个 API 令牌的弹窗，并照 7.7 的弹窗规则核对它们）：新建的弹窗原来在请求路上时取消照样能按（M2 的写法：弹窗的"一次打开"由 `phase` 计数，迟到的回答什么都不做，A11 的第二个故事正是按了取消再等回答）。这违反 F-1（请求在路上时弹窗关不掉），所以取消在 `isSubmitting` 时禁用（弹窗的 `handleClose` 本来就不关），表单忙到页面跟进做完。弹窗的一次打开现在只在离开页面时提前结束，`phase` 照旧管它和关上之后 350 ms 的清理，所以 A11 的第二个故事在请求被扣住时核对取消不可用，再经浏览器的后退离开页面（tokens 页从 security 页在应用之内进入），之后的断言照旧：迟到的回答不显示令牌、不下载，列表里有它，再打开的弹窗照旧要新的令牌。撤销的确认框是 `AlertModalCore` 的调用方，传了 `isSubmitting`，第 2 条之后请求在路上时关不掉、忙到跟进做完；两个弹窗都在会话里跟进（2.12），连按两下：按钮在忙时不可用（`loading`）。W17：取消在请求路上时按不了（第 12 条）。
19. **拖动把手只给指针用：`@nerve/ui` 的 `DragHandle` 不是 Tab 停留点、辅助技术不读它**（预检的 L7，裁定的修法 (a)）：pragmatic-dnd 的拖动只认指针，把手原来是没有名字、看不见（`opacity-0`）的 Tab 停留点，按了 Enter 什么都不做。`DragHandle` 的 4 个调用方都只把它当指针的把手：标签的行（它就是登记的把手）、侧边栏项目的行、收藏夹和收藏项（`DragHandle` 只是外层的按钮或 `div` 里面的图标；登记的把手是项目的行的外层按钮、收藏夹和收藏项的整个元素）。所以改在组件里：`tabIndex={-1}`、`aria-hidden`，每个调用方随之一样（W17：侧边栏里的 `DragHandle` 同样不再是停留点；项目的行和收藏夹的外层按钮照旧是显示时的停留点，里面只剩 `aria-hidden` 的图标，现在没有名字，交给第 5 节收尾的键盘清单（最终评审的 M5）；P10 的 P8 照旧通过）。状态的行没有把手按钮（拖动指示是一个 `div`），不用改。标签组的展开按钮（`Disclosure.Button`）以标签的名字为名，与 Task 2 的状态组标题一致。P7 的页面版本按名字和 `expanded` 找到它，并从 Feature 的删除按钮按 Tab 直接到 Docs 的菜单（经过把手而不停），`T5.16`、`T5.17` 让它们失败。

**预检之后的修订**（预检的发现和控制者 2026-10-11 的裁定；每一条都改在原型的 Task 脚本里，从改到的第一个 Task 起重新写出快照、跑门禁和变异，plan 的块由新的快照写出，再逐 Task 复现，附录 A）：

| 发现或裁定 | 修订 | Task |
|---|---|---|
| H1：工作区设置侧边栏的 `className` 有读者 | 不删：`settings/mobile/nav.tsx:38` 经 `hamburgerContent` 传入（`ComponentType<{ className?: string; isMobile?: boolean }>`，`deadsym.mjs` 看不到经组件类型的变量的读）；删去的行 27 → 26，产品代码的行 80 → 54。其余 25 处删除照同一形状逐个扫过（组件作为值传入、`component=`、`as=`、渲染 prop、组件的表）：没有（附录 A.5 的 C-b） | 8 |
| M1：security 页的 API 令牌的两个弹窗不经会话的核对 | 修法 (a)：两个弹窗经 `followInSession`，各两行换账户之后兑现的 vitest，2.12 两行，`T7.7`、`T7.8`；照 7.7 的弹窗规则核对：新建的取消在请求路上时可按，改为禁用（第 18 条，`T7.9`，A11） | 7 |
| L1：标签的删除确认框"原来先关"不实 | 原来是 nerve 删除之后才关；原来的缺陷是请求在路上时关得掉、回答之后不经会话的核对就关（第 12 条、2.4、附录 A.4） | 文档 |
| L2：2.12 对 `lateSettlings` 两种的说法不实 | 两种都是换了账户之后 nerve 才回答或才拒绝（2.12） | 文档 |
| L3：`not_allowed` 有四种；移出组的那一行走不到 | 四种，文案更完整（两种语言）；移出组的拒绝经 `useRefusalToast`，删去走不到的测试行（hook 的测试 27 → 26） | 4 |
| L4：三处复制链接的拒绝没有处理，不是交回调用方 | 附录 A.6 改为"交回调用方 6、没有处理 6"（Task 9 的评审又改正了处数：链后 23、交回 6、没有处理 6，裁定 T9-a）；迭代、模块的复制链接交给 M6，默认视图的交给 M7（第 5 节） | 文档 |
| L5：页面测试的 store 替身不看项目的 id，状态的替身是空的 | 替身只对页面的项目回答，状态的替身有 web 的两组；`T3.10`（根组件按别的 id 读状态） | 3、5 |
| L6：行内表单的测试标题说的比断言多 | 测试一并记下 react-hook-form 的 `setError`，断言名称下的文案；`T4.11` 在 vitest 一层也被发现 | 4 |
| L7：标签行的把手和组的展开按钮 | 第 19 条；`T5.16`、`T5.17` | 5 |
| L8：顶部的表单在编辑别的标签时显示"Update"、提交的是新建 | 表单只由 `labelToUpdate` 决定按钮的文字和发出的修改，`isUpdating` prop 删除；W17 一行 | 4 |
| L9：写错位置的 `return;` 过得了每道检查 | 根目录 `.oxlintrc.json` 的 `no-unreachable` 设为 error（今天 0 处，上限不变）；`T9.3` | 9 |
| Q2 的条件和加固 | 其余 11 个调用方的核对写进附录 A.4；两个工作项的删除确认框的 `onSubmit` 必传；第 5 节收尾一行写明重写保持这一点 | 1 |
| Q5 | 第 5 条、第 5 节收尾、第 7 节；M3 设计第 12 节收尾（单独的设计提交） | 文档 |
| Q8 (b) | 第 10 条；`label-drag-n-drop-HOC.test.tsx`、`T5.14`、`T5.15` | 5 |

**交接到 P11 的事项的落点**（brief 的"Carried into P11"）：

| 来源 | 事项 | 落点 |
|---|---|---|
| P10 spec 第 5 节、P10 review 第 6 节 P11 一行 | 标签设置页的拖动把手只因 `onDrop` 重建才重新登记（P8b 的 P7） | Task 5（把手存在状态里，由 `label-drag-n-drop-HOC.test.tsx` 看住，第 3 节第 10 条） |
| | 标签的同名 409 落到名称下（T11-a） | Task 4（`useLabelChanges`，`T4.6`；P7 的页面版本） |
| | 两页和标签的两个组件从 router store 读地址；加进 `project-address-route`（F-2、F-12） | Task 3（状态页、根组件）、5（标签页、列表、拖放组件）；2.13 |
| | P10 定下的写法：A-M2、`allowPermissions` 写明工作区和项目、store 自己的权限的 vitest（`callerInWeb`、`its`）、`lateSettlingsAnswering`、`enabledWithin`、`closedWithin` | 每个页面的 Task：确认框被拒绝时留着（Task 2、4）；`states-labels.test.tsx`、`project-setting-label-group.test.tsx`（Task 3、5）；两个 hook 的测试（Task 2、4）；P6、P7 的页面版本 |
| P9 spec 第 5 节、P9 review 第 6 节 P11 一行（裁定 P8） | M2 的账户页面自己的提示不核对会话（五个文件；预检又找到 security 页的 API 令牌的两个弹窗） | Task 7（第 3 节第 7、18 条） |
| | 个人主页的页面（故事 P9 的页面版本，C10） | Task 6（第 3 节第 6 条）；C10 是控制者的 |
| | 访客在界面上没有离开工作区的路 | 第 3 节第 5 条（负责人：收尾在侧边栏的工作区菜单加入口） |
| | `/{slug}/profile/{id}` 调 Plane 的 `user-issues`（404），工作项一栏一直是骨架 | 设计 3.1 定下：成员和管理员的工作项一栏是 M4 的，404 是已知的过渡（第 3 节第 6 条，附录 A.3） |
| P9 review 第 6 节收尾一行 | `confirm-workspace-member-remove.tsx` 的英文 | 收尾（第 3 节第 11 条） |
| P8b spec 第 5 节、附录 A.8、P8b review 第 6 节 P11 一行 | 只经机械步骤到达的 36 个文件中 5 个的 14 条 | Task 2（`group-item.tsx` 2 条）、8（`base.tsx` 5 条、`use-work-item-filters-config.tsx` 3 条、`state_group.tsx` 1 条）、9（`work-item/commands.ts` 3 条 `always-return`）；附录 A.5 |
| P8b spec 附录 A.9 | `state_group.tsx` 的 `completed`、`state`、`total` | 41 行之一，M6（第 3 节第 17 条） |
| | `label-item-block.tsx` 的 `draggable` | Task 5 |
| | `group-list.tsx` 的三个 `className`、`state-item-title.tsx` 的 `shouldShowDescription` | Task 2 |
| | `common-attributes.tsx` 的 `handleFormOnChange` | Task 8 |
| | `project-create-buttons.tsx` 的 `isMobile` | 已在 P10 删除（P10 改写创建项目的按钮），没有落点 |
| P8b 的裁定 P1 | `dropdowns/project/base.tsx` 的 getter 的类型 | Task 8（`Project`） |
| P8b spec 第 5 节、P8a spec 第 5 节 | 核对 M3 加的关键词例外只剩 `until: "M4"` 的一条 | Task 10 的核对（附录 A.5 的 C-c） |
| P8a spec 附录 A.8、P8a review 第 6 节 | 只经机械步骤到达的 19 个文件的 15 条 | 现在只剩 `create-webhook-modal.tsx` 1 条（`always-return`，Task 9）和 `use-work-item-filters-config.tsx` 3 条（与 P8b 的同一组，Task 8）：其余的文件已由 P9、P10 手改清零（附录 A.5） |
| P8a spec 第 5 节 | `always-return` 全仓清零；`--rows M3`（附录 A.9） | Task 9；Task 8 和附录 A.9 |
| P8a review 第 6 节"契约"的 **P11** | 下一个改 `workspace.yaml` 的人收紧 `organization_size` | 这是 P8a 修复轮的搁置项 P11，不是本 Phase；P11 不改契约，照旧交给下一个改它的人（收尾的清单） |
| P7a spec 第 5 节 P11 一行 | P6 的页面版本（含服务端并发下仍拒绝时 409 的说明）；W11 的页面版本 | Task 3 |
| P7b spec 第 5 节、P7b review 第 6 节"P11（页面）" | 写标签的只有项目管理员和是工作区管理员的项目成员，控件照此隐去 | Task 5（组的菜单和把手、空状态的按钮也是） |
| | 409 落到名称字段 | Task 4 |
| | 422 `parent_id not_allowed` 显示服务端的说明 | Task 4，第 3 节第 4 条（已裁定：页面说标签只有两层） |
| | 拖进空的子列表时只发 `parent_id` | P7b 的 store 已是这样；P7 的页面版本核对（UI 拖到 Bug 之下只发 `parent_id`） |
| | 删除一个标签连同它的子标签：刷新 store | P8b 的 store 已删除子标签（`label.store.test.ts`）；P7 的页面版本核对 Defect 下的标签一起不见 |
| P4a spec 的 M1-P2 一行、M1-P2 交接 | 个人主页的页面（C10） | Task 6；Task 10 写交接的处理结果 |
| M1 收尾交接 | 死成员和死 prop、oxlint | Task 8、9；Task 10 写处理结果；附录 A.5、A.6、A.9 |
| M2 收尾交接 | 只有第 12 节留着，等 M3 的收尾 | P11 没有它的事项 |
| P1–P6 的 spec 和 review、P7a review、P8b review、P10 review 中其余的"P11" | 各自的搁置项编号（如 P7a 修复轮的 P11：项目 HTTP 测试的常量；P10 修复轮的 P11：扩展侧边栏的拖动，F-4 已解决）或"前端在 P8–P11"的泛指 | 没有落点 |
| Codex 设计评审的 M1-closeout 一行 | 死成员、误归的 41 行、oxlint、全仓规则清零、下调上限；"不能只因归别的 M 自动标误报" | Task 8、9；41 行逐行列出（第 5 节），留下的每一行写明谁读它（附录 A.9） |
| 设计 7.3 | 旧 `ProjectService` 只剩三个方法 | Task 10 的核对 |
| 设计 9.5 | P11 没有点名的 vitest（9.5 的 P11 一项是页面的） | 2.12 的每一处有它的 vitest，各有一个发现它的变异 |
| 设计 3.20、13.1 中 P11 的行 | README、前端改动清单、M1 收尾和 M1-P2 的交接 | Task 10 |

## 4. 验收标准（完成线，M3 设计 12 节 P11）

- [ ] 全部 21 个故事（W1–W12、P1–P9）的页面版本和接口版本通过（P9 没有接口版本：它的接口在 M4，设计第 2 节；W12 没有页面版本：它是命令行）：worktree 中 `make e2e` 110 个全部通过（`7d6a416b` 是 106 个）；`make test-web` 通过（web 应用 106 个测试文件、965 个测试，附录 A.1）。
- [ ] P6、P7、P9、W11 的页面版本：管理员的修改（请求体、扣住时的忙碌、被拒绝时的说明）、成员只能看、是工作区管理员的项目成员能改、访客看到没有权限、访客打开个人主页不取工作项（S2）。
- [ ] 2.12 的每一处有一个"不经 `followInSession` 时失败"的 vitest（附录 A.2 的会话一类）。
- [ ] `domains.mjs --rows M3` 剩下的每一行写明文件、名字、谁读或传它（附录 A.9），按路径误归的 41 行列出（第 5 节），写进评审是控制者的。
- [ ] oxlint：P11 有手改的文件 0 条警告，除了 `alert-modal.tsx` 的一条（第 3 节第 3 条）和 Task 9 的文件的其余 14 条（第 3 节第 8 条）；P8a、P8b 留下的只经机械步骤到达的文件 0 条；`promise(always-return)` 全仓 0 条；上限 web 348 → 303、editor 65 → 64，其余不变（第 3 节第 9 条）；没有新的抑制（附录 A.5）。
- [ ] `node tools/keywords.mjs` 通过：69 条规则、3 个例外（`analytics` 到 M6、`brand` 到 M9、`plane-workspace-urls` 到 M4，count 2），没有 `until: "M3"`；旧 `ProjectService` 只有三个方法；`tsc`、knip 通过。
- [ ] 根目录 `.oxlintrc.json`（2.13、2.9）：P11 新建的模块和测试在 `no-non-null-assertion` 的范围里，每一组各有一个变异让 `check:lint` 失败；`no-unreachable` 是 error（`T9.3`）。
- [ ] 没有新的 `as`、`any`、`!`（附录 A.7）。
- [ ] 逐 Task 复现：从 `7d6a416b` 照 plan 应用，每个 Task 之后门禁通过，最终与原型逐文件相同（附录 A.10）。
- [ ] 控制者的浏览器核对 C9、C10 写进评审（9.7；不在 plan 中）。

## 5. 不在 P11 范围内

- **M4**：
  - 工作项的页面上的标签新建（`issues/issue-detail/label/root.tsx`、`issue-detail/label/select/root.tsx`、`issue-layouts/properties/label-dropdown.tsx`、`issues/select/dropdown.tsx`、`issues/select/base.tsx` 直接调 store 的 `createLabel`）：改经 `useLabelChanges().create`（会话的核对、名称已被占用在名称下），之后可以加关键词规则（第 3 节第 13 条）。
  - 个人主页的工作项一栏（成员和管理员；`user-issues` 的 404，3.1）和访客看到的筛选工具栏（第 3 节第 6 条）。
  - `power-k/ui/pages/context-based/work-item/commands.ts` 418 行（第 3 节第 14 条）。
  - `--rows M3` 中按路径误归的 16 行：`workspace-draft-issues/base.ts`（附录 A.9）；工作项搜索的类型和参数（`types/src/workspace.ts` 24 行、`workspace.service.ts` 3 行、`types/src/project/projects.ts` 9 行，P8a 已记）。
  - Task 9 的文件中 M4 的：`issue-activity/helper.tsx` 2 条（`preserve-caught-error`、`exhaustive-deps`）；复制的拒绝没有处理的两处（`helper.tsx` 的评论链接、`peek-overview/header.tsx` 的工作项链接，附录 A.6；P10 已把复制列给 M4）。
  - 两个工作项的删除确认框的 `onSubmit` 已必传（Task 1，附录 A.4），M4 改写它们时保持。
  - P8a、P8b、P9、P10 交给 M4 的照旧。
- **M6**：`progress-stats/` 的 8 行（`label.tsx` 5 行、`state_group.tsx` 3 行，第 3 节第 17 条）；Task 9 的文件中 M6 的：`cycles/modal.tsx` 1 条（`no-shadow`）、`module-card-item.tsx` 5 条；迭代、模块的复制链接的拒绝没有处理（`cycles/quick-actions.tsx`、`modules/quick-actions.tsx`：复制交给菜单的 `action`，它是 `() => void`，丢下 promise，附录 A.6）；P10 交给 M6 的照旧。
- **M7**：`workspace-notifications.ts` 的 17 行（附录 A.9）；`snooze/modal.tsx` 5 条；默认视图的复制链接的拒绝没有处理（`workspace/views/default-view-quick-action.tsx`，附录 A.6）；P10 交给 M7 的照旧。
- **M8**：`webhook.store.ts` 1 条（`no-accumulating-spread`）；P10 交给 M8 的照旧。
- **收尾**：
  - `ModalCore` 的重写（负责人 2026-10-10；M3 设计第 12 节收尾）：`AlertModalCore` 的 `tabIndex={1}` 和它的一条警告（第 3 节第 3 条），之后 P6、P7 的页面版本加上删除确认框的 Tab 循环。重写之后，`isSubmitting` 时取消禁用、Escape 和点外面不关这一点照旧（第 3 节第 2 条）。
  - 写死的英文：P11 改到的状态、标签的文件中 Plane 原有的（"Delete State"、"Delete Label"和它们的正文、"State created successfully."、"State updated successfully."和它们的标题"Success!"（`project-states/create-update/create.tsx:32`、`update.tsx:30`，裁定 F-14）、"Name"、"Describe this state for your members."、"Cancel"、"Create"、"Creating"、"Update"、"Updating"、"Cannot delete the default state."、"Cannot have an empty group."、"Mark as default"、"Marking as default"（`project-states/options/mark-as-default.tsx:43`）、"Default"，标签的菜单的"Edit label"、"Delete label"、"Remove from group"（`labels/project-setting-label-item.tsx:55,65`、`project-setting-label-group.tsx:60,69`）等）；`confirm-workspace-member-remove.tsx`（第 3 节第 11 条）；P9、P10 列过的照旧。
  - 离开工作区的入口（负责人 2026-10-11："菜单加入口，收尾做"；M3 设计第 12 节收尾）：侧边栏的工作区菜单加"离开工作区"，每个角色都有；唯一的管理员离开时 nerve 拒绝，页面说明原因（第 3 节第 5 条）。
  - 键盘清单（P11 的任务评审找到的，P11 不改；收尾逐项改，或抄进各 M 的交接）：
    - react-color 的 `TwitterPicker` 的十六进制输入框带内联的 `outline: 'none'`（第三方的样式）：状态、标签的颜色弹层（`project-states/create-update/form.tsx:75`、`labels/create-update-label-inline.tsx:105`）中 Tab 到达它时只看得到光标。
    - propel 的 `IconButton` 没有名字的 27 处（应用中共 37 处，其余 10 处有 `aria-label`）：M2 1 处（`settings/profile/modal.tsx`）；M3 6 处（`profile/sidebar.tsx`、`project/search-projects.tsx`、`settings/mobile/nav.tsx`、`settings/workspace/sidebar/header.tsx`、`sidebar/sidebar-toggle-button.tsx`、`workspace/sidebar/projects-list-item.tsx:346`）；M4 8 处（`comments/quick-actions.tsx`、`core/description-versions/modal.tsx`、`issue-activity/activity-filter.tsx`、`issue-activity/sort-root.tsx`、`issue-detail-quick-actions.tsx`、`quick-action-dropdowns/issue-detail.tsx`、`peek-overview/header.tsx`、`rich-filters/filters-toggle.tsx`）；M6 6 处（迭代、模块详情的页头，`cycles-view-header.tsx`、`cycles/quick-actions.tsx`、`module-view-header.tsx`、`modules/quick-actions.tsx`）；M7 6 处（`views/quick-actions.tsx`、`views/view-list-header.tsx`，通知侧边栏的 4 处）。
    - 焦点的轮廓看不到或 Tab 到不了的 `IconButton` 五处：`workspace/sidebar/favorites/favorites-menu.tsx:215`（外层 `opacity-0`，没有 `focus-within`）、`workspace/sidebar/projects-list-item.tsx:346`（项目一行的菜单，`CustomMenu` 的根 `opacity-0`）、`issue-activity/activity-filter.tsx:33`（外面的 `PopoverMenu` 按钮 `outline-none`，M4）、`workspace/sidebar/projects-list.tsx:151` 和 `projects-list-item.tsx:407`（指针经过之前 `hidden`，Tab 到不了）。成员页的 `project/settings/member-columns.tsx:72` 同样：`CustomMenu` 的省略号按钮带它自己的 `opacity-0`（P10 的成员页原有的）。
    - 项目下拉框（`dropdowns/project/base.tsx`）的根：调用方给它 `tabIndex`（工作项的表单给正数，迭代、模块的表单给 0），根是一个可以聚焦、没有名字、按 Enter 打开列表的 Tab 停留点，Task 8 给它的 `role="presentation"` 在可以聚焦的元素上不生效。改法：把 `tabIndex` 给里面的按钮，根不再可以聚焦，`role="presentation"` 才成立。工作项的标签选择（`issues/select/base.tsx:156-159`，M4）是同样的写法。
    - 侧边栏的两个外层手柄按钮（最终评审的 M5）：项目的一行（`workspace/sidebar/projects-list-item.tsx:301-314`，pragmatic-dnd 登记的把手）和收藏夹（`workspace/sidebar/favorites/favorite-folder.tsx:196-207`，收藏夹整个可拖，按钮是拖动的图示）的 `<button>`：Task 5 让里面的 `DragHandle` 不再是停留点、辅助技术不读它之后，它们照旧是显示时的 Tab 停留点（指针经过时；项目的一行在菜单开着时和扩展的项目侧边栏里也显示），现在没有名字，按 Enter 什么都不做。改法：给它 `tabIndex={-1}` 和名字，或给这一行自己的键盘移动。
    - 状态的表单打开时把焦点放到名称（一致的候选，最终评审的观察）：标签的行内表单打开时焦点在名称上（`setFocus("name")`），状态的表单不移动焦点，Tab 从页面原来的位置走。
  - `ButtonAvatars`（`dropdowns/member/avatar.tsx:17`）的 `showTooltip`：必传，没有代码读它，9 个文件的 10 处都传（成员下拉框的 `dropdowns/member/base.tsx`；M4 的工作项详情的侧边栏、预览的属性；M5 的附件；M6 的迭代 2 处、模块 2 处；M7 的收集箱、视图）。`deadsym.mjs` 看不到每个调用方都传、却没有读者的 prop；删它要同时改跨 M3–M7 的每一个调用方，所以在收尾一次做。
  - 只读工作区的 router store 的 36 个文件（P10 的跟进 F-12）中有个人主页的 `layout.tsx`（第 3 节第 6 条）。
  - 按路径误归的 41 行写进 M4、M6、M7 的交接（13.2，本节上面各行）。
  - 收尾的交接一轮（M3 设计 13.2，M2 收尾的先例；修复轮的 F-10）：M4–M8 的五份 M1 收尾交接（`M4-issue-core/handoffs/M1-closeout.md:49`、`M5-files/handoffs/M1-closeout.md:28`、`M6-cycles-modules/handoffs/M1-closeout.md:28`、`M7-collaboration/handoffs/M1-closeout.md:28`、`M8-open-release/handoffs/M1-closeout.md:40` 的"按规则清一类"）写明"`eslint-plugin-promise(always-return)` 已由 M3/P11 全仓清零（M3/P11 spec 2.9）"。P11 不改这五份交接，由收尾的交接一轮改。
  - P8a 修复轮的搁置项 P11（`Workspace.organization_size`，下一个改 `workspace.yaml` 的人）照旧。
  - P10 第 5 节收尾的其余各项（Safari 的输入法、`CustomSelect` 上的 Enter、`aria_labels.app_sidebar.*`、`identity.ts` 的拆分、之前各 Phase 的端到端的发现、表情的 ETag）照旧，P11 的页面没有被它们挡住。
  - P11 让它们不再成立的"将来"句（之前各 Phase 的 spec、plan、评审中"P11 将…"的句子是记录，不改）：M3 设计 §15 中 P11 的一行由评审的提交改。

## 6. 风险

- **S2 的清单是一份要维护的名单**：M4–M7 每在挂载路径上加一个取数，S2 都会失败，直到清单改过（P8b 的裁定 D5）。
- **拖动的端到端**：P6、P7 的拖动经 `dragOnto`（按住把手、在行内移动、移到目标的点上），落点由 pragmatic-dnd 的命中区决定（标签树的项在上半部是"放在之前"，`{ x: 60, y: 2 }`）。它的命中区变了时这一步先失败（退路：照新的命中区调整点）；反复 3 次通过（附录 A.1）。
- **A11 经浏览器的后退离开页面**：新建令牌的请求被扣住时离开页面，靠的是 tokens 页在应用之内由 security 页进入，后退也在应用之内（第 3 节第 18 条）；这样进入的方式变了时这一步先失败（退路：经侧边栏的按钮离开，弹窗那时不挡住它的话）。
- **标签表单打开时的焦点**：表单的副作用在表单显示之后才可能把焦点放到名称（`setFocus`），故事先等名称有焦点再走 Tab（原型中第一次并行运行时见过一次焦点被后到的副作用抢走，加了这一步之后反复都通过）。
- **拖动把手的 vitest 自己写 React 的语义**（第 3 节第 10 条）：它照 React 的 `useState`、`useRef`、`useEffect` 和提交时调用 ref 的次序写；React 改了这些次序时，测试和页面会不一致，P7 的页面版本照旧拖动、看住真实的那一边。
- **不是 git 仓库的副本里 S3 失败**（P4b spec F4）：复现在副本中运行，只有 S3 因为读不到提交而失败，由复现脚本单独认出；worktree 中 110 个都要通过。

## 7. 已知的限制、交接和关闭条件

- **关闭**：M1 收尾交接的死成员、oxlint（`--rows M3` 剩下的每一行写明、全仓 `always-return` 清零、上限照新的条数；交接在 M3 的收尾改为 `closed`）；M1-P2 交接的个人主页的页面一侧（C10 写进评审之后）。P10、P9、P8b、P8a 的 review 第 6 节 P11 一行；P7a、P7b 的页面一行；P9 review 收尾一行中 `confirm-workspace-member-remove.tsx` 的去处（收尾）。
- **交接**：第 5 节各行照录进 M4–M8 的交接和收尾（13.2）；第 3 节中裁定的结果写进 P11 的评审。
- **放不下的关闭条件**：没有。完成线的"各包的上限已调低"按第 3 节第 9 条的裁定读；"`--rows M3` 没有输出，或剩下的每一行写进 review"：附录 A.9 逐行写明，写进评审是控制者的。
- **限制**：每次挂载都重取，聚焦时不取（P8a 的 F-1）；删除确认框的 Tab 循环到收尾的重写之前不核对（第 3 节第 3 条）；访客经 PAT 离开工作区，直到收尾的入口（第 3 节第 5 条）。

## 附录 A：原型验证记录（2026-10-11）

### A.1 方法与门禁

- 原型 `$M3TMP/p11proto` 是 `7d6a416b` 的 `git archive` 副本（`$M3TMP/p11tools/base-7d6a416b.tar`，`pnpm install --frozen-lockfile` 之后）。每个 Task 由一个脚本从上一个快照写出（`$M3TMP/p11tools/edit/T<n>.py`，共用 `sub.py`），照仓库的格式化（oxfmt）排版，`tsc` 通过之后存一份快照（`$M3TMP/p11snap/T1`…`T10`；`T0` 是 `7d6a416b`）。plan 的块由快照之间的差异写出（`mkblocks.py`：10 个 Task，0 个问题；它把每个 Task 的块依次应用到上一个快照，与那个 Task 的快照逐字节比较）。代码或测试改过时，从改动的那个 Task 起重新写出之后的每个快照（`chain.sh`、`rebuildgates.sh`），再重新写出块。原型中途把原来的第一个 Task 分成 Task 1、2（第 3 节第 1 条），之后的脚本、快照和变异的编号各加一。
- 每个 Task 的快照上 `make lint-web`、`make knip`、`make test-web` 都通过（`rebuildgates.sh 1 10`，`logs/rebuildgates.log`）；有页面或挂载路径改动的 Task 跑了它的故事和 `make e2e`；最后一次逐 Task 复现在每个 Task 的树上再跑全部门禁（A.10）。`7d6a416b` 上：`make lint-web`（69 条规则、3 个例外）、`make knip`、`make test-web` 通过，`make e2e` 106 个中 105 个通过（S3 在副本中失败，F4）。
- 最终原型上：`make lint-web` 通过（关键词守卫 69 条规则、3 个例外、没有命中）；`make knip` 通过；`make test-web` 通过（web 应用 106 个测试文件、964 个测试；`7d6a416b` 上是 95 个、849 个）；`make e2e`（含 `make build`）110 个中 109 个通过（S3，F4）。生成的文件与 `make gen` 的输出相同（0 处差异，`make gen-check` 的核对）。没有 Go 和接口描述的改动，`make lint-go`、`make test` 不需要。
- 端到端的反复：P6、P7、P9、W11、S2 的页面版本和 P8（`dragOnto` 移进 fixture 之后）各 3 次，19 个测试 57 次，都通过（`logs/repeat-final3.log`）；P7 的页面版本另在 Task 5 的树上反复 8 次（标签表单的焦点，第 6 节）。本稿的修订之后（拖动把手、组的展开按钮、标签的表单、API 令牌的弹窗）再跑 A11、P6、P7、W11、S2、P8 各 3 次，22 个测试 66 次，都通过（`logs/repeat-amend3.log`；P9 和它的页面不变）。
- 修复轮之后（最终评审的 F-1–F-13，代码在 `0d2061f5`）在 worktree 中：`make lint-web`（69 条规则、3 个例外；上限 web 303、editor 64、ui 15、propel 11 都等于条数；手改的文件 0 条，除第 3 节第 3、8 条的）、`make knip`、`make test-web`（web 应用 106 个测试文件、965 个测试）、`make e2e`（110 个）都通过；P6、P7、P9、W11、S2、A11 的页面版本和 P8 各 3 次，23 个测试 69 次，都通过。下表是各 Task 提交时的数（任务评审的修正轮在内；Task 5 的提交照裁定 T4-c 给行内表单的 vitest 加了一个）。

| Task | 改到的文件 | web 的测试文件 / 测试 | 端到端 |
|---|---|---|---|
| `7d6a416b` | – | 95 / 849 | 106 |
| 1 | 6 | 95 / 849 | 106 |
| 2 | 20 | 97 / 888 | 106 |
| 3 | 21 | 98 / 893 | 108 |
| 4 | 11 | 99 / 921 | 108 |
| 5 | 32 | 101 / 932 | 109 |
| 6 | 6 | 101 / 932 | 110 |
| 7 | 32 | 106 / 965 | 110 |
| 8 | 17 | 106 / 965 | 110 |
| 9 | 29 | 106 / 965 | 110 |
| 10 | 4 | 106 / 965 | 110 |

（测试数按 vitest 自己的计数，`it.each` 的每一行算一个（`vitestcounts.py`），只算 web 应用。一个文件可以由几个 Task 改到，所以改到的文件之和大于 141；Task 10 另改了本 spec。端到端的数是 `make e2e` 的测试数，副本中其中的 S3 失败（F4）。）

### A.2 变异

`mutants_p11.py`：72 个变异，每个只改一处或几处。`mut.py` 在最终原型的副本上、在它写的每个检查上各跑一次：72 个都被发现，没有存活的；每个写的每个检查都让它失败（`$M3TMP/p11tools/mut-results.json`，日志 `mut-logs/`）。每个变异另在它自己的 Task 的快照上跑过（`mutpertask.py`，端到端在内，`mut-results-pertask.json`）：62 个在自己的 Task 的树上就被发现；10 个只在之后的 Task 起，因为发现它们的是之后的 Task 才加的页面版本：`T1.2`、`T1.3`、`T2.7`、`T2.8`、`T2.9`、`T2.11`、`T2.12` 由 Task 3 的 P6 的页面版本，`T4.8`–`T4.10` 由 Task 5 的 P7 的页面版本；各在那个 Task 的树上被发现（`mutfrom.py`，`mut-results-from.json`；plan 的表中标出）。`T2.6`、`T2.10`、`T4.6`、`T4.7`、`T4.11` 在自己的树上已由 vitest 发现，它们的端到端一层同样从 Task 3、5 起（`T4.11` 原来只有端到端，预检的 L6 之后行内表单的 vitest 也发现它）。每个变异要改的文字在它的 Task 的快照上和最终原型上都恰好出现一次（`mutchecksnap.py`：0 个不适用；`T2.10`、`T3.2`、`T3.3`、`T3.7`、`T3.8`、`T5.2`、`T5.11`、`T5.12`、`T6.1`、`T6.2`、`T7.1` 要改的文字在更早的快照上已有，不影响结果）。"静态"是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的范围）、关键词守卫，以及 `make knip`。

**任务评审加的 26 行**（`mutants_review_p11.py`，plan 的表中与原有的行按编号排在一起）：任务评审的修正轮和裁定加的 `T2.14`–`T2.16`、`T3.11`–`T3.13`、`T4.13`–`T4.17` 和 `T4.14b`、`T5.18`–`T5.24`、`T7.10`–`T7.12`，以及 Task 6 的探查 `P6.1`–`P6.4`（裁定 T6-b），各在它的轮次的树上被发现（`$M3TMP/p11mut/t3`、`t5`、`t6`、`t7`）；其中 10 行由之后的 Task 的检查才发现（`T2.14`–`T2.16` 由 Task 3 的 P6，`T3.13`、`T4.13`–`T4.17`、`T4.14b` 由 Task 5 的检查），plan 的表中标出。最终评审的修复轮移动或改写了检查（F-1–F-5，F-7 的三个测试）之后，检查被移动或改写的 71 行（plan 的、评审加的，和 Task 6、Task 7 的探查）在修复之后的树上再跑一遍（`$M3TMP/p11mut/fw`）：每一行在它写的每个端到端和改写过的 vitest、关键词守卫、oxlint 上都被发现，`P6.3`（卡片显示看的人的邮箱）由 `expectNoAddress` 发现；Task 7 的探查 `P7.1`、`P7.2` 的 vitest 一层照旧存活，与 Task 7 的轮次相同（它们只由 A11 发现）。合计 98 行，没有存活的。

按缺陷类别（plan 的 72 个和评审加的 26 行；一个变异可以被几层发现）：

| 类别 | 变异 | 静态 | vitest | 端到端 | 存活 |
|---|---|---|---|---|---|
| W3：会话外的跟进（2.12） | 17 | 0 | 17 | 0 | 0 |
| 权限的门：页面只给 nerve 允许的人（W11，P10 的第四类） | 18 | 3 | 11 | 14 | 0 |
| 键盘：Tab 能到、按次序、有名字、到达时看得见（P10 的第一类） | 22 | 0 | 0 | 22 | 0 |
| 请求在路上时弹窗关得掉、表单不忙，被拒绝时弹窗关上（P9 第一、二类，A-M2） | 15 | 0 | 0 | 15 | 0 |
| 错误的读法：拒绝落在哪里、说什么（P8b 第二类） | 6 | 0 | 5 | 5 | 0 |
| W12：非空断言的范围（2.13） | 6 | 6 | 0 | 0 | 0 |
| 清理：oxlint 的改法、`always-return` 和 `no-unreachable`（附录 A.5、A.6） | 5 | 5 | 0 | 0 | 0 |
| 以项目的 id 读的 getter（第 3 节第 16 条） | 4 | 0 | 4 | 0 | 0 |
| 拖动：只由把手开始，以行此刻的把手登记 | 3 | 1 | 2 | 1 | 0 |
| 请求体：只有表单编辑的字段（P8b 的第一类） | 1 | 0 | 1 | 0 | 0 |
| 接线：控件发出它的修改 | 1 | 0 | 0 | 1 | 0 |
| 合计 | 98 | 15 | 40 | 58 | 0 |

按发现它的层：只有端到端的 47 个、只有 vitest 的 25 个、只有静态的 11 个、vitest 和端到端的 11 个、静态和 vitest 的 4 个。按 Task 的表：Task 1 4 个、2 16 个、3 13 个、4 18 个、5 24 个、6 6 个、7 12 个、8 2 个、9 3 个；Task 10 只改文档，没有变异。

- **W3 的 17 个**：2.12 的每一处"不论会话都跟进"（`T2.1`–`T2.5`、`T4.1`–`T4.5`、`T7.1`–`T7.5`、API 令牌的新建和撤销 `T7.7`、`T7.8`），每个由它自己的 vitest 发现（`lateSettlings`、`lateSettlingsAnswering` 的两种）。会话切换的端到端在 P9 的 W3（M3 设计 9.6）。
- **权限的门的 18 个**：页面读 router store 的地址（`T3.1`、`T5.1`、`T5.3`：关键词规则 `project-address-route` 和 `states-labels.test.tsx`）、根组件按别的 id 读状态（`T3.10`：`states-labels.test.tsx` 的替身只对页面的项目回答，预检的 L5；P6）、访客看得到状态、标签（`T3.2`、`T5.2`）、项目成员得到改它们的控件（`T3.3`、`T5.4`）、组的菜单和把手给每个成员（`T5.5`，`project-setting-label-group.test.tsx` 和 P7）、空状态的按钮给每个成员（`T5.6`）、默认状态提供删除（`T2.10`）、访客的个人主页挂载工作项一栏或算进看得到工作项的人（`T6.1`、`T6.2`，P9 的页面版本：工作项区域、没有 `issues` 的请求）。任务评审加的 5 个：项目成员拖得动状态（`T3.12`，Task 3 评审的 I2：P6 的成员拖动一秒之内不发请求，`sentWithin`）；Task 6 的探查 `P6.1`–`P6.4`（访客的个人主页挂载工作项一栏、显示工作项的标签页、卡片显示看的人的邮箱、访客在自己的主页上看到工作项），由 P9 的页面版本发现（`P6.3` 由 `expectNoAddress`）。
- **键盘的 22 个**：组标题又是 `div`（`T2.12`）、行的控件在指针经过之前 Tab 到不了或 Tab 到达时看不见（`T3.4`、`T3.5`、`T5.10`）、按钮没有名字（`T3.6`、`T5.8`、`T5.9`，标签组的展开按钮 `T5.16`）、表单的按钮有正的 `tabIndex`（`T3.7`、`T5.11`）、拖动把手又是 Tab 停留点（`T5.17`），都由 P6、P7 的页面版本发现（`tabCycle`、按名字找按钮、`opacity`）：服务端渲染看不到焦点和指针。任务评审加的 11 个：组标题总说展开着（`T2.16`）、状态一行的删除按钮留着 `focus:outline-none`（`T3.11`）、标签表单的颜色按钮的轮廓和两个颜色按钮的名字（`T4.13`、`T4.14`、`T4.14b`）、标签一行的菜单的 `outline-none`（`T5.18`）、`Button`、`IconButton` 的轮廓和离开的距离（`T5.20`–`T5.23`）、返回箭头的名字（`T5.24`），都由 P6、P7 的页面版本发现（焦点的 `outline-style`、`outline-offset`，按名字找）。
- **请求在路上和被拒绝时的 15 个**：确认框在请求路上时 Escape 关上、取消可用（`T1.2`、`T1.3`）、状态和标签的表单不忙或取消可用（`T2.7`、`T2.9`、`T4.8`、`T4.9`）、删除状态的确认框被拒绝时关上（`T2.11`）、删除标签的确认框在 nerve 回答之前关上（`T4.10`）、新建 API 令牌的弹窗在请求路上时可以取消（`T7.9`），都由 P6、P7 的页面版本和 A11 扣住请求发现（`sentHeld`、`closedByEscape`、`enabledWithin`、`closedWithin`）：服务端渲染看不到点击之后的状态（P9 的先例）。任务评审加的 6 个：设默认在 nerve 回答之前不忙、按钮仍可用（`T2.14`、`T2.15`，P6）；删除标签的确认框在 nerve 拒绝时关上（`T4.15`，P7）；新建 API 令牌的请求在路上时"Generate"可用、弹窗按 Escape 和点外面关上，撤销的确认框不忙（`T7.10`–`T7.12`，A11）。
- **错误的读法的 6 个**：`formRefusal` 不认已被占用的码（`T1.1`，`project-refusal.test.ts`）、名字已被占用在提示里（`T2.6`、`T4.6`）、表单在字段下什么都不显示（`T2.8` 只有端到端：字段下的错误在提交之后才有；`T4.11` 也由行内表单的 vitest，它记下 react-hook-form 的 `setError`）、第三层按普通的拒绝说（`T4.7`）。
- **W12 的 6 个**：根目录 `.oxlintrc.json` 的范围中 P11 加的每一组各写一个非空断言（`T1.4`、`T2.13`、`T3.9`、`T4.12`、`T5.13`、`T7.6`），由 `check:lint` 发现，在自己的 Task 的树上和最终原型上都是（`mutw12.py`）。
- **清理的 5 个**：`ComboDropDown` 没有 `role`（`T8.1`）、状态组的进度以下标为 key（`T8.2`）、两处 `then()` 的回调不返回（`T9.1` editor、`T9.2` web），都由 oxlint 的上限发现（警告数超过上限）；`return;` 写在回调最后一个语句之前（`T9.3`），由 `no-unreachable` 发现。
- **getter 的 4 个**：按组分时少一个组（`T3.8`）、组下的标签也放在顶层（`T5.12`）、两个 getter 读路由 store 的项目（`T3.13`、`T5.19`，Task 5 评审的 I1：store 的测试再以路由不在的项目读它们），由 store 的测试发现。
- **拖动的 3 个**：标签由整行开始拖动（`T5.7`），由 P7 的页面版本发现（项目成员的行上 `[draggable="true"]` 0 个）；把手又经 ref 读（`T5.14`，原型中的 `T5.ref`，Plane 的写法）、把手照行的渲染存着但不在副作用的依赖里（`T5.15`，也由 oxlint 的 `exhaustive-deps`），由 `label-drag-n-drop-HOC.test.tsx` 发现（第 3 节第 10 条）。
- **请求体的 1 个、接线的 1 个**（Task 4 评审的 M1、M2）：没有颜色的标签改名时发出 `#000`（`T4.16`，行内表单的 vitest）；一行的"移出组"什么都不发（`T4.17`，P7 的页面版本）。
- **W18（每个新的 vitest 都能失败）**：`mut.py` 只跑一个变异写的检查，所以另由 `w18p11.py` 把每个变异对 P11 写或改的 17 个 web 应用的 vitest 文件各跑一次，再对照最终的 vitest 报告：P11 新加或改写的 122 个测试（`it.each` 的每一行算一个）中，plan 的变异让 56 个失败。其余 66 个多是正向的行（"发给 store 的是页面要的""做完时跟进、不提示"、权限表中能改的那几行、M2 账户页面和 API 令牌的弹窗的成功的提示、令牌的下载），另写 32 个探查变异（`mutants_w18_p11.py`，`W18.1`–`W18.32`，只在最终原型上跑，不在 plan 的表中；`probecheck.py` 核对每个要改的文字恰好出现一次）：每个都让它守着的测试失败（`logs/w18-run.log`；`W18.29`–`W18.32` 之后另跑，日志在 `mut-logs-w18/`），122 个测试都至少失败一次。改到的文件中 `7d6a416b` 已有的 53 个测试（状态、标签的 store 的测试改读 getter，新手引导的测试的原有部分）不算。W18 在原型上跑；任务评审之后加的测试（行内表单的没有颜色的改名）由 `T4.16` 发现。
- **只由评审看住的**（没有会失败的检查的性质，不是会话、权限或取数的性质）：删除确认框的 Tab 循环：`AlertModalCore` 的确认按钮的 `tabIndex={1}` 属于 M3 收尾的 `ModalCore` 重写，P6、P7 的页面版本不核对它（第 3 节第 3 条）。标签的拖动把手以状态登记原来在这里（原型中的 `T5.ref`），按裁定 Q8 的 (b) 加了 `label-drag-n-drop-HOC.test.tsx`，`T5.14`（即 `T5.ref`）、`T5.15` 由它发现。

### A.3 W2：挂载清单（真实运行）

S2 的清单在 Task 3、5、6 加入，每次 `make e2e` 都核对（2.11）。另在 `7d6a416b` 和最终原型上各跑一次一次性的清单（`$M3TMP/p11tools/w2/w2-inventory.spec.ts`：五个账户各在自己的浏览器中打开 P11 的五页，记下每页载入时的请求和 400 以上的回答；跑完删除，不在 plan 中）。账户：工作区管理员（创建 Web 的，Web 的管理员）、Web 的管理员（工作区成员）、Web 的成员、Web 的访客（工作区访客）、不是 Web 成员的工作区成员。之前与之后相同（`w2/table.txt`）：

| 页面 | 工作区管理员 | Web 的管理员 | Web 的成员 | Web 的访客 | 不是 Web 成员的 |
|---|---|---|---|---|---|
| Web 的状态设置 | `APP` + `WORKSPACE` + `PROJECT` + `PROJECT_MEMBER` | 同左 | 同左 | 同左（页面说没有权限） | `APP` + `WORKSPACE` + `PROJECT` |
| Web 的标签设置 | 同上 | 同上 | 同上 | 同上 | 同上 |
| 工作区管理员、Web 的成员、访客的个人主页 | `APP` + `WORKSPACE` + Plane 的 `GET /api/workspaces/{slug}/user-issues/{id}/`（404） | 同左 | 同左 | `APP` + `WORKSPACE` | 同左（404） |

- 状态、标签的设置页没有失败的请求；访客取 `PROJECT_MEMBER` 是项目的包装层的（P8b，nerve 答 200）。
- 个人主页：会看工作项一栏的四个账户请求 Plane 的旧接口，nerve 答 404，工作项一栏一直是骨架：设计 3.1 定下的过渡，M4 对接（第 3 节第 6 条）。访客打开谁的主页（自己的在内）都只取应用和工作区的：卡片来自工作区的包装层取的成员列表，工作项区域不挂载。没有别的 `/api/v0` 之外的请求。
- 故事的页面版本都经 `watchPage`：P6、P7 断言记下的失败的请求恰好是它要的那几个（P6 的两次 409，P7 的 409 和 422），P9 断言没有失败的请求、没有含 `issues` 的地址；都没有旧地址的请求、没有页面错误。

### A.4 缺陷类的清扫

- **封闭的请求体**（P8b 的第一类）：P11 的页面发出的每个请求体都由表单编辑的字段构成，类型是生成的 `StateCreate`、`StateUpdate`、`LabelCreate`、`LabelUpdate`（M2 的账户页面照旧是 `UserUpdate`、`ProfileUpdate`、`ApiTokenCreate` 和改密码的请求体），每个由一个测试钉住：`use-state-changes.test.ts`、`use-label-changes.test.ts` 的"发给 store 的是页面要的"；`create-update.test.tsx`（新建发页面的项目和表单的组，修改只发 `StateUpdate` 的字段）、`create-update-label-inline.test.tsx`；P6、P7 的 `sentHeld`、`sentTo`（新建、改色、改名、拖动的请求体逐个比较）；`general/form.test.tsx`（地址不在其中）、`security.test.tsx`、`language-and-timezone-list.test.tsx`、`create-token-modal.test.tsx`。W18 的探查 `W18.1`、`W18.9`（发页面没有要的）、`W18.16`（新建发别的组）、`W18.17`（标签的新建多发 `sort_order`）各被 vitest 发现。
- **错误的读法**（第二类）：P11 改到的页面的每一条出错的路径都经 `errorMessageKey`（`useFormRefusal`、`useRefusalToast`、M2 页面自己的映射），没有吞进 `console.error` 的拒绝，被拒绝的修改结束忙碌。P8b 起这两页的拒绝已经 `useRefusalToast`，P11 改的是拒绝落在哪里（字段下或提示）；删掉的是状态表单的 `catch` 中的 `console.log`（它接不到拒绝）。6 个变异（A.2）。
- **请求体取自所持的值**（第三类）：状态、标签的移动在轮到它时从 nerve 最近的回答算出（P7a、P7b 的 store，P11 不改）；页面自己不算请求体。
- **请求在路上时弹窗关得掉**（P9 的第一类）：状态、标签的删除确认框（`AlertModalCore`，第 3 节第 2 条）；新建 API 令牌的弹窗的取消（第 3 节第 18 条，A11）。3 个变异（`T1.2`、`T1.3`、`T7.9`）；被拒绝时留着（A-M2）：删除状态的原来被拒绝时关上（`T2.11`），删除标签的原来请求在路上时关得掉、回答之后不经会话的核对就关上（`T4.10`；它原来就只在 nerve 删除之后关，预检的 L1）。任务评审之后另有：新建 API 令牌的弹窗在请求路上时 Escape、点外面关上（`T7.11`），撤销的确认框不忙（`T7.12`），删除标签被拒绝时确认框关上（`T4.15`）。
- **`AlertModalCore` 的其余 11 个调用方**（裁定 Q2 的条件；预检逐个核对过，结论相同）：`isSubmitting` 是必传的 prop，11 个都传；每一个在每条被拒绝的路上都把忙碌复位，没有一个在被拒绝之后卡在忙里、再也关不掉；卸载时忙碌随组件消失。不用改调用方；它们看得到的不同（请求在路上时关不掉）来自组件，与状态、标签的删除确认框一起写在前端改动清单的确认框一行（W17，第 3 节第 2 条）。

| 调用方 | M | store 抛出或拒绝 | 提前返回 | 忙碌 |
|---|---|---|---|---|
| `issues/delete-issue-modal.tsx` | M4 | `.catch` 提示，`.finally` 的 `onClose` 复位 | 没有权限：提示，`onClose` 复位之后返回；打开、关上时副作用也复位 | 复位 |
| `issues/workspace-draft/delete-modal.tsx` | M4 | 同上 | 同上 | 复位 |
| `issues/attachment/delete-attachment-modal.tsx` | M5 | `remove` 的 `.finally` 的 `handleClose` 复位（拒绝由 `remove` 提示） | 没有 | 复位 |
| `cycles/delete-modal.tsx` | M6 | `.catch` 提示、`.finally` 关上；外层的 `try` 之后 `setLoader(false)` | 没有迭代时在置忙之前返回 | 复位 |
| `modules/delete-module-modal.tsx` | M6 | `.catch` 提示，`.finally` 的 `handleClose` 复位 | 没有地址时在置忙之前返回 | 复位 |
| `views/delete-view-modal.tsx` | M7 | `catch` 提示，之后复位 | 没有地址时在置忙之前返回 | 复位 |
| `workspace/views/delete-view-modal.tsx` | M7 | `catch` 提示，之后复位、关上 | 没有地址时在置忙之前返回 | 复位 |
| `inbox/modals/delete-issue-modal.tsx` | M7 | `.catch` 提示，`.finally` 的 `handleClose` 复位 | 没有 | 复位 |
| `inbox/modals/decline-issue-modal.tsx` | M7 | `.finally` 复位（拒绝交回调用方的 `onSubmit`） | 没有 | 复位 |
| `web-hooks/delete-webhook-modal.tsx` | M8 | `catch` 提示，之后复位 | 没有地址时在置忙之前返回 | 复位 |
| `api-token/delete-token-modal.tsx` | M2 | `followInSession` 的 `failed` 提示原因，之后复位（Task 7；原来在 `catch` 之后复位） | 没有 | 复位 |

  两个工作项的删除确认框另有一条今天走不到的路：`onSubmit` 原来可以不传，有权限而不传时一直忙，有了 F-1 就再也关不掉；11 个调用处都传。Task 1 把它改为必传（加固，`tsc` 核对）；两个文件各有的一条 `always-return` 在 Task 9 清零，P11 结束时 0 条警告。
- **忙碌在跟进之前结束**（第二类）：两个表单的 `isSubmitting` 等 `useStateChanges`、`useLabelChanges` 交回的 `Promise`，它在跟进做完时兑现；确认框同样。4 个变异（`T2.7`、`T2.9`、`T4.8`、`T4.9`）；任务评审之后另有设默认的 `T2.14`、`T2.15` 和新建 API 令牌的"Generate"的 `T7.10`。
- **包装层之上的状态**（第三类）：没有。表单的值在页面里，换账户时随页面卸载。
- **连按两下**（第四类）：状态表单原来的 `loader` 从不置真，连按两下发两次；现在 `isSubmitting` 时按钮不可用（`T2.7`）。标签的表单、两个确认框同样；API 令牌的新建（表单的 `isSubmitting`）和撤销（确认框的 `loading`）在忙时按钮不可用。
- **键盘**（P10 的第一类）：P6、P7 的页面版本走表单的 Tab 顺序（`tabCycle`）、行的控件由 Tab 到达并显示、按名字找每个按钮；组标题是按钮，标签组的展开按钮以标签的名字为名；拖动把手不是 Tab 停留点（`DragHandle`，第 3 节第 19 条，P7 从一行的删除按钮按 Tab 直接到下一行的菜单）；共用组件的轮廓和名字（第 3 节第 12 条）。22 个变异（任务评审加了 11 个，A.2）。删除确认框的 Tab 循环不核对（第 3 节第 3 条）。
- **输入法**（P10 的第二类）：P11 的表单没有自己的按键处理（`project-states`、`labels` 中没有 `onKeyDown`、`useKeypress`），确认框的 Escape 是 `ModalCore` 的，P10 的 F-3 已让它在组字时不关。
- **地址取自路由**（P10 的第三类）：两页和根组件、列表、拖放组件读路由参数，`allowPermissions` 写明工作区和项目；关键词规则 `project-address-route` 加上这些文件（2.13），`T3.1`、`T5.1`、`T5.3` 让它失败。
- **权限的门**（P10 的第四类；照 nerve 的权限矩阵 `permission_matrix_states_test.go`、`permission_matrix_labels_test.go` 逐个角色）：读状态、标签的是项目的每个有效成员（管理员、成员、访客、是工作区管理员的成员），写的是项目管理员和是工作区管理员的项目成员。页面：能写的人看到状态、标签和控件（`its.admin`、`its.memberAdmin`、`its.guestAdmin`），项目成员只看到状态、标签，访客看到没有权限（比 nerve 少：W11 的页面一半），不是项目成员的由项目的包装层挡住（P8b）。每一格有一个会失败的检查（权限的门一类，A.2；项目成员拖不动状态由 P6 核对，`T3.12`）。
- **被拒绝时弹窗留着**（P10 的 A-M2）：两个删除确认框、两个表单只在成功的跟进中关上（`T2.11`、`T4.10`，P6 中被拒绝的删除；P7 中被拒绝的删除，`T4.15`）。
- **会话外的跟进**（W3）：web 应用中提示、关表单的每一处 P11 的页面都经 `followInSession`（2.12），17 个变异。预检找到 security 页的 API 令牌的两个弹窗不经会话的核对（M1），Task 7 改了它们（`T7.7`、`T7.8`）；再扫一遍账户（个人设置、API 令牌、新手引导、停用和切换账户）、状态、标签的页面上修改之后的提示、字段错误、关弹窗和下载（`followsweep.py` 和逐个 `await`、`.then`）：都经 `followInSession`，停用账户和切换账户的弹窗除外：`deactivateAccount`、`signOut` 交回这一次是否结束了本标签页的会话，弹窗照它跟进（M2 的写法；停用账户的由它自己的 vitest 守着，切换账户的弹窗只在结束了本标签页的会话时关上）。
- **19（每个新的方法有失败测试）**：`useStateChanges`、`useLabelChanges` 的十个修改各有被拒绝的测试；`formRefusal` 的决定由 `project-refusal.test.ts` 的表钉住；两个 getter 没有失败的路径（store 的测试核对它们按项目给出）。
- **36（另一条路径到达同一结果）**：设置页上状态、标签的每个修改只经两个 hook。store 的 `createLabel` 另有 M4 的五个调用方（工作项页的标签选择器），不经 `useLabelChanges`：交给 M4（第 3 节第 13 条）。删除确认框的关不掉在组件里（第 3 节第 2 条），不是两份。
- **45（没有不能失败的断言）**：W18（A.2）：P11 写的 122 个测试都至少被一个变异或探查变异发现；任务评审之后加的一个（行内表单的没有颜色的改名）由 `T4.16` 发现。
- **46（每个等待有期限）**：vitest 中等待页面跟进的是 `pageSettled()`；端到端的等待是 Playwright 的 `expect`、`expect.poll`、`toPass`、`waitForResponse`（`sentTo`、`sentHeld` 的 10 秒）、`dialogs.ts` 的 `enabledWithin`、`closedByEscape`、`closedByBackdropClick`、`closedWithin` 和 `happenedWithin`（`sentWithin`、`downloadedWithin`）的一秒；`dragOnto` 等过渡结束（`transitionsEnded`，有期限），`tabCycle` 按给定的次数，都有期限。
- **50（说明与代码一致）**：每个 Task 结束时，新写或改动的注释、JSDoc 与它描述的代码或测试逐句核对；最终原型上 P11 加或改的注释（`addedcomments.py`，`logs/comments.txt`，60 个文件 614 行）再全部读过一遍（本稿修订之后，修订改到的文件的注释再读一遍；修复轮之后 65 个文件 764 行，`p11tools-fw/comments.txt`，修复轮改到的文件的注释再读一遍）；README、前端改动清单和两份交接的行与最终的代码核对（Task 10）。

### A.5 清理的清扫和 oxlint（7.9，R3）

**C-a（每处改法不改看得到的行为，或是 W17 的一行）**：

| 改法 | Task | 为什么看不出，或 W17 |
|---|---|---|
| `project-states/group-item.tsx`：组标题 `div` → `button`（两条 a11y 警告） | 2 | W17（第 3 节第 12 条"键盘和名字"）：`type="button"`，不提交表单；里面的 `div` 换成 `span`（按钮里不能有块级元素），类名照旧；加状态的按钮移到它旁边（按钮里不能有按钮），位置和样式照旧。P6 的页面版本走它（`T2.12`） |
| `dropdowns/project/base.tsx`：`no-shadow` 3 条 | 8 | 取显示名、图标的函数不再收与外层同名的参数，读的是同一个值，结果相同（`tsc`） |
| 同上：`consistent-function-scoping` | 8 | 画图标的函数移到模块，函数体不变 |
| 同上：`ComboDropDown` 收 `role="presentation"` | 8 | 清掉一条 oxlint 警告（`T8.1`）；辅助技术读到的照旧：三个调用方（工作项的表单（M4），迭代、模块的表单（M6））在桌面上都给根 `tabIndex`，浏览器不理会可以聚焦的元素上的这个角色（Task 8 评审的 m1，裁定 T8-a）；根是没有名字的 Tab 停留点，原来就是，改法在第 5 节收尾的键盘清单。W17 的"看不出的"一行 |
| `use-work-item-filters-config.tsx`：`no-shadow` 3 条 | 8 | 内层变量改名，值不变（`tsc`） |
| `progress-stats/state_group.tsx`：`no-array-index-key` | 8 | key 由下标改为 `group.state`：每组一行、顺序不变，显示相同；在 M6 的页面上（`T8.2`） |
| 35 处 `then()` 的回调加 `return;` | 9 | A.6 |

Task 9 另把根目录 `.oxlintrc.json` 的 `no-unreachable` 设为 error（预检的 L9）：`return;` 写在回调最后一个语句之前时，那个语句走不到，`check:lint` 失败（`T9.3`）；今天全仓 0 处，上限不变，是一道检查，不改任何代码。

brief 点名的三种：`.then` 链中新浮出或被藏起的拒绝：没有（A.6）；`div` 变成按钮：只有 `group-item.tsx`（上表第一行）；`exhaustive-deps` 改副作用运行的时机：P11 没有这样的改法（`issue-activity/helper.tsx` 的一条留着，第 3 节第 8 条）。

**C-b（删掉的死成员、死 prop 在哪里都没有读者）**：`--rows M3` 的产品代码行中 26 行删去（A.9），另删 `@nerve/types` 的 `state.ts`（`TStateOperationsCallbacks`，连同 `index.ts` 的再导出）和 `labels` 的 `TLabelOperationsCallbacks`；评审找到、`deadsym.mjs` 看不到的（每个调用方都传、没有读者）在修复轮删除：`dashed-circle.tsx` 的 `radius`（F-6），状态的组列表一层层传到每一行的 `groupedStates`（F-1），标签的行内表单的 `labelForm`、`setLabelForm`、`forwardRef` 和列表的 `scrollToRef`（F-2），以及状态的根组件与项目的包装层重复的取数（F-3）。每个名字在仓库中逐个搜过：没有别的读者，测试在内（状态、标签的表单的测试由 Task 2、4 改写）；都不是文案的键、`?url` 的资源或包的再导出（`state.ts` 的再导出随文件删除）；`tsc`、knip 在每个 Task 之后通过；没有新出现的行。没有调用方传的 prop 原来取默认值，组件现在把原来的默认值写在用到它的地方，显示的照旧（项目下拉框的按钮、箭头、`renderByDefault`，成员头像的 `size`，`dashed-circle` 的段数等）。

预检的 H1：原稿还删了工作区设置侧边栏（`settings/workspace/sidebar/root.tsx`）的 `className`，它有读者：移动端的设置导航把这个组件作为值交给 `hamburgerContent`（`settings/mobile/nav.tsx:38`，类型 `ComponentType<{ className?: string; isMobile?: boolean }>`），再以 `className` 渲染它；`deadsym.mjs` 看不到经组件类型的变量的读，删了它，移动端的设置菜单就没有了高度的限制和边框。本稿留下它（A.9 的一行）。其余 25 处删除照同一形状逐个扫过（`$M3TMP/p11tools/sweep_h1.py`）：删去的 prop 所在的组件或它的类型作为值出现的每一处（赋给变量或 prop、`component=`、`as=`、渲染 prop、组件的表、`ComponentType`、`ComponentProps`），删去的名字作为字符串出现的每一处，以及把 props 展开交出去的每一处。结果：组件作为值交出去的只有工作区设置的侧边栏这一个；名字以字符串出现的只有 `Pick<IMemberRootStore, "memberMap">`（根 store 的 `memberMap`，是另一个成员）；展开的两处都不带删去的名字（`StateItemTitle` 收 `{...commonStateItemListProps}`，其中没有 `shouldShowDescription`；`ProjectDropdownBase` 收 `{...props}`，没有调用方传删去的 prop）；`iconName` 所在的成员列的读者 `member-header-column` 读的是别的字段。没有第二处。

**C-c（关键词守卫的最终状态）**：69 条规则、3 个例外，没有 `until: "M3"`：

| 规则 | 文件 | 数 | `until` |
|---|---|---|---|
| `analytics` | `web/apps/web/core/services/cycle.service.ts`（Plane 的迭代进度接口） | 1 | M6 |
| `brand` | `pnpm-workspace.yaml` 的来源说明 | 3 | M9 |
| `plane-workspace-urls` | `core/services/project/project.service.ts` 的 `/user-properties/`（`getProjectUserProperties`、`updateProjectUserProperties`） | 2 | M4 |

`project-address-route` 加上状态设置页、`project-states/root.tsx`（Task 3）和标签设置页、`labels/project-setting-label-list.tsx`、`labels/label-drag-n-drop-HOC.tsx`（Task 5）。旧 `ProjectService` 只有 7.3 的三个方法（`getProjectUserProperties`、`updateProjectUserProperties`、`projectIssuesSearch`），P8b 起就是这样，P11 只核对。

**oxlint**：

- 上限（各包的上限要求警告数等于上限；`oxall.py`、`finalmeasure.sh`）：

| 包 | `7d6a416b` | 最终 | 变化 |
|---|---|---|---|
| web 应用 | 348 | 303 | 346（Task 2）→ 337（Task 8）→ 303（Task 9） |
| `@nerve/editor` | 65 | 64 | Task 9 |
| `@nerve/ui` | 15 | 15 | `alert-modal.tsx` 的一条留着（第 3 节第 3 条） |
| `@nerve/propel` | 11 | 11 | P11 改到的 6 个文件原来 0 条 |
| `@nerve/utils` | 7 | 7 | P11 没有改到 |
| `@nerve/hooks` | 3 | 3 | P11 没有改到 |
| `@nerve/constants` | 1 | 1 | `members.ts` 原来 0 条 |
| `@nerve/types`、`@nerve/i18n` 等 | 0 | 0 | |

- plan 改到的 web 的 105 个 TS 文件（修复轮之后重新数，`p11tools-fw/oxfiles.py`；任务评审和修复轮加的 10 个在 `7d6a416b` 上已是 0 条）：`7d6a416b` 上共 61 条（其中 `always-return` 35 条），最终 15 条：`alert-modal.tsx` 1 条（`tabindex-no-positive`，第 3 节第 3 条）和 Task 9 的五个文件的 14 条（`cycles/modal.tsx` 1、`issue-activity/helper.tsx` 2、`module-card-item.tsx` 5、`snooze/modal.tsx` 5、`webhook.store.ts` 1，第 3 节第 8 条）。其余 99 个是 0 条（两个工作项的删除确认框各 1 → 0，Task 1 的加固之后）。
- P8a、P8b 留下的只经机械步骤到达的文件：现在的 6 个文件 15 条（`create-webhook-modal.tsx` 1、`use-work-item-filters-config.tsx` 3、`state_group.tsx` 1、`dropdowns/project/base.tsx` 5、`work-item/commands.ts` 3、`group-item.tsx` 2）→ 0。P8a 列表的其余文件（成员下拉框、`member-options.tsx`、导航的命令、移出的确认框、成员的默认值）在 `7d6a416b` 上已是 0 条（P9、P10 手改）。
- 抑制：181 → 181，没有新的（`suppressions.py`）。
- 最终的 web 应用按规则：`no-shadow` 128、`no-static-element-interactions` 60、`click-events-have-key-events` 51、`exhaustive-deps` 30、`no-array-index-key` 21，其余每条规则 16 条以下；`always-return` 0（`logs/finalmeasure.log`）。

### A.6 `promise(always-return)`

- 之前 35 处：web 应用 34 处在 25 个文件中，`@nerve/editor` 1 处（`helpers/floating-ui.ts`）；之后 0 处，全仓。一次性的脚本（`$M3TMP/p11tools/alwaysreturn.mjs`，不在 plan 中）读 oxlint 的输出，在每个被指出的回调的最后一个语句之后写 `return;`，再列出这一处的拒绝由谁处理（`logs/alwaysreturn-sites.tsv`）；plan 照它的结果写成 Task 9 的块，逐处可读。
- 为什么不改行为：回调原来走到末尾结束，完成值是 `undefined`；`return;` 的完成值也是 `undefined`，`then()` 给出的 promise 以同样的值、在同样的时刻兑现（没有多出的 `await` 或 promise）。拒绝的路径一处都不变：不加也不删 `catch`。oxlint 接受 `return;`（原型上的探查：`return;`、`return undefined;`、表达式体都通过这条规则）。
- 每一处的拒绝：

| 处理 | 处数 | 位置 |
|---|---|---|
| 链的后面有 `.catch`，或交给 `setPromiseToast` | 23 | 链的后面有 `.catch` 的 21 处：迭代的删除、创建、修改、恢复、转移；收集箱、工作项、草稿的删除；模块的删除和两处修改；命令面板的六处复制（其他命令 1、迭代 1、模块 1、工作项 3）；Webhook 的创建和重新生成密钥；视图的修改；editor 的浮层定位。交给 `setPromiseToast` 的 2 处：`module-card-item.tsx`、`module-list-item-action.tsx` 的加入收藏（按钮的处理函数；`setPromiseToast` 的 `promise()` 接上 `.then().catch()`，错误提示就是拒绝的处理，它交回的 promise 再次拒绝时由 `void toasted.catch(() => undefined)` 接住，propel 的 `toast/toast.tsx`） |
| 交回调用方，由它处理 | 6 | `cycles/modal.tsx` 的日期检查和创建、工作项预览的删除、通知的暂缓、`global-view.store.ts`、`webhook.store.ts` 的删除 |
| 没有处理 | 6 | `entry.client.tsx`：`initPromise` 的拒绝在 `then` 之前已由 `.catch(() => {})` 吞下，`then` 之后没有能拒绝的；其余 5 处之前之后同样没有处理：`issue-activity/helper.tsx` 的评论链接、`peek-overview/header.tsx` 的工作项链接的复制（M4），迭代、模块的复制链接（`cycles/quick-actions.tsx`、`modules/quick-actions.tsx`，M6：交给菜单的 `action`，它是 `() => void`，丢下 promise）和默认视图的复制链接（`workspace/views/default-view-quick-action.tsx`，M7，同样交给菜单的 `action`）（预检的 L4；第 5 节） |

- 变异 `T9.1`、`T9.2`（去掉一处 `return;`）让 oxlint 的上限失败；`T9.3`（`return;` 移到回调最后一个语句之前）让 `no-unreachable` 失败。

### A.7 W12：类型断言；文案

- `w12.cjs`（TypeScript 的语法树，`7d6a416b` 对修复轮之后的代码，`p11snapfw` 的 `T0`、`T10`）：P11 改到的 88 个 web 源文件（测试除外）中，`as`（不含 `as const`）26 → 26、`any` 7 → 7、非空断言 2 → 2；剩下的都是 Plane 原有、P11 没有改的那几行。`castsall.py` 对整个 Phase 的 TS 文件（测试和 e2e 在内）核对：没有新的 `as`、`any`、`!` 或 `oxlint-disable`（它列出的都是字符串和注释里的英文 "as"、`disabled` 和 `!` 的取反，以及 `use-work-item-filters-config.tsx` 原有的 `as Project[]`：Task 8 只给那一行的回调参数改名；修复轮之后另有 `dialogs.ts` 的 `import { setTimeout as sleep }`（导入的别名），和 A11 改写的两行各带的 `as components["schemas"]["ApiTokenCreated"]`：`Response.json()` 的回答的类型，M2 的写法，文件中这样的 5 处之前之后相同，两处随 `downloadedWithin` 改写）。非空断言另有 oxlint 看住（2.13）。
- 没有手写的、重述契约的类型：表单的值是 `TStateFormData`（`Omit<StateCreate, "group">`）、标签的表单是 `LabelCreate` 的字段；`FormRefusal<F>` 是表单自己的决定。knip 没有未使用的导出。
- 文案（`i18nadded.py`，两种语言相同，`check:sync` 核对 `zh-CN` 的键相同）：加 2 条（`project_settings.labels.parent_not_allowed`，Task 4，第 3 节第 4 条；`common.color`，Task 5，两个颜色按钮的名字）、删 0 条、改写 0 条；`i18norphans.py` 在最终原型上 0 条。

### A.8 规模和 Phase 的大小

- **plan** 共 8,892 行（修复轮由提交的代码重新写出之后，附录 A.10；写 plan 时 7,485 行）：Task 1 之前（约束、文件结构）110 行；各 Task 一节：

| Task | plan 行数 | 其中代码块 | 文件 |
|---|---|---|---|
| 1 | 329 | 254 | 6 |
| 2 | 1,461 | 1,341 | 20 |
| 3 | 1,107 | 1,023 | 21 |
| 4 | 854 | 783 | 11 |
| 5 | 1,674 | 1,567 | 32 |
| 6 | 277 | 213 | 6 |
| 7 | 1,832 | 1,713 | 32 |
| 8 | 632 | 577 | 17 |
| 9 | 450 | 344 | 29 |
| 10 | 136 | 72 | 4 |

- **写 plan 之前的核对**（brief）：设计的 9 个任务加上交接来的（M2 的账户页面、个人主页的页面版本、P8a 和 P8b 的只经机械步骤到达的文件），估出约 11–12 个 Task、约 11,000–12,500 行，每个在约 1,500 行之内：在约 16 个、约 1,500 行之内，不拆。原型中实测：10 个 Task、7,485 行（`always-return` 和关键词守卫的核对比估的小；预检之后的修订加了约 770 行，主要是 API 令牌的两个弹窗和它们的测试、拖动把手的测试）；原来的第一个 Task 量出 1,458 行块，加上说明会超过约 1,500 行，所以分成 Task 1、2（第 3 节第 1 条）；最长的 Task 2 是 1,458 行。修复轮之后：Task 5、7 超过约 1,500 行（1,674、1,832 行，第 3 节第 1 条）。
- **文件的行数**（约 400 行；修复轮之后重新数，`p11tools-fw` 的 `sizes.py`、`sizes400.py`）：P11 写的文件中最长的是 `s2-web-app.spec.ts` 356 → 375 行、`a11-api-tokens.spec.ts` 344 → 370 行、`p7-labels-page.spec.ts` 296 行、`p6-states-page.spec.ts` 289 行、`use-state-changes.test.ts` 175 行、`dialogs.ts` 163 行、`use-label-changes.test.ts` 160 行；`settings-pages.ts` 387 → 304 行（弹窗的检查移进 `dialogs.ts`），只改导入的 `w1-create-workspace.spec.ts` 403 → 396 行；改到的 Plane 文件中最长的是 `use-work-item-filters-config.tsx` 394 → 391 行、`workspace-member.store.ts` 367 → 359 行。超过约 400 行的都是原来就超过的：变长的 `work-item/commands.ts` 415 → 418 行、`tools/keywords.json` 2,387 → 2,390 行、两种语言的 `common.json` 527 → 528 行、`e2e/fixtures/assert/identity.ts` 472 → 485 行，行数不变的 `@nerve/ui` 的 `custom-menu.tsx` 492 行（第 3 节第 14 条）。
- **变长的 Plane 文件**（brief：只为使用方改到的不变长；修复轮之后重新数，`p11tools-fw/planegrow.py`）：41 个变长，都是本 Phase 的对象（行为改在它们里面）：Task 9 的 26 个（每处 `return;` 一行）；状态的表单 `form.tsx` 125 → 143（react-hook-form、字段下的错误、忙碌，颜色按钮的名字）、`label-item-block.tsx` 112 → 119（名字、焦点时显示）、标签设置页 65 → 71、状态设置页 59 → 63（路由参数、写明工作区和项目）、`security.tsx` 284 → 287、`general/form.tsx` 279 → 280、`start-of-week-preference.tsx` 72 → 73、`create-token-modal.tsx` 107 → 115、`delete-token-modal.tsx` 65 → 69（`followInSession`）、`drag-handle.tsx` 43 → 47（只给指针用，和它的说明）、`dashed-circle.tsx` 46 → 47（段数写成常量和它的说明）、`alert-modal.tsx` 102 → 103（F-1）、`popover.tsx` 86 → 88 和 `popovers/types.ts` 37 → 39（`buttonAriaLabel` 和它的说明）、项目设置侧边栏的 `header.tsx` 63 → 64（返回箭头的名字）。其余有手改的 Plane 文件变短或不变（例如 `state-item.tsx` 157 → 115、`root.tsx` 71 → 39、`create-update-label-inline.tsx` 211 → 164、`project-setting-label-list.tsx` 170 → 168、`dropdowns/project/base.tsx` 293 → 267、`invite-modal/actions.tsx` 72 → 48；propel 的 `Button`、`IconButton` 的 helper 和 `custom-menu.tsx` 行数不变）。

### A.9 `domains.mjs --rows M3`：之前和之后

`$M3TMP/p11tools/dead/run.sh`（M1 收尾的 plan 附录 A 的 `deadsym.mjs`、`domains.mjs`、`lib/program.mjs`；副本不是 git 仓库，经 `fakegit` 列出文件）。`deadsym`：死的 prop 382 → 362、成员 935 → 975（新的测试的 `it.each` 的列和替身；修复轮之前是 981，三个测试的替身以 `KeptForm` 为类型之后少了它们重述的 6 个，修复轮的 F-7，`p11mut/fw/dead`）、导出 2 → 2。`--rows M3` 修复轮之后照旧是下表的 218 行。`--rows M3`：

| | `7d6a416b` | 最终 |
|---|---|---|
| 测试文件中的行（`it.each` 的列、`vi.hoisted` 的替身，测试自己读它们） | 96 | 123 |
| 按路径误归到 M3 的 41 行（第 5 节） | 41 | 41 |
| 产品代码的行 | 80 | 54 |
| 合计 | 217 | 218 |

- **删去的 26 行**（`deadsplit.py`；没有新出现的）：项目下拉框的 `button`、`currentProjectId`、`dropdownArrow`、`dropdownArrowClassName`、`onClose`、`renderByDefault`；邀请弹窗的按钮行的 `addMoreButtonText`、`cancelButtonText`、`className`、`isInviteDisabled`、`submitButtonText`；`group-list.tsx` 的 `groupListClassName`、`groupItemClassName`、`stateItemClassName`；归档页头的 `href` 和图标类型中与 `SVGAttributes` 重复的 `className`；工作区成员 store 的 `memberMap`（2 行）；成员头像的 `size`、状态选项的 `query`、`label-item-block.tsx` 的 `draggable`、命令面板成员菜单的 `heading`、`state-item-title.tsx` 的 `shouldShowDescription`、创建项目的 `handleFormOnChange`、`members.ts` 的 `iconName`、`dashed-circle.tsx` 的 `totalSegments`。
- **留下的 54 行**，每一行有读者或属于之后的 M：

| 文件 | 行 | 名字 | 谁读或传它 |
|---|---|---|---|
| `dropdowns/member/base.tsx:125-126` | 6 | `multiple`、`value`、`onChange`（两种选择的联合） | `:170` 把它们展开（`...`）交给选择；`domains.mjs` 看不到展开中的读 |
| `project/member-select.tsx:62-64` | 3 | `value`、`query`、`content` | 选项的形状，由 `CustomSearchSelect` 读 |
| `settings/workspace/sidebar/root.tsx:15` | 1 | `className` | 移动端的设置导航把组件作为值交给 `hamburgerContent`（`settings/(workspace)/layout.tsx:41`），在 `settings/mobile/nav.tsx:38` 以 `className` 渲染它；`domains.mjs` 看不到经组件类型的变量的读（预检的 H1，A.5 的 C-b） |
| `workspace/sidebar/workspace-menu-root.tsx:79` | 2 | `open`、`close` | Headless UI 的 `Menu` 的渲染参数 |
| `core/services/workspace.service.ts:21-23` | 3 | `project_id`、`search`、`workspace_search` | 工作项搜索的参数（M4，P8a 已记） |
| `core/store/project/project.store.ts:195` | 1 | `answeredAt` | `maxBy` 以字符串 `"answeredAt"` 读它 |
| `@nerve/constants` 的 `members.ts:22-26` | 5 | `full_name`、`display_name`、`email`、`joining_date`、`role` | `IProjectMemberDisplayProperties` 的键，经 `keyof` 读 |
| `@nerve/types` 的 `project/projects.ts:16-24` | 9 | 工作项搜索的参数 | M4（P8a 已记） |
| `@nerve/types` 的 `workspace.ts:14-49` | 24 | 工作区搜索的结果的类型 | M4（P8a 已记） |

- **按路径误归的 41 行**（设计第 1 节按路径归到 M3，实际属于别的 M；第 5 节各 M 一行）：`@nerve/types` 的 `workspace-notifications.ts` 17 行（M7）、`workspace-draft-issues/base.ts` 16 行（M4）、`progress-stats/label.tsx` 5 行和 `state_group.tsx` 3 行（M6；`state_group.tsx` 第 3 节第 17 条）。之前之后相同。

### A.10 逐 Task 复现

`replay.py` 在 `7d6a416b` 的 `git archive` 副本上（`base-7d6a416b.tar`，`pnpm install --frozen-lockfile` 之后，`$M3TMP/p11replay`），照 plan 逐个 Task 执行：写入块（`planapply.mjs`），核对 Go 的模块文件不变、`pnpm-lock.yaml` 与那个 Task 的快照相同，再执行这个 Task 的每个 `Run:`（make 的目标照写；Task 10 的 `node -e`、`grep -nE` 在 bash 中照 plan 的原文运行）。副本不是 git 仓库，所以 `make lint-web` 的关键词守卫经 `kwreal.py` 运行仓库自己的 `tools/keywords.mjs`（`fakegit` 照 `git ls-files` 列出副本的文件；规则的样例、命中和例外都核对），`make e2e` 只有 S3 因为读不到提交而失败（F4），由脚本单独认出。

| Task | 写入的文件 | 检查 |
|---|---|---|
| 1 | 6 | `pnpm install --frozen-lockfile` 通过；`make lint-web`（69 条规则、3 个例外、没有命中）、`make knip`、`make test-web` 通过；`make e2e` 105 个通过（S3 除外） |
| 2–5 | 20、16、11、18 | 同 Task 1，`make e2e` 105、107、107、108 个通过 |
| 6–9 | 2、16、14、29 | 同 Task 1，`make e2e` 109 个通过 |
| 10 | 4 | 关键词守卫的例外：`69 [["analytics","M6",1],["brand","M9",3],["plane-workspace-urls","M4",2]]`；旧 `ProjectService` 的方法：`getProjectUserProperties`、`updateProjectUserProperties`、`projectIssuesSearch`；其余同 Task 9 |

这一次复现用的 plan 是写 plan 时与本 spec 同一个提交中的那一版（7,485 行，SHA-256 `3a6736d7e08934e5…`），10 个 Task 都通过（`$M3TMP/p11tools/replay-logs/summary.txt`）。最后的副本与原型逐文件相同：`treediff.mjs p11replay p11snap/T10`，3,231 个文件对 3,231 个，0 处差异（本提交的文档：spec、plan 和 M3 设计的第 15 节不在副本和快照中）。

**最终评审的修复轮**（F-1–F-13，`$M3TMP/p11tools-fw`）：任务评审的修正轮和最终评审的修复轮改了代码，plan 的块不再是提交的代码的，所以由提交的代码重新写出。每个 Task 的树取完成它的提交（`git archive`，P11 改到的 141 个文件；`T0` 是 `7d6a416b`），修复轮的改动并进最后一个改到那些文件的 Task：F-1、F-3 → Task 3，F-2 → Task 5，F-5 → Task 6，F-4、F-7 → Task 7，F-6 → Task 8，F-8、F-9 的前端改动清单 → Task 10；之后的 Task 还改的 `states-labels.test.tsx` 在 Task 3、4 的版本中同样去掉两个替身（`build_snaps.py`，0 个问题）。`mkblocks.py` 由这些树写出块（0 个问题），`compose_plan.py` 拼出 plan（8,892 行，评审加的变异在各 Task 的表中）。核对：`planapply.mjs check` 从 `T0` 起 440 个块都恰好应用；`replayblocks.sh` 逐 Task 应用之后与那个 Task 的树逐字节相同，Task 10 之后与修复轮之后的代码逐文件相同。块与原来相同的只有 Task 2、4、9 的。修复轮没有再在副本中逐 Task 跑门禁（brief 不要求完整的复现）；修复轮之后的代码上的门禁和反复见 A.1。修复轮另改了 M3 设计的一句（第 1300 行：M3 自己的 167 行由 P11 删去没有读者的 26 行，其余每一行有读者），它与本 spec、plan 在同一个提交里，不属于任何 Task。
