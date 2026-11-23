# M7 / P4：网页的通知与关注（阶段记录）

> 状态：已完成，合入 main（noff `89be91e5`），main CI 45051719903 全绿（§9）。spec 与 plan 见 [P4 spec](../specs/P4-web-notifications.md)、[P4 plan](../plans/P4-web-notifications.md)，预检 [re-93](../../../reviews/re-93-M7P4预检.md)。基线 main `3d9293bd`（P3d 的 noff），分支 `codex/m7-p4`。台账 `.superpowers/sdd/P4-web-notifications/progress.md`（ignored，随阶段归档）。

## 1. 方法与范围

- **范围**（spec §1）：通知的数据层（service、store、纯函数照 v0 §7.7 重写，Plane 的旧数据层、手写类型、常量与工具整套删除）；通知页（两个页签与计数、类型筛选、未读 / 已归档 / 已暂缓、卡片、加载更多、刷新、全部已读、卡片的已读 / 归档 / 预设与自定义的暂缓、嵌入的工作项浮层）；顶栏的未读数；详情页头、浮层页头与命令面板的“关注 / 取消关注”；收起筛选行的“保存为视图”；守卫与告警；新的三引擎页面配置 `e2e/playwright.m7-pages.config.ts` 与八个页面故事文件（H20 在 T14 按关注点拆成两个）。M7 的第一个页面阶段，不改 Go 与契约。
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
| T15 | `da255dd9` | 文档与交接 |
| 修复轮 | `3f4c3ba8`、`6ffa9bcf`、`58eb9dd3`、`5a2d3b7e`、`88f0a788`、`afb642cd`、`03a6acf1`、`48c5cc52`、`e5a2e818`、`82287ef1`、`2ef8ef8c`；复审之后 `3df03b12`、`e20cd2e4`、`228270ed`、`dc86d298`、`1ebb0312`、本提交 | 整分支审查的 I1、m1、m2、m3、m4（连同 m7 的注释）、T15 裁定的顶栏 Tooltip、O4、m6、m5、本轮注释的折行、文档（m7 的其余）；复审的 I1（O4 的指针路径）、m1、m3、m2、O3、文档（§9） |

| 验收（spec §4） | Task | 状态 |
|---|---|---|
| A1 数据层 | T2、T3、T4、T5；修复 `39fc1282`、`b12dbdd4`、`bbadf7f7` | 达成：service 只调 `/api/v0`；按会话分代、`useSessionSWR`、键带 `loginId`；`ReconciledByKey.fetchPart`；页面状态不在 store；纯函数与陷阱 T1–T8 有单测；性质的变异皆杀 |
| A2 通知页 | T5–T9；修复 `0d107ac0`、`b143d6a1`、`7aa8fe92`、`ee14f2fb` | 达成；I4 的页面一格经工作项读被拒（§3） |
| A3 未读数 | T2、T4、T7 | 达成：顶栏与页签共用一个取数；`WORKSPACE` 与读它的 S2 的两个、G3、G8、G13 跟随（spec 写“六个故事”，读 `WORKSPACE` 的是这五个，第六个 `p2-visibility` 只读 `PROJECT_MEMBER`） |
| A4 关注 | T10、T11（C12） | 达成 |
| A5 页面故事 | T12、T13、T14 | 达成：三引擎 27 过，chromium `--repeat-each=3` 过；m6-pages、m6-acceptance、m4-detail 不回归（§5） |
| A6 删除与守卫 | T2、T5、T9、T11、T15 | 达成（§5 的死成员与告警） |
| A7 文档 | T1、T15 | 本提交（§8） |
| A8 门禁、审查与集成 | T16 | 已完成：整分支审查、修复轮、复审与复审修复轮，终检、最终分支 CI、合并与 main CI 皆一次过（§9） |

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
- **T16 修复轮的裁定**（细节与提交见 §9）：m1 不照审查建议的 `!isValidating`（不够，SWR 新挂载的第一次渲染就带着旧的失败），照 browse 详情的先例每个预览自己读一次；m2 只有全部已读受影响（SWR 的 `mutate` 在调用时定键，刷新在发出时就调），回答到达时键已变就只经 store 重读；O4 属实，回焦改为看取得焦点的控件，焦点在菜单里（面板，或指针按下的菜单项）时看菜单名字（`aria-labelledby`）所指的按钮，键盘与指针选的预设被拒时都回到暂缓按钮（复审 I1 补上指针一路）；m6 三处各取其宜：预览是有名字的区域、卡片动作看按钮本身是否画出、顶栏的点带 `data-unread-dot`；Tooltip 让 `AppSidebarItem` 的链接收下触发器的 ref 与事件。
- **T16 复审修复轮的裁定**（细节与提交见 §9）：回焦的控件按它是什么定（在菜单里的认菜单的按钮，否则是它自己），第一版的 `card.contains` 与 `:enabled` 两个判断变异存活、没有可观察的差别，删去；m1 在列表变化时被看的控件已不在 DOM 就结束 watch，不论此刻谁持有焦点；m2 的按钮变体在类型上不收包它的 props（以 `variant` 区分的并集），两个按钮调用者不变；O3 同时比分钟。O1、O2、O4、O5 不改。
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
| 终检（45048776015） | 879 / 597.5 | 890 / 864.6（72.1%） | 956（96 / 807 / 19） | 697 | 749 |
| main 45051719903（合并后） | 869 / 585.2 | 1002 / 970.9（80.9%） | 604（60 / 508 / 12） | 817 | 719 |

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
    - **处理结果（M7/P5a）：** P5 收集箱的 39 行：33 行随旧代码在 T6a–T6c 消失，上面 `inbox/content/root.tsx:28-29` 的两个 prop 在 T6a 不再收（R32，P5b 随通知预览里嵌入的分诊加回），留下 6 行是两份新建弹窗测试里 `ResizeObserver` 替身的成员（新建弹窗随 P5b）；逐条去处见 [P5a 阶段记录](P5a-web-intake-review.md) §5。
