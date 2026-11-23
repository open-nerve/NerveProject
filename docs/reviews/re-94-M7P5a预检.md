# M7 / P5a 预检

> 2026-11-23：结论 **修正后通过**（Ready after fixes）。严重 1、重要 4、次要 11、观察 6。合同在 `89be91e5` 上都做得出来：六个收集箱操作、`IssueDetail.intake`、`IntakeIssue`、`IntakeIssuePage`、`IntakeIssueUpdate` 的七个字段、`IntakeTriage`、`IntakeCounts`、五个问题码与 `must_be_future` 都在，名字与形状和 spec 写的一致；`fetchPart`、`useSessionSWR` 的三个选项、`followInSession`、`refreshAuthoritative`、`confirmCreation`、`getIssueDetailById`、`knownProjectIds`、三处入口的行号都对得上。两条高风险裁定 R3、R4 的方向成立（接受经 `confirmCreation` 走 M4 新建的证据路径是对的，纯搬移做得出，没有测试读私有成员），但接缝写得不完整，五件事要先改：① spec 把收集箱的写指回 `write()` 的 `:108-133`，那一段含目录准备，Triage 状态不在任何目录里，会话里持有一份按状态组的列表时收集箱的修改在发出之前就抛错；② 非接受的分诊不动工作项的 `updated_at`，照回答确认进 canonical 挡不住在途的旧读，而 plan 要“断言照旧”的 `"inbox"` 一格钉的正是写后的重读，与“接受照回答确认”两者不能同时成立；③ 接受弹窗照 Plane 在 `beforeFormSubmit` 里关弹窗、移选中，nerve 的 `formScope` 随之失效，弹窗里改的字段静默不发；④ T6b 的“新建弹窗只改提交、其余不动”做不出（另有三处读旧层、一份测试断言旧地址）；⑤ 标重复与改暂缓成功之后打开者已不在，焦点的落点没有写。未知动态值的通用文案键应保留（合同里 `verb` 是不设枚举的字符串）。
>
> - **目标**：[P5a spec](../v0/M7-collaboration/specs/P5a-web-intake.md)、[P5a plan](../v0/M7-collaboration/plans/P5a-web-intake.md)（未提交的草稿），次要目标台账的 `implementer-brief.md`、`final.sh`、`gates.sh`，以及 T1 未提交的三处改动（设计 §12、§14，P4 阶段记录 §5、§9，`tools/keywords.json` 的 phase）。依据台账 `rulings.md` 与 `research/`，[M7 设计](../v0/M7-collaboration/M7-design.md) §2、§3.5、§3.6、§3.11、§12、§15 A、B，[总体设计](../v0/v0-design.md) §7.7，P2a、P2b spec，先例 [re-93](re-93-M7P4预检.md) 与 [P4 阶段记录](../v0/M7-collaboration/reviews/P4-web-notifications-review.md) §9。
> - **基线**：main `89be91e5`（P4 的 noff），worktree `codex/m7-p5a`。

下文 `C/` = `web/apps/web/core/`，`IN/` = `C/components/inbox/`，`IY` = `api/modules/issue.yaml`，`K` = `tools/keywords.json`。

## 1. 方法

一次 fresh 只读审查（`fable`）：只写本文件，不动 git 状态（只用了 `git status`、`git diff`），不起容器、不跑测试。跑过的只有 `grep`、`sed`、`wc`、`ls`，以及两个只读的检查：`node tools/keywords.mjs`（79 条规则、7 个例外、无命中）与台账的 `doc-links.py`（231 个文档、2302 个链接、0 断）。oxlint、knip、deadsym、vitest、Playwright 都没有运行。

对照代码核了：

- 合同：`IY:744-1088` 的五条收集箱路径全文、`IY:2205-2222`、`:2325-2423`、`:2520-2607` 的 schema、`IssueActivity`（`IY:3188-3222`）、`schema.gen.ts` 的 `listIntakeIssues` 查询、`web/packages/api-client/src/index.ts:31-62` 的 CSV 名单、`C/lib/error-messages.ts:87-122`。
- 写的接缝：`C/store/issue/issue-writes.ts` 全文、`issue.store.ts` 全文、`issue-verification.ts` 全文、`root.store.ts:124-319`、`issue-write-fields.ts`、`helpers/issue-list-binding.ts`、`helpers/issue-list-catalog.ts`、`C/lib/issue-membership.ts`、`C/lib/reconciled.ts`、`issue-details/{issue,comment,subscription,description-versions}.store.ts`、`issue-changes.test.ts:595-712`、服务端 `domain/intake.go:66-215` 与 `app/triage_intake.go:184`。
- 页面：`use-issue-operations.ts`、`detail-host.tsx`、`issue-detail-widgets/use-detail-scope.ts`、`issue-modal/{base,update-command,form-submission}.ts(x)`、`peek-overview/root.tsx`、`IN/` 旧树的行数与外部读者、`IN/content/inbox-issue-header.tsx:100-275`、`IN/modals/create-modal/` 四个产品文件与两份测试、`select-duplicate.tsx` 与测试、`activity/actions/inbox.tsx`、`activity-list.tsx:70-90`、三处入口（侧栏、页签栏、命令面板）、`C/lib/planning-switch.ts`、`use-session-swr.ts`、`in-session.ts`、`notification.store.ts:100-125`、`project.store.ts:244-251`、`permissions.store.ts:60-75`。
- 门禁与 e2e：`knip.jsonc`、`.oxlintrc.json:160-168`、`web/apps/web/package.json:10`、`K` 的 `old-notification-urls`、`copy-hook` 例外与 `until`、`e2e/playwright.m7-pages.config.ts`、`e2e/fixtures/mounts.ts`、`e2e/stories/collaboration/` 的文件名、`final.sh`、`gates.sh`、P4 归档台账的末行与 `dead-members/`。

