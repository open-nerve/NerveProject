# M4/P10b 列表数据层规格

> 状态：准备稿；六项局部问题已独立复审关闭，有条件通过静态依赖预检，尚未实施；P10A-S1 已获负责人A裁定，文档定向复审见P10a review。P10b 必须在 P10a 合并且准确 main CI 通过之后再记录执行基线、复核漂移；本文不是已完成的产品验收。

## 1. 目标

将项目、已归档、个人主页的工作项列表迁到生成客户端；在既有 `Reconciled` 上扩充分组分页，并让确认的修改写入该会话所有列表和工作项的唯一副本。项目页挂载不再发旧 Plane 地址。完成 M4-design §12 P10b 的 14 个任务，保持 §3.22 的分页、重放、版本与会话约定。

准备盘点基线 main `4599a9cfc5be90b2c242977f194157778a5f6ec9`，其产品树与 P9 最终 `369b39a09d035031dd15a1ccf18cb2b49ea33f4e` 相同。P10a 产品尚不存在，本轮不编写它或 P10b 的实现、原型、测试。本规格的目标接口用于后续实现，不代表现有导出。

权威输入：[M4 设计](../M4-design.md) §3.16–3.17、§3.22、§7.1–7.3、§7.7–7.9、§9.5–9.6、§12 P10b、§13；[推进方式](../../claude_to_gpt/02-method.md)；[硬规则](../../claude_to_gpt/03-rules-and-pitfalls.md)。P1–P9交接与P10a准备之间的逐项映射由预检review归档；P5列表/偏好合同和P8个人主页/搜索边界不重新裁定。

## 2. 交付物

### 2.1 文件职责与14项映射

`C/` = `web/apps/web/core/`，`P/` = `web/packages/`。新文件均明确标注“新”。任务按编号顺序推进，每个完成时应用与受影响包可编译，不留跨Task半接通接口。

| Task | 文件 / 职责 | 测试位置（新文件在实施时创建） |
|---|---|---|
| T1 | 新 C/services/issue/issue-list.service.ts、issue-api.service.ts及直接测试；核现有project-preferences薄服务/生成类型；旧列表容器与服务暂留至T5消费闭合 | 新 C/services/issue/issue-list.service.test.ts、issue-api.service.test.ts；既有P/api-client/test/client.test.mjs |
| T2 | C/lib/reconciled.ts：fetchPart及原应答重放上下文 | 新 C/lib/reconciled.test.ts；既有project/workspace/member/state/label stores回归 |
| T3 | 新 C/lib/paged-groups.ts：组/格/行/页模型；新 C/lib/issue-order.ts：排序和时间精度 | 新 C/lib/issue-order.test.ts、paged-groups.test.ts |
| T4 | paged-groups确认、重放、stale/getIssue意图 | 新 C/lib/paged-groups.changes.test.ts、paged-groups.replay.test.ts |
| T5 | C/store/issue/helpers/base-issues.store.ts及utils/filter-helper；project/archived/profile列表；正式切新service、删除旧容器/枚举/归档服务，首次加入所需query纯函数 | 新 helpers/base-issues.store.test.ts、base-issues-utils.test.ts |
| T6 | C/store/issue/issue.store.ts唯一副本、ReconciledByKey接收API，description/版本/墓碑、项目read epoch；同Task闭合P11过渡详情/子项/关联/link的canonical写入 | 新 C/store/issue/issue.store.test.ts |
| T7 | C/store/issue/project/filter.store.ts；C/store/project/preferences.store.ts按需联动；删/user-properties两方法/例外/放行 | 新 project/filter.store.test.ts；既有 project/preferences.store.test.ts和三E2E故事 |
| T8 | 新 C/lib/issue-filters.ts；P/shared-state/types/utils富筛选否定层清除；hooks消费者 | 新 C/lib/issue-filters.test.ts |
| T9 | archived/profile浏览器持久化；删workspace-draft/filter.store及root/use-issues引用 | 新 archived/filter.store.test.ts、profile/filter.store.test.ts |
| T10 | issue/root.store计算属性、列表SessionSWR、三处同步读取、限制规则 | 既有 C/lib/store-context.test.ts/use-session-swr.test.ts；新 root.store.test.ts |
| T11 | kanban/calendar stores和布局拖放入口排队 | 新 C/store/issue/issue-drag.test.ts |
| T12 | 根store唯一确认分发，全部已取列表+canonical | 新 C/store/issue/issue-changes.test.ts |
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

