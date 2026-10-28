# M4/P10a 工作项换型与列表收窄：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M4/P10a `web-issue-types`，设计已批准的 P10 第一段，9 Task |
| 日期 | 2026-10-28 |
| 状态 | 负责人已裁定 P10A-S1 采用 A；验收归属已同步并通过独立定向静态预检，无开放发现；负责人已授权按通过预检的 spec/plan 实施，当前推进 T1 |
| 代码基线 | P9 合并 main `4599a9cfc5be90b2c242977f194157778a5f6ec9`；准确 main CI 与归档清理以 P9 review 集成回执为准 |
| 分支 | `codex/m4-p10a`；`.codex/worktress/codex/m4-p10a`；主目录 HEAD 保持 main |
| 输入 | [M4 设计](../M4-design.md) §3.1/3.2、7.1–7.9、9.5/9.6、12/P10a、13.1/13.2、16、17.2/I7、17.3/M7；[02](../../claude_to_gpt/02-method.md)、[03](../../claude_to_gpt/03-rules-and-pitfalls.md) |
| 计划与评审 | [plan](../plans/P10a-web-issue-types.md)、[review](../reviews/P10a-web-issue-types-review.md)；后续 [P10b spec](P10b-web-list-data.md) |

准备期未运行完整原型、未逐任务重放、未开始 T1；这是历史边界。2026-10-28 负责人已授权按通过预检的 spec/plan 实施 P10a，当前从 T1 开始逐任务执行，尚未完成本阶段验收。以下新增符号和文件都是实施目标，不代表已经存在。已批准的 P10a/P10b 拆分保持：先删除新实体没有的读取，再机械换型；P10b 接列表数据层，P11 接详情数据层。

## 1. 目标

1. 用 `@nerve/api-client` 生成的 `Issue`、`IssueDetail` 取代 `TIssue`、`TBaseIssue`、`TIssueMap`，列表、详情、浮层、弹窗、收集箱等同一份工作项的读者在本段一起编译。
2. 删除迭代、模块、项目视图、工作区视图的工作项列表及只为它们存在的 store、筛选、布局分支，不保留 Plane 响应兼容层。
3. 保留这些领域本身的页头、详情及自身取数；M6/M7 接通前这些页面不保证完整可用，且没有工作项列表。这两项可见代价分别写进前端改动清单和后续移交，不能将保留组件等同于已经接通。
4. 每 Task 结束可编译，现有 S2 四身份与 Vitest/knip 保持通过；旧 M4 service 仅保留设计 §7.2 已批准、明确到期的过渡，不声称本段接通列表或详情 API。

## 2. 交付物

### 2.1 路径与职责

本页 `C/`=`web/apps/web/core/`，`A/`=`web/apps/web/app/`，`P/`=`web/packages/`；前缀仅为缩写，不是新目录。基线实际有 132 个文件命中三核心类型任一，四个待删 store kind 在 50 个文件中被消费；设计“131/53”保留为当时估计，本期不得拿旧数冒充新盘点。

| Task | 精确入口与职责 | 交付接口/依赖 |
|---|---|---|
| T1 | `tools/keywords.json`；`api/dist/openapi.yaml`、`P/api-client/src/schema.gen.ts`/`index.ts` | 实施时 phase 设 `M4/P10`；生成类型已存在，重生成/幂等核对，产出当前类型与消费者清单 |
| T2 | 删除 `C/components/issues/issue-layouts/roots/{cycle,module,project-view,all-issue}-layout-root.tsx`；A 中四类详情页与 archived cycle/module；`C/components/{cycles/archived-cycles,modules/archived-modules}/root.tsx`；活动迭代、进度栏、transfer 组件 | 先断列表挂载及共用独占分支；自身领域页头/详情保留；存储根暂留到 T3 |
| T3 | 删除 `C/store/issue/{cycle,module,project-views,workspace}/` 的 issue/filter/index；改 `C/store/issue/root.store.ts`、`C/hooks/store/use-issues.ts`、`P/types/src/issues/issue.ts`；`C/services/{cycle,module,workspace}.service.ts`；`C/store/global-view.store.ts`、`C/store/issue/issue-details/issue.store.ts` 及其直接成员调用链 | 删除 CYCLE/MODULE/PROJECT_VIEW/GLOBAL 与全部分支、旧 TIssuesResponse 方法；保留 PROJECT/PROFILE/ARCHIVED/DEFAULT/WORKSPACE_DRAFT |
| T4 | `C/components/issues/issue-layouts/` 的 spreadsheet 列、properties、list/kanban/calendar；`P/constants/src/issue/`、`P/types/src/issues.ts`、`P/utils/src/work-item/` 的相关定义 | 删除非 M4 实体字段读取，旧列表形状/排序枚举仍留 P10b；每 Task 编译 |
| T5 | `C/components/issues/{issue-detail,issue-detail-widgets,peek-overview,issue-modal}/`、`C/components/inbox/`、`C/components/power-k/`；对应 `C/store/issue/issue-details/` | 详情/表单/收集箱读取收窄；克隆来源等 UI 数据与 API 实体分离；不重做 P11–P15 的新行为 |
| T6 | `P/types/src/issues/issue.ts` 和附录 B 的全部核心消费者；`C/store/issue/issue.store.ts`；§2.3 四个旧 service 及 inbox 参数 | 删除三旧类型，直接导入 Issue/Detail；canonical、描述判别、可空性和 `_id` 字段对齐 |
| T7 | 本期改动文件、各受影响 `package.json` 的 lint cap、`tools/keywords.json` | 死成员/prop/oxlint 收口；old-issue-urls 按命中登记到期例外；knip无残件 |
| T8 | `e2e/stories/smoke/s2-web-app.spec.ts`；新 `C/store/issue/issue.store.types.test.ts`；按需新 `C/components/issues/issue-layouts/removal.test.tsx` | 四身份 S2；真实入口的列表拆除边界、nullable/描述保留回归；不假造 M6/M7 后端 |
| T9 | 本 spec/plan/review；`docs/v0/frontend-changes.md`、设计 §13.2/§15；相关输入 handoff 的本期处理结果 | 整分支 deep-review、修复复审、真实验收证据；终审不自动授权合并 |

