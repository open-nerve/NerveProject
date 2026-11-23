# M7 / P5a：网页的收集箱——数据层、列表、详情与分诊（阶段记录）

> 状态：实施完成，待整分支审查，未合并（§9）。spec 与 plan 见 [P5a spec](../specs/P5a-web-intake.md)、[P5a plan](../plans/P5a-web-intake.md)，预检 [re-94](../../../reviews/re-94-M7P5a预检.md)。基线 main `89be91e5`（P4 的 noff），分支 `codex/m7-p5a`。台账 `.superpowers/sdd/P5a-web-intake/progress.md`（ignored，随阶段归档），主控裁定 `rulings.md`。

## 1. 方法与范围

- **范围**（spec §1，P5 实施前再划为 P5a、P5b）：收集箱的数据层（`IntakeService`、issue 根 store 的子 store `root.intake`、纯函数 `filters.ts` 与 `status.ts`）；M4 逐工作项写队列上的接缝（内容修改、描述版本的恢复、分诊、删除）；收集箱页（页签与总数、七种筛选、排序、滚动分页、地址里的选中）；详情（名字、描述、表情、按状态名的动态、评论、描述版本、按角色与状态的属性、“重复于”）；页头按关注点拆开与分诊（接受弹窗、拒绝、暂缓与改暂缓、取消暂缓、标重复、删除）；侧栏的待处理数；入口规则 `intakeEntryOn`；Plane 的旧收集箱层整套删除（详情侧 T6a、列表侧 T6b、数据层 T6c）；新建弹窗只改经新接口提交所必需的；H9、H10、H11（文字内容与边界）、H20（收集箱页）的页面故事。不改 Go、契约与 CI。
- **不做**：P5b 的各项（新建弹窗的其余与关闭政策、详情的图片与附件、链接、通用详情与浮层隐去状态、浏览页的重定向、通知预览里嵌入的分诊、助手多选的忙时守卫，spec §1）；`getTabIndex` 与 `projectMemberIds` 的最终删除（P6a，R23）；服务端的 G1–G3（R26）。
- **做法**：主控派实施者逐个 Task 实施，每个 Task 之后主控审（T4 之后主控另派 T4b 跟进，T5 之后另派 T5c 的纯搬移）；先写失败的测试（RED 在断言处失败），每个性质至少一个能让测试失败的变异（`mutate.py`，`restored=yes`），页面故事先以期望错误的版本看它在被改处失败；每条命令经 `run-check.py` 留回执、输入前后的哈希与日志，标签不重用；三个检查点在 T2、T9、T13 之后（本机七项、四个页面配置、分支 CI）。实施中查出的三个产品缺陷（浮层的位置、D1、D2）与两个既有故事的时序竞态（M4 的 I9、P4 的 H6）各以单独的提交按根因修（§3）。T14 的文档由主控派实施者起草，主控核对后提交。
- **规模**（至 `68160da2`，T14 之前）：23 个提交（T1–T13、T4b、T5c 与五个修复；T5 两个），174 个文件 +15405 −4971；新增 80、删除 38、修改 55、改名 1（`core/components/projects/settings/intake/header.tsx` → `core/components/inbox/page-header.tsx`）。

## 2. 验收 → Task

| Task | 提交 | 内容 |
|---|---|---|
| T1 | `59285eaf` | spec、plan、预检 re-94；P4 记录 §5、§9 补记；设计 §12 把 P5 再划为 P5a、P5b，§14 的 P4 与 P5a；关键词的阶段 `M7/P5` |
| T2 | `840d553b` | 待处理数的 service、store 与侧栏的数；入口规则 `intakeEntryOn`（侧栏、页签栏、命令面板）；`mounts.ts` 的头注释；死成员基线；**检查点 0** |
| T3 | `0fdbed40` | 筛选的键与查询（C1–C6）、列表的读（认游标的下一页） |
| T4 | `fbd4b26c` | store 的确认：写进持有的每一份，只替换或移出，不插入 |
| T4b | `b2efc483` | 主控跟进：确认只按项目选列表，删去 `fetched` 的工作区与 `fetchList` 的工作区参数 |
| T5 | `ff5016c6`、`11233e9b` | ① `issue-writes.ts` 纯搬移拆出 `issue-write-confirm.ts`；② 收集箱条目的写经 M4 的逐工作项写队列（R3、R4、R33、R34） |
| T5c | `49a5f06c` | 主控派：收集箱条目的写纯搬移到 `issue-write-intake.ts`（`issue-write-confirm.ts` 回到 300 行之下） |
| T6a | `3ea45cd4` | 详情侧：新的详情、页头与动作模型；删除旧的详情树、版本 service 与 DTO、三个权限键、`copy-hook` 的例外 |
| T6b | `2ac1928c` | 列表侧：路由、页面状态、侧栏与页头、上一条下一条；删除旧的筛选树与旧侧栏；上限 31 → 18 |
| T6c | `bd1802fd` | 删除旧的 store、service、hook、类型、常量与工具（`RootStore` 21 → 20）；新建弹窗经新接口提交；`old-intake-urls` |
| T7 | `1fd902bb` | 筛选与排序：精确的查询、状态至少一个、日期按本地日的区间 |
| T8 | `bc0eeadf` | 分诊：只有管理员、接受弹窗的提交落定之后才移选中、暂缓最早明天、选择器收 id、焦点的落点 |
| T9 | `dd5f754b` | 属性按角色与状态、“重复于”经一次读、动态按状态名；**检查点 1** |
| 修复 | `30ccb25a` | 检查点 1：M4 的 I9 (page) 跳转之前等跨项目关联的目录读答完（§3） |
| 修复 | `b1aeccb1` | 列表外层显示时不再平移，筛选浮层与排序菜单回到按钮旁（T10 停下，§3） |
| T10 | `907d4699` | 收集箱页的夹具、条件化的 `INTAKE_COUNTS`、H9 (page) |
| T11 | `cdd17fc8` | H10 (page)（两个文件） |
| T12 | `5fa242b2` | H11 (page) 的文字内容与边界 |
| 修复 | `67b34153`、`40eb2d66` | T13 停下的 D1（窄屏列表藏起时的落点）、D2（有焦点的行离开列表）（§3） |
| T13 | `4612b2ce` | H20 (page) 的收集箱页（三个文件）；**检查点 2** |
| 修复 | `68160da2` | P4 的 H6 (page) 移出项目之前等 All 页签的列表读答完（§3） |
| T14 | 本提交 | 文档与交接 |

