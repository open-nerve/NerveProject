# M4 / closeout 执行计划

> 预检已通过，2026-11-01 获准使用 `superpowers:executing-plans` 从 T1 执行，依据 [spec](../specs/closeout.md)；每一步凭实际证据勾选。工作树 `<repo>/.codex/worktress/codex/m4-closeout`，基线 `01e40e113e11365a4eeacc2794973de2751dde61`。本轮完成后 deep-review、保留分支，先不合并。

**目标：** 完成 C12、设计 §14 十八条及最终交接，以准确 main CI 和归档收据关闭 M4。

**结构：** 沿既有领域/应用/适配器、真实 API、数据库夹具和受控浏览器测试，补验收缺口并修局部根因；不增加生产依赖、状态所有者或跨模块协议。技术栈是仓库已锁定的 Go/PostgreSQL/River、React/MobX、Vitest/Playwright，不升级版本。

**文件结构：** 以 spec §2 的十四行文件表为准。S/E/C/P/D缩写同spec。新增 `reviews/closeout-{acceptance,browser-matrix,dead-members,review}.md` 按证据责任拆开；两份 `docs/reviews/re-NN-*` 分别记录全M4与本分支审查。只在真实缺口需要时改产品或补测试。

## Global Constraints

- 主目录永远 main，所有修改在指定 worktree；plane/refer只读、不在其中用git，不动已有容器/Docker、不全局安装/新依赖、不stash/clean/reset-hard，不提取凭据。详细中文提交、系统 `/usr/bin/git` 推送，保留 `test/fix/docs(M4/closeout): …` 前缀。
- T1实施才改phase=M4/closeout。规则、ignore、到期例外、lint cap不放宽；产品新文案同步en/zh-CN，键盘用既有isComposing/IME约定，hook的await后副作用核会话。
- 14Task，plan不超过约1500行，估计本阶段净手写新增/修改约3500–6500行（测试/文档为主，删除单列）；每Task预计范围见下。局部发现多时先按职责拆至≤16Task；仍超出或架构/协议/跨领域责任变化先提交已完成工作再上报。
- DB/testcontainers与三引擎套件串行，不在runner期间改输入。保留严格watchPage、console、CSP、权限、计数、真实连接池/任务合同；失败来自产品、测试同步、基础设施分别归类。
- 公用帮手：pgtest.Soon/RowWaits、postgres.LockNotAvailable、apitest、issueWorld/newSplitRoles、job_state_observer、API fixtures、clock世界、watchPage/recordFocuses/pageBehindHidden/modalDialog。不新造第二套完整世界；业务E2E数据只经API，SQL仅断言/明确时钟夹具；opt-in Go规模seed沿既有隔离库做法。
- 原有正确性质直接验；真实缺口先最小红例，修后绿及有效变异。文档整理不写镜像实现的单测，不把无测试命中/skip/编译失败当业务反证。
- 未来实施完成先不合并。T14准备集成包，只有负责人给出当次集成授权才执行合并后步骤；本轮P16集成授权不延伸到M4 closeout。

## Review Focus

1. 旧阶段证据在当前源码已失效：T1/T10固定全期范围与输入；T13做最终等价校验，不按旧绿勾完成。
2. 验收选到了零个Firefox/WebKit或漏入口：T5–T7逐id清单与成功截图、真实browserName，不依赖题名或默认项目。
3. 性能测量只报种数据/总test或默认skip：T8/T9要求actual samples、oracle、job边界与非零工作量。
4. 交接关闭只改status、丢原文或未实现领域被误报通过：T12逐条双向索引/原文比较，不扩大M4验收归属。
5. 完成标签先于mainCI，清理丢ignored证据：T13/T14保留未集成状态、双阶段精确SHA收据和源/归档清单对比。

## 验证命令与证据协议

命令默认在worktree根；Go命令明确在server内。测试日志和结构化输出保存 `.superpowers/sdd/M4-closeout/`；环境变量 `M4_CLOSEOUT_EVIDENCE` 设为该目录绝对路径，不能覆盖HOME/CODEX_HOME。每条记录argv/cwd/start/end/exit、commit或dirty逐文件SHA、executed/skipped/flaky/失败原因；程序/截图/trace/SQL计划都在manifest中。各任务结束更新review和台账、提交可审查结果。