- **T16 修复轮之后**（`t16-dead`）1612 行，比 T15 终表多一行：`core/hooks/use-project-read.ts` 的 `revalidateOnMount`（m1 给 `useProjectRead` 加的可选参数的类型字面量成员，只经展开 `...config` 交给 `useSessionSWR` 读，工具不把展开算作读；它的读者是 `useSessionSWR` 的 `config?.revalidateOnMount`）。其余去掉行号之后逐行相同，只有 `root.test.tsx` 替身的三行读数随新格变化。oxlint 在本轮改到的 19 个 web 文件里为零，web 仍是 31 = 上限 31（`t16-oxlint-files`）。
- **T16 复审修复轮之后**（`t16r-dead`）1614 行，比 `t16-dead` 多两行：`leaving-focus.test.tsx` 的 `watchesMade` 记下的字面量成员 `element`、`ended`（m1 的单测，读者只有测试本身，与基线的同类一样）。另有 30 行的标记多了 `str`：15 个测试文件替身 observer 的 `observe`、`disconnect`，因为 m1 的单测以字符串 `"observe"`、`"disconnect"` spy `MutationObserver.prototype`，工具把同名字符串记作可能的读；行数不变。其余去掉行号之后逐行相同。oxlint 在本轮改到的 6 个 web 文件里为零，web 仍是 31 = 上限 31（`t16r-oxlint-files`）。
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
- **T16 修复轮**（`t16-m7-pages`，三引擎整配置 27 过 49.7 秒，chromium / firefox / webkit）：H1 每条 4.0–7.7，H2–H4 卡片 4.1 / 5.8 / 4.7，自动归档 2.7 / 4.9 / 3.2，H5 9.2 / 18.5 / 16.7，H6 11.2 / 14.3 / 14.3（m5 的窗口放到 5 秒，多约 3 秒），H7 5.5 / 9.5 / 6.1，H20 键盘 8.1 / 11.2 / 10.8（O4 一段），H20 迟到 10.2 / 19.3 / 13.0；chromium `--repeat-each=3`（`t16-repeat`，本轮改到的与夹具牵动的五个文件 15 过 34.3 秒）：H2–H4 卡片 3.2–4.2，H6 10.7–11.0，H7 5.0–5.7，H20 键盘 7.1–8.3，H20 迟到 10.1–10.2。都在 30 秒之下。
- **T16 复审修复轮**（`t16r-m7-pages`，三引擎整配置 27 过 47.9 秒，chromium / firefox / webkit）：H1 每条 4.0–6.7，H2–H4 卡片 3.5 / 5.4 / 3.6，自动归档 2.9 / 4.5 / 3.0，H5 9.3 / 16.2 / 15.2，H6 11.2 / 13.6 / 12.8，H7 5.6 / 6.3 / 6.0，H20 键盘 7.3 / 10.8 / 10.3（O4 一段多了指针那次），H20 迟到 10.2 / 17.5 / 12.5；chromium `--repeat-each=3` 的 H20 键盘（`t16r-repeat`，3 过 21.4 秒）6.4–6.9。都在 30 秒之下。
- 新的 vitest 文件多在毫秒级；较慢的是 `snooze-modal.test.tsx`（15 例约 1.2 秒，每格约 80 毫秒：弹窗的过渡、日历与列表的弹出）与 `subscription.store.test.ts`（8 例约 0.95 秒，首格动态载入 RootStore），`save-view-collapsed.test.tsx`（3 例约 0.39 秒）。