| 验收（spec §4） | Task | 状态 |
|---|---|---|
| A1 数据层 | T2、T3、T4、T4b、T6b、T6c | 达成：service 只调 `/api/v0`、每代以自己的客户端构造；store 是 issue 根 store 的子 store，计数按工作区 id、列表按项目与筛选键存；取数经 `useSessionSWR`，键带 `loginId` 与 `identity`，fetcher 只读键（`filtersOf`）；`fetchPart` 与认游标的合并；C1–C10 各有单测与变异（C1 `t3-mut-omit-status`、C2 `t3-mut-empty-array`、C3 `t3-mut-range-open-end`、C4 `t3-mut-order-plane`、C5 `t3-mut-no-limit`、C6 `t3-mut-cursor-made`、C7 `t4-mut-snooze-never`、C8 `t8-mut-snooze-today` 与 `t8-mut-snooze-utc`、C9 `t5-mut-body-state`、C10 `t6c-mut-target-null`）；页面状态不在 store |
| A2 写的接缝 | T5、T5c | 达成：先纯搬移（`t5-move-identical`、`t5-move-tests-untouched`）；端点在轮到时按所持详情选；收集箱的去处不经两段目录准备、不写项目列表（持有带状态组筛选与按状态组分组的列表时照样发出）；分诊在轮次里重读；接受像新建一样、只一次；拒绝值不变，`issue.intake_not_found` 移出，冲突在轮次里重读；行盖在详情上不留未接受的 `intake`；三条 `it.each` 的断言照旧；每条有变异。既有测试的改动只有允许清单的 `issue-changes.test.ts` 一格与主控裁定加进清单的 `issue.store.types.test.ts:55`（§3） |
| A3 列表与筛选 | T3、T6b、T7；修复 `b1aeccb1` | 达成：两个页签、总数（R31）、七种筛选、排序、已选的一行、滚动分页、空态、读失败可重试；查询照 C1–C6（H9 (page) ③ 每步恰一个列表读、查询精确）；换页签复位、离开复位；动作引起的选中变化是 `replace` |
| A4 详情 | T6a、T9 | 达成：动态按状态名与通用键（en、zh-CN）；描述版本对访客隐藏；“重复于”经 `fetchIssueCore`、不碰项目列表；FORMS 分支与模块级 service 删除；读不到在这次挂载自己的读落定之后才清选中，别的失败可重试；分诊的弹窗画在详情区域里、宿主之外，不随重读撤下 |
| A5 分诊 | T8；修复 `67b34153`、`40eb2d66` | 达成；R12 的落地与 spec 的字面不同（主控裁定 A：修改被拒之后取消也移选中，spec §3 补记）；R35 的落点表补了两种 spec 没写的情形：窄屏列表藏起时落详情区域（D1），后退时离开列表的有焦点的行（D2）；三种引擎由 H20 (page) 核对 |
| A6 侧栏与入口 | T2、T6b | 达成：待处理数按门与键读一次，大于 0 时显示，名字含数；`intakeEntryOn` 一处定义，读者是三处入口（侧栏 `project-navigation.tsx:120`、页签栏 `use-navigation-items.ts:90`、命令面板 `project-commands.ts:76`），另有页头的 “Add work item”（`IN/page-header.tsx:46-48`）与列表读的门（`IN/use-intake-list.ts:47`）；六格单测（`intake-entry.test.ts`）；已归档的项目没有 “Add work item” |
| A7 删除干净与守卫 | T2、T6a、T6b、T6c、T9、T14 | 达成：spec §2.10 的三组各一个提交、各自全绿；新建弹窗经新接口提交、没有日期不带键、成功跳到新条目；`RootStore` 20；`old-intake-urls`；地址规则收入新文件；`copy-hook` 例外删除；swr 名单；oxlint 等于上限；死成员整份比对，P5 的 39 行各有去处；三个权限键删除，T9 清掉五个没有读者的旧键（§5） |
| A8 e2e | T10–T13；修复 `30ccb25a`、`68160da2` | 达成：四个故事（七个文件、22 个测试）三引擎 66 过，chromium `--repeat-each=3` 66 过；`INTAKE_COUNTS` 在 T10 随第一个读者 H9 加（偏离 plan 的 T2，§3）；m6-pages、m6-acceptance、m4-detail 与整套默认 e2e 不回归（检查点 1 的 I9 是 M4 故事的竞态，按根因修了故事，§5） |
| A9 收尾 | — | 待 P5b 的 T1 补记（终检、最终分支 CI、合并、main CI 与归档，§9） |

## 3. 实施中的裁定

