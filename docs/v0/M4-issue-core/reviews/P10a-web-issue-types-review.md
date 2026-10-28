# M4/P10a 工作项换型与列表收窄：实施与评审记录

日期：2026-10-28。**实施与终审完成、集成待办；最终文稿提交后准确HEAD CI以交付回执核验。** 工作树`codex/m4-p10a`，主项目HEAD保持main。基线main `4599a9cfc5be90b2c242977f194157778a5f6ec9`；最后产品提交`5f7085bce4aa5103a0133fa982847d274efa6e47`，tree `d0b4ea997e3dd7cdd61e9cbc662b75b064636364`。T1直接父`4569a9e7386031cffb31aa37d6cf06a726b08c2c`含准备文档，整审从main基线覆盖全部。

配套：[spec](../specs/P10a-web-issue-types.md)、[plan](../plans/P10a-web-issue-types.md)、[整体审核（实施与终审完成）](../../../reviews/re-11-M4P10a整体审核.md)、[03/04当前接续](../../claude_to_gpt/04-next-steps.md#12-p10a实施后接续2026-10-28)。准备期[re-10](../../../reviews/re-10-M4P10预检审核.md)不代表产品终审。root已获分支CI推送授权并系统git推T8，808产品检查点CI已核全绿，最后修复产品CI已核成功，文档最终交付HEAD CI待核；无人获本阶段合并授权。不合并、不清理分支/worktree/X。

## 1. 方法、身份与证据边界

逐Task独立要求/质量评审、根因修复/定向复审后只提交已审blob；每轮source/manifest/diff/checks/raw/exit/pre及首失败保留。X指本工作树`.superpowers/sdd/P10a-web-issue-types`，下文列关键原件名供未来归档，但正文自足记身份、结果、限制和移交，不以本机ignored链接代替结论。准备期原件另在主项目`.superpowers/sdd/P10-preparation-20261028/`及decision-a，未来归档不可遗漏。

主控A/B已完成原代码包全文及A-01/A-02修复独立复审；C已完成round0的22路径全文及guard增量，C r1已读14份增量，唯一Minor C-r1-01经C r2复审ADDRESSED，无新增。最后产品ready集合355路径=A93/B240/C22，root已核333代码唯一路径/54475全文行0遗漏，最后文稿22路径/hash桥仍由root合并；root另补最终删除helper的main原220行全文，与r2前222行不同源（final-root-deleted-base-coverage.json）；代码无剩余发现，最后文稿已按C-r2真实复审通过，不以旧结论外推。本文§12保留准备期历史原文，当时“当前T1/禁止push”等不作为当前授权或状态。

## 2. 验收标准核对

| spec §4 | 当前实际证据 | 状态 |
|---|---|---|
| 1 编译/生成/依赖 | T1生成前后指纹相同；T1–T8末轮独评及精确提交，R14仅删除无读者依赖 | 产品子项与文稿独评已核，提交后交付CI另待 |
| 2 核心类型/消费者 | T6生成合同Approved；T9在5f7085bc逐路径核132/50，旧三名/四枚举零命中，114直接generated，见spec附录B | 产品静态核、代码与文稿复审完成 |
| 3 四类列表/独占链 | T2/T3、T5最后写链、T7 r1漏清理均闭合，T8真实六入口列表/动作/请求移除 | Task级完成 |
| 4 null/Detail/表单 | T4 R8、T5 R10/R15与两轮修复、T6 generated/描述/nullable、T7最终progress | Task级完成 |
| 5 回归/guard/dead | T6描述合并/R16变异；T7三类guard红绿；最终原脚本触及236项/61文件，保留既存候选责任见§9 | Task级+本轮静态，非全仓零dead |
| 6 S2/六入口/领域 | 原S2四身份10/10、新6/6，八最低项与75API及有效实体组件分层见§6 | 拆除边界完成，M6/M7整页未完成 |
| 7 正文/移交/open | spec/plan/设计/两差异/五输入/03/04本轮落实 | C原22全文无finding；r1增量唯一C-r1-01已修，r2 Approved |
| 8a 产品完成线 | 最后本地F/六web桥、A/B代码整审与两项复审、准确5f708产品CI全绿 | 已完成，spec产品子项勾选 |
| 8b 文稿复审 | C-r1-01经C-r2复审ADDRESSED，无新增；正文移交及文稿复审完成 | 已完成，spec具名子项勾选 |
| 8c 最终交付 | 文稿提交后root须核准确交付HEAD CI并生成final-delivery-receipt.json | 提交时尚未发生，不勾；必须最终文稿HEAD CI全绿才结束任务 |
| 9 merge/mainCI/归档清理 | 本期无集成授权/执行 | 保持开放，不勾 |

本地F、分支CI、未来mainCI是不同事实。M4§14/总体§9.4、五输入总status=open、M5–M8最终closeout保持，不因本表部分满足关闭M4。

## 3. Task实施、独评与修复

下表提交身份、连续父链和路径数采用 root 已核实际 Git 对象的 `task-9-T1-T7-verified-commits.json`；它不替代每 Task 的验证范围。各 `task-N-commit-receipt.json` 保留逐 blob 与已审 manifest 对照，完整原件按 X 证据索引查阅并在未来归档时逐 hash 验证。

| Task | 起点 → 最终提交 | 路径数 | 独评与修复结论 | 验证边界 |
|---|---|---:|---|---|
| T1 | `4569a9e7386031cffb31aa37d6cf06a726b08c2c` → `e4a3dfa351c0ebef86b4f7d95571ede392844507` | 6 | 首轮I1把普通issue-links误归M5；round1修为P11/M4/P12，`task-1-round1-rereview.md` Approved | 生成/幂等/types/关键词等；不是完整F |
| T2 | `e4a3dfa351c0ebef86b4f7d95571ede392844507` → `5f05a71500cb1f9a40802a1511d629d1d6c19b45` | 80 | F1目录变化未重判缺失态、F2预览多发普通详情请求均修复；`task-2-round1-rereview.md` Approved | 最后3文件8测及types/lint/knip；组件有效领域fixture不冒充HTTP接通 |
| T3 | `5f05a71500cb1f9a40802a1511d6c19b45` → `11cd21369c72ce64a8e5f7e3e2dea8cfabd986b2` | 61 | `task-3-review.md` Spec✅/Approved，无待修产品发现 | 完整F；首次Firefox缺artifact属基础设施失败，安装锁定artifact后只重跑F7 |
| T4 | `11cd21369c72ce64a8e5f7e3e2dea8cfabd986b2` → `d9415375695eea18e72a51b45c6b89fd716892de` | 68 | `task-4-review.md` Spec✅/Approved；T4-M1由本期稳定证据锚及正文链接处理 | 35focused、types/lint/knip/test-web；未完整F/E2E |
| T5 | `d9415375695eea18e72a51b45c6b89fd716892de` → `bd862bf2f5e740f436d2aa750ef3c031f8d1beaa` | 96 | 首轮I1/I2/I3在r1均ADDRESSED；r1新N1在r2 ADDRESSED；`task-5-r2-rereview.md` Spec✅/Approved | r2最终14 UI/clone + types/lint/knip；r1资源/R10与r0 full-suite分账；未完整F |
| T6 | `bd862bf2f5e740f436d2aa750ef3c031f8d1beaa` → `e5031c0704e09998422ec839dfffe47efb482369` | 134 | 唯一I1是真实project/workspace测试夹具不一致，r1限一测试文件8处issueOf修正；`task-6-r1-rereview.md` Approved | F1–F6到r1精确2个非生产差异；定向验证补齐，见下文源桥 |
| T7 | `e5031c0704e09998422ec839dfffe47efb482369` → `fdd3ef2ee48995f47b25144424369de5d5496f17` | 45 | G7-1死callback及最后依赖、G7-2根issue只写链均修复；旧ledger误判明确撤回；`task-7-r1-rereview.md` Approved | 44现有测试及最终static；测试后两占位+cap三路径桥，未完整F |
| T8 | `fdd3ef2ee48995f47b25144424369de5d5496f17` → `8081112492a2fc744a43b3e814c6d36ff4ba3959` | 7 | 原G Approved；唯一Minor按R19加固后 `task-8-r1-rereview.md` Spec✅/Approved；root逐blob核已提交 | 原S2四身份10/10、新六入口6/6、领域5文件12测、T5/T6定向10文件76测；round0 F6 1277/F7 169与r1定向/静态按§8源桥分账 |
| 整审修复 | `8081112492a2fc744a43b3e814c6d36ff4ba3959` → `5f7085bce4aa5103a0133fa982847d274efa6e47` | 9 | A-01 r1/A-02 r2均Approved，C guard delta Approved；root精确提交 | 本轮最后F通过，F1–3与最终六web删除差异，F4–7同源；产品CI run40712479050 attempt1三job成功；最终文稿CI另待 |
| T9文稿 | 产品5f7085bce4aa5103a0133fa982847d274efa6e47 +本轮14文档 → 待提交 | 14 | 原C22全文无finding；r1增量唯一Minor已修，C-r2 Approved无新增 | 最后文档提交/准确HEAD CI未发生，不预填 |

T5 的 I1 是新增详情资源拒绝未由真实 Base/FormRoot 拥有，导致 unhandled/永久loading；修复加入可重试错误状态、提交/move/草稿保存保护以及关闭/切来源后旧结算退休。I2 删除 Base 八个普通 cycle/module 写方法、失去最后读者的服务方法和草稿空壳；真实草稿方法保留。I3 把两retrieve入口资源编排收为类内唯一私有helper。r1的新N1不是仍未修的I1：retry成功HTML只写RHF、未通知DraftWrapper已有快照，最终保存漏描述；r2以实际非空HTML和空串载荷红绿闭合。最终真实wrapper保存断言在既有hook/store边界捕获，不声称真实draft HTTP。

T6 的 I1 只修 quick-add 测试夹具事实链，不是生产跨workspace缺陷；projectA/projectB、迟到A响应、toast action及prefill各取所属project.workspace_id，11个既有行为断言保持。T7 G7-1删peek操作对象remove/archive及其dep-only关闭函数/pathname/is_draft链，并删最后暴露的archived/draft archive占位；真实IssueView关闭与archive/restore能力保留。G7-2删根TIssueCrudState.issue及两条只写输入链，保留独立sub-issues/content编辑状态。原ledger“spread/模态会读取根issue”的理由错误，明确撤回；**T7的r1账225条/59文件替代其错误r0账，808时T9增量227为历史，最后236条见§9**。

## 4. T4证据与限制

<a id="t4-r8-evidence"></a>

### T4/R8：quick-add 输入与真实实体分离的证据

T4 已提交 `d9415375695eea18e72a51b45c6b89fd716892de`，父提交 `11cd21369c72ce64a8e5f7e3e2dea8cfabd986b2`。`task-4-review.md` 对冻结source `b03f02e34121a376d415d28606019c45f93fad36bc57415914e7d3a46e16f6f7` 给出 Spec✅/Approved、无Critical/Important；source manifest SHA256为 `6356385204b34fa715b8e2c1b230406d4b080b05801accc4c725ac7c5c055d53`，`task-4-commit-receipt.json` 核68路径提交blob与该已审源一致。

创建请求结算前，输入只留在创建入口，原生fieldset禁用及独立pending提示保持；第二个真实RHF handler结束无权释放首请求pending。失败/undefined保留输入供重试，关闭guard与同步busy ref共同阻重复/关闭；workspace/project key隔离输入实例，晚到A不清B输入或插B分组，toast action仍捕获A。真实响应才由既有store路径入canonical及正式分组/计数各一次。可见代价是等待期不再以普通工作项卡片参与排序、分组和计数，预览留在quick-add位置；没有新增全局pending引擎、P10b确认队列或版本协调。

T4最终同源6文件35项行为/回归、types、完整lint、knip、强制test-web和diffcheck均exit0，详见X的 `task-4-report.md`、`task-4-checks.json`、各raw/exit/pre及manifest；test-web为18/18任务、0cache，成功包详情被抑制，不杜撰包用例总数或无警告。T4未跑完整F/E2E，T6只机械接生成核心并修测试workspace事实，后续最终阶段证据另列。

R8“先同步正文”的早期全文仅部分可证：red-2记录quick-add/root仍为base、plan已为新文，另四份文档当时仅有hash且随后有状态/尺寸更新，不能把最终diff当完整早期全文/操作先后证明。最早state红只有可重建helper/test子集，不冒充完整pre-run tree。`spreadsheet-table.tsx`经独评核为通用列传递无需改动，不是漏改。完整原件尚未归档，当前原件保留于X，未来另授权集成后记录实际归档及逐hash核验回执，稳定正文入口保留本节。

## 5. 实施期局部裁定

| 裁定 | 最终已核范围 | 代价/未完成责任 | 既有证据 |
|---|---|---|---|
| R8 | quick-add局部等待，不完整输入不进canonical/group/count，首请求pending及scope隔离 | 短暂视觉差异；P10b协调未实现 | 上方T4稳定锚；T4红绿/35focused；T6真实workspace夹具修正 |
| R10 | create描述输入`Partial<IssueDetail>`；response为`Issue \| IssueDetail`；update边界独立`TLegacyIssueUpdateRequest = Partial<IssueDetail> & { skip_activity?: string }`；metadata仍所属DTO | 旧body/string值与写请求次数不变，未接通旧后端；M7到期不延长。M4新强类型API实际接入时闭合所需boolean，不借P14延命 | T5原exact三文件25测；T6同三文件机械换生成后25测；canonical/分组/rollback不带flag，absent-key真正删除；accept只保留单个已知HTML且响应空串优先 |
| R11 | `/search/`不受既有plane-workspace-urls或新增八类守卫覆盖；广义手工账保留P15删除/until M4/P16 | 无自动guard，不加第九类/不改M3语义 | T7最终spec修正、广义逐方法账及matcher正反例 |
| R12 | 同file+match exactkey按最早until：create/list以及patch/retrieve均先M4/P11；patch实际Base.issueUpdate归P10b | P10b删早期方法后重核真实剩余count，才可把P11 retrieve另登M4/P12；3个Base HTTP测试命中也随P10b | T7 40hits/12files/28keys最终账；不沿用历史count3的create/list |
| R13 | T5五字段局部化是中间态；T7核最后getter无读者后删getter、id/name/size/type和tempId，最终只留真实progress | 上传请求/进度回调/finally/TFileHandler保持；不宣称M5上传协议已重写/验收 | T7全web实际读者核验、精确恢复hash及最终static |
| R14 | 删除无读者react-dropzone、web依赖、废弃catalog及孤立attr-accept/file-selector锁条目；锁diff仅删除 | 不加包/升级，独立editor合法依赖保持；不是“全部依赖不变” | T5全workspace读者核、离线frozen-lockfile一致性、最终knip |
| R15 | 两retrieve入口完整core入canonical一次；原reaction/link DTO进自身cache，非null父按响应project_id raw retrieve，不递归、不猜null父；唯一私有helper | 目标链为3 GET，非null父4 GET，其他历史side-load另计；成功promise等待新增资源，拒绝由真实Base提示/重试。历史activity/comment/subissue/relation/state detached不泛修 | T5两入口12真实store/service HTTP样本与I1真实Base回归；T6原样机械接续 |
| R16 | 删列表初始/翻页extractRelationsFromIssues两调用及extractor接口/action/实现；详情真实fetchRelations保留 | 无列表资源预热，详情依所属资源到达；P11数据层/历史detached拒绝边界不提前完成 | T6真实HTTP两retrieve×初始/翻页四矩阵；恢复旧extractor变异4红、精确恢复全文件16绿 |
| R17 | I10观察夹具`SELECT id::text ORDER BY id`按输出text别名排9/10；仅改`ORDER BY river_job.id` | 非生产crawler/W11；不扩修同式但无该顺序断言的其他夹具 | 首次F7 162/1失败报告/trace/DB；独立SQL旧10,9/新9,10探针；受影响e2e静态与完整F7 163绿 |
| R18 | CycleService新transport-boundary测试的analytics字面量按单文件/match/count1/until M6登记；规则不扩 | 原F4首次exit2保留，最终73规则/52例外；old-issue-urls仍28keys/40hits，另账 | T8原G已核精确登记与原件 |
| R19 | 新故事finally先落核心raw再有界1000ms采title/DOM，诊断错误单列；不catch真实expect/I/O | 原/新双故障探针内层均预期exit1；wrapper性质核验exit0不叫正常绿。正常6/6与e2e static最终同源 | T8 r1独评Approved，原633项artifact保持；未重跑1277/169 |

R10 的 `Partial<IssueDetail>` 是输入/patch，不能据此把partial变成完整canonical实体，不能恢复string flag到Issue/IssueDetail。描述读取只经真实Detail判别，真实空HTML是有效Detail；后到Issue行保留既有Detail描述。nullable作者/父项不做非空断言，不补虚假默认。


## 6. 移交事项

P10a已删除四类工作项列表及独占交互，保留领域自身页头/详情/目录加载；这是两项分别存在的可见代价：M6/M7接通前既没有这些工作项列表，也不保证领域自身名称、详情、统计或目录可取得。T8拆除边界与M6/M7自身整页完整可用/零失败是不同完成线。保留组件或一次404不得表述成有效实体页面已验收；接收方接通时替换旧接口、删除未接通过渡记录，以真实有效实体重新验收详情、统计、目录和列表。

| 接收方 | 本期明确交付/保留事实 | 接收动作与期限 |
|---|---|---|
| P10b | 直接generated核心、旧列表容器只换行；旧create/list/patch/delete、archive、preferences/filter等仍有界过渡；DEFAULT保留项目通用创建/详情命令能力 | 只能在P10a真实main集成且准确mainCI通过后，重核spec§2.3、实际文件/消费者及全部未完成过渡；完成新列表数据层/21码/版本协调与S2增长，按最早M4/P11删相关exactkey，重数剩余P11方法 |
| P11 | 详情核心形状已收窄；reaction/link/parent及relation等资源仍为自身DTO/旧service | 迁真实详情数据层/其余DTO、JSONContent/comment_json/is_bot、模块实例及原按需拒绝责任；普通issue-links四方法属P11/M4/P12，非M5；不把R15/I1局部拒绝修复称全局已完成 |
| M5 | M4普通实体不再读附件计数、详情附件挂载和独占链已删；editor独立上传只留局部progress | 随M5真实文件数据恢复attachment_count/显示/附件区域等；本期不交付M5接口或上传协议验收 |
| M6 | 删cycle/module两个布局根、列表/筛选store与CYCLE/MODULE分支、详情/归档入口列表、group/calendar添加已有分支、专空态、活动迭代/进度侧栏列表调用、transfer及普通Base最后cycle/module写链；领域详情/统计/目录取数保留 | 在新列表引擎加回列表/筛选/控件及Issue字段写入，保留M4共享ExistingIssues选择器；按本节实际请求记录改接领域API，并以有效实体核详情/统计/归档目录完整可用与整页零失败 |
| M7 | 删项目/工作区视图工作项布局根、列表/筛选store及PROJECT_VIEW/GLOBAL分支，保留视图领域CRUD/目录/名称；收集箱准确输入/响应/独立旧update请求已由T5/T6闭合 | 视图在新引擎重建并验真实有效实体整页；收集箱create PartialDetail、response union、metadata所属DTO、旧stringflag只在request边界，旧路径until M7，接自身API时删过渡与最后旧request type；不将来源/通知等未实施事项关闭 |
| P12–P16/M4收尾 | 本期未接通完整页面、新搜索/协作等后续行为；T7的225、808的227均历史；最后236项/61文件含新纳入15既存领域候选，具体责任见§9，非全仓无dead | P12–P15各页面/搜索、P16守卫终态/全仓no-shadow及最终死成员核；M4收尾核输入关闭条件并统一创建M5–M8最终M4-closeout |

### 六入口与八项自身请求的分层证据

本节取已提交T8 r1的正常六入口矩阵，不混用round0端口/实体。运行host为`http://127.0.0.1:51952`，75条API请求=69父200+6自身404，均已结束、无requestfailure；六页pageErrors/CSP均0，console resource error逐条精确绑定唯一领域404的完整URL，emoji/probe前后原数组保留。原watchPage/S2/mounts逐字未改，不清数组、不泛放行旧前缀/4xx；原始requestUrls/apiRequests/oldApiRequests/apiFailures/pageErrors/console/CSP及旁路/DOM附件保留在X的`task-8-r1-normal-test-results`，矩阵`task-8-r1-request-matrix.json`。

三个缺失详情到达真实Cycle/Module/ProjectView Page及页头，显示各自does-not-exist，而非Join/权限重定向；默认workspace all-issues真实子Page有title。两归档是project detail layout的兄弟路由，真实Archives项目链接、自身breadcrumb/Search/GET及单个loader证明挂载。API新建project默认cycle_view/module_view=false，ArchiveTabsList不显示两领域tab，未为测试伪开feature/实体。领域404后fetchedMap未置位、ids为null，归档持续loader，不能称空目录成功。

| 最低项 | 实际调用者/触发 | 本轮网络与验证层 | 接收/删除期限 |
|---|---|---|---|
| cycle详情 | Page CYCLE_DETAILS → CycleStore.fetchCycleDetails → CycleService.getCycleDetails；有效project成员直接缺失ID | 下表完整URL，GET404×1 | M6接通详情时 |
| module详情 | Page CURRENT_MODULE_DETAILS → ModulesStore.fetchModuleDetails → ModuleService.getModuleDetails | GET404×1 | M6接通详情时 |
| project-view详情 | Page VIEW_DETAILS → ProjectViewStore.fetchViewDetails → ViewService.getViewDetails | GET404×1；真实header+Page组件另核详情一次，旧filter两store已删，不能只靠共享URL放行 | M7接通视图时 |
| workspace views目录 | layout WORKSPACE_GLOBAL_VIEWS → GlobalViewStore.fetchAllGlobalViews → WorkspaceService.getAllViews；默认all-issues也触发 | GET404×1，默认Outlet照常挂载；单view URL不在允许集合 | M7接通目录时 |
| archived-cycles目录 | ArchivedCycleLayoutRoot SWR → CycleStore.fetchArchivedCycles → CycleArchiveService.getArchivedCycles | GET404×1，持续loader | M6接通目录时 |
| archived-modules目录 | ArchivedModuleLayoutRoot SWR → ModulesStore.fetchArchivedModules → ModuleArchiveService.getArchivedModules | GET404×1，持续loader | M6接通目录时 |
| cycle analytics | Page/Sidebar → useCyclesDetails DURATION → fetchActiveCycleDistribution → CycleService.cycleDistribution；有效实体无distribution时 | **真实页面网络0，状态未测**；有效store组件各一次，真实APIService interceptor/拒绝adapter核`/api/workspaces/ws/projects/project/cycles/cycle/analytics/?type=issues`，不是有效实体E2E成功 | M6有效实体统计网络/整页验收 |
| cycle progress | useCyclesDetails PROGRESS → fetchActiveCycleProgress → CycleService.cycleProgress | **真实页面网络0，状态未测**；有效store组件一次，service边界核`/api/workspaces/ws/projects/project/cycles/cycle/progress/`，无query | M6有效实体进度网络/整页验收 |

有效cycle/module/project-view/custom-workspace-view仅真实组件与store fixture覆盖，未造SQL实体/route.fulfill成功。五文件12测保留标题/描述/侧栏、目录item/search和领域装载；archive有效cache+刷新失败走真实reject/catch复位loader，非手写loader/假成功，生命周期与StoreWrapper同序。旧列表service零调用的组件断言不能代替全网络集合；完整网络来自六入口。下表每行一次，完整保留query与尾斜杠；analytics/progress未进入这75条。

| 入口 | Method / 完整实际URL | 次数 / 状态 / 终态 |
|---|---|---|
| workspace view | `POST http://127.0.0.1:51952/api/v0/auth/refresh` | 1 / 200 / ended |
| workspace view | `GET http://127.0.0.1:51952/api/v0/instance` | 1 / 200 / ended |
| workspace view | `GET http://127.0.0.1:51952/api/v0/me` | 1 / 200 / ended |
| workspace view | `GET http://127.0.0.1:51952/api/v0/me/profile` | 1 / 200 / ended |
| workspace view | `GET http://127.0.0.1:51952/api/v0/workspaces` | 1 / 200 / ended |
| workspace view | `GET http://127.0.0.1:51952/api/workspaces/w-f525c48cd5bf1635/views/` | 1 / 404 / ended |
| workspace view | `GET http://127.0.0.1:51952/api/v0/workspaces/w-f525c48cd5bf1635/members` | 1 / 200 / ended |
| workspace view | `GET http://127.0.0.1:51952/api/v0/me/workspaces/w-f525c48cd5bf1635/preferences` | 1 / 200 / ended |
| workspace view | `GET http://127.0.0.1:51952/api/v0/workspaces/w-f525c48cd5bf1635/projects?archived=false` | 1 / 200 / ended |
| workspace view | `GET http://127.0.0.1:51952/api/v0/workspaces/w-f525c48cd5bf1635/states` | 1 / 200 / ended |
| module | `POST http://127.0.0.1:51952/api/v0/auth/refresh` | 1 / 200 / ended |
| module | `GET http://127.0.0.1:51952/api/v0/instance` | 1 / 200 / ended |
| module | `GET http://127.0.0.1:51952/api/v0/me` | 1 / 200 / ended |
| module | `GET http://127.0.0.1:51952/api/v0/me/profile` | 1 / 200 / ended |
| module | `GET http://127.0.0.1:51952/api/v0/workspaces` | 1 / 200 / ended |
| module | `GET http://127.0.0.1:51952/api/v0/projects/01a182d0-f11e-738d-b4e3-49124cb341d3` | 1 / 200 / ended |
| module | `GET http://127.0.0.1:51952/api/v0/workspaces/w-6f85a7246732d665/members` | 1 / 200 / ended |
| module | `GET http://127.0.0.1:51952/api/v0/me/workspaces/w-6f85a7246732d665/preferences` | 1 / 200 / ended |
| module | `GET http://127.0.0.1:51952/api/v0/workspaces/w-6f85a7246732d665/projects?archived=false` | 1 / 200 / ended |
| module | `GET http://127.0.0.1:51952/api/v0/workspaces/w-6f85a7246732d665/states` | 1 / 200 / ended |
| module | `GET http://127.0.0.1:51952/api/workspaces/w-6f85a7246732d665/projects/01a182d0-f11e-738d-b4e3-49124cb341d3/modules/0199f1c2-7a1b-7c3d-8e4f-5a6b7c8d9e0f/` | 1 / 404 / ended |
| module | `GET http://127.0.0.1:51952/api/v0/me/projects/01a182d0-f11e-738d-b4e3-49124cb341d3/preferences` | 1 / 200 / ended |
| module | `GET http://127.0.0.1:51952/api/v0/projects/01a182d0-f11e-738d-b4e3-49124cb341d3/labels` | 1 / 200 / ended |
| module | `GET http://127.0.0.1:51952/api/v0/projects/01a182d0-f11e-738d-b4e3-49124cb341d3/members` | 1 / 200 / ended |
| module | `GET http://127.0.0.1:51952/api/v0/projects/01a182d0-f11e-738d-b4e3-49124cb341d3/states` | 1 / 200 / ended |
| cycle | `POST http://127.0.0.1:51952/api/v0/auth/refresh` | 1 / 200 / ended |
| cycle | `GET http://127.0.0.1:51952/api/v0/instance` | 1 / 200 / ended |
| cycle | `GET http://127.0.0.1:51952/api/v0/me` | 1 / 200 / ended |
| cycle | `GET http://127.0.0.1:51952/api/v0/me/profile` | 1 / 200 / ended |
| cycle | `GET http://127.0.0.1:51952/api/v0/workspaces` | 1 / 200 / ended |
| cycle | `GET http://127.0.0.1:51952/api/v0/projects/01a182d0-ef81-7e93-97e2-dadf3363bac2` | 1 / 200 / ended |
| cycle | `GET http://127.0.0.1:51952/api/v0/workspaces/w-68f7538e5353cb61/members` | 1 / 200 / ended |
| cycle | `GET http://127.0.0.1:51952/api/v0/me/workspaces/w-68f7538e5353cb61/preferences` | 1 / 200 / ended |
| cycle | `GET http://127.0.0.1:51952/api/v0/workspaces/w-68f7538e5353cb61/projects?archived=false` | 1 / 200 / ended |
| cycle | `GET http://127.0.0.1:51952/api/v0/workspaces/w-68f7538e5353cb61/states` | 1 / 200 / ended |
| cycle | `GET http://127.0.0.1:51952/api/workspaces/w-68f7538e5353cb61/projects/01a182d0-ef81-7e93-97e2-dadf3363bac2/cycles/0199f1c2-7a1b-7c3d-8e4f-5a6b7c8d9e0f/` | 1 / 404 / ended |
| cycle | `GET http://127.0.0.1:51952/api/v0/me/projects/01a182d0-ef81-7e93-97e2-dadf3363bac2/preferences` | 1 / 200 / ended |
| cycle | `GET http://127.0.0.1:51952/api/v0/projects/01a182d0-ef81-7e93-97e2-dadf3363bac2/labels` | 1 / 200 / ended |
| cycle | `GET http://127.0.0.1:51952/api/v0/projects/01a182d0-ef81-7e93-97e2-dadf3363bac2/members` | 1 / 200 / ended |
| cycle | `GET http://127.0.0.1:51952/api/v0/projects/01a182d0-ef81-7e93-97e2-dadf3363bac2/states` | 1 / 200 / ended |
| project view | `POST http://127.0.0.1:51952/api/v0/auth/refresh` | 1 / 200 / ended |
| project view | `GET http://127.0.0.1:51952/api/v0/instance` | 1 / 200 / ended |
| project view | `GET http://127.0.0.1:51952/api/v0/me` | 1 / 200 / ended |
| project view | `GET http://127.0.0.1:51952/api/v0/me/profile` | 1 / 200 / ended |
| project view | `GET http://127.0.0.1:51952/api/v0/workspaces` | 1 / 200 / ended |
| project view | `GET http://127.0.0.1:51952/api/v0/projects/01a182d0-f2ac-7d2f-b38f-63f04f222af0` | 1 / 200 / ended |
| project view | `GET http://127.0.0.1:51952/api/v0/workspaces/w-1ee7f2c04de2a3a2/members` | 1 / 200 / ended |
| project view | `GET http://127.0.0.1:51952/api/v0/me/workspaces/w-1ee7f2c04de2a3a2/preferences` | 1 / 200 / ended |
| project view | `GET http://127.0.0.1:51952/api/v0/workspaces/w-1ee7f2c04de2a3a2/projects?archived=false` | 1 / 200 / ended |
| project view | `GET http://127.0.0.1:51952/api/v0/workspaces/w-1ee7f2c04de2a3a2/states` | 1 / 200 / ended |
| project view | `GET http://127.0.0.1:51952/api/workspaces/w-1ee7f2c04de2a3a2/projects/01a182d0-f2ac-7d2f-b38f-63f04f222af0/views/0199f1c2-7a1b-7c3d-8e4f-5a6b7c8d9e0f/` | 1 / 404 / ended |
| project view | `GET http://127.0.0.1:51952/api/v0/me/projects/01a182d0-f2ac-7d2f-b38f-63f04f222af0/preferences` | 1 / 200 / ended |
| project view | `GET http://127.0.0.1:51952/api/v0/projects/01a182d0-f2ac-7d2f-b38f-63f04f222af0/labels` | 1 / 200 / ended |
| project view | `GET http://127.0.0.1:51952/api/v0/projects/01a182d0-f2ac-7d2f-b38f-63f04f222af0/members` | 1 / 200 / ended |
| project view | `GET http://127.0.0.1:51952/api/v0/projects/01a182d0-f2ac-7d2f-b38f-63f04f222af0/states` | 1 / 200 / ended |
| archived cycles | `POST http://127.0.0.1:51952/api/v0/auth/refresh` | 1 / 200 / ended |
| archived cycles | `GET http://127.0.0.1:51952/api/v0/instance` | 1 / 200 / ended |
| archived cycles | `GET http://127.0.0.1:51952/api/v0/me` | 1 / 200 / ended |
| archived cycles | `GET http://127.0.0.1:51952/api/v0/me/profile` | 1 / 200 / ended |
| archived cycles | `GET http://127.0.0.1:51952/api/v0/workspaces` | 1 / 200 / ended |
| archived cycles | `GET http://127.0.0.1:51952/api/workspaces/w-29f7e4468a3bfc93/projects/01a182d0-f5bc-7e45-9640-ad7b358afc83/archived-cycles/` | 1 / 404 / ended |
| archived cycles | `GET http://127.0.0.1:51952/api/v0/workspaces/w-29f7e4468a3bfc93/members` | 1 / 200 / ended |
| archived cycles | `GET http://127.0.0.1:51952/api/v0/me/workspaces/w-29f7e4468a3bfc93/preferences` | 1 / 200 / ended |
| archived cycles | `GET http://127.0.0.1:51952/api/v0/workspaces/w-29f7e4468a3bfc93/projects?archived=false` | 1 / 200 / ended |
| archived cycles | `GET http://127.0.0.1:51952/api/v0/workspaces/w-29f7e4468a3bfc93/states` | 1 / 200 / ended |
| archived modules | `POST http://127.0.0.1:51952/api/v0/auth/refresh` | 1 / 200 / ended |
| archived modules | `GET http://127.0.0.1:51952/api/v0/instance` | 1 / 200 / ended |
| archived modules | `GET http://127.0.0.1:51952/api/v0/me` | 1 / 200 / ended |
| archived modules | `GET http://127.0.0.1:51952/api/v0/me/profile` | 1 / 200 / ended |
| archived modules | `GET http://127.0.0.1:51952/api/v0/workspaces` | 1 / 200 / ended |
| archived modules | `GET http://127.0.0.1:51952/api/workspaces/w-17ebc32046f4a312/projects/01a182d0-f73c-7d0e-85df-c18d6e5a5337/archived-modules/` | 1 / 404 / ended |
| archived modules | `GET http://127.0.0.1:51952/api/v0/workspaces/w-17ebc32046f4a312/members` | 1 / 200 / ended |
| archived modules | `GET http://127.0.0.1:51952/api/v0/me/workspaces/w-17ebc32046f4a312/preferences` | 1 / 200 / ended |
| archived modules | `GET http://127.0.0.1:51952/api/v0/workspaces/w-17ebc32046f4a312/projects?archived=false` | 1 / 200 / ended |
| archived modules | `GET http://127.0.0.1:51952/api/v0/workspaces/w-17ebc32046f4a312/states` | 1 / 200 / ended |

父请求调用责任：APP的POST refresh由token-manager，instance由InstanceWrapper/InstanceStore，me和me/profile由AuthenticationWrapper/UserStore，workspaces由useWorkspacesFetch；WORKSPACE的preferences/members/projects?archived=false/states由WorkspaceAuthWrapper与相应hooks。仅三个project详情由ProjectAuthWrapper/useProjectFetch增加project及project-member preferences/labels/members/states；workspace-view/归档没有该额外父链。精确多重集合同时拒绝issues/issues-detail/v2/issues任意query、user-properties、workspace单view及所有不应触发写动作；500/无响应/额外query/无归属异常不归入领域自身账。

## 7. 已知限制与真实stderr责任

T5原全web日志18段stderr的来源经定向核未发现本轮新增，T6保存的18段测试标题来源集合/类别与T5一致，但不宣称含运行标识的文本逐字相同。更早full-suite只有errors-only摘要，无法逐段证明这18段的更早数量/文本完全相同。历史原件保留，focused无stderr不推导完整suite无警告。

T8全suite现为**23段**：旧18段按四个明确重命名归一仍全部存在，missing0；新增archive cycle3/module2均是合法fixture覆盖暴露的**真实既有生产缺陷**，不是无害噪声。独评另核真实时序与分支，而非只凭生产文件未改：RootStore构造后RouterStore.setQuery(action.bound)与StoreWrapper同序；有效cache刷新失败必须真实reject走catch，永久pending反会遮住目录，未手写loader/伪造成功；unmount→dispose→restoreMocks保持。

| T8 web原package日志行 | 缺陷与影响 | 明确接收方 |
|---|---|---|
| 214、280 | IssueRootStore autorun在root.store.ts:104–107非action写workspaceSlug/projectId，违反MobX更新约束；不是fixture直接写受保护槽 | M4 root/列表协调P10b |
| 276、284 | T8时CycleStore:386–387与ModulesStore:298–299（最后源catch写入分别为cycle.store.ts:384、module.store.ts:297）异步catch直接loader=false，无action边界；有效缓存+刷新失败的合法分支 | M6归档错误/刷新路径 |
| 220 | CycleQuickActions:139–141真实IconButton嵌在CustomMenu Menu.Button，非法交互元素嵌套，存在hydration/交互风险 | M6有效归档组件验收 |

上述行号指T8原日志；最后F6仍23段，归一测试标题及去ANSI/源码行号/MobX实例号/空行/交错成功stdout后，warning/stack/HTML多重集合增删均0（final-fix-checks/F6-stderr-comparison-r1.json，初版comparison由r1更正）。A-02令两store全文hash变化，不能沿用T8“7个生产源全文未改”当最后证据；已具名核实际catch分支body与新行号，缺陷责任保持。日志为X/task-8-F6-package-logs/web__apps__web__.turbo__turbo-test.log，归一来源表见task-8-suite-metrics.json，相关未改源hash见task-8-noise-source-hashes.json。T8 G明确按A批准的列表拆除边界不阻断，但M6/M7整页⚠保持，不能写pristine或页面零失败。P11仍接其余按需拒绝与历史detached资源；P14撤销再重做/isMigrationUpdate原性质未因局部修复关闭。新增公开错误码0；R14仅删死依赖，不把未新增写成全部依赖未变。

## 8. 本地门禁、准确CI与源桥

| 检查点 | 本地事实 | 准确分支CI（只对应本行head） |
|---|---|---|
| T3 `11cd21369c72ce64a8e5f7e3e2dea8cfabd986b2` | F1–F7通过；F6 18任务/0cache，145 Vitest文件1192测+Node1文件5测=146文件1197测；F7最终163=161Chromium+2Firefox，skip/flaky0。首次Firefox缺artifact与最终F7分别保存；F1-final后未改产品 | [run40662388783](https://github.com/open-nerve/NerveProject/actions/runs/40662388783)，attempt1，ref codex/m4-p10a，server122728355602/web122728356197/e2e122732366270全completed/success；`task-3-ci-success-receipt.json` |
| T6 `e5031c0704e09998422ec839dfffe47efb482369` | F1–F6通过；F6 18任务/0cache，7 Vitest包155文件1268测+api-client Node5=1273，skip0。首次F7 162/1失败，经R17一行夹具修正后F7 163=161Chromium+2Firefox、skip0/retry0；r1夹具定向11测及types/lint/format通过 | [run40685752811](https://github.com/open-nerve/NerveProject/actions/runs/40685752811)，attempt1，ref codex/m4-p10a，server122807539855/web122807540111/e2e122812528945全completed/success；`task-6-ci-success-receipt.json` |
| T8 `8081112492a2fc744a43b3e814c6d36ff4ba3959` | round0七项通过、r1按下表桥接；原F6 1277与F7 169不是r1全字节重跑 | [run40703659542](https://github.com/open-nerve/NerveProject/actions/runs/40703659542)，attempt1/ref codex/m4-p10a；server122869877132/web122869876152/e2e122875932420全completed/success；task-8-ci-success-receipt.json。仅T8产品检查点，非修复或最后文稿CI |
| 未来main集成 | 本期不执行 | 尚未发生，未来另授权后留真实merge/mainCI/归档核hash与清理回执 |

T6原F1–F6/首次F7到r0 final唯一差异是 `e2e/fixtures/links.ts` 的数值排序夹具，已由独立SQL探针、受影响e2e types/lint/format及完整F7覆盖；r0 final到r1只有 `web/apps/web/core/components/issues/issue-layouts/quick-add/root.pending.test.tsx` 的workspace事实修正，已由11项既有行为和web types/lint/format覆盖。故原F1–F6到已提交r1是精确两个**非生产**路径差异；最终F7到r1仅测试夹具差异。`task-6-source-bridge.json` 与 `task-6-r1-source-bridge.json` 保留逐路径事实，不能称七门禁在r1每个最终字节上重跑。

T7 r1的44个现有测试在G7-1/G7-2主体修复后运行，测试后仅 `web/apps/web/core/store/issue/archived/issue.store.ts`、`web/apps/web/core/store/issue/workspace-draft/issue.store.ts` 的两无业务读者archive占位删除及 `web/apps/web/package.json` cap221→219三路径不同；最终types/lint/knip/dead/keywords/diffcheck与45路径源同源覆盖尾部差异。T7没有直接peek callback专用行为套件，不虚报测试覆盖最终每一字节或已跑完整F。guard引擎/配置/12命中文件共14路径r0/r1字节相同，原三类真实guard红→恢复绿继续适用；r1未重做全源变异。


### T8 本地F与R19最终字节的桥

| 原件/源 | 实际结果 | 到已审提交r1的差异与补证 |
|---|---|---|
| F1 make gen-check / F2 make lint-go / F3 GOFLAGS=-p=1 make test | 全exit0；Go51包通过、16无测试、0cache；case级skip未披露 | 仅keywords测试精确登记和R19新故事finally两路径；前者无Go/生成影响，后者由最终e2e static/正常6入口覆盖 |
| F4首次 → F4-final make lint-web | 首次exit2 analytics单测试字面量未登记；R18后exit0，54任务/33cache、73rules/52exceptions、i18n1268、web219=cap | 至r1仅新故事finally；原失败保留，不拼接隐藏URL/扩大规则 |
| F5 make knip | exit0，typegen及treat-config-hints-as-errors | 至r1仅finally；最终e2e types/lint/format0warning补该差异 |
| F6 TURBO_FORCE=true make test-web | round0唯一一次：18任务/0cache，157 Vitest文件1272测+Node5=1277，fail/skip0 | 至r1仅finally；没有将1277写成r1重跑；23段stderr见§7 |
| F7 make e2e | round0 169=167Chromium+2Firefox，skip/retry/flaky0；构建11任务11cache | 至r1仅finally；原S2四身份10例与新增六例实际命中；不将169写成r1重跑 |
| r1正常focused-six与e2e-static | 最终同源6/6、types/lint/format全0，0warning；75API及正常DOM无diagnosticError | 精确覆盖R19差异；NO_COLOR/FORCE_COLOR环境提示仍披露，不称命令输出pristine |

原round0 source=`7920f198d3c8264568553d566f4241406411981ab2550cd720c1820626cedaff`；r1 source=`f007e0ec2ceba7fbf81dd94fd1e4c4d329bd01524027482759404497cd848870`，manifest SHA=`bc57130e521a72bcfc400323c78719583d3103101ffb986408c04b7f785585a0`。七个提交blob与r1 manifest由root核一致。原633项artifact未改，原F/首失败、r1新95项artifact及15项checks分别保留；原件索引为X的task-8-check-index.json/task-8-r1-check-index.json，桥为task-8-source-bridge.json/task-8-r1-source-bridge.json，提交及root核证见task-8-commit-receipt.json/task-8-root-commit-check.json。

R19探针旧/新实际用例均关闭page后故意expect失败，均exit1；旧版首错变page.title关闭且raw丢失，新版保原expect/raw/attachment并记录diagnosticError。wrapper精确性质核验exit0只证明这一错误保真性质，不当正常测试绿；正常6/6另跑并绑定最终字节。首次新探针wrapper受ANSI干扰解析失败原件保留；去ANSI后新名验证，源码finally均逐hash恢复。未专门注入永远pending的title或I/O失败，deadline/拒绝处理/I/O传播属静态判断，race不主动取消浏览器调用。

T7/T8历史八类40hits/12files/28keys，T8总体73rules/52exceptions。A-01新真实测试字面量新增count1/untilM4/P12后，最后73rules/53exceptions、old-issue-urls41hits/29keys，未放宽任何rule；既有40（生产37+既有测试3）与新增测试1分账，不改历史原件。T1广义140条保留106/删34是另一份逐方法账，不是自动命中或网络数。最后文稿提交/source桥与准确HEAD分支CI待本轮文档独评/提交后由root填写，准确产品CI已核只完成§4产品子项，文稿复审已通过，交付子项继续待办。

### 整审A-01/A-02修复与独立复审

代码整审发现两项Minor/P2/Local Fix，均已闭合；C文稿r1另发现一项Minor/P2/Local Fix，已修并经C-r2复审ADDRESSED。无Structural/Needs Decision。A-01为跨项目子项在本期删除相关UI读者后仍预取cycle/module目录：每外部项目两次无消费者旧HTTP及loader/cache写入；两store自己catch，不是unhandled。只删两预取和仅供其用的workspaceSlug参数，保留state/member/label目录、canonical/count/映射/分布及原detached责任。真实fetchSubIssues/createSubIssues两入口×本项目/跨项目四场景，先红两条跨项目额外GET、两本项目先绿，修后3文件23测绿；严格GET全集及POST URL/body、真实目录cache/canonical/父count均有断言。A席r1 scoped Approved，仅关闭A-01；最终F4又覆盖最终测试字节类型检查。新测试唯一subissues URL精确count1/until M4/P12，与真实IssueService同到期，不延到P14/P15。

A-02最初仅被A报告写为“能力证明限制”，root具名追问base生产读者后更正为本期清理缺口：两个已删cycle/module issue store是分布乐观增减链最后生产入口，测试不是生产消费者，不得自动延期M6。最终六路径删除distribution-update全文件/barrel、cycle/module两wrapper接口/实现/import、progress六个旧optimistic测试及专属fixture、constants/state仅该helper消费的COMPLETED_STATE_GROUPS/STATE_DISTRIBUTION两块；真实领域CRUD/目录/analytics/progress与calculateCycleProgress七测试及其他state常量保留。纯死链删除不造镜像红；utils1文件7测、web6文件29测及types/static/knip全绿。A r2全文补读cycle617/module552/constants84/utils index35四新主路径，连同helper删除前222、progress最终54共1564行，Approved且A-01字节未变，无剩余A发现。

原A/B全文报告、A boundary note、r1/r2报告与覆盖账均保留。合计九个已审产品/测试/guard路径由root精确提交5f7085bce4aa5103a0133fa982847d274efa6e47（父8081112492a2fc744a43b3e814c6d36ff4ba3959，tree d0b4ea997e3dd7cdd61e9cbc662b75b064636364），提交receipt核九blob；未夹带14文稿。A r1/r2独评与C已审guard增量分别负责各自范围，C已完成r1全部14文稿增量；唯一C-r1-01是原40误称生产40，本轮改为生产37+既有测试3+新测试1=41，未改规则/产品，C-r2 Approved。

### 整审修复后最后产品本地F与CI

| 项 | 准确运行结果 | pre_source至最后产品5f7085bc的桥 |
|---|---|---|
| F1 make gen-check | exit0，生成一致性 | 在A-01冻结源执行；后仅A-02六web删除路径不同，无docs/Go/api变化 |
| F2 make lint-go | exit0，两模块0issues | 同上六路径，不冒称最终逐字重跑 |
| F3 GOFLAGS=-p=1 make test | exit0，51测试包+16无测试包，count=1/0cache；case级skip未知 | F3正常结束后才执行A-02；六路径桥另存，不无故重跑Go |
| F4 make lint-web | exit0，54任务/12cache，73rules/53exceptions，i18n1268，web219=cap | 与最终产品逐字同源 |
| F5 make knip | exit0，含typegen/config-hints-as-errors | 与最终产品逐字同源 |
| F6 TURBO_FORCE=true make test-web | exit0，18任务/0cache；158 Vitest文件1270测+Node5=1275；报告均passed、Node skip/todo0；23段同来源stderr | 与最终产品逐字同源；web138文件1113测、utils7文件62测 |
| F7 make e2e | exit0，169=167Chromium+2Firefox，skip/retry/flaky0、169attempts、report errors空 | 与最终产品逐字同源；新artifact另存，不覆盖T8 |
| 产品5f7085bc分支CI | [run40712479050](https://github.com/open-nerve/NerveProject/actions/runs/40712479050)，head `5f7085bce4aa5103a0133fa982847d274efa6e47`，attempt1/ref codex/m4-p10a；server122901158718/web122901159223/e2e122907311901均completed/success，root已系统git push成功并核receipt `final-product-ci-success-receipt.json` | 仅此产品检查点，不能替代纯文稿交付HEAD CI |
| 最终交付文档HEAD分支CI | 本轮文稿独评已通过，提交尚未发生；待root提交后核 | 不伪造自身未来SHA，不循环为自指提交更新正文 |

顺序为F1→F2→F3正常结束→授权A-02删除/定向冻结→F4→F5→F6→F7，数据库门禁未并跑。X/final-fix-checks/command-index-final.json、final-source.json、F1-F3-to-r2-final-bridge.json及r2-freeze-to-final-bridge.json分别保留精确命令/cwd/raw/exit/pre-source/hash和桥。F6/F7原件及逐包/HTML统计由本轮新metrics提供，A r2独核源/原件，root主核全套；不是拿T8旧1277/169外推新总数。

最后产品已核到九个提交blob，准确产品CI已三job成功；本轮最终交付还包含文档提交，准确最终HEAD、ref、run/attempt/server-web-e2e身份及raw hash由root提交后保存最终交付回执。已核检查点与该提交后回执分开，无需为把自身新SHA写进自身循环提交。spec§4原第8项现拆产品已发生事实与文稿/交付余项；最终交付未齐仍未勾；未来mainCI/归档清理另授权后接续。

## 9. 最后产品消费者、大文件与dead增量

T9在5f7085bc最后产品源按tracked web逐路径重核：旧TIssue/TBaseIssue/TIssueMap、四删除kind均0；直接generated114，较808时点116仅distribution-update整文件删除与progress.test去专属Issue import退出，新增0。基线132核心逐项为106直接生成/13删除/13局部或推断保留；50四kind为6直接生成/24删除/20合法领域或共享保留；交集9并集173为106生成/34删除/33保留。当前114=106基线+8补增，T6两118集合差及808的116保留为历史，不作计数抵销。每路径见[spec附录B](../specs/P10a-web-issue-types.md#附录-b基线核心类型消费者)，最后机器账task-9-final-product-inventory.json。

附录C为main→5f7085bc全部13个既有大生产TS/TSX（含删除/降限）与keywords配置；本轮新触及cycle635→617、module570→552，keywords2670→2960，form407→416等逐项实核，非借旧检查点。A-02仅删退休列表乐观更新，真实calculateCycleProgress七测保留。

最终原脚本全仓prop371/member978/export11，较前371/988/11净删10、真正新增身份0：两wrapper接口/实现4项+helper path/value字面成员6项。另7项DTO read-count/str属性变化单列，不能当身份增删。最终main→全部改动文件集合为236候选/61文件（163member/73prop），不是旧227减10：旧已触及helper退出6项，新纳入cycle12/module3个既存候选，227−6+15=236；另4wrapper原不在旧diff集合，不能再扣。原225/227是历史检查点，最终ledger为final-fix-checks/final-touched-dead-ledger.json，原TSV及更正delta-r1保留。

新增纳入15项如下。它们在修复前全局TSV已存在；本轮仅闭合A-02的退休列表独占链，不将其他领域接口/属性/历史候选一并清空，也不把class候选0的旧时点结论覆盖最后源。接口槽须结合具体类使用，r0/w0本身不是“全仓无读者”证明；这些保留项由M6领域完整接通/清理时逐项复核，不因本表承诺当前有业务入口。

| 文件:行 / 名称 | 声明owner / r-w | 处置与责任 |
|---|---|---|
| `web/apps/web/core/store/cycle.store.ts:24` `progressLoader` | ICycleStore; w 0 / r0-w0 | 既存领域候选，本次新增纳入改动集合；保留并交M6按真实读者复核，不称已清零 |
| `web/apps/web/core/store/cycle.store.ts:28` `activeCycleIdMap` | ICycleStore; w 0 / r0-w0 | 既存领域候选，本次新增纳入改动集合；保留并交M6按真实读者复核，不称已清零 |
| `web/apps/web/core/store/cycle.store.ts:32` `currentProjectCompletedCycleIds` | ICycleStore; w 0 / r0-w0 | 既存领域候选，本次新增纳入改动集合；保留并交M6按真实读者复核，不称已清零 |
| `web/apps/web/core/store/cycle.store.ts:33` `currentProjectIncompleteCycleIds` | ICycleStore; w 0 / r0-w0 | 既存领域候选，本次新增纳入改动集合；保留并交M6按真实读者复核，不称已清零 |
| `web/apps/web/core/store/cycle.store.ts:36` `currentProjectActiveCycle` | ICycleStore; w 0 / r0-w0 | 既存领域候选，本次新增纳入改动集合；保留并交M6按真实读者复核，不称已清零 |
| `web/apps/web/core/store/cycle.store.ts:44` `getProjectCycleDetails` | ICycleStore; w 0 / r0-w0 | 既存领域候选，本次新增纳入改动集合；保留并交M6按真实读者复核，不称已清零 |
| `web/apps/web/core/store/cycle.store.ts:50` `fetchActiveCycle` | ICycleStore; w 0 / r0-w0 | 既存领域候选，本次新增纳入改动集合；保留并交M6按真实读者复核，不称已清零 |
| `web/apps/web/core/store/cycle.store.ts:80` `progressLoader` | class CycleStore; w 2 / r0-w2 | 既存领域候选，本次新增纳入改动集合；保留并交M6按真实读者复核，不称已清零 |
| `web/apps/web/core/store/cycle.store.ts:88` `issueService` | class CycleStore; w 1 / r0-w1 | 既存领域候选，本次新增纳入改动集合；保留并交M6按真实读者复核，不称已清零 |
| `web/apps/web/core/store/cycle.store.ts:149` `currentProjectCompletedCycleIds` | class CycleStore; w 0 / r0-w0 | 既存领域候选，本次新增纳入改动集合；保留并交M6按真实读者复核，不称已清零 |
| `web/apps/web/core/store/cycle.store.ts:168` `currentProjectIncompleteCycleIds` | class CycleStore; w 0 / r0-w0 | 既存领域候选，本次新增纳入改动集合；保留并交M6按真实读者复核，不称已清零 |
| `web/apps/web/core/store/cycle.store.ts:211` `currentProjectActiveCycle` | class CycleStore; w 0 / r0-w0 | 既存领域候选，本次新增纳入改动集合；保留并交M6按真实读者复核，不称已清零 |
| `web/apps/web/core/store/module.store.ts:29` `getModulesFetchStatusByProjectId` | IModuleStore; w 0 / r0-w0 | 既存领域候选，本次新增纳入改动集合；保留并交M6按真实读者复核，不称已清零 |
| `web/apps/web/core/store/module.store.ts:34` `getProjectModuleDetails` | IModuleStore; w 0 / r0-w0 | 既存领域候选，本次新增纳入改动集合；保留并交M6按真实读者复核，不称已清零 |
| `web/apps/web/core/store/module.store.ts:149` `getModulesFetchStatusByProjectId` | class ModulesStore; w 0 / r0-w0 | 既存领域候选，本次新增纳入改动集合；保留并交M6按真实读者复核，不称已清零 |

旧T9的task-9-final-static-inventory.json在本轮被作者误覆盖，未找到原bytes或原全文件SHA备份；旧名现明确标reconstructed-from-808，不能证明逐字恢复。808语义可由准确tree/T1/T6账及C独立复算补证，最后源另以task-9-final-product-inventory.json重核；旧audit/文稿manifest/draft及产品/测试/CI raw未改。此静态库存历史保真限制保留，不能泛称全部原件逐字保留。


## 10. 实施与终审完成、最终交付待项

C-r2真实回执为X/final-docs-r2-rereview.md，SHA256 `003d2148fce2ed649ba26f107fd3465a8d46dffffdea3bfd403f2b4393effc78`：C-r1-01 ADDRESSED、Approved、无新增。三项Local Fix均闭合；最终文稿提交/准确交付HEAD CI及回执仍按提交后事实核验，不预填未来SHA。

A/B原全文、A-01/A-02局部修复复审及C round0/guard delta均已收齐，代码无剩余finding；355唯一路径分配及审查源/限制见[整体审核](../../../reviews/re-11-M4P10a整体审核.md)。C-r1已读14文稿增量，唯一C-r1-01已修且C-r2 Approved无新增；准确5f708产品CI已核成功，最后文稿交付HEAD CI仍待root提交后核；现阶段状态为“实施与终审完成、集成待办；最终文稿提交后准确HEAD CI以交付回执核验”。

本轮文稿在已审产品上只改原14docs；最后doc-only hash桥/manifest由本轮新静态收据记录。root在真正提交后核准确HEAD/ref/run/attempt/jobs并保存最终交付收据，不为正文自指SHA循环提交。文稿独评已完成，提交后准确最终交付HEAD CI尚未发生，spec§4最终交付子项仍未勾，不将旧808成功当最后交付。

## 11. 输入处理与未来集成边界

五份输入已追加本期子项：[M1-P2](../handoffs/M1-P2-trim-content.md)、[M1-P3](../handoffs/M1-P3-trim-platform.md)、[M1-closeout](../handoffs/M1-closeout.md)、[M2-closeout](../handoffs/M2-closeout.md)、[M3-closeout](../handoffs/M3-closeout.md)，顶层status均open。M4§14/总体§9.4与未来M5–M8最终closeout不关闭/不预建；P10b只能等P10a真实main集成及准确mainCI后重核。未来集成另获授权后才--no-ff合并、核准确mainCI、归档所有根外准备/轮次/CI/E2E原件逐hash，再清理并复核；当前均未发生。

## 12. 准备期历史记录（保留当时时态）

以下整段保留准备期原文及预检发现，所有“当前”均仅指当时；不覆盖上文现在态。
### 准备期原记录（历史全文）

日期：2026-10-28。状态：负责人已裁定P10A-S1采用A，验收归属同步后已通过独立定向静态预检、无开放发现。负责人随后已授权按通过预检的 spec/plan 实施 P10a，当前推进 T1；本阶段验收仍开放。准备分支 `codex/m4-p10a`，worktree `.codex/worktress/codex/m4-p10a`；基线 main `4599a9cfc5be90b2c242977f194157778a5f6ec9`。配套：[spec](../specs/P10a-web-issue-types.md)、[plan](../plans/P10a-web-issue-types.md)、[deep-review](../../../reviews/re-10-M4P10预检审核.md)。

### 历史1. 评审方式

按02允许的紧凑方法细化已批准设计；作者准备后，独立审者按deep-review的逻辑、集成、覆盖三镜头阅读全文并追踪代码。静态盘点和依赖推导不冒充产品原型、逐Task重放、行为测试或变异。原件在主项目 `.superpowers/sdd/P10-preparation-20261028/`，准备工作树之外，后续清理不得遗漏。

### 历史2. 验收标准逐条核对

P9已完成授权集成：准确main CI40642368845三个任务全绿；3,317文件/136,629,586字节/1空目录归档核hash，P9分支/worktree正常清理，清理后再次核验通过。完整证据见[P9 review §14](P9-purge-review.md#14-授权合并main-ci-与归档清理2026-10-28)。这是基线证明，不是本阶段产品验证。

准备期本阶段9任务均未实施；现已获实施授权，从 T1 逐任务推进。spec§4 阶段产品验收仍全未勾。准备期只核路径、接口、输入映射、任务闭合、链接和守卫；最终实际命令/文件指纹由主控预检收据记录，未运行的检查不能列通过。

### 历史3. 准备期裁定

沿用P10a先收窄后换型、P10b扩既有Reconciled及纯分组模型的设计，不重开架构。具体局部结论/理由/代价见spec§3。准备提交不改phase；获实施授权后T1用合法M4/P10，a/b共用阶段号。归档localStorage键按原任务分配在P10b T9修正，P13只核页面。

### 历史4. 预检发现与修订

独立审者 `p10a_independent_preflight` 首轮及定向复审累计全文 86 文件、15,368 行，节选与枚举另列；132 核心类型消费者及 50 枚举消费者不冒称均全文审阅。原报告 `p10a-preflight.md` SHA `bedb5783fe466ebe19d38aef982ca5b333fbdceb0846f86202d4ea45cc686095`，覆盖清单 `p10a-coverage.json` SHA `f72ac51fd6ad230e73a96576279531f241a031b8b8c8170ab08ebf70cadbde98`，均在上述 root 证据目录。

- **T3-L1：文档层关闭。** GlobalView 的 workspaceIssues 同步、detail 五个领域动作，以及 inbox/power-k 间接调用原先不在 T3 完整闭合清单。已将相关 T5 子集前移 T3，分别检查八成员与五方法引用，合法同名领域方法单独解释；原审者核正式 spec/plan 修订后确认关闭，仍需实施时真实编译。
- **P10A-S1：Major / 已获负责人裁定A。** 首轮发现保留领域自身请求与整页零失败冲突，完整链及原方案见综合报告。负责人随后明确保持列表删除范围，把M6/M7自身页面完整可用/零失败归回各自里程碑；本次已同步设计/spec/plan/T8与交接责任。精确请求记录、领域加载保留和不可冒称整页零失败的边界见spec§2.4；定向复审结果单独记录，不倒改冻结首轮报告。

首稿与 T3-L1 复审稿均冻结保存。该次复审 spec SHA `6235f67114b871ca8b9e8997d7036501e09efb8f1fa80536fe803877fb0f3865`、plan SHA `340ce6450be27abbb73ab41c01bd4b1a417df8823c97d474da671b75689c16a9`；后续主控仅补待决状态、报告链接与中文标题，最终字节由提交前收据绑定。不用首稿哈希冒充后续稿。

### 历史5. 方法边界

准备期仅文档准备，未运行完整本地CI、生成器、数据库套件、E2E、产品测试或变异；没有新增产品文件、依赖或兼容层。小范围语言能力探针如实际运行，单列命令/输出，不当产品测试。

### 历史6. 移交事项

P10a实施并经独立整审和负责人授权集成后，P10b才能从准确main重核计划和消费者；当前P10b是依赖预检。P11接详情数据层，P12–P15接页面，P16与M4收尾接最终守卫/浏览器/性能/交接。每段实施完成仍需整分支deep-review和局部修复闭环；准备期不开始 T1 的限制已由后续实施授权取代；当前推进 T1，仍禁止 push/merge。

P10A-S1已由负责人选A；M6接迭代、模块及两归档目录自身页面，M7接项目/工作区视图自身页面的完整可用与零失败验收。T8逐请求记录方法、完整URL/query、触发、实际状态、源码归属、证据和期限，T9把记录与两项可见代价交接；接收方接通时替换旧接口、删除过渡记录并核真实有效实体。最终M4-closeout仍在收尾创建。当时决定只授权文档同步与预检；后续负责人已授权实施 P10a，当前从 T1 推进。P10b 仍等 P10a 真实集成基线。

### 历史7. 已知限制

静态预检不能证明迁移后每Task确实编译、全部真实页面请求或分页并发性质。实施期须逐项保留红绿/拒绝/会话变化/恢复证据；P10b执行前须P10a集成main CI通过并完成差异复核。M4§14、总体设计§9.4、输入handoff总status和M5–M8最终closeout保持开放。

### 历史8. 负责人A裁定后的定向复审

2026-10-28，负责人回答“A：保持范围，调整验收归属（推荐）”。前一文档提交为 `4dc3173bd0651d06d0aa7780cff9b581ad8c980d`；本次只补裁定落点，未开始产品实现。原件独立保存在主项目 `.superpowers/sdd/P10-preparation-20261028/decision-a/`，不覆盖原 `final-preparation-receipt.json` 和冻结报告。

已同步设计§3.1/7.2/12P10a/13.2/15、spec§2.4/3/4/5/7、plan T2/T8/T9、P10b状态与04接续。T2明确迁出cycle详情与workspace views领域装载，T8最低核8个自身GET URL，补完整query和共享URL的调用来源边界；旧工作项列表/筛选读者不能借领域URL留存。S2/watchPage/关键词规则未变。独立定向静态预检通过、无开放发现，报告与字节指纹见综合报告§5及 `decision-a/review.md`；文档静态检查另记最终收据，不冒称产品验证通过。

### 历史9. T1 实施接续

2026-10-28，负责人授权按通过预检的 spec/plan 实施 P10a。T1 在源 `4569a9e7386031cffb31aa37d6cf06a726b08c2c` 冻结 132 类型/50 枚举消费者并逐文件分配 T2–T6（spec 附录 B），记录 23 文件/140 处广义旧 URL 及 9 个既有大文件的前值（附录 C 与本地台账），phase 仅设 M4/P10。原准备/失败/预检证据保持不变。实际命令原件、退出码、生成文件指纹及独评回执见工作树 `.superpowers/sdd/P10a-web-issue-types/task-1-*`；完成/提交状态由主控准确源回执接续，不据本文提前宣称 T1 独评、提交或全阶段验收通过。
