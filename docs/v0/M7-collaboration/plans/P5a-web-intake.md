# M7 / P5a plan：网页的收集箱——数据层、列表、详情与分诊

> spec：[P5a spec](../specs/P5a-web-intake.md)（§2 是合同，§3 的 R1–R35 是裁定，§6 的高风险是 R3、R4、R34）；预检 [re-94](../../../reviews/re-94-M7P5a预检.md) 修正后通过（处理见它的 §6 与 spec §3 的“预检 re-94 的处理”）。基线 main `89be91e5`（P4 的 noff）。worktree `.codex/worktress/codex/m7-p5a`，分支 `codex/m7-p5a`。台账 `.superpowers/sdd/P5a-web-intake/progress.md`（ignored）；主控裁定 `rulings.md`，调研 `research/`。实现代理每个 Task 先读同目录的 `implementer-brief.md`。17 个 Task（`research/split.md` §2 的 15 条；T6 拆为 T6a 详情侧、T6b 列表侧、T6c 旧数据层，re-94 m11）。下文 `C/` = `web/apps/web/core/`，`A/` = `web/apps/web/app/(all)/[workspaceSlug]/(projects)/`，`IN/` = `C/components/inbox/`，`K` = `tools/keywords.json`，`IY` = `api/modules/issue.yaml`，`HO/` = `docs/v0/M7-collaboration/handoffs/`。

## Global Constraints

**分支与提交**
- 主目录永远在 main。提交用详细中文，前缀 `feat/fix/docs/test/refactor(M7/P5a)`，结尾一行 `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`。提交只在本 Task 的 lint 过了之后，不与它串在一条命令里，用显式路径（`git commit -F <msg> -- <paths>`）。
- 实现代理不推送、不合并；检查点由主控推送。
- 关键词的 phase 是 `M7/P5`（P5a、P5b 共用，spec R1）；不延长任何 `until`。
- 本段不改 `server/`、`api/`、`deploy/`、`.github/`。超出 spec 的契约或架构改动先停下报告。

**只读与禁区**：`plane/`、`refer/` 只读，不在其内跑 git。不 stash、`git clean`、`reset --hard`、强推。不动别的项目的容器。容器套件一次一个：`make test`、`make test-shard`、`make e2e` 与 Playwright 的配置不并发。

**代码规则**（总体设计 §7.2、§7.7）
- 手写文件约 400 行以内，新代码进新文件，按关注点分（spec §2.13 的表）。新文件的版权头 `Copyright (c) 2026-present OpenNerve`，改写的 Plane 文件保留原头。
- 类型只来自 `@nerve/api-client`；只给界面用的形状在 `C/store/issue/intake/filters.ts`，选项在 `@nerve/constants` 的 `intake.ts`。
- service 每代以自己的客户端构造、不转换；没有模块级的带令牌 service；store 只经自己的根找兄弟；SWR 只经 `useSessionSWR`，只用它的三个选项（`C/lib/use-session-swr.ts:37`）；列表、计数与“重复于”不传 `revalidateOnReconnect: false`，拥有详情宿主的详情读照 M5 设计 §9 传（整分支审查 I1 之后改，`d673c9f3`）；新代码不读 router store 的地址（路由参数或 `useRouteWorkspace()`）。
- 修改一个接一个；请求体轮到时才算；确认写进持有的每一份；取数与修改的应答对齐（`Reconciled`）。收集箱条目的写只经 M4 的逐工作项写队列（spec R3），不另开队列；收集箱的去处不经 `write()` 的两段目录准备（spec §2.4，re-94 S1）。
- 页签与选中在地址（用户点行 `push`，动作引起的变化 `replace`），筛选与排序在 `IN/page-state.tsx`，不进 store（spec R5、R14）。
- `await` 之后的提示、导航与跟进一律经 `followInSession`（`C/lib/in-session.ts:40`）；拒绝经 `useRefusalToast`；不 `console.error`；焦点照 spec R35，不落到 `body`。
- 删掉不用的代码，不加开关、桩或兼容别名。新旧收集箱只在 T2–T6c 并存。
- 不放宽 `K` 的规则、`lint-cap`、knip、`error-messages.test.ts`、i18n 的 `check:sync`。
- **不再加长的文件**：
  - `C/store/issue/issue-writes.ts` 302：T5 先纯搬移拆分，拆后加命令不超过 220 行（spec §2.4）
  - `C/components/workspace/sidebar/projects-list-item.tsx` 431（已过 400，不碰）
  - `e2e/stories/smoke/s2-web-app.spec.ts` 582（只经常量跟随）
  - `e2e/fixtures/cycle-pages.ts` 388
  - `C/components/issues/peek-overview/view.tsx` 303（不碰）
  - `C/components/issues/issue-modal/base.tsx` 313（不碰：接受弹窗只经它既有的 prop）
  - `C/store/issue/root.store.ts` 319（只加 `intakeApi`、`intake` 两个成员）

**knip 与第一个读者**（re-94 m6）
- 新文件、新导出在它第一个读者出现的 Task 里建立（测试数据的导出以第一个读它的测试为准，knip 把测试当入口）：
  - T2：`IntakeService.counts`、intake store 的计数、`use-intake.ts`、`use-intake-counts.ts`（读者：`projects-list.tsx`）、`pendingCount`（读者：`project-navigation.tsx`）、`intakeEntryOn`（读者：侧栏、页签栏、命令面板）；`fake-intake.ts` 只有 `countsOf`。
  - T3：`IntakeService.list`；`filters.ts` 的 `IntakeFilters`、`filterKey`、`listQuery`（读者：store 的列表）；`fake-intake.ts` 加 `intakeIssueOf`、`pageOf`。
  - T4：`filters.ts` 的 `matches`；`status.ts` 的 `effectiveStatus`、`rowOf`（读者：store 的确认）；`fake-intake.ts` 加 `detailOf`。
  - T5：`IntakeService.triage`、`update`、`delete`；`status.ts` 的 `inIntake`；`issue-write-fields.ts` 的 `intakeUpdateBody`；`issue-write-confirm.ts`（读者：`IssueWrites`）。
  - T6a：新的 `IN/content/`、页头与动作模型；`@nerve/constants` 的 `intake.ts` 加状态名的键表（读者：状态标签；旧导出仍被旧 store 读，留到 T6c）。
  - T6b：`filters.ts` 的 `IntakeTab`、`defaultFilters`、`filtersOf`（读者：页面状态与列表的取数）；`@nerve/constants` 加页签的选项；列表侧的新树。
  - T6c：`IntakeService.submit` 与 store 的 `submit`（读者：新建弹窗）。
  - T7：筛选菜单、日期区间的换算、`@nerve/constants` 的排序与日期预设的选项。
  - T8：`status.ts` 的 `snoozeDay`、`snoozeInstant`（读者：暂缓弹窗）；接受弹窗。
  - T9：`duplicate-of.tsx`。
- **例外，只有一类**：store 的成员先于它的产品读者。`getList`、`fetchList`、`fetchMore`（T3）的读者是 T6b 的页面；`confirm`、`drop`、`countAgain`（T4）的读者是 T5 的写；`triageIntake`、`deleteIntake`（T5）的读者是 T6a 的动作。T5、T6a、T6b 结束时各核一次这些成员都有了产品读者，台账记下。
- knip 把 vitest 文件当入口，只被测试读的产品导出它查不出；“只给测试读不算读者”由逐 Task 的审查执行。knip 也查不出包的 barrel 导出：那里靠死成员的整份比对。
- 删掉某个文件、导出的最后一个读者的 Task，同一提交删去它（spec §2.10 的三组）。

**告警上限**：`lint-cap` 要求 oxlint 的告警数恰好等于 `web/apps/web/package.json:10` 的上限（现 31）。新文件不得新增告警；T6a–T6c 删掉的文件若带告警，同一提交照实降上限并记收据（`t6a-oxlint-count` 等）；T15 只核对。

**文案**：新文案 en 与 zh-CN 同一提交，不许一个键是另一个键的前缀；文案键写完整的字面量（`tools/i18n-keys.mjs` 才认得），按状态名查表时表里写全每个键。三个 `inbox_issue.errors.*_permission` 在 T6a（它们最后的读者是旧页头）删。不再有读者的旧键只在 T9 删一次：只看 `inbox_issue.*` 与 `project_empty_state.intake_*`，人工逐个 `grep`，台账列出删了哪些。

**守卫**
- `project-address-route`：T5 收入 `issue-write-confirm.ts`（hit 样例）；T6a、T6b 收入读项目的新文件（`HO/M3-closeout.md:61`）。
- `workspace-address-route`（`K:2785`）：T2 收入 `use-intake-counts.ts`；T6b 收入读工作区的新文件。
- `copy-hook` 的例外（`K:3262-3268`）随 T6a 删除旧页头。
- `old-intake-urls` 在 T6c 加：先看它命中旧 service 与两份新建弹窗的测试（红的收据，不提交），再与删除、改写同一提交（P4 re-93 m1）。
- `.oxlintrc.json` 的 swr 名单（`:121-203`）随各 Task 收入新的取数文件，T6a 删去 `inbox/content/description-versions.tsx`（`:165`）。
- 每次改 `K` 跑 `node tools/keywords.mjs`。

