# M4/P10a 工作项换型与列表收窄实施计划

> **执行者：** 获负责人后续实施授权后，使用 `superpowers:subagent-driven-development` 或 `superpowers:executing-plans` 逐 Task 执行。本轮仅准备及静态预检，不运行下列产品步骤。

**负责人裁定：** P10A-S1 已选 A，见[报告](../../../reviews/re-10-M4P10预检审核.md)。只删工作项列表，保留领域自身加载；T8 按 spec§2.4 验收拆除边界，M6/M7 自身页面完整可用与零失败归所属里程碑。文档同步已通过独立定向静态预检，无开放发现；本轮仍不执行 T1。

**目标：** 删除四类尚属 M6/M7 的工作项列表，先收窄读取，再将核心工作项实体换为生成的 Issue/IssueDetail，保持每任务编译。

**结构：** 直接消费 OpenAPI 类型，不建立 Nerve→Plane 转换层。四类列表连同独占状态/布局分支删除；其他页头/详情保留。共享工作项 map 只承载生成的实体，UI 辅助状态留在自身表单；旧 service 按设计已批准的 P10b/P11/M7 期限过渡。

**技术栈：** 当前仓库锁定的 TypeScript、React、MobX、React Router、Vitest、Playwright、pnpm；不新增依赖、不全局安装。

**Spec：** [P10a spec](../specs/P10a-web-issue-types.md)，以其责任边界为准；[M4设计](../M4-design.md) §7.2、12/P10a。执行基线 main `4599a9cfc5be90b2c242977f194157778a5f6ec9`，P9 的准确 CI/归档见其集成回执。

## 全局约束

- 当前只做文档准备，全部产品步骤未执行；准备提交不改 `tools/keywords.json` phase。T1 获授权后才设 **M4/P10**，P10b同值；现有守卫只支持数字，禁止写无效M4/P10a或为此放宽解析。
- 主目录HEAD始终main；所有实施在 `.codex/worktress/codex/m4-p10a`。plane/和refer/只读且不在其中跑git；不操作现有容器/Docker；不stash/clean/reset--hard/强推。
- 类型直接来自 `@nerve/api-client`。没有全字段兼容实体、`as unknown as Issue`、虚假null/default、API响应翻译层；生成物只由make gen-web生成。
- P10a先收窄后换型；每Task后 `pnpm --filter web check:types` 通过；共享包同时由完整 `make lint-web` 核对。T6不能用关闭检查或把文件排除来救编译。
- 精确只删除EIssuesStoreType四值CYCLE/MODULE/PROJECT_VIEW/GLOBAL；保留PROJECT/PROFILE/ARCHIVED/DEFAULT/WORKSPACE_DRAFT及EIssueLayoutTypes。
- P10a只用Issue/IssueDetail核心类型；列表容器/排序枚举正式换型P10b，详情其他DTO/P11；本段不把已有API再造一套。M5/M6/M7字段随所属真实数据恢复。
- 描述只能从IssueDetail读；Issue后到不能抹掉已取描述；版本决胜/确认队列/列表协调留P10b。nullable作者/父项不做非空断言。
- old-issue-urls暂留命中逐文件、样例、count、原因、until；M4列表到M4/P11、P11详情到M4/P12、P15剩余搜索到M4/P16、M5/M6/M7到各自M；不整目录豁免、不藏URL拼接。
- 新生产逻辑文件约400行内；生成物例外；本期机械改动的既有大文件在spec§2.5/附录C登记改前/改后行数、理由，不扩新大段逻辑。预计9任务，计划控制在约1500行内并留修复余量。
- 现有超400行base-issues.store等在本期只做删除/机械换型，P10b承担算法重写；不要在P10a附带重做P11–P15完整页面行为。
- 新公开错误码预期0；若实际新增，契约、`C/lib/error-messages.ts` PROBLEM_MESSAGES 与 en/zh-CN errors.json 同Task，强制test-web，不以旧error?.error写法扩散。
- 测试复用既有deferred、store/root/session fixture、watchPage、API创建工作区/项目/成员；不造假的M6/M7成功API或加空service掩盖。新测试按行为，不为机械每行修改造镜像测试。
- 每任务要求/质量独评，局部问题根因修复后复审。架构回归、跨模块协议改变或意外高风险，提交已完成工作后上报。
- 提交详细中文多行，前缀feat/fix/docs(M4/P10a)；只stage精确路径；push优先`/usr/bin/git`。阶段终审不自动授权合并；本次明确不实现。
- 本地七项、最终分支CI、未来mainCI是独立事实，各绑准确SHA。变异红必须行为断言，不把无测试匹配/编译失败/基础设施失败算性质红。