- **spec 之前**：调研的 Q1–Q38 都是局部裁定（`rulings.md`），落成 R1–R26；起草时的补充 R27–R32、预检 re-94 之后的 R33–R35 主控同意。Q12、Q13、Q14、Q21、Q34 标为 P5b。
- **高风险的局部裁定：R3、R4、R33、R34 动了 M4 网页的共用代码**（`rulings.md` 预检之后一节；收官报告列出）。R3：收集箱条目的内容修改、版本恢复、分诊与删除走 M4 的逐工作项写队列 `IssueWrites`，端点在轮到时按所持详情的 `intake` 选（`issue-writes.ts:118-124`）；收集箱的去处是与 `write()` 并列的一条路（`issue-write-intake.ts:73-95`），只取权限水位，不经两段目录准备、不碰项目列表、不登记在途的写。R4：接受经 `confirmCreation` 把重读的回答确认进持有的项目列表，只一次（`issue-writes.ts:178-200`）。R33：`issue.store.ts` 的 `choose`，列表的行落在 `intake` 未被接受的详情上时合并结果取已接受。R34：每次分诊在轮次里以 `refreshAuthoritative(id, { afterWrite: true })` 重读（`issue-write-intake.ts:119-129`），T5 的主控裁定把删除并入。落点：`issue-writes.ts`（209 行）、`issue-write-confirm.ts`（184）、`issue-write-intake.ts`（134）、`issue.store.ts`、`issue-write-fields.ts`（`intakeUpdateBody`）。缓解：先纯搬移、既有测试一行未改；`issue-changes.test.ts` 三条 `it.each` 的断言一条未改；T5 之后另跑 chromium 的 m4-detail（`t5-m4-detail`，44 过，42.3 秒）；三个检查点的 m4-detail 与整套默认 e2e 不回归（§5）。
- **T2**：`INTAKE_COUNTS` 常量挪到它第一个读者 H9（T10）：plan 的“T2 加常量”与同一 plan 的 knip 第一个读者规则冲突（`t2-probe-knip-mounts` 报未用的导出），先例 M6/P6a 的 `PROJECT_MODULES`；T2 只改 `mounts.ts` 的头注释，T10 加常量并删去头注释里的那一句（主控同意）。页签栏与侧栏的收集箱一项经 `intakeEntryOn`，Plane 的通用 access 过滤照旧（两层都查“有角色”，两层一起改的变异杀、一层的对照存活并记下）；侧栏的数照 Plane 显示在名字旁，链接名是整句 `inbox_issue.a11y.pending_count`；命令面板抽出 `getContextRole`，规划与收集箱两条入口共用。
- **T3、T4 与 T4b**：`filtersOf` 与往返的格留给它的读者 T6b；`IntakeFilters.statuses` 是非空元组（R9 的“至少一个”成为类型上的事实）；`IntakeTab`、`DayRange` 待读者再导出。T4 的 `confirm` 照 spec 字面按工作区与项目两层选列表，一层的对照存活；主控跟进为只按项目（同 `drop`）：`FetchedList.workspaceId` 与 `fetchList` 的工作区参数删去，spec §2.3 随改（`b2efc483`，变异 `t4b-mut-other-project`、`t4b-mut-drop-other-project`、`t4b-mut-other-project-both`）。
- **T5 的 R33 冲突**：接缝写完之后 web 全套只剩一格失败：M7/P2a 的 `issue.store.types.test.ts:52-60` 断言“行落在 snoozed 的详情上 `intake` 仍是 snoozed”，正是 R33 改变的情形，而它不在 plan 的允许清单。实施者停下报告。主控裁定取 A：这一格加进允许清单，只把所持详情的 `intake` 由 snoozed 改为 accepted（`:55`），仍钉住 P2a 的性质“列表的行不抹掉只在详情上的字段”；pending、snoozed、declined、duplicate 被行盖成 accepted 由 `intake-writes.test.ts` 的 R33 `it.each` 覆盖。同一次裁定：`inIntake` 把缺 `intake` 键的详情（M4 的旧夹具）当作 `null`；非接受的分诊在重读做成之后再读详情的动态与评论（`refreshAfterChange`，spec §3 补记）；发出前的权限核对由队列做（调用时捕获 `issue-writes.ts:53`，轮到时核对 `:69-79`），收集箱的去处不另核，另加“排在同一条目另一写之后时降了角色”一格与控制变异 `t5-mut-queue-permission-ignored`。
- **T5 的删除与别人的接受赛跑**（主控裁定 5，并入 R34）：删除在自己的轮次里先从每份收集箱列表移出，再在轮次里重读；canonical 只在重读读不到、且核验记它为 missing 时删（`refreshAuthoritative` 对 404 与被超过都给 `undefined`，`issue-verification.ts:243-256` 只为 404 记 missing）；读到详情（别人刚接受，nerve 只删收集箱的行）由重读写进 canonical；项目列表不碰；计数再取（`issue-write-intake.ts:57-64`）。先红 `t5-red-delete`，变异 `t5-mut-delete-no-reread`、`t5-mut-delete-overtaken-deleted`、`t5-mut-delete-drops-neither`。代价：删除多一个 `GET`，核验把覆盖这个项目的列表格标成陈旧（同 R34），所以删除一格对项目列表改为比显示的行与总数。spec §3 补记。
- **T5c：`issue-write-confirm.ts` 304 行的纯搬移**：T5 之后它过了 plan 的 300 行拆分线，主控在 `11233e9b` 之上派一个纯搬移提交（`49a5f06c`）：`TRIAGED_ELSEWHERE` 与 edit、triage、delete、intake、refused、readAgain、countAgain（原 `:39-40`、`:188-303` 逐字）搬进新文件 `issue-write-intake.ts` 的 `IssueWriteIntake`，`IssueWrites` 持有它（`intakeRoute`），三处调用直接交给它；`Unread` 留在 `issue-writes.ts`（只有它抛、它接）。`t5c-move-identical`、`t5c-tests-untouched` 过，既有测试一行未改；三个文件 209 / 184 / 134 行，`project-address-route` 的模式随之收入新文件。
- **T6a**：八条裁定主控同意：分诊的弹窗画在详情区域里、`IssueDetailHost` 之外，开关在动作模型；接受暂接在 `root.tsx` 到 T8；删除被拒只由动作模型说一次；去掉标签选择的 `showCreationToast`（唯一的传入者在 `isInboxIssue` 删去之后恒传真）；删 `editor/rich-text/description-input/loader.tsx`（唯一的读者是旧的 `issue-root.tsx`）；swr 名单只列 `content/root.tsx`；“打开”是按钮样式的 `ControlLink`（不在链接里套按钮）；清选中照宿主的三个子句。**TDD 的疏漏**：实现先于测试起草，RED 以把实现改回 Plane 式写法的骨架取得（`t6a-red-4` 21/43 在断言处），允许清单的两处改动也记于改动之后；主控记为过程备注，变异证据补足（表内 10 行、表外 12 个、一层对照 3 个），此后各 Task 先写失败的格。疑问 `IssueLabel.onLabelUpdate` 已没有产品传入者：主控裁定在 T6b 删去，fake 随之改（编译跟随）。
- **T6b**：七条裁定主控同意：读不到的条目先从收集箱的列表移出再清选中（否则自动选中会再选回它、再读、再提示）；列表的键带 `member=true` 的身份；上一条、下一条是用户的 `push`、两端回绕，↑↓ 不在输入框、名字、编辑器与对话框里；首读之后的读只在列表里等；空态经 `t()`、去掉 Plane 的空跳转按钮；页头不读 `useProject().loader`；窄页头收 `projectId`。删去的文件带走 13 条告警，上限 31 → 18。疑问：`ProjectStore.currentProjectDetails` 与 `LabelStore.projectLabels` 失去最后的产品读者——主控裁定在 T6c 删去，连同 `project.store.test.ts:67-68`、`label.store.test.ts:48` 各一条断言（记进允许清单；先例：去掉最后一个读者的阶段删它）。**险情**：实施者试过以 `git checkout -- <file>` 还原变异的助手，被权限分类器挡下、没有执行，文件完好；之后只用 `mutate.py`。
- **T6c**：store 的 `submit` 不等列表的重读与计数（提交已做成，重读失败不该让新建弹窗报失败，跳转也不等列表）；新建弹窗的 `if (!workspaceId) return;` 只为收窄类型；`old-intake-urls` 先看它命中 11 处（`t6c-red-old-urls`，只留收据），再与删除、改写同一提交。主控留给 T14 的三件（本提交）：`project-address-route` 的 `why`、`HO/M4-closeout.md` M7-04 的处理结果、spec 的两处补记。
- **T7**：十条裁定主控同意：“清除全部”回到页签的首选筛选、排序留着；`tabs.tsx` 的 `actions` 槽在 tablist 之外；最后一个状态在菜单里照旧按下、按了不变（不停用：停用的按钮收不回焦点）；预设的跨度同 Plane；负责人与创建者的选项含访客（`getProjectMemberIds(projectId, true)`）；已选一行的值以 `common.remove_item` 为名；T9 清旧键时留 `inbox_issue.order_by.*`。T6b → T7 的性质空窗（已选筛选的移除按钮有名字、留下其余的值、不提交外层表单）由 `IN/filters/applied.test.tsx` 关闭。
- **T8 与接受弹窗的 A**：实施者停下报告 spec 与代码不一致：spec §2.8、R12 说“更新确认时弹窗调 `onSubmit`，动作模型在 `onSubmit` 记下、在 `onClose` 里移选中”，而对已有的工作项 `base.tsx:216-221` 的回答来自 `handleUpdateIssue`，`update-command.tsx:60`、`:98` 交回 `undefined`，`onSubmit` 永不被调，确认之后的关闭与用户的取消走同一个 `onClose`。主控取 A：动作模型在 `beforeFormSubmit` 分诊做成（或 R29 见条目已不在收集箱）时记下“已接受”，之后的 `onClose` 以 `replace` 移选中（`isBusy` 在提交在途时挡住关闭，`base.tsx:308`）；修改被拒不移、弹窗留着、标题说已接受而修改未保存；被拒之后取消也移选中（条目已接受、已离开“未处理”页签），spec §3 补记。B（改 M4 的 `update-command` 与 `updateIssue` 的返回）、C（改 `base.tsx`）不取。另九条裁定同意：暂缓用 M4 的 `DateDropdown`（字段要能被拒）；每个分诊弹窗同样处理冲突（提示、关弹窗、选中不移）；删除被拒弹窗留着、拒绝在途不可关；落点分页面层 `landing.ts` 与详情层 `detail-focus.ts`，列表区域是有名字的 section；读不到的清选中也经 `leave`；写在途时 Accept、Decline 与菜单停用，复制不停用；窄屏时主动作的落点是窄页头的菜单按钮。待办：H10 另加“不改任何字段就接受”一步（T11 做了）；`use-intake-actions.ts` 295 行，之后不再加长（至今 295）。
- **T9**：重复于经 `fetchIssueCore`、读到之前不画、键带权限身份，画在属性的变暗容器之外；动态的键表是 `Map`（合同的 verb 是任意字符串），未知动词用通用文案；没有读者的旧键删五个（`inbox_issue.status.{pending,declined,accepted,duplicate}.description`、`project_empty_state.intake_sidebar.cta_primary`）。
- **检查点 1 的 I9 竞态**（`30ccb25a`）：m4-detail 的 I9 (page) [firefox] 失败一次：`expectListPageQuiet` 记下三个 `NS_BINDING_ABORTED`。不是登记的偶发签名，先查根因：M4 的 `relation.store.ts` 的 `createRelation` 在确认之后对跨项目的关联读 OPS 的成员、标签与工作区的状态（`directories`），故事在数据库核对之后就 `page.goto`，负载下这三个读来得晚、被跳转中止；自 M4/P11 起如此，P5a 没改这条路径。修故事（M4 的故事，只加等待、不改断言）：点 “Add selected work items” 之前登记三个读的 `waitForResponse`，跳转之前等它们答完。RED 以延迟这三个回答在 firefox、webkit 复现（chromium 对被路由扣住、遭导航取消的请求不报失败，没有 chromium 的 RED，已记）；m4-detail 三引擎 132 过。
- **T10 停下：筛选浮层与排序菜单在视口之外**（`b1aeccb1`）：三引擎相同，浮层在 x≈−512、排序菜单在 x≈−280。根因：`IN/root.tsx:103` 列表外层的 `translate-x-0`（照抄 Plane `apps/web/core/components/inbox/root.tsx:89-90`）计算值是 `0px` 而不是 `none`，任何平移都让元素成为 fixed 后代的包含块，而共用的浮层以 fixed 策略按视口算坐标（Plane 的 `FiltersDropdown` 不用 fixed，所以 Plane 不出错）；jsdom 看不到包含块。主控裁定只在收集箱页修、不动共用的 `FiltersDropdown`、`CustomMenu`：显示时 `translate-none`，只在 lg 以下隐藏时平移；以真指针与 `expectPopupBesideButton` 在 1280 宽与 800 宽三引擎先红后绿（`fix-menus-red-*`、`fix-menus-green-1`），计算值三引擎核过（`fix-menus-probe-computed-2`）；P5a 画的其余祖先没有 transform、filter、contain。
- **T10–T12 的故事裁定**（主控同意）：H9 拆四个测试（只看自己的访客另建世界）；滚动用滚轮（`scrollIntoView` 会卷动 `overflow:hidden` 的祖先）；③ 的浏览器时区定 Asia/Tokyo；H10 拆两个文件（接受与冲突；暂缓、标重复、拒绝与角色）；R4 在页面上只能核“回到项目列表时各一行、不刷新、未接受的不在”（工作项列表每次挂载都重读自己），插入与“只一次”由 T5 的单测钉住；分诊之后先核焦点、再地址、再弹窗已关（反过来时错误的“仍是旧的”在过渡的一刻成立）；H11 一个文件四个测试，每个可改的描述先等块 id 的迁移答完；写与请求体被动记下，只在 Chromium 放一律放行的路由（WebKit 在应用代码载入之前有路由时重取预载、控制台警告）；R27 在页面上是 M3 的“项目已归档”页（spec §3 补记）。
- **T13 停下：D1、D2 两个焦点缺陷**（R35）：D1 窄屏（800 宽）按一行之后列表藏起，分诊之后焦点给了屏幕外的邻行（三引擎 `toBeInViewport` 失败）；D2 分诊在途时后退或前进，详情与弹窗随选中卸下，Headless UI 把焦点还给被分诊的那一行，确认之后那一行离开列表，焦点落到 body。主控两个都取 A，各一个 `fix(M7/P5a)` 提交、在故事之前：D1（`67b34153`）`root.tsx` 告诉落点列表在不在屏幕上（`isMobileSidebar` 或窗口宽 ≥ 1024），藏起时本该落行的落新选中的详情区域，清空而还有行时等自动选中、落那一行的详情区域，没有行时落窄屏的 “Intake list” 开关；D2（`40eb2d66`）收集箱自己的 `use-leaving-row.ts`：有焦点的行离开列表时焦点给下一行、上一行、列表区域。D2 不抽 P4 的 `useLeavingFocus`：它按通知的选择器找元素、另持停用控件再启用时的回焦与菜单的 `controlOf`，抽成共用不是纯搬移、P4 的测试得改（主控同意）。各有 jsdom 的 RED 与变异，页面三引擎先红后绿（`t13-fix1-red-page`、`t13-fix2-red-page-2`）。
- **T13 的故事裁定**（主控接受）：H20 三个文件（落点、键盘与名字、迟到的回答）与新夹具 `intake-focus.ts`；落点的核对次序 `expect.poll` 焦点 → `toBeFocused` 与 `toBeInViewport` → 地址 → 弹窗已关；锁住（Escape、点背景不关、Cancel 停用、焦点在内）每个弹窗只核一次；两处按引擎（Firefox 在筛选面板的滚动列表上多一个 Tab 停点；WebKit 的指针一路后退之后落详情区域，点链接不取焦点）；⑩ 先等 X4 的页头再扣回答（WebKit 预载的警告，`t13-webkit-preload-red` 12 中 6 败，改后两次各 12 过）。
- **H6 (page) 的竞态**（`68160da2`）：T13 跑整份 m7-pages 时，P4 的 H6 (page) 在 WebKit 超时一次（`h6-notification-states-page.spec.ts:276` 的悬停）。主控读 trace：点 “All” 页签的挂载读与故事移出 bob 重叠，读在移出之后落库，Rate limit 的卡片随这次回答离开；产品行为正确，是故事的时序竞态，同 I9 的先例。实施者单独提交修：点 All 之前登记按解析后的地址精确匹配的 `waitForResponse`，移出之前等到 200，悬停之前先断言卡片可见（+14 行）；RED 以扣住那次读 1.5 秒、移出之后等它答完的探针，三引擎都在悬停处超时（`t13h6-red-probe`），同一探针下修后三引擎过；整份 m7-pages 三引擎 93 过（`t13h6-m7pages-whole`）。
- **故事时长超过 8 秒的先例**（主控接受，单个测试都在 30 秒之下）：H9 本机 chromium 12.2 秒，H11 21.8，H20 三个文件 20.0 / 15.2 / 10.4；原因见 §6。

