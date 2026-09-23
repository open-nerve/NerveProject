# M1/P2 删减内容：评审记录

| 项 | 内容 |
|---|---|
| Phase | M1/P2 `trim-content` |
| 日期 | 2026-09-23 |
| 结论 | **通过** |
| spec / plan | [spec](../specs/P2-trim-content.md) / [plan](../plans/P2-trim-content.md) |
| 分支 | `worktree-m1-p2-trim-content`，从 `main` 的 `57fccb6` 分出，提交从 `da0d09f` 到本评审记录所在的提交 |
| 评审 | 各 Task 评审（sonnet，第 2 节）；整分支评审（opus，`57fccb6..36172ca`）：Ready to merge after fixes；修复轮后的范围复核（sonnet）：Approved |

## 1. 范围与结果

P2 删掉 M1 设计 2.1 列出的 11 个 Plane 功能。保留功能的唯一入口逐条保住（spec 2.3，以及 spec 3 节第 2、3、9 项）：

| Task | 删掉的功能 | 保住或改写的保留功能 |
|---|---|---|
| 1 | 侧边栏的自定义导航（固定、隐藏、排序，`/sidebar-preferences/`） | 侧边栏改为常量里的固定列表。项目导航偏好原来的唯一入口在自定义导航弹窗里，现在改由 `ProjectNavigationDialog` 配置 |
| 2 | 工作区数据分析、活跃迭代推广页，连同 propel 的 7 个图表和 `table/` | 迭代和模块的进度代码由 analytics 改名为 progress；项目迭代列表中的"当前迭代"区块保留 |
| 3 | 个人主页的统计、动态和项目分布 | `/profile/:userId` 重定向到"分配给他的"；个人主页只剩用户卡片和工作项分页。新增 `use-profile-member.ts`，区分加载中、加载失败、不是成员、是成员四种状态 |
| 4 | 估算（点数、估算设置、各处的点数切换） | 迭代和模块的进度只按工作项数计算；`calculateCycleProgress` 和乐观更新有 13 个单元测试 |
| 5 | 甘特图、工作项甘特布局、模块时间线布局、`POST /issue-dates/` | 工作项关联保留，`REVERSE_RELATIONS` 从 `constants/gantt-chart.ts` 移到 `constants/issue/relation.ts` |
| 6 | "自动化"设置页中的自动关闭 | 自动归档保留，月份选择框只为它服务 |
| 7 | 文档页、协作编辑（Yjs、Hocuspocus、live 服务、PDF 导出、工作项嵌入）、更新日志 | 编辑器去掉协作相关的部分（`provider`、文档信息与标题回调、`getDocument` 等）；新手导览以"视图"结束 |
| 8 | 编辑器侧边菜单的 AI 处理器、AI 菜单、页面专用的排版变量 | `UniqueID`、`copyMarkdownToClipboard`、@成员、图片与附件节点保留（M1 设计 3.1） |
| 9 | 便签（页面、首页组件、便签编辑器）和 `react-masonry-component` | 工具栏由"按编辑器类型分组"合并为一组 |
| 10 | 首页快捷链接、首页个性化（组件开关）、自定义主题、已经没有入口的旧首页仪表盘 | 新增 `home-body.tsx`，首页正文固定为"没有项目"空状态和最近访问（M1 设计 3.14）；`theme-legacy.ts` 改名为 `theme.ts` |
| 11 | 工作区导出设置页，连同它借用的、从未创建过的集成服务 | — |
| 12 | 七个遗留依赖 | 编辑器包有了自己的测试：组成测试 7 个、交互测试 7 个，运行在 jsdom 中 |

数字（P1 结束 → P2 结束，`ef3be19`）：

| 项 | P1 结束 | P2 结束 |
|---|---|---|
| 关键词守卫 | 12 条规则、0 条例外 | **24 条规则、14 条例外**（11 条 `M1/P3`、1 条 `M6`、2 条 `M9`） |
| `make lint-web` / `make test-web` / `make build-web` | 52 / 12 / 11 | 52 / **15** / 11 |
| `pnpm exec turbo run check:types` | 23 | 23 |
| lint 上限合计 | 954 | **809** |
| — web / editor / utils / ui / propel | 777 / 75 / 34 / 31 / 29 | 658 / 67 / 27 / 28 / 21 |
| — hooks / constants / i18n / types | 4 / 2 / 1 / 1 | 不变 |
| knip：未使用文件 / 依赖 / 导出 / 类型 / 枚举成员 / 重复导出 | 123 / 1 / 133 / 81 / 4 / 1 | 85 / **0** / 108 / 61 / 4 / **0** |
| 构建体积 js | 505 个 / 10,270,419 字节 | 439 个 / 7,319,739 字节 |
| 构建体积 css / fonts / other | 3 / 322,306；34 / 6,849,620；142 / 9,719,426 | 3 / 305,244；25 / 3,755,608；122 / 8,043,211 |
| 最大 chunk / 语言 chunk | `toolbar` 1,821,937 / 40 | `use-editor-flagging` 1,381,850 / 34 |

- **改动规模**：整个分支 `57fccb6..ef3be19` 共 **940 个文件改动，+6489/−49096 行，629 个文件被删除**（`git diff --shortstat`）。增加的行主要是 spec、plan、四个测试文件，以及 jsdom 在锁文件中的条目。
- **依赖**：
  - 删除 24 个直接依赖：Task 7 删 11 个、Task 9 删 1 个、Task 10 删 2 个、Task 12 删 8 个、修复轮删 2 个。
  - catalog 删除 20 条；覆盖项 `ws@8` 和 `react-masonry-component>react` 的 peer 规则随之删除。
  - 新增唯一的依赖 `jsdom` 30.1.1。它是编辑器包的开发依赖，许可证为 MIT，经 catalog 引入，满足 `minimumReleaseAge`。
- **一处回归在 P2 内部发现并修复**：
  - **原因**：Task 7 把每个编辑器各自的 `UniqueID.configure({ provider })` 换成了共享的 `UniqueID`。它的 `onCreate` 对只读编辑器执行 `this.options.updateDocument = false`，写的是所有编辑器共享的选项。
  - **后果**：页面上出现第一个只读编辑器（评论、描述历史、通知预览）之后，之后创建的可编辑编辑器既没有 id 插件，也不补 id。块的 `data-id` 因此消失，只补 id 的迁移更新（`skip_activity`）也不再发出。
  - **发现与修复**：Task 12 在编写交互测试前的探测中发现了它。修复提交 `79ef2b5` 让 `onCreate` 只读取、不写入共享选项。
  - **防护**：去掉这处修复时，交互测试"可编辑 → 只读 → 可编辑"失败，浏览器探测 2 的第 16、17 项也失败（附录 6.3、6.5）。
- **保留行为的核对**（M1 设计 7.5、spec 2.17）：三处进仓库的测试（附录 6.1）和三段临时核对脚本（附录 6.2–6.4，共 78 项检查），在 `36172ca` 上全部通过；修复轮之后在 `ef3be19` 的构建上重跑，仍全部通过。
- **文档**：`docs/v0/frontend-changes.md` 第二节有 11 行改为"已完成 | M1/P2"；M1 设计 12 节 P2 一行在本评审提交中更新。

### spec 第 4 节验收标准

| # | 验收标准 | 结论 |
|---|---|---|
| 1 | 12 个 Task 各一个提交，`check:types` 通过 | **满足**。12 个 Task 提交，另有回归修复 `79ef2b5` 和修复轮的 3 个提交。全部通过的任务数是 **23**，不是 spec 写的 22（计划数错了，见第 5 节） |
| 2 | `make lint-web`：24 条规则、14 条例外，52 个任务，警告数等于上限 | **满足**。上限合计 **809**，低于 spec 预计的 815：实测一直比原型低，每次都按实测值写回（第 4 节裁定 2） |
| 3 | `make test-web` 15，`make build-web` 11 | **满足** |
| 4 | 守卫：没有未登记的命中；14 条例外精确，`until` 为 M1/P3、M6 或 M9 | **满足**。每条 P2 规则的每个顶层备选项都有命中样本（修复轮 `ef3be19` 补齐了 17 个） |
| 5 | knip 不再报告这些功能；未使用依赖为零 | **修复轮之后满足**。整分支评审发现，Task 1、Task 4 删掉了 `@plane/ui` 的 `Sortable` 的全部两个使用者，而 knip 因为包入口的再导出看不到它。修复轮 `2f2228c` 删除了它和 `ui` 的两个 atlaskit 依赖，knip 的重复导出随之归零 |
| 6 | 锁文件核对只有删除直接引起的差异 | **满足**。每次改依赖都跑了 `lock-diff.mjs`，不是"纯删除"的行逐条解释（详见第 4 节）：<br>- Task 7：`source-map@0.6.1: + optional: true`，它唯一的非可选路径随 `react-pdf-html` 删除；<br>- Task 12：jsdom 的依赖树（32 个包），以及其他 importer 的 vitest 快照键（jsdom 填上了 vitest 的可选 peer） |
| 7 | `check:sync`；没有本 Phase 弄成无引用的键 | **修复轮之后满足**。计划的 `keyuse.mjs` 以"键文本在代码中出现过"判断有引用，会把 `export`、`required`、`optional`、`refreshing`、`common.month` 这类常见词算作有引用。修复轮改用"完整字符串字面量 + 模板前缀"，对比基线和 HEAD，找出并删除了这 5 个键；剩下的 4 个是误报（第 4 节裁定 12） |
| 8 | M1 设计 7.5 中 P2 的四行核对过 | **满足**，见附录 6 |
| 9 | S1–S4 通过；持续集成 `server`、`web`、`e2e` 通过 | **满足**。分支推送后，持续集成 run 35840920898 在 `ef3be19` 上通过：`server` 59 秒，`web` 134 秒（Lint 一步 73 秒，Unit tests 一步 13 秒），`e2e` 111 秒（S1–S4 所在的 E2E 一步 16 秒） |
| 10 | `frontend-changes.md` 第二节同步；M1 设计 12 节更新 | **满足** |

## 2. 各 Task 的评审

- 实现者全部用 Opus 5.5（用户在会话中途切换了模型），评审者用 sonnet。
- "孤儿"指只被本 Task 删除的代码使用的符号，在 Task 的基点用 `git grep` 核对。基点之前就已经没人用的代码留给 P3（第 4 节裁定 1）。

| Task | 内容 | 提交 | 核实方式 |
|---|---|---|---|
| 1 | 侧边栏固定列表 | `a178948` | 实现者 DONE（web 上限 775）。评审 Approved；评审者确认了三点：<br>- `ProjectNavigationDialog` 可以从侧边栏的偏好按钮打开；<br>- 被删的接口和 store 没有残留引用；<br>- 没有保留条目被悄悄丢掉（pi_chat 在另一个用户菜单里）。<br>修复轮删掉 `common.unpin`。`SidebarItemBase.additionalRender` 没有调用方，但基点就是如此，交 P3 |
| 2 | 数据分析、活跃迭代推广页 | `e3760cb` | 实现者 DONE_WITH_CONCERNS。**裁定**：<br>- 本 Task 造成的孤儿在本 Task 删：propel `portal/`、`hexToHsl`、`ProIcon` 与 `MARKETING_PRICING_PAGE_LINK`、`CustomRadarAxisTick`；<br>- 中文说明随英文改为"进度"。<br>评审 Approved，两条 Minor 在修复轮一并处理：模块头部多余的 fragment 三元式；推广规则里多余的备选项 |
| 3 | 个人主页 | `a75237e` | 实现者 DONE_WITH_CONCERNS。**计划缺陷**：`profile.actions` 整删会删掉资料设置侧边栏的 5 个在用标签，实际只删 `activity`、`connections`。<br>评审 Needs fixes，Important：头部面包屑没有用卡片的 `is_active` 判断，被移除的成员页面会显示"{name} Work"。<br>**裁定**：从根源修，卡片和头部共用一个推导（新增 `use-profile-member.ts`，返回判别联合），不复制判断。<br>评审者按 SWR 2.4.2 的实际实现追了四种状态：服务在网络错误时抛 `undefined`，SWR 的 `error` 是假值，所以"加载失败"必须单独判断 |
| 4 | 估算、按工作项数计算进度 | `0d80deb` | 实现者 DONE：13 个进度测试，每个都在 14 种人为改坏中的至少一种下失败。四处偏离全部采纳：<br>- 保留 0–100 的钳制；<br>- "已删除"测试多两条断言；<br>- 模块头部 `completedIssues != 0 && completedIssues != 0` 是 Plane 的笔误，改为判断 `totalIssues`；<br>- `ICycle.progress/version`、`cycleChartOptions` 作为孤儿删除。<br>**计划缺陷**：power-k 的 `i18n-del` 少了 `power_k.` 前缀。评审 Approved |
| 5 | 甘特图与模块时间线 | `ea5e31f` | 实现者 NEEDS_CONTEXT：规则还命中 `pages/…/version-history.tsx` 里的 6 处 `timeline` 注释（文档页版本历史的装饰竖线）。<br>**裁定**：登记精确例外（`until: M1/P3`），不为躲守卫改注释，也不收窄正则。Task 7 删这个文件时例外会变陈旧、守卫失败，从而迫使 Task 7 删掉它（第 4 节裁定 5）。<br>六个甘特专属的孤儿随本 Task 删除。评审 Approved，无发现 |
| 6 | 自动关闭 | `790550f` | 实现者 DONE，改动与计划完全一致。`default_state` 在 Plane 里只是自动关闭的目标状态；新工作项的默认状态来自 `State.default`。评审 Approved，无发现 |
| 7 | 文档页、协作编辑、更新日志 | `dc10639` | 实现者 NEEDS_CONTEXT：锁文件有两行不是删除。**裁定**：<br>- 接受 `ws@8` 覆盖项的删除和 `source-map` 的 `optional` 标记；<br>- `getDocument` 整体删除（它唯一的调用方是文档页）；<br>- 控制者预检发现的 `description_binary`、`page_view` 一并删除。<br>**计划缺陷**：<br>- `rm -r app/assets/onboarding` 会删掉保留的导览用的 4 张图（实现者恢复了它们）；<br>- `live.service.ts` 在 P1 已删；<br>- "更新日志借用文档页类型"一句不成立。<br>评审 Approved，一条 Minor：收藏图标的兜底 `PagesOutline`，基点就有，交 P3。<br>**本 Task 引入了 `UniqueID` 共享选项的回归**，由 Task 12 发现、`79ef2b5` 修复 |
| 8 | 编辑器内核只留本地能力 | `3745a63` | 实现者 DONE_WITH_CONCERNS：<br>- `tippy.js` 按计划留到 Task 12 集中删；<br>- 表格菜单的"适应宽度"读取只有页面容器才设置的 `--editor-content-width`，在基点就对保留的编辑器无效，属于产品决定，交 P3。<br>评审 Approved，无发现 |
| 9 | 便签、工具栏收敛 | `651a4f6` | 实现者 DONE_WITH_CONCERNS。**计划缺陷**：`react-masonry-component>react` 在 `peerDependencyRules.allowedVersions` 里，不在 `overrides`；只删这一行，保留同时说明下一行的注释。`"enter-key"` 扩展开关只有便签编辑器关过，作为孤儿删除。评审 Approved |
| 10 | 首页固定内容、自定义主题、旧仪表盘 | `2e8174a` | 实现者 DONE_WITH_CONCERNS。**计划缺陷**：<br>- `rm -r core/components/core/theme` 会删掉保留的固定主题选择框 `theme-switch.tsx`；<br>- `home/index.ts` 要保留 `export * from "./root"`。<br>**裁定**（适用于全 Phase）：以被删功能命名的文案和图片随功能删除，即使基点就已无引用（第 4 节裁定 3）。<br>评审 Approved。修复轮删掉了：14 个自定义主题键；头部只剩一个子元素的 fragment；只剩一个键的 `WidgetLoader`/`EWidgetKeys` |
| 11 | 导出与集成服务 | `dc492a3` | 实现者 DONE_WITH_CONCERNS。去掉规则里多余的 `EXPORTERS_LIST`：有 `i` 标志时 `exporter` 已经覆盖它。<br>**裁定**：`jira.svg` 和 12 个集成、导入文案留给 P3，因为 M1 设计 7.4 把这组词汇整体交给 P3 的守卫规则。<br>评审 Approved，无发现 |
| 12 | 收尾：遗留依赖、编辑器测试、文档 | `79ef2b5`、`36172ca` | 实现者 NEEDS_CONTEXT：探测发现 `UniqueID` 回归。**裁定**：<br>- 采用方案 A（`onCreate` 不写共享选项），单独一个修复提交；<br>- 交互测试要包含"可编辑 → 只读 → 可编辑"。<br>其他结果：<br>- `vitest.setup.ts` 为 jsdom 声明没有 canvas、`Range` 没有布局，返回值与 jsdom 自己的相同；<br>- 第三方包的 6 条 sourcemap 警告记录在案，不加日志过滤；<br>- 15 次人为改坏，每次都让目标测试失败；<br>- 去掉 `mainFields` 一行时，第一个编辑器抛 `RangeError`（附录 6.5）。<br>评审 Approved |

## 3. 整分支评审（opus）：Ready to merge after fixes

评审范围 `57fccb6..36172ca`，结果为 0 Critical、4 Important、6 Minor。评审者读了全部代码差异，还写了一组只读脚本：

- 对比基线和 HEAD，找出 knip 看不见的包内孤儿；
- 用 TypeScript AST 找只剩一个子元素的 fragment；
- 逐个顶层备选项核对守卫样本；
- 对比基线和 HEAD 的文案引用报告。

| 编号 | 级别 | 问题 | 处理 |
|---|---|---|---|
| I1 | Important | `@plane/ui` 的 `Sortable` 失去了全部使用者（Task 1、Task 4 各删一个），knip 因包入口的再导出看不到它；`ui` 的两个 atlaskit 依赖只为它存在 | 已修（`2f2228c`）：删除 `ui/src/sortable/`、再导出和两个依赖。`lock-diff.mjs` 只有 `ui` importer 的两条删除；`ui` 上限 29 → 28 |
| I2 | Important | 编辑器的 `wideLayout`、`fontStyle` 两个显示选项只有文档页设置和读取过 | 已修（`2f2228c`）：删除两个选项、`TEditorFontStyle`、`EDITOR_FONT_STYLES`、三个 propel 字体图标和 `.serif`/`.monospace`/`.sans-serif` 样式块，保留根 `--font-style` |
| I3 | Important | `TExtensions` 的 `"ai"` 是 Task 7 的孤儿（控制者在 Task 9 期间已经发现） | 已修（`2f2228c`）。之后 `useEditorFlagging` 成了没有作用的企业版空壳，按 M1 设计交给 P3 |
| I4 | Important | 31 个以被删功能命名、基点就无引用的键（`project_page.*`、`workspace_pages.*`、`workspace_dashboard.*`、`workspace_cycles.empty_state.active.*`）和 12 张图片仍在 | 已修（`887acc1`），连同 `common.page(s)` 和一次按功能名的系统清扫：每种语言删 42 个键，删 12 张图片 |
| M1 | Minor | 5 处 fragment 只剩一个子元素 | 已修（`2f2228c`） |
| M2 | Minor | 三处只剩一个成员的结构：`COMPLEX_ITEMS`、侧边菜单的 `handlesConfig`、迭代进度图的外层 `<div>` | 已修（`2f2228c`） |
| M3 | Minor | `RESTRICTED_URLS` 仍保留 live 服务的 `"live"` | 已修（`2f2228c`） |
| M4 | Minor | 17 个守卫备选项没有命中样本 | 已修（`ef3be19`），样本取自基点的真实代码行 |
| M5 | Minor | 两条规则比 M1 设计 7.4 的词表窄，但没有写明原因 | 已修（`ef3be19`）：`why` 说明了留下的词会命中保留代码 |
| M6 | Minor | 5 个 `themes.theme_options.*.label` 键在 HEAD 无引用，但在基点也无引用（固定主题用的是常量里的 `i18n_label`） | 记录，交 M1 收尾 |

### 3.1 修复后的复核（sonnet，范围复核）

修复轮 `2f2228c`、`887acc1`、`ef3be19` 之后，范围复核的结论是 **Approved**：I1–I4、M1–M5 全部为 Fixed，修复本身没有引入新问题。复核者对每一项都自己取证：

- **I1**：`pnpm-lock.yaml` 的差异只有 `web/packages/ui` 这个 importer 的两条依赖（6 行）；web 自己的 3 个 atlaskit 依赖没有变。
- **I2、I3**：被删的编辑器选项、类型、常量、图标在 `web/` 中已经没有读取者；`useEditorFlagging` 按裁定保留，交 P3。
- **I4**：
  - 每个被删的键在中英文里都已删除（逐文件的删除行数中英文一一对应），被删的图片文件名在 `web/` 中没有引用；
  - 抽查了 10 个以上保留的键，包括 `profile-view.tsx` 按模板读取的 `profile.empty_state.*`，都原样保留；
  - `user-menu.tsx` 仍在读取的 `workspace_dashboards`（复数，是另一个键）没有被误删。
- **M2**：`TOOLBAR_ITEMS` 的键不变（`extensions.test.ts` 断言，14 个测试通过）；侧边菜单里对 `handlesConfig.dragDrop` 的读取逐行换成 `dragDropEnabled`，逻辑不变；迭代进度图的根节点换成原来那个 `div.py-4`，在 `divide-y` 面板里的位置和布局都不变。
- **M4**：17 个新样本逐条确认是 `57fccb6` 上真实存在的代码行，并且各自匹配它要覆盖的那个备选项。
- **M5**：`why` 里提到的保留符号逐个确认仍在使用。
- **复跑**：`node tools/keywords.mjs`（24 条规则、14 条例外、没有命中）、编辑器测试（14 个）、`check:types`（23 个）。

范围之外的一条观察：`editor-container.tsx` 的根节点外面仍有一个只包一个子元素的 fragment。它在基点就是这样，交 P3。

修复轮之后，控制者在 `ef3be19` 的构建上重跑了三段临时核对脚本：37、17、24 项全部通过，端口全部释放，没有残留的浏览器进程。附录 6.2–6.4 的输出就是这次重跑的结果。

## 4. 控制者的裁定

执行中的裁定都按"能自己定的就自己定"的原则做出：只要符合 SOLID、从根源解决、不打补丁，就不升级给用户。没有一项属于架构级、跨模块或意料之外的高风险，因此都没有升级。

1. **孤儿的口径**：本 Task 删除的代码是某个符号唯一的使用者时，这个符号在本 Task 删掉，以 `git grep` 在本 Task 的基点核对。基点之前就已经没人用的代码留给 P3。整分支评审和修复轮又补上了跨 Task 的孤儿（`Sortable`、`"ai"`、两个编辑器显示选项），P2 不把自己造成的孤儿交给 P3。
2. **lint 上限按实测**：上限只降不升。实测一直比原型低（web 比计划少 2 到 7），每个 Task 都按实测值写回；计划里写的数字只作参考。
3. **以被删功能命名的文案和图片**（Task 10 期间定下，覆盖 Task 5、7、11 的同类情况）：随功能一起删除，即使基点就已无引用，因为"被砍的功能要删干净"。不指向任何被删功能的通用死资源留给 M1 收尾，即 spec 2.16 说的 564 个键和 254 张图片。本 Phase 因此从收尾基线中拿走了：
   - 键：Task 10 修复轮的 14 个，修复轮 `887acc1` 的 31 个；
   - 图片：Task 5 的 10 张甘特图，Task 7 的 45 张（按文件名口径是 35 张），Task 11 的 3 张图标，修复轮的 12 张。

   `workspace_empty_state.dashboard` 是企业版仪表盘的文案，随 dashboard 词汇一起删除，计入收尾口径。
4. **守卫命中的处理**：属于本功能的命中直接删掉；属于后续 Phase 的，登记精确例外。不为躲守卫去改文字，也不为躲合法的命中去收窄正则。注释的主题还在、只是提到了被删功能的，改写成仍然成立的说法（Task 9 的两处颜色注释）；这不算躲守卫，因为不改的话注释在删除之后就是错的。
5. **跨 Task 的例外**：Task 5 的规则命中了 Task 7 才删除的文件，于是登记 `until: M1/P3` 的精确例外。Task 7 删文件后，这条例外变陈旧、守卫失败，逼着 Task 7 删掉它，这也同时证明了文件确实被删了。
6. **锁文件里的非删除行**：只接受由删除直接引起、能说清原因的行。
   - `ws@8` 覆盖项：`@hocuspocus/provider` 走后，图中已经没有 ws@8。
   - `source-map` 的 `optional` 标记：见验收标准第 6 项。
   - jsdom 的依赖树和 vitest 快照键：见验收标准第 6 项。

   jsdom 30 不依赖 ws，所以不需要重新加覆盖项。
