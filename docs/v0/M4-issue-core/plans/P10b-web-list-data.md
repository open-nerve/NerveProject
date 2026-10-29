# M4/P10b 列表数据层实施计划

> **给执行代理：** 按 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans` 逐Task执行，以 `- [ ]` 记录步骤。T1–T9既定任务门禁已完成；负责人已批准[P10B-S1权限读取失效方案A](../reviews/P10b-permission-read-decision.md)，先同步设计/spec/plan、由root核对后恢复T10。任务勾选由主控在独立评审后更新，当前不得开始T11。

> **当前状态：** T10部分实现6b406e56的22文件202项及静态检查通过，红反例单独提交2e4ca695，当前6项中1项真实失败，不能宣称T10门禁全绿。负责人已选根层方案A，合同同步不等于实现或通过。T9完成证据见review§12，T10检查点见§13；最近完整F/准确CI仅绑定T8的481c30b4。全部阶段产品验收仍未勾选，main基线仍为 `a06e3f70547b755b03bf8e6b33caafd52725d0f6`，阶段不合并。

**目标：** 交付生成接口驱动的项目/归档/个人主页列表、分组分页、确认修改协调与会话隔离，项目挂载不发旧接口。

**结构：** 薄service按会话ApiClient创建；扩充既有Reconciled.fetchPart，纯paged-groups实现原应答重放与计数；IssueRootStore协调每份列表及唯一canonical，oneAtATime串行确认。P10a核心换型是前置，P11详情与后续页面交互保持分段。

**技术栈：** 仓库锁定Node24、pnpm11.10.0、TypeScript、MobX、SWR、Vitest、Playwright、生成openapi-fetch客户端；无新依赖。测试完整门禁使用既有Go工具链与数据库运行器，不改Docker。

**Spec：** [P10b spec](../specs/P10b-web-list-data.md) §2目标接口唯一来源；[M4设计](../M4-design.md) §3.22/§9.5；[推进纪律](../../claude_to_gpt/03-rules-and-pitfalls.md)。

## 全局约束

- 主目录始终main；P10b开发固定使用既有独立worktree `.codex/worktress/codex/m4-p10b`。基线/CI 与路径、例外已纳入执行前预检，实施起点已核无漂移；负责人已授权T1–T14，阶段结束不合并。
- 准备不改 phase；T1 核既有 M4/P10，字母后缀不写进keywords解析，不放宽守卫。`/user-properties/`现until=M4/P11，T7删除。共享 exact key 按最早期限：patch/retrieve 当前同为 M4/P11，T5 删 patch 后重数剩余 retrieve 才记 M4/P12；M5–M7各自期限，详见 spec B.3。
- plane/refer只读，不动已有容器/Docker，不全局安装、不加未点名依赖、不用node_modules链接；不用stash/clean/reset --hard/force，不取keychain或其它凭据。
- 生成类型来自api-client；ProjectIssueQuery用operations.listIssues，MemberIssueQuery用operations.listMemberIssues；完整query不能冒用ProjectIssueFilters。新服务每方法一次生成调用，无Plane响应转换层。
- 发出请求的队列轮次内冻结该操作before证据，确认从pending取而非当前canonical；期间第三方版本无法证明则unknown/stale。成功树删除使该项目旧read令牌失效，拒绝/换会话不造删除代际。
- 只用确认结果，修改排队；drag的邻居IDs/实时sort_order在任务轮到发出时才算。PATCH仅IssueUpdate封闭字段，不spread实体。pending UI不写canonical或组模型。
- 原响应只存Row/key/version证据，固定boundary不回退，cursorBoundary独立，nextCursor不解析不生成；分页末页不是整组回答。partial的replay仅对resetCounters和rowParts补差，不能重复更动未回答outer/subgroupTotals。
- 比较服务器updated_at保留纳秒，不用毫秒排序；名称ASCII折叠+Unicode码位；NULL双向最后、id跟随方向；计数BigInt，生产number不安全整数拒绝，fixture原token独立解码。
- 服务每会话构造，读写/回查拒绝交回，SessionChanged不写；跟进经followInSession。已有useSessionSWR和oneAtATime/ProjectPreferencesService复用，不另造同步框架。
- shared-state无api-client/test依赖，query纯函数放web应用；否定层仍在共享types/utils/state/hook按职责删除。原getFilteredWorkItems/fromnow保留。
- 新公开错误码如确需增加，同Task更新C/lib/error-messages.ts、P/i18n/src/locales/en与zh-CN实际键文件及其测试；本计划不预期新增服务端码，现有拒绝走errorMessageKey。
- 新逻辑文件约400行；设计允许paged-groups纯模型约450行，比较器另有issue-order.ts；大型数据表/测试合理例外记录，不为尺寸删安全断言。
- 通用帮手复用C/lib/auth/{fake-nerve,fake-tab,fake-time}、C/store/{fake-root,fake-project-lists}及project/fake-projects；扩展已有 C/store/issue/fake-issues.ts 的 issueOf，加入本段共同的 list 应答和时序，禁止各测试重复近似HTTP客户端。
- 每Task独评与修复，局部自行根因解决；架构/协议变化或意外高风险先提交已完成工作再上报。
- 中文文档/详细中文提交，保留feat/fix/docs(M4/P10b)；精确add，系统/usr/bin/git push。阶段完成deep-review后先不合并，另按负责人授权处理。
- F在T4/T8/T12/T14与后续产品修复后、产品检查点push前串行执行；每Task聚焦。缓存/零匹配/编译红/skip不冒充行为证据。仅新改动/失败/未解风险才扩大重跑。

## 评审重点

1. 续页最后一页、drop扩窗后续页、raw回答缺行：T3/T4证明cursorBoundary与显示boundary分离，不能把末页的缺席当整组证明。
2. 同毫秒微秒版本、JSON大整数、较旧首次Detail：T3/T6证明真实精度、unsafe number拒绝和一次补取限制，防止缓存新旧混合。
3. 删除204带未知后代且旧分页晚到：T4/T6/T12证明确认墓碑、未知范围只stale相关项目；拒绝/404/archive不能伪删除。
4. 两个不同卡片共享邻居、前序被拒绝、换会话仍有排队：T11/T12证明发出时计算、固定队列顺序与每份确认恰好一次。
5. 角色变化、损坏旧浏览器偏好和项目请求分组词汇：T7–T10/T13证明显示错误/结束加载、不带旧会话值、不以宽string断言掩盖非法query。

## 文件与验证协议

C/P缩写、目标签名、各文件职责见spec§2.1；下方所有“新”文件/函数是实施时添加，不能在预检记为已存在。P10a删减导致的实际差异先更新计划再实施，不复活已删除M6/M7文件。

**G，每Task完成线：** 写具体行为测试→聚焦红（缺接口的编译红另记）→最小实现→同命令行为绿→表中生产变异红→恢复字节/hash后绿→受影响回归和diff-check→独评/修复复审→精确路径提交。记录argv/cwd/源码SHA或dirty摘要/raw/数量/skip/exit/恢复hash；每轮证据使用唯一前缀，已存在则拒绝覆盖，失败原件也保留。

**F，完整本地七门禁，严格依次：**

```sh
make gen-check
make lint-go
GOFLAGS=-p=1 make test
make lint-web
make knip
TURBO_FORCE=true make test-web
make e2e
```

每项退出0，web保留非缓存任务原日志，E2E默认chromium全套+firefox A4。`make e2e`已经build并传NERVE_VERSION，完整验收优先它。定向E2E先 `make build`，再从repo根 `NERVE_VERSION=0.1.0-dev pnpm --dir e2e exec playwright test stories/smoke/s2-web-app.spec.ts --project=chromium`；这是当前Makefile VERSION默认值，若当前build显式用另一VERSION，定向命令必须同值。不能因构建旧bin使测试看似绿。数据库测试不能并跑。

**类型与前端聚焦：** `pnpm --filter web check:types`；共享包变动后 `pnpm exec turbo run build --filter=web^... --force` 生成其现有本地dist，再应用聚焦；全静态用make lint-web/knip。新测试定向`pnpm --filter web exec vitest run <从web/apps/web相对的真实文件>`；无匹配不是通过。模型测试node环境；DOM测试使用现有单文件environment注解。

依赖顺序：T1→T2→T3→T4→T5→T6→T7→T8→T9→T10→T11→T12→T13→T14。T3/T4 纯模型通过后 T5 才取代旧引擎，同时交付唯一 root 写协调入口的基本实现（冻结 before、pending、每 id 队列、列表注册/确认分发）；T6 替换 canonical 并关闭旧写入 API；T11 完善拖放邻居计算，T12 扩全矩阵而非首次接线。T1只准备新service及直接测试；旧类型/消费者整体迁移与必要query纯函数在T5一同闭合，T8继续否定层删除/全语义矩阵，不造临时response转换层。

## Task 1：列表接口与核心写服务准备（约300–600行）

**Files：** 新 C/services/issue/issue-list.service.ts、issue-api.service.ts和对应test；project-preferences.service.ts必要注释；扩展已有 C/store/issue/fake-issues.ts（issueOf 已存在）。T1旧issue.service/issue_archive与列表类型仍保留供现有base使用，不加无生产调用的barrel导出；新service由测试直接import。删除与生产接线明确归T5。

**Interfaces：** 完整消费/产出spec§2.2两个service、ProjectIssueQuery/MemberIssueQuery；新服务均由RootStore传api。归档旧 service 当前三方法（列表、归档、恢复）在 T5 一并替换；归档详情页头的 IssueService.retrieve 留 P11。

- [x] 新service测试逐方法捕获method/path/body/query，list-project无project_id筛选，member relation必填；项目CSV与日期重复AND、归档`archived:"true"`、封闭写body。用FakeNerve真实client返回Problem/SessionChanged，断言原错交回无额外请求。
- [x] 红：`pnpm --filter web exec vitest run core/services/issue/issue-list.service.test.ts core/services/issue/issue-api.service.test.ts`。缺模块红只作起步，随后对错误路径/重复数组query的实现变异取真实断言红。
- [x] 实现service及直接测试，不接线旧base；项目偏好现服务已支持三字段，不重复造一层。记录T5要同时删掉的旧service/形状及消费者清单，旧非列表明确P11边界。
- [x] 绿：同定向、`pnpm --filter @nerve/api-client test`、`pnpm --filter web check:types`；T1不以旧形状无残留为门禁，该项T5接线后执行。
- [x] G，提交 `feat(M4/P10b): 接入工作项列表和确认写入的生成服务`。

## Task 2：Reconciled 部分取数与原应答重放（约250–500行）

**Files：** 改C/lib/reconciled.ts，新reconciled.test.ts；既有project/workspace/member/state/label测试回归。
**Interfaces：** `Change<V>=(value,answered?:V)=>V`；`fetchPart(part,read:()=>Promise<V>,merge:(V,V)=>V):Promise<V|undefined>`；`invalidate():void`使在外whole/part作废但保留held；整体fetch以原V作为第二参数重放，confirm直接只传当前值。

- [x] 测试 `fetchPartIsInertBeforeFetch`、`wholeFetchSupersedesAllOlderParts`、`samePartSupersedesOlderAnswer`、`differentPartsMergeIntoLatestValue`、`replaysConfirmationsInOrderWithRawAnswer`、`obsoleteFailureAndSessionChangeDoNotWrite`、`invalidatedReadsCannotClearHeldStaleValue`；deferred逐个放行，断言read调用次数和值历史，不sleep。
- [x] 红：`pnpm --filter web exec vitest run core/lib/reconciled.test.ts`。
- [x] 实现每part序号/在外期间changes袋、whole代号；finally只清自己；merge发生在当前值，然后每change拿未经merge原应答。现有单参数Change兼容，不用Issue专用类型侵入共用库。
- [x] 同命令绿，`pnpm --filter web exec vitest run core/store/project core/store/workspace core/store/member core/store/state.store.test.ts core/store/label.store.test.ts`；实际baseline文件如更名先改命令并记录，不允许不存在路径混入通过。
- [x] 变异M1去whole超越、M2删samepart序号、M3用merge后值作proof，三个分别在对应测试红；G，提交 `feat(M4/P10b): 扩充分组取数并保留重放原应答`。

## Task 3：组页模型、全序与精确版本（约450–800行）

**Files：** 新C/lib/paged-groups.ts、issue-order.ts与各test；测试只读api/testdata/issue-order.json。生产代码不import该fixture。
**Interfaces：** spec§2.4 Row/Group/PagedGroups/CounterRef/PageReadMeta（resetCounters/rowParts）/PageAnswer、decodePage、五个order函数；ListContext在paged-groups定义为scope/query+目录+currentUserID组成，字段直接使用生成类型/operationquery。

- [x] `ordersAllSharedFixtureScenarios`逐场景固定ascending_ids/descending_ids：名称/日期/优先级等合法Issue场景调用生产issueNameKey/issueSortKey/compareRows；大整数计数场景原token经context.source解BigInt，构造合法Row.key，直接调用生产compareRows。保留9007199254740993/9223372036854775807，不把bigint断言成Issue.number，不从被测sort生成expected。另加 `safeIssueCountProducesExactSortKey` 用真实Issue safe number验证issueSortKey转BigInt，`unsafeIssueCountIsRejected`验证拒绝。
- [x] `comparesSameMillisecondVersionsExactly`包含.123456/.123457、.1/.100000000、等瞬时时区offset；`refusesUnsafeDecodedCounts`对已经损失精度number拒绝，不声称恢复。`decodesMissingGroupsAsEmpty`、`separatesCursorAndDisplayBoundary`、`lastPageDoesNotCoverEarlierRows`核raw范围。
- [x] 红后实现spec数据结构与decoder；目录名ASCII折叠+codepoint，NULL最后、id同方向；page分支按groups/subgroups存在性，空组需要时建立。元数据记录原应答范围，合并值不能冒充proof。
- [x] 命令：`pnpm --filter web exec vitest run core/lib/issue-order.test.ts core/lib/paged-groups.test.ts`；附`pnpm --filter web check:types`。
- [x] M4改localeCompare、M5反向排序令NULL最前、M6id始终升序、M7Date.parse全值毫秒、M8fixture普通JSON.parse，分别实际红；恢复G，提交 `feat(M4/P10b): 建立固定分页窗口与精确全序`。

## Task 4：确认变化与三分支重放（约500–1000行，F）

**Files：** paged-groups.ts（decode/merge/confirm）；新 issue-membership.ts（三态成员与组键）、paged-groups.replay.ts（raw证据重放，直接导出 replayGroups）、paged-groups.evidence.ts（计数自己的 Row/range/每id最新确认事实与共用三分支）；新paged-groups.changes.test.ts、paged-groups.replay.test.ts。T5 分别从主模型与 replay 文件导入，不经反向 re-export；membership 对模型仅 type-only 依赖。member ListContext 必填 workspaceID（与 workspaceSlug 并列）；T5 从既有 Workspace 目录绑定 ID，未知 ID 不创建可请求 scope，T4 覆盖同 user 跨 workspace 的 confirm/replay 隔离。
**Interfaces：** spec§2.4/2.5 的可选 counterEvidence / CounterEvidence / ReadEvidence（首次partial启用、reset只换所属来源与确认事实、whole替换；不存Issue、每id无历史链）；spec§2.5 ConfirmedIssueChange/Before/GroupChangeResult；confirmGroups、replayGroups返回value+verifyIds；纯mergePages(value, answer, context)供fetchPart合并及T5复用；绝不从纯函数发请求。

- [x] 表G01–G16覆盖四旧新组合×rows在/不在×window内外，另分完整/未完成组；构造非零独立计数并断言整个结构，不只截图行id。覆盖create absent/delete null/archive/unarchive、多标签/负责人、泳道subgroupTotals独立与最外total。
- [x] 完整索引I2-1…9与R3-1…5按下表对应M4设计§3.22及spec§4 A3；本Task验证纯模型部分，真实canonical/回查行在T6/T12补齐，不在纯函数测试伪造缓存或HTTP。额外R-TAIL证明末页不覆盖lower以前，R-DROP证明扩窗后续页range仍从server cursorBoundary，D-204证明after=null不用版本比较且三种删除证据判别。
- [x] 新 `partialReplayAdjustsOnlyCountersResetByItsAnswer`：A5/B4/outer9，B旧页在外，A→B确认后A4/B5/outer9；旧B total4应答重放后仍A4/B5/outer9。再表两part交错、多值标签、单格回答保留未回答outer/subgroupTotals/rows；M48故意对未回答计数再减必须红。
- [x] 红：`pnpm --filter web exec vitest run core/lib/paged-groups.changes.test.ts core/lib/paged-groups.replay.test.ts`。
- [x] 实现按原应答/Before重放；decode标定resetCounters/rowParts，merge/replay只改本次拥有的集合；partial flat total只归该组/格，未回答计数/rows沿当前确认状态，whole才覆盖全部；total取response而非当前值再叠加，drop扩boundary且不回退，未知只stale受影响部分；第2判断返回去重verifyIds。
- [x] M9旧属读取当前已merge行、M10丢空组、M11重算boundary、M12所有partial当whole、M13总数sum各组、M14删除双减，均对应表红；恢复全模型绿、G、F，提交 `feat(M4/P10b): 按原回答重放确认修改并修正组计数`。

### T4 与后续任务的反例证据索引

以下只是职责分配，未运行行不能提前标通过。T14 汇总时须补齐全部实际证据。

| 设计反例 | T4 纯模型验证 | 后续真实调用链 |
|---|---|---|
| I2-1 双减 | 原回答之前/之后、部分计数写集合 | T12全部列表确认 |
| I2-2 判断不了版本 | Row.updatedAt与原回答三分支；承接T3精确比较 | T6服务器版本接收 |
| I2-3 重放的修改之前无定义 | raw+冻结Before，不用merge后行 | T12发出前冻结证据 |
| I2-4 窗口后退 | merge/drop的boundary单调、cursorBoundary独立 | T5真实续页 |
| I2-5 旧答覆盖唯一实体 | 不伪造canonical测试 | T6版本choose、T12真实交错 |
| I2-6 第三方移动后两列出现 | 缺行推断返回verifyIds，未知范围stale | T6/T12真实权威回读及列表失效 |
| I2-7 列表抹详情描述 | 纯模型仅Row，不复制实体 | T6真实Issue/Detail接收 |
| I2-8 空组不能插入 | 缺组/空格确认插入 | T5列表读者 |
| I2-9 列尾/满日历格丢卡片 | drop强制可见与扩窗 | T11/T12真实拖放入口 |
| M3-1 原应答旧属、M3-2 response total、M3-4 boundary不后退 | 均在T4直接验证 | T5复用纯merge/replay |
| M3-3 缺行后第三方修改 | T4返回verifyIds，不发请求 | T6/T12权威回读及stale |
| M3-5 自己的pending读不当第三方 | 不在纯模型伪造pending网络链 | T6/T12读写协调 |

## Task 5：替换列表引擎并接通确认写入口（约1100–1500行）

**Files：** C/store/issue/root.store.ts 的基本协调入口、C/store/root.store.ts 退役接线、C/lib/store-context.test.ts、C/services/user.service.ts；C/store/issue/helpers/{base-issues.store,issue-list-binding,base-issues-utils,issue-filter-helper.store}.ts；project/archived/profile/issue.store.ts；新helpers两test；首次新增C/lib/issue-filters.ts/test的spec§2.7四纯函数；删除旧列表services/type容器/枚举/映射并接上T1新service；必须同Task跟进C/hooks/use-issues-actions.tsx、新 C/hooks/use-issue-list-refresh.tsx/test.tsx（绑定实例的可见 stale 一轮刷新与显式重试）、issue-layouts/calendar/base-calendar-root.tsx及list/kanban/spreadsheet读者；group-consumers.test.tsx 覆盖真实看板续页与分组快建，子项 issues-list/list-group.tsx 同步未分组键。
**Interfaces：** root 首次产出 spec§2.9 核心写入口及§2.6会话生命周期（T11/T12复用，不另造）；BaseIssuesStore为选择门面，IssueListBinding持Reconciled<PagedGroups>；fetchFirst/fetchNext/refreshStale/acceptConfirmed/markMembershipStale，spec§2.6；首次产出toProjectFilters/toFilterExpression/projectListQuery/memberListQuery，T8不重复新建，只扩大语义测试及去否定。PageAnswer返回V，通用Reconciled不理解分页业务。

- [x] 首项关闭 W1（spec§2.6/R13，原始复现见 task-4-fix1-whole-boundary-note.md）：whole已计入窗口外create但未返回该id，不能在confirm时盲加outer/group并保持fresh。提出并裁定区分“实际发出前已持有计数来源”与“操作在外新回答来源”的最小内部证据方案；未知只stale，已知发出前Before仍有效，不能一律whole刷新。补模型和真实FakeNerve交错红→绿、错误来源判定生产变异，并保留T4 I1/M48全回归；同步spec内部形状。本项随T5独评关闭，不能移交T12/T14。

- [x] `nextPageKeepsExactQueryAndCursor`捕获group_key/sub_group_key、cursor为server原值；`staleReloadReplacesOnlyTargetFirstPage`、`pageTotalReplacesEarlierAdjustedTotal`、`pageMergesDeduplicateDroppedRow`、`subIssueRelativeDatesStayLocal`覆盖fromnow所有既有选项。
- [x] 独评修复回归：普通看板 optional 子组参数同步 count/cursor/loader/next，并保留泳道；A 刷新拒绝→展开 B→显式重试 A 后 B 仍能首次发送；真实列键→列表/看板主组及泳道预填→根写 HTTP 不发送标签/负责人 none 哨兵，保留有效 ID 与 priority:none；普通/递归子项未分组不增折叠层。
- [x] 红：`pnpm --filter web exec vitest run core/store/issue/helpers/base-issues.store.test.ts core/store/issue/helpers/base-issues-utils.test.ts`。
- [x] 同任务在 root.store.ts 实现 spec§2.9 的真实 create/update/delete/archive/unarchive 入口、每 id oneAtATime、发出前冻结 before/pending 与同代列表注册/分发；调用 T4 的 acceptConfirmed。canonical 暂用现有完整应答入口，不能调用旧 partial mutator伪造时间；T6 同提交换接 ReconciledByKey。先测 `coreWritesUseSingleCoordinatorBeforeCanonicalMigration`：请求未确认时实体/全部已取列表不动，拒绝不动；确认仅分发一次，队列前序失败后后序仍发；空/未取列表不被捏造。不得新增空方法等到 T12 填。IssueRootStore 构造时捕获不可变 owner，现有 C/store/root.store.ts 的 dispose 调用幂等 retire；每次队列发出/结算/追加读取先核 active。按 spec§2.6 加 `retiredRootDropsSuccessfulRepliesAndQueuedWrites`，以真实 store-context 切换会话和 held 200 核旧缓存/新缓存均不写、旧后续请求不发；不靠 HTTP 401 证明成功回答隔离。T10 再覆盖完整挂载矩阵，不首次补这个安全前提。
- [x] 同任务删除 UserService.getUserProfileIssues 及其旧类型/继承（保留真实 M2 新 API 方法），profile 首/续页均使用 listMember；测试捕获 relation 和项目范围参数，不能让 TIssuesResponse 删除后还存在该消费者。
- [x] 同步 hooks/详情 main-content、peek-overview/quick-add/创建表单调用签名：M4 直接 IssueCreate/IssueUpdate 白名单字段，skip_activity 源写者发 boolean、只随 description_html；不 spread 实体、不加 string 转换。M7 inbox 独立输入保留旧 string 合同。改造既有 base-issues.update-request.test 为新 HTTP/确认语义（pending不动、拒绝保持空HTML/缺属性、成功采用服务器HTML），去旧 local-only 绕写测试；其3个旧URL例外随代码一起删。
- [x] 迁移 P10a 既有 project/issue.quick-create.test.ts 和 quick-add/root.pending.test.tsx 到新服务边界，覆盖 R8 请求所有权/undefined和失败保留输入/迟答A不插B/真实应答只确认一次；保留测试的实际目的，不能只改 spy 让它空过。
- [x] 替换当前 1262 行旧 base；同步删TIssuesResponse/TGroupedIssues/TSubGroupedIssues/TGroupedIssueCount/TIssuePaginationData/TPaginationData及旧排序/分组/优先级枚举、两映射，所有consumer同提交改新形状。切T1 services并删旧列表方法/issue_archive（当前三方法；归档详情的独立 retrieve 留 P11）。首次落实必要query纯函数，旧偏好来源暂存的表达式正常翻译新query；T7才切持久化API、T8清否定。删除localoffset cursor、乐观回滚路径、旧response并入。calendar/base-calendar-root旧perPageCount/before/after/groupedBy调用改为生成limit/query与选组续页；不能只让tsc过却继续发旧字段。页边界lower来自请求捕获cursorBoundary。
- [x] 同定向加 `core/lib/issue-filters.test.ts` 绿；运行 `pnpm --filter web exec vitest run core/store/issue/helpers/base-issues.update-request.test.ts core/store/issue/project/issue.quick-create.test.ts core/components/issues/issue-layouts/quick-add/root.pending.test.tsx`、typecheck；rg全web核旧列表类型/映射无残留，make knip；M15恢复loaded-count cursor、M16移除group_key、M17滤掉fromnow路径，对应红。G，提交 `feat(M4/P10b): 以新组页模型驱动全部列表读取`。

## Task 6：工作项唯一副本与删除终态（约450–850行）

**Files：** C/store/issue/issue.store.ts、新issue.store.test.ts；新 issue-verification.ts 由根拥有，统一核实 round、id 去重、可见 scope 预算与 freshnessFloor，仅保存元数据；C/lib/reconciled.ts/ReconciledByKey增加receive及测试；root.store读令牌/项目删除水位、helpers/issue-list-binding.ts及必要use-issue-list-refresh联动；同Task改issue-details/{issue,sub_issues,link,relation}.store.ts，inbox/inbox-issue.store.ts、hooks/store/use-issues.ts 只读 map 消费；attachment 已删除，不恢复。准确写者清单见 spec B.1。
**Interfaces：** `receive(key,answer,choose)`初始化或合并并在在外read上重放choose；IssueStore `acceptRead(answer,token)`/getIssueById/getIssueDetailById/confirmChange、`refreshAuthoritative(issueId):Promise<IssueDetail|undefined>`；IssueRootStore captureRead（每次发出前核 active 并独立++readSerial，token携本代不可变 owner）/invalidateProjectReads及token守卫；核实元数据另记每id freshnessFloor；spec§2.6保留Issue|IssueDetail|null，issuesMap为派生Readonly视图。

**T6 fix1 异步边界补充：** 协调器内部 `begin(fresh)` 让显式 retry/refresh 在旧慢读仍在外时建立新去重与 scope 预算；真正 inFlight 同 id 仍共享并登记加入轮次，旧 end 只释放自身。两详情入口辅助 await 后再核原 token，子项父权威 await 后再核原 token，再允许标识/后续请求或子项 ID/分布/返回。正式回归覆盖慢 whole + 403/404 重试和重新挂载、同轮 part 与跨轮真 inFlight 去重、旧 round 结束不清新轮次，以及双入口辅助期间退役/删除和父 GET 期间成功删除。

- [x] 首项关闭 R17（spec§2.6）：direct-confirm 产生 verifyIds 后立即进入与 fetch/replay 共用的核实 round，跨副本/并发 part 去重；真实 GET、核实拒绝后的显式重试、退役无跟进均有测试。已成功的业务写保持成功，核实拒绝通过读取/可见错误状态交回，不能重发业务写。下述四路状态及 freshness 合同同 Task 一次闭合。
- [x] `latestServerVersionWinsAcrossAnswerOrder`、`listKeepsConfirmedDescription`、`staleFirstDetailDoesNotFillDescription`、`detailRetryIsBoundedAndRefusalReturns`、`thirdPartyMembershipMarksOldAndNewGroups`、`ownPendingReadDoesNotMarkStale`。
- [x] `confirmedDeleteCannotBeResurrected`用先发list后确认204再放旧list，canonical和画面均无该id且不虚算total；refused delete/SessionChanged/get404不产生tombstone；archive+unarchive能恢复。`unknownDescendantsRefreshOnlyTheirProject`区分另一项目列表保持。
- [x] `legacyDetailConsumersCannotBypassCanonicalVersion` 覆盖旧Detail/关联旧答不能降级新实体；link和sub项成功后count来自带真实updated_at的权威get，拒绝不加减。实际链为 issue 的本体和 parent、sub_issues 的整行/parent count、link 的 count、relation 整行和 inbox accept（spec B.1）；不是把这些旧成员API包装成空壳。
- [x] 同步 P10a 的 issue.store.types.test.ts、issue-details/issue.fetch.test.ts、sub_issues.fetch.test.ts、inbox/inbox-issue.request.test.ts：新版列表 fixture/接收 API 不清 relation 与描述，空 HTML/nullable 字段正确；两详情入口及 parent 单独 token、await/拒绝归属保持；父 count 改权威 get 后断言真实版本；跨项目只取唯一 state/member/label 目录，不恢复 cycle/module。inbox accept 的权威回读拒绝不能被旧 catch 吞掉，按 spec§2.6 保留不可确认状态并交回。旧资源 HTTP 自身整体迁移仍是 P11/M7。
- [x] 实现版本choose/tombstone/描述规则与只读issuesMap；完整应答捕获token→acceptRead，partial/count算术改成功后invalidate+refreshAuthoritative；根CRUD只统一确认。删除旧addIssue/partial updateIssue和direct map set，静态枚举零残留；全应用typecheck在T6过，不等P11。
- [x] 实现spec§2.6项目读代际：所有list/详情/回查在canonical接收之前检查 active/owner/token；增加 `foreignRootReadTokenIsRejected`，即使 serial/updated_at 相同，新根也不接受旧 owner；树删除先invalidate覆盖项目的whole/part，再mark stale/墓碑、新token刷新。多项目profile若覆盖失效项目则整页discard，不裁行复用旧total。
- [x] 实现缺行核实四路状态：成功清核实或第三方stale；404隐藏未经证实活值且无墓碑、相关范围至多一轮刷新；403/500保留error/stale等待显式retry/再挂载；SessionChanged丢弃旧流程不安排后续。每round共享去重id与scope预算，失败不能自动开新round。回查404/其它拒绝落定时将本id freshnessFloor设为当时readSerial；只有之后新发的token.serial严格大于此水位、项目令牌有效且版本合格的权威答能恢复。旧同版本list在canonical receive/清核实/清stale前被挡住；此水位和项目删除水位分开，普通404不冒删除。访问器先核scope和核实状态，不建第二个Issue缓存。
- [x] 命令：`pnpm --filter web exec vitest run core/store/issue/issue.store.test.ts core/lib/reconciled.test.ts`；`pnpm --filter web exec vitest run core/store/issue/issue.store.types.test.ts core/store/issue/issue-details/issue.fetch.test.ts core/store/issue/issue-details/sub_issues.fetch.test.ts core/store/inbox/inbox-issue.request.test.ts`；`pnpm --filter web check:types`；rg枚举全部issuesMap写入和 `.issues.addIssue`/`.issues.updateIssue` 并逐项核无绕过，读引用正常保留。M47恢复任一旧count mutator必须在旧消费者测试红；零匹配静态不替代行为测试。
- [x] M18按answeredAt覆盖、M19list抹description、M20旧Detail补HTML、M21读404造tombstone、M22不处理list中的tombstone，分别红；G，提交 `feat(M4/P10b): 按服务端版本维护唯一工作项副本`。

## Task 7：项目显示偏好与旧地址下线（约350–700行）

**Files：** C/store/issue/project/filter.store.ts及新test；C/store/project/preferences.store.ts/现test按需；C/services/project/project.service.ts；tools/keywords.json；e2e/fixtures/mounts.ts；stories/project/{p2-visibility,p8-project-preferences}.spec.ts、stories/smoke/s7-chinese-project.spec.ts。
**Interfaces：** ProjectPreferencesService现get/update；filter store每字段整体PATCH+确认结果，query 拼装和基本表达式转换已在 T5，T8 扩完整语义矩阵。

页面跟进还涉及 C/hooks/use-project-preferences-follow.ts及其测试、C/hooks/use-issues-actions.tsx、C/components/issues/filters.tsx 与 C/components/issues/issue-layouts/roots/project-layout-root.tsx；三故事同地址双GET的时序夹具涉及 e2e/fixtures/settings-pages.ts。页面将保存和后续列表读取分为两个会话跟进，读取失败保留列表自身重试。

- [x] `writesOneWholePreferenceField`断言body只有目标字段、同字段后续patch用前序确认结果；navigation/sort_order未动；load.preferences未自动套list，query显式转换；拒绝/session更换不写。group/order宽string经允许词汇收窄。
- [x] 红：`pnpm --filter web exec vitest run core/store/issue/project/filter.store.test.ts core/store/project/preferences.store.test.ts`。初始红仅证明新GET路径，其余条件以真实行为反例与变异补证，范围见review §10。
- [x] 实现并同提交删除user-properties两方法/例外和workItemFilters/三放行；保留仅search用途ProjectService直到对应后段，注释真实说明。
- [x] 同定向、`make lint-web`、`make build`后按F协议单套跑这三个e2e文件；若页挂载尚未T13接全，T7必须把原挂载受影响的preferences请求跟到新方法，使现有故事通过，不能留失败到T13。后续并发修复由真实store回归覆盖，下一完整浏览器验证归T8 F。
- [x] M23浅merge字段/带navigation默认回写、M24省偏好查询交由后端自动应用，真实红；G，提交 `feat(M4/P10b): 保存项目工作项偏好并移除旧地址`。

## Task 8：筛选翻译与删除否定层（约500–1000行，F）

**Files：** 扩充T5已加的C/lib/issue-filters.ts/test；P/shared-state/src/store/{rich-filters,work-item-filters}与utils；P/types/src/rich-filters和view-props.ts；P/utils/src/rich-filters；C/hooks/rich-filters/use-filters-operator-configs.ts及work-item-filters调用者；tools/keywords.json 的既定 negation-filters 守卫。
**Interfaces：** spec§2.7四纯函数toProjectFilters/toFilterExpression/projectListQuery/memberListQuery；不向shared-state新增api-client依赖。

- [x] 表F01…逐支持property/operator/value：state/groups/priority/labels/assignees/creator/mention/subscriber、none/me、两条日期AND、开区间/闭区间、空清除；回读保存值等价，不比较UI临时node ID。member排除不支持字段且relation必填；损坏存储值恢复明确默认。
- [x] 红：`pnpm --filter web exec vitest run core/lib/issue-filters.test.ts`。
- [x] 实现纯转换，删TAllAvailable*恒等alias/TNegativeOperatorConfig/isNegation/allowNegative/useFiltersOperatorConfigs及所有参数传递；先编译types→utils/shared-state→web，每Task终态全编译，不留临时any。
- [x] M25日期AND误变OR、M26日期none与实际日期条件混合（标签/负责人允许none与id并列）、M27负向层残留和空筛选不清除，分别测试/关键词红；否定层删完即启用既定静态守卫，T13再核M43。同定向、lint-web/knip，G、F，提交 `feat(M4/P10b): 统一列表筛选词汇并移除否定层`。

## Task 9：归档个人主页偏好与草稿去筛选（约350–650行）

**Files：** C/store/issue/{archived,profile}/filter.store.ts及新tests；删除workspace-draft/filter.store.ts，改其index/root.store.ts/use-issues.ts和草稿引用。
**Interfaces：** localStorage issue_local_filters同键rich_filters，scope包含既有账户/项目或目标用户视图；草稿useIssues分支不再给issuesFilter。

- [x] `archiveFiltersSurviveFreshStore`写后重建store读相同条件；`legacyArchiveKeyMigratesOnce`、`invalidStorageDefaultsWithoutLeakingScope`、`profileViewsKeepIndependentSettings`；草稿挂载不调getFilters/写偏好、不发新请求。
- [x] 红及绿：`pnpm --filter web exec vitest run core/store/issue/archived/filter.store.test.ts core/store/issue/profile/filter.store.test.ts`，附typecheck/knip。
- [x] M28写richFilters读rich_filters、M29共享账户key、M30草稿残留filter构造，分别行为/静态红；G，提交 `feat(M4/P10b): 修复本地列表偏好并移除草稿筛选`。

## Task 10：根store计算属性与列表会话取数（约350–700行）

**Files：** C/store/issue/root.store.ts及新root.store.test.ts；C/lib/store-context.test.ts；C/components/issues/issue-layouts/roots/{project-layout-root,archived-issue-layout-root}.tsx、C/components/profile/profile-issues.tsx；三同步读取utils/command-palette/issue.store；.oxlintrc.json范围。
**实际接线范围：** 四种 base layout 的首读、`use-issue-list-mount`、`use-issues-actions`、归档/profile filter 的读拒绝及 IssueListBinding 权限复核一并纳入；旧测试 fixture 改从所属 user/router 设置输入。`issue.store.ts` 已由 T6 改用所属 root，不为凑同步读处数重新添加旧引用。新增 `useSessionSWR` 的窄 dedupingInterval 选项仅供本列表设 0，见 spec §2.8。
**P10B-S1 接续范围：** `IssueRootStore` 与既有权限目录观察/retire、`issue-list-binding`、canonical及详情/verify现有读入口、列表挂载和核实去重须按 spec §2.6 的权限身份协同；`issue.store.ts` 因此需补权限证据守卫，不恢复全局 import。同步修正 `RootStore.dispose` 注释中已不存在的 issue autorun，保留 cycle/module 的实际归属，不改其生命周期。`useSessionSWR` 保持封闭配置，不新增第二缓存或框架。T10 完成这些本任务直接安全回归；T12仍做全副本交错。
**Interfaces：** root代理router/currentuser/目录computed；useSessionSWR([name,scope…]|null,fetcher)；相同scope换loginId不得复用key。

- [ ] `computedScopeClearsEmptyCatalogs`从有目录→空目录，currentUserId登出归空，不只truthy更新；`rootConstructionMakesNoStrictModeWarning`；列表hook测试换account/unknownid/guest/nonmember，断言不请求及旧答不写。
- [ ] 红/绿：`pnpm --filter web exec vitest run core/store/issue/root.store.test.ts core/lib/store-context.test.ts core/lib/use-session-swr.test.ts`。
- [ ] 去autorun及observable副本，保持只同步读代理；旧SWR都按列表范围替换（含archived direct useSWR），按需目录拒绝交回。oxlint限制同步但不提前把P11–15未迁移文件纳入无界失败。
- [ ] 保留2e4ca695真实红例，补 finally 清理并独立断言 canonical/rows/total；失权→旧答和失权→恢复→旧答均丢弃，旧请求仍在外时恢复的新GET可发出并只收新答。先行为红，再按 spec §2.6 以根范围水位修复，恢复权限前后都不暴露旧持有值。
- [ ] 真正目录观察覆盖相同签名重取、admin/member→guest、guest_view_all_features变化、工作区及项目失权；不推断 guest own-only。覆盖显式项目集合与工作区范围的 member whole/part、单项目 detail/verify、无关范围、同scope多消费者/普通卸载、在途去重和新身份；保留退役/删除/404/403/500/目录拒绝/loading及T7保存后读取回归。新增至少一个去权限水位或恢复身份的生产变异，行为红并恢复字节后绿；M31–33旧证据按实际改动决定是否复跑。
- [ ] M31key删loginId、M32不清空目录、M33忽略按需拒绝，分别红；G，提交 `feat(M4/P10b): 按会话和权限驱动列表挂载`。

## Task 11：拖放与串行请求体计算（约400–750行）

**Files：** C/store/issue/issue_kanban_view.store.ts、issue_calendar_view.store.ts；issue-layouts/utils.tsx与kanban/utils.ts等实际拖放预算sort_order入口；新C/store/issue/issue-drag.test.ts。
**Interfaces：** spec§2.9 updateIssue(id,makeBody,drop)；根drag队列→id队列单向嵌套；Drop只记neighbor IDs和组位置。

- [ ] `secondDropUsesFirstConfirmedNeighbors`安排不同卡片同列、首请求held，修改邻居后才放第二请求，断言实际PATCH中点；前序失败继续；missing neighbor按当前合法落点重算；移至列尾/空列/满日历确认后保留。
- [ ] `waitingDropDoesNotCrossSession`换会话后不再发旧队列请求或写旧view；请求body不带id/project_id；pending时模型不变且拒绝回原位置。
- [ ] 红/绿：`pnpm --filter web exec vitest run core/store/issue/issue-drag.test.ts`。删除组件提前计算sort_order传常数的旧路径，calendar月/周依生成query，不发before/after。
- [ ] M34把makeBody移出队列、M35全实体spread、M36前序拒绝卡住后续，各红；G，提交 `feat(M4/P10b): 在拖放发出时计算位置并串行确认`。

## Task 12：确认分发的全副本与竞态闭环（约450–850行，F）

**Files：** root.store.ts/issue.store.ts/base-issues.store.ts及project/archived/profile；新C/store/issue/issue-changes.test.ts；改到的create/delete/archive跟进消费者。
**Interfaces：** 复用 T5 已接通、T6 已换 canonical、T11 已闭合拖放的根协调入口，补全下列交错而非新增第二入口；每write调用同一根协调入口；实际发送前pending深拷贝before与读证据，确认取该操作冻结值，若无法证明则unknown/stale；删除先推进项目read epoch；broadcast到本代已注册scope，再canonical，恰好一次；未取不虚造value。

- [ ] create/update/delete/archive/unarchive/drag六张表各包含三列表+canonical、未取列表、server拒绝、SessionChanged、读写交错。断言请求数、确认次数、各total/rows/version，而非只测当前项目。
- [ ] B-L1 `ownReadBeforeWriteAnswerKeepsOffWindowBeforeMembership`：X在A后页、A.total=5/B.total=4、outer不变，pending read先收到本次B版本，写答确认仍A4/B5且无无谓whole fetch；另测排队前序改before、第三方版本令证据unknown/stale。M45改成确认时读canonical必须红。
- [ ] B-L2 `subtreeDeletionInvalidatesUnknownDescendantReplies`：根→子→孙（含归档），Y从未进canonical，隐藏/可见列表各有旧whole/part/get/verify在外；204后放旧答，Y无canonical/rows，旧答不能清stale；另一项目保持。并发两次删除夹一轮刷新；拒绝/SessionChanged不推进epoch。M46去掉项目token守卫或只过滤已知墓碑必须红。
- [ ] P10B-S1 A 的权限范围水位与删除水位、每id核实 freshnessFloor、全副本确认及同/异scope whole/part/get/verify交错，按 spec §2.6 核互不混淆、旧读不写、新身份收敛；不替代 T10 已要求的真实ABA和直接安全回归。
- [ ] B-L6 `missingReplayVerificationLeavesUncertainGroupsStale`：after已补回，get404后访问器不给可靠活值、相关counts stale、无墓碑/无关项目不变；受控新token空页使列表收敛。403/500保留error且不自动重复；SessionChanged不写新代、不请求后续；同round重复part仅一次get。M50在catch只throw而清stale或让每个catch自动重开round必须红。
- [ ] B-L6时序补例 `preVerificationListCannotClearUnverifiedState`：先发旧list(serial=7)，再发verify get(serial=8)，verify在外又发一份list(serial=9)；get404落定设置freshnessFloor=9，随后两旧list携相同updated_at返回，均不能恢复可靠活值/清核实或相关stale。之后新发权威get/list(serial=10)返回同版本完整Issue才可恢复；后发空页只收敛列表不造墓碑。M51让captureRead不逐请求递增或仅核版本/删除token必须红，保留普通404不推进删除水位断言。
- [ ] pending计数不被前一finally清掉；B-L3旧详情共享缓存边界也经本表核拒绝/回读真实版本。
- [ ] 红/绿：`pnpm --filter web exec vitest run core/store/issue/issue-changes.test.ts core/store/issue/issue.store.test.ts core/store/issue/issue-drag.test.ts`。
- [ ] M37只改当前list、M38提前乐观写、M39重复confirm、M40漏SessionChanged，各红；G、F，提交 `feat(M4/P10b): 将确认修改同步到会话内每份列表`。

## Task 13：项目挂载与精确请求门禁（约400–800行）

**Files：** project/archived root、profile-issues，各布局首次取数及filter HOC；新list-data.test.tsx；e2e/stories/smoke/s2-web-app.spec.ts；tools/keywords.json；改到的lint警告上限。
**Interfaces：** project页先取得允许的目录/偏好，显式生成list query；S2 REQUESTS四角色加入issues页；守卫negation-filters、issue-optimistic运行表。

- [ ] `mountedProjectListUsesGeneratedRequest`真实store+FakeNerve证明首次/已保存筛选/切project/拒绝错误态；角色guest-own-only服从服务端，非projectmember不取；archived guest不取；profile访客不取。
- [ ] S2 admin/member/guest/project non-member进入项目issues页，保存精确GET集合及失败放行，只允许预期权限页，不留下旧地址。三原user-properties放行故事也运行，不能从watch里过滤所有issues。
- [ ] 命令：新DOM测试+`make lint-web`、`make knip`；`make build`后按协议串行跑S2及三个旧放行文件，chromium。
- [ ] M41重新发/user-properties、M42绕过权限requestgate、M43negation词项回引、M44await前写canonical，各真实门禁红；恢复G，提交 `feat(M4/P10b): 接通项目列表挂载并收紧旧接口门禁`。

## Task 14：完整验证、deep-review与交接（不实施后段）

**Files：** docs/v0/v0-design.md总体7.7、frontend-changes.md、M4-design.md对应行；spec本§4、plan、reviews/P10b-web-list-data-review.md；docs/reviews下一编号整审记录；落在本段的handoff处理行。

- [ ] 汇总任务1–13所有原件/失败/变异/恢复hash，验收A1–A10逐项映射。原型未做保持事实，静态预检不冒充生产证明；过渡旧文件逐个写到P11/P12/P14/M5–M7。
- [ ] 完整F；按准确待交付HEAD推送后看CI server/web/e2e。已授权登录Chrome UI读详细失败并重跑基础设施/已知W11；第二次W11若在main合并CI仍失败按纪律上报，不擅改主线。
- [ ] deep-review固定P10b实际基线..最终候选HEAD，逻辑/集成/覆盖三个视角；抽至少15个性质变异证据并核shared Reconciled所有旧消费者回归、微秒/大整数/删除204/续页范围；局部修复后按影响重验和独立复审。
- [ ] review §1–7写真实结果；spec完成复选框只有实际通过才勾；design§15写“实施与终审完成、集成待办”，handoff总status仍open。提交 `docs(M4/P10b): 记录列表数据层验证与整分支审查`。
- [ ] 报告分支、实际main基线、spec/plan/review、准确HEAD/CI；停在未合并，等待负责人下一步，不开始P11。

## 作者自查（准备期）

- [x] spec§2.2–2.10分别覆盖T1–T14，无新增架构方案；本轮只文档。
- [x] 真实列表operation名、生成字段、时间编码和Makefile命令已核；未来新增tests均显式新；T1→T5依赖修正已落实，未制造临时响应兼容。
- [x] Review Focus五条均有归属task和具体反例；I2九例/M3五例与B-L1…6精确反例均已写入T3/T4/T6/T12；这只是作者文档自查，独立复审结论另见review，不能当产品验证。
- [x] T5类型变化消费者、T5旧calendar分页参数、T10 archived SWR、T11组件预计算都同Task处理；P11/后段有界旧地址逐条留账。
- [x] 计划不提交整套代码体；函数签名在spec统一，测试给具体输入/断言与命令；F和deep-review不是本轮已完成。
- [x] 本轮准确 main CI 全绿、归档清理和依赖漂移已复核，静态/版本桥见准备收据；独立执行者检查无待回答问题。文档通过不勾任何产品 Task。