## 7. 交给 M8（M7 收官时写进 M8 的交接）

- **通知的快照在扇出任务运行时读**（P1b §2.6 的设计）：写入之后、任务运行之前关注的人也会收到那一次写的通知（H5 的造法因此先等前一条再关注，T13 的 `t13-explore-h5-timing` 抓到过 104 条）。观察，不改。
- **被拒的迟到回答在同一会话的新页面提示**：`followInSession` 按会话守、不按页面，离开通知页（或后退）之后到达的拒绝仍会提示；成功的回答本来就不提示。M4 起所有写的共同行为，页面级的守卫是横切的改动，记为观察。
- **顶栏收件箱的 Tooltip 打不开**（T2 时记下、T15 核实）：`top-navigation-root.tsx` 以 `<Tooltip label={inboxLabel} side="bottom">` 包 `AppSidebarItem`；Propel 0.3.0 的 `Tooltip` 把 children 交给 Base UI `Tooltip.Trigger` 的 `render`，Base UI 1.7.0 以 `cloneElement` 把合并的 props 与 `ref`（`[forwardedRef, registerTrigger, triggerElementRef]`）和 `onFocus`、`onPointerEnter` 等交给这个元素；`AppSidebarItem`（`core/components/sidebar/sidebar-item.tsx:95-115`）只收 `variant` 与 `item`，不转发 ref 与别的 props，`AppSidebarLinkItem`（`:77-85`）只渲染 `<Link to aria-label className>`。于是触发器从不登记、处理函数到不了 `<a>`，提示在悬停与聚焦时都不出现。Plane 同形（`apps/web/core/components/sidebar/sidebar-item.tsx:129`、`apps/web/core/components/navigation/top-navigation-root.tsx:63-79`）。它是唯一一个包着 `AppSidebarItem` 的 Tooltip；单测 `top-navigation-root.test.tsx` 用 `@/lib/fake-controls` 替掉 Tooltip，抓不到。链接的名字来自 `aria-label`（“Inbox, 5 unread”），不靠提示。修法是让 `AppSidebarItem` 转发 ref 与 props，或去掉这个 Tooltip——全应用的最后核验时定。 **主控裁定（T15）：不交 M8，在 T16 的修复轮修**——包它的是 P4 重写的顶栏，提示既是 Plane 原意又对只有图标的入口有用：让 `AppSidebarItem` 的链接转发 ref 与 Base UI 合并的 props，悬停与聚焦时提示打开，附单测（用真的 Tooltip）与变异，修好之后本条与 frontend-changes、`HO/M3-closeout.md` 的说法随之改。 **已修（`afb642cd`，§9）**：`AppSidebarItem` 的链接收下触发器给的 ref 与事件，悬停与聚焦时提示打开；单测 `core/components/sidebar/sidebar-item.test.tsx` 用真的 Propel Tooltip，两个变异皆杀；frontend-changes 与 `HO/M3-closeout.md` 的说法已改。本条不再交给 M8。复审 m2 之后按钮变体在类型上不收包它的 props（`dc86d298`），以 `<Tooltip>` 包按钮变体过不了类型检查。
- **拨慢那天重复的半点**（m4 记下的已知限制）：夏令时结束那天（如洛杉矶 11 月第一个周日）01:00 与 01:30 各出现两次，自定义暂缓的时段以 `new Date(y, m, d, h, mm)` 取浏览器时区的那一刻，只能选到第一次（夏令时的那一个），第二次够不着；拨快那天跳过的 02:00、02:30 已不给（`88f0a788`），半小时拨快的时区（Australia/Lord_Howe）跳过的 02:00 也不给（`1ebb0312`，复审 O3）。要选到第二次得让时段带上时区偏移，不在 P4 做；观察，不改。
- **共用的 `DateDropdown` 的名字不带所选的日子**（T9 修复 ③ 记下的疑点，台账没有主控的裁定）：按钮的 `aria-label` 恒为 placeholder（`core/components/dropdowns/date.tsx:102`），选了日子之后名字仍只是 “Pick a date”；暂缓弹窗里日期与时间两个控件的名字因此不对称（时间是 “Time, 12:30 AM”）。整个网页的 `DateDropdown` 都如此。 **主控裁定（T15）：交 M8**——`DateDropdown` 是 M4 起的共用控件（14 个产品文件），改名字牵动各页的读屏名与既有故事的定位，不在 P4 的末段单改暂缓弹窗一处（只改一处是补丁）；M8 以“所有带字段名的选择控件读作‘字段，值’”统一处理，时间选择的 “Time, 12:30 AM” 是参照。
- **交给 P5**（不是 M8）：收集箱条目以普通浮层打开时，属性里的 “State” 一栏显示占位文字 “State”（Triage 状态不在项目的状态列表里，按钮名 “State: State”），没有接受与拒绝；P5 接回分诊时处理。
  - **处理结果（M7/P5a）：** 随 P5b：通用详情与浮层在条目未被接受时隐去状态（Q12），通知预览里嵌入收集箱的分诊（Q13，[P5a spec](../specs/P5a-web-intake.md) §1、R32）。P5a 在收集箱页里接回分诊，浮层不变。