## 4. 向负责人说明

- **R8 与 P4 R6：侧栏的待处理数不是实时的。** v0 §7.7 的会话取数只有一份 SWR 配置、聚焦时不重取，所以侧栏的数只在侧栏的项目列表挂载（进入工作区）、键变（某个项目的收集箱开关、归档或成员身份变了）、网络重连（SWR 的默认，`useSessionSWR` 不关它）与本人的提交、分诊、删除之后更新（`countAgain`）；别人提交的条目要等这些时刻才计入。Plane 以项目的 `intake_count` 在本地加减。与 P4 R6（顶栏的未读点）同类，同一个可选的 (c)（为这类键开聚焦重取的例外，改 v0 §7.7 一句与 `useSessionSWR`），一起进 M7 收官给负责人的报告。
- **P4 R25 的缺口照旧**：既归档又在暂缓中的通知在网页的每个视图里都看不到，直到暂缓到期（P4 记录 §4）；P5a 不涉及，照旧进收官报告。
- **G1–G3 不改合同**（R26）：“重复于”多一次读（G1）；选择器不给已归档的目标（G2，P2a 已登记）；带修改的接受是两个请求，修改可在接受之后被拒（G3，弹窗在字段下说明）。收官报告列出。
- **P5a 合入、P5b 之前 main 的样子**（spec §1）：
  - 收集箱页经 `/api/v0` 可用：列表、筛选、排序、滚动分页、详情、分诊、删除、侧栏的待处理数；网页没有 Plane 的收集箱地址（`old-intake-urls` 拦住）。
  - 新建弹窗经新的接口提交，成功之后照旧跳到新条目；关闭政策仍是 Plane 的（文档级 Escape、编辑器未就绪时拒绝关闭，M6-M7-01）。
  - 收集箱详情没有图片上传，也没有附件与链接一栏：新的详情 `IN/content/detail.tsx` 不给描述编辑器 `images`，没有这两栏（spec 原引的 `IN/content/issue-root.tsx:191` 随 T6a 删除）。
  - 收集箱条目的通知仍以普通浮层打开，属性里 “State” 一栏是占位（P4 记录 §7）。
  - 浏览页仍把收集箱条目显示成普通详情，它的写答 409 `issue.in_triage`。

  这些在 main 上本来就是这样，P5a 没有让它们变坏；只有显式打开开关的项目有收集箱，写都在服务端按合同判定。