**允许改的已有测试与夹具**（改之前在台账逐条写明，只照这里写的改法；其余断言是回归，不改；编译跟随只为编译、不改断言，逐处记台账）：
- `C/store/issue/issue-changes.test.ts`（re-94 I1）：T5 把 `"inbox"` 一格（`:605` 的名单与 `:651-662` 的旧 `InboxIssueStore`）改为新的 `triageIntake(…, { status: "accepted" })`（以 `vi.spyOn(root.intakeApi, "triage")` 给回答或拒绝），名字改为 `"triage"`。**三条** `it.each`（`:663-684`、`:685-695`、`:696-712`）的断言一条不改，预期照旧成立：接受做成之后在轮次里有写后的 `GET /api/v0/issues/x`，写之前发出的读到达时不被采纳，canonical 取写后读的回答；被拒（`new Error("resource refused")`，不是 R28 的冲突码）不发写后的读；写后的读在途时换代，`send` 里的 `assertActive` 抛 `SessionChangedError`，`fetchActivities` 不被调。若哪一条断言须改，先停下报告主控，不自行改。
- `C/store/root.store.test.ts:106`：T6c 21 → 20。
- `IN/modals/select-duplicate.test.tsx`（126）：T8 把 `useParams` 的 mock（`:18`）换成以 prop 给的 id；断言不改，另加“地址的参数与 prop 不同时用 prop”“不列条目自己”两格。
- `IN/modals/create-modal/create-root.request.test.tsx`（181，re-94 I3）：T6c 改写。
  - `:133-166` 一格（`it.each`）：旧 service 的 `post`、旧地址与 `{source, issue: {…, target_date: null}}`（`:142-152`）→ 生成客户端的 `POST /api/v0/projects/{project_id}/intake-issues`，请求体 `{name, description_html, priority: "none", label_ids: [], assignee_ids: []}`，没有 `target_date`（C10）；回答之后对旧 store 的三条断言（`:160-162`：`projectInbox` 的条目、`inboxIssueIds`、`total_results`）→ canonical 没有它、持有的收集箱列表重取第一页；`navigate` 一次（到 `?currentTab=open&inboxIssueId=<回答的 id>`）、`close` 一次照旧。理由：旧 store 删除，回答的形状变了（`IssueDetail`）。
  - `:167-181` 一格：`console.error` 的替身与断言（`:170`、`:175`、`:180`）→ 拒绝经 `useRefusalToast` 提示一次、`console.error` 没有被调；没有跳转、关闭、清空，`issuesMap` 为空照旧。理由：T6c 去掉 `console.error`（spec §2.10）。
- `IN/modals/create-modal/create-root.files.test.tsx`（154，re-94 I3）：T6c。`:40` 的旧 hook mock 换成新的（编译跟随）；`:144-152` 的断言是旧路径的特征，改为：
  - `:144-147` 旧地址 `/api/workspaces/…/inbox-issues/` 与 `{issue: …}` → 生成客户端的 `POST /api/v0/projects/{project_id}/intake-issues`，请求体含 `name: "New intake"`（旧地址会被 `old-intake-urls` 命中）；
  - `:148` 从 `issue.description_html` 读 → 从请求体的 `description_html` 读，`:149` 的断言（不含图片）不改；
  - `:151` “旧 service 的别的方法没有被调” → `APIService` 的任何方法都没有被调；
  - `:152` `nerve.calls` 为空 → 恰好那一个 `POST`；
  - `:153` 没有原始的传输：不改。
  - 它钉的性质（发出的描述不含图片、没有任何文件请求）不变。
- `IN/inbox-filter/applied-filters/applied-filters.test.tsx`（91）：T6b 随旧筛选树删除。它钉的性质（`:50`：已选筛选的移除按钮有名字、留下其余的值、不提交外层表单）由 T7 的已选筛选测试接上。**性质的空窗 T6b → T7**，台账记下。
- `C/store/inbox/inbox-issue.request.test.ts`（212）：T6c 随旧 store 删除；它钉的请求由 T5 的新测试接上（T5 在前，没有空窗）。
- `C/services/inbox/intake-work_item_version.service.test.ts`（57）：T6a 随旧版本 service 删除（M4 的版本 service 有自己的测试）。
- `C/components/issues/fake-issue-properties.tsx:30-32`：T6a 删去对旧 hook 的 mock（`label/root.tsx` 不再读它）。
- 不改、不受影响：`C/components/power-k/config/navigation/commands.test.tsx`（`:139` 一行在 `intakeEntryOn` 之后不变，回归）；`C/components/workspace-notifications/root.test.tsx`（`:58` mock 的 `InboxContentRoot` 仍从 `@/components/inbox/content` 导出，`:215-221` 回归，P5b 改）；`issue-writes.test.ts`、`issue-drag.test.ts`、`issue.store.test.ts` 与读 `root.writes` 的详情测试（T5 的拆分是纯搬移、R33 只加一种情形，回归）；`issue-activity/data.test.tsx`；`close-policy.test.tsx`、`update-command.test.tsx`（接受弹窗的回归）。
- e2e：
  - `e2e/fixtures/mounts.ts`：T2 加 `INTAKE_COUNTS` 与头注释 `:5-16`；读它的八个文件只经常量跟随，不改。
  - `e2e/fixtures/intake.ts`（134）：只可追加接口帮手，既有导出不改。
- 产品代码里动到的 M4、M6 文件：`issue-writes.ts`（T5 拆分与两个命令）、`issue-write-fields.ts`（T5，`intakeUpdateBody`）、`C/store/issue/issue.store.ts`（T5，`choose` 的一种情形，R33）、`C/store/issue/root.store.ts`（T2、T5）、`label/root.tsx`（T6a，删 `isInboxIssue`）、`…/activity/actions/inbox.tsx` 与 `activity-list.tsx:85-87`（T9）、`project-navigation.tsx`、`use-navigation-items.ts`、`project-commands.ts`、`projects-list.tsx`（T2）。

**红的窗口**：不预期有，也没有红的提交。
- 收集箱页在 main 上本来就坏（每个旧地址都 404），T2–T6b 之间仍是坏的；没有故事访问它（`grep` 过 `e2e/stories`：只有 API 故事与经接口打开开关的故事）。T6a 之后旧侧栏与旧根仍在、渲染新的详情；它们的旧读仍是 404，不循环（新的详情只按自己的读决定选中）。
- `INTAKE_COUNTS` 与侧栏的取数在 T2 同一提交。
- T6c 的守卫先看红（只留收据），与让它变绿的删除、改写同一提交。
- 名单之外的任何红先停下报告。

**检查点**（七项用 `sh .superpowers/sdd/P5a-web-intake/gates.sh <前缀>`，带页面配置用 `final.sh <前缀>`：七项之后跑 m7-pages、m6-pages、m6-acceptance、m4-detail 四个配置，三种引擎，re-94 m5）：
- **0（T2 之后）**：`final.sh cp0`；主控推送、核分支 CI；台账记基线：本机第 1 路（spec R25 的基线）、web 作业的 Lint、Unit tests 与 Build 三步、e2e 两片、Go 两路（以 `gh` 读 main CI 45051719903 的同名作业与步作对照）。
- **1（T9 之后，产品代码完成）**：`final.sh cp1`；主控推送、核分支 CI。
- **2（T13 之后，故事完成）**：`final.sh cp2`；四个新故事文件另跑 `--repeat-each=3`；推送之前照 spec §2.12 估出 e2e 两片的时长，估出超过 30 分钟先报告、不推送；之后核分支 CI。
- **终检（T15）**：`final.sh fin`。
- 第 1 路：本机第 1 路超过 CP0 基线 +30 秒先停下查；CI 的 bootstrap 过期限的八成只报告。本机 Go 两路因 M6-M8-05 失败且重跑一次仍不过时，以分支 CI 的服务端两路为准并记录。

**证据**
- 每次检查经 `run-check.py`，变异经 `mutate.py`（每个旧串恰出现一次，工具恢复并核 sha256）；标签 `t<N>-…`、`cp<k>-…`。
- 下表每个性质至少一个能编译、能让测试失败的变异。两层都在查同一件事时两层一起改，并留只改一层的对照。只有竞态杀得死的变异不算。
- 页面故事先写一个期望错误的版本（或以 e2e 变异），看到断言失败，再改正。

**测试时间**
- Go 不变，不加顶层名。
- web 作业期限 20 分钟、960 秒报告线；每个检查点记 Lint、Unit tests、Build 三步；加大量 vitest 的 Task 记新文件的时长。**越线时的判别**（spec R25，re-94 O3）：三步对同一次 main CI 的同名步各算比值；三个比值相近（同比例变慢）是慢 runner，照报告、不停；只有 Unit tests 的比值明显高于另两步才是本段的增长，先报告主控再继续。
- e2e 每片 35 分钟、报告线 30 分钟。T10–T13 记每个新文件在三种引擎的时长，本机 chromium 每个约 8 秒为先例的上限；H9 与 H20 会超，写明原因；任何一个 `test()` 单引擎超过 30 秒先报主控。

## 性质与变异