T1→T2→T3→T4→T5→T6→T7→T8→T9。T2/T3 的 imports、barrel、调用方必须在各任务内部闭合，不允许靠后续 Task 才恢复编译。T3 还须清理不出现四值枚举的直接成员读取：GlobalViewStore 对 `workspaceIssuesFilter/workspaceIssues` 的同步，IssueDetailStore 对 `cycleIssues/moduleIssues` 的五个方法及其接口/详情/浮层/选择器回调，以及 `C/store/inbox/inbox-issue.store.ts` 和 `C/components/power-k/ui/pages/context-based/work-item/commands.ts` 的间接调用。为编译闭合所需的 T5 收窄子集提前同 T3 删除，不能保留无动作 handler、恢复被删 store 或等 T5 救编译；全体四组 store 成员名和五个删除方法名分别作全局引用检索；保留同名合法领域方法必须逐项说明读者与归属。9 Task 是设计既定边界；若现场实测超过 16 Task/约 1500 行计划的限度，按设计 D2 先记录细分，不隐藏超量。

### 2.2 实体与局部状态

生成 `Issue` 包含非空 `workspace_id/project_id/state_id`、非空 priority、数组 assignee_ids/label_ids、可空日期/parent_id/created_by_id/updated_by_id；`IssueDetail` 才有必填 `description_html`。使用当前生成定义，不复制字段形成兼容实体，不把服务端非空字段改为可空以迁就旧 UI。

目标接口在 `C/store/issue/issue.store.ts`：

```ts
issuesMap: Record<string, Issue | IssueDetail>;
addIssue(issues: (Issue | IssueDetail)[]): void;
getIssueById(issueId: string): Issue | IssueDetail | undefined;
getIssueDetailById(issueId: string): IssueDetail | undefined;
updateIssue(issueId: string, issue: Partial<IssueDetail>): void;
```

`getIssueDetailById` 只在值存在且 `"description_html" in value` 时返回，不补空字符串冒充已取详情。普通展示读 Issue，编辑描述读显式 Detail。本段保留现有 addIssue 对字段的合并语义，不提前实现 P10b 的 updated_at 决胜、确认队列或列表协调；但列表行不得因为换型改成替换整个对象而抹掉已取描述。

`TIssuesResponse` 在 P10a 仍保留旧容器形状，仅其行换为 `Issue`；完整容器迁移归 P10b。评论、关联等其他 DTO 名称留 P11。工作项 map 的类型直接 Record，不导出“新 TIssueMap”别名。

`tempId/sourceIssueId/state__group` 是旧前端附加值，不属于新实体：没有写者的 tempId UI 分支删除；若仍有活写者，其 pending 标识在拥有它的 UI/store 中单独维护。克隆来源改为表单/弹窗的独立 `sourceIssueId?: string` prop 或局部状态，不能进入生成实体或 HTTP 载荷；状态组从现有状态目录取。草稿场景继续由现有显式草稿上下文表达，不在 Issue 上恢复 is_draft。新的本地表单类型只声明表单实际数据，不建立全体 Plane 字段兼容层。

### 2.3 有界旧 service 过渡

| 文件（均在 C/services） | 本段动作 | 到期 |
|---|---|---|
| `issue/issue.service.ts` 列表方法、`issue/issue_archive.service.ts` | 行类型改 Issue，保留旧地址到列表数据层替换 | P10b 删除，规则 until M4/P11 |
| `issue/issue.service.ts` 其余 M4 方法、`issue/issue_relation.service.ts`、`issue/workspace_draft.service.ts` | 核心实体签名改 Issue/Detail；详情 DTO 形状仍属 P11 | P11 删除，规则 until M4/P12 |
| `inbox/inbox-issue.service.ts` create/update | 参数 Partial<Issue>，其余收集箱字段来自所属 DTO/调用上下文 | M7，规则 until M7 |
| 合法 M5/M6/M7 旧 service | 仅保存仍有消费者、属于该 M 的方法；M4 service 中这些方法按设计 §7.3 迁往对应所属旧 service | 各自 M，不成为新 M4 service 的能力 |