## 8. 文档

- plane-diff（四、行为差异）十五行：通知页的页签、筛选与选中（R3，W6）；未读数的位置与键（R5，W3）；未读数的更新（R6，D2 的网页一侧）；自己改一条通知之后（R8，D1 的网页一侧）；全部已读（D5 的网页一侧，R9）；刷新（W4，R7）；分页与工作区（W5，R21）；卡片的文字（W2，R11、R22）；自定义暂缓（W1，R15）；键盘与名字（W7，R16）；失败（W8）；卡片的动作随视图（R25 与它的缺口）；关注（R13、R14）；新建弹窗的默认项目（`408f3e46`）；卡片离开之后的焦点（`0d107ac0`、`7aa8fe92`、`ee14f2fb`、`2a112d81`）。Plane 的行号都对过 `plane/` 的源码。
- frontend-changes：3.1 的 M7 一行改为进行中；“挂载时的取数”一行补顶栏的未读数；3.2 加通知的数据层、通知页、未读通知数、关注、收起“保存为视图”五行。
- 交接的处理结果（spec §5 的表，逐行）：M1-P2-trim-content、M1-P3-trim-platform、M1-closeout 两节、M2 §2、§3、§6、M3 §1（未读数关闭）、§2（收集箱项、17 行死成员、以 id 为 key 的布局的通知部分关闭）、§3（暂缓弹窗、两个通知菜单与页头链接、两条规则与 AM/PM 关闭；`FiltersDropdown` 一条本段不涉及）、§4（地址的通知两个、两条告警关闭；phase）、M4 的 M7-05、M5 的 M5-M7-02（写明不链）与 M5-M7-03（关闭）。没写关闭的保持 open。表的末行另写进 P1a spec `:84`、P1b spec R6 与 §5 的 P4 一条、P1c spec §5 的两条、P3b spec 的“不做”与设计 §7 `:240`、`:241`（各自的通知部分关闭，同 M6/P8 写进 M6 P4 spec 的先例）。`HO/M6-closeout.md` 没有以 P4 为关闭阶段的条目，不写。
- 设计 §12 的 P4 一行加 spec 链接，§14 的 P4 一行。