F（有产品变化的push检查点T4/T9及最终T13，串行；未变且没有失败/风险的输入可核验复用）：

~~~sh
make gen-check
make lint-go
GOFLAGS=-p=1 make test
make lint-web
make knip
TURBO_FORCE=true make test-web
make e2e
~~~

预期：七条exit0，Go无失败/死锁，lint零error且cap实数一致，knip无未用，生成无diff，E2E实际选择无unexpected/flaky/意外skip。规模opt-in在普通Go全测中跳过是已知事实，只由T8/T9显式运行验收。每次push前先提交实际受测输入，核本地/远端HEAD与CI head_sha一致；最后文稿再提交则核新的HEAD CI。

浏览器B（先make build，三个引擎依次执行，保留成功截图与trace；配置原输入应完整包含P16每引擎125项）：

~~~sh
pnpm --filter @nerve/e2e exec playwright test --config=playwright.m4-cleanup.config.ts --list --reporter=json
pnpm --filter @nerve/e2e exec playwright test --config=playwright.m4-cleanup.config.ts --project=chromium --workers=2 --trace=on
pnpm --filter @nerve/e2e exec playwright test --config=playwright.m4-cleanup.config.ts --project=firefox --workers=2 --trace=on
pnpm --filter @nerve/e2e exec playwright test --config=playwright.m4-cleanup.config.ts --project=webkit --workers=2 --trace=on
~~~

上述自动回归不代替控制者实际操作/截图判读。可以按T5/T6/T7用准确file/title子集分批，最终并集必须等于原选择加新增用例；保留选取证明。控制者补看自动未覆盖路径，用项目已锁Playwright和现有fixtures，成功时显式截图；每格记真实按钮/焦点/响应/观察。不能仅读trace摘要而签视觉通过。

## Task 1：固定范围和十八条验收目录（约250–450行）

**Files：** tools/keywords.json；D/reviews/closeout-acceptance.md（新增）、closeout-browser-matrix.md（新增）；准备inventory及所有Phase review。
**Interfaces：** D01–D18对应spec§4；故事条目键为story+page/API+增长条款，交错键为设计编号，浏览器键为页面组+路径+身份+引擎。

- [x] Step 1：核main和本分支基线、P16精确mainCI/归档；记录full-M4与closeout两个范围及全部文件清单、基线源码/文档hash。没有实施授权不得做后续步骤。
- [x] Step 2：phase改M4/closeout；确认copy-hook无M4例外、old URLs/negation没有因收尾新增放行；运行`make lint-web`，以原关键词samples核到期行为。
- [x] Step 3：从设计§2/§3.5/§3.23/§4.17/§8.8/§9.7/§14逐条生成映射并人工核增长项；列22故事、26交错、44同步行、45差异行、C1–C12、18完成条款，按当前实际内容核数而非硬删多项。
- [x] Step 4：给每条来源、当前测试/源码、待验证证据位置、owner Task；核所有18个Phase spec§5/review§6与8输入。每个未满足项保持未勾；记录预算与预计改动量，提交 `docs(M4/closeout): 固定最终验收范围与证据目录`。

## Task 2：收掉历史锁断言重复（约100–300行）

**Files：** identity/provide_test.go；identity/adapter/postgres/{admin,accounts,credentials}_test.go；workspace/adapter/postgres/{directory,directory_share,update_invitation,locks,deactivation,deactivation_invitations_lock,delete_workspace_invitations}_test.go；project/adapter/postgres/state_writes_test.go；bootstrap/{project_write_locks,demotion,interleaving_growth}_test.go。均相对S/modules或S。
**Interfaces：** `postgres.LockNotAvailable(err error) bool` 分类真实/包装PgError；`pgtest.RowWaits(t testing.TB,pool *pgxpool.Pool,table string,id uuid.UUID,lock string) bool` 用NOWAIT且缺行/其他错误失败。workspace旧waitsFor的300ms超时与NOWAIT并非完全同义。