## 评审重点

1. 删循环/模块列表却误删已归档“迭代目录”或项目日历共用选择器：T2/T3静态引用闭合、T8真实路由及组件边界验证，领域目录保留。
2. Issue列表行后到把已取描述抹掉，只有列表数据却当成已取详情：T6 addIssue两种来源顺序、显式detail accessor分别验证。
3. 空created_by_id/updated_by_id/parent_id及非空project_id/state_id在详情/浮层/收集箱触发旧假设：T6编译矩阵与实体存储回归，T8现有页面冒烟。
4. 克隆sourceIssueId、tempId/state__group被塞进生成实体或HTTP载荷：T5独立表单输入和实际提交载荷测试；没有写者的分支删除，状态组目录未到不能伪造。
5. 新旧service过渡期限混淆，新URL守卫只放行文件却扩大命中：T7规则样例/count/until与全清单独评，P10b/P11接收逐条明示。

## 文件结构和依赖

`C/`、`A/`、`P/`展开方式见spec§2.1；核心132文件逐路径清单在spec附录B，枚举四值消费者在T1清单，任何新出现消费者同步补清单。T2/T3不是删除整个cycles/modules/views模块，只删除issue列表与独占读者。

T1→T2→T3→T4→T5→T6→T7→T8→T9；任何任务结束都先编译，不能保存红状态交给下Task补。静态独评可并行，构建/数据库套件必须串行。

**G，每Task通用收尾：** 保存起止SHA、dirty source hash、命令/cwd/raw/退出码；新增行为先可解释红再绿；本段机械删除由类型检查、全局引用/knip和真实受影响行为验证，不强造无意义测试；`git diff --check`→独立要求/质量评审→修复复审→精确stage→详细中文提交。未运行步骤不勾。

**F，完整本地检查点：** T3、T6、T8/最终产品修复后及每次有产品变化的push之前，以下严格串行；已过且没有新变化/失败/未解风险不重复。

```sh
make gen-check
make lint-go
GOFLAGS=-p=1 make test
make lint-web
make knip
TURBO_FORCE=true make test-web
make e2e
```

通过要求：七项exit0；web记录真实运行而非turbo缓存；E2E以当次配置/环境要求浏览器为准，准确计数skip/失败；任何失败保留原件。F当前不执行，P9的基线证据不能当P10a F。

## Task 1：阶段、类型合同与消费者冻结

**Files：** 改 `tools/keywords.json`；核验生成 `api/dist/openapi.yaml`、`P/api-client/src/schema.gen.ts`、`P/api-client/src/index.ts`；记录 `specs/P10a-web-issue-types.md` 附录B/C与本地台账；不新建service或无读者导出。

**Interfaces：** 消费当前generated Issue/IssueDetail；确认IssueDetail只多description_html、Issue的实际null字段/数组/priority。产出可审的类型消费者、四store kind消费者、所有旧URL及大文件尺寸清单。

- [ ] 记录当前git HEAD、基线祖先关系、工作树clean、P9准确mainCI回执；核main/root仍保持main。
- [ ] 用 `rg -l '\b(TIssue|TBaseIssue|TIssueMap)\b' web` 和 `rg -l 'EIssuesStoreType\.(CYCLE|MODULE|PROJECT_VIEW|GLOBAL)\b' web` 冻结清单；逐文件标T2–T6责任。数量变化必须解释，不能仅用设计旧131/53作依据。
- [ ] 把phase改M4/P10，核所有until相对当前phase仍合法；到期只按已批准归属修正，不延长P10b应删除的/user-properties/期限。
- [ ] 运行 `make gen-web`、`make gen-check-web`，保存生成文件sha256/行数与git diff；当前类型已经存在，若出现非预期API变化先查根因，不手改生成物。
- [ ] `pnpm --filter web check:types`、`node tools/keywords.mjs`、`git diff --check`通过；此Task无新运行行为，不新增重复实现的类型测试。执行G，提交 `feat(M4/P10a): 核定工作项生成类型并切换前端阶段`。

