# M7 / P5b plan：网页的收集箱——新建弹窗、文件与嵌入

> spec：[P5b spec](../specs/P5b-web-intake-files-embed.md)（§2 是合同，§3 的 R1–R30 是裁定，§6 的高风险是 R9、R20，§8 是主控对 O-1–O-4 的裁定：O-1 乙、O-2 甲（由 R26 细化）、O-3 关闭、O-4 甲）；预检 [re-95](../../../reviews/re-95-M7P5b预检.md)（修正后通过，重要 5、次要 14、观察 7），本 plan 已照主控的裁定修正。基线 main `ab86965f`（`ab86965fc615fce57bfa4197515e58205e939222`，P5a 的 noff）。worktree `.codex/worktress/codex/m7-p5b`，分支 `codex/m7-p5b`。台账 `.superpowers/sdd/P5b-web-intake-files-embed/progress.md`（ignored）；主控裁定 `rulings.md`（B-1–B-23），调研 `research/`。实现代理每个 Task 先读同目录的 `implementer-brief.md`。14 个 Task（B-1 的 (b)；对照见下）。下文 `C/` = `web/apps/web/core/`，`A/` = `web/apps/web/app/(all)/[workspaceSlug]/(projects)/`，`IN/` = `C/components/inbox/`，`WN/` = `C/components/workspace-notifications/`，`K` = `tools/keywords.json`，`IY` = `api/modules/issue.yaml`，`HO/` = `docs/v0/M7-collaboration/handoffs/`，`E/` = `e2e/`，`L` = 台账目录。

**与 B-1 的对照**：B-1 的 T1→T1；T2、T3→T2（`getTabIndex` 并进新建弹窗，spec R1）；T4→T3；T5→T4；T6→T5；T7a→T6a；T7b→T6b（检查点 1）；T8→T7；T9→T8a、T8b；预检 I5 加 T8c（负责人集合在轮到时算，产品代码，在检查点 1 之后）；T10→T9（检查点 2）；T11→T10；T12→T11。

## Global Constraints