| 性质 | 变异（标签） | Task |
|---|---|---|
| 计数按工作区 id 存、读按 slug | 计数按 slug 存（`t2-mut-count-key-slug`） | T2 |
| 计数的门 | 门不看 `intake_view`（`t2-mut-gate-switch`）；不看归档（`t2-mut-gate-archived`）；不看 `member_role`（`t2-mut-gate-role`） | T2 |
| 计数的键带过门的项目 | 键不带项目 id（`t2-mut-key-no-projects`，开关一变即重读的一格失败） | T2 |
| `intakeEntryOn` 恰为设计 §3.11 | 只给管理员与成员（`t2-mut-entry-no-guest`）；排除已归档（`t2-mut-entry-archived`）；不看开关（`t2-mut-entry-switch`） | T2 |
| 三处一条规则 | 命令面板回到只看 `intake_view`（`t2-mut-palette-own-rule`，没有角色的一格失败）；页签栏同（`t2-mut-tabnav-own-rule`） | T2 |
| 数大于 0 才显示、名字含数 | 0 也显示（`t2-mut-count-zero`） | T2 |
| 每个页签显式发 `status`（C1） | 等于默认值时不发（`t3-mut-omit-status`） | T3 |
| 空数组不发、不重复参数（C2） | 空的优先级发 `priority=`（`t3-mut-empty-array`） | T3 |
| 每页 50（C5） | 不发 `limit`（`t3-mut-no-limit`） | T3 |
| 排序是合同的值（C4） | 发 `issue__created_at`（`t3-mut-order-plane`） | T3 |
| 键规范化、往返 | `filterKey` 不排序数组（`t3-mut-key-order`） | T3 |
| 认游标的合并 | 不核游标即合并（`t3-mut-merge-any-cursor`） | T3 |
| 下一页去重、取回答的游标 | 不去重（`t3-mut-no-dedupe`）；留旧的 `next_cursor`（`t3-mut-keep-cursor`） | T3 |
| 刷新换第一页 | 第一页并进旧行（`t3-mut-refresh-merge`） | T3 |
| 列表按项目存 | 键不带项目（`t3-mut-key-no-project`） | T3 |
| 有效状态（C7） | `effectiveStatus` 不看 `now`（`t4-mut-snooze-any`） | T4 |
| 确认不插入 | 不相符也插（`t4-mut-insert`） | T4 |
| 确认写进持有的每一份 | 只写第一份（`t4-mut-first-list-only`） | T4 |
| `total` 随移出减一 | 不减（`t4-mut-total-stays`）；替换也减（`t4-mut-total-on-replace`） | T4 |
| 只写它的项目 | 不比项目（`t4-mut-other-project`） | T4 |
| 计数重取不越过门、失败留值 | 没有持有的计数也取（`t4-mut-count-ungated`）；失败清掉（`t4-mut-count-failure-clears`） | T4 |
| 端点在轮到时选 | 调用时选（`t5-mut-route-at-call`，排在接受之后的修改的一格失败） | T5 |
| `inIntake(undefined)` 为假 | 只持有行时走收集箱（`t5-mut-undefined-in-intake`） | T5 |
| 收集箱的去处不经目录准备（re-94 S1） | 收集箱的去处也准备目录（`t5-mut-intake-prepares-catalog`）；删除经 `write()` 的确认那一段（`t5-mut-delete-through-write`，列表被标成失败的一格失败） | T5 |
| 收集箱的确认不写项目列表 | 收集箱的去处也调 `acceptConfirmed`（`t5-mut-intake-into-project-lists`） | T5 |
| 分诊在轮次里重读、旧读不改回（R34） | 分诊照回答写进 canonical、不重读（`t5-mut-triage-no-floor`）；重读不带 `afterWrite`（`t5-mut-reread-no-floor`） | T5 |
| 接受像新建一样、只一次 | 接受走收集箱的去处（`t5-mut-accept-intake-only`）；接受以 `before: known` 确认（`t5-mut-accept-known`）；来源在分诊发出之后才冻结（`t5-mut-accept-ignores-sources`） | T5 |
| 重读读不到时移出 | 读不到时留着（`t5-mut-reread-undefined-stays`） | T5 |
| 与同一工作项的别的写一个接一个 | `triageIntake` 不经 `enqueueIssueChanges`（`t5-mut-triage-unqueued`） | T5 |
| 被拒值不变 | 被拒也确认（`t5-mut-confirm-on-refusal`） | T5 |
| `issue.intake_not_found` 移出 | 不移出（`t5-mut-not-found-stays`） | T5 |
| 冲突在轮次里重读（R28） | 不重读（`t5-mut-conflict-no-reread`）；重读放到轮次之外（`t5-mut-conflict-reread-outside-turn`，排着的写的一格失败） | T5 |
| 行盖在详情上不留未接受的 `intake`（R33） | `choose` 照旧保留详情的 `intake`（`t5-mut-row-keeps-stale-intake`） | T5 |
| 计数：分诊与删除重取，内容修改不重取 | 分诊不重取（`t5-mut-no-count-on-triage`）；内容修改也重取（`t5-mut-count-on-edit`） | T5 |
| 内容修改只有七个字段（C9） | `intakeUpdateBody` 放过 `state_id`（`t5-mut-body-state`） | T5 |
| 读不到在这次挂载的读落定之后才判定（re-94 m4） | 以 SWR 留着的旧失败判定（`t6a-mut-stale-error-clears`） | T6a |
| 别的失败可重试、不清选中 | 网络失败也清选中（`t6a-mut-failure-clears`） | T6a |
| 详情以条目 id 为 key | 去掉 key（`t6a-mut-detail-unkeyed`） | T6a |
| 分诊的弹窗不随重读撤下 | 弹窗画进 `IssueDetailHost` 的正文（`t6a-mut-modal-in-body`） | T6a |
| 动作矩阵（spec §2.8 的表） | 成员看到分诊（`t6a-mut-member-triage`）；访客提交者有删除（`t6a-mut-guest-delete`）；已处理有删除（`t6a-mut-closed-delete`）；已归档有分诊（`t6a-mut-archived-triage`） | T6a |
| 可改的规则 | 非提交者的成员可改（`t6a-mut-member-edit`）；已归档可改（`t6a-mut-archived-edit`） | T6a |
| 页签与选中在地址 | 选中存进 context（`t6b-mut-selection-context`，后退的一格失败） | T6b |
| 自动选中是 `replace` | 自动选中 `push`（`t6b-mut-autoselect-push`） | T6b |
| 换页签复位筛选 | 页面状态不以页签为 key（`t6b-mut-state-not-keyed`） | T6b |
| 收集箱关着不读列表 | 列表的门不看开关（`t6b-mut-list-gate-switch`） | T6b |
| 列表的取数只读键 | fetcher 闭包当前的筛选（`t6b-mut-fetcher-closure`） | T6b |
| 列表读失败可重试 | 失败显示占位（`t6b-mut-failure-loader`） | T6b |
| 只有选中的“未处理”显示数 | “已处理”也显示（`t6b-mut-closed-total`） | T6b |
| “Add work item” 只在开着、未归档、有角色时 | 已归档也给（`t6b-mut-add-on-archived`） | T6b |
| ↑、↓ 只在编辑器外 | 编辑器里也换（`t6b-mut-arrows-in-editor`） | T6b |
| 提交写后重取、不进项目列表（R30） | 提交之后不重取（`t6c-mut-submit-no-refetch`）；提交经 `confirmCreation`（`t6c-mut-submit-creation`） | T6c |
| 没选日期不带 `target_date`（C10） | 发 `target_date: null`（`t6c-mut-target-null`） | T6c |
| 成功跳到新条目 | 读旧回答的 `issue.id`（`t6c-mut-nav-old-shape`） | T6c |
| 每个筛选的查询精确 | 负责人发成重复参数（`t7-mut-repeat-param`） | T7 |
| 状态至少一个（R9） | 允许清空（`t7-mut-empty-status`） | T7 |
| 日期区间的形状与本地日（C3、R10） | “之后”发成 `..a`（`t7-mut-after-range`）；预设按 UTC 的今天（`t7-mut-utc-today`） | T7 |
| 已选筛选的移除 | 移除按钮是 `submit`（`t7-mut-chip-submit`） | T7 |
| 接受被拒不发 `PATCH /issues`（R12） | `beforeFormSubmit` 吞掉拒绝（`t8-mut-accept-swallow`） | T8 |
| 接受的移选中在弹窗的提交落定之后（re-94 I2） | 在 `beforeFormSubmit` 里移选中（`t8-mut-move-in-before-submit`）；修改被拒也移（`t8-mut-move-on-update-refusal`） | T8 |
| 状态从默认状态起 | 初值留条目的状态（`t8-mut-state-triage`） | T8 |
| 再次提交只发修改（R29） | 不看是否已接受（`t8-mut-retry-triage`） | T8 |
| 冲突之后接受弹窗关上 | 冲突之后弹窗开着（`t8-mut-conflict-keeps-modal`） | T8 |
| 暂缓最早明天、那天的第一个时刻（R13） | 今天可选（`t8-mut-snooze-today`）；发那天的 UTC 零点（`t8-mut-snooze-utc`） | T8 |
| 选择器以 prop 收 id、不列自己 | 回到 `useParams`（`t8-mut-picker-params`）；列出条目自己（`t8-mut-picker-lists-self`） | T8 |
| 之后的选中（R14） | 标重复也移（`t8-mut-move-on-duplicate`）；被拒也移（`t8-mut-move-on-refusal`）；选中已换仍移（`t8-mut-move-stolen`）；移选中用 `push`（`t8-mut-move-push`） | T8 |
| 焦点的落点（R35） | 标重复成功之后焦点不管（`t8-mut-duplicate-focus-body`）；暂缓成功之后焦点给旧按钮（`t8-mut-snooze-focus-opener`） | T8 |
| 拒绝都提示 | 绕过 `useRefusalToast`（`t8-mut-no-toast`） | T8 |
| 在途锁 | 在途能关弹窗（`t8-mut-close-in-flight`） | T8 |
| 访客提交者属性只读 | 访客提交者可改优先级（`t9-mut-guest-props`） | T9 |
| 已处理只读 | 拒绝的可改（`t9-mut-closed-edit`） | T9 |
| 状态一栏只在接受时 | 收集箱中也显示（`t9-mut-state-shown`） | T9 |
| 重复于一次读、不碰项目列表（re-94 m3） | 不读、直接显示 id（`t9-mut-duplicate-unread`）；经 `refreshAuthoritative` 读（`t9-mut-duplicate-verifies`，项目列表带上错误的一格失败） | T9 |
| 动态按名字与通用键 | 按数字分支（`t9-mut-activity-numbers`）；未知的值不显示（`t9-mut-activity-unknown`） | T9 |

