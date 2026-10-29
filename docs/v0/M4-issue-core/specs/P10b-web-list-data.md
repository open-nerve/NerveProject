# M4/P10b 列表数据层规格

> 状态：T1–T9既定任务门禁已完成。T10部分实现6b406e56通过202项及静态检查，但2e4ca695保留了“失权后恢复、旧回答仍写入”的真实红测；负责人已批准[P10B-S1](../reviews/P10b-permission-read-decision.md)方案A，先同步合同再恢复T10。红测尚未修复、T10未完成，T11–T14未开始。最近完整F及准确CI仍仅绑定T8的481c30b4。阶段最终门禁、整审与最终CI尚未完成，全部产品验收保持未勾。

## 1. 目标

将项目、已归档、个人主页的工作项列表迁到生成客户端；在既有 `Reconciled` 上扩充分组分页，并让确认的修改写入该会话所有列表和工作项的唯一副本。项目页挂载不再发旧 Plane 地址。完成 M4-design §12 P10b 的 14 个任务，保持 §3.22 的分页、重放、版本与会话约定。

执行前盘点基线 main `a06e3f70547b755b03bf8e6b33caafd52725d0f6`，树 `6eb00a9b8238d3dcbbb9ee9cebfdfe10020e4c66`，与 P10a 最终 `d1a19e78fa3ba83b289fdfc0f2788e3105d658da` 完全相同。准确 main CI 和归档清理事实由 [P10a review §10](../reviews/P10a-web-issue-types-review.md#10-授权合并main-ci-与归档清理2026-10-29) 绑定。2026-10-28 的 P9 基线准备稿保留为历史，当前依实际 P10a 树修订；执行前预检没有编写 P10b 实现、原型或测试；之后已按授权实施 T1–T6，下列尚未交付的接口仍是目标；T5/W1 证据见阶段 review §8，T6/R17 与当前接续见 §9。

权威输入：[M4 设计](../M4-design.md) §3.16–3.17、§3.22、§7.1–7.3、§7.7–7.9、§9.5–9.6、§12 P10b、§13；[推进方式](../../claude_to_gpt/02-method.md)；[硬规则](../../claude_to_gpt/03-rules-and-pitfalls.md)。P1–P9交接与P10a准备之间的逐项映射由预检review归档；P5列表/偏好合同和P8个人主页/搜索边界不重新裁定。

## 2. 交付物

### 2.1 文件职责与14项映射

`C/` = `web/apps/web/core/`，`P/` = `web/packages/`。新文件均明确标注“新”。任务按编号顺序推进，每个完成时应用与受影响包可编译，不留跨Task半接通接口。

| Task | 文件 / 职责 | 测试位置（新文件在实施时创建） |
|---|---|---|
| T1 | 新 C/services/issue/issue-list.service.ts、issue-api.service.ts及直接测试；核现有project-preferences薄服务/生成类型；旧列表容器与服务暂留至T5消费闭合 | 新 C/services/issue/issue-list.service.test.ts、issue-api.service.test.ts；既有P/api-client/test/client.test.mjs |
| T2 | C/lib/reconciled.ts：fetchPart及原应答重放上下文 | 新 C/lib/reconciled.test.ts；既有project/workspace/member/state/label stores回归 |
| T3 | 新 C/lib/paged-groups.ts：组/格/行/页模型；新 C/lib/issue-order.ts：排序和时间精度 | 新 C/lib/issue-order.test.ts、paged-groups.test.ts |
| T4 | C/lib/paged-groups.ts：确认与纯merge；新 issue-membership.ts：三态成员/组键；新 paged-groups.replay.ts：raw重放与verifyIds意图；新 paged-groups.evidence.ts：计数自己的来源、确认事实和共用三分支 | 新 C/lib/paged-groups.changes.test.ts、paged-groups.replay.test.ts |
| T5 | BaseIssuesStore选择门面、helpers/issue-list-binding.ts不可变scope/query与Reconciled及目录准备；三个列表切新service；root唯一写协调/退役/heldSources分发；issue-filters纯函数；use-issue-list-refresh.tsx接四布局可见刷新/拒绝/显式重试；同次删旧形状和服务 | 新 helpers/base-issues.store.test.ts、base-issues-utils.test.ts、lib/paged-groups.sources.test.ts、lib/issue-filters.test.ts、hooks/use-issue-list-refresh.test.tsx；迁移R8/R10与store-context |
| T6 | C/store/issue/issue.store.ts唯一副本、ReconciledByKey接收API，description/版本/墓碑、项目read epoch；根拥有 issue-verification.ts 的 round/去重/freshness 元数据；同Task闭合过渡详情/子项/关联/link/inbox 的 canonical 写入与只读消费合同 | 新 C/store/issue/issue.store.test.ts；既有详情、inbox、会话和复制弹窗回归 |
| T7 | C/store/issue/project/filter.store.ts；C/store/project/preferences.store.ts按需联动；删/user-properties两方法/例外/放行 | 新 project/filter.store.test.ts；既有 project/preferences.store.test.ts和三E2E故事 |
| T8 | 新 C/lib/issue-filters.ts；P/shared-state/types/utils富筛选否定层清除；hooks消费者 | 新 C/lib/issue-filters.test.ts |
| T9 | archived/profile浏览器持久化；删workspace-draft/filter.store及root/use-issues引用 | 新 archived/filter.store.test.ts、profile/filter.store.test.ts |
| T10 | issue/root.store计算属性、列表SessionSWR、三处同步读取、限制规则 | 既有 C/lib/store-context.test.ts/use-session-swr.test.ts；新 root.store.test.ts |
| T11 | kanban/calendar stores和布局拖放入口排队 | 新 C/store/issue/issue-drag.test.ts |
| T12 | 扩充T5/T6已接通的根store确认分发矩阵，全部已取列表+canonical及竞态 | 新 C/store/issue/issue-changes.test.ts |
| T13 | 项目/归档layout root、profile-issues挂载，S2 REQUESTS，新守卫、清理 | S2与三旧放行故事；新 C/components/issues/issue-layouts/list-data.test.tsx |
| T14 | 总体设计7.7、frontend-changes、交接处理、验收和deep-review | 证据原件+准确SHA完整门禁/CI，非新运行不得标绿 |

### 2.2 T1：生成接口与有界旧服务

消费 `operations` 中真实名称，不手抄完整查询类型：

```ts
type ProjectIssueQuery = NonNullable<operations["listIssues"]["parameters"]["query"]>;
type MemberIssueQuery = operations["listMemberIssues"]["parameters"]["query"];
class IssueListService {
  constructor(api: ApiClient);
  listProject(projectId: string, query: ProjectIssueQuery): Promise<IssueListPage>;
  listMember(slug: string, userId: string, query: MemberIssueQuery): Promise<IssueListPage>;
}
class IssueApiService {
  constructor(api: ApiClient);
  get(issueId: string): Promise<IssueDetail>;
  create(projectId: string, body: IssueCreate): Promise<IssueDetail>;
  update(issueId: string, body: IssueUpdate): Promise<IssueDetail>;
  delete(issueId: string): Promise<void>;
  archive(issueId: string): Promise<IssueDetail>;
  unarchive(issueId: string): Promise<IssueDetail>;
}
```

每方法仅一次生成客户端调用+unwrap。地址为 `/api/v0/projects/{project_id}/issues`、`/api/v0/workspaces/{slug}/members/{user_id}/issues`、`/api/v0/issues/{issue_id}` 及 `/archive`、`/unarchive`。Query沿已有api-client序列化：集合CSV、日期重复条件AND；sub_issue/archived是 `"true"|"false"` 的字符串，不发boolean。Member query必有relation，不夹项目专有state_id/assignee_id/sub_group_by/archived。

新文件名与旧 `issue.service.ts` 分开，旧类只保留P11尚未迁移的非列表消费者；`issue_relation.service.ts`、`workspace_draft.service.ts` 同样仍是 P11 的有界过渡。M5/M6/M7旧消费者迁到各自旧资源service，不挂新M4路径。`issue_archive.service.ts` 本段T5全体替换，当前仅 getArchivedIssues/archiveIssue/restoreIssue 三方法，没有旧详情 get；归档详情页头实际调用 IssueService.retrieve，仍归 P11，不增加 P13 页面功能。

TIssuesResponse/TGroupedIssues/TSubGroupedIssues/TGroupedIssueCount/TIssuePaginationData/TPaginationData与旧排序/分组/优先级枚举、两server映射在T5重写其base消费者时删除。T1只准备薄service并用直接测试消费，暂不从生产barrel新增无调用导出；不能T1先删类型、等T5修编译，也不为中间态造新旧response转换层。旧 `ProjectService` 本段只删get/updateProjectUserProperties；仅剩projectIssuesSearch的页面消费者由P12/P14消除，不能提前声称整个文件消失。

### 2.3 T2：Reconciled 的部分读取

保持既有 `fetch(read)`/`confirm(change)`调用兼容。目标：

```ts
type Change<V> = (value: V, answered?: V) => V;
fetchPart(part: string, read: () => Promise<V>,
          merge: (held: V, answer: V) => V): Promise<V | undefined>;
invalidate(): void; // Supersede all outstanding reads without clearing the held value.
```

`read` 在 service 应答后用同一纯decoder把页面转成带read元数据的 `PagedGroups`，所以整体与部分均返回V，不扩展通用库为Issue专用协议。重放固定调用 `change(merge后值, 原read返回值)`，整体fetch也传未重放的原V；直接confirm只传第一个参数。不把merge后的行当原应答证据。原有单参数Change自然兼容。

未取整体值时不调用read。整体fetch开始使旧part全部作废；同一part后发超越先发，不同part互不覆盖。merge以提交时的当前值为输入，只替换回答部分的total；再按确认先后重做期间change。整体fetch仍按其原应答重放所有期间change。SessionChanged或已被超越的失败返回undefined；当前其他拒绝交回，既有值不动。等待序号和期间change在finally按所有者清理，不能清掉后继。`invalidate()` 推进整体读代号并使所有在外part作废，保留当前value；它不发请求，stale由列表调用方确认写入。这用于成功树删除后的项目读失效（§2.6），旧应答即使完成也不能覆盖stale。

### 2.4 T3：组、行、全序和精度

```ts
type SortKey = string | number | bigint | null;
type Boundary = { key: SortKey; id: string };
type PagePart = { group: string; subgroup?: string };
type Row = Boundary & { updatedAt: string };
type Group = { total: number; rows: Row[]; nextCursor: string | null;
  boundary: Boundary | null; cursorBoundary: Boundary | null; stale: boolean;
  subgroups?: Map<string, Group> };
type PagedGroups = { total: number; groups: Map<string, Group>;
  subgroupTotals?: Map<string, number>; read?: PageReadMeta;
  counterEvidence?: Map<string, CounterEvidence> };
type ReadEvidence = { whole: boolean;
  parts: {part: PagePart; rows: Row[]; range: PageRange}[] };
type CounterEvidence = { sourceID: number; held: {rows: Row[]; complete: boolean};
  read?: ReadEvidence;
  confirmed: Map<string, {updatedAt: string|null; member: boolean}> };
type PageRange = { lower: Boundary | null; upper: Boundary | null; complete: boolean };
type CounterRef = { kind: "outer" } | { kind: "group"; group: string }
  | { kind: "cell"; group: string; subgroup: string }
  | { kind: "subgroup"; subgroup: string };
type PageReadMeta = { sourceID: number; part: { group: string; subgroup?: string } | null;
  ranges: Map<string, PageRange>; first: boolean; resetCounters: CounterRef[];
  rowParts: { group: string; subgroup?: string }[] };
type PageAnswer = PagedGroups & { read: PageReadMeta };
```

不分组用空字符串键；服务端未回答的组/格需要时按total0、rows[]、nextCursor null、boundary null建出。`boundary` 是显示窗口最远边界，只前进；`cursorBoundary`只记上一服务端页末行，与drop扩大的boundary独立。nextCursor完全采用响应，不解析、不生成。纯decoder `decodePage(page: IssueListPage, meta: PageReadMeta, context: ListContext): PageAnswer` 保留原响应行和本次range；PageAnswer仅含Row，不保留第二份完整Issue。普通merged value的read元数据不作为证据；首次改动前将 raw Row/range 惰性冻结到 CounterEvidence，重放只用 Change 第二参数的原值。生产 decoder 要求 sourceID，列表按回答分配；`heldCounterSources(value)` 只读取当前 held 模型的来源集合，确认通过显式参数传递本操作/本列表许可。原PageAnswer冻结本次范围：第一页lower为空表示负无穷；续页lower为发出时cursorBoundary；末页complete仅表示到正无穷，不能把lower之前当已覆盖。空续页保持cursorBoundary，已完没有下一请求。

`issue-order.ts` 产出 `issueNameKey(text): string`、`compareCodePoints(a,b): number`、`compareIssueTime(a,b): number`、`issueSortKey(issue,order,catalog): SortKey`、`compareRows(a,b,order): number`。Catalog通过生成State/Label/MemberUser目录解析名称/状态组，不向Issue补接口外字段。名字只折ASCII A–Z；逐Unicode码位比（不用localeCompare/UTF-16 `<`）。多值名始终取最小；行排序的优先级等级为none=0、low=1、medium=2、high=3、urgent=4，`priority`升序none在前、`-priority`降序urgent在前，与分组展示顺序urgent→none分开（M4设计§3.16、P5 spec排序契约）。空值双向最后，同键UUID随方向。时间键保留纳秒，日期按YYYY-MM-DD。计数key用BigInt：从生产Issue number输入仅接受Number.isSafeInteger且非负，再转BigInt；unsafe值交回错误，不假称能恢复JSON解析已经丢掉的精度。排序fixture含9007199254740993与9223372036854775807，测试须按Node24 JSON.parse reviver的context.source读原数字token转BigInt；其余数值按字段保留原类型。大整数fixture行按原token构造合法 `Row.key: bigint`，直接调用生产 `compareRows` 验证固定预期；不把bigint塞进生成 `Issue.sub_issues_count/link_count:number`，不使用强制类型断言。另用真实生成Issue的safe number调用 `issueSortKey` 验证转key，并单测unsafe number拒绝；名称、日期、优先级等能构成合法Issue的fixture仍覆盖 `issueSortKey`。不修改全局API客户端或HTTP协议。所有真实生产比较器由 `api/testdata/issue-order.json` 的独立预期测试，fixture不进入生产bundle。

实际HTTP由 `issues.go` 将 `time.Time` 放生成响应，`issues_test.go`已有 `.123456Z`；JavaScript毫秒不足。将RFC3339的无小数秒部分解析为UTC整秒，原小数右补9位成纳秒（允许0–9位），用BigInt比较；时区偏移先归UTC。`2026-10-17T10:00:00.123456Z < ...123457Z`，`.1Z == .100000000Z`，等值时后应答赢。只Date.parse整秒不是丢小数版本的比较。不能无效值默认为最新；不合法响应向调用方报告，不写缓存。

### 2.5 T4：确认与重放

`Before = {kind:"known"; issue:Issue} | {kind:"absent"} | {kind:"unknown"}`；`ConfirmedIssueChange={id,beforeSnapshot:Before,after:IssueDetail|Issue|null,drop?:{group,subgroup?,beforeId?,afterId?}}`。每个操作进入实际发送的队列轮次后、发请求之前，从canonical深拷贝冻结beforeSnapshot到该操作pending记录（数组也复制），新建为absent，从未见为unknown。不得在用户排队时过早取、也不得在确认时从已被read更新的canonical重取。pending同时保留这期间观察到的成员关系/排序版本证据；确认从同一操作取冻结before。与after同版本且相关字段相同的read可识别为本次写，不推翻before；若观察到更晚版本，或早于after但晚于before且改变相关字段的其它版本，冻结before不足以证明本次实际旧属，改为unknown并使受影响旧/新组stale，不能猜计数。只保留作此判定所需版本与字段，不建立第二个可写实体缓存。after=null专门表示**已确认删除**。纯函数 `confirmGroups(value, change, context, heldSources?: ReadonlySet<number>): GroupChangeResult`、`replayGroups(value, change, answer, context): GroupChangeResult` 产出 `{value, verifyIds:string[]}`，context包含query、comparison/catalog和pendingIDs，不在纯模型里请求。`ListContext` 的 member scope 同时要求 `workspaceSlug` 与 `workspaceID`；成员判断先比较 `Issue.workspace_id`，同一 user 的其它工作区不改该列表的行、计数或 stale。T5 从已有 Workspace 生成目录取得 ID 来创建绑定 scope，ID 未知时不创建可请求 scope，不新增 HTTP 协议。 同一纯模型提供 `mergePages(value: PagedGroups, answer: PageAnswer, context: ListContext): PagedGroups`，按原回答的计数/行写集合合并，T5 的 fetchPart 直接复用；它不改变 raw PageAnswer。`confirmGroups`/`mergePages` 由 `paged-groups.ts` 导出，`replayGroups` 由 `paged-groups.replay.ts` 直接导出，T5 分别导入复用；成员/组键纯判断在 `issue-membership.ts`，该文件对模型仅 type-only 引用，主模型不反向转发 replay，避免运行时循环。

| 旧属 | 新属 | total | rows |
|---|---|---|---|
| 是 | 是 | 不变 | 替换key/version并重排；超出固定window则移出；缺行但窗口内则插入 |
| 是 | 否 | -1 | 移出 |
| 否 | 是 | +1 | 窗口内插入，窗口外等待后页 |
| 否 | 否 | 不变 | 不动 |

直接确认的旧属由该计数自己的来源与本操作的 heldSources 决定：已确认事实优先；发送前实际持有的来源按原行/完整缺席/冻结 Before 判定；新来源只按 raw 三分支，缺许可不回退猜 Before。多标签/负责人多个组分别判，外层total按去重membership变化，泳道subgroupTotals按其独立membership变化，不能求各格和。

**计数自己的证据（T4 fix1/R12）：** 可选 `counterEvidence` 属于合并后的纯模型，decoder 的 raw PageAnswer 不写该字段。首次 confirm 或 partial merge 在改 rows 前，按稳定 JSON 数组 CounterRef key 为每个计数保存自己的 Row/完整缺席证据；包括新格带出的缺省零父组/泳道。已有来源沿用，不能用后续 merged.rows 反推旧共享计数。仅 resetCounters 的来源替换为本次原 Row/range 并清空该计数的确认事实；其它计数的来源与确认事实不动。whole 重取整体替换并清除此增量元数据。

直接确认将计数与行摆放分开：计数先读自身最新确认事实；来源在本操作 heldSources 中的计数按自己的旧 Row/完整缺席或冻结 Before 补差；其余计数使用与 raw replay 共用的 before/after/unknown 判断。来源 ID 随原回答保存在 PageReadMeta，按需进入 CounterEvidence；whole 清旧证据后由新原回答惰性建立来源，partial 只替换 resetCounters。范围不覆盖且没有行时，即使 Before 已知也不猜该服务器 total 是否已包含写，只标受影响部分 stale。确知补差或确知已经包含结果后，每计数每 id 只保存最新 `{updatedAt|null, member}` 一条事实，不保留历史链；unknown 不记录已应用。删除的 null 是计数已处理删除的证据，不构造第二份 Issue 或 canonical 墓碑。行摆放继续读当前 rows/window。缺行推断的 verifyIds 在同次结果内去重。replay 只同步此次拥有的计数事实和行；全部操作不改输入 value 或 raw Answer。`paged-groups.evidence.ts` 集中这些纯判定，仅 type-only 引用模型，主模型和 replay 单向消费它。

**部分回答的写集合：** `resetCounters` 列出本次响应被merge重置的计数，`rowParts` 列出本次回答的组/格。whole回答拥有outer、其group/cell及返回的subgroupTotals（未返回空组按0）；单组flat回答只拥有该group.total，单格flat回答只拥有该cell.total，其flat total绝非outer。merge与replay都只改这些回答部分：replay只为被重置的计数补确认差量，未回答的group、outer、subgroupTotals和rows保持当前值；已有确认早已改过它们。不同part同时在外时各自重放只改各自集合，不再次更动共享计数。whole可按完整原应答覆盖范围重放。

重放只用PageAnswer原响应：有行比版本；没行而范围覆盖新位置是之前，覆盖旧位置是之后；否则不知道。之后只摆放，之前依原应答+beforeSnapshot计数，不知道只对旧新关系不明的组stale。续页只判断它回答的组。第2种缺行推断返回verifyIds，store每id至多本轮追加一次 `getIssue`；核实中刚补回的id带“待核实”元数据，不能作为可靠可见值。核实成功与拒绝的收敛见§2.6四路表，不能只把错误交回却把推测行和计数永久当真。

删除无after.updated_at：原响应含该id则是删除之前，移行并减该响应所证计数；完整或覆盖beforeKey且缺id则之后不重复减；否则stale。本会话成功delete的id墓碑防晚到read/list复活；拒绝、SessionChanged、普通404都不能创建墓碑；archive/unarchive可逆且不用墓碑。树删除只给缓存中已知后代墓碑；未知后代不猜id/精确减数。成功204先推进该项目read epoch并使受影响列表的在外读取失效，再写已知墓碑及stale，随后才可发新epoch的刷新（§2.6）。因此未显示列表的旧whole/part、详情get和回查也不能把此前未知后代写回。其它独立项目列表不清。

Drop在目标格总显示，落点越界扩boundary，后页按id去重、按全序并入；不清已加载页/滚动。stale来自unknown-before、本地不可判断mention/subscriber/subscribed筛选、重放无法证明、读到第三方membership变化，以及已确认树删除导致旧读失效/缺行回查未证实这两种前述“无法证明”的具体情形。正常可见组立即、未显示组下次显示重取第一页；范围不可判断的整列表fetch；回查拒绝后的有界再验证按§2.6，不因stale形成同步自动循环。

### 2.6 T5/T6：列表协调与唯一工作项

**T5 首项 W1 / R13：** T4 补充审查实证，whole 已将窗口外新建 X 计入 outer10/B5、但未返回 X 时，现有 direct confirm 会依据 absent Before 猜成 outer11/B6 且保持 fresh；该行为在 fix1 前已存在。T5 必须在真实写协调链关闭此边界：区分操作实际发出前已持有的计数来源与操作在外时新到的原回答来源，后者按自身 raw 三分支判，未知不猜；前者保留冻结 Before 的意义，不能让普通离屏修改一律重取 whole。限定 issue 模块内部 pending/计数证据，不改 HTTP、通用 Reconciled 或会话协议，不保存第二份完整实体。内部形状已随 T5 同步本节与 §2.4/2.5；本段保留原问题和约束，关闭结论见阶段 review §8。必要测试包括 whole 窗口外 create、发出前已持有来源、partial/whole 两种到达顺序，以及真实 FakeNerve 读写交错；W1 未关闭前不推产品检查点。

R13 最小方案已在 T5 实施并通过独评：每个绑定列表为原回答分配可 structuredClone 的数字 sourceID，仅比较身份，不将其当 cursor、版本、请求序号或删除水位。每计数保留自身来源；实际发送前，从各列表当前 Reconciled.value 冻结已持有来源集合，绑定到本操作与不可变列表身份。确认时已有事实优先；来源在集合内才可使用发送前 Before 证据，否则只看该来源自己的 raw 三分支，未知不猜。不能用“已分配序号水位”代替实际 held 集合，因为 decode 到接受回答之间仍有微任务窗口。whole 替换旧来源和确认事实，partial 只替换自身计数来源；raw replay 不使用该许可。来源集合只存内部标识，不存第二份实体，不用全根可变 current-operation 槽隐式传递并发操作的证据；内部类型与 heldSources 参数已随 T5 实现同步。

BaseIssuesStore 是当前选择和读取门面；`helpers/issue-list-binding.ts` 的 IssueListBinding 每实例持一个不可变 scope/query 与 `Reconciled<PagedGroups>`，注册在本代 IssueRootStore；project/archived/profile只决定scope/query。门面保留曾取过的绑定，切选择不修改旧实例，根操作按实例冻结 heldSources，全部已取副本继续确认；无第二状态层，root/model依赖用 type-only。对外保留当前调用者需要的getIssueIds/getGroupIssueCount/getPaginationData等读取入口，内部不保留Plane分页兼容模型。无子组仅以 optional `undefined` 表示，看板 count/cursor/loader/next 四个消费口使用同一参数；不再传字符串占位 `"null"`。共享未分组键为 `""`，None 列/泳道键为 `"none"`；列表/看板快建预填不将 None 标签或负责人键写入 ID 数组，合法 `priority:"none"` 仍保留；普通及递归子项识别同一空字符串未分组键。接口迁移时修改使用方编译，不把新响应转换回旧TIssuesResponse。

`use-issue-list-refresh.tsx` 在四种现有布局按本次绑定实例和可见/展开的组或格调用 refreshStale(parts)（R16）；同一 stale 周期只尝试一次，尝试仅在实际启动读取时登记；绑定级错误可以暂缓其它 part，但不得消耗尚未发送部分的尝试，显式重试成功清错后须补发当前可见且从未尝试的 stale 部分。失败 part 不因 rerender 自动循环。拒绝保留 stale，经 followInSession/useRefusalToast 显示原因、局部按钮显式重试，rerender 不重试。绑定按 part 保留失败请求的原 cursor/lower/固定 query；retryFailed 重发该请求，首读无 value 时仍 fetchFirst；仅页面 fire-and-forget 边界消费拒绝，store promise 继续拒绝。未显示部分展开/挂载后才取；切绑定/卸载/退役停止旧 UI 跟进，不以根 currentList 反推可见性。已成功 issue 写的后续目录拒绝只使对应列表 markUnavailable/error/stale 并失效其在外页；canonical 仍接收完整成功应答，业务写 promise 成功，不引导重复提交。列表的下一次显式重取补齐目录后收敛，SessionChanged 仍丢弃整条旧流程。

名称排序所需目录按绑定 query 和不可变 scope 准备（R14）：state_name/state_group 使用现有项目 state fetch/get；label_name 对本页每个项目用项目 label fetch/get，不请求不存在的 workspace label 端点；assignee_name 使用 workspace memberships 的公共 profile，保留 inactive 历史负责人。目录未到或缺任一非空引用时至多一轮权威重取，仍缺或拒绝则交回，不把未知名称当 NULL；多值名字全部补齐后才取最小。读须先核 active/scope、补目录、完整 decode/校验所有行，再交现有 canonical 完整应答入口。确认 before/after 的离屏成员/排序字段也按相同规则补目录；若发送前等待目录，body、Before 和 heldSources 仍在真正发 issue 写前重新冻结。

T5为新的list请求首次加入§2.7四个query/表达式纯函数及基本请求测试，消费现有UI表达式与显示设置，不改变未迁移偏好的持久化来源；T7切偏好service，T8完成整个否定层清除和各条件/回读全面测试。查询的语义翻译是批准的业务功能，不是旧响应兼容层。

新的公开协作入口：`fetchFirst(): Promise<PagedGroups|undefined>`、`fetchNext(group,subgroup?): Promise<PagedGroups|undefined>`、`refreshStale(parts?: readonly PagePart[]): Promise<void>`、`acceptConfirmed(change, heldSources: ReadonlySet<number>): void`、`markMembershipStale(before,after): void`。实际页面调用参数由绑定的scope对象固定，变scope创建/选择另一个key，不篡改已有列表身份。原getFilteredWorkItems改吃Issue，保留子工作项相对日期fromnow行为。

canonical为 `ReconciledByKey<Issue|IssueDetail|null>`，null仅成功delete墓碑。补通用 `receive(key, answer, choose): V` 入口：以choose(held,answer)同步合并，新key可初始化，计应答序；已有在外read重放同一merge以免后到旧应答倒退，不另建parallel cache。IssueStore的choose是服务端updated_at比较，并应用description规则和墓碑。IssueStore公开 `acceptRead(answer: Issue|IssueDetail, token: IssueReadToken): void`、`getIssueById(id): Issue|IssueDetail|undefined`、`getIssueDetailById(id): IssueDetail|undefined`、`confirmChange(change): void`，对删除返回undefined。

较新Issue更新基本字段并保留已知description；较新/相等Detail替换描述；较旧Detail不写任何字段。若只有较新Issue而首次Detail已旧，不补旧描述，详情保持未取得并只补取一次最新Detail，继续竞态则保持可重试状态；无无限自动回路。没有description的旧响应不能把Detail降成Issue。读时membership/排序键变化给含它的列表旧新组stale；同代自己的in-flight修改ID暂不按第三方处理，确认后统一change分发。已失去项目/工作区可见性的canonical访问器不给旧值。

#### 项目读取代际与成功删除（B-L2）

`IssueReadToken = Readonly<{owner:symbol; serial:number}>`；每代IssueRootStore维护单调readSerial、每项目lastDeletionSerial及workspace下的最近项目删除serial。`captureRead(): IssueReadToken` 在**每个请求真正发出前独立执行 `++readSerial`**，取得唯一发出序号；它不只在删除时递增。删除水位与下面每id核实的freshness水位分别保存，不能混成一项。单项目list、详情/回查的canonical接收先核本代仍 active、token.owner 属于本代且 token.serial不早于响应project的lastDeletionSerial；多项目profile查询有显式project_id时按该集合、未指定时按workspace可见项目范围核删除serial，有任一相关项目删除即**整页丢弃并保持stale**，不得裁剪旧行后沿用旧total。普通单项目读不因别的项目删除而失效。

会话身份沿用 M2 的“一次会话一个 RootStore”，不从 ApiClient 反查 loginId，也不在旧回调中读取当前全局 loginId 给旧请求重新贴标签。T5 在 IssueRootStore 构造时生成不可变 owner（Symbol），由现有 RootStore.dispose() 调用其幂等 retire() 标记失效；无需修改 apiFor、TokenManager 或认证协议。每轮队列实际发送前、每次成功/失败结算和后续核实请求前均核 active；退役根的 captureRead 拒绝，已发请求的回答丢弃，不更新任何列表/canonical/核实水位。T6 的 acceptRead 同时核 owner，不能把旧根 token 交新根接收。SWR 仍用既有 loginId 键，owner 是内部生命周期令牌，二者不互换。必要反例：旧会话发出后收到 200（认证中间件不负责拦这种成功答）、退役后队列轮到发出、把旧 token 交新根、同账号重新登录得到新根，均不得写新旧缓存或追加请求；新根自己的读取正常。

成功树删除只在当前session的204确认后调用 `invalidateProjectReads(projectId)`：readSerial+1并记该项目/工作区水位；先对覆盖此项目的list调用Reconciled.invalidate（含未显示list），再保留stale及已知墓碑，最后发新token读取。canonical的acceptRead在任何receive/版本/description写入之前核令牌；get和verify也如此。list在接收完整Issue行、decode/merge、清stale之前均核令牌，旧read不能先写canonical再被列表层丢掉。新refresh期间若又发生一次成功删除，水位再次推进，夹在两次删除之间的read同样失效。拒绝、SessionChanged或普通404绝不推进删除水位；这些水位只属于当前RootStore，不写localStorage、不改服务端协议。

#### 旧详情消费者的同Task收口（B-L3）

T6同时改 `C/store/issue/issue-details/{issue,sub_issues,link,relation}.store.ts`；实际 P10a 树还包括 inbox/inbox-issue.store.ts 的 accept 写入；attachment.store 已删除，不恢复。完整写入清单和回归归属见附录 B。当前读完整实体的 `addIssue` 调用改为捕获read token后统一acceptRead；已有本段CRUD/归档操作只走根确认协调器。旧 `updateIssue(id, Partial<...>)`、直接 `set(issuesMap, …)`、计数±1/用子项数组长度等不能绕过canonical：对应旧操作成功后先标该对象待权威重验，再经 `refreshAuthoritative(issueId): Promise<IssueDetail|undefined>`（IssueApiService.get+token检查）取得真实count/updated_at；失败不本地加减，不伪造版本。只有partial/旧关联片段时按已有id走同一权威读取，不能把片段断言成生成Issue。

供旧读者的 `issuesMap` 是 `Readonly<Record<string, Issue|IssueDetail>>` 派生只读视图，不是第二份实体缓存；需要的旧addIssue/updateIssue写API彻底删除，不留无行为壳。这里只收紧共享存储边界，旧详情资源地址/业务流程完整替换仍由P11负责。若权威回读失败，采用下表未证实状态，不把“旧写成功”伪报为“最新实体已取得”。

两个详情入口在辅助资源（含 parent）await 返回后、标识写入和任何后续请求之前，必须再核本次读取的 owner/active/项目删除水位；子项读取在父项权威回读 await 后、发布子项 ID/分布或返回旧应答前，也必须再核原请求令牌。权威回读 fulfilled 但返回 undefined 可能表示旧流程已失效，不能据此略过令牌检查；失效读取按既有 SessionChanged 结束。

#### 缺行回查的四路状态转移（B-L6）

**T6 首项 R17：** T5 在 fetch/replay 后发出真实核实 GET，direct-confirm 当时仅登记 verifyIds。T6 由 IssueRootStore 单独拥有 `issue-verification.ts`，将两条入口收口到下述完整 round，跨副本与并发 part 去重；已成功的业务写不能因后续核实拒绝而伪报写失败或重发。核实错误保留于不可确认状态并交给其读取/显式重试通道。真实 direct-confirm→GET、拒绝后显式新轮次及退役后无跟进已由 T6 独评和 fix1 限定复审确认关闭，证据见阶段 review §9；后续全副本矩阵仍按 T12 验收。

核实元数据只存id、round、状态、error和 `freshnessFloor`，不复制实体。进入待核实时先记当前readSerial；回查404/其它拒绝实际落定时把该id的freshnessFloor提升到此刻当前readSerial（覆盖此前已经发出的所有read，包括核实在外期间新发的read）。它只是“恢复可见性必须有之后新发出的证据”的水位，不是删除水位，也不产生墓碑。`getIssueById/getIssueDetailById` 对待核实/不可确认可见的id不给可靠活值；列表展示也不把它作为普通确认卡片，相关stale计数不能表现为最终值。元数据不等于删除墓碑。只有同session、项目令牌有效、`token.serial > freshnessFloor` 且服务端版本规则允许的完整权威读，才能清该id核实状态；这证明请求在最近核实拒绝之后才发出。404之前已在外的旧list即使稍后回答、updated_at与旧canonical相同、项目删除令牌仍有效，也不能写回该id、恢复其可靠可见性或清相关组的stale。该freshness检查在canonical receive及清核实/清stale前执行；旧页即使缺此id，也不能据其旧范围解除本轮待重验状态。

| 回查结果 | canonical / 列表 | 后续请求与收敛 |
|---|---|---|
| 当前token且高于本id freshnessFloor的成功IssueDetail | 经版本规则接收；同after可确认本次推断，较新成员关系变化标对应组stale | 清本id核实状态；受影响范围至多安排一轮新token刷新，不回溯重复同一个get；序号不足的成功答按旧读丢弃 |
| 404 | 落定时提升本id freshnessFloor；对象不可确认为可见、访问器不供旧值；推测行不当活值、相关组/list保持stale；无永久墓碑 | 同一round最多一次受控可见范围刷新；空页可让组计数/rows收敛，但对象仍待以后有效权威读再恢复，不从“空页”推永久删除 |
| 其它拒绝（含403/500） | 落定时提升本id freshnessFloor，保留error与待重验状态，相关组/list仍stale，不确认推测行/计数 | 错误交回调用方；本round不自动再get或循环刷新，下一次显式重试/重新挂载开启新round，使用新token |
| SessionChanged / 令牌已因删除失效 | 丢弃此旧流程，不写新一代cache/错误/核实元数据，不清新一代stale | 当前session的正常挂载决定后续；旧round不安排请求 |

核实协调器只保存 id/round/状态/error/freshnessFloor 与在外请求、去重/预算，不保存第二份实体。绑定通过既有 useIssueListRefresh 登记可见范围；核实在外时 hook 不重复刷新，隐藏绑定保留 stale 到显示时，重新挂载可显式进入新核实轮次。成功核实后的列表刷新失败只保留该列表错误，不反向否定已经取得的权威实体。旧详情入口返回 canonical Detail 或 undefined，不再把被版本守卫拒绝的旧描述作为函数结果交给复制弹窗。

每个read/replay round维护已核实id集合及每scope刷新预算1，结束后不因同一stale自动重新开round；UI重试/下次挂载才建立新预算。并发part共享当前协调round对同id去重。仅高于该id freshnessFloor的新发有效list返回完整该id时才可重新证实可见并清核实元数据；新发的空页可使回答范围的rows/计数收敛，但不把该id永久删除。较旧版本、失效令牌或核实拒绝前已发出的list均不能清，即使服务器updated_at相同。404/拒绝不推进项目删除水位，不影响无关项目。

显式 retry/权威 refresh 通过内部 `begin(true)` 建立新 round：即使无关慢 whole/part 仍持有旧 round，也不能复用旧 round 中已落定的拒绝 Promise。真正仍在外的同 id 请求可以跨 round 共享，并登记到加入轮次的已核实 ID 集合；旧轮次结束仅释放自身，不能清除较新的当前轮次。自动 require/drain 仍使用原轮次去重，不因拒绝自动重开；不新增实体缓存或 HTTP 合同。

#### 权限范围读取失效（P10B-S1，T10）

同一登录中已观察失权再恢复，读前和读后的角色相同也不能令旧答有效。`IssueRootStore` 单一拥有权限观察、范围水位和生命周期：观察所属 workspace 目录 `workspaces` 与 project 目录经现有 `getProjectById`、权限 getter 给出的事实，覆盖已有绑定、持有实体及在途请求所涉及范围。目录权威应答被接收或确认的成员变化生效时观察语义签名；新范围先建立当前基线，未知/非成员不能发受限读取。若使用 MobX 观察器，它只跟踪权限，不复制 router/user/catalog；`retire()` 幂等释放。不能在 computed getter 偷写，不因对象引用或无变化目录重取推进水位。

工作区签名使用该 `Workspace.role` 和身份；项目签名同时保留已有 `getProjectRoleByWorkspaceSlugAndProjectId` 的有效读门控、所属工作区身份及原始 `Project.member_role`，原始项目角色为 guest 时再包含项目 `guest_view_all_features`。工作区 admin 兼项目 guest 仍按原始项目角色及开关受服务器 guest 可见性规则约束（M4设计3.4），不能因 getter 将有效角色折为 admin 而忽略该开关。已知可读签名变不可读或变为不同签名即失效原身份，覆盖 admin/member→guest 和原始 guest 开关变化；原始项目 member 的 guest 开关变化不影响其签名。恢复或改变后以新身份重取。当前 scope 是否允许读取仍遵循项目成员含 guest、归档及成员列表仅 admin/member；guest 行范围由服务端裁定，不在浏览器推断 own-only。

每个 HTTP 请求真正发出前以其不可变范围捕获本次权限证据，与既有 owner/readSerial 一同贯穿整个流程；已观察到的工作区失权使其项目/成员范围失效，项目失权或签名变化使该项目和覆盖它的成员范围失效，独立项目/工作区不受影响。单项目 list/detail/verify 核工作区及项目；成员列表带显式项目集合按该集合核，否则核工作区可覆盖的项目范围。任一相关变化使整页无效、保持待刷新，不得剪行后沿用旧 total。whole、part、detail、verify 在 canonical、rows、total、stale 及核实状态的任何写入之前核证据，包含每次目录或辅助资源 await 后及后续读取前。权限水位独立于成功树删除水位和每 id freshnessFloor，不改 owner/readSerial 含义、不造墓碑；拒绝、普通404不推进删除水位。

权限身份先于同一身份内的版本选择：新身份有效读可以替换旧身份已失效的持有值，即使旧值版本较高，也不继承旧描述；同身份仍按纳秒版本比较，等版本后答赢，成功删除墓碑仍不可复活。旧身份的非删除写已成功时，向调用方保留成功结果，但失效回答不得改写新身份已证实的实体、列表行或计数；在写答及目录等待后再次核权限，避免恢复后的有效列表出现空洞或被旧成员字段改写。成功删除仍走既有删除协调及墓碑规则。

成员列表应答中的项目若尚不在所属项目目录，使用现有 `ProjectStore.fetchProject` 按需取得该项目权限事实后才发布整页；403/404 等目录拒绝交回，仍无成员权限则整页拒绝，不裁行或沿用 total。带显式 `project_id` 的应答若包含集合外项目，即使该项目可见也按 scope mismatch 拒绝整页；合法应答中实际返回的项目也核权限水位。目录成功建立首次基线不自行作废本页；等待期间若观察到相关失权，原水位不可被新目录覆盖，旧页丢弃，恢复后新身份的 whole GET 收敛。该读取不新增端点或权限协议，也不无限重取。

失权时相关已持有实体和列表不得显示；恢复后也不得先暴露旧值，须由新身份下有效的完整 Issue/IssueDetail 读取及列表 whole 回答分别证实后才提供可靠值，完整列表行可作为实体的证据。保留现有协调状态和范围/水位元数据，不增第二缓存。恢复后的读取身份作为现有 sessionKey 的 scope 参数，确保新 GET 可在旧请求仍在外时发出；相同有效身份的多个消费者仍共享在途请求，普通卸载不取消共享绑定。核实协调的同 id 去重和重试也以此身份区分：失效旧轮不写核实元数据，恢复后的新轮不复用其旧 Promise。原 `useSessionSWR` 的 loginId、拒绝不重试和焦点规则保持封闭；局部 dedupingInterval=0 只处理已完成请求的短窗，不能作为 ABA 方案。

T10 必须直接验证失权→旧答、失权→恢复→旧答、新旧请求同时在外及新答收敛、共享消费者与普通卸载、无变化目录、其他范围、guest、whole/part/detail/verify、拒绝/loading 和既有会话/删除/核实水位，并做真实生产变异红及恢复；T12 仍执行全副本交错矩阵，T13 仍执行四角色完整请求集及 S2。M4 只收口现有读者，P11页面及其完整详情资源迁移不提前。

### 2.7 T7/T8：显示偏好和筛选

ProjectPreferencesService现有get/update签名不变；project filter按生成filters/display_filters/display_properties消费。每次PATCH只带要变的顶层字段的全值，其余字段省略；同字段排队时从最后确认值计算。navigation/sort_order不被覆盖。服务端不会自动使用偏好，列表显式拼查询。

T7 的页面跟进以已选列表记录的偏好查询部分与最新确认偏好比较，不能以调用排队前快照判断 no-op；high→low→high 连续成功保存后必须选回 high 列表。该记录只描述所选查询，不存实体；limit 与日历等额外查询参数不参与偏好比较。保存与读取拒绝分别处理，读取失败的按钮只重试 GET，不重复 PATCH。

项目页面的显示设置入口与富表达式入口通过页面层会话跟进：偏好 store 交回写入结果或拒绝，确认查询改变后才独立刷新原项目列表；刷新拒绝作为读失败保留并可重试，不重发保存。组件草稿和已确认查询分开，会话或项目切换后不跟进旧页面。通用筛选回调、归档和个人列表不改变合同。

`issue-filters.ts` 目标纯接口：`toProjectFilters(expression): ProjectIssueFilters`、`toFilterExpression(filters): TWorkItemFilterExpression`、`projectListQuery(filters, display): ProjectIssueQuery`、`memberListQuery(filters, display, relation): MemberIssueQuery`。标识字段映射state→state_id、labels→label_id、assignees→assignee_id、created_by→created_by_id；日期条件保持重复AND，不把两个条件随意合并成一个。one-level AND和每种已支持条件逐一测试；None/me按契约；不支持的持久化属性/组合在本地选择明确默认并清理，不偷偷传给服务端造成整页失败。 清理以字段为单位：一个字段出现任一非法值、错误 operator 或非日期字段的多个独立 AND 条件时，整字段回到未筛选默认，不部分保留或合并为 OR；其它合法字段保留。未知属性及非 one-level AND 结构不进入查询。日期重复条件保持顺序与 AND（包括重复值），单个 none 合法，与任何另一日期条件并存则清空该日期字段；标签/负责人仍允许 none 与合法 id 并列。空值/空表达式归一为 `{}`，回读同样清理，范围空端点必须经 UI adapter 往返保留。 日期年份只接受 0001–9999，与服务端 ParseDate 一致；已接受的 UUID 在表达式、保存回读和 project/member query 词汇边界归一为 canonical 小写，不能将合法大写 UUID 当坏值清掉。web 内部 `toMemberFilters(expression)` 供个人筛选回读和成员列表共用，提取 project_id 前检查根/各节点的 record 与 one-level AND 形状，项目转换仍不接受 project_id，坏结构不阻断其它合法条件发出请求。该默认会移除无法表达的字段限制，界面同时显示清理后的确认值。

ProjectDisplayFilters生成字段为string，query为enum；用允许值目录显式收窄，不用断言伪造安全。项目分组不带project_id，member允许更窄的词汇，无subgroups。表达式只用于UI state，持久化用ProjectIssueFilters。 subscriber_id 的表达式/偏好/查询往返在 T8 支持，并验证真实 FilterInstance 编辑其它字段及再次保存时保留它；现有 UI 没有 subscriber 创建入口，缺配置时仍可能呈现 InvalidFilterItem。本段不新增选择器，该呈现限制交 M4 Phase P13 的筛选 UI 核对（不是 P10b Task 13）。删除否定参数/恒等alias/hook，不留allowNegative:false壳。旧/user-properties两个方法、两例外、mounts.workItemFilters、p2-visibility/p8-project-preferences/s7-chinese-project三故事放行在T7一起消失。

T8 删除否定层时同步启用既定 negation-filters 精确静态守卫，以完成 M27 的词项回引反例；T13 继续核 M43 与挂载门禁。该前移不放宽守卫，不把普通数值或排序中的负数概念当作否定筛选 API。

### 2.8 T9/T10：浏览器偏好、会话与权限

已归档/profile在localStorage的issue_local_filters，用现有主体作用域隔离；归档richFilters/rich_filters读写统一成rich_filters，与profile一致，历史错误键读一次可迁移但不同时保留两份权威。存储失效/损坏恢复默认，不抛未处理错误；不把其它账户偏好混用。看板折叠kanban_filters保留。草稿无筛选：WorkspaceDraftIssuesFilter及root/useIssues/types/barrel引用删除，draft列表本身留P11。

IssueRootStore的原autorun副本改为rootStore/router/user目录计算属性；权限观察如需reaction仅按§2.6的独立职责及retire生命周期进行。写协调器的 retire 仍按§2.6接入既有 RootStore.dispose；空目录不能保留上一项目数据。列表侧project-layout-root、archived-issue-layout-root、profile-issues统一useSessionSWR；键名、loginId、权限身份和scope参数完整，null不请求，拒绝停止loading。访客个人主页不取issues，非成员项目页不取list；归档只admin/member。三处rootStore同步读继续只读，发请求经拥有者RootStore。组回查第三方版本也受同一会话及权限水位守卫。恢复后的新身份不得复用失效在途请求；局部 dedupingInterval 选项的边界见§2.6。

### 2.9 T5起接通、T11/T12完成：串行修改与全副本确认

每工作项的修改共用根store一条oneAtATime队列；为避免共享邻接列两个不同ID拖动冲突，拖放使用该根store统一的拖放队列，再在对应工作项队列里执行（固定取得顺序，任何其它操作不反向持有两队列）。请求到发出时才从canonical读取邻接IDs现值计算sort_order，邻居已不存在则按当前有效落点/首尾65535间隔重算，不发送旧平均数。无邻居使用65535；首尾/中间算法与现有布局/服务端sort_order契约一致。

目标入口 `updateIssue(id, makeBody: (held:Issue)=>IssueUpdate, drop?, operationScope?: IssueWriteScope): Promise<IssueDetail>`，拖放传makeBody闭包，不提前构造body；create/delete/archive/unarchive同根协调。body逐字段构造，禁止spread Issue带id/project_id。请求中仅UI pending/临时拖放呈现变化，canonical/PagedGroups不能变；拒绝交回且UI回原处。成功确认一次，从该操作pending取实际发出前冻结的beforeSnapshot并核对期间读证据→按已确认删除时先失效读代际→写各列表→写canonical，pending在对应请求finally结束，队列拒绝后继续。

每修改表覆盖全部已取project/archived/profile副本、未取副本、pending、失败、SessionChanged、晚到读取与重取。跟进toast/导航必须followInSession；本段改到的组件签名跟进归本段，完整详情/草稿新交互仍留后段。

### 2.10 T13/T14：挂载、门禁与文档

S2四角色加入项目issues页，按角色列真实请求；成功只放行新list/preferences，非成员不放行list。P10a删除的M6/M7页不恢复。旧服务路径守卫只缩小例外；issue-optimistic不写跨函数await位置的脆弱正则，使用明确范围的运行mutation表证明未确认不写；negation-filters词项全删后用静态规则。改到文件死成员/prop清理、警告上限随真实减少调低。

T14更新总体设计7.7与frontend-changes（真实路径`docs/v0/frontend-changes.md`）、spec验收、阶段review和deep-review；只在本阶段实现后打完成勾。M4总体、handoff总status仍open。

## 3. 与设计的补充裁定

| 编号 | 裁定 / 理由 / 错误代价 |
|---|---|
| D1 | P10a/P10b用既有M4/P10阶段值，准备不改；避免放宽只接受整数phase的守卫。错误会让到期例外失效。 |
| D2 | 新IssueApiService与旧P11过渡文件分开；薄生成client不混旧APIService。代价是临时两个明确命名资源入口，P11移除旧者。 |
| D3 | cursorBoundary与展示boundary分离，PageAnswer冻结续页范围；不能从opaque cursor逆推，也不把末页当全组。多一个局部字段换取正确replay。 |
| D4 | 时间比较保留纳秒；真等值才按后应答。使用仓库JS BigInt，不新依赖；数毫秒精度错误会让缓存倒退。 |
| D5 | 删除204专门重放、同代确认墓碑和未知后代局部stale；避免对null比较版本及旧列表复活。部分删除会多一次本项目重取。 |
| D6 | 较旧Detail不能首次补描述；最多一次追加读取、失败回调用者。可能多一次请求，防止旧HTML配新对象。 |
| D7 | operationquery纯函数放应用lib，shared-state只保留表达式；避免新增包依赖及复制生成类型。 |
| D8 | 归档localStorage键修正按§12在P10b T9落地（§3.17旧文字说P13，后者页面故事再验）；M4设计同次同步此落点。 |
| D9 | issue-optimistic采用真实修改表+变异作为门禁，按§7.9允许的静态不可靠退路，不声称简单正则能证明异步写序。 |
| D10 | root统一拖放队列+每ID队列，固定顺序、无反向等待；满足不同卡片共享邻接位依赖，不新增外部协调器。 |
| D12 | 六项预检细化：发出前冻结before、树删除失效项目读epoch、T6旧canonical写者同任务收口、partial只重放被重置计数、大整数fixture与生成Issue合法入口分开、回查四路有界收敛。均为批准模型内的缓存正确性；代价是少量pending/令牌/核实元数据与必要权威读取，不改变模块接口/HTTP协议或新增依赖。 |
| D11 | §12粗任务落到可编译提交：T1准备有测试消费的薄服务，T5基类和消费者同时删除旧形状、接线，并前移所需query纯函数；T8仍负责全部否定层/语义矩阵。避免跨Task坏编译和临时response转换层，代价是T5较大（约1100–1500行，独评检查拆分职责）。 |

均是已批准设计内的局部精化；如实施发现需要改变模块边界/跨模块协议，先提交已完成工作再上报。

## 4. 验收标准

- [ ] A1 全部14tasks实施及逐Task独评完成，无未决Important/Critical；核心类型承接P10a，列表旧类型/旧游标/乐观回滚移除。
- [ ] A2 list services和preferences消费真实operation；query/body精确，拒绝及SessionChanged不写；旧列表服务和/user-properties放行消失。
- [ ] A3 fetchPart全部序列、原应答replay、partial计数写集合、组/格/窗口/多值/总数矩阵通过；I2九例、M3五例、续页范围和删除204附加例逐条有证据；纯模型在T4，canonical/真实回查在T6/T12，精确映射见plan的T4反例索引。
- [ ] A4 真实比较器读取共用fixture测试；Unicode码位/ASCII折叠/多值最小/NULL/id方向/微秒时间精度正确。
- [ ] A5 canonical仅一份，T6过渡消费者无直接或partial绕写；版本/description、发出前before证据、第三方stale、项目删除读epoch/未知后代、墓碑、回查四路有界收敛均通过。
- [ ] A6 筛选每种条件可保存与还原、query显式拼装；否定层全删；fromnow子工作项筛选保持。
- [ ] A7 归档/profile偏好刷新恢复且作用域隔离，草稿无filters；root无autorun严格警告。
- [ ] A8 串行修改延迟计算sort_order且每副本只写确认结果，删除未知后代只重取相关项目。
- [ ] A9 S2四角色项目挂载与三旧放行故事通过，无未放行旧请求/拒绝/控制台异常。
- [ ] A10 真实七门禁/准确分支HEAD CI全绿、deep-review闭环；阶段结束先不合并。授权集成后准确main CI另验。

## 5. 不在范围与后续

P11：详情/关联/评论/表情/动态/版本/草稿/搜索services与stores全迁移，最后旧IssueService等删除；本段只是已存在消费方的签名闭合。P12：列表/看板/日历/表格完整新交互与C1/C3；P13：筛选/归档/profile页面、后退；P14：详情/父子选择器、I22/会话切换与编辑器；P15：协作/草稿/命令搜索；P16：全范围死成员/全仓no-shadow/最终关键词；M4收尾C12三浏览器。

M6/M7列表不加回，不实现祖先搜索或工作区视图。P9后端删除/清理合同保持，不以之前local/CI绿冒充本段行为验证。

## 6. 风险

最高风险为重放范围与计数、server版本精度、canonical description、不同scope/会话混写、不同卡片共享邻接排序。已分别绑定T2–T6/T10–T12测试与变异。共享Reconciled是通用设施，改变接口必须保留原M2/M3所有消费者语义并跑其真实回归。14任务，T5约1100–1500行，其余约250–1100行/任务，paged-groups设计允许约450行纯模型；超出职责再按order/model分拆，不为指标压缩测试。计划与测试预计接近前端改写规模，发现任务不可编译的跨段循环应重切局部任务并记录，架构变化才上报。

## 7. 已知限制、移交与关闭条件

执行前预检没有 P10b 实现/原型/行为测试；实际 P10a 合并树的路径、写入方和 guard 盘点见附录 B。实施起点已复核，当前 T1–T4 既定任务门禁已完成；W1 是 T5 必修的已知 Important，完整阶段与后续任务仍未验收。输入handoff的项目只在实施且验证后追加处理结果，总status保持open至M4收尾。服务端列表没有每行排序key，前端必须基于当前目录按相同合同形成key；目录异步拒绝不能转成静默空列表。

关闭条件：本规格§4全通过、deep-review无未决项，按负责人另行授权--no-ff集成、准确main CI绿、证据归档哈希与清理完成。当前不满足这些完成条件。

## 附录 A：准备记录与证据边界

2026-10-28已只读核实设计§3.17/3.22/7/9.5/12、实际schema/HTTP时间字段、列表store/service/挂载、Makefile/package/test设施及关键词解析。盘点原件在仓库根 `.superpowers/sdd/P10-preparation-20261028/code-inventory.md`，与P9旧worktree生命周期无关。未执行生成、安装、原型、产品测试或变异。仅运行Node24语法能力probe：实际v24.15.0的JSON.parse reviver对上述两个大整数返回原token，转BigInt后exact=true；这只证明fixture无损读取方式，不是产品比较器通过。所有产品命令和目标测试是未来实施步骤。本轮首审提出六项Local文档问题，本修订逐项补齐规则/反例，原审者已确认六项文档层关闭，见[review](../reviews/P10b-web-list-data-review.md)；不把作者自查当独立结论，也不把文档通过当产品验证。P10A-S1已获负责人A裁定，验收归属同步后的定向复审见P10a review；当前仍不实施。这段记录属于首次依赖预检；2026-10-29 执行前复核另见附录 B 和阶段 review。实施仍须负责人授权。


## 附录 B：P10a 集成后复核（2026-10-29）

本轮只复核实际依赖并修订文档。历史六项模型修订不重开；P10a 产品、测试与 CI 的通过不证明未来 P10b 的行为。准备证据根为主目录 `.superpowers/sdd/P10b-preparation-20261029/`；原 P10 准备/裁定原件与 P10a 产品证据的稳定归档入口见 P10a review §10。

### B.1 实际文件与移交

- 预检基线已有 `C/store/issue/fake-issues.ts` 的 `issueOf` 和 1,262 行旧 `BaseIssuesStore`，以及 `Reconciled`/`oneAtATime`、生成 Issue/IssueDetail/IssueUpdate 合同。T1 已扩展既有夹具；T1–T6 已交付薄服务、fetchPart/receive、PagedGroups、绑定列表与唯一实体，没有另建近似客户端或复活旧引擎。
- `IssueArchiveService` 当前三方法在 T5 全部替换并删除文件/barrel；`IssueService` 的 create/getIssuesFromServer/getIssues/patch/delete 在 T5 移除；retrieve（包括 parent 和归档详情页头）、编号读取、子项、普通链接等仍留 P11。不得把所有剩余方法期限一律推迟。
- `UserService.getUserProfileIssues` 仅被 ProfileIssues 的首/续页调用。T5 改为 listMember 后，同任务删除该方法、TIssuesResponse import、无其它旧方法时的 APIService 继承及 super；保留 M2 的认证资料方法和 ApiClient。设计原 P11 条目同步为接收本段已完成事实，不能留下无使用者旧接口到 P11。
- 预检基线的 canonical 外部写者是 helpers/base-issues、archived/issue、issue-details 的 issue/sub_issues/link/relation、inbox/inbox-issue 共 7 个文件。T5 已将前两处收口至 root.store，T6 按实际树处理 root.store、4 个详情 store 和 inbox 共 6 个外部写者，以及 helpers/issue-list-binding 经 root.acceptListIssues 的间接读入；另有 canonical 自身。workspace-draft 的同名 issuesMap 是草稿缓存，不当普通工作项 map 盲删。详情父项读取必须在发出前单独 captureRead，不能沿用子项请求令牌。
- T6 同步把 useIssues 的 issueMap 类型和其它读取合同改为只读；不止 map 外壳 readonly，生产读取方也不得通过返回的实体/数组修改 canonical。共享对象的防绕写由类型与实际调用枚举证明，不以全局禁用 MobX 守卫代替。
- IssueRootStore 的既有 autorun 严格模式警告归 T10；M6 归档 loader / CycleQuickActions 的既有问题及未实现领域 API 仍归 M6。M6/M7 自身页面验收归属 A 不变，既不能恢复旧列表，也不能把全域失败过滤掉。

### B.2 逐任务闭合与现有性质

T5 在接上新服务时同时落地**唯一根写入口和列表注册/确认分发的基本路径**（§2.9 的签名），复用 T4 的纯模型；T6 才把 canonical 换为 ReconciledByKey 并收口全部旧写者。T11 将拖放的邻居计算延后到该入口的队列轮次；T12补全每修改/每副本/竞态矩阵与未闭合边界，不能等 T12 才首次提供 T5/T6 调用的入口。T5 中已有 canonical 完整应答入口只接真实响应，T6 同提交改成 confirmChange/acceptRead；不临时复制一份可写缓存、不新增旧响应转换层或 local-only 绕写接口。T5 的基本拒绝/排队/确认只写一次及退役会话测试必须当场通过；Task 仍为 14 项，若实作单任务超限先在同阶段重切并记账。

新接口接通时请求体改用 IssueCreate/IssueUpdate。普通详情 main-content 与 peek-overview 的 `skip_activity` 写者在 **T5 同任务**改成 boolean；仅 description_html 独立保存允许附此字段，不和 sort_order/其它字段混发，不做 string→boolean 兼容转换。旧 M7 inbox 的 string 输入独立保留至 M7；共享旧请求类型不得因为 M4 接通而提前删坏 M7。创建显式选择 IssueCreate 字段、要求真实 name，合同不允许的 null 父项以未指定表达，没有 id/project_id/audit 的 spread 或伪造实体默认值。

| 性质 / 现有测试 | 必须承接的任务与证据 |
|---|---|
| R8：quick-add/root.pending.test.tsx；project/issue.quick-create.test.ts | T5、T12：本入口 pending/ref 的请求归属、scope key、失败/undefined 保留输入；旧 A 回答不清 B 输入、不插 B 列表；仅成功真实应答写 canonical/组/计数一次。用新 HTTP 边界改造测试，不删性质 |
| R10：helpers/base-issues.update-request.test.ts；inbox/inbox-issue.request.test.ts | T5 改 Base 测试为新 boolean wire 与等待期间实体/计数不动、拒绝保持 HTML 原值/空串/不存在；旧乐观回滚/local-only 断言按新合同替换。M7 独立旧输入不扩大修改；T6 accept 不用 partial editor 覆盖较新 canonical，空 HTML 的真实应答优先 |
| R15/R16：issue-details/issue.fetch.test.ts；issue.store.types.test.ts | T5/T6 迁移测试的列表应答形状与接收入口，保留两详情路径 canonical 一次、反应/链接/非空 parent 的 await/拒绝责任；纯列表缺关联字段不清 relation cache，缺描述不降级，空描述和 nullable actor/parent 真实保留 |
| A-01：issue-details/sub_issues.fetch.test.ts | T6 按权威回读更新 count 断言和 HTTP 请求集，保持跨项目仅 state/member/label 目录去重、不再预取 cycle/module；子项旧资源 URL 直到 P11 接通才改，不能为 P10b 省测试而删旧边界 |

T6 过渡操作的权威回读失败必须交回并保留未证实状态；例如 inbox accept 的旧 catch 不能吞掉新 canonical 重验拒绝而报告已获取真实实体。只适配该共享缓存边界，不扩为 M7 资源重写。R15 已有宿主错误/重试与迟答隔离继续回归，尚存其它 detached 详情请求由 P11 统一处理。

### B.3 关键词期限与静态验收

实际 phase=M4/P10，73 条规则、53 个例外。T1 不写 P10b 字母值，不为准备修改 guard。旧工作项规则共有 41 命中/29 exact key（37 产品 + 3 Base 测试 + 1 子项测试）；数字只绑定本基线，实施按真实命中重数。

- IssueService 的 create/list 共用 issues/ key：count=2，until=M4/P11；T5 删除。patch/retrieve 共用 issues/${issueId}/ key：count=2，同样 until=M4/P11；T5 去 patch 后只将实际剩余 retrieve 重数为1、归 P11/期限 M4/P12。delete 的 issues/${issuesId}/ key 随 T5 删除。
- Base 请求测试的 3 处旧地址随 T5 迁移；sub_issues.fetch.test 的 1 处地址属于 P11 数据层，until=M4/P12。不能把普通 issue-links 当 M5 或把数据层期限推到 P14/P15。
- user-properties 两处产品命中和例外/三故事放行在 T7 一起删；/search/ 另有人工清单，P15/P16 到期，不声称自动守卫覆盖它。
- 准备提交仅文档；完整七门禁/产品变异仍留实施。每次产品检查点前严格串行 F，最终 deep-review 后不自动合并。