## 5. 门禁与 CI

| 时点 | 本地 | 分支 CI |
|---|---|---|
| 检查点 0（`840d553b`） | build 与 e2e 502 过 1 败：F11 (page) [local] 以 M6-M8-07 的签名失败（`net::ERR_ABORTED` 的资产内容读，test-results 存 `cp0-test-results`），只重跑该文件一次 7 过（`cp0-e2e-f11-rerun`）；其余六项一次过（bootstrap 452.3 / 643.7——643.7 是 P5a 第 1 路的本机基线，停查线 673.7）；m7-pages 三引擎 27 过（45.1 秒），m6-pages 135 过，m6-acceptance 33 过，m4-detail 132 过 | 45068318695 attempt 1 全绿 |
| 检查点 1（`dd5f754b`） | 七项一次过（bootstrap 451.1 / 652.6，第 1 路 +8.9；e2e 503 过）；m7-pages 27 过，m6-pages 135 过，m6-acceptance 33 过；m4-detail 131 过 1 败：I9 (page) [firefox]（§3，test-results 存 `cp1-m4-detail-test-results`），按根因修了故事之后整个配置三引擎 132 过（`cp1fix-m4-detail`） | 45103652178 attempt 1 全绿 |
| 检查点 2（`4612b2ce`） | 七项一次过（bootstrap 453.2 / 653.6，第 1 路 +9.9；e2e 525 过，4.0 分）；m7-pages 三引擎 93 过（`cp2-m7-pages`，91.6 秒）；m6-pages 跑到一半机器异常（负载很高、Docker 守护进程随之停了），WebKit 的 G3、G9 两格以 “Target page, context or browser has been closed” 失败、没有收据——环境，不算重跑：重启 Docker Desktop，同一 HEAD、没有改动，以前缀 `cp2r` 只补跑三个配置：m6-pages 135 过（2.6 分），m6-acceptance 33 过（37.0 秒），m4-detail 132 过（1.3 分）；m7-pages chromium `--repeat-each=3`（P5a 的七个故事文件）66 过（`cp2r-m7-repeat`，57.7 秒） | 45147321651 attempt 1 全绿 |
| 修 H6（`68160da2`） | 整份 m7-pages 三引擎 93 过（`t13h6-m7pages-whole`，93.6 秒），H6 文件三引擎与 chromium `--repeat-each=3` 过 | 未推送（随终检） |