7. **遗留依赖集中在 Task 12 删**：计划有意为之，锁文件只改一次。Task 8 后没人用的 `tippy.js` 因此留到 Task 12。它在锁文件里仍然存在，是 TipTap 自己的气泡菜单和浮动菜单的依赖。
8. **`UniqueID` 回归**：采用方案 A。`onCreate` 在只读或关闭时提前返回，永远不写共享选项。这样每个编辑器的行为与 Task 7 之前完全相同：插件总是安装，只有可编辑时才补初始 id。不采用方案 B（每个编辑器各自 `configure()` 一份），因为那样会把"写共享选项"这个根因藏起来。检查了编辑器包里其他写 `this.options` 的地方：只有这一处。`utility.ts` 写的是每个编辑器独立的 `this.storage`。
9. **测试环境的声明**：`vitest.setup.ts` 为 jsdom 声明没有 canvas（`getContext` 返回 `null`，与 jsdom 自己的返回值相同；探测它的是 `is-emoji-supported`），并声明 `Range` 没有布局（`getClientRects` / `getBoundingClientRect`）。原因是 TipTap 聚焦时会调用 `scrollIntoView`，在 jsdom 里会抛错。这些是声明环境，不是隐藏输出。第三方包 `prosemirror-codemark`、`is-emoji-supported` 发布的 sourcemap 缺源文件，产生 6 条警告，没有配置能关掉它们；决定记录在案，不加自定义日志过滤，因为过滤器也会把以后真正的警告藏起来。
10. **集成与导入的词汇**：`jira.svg`、12 个集成和导入文案、计费和订阅里的集成文字、`RESTRICTED_URLS` 的 `silo`，全部留给 P3。M1 设计 7.4 把这组词汇整体交给 P3 的守卫规则；现在删一半，这组词汇就会分散在两个 Phase，而且 P2 删的那一半背后没有守卫。
11. **退化结构从根源收掉**：只剩一个子元素的 fragment、只剩一个键的映射或枚举、只剩一个分支的三元式，只要是本 Phase 的删除造成的，就由本 Phase 收掉（Task 2、10 的修复轮和整分支评审的 M1、M2）。基点就是这样的留给 P3。
12. **文案引用的判断方法**：计划的 `keyuse.mjs` 以"键文本在任何代码中出现过"判断有引用，对常见词会误判为有引用。修复轮改用"完整字符串字面量 + 模板前缀"，对比基线和 HEAD，找出 5 个被漏掉的键。M1 收尾处理 564 个死键时要用这种更严格的方法（第 6 节）。

## 5. 计划缺陷

计划由原型写成，执行时发现以下缺陷，都已在对应 Task 中修正，不需要改动计划文件本身：

| Task | 缺陷 | 实际做法 |
|---|---|---|
| 全部 | `check:types` 写作 22 个任务 | 全部通过是 23 个任务（计划数错） |
| 全部 | 预期的 lint 上限、shortstat、构建体积、knip 数字 | 原型与本次执行的前序 Task 不完全相同，数字逐次解释，以实测为准 |
| 3 | `i18n-del … profile.actions` 会删掉资料设置侧边栏的 5 个在用标签 | 只删 `activity`、`connections` |
| 4 | power-k 的 `i18n-del` 少了 `power_k.` 前缀 | 用完整路径 |
| 5 | `core/components/gantt-chart/`（34） | 实际 39 个文件，总数 73 与计划一致 |
| 7 | `rm -r app/assets/onboarding` 会删掉保留的导览用的 4 张图 | 恢复这 4 张 |
| 7 | 删除 `packages/services/src/live.service.ts` | P1 已删 |
| 7 | "更新日志借用了文档页类型" | 本次的代码中不成立，规则说明和提交信息已改 |
| 7 | `project_page.*` 在 `empty-state.json` 里找 | 键实际在 `project.json`，由修复轮 `887acc1` 删除 |
| 9 | `react-masonry-component>react` 在 `overrides` 里 | 实际在 `peerDependencyRules.allowedVersions` |
| 10 | `rm -r core/components/core/theme` | 会删掉保留的 `theme-switch.tsx`，只删 5 个自定义主题文件 |
| 10 | `home/index.ts` 改为只导出 `home-body` | `page.tsx` 从这里导入 `WorkspaceHomeView`，保留 `export * from "./root"` |
| 12 | 编辑器的 `vitest.config.ts` 只有别名、用 node 环境 | 控制者评审补充第 4 条推翻：jsdom、`mainFields`、交互测试 |
| 12 | 验收时 `git status` 预期有 `?? .superpowers/` | 控制者评审补充第 2 条推翻：必须为空 |
| 12 | `check:lint` 中"等于上限"的包有 12 个 | 有上限的包是 13 个 |
| 全部 | `keyuse.mjs` 的引用判断 | 按子串判断会漏掉常见词键（第 4 节裁定 12） |

## 6. 附录（M1 设计 7.5）

临时核对脚本不进仓库。下面是它们的全文、假数据、运行命令、断言和输出，可以复制出来重跑：`$P2TMP` 是任意临时目录，脚本从仓库根目录运行，Playwright 取自 `e2e/`；先 `make build-web`。以后的 Phase 改到同一块区域时，要重跑这些场景（P4 重跑所有与路由有关的场景）。各探测报告正文是英文（由执行探测的子代理写成），原样收录。

### 6.1 三处进仓库的测试

```
$ pnpm --dir web/packages/utils exec vitest run src/progress.test.ts --reporter=verbose
 RUN  v4.1.11 <repo>/web/packages/utils
 ✓ src/progress.test.ts > calculateCycleProgress > is zero without a cycle 0ms
 ✓ src/progress.test.ts > calculateCycleProgress > is zero for an empty cycle 0ms
 ✓ src/progress.test.ts > calculateCycleProgress > leaves the cancelled work items out of the total 0ms
 ✓ src/progress.test.ts > calculateCycleProgress > counts the started work items as well when asked to 0ms
 ✓ src/progress.test.ts > calculateCycleProgress > reports 100 once everything that was not cancelled is done 0ms
 ✓ src/progress.test.ts > calculateCycleProgress > reports zero when every work item was cancelled 0ms
 ✓ src/progress.test.ts > calculateCycleProgress > prefers the snapshot of a completed cycle over the live counts 0ms
 ✓ src/progress.test.ts > the optimistic count update > adds a new work item to the total and to its own state group 1ms
 ✓ src/progress.test.ts > the optimistic count update > moves the count between state groups when a work item is completed 0ms
 ✓ src/progress.test.ts > the optimistic count update > moves it back when the work item is reopened 0ms
 ✓ src/progress.test.ts > the optimistic count update > takes a removed work item out of the total 0ms
 ✓ src/progress.test.ts > the optimistic count update > keeps the assignee and label counts of a module in step 0ms
 ✓ src/progress.test.ts > the optimistic count update > takes the completed work item off the burn-down chart on the day it was completed 0ms
 Test Files  1 passed (1)
      Tests  13 passed (13)
   Start at  17:06:59
   Duration  506ms (transform 44ms, setup 0ms, import 453ms, tests 3ms, environment 0ms)

$ pnpm --dir web/packages/constants exec vitest run src/navigation.test.ts --reporter=verbose
 RUN  v4.1.11 <repo>/web/packages/constants
 ✓ src/navigation.test.ts > the workspace sidebar is a fixed list > shows these items above the workspace group, in this order 1ms
 ✓ src/navigation.test.ts > the workspace sidebar is a fixed list > shows these items inside the workspace group, in this order 0ms
 ✓ src/navigation.test.ts > the workspace sidebar is a fixed list > gives every item a label, a link and at least one role 0ms
 ✓ src/navigation.test.ts > the profile page > keeps exactly the three work-item tabs 0ms
 ✓ src/navigation.test.ts > the profile page > points each tab at its own route 0ms
 Test Files  1 passed (1)
      Tests  5 passed (5)
   Start at  17:07:00
   Duration  77ms (transform 16ms, setup 0ms, import 22ms, tests 2ms, environment 0ms)

$ pnpm --dir web/packages/editor exec vitest run --reporter=verbose
 RUN  v4.1.11 <repo>/web/packages/editor
Sourcemap for "<repo>/node_modules/.pnpm/prosemirror-codemark@0.4.2_prosemirror-inputrules@1.5.0_prosemirror-model@1.25.3_prosem_b033beb31662c628da53ea9ebf5dd9a5/node_modules/prosemirror-codemark/dist/esm/index.js" points to missing source files
Sourcemap for "<repo>/node_modules/.pnpm/prosemirror-codemark@0.4.2_prosemirror-inputrules@1.5.0_prosemirror-model@1.25.3_prosem_b033beb31662c628da53ea9ebf5dd9a5/node_modules/prosemirror-codemark/dist/esm/plugin.js" points to missing source files
Sourcemap for "<repo>/node_modules/.pnpm/prosemirror-codemark@0.4.2_prosemirror-inputrules@1.5.0_prosemirror-model@1.25.3_prosem_b033beb31662c628da53ea9ebf5dd9a5/node_modules/prosemirror-codemark/dist/esm/utils.js" points to missing source files
Sourcemap for "<repo>/node_modules/.pnpm/prosemirror-codemark@0.4.2_prosemirror-inputrules@1.5.0_prosemirror-model@1.25.3_prosem_b033beb31662c628da53ea9ebf5dd9a5/node_modules/prosemirror-codemark/dist/esm/inputRules.js" points to missing source files
Sourcemap for "<repo>/node_modules/.pnpm/prosemirror-codemark@0.4.2_prosemirror-inputrules@1.5.0_prosemirror-model@1.25.3_prosem_b033beb31662c628da53ea9ebf5dd9a5/node_modules/prosemirror-codemark/dist/esm/actions.js" points to missing source files
Sourcemap for "<repo>/node_modules/.pnpm/is-emoji-supported@0.0.5/node_modules/is-emoji-supported/dist/esm/is-emoji-supported.js" points to missing source files
 ✓ src/extensions/extensions.test.ts > the extensions every editor installs > installs no collaboration extension 1ms
 ✓ src/extensions/extensions.test.ts > the extensions every editor installs > keeps undo and redo local, in the history of the starter kit 0ms
 ✓ src/extensions/extensions.test.ts > the extensions every editor installs > switches the history off when the caller asks it to 0ms
 ✓ src/extensions/extensions.test.ts > the extensions every editor installs > keeps the nodes the description, the comments and the history view need 0ms
 ✓ src/extensions/extensions.test.ts > the extensions every editor installs > leaves the image out when the caller disables it 0ms
 ✓ src/extensions/extensions.test.ts > the toolbar > has one set of items, since one editor type is left 0ms
 ✓ src/extensions/extensions.test.ts > the toolbar > offers no item that only the deleted page editor showed 0ms
 ✓ src/editor-interaction.test.ts > a rich-text editor built like the kept ones > takes typed text and gives it back as HTML and as Markdown 53ms
 ✓ src/editor-interaction.test.ts > a rich-text editor built like the kept ones > undoes and redoes typing with the keyboard 14ms
 ✓ src/editor-interaction.test.ts > a rich-text editor built like the kept ones > shows a description read-only, and edits it once it is editable again 20ms
stderr | src/editor-interaction.test.ts > a rich-text editor built like the kept ones > inserts a user mention that is saved, read back and copied as Markdown
linkifyjs: already initialized - will not register custom scheme "http" until manual call of linkify.init(). Register all schemes and plugins before invoking linkify the first time.
linkifyjs: already initialized - will not register custom scheme "https" until manual call of linkify.init(). Register all schemes and plugins before invoking linkify the first time.
stderr | src/editor-interaction.test.ts > a rich-text editor built like the kept ones > inserts an image from the toolbar, and keeps an uploaded image through a save
linkifyjs: already initialized - will not register custom scheme "http" until manual call of linkify.init(). Register all schemes and plugins before invoking linkify the first time.
linkifyjs: already initialized - will not register custom scheme "https" until manual call of linkify.init(). Register all schemes and plugins before invoking linkify the first time.
stderr | src/editor-interaction.test.ts > a rich-text editor built like the kept ones > gives every block node an id, keeps it through a save and gives new blocks new ones
linkifyjs: already initialized - will not register custom scheme "http" until manual call of linkify.init(). Register all schemes and plugins before invoking linkify the first time.
linkifyjs: already initialized - will not register custom scheme "https" until manual call of linkify.init(). Register all schemes and plugins before invoking linkify the first time.
 ✓ src/editor-interaction.test.ts > a rich-text editor built like the kept ones > inserts a user mention that is saved, read back and copied as Markdown 37ms
 ✓ src/editor-interaction.test.ts > a rich-text editor built like the kept ones > inserts an image from the toolbar, and keeps an uploaded image through a save 19ms
 ✓ src/editor-interaction.test.ts > a rich-text editor built like the kept ones > gives every block node an id, keeps it through a save and gives new blocks new ones 18ms
 ✓ src/editor-interaction.test.ts > a rich-text editor built like the kept ones > creates editors one after another in one process: editable, read-only, editable 22ms
 Test Files  2 passed (2)
      Tests  14 passed (14)
   Start at  17:07:01
   Duration  3.05s (transform 2.93s, setup 24ms, import 5.15s, tests 187ms, environment 436ms)
```

### 6.2 临时核对脚本 1：首页、个人主页、侧边栏（7.5 "P2 首页、个人主页、侧边栏"）

#### P2 retained-behaviour probe 1: home page, profile page, sidebar

##### 1. Purpose

Covers the M1 design 7.5 row "P2 首页、个人主页、侧边栏" against the built web app at the branch head
(36172ca; build `web/apps/web/build/client`, 2026-09-23 15:52, after 79ef2b5): recent visits with data and
empty, no pages entry anywhere, the profile address redirecting to "assigned", the profile card for a member, a
guest, an absent or inactive user, a failed and a still-loading members request, and the fixed sidebar with its
project navigation dialog. Plane's API is stubbed; the repository is not modified.

##### 2. Script (`$P2TMP/probe-1/home-profile-sidebar.mjs`, full text)

```js
// One-off (M1/P2 retained-behaviour probe 1, M1 design 7.5 row "P2 首页、个人主页、侧边栏"): serves the built web
// app (web/apps/web/build/client, SPA fallback to its index.html) on a free port, signs a stubbed admin into
// workspace probe-ws with project p1, and checks what a person sees on the home page, the profile page and the
// sidebar: recent visits with data and empty, no pages entry, the profile redirect to "assigned", the four
// profile card states, the fixed sidebar and the project navigation dialog. The stubs answer the Plane
// endpoints these pages call; every other /api request gets 404 and is listed at the end. Each scenario runs in
// a fresh browser context. Run from the repository root (Playwright comes from e2e/).
// usage: node home-profile-sidebar.mjs
import { createRequire } from "node:module";
import fs from "node:fs";
import http from "node:http";
import net from "node:net";
import path from "node:path";

const { chromium } = createRequire(path.resolve("e2e/package.json"))("@playwright/test");

// ---------------------------------------------------------------- fake data (minimal: no estimates, pages, epics)
const WS = "probe-ws";
const ME = { id: "u1", email: "probe@example.com", display_name: "probe", first_name: "Probe", last_name: "User", avatar_url: "" };
const ADA = { id: "u2", email: "ada@example.com", display_name: "ada", first_name: "Ada", last_name: "Lovelace", avatar_url: "" };
// page_view is the flag Plane's API still sends for a project: it is on here so that a leftover "Pages" entry
// would show; no page itself exists in the data.
const PROJECT = {
  id: "p1", name: "Probe Project", identifier: "PRB", sort_order: 65535, logo_props: {}, member_role: 20,
  archived_at: null, workspace: "w1", cycle_view: true, module_view: true, issue_views_view: true, inbox_view: true,
  page_view: true,
};
const STATE = { id: "s1", name: "Todo", group: "unstarted", color: "#3a3a3a", project_id: "p1", sequence: 1, default: true };
const member = (user, role, extra = {}) => ({
  id: `wm-${user.id}`, member: user, role, is_active: true, created_at: "2026-02-03T12:00:00Z", ...extra,
});
const RECENTS = [
  {
    id: "rv1", entity_name: "issue", entity_identifier: "i1", visited_at: "2026-09-23T08:00:00Z",
    entity_data: {
      id: "i1", name: "Probe work item", state: "s1", priority: "high", assignees: [], type: null, sequence_id: 7,
      project_id: "p1", project_identifier: "PRB", is_epic: false,
    },
  },
  {
    id: "rv2", entity_name: "project", entity_identifier: "p1", visited_at: "2026-09-23T07:00:00Z",
    entity_data: { id: "p1", name: "Probe Project", logo_props: {}, project_members: [], identifier: "PRB" },
  },
];

const NO_WORK_ITEMS = {
  grouped_by: null, sub_grouped_by: null, next_cursor: "100:1:0", prev_cursor: "100:-1:1", next_page_results: false,
  prev_page_results: false, total_count: 0, count: 0, total_pages: 1, extra_stats: null, results: [], total_results: 0,
};

class Reply {
  constructor(status, json, delayMs = 0) {
    Object.assign(this, { status, json, delayMs });
  }
}
// The start state: an onboarded admin (role 20) of probe-ws, which has project p1 and two members.
const startStubs = () => ({
  "GET /api/instances/": { instance: { is_setup_done: true }, config: { is_email_password_enabled: true } },
  "GET /api/users/me/": ME,
  "GET /api/users/me/profile/": { id: "pr1", user: "u1", language: "en", is_onboarded: true, is_tour_completed: true, theme: {} },
  "GET /api/users/me/settings/": { id: "u1", email: ME.email, workspace: { last_workspace_slug: WS } },
  "GET /api/users/me/workspaces/": [{ id: "w1", slug: WS, name: "Probe WS", total_members: 2, role: 20 }],
  [`GET /api/workspaces/${WS}/workspace-members/me/`]: { id: "wm-u1", member: "u1", workspace: "w1", role: 20, is_active: true },
  [`GET /api/users/me/workspaces/${WS}/project-roles/`]: { p1: 20 },
  [`GET /api/workspaces/${WS}/projects/`]: [PROJECT],
  [`GET /api/workspaces/${WS}/members/`]: [member(ME, 20), member(ADA, 15)],
  [`GET /api/workspaces/${WS}/states/`]: [STATE],
  [`GET /api/workspaces/${WS}/user-favorites/`]: [],
  [`GET /api/workspaces/${WS}/user-properties/`]: { navigation_control_preference: "ACCORDION", navigation_project_limit: 0 },
  // the work-item list of a profile tab: empty
  [`GET /api/workspaces/${WS}/user-issues/u1/`]: NO_WORK_ITEMS,
  [`GET /api/workspaces/${WS}/user-issues/u2/`]: NO_WORK_ITEMS,
});

// ---------------------------------------------------------------- static server over the build
const root = path.resolve("web/apps/web/build/client");
const shell = path.join(root, "index.html");
const types = {
  ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".json": "application/json",
  ".svg": "image/svg+xml", ".png": "image/png", ".gif": "image/gif", ".webp": "image/webp", ".jpg": "image/jpeg",
  ".ico": "image/x-icon", ".woff": "font/woff", ".woff2": "font/woff2", ".ttf": "font/ttf",
};
const server = http.createServer((req, res) => {
  let file = path.join(root, decodeURIComponent(new URL(req.url, "http://x").pathname));
  if (!file.startsWith(root) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) file = shell;
  res.writeHead(200, { "content-type": types[path.extname(file)] ?? "application/octet-stream" });
  fs.createReadStream(file).pipe(res);
});

// ---------------------------------------------------------------- checks
const results = [];
const check = (name, ok, detail = "") => {
  results.push({ name, ok });
  console.log(ok ? `PASS ${name}` : `FAIL ${name}: ${detail}`);
};
const visible = async (locator, timeout = 10000) => {
  try {
    await locator.first().waitFor({ state: "visible", timeout });
    return true;
  } catch {
    return false;
  }
};
const count = (locator) => locator.count();
const bodyText = (page) => page.locator("body").innerText();
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const REMOVED_ENDPOINTS = /home-preferences|quick-links|stickies|widgets|\/pages/;
const WIDGET_TEXT = /manage widgets|quick ?links?|stick(y|ies)/i;

const log = {}; // scenario -> { unstubbed, consoleErrors }
let browser;
let base;
async function scenario(name, stubs, fn) {
  console.log(`\n== ${name}`);
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 }, locale: "en-US" });
  const page = await context.newPage();
  const ctx = { requests: [], unstubbed: [], consoleErrors: [] };
  log[name] = ctx;
  page.on("console", (m) => m.type() === "error" && ctx.consoleErrors.push(m.text().split("\n")[0]));
  page.on("pageerror", (e) => ctx.consoleErrors.push(`pageerror: ${e.message}`));
  await context.route("**/api/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const key = `${request.method()} ${url.pathname}`;
    ctx.requests.push({ key, search: url.search, body: request.postData(), at: Date.now() });
    let stub = stubs[key];
    if (typeof stub === "function") stub = stub(request);
    try {
      if (stub === undefined) {
        ctx.unstubbed.push(key);
        return await route.fulfill({ status: 404, json: {} });
      }
      const reply = stub instanceof Reply ? stub : new Reply(200, stub);
      if (reply.delayMs) await sleep(reply.delayMs);
      return await route.fulfill({ status: reply.status, json: reply.json });
    } catch {
      // the context closed while a delayed reply was pending
    }
  });
  try {
    await fn(page, ctx);
  } catch (error) {
    check(`${name}: ran to the end`, false, error.message.split("\n")[0]);
  } finally {
    await context.close();
  }
}
const goto = (page, p) => page.goto(`${base}${p}`, { waitUntil: "domcontentloaded" });
// The content area: the <main> that does not hold the sidebar.
const content = (page) => page.locator("main").filter({ hasNot: page.locator("#main-sidebar") }).first();
// Waits for the address to match, then prints where the page landed.
const landsOn = async (page, pattern) => {
  const ok = await page.waitForURL(pattern, { timeout: 20000 }).then(() => true, () => false);
  const url = new URL(page.url());
  console.log(`  (landed on ${url.pathname}${url.search})`);
  return ok;
};
// Waits for a request without leaving an unhandled rejection when the step before it fails.
const requestSeen = (page, predicate) => page.waitForRequest(predicate, { timeout: 10000 }).then((r) => r, () => null);
// A project's navigation as "label -> href", in display order: every link of the nearest block that holds both
// its Cycles and its Intake link (the sidebar accordion or the tab bar). The sidebar's links end in "/", the
// tab bar's do not; the trailing "/" is dropped before comparing.
const PROJECT_NAVIGATION = ["Work items", "Cycles", "Modules", "Views", "Intake"].map(
  (label) => `${label} -> /${WS}/projects/p1/${label === "Work items" ? "issues" : label.toLowerCase()}`
);
const projectNavigation = async (scope) => {
  const p1 = `/${WS}/projects/p1`;
  const intake = scope.locator(`a[href="${p1}/intake"], a[href="${p1}/intake/"]`).first();
  await visible(intake, 20000);
  const block = intake.locator(`xpath=ancestor::div[.//a[@href="${p1}/cycles" or @href="${p1}/cycles/"]][1]`);
  return block.locator("a[href]").evaluateAll((as) =>
    as.map((a) => `${a.innerText.trim()} -> ${a.getAttribute("href").replace(/\/$/, "")}`));
};

