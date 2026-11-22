# M7 / P4：网页的通知与关注（阶段记录）

> 状态：实施完成，待整分支审查（§9）。spec 与 plan 见 [P4 spec](../specs/P4-web-notifications.md)、[P4 plan](../plans/P4-web-notifications.md)，预检 [re-93](../../../reviews/re-93-M7P4预检.md)。基线 main `3d9293bd`（P3d 的 noff），分支 `codex/m7-p4`。台账 `.superpowers/sdd/P4-web-notifications/progress.md`（ignored，随阶段归档）。

## 1. 方法与范围

- **范围**（spec §1）：通知的数据层（service、store、纯函数照 v0 §7.7 重写，Plane 的旧数据层、手写类型、常量与工具整套删除）；通知页（两个页签与计数、类型筛选、未读 / 已归档 / 已暂缓、卡片、加载更多、刷新、全部已读、卡片的已读 / 归档 / 预设与自定义的暂缓、嵌入的工作项浮层）；顶栏的未读数；详情页头、浮层页头与命令面板的“关注 / 取消关注”；收起筛选行的“保存为视图”；守卫与告警；新的三引擎页面配置 `e2e/playwright.m7-pages.config.ts` 与七个页面故事文件。M7 的第一个页面阶段，不改 Go 与契约。
- **不做**：收集箱的嵌入分诊（P5，收集箱条目的通知以普通浮层打开）；视图（P6a/P6b）；收藏与最近访问（P7）；应用侧栏的未读数（Plane 的那一项从不渲染）；`archived=true` 是否也列暂缓中的（P1c 合同的语义，§4）。
- **做法**：主控派实施者逐个 Task 实施（T1、T15、T16 主控），每个 Task 之后主控审；先写失败的测试（RED 在断言处失败），每个性质至少一个能让测试失败的变异（`mutate.py`，`restored=yes`）；每条命令经 `run-check.py` 留回执、输入前后的哈希与日志，标签不重用；三个检查点在 T2、T11、T14 之后（本机七项、页面配置、分支 CI）。
- **规模**：24 个提交（T1–T14 与修复），150 个文件 +11172 −3083；新增 74、删除 32、修改 43、改名 1（`sidebar/header/root.tsx` → `sidebar/header.tsx`）。

## 2. 验收 → Task

| Task | 提交 | 内容 |
|---|---|---|
| T1 | `8f718622` | spec、plan、预检 re-93；P3d 记录 §9 补记；设计 §14；关键词的阶段 `M7/P4` |
| T2 | `0c87739b` | 未读数的 service、store 与顶栏的点；`mounts.ts` 的 `WORKSPACE`；死成员基线；**检查点 0** |
| T3 | `e2e22824` | `ReconciledByKey.fetchPart`、筛选的键与查询、列表的读 |
| T4 | `a31d26a7` | store 的写：一个接一个，确认写进持有的每一份，写后重取计数 |
| T5 | `eb86e20b` | 通知页的骨架与预览；删除旧数据层、类型、常量与失去读者的导出；`old-notification-urls`；上限 40 → 31 |
| T6 | `98611945` | 卡片的文字（`describeChange`）与执行者（`useActorName`） |
| T7 | `e7f233d1` | 页头：全部已读、刷新、类型筛选与视图菜单 |
| 修复 | `39fc1282` | 刷新把第一页交给列表的读（列表首读失败之后刷新即显示列表） |
| T8 | `8c16960a` | 卡片的已读、归档与预设的暂缓，随视图收窄 |
| T9 | `d160de9c` | 自定义暂缓的弹窗 |
| 修复 | `b12dbdd4`、`0d107ac0`、`b143d6a1` | ① 全部已读把重读的第一页交给列表的读；② 卡片离开之后焦点给下一张；④ 时间选择的名字带字段 |
| T10 | `81f32236` | 关注与取消关注 |
| T11 | `aeff8b9a` | 收起“保存为视图”；**检查点 1** |
| C12 | `5fe40067` | M4 的故事 C12 (page) 随关注改断言（§3） |
| C4 | `408f3e46` | 新建弹窗在项目目录晚到时也会出现（§3） |
| T12 | `4eb1027a` | 页面配置、夹具与 H1、H2–H4 (page) |
| 修复 | `bbadf7f7` | 列表回到读过的筛选时立即重读（`dedupingInterval: 0`） |
| T13 | `3b1c4bdb` | H5、H6、H7 (page) |
| 修复 | `7aa8fe92`、`ee14f2fb`、`2a112d81` | 两个真实引擎里的焦点缺陷与简化（§3） |
| T14 | `1325790c` | H20 (page) 的通知页；**检查点 2** |
| T15 | 本提交 | 文档与交接 |