**类型过渡与 URL 例外是两份账。** 上表只列用新 Issue/Detail 临时类型化旧响应的四核心文件；其他尚未迁移的 M4 旧地址同样要逐命中登记，不能误删或漏放。`issue_activity/issue_comment/issue_reaction/work_item_version` 等详情资源旧地址、`user.service.ts#getUserProfileIssues`、`workspace.service.ts#searchEntity` 按 P11 删除（until M4/P12）；`project/project.service.ts#projectIssuesSearch` 的调用方在 P12/P14改接，旧文件及残余搜索方法在P15清理（until M4/P16，P14若已无读者应立即删并同步账）。`workspace.service.ts#searchWorkspace` 在P15删除；其当前 `/search/` 是否被新规则覆盖须按实际正则核，既有plane-workspace-urls规则继续有效，不伪称八类新规则已覆盖它。M5/M6/M7例外只限实际属于它们的路径/方法。

这只是类型过渡，Nerve 没有对应 Plane 路由，不宣称运行时已接通。`old-issue-urls` 规则在 `web/` 检测 `/api/workspaces/…` 下的 issues/work-items/archived-issues/draft-issues/draft-to-issue/search-issues/entity-search/user-issues 旧地址。每个暂留命中登记文件、匹配样例、精确 count、原因和 until，不使用整目录豁免；P16 终态只剩 M5–M7。

### 2.4 删除后的页面与回归

移除列表的位置：cycle、module、project-view 的详情页；`workspace-views/[globalViewId]/page.tsx`；archived cycle/module 入口上的工作项列表分支。归档迭代/模块目录根本身不是Issue列表，保留其目录与领域取数，不按名称整删。保留自身页头/详情，不新增说明性占位产品、不用返回空数据伪装完整列表。

共用组件只为这些列表存在的“添加已有”、日历 addIssuesToView、特有空态、活动迭代/进度侧栏列表 store 调用、迭代 transfer 弹窗随之删除。项目日历/子项/关系共用的 ExistingIssues 选择器本身保留给 P12/P14，不能因为移除 M6 调用方把 M4 选择器也删掉。

**负责人裁定 A（2026-10-28）：** S2 保持四身份现有主路径和请求/控制台断言，M4 设计 §3.1 已纳入的页面标准不降低；P10a 不提前把项目工作项页加入新请求断言，那是 P10b。迭代、模块、项目视图、工作区视图及归档目录的自身页面完整可用、整页零失败验收归 M6/M7。P10a T8 验收它们的工作项列表挂载、独占控件、旧列表请求已经移除，并如实记录保留领域请求未接通的代价。

T2 先区分被删列表根里的列表取数与领域自身加载；仍为保留页头/详情所需的加载迁到所属页面/领域组件，继续走所属 store/service，不能因删除 root 丢掉它、借空态消除失败。对保留内容用真实组件和有效领域 store fixture 核结构，不模拟 HTTP 成功；缺少真实 M6/M7 实体前置时不冒称已做有效详情的端到端验证。

T8 在实际进入目标页面（而非权限重定向）前启用原 `watchPage`，保存未过滤的 requestUrls/apiRequests/oldApiRequests/apiFailures、pageErrors/console/CSP 原件。分别证明：旧工作项列表及已删动作请求为零；未实现的领域自身请求逐项有方法、完整 URL/查询、触发条件、状态、源码归属、证据及接收里程碑；其他请求/错误不被这份记录收容。不得改基础 watcher、清空失败数组、笼统按 `/api/workspaces/` 或 4xx 放行，也不得将这种拆除边界通过写成整页零失败。控制台记录若需归因，只能关联到对应领域请求/调用链，不能吞掉无归属异常。

已知自身请求的最低核对清单与后续所有者见下表；这是静态证据，不是穷尽清单或本轮网络实测。T2/T8 必须按实际父 layout/页头/加载链补齐精确记录；新增列表请求、删除动作残留和无归属失败仍属缺陷。未来 M6/M7 接通时替换所属旧接口并清除此项未接通记录，以真实有效实体重跑相应整页验收，不能继承过渡记录永久放行。

| 入口 / 条件 | 已知自身 GET 路径（前缀 `/api/workspaces/{slug}`） | 验收接收方 / 到期 |
|---|---|---|
| 模块详情，合法项目成员，含不存在 moduleId | `/projects/{project}/modules/{module}/` | M6 接通模块页面时 |
| 项目视图详情，合法项目成员，含不存在 viewId | `/projects/{project}/views/{view}/` | M7 接通项目视图时 |
| 已归档迭代 / 模块目录，项目参数有效 | `/projects/{project}/archived-cycles/`、`/projects/{project}/archived-modules/` | M6 接通各归档目录时 |
| 迭代自身详情，原列表 `fetchParentStats` 的领域加载迁出后 | `/projects/{project}/cycles/{cycle}/` | M6 接通迭代页面时 |
| 有效迭代详情的统计 / 进度 | `/projects/{project}/cycles/{cycle}/analytics?type=issues`、`/projects/{project}/cycles/{cycle}/progress/` | M6 接通迭代页面时 |
| 工作区视图目录（原 root 只要 slug 即加载，含默认页）；自定义视图名称 / 详情依赖它 | `/views/` | M7 接通工作区视图时 |

