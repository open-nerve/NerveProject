# M3/P10 项目的页面：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M3/P10 `web-project-pages` |
| 日期 | 2026-10-10 |
| 状态 | 进行中：第 3 节由控制者的裁定 R1–R7 和预检的发现（0 高、5 中、10 低）定下，修订一轮，各条的落点在第 3 节的"预检之后"；控制者的浏览器核对 C6–C8 和评审在 plan 的 Task 之后 |
| 上级文档 | [M3 设计文档](../M3-design.md) 第 2（P1–P5、P8、S2）、3.4、3.5、3.18、3.19、3.20、7.1、7.6、7.7、7.8、8.5、9.5、9.6、9.7、12（P10）、13.1 节；[总体设计](../../v0-design.md) 7.7 |
| 前置交接 | [P9 spec](P9-web-workspace-pages.md) 第 5 节 P10 一行；[P9 review](../reviews/P9-web-workspace-pages-review.md) 第 6 节 P10 一行；[P8b spec](P8b-web-project-data.md) 第 5 节 P10 一行；[P8b review](../reviews/P8b-web-project-data-review.md) 第 6 节 P10 一行（F-2、F-3、T4-b、T5-a）；[P8a spec](P8a-web-workspace-data.md) 第 5 节 P10 一行；[M2 收尾交接](../handoffs/M2-closeout.md) 第 10、11、14 节；[M1-P4 交接](../handoffs/M1-P4-router-native.md)；[P4a review](../reviews/P4a-projects-review.md) 的 P10 一项；P4a、P4b、P5a 的 spec 和 [P5b spec](P5b-project-memberships.md) 第 5 节、[P5b review](../reviews/P5b-project-memberships-review.md) 第 6 节中 P10 的行；[Codex 设计评审](../reviews/M3-design-codex-adversarial-review.md) 4.3、5.3 |
| 计划 | [P10 plan](../plans/P10-web-project-pages.md) |

本 spec 只写 M3 设计交给 P10 决定的东西：名字、签名、文件的位置、测试名，以及原型证明了什么。规则本身以 M3 设计和总体设计 7.7 为准，这里引用节号，不重述。P10 依赖 P8a、P8b、P9（`6c1a090a` 的 `main`），不改契约、Go 代码和生成的文件。

P10 的每一处页面改动都按安全测试看待（brief）：一个角色不能发的请求在页面上是 403；换账户之后才落地的写以错误的身份写入；页面不能显示角色读不到的东西（看得到而不是成员的公开项目、访客、不是项目管理员的成员）。所以附录 A 的变异逐个核对：页面发出项目一侧的修改之后的跳转、提示和界面变化只在发出它的会话里进行（2.13 的每一处由它自己的 vitest 守着）；页面只取它的角色能读的（不是成员的人和已归档的项目只读项目本身，S2 和 P2 的页面版本，2.12）；页面只提供 nerve 允许调用者做的（角色、移出、添加，3.5）；每个请求体只有表单编辑的字段，类型来自生成的客户端，数字就是数字；每个错误经 `errorMessageKey` 显示；请求在路上时弹窗关不掉，按钮忙到跟进做完；被拒绝时弹窗留着，只在成功的跟进中关上。只有评审才能发现的会话、权限或取数的性质算缺口：没有（106 个变异都被发现；只在之后的 Task 起才被发现的三个，`T2.8`、`T2.9`、`T2.14`，是 Tab 的停留点，由之后的 Task 的页面版本发现，各在那个 Task 的树上核对过）；只由评审看住的是 P10 的页面走不到的几处（成员下拉框对 M4–M6 的调用方的约定中要指针、手机或窗口边缘的，Headless UI 自己做的点击，面包屑的最后一节），都不是会话、权限或取数的性质（附录 A.2）。

## 1. 目标

按 M3 设计 12 节 P10 和第 2 节的故事：

- 项目列表和卡片：看得到而不是成员的项目，卡片上加入；复制项目的链接（7.6、7.7）。已归档的项目页：恢复、删除（P4）。
- 创建项目：请求体是表单的字段，负责人从工作区的管理员和成员中选，标识只留 nerve 接受的字符，nerve 指名的字段错误显示在字段下方（P1，3.19）。
- `ProjectAuthWrapper` 的三种界面：不是成员的"加入项目"（只读项目本身）、已归档、找不到（7.6，P2）。
- 项目设置：general（名、标识、说明、可见性、时区、图标）；members（添加、改角色、移出、离开；负责人、默认负责人、访客可见全部）；features、automations（P3、P5）。归档、恢复、删除（P4）。
- 离开项目的顺序（M1-P4 交接）：nerve 做完之后才回到项目列表。删除、离开、创建等页面级的副作用只在发出修改的会话里进行，经 P9 的 `followInSession`；删除项目组件的 vitest（7.1，9.5）。
- 侧边栏拖动项目的顺序和项目页头的标签栏（P8）；标签栏取到之前页头不提供修改（P8b 的 T5-a）。
- M2 交接第 14 节（下拉框：`CustomSearchSelect` Tab 能到、在按钮旁展开；复制到剪贴板处理失败）、第 10 节（表情选择器的数据由 Nerve 提供，不请求 `cdn.jsdelivr.net`）、第 11 节的页面一侧（项目 general 页的时区）。
- P1–P5、P8 的页面版本，S2 的挂载清单加上 P10 的页面。P8b、P9 留下的三类和四类缺陷在 P10 的页面上清零（附录 A.4）。
- 3.20 中 P10 的行：README 的"前端"一节、前端改动清单；M2 收尾交接和 M1-P4 交接的处理结果。控制者的浏览器核对 C6–C8 和评审不在 plan 中（Task 11 只改文档）。

## 2. 交付物

**文件总览**：plan 的"文件结构"一节逐个列出 Task 1–11 改到的 116 个文件和每个文件所属的 Task：新文件 23 个（web 的 21 个：8 个模块和 hook、13 个测试文件；e2e 的 `fixtures/project-pages.ts`、`fixtures/mounts.ts`），删除 1 个（`dropdowns/member/member-options.tsx`），修改 92 个。web 的 TS 文件 85 个（删除的一个在内；其中测试 23 个：web 应用 22 个、`@nerve/utils` 1 个；web 应用 76 个，`@nerve/propel` 4 个、`@nerve/ui` 3 个、`@nerve/utils` 2 个）都是手改，没有机械步骤；web 的其余 8 个是两个 `package.json`（依赖和 oxlint 的上限）和三个文案文件的两种语言；e2e 15 个（fixture 7 个、故事 8 个）；根目录 3 个（`.oxlintrc.json`、`pnpm-workspace.yaml`、`pnpm-lock.yaml`，Task 1 的依赖声明和 2.14 的范围）和 `tools/keywords.json`（2.14 的规则）；文档 4 个（`README.md`、`docs/v0/frontend-changes.md`、M2 收尾交接、M1-P4 交接）。契约、Go 代码和生成的文件都不变。

**依赖**：不加新的 npm 包。web 应用声明 `emojibase-data` 15.3.2（catalog 一行、`devDependencies` 一行）：它已在锁文件中（`@tiptap/extension-emoji` 依赖它），锁文件只多 web 应用 importer 的两处条目，`pnpm install --frozen-lockfile` 不下载任何东西（M3 设计 7.7、M2 交接第 10 节）。

**共用的新部分**：
- `core/hooks/use-copy-link.ts`：`useCopyLink(): (link, copied) => Promise<void>`（写剪贴板，成功说复制了什么，浏览器不让写时提示失败，自己不拒绝）和 `useCopyProjectLink(): (projectId) => Promise<void>`（地址的工作区中项目的工作项页）。项目的四处复制、P9 的邀请行和工作区 general 页都经它（Task 3）。
- `core/components/project/project-refusal.ts`：`projectRefusal(error)`，`{ kind: "fields"; fields } | { kind: "toast" }`，和 `useProjectRefusal()`：项目表单的拒绝落到名称、标识下方或提示（创建和 general 页共用，Task 4）。`core/components/project/logo-props.ts` 的 `logoPropsOf(picked)`：选择器给出的图标写成 nerve 封闭的 `LogoProps`（Task 4）。
- `core/components/project/project-roles.ts`（整个文件）：`roleChoices(caller, membership)`、`canRemove(caller, membership)`（Task 7）、`addableRoles(workspaceRole)`（Task 8）：页面只提供 nerve 允许的角色和移出（M3 设计 3.5）。`PROJECT_ROLES` 照旧。
- 页面一侧的修改的 hook，每个经 `followInSession`、拒绝经 `useRefusalToast`（或 `useProjectRefusal`）、各有换账户之后兑现的 vitest（`lateSettlings`）：`useJoinProject`（Task 3）、`useCreateProject`（Task 4）、`useUpdateProjectDetails`（Task 5）、`useProjectMembershipChanges`（Task 7）；`useFeatureToggle`（Task 6）、`useProjectDrop`、`useTabPreferences`（Task 10）照同样的写法重写。
- 数据层的两处（扩展，不复制）：`core/lib/reconciled.ts` 的 `answeredAt`（第 3 节第 4 条）；`useProjectFetch` 的 `{ kind: "archived"; project }`（Task 3）。
- 测试的共用部分：`core/lib/fake-controls.ts` 的搜索选择的替身留下它收到的 `focusSearchOnOpen`、`popperModifiers`（Task 2）和选项（Task 8）；其余用 P8b、P9 的（`fake-tab.ts` 的 `lateSettlings`、`fake-refusal.ts`、`fake-toast.ts`、`fake-i18n.ts`、`fake-store-hooks.ts`、`fake-projects.ts`），不另写一份。
- e2e：`e2e/fixtures/browser.ts` 的 `watchPage` 记下每个请求的地址（`requestUrls`），`requestsElsewhere(page, watch)` 给出发往页面之外的源的（Task 1：P1、P3 的页面版本和 S2 都经它，S2 原来自己记的一份删除）；新文件 `e2e/fixtures/mounts.ts`（Task 3：S2 的 `REQUESTS` 由它的列表组成，`APP`、`WORKSPACE`、`PROJECT`、`PROJECT_MEMBER`、`GENERAL`、`ARCHIVED`、`INVITATIONS` 和 `valued` 从 S2 移来，P2 的页面版本读同一份 `PROJECT_MEMBER`，第 3 节第 16 条）；`e2e/fixtures/settings-pages.ts` 的 `transitionsEnded(locator)`（Task 2：等弹窗进场的过渡结束，popper 按按钮此刻的位置定位）、`moveWithinApp(page, path)`（Task 5：在同一个路由里移到另一个地址，不重新载入，React Router 照 `popstate` 移动；第 3 节第 7 条）、`shownWithin(locator)`（Task 10：一秒之内是否显示，只把一秒用完算作没显示）；`e2e/fixtures/workspace-pages.ts` 的 `pickTimeZone`（Task 2：只用键盘选工作区的时区）；新文件 `e2e/fixtures/project-pages.ts`（Task 7、8：`Sent`、`removalOf`、`leavingOf`、`endProjectMembership`、`shownNameOf`）；`e2e/fixtures/api.ts` 的 `archiveProject`（Task 3）、`projectMembershipOf`、`changeProject`、`projectMemberWrites`（Task 8）；`e2e/fixtures/assert/project.ts` 的 `projectSettingsOf`（Task 8）。P9 的 `sentTo`、`sentHeld`、`holdAnswer`、`enabledWithin`、`closedByEscape`、`watchPage`、`expectQuietConsole` 照用。

### 2.1 表情选择器的数据由 Nerve 提供（Task 1；7.7、8.5、9.5；M2 交接第 10 节）

- `web/apps/web/emojibase.ts`：`emojibaseUrl`（`/assets/emojibase/<安装的版本>`，版本读 `emojibase-data/package.json`）、`emojibaseFiles()`（`en/data.json`、`en/messages.json`：frimousse 只读这两个）和 Vite 插件 `emojibase()`：客户端的构建把它们写进资源（`generateBundle` 的 `emitFile`），开发服务器答同样的路径。Nerve 嵌入、提供它们，同其余的资源（webui 的 `/assets/`：长期缓存，新版本是新路径）。`vite.config.ts` 用它。
- `@nerve/propel`：`EMOJIBASE_URL = "/assets/emojibase/15.3.2"`（`emoji/emoji.tsx`，经 `emoji-icon-picker/index.ts` 导出），`EmojiPicker.Root` 传 `emojibaseUrl`：选择器不再请求 frimousse 默认的 CDN（页面的 CSP `connect-src 'self'` 挡住它）。选择器的按钮收 `ariaLabel`（`TCustomEmojiPicker`），创建项目的图标按钮名为"Project icon"（`aria_labels.project_icon`，两种语言）。
- oxlint（7.9）：本 Task 改到的两个 propel 文件清零（`emoji.tsx` 三个插槽把传下去的属性叫 `rest`，不再遮住组件的 `props`；`helper.tsx` 的 `toHex` 移到模块，用 `padStart`，结果相同），propel 的上限 16 → 11。
- 测试：`emojibase.test.ts` 2 个（9.5 的"表情选择器的数据地址和版本"：`EMOJIBASE_URL` 与插件的地址相同、版本是锁定的；两个文件的名字和内容）。端到端：P1 的页面版本打开图标的表情一页，选择器显示表情，没有发往别处的请求（`requestsElsewhere`）、没有 CSP 违规；P1、P3 的其余页面版本也打开它。
- **`HEAD` 和 ETag**（7.7"P10 核对并写进 review"）：原型上 `webui` 对 `HEAD /assets/emojibase/15.3.2/en/data.json` 答 200，`Cache-Control: public, max-age=31536000, immutable`，没有 `ETag`、没有 `Last-Modified`（`GET` 同样，700,120 字节；附录 A.3）。frimousse 0.3.0 的做法（`dist/index.js`）：localStorage 没有数据时下载两个文件；有数据而 sessionStorage 中没有本次标签页会话的记录时，先 `HEAD` 两个文件比对 ETag，没有 ETag 就重新下载；同一标签页会话之后不再请求。所以每个新的标签页会话第一次打开选择器时重新下载约 700 KB（浏览器的 HTTP 缓存通常直接答它，`immutable`），功能不受影响。给嵌入的文件加 ETag 是 `webui` 的改动（Go），不在 P10（第 5 节收尾）；写进评审是控制者的。

### 2.2 下拉框（Task 2；7.7、9.7 C7；M2 交接第 14 节，根因见 M2/P5 spec 2.3）