| 验收（spec §4） | Task | 状态 |
|---|---|---|
| A1 数据层 | T2、T3、T4、T5；修复 `39fc1282`、`b12dbdd4`、`bbadf7f7` | 达成：service 只调 `/api/v0`；按会话分代、`useSessionSWR`、键带 `loginId`；`ReconciledByKey.fetchPart`；页面状态不在 store；纯函数与陷阱 T1–T8 有单测；性质的变异皆杀 |
| A2 通知页 | T5–T9；修复 `0d107ac0`、`b143d6a1`、`7aa8fe92`、`ee14f2fb` | 达成；I4 的页面一格经工作项读被拒（§3） |
| A3 未读数 | T2、T4、T7 | 达成：顶栏与页签共用一个取数；`WORKSPACE` 与读它的 S2 的两个、G3、G8、G13 跟随（spec 写“六个故事”，读 `WORKSPACE` 的是这五个，第六个 `p2-visibility` 只读 `PROJECT_MEMBER`） |
| A4 关注 | T10、T11（C12） | 达成 |
| A5 页面故事 | T12、T13、T14 | 达成：三引擎 27 过，chromium `--repeat-each=3` 过；m6-pages、m6-acceptance、m4-detail 不回归（§5） |
| A6 删除与守卫 | T2、T5、T9、T11、T15 | 达成（§5 的死成员与告警） |
| A7 文档 | T1、T15 | 本提交（§8） |
| A8 门禁、审查与集成 | T16 | 待填（§9） |

## 3. 实施中的裁定