## Task 2：删除四类工作项列表挂载与独占组件

**Files：** 删除 `C/components/issues/issue-layouts/roots/cycle-layout-root.tsx`、`module-layout-root.tsx`、`project-view-layout-root.tsx`、`all-issue-layout-root.tsx`。修改下列真实页文件（前缀A）：

```text
(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/[cycleId]/page.tsx
(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/[moduleId]/page.tsx
(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/views/(detail)/[viewId]/page.tsx
(all)/[workspaceSlug]/(projects)/workspace-views/[globalViewId]/page.tsx
(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/cycles/page.tsx
(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/modules/page.tsx
```

核对 `C/components/cycles/archived-cycles/root.tsx`、`modules/archived-modules/root.tsx`：它们的归档领域目录不是工作项列表，不能按名称整删。改 `C/components/cycles/active-cycle/use-cycles-details.ts`、`root.tsx`，cycles/modules的progress-sidebar，cycles/transfer-issues-modal.tsx、transfer-issues.tsx，list/kanban headers/group-by-card.tsx、calendar/base-calendar-root.tsx及其真正依赖。

**Interfaces：** 消费现有四layout root与issue store；产出页面无四root挂载/独占操作，保留自身领域详情/页头。T3将删除已无读者store。

- [ ] 先按T1枚举四root/四kind/transfer/addIssuesToView的实际读者；标“工作项列表独占”或“其他合法领域职责”，检查后者仍有消费者。
- [ ] 删除四root及页面import/JSX；页面不加空数组/空service/featureflag来伪装列表；保留自身页头/详情。删除由此无读者的状态、callbacks和barrel导出。
- [ ] 先迁出 root 内仍需保留的领域加载：cycle `fetchParentStats` 的详情读取移到所属页面/领域hook，不再依赖已删工作项列表成功；`all-issue-layout-root` 的 `fetchAllGlobalViews` 迁到保留的工作区视图宿主，覆盖默认/自定义视图。仍调用所属store/service，不复制列表引擎；真实组件配有效领域store fixture验证加载调用与页头/详情保留，不模拟HTTP成功、不以冷启动空态冒充已保留详情。
- [ ] 共用分组头/日历只删M6/M7分支；ExistingIssues选择器仍给项目日历和P14子项/关系保留；活动迭代/progress删issue-list请求及操作，不能把领域自己的统计代码冒充同一职责整删。
- [ ] 若transfer组件所有读者已随四列表移除，连文件/导出/仅为它存在的文案一起删除；若有合法领域读者，先剥离issue-list职责，在台账列精确剩余读者，不能留下引用将删store的接口。
- [ ] `pnpm --filter web check:types`通过；用 `rg -n '(CycleLayoutRoot|ModuleLayoutRoot|ProjectViewLayoutRoot|AllIssueLayoutRoot)' web`核无旧root；错误退出2与无命中exit1区分。`make knip`查删除闭合；根据实际清理精准降低lint cap，不改限额掩盖警告。
- [ ] 执行G；按负责人A裁定，本Task及T8都不得写“所有M6/M7页面已无失败API”；其完整页面验收归所属里程碑。提交 `feat(M4/P10a): 移除后续模块的工作项列表挂载`。

## Task 3：删除四类store、筛选和枚举分支

**Files：** 删除 `C/store/issue/cycle/{issue.store.ts,filter.store.ts,index.ts}`、`module/{issue.store.ts,filter.store.ts,index.ts}`、`project-views/{issue.store.ts,filter.store.ts,index.ts}`、`workspace/{issue.store.ts,filter.store.ts,index.ts}`。改 `C/store/issue/root.store.ts`、`C/hooks/store/use-issues.ts`、`C/services/cycle.service.ts`、`module.service.ts`、`workspace.service.ts`、`P/types/src/issues/issue.ts`；T1枚举其余50消费者逐个闭合；此外改 `C/store/global-view.store.ts`、`C/store/issue/issue-details/issue.store.ts`，以及 `C/components/issues/{issue-detail,peek-overview}/` 的直接成员调用链，并包括 `C/store/inbox/inbox-issue.store.ts`、`C/components/power-k/ui/pages/context-based/work-item/commands.ts` 的间接方法调用，不以类型/枚举清单充当全部引用清单。