| 运行 | server (0) Test / bootstrap | server (1) Test / bootstrap | web（Lint / Unit / Build） | e2e (1) | e2e (2) |
|---|---|---|---|---:|---:|
| main 45051719903（基线） | 869 / 585.2 | 1002 / 970.9（80.9%） | 604（60 / 508 / 12） | 817 | 719 |
| 检查点 0（45068318695） | 693 / 503.4 | 777 / 754.8（62.9%） | 717（71 / 606 / 13） | 756 | 992 |
| 检查点 1（45103652178） | 884 / 597.8 | 779 / 757.7（63.1%） | 663（65 / 555 / 10） | 879 | 980 |
| 检查点 2（45147321651） | 680 / 492.2 | 785 / 764.0（63.7%） | 661（64 / 548 / 10） | 1007 | 873 |

- 秒数取自 jobs 接口的步骤时刻（Test 步、web 作业与三步、e2e 作业），bootstrap 取自 server 作业日志里 `internal/bootstrap` 包的那一行；括号里是 bootstrap 占期限 1200 秒（20 分）的比例。
- **web 作业**（20 分钟期限、960 秒报告线）：都没到报告线。检查点 0 的 717 秒，三步对 main 的同名步各慢约 1.18、1.19、1.08 倍（T2 只加了毫秒级的测试），是 runner；检查点 1、2 回到 663、661。
- **第 1 路**（本机以检查点 0 的 643.7 秒为基线、+30 秒停查；CI 的八成只报告）：本机 +8.9、+9.9，线内。CI 的第 1 路 bootstrap 754.8、757.7、764.0 秒（62.9%–63.7%），离八成很远；P5a 不改 Go。
- **e2e 两片**（35 分钟期限、30 分钟报告线）：最长 1007 秒（检查点 2 的第 1 片，其中装浏览器 349 秒，另一片 24 秒：下载慢；它的 E2E 步 573 秒）。测试数本机 503 → 525、CI 两片合计 508 → 530，多出的 22 是 P5a 的 22 个页面测试。
- **runner 的映像**（检查点 1 时主控记下，T14 以作业日志的 “Runner Image” 一节补全）：各作业的映像 `ubuntu24/20261116.133` 与 `.376` 混用，`.376` 上的 server 与 e2e 作业一致地慢——第 1 路 bootstrap 在 `.376` 为 970.9（main）、864.6（P4 终检，台账），在 `.133` 为 754.8、757.7、764.0（三个检查点）；第 0 路 bootstrap 在 `.376` 为 585.2（main）、597.8（检查点 1），在 `.133` 为 503.4、492.2（检查点 0、2）；e2e 第 2 片在 `.376` 为 992、980、873（三个检查点），在 `.133` 为 719（main）、749（P4 终检，台账）；e2e 第 1 片在 `.376` 为 817（main）、879（检查点 1），在 `.133` 为 697（P4 终检，台账）、756（检查点 0）、1007（检查点 2，见上条的下载）。web 不一致（main 的 604 与检查点 0 的 717 都在 `.376`）。慢在 runner 的那一池，不在代码；收官报告与第 1 路分界的建议带上这条。台账里检查点 2 的 “runner 映像 20261111.635” 是 “Runner Image Provisioner” 的版本（各作业都一样），不是映像本身。
- **推送之前的估算**（主控自记的流程疏漏）：plan 的检查点 2 要求推送之前照 spec §2.12 估出 e2e 两片的时长，主控没有先估就推送了；事后两片 1007、873 秒，离 30 分钟的报告线很远，没有后果；终检推送之前照做。
- **环境**：PostgreSQL 容器 10 秒内没绑定端口五次（T10 一次、T12 两次、T13 两次），都没有测试运行，以新标签重跑即过（主控记为 M6-M8-05 一类）。

**死成员与告警**

- 死成员（M1 交接）：T2 基线 1614 行（成员 1217、prop 393、导出 4；`--rows M7` 98 行，与 P4 的 `rows-M7-t15.tsv` 逐行相同，含行号），T14 终表 `t14-dead-final` 1598 行（成员 1204、prop 390、导出 4；`--rows M7` 88 行），与 D2 修复之后的 `t13-fix2-dead-after` 一字不差。整份按（种类、文件、名字）比对：消失 39 行、新出现 23 行。
  - **P5 的 39 行各自的去处**：`services/inbox/intake-description-version.ts` 15 行随文件在 T6a 删除；`inbox/content/root.tsx:28-29` 的 `isNotificationEmbed`、`embedRemoveCurrentNotification` 在 T6a 不再收（R32，P5b 随读者加回）；`inbox-issue-status.tsx:20` 的 `showDescription` 随文件在 T6b 删除；`store/inbox/project-inbox.store.ts` 9 行、`inbox-issue.store.ts:31` 的 `fetchIssueActivity`、`@nerve/types` 的 `inbox.ts` 3 行随文件在 T6c 删除；`create-root.files.test.tsx:148` 的 `issue`、`description_html` 随 T6c 的改写消失。留下 6 行：两份新建弹窗测试里 `NoResize` 替身的 `observe`、`unobserve`、`disconnect`（`create-root.files.test.tsx:41-43`、`create-root.request.test.tsx:78-80`），浏览器 `ResizeObserver` 的替身，读者是 `create-root.tsx` 的 effect 经全局构造的 `ResizeObserver`（工具看不到），新建弹窗的其余属 P5b。合计 33 行消失、6 行留下。
  - **消失的另外 6 行**：`@nerve/types` 的 `common.ts` 里 `TPaginationInfo` 的成员，随它在 T6c 删除（不在 M7 的行里）。
  - **新出现、留下的 23 行**：都是 P5a 自己的测试与夹具里字面量的成员，读者是测试本身（整体的 `toEqual`、断言），基线已有同类：路由的 `intake/page.test.tsx:64` 的 `params`；`content/header/navigation.test.tsx:18`、`root.test.tsx:29`、`landing.test.tsx:20`、`sidebar/sidebar.test.tsx:32` 的 `search`、`type`；`content/root.test.tsx:41` 的 `id`、`answer`、`fail` 与 `:155` 的 `where`、`search`、`type`、`navigate`；`fake-intake-page.tsx:36` 的 `type`；`page-state.test.tsx:17` 的 `state`、`:18` 的 `navigate`；`sidebar.test.tsx:35`、`:82` 的 `IntersectionObserver` 记录 `callback`、`element`。
  - **M7 其余领域的行**：`rows-M7-t14.tsv` 的 88 行 = P5 留下的 6 行 + 上面 23 行 + P6 视图 30 行、P7 收藏与首页 14 行、M4 列表测试的替身 9 行、P4 自己的测试字面量 6 行（后四组与 P4 的终表相同），随各自的阶段处理。
  - T6a–T6c 之间旧 store 留下待删的行（T6a 21 行、T6b 34 行）与 `currentProjectDetails`、`projectLabels` 的 4 行都在 T6c 出表；每个 Task 的整份 TSV 与差异在台账的 `dead-members/`。
