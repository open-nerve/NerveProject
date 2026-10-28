# M4/P9 物理清理、版本保留与异步删除评审记录

> **当前接续（2026-10-28）：P9 已完成集成与归档清理。** 准确 main、CI 和新证据根见§14。§1–13及其待授权/待CI/不合并语句保留各自时点的历史，不覆盖本节最新接续。

日期：2026-10-27。分支 `codex/m4-p9`，基线 `6b9d08d4821f1423c7bb9795557727c206b7880f`。**准备与独立静态预检 Approved；2项局部遗漏已修订复审，开放 High/Medium/Low 均为0；未实现 P9。** 配套：[spec](../specs/P9-purge.md)、[plan](../plans/P9-purge.md)、[deep-review报告](../../../reviews/re-08-M4P9预检审核.md)。

## 1. 评审方式

沿 [02 方法](../../claude_to_gpt/02-method.md) §3 允许的紧凑方法，由唯一作者细化已批准设计，独立审者核查逻辑正确性、集成安全与测试覆盖。准备工作不做完整产品原型或逐 Task 重放，静态审查不替代实施期的真实 SQL、锁交错、River 调度、变异和规模证据。

准备原件在 P9 worktree 的 ignored `.superpowers/sdd/P9-purge/`，下文简称 E9。输入 `design-inventory.md`、`code-inventory.md` 已核既定协议、前序交接和当前源码；作者、独立审者及主控分别记录范围和结论。唯一作者冻结后，两位独立审者均全文读完 v1 spec/plan，再分别追踪删除/锁/事件和清理/保留/调度/故事/规模；主控核因果链、覆盖清单与准确指纹。删除侧101文件/15,568行、清理侧106文件/13,367行有重叠，不相加为独立文件总数；节选另列，不冒称全文。主控已核两份全文清单每项行数、字节和SHA匹配，见 `preflight-coverage-verification.json`。

## 2. 验收标准逐条核对