**分支与提交**
- 主目录永远在 main。提交用详细中文，前缀 `feat/fix/docs/test/refactor(M7/P5b)`，结尾一行 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`。提交只在本 Task 的 lint 过了之后，不与它串在一条命令里，用显式路径（`git commit -F <msg> -- <paths>`）。
- 实现代理不推送、不合并；检查点由主控推送。
- 关键词的 phase 仍是 `M7/P5`（P5a、P5b 共用，S5a R1）；不延长任何 `until`。
- 本段不改 `server/`、`api/`、`deploy/`、`.github/`，不改设计文档。超出 spec 的契约或架构改动先停下报告。

**只读与禁区**：`plane/`、`refer/` 只读，不在其内跑 git。不 stash、`git clean`、`reset --hard`、强推；还原变异只经 `mutate.py`（P5a T6b 的险情：`git checkout -- <file>` 被权限分类器挡下）。不动别的项目的容器。容器套件一次一个：`make test`、`make test-shard`、`make e2e` 与 Playwright 的配置不并发。

**代码规则**（总体设计 §7.2、§7.7）
- 手写文件约 400 行以内，新代码进新文件，按关注点分（spec §2.9 的表）。新文件的版权头 `Copyright (c) 2026-present OpenNerve`，改写的 Plane 文件保留原头。
- 类型只来自 `@nerve/api-client`；service 不转换；store 只经自己的根找兄弟；SWR 只经 `useSessionSWR`、只用它的三个选项（`C/lib/use-session-swr.ts:38`）；新代码不读 router store 的地址。
- 收集箱条目的写只经 M4 的逐工作项写队列（S5a R3）；本段只在 T8c 改写的层（R20），请求体轮到时才算（`docs/v0/v0-design.md:638`）。
- `await` 之后的提示、导航与跟进一律经 `followInSession`；拒绝经 `useRefusalToast`；不 `console.error`；焦点照 S5a R35，嵌入与通知页照 spec §2.5（R26），任何结局都不落到 `body`。
- 删掉不用的代码，不加开关、桩、只给测试用的钩子或兼容别名。
- 不放宽 `K` 的规则、`lint-cap`、knip、`error-messages.test.ts`、i18n 的 `check:sync`。
- **不再加长的文件**（`wc -l` 于基线）：
  - `IN/content/header/use-intake-actions.ts` 300：T5 只改 `offeredOn` 的返回、不变长（收据 `t5-wc`）；T6a 先纯搬移，之后不超过 300
  - `C/components/issues/peek-overview/view.tsx` 303（不碰）
  - `C/components/issues/issue-modal/base.tsx` 313（不碰）
  - `C/store/issue/root.store.ts` 329（不碰）
  - `C/components/workspace/sidebar/projects-list-item.tsx` 431（不碰）
  - `web/packages/ui/src/dropdowns/custom-search-select.tsx` 302（不碰：R20 修在写的层与调用方）
  - `IN/content/root.test.tsx` 404（只为端口的形状编译跟随，嵌入的格进新文件）
  - `C/components/browse/detail-route.test.tsx` 480（不碰，重定向的格进新文件）
  - `C/components/issues/peek-overview/properties.test.tsx` 351、`C/components/issues/issue-detail/sidebar.test.tsx` 346（不碰，隐去状态与负责人增删的格进新文件）
  - `WN/root.test.tsx` 238（只照允许清单改，新的格进新文件）
  - `E/stories/collaboration/h9-intake-list-page.spec.ts` 398、`h11-intake-content-page.spec.ts` 397、`h20-intake-pages.spec.ts` 363（不碰）
  - `E/fixtures/intake-pages.ts` 344（新帮手进新夹具）
  - `E/stories/issue/editor-loading.spec.ts` 398（T7 的搬移之后变短）
  - `E/stories/smoke/s2-web-app.spec.ts` 582（不碰）
  - `E/fixtures/cycle-pages.ts` 388（只导入）
  - `C/store/issue/issue-writes.ts` 209、`issue-write-confirm.ts` 184、`issue-write-intake.ts` 149（只在 T8c 动，各不过 220）

**knip 与第一个读者**（新文件、新导出在它第一个读者出现的 Task 里建立；测试数据的导出以第一个读它的测试为准）：
- T2：确认放弃的两个文案键；`create-root.tsx` 交出 `requestClose` 的 ref 类型与 `onSubmittingChange` prop（读者：`modal.tsx`）；`IN/modals/create-modal/fake-create-modal.tsx` 的每个导出（读者：三份新测试）。
- T3：`status.ts` 的 `intakeEditableBy`（读者：`content-view.ts`、`detail.tsx`）。
- T4：`IN/content/widgets.tsx`（读者：`detail.tsx`）。
- T5：`IN/address.ts` 的 `intakeAddress`（读者：浏览页的布局、`create-root.tsx:146`）。
- T6a：`IN/content/header/intake-actions.ts`（读者：两个页头、动作模型、`content/root.tsx`）；`IN/content/selection-port.ts` 的端口类型（读者：`IN/root.tsx`、`content/root.tsx`、动作模型）。
- T6b：`peek-overview/use-peek-issue-read.ts`（读者：`PeekContent`、`WN/root.tsx`）；`WN/landing.ts`（读者：`NotificationsRoot`）；嵌入的端口（`WN/root.tsx`）；`openLink`（读者：`take("open")`、宽页头）；`intakeAddress` 的第三个读者。
- T7：`E/fixtures/editor-chunks.ts`（第一个读者：搬移提交里的 `editor-loading.spec.ts`）；`E/fixtures/intake-create.ts`（读者：H8 的故事）。
- T8a：`E/fixtures/intake-files.ts`（读者：H11 的两份文件故事）。
- T8c：负责人增删的类型与帮手（读者：`changeBody` 与七个调用方）。
- 没有“成员先于读者”的例外。删掉某个导出最后一个读者的 Task 同一提交删去它（T2 的 `INTAKE_ISSUE_FORM` 一组）。knip 查不出只被测试读的产品导出与包的桶文件导出：逐 Task 的审查与死成员的整份比对兜底。

**告警上限**：`lint-cap` 要求 oxlint 的告警数恰好等于上限（web `web/apps/web/package.json:10` 现为 18；e2e、`@nerve/constants` 为 0）。新文件不得新增告警。T2 删去 `create-root.tsx:2-3` 两条 `oxlint-disable` 与它们管的 `div role="button"`：若计数变了，同一提交照实改上限并留收据（`t2-oxlint-count`）；T11 只核对。

**文案**：新键只有 `inbox_issue.modals.discard.title`、`inbox_issue.modals.discard.content`（T2，en 与 zh-CN 同一提交）；文案键写完整的字面量（`tools/i18n-keys.mjs` 才认得），不许一个键是另一个键的前缀。别的一律用已有的键（spec R6、§2.5 的 `close`、§2.3 的 `issue.add.link`）；实现中觉得要新键先记台账并报主控。

**守卫**
- `project-address-route`：T4 的 `IN/content/widgets.tsx` 读项目与角色，收入规则的文件名单并带 hit 样例（预检 m13）；T6a 搬出的新文件若读项目（`useProject`、权限 store）同样收入（`use-intake-actions.ts` 在它的范围里，先例 P5a T5c）；改后跑 `node tools/keywords.mjs`。
- `.oxlintrc.json` 的 swr 名单（`:120-210`）：T6b 的 `peek-overview/use-peek-issue-read.ts` 收入；`WN/root.tsx`（`:198`）、`peek-overview/root.tsx`（`:162`）已在；任何新取数文件收入名单。
- `old-intake-urls`、`copy-hook` 等既有规则不动。

**允许改的已有测试与夹具**（改之前在台账逐条写明，只照这里写的改法；其余断言是回归，不改；编译跟随只为编译、不改断言，逐处记台账；任何一条要改断言而这里没写，先停下报告）：
- T2 `IN/modals/create-modal/create-root.request.test.tsx`（191）：删 `:53` 的 `use-platform-os` 替身、`:77-81` 的 `NoResize` 与 `:86` 的 `stubGlobal`、`:99-100` 两个 prop；加 `onSubmittingChange={() => {}}`（编译跟随）；加对 `@nerve/ui` 的部分替身，只把 `ModalTitle` 换成普通的 `h3`（预检 I1：表单在 `ModalCore` 之外渲染，`DialogTitle` 取不到 `Dialog` 即抛错；先例 `C/components/issues/issue-modal/base.clone.test.tsx:94-95`），原文：
  ```tsx
  vi.mock("@nerve/ui", async (importOriginal) => ({
    ...(await importOriginal<typeof import("@nerve/ui")>()),
    ModalTitle: ({ children }: { children: React.ReactNode }) => <h3>{children}</h3>,
  }));
  ```
  断言一条不改（成功之后 `ui.close` 一次照旧：成功直接关，spec §2.1）。
- T2 `IN/modals/create-modal/create-root.files.test.tsx`（143）：删 `:28` 的 `use-platform-os` 替身、`:40-44` 的 `NoResize` 与 `:56` 的 `stubGlobal`、`:109-110` 两个 prop；加 `onSubmittingChange={() => {}}` 与上面同一段 `@nerve/ui` 的部分替身；断言不改。删去 `stubGlobal` 之后若有格因 jsdom 没有 `ResizeObserver` 失败，先停下报告（R3 的前提不成立）。
- T3 `C/store/asset/content-view.test.ts`（197）、`C/store/issue/intake/status.test.ts`（144）、`IN/content/detail.test.tsx`（181）：只追加格；`detail.test.tsx:39` 的 `useIssueImages` 替身的回答可改为一个可辨的 `images`（只为图片的那一格），别的不改。
- T4 `IN/content/detail.test.tsx`：加一行 `vi.mock("./widgets", …)`（`widgets.tsx` 的格在它自己的测试里）。渲染真实 `IntakeDetail` 的只有这一份（`root.test.tsx`、`root.after-read.test.tsx`、`triage.test.tsx`、`accept.test.tsx` 都替身了它）。**实施中改**（T4，主控同意）：不止这一份——P5a 的 `IN/content/issue-properties.test.tsx` 与 T3 新加的 `IN/content/detail.images.test.tsx` 也画真实的 `IntakeDetail`（不在宿主里），`t4-green-1` 里它们 31 格败在 “Issue detail modals require their detail or peek host”；两份各加同一行 `vi.mock("./widgets", () => ({ IntakeWidgets: () => null }))`，断言不改（编译跟随的替身）。
- T5 `IN/content/header/header.test.tsx`（239）：`:151-152` 的 `TRIAGE` 去掉 `open`，`READ` 拆成已接受的 `["copy","open"]` 与其余的 `["copy"]`；`:155-177` 矩阵的期望照 spec §2.4（待处理、暂缓、拒绝、重复、已归档只有 `copy`，已接受有 `copy`、`open`）；`:187-192` 一格随 `TRIAGE`。理由：R15。
- T5 `E/stories/collaboration/h10-intake-triage-page.spec.ts`：`expectNoTriage` 的注释（`:52-54`）与 `:58` 改为断言 “Open work item” 不在（它的三个读者 `:217`、`:220`、`:226` 看的都是待处理的条目）。理由：R15。
- T6a `IN/content/header/header.test.tsx:105-123`（`:105` 的 `selection` 加上 `neighbour`，窄页头的 `isMobileSidebar`、`setIsMobileSidebar` 换成端口）、`IN/content/root.test.tsx:160-175`（`InboxContentRoot` 的 props）：只为端口的形状编译跟随；导入路径随纯搬移跟随（`header.test.tsx:17`）。断言不改。
- T6b `WN/root.test.tsx`（238）：读的 hook 与 SWR 用真的，只替身取数（预检 I4、m11）：
  - `:23-27` 的 `fixture` 加工作项的读（同 `reads` 的形状）与 canonical 的详情；`:73-75` 的 `beforeEach` 重置它们；
  - `:44` 的 `use-issue-detail` 替身给出 `rootIssueStore`：读的 hook 与 `useDetailScope` 读到的成员（`rootStore.workspaceRoot.getWorkspaceBySlug`、`permissionReads.identity`、`issues.getIssueDetailById`、`active`、`rootStore.user.permission.getProjectRoleByWorkspaceSlugAndProjectId`）；
  - 加 `vi.mock("@/components/issues/issue-detail/use-issue-operations", …)`，`fetch` 立即以夹具的详情（不在收集箱）答、写进 canonical，P4 的各格照旧见到浮层；选中之后若要多等一次落定，用既有的 `settled()` 帮手，只为读的落定；
  - `:45-57` 浮层组件的替身照旧（`IssuePeekOverview` 一个；读的 hook 在它自己的文件里，不经这个桶文件）；
  - `:215-221` 一格改为“未被接受的收集箱条目嵌入收集箱的内容（`[data-triage]` 在、没有浮层）”。
  - 别的断言不改。
- T6b `E/stories/collaboration/h2-h4-notification-cards-page.spec.ts`：`:257-264` 改为断言嵌入（spec §2.8 的表），“Close” 限在名为 “Preview” 的区域里定位（toast 的关闭也叫 “Close”，`web/apps/web/app/provider.tsx:35`）；文件头注释 `:49` 与测试标题 `:162` 里 “an intake item's card opens a plain peek” 一句随之改写（标题仍带 `H2–H4 (page)`）；点卡片之前登记嵌入自己那一包读（第二个 `GET /api/v0/issues/{id}` 与它之后的 `/activities`、`/comments`、`/description-versions`）的 `waitForResponse`（**实施中改**，T6b：一次取数有两个这样的 `GET`，子工作项的读核对父项再读一次，`C/store/issue/issue-details/sub_issues.store.ts:99`；嵌入的那一包是第三、第四个，故事等第四个与 `/description-versions`，其余的读以 `recordApi` 的在途归零兜住），离开项目（`:270`）之前等它们答完（预检 m12：末尾对 `watch.apiFailures` 是精确比对，`:292-296`）；`:272-297` 的 404 一步不改（O-1 乙：失败界面是预览自己的、同一个 `DetailResourceError`）。**这一步在 T6b 与产品改动同一提交**（O-4 甲）。
- T6b P4 的其余通知测试（`WN/page-state.test.tsx`、`WN/sidebar/*.test.tsx`）：`WN/sidebar/list.tsx` 只加一个 `data-notification-list` 标记，预期不受影响；有断言要改先停下报告。
- T7 `E/stories/issue/editor-loading.spec.ts`（398）：`:14-53` 的 `editorChunks`、`holdEditorImplementation` 搬到 `E/fixtures/editor-chunks.ts`，导入跟随；断言不改。“逐字”之外只许两处（预检 m8）：`editorChunks` 一起导出（故事里另有两个读者，`:160`、`:215`）；构建产物的相对路径从 `../../../web/…`（`:17`）改为 `../../web/…`。
- T8c `C/store/issue/issue-write-fields.test.ts`：`changeBody` 的第二个参数由所持的模块集合改为所持的工作项，`:127`、`:150`、`:151`、`:155`、`:156`、`:163`、`:165` 的实参照此改写（`{ module_ids: [...], assignee_ids: [] }`），断言不改；负责人增删的格追加在新文件。别的既有测试预期不改：请求体仍是整集（读请求体的测试与故事照旧），组件测试里没有断言下拉框交出整集的格（预检之后 `git grep` 核过：`peek-overview/properties.test.tsx`、`issue-detail/sidebar.test.tsx`、`IN/content/issue-properties.test.tsx` 只核负责人下拉框打开、列出成员）；有断言要改先停下报告。
- 不改、不受影响（回归）：`C/components/issues/issue-modal/close-policy.test.tsx`、`update-command.test.tsx`；`IN/content/root.after-read.test.tsx`、`triage.test.tsx`、`accept.test.tsx`、`navigation.test.tsx`、`IN/root.test.tsx`（`:45` 的替身只读 `inboxIssueId`）；M5 的上传与附件的测试；P4 的通知测试（除上面几条）；`issue-drag.test.ts`、`group-consumers.test.tsx`（拖放与新建的负责人照旧是整集）。

**红的窗口**：没有红的提交。产品改动让既有的格或故事变的，都在同一提交里照上面的清单改（T5 的 `header.test.tsx` 与 H10 `:58`，T6b 的 `WN/root.test.tsx` 与 H2–H4）。T8c 的连点一步先在没修的构建上跑红只留收据（`t8c-red-page-q34`），与修复同一提交才进库。名单之外的任何红先停下报告。

**检查点**（七项用 `sh L/gates.sh <前缀>`；带页面配置用 `sh L/final.sh <前缀>`：七项之后跑 m7-pages、m6-pages、m6-acceptance、m4-detail、m5-content、m4-list 六个配置，三种引擎，spec R2）：
- **0（T2 之后）**：`final.sh cp0`；主控推送、核分支 CI；台账记基线：本机第 1 路 bootstrap（P5b 的基线）、web 作业的 Lint、Unit tests、Build 三步、e2e 两片、Go 两路，以 `gh` 读 main CI `45197451256` 的同名作业与步作对照；m4-list 这一次是 T8c 之前的基线。
- **1（T6b 之后：除 T8c 之外的产品代码完成）**：`final.sh cp1`；页面上核实非收集箱的通知只一个 `GET /api/v0/issues/{id}`（收据 `cp1-peek-reads`，见 T6b Step 4）；主控推送、核分支 CI。T8c 是检查点 1 之后的产品代码（spec R1）。
- **2（T9 之后，故事与 T8c 完成）**：`final.sh cp2`（六个配置，m4-detail 与 m4-list 兜住 T8c）；新故事文件 chromium `--repeat-each=3`（收据 `cp2-repeat`）；推送之前照 P4 spec §2.11 `:270` 估出 e2e 两片的时长（以最近一次分支 CI 的两片为底，加新文件的 chromium 时长 × CI 与本机之比），估出超过 30 分钟先报告、不推送（P5a 检查点 2 漏过一次，R5a `:110`）；之后核分支 CI。
- **终检（T11）**：`final.sh fin`（同六个配置）。
- **另跑的单配置**（因为动了别的 M 的共用件，先 `make build`）：T3 之后 chromium 的 m5-content（`t3-m5-content`）；T5 之后 chromium 的 m4-detail（`t5-m4-detail`）；T6b 之后 chromium 的 H2–H4（`t6b-h2-h4-follow`）；T7 的搬移之后以 m4-detail 配置三引擎跑 `editor-loading.spec.ts`（`t7-move-editor-loading`）；T8c 之后 chromium 的 m4-detail 与 m4-list（`t8c-m4-detail`、`t8c-m4-list`）。
- 第 1 路：本机超过 CP0 基线 +30 秒先停下查；CI 的 bootstrap 过期限的八成只报告（P5a 的最终分支 CI 是 80.7%，`.376` 映像）。本机 Go 两路因 M6-M8-05 失败且重跑一次仍不过时，以分支 CI 的服务端两路为准并记录。

**证据**
- 每次检查经 `run-check.py`（`python3 L/run-check.py <label> <command…>`），变异经 `mutate.py`（每个旧串恰出现一次，工具恢复并核 sha256、打印 `restored=yes`）；标签 `t<N>-…`、`cp<k>-…`，不重用。
- **RED 先于实现**：先写失败的测试，看它因所测性质失败（不是编译失败），留 RED 收据，再实现到绿。P5a 有两次实现先于测试（R5a §3 的 T6a 与修复轮 m4），记为过程疏漏；本段不许。
- 下表每个性质至少一个能编译、能让测试失败的变异。两层都在查同一件事时两层一起改，并留只改一层的对照。只有竞态杀得死的变异不算。变异改的是别的 M 的共用件（`modal-core.tsx`）时，照样经 `mutate.py`，工具恢复。
- **页面故事先写一个期望错误的版本**（或以 e2e 变异），看到断言在被改处失败，再改正；收据 `t<N>-red-page-…`。

**测试时间**
- Go 不变，不加顶层名。
- web 作业期限 20 分钟、960 秒报告线；每个检查点记 Lint、Unit tests、Build 三步；加大量 vitest 的 Task 记新文件的时长。越线时的判别（spec R2）：三步对 main CI `45197451256` 的同名步（93 / 818 / 17 秒）各算比值，三个比值相近是慢 runner，照报告、不停；只有 Unit tests 的比值明显高于另两步才是本段的增长，先报告主控再继续。基线本身是慢 runner 的（959 秒），本段净增 vitest，慢 runner 上越线几乎是确定的；快 runner 上三个比值都小于 1，照样看三者是否相近（预检 O6，只报告）。
- e2e 每片 35 分钟、报告线 30 分钟。T7–T9 记每个新文件每个 `test()` 在三种引擎的时长（JSON 报告）；本机 chromium 每个文件约 8 秒是先例的上限，超了写原因（`[s3]` 每个测试自起 nerve 与桶，会超）；任何一个 `test()` 单引擎超过 30 秒先报主控。

**页面故事的教训**（R5a §3、§7、§9，预检 re-95）
- jsdom 只钉落点的目标，真实焦点以三引擎的故事为准；键盘、指针、窄屏（800 宽）三条路分开看（P5a 的 D1、D2 都是 T13 的故事才查出）。
- 跳转、移出、离开项目或下一步写之前先等在途的读答完：动作之前登记 `waitForResponse`（P5a 的 I9、H6 两个竞态；预检 m12 的 H2–H4）。嵌入的故事先等判定的读与嵌入自己的读答完再分诊。
- 持有详情宿主的读不随重连重读（P5a 整审 I1；预检 I4 的判定的读同样）：每次挂载以 `mutate()` 读自己的，判定锁定，不看共用键上的 `isValidating`。
- 编辑器代码块的扣住在页面载入之前登记（C10 的次序），页面上不能先有编辑器建成（空的收集箱），每步之前断言 `held.length > 0` 且没有 `[contenteditable="true"]`（预检 I2）。
- 共用浮层以 fixed 定位，祖先带 `transform` 会让它开到视口之外（`b1aeccb1`）：嵌入的菜单与下拉框以 `expectPopupBesideButton` 核。
- “今天”“明天”在动作那一刻向浏览器取（`browserDay`，`E/fixtures/intake-pages.ts:104`），不取测试进程先算好的日（P5a 整审 m5）。
- 按名字定位的按钮限在它的区域里：嵌入的 “Close” 与 toast 的关闭同名（预检 m12）。
- 一律放行的路由只在 Chromium 放（WebKit 在应用代码载入之前有路由时重取预载、控制台警告）；不在重载之后查控制台（M6-M8-09）；不给任何东西起名 “Notifications”。
- 每个故事以 `expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []])`（预期的拒绝写进去，如 `completeAsset` 的 409）与 `expectQuietConsole` 结束。
- `AlertModalCore` 的默认按钮文字是英文（`web/packages/ui/src/modals/alert-modal.tsx:62-66`）：显式传 `t()`（P5a 整审 m4）。
- 已知偶发：F11 的 `net::ERR_ABORTED GET /api/v0/assets/…/content`（M6-M8-07），H11 的图片故事可能遇到：该文件重跑一次并记下；第二次失败停下报告。

## 性质与变异

| 性质 | 变异（标签） | Task |
|---|---|---|
| 在途不关（两层） | `modal.tsx` 在途也给 `handleClose` 且 `requestClose` 不看在途（`t2-mut-close-in-flight`）；只改一层的对照 `t2-ctl-close-in-flight-one-layer` | T2 |
| 在途的 Escape 不动焦点（R28） | 在途也给 `handleClose`（`t2-mut-handleclose-in-flight`，在途按 Escape 之后焦点离开名字的一格失败） | T2 |
| 编辑器没建成可关 | 引用为 `null` 也拒（回到 Plane 的 `!ref?.isEditorReadyToDiscard()`，`t2-mut-null-ref-refuses`） | T2 |
| 已建成且未就绪拒绝 | 照关（`t2-mut-unready-closes`） | T2 |
| 有改动先确认、没有改动直接关 | 有改动也直接关（`t2-mut-dirty-closes`）；没有改动也确认（`t2-mut-clean-confirms`） | T2 |
| 描述按 `isEmptyHtmlString` 判空、名字去空白 | 按字面比（`t2-mut-description-literal`，编辑器写下空段落的一格失败）；名字不去空白（`t2-mut-dirty-untrimmed`，只有空格的名字弹确认框的一格失败） | T2 |
| 只经 `handleClose`，没有文档级 Escape | 加回 `useKeypress("Escape")`（`t2-mut-document-escape`，确认框开着时按 Escape 的一格：外层也关） | T2 |
| IME 组字的 Escape 不关 | `modal-core.tsx:99` 不看 `isComposing`（`t2-mut-ime-escape-closes`） | T2 |
| 每次打开从初值起 | `ModalCore` 的 `Dialog` 关上时不卸下内容（`modal-core.tsx` 加 `unmount={false}`，`t2-mut-modal-keeps-content`，确认放弃、再开的一格失败） | T2 |
| 提交仍要求就绪 | 引用为 `null` 也发（`t2-mut-submit-unready`） | T2 |
| 成功直接关 | 成功也经 `requestClose`（`t2-mut-success-through-request`，有改动的成功弹出确认框的一格失败） | T2 |
| 迟到的跟进只在表单还挂着时（R29） | 去掉挂载的范围（`t2-mut-follow-unmounted`，卸下之后放开提交的一格：有跳转与提示） | T2 |
| 名字按字符、去空白 | 按 `name.length`（`t2-mut-length-units`，255 个 emoji 被拒的一格失败）；不去空白（`t2-mut-no-trim`，只有空格的名字发出的一格失败） | T2 |
| 没有原生的 `required` | 加回 `required`（`t2-mut-native-required`，空名字提交显示 `FieldError` 的一格失败：jsdom 30.1.1 的 `requestSubmit` 先 `reportValidity`，无效即不发 `submit`） | T2 |
| 提交不停用、被拒焦点给名字 | 超长时停用提交（`t2-mut-submit-disabled`）；不经 `useRefusedFocus`（`t2-mut-no-refused-focus`） | T2 |
| `aria-invalid` 恰在提示显示时 | 恒为假（`t2-mut-invalid-off`） | T2 |
| nerve 点名 `name` 放在字段下 | 一律提示（`t2-mut-name-refusal-toast`） | T2 |
| 弹窗以 `ModalTitle` 为名 | 回到普通 `h3`（`t2-mut-plain-title`） | T2 |
| 开关不套在按钮里、在途停用 | 包回 `div role="button"`（`t2-mut-switch-in-button`）；在途可点（`t2-mut-switch-in-flight`） | T2 |
| 自然 Tab 次序 | 给提交按钮 `tabIndex={1}`（`t2-mut-positive-tabindex`） | T2 |
| 英文改键 | 就绪的提示回到英文字面量（`t2-mut-literal-toast`） | T2 |
| 收集箱里按收集箱的规则写 | `issue` 宿主不看 `inIntake`（`t3-mut-canwrite-role-only`，访客提交者与非提交者成员两格失败） | T3 |
| 已处理、收集箱关着不能写 | 谓词不看状态（`t3-mut-closed-writable`）；不看 `intake_view`（`t3-mut-intake-off-writable`） | T3 |
| 接受之后照旧 | 已接受也按收集箱的规则（`t3-mut-accepted-intake-rule`，已接受的成员一格在 `canWrite` 一边失败） | T3 |
| 一处谓词，详情与内容视图在收集箱里同答 | 谓词去掉提交者一支（`t3-mut-predicate-submitter`，`detail.test.tsx` 与 `content-view.test.ts` 都失败） | T3 |
| 要求项目角色 | 谓词不看角色（`t3-mut-predicate-no-role`） | T3 |
| 评论宿主不受影响 | 收集箱的规则也用到 `comment`、`new_comment`（`t3-mut-comment-host-intake`） | T3 |
| 删附件随 `canWrite` | `canRemoveAttachment` 不看 `canWrite`（`t3-mut-remove-attachment-closed`，管理员在已拒绝的条目上仍可删的一格失败） | T3 |
| 描述有图片能力 | 不给 `images`（`t3-mut-no-images`） | T3 |
| `ConfirmedContentNotice` 在宿主之内 | 画到宿主之外（`t3-mut-notice-outside-host`，确认的上传之后没有提示的一格失败） | T3 |
| 附件按谓词 | `widgets.tsx` 给 `disabled={false}` 且 `canWrite` 恒真（`t4-mut-attachments-writable`）；只改一层的对照 `t4-ctl-attachments-one-layer` | T4 |
| 链接照 `issue.update`、任何状态 | 链接按收集箱的谓词（`t4-mut-links-intake-rule`，非提交者成员在已拒绝条目上的一格失败） | T4 |
| 访客只读、已归档只读 | 访客有按钮（`t4-mut-links-guest`）；已归档有按钮（`t4-mut-links-archived`） | T4 |
| 没有子工作项与关联的按钮 | 渲染 `IssueDetailWidgets`（`t4-mut-m4-widgets`） | T4 |
| 链接按钮有名字 | 去掉 `customButton`（`t4-mut-link-button-unnamed`） | T4 |
| 通用面隐去状态 | 浮层不隐（`t5-mut-state-peek`）；侧栏不隐（`t5-mut-state-sidebar`）；已接受也隐（`t5-mut-state-hidden-accepted`） | T5 |
| 重定向在页头与详情挂上之前 | 先渲染 `Outlet` 再送（`t5-mut-redirect-after-mount`，“详情从未挂上”的一格失败） | T5 |
| 以 `replace`、按状态选页签 | `push`（`t5-mut-redirect-push`）；一律 `open`（`t5-mut-redirect-open-only`） | T5 |
| 关着照样送、已接受不送 | 看开关（`t5-mut-redirect-gated`）；`intake` 非空就送（`t5-mut-redirect-accepted`） | T5 |
| 收集箱页的“打开”只给已接受的 | 待处理也给（`t5-mut-open-unaccepted`）；宽页头无条件渲染（`t5-mut-open-always-rendered`） | T5 |
| 纯搬移 | 收据：搬移前后逐行相同、只多几处 `export`（`t6a-move-identical`）、既有测试一行未改且全绿（`t6a-move-tests-untouched`） | T6a |
| 动作模型不读页面状态、邻行只来自端口 | 加回 `useIntakePageState()`（`t6a-mut-reads-page-state`，provider 之外的一格抛错）；`neighbour` 恒为空（`t6a-mut-neighbour-ignored`，既有 R14 的格失败） | T6a |
| 嵌入里分诊之后不移 | 嵌入的分诊也调 `leave`（`t6b-mut-embed-leaves`） | T6b（收集箱一侧） |
| 嵌入没有导航、关闭按钮有名字 | 嵌入也渲染页面的导航（`t6b-mut-embed-navigation`）；关闭按钮没有名字（`t6b-mut-close-unnamed`） | T6b（收集箱一侧） |
| 嵌入的“打开”按状态、复制不变（R30） | `openLink` 一律是 `link`（`t6b-mut-open-link-shared`，待处理条目的“打开”到浏览页的一格失败）；复制读 `openLink`（`t6b-mut-copy-open-link`） | T6b（收集箱一侧） |
| 读后判定 | 读落定之前以 SWR 留着的回答判定（`t6b-mut-decide-before-read`） | T6b（通知一侧） |
| 未被接受的嵌入（含拒绝）、已接受的开浮层 | 拒绝的开浮层（`t6b-mut-declined-peek`）；已接受的也嵌入（`t6b-mut-accepted-embed`） | T6b（通知一侧） |
| 判定一次 | 随 canonical 改判（`t6b-mut-rejudge`，嵌入里接受之后换成浮层的一格失败） | T6b（通知一侧） |
| 预览的读不随重连重读（R27） | 预览的读不传 `revalidateOnReconnect: false`（`t6b-mut-reconnect-rereads`，越过去重窗口之后发 `online` 再取数的一格失败） | T6b（通知一侧） |
| 判定锁定，不看共用键的验证 | 落定按 `answered && !isValidating` 每次重算（`t6b-mut-validating-unmounts`，浮层的重试与重连两格：浮层被卸下、输入没了） | T6b（通知一侧） |
| 判定按代记（R27） | 判定只按通知记（`t6b-mut-decision-ignores-generation`，换了权限身份之后沿用旧判定、没有新读的一格失败） | T6b（通知一侧） |
| `setPeekIssue` 只给浮层 | 嵌入也设（`t6b-mut-peek-set-for-embed`） | T6b（通知一侧） |
| 读失败的一支（O-1 乙） | 失败时渲染浮层（`t6b-mut-failure-peek`，“浮层没有挂上、失败界面是预览自己的”一格失败）；重试落定之后不再判定（`t6b-mut-retry-no-judge`） | T6b（通知一侧） |
| 通知页的落点链（R26） | 不找卡片（`t6b-mut-landing-no-card`）；不找列表的区域（`t6b-mut-landing-no-list`，卡片不在的一格落到预览）；没有预览的区域一环（`t6b-mut-landing-no-preview`，列表没画出来的一格落到 `body`）；`leave` 不清选中（`t6b-mut-delete-keeps-selection`） | T6b（通知一侧） |
| 普通浮层的关闭也走落点（O7） | 浮层的 `embedRemoveCurrentNotification` 直接 `clearSelection`（`t6b-mut-peek-close-no-landing`） | T6b（通知一侧） |
| 负责人集合在轮到时由增删算（R20） | `changeBody` 不看所持的集合、调用方交整集（`t8c-mut-assignees-at-call`，排在一个负责人写之后的第二次点击的一格与连点的页面一步失败）；收集箱详情的调用方交整集（`t8c-mut-caller-whole-set`）；合并时不理会删除（`t8c-mut-merge-drops-remove`） | T8c |

## Tasks

### T1 基线与文档（主控）

- **提交**：spec、plan、预检（含处理）。
- **P5a 的回填**（spec R1；数字取自归档台账 `/Users/xiaoruan/project/nerve-project/.superpowers/sdd/M7-P5a-integration-20261123/payload/worktree/.superpowers/sdd/P5a-web-intake/progress.md:683-687`）：`reviews/P5a-web-intake-review.md` 的状态行（`:3`）、§2 的 A9（`:50`）、§9 末句（`:179`）补“终检与集成”：终检 `fin`（`9983ed46`）一次全过，最终分支 CI `45189969560` attempt 1 全绿（第 1 路 bootstrap 968.9 秒 = 80.7%，`.376`，只报告），`--no-ff` 合入 `ab86965f`，main CI `45197451256` attempt 1 全绿（web 959 秒，同比例的慢 runner），归档 `.superpowers/sdd/M7-P5a-integration-20261123`；S5a `:387` 的 A9 勾上。
- **设计**：§14 的 P5a 一行（`D:351`）改为已完成、加 P5b 一行；§12 的 P5b 一行（`D:307`）链 spec。主控改，实现代理不碰设计。
- **关键词**：phase 仍 `M7/P5`；跑 `node tools/keywords.mjs`。
- **台账工具**：`final.sh` 已跑六个页面配置（含 m5-content、m4-list），只核对。
- **依赖**：根目录 `pnpm install --frozen-lockfile`（收据 `t1-install`）。
- **死成员的基线**（rulings.md 末条：P5a 修复轮的 `t15-dead` 是 1598 行、`--rows M7` 88 行，P5b 的 T1 再跑一次作基线）：先 `pnpm --filter web exec react-router typegen`，再 `node L/dead-members/deadsym.mjs --tsv > L/dead-members/dead-before.tsv`、`node L/dead-members/domains.mjs L/dead-members/dead-before.tsv --rows M7`（收据 `t1-dead-before`）；与 1598、88 不一致时逐行写明来由；列出 P5 剩下的 6 行（R5a `:116`）。
- **检查**：文档链接（`doc-links.py`）与关键词，然后提交 `docs(M7/P5b): …`。

### T2 新建弹窗 → **检查点 0**

**Files**：
- 改：`IN/modals/create-modal/{modal,create-root,issue-title,issue-description,issue-properties}.tsx`；`web/packages/constants/src/tab-indices.ts`（删收集箱一组）；en、zh-CN 的 `inbox.json`（两个新键）；两份既有测试（允许清单）。
- 新：`IN/modals/create-modal/fake-create-modal.tsx`（共用夹具）、`create-root.close.test.tsx`、`create-root.name.test.tsx`、`create-root.form.test.tsx`（预检 m4：约 35 格按关注点拆开，每份不过 400 行）。

**Steps**：
- [ ] Step 0：读 T1 的死成员基线（`L/dead-members/dead-before.tsv`，收据 `t1-dead-before`）与它列出的 P5 的 6 行。
- [ ] Step 1：先写测试（`@/lib/fake-toast`、`@/lib/fake-i18n`、`@/lib/fake-refusal`；编辑器以替身的引用：`null`、就绪、未就绪三种），看到它们因所测性质失败。要真实 `ModalCore` 的格经 `InboxIssueCreateModalRoot` 渲染、从 `document` 查：
  - `create-root.close.test.tsx`：关闭规则（spec §2.1 的表）逐行：在途时 Escape、点背景、Discard 都不关、Discard 与开关停用、`ModalCore` 没有 `handleClose`（在途按 Escape 之后焦点还在名字里）；引用为 `null` 时空表单关上；引用为 `null` 而名字有字时弹确认框；未就绪时提示 `editor_is_not_ready_to_discard_changes` 且不关；有改动（名字、描述、优先级、负责人、标签、截止日各一格）弹确认框；只有空格的名字、编辑器写下 `<p></p>` 而没有别的改动时直接关；确认框以 `inbox_issue.modals.discard.title` 为名、按钮是 `discard` 与 `common.cancel`；确认 → 关（`handleModalClose` 一次）；取消与 Escape → 只关确认框、名字还在、外层没关（文档级 Escape 不在）；点背景经 `handleClose` 调 `requestClose`（背景上派发 `pointerdown`、`pointerup`；jsdom 里触发不了时这一格写明留给 H8，单测只钉接线，记台账）；IME 组字的 Escape（`isComposing: true`）不关、不弹确认框；填字、确认放弃、等关上、再开是空的。
  - `create-root.name.test.tsx`：超长（`Array.from(name.trim()).length > 255`，含 255 个 emoji 不算超长的一格，含首尾空格的 255 字不算超长的一格）边打边显示 `errors.field.too_long`、`aria-invalid`、以提示为描述；空的与只有空格的各一格：提交被拒、显示 `title_is_required`、没有请求、焦点在名字上（以 `requestSubmit` 提交，原生校验在 jsdom 里照样拦）；提交按钮不停用；nerve 的 `validation_failed`（点名 `name`）在字段下并取得焦点，`issue.intake_disabled` 经提示、表单留着。
  - `create-root.form.test.tsx`：引用为 `null` 时提交提示、没有请求；成功之后直接关（有改动也不弹确认框）；“Create more” 开着时清空不关；成功提示的键；扣住提交、卸下、放开：没有跳转、没有提示（R29）；弹窗以 `inbox_issue.modal.title` 为名（`role="dialog"`）；“Create more” 是 `role="switch"`、不在任何 `role="button"` 里、在途停用；表单里没有正的 `tabIndex`；四个英文字面量不在。
- [ ] Step 2：实现到通过（spec §2.1）；`modal.tsx` 在途不给 `handleClose`；删 `required`；删死代码与两份测试的替身、加 `@nerve/ui` 的部分替身与 `onSubmittingChange`（允许清单）；`tab-indices.ts` 删收集箱一组；核对 `git grep getTabIndex -- web` 只剩 `views/form.tsx:75` 与函数本身。
- [ ] Step 3：变异照上表 T2 各行。`make lint-web`、聚焦 vitest、`make knip`、`make test-web`；三份新测试的时长；死成员整份比对（收据 `t2-dead-after`：P5 的 6 行消失，新出现的每一行写明去处）；oxlint 计数（`t2-oxlint-count`）。提交 `feat(M7/P5b): 收集箱新建弹窗的关闭政策——Escape、点背景与 Discard 一条规则，在途不给 handleClose，编辑器没建成可关，有改动先确认；迟到的跟进限在表单还挂着时；名字的提示与被拒、ModalTitle、开关本身、离开 getTabIndex`。
- [ ] Step 4（**检查点 0**）：`final.sh cp0`；主控推送、核分支 CI，台账记基线（Global Constraints）。

### T3 共用谓词、`canWrite`、描述图片与 `ConfirmedContentNotice`（spec §2.2，R9，高风险）

**Files**：`C/store/issue/intake/status.ts`（`intakeEditableBy`）与 `status.test.ts`（追加）；`C/store/asset/content-view.ts`（`canWrite` 的 `issue` 宿主一支）与 `content-view.test.ts`（追加）；`IN/content/detail.tsx`（读谓词、`useIssueImages`）与 `detail.test.tsx`（追加）；`IN/content/root.tsx`（`ConfirmedContentNotice`）与新测试文件或 `detail.test.tsx` 追加。

**Steps**：
- [ ] Step 1：先写测试，看到失败：
  - 谓词：管理员、成员提交者、访客提交者在待处理与暂缓为真；非提交者的成员、没有角色为假；拒绝、重复（管理员也）为假；收集箱关着、项目已归档为假；到期的暂缓（存的状态仍是暂缓）为真。
  - `canWrite`：所持详情在收集箱里时逐格同上（`issue` 宿主，`description` 与 `attachment` 两种用途）；已接受：成员真、访客提交者假（照旧）；`intake` 为 `null`：照旧；`comment` 宿主（本人的评论）与 `new_comment` 宿主在已拒绝的条目上照旧为真；`draft` 宿主不变；`canRemoveAttachment` 在已拒绝的条目上对管理员为假（删除的命令经它，`content-deletion.ts:45`）。
  - 详情与内容视图同答：一份表驱动两边，**只含条目在收集箱里的格**（待处理、暂缓、拒绝、重复四种状态，预检 m5）；已接受一格两边分别断言（详情只读、`canWrite` 按角色）。
  - 详情：`DescriptionInput` 收到 `useIssueImages` 的 `images`；可改时编辑器有上传能力，不可改时没有。
  - `ConfirmedContentNotice`：确认的上传之后详情区域说出 `content_files.confirmed_upload`（在宿主之内）。
- [ ] Step 2：实现到通过。`detail.tsx` 的 `editableBy` 删去、读谓词。
- [ ] Step 3：变异照上表 T3 各行。`make lint-web`、聚焦 vitest、`make knip`、`make test-web`；因 R9 的风险另跑 chromium 的 m5-content（先 `make build`，收据 `t3-m5-content`）。提交 `feat(M7/P5b): 收集箱详情的描述图片——前端的写权限照收集箱的规则（一个谓词，详情与内容视图都读），确认的上传与删除有提示`。

### T4 附件与链接（spec §2.3，R10、R11）

**Files**：新 `IN/content/widgets.tsx` 与 `widgets.test.tsx`；`IN/content/detail.tsx`（渲染它，传可改与否）；`detail.test.tsx`（允许清单：加替身）；`K`（`widgets.tsx` 收入 `project-address-route`，带 hit 样例）。

**Steps**：
- [ ] Step 1：先写测试，看到失败：
  - 附件：可改时有选择器，不可改时只读（谓词的四种不可改各一格：非提交者成员、已拒绝、关着、已归档）。
  - 链接：管理员与非提交者的成员在待处理、暂缓、拒绝、重复、已接受、收集箱关着各一格都有 “Add link”（以 `issue.add.link` 为名）；访客没有按钮、看得到列表；已归档没有按钮；`link_count > 0` 时才有链接的折叠栏（同 M4）；按下 “Add link” 打开 `IssueLinkCreateUpdateModal`，提交走 `POST /api/v0/issues/{id}/links`。
  - 没有 “Add sub-work item”、“Add relation” 的按钮。
- [ ] Step 2：实现到通过；`K` 的规则收入 `widgets.tsx`，跑 `node tools/keywords.mjs`。
- [ ] Step 3：变异照上表 T4 各行。门禁同 T3（不另跑页面配置）。提交 `feat(M7/P5b): 收集箱详情的附件与链接——附件照收集箱的写权限，链接照合同的 issue.update（任何状态，访客只读）`。

### T5 隐去状态、浏览页的重定向、“打开”只给已接受的（spec §2.4，R13–R15）

**Files**：`C/components/issues/peek-overview/properties.tsx`、`C/components/issues/issue-detail/sidebar.tsx`；新 `C/components/issues/intake-state.test.tsx`（两处的格）；`A/browse/[workItem]/layout.tsx`；新 `IN/address.ts`；`IN/modals/create-modal/create-root.tsx:146`（改读 `intakeAddress`，字符串不变，预检 m9）；新 `C/components/browse/intake-redirect.test.tsx`；`IN/content/header/use-intake-actions.ts`（`offeredOn`）、`IN/content/header/root.tsx`（“打开”按 `offered`）；`header.test.tsx`、`E/stories/collaboration/h10-intake-triage-page.spec.ts`（允许清单）。

**Steps**：
- [ ] Step 1：先写测试，看到失败：
  - 浮层与侧栏：待处理、暂缓、拒绝、重复时没有状态一栏；已接受、`intake` 为 `null`、只持有行时有。
  - 重定向（以 `createMemoryRouter`，先例 `detail-route.test.tsx`）：待处理与暂缓 → `/acme/projects/<p>/intake?currentTab=open&inboxIssueId=<id>`，拒绝与重复 → `currentTab=closed`；历史是 `REPLACE`（后退不回浏览页）；页头与详情的正文从未挂上（记每次挂载）；收集箱关着照样送；已接受与 `intake` 为 `null` 照旧显示详情。
  - “打开”：页面的动作矩阵照允许清单的新期望；宽页头不再无条件渲染 “Open work item”。
- [ ] Step 2：实现到通过；`use-intake-actions.ts` 不变长（`wc -l`，收据 `t5-wc`）；`create-root.request.test.tsx:160-162` 不用改（跳转的字符串不变）；H10 (page) `:58` 照允许清单改，并在 chromium 跑这份故事（`t5-h10-follow`）。
- [ ] Step 3：变异照上表 T5 各行。门禁同 T3；另跑 chromium 的 m4-detail（`t5-m4-detail`）。提交 `feat(M7/P5b): 通用详情与浮层对收集箱条目隐去状态，浏览页在挂上之前把未被接受的条目以 replace 送到收集箱页，收集箱页的“打开”只给已接受的`。

### T6a 动作模型的纯搬移与选中的端口（spec §2.5，R18）

两个提交，各自全绿；两步之间跑全部既有测试。

**Files**：
- Step 1：新 `IN/content/header/intake-actions.ts`；`use-intake-actions.ts`（搬走 `:32-133`）；四个读者的导入（`header/root.tsx:30`、`mobile.tsx:21`、`content/root.tsx:37`、`header.test.tsx:17`）。
- Step 2：新 `IN/content/selection-port.ts`；`IN/content/root.tsx`（`selection` 取代四个 prop）；`IN/content/header/{root,mobile}.tsx`（导航从端口来）；`use-intake-actions.ts`（收端口、不读 `useIntakePageState`）；`IN/root.tsx`（建页面的端口）；`header.test.tsx`、`content/root.test.tsx`（允许清单的编译跟随）；新 `IN/content/header/selection-port.test.tsx`（端口的格单独成文件，`header.test.tsx` 只编译跟随）。

**Steps**：
- [ ] Step 1（纯搬移，`refactor`）：类型、文案表与 `offeredOn` 逐字搬进 `intake-actions.ts`，只改导入；“逐字”之外只许给 hook 要导入的 `OpenModal`（`:61`）、`offeredOn`（`:111`）与 `IntakeModal`（`:58`，hook 读到时）加 `export`（预检 m8）。收据：搬移前后逐行相同、只差这几处 `export`（`t6a-move-identical`）、`make lint-web`、`IN/` 全部测试一行未改且全绿（`t6a-move-tests-untouched`）、`make test-web`、`wc -l`（`t6a-move-wc`）。若搬出的文件读项目，收入 `project-address-route`。提交 `refactor(M7/P5b): use-intake-actions.ts 按关注点纯搬移——动作的种类、文案表与 offeredOn 搬到 intake-actions.ts，hook 留下`。
- [ ] Step 2（端口，`refactor`，行为不变）：先写测试，看到失败：动作模型在没有 `IntakePageStateProvider` 的树里不抛错；邻行来自端口的 `neighbour`（端口给的邻行与页面状态不同的一格）；页面的端口下页头照旧有上一条、下一条与窄屏开关。实现：`InboxContentRoot` 收 `selection`；页面建 `{ kind: "page", leave: landing.leave, neighbour, region: landing.detail, navigation }`；（**实施中改**，T6a 的裁定：T6a 的端口不含 `kind`，它与嵌入的一种在 T6b 随读者一起加；`neighbour(id)` 收工作项的 id，`navigation(screen)` 是页面给的函数，spec §3 的补记）动作模型的 `neighbour` 只读端口。既有测试只为形状编译跟随（允许清单），断言不改，全绿。变异照上表 T6a。门禁同 T3；死成员整份比对（`t6a-dead-after`）。提交 `refactor(M7/P5b): 收集箱的详情区域收一个选中的端口——邻行、落点与导航由页面给，动作模型不再读页面状态`。

### T6b 通知嵌入与通知页的落点 → **检查点 1**（spec §2.5，R16、R17、R26、R27、R30）

**裁定**：O-1 乙、O-2 甲（由 R26 细化）、O-4 甲（spec §8）。

一个提交（knip 的第一个读者规则：嵌入端口的读者是 `WN/root.tsx`，两侧拆不成两个提交；同 re-94 m11），写成两个可分别审查的步骤，各有自己的 RED 收据，审查分两次看（预检 m14）。

**Files**：
- 收集箱一侧（Step A）：`IN/content/header/{root,mobile,use-intake-actions,intake-actions}.ts(x)`（嵌入的关闭按钮、分诊之后不移、`openLink`、`offeredOn` 按 `kind`）；`IN/content/root.tsx`；新 `IN/content/root.embed.test.tsx`。
- 通知一侧（Step B）：新 `C/components/issues/peek-overview/use-peek-issue-read.ts`（读的 hook；`peek-overview/root.tsx` 的 `PeekContent` 改调它；收入 swr 名单）；`WN/root.tsx`（判定、锁定与代、失败界面、嵌入的端口、预览区域的 ref 与 `tabIndex={-1}`）；新 `WN/landing.ts`（通知页的落点）；`WN/sidebar/list.tsx`（两个区域元素 `:53`、`:65` 加 `data-notification-list`）；新 `WN/root.embed.test.tsx`、新 `WN/landing.test.tsx`；`WN/root.test.tsx`（允许清单）；`E/stories/collaboration/h2-h4-notification-cards-page.spec.ts`（允许清单，O-4）。

**Steps**：
- [ ] Step A（收集箱一侧）：先写 `IN/content/root.embed.test.tsx`（以真实的动作模型、嵌入的端口），看到失败（收据 `t6b-red-inbox`）：没有上一条、下一条与窄屏开关；有以 `close` 为名的按钮，按下调 `selection.close`；管理员分诊（拒绝、暂缓、接受）确认之后 `leave` 没被调、焦点在新页头的主动作或嵌入的区域；删除确认之后 `leave` 被调；“打开”读 `openLink`：对待处理链到 `intakeAddress`（`currentTab=open`），对拒绝链到 `currentTab=closed`，对已接受链到浏览页；复制照旧是浏览页的地址。实现到通过（收据 `t6b-green-inbox`）。
- [ ] Step B（通知一侧）：先写测试，看到失败（收据 `t6b-red-notifications`）。`WN/root.embed.test.tsx` 用真实的读 hook 与真实的 SWR（`SWRConfig` 的缓存，同 `WN/root.test.tsx:21` 的写法），只替身取数与浮层组件：
  - 判定：调用者有角色、预览的读扣住时说在加载、没有浮层也没有嵌入；读落定为待处理、暂缓、拒绝、重复 → 嵌入（`InboxContentRoot` 收到 `kind: "embed"` 的端口），`setPeekIssue` 没被调；不在收集箱、`intake` 为 `null`、已接受 → `setPeekIssue` 与浮层；SWR 对这个键留着较早的回答（拒绝的）而这次的读答已接受 → 浮层；嵌入之后 canonical 变成已接受 → 仍是嵌入。
  - 只一个读：非收集箱的预览，浮层的替身调真实的读 hook，取数恰一次。
  - 锁定（预检 I4）：越过去重窗口（推进假时钟 2 秒以上）之后发 `online`，预览不再取数，浮层（或嵌入）是同一个元素、替身里输入的字还在；浮层的重试（替身调读 hook 的 `mutate()`）不卸下浮层。（**实施中改**，T6b 的修正 `791c29ce`：判定之后判定的组件卸下、读离开这个键，越过去重窗口之后发 `online`，普通浮层的 hook 照别的页面再读、浮层与输入还在，嵌入不读；spec §2.5 的修正）
  - 代（预检 m10）：换了权限身份（或 `loginId`）之后有新的读、以它重新判定。
  - 读失败（O-1 乙）：预览画 `IssuePeekOverviewError`、浮层没有挂上；重试再读，落定之后判定；失败界面的“关闭”经落点清选中。
  - `WN/landing.test.tsx`（R26）：`leave` 清选中，焦点在那次提交之后一个微任务里落：卡片在 → 它的主按钮；卡片不在、列表在 → 列表的区域；列表没画出来（读失败、读着）→ 预览的区域；从不落到 `document.body`；选中已换成别的通知时 `leave` 不动。
  - 普通浮层的关闭（O7）：浮层的 `embedRemoveCurrentNotification` 经落点，焦点到那张卡片的主按钮。
  实现到通过；`WN/root.test.tsx` 与 H2–H4 (page) 照允许清单改（同一提交）；`make build` 之后在 chromium 跑 H2–H4（`t6b-h2-h4-follow`）。
- [ ] Step C：变异照上表 T6b 各行（收集箱一侧与通知一侧分开记）。门禁同 T3；死成员整份比对（`t6b-dead-after`：R32 的两个 prop 以端口回来，不新出死行）。提交 `feat(M7/P5b): 通知预览读完工作项之后嵌入收集箱的内容——未被接受的（含拒绝）在嵌入里分诊，已接受的仍是浮层；判定锁定，不随共用键的再验证撤下；关掉嵌入、失败界面或浮层之后焦点回到那张卡片`。
- [ ] Step 4（**检查点 1**：除 T8c 之外的产品代码完成）：`final.sh cp1`；页面上核实非收集箱的通知只一个 `GET /api/v0/issues/{id}`（以 chromium 聚焦跑 P4 的 `h2-h4-notification-cards-page.spec.ts` 并读请求，收据 `cp1-peek-reads`）：多了一个先看 spec §2.5 的时间窗，再报主控（B-16）；（**实施中改**：一次取数有两个 `GET`，普通浮层 2 个、收集箱条目 4 个，T6b 的 `t6b-peek-reads-probe-h20-2` 与修正之后的 `t6b-fix-peek-reads-probe-h20` 核过；判定的读被浮层的挂载读并入，没有多出；spec §2.5 的修正）主控推送、核分支 CI，记 web 作业三步与 e2e 两片的时长。

### T7 H8 (page)（spec §2.8，R7、R8）

**Files**：新 `E/fixtures/editor-chunks.ts`；`E/stories/issue/editor-loading.spec.ts`（导入）；新 `E/fixtures/intake-create.ts`；新 `E/stories/collaboration/h8-intake-submit-page.spec.ts` 与 `h8-intake-close-page.spec.ts`（两份合起来不过 400 行可合为一份）。

**Steps**：
- [ ] Step 1（搬移，`test` 提交）：`editorChunks` 与 `holdEditorImplementation` 搬进 `editor-chunks.ts`，两者都导出，构建产物的相对路径少一层（`../../web/…`），别的逐字（预检 m8）；M4 的故事改为导入、断言不改；`make build`，以 m4-detail 配置三引擎跑 `editor-loading.spec.ts`（`t7-move-editor-loading`）；e2e 的类型与 lint 过。提交 `test(M7/P5b): 编辑器代码块的扣住从 M4 的故事搬到夹具`。
- [ ] Step 2：写故事（spec §2.8 的两行），标题都带 `H8 (page)`：
  - 提交：管理员、成员、访客各一次，断言请求体精确、数据库的条目行（`E/fixtures/assert/intake.ts`）、地址选中新条目、列表与侧栏的数随之（`expect.poll` 内容，不数计数的请求）；名字的被拒（超长、空的、只有空格的：不发请求、焦点在名字、`aria-invalid`）；收集箱关掉之后提交 → 提示、表单留着；“Create more”。
  - 关闭（预检 I2）：**页面载入之前**登记 `holdEditorImplementation`，收集箱是空的（详情不挂编辑器）；每个关闭与提交的步骤之前 `expect.poll(() => held.length).toBeGreaterThan(0)` 且弹窗里没有 `[contenteditable="true"]`。Escape 关空表单；再开、Discard 关；有字时 Escape → 确认框（`toHaveAccessibleName`）→ 取消、字还在 → Escape → 确认 → 关、焦点回 “Add work item”，再开是空的；扣住时提交 → 提示、没有 `POST`；放开代码块之后点背景（没有改动）关；`holdAnswer` 扣住 `POST` 时 Escape 与点背景关不掉、Discard 停用，Escape 之后、点背景之后、被拒之后焦点都在弹窗里（`dialog.evaluate((d) => d.contains(document.activeElement))`，R28；每个引擎落在哪个控件记台账）；负责人与截止日的浮层开着时 Escape 只关浮层；负责人下拉框以键盘：Enter 打开、`expectPopupBesideButton`、搜索取得焦点、Escape 回到按钮；Tab 的自然次序（`tabStops` 从名字到提交，编辑器是否多出停点照实记）。三引擎各有的差异写在台账。（**实施中改**，T7 的主控裁定：页面载入之前扣住代码块的测试只在 Chromium、Firefox 跑，WebKit 以写明原因的 `test.skip` 跳过，有改动时的确认放进编辑器已建成的测试、三引擎核；Tab 的次序多一个 “Continue writing”，描述带 `tabIndex={-1}`，作为故事之前的一个 `fix` 提交 `ec6b91a5`；spec §2.1、§2.8 的修正）
  - 截止日按浏览器的“今天”（`browserDay`）选。
  红色版本：关闭的故事去掉扣住之后跑一次，“提交被拒、没有 `POST`”一步失败（`t7-red-page-hold`）；提交的故事先跑一个期望错误的版本（`t7-red-page-submit`）。再改正。
- [ ] Step 3：`make build build-s3-fixture`；聚焦 chromium（`cd e2e && NERVE_VERSION=0.1.0-dev pnpm exec playwright test --project=chromium stories/collaboration/h8-intake-*.spec.ts`）；`--config=playwright.m7-pages.config.ts` 三引擎；`--repeat-each=3`（chromium）；台账记每个 `test()` 各引擎的时长；`make lint-web`。提交 `test(M7/P5b): H8 的页面故事——三种角色经新建弹窗提交，关闭政策在编辑器代码块扣住时与有改动时、在途的焦点，嵌套 Escape 与下拉框的键盘`。

### T8a H11 (page) 的资产：图片与附件（spec §2.8，R12）

**Files**：新 `E/fixtures/intake-files.ts`（`intakeWorld(world.api, info)` 的四个人在 `world.page` 上登录、传输的核对：本机文件或桶里的对象，从 `content-files-page.ts` 导入共用的帮手，不改它）；新 `E/stories/collaboration/h11-intake-images-page.spec.ts`、`h11-intake-attachments-page.spec.ts`，都经 `pageFileStory`（`[local]`/`[s3]`）。

**Steps**：
- [ ] Step 1：写故事（spec §2.8 的两行），标题带 `H11 (page)`：
  - 图片：访客提交者加图片（`pasteInto` 或选文件），`PATCH /api/v0/intake-issues/{id}` 的描述带资产 id，字节相同；非提交者的成员只读、没有上传；在途被拒（预检 m7）：扣住 `POST /api/v0/assets` 的回答，另一会话经接口拒绝条目，放开之后 `POST /api/v0/assets/{id}/complete` 答 409 `issue.intake_closed` 并提示，这个 409 写进 `watch.apiFailures` 的期望；经接口接受之后**在浏览页上**（`/browse/…`）访客提交者只读、成员可传（预检 m5）。
  - 附件：访客提交者加附件、`ConfirmedContentNotice` 说出、下载字节相同（`downloadedExactly`）；非提交者的成员没有选择器；扣住 `createAsset` 的回答、经接口关掉收集箱、放开之后 `completeAsset` 答 409 `issue.intake_disabled` 并提示（写进期望）；传输中取消；传输在途时换选中（另一行），放开之后迟到的结局不写进新条目（新条目的附件数与列表不变）；删除附件，本机文件或桶里的对象没了；清理任务触发不了时在故事的注释与台账写明由 `E/stories/files/f15-cleanup.spec.ts` 覆盖。（**实施中改**，T8a：删除之后本机文件与桶里的对象留到清理任务，M5 F14 的合同，物理清除由 F15 覆盖；故事断言文件行软删、读答 404、对象还在；spec §2.8 的修正）
  - 每个“在途”都以扣住回答造出（`holdAnswer`），不靠时序。（**实施中改**，T8a 的主控裁定：传输中取消与换选中扣的是上传字节的 `PUT`（新夹具 `holdUploads`），仍是扣住、不靠时序；两处被拒照旧以 `holdAnswer` 扣 `createAsset` 的回答；数据库的读追加在 `E/fixtures/assert/intake.ts`，提交多一个文件；spec §2.8 的修正）
  先跑期望错误的版本（`t8a-red-page`）。
- [ ] Step 2：同 T7 的跑法；`[s3]` 在 Firefox 或 WebKit 里若因 M5 的上传件失败，停下报告（跨模块的发现，spec R12）。台账记各引擎、各变体每个 `test()` 的时长。提交 `test(M7/P5b): H11 的页面故事（收集箱宿主上的图片与附件，local 与 S3，三种引擎）`。

### T8b 链接与详情的负责人下拉框（spec §2.6、§2.8，R10、R21）

**Files**：新 `E/stories/collaboration/h11-intake-links-assignees-page.spec.ts`（约 120–180 行，T8c 在它的下拉框那个 `test()` 里接着加连点的一步）。

**Steps**：
- [ ] Step 1：写故事，标题带 `H11 (page)`：
  - 链接：非提交者的成员在已拒绝的条目上加、改、删链接（请求与数据库）；访客看得到列表、没有按钮。
  - 详情的负责人下拉框（管理员、待处理的条目）以键盘：Enter 打开、`expectPopupBesideButton`、搜索取得焦点、Escape 关上并回到按钮。
  先跑期望错误的版本（`t8b-red-page`）。
- [ ] Step 2：三引擎、`--repeat-each=3`。提交 `test(M7/P5b): H11 的页面故事（链接照 issue.update、详情的负责人下拉框的键盘）`。

### T8c 负责人集合在轮到时由增删算（spec §2.6，R19、R20，**高风险，M4 的共用写层**）

产品代码，在检查点 1 之后（spec R1）。**动代码之前**：把方案写进台账 `progress.md` 的一段（类型与帮手的名字与位置、`changeBody` 的新签名、七个调用方各怎样算增删、调用方的格放在哪），报主控，主控看过再写代码。

**Files**：`web/packages/types/src/issues/issue-change.ts`（负责人的增删，与 `assignee_ids` 类型上互斥）；`C/store/issue/issue-write-fields.ts`（`changeBody(change, held)`，增删在轮到时并到所持的集合上）；`changeBody` 的三个调用方：`C/store/issue/issue-writes.ts:143`、`C/store/issue/helpers/base-issues.store.ts:94`、`C/components/power-k/ui/pages/context-based/work-item/writes.ts:45`；七个调用方：`IN/content/issue-properties.tsx:83`、`C/components/issues/peek-overview/properties.tsx:93`、`C/components/issues/issue-detail/sidebar.tsx:97`、`C/components/issues/issue-layouts/properties/all-properties.tsx:73`、`C/components/issues/issue-layouts/spreadsheet/columns/assignee-column.tsx:32`、`C/components/issues/issue-detail-widgets/sub-issues/issues-list/properties.tsx:216-219`、`C/components/issues/relations/properties.tsx:52-56`；`issue-write-fields.test.ts`（允许清单的编译跟随）；新的格（`C/store/issue/issue-write-assignees.test.ts` 与调用方的格，位置照方案）；`E/stories/collaboration/h11-intake-links-assignees-page.spec.ts`（连点的一步）。

**Steps**：
- [ ] Step 1（页面上重现）：在 T8b 下拉框的那个 `test()` 里接着写连点：管理员点第一位成员，扣住它的 `PATCH /api/v0/intake-issues/{id}` 的回答（`holdAnswer`），点第二位，放开；断言两次请求体（第二次含两位）与数据库里的最终集合（两位）。`make build`，在没修的构建上 chromium 跑，看到它在第二次的请求体或最终集合处失败（收据 `t8c-red-page-q34`，这就是重现的证据，写进台账）。不扣住的“快速连点”只能靠时序，不作证据，不写。
- [ ] Step 2（单测先红）：`changeBody` 把增删并到所持的集合上（保留次序、不认识的 id 留着、加已有的与删没有的都不变）；以假的队列（`@/store/fake-queue` 的 `inTurn`、`sent`）：排在一个负责人写之后的第二次点击，它的请求体在轮到时从所持的集合算、含第一位；收集箱详情、浮层、详情侧栏三个调用方各一格（点一位成员，交给写的层的是相对显示的值的增删），其余四个调用方的格照方案（至少靠类型：`assignee_ids` 与增删互斥）。看到失败（收据 `t8c-red-unit`）。
- [ ] Step 3：实现到通过；`issue-write-fields.test.ts` 照允许清单编译跟随。连点的一步三引擎、`--repeat-each=3` 全绿；先 `make build`，chromium 的 m4-detail（`t8c-m4-detail`）与 m4-list（`t8c-m4-list`）。
- [ ] Step 4：变异照上表 T8c 各行（`t8c-mut-assignees-at-call` 也在页面的连点一步上看它失败，收据 `t8c-mut-page-q34`）。`make lint-web`、`make knip`、`make test-web`；`wc -l` 写的层三个文件（各不过 220）；死成员整份比对（`t8c-dead-after`）。提交 `fix(M7/P5b): 负责人多选的每次修改以增删交给写的层，在轮到时并到所持的集合上（请求体仍是整集）；七个下拉框的调用方相对被点时显示的值算增删，连点不再丢掉前一位`。

### T9 嵌入的故事、重定向 → **检查点 2**（spec §2.8）

**Files**：新 `E/stories/collaboration/h10-intake-embed-page.spec.ts`（过 400 行时重定向一步另拆 `h20-intake-redirect-pages.spec.ts`）；夹具按需（新夹具，`intake-pages.ts` 不加长）。

**Steps**：
- [ ] Step 1：写故事（spec §2.8 的嵌入一行），标题带 `H10 (page)`，重定向一步带 `H20 (page)`：
  - 点开卡片之前登记判定的读与嵌入自己的读的 `waitForResponse`，两个都答完再动作；断言收集箱条目恰两个 `GET /api/v0/issues/{id}`，普通工作项的通知恰一个。（**实施中改**：4 个与 2 个，删除之后多一个 404；spec §2.5 的修正）
  - 嵌入里已有的标签与负责人显示出来；接受弹窗的状态有默认值（预检 O5）。
  - 管理员在嵌入里拒绝、暂缓、接受各一次（三个条目）：选中不移、通知列表不变、嵌入显示新的状态；焦点的落点（键盘一次、指针一次）；800 宽一次。
  - 成员与访客只读的页头；已拒绝的也嵌入；已接受的开浮层（“Close the peek view”），关上之后焦点在那张卡片的主按钮（O7）。
  - 没有上一条、下一条；↑↓ 不换条目；嵌入的菜单与负责人下拉框在按钮旁（`expectPopupBesideButton`）。
  - “Close”（限在 “Preview” 区域里定位）清选中、焦点在卡片的主按钮；删除之后清选中、同上；先换页签（卡片不在）再关，焦点在列表的区域（R26，预检 I3）；都不是 `body`。
  - “打开”：待处理的到收集箱页并选中它。
  - 重定向：打开待处理条目的浏览地址 → 收集箱页、选中它、“未处理”页签；拒绝的到“已处理”；`page.goBack()` 不回浏览页。先等布局的读答完再断言。
  先跑期望错误的版本（`t9-red-page`）。
- [ ] Step 2：同 T7 的跑法与时长记录；提交 `test(M7/P5b): 嵌入的页面故事、通知页的落点与浏览页的重定向`。
- [ ] Step 3（**检查点 2**）：`final.sh cp2`（七项加六个页面配置，各留 JSON 报告）；新文件 `--repeat-each=3`（`cp2-repeat`）；照 P4 spec §2.11 估出 e2e 两片，超过 30 分钟先报告、不推送；主控推送、核分支 CI，记两片的实际时长与 web 作业三步。

### T10 文档与交接（主控）

- **plane-diff**：spec §7 的八行进 `docs/v0/plane-diff.md` 收集箱一节（`:358-384`），第 7 条补进 P2c 的 `:369` 一行；Plane 的行号对照主目录的只读 `plane/` 逐一核过（预检只抽了 10 处）。
- **frontend-changes**：3.1 的 M7 一行写 P5b（收集箱一侧完成，`getTabIndex` 只剩视图）；3.2 加行：新建弹窗的关闭政策（在途不给 `handleClose`、迟到的跟进）、`canWrite` 的收集箱一支（R9）、附件与链接、通用面隐去状态与浏览页的重定向、通知嵌入、判定锁定与选中的端口、通知页的落点（浮层的关闭也走它）、负责人集合在轮到时算（R20）。
- **交接**：spec §5 的表，每条写“处理结果（M7/P5b）”；写关闭的照写关闭，其余保持 open；M7-04 逐项写本段处理的与留下的。
- **死成员**：重跑脚本（收据 `t10-dead-final`），剩下的行写进阶段记录。
- **告警**：核对本段改到与新建的文件 oxlint 为零、计数等于上限（`t15-oxlint-files.py` 的同类脚本，改名 `t10-…`）。
- **设计**：§12 的 P5b 一行链 spec、§14 的 P5b 一行（主控）。
- **阶段记录** `reviews/P5b-web-intake-files-embed-review.md`：§1 方法与范围、§2 验收 → Task、§3 实施中的裁定（R9、R20 写明是动了别的 M 共用代码的高风险局部裁定；R26 改了 P4 通知页上浮层关闭的焦点）、§4 向负责人说明（P5b 之后 main 的样子；嵌入的读是两整包的代价，预检 O1）、§5 门禁与 CI、§6 时长、§7 交给 P6a 与 M8、§8 文档、§9 审查与集成；`HO/M3-closeout.md:64` 要的“浏览器核对”逐项写在 §8。
- 检查文档链接与关键词，提交 `docs(M7/P5b): …`。

### T11 门禁、整审与集成（主控）

1. **整分支审查**：一次 fresh 只读的整分支 deep-review（fable），重点是 Review Focus 的 1–4。Critical/Important 在一次修复轮里各有 RED/GREEN；Minor 记台账，主控裁定改否。之后一次限定复审。
2. **终检**：`final.sh fin`（六个页面配置）。
3. **推送前的估算**：照 P4 spec §2.11 估出 e2e 时长。
4. **最终分支 CI**：推送准确的 HEAD（`git rev-parse`），核分支 CI（`wait-run.sh`）。
5. **合并**：`--no-ff` 合入 main，推送后以 `ls-remote` 核对。
6. **main CI**：核准确的 main CI。
7. **归档与清理**：归档 `.superpowers/sdd/M7-P5b-integration-<日期>`（`archive.py`），清理 worktree 与分支。
8. **回填**：spec A10 与设计 §14 由下一段（P6a）的 T1 回填。

## Review Focus

1. **R9（高风险，M5 的共用件）**：`canWrite` 只改 `issue` 宿主、只在 `inIntake` 时；谓词一处、详情与内容视图在收集箱里的四种状态同答，已接受分别断言；`new_comment`、`comment`、`draft` 不变；`canRemoveAttachment` 随之（删除的命令 `content-deletion.ts:45`）；访客提交者能传、非提交者成员与已处理、关着不能；m5-content 三引擎不回归；页面上被别的会话改了状态时由 nerve 拒（`completeAsset` 的 409）、提示说出，不重读是已知的行为（交 M8）。
2. **R20（高风险，M4 的共用写层）**：方案先给主控看；增删在轮到时并到所持的集合上、请求体仍是整集；七个调用方相对被点时显示的值算增删，`assignee_ids` 与增删互斥；拖放与命令面板照旧（已在轮次里算）；连点的一步先在没修的构建上红；m4-detail 与 m4-list 不回归。
3. **关闭政策（R4、R5、R28、R29）**：只经 `requestClose`；在途不给 `handleClose`、Escape 不动焦点、点背景与被拒之后焦点在弹窗里；引用为 `null`、已建成未就绪、有改动、没有改动；“有改动”的规范化（名字去空白、描述按 `isEmptyHtmlString`）；成功直接关；迟到的跟进只在表单还挂着时；文档级 Escape 全部删除；IME；嵌套 Escape 只关最内层；提交仍要求就绪；名字按 `Array.from(trim)`、没有 `required`、提交不停用、被拒的焦点与 `aria-invalid`；确认框的按钮文字经 `t()`；再开从初值起；H8 的扣住真的扣住了（`held.length > 0`）。
4. **嵌入与通知页（R16–R18、R26、R27、R30）**：纯搬移逐行相同（只多几处 `export`）；端口的提交行为不变、动作模型不读页面状态；读的 hook 在自己的文件、预览的读关重连重读；判定锁定、按代记、以这次的读为准；非收集箱的通知只一个读（时间窗写明）；`setPeekIssue` 只给浮层；失败界面是预览的；嵌入里分诊不移选中、焦点照“选中没有移”；落点链的三环、在 Headless 还焦点之后落；嵌入、失败界面与浮层的关闭都走它；没有导航与 ↑↓；`openLink` 与 `link` 各有读者；R32 的对应。
5. **重定向与“打开”（R14、R15）**：在布局、挂上之前、`replace`、按状态选页签、关着照样送、已接受不送；收集箱的地址一处给出；收集箱页的“打开”只给已接受的，复制不变。
6. **附件与链接（R10、R11）**：不渲染 M4 的 `IssueDetailWidgets`；链接照 `issue.update`、任何状态、访客只读；`widgets.tsx` 在 `project-address-route` 里；`ConfirmedContentNotice` 在宿主之内；H9 (page) ② 的读表不变。
7. **会话分代（v0 §7.7）**：判定的读与嵌入的读的键、`identity`、`loginId`；判定的代；`await` 之后都经 `followInSession`；换选中、换代之后旧的回答不写进新的预览；新建弹窗的跟进限在表单还挂着时。
8. **可访问与拒绝**：弹窗、确认框、关闭按钮、链接按钮的名字；开关不套在按钮里；自然 Tab 次序；`FieldError` 与 `useRefusedFocus`；每个失败都提示、不 `console.error`；英文字面量全部改键、只有两个新键。
9. **删除干净**：死代码、`INTAKE_ISSUE_FORM` 一组、两份测试的替身；P5 的死成员归零；死成员整份比对；告警等于上限。
10. **共用的外壳**：M4 的浮层（只把读的 hook 搬进自己的文件）、详情侧栏、浏览路由、列表行与表格的负责人下拉框；P4 的通知页（落点、列表的标记）；M5 的上传件。检查点与终检的六个页面配置与整套默认 e2e。