当前 Nerve 旧 `/api/` 路由由 fallback 回答 404；实施时仍记录实际状态，不能将 500、无响应或其它未知错误自动归为本项。`watchPage.apiRequests` 仅保留 pathname，查询串须从 requestUrls 或单独原始请求记录核对，不能丢掉 analytics 的 `type=issues`。本表不改变 §2.3 的类型过渡和关键词例外到期账。 cycle/module的列表 `user-properties/`、active-cycle issue列表、transfer及已删detail动作也不得收入自身账；实施按完整调用链列反向清单，这些只是已知例子。 归属还须按调用者核：旧 cycle/module/project-view 工作项列表共用 `IssueService.getIssues` 的 `/projects/{project}/issues/`、`issues-detail/` 及 `v2/issues/`，workspace列表还有工作区级 `issues/`、`issues-detail/`，不能只找带 cycle/module 的URL；project-view旧筛选与保留详情页会共用 `/views/{view}/`，必须有静态引用、真实组件调用来源和次数证据证明已删读者消失，不能只按URL留账。工作区 `/views/{view}/` 原仅供已删筛选读取，不能自动列为保留领域请求；如迁移领域加载确需采用它，须记录真实读者和理由后精确核对。

### 2.5 尺寸与现有大文件

新生产逻辑文件约 400 行内；新测试按行为拆文件，生成物例外。既有超 400 行的大型布局/store 文件允许本期机械删字段/换类型，逐文件把改前/改后行数与原因记在 T1 清单及最终 review；不得以这个例外新增大段逻辑。`C/store/issue/helpers/base-issues.store.ts` 的重写已归 P10b，本期仅必要删分支/换类型；`issue-modal/base.tsx`/`form.tsx` 等本段仅删除越界读取与分离 UI 状态，不附带重构 P14 的完整交互。附录 B 是确定的消费者集合，T1 对其中超过 400 行者生成尺寸表，未列入的新大文件须局部裁定后记录。

## 3. 与设计的差异和补充

| 编号 | 裁定：结论 | 理由 | 错了的代价/验证 |
|---|---|---|---|
| D1 | P10a/P10b 实施关键词 phase 都用 M4/P10，准备期不改 | 现有解析只支持数字，P1a/P1b已有同例；不改守卫语法 | T1 keywords 正常运行、到期 count 精确；提交/文档仍区分 a/b |
| D2 | 准备期只静态预检，无原型/重放/产品测试结论 | 当时用户明确暂不实现；02§3允许调整方法 | 实施时所有验收/变异必须实跑，不能继承“文档已过”结论 |
| D3 | canonical 用生成 Issue/Detail 与显式 detail accessor，不造 optional entity | 普通 Issue 不能读描述，已取 Detail 要能保留 | T6 编译/nullable测试，Issue 后到不抹描述；P10b另接版本规则 |
| D4 | old-issue-urls 为设计点名的 M4 过渡按精确命中登记到期 | §7.2允许到P10b/P11，§7.9的M5–M7是最终余项 | T7规则正反样例与失配count检查；P10b/P11分别删账 |
| D5 | 清理 UI 附加字段，克隆来源留独立表单prop，状态组由目录读 | 不能用生成 Issue 的交叉类型恢复旧兼容实体 | 克隆来源不进入payload；目录未到不伪造group；T5/T6行为/编译核对 |
| D6 | P10b 先修归档本地键，P13页面复核；准备文档统一设计旧句 | §12 P10b T9及§7.3已经指定；§3.17的P13是旧分配 | 只修责任文字，本段不改归档代码；P10b刷新恢复测试 |

**已裁定 P10A-S1：** 负责人选择 A，保持列表删除范围，自身页面的完整可用/零失败验收归回 M6/M7，按 §2.4 精确登记未接通代价；M4 范围门禁、关键词和架构守卫不变。决策记录与定向复审见[预检报告](../../../reviews/re-10-M4P10预检审核.md)。文档同步与定向静态预检已完成；负责人随后已授权按本 spec/plan 实施 P10a，集成仍须另获授权。若后续发现必须改变新实体、重新接 M6/M7 列表或共享模块协议，按03先提交已完成工作再上报。

## 4. 验收标准

下列全部为未来产品完成线，准备阶段不打勾。

- [ ] T1–T8 每个 Task 结束类型检查通过，生成物由生成器产出且幂等；没有新未批准依赖。
- [ ] TIssue/TBaseIssue/TIssueMap 的定义、导入、使用、注释引用在 web 中消失；核心实体直接用 Issue/IssueDetail；没有复制兼容 DTO 或 unsafe 双断言。
- [ ] 四类列表根、页面列表挂载、12 个 issue/filter/index 文件、三旧 service 列表方法、四 EIssuesStoreType 值/分支和独占共用组件均删除，合法其他消费者仍可用。
- [ ] 不再读取新 Issue 没有的 M5/M6/M7 字段；created_by_id/updated_by_id/null、Detail 描述和独立 UI 状态正确。
- [ ] T6 描述保留/可空字段回归及相关现有 Vitest 通过；T7 old-issue-urls 精确例外/样例有效；knip无死导出/依赖，改到文件的死成员/警告处理有记录。
- [ ] S2 四角色保持现有请求/控制台断言通过；六类受影响入口的工作项列表、独占操作及其旧请求已移除，保留领域内容/加载边界有组件回归；领域自身未接通请求按 §2.4 精确登记并交 M6/M7，原始失败证据保留，未把拆除边界通过写成整页零失败。
- [ ] 前端改动清单和 M6/M7 恢复列表/收集箱参数移交明确；P10b/P11旧service到期账精确；输入handoff总状态保持open。
- [ ] 最终完整本地七门禁在准确提交通过，记录原件/退出码/计数/skip/cache；准确最终分支CI全绿；deep-review整分支覆盖逻辑、集成、测试并完成修复复审。
- [ ] 获本阶段明确集成授权后 --no-ff 合并；准确main CI全绿、证据归档逐hash核验后清理。实施授权不包含集成，不因这行自行合并。