## Tasks

### T1 基线与文档（主控）

- **提交**：spec、plan、预检 re-94（含 §6 的处理）。
- **P4 的回填**（工作树里已有主控的改动，T1 核对后一起提交，spec R1）：`reviews/P4-web-notifications-review.md` 的状态行、A8、§5 的最终分支 CI 45048776015 与 main CI 45051719903、§9；数字取自归档台账 `.superpowers/sdd/M7-P4-integration-20261123/payload/worktree/.superpowers/sdd/P4-web-notifications/progress.md:602`。re-94 O4：§5、§9 的 main CI 一行里 “server (0) Test 869、server (1) Test 1002” 两个数在归档台账里没有出处，以 `gh run view 45051719903` 补出处或删去。
- **设计**：§12 的 P5a、P5b 两行与 §14 的 P5a 一行由主控改（re-94 m8），实现代理不碰设计。
- **关键词**：`K:2` 的 `phase` 为 `M7/P5`（已在工作树），跑 `node tools/keywords.mjs`（M7 的例外都是 `until: "M7/closeout"`，没有到期的）。
- **台账工具**：`final.sh` 已跑四个页面配置（m7-pages、m6-pages、m6-acceptance、m4-detail），只核对，不改。
- **依赖**：在根目录跑一次 `pnpm install --frozen-lockfile`（收据 `t1-install`）。
- **检查**：文档链接（`doc-links.py`）与关键词，然后提交 `docs(M7/P5a): …`。

### T2 待处理数、入口规则与挂载清单 → **检查点 0**

**Files**：
- 新：`C/services/issue/intake.service.ts`（本 Task 只有 `counts`）与测试；`C/store/issue/intake/intake.store.ts`（本 Task 只有计数）与测试；`C/store/issue/intake/fake-intake.ts`（本 Task 只有 `countsOf`）；`C/hooks/store/use-intake.ts`；`C/hooks/use-intake-counts.ts` 与测试；`C/lib/intake-entry.ts` 与测试；命令面板与页签栏的成员一格的新测试文件。
- 改：`C/store/issue/root.store.ts`（`intakeApi`、`intake`；`RootStore` 的数不变）；`C/components/workspace/sidebar/projects-list.tsx`（调一次 `useIntakeCounts`）；`C/components/workspace/sidebar/project-navigation.tsx`（收集箱一项的 `shouldRender` 经 `intakeEntryOn`，显示数）；`C/components/navigation/use-navigation-items.ts`（`:83`）；`C/components/power-k/config/navigation/project-commands.ts`（`:200-201`）；en、zh-CN 的 `inbox_issue.a11y.pending_count`；`.oxlintrc.json`（swr 名单）；`K`（`workspace-address-route`）；`e2e/fixtures/mounts.ts`（`INTAKE_COUNTS` 与头注释）。

**Steps**：
- [ ] Step 0：死成员基线。把 P4 归档的 `deadsym.mjs`、`domains.mjs` 与 `lib/` 从 `/Users/xiaoruan/project/nerve-project/.superpowers/sdd/M7-P4-integration-20261123/payload/worktree/.superpowers/sdd/P4-web-notifications/dead-members/` 复制进台账 `dead-members/`；先 `pnpm --filter web exec react-router typegen`，再 `deadsym.mjs --tsv > dead-before.tsv`、`domains.mjs dead-before.tsv --rows M7`（收据 `t2-dead-before`）。核对 P5 的 39 行（P4 阶段记录 §5 `:112`，对照 P4 的 `rows-M7-t15.tsv`），台账另记一段。
- [ ] Step 1：先写测试（`@/lib/auth/fake-nerve` 的 `FakeNerve`、`json`、`problem`；`@/store/fake-queue`），看到它们因所测性质失败：
  - service：`counts` 的路径；拒绝成 `ApiError`。
  - store：按工作区 id 存、经 slug 读；不在调用者工作区列表里的 slug 答 `undefined`；回答里没有的项目答 `undefined`；两个工作区互不影响；较旧的取数被超过时不写；`SessionChangedError` 给 `undefined`。
  - hook 的门与键：路由的工作区为 `null`，或没有一个项目过门（关着、已归档、`member_role` 为 `null` 各一格）时键为 `null`、不发请求；访客的项目过门；键带过门项目的 id，顺序与项目列表的顺序无关；一个项目的开关打开时键变、重读。
  - `intakeEntryOn`：管理员、成员、访客为真；没有角色、关着为假；已归档为真。
  - 侧栏：数大于 0 时显示、名字是 `inbox_issue.a11y.pending_count`；0 与 `undefined` 不显示；没有角色时没有收集箱一项。
  - 页签栏与命令面板：成员与访客各一格为真、没有角色为假（新文件；`commands.test.tsx` 不改）。
- [ ] Step 2：实现到通过。核对 `mounts.ts` 的八个读者只经常量跟随；`projects-list-item.tsx` 不动。
- [ ] Step 3：变异照上表 T2 各行。`make lint-web`、聚焦 vitest、`make knip`、`make test-web`。提交 `feat(M7/P5a): 收集箱的待处理数与入口规则——有开着收集箱的项目才读，侧栏、页签栏、命令面板一条规则`。
- [ ] Step 4（**检查点 0**）：`final.sh cp0`；主控推送、核分支 CI，台账记基线（见 Global Constraints）。规划与项目的故事有几处经接口打开收集箱（`e2e/fixtures/furnish.ts:134-150`、`:168-180`）：任何一条红先停下，查是不是精确请求表多了 `INTAKE_COUNTS`。

### T3 筛选的键与查询、列表的读

**Files**：新 `C/store/issue/intake/filters.ts`（`IntakeFilters`、`filterKey`、`listQuery`）与 `filters.test.ts`；`intake.service.ts` 加 `list` 与测试；`intake.store.ts` 加列表（`lists`、`fetched`、`getList`、`fetchList`、`fetchMore`）与测试；`fake-intake.ts` 加 `intakeIssueOf`、`pageOf`。`C/lib/reconciled.ts` 不改（`fetchPart` 由 P4 加，`:167-175`）。

**Steps**：
- [ ] Step 1：先写测试，看到失败：
  - service：`list` 的路径；数组经 CSV 成一个参数（`status=pending,snoozed`）；`limit` 是数字。
  - `filterKey`：与数组的先后无关、去重。
  - `listQuery`（spec §2.2 的 C1–C6，每条一格）：两个页签都带 `status`，含等于默认值时；空数组不带；日期只一个区间、四种形状；`order_by` 只取六个值之一；`limit: 50`；`cursor` 只在给了时带。
  - store：
    - 第一页按 `${projectId}|${filterKey}` 存；
    - `fetchMore` 以持有页的 `next_cursor` 发；回答到时持有页的 `next_cursor` 仍是发出的那个才接上，接上本页没有的行、`next_cursor` 与 `total` 取回答的；没有 `next_cursor` 时不发；
    - 在外的 `fetchMore` 被 `fetchList` 作废，什么都不写（P4 I1，`3f4c3ba8`）；
    - 刷新之后别处移走的行不在了；
    - A 项目迟到的回答不进 B 项目的列表与游标；
    - `SessionChangedError` 不写。
- [ ] Step 2：实现到通过。
- [ ] Step 3：变异照上表 T3 各行。门禁同 T2（不含检查点）。提交 `feat(M7/P5a): 收集箱筛选的键与查询、列表的读——每个页签显式发状态，下一页认游标`。

### T4 store 的确认

**Files**：`filters.ts` 加 `matches` 与测试；新 `C/store/issue/intake/status.ts`（`effectiveStatus`、`rowOf`）与测试；`intake.store.ts` 加 `confirm`、`drop`、`countAgain` 与测试（`intake.store.ts` 过 300 行时把这三个拆到 `intake-changes.ts`，先例 `notification-changes.ts`）；`fake-intake.ts` 加 `detailOf`。