// ---------------------------------------------------------------- scenarios
async function run() {
  // 1, 3, 6: the home page with recent visits, the sidebar, a project's navigation and the preferences dialog.
  await scenario("home with recent visits", {
    ...startStubs(),
    [`GET /api/workspaces/${WS}/recent-visits/`]: RECENTS,
    [`PATCH /api/workspaces/${WS}/user-properties/`]: (request) => ({
      navigation_control_preference: "ACCORDION", navigation_project_limit: 0, ...request.postDataJSON(),
    }),
  }, async (page, ctx) => {
    await goto(page, `/${WS}`);
    const main = content(page);
    check(
      "1.1 home greets the signed-in user",
      await visible(main.getByRole("heading", { level: 2, name: /^Good (morning|afternoon|evening), Probe User$/ }), 20000),
      await bodyText(page).catch(() => "")
    );
    const workItem = main.locator(`a[href="/${WS}/browse/PRB-7/"]`);
    const workItemOk = (await visible(workItem)) && /PRB-7[\s\S]*Probe work item/.test(await workItem.innerText());
    check("1.2 recents list the visited work item (PRB-7, its name, its link)", workItemOk,
      await main.innerText());
    const project = main.locator(`a[href="/${WS}/projects/p1/issues"]`);
    const projectOk = (await visible(project)) && /PRB[\s\S]*Probe Project/.test(await project.innerText());
    check("1.3 recents list the visited project (identifier, name, link to its work items)", projectOk,
      await main.innerText());
    const recentRequests = ctx.requests.filter((r) => r.key === `GET /api/workspaces/${WS}/recent-visits/`);
    check("1.4 the recents request asks for all entities (no entity_name)",
      recentRequests.length > 0 && recentRequests.every((r) => !r.search.includes("entity_name")),
      JSON.stringify(recentRequests));
    check("1.5 with a joined project and two members the quickstart guide stays hidden",
      (await count(main.getByText("Your quickstart guide"))) === 0, await main.innerText());
    const text = await bodyText(page);
    const removedCalls = ctx.requests.filter((r) => REMOVED_ENDPOINTS.test(r.key)).map((r) => r.key);
    check("2.3a home (with data) has no manage-widgets button, quick links or stickies",
      !WIDGET_TEXT.test(text) && removedCalls.length === 0,
      `text match: ${text.match(WIDGET_TEXT)}; calls: ${removedCalls}`);

    // 6. The sidebar is the fixed list of the constants, in their order.
    const sidebar = page.locator("#main-sidebar");
    const anchors = await sidebar.locator("a[href]").evaluateAll((as) =>
      as.map((a) => ({ href: a.getAttribute("href"), text: a.innerText.trim() })));
    const navItems = anchors.filter((a) => !a.href.startsWith(`/${WS}/projects/p1`));
    const expected = [
      { href: `/${WS}/`, text: "Home" },
      { href: `/${WS}/profile/u1/`, text: "Your work" },
      { href: `/${WS}/drafts/`, text: "Drafts" },
      { href: `/${WS}/projects/`, text: "Projects" },
      { href: `/${WS}/workspace-views/all-issues/`, text: "Views" },
      { href: `/${WS}/projects/archives/`, text: "Archives" },
    ];
    check("6.1 the sidebar shows exactly Home, Your work, Drafts | Projects, Views, Archives, in this order",
      JSON.stringify(navItems) === JSON.stringify(expected), JSON.stringify(anchors));
    const sidebarText = await sidebar.innerText();
    const forbiddenButtons = await count(sidebar.getByRole("button", { name: /^(pin|unpin|more|customi[sz]e.*)$/i }));
    check("6.2a the sidebar offers no customize, pin, unpin or more control",
      !/customi[sz]e|\bunpin\b|\bpin\b|^more$/im.test(sidebarText) && forbiddenButtons === 0,
      `buttons: ${forbiddenButtons}; text: ${sidebarText}`);
    const itemControls = [];
    for (const item of expected) {
      const link = sidebar.locator(`a[href="${item.href}"]`).first();
      await link.hover();
      await sleep(150);
      const inside = await link.evaluate((a) => ({
        buttons: a.querySelectorAll("button, [role=button]").length,
        draggable: a.matches('[draggable="true"]') || a.querySelectorAll('[draggable="true"], .cursor-grab').length > 0,
      }));
      if (inside.buttons || inside.draggable) itemControls.push({ ...item, ...inside });
    }
    check("6.2b hovering a sidebar item shows no control and no item can be dragged",
      itemControls.length === 0, JSON.stringify(itemControls));

    // 3. No pages entry: not on the home page, not in the sidebar, not in the project's navigation.
    const pagesEntries = async () => {
      const hrefs = await page.locator("a[href]").evaluateAll((as) => as.map((a) => a.getAttribute("href")));
      return { hrefs: hrefs.filter((h) => h.includes("/pages")), labels: await count(page.getByText("Pages", { exact: true })) };
    };
    const onHome = await pagesEntries();
    check("3.1 the home page and the sidebar have no link to /pages and no \"Pages\" item",
      onHome.hrefs.length === 0 && onHome.labels === 0, JSON.stringify(onHome));
    await sidebar.getByRole("button", { name: "Open project menu" }).first().click();
    const projectNavItems = await projectNavigation(sidebar);
    check("3.2 the project's sidebar navigation (accordion) is Work items, Cycles, Modules, Views, Intake, with no Pages",
      JSON.stringify(projectNavItems) === JSON.stringify(PROJECT_NAVIGATION), JSON.stringify(projectNavItems));
    const withProjectNav = await pagesEntries();
    check("3.3 with the project's navigation open there is still no /pages link and no \"Pages\" item",
      withProjectNav.hrefs.length === 0 && withProjectNav.labels === 0, JSON.stringify(withProjectNav));

    // 6. The project navigation preferences stay reachable from the sidebar.
    const title = sidebar.locator("span.text-16", { hasText: /^Projects$/ });
    await title.locator("xpath=..").locator("button").first().click();
    const dialog = page.getByRole("dialog");
    const dialogOk = (await visible(dialog.getByText("Accordion sidebar navigation"))) &&
      (await visible(dialog.getByText("Tabbed Navigation"))) &&
      (await visible(dialog.getByText("Show limited projects on sidebar")));
    check("6.3 the sidebar preferences button opens the project navigation dialog", dialogOk,
      await bodyText(page));
    const patch = requestSeen(page, (r) => r.method() === "PATCH" && r.url().includes(`/api/workspaces/${WS}/user-properties/`));
    await dialog.locator('input[value="TABBED"]').check();
    const patchBody = (await patch)?.postDataJSON() ?? "no PATCH request";
    check("6.4 choosing Tabbed Navigation saves {navigation_control_preference: TABBED}",
      JSON.stringify(patchBody) === JSON.stringify({ navigation_control_preference: "TABBED" }), JSON.stringify(patchBody));
    await page.keyboard.press("Escape");
    const closed = await dialog.waitFor({ state: "hidden", timeout: 5000 }).then(() => true, () => false);
    check("6.5 the dialog closes on Escape", closed, "still open");

    // 4. The sidebar's "Your work" lands on the assigned tab of one's own profile.
    await sidebar.locator(`a[href="/${WS}/profile/u1/"]`).click();
    const landed = await landsOn(page, new RegExp(`/${WS}/profile/u1/assigned/?$`));
    check("4.1 the sidebar's Your work lands on /probe-ws/profile/u1/assigned", landed, page.url());
    check("4.2 one's own profile is labelled Your work in the breadcrumb and Assigned is a tab",
      (await visible(main.getByText("Your work", { exact: true }))) &&
        (await visible(main.getByRole("link", { name: "Assigned" }))),
      await main.innerText());
  });

  // 3. A project opened in tabbed mode shows its navigation as tabs: still no pages.
  await scenario("project in tabbed navigation", {
    ...startStubs(),
    [`GET /api/workspaces/${WS}/user-properties/`]: { navigation_control_preference: "TABBED", navigation_project_limit: 0 },
    [`GET /api/workspaces/${WS}/projects/p1/`]: PROJECT,
    [`GET /api/workspaces/${WS}/projects/p1/project-members/me/`]: { id: "pm-u1", member: "u1", role: 20 },
    // the member's project preferences: which tab is the default and which tabs are hidden in the "more" menu
    [`GET /api/workspaces/${WS}/projects/p1/user-properties/`]: {
      sort_order: 65535, rich_filters: {}, display_filters: {}, display_properties: {},
      preferences: { navigation: { default_tab: "work_items", hide_in_more_menu: [] } },
    },
    // the project's work-item list, which those preferences start: empty
    [`GET /api/workspaces/${WS}/projects/p1/issues/`]: NO_WORK_ITEMS,
  }, async (page) => {
    await goto(page, `/${WS}/projects/p1/issues`);
    const main = content(page);
    const tabs = await projectNavigation(main);
    check("3.4 the project's tabs are Work items, Cycles, Modules, Views, Intake, with no Pages",
      JSON.stringify(tabs) === JSON.stringify(PROJECT_NAVIGATION), JSON.stringify(tabs));
    const hrefs = await page.locator("a[href]").evaluateAll((as) => as.map((a) => a.getAttribute("href")));
    const labels = await count(page.getByText("Pages", { exact: true }));
    check("3.5 the project page has no link to /pages and no \"Pages\" item",
      !hrefs.some((h) => h.includes("/pages")) && labels === 0, JSON.stringify({ hrefs, labels }));
  });

  // 2. A new workspace: no projects, no recent visits.
  await scenario("home, empty", {
    ...startStubs(),
    [`GET /api/users/me/workspaces/${WS}/project-roles/`]: {},
    [`GET /api/workspaces/${WS}/projects/`]: [],
    [`GET /api/workspaces/${WS}/recent-visits/`]: [],
  }, async (page, ctx) => {
    await goto(page, `/${WS}`);
    const main = content(page);
    check("2.1 with no recent visits the recents empty state shows",
      await visible(main.getByText("You don't have any recents yet."), 20000), await bodyText(page).catch(() => ""));
    check("2.2 with no projects the no-projects empty state (quickstart guide) shows, with Create a project",
      (await visible(main.getByText("Your quickstart guide"))) &&
        (await visible(main.getByText("Create a project"))) &&
        (await visible(main.getByRole("button", { name: "Get started" }))),
      await main.innerText());
    const text = await bodyText(page);
    const removedCalls = ctx.requests.filter((r) => REMOVED_ENDPOINTS.test(r.key)).map((r) => r.key);
    check("2.3b home (empty) has no manage-widgets button, quick links or stickies",
      !WIDGET_TEXT.test(text) && removedCalls.length === 0,
      `text match: ${text.match(WIDGET_TEXT)}; calls: ${removedCalls}`);
    const filtered = requestSeen(page, (r) => r.url().includes(`/api/workspaces/${WS}/recent-visits/?entity_name=project`));
    await main.locator('button[aria-haspopup="menu"]', { hasText: "All" }).click();
    await page.getByRole("menuitem", { name: "Projects" }).click();
    const sent = (await filtered) !== null;
    check("2.4 filtering recents by Projects asks for entity_name=project and shows the projects empty state",
      sent && (await visible(main.getByText("Your recent projects will appear here once you visit one."))),
      `request sent: ${sent}; ${await main.innerText()}`);
  });

  // 4. The profile address redirects to the assigned tab and keeps the query.
  await scenario("profile redirect", startStubs(), async (page) => {
    await goto(page, `/${WS}/profile/u2?probe=1`);
    check("4.3 /probe-ws/profile/u2?probe=1 lands on /probe-ws/profile/u2/assigned?probe=1",
      await landsOn(page, new RegExp(`/${WS}/profile/u2/assigned/?\\?probe=1$`)), page.url());
    await goto(page, `/${WS}/profile/u2`);
    const landedPlain = await landsOn(page, new RegExp(`/${WS}/profile/u2/assigned/?$`));
    const titled = await page.waitForFunction(() => document.title === "Profile - Assigned", null, { timeout: 10000 })
      .then(() => true, () => false);
    check("4.4 /probe-ws/profile/u2 lands on /probe-ws/profile/u2/assigned, titled \"Profile - Assigned\"",
      landedPlain && titled, `${page.url()} title=${await page.title()}`);
  });

  // 5. The profile card and breadcrumb for each state of the workspace members request.
  const NOT_A_MEMBER = "This user is not a member of this workspace.";
  const LOAD_FAILED = "Could not load this workspace's members.";
  const profileCase = (label, membersReply, expectation) =>
    scenario(`profile card: ${label}`, { ...startStubs(), [`GET /api/workspaces/${WS}/members/`]: membersReply },
      async (page) => {
        await goto(page, `/${WS}/profile/u2`);
        const main = content(page);
        await visible(main.getByRole("link", { name: "Assigned" }), 20000);
        await expectation(page, main);
      });
  const noUndefined = async (main) => !(await main.innerText()).includes("undefined");
  const showsMember = (id) => async (page, main) => {
    const ok = (await visible(main.getByRole("heading", { level: 4, name: "Ada Lovelace" }))) &&
      (await visible(main.getByText("(ada)"))) &&
      (await visible(main.getByText("Joined on"))) &&
      (await visible(main.getByText("Feb 03, 2026"))) &&
      (await visible(main.getByText("Ada Lovelace Work", { exact: true })));
    check(`${id} card shows Ada Lovelace (ada), joined Feb 03, 2026; breadcrumb "Ada Lovelace Work"`, ok,
      await main.innerText());
    check(`${id} neither the not-a-member nor the load-failed message shows, no "undefined"`,
      (await count(main.getByText(NOT_A_MEMBER))) === 0 && (await count(main.getByText(LOAD_FAILED))) === 0 &&
        (await noUndefined(main)), await main.innerText());
  };
  const showsMessage = (id, message, other) => async (page, main) => {
    check(`${id} card shows "${message}"`, await visible(main.getByText(message)), await main.innerText());
    check(`${id} no name and no "${other}"; breadcrumb is "Work", no "undefined"`,
      (await count(main.getByText("Ada Lovelace"))) === 0 && (await count(main.getByText(other))) === 0 &&
        (await visible(main.getByText("Work", { exact: true }))) && (await noUndefined(main)),
      await main.innerText());
  };
  await profileCase("active member", [member(ME, 20), member(ADA, 15)], showsMember("5a"));
  await profileCase("guest", [member(ME, 20), member(ADA, 5)], showsMember("5b"));
  await profileCase("absent from the list", [member(ME, 20)], showsMessage("5c", NOT_A_MEMBER, LOAD_FAILED));
  await profileCase("inactive", [member(ME, 20), member(ADA, 15, { is_active: false })],
    showsMessage("5d", NOT_A_MEMBER, LOAD_FAILED));
  await profileCase("members request fails (500)", new Reply(500, { error: "probe failure" }),
    showsMessage("5e", LOAD_FAILED, NOT_A_MEMBER));
  await scenario("profile card: members still loading",
    { ...startStubs(), [`GET /api/workspaces/${WS}/members/`]: new Reply(200, [member(ME, 20), member(ADA, 15)], 5000) },
    async (page, ctx) => {
      await goto(page, `/${WS}/profile/u2`);
      const main = content(page);
      await visible(main.getByRole("link", { name: "Assigned" }), 20000);
      const loaders = await count(main.getByRole("status"));
      const whileLoading = loaders > 0 && (await count(main.getByText("Ada Lovelace"))) === 0 &&
        (await count(main.getByText(NOT_A_MEMBER))) === 0 && (await count(main.getByText(LOAD_FAILED))) === 0 &&
        (await noUndefined(main));
      const asked = ctx.requests.find((r) => r.key === `GET /api/workspaces/${WS}/members/`)?.at;
      const inWindow = asked !== undefined && Date.now() - asked < 5000;
      check("5f while the members load (before the delayed reply): a loading skeleton, no name, no not-a-member, " +
        "no load-failed, no \"undefined\"", whileLoading && inWindow,
        `loaders ${loaders}, checked ${asked === undefined ? "?" : Date.now() - asked} ms after the request; ${await main.innerText()}`);
      const arrived = await visible(main.getByRole("heading", { level: 4, name: "Ada Lovelace" }), 15000);
      const loadersAfter = await count(main.getByRole("status"));
      check("5g once they arrive the skeleton gives way to the card of Ada Lovelace", arrived && loadersAfter < loaders,
        `loaders ${loaders} -> ${loadersAfter}; ${await main.innerText()}`);
    });
}

// ---------------------------------------------------------------- main
await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
const { port } = server.address();
base = `http://127.0.0.1:${port}`;
console.log(`serving ${path.relative(process.cwd(), root)} (SPA fallback: index.html) on port ${port}`);
try {
  browser = await chromium.launch();
  await run();
} finally {
  await browser?.close();
  await new Promise((resolve) => server.close(resolve));
}

console.log("\n== unstubbed requests (answered 404)");
for (const [name, ctx] of Object.entries(log)) {
  const counts = {};
  for (const key of ctx.unstubbed) counts[key] = (counts[key] ?? 0) + 1;
  const lines = Object.entries(counts).map(([key, n]) => `  ${key}${n > 1 ? ` x${n}` : ""}`);
  console.log(`${name}:${lines.length ? `\n${lines.join("\n")}` : " none"}`);
}
console.log("\n== console errors");
for (const [name, ctx] of Object.entries(log)) {
  const unstubbed = ctx.consoleErrors.filter((e) => /404|Not Found/.test(e));
  const deliberate = ctx.consoleErrors.filter((e) => !/404|Not Found/.test(e) && /500|Internal Server Error/.test(e));
  const other = ctx.consoleErrors.filter((e) => !unstubbed.includes(e) && !deliberate.includes(e));
  console.log(`${name}: ${unstubbed.length} from unstubbed requests, ${deliberate.length} from the deliberate 500, ${other.length} other`);
  for (const e of other) console.log(`  other: ${e}`);
}

const probe = net.createServer();
const free = await new Promise((resolve) => {
  probe.once("error", () => resolve(false));
  probe.listen(port, "127.0.0.1", () => probe.close(() => resolve(true)));
});
console.log(`\nport ${port} is ${free ? "free" : "STILL IN USE"}; browser connected: ${browser?.isConnected() ?? false}`);
const failed = results.filter((r) => !r.ok);
console.log(`\n${results.length - failed.length} passed, ${failed.length} failed`);
process.exit(failed.length || !free ? 1 : 0);
```

##### 3. Fake data

The fake data lives in the script (section "fake data" and `startStubs()`); no separate file. Every stub is keyed
by method and path; any other `/api` request is answered 404 and recorded.

- Start state (all scenarios): instance set up; user `u1` (Probe User, `probe@example.com`) with profile
  `language: "en"`, `is_onboarded: true` and `is_tour_completed: true` (without it the tour overlay covers the
  home page); workspace `probe-ws` (`w1`, 2 members) in which `u1` is admin (role 20); project `p1`
  ("Probe Project", identifier `PRB`, joined, cycles / modules / views / intake on); members `u1` (admin) and `u2`
  (Ada Lovelace, `ada`, member role 15, joined 2026-02-03); one workspace state; no favorites; project navigation
  preference `ACCORDION`; an empty work-item list for the profile tabs of `u1` and `u2`.
- `p1` also carries `page_view: true`, the flag Plane's API sends for a project. It is there so that a leftover
  "Pages" entry would show (Plane rendered it exactly when this flag was on); no page, estimate or epic exists in
  the data.
- Per scenario:
  - home with recent visits: `recent-visits/` returns one work item (`PRB-7` "Probe work item", state Todo,
    priority high) and one project (`p1`); `PATCH user-properties/` echoes the body merged into the preferences.
  - project in tabbed navigation: preference `TABBED`; project detail of `p1`; `u1`'s project membership (role 20);
    `u1`'s project preferences (default tab `work_items`, no hidden tab); an empty project work-item list.
  - home, empty: no projects, no project roles, no recent visits.
  - profile card: the workspace members request returns, in turn, `u2` as member (15), `u2` as guest (5), a list
    without `u2`, `u2` with `is_active: false`, a 500, and the member list delayed by 5 s.

##### 4. Run command

From the repository root (`<repo>`), with
`P2TMP=$P2TMP`:

```sh
node $P2TMP/probe-1/home-profile-sidebar.mjs
```

The script serves the build on a free port (SPA fallback to `index.html`, the only HTML shell the SPA build emits),
runs every scenario in a fresh Chromium context (1440x900, en-US), closes the browser and the server in `finally`,
then proves the port is free by listening on it again. Exit status is non-zero if any check fails or the port is
still taken. The final run took 10 s.

##### 5. Assertions

7.5 row items: (i) recent visits with data and empty; (ii) no pages entry; (iii) the profile address redirects to
"assigned"; (iv) member, guest, absent member and failed members request each shown correctly; (v) the sidebar is a
fixed list.

| Check | What it proves | 7.5 item |
|---|---|---|
| 1.1 | The home page renders for a signed-in user: "Good morning/afternoon/evening, Probe User" | (i) |
| 1.2 | Recent visits render a stubbed work item: `PRB-7` and its name, linking to `/probe-ws/browse/PRB-7/` | (i) |
| 1.3 | Recent visits render a stubbed project: `PRB` and its name, linking to `/probe-ws/projects/p1/issues` | (i) |
| 1.4 | The home asks `GET recent-visits/` for all entities (no `entity_name`) | (i) |
| 1.5 | With a joined project and two members the no-projects empty state (quickstart guide) stays hidden | (i) |
| 2.1 | With no recent visits the recents empty state shows ("You don't have any recents yet.") | (i) |
| 2.2 | With no projects `NoProjectsEmptyState` shows ("Your quickstart guide", "Create a project", "Get started") | (i) |
| 2.3a, 2.3b | Neither home (with data, empty) shows "Manage widgets", quick links or stickies, and neither calls a home-preferences, quick-links, stickies, widgets or pages endpoint | (i), M1 3.14 |
| 2.4 | The recents filter still works: choosing Projects sends `entity_name=project` and shows the projects empty state | (i) |
| 3.1 | Home page and sidebar (including the hidden peek copy): no `href` containing `/pages`, no element whose text is "Pages" | (ii) |
| 3.2 | The project's accordion navigation in the sidebar is exactly Work items, Cycles, Modules, Views, Intake (with `page_view: true`) | (ii) |
| 3.3 | With that navigation open there is still no `/pages` link and no "Pages" item | (ii) |
| 3.4 | In tabbed mode the project's tab bar is exactly Work items, Cycles, Modules, Views, Intake | (ii) |
| 3.5 | The project page has no `/pages` link and no "Pages" item | (ii) |
| 4.1 | The sidebar's "Your work" lands on `/probe-ws/profile/u1/assigned/` | (iii) |
| 4.2 | One's own profile is labelled "Your work" in the breadcrumb and has the Assigned tab | (iii), (iv) |
| 4.3 | `/probe-ws/profile/u2?probe=1` lands on `/probe-ws/profile/u2/assigned/?probe=1` (the query is kept) | (iii) |
| 4.4 | `/probe-ws/profile/u2` lands on `/probe-ws/profile/u2/assigned/`, page title "Profile - Assigned" | (iii) |
| 5a (2 checks) | `u2` as active member: card "Ada Lovelace", "(ada)", "Joined on Feb 03, 2026"; breadcrumb "Ada Lovelace Work"; no not-a-member or load-failed message; no "undefined" | (iv) member |
| 5b (2 checks) | `u2` as guest (role 5): the same card and breadcrumb | (iv) guest |
| 5c (2 checks) | `u2` absent from the list: "This user is not a member of this workspace."; no name, no load-failed message; breadcrumb "Work"; no "undefined" | (iv) absent |
| 5d (2 checks) | `u2` with `is_active: false`: the same not-a-member state | (iv) absent |
| 5e (2 checks) | Members request 500: "Could not load this workspace's members.", distinct from not-a-member; breadcrumb "Work"; no "undefined" | (iv) request fails |
| 5f | Members still loading (reply delayed 5 s, checked inside the window): a loading skeleton (`role=status`), no name, no message, no "undefined" | (iv) loading |
| 5g | When the members arrive the skeleton gives way to Ada's card | (iv) loading |
| 6.1 | `#main-sidebar` lists exactly Home, Your work, Drafts, then Projects, Views, Archives, with these hrefs and in this order | (v) |
| 6.2a | The sidebar has no customize, pin, unpin or "More" control (text or button) | (v) |
| 6.2b | Hovering each of the six items reveals no button inside it; no item is draggable | (v) |
| 6.3 | The preferences button beside the sidebar title opens `ProjectNavigationDialog` (accordion, tabbed, limited projects) | (v) |
| 6.4 | Choosing "Tabbed Navigation" sends `PATCH user-properties/` with body `{"navigation_control_preference":"TABBED"}` | (v) |
| 6.5 | The dialog closes on Escape | (v) |

##### 6. Output of the final run

```
serving web/apps/web/build/client (SPA fallback: index.html) on port 63314

== home with recent visits
PASS 1.1 home greets the signed-in user
PASS 1.2 recents list the visited work item (PRB-7, its name, its link)
PASS 1.3 recents list the visited project (identifier, name, link to its work items)
PASS 1.4 the recents request asks for all entities (no entity_name)
PASS 1.5 with a joined project and two members the quickstart guide stays hidden
PASS 2.3a home (with data) has no manage-widgets button, quick links or stickies
PASS 6.1 the sidebar shows exactly Home, Your work, Drafts | Projects, Views, Archives, in this order
PASS 6.2a the sidebar offers no customize, pin, unpin or more control
PASS 6.2b hovering a sidebar item shows no control and no item can be dragged
PASS 3.1 the home page and the sidebar have no link to /pages and no "Pages" item
PASS 3.2 the project's sidebar navigation (accordion) is Work items, Cycles, Modules, Views, Intake, with no Pages
PASS 3.3 with the project's navigation open there is still no /pages link and no "Pages" item
PASS 6.3 the sidebar preferences button opens the project navigation dialog
PASS 6.4 choosing Tabbed Navigation saves {navigation_control_preference: TABBED}
PASS 6.5 the dialog closes on Escape
  (landed on /probe-ws/profile/u1/assigned/)
PASS 4.1 the sidebar's Your work lands on /probe-ws/profile/u1/assigned
PASS 4.2 one's own profile is labelled Your work in the breadcrumb and Assigned is a tab

== project in tabbed navigation
PASS 3.4 the project's tabs are Work items, Cycles, Modules, Views, Intake, with no Pages
PASS 3.5 the project page has no link to /pages and no "Pages" item

== home, empty
PASS 2.1 with no recent visits the recents empty state shows
PASS 2.2 with no projects the no-projects empty state (quickstart guide) shows, with Create a project
PASS 2.3b home (empty) has no manage-widgets button, quick links or stickies
PASS 2.4 filtering recents by Projects asks for entity_name=project and shows the projects empty state

== profile redirect
  (landed on /probe-ws/profile/u2/assigned/?probe=1)
PASS 4.3 /probe-ws/profile/u2?probe=1 lands on /probe-ws/profile/u2/assigned?probe=1
  (landed on /probe-ws/profile/u2/assigned/)
PASS 4.4 /probe-ws/profile/u2 lands on /probe-ws/profile/u2/assigned, titled "Profile - Assigned"

== profile card: active member
PASS 5a card shows Ada Lovelace (ada), joined Feb 03, 2026; breadcrumb "Ada Lovelace Work"
PASS 5a neither the not-a-member nor the load-failed message shows, no "undefined"

== profile card: guest
PASS 5b card shows Ada Lovelace (ada), joined Feb 03, 2026; breadcrumb "Ada Lovelace Work"
PASS 5b neither the not-a-member nor the load-failed message shows, no "undefined"

== profile card: absent from the list
PASS 5c card shows "This user is not a member of this workspace."
PASS 5c no name and no "Could not load this workspace's members."; breadcrumb is "Work", no "undefined"

== profile card: inactive
PASS 5d card shows "This user is not a member of this workspace."
PASS 5d no name and no "Could not load this workspace's members."; breadcrumb is "Work", no "undefined"

== profile card: members request fails (500)
PASS 5e card shows "Could not load this workspace's members."
PASS 5e no name and no "This user is not a member of this workspace."; breadcrumb is "Work", no "undefined"

== profile card: members still loading
PASS 5f while the members load (before the delayed reply): a loading skeleton, no name, no not-a-member, no load-failed, no "undefined"
PASS 5g once they arrive the skeleton gives way to the card of Ada Lovelace

== unstubbed requests (answered 404)
home with recent visits:
  GET /api/workspaces/probe-ws/users/notifications/unread/
project in tabbed navigation:
  GET /api/workspaces/probe-ws/users/notifications/unread/
  GET /api/workspaces/probe-ws/projects/p1/issue-labels/ x2
  GET /api/workspaces/probe-ws/projects/p1/members/ x2
  GET /api/workspaces/probe-ws/projects/p1/states/ x2
  GET /api/workspaces/probe-ws/projects/p1/intake-state/ x2
  GET /api/workspaces/probe-ws/projects/p1/cycles/ x2
  GET /api/workspaces/probe-ws/modules/ x2
  GET /api/workspaces/probe-ws/projects/p1/modules/ x2
  GET /api/workspaces/probe-ws/projects/p1/views/ x2
home, empty:
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile redirect:
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile card: active member:
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile card: guest:
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile card: absent from the list:
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile card: inactive:
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile card: members request fails (500):
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile card: members still loading:
  GET /api/workspaces/probe-ws/users/notifications/unread/

== console errors
home with recent visits: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
project in tabbed navigation: 18 from unstubbed requests, 0 from the deliberate 500, 0 other
home, empty: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
profile redirect: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
profile card: active member: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
profile card: guest: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
profile card: absent from the list: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
profile card: inactive: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
profile card: members request fails (500): 2 from unstubbed requests, 1 from the deliberate 500, 0 other
profile card: members still loading: 2 from unstubbed requests, 0 from the deliberate 500, 0 other

port 63314 is free; browser connected: false

37 passed, 0 failed
exit status: 0 (run time 10 s)
```