- `@nerve/ui` 的 `CustomSearchSelect`（整个文件；props 加两个可选的，原有的不变）：按钮是 Headless UI 的 `Popover.Button`，Tab 能到，点击、Enter、空格打开（原来是 `Combobox.Button`，Headless UI 2.2 固定给它 `tabIndex: -1`）；搜索框和选项是面板里的 `Combobox`（`static`），打开状态只有 Popover 的一份。列表打开时搜索框取得焦点，调用方可以不要（`focusSearchOnOpen`，默认要；不滚动：popper 那时还没定位）；Escape 从按钮或搜索框都关上列表，焦点回到按钮，不传给外面的弹窗（输入打开 Combobox 之后，它的输入框先处理 Escape 并 `preventDefault`，Headless UI 随之跳过面板自己的处理，所以面板在捕获阶段处理），输入法组字时的 Escape 留给输入法（`event.nativeEvent.isComposing`，预检的 L7），也不传出列表：列表和外面的弹窗都留着（Task 2 的修正轮，裁定 T2-f）；单选选了一项就关；列表关上时清空搜索；popper 的 ref 在面板本身（门户里），列表在按钮旁展开，调用方的 `popperModifiers` 交给 popper，调用方的 `optionsClassName` 在面板上（宽度、z-index，裁定 T2-a）。按钮由键盘取得焦点时显示一圈轮廓（Headless UI 的 `data-focus`，裁定 T2-b）。`helper.tsx` 的 `placement` 用 popper 的 `Placement`（手写的联合删除），两个新 props 的类型（`popperModifiers` 是 popper 的 `Modifier`）。
- `@nerve/ui` 的面包屑下拉框 `breadcrumbs/navigation-search-dropdown.tsx`：一节一个 Tab 停留点，选择的按钮（原来 Combobox 的按钮不在 Tab 顺序中、里面的标题按钮在；选择改成 Popover 的按钮之后成了两个，预检的 L9）：最后一节的标题是 `<span>`（点它由外面的按钮打开列表），之前一节的标题按钮 `tabIndex={-1}`，点它去那一节的页，键盘从侧边栏到那里（第 3 节第 15 条）。
- 成员下拉框 `dropdowns/member/base.tsx`（整个文件）：以 `CustomSearchSelect` 为底（工作区的成员，按 `memberIds` 筛选，单选、多选），`member-options.tsx` 删除（popper 的 ref 在列表唯一的子元素上，列表停在页面左上角；两份打开状态，第 3 节第 2 条）。对调用方的约定照旧（预检的 A-M5，第 3 节第 15 条）：调用方推迟时（`renderByDefault` 为 false：桌面上列表的每一行）指针第一次经过之前只渲染按钮，没有选择和 popper（与 `@nerve/ui` 的 `ComboDropDown` 相同：`useState(renderByDefault)`、`onMouseEnter`）；手机上（`usePlatformOS` 的 `isMobile`）打开时搜索框不取得焦点（`focusSearchOnOpen={!isMobile}`：键盘会盖住列表）；列表离窗口边缘 12 像素（`popperModifiers` 的 `preventOverflow`）；按钮的点击不冒泡到行，也不做默认的动作，由 Headless UI 的 `Popover.Button` 自己做（它的点击先 `preventDefault`、`stopPropagation`，再切换，与原来的 `handleOnClick` 相同；成员下拉框不另写，自己的处理器 `preventDefault` 反而会让 Headless UI 跳过它的切换）。Tab 只停在里面的 `DropdownButton`：选择自己的按钮 `tabIndex={-1}`，`DropdownButton`（`dropdowns/buttons.tsx`）收调用方的 `tabIndex`、交给三种样式的按钮（第 3 节第 3 条）。`types.d.ts` 删去只有旧写法读的 `button`。web 的 oxlint 上限 356 → 350（`dropdowns/member/` 的六条随旧的写法删除）。
- 测试：`dropdowns/member/base.test.tsx` 4 个（服务端渲染，`CustomSearchSelect` 是 `fake-controls.ts` 的替身，留下它收到的 props：调用方推迟时只有按钮、没有选择；不推迟时是选择，按钮在选择的按钮里（渲染替身收到的 `customButton`）；桌面上打开时搜索框取得焦点，手机上不；列表离窗口边缘 12 像素）。打开、焦点和位置服务端渲染看不到，由端到端核对：P1 的负责人一个（外面那个选择自己的按钮 `tabindex="-1"`；创建弹窗的进场过渡结束之后点负责人，列表在按钮旁（`expectListBesideButton`），搜索框有焦点；Escape 关上列表、选择的按钮显示焦点；在按钮上按 Enter 再打开，输入之后的 Escape 也一样，输入法组字时的 Escape 列表留着；没有 Escape 传到 `document`（创建弹窗在那里听 Escape）；再打开时没有搜索）；W3 的页面版本中时区改为只用键盘（`pickTimeZone`）；个人设置的时区（A8、A9）也是 `CustomSearchSelect`，照旧通过；项目页头的项目一节只停一次 Tab（P2，Task 3）；添加成员弹窗的成员选择用 Tab 到达（P5，Task 8）。`CustomSelect` 的 9 个调用方和 `CustomSearchSelect` 的调用方（M3 的 6 个，M4–M6 经成员下拉框的 14 个文件和页头的 9 个）逐个的浏览器核对是 C7（控制者的，M3 设计 9.7）。

### 2.3 项目列表、卡片和加入；包装层的界面；复制链接；P2 的页面版本（Task 3；P2，3.4、3.5，7.1、7.6、7.7；P8b 的 F-3；Codex 4.3 第 1 条）

- 复制：`useCopyLink`、`useCopyProjectLink`（第 2 节的共用部分）。M2 交接第 14 节的三处（侧边栏 `projects-list.tsx`、扩展侧边栏、卡片）和项目页头的菜单（`use-project-actions.ts`）经 `useCopyProjectLink`；P9 的邀请行（`use-copy-invitation-link.ts`）和工作区 general 页的地址（`workspace-details.tsx`）经 `useCopyLink`，各自的成功、失败的写法删除（一处复制，第 3 节第 8 条）。卡片的复制按钮有名字（`copy_link`）。失败的提示是 `something_went_wrong_please_try_again`（经 `t()`，7.7）。
- 加入：`core/components/project/use-join-project.ts` 的 `useJoinProject(): (projectId, done?) => Promise<void>`：`joinProject` 经 `followInSession`，加入之后做页面的下一步（`done`），拒绝按 `code` 提示，忙到跟进做完，自己不拒绝。`JoinProjectModal`（整个文件）经它：加入之后关闭、打开项目的工作项页；加入在途时关不掉（取消禁用，`ModalCore` 的 `handleClose` 为空），按钮加载。
- `useProjectFetch`：`ProjectAccess` 加 `{ kind: "archived"; project }`：项目已归档时（不论角色）是它，不取成员才读的四项（显示设置、标签、成员、状态）。决定的顺序：取不到、加载中、找不到（P8b）之后，已归档在不是成员、成员之前（`it.each` 钉住）。
- `ProjectAuthWrapper`（整个文件）照 `kind` 渲染：`not-member` 是加入界面（经 `useJoinProject`，按钮忙到跟进做完），`archived` 是已归档的界面（`project_empty_state.archived.*`：标题"This project is archived"，按钮去 `/{slug}/projects/archives`），`not-found` 照旧。`project-access-restriction.tsx` 给出这三种。
- F-3：`Reconciled.answeredAt`（一个值显示的是第几个写入的回答；所有 `Reconciled` 共用一个计数，0 是还没取到）和 `ReconciledByKey.answeredAt(key)`；项目 store 的 `getProjectById` 在项目自己的读取和它的工作区的两个列表中给出 nerve 最后回答的那一份（第 3 节第 4 条）。
- 文案：`project_empty_state.archived.title`、`description`、`cta_primary`（两种语言）。
- 测试：`use-copy-link.test.ts` 3 个（写剪贴板并说复制了什么；浏览器不让写时提示失败、不拒绝；项目的链接）；`use-join-project.test.ts` 5 个（加入之后做下一步；没有下一步；拒绝提示、不做下一步；换账户之后兑现的两种）；`project-wrapper.test.tsx` 加 1 个（已归档的界面）；`use-project-fetch.test.ts` 的取数表加 1 行、决定表加 2 行；`project.store.test.ts` 改写 1 个、加 1 个（F-3）。Task 3 的修正轮（裁定 T3-d、T3-e）：决定表再加 3 行（已归档的项目：nerve 找不到时是找不到，不可达时是不可达，没回答时是加载：拒绝和没回答在已归档之前，`T3.15`）；`project.store.test.ts` 再加 2 个（列表取了两次之后回答的自己的读取：所有副本共用一个计数，`T3.16`；自己的读取之后回答的已归档的列表：它也是一份副本，`T3.17`）。
- 端到端：P2 的两个页面版本（成员直接打开看得到而不是成员的公开项目的地址：加入界面，只读项目本身；加入之后页面显示、读其余四项；私密的找不到，已归档的说已归档。项目页的卡片：成员看到可以加入的公开项目、看不到私密的；复制链接，浏览器拒绝时说不能；加入经 `sentHeld` 扣住：取消禁用、Escape 不关，只发一次；加入之后项目页头的项目一节只停一次 Tab，2.2）。S2 加成员、访客、不是项目成员的成员打开项目列表的行，和"项目的管理员打开已归档的项目"（2.12）。Task 3 的修正轮（裁定 T3-c）：卡片的故事另有一次被拒绝的加入（管理员在对话框开着时删除项目）：原因在提示里，对话框留着（`settings-pages.ts` 的 `closedWithin`：一秒之内不关；`closedByEscape` 改为经它），按钮重新可用（`T3.14`）；加入在途时对话框的按钮忙（`enabledWithin`，`T3.13`）。

### 2.4 创建项目；P1 的页面版本（Task 4；P1，3.19，7.1、7.6；P8b spec 第 5 节 P10 一行；P4a 评审的 P10）

- `projectRefusal`、`useProjectRefusal`（第 2 节的共用部分）：名称、标识已被占用（`project.name_taken`、`project.identifier_taken`）落到各自的字段下；nerve 的字段错误只指向名称、标识时落到字段下（`FIELD_ERROR_MESSAGES`）；其余（负责人不被允许等）按原因提示。
- `logoPropsOf(picked: TChangeHandlerProps): LogoProps`：表情只留它的值，图标只留名字和颜色；`@nerve/propel` 导出 `TChangeHandlerProps`。页面不写 `logo_props.emoji.url`，也没有按它请求图片的读者（8.5"P10 核对"：web 的源文件中没有读它的地方）。
- `core/components/projects/create/use-create-project.ts`：`ProjectCreationForm`（`ProjectCreate` 的 `name`、`identifier`、`description`、`network`、`logo_props`，加 `project_lead_id: string | null`）；`useCreateProject(): (workspaceSlug, form, setError) => Promise<Project | undefined>`：请求体由这些字段构成，没选负责人时不发 `project_lead_id`；经 `followInSession`：创建之后提示、交回项目，被拒绝时经 `useProjectRefusal`，换账户之后交回 `undefined`。
- `CreateProjectForm`（`projects/create/root.tsx`，整个文件）：交回项目时进入功能一步；创建在途时 Escape 不关，页头的关闭按钮（有名字）和取消禁用；`create-project-modal.tsx` 的 Escape 只在功能一步关弹窗。`utils.ts` 的默认值是 `ProjectCreationForm`，没有负责人。负责人（`attributes.tsx`）：`MemberDropdownBase`，候选是工作区的有效管理员和成员（不含访客，3.19），再选一次已选的就是没有负责人；它在表单的 Tab 顺序中的位置照旧（`tabIndex={getIndex("lead")}`，第 3 节第 12 条）。
- `@nerve/utils` 的 `projectIdentifierSanitizer`：先转大写，再只留 A–Z、0–9 和 ÇŞĞİÖÜ（原来保留小写，nerve 只收大写）。标识里不再可能有 `.`，实时检查不会把 `.`、`..` 放进地址（P4a 评审的 P10）。
- 测试：`project-refusal.test.ts`（`it.each`，决定的顺序：名称已被占用、标识已被占用、只指向表单字段的字段错误、也指向表单没有的字段的、没有字段的拒绝、没有到达 nerve）；`logo-props.test.ts`（表情、图标各一行）；`use-create-project.test.ts`（表单的字段、没选负责人时不发；发选的负责人；字段下的两种拒绝；提示的拒绝；换账户之后兑现的两种）；`@nerve/utils` 的 `project.test.ts`（转大写、去掉别的字符，土耳其字母照留；`..` 成为空，`w.e.b` 成为 `WEB`）。Task 4 的修正轮（裁定 T4-a、T4-b）：`fake-tab.ts` 的 `lateSettlings` 扩展为 `lateSettlingsAnswering(answer)`，`use-create-project.test.ts` 换账户之后的两行答以创建出的项目，交回它就失败（`T4.12`）；负责人的候选由 `projects/create/lead-candidates.ts` 的 `leadCandidates` 筛选，`lead-candidates.test.ts` 4 行（管理员、成员、访客、成员资格已结束的成员；`T4.13`，`T4.6` 也由它发现）。
- 端到端：P1 的页面版本（名称给出标识；标识输入转大写、去掉 `.`；表单的顺序中负责人紧在取消之前：从取消按 Shift+Tab 到它，Enter 打开它的列表，有管理员和成员、没有访客；被占用的标识和不允许的名称各在字段下；创建的请求体经 `sentHeld`：只有表单的字段，扣住时取消和页头的关闭禁用、Escape 不关（预检的 A-M3）；之后列表和侧边栏有它；数据库中的一行和成员、显示设置）。Task 4 的修正轮（裁定 T4-c）：扣住时创建的按钮（"Creating"）也禁用（`T4.14`）；P1 的 `named` 移到 `e2e/fixtures/auth.ts`。

### 2.5 项目设置的 general；P3 的页面版本（一）（Task 5；P3，3.19，7.1、7.6；P8b 的 P5、P6；M2 交接第 11 节）

- `core/components/project/use-update-project-details.ts`：`ProjectDetails`（`ProjectUpdate` 中 general 页编辑的 `name`、`identifier`、`description`、`network`、`logo_props`、`timezone`）、`projectDetailsOf(project)`；`useUpdateProjectDetails(): (project, workspaceSlug, details, setError) => Promise<void>`：经 `followInSession` 发出一个修改：标识改了时先问 nerve（`checkProjectIdentifier`），被占用就在标识下说明，什么都不发；否则发这些字段，成功提示；被拒绝时经 `useProjectRefusal`（检查本身被拒绝也是）。原来的检查在 `try` 之外，被拒绝时加载一直不结束（P8b 的 P5），`!available` 的一支没有测试（P6）。
- `ProjectDetailsForm`（`project/form.tsx`，整个文件，411 → 292 行）：默认值是 `projectDetailsOf(project)`；提交经 `useUpdateProjectDetails`，按钮忙到跟进做完（`isSubmitting`）；标识照 Task 4 的 `projectIdentifierSanitizer`；图标经 `logoPropsOf`，选择器的按钮有名字；时区是 `GET /api/v0/timezones` 的列表（M2 交接第 11 节）；不能编辑的人看到 nerve 所持的值，改不了。原来按 Plane 的错误形状读拒绝、`console.error`、提交之后 300 毫秒才结束的加载和 `useEffect` 的 `reset`（和它的 `react-hooks/exhaustive-deps` 抑制）都删除：页面按项目挂载一个表单（`key={project.id}`，第 3 节第 7 条）。
- general 页（`settings/projects/[projectId]/page.tsx`）：地址的项目经 `getProjectById(projectId)`（不读 `currentProjectDetails`）。
- 文案：没有读者的四条删除（`project_name_already_taken`、`project_name_cannot_contain_special_characters`、`project_identifier_already_taken`、`common.identifier_already_exists`，两种语言；拒绝的文案是 `errors` 的）。
- 测试：`use-update-project-details.test.ts`（发编辑的字段、没改的标识不问；改了的先问再发；被占用时在标识下说、不发；字段下的拒绝；提示的拒绝；检查被拒绝时提示、不发、结束；换账户之后兑现的修改；换账户之后才回答"被占用"）。
- 端到端：P3 的 general 页的页面版本（改名、标识、说明、可见性、时区、图标，重新载入之后都在；输入 `o.ps` 显示 `OPS`，另一个项目的标识，在标识下说明，没有 `PATCH`；修改的请求体经 `sentHeld`，扣住时按钮不可用；成员的页面只读；没有发往别处的请求，`requestsElsewhere`）；另一个项目的 general 页经同一个路由里的移动到达（`moveWithinApp`：Web 的、Ops 的、再回 Web 的，Web 的读取已在会话的缓存里，页面不重新挂载），显示的是那个项目的值（第 3 节第 7 条，预检的 A-M1）。

### 2.6 项目设置的 features 和 automations；P3 的页面版本（二）（Task 6；P3，7.1、7.6；总体设计 7.7 的 PF-M2）

- `useFeatureToggle`（整个文件）：一下开关经 `followInSession(() => toggleProject(projectId, field), …)`：完成时提示（Plane 的英文"Project feature updated successfully."），被拒绝时经 `useRefusalToast`；原来的 `setPromiseToast` 不论会话都提示，失败的文案也不是 nerve 的原因。开关作用于 nerve 最近的回答，在轮到它时由 store 决定（P8b 已做，P10 不改）。
- `AutoArchiveAutomation`：项目是地址的（`useParams` 的 `projectId`，经 `getProjectById`；原来读路由 store 的 `currentProjectDetails`，页面读地址只有一种写法）；`send(change, done?)` 经 `followInSession`，拒绝经 `useRefusalToast`（原来提示固定的"Something went wrong"）；开关经 store 的 `toggleAutoArchive`（P8b）；`handleChange(formData, done?)` 把 `done` 交给自定义时长的弹窗。`SelectMonthModal`：提交时 `await handleChange(formData, onClose)`，nerve 做完之后才关（原来发出就关，被拒绝时弹窗已不在）；在途时关不掉。
- 测试：`use-feature-toggle.test.tsx`、`auto-archive-automation.test.tsx` 各改写 1 个（拒绝说 nerve 的原因）、加换账户之后兑现的两种（4 → 6、3 → 5 个）；`select-month-modal.test.tsx` 加 1 个（nerve 做完才关，不是发出就关）。
- 端到端：P3 的 features、automations 页的页面版本（nerve 建的项目四个功能都关着；每页的开关只发它的字段，开关随之打开；自动归档打开发 `{ archive_in: 1 }`；自定义 6 个月的修改被扣住：取消禁用、Escape 不关，放行之后关上；再从列表选 3 个月）。