**Interfaces：** EIssuesStoreType保留PROJECT/PROFILE/ARCHIVED/DEFAULT/WORKSPACE_DRAFT；useIssues对应TStoreIssues map/switch不含四值；RootStore不再实例化四组列表/筛选。

- [ ] 先删除仍在合法共用读者里的四值分支并核switch默认不会把以前GLOBAL意外映射PROJECT；删四目录全部12文件和其barrel/constructors/类型引用。
- [ ] 三旧service仅删除返回TIssuesResponse的列表方法及无读者参数/导入，保留自身领域CRUD/详情；四enum值与读分支同次删除。
- [ ] 直接成员闭合：移除GlobalViewStore.updateGlobalView向workspaceIssuesFilter/workspaceIssues同步列表的分支，保留视图自身CRUD；删除IssueDetailStore的addCycleToIssue/addIssueToCycle/removeIssueFromCycle/changeModulesInIssue/removeIssueFromModule及接口声明，同提交删除详情/浮层/选择器、inbox store 和 power-k 对应回调/分支/控件（原T5该子集前移），不留空handler。检索 `rg -n '\b(cycleIssues|cycleIssuesFilter|moduleIssues|moduleIssuesFilter|projectViewIssues|projectViewIssuesFilter|workspaceIssues|workspaceIssuesFilter)\b' web`，逐个解释剩余同名非相关项；再全局检索 `rg -n '\b(addCycleToIssue|addIssueToCycle|removeIssueFromCycle|changeModulesInIssue|removeIssueFromModule)\b' web`，区分合法同名领域方法与已删detail接口调用，逐链闭合；这项覆盖不依赖132类型或50枚举清单。
- [ ] `rg -n 'EIssuesStoreType\.(CYCLE|MODULE|PROJECT_VIEW|GLOBAL)\b' web`无命中；枚举定义人工/类型检查确认无四值；rg目录import/字符串不会漏barrel；WORKSPACE_DRAFT仍能编译。
- [ ] `pnpm --filter web check:types`、`make knip`、`make lint-web`通过；执行F作为删除检查点，保存真实七项原件。
- [ ] 执行G；提交 `feat(M4/P10a): 删除四类旧列表状态与枚举分支`。未经本地完整绿不推产品检查点。

## Task 4：收窄布局和显示字段

**Files：** `C/components/issues/issue-layouts/spreadsheet/columns/{cycle-column,module-column,attachment-column}.tsx`及spreadsheet-table.tsx/issue-row.tsx；properties/all-properties.tsx、list/block.tsx、kanban/block.tsx、calendar/issue-block.tsx、utils.tsx；`P/types/src/issues.ts`、`P/constants/src/issue/`、`P/utils/src/work-item/`中实际消费者。

**Interfaces：** 仍以TIssue编译，但布局不依赖cycle_id/module_ids/attachment_count/estimate_point等新Issue不存在的字段；列表/分组容器与本地排序枚举先保留至P10b，不混入新接口迁移。

- [ ] 按字段rg定位到成员访问/解构/计算key，区分M6自己域对象同名字段；只删工作项实体的相关列、显示属性、分组/筛选选项和独占组件。
- [ ] 移除enum值/映射时同步所有消费者和i18n死键；保留EIssueLayoutTypes四布局kind；不把不支持选项隐藏到默认值或无动作handler。
- [ ] tempId先核写者：无写者删除卡片disabled/loader读取；仍有活pending使用方时给拥有它的局部store独立字段，不在TIssue/未来Issue附加属性。state__group fallback改从状态目录读，未知目录显式无值。
- [ ] 相关现有Vitest回归；`pnpm --filter web check:types`、`make lint-web`、`make knip`通过；静态字段清单复核没有“以动态索引绕过类型”的新增。
- [ ] 执行G；提交 `feat(M4/P10a): 收窄工作项布局读取与显示选项`。

## Task 5：收窄详情、弹窗、收集箱和局部UI状态