Console errors: every one comes from an unstubbed request (all contain 404) or, in the 500 scenario, from the
deliberate 500 of the members request; there are no other console errors and no uncaught page errors. Two page
errors seen while writing the probe ("pageerror: Object", an unhandled rejection carrying the 404 body) came from
the then-unstubbed work-item lists of the profile tab and of the project page; both disappeared once those lists
were stubbed empty, so they are not findings.

After the run (same shell, right after the script exited):

```
$ lsof -nP -iTCP:63314
(exit 1 - nothing uses port 63314)
$ ps -axo command | grep -c '^node .*probe-1/home-profile-sidebar'
0
$ pgrep -f 'chrome-headless-shell.*remote-debugging-pipe'
75187
75199
```

The two headless browsers still running belong to another probe: 75187's parent is
`node $P2TMP/probe-2/editor-probe.mjs`, and 75199 is 75187's child. No process started by this probe is left.

##### 7. Not covered, and why

- **Failure against the old code is not demonstrated.** The probe ran only against the P2 head build; building the
  pre-P2 app would need `make build-web`, which the brief rules out. For the "no pages" checks the project carries
  `page_view: true`, so a surviving Pages entry (Plane shows it exactly when that flag is on) would make 3.2 to 3.5
  fail.
- **English only.** Labels are checked in `en`; `zh-CN` is not loaded.
- **Only an admin viewer.** The signed-in user is a workspace admin; the sidebar of a guest viewer (Your work,
  Drafts and Archives hidden by their `access` lists) and a guest's view of a profile are not probed.
- **Desktop width only** (1440x900): the small-screen profile card toggle and the project tab overflow menu are not
  exercised.
- **Recent items are checked by their links, not clicked.** Clicking the work item opens a peek overview that needs
  the work-item endpoints; clicking the project navigates into the project page.
- **Unstubbed on purpose:** `users/notifications/unread/` in every scenario, and in the tabbed-project scenario the
  project's labels, members, states, intake state, cycles, modules and views. No check reads them: the tab bar is
  built from the partial project, the project roles, the project detail, the project membership and the project
  preferences, all of which are stubbed.
- The loading state (5f) is judged by a `role=status` skeleton and the absence of every other state, checked within
  the 5 s delay of the members reply (the check fails if it runs after the reply).

Observations outside this row's behaviour (no finding):
- Links rendered through the `next/link` shim end in "/" (sidebar "Your work" is `/probe-ws/profile/u1/`, the
  sidebar project navigation links end in "/"); the react-router tab bar's links do not; the profile redirect
  ends on `/assigned/`. The checks accept both forms.
- `common.pages` ("Pages" / "页面") is still in `en` and `zh-CN` `common.json`, with no literal use of `common.pages`
  or `t("pages")` under `web/apps` or `web/packages`. It is never rendered (3.1 to 3.5 pass); it may be a leftover
  key worth a look.
- Another member's empty Assigned tab reads "No work items are assigned to you": Plane's original copy, not a P2
  change.

#### 修复轮之后在 `ef3be19` 的构建上重跑

```
serving web/apps/web/build/client (SPA fallback: index.html) on port 51818

== home with recent visits
PASS 1.1 home greets the signed-in user
PASS 1.2 recents list the visited work item (PRB-7, its name, its link)
PASS 1.3 recents list the visited project (identifier, name, link to its work items)
PASS 1.4 the recents request asks for all entities (no entity_name)
PASS 1.5 with a joined project and two members the quickstart guide stays hidden
PASS 2.3a home (with data) has no manage-widgets button, quick links or stickies
PASS 6.1 the sidebar shows exactly Home, Your work, Drafts | Projects, Views, Archives, in this order
PASS 6.2a the sidebar offers no customize, pin, unpin or more control
PASS 6.2b hovering a sidebar item shows no control and no item can be dragged
PASS 3.1 the home page and the sidebar have no link to /pages and no "Pages" item
PASS 3.2 the project's sidebar navigation (accordion) is Work items, Cycles, Modules, Views, Intake, with no Pages
PASS 3.3 with the project's navigation open there is still no /pages link and no "Pages" item
PASS 6.3 the sidebar preferences button opens the project navigation dialog
PASS 6.4 choosing Tabbed Navigation saves {navigation_control_preference: TABBED}
PASS 6.5 the dialog closes on Escape
  (landed on /probe-ws/profile/u1/assigned/)
PASS 4.1 the sidebar's Your work lands on /probe-ws/profile/u1/assigned
PASS 4.2 one's own profile is labelled Your work in the breadcrumb and Assigned is a tab

== project in tabbed navigation
PASS 3.4 the project's tabs are Work items, Cycles, Modules, Views, Intake, with no Pages
PASS 3.5 the project page has no link to /pages and no "Pages" item

== home, empty
PASS 2.1 with no recent visits the recents empty state shows
PASS 2.2 with no projects the no-projects empty state (quickstart guide) shows, with Create a project
PASS 2.3b home (empty) has no manage-widgets button, quick links or stickies
PASS 2.4 filtering recents by Projects asks for entity_name=project and shows the projects empty state

== profile redirect
  (landed on /probe-ws/profile/u2/assigned/?probe=1)
PASS 4.3 /probe-ws/profile/u2?probe=1 lands on /probe-ws/profile/u2/assigned?probe=1
  (landed on /probe-ws/profile/u2/assigned/)
PASS 4.4 /probe-ws/profile/u2 lands on /probe-ws/profile/u2/assigned, titled "Profile - Assigned"

== profile card: active member
PASS 5a card shows Ada Lovelace (ada), joined Feb 03, 2026; breadcrumb "Ada Lovelace Work"
PASS 5a neither the not-a-member nor the load-failed message shows, no "undefined"

== profile card: guest
PASS 5b card shows Ada Lovelace (ada), joined Feb 03, 2026; breadcrumb "Ada Lovelace Work"
PASS 5b neither the not-a-member nor the load-failed message shows, no "undefined"

== profile card: absent from the list
PASS 5c card shows "This user is not a member of this workspace."
PASS 5c no name and no "Could not load this workspace's members."; breadcrumb is "Work", no "undefined"

== profile card: inactive
PASS 5d card shows "This user is not a member of this workspace."
PASS 5d no name and no "Could not load this workspace's members."; breadcrumb is "Work", no "undefined"

== profile card: members request fails (500)
PASS 5e card shows "Could not load this workspace's members."
PASS 5e no name and no "This user is not a member of this workspace."; breadcrumb is "Work", no "undefined"

== profile card: members still loading
PASS 5f while the members load (before the delayed reply): a loading skeleton, no name, no not-a-member, no load-failed, no "undefined"
PASS 5g once they arrive the skeleton gives way to the card of Ada Lovelace

== unstubbed requests (answered 404)
home with recent visits:
  GET /api/workspaces/probe-ws/users/notifications/unread/
project in tabbed navigation:
  GET /api/workspaces/probe-ws/users/notifications/unread/
  GET /api/workspaces/probe-ws/projects/p1/issue-labels/ x2
  GET /api/workspaces/probe-ws/projects/p1/members/ x2
  GET /api/workspaces/probe-ws/projects/p1/states/ x2
  GET /api/workspaces/probe-ws/projects/p1/intake-state/ x2
  GET /api/workspaces/probe-ws/projects/p1/cycles/ x2
  GET /api/workspaces/probe-ws/modules/ x2
  GET /api/workspaces/probe-ws/projects/p1/modules/ x2
  GET /api/workspaces/probe-ws/projects/p1/views/ x2
home, empty:
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile redirect:
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile card: active member:
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile card: guest:
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile card: absent from the list:
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile card: inactive:
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile card: members request fails (500):
  GET /api/workspaces/probe-ws/users/notifications/unread/
profile card: members still loading:
  GET /api/workspaces/probe-ws/users/notifications/unread/

== console errors
home with recent visits: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
project in tabbed navigation: 18 from unstubbed requests, 0 from the deliberate 500, 0 other
home, empty: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
profile redirect: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
profile card: active member: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
profile card: guest: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
profile card: absent from the list: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
profile card: inactive: 2 from unstubbed requests, 0 from the deliberate 500, 0 other
profile card: members request fails (500): 2 from unstubbed requests, 1 from the deliberate 500, 0 other
profile card: members still loading: 2 from unstubbed requests, 0 from the deliberate 500, 0 other

port 51818 is free; browser connected: false

37 passed, 0 failed
```

### 6.3 临时核对脚本 2：应用中的编辑器（7.5 "P2 编辑器"的应用层部分）

#### Probe 2 — the editor inside the app (M1 design 7.5 row "P2 编辑器", app-level part)

(Written by the controller from the probe agent's reply: the harness refused the agent's own REPORT.md write.)

Files: `editor-probe.mjs` (script, 391 lines, fake data inline), `run-final.txt` (final run), `run-mutated.txt` (run against a build copy with only the pre-79ef2b5 UniqueID behaviour put back), `build-pre-79ef2b5/` (that build copy, 20 MB, for reruns).

##### 1. Purpose
In the built app (36172ca, 15:52) with Plane's API stubbed, on a work item's full page `/probe-ws/browse/P1-<n>/` and in its peek: description save with a UniqueID `data-id` on every block; undo/redo; ids after a read-only editor has appeared (the 79ef2b5 regression); a comment with an @mention; description history (view a version, copy its Markdown, restore it); an image node and locating a block by its id.

##### 2. Script
P1 §5.5 model (header, `createRequire(e2e/package.json)`); Node `http` server over the build with fallback to `index.html` (the only HTML the build emits), port 0, closed in `finally`; `page.route("**/api/**")` answers from a map keyed by method + path (values, functions or a PNG body); every unstubbed request gets a 404 naming the path and is recorded; an init script prints unhandled rejections; `PROBE_BUILD=<dir>` serves another build directory (mutation run only).

##### 3. Fake data (inline, minimal)
- Start endpoints (P1 lang-probe set, extended): instance, users/me, profile, settings, workspaces `[probe-ws]`, workspace member-me (role 20), project roles `{p1:20}`, workspace members `u1 probe`, `u2 mia`, projects `[p1 "P1"]`, states `[s1]`, workspace user properties.
- Work-item page: `work-items/P1-1/`, `work-items/P1-2/`, `issues/i3/` (fetched as parent); project p1 detail, project-members/me, project members, states, labels `[]`; per issue history (comments for `activity_type=issue-comment`, else `[]`), empty sub-issues, empty relations, empty description-versions (except i1); `PATCH issues/<id>/` echoes the merged issue; `POST comments/` echoes the comment.
- P1-1 description: paragraph `b-alpha-1`, `image-component` (src `asset-1`, id `img-1`), 30 filler paragraphs, paragraph `b-alpha-2`. P1-2: one paragraph, one comment "Earlier comment on beta", parent i3. P1-3: one legacy paragraph with no `data-id`.
- Version `v1` (owner mia): `<h2 data-id="v-h">Version heading</h2><p data-id="v-p">Version <strong>bold</strong> text</p>`.
- entity-search returns `user_mention` with `u2 mia`; the asset endpoint returns a 2×1 PNG. No estimates, pages or epics.
- Unstubbed on purpose (layout around the editor; listed in the output): notifications/unread, user-favorites, project user-properties, intake-state, cycles, modules (workspace and project), views.

##### 4. Run command
From the repository root: `node $P2TMP/probe-2/editor-probe.mjs`. Afterwards `lsof -nP -iTCP:<port>` and `pgrep -fl "editor-probe.mjs|chrome-headless-shell|ms-playwright"` both exit 1.
Mutation run: copy `build/client` to `build-pre-79ef2b5`; in `assets/use-editor-flagging-U_MB16gH.js` replace the single occurrence of
`onCreate(){!this.editor.isEditable||!this.options.updateDocument||PP(this.editor.view,this.options)}` with
`onCreate(){this.editor.isEditable||(this.options.updateDocument=!1),this.options.updateDocument&&PP(this.editor.view,this.options)}`;
then `PROBE_BUILD=$P2TMP/probe-2/build-pre-79ef2b5 node $P2TMP/probe-2/editor-probe.mjs`.

##### 5. Checks (all 7.5 "P2 编辑器", app level)
1. Control: opened without `#id`, block `b-alpha-2` starts below the fold (makes 3 meaningful).
2. Image: the `image-component` renders an `<img>` whose src resolves to `/api/assets/v2/workspaces/probe-ws/projects/p1/asset-1/` and loads (`naturalWidth` 2).
3. Block id and location: exactly one `[data-id="b-alpha-2"]` holding the stored text; `/probe-ws/browse/P1-1/#b-alpha-2` scrolls it into the viewport (`scrollToNode` by id).
4. Save: click at the end of the first paragraph, Enter, type → `PATCH …/projects/p1/issues/i1/` whose `description_html` contains the typed text.
5. Ids: every top-level block of the sent HTML has a `data-id`, all distinct, the new paragraph a fresh one, stored ids `b-alpha-1`, `b-alpha-2`, `img-1` kept.
6. Undo: `ControlOrMeta+z` removes the typed text.
7. Redo: `ControlOrMeta+Shift+z` brings it back.
8. History dropdown: "Last edited by…" lists v1 by "mia" (`…/work-items/i1/description-versions/`).
9. View: choosing v1 shows it in a read-only editor (`contenteditable=false`, `#editor-container-v1`).
10. Copy Markdown: clipboard pre-filled with a placeholder; after "Copy markdown" it reads `## Version heading\n\nVersion **bold** text\n`, no HTML.
11. Restore: the description becomes the version; the next PATCH for i1 carries the version's HTML, not the old text.
12. Read-only first: the existing comment renders in a read-only editor before anything else opens.
13. Mention list: typing `@` in the comment box lists "mia" (`GET …/entity-search/`).
14. Mention pick: an `@mia` link appears in the comment box.
15. Comment create: "Comment" sends `POST …/issues/i2/comments/` whose `comment_html` holds exactly one `<mention-component entity_identifier="u2" entity_name="user_mention" id=<uuid>>`; the new comment shows `@mia`.
16. Id migration after a read-only editor: the parent chip opens P1-3 in the peek (its editor created after the read-only comment editor); the legacy block gets an id — a PATCH for i3 with `skip_activity:"true"` and a `data-id`.
17. Edit after read-only: typing in the peek sends a `description_html` where every block has its own `data-id` (the 79ef2b5 guard).

##### 6. Final output (`run-final.txt`, exit 0)
Checks 1–17 PASS. Sent HTML (4): `…data-id="b-alpha-1">Existing alpha text</p><p … data-id="5543025b-…">Typed by probe</p><image-component data-id="img-1" src="asset-1" id="img-1" …>[30 fillers]<p … data-id="b-alpha-2">…`. Sent comment: `Hello <mention-component id="a15abf33-…" entity_identifier="u2" entity_name="user_mention"></mention-component>`. Migration PATCH: `{"description_html":"<p … data-id=\"b9c5434e-…\">Legacy gamma text without an id</p>","skip_activity":"true"}`. Unstubbed requests (404): 8, all listed above. Console errors — page A0: 20 caused by unstubbed requests / 0 other; A: 16 / 0; B: 10 / 0 (the 404s, `WorkspaceNotificationStore…404`, "Failed to fetch favorites from workspace store" from `favorite.store.ts:437`); no unhandled rejections. `port 64454 is free`.
Mutation run (`run-mutated.txt`, exit 1): 15 PASS, exactly 16 and 17 FAIL — `no PATCH for P1-3 after it opened`; `blocks without data-id: [{"tag":"p","id":null,"text":"Legacy gamma text without an id"},{"tag":"p","id":null,"text":"Typed after read-only"}]`. The full page cannot show this regression (its description editor is created before any read-only editor), hence the peek. The checks demand distinct ids: without the plugin, splitting a block copies its id, giving a duplicate rather than a missing id.

##### 7. Inherited from Plane (code unchanged since the import 48e1a63; not P2 regressions)
- Quick undo + redo inside the 1.5 s save delay saves the undone text while the editor shows the typed text (`saves sent after undo+redo: [{"has_typed_text":false}]; editor shows the typed text: true`). Undo schedules a save with the undone HTML; redo returns early because its HTML equals the last saved content, so the pending save keeps the undone version. `web/apps/web/core/components/editor/rich-text/description-input/root.tsx`. Not an assertion; backlog.
- `#id` links scroll to the block but never highlight it: `NodeHighlightPlugin` (`web/packages/editor/src/plugins/highlight.ts`) is registered nowhere, nor was it at the import.
- A link typed without the trailing slash loses its `#id`: the trailing-slash redirect in `web/apps/web/app/layout.tsx` builds from `request.url`, which has no fragment. App-generated links end in `/` and keep it.
Test-script timing (not an app bug): after the focusing click the editor adopts the clicked position a few ms later; `clickEndOf` pauses 500 ms, checks the caret is at the block end and retries; 10/10 placements correct afterwards.

##### 8. Not covered
Image upload, duplication and image restore (uploads, signed URLs; the editor package tests cover inserting an image node); the real asset endpoint's redirect; an @mention inside the description (same search path); collaboration (deleted by P2).

#### 脚本全文（`$P2TMP/probe-2/editor-probe.mjs`，假数据在其中）