### 2.7 成员页的改角色、移出、离开；P5 的页面版本（一）（Task 7；P5，3.5、3.7，7.1、7.6；M1-P4 交接；P8b 的 P5、T4-b；P9 spec 第 5 节）

- `core/components/project/settings/use-project-membership-changes.ts`：`useProjectMembershipChanges(workspaceSlug, projectId)` 给出 `changeRole(userId, role: ProjectRole)`（发编号）、`remove(userId, done?)`、`leave(done?)`，都经 `followInSession`，拒绝经 `useRefusalToast`；`done` 是发出它的弹窗的关闭，只在成功的跟进中（被拒绝时弹窗留着，第 3 节第 6 条）；`leave` 在 nerve 做完之后才关弹窗、去 `/{slug}/projects`（M1-P4：原来先跳转再离开，nerve 拒绝时人已离开页面），忙到跳转做完；唯一的管理员离开被拒绝（409 `project.sole_admin`）时说明原因、停在原处（P8b 的 T4-b：原来这个原因从不显示）。
- `project-roles.ts`：`MembershipCaller`、`ShownMembership`；`roleChoices(caller, membership)`：nerve 允许的角色（3.5：调用者是项目管理员或是工作区管理员的项目成员才有；工作区管理员不受相对规则限制；别人不改自己的、只改低于自己的、只给低于自己的；工作区访客只能是访客）；`canRemove(caller, membership)`：不是自己的，角色不高于自己的。成员页只提供这些（P5b spec 第 5 节；隐藏不是保护，第 3 节第 10 条）。
- 成员页：`member-columns.tsx`（整个文件）去掉从没有提交过的 `useForm`、`console.log` 和 `err.error`；角色列给出 `roleChoices`，没有选项时只显示角色，角色的名字经 `ROLE_DETAILS[role].i18n_title` 翻译（原来是英文的 `ROLE`）；名字列的菜单在自己一行是"离开"，别人一行在 `canRemove` 时是"移出"。`useProjectColumns` 算出调用者和每一行的 `ShownMembership`。`member-list-item.tsx`（整个文件）经 `useProjectMembershipChanges`，把确认框的关闭作为 `done` 交给 `remove`、`leave`（原来读 `err.error`、自己跳转；P9 spec 第 5 节的 `:60`、`:68`、`member-columns.tsx:152`）。
- 确认框 `confirm-project-member-remove.tsx`：在途时关不掉；不再自己关上（原来 `await onSubmit()` 之后不论结果都关，被拒绝的离开也关，预检的 A-M2），关上是 `onSubmit` 成功的跟进（上一条）；确认按钮的 `tabIndex={1}` 删除（oxlint：正的 `tabIndex`），web 的上限 350 → 349。离开弹窗 `leave-project-modal.tsx`：表单的值有类型（原来是 `any`），核对名称和"Leave Project"之后 `await leave(handleClose)`；在途时关不掉；nerve 拒绝时弹窗留着，原因在提示里（第 3 节第 6 条）。侧边栏和项目页头的离开都经这个弹窗。离开只有一处：关键词规则 `project-leave`（2.14）。
- 测试：`use-project-membership-changes.test.ts`（改角色发编号、发到成员关系的项目；移出之后关上发出它的弹窗、停在原页；离开之后才关上弹窗、显示项目列表，显示之后才兑现；三种修改被拒绝的 `it.each`，停在原处、弹窗不关；三种修改各两种换账户之后兑现）；`project-roles.test.tsx` 的角色表加上每个角色显示的键（成员页的选择按键选），加 `roleChoices`、`canRemove` 的表（项目管理员、工作区管理员（是或不是项目的成员）、项目成员对成员、另一位管理员、自己、工作区的访客）。
- 端到端：P5 的页面版本（不是工作区管理员的项目管理员：只看到低于自己的角色，另一位管理员一行没有角色选择；改角色的请求体 `{ role: 5 }`；移出另一位管理员和访客，确认被扣住时取消禁用、Escape 不关；只剩他一个管理员时离开被拒绝，页面说明原因，确认框留着、按钮可用；成员离开，回答被扣住时页面不跳转、放行之后到项目列表；访客从侧边栏离开（侧边栏和项目页头只给访客提供离开）：她的弹窗开着时 pat 经 API 结束她的成员关系，离开被拒绝（403），弹窗留着、按钮可用；pat 再加回她，离开被扣住时弹窗关不掉，放行之后 Web 从侧边栏消失）。

### 2.8 添加成员和成员的默认值；P5、P3 的页面版本（二）（Task 8；P3、P5，3.5、3.19，7.1、7.6；P8b 的 P5）

- `addableRoles(workspaceRole)`：添加时 nerve 允许的角色（工作区管理员只能是管理员，工作区访客只能是访客，成员任意；还没选人时任意）。添加没有相对规则，不按调用者的角色筛（第 3 节第 10 条）。
- `AddProjectMembersModal`：候选是工作区的有效成员中还不是项目成员的（原来也列出成员关系已结束的人）；每行的角色按 `addableRoles`，名字同成员页（`ROLE_DETAILS`）；成员的选择用 `customButton` 的 `<span>`（不在按钮里再放按钮）；提交经 `followInSession`（生成的 `ProjectMembersAdd`）：成功时关闭、提示，被拒绝时说原因、表单留着（原来只有 `console.error`，P8b 的 P5）；在途时关不掉。不再收 `workspaceSlug`（`member-list.tsx` 不传）。
- `ProjectSettingsMemberDefaults`（整个文件）：负责人、默认负责人（项目中不是访客的成员）、访客可见全部，显示 nerve 所持的值（`getProjectById(projectId)`），一次改一个字段，经 `followInSession`：成功提示，被拒绝时说原因；原来的 `useForm` 和先改显示再发删除（第 3 节第 11 条）。访客可见全部经 store 的 `toggleProject`（轮到它时由 nerve 最近的回答决定，P8b）。
- 测试：`add-project-members-modal.test.tsx` 5 个（候选；添加之后关闭并提示；拒绝时留着并说原因；换账户之后兑现的两种）；`project-roles.test.tsx` 加 `addableRoles` 的表，添加的选择也按键选；`project-settings-member-defaults.test.tsx` 2 → 5 个（说完成和 nerve 的原因；换账户之后兑现的两种）。
- 端到端：P5 的页面版本在开头加上添加（用 Tab 到成员的选择，Enter 打开列表、搜索框有焦点，预检的 A-M4；候选只有工作区的管理员、ann 和 gus；gus 只能是访客；请求体经 `sentHeld`，扣住时取消禁用、Escape 不关）；P3 的 members 页的页面版本（负责人、默认负责人各从项目中不是访客的成员里选，每个修改只发它的字段；打开访客可见全部；数据库中的设置）。
- 为了让 P3、P5 的故事和 `assert/project.ts` 保持在约 400 行，它们原来各写的经 API 的修改和读取移到共用的 fixture（`api.ts` 的 `projectMembershipOf`、`changeProject`、`projectMemberWrites`，`assert/project.ts` 的 `projectSettingsOf`；P3 经 API 被拒绝的添加也经 `projectMemberWrites`；第 3 节第 13 条）。

### 2.9 归档、恢复和删除项目；P4 的页面版本（Task 9；P4，3.4、3.6，7.1、7.6，9.5；Codex 4.3 第 2 条）

- `DeleteProjectModal`：`TDeleteProjectForm`、`CONFIRMATION = "delete my project"`；按提交的值核对名称和确认的话（提交按钮在两者都对之前不可用）；`deleteProject(project)` 经 `followInSession`：成功时关闭，只在页面是这个项目的（地址的 `projectId` 是它）时回到 `/{slug}/projects`，提示；被拒绝时经 `useRefusalToast` 说原因、停在弹窗；在途时关不掉。照 P9 的 `DeleteWorkspaceModal` 写（关闭在先，跳转不等，与它相同）。原来不论会话都跳转、提示，被拒绝时提示固定的文字，在途时弹窗可以关。
- `ArchiveRestoreProjectModal`（整个文件）：归档或恢复经 `followInSession`：成功时提示、关闭、回到 `/{slug}/projects`，忙到跳转做完；被拒绝时说原因、停在弹窗；在途时关不掉。
- 已归档的卡片（`card.tsx`）：恢复、删除提供给项目的管理员，也提供给是工作区管理员的项目成员（`allowPermissions([ADMIN], PROJECT, …)`，与项目设置提供归档的条件相同，nerve 3.4 也允许他们；原来只看项目角色，第 3 节第 9 条）；删除的图标按钮有名字（"Delete"）。
- 测试：`delete-project-modal.test.tsx` 7 个（9.5"删除项目的组件"：项目自己的页让位给项目列表；已归档的项目页停着；表单两种填错的不删；拒绝说原因、停着；换账户之后兑现的两种不跳转、不提示）；`archive-restore-modal.test.tsx` 5 个（归档、恢复的成功；拒绝；换账户之后兑现的两种）。
- 端到端：P4 的页面版本（管理员在设置页归档，弹窗被扣住时取消禁用、Escape 不关、地址不变，之后从侧边栏消失；是工作区管理员的项目成员在已归档的项目页恢复；经 API 再归档，管理员在那里按名称删除，同样被扣住；删除之后停在已归档的项目页；`expectProjectDeleted`：项目的每张表先各有一行）。

### 2.10 侧边栏拖动项目和项目页头的标签栏；P8 的页面版本（Task 10；P8，3.18，7.1、7.6；总体设计 7.7 的 PF-M2；P8b 的 T5-a；P9 spec 第 5 节）

- `useProjectDrop`（整个文件）：移动经 `followInSession`，失败时经 `useRefusalToast` 说原因（原来是固定的"something went wrong"，不论会话）；没有到达 nerve 的失败说 `errors.unknown`。移动的位置在轮到它时由 store 算（P8b）。
- `useTabPreferences(projectId)`（整个文件）：交回 `{ navigation, changes }`；`navigation` 在设置取到之前是 nerve 的默认，`changes`（`TTabChanges`：`toggleDefault`、`hide`、`show`）在那之前是 `undefined`：页头不提供 store 不会发出的修改（P8b 的 T5-a：原来菜单在设置到达之前就可用，点了提示失败）。每个修改经 `followInSession`：设为默认时提示，被拒绝时经 `useRefusalToast`（原来提示固定的文字，再进 `console.error`；P9 spec 第 5 节）。
- 页头：`tab-navigation-root.tsx` 把 `changes` 交给每个标签和"更多"；`tab-navigation-visible-item.tsx`（整个文件）只在有 `changes` 时有右键菜单；`tab-navigation-overflow-menu.tsx` 只在有 `changes` 时有"显示"和"设为默认"的按钮。
- 测试：`use-project-drop.test.ts` 改写 1 个、加换账户之后兑现的两种（4 → 6 个）；`use-tab-preferences.test.ts`（整个文件，1 → 5 个：取到之前是默认、没有修改；每个修改作用于 nerve 最近的回答；设为默认的提示和拒绝；换账户之后兑现的两种）。
- 端到端：P8 的页面版本（工作区的导航设为标签式；扣住设置的取数时右键"Modules"没有菜单（`shownWithin`），放行之后有；设为默认、收起视图的请求体；按住 Ops 的拖动柄拖到 Docs 上（`toPass` 有 5 秒的期限，预检的 L4），发 `{ sort_order: 35535 }`；刷新之后都在；数据库中两个项目的显示设置；工作项页请求 M4 的筛选的旧地址，两次 404 是预期的，2.12）。

### 2.11 文档（Task 11；3.20 中 P10 的行；M2 交接第 10、11、14 节；M1-P4 交接）

- `README.md` 的"前端"一节：能用的项目页面（M3/P10 起），状态、标签的设置页和个人主页在 M3/P11。
- `docs/v0/frontend-changes.md`：3.1 中 M3 一行的状态；3.2"所有处理接口错误的地方"写明 P10 的页面；"页面跟进修改的结果"一行写 P9、P10；新增八行（表情选择器的数据、下拉框、复制到剪贴板、创建项目、已归档的项目、项目成员页、项目的成员默认值、项目页头的标签栏），都是"已完成，M3/P10"；下拉框一行的成员下拉框是"创建项目的负责人；M4–M6 的指派人、负责人"，复制一行的失败是"浏览器拒绝时"（预检的 L10）。
- M2 收尾交接的"处理结果（M3/P10）"：第 10、11 节关闭；第 14 节代码一侧完成，关闭条件中的 `member-options` 一项改为成员下拉框（`member-options.tsx` 已删除，预检的 L1），C7、C8 写进评审之后关闭；状态保持 `open`（第 12 节等 M3 的收尾）。M1-P4 交接的"处理结果（M3/P10）"：离开项目的顺序完成；保留名在 P8a 已完成；状态在 M3 的收尾改为 `closed`。

### 2.12 挂载时的取数（W2；7.1；S2）

S2 的 `REQUESTS` 是每个账户每一页的挂载清单唯一的地方（P8b 的裁定 D5），由 `e2e/fixtures/mounts.ts` 的列表组成（Task 3 从 S2 移出，第 3 节第 16 条）。P10 加的（Task 3；附录 A.3 是这时的记录）：

| 页面 | 管理员 | 成员 | 访客 | 不是项目成员的成员 | 条件 |
|---|---|---|---|---|---|
| `/{slug}/projects` | `APP`、`WORKSPACE`、`ARCHIVED`（P8b 起） | 同管理员（P10 加） | 同管理员（P10 加） | 同管理员（P10 加） | 已归档的列表是项目页自己的取数（`useArchivedProjectsFetch`），每个角色都取（nerve 对每个角色答他看得到的已归档项目） |
| 项目的管理员打开已归档的项目（新测试） | `APP`、`WORKSPACE`、`PROJECT` | | | | 已归档的界面；不取成员才读的四项（`PROJECT_MEMBER`） |

`APP`、`WORKSPACE`、`GENERAL`、`INVITATIONS` 同 P9；`ARCHIVED` 是 `GET /api/v0/workspaces/{slug}/projects?archived=true`；`PROJECT` 是 `GET /api/v0/projects/{project}`；`PROJECT_MEMBER` 是显示设置、标签、成员、状态四个。每张清单是恰好的（`watchPage`）。不是项目成员的成员打开项目设置只取 `PROJECT`（P8b 起）。工作项页（`/{slug}/projects/{id}/issues`）的成员还请求 M4 的 `GET /api/workspaces/{slug}/projects/{id}/user-properties/`（Plane 的旧地址，404，关键词守卫的例外 `until: "M4"`）：它在 S2 之外，P2、P8 的页面版本把这两个 404 写成预期的失败。

### 2.13 页面级副作用的会话核对（7.1：组件逐个列出）

页面在修改兑现之后跳转、提示、改界面的每一处，都经 `followInSession`（只在发出它的会话里跟进）。每一处由它自己的 vitest 守着：修改在另一个标签页换了账户之后才兑现（`lateSettlings` 的两种），页面什么都不做；那一处不经 `followInSession` 自己跟进时，这个测试失败（附录 A.2 的 W3 一类）：