## 9. 整分支审查与集成

- **整分支审查**（fable，只读，`3d9293bd..da255dd9`，台账旁的 `review-whole-branch.md`）：修复后可合并；严重 0、重要 1、次要 7、观察 7。I1 刷新在途时点“加载更多”，列表静默漏掉一段（`fetchMore` 的合并不认游标）；m1 预览被上一次留下的项目读失败立刻再拒；m2 页头的全部已读或刷新在途时换筛选，迟到的重读写进新筛选的 SWR 键；m3 卡片的“暂缓至”在中文里是英文的 12 小时制；m4 拨快那天的时段有重复与够不着的时刻；m5 H6 (page) 的“暂缓到两秒之后”只有两秒余量；m6 三个 e2e 定位器认样式类；m7 文档与注释三处。观察 O1–O7（O4：预设的暂缓被拒时焦点可能停在 body，未核实）。
- **修复轮**（主控裁定：I1、m1–m7 全修；T15 裁定的顶栏 Tooltip 同轮修；O4 在真实引擎核实、属实即修；O1–O3、O5–O7 不改）。每个产品修复都先有失败的单测格（RED）与一个能让它失败的变异（`mutate.py`，`restored=yes`），`make lint-web`、`make knip`、`make test-web` 过：
  - `3f4c3ba8`（I1）：`fetchMore` 的合并只在持有页的 `next_cursor` 仍是这一页发出时的游标时接上，否则留持有页（游标即位置：只加载过一页、刷新答回同一个游标时照常接上）；`Reconciled` 不改。单测与原有一格次序相反（刷新的读在外 → `fetchMore` → 第一页先答 → 后续页后答），变异“合并不比游标”杀。
  - `6ffa9bcf`（m1）：审查建议的“另要 `!isValidating`”不够（`t16-m1-try-validating` 新格仍失败）：读失败之后 SWR 2.4.2 的缓存是 `{error, isValidating: false}`，新挂载的第一次渲染返回的正是它，挂载读在 layout effect 里才置 `isValidating`，离散事件的提交先冲刷 passive effect。照 browse 详情的先例，预览关掉 SWR 的挂载读、挂载时自己 `mutate()` 一次，这次落定之后才判定；`useProjectRead` 加可选的 `{ revalidateOnMount?: false }`，另两个读者不传、不变。两个变异（不看 `answered`；不关挂载读，一个预览两个读）皆杀。
  - `58eb9dd3`（m2）：SWR 的绑定 `mutate` 在调用时读 `keyRef.current` 定键——刷新在发出时就调，写进发出时的键，不受影响（加一个守门格）；全部已读在回答之后才调。页头以 ref 记页面此刻的筛选，回答到达时键已不是发出时的就只经 store 的 `fetchList` 重读旧筛选（R9 不变）。不选“按键 mutate”：要从 `swr` 直接取全局 `mutate`，或给 `use-session-swr` 加接口。
  - `5a2d3b7e`（m3）：`card/snooze-time.ts` 的 `timeLabel(moment, locale)`（`Intl.DateTimeFormat(locale, { hour: "numeric", minute: "2-digit" })`），弹窗的 `slotLabel` 与卡片的“暂缓至”共用；日期一半仍是 `renderFormattedDate`。英文的时间随之从 “08:30 PM” 变为 “8:30 PM”。
  - `88f0a788`（m4，连同 m7 的 `snooze-modal.tsx:48` 注释重排）：`snoozeSlots` 滤掉换算回来不是这个钟点的时段，拨快那天 46 项；拨慢那天的第二个 01:00、01:30 记为已知限制（§7）。
  - `afb642cd`（顶栏 Tooltip，T15 裁定）：`AppSidebarItem` 的链接收下包它的元件给的 props（`ComponentPropsWithRef<"a">` 去掉地址、名字、样式与 children），经 `AppSidebarLinkItem` 展开到 `Link`，自己的地址、名字与样式优先；React 19 的 ref 是普通 prop。按钮变体与它的两个调用者不变。jsdom 可用真的 Propel Tooltip：键盘聚焦与鼠标停留（Base UI 以 `restMs` 开，要 `mousemove`）都打开；Base UI 的触发器不设 `aria-describedby`。
  - `03a6acf1`（O4）：真实引擎属实——拒绝扣住 800 ms 时三种引擎的焦点都停在 body；立即拒绝时才碰巧回到按钮（早于 Headless UI 还焦点）。根因：旧 hook 只看“失焦到无处的那个元素”再启用，而预设从菜单选出时失焦的是菜单面板（随菜单卸载、永不再启用），暂缓按钮在途停用，Headless UI 还焦点落空。`useLeavingFocus` 改为在取得焦点时看卡片自己的、能被停用的控件（`:enabled`，不是菜单面板、也不是弹窗），它再启用、焦点无处时回到它——焦点进了它打开的菜单或弹窗，看的仍是打开者。H20 (page) 键盘故事加一段：评论的卡片以键盘选“1 day”，PATCH 被代理的 503 拒（不经 nerve），菜单走、两帧之后 body 有焦点再放行，卡片留下、焦点回到暂缓按钮；修复前的构建三种引擎都在这一步失败（`t16-o4-e2e-red-3`）。watch 里“卡片仍持有”的判断变异存活，按不过度设计删去（`nowhere()` 已保证不抢有焦点的元素）；“再启用时焦点已去别处”另加守门格。这一版只覆盖键盘：指针按下菜单项会让它取得焦点，见下面复审修复轮的 `3df03b12`。
  - `48c5cc52`（m6）：预览区成为名为 “Preview”（zh-CN “预览”，`notification.a11y.preview`）的区域，夹具按角色与名字找；卡片的动作不改产品，夹具新的 `expectActionsShown` 看动作按钮本身是否画出（`checkVisibility({ opacityProperty: true })`），H6、H20 各自的 `actionsOf` 删去；顶栏的点是装饰（链接名已说未读数），按 `data-notification-id` 的先例带 `data-unread-dot`。三条期望错误的版本各在被改的断言处失败。
  - `e5a2e818`（m5）：H6 的暂缓窗口 2 → 5 秒（`unreadReach` 的 10 秒上限仍够），chromium 多约 3 秒；e2e 变异（不暂缓）在刷新后的 `toHaveCount(0)` 处失败。
  - `82287ef1`：本轮三处超过 120 列的注释折行（只改注释）。`2ef8ef8c`（m7 的文档一半）：§1 的“八个页面故事文件”；plane-diff 依赖 I1 的“刷新”“分页与工作区”两行与“卡片离开之后的焦点”“自定义暂缓”两行；frontend-changes 与 `HO/M3-closeout.md` 的 Tooltip 说法；§2、§3、§5、§6、§7 与本节。
  - 修复轮的验证：`make build build-s3-fixture`；m7-pages 三引擎 27 过（`t16-m7-pages`，49.7 秒）；chromium `--repeat-each=3` 改到的与夹具牵动的五个文件 15 过（`t16-repeat`）；`make lint-web`、`make knip`、`make test-web`、`tools/keywords.mjs` 过（`t16-final-*`）；死成员 1612（§5）。