- oxlint（`t14-oxlint-files`，只读）：P5a 新建与改到的 124 个 `.ts`/`.tsx` 文件（web 应用 106、`@nerve/constants` 1、`@nerve/types` 3、`@nerve/utils` 1、e2e 13）警告与错误为零；web 18 = 上限 18，`@nerve/utils` 2 = 2，`@nerve/constants`、`@nerve/types`、`@nerve/i18n`、e2e 都是 0 = 0。web 的上限在 T6b 同一提交 31 → 18：13 条全在删掉与改写的旧文件里（旧已选筛选的 priority 4、state 4、status 2，`filter-selection.tsx` 1，旧 `sidebar/root.tsx` 2，`t6b-oxlint-base-files-2`）。本段没有按规则清一类。
- 关键词：`old-intake-urls`（新，T6c）；`project-address-route` 收入写的接缝的三个文件与收集箱页读项目的新文件（T5、T5c、T6b、T7、T8、T9），`why` 的 `currentProjectDetails` 一句改为照今天的代码（T14）；`workspace-address-route` 收入 `use-intake-counts`（T2）与页头、列表的读、路由页（T6b）；`copy-hook` 的收集箱例外随旧页头删除（T6a）。`.oxlintrc.json` 的 swr 名单收入收集箱的取数文件（T2、T6a、T6b、T9），删去旧的 `inbox/content/description-versions.tsx`（T6a）。

## 6. 故事时长（秒）

检查点 2 的 `cp2-m7-pages`（三引擎整配置 93 过，91.6 秒；P5a 的七个文件 22 个测试，括号里是每个测试）。

| 故事文件 | chromium | firefox | webkit |
|---|---:|---:|---:|
| `h9-intake-list-page`（①空 ②51 条 ③筛选与排序 ④只看自己的访客） | 12.2（1.8、2.5、6.9、0.9） | 28.1（2.9、11.5、12.3、1.4） | 20.1（3.9、3.6、11.7、0.9） |
| `h10-intake-accept-page`（①接受 ②冲突被拒） | 4.1（2.8、1.2） | 6.3（4.5、1.8） | 5.3（3.8、1.5） |
| `h10-intake-triage-page`（③暂缓 ④标重复、拒绝与角色） | 6.1（2.9、3.3） | 11.4（6.1、5.3） | 10.0（4.9、5.1） |
| `h11-intake-content-page`（①管理员 ②成员 ③访客提交者 ④已处理与归档） | 21.8（9.1、5.0、6.0、1.8） | 26.1（10.4、6.2、6.5、2.9） | 26.2（10.3、6.8、7.1、2.1） |
| `h20-intake-pages`（①标重复 ②暂缓 ③改暂缓 ④窄屏） | 20.0（5.0、5.3、6.8、2.9） | 31.0（6.7、7.5、11.0、5.8） | 27.6（6.6、7.7、9.6、3.6） |
| `h20-intake-keys-pages`（⑤页签与筛选 ⑥排序、菜单、↑↓、弹窗 ⑦锁与落点、剪贴板） | 15.2（1.5、1.8、12.0） | 18.0（2.3、3.0、12.7） | 16.9（1.6、2.7、12.6） |
| `h20-intake-late-pages`（⑧后退 ⑨换项目与工作区 ⑩换账号） | 10.4（2.8、4.1、3.5） | 14.3（4.3、5.4、4.6） | 13.8（3.8、5.3、4.7） |

- chromium 重复三次（`cp2r-m7-repeat`，66 过，57.7 秒），每个测试：H9 ① 1.8–2.8、② 2.7–3.5、③ 7.1–10.0、④ 0.8–1.0；H10 接受 2.6–3.5、冲突 1.1–1.4、暂缓 2.7–4.4、标重复与拒绝 3.6–3.7；H11 ① 9.5–10.0、② 5.1–5.5、③ 5.8–6.3、④ 1.6–2.1；H20 ① 4.6–6.5、② 5.4–5.9、③ 6.5–7.4、④ 2.6–3.5、⑤ 1.1–1.6、⑥ 1.8–1.9、⑦ 11.6–12.0、⑧ 2.5–3.2、⑨ 4.1–4.4、⑩ 3.3–3.8。
- P5a 单个测试最长是 H20 ⑦ 的 firefox 12.7 秒；同一配置里最长的是 P4 的 H20 迟到（firefox 14.9），都在 30 秒之下。
- **超过 8 秒的原因**（本机 chromium）：H9（12.2）：② 51 条与 50 行的列表，③ 18 次列表的读与两种宽度的浮层；firefox 的 ② 滚轮一转只滚几行。H11（21.8）：每个可改的描述都有 1.5 秒去抖的自动保存（打开时的块 id 迁移、名字、描述、恢复之后的迁移），管理员一格四次、访客三次、成员两次。H20 落点（20.0）：每个锁住的弹窗要等三个 1 秒的否定（Escape、点背景不关、Cancel 不启用），三种动作各走键盘、指针、窄屏菜单三条路、各有取消与成功。H20 键盘（15.2）：⑦ 三个弹窗的锁约 9 秒。H20 迟到（10.4）：⑧–⑩ 共四个 1 秒的“不出提示”。
- 修 H6 之后（`t13h6-m7pages-whole`，93 过，93.6 秒）：H6 11.2 / 12.3 / 12.7（修前检查点 2 为 11.7 / 13.0 / 11.9）；P5a 的七个文件与上表相差都在 3 秒以内。
- 新的 vitest 文件多在毫秒级；较慢的是首格动态载入 `RootStore` 的几份 store 测试（各约 1 秒，`intake.store.confirm.test`、`intake.store.test`、`intake.store.submit.test`）与 `content/header/navigation.test`（第一格冷启动约 1.3 秒）。每个检查点的 web 作业 Unit tests 见 §5。

## 7. 交给 P5b 与 M8

