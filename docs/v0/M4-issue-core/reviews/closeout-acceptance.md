# M4 收尾验收台账

状态：实施中。2026-11-01 获准从 T1 开始；各项先登记、凭本次证据签结。最终 main CI、M4 §14 全勾和总体 §9.4 完成标记仍待集成。

基线 `01e40e113e11365a4eeacc2794973de2751dde61`；实施起点 `726f7156c2a053bd67fba1bfcaef5541df347749`；全期范围 `1eb9686a6ca527984337821af5fa7efae8c620e2..候选HEAD`。主目录 main 保持不动。P16 main CI [41290496985](https://github.com/open-nerve/NerveProject/actions/runs/41290496985) 和本分支预检 CI [41293001954](https://github.com/open-nerve/NerveProject/actions/runs/41293001954) 均为准确 SHA、attempt 1、success。

## 1. 证据协议

原始证据在 worktree `.superpowers/sdd/M4-closeout/`，技能 brief/进度在 `.superpowers/sdd/closeout/`；两根均保留。`t1-source-files.json` 逐文件 SHA/bytes，`t1-review-ranges.json` 固定两套 diff 清单；`t1-requirements.json` 保存设计每行原文、故事全部增长、36 个 Phase 来源和八份输入全文。来源的历史勾选与本次实际验收分开。

计划 14 Task、净手写新增/修改约 3500–6500 行；真实新增/删除/生成/文档在 T13 单列。集成包为 T14 Step 1，Step 2–4 需之后授权。

## 2. 十八条完成标准

| ID | 范围 | Owner | 当前证据与结论 |
|---|---|---|---|
| D01 | I1–I22适用的页面/API/任务版本与全部增长断言、S1/S2在CI通过 | T1/T4/T13/T14 | 待验；故事/增长逐行映射，实际id、raw、精确main run |
| D02 | 完整权限矩阵/访客一致性/26交错、拒绝不取锁、真实池与503 | T2/T3/T9/T10 | 待验；身份×操作矩阵、26行-race记录、等待证据/拒绝反证 |
| D03 | issue与project双向公开码、access判定/规则/操作目录/矩阵齐全 | T3/T13 | 待验；apitest与架构测试、操作清单无漏项 |
| D04 | 21个合法排序码（含各自升降取值，sort_order仅升序）/筛选/分组/续页/共享fixture/游标可见性/100k/连接设置/9反例 | T3/T4/T8 | 待验；Go+vitest、精确场景、SQL与请求分层样本、连接读回 |
| D05 | 事务投递提交/回滚、两类缺worker变异、registrationRows、固定时钟/无固定余量 | T3/T10/T13 | 待验；实际river_job、任务名全表、变异与clock故事 |
| D06 | P7出站IANA/DNS/固定IP/每跳重检/本机地址与I10 blocked | T3/T6/T10 | 待验；Go零危险dial证据、I10页面与API，历史子项不代替最终CI |
| D07 | purge目录/外键/I21、删除窗口/交错24/25/M3有限轮询 | T3/T9/T13 | 待验；catalog、runtime、实际行/时刻与轮询结果 |
| D08 | 七门禁、架构/SQLCSchemaScope/grants双测/CLI/keywords/lint/knip/gen | T4/T13/T14 | 待验；F每命令exit/raw、最终候选与mainCI |
| D09 | 八决策/十一架构问题已批准并落实 | T1/T10 | 待验；已批准文本与最终实现对照，无重开未经批准合同 |
| D10 | §13.1逐项/8输入closed；§13.2四输出及原文/收集箱细节 | T12/T13 | 待验；双向交接索引、逐条证据、原文一致性 |
| D11 | §3.23每行、§4.17每行、frontend M4、README§8.8 | T11 | 待验；44/45逐行映射与最终数量、目标段/来源 |
| D12 | C1–C12各Phase有证据，C12最终三引擎 | T5/T6/T7/T13/T14 | 待验；旧C索引、21组×3逐格截图/请求/控制者签结、最终main绑定 |
| D13 | M4 dead无未解释行；全仓no-shadow零/error；cap实数下降 | T11/T13 | 待验；原脚本hash、完整raw、逐行ledger、14范围实际cap |
| D14 | M4挂载零旧/无权请求、S2四身份、copy-hook无M4例外、13实例清空 | T4/T5/T6/T7/T11 | 待验；精确请求及重复数、关键词/源码检索与入口范围 |
| D15 | await后跳转/提示核会话，改动hook有换账户兑现测试，C11 | T4/T7/T10 | 待验；全期hook路径映射、真实迟到兑现/静默/新会话不污染 |
| D16 | 列表10k/20k/100k、中文两字搜索、规模删除请求/任务、一次清理并交M8 | T8/T9/T12 | 待验；场景/分布/正确性/原始samples/耗时表、M8链接 |
| D17 | 规模估计与实际对照 | T1/T13 | 待验；分阶段与本收尾计划/实改文件和行数、差异原因 |
| D18 | 总体设计9.4为已完成 | T14（集成后） | 待验；完成标记提交、最终mainSHA/三作业success、归档清理收据 |

## 3. 二十二个故事与增长断言

每条原文的页面、数据库、异步、PAT 和逐 Phase 增长均在 `t1-requirements.json:stories`；T4 逐子项核断言位置及实际执行。I21 仅任务/数据库，I22 仅页面，I16 PAT 读取任务结果，I7 接口沿 I1，不虚构版本。

| ID | 故事 | 当前测试入口 | Owner/状态 |
|---|---|---|---|
| I1 | 创建工作项 | [i1-create-issue.spec.ts](../../../../e2e/stories/issue/i1-create-issue.spec.ts) | T4，待验 |
| I2 | 在详情页改每个字段 | [i2-description.spec.ts](../../../../e2e/stories/issue/i2-description.spec.ts)、[i2-update-issue.spec.ts](../../../../e2e/stories/issue/i2-update-issue.spec.ts) | T4，待验 |
| I3 | 列表与看板：分组、子分组 | [i3-list-groups.spec.ts](../../../../e2e/stories/issue/i3-list-groups.spec.ts) | T4，待验 |
| I4 | 表格排序与日历 | [i4-list-order-calendar.spec.ts](../../../../e2e/stories/issue/i4-list-order-calendar.spec.ts) | T4，待验 |
| I5 | 筛选与显示设置的保存 | [i5-list-preferences.spec.ts](../../../../e2e/stories/issue/i5-list-preferences.spec.ts) | T4，待验 |
| I6 | 在看板上拖动（总体设计 8.2 的示例） | [i6-board-order.spec.ts](../../../../e2e/stories/issue/i6-board-order.spec.ts) | T4，待验 |
| I7 | 快速添加 | [i7-quick-add.spec.ts](../../../../e2e/stories/issue/i7-quick-add.spec.ts) | T4，待验 |
| I8 | 子工作项 | [i8-sub-issues.spec.ts](../../../../e2e/stories/issue/i8-sub-issues.spec.ts) | T4，待验 |
| I9 | 关联 | [i9-relations.spec.ts](../../../../e2e/stories/issue/i9-relations.spec.ts) | T4，待验 |
| I10 | 链接与元数据 | [i10-links.spec.ts](../../../../e2e/stories/issue/i10-links.spec.ts) | T4，待验 |
| I11 | 评论与表情回应 | [i11-comments-reactions.spec.ts](../../../../e2e/stories/issue/i11-comments-reactions.spec.ts) | T4，待验 |
| I12 | 操作动态的时间线 | [i12-activity.spec.ts](../../../../e2e/stories/issue/i12-activity.spec.ts) | T4，待验 |
| I13 | 描述的历史与还原 | [i13-description-history.spec.ts](../../../../e2e/stories/issue/i13-description-history.spec.ts) | T4，待验 |
| I14 | 草稿发布 | [i14-drafts.spec.ts](../../../../e2e/stories/issue/i14-drafts.spec.ts) | T4，待验 |
| I15 | 归档、恢复与归档页 | [i15-archive-issue.spec.ts](../../../../e2e/stories/issue/i15-archive-issue.spec.ts) | T4，待验 |
| I16 | 自动归档 | [i16-auto-archive.spec.ts](../../../../e2e/stories/issue/i16-auto-archive.spec.ts) | T4，待验 |
| I17 | 删除（连同子树） | [i17-catalog.spec.ts](../../../../e2e/stories/issue/i17-catalog.spec.ts)、[i17-delete-issue.spec.ts](../../../../e2e/stories/issue/i17-delete-issue.spec.ts) | T4，待验 |
| I18 | 搜索 | [i18-search.spec.ts](../../../../e2e/stories/issue/i18-search.spec.ts) | T4，待验 |
| I19 | 个人主页的工作项 | [i19-member-issues.spec.ts](../../../../e2e/stories/issue/i19-member-issues.spec.ts) | T4，待验 |
| I20 | 访客的边界（权限矩阵的页面抽样，9.2） | [i20-guest-boundary.spec.ts](../../../../e2e/stories/issue/i20-guest-boundary.spec.ts) | T4，待验 |
| I21 | 物理清理 | [i21-purge.spec.ts](../../../../e2e/stories/issue/i21-purge.spec.ts) | T4，待验 |
| I22 | 工作项页上的会话切换 | [i22-session-switch.spec.ts](../../../../e2e/stories/issue/i22-session-switch.spec.ts) | T4，待验 |

## 4. 二十六个交错

T3 逐项绑定具名测试、子例、真实锁/闸门观察及五次 race 执行；T9 补池的测量。当前只登记来源，不以名称命中判通过。

| ID | 原文合同 | Owner/证据 |
|---|---|---|
| L01 | 同一项目两次创建（编号连续、不重复；第二个等第一个提交）； | T3（L20 另 T9），待验 |
| L02 | 创建与删除项目（两种顺序：先建的随项目删除，与项目同一时刻，P9 起由任务写下；后建的 404）； | T3（L20 另 T9），待验 |
| L03 | 修改工作项与删除工作区（两种顺序；删除先拿到工作区 N 而工作项的写已持项目 N 时，删除等它提交，删除的时刻晚于它的 `updated_at`，任务写下的也是这个时刻）； | T3（L20 另 T9），待验 |
| L04 | 修改工作项与移出调用者（移出先提交：写在锁下判定得到 404）； | T3（L20 另 T9），待验 |
| L05 | 降为访客与修改工作项（降级先提交：403，或访客规则下 404）； | T3（L20 另 T9），待验 |
| L06 | 删除状态与创建、修改用它的工作项（状态先删：422 `state_id`；工作项先写：409 `project.state_in_use`）； | T3（L20 另 T9），待验 |
| L07 | 删除标签与给工作项加这个标签； | T3（L20 另 T9），待验 |
| L08 | 归档项目与创建工作项（归档先提交：409 `project.archived`）； | T3（L20 另 T9），待验 |
| L09 | 两次改父工作项（两次移动；加子工作项与移动）各自合法、合起来成环（第二个在锁下沿祖先走，422 `parent_id` 或 `sub_issue_ids[i]`）； | T3（L20 另 T9），待验 |
| L10 | 两个方向相反的跨项目关联同时加（同一对只留一行，后到的 422 `duplicate`；两个项目按 `id` 升序锁，不死锁）； | T3（L20 另 T9），待验 |
| L11 | 删除带跨项目关联的工作项与删除对方项目（关联行按 `id` 升序锁，不死锁；关联行只有一次删除）； | T3（L20 另 T9），待验 |
| L12 | 归档与把它的状态改到"进行中"（先改状态：409 `issue.not_archivable`）； | T3（L20 另 T9），待验 |
| L13 | 删除父工作项与给它加子工作项（删除先提交：404；加先提交：新的子工作项一并被删除）； | T3（L20 另 T9），待验 |
| L14 | 发评论与删除工作项（删除先提交：404；评论先提交：评论被连带删除，同一时刻）； | T3（L20 另 T9），待验 |
| L15 | 同一个人连续两次改描述（动态合并为一行、版本合并为一行，时刻不减）； | T3（L20 另 T9），待验 |
| L16 | 发布草稿与删除它的项目（删除先提交：草稿已随项目删除，P9 起任务还没到时草稿的读也把它当作不存在，404 `issue.draft_not_found`）； | T3（L20 另 T9），待验 |
| L17 | 发布草稿与另一个请求把草稿改到别的项目（409 `issue.draft_changed`）； | T3（L20 另 T9），待验 |
| L18 | 自动归档与把工作项的状态改回"进行中"（改先提交：不归档；归档先提交：修改 409 `issue.archived`）； | T3（L20 另 T9），待验 |
| L19 | 抓取与改链接地址（抓取写 0 行，新的地址另有一次抓取）； | T3（L20 另 T9），待验 |
| L20 | 饥饿与连接池：经真实的连接池和写闸门，一串首尾相连的工作项写（同一工作区的若干项目）不让改工作区、改工作区成员的角色等到期限；删除工作区只等每个项目上正在进行和闸门放进来的写；200 个写压在同一个项目上时，另一个项目上无关的读的耗时不超过上限，多出的写得到 503 `server_busy`（S5、SP1 的回归测试）； | T3（L20 另 T9），待验 |
| L21 | 取锁的形状：每个工作项的写等第一把锁时在工作区行上什么都不持，它取的第一把锁是项目的 N，之后也不取与 `FOR NO KEY UPDATE` 冲突的工作区行的锁（插入经外键取工作区行的 `FOR KEY SHARE`，与它不冲突，S5 的 E3）；看不到项目的账户、看得到而不能写的账户（公开项目里不是成员的工作区成员、不是项目成员的工作区管理员）的写不取行锁、不写；加关联指向看不到的项目时不锁那个项目（不加锁的预检；这一例是自己的测试 `TestARelationLocksNoProjectItDoesNotReach`，与取锁形状的测试的 `createIssueRelations`、`deleteIssueRelation` 两行并列：取锁形状的测试按操作锁表，造不出"看得到工作项、看不到对方的项目"）。用 `pg_locks` 观察，与 M3 的同类测试同一写法； | T3（L20 另 T9），待验 |
| L22 | 移出项目的默认负责人与不带负责人的创建（移出先提交：新的工作项没有负责人；创建先提交：他被指派，随后的移出不改这一行，与 Plane 相同）； | T3（L20 另 T9），待验 |
| L23 | 加关联 A→B 与删除 B 的项目（两种顺序：删除先提交：关联的写在锁下看到 B 的项目已删除，422，不写任何行；关联先提交：随项目删除这条关联行和反向的动态，与项目同一时刻（P9 起由任务写下），没有活的关联行指向已删除的工作项；S5 的 E5、E5b）； | T3（L20 另 T9），待验 |
| L24 | 删除项目 P 的任务与另一个项目 Q 上删除关联另一端的工作项 Y（P9）：P 先删除、任务还没写到时删除 Y，Y 的删除不写这条关联行，P 的任务随后写下 P 的时刻和删除者；Y 先删除（P 还在）时这一行是 Y 的时刻和删除者，P 的任务跳过它；任务与 Y 的删除同时锁关联行时按 `id` 先锁，两种顺序都不死锁；断言每一行的 `deleted_at`、`updated_by_id`，不只是"只删了一次"。任务挂起期间，从另一个项目对已删除项目里的工作项加关联 422、删除指向它的关联 404 `issue.relation_not_found`； | T3（L20 另 T9），待验 |
| L25 | 删除项目 Old，它的任务还没做完时删除它的工作区（P9，聚焦复核 I1）：Old 的每一行工作项一侧的行（含 Old 与工作区另一个项目之间的关联行）都是 Old 的时刻和删除者，工作区其余项目的行都是工作区的；没有项目的草稿是工作区的。M3 的 W3（先删 Old、再删工作区，断言 Old 的行留着 Old 的删除）在 P9 的轮询之下因此确定地通过。 | T3（L20 另 T9），待验 |
| L26 | 删除关联与删除它另一端的工作项（P3 加，编号接在最后：20–25 在各文档里被引用）：删除关联不论在哪一端判定都锁两端的项目，与删除工作项在那个工作项的项目行上排队；关联先删，工作项的删除读到它已删除、跳过，留着那次删除；工作项先删，关联的删除在锁下重读两端，读到那个工作项已删除（关联已随它删除），404 `issue.relation_not_found`，什么都不写。 | T3（L20 另 T9），待验 |

## 5. 同步与差异逐行索引

以下 stable ID 绑定 T1 保存的完整原文；不是删除或改写设计行。T11 必须读实际目标完整段并给事实证据。

| ID | 原文键/主题 | 目标/责任 | 状态 |
|---|---|---|---|
| SYNC01 | P1a | 总体设计 / 4.2 / 加锁顺序延伸到 M4 的表；工作项的写最先锁项目行、不锁工作区行，取锁之前做不加锁的预检（凡拒绝的都在那里答）；删除工作区先锁它的项目再读时刻；写明 B 的两个代价（3.5，决策点 1） | T11 待验 |
| SYNC02 | P1b | 总体设计 / 4.2 / 取连接之前进入按项目的写闸门、`lock_timeout` 的兜底，满时 503 `server_busy`；写明连接池的代价（3.5，架构问题 11） | T11 待验 |
| SYNC03 | P1b | 总体设计 / 6.2 / 平台包 `platform/gate`（带 `context` 的按键的信号量，3.5，架构问题 11） | T11 待验 |
| SYNC04 | P1a | 总体设计 / 6.2 / 模块 `issue` 和它的 13 张表（3.3，架构问题 1） | T11 待验 |
| SYNC05 | P1a | M3 设计 / 3.6 的取锁表、17.4 / "删除工作区"一行改为先锁项目、再读时刻；17.4"M4 的工作项写入……不默认沿用"后补一句 M4 的选择（3.5） | T11 待验 |
| SYNC06 | P1a | M3 设计 / 3.17 / 删除状态：有工作项（含已归档的）用它时 409 `project.state_in_use`（3.6） | T11 待验 |
| SYNC07 | P1a | 差异清单 / 二·按表 / `issues`、`issue_assignees`、`issue_labels`、`issue_subscribers`、`issue_activities` 逐列（4.2–4.5、4.11） | T11 待验 |
| SYNC08 | P1a | 差异清单 / 四 / 4.17 中标 P1a 的行 | T11 待验 |
| SYNC09 | P1b | 差异清单 / 四 / 4.17 中标 P1b 的行 | T11 待验 |
| SYNC10 | P2 | 总体设计 / 6.5 / `AllowCreator` 要求创建者仍是项目的有效成员、角色不低于成员；`OwnOnly` 与"一条访客规则"（3.4，架构问题 3） | T11 待验 |
| SYNC11 | P2 | M2 设计 / 3.11 / 格式检查器登记 `date`；字段错误的路径按数组下标的数值排序；不限类型的节点检查数的范围（6.1，M2 交接第 4 节） | T11 待验 |
| SYNC12 | P2 | 差异清单 / 四 / 4.17 中标 P2 的行（访客规则、创建者、归档只读、按编号取的 404 和折叠、起止日期） | T11 待验 |
| SYNC13 | P3 | 总体设计 / 3.4 / 集合型的列表加上子工作项、关联、链接、工作项的表情（3.16 末尾）。P3 有第一个（`listSubIssues`） | T11 待验 |
| SYNC14 | P3 | 总体设计 / 3.6 / "关联对象只返回 ID"的第二个例外：关联内嵌对方工作项的列表行（3.10） | T11 待验 |
| SYNC15 | P3 | 总体设计 / 4.2 / 约定七（跨项目的写按 `id` 升序锁住涉及的项目）和关联行的约定五例外（3.5） | T11 待验 |
| SYNC16 | P3 | 差异清单 / 二·按表、四 / `issue_relations` 逐列（4.6）；4.17 中标 P3 的行 | T11 待验 |
| SYNC17 | P4 | 总体设计 / 1.1 / "评论（支持回复）"改为"评论"（决策点 6）；"历史版本：工作项快照、描述修改历史"改为"描述修改历史（可查看、可还原）"（决策点 5） | T11 待验 |
| SYNC18 | P4 | 总体设计 / 5.2、5.3、6.6、6.7 / 表数 45 改为 44，删去 `issue_versions`；6.6 删去"同时写 `issue_versions`"；6.7 的保留任务写明"每个工作项最新的 20 个描述版本"（3.14） | T11 待验 |
| SYNC19 | P4 | M2 设计 / 3.12 / 页大小的规则移到 `shared`（M3 交接第 9 节） | T11 待验 |
| SYNC20 | P4 | 差异清单 / 一、二·按表、四 / 一：`issue_versions` 不保留；二：`issue_comments`、`issue_reactions`、`comment_reactions`、`issue_mentions`、`issue_description_versions` 逐列（4.7、4.9、4.10、4.12、4.14）；四：4.17 中标 P4 的行 | T11 待验 |
| SYNC21 | P5 | 总体设计 / 3.3 / 筛选的写法补上 `none`（含日期的 `none`）、可重复的日期参数、单日、不合格的取值 422（3.16）；已在P5/T16同步，实际依据见 [P5 review §2](P5-issue-list-review.md#2-验收标准逐条核对) | T11 待验 |
| SYNC22 | P5 | 总体设计 / 3.4 / 不分组的列表带 `total`；子分组的结构 `subgroups` 和泳道的总数 `subgroup_totals`；只返回非空的组；分组时不带 `group_key` 的游标 422；列表的读在一个 `REPEATABLE READ` 的快照里；一组（一格）的"加载更多"是同一个请求加 `group_key`（`sub_group_key`），不再是"加上这一组的筛选"；游标的载荷写明排序、以 `id` 结尾；未知的 `order_by`、`group_by` 422（3.16，架构问题 6）；已在P5/T16同步，实际依据见 [P5 review §2](P5-issue-list-review.md#2-验收标准逐条核对) | T11 待验 |
| SYNC23 | P5 | 总体设计 / 6.2 / 列表的词汇在 `issue/domain`，`project` 经它声明、`issue.Provide` 实现的 `IssueListSettings` 检查显示设置（3.17，架构问题 10）；已在P5/T16同步，实际依据见 [P5 review §2](P5-issue-list-review.md#2-验收标准逐条核对) | T11 待验 |
| SYNC24 | P5 | M3 设计 / 3.18 / 项目的显示设置加 `filters`、`display_filters`、`display_properties` 三列，格式见本文 3.17；已在P5/T16同步，实际依据见 [P5 review §2](P5-issue-list-review.md#2-验收标准逐条核对) | T11 待验 |
| SYNC25 | P5 | 差异清单 / 二·按表、三、四 / `project_user_properties` 加三列、不建 `rich_filters`（4.15）；三：列表的响应、排序键、分组键的名字；四：4.17 中标 P5 的行；已在P5/T16同步，实际依据见 [P5 review §2](P5-issue-list-review.md#2-验收标准逐条核对) | T11 待验 |
| SYNC26 | P6 | 总体设计 / 6.2、6.3 / `shared` 加事件的值和 `EventPublisher`；6.3 第 3 条写明订阅者的写法：同步、在事务里、出错回滚，异步的事经 `InsertTx`（3.19，决策点 3） | T11 待验 |
| SYNC27 | P6 | 总体设计 / 6.7 / 定时任务的清单和间隔、两个队列；测试的时钟偏移（3.20，架构问题 7） | T11 待验 |
| SYNC28 | P6 | M2 设计 / 3.15、3.17 / 事务内投递经只投递的 River 客户端（在模块之前建出）；队列与并发；定时任务和模块声明会投递的种类没有 worker 时构造失败（3.20） | T11 待验 |
| SYNC29 | P6 | 差异清单 / 四 / 自动归档的执行者为空（决策点 4）；按间隔而不是固定时刻运行 | T11 待验 |
| SYNC30 | P7 | 总体设计 / 6.2、8（安全） / `platform/outbound`：防 SSRF 的出站客户端，M8 复用（3.11、8.1，架构问题 8） | T11 待验 |
| SYNC31 | P7 | 差异清单 / 二·按表、四 / `issue_links` 逐列（4.8）；抓取的上限、`metadata` 的形状 | T11 待验 |
| SYNC32 | P7 | README / "安全"一节 / 8.8 中 P7 的行；P6、P9 的"部署"行同样按 8.8 | T11 待验 |
| SYNC33 | P8 | 总体设计 / 5.4 / 草稿：`project_id` 是一列；发布用存着的载荷、要求 `issue.create`；409 `issue.draft_changed`（3.15） | T11 待验 |
| SYNC34 | P8 | 总体设计 / 6.9 / `description_stripped` 不用于搜索；搜索的匹配和顺序、选择器的排除、`q` 为空时给出最近更新的、日期的条件、编号的折叠（3.7、3.18） | T11 待验 |
| SYNC35 | P8 | 差异清单 / 二·按表、四 / `draft_issues` 逐列（4.13）；`issues` 的 trigram 索引落地；4.17 中标 P8 的行 | T11 待验 |
| SYNC36 | P8 | README / "部署"一节 / 迁移执行 `CREATE EXTENSION pg_trgm`（库属主即可，8.8） | T11 待验 |
| SYNC37 | P9 | 总体设计 / 5.5、6.7 / 60 天清理：每个模块清理自己的表，一个任务按顺序运行；删除关系图（3.21、4.16，架构问题 9）；描述版本保留只碰活版本、清理处理墓碑；迟到软删与父CASCADE的交叠例外见3.6 **P9回执：** 已同步总体§5.5/6.7：23表单任务清理、活版本保留、严格cutoff/父CASCADE与迟到40P01边界；T1–6/T12–15证据见P9 review | T11 待验 |
| SYNC38 | P9 | 总体设计 / 5.5、6.3 / "连带删除在同一个事务里同步完成"改为：删除项目、工作区时工作项一侧的行由事件的订阅者投递的任务分批软删除，写下请求的时刻和删除者；第一个跨模块的异步副作用（`project.deleted`、`workspace.deleted`）；v0 的规模目标（一个项目至多约 2 万个未删除的工作项）（3.6、3.19，决策点 7） **P9回执：** 已同步总体§5.5/6.3：M3同步、issue同事务投递后分批软删/立即不可见/原AtBy；T9–13与约2万issue每项目实测见P9 spec附录B | T11 待验 |
| SYNC39 | P9 | M3 设计 / 3.6、13.2 / 删除项目、工作区的步骤后面是发布事件，不再调 `IssueCascade` 的删除；13.2 设想的"删除工作区的连带挂到事件上"对工作项一侧做了（3.19） **P9回执：** 已同步M3§3.3/3.6/6.2/6.5/13.2：只移除issue两同步删除能力，邀请/M3同步与StateInUse/DetachLabel保留；T9/T10回执见P9 review | T11 待验 |
| SYNC40 | P9 | 差异清单 / 四 / 动态随工作项物理删除（`CASCADE`，不是 `DO_NOTHING`）；已登记的"连带在同一事务"改为只对删除工作项、状态、标签成立，删除项目、工作区的工作项一侧与 Plane 一样在事务之外完成，但写下请求的时刻（3.6） **P9回执：** 已同步plane-diff第四节及版本索引行：活动/评论子链CASCADE、同步/异步按操作划分；T3/T5/T9–13证据见P9 review | T11 待验 |
| SYNC41 | P10b | 总体设计 / 7.7 / `Reconciled.fetchPart` 和 `paged-groups.ts`；一个工作项只有一份，取服务端 `updated_at` 大的（3.22） | T11 待验 |
| SYNC42 | P10a–P16 | 前端改动清单 / 3.1、3.2 / M4 一行的状态；删到 M5、M6、M7 的入口和取数（3.1、3.2）；迭代、模块、项目视图、工作区视图的工作项列表在 P10a 删除，M6、M7 加回（7.3） | T11 待验 |
| SYNC43 | P12 | README / "前端"一节 / 已按实际更新四布局/原生搜索及后续页面边界；最终e6d1329b与main ff89d0a1准确CI均通过、归档清理完成，见P12 review§28 | T11 待验 |
| SYNC44 | 收尾 | 总体设计 / 9.4 / M4 的状态改为"已完成" | T11 待验 |
| DIFF01 | 工作项的写的取锁 | 差异清单第四节；P1a | T11 待验 |
| DIFF02 | 一个项目上的并发写 | 差异清单第四节；P1b | T11 待验 |
| DIFF03 | 删除项目、工作区时工作项一侧的行 | 差异清单第四节；P9 | T11 待验 |
| DIFF04 | 负责人、标签不合格 | 差异清单第四节；P1a | T11 待验 |
| DIFF05 | 起止日期 | 差异清单第四节；P2 | T11 待验 |
| DIFF06 | 默认负责人 | 差异清单第四节；P1a | T11 待验 |
| DIFF07 | 负责人的动态 | 差异清单第四节；P1a | T11 待验 |
| DIFF08 | 只补块 id 的保存 | 差异清单第四节；P4 | T11 待验 |
| DIFF09 | 描述的动态的合并 | 差异清单第四节；P4 | T11 待验 |
| DIFF10 | 自动关注 | 差异清单第四节；P1a（评论、提及 P4） | T11 待验 |
| DIFF11 | 编号 | 差异清单第四节；P1a | T11 待验 |
| DIFF12 | 删除状态 | 差异清单第四节；P1a | T11 待验 |
| DIFF13 | 修改的响应 | 差异清单第四节；P1b | T11 待验 |
| DIFF14 | 负责人的读取 | 差异清单第四节；P1b | T11 待验 |
| DIFF15 | 创建者的权限 | 差异清单第四节；P2 | T11 待验 |
| DIFF16 | 访客只看自己的工作项 | 差异清单第四节；P2（各列表在各自的 Phase） | T11 待验 |
| DIFF17 | 访客删除自己建的工作项 | 差异清单第四节；P2 | T11 待验 |
| DIFF18 | 已归档的工作项 | 差异清单第四节；P2（子资源在各自的 Phase） | T11 待验 |
| DIFF19 | 归档、恢复 | 差异清单第四节；P2 | T11 待验 |
| DIFF20 | 按编号取 | 差异清单第四节；P2 | T11 待验 |
| DIFF21 | 删除工作项 | 差异清单第四节；P3（确认框 P14） | T11 待验 |
| DIFF22 | 父工作项 | 差异清单第四节；P3 | T11 待验 |
| DIFF23 | 加子工作项的动态 | 差异清单第四节；P3 | T11 待验 |
| DIFF24 | 子工作项数 | 差异清单第四节；P3 | T11 待验 |
| DIFF25 | 关联 | 差异清单第四节；P3 | T11 待验 |
| DIFF26 | 关联的动态 | 差异清单第四节；P3 | T11 待验 |
| DIFF27 | 空评论 | 差异清单第四节；P4 | T11 待验 |
| DIFF28 | 改、删别人的评论 | 差异清单第四节；P4 | T11 待验 |
| DIFF29 | 表情的值 | 差异清单第四节；P4 | T11 待验 |
| DIFF30 | 评论、表情的动态 | 差异清单第四节；P4 | T11 待验 |
| DIFF31 | 提及的同步 | 差异清单第四节；P4 | T11 待验 |
| DIFF32 | 描述的历史 | 差异清单第四节；P4 | T11 待验 |
| DIFF33 | 动态的列表 | 差异清单第四节；P4 | T11 待验 |
| DIFF34 | 时间线的增量读取 | 差异清单第四节；P4 | T11 待验 |
| DIFF35 | 列表的分组、排序 | 差异清单第四节；P5 | T11 待验 |
| DIFF36 | 已归档的项目的工作项列表 | 差异清单第四节；P5 | T11 待验 |
| DIFF37 | 个人主页的工作项 | 差异清单第四节；P5 | T11 待验 |
| DIFF38 | 显示设置 | 差异清单第四节；P5 | T11 待验 |
| DIFF39 | 自动归档的执行者 | 差异清单第四节；P6 | T11 待验 |
| DIFF40 | 定时任务的时刻 | 差异清单第四节；P6 | T11 待验 |
| DIFF41 | 链接的地址 | 差异清单第四节；P7 | T11 待验 |
| DIFF42 | 链接的抓取 | 差异清单第四节；P7 | T11 待验 |
| DIFF43 | 草稿 | 差异清单第四节；P8 | T11 待验 |
| DIFF44 | 搜索 | 差异清单第四节；P8 | T11 待验 |
| DIFF45 | 物理清理 | 差异清单第四节；P9 | T11 待验 |

## 6. C1–C12 与最终浏览器记录

| 编号 | 原阶段 | 当前验收位置 |
|---|---|---|
| C1 | P12 | T5–T7；[最终矩阵](closeout-browser-matrix.md)，待验；完整合同见原文清单 |
| C2 | P13 | T5–T7；[最终矩阵](closeout-browser-matrix.md)，待验；完整合同见原文清单 |
| C3 | P12、P14 | T5–T7；[最终矩阵](closeout-browser-matrix.md)，待验；完整合同见原文清单 |
| C4 | P14 | T5–T7；[最终矩阵](closeout-browser-matrix.md)，待验；完整合同见原文清单 |
| C5 | P14 | T5–T7；[最终矩阵](closeout-browser-matrix.md)，待验；完整合同见原文清单 |
| C6 | P15 | T5–T7；[最终矩阵](closeout-browser-matrix.md)，待验；完整合同见原文清单 |
| C7 | P15 | T5–T7；[最终矩阵](closeout-browser-matrix.md)，待验；完整合同见原文清单 |
| C8 | P15 | T5–T7；[最终矩阵](closeout-browser-matrix.md)，待验；完整合同见原文清单 |
| C9 | P13、P15 | T5–T7；[最终矩阵](closeout-browser-matrix.md)，待验；完整合同见原文清单 |
| C10 | P14 | T5–T7；[最终矩阵](closeout-browser-matrix.md)，待验；完整合同见原文清单 |
| C11 | P14 | T5–T7；[最终矩阵](closeout-browser-matrix.md)，待验；完整合同见原文清单 |
| C12 | 收尾 | T5–T7；[最终矩阵](closeout-browser-matrix.md)，待验；完整合同见原文清单 |

## 7. 阶段遗留与输入交接

18 个 Phase 的 spec §5 与 review §6 各一份，36 份来源均读取并逐 hash 固定。P16 dead-members §6 作为附加支持材料，不冒充第十九个 Phase。T12 逐原条款列已完成证据或具名后续归属，不能只改 status。

| Phase | spec §5 / review §6 | 后续归属 |
|---|---|---|
| P10a | [spec](../specs/P10a-web-issue-types.md) / [review](P10a-web-issue-types-review.md) | T12 待核全部原条款 |
| P10b | [spec](../specs/P10b-web-list-data.md) / [review](P10b-web-list-data-review.md) | T12 待核全部原条款 |
| P11 | [spec](../specs/P11-web-detail-data.md) / [review](P11-web-detail-data-review.md) | T12 待核全部原条款 |
| P12 | [spec](../specs/P12-web-list-pages.md) / [review](P12-web-list-pages-review.md) | T12 待核全部原条款 |
| P13 | [spec](../specs/P13-web-filters-archives-profile.md) / [review](P13-web-filters-archives-profile-review.md) | T12 待核全部原条款 |
| P14 | [spec](../specs/P14-web-issue-detail.md) / [review](P14-web-issue-detail-review.md) | T12 待核全部原条款 |
| P15 | [spec](../specs/P15-web-collaboration-drafts-search.md) / [review](P15-web-collaboration-drafts-search-review.md) | T12 待核全部原条款 |
| P16 | [spec](../specs/P16-web-cleanup.md) / [review](P16-web-cleanup-review.md) | T12 待核全部原条款 |
| P1a | [spec](../specs/P1a-issues.md) / [review](P1a-issues-review.md) | T12 待核全部原条款 |
| P1b | [spec](../specs/P1b-issue-update.md) / [review](P1b-issue-update-review.md) | T12 待核全部原条款 |
| P2 | [spec](../specs/P2-issue-access.md) / [review](P2-issue-access-review.md) | T12 待核全部原条款 |
| P3 | [spec](../specs/P3-issue-hierarchy.md) / [review](P3-issue-hierarchy-review.md) | T12 待核全部原条款 |
| P4 | [spec](../specs/P4-issue-collaboration.md) / [review](P4-issue-collaboration-review.md) | T12 待核全部原条款 |
| P5 | [spec](../specs/P5-issue-list.md) / [review](P5-issue-list-review.md) | T12 待核全部原条款 |
| P6 | [spec](../specs/P6-issue-events.md) / [review](P6-issue-events-review.md) | T12 待核全部原条款 |
| P7 | [spec](../specs/P7-issue-links.md) / [review](P7-issue-links-review.md) | T12 待核全部原条款 |
| P8 | [spec](../specs/P8-drafts-search.md) / [review](P8-drafts-search-review.md) | T12 待核全部原条款 |
| P9 | [spec](../specs/P9-purge.md) / [review](P9-purge-review.md) | T12 待核全部原条款 |

| 输入 | 关闭条件 |
|---|---|
| [M0-P4-pg-trgm](../handoffs/M0-P4-pg-trgm.md) | T12 全部条款完成或具名转交；当前保持 open |
| [M1-P2-trim-content](../handoffs/M1-P2-trim-content.md) | T12 全部条款完成或具名转交；当前保持 open |
| [M1-P3-trim-platform](../handoffs/M1-P3-trim-platform.md) | T12 全部条款完成或具名转交；当前保持 open |
| [M1-P4-router-native](../handoffs/M1-P4-router-native.md) | T12 全部条款完成或具名转交；当前保持 open |
| [M1-P5-brand](../handoffs/M1-P5-brand.md) | T12 全部条款完成或具名转交；当前保持 open |
| [M1-closeout](../handoffs/M1-closeout.md) | T12 全部条款完成或具名转交；当前保持 open |
| [M2-closeout](../handoffs/M2-closeout.md) | T12 全部条款完成或具名转交；当前保持 open |
| [M3-closeout](../handoffs/M3-closeout.md) | T12 全部条款完成或具名转交；当前保持 open |

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