| 组件 | 修改 | 跟进 | 守着它的 vitest | Task |
|---|---|---|---|---|
| `use-join-project.ts`（卡片的对话框、包装层的加入界面） | 加入 | 对话框打开项目；拒绝提示 | `use-join-project.test.ts` | 3 |
| `use-create-project.ts`（创建表单） | 创建 | 提示、交回项目（表单进入功能一步）；字段下或提示 | `use-create-project.test.ts` | 4 |
| `use-update-project-details.ts`（general 页） | 标识的检查和修改 | 提示；标识下或提示 | `use-update-project-details.test.ts` | 5 |
| `use-feature-toggle.ts`（features 的四页） | 功能的开关 | 提示；拒绝提示 | `use-feature-toggle.test.tsx` | 6 |
| `auto-archive-automation.tsx`（和自定义时长的弹窗） | 自动归档的开关和月数 | 关上自定义时长的弹窗；拒绝提示 | `auto-archive-automation.test.tsx`（关上由 `select-month-modal.test.tsx` 守） | 6 |
| `use-project-membership-changes.ts`（成员页、离开弹窗） | 改角色、移出、离开 | 移出、离开之后关上发出它的弹窗（`done`），离开之后去项目列表；拒绝提示 | `use-project-membership-changes.test.ts` | 7 |
| `add-project-members-modal.tsx` | 添加成员 | 关闭、提示；拒绝提示 | `add-project-members-modal.test.tsx` | 8 |
| `project-settings-member-defaults.tsx` | 负责人、默认负责人、访客可见全部 | 提示；拒绝提示 | `project-settings-member-defaults.test.tsx` | 8 |
| `delete-project-modal.tsx` | 删除项目 | 关闭、项目自己的页回到项目列表、提示；拒绝提示 | `delete-project-modal.test.tsx`（9.5） | 9 |
| `archive-restore-modal.tsx` | 归档、恢复 | 提示、关闭、回到项目列表；拒绝提示 | `archive-restore-modal.test.tsx` | 9 |
| `use-project-drop.ts`（侧边栏） | 移动项目 | 拒绝提示 | `use-project-drop.test.ts` | 10 |
| `use-tab-preferences.ts`（项目页头） | 标签栏 | 设为默认的提示；拒绝提示 | `use-tab-preferences.test.ts` | 10 |

不跟进的：
- 复制到剪贴板的提示（不是 nerve 的修改，标签页的会话不相干）。
- 组件自己的状态：确认框、离开弹窗在 `await` 之后结束忙碌（关上在成功的跟进中，上表），general 页、创建表单的 `isSubmitting`。另一个标签页以另一个账户登录时，`AuthenticationWrapper` 在新账户的资料到达之前显示加载，页面连同这些组件卸载（P9 的裁定 P4，P9 spec 2.13）。提示和跳转是全局的，所以都经 `followInSession`。
- 离开弹窗在发出之前自己核对名称和"Leave Project"的两条提示（还没有修改）。
- 会话切换的端到端在 P9 的 W3（M3 设计 9.6）；P10 的每一处由上表的 vitest 守着，没有另加端到端。

### 2.14 静态检查的范围（Task 1–10；总体设计 7.7；P9 review 的 M4）

- 根目录 `.oxlintrc.json` 的 `typescript/no-non-null-assertion` 范围（P9 的写法）：每个 Task 加它新建的模块和测试、它改动的 nerve 模块和测试中不在已有范围里的：Task 1 `emojibase.ts`、`emojibase.test.ts`；Task 2 `dropdowns/member/base.test.tsx`、`core/lib/fake-controls.ts`；Task 3 `use-join-project.ts`、`use-copy-link.ts` 和它们的测试；Task 4 `project-refusal.ts`、`logo-props.ts`、`use-create-project.ts` 和它们的测试；Task 5 `use-update-project-details.ts` 和它的测试；Task 6 `use-feature-toggle.ts`、`use-feature-toggle.test.tsx`、两个自动化组件的测试；Task 7 `use-project-membership-changes.ts`、`project-roles.ts` 和它们的测试；Task 8 `add-project-members-modal.test.tsx`、`project-settings-member-defaults.test.tsx`；Task 9 `delete-project-modal.test.tsx`、`archive-restore-modal.test.tsx`；Task 10 `use-project-drop.ts` 和它的测试、`use-tab-preferences.test.ts`。它们都没有非空断言。`no-restricted-imports`（会话的取数不从 `swr` 导入值）不变：P10 没有新的取数文件，`useProjectFetch` 的改动在 P8b 已在范围里的文件中。
- `tools/keywords.json` 加一条规则 `project-leave`（Task 7，68 条、3 个例外）：web 应用的 `app`、`core` 中只有 `use-project-membership-changes.ts` 调 store 的 `leaveProject`（store 自己的定义和测试不算）：离开的顺序（M1-P4）和拒绝只写在 `leave` 里，第二个调用方就是第二份。`refusal-toast`、`sign-out-toast`、`workspaces-list-fetch` 已看 P10 的页面（P10 的每个失败提示都经 `useRefusalToast`）。复制没有加规则（第 3 节第 8 条）。
- 变异（附录 A.2）：W12 一类，每个 Task 加的一组各一个（在其中一个文件里写一个非空断言），`check:lint` 因为那条规则失败；`T7.13`（离开弹窗直接调 store 的 `leaveProject`），关键词守卫失败。

## 3. 与设计的差异、补充和需要裁定的

标"裁定"的 7 条由控制者裁定（R1–R7，2026-10-10，`$M3TMP/p10-architect-rulings.md`；R6 和 R7 由预检之后的裁定改定）；其余是本 Phase 在设计之内做的决定和理由，控制者照写接受。预检之后的修订在本节末尾的"预检之后"。

1. **Task 的切分**（裁定 R1：接受）：设计 12 节 P10 的草稿是 15 个任务，plan 是 11 个（P7、P8、P9 的先例：在上限之内重新切分）。对应：设计 1（项目列表、卡片、加入）→ Task 3；2（已归档的项目页）→ Task 3（包装层的已归档界面）、Task 9（那里的恢复、删除）；3（创建）→ 4；4（`ProjectAuthWrapper`）→ 3；5（general）→ 5；6（members）→ 7（改角色、移出、离开）、8（添加、负责人、默认负责人、访客可见全部）；7（features、automations）→ 6；8（离开的顺序、会话核对、删除项目组件的 vitest）→ 7（离开）、9（删除）、4（创建）和各页面的 hook（2.13）；9（侧边栏的顺序、导航偏好）→ 10；10（下拉框）→ 2；11（复制）→ 3；12（表情选择器）→ 1；13（端到端）并进各页面的 Task（测试与代码同一个提交，每个 Task 之后 `make e2e` 通过）；14（C6–C8）和 15 的评审是控制者的，15 的 3.20 是 Task 11（只改文档）。11 个 Task 都在约 16 个之内，每个在约 1,500 行之内：最长的 Task 3 是 1,478 行（附录 A.6、A.9）。
2. **成员下拉框以 `CustomSearchSelect` 为底，`member-options.tsx` 删除**（裁定 R2：接受；设计 7.7 原来写"`member-options.tsx`：popper 的 ref 放在列表元素本身，有搜索框的用 `Combobox`、打开时输入框取得焦点"，9.7 的 C7 和 13.1 写"`member-options` 的列表在按钮旁"）：`member-options.tsx` 是带搜索的选择的第二份实现（自己的 `Combobox`，自己的打开状态，外面还有一层 Popover 的），Task 2 让 `CustomSearchSelect` 做到的正是 7.7 要它做的（Popover 的按钮、面板里的 `Combobox`、popper 的 ref 在面板本身、打开时搜索框取得焦点）。在原处修它就是同一件事的两份实现（brief：没有重复的逻辑）。成员下拉框的 props 不变（`types.d.ts` 只删去没有人传的 `button`），它在 M4–M6 页面上的调用方（工作项的指派人、迭代和模块的负责人等）照旧编译、照旧渲染，对它们的约定照旧（推迟渲染、点击不冒泡、手机上不取焦点、列表离窗口边缘 12 像素：预检的 A-M5，第 15 条）。设计 7.7 的这一句、9.7 C7 和 13.1 的说法在本次修订中改为"成员下拉框以 `CustomSearchSelect` 为底，列表在按钮旁"，C7 和 13.1 的调用方逐个写明（M3 的 6 个，M4–M6 经成员下拉框的 14 个文件和页头的 9 个；预检的 L1）。
3. **嵌套的按钮留着**（裁定 R3：接受，交给 M4）：`dropdowns/` 的每个下拉框都把 `DropdownButton`（一个按钮）放在 Headless UI 的按钮里（Plane 的写法），HTML 不许按钮里有按钮。P10 让成员下拉框照它们的样子：选择自己的按钮 `tabIndex={-1}`，Tab 只停在里面的按钮，Enter、空格由外面的按钮打开列表（P1 的两个页面版本核对：变异 `T2.6`、`T2.8`、`T4.10`）。把 `DropdownButton` 改成不是按钮的元素会改到 `dropdowns/` 下全部的下拉框（状态、优先级、日期、迭代、模块、标签……都在 M4–M6 的页面上），不在 P10；P10 手改的添加成员弹窗不再嵌套（成员的选择用 `<span>`，Task 8；Tab 能到它，P5，`T2.9`）。M4 随工作项的属性下拉框一起改它（第 5 节 M4 一行）。
4. **项目的几份副本：显示 nerve 最后回答的那一份**（裁定 R4：接受；P8b 的 F-3"一般的规则由 P10 与架构一起定"；总体设计 7.7）：store 中一个项目有三份副本（项目自己的读取、工作区未归档和已归档的列表）。规则：`getProjectById` 给出 nerve 最后回答的那一份；store 确认的修改照旧写进每一份，所以哪一份胜出只在两次取数之间有不同。实现扩展 `reconciled.ts`（不复制）：模块中一个只增的计数，每次写入取数的回答取下一个号（`Reconciled.answeredAt`、`ReconciledByKey.answeredAt(key)`）。没有选的两种：列表到了就丢掉自己的读取（不是成员的人看得到的公开项目只有自己的读取，列表不一定有它）；按 nerve 的 `updated_at`（角色 `member_role` 的变化不改项目的 `updated_at`，F-3 的例子正是角色）。代价：一个模块级的计数（各代的 store 共用，只增；换代之后旧 store 被丢弃，跨代的先后不起作用）。守着它的是 `project.store.test.ts` 的两个测试（变异 `T3.10`、`T3.11`）。总体设计 7.7 在本次修订中加一句（裁定 R4，文字照预检的 L2）："一个实体在 store 中有几份副本（自己的读取、所在的列表）时，显示 nerve 最后回答的那一份（`reconciled.ts` 的 `answeredAt`）；store 确认的修改写进持有它的每一份（nerve 不改已归档的项目，修改不写已归档的列表）"。
5. **只为使用方改到的 Plane 文件变长一行**（裁定 R5：接受；P9 的裁定 P6 的先例）：`@nerve/propel` 的 `emoji-icon-picker/index.ts` 11 → 12 行：导出 `EMOJIBASE_URL`，web 应用的 `emojibase.test.ts`（9.5 的守卫）读它，核对选择器的地址与构建写出的相同。其余变长的 Plane 文件都是本 Phase 的对象（行为改在它们里面，附录 A.6）。
6. **被拒绝时弹窗留着，原因在共用的拒绝提示里；成功之后才关上**（裁定 R6：接受；预检之后的裁定 A-M2 扩到每个弹窗；设计 7.6 原来写"失败时留在原页、弹窗显示错误"，本次修订改为"失败时留在原页，弹窗留着，原因在共用的拒绝提示里；成功之后才关上、跳转"）：离开有三个入口（成员页自己一行的确认框、侧边栏和项目页头的离开弹窗），都经 `useProjectMembershipChanges` 的 `leave`（关键词规则 `project-leave`，2.14），拒绝经 `useRefusalToast`（关键词规则 `refusal-toast`：失败提示只写在一处）。P10 的每个弹窗一条规则：nerve 拒绝时弹窗留着、按钮重新可用，原因在提示里，页面不动；弹窗只在成功的跟进中关上（经 `followInSession`）：加入、创建、移出和离开的确认、离开、添加成员、自定义时长、删除、归档和恢复。成员页的确认框原来在 `await onSubmit()` 之后不论结果都关（被拒绝的离开也关），现在由 `remove(userId, done)`、`leave(done)` 的 `done` 关上（2.7）。用户仍在项目里，页面与 nerve 一致（7.6 要的）；原因显示在提示里而不是弹窗里。P9 的停用弹窗把原因写在弹窗里（P9 第 3 节第 3、4 条），那是一个入口、一个弹窗。P5 的页面版本核对确认框和侧边栏的离开弹窗被拒绝时都留着（变异 `T7.15`、`T7.16`、`T7.17`）。
7. **general 页按项目挂载一个表单**（裁定 R7；预检之后的裁定 A-M1 取 (b′)）：`ProjectDetailsForm` 不再在 `useEffect` 中 `reset`，页面以 `key={project.id}` 让每个项目一个表单。应用中没有从一个项目的设置页直接到另一个项目的设置页（同一个路由组件不卸载）的路：命令面板的'打开项目'、项目搜索和两个项目切换都去工作项页，设置的命令只到当前项目的设置；另外，包装层在 B 的读取还没有缓存时显示加载，页面重新挂载。所以今天 `key` 守着一条用户走不到的路，它守的是之后第一个加在设置页上的项目切换：没有它时项目 B 的设置页显示项目 A 的值，提交会把 A 的值写进 B。P3 的页面版本（Task 5）在同一个路由里移动，代替那个切换（`moveWithinApp`：`history.pushState` + `popstate`，React Router 照它移动）：Web 的设置页、Ops 的、再回 Web 的（Web 的读取已在会话的缓存里，路由一直挂载着），显示的是 Web 的值；去掉 `key` 的变异 `T5.7` 由它发现（附录 A.2）。
8. **复制只有一处**（W17）：项目的四处复制、P9 的邀请行和工作区 general 页都经 `useCopyLink`。工作区 general 页的成功提示原来标题是"Workspace URL copied to the clipboard."，现在标题是 `common.link_copied`（"Link copied!"），那句话是说明（与邀请行相同的形状）。项目的四处的提示也成了同一个样子（Task 3 的修正轮，裁定 T3-g；第 15 条）：成功是 `SUCCESS`，标题 `common.link_copied`，说明 `project_link_copied_to_clipboard`；失败是共用的错误提示。不加关键词规则：web 应用中另有 21 个文件自己写 `copyTextToClipboard`、`copyUrlToClipboard`（工作项、迭代、模块、视图、收集箱、Webhook 的页面，M4–M8；M2 的 API 令牌弹窗；命令面板的"复制当前页面的地址"），规则要为每个写一个例外；它们在各自的页面对接时改用这个 hook（第 5 节）。
9. **已归档的项目页对是工作区管理员的项目成员也提供恢复、删除**（W17；设计 3.4）：Plane 的卡片只看项目角色（项目管理员）；项目设置提供归档的条件是 `allowPermissions([ADMIN], PROJECT, …)`（是工作区管理员的项目成员也算），nerve 3.4 对归档、恢复、删除用同一条规则。原来他能在设置页归档，却不能在已归档的项目页恢复。P4 的页面版本由这样的人恢复（变异 `T9.11`）。
10. **添加成员的角色不按调用者的项目角色筛**（W17；P5b review 的 I1，负责人取 (a)）：Plane 只提供不高于调用者项目角色的（`role > (currentProjectRole ?? GUEST)` 不显示）：不是项目成员的工作区管理员只能加访客，是工作区管理员的项目成员最多加成员。nerve 的添加没有相对规则（P5b spec 第 7 节），只按被添加的人的工作区角色（3.5）。P10 提供 nerve 允许的（`addableRoles`），不把隐藏当作保护（P5b review 第 6 节）。项目管理员本来就能给任何角色，对他没有可见的不同。
11. **成员的默认值显示 nerve 的回答**（W17）：Plane 先改表单再发，被拒绝时仍显示被拒绝的值；P10 只显示 nerve 所持的值，修改之后由 store 写入回答（总体设计 7.7）。负责人、默认负责人一次只发一个字段（P8b 起）。
12. **创建表单的正的 `tabIndex` 留着**：Plane 的创建表单用 `getTabIndex(ETabIndices.PROJECT_CREATE)` 给每个字段正的 `tabIndex`；P10 不改这个写法（工作项、迭代、模块、视图的表单用同一个，M4、M6 的），只让负责人留在它的位置：Task 2 的成员下拉框把 `tabIndex` 交给 Tab 到达的按钮（`dropdowns/buttons.tsx` 收 `tabIndex`），没有它时负责人在表单的每个按钮之后（`tabIndex` 为 0 的元素排在所有正的之后）。P1 的页面版本从取消按 Shift+Tab 到负责人（变异 `T2.8`、`T4.10`）。
13. **e2e 的 fixture**：P3、P5 的故事和 `assert/project.ts` 加了页面版本之后超过约 400 行，原来各自写的经 API 的修改和读取移到共用的 fixture（`api.ts`、`assert/project.ts`、新文件 `project-pages.ts`，2.8）；P1 的时区不变，W3 的键盘选时区移进 `workspace-pages.ts` 的 `pickTimeZone`（Task 2，W3 原来 397 行）。预检之后 P3 加了同一路由内的移动：移动本身是 `settings-pages.ts` 的 `moveWithinApp`（Task 5），P3 经 API 被拒绝的添加改用 `projectMemberWrites`（Task 8），P3 是 399 行；P5 加了侧边栏的拒绝和 Tab 到成员的选择，396 行（附录 A.6）。
14. **删除项目之后不等跳转**：删除的弹窗先关、再跳转（`void navigate`），与 P9 的 `DeleteWorkspaceModal` 相同（P9 的裁定 F-10 的第 4 条：弹窗已关，没有看得见的忙碌的窗口）。归档、恢复的弹窗不关就跳转，所以交回导航的 `Promise`，忙到跳转做完（P9 的 F-2）。
15. **页面上看得到的不同**（W17；前端改动清单的八行，2.11）：
    - 会话：项目一侧的修改（2.13）在另一个标签页换了账户之后兑现时，页面不跳转、不提示（原来照样跳转、提示，显示在那个账户的页面上）。
    - 错误：加入、创建、general 页的修改和标识的检查、功能和自动归档的开关、成员页的三个修改、添加成员、成员的默认值、删除、归档、恢复、侧边栏的移动、标签栏被拒绝时，都说 nerve 的原因（原来读 `err.error`、Plane 的字段码、显示固定的文字，或只进控制台）；创建和 general 页中名称、标识的原因在字段下方；被拒绝的修改结束加载。唯一的管理员离开时说明原因（原来固定的文字）。
    - 弹窗：加入、创建、移出和离开的确认、离开、添加成员、自定义时长、删除、归档和恢复，请求在路上时关不掉（原来可以取消、按 Escape，回答之后照样跳转、提示）；创建表单页头的关闭按钮同样。自定义时长 nerve 做完才关（原来发出就关）。被拒绝时弹窗都留着、按钮重新可用（原来成员页的确认框被拒绝时也关上，第 6 条）。
    - 表情选择器显示表情（原来 CSP 挡住 CDN，选择器是空的）；图标按钮有名字。
    - 下拉框：带搜索的选择 Tab 能到、键盘打开，列表在按钮旁，搜索框有焦点，Escape 关上列表而不关弹窗（输入法组字时的 Escape 留给输入法，预检的 L7），再打开没有搜索；成员下拉框的列表在按钮旁（原来在页面左上角），Tab 只停一次。面包屑的下拉框一节一个 Tab 停留点（预检的 L9：选择改成 Popover 的按钮之后成了两个）：最后一节的标题改为 `<span>`；之前一节的标题仍是选择的按钮里的按钮，`tabIndex={-1}`，点它去那一节的页，键盘从侧边栏到那里。它留作按钮的理由：只能用鼠标的 `<span>` 加点击，jsx-a11y 标出它（`click-events-have-key-events`、`no-static-element-interactions`）；放到选择的按钮旁边成为另一个控件，会改 M4–M6 页头的布局（第 5 节 M4 一行）。成员下拉框对 M4–M6 的调用方照旧（预检的 A-M5）：调用方推迟时（`renderByDefault` 为 false，桌面上列表的每一行）指针经过之前只有按钮，没有选择和 popper；按钮的点击不冒泡到行，由 Headless UI 的 `Popover.Button` 自己做（它先 `preventDefault`、`stopPropagation`，再切换；我们的处理器若 `preventDefault`，Headless UI 的 `mergeProps` 就跳过它自己的切换，所以不另写）；手机上打开时搜索框不取得焦点；列表离窗口边缘 12 像素。按调用方看得到的不同有两处（第 5 节 M4 一行）：按键由 Headless UI 处理（Enter、空格打开，Escape 关上，Tab 离开），代替原来的 `useDropdownKeyDown`（预检的 A-M5 第 5 项）；`CustomSearchSelect` 在列表打开时记下调用方的 `onOpen`、`onClose`（`WhileOpen`），列表开着时调用方换掉的回调在这一次关上时不用（预检的 L8；现在的调用方都传 setter 或不变的回调）。Task 2 的修正轮（裁定 T2-a–T2-c、T2-f）：调用方的 `optionsClassName` 是 popper 定位的面板的，调用方给的宽度和 z-index 照旧有效（M4 的三处行给成员下拉框的列表 `z-10`、`z-[9]`）；选择的按钮由键盘取得焦点时显示一圈轮廓（Headless UI 的 `data-focus`，只在 `CustomSearchSelect` 一处），去掉轮廓的调用方（面包屑、项目页头、成员下拉框）也看得出焦点；成员下拉框的根照旧在行中可以收缩，选择的按钮没有自己的悬停底色。输入法组字时的 Escape 也不传出列表：列表和外面的弹窗都留着（修正之前它传到 `document`，创建弹窗的 `useKeypress` 关上弹窗）。成员下拉框的列表换成 `CustomSearchSelect` 的样子（裁定 R2 接受）：没有 `shadow-raised-200` 的阴影，边框是 `border-subtle-1`、`rounded-md`（原来 `border-strong`、`rounded-sm`），搜索框没有 `bg-surface-2` 的底色，选项没有选中的 `text-primary`、未选中的 `text-secondary`，停用的成员一行 `opacity-60`。
    - 复制：失败时提示（原来是未处理的拒绝）；卡片的复制按钮有名字；工作区 general 页的成功提示的标题（第 8 条）。项目的四处的提示经一个 hook 成了同一个样子（Task 3 的修正轮，裁定 T3-g）：卡片和项目页头菜单的成功原来是 `INFO`，标题"Link Copied!"（页头"Link copied!"），说明"Project link copied to clipboard."，现在是 `SUCCESS`，标题 `common.link_copied`（"Link copied!"），说明 `project_link_copied_to_clipboard`（"Project link copied to clipboard"）；两个侧边栏的成功的标题原来是 `link_copied`（"Link copied"，这个键随之删除），现在多了"!"；项目页头菜单的失败原来是"Copy failed"、"We couldn't copy the link. Please try again."，现在是共用的错误提示（`toast.error` 的"Error!"、`something_went_wrong_please_try_again`）。
    - 包装层：已归档的项目，它的成员看到已归档的界面（原来显示项目的页面，取四个子资源，工作项页另有 M4 的 404）；看得到而不是成员的也看到已归档的界面（原来是加入界面）。
    - 创建：负责人的候选不含访客和成员关系已结束的人；标识输入时转大写（原来留小写，nerve 拒绝）；没选负责人时不发 `project_lead_id`。
    - general 页：标识被占用时在标识下说，什么都不发；标识的检查被拒绝时说原因、按钮结束加载（原来是未处理的拒绝，一直加载）。
    - 成员页：角色的名字随界面的语言（`ROLE_DETAILS`，英文仍是 Guest、Member、Admin）；只提供 nerve 允许的角色和移出（原来不是工作区管理员的项目管理员也看到"设为管理员"、能改另一位管理员，nerve 拒绝；工作区管理员只能改自己的角色）；添加的候选不含已是成员和成员关系已结束的人；添加的角色（第 10 条）；离开先等 nerve（M1-P4）；成员的默认值（第 11 条）。
    - 已归档的项目页：恢复、删除的提供者（第 9 条）；删除的图标按钮有名字。
    - 标签栏：取到之前页头不提供修改（原来提供，点了提示失败，P8b 的 T5-a）。