- [x] Step 1：读完整被改测试，区分“持锁可观测”与“实际等待/lock_timeout本身”性质，记录16处比较和3个消费者的处置；核原测试先过。
- [x] Step 2：仅把错误分类重复换成postgres.LockNotAvailable；只有在原断言不依赖等待时长/顺序时才用RowWaits，移除无读者的waitsFor和imports。仍证明超时的路径保留真实超时，不改生产事务行为。
- [x] Step 3：在server串行运行 `go test -p=1 -count=1 ./internal/platform/postgres/... ./internal/modules/identity/... ./internal/modules/workspace/... ./internal/modules/project/... ./internal/bootstrap`。检查原具体测试确实执行、非法错误/缺行不会算成功、锁被去掉会失败；合法分类专测不删。
- [x] Step 4：记录残留字面量各自理由与无重复helper，受影响交错用`-race -count=5`重验无40P01，提交 `test(M4/closeout): 统一历史锁错误断言并保留等待语义`。

## Task 3：服务端全量性质与二十六交错（约250–450行台账；补测按缺口）

**Files：** S/modules/{access,issue,project}及相关workspace/identity矩阵；S/archtest、platform/{jobs,outbound,postgres,httpserver/apitest}；bootstrap的interleaving/issue_*、purge*、registration测试。
**Interfaces：** 设计§3.5逐项→实际test/subtest→真实连接/锁/闸门→观察结果；D02–D07映射不能只填一个`make test`。

- [x] Step 1：逐operation核规则表/判定/矩阵、双向apitest公开码与project.state_in_use；核全部访客/创建者/离开身份、读列表与计数/动态/关注，拒绝路径在取锁前结束。
- [x] Step 2：按当前最终写路径重推全局锁序与gate资源；跨项目同时取闸门/锁顺序、邀请排序、关联归属、异步删除窗口/固定时钟/迟到purge边界均有真实观察。26项每一项指向具名test，不以文件名命中数充数。I21仅任务/数据库、I22仅页面、I16的PAT读取严格按设计适用范围，不补虚构版本。
- [x] Step 3：在server执行 `go test -p=1 -count=1 ./internal/archtest ./internal/modules/access/... ./internal/platform/jobs/... ./internal/platform/outbound/...`；再按26项映射构造准确包/锚定`-run`分批跑 `go test -p=1 -race -count=5`。保存选择清单和每项5次执行，零40P01；不把已设计的迟到清理重试案例当未经允许的新死锁。
- [x] Step 4：核清理catalog/grants/runtime、registrationRows实际river_job、InsertTx提交/回滚、两类漏worker变异、业务clock无固定余量；缺口先红再根因修复/复审。汇入至少15有效变异候选，记录D02–D07，提交测试或证据文稿。

## Task 4：故事增长、会话和门禁检查点（约250–450行）

**Files：** E/stories/issue、smoke/{s1*,s2*}、fixtures/{browser,mounts,clock,assert}；C/hooks、store/issue、各Phase已改会话hooks；不复制fixture。
**Interfaces：** D01/D04/D08/D14/D15；每个修改hook的`旧账户请求完成→无旧跳转/提示→新会话状态不变`具名测试。

- [x] Step 1：读设计22行的增长项、现有page/API测试和共享assert；标注I12打开读完整页、I16新鲜attempt、I21固定clock、删除M3轮询和拒绝侧效应等，不靠story题名代替断言。
- [x] Step 2：由full-M4变更清单找await后navigation/toast/theme/refresh/clipboard，逐hook映射账户切换后兑现测试；存在实际漏网先写最小用例并修，保持root权限代际唯一owner。
- [x] Step 3：执行相关vitest及默认F；S2四角色逐冷加载精确method/path/query/重复数，M4入口无旧API/无权请求，13模块级实例按P15具名账本复核；M6/M7自身页面按原归属记录。
- [x] Step 4：原scope通过且新缺口闭环，提交并push检查点；核远端准确HEAD和CI（已登录连接器可取日志/重跑），补D01/D08/D14/D15；非W11失败按实际根因处理。

## Task 5：C12 列表与筛选（约200–400行矩阵）

**Files：** closeout-browser-matrix.md；既有list-interactions、filter-archive-profile-interactions及I3–I7/I15/I19故事；playwright.m4-cleanup.config.ts。
**Interfaces：** spec§2.2前九组：项目列表/看板/日历/表格/quick-add/peek/筛选显示/归档/个人三关系；每组Cr/Fx/Wk都有路径细分。

