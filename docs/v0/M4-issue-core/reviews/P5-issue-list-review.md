# M4/P5 工作项列表与显示设置：评审记录

日期：2026-10-25。状态：**实施中；T1–T10完成，T11待开始；阶段尚未完成，不合并**。

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

产品验收清单以 spec §4 为准；T1–T10已有下列实施证据，T11及后续任务尚未完成。下表保留起步检查，后面追加逐任务证据。

| 检查 | 结果 |
|---|---|
| 绿色 main 基线 | 精确 SHA 和三个 CI 任务见上表；没有用 P4 分支 CI 代替 main CI |
| 工作树与依赖 | 从上述 main 创建独立工作树；`pnpm install --frozen-lockfile` exit0，使用现有锁文件，未增加依赖 |
| 本地架构基线 | `go -C server test ./internal/archtest -count=1` exit0，包耗时1.563s；没有放宽守卫 |
| 阶段关键词 | `phase=M4/P5`；没有在 P4/P5 到期的例外，`/user-properties/` 的 M4/P11 例外保留；`node tools/keywords.mjs` 为72规则、23例外、无命中 |
| 起步文档自检 | 16个任务顺序完整、产品验收均未勾选；53处本地链接无新增缺失。M4设计原有4个M5–M8收尾文件链接仍是未来交付，不提前创建或标完成 |
| 独立预检 | 首轮4项 Medium、2项 Low，均为局部计划/文档缺口；修订聚焦复核Approved，新增L3也已关闭，见§4 |
| P5 产品测试、性能、交错及变异 | T1–T10证据见下文；后续任务不能以起步检查或已完成任务结果代替 |

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

### 分支检查点 CI

- `48dfae8ac42007addf199bead13ccdbb5f3e0d47` 已由系统 `/usr/bin/git` 自主推送；[CI 40190689176](https://github.com/open-nerve/NerveProject/actions/runs/40190689176) 的 server、web、e2e 全部 success。此检查点包含T1–T3及其文档，不包含后续T4，不能冒称T4远端验证。
- `6a94c5ce4abd9a6d1c0f66a9a22ecf365a7a1ff8`（含T4）也已由系统Git推送；[CI 40191919305](https://github.com/open-nerve/NerveProject/actions/runs/40191919305) server、web、e2e全部success。此检查点不包含后续T5a。
- `b12ffbf234b7a914c597fe986d9fd390378b6eac`（含T5a）已由系统Git推送；[CI 40193812287](https://github.com/open-nerve/NerveProject/actions/runs/40193812287) server、web、e2e全部success。此检查点不包含后续T5b。
- `dad6a560a8e0ff509385f9d20a71979c8bddb11a`（含全部T5）已由系统Git推送；[CI 40195404827](https://github.com/open-nerve/NerveProject/actions/runs/40195404827) server、web、e2e全部success。此检查点不包含T6。
- `b550992c4a8babf711befea10bb3d0a26eceaf61`（含T6）已由系统Git推送；[CI 40196872428](https://github.com/open-nerve/NerveProject/actions/runs/40196872428) server、web、e2e全部success。此检查点不包含T7。
- `1c318ddc7dc7f1f0e4b733c081a7018d2883c0a9`（含T7）已由系统Git推送；[CI 40198919892](https://github.com/open-nerve/NerveProject/actions/runs/40198919892) server、web、e2e全部success。此检查点不包含T8。
- `add5f6d9e9e7363d1d7b80b717df38066d50ebe3`（含T8）已由系统Git推送；[CI 40200997593](https://github.com/open-nerve/NerveProject/actions/runs/40200997593) server、web、e2e全部success。此检查点不包含T9。
- `0b599e6860da5212084776abd2d6f6dc4df0acdb`（含T9）已由系统Git推送；[CI 40202959281](https://github.com/open-nerve/NerveProject/actions/runs/40202959281) server、web、e2e全部success。此检查点不包含T10。
- 公开jobs元数据保存在ignored `ci-40190689176-jobs.json`、`ci-40191919305-jobs.json`、`ci-40193812287-jobs.json`、`ci-40195404827-jobs.json`、`ci-40196872428-jobs.json`、`ci-40198919892-jobs.json`、`ci-40200997593-jobs.json`、`ci-40202959281-jobs.json`；没有访问受限日志或提取凭据。主目录仍main，P5未合并。

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