16. **e2e 的记录各只有一份**：页面加载时的请求清单由 S2 的 `REQUESTS` 断言（P8b 的裁定 D5，不变）；它的列表移进新的 fixture `e2e/fixtures/mounts.ts`（Task 3），P2 的页面版本读同一份 `PROJECT_MEMBER`（不是成员的人和已归档的项目不取的四项），用 `watchPage` 的 `apiRequests`，不另写清单和记录器（Playwright 不许一个测试文件导入另一个，所以只能放进 fixture）。发往页面之外的源的请求同样只记一处：`watchPage` 的 `requestUrls` 和 `requestsElsewhere`（Task 1），S2 原来自己记的一份删除，P1、P3 的页面版本用它。P9 spec 2.12 的"S2 的 `REQUESTS` 是挂载清单唯一的地方"照旧成立：每个账户每一页的清单仍只在那里。

**交接到 P10 的事项的落点**（brief 的"Carried into P10"）：

| 来源 | 事项 | 落点 |
|---|---|---|
| P9 spec 第 5 节、P9 review 第 6 节 P10 一行 | 项目成员页读 `err.error`（`member-list-item.tsx:60`、`:68`，`project/settings/member-columns.tsx:152`） | Task 7（`useProjectMembershipChanges`，拒绝经 `useRefusalToast`） |
| | `navigation/use-tab-preferences.ts` 的拒绝进 `console.error` | Task 10 |
| | `ProjectAuthWrapper` 中已归档项目的界面 | Task 3 |
| | 删除、离开、创建项目之后的会话核对，照 `followInSession` 写 | Task 9、7、4 和 2.13 的每一处 |
| | 修复轮的写法：F-1 弹窗关不掉、F-2 忙到跟进做完、F-4 `useRefusalToast`；e2e 的 `holdAnswer`、`sentHeld`、`enabledWithin`、`closedByEscape`、`deferred` | 每个弹窗（第 3 节第 15 条）和每个页面版本；`anotherTabSignsIn` 没有用到：P10 的会话切换由 vitest 守（2.13） |
| | M2 交接第 11 节的页面一侧（项目 general 页的时区） | Task 5（P3 的页面版本改时区）：关闭 |
| P8b spec 第 5 节、P8b review 第 6 节 P10 一行 | 已归档项目的界面；删除之后的会话核对；下拉框、复制、表情；创建弹窗的页面版本 | Task 3；9；2、3、1；4 |
| | P5 读不到 nerve 的原因：成员页三处（另有 `console.log`）、离开弹窗的固定文字、加成员只进控制台、创建和设置读 Plane 的字段码、`form.tsx:153` 的检查不在 `try` 之内 | Task 7（成员页、离开）、8（加成员）、4（创建：`projectRefusal`）、5（设置：检查在 `followInSession` 的修改之内） |
| | P6 `form.tsx:154` 的 `!available` 一支 | Task 5（`T5.1`，vitest 和 P3 两层） |
| | P7 标签设置页的拖动把手 | P11：标签设置页是 P11 的（设计 12 节 P11 的任务 2），把手的登记在那里随页面改（第 5 节） |
| | P9 三个标签选择器的取标签被拒绝 | M4：都在工作项的页面上，与 P8b 交给 M4 的状态、成员下拉框的按需取数同一类（第 5 节） |
| | F-3 项目的两份副本谁先 | Task 3（第 3 节第 4 条，裁定 R4） |
| | T4-b `project.sole_admin` 从不显示 | Task 7（P5 的页面版本核对原因） |
| | T5-a 标签栏取到之前藏起控件 | Task 10（`T10.4`） |
| P8a spec 第 5 节、review 第 6 节 P10 一行 | 项目封面的删除；项目成员角色下拉框发出字符串 | 已在 P8b 做了（P8b review 第 6 节） |
| | 下拉框、复制、表情选择器 | Task 2、3、1 |
| M2 收尾交接第 10 节 | 表情选择器的数据 | Task 1：关闭 |
| 第 14 节 | 下拉框和复制 | Task 2、3：代码一侧；关闭条件的 `member-options` 一项改为成员下拉框（Task 11）；C7、C8 写进评审之后关闭 |
| M1-P4 交接 | 离开项目的顺序 | Task 7（vitest 和 P5，`T7.1`） |
| M1-P3 交接 | 检查标识的地址随项目页面改调 | P8b 已改调新接口；general 页经它（Task 5），P1 的检查不发 `.`（下一行） |
| P4a review 的 P10 | 标识的实时检查不发 `.`、`..` | Task 4（`projectIdentifierSanitizer` 不留 `.`，`@nerve/utils` 的测试 `..`、`w.e.b` 两行） |
| P4a、P4b spec | P1–P4、P8 的页面版本 | Task 4、3、5/6/8、9、10 |
| P5a spec 第 7 节 | M1-P4 离开的顺序 | Task 7 |
| P5b spec 第 5 节、P5b review 第 6 节 | P5 的页面版本；离开先等接口；页面只给调用者能做的；403、409、422 照文案表；隐藏不是保护 | Task 7、8（`roleChoices`、`canRemove`、`addableRoles`；第 3 节第 10 条） |
| P1、P2、P6 的 review 和 spec，P7a review | 其中的"P10"是各自的搁置项编号（如 P7a 的 T8-a 搁置 P10），不是本 Phase | 没有落点 |
| Codex 设计评审 4.3 第 1 条 | 包装层给不是成员的人只取项目 | P8b 的 hook；页面版本：P2（加入界面只读项目）、S2；已归档的同样只读项目（Task 3） |
| Codex 设计评审 4.3 第 2 条 | 迟到的成功回答核对会话 | 2.13 的每一处；删除项目组件的 vitest（9.5，Task 9） |
| Codex 设计评审 5.3 的 M1-P4 一行 | 离开先成功后跳转；成功之后的会话检查 | Task 7（`leave` 经 `followInSession`） |
| Codex 设计评审 5.3 的表情一句 | 本站路径可行，没有在浏览器中验证 | Task 1：vitest 和 P1 的页面版本（真实的浏览器，没有外部请求、没有 CSP 违规） |
| 设计 7.7 | `webui` 对 `HEAD` 的回答 | 原型核对（2.1、附录 A.3）；写进评审是控制者的 |
| 设计 8.5 | 页面不写、不按 `logo_props.emoji.url` 请求图片 | Task 4（`logoPropsOf` 只留值；web 中没有读它的地方） |
| 设计 9.5 | P10 的三个 vitest：删除项目的组件、离开失败时不跳转、表情的地址和版本 | Task 9（`T9.1`）、7（`T7.1`、`use-project-membership-changes.test.ts` 的拒绝一行）、1（`T1.3`） |
| M1 收尾（7.9，R3） | 手改的文件 0 条 oxlint 警告；上限调低 | 每个 Task；附录 A.5 |
| 3.20 | README"前端"一节、前端改动清单 | Task 11 |

**预检之后**（预检 `$M3TMP/p10-preflight/preflight.md`：0 高、5 中、10 低；控制者的裁定 `$M3TMP/p10-amend-brief.md`，除下表写明的都取预检给的修法；修订一次，之后从 `6c1a090a` 重新逐 Task 复现，附录 A.10）：