没有核实的：

- oxlint 的 31 条、死成员的 39 行、knip 在删除之后的结果、任何 vitest 与页面故事的时长（都没有运行）。
- 个人主页的列表默认是否按状态组分组（只确认它是可选的分组，`web/packages/constants/src/issue/filter.ts:137`；S1 对“带状态组筛选的项目列表”同样成立）。
- 服务端对 `target_date: null` 的提交是否宽容（I4 只按合同“不给即无”与生成的类型判断）。
- 权限的 `identity` 是否随 `guest_view_all_features` 变化（m4 的情形因此只写成“可能”）。
- spec 里约 150 处行号只抽查了约 60 处，都对得上；`plane/` 的行号没有核。

## 2. 发现

### 严重

| # | 位置 | 发现 | 证据 | 须做的修正 |
|---|---|---|---|---|
| S1 | spec §2.4“端点与确认的去处”的表与“收集箱的确认不写项目列表”“权限与换代：照 … `write` 既有的 …（`:108-133`、`:148-160`），不另写”；plan T5 Step 2 的格与变异 `t5-mut-intake-into-project-lists` | **收集箱的去处只写了“不调 `acceptConfirmed`、不 `invalidateProjectReads`”，漏了 `write()` 的两次目录准备；Triage 状态不在任何目录里，于是收集箱的写在发出之前就抛错。** `write()` 在发出之前对每份覆盖这个项目、已有值的列表 `await list.prepareCatalog([initial])`，确认时再来一次。`prepareListCatalog` 在列表用到状态时（按状态组分组或子分组、按状态名排序、带状态组筛选）先重读项目的状态，目录里仍没有这个工作项的 `state_id` 就抛 `Required issue directory is unavailable`。状态接口从不列 Triage。列表在这一代里只登记、不注销，工作区级的列表覆盖工作区里每个项目。场景：用户看过一份带“状态组”筛选的项目列表，或按状态组分组的个人主页，再到收集箱改标题：`PATCH /intake-issues` 没有发出，提示一条通用错误，每次重试多一个 `GET /projects/{id}/states`；版本恢复、拒绝、暂缓、标重复、删除只要经 `write(id, …)` 都一样。删除若走确认那一段，错误被接住后是 `list.markUnavailable`：项目列表整份标成失败。spec 引的 `:108-133` 恰好包含第一段准备（`:111-123`），照字面实现就会带上。plan 的格“一份持有的项目列表不变”用普通列表，查不出。接受不受影响：`write(undefined, …)` 没有 `initial`，确认时只准备 `after`，它已在默认状态。 | `C/store/issue/issue-writes.ts:110-123`、`:161-180`；`C/store/issue/helpers/issue-list-catalog.ts:85-94`、`:108-112`、`:138-148`；`C/services/project/states.service.ts:9`；`C/store/issue/root.store.ts:225-228`（只有 `registerList`）；`helpers/issue-list-binding.ts:62-69`；`web/packages/constants/src/issue/filter.ts:131-148` | spec §2.4 写明收集箱的去处（内容修改、四种非接受的分诊、删除）**不经 `write()` 的两段目录准备**，只要它的权限水位（`:107-109`、`:124-133`、`:147-160`）与 canonical、详情、收集箱列表的确认；“照 `:108-133`”改成不含 `:110-123` 的说法。plan T5 加格：持有一份带 `state_group` 筛选的项目列表与一份按状态组分组的工作区列表时，收集箱的内容修改、拒绝、删除各发出并确认，没有 `GET /projects/{id}/states`，两份列表的值与 `error` 不变；变异 `t5-mut-intake-prepares-catalog`（收集箱的去处也准备目录）。brief 同改。 |

### 重要

