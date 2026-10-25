# M4/P5 工作项列表与显示设置：评审记录

日期：2026-10-26。状态：**实施中；T1–T15完成，T16文档与最终审查待实施；阶段尚未完成，不合并**。

| 项目 | 当前证据 |
|---|---|
| 分支 | `codex/m4-p5` |
| 基线 | main `e9782b8f7e7571b39a4eb8924b4b72a6e60eac51` |
| 工作树 | `.codex/worktress/codex/m4-p5`；主目录保持 main |
| 规格与计划 | [spec](../specs/P5-issue-list.md)、[plan](../plans/P5-issue-list.md)，16验收任务/19实施单元 |
| P4 接收 | [整分支审核](../../../reviews/re-01-M4P4整体审核.md)的三项 Minor 已修复并独立复核；最终分支 `a37f82ec146efca84793b6620aa3e777532bd75a` 的 [CI 40179571485](https://github.com/open-nerve/NerveProject/actions/runs/40179571485) 通过 |
| P4 合并 | `--no-ff` 双亲为 `589d9f01d4f8c30e55f449e430fe1104e2036866` 和 `a37f82ec146efca84793b6620aa3e777532bd75a`；合并后的 [main CI 40180521025](https://github.com/open-nerve/NerveProject/actions/runs/40180521025) server/web/e2e 全部成功 |
| P4 清理 | main CI通过后归档897个过程文件并逐文件SHA核对；旧工作树、本地及远端 `codex/m4-p4` 已删除，主目录保持main且干净 |

## 1. 评审方式

起步规格作者与独立预检者分开。独立者全文审查首版规格、计划和重点源码接缝，核对上级设计及 P1–P4 交接，不将签名核对说成编译或运行通过。按 [02 方法 §3](../../claude_to_gpt/02-method.md) 允许调整的范围，本阶段采用可执行的任务边界、真实 TDD/变异、逐任务独立评审和最终集成证据，替代一次性完成整段原型及逐字节重放。各项完成线不变。

原始预检记录、文件指纹和后续复核放在 ignored 的 `.superpowers/sdd/P5-issue-list/`；P4 完整过程证据在主仓库 `.superpowers/sdd/P4-issue-collaboration/` 归档。正式文档包含全部需要执行的结论，不依赖该本地归档才能理解规格。

P5 完成后仍先交负责人安排整分支终审，处理意见后才合并；P4 本轮的特别安排不扩展为后续阶段免审。架构回归、跨模块语义变动或意外高风险仍按负责人纪律处理。

## 2. 验收标准逐条核对

产品验收清单以 spec §4 为准；T1–T15已有下列实施证据，T16文档与最终审查尚未完成。下表保留起步检查，后面追加逐任务证据。

| 检查 | 结果 |
|---|---|
| 绿色 main 基线 | 精确 SHA 和三个 CI 任务见上表；没有用 P4 分支 CI 代替 main CI |
| 工作树与依赖 | 从上述 main 创建独立工作树；`pnpm install --frozen-lockfile` exit0，使用现有锁文件，未增加依赖 |
| 本地架构基线 | `go -C server test ./internal/archtest -count=1` exit0，包耗时1.563s；没有放宽守卫 |
| 阶段关键词 | `phase=M4/P5`；没有在 P4/P5 到期的例外，`/user-properties/` 的 M4/P11 例外保留；`node tools/keywords.mjs` 为72规则、23例外、无命中 |
| 起步文档自检 | 16个任务顺序完整、产品验收均未勾选；53处本地链接无新增缺失。M4设计原有4个M5–M8收尾文件链接仍是未来交付，不提前创建或标完成 |
| 独立预检 | 首轮4项 Medium、2项 Low，均为局部计划/文档缺口；修订聚焦复核Approved，新增L3也已关闭，见§4 |
| P5 产品测试、性能、交错及变异 | T1–T15证据见下文；最终收口不能以起步检查或已完成任务结果代替 |

### T1：连接运行参数，已完成

- 实施提交 `a306368fde93c0573b0049f5b413b0c4986bad54`；独立审查结论 Spec compliant、Approved，无未解决问题。
- ParseConfig之后强制两项运行参数，保留其他URL参数及UTC AfterConnect。两条同时借出的真实连接读回两项值，并验证application_name与UTC。
- 初始真实RED为两连接均读到on/auto；加入设置后GREEN。分别删除jit及plan_cache_mode赋值，均仅被相应业务断言杀死，原字节恢复后的控制组均绿。
- postgres整包非缓存通过，`make lint-go`两段0 issues，`make test`完整server/tools非缓存通过；没有契约/生成物/web变动。原日志、变异diff及与提交对应的源文件哈希在ignored `task-1-evidence/`，独立记录为 `task-1-review.md`。

### T2：共用列表词汇，已完成

- 实施提交 `6a6f87677072115c44fe14208ab5fff67d34eedd`；独立评审 Spec compliant、Approved，无待修问题。新增四个纯领域文件，DateFilter与唯一日期原子解析提前到T2，T3/T13复用。
- 测试覆盖19排序、项目/个人字段及分组、me/none、未知scope和日期边界。缺API的初始编译红单列；放行link_count正负码及个人assignee分组的两类实际变异被业务断言杀死，恢复后绿。
- domain整包与archtest非缓存通过，`make lint-go`两段0 issues。纯领域尚未接线，完整`make test`按实际影响留T3链末；没有把T1基线冒称T2全量。未单独捕获的四项早期命令由原实施者从真实工具返回原文补证，明确标为“工具输出转录”，未伪称当时原始日志；reviewer已核实。

### T3：查询与 canonical 游标，已完成

- 实施提交 `a0fb27c3ec9a1c246dd4e3448427f52762247e4e`，共享样例修复 `62148b1d83d5c41cb4458e9b3fd47b5bba195a66`；独立首审1项Important已在第1轮修复并聚焦复审关闭，最终 Spec compliant、Approved。
- 纯map解析复用T2词汇/日期及shared.PageSize，保留全部重复日期条件；32种组/子组/键/cursor组合和19码游标形状、类型、范围及canonical均有固定反例。时间原始与UTC年份均校验，固定六位微秒；count保持int64，sort有限float64/零规范，不引入无消费者的Go全序比较器。
- 缺API初红明确为编译红；年0000带时区换算的真实缺陷有业务红/根因修复/绿。丢早期日期条件、Unicode lower、截毫秒、count经float64四项实际变异均被业务断言杀死，恢复后通过。
- 产品提交通过domain/archtest、完整`make test`（server/tools）及`make lint-go`（两段0 issues）。首次lint的QF1001失败与修复完整保留。样例修复只改JSON，修复后domain整包再次通过，未重复容器全套。
- 首审发现旧样例的不同键与UUID同向，精度坍缩或多值min改max也能得到原预期。修复为反向UUID和min/max会反序的名字组合，保留同键tie；手工固定期望经独立复核。误脚本执行前旧JSON的绿色明确排除，已撤回临时样例断言的红不冒充SQL失败。
- 原始证据在ignored `task-3-evidence/`；审查记录 `task-3-review.md`、`task-3-rereview-1.md`。SQL日期AND、null/UUID方向、数值转换与分页仍由T5/T12真实查询验证，不提前打勾。

### T4：普通与完整名字目录，已完成

- 实施提交 `1dd7e1b0af395932c5c506c8c7052d6e732eed86`；独立审查 Spec compliant、Approved，无Critical/Important/Minor。范围含中间root纯文档提交 `559d98db`，产品21文件805+/27−，其中183行生成物。
- 普通Visible独立约束项目/成员未删除、项目未归档、成员active及项目OwnOnly；Historical原SQL/生成物最终无差异。批状态/标签含unused与显式授权归档项目，新状态排triage、原写目录保留。负责人ID只取所选项目未删除指派，真实组合验证先指派后退出/停用或降guest仍取DisplayName。
- 编译初红与测试import冲突均如实记录；五个生产SQL变异（漏普通active、漏archived、漏关系deleted、Historical误加项目deleted、误加成员deleted）生成成功后均被业务断言杀死，逐次恢复重新生成并绿。
- 批目录转换验证所有字段、顺序、参数、同ctx、原错误与空非nil；真实单连接读快照覆盖目录和身份读取沿原事务ctx。此证据不替代后续Listing并发快照交错。
- 五包定位回归、最终三包定位、`make lint-go`两段0 issues、完整`make test`48包/117.035s/stderr空通过；提交后`make gen-check`无差异，工作树干净。首次goimports失败及修复保留。未改契约、前端、迁移或故事，未额外运行web/E2E。
- 原始证据 `task-4-evidence/`，独立记录 `task-4-review.md`；名字编排与真实Listing消费交T5，不提前删除ArchivedProjects/seeEach。

### T5a：Flat仓储，已完成

- 实施提交 `ff9127c9417c4687ca81364651bacbbef718329b`；独立审查 Spec compliant、Approved，无Critical/Important。手写919+/3−，生成263行；只引入Flat实际需要的输入/记录/接口及三职责私有适配，不提前实现Listing/Groups。
- 唯一Flat静态SQL按范围/OwnOnly/triage/全部筛选后分页，total在cursor前，完整关系数组仅对输出行取值，子项数与seen同义。数值CASE明确numeric，sort_order采用有限浮点位序int64；公开cursor仍原值。19码基础排序、微秒及邻接float逆UUID/limit=1真实分页通过。
- 直接sqlc NULL与空数组防御、生产mapper非nil并列数组分别验证。提前LIMIT、遗漏mapper规整、删SQL coalesce三个实际变异均有明确目标失败与恢复。首次API编译红单列；共享标签样例因项目lower(name)唯一约束失败后，调整为每行独立全可读项目，保留原JSON IDs/固定期望并通过。
- 实际sqlc接缝：开放日期端点改为私有text数组+NULLIF转date；nullable名字生成string，采用coalesce文本+NameNull布尔。公开领域/API类型未改。合法状态目录无法构造state_name=null行，真实样例明确去该行；count超大值不伪造海量子项，T5b另做精确证明。
- 完整`make test`48成功包/119.036s/stderr空、`make lint-go`两段0 issues、提交后`make gen-check`均通过。09 mapper变异恢复后只有literal排版/尾逗号调整；内存重建SHA精确匹配原恢复记录，最终lint/full覆盖排版后版本，不声称两个整文件SHA相同。
- 审查Minor：事务ctx测试接受任意非nil错误，已在T5b修成精确rollback sentinel并独立关闭；另记录定位日志中的testcontainers/ryuk `Shell not found in container`诊断，后续ready及测试成功，不清洗日志或为此改产品/依赖/已有容器。
- 原始证据 `task-5a-evidence/`，独立记录 `task-5a-review.md`。T5b补齐公共编排、完整精度和剩余变异后，整个T5已独立验收，见下一节。

### T5b：公共编排与精度收口，已完成

- 实施提交 `a2ecc1df8809af36dff6eea06f1d4b80125e4028`；独立审查 Spec compliant、Approved，无Critical/Important/Minor。8文件873+/36−，最终SQL和生成物无永久增量；T5a事务ctx测试改为固定rollback sentinel，已独立关闭。
- Listing沿原ctx读取完整状态/标签与历史负责人姓名，校验Actor/Target配对，空项目返回空页；limit+1只用实际末行生成游标。越尾页复制原Query，仅去cursor、limit=1，同目录/筛选补原total；端口错误原样。公共hideParents沿HiddenFrom，保留SQL子项数，不重复seen查询。
- 真实单连接READ ONLY/REPEATABLE READ组合验证退出且停用、降guest后的负责人仍参与两向姓名分页、OwnOnly父裁剪及带原筛选的空页total。fake保留组字段只证明Query复制，当前不代表Groups/Subgroups真实SQL。
- 全部适用共享JSON排序码真实limit=1连续分页；有限浮点26个边界值及extra_float_digits=-15/0/3、宽年份微秒/日期均验证。巨大count通过直接提取生产numeric CASE的PG类型/数值探针及真实cursor参数验证，实际子项分页仍为0/1/2，不冒称海量子项。state_name=null不可合法构造的边界沿T5a保留。
- 十二项实际SQL变异均生成成功→业务断言红→恢复生成→同组绿：cursor后total、min→max、OwnOnly非排序及排序两路径、日期AND→OR、null降序、UUID方向、float直接numeric、负数符号、负零分键、CASE误浮点、count舍入。全部恢复SQL哈希一致；与T5a三项变异合并覆盖整个T5要求。
- 缺API编译红与测试夹具引用错误分开保留；extra_float_digits首次失败来自测试配置经ConnString丢参数，改为直接消费克隆config后绿，未改生产池。最终三包定位、`make lint-go`两段0 issues、完整`make test`48成功包/120.932s/stderr空、提交后`make gen-check`均通过。
- 原始证据 `task-5b-evidence/`，独立记录 `task-5b-review.md`。T5已完成；T6/T7负责组/格真实查询，T9/T10负责HTTP/授权接线，T12负责最终全面分页、快照交错和性能，不能提前宣称这些完成。

### T6：分组查询与组续页，已完成

- 实施提交 `963f972e12e278117c4ac46869c33ad9b9163cd4`；独立审查 Spec compliant、Approved，无Critical/Important/新增Minor。手写832+/38−，生成302行；15文件，新增最长手写文件205行。
- 第二条静态Groups SQL在展开前计总体、组展开后/cursor前计GroupTotal，按组row_number取limit+1。组键作为原筛选之外的谓词，A∩C不退化；八种分组字段、约定组序/none末尾、完整关系数组均有真实查询证据。
- Flat/Groups生成行按同形窄投影，以编译期转换复用唯一Issue/Position映射及公共参数构造；Flat新增的空/零组元数据只属内部row。Listing分别组织组选项/组续页，按实际输出行统一裁父，组越尾同ctx/原目录/原条件补GroupTotal；未提前加入Subgroups空实现。
- Groups自身用相邻float、2^53微秒邻域、逆UUID、日期/null两方向limit1验证公开cursor；nil/空数组、直接sqlc NULL、生产mapper及精确事务sentinel另测。真实单连接RO/RR组合验证历史姓名、隐藏父、组选项游标消费、组越尾total=1及不存在组total=0，不冒充T12并发快照或全矩阵。
- 四项实际SQL变异均生成成功→目标业务断言红→恢复生成→同组绿：组键覆盖filter、全局limit、数组裁成当前组、总体计展开行；恢复SHA与提交SQL一致。初红明确为缺Groups接口/模型的编译红；首次lint只因import分组，修复后通过。
- 最终三包定位、`make lint-go`两段0 issues、完整`make test`48成功包/123.355s/stderr空、提交后`make gen-check`均通过。原始证据 `task-6-evidence/`，独立记录 `task-6-review.md`。T7接真实子分组，T12补全部排序/组格与性能验收。

### T7：子分组、泳道总数与格续页，已完成

- 实施提交 `1e3f6fca964b04a0091b69b130f6a3cd826415bb`；独立审查 Spec compliant、Approved，无Critical/Important/Minor。17文件，手写974+/14−，生成346行；三SQL同形窄投影继续编译期转换，共用唯一Issue/Position及精确参数映射。
- 第三静态SQL在单组展开后计GroupTotal，从独立泳道单展开计SubgroupTotal，双展开后计CellTotal；每格limit+1，原筛选和两格键独立AND。固定五项数据总体5、组/泳道各3/3/1、格3/2/2/3/1，格和11；各层逐值、完整数组和独立cursor均真实验证。
- Listing一次处理实际输出的父裁剪/数组复制；格越尾同ctx/目录/原条件补CellTotal。顶层泳道由真实消费者按SubGroupBy独立排序，七个字段都用首组缺较早泳道反例验证，不依赖首次出现顺序；HTTP三封套仍交T9映射。
- Subgroups自身相邻float/2^53微秒/逆UUID/双向日期null连续分页、生产非nil参数与直接sqlc NULL、事务sentinel/取消ctx均通过。单连接RO/RR真实历史姓名组合覆盖格续页、越尾补1与不存在格0，不冒充T12并发或性能。
- 六项真实变异被业务断言杀死并恢复：泳道累加格、双展开后组计数、全局limit、first-seen泳道序、标签键覆盖原filter、负责人泳道键覆盖原filter。五SQL和一真实app变异恢复SHA均与最终源码一致；独立审查核对17文件最终SHA。
- 唯一完整`make test`48成功包/119.278s/stderr空后，自查仅将交集测试改为label-only/assignee-only/both，避免两原筛选互相掩盖。两新增独立案经实际变异、最终三包定位和`make lint-go`验证；产品/其余测试字节未变，不重复全量，也不声称先前full已执行后来两案。提交后`make gen-check`通过。
- 原始证据 `task-7-evidence/`，独立记录 `task-7-review.md`。三SQL核心已齐；全部19排序、7/5/42字段矩阵、并发快照及规模成本仍按T12完成线推进，阶段尚未完成。

### T8：排序更新、描述互斥与写交错，已完成

- 实施提交 `48c744ae958fb110512d4ef8e476a501f32838f2`，基线 `1c318ddc7dc7f1f0e4b733c081a7018d2883c0a9`；独立审查Spec compliant、Approved，无Critical/Important/Minor。28文件，手写604+/22−，生成41+/30−；新增最长文件243行，沿用既有IssueFields和Locks.run。
- nil省略、明确0写入、同值含正负零不写；纯domain拒绝NaN/±Inf。锁下完整行携带原排序，sort-only只更新审计，无活动/描述版本/提及/订阅副作用；state+sort保持完成时间和一条状态动态，同state不重置完成时间。普通description+sort可写，skip_activity出现时true/false均与sort互斥，即便排序同值。
- 初始真实HTTP行为红为未知sort返回400、组合预期422却400；缺字段编译红另行记录。四项实际生产变异：None漏排序、明确0当省略、skip=false漏互斥、只合并state未写sort，均业务断言失败且按原字节恢复后的同组通过。
- 完整原始issues/旁表逐值检查拒绝和无副作用，不以行数代替；state组合核对全活动集合恰一新增。新交错覆盖项目/工作项归档、退出/降级（own/another）、另一sort、state-only的16方向；最终 `TestIssueSortOrderInterleaving -race -count=5` 精确80方向通过。最初state夹具误读取创建者NULL活动，分离创建者与写者并增加全原始活动集合断言后通过，未改生产语义。
- 六个既有顶层交错各 `-race -count=5` 通过：`TestTwoChangesOfAWorkItemSerialize`、`TestAWorkItemGoneWhileItsChangeWaitsIsNotFound`、`TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`、`TestArchivingAWorkItemAndMovingItSerialize`、`TestAWorkItemsWriteAndDeletingItsProjectOrWorkspaceSerialize`、`TestDeletingAWorkItemAndChangingItSerialize`。权限矩阵及原写锁/闸门/busy定位也通过，不以复用路径免除验证。
- 最终 `make gen`、Go定位及issue HTTP整包、`make lint-go`、`make test`（48成功包，bootstrap123.094s）、`make lint-web`、`make knip`、`make test-web`、提交后 `make gen-check` 均通过。Web测试17任务中11缓存，如实记录；本任务未跑e2e，I6完整故事和排序落库断言仍属T15。
- 原始证据 `task-8-evidence/`；01–16普通diff仅含tracked文件，变异另有源diff/SHA；18起含新文件内容，最终提交覆盖全文件，不倒填早期证据。独立记录 `task-8-review.md`，原始最终28文件哈希逐项一致。未变Locks内部和全局表序沿已验收基线，审查另核当前路径与既有锁/闸门回归；没有把单Task审查冒称全库证明。创建排序/P3子项排序未改，不将巨大有限数静态加法疑点当溢出复现。

### T9：项目列表API与HTTP边界，已完成

- 实施提交 `f3c740e0d8dbf1eaef7eafd618f721bac3669ba8`，基线 `add5f6d9e9e7363d1d7b80b717df38066d50ebe3`；独立审查Spec compliant、Approved，无Critical/Important；唯一Minor为已记录的既有容器日志背景。手写14文件1154+/3−，生成3文件1056+/128−；新手写最长232行，生产新文件最长66行，未新增bootstrap生产依赖或SQL。
- ListIssues在RequireActor/Parse后开启独立ReadTx，Find、issue.read/issue.archive授权、目录、三SQL和补total都传txctx，只给路径项目grant；显式归档项目可读。真实单连接组合核READ ONLY/RR及隐藏父/空页补total，尚不等同T12提交交错和性能。
- 标准ErrorHandlerFunc只对精确GET路由、typed InvalidParamFormatError、该路由声明的非日期query且真实重复值映射422；其他仍400。StrictMiddleware深拷贝map和片段，不改RawQuery，不重复认证/限流。17种重复参数、缺token/坏path顺序、旧comments及日期AND真实边界通过。
- 真实生成Page/Group可选nullable游标与可选数组，mapper按分支显式null/值/省略及[]。HTTP八种完整JSON实例和bootstrap双标签×双负责人实查三封套、原筛选交集/组格续页total、完整20字段Issue行；没有description_html/link_count。运行server角色实际执行三SQL及续页、归档项目两权限分支。
- 权限矩阵4行40格覆盖活/归档项目与活/归档工作项，完整issueColumns/archivedColumns与独立集合/total期望；prepare行沿原harness的write:true作每格隔离，GET读写分类与守卫不变。实际初红包含缺接口编译红、未登记matrix、夹具未用import/错误message/prepare命令问题，报告逐一记录修正，未伪称全为产品业务红。
- 七项真实变异均业务失败、原字节恢复后同组通过：归档误用read、移除ReadTx、日期重复压扁、错误映射扩旧comments、坏path同名参数误判、片段浅拷贝、末页cursor误省略。最终文件SHA与各变异恢复值核对，未残留变异。
- 最终定位、HTTP整包反向码、`make lint-go`、唯一全量`make test`（48成功包，墙钟122.956s）、`make lint-web`54任务（8缓存）、`make knip`、`make test-web`17任务（11缓存）及提交后`make gen-check`通过，最终status干净。未改迁移/故事，未跑e2e；T15仍须实际客户端按普通CSV单键、日期重复键序列化，不以TS类型推定运行时explode。
- 原始证据 `task-9-evidence/`，完整报告 `task-9-report.md`，独立记录 `task-9-review.md`，审查核对最终17文件SHA及七变异证据。保留既有Ryuk Shell not found诊断及随后ready成功的原日志，不为此改变既有容器/依赖。

### T10：个人主页三关系与目标成员目录，已完成

- 实施提交 `a8bba26def0f171697573c0b5f40ab1966311541`，基线 `0b599e6860da5212084776abd2d6f6dc4df0acdb`；独立审查Spec compliant、Approved，无Critical/Important；两个Minor为环境异常/日志和多余验证成本，非产品缺陷。手写25文件998+/8−，生成3文件546+；新手写最长234行，生产新文件最长74行。
- 新member_issue.list仅允许workspace Admin/Member；同ReadTx依次查workspace、caller权限、路径账户的未删membership、caller普通Visible和Listing。ActorID与TargetMemberID独立；workspace不可见先404，目标inactive仍存在、缺失/软删/仅other成员404member_not_found。Workspace目录复用MemberOf，无新SQL、锁或跨模块导入。
- 新成员GET与顶层转引/生成客户端齐备，复用T9原query/错误映射/封套。11种重复非日期参数、relation必填400/非法422、坏user_id path400、缺token优先级、禁用参数及日期AND均实测；HTTP整包双向码包括复用的workspace.member_not_found。
- 真实API构造A/B/C互异关系：创建者/负责人自动订阅，B评论形成仅订阅项，另有A-only与C-only。逐目标、逐relation都有不同明确集合；不假设订阅互斥。OwnOnly按caller且B-created∩A-own为空，退出/未入/归档/软删项目、跨项目求交、隐藏parent及计数均有反例；runtime角色真实请求三关系、分组和续页，再降caller为guest验证OwnOnly。
- 三relation×完整workspace列共18矩阵格；该seed的两个允许caller均无普通可见项目，精确空页是预期，非空范围由独立API/runtime场景证明。仅该endpoint的user_id按账户校验，静态入口独立synthetic非nil账户，保留newSeeded旧nil契约；错workspace/随机/行ID冒充/nil/其他路由及固定目标均有正反例，无targets豁免。
- 单连接真实workspace/access/project/identity/issue端口检查READ ONLY/RR、target/Visible/SQL/parent/补total同ctx；无新提交屏障，不冒充T12并发证据。八项有效变异（七生产、一矩阵守卫）业务红并恢复：关系误用actor、target强制active、admin扩大项目、OwnOnly误用target、忽略项目退出、账户目标放行、坏path误422、移除ReadTx。
- 退出条件SQL变异恢复后首次控制组因新建Ryuk无8080映射在业务前失败，62.58s退出1；源码/生成物已恢复。原样唯一重跑2.75s成功，未操作Docker或依赖。与既有Shell not found后ready背景分别保留，不把基础设施失败记业务红。
- 24全量发生在C-only补强前；25补强定位实际命中13顶层和18格；29lint要求测试if链改switch，30全量已覆盖补强，31等价调整后lint通过。作者为逐字节SHA再次跑full，root纠正无需仅为证据整齐重复；已启动32自然结束（48成功包、129.50s），之后未再追加。最终Web54/17任务、knip及提交后gen-check均通过，28文件SHA与最终提交一致、status干净。
- 原始证据 `task-10-evidence/`，报告 `task-10-report.md`，独立记录 `task-10-review.md`；审查独立核对28项当前SHA、最终full manifest和变异恢复，gen-check自身metadata未存28项SHA，不混同两份记录。本Task无迁移/故事、未跑e2e；后续接口故事、并发/性能仍按原完成门推进。

### T11：关联列表普通目录替换与旧链清理，已验收

- 实施提交 `1292ceb80b6847d10687a950c4791eb5f2f2fe59`，基线 `b733b60186265915587d518b6875b5f9e4aff4b3`；独立评审Spec compliant、Approved，C0/I0/M1。唯一Minor为既有容器Shell诊断，定位随后ready并通过，按既有环境噪声保留记录，不改容器或依赖。手写15文件249+/272−，生成1文件删除29行，总16文件；ListIssueRelations生产文件90行。
- ListIssueRelationsDeps加入既有shared.TxManager的ReadTx，RequireActor后从首次源读取/授权到关系、目标、Visible、子计数及隐藏父批读均在同一独立快照；issue.New显式传d.ReadTx，无写闸门/FOR锁。普通目录按caller映射，再用完整已授权sourceGrant覆盖或补入源项目，保留归档源同项目目标；两向冲突grant反例防止普通目录扩大或收窄源权限。
- 保留seen/HiddenFrom/hideParents及All/Own子计数，按原relations顺序组装ID/TypeFrom/CreatedAt，不按目标ID重排；目标归档/软删排除，OwnOnly和父项遮挡不退化。删除前确认唯一产品入口，清理seeEach/projectsBut、Archived双方端口/provider/module/bootstrap/fake、专属测试及SQL；仅make gen-go删除对应生成查询。全server及外部Go/SQL旧符号无匹配，Historical完整保留，普通/Historical/relations/sub_issues查询与基线无差异。
- 初始编译成功业务红：app未消费Begin/Visible、真实ReadSnapshot的tx.calls=0；普通集合新增场景在旧实现已绿，明确作为行为保持控制组。最终准确8顶层app/bootstrap关系/历史测试、4项project adapter目录测试通过。新增单连接实查READ ONLY/RR及真实父/计数读取，无并发屏障，不声称race五遍。
- 四项实际生产变异业务红并原SHA恢复、同8顶层控制组绿：漏sourceGrant导致归档源同项目目标丢失，跨项目OwnOnly漏判导致他人/未知创建者泄漏，Visible误用Historical带回软删/归档目录，Visible外ctx触发真实单连接deadline。没有以fake panic或编译错误代替业务失败。
- 唯一完整`make test`48成功包、127.675s、stderr空；`make lint-go`两段0 issues、提交后`make gen-check`通过，status干净。15个存续文件SHA及删除状态与最终提交一致，full后未改产品/测试或重复全量。无API/迁移/前端/故事变动，未跑Web/e2e；verbose既有Ryuk Shell诊断随后ready，未操作容器或依赖。
- 原始证据 `task-11-evidence/`，报告 `task-11-report.md`，独立记录 `task-11-review.md`。T12仍须完成全面分页、真实提交交错和10k/100k性能，不能由单连接结果替代。

### T12a：完整分页矩阵与真实提交快照，已验收

- 基线 `26fa7cf8aa179473d77160fca95d90eea09ec6da`，实施提交 `8238508e311c42a607add9b2827a30155db8e10b`；独立评审Spec compliant、Approved，C0/I0/M1。唯一Minor为既有Ryuk日志噪声，成功ready/无启动重试，保留环境维护记录，不改容器或依赖。8个手写测试文件1051+/4−，每文件不足400行；无永久产品/SQL/生成物/契约变化，父T12尚未验收。
- 真实Flat/Groups/Subgroups及caller≠target成员HTTP入口分别19排序配置；另按默认序跑项目7字段、个人5字段与全部42个有序不同字段组对，含真实跨项目、nullable creator/date与多值none、A筛选∩C组、OwnOnly、隐藏父、归档/软删邻居。独立固定ID序与key→ID成员表断言各层total及完整数组，不调用生产比较器/分组器生成期望，不把两维交叉成额外要求。
- 逐页保留HTTP原顺序，直接消费返回的组/格首cursor，逐值检查公开key/ID、末页null与越尾补total。共享JSON在bootstrap因真实label唯一约束复用ALPHA、给äBC固定后缀；时间映射到2255年相邻2^53微秒，count映射为0/0/1/2以造真实子项，不冒称逐字复现原种子。大整数由三条生产SQL原CASE执行探针和Groups/Subgroups typed int64参数单独证明，邻接float及逆UUID仍实际翻页。
- 八个真实提交交错：state/parent、label名、assignee名、caller退出，以及flat/groups/subgroups/member空页fallback。目录真实返回后且Rows前、或真实Rows首查空后暂停，另一连接真实提交后放行；本页旧快照、下一请求新快照。观察真实READ ONLY/RR/同pid及独立writer pid，端口只委托/观察，记录有同步，读goroutine返结果，失败release/cancel/join后清池，无sleep或MVCC假锁。fallback保留目录/filter/actor/target/组格键，只改Query.Cursor=nil、Query.Limit=1；adapter PageSize另含lookahead为2。
- 五项实际生产变异编译成功后业务红并恢复绿：creator/project/date错列（三项均真实生成/恢复生成）、两入口去ReadTx、Listing目录外ctx。去ReadTx在真实source/workspace检测read committed/off时失败，未冒称该变异已走到writer提交；外ctx在真实states检查失败。所有生产/生成恢复SHA匹配，未留变异。
- 初始正确行为如实baseline绿；02失败是fixture把start改2030却留2026 target，触发已有日期CHECK，修正fixture清target后绿，未改产品约束。八快照实际各跑五次race，共40子场景、26.84s通过；lint两段0 issues；唯一完整make test 48包、130.911s、stderr空；提交后gen-check通过。八测试最终SHA与lint/race/full一致，最终status干净。既有Ryuk Shell诊断随后ready，未动容器或依赖。
- 原证据 `task-12a-evidence/`、实施报告 `task-12a-report.md`；独立记录 `task-12a-review.md`。本单元未运行Web/e2e或规模性能，不以测试工具/默认skip替代T12b实际10k/100k、五样本及EXPLAIN完成门。

### T12b：真实规模、性能修正与单一排序收口，已验收

- 基线 `c08e1061b2ec5188fd8e1a8f3eb9a22c2f157957`，实施提交 `7df20e80229c19787eb7d9bc946789ed8dc1718c`，修复提交 `f73e85ed46303e813c8cee44b5145e22607e474d`。首轮独评C0/I1/M1，I1为Flat优化复制完整排序定义；原作者修复后范围复审I1关闭、新增C/I/M均0。唯一既有Minor仍是成功ready后的Ryuk日志噪声，保留环境维护记录。本单元与T12a共同完成父T12，未验收后续任务或P5整阶段。
- `list-scale-v1`两库真实迁移/runtime grants，owner只seed/统计/ANALYZE，业务与EXPLAIN均runtime。N为正常根项10000/100000；主项目物理10700/107000、未删10500/105000、默认含活跃孩子匹配10200/102000，全库11556/107856。五状态含另置triage、全priority、null日期、0/1/2多值、父子及归档/软删邻居、普通/OwnOnly/退出/归档项目、caller≠target三关系均有DB实际分布与独立期望核对。OwnOnly另仅52匹配，不冒称同等100k负载。
- 复用issueWorld的已准备URL/pool构造，补既有Members与外ctx接线；生产NewPool.Config复制后仅加测试Tracer，保留jit=off/custom_plan/UTC codec、MaxConns4/cache statement。捕获原sqlc SQL、typed参数深拷贝、request/pid/起止/错误；rows关闭才结束，回调不查库。计时仅同步ServeHTTP模块路由，含API中间件与tokenIsAccount，不含网络/真实token认证；构造、契约/oracle/DB核对、写盘及EXPLAIN均在外。
- 两规模各13场景、各1预热+5实测，最终130样本/26预热全部保留；共456份运行角色RO/RR计划，内存原typedargs重用、安全位置Prepare比对OID，捕获三SQL与最终生成常量逐字相等。26次越尾各有首空与fallback两语句，PageSize51→2、原filter/组格/target保留；真实续页/末页/总数/完整数组/父子计数均以独立oracle核对。
- 优化前完整绿05的默认计划完整回表102000次。首版延后投影虽改善耗时，却复制全序；最终唯一row_number窗口定义业务全序，cursor仍一份且先于窗口、total在cursor前，内部序位选页后才完整回表，输出仅按序位。为收口窗口候选的行数高估，序位上界另加同值typed LIMIT；两处共用一个int32页参数，不改变选中集合或公开游标。最终默认与两成员场景页后均51次issues_pkey，没有新增索引、SET、架构或守卫例外。
- 最终性能为32轮对应的源码：运行时HEAD仍7df，但完整diff及16文件SHA与最终f73提交/工作字节完全相等，独立复审另从Git对象核实，不为metadata HEAD重复采样。旧22及窗口候选25保持为对照，不能冒称最终成绩。环境为macOS26.6.2/Apple M5 Max/18 CPU/128GiB，Docker VM报告7931MB，PG18.6 aarch64、Go1.27.1；实际120条现有索引DDL与所有SQL/参数/BUFFERS另存原证据。

| 场景 | 10k SQL min/median/max ms | 100k SQL min/median/max ms | 100k目录中位 ms | 100k模块 min/median/max ms |
|---|---:|---:|---:|---:|
| 默认页 | 11.02/11.34/11.65 | 104.27/105.53/106.02 | 7.84 | 115.96/117.02/117.75 |
| label_name | 22.68/22.73/22.91 | 230.31/231.55/233.64 | 8.14 | 242.07/243.14/245.42 |
| sort_order | 14.23/14.35/14.61 | 146.55/147.71/148.18 | 7.94 | 157.91/158.94/159.25 |
| state组 | 19.99/20.24/20.97 | 201.11/205.16/206.83 | 8.12 | 216.93/219.24/221.75 |
| label组 | 26.11/26.21/27.34 | 287.36/287.62/289.51 | 8.17 | 301.67/303.21/304.89 |
| 泳道 | 55.77/56.14/58.02 | 619.23/622.38/625.09 | 8.49 | 643.60/646.32/649.97 |
| 筛选∩组 | 12.28/12.29/12.55 | 127.93/128.43/131.38 | 8.06 | 138.83/139.40/142.64 |
| none筛选 | 9.79/9.93/9.95 | 100.03/100.65/104.30 | 8.03 | 111.52/112.02/115.63 |
| 重复日期AND | 7.33/7.37/7.40 | 66.41/66.62/67.07 | 7.87 | 76.84/77.19/77.69 |
| 成员创建 | 6.29/6.33/6.53 | 48.62/49.20/49.87 | 7.46 | 60.12/60.64/61.67 |
| 成员分配 | 6.63/6.65/6.73 | 95.99/96.59/99.84 | 7.57 | 106.95/107.81/110.77 |
| 成员订阅 | 6.79/6.82/6.92 | 97.19/97.34/98.95 | 7.74 | 108.13/108.47/111.82 |
| OwnOnly（52匹配） | 1.06/1.07/1.13 | 1.15/1.20/1.24 | 0.69 | 3.23/4.10/4.79 |

- 核心10k处于2–100ms基准；100k默认105.527ms、label_name231.554ms分别在55–125/210–430ms内，state组205.164ms、label组287.624ms、泳道622.383ms。分布与原S1不同且原seed未留存，不冒称同seed复现；更低数字不外推大目录/高关系度数，设计区间也不是CI硬毫秒断言。
- 单一排序的实际代价保留：相对旧22，sort_order约+59.5%、member_assigned/member_subscribed分别约+87.2%/+83.2%。页后回表已按51行收口，分页前成员过滤仍102256次相关Index Only Scan；旧assigned计划为hashed成员子计划。默认窗口排序external merge 6592kB、run condition<=51，其前仍处理102000候选；EXPLAIN execution143.621ms/shared hits3437、temp read/write1173/2364（旧22为711/1538）。EXPLAIN与请求取行计时不同，不逐样本外推一份计划。保留每项目20k目标，相关过滤/完整排序/临时块/跨项目扫描与后续索引特化交M8。
- 有效变异为参数浅拷贝、提前End及最终序位lookahead少一行，均真实业务红/恢复绿，生产SQL经真实生成。历史10的Fatal前未defer rows.Close导致清理阻塞，错误启动11造成短暂suite重叠；精确终止自有进程、补资源释放后12/13重做，旧10/11不作有效证据，11编译红亦明确无效。原始时间证明08/22在其外，最终修复23–38全部22条命令无重叠；未操作已有Docker/他人进程。离线核验正则曾漏匹配分组row_number，修正并留audit记录，不包装成产品suite失败。
- 最终31覆盖四入口19序、7/5/42、八快照及三SQL精度/typed/tracer；33有效lookahead变异恢复；34八快照各race五遍共40子场景、27.17s；35唯一针对最终SQL的完整make test48包/143.34s/stderr空，36 lint两段0 issues，38提交后gen-check通过。前候选29全量仅支持其源码，未错记为最终结果；默认skip明确不算性能通过。
- 原证据 `task-12b-evidence/`，最终 `32-fix1-scale-bound/`、`fix1-performance-summary.json`、`fix1-final-plan-comparison.json`、`fix1-final-verification.json`；实施及修复报告 `task-12b-report.md`，独评 `task-12b-review.md`、范围复审 `task-12b-rereview1.md`。首轮22的成员订阅有两次偏慢86.033/84.788ms，全部保留，未只挑最快值或删慢样本。

### T13a：迁移、持久化与纯校验，已验收

- 基线 `bd5702883aa04cdd98d2a1a23179a9fff4e6e73b`，实施提交 `433c82c85782a8286868c771dfb19f7240e9a95f`；独立评审Spec compliant、Approved，C0/I0/M1。21文件，手写780行差异、生成102行差异；父T13保持未完成，API接线与并发留在T13b独立验收。
- 00026新增三列JSONB、准确默认与形状CHECK；从真实1–25迁移原字节构造旧库，插入非默认旧行后升级、Down、Up，比较全部原字段及审计不变。SQL NULL准确23502/列名；JSON null、非对象、混合数组及缺键准确23514/约束名，形状合法的未来词留给应用校验。
- 域值保留nil与非nil空数组、null分组及false；顶层整体替换。存储首写为默认叠加patch，冲突更新的五个set标志分别保留未提供列；Create/Ensure使用列默认，Ensure不覆盖既有值或审计。纯issue ListSettings复用列表词汇/日期规则，无数据库、目录或跨模块domain依赖，确定性返回字段及数组索引错误。
- 缺迁移、旧存储忽略新值为真实业务红；缺类型编译红另列。CASE保护删除、SetFilters常真、放行cycle_id、Ensure改为覆盖四项实际生产变异均业务红并恢复绿；SQL变异实际重新生成。中间JSONB文本键顺序fixture失败及goimports失败已修复，未删除原日志。
- 最终lint两段0 issues，完整make test48包通过（148.234s、stderr空）；完整make e2e总53.187s，135 passed（133 Chromium、2 Firefox A4，Playwright自身46.4s），含旧P8及烟测。提交后gen-check通过（2.985s、空diff）。21源文件SHA与完整测试/e2e及最终提交逐一一致。唯一Minor是NO_COLOR/FORCE_COLOR及testcontainers探测日志噪声，登记环境维护，不为消噪重复suite或修改容器。
- 原始证据 `task-13a-evidence/`，实施报告 `task-13a-report.md`，独评 `task-13a-review.md`。评审中T13b接口、闭合DTO、真实API422、GET无写/用户隔离/恢复及真实并发的跨任务未验收项已逐项归入b完成门，不以a批准代替。

### T13b：应用接线、闭合API与并发闭环，已验收

- 基线 `c4ad379e0fbc752c459999a6952969bd3b487cc8`，实施提交 `7672525e220639c8d4e47ab47781d36d1b1034fe`；独立评审Spec compliant、Approved，C0/I0/M1。26文件、手写777+/44−共821行，生成物另计；与T13a一起验收父T13。唯一Minor仍为testcontainers/颜色变量日志噪声，保留环境维护记录，不重复suite消噪。
- project/app声明必需IssueListSettings与自己的小值，issue提供真实纯实现，bootstrap逐字段转换；三个project.New组合根和两处Update构造者全部接通，无nil旁路。RequireActor先行、旧navigation错误在前、新词错误追加，一次返回确定性字段错误，读写仍按actor.UserID。
- 闭合GET/PATCH三字段、十filters键、七display键/两calendar键、十一properties布尔值真实生成；nil/显式[]、{}/省略、内层null和false双向typed转换分别验证。结构错误400不进入用例，合法形状的非法UUID/日期/词值由真实API返回422；HTTP整包双向错误码检查通过，未放宽守卫或新增错误码。
- 真实 `TestIssuePreferencesAPI` / `TestIssuePreferencesAsRuntimeRole` 覆盖GET缺行默认且不写、首次保存、整体替换/省略/清空、两actor与另一项目隔离、归档规则，以及真实leave→add后完整原偏好行及审计不变。runtime使用独立登录与最终grants；既有GET/PATCH/归档PATCH的27权限cell均用完整新JSON断言。P8 API/页面故事及TS默认同步，SQL直接核三列持久化，没有新增UI。
- `TestPreferenceFieldsDoNotOverwrite` 在已有同一行上验证两组各双方向：navigation↔filters、display_filters↔display_properties。首次tuple探针超时后，真实锁诊断证明UPSERT在transactionid等待；改为既有key wait加精确已存行RowWaits后再释放A，旧tuple调用及清理保持。真实store预读整行并五set=true变异，四方向均在两个成功响应后出现终态失字段；漏词校验使坏UUID由422变200、Ensure覆盖使退出恢复丢自定义三列，均业务红并恢复绿。
- 新增三包测试 `-p 1 -race -count=5` 通过（146.37s），7顶层及4并发子项逐项各五次；race后唯一变化为删除未读取的误加CascadeDeps字段及注释，精确旧源重建SHA和清理patch闭合映射，不伪称全部SHA相同。后续完整make test48包155.66s通过；lint-go两段0 issues、lint-web54/54（8 cached）、knip、test-web17/17（11 cached）通过；完整make e2e总46.38s、135 passed（133 Chromium、2 Firefox，runner40.7s）。提交后gen-check2.76s、空diff；26最终源SHA与六个最终门及提交一致。
- 原始证据 `task-13b-evidence/`、实施报告 `task-13b-report.md`、独评 `task-13b-review.md`。编译红、非法fixture、未用import、探针超时及编辑工具错误均保留且不算有效变异红。评审跨任务项由a已批准记录闭合；actor写范围专门退化仍归T14，本阶段普通两actor隔离不代替该证据。实际文件/测试名已同步plan，后续命令不沿用未存在的拟名。

### T14：矩阵、整程序读边界与回归闭环，已验收

- 基线 `ff13a6b0c8a64f49e46c79a29c4e01ea911cd8f4`，实施提交 `15388587d348ef812c1b8a315132bc66515f8fdb`；独立评审Spec compliant、Approved，C0/I0/M1。五文件317+/1−共318手写差异，新增四个专职测试文件，旧tracer仅加可选池配置；没有永久产品/SQL/API/生成物变化。Minor为既有testcontainers探测日志噪声，保留环境记录，不改容器/依赖。
- 真实完整应用写位=1且无队列：项目行锁阻塞首个sort_order写，第二个合法写准确503/server_busy/Retry-After1；未释放首写时，活项、归档项、成员列表仍准确回答。释放后首写成功且落库19。EXCLUSIVE表锁期间三SQL形状、成员列表和原偏好GET均完成，证明普通读不请求FOR锁或写锁；没有用MVCC普通读伪造行锁等待。
- 单连接真实tracer与真实ReadTx/目录/SQL覆盖Flat/Groups/Subgroups、个人及越尾补total。每请求RO/RR BEGIN→全部同PID且完成无错→COMMIT，SQL名及fallback两查精确核对，隐藏父项/完整数组/total独立断言；既有current_setting与八个提交快照在本次整包/race实际回归，未重造平行编排。
- 前序矩阵经本次完整执行重新计数：listIssues40、listMemberIssues18、preferences GET12/PATCH15。有效且仍存在的Bob目标补guest caller三关系403、inactive/deleted caller三关系404，防止目标缺失遮住角色退化。45项指定历史/runtime/contract实际匹配一次，覆盖activities、description skip/版本/快照、relations、subissues、旧preferences、目标/矩阵完整性、三运行角色和原写闸门。
- 三项真实生产变异均业务红/恢复绿：允许workspace guest后实际200替代403；States目录逃离事务ctx后单连接无法另借，五形状实际500替代200；Upsert合法扩大同项目recipient但只返回caller后，A的PATCH200且准确响应通过，B真实整行filters medium→high及审计变化被抓住。actor前四候选的生成/nullable参数编译失败单独保留，不计业务红；最终RETURNING is_caller与显式uuid通过真实生成。全部生产源/生成物原字节恢复。
- 强制race第一组93.15s：`TestIssueListSnapshot`、`TestIssueListTraceArguments`、`TestIssueListTraceBoundary`、`TestPreferenceFieldsDoNotOverwrite`、`TestIssueSortOrderInterleaving`、`TestP5ReadBoundary`六顶层均实际五遍；覆盖八快照、四偏好方向、16排序方向与full-gate/no-for-locks/snapshot-connection五形状。原始计数为6顶层/44子路径各五次，包含中间分组，不将路径数冒充独立业务方向数。
- 强制race第二组70.33s：`TestTwoChangesOfAWorkItemSerialize`、`TestAWorkItemGoneWhileItsChangeWaitsIsNotFound`、`TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`、`TestArchivingAWorkItemAndMovingItSerialize`、`TestAWorkItemsWriteAndDeletingItsProjectOrWorkspaceSerialize`、`TestDeletingAWorkItemAndChangingItSerialize`六顶层及26子路径各实际五遍，无FAIL/data race。完整逐路径清单及行号在独评 `task-14-review.md`，原计数 `p5-race-counts.json` / `existing-race-counts.json`。
- 定位、archtest/bootstrap整包（140.23s）、issue/project HTTP整包（11.22s）与完整make test48包（155.03s）通过；lint-go0 issues、lint-web54/54（38 cached）、knip、test-web17/17（全部cached）通过，缓存如实保留。提交后gen-check3.13s、空diff；五交付源在各门与提交逐字节相同，34条运行记录无重叠。没有永久SQL/故事/API变化，本任务未重复性能或e2e，T15仍需其独立门。
- 实施/独评记录 `task-14-report.md`、`task-14-review.md`，原始证据 `task-14-evidence/`。评审跨任务项由P4/main基线及T9/T10/T13批准记录承接；本次真实矩阵、HTTP/runtime/历史执行不能扩称前序全分支终审，P5最终审查仍待T16。

### T15：八接口故事与完整e2e，已验收

- 基线 `97d6ead3cc125007bbc8e286045768e4a2032632`，实施提交 `47cefc111be9a7d1e934fcd2e2b6c259c8483ecb`；独立评审Spec compliant、Approved，C0/I0/M1（既有颜色环境警告）。仅11个e2e文件953+/1−共954手写差异，七新故事最大200行；没有永久server/api/web/tools变化。所有前置用生成客户端API，DB仅SELECT断言，无新框架、页面、依赖或as any。
- 真实`--list`为8 tests/8 files；构建后八故事基线全部通过。生成operations/schema直接进入窄list/page helper，普通数组单键CSV、日期逐atom重复键，undefined/空普通数组不发筛选。typed偏好GET沿旧资源，保存布尔显式转查询string。I3验证state组及priority×label泳道各层total/完整数组/逐页，并以C-only干扰抓住A∩C续页；I4独立字面量表覆盖十九码各三页、日期范围/重复AND/none/空交集，不从大页反算期望或截断时间精度。
- I5两人互异三类非默认、me随caller、保存值请求与显式参数整页等价，无参数仍全列表；SQL独立核三列/作者，清filters仅改一类，错误422不写。I6一次state+sort=200与真实DB/完成戳/作者、唯一状态动态old/new/actor/时刻及左右邻居全验证；I7复用普通创建，状态内max300→10300，另一状态90000不干扰，创建活动/订阅沿原断言。I19每relation三目标集合不同且含A-only/C-only，自动订阅保留，跨项目caller可见性/OwnOnly与workspace guest403明确；I20四格及guestViewAll后仍禁创建；I15只增长列表断言，原顺序/幂等/活动/评论/关系/删除全保留。
- 四项生产变异均真实build→指定1故事业务红→原字节恢复/再build→同故事绿：group续页漏label使total3≠2；省略sort赋值使成功state+sort响应65535≠200（夹具预设PATCH也受变异，未虚称旧900）；真实Preferences SELECT读另一有效用户使GET200返回错误完整设置，SQL实际生成/恢复；OwnOnly=false使列表total2≠1而详情404/创建403对照仍正确。读隔离变异与T14已验的写范围反例分别记录。
- 初始ProjectCreate错误字段类型、I7误期望required而既有为too_short、no-await-in-loop、闭包数组推断与多余export均如实保留/修复，不计有效业务红。变异后五测试文件仅同序reduce、真实返回类型注解、去export；独立重建原字节SHA与精确cleanup patch证明输入/断言/请求顺序不变，最终完整e2e覆盖整理后的字节，不为哈希变化重做变异。
- 最终lint-web54/54（36 cached）、knip、test-web17/17（全部cached）通过；完整默认make e2e总43.052s、142 passed（runner41.7s，140 Chromium+2 Firefox），八故事各实际匹配一次，旧W11通过且未专查。11交付文件在四个最终门、当前与提交blob完全一致；BASE/HEAD的server/api/web/tools树相同，全部变异源恢复。按G范围复用T14完整Go/race与T12最终性能，无永久生成物改动不重复无关gen-check，临时SQL的真实生成/恢复证据仍齐备。
- 原证据 `task-15-evidence/`、实施报告 `task-15-report.md`、独评 `task-15-review.md`。root以P4/main与T1–14已批准记录承接跨任务约束；本任务批准不代替T16文档/整分支审查或合并授权。

### 分支检查点 CI

- `48dfae8ac42007addf199bead13ccdbb5f3e0d47` 已由系统 `/usr/bin/git` 自主推送；[CI 40190689176](https://github.com/open-nerve/NerveProject/actions/runs/40190689176) 的 server、web、e2e 全部 success。此检查点包含T1–T3及其文档，不包含后续T4，不能冒称T4远端验证。
- `6a94c5ce4abd9a6d1c0f66a9a22ecf365a7a1ff8`（含T4）也已由系统Git推送；[CI 40191919305](https://github.com/open-nerve/NerveProject/actions/runs/40191919305) server、web、e2e全部success。此检查点不包含后续T5a。
- `b12ffbf234b7a914c597fe986d9fd390378b6eac`（含T5a）已由系统Git推送；[CI 40193812287](https://github.com/open-nerve/NerveProject/actions/runs/40193812287) server、web、e2e全部success。此检查点不包含后续T5b。
- `dad6a560a8e0ff509385f9d20a71979c8bddb11a`（含全部T5）已由系统Git推送；[CI 40195404827](https://github.com/open-nerve/NerveProject/actions/runs/40195404827) server、web、e2e全部success。此检查点不包含T6。
- `b550992c4a8babf711befea10bb3d0a26eceaf61`（含T6）已由系统Git推送；[CI 40196872428](https://github.com/open-nerve/NerveProject/actions/runs/40196872428) server、web、e2e全部success。此检查点不包含T7。
- `1c318ddc7dc7f1f0e4b733c081a7018d2883c0a9`（含T7）已由系统Git推送；[CI 40198919892](https://github.com/open-nerve/NerveProject/actions/runs/40198919892) server、web、e2e全部success。此检查点不包含T8。
- `add5f6d9e9e7363d1d7b80b717df38066d50ebe3`（含T8）已由系统Git推送；[CI 40200997593](https://github.com/open-nerve/NerveProject/actions/runs/40200997593) server、web、e2e全部success。此检查点不包含T9。
- `0b599e6860da5212084776abd2d6f6dc4df0acdb`（含T9）已由系统Git推送；[CI 40202959281](https://github.com/open-nerve/NerveProject/actions/runs/40202959281) server、web、e2e全部success。此检查点不包含T10。
- `b733b60186265915587d518b6875b5f9e4aff4b3`（含T10）已由系统Git推送；[CI 40205751190](https://github.com/open-nerve/NerveProject/actions/runs/40205751190) server、web、e2e全部success。此检查点不包含T11。
- `26fa7cf8aa179473d77160fca95d90eea09ec6da`（含T11）已由系统Git推送；[CI 40207373659](https://github.com/open-nerve/NerveProject/actions/runs/40207373659) server、web、e2e全部success。此检查点不包含T12a。
- `c08e1061b2ec5188fd8e1a8f3eb9a22c2f157957`（含T12a）已由系统Git推送；[CI 40210086901](https://github.com/open-nerve/NerveProject/actions/runs/40210086901) server、web、e2e全部success。此检查点不包含T12b及修复。
- `bd5702883aa04cdd98d2a1a23179a9fff4e6e73b`（含T12全部及修复）已由系统Git推送；[CI 40215557907](https://github.com/open-nerve/NerveProject/actions/runs/40215557907) server、web、e2e全部success。此检查点不包含T13a。
- `c4ad379e0fbc752c459999a6952969bd3b487cc8`（含T13a）已由系统Git推送；[CI 40217399252](https://github.com/open-nerve/NerveProject/actions/runs/40217399252) server、web、e2e全部success。此检查点不包含T13b。公开jobs元数据为ignored `ci-40217399252-jobs.json`。
- `ff13a6b0c8a64f49e46c79a29c4e01ea911cd8f4`（含T13全部）已由系统Git推送；[CI 40219840173](https://github.com/open-nerve/NerveProject/actions/runs/40219840173) server、web、e2e全部success。此检查点不包含T14。公开jobs元数据为ignored `ci-40219840173-jobs.json`。
- `97d6ead3cc125007bbc8e286045768e4a2032632`（含T14）已由系统Git推送；[CI 40222281223](https://github.com/open-nerve/NerveProject/actions/runs/40222281223) server、web、e2e全部success。此检查点不包含T15。公开jobs元数据为ignored `ci-40222281223-jobs.json`。
- 公开jobs元数据保存在ignored `ci-40190689176-jobs.json`、`ci-40191919305-jobs.json`、`ci-40193812287-jobs.json`、`ci-40195404827-jobs.json`、`ci-40196872428-jobs.json`、`ci-40198919892-jobs.json`、`ci-40200997593-jobs.json`、`ci-40202959281-jobs.json`、`ci-40205751190-jobs.json`、`ci-40207373659-jobs.json`、`ci-40210086901-jobs.json`、`ci-40215557907-jobs.json`；没有访问受限日志或提取凭据。主目录仍main，P5未合并。

## 3. 执行中的决定

1. 按既有 D2 授权切成16个任务，单列 `sort_order` 全链和普通目录替换，保留原15项的全部验收范围。任务与代价见 spec §3，预计规模与重切条件见 plan。
2. `shared.PageSize`、`ReadOnlySnapshot()` 与 Historical 已在 P4 交付，直接消费；普通 Visible 单独实现，不改变历史引用的宽目录语义。
3. HTTP 保留 M2 绑定先于认证的既定顺序。使用生成器已有标准扩展点，只映射 P5 两条 GET 上真实重复的已声明非日期 query；其他错误回到既有400。原始 query 经 strict middleware 深拷贝传入，不重写 URL、不修改平台或生成器。真实接口矩阵仍待 T9 证明。
4. 个人主页的目标成员与调用者分开进入引擎：关系过滤用目标，权限/OwnOnly 用调用者；目标来自已核验的路径对象，不能由 query 覆盖。
5. M4 §5.1 子项排序残留改回 P3 已裁定并交付的 `(created_at, id)` 升序。P5 排序 PATCH 不改变该集合接口行为。

6. T3不增加只有测试调用的生产全序比较器。纯领域验证真实键归一化、类型、canonical及日期条件保留；null方向、UUID方向、多值min和日期AND/OR的实际变异移至T5真实SQL，T12收口，全部完成要求保留。
7. 游标拼写按spec第2.5节固定：微秒UTC、按排序定型及可空性、精确int64、有限float64及正负零统一。不能把RawMessage直接回写当唯一规范拼写。Flat数值策略已由T5实证；Groups/Subgroups已由T6/T7独立分页证明，最终完整矩阵在T12继续验证。

8. T4没有“活跃成员姓名目录”的生产消费者可供替换；新增依赖或篡改测试来源不能算生产变异。T4用真实关系ID→显示名组合证明退出/降guest仍有姓名，实际变异改取软删除关系检验目录边界；T5在Listing真实消费者再次覆盖。普通active/archived和Historical删除条件仍逐项真实变异。代价是T4不宣称完成一个尚无合法调用点的替代来源变异，姓名来源完成线不减少。

9. T5执行前静态估算1900–2400手写行，超过单任务目标。按D2在同一范围拆成T5a Flat仓储（1050–1350）与T5b精度收口/Listing（850–1300），分别由新实现者交付并独立审查，全部完成后才验收T5。首版必须正确实现数值策略，后续单元深化验证而非接手已知错误。当前16验收任务/19实施单元，估算不是实际diff或测试证据；若再超界继续按D2核对切分，不削测试。

10. T5a按真实sqlc输出采用私有text日期端点与独立名字null位，详见T5a证据；公开契约不变。共享标签样例以合法多项目数据保留固定排序反例，state_name不可构造null与超大count的验证边界明确交接，不改预期来迎合结果。

11. T13按三列CHECK、真实迁移、域值/存储、完整纯校验、闭合API和并发职责静态复核为1835–2590手写行，按既有D2拆成T13a（970–1350）与T13b（865–1240），当前16验收任务/19实施单元。a不留空端口/可选校验占位，b一次接通真实消费与全部API/并发证据；全部T13只在b独立批准后验收。错误代价是跨单元遗漏或返工，以逐项归属、真实API422/恢复/丢更新变异和整Task完成线约束；估计不当运行证据。
12. 执行措辞对齐已采用的SDD：实现验证并精确提交后，以固定BASE..HEAD独立审查，必要修复另提交/复审。T15补测已交付API的故事时如实记录基线；正确行为可基线绿，业务红由指定真实生产变异证明，不为字面“初红”制造错误。两个局部方法调整均不减少产品验收或最终审查门。

T13拆分及上述流程措辞已获一次独立文档预检Approved，无阻断项，记录为ignored `task-13-split-review.md`；预检核对归属、中间提交边界和原验收保留，未运行产品测试，也不代表T13实施通过。Upsert首次INSERT保留显式patch、Create/Ensure才依赖列默认的区别已进一步写明。

13. T12完整分页/快照与真实性能采集的静态职责复核为2080–2770手写行，按D2拆成T12a（1000–1320）与T12b（1080–1450），当前16验收任务/19实施单元。19排序与7/5/42字段矩阵分别覆盖，不擅加笛卡尔乘积；三SQL均须真实排序证据。a负责分页/全部字段/快照/race，b负责实际seed核对、typed捕获/同SQL计划/五样本性能及超预算修复。代价是b优化可能使a证据过时，故b如改产品必须重验受影响正确性/快照并记录最终性能，父T12仅在两单元齐备后验收；当前只是规模估计。

T12拆分已获独立文档预检Approved，无阻断项，记录为ignored `task-12-split-review.md`；原父任务验收全部保留，未运行或宣称T12正确性/性能通过。T11定位正则另按真实函数名修正，补独立project adapter历史目录定位，避免原拟名未匹配旧集合用例。

14. T10关系夹具须保留创建者/负责人自动订阅，三集合不同不等于互斥；通过真实评论构造仅订阅关系，不能假设已有取消订阅API或新增M7能力。先前只读准备的循环订阅表已修正并复核；错误代价是夹具脱离真实API，故使用明确独立集合与关系读回证伪。

15. 最新授权要求P5阶段完成先不合并。T16开发收口（内部审查、文档、最终分支CI）与后续负责人安排的终审/集成分别登记；未做main CI和清理前不勾T16整项，不用这条未来完成线阻断当前已授权开发。

16. 本机默认缓存仍缺Firefox；使用锁定Playwright的显式PLAYWRIGHT_BROWSERS_PATH，仅在P5工作树ignored目录下载headless Chromium1243、Firefox1543和ffmpeg1011，未全局安装、未改依赖/既有缓存/容器。CLI下载exit0；Firefox前两官方URL断链后自动转第三官方CDN成功，原日志在browser-cache-prep。选择的代价是额外工作树磁盘占用，资产不进入代码或过程证据归档；下载不等于浏览器测试通过，后续完整e2e显式使用该目录并保留两个project。

## 4. 预检发现与处理

| 编号/级别 | 原问题 | 修订要求及证据落点 | 当前结论 |
|---|---|---|---|
| M1 / Medium | 目标成员 ID 未进入 Listing/SQL 输入，可能把调用者误当查询目标 | 显式 `ListingInput.TargetMemberID`、`ListRead.TargetMemberID`、独立 `relation_user_id`；T10 的 A≠B 三关系及 OwnOnly 反例/变异 | 已修订，独立复核关闭 |
| M2 / Medium | 新交错只列普通运行，`-race` 被写成可选 | T12/T13 与最终门明确 `-race -count=5`、真实匹配函数及无40P01；包含快照、偏好并发与受影响排序锁路径 | 已修订，独立复核关闭 |
| M3 / Medium | 有词汇表接受测试，未明确全部分组码的真实 SQL/续页覆盖 | T12 覆盖项目7码、个人5码、每个合法子分组字段；独立期望集合、计数、完整数组和翻页 | 已修订，独立复核关闭 |
| M4 / Medium | 缺标准 spec 验收表、局部裁定代价、逐来源交接与方法说明 | 标准7节及附录；验收不打勾；明确静态预检与运行证据的界线 | 已修订，独立复核关闭 |
| L1 / Low | 排序字段文件表漏掉 IssueFields 真正所在的 rows.go | T8 补实际文件路径，不重复声明类型 | 已修订，独立复核关闭 |
| L2 / Low | app fake Rows 无法证明 Store 构造出的 sqlc 数组非nil | T5 核对生产参数映射或真实执行边界，直接NULL的SQL防御另测 | 已修订，独立复核关闭 |
| L3 / Low（复核发现） | 文档修订误把本仓库标准库 `"uuid"` 写成第三方包 | 对照现有 rows.go 等导入，spec/plan恢复标准库；未改产品或依赖 | 已定点复读关闭 |

2026-10-25独立聚焦复核结论 **Approved**；首轮4项Medium、2项Low及修订中新发现的L3全部关闭。批准仅限起步规格与计划，后续任务的产品证据仍待实施。

本轮没有发现需负责人重新裁定的架构问题。SQL数值键的无损转换仍是 T5/T12 必须实证的实施风险，不能将候选 `numeric` 转换写成已验证结果。

## 5. 最终实现与 plan 的差异

阶段尚未结束。T1与计划一致，生产仅增加两项参数；测试额外核对无关application_name保留。T2提前定义共用日期原子解析，T3复用；T3将SQL特有变异留到T5真实查询，样例修正未改变产品行为。T4验证分工与T5规模切分调整见第3节。后续差异随任务登记。

## 6. 移交事项

| 接收方 | 范围 |
|---|---|
| P5 实施者 | T1–T10已完成；从T11继续按spec/plan逐任务交付、测试与评审 |
| P8 | `link_count` 字段与两个排序码；列表/搜索共用词汇 |
| P10b | 共用排序样例、生产前端码点比较器与vitest；旧筛选store和 `/user-properties/` 迁移 |
| P12–P15 | 接口故事增长为页面故事，列表/看板/日历/详情与访客页面 |
| M8 | 大规模与跨项目扫描成本、是否需要排序索引特化；接收最终生产SQL性能证据 |
| M4 收尾 | handoffs仍open，按各阶段处理结果最终关闭；不能用P4完成代替M4整体完成 |

## 7. 已知限制

当前三条列表SQL、公共编排、排序PATCH及项目/个人主页列表API已有上述实施证据；关联目录替换、偏好迁移和100k性能等后续任务尚未完成，不能从已通过结果外推。

默认缓存的Firefox缺失已通过项目本地资产目录准备解决，实际e2e仍待后续任务运行；已知W11偶发按既定纪律处理。各分支检查点的远端完整结果按上文精确SHA登记，不代表未提交的后续任务。