- **复审**（fable，只读，`da255dd9..2ef8ef8c` 的 11 个提交，台账旁的 `re-review-fix-round.md`）：修复后可合并；严重 0、重要 1、次要 3、观察 5。I1 O4 只修了键盘一路：菜单项是 `tabindex=-1` 的按钮，指针按下时它取得焦点，`focusIn` 把 watch 从暂缓按钮换到菜单项，菜单项随菜单卸载，被拒之后焦点仍停在 body；m1 焦点离开卡片之后 holder 已清，这张卡片再离开时 order effect 在 `stopReturning()` 之前返回，watch 留着脱离的控件（留存，不抢焦点）；m2 `AppSidebarItem` 的按钮变体在类型上收下包它的 props 又丢掉，`<Tooltip>` 包按钮变体能过类型检查却打不开；m3 `sidebar-item.tsx` 的注释说触发器合并 `aria-describedby`，与本节 `afb642cd` 的说法不符。观察：O1 m1 的 `answered` 不复位（`readable` 由真变假时先按旧回答判定，修复前同样，不是回退）；O2 m2 的 ref 在 passive effect 写、SWR 的键在 layout effect 写，换筛选是点击、提交时就冲刷 passive effect，构造不出真实情形；O3 m4 只比钟点，半小时拨快的时区留下与 “02:30” 重复的 “02:00”；O4 holder 换到另一张卡片之后 watch 可仍在前一张卡片的控件上（要那张卡片的改动在途且焦点失到无处，与裁定 3 同形）；O5 `data-unread-dot` 与 `checkVisibility({ opacityProperty: true })` 无问题。对修复轮裁定的意见：m1 自己读一次、m2 只修全部已读、O4 删去 holder 判断、m6 三处各取其宜、m3 的 “8:30 PM”、m4 记下的已知限制，都接受。
- **复审修复轮**（主控裁定：I1、m1、m2、m3、O3 修；O1、O2、O4、O5 不改，理由同复审）。每个产品修复都先有失败的 RED 与能让它失败的变异（`restored=yes`），`make lint-web`、`make knip`、`make test-web` 过：
  - `3df03b12`（I1，O4 的指针路径）：真实引擎探查（`t16-o4-ptr-probe`，`2ef8ef8c` 的构建，拒绝扣 800 ms）三引擎属实，焦点停在 body。事件序列三引擎相同：暂缓按钮的 pointerdown 被 Headless UI 2.2.10 阻止默认，按钮从不取得焦点，焦点直接进菜单面板；菜单项的 mousedown 不被阻止，点它时它取得焦点（WebKit 也是）。修法：回焦的控件按它是什么定（`controlOf`）——在 `[role="menu"]` 里的（面板或菜单项）认菜单的按钮（面板的 `aria-labelledby`，Headless UI 设为按钮的 id），否则是取得焦点的元素；键盘、指针，连同指针从不让打开者取得焦点的情形，焦点一进菜单，看的就是菜单的按钮。第一版的 `card.contains` 与 `:enabled` 两个判断的三个变异都存活，删去：弹窗的控件取得焦点时 watch 移过去也没有可观察的差别（提交被拒时焦点留在弹窗里；取消时 Headless UI 把焦点还给暂缓按钮，watch 随之回来）。删去之后取得焦点即换 watch，watch 与 holder 一起换，复审 O4 的情形只剩取得焦点的不是 HTML 元素、或菜单没有名字时。单测一格：指针开菜单（面板取得焦点）、菜单项先取得焦点再 click，被拒之后焦点回到暂缓按钮。H20 (page) 键盘故事的 O4 一段改为 `refusedPreset` 帮手：先键盘选 “1 day”，再指针点开菜单点 “3 days”，各被代理的 503 扣住，菜单走、两帧之后 body 有焦点再放行，焦点回到暂缓按钮；`2ef8ef8c` 的构建三引擎都在指针那次失败（`t16-o4-ptr-e2e-red`，键盘那次过），修复后三引擎过（`t16-o4-ptr-e2e-green`）。变异：菜单里的不换成按钮、只换面板不换菜单项、不回焦、不看 `nowhere()`、失到无处也算离开，皆杀。
  - `e20cd2e4`（m1）：`returning` 记 `{ control, watch }`；列表变化时，被看的控件已不在 DOM（随它的卡片走了，不论此刻谁持有焦点）就结束 watch。单测记下 `MutationObserver` 的 observe 与 disconnect：焦点去了别处之后卡片被归档，它的控件上没有未结束的 watch。变异（去掉这一判断）杀。
  - `228270ed`（m3）：注释改为触发器合并 ref 与处理函数（只改注释）。
  - `dc86d298`（m2）：`AppSidebarItemProps` 改为以 `variant` 区分的并集，链接（`variant` 可省）收下 `AppSidebarLinkWrapperProps`，按钮（`variant: "button"`）只有 `item`；两个按钮调用者不变。单测以 `@ts-expect-error` 钉住按钮变体不收 `onFocus`（由 `check:types` 检查）；变异（按钮一支也并上包它的 props）→ TS2578。
  - `1ebb0312`（O3）：`snoozeSlots` 同时比钟点与分钟；单测 Australia/Lord_Howe 2026-10-04 47 项，“01:30” 之后是 “02:30”，47 个时刻各不相同。顺手补全 `88f0a788` 留下的半句注释。变异（只比钟点）杀。
  - 本提交：§2、§3、§5、§6、§7、本节与 plane-diff 的“卡片离开之后的焦点”一行；hook 的注释已在 `3df03b12` 改为键盘与指针两条路。
  - 复审修复轮的验证（`1ebb0312`）：`make build build-s3-fixture`（`t16r-final-build`）；m7-pages 三引擎 27 过（`t16r-m7-pages`，47.9 秒）；chromium `--repeat-each=3` 的 H20 键盘（本轮唯一改到的故事）3 过（`t16r-repeat`）；`make lint-web`、`make knip`、`make test-web`、`tools/keywords.mjs` 过（`t16r-final-*`）；改到的 6 个 web 文件 oxlint 为零，web 31 = 上限 31（`t16r-oxlint-files`）；死成员 1614（§5）。