| # | 位置 | 发现 | 证据 | 须做的修正 |
|---|---|---|---|---|
| I1 | spec §2.4 的表（四种非接受的分诊确认进“收集箱、计数”）与“`issue-changes.test.ts` 的 `"inbox"` 一格 … 改为新的 `triageIntake`，断言照旧”；plan 允许清单“两条 `it.each` 的断言不改”、T5 Step 2 | **两件事合成一个洞。** (a) 非接受的分诊不改工作项（合同“Any other triage changes no work item”，服务端在 `IntakeAccepted` 之外直接返回），`updated_at` 不动。canonical 只按 `updated_at` 排先后：相等的读照样写入。于是一个在分诊之前发出、在确认之后到达的详情读（同一 `updated_at`、`intake` 仍是待处理）把 canonical 的 `intake` 改回去，页头又给出分诊按钮，再点得 409。P4 的关注正是为这个原因不照回答写 `is_subscribed`，而是在队列的轮次里 `refreshAuthoritative(id, { afterWrite: true })`（它把新鲜度的下限提到写之后，之前发出的读被拒）。(b) plan 要保留断言的那一格是**三条** `it.each`，不是两条，它们钉的是“写成功之后发第二个 `GET /api/v0/issues/x`，写之前发出的读到达时不被采纳”“被拒时不发写后的读”“写后的读在途时换代即停”。接受若只经 `confirmCreation` 照回答确认，没有第二个 GET：第一条在 `expect(reads).toHaveLength(2)` 失败，第三条等不到写后的 GET。“接受照回答确认、不重读”与“断言照旧”不能同时成立。 | `IY:945-946`；`server/internal/modules/issue/app/triage_intake.go:184`；`C/store/issue/issue.store.ts:28-34`、`:50-53`、`:60-76`；`C/store/issue/issue-details/subscription.store.ts:7-15`、`:55-64`；`C/store/issue/issue-verification.ts:64-66`、`:165-174`；`C/store/issue/issue-changes.test.ts:605`、`:659-684`、`:685-695`、`:696-712` | 主控二选一并写进 spec §2.4、R3、R4：**(甲，建议)** 五种分诊在它的轮次里、回答确认之后都再读一次（`refreshAuthoritative(id, { afterWrite: true })`）：列表（收集箱的，接受时另加项目列表经 `confirmCreation`）照回答确认，canonical 的最终值来自这次重读；三条断言照旧成立，(a) 一并解决，代价是每次分诊多一个 GET。**(乙)** 接受不重读：那三条里 `"inbox"` 的一格须改写，改法逐条写进允许清单（这是改 M4 的回归断言），并另给非接受的分诊一个挡旧读的办法。两种都加格与变异：分诊之前发出的读在确认之后到达（同一 `updated_at`、`intake` 为待处理），canonical 的 `intake` 仍是分诊之后的（`t5-mut-triage-no-floor`）。plan 的“两条”改为三条。取 (甲) 时一并核对：重读在途时 `getIssueById` 给 `undefined`，详情的正文连同画在里面的弹窗会被撤下（`C/components/issues/detail-host.tsx:28-45`），接受弹窗须画在不被撤下的一层（见 I2）。 |
| I2 | spec §2.8“接受”与“之后的选中（Q30，R14）… Plane `inbox-issue-header.tsx:116-135`”；plan T8 的接受与选中两组格 | **照 Plane 的接法，弹窗里改的字段静默不发。** Plane 的 `handleInboxIssueAccept`（今天仍在 `IN/content/inbox-issue-header.tsx:136-140`）在 `beforeFormSubmit` 里分诊、关弹窗、再跳到邻行。nerve 的弹窗在 `beforeFormSubmit` 之后查 `formScope()`，它的键含 `isOpen` 与 `data?.id`：弹窗一关或详情因选中移走而卸下，`inSession()` 为假，抛 `SessionChangedError`，`handleUpdateIssue` 不执行，表单的 `.catch(() => undefined)` 吞掉。结果：条目被接受了，用户在弹窗里改的优先级、负责人没有发出，也没有任何提示。spec 只写“接受 … 确认之后，选中移到 …”，没有说接受的移选中要等弹窗自己的提交（分诊与之后的 `PATCH /issues`）落定；T8 的“接受确认之后只发改过的字段”一格若只测弹窗的接线、不带动作模型的移选中，查不出。另一件：分诊确认之后 canonical 的详情换了对象，弹窗的 `data` 若直接读 canonical 会在提交途中变。 | `C/components/issues/issue-modal/base.tsx:93-98`、`:209-217`；`form-submission.ts:117-131`；`IN/content/inbox-issue-header.tsx:116-140`、`:253-269` | spec §2.8 写明：`beforeFormSubmit` 只分诊（被拒即提示并抛出），不关弹窗、不移选中；移选中与关弹窗在弹窗的提交整个落定之后（`onSubmit` 或 `onClose`）；修改被拒时不移、弹窗留着并说明条目已被接受；弹窗的 `data` 是打开时取的一份，画在随重读撤下的详情正文之外。plan T8 加格：接受并改了一个字段，`PATCH /issues` 发出并确认之后才移选中；修改被拒时选中不动、弹窗仍开着；变异 `t8-mut-move-in-before-submit`。H10 (page) 的接受一段带一个改过的字段，断言 `PATCH /issues` 的请求体。 |
| I3 | spec §2.10“新建弹窗：只把 `create-root.tsx` 的提交 … 改为 `root.intake.submit` … `create-root.files.test.tsx` 只跟随路径；其余不动”、§1“成功之后不跳转到新条目（P5b）”；plan T6b 的 Files 与允许清单 | **“只改提交、其余不动”做不出，四处。** ① `create-modal/issue-description.tsx:21`、`:43` 也读旧 hook（`useProjectInbox().loader`），T6b 删掉 hook 之后它不编译，不在 T6b 的文件清单里。② 成功之后今天就在跳转（`create-root.tsx:139`），读的是旧回答的 `res?.issue?.id`；新回答是 `IssueDetail`，这一行必须改，改成 `res.id` 只是一个词，spec 却写“不跳转（P5b）”：照 spec 得删掉跳转、P5b 再加回，比改对更费事。③ 请求体今天带 `target_date: formData.target_date || null`（`:131`）；`IntakeIssueCreate.target_date` 是不可为 `null` 的字符串、不给即无，`additionalProperties: false`：没选日期时须省略这个键。④ `create-root.files.test.tsx:143-151` 断言旧地址（`/api/workspaces/…/inbox-issues/`，新守卫 `old-intake-urls` 连测试一起查）、旧请求体 `{issue: {…}}`、以及“没有任何生成客户端的请求”（`nerve.calls` 为空）；改接之后三处都不成立，“断言不改、只换 `:40` 的 mock”做不到。 | `IN/modals/create-modal/issue-description.tsx:21`、`:43-48`；`create-root.tsx:61`、`:124-140`；`IY:2353-2388`；`create-root.files.test.tsx:40`、`:143-151` | spec §2.10 与 plan T6b 改为“新建弹窗只改到让它经新接口提交所必需的”：列出 `issue-description.tsx`（去掉 `loader` 的读与那段占位）、请求体照 `IntakeIssueCreate`（没有日期不带键）、成功之后以 `res.id` 跳转（建议 P5a 就做，§1 与 §5 的 P5b 一项删去；若仍留给 P5b，写明 T6b 删掉那一行）。允许清单把 `create-root.files.test.tsx:143-151` 的三处断言写明改法（新地址、新请求体、对生成客户端只有那一个 POST），性质不变（发出的描述不含图片）。T6b 加格：没选日期时请求体没有 `target_date`。`console.error`（`:153`）与英文字面量的提示是否随 T6b 改，写一句（§2.8 说不 `console.error` 并点了这一行，§1 又说弹窗其余属 P5b）。 |
| I4 | spec §2.8“在途”：“每个弹窗关上时焦点回到打开它的按钮。动作确认、选中移走之后，焦点给新选中的那一行”；§2.12 H20 一行；plan T8 的在途一组格、T13 | **两种成功之后既不移选中、打开者也不在了，焦点没有落点；指针一路没有写。** ① 标重复不移选中（R14），成功之后条目成了已处理，页头只剩复制与打开，打开弹窗的“标重复”按钮已卸下，Headless UI 还焦点落空，焦点到 `body`。改暂缓、暂缓（按钮换成“取消暂缓 / 改暂缓”）同理；访客或成员提交者改不了状态所以不涉及。② 窄屏的页头把动作放在菜单里：菜单项随菜单卸下，“回到打开它的按钮”只能是菜单的按钮；P4 在这里出过两轮（键盘一路修了，指针一路在复审才发现：Headless 的菜单按钮在 pointerdown 时不取得焦点，菜单项在 mousedown 时取得）。③ 动作在途时按钮停用，P4 O4 的根因正是“停用的按钮收不回焦点”。spec 的 H20 只写“键盘、读屏与焦点”，没有分键盘与指针，也没有“扣住回答、等两帧再看焦点”的写法。 | spec `:186-187`；P4 阶段记录 §9 的 `03a6acf1`、`3df03b12`（`docs/v0/M7-collaboration/reviews/P4-web-notifications-review.md:166`、`:173`）；`C/components/workspace-notifications/sidebar/use-leaving-focus.ts` | spec §2.8 给每种结局写落点：移了选中 → 新选中的行；没有选中 → 列表；**没移选中而打开者不在（标重复、暂缓、改暂缓成功）→ 选中的那一行**（或详情的状态标签，取一种）；被拒 → 弹窗里；窄屏菜单里打开的弹窗取消时 → 菜单的按钮。H20 (page) 写明每个弹窗键盘打开一次、指针打开一次，成功与取消各看一次焦点，成功的那次扣住回答、放行之后再断言（三引擎）。plan T8 的 jsdom 格加“标重复成功之后焦点在选中的行”，变异 `t8-mut-duplicate-focus-body`。 |