| 编号 | 预检的发现 | 落点 |
|---|---|---|
| A-M1（M1，裁定 R7） | 照 R7 写不出经命令面板的测试：应用中没有从一个项目的设置页到另一个项目的设置页而路由不卸载的路；第 7 条"只有命令面板一条路"不对 | 取 (b′)：P3 的页面版本在同一个路由里移动（`moveWithinApp`，`history.pushState` + `popstate`，放进共用的 `settings-pages.ts`；Task 5，2.5）；去掉 `key` 的变异是 `T5.7`，由 P3 发现（预检的 PF18）；第 7 条、第 6 节、附录 A.2 照预检的文字改；P3 399 行（第 13 条） |
| A-M2（M2，裁定 R6） | 成员页的确认框被拒绝时也关上；离开弹窗被拒绝时留着没有检查（PF19 存活） | 取代码的一种：每个弹窗被拒绝时留着，只在成功的跟进中关上（第 6 条）。确认框由 `remove(userId, done)`、`leave(done)` 的 `done` 关上（Task 7，2.7），离开弹窗 `await leave(handleClose)`；P5 被拒绝的离开核对确认框还开着（`T7.16`）；侧边栏的离开弹窗加被拒绝的一步（`T7.15`，PF19）：侧边栏和项目页头只给访客提供离开，所以照预检的写法由 pat 从侧边栏离开走不通，被拒绝的是访客 ann（她的弹窗开着时 pat 经 API 结束她的成员关系，nerve 答 403），pat 加回她之后照旧扣住成功的一次；确认框自己的变异是移出成功时不关（`T7.17`，vitest 和 P5）；2.13 的"不跟进的"没有确认框的例外；M3 设计 7.6 本次修订改为"失败时留在原页，弹窗留着，原因在共用的拒绝提示里；成功之后才关上、跳转"，9.7 C6 的离开加上"弹窗留着" |
| A-M3（M3） | 创建在途时页头的关闭按钮可用（PF6 存活） | P1 核对它不可用（Task 4，2.4）；`T4.11`（PF6） |
| A-M4（M4） | 带搜索的选择用自定义按钮时出了 Tab 顺序没有检查（PF15 存活） | P5 用 Tab 到添加弹窗的成员选择，Enter 打开、搜索框有焦点（Task 8，2.8）；`T2.9`（PF15，Task 8 起） |
| A-M5（M5） | 成员下拉框改写之后，M4–M6 的调用方得到的不同（推迟渲染、点击冒泡、手机上的焦点、popper 的边距、按键）没有写出 | 对调用方的约定照旧（Task 2，2.2，第 15 条）：1. 推迟时指针经过之前只有按钮（成员下拉框 `useState(renderByDefault)`，`T2.10`）；2. 点击不冒泡：Headless UI 的 `Popover.Button` 自己 `preventDefault`、`stopPropagation`，不另写（我们的 `preventDefault` 会让它跳过切换），只由评审看住；3. 手机上不取焦点（`CustomSearchSelect` 的 `focusSearchOnOpen`，`T2.11`）；4. 离窗口边缘 12 像素（`popperModifiers`，`T2.12`）。`T2.10`–`T2.12` 由新文件 `dropdowns/member/base.test.tsx`（4 个测试）发现；只由评审看住的几项和理由在附录 A.2。第 5 项（Headless UI 的按键）和 L8 写进第 15 条和第 5 节 M4 一行 |
| L1（裁定 R2 的文字） | M3 设计 9.7 C7 和 13.1 的调用方数和 `member-options`；M2 交接第 14 节的关闭条件 | M3 设计 7.7、9.7 C7、13.1 本次修订改为"成员下拉框以 `CustomSearchSelect` 为底，列表在按钮旁"，调用方逐个（M3 的 6 个，M4–M6 经成员下拉框的 14 个文件和页头的 9 个）；Task 11 的 M2 交接第 14 节的处理结果说关闭条件的 `member-options` 一项改为成员下拉框。`docs/v0` 中其余写 `member-options` 的是记录（M2 的 spec、plan、评审，M3 的 P8a，M2 交接第 14 节的问题本身）和本 Phase 的 spec、plan（说它删除），不改 |
| L2（裁定 R4 的文字） | "写进每一份"字面上不对：已归档的列表不写 | 总体设计 7.7 加的一句照预检的文字："持有它的每一份（nerve 不改已归档的项目，修改不写已归档的列表）"（第 4 条） |
| R3 的第 5 节 | M4 一行没有写 HTML 的理由和"M4 随工作项的属性下拉框改它" | 写上（第 5 节） |
| L3 | `T2.8` 的标记应是"（Task 4 起）" | 改了；plan 的表中只在之后的 Task 起才被发现的三个变异都写明从哪个 Task 起（`T2.8` Task 4、`T2.9` Task 8、`T2.14` Task 3），各在那个 Task 的树上核对过（附录 A.2） |
| L4 | P8 的 `toPass()` 没有期限 | `toPass({ timeout: 5_000 })`（Task 10）；附录 A.4 第 46 条列出 `toPass` |
| L5 | 2.13 的"不跟进的"没有列出确认框的关闭 | 由 A-M2 代替：确认框只在成功的跟进中关上，"不跟进的"只剩组件自己的忙碌（2.13） |
| L6 | plan 的 Task 2 写了不存在的 `optionsFilter` | 删去 |
| L7 | 输入法组字时的 Escape 关上列表 | 面板的处理先看 `event.nativeEvent.isComposing`（Task 2，2.2）；P1 在搜索框里发一个组字的 Escape，列表留着、还能输入（`T2.13`） |
| L8 | `WhileOpen` 在列表打开时记下 `onOpen`、`onClose` | 第 15 条和第 5 节 M4 一行写明 |
| L9 | 面包屑的下拉框有两个 Tab 停留点 | 一个（Task 2，2.2）：最后一节的标题是 `<span>`，之前一节的标题按钮 `tabIndex={-1}`；P2 核对项目页头的项目一节只停一次（`T2.14`，Task 3 起）。之前一节仍是按钮里的按钮，理由在第 15 条和第 5 节 M4 一行 |
| L10 | Task 11 的两行说的比代码多 | "成员下拉框（创建项目的负责人；M4–M6 的指派人、负责人）"、"复制失败（浏览器拒绝时）"（2.11） |

## 4. 验收标准（完成线，M3 设计 12 节 P10）

- [ ] P1–P5、P8 的页面版本、S1、S2 和此前的全部故事通过：worktree 中 `make e2e` 106 个全部通过（`6c1a090a` 是 93 个）；`make test-web` 通过（web 应用 87 个测试文件、761 个测试，附录 A.1）。
- [ ] 打开表情选择器时没有外部请求、没有 CSP 违规（P1 的页面版本，`watchPage` 和 `requestsElsewhere`）。
- [ ] 9.5 中 P10 的 vitest 通过：删除项目的组件（`delete-project-modal.test.tsx`）、离开失败时不跳转（`use-project-membership-changes.test.ts`）、表情的地址和版本（`emojibase.test.ts`）；每个都有一个发现它的变异，P10 写或改的每个测试都至少被一个变异发现（W18，附录 A.2）。
- [ ] 2.13 的每一处有一个"不经 `followInSession` 时失败"的 vitest（附录 A.2 的 W3 一类）。
- [ ] 每个弹窗请求在路上时关不掉，被拒绝时留着，只在成功的跟进中关上（第 3 节第 6 条；附录 A.2 的两类）。
- [ ] 成员下拉框对 M4–M6 的调用方照旧：推迟渲染、手机上不取焦点、列表离窗口边缘 12 像素（`dropdowns/member/base.test.tsx`）；面包屑的下拉框一节一个 Tab 停留点（P2）。
- [ ] P2 的页面版本：不是成员的人直接打开公开项目的地址，只请求项目本身，没有失败的请求；已归档的项目同样（S2）。
- [ ] `node tools/keywords.mjs` 通过：68 条规则（P10 加 `project-leave`）、3 个例外，没有命中（`make lint-web`）；`tsc`、knip 通过。
- [ ] 根目录 `.oxlintrc.json` 的范围（2.14）：P10 新建的模块和改到的 nerve 测试在 `no-non-null-assertion` 的范围里；每一组各有一个变异让 `check:lint` 失败。
- [ ] 改到的文件按 7.9 没有 oxlint 警告：web 的 85 个 TS 文件都是 0 条；web 的上限 356 → 349，`propel` 16 → 11，其余各包不变；没有新的抑制（附录 A.5）。
- [ ] 没有新的 `as`、`any`、`!`（附录 A.7）。
- [ ] 逐 Task 复现：从 `6c1a090a` 照 plan 应用，每个 Task 之后门禁通过，最终与原型逐文件相同（附录 A.10）。
- [ ] 控制者的浏览器核对 C6–C8 写进评审（9.7；不在 plan 中），含 `webui` 对表情数据的 `HEAD` 的回答（2.1）。

## 5. 不在 P10 范围内

- **P11**：状态、标签的设置页（P8b 的 P7：标签设置页的拖动把手只因 `onDrop` 重建才重新登记，`project-setting-label-list.tsx:62`、`project-setting-label-item.tsx:77-78`、`:105-106`）；P8a、P8b、P9 留给 P11 的照旧。P10 没有只经机械步骤到达的文件，A.8 的清单不变。
- **M4**：`dropdowns/` 的下拉框的嵌套按钮（第 3 节第 3 条，裁定 R3）：`DropdownButton` 是一个按钮，放在 Headless UI 的按钮里，HTML 不许按钮里有按钮；M4 随工作项的属性下拉框一起改它。面包屑的下拉框之前一节的标题按钮也在选择的按钮里（`tabIndex={-1}`，第 3 节第 15 条），M4 改工作项的页头时一起改。成员下拉框以 `CustomSearchSelect` 为底之后 M4 的调用方看得到的两处（第 3 节第 15 条）：按键由 Headless UI 处理（Enter、空格打开，Escape 关上，Tab 离开），原来的 `useDropdownKeyDown` 没有了；`CustomSearchSelect` 在列表打开时记下调用方的 `onOpen`、`onClose`（`WhileOpen`），列表开着时换掉的回调这一次关上时不用。按钮的点击不冒泡是 Headless UI 的 `Popover.Button` 自己做的，升级 Headless UI 时重看；推迟渲染、手机上不取焦点、列表离窗口边缘 12 像素照旧（`dropdowns/member/base.test.tsx`）。成员下拉框中 Escape 关上列表之后，焦点在选择自己的按钮上（不在 Tab 顺序里，键盘取得焦点时有轮廓），不回到里面的 `DropdownButton`（原来落到 `body`）：M4 改嵌套的按钮时一起定。成员下拉框的列表换成 `CustomSearchSelect` 的样子（阴影、边框、选项的颜色、停用一行的透明度，第 3 节第 15 条），在 M4 的页面上核对。工作项页的三个标签选择器的取数被拒绝（P8b 的 P9：`issue-layouts/properties/label-dropdown.tsx:149-153`、`issue-detail/label/select/label-select.tsx:57`、`issues/select/dropdown.tsx:38`）；工作项页的 `user-properties` 的 404（2.12）；工作项页面的复制（第 3 节第 8 条：`issues/` 下 7 个文件、`power-k` 的工作项命令）；工作项、迭代、模块、视图的表单的正的 `tabIndex`（第 3 节第 12 条）；P8a、P8b、P9 交给 M4 的照旧。
- **M6**：迭代、模块、视图的复制（`cycles/quick-actions.tsx`、`modules/quick-actions.tsx`、`modules/links/list-item.tsx`、`views/quick-actions.tsx`、`workspace/views/` 的两个、`power-k` 的迭代和模块命令）；P9 交给 M6 的照旧。
- **M7**：收集箱的复制（`inbox/content/inbox-issue-header.tsx`）；P9 交给 M7 的照旧。
- **M8**：Webhook 的密钥的复制（`web-hooks/form/secret-key.tsx`）；P9 交给 M8 的照旧。
- **收尾**：
  - P10 改到的文件中仍写死的英文（Plane 原有，P10 没有改它们的文字）：成功的提示（删除项目、添加成员、功能的开关、标签栏的设为默认的"Success!"和说明，归档、恢复的"Archive success"、"Restore success"和说明）；删除弹窗的标题；添加成员的"Select co-worker"；离开弹窗的两条核对（"Please enter the project name as shown in the description."、"Please confirm leaving the project by typing the 'Leave Project'."）和它的正文；删除弹窗的正文；成员默认值的标题和说明（"Project Lead"、"Default Assignee"、"Guest access" 和它们的一句说明）；工作区 general 页复制的说明"Workspace URL copied to the clipboard."（P9 列过）。
  - 复制的另两处：M2 的 API 令牌弹窗（`api-token/modal/generated-token-details.tsx`）和命令面板的"复制当前页面的地址"（`power-k/config/miscellaneous-commands.ts`）自己写复制，没有失败的一支；全部改用 `useCopyLink` 之后，加一条关键词规则让复制只写在 hook 里（第 3 节第 8 条）。
  - 角色的名字：P10 的页面经 `ROLE_DETAILS[...].i18n_title`；P9 的工作区成员页和邀请行（`workspace/settings/member-columns.tsx:124`、`:133`、`:142`，`invitations-list-item.tsx:98`、`:104`、`:118`）和编辑器的提及卡片（`editor/embeds/mentions/user.tsx:75`）仍用英文的 `ROLE`，不是 P10 的页面；落点由收尾定。
  - `webui` 给嵌入的文件加 `ETag`（Go 的改动）：frimousse 就能在新的标签页会话中用 `HEAD` 确认缓存，不重新下载约 700 KB（2.1）。是否值得由收尾决定。
  - `@nerve/ui` 的 `dropdowns/helper.tsx` 的 7 个 `any`（`CustomSelect`、`CustomSearchSelect`、`CustomMenu` 的 `value`、`onChange` 等，调用方传各种值）和 `custom-search-select.tsx` 的 2 个（`comboboxProps`、`displayValue`）是 Plane 原有的，P10 重写这个组件时没有改它们的类型（给它们类型是这些组件的泛型，改到 13 个调用方；附录 A.7）。
  - M3 设计 §15 中 P10 的一行由评审的提交改。P10 让它们不再成立的设计的句子（7.6 离开弹窗的一句，7.7、9.7 C6、C7、13.1 中 `member-options` 的说法）在本次修订中改了（第 3 节第 2、6 条和"预检之后"）；之前各 Phase 的 spec、plan、评审中"P10 将…"的句子是记录，不改。

## 6. 风险

- **S2 的清单是一份要维护的名单**：P11、M4–M7 每在挂载路径上加一个取数，S2 都会失败，直到清单改过（P8b 的裁定 D5，这正是它守的性质）。
- **general 页的 `key`**（第 3 节第 7 条）：应用中今天没有走到它的路，守着它的是 P3 的同一路由内的移动（`moveWithinApp`），代替之后加在设置页上的项目切换。React Router 不再照 `popstate` 移动时这一步先失败。退路：同一路由内移动的页面版本（`history.pushState` + `popstate`）照路由的新写法移动；加了项目切换的 Phase 改由那个切换移动。
- **P8 的拖动用逐步的鼠标操作**：Playwright 的 `dragTo` 拖不动侧边栏的项目（指针离开时拖动柄就隐藏），P8 的页面版本按住柄、分几步移到目标上。pragmatic-dnd 的事件顺序变了时这一步会先失败（退路：照它的新顺序调整步数）；反复 3 次通过（附录 A.1）。
- **P1 的列表位置靠过渡结束**：popper 在列表打开时按按钮此刻的位置定位，弹窗的进场过渡移动按钮；故事先等过渡结束（`transitionsEnded`）。过渡的写法变了时 `expectListBesideButton` 会失败（退路：等它的新的结束条件）。
- **不是 git 仓库的副本里 S3 失败**（P4b spec F4）：复现在副本中运行，只有 S3 因为读不到提交而失败，由复现脚本单独认出；worktree 中 106 个都要通过。

## 7. 已知的限制、交接和关闭条件

- **关闭**：M2 收尾第 10 节、第 11 节（交接的这两节至此关闭）；第 14 节在控制者的 C7、C8 写进评审之后关闭；M1-P4 交接的离开项目的顺序（交接在 M3 的收尾改为 `closed`）。P8b review 第 6 节 P10 一行（P7 交 P11、P9 交 M4，第 5 节）；P9 review 第 6 节 P10 一行；P8a 交给 P10 的行；P4a review 的 P10；P5b 交给 P10 的行；Codex 4.3 第 1、2 条和 5.3 的 M1-P4、表情两句的页面一侧。
- **交接**：第 5 节各行照录进 P11 的 spec 和 M4–M8 的交接（13.2）；第 3 节中裁定的结果写进 P10 的评审。
- **放不下的关闭条件**：没有。设计 12 节 P10 的"关闭"三节一项都有落点；C6–C8 和评审按 brief 是控制者的。
- **限制**：每次挂载都重取，聚焦时不取（P8a 的 F-1）；项目的几份副本在两次取数之间可能不同（第 3 节第 4 条）；新的标签页会话第一次打开表情选择器时重新下载数据（2.1）。

## 附录 A：原型验证记录（2026-10-10）

### A.1 方法与门禁