```js
// One-off (M1/P2 retained-behaviour probe 2, M1 design 7.5 row "P2 编辑器", app-level part): serves the built
// web app (web/apps/web/build/client, SPA fallback to index.html) on a free port, stubs every Plane /api request
// with minimal fake data, and drives the editor on a work item's full page (/probe-ws/browse/P1-<n>) and its
// peek: description typing and save (with UniqueID data-id on every block), undo/redo, ids after a read-only
// editor (the 79ef2b5 regression), a comment with an @mention, the description history (view, copy Markdown,
// restore), and an image node plus id-based node location. Every /api request without a stub gets 404 and is
// listed at the end. Prints PASS/FAIL per check and exits non-zero if any check fails. Run from the repository
// root (Playwright comes from e2e/). usage: node editor-probe.mjs (PROBE_BUILD=<dir> serves another build directory,
// e.g. a copy of the build with a fix reverted, to show which checks catch it)
import { createRequire } from "node:module";
import fs from "node:fs";
import http from "node:http";
import path from "node:path";

const { chromium } = createRequire(path.resolve("e2e/package.json"))("@playwright/test");

// ---------------------------------------------------------------- static server over the build (SPA fallback)
const root = path.resolve(process.env.PROBE_BUILD ?? "web/apps/web/build/client"); // PROBE_BUILD: a mutated copy
const shell = path.join(root, "index.html"); // the only HTML file React Router's SPA build emits
const mime = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".json": "application/json",
  ".svg": "image/svg+xml", ".png": "image/png", ".webp": "image/webp", ".gif": "image/gif", ".ico": "image/x-icon",
  ".woff2": "font/woff2", ".woff": "font/woff", ".ttf": "font/ttf" };
const server = http.createServer((req, res) => {
  const file = path.join(root, decodeURIComponent(new URL(req.url, "http://x").pathname));
  const hit = file.startsWith(root + path.sep) && fs.existsSync(file) && fs.statSync(file).isFile();
  res.writeHead(200, { "content-type": mime[path.extname(hit ? file : shell)] ?? "application/octet-stream" });
  fs.createReadStream(hit ? file : shell).pipe(res);
});

// ---------------------------------------------------------------- fake data
const T0 = "2026-09-01T09:00:00Z";
const PNG = Buffer.from("iVBORw0KGgoAAAANSUhEUgAAAAIAAAABCAIAAAB7QOjdAAAAD0lEQVR4nGP4z8DAwPAfAAcAAf9+CLHQAAAAAElFTkSuQmCC", "base64"); // 2x1 RGB
const member = (id, display_name, first_name) => ({ id, display_name, first_name, last_name: "", email: `${display_name}@example.com`, avatar_url: "" });
const issue = (id, sequence_id, name, description_html, extra = {}) => ({
  id, sequence_id, name, description_html, project_id: "p1", workspace_id: "w1", state_id: "s1", priority: "none",
  assignee_ids: [], label_ids: [], module_ids: [], cycle_id: null, parent_id: null, start_date: null, target_date: null,
  created_by: "u1", created_at: T0, updated_at: T0, archived_at: null, is_draft: false, sort_order: 1,
  sub_issues_count: 0, attachment_count: 0, link_count: 0, ...extra,
});
// 30 short filler paragraphs push the block to locate below the fold
const filler = Array.from({ length: 30 }, (_, n) => `<p data-id="b-fill-${n}">Filler ${n}</p>`).join("");
const issues = {
  // P1-1: an image component node, then filler, then the block a #id link locates; all blocks already carry ids
  i1: issue("i1", 1, "Alpha item",
    '<p data-id="b-alpha-1">Existing alpha text</p>' +
    '<image-component src="asset-1" id="img-1" width="120px" height="60px" aspectratio="2" alignment="left" status="uploaded"></image-component>' +
    filler + '<p data-id="b-alpha-2">Second block to locate</p>'),
  // P1-2: has a comment (rendered by a read-only editor) and P1-3 as its parent
  i2: issue("i2", 2, "Beta item", '<p data-id="b-beta-1">Beta description</p>',
    { parent_id: "i3", parent: { id: "i3", project_id: "p1", name: "Gamma parent", sequence_id: 3 } }),
  // P1-3: a legacy description whose block has no id yet
  i3: issue("i3", 3, "Gamma parent", "<p>Legacy gamma text without an id</p>"),
};
const comment = (id, issueId, comment_html, created_at = "2026-09-02T09:00:00Z") => ({
  id, issue: issueId, project: "p1", workspace: "w1", actor: "u1", actor_detail: member("u1", "probe", "Pro"),
  comment_html, access: "INTERNAL", comment_reactions: [], created_at, updated_at: created_at, edited_at: null,
});
const comments = { i2: [comment("c1", "i2", '<p data-id="b-comment-1">Earlier comment on beta</p>')] };
const version = { id: "v1", owned_by: "u2", project: "p1", created_by: "u2", updated_by: "u2",
  created_at: "2026-09-10T09:00:00Z", updated_at: "2026-09-10T09:00:00Z", last_saved_at: "2026-09-10T09:00:00Z" };
const versionHTML = '<h2 data-id="v-h">Version heading</h2><p data-id="v-p">Version <strong>bold</strong> text</p>';

const W = "/api/workspaces/probe-ws";
const P = `${W}/projects/p1`;
const stubs = {
  "GET /api/instances/": { instance: { is_setup_done: true }, config: { is_email_password_enabled: true } },
  "GET /api/users/me/": { id: "u1", email: "probe@example.com", display_name: "probe", first_name: "Pro", last_name: "" },
  "GET /api/users/me/profile/": { id: "pr1", user: "u1", language: "en", is_onboarded: true, theme: {} },
  "GET /api/users/me/settings/": { id: "u1", email: "probe@example.com", workspace: { last_workspace_slug: "probe-ws", last_workspace_id: "w1" } },
  "GET /api/users/me/workspaces/": [{ id: "w1", slug: "probe-ws", name: "Probe WS", role: 20 }],
  "GET /api/users/me/workspaces/probe-ws/project-roles/": { p1: 20 },
  [`GET ${W}/workspace-members/me/`]: { id: "wm1", member: "u1", workspace: "w1", role: 20 },
  [`GET ${W}/members/`]: [
    { id: "wm1", role: 20, is_active: true, created_at: T0, member: member("u1", "probe", "Pro") },
    { id: "wm2", role: 15, is_active: true, created_at: T0, member: member("u2", "mia", "Mia") },
  ],
  [`GET ${W}/projects/`]: [{ id: "p1", identifier: "P1", name: "Probe project", workspace: "w1", is_member: true, member_role: 20, sort_order: 1, network: 2 }],
  [`GET ${W}/states/`]: [{ id: "s1", name: "Todo", group: "unstarted", color: "#3a3a3a", project_id: "p1", sequence: 1, default: true }],
  [`GET ${W}/user-properties/`]: { navigation_project_limit: 10, navigation_control_preference: "ACCORDION" },
  [`GET ${W}/entity-search/`]: { user_mention: [{ member__id: "u2", member__display_name: "mia", member__avatar_url: "" }] },
  [`GET ${W}/work-items/P1-1/`]: issues.i1,
  [`GET ${W}/work-items/P1-2/`]: issues.i2,
  [`GET ${P}/issues/i3/`]: issues.i3,
  [`GET ${P}/`]: { id: "p1", identifier: "P1", name: "Probe project", workspace: "w1", is_member: true, member_role: 20, sort_order: 1, network: 2 },
  [`GET ${P}/project-members/me/`]: { id: "pm1", member: "u1", role: 20 },
  [`GET ${P}/members/`]: [{ id: "pm1", member: "u1", role: 20 }, { id: "pm2", member: "u2", role: 15 }],
  [`GET ${P}/states/`]: [{ id: "s1", name: "Todo", group: "unstarted", color: "#3a3a3a", project_id: "p1", sequence: 1, default: true }],
  [`GET ${P}/issue-labels/`]: [],
  [`GET ${P}/work-items/i1/description-versions/`]: { results: [version], total_results: 1, total_pages: 1, page_count: 1,
    cursor: "", next_cursor: null, prev_cursor: null, next_page_results: false, prev_page_results: false },
  [`GET ${P}/work-items/i1/description-versions/v1/`]: { ...version, description_html: versionHTML, description_json: null, description_stripped: null },
  "GET /api/assets/v2/workspaces/probe-ws/projects/p1/asset-1/": { body: PNG, contentType: "image/png" },
};
for (const id of Object.keys(issues)) {
  stubs[`GET ${P}/issues/${id}/history/`] = (url) => (url.searchParams.get("activity_type") === "issue-comment" ? (comments[id] ?? []) : []);
  stubs[`GET ${P}/issues/${id}/sub-issues/`] = { sub_issues: [], state_distribution: {} };
  stubs[`GET ${P}/issues/${id}/issue-relation/`] = {};
  stubs[`GET ${P}/work-items/${id}/description-versions/`] ??= { results: [], total_results: 0 };
  stubs[`PATCH ${P}/issues/${id}/`] = (url, body) => ({ ...issues[id], ...body });
  stubs[`POST ${P}/issues/${id}/comments/`] = (url, body) => comment("c-new", id, body.comment_html, new Date().toISOString());
}

// ---------------------------------------------------------------- run
const requests = []; // every /api request: { page, method, path, search, body, stubbed }
const consoleErrors = []; // { page, text }
const results = [];
const check = async (name, fn) => {
  try {
    await fn();
    results.push(true);
    console.log(`PASS ${name}`);
  } catch (error) {
    results.push(false);
    console.log(`FAIL ${name}: ${error.message}`);
  }
};
const assert = (ok, detail) => {
  if (!ok) throw new Error(detail);
};
const since = (mark, pred) => requests.slice(mark).find(pred);
const waitForRequest = async (page, mark, pred, timeout = 10000) => {
  for (const t0 = Date.now(); Date.now() - t0 < timeout; await page.waitForTimeout(100)) {
    const hit = since(mark, pred);
    if (hit) return hit;
  }
  return undefined;
};
const isPatchOf = (id) => (r) => r.method === "PATCH" && r.path === `${P}/issues/${id}/` && r.body?.description_html !== undefined;
// top-level blocks of an HTML fragment as the browser parses it
const blocksOf = (page, html) =>
  page.evaluate((h) => [...new DOMParser().parseFromString(h, "text/html").body.children].map((el) =>
    ({ tag: el.tagName.toLowerCase(), id: el.getAttribute("data-id"), text: el.textContent })), html);
// clicks just left of the last character's right edge of a block, once it stops moving (the peek slides in),
// pauses like a person would (ProseMirror adopts the clicked position a few ms after the click that focuses
// the editor; a key pressed at machine speed would still act on the old selection at the start of the document),
// and retries until the caret really sits at the end of that block's text
const clickEndOf = async (page, block) => {
  const endOfText = () => block.evaluate((el) => {
    const range = document.createRange();
    range.selectNodeContents(el);
    const rects = range.getClientRects();
    const last = rects[rects.length - 1];
    return { x: last.right - 1, y: last.top + last.height / 2 };
  });
  const caretAtEnd = () => block.evaluate((el) => {
    const s = getSelection();
    const node = s.anchorNode;
    return s.isCollapsed && node?.nodeType === Node.TEXT_NODE && el.contains(node) &&
      s.anchorOffset === node.textContent.length && el.textContent.endsWith(node.textContent);
  });
  for (let attempt = 0; attempt < 5; attempt++) {
    await block.scrollIntoViewIfNeeded();
    let at = await endOfText();
    for (let n = 0; n < 50; n++) {
      await page.waitForTimeout(100);
      const next = await endOfText();
      if (next.x === at.x && next.y === at.y) break;
      at = next;
    }
    await page.mouse.click(at.x, at.y);
    await page.waitForTimeout(500);
    if (await caretAtEnd()) return;
  }
  throw new Error("could not put the caret at the end of the block");
};
const inViewport = (locator) =>
  locator.evaluate((el) => {
    const r = el.getBoundingClientRect();
    return r.top >= 0 && r.bottom <= window.innerHeight;
  });
const assertIds = (blocks) => {
  const ids = blocks.map((b) => b.id);
  assert(ids.every(Boolean), `blocks without data-id: ${JSON.stringify(blocks.filter((b) => !b.id))}`);
  assert(new Set(ids).size === ids.length, `duplicate data-id values: ${JSON.stringify(ids)}`);
};

await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
const { port } = server.address();
const origin = `http://127.0.0.1:${port}`;
const browser = await chromium.launch();
try {
  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
  await context.grantPermissions(["clipboard-read", "clipboard-write"], { origin });
  // name the reason of every unhandled rejection (Playwright's pageerror only says "Object" for non-Error values)
  await context.addInitScript(() =>
    window.addEventListener("unhandledrejection", (e) => console.error(`unhandledrejection ${JSON.stringify(e.reason)}`)));
  const openPage = async (label, target) => {
    const page = await context.newPage();
    page.on("console", (m) => m.type() === "error" && consoleErrors.push({ page: label, text: m.text().split("\n")[0].slice(0, 200) }));
    page.on("pageerror", (e) => consoleErrors.push({ page: label, text: `pageerror: ${String(e.message).split("\n")[0].slice(0, 200)}` }));
    await page.route("**/api/**", (route) => {
      const request = route.request();
      const url = new URL(request.url());
      let body;
      try { body = request.postDataJSON(); } catch { body = request.postData(); }
      const stub = stubs[`${request.method()} ${url.pathname}`];
      requests.push({ page: label, method: request.method(), path: url.pathname, search: url.search, body, stubbed: stub !== undefined });
      if (stub === undefined) return route.fulfill({ status: 404, json: { detail: "no probe stub", path: url.pathname } });
      const value = typeof stub === "function" ? stub(url, body) : stub;
      return value?.body instanceof Buffer ? route.fulfill({ body: value.body, contentType: value.contentType }) : route.fulfill({ json: value });
    });
    await page.goto(`${origin}${target}`);
    return page;
  };

  // ============ page A0: P1-1 without a block id in the URL (control for the #id check) ============
  await check("S6 control: without a block id in the URL the block to locate starts below the fold", async () => {
    const a0 = await openPage("A0 P1-1", "/probe-ws/browse/P1-1/");
    const target = a0.locator('#editor-container-i1 .ProseMirror [data-id="b-alpha-2"]');
    await target.waitFor({ state: "attached", timeout: 20000 });
    await a0.waitForTimeout(1500);
    assert(!(await inViewport(target)), "the block is already in the viewport");
    // info only: the same link without the trailing slash goes through the app's trailing-slash redirect
    await a0.goto(`${origin}/probe-ws/browse/P1-1#b-alpha-2`);
    await target.waitFor({ state: "attached", timeout: 20000 });
    console.log(`  info: /probe-ws/browse/P1-1#b-alpha-2 lands on ${a0.url().replace(origin, "")}`);
    await a0.close();
  });

  // ============ page A: P1-1 full page, opened on a block id (S6, S1, S2, S5) ============
  const a = await openPage("A P1-1", "/probe-ws/browse/P1-1/#b-alpha-2");
  const descA = a.locator("#editor-container-i1 .ProseMirror");
  await descA.getByText("Existing alpha text").waitFor({ state: "attached", timeout: 20000 });

  await check("S6 the image node renders through the asset endpoint", async () => {
    const img = descA.locator('img[src$="/api/assets/v2/workspaces/probe-ws/projects/p1/asset-1/"]');
    await img.waitFor({ state: "attached", timeout: 10000 });
    await a.waitForFunction((el) => el.complete && el.naturalWidth === 2, await img.elementHandle(), { timeout: 10000 });
  });
  await check("S6 the stored block id is kept in the editor and the #id link scrolls to that block", async () => {
    const block = descA.locator('[data-id="b-alpha-2"]');
    assert((await block.count()) === 1, `blocks with data-id=b-alpha-2: ${await block.count()}`);
    assert((await block.innerText()) === "Second block to locate", `text: ${await block.innerText()}`);
    assert(await inViewport(block), `with #b-alpha-2 the block is not in the viewport (url ${a.url()})`);
  });

  let mark = requests.length;
  let saved;
  await check("S1 typing in the description sends it in description_html", async () => {
    await clickEndOf(a, descA.locator('[data-id="b-alpha-1"]'));
    await a.keyboard.press("Enter");
    await a.keyboard.type("Typed by probe");
    saved = await waitForRequest(a, mark, isPatchOf("i1"));
    assert(saved, `no PATCH ${P}/issues/i1/ with description_html; since typing: ${JSON.stringify(requests.slice(mark).map((r) => `${r.method} ${r.path}`))}`);
    assert(saved.body.description_html.includes("Typed by probe"), `sent: ${saved.body.description_html}`);
  });
  await check("S1 every block of the sent description carries a unique data-id", async () => {
    assert(saved, "no save to inspect");
    const blocks = await blocksOf(a, saved.body.description_html);
    assertIds(blocks);
    const typed = blocks.find((x) => x.text === "Typed by probe");
    assert(typed && !["b-alpha-1", "b-alpha-2", "img-1"].includes(typed.id), `typed block: ${JSON.stringify(typed)}; blocks: ${JSON.stringify(blocks.filter((x) => !x.id?.startsWith("b-fill-")))}`);
    assert(blocks.some((x) => x.id === "b-alpha-1") && blocks.some((x) => x.id === "b-alpha-2"), `stored ids lost: ${JSON.stringify(blocks)}`);
    const fillers = /(<p class="editor-paragraph-block" data-id="b-fill-\d+">Filler \d+<\/p>)+/;
    console.log(`  sent description_html: ${saved.body.description_html.replace(fillers, "[30 filler paragraphs, ids b-fill-0..29]")}`);
  });

  mark = requests.length;
  await check("S2 Cmd/Ctrl+Z removes the typed text", async () => {
    await a.keyboard.press("ControlOrMeta+z");
    await a.waitForTimeout(300);
    const text = await descA.innerText();
    assert(!text.includes("Typed by probe") && text.includes("Existing alpha text"), `editor text: ${JSON.stringify(text.slice(0, 80))}`);
  });
  await check("S2 Shift+Cmd/Ctrl+Z brings it back", async () => {
    await a.keyboard.press("ControlOrMeta+Shift+z");
    await a.waitForTimeout(300);
    const text = await descA.innerText();
    assert(text.includes("Typed by probe"), `editor text: ${JSON.stringify(text.slice(0, 80))}`);
  });
  await a.waitForTimeout(2500); // let the debounced save of undo/redo go out
  console.log(`  info: saves sent after undo+redo: ${JSON.stringify(requests.slice(mark).filter(isPatchOf("i1")).map((r) => ({ has_typed_text: r.body.description_html.includes("Typed by probe") })))}; editor shows the typed text: ${(await descA.innerText()).includes("Typed by probe")}`);

  const dialog = a.getByRole("dialog");
  await check("S5 the versions dropdown lists the stubbed version", async () => {
    await a.getByRole("button", { name: /Last edited by/ }).click();
    await a.getByRole("menuitem", { name: /mia/ }).waitFor({ timeout: 5000 });
  });
  await check("S5 viewing a version renders its content read-only", async () => {
    await a.getByRole("menuitem", { name: /mia/ }).click();
    const versionEditor = dialog.locator("#editor-container-v1 .ProseMirror");
    await versionEditor.getByText("Version heading").waitFor({ timeout: 10000 });
    assert((await versionEditor.innerText()).includes("Version bold text"), `version text: ${await versionEditor.innerText()}`);
    assert((await versionEditor.getAttribute("contenteditable")) === "false", "version editor is editable");
  });
  await check("S5 copy Markdown puts the version's Markdown on the clipboard", async () => {
    const copy = dialog.getByRole("button").filter({ hasNotText: /\S/ }).last(); // prev, next, copy: the icon-only buttons
    const tooltip = a.getByText("Copy markdown", { exact: true });
    for (let attempt = 0; attempt < 3 && !(await tooltip.isVisible()); attempt++) {
      await copy.hover();
      await tooltip.waitFor({ timeout: 3000 }).catch(() => {});
    }
    assert(await tooltip.isVisible(), 'no "Copy markdown" tooltip on the icon button');
    await a.evaluate(() => navigator.clipboard.writeText("clipboard before the copy"));
    await copy.click();
    const clip = await a.evaluate(() => navigator.clipboard.readText());
    assert(clip.includes("## Version heading") && clip.includes("Version **bold** text") && !clip.includes("<"), `clipboard: ${JSON.stringify(clip)}`);
    console.log(`  clipboard: ${JSON.stringify(clip)}`);
  });
  mark = requests.length;
  await check("S5 restore puts the version back and sends its description_html", async () => {
    await dialog.getByRole("button", { name: "Restore" }).click();
    await descA.getByText("Version heading").waitFor({ timeout: 5000 });
    const restored = await waitForRequest(a, mark, isPatchOf("i1"));
    assert(restored, "no PATCH with description_html after restore");
    const html = restored.body.description_html;
    assert(html.includes("Version heading") && html.includes("Version <strong>bold</strong> text") && !html.includes("Existing alpha text"), `sent: ${html}`);
    console.log(`  sent description_html: ${html}`);
  });

  // ============ page B: P1-2 full page with a comment on display (S4), then its parent P1-3 in the peek (S3) ============
  const b = await openPage("B P1-2", "/probe-ws/browse/P1-2/");
  await check("S3 an existing comment renders in a read-only editor", async () => {
    const shownComment = b.locator(".ProseMirror", { hasText: "Earlier comment on beta" });
    await shownComment.waitFor({ timeout: 20000 });
    assert((await shownComment.getAttribute("contenteditable")) === "false", "comment editor is editable");
  });

  const commentBox = b.locator("#editor-container-add_comment_i2 .ProseMirror");
  await check("S4 typing @ in the comment box lists the stubbed member", async () => {
    await commentBox.click();
    await b.waitForTimeout(500); // see clickEndOf
    await b.keyboard.type("Hello @");
    await b.getByRole("button", { name: /mia/ }).waitFor({ timeout: 10000 });
  });
  await check("S4 picking the member puts an @mia mention in the comment box", async () => {
    await b.getByRole("button", { name: /mia/ }).click();
    await commentBox.getByRole("link", { name: "@mia" }).waitFor({ timeout: 5000 });
  });
  mark = requests.length;
  await check("S4 the created comment carries a mention node for that member and shows it", async () => {
    await b.getByRole("button", { name: "Comment", exact: true }).click();
    const created = await waitForRequest(b, mark, (r) => r.method === "POST" && r.path === `${P}/issues/i2/comments/`);
    assert(created, "no POST comments request");
    const html = created.body.comment_html;
    console.log(`  sent comment_html: ${html}`);
    const mentions = await b.evaluate((h) => [...new DOMParser().parseFromString(h, "text/html").querySelectorAll("mention-component")]
      .map((el) => Object.fromEntries([...el.attributes].map((x) => [x.name, x.value]))), html);
    assert(mentions.length === 1 && mentions[0].entity_identifier === "u2" && mentions[0].entity_name === "user_mention", `mention nodes: ${JSON.stringify(mentions)}`);
    await b.locator('.ProseMirror[contenteditable="false"]', { hasText: "Hello" }).getByRole("link", { name: "@mia" }).waitFor({ timeout: 5000 });
  });

  mark = requests.length;
  await check("S3 the parent opened after it (peek) still gets ids: the id-only migration update goes out", async () => {
    await b.getByRole("link", { name: /Gamma parent/ }).first().click(); // the parent chip above the title
    await b.locator("#editor-container-i3 .ProseMirror").getByText("Legacy gamma text without an id").waitFor({ timeout: 15000 });
    const migrated = await waitForRequest(b, mark, isPatchOf("i3"));
    assert(migrated, "no PATCH for P1-3 after it opened");
    assert(migrated.body.skip_activity === "true", `body: ${JSON.stringify(migrated.body)}`);
    assertIds(await blocksOf(b, migrated.body.description_html));
    console.log(`  sent: ${JSON.stringify(migrated.body)}`);
  });
  mark = requests.length;
  await check("S3 editing that description after the read-only editor keeps a unique data-id on every block", async () => {
    const peekEditor = b.locator("#editor-container-i3 .ProseMirror");
    await clickEndOf(b, peekEditor.locator("p", { hasText: "Legacy gamma text without an id" }));
    await b.keyboard.press("Enter");
    await b.keyboard.type("Typed after read-only");
    const sent = await waitForRequest(b, mark, (r) => isPatchOf("i3")(r) && r.body.description_html.includes("Typed after read-only"));
    assert(sent, "no PATCH for P1-3 with the typed text");
    assertIds(await blocksOf(b, sent.body.description_html));
    console.log(`  sent description_html: ${sent.body.description_html}`);
  });
} finally {
  await browser.close();
  await new Promise((resolve) => server.close(resolve));
}

// ---------------------------------------------------------------- report
const unstubbed = [...new Set(requests.filter((r) => !r.stubbed).map((r) => `${r.method} ${r.path}`))];
console.log(`\nunstubbed requests (404): ${unstubbed.length}`);
for (const r of unstubbed) console.log(`  ${r}`);
// errors the unstubbed requests cause: the 404 itself, the rejection naming the unstubbed path, and the message
// favorite.store.ts logs when GET /user-favorites/ fails
const fromUnstubbed = (t) => /404|no probe stub/.test(t) || t === "Failed to fetch favorites from workspace store";
const tally = (list) => Object.entries(list.reduce((acc, e) => ({ ...acc, [e.text]: (acc[e.text] ?? 0) + 1 }), {}));
for (const label of [...new Set(consoleErrors.map((e) => e.page))]) {
  const errs = consoleErrors.filter((e) => e.page === label);
  const expected = errs.filter((e) => fromUnstubbed(e.text));
  const others = errs.filter((e) => !fromUnstubbed(e.text));
  console.log(`console errors on page ${label}: ${expected.length} caused by unstubbed requests, ${others.length} others`);
  for (const [text, n] of tally(expected)) console.log(`  [unstubbed] ${n}x ${text}`);
  for (const [text, n] of tally(others)) console.log(`  [OTHER] ${n}x ${text}`);
}
const free = await new Promise((resolve) => {
  const probe = http.createServer().once("error", () => resolve(false)).listen(port, "127.0.0.1", () => probe.close(() => resolve(true)));
});
console.log(`port ${port} is ${free ? "free" : "STILL IN USE"}`);
const failed = results.filter((ok) => !ok).length;
console.log(`${results.length - failed} passed, ${failed} failed`);
process.exit(failed || !free ? 1 : 0);
```

#### 在 `36172ca` 构建上的最终输出

```
  info: /probe-ws/browse/P1-1#b-alpha-2 lands on /probe-ws/browse/P1-1/
PASS S6 control: without a block id in the URL the block to locate starts below the fold
PASS S6 the image node renders through the asset endpoint
PASS S6 the stored block id is kept in the editor and the #id link scrolls to that block
PASS S1 typing in the description sends it in description_html
  sent description_html: <p class="editor-paragraph-block" data-id="b-alpha-1">Existing alpha text</p><p class="editor-paragraph-block" data-id="5543025b-0c7e-4abe-b70e-fea06421b97f">Typed by probe</p><image-component data-id="img-1" src="asset-1" id="img-1" width="120px" height="60px" aspectratio="2" alignment="left" status="uploaded"></image-component>[30 filler paragraphs, ids b-fill-0..29]<p class="editor-paragraph-block" data-id="b-alpha-2">Second block to locate</p>
PASS S1 every block of the sent description carries a unique data-id
PASS S2 Cmd/Ctrl+Z removes the typed text
PASS S2 Shift+Cmd/Ctrl+Z brings it back
  info: saves sent after undo+redo: [{"has_typed_text":false}]; editor shows the typed text: true
PASS S5 the versions dropdown lists the stubbed version
PASS S5 viewing a version renders its content read-only
  clipboard: "## Version heading\n\nVersion **bold** text\n"
PASS S5 copy Markdown puts the version's Markdown on the clipboard
  sent description_html: <h2 class="editor-heading-block" data-id="v-h">Version heading</h2><p class="editor-paragraph-block" data-id="v-p">Version <strong>bold</strong> text</p>
PASS S5 restore puts the version back and sends its description_html
PASS S3 an existing comment renders in a read-only editor
PASS S4 typing @ in the comment box lists the stubbed member
PASS S4 picking the member puts an @mia mention in the comment box
  sent comment_html: <p class="editor-paragraph-block" data-id="a3ba35cc-89a5-4051-b0aa-4f67db234cf1">Hello <mention-component id="a15abf33-62d8-42fe-a2d7-f738a164f58b" entity_identifier="u2" entity_name="user_mention"></mention-component> </p>
PASS S4 the created comment carries a mention node for that member and shows it
  sent: {"description_html":"<p class=\"editor-paragraph-block\" data-id=\"b9c5434e-0d4f-4474-b4d9-4944ba834e95\">Legacy gamma text without an id</p>","skip_activity":"true"}
PASS S3 the parent opened after it (peek) still gets ids: the id-only migration update goes out
  sent description_html: <p class="editor-paragraph-block" data-id="b9c5434e-0d4f-4474-b4d9-4944ba834e95">Legacy gamma text without an id</p><p class="editor-paragraph-block" data-id="0c4fe165-6668-4d90-a0aa-b638a691b333">Typed after read-only</p>
PASS S3 editing that description after the read-only editor keeps a unique data-id on every block

unstubbed requests (404): 8
  GET /api/workspaces/probe-ws/users/notifications/unread/
  GET /api/workspaces/probe-ws/user-favorites/
  GET /api/workspaces/probe-ws/projects/p1/user-properties/
  GET /api/workspaces/probe-ws/projects/p1/intake-state/
  GET /api/workspaces/probe-ws/projects/p1/cycles/
  GET /api/workspaces/probe-ws/modules/
  GET /api/workspaces/probe-ws/projects/p1/modules/
  GET /api/workspaces/probe-ws/projects/p1/views/
console errors on page A0 P1-1: 20 caused by unstubbed requests, 0 others
  [unstubbed] 16x Failed to load resource: the server responded with a status of 404 (Not Found)
  [unstubbed] 2x WorkspaceNotificationStore -> getUnreadNotificationsCount -> error AxiosError: Request failed with status code 404
  [unstubbed] 2x Failed to fetch favorites from workspace store
console errors on page A P1-1: 16 caused by unstubbed requests, 0 others
  [unstubbed] 12x Failed to load resource: the server responded with a status of 404 (Not Found)
  [unstubbed] 2x WorkspaceNotificationStore -> getUnreadNotificationsCount -> error AxiosError: Request failed with status code 404
  [unstubbed] 2x Failed to fetch favorites from workspace store
console errors on page B P1-2: 10 caused by unstubbed requests, 0 others
  [unstubbed] 8x Failed to load resource: the server responded with a status of 404 (Not Found)
  [unstubbed] 1x WorkspaceNotificationStore -> getUnreadNotificationsCount -> error AxiosError: Request failed with status code 404
  [unstubbed] 1x Failed to fetch favorites from workspace store
port 64454 is free
17 passed, 0 failed
exit=0
```

#### 去掉 `79ef2b5` 修复之后的构建副本上的输出

```
  info: /probe-ws/browse/P1-1#b-alpha-2 lands on /probe-ws/browse/P1-1/
PASS S6 control: without a block id in the URL the block to locate starts below the fold
PASS S6 the image node renders through the asset endpoint
PASS S6 the stored block id is kept in the editor and the #id link scrolls to that block
PASS S1 typing in the description sends it in description_html
  sent description_html: <p class="editor-paragraph-block" data-id="b-alpha-1">Existing alpha text</p><p class="editor-paragraph-block" data-id="264fe7a8-6b8a-4711-87f6-f8184b11d2d6">Typed by probe</p><image-component data-id="img-1" src="asset-1" id="img-1" width="120px" height="60px" aspectratio="2" alignment="left" status="uploaded"></image-component>[30 filler paragraphs, ids b-fill-0..29]<p class="editor-paragraph-block" data-id="b-alpha-2">Second block to locate</p>
PASS S1 every block of the sent description carries a unique data-id
PASS S2 Cmd/Ctrl+Z removes the typed text
PASS S2 Shift+Cmd/Ctrl+Z brings it back
  info: saves sent after undo+redo: [{"has_typed_text":false}]; editor shows the typed text: true
PASS S5 the versions dropdown lists the stubbed version
PASS S5 viewing a version renders its content read-only
  clipboard: "## Version heading\n\nVersion **bold** text\n"
PASS S5 copy Markdown puts the version's Markdown on the clipboard
  sent description_html: <h2 class="editor-heading-block" data-id="v-h">Version heading</h2><p class="editor-paragraph-block" data-id="v-p">Version <strong>bold</strong> text</p>
PASS S5 restore puts the version back and sends its description_html
PASS S3 an existing comment renders in a read-only editor
PASS S4 typing @ in the comment box lists the stubbed member
PASS S4 picking the member puts an @mia mention in the comment box
  sent comment_html: <p class="editor-paragraph-block" data-id="153beae6-d1c1-4636-8da7-df06cf000425">Hello <mention-component id="67c77382-87b9-4ce1-9efb-12ee6060a217" entity_identifier="u2" entity_name="user_mention"></mention-component> </p>
PASS S4 the created comment carries a mention node for that member and shows it
FAIL S3 the parent opened after it (peek) still gets ids: the id-only migration update goes out: no PATCH for P1-3 after it opened
FAIL S3 editing that description after the read-only editor keeps a unique data-id on every block: blocks without data-id: [{"tag":"p","id":null,"text":"Legacy gamma text without an id"},{"tag":"p","id":null,"text":"Typed after read-only"}]

unstubbed requests (404): 8
  GET /api/workspaces/probe-ws/users/notifications/unread/
  GET /api/workspaces/probe-ws/user-favorites/
  GET /api/workspaces/probe-ws/projects/p1/user-properties/
  GET /api/workspaces/probe-ws/projects/p1/intake-state/
  GET /api/workspaces/probe-ws/projects/p1/cycles/
  GET /api/workspaces/probe-ws/modules/
  GET /api/workspaces/probe-ws/projects/p1/modules/
  GET /api/workspaces/probe-ws/projects/p1/views/
console errors on page A0 P1-1: 20 caused by unstubbed requests, 0 others
  [unstubbed] 16x Failed to load resource: the server responded with a status of 404 (Not Found)
  [unstubbed] 2x WorkspaceNotificationStore -> getUnreadNotificationsCount -> error AxiosError: Request failed with status code 404
  [unstubbed] 2x Failed to fetch favorites from workspace store