### 次要

| # | 位置 | 发现 | 须做的修正 |
|---|---|---|---|
| m1 | spec §2.4 拒绝一段“重读不占队列”（R28） | 既有的写后重读都在队列的轮次里做（关注、评论、表情、链接）。重读在轮次之外时，同一条目排在后面的写先轮到：重读在途时 `getIssueById` 是 `undefined`，`updateIssue` 抛 `Issue has not been read`（`issue-writes.ts:255`），用户看到一条通用错误。另：重读可能给 `undefined`（已看不到、被更新的读超过），spec 没有写这时怎么办。 | 重读放在被拒那次写的轮次里、交出拒绝之前；重读给 `undefined` 时从收集箱的每份列表移出（同 `issue.intake_not_found`）。T5 加一格（冲突之后排着的写等重读落定）。 |
| m2 | spec §2.4 的表“轮到时所持的详情 `inIntake(root.issues.getIssueDetailById(id))`” | 列表的行落在已持有的详情上时，canonical 保留详情旧的 `intake`（`issue.store.ts:31-34`）。别人接受了一个条目，本人此前在收集箱读过它的详情，之后在项目列表里看到这一行并改它：`inIntake` 仍为真，发到收集箱的端点，得 `issue.intake_closed`，R28 重读之后用户要再做一次；若是看板的拖动，这次拖动落空。另 `inIntake(undefined)`（只持有行）没有定义。 | spec 写明 `inIntake(undefined)` 为假（行不在收集箱，服务端的列表不列它们），并把上面的情形记为已知的一次性拒绝（R28 之后自愈）；T5 加一格钉住“重读之后再改走 `PATCH /issues`”。 |
| m3 | spec §2.5 “重复于”的 fetcher `root.issues.refreshAuthoritative(duplicateToId)`、§2.9 | `refreshAuthoritative` 是核验的入口：它把覆盖目标所在项目的每份列表记为受影响、把目标所在的格标成陈旧；读失败（非 404）时把错误写到这些**项目列表**上（`issue-verification.ts:139-146`、`:236`、`:253`；`issue-list-binding.ts:231-236`）。只为显示一行“重复于”不该动项目列表。读另一个工作项来显示的先例是父工作项：`fetchIssueCore`（`issue-details/issue.store.ts:56-70`、`:81`）。 | 改为 `issueDetail.issue.fetchIssueCore(slug, projectId, duplicateToId)`；T9 加一格：读被拒时持有的项目列表没有错误。 |
| m4 | spec §2.6“选中读不到 … 以 `replace` 清掉选中并提示”、§2.5 详情的键 | ① P4 m1 的同一形：SWR 对这个键留着上一次的失败，新挂载的第一次渲染就返回它，页面会不读就清选中、再提示一次。② 非 404 的失败（网络、5xx）spec 没有写：既不该清选中，也不该空白。 | 照 browse 详情与通知预览的先例，“读不到”只在这次挂载自己的读落定之后判定；别的失败显示可重试的失败态（浮层今天的 `error` + `retry`，`peek-overview/root.tsx:93-116`）。T6a 加两格与变异 `t6a-mut-stale-error-clears`。 |
| m5 | spec R25、plan 的检查点与 T1（“改为 `m7-pages m6-pages m4-detail`”）、T13；brief `:236`（“all three configs”） | 主控裁定是四个配置都跑（`rulings.md:59`：加 m4-detail，不替换 m6-acceptance），`final.sh` 已经是四个；spec、plan、brief 都写三个，plan T1 还要把它“改为”三个。 | 三处都改为 m7-pages、m6-pages、m6-acceptance、m4-detail；plan T1 的那一步改为“核对”。 |
| m6 | plan “knip 与第一个读者”、T2 的 `fake-intake.ts`、T6b 的 `@nerve/constants` | ① `fake-intake.ts` 在 T2 建四个导出（`intakeIssueOf`、`pageOf`、`countsOf`、`detailOf`），T2 只有计数的测试，另外三个没有读者，`make knip` 会报未用的导出。② 页签、状态、排序的选项常量（`@nerve/constants` 的 `intake.ts`）第一个读者是 T6a 的新树，plan 只在 T6b“改写”它，T6a 的文件清单里没有。 | T2 只建 `countsOf`，其余在 T3、T4、T5 随第一个读者加；T6a 往 `intake.ts` 加新选项（旧的仍被旧 store 读），T6b 只删旧的。 |
| m7 | plan 的性质与变异表 | 几条验收里写了的性质没有格或变异：“Add work item 只在开着、未归档、有角色时给”（A6、R21、R27，plan 里没有这几个字）；R4 的“只一次”只有“取数在外”一格，没有“列表的读在接受发出之后发出、在回答之前落定”一格（`heldSources` 管的那一种，`issue-writes.ts:141`）；重复选择器不含条目自己；R31 的“已处理不显示数”。 | 各加一格；前两条加变异（`t6a-mut-add-on-archived`、`t5-mut-accept-ignores-sources`）。 |
| m8 | spec §7；设计 §12、§14 的 T1 改动 | ① 主控裁定“收集箱列表的行是自己的副本 … 记入 spec 的差异”（`rulings.md:55`）没有进 spec §7（全文没有“副本”）。② 设计 §12 的 P5b 一行没有链接（Q14）与多选的忙时守卫（Q34），与 spec §1 的 P5b 清单不一致。③ §14 的 P5a 一行还没有加（spec R1、plan T1 都说要加）。 | ① spec §7 加一行（并在 §2.3 的确认处说一句）；② 补进 P5b 一行；③ T1 提交之前加上。 |
| m9 | spec §2.12 的 H9、H10 两行；plan T10、T11 | ① H10“接受被拒（扣住分诊的回答，在服务端先由另一个管理员接受）”：扣住回答改不了结果，要的是弹窗开着时由另一个管理员经接口先接受、再提交。② H9“直接打开时的精确请求表”：列表有行时页面自动选第一行，随之有详情的一串读（工作项、表情、链接、动态、评论、子工作项、关联、项目状态第二次、描述版本）；表要么含它们，要么在空的收集箱上断言。③ H9 一条里有 51 次提交、七种筛选、分页、两种角色、计数与停用页，火狐上可能过 30 秒。 | ① 改写法；② 写明取哪一种（建议空收集箱断言请求表，另一条断言详情的读）；③ H9 拆成两条测试（标题都带 `H9 (page)`）。 |
| m10 | spec §2.6、§2.8 | ① 分诊之后移选中是 `push` 还是 `replace` 没有写：`push` 时后退回到刚分诊掉、已不在这个页签列表里的条目。② 接受弹窗在分诊被冲突拒绝（`issue.intake_transition_not_allowed`、`issue.intake_not_found`）之后的去留没有写；R29 的判断是“所持详情不在收集箱”，别人接受之后本人再提交会直接发 `PATCH /issues`，这是可以的，但要说明。 | ① 取 `replace`；② 冲突拒绝之后关掉弹窗（重读已把页头换成已处理），R29 加一句“不论是谁接受的”。 |
| m11 | plan T6a | 一个提交里新建约 20 个文件与它们的测试、删约 35 个文件、改四个弹窗、两条守卫与文案、16 个变异，是 P4 任何一个 Task 的两倍以上；逐 Task 的审查很难看全。 | 保持一个删除提交，但在 plan 里把 T6a 写成两个可分别审查的步骤（列表侧：路由、页面状态、侧栏；详情侧：详情、页头与动作模型、删除），各有自己的 RED 收据，审查分两次看；或把详情侧拆成 T6b、原 T6b 顺延为 T6c（17 个 Task，超一条，主控定）。 |