- **spec 之前**：调研的 WP-1–WP-18 主控皆取 (a)，成为 spec 的 R2–R19；Plane 网页缺陷 W1–W8 都修、都登记（R20）。
- **re-93 I1 → R25**（主控）：网页侧修——“已暂缓”的视图里卡片不给“归档”，“已归档”的视图里不给“暂缓”，页面不造出两者兼有的通知；服务端 `archived=true` 是否也列暂缓中的是 P1c 已合入合同的语义、跨模块，P4 不动，进 M7 收官的报告（§4），登记 plane-diff。re-93 其余的重要与次要照审查建议采纳。
- **T2**：未读的读屏名一整句（`notification.a11y.unread_count`，不在代码里拼逗号）；钩子的格写在顶栏的测试里；删去 `sidebar-item.tsx` 两个没人传的 `className`。
- **T3、T4**：`update` 收路由的工作区 `{id, slug}`（与别的 store 方法一致）；`fetched` 登记在 T4 建；计数与列表保持私有；`matches` 的此刻在确认时取一次；T4 不做 `followInSession` 的变异（归 T7–T10）。
- **T5**：页面状态的五个开关在 T5 落、T7 给产品读者（死成员 1615 → 1610 核对）；调用者对项目没有角色时预览都读项目；第四种交代（读到了仍没有角色）清选中并提示；页签用 Headless `TabGroup`；oxlint 的 swr 名单逐个列文件；Plane 路径的文件保留原版权头。
- **T6、T7**：`describeChange(change, format)`；共用 `menu-toggle.tsx`；刷新在途停用；菜单按钮不套 Tooltip（触发器会成为按钮里的第二个可交互元素）；jsdom 里以“都是普通按钮、`tabIndex` 0”代真实的 Tab，真 Tab 由 H20 核。疑点“列表首读失败之后页头的刷新清不掉读失败”主控裁定修：`39fc1282`。
- **T8、T9 与三处修复**：拒绝一处判定 `use-notification-refusal.ts`；`@nerve/i18n` 的 `useTranslation()` 另回答 `language`（时段按页面语言标注）；英文 “Unarchive / Unsnooze” 改正；没有读者的旧键只删 `notification.options.mark_snooze`（89 个叶子键逐个 `grep`）。疑点：① 全部已读同样把第一页交给列表的读（`b12dbdd4`：store 的 `markAllRead` 只重取计数，第一页经列表的 SWR 键）；② 自己的动作让卡片离开之后焦点掉到 body——修为给下一张卡片的主按钮，没有则上一张，再没有则列表区域（`0d107ac0`）；③ 真实引擎里的回焦由 H20 核；④ 时间选择选定之后名字只剩时段——修为 “Time, 12:30 AM”（`b143d6a1`，看得见的仍是时段）；⑤ 没有关闭动画，不改。
- **T10、T11 与 C12**：关注与退订都是 POST（契约如此）；T11 用 `refactor` 前缀（plan 原文）；T10 写后重读时详情撤下、页头短暂消失，是 M4 写后重读的共同路径，沿用，H1 (page) 按次序断言。**C12 的故事改动**（`5fe40067`）：M4 的 `e2e/stories/editor-loading.spec.ts:360` 原断言详情没有关注动作，T10 之后详情页头带“取消关注”，改为恰有一个 “Unsubscribe”，新建表单仍然没有——这是对 M4 故事的有意改动，不是回归。
- **C4 的调查与修复**（`408f3e46`）：T11 的 chromium 运行里 M4 的 C4（`detail-interactions.spec.ts:117`）失败一次、整套重跑过，没有登记的签名。只读调查（`research/c4-create-dialog.md`）推翻了“权限水位让对话框关上”的猜测：根因是 Plane 继承的 `issue-modal/base.tsx` 选默认项目的 effect 不依赖允许的项目，目录晚到则弹窗永不出现、此后这个项目各页的新建入口都没有反应；与 P4 无关，真实用户可遇。主控裁定在 P4 单独提交修：依赖加上允许的项目的个数（等待时只可能是空列表，选过项目之后重跑幂等）、单测与变异 `c4-mut-no-allowed-dep`；`eslint-disable` 保留（去掉则 oxlint 32 > 31）；登记 plane-diff。
- **T12**：
  - H2–H4 (page) 不归档、不暂缓，那是 H6 的。
  - **I4 的页面一格与 spec 的措辞不同**：spec 写“标已读答 `notification.not_found`，项目读也被拒”；页面载入时 bob 仍是项目的成员、store 里有他的角色，所以选中即设浮层、读工作项（`GET /issues/{id}` 404，浮层显示 “The work item does not exist, or you cannot see it.” 与 Retry，不是空白），`useProjectRead` 只在角色不知道时发生（已归档的项目，T5 的单测）。之后 `notification.not_found` 提示一次、这一行离开、预览回到没有选中。I4 要的“不留空白”照样成立，主控接受。
  - `parent_changed` 与 `updated` 两种卡片只在单测里。
  - 快照在扇出任务运行时读（P1b §2.6）：bob 在任务运行之前关注，会多收一条“created the work item”，故事因此先等前一条再关注（§7）。
  - 收集箱条目的浮层里“State”一栏显示占位（§7，P5）；嵌入的浮层里的关注切换由 H20 核。
- **T13**：
  - H5 造 101 条：一次加 100 个标签（`label_ids` 至多 100），Many 由 walt 负责、等他的两条之后 bob 才关注。
  - 真实浏览器里查出列表回到读过的筛选时被 SWR 的去重窗口挡住、不重读（默认 → 已暂缓 → 默认）：修为列表的 `dedupingInterval: 0`（`bbadf7f7`），v0 §7.7 只许缩短去重窗口（`use-session-swr.ts:37`），先例六处。
  - H6 的 `must_be_future` 让浏览器的钟落后 nerve 一天（`page.clock`，服务端的钟不动）。
  - H7 的“不在聚焦时重取”以 `page.clock.fastForward` 越过去重窗口再发 focus 与 `visibilitychange`，e2e 变异 `t13-emut-h7-focus-refetch` 证明这一格抓得到聚焦重取。