**Files：** `C/components/issues/issue-detail/root.tsx`、issue-detail-widgets/issue-detail-widget-modals.tsx、peek-overview/root.tsx、issue-modal/base.tsx、form.tsx、modal.tsx、components/default-properties.tsx；quick-action-dropdowns/{project-issue,issue-detail,all-issue}.tsx仍有者；`C/components/inbox/content/{issue-root,issue-properties}.tsx`、inbox/modals/create-modal下调用方；对应 `C/store/issue/issue-details/`；新（如需）`C/components/issues/issue-modal/types.ts`，只定义局部表单合同。

**Interfaces：** sourceIssueId从实体data移到独立 `sourceIssueId?: string` prop，由调用者传到base/form；payload只含实际实体/请求允许字段。详情描述仍由已有detail读取，在T6替换类型时转显式accessor。

- [ ] 删除is_subscribed/is_intake/description_binary等不属于Issue的读取与M4不可达附件/关注挂载；M5/M7自身域DTO保留。不要把comment/activity/relations的完整DTO重构提前到本段。
- [ ] 克隆调用者先将sourceIssueId作为独立props传给弹窗，base取详情与form的项目锁定消费该prop；提交不再通过删data.sourceIssueId修补实体。新 `C/components/issues/issue-modal/base.clone.test.tsx` 用现有DOM测试写法挂载 `CreateUpdateIssueModalBase`，隔离重型叶组件与store边界，捕获实际提交参数：`sourceIssueId`能触发源详情读取、项目选择保持锁定，body没有该字段。项目锁定实际由 `form.tsx` 实现，须保留真实 FormRoot 或补真实 form 回归；mock 整个 FormRoot 只能证明 prop 传递，不能证明禁用。该组件层测试不冒充真实HTTP。
- [ ] 草稿上下文从现有modal props/store kind传入，普通Issue不恢复is_draft；只保留当前表单字段，不复制完整旧TIssue。代码若需要生成Issue缺失的项目/状态时，使用Partial表单并在提交时走现有校验，不能断言成完整Issue。
- [ ] 运行 `pnpm --filter web exec vitest run core/components/issues/issue-modal/base.clone.test.tsx`，先记录真实载荷性质红，再最小调整并绿；不以不存在测试文件的失败算红。`pnpm --filter web check:types`、`make lint-web`、`make knip`通过。
- [ ] 执行G；提交 `feat(M4/P10a): 分离弹窗局部状态并收窄详情实体读取`。

## Task 6：机械换型与描述/null回归

**Files：** `P/types/src/issues/issue.ts`；spec附录B全部剩余核心消费者；`C/store/issue/issue.store.ts`；`C/hooks/store/use-issues.ts`；§2.3点名四旧service和inbox-issue.service.ts；新 `C/store/issue/issue.store.types.test.ts`。

**Interfaces：** 严格采用spec§2.2五个接口；普通getter回Issue|IssueDetail|undefined、detailgetter只回真实Detail。旧TIssuesResponse元素Issue；三个旧核心类型从定义和barrel消失。

- [ ] 新存储回归 `keeps_detail_when_a_list_row_arrives`：先addIssue完整Detail（非空HTML），后addIssue同id的Issue（不含description_html），detailgetter保留HTML、普通字段按现有合并更新；反序Issue→Detail也能读到描述。
- [ ] 新 `does_not_claim_list_rows_are_details`：只有Issue时detailgetter为undefined，空HTML的真正Detail返回Detail，不以HTML truthy判定。新 `keeps_nullable_actor_and_hidden_parent`：created_by_id/updated_by_id/parent_id均null保存和读取不崩、不恢复旧created_by/updated_by。
- [ ] 先运行 `pnpm --filter web exec vitest run core/store/issue/issue.store.types.test.ts`记录缺接口/不满足断言；新增接口编译红与描述保护的行为红区分。实现显式in判别与现有合并，不加P10b版本决胜。
- [ ] 分目录机械替换导入、created_by/updated_by→*_id、可空性；描述调用者改detailgetter/显式in判别。map直接Record，不用新全局别名复活旧类型；全部注释引用也改为真实当前类型。
- [ ] TIssueResponseResults的TBaseIssue[]换Issue[]，容器其它字段留P10b；旧四service只做已批准签名过渡，M7create/update参数Partial<Issue>，不冒称旧路由有后端。
- [ ] `rg -n '\b(TIssue|TBaseIssue|TIssueMap)\b' web`无命中；`pnpm --filter web check:types`/`make lint-web`/`make knip`通过；新增类型编译不能靠any/双断言/重新扩Issue字段。
- [ ] 对描述测试做有效变异：把既有合并临时改为整行替换，断言HTML丢失而红；恢复精确原字节后再绿。nullable/accessor测试若纯编译保护，按真实验证层记录，不伪称运行变异。
- [ ] 执行F和G；提交 `feat(M4/P10a): 用生成工作项实体替换旧核心类型`。