console errors on page A P1-1: 18 caused by unstubbed requests, 0 others
  [unstubbed] 13x Failed to load resource: the server responded with a status of 404 (Not Found)
  [unstubbed] 3x WorkspaceNotificationStore -> getUnreadNotificationsCount -> error AxiosError: Request failed with status code 404
  [unstubbed] 2x Failed to fetch favorites from workspace store
console errors on page B P1-2: 16 caused by unstubbed requests, 0 others
  [unstubbed] 12x Failed to load resource: the server responded with a status of 404 (Not Found)
  [unstubbed] 2x WorkspaceNotificationStore -> getUnreadNotificationsCount -> error AxiosError: Request failed with status code 404
  [unstubbed] 2x Failed to fetch favorites from workspace store
port 64715 is free
15 passed, 2 failed
exit=1
```

#### 修复轮之后在 `ef3be19` 的构建上重跑

```
  info: /probe-ws/browse/P1-1#b-alpha-2 lands on /probe-ws/browse/P1-1/
PASS S6 control: without a block id in the URL the block to locate starts below the fold
PASS S6 the image node renders through the asset endpoint
PASS S6 the stored block id is kept in the editor and the #id link scrolls to that block
PASS S1 typing in the description sends it in description_html
  sent description_html: <p class="editor-paragraph-block" data-id="b-alpha-1">Existing alpha text</p><p class="editor-paragraph-block" data-id="d09ebb44-9297-48d3-b8bf-beab72459823">Typed by probe</p><image-component data-id="img-1" src="asset-1" id="img-1" width="120px" height="60px" aspectratio="2" alignment="left" status="uploaded"></image-component>[30 filler paragraphs, ids b-fill-0..29]<p class="editor-paragraph-block" data-id="b-alpha-2">Second block to locate</p>
PASS S1 every block of the sent description carries a unique data-id
PASS S2 Cmd/Ctrl+Z removes the typed text
PASS S2 Shift+Cmd/Ctrl+Z brings it back
  info: saves sent after undo+redo: [{"has_typed_text":false}]; editor shows the typed text: true
PASS S5 the versions dropdown lists the stubbed version
PASS S5 viewing a version renders its content read-only
  clipboard: "## Version heading\n\nVersion **bold** text\n"
PASS S5 copy Markdown puts the version's Markdown on the clipboard
  sent description_html: <h2 class="editor-heading-block" data-id="v-h">Version heading</h2><p class="editor-paragraph-block" data-id="v-p">Version <strong>bold</strong> text</p>
PASS S5 restore puts the version back and sends its description_html
PASS S3 an existing comment renders in a read-only editor
PASS S4 typing @ in the comment box lists the stubbed member
PASS S4 picking the member puts an @mia mention in the comment box
  sent comment_html: <p class="editor-paragraph-block" data-id="81ec189b-fec5-4929-888d-e5c0e5f52adc">Hello <mention-component id="54266fa4-7c8f-4cc4-9f8e-05e63d90e23e" entity_identifier="u2" entity_name="user_mention"></mention-component> </p>
PASS S4 the created comment carries a mention node for that member and shows it
  sent: {"description_html":"<p class=\"editor-paragraph-block\" data-id=\"3dc12374-0a09-4096-aee6-d328c090d038\">Legacy gamma text without an id</p>","skip_activity":"true"}
PASS S3 the parent opened after it (peek) still gets ids: the id-only migration update goes out
  sent description_html: <p class="editor-paragraph-block" data-id="3dc12374-0a09-4096-aee6-d328c090d038">Legacy gamma text without an id</p><p class="editor-paragraph-block" data-id="1ec697e5-bb2d-4b54-be2e-c37fd13c2141">Typed after read-only</p>
PASS S3 editing that description after the read-only editor keeps a unique data-id on every block

unstubbed requests (404): 8
  GET /api/workspaces/probe-ws/users/notifications/unread/
  GET /api/workspaces/probe-ws/user-favorites/
  GET /api/workspaces/probe-ws/projects/p1/user-properties/
  GET /api/workspaces/probe-ws/projects/p1/intake-state/
  GET /api/workspaces/probe-ws/projects/p1/cycles/
  GET /api/workspaces/probe-ws/modules/
  GET /api/workspaces/probe-ws/projects/p1/modules/
  GET /api/workspaces/probe-ws/projects/p1/views/
console errors on page A0 P1-1: 20 caused by unstubbed requests, 0 others
  [unstubbed] 16x Failed to load resource: the server responded with a status of 404 (Not Found)
  [unstubbed] 2x WorkspaceNotificationStore -> getUnreadNotificationsCount -> error AxiosError: Request failed with status code 404
  [unstubbed] 2x Failed to fetch favorites from workspace store
console errors on page A P1-1: 16 caused by unstubbed requests, 0 others
  [unstubbed] 12x Failed to load resource: the server responded with a status of 404 (Not Found)
  [unstubbed] 2x WorkspaceNotificationStore -> getUnreadNotificationsCount -> error AxiosError: Request failed with status code 404
  [unstubbed] 2x Failed to fetch favorites from workspace store
console errors on page B P1-2: 13 caused by unstubbed requests, 0 others
  [unstubbed] 10x Failed to load resource: the server responded with a status of 404 (Not Found)
  [unstubbed] 2x WorkspaceNotificationStore -> getUnreadNotificationsCount -> error AxiosError: Request failed with status code 404
  [unstubbed] 1x Failed to fetch favorites from workspace store