## 5. 不在本段与后续接手

| 接收方 | 保留责任 |
|---|---|
| P10b | 列表service/store、Reconciled.fetchPart/PagedGroups/canonical版本、21码比较器、preferences/filter迁移、草稿filter删除、oneAtATime、挂载新请求与S2增长；从P10a完成集成的main再核基线 |
| P11 | 详情DTO/service/store、JSONContent/comment_json/is_bot、模块级实例、按需拒绝/标签create/弹窗宿主；结束§2.3其他M4旧service |
| P12–P15 | 列表/看板/日历新故事、筛选/归档/个人、详情/创建/父子/关联、协作/草稿/搜索；C1–C11分配照设计 |
| P16 | 死成员终态、全仓no-shadow、关键词终态、剩余getTabIndex/projectMemberIds消费者 |
| M5/M6/M7 | 新字段随本模块真实数据恢复；M6迭代/模块和M7视图列表在新引擎重建；M6/M7分别接 §2.4 自身页面完整可用/零失败验收及精确未接通请求清单，接通时替换旧接口、删除过渡记录并核真实有效实体；M7收集箱Partial<Issue>旧参数过渡到期 |
| M4收尾/M8 | 所有输入status关闭、M5–M8四份M4-closeout、总体§9.4、M4§14、C12与完整性能/部署 |

本段不重写后端、不加接口或依赖、不把P9清理/事件再实现、不放宽架构守卫，不改 plane/、refer/、现有Docker。

## 6. 风险

| 风险 | 对应措施 |
|---|---|
| 同一canonical牵动132个消费者，中间提交不能编译 | T2–T5先收窄；T6只类型/命名/null；每Task单独类型检查，完整清单逐项核 |
| 删除四类列表误删共用选择器/合法领域详情 | T2/T3按引用闭合，仅删独占分支；T8入口/组件边界回归，M4 ExistingIssues保留 |
| 列表Issue覆盖已取描述、空作者引起崩溃 | Detail显式判别；T6两种来源先后与null测试；P10b后续版本规则另验 |
| 新URL守卫过宽掩盖旧请求或过早卡住P11 | 精确文件/样例/count/until，独立核到期账；不整目录放行 |
| 只换类型让旧API被误写成已接通 | §2.3明确运行时限制；阶段报告区分编译证明与实际请求；P10b/P11按责任结束过渡 |

## 7. 输入交接与关闭条件

P1a–P4的§5/评审§6已交付的Issue/Detail、nullable creator、parent遮挡、描述语义作为合同使用，不重做后端；P5/P8留给P10b的生产码点比较器、共享issue-order.json和当前21码本段不提前关闭。P6/P7/P9无新增P10a后端责任。

M1-P2/平台删字段输入在本段关闭“前端不再读取”子项；M1-closeout的死成员/prop/oxlint按实际改到文件记录；M2§12、M3§5/6的会话/列表/旧preferences主要在P10b和后续，本段不声称整项关闭。M6/M7列表恢复、自身页面验收及逐请求过渡记录、收集箱参数过渡写进本段review§6与设计§13.2，最终handoffs/M4-closeout.md由M4收尾统一创建；本次只登记责任，不提前关闭输入handoff或创建最终交接。

## 附录 A：准备验证（2026-10-28）

已做：阅读上级设计、前序spec§5/review§6和M0–M3交接；只读盘点当前生成类型、消费者、store/service、关键词解析和测试入口。未做：原型、逐Task重放、产品测试、运行变异、P10a实施、P10a合并。静态预检结论及修订轮次以本段review/独立预检报告填写，作者不预填通过。

P9 main CI 是基线事实，由P9集成回执提供；它不能代替P10a行为验证。完整原始输入盘点在主项目 `.superpowers/sdd/P10-preparation-20261028/requirements.md` 和 `code-inventory.md`，属于本机准备材料，tracked文档必须自身足以实施。

## 附录 B：基线核心类型消费者

T1 在源 HEAD `4569a9e7386031cffb31aa37d6cf06a726b08c2c` 重新运行两条计划原文 `rg -l`：核心类型 132 文件、四 store kind 50 文件，交集 9 文件、并集 173 文件，与预检实数相同；设计旧“131/53”只是历史估计。下表分配每文件的首次收窄/删除任务及剩余 T6 换型责任；属于 T3 的五动作调用链优先在 T3 闭合，不等 T5。清单不是已修改/已完成列表。原始路径、文件 sha256/改前行数、全量 URL 及退出码保存在工作树 `.superpowers/sdd/P10a-web-issue-types/task-1-*`，最终由主控归档。