- **T14 与焦点的两个修复**：真实引擎里查出两个焦点缺陷——自定义暂缓的提交按钮在途停用时被浏览器失焦（`keepFocusIn` 先把焦点给弹窗，之后迟到的 blur 结束了卡片的持有），以及卡片的控件在途停用时失焦到 body。按根因修：`7aa8fe92`（持有不被焦点回到卡片之后才到的 blur 结束）、`ee14f2fb`（失焦到无处时卡片仍持有；离开后焦点给下一张，留下则控件再启用时回到它）、`2a112d81`（元素判断在 `ee14f2fb` 之后没有测试需要、变异存活，删去，持有回到 id）。jsdom 由脚本派发事件、监听器之间不跑微任务，单测抓不到，H20 (page) 三种引擎抓得到。H20 按关注点拆成两个文件（键盘与读屏、迟到的回答），标题同为 “H20 (page)”。接受：控件再启用时焦点回到它，若用户其间点了空白处也一样（`ModalCore` 的 `keepFocusIn` 同此）。
- **故事时长超过 8 秒的先例**（主控接受，都在 R24 的 30 秒之下）：H1 (page)（T12 时 firefox 9.8、webkit 11.5，两条测试、第二条开三个浏览器）；H5（T14 时 firefox 17.0：100 个标签约 1.9 秒与页内 100 次归档）；H20 迟到（firefox 18.8：100 张卡片的“加载更多”、Beta 工作区与第二个账号）。检查点 2 的数字见 §6：H6 与 H20 键盘在 firefox、webkit 也过 8 秒，同样在 30 秒之下。

## 4. 向负责人说明

- **R6：顶栏的未读点不是实时的。** v0 §7.7 的会话取数只有一份 SWR 配置、聚焦时不重取，顶栏在一个工作区里也不重新挂载，所以点只在进入工作区、进通知页、刷新与自己的写之后更新；别人让你收到新通知，点要等这些时刻才亮。Plane 在窗口聚焦时重取。P4 照规则走、登记差异，没有等这个决定。若要实时的点，可选 **(c)**：为这一个键开聚焦重取的例外，改 v0 §7.7 一句与 `useSessionSWR`（spec §5 的后续：P8 之前任一页面阶段）。
- **R25 的缺口：既归档又在暂缓中的通知。** 服务端的 `archived`、`snoozed` 两个筛选是相与的等式，页面发不出两者都为真的列表；页面已不造出这种通知（视图收窄卡片的动作），但经接口造出的那一条在网页的每个视图里都看不到，直到暂缓到期（之后出现在“已归档”里）。是否让 `archived=true` 也列暂缓中的，是 P1c 已合入合同的语义、跨模块，P4 不改；进 M7 收官的报告请负责人定。

## 5. 门禁与 CI

| 时点 | 本地 | 分支 CI |
|---|---|---|
| 检查点 0（`0c87739b`） | 七项一次过（bootstrap 460.4 / 644.7，e2e 494 passed）；m6-pages 三引擎 135 过（137 秒） | 44999385236 attempt 1 全绿 |
| 检查点 1（`5fe40067`） | build 与 e2e 493 过 1 败：F11 (page) [local] 以 M6-M8-07 的签名失败（`net::ERR_ABORTED` 的资产内容读，test-results 存 `cp1-test-results`），只重跑该文件一次 7 条全过（`cp1-e2e-f11-rerun`）；其余六项一次过（bootstrap 462.2 / 651.8）；m4-detail 三引擎 132 过（72 秒），m6-pages 三引擎 135 过（141 秒） | 45018858067 attempt 1 全绿 |
| 检查点 2（`1325790c`） | `final.sh cp2` 一次全过、没有重跑：七项（bootstrap 460.0 / 649.4，e2e 503 passed）；m7-pages 三引擎 27 过（48.3 秒），m6-pages 135 过，m6-acceptance 33 过；m7-pages chromium `--repeat-each=3` 27 过（`cp2-m7-repeat`，38.2 秒） | 45033492263 attempt 1 全绿 |

| 运行 | server (0) Test / bootstrap | server (1) Test / bootstrap | web（Lint / Unit / Build） | e2e (1) | e2e (2) |
|---|---|---|---|---:|---:|
| main 44989539530（基线） | 692 / 498.9 | 757 / 736.3（61.4%） | 607（64 / 501 / 10） | | |
| 检查点 0（44999385236） | 663 / 474.5 | 757 / 735.8（61.3%） | 913（94 / 770 / 19） | 663 | 736 |
| 检查点 1（45018858067） | 675 / 488.8 | 914 / 886.6（73.9%） | 618（60 / 511 / 10） | 697 | 729 |
| 检查点 2（45033492263） | 668 / 483.7 | 771 / 750.0（62.5%） | 607（61 / 509 / 10） | 791 | 745 |