- [x] Step 1：make build，记源码/bin/配置hash；`--list`核三个引擎完整非零，API建立数据。matrix先列待验格，禁止预填通过。
- [x] Step 2：三个引擎逐格操作分组/排序/筛选/显示、空/多页/拒绝、键盘右键、慢网拖动与拒绝回退、日期/周起始、快捷创建/peek关闭；核真实写入或不写。适用身份见P16表，S2四身份保全。
- [x] Step 3：每个代表控件及不同裁切祖先状态留成功截图，控制者看焦点轮廓、弹层锚点、背景隔离/回焦；Tab/ShiftTab/IME/Escape/Back和会话退役逐项记动作/实际结果；收网络/console/CSP，无任意忽略。
- [x] Step 4：局部缺陷先具名重现测试再修；重建受影响候选，重验相关引擎/格。记录当前证据不声称最终main已验，提交本组记录和必要修复。

## Task 6：C12 详情与编辑器（约250–500行矩阵/必要回归）

**Files：** 详情相关I1/I2/I8/I9/I10/I17/I20/I22、detail-interactions/editor-loading/cleanup-interactions；真实编辑器/弹层组件。
**Interfaces：** 编号/项目详情、创建更新、子项/关联/链接、删除、描述、表格行列菜单六组；角色与当前路由/权限取数合同不变。

- [x] Step 1：三引擎核编号大小写/错误地址/404/403，作者与访客权限；创建更新的默认字段/迁移id不脏、输入错误先说明、父子和关联排除、链接blocked/failed与元数据更新。
- [x] Step 2：核嵌套弹层单层Escape/IME/Tab隔离、忙碌不重复写、Back/旧触发器退役；删除说明子树与结果，拒绝不丢用户输入。每页用真实API/DB证明写结果。
- [x] Step 3：描述输入/粘贴/撤销重做、表行列菜单正确作用于被选行列、只读/销毁释放、草稿和peek外层隔离；en/zh-CN名称和轮廓，截图由控制者检查。I10危险dial证明来自Go，页面只证明用户结果，不混层。
- [x] Step 4：局部修复先红绿再三引擎受影响回归，更新本组矩阵版本；编排/模块合同需改变则先提交并上报，否则提交局部结果。

## Task 7：C12 协作、搜索与真实LAN（约250–500行）

**Files：** I11–I14/I16/I18/I20、collaboration-interactions、clipboard故事及shared消费者；closeout-browser-matrix.md。
**Interfaces：** 评论/表情/动态、历史、草稿、命令面板+top搜索、LAN复制、共享设置六组；C6–C11各旧证据与C12新记录分列。

- [x] Step 1：三引擎按角色操作评论新增/作者编辑/非作者拒绝/删除、表情开关、完整分页/时间线筛选，历史两页/查看/还原，草稿无project/迁移/存储payload发布与冲突、自动归档系统操作者。
- [x] Step 2：双搜索宿主即时query/迟到响应/工作区或账号切换、命令当前对象、标签多选/IME；所有await后提示/路由只属当前会话，Back时窗口/请求范围退役。
- [x] Step 3：真实LAN HTTP测十二复制入口，记录实际origin/isSecureContext/手势/返回值/可见提示，不强制有手势失败；设置/主题/成员/日期选择器消费者回归。控制者逐格检查截图、焦点/裁切和网络控制台。
- [x] Step 4：核21组×3细分完整，原125项/引擎加新用例的自动回归并集无漏/skip；未通过格保持待验。局部修复及受影响格复审后提交，D12标候选验收完成，main绑定仍待T14。

## Task 8：列表三规模与搜索（约100–300行）

**Files：** S/bootstrap/issue_list_scale_test.go、issue_list_scale_{seed,oracle,check}_test.go（以实际现有文件核路径）；S/modules/issue/adapter/postgres/search_performance_test.go及既有helpers。
**Interfaces：** TestIssueListScale原循环增20000；TestSearch100KPerformance保持13场景。精确输出沿原listEvidence/search证据结构，新增20k不能只复制结果目录。

- [x] Step 1：给规模选择清单增加20k的必要断言/静态校验，原10k/100k仍在；读fixture分布/独立oracle，先确认缺20k记录会拒绝完整性验收。测试大数据由显式env控制。
- [x] Step 2：先以原生产 SQL 扩规模枚举并测量，不改阈值。实测发现的局部问题按 spec C06 修复并复验：等待启动 purge 完成后才写历史墓碑样本；子分组 `cells` 固定物化边界，避免单格过滤下推后两路窗口结果的嵌套循环。保留所有原始失败和修复后样本；在server执行（M4_CLOSEOUT_EVIDENCE是根的绝对路径）：