新文件名与旧 `issue.service.ts` 分开，旧类只保留P11尚未迁移的非列表消费者；`issue_relation.service.ts`、`workspace_draft.service.ts` 同样仍是 P11 的有界过渡。M5/M6/M7旧消费者迁到各自旧资源service，不挂新M4路径。`issue_archive.service.ts` 本段T5全体替换，其旧详情get也交 `IssueApiService.get`，改调用者签名使可编译，不增加P13页面功能。

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
type Row = Boundary & { updatedAt: string };
type Group = { total: number; rows: Row[]; nextCursor: string | null;
  boundary: Boundary | null; cursorBoundary: Boundary | null; stale: boolean;
  subgroups?: Map<string, Group> };
type PagedGroups = { total: number; groups: Map<string, Group>;
  subgroupTotals?: Map<string, number>; read?: PageReadMeta };
type PageRange = { lower: Boundary | null; upper: Boundary | null; complete: boolean };
type CounterRef = { kind: "outer" } | { kind: "group"; group: string }
  | { kind: "cell"; group: string; subgroup: string }
  | { kind: "subgroup"; subgroup: string };
type PageReadMeta = { part: { group: string; subgroup?: string } | null;
  ranges: Map<string, PageRange>; first: boolean; resetCounters: CounterRef[];
  rowParts: { group: string; subgroup?: string }[] };