### 观察

- **O1 未知动态值的通用文案键：保留。** 合同里 `IssueActivity.verb` 是不设枚举的 `string`（`IY:3202-3203`）；`IntakeStatus` 是五个值的闭集，服务端今天只写这五个（`domain/intake.go:204`），但合同没有把 `verb` 绑到它。按合同渲染一个开放的字段，一条通用键不是死代码；T9 的那一格以一个不在五个里的值断言它。
- **O2 收集箱条目进了 canonical 之后，M4 的核验会把项目列表的格标成陈旧。** 条目上的评论、表情、关注与 R28 的重读都经 `refreshAuthoritative`，它按 `issueMembership`（不看 `intake`）算出“这一行会在的格”并标陈旧，回到项目列表时多一次那一格的重读；核验的读失败时 `hasRefusal` 对覆盖这个项目的列表为真。都不会插错行（重读来自服务端），是既有机制对“持有而不在列表里的工作项”的一贯行为；H10、H11 各断言一次项目列表没有未接受的条目已够。
- **O3 web 作业的报告线。** P4 的最终分支 CI 在慢 runner 上是 956 秒；P5a 净增 vitest，慢 runner 上越过 960 秒是大概率的事。plan 写了“超报告线先报告”，建议同时写明判别法（Lint、Unit 两步各对同一次 main 的同比），免得每次都停。e2e 两片 817、719 秒，离 1800 秒很远。
- **O4 P4 回填的两个数没有出处。** 阶段记录 §5、§9 的 main CI 一行写“server (0) Test 869、server (1) Test 1002”，归档台账的末行只有 bootstrap 的 585.2 与 970.9；其余数字都对得上。补一个出处（`gh run view`）或删去这两个数。
- **O5 `filters.ts` 装了三种东西。** 筛选与查询、条目的状态（`effectiveStatus`、`rowOf`、`inIntake`）、暂缓的日与时刻、T7 的日期换算都进一个文件；`issue-writes.ts` 为了 `inIntake` 去读 `intake/filters.ts`。建议状态与暂缓另放一个文件（如 `intake/status.ts`），行数也更稳。
- **O6 R31 的总数只对持有的行准。** 经地址直接选中一个不在已取各页里的条目并分诊，确认时没有行可移，`total` 不减，到下一次取数才对。与“不插入”的代价同类，记一句即可。