P8 已按授权以 `--no-ff` 合并为上述基线，[main CI 40520609987](https://github.com/open-nerve/NerveProject/actions/runs/40520609987) 的 server/web/e2e 三任务全部成功。1,598 文件、37,552,117 字节及 1 个空证据目录归档校验后，P8 本地/远端分支和 worktree 已清理，清理后再核哈希通过。完整回执见 [P8 review](P8-drafts-search-review.md) 第 9 节；这些证据证明 P9 基线状态，不证明 P9 产品通过验收。

P9 所有产品验收均未完成，逐条责任以 spec §4 的A1–A13和 plan 的16 Task为准。准备期实际运行 `node tools/keywords.mjs`（72规则、23例外、零命中）和 `go -C server test -count=1 ./internal/archtest`，均 exit0；前后输入状态一致，原始日志哈希已核，分别见 E9 的 `checks/preparation-keywords/`、`checks/preparation-architecture/`。首稿文档检查144处本地链接/锚点无新增失败，`git diff --check`通过；4个既有M5–M8最终closeout目标留待M4收尾，不将其记作新增失败。最终回执后的检查由 `checks/preparation-docs-final/`、`checks/preparation-diff-final/` 和 `final-preparation-receipt.json` 绑定准确文件身份。没有执行P9迁移、生成、产品测试、性能测量或完整本地CI。

## 3. 准备期决定

沿用已批准的四模块清理者、单个清理任务、版本保留项目 N 锁、事务内删除事件及按项目投递；不引入新的模块协议、持久化事件表或依赖。具体局部细化由 spec §3 固定，独立预检发现会在 §4 闭环。账户不做物理删除，旧 P2 空创建者权限边界另以测试数据准备验证；不据历史措辞增加账户清理功能。

## 4. 预检发现与处理

首稿 v1 提交为 `26f004057f3a42b0f880257965f9693f1bb68ec5`，包含P8集成回填与P9准备，phase首次改为M4/P9，72规则/23例外未变。删除侧初审零发现；清理侧2项Medium，经主控核实现有源码后修订，原审者定向复审均为ADDRESSED。没有需要重新裁定的架构问题。

| 原始ID / 级别 | 根因与修订 | 闭环与实施期验证 |
|---|---|---|
| P9-PURGE-M1 / Medium | T1新增23个索引却遗漏既有精确全量schema/index期望。spec/plan补四个migration测试文件、23个`iw`项和draft精确indexdef；不放宽全量相等，T3新目录测试仍保留 | `purge-rereview-v2.md`确认ADDRESSED；未来T1实际执行`TestConstraintAndIndexNames`、`TestDraftSchema`红绿及migrations回归 |
| P9-PURGE-M2 / Medium | T10异步切换后，runtime无项目草稿测试仍在204后立即Scan审计bool，合法未完成窗口会失败。补有限等待准确workspace/At/By的真实workspace_drafts任务completed，保留实际runtime角色、两个草稿三审计字段和邻workspace整行不变 | 同一复审确认ADDRESSED；未来T10精确选择器命中`TestRuntimeRoleDeletesDraftsInWorkspaceWithoutProjects`，不可用owner直调、睡眠或宽松断言替代 |

| 冻结稿 | spec SHA-256 | plan SHA-256 |
|---|---|---|
| v1：提交26f00405，337/259行 | `b4609d278b2a05a4f573881e55aa4ebec18ba6236aea3fba5b998da39b766763` | `ab299bc8ac3c80a16fae77b88ed9f3295044bd4cf133908013c1db8739a626e6` |
| v2：26f00405上两项修订，341/261行 | `17dc388fe4d74052d62a1427a94926e9417224bf8a1490371caa659a42240129` | `879d2a20d760fa80e6d4d5e05ffbd084f55b8948814c82e4e25e1fffe51069c9` |

原始报告/清单为 `deletion-preflight.md`、`deletion-coverage.md`、`purge-preflight.md`、`purge-coverage.md`；作者修订记录 `author-revision-v2.md`，闭环为 `purge-rereview-v2.md`。v1/v2原字节分别保存在 `frozen-v1/`、`frozen-v2/`，`revision-v1-v2.diff`为14增/8删的两文件修订。冻结前已吸收的事件登记顺序、eitherOf两端预检、W3两处即时查询及周期任务防重叠等细化不重复计作这2项发现。v2之后只改预检状态与证据入口，不改实施协议/任务；最终回执定向核验另见 `final-receipt-review.md`，不将其冒充再做一次产品整审。

## 5. 与计划方法的差异

本轮只准备可执行 spec、任务计划和静态预检。没有生成产品实现、预造生成文件指纹或虚构测试红绿、并发/性能数字。实施时应逐 Task 保存真实命令、选择器命中数、输出和恢复证据；已有阶段的运行结果只能证明其各自范围。

## 6. 移交事项

本轮暂不实现 P9，也不合并 P9。预检完成后由负责人安排启动 T1；P10a–P16、M4 总体完成标准、输入 handoff 总状态和 M5–M8 最终交接均保持原阶段归属。P9 实施结束仍须完整整分支 deep-review、局部问题修复及复审，再按负责人的集成安排处理。

## 7. 已知限制

静态预检不能证明运行时锁序、真实 worker 的重试/snooze、索引和 SQL 执行效果或五个目标规模项目的删除性能；计划必须保留相应实施门禁。真正的架构回归、新跨模块协同方案或意外高风险，按负责人纪律先提交已完成工作再上报。

## 8. 实施验收与独立整审（2026-10-28）

T1–T15已实施、逐Task独立要求/质量审查并由主控核原件；全分支整审和后续局部修复复审已闭环，产品HEAD为 `e003e96625ba51264d3aeb77729729d53a770103`。本次T16同步文档及一处普通SQL注释，不新增产品行为运行。准确e003最终七项F已通过，见下列主控收据；准确产品分支CI已由主控核绿，两席文档独评及修复复审已闭环，最终文档HEAD CI仍待核；产品检查点不能代替文档交付门禁。

| 身份 | 准确范围与结论 |
|---|---|
| prepared/static | §1–7及re-08是2026-10-27准备/独立静态预检；不证明产品运行 |
| runtime / Task | T1–15各自report/review/controller-verification及raw；命令、源码SHA/dirty字节、RUN/skip/exit、恢复hash分别保留 |
| whole-branch review | `6b9d08d4821f1423c7bb9795557727c206b7880f..7881127ac0e6a2884a5ea1252cff8a7707ffc697`，238文件60363行，含生成物和M4设计2721行；主控核独立读者完整范围并集，不称主控亲自重读全部 |
| scoped fix review | 788→d25363b4480af57e5e29e2ee26088442e1352bf8关闭fixture-M1/L-DOC-1/I-D1；d253→e003e96625ba51264d3aeb77729729d53a770103关闭FF-CLEAN-1。4项均ADDRESSED，无开放发现/架构待决 |
| final product F | 准确e003七项串行F均exit0；Go51包+16无test包（非verbose逐test skip未知），web138files/1165Vitest+5Node且0cache/skip，E2E163=161Cr+2Fx/0skip；[主控最终F收据](../../../../.superpowers/sdd/P9-purge/final-checkpoint-controller-receipt.json) |
| final product branch CI | 准确产品e003的CI40630029402 attempt1已由主控核验server/web/e2e全部success，见[产品CI收据](../../../../.superpowers/sdd/P9-purge/final-checkpoint-ci-receipt.json)；不替代本轮文档提交或未来main CI |
| T16 static/doc review | 首次31文档静态核对见[静态收据](../../../../.superpowers/sdd/P9-purge/task-16-static-summary.json)；两席文档独评及[CORE-1/2修复复审](../../../../.superpowers/sdd/P9-purge/task-16-fix1-review.md)已闭环，见[主控核证](../../../../.superpowers/sdd/P9-purge/task-16-fix1-controller-verification.json)；最终状态/注释小diff仍交主控限定核验，最终文档HEAD及其准确分支CI由主控[交付收据](../../../../.superpowers/sdd/P9-purge/delivery-final-receipt.json)记录；该可变指针当前待核，须按准确HEAD核验，不代替未授权的main集成/CI |
| delivery HEAD | 首次文档7f5625bf身份保留在原 `task-16-final-receipt.json`；最终交付HEAD与分支CI见主控[交付收据](../../../../.superpowers/sdd/P9-purge/delivery-final-receipt.json)，不倒填产品或规模测量HEAD |
| main CI / integration | 未授权合并；无P9 main CI结论，分支/worktree及ignored证据保留 |

实施证据入口：[A1–A13与24生产代表索引](../../../../.superpowers/sdd/P9-purge/task-15-audit-evidence.md)；[逐函数/批写审计](../../../../.superpowers/sdd/P9-purge/task-15-batch-audit.md)；[独立整审闭环收据](../../../../.superpowers/sdd/P9-purge/final-review-closure-e003e966.json)。各项A的命令/raw位置由前者逐条展开，不将PASS摘要当原件，不把作者审计当独立整审。

| A | 已取得的具体证明与边界 |
|---|---|
| A1 | 00030–00033的23索引、模块schema往返、23清理者与真实入边目录；T15真实迁移CASCADE→RESTRICT被guard拒绝并恢复，不把probe算生产变异 |
| A2 | 四模块61d删、59d/精确cutoff/活行留；子先父、id键集、pool独立批提交/错误已成功计数、runtime角色。父CASCADE可能带走较新子行；不删users/profiles，session另管 |
| A3 | 一次业务Clock/固定模块序、四种periodic真实completed、请求kind/周期worker缺失启动拒绝；purge active-state唯一，完成后可新建任务 |
| A4 | 活项目/issue/version按last_saved_at DESC,id DESC留20，归档仍参与，三层墓碑不碰；每项目N+JobTx和真实描述写/删除两顺序，不虚称version DELETE物理id锁序 |
| A5 | project/workspace原事务M3写/发布/InsertTx原子回滚，本次活项目集合与零项目draft任务、原At/By；两同步删除能力及CLI过渡依赖移除，状态/标签能力保留 |
| A6 | 13表短批/幂等，relation双向先锁后判归属、sameAt/不同actor/Old；After按Candidates而非Written推进，零Written仍继续 |
| A7 | 37=15读+22写完整窗口与契约双向相等；未知资源逐值对照、游标/计数/顺序/422、历史引用、OwnOnly与NULL creator；18普通+3草稿CRUD+1发布锁形状不放宽 |
| A8 | 旧交错2/3/11/16/23与24/25、双UUID次序、真实PID/XID；整审fixture-M1补第二请求本人等待证明，最终共享helper28族205、普通38/race190通过，普通组无40P01 |
| A9 | 真Runner多轮snooze/attempt1、部分提交后23514与原job重试、真实25 discarded/耗尽日志；先purge迟到零候选与两victim原40P01重试收敛分列 |
| A10 | I21/I16最终2/2、P4/W2/W3 8/8；API造齐目录、同库单nerve换代+1464h、fresh attempted微秒门与crossing反例、保存ID直查旧链消失/recent/live整行留；Old/邻居原戳保留 |
| A11 | 六项目每个660000行/1KB动态、7job一次running无snooze/错误、正常15s请求deadline、两类正确存活探针160样本/双处理中窗口，详见spec附录B |
| A12 | 24真实生产代表、T4/T8/T12/T15检查点与独立整审/两次修复复审已取得；本产品最终F/branch CI已核绿，文档提交CI待核，见上表，未来main CI仍独立待授权 |
| A13 | 本次同步M4§3.23四行/README/上级设计/契约语义及输入回执，正式静态验证、两席文档独评及修复复审已闭环，见本节收据和§12接续；不关闭M4总体或后续阶段 |

24代表为M1/M2/M4/M6/M7/M8/M9/M11b/M12-created/M15/M17-current/M18/M21/M23-kind/M25/M27/M31-final/M32-final/M34/M36/M37/M39/M40/T15-FK。M29/M30/M38/M41和catalog probe单列测试层，不计入生产数量；同一性质早轮/final不双计。M34前两轮存活、M36正61d超时/负61d明确红、M37旧超时/后明确红均保留。

## 9. 整审发现、修复与局部裁定

| 发现/原级别 | 根因、修复与独立结论 | 验证与边界 |
|---|---|---|
| fixture-M1 / Major / Local test-evidence | 宽projects waiter可误认真实trim，使第二请求尚未到锁便放第一gate；d253绑定两请求及第二PID/事务/实际SQL/blocker链，原响应/零写/时钟不变；独评ADDRESSED | 同一回归旧helper真实红→身份修复绿→宽谓词变异红→恢复绿；28族205、普通38/race190、回归race1。不是生产锁错误，不以F绿替代等待身份 |
| L-DOC-1 / Minor | module入口误述同步Cascade、fake误称P9删账户；d253改英文说明为当前职责/NULL语义，独评ADDRESSED | 纯注释，无生产行为改动 |
| I-D1 / Minor | relation契约漏任一端工作项/项目已删404；d253同步源说明与正常make gen两生成副本，独评ADDRESSED | 仍允许存活但不可见另一端按任意可编辑端操作，不能误写两端都必须可见；81生成输出/112实际输入及gen-check留原件 |
| FF-CLEAN-1 / Minor | 新回归afterWait前失败时users暂停锁释放晚于第二join；e003把释放放第一请求返回前defer，独评ADDRESSED，无新问题 | 最终test count1/race1/wide性质红→恢复绿；异常顺序靠明确控制流复审，未注入Fatal。不重跑未改205/38/190与规模，最终F另接 |

原三项修复证据：[d253作者报告](../../../../.superpowers/sdd/P9-purge/final-fix-report.md)、[d253独立复审](../../../../.superpowers/sdd/P9-purge/final-fix-review.md)；续补：[e003续补报告](../../../../.superpowers/sdd/P9-purge/final-fix-cleanup-report.md)、[e003独立复审](../../../../.superpowers/sdd/P9-purge/final-fix-cleanup-review.md)。d253的19份旧raw/收据保持原样，cleanup独立10份原件/源码绑定；不把续补新test字节套用旧回归hash。

局部裁定按当时批准范围记录如下。完整原话导航见[26条历史裁定导航](../../../../.superpowers/sdd/P9-purge/final-rulings-inventory.json)，后续追加见[执行台账](../../../../.superpowers/sdd/P9-purge/progress.md)；下表归并同根因而保留各原任务入口、理由和错了的代价/覆盖补偿，不重开已批准架构。

| 原裁定来源 | 结论与理由 | 代价/覆盖补偿 |
|---|---|---|
| T1配置/schema；T2空白修正 | 六字段全量期望及33→29回退/OID不放宽；两SQL EOF空白修正只生成+聚焦 | 错了会损配置/迁移保障；保留原失败、全场景与准确最终字节，不将旧回归假作新全包 |
| T3非purge FK probe | 先在业务schema核旧keysTo，再隔离加probe验全部入边；不扩comment_reactions例外 | 隔离不准会隐藏新业务表，审查核时序/非空/反例；T15另补真实迁移FK生产链 |
| T4 runtime文件/testConfig | 独立purge_runtime_test复用splitRoles，testConfig补合法默认；原任务保留角色/模块序 | 不把owner观察当runtime执行；scope变化同步spec/plan并独评 |
| T4 retryable窗口 | River首失败可available，使用同job两次真实失败进入retryable跨两个周期后第三次完成 | 错误状态假设会将timeout误计性质红；留available/retryable/attempt与原失败 |
| T5私有pool-factory | 默认仍postgres.NewPool；test按现有RequestID核真实网络请求，保留JWT/worker/池/200burst | 小范围生产组合变化须独评及默认路径回归，不放宽门/锁断言 |
| T5/T8实际请求锁归属 | test Tx包装取得请求PID/XID；允许锁证明用真实moduleRoute，拒绝与兼容故事保留full app；网络项目写另用RequestID tracer | 调用边界变化明确留证；tuple等待不冒行锁，混合fixture/request只计仍存在owner，不称未知历史peer为trim |
| T8准备锁获取 | 已盘59调用只要求返回时持锁；holding/lockVersionPage共享最多5s有界获取，NOWAIT性质探针不改 | 遗留锁从即时失败变有界失败；受控持N/取消/短期限/连接归还及原强度/顺序反例补偿 |
| T7分页与trim | SHARE保护>20版本的分页前置，SQL seed替代100次PATCH，真实HTTP分页容量/顺序/cursor/detail与HTTP保存测试保留 | 不停worker/延周期/减行；清理先放锁，有真实trim反例；原全包分页失败保留，仅聚焦接续、下一F兜底 |
| T10 catalog清单 | 纯目录读器只复用，不做无意义touch；caller承担等待，修正spec/plan清单并独立文档复核 | 文档精度修正不冒生产改变；不为清单重跑无关套件 |
| T10 W3/M27–29 | Old真实未完成屏障后删workspace，后按Old戳核；M27按实际漏deleted谓词记录，零项目draft不漏 | 仅API业务前置、隔离只读行锁/有界释放，不伪称另一种变异或放宽原审计 |
| T10 E2E lint归属 | issueOwnedTable统一到catalog、纯期望模块级；M3先即时，issue后有限poll | 不豁免未知表/旧行/邻居；最终types/lint/故事及M29最终字节红绿接续，不重跑无关Go包 |
| T11既有接口形状 | search不新增total/cursor，member不支持sub_group_by；支持入口证明分页/子组，非法仍精确422 | 不扩API也不丢性质；37完整窗口/目录双向相等，fixture假设失败不计生产红 |
| T12世界与Runner职责 | prepareIssueWorld复用同一已启动app数据准备；单独lifecycle fixture组合真实Runner/原错观察 | 保留普通/Observed、API/store顺序与cleanup；无新生产协议，最终共享调用族/独评接续 |
| T12目录/关系归属 | 25的12自有表逐项目非空，relation按双端逻辑集合/实际物理归属核戳；link用已批准store catalog seed | 不伪造物理双归属、不加未知表豁免；明确非全API，T13 E2E仍全API |
| T12 M34合法计划反例 | 隔离夹具非HOT反heap且最终业务字节不变，test连接限定合法Nested Loop，原ORDER BY恢复后再变异 | 前两存活轮保留不计红；不改生产planner/SQL守卫，以真实PID/持锁区分顺序 |
| T13文件与证据身份 | 只改I21/共享周期门/I16，既有P/W断言复用；M38 TS门与M39 SQL→gen→binary分别计 | 不机械touch旧清单、不混clean/dirty binary；最终2/2与8/8、源码绑定承接lint后实际字节 |
| T14影响范围 | scale三文件未改共享helper，五族影响+完整非skip规模足够，下一完整F由root接续 | 未覆盖旁路由F兜底；若新变化/真实失败/风险再扩大，不把聚焦拼成全包 |
| T15生产代表/约定五 | 测试层反例不混生产，真实FK另补；purge候选排序+pool单DELETE，trim项目N，短批/relation显式先锁 | 不虚称DELETE物理锁序或全系统无死锁；取消/commit未知结果/rollback优先原协议原样记录 |
| 阶段执行与证据保留 | 实施授权覆盖旧仅准备状态；F在T4/T8/T12/T15/最终修复，不逐Task重跑；ignored证据保留到授权集成/mainCI/归档 | 原件及失败不删，局部修复不中途因通用评审波次停下；真实架构回归才上报 |
| FF-CLEAN-1续补 | 用户已授权局部问题全部修复，不因通用单波次建议留下Minor；仅一回归闭包8行差异 | count1/race1/wide红恢复与控制流独评，无Fatal注入/无未改大套重跑，root最终F接续 |
| Shell fallback归类 | 锁定Testcontainers external成功后internal errShellNotFound被上游接受并return nil；保留日志，不修产品/依赖/Docker | 不声称历史镜像缺sh成因已查明；新启动不ready/exit失败必须独立判断，旧分类不能放行 |

## 10. 批写、锁图及失败边界

逐函数原件见§8批写审计。23条物理purge各用pool隐式单语句事务，成功批读完RETURNING/rows.Err才计数，错误保已成功数；候选ORDER BY不是DELETE实际锁序承诺。trim逐未删项目N+JobTx，提交后计数，等待后项目消失则跳过；活版本集合与软删purge分开。

软删实际为12条普通project批+1条workspace_drafts批及独立relation协议，对应issue侧13表。普通批候选id升序N，只更新锁定live行；关系先锁双方向候选再无锁读另一项目DeletedAt，不额外锁对端项目。普通deleteIssue跳过已删另一端，job只让给严格更早删除端，同At按先持关系锁处理。每批成功提交后推进Candidates/After、累计Written；不持久化cursor、不以零Written终止。

请求仍18普通P(N)、草稿3CRUD W(S)→P(S)→D(U)、publish W(S)→P(N)→D(U)；拒绝先于gate/锁、锁下重读重判、最后锁后时钟未放宽。M3删除原W/P锁与邀请同步保留，同事务事件投递失败使请求全部回滚；未来worker各批不是该请求事务的一部分。

业务ctx沿SQL；COMMIT/ROLLBACK仍用WithoutCancel与独立期限，commit无答复是结果不确定，取消不保证已结束语句一定撤回；rollback基础设施失败优先的旧协议不改。JobTx保原PG错，不映射HTTP503。普通race结论与>60d父CASCADE/迟到软删的40P01重试例外分列，不写绝无死锁或任意阻塞都在budget+5s内结束。

## 11. 检查点、真实失败和测量身份

T4/T8/T12/T15 F沿各Task/root收据保留。T12准确 `2aa26ab9b14383d1415a85fa80a3f470a4c8f1cd` 的 [CI 40610708261](https://github.com/open-nerve/NerveProject/actions/runs/40610708261) attempt2三job成功；attempt1 W11失败原输出保留，重跑成功不等于修明W11根因。T15准确 `7881127ac0e6a2884a5ea1252cff8a7707ffc697` 的 [CI 40623599731](https://github.com/open-nerve/NerveProject/actions/runs/40623599731) attempt1三job成功。

T15七项F：Go51包通过+16无test包，非verbose不报逐test skip；Vitest138文件1165项及Node5项，web非缓存/0skip；E2E163=161Chromium+2Firefox/0skip，WebKit未选，C12归M4收尾。准确产品e003的最终F/CI仍以上表单独收据为准；不能借旧788结论覆盖修复后的HEAD。

T14测量为 `4c7379f1d4dd53a8af932f2ae148ae2f9d07f859` 加已归档dirty scale字节，后续 `d26ae92686d4c4dde798bc3371ce8f287e7eeaf6` 绑定该源码；后续文档/局部helper与注释修复不改性能路径，也未重测。六项目396万行、160样本、7job、双窗口及全部机器/DB/池/缓存/observer参数见 [spec附录B](../specs/P9-purge.md)。workspace负载PATCH最大525.226791ms仍计入；不移植旧SP2/SP4数字或承诺跨环境毫秒上限。

保留真实失败分类：T9/T10全包早轮失败按其已审最终聚焦接续；T7原全包分页失败与之后fixture修正不拼成新全包；T12存活变异/超时、T13零匹配/lint/证据脚本、T14编译/源码定位失败分别记录，均不冒性质红。Shell fallback保留原输出，未称所有日志pristine。当前产品/整审发现无开放项；这些历史记录不等于仍有未修产品缺陷。

## 12. 文档、后端交接与未关闭责任

M4§3.23四行分别落到总体§5.5/6.7清理、§5.5/6.3异步、M3§3.3/3.6/6.2/6.5/13.2、plane-diff；README六配置/失败窗口与契约实际行为逐项核对。两输入handoff仅追加P9后端结果、status仍open；P1a–P8各spec§5/review§6追加回执。P2旧“物理删除账户后NULL creator”只承接空创建者权限组合，账户不purge；T11用API造issue后隔离置NULL、OwnOnly/adminGuest逐值404、member不继承AllowCreator为403、admin可删。

M4总体§14/总体§9.4、P10–P16页面与收尾仍开放；不创建M5–M8最终M4-closeout。M5首个CLI投递/附件内容标识，M6新增表/事件/订阅与归档条件，M7通知发送时权限重判/收集箱，M8Webhook/网络/locale/池/多进程/性能仍按M4§13.2归属。四个既有未来closeout链接是已知待收尾，不报P9新增链接失败。

首次31文档静态核对见[静态收据](../../../../.superpowers/sdd/P9-purge/task-16-static-summary.json)；两席文档独评及[CORE-1/2修复复审](../../../../.superpowers/sdd/P9-purge/task-16-fix1-review.md)已闭环，见[主控核证](../../../../.superpowers/sdd/P9-purge/task-16-fix1-controller-verification.json)；最终状态/注释小diff仍交主控限定核验，最终文档HEAD及其准确分支CI由主控[交付收据](../../../../.superpowers/sdd/P9-purge/delivery-final-receipt.json)记录；该可变指针当前待核，须按准确HEAD核验，不代替未授权的main集成/CI

只文档修改不谎称新行为运行，不重跑未受影响规模。本轮不合并；保留分支/worktree及ignored证据。未来获负责人明确授权才--no-ff，另核准确main SHA CI及归档哈希后清理。


### T16文档独评历史接续：32e52e38提交时局部修复待复审（2026-10-28）

准确文档范围e003..7f5625bf的[18回执审查](../../../../.superpowers/sdd/P9-purge/task-16-receipts-review.md)为Approved/0发现；[核心13文件审查](../../../../.superpowers/sdd/P9-purge/task-16-core-review.md)发现T16-CORE-1/2两项Local/Minor，整体文档尚未Approved。CORE-1修正M3三处DeleteWorkspaceProjects旧projectIDs入参，锁返回的ID集合仍由workspace用例用于事件；CORE-2修正M4两处/plane-diff一处state_id CASCADE无条件承诺，区分单独删状态守卫与父级异步迟到时过期状态/父行回收活标记issue。

本轮只修文档并定向静态核对端口/调用、StateInUse、状态清理顺序及FK，未改产品/测试，未重新运行F或规模；修复独立复审、文档提交CI仍待主控接续，不提前关闭A13或阶段状态。产品4项发现闭环与e003 F/CI、7f原文档证据保持各自身份。


### 当前终审闭环接续（2026-10-28）

M4/P9实施与终审完成、集成待办。两席文档独评及[CORE-1/2修复复审](../../../../.superpowers/sdd/P9-purge/task-16-fix1-review.md)已闭环，见[主控核证](../../../../.superpowers/sdd/P9-purge/task-16-fix1-controller-verification.json)；最终状态/注释小diff仍交主控限定核验，最终文档HEAD及其准确分支CI由主控[交付收据](../../../../.superpowers/sdd/P9-purge/delivery-final-receipt.json)记录；该可变指针当前待核，须按准确HEAD核验，不代替未授权的main集成/CI。上段32e52e38待复审文字保留为当时历史；CORE-1/2现均ADDRESSED且无新破坏。主控另接受同CORE-2的00015迁移普通注释接续，仅限定单独删除状态与父级过期回收，不改SQL/goose指令；执行文本相等、gen-check及生成物零净变化由本轮收据记录，小diff由主控最终限定核验。


## 13. 最终 CI 草稿快照误报修复（2026-10-28）

最终文档提交 `f9a84e2ea07e9aa86b20aac8c5bd367ae58dd144` 的 [CI 40634485363](https://github.com/open-nerve/NerveProject/actions/runs/40634485363) attempt1：server 失败、web 成功、E2E 跳过。负责人授权使用本机已登录 GitHub 浏览器后，主控直接读取唯一失败 `TestDraftPublishRefusalOrder` / `issue_draft_publish_effects_test.go:93: refusals wrote rows`；不再要求负责人手工摘录。页面日志可读；ZIP 下载受到客户端拦截，未下载成功也未绕过，详见[浏览器能力记录](../../../../.superpowers/sdd/P9-purge/ci-refusal-browser-capabilities.md)。

**局部裁定 CI-SNAPSHOT-1**：完整 app 的周期任务独立运行，旧 `publishSnapshot` 却把整个 `river_job` 的插入、状态与完成时间混入拒绝请求的零写判断。原配置目标测试20遍通过；可控真实 trim 从 running 到 completed 时，窗口零 HTTP 请求、14张业务表逐值不变，仅队列行变化，旧快照明确红。原CI没有逐表差异，不能追溯其具体触发job；本地证据证明同一断言的真实误报机制，见[诊断](../../../../.superpowers/sdd/P9-purge/ci-refusal-diagnosis/diagnosis.md)及[主控核验](../../../../.superpowers/sdd/P9-purge/ci-refusal-diagnosis-controller-verification.json)。

修复只在三个测试文件：共享快照保留全部业务行，严格识别四种既定周期调度的kind/queue/空参数/来源标记；其余所有任务保留稳定身份、参数和入队配置，包括事件probe、未知kind及错误调度。保留真实Runner，不关闭后台、不用固定sleep、不放宽架构守卫。若将它错写成请求kind白名单，会漏掉probe/未知任务；反向变异已实际捕捉这种错误。既有发布事务回滚与拒绝次序均保留，原拒绝断言补逐表差异便于后续定位。

正式回归先在旧helper上明确红、修复后绿；调用族6个顶层测试/105个含子测试PASS，聚焦race五遍20个顶层/135个含子测试PASS、0skip；白名单反例红后精确恢复，恢复测试及局部lint通过。首次新增夹具试写SQL NULL被真实NOT NULL约束拒绝，已改为允许的JSON null，失败原件保留，不当产品性质红。全部源码身份与结果见[修复报告](../../../../.superpowers/sdd/P9-purge/ci-refusal-fix-report.md)和[原件收据](../../../../.superpowers/sdd/P9-purge/ci-refusal-fix-receipt.json)。没有产品、数据库迁移或性能路径变更，不重测T14规模。

本轮[限定 deep-review 复审](../../../../.superpowers/sdd/P9-purge/ci-refusal-fix-review.md)已通过，CI-SNAPSHOT-1 ADDRESSED、0新增发现；主控已核10份原件与最终源码身份，见[核验收据](../../../../.superpowers/sdd/P9-purge/ci-refusal-fix-controller-verification.json)。完整本地七项及准确最终分支CI由主控接续，并写入[最新交付收据](../../../../.superpowers/sdd/P9-purge/delivery-final-receipt.json)。历史f9失败保留，不把此前e003全绿冒作本次通过。P9仍不合并、不清理，main和M4总体收尾状态不变。


## 14. 授权合并、main CI 与归档清理（2026-10-28）

负责人明确授权 P9 `--no-ff` 合并、核准确 main CI 后归档清理，再准备 P10 spec/plan/预检、暂不实现。最终分支 `369b39a09d035031dd15a1ccf18cb2b49ea33f4e` 的七项本地门禁与[分支 CI 40639521407](https://github.com/open-nerve/NerveProject/actions/runs/40639521407)已核通过，最终修复复审零开放发现；没有沿用 e003 或 f9 的结果替代它。

- 合并前 main `6b9d08d4821f1423c7bb9795557727c206b7880f`；`--no-ff` 合并为 `4599a9cfc5be90b2c242977f194157778a5f6ec9`，两父提交分别为原 main 与上述最终分支。合并树 `892cc0583a018ee2110499420b1c3751ab9e7d47` 与已测分支完全相同；本轮没有重复无变化的本地七门禁，仍独立验证 main CI。
- 系统 `/usr/bin/git push origin main` 成功，local/origin main 身份相同；[准确 main CI 40642368845](https://github.com/open-nerve/NerveProject/actions/runs/40642368845) 全绿：server `122663210129`（8分18秒）、web `122663210418`（4分30秒）、e2e `122665873450`（5分34秒）。本次未重跑。公开 HTML 原件和指纹保存在 [main 核验收据](/Users/xiaoruan/project/nerve-project/.superpowers/sdd/P9-integration-20261028/verified-success.json)；已登录 Chrome UI 同样显示 E2E succeeded。
- 归档 **3,317 文件、136,629,586 字节及 1 空目录**，逐文件 SHA-256/字节/模式和完整集合核对通过；另将 77,503 个可再生依赖/构建文件及 2 空目录明确分类，未知项为 0。未用强制清理。manifest SHA-256：`7d7d51e8007591b92cab787189af7976a6da0ced224f414343fc5f646e6e0180`。
- 先 archive/verify，再正常删除远端合并分支、`git worktree remove` 和 `git branch -d`；P9 本地/远端分支及原 worktree 均已清理，清理后 `verify-archive` 再核通过。[清理收据](/Users/xiaoruan/project/nerve-project/.superpowers/sdd/P9-integration-20261028/cleanup.json)、[完整归档清单](/Users/xiaoruan/project/nerve-project/.superpowers/sdd/P9-integration-20261028/archive-manifest.json)、[独立归档方案审查](/Users/xiaoruan/project/nerve-project/.superpowers/sdd/P9-integration-20261028/archive-audit.md)保留全过程。

**历史原件迁移规则**：原工作树根 `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m4-p9/` 下的 ignored 证据，现以相同相对路径保存在 `/Users/xiaoruan/project/nerve-project/.superpowers/sdd/P9-integration-20261028/worktree-evidence/`。因此原 E9 `.superpowers/sdd/P9-purge/` 的新根为 [/Users/xiaoruan/project/nerve-project/.superpowers/sdd/P9-integration-20261028/worktree-evidence/.superpowers/sdd/P9-purge/](/Users/xiaoruan/project/nerve-project/.superpowers/sdd/P9-integration-20261028/worktree-evidence/.superpowers/sdd/P9-purge/)。旧 JSON 里的绝对路径和旧报告字节均未重写；本文旧相对链接/绝对源路径按此映射读取。集成回执位于 archive 外；原可变交付指针在归档前更新，其旧字节另存 `branch-final-receipt-before-integration.json`。归档冻结后不再补写。

P9 的 T16 集成门关闭。P10a/P10b 两段按既定拆分准备文档；P10b 执行前仍须取未来 P10a 集成后的 main 重核。M4 §14 的复合完成线、总体 §9.4、8份输入 handoff 总状态、M5–M8 最终 closeout、C12 和整体性能收尾均保持开放。