type PageAnswer = PagedGroups & { read: PageReadMeta };
```

不分组用空字符串键；服务端未回答的组/格需要时按total0、rows[]、nextCursor null、boundary null建出。`boundary` 是显示窗口最远边界，只前进；`cursorBoundary`只记上一服务端页末行，与drop扩大的boundary独立。nextCursor完全采用响应，不解析、不生成。纯decoder `decodePage(page: IssueListPage, meta: PageReadMeta, context: ListContext): PageAnswer` 保留原响应行和本次range；PageAnswer仅含Row，不保留第二份完整Issue。普通merged value的read元数据不作为证据，只有Change第二参数的原值可以作证。原PageAnswer冻结本次范围：第一页lower为空表示负无穷；续页lower为发出时cursorBoundary；末页complete仅表示到正无穷，不能把lower之前当已覆盖。空续页保持cursorBoundary，已完没有下一请求。

`issue-order.ts` 产出 `issueNameKey(text): string`、`compareCodePoints(a,b): number`、`compareIssueTime(a,b): number`、`issueSortKey(issue,order,catalog): SortKey`、`compareRows(a,b,order): number`。Catalog通过生成State/Label/MemberUser目录解析名称/状态组，不向Issue补接口外字段。名字只折ASCII A–Z；逐Unicode码位比（不用localeCompare/UTF-16 `<`）。多值名始终取最小，优先级urgent/high/medium/low/none，空值双向最后，同键UUID随方向。时间键保留纳秒，日期按YYYY-MM-DD。计数key用BigInt：从生产Issue number输入仅接受Number.isSafeInteger且非负，再转BigInt；unsafe值交回错误，不假称能恢复JSON解析已经丢掉的精度。排序fixture含9007199254740993与9223372036854775807，测试须按Node24 JSON.parse reviver的context.source读原数字token转BigInt；其余数值按字段保留原类型。大整数fixture行按原token构造合法 `Row.key: bigint`，直接调用生产 `compareRows` 验证固定预期；不把bigint塞进生成 `Issue.sub_issues_count/link_count:number`，不使用强制类型断言。另用真实生成Issue的safe number调用 `issueSortKey` 验证转key，并单测unsafe number拒绝；名称、日期、优先级等能构成合法Issue的fixture仍覆盖 `issueSortKey`。不修改全局API客户端或HTTP协议。所有真实生产比较器由 `api/testdata/issue-order.json` 的独立预期测试，fixture不进入生产bundle。

实际HTTP由 `issues.go` 将 `time.Time` 放生成响应，`issues_test.go`已有 `.123456Z`；JavaScript毫秒不足。将RFC3339的无小数秒部分解析为UTC整秒，原小数右补9位成纳秒（允许0–9位），用BigInt比较；时区偏移先归UTC。`2026-10-17T10:00:00.123456Z < ...123457Z`，`.1Z == .100000000Z`，等值时后应答赢。只Date.parse整秒不是丢小数版本的比较。不能无效值默认为最新；不合法响应向调用方报告，不写缓存。

### 2.5 T4：确认与重放

`Before = {kind:"known"; issue:Issue} | {kind:"absent"} | {kind:"unknown"}`；`ConfirmedIssueChange={id,beforeSnapshot:Before,after:IssueDetail|Issue|null,drop?:{group,subgroup?,beforeId?,afterId?}}`。每个操作进入实际发送的队列轮次后、发请求之前，从canonical深拷贝冻结beforeSnapshot到该操作pending记录（数组也复制），新建为absent，从未见为unknown。不得在用户排队时过早取、也不得在确认时从已被read更新的canonical重取。pending同时保留这期间观察到的成员关系/排序版本证据；确认从同一操作取冻结before。与after同版本且相关字段相同的read可识别为本次写，不推翻before；若观察到更晚版本，或早于after但晚于before且改变相关字段的其它版本，冻结before不足以证明本次实际旧属，改为unknown并使受影响旧/新组stale，不能猜计数。只保留作此判定所需版本与字段，不建立第二个可写实体缓存。after=null专门表示**已确认删除**。纯函数 `confirmGroups(value, change, context): GroupChangeResult`、`replayGroups(value, change, answer, context): GroupChangeResult` 产出 `{value, verifyIds:string[]}`，context包含query、comparison/catalog和pendingIDs，不在纯模型里请求。

| 旧属 | 新属 | total | rows |
|---|---|---|---|
| 是 | 是 | 不变 | 替换key/version并重排；超出固定window则移出；缺行但窗口内则插入 |
| 是 | 否 | -1 | 移出 |
| 否 | 是 | +1 | 窗口内插入，窗口外等待后页 |
| 否 | 否 | 不变 | 不动 |

旧属首先按该列表原行版本决定，已含after只摆放；无行且整组已加载为否；其余用beforeSnapshot，不知道则stale。多标签/负责人多个组分别判，外层total按去重membership变化，泳道subgroupTotals按其独立membership变化，不能求各格和。

**部分回答的写集合：** `resetCounters` 列出本次响应被merge重置的计数，`rowParts` 列出本次回答的组/格。whole回答拥有outer、其group/cell及返回的subgroupTotals（未返回空组按0）；单组flat回答只拥有该group.total，单格flat回答只拥有该cell.total，其flat total绝非outer。merge与replay都只改这些回答部分：replay只为被重置的计数补确认差量，未回答的group、outer、subgroupTotals和rows保持当前值；已有确认早已改过它们。不同part同时在外时各自重放只改各自集合，不再次更动共享计数。whole可按完整原应答覆盖范围重放。

重放只用PageAnswer原响应：有行比版本；没行而范围覆盖新位置是之前，覆盖旧位置是之后；否则不知道。之后只摆放，之前依原应答+beforeSnapshot计数，不知道只对旧新关系不明的组stale。续页只判断它回答的组。第2种缺行推断返回verifyIds，store每id至多本轮追加一次 `getIssue`；核实中刚补回的id带“待核实”元数据，不能作为可靠可见值。核实成功与拒绝的收敛见§2.6四路表，不能只把错误交回却把推测行和计数永久当真。

删除无after.updated_at：原响应含该id则是删除之前，移行并减该响应所证计数；完整或覆盖beforeKey且缺id则之后不重复减；否则stale。本会话成功delete的id墓碑防晚到read/list复活；拒绝、SessionChanged、普通404都不能创建墓碑；archive/unarchive可逆且不用墓碑。树删除只给缓存中已知后代墓碑；未知后代不猜id/精确减数。成功204先推进该项目read epoch并使受影响列表的在外读取失效，再写已知墓碑及stale，随后才可发新epoch的刷新（§2.6）。因此未显示列表的旧whole/part、详情get和回查也不能把此前未知后代写回。其它独立项目列表不清。

Drop在目标格总显示，落点越界扩boundary，后页按id去重、按全序并入；不清已加载页/滚动。stale来自unknown-before、本地不可判断mention/subscriber/subscribed筛选、重放无法证明、读到第三方membership变化，以及已确认树删除导致旧读失效/缺行回查未证实这两种前述“无法证明”的具体情形。正常可见组立即、未显示组下次显示重取第一页；范围不可判断的整列表fetch；回查拒绝后的有界再验证按§2.6，不因stale形成同步自动循环。

### 2.6 T5/T6：列表协调与唯一工作项

BaseIssuesStore持有 `Reconciled<PagedGroups>`，注册在本代IssueRootStore；project/archived/profile只决定scope/query。对外保留当前调用者需要的getIssueIds/getGroupIssueCount/getPaginationData等读取入口，内部不保留Plane分页兼容模型。接口迁移时修改使用方编译，不把新响应转换回旧TIssuesResponse。

T5为新的list请求首次加入§2.7四个query/表达式纯函数及基本请求测试，消费现有UI表达式与显示设置，不改变未迁移偏好的持久化来源；T7切偏好service，T8完成整个否定层清除和各条件/回读全面测试。查询的语义翻译是批准的业务功能，不是旧响应兼容层。

新的公开协作入口：`fetchFirst(): Promise<PagedGroups|undefined>`、`fetchNext(group,subgroup?): Promise<PagedGroups|undefined>`、`refreshStale(): Promise<void>`、`acceptConfirmed(change): void`、`markMembershipStale(before,after): void`。实际页面调用参数由绑定的scope对象固定，变scope创建/选择另一个key，不篡改已有列表身份。原getFilteredWorkItems改吃Issue，保留子工作项相对日期fromnow行为。

canonical为 `ReconciledByKey<Issue|IssueDetail|null>`，null仅成功delete墓碑。补通用 `receive(key, answer, choose): V` 入口：以choose(held,answer)同步合并，新key可初始化，计应答序；已有在外read重放同一merge以免后到旧应答倒退，不另建parallel cache。IssueStore的choose是服务端updated_at比较，并应用description规则和墓碑。IssueStore公开 `acceptRead(answer: Issue|IssueDetail, token: IssueReadToken): void`、`getIssueById(id): Issue|IssueDetail|undefined`、`getIssueDetailById(id): IssueDetail|undefined`、`confirmChange(change): void`，对删除返回undefined。

较新Issue更新基本字段并保留已知description；较新/相等Detail替换描述；较旧Detail不写任何字段。若只有较新Issue而首次Detail已旧，不补旧描述，详情保持未取得并只补取一次最新Detail，继续竞态则保持可重试状态；无无限自动回路。没有description的旧响应不能把Detail降成Issue。读时membership/排序键变化给含它的列表旧新组stale；同代自己的in-flight修改ID暂不按第三方处理，确认后统一change分发。已失去项目/工作区可见性的canonical访问器不给旧值。

#### 项目读取代际与成功删除（B-L2）

`IssueReadToken = Readonly<{loginId:string; serial:number}>`；每代IssueRootStore维护单调readSerial、每项目lastDeletionSerial及workspace下的最近项目删除serial。`captureRead(): IssueReadToken` 在**每个请求真正发出前独立执行 `++readSerial`**，取得唯一发出序号；它不只在删除时递增。删除水位与下面每id核实的freshness水位分别保存，不能混成一项。单项目list、详情/回查的canonical接收先核同loginId且token.serial不早于响应project的lastDeletionSerial；多项目profile查询有显式project_id时按该集合、未指定时按workspace可见项目范围核删除serial，有任一相关项目删除即**整页丢弃并保持stale**，不得裁剪旧行后沿用旧total。普通单项目读不因别的项目删除而失效。

成功树删除只在当前session的204确认后调用 `invalidateProjectReads(projectId)`：readSerial+1并记该项目/工作区水位；先对覆盖此项目的list调用Reconciled.invalidate（含未显示list），再保留stale及已知墓碑，最后发新token读取。canonical的acceptRead在任何receive/版本/description写入之前核令牌；get和verify也如此。list在接收完整Issue行、decode/merge、清stale之前均核令牌，旧read不能先写canonical再被列表层丢掉。新refresh期间若又发生一次成功删除，水位再次推进，夹在两次删除之间的read同样失效。拒绝、SessionChanged或普通404绝不推进删除水位；这些水位只属于当前RootStore，不写localStorage、不改服务端协议。

#### 旧详情消费者的同Task收口（B-L3）

T6同时改 `C/store/issue/issue-details/{issue,sub_issues,link,relation}.store.ts`；执行前对实际P10a合并树再枚举inbox/attachment等剩余写者。当前读完整实体的 `addIssue` 调用改为捕获read token后统一acceptRead；已有本段CRUD/归档操作只走根确认协调器。旧 `updateIssue(id, Partial<...>)`、直接 `set(issuesMap, …)`、计数±1/用子项数组长度等不能绕过canonical：对应旧操作成功后先标该对象待权威重验，再经 `refreshAuthoritative(issueId): Promise<IssueDetail|undefined>`（IssueApiService.get+token检查）取得真实count/updated_at；失败不本地加减，不伪造版本。只有partial/旧关联片段时按已有id走同一权威读取，不能把片段断言成生成Issue。

供旧读者的 `issuesMap` 是 `Readonly<Record<string, Issue|IssueDetail>>` 派生只读视图，不是第二份实体缓存；需要的旧addIssue/updateIssue写API彻底删除，不留无行为壳。这里只收紧共享存储边界，旧详情资源地址/业务流程完整替换仍由P11负责。若权威回读失败，采用下表未证实状态，不把“旧写成功”伪报为“最新实体已取得”。

#### 缺行回查的四路状态转移（B-L6）

核实元数据只存id、round、状态、error和 `freshnessFloor`，不复制实体。进入待核实时先记当前readSerial；回查404/其它拒绝实际落定时把该id的freshnessFloor提升到此刻当前readSerial（覆盖此前已经发出的所有read，包括核实在外期间新发的read）。它只是“恢复可见性必须有之后新发出的证据”的水位，不是删除水位，也不产生墓碑。`getIssueById/getIssueDetailById` 对待核实/不可确认可见的id不给可靠活值；列表展示也不把它作为普通确认卡片，相关stale计数不能表现为最终值。元数据不等于删除墓碑。只有同session、项目令牌有效、`token.serial > freshnessFloor` 且服务端版本规则允许的完整权威读，才能清该id核实状态；这证明请求在最近核实拒绝之后才发出。404之前已在外的旧list即使稍后回答、updated_at与旧canonical相同、项目删除令牌仍有效，也不能写回该id、恢复其可靠可见性或清相关组的stale。该freshness检查在canonical receive及清核实/清stale前执行；旧页即使缺此id，也不能据其旧范围解除本轮待重验状态。

| 回查结果 | canonical / 列表 | 后续请求与收敛 |
|---|---|---|
| 当前token且高于本id freshnessFloor的成功IssueDetail | 经版本规则接收；同after可确认本次推断，较新成员关系变化标对应组stale | 清本id核实状态；受影响范围至多安排一轮新token刷新，不回溯重复同一个get；序号不足的成功答按旧读丢弃 |
| 404 | 落定时提升本id freshnessFloor；对象不可确认为可见、访问器不供旧值；推测行不当活值、相关组/list保持stale；无永久墓碑 | 同一round最多一次受控可见范围刷新；空页可让组计数/rows收敛，但对象仍待以后有效权威读再恢复，不从“空页”推永久删除 |
| 其它拒绝（含403/500） | 落定时提升本id freshnessFloor，保留error与待重验状态，相关组/list仍stale，不确认推测行/计数 | 错误交回调用方；本round不自动再get或循环刷新，下一次显式重试/重新挂载开启新round，使用新token |
| SessionChanged / 令牌已因删除失效 | 丢弃此旧流程，不写新一代cache/错误/核实元数据，不清新一代stale | 当前session的正常挂载决定后续；旧round不安排请求 |

每个read/replay round维护已核实id集合及每scope刷新预算1，结束后不因同一stale自动重新开round；UI重试/下次挂载才建立新预算。并发part共享当前协调round对同id去重。仅高于该id freshnessFloor的新发有效list返回完整该id时才可重新证实可见并清核实元数据；新发的空页可使回答范围的rows/计数收敛，但不把该id永久删除。较旧版本、失效令牌或核实拒绝前已发出的list均不能清，即使服务器updated_at相同。404/拒绝不推进项目删除水位，不影响无关项目。

### 2.7 T7/T8：显示偏好和筛选

ProjectPreferencesService现有get/update签名不变；project filter按生成filters/display_filters/display_properties消费。每次PATCH只带要变的顶层字段的全值，其余字段省略；同字段排队时从最后确认值计算。navigation/sort_order不被覆盖。服务端不会自动使用偏好，列表显式拼查询。

`issue-filters.ts` 目标纯接口：`toProjectFilters(expression): ProjectIssueFilters`、`toFilterExpression(filters): TWorkItemFilterExpression`、`projectListQuery(filters, display): ProjectIssueQuery`、`memberListQuery(filters, display, relation): MemberIssueQuery`。标识字段映射state→state_id、labels→label_id、assignees→assignee_id、created_by→created_by_id；日期条件保持重复AND，不把两个条件随意合并成一个。one-level AND和每种已支持条件逐一测试；None/me按契约；不支持的持久化属性/组合在本地选择明确默认并清理，不偷偷传给服务端造成整页失败。

ProjectDisplayFilters生成字段为string，query为enum；用允许值目录显式收窄，不用断言伪造安全。项目分组不带project_id，member允许更窄的词汇，无subgroups。表达式只用于UI state，持久化用ProjectIssueFilters。删除否定参数/恒等alias/hook，不留allowNegative:false壳。旧/user-properties两个方法、两例外、mounts.workItemFilters、p2-visibility/p8-project-preferences/s7-chinese-project三故事放行在T7一起消失。

### 2.8 T9/T10：浏览器偏好、会话与权限

已归档/profile在localStorage的issue_local_filters，用现有主体作用域隔离；归档richFilters/rich_filters读写统一成rich_filters，与profile一致，历史错误键读一次可迁移但不同时保留两份权威。存储失效/损坏恢复默认，不抛未处理错误；不把其它账户偏好混用。看板折叠kanban_filters保留。草稿无筛选：WorkspaceDraftIssuesFilter及root/useIssues/types/barrel引用删除，draft列表本身留P11。

IssueRootStore的autorun改为rootStore/router/user目录计算属性，无需新reaction或dispose；空目录不能保留上一项目数据。列表侧project-layout-root、archived-issue-layout-root、profile-issues统一useSessionSWR；键名、loginId、scope参数完整，null不请求，拒绝停止loading。访客个人主页不取issues，非成员项目页不取list；归档只admin/member。三处rootStore同步读继续只读，发请求经拥有者RootStore。组回查第三方版本也受同一会话守卫。

### 2.9 T11/T12：串行修改与全副本确认

每工作项的修改共用根store一条oneAtATime队列；为避免共享邻接列两个不同ID拖动冲突，拖放使用该根store统一的拖放队列，再在对应工作项队列里执行（固定取得顺序，任何其它操作不反向持有两队列）。请求到发出时才从canonical读取邻接IDs现值计算sort_order，邻居已不存在则按当前有效落点/首尾65535间隔重算，不发送旧平均数。无邻居使用65535；首尾/中间算法与现有布局/服务端sort_order契约一致。

目标入口 `updateIssue(id, makeBody: (held:Issue)=>IssueUpdate, drop?): Promise<IssueDetail>`，拖放传makeBody闭包，不提前构造body；create/delete/archive/unarchive同根协调。body逐字段构造，禁止spread Issue带id/project_id。请求中仅UI pending/临时拖放呈现变化，canonical/PagedGroups不能变；拒绝交回且UI回原处。成功确认一次，从该操作pending取实际发出前冻结的beforeSnapshot并核对期间读证据→按已确认删除时先失效读代际→写各列表→写canonical，pending在对应请求finally结束，队列拒绝后继续。

每修改表覆盖全部已取project/archived/profile副本、未取副本、pending、失败、SessionChanged、晚到读取与重取。跟进toast/导航必须followInSession；本段改到的组件签名跟进归本段，完整详情/草稿新交互仍留后段。

### 2.10 T13/T14：挂载、门禁与文档

S2四角色加入项目issues页，按角色列真实请求；成功只放行新list/preferences，非成员不放行list。P10a删除的M6/M7页不恢复。旧旧服务路径守卫只缩小例外；issue-optimistic不写跨函数await位置的脆弱正则，使用明确范围的运行mutation表证明未确认不写；negation-filters词项全删后用静态规则。改到文件死成员/prop清理、警告上限随真实减少调低。

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
| D11 | §12粗任务落到可编译提交：T1准备有测试消费的薄服务，T5基类和消费者同时删除旧形状、接线，并前移所需query纯函数；T8仍负责全部否定层/语义矩阵。避免跨Task坏编译和临时response转换层，代价是T5较大（约900–1400行，独评检查拆分职责）。 |

均是已批准设计内的局部精化；如实施发现需要改变模块边界/跨模块协议，先提交已完成工作再上报。

## 4. 验收标准

- [ ] A1 全部14tasks实施及逐Task独评完成，无未决Important/Critical；核心类型承接P10a，列表旧类型/旧游标/乐观回滚移除。
- [ ] A2 list services和preferences消费真实operation；query/body精确，拒绝及SessionChanged不写；旧列表服务和/user-properties放行消失。
- [ ] A3 fetchPart全部序列、原应答replay、partial计数写集合、组/格/窗口/多值/总数矩阵通过；I2九例、M3五例、续页范围和删除204附加例逐条有证据。
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

最高风险为重放范围与计数、server版本精度、canonical description、不同scope/会话混写、不同卡片共享邻接排序。已分别绑定T2–T6/T10–T12测试与变异。共享Reconciled是通用设施，改变接口必须保留原M2/M3所有消费者语义并跑其真实回归。14任务约350–1100行/任务，paged-groups设计允许约450行纯模型；超出职责再按order/model分拆，不为指标压缩测试。计划与测试预计接近前端改写规模，发现任务不可编译的跨段循环应重切局部任务并记录，架构变化才上报。

## 7. 已知限制、移交与关闭条件

准备期无实现/原型/行为测试，依赖P10a将删去大量当前文件，执行前必须重新盘点精确路径与基线。输入handoff的项目只在实施且验证后追加处理结果，总status保持open至M4收尾。服务端列表没有每行排序key，前端必须基于当前目录按相同合同形成key；目录异步拒绝不能转成静默空列表。

关闭条件：本规格§4全通过、deep-review无未决项，按负责人另行授权--no-ff集成、准确main CI绿、证据归档哈希与清理完成。当前不满足这些完成条件。

## 附录 A：准备记录与证据边界

2026-10-28已只读核实设计§3.17/3.22/7/9.5/12、实际schema/HTTP时间字段、列表store/service/挂载、Makefile/package/test设施及关键词解析。盘点原件在仓库根 `.superpowers/sdd/P10-preparation-20261028/code-inventory.md`，与P9旧worktree生命周期无关。未执行生成、安装、原型、产品测试或变异。仅运行Node24语法能力probe：实际v24.15.0的JSON.parse reviver对上述两个大整数返回原token，转BigInt后exact=true；这只证明fixture无损读取方式，不是产品比较器通过。所有产品命令和目标测试是未来实施步骤。本轮首审提出六项Local文档问题，本修订逐项补齐规则/反例，原审者已确认六项文档层关闭，见[review](../reviews/P10b-web-list-data-review.md)；不把作者自查当独立结论，也不把文档通过当产品验证。P10A-S1已获负责人A裁定，验收归属同步后的定向复审见P10a review；当前仍不实施。P10a合并后还需执行前复核，并由负责人授权开始。
