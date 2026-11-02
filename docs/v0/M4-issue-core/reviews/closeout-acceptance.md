# M4 收尾验收台账

状态：实施与终审完成、集成待办。最终准确候选SHA与三个分支CI作业由外置final-delivery.json事后签结；不合并。D18、最终main绑定及完成标记保持待集成。

基线 `01e40e113e11365a4eeacc2794973de2751dde61`；实施起点 `726f7156c2a053bd67fba1bfcaef5541df347749`；全期范围 `1eb9686a6ca527984337821af5fa7efae8c620e2..候选HEAD`。主目录 main 保持不动。P16 main CI [41290496985](https://github.com/open-nerve/NerveProject/actions/runs/41290496985) 和本分支预检 CI [41293001954](https://github.com/open-nerve/NerveProject/actions/runs/41293001954) 均为准确 SHA、attempt 1、success。

## 1. 证据协议

原始证据在 worktree `.superpowers/sdd/M4-closeout/`，技能 brief/进度在 `.superpowers/sdd/closeout/`；两根均保留。`t1-source-files.json` 逐文件 SHA/bytes，`t1-review-ranges.json` 固定两套 diff 清单；`t1-requirements.json` 保存设计每行原文、故事全部增长、36 个 Phase 来源和八份输入全文。来源的历史勾选与本次实际验收分开。

计划 14 Task、净手写新增/修改约 3500–6500 行；真实新增/删除/生成/文档在 T13 单列。集成包为 T14 Step 1，Step 2–4 需之后授权。

## 2. 十八条完成标准

| ID | 范围 | Owner | 当前证据与结论 |
|---|---|---|---|
| D01 | I1–I22适用的页面/API/任务版本与全部增长断言、S1/S2在CI通过 | T1/T4/T13/T14 | 分支范围通过：[22故事增长](closeout-web-contracts.md#1-二十二故事的实际观察)、最终默认E2E及三引擎C12见[review](closeout-review.md)；准确分支CI由final-delivery.json签结，最终main CI待T14 |
| D02 | 完整权限矩阵/访客一致性/26交错、拒绝不取锁、真实池与503 | T2/T3/T9/T10 | 通过：T2/T3矩阵与26交错，43测试各5次、零skip/race/意外40P01；[直接证据](closeout-server-contracts.md)。T9真实10连接/K2/queue64、200并发及78次无关读，[规模](closeout-performance.md#5-删除与热项目连接池)；T10复核、最终F与六份race输入绑定已核 |
| D03 | issue与project双向公开码、access判定/规则/操作目录/矩阵齐全 | T3/T13 | 通过：operation/规则/矩阵/issue与project双向公开码，[直接证据](closeout-server-contracts.md)；最终F全部Go包和架构门禁通过 |
| D04 | 21个合法排序码（含各自升降取值，sort_order仅升序）/筛选/分组/续页/共享fixture/游标可见性/100k/连接设置/9反例 | T3/T4/T8 | 通过：Go/前端排序、游标、分组、筛选、共享fixture和九反例；T8三规模39场景及runtime双连接，[实测与代价](closeout-performance.md)；最终F和C12原故事全部保留 |
| D05 | 事务投递提交/回滚、两类缺worker变异、registrationRows、固定时钟/无固定余量 | T3/T10/T13 | 通过：实际InsertTx/river_job、两类漏worker有效变异及源码恢复；[服务端](closeout-server-contracts.md)、最终clock生命周期/I16/I21和F；业务clock与River实时时间不混用 |
| D06 | P7出站IANA/DNS/固定IP/每跳重检/本机地址与I10 blocked | T3/T6/T10 | 通过：IANA每网段首尾、DNS全部候选/每跳固定IP/本机危险分支零dial，[服务端](closeout-server-contracts.md)；I10真实页面blocked/metadata及最终三引擎C12 |
| D07 | purge目录/外键/I21、删除窗口/交错24/25/M3有限轮询 | T3/T9/T13 | 通过：catalog/runtime/删除窗口/L24/L25与已批准迟到重试边界；[服务端](closeout-server-contracts.md)；T9七job一次完成、真实清理660026行且59天/活行保留；最终I21/F通过 |
| D08 | 七门禁、架构/SQLCSchemaScope/grants双测/CLI/keywords/lint/knip/gen | T4/T9/T13/T14 | 分支本地七门禁通过，精确argv/exit/输入/log hash与实际计数见[review](closeout-review.md)及t13-F-summary.json；分支CI另核准确HEAD，main CI待T14 |
| D09 | 八决策/十一架构问题已批准并落实 | T1/T10 | 通过：[八决策/十一架构逐项实现与证据](#15-d09-批准决定与最终实现)；R06按负责人批准A落地，没有未经批准的合同变动 |
| D10 | §13.1逐项/8输入closed；§13.2四输出及原文/收集箱细节 | T12/T13 | 通过：[交接索引](closeout-handoffs.md)，205原条款/43结论、36阶段移交核对；8输入closed、4输出open；M2/M0两段原文逐字相同、历史正文保留；最终链接/输入再核 |
| D11 | §3.23每行、§4.17每行、frontend M4、README§8.8 | T11 | 分支可执行内容通过：[44同步/45差异](closeout-sync.md)，43同步已核、SYNC44为最终集成状态；目标段/原文/hash齐全，README及前端M4边界已同步 |
| D12 | C1–C12各Phase有证据，C12最终三引擎 | T5/T6/T7/T13/T14 | 分支C12通过：[21组×3引擎](closeout-browser-matrix.md)，最终新build10各完整167项/引擎，无skip/flaky；原并集仅保留历史；历史730图已看，最终增量57状态/引擎及各组代表图另签。完整原始附件保留，main等价待T14 |
| D13 | M4 dead无未解释行；全仓no-shadow零/error；cap实数下降 | T11/T13 | 通过：[354行/102来源](closeout-dead-members.md)、历史52逐名；最终AST具名候选一致，无新增未解释行；14范围86告警、M4和全期触及路径零诊断，no-shadow零/error；最终lint/knip通过 |
| D14 | M4挂载零旧/无权请求、S2四身份、copy-hook无M4例外、13实例清空 | T4/T5/T6/T7/T11 | 通过：S2四身份精确请求/重复数、76规则13例外无命中、13历史实例AST为零；[Web合同](closeout-web-contracts.md#3-s1s2-与旧实例)、最终三引擎C12与lint/knip；copy-hook无M4例外 |
| D15 | await后跳转/提示核会话，改动hook有换账户兑现测试，C11 | T4/T7/T10 | 通过：[异步hook/调用方](closeout-web-contracts.md#2-等待后的会话与宿主)、复制晚答/路由缺口红绿、R06失权与热ID/兄弟菜单；最终C11及29项整审页面补验保持旧范围晚答无副作用 |
| D16 | 列表10k/20k/100k、中文两字搜索、规模删除请求/任务、一次清理并交M8 | T8/T9/T12 | 通过：[T8列表39/搜索13、T9删除/池/清理](closeout-performance.md)，原测量及三有效负例保留；[M8-05](../../M8-open-release/handoffs/M4-closeout.md#M8-05)完整接收代价。t13-performance-equivalence.json证实工作负载/运行SQL等价，不声称新二进制重测 |
| D17 | 规模估计与实际对照 | T1/T13 | 通过：[review§5](closeout-review.md#5-最终实现与计划的差异规模)，各Phase最终plan任务/有行估计数与first-parent增删逐项对照；手写、生成/锁和文档分列，不补造历史估计 |
| D18 | 总体设计9.4为已完成 | T14（集成后） | 待获准集成：验收内容mainCI通过后另worktree写完成标记，再--no-ff合入并核最终mainCI，归档复验/正常清理；本分支不提前标完成 |

## 3. 二十二个故事与增长断言

每条原文的页面、数据库、异步、PAT 和逐 Phase 增长均在 `t1-requirements.json:stories`；T4 逐子项核断言位置及实际执行。I21 仅任务/数据库，I22 仅页面，I16 PAT 读取任务结果，I7 接口沿 I1，不虚构版本。

| ID | 故事 | 当前测试入口 | Owner/状态 |
|---|---|---|---|
| I1 | 创建工作项 | [i1-create-issue.spec.ts](../../../../e2e/stories/issue/i1-create-issue.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I2 | 在详情页改每个字段 | [i2-description.spec.ts](../../../../e2e/stories/issue/i2-description.spec.ts)、[i2-update-issue.spec.ts](../../../../e2e/stories/issue/i2-update-issue.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I3 | 列表与看板：分组、子分组 | [i3-list-groups.spec.ts](../../../../e2e/stories/issue/i3-list-groups.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I4 | 表格排序与日历 | [i4-list-order-calendar.spec.ts](../../../../e2e/stories/issue/i4-list-order-calendar.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I5 | 筛选与显示设置的保存 | [i5-list-preferences.spec.ts](../../../../e2e/stories/issue/i5-list-preferences.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I6 | 在看板上拖动（总体设计 8.2 的示例） | [i6-board-order.spec.ts](../../../../e2e/stories/issue/i6-board-order.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I7 | 快速添加 | [i7-quick-add.spec.ts](../../../../e2e/stories/issue/i7-quick-add.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I8 | 子工作项 | [i8-sub-issues.spec.ts](../../../../e2e/stories/issue/i8-sub-issues.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I9 | 关联 | [i9-relations.spec.ts](../../../../e2e/stories/issue/i9-relations.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I10 | 链接与元数据 | [i10-links.spec.ts](../../../../e2e/stories/issue/i10-links.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I11 | 评论与表情回应 | [i11-comments-reactions.spec.ts](../../../../e2e/stories/issue/i11-comments-reactions.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I12 | 操作动态的时间线 | [i12-activity.spec.ts](../../../../e2e/stories/issue/i12-activity.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I13 | 描述的历史与还原 | [i13-description-history.spec.ts](../../../../e2e/stories/issue/i13-description-history.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I14 | 草稿发布 | [i14-drafts.spec.ts](../../../../e2e/stories/issue/i14-drafts.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I15 | 归档、恢复与归档页 | [i15-archive-issue.spec.ts](../../../../e2e/stories/issue/i15-archive-issue.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I16 | 自动归档 | [i16-auto-archive.spec.ts](../../../../e2e/stories/issue/i16-auto-archive.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I17 | 删除（连同子树） | [i17-catalog.spec.ts](../../../../e2e/stories/issue/i17-catalog.spec.ts)、[i17-delete-issue.spec.ts](../../../../e2e/stories/issue/i17-delete-issue.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I18 | 搜索 | [i18-search.spec.ts](../../../../e2e/stories/issue/i18-search.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I19 | 个人主页的工作项 | [i19-member-issues.spec.ts](../../../../e2e/stories/issue/i19-member-issues.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I20 | 访客的边界（权限矩阵的页面抽样，9.2） | [i20-guest-boundary.spec.ts](../../../../e2e/stories/issue/i20-guest-boundary.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I21 | 物理清理 | [i21-purge.spec.ts](../../../../e2e/stories/issue/i21-purge.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |
| I22 | 工作项页上的会话切换 | [i22-session-switch.spec.ts](../../../../e2e/stories/issue/i22-session-switch.spec.ts) | T4源码/增长及默认E2E通过；[直接映射](closeout-web-contracts.md#1-二十二故事的实际观察)，最终C12已通过；main待集成 |

## 4. 二十六个交错

T3 逐项绑定具名测试、子例、真实锁/闸门观察及五次 race 执行；T9 补池的测量。当前只登记来源，不以名称命中判通过。

| ID | 原文合同 | Owner/证据 |
|---|---|---|
| L01 | 同一项目两次创建（编号连续、不重复；第二个等第一个提交）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L02 | 创建与删除项目（两种顺序：先建的随项目删除，与项目同一时刻，P9 起由任务写下；后建的 404）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L03 | 修改工作项与删除工作区（两种顺序；删除先拿到工作区 N 而工作项的写已持项目 N 时，删除等它提交，删除的时刻晚于它的 `updated_at`，任务写下的也是这个时刻）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L04 | 修改工作项与移出调用者（移出先提交：写在锁下判定得到 404）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L05 | 降为访客与修改工作项（降级先提交：403，或访客规则下 404）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L06 | 删除状态与创建、修改用它的工作项（状态先删：422 `state_id`；工作项先写：409 `project.state_in_use`）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L07 | 删除标签与给工作项加这个标签； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L08 | 归档项目与创建工作项（归档先提交：409 `project.archived`）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L09 | 两次改父工作项（两次移动；加子工作项与移动）各自合法、合起来成环（第二个在锁下沿祖先走，422 `parent_id` 或 `sub_issue_ids[i]`）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L10 | 两个方向相反的跨项目关联同时加（同一对只留一行，后到的 422 `duplicate`；两个项目按 `id` 升序锁，不死锁）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L11 | 删除带跨项目关联的工作项与删除对方项目（关联行按 `id` 升序锁，不死锁；关联行只有一次删除）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L12 | 归档与把它的状态改到"进行中"（先改状态：409 `issue.not_archivable`）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L13 | 删除父工作项与给它加子工作项（删除先提交：404；加先提交：新的子工作项一并被删除）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L14 | 发评论与删除工作项（删除先提交：404；评论先提交：评论被连带删除，同一时刻）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L15 | 同一个人连续两次改描述（动态合并为一行、版本合并为一行，时刻不减）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L16 | 发布草稿与删除它的项目（删除先提交：草稿已随项目删除，P9 起任务还没到时草稿的读也把它当作不存在，404 `issue.draft_not_found`）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L17 | 发布草稿与另一个请求把草稿改到别的项目（409 `issue.draft_changed`）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L18 | 自动归档与把工作项的状态改回"进行中"（改先提交：不归档；归档先提交：修改 409 `issue.archived`）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L19 | 抓取与改链接地址（抓取写 0 行，新的地址另有一次抓取）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L20 | 饥饿与连接池：经真实的连接池和写闸门，一串首尾相连的工作项写（同一工作区的若干项目）不让改工作区、改工作区成员的角色等到期限；删除工作区只等每个项目上正在进行和闸门放进来的写；200 个写压在同一个项目上时，另一个项目上无关的读的耗时不超过上限，多出的写得到 503 `server_busy`（S5、SP1 的回归测试）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L21 | 取锁的形状：每个工作项的写等第一把锁时在工作区行上什么都不持，它取的第一把锁是项目的 N，之后也不取与 `FOR NO KEY UPDATE` 冲突的工作区行的锁（插入经外键取工作区行的 `FOR KEY SHARE`，与它不冲突，S5 的 E3）；看不到项目的账户、看得到而不能写的账户（公开项目里不是成员的工作区成员、不是项目成员的工作区管理员）的写不取行锁、不写；加关联指向看不到的项目时不锁那个项目（不加锁的预检；这一例是自己的测试 `TestARelationLocksNoProjectItDoesNotReach`，与取锁形状的测试的 `createIssueRelations`、`deleteIssueRelation` 两行并列：取锁形状的测试按操作锁表，造不出"看得到工作项、看不到对方的项目"）。用 `pg_locks` 观察，与 M3 的同类测试同一写法； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L22 | 移出项目的默认负责人与不带负责人的创建（移出先提交：新的工作项没有负责人；创建先提交：他被指派，随后的移出不改这一行，与 Plane 相同）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L23 | 加关联 A→B 与删除 B 的项目（两种顺序：删除先提交：关联的写在锁下看到 B 的项目已删除，422，不写任何行；关联先提交：随项目删除这条关联行和反向的动态，与项目同一时刻（P9 起由任务写下），没有活的关联行指向已删除的工作项；S5 的 E5、E5b）； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L24 | 删除项目 P 的任务与另一个项目 Q 上删除关联另一端的工作项 Y（P9）：P 先删除、任务还没写到时删除 Y，Y 的删除不写这条关联行，P 的任务随后写下 P 的时刻和删除者；Y 先删除（P 还在）时这一行是 Y 的时刻和删除者，P 的任务跳过它；任务与 Y 的删除同时锁关联行时按 `id` 先锁，两种顺序都不死锁；断言每一行的 `deleted_at`、`updated_by_id`，不只是"只删了一次"。任务挂起期间，从另一个项目对已删除项目里的工作项加关联 422、删除指向它的关联 404 `issue.relation_not_found`； | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L25 | 删除项目 Old，它的任务还没做完时删除它的工作区（P9，聚焦复核 I1）：Old 的每一行工作项一侧的行（含 Old 与工作区另一个项目之间的关联行）都是 Old 的时刻和删除者，工作区其余项目的行都是工作区的；没有项目的草稿是工作区的。M3 的 W3（先删 Old、再删工作区，断言 Old 的行留着 Old 的删除）在 P9 的轮询之下因此确定地通过。 | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |
| L26 | 删除关联与删除它另一端的工作项（P3 加，编号接在最后：20–25 在各文档里被引用）：删除关联不论在哪一端判定都锁两端的项目，与删除工作项在那个工作项的项目行上排队；关联先删，工作项的删除读到它已删除、跳过，留着那次删除；工作项先删，关联的删除在锁下重读两端，读到那个工作项已删除（关联已随它删除），404 `issue.relation_not_found`，什么都不写。 | T3 各5次通过，见[具名映射](closeout-server-contracts.md#3-二十六交错逐项)；L20规模记录另T9 |

## 5. 同步与差异逐行索引

以下 stable ID 绑定 T1 保存的完整原文；不是删除或改写设计行。T11 必须读实际目标完整段并给事实证据。

| ID | 原文键/主题 | 目标/责任 | 状态 |
|---|---|---|---|
| SYNC01 | P1a | 总体设计 / 4.2 / 加锁顺序延伸到 M4 的表；工作项的写最先锁项目行、不锁工作区行，取锁之前做不加锁的预检（凡拒绝的都在那里答）；删除工作区先锁它的项目再读时刻；写明 B 的两个代价（3.5，决策点 1） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC02 | P1b | 总体设计 / 4.2 / 取连接之前进入按项目的写闸门、`lock_timeout` 的兜底，满时 503 `server_busy`；写明连接池的代价（3.5，架构问题 11） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC03 | P1b | 总体设计 / 6.2 / 平台包 `platform/gate`（带 `context` 的按键的信号量，3.5，架构问题 11） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC04 | P1a | 总体设计 / 6.2 / 模块 `issue` 和它的 13 张表（3.3，架构问题 1） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC05 | P1a | M3 设计 / 3.6 的取锁表、17.4 / "删除工作区"一行改为先锁项目、再读时刻；17.4"M4 的工作项写入……不默认沿用"后补一句 M4 的选择（3.5） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC06 | P1a | M3 设计 / 3.17 / 删除状态：有工作项（含已归档的）用它时 409 `project.state_in_use`（3.6） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC07 | P1a | 差异清单 / 二·按表 / `issues`、`issue_assignees`、`issue_labels`、`issue_subscribers`、`issue_activities` 逐列（4.2–4.5、4.11） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC08 | P1a | 差异清单 / 四 / 4.17 中标 P1a 的行 | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC09 | P1b | 差异清单 / 四 / 4.17 中标 P1b 的行 | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC10 | P2 | 总体设计 / 6.5 / `AllowCreator` 要求创建者仍是项目的有效成员、角色不低于成员；`OwnOnly` 与"一条访客规则"（3.4，架构问题 3） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC11 | P2 | M2 设计 / 3.11 / 格式检查器登记 `date`；字段错误的路径按数组下标的数值排序；不限类型的节点检查数的范围（6.1，M2 交接第 4 节） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC12 | P2 | 差异清单 / 四 / 4.17 中标 P2 的行（访客规则、创建者、归档只读、按编号取的 404 和折叠、起止日期） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC13 | P3 | 总体设计 / 3.4 / 集合型的列表加上子工作项、关联、链接、工作项的表情（3.16 末尾）。P3 有第一个（`listSubIssues`） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC14 | P3 | 总体设计 / 3.6 / "关联对象只返回 ID"的第二个例外：关联内嵌对方工作项的列表行（3.10） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC15 | P3 | 总体设计 / 4.2 / 约定七（跨项目的写按 `id` 升序锁住涉及的项目）和关联行的约定五例外（3.5） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC16 | P3 | 差异清单 / 二·按表、四 / `issue_relations` 逐列（4.6）；4.17 中标 P3 的行 | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC17 | P4 | 总体设计 / 1.1 / "评论（支持回复）"改为"评论"（决策点 6）；"历史版本：工作项快照、描述修改历史"改为"描述修改历史（可查看、可还原）"（决策点 5） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC18 | P4 | 总体设计 / 5.2、5.3、6.6、6.7 / 表数 45 改为 44，删去 `issue_versions`；6.6 删去"同时写 `issue_versions`"；6.7 的保留任务写明"每个工作项最新的 20 个描述版本"（3.14） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC19 | P4 | M2 设计 / 3.12 / 页大小的规则移到 `shared`（M3 交接第 9 节） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC20 | P4 | 差异清单 / 一、二·按表、四 / 一：`issue_versions` 不保留；二：`issue_comments`、`issue_reactions`、`comment_reactions`、`issue_mentions`、`issue_description_versions` 逐列（4.7、4.9、4.10、4.12、4.14）；四：4.17 中标 P4 的行 | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC21 | P5 | 总体设计 / 3.3 / 筛选的写法补上 `none`（含日期的 `none`）、可重复的日期参数、单日、不合格的取值 422（3.16）；已在P5/T16同步，实际依据见 [P5 review §2](P5-issue-list-review.md#2-验收标准逐条核对) | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC22 | P5 | 总体设计 / 3.4 / 不分组的列表带 `total`；子分组的结构 `subgroups` 和泳道的总数 `subgroup_totals`；只返回非空的组；分组时不带 `group_key` 的游标 422；列表的读在一个 `REPEATABLE READ` 的快照里；一组（一格）的"加载更多"是同一个请求加 `group_key`（`sub_group_key`），不再是"加上这一组的筛选"；游标的载荷写明排序、以 `id` 结尾；未知的 `order_by`、`group_by` 422（3.16，架构问题 6）；已在P5/T16同步，实际依据见 [P5 review §2](P5-issue-list-review.md#2-验收标准逐条核对) | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC23 | P5 | 总体设计 / 6.2 / 列表的词汇在 `issue/domain`，`project` 经它声明、`issue.Provide` 实现的 `IssueListSettings` 检查显示设置（3.17，架构问题 10）；已在P5/T16同步，实际依据见 [P5 review §2](P5-issue-list-review.md#2-验收标准逐条核对) | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC24 | P5 | M3 设计 / 3.18 / 项目的显示设置加 `filters`、`display_filters`、`display_properties` 三列，格式见本文 3.17；已在P5/T16同步，实际依据见 [P5 review §2](P5-issue-list-review.md#2-验收标准逐条核对) | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC25 | P5 | 差异清单 / 二·按表、三、四 / `project_user_properties` 加三列、不建 `rich_filters`（4.15）；三：列表的响应、排序键、分组键的名字；四：4.17 中标 P5 的行；已在P5/T16同步，实际依据见 [P5 review §2](P5-issue-list-review.md#2-验收标准逐条核对) | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC26 | P6 | 总体设计 / 6.2、6.3 / `shared` 加事件的值和 `EventPublisher`；6.3 第 3 条写明订阅者的写法：同步、在事务里、出错回滚，异步的事经 `InsertTx`（3.19，决策点 3） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC27 | P6 | 总体设计 / 6.7 / 定时任务的清单和间隔、两个队列；测试的时钟偏移（3.20，架构问题 7） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC28 | P6 | M2 设计 / 3.15、3.17 / 事务内投递经只投递的 River 客户端（在模块之前建出）；队列与并发；定时任务和模块声明会投递的种类没有 worker 时构造失败（3.20） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC29 | P6 | 差异清单 / 四 / 自动归档的执行者为空（决策点 4）；按间隔而不是固定时刻运行 | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC30 | P7 | 总体设计 / 6.2、8（安全） / `platform/outbound`：防 SSRF 的出站客户端，M8 复用（3.11、8.1，架构问题 8） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC31 | P7 | 差异清单 / 二·按表、四 / `issue_links` 逐列（4.8）；抓取的上限、`metadata` 的形状 | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC32 | P7 | README / "安全"一节 / 8.8 中 P7 的行；P6、P9 的"部署"行同样按 8.8 | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC33 | P8 | 总体设计 / 5.4 / 草稿：`project_id` 是一列；发布用存着的载荷、要求 `issue.create`；409 `issue.draft_changed`（3.15） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC34 | P8 | 总体设计 / 6.9 / `description_stripped` 不用于搜索；搜索的匹配和顺序、选择器的排除、`q` 为空时给出最近更新的、日期的条件、编号的折叠（3.7、3.18） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC35 | P8 | 差异清单 / 二·按表、四 / `draft_issues` 逐列（4.13）；`issues` 的 trigram 索引落地；4.17 中标 P8 的行 | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC36 | P8 | README / "部署"一节 / 迁移执行 `CREATE EXTENSION pg_trgm`（库属主即可，8.8） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC37 | P9 | 总体设计 / 5.5、6.7 / 60 天清理：每个模块清理自己的表，一个任务按顺序运行；删除关系图（3.21、4.16，架构问题 9）；描述版本保留只碰活版本、清理处理墓碑；迟到软删与父CASCADE的交叠例外见3.6 **P9回执：** 已同步总体§5.5/6.7：23表单任务清理、活版本保留、严格cutoff/父CASCADE与迟到40P01边界；T1–6/T12–15证据见P9 review | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC38 | P9 | 总体设计 / 5.5、6.3 / "连带删除在同一个事务里同步完成"改为：删除项目、工作区时工作项一侧的行由事件的订阅者投递的任务分批软删除，写下请求的时刻和删除者；第一个跨模块的异步副作用（`project.deleted`、`workspace.deleted`）；v0 的规模目标（一个项目至多约 2 万个未删除的工作项）（3.6、3.19，决策点 7） **P9回执：** 已同步总体§5.5/6.3：M3同步、issue同事务投递后分批软删/立即不可见/原AtBy；T9–13与约2万issue每项目实测见P9 spec附录B | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC39 | P9 | M3 设计 / 3.6、13.2 / 删除项目、工作区的步骤后面是发布事件，不再调 `IssueCascade` 的删除；13.2 设想的"删除工作区的连带挂到事件上"对工作项一侧做了（3.19） **P9回执：** 已同步M3§3.3/3.6/6.2/6.5/13.2：只移除issue两同步删除能力，邀请/M3同步与StateInUse/DetachLabel保留；T9/T10回执见P9 review | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC40 | P9 | 差异清单 / 四 / 动态随工作项物理删除（`CASCADE`，不是 `DO_NOTHING`）；已登记的"连带在同一事务"改为只对删除工作项、状态、标签成立，删除项目、工作区的工作项一侧与 Plane 一样在事务之外完成，但写下请求的时刻（3.6） **P9回执：** 已同步plane-diff第四节及版本索引行：活动/评论子链CASCADE、同步/异步按操作划分；T3/T5/T9–13证据见P9 review | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC41 | P10b | 总体设计 / 7.7 / `Reconciled.fetchPart` 和 `paged-groups.ts`；一个工作项只有一份，取服务端 `updated_at` 大的（3.22） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC42 | P10a–P16 | 前端改动清单 / 3.1、3.2 / M4 一行的状态；删到 M5、M6、M7 的入口和取数（3.1、3.2）；迭代、模块、项目视图、工作区视图的工作项列表在 P10a 删除，M6、M7 加回（7.3） | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC43 | P12 | README / "前端"一节 / 已按实际更新四布局/原生搜索及后续页面边界；最终e6d1329b与main ff89d0a1准确CI均通过、归档清理完成，见P12 review§28 | T11已核，见[逐行证据](closeout-sync.md) |
| SYNC44 | 收尾 | 总体设计 / 9.4 / M4 的状态改为"已完成" | 集成待办（D18） |
| DIFF01 | 工作项的写的取锁 | 差异清单第四节；P1a | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF02 | 一个项目上的并发写 | 差异清单第四节；P1b | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF03 | 删除项目、工作区时工作项一侧的行 | 差异清单第四节；P9 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF04 | 负责人、标签不合格 | 差异清单第四节；P1a | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF05 | 起止日期 | 差异清单第四节；P2 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF06 | 默认负责人 | 差异清单第四节；P1a | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF07 | 负责人的动态 | 差异清单第四节；P1a | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF08 | 只补块 id 的保存 | 差异清单第四节；P4 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF09 | 描述的动态的合并 | 差异清单第四节；P4 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF10 | 自动关注 | 差异清单第四节；P1a（评论、提及 P4） | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF11 | 编号 | 差异清单第四节；P1a | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF12 | 删除状态 | 差异清单第四节；P1a | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF13 | 修改的响应 | 差异清单第四节；P1b | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF14 | 负责人的读取 | 差异清单第四节；P1b | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF15 | 创建者的权限 | 差异清单第四节；P2 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF16 | 访客只看自己的工作项 | 差异清单第四节；P2（各列表在各自的 Phase） | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF17 | 访客删除自己建的工作项 | 差异清单第四节；P2 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF18 | 已归档的工作项 | 差异清单第四节；P2（子资源在各自的 Phase） | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF19 | 归档、恢复 | 差异清单第四节；P2 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF20 | 按编号取 | 差异清单第四节；P2 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF21 | 删除工作项 | 差异清单第四节；P3（确认框 P14） | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF22 | 父工作项 | 差异清单第四节；P3 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF23 | 加子工作项的动态 | 差异清单第四节；P3 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF24 | 子工作项数 | 差异清单第四节；P3 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF25 | 关联 | 差异清单第四节；P3 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF26 | 关联的动态 | 差异清单第四节；P3 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF27 | 空评论 | 差异清单第四节；P4 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF28 | 改、删别人的评论 | 差异清单第四节；P4 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF29 | 表情的值 | 差异清单第四节；P4 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF30 | 评论、表情的动态 | 差异清单第四节；P4 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF31 | 提及的同步 | 差异清单第四节；P4 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF32 | 描述的历史 | 差异清单第四节；P4 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF33 | 动态的列表 | 差异清单第四节；P4 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF34 | 时间线的增量读取 | 差异清单第四节；P4 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF35 | 列表的分组、排序 | 差异清单第四节；P5 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF36 | 已归档的项目的工作项列表 | 差异清单第四节；P5 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF37 | 个人主页的工作项 | 差异清单第四节；P5 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF38 | 显示设置 | 差异清单第四节；P5 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF39 | 自动归档的执行者 | 差异清单第四节；P6 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF40 | 定时任务的时刻 | 差异清单第四节；P6 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF41 | 链接的地址 | 差异清单第四节；P7 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF42 | 链接的抓取 | 差异清单第四节；P7 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF43 | 草稿 | 差异清单第四节；P8 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF44 | 搜索 | 差异清单第四节；P8 | T11已核，见[逐行证据](closeout-sync.md) |
| DIFF45 | 物理清理 | 差异清单第四节；P9 | T11已核，见[逐行证据](closeout-sync.md) |

## 6. C1–C12 与最终浏览器记录

| 编号 | 原阶段 | 当前验收位置 |
|---|---|---|
| C1 | P12 | T5–T7历史观察＋T13最终候选补验通过；[逐格及视觉边界](closeout-browser-matrix.md)，main绑定待T14 |
| C2 | P13 | T5–T7历史观察＋T13最终候选补验通过；[逐格及视觉边界](closeout-browser-matrix.md)，main绑定待T14 |
| C3 | P12、P14 | T5–T7历史观察＋T13最终候选补验通过；[逐格及视觉边界](closeout-browser-matrix.md)，main绑定待T14 |
| C4 | P14 | T5–T7历史观察＋T13最终候选补验通过；[逐格及视觉边界](closeout-browser-matrix.md)，main绑定待T14 |
| C5 | P14 | T5–T7历史观察＋T13最终候选补验通过；[逐格及视觉边界](closeout-browser-matrix.md)，main绑定待T14 |
| C6 | P15 | T5–T7历史观察＋T13最终候选补验通过；[逐格及视觉边界](closeout-browser-matrix.md)，main绑定待T14 |
| C7 | P15 | T5–T7历史观察＋T13最终候选补验通过；[逐格及视觉边界](closeout-browser-matrix.md)，main绑定待T14 |
| C8 | P15 | T5–T7历史观察＋T13最终候选补验通过；[逐格及视觉边界](closeout-browser-matrix.md)，main绑定待T14 |
| C9 | P13、P15 | T5–T7历史观察＋T13最终候选补验通过；[逐格及视觉边界](closeout-browser-matrix.md)，main绑定待T14 |
| C10 | P14 | T5–T7历史观察＋T13最终候选补验通过；[逐格及视觉边界](closeout-browser-matrix.md)，main绑定待T14 |
| C11 | P14 | T5–T7历史观察＋T13最终候选补验通过；[逐格及视觉边界](closeout-browser-matrix.md)，main绑定待T14 |
| C12 | 收尾 | T5–T7历史观察＋T13最终候选补验通过；[逐格及视觉边界](closeout-browser-matrix.md)，main绑定待T14 |

## 7. 阶段遗留与输入交接

18 个 Phase 的 spec §5 与 review §6 各一份，36 份来源均读取并逐 hash 固定。P16 dead-members §6 作为附加支持材料，不冒充第十九个 Phase。T12 逐原条款列已完成证据或具名后续归属，不能只改 status。

| Phase | spec §5 / review §6 | 后续归属 |
|---|---|---|
| P10a | [spec](../specs/P10a-web-issue-types.md) / [review](P10a-web-issue-types-review.md) | T12全部原条款已核，见[交接索引](closeout-handoffs.md) |
| P10b | [spec](../specs/P10b-web-list-data.md) / [review](P10b-web-list-data-review.md) | T12全部原条款已核，见[交接索引](closeout-handoffs.md) |
| P11 | [spec](../specs/P11-web-detail-data.md) / [review](P11-web-detail-data-review.md) | T12全部原条款已核，见[交接索引](closeout-handoffs.md) |
| P12 | [spec](../specs/P12-web-list-pages.md) / [review](P12-web-list-pages-review.md) | T12全部原条款已核，见[交接索引](closeout-handoffs.md) |
| P13 | [spec](../specs/P13-web-filters-archives-profile.md) / [review](P13-web-filters-archives-profile-review.md) | T12全部原条款已核，见[交接索引](closeout-handoffs.md) |
| P14 | [spec](../specs/P14-web-issue-detail.md) / [review](P14-web-issue-detail-review.md) | T12全部原条款已核，见[交接索引](closeout-handoffs.md) |
| P15 | [spec](../specs/P15-web-collaboration-drafts-search.md) / [review](P15-web-collaboration-drafts-search-review.md) | T12全部原条款已核，见[交接索引](closeout-handoffs.md) |
| P16 | [spec](../specs/P16-web-cleanup.md) / [review](P16-web-cleanup-review.md) | T12全部原条款已核，见[交接索引](closeout-handoffs.md) |
| P1a | [spec](../specs/P1a-issues.md) / [review](P1a-issues-review.md) | T12全部原条款已核，见[交接索引](closeout-handoffs.md) |
| P1b | [spec](../specs/P1b-issue-update.md) / [review](P1b-issue-update-review.md) | T12全部原条款已核，见[交接索引](closeout-handoffs.md) |
| P2 | [spec](../specs/P2-issue-access.md) / [review](P2-issue-access-review.md) | T12全部原条款已核，见[交接索引](closeout-handoffs.md) |
| P3 | [spec](../specs/P3-issue-hierarchy.md) / [review](P3-issue-hierarchy-review.md) | T12全部原条款已核，见[交接索引](closeout-handoffs.md) |
| P4 | [spec](../specs/P4-issue-collaboration.md) / [review](P4-issue-collaboration-review.md) | T12全部原条款已核，见[交接索引](closeout-handoffs.md) |
| P5 | [spec](../specs/P5-issue-list.md) / [review](P5-issue-list-review.md) | T12全部原条款已核，见[交接索引](closeout-handoffs.md) |
| P6 | [spec](../specs/P6-issue-events.md) / [review](P6-issue-events-review.md) | T12全部原条款已核，见[交接索引](closeout-handoffs.md) |
| P7 | [spec](../specs/P7-issue-links.md) / [review](P7-issue-links-review.md) | T12全部原条款已核，见[交接索引](closeout-handoffs.md) |
| P8 | [spec](../specs/P8-drafts-search.md) / [review](P8-drafts-search-review.md) | T12全部原条款已核，见[交接索引](closeout-handoffs.md) |
| P9 | [spec](../specs/P9-purge.md) / [review](P9-purge-review.md) | T12全部原条款已核，见[交接索引](closeout-handoffs.md) |

| 输入 | 关闭条件 |
|---|---|
| [M0-P4-pg-trgm](../handoffs/M0-P4-pg-trgm.md) | T12 全部条款完成或具名转交，现closed；见[逐条结论](closeout-handoffs.md) |
| [M1-P2-trim-content](../handoffs/M1-P2-trim-content.md) | T12 全部条款完成或具名转交，现closed；见[逐条结论](closeout-handoffs.md) |
| [M1-P3-trim-platform](../handoffs/M1-P3-trim-platform.md) | T12 全部条款完成或具名转交，现closed；见[逐条结论](closeout-handoffs.md) |
| [M1-P4-router-native](../handoffs/M1-P4-router-native.md) | T12 全部条款完成或具名转交，现closed；见[逐条结论](closeout-handoffs.md) |
| [M1-P5-brand](../handoffs/M1-P5-brand.md) | T12 全部条款完成或具名转交，现closed；见[逐条结论](closeout-handoffs.md) |
| [M1-closeout](../handoffs/M1-closeout.md) | T12 全部条款完成或具名转交，现closed；见[逐条结论](closeout-handoffs.md) |
| [M2-closeout](../handoffs/M2-closeout.md) | T12 全部条款完成或具名转交，现closed；见[逐条结论](closeout-handoffs.md) |
| [M3-closeout](../handoffs/M3-closeout.md) | T12 全部条款完成或具名转交，现closed；见[逐条结论](closeout-handoffs.md) |

## 8. T1 核对记录

- 基线/完整文件清单、22/26/44/45/12/18 源条数已固定，增长原文无删减；当前条目均待后续验证。
- phase 改为 `M4/closeout`；规则与例外的逐字段内容保持不变。`t1-lint-web` exit 0：原 76 规则/13 例外的 samples 与到期检查通过、无命中；54/54 任务成功（37 命中本地共享缓存），1294 翻译键有定义，输入未变。
- P16 已归档清理，本阶段不重复清理；原始归档收据位于主目录 `.superpowers/sdd/P16-integration-20261101/`。

## 9. T2 历史锁断言收口

15 个测试文件去掉重复错误分类/探针，生产代码没有变更。原 13 文件的 16 处 SQLSTATE 比较逐处记录于 `t2-lock-disposition.json`，全文读取的输入见 `source-read-ledger.json`。

| 原场景 | 处理及保持的性质 |
|---|---|
| identity 的账户/凭据、workspace 的目录/成员/邀请、project 的默认状态 | 复用 `postgres.LockNotAvailable`；真实 200/300/500ms 锁超时及副作用断言保留 |
| identity `rowLockedByAnother`、bootstrap `webFree` | 复用 `pgtest.RowWaits` 的 NOWAIT 持锁探针；缺行和其他错误直接失败 |
| workspace `waitsFor` 及其三个消费者 | 原断言只判断锁兼容性，改用 `RowWaits`；删除无读者的旧帮手。实际排队、排序和重读仍用独立连接/锁等待观察证明 |
| bootstrap `heldBy`、demotion 的联表探针 | 保留带多条件/联表的原 SQL、单行要求及失败分支，只复用错误分类 |
| 残留 `55P03` | 仅生产分类常量、分类/503 专测的输入、注释及诊断文案；不做字面量机械清零。`40P01` 的独立拒绝断言保留 |

实际运行（所有收据和完整日志均在实施证据根）：

- `t2-before`：原 63 个顶层测试全部先通过，含子测试 163 项；39.12s。
- `t2-after`：计划指定的 postgres、identity、workspace、project、bootstrap 完整包范围，1177 个顶层、含子测试 6305 项通过；377.11s。只有 2 个显式 opt-in 规模测试跳过，另 8 个生成/无测试包无用例；规模结果留 T8/T9 单独验。
- `t2-mutant-remove-lock`：仅临时移除 `LockMemberWorkspaces` 的 `FOR NO KEY UPDATE`，`TestLockMemberWorkspaces` 明确因 acme/gamma 的 `FOR SHARE` 不再阻塞而失败。恢复收据记录前后相同 SHA256 `6489b3444692470325a448a7b6f82fbf0bad57ca4dc4ff58f44c6bcd51f5162e`；不提交变异。
- `t2-race`：同一受测输入下原 63 项各跑 5 次，315 个顶层执行、含子测试 815 项通过；132.04s，无 race/意外 40P01。`t2-race-summary.json` 逐测试计数为 5，和全包输入相等。

T2 四步完成。这些证据覆盖锁断言整理；D02 的全 26 交错和真实池验收仍分别由 T3/T9 补齐。

## 10. T3 服务端性质复核

[服务端直接证据](closeout-server-contracts.md)列完整26项具名测试、D02–D07具体观察、全部skip去向和18个变异候选。专项506顶层/3074含子例通过；交错215顶层/2950含子例通过。两种漏worker变异均业务断言失败并恢复；T3未发现需修改的生产缺陷。

## 11. T4 会话与路由的局部修复（完成）

- 复制 hook 在剪贴板异步成功/拒绝后都可能给新账户弹旧提示。`t4-copy-session-red` 新增12例全部业务失败、原5例通过；统一复用 `sessionGuard` 后 `t4-copy-session-green` 4文件67例通过，含创建提示、评论、列表和7种详情入口的真实消费者。没有新建权限代际 owner；浏览器已经执行的剪贴板写入不追溯撤销，只禁止旧操作后续提示污染当前会话。
- 归档动作 hook 的三个回调遗漏 `workspaceSlug` 依赖。同一挂载切换路由的真实 store 测试分别观察到：恢复收到成功响应却被判 `SessionChangedError`、删除后列表 total 仍为1、当前工作区折叠值仍为空。`t4-archive-route-red` 3例业务失败；修正依赖并对齐续页/个人动作 memo 的依赖后 `t4-archive-route-green-2` 7文件123例通过。该测试证明 hook 复用时的缺口，不声称所有页面跳转都会复用该挂载。
- 首次归档复验因测试快照直接 structuredClone MobX 嵌套代理报 DataCloneError，改为比较原/现 JSON 值后通过；原失败日志保留，不计有效变异。6个修改源码/测试文件 oxlint 零告警/错误。
- [Web合同记录](closeout-web-contracts.md)已完成22条增长、异步hook/调用方、S1/S2四角色和13实例核对。本地七门禁通过，默认E2E284项全过；类型/测试环境失败及根因修正均保留。准确检查点 `7f58273592ec0e9d488b2d87501a40b9c83b1f57` 的 [CI 41300361270](https://github.com/open-nerve/NerveProject/actions/runs/41300361270) attempt 1 三作业全绿；收据 `t4-ci-receipt.json`。C12/最终main仍未签结。


## 12. T5 列表与筛选验收（完成）

三引擎原选择各125项中，本组准确覆盖46项，含S2四身份；逐id结果与输入等价证明见 `t5-coverage.json`。最新有效并集每引擎46项全部通过，无最终未解失败/skip/flaky；保留 Firefox 中途45通过/1失败及修正后的单项补验，未将它改写为一次全绿。

C12 图像证据见[浏览器矩阵](closeout-browser-matrix.md#t5-列表筛选与局部修复)。稳定截图201张及真实Tab补验27张均经控制者看图，原图、JSON、trace、失败尝试、26张比较索引和逐图观察 `t5-visual-review.json` 保留。B06的复制/编辑器加载、B11及B21后续细分仍由T6/T7完成。

实际产品缺陷：peek 子项显示面板底部741.375px超过720px，筛选面板也越界。具名 `t5-peek-clipping-red` 先业务失败；在既有 FiltersDropdown 内按 Popper 裁切边界限制内容高度，并跟踪面板大小变化，保留原滚动内容和锚点。重建后720/500两高度、真实Tab末项/返回焦点在三引擎通过；单位回归7项通过。未改权限/接口/owner/依赖。

截图首次过早读取过渡开始帧，记录帮手改为等实际有限动画结束。Firefox偏好拒绝的错误toast挡住下一按钮鼠标中心，改用焦点+真实Space复核下一次写入，拒绝/滚动/DB断言不变。原生轮廓允许浏览器 auto 或 solid，不能用只接受 solid 的测试误报产品失败。全套Web lint54/54及本组类型/lint通过，最终F另由T9/T13收口。

## 13. T6 详情与编辑器（完成）

原38项/引擎均已通过（`t6-chromium/firefox/webkit`），198张成功截图已由控制者检查。覆盖字段拒绝/慢写、父子/关联/链接、删除、权限与会话、描述及表格行列菜单；最终补验见下。

补真实Tab时发现链接卡片复制按钮的 `outline-none` 令焦点不可见：`t6-keyboard-chromium` 的链接复制例先业务失败，改用既有 `focus-visible` 轮廓；`t6-keyboard-4-chromium/firefox` 各8项通过。菜单须从真实Tab触发器进入，程序focus加方向键在Firefox不足以证明键盘轮廓；表格相邻内容区会消费Tab导航单元格，补验从相邻控件往返，没有放宽断言或修改编辑器。

WebKit补验到4项通过时Docker socket消失，登录/DB连接继而失败；只读 `docker info` 也无法连接。`t6-docker-interruption.json` 保留中断原因，当时等待负责人恢复服务，不能把中途结果当全绿。新增中文编辑器具名行/列菜单及继续输入用例，类型/lint/格式已通过，中断时浏览器尚待验。`t6-final-lint-3` 全部54项通过；此前新增测试导入与await-in-loop问题已修正，未放宽规则。

负责人恢复Docker后，WebKit八个原生键盘补验全部通过；新增中文例的测试定位先后修正后，`t6-chinese-2-{chromium,firefox,webkit}` 各1通过。中文行/列菜单及继续输入的实际名称、Tab轮廓、菜单位置和回焦均已查看。原38+新增1项/引擎的最终有效并集为39，262份观察/30张比较索引已检查；包含的中途失败部分图不算整例通过。[矩阵T6记录](closeout-browser-matrix.md#t6-详情与编辑器验收)、`t6-coverage.json` 和 `t6-visual-review.json` 保留完整来源。T6四步完成，B13命令入口及协作六组留T7，未签结整体C12。

## 14. T7 协作、搜索与真实 LAN

T7三引擎各53唯一项加S2四身份补验通过；合并T5/T6后138项/引擎、原375项全覆盖。`t7-coverage.json`保留每个file/title/id/project的最终成功来源、失败尝试收据、产品等价与72项真实LAN原生返回值。完整三引擎矩阵见[浏览器台账](closeout-browser-matrix.md#t7-协作搜索与-lan-验收)。

本任务只补E2E观察与有手势LAN路径，无生产修改。Chromium初次toast文本重复、命令末尾句号的断言问题已修正；Firefox警告核对真实eval调用位置及完整文案，未忽略意外console。240图由控制者在27张比较索引及3张额外原图逐张核查，含I16系统归档/成员恢复的六张原附件。源码输入/日志hash与全部原始trace留存。D12为候选验收完成，最终输入等价、mainCI及集成签结不提前标完成。


## 15. D09 批准决定与最终实现

八决策和十一架构项均按已批准选项逐项核对；下表只证明现有合同已落实，不重开架构批准。实际执行与输入绑定分别见服务端合同、C12、七门禁和`t13-evidence-binding.json`。

| 批准项 | 最终实现与保留边界 | 源码 / 回归入口 | 本次证据 |
|---|---|---|---|
| 决策1 B | 普通工作项预检→gate→项目N；无工作区S；草稿例外及审计时刻代价保留 | [实现](../../../../server/internal/modules/issue/app/lock.go)、[回归](../../../../server/internal/bootstrap/issue_write_locks_test.go) | L03/04/05/20/21、T3五遍race；T9真实10连接/K2 |
| 决策2 A | 一个M4、18个Phase加收尾，未提前分拆完成标记 | [设计§15](../M4-design.md#15-phase-进度表) | 本页D17逐first-parent集成范围与M4§15；本轮集成待办 |
| 决策3 A | 同步事务内发布，订阅者经InsertTx投递；失败随事务回滚 | [实现](../../../../server/internal/shared/events.go)、[回归](../../../../server/internal/bootstrap/events_test.go) | T3实际river_job提交/回滚及T10 R03真实订阅者观察 |
| 决策4 A | 自动归档ActorID空表示Nerve，不造系统账户 | [实现](../../../../server/internal/modules/issue/domain/events.go)、[回归](../../../../server/internal/bootstrap/issue_lifecycle_events_test.go) | I16 API/PAT读取及三引擎真实系统动态截图 |
| 决策5 A | 只有描述历史，无全工作项issue_versions快照 | [实现](../../../../server/internal/modules/issue/app/description_versions.go) | I13历史还原；44同步/45差异对应原文已核 |
| 决策6 A | 评论平铺，无回复parent_id | [实现](../../../../server/internal/modules/issue/app/comments.go)、[回归](../../../../server/internal/bootstrap/issue_comments_test.go) | I11评论/反应；契约与数据库无回复字段 |
| 决策7 A | 项目/工作区删除立即不可见；事件投递键集分批软删，原At/By及45秒预算 | [实现](../../../../server/internal/modules/issue/app/delete_rows.go)、[回归](../../../../server/internal/modules/issue/adapter/river/delete_rows_test.go) | T9真实20k项目、五项目工作区、七job单次完成；R02时长溢出已修 |
| 决策8 B | 管理员可删他人评论不能改；updated_by/ActorID留痕，不加删除动态 | [实现](../../../../server/internal/modules/issue/app/comment_writes.go)、[回归](../../../../server/internal/bootstrap/issue_comment_events_test.go) | 权限矩阵、I11/I12页面管理员他人菜单、T3公开码 |
| 架构11.1 | issue拥有13表；app声明端口、adapter清洗，domain保持纯净 | [实现](../../../../server/internal/modules/issue/module.go)、[回归](../../../../server/internal/archtest/composition_test.go) | Provide→组合根→New及purity/deps/SQLC守卫；R06只改变已有ProjectStore事实选择 |
| 架构11.2 | 跨项目gate/行锁去重升序；关联与草稿保持专门编排 | [实现](../../../../server/internal/modules/issue/app/lock_across.go)、[回归](../../../../server/internal/bootstrap/issue_relation_locks_test.go) | L01–26全部具名映射；后台/版本保留/迟到重试边界单列 |
| 架构11.3 | Grant.OwnOnly与HiddenFrom、SQL可见集合一致 | [实现](../../../../server/internal/modules/issue/app/lock.go)、[回归](../../../../server/internal/modules/issue/app/guest_rule_test.go) | 全矩阵、访客逐body一致；R06真实目录缺席+晚答页面回归 |
| 架构11.4 | 事务内事件与River插入分层，工作客户端最后登记；缺worker启动失败 | [实现](../../../../server/internal/platform/jobs/inserter.go)、[回归](../../../../server/internal/platform/jobs/registration_test.go) | T3两种有效漏登记变异、提交/回滚及clock边界 |
| 架构11.5 | 目录经typed数组传入静态SQL，无跨模块读表例外 | [实现](../../../../server/internal/modules/issue/domain/listvocab.go)、[回归](../../../../server/internal/archtest/sqlc_test.go) | TestSQLCSchemaScope；T8三规模39场景，cells物化代价明确 |
| 架构11.6 | none/日期AND/组格续页/21合法排序码和精确游标；两端同全序 | [实现](../../../../server/internal/modules/issue/domain/listvocab.go)、[回归](../../../../server/internal/bootstrap/issue_list_pages_test.go) | 共享issue-order.json、九反例、Go/vitest与I3/I4/I5 |
| 架构11.7 | 测试offset只影响业务时钟；新库每刻仅一个nerve进程 | [实现](../../../../server/internal/bootstrap/clock.go) | e2e/fixtures/clock.ts和smoke/clock-lifecycle.spec.ts；I16/I21 |
| 架构11.8 | outbound独立包，所有DNS候选/每跳重检/固定IP且不用代理 | [实现](../../../../server/internal/platform/outbound/client.go)、[回归](../../../../server/internal/platform/outbound/transport_test.go) | IANA首尾/本机地址、零危险dial、I10 blocked与真实metadata |
| 架构11.9 | 模块清理issue→project→workspace→identity；CASCADE/SET NULL目录守卫 | [实现](../../../../server/internal/bootstrap/purge.go)、[回归](../../../../server/internal/bootstrap/purge_runtime_test.go) | T3目录与L24/25，T9真实runtime一次660026行；账户不清理 |
| 架构11.10 | 列表词汇归issue/domain，project经IssueListSettings端口验证 | [实现](../../../../server/internal/modules/issue/domain/listvocab.go)、[回归](../../../../server/internal/modules/issue/app/list_settings_test.go) | 显示三列/合法形状、I5真实保存，未引入第二份词汇 |
| 架构11.11 | 预检后取连接前进入按项目gate；K2/queue64、期限/503集中映射 | [实现](../../../../server/internal/bootstrap/gate.go)、[回归](../../../../server/internal/bootstrap/gate_test.go) | T9 200并发、无关读78次、503/Retry-After；多热项目代价交M8 |

## 16. 终审修复后的最终候选

新build10对应`t13-c12-final3-{chromium,firefox,webkit}`，三引擎各完整167/167通过，无skip/flaky，三个整轮与构建输入逐字节一致；不再使用旧并集作为最终候选。原375项选择和72项原生LAN复制全部保留。最终819原图选择303张，34张比较图板；本轮实际查看21张图板中的189张原图，另114张逐场景身份和PNG hash继承上轮实际观察，全部58张变化图已重新查看。其余原图作为自动回归附件保留，不冒称人工逐图检查。 R39/R40及F2/F3已闭环；Web3392、默认E2E336、Go1893顶层/10191含子例和七门禁见阶段review。D10/D13补旧编辑器复制入口具名风险/首次复用验收，八输入关闭、四输出开放的边界不变。准确候选SHA及分支CI只以外置final-delivery.json核准；D18和最终main绑定继续待获准集成。