## Task 7：旧地址守卫、死代码与警告收口

**Files：** `tools/keywords.json`；本期所有被改文件与有死引用的barrels/i18n keys；实际警告下降包的package.json lint上限。`tools/keywords.mjs`只读，不改规则语义。

**Interfaces：** old-issue-urls的新content rule检测/api/workspaces下八类旧后缀，生成/api/v0不命中；例外严格spec§2.3到期。未来P10b和P11可按精确表删自己的行。

- [ ] 把T1旧URL全清单与T6剩余逐项对照，移除无消费者旧service方法；应保留的M5/M6/M7能力只放自己的旧service，不混进未来新薄service。
- [ ] 新规则正样例至少覆盖issues、work-items、archived-issues、draft-issues、draft-to-issue、search-issues、entity-search、user-issues，负样例含/api/v0及普通页面路由；拒绝拼接隐藏规则命中。
- [ ] 每暂留条目精确文件/匹配样例/count/until/所属Task，列表到M4/P11、P11详情到M4/P12、P15清理的projectIssuesSearch等到M4/P16、M5–M7各自M；类型过渡四文件与URL例外全命中分账（spec§2.3）；失去命中的例外删除，不留宽目录规则。
- [ ] `node tools/keywords.mjs`通过；在隔离临时变更中新增未登记旧URL命中应失败、增第二个命中应count失败、把until改当前phase应过期失败；每次恢复原文件hash并再绿，不往产品留探针。
- [ ] 按 `docs/v0/M1-frontend-trim/plans/closeout.md` 附录A的完整原脚本，在 ignored `.superpowers/sdd/P10a-web-issue-types/dead/`恢复 `deadsym.mjs`、`domains.mjs`、`lib/program.mjs`，不放入tools/。先 `pnpm --filter web exec react-router typegen`，从worktree根运行 `node .superpowers/sdd/P10a-web-issue-types/dead/deadsym.mjs --tsv > .superpowers/sdd/P10a-web-issue-types/dead/current.tsv`，再 `node .superpowers/sdd/P10a-web-issue-types/dead/domains.mjs .superpowers/sdd/P10a-web-issue-types/dead/current.tsv --rows M4`；对本段改动文件剩余成员逐项读者核对，脚本失败不当作零输出；`make knip`、`make lint-web`通过。真实下降多少就调多少上限，不调高。
- [ ] 执行G；提交 `feat(M4/P10a): 约束旧工作项地址并清理换型残留`。

## Task 8：阶段回归与页面完成线核验

**Files：** `e2e/stories/smoke/s2-web-app.spec.ts`；新（实施时确定独立文件）`e2e/stories/smoke/s2-issue-list-removal.spec.ts`；受影响Vitest及必要 `C/components/issues/issue-layouts/removal.test.tsx`。前置数据复用fixtures/api、auth、test、browser和mounts，不改服务端增加M6/M7 API。

**Interfaces：** S2四身份既有请求/console断言继续；新增真实登录页面验证列表/独占操作及其旧请求已经移除。按负责人A裁定，领域自身页面整体验收归M6/M7，P10a保存其精确未接通记录；不将拆除边界通过称为整页零失败。原watchPage和M4页面门禁不变。