- 原型 `$M3TMP/p10proto` 是 `6c1a090a` 的 `git archive` 副本（`pnpm install --frozen-lockfile` 之后）。每个 Task 由一个脚本从上一个快照写出（`$M3TMP/p10tools/edit/T<n>.py`），照仓库的格式化（oxfmt）排版，`tsc` 通过之后存一份快照（`$M3TMP/p10snap/T1`…`T11`）。plan 的块由快照之间的差异写出（`mkblocks.py`：11 个 Task，0 个问题；它把每个 Task 的块依次应用到上一个快照，与那个 Task 的快照逐字节比较）。代码或测试改过时，从改动的那个 Task 起重新写出之后的每个快照（`chain.sh`），再重新写出块。预检之后的修订（第 3 节"预检之后"）改了 Task 2–8、10、11 的脚本，从 Task 2 起重新写出每个快照（修订之前的快照留在 `$M3TMP/p10snap-v1`）。
- 原型上每个 Task 写出之后 `make lint-web`、`make knip`、`make test-web` 都跑过，有页面或挂载路径改动的 Task 跑了它的故事和 `make e2e`；最后一次逐 Task 复现在每个 Task 的树上再跑全部门禁（A.10）。`6c1a090a` 上：`make lint-web`（67 条规则）、`make knip`、`make test-web` 通过，`make e2e` 93 个中 92 个通过（S3 在副本中失败，F4）。
- 最终原型上：`make lint-web` 通过（关键词守卫 68 条规则、3 个例外、没有命中）；`make knip` 通过；`make test-web` 通过（web 应用 87 个测试文件、761 个测试；`6c1a090a` 上是 75 个、657 个；`@nerve/utils` 加 1 个文件、6 行）；`make e2e`（含 `make build`）106 个中 105 个通过（S3，F4）。生成的文件与 `make gen` 的输出相同（0 处差异，`make gen-check` 的核对）。没有 Go 和接口描述的改动，`make lint-go`、`make test` 不需要（`git diff --stat 6c1a090a -- server/ api/` 没有输出）。
- 端到端的反复：P1、P2、P3、P4、P5、P8、S2、W3 各 3 次（31 个测试，93 次），都通过（`logs/repeat-final.log`）。

| Task | 改到的文件 | web 的测试文件 / 测试 | 端到端 |
|---|---|---|---|
| `6c1a090a` | – | 75 / 657 | 93 |
| 1 | 18 | 76 / 659 | 94 |
| 2 | 15 | 77 / 663 | 95 |
| 3 | 26 | 79 / 676 | 98 |
| 4 | 18 | 82 / 693 | 99 |
| 5 | 9 | 83 / 705 | 101 |
| 6 | 8 | 83 / 710 | 102 |
| 7 | 14 | 84 / 731 | 103 |
| 8 | 14 | 85 / 743 | 104 |
| 9 | 7 | 87 / 755 | 105 |
| 10 | 10 | 87 / 761 | 106 |
| 11 | 4 | 87 / 761 | 106 |

（测试数按 vitest 自己的计数，`it.each` 的每一行算一个（`vitestcounts.py`），只算 web 应用；Task 4 另加 `@nerve/utils` 的 `project.test.ts` 6 行。一个文件可以由几个 Task 改到，所以改到的文件之和大于 116。）

### A.2 变异

`mutants_p10.py`：106 个变异，每个只改一处或几处，`mut.py` 在最终原型上、在它写的每个检查上各跑一次：106 个都被发现，没有存活的；每个写的每个检查都让它失败（`$M3TMP/p10tools/mut-results.json`，日志 `mut-logs/`）。每个变异另在它自己的 Task 的快照上跑过（`mutpertask.py`，`mut-results-pertask.json`）：103 个在自己的 Task 的树上就被发现；3 个只在之后的 Task 起（`T2.8` Task 4 起、`T2.9` Task 8 起、`T2.14` Task 3 起，plan 的表中标出），各在那个 Task 的树上被发现（`mutfrom.py`，`mut-results-from.json`）。每个变异要改的文字在它的 Task 的快照上和最终原型上都恰好出现一次（`mutchecksnap.py`：0 个不适用；`T3.5`、`T3.7`、`T4.10`、`T6.7`、`T10.7` 要改的文字在 Task 1 的快照上已有，在它们自己的 Task 之前就适用，不影响结果）。"静态"是 `make lint-web` 的 `tsc`、oxlint（上限和 `.oxlintrc.json` 的范围）、关键词守卫，以及 `make knip`。Task 2 的修正轮（裁定 T2-b、T2-e、T2-f）加了 3 个（`T2.16`、`T2.17`、`T2.18`），在修正之后的 Task 2 的树上跑过，都被发现（`$M3TMP/p10mut/t2`），没有在最终原型上跑。Task 3 的修正轮（裁定 T3-c、T3-d、T3-e）加了 5 个（`T3.13`–`T3.17`），同样在修正之后的 Task 3 的树上跑过，都被发现（`$M3TMP/p10mut/t3`）。下面的表和计数含这 8 个，共 114 个。按缺陷类别（一个变异可以被几层发现）：

| 类别 | 变异 | 静态 | vitest | 端到端 | 存活 |
|---|---|---|---|---|---|
| 表情的数据（CSP，M2 交接第 10 节） | 4 | 0 | 1 | 4 | 0 |
| 下拉框（M2 交接第 14 节） | 13 | 0 | 0 | 13 | 0 |
| 成员下拉框对调用方的约定（M4–M6，预检的 A-M5） | 4 | 0 | 4 | 0 | 0 |
| 复制（M2 交接第 14 节） | 2 | 0 | 2 | 2 | 0 |
| W3：会话外的跟进（2.13） | 12 | 0 | 12 | 0 | 0 |
| W2：页面只取角色能读的 | 1 | 0 | 1 | 1 | 0 |
| 页面只提供 nerve 允许的（3.5） | 5 | 0 | 4 | 4 | 0 |
| M1-P4：离开的顺序 | 1 | 0 | 1 | 1 | 0 |
| 封闭的请求体（P8b 第一类） | 5 | 1 | 5 | 1 | 0 |
| 错误的读法（P8b 第二类） | 10 | 0 | 10 | 0 | 0 |
| 所持的值（P8b 第三类：几份副本、取到之前的修改） | 6 | 0 | 5 | 2 | 0 |
| 标识被占用的一支（P8b 的 P6） | 1 | 0 | 1 | 1 | 0 |
| 请求在路上时弹窗关得掉（P9 第一类） | 17 | 0 | 0 | 17 | 0 |
| 被拒绝时弹窗关上（预检的 A-M2：成功之后才关） | 4 | 0 | 1 | 4 | 0 |
| 忙碌在跟进之前结束（P9 第二类） | 4 | 0 | 1 | 4 | 0 |
| W17：页面上看得到的不同 | 12 | 0 | 7 | 10 | 0 |
| 一处实现（关键词规则） | 1 | 1 | 0 | 0 | 0 |
| 角色的名字（`ROLE_DETAILS`） | 2 | 0 | 2 | 0 | 0 |
| W12：非空断言的范围（2.14） | 10 | 10 | 0 | 0 | 0 |
| 合计 | 114 | | | | 0 |

按发现它的层：只有端到端的 46 个、只有 vitest 的 38 个、vitest 和端到端的 18 个、只有静态的 11 个、静态和 vitest 的 1 个。按 Task：Task 1 5 个、2 18 个、3 17 个、4 11 个、5 8 个、6 7 个、7 17 个、8 11 个、9 13 个、10 7 个；Task 11 只改文档，没有变异。预检的变异中修订针对的和离开的跟进改过之后要重跑的（`mutants_pf_amend.py`：PF1、PF6、PF7、PF13、PF15、PF18、PF19，写在修订后的代码上；PF6、PF15、PF18、PF19 就是 plan 的 `T4.11`、`T2.9`、`T5.7`、`T7.15`）在最终原型上：7 个都被发现（`mut-results-pf.json`）。

- **W3 的 12 个**：2.13 的每一处"不论会话都跟进"（`T3.1`、`T4.3`、`T5.4`、`T6.1`、`T6.3`、`T7.3`、`T8.6`、`T8.8`、`T9.1`、`T9.7`、`T10.1`、`T10.3`），每个由它自己的 vitest 发现（`lateSettlings`）；`T9.1` 是 9.5 的"删除项目的组件"。P10 没有会话切换的端到端（2.13）。
- **关不掉的弹窗的 17 个**（P9 的第一类）：每个弹窗的 Escape 和取消各一个（加入、创建、自定义时长、移出、离开、添加、删除、归档），和创建表单页头的关闭（`T4.11`，预检的 A-M3），都由它的故事扣住请求发现（`closedByEscape`、`enabledWithin`）：服务端渲染看不到之后才有的状态（P9 的先例）。
- **被拒绝时弹窗关上的 4 个**（预检的 A-M2，第 3 节第 6 条）：离开弹窗（`T7.15`，预检的 PF19）、成员页的确认框（`T7.16`）被拒绝时关上，分别由 P5 中被拒绝的两次离开发现；加入的对话框被拒绝时关上（`T3.14`，Task 3 的修正轮），由 P2 的卡片的故事中被拒绝的加入发现（`closedWithin`）；移出成功时确认框不关（`T7.17`：`remove` 不把 `done` 交给 `followInSession`），由 vitest 和 P5 发现。
- **成员下拉框对调用方的约定的 3 个**（预检的 A-M5）：推迟时也渲染选择和 popper（`T2.10`）、手机上打开时搜索框取得焦点（`T2.11`）、列表没有离窗口边缘的距离（`T2.12`），由 `dropdowns/member/base.test.tsx` 发现。
- **只有端到端的**：都是页面上的行为，P10 的页面在 vitest 中没有渲染它的那一部分（下拉框的焦点和位置、弹窗的关闭、卡片的按钮、Tab 的顺序）；每个由它的故事的页面版本发现。
- **W18（每个新的 vitest 都能失败）**：`mut.py` 只跑一个变异写的检查，所以另由 `w18p10.py` 把每个变异对 P10 写或改的全部 22 个 web 的 vitest 文件各跑一次，再对照最终的 vitest 报告：P10 新加或改写的 109 个测试（`it.each` 的每一行算一个）中，106 个变异中的 50 个让其中 67 个失败（其余的变异由它们的端到端或静态检查发现）。其余 42 个多是正向的行（例如"在会话里跟进""移出之后停在原页"、决定表中每一步的一行、成员下拉框不推迟时是选择），另写 31 个探查变异（`mutants_w18_p10.py`，`W18.1`–`W18.31`，只在最终原型上跑，不在 plan 的表中）：每个都让它守着的测试失败（`logs/w18-probes.log`）。31 个都被发现，109 个测试都至少失败一次（`w18split.py`）。改到的 10 个 P8 的测试文件中 P8 原有的 57 个测试不算（其中 9 个也被 P10 的变异发现）。`@nerve/utils` 的 `project.test.ts` 6 行在 web 应用之外：`T4.7` 让其中 5 行失败，`..` 一行由一个手工的探查变异（净化保留 `.`）发现（它让 `..`、`w.e.b` 两行失败，`$M3TMP/p10tools/logs/utilsprobe.log`）。Task 3 的修正轮加的 5 个测试（`use-project-fetch.test.ts` 决定表的 3 行、`project.store.test.ts` 的 2 个）各被它的变异发现：`T3.15` 让那 3 行都失败，`T3.16` 让那 2 个都失败，`T3.17` 让已归档的列表的一个失败（`$M3TMP/p10mut/t3/logs-fix1`）。
- **只由评审看住的**（没有会失败的检查的性质，各有理由；都不是会话、权限或取数的性质）：
  - 成员下拉框被推迟时，指针经过之后换成选择（`onMouseEnter`）：P10 的页面没有推迟它的调用方（推迟的是 M4 的列表的行），vitest 是服务端渲染，没有指针；
  - `CustomSearchSelect` 照 `focusSearchOnOpen={false}` 不让搜索框取得焦点：P10 的页面不在手机上跑（vitest 核对成员下拉框在手机上传了它，`T2.11`）；
  - `CustomSearchSelect` 把 `popperModifiers` 交给 popper：P10 的列表没有贴着窗口边缘的（vitest 核对成员下拉框传了它，`T2.12`）；
  - 成员下拉框的按钮的点击不冒泡、不做默认的动作：是 Headless UI 的 `Popover.Button` 自己的代码，没有我们的代码可以变异（原型上的一次探查：P1 中点负责人，`document`、`window` 上的监听都没有收到这次点击；升级 Headless UI 时重看，第 5 节 M4 一行）；
  - 面包屑最后一节的标题是 `<span>`：P10 的页面上没有作为最后一节的面包屑下拉框（项目页头上项目一节之后还有一节；作为最后一节的在 M4–M6 的页头上）；
  - 调用方的 `optionsClassName` 到达 popper 定位的面板，调用方给列表的 z-index 有效（Task 2 的修正轮，裁定 T2-a）：给列表 z-index 的调用方都在 M4 的行中，P10 的页面上没有，vitest 的替身不渲染选择；
  - 成员下拉框的根在行中可以收缩（`shrink`）、它的选择按钮没有自己的悬停底色（`hover:bg-transparent`）（修正轮，裁定 T2-c）：只是类名，行和悬停在 M4 的页面上；
  - `transitionsEnded` 不等无限的动画（修正轮，裁定 T2-d）：测试的辅助函数，P10 的弹窗进场时没有无限的动画。

  原来的例外（general 页的 `key`）由 P3 的同一路由内的移动看住（`T5.7`，第 3 节第 7 条）：应用中今天没有走到它的路，这一步代替之后加在设置页上的项目切换。

### A.3 W2：挂载清单和表情的数据（真实运行）

S2 的清单在 Task 3 的树上加入，每次 `make e2e` 都核对（2.12）。另在 `6c1a090a` 和最终原型上各跑一次一次性的清单（`$M3TMP/p10tools/w2/w2-inventory.spec.ts`：五个账户各打开 P10 的 11 个页面，记下每页的请求和失败的请求；跑完删除，不在 plan 中）。账户：工作区管理员（Web、Old 的管理员；Old 已归档）、Web 的管理员、Web 的成员、Web 的访客、不是 Web 成员的工作区成员。之前与之后（`w2/table.txt`）：

- 项目列表、已归档的项目页：五个账户都是 `APP` + `WORKSPACE` + `ARCHIVED`，之前之后相同。
- Web 的设置页（general、members、四个 features、automations）：是成员的四个账户是 `APP` + `WORKSPACE` + `PROJECT` + `PROJECT_MEMBER`（general 另有 `GENERAL`），不是成员的只有 `APP` + `WORKSPACE` + `PROJECT`（P8b 起），之前之后相同；没有失败的请求。
- Web 的工作项页：是成员的四个账户另请求 M4 的 `user-properties`（404，2.12），之前之后相同。
- 已归档的 Old 的工作项页：它的管理员之前是 `APP` + `WORKSPACE` + `PROJECT` + `PROJECT_MEMBER` + `user-properties` 的 404（页面照常显示），之后只有 `APP` + `WORKSPACE` + `PROJECT`（已归档的界面，Task 3）；看得到而不是成员的三个账户之前之后都只取 `PROJECT`，界面从"加入项目"变为已归档；访客看不到它（nerve 404，"找不到项目"），之前之后相同。
- 没有 `/api/v0` 之外的请求，没有 `cdn.jsdelivr.net`。故事的页面版本（P1–P5、P8）都经 `watchPage`：每个故事断言记下的失败的请求恰好是它要的那几个（例如 P5 的 409、P2 和 P8 的 M4 的 404），没有旧地址的请求、没有页面错误、没有 CSP 违规。
- 表情的数据（2.1）：之后的原型上 `HEAD /assets/emojibase/15.3.2/en/data.json` 答 200，`Cache-Control: public, max-age=31536000, immutable`，没有 `ETag`、`Last-Modified`；`GET` 700,120 字节；`messages.json` 同样。之前这个路径不存在，选择器请求 frimousse 默认的 `cdn.jsdelivr.net`，页面的 CSP 挡住它（变异 `T1.1`：选择器读 CDN，P1 的页面版本失败）。

### A.4 缺陷类的清扫