port 51926 is free
17 passed, 0 failed
```

### 6.4 临时核对脚本 3：动态、通知、工作项列表（7.5 "P2、P3 动态、通知、工作项列表"）

#### Probe 3 — activity, notifications, work-item lists (M1 design 7.5 row "P2、P3 动态、通知、工作项列表")

(Written by the controller from the probe agent's reply: the harness refused the agent's own REPORT.md write.)

Files in this directory, copied into the appendix as they are:
- `activity-notifications-lists-probe.mjs` — the script (475 lines, fake data inline)
- `run-final.txt` — output of the final run
- `make-negative.py`, `negative-control.mjs`, `run-negative.txt` — negative control
- `debug-patch.mjs` — helper used to explain the description PATCH

##### 1. Purpose
Against the branch-head build (36172ca, built 15:52), with Plane's API stubbed by representative data (no estimates, pages or epics): a work item's activity shows state, relation, cycle, module and comment entries; notifications list and preview, and "mark all as read" works; project / cycle / module / profile ("assigned") / archive pages each pick the right work-item store; the project view offers exactly list, board, table, calendar, and each renders.

##### 2. Script
P1 §5.5 model: header comment, Playwright via `createRequire` on `e2e/package.json`, own static server on port 0 with SPA fallback to `index.html` (React Router SPA mode), routes `**/api/**` and `**/auth/**`; every unstubbed request gets 404 and is listed at the end.

##### 3. Fake data (inline in the script)
- Start state as P1 (instance, users/me, onboarded profile, settings).
- Workspace `probe-ws`; users u1 "probe" (signed in), u2 "otto"; project p1 "Probe Project" (PRB); five states; cycle c1 "Sprint One"; module m1 "Module One".
- Work items: PRB-1 "Project item alpha" (cycle + module); PRB-2 "Cycle item gamma" (cycle); PRB-3 "Module item delta" (module); PRB-4 "Assigned item epsilon" (assigned to u1); PRB-5 "Archived item zeta" (archived).
- List endpoints filter by the query the page sends (`cycle`, `module`, `assignees`, `created_by`, `priority`) and group by `group_by`/`sub_group_by`, as Plane does.
- PRB-1 history: created; state Todo → In Progress; blocking PRB-2; added to Sprint One; added to Module One; one comment by otto.
- Notifications: two unread from otto (state change on PRB-1, comment on PRB-2); unread count 2.
- PRB-1's description carries a `data-id`. Without one, the editor's id migration sends `PATCH …/issues/i1/ {description_html, skip_activity:"true"}` about 2 s after opening — upstream Plane behaviour (`use-editor.ts` `uniqueIdOnlyChange`), not a P2 regression.

##### 4. Run command
```sh
cd <repo>
node "$P2TMP/probe-3/activity-notifications-lists-probe.mjs"   # exit 0 when every check passes, 1 otherwise
```

##### 5. Assertions (all under the 7.5 row above)
Activity: `/projects/p1/issues/i1/` redirects through `…/meta/` to `/browse/PRB-1/`; exactly 5 property entries (created, state, relation, cycle, module), none blank; texts "otto set the state to In Progress." / "otto marked this work item is blocking work item PRB-2." / "…to the cycle Sprint One" (link `/probe-ws/projects/p1/cycles/c1`) / "…to the module Module One" (link `/modules/m1`); the comment block shows "otto commented … Probe comment on alpha"; no raw i18n key, `undefined`, `null`, `NaN`.
Notifications: both cards render; before reading the top-nav Inbox dot shows, All tab 2, 2 card dots; opening n1 posts `…/n1/read/`, All → 1; the preview shows "Project item alpha" and its description; "Mark all as read" posts `…/mark-all-read/` with body exactly `{"snoozed":false,"archived":false}`; afterwards the Inbox dot, All count and card dots are gone.
Store per context: project → alpha, gamma, delta, epsilon (`projects/p1/issues/` without cycle/module); cycle → alpha, gamma (`?cycle=c1`); module → alpha, delta (`?module=m1`); profile → epsilon (`/profile/u1/` → `/assigned/`; `user-issues/u1/?assignees=u1`); archives → zeta (`archived-issues/`).
Layouts: switcher aria-labels exactly Board, Calendar, List, Table Layout; "gantt"/"timeline" nowhere in the page HTML; each switch PATCHes `user-properties` with the matching `display_filters.layout` and its DOM appears (Board columns with PRB-1/PRB-2; Table `td#issue-i1`/`#issue-i4`; Calendar "Mon" header with PRB-1 on its date; List rows); no error page or new page error after any switch.
End of run: port free, no browser process left (a leftover counts as failure).

##### 6. Final run output (`run-final.txt` has every line)
```
PASS … (24 lines, all PASS)
unstubbed requests (0):
browser console errors per scenario: activity 0, notifications 0, store-project 0, store-cycle 0, store-module 0, store-profile 0, store-archives 0, layouts 0
port 62957 is free
browser processes started by this run: 3; still running: none
all checks passed        (exit code 0)
```
External check: `lsof -nP -iTCP:62957` exits 1; `ps` shows no probe process.
Negative control (`run-negative.txt`), breakages in fake data only: N1 relation field `blocks`; N2 list endpoint ignores `cycle`; N3 `mark-all-read` unstubbed (404). Exactly 4 checks fail ("5 property entries" got 4; the relation entry; "after mark all read" dot stays; the cycle store shows all 4 titles); the rest pass.

##### 7. Not covered
Mobile layout switcher and the switchers in cycle/module/view/profile/archive headers (same `ISSUE_LAYOUTS` constant); calendar week view; board grouped by anything but state; assignee/label/date/link/attachment/parent/archive/intake activities and the activity filter/sort; notification tabs other than All, mentions, filters, snooze, archive, intake preview; profile "created"/"subscribed" tabs.
Observation (not a defect): `workspace/sidebar/user-menu.tsx`, `user-menu-item.tsx`, `workspace-notifications/notification-app-sidebar-option.tsx` are imported nowhere (`user-menu.tsx` still lists dashboards and pi-chat) — already in knip's unused-file list → P3.

#### 脚本全文（`$P2TMP/probe-3/activity-notifications-lists-probe.mjs`，假数据在其中）

```js
// One-off (M1/P2, design 7.5 row "P2、P3 动态、通知、工作项列表"): serves the built web app
// (web/apps/web/build/client, SPA fallback to index.html) on a free port, signs in a stubbed, onboarded user
// of workspace probe-ws / project p1, and checks in a real browser that
//   1. a work item's activity shows its state, relation, cycle, module and comment entries as readable text;
//   2. the notifications page lists the stubbed notifications, opening one shows the work item preview and
//      "mark all as read" posts the right request and clears the unread count and indicator;
//   3. the project, cycle, module, profile ("assigned") and archive pages each list the work items of THAT
//      context (every context has its own titles, so a wrong issue store shows the wrong titles or none);
//   4. the project work-item view offers exactly the list, board, table and calendar layouts and each renders.
// The stubs answer the Plane endpoints these pages call, with minimal data (no estimates, no pages, no epics);
// every other /api or /auth request gets 404 and is listed at the end. Run from the repository root
// (Playwright comes from e2e/). usage: node activity-notifications-lists-probe.mjs
import { execFileSync } from "node:child_process";
import { createRequire } from "node:module";
import fs from "node:fs";
import http from "node:http";
import net from "node:net";
import path from "node:path";

const { chromium } = createRequire(path.resolve("e2e/package.json"))("@playwright/test");
const ROOT = path.resolve("web/apps/web/build/client");

// ---------------------------------------------------------------- fake data
const W = "/api/workspaces/probe-ws";
const P = `${W}/projects/p1`;
const now = Date.now();
const iso = (t) => new Date(t).toISOString();
const localDate = (d) =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
// a weekday of the current month (the calendar hides weekends by default)
const dueDate = (() => {
  const d = new Date();
  const shift = { 0: 1, 6: -1 }[d.getDay()] ?? 0;
  const moved = new Date(d.getFullYear(), d.getMonth(), d.getDate() + shift);
  if (moved.getMonth() !== d.getMonth()) moved.setDate(moved.getDate() - 3 * Math.sign(shift));
  return localDate(moved);
})();
const users = {
  u1: { id: "u1", display_name: "probe", first_name: "Probe", last_name: "User", avatar_url: "", is_bot: false },
  u2: { id: "u2", display_name: "otto", first_name: "Otto", last_name: "Other", avatar_url: "", is_bot: false },
};
const states = [
  ["s-backlog", "Backlog", "backlog"],
  ["s-todo", "Todo", "unstarted"],
  ["s-progress", "In Progress", "started"],
  ["s-done", "Done", "completed"],
  ["s-cancel", "Cancelled", "cancelled"],
].map(([id, name, group], i) => ({
  id, name, group, color: "#888888", default: i === 1, description: "", project_id: "p1",
  sequence: (i + 1) * 1000, workspace_id: "w1", order: i,
}));
const project = {
  id: "p1", name: "Probe Project", identifier: "PRB", sort_order: 1, logo_props: { in_use: "emoji", emoji: { value: "128640" } },
  member_role: 20, archived_at: null, workspace: "w1", cycle_view: true, issue_views_view: true, module_view: true,
  inbox_view: false, network: 2, created_at: "2026-09-01T00:00:00Z", members: ["u1", "u2"], is_member: true, anchor: null,
};
const counts = { total_issues: 2, completed_issues: 0, backlog_issues: 0, started_issues: 1, unstarted_issues: 1, cancelled_issues: 0 };
const cycle = {
  id: "c1", name: "Sprint One", description: "", project_id: "p1", workspace_id: "w1", owned_by_id: "u1", sort_order: 1,
  start_date: iso(now - 7 * 86400000), end_date: iso(now + 7 * 86400000), status: "current", archived_at: null,
  is_favorite: false, view_props: { filters: {} }, project_detail: { id: "p1" }, assignee_ids: [], ...counts,
};
const module = {
  id: "m1", name: "Module One", description: "", description_text: null, description_html: null, workspace_id: "w1",
  project_id: "p1", lead_id: null, member_ids: [], is_favorite: false, sort_order: 1, view_props: { filters: {} },
  status: "in-progress", archived_at: null, start_date: null, target_date: null,
  created_at: "2026-09-01T00:00:00Z", updated_at: "2026-09-01T00:00:00Z", ...counts,
};
const issue = (id, seq, name, extra = {}) => ({
  id, sequence_id: seq, name, sort_order: seq * 1000, state_id: "s-todo", priority: "medium", label_ids: [],
  assignee_ids: [], sub_issues_count: 0, attachment_count: 0, link_count: 0, project_id: "p1", parent_id: null,
  cycle_id: null, module_ids: [], type_id: null, created_at: "2026-09-10T00:00:00Z", updated_at: "2026-09-10T00:00:00Z",
  start_date: null, target_date: dueDate, completed_at: null, archived_at: null, created_by: "u2", updated_by: "u2",
  is_draft: false, ...extra,
});
// the project's work items: alpha is in the cycle and the module, gamma only in the cycle, delta only in the
// module, epsilon is assigned to the signed-in user; zeta is archived. Alpha's description is stored the way the
// editor saves it (paragraph with a node id); without the id the editor's one-off id migration PATCHes it on view
const ISSUES = [
  issue("i1", 1, "Project item alpha", {
    state_id: "s-progress", cycle_id: "c1", module_ids: ["m1"],
    description_html: '<p class="editor-paragraph-block" data-id="5f1c2a4e-1111-4a2b-9c3d-000000000001">Alpha description text</p>',
  }),
  issue("i2", 2, "Cycle item gamma", { cycle_id: "c1" }),
  issue("i3", 3, "Module item delta", { module_ids: ["m1"] }),
  issue("i4", 4, "Assigned item epsilon", { assignee_ids: ["u1"] }),
];
const ARCHIVED = [issue("i5", 5, "Archived item zeta", { state_id: "s-done", archived_at: "2026-09-15T00:00:00Z" })];
const TITLES = [...ISSUES, ...ARCHIVED].map((i) => i.name);
// list endpoints answer like Plane: filtered by the context's query parameters, grouped by group_by/sub_group_by
const GROUP_PROP = {
  state_id: "state_id", priority: "priority", labels__id: "label_ids", assignees__id: "assignee_ids",
  cycle_id: "cycle_id", issue_module__module_id: "module_ids", target_date: "target_date",
  project_id: "project_id", created_by: "created_by",
};
const groupKeys = (i, g) => {
  const v = i[GROUP_PROP[g]];
  return Array.isArray(v) ? (v.length ? v : ["None"]) : [v ?? "None"];
};
const groupBy = (list, g) => {
  const out = {};
  for (const i of list) for (const k of groupKeys(i, g)) (out[k] ??= []).push(i);
  return out;
};
const wrap = (list, results = list) => ({ results, total_results: list.length });
const paginated = (list, q) => {
  const [g, sg] = [q.get("group_by"), q.get("sub_group_by")];
  const nest = (arr) =>
    sg ? Object.fromEntries(Object.entries(groupBy(arr, sg)).map(([k, sub]) => [k, wrap(sub)])) : arr;
  const results = g ? Object.fromEntries(Object.entries(groupBy(list, g)).map(([k, arr]) => [k, wrap(arr, nest(arr))])) : list;
  return {
    grouped_by: g, sub_grouped_by: sg, next_cursor: "100:1:0", prev_cursor: "100:-1:1", next_page_results: false,
    prev_page_results: false, total_count: list.length, count: list.length, total_pages: 1, extra_stats: null,
    results, total_results: list.length,
  };
};
const filtered = (q) =>
  ISSUES.filter(
    (i) =>
      (!q.get("cycle") || i.cycle_id === q.get("cycle")) &&
      (!q.get("module") || i.module_ids.includes(q.get("module"))) &&
      (!q.get("assignees") || i.assignee_ids.includes(q.get("assignees"))) &&
      (!q.get("created_by") || i.created_by === q.get("created_by")) &&
      (!q.get("priority") || q.get("priority").split(",").includes(i.priority)) &&
      !q.get("subscriber")
  );
const activity = (id, minutesAgo, extra) => ({
  id, workspace: "w1", workspace_detail: { id: "w1", slug: "probe-ws", name: "Probe WS" }, project: "p1",
  project_detail: { id: "p1", identifier: "PRB", name: "Probe Project" }, issue: "i1",
  issue_detail: { id: "i1", sequence_id: 1, name: "Project item alpha" }, actor: "u2", actor_detail: users.u2,
  created_at: iso(now - minutesAgo * 60000), updated_at: iso(now - minutesAgo * 60000), verb: "updated", field: null,
  old_value: "", new_value: "", old_identifier: null, new_identifier: null, comment: "", epoch: 0,
  issue_comment: null, attachments: [], ...extra,
});
const ACTIVITIES = [
  activity("a0", 60, { verb: "created" }),
  activity("a1", 50, { field: "state", old_value: "Todo", new_value: "In Progress", old_identifier: "s-todo", new_identifier: "s-progress" }),
  activity("a2", 40, { field: "blocking", new_value: "PRB-2", new_identifier: "i2" }),
  activity("a3", 30, { verb: "created", field: "cycles", new_value: "Sprint One", new_identifier: "c1" }),
  activity("a4", 20, { verb: "created", field: "modules", new_value: "Module One", new_identifier: "m1" }),
];
const COMMENTS = [
  {
    ...activity("cm1", 10, {}), comment_html: "<p>Probe comment on alpha</p>", comment_stripped: "Probe comment on alpha",
    comment_json: {}, comment_reactions: [], access: "INTERNAL", external_id: null, external_source: null,
  },
];
const notification = (id, item, issueActivity, minutesAgo) => ({
  id, title: item.name, entity_identifier: item.id, entity_name: "issue", message_html: "<p></p>",
  sender: "in_app:issue_activities:updated", receiver: "u1", triggered_by: "u2", triggered_by_details: users.u2,
  read_at: null, archived_at: null, snoozed_till: null, is_inbox_issue: false, is_mentioned_notification: false,
  workspace: "w1", project: "p1", created_at: iso(now - minutesAgo * 60000), updated_at: iso(now - minutesAgo * 60000),
  created_by: "u2", updated_by: "u2",
  data: {
    issue: { id: item.id, sequence_id: item.sequence_id, identifier: "PRB", name: item.name, state_name: "Todo", state_group: "unstarted" },
    issue_activity: { id: `na-${id}`, actor: "u2", issue_comment: null, ...issueActivity },
  },
});
const NOTIFICATIONS = [
  notification("n1", ISSUES[0], { field: "state", verb: "updated", old_value: "Todo", new_value: "In Progress" }, 5),
  notification("n2", ISSUES[1], { field: "comment", verb: "created", old_value: "", new_value: "<p>Looks good to me</p>" }, 3),
];
const filterProps = (layout) => ({ rich_filters: {}, display_filters: { layout }, display_properties: {} });
const emptyPage = { next_page_results: false, prev_page_results: false, results: [], total_pages: 0 };

// "METHOD path" keys answer that method only; bare path keys answer GET only; functions get (query, request)
const STUBS = {
  "/api/instances/": { instance: { is_setup_done: true }, config: { is_email_password_enabled: true } },
  "/api/users/me/": { id: "u1", email: "probe@example.com", display_name: "probe", first_name: "Probe", last_name: "User" },
  "/api/users/me/profile/": { id: "p1", user: "u1", language: "en", is_onboarded: true, theme: {} },
  "/api/users/me/settings/": { id: "u1", email: "probe@example.com", workspace: {} },
  "/api/users/me/workspaces/": [{ id: "w1", slug: "probe-ws", name: "Probe WS", role: 20 }],
  "/api/users/me/workspaces/probe-ws/project-roles/": { p1: 20 },
  [`${W}/workspace-members/me/`]: { id: "wm1", member: "u1", role: 20, workspace: "w1", view_props: {}, default_props: {}, draft_issue_count: 0 },
  [`${W}/members/`]: Object.values(users).map((member, i) => ({ id: `wm${i + 1}`, member, role: 20, is_active: true, created_at: "2026-09-01T00:00:00Z" })),
  [`${W}/projects/`]: [project],
  [`${W}/states/`]: states,
  [`${W}/labels/`]: [],
  [`${W}/cycles/`]: [cycle],
  [`${W}/modules/`]: [module],
  [`${W}/user-favorites/`]: [],
  [`${W}/user-properties/`]: { ...filterProps("list"), navigation_project_limit: 10, navigation_control_preference: "ACCORDION" },
  [`${W}/users/notifications/unread/`]: { total_unread_notifications_count: 2, mention_unread_notifications_count: 0 },
  [`${W}/users/notifications/`]: { ...emptyPage, next_cursor: "30:1:0", prev_cursor: "30:-1:1", count: 2, total_count: 2, total_pages: 1, results: NOTIFICATIONS },
  [`POST ${W}/users/notifications/n1/read/`]: { ...NOTIFICATIONS[0], read_at: iso(now) },
  [`POST ${W}/users/notifications/mark-all-read/`]: {},
  [`${W}/user-issues/u1/`]: (q) => paginated(filtered(q), q),
  [`${P}/`]: project,
  [`${P}/project-members/me/`]: { id: "pm1", member: "u1", role: 20, original_role: 20, created_at: "2026-09-01T00:00:00Z" },
  [`${P}/members/`]: Object.keys(users).map((member, i) => ({ id: `pm${i + 1}`, member, role: 20, original_role: 20, created_at: "2026-09-01T00:00:00Z" })),
  [`${P}/user-properties/`]: { ...filterProps("list"), sort_order: 1, preferences: { navigation: { default_tab: "work_items", hide_in_more_menu: [] } } },
  [`PATCH ${P}/user-properties/`]: (q, request) => request.postDataJSON(),
  [`${P}/issue-labels/`]: [],
  [`${P}/states/`]: states,
  [`${P}/intake-state/`]: states[0],
  [`${P}/views/`]: [],
  [`${P}/cycles/`]: [cycle],
  [`${P}/cycles/c1/`]: cycle,
  [`${P}/cycles/c1/user-properties/`]: filterProps("list"),
  [`${P}/cycles/c1/progress/`]: counts,
  [`${P}/cycles/c1/analytics/`]: { assignees: [], labels: [], completion_chart: {} },
  [`${P}/cycles/c1/cycle-issues/`]: (q) => paginated(filtered(q).filter((i) => i.cycle_id === "c1"), q),
  [`${P}/modules/`]: [module],
  [`${P}/modules/m1/`]: module,
  [`${P}/modules/m1/user-properties/`]: filterProps("list"),
  [`${P}/issues/`]: (q) => paginated(filtered(q), q),
  [`${P}/archived-issues/`]: (q) => paginated(ARCHIVED, q),
  [`${P}/issues/i1/`]: ISSUES[0],
  [`${P}/issues/i1/meta/`]: { project_identifier: "PRB", sequence_id: 1 },
  [`${W}/work-items/PRB-1/`]: ISSUES[0],
  [`${P}/issues/i1/history/`]: (q) => (q.get("activity_type") === "issue-comment" ? COMMENTS : ACTIVITIES),
  [`${P}/issues/i1/sub-issues/`]: { sub_issues: [], state_distribution: {} },
  [`${P}/issues/i1/issue-relation/`]: { blocking: [ISSUES[1]], blocked_by: [], duplicate: [], relates_to: [] },
  [`${P}/work-items/i1/description-versions/`]: { ...emptyPage, cursor: "", next_cursor: null, prev_cursor: null, page_count: 0 },
};

// ---------------------------------------------------------------- harness
const TYPES = { ".js": "text/javascript", ".css": "text/css", ".html": "text/html", ".json": "application/json", ".svg": "image/svg+xml", ".png": "image/png", ".gif": "image/gif", ".webp": "image/webp", ".jpg": "image/jpeg", ".ico": "image/x-icon", ".woff2": "font/woff2" };
const server = http.createServer((req, res) => {
  let file = path.join(ROOT, decodeURIComponent(new URL(req.url, "http://x").pathname));
  if (!file.startsWith(ROOT + path.sep) || !fs.existsSync(file) || fs.statSync(file).isDirectory())
    file = path.join(ROOT, "index.html"); // React Router SPA mode emits index.html as the shell
  res.writeHead(200, { "content-type": TYPES[path.extname(file)] ?? "application/octet-stream" });
  fs.createReadStream(file).pipe(res);
});
await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
const port = server.address().port;
const base = `http://127.0.0.1:${port}`;

const unstubbed = [];
const consoleErrors = {};
let failures = 0;
const norm = (s) => s.replace(/\s+/g, " ").trim();
const fail = (detail) => {
  throw new Error(detail);
};
async function check(name, fn) {
  try {
    await fn();
    console.log(`PASS ${name}`);
  } catch (error) {
    failures += 1;
    console.log(`FAIL ${name}: ${error.message.split("\n")[0]}`);
  }
}

let browser;
let started = [];
// the browser and its helpers are descendants of this process: they are listed before the browser closes and
// checked at the end, so the run proves it leaves no process behind
const descendants = (pid) => {
  let children = [];
  try {
    children = execFileSync("pgrep", ["-P", String(pid)]).toString().split("\n").filter(Boolean).map(Number);
  } catch {} // pgrep exits 1 when there are none
  return children.flatMap((child) => [child, ...descendants(child)]);
};
const alive = (pid) => {
  try {
    process.kill(pid, 0);
    return true;
  } catch {
    return false;
  }
};
async function openPage(scenario) {
  const context = await browser.newContext({ viewport: { width: 1600, height: 1000 } });
  const page = await context.newPage();
  const requests = [];
  consoleErrors[scenario] ??= [];
  page.on("console", (m) => m.type() === "error" && consoleErrors[scenario].push(m.text().split("\n")[0]));
  page.on("pageerror", (e) => consoleErrors[scenario].push(`pageerror: ${String(e.message).split("\n")[0]}`));
  const handler = (route) => {
    const request = route.request();
    const url = new URL(request.url());
    requests.push({ method: request.method(), path: url.pathname, query: url.searchParams, body: request.postData() });
    let stub = STUBS[`${request.method()} ${url.pathname}`] ?? (request.method() === "GET" ? STUBS[url.pathname] : undefined);
    if (typeof stub === "function") stub = stub(url.searchParams, request);
    if (stub === undefined) {
      unstubbed.push(`[${scenario}] ${request.method()} ${url.pathname}${url.search}`);
      return route.fulfill({ status: 404, json: {} });
    }
    return route.fulfill({ json: stub });
  };
  await page.route("**/api/**", handler);
  await page.route("**/auth/**", handler);
  return { page, requests, close: () => context.close() };
}
const waitForRequest = (page, method, pathname, predicate = () => true) =>
  page.waitForRequest((r) => r.method() === method && new URL(r.url()).pathname === pathname && predicate(r), { timeout: 15000 });
const ERROR_PAGE = "Looks like something went wrong";

try {
  browser = await chromium.launch();

  // ---------------------------------------------------------- 1. activity
  {
    const { page, close } = await openPage("activity");
    await page.goto(`${base}/probe-ws/projects/p1/issues/i1/`);
    let section;
    let entries = [];
    await check("activity: the work item opens from its project path (redirects to /browse/PRB-1/)", async () => {
      await page.waitForURL(/\/probe-ws\/browse\/PRB-1\/?$/, { timeout: 15000 });
      section = page.locator("div.text-h5-medium", { hasText: /^Activity$/ }).locator("xpath=../..");
      await section.getByText("Probe comment on alpha").waitFor({ timeout: 15000 });
      entries = (await section.locator("div.w-full.truncate.text-secondary").allInnerTexts()).map(norm);
    });
    await check("activity: 5 property entries (created, state, relation, cycle, module), each with actor and text", async () => {
      if (entries.length !== 5) fail(`got ${entries.length}: ${JSON.stringify(entries)}`);
      const blank = entries.filter((e) => !/^otto \S.{8,}/.test(e));
      if (blank.length) fail(`entries without text: ${JSON.stringify(blank)}`);
    });
    // each entry is "<actor> <text> <time ago>"; the cycle and module names also link to their pages
    const entry = (name, text, href) =>
      check(`activity: ${name} entry reads "${text}"${href ? ` and links to ${href}` : ""}`, async () => {
        if (!entries.some((e) => e.startsWith(`${text} `))) fail(`not among ${JSON.stringify(entries)}`);
        if (href && !(await section.locator(`a[href="${href}"]`).count())) fail(`no link to ${href}`);
      });
    await entry("state", "otto set the state to In Progress.");
    await entry("relation", "otto marked this work item is blocking work item PRB-2.");
    await entry("cycle", "otto added this work item to the cycle Sprint One", "/probe-ws/projects/p1/cycles/c1");
    await entry("module", "otto added this work item to the module Module One", "/probe-ws/projects/p1/modules/m1");
    await check("activity: comment entry shows its author and text", async () => {
      const text = norm(await section.locator('div[id="cm1"]').first().innerText());
      if (!/^O otto commented .* Probe comment on alpha$/.test(text)) fail(JSON.stringify(text));
    });
    await check("activity: no raw i18n key, undefined, null or NaN in the activity section", async () => {
      const text = norm(await section.innerText());
      const bad = text.match(/\b(undefined|null|NaN)\b|\b[a-z][a-z0-9_]*(\.[a-z0-9_]+)+\b/g);
      if (bad) fail(`${JSON.stringify(bad)} in ${JSON.stringify(text)}`);
    });
    await close();
  }

  // ---------------------------------------------------------- 2. notifications
  {
    const { page, close } = await openPage("notifications");
    const inboxDot = page.locator('a[href="/probe-ws/notifications/"] span.bg-danger-primary');
    const allTab = page.locator("div.cursor-pointer", { hasText: /^All/ }).first();
    const unreadDots = page.locator("div.border-b.border-subtle div.absolute.rounded-full.bg-accent-primary");
    const card = (identifier, title) => page.getByText(new RegExp(`^${identifier}\\s${title}$`));
    await page.goto(`${base}/probe-ws/notifications/`);
    await check("notifications: the page lists both stubbed notifications with readable text", async () => {
      await card("PRB-1", "Project item alpha").waitFor({ timeout: 15000 });
      await card("PRB-2", "Cycle item gamma").waitFor();
      const text = norm(await page.locator("body").innerText());
      for (const s of ["otto updated state to In Progress.", "otto commented Looks good to me."])
        if (!text.includes(s)) fail(`missing "${s}"`);
    });
    await check("notifications: before reading, the inbox shows the unread dot and All shows 2", async () => {
      await inboxDot.waitFor({ timeout: 10000 });
      if (norm(await allTab.innerText()) !== "All 2") fail(`All tab reads ${JSON.stringify(await allTab.innerText())}`);
      if ((await unreadDots.count()) !== 2) fail(`${await unreadDots.count()} unread card dots`);
    });
    const markRead = waitForRequest(page, "POST", `${W}/users/notifications/n1/read/`);
    await card("PRB-1", "Project item alpha").click();
    await check("notifications: opening one sends its read request and All drops to 1", async () => {
      await markRead;
      await page.waitForFunction(() => {
        const tab = [...document.querySelectorAll("div.cursor-pointer")].find((d) => /^All/.test(d.innerText));
        return tab && tab.innerText.replace(/\s+/g, " ").trim() === "All 1";
      }, null, { timeout: 10000 });
    });
    await check("notifications: opening one shows the work item preview (title and description)", async () => {
      await page.waitForFunction(() => document.querySelector("#title-input")?.value === "Project item alpha", null, { timeout: 15000 });
      await page.getByText("Alpha description text").waitFor({ timeout: 15000 });
    });
    await check('notifications: "mark all as read" posts {snoozed:false, archived:false}', async () => {
      const button = page.locator("div.h-header", { hasText: "Inbox" }).locator("button").first();
      await button.hover();
      await page.getByText("Mark all as read", { exact: true }).waitFor({ timeout: 5000 });
      const request = waitForRequest(page, "POST", `${W}/users/notifications/mark-all-read/`);
      await button.click();
      const body = (await request).postDataJSON();
      if (JSON.stringify(body) !== JSON.stringify({ snoozed: false, archived: false })) fail(`body ${JSON.stringify(body)}`);
    });
    await check("notifications: after mark all read, the inbox dot, the All count and the card dots are gone", async () => {
      await inboxDot.waitFor({ state: "detached", timeout: 10000 });
      if (norm(await allTab.innerText()) !== "All") fail(`All tab reads ${JSON.stringify(await allTab.innerText())}`);
      if (await unreadDots.count()) fail(`${await unreadDots.count()} unread card dots left`);
    });
    await close();
  }

  // ---------------------------------------------------------- 3. work-item store per context
  const CONTEXTS = [
    { name: "project", url: "/probe-ws/projects/p1/issues/", titles: [0, 1, 2, 3],
      request: (r) => r.path === `${P}/issues/` && !r.query.get("cycle") && !r.query.get("module") },
    { name: "cycle", url: "/probe-ws/projects/p1/cycles/c1/", titles: [0, 1],
      request: (r) => r.path === `${P}/issues/` && r.query.get("cycle") === "c1" },
    { name: "module", url: "/probe-ws/projects/p1/modules/m1/", titles: [0, 2],
      request: (r) => r.path === `${P}/issues/` && r.query.get("module") === "m1" },
    { name: "profile", url: "/probe-ws/profile/u1/", finalUrl: /\/probe-ws\/profile\/u1\/assigned\/?$/, titles: [3],
      request: (r) => r.path === `${W}/user-issues/u1/` && r.query.get("assignees") === "u1" },
    { name: "archives", url: "/probe-ws/projects/p1/archives/issues/", titles: [4],
      request: (r) => r.path === `${P}/archived-issues/` },
  ];
  for (const ctx of CONTEXTS) {
    const { page, requests, close } = await openPage(`store-${ctx.name}`);
    const expected = ctx.titles.map((i) => TITLES[i]);
    await check(`store: ${ctx.name} page lists exactly ${JSON.stringify(expected)}`, async () => {
      await page.goto(`${base}${ctx.url}`);
      if (ctx.finalUrl) await page.waitForURL(ctx.finalUrl, { timeout: 15000 });
      await page.getByText(expected[0], { exact: true }).first().waitFor({ timeout: 15000 });
      await page.waitForTimeout(1000);
      const shown = [];
      for (const title of TITLES)
        if (await page.getByText(title, { exact: true }).filter({ visible: true }).count()) shown.push(title);
      if (JSON.stringify(shown) !== JSON.stringify(expected)) fail(`shown ${JSON.stringify(shown)}`);
      const listCalls = requests.filter((r) => /\/(issues|archived-issues|user-issues\/u1)\/$/.test(r.path));
      if (!listCalls.some(ctx.request))
        fail(`no matching list request; list requests: ${JSON.stringify(listCalls.map((r) => `${r.path}?${r.query}`))}`);
    });
    await close();
  }

  // ---------------------------------------------------------- 4. layouts
  {
    const { page, close } = await openPage("layouts");
    await page.goto(`${base}/probe-ws/projects/p1/issues/`);
    await page.getByText(TITLES[0], { exact: true }).first().waitFor({ timeout: 15000 });
    const layoutButtons = page.locator('button[aria-label$=" Layout"]');
    await check("layouts: the view offers exactly List, Board, Table and Calendar (no Gantt/Timeline)", async () => {
      const labels = (await layoutButtons.evaluateAll((els) => els.map((e) => e.getAttribute("aria-label")))).sort();
      const want = ["Board Layout", "Calendar Layout", "List Layout", "Table Layout"];
      if (JSON.stringify(labels) !== JSON.stringify(want)) fail(`labels ${JSON.stringify(labels)}`);
      const html = await page.content();
      if (/gantt|timeline/i.test(html)) fail(`"gantt" or "timeline" in the page: ${html.match(/.{40}(gantt|timeline).{40}/i)?.[0]}`);
    });
    const LAYOUTS = [
      ["Board Layout", "kanban", '[id="s-progress__null"] [id="issue-i1"]', '[id="s-todo__null"] [id="issue-i2"]'],
      ["Table Layout", "spreadsheet", 'table td[id="issue-i1"]', 'table td[id="issue-i4"]'],
      ["Calendar Layout", "calendar", 'a[id="issue-i1"]', 'div.grid > div:text-is("Mon")'],
      ["List Layout", "list", 'a[id="issue-i1"]', 'a[id="issue-i4"]'],
    ];
    for (const [label, key, ...markers] of LAYOUTS) {
      await check(`layouts: switching to ${label} saves layout=${key} and renders it without an error`, async () => {
        const errorsBefore = consoleErrors.layouts.filter((e) => e.startsWith("pageerror")).length;
        const saved = waitForRequest(page, "PATCH", `${P}/user-properties/`, (r) => r.postDataJSON()?.display_filters?.layout === key);
        await page.locator(`button[aria-label="${label}"]`).click();
        await saved;
        for (const marker of markers) await page.locator(marker).first().waitFor({ timeout: 15000 });
        if (key !== "spreadsheet" && (await page.locator("table td[id^='issue-']").count())) fail("table rows still shown");
        if ((await page.locator("body").innerText()).includes(ERROR_PAGE)) fail("error page shown");
        const errorsAfter = consoleErrors.layouts.filter((e) => e.startsWith("pageerror"));
        if (errorsAfter.length > errorsBefore) fail(`page error: ${errorsAfter.at(-1)}`);
      });
    }
    await close();
  }
} finally {
  started = descendants(process.pid);
  await browser?.close();
  await new Promise((resolve) => server.close(resolve));
}

console.log(`\nunstubbed requests (${unstubbed.length}):`);
for (const r of [...new Set(unstubbed)]) console.log(`  ${r}`);
console.log("\nbrowser console errors per scenario:");
for (const [scenario, errors] of Object.entries(consoleErrors)) {
  const expected = unstubbed.length ? errors.filter((e) => /404|Failed to load resource/.test(e)) : [];
  const other = errors.filter((e) => !expected.includes(e));
  console.log(`  ${scenario}: ${errors.length} (from unstubbed requests: ${expected.length}, other: ${other.length})`);
  for (const e of [...new Set(other)]) console.log(`    other: ${e.slice(0, 200)}`);
}
const probe = net.createServer();
await new Promise((resolve, reject) => probe.once("error", reject).listen(port, "127.0.0.1", resolve));
await new Promise((resolve) => probe.close(resolve));
console.log(`port ${port} is free`);
await new Promise((resolve) => setTimeout(resolve, 500));
const left = started.filter(alive);
console.log(`browser processes started by this run: ${started.length}; still running: ${left.length ? left.join(", ") : "none"}`);
if (left.length) failures += 1;
console.log(`\n${failures ? `${failures} check(s) FAILED` : "all checks passed"}`);
process.exit(failures ? 1 : 0);
```

#### 反向对照：只改假数据的生成脚本（`make-negative.py`）

```python
# Builds negative-control.mjs: the probe with three deliberate breakages in the FAKE DATA (never the app),
# to show the checks are not vacuous. Expected: exactly the checks named below fail.
import pathlib

here = pathlib.Path(__file__).parent
s = (here / "activity-notifications-lists-probe.mjs").read_text()
mutations = [
    # N1 activity: a relation activity with a field the app does not know -> the relation entry disappears
    ('activity("a2", 40, { field: "blocking",', 'activity("a2", 40, { field: "blocks",'),
    # N2 store: the list endpoint ignores the cycle parameter -> the cycle page would show every project item
    ('(!q.get("cycle") || i.cycle_id === q.get("cycle")) &&', "true &&"),
    # N3 notifications: mark-all-read is not answered (404) -> the app must keep the unread state
    ("[`POST ${W}/users/notifications/mark-all-read/`]: {},", ""),
]
for old, new in mutations:
    assert s.count(old) == 1, old
    s = s.replace(old, new)
(here / "negative-control.mjs").write_text(s)
print("wrote negative-control.mjs")
```

#### 反向对照的输出

```
PASS activity: the work item opens from its project path (redirects to /browse/PRB-1/)
FAIL activity: 5 property entries (created, state, relation, cycle, module), each with actor and text: got 4: ["otto created the work item. about 1 hour ago","otto set the state to In Progress. about 1 hour ago","otto added this work item to the cycle Sprint One 30 minutes ago","otto added this work item to the module Module One 20 minutes ago"]
PASS activity: state entry reads "otto set the state to In Progress."
FAIL activity: relation entry reads "otto marked this work item is blocking work item PRB-2.": not among ["otto created the work item. about 1 hour ago","otto set the state to In Progress. about 1 hour ago","otto added this work item to the cycle Sprint One 30 minutes ago","otto added this work item to the module Module One 20 minutes ago"]
PASS activity: cycle entry reads "otto added this work item to the cycle Sprint One" and links to /probe-ws/projects/p1/cycles/c1
PASS activity: module entry reads "otto added this work item to the module Module One" and links to /probe-ws/projects/p1/modules/m1
PASS activity: comment entry shows its author and text
PASS activity: no raw i18n key, undefined, null or NaN in the activity section
PASS notifications: the page lists both stubbed notifications with readable text
PASS notifications: before reading, the inbox shows the unread dot and All shows 2
PASS notifications: opening one sends its read request and All drops to 1
PASS notifications: opening one shows the work item preview (title and description)
PASS notifications: "mark all as read" posts {snoozed:false, archived:false}
FAIL notifications: after mark all read, the inbox dot, the All count and the card dots are gone: locator.waitFor: Timeout 10000ms exceeded.
PASS store: project page lists exactly ["Project item alpha","Cycle item gamma","Module item delta","Assigned item epsilon"]
FAIL store: cycle page lists exactly ["Project item alpha","Cycle item gamma"]: shown ["Project item alpha","Cycle item gamma","Module item delta","Assigned item epsilon"]
PASS store: module page lists exactly ["Project item alpha","Module item delta"]
PASS store: profile page lists exactly ["Assigned item epsilon"]
PASS store: archives page lists exactly ["Archived item zeta"]
PASS layouts: the view offers exactly List, Board, Table and Calendar (no Gantt/Timeline)
PASS layouts: switching to Board Layout saves layout=kanban and renders it without an error
PASS layouts: switching to Table Layout saves layout=spreadsheet and renders it without an error
PASS layouts: switching to Calendar Layout saves layout=calendar and renders it without an error
PASS layouts: switching to List Layout saves layout=list and renders it without an error

unstubbed requests (1):
  [notifications] POST /api/workspaces/probe-ws/users/notifications/mark-all-read/

browser console errors per scenario:
  activity: 0 (from unstubbed requests: 0, other: 0)
  notifications: 3 (from unstubbed requests: 3, other: 0)
  store-project: 0 (from unstubbed requests: 0, other: 0)
  store-cycle: 0 (from unstubbed requests: 0, other: 0)
  store-module: 0 (from unstubbed requests: 0, other: 0)
  store-profile: 0 (from unstubbed requests: 0, other: 0)
  store-archives: 0 (from unstubbed requests: 0, other: 0)
  layouts: 0 (from unstubbed requests: 0, other: 0)
port 63051 is free
browser processes started by this run: 3; still running: none

4 check(s) FAILED
```

#### 在 `36172ca` 构建上的最终输出

```
PASS activity: the work item opens from its project path (redirects to /browse/PRB-1/)
PASS activity: 5 property entries (created, state, relation, cycle, module), each with actor and text
PASS activity: state entry reads "otto set the state to In Progress."
PASS activity: relation entry reads "otto marked this work item is blocking work item PRB-2."
PASS activity: cycle entry reads "otto added this work item to the cycle Sprint One" and links to /probe-ws/projects/p1/cycles/c1
PASS activity: module entry reads "otto added this work item to the module Module One" and links to /probe-ws/projects/p1/modules/m1
PASS activity: comment entry shows its author and text
PASS activity: no raw i18n key, undefined, null or NaN in the activity section
PASS notifications: the page lists both stubbed notifications with readable text
PASS notifications: before reading, the inbox shows the unread dot and All shows 2
PASS notifications: opening one sends its read request and All drops to 1
PASS notifications: opening one shows the work item preview (title and description)
PASS notifications: "mark all as read" posts {snoozed:false, archived:false}
PASS notifications: after mark all read, the inbox dot, the All count and the card dots are gone
PASS store: project page lists exactly ["Project item alpha","Cycle item gamma","Module item delta","Assigned item epsilon"]
PASS store: cycle page lists exactly ["Project item alpha","Cycle item gamma"]
PASS store: module page lists exactly ["Project item alpha","Module item delta"]
PASS store: profile page lists exactly ["Assigned item epsilon"]
PASS store: archives page lists exactly ["Archived item zeta"]
PASS layouts: the view offers exactly List, Board, Table and Calendar (no Gantt/Timeline)
PASS layouts: switching to Board Layout saves layout=kanban and renders it without an error
PASS layouts: switching to Table Layout saves layout=spreadsheet and renders it without an error
PASS layouts: switching to Calendar Layout saves layout=calendar and renders it without an error
PASS layouts: switching to List Layout saves layout=list and renders it without an error

unstubbed requests (0):

browser console errors per scenario:
  activity: 0 (from unstubbed requests: 0, other: 0)
  notifications: 0 (from unstubbed requests: 0, other: 0)
  store-project: 0 (from unstubbed requests: 0, other: 0)
  store-cycle: 0 (from unstubbed requests: 0, other: 0)
  store-module: 0 (from unstubbed requests: 0, other: 0)
  store-profile: 0 (from unstubbed requests: 0, other: 0)
  store-archives: 0 (from unstubbed requests: 0, other: 0)
  layouts: 0 (from unstubbed requests: 0, other: 0)
port 62957 is free
browser processes started by this run: 3; still running: none

all checks passed
```

#### 修复轮之后在 `ef3be19` 的构建上重跑

```
PASS activity: the work item opens from its project path (redirects to /browse/PRB-1/)
PASS activity: 5 property entries (created, state, relation, cycle, module), each with actor and text
PASS activity: state entry reads "otto set the state to In Progress."
PASS activity: relation entry reads "otto marked this work item is blocking work item PRB-2."
PASS activity: cycle entry reads "otto added this work item to the cycle Sprint One" and links to /probe-ws/projects/p1/cycles/c1
PASS activity: module entry reads "otto added this work item to the module Module One" and links to /probe-ws/projects/p1/modules/m1
PASS activity: comment entry shows its author and text
PASS activity: no raw i18n key, undefined, null or NaN in the activity section
PASS notifications: the page lists both stubbed notifications with readable text
PASS notifications: before reading, the inbox shows the unread dot and All shows 2
PASS notifications: opening one sends its read request and All drops to 1
PASS notifications: opening one shows the work item preview (title and description)
PASS notifications: "mark all as read" posts {snoozed:false, archived:false}
PASS notifications: after mark all read, the inbox dot, the All count and the card dots are gone
PASS store: project page lists exactly ["Project item alpha","Cycle item gamma","Module item delta","Assigned item epsilon"]
PASS store: cycle page lists exactly ["Project item alpha","Cycle item gamma"]
PASS store: module page lists exactly ["Project item alpha","Module item delta"]
PASS store: profile page lists exactly ["Assigned item epsilon"]
PASS store: archives page lists exactly ["Archived item zeta"]
PASS layouts: the view offers exactly List, Board, Table and Calendar (no Gantt/Timeline)
PASS layouts: switching to Board Layout saves layout=kanban and renders it without an error
PASS layouts: switching to Table Layout saves layout=spreadsheet and renders it without an error
PASS layouts: switching to Calendar Layout saves layout=calendar and renders it without an error
PASS layouts: switching to List Layout saves layout=list and renders it without an error

unstubbed requests (0):

browser console errors per scenario:
  activity: 0 (from unstubbed requests: 0, other: 0)
  notifications: 0 (from unstubbed requests: 0, other: 0)
  store-project: 0 (from unstubbed requests: 0, other: 0)
  store-cycle: 0 (from unstubbed requests: 0, other: 0)
  store-module: 0 (from unstubbed requests: 0, other: 0)
  store-profile: 0 (from unstubbed requests: 0, other: 0)
  store-archives: 0 (from unstubbed requests: 0, other: 0)
  layouts: 0 (from unstubbed requests: 0, other: 0)
port 52033 is free
browser processes started by this run: 3; still running: none

all checks passed
```

### 6.5 编辑器测试：人为改坏与 `mainFields` 的必要性（Task 12 报告摘录）

#### Mutation evidence (`$P2TMP/t12-run-mutate.py`, output `$P2TMP/t12-run-mutate.txt`)

- **Method.** Each mutation edits one product file in memory, runs only its target test (`vitest run <file> -t <name>`),
  and writes the file back byte for byte (asserted).
- **Restore check.** Afterwards `git diff --stat -- web/packages/editor/src` was empty.
- **Baseline:** 7/7 and 7/7.

| # | Mutation (file) | Result |
|---|---|---|
| C1 | add `UniqueID.extend({ name: "collaboration" })` to the list (extensions.ts) | `1 failed \| 6 skipped` |
| C2 | starter kit always `history: false` (starter-kit.ts) | `1 failed \| 6 skipped` |
| C3 | drop `...(enableHistory ? {} : { history: false })` (starter-kit.ts) | `1 failed \| 6 skipped` |
| C4 | remove `Table` from the list (extensions.ts) | `1 failed \| 6 skipped` |
| C5 | image condition → `if (true)` (extensions.ts) | `1 failed \| 6 skipped` |
| C6 | add a `document:` set to `TOOLBAR_ITEMS` (constants/common.ts) | `1 failed \| 6 skipped` |
| C7 | put a `table` item back into `COMPLEX_ITEMS` (constants/common.ts) | `1 failed \| 6 skipped` |
| I1a | `getMarkDown` returns the HTML (helpers/editor-ref.ts) | `1 failed \| 6 skipped` |
| I1b | `heading: false` in the starter kit (starter-kit.ts) | `1 failed \| 6 skipped` |
| I2 | starter kit always `history: false` (starter-kit.ts) | `1 failed \| 6 skipped` |
| I3 | UniqueID `onCreate` ignores `isEditable` (unique-id/extension.ts) | `1 failed \| 6 skipped` |
| I4 | mention `parseHTML` tag → `user-mention` (mentions/extension-config.ts) | `1 failed \| 6 skipped` |
| I5 | image `parseHTML` tag → `image-block` (custom-image/extension-config.ts) | `1 failed \| 6 skipped` |
| I6 | unique-id plugin never assigns ids (unique-id/plugin.ts) | `1 failed \| 6 skipped` |
| I7 | **the one-line fix reverted** (unique-id/extension.ts) | `1 failed \| 6 skipped` |

**I7 in detail** (controller ruling 2).

- Run alone with the fix reverted, the regression test fails at `editor-interaction.test.ts:267`, the third editor's
  initial ids: `AssertionError: expected false to be true`.
- In a full-file run with the fix reverted, **two** tests fail: I6 at line 241 and I7 at line 261.
  - I6 fails because the earlier read-only test in the same file has already turned ids off for the whole process.
    That is the bug itself, showing up across tests.
  - I7 already fails on its first editor at line 261, for the same reason.
- Outputs: `$P2TMP/t12-run-revert-fix.txt` and `$P2TMP/t12-run-revert-fix-isolated.txt`.


#### Proof that `mainFields` is needed (`$P2TMP/t12-run-mainfields.py`, output `$P2TMP/t12-run-mainfields.txt`)

**Method.** The line is removed in memory, the interaction tests run, and the config is written back byte for byte
(asserted) before the second run.

**Result without the line.** The first `new Editor` throws the `RangeError`. Later editors construct, but CJS and ESM
ProseMirror classes mix: `DecorationGroup.locals` / `.eq` read `undefined` (5 unhandled errors), and Enter/typing
break. 3 tests fail, the file fails, and vitest exits non-zero.

Without the line (`<repo>` = the worktree; sourcemap lines removed here, they are listed in §3 below):

```
=== without the mainFields line ===

 RUN  v4.1.11 <repo>/web/packages/editor

 ❯ src/editor-interaction.test.ts (7 tests | 3 failed) 148ms
     × takes typed text and gives it back as HTML and as Markdown 17ms
     × shows a description read-only, and edits it once it is editable again 22ms
     × gives every block node an id, keeps it through a save and gives new blocks new ones 11ms

 Test Files  1 failed (1)
      Tests  3 failed | 4 passed (7)
     Errors  5 errors
   Start at  15:49:23
   Duration  1.55s (transform 320ms, setup 9ms, import 1.14s, tests 148ms, environment 197ms)


⎯⎯⎯⎯⎯⎯⎯ Failed Tests 3 ⎯⎯⎯⎯⎯⎯⎯

 FAIL  src/editor-interaction.test.ts > a rich-text editor built like the kept ones > takes typed text and gives it back as HTML and as Markdown
RangeError: Adding different instances of a keyed plugin (plugin$)
 ❯ ../../../node_modules/.pnpm/prosemirror-state@1.4.3/node_modules/prosemirror-state/dist/index.js:729:27
 ❯ new Configuration ../../../node_modules/.pnpm/prosemirror-state@1.4.3/node_modules/prosemirror-state/dist/index.js:727:21
 ❯ EditorState.reconfigure ../../../node_modules/.pnpm/prosemirror-state@1.4.3/node_modules/prosemirror-state/dist/index.js:864:23
 ❯ Editor.createView ../../../node_modules/.pnpm/@tiptap+core@2.26.3_@tiptap+pm@2.26.1/node_modules/@tiptap/core/src/Editor.ts:379:32
 ❯ new Editor ../../../node_modules/.pnpm/@tiptap+core@2.26.3_@tiptap+pm@2.26.1/node_modules/@tiptap/core/src/Editor.ts:107:9
 ❯ createEditor src/editor-interaction.test.ts:60:18
     58|   const element = document.createElement("div");
     59|   document.body.append(element);
     60|   const editor = new Editor({
       |                  ^
     61|     element,
     62|     editable,
 ❯ src/editor-interaction.test.ts:127:26

⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯[1/3]⎯

 FAIL  src/editor-interaction.test.ts > a rich-text editor built like the kept ones > shows a description read-only, and edits it once it is editable again
AssertionError: expected false to be true // Object.is equality

- Expected
+ Received

- true
+ false

 ❯ src/editor-interaction.test.ts:179:38
    177|     editable.commands.focus("end");
    178|     expect(editable.view.dom.getAttribute("contenteditable")).toBe("tr…
    179|     expect(press(editable, "Enter")).toBe(true);
       |                                      ^
    180|     typeText(editable, "check the result");
    181|     expect(parse(editable.getHTML()).querySelectorAll("li")).toHaveLen…

⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯[2/3]⎯

 FAIL  src/editor-interaction.test.ts > a rich-text editor built like the kept ones > gives every block node an id, keeps it through a save and gives new blocks new ones
AssertionError: expected [ 'listItem', 'paragraph' ] to deeply equal [ 'paragraph' ]

- Expected
+ Received

  [
+   "listItem",
    "paragraph",
  ]

 ❯ src/editor-interaction.test.ts:250:43
    248|     const after = blockIds(editor);
    249|     const added = after.filter(({ id }) => !initial.some((block) => bl…
    250|     expect(added.map(({ type }) => type)).toEqual(["paragraph"]);
       |                                           ^
    251|     expect(added[0].id).toEqual(expect.any(String));
    252|

⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯[3/3]⎯

⎯⎯⎯⎯⎯⎯ Unhandled Errors ⎯⎯⎯⎯⎯⎯

Vitest caught 5 unhandled errors during the test run.
This might cause false positive tests. Resolve unhandled errors to make sure your tests are not affected.

⎯⎯⎯⎯⎯ Uncaught Exception ⎯⎯⎯⎯⎯
TypeError: Cannot read properties of undefined (reading 'localsInner')
 ❯ DecorationGroup.locals ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:4281:42
 ❯ iterDeco ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:2051:23
 ❯ NodeViewDesc.updateChildren ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:1374:9
 ❯ NodeViewDesc.updateInner ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:1470:18
 ❯ NodeViewDesc.update ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:1462:14
 ❯ EditorView.updateStateInner ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:5425:45
 ❯ EditorView.updateState ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:5376:14
 ❯ Editor.dispatchTransaction ../../../node_modules/.pnpm/@tiptap+core@2.26.3_@tiptap+pm@2.26.1/node_modules/@tiptap/core/dist/index.js:4727:19
 ❯ EditorView.dispatch ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:5751:29
 ❯ Object.method [as splitListItem] ../../../node_modules/.pnpm/@tiptap+core@2.26.3_@tiptap+pm@2.26.1/node_modules/@tiptap/core/dist/index.js:66:26

This error originated in "src/editor-interaction.test.ts" test file. It doesn't mean the error was thrown inside the file itself, but while it was running.
The latest test that might've caused the error is "shows a description read-only, and edits it once it is editable again". It might mean one of the following:
- The error was thrown, while Vitest was running this test.
- If the error occurred after the test had been completed, this was the last documented test before it was thrown.

⎯⎯⎯⎯⎯ Uncaught Exception ⎯⎯⎯⎯⎯
TypeError: Cannot read properties of undefined (reading 'localsInner')
 ❯ DecorationGroup.locals ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:4281:42
 ❯ iterDeco ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:2051:23
 ❯ NodeViewDesc.updateChildren ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:1374:9
 ❯ NodeViewDesc.updateInner ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:1470:18
 ❯ NodeViewDesc.update ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:1462:14
 ❯ EditorView.updateStateInner ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:5425:45
 ❯ EditorView.updateState ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:5376:14
 ❯ Editor.dispatchTransaction ../../../node_modules/.pnpm/@tiptap+core@2.26.3_@tiptap+pm@2.26.1/node_modules/@tiptap/core/dist/index.js:4727:19
 ❯ EditorView.dispatch ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:5751:29
 ❯ Object.method [as splitListItem] ../../../node_modules/.pnpm/@tiptap+core@2.26.3_@tiptap+pm@2.26.1/node_modules/@tiptap/core/dist/index.js:66:26

This error originated in "src/editor-interaction.test.ts" test file. It doesn't mean the error was thrown inside the file itself, but while it was running.
The latest test that might've caused the error is "gives every block node an id, keeps it through a save and gives new blocks new ones". It might mean one of the following:
- The error was thrown, while Vitest was running this test.
- If the error occurred after the test had been completed, this was the last documented test before it was thrown.

⎯⎯⎯⎯⎯ Uncaught Exception ⎯⎯⎯⎯⎯
TypeError: Cannot read properties of undefined (reading 'eq')
 ❯ DecorationGroup.eq ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:4274:34
 ❯ NodeViewDesc.matchesNode ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:1360:67
 ❯ EditorView.updateStateInner ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:5404:49
 ❯ EditorView.updateState ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:5376:14
 ❯ Editor.dispatchTransaction ../../../node_modules/.pnpm/@tiptap+core@2.26.3_@tiptap+pm@2.26.1/node_modules/@tiptap/core/dist/index.js:4727:19
 ❯ EditorView.dispatch ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:5751:29
 ❯ Object.method [as splitListItem] ../../../node_modules/.pnpm/@tiptap+core@2.26.3_@tiptap+pm@2.26.1/node_modules/@tiptap/core/dist/index.js:66:26
 ❯ Enter ../../../node_modules/.pnpm/@tiptap+extension-task-item@2.26.1_@tiptap+core@2.26.3_@tiptap+pm@2.26.1__@tiptap+pm@2.26.1/node_modules/@tiptap/extension-task-item/dist/index.js:70:47
 ❯ ../../../node_modules/.pnpm/@tiptap+core@2.26.3_@tiptap+pm@2.26.1/node_modules/@tiptap/core/dist/index.js:1285:45
 ❯ Plugin.<anonymous> ../../../node_modules/.pnpm/prosemirror-keymap@1.2.3/node_modules/prosemirror-keymap/dist/index.js:100:23

This error originated in "src/editor-interaction.test.ts" test file. It doesn't mean the error was thrown inside the file itself, but while it was running.
The latest test that might've caused the error is "gives every block node an id, keeps it through a save and gives new blocks new ones". It might mean one of the following:
- The error was thrown, while Vitest was running this test.
- If the error occurred after the test had been completed, this was the last documented test before it was thrown.

⎯⎯⎯⎯⎯ Uncaught Exception ⎯⎯⎯⎯⎯
TypeError: Cannot read properties of undefined (reading 'localsInner')
 ❯ DecorationGroup.locals ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:4281:42
 ❯ iterDeco ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:2051:23
 ❯ NodeViewDesc.updateChildren ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:1374:9
 ❯ NodeViewDesc.updateInner ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:1470:18
 ❯ NodeViewDesc.update ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:1462:14
 ❯ EditorView.updateStateInner ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:5425:45
 ❯ EditorView.updateState ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:5376:14
 ❯ Editor.dispatchTransaction ../../../node_modules/.pnpm/@tiptap+core@2.26.3_@tiptap+pm@2.26.1/node_modules/@tiptap/core/dist/index.js:4727:19
 ❯ EditorView.dispatch ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:5751:29
 ❯ Object.method [as splitListItem] ../../../node_modules/.pnpm/@tiptap+core@2.26.3_@tiptap+pm@2.26.1/node_modules/@tiptap/core/dist/index.js:66:26

This error originated in "src/editor-interaction.test.ts" test file. It doesn't mean the error was thrown inside the file itself, but while it was running.
The latest test that might've caused the error is "creates editors one after another in one process: editable, read-only, editable". It might mean one of the following:
- The error was thrown, while Vitest was running this test.
- If the error occurred after the test had been completed, this was the last documented test before it was thrown.

⎯⎯⎯⎯⎯ Uncaught Exception ⎯⎯⎯⎯⎯
TypeError: Cannot read properties of undefined (reading 'eq')
 ❯ DecorationGroup.eq ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:4274:34
 ❯ NodeViewDesc.matchesNode ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:1360:67
 ❯ EditorView.updateStateInner ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:5404:49
 ❯ EditorView.updateState ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:5376:14
 ❯ Editor.dispatchTransaction ../../../node_modules/.pnpm/@tiptap+core@2.26.3_@tiptap+pm@2.26.1/node_modules/@tiptap/core/dist/index.js:4727:19
 ❯ EditorView.dispatch ../../../node_modules/.pnpm/prosemirror-view@1.40.0/node_modules/prosemirror-view/dist/index.js:5751:29
 ❯ Object.method [as splitListItem] ../../../node_modules/.pnpm/@tiptap+core@2.26.3_@tiptap+pm@2.26.1/node_modules/@tiptap/core/dist/index.js:66:26
 ❯ Enter ../../../node_modules/.pnpm/@tiptap+extension-task-item@2.26.1_@tiptap+core@2.26.3_@tiptap+pm@2.26.1__@tiptap+pm@2.26.1/node_modules/@tiptap/extension-task-item/dist/index.js:70:47
 ❯ ../../../node_modules/.pnpm/@tiptap+core@2.26.3_@tiptap+pm@2.26.1/node_modules/@tiptap/core/dist/index.js:1285:45
 ❯ Plugin.<anonymous> ../../../node_modules/.pnpm/prosemirror-keymap@1.2.3/node_modules/prosemirror-keymap/dist/index.js:100:23

This error originated in "src/editor-interaction.test.ts" test file. It doesn't mean the error was thrown inside the file itself, but while it was running.
The latest test that might've caused the error is "creates editors one after another in one process: editable, read-only, editable". It might mean one of the following:
- The error was thrown, while Vitest was running this test.
- If the error occurred after the test had been completed, this was the last documented test before it was thrown.
⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯⎯
```

With the line restored:

```
=== with the mainFields line ===

 RUN  v4.1.11 <repo>/web/packages/editor


 Test Files  1 passed (1)
      Tests  7 passed (7)
   Start at  15:49:25
   Duration  2.94s (transform 1.50s, setup 8ms, import 2.50s, tests 186ms, environment 196ms)
```


#### The sourcemap warnings (controller ruling 3: record, no logger, no filter)

These are the exact lines on every run (`<repo>` = the worktree):

```
Sourcemap for "<repo>/node_modules/.pnpm/prosemirror-codemark@0.4.2_prosemirror-inputrules@1.5.0_prosemirror-model@1.25.3_prosem_b033beb31662c628da53ea9ebf5dd9a5/node_modules/prosemirror-codemark/dist/esm/index.js" points to missing source files
Sourcemap for "<repo>/node_modules/.pnpm/prosemirror-codemark@0.4.2_…/node_modules/prosemirror-codemark/dist/esm/plugin.js" points to missing source files
Sourcemap for "<repo>/node_modules/.pnpm/prosemirror-codemark@0.4.2_…/node_modules/prosemirror-codemark/dist/esm/utils.js" points to missing source files
Sourcemap for "<repo>/node_modules/.pnpm/prosemirror-codemark@0.4.2_…/node_modules/prosemirror-codemark/dist/esm/inputRules.js" points to missing source files
Sourcemap for "<repo>/node_modules/.pnpm/prosemirror-codemark@0.4.2_…/node_modules/prosemirror-codemark/dist/esm/actions.js" points to missing source files
Sourcemap for "<repo>/node_modules/.pnpm/is-emoji-supported@0.0.5/node_modules/is-emoji-supported/dist/esm/is-emoji-supported.js" points to missing source files
```

**Why they appear.**
- Vite 8.0.16's `loadAndTransform` reads the sourcemap of any file it loads from disk, and `injectSourcesContent` warns
  whenever a map's `sources` are not on disk. That covers `../../src/*.ts`, which these packages do not ship. No option
  gates it; `server.sourcemapIgnoreList` runs after the warning.
- vitest must inline these two ESM builds rather than hand them to Node. They have ESM syntax without
  `"type": "module"`, and prosemirror-codemark's ESM also uses extensionless relative imports.
- So these are third-party noise, left as they are, per the ruling.

Apart from those six lines, the test output is clean: no jsdom "Not implemented", no unhandled errors.

#### Does the package still build and type-check with tests present?

**Type-check.** Test files live in `src/` and are type-checked by `tsc` (the `include` is `src/**/*`), like `utils` and
`constants`. `check:types` passes.

**Build.**
- `build` is `tsc && tsdown`, and the base config has `noEmit: true`, so `tsc` only type-checks.
- tsdown bundles from `src/index.ts` only; tests are unreachable from it.
- A direct `pnpm --dir web/packages/editor run build` passes. `dist/` holds `index.js`, `index.d.ts`, their maps and the
  copied `styles/`, and has no test or vitest file (checked with `find` and `grep`).

**Out of scope for oxlint and tsc.** `vitest.config.ts` matches oxlint's `*.config.*` ignore. `vitest.setup.ts` is
outside the tsc include, and oxlint does lint it.

---

## 7. 交接与延后项

交给后续 M 的事项写进对应 M 的 `handoffs/M1-P2-trim-content.md`：

| M | 事项 |
|---|---|
| M2 | `IUserTheme` 只剩 `{ theme }`，新接口的用户主题字段不要再有 `primary`、`background`、`darkPalette`。保存为 `"custom"` 的旧资料在前端显示亮色调色板，主题选择框显示占位符 |
| M3 | 让后端与前端保持一致：<br>- `IProject` 不再读 `close_in`、`default_state`、`page_view`、`estimate_id`；<br>- 不再需要 `/sidebar-preferences/` 接口；<br>- `RESTRICTED_URLS` 已去掉 `import`、`importers`、`integrations`、`integration`、`pages`、`live`，后端的保留名单与之一致；<br>- 访客能否查看个人主页，由 M3 的权限矩阵决定 |
| M4 | - 工作项不再读 `estimate_point`；<br>- 不再调用 `POST /issue-dates/`；<br>- 搜索结果不再有 `page` 类型；<br>- `description_binary` 不再读取；<br>- 只补 id 的迁移更新带 `skip_activity: "true"`，后端要照此不记动态。<br>**继承自 Plane 的描述保存问题**（`web/apps/web/core/components/editor/rich-text/description-input/root.tsx`，自导入以来未改）：1.5 秒的保存延迟内先撤销再重做，发出的保存会带着撤销后的内容，而编辑器里显示的是重做后的内容。原因是重做时 HTML 等于上次保存的内容，于是提前返回，待发的保存仍是撤销后的版本。M4 重写描述保存时一并修复（附录 6.3 第 7 节） |
| M5 | 资源类型 `PAGE_DESCRIPTION` 和 `file_assets.page_id` 不再使用 |
| M6 | - 迭代和模块的进度不再带 `points`、`estimate_distribution`、各类 `*_estimate_points`，快照里也不再有点数分布；<br>- 守卫里 `cycle.service.ts` 的 `analytics` 地址例外（`until: M6`），在进度接口被替换时随之消失 |
| M7 | - 收藏和最近访问不再有 `page` 类型；<br>- 通知实体里已删功能的类型不再出现；<br>- 首页的最近访问接口由 M7 替换 |

交给 M1 后续 Phase 和收尾：

| 去向 | 事项 |
|---|---|
| P3 | **守卫**：<br>- 11 条 `until: M1/P3` 的守卫例外，在删计费文案（`plans.tsx`、`subscription.ts`）、企业版类型（`epics.ts`）、公开发布类型（`publish.ts`）和通知卡片里的估算字段时随之删除；<br>- `integration`、`importer`、`jira`、`slack`、`github-repository`、`app-installation`、`indexeddb` 的守卫规则，连同 `jira.svg`、12 个集成和导入文案、`silo`（第 4 节裁定 10）。<br>**企业版空壳**：`useEditorFlagging` 在 `"ai"` 删除后已无作用。<br>**knip**：仍报告 85 个未使用文件、108 个导出、61 个类型、4 个枚举成员，逐个复核后删除，然后 knip 改为门禁。<br>**基点就已死的代码**（各 Task 报告逐条列出）：`base-layouts/`、`ISSUE_DISPLAY_FILTERS_BY_LAYOUT`、`generateDateArray`、`TCycleProgress`、`UserActivityIcon`、`IWorkspaceProgressResponse`、`SidebarItemBase.additionalRender`、`auto-remind.*` 文案、`findTotalDaysInRange` 的外部导出、`NodeHighlightPlugin`（从未注册）、侧边栏的 `user-menu.tsx`、`user-menu-item.tsx` 和 `notification-app-sidebar-option.tsx`、空的 FEATURES 设置分组、`editor-container.tsx` 的单子元素 fragment、`RESTRICTED_URLS` 中重复的 `config` 和 `mobile`。<br>**需要产品判断的**：<br>- 收藏图标用 `type: string` 加兜底 `PagesOutline`，根本做法是把 `type` 定为收藏实体的联合类型，并去掉兜底；<br>- 表格菜单的"适应宽度"对保留的编辑器无效。<br>**编辑器测试输出里的继承噪声**：`custom-link` 扩展默认 `protocols: ["http", "https"]`，而 linkify 本身就支持这两个协议。于是第一个编辑器之后，每个编辑器都会打印 `linkifyjs: already initialized - will not register custom scheme …`，浏览器控制台里也一样；根本做法是去掉这两个多余的默认协议 |
| P4 | - `routes/core.ts` 末尾剩下的 10 条旧地址重定向；<br>- `app/layout.tsx` 补尾部斜杠的重定向用的是 `request.url`，不带 `#` 片段，手写的、不带尾部斜杠的 `#块id` 链接会丢掉定位（应用自己生成的链接都以 `/` 结尾，不受影响） |
| M1 收尾 | **死资源**：P1 结束时就已无引用的键和图片，减去本 Phase 按裁定 3 删除的部分；另有 5 个 `themes.theme_options.*.label` 键（整分支评审 M6），决定删除它们，或者让 `THEME_OPTIONS` 用上它们。<br>**方法**：清理死键时用"完整字符串字面量 + 模板前缀"的方法（裁定 12）。<br>**`M9` 例外**：重新核对两条（`tlds.ts` 的 `analytics`、`wiki`，理由都是顶级域名数据）。<br>**构建体积**：写进收尾评审，对比数据见第 1 节 |