~~~sh
NERVE_LIST_SCALE=1 NERVE_LIST_SCALE_EVIDENCE="$M4_CLOSEOUT_EVIDENCE/perf/list" go test -p=1 -count=1 -timeout=40m -v ./internal/bootstrap -run '^TestIssueListScale$'
NERVE_SEARCH_PERF=1 NERVE_SEARCH_PERF_EVIDENCE="$M4_CLOSEOUT_EVIDENCE/perf/search" go test -p=1 -count=1 -timeout=20m -v ./internal/modules/issue/adapter/postgres -run '^TestSearch100KPerformance$'
~~~

- [x] Step 3：核3×13列表和13搜索均实际运行、不为空或skip；1首样本+5warm、正确ID/顺序/total/continuation/空fallback，运行role/locale/连接设置。分别列module/request、SQL、EXPLAIN execution与计划，不混加不同口径。
- [x] Step 4：按spec判据对照§3.16/最终P5；异常先核分布/执行计划与负载后只对未解场景复测，所有样本保留。未解释的100k退化留D04未勾，架构级优化不自行新增索引/改分页合同。写perf表并提交。

## Task 9：删除、连接池与一次物理清理（约450–900行）

**Files：** 既有S/bootstrap/issue_scale*_test.go、issue_pool_test.go、purge_runtime_test.go、job_state_observer_test.go；新增purge_scale_test.go，按职责可拆purge_scale_fixture_test.go/purge_scale_evidence_test.go，各约400行内。
**Interfaces：** `func TestPurgeScale(t *testing.T)`，只有NERVE_PURGE_SCALE=1运行；NERVE_PURGE_SCALE_EVIDENCE输出绝对目录。复用`newSplitRoles`、`newIssueWorldOn`、`issueWorld.seedAtScale`、`observeJobStates`和真实Runner，不能直接调用purger冒充任务。

- [x] Step 1：定义20k旧项目约660k主体行、59天/活行/PAT对照和稀疏子表；准备阶段runtime API建立框架，隔离owner按既有scale seed补量。以独立count/fanout确认输入。停准备应用后固定年龄化，断言没有仍跑的准备任务。
- [x] Step 2：实现opt-in测量，用runtime登录启动真实观察应用。记录cutoff和启动后新job watermark，旧/新PAT及每张实际有数据表的前后count/快照；观察真实running/终态，必须是本次aging之后的新尝试。保存job id、queue、attempt、running/finalized及种数据/等待/工作各时间。无任务、空清理、删近期行或owner执行必须失败。
- [x] Step 3：新增测试先以漏运行/错误cutoff/错证据边界的有效负例证明不空过，再恢复。数据库套件串行，在server运行：

~~~sh
NERVE_MEASURE_SCALE=1 NERVE_SCALE_EVIDENCE="$M4_CLOSEOUT_EVIDENCE/perf/deletion" go test -p=1 -count=1 -timeout=40m -v ./internal/bootstrap -run '^TestTheDeletionsAtTheScaleTarget$'
go test -p=1 -count=1 -timeout=10m -v ./internal/bootstrap -run '^(TestChainedWorkItemWritesHoldOffNoWorkspaceWrite|TestAHotProjectHoldsUpNoReadOfAnother)$'
NERVE_PURGE_SCALE=1 NERVE_PURGE_SCALE_EVIDENCE="$M4_CLOSEOUT_EVIDENCE/perf/purge" go test -p=1 -count=1 -timeout=30m -v ./internal/bootstrap -run '^TestPurgeScale$'
~~~

- [x] Step 4：删除请求<15s且七job各一个running/attempt1/无snooze/error，连接池真10/K2/queue64和503，purge确实清掉旧范围保留近期/活行。清理无设计耗时SLA，实际成本全交M8；意外高风险先提交上报。完整F后提交并push检查点，记录D02/D07/D16。

## Task 10：整个M4深度审查与修复（约300–800行报告；产品按发现）