- **P5b**（spec §5 的“后续”）：新建弹窗的其余（M6-M7-01 的 `requestClose`、`HO/M3-closeout.md:40-41` 的新建弹窗一半、`:55` 的标题长度提示、`:56` 的文档级 Escape、`getTabIndex` 的四个文件、弹窗里的英文提示）；详情的描述图片与附件（M5-M7-01，`[local]` 与 `[s3]` 三引擎，Q21）；链接（Q14，登记与 Plane 的差异）；通用详情与浮层在未接受时隐去状态、浏览页把未接受的条目重定向到收集箱页（Q12）；通知预览嵌入收集箱的内容、两个 prop 加回、`workspace-notifications/root.test.tsx:215-221` 改写、H2–H4 (page) 收集箱一步（Q13，登记“已拒绝的也嵌入”）；助手多选的忙时守卫（Q34：故事里快速连点，重现即在共用控件按根因修，否则写“未重现”交 M8）；P5a 的 A9 回填；M7-01、M7-04 的关闭。另：`HO/M3-closeout.md:47` 的详情的负责人下拉框还没有在页面上核过（§8），P5b 的故事打开它时一并核对（或由主控另定）。
- **P6a**：`getTabIndex`、`projectMemberIds` 的最终删除（R23；`projectMemberIds` 只剩视图的两个读者）；M6-M7-07 的视图一半。
- **M8 的候选**（M7 收官时写进 M8 的交接；都是 M4 的既有行为，P5a 没改）：
  - **M4 的工作项列表每次挂载同一个 GET 发两次**（T11 的观察）：`GET /api/v0/projects/{id}/issues?order_by=-created_at&sub_issue=true&limit=100`，不分诊、只在列表与收集箱之间来回也是两次（`t11-explore-4`、`-5`）。
  - **M4 描述编辑器的自动保存窗口里工作项被重读**（T12 的观察）：块 id 的迁移还在 1.5 秒的去抖里未发出时，若有一个在轮次里重读工作项的写（如评论的 `refreshAfterChange` → `refreshAuthoritative({ afterWrite: true })`），重读在途时 canonical 不定、详情的正文撤下，编辑器卸下时按“canonical 的主人消失即挂起输入”不冲刷（`use-confirmed-autosave.tsx:93-109`），重挂之后显示 “Your unsaved input has been preserved. Save it when you are ready.”、页头一直 “Saving”，之后换行被离开确认挡住。M4 的工作项详情同样（I13 的故事也先等迁移落定）；用户在打开或恢复版本之后 1.5 秒内评论就会遇到。
  - P4 的 H6 (page) 的竞态已在本段修好（`68160da2`），不交。
  - 本机 PostgreSQL 容器 10 秒内没绑定端口（§5 的环境，M6-M8-05 一类），观察。

## 8. 文档

- **plane-diff**（四、行为差异，收集箱一节）：spec §7 逐条登记——筛选与排序的寿命（R5）、列表的行是页面自己的副本、分诊之后的选中（R14）、每页 50（R6）、侧栏的待处理数（R7、R8，P2b 的计数行之后另加网页一行）、只有管理员看到分诊（R11）、接受弹窗（R12、R29、G3）、暂缓到某日（R13）、访客提交者（R15）、“重复于”与选择器（R17、G1）、动态按状态名（R18）、入口规则与已归档的项目（R21、R27）、冲突之后重读（R28）；“没有 FORMS 来源”改写在 P2a 的“收集箱的来源”一行。另加 spec §7 之外、与 Plane 不同的三行：分诊之后的焦点（R35 与 D1、D2）、上一条下一条与 ↑↓、分诊与删除的弹窗在途与被拒。T10 的浮层修复让收集箱页回到 Plane 的样子（浮层在按钮旁），不登记。Plane 的行号都对过 `plane/` 的源码。
- **frontend-changes**：3.1 的 M7 一行写 P5a（收集箱的范围、P5b 余下的）；“挂载时的取数”一行补“侧栏的待处理数由 M7/P5a 加回”；3.2 加收集箱的数据层、写的接缝、收集箱页、入口规则、旧收集箱层的删除五行。
- **交接的处理结果**（spec §5 的表，逐行）：`HO/M4-closeout.md` 的 M7-01、M7-04（旧 DTO、`TLegacyIssueUpdateRequest`、版本 DTO 与 service、`inbox/content/description-versions` 的过渡删除；命令面板的收集箱结果按 R24 关闭；`frame-src` 写明不嵌 iframe；关闭政策随 P5b）、M7-05；`HO/M6-closeout.md` 的 M6-M7-07（收集箱的一半关闭）；`HO/M3-closeout.md` 的 `:14`（侧栏的待处理数，收集箱的部分关闭）、`:33`（收集箱的部分关闭）、`:40-41`（拒绝、删除、暂缓、重复四个弹窗关闭）、`:47`（**不关闭**：spec 表写“详情的一个关闭”，但页面故事没有打开详情的负责人下拉框，H11 (page) 只断言它停用，只有单测 `issue-properties.test.tsx:194`）、`:48`（收集箱一个调用方关闭）、`:52`（关闭）、`:61`（收集箱的 4 个文件关闭）、`:63`（收集箱一处关闭）、`:64-65`（收集箱的部分关闭）、`:68` 与 §7（收集箱读者去掉，getter 随 P6a）；`HO/M2-closeout.md` 的 §2（M7 的 3 处单例都已消失，最近访问的键随 P7）、§3（`intake-state` 一半按不适用关闭）、§4（收集箱一处关闭）、§6（写明不嵌 iframe）；`HO/M1-closeout.md` 的死成员与 oxlint 两节；P4 记录 §5 的 `:112` 与 §7 的 `:147`、P4 spec §5 的 P5 一条；P2a、P2b、P2c spec 的 P5 一条与 P2c 记录的忙时守卫一条。没写关闭的保持 open。表里本段不涉及的四行（M6-M7-05、`HO/M3-closeout.md:55-56`、M5-M7-01、M6-M7-01）不写；表外的 `HO/M3-closeout.md:21`（查重改用 M4 的搜索，M4 已做，P5a 在 H10 (page) 走了真实页面）也没写，交主控定。
- **spec**：§3 末尾补记四条（非接受的分诊之后读详情的动态与评论；删除在轮次里重读；R12 的落地；R27 在页面上）。
- **关键词**：`project-address-route` 的 `why` 里 `currentProjectDetails` 一句照今天的代码改写（getter 已在 T6c 删去）。
- **设计**：§12 的 P5a 一行已在 T1 链上 spec；§14 的 P5a 一行改为实施完成、待整分支审查。

## 9. 审查与集成

整分支审查、修复轮、终检与集成见下（T15 写）。