## 8. spec 第 3 节的裁定

第 3 节的 15 项全部采纳，其中两项在执行中进一步落实：

1. **Task 划分**：活跃迭代推广页并入数据分析，自动关闭单列为 Task 6，照此执行。9 节的 5 条"硬"顺序全部遵守（文档页在编辑器收敛和首页之前、便签在首页之前、导出与集成服务同删、先搬 `REVERSE_RELATIONS`、`EIssueLayoutTypes.GANTT` 最后删）。
2. **自定义导航弹窗拆开**：项目导航偏好由 `ProjectNavigationDialog` 配置，浏览器探测 1 验证过（选"标签页"会保存 `navigation_control_preference: "TABBED"`）。
3. **固定侧边栏显示常量里的全部条目**：`navigation.test.ts` 守住，浏览器探测 1 验证了 6 个条目及其顺序。
4. **propel 多删的图表**：照此执行，`area-chart` 和 `recharts` 保留。
5. **数据分析文案里的 Epic 键**：随页面删除，P3 的 Epic 规则按确切符号写，不受影响。
6. **`useTimeLineRelationOptions` 删除**：7 个调用方直接引用 `ISSUE_RELATION_OPTIONS`。
7. **工具栏收敛放在便签那个 Task**：照此执行。`extensions.test.ts` 断言工具栏只有一组，而且没有只属于文档页的条目。
8. **页面专用的 CSS 变量放在 Task 8**：照此执行。整分支评审另外发现两个页面专用的显示选项（`wideLayout`、`fontStyle`），由修复轮删除。
9. **首页新增 `home-body.tsx`**：照此执行，浏览器探测 1 验证了"有数据"和"空状态"。
10. **`theme-legacy.ts` 改名 `theme.ts`**：照此执行。
11. **`integration` 命名空间在 P2 整删**：照此执行。集成和导入的其余词汇按裁定 10 留给 P3。
12. **遗留依赖集中在收尾删**：照此执行。修复轮又删了 `ui` 的两个 atlaskit 依赖，锁文件差异只有 `ui` importer 的两条删除。
13. **编辑器交互行为进仓库测试（控制者裁定）**：已落实。jsdom、`mainFields`、7 个交互测试加 7 个组成测试；去掉 `mainFields` 时测试以 `RangeError` 失败（附录 6.5）。交互测试还发现了 `UniqueID` 回归，证明这项裁定值得做。
14. **基点就已死的文案和图片不在 P2 删**：执行中改为裁定 3。以被删功能命名的随功能删，其余仍留给收尾。
15. **`.superpowers/` 的忽略（控制者裁定）**：已落实。SDD 工作区由脚本写入 `.superpowers/sdd/.gitignore`。每次提交前 `git status --short` 只含本次改动，全程没有出现 `?? .superpowers/`，关键词守卫也从未因工作区里的任务说明而失败。