## 3. 逐项回答

### 3.1 可行性

合同与代码里被点名的东西都在，名字与形状一致（§4）。做不出或与现状不符的只有：I3（新建弹窗的四处）、I1 (b)（三条断言与“不重读”）、m5（配置的个数）、m6（knip）。spec §2.4 说 `useIssueOperations.update` → `IssueWrites.updateIssue`，实际中间经 `projectIssues.updateIssue` → `changeIssue`（`helpers/base-issues.store.ts:73-74`），结论不变：收集箱的控件与版本恢复确实都到 `updateIssue`，“轮到时选端点”放在那里能管住两者。

### 3.2 R3、R4

- **纯搬移**：做得出。`pending`、`pendingIDs`、`observePendingRead` 的读者只有 `root.store.ts:306` 与 `issue-list-binding.ts:58`，没有测试读私有成员；`K:2539` 有 `issue-writes.ts`。拆后约 165 + 150 行，加两个命令与端点的选择在 220 行之内是可信的。
- **端点在轮到时选**：`getIssueDetailById` 在；接受的确认经 `confirmChange` 写进 canonical（带 `intake`），所以排在接受之后的修改在它的轮次里看到已接受、走 `PATCH /issues`，plan 的格与变异 `t5-mut-route-at-call` 对。漏的是 m2（行盖在详情上保留旧 `intake`）。
- **哪些确认可以碰项目列表**：只有接受。spec 的理由成立（`issueMembership` 不看 `intake`，`C/lib/issue-membership.ts:44-99`）。但“不碰”要连目录准备一起不碰（S1），重复目标的读也不该碰（m3）。
- **接受像新建**：`write(undefined, …)` 以 `absent` 确认、发出之前冻结各列表的来源（`:141`），回答先于或后于列表的读都只一行，这正是新建已有的保证；canonical 里已有这个条目不碍事（`confirmChange` 照 `updated_at` 取新的）。缺一格（m7）。接受的 `after` 在默认状态，目录准备不出错。
- **版本恢复**：`description-versions.store.ts:76` 经 `updateIssue`，随端点的选择走收集箱，`canRead` 只给管理员与成员，对。
- **拒绝、not_found、冲突**：方向对；重读要进轮次、要处理 `undefined`（m1）；非接受的分诊另有旧读的问题（I1）。
- **与评论等的先后**：评论、表情、链接、关注都经 `enqueueIssueChanges([id])`，新命令也经它，先后成立；变异 `t5-mut-triage-unqueued` 对。
- **会坏掉既有 M4 行为的地方**：纯搬移本身之外，是 `updateIssue` 里新加的分支（对从未提交的工作项 `intake` 为 `null`，走原路）与 `"inbox"` 一格的改写（I1）。既有测试加 `m4-detail` 能看住前者。

### 3.3 R27–R32 与通用键

- R27（已归档只读）：与合同一致（分诊、修改、删除、提交都答 `project.archived`），同意。
- R28（冲突重读）：同意，按 m1 改位置。
- R29（再提交只发修改）：同意，按 I2、m10 补。
- R30（提交不排队、写后重取）：同意；提交不能经 `confirmCreation` 的理由成立。
- R31（总数）：同意，O6 记一句。
- R32（两个死 prop 删除）：同意；`InboxContentRoot` 的产品读者是 `IN/root.tsx:16`，导出本来就留着。
- 通用键：保留（O1）。

### 3.4 plan 的质量

