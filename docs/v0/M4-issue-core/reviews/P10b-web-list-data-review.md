# M4/P10b 列表数据层：实施与评审记录

日期：2026-10-29。**T1–T5 既定任务门禁已完成，T5 三项消费链缺陷及附带 M1 修复复审通过，W1 已关闭；T6 开始。** 分支 `codex/m4-p10b`；实际 main 基线 `a06e3f70547b755b03bf8e6b33caafd52725d0f6`，实施起点 `6dfd7258bfe8c6350116131488bf3b83822425c9`。T4 完整本地七门禁通过；阶段最终门禁、终审与准确分支 CI 尚未完成，产品验收十项仍未勾，阶段结束先不合并。配套：[spec](../specs/P10b-web-list-data.md)、[plan](../plans/P10b-web-list-data.md)。执行前 [deep-review](../../../reviews/re-12-M4P10b预检审核.md) 仍为历史准备证据。当前实施记录见第8节，第7节保留 T1–T4 与 T5 初审过程；以下第1–6节保留预检时点事实。

## 1. 实际前置与证据

P10a 最终 d1a19e78 经负责人授权 --no-ff 合并上述 main；[main CI 40723697041](https://github.com/open-nerve/NerveProject/actions/runs/40723697041) attempt 1 的 server/web/e2e 全 success。5,898 证据文件 / 356,803,778 字节 / 22 空目录归档核哈希，清理远端/本地 P10a 分支及 worktree 后再次核验，详见 [P10a review §10](P10a-web-issue-types-review.md#10-授权合并main-ci-与归档清理2026-10-29)。这是当前产品基线的验证，不是尚未编写的 P10b 行为通过。

当前准备证据根为主项目 `.superpowers/sdd/P10b-preparation-20261029/`，在 P10b worktree 外，未来归档必须保留；P10 原准备记录的冻结归档路径见上段，原根目录也仍保留。P10a 的 inventory 重建及缺失 base hash 校正限制没有被本轮归档抹去。

## 2. 本轮预检方法与覆盖

主控按 deep-review 文档模式逐一检查逻辑、集成、测试覆盖，全文重读当前 spec/plan，沿真实生成客户端、三个列表、canonical 全部写者、会话构造/退役及相关测试追踪。前次独立报告的 52 个全文文件与当前树逐哈希对照：33 相同、19 变化（18 源码和 03 规则）；该比较本身只是版本桥，不能冒称本轮重读全部 52 文件。当前源码/测试全文清单与哈希由 `review-coverage.json` 记录，schema/guard/设计的定向片段和广域检索另列，不宣称已全面验证未来算法。

按 brainstorming 的执行者检查步骤，独立 `p10b_builder_check` 全读 spec/plan 并追踪实际接线，只问实施前必须明确的问题。首轮发现 BQ-1；修订后 followup 确认关闭。最终又核 owner/retire 与生成接口，`builder-final.md` SHA256 `dbd3fd4fb1b61d5a35f4209559230cf8017e82db8efa3ec413e7244cf979ddc5` 无剩余 must-answer；它是独立执行者检查，不冒充整分支产品终审。

独立最终实读 spec SHA `b6475d740435432bfb0c5901795150045663a0339729eaff2e651a8b2a05a8c0`、plan SHA `42e12e0ec4d7e83b2a6665380226729f3b35f1f2b09cf74274bcf01133acbdbb` 的冻结副本在 `builder-reviewed/`。之后主控只修 A1–A10 的验收范围笔误、准备完成状态及作者核验勾，精确 diff 与最终文件哈希另存；不把独立结论外推到未读改动。

## 3. 发现、处理与复核

| 编号 | 文档局部问题 | 已完成修订 / 未来实施落点 |
|---|---|---|
| PF-1 / BQ-1 | T5/T6 已依赖根确认协调器，旧计划到 T12 才明确交付；逐 Task 可编译要求不闭合 | T5 接真实协调入口、before/pending/每 ID 队列、列表注册与基本确认；T6 换 canonical，T11 拖放，T12 扩完整矩阵；独立 followup 及 final 均确认可实施 |
| PF-2 | P10a 已改变夹具、归档服务、共享缓存写者与请求输入，P9 准备稿不能直接执行 | 扩已有 issueOf；归档旧服务三方法 T5 一次删除；UserService 旧列表和继承同 T5 删除；七个 canonical 外部写者 T6 收口；普通 boolean skip_activity 在 T5 改源写者，M7 string 合同留其边界；R8/R10/R15/R16/A-01 性质逐任务保留；guard 按真实 count/最早期限 |
| PF-3 | read token 原写 loginId，但 RootStore/ApiClient 未提供该值；只有 HTTP 中间件不足以拦晚到成功答 | 按已有“一代一个 RootStore”在 IssueRootStore 构造时固定 owner，已有 RootStore.dispose 调幂等 retire；T5 核发出/结算/后续读 active，T6 核 token owner；SWR 仍 loginId，不改认证协议。必要 held 200/排队/跨根反例已入 plan，独立 final 核无未答接线问题 |

历史 B-L1–B-L6 六项仍是实施约束，未因重核简化。上述均关闭于**文档层**；没有实施任何修复，也没有新增架构裁定、接口协议或依赖。主控末轮检查将 T14 的 A1–A9 笔误修为 A1–A10，避免最终验证门禁遗漏。

后续实施裁定另见第7节：R9/R10 纯模型职责拆分、R11 workspaceID 明确作用域、R12 每计数来源修复均已实施；R13/W1 冻结实际已持有来源集合的方案已裁定，由 T5 首项实施验证。它们不回写为本节历史预检已实现的事实。

## 4. 实际静态验证与边界

准确命令、退出码、文件指纹、基线及最终准备提交桥由 `preflight-receipt.json` / `commit-receipt.json` 记录。静态核对包括：`git diff --check`；所有修改仅 docs；新增相对 Markdown 链接可解析；T1–T14 顺序与 A1–A10 均开放；phase/关键词文件未改；独立已审稿到最终稿的有限差异；P10a main/归档/清理准确收据。

实际关键词基线 M4/P10、73 rules/53 exceptions；old-issue-urls 41 hits/29 exact keys（37 产品+3 Base 测试+1 子项测试）；114 个直接导入生成 Issue/IssueDetail 的文件逐路径盘点。它们只绑定本基线，T5/T7 删除时必须重数，不能沿用数字代替检查。不存在的未来新文件已显式标“新”，不是漏文件。

本轮不安装依赖、不执行生成器、数据库、产品测试、变异或本地七门禁；未实现的 fetchPart/PagedGroups、新 service、协调器及全部目标测试不得写已通过。完整产品验证仍按 plan F/G 和 A1–A10 执行。

## 5. 接续与未完责任

下一步在该 worktree 获实施授权后从 T1 开始；先核 main 与基线/工作树未漂移，按 spec/plan 逐任务推进。阶段实现后做整分支 deep-review、局部修复和准确分支 CI，先不合并。架构级问题仍先提交已完成工作再上报。

P11 仍接完整详情资源迁移，P12–P15 接页面，P16/M4 收尾接终态门禁与交接；P10a 的 M6/M7 自身页面验收归属 A 不变。M4 §14、总体设计 §9.4、输入 handoff 顶层 open 与 M5–M8 最终 closeout 保持开放。

## 6. 首次依赖预检历史（2026-10-28）

以下为 P9 基线准备时原记录，分支/基线/“P10a 待集成”等时态只代表当时，当前执行依据以上文为准。

### 历史原记录

日期：2026-10-28。状态：六项局部问题文档复审关闭，P10b有条件通过静态依赖预检；P10A-S1已获负责人A裁定，文档定向复审见P10a review；尚未实施。准备分支 `codex/m4-p10a`，worktree `.codex/worktress/codex/m4-p10a`；基线 main `4599a9cfc5be90b2c242977f194157778a5f6ec9`。配套：[spec](../specs/P10b-web-list-data.md)、[plan](../plans/P10b-web-list-data.md)、[deep-review](../../../reviews/re-10-M4P10预检审核.md)。

### 1. 评审方式

按02允许的紧凑方法细化已批准设计；作者准备后，独立审者按deep-review的逻辑、集成、覆盖三镜头阅读全文并追踪代码。静态盘点和依赖推导不冒充产品原型、逐Task重放、行为测试或变异。原件在主项目 `.superpowers/sdd/P10-preparation-20261028/`，准备工作树之外，后续清理不得遗漏。

### 2. 验收标准逐条核对

P9已完成授权集成：准确main CI40642368845三个任务全绿；3,317文件/136,629,586字节/1空目录归档核hash，P9分支/worktree正常清理，清理后再次核验通过。完整证据见[P9 review §14](P9-purge-review.md#14-授权合并main-ci-与归档清理2026-10-28)。这是基线证明，不是本阶段产品验证。

本阶段14任务全部未实施，spec§4产品验收全未勾。准备期只核路径、接口、输入映射、任务闭合、链接和守卫；最终实际命令/文件指纹由主控预检收据记录，未运行的检查不能列通过。

### 3. 准备期裁定

沿用P10a先收窄后换型、P10b扩既有Reconciled及纯分组模型的设计，不重开架构。具体局部结论/理由/代价见spec§3。准备提交不改phase；获实施授权后T1用合法M4/P10，a/b共用阶段号。归档localStorage键按原任务分配在P10b T9修正，P13只核页面。

### 4. 预检发现与修订

独立审者 `p9_integration_archive_audit` 首轮全文52文件/11,219行，源码/生成文件节选另列；首轮 `p10b-preflight.md` SHA `200ed40c8b391e324313088d26d606ee87314b0e1a31ea7dc526170ed5060dd5`。六项局部问题均已修订并由原审者完成定向复审：

| 编号 | 已闭合的文档规则及实施反例 |
|---|---|
| B-L1 | 实际发出前冻结 pending before，own read 抢先返回不丢旧组计数；第三方无法证明转 stale |
| B-L2 | 成功树删除使项目旧读代际失效，包含隐藏列表/未知后代/回查；profile整页失效不复用裁剪后的total |
| B-L3 | T6同任务关闭旧详情/子项/link/relation绕写canonical，partial与count改权威回读，保留P11完整资源边界 |
| B-L4 | partial replay只重放本次重置计数/回答行范围，未回答组和outer/subgroupTotals不重复扣减 |
| B-L5 | 大整数fixture直接测精确SortKey比较，safe Issue数值提取和unsafe拒绝分别测试，不强塞bigint进生成number |
| B-L6 | 回查四路有界收敛；每次captureRead递增请求序号，核实拒绝更新id水位，旧同版本list晚到不能清状态，新发有效权威读才可恢复 |

规则、精确反例、变异及任务归属均已进入spec/plan；设计§3.22/9.5/12同步，没有增加模块接口或依赖。复审 `p10b-rereview.md` SHA `c923b6fd806ee23b2bfb6cf264690571927166cd1a408ce1f6d115d9f52778e7` 确认六项**文档层**关闭，无新增必须修订问题；不是产品行为已通过。

复审准确字节：spec `13d558bd58d81353d759300bb3d5985e1d19629c90b5b7102fe0418cf6c95964`，plan `2529eadf9564639fe4b8e3cd92004277a6c8733d928d2aedf29717baba87d75c`，设计 `d72dc0371bd19ce3ae25ca06d2738cb10a669dde1590521c30d5fc6e60ae47bc`，由 `p10b-freshness.json` 指向冻结副本。主控后续只更新预检结论状态与中文标题，最终字节由准备收据另绑；首轮和中间修订原件不倒改。

### 5. 方法边界

本轮仅文档准备。未运行完整本地CI、生成器、数据库套件、E2E、产品测试或变异；没有新增产品文件、依赖或兼容层。小范围语言能力探针如实际运行，单列命令/输出，不当产品测试。

### 6. 移交事项

P10a实施并经独立整审和负责人授权集成后，P10b才能从准确main重核计划和消费者；当前P10b是依赖预检。P11接详情数据层，P12–P15接页面，P16与M4收尾接最终守卫/浏览器/性能/交接。每段实施完成仍需整分支deep-review和局部修复闭环；本轮不开始T1、不合并准备分支。

### 7. 已知限制

静态预检不能证明迁移后每Task确实编译、全部真实页面请求或分页并发性质。实施期须逐项保留红绿/拒绝/会话变化/恢复证据；P10b执行前须P10a集成main CI通过并完成差异复核。M4§14、总体设计§9.4、输入handoff总status和M5–M8最终closeout保持开放。

P10A-S1已由负责人选择A，保持列表删除范围、调整M6/M7自身页面验收归属，见综合报告的后续决策记录。本段六项模型规则不变，仍需P10a实际集成基线复核；裁定及文档预检不授权启动任何产品Task。

## 2026-10-29 执行启动说明

负责人已授权实施整个 P10b，阶段结束先不合并。主控在既有 `codex/m4-p10b` worktree 按 `6dfd7258bfe8c6350116131488bf3b83822425c9` 启动 T1；该提交包含 P10a 集成后的准备稿，P10a/main 的上述集成事实与原预检证据保持原样。当前 T1 进行中，全部产品验收仍未勾选，逐任务评审及阶段 deep-review 另留真实实施证据；phase 保持 M4/P10。

## 7. T1–T4 实施、独立审查与 T5 接续

实施证据根为本 worktree `.superpowers/sdd/P10b-web-list-data/`，准备原件仍在主目录外置的 P10b-preparation-20261029；两者未来都须归档。主控 ledger 逐 Task 记准确范围、审者、命令原件与源码桥，不能以准备稿审查或 main 基线 CI 代替产品验证。

| 任务 / 最终提交 | 实际验证与独立结论 |
|---|---|
| T1 / ce35d7a9 | 薄服务20项、直接受影响39文件455项（包含前述20，不能重复相加）、api-client 5项通过；错误路径/重复日期query变异红；类型/格式通过。task-1-review.md Spec compliant / Approved |
| T2 / daac89e7 | 并发11项、9个旧生产消费者对应14文件212项回归通过；M1–M3实际红并恢复；类型/定向lint/格式通过。task-2-review.md Spec compliant / Approved |
| T3 / 6dcbeeee | 首版7f4da0dc的29项及M4–M8实际变异；独评发现优先级映射相反，修复后31项、types/lint/format通过，新增真实priority错误映射变异红。task-3-fix1-review.md Spec PASS / Quality APPROVED；旧M4–M8按限定源码桥承接，不能宣称是在修复后重跑 |
| T4 / dde969f8 | 首版247cc370的纯确认/重放；独评 I1 复现 partial 先见创建/删除结果后共享计数漏加/漏减。fix1 按各计数保存来源，129模型+7service通过；六项修复变异真实红并恢复，task-4-fix1-review.md I1/M1均 ADDRESSED、无新破坏。该提交完整本地F全绿；范围外既有W1仍由T5必修，不算已修 |

局部裁定：

- R7：设计 I2 九例和 M3 五例包含 canonical、描述及权威回读，不能全挤进 T4 纯模型测试。按既定 spec 职责分配，T4 保留完整索引、证明纯模型和 verifyIds 意图，T6/T12补真实链。成本是最终A3须逐行核齐，不能提前标全通过；无协议或模块边界变化。
- R8：上级 M4 设计§3.16/P5 spec 明确 priority 行排序 none=0/low=1/medium=2/high=3/urgent=4；阶段 spec 的枚举措辞被误当等级，前端和数字fixture同时反向而自洽。修复仅前端映射、真实Issue语义测试及本阶段spec，保持服务端已有协议；组展示urgent→none另算。
- 证据限制：T3首轮五份mutation raw被同名最终重跑覆盖，原JSONL记录保留，不能重新复验那五份旧输出，也不补造。最后一轮变异、恢复hash与绿灯原件完整，当前98份原件已冻结；fix1用唯一前缀并拒绝覆盖，22条raw/exit及最终6文件hash均核验。此历史Minor不等于产品缺陷，仍需最终整审如实披露。
- T3初始TypeError数量由“两个”纠正为实际一个，冻结旧报告保留；不是改变测试结果。固定显示boundary与raw PageAnswer不被合并改写的实际消费由T4/T5承接。

T4 局部裁定与证据：

- R9/R10：纯 mergePages 与 confirm/replay 共享模型；成员判断、raw重放分别分文件，运行时依赖单向，避免反向转发形成循环。
- R11：member scope 明确 workspaceID；同 user 跨工作区和 unknown-before 的 confirm/replay 均核行、计数、stale 不变。T5 从既有 Workspace 目录绑定 ID，未知不请求；HTTP 不变。
- R12：I1 修复按 counter 保存 Row/范围/最新确认事实，partial 只替换自身来源，显示行与计数证据分开。主模型434行、evidence140行、replay29行；没有第二份实体缓存。27份 fix1 raw/exit、14个最终输入与提交字节核验；restored-green 后补过 spec 文档，产品/测试字节与最终一致，最终 lint-web/knip 已含最终 spec。旧 Task4 runner 漏记 service 逐轮指纹如实保留，新 runner 已补；没有回填旧日志。
- **W1 / R13（Important，T5必须关闭）：** 独立最小 probe 对247cc370与dde969f8均复现 whole 已含窗口外 X、但缺行时 confirm 将 outer10/B5错加为11/6且fresh。它是既有 direct whole fallback 与 raw unknown 原则未闭合的边界，不是 fix1 新回归；原复审结论不改。T5 以实际发送前冻结的来源证据区分旧计数与写期间新回答，补真实请求交错及模型回归；不能一律猜差量或一律 whole 刷新，不能拖到T12/T14。证据 `task-4-fix1-whole-boundary-note.md` 及同名前缀probe/raw/回执；未关闭前不推产品检查点。

T4 完整本地 F 绑定干净提交 `dde969f8bb526dcf2641ad45eb1a93b8f5d1e5d4`，七项均 exit0 且各自执行前后 HEAD/工作树一致：gen-check、lint-go、`GOFLAGS=-p=1 make test`、lint-web、knip、`TURBO_FORCE=true make test-web`、e2e。服务端51包通过、16包无测试文件；前端18任务0缓存，8份包级原日志共1436项通过（web 145文件1274项、其它Vitest157项、api-client Node测试5项）；Playwright169项通过，构建11任务中10缓存，不能将构建缓存说成E2E跳过。web原日志含既有IssueRootStore严格模式警告（T10处理）、Cycle/Modules及React key警告，输出不是零警告。准确argv/cwd/raw/hash、8份包原日志与源码桥见 `task-4-root-final-f-attempt1/receipts.json`、`root-verification.json`。首轮F在lint-web失败的原件仍保留，不与此最终轮混用。F通过不证明W1已修，也不是最终阶段或远端CI通过。

T5接续：先关闭W1，再以唯一root写协调入口接通项目/归档/个人主页列表、同任务移除旧形状及旧列表服务。T5–T14及全部阶段产品验收仍开放；不合并、不启动P11。


### T5 首轮独评与 fix round 1（实施者记录，待限定复审）

首版 `affdcf337f1058fccc72ce560165c3719473a6f5` 的独评结论为 Needs fixes：W1/R13 已获确认；普通看板无泳道的旧子组占位使续页不可达（I1）、其它 part 拒绝期间展开新组会永久漏刷（I2）、None 标签/负责人快建预填向 HTTP 发送哨兵 ID（I3）须在 T5 修复。原 `task-5-review.md` 与独立 probe 完整保留，不以已有全绿代替这些反例。

本轮修复统一 optional 子组读取/续页参数、仅在启动刷新时记录尝试，并跟进小写 none 与空字符串未分组键的消费者；M1 普通/递归子项的多余折叠层同范围修正。新挂载测试贯穿真实 KanBan/ListGroup、列键生产、QuickAddRoot/RHF、根写协调与 FakeNerve，保留泳道、有效 ID 和 priority:none。A 拒绝→展开 B→显式重试 A 后验证 B 的首次请求和收敛。准确红绿、静态门禁、全前端及源码桥记录追加于 `task-5-report.md` 的 fix1 节；本段不代替独立复审，不勾 Task 或 A1–A10。

M2 既有 MobX/React key 等噪声保留原日志并沿原 M6/T10 所有者处理，不升警告阈值或吞 stderr。R17 direct-confirm 即时核实及完整 round/freshness/canonical 仍归 T6，不能从本轮 UI 修复推定已完成。


## 8. T5 完成与 T6 接续

Task5 全段范围为 `dde969f8bb526dcf2641ad45eb1a93b8f5d1e5d4..659ac3cbb22a68bf06a35f936372a14614027693`：起步文档 `10308a96`、首版 `affdcf33`、修复 `659ac3cb`。T5 根协调、固定 scope/query 绑定、新列表服务、目录就绪与可见 stale 刷新均已接通；W1 由模型及真实请求交错证明关闭，发出前冻结实际 held 来源，不猜新回答的窗口外计数。Task5 独评与限定复审通过，后续 Task 和 A1–A10 尚未完成。

- 初审 I1 普通看板续页、I2 拒绝期间展开组漏刷、I3 None 快建哨兵 ID 均已修复；附带 M1 未分组子项多余折叠层一并关闭。原件 `task-5-review.md`、`task-5-report.md` §7、`task-5-fix1-review.md` 和 `task-5-fix1-review-retry.md` 保存在本 worktree 的 `.superpowers/sdd/P10b-web-list-data/`。两份限定复审均确认全部发现已处理、无新增问题。首席结束时返回容量错误，但实际已留下完整报告；主控此前“未开始执行”的判断已在 ledger 更正，替补报告另存，未覆盖原件。
- 最终合法查询/最终测试在六个修复前产品文件上产生 9 项真实断言失败，恢复字节后七文件 73 项通过。lint-web 54/54（36 缓存）、knip 通过；强制完整前端 18 任务 0 缓存，8 包共 1494 项（web 1332/151 文件，其他包 162）。73 项属于 1494 的子集。raw/源码桥/提交桥见 `task-5-fix1-root-audit-precommit.json`、`task-5-fix1-root-package-check.json`、`task-5-fix1-root-commit-check.json`；所有最终输入等该提交，测试并非最终整阶段 F。
- 原六项生产变异按未改变的源码桥承接，未声称修复轮重跑。中间夹具、编译红及失败原件保留；全前端原日志仍有旧警告。新挂载暴露的 KanBan 隐藏列空 Fragment 缺 key 在修复前同样存在，留 T13 同阶段整理与挂载回归；其余 M2 按 T10/M6 归属，不屏蔽 stderr 或放宽阈值。
- R17 为 T6 首项：direct-confirm 的 verifyIds 目前待下一次 fetch 才 drain。T6 完成即时发起、共享 round/去重、四路状态、freshnessFloor，并与唯一 canonical 一同闭合；业务已成功后的核实拒绝不伪报写失败。T6 同时按实际树收口 6 个普通 canonical 外部写者及 Binding 间接读入，原“7 文件”为 T5 前盘点。T6 独评前必须关闭，不能延后到 T12/T14。

下一完整 F 与产品检查点按计划在 T8 执行。阶段终审、准确分支 CI、集成和 M4 总体完成线均开放；不合并、不启动 P11。