**Steps**：
- [ ] Step 1：先写测试（`fake-queue.ts` 的 `inTurn`、`fetchedWhileChangeIsOut`），看到失败：
  - `effectiveStatus`：到期的暂缓是待处理，没到期的是暂缓，`now` 恰在 `snoozed_till` 一格。
  - `matches`：每个筛选维度各一格（状态用有效状态；`me` 与 `none`；日期按 UTC 日、首尾都含）。
  - `rowOf`：取 `IntakeIssue` 的键；`intake` 为 `null` 时没有行。
  - `confirm`：写进这个项目持有的每一份列表（默认的与另一组筛选各持一份）；相符则原地替换、不相符则移出；**不插入**到原来没有它的列表；移出时 `total` 减一，替换不变；别的项目、别的工作区的列表不动；取数在外时确认的修改重放到它的回答上、只一次。
  - `drop`：从每一份移出，`total` 同上。
  - `countAgain`：只重取这一代已经取过的那个工作区的计数，没有持有的计数时不发请求；不占队列；失败留着原值。
- [ ] Step 2：实现到通过。
- [ ] Step 3：变异照上表 T4 各行。门禁同 T3。提交 `feat(M7/P5a): 收集箱 store 的确认——写进持有的每一份，只替换或移出，不插入`。

### T5 写的接缝：M4 的逐工作项写队列（spec §2.4，R3、R4、R33、R34，高风险）

两个提交。**先纯搬移，再加去处**；两步之间跑全部既有测试。

**Files**：
- Step 1：新 `C/store/issue/issue-write-confirm.ts`；`C/store/issue/issue-writes.ts`（搬走 `:44-54` 与 `:100-227`）；`K`（`project-address-route` 的文件名单与 hit 样例收入新文件）。
- Step 2 起：`intake.service.ts` 加 `triage`、`update`、`delete` 与测试；`status.ts` 加 `inIntake`；`issue-write-fields.ts` 加 `intakeUpdateBody` 与测试；`issue-write-confirm.ts` 加收集箱的去处（与 `write()` 并列，不经两段目录准备）；`issue-writes.ts` 加 `triageIntake`、`deleteIntake`，`updateIssue` 在轮到时选端点；`C/store/issue/issue.store.ts`（`choose` 的一种情形，R33）；`root.store.ts` 的 `WritesRoot` 收入 `intake`、`intakeApi`；新测试文件 `C/store/issue/intake/intake-writes.test.ts`；`issue-changes.test.ts`（允许清单的一项）。

**Steps**：
- [ ] Step 1（纯搬移，`refactor`）：把 `write()` 与 `pending`、`pendingIDs`、`observePendingRead` 移进 `issue-write-confirm.ts` 的一个类，`IssueWrites` 持有它并转发两个公开成员；公开的名字与签名不变；不改一行逻辑。
  - 收据：`make lint-web`；`issue-writes.test.ts`、`issue-changes.test.ts`、`issue-drag.test.ts` 与 `C/store/issue/issue-details/` 下读 `root.writes` 的测试（收据 `t5-move-tests`），再 `make test-web`；`wc -l` 两个文件（收据 `t5-move-wc`）。
  - 提交 `refactor(M7/P5a): issue-writes.ts 按关注点纯搬移——确认进持有的每一份搬到 issue-write-confirm.ts，命令与队列留下`。
- [ ] Step 2：先写测试，看到失败（每格先确认它因所测性质失败，不因编译）：
  - `intakeUpdateBody`：七个字段原样；别的字段有值即抛错；`skip_activity` 的规则由 `updateBody` 照旧。
  - `inIntake`：`undefined` 为假；`intake` 为 `null`、已接受为假；待处理、暂缓、拒绝、重复为真。只持有行的工作项的修改走 `PATCH /issues`。
  - **目录准备（re-94 S1）**：持有一份带 `state_group` 筛选的项目列表与一份按状态组分组的工作区列表时，收集箱的内容修改、拒绝、删除各发出并确认；没有 `GET /api/v0/projects/{id}/states`；两份列表的值与 `error` 不变（删除也不把它们标成失败）。
  - 内容修改：所持详情在收集箱中时 `PATCH /api/v0/intake-issues/{id}`，请求体是 `intakeUpdateBody` 的；从不发 `PATCH /issues`。canonical 得到回答；收集箱的列表替换；**一份持有的项目列表不变**（没有这一行、计数不变）；不重取计数。
  - 版本恢复（`description-versions.store.ts` 的 `restore`）在收集箱条目上走收集箱的端点。
  - 端点在轮到时选：排在一个接受之后的修改发 `PATCH /issues`（接受的回答扣住时排上修改，再放开）。
  - 与评论的写一个接一个：同一条目上一个评论的写在外时，`triageIntake` 不发出。
  - **分诊在轮次里重读（R34）**：每种分诊做成之后同一轮次里有 `GET /api/v0/issues/{id}`；一个在分诊之前发出、在确认之后到达的详情读（同一 `updated_at`，`intake` 为待处理）不把 canonical 的 `intake` 改回去；命令在重读落定之后才落定。
  - 接受：`POST …/triage` `{status:"accepted"}`；一份持有的项目列表在确认之后有它（重读的回答）、只一次，不重读列表；取数在外而回答已含它时最后仍只一次；**列表的读在接受发出之后发出、在回答之前落定**时也只一次（来源在分诊发出之前冻结，`issue-writes.ts:141`）；收集箱的列表按筛选替换或移出；重取计数。
  - 拒绝、暂缓、标重复、回到待处理：项目列表不变；收集箱的列表按筛选替换或移出；重取计数。
  - 重读的结局：给 `undefined` 时从收集箱的每份列表移出、项目列表不确认、命令不失败；失败（500）时命令不失败、项目列表不确认、错误留在这个工作项的核验上；`SessionChangedError` 照常交出。
  - 删除：`DELETE /api/v0/intake-issues/{id}`；canonical 的删除；收集箱的列表移出；重取计数；项目列表不变。
  - 拒绝：值不变、交出拒绝；`issue.intake_not_found` 时从收集箱的每份列表移出、重取计数再交出；`issue.intake_transition_not_allowed` 与 `issue.intake_closed` 时**在同一轮次里**以 `GET /api/v0/issues/{id}` 重读、重读的回答确认进收集箱的列表（`undefined` 时移出）、重取计数再交出；排在后面的同一条目的修改等这次重读落定，不以 `Issue has not been read` 失败；重读之后再改走 `PATCH /issues`（别人接受了的情形）。
  - **行盖在详情上（R33）**：所持详情的 `intake` 为待处理时，一个不带 `intake`、`updated_at` 更新的行落下，`getIssueDetailById(id).intake.status` 是已接受，下一次修改走 `PATCH /issues`；`intake` 为 `null` 或已接受时合并照旧。
  - 权限与换代：`issue-changes.test.ts` 的 `"triage"` 一格（原 `"inbox"`）三条 `it.each` 的断言照旧（允许清单）；`SessionChangedError` 时什么都不写。
- [ ] Step 3：实现到通过。`issue-writes.ts` 不超过 220 行（收据 `t5-wc`）。台账核对 T4 的 `confirm`、`drop`、`countAgain` 已有产品读者。
- [ ] Step 4：变异照上表 T5 各行。`make lint-web`、聚焦 vitest、`make knip`、`make test-web`；因 R3 的风险另以 `playwright.m4-detail.config.ts` 在 chromium 跑一次（先 `make build`，收据 `t5-m4-detail`）。提交 `feat(M7/P5a): 收集箱条目的写经 M4 的逐工作项写队列——端点轮到时按所持详情选，不经目录准备，分诊在轮次里重读，接受像新建一样进项目列表`。

### T6a 详情侧：新的详情、页头与动作模型，删除旧的详情树

**一个提交**，自身全绿（含 knip）。旧的 `IN/root.tsx` 与旧侧栏照旧，渲染新的 `InboxContentRoot`（同名导出、收旧根给的 props）；它们的旧读仍是 404，新的详情只按自己的读决定选中，不会与旧根互相导航。

**Files**：
- 新：`IN/content/{index,root,detail,issue-properties}.tsx` 与测试（`root.tsx` 是详情区域：有名字、可聚焦，画在 `IssueDetailHost` 之外，分诊的弹窗与读失败的提示也在那里）；`IN/content/header/{root,mobile,status-chip}.tsx`、`IN/content/header/use-intake-actions.ts` 与测试。
- 改：四个弹窗（`IN/modals/{decline,delete,snooze}-issue-modal.tsx`、`select-duplicate.tsx`）只改接线到 `triageIntake`、`deleteIntake`；`C/components/issues/issue-detail/label/root.tsx`（删 `isInboxIssue`）；`C/components/issues/fake-issue-properties.tsx`（删 `:30-32`）；`web/packages/constants/src/intake.ts`（加状态名的键表）。
- 删除：`IN/content/` 的旧文件（含 `description-versions.tsx`、`inbox-issue-header.tsx`、`inbox-issue-mobile-header.tsx`、`issue-root.tsx`）；`C/services/inbox/` 的版本 service、它的 DTO 与测试和 `index.ts` 的那一行；三个 `inbox_issue.errors.*_permission` 键。
- 守卫：`K`（`copy-hook` 例外删除；`project-address-route` 收入读项目的新文件，带样例）；`.oxlintrc.json`（swr 名单加新的取数文件、删 `:165`）。
- 文案：en、zh-CN：详情区域的名字 `inbox_issue.a11y.detail`、读不到的提示等。