- 检查点 0 的 CI bootstrap 与检查点 1、2 的 web Build 由主控在 T15 以作业日志与 jobs 接口补记。
- **web 作业**（20 分钟期限、16 分钟即 960 秒的报告线）：检查点 0 的 913 秒比 main 的 607 慢约 1.5 倍且各步同比，T2 只加了几个毫秒级的测试，是慢的 runner；检查点 1、2 回到 618、607，没到报告线。
- **第 1 路**（本机以检查点 0 的 644.7 秒为基线、+30 秒停查；CI 的八成即 960 秒只报告）：本机 +7.1、+4.7。检查点 1 的 CI bootstrap 886.6 秒（73.9%）没到八成，P4 不改 Go：同一运行第 0 路非 bootstrap 的包合计 654.0 秒、对 main 的 676.9，第 0 路的 runner 不慢，第 1 路的摆动是 runner（P3d 检查点 2 曾 78.6%）；检查点 2 回到 750.0（62.5%）。
- **e2e 两片**（35 分钟期限、30 分钟报告线）：检查点 2 比检查点 1 各多约 1 分钟（791、745），与推送前的估算相符。

**死成员与告警**

- 死成员（M1 交接）：T2 基线 1618 行（成员 1224、prop 390、导出 4；`--rows M7` 95 行），T15 终表 `t15-dead-final` 1611 行（成员 1214、prop 393、导出 4；`--rows M7` 98 行），与 T14 简化之后的一份逐行相同。
  - **消失的**：`@nerve/types` 的 `workspace-notifications.ts` 17 行（M3 交接 `:29`）、旧 store 的 `updateNotification` 两行与 `mutateNotifications`、旧筛选菜单的 `label`、旧卡片的 `renderCommentBox` 与 `buttonClassName`、`sidebar-item.tsx` 的两个 `className`；其余是标记或计数的变化。
  - **新出现、留下的**：
    - `inbox/content/root.tsx:28` `isNotificationEmbed`、`:29` `embedRemoveCurrentNotification`（prop，唯一的传入者是删掉的旧通知预览）——P5 接回。
    - `work-item-filters/filters-hoc/shared.ts:26` `enableSaveView`（prop，T11 去掉两个传入者；同一文件的 `saveViewOptions`、`enableUpdateView`、`updateViewOptions`（`:27`、`:31`、`:32`）在基线里就没有传入者）——P6a/P6b 接回或删除，读它们的是 `filters-hoc/project-level.tsx`、`workspace-level.tsx` 与 `rich-filters/filters-row.tsx` 的保存与更新分支。
    - `notification.service.test.ts:39-44` 六个字面量（T3 的 `expectTypeOf` 查询形状），以及 `page-state.test.tsx:15`、`root.test.tsx:25`、`:47-49`、`:60`、`sidebar/list.test.tsx:31` 测试替身的 prop 与字面量：读者只有测试本身（基线已有 706 行同类）。
  - **M7 其余领域的 98 行**（`rows-M7-t15.tsv`）：P5 收集箱 39 行（`intake-description-version.ts` 15、`project-inbox.store.ts` 9、`types/inbox.ts` 3、`inbox-issue.store.ts` 1、`inbox-issue-status.tsx` 1、上面两个 prop，测试 `create-root.files.test.tsx` 5、`create-root.request.test.tsx` 3）；P6 视图 30 行（`types/workspace-views.ts` 12、`types/view-props.ts` 10、`global-view.store.ts`、`views.helper.ts`、`constants/views.ts`、`default-view-quick-action.tsx` 各 1，测试 4）；P7 收藏与首页 14 行（`new-fav-folder.tsx` 4、`favorite.store.ts` 2、`ui/favorite-star.tsx` 2、`types/favorite/favorite.ts` 1，首页测试 5）；路径正则收进来的 M4 列表测试的替身 9 行（`list/view.test.tsx` 4、`spreadsheet/view.test.tsx` 5）；P4 自己的测试字面量 6 行。随各自的阶段处理。
- oxlint（`t15-oxlint-p4-files`，只读）：P4 改到与新建的 118 个文件里警告为零；web 31 = 上限 31，`@nerve/utils` 2 = 2，`@nerve/constants`、`@nerve/i18n`、`@nerve/types`、e2e 都是 0 = 0。web 的上限在 T5 同一提交 40 → 31（9 条全在删掉的三个旧通知文件里）；`no-empty-pattern` 升为 error，`activity-list.tsx` 被行内 disable 压住的空解构删掉。
- 关键词：`old-notification-urls`（新），`workspace-address-route` 收入通知的文件，各有命中与不命中的样例；T15 的 `t15-keywords` 过。