- **Task 的大小**：T6a 过大（m11）；T6b 的范围写窄了（I3）；T5 两个提交的切法好。
- **顺序**：T2–T5 数据层在前、T6a 接页面、T6b 删数据层，knip 的第一个读者规则除 m6 两处外成立；“store 的成员先于产品读者”的例外写得清楚。删除各在一个提交、守卫先红后绿，对。
- **RED 格**：多数会因所测性质先失败；查不出问题的是 S1（普通列表）、I2（不带移选中）、m4（没有旧失败一格）。
- **变异**：安全、可见、幂等的性质大多有；缺口见 S1、I1、I2、I4、m4、m7。
- **检查点**：三个位置合理；T5 之后另跑 chromium 的 m4-detail 值得保留；配置的个数按 m5 改。
- **故事与时长**：四个文件的名字都合 m7-pages 的 `testMatch`，标题合 `grep`；H9 建议拆（m9）；H20 的拆分规则已写。
- **CI 线**：web 960 秒见 O3；e2e 余量大；服务端第 1 路 P5a 不加 Go，“本机 +30 秒停查、CI 八成只报告”的写法对，见 §5。
- **文件长度、死成员、oxlint、关键词**：`issue-writes.ts` ≤ 220、`intake.store.ts` 过 300 即拆、`cycle-pages.ts` 不加长都写了；死成员三次整份比对；oxlint 等式；`old-intake-urls` 的模式覆盖旧 service 的两种地址（`inbox-issues`、`intake-work-items`），并会命中 `create-root.files.test.tsx:144`（I3）与 `create-root.request.test.tsx:142`（已在允许清单）。

### 3.5 一致性

- spec ↔ plan ↔ brief：m5（配置个数）、I1（“两条”）、I3（新建弹窗）之外一致。
- spec ↔ 裁定：R↔Q 的表完整（`questions.md` 的 32 个 Q 都在，P5b 的五个标明）；缺“列表行是自己的副本”进 §7（m8）。
- spec ↔ 设计：§3.11 的入口规则、§15 A、B 都照做；§12、§14 见 m8。
- Plane 的差异：§7 十二条之外加 m8 一条；I3 若在 P5a 就跳转，没有新差异。
- P5b 的后续与 M8 的交接：spec §5 与 `split.md` §3 对得上，没有丢项。
- 文档链接：0 断。

## 4. 核实为正确的事实（摘要）

- 六个操作的路径、方法、回答与问题码同 spec §2.1（`IY:744-1088`）；分诊的问题码里没有 `issue.intake_closed`（它只在内容修改），spec 把两者并列处理没有问题。
- `status` 省略即全部、每个参数只一次、日期四种形状不许 `none`、`order_by` 六个值、`limit` 默认 50、状态按回答时刻（`IY:752-846`）；CSV 的路径名单含收集箱列表、键含 `status`（`web/packages/api-client/src/index.ts:35-62`）。
- `IntakeIssueUpdate` 恰七个字段、`additionalProperties: false`（`IY:2538-2579`）；`IntakeTriage` 三个字段（`:2389-2409`）；`IssueIntake` 的 `duplicate_to_id` 在目标删除或看不到时为 `null`（`:2345-2352`）；删除对任何状态都许、已接受的只删条目行（`:1023-1057`）。
- 计数只列调用者是有效成员、未删、未归档、开着收集箱的项目（`IY:1066-1076`），与 spec 的门一致；`Project.intake_view` 在生成的类型里。
- 非接受的分诊不改工作项（`IY:945-946`，`triage_intake.go:184`）；分诊动态的 `verb` 是新状态的名字（`domain/intake.go:203-206`）。
- `ReconciledByKey.fetchPart`（`C/lib/reconciled.ts:167-175`）、通知 store 的认游标合并（`notification.store.ts:113-122`）、`useSessionSWR` 的三个选项（`use-session-swr.ts:34-45`）、`followInSession`（`in-session.ts:40`）、`refreshAuthoritative`（`issue.store.ts:113-120`）、`confirmCreation`（`issue-writes.ts:232-238`）、`knownProjectIds`（`project.store.ts:244-251`）、项目角色（`permissions.store.ts:68-75`，工作区管理员给 ADMIN）、`useRouteWorkspace`（`:16`）都在 spec 说的位置。
- 三处入口今天各自只看 `intake_view`（`project-navigation.tsx:108-117`、`use-navigation-items.ts:76-84`、`project-commands.ts:191-202`）；`planningEntryOn` 另排除已归档（`planning-switch.ts:62-71`）。
- 行数：`issue-writes.ts` 302、`root.store.ts`（issue）319、`projects-list.tsx` 240、`projects-list-item.tsx` 431、`peek-overview/view.tsx` 303、`issue-modal/base.tsx` 313、旧页头 470 + 239、旧 store 511 + 194、`cycle-pages.ts` 388、`s2-web-app.spec.ts` 582、`e2e/fixtures/intake.ts` 134；`root.store.test.ts:106` 是 21。
- 弹窗：`beforeFormSubmit` 抛出即停（`base.tsx:214`）；更新经 `onRefused: underFields`（`form-submission.ts:121`）；已有工作项的提交总带 `description_html` 与改过的字段，所以把状态字段的初值设为默认状态不会多发 `state_id`。
- 详情的访客路径不被 M4 的页面层挡住：`useDetailScope` 只看身份与“有项目角色”，`enqueueIssueChanges` 不看角色。
- 旧层的外部读者：`TLegacyIssueUpdateRequest` 只在旧收集箱；`TPaginationInfo` 只在 `types/inbox.ts`；`InboxContentRoot` 的产品读者是 `IN/root.tsx`。
- 门禁：`web/apps/web/package.json:10` 的上限 31；`.oxlintrc.json:165` 是 `inbox/content/description-versions.tsx`；`K:3262-3268` 是页头的 `copy-hook` 例外；`K:2785` 是 `workspace-address-route`；M7 的例外都是 `until: "M7/closeout"`；phase 已是 `M7/P5`，`node tools/keywords.mjs` 无命中。
- e2e：`playwright.m7-pages.config.ts` 的 `testMatch` 与 `grep` 同 spec；`m4-detail`、`m6-pages`、`m6-acceptance` 三个配置都在；点名的帮手（`sentHeld`、`holdAnswer`、`anotherTabSignsIn`、`moveWithinApp`、`refuseClipboardWrites`、`expectListBesideButton`）都在所说的夹具里；`mounts.ts` 的 `PROJECT_CYCLES` 是条件化清单的先例。
- T1 的改动：设计 §12 的两行在 `:306-307`；P4 的状态行、A8、§5 两行、§9 四条与归档台账末行一致（O4 的两个数除外）；归档里 `deadsym.mjs`、`domains.mjs`、`lib/`、`rows-M7-t15.tsv` 都在；P4 阶段记录 §5 的“P5 收集箱 39 行”属实。