**Steps**：
- [ ] Step 1：先写组件测试（jsdom，`@/lib/fake-session-swr`、`@/hooks/store/fake-store-hooks`、`@/lib/fake-refusal`），看到失败：
  - 详情的读：在 `IssueDetailHost` 之内、以条目 id 为 key；键是 spec §2.5 的；`revalidateOnMount: false`，每次挂载以 `mutate()` 读一次。
  - 读不到（re-94 m4）：SWR 对这个键留着上一次的 404 时，新挂载先读、读落定为 404 才以 `replace` 清掉选中并提示一次；回答的 `intake` 为 `null` 同样；网络失败或 500 不清选中，详情区域给可重试的失败态，重试再读。
  - 详情区域与弹窗：分诊的重读在途（canonical 给 `undefined`）时详情区域与开着的弹窗仍在。
  - 描述版本只给管理员与成员；可改的规则（spec §2.7）：管理员、提交者、待处理与暂缓、未归档、开着；非提交者的成员、已处理、已归档只读。
  - 页头：动作矩阵（spec §2.8 的表）逐格，与每个状态的主动作；复制经 `useCopyLink`（成功与失败各一格）；窄屏的页头读同一个动作模型；读路由参数（router store 指向别的项目的一格）。
- [ ] Step 2：实现新的详情侧；删除旧的详情树与随之失去读者的文件、导出、键与例外。
  - knip：`make knip` 的收据。
  - 死成员：重跑 T2 Step 0 的脚本（收据 `t6a-dead-after`），比对 `dead-before` 与 `dead-after` 的**整份**差异：消失的行逐条对上删掉的文件；**新出现的每一行**都要有去处（同一提交删掉，或写明留用与它的读者；旧 store 失去读者的成员留到 T6c 删，逐条列出），台账列表。
  - 告警：`oxlint` 计数的收据（`t6a-oxlint-count`），与上限相等。
- [ ] Step 3：变异照上表 T6a 各行。`make lint-web`、`make knip`、`make test-web`；`node tools/keywords.mjs` 的绿收据。提交 `feat(M7/P5a): 收集箱的详情侧——详情复用工作项详情、读不到才清选中、页头按关注点拆开与动作模型，删除 Plane 的旧详情树`。台账核对 T5 的两个命令已有产品读者。

### T6b 列表侧：路由、页面状态、侧栏与页头，删除旧的列表树

**一个提交**，自身全绿（含 knip）。

**Files**：
- 新：`IN/page-state.tsx`（以页签为 key 的筛选与排序；`defaultFilters` 的第一个读者）与测试；`IN/use-intake-list.ts`（列表的 `useSessionSWR`，fetcher 经 `filtersOf` 从键得到筛选）；`IN/root.tsx`（改写：布局、窄屏的开关按钮、自动选第一行）与测试；`IN/page-header.tsx`（从 `C/components/projects/settings/intake/header.tsx` 移来）；`IN/sidebar/{root,tabs,list,list-item}.tsx` 与测试；`IN/content/header/navigation.tsx`（上一条、下一条与 ↑、↓）与测试。
- 改：`A/projects/(detail)/[projectId]/intake/{layout,page}.tsx`；`web/packages/constants/src/intake.ts`（加页签的选项）。
- 删除：`IN/inbox-filter/` 整棵（十七个文件，含 `filter-selection.tsx` 与 `applied-filters.test.tsx`）、`IN/sidebar/` 的旧文件、`IN/inbox-issue-status.tsx`、`C/components/projects/settings/intake/header.tsx`。`IN/inbox-status-icon.tsx` 与两个 loader 留用或改写，台账写明。
- 守卫：`K`（`project-address-route`、`workspace-address-route` 收入新文件，带样例）；`.oxlintrc.json`（swr 名单）。
- 文案：en、zh-CN：上一条、下一条、窄屏开关、页面标题。

**Steps**：
- [ ] Step 1：先写组件测试，看到失败：
  - 页面状态：卸下再挂上回到默认；换页签回到那个页签的默认；未知的 `currentTab` 按“未处理”。
  - 列表：键带 `identity` 与 `filterKey`，换筛选即换键；fetcher 只读键；收集箱关着时不取（停用页）；首读的占位；读失败给可重试的 `PlanningReadFailure`；每个页签的空态；滚到哨兵时 `fetchMore`，没有 `next_cursor` 不取。
  - 页签：Headless `Tab`、`aria-selected`；选中的“未处理”显示 `total`，“已处理”不显示。
  - 选中：没有选中而有行时以 `replace` 选第一行；点行是 `push` 的导航。
  - 页头：“Add work item” 只在开着、未归档、有项目角色时给；面包屑经 `t()`。
  - 上一条、下一条的名字经 `t()`；↑、↓ 只在焦点不在编辑器与输入框时换。
  - 页面与页头读路由参数（router store 指向别的项目的一格）。
- [ ] Step 2：实现新的列表侧；删除旧的列表树与随之失去读者的文件。knip、死成员整份比对（收据 `t6b-dead-after`）、告警计数（`t6b-oxlint-count`）同 T6a。
- [ ] Step 3：变异照上表 T6b 各行。门禁同 T6a。提交 `feat(M7/P5a): 收集箱的列表侧——页签与选中在地址、筛选在页面状态、侧栏与页头，删除 Plane 的旧列表树`。台账核对 T3 的 `getList`、`fetchList`、`fetchMore` 已有产品读者。

### T6c 旧数据层的删除与新建弹窗的提交

**一个提交**：守卫先看红（只留收据，不提交），与让它变绿的删除、改写同一提交。

**Files**：
- 改：`intake.service.ts` 加 `submit` 与测试；`intake.store.ts` 加 `submit` 与测试；`IN/modals/create-modal/create-root.tsx`（spec §2.10：提交、请求体、跳转、失败的提示）；`IN/modals/create-modal/issue-description.tsx`（`:21`、`:43`、`:48` 起的旧读与占位）；`create-root.request.test.tsx`、`create-root.files.test.tsx`（允许清单）；`C/store/root.store.ts`（删 `projectInbox`）、`root.store.test.ts`（21 → 20）；`web/packages/constants/src/intake.ts`（删旧导出）。
- 删除：spec §2.10 的 T6c 一组（`C/store/inbox/`、`C/services/inbox/` 余下的、`C/hooks/store/use-project-inbox.ts`、`web/packages/types/src/inbox.ts` 与 barrel 一行、`TPaginationInfo`、`TLegacyIssueUpdateRequest`、`web/packages/utils/src/intake.ts` 与 barrel 一行）。
- 守卫：`K` 加 `old-intake-urls`（spec §2.11 的模式与样例）。

**Steps**：
- [ ] Step 1：守卫先看红，不提交：加 `old-intake-urls`，看到 `node tools/keywords.mjs` 因旧 service 与两份测试（`create-root.files.test.tsx:145`、`create-root.request.test.tsx:142`）命中而失败（收据 `t6c-red-old-urls`）。
- [ ] Step 2：先写测试，看到失败：
  - service：`submit` 的路径与 201。
  - store：`submit` 成功之后重取这个项目持有的每一份收集箱列表的第一页并 `countAgain`；不往任何列表插行；一份持有的项目列表不变；被拒时值不变、交出拒绝。
  - 新建弹窗（允许清单的改法）：请求是 `POST /api/v0/projects/{project_id}/intake-issues`，请求体照 `IntakeIssueCreate`，没选日期时没有 `target_date`（C10，一格）、选了时是日期串；成功之后跳到 `?currentTab=open&inboxIssueId=<回答的 id>`、关闭；拒绝经 `useRefusalToast` 提示、没有 `console.error`、没有成功的动作；发出的描述不含图片、没有文件请求、生成客户端只有那一个 `POST`。
- [ ] Step 3：实现；删除旧数据层与三个包里随之失去读者的导出；`RootStore` 20；常量删旧。
  - knip、死成员整份比对（收据 `t6c-dead-after`，P5 的 39 行逐条有去处）、告警计数（`t6c-oxlint-count`）。
- [ ] Step 4：变异照上表 T6c 各行。`make lint-web`、`make knip`、`make test-web`；守卫的绿收据（`t6c-green-keywords`）。提交 `feat(M7/P5a): 删除 Plane 的收集箱 store、service、hook、类型、常量与工具，新建弹窗经新接口提交并跳到新条目，加 old-intake-urls`。

### T7 筛选与排序

**Files**：新 `IN/filters/{menu,applied,order-by}.tsx` 与测试；`filters.ts` 加日期区间的换算（预设与 `DateFilterModal` 的输出 → 合同的区间）与测试；`IN/sidebar/root.tsx`（接上筛选按钮与已选的一行）；`web/packages/constants/src/intake.ts`（排序与日期预设的选项）；en、zh-CN 的预设名字、筛选的名字。