- **封闭的请求体**（P8b 的第一类）：P10 的页面发出的每个请求体都由表单编辑的字段构成，类型是生成的 `ProjectCreate`、`ProjectUpdate`、`ProjectMemberUpdate`、`ProjectMembersAdd`、`ProjectPreferencesUpdate`，角色和月数是数字，每个由一个测试钉住：创建（`use-create-project.test.ts`、P1 的 `sentHeld`）；general 页（`use-update-project-details.test.ts`、P3）；功能和自动归档（P3 的 `sentTo`；store 的测试在 P8b）；改角色（`use-project-membership-changes.test.ts`、P5 的 `{ role: 5 }`）；添加（`add-project-members-modal.test.tsx`、P5）；成员的默认值（`project-settings-member-defaults.test.tsx`、P3）；图标（`logo-props.test.ts`、P3）；标签栏和侧边栏（P8）。5 个变异（A.2 的 `body`），都被发现。
- **错误的读法**（第二类）：P10 改到的页面的每一条出错的路径都经 `errorMessageKey`（`useRefusalToast`、`projectRefusal`），没有吞进 `console.error` 的拒绝，被拒绝的修改结束加载。P10 删掉的：`member-list-item.tsx` 的两处 `err.error`、`member-columns.tsx` 的 `err.error` 和 `console.log`、`project/form.tsx` 的 Plane 字段码和 `console.error`、`add-project-members-modal.tsx` 的 `console.error`、`use-tab-preferences.ts` 的 `console.error`、`useFeatureToggle` 的 `setPromiseToast`、`use-project-drop.ts` 和 `auto-archive-automation.tsx` 的固定文字。web 应用中仍读 `err.error`、`error?.error` 的都不在 P10 的页面中（工作项 M4、迭代和模块 M6、Webhook M8，P9 spec 第 5 节）。10 个变异。
- **请求体取自所持的值**（第三类）：P10 的页面中这样的修改（功能的开关、访客可见全部、自动归档的开关、侧边栏的移动、标签栏的修改）都经 P8b 的 store 方法，在轮到它时从 nerve 最近的回答算出；P10 只改它们的跟进，页面自己不算请求体。成员的默认值一次只发一个字段。页面上同一类的显示问题：store 的几份副本（F-3，`T3.10`、`T3.11`；Task 3 的修正轮加 `T3.16` 共用的计数、`T3.17` 已归档的列表的一份）、标签栏取到之前的修改（`T10.4`）。
- **请求在路上时弹窗关得掉**（P9 的第一类）：P10 改到的八个弹窗（加入、创建、自定义时长、移出和离开的确认、离开、添加、删除、归档和恢复）在途时取消禁用、`handleClose` 为空；创建表单另在页头的关闭按钮上（`T4.11`）。17 个变异。被拒绝时这些弹窗都留着、按钮重新可用，只在成功的跟进中关上（第 3 节第 6 条）：成员页的确认框原来不论结果都关，修订之后由 `done` 关上；4 个变异（A.2 的 `stays`）。
- **忙碌在跟进之前结束**（第二类）：加入界面的按钮、general 页、归档和恢复、离开忙到跟进做完（`followInSession` 等跟进交回的 `Promise`）；删除弹窗先关（第 3 节第 14 条）。4 个变异（`T3.5`、`T5.5`、`T6.6`，和 Task 3 的修正轮加的加入对话框的按钮 `T3.13`）。
- **包装层之上的状态**（第三类）：P10 没有在 `AuthenticationWrapper` 之上的状态；general 页和创建表单的值在页面里，换账户时随页面卸载。
- **连按两下**（第四类）：P10 的每个修改按钮在途时不可用（`isSubmitting`、`loading`）；P2 的两个页面版本扣住加入时数请求，只发一次；删除、归档被扣住时弹窗关不掉、取消不可用（P4），没有另数请求。
- **会话外的跟进**（W3）：`followsweep.py` 列出 web 应用中提示或跳转的每个文件和它们等待的 store 调用，逐个读过：项目一侧的都在 2.13 的表中，每一处有它自己的 vitest（12 个变异）。
- **19（每个新的 hook 方法有失败测试）**：`useJoinProject`、`useCreateProject`、`useUpdateProjectDetails`（修改和检查各一个）、`useProjectMembershipChanges` 的三个修改、`useFeatureToggle`、自动归档、添加成员、成员的默认值、删除、归档和恢复、`useProjectDrop`、`useTabPreferences` 的三个修改、`useCopyLink` 都有被拒绝（或浏览器拒绝）的测试；`projectRefusal` 的表中有没有到达 nerve 的一行。
- **36（另一条路径到达同一结果）**：同一个修改的几个入口共用一个实现：加入（卡片的对话框和包装层的界面，`useJoinProject`）、离开（成员页、侧边栏、项目页头，`leave`）、复制（六处，`useCopyLink`）、项目表单的拒绝（创建和 general 页，`projectRefusal`）、图标（创建和 general 页，`logoPropsOf`）、角色的规则（成员页和添加，`project-roles.ts`）、带搜索的选择（`CustomSearchSelect` 和成员下拉框，第 3 节第 2 条）。
- **45（没有不能失败的断言）**：W18（A.2）：109 个测试都至少被一个变异或探查变异发现。
- **46（每个等待有期限）**：vitest 中等待页面跟进的是 `pageSettled()` 和 `vi.waitFor`；端到端的等待是 Playwright 的 `expect`、`expect.poll`、`toPass`（P8 的拖动，5 秒，预检的 L4）、`waitForResponse`、P9 的 `enabledWithin`、`closedByEscape` 和 Task 10 的 `shownWithin`（一秒），都有期限。
- **50（说明与代码一致）**：每个 Task 结束时，新写或改动的注释、JSDoc 与它描述的代码或测试逐句核对；最终原型上 P10 加或改的注释（`addedcomments.py`，`logs/comments.txt`）再全部读过一遍；README、前端改动清单和两份交接的行与最终的代码核对（Task 11）。

### A.5 oxlint（7.9，R3）

- 有手改的 TS 文件：web 的 85 个（web 应用 76 个，含 22 个测试；`@nerve/propel` 4 个、`@nerve/ui` 3 个、`@nerve/utils` 2 个），都是 0 条警告（`oxsnapfiles.py`）。P10 没有机械步骤，没有只经机械步骤到达的文件。`6c1a090a` 上这些文件共有 12 条：`dropdowns/member/base.tsx` 4 条、`member-options.tsx` 2 条（Task 2 删除、重写）、`confirm-project-member-remove.tsx` 1 条（Task 7，正的 `tabIndex`）、propel 的 `emoji/emoji.tsx` 3 条、`helper.tsx` 2 条（Task 1）。
- 上限：web 应用 356 → 350（Task 2）→ 349（Task 7）；`propel` 16 → 11（Task 1）；`utils` 7、`ui` 19、`constants` 1、`types` 0、`i18n` 0 不变（各包的上限要求警告数等于上限）。
- 抑制：没有新的抑制；删去两处（Task 5：`project/form.tsx` 的 `react-hooks/exhaustive-deps`；Task 7：`member-list-item.tsx` 的 `promise/always-return`），185 → 183（`suppressions.py`）。
- 根目录 `.oxlintrc.json` 的 `typescript/no-non-null-assertion` 的范围加 P10 的新模块和测试，各在加入它们的 Task（2.14）；每组一个变异（A.2 的 W12）。
- 关键词守卫：Task 7 加规则 `project-leave`（2.14），67 → 68 条规则、3 个例外，每个 Task 都没有命中；变异 `T7.13`（离开弹窗直接调 store 的 `leaveProject`）让它失败。

### A.6 规模

- **plan** 共 9,718 行：Task 1 之前（约束、文件结构）119 行；各 Task 一节：

| Task | plan 行数 | 其中代码块 | 文件 |
|---|---|---|---|
| 1 | 559 | 446 | 18 |
| 2 | 1,052 | 942 | 15 |
| 3 | 1,478 | 1,331 | 26 |
| 4 | 974 | 857 | 18 |
| 5 | 886 | 800 | 9 |
| 6 | 575 | 496 | 8 |
| 7 | 1,251 | 1,140 | 14 |
| 8 | 1,249 | 1,145 | 14 |
| 9 | 713 | 630 | 7 |
| 10 | 728 | 642 | 10 |
| 11 | 134 | 87 | 4 |

- **文件的行数**（约 400 行）：P10 改到的文件中最长的是 `p3-project-settings.spec.ts` 399 行、`e2e/stories/workspace/w3-workspace-settings.spec.ts` 397 行、`e2e/fixtures/assert/project.ts` 397 行、`e2e/stories/project/p1-create-project.spec.ts` 397 行、`p5-project-members.spec.ts` 396 行、`e2e/fixtures/api.ts` 390 行、`s2-web-app.spec.ts` 383 → 356 行（都是原有的故事和 fixture，P10 加了页面版本和行；P3、P5 的经 API 的部分移到共用的 fixture，P3 的同一路由内的移动是 `settings-pages.ts` 的 `moveWithinApp`，S2 的列表移进 `mounts.ts`，第 3 节第 13、16 条）；web 中最长的是 `project/form.tsx` 292 行（原来 411 行）、`project.store.ts` 376 行；新文件中最长的是 `use-update-project-details.test.ts` 143 行、`use-create-project.test.ts` 114 行。超过的只有文案文件（`common.json` 532 → 528 行）和锁文件（`pnpm-lock.yaml`，Task 1 加 6 行）。
- **变长的 Plane 文件**（brief：只为使用方改到的不变长）：22 个变长，21 个是本 Phase 的对象（行为改在它们里面）：`custom-search-select.tsx` 237 → 272、成员下拉框 `dropdowns/member/base.tsx` 187 → 195（对调用方的约定）、面包屑的 `navigation-search-dropdown.tsx` 111 → 124（一个 Tab 停留点）、`project-access-restriction.tsx` 57 → 81（已归档的界面）、`delete-project-modal.tsx` 159 → 175、`projects/create/attributes.tsx` 94 → 110（负责人的候选和位置）、`useProjectColumns.tsx` 135 → 149（调用者和每一行）、`dropdowns/buttons.tsx` 143 → 151（`tabIndex`）、`propel` 的 `emoji/emoji.tsx` 88 → 96（`EMOJIBASE_URL`）、`confirm-project-member-remove.tsx` 94 → 101、`card.tsx` 321 → 327、`select-month-modal.tsx` 106 → 112、`tab-navigation-visible-item.tsx` 83 → 88、`utils/src/project.ts` 73 → 78、`tab-navigation-overflow-menu.tsx` 99 → 102、`projects/create/utils.ts` 21 → 24（默认值没有负责人）、`project.store.ts` 374 → 376、`create-project-modal.tsx` 59 → 60、`join-project-modal.tsx` 71 → 72、`emoji-picker.tsx` 154 → 155（`ariaLabel`）、`emoji-icon-picker/helper.tsx` 156 → 157（`ariaLabel`、导出的类型）；只为使用方改到的是 `emoji-icon-picker/index.ts` 11 → 12（第 3 节第 5 条）。其余有手改的 Plane 文件变短或不变（例如 `project/form.tsx` 411 → 292、`member-columns.tsx` 186 → 125、`project-settings-member-defaults.tsx` 188 → 139、`member-options.tsx` 201 → 删除）（`sizes.py`，`logs/sizes.tsv`）。

### A.7 W12：类型断言

- `w12.cjs`（TypeScript 的语法树，`6c1a090a` 对最终原型）：P10 改到的 61 个源文件（测试除外）中，`as`（不含 `as const`）3 → 1、`any` 14 → 10、非空断言 0 → 0。去掉的：`add-project-members-modal.tsx`、`member-columns.tsx` 各 1 个 `as`；`member-options.tsx`、`project/create/header.tsx`、`project/form.tsx`、`leave-project-modal.tsx` 各 1 个 `any`。剩下的（`w12left.cjs`）都是 Plane 原有、P10 没有改的那几行：`@nerve/utils` 的 `project.ts` 的 `key as keyof TProjectFilters`；`@nerve/ui` 的 `dropdowns/helper.tsx` 7 个 `any`、`custom-search-select.tsx` 2 个（第 5 节收尾）。
- 新的 `as const`：没有（Task 3 的测试原来写的一个改为有类型的常量）。`castsall.py` 对整个 Phase 的 99 个 TS 文件（测试和 e2e 在内）核对：没有新的 `as`、`any`、`!` 或 `oxlint-disable`（它列出的都是字符串里的英文 "as"、`disabled` 和 `!` 的取反）。非空断言另有 oxlint 看住（2.14）。
- 没有手写的、重述契约的类型：表单的值是 `ProjectCreationForm`（`Pick<ProjectCreate, …>` 加可空的负责人）、`ProjectDetails`（`Pick<ProjectUpdate, …>`）、`TDeleteProjectForm`（表单自己的两段文字）；knip 没有未使用的导出。

### A.8 文案

两种语言相同，从快照逐个量出（`i18nadded.py`，`en`；`check:sync` 核对 `zh-CN` 的键相同）：加 4 条、删 4 条、改写 0 条。

| Task | 文件 | 加 | 删 |
|---|---|---|---|
| 1 | `accessibility.json` | `aria_labels.project_icon` | |
| 3 | `empty-state.json` | `project_empty_state.archived.title`、`description`、`cta_primary` | |
| 5 | `common.json` | | `project_name_already_taken`、`project_name_cannot_contain_special_characters`、`project_identifier_already_taken`、`common.identifier_already_exists` |

- **删除是完整的**（P8b 的 PF-L5）：`i18norphans.py`（P9 加宽之后的写法：全名，或经导出的常量的字段）在最终原型上 0 条：Task 5 删除的四条是 `project/form.tsx` 的旧错误分支唯一的读者。
- 新的文案说 nerve 的事：已归档的界面说项目管理员在已归档的项目中恢复它之后页面会重新显示。

### A.9 Phase 的大小

写 plan 之前的核对（brief）：设计的 15 个任务中，13（端到端）并进各页面的 Task（测试与代码同一个提交），14 是控制者的，15 只改文档，估出约 13 个 Task、约 11,000–13,000 行，最长的约 1,400 行：在约 16 个、约 1,500 行之内，不拆。原型中实测：11 个 Task（列表、卡片、加入、包装层、复制并成 Task 3；添加和成员的默认值并成 Task 8），修订之前 9,344 行、最长的 Task 3 是 1,473 行。修订（第 3 节"预检之后"）共 9,718 行，最长的 Task 3 是 1,478 行，其余都在 1,251 行之内。

### A.10 逐 Task 复现

`replay.py` 在 `6c1a090a` 的 `git archive` 副本上，换进本提交的四份文档（M3 设计、总体设计、本 spec 和 plan；`amend/mkbase.py` 写出 `base-amended.tar`），`pnpm install --frozen-lockfile` 之后（`$M3TMP/p10replay`），照 plan 逐个 Task 执行：写入块（`planapply.mjs`），核对 Go 的模块文件不变、`pnpm-lock.yaml` 与那个 Task 的快照相同，再执行这个 Task 的每个 `Run:`。副本不是 git 仓库，所以 `make lint-web` 的关键词守卫经 `kwreal.py` 运行仓库自己的 `tools/keywords.mjs`（`fakegit` 照 `git ls-files` 列出副本的文件；规则的样例、命中和例外都核对），`make e2e` 只有 S3 因为读不到提交而失败（F4），由脚本单独认出。

| Task | 写入的文件 | 检查 |
|---|---|---|
| 1 | 18 | `pnpm install --frozen-lockfile` 通过；`make lint-web`（67 条规则）、`make knip`、`make test-web` 通过；`make e2e` 93 个通过（S3 除外） |
| 2–6 | 15、26、18、9、8 | 同 Task 1，`make e2e` 94、97、98、100、101 个通过 |
| 7–10 | 14、14、7、10 | 同 Task 1，关键词守卫 68 条规则；`make e2e` 102、103、104、105 个通过 |
| 11 | 4 | 同 Task 10 |

这一次复现用的 plan 就是本 spec 同一个提交中的 plan（9,718 行，SHA-256 `c02e041f921e6683…`），11 个 Task 都通过（`$M3TMP/p10tools/replay-logs/summary.txt`；修订之前的一次在 `replay-logs-v1`）。最后的副本与原型逐文件相同，只差换进的文档：`treediff.mjs p10replay p10snap/T11`，3,203 个文件对 3,201 个，4 处差异，都是本提交的文档（M3 设计、总体设计改了，本 spec 和 plan 不在快照中）。副本中的 spec 是本提交的这一份，只有本节是复现之后写的。