## 6. 故事时长（秒）

检查点 2 的 `cp2-m7-pages`（三引擎整配置 27 过，48.3 秒）；H1 一个文件两条测试，括号里是每条。

| 故事文件 | chromium | firefox | webkit |
|---|---:|---:|---:|
| `h1-subscription-page` | 8.3（4.1、4.2） | 14.2（6.3、7.9） | 12.8（5.7、7.2） |
| `h2-h4-notification-cards-page` | 4.1 | 7.3 | 5.4 |
| `h2-h4-auto-archive-page` | 2.7 | 2.1 | 3.2 |
| `h5-notification-list-page` | 8.8 | 16.8 | 14.5 |
| `h6-notification-states-page` | 8.2 | 11.1 | 9.9 |
| `h7-unread-counts-page` | 4.4 | 7.5 | 6.9 |
| `h20-notification-pages`（键盘与读屏） | 7.2 | 12.2 | 9.4 |
| `h20-notification-late-pages`（迟到的回答） | 9.7 | 19.1 | 12.6 |

- chromium 重复三次（`cp2-m7-repeat`，27 过 38.2 秒）：H1 每条 4.1–4.4，H2–H4 卡片 4.1–4.5、自动归档 1.7–5.5，H5 9.2–9.8，H6 8.0–8.3，H7 4.3–6.6，H20 键盘 7.2–8.5、迟到 9.9–10.4。
- T14 时的同一配置 27 过 48.9 秒，各文件与上表相差都在 2 秒以内（H5 firefox 17.0、H20 迟到 firefox 18.8 是主控引用的数）。
- 长的原因：H5 造 101 条通知要 100 个标签（约 1.9 秒）与页内 100 次归档；H6 一个故事里读、归档、预设、自定义、过去的时刻、到期、全部已读、移出共八段；H20 迟到要 100 张卡片的“加载更多”、Beta 工作区与第二个账号。都低于 30 秒。
- 新的 vitest 文件多在毫秒级；较慢的是 `snooze-modal.test.tsx`（15 例约 1.2 秒，每格约 80 毫秒：弹窗的过渡、日历与列表的弹出）与 `subscription.store.test.ts`（8 例约 0.95 秒，首格动态载入 RootStore），`save-view-collapsed.test.tsx`（3 例约 0.39 秒）。

## 7. 交给 M8（M7 收官时写进 M8 的交接）