**Steps**：
- [ ] Step 1：先写测试，看到失败：
  - 筛选按钮是有名字的按钮（`FiltersDropdown`，`HO/M3-closeout.md:48`），带 `aria-expanded`，Escape 关上、焦点回到按钮。
  - 每种筛选（状态、优先级、负责人、创建者、标签、创建日、更新日）选一个值之后的请求，断言精确的查询串；同一个键只出现一次。
  - 状态：“未处理”只给待处理、暂缓，“已处理”只给三个终态；最后一个不能取消（Plane `pw/core/components/inbox/inbox-filter/filters/status.tsx:44-47`）。
  - 负责人与创建者的选项来自路由项目的显式成员 id（`getProjectMemberIds`），不读 `projectMemberIds`。
  - 日期：四个预设按本地日写成日期串（`TZ=America/Los_Angeles` 一格，日界附近）；自定义的“之后”“之前”“区间”分别成 `a..`、`..b`、`a..b`；每个字段只一个区间。
  - 已选的一行：每个移除按钮以它移除的值命名、留下其余的值、不提交外层表单（接上 T6b 删掉的 `applied-filters.test.tsx:50` 的性质）；“全部清除”回到页签的默认。
  - 排序：字段与升降合成合同的值；默认 `-created_at`。
- [ ] Step 2：实现到通过。
- [ ] Step 3：变异照上表 T7 各行。门禁同 T3。提交 `feat(M7/P5a): 收集箱的筛选与排序——精确的查询、状态至少一个、日期按本地日的区间`。

### T8 分诊的规则

**Files**：新 `IN/modals/accept.tsx`（接受弹窗的接线）与测试；`status.ts` 加 `snoozeDay`、`snoozeInstant` 与测试；`IN/modals/snooze-issue-modal.tsx`、`select-duplicate.tsx`、`decline-issue-modal.tsx`、`delete-issue-modal.tsx`；`IN/content/header/use-intake-actions.ts`（之后的选中、在途锁、焦点的落点）；`select-duplicate.test.tsx`（允许清单）；en、zh-CN（接受之后修改被拒的说明等）。

**Steps**：
- [ ] Step 1：先写测试，看到失败：
  - 接受（spec §2.8、R12、R29，re-94 I2）：
    - `beforeFormSubmit` 只发分诊；分诊被拒时弹窗不发 `PATCH /issues`、提示拒绝、弹窗开着；
    - 状态字段的初值是项目的默认状态；弹窗的 `data` 是打开时的那一份，分诊之后的重读不改它；
    - **接受并改了一个字段**：`PATCH /issues` 的请求体带那个字段；`PATCH` 确认、弹窗关上之后才移选中（`replace`），之前地址不变；
    - 修改被拒：错误在字段下，选中不动，弹窗开着并说明条目已被接受；
    - 再次提交不再分诊，只发修改；条目已被别人接受（所持详情不在收集箱）时首次提交也只发修改；
    - 分诊被冲突拒绝（`issue.intake_transition_not_allowed`、`issue.intake_not_found`）：提示、弹窗关上、选中不移（`intake_not_found` 时随读不到清掉）。
  - 暂缓（R13）：日历最早明天、默认明天；`snoozeInstant` 是那天的第一个本地时刻（`TZ=America/Los_Angeles` 一格；午夜被跳过的时区一格，如 `America/Santiago` 的拨快日，日期以 `Intl` 核对）；请求体只带 `status` 与 `snoozed_till`；标题经 `ModalTitle`；`must_be_future` 在字段下。
  - 取消暂缓发 `{status:"pending"}`；改暂缓再选一天。
  - 标重复：选择器以 prop 收条目与项目的 id（地址的参数不同的一格）；不列条目自己；请求体只带 `status` 与 `duplicate_to_id`。
  - 删除：按 spec §2.8 的表；成功以 `replace` 清选中。
  - 之后的选中（R14）：接受、拒绝、暂缓、改暂缓、取消暂缓确认之后以 `replace` 到动作开始时那一行的下一行，没有则上一行，都没有则清；标重复不移；被拒不移；等待时选中已换则不移；后退不回到刚分诊的条目。
  - **焦点的落点（R35）**：移到邻行 → 那一行；清空 → 列表；标重复成功（选中不移、打开者不在）→ 详情区域（已处理没有主动作）；暂缓、改暂缓成功（R14 移选中）→ 邻行，没有邻行 → 列表；冲突之后 → 新页头的主动作（管理员面对待处理时是“接受”）或详情区域；等待时选中已换 → 焦点不动；弹窗取消 → 打开它的按钮；从窄屏菜单打开的弹窗取消 → 菜单的按钮；被拒 → 弹窗里。任何一格都不是 `document.body`（jsdom 只钉落点，真实焦点由 H20 (page) 三引擎核）。
  - 在途：按钮停用；弹窗不能关。
  - 每个拒绝经 `useRefusalToast`；没有 `console.error`。
- [ ] Step 2：实现到通过。
- [ ] Step 3：变异照上表 T8 各行。门禁同 T3。提交 `feat(M7/P5a): 分诊——只有管理员，接受弹窗的提交落定之后才移选中，暂缓最早明天，选择器收 id，焦点有确定的落点`。

### T9 属性、“重复于”与动态 → **检查点 1**

**Files**：`IN/content/issue-properties.tsx`；新 `IN/content/duplicate-of.tsx` 与测试；`C/components/issues/issue-detail/issue-activity/activity/actions/inbox.tsx`（按状态名）与新测试；`activity-list.tsx`（删 `:86` 的 `case "inbox":`）；en、zh-CN 的 `inbox_issue.activity.*`；旧文案键的清点。

**Steps**：
- [ ] Step 1：先写测试，看到失败：
  - 属性：角色 × 状态的矩阵（管理员、成员提交者、访客提交者、非提交者的成员；待处理、暂缓、接受、拒绝、重复；已归档）：访客提交者只读；已处理只读；状态一栏只在已接受时显示；负责人的下拉框可用键盘打开。
  - 重复于（re-94 m3）：`duplicate_to_id` 不为 `null` 时经 `fetchIssueCore` 一次 `GET /api/v0/issues/{id}`，显示编号与名字、链到它；读被拒时不显示这一行，**一份持有的项目列表没有错误、没有陈旧的格**；`null` 时不读。
  - 动态：五个状态名各一格（en 与 zh-CN 的键都在），一个不在五个里的值用通用的键（re-94 O1）；不再按数字。
- [ ] Step 2：实现到通过；删去不再有读者的旧键（Global Constraints 的“文案”）。
- [ ] Step 3：变异照上表 T9 各行。门禁同 T3。提交 `feat(M7/P5a): 收集箱详情的属性按角色与状态、“重复于”经一次读、动态按状态名`。
- [ ] Step 4（**检查点 1**）：`final.sh cp1`；主控推送、核分支 CI，记 web 作业三步与 e2e 两片的时长（越线时照 Global Constraints 的判别法）。

### T10 e2e 的夹具与 H9 (page)

**Files**：新 `e2e/fixtures/intake-pages.ts`（世界：管理员、成员、访客与只看自己的访客，开着收集箱的项目；地址；定位器；共用的帮手从 `cycle-pages.ts`、`notification-pages.ts`（`person`、`zonedDay`、`zonedMoment`）、`settings-pages.ts`、`browser.ts`、`dialogs.ts`、`auth.ts` 导入）；新 `e2e/fixtures/assert/intake.ts`（条目行、动态行与通知行的只读查询）；新 `e2e/stories/collaboration/h9-intake-list-page.spec.ts`。

**Steps**：
- [ ] Step 1：写故事（spec §2.12 的 H9 一行），至少三个 `test()`，标题都带 `H9 (page)`（re-94 m9）：
  - ①空的收集箱：直接打开时断言精确的请求表（`APP`、`WORKSPACE`、`PROJECT`、`PROJECT_MEMBER`、`INTAKE_COUNTS` 与列表的读；没有行就没有自动选中，没有详情的读）；访客与成员的侧栏入口；关着的项目：停用页、没有列表的读。
  - ②51 条经接口提交（每页 50）：滚到底才有第二次读，第 51 条出现；“未处理”的总数；自动选中第一行之后详情的读以 `expect.poll` 等到（不并进精确的表）；只看自己的访客的条目由他作为成员时提交、再降为访客；侧栏的数在一次分诊前后；应用内跳转之后用 `expect.poll` 等内容，不数计数的请求（P4 re-93 m12）。
  - ③每种筛选与排序各一次，断言精确的查询。

  先跑一个期望错误的版本，看到断言失败，再改正。
- [ ] Step 2：`make build build-s3-fixture`，聚焦 chromium（`cd e2e && NERVE_VERSION=0.1.0-dev pnpm exec playwright test --project=chromium stories/collaboration/h9-intake-list-page.spec.ts`）；再 `--config=playwright.m7-pages.config.ts` 三种引擎，再 `--repeat-each=3`；台账记各引擎每个 `test()` 的时长（任何一个超过 30 秒先报主控）；`make lint-web`。提交 `test(M7/P5a): 收集箱页的夹具与 H9 的页面故事`。

### T11 H10 (page)

**Files**：新 `e2e/stories/collaboration/h10-intake-triage-page.spec.ts`；夹具按需。