## 5. 需要负责人的事

无新的。S1、I1–I4 都是 M4 网页共用代码之内的写法，属主控的局部裁定（I1 取 (乙) 时要改 M4 的三条回归断言，建议在阶段记录里与 R3、R4 一起向负责人说明）。提醒一件已经报告、仍在等的事：main CI 45051719903 的服务端第 1 路是 80.9%，P5a 不改 Go，分支 CI 在慢 runner 上仍可能再越八成，照裁定只报告；调 `tools/go-test-shard.sh` 的分界要负责人定。

## 6. 处理（主控，2026-11-23）

全部发现都改进了 spec、plan 与实施者须知（spec §3 的“预检 re-94 的处理”逐条指到落点），没有需要负责人的事。裁定：

- **S1**：收集箱的去处是 `issue-write-confirm.ts` 里与 `write()` 并列的一条路，不经两段目录准备、不确认进项目列表、不登记 `pending`，只取权限的水位（spec §2.4、R3）；加持有按状态组筛选与分组的两份列表的 RED 格与两个变异（plan T5）。
- **I1**：取（甲）。五种分诊都在自己的队列轮次里以 `refreshAuthoritative({ afterWrite: true })` 重读，分诊的回答不直接写进 canonical；接受另经 `confirmCreation`，`send` 为“分诊、再重读”（R34，与 R3、R4 同列为高风险）。`issue-changes.test.ts` 的三条 `it.each` 断言都不改；若有一条非改不可，实施者停下报告。
- **I2**：接受时选中的移动与弹窗的关闭在整次提交落定之后（`onSubmit` 记下、`finishClose` 的 `onClose` 移动），`beforeFormSubmit` 只分诊（R12、R14）。另定（m10）：用户点行是 push，动作引起的选中变化一律 replace（同 Plane 的自动选中，与 Plane 分诊之后的 push 不同，登记差异）；接受弹窗遇冲突被拒：提示、关闭、选中不动。
- **I3**：T6c 照实列出新建弹窗要改的四处与要改写的两份测试的断言（旧地址、旧形状、“不经生成的客户端”是旧路径的刻画），失败经 `useRefusalToast`；其余留 P5b（spec §2.10、R30）。
- **I4**：按结局定焦点的落点（R35，spec §2.8 的表）：选中移动 → 新的行；选中清空 → 列表；选中不动而打开者已去 → 新页头的主动作，否则有名字、可聚焦的详情区域；等待时用户换了选中 → 不动焦点；取消 → 打开者或窄页头的菜单按钮。H20 覆盖键盘、指针与窄页头菜单三条路（P4 O4 的教训）。暂缓与改暂缓照 R14（同 Plane）移动选中，只有标重复留在原条目——主控对 I4 的原话把暂缓列进“不移动”，以 R14 为准。
- **m1–m11**：照报告的建议全改（m1 冲突后的重读在被拒的写的轮次里；m2 定 R33：没有 `intake` 的列表行落到未被接受的详情上，合并后的 `intake` 记为已接受——列表从不列收集箱中的条目——这是对 M4 canonical 合并的第三处改动，与 R3、R4、R34 同列；m3 经 `fetchIssueCore`；m4 照 P4 m1 每次挂载自己读；m5 四个配置；m6 knip 的第一个读者；m7 补格与变异；m8 列表的行是 store 自己的副本，记进 §7，设计 §12 的 P5b 行补链接与忙时守卫、§14 加 P5a 行由主控改；m9 H10 的拒绝改为第二位管理员先接受、H9 拆为至少三个测试；m11 T6 拆为 T6a（详情一侧）、T6b（列表一侧）、T6c（数据层与新建弹窗的提交），各自一个提交、各自全绿）。
- **观察**：O1 保留通用的动态文案键（`verb` 是不受限的字符串）并加一格；O2 记为 R34 的代价；O3 写明分辨慢 runner 与真实增长的办法（各步对 main 45051719903 同比）；O4 已在 P4 阶段记录注明 main CI 的 Test 时长取自 jobs 接口；O5 状态相关的纯函数移到 `core/store/issue/intake/status.ts`；O6 记进 R31。
- **规模**：17 个 Task（T6 拆三），比约 16 多一个，为了每个提交都能单独审查，接受。