- **通知的快照在扇出任务运行时读**（P1b §2.6 的设计）：写入之后、任务运行之前关注的人也会收到那一次写的通知（H5 的造法因此先等前一条再关注，T13 的 `t13-explore-h5-timing` 抓到过 104 条）。观察，不改。
- **被拒的迟到回答在同一会话的新页面提示**：`followInSession` 按会话守、不按页面，离开通知页（或后退）之后到达的拒绝仍会提示；成功的回答本来就不提示。M4 起所有写的共同行为，页面级的守卫是横切的改动，记为观察。
- **顶栏收件箱的 Tooltip 打不开**（T2 时记下、T15 核实）：`top-navigation-root.tsx` 以 `<Tooltip label={inboxLabel} side="bottom">` 包 `AppSidebarItem`；Propel 0.3.0 的 `Tooltip` 把 children 交给 Base UI `Tooltip.Trigger` 的 `render`，Base UI 1.7.0 以 `cloneElement` 把合并的 props 与 `ref`（`[forwardedRef, registerTrigger, triggerElementRef]`）和 `onFocus`、`onPointerEnter` 等交给这个元素；`AppSidebarItem`（`core/components/sidebar/sidebar-item.tsx:95-115`）只收 `variant` 与 `item`，不转发 ref 与别的 props，`AppSidebarLinkItem`（`:77-85`）只渲染 `<Link to aria-label className>`。于是触发器从不登记、处理函数到不了 `<a>`，提示在悬停与聚焦时都不出现。Plane 同形（`apps/web/core/components/sidebar/sidebar-item.tsx:129`、`apps/web/core/components/navigation/top-navigation-root.tsx:63-79`）。它是唯一一个包着 `AppSidebarItem` 的 Tooltip；单测 `top-navigation-root.test.tsx` 用 `@/lib/fake-controls` 替掉 Tooltip，抓不到。链接的名字来自 `aria-label`（“Inbox, 5 unread”），不靠提示。修法是让 `AppSidebarItem` 转发 ref 与 props，或去掉这个 Tooltip——全应用的最后核验时定。 **主控裁定（T15）：不交 M8，在 T16 的修复轮修**——包它的是 P4 重写的顶栏，提示既是 Plane 原意又对只有图标的入口有用：让 `AppSidebarItem` 的链接转发 ref 与 Base UI 合并的 props，悬停与聚焦时提示打开，附单测（用真的 Tooltip）与变异，修好之后本条与 frontend-changes、`HO/M3-closeout.md` 的说法随之改。
- **共用的 `DateDropdown` 的名字不带所选的日子**（T9 修复 ③ 记下的疑点，台账没有主控的裁定）：按钮的 `aria-label` 恒为 placeholder（`core/components/dropdowns/date.tsx:102`），选了日子之后名字仍只是 “Pick a date”；暂缓弹窗里日期与时间两个控件的名字因此不对称（时间是 “Time, 12:30 AM”）。整个网页的 `DateDropdown` 都如此。 **主控裁定（T15）：交 M8**——`DateDropdown` 是 M4 起的共用控件（14 个产品文件），改名字牵动各页的读屏名与既有故事的定位，不在 P4 的末段单改暂缓弹窗一处（只改一处是补丁）；M8 以“所有带字段名的选择控件读作‘字段，值’”统一处理，时间选择的 “Time, 12:30 AM” 是参照。
- **交给 P5**（不是 M8）：收集箱条目以普通浮层打开时，属性里的 “State” 一栏显示占位文字 “State”（Triage 状态不在项目的状态列表里，按钮名 “State: State”），没有接受与拒绝；P5 接回分诊时处理。

## 8. 文档

- plane-diff（四、行为差异）十五行：通知页的页签、筛选与选中（R3，W6）；未读数的位置与键（R5，W3）；未读数的更新（R6，D2 的网页一侧）；自己改一条通知之后（R8，D1 的网页一侧）；全部已读（D5 的网页一侧，R9）；刷新（W4，R7）；分页与工作区（W5，R21）；卡片的文字（W2，R11、R22）；自定义暂缓（W1，R15）；键盘与名字（W7，R16）；失败（W8）；卡片的动作随视图（R25 与它的缺口）；关注（R13、R14）；新建弹窗的默认项目（`408f3e46`）；卡片离开之后的焦点（`0d107ac0`、`7aa8fe92`、`ee14f2fb`、`2a112d81`）。Plane 的行号都对过 `plane/` 的源码。
- frontend-changes：3.1 的 M7 一行改为进行中；“挂载时的取数”一行补顶栏的未读数；3.2 加通知的数据层、通知页、未读通知数、关注、收起“保存为视图”五行。
- 交接的处理结果（spec §5 的表，逐行）：M1-P2-trim-content、M1-P3-trim-platform、M1-closeout 两节、M2 §2、§3、§6、M3 §1（未读数关闭）、§2（收集箱项、17 行死成员、以 id 为 key 的布局的通知部分关闭）、§3（暂缓弹窗、两个通知菜单与页头链接、两条规则与 AM/PM 关闭；`FiltersDropdown` 一条本段不涉及）、§4（地址的通知两个、两条告警关闭；phase）、M4 的 M7-05、M5 的 M5-M7-02（写明不链）与 M5-M7-03（关闭）。没写关闭的保持 open。表的末行另写进 P1a spec `:84`、P1b spec R6 与 §5 的 P4 一条、P1c spec §5 的两条、P3b spec 的“不做”与设计 §7 `:240`、`:241`（各自的通知部分关闭，同 M6/P8 写进 M6 P4 spec 的先例）。`HO/M6-closeout.md` 没有以 P4 为关闭阶段的条目，不写。
- 设计 §12 的 P4 一行加 spec 链接，§14 的 P4 一行。

## 9. 整分支审查与集成

（T16 填：整分支审查与修复复审、终检、最终分支 CI、`--no-ff` 合并、main CI、归档。）