**Files：** full-M4范围所有业务/测试/契约文件，新增docs/reviews/re-NN-M4整体审核.md；既有18Phase review作来源。
**Interfaces：** file→完整阅读批次→相关测试/调用者→逻辑/集成/覆盖结论；findings带source、影响、严重度、根因分类和复审状态。

- [ ] Step 1：调用deep-review，使用固定全期基线；按≤15业务文件分批，完整读取当前每个文件及相关测试、上下游调用/事件/依赖图。重复共享文件读一次但全覆盖有清单；删除文件核最后读者。接口/领域/domain纯净、adapter清洗、root权限owner、任务事务、归档/访客/会话是重点。
- [ ] Step 2：逐三视角记发现；局部先最小具体红例→根因修复→聚焦绿→受影响回归。结构/跨模块责任或意外高风险不能通过增加转接层修，先提交完成部分和建议后等决定。
- [ ] Step 3：从全期有效反证抽至少15个不同关键性质，逐核变异补丁/失败原因/恢复hash与当前等价；若受修改影响则重跑。错误阶段失败不计。按改动重跑对应交错-race，不能改Timeout/忽略来消失败。
- [ ] Step 4：全文复审修复及调用者；更新C12/性能输入等价，影响哪格/哪测就重验。报告清楚全期范围/文件数、发现闭环和仍在后续域的已批准限制，提交。

## Task 11：同步表、差异、dead与lint终态（约400–800行）

**Files：** M4-design.md、v0-design.md（此时不改9.4完成）、M2/M3设计涉及段、plane-diff.md、frontend-changes.md、README.md；closeout-dead-members.md（新增）、各包cap（只有实际告警改变时）。
**Interfaces：** §3.23当前44条、§4.17当前45条，每行原文键→实际目标段→事实证据→核对结论；不靠grep关键字就记同步。

- [ ] Step 1：逐同步行看目标完整段落、逐差异看行为/表列/索引；README§8.8安全/部署/前端、M4前端改动、旧阶段历史状态标识准确。集成项如9.4仍明确待T14。
- [ ] Step 2：按M1附录原版deadsym/domains/lib/program和P16ledger，在当前typegen输入跑完整AST；保存脚本hash、输入、raw/global及M4行。逐当前行记录删除或真实读者，追溯历史52行；不能改DOMAINS/过滤测试或把453/353历史数当终数。
- [ ] Step 3：完整lint按原14范围实数，M4域及全M4改动文件零诊断、全仓no-shadow=0且error；cap实际相等只降、无屏蔽残留。`make lint-web`/`make knip`、关键词及13实例精确ledger验证；动态/第三方/测试读者逐行可查。
- [ ] Step 4：任何删除代码先读最后消费者并补适当回归；这一步后如输入变了更新影响证据。写终数/规则hash/误报理由及44/45映射，提交，不提前勾总体完成。

## Task 12：八输入和四输出交接（约500–900行）

**Files：** D/handoffs/{M0-P4-pg-trgm,M1-P2-trim-content,M1-P3-trim-platform,M1-P4-router-native,M1-P5-brand,M1-closeout,M2-closeout,M3-closeout}.md；新增docs/v0/{M5-files,M6-cycles-modules,M7-collaboration,M8-open-release}/handoffs/M4-closeout.md。
**Interfaces：** input文件+原条款→Phase review+验收证据或outgoing文件+具名子项+完成条件；输出status=open，输入所有条款有结论后status=closed。

- [ ] Step 1：用§13.1逐原条款及此前各Phase spec§5/review§6核落点；八文件不只按顶层标题算条数。实际已完成项链接当前/历史等价证据；转交写清原因/代价/后续何时验，保留原历史处理段。
- [ ] Step 2：写M5附件/同事务迁移/图片identity/删除图/事件决定，M6新列表引擎/自身页面未接通的实际75请求等证据/条件/最后读者，M7收集箱枚举与系统身份/通知发出时权限/视图/跨项目/旧DTO与版本最后读者，M8事件列表/出站/部署/性能/锁等待代价；照spec§2.4并逐个原交接防漏。
- [ ] Step 3：M2-closeout§1的原标题与全部内容逐字复制到M5，再另写已提供NewInserter和未来CLI组合职责；M0-P4“部署要求”整段原样复制到M8。以读取源文本和目标引用块的逐字比较留收据；修历史相对链接时原文块不改，外加新可点击链接。
- [ ] Step 4：核新表扩furnishIssueRows、订阅project/workspace.deleted、物理清理owner，M6/M7恢复列表期限/不恢复旧引擎，M1-P3/P5来源/文案/系统创建者细节完整。原条款全有结论才改8个closed；检查四输出存在且open、双向链接有效。提交交接结果，D10才可签实施完成。