- 复审修复轮的改动（`2ef8ef8c..1ebb0312`）由主控逐项读过：指针一路照复审的建议修，`card.contains` 与 `:enabled` 删去之后弹窗里的控件也会被看，提交按钮在途停用、被拒后再启用时焦点回到它，与弹窗的意图一致；不再做第二次复审。合并前的终检、最终分支 CI、`--no-ff` 合并、main CI 与归档在 P5 的 T1 补记。
- 以下在 P5a 的 T1 补记，数字取自归档台账的末行（main CI 两路 server 的 Test 时长取自 jobs 接口的步骤时刻，预检 re-94 O4）：
  - **合并前的终检** `final.sh fin`（`3280e48e`，复审修复轮之后补的文档提交）一次过、没有重跑：七项收据全 0（bootstrap 第 0 路 470.6 秒、第 1 路 654.0 秒，比本机基线 644.7 多 9.3，线内；e2e 503 passed）；m7-pages 三引擎 27 过（47.5 秒），m6-pages 135 过，m6-acceptance 33 过。
  - **最终分支 CI** 45048776015 第一次全绿：server (0) Test 879 / bootstrap 597.5，server (1) Test 890 / bootstrap 864.6（期限的 72.1%）；web 956 秒（Lint 96 / Unit 807 / Build 19，各步对 main 同比慢约 1.6 倍，同一运行第 0 路的 bootstrap 也慢：慢 runner，贴着 960 秒的报告线、在 20 分钟期限之内）；e2e 两片 697、749 秒。
  - **合并**：`--no-ff` 合入 main，`89be91e5`（父 `3d9293bd`、`3280e48e`），推送后以 `ls-remote` 核对。**main CI** 45051719903 第一次全绿：server (0) Test 869 / bootstrap 585.2，server (1) Test 1002 / bootstrap 970.9（80.9%，第一次越过八成的报告线）；web 604 秒（60 / 508 / 12），e2e 两片 817、719 秒。P4 不改 Go，同一份 Go 在 main 44989539530 为 736.3 秒；这次两路的 bootstrap 同比慢 1.2–1.3 倍而同一运行的 web 正常，是两台 server runner 慢。已报告负责人，建议只调 `tools/go-test-shard.sh` 的分界、把第 1 路的一部分测试移到第 0 路（不加作业、不改期限），等负责人定。
  - **归档与清理**：归档 `.superpowers/sdd/M7-P4-integration-20261123`（载荷 3562 个文件，分类只有可复现与证据两种、没有未知，`source_unchanged` 为真，清理之后核过）；`cleanup.log`（2026-11-22T18:23:33Z）：worktree `m7-p4` 已删，本地分支 `codex/m7-p4`（在 `3280e48e`）与远端分支已删。