**Steps**：
- [ ] Step 1：写故事（spec §2.12 的 H10 一行）：
  - 先在应用内打开项目的工作项列表、再到收集箱，经弹窗接受并改一个字段（优先级）：断言 `PATCH /issues` 的请求体带它；弹窗关上之后地址才换到邻行；回到列表时条目在、没有刷新（R4）；一条状态动态与关注者的通知行（`assert/intake.ts`、`assert/notification.ts`）；
  - 接受被拒（re-94 m9）：弹窗开着时另一个管理员经接口先接受，再提交：没有 `PATCH /issues`，提示，弹窗关上，页头随重读成为已接受；
  - 暂缓以 `test.use({ timezoneId: "America/Los_Angeles" })` 定时区，期望的 `snoozed_till` 按这个时区的明天 00:00 算；改暂缓与取消暂缓各一次；
  - 拒绝；经选择器标重复（选中不移）；
  - 成员与访客没有分诊控件；
  - 项目列表里没有未接受的条目（spec §6）。

  先看到错误的期望失败，再改正。
- [ ] Step 2：同 T10 的跑法与时长记录；提交 `test(M7/P5a): H10 的页面故事`。

### T12 H11 (page) 的文字内容与边界

**Files**：新 `e2e/stories/collaboration/h11-intake-content-page.spec.ts`；夹具按需。

**Steps**：
- [ ] Step 1：写故事（spec §2.12 的 H11 一行）：
  - 每个角色改名字与描述，按 spec §2.7 的规则看到可改或只读；访客提交者改名字之后请求体只有名字；
  - 管理员恢复一个描述版本，请求是 `PATCH /api/v0/intake-issues/{id}`；访客看不到版本；
  - 评论与表情；“重复于”一行；按角色的删除；项目列表里没有未接受的条目；
  - 描述不带图片（M6-M8-07）。

  先看到错误的期望失败，再改正。
- [ ] Step 2：同 T10；提交 `test(M7/P5a): H11 的页面故事（文字内容与边界）`。

### T13 H20 (page) 的收集箱页 → **检查点 2**

**Files**：新 `e2e/stories/collaboration/h20-intake-pages.spec.ts`，过 400 行时按关注点拆出 `h20-intake-late-pages.spec.ts`，标题同为 `H20 (page)`；夹具按需。

**Steps**：
- [ ] Step 1：写故事（spec §2.12 的 H20 一行）：
  - 迟到的回答用 `sentHeld`/`holdAnswer`；换账号用 `anotherTabSignsIn`；换工作区、换项目用 `moveWithinApp` 与浏览器的后退、前进；
  - 键盘：页签、筛选菜单、排序菜单、动作菜单、四个弹窗与选择器的 Tab、Enter、Space、Escape 与焦点的回去；↑、↓ 在描述编辑器里不换条目；
  - **焦点的落点（R35，re-94 I4）**：标重复、暂缓、改暂缓各走三条路——键盘打开一次、指针打开一次、窄屏（< lg）页头的菜单打开一次；成功与取消各看一次焦点：成功的那次扣住回答，放行之后等页头重画（`expect.poll` 焦点所在的元素）再断言落点（spec §2.8 的表）；取消回到打开它的按钮或菜单的按钮；任何一次都不是 `body`；三种引擎；
  - 窄屏的开关按钮与页头；
  - 剪贴板用 `refuseClipboardWrites`；弹窗用 `dialogs.ts`；列表在按钮旁用 `expectListBesideButton`。

  先看到错误的期望失败，再改正。
- [ ] Step 2：同 T10 的跑法与时长记录；提交 `test(M7/P5a): H20 的收集箱页`。
- [ ] Step 3（**检查点 2**）：
  - `final.sh cp2`：七项加四个页面配置，各留 JSON 报告。
  - 新文件 `--repeat-each=3`（收据 `cp2-repeat`）。
  - 照 spec §2.12 估出 e2e 两片的时长，估出超过 30 分钟先报告、不推送。
  - 主控推送、核分支 CI，记两片的实际时长与 web 作业三步。

### T14 文档与交接（主控）

- **plane-diff**：spec §7 逐条，补 `docs/v0/plane-diff.md:367` 的计数一行；行号对照主目录的只读 `plane/` 逐一核过。
- **frontend-changes**：3.1 的 M7 一行（收集箱的范围，P5b 余下的写明）；3.2 加行：收集箱的数据层、写的接缝（R3、R4、R33、R34）、收集箱页、入口规则、旧层的删除；“挂载时的取数”一行（`docs/v0/frontend-changes.md:233`）补“侧栏的待处理数由 M7/P5a 加回（有开着收集箱的项目时）”。
- **交接**：spec §5 的表，每条写“处理结果（M7/P5a）”；写关闭的照写关闭，其余保持 open。
- **死成员**：重跑脚本（收据 `t14-dead-final`），剩下的行写进阶段记录。
- **告警**：核对本段改到与新建的文件 oxlint 为零、计数等于上限。
- **设计**：§12 的 P5a 一行加 spec 链接，§14 的 P5a 一行（主控）。
- **阶段记录** `reviews/P5a-web-intake-review.md`：§1 方法与范围、§2 验收 → Task、§3 实施中的裁定（R3、R4、R33、R34 写明是动了 M4 共用代码的高风险局部裁定）、§4 向负责人说明（R8 与 P4 R6；spec §1 的“P5a 合入、P5b 之前 main 的样子”）、§5 门禁与 CI、§6 时长、§7 交给 P5b 与 M8、§8 文档、§9 审查与集成。
- 检查文档链接与关键词，提交 `docs(M7/P5a): …`。

### T15 门禁、整审与集成（主控）

1. **整分支审查**：一次 fresh 只读的整分支 deep-review（fable），重点是 Review Focus 的 1、2。Critical/Important 在一次修复轮里各有 RED/GREEN；Minor 记台账，主控裁定改否。之后一次限定复审。
2. **终检**：`final.sh fin`。
3. **推送前的估算**：照 spec §2.12 估出 e2e 时长。
4. **最终分支 CI**：推送准确的 HEAD，核分支 CI。
5. **合并**：`--no-ff` 合入 main，推送后以 `ls-remote` 核对。
6. **main CI**：核准确的 main CI。
7. **归档与清理**：归档 `.superpowers/sdd/M7-P5a-integration-<日期>`，清理 worktree 与分支。
8. **回填**：spec A9 与设计 §14 由 P5b 的 T1 回填。

## Review Focus

1. **写的接缝（R3、R4、R33、R34，高风险）**：`issue-writes.ts` 的拆分是纯搬移、既有测试一行未改；端点在轮到时按所持详情选，`inIntake(undefined)` 为假；收集箱的去处不经两段目录准备、不碰项目列表（`issueMembership` 不看 `intake`），删除不把列表标成失败；分诊在轮次里重读，旧读不改回 `intake`；接受经 `confirmCreation` 确认重读的回答、只出现一次；重读的三种结局；与同一工作项的别的写一个接一个；拒绝值不变，`issue.intake_not_found` 移出，冲突在轮次里重读；行盖在详情上不留未接受的 `intake`；三条 `it.each` 照旧；计数只在分诊、删除、提交之后、且不越过门重取。
2. **会话分代（v0 §7.7）**：客户端来自这一代；SWR 键带 `loginId`、工作区 id 与 slug（列表与详情另带 `identity`），fetcher 只读键；`fetchPart` 与认游标的合并；详情每次挂载读自己的；换代、换工作区、换项目后旧的回答不写进新页面；`await` 之后都经 `followInSession`。
3. **合同的陷阱**：每个页签都发 `status`、至少一个；空数组不发、不重复参数；日期一个区间、四种形状、本地日；`order_by` 的六个值；每页 50；游标只取回答的；有效状态；暂缓的时刻；内容修改只有七个字段；提交没有日期不带键。
4. **页面状态与地址**：页签与选中在地址，用户点行 `push`、动作引起的 `replace`；筛选与排序在页面状态、以页签为 key、离开即复位；详情以条目 id 为 key；读不到在自己的读落定之后才清，别的失败可重试，不留空白。
5. **权限与动作**：只有管理员分诊；动作矩阵与主动作；接受弹窗的顺序（只在 `beforeFormSubmit` 分诊，提交落定之后才关、才移）；可改的规则（提交者、访客提交者、已处理、已归档）；描述版本对访客隐藏；入口规则一处定义三处使用。
6. **可访问与拒绝**：页签、菜单、弹窗的名字、键盘与焦点；焦点的落点表，键盘、指针、窄屏菜单三条路，不落到 `body`；上一条下一条与窄屏开关有名字；↑、↓ 不在编辑器里；没有嵌套按钮；每个失败都提示、不 `console.error`；被拒不提示成功。
7. **删除干净**：spec §2.10 的三组，各自全绿；新建弹窗只改到必需的；三个包随之失去读者的导出；死成员前后整份比对；`old-intake-urls`；`copy-hook` 例外；告警等于上限。
8. **共用的外壳**：`INTAKE_COUNTS` 条件化、`WORKSPACE` 不变；M4 详情、命令面板、侧栏、页签栏的回归（检查点与终检的 m6-pages、m6-acceptance、m4-detail 与整套默认 e2e）。