- [ ] 先把六条路由及其父layout/页头/自身领域fetch逐一列清：cycle/module/project-view/workspace-view详情、archived-cycle/module。API创建workspace/project/身份可用；M6/M7域实体当前无API创建，不用SQL或route.fulfill伪造它们。
- [ ] 选择真实可达的默认工作区视图、空/不存在领域详情等路径并明确其预期；不能用被权限挡在外层、未挂载页面的测试证明已删除列表。真正需要有效cycle/module/view实体才覆盖的分支，组件层明确用已有store fixture验证结构，不冒充e2e页面事实。
- [ ] 新拆除故事用已有API/auth/项目成员fixture真正挂载六类入口，确认工作项列表和独占操作无挂载/无旧请求；新增回归 `preserves_domain_loading_without_issue_lists` 以有效领域store fixture保留真实组件，核T2两处领域加载迁出与数据读取，不模拟后端成功。缺少有效领域实体的真实页面分支只记组件覆盖，不冒充E2E事实。
- [ ] 导航前启用原watchPage，原始数组不修改并完整附报告；按spec§2.4的8个已知自身GET路径作为最低核对清单，对实际父layout/页头/加载链逐项补方法、完整URL/query、触发条件、状态、源码归属、证据和M6/M7接收期限。watchPage的API数组只存path，analytics的query从requestUrls核。旧工作项列表/已删动作请求必须为零，不能记进领域自身账；实际旧列表还用IssueService的issues/issues-detail地址，按调用链/query核；project-view筛选与保留详情共享view URL时，以引用/组件来源和次数证明筛选读者已删，不能仅凭URL放行；workspace单视图URL不自动归为保留账。没有归属的请求、500/无响应、其它query和未归因异常不自动放行。
- [ ] 拆除故事按精确请求集合核自身失败记录与原始观察一致，保留全部失败而非清数组/改watchPage/按旧API前缀或4xx泛放行；M4已有S2断言不受影响。将完整页面零失败及有效领域实体回归明确交M6/M7，不给空service、隐藏领域取数或扩大P10a实现后端。本Task验收按已裁定的拆除边界关闭，自身页面未接通项继续归所属里程碑。
- [ ] 运行现有S2四身份（先 `make build`，再 `pnpm --dir e2e exec playwright test stories/smoke/s2-web-app.spec.ts --project=chromium`）及新拆除回归，确认命中真实用例；不把signed-out重定向当四类页面成功。
- [ ] 运行T5/T6定向与全部 `TURBO_FORCE=true make test-web`；最后执行F。任何W11只按06处理，不专项调查。
- [ ] 执行G，提交 `test(M4/P10a): 核验换型与列表移除后的入口边界`。所有完成线满足前不得宣称阶段完成。

## Task 9：整分支deep-review与交接记录

**Files：** 本spec/plan、`reviews/P10a-web-issue-types-review.md`、新当次序号 `docs/reviews/re-NN-M4P10a整体审核.md`；`docs/v0/frontend-changes.md`、`docs/v0/M4-issue-core/M4-design.md` §13.2/§15、相关输入handoff处理结果。

**Interfaces：** P10b接已集成P10a main（仍要另获集成授权），重新核对spec§2.3/实际文件清单和所有未完成过渡；P11/M6/M7各接明确条目，不能把P10a文档预检当后续行为验证。

- [ ] 自查spec§4逐条附本期证据，未运行/不满足保持未勾；记录每Task起止SHA、独评/修复、F和最终分支CI身份，保留首次失败与修复红绿。
- [ ] 使用deep-review审查基线4599a9cf到当前最终HEAD全部diff，逻辑/集成/覆盖三视角；聚焦132消费者完整性、四类删除边界、Issue/Detail/null/表单local、过渡期限、真实页面请求，修复后按受影响范围独立复审。
- [ ] 前端改动清单写明实际删除范围及未来领域页面仍未接通的代价；M6加回迭代/模块列表、M7加回项目/工作区视图列表及Partial<Issue>收集箱过渡，连同各自页面完整可用/零失败验收与逐请求记录写进review§6和设计§13.2。接收方接通时替换旧接口、删除过渡记录并核有效实体页面；不提前创建最终M5–M8 closeout或关handoff总status。
- [ ] 设计§15只写“实施与终审完成、集成待办”，M4§14和总体§9.4仍开放；阶段main集成完成后再有后续真实回执，不在本Task自行合并。
- [ ] 纯文档静态路径/链接/任务覆盖/时间态核对，独评后提交 `docs(M4/P10a): 记录换型终审证据与后续数据层交接`；准确最终HEAD CI核验与local evidence分别记录。

## 作者自查（准备期）

本计划按spec§1–7覆盖九任务、类型/过渡/删除/验证责任，并给五类Review Focus指定Task。新接口只在T6定义，后文不改名；未把计划命令写成已运行。代码原型、逐Task重放、运行变异、F、整分支产品审查均是后续实施动作；本轮独立静态预检结论由review另记。