### B.1 核心类型消费者（132）

| 文件 | T2–T6 责任 |
|---|---|
| `web/apps/web/app/(all)/[workspaceSlug]/(projects)/browse/[workItem]/page.tsx` | T6 剩余核心换型 |
| `web/apps/web/core/components/browse/workItem-detail.tsx` | T6 剩余核心换型 |
| `web/apps/web/core/components/inbox/content/issue-properties.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/inbox/content/issue-root.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/inbox/modals/create-modal/create-root.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/inbox/modals/create-modal/issue-description.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/inbox/modals/create-modal/issue-properties.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/inbox/modals/create-modal/issue-title.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/inbox/modals/decline-issue-modal.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/inbox/modals/delete-issue-modal.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/archive-issue-modal.tsx` | T6 剩余核心换型 |
| `web/apps/web/core/components/issues/delete-issue-modal.tsx` | T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-detail-widgets/issue-detail-widget-modals.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-detail-widgets/relations/content.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-detail-widgets/relations/helper.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-detail-widgets/sub-issues/content.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-detail-widgets/sub-issues/issues-list/list-group.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-detail-widgets/sub-issues/issues-list/list-item.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-detail-widgets/sub-issues/issues-list/properties.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-detail-widgets/sub-issues/issues-list/root.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-detail-widgets/sub-issues/quick-action-button.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-detail/label/root.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-detail/parent/root.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-detail/parent/siblings.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-detail/root.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/calendar/calendar.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/calendar/day-tile.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/calendar/issue-block.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/calendar/issue-blocks.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/calendar/quick-add-issue-actions.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/calendar/utils.ts` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/calendar/week-days.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/kanban/block.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/kanban/blocks-list.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/kanban/default.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/kanban/headers/group-by-card.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/kanban/kanban-group.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/kanban/swimlanes.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/list/block-root.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/list/block.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/list/blocks-list.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/list/default.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/list/headers/group-by-card.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/list/list-group.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/list/list-view-types.d.ts` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/properties/all-properties.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/all-issue.tsx` | T3 枚举分支闭合；T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/cycle-issue.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/helper.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/issue-detail.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/module-issue.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/project-issue.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/quick-add/form/root.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/quick-add/root.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/assignee-column.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/attachment-column.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/created-on-column.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/cycle-column.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/due-date-column.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/label-column.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/link-column.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/module-column.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/priority-column.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/start-date-column.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/state-column.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/sub-issue-column.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/spreadsheet/columns/updated-on-column.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/spreadsheet/issue-column.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/spreadsheet/issue-row.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/spreadsheet/spreadsheet-table.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/spreadsheet/spreadsheet-view.tsx` | T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/utils.tsx` | T3 枚举分支闭合；T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-modal/base.tsx` | T3 枚举分支闭合；T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-modal/components/default-properties.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-modal/components/description-editor.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-modal/components/parent-tag.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-modal/components/project-select.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-modal/components/title-input.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-modal/draft-issue-layout.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-modal/form.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-modal/modal.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/peek-overview/root.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/preview-card/root.tsx` | T6 剩余核心换型 |
| `web/apps/web/core/components/issues/relations/issue-list-item.tsx` | T6 剩余核心换型 |
| `web/apps/web/core/components/issues/relations/issue-list.tsx` | T6 剩余核心换型 |
| `web/apps/web/core/components/modals/work-item-level.tsx` | T6 剩余核心换型 |
| `web/apps/web/core/components/navigation/use-active-tab.ts` | T6 剩余核心换型 |
| `web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/power-k/ui/pages/context-based/work-item/cycles-menu.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/power-k/ui/pages/context-based/work-item/labels-menu.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/power-k/ui/pages/context-based/work-item/modules-menu.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/power-k/ui/pages/context-based/work-item/priorities-menu.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/power-k/ui/pages/context-based/work-item/states-menu.tsx` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/workspace/sidebar/quick-actions.tsx` | T6 剩余核心换型 |
| `web/apps/web/core/hooks/store/use-issues.ts` | T3 枚举分支闭合；T6 剩余核心换型 |
| `web/apps/web/core/hooks/use-group-dragndrop.ts` | T3 枚举分支闭合；T6 剩余核心换型 |
| `web/apps/web/core/hooks/use-issue-peek-overview-redirection.tsx` | T6 剩余核心换型 |
| `web/apps/web/core/hooks/use-issues-actions.tsx` | T3 枚举分支闭合；T6 剩余核心换型 |
| `web/apps/web/core/services/inbox/inbox-issue.service.ts` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/services/issue/issue.service.ts` | T6 剩余核心换型 |
| `web/apps/web/core/services/issue/issue_archive.service.ts` | T6 剩余核心换型 |
| `web/apps/web/core/services/issue/issue_relation.service.ts` | T6 剩余核心换型 |
| `web/apps/web/core/services/issue/workspace_draft.service.ts` | T6 剩余核心换型 |
| `web/apps/web/core/store/inbox/inbox-issue.store.ts` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/store/issue/cycle/issue.store.ts` | T3 删除 |
| `web/apps/web/core/store/issue/helpers/base-issues-utils.ts` | T6 剩余核心换型 |
| `web/apps/web/core/store/issue/helpers/base-issues.store.ts` | T6 剩余核心换型 |
| `web/apps/web/core/store/issue/issue-details/issue.store.ts` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/store/issue/issue-details/relation.store.ts` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/store/issue/issue-details/root.store.ts` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/store/issue/issue-details/sub_issues.store.ts` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/store/issue/issue-details/sub_issues_filter.store.ts` | T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/store/issue/issue.store.ts` | T6 剩余核心换型 |
| `web/apps/web/core/store/issue/module/issue.store.ts` | T3 删除 |
| `web/apps/web/core/store/issue/profile/issue.store.ts` | T6 剩余核心换型 |
| `web/apps/web/core/store/issue/project-views/issue.store.ts` | T3 删除 |
| `web/apps/web/core/store/issue/project/issue.store.ts` | T6 剩余核心换型 |
| `web/apps/web/core/store/issue/workspace-draft/issue.store.ts` | T6 剩余核心换型 |
| `web/apps/web/core/store/issue/workspace/issue.store.ts` | T3 删除 |
| `web/packages/constants/src/issue/common.ts` | T3 枚举分支闭合；T4 收窄读取；T6 剩余核心换型 |
| `web/packages/constants/src/issue/modal.ts` | T4 收窄读取；T6 剩余核心换型 |
| `web/packages/types/src/inbox.ts` | T6 剩余核心换型 |
| `web/packages/types/src/issues.ts` | T4 收窄读取；T6 剩余核心换型 |
| `web/packages/types/src/issues/issue.ts` | T3 删除四枚举定义；T6 删除三核心旧类型 |
| `web/packages/types/src/issues/issue_relation.ts` | T6 剩余核心换型 |
| `web/packages/types/src/issues/issue_sub_issues.ts` | T6 剩余核心换型 |
| `web/packages/types/src/search.ts` | T6 剩余核心换型 |
| `web/packages/types/src/view-props.ts` | T6 剩余核心换型 |
| `web/packages/utils/src/distribution-update.ts` | T6 剩余核心换型 |
| `web/packages/utils/src/progress.test.ts` | T6 剩余核心换型 |
| `web/packages/utils/src/work-item/base.ts` | T4 收窄读取；T6 剩余核心换型 |
| `web/packages/utils/src/work-item/modal.ts` | T4 收窄读取；T6 剩余核心换型 |

### B.2 四 store kind 消费者（50）

| 文件 | T2–T6 责任 |
|---|---|
| `web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/[cycleId]/page.tsx` | T2 拆列表挂载；T3 枚举闭合 |
| `web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/header.tsx` | T3 枚举分支闭合 |
| `web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/mobile-header.tsx` | T3 枚举分支闭合 |
| `web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/[moduleId]/page.tsx` | T2 拆列表挂载；T3 枚举闭合 |
| `web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/header.tsx` | T3 枚举分支闭合 |
| `web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/mobile-header.tsx` | T3 枚举分支闭合 |
| `web/apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/views/(detail)/[viewId]/header.tsx` | T3 枚举分支闭合 |
| `web/apps/web/app/(all)/[workspaceSlug]/(projects)/workspace-views/header.tsx` | T3 枚举分支闭合 |
| `web/apps/web/core/components/cycles/active-cycle/cycle-stats.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/cycles/active-cycle/use-cycles-details.ts` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/cycles/progress-sidebar/issue-progress.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/cycles/transfer-issues-modal.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/issues/issue-layouts/calendar/base-calendar-root.tsx` | T3 枚举分支闭合；T4 收窄读取 |
| `web/apps/web/core/components/issues/issue-layouts/calendar/roots/cycle-root.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/issues/issue-layouts/calendar/roots/module-root.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/issues/issue-layouts/empty-states/cycle.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/issues/issue-layouts/empty-states/index.tsx` | T3 枚举分支闭合；T4 收窄读取 |
| `web/apps/web/core/components/issues/issue-layouts/empty-states/module.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/issues/issue-layouts/empty-states/project-view.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/issues/issue-layouts/kanban/base-kanban-root.tsx` | T3 枚举分支闭合；T4 收窄读取 |
| `web/apps/web/core/components/issues/issue-layouts/kanban/roots/cycle-root.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/issues/issue-layouts/kanban/roots/module-root.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/issues/issue-layouts/list/base-list-root.tsx` | T3 枚举分支闭合；T4 收窄读取 |
| `web/apps/web/core/components/issues/issue-layouts/list/roots/cycle-root.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/issues/issue-layouts/list/roots/module-root.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/all-issue.tsx` | T3 枚举分支闭合；T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/cycle-issue.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/module-issue.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/issues/issue-layouts/roots/all-issue-layout-root.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/issues/issue-layouts/roots/cycle-layout-root.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/issues/issue-layouts/roots/module-layout-root.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/issues/issue-layouts/roots/project-view-layout-root.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/issues/issue-layouts/spreadsheet/base-spreadsheet-root.tsx` | T3 枚举分支闭合；T4 收窄读取 |
| `web/apps/web/core/components/issues/issue-layouts/spreadsheet/roots/workspace-root.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/issues/issue-layouts/utils.tsx` | T3 枚举分支闭合；T4 收窄读取；T6 剩余核心换型 |
| `web/apps/web/core/components/issues/issue-modal/base.tsx` | T3 枚举分支闭合；T5 详情/UI收窄（五动作链 T3 前移）；T6 剩余核心换型 |
| `web/apps/web/core/components/modules/progress-sidebar/issue-progress.tsx` | T2 拆除/独占组件闭合；剩余枚举引用 T3 |
| `web/apps/web/core/components/views/form.tsx` | T3 枚举分支闭合 |
| `web/apps/web/core/components/views/modal.tsx` | T3 枚举分支闭合 |
| `web/apps/web/core/components/workspace/views/form.tsx` | T3 枚举分支闭合 |
| `web/apps/web/core/components/workspace/views/modal.tsx` | T3 枚举分支闭合 |
| `web/apps/web/core/hooks/store/use-issues.ts` | T3 枚举分支闭合；T6 剩余核心换型 |
| `web/apps/web/core/hooks/use-group-dragndrop.ts` | T3 枚举分支闭合；T6 剩余核心换型 |
| `web/apps/web/core/hooks/use-issue-layout-store.ts` | T3 枚举分支闭合 |
| `web/apps/web/core/hooks/use-issues-actions.tsx` | T3 枚举分支闭合；T6 剩余核心换型 |
| `web/apps/web/core/store/issue/cycle/filter.store.ts` | T3 删除 |
| `web/apps/web/core/store/issue/module/filter.store.ts` | T3 删除 |
| `web/apps/web/core/store/issue/project-views/filter.store.ts` | T3 删除 |
| `web/apps/web/core/store/issue/workspace/filter.store.ts` | T3 删除 |
| `web/packages/constants/src/issue/common.ts` | T3 枚举分支闭合；T4 收窄读取；T6 剩余核心换型 |

## 附录 C：既有大文件的机械改动例外

下表仅允许本段删除分支、导入/字段名/null机械调整；新增算法仍按400行内拆分。T1 冻结改前行数；本 Task 不修改这些生产文件，故 T1 改后行数与改前相同。后续各 Task 与最终 review 更新改后行数/新增职责，不因表中存在就放行体积增长。生成物不计入此表。

| 基线文件 | T1 改前/改后行数 | 例外理由 |
|---|---:|---|
| `web/apps/web/core/components/issues/issue-layouts/properties/all-properties.tsx` | 451/451 | 既有核心类型消费者，P10a先收窄/机械换型；后续职责重写按所属Phase |
| `web/apps/web/core/components/issues/issue-layouts/utils.tsx` | 812/812 | 既有核心类型消费者，P10a先收窄/机械换型；后续职责重写按所属Phase |
| `web/apps/web/core/components/issues/issue-modal/form.tsx` | 407/407 | 既有核心类型消费者，P10a先收窄/机械换型；后续职责重写按所属Phase |
| `web/apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts` | 418/418 | 既有核心类型消费者，P10a先收窄/机械换型；后续职责重写按所属Phase |
| `web/apps/web/core/hooks/use-issues-actions.tsx` | 685/685 | 既有核心类型消费者，P10a先收窄/机械换型；后续职责重写按所属Phase |
| `web/apps/web/core/store/issue/cycle/issue.store.ts` | 408/408 | 既有核心类型消费者，P10a先收窄/机械换型；后续职责重写按所属Phase |
| `web/apps/web/core/store/issue/helpers/base-issues.store.ts` | 1775/1775 | 既有核心类型消费者，P10a先收窄/机械换型；后续职责重写按所属Phase |
| `web/apps/web/core/store/issue/issue-details/root.store.ts` | 412/412 | 既有核心类型消费者，P10a先收窄/机械换型；后续职责重写按所属Phase |
| `web/apps/web/core/store/issue/workspace-draft/issue.store.ts` | 408/408 | 既有核心类型消费者，P10a先收窄/机械换型；后续职责重写按所属Phase |

T3 删除检查点（相对 T2 完成源）：`issue-layouts/utils.tsx` 812→812（仅移除 GLOBAL 条件）；`power-k/.../work-item/commands.ts` 418→350、`hooks/use-issues-actions.tsx` 685→345、`issue/cycle/issue.store.ts` 408→删除、`issue/issue-details/root.store.ts` 412→396。均为删除旧列表及五动作调用链，未增加算法；全量改动路径和前后行数由本 Task 源 manifest 保存。


T1 广义旧地址账冻结 `web/` 中全部 `/api/workspaces/` 字面量：23 文件、140 处。逐方法的文件/行、完整样例、count、理由与 until 见本地 `task-1-old-urls.md/json`；它广于 T7 八类规则，不能直接充当该规则匹配数或运行时请求验收。项目 `/user-properties/` 两处仍严格 until `M4/P11`；cycle/module 的四个同名筛选方法归 T3 删除，不延长到 M6。工作区 `getViewIssues` 两地址归 T3 删除；`/search/` 归 P15、until M4/P16，由既有 plane-workspace-urls 守卫负责，不声称被八类新规则覆盖。其他 M4 列表、详情、残余搜索分别按 §2.3 到期；M5/M6/M7 及广义账中的 M8 Webhook 自身地址按所属里程碑处理，不为 T7 新增目录豁免。新规则与例外只在 T7 实施。