## Task 13：最终验收与收尾分支整审（约400–700行）

**Files：** closeout-review.md、closeout-acceptance.md、spec/plan；新增docs/reviews/re-NN-M4收尾整体审核.md；M4§15状态与04-next-steps当前接续。
**Interfaces：** review沿P3七节：方式/验收/决定/发现修复/plan差异/交接/限制；十八D项各有完整证据而非只“完成”。

- [ ] Step 1：固定最终产品候选，核所有C12格/性能/变异/各门禁与当前输入一致；新产品变化才重跑受影响检查，再串行F收口。数实际全部用例与skip原因，保存输入/输出manifest。
- [ ] Step 2：deep-review closeout基线至最终候选完整文件，主控三视角按任务间集成复审；至少15有效反证与-race证据可核，修复后同范围复审。全M4报告与此报告是两个范围，不互相冒充。
- [ ] Step 3：十八D逐条给命令/raw/路径/局限/后置集成项；各Phase估计与实际、新增/删除/生成/文档分开，写D17。更新spec验收和plan已执行步骤，但D18/最终main绑定仍待集成，M4§14复合项不提前全勾、总体9.4继续进行中。
- [ ] Step 4：提交终稿并push准确HEAD，等三个分支CI作业success；核本地/远端相同40位SHA，最终外置final-delivery.json记run/attempt/jobs。阶段状态“实施与终审完成、集成待办”，保留分支/worktree，通知负责人，不自动合并。

## Task 14：集成准备与获准后的最终签结（约150–350行文档/收据）

**Files：** M4-design.md§14/§15、v0-design.md§9.4、closeout-review、04-next-steps；ignored最终集成目录和两个证据根归档manifest。
**Interfaces：** 两个mainCI事实：验收内容合并M1成功→文档完成标记候选C→最终main合并M2成功。最后收据外置，CI不写回Git形成自引用。

- [ ] Step 1：准备分支名/基线/最终SHA、spec/plan/两个整审/交接索引、完整F和分支CI、无未解项、归档清单；确认root main未动。**这里完成可评审的集成包并等待负责人未来授权。**
- [ ] Step 2（获准后）：fetch，核主干漂移/祖先/clean、精确候选CI；按`--no-ff`合入验收分支，系统git推main，核双parent/tree与准确main CI。真实失败在新隔离分支修复；W11重跑仍败按用户约定停报。CI未绿不执行最后状态标记或清理。
- [ ] Step 3（同次收尾集成授权内）：从已绿main新建`codex/m4-closeout-record`及约定路径worktree，只写事实：M4§14逐项勾并附证据，§15/头部及总体9.4标已完成，补C12等价到main/四交接与审查链接。无产品修改，静态深审/链接/18条coverage通过，提交推送并核分支CI；`--no-ff`合入main，再核最终main三作业全绿。若期间产品改动，退回相应验收，不能靠文档提交遮住。
- [ ] Step 4：最终外置收据绑定两个mainSHA/CI、completion提交及tree等价。归档closeout实施根、主目录准备根、record分支证据和浏览器/性能全部raw；源/目标逐hash/byte/mode/成员清单比对，unknown=0。确认origin/main包含两分支且remote-only=0，再删远端分支、正常worktree remove、branch -d；不force。归档后复验，root main clean且与origin/最终CI同SHA，才宣告M4全部完成。

## 预检与交付边界

预检时上述步骤全部未执行；静态预检只核每D对应Task、每C12组归属、数据/计时边界、源码帮手与后续交接。当前实施所需环境、真实结果和 phase/status 变更由 T1 以后逐项记录，不能引用预检作为运行成功的证据。

预检方法与发现见 [closeout-preflight](../reviews/closeout-preflight.md)；原始盘点在主目录 `.superpowers/sdd/M4-closeout-preparation-20261101/`。实施证据与进度见 [验收台账](../reviews/closeout-acceptance.md)。
