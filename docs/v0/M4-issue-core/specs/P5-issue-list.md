# M4/P5 工作项列表与显示设置：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M4/P5 issue-list |
| 状态 | 已完成：T1–T16、整分支 deep-review/修复复审及授权合并/main CI/清理全部完成；精确证据见阶段评审 |
| 基线 | `e9782b8f7e7571b39a4eb8924b4b72a6e60eac51`，P4 合并 main |
| 基线验证 | [main CI 40180521025](https://github.com/open-nerve/NerveProject/actions/runs/40180521025) 的 server/web/e2e 全部成功；此证据不代表 P5 验证 |
| 上级设计 | [M4-design](../M4-design.md) §0.3、§2、§3.4、§3.7、§3.16/3.17、§3.23、§4.15、§5.1/5.2、§6.5/6.6、§12 |
| 方法与纪律 | [02-method](../../claude_to_gpt/02-method.md) §3；[03-rules-and-pitfalls](../../claude_to_gpt/03-rules-and-pitfalls.md) |
| 计划 | [P5 plan](../plans/P5-issue-list.md)，16 Task、文件表、消费/产出签名与验证命令 |
| 评审记录 | [P5 review](../reviews/P5-issue-list-review.md)，保留静态预检来源与实施证据 |

## 1. 目标
交付两条列表 API，19 个排序码、筛选、分组与泳道、精确总数和游标；普通可读项目目录及 P3 关联列表的替换；`sort_order` PATCH；个人项目偏好的三类显示设置；权限、快照、性能和 I3–I7/I19/I20 接口故事，扩充 I15。列表词汇供偏好校验复用。

直接消费 P4 已有 `shared.PageSize`（默认 50、1–100）和 `postgres.ReadOnlySnapshot()`，不重写两者。已有 `Issue` 为列表行，`IssueDetail` 才带描述。P5 不提前加入 `link_count`、附件、迭代、模块、搜索、草稿、事件、前端列表页面；`link_count` 的两个排序码留 P8。旧筛选 store 与 `/user-properties/` 的迁移留 P10b，其 `until: M4/P11` 例外不提前删除。仅修复新增生成字段导致的现有类型夹具。

## 2. 交付物

文件、职责、类型签名与依赖以 [plan 文件表和跨 Task 签名](../plans/P5-issue-list.md#文件结构职责与依赖) 为唯一落点；以下固定全部对外契约与重要内部不变量。16个验收任务不缩减原设计范围；T5、T12与T13各按规模细分两个连续实施/独立评审单元，当前共19个实施单元，各自父任务验收仍在同一完成线。

### 2.1 不可变约束

- 实施工作树为 `.codex/worktress/codex/m4-p5`，主目录始终 main；开始实现前落正式 spec/plan 并做独立预检。P5 起步/局部实现按既有自主授权与独立预检继续；本阶段完成后先交负责人安排整分支终审，再处理意见并合并，除非负责人另作安排。真正架构改动、跨模块语义改变或高风险仍须上报。
- 模块不互相导入、不跨模块 SQL JOIN；跨模块小值由 bootstrap 转换。SQL 仅经 sqlc；三条静态列表 SQL，禁止拼 SQL、模块 raw pgx/SET、放宽守卫、提前加默认排序索引分支。
- 所有列表鉴权、目录、查询、空页补 total 均使用一个独立 READ ONLY / REPEATABLE READ 事务内部的 ctx；不从已有写事务调用来期待隔离升级。端口错误原样传播。
- 写继续经 Locks.run；不可见先 404，再角色/作者 403；项目归档→工作项归档→无改动→依赖检查→锁下时钟→写。全局表序：issues→assignees→labels→relations→links→subscribers→mentions→comments→issue_reactions→comment_reactions→activities→description_versions。
- 不新增依赖，不全局安装；测试使用现有 pgtest/issueWorld/e2e fixtures。新逻辑文件约 400 行内；生成物仅由生成命令产生。容器测试独占执行，不同时跑多套。
- 新操作当 Task 加入双向契约、权限矩阵和运行角色验证；变异必须实际运行、恢复并重跑。测试名/数字/性能/CI 结论只能登记实际证据。

### 2.2 API 与可见性

路径均前缀 `/api/v0`，认证沿现有 bearer/PAT。

| operationId | 方法/路径 | 范围与权限 |
|---|---|---|
| listIssues | GET `/projects/{project_id}/issues` | `archived=false` 默认，仅活工作项，`issue.read`；true 仅已归档工作项，`issue.archive`。项目本身已归档仍允许读，按显式项目授权；不能用排除归档项目的跨项目目录决定此路径范围 |
| listMemberIssues | GET `/workspaces/{slug}/members/{user_id}/issues` | `relation` 必填 assigned/created/subscribed；`member_issue.list`，工作区管理员/成员允许、访客拒绝；目标存在未删除成员记录即可，允许 is_active=false；否则 `workspace.member_not_found`。只列调用者有效加入且未归档、未删除项目中的未归档工作项 |

个人主页严格区分调用者与目标成员：`ListingInput{ActorID uuid.UUID; TargetMemberID *uuid.UUID; Projects []VisibleProject; Query domain.ListQuery}` 由 app 构造，`Listing.Read(ctx, input)` 消费；仓储 `ListRead` 同样带 TargetMemberID。项目列表 target=nil 且 Relation为空；个人主页在同一快照完成授权和目标成员存在检查后，按路径 user_id 设置 target，Relation只能为 assigned/created/subscribed。SQL 的 relation_user_id 从 target 取得，ActorID只用于可见性、OwnOnly和调用者计数，二者不能互换。Listing/仓储入口校验 Relation 与 target 是否同时匹配，内部不一致报错误，不能静默回落为actor；目标不能从query改写。uuid 沿用本仓库 Go 工具链的标准库 `"uuid"`。T10以A≠B、三类关系与caller OwnOnly的实际SQL反例验证。

两者排除已删除工作项和分诊状态。每个项目独立按调用者的 OwnOnly 过滤；工作区管理员兼项目访客仍受 OwnOnly 限制；工作区管理员/公开项目旁观者不是项目成员时不自动获得工作项可见性。总数、排序用子项数、父项 ID 的可见性必须与已有 HiddenFrom/seen 规则一致。

`VisibleProjects` 增加普通 `Visible(ctx, workspaceID, actorID)`；现有 `Historical` 保持原样：历史引用目录忽略项目/成员 deleted_at 与项目 archived_at，但必须 is_active。普通方法要求项目/成员未删除、项目未归档、成员 active；不把两种查询合并为松散布尔开关。

状态/标签通过 project 提供完整批量目录；仅列表状态目录排除 triage，现有写入 States 不变。姓名目录先从所选项目的未删除 issue_assignees 行取得账户 ID，再调用现有 PublicProfiles.DisplayNames，包含已离开或降为访客的负责人，不能改用活跃成员名单。

P3 `ListIssueRelations` 改用普通目录过滤跨项目目标，替换逐项目 seeEach 和 ArchivedProjects 链，保持返回顺序、Issue 行裁剪、子项计数、父项隐藏与错误传播。源项目先按原规则授权；如果源项目已归档，显式保留源项目 grant，使同项目可读目标不会被普通目录误滤。目标工作项归档仍过滤。整条读取链放入同一 ReadTx。确认无消费者后删除旧端口/适配/假对象，不影响 Historical。

### 2.3 查询语言与错误

| 参数 | 项目 | 个人主页 | 语法 |
|---|---|---|---|
| state_id | 是 | 否 | UUID CSV |
| state_group | 是 | 是 | backlog/unstarted/started/completed/cancelled CSV |
| priority | 是 | 是 | urgent/high/medium/low/none CSV；none 是枚举值 |
| label_id | 是 | 是 | UUID/none CSV |
| assignee_id | 是 | 否 | UUID/me/none CSV |
| created_by_id、mention_id、subscriber_id | 是 | 否 | UUID/me CSV |
| project_id | 否 | 是 | UUID CSV，与可读项目集合求交 |
| start_date、target_date | 是 | 是 | 单日、a..b、..b、a..、none，可重复，条件 AND |
| sub_issue | 是 | 是 | true 默认；false 排除有 parent_id 的行 |
| archived | 是 | 否 | true/false，默认 false |

CSV 内 OR，字段间 AND；none 与 UUID 可并列。日期范围闭合，缺日期仅满足 none，none 不与其他日期条件并存。非法日期、反向单个区间、空 CSV 项、错误 UUID/词、重复非日期参数均 422，field 为参数名，code=not_allowed。两个各自合法但交集为空的日期区间返回空列表，不额外判非法。me 按调用者解析。已删除/外项目但语法合法的筛选 ID 只影响结果，不产生对象存在性泄漏。

未知参数（包括 cycle_id/module_id/link_count 相关参数）422 not_allowed；个人主页不接受项目专用参数。`limit` 沿 shared.PageSize：整数越界 422/out_of_range；非整数保留既有绑定的 400；非法 cursor 400/bad_request。必填 relation 缺失沿生成必填参数 400；给出未知 relation 422。HTTP 参数 schema 按下表固定，UUID/date/bool 的业务词值仍在领域解析。

| 查询参数 schema | style/explode | 验证归属 |
|---|---|---|
| state_id/state_group/priority/label_id/assignee_id/created_by_id/mention_id/subscriber_id/project_id：array，items string | form / false，CSV | 不设UUID format或pattern；列表词汇检查每项 |
| start_date/target_date：array，items string | form / true，可重复 | 不设date format；每次条件AND，领域校验 |
| sub_issue/archived：string（文档写true/false及默认） | form / true，单值 | 不用boolean schema；领域只允许true/false |
| group_by/sub_group_by/order_by/relation：string，共享枚举schema | form / true，单值 | 领域检查页面词汇；relation仅个人主页required |
| group_key/sub_group_key/cursor：string | form / true，单值 | key词法422；cursor载荷400 |
| limit：integer，默认50，范围1–100 | form / true，单值 | 生成绑定检查整数，shared.PageSize检查范围 |

**HTTP 接缝的最终方案（T9/T10真实接口与T14回归已验证）：** M2 已明确参数绑定错误可以在认证/限流前回答，保留此顺序。Register 的标准 StdHTTPServerOptions.ErrorHandlerFunc 仅在两条P5 GET路由、errors.As 为 gen.InvalidParamFormatError、ParamName属于该路由已声明的非日期query且真实 r.URL.Query()[name] 长度>1时，映射422/not_allowed；其余全部调用既有 api.Errors.BadRequest。不检查错误文本，不误用未实际产生的 TooManyValuesForParamError。正常请求通过 gen.NewStrictHandlerWithOptions 的 StrictMiddlewareFunc，只对 ListIssues/ListMemberIssues 把真实query深拷贝为纯 map[string][]string 放入私有context，随后handler交给用例完整解析。无URL改写、无占位、无BaseRouter装饰、无第二路由、无平台/生成器/守卫改动。缺token+重复在绑定阶段仍422；坏path先400，其他API原绑定错误仍400；绑定成功的缺token请求仍401。日期重复不走错误映射，保留全部值AND。来源与静态核对证据写入 [P5 review](../reviews/P5-issue-list-review.md)；原401优先假设与占位候选均撤销。

**追加整审明确的编码边界：** 公共 `@nerve/api-client.createClient` 为两条列表 GET（含 `request("get")`）默认按参数区分 CSV 数组与重复日期；调用者显式 serializer 仍优先，其他路由/方法不变。故事直接消费此生产实现。绑定成功后，strict middleware 用 `url.ParseQuery` 读取 RawQuery；非法百分号转义或裸分号不得静默丢字段，返回400/bad_request；可解码但非法的业务词仍422。该校验在既有认证之后，缺token仍401；先发生的路径/必填/整数/重复绑定错误顺序不变，不扩展到旧接口。根因、局部裁定及真实红绿见[本轮 deep-review](../../../reviews/re-02-M4P5整体审核.md)。

### 2.4 分组与响应

项目 group_by：state_id/state_group/priority/label_id/assignee_id/created_by_id/target_date。个人主页：state_group/priority/label_id/target_date/project_id。sub_group_by 仅项目，使用项目同组词汇，必须有 group_by 且不同。组键按字段为 UUID/枚举/日期/none；UUID类的group_key/sub_group_key按UUID值规范化，大写、compact等合法拼写与标准文本指向同一组，首段、续页及空页total一致；none 仅适用可为空或多值为空的字段，不接受 state_group=none 等虚构键。

多值字段按每个值展开；无标签/负责人为 none；列表行自身的 label_ids/assignee_ids 始终完整。组序：priority urgent→none，state_group backlog→cancelled，日期升序，其他文本升序，none 最后。只返回非空组，limit 是每组/格条数。

```text
平面/组格续页 {total,data:Issue[],next_cursor:string|null}
分组          {total,groups:[{key,total,data:Issue[],next_cursor}]}
泳道          {total,groups:[{key,total,subgroups:[{key,total,data,next_cursor}]}],
                    subgroup_totals:[{key,total}]}
```

没有的分支省略；空数组 []，末页 cursor=null；契约闭合结构，不用 oneOf/allOf。总体/组/泳道/格 total 分别是该层不重复工作项数，不能累加格计数或把展开后的行数当总体。

加载更多保留原筛选、group_by/sub_group_by/order_by，另加 group_key，格再加 sub_group_key，返回平面封套且 total 为所选组/格总数。`label_id=A&group_by=label_id&group_key=C` 必须 A AND C。日期 none 组同理，不改原日期筛选。

group_key 无 group_by、sub_group_key 无 group_key/sub_group_by、键语法不符均 422；分组 cursor 缺 group_key，泳道 cursor 缺 sub_group_key 为 cursor/not_allowed。局部确定：有 sub_group_by 时出现 group_key 必须同时给 sub_group_key，即只支持格续页；缺它返回 sub_group_key/not_allowed（cursor 同时出现时优先 cursor/not_allowed）。不存在但语法合法的组键返回 total=0。

### 2.5 排序与游标

P5 恰好 19 码：sort_order 仅升序；created_at/updated_at/start_date/target_date/priority/state_name/label_name/assignee_name/sub_issues_count 各升降。默认 -created_at。-sort_order、±link_count 及其他未来码 422。

- priority 数值 urgent=4/high=3/medium=2/low=1/none=0；排序与分组序分开。
- 名字只折 ASCII A–Z，按 Unicode 码点序；Go UTF-8 比较、SQL COLLATE C。label/assignee 总取最小名字键，降序也不改成最大值。没有名字为 null。三种名称游标的字符串不能含U+0000，返回400/bad_request及cursor/invalid_format；合法Unicode、空字符串和null语义保持。
- 两方向 null 都在最后；同键 UUID 与主键同方向。时间保留 PostgreSQL 微秒，日期保持纯日期；计数用精确整数，不经 float64 丢失。sort_order 是有限浮点数，按存储值排序。SQL CASE 不因 sort_order 的 float8 分支把微秒/计数转为 float8；最终k_num各分支显式numeric，sort_order先转有限浮点位序int64，游标使用同一转换与全序。相邻可表示 sort_order、整数精度边界和逆 UUID 必须在真实 SQL 对照验证。T5已证明Flat的有限浮点位序、时间微秒/宽日期及精确count表达式；T6已对Groups自身验证邻接浮点/2^53微秒/null两方向续页；T7也已对Subgroups自身验证同类精度/双向日期null及公开游标。最终T12已对四入口各19排序及项目7/个人5/42字段组对分别验证，不把两维扩成笛卡尔积，也不把代表性结果外推。
- 子项数只计调用者可见、未删、未归档的直接子项；排序键与行中 sub_issues_count 同义。
- payload 为 `[order,key,id]`，经现有 shared.EncodeCursor/DecodeCursor 的 v1 canonical base64url 封套；key 按排序为 UTC 时间字符串、日期字符串、名字字符串、数值或 null。形状、键类型、非法 UUID、非 canonical、版本、排序码与请求不符均 400。数值解码使用 json.Number/明确类型，不能 unmarshal 到 interface{} 后默认为 float64。
- 时间游标输出为UTC固定六位小数，拒绝亚微秒及四位年份范围以外输入；日期和名字族允许null，created_at/updated_at、sort_order、priority、sub_issues_count的key不可null；priority为0–4整数，子项数为非负int64，sort_order为有限float64，正负零统一编码为正零。两页之间姓名/子项数量变化可导致行移动，这是现有全序键集方案的已知代价，不宣称跨请求固定快照。

`api/testdata/issue-order.json` 固定名字、日期、null、同键逆 ID、ASCII 与非 ASCII 大小写、U+FF5E/U+1F600 的期望序。P5 Go 消费；P10b 必须让生产前端比较器及 vitest 消费同一文件，不提前创建无消费者的 TS 排序器。

### 2.6 查询结构与性能完成线

三条 SQL：list_issues.sql、list_issue_groups.sql、list_issue_subgroups.sql；参数目录采用 `ROWS FROM(unnest(...),unnest(...))` 并列数组，长度一致，不交叉相乘。接口类型为 interface{} 的列若选出需显式转型。所有数组适配端规整非 nil，SQL 同时 `coalesce(cardinality(...),0)=0` 防御。

filtered AS MATERIALIZED 在分页前完成范围/可见性/全部过滤，每 issue 一行。窄键 CTE 计算 k_null/k_num/k_text，ORDER BY 与游标谓词每条 SQL 各一份；分组 LEFT JOIN LATERAL 展开，row_number 按组/格全序取 limit+1。总体窗口在 filtered，组窗口在组展开后、子组展开前，泳道窗口在独立子组展开层，格窗口在双展开后。完整数组及非排序计数仅对返回行计算；排序用名/计数才对候选行算键。

空的无 cursor 页 total=0；cursor 页没有行时同事务、同原筛选及组格选择、去 cursor、limit=1 再查 total。不得换目录、换连接、扩大查询范围。

池 RuntimeParams 强制 jit=off、plan_cache_mode=force_custom_plan，即使 URL 相反也覆盖；保留现有 UTC codec。实际池不同连接读回验证，不只测 map。

最终生产 SQL 用 10k/100k 对照：默认页、label_name、state_id 分组、label_id 分组、泳道与代表性筛选；完整查询含目录/映射另测端到端耗时。记录 PG/硬件、seed 分布与核对、预热、执行次数、每次/分位耗时、EXPLAIN(ANALYZE,BUFFERS)、实际索引；原 S1 程序未留存，不声称复用了原 seed。

设计基准 10k 约 2–100ms；100k 默认55–125ms、label_name210–430ms、state组0.2–0.5s、label组0.45–0.7s、泳道0.7–1s，原负载波动约±20–30%。不是不稳定 CI 的硬毫秒断言；显著越界须解释/优化后实测，不能以均值掩盖超预算项。v0 目标每项目约20k 未删除项；CASE 排序扫描全部匹配行、跨项目 VisibleProjects 哈希连接可能扫描整个工作区，是明确成本。更大规模/索引特化交 M8；实际结果与索引如下。

T12最终源码为 `f73e85ed46303e813c8cee44b5145e22607e474d`。32轮采样时HEAD仍为修复基线，完整diff与16文件SHA已独立核对为该最终提交；两规模各13场景、1预热+5实测，130样本及456份真实SQL/typedargs计划全部保存。运行角色、RO/RR及生产池配置不变；环境macOS26.6.2、M5 Max/18 CPU/128GiB（Docker VM报告7931MB）、PG18.6 aarch64、Go1.27.1，MaxConns4/cache statement。N=10k/100k正常根项，默认含活跃孩子实际匹配10200/102000；小目录/0–2关系度数，OwnOnly另仅52匹配。模块时间仅httptest ServeHTTP，不含网络/真实token认证。

| 场景 | 10k列表SQL中位 ms | 100k列表SQL中位 ms | 100k模块中位 ms |
|---|---:|---:|---:|
| 默认页 | 11.34 | 105.53 | 117.02 |
| label_name | 22.73 | 231.55 | 243.14 |
| state组 | 20.24 | 205.16 | 219.24 |
| label组 | 26.21 | 287.62 | 303.21 |
| 泳道 | 56.14 | 622.38 | 646.32 |
| sort_order | 14.35 | 147.71 | 158.94 |
| 成员分配 | 6.65 | 96.59 | 107.81 |
| 成员订阅 | 6.82 | 97.34 | 108.47 |

全部场景min/median/max、五样本、目录/其他SQL、分布与原计划见[评审T12b](../reviews/P5-issue-list-review.md#t12b真实规模性能修正与单一排序收口已验收)。最终Flat用唯一窗口业务全序生成内部序位，序位上界与同值LIMIT选keys，完整投影后仅按序位输出；游标及total位置不变。默认完整回表从优化前102000次降为51次，实际使用issues_pkey；默认/label_name还使用issues_project_id_idx、issues_parent_id_idx及两关系唯一索引，成员计划另见issues_project_id_state_id_idx和issue_subscribers_issue_id_subscriber_id_key。120条现存索引DDL已保存，未新增索引。

单一排序有代价：相对首版双排序实现，sort_order及分配/订阅场景约+59.5%/+87.2%/+83.2%；不能隐去退化，也不机械套用默认页预算。默认EXPLAIN execution143.621ms，窗口前全量102000候选external merge（6592kB）、temp read/write1173/2364 blocks；成员页后51次主键查，但分页前仍有102256次相关关系索引扫描。事后EXPLAIN与请求计时分开。保留20k目标、M8的大目录/跨项目/相关过滤/临时块与索引特化评估，不宣称消除扫描或同seed复现。

### 2.7 sort_order PATCH

现有 updateIssue 请求新增可省略、不可 null 的 number；零允许，省略保持旧值。IssuePatch.SortOrder *float64、Changes.SortOrder *Changed[float64]、Changes.None、IssueFields.SortOrder、合并/SQL/HTTP 全链同 Task 加齐，拒绝非有限纯用例输入。排序修改是有效写，锁下更新 updated_at/by；相同值为无变更，仍服从归档拒绝顺序。

与 state_id 同改时按既有状态变更设置 completed_at，产生一条状态动态；排序本身不创造新 activity field。`skip_activity` 只要出现（true/false）且带 sort_order 都 422 skip_activity/not_allowed。只补块 ID 的 P4 路径不能混入排序写。创建最大值+步长逻辑不变。P3 子工作项列表继续 created_at/id；M4 §5.1已正确写(created_at,id)升序，T16仅复核，不改现有产品行为。

### 2.8 三类项目显示设置

迁移 `server/migrations/sql/00026_project_project_user_properties_issue_display.sql`：project 所有的 project_user_properties 加 filters/display_filters/display_properties 三列 jsonb NOT NULL，固定默认值及形状 CHECK。ADD COLUMN DEFAULT 给已有行默认；无需新表/删除路径。Down 删除三列及约束，验证重迁移；已有成员恢复保留偏好。

- filters 默认 {}；闭合 10 个可选键：state_id/state_group/priority/label_id/assignee_id/created_by_id/mention_id/subscriber_id/start_date/target_date；每值 string[]，每个元素为一个 CSV 项或一个日期条件。不含 archived/sub_issue/project_id；空数组等于该字段无筛选。不能一个元素再塞 CSV 多项。
- display_filters 恰好 layout/group_by/sub_group_by/order_by/sub_issue/show_empty_groups/calendar；默认 `{layout:"list",group_by:null,sub_group_by:null,order_by:"-created_at",sub_issue:true,show_empty_groups:true,calendar:{layout:"month",show_weekends:false}}`。layout 为 list/kanban/calendar/spreadsheet；calendar.layout month/week；其余词汇复用项目列表。
- display_properties 恰好 assignee/start_date/due_date/labels/key/priority/state/sub_issue_count/link/created_on/updated_on，全部 boolean，默认 true。link 是显示偏好，不使列表提前多出 link_count。

每个 CHECK 先 CASE WHEN jsonb_typeof=object，再查键集合与值类型；filters 用 immutable jsonb_path_exists 查 string[]。JSON null/数组/标量均 check_violation，不因运算函数类型错误逃成另一种错误。DB 不查具体分组/布局/排序词值；词值交 issue/domain/listvocab。

project.app 声明 IssueListSettings.Validate，issue.Provide 交纯实现，bootstrap 转换各模块 app 小值。GET/PATCH 闭合 typed DTO，不用开放 map schema，不放宽 closedObject 守卫。结构错误/null/未知 body 字段 400；合法结构中的非法词值 422 not_allowed，定位 filters.label_id[0]、display_filters.group_by 等。

GET 缺行返回全部默认而不 INSERT；PATCH 哪个顶层字段出现就整体替换哪个，{} 清空 filters，其他字段省略保留。Upsert 每列显式 set 标志；不能先读整行再全列覆盖。navigation 与 filters、两类新设置并发写不丢失对方修改。只改自己的行，其他成员不变；项目归档读写权限沿已有 preferences 规则。

### 2.9 故事、文档与交付范围

16 Task 的测试与变异在 [plan](../plans/P5-issue-list.md)；重点反例：多值同时筛选/分组/双展开，empty cursor total，微秒+逆 ID，两方向 null，退出成员仍在姓名目录，管理员兼 guest，归档源项目关联，原始重复 query 认证顺序，偏好并发整字段替换。

I3/I4/I5/I6/I7/I19/I20 API 故事各文件，I15 增加归档列表；全部前置经 API。I20 访客项由成员创建后降级，四格包含 P4 评论201。I4 覆盖19排序码与日期none，不测 P8 link_count。I7复用创建接口，不加快速创建端点。页面故事由 P12–P15 增长。

同步 M4 §3.23 五个 P5 行：v0-design §3.3/3.4/6.2、M3-design §3.18、差异清单逐列/列表名称/行为；M2/M3/P1–P4 handoff 对应项登记处理结果。M4仍进行中；保留 P10b/P11/P8/M8 未完成交接。

## 3. 局部裁定

以下为既定架构内的实现选择；每项同时记录理由与判断错误时的代价，不把静态预检当运行证明。

| 裁定 | 结论 | 理由 | 错了的代价与验证 |
|---|---|---|---|
| 方法调整 | 采用职责/签名/文件/测试/命令驱动的紧凑计划，逐 Task TDD、实际变异、独立评审，替代完整预制原型及逐字重放 | 02-method §3 明确允许调整原型、重放及 plan 写法；不省去性质证伪、真实记录和完成门 | 接缝错误可能延后暴露；每 Task 保留红/绿/变异/恢复/评审证据，未跑原型不声称跑过 |
| 任务切分 | 原15项按既有D2授权拆成16个验收任务，sort_order单列T8；T5、T12与T13按规模各分a/b两个实施单元 | 避免列表HTTP与写语义挤在同一Task | 可能产生跨Task遗漏；§4逐项映射及签名核对，不缩范围 |
| caller与target | 使用ListingInput与ListRead的独立TargetMemberID，SQL relation_user_id只来自授权后路径目标 | A读B主页必须筛B关系，同时以A判可见性 | 混用会返回A的条目或泄漏B权限；T10的A≠B三关系+OwnOnly实际SQL反例 |
| 普通/历史目录 | 新增Visible，保留Historical，状态/标签批读、负责人行取ID再读PublicProfiles | 两种读的软删除/归档语义不同；历史负责人仍需名字 | 误共用会漏项/泄漏；T4/T11及P4历史回归 |
| HTTP边界 | 标准ErrorHandlerFunc窄映射重复参数，StrictMiddlewareFunc保存真实query；绑定顺序不变 | 保留M2契约及生成器，不伪造输入、不建旁路router | 可能误改旧API或坏path状态；T9矩阵与原query不变断言 |
| 组格续页 | sub_group_by存在时group_key须配sub_group_key，cursor缺格键优先报cursor/not_allowed | 明确只支持格续页，原筛选不被组键替代 | 调用者组合错误得到422；T3/T7/T12覆盖全部字段与组合 |
| SQL数值键 | 三静态SQL、窄键CTE方案不变；数值转换须证明时间/计数/float8排序全序一致 | CASE不能隐式把精确整数降为float8，简单float8::numeric也不能未经证据断言无损 | 邻接键可能合并导致重复/遗漏；T3/T5/T12精度边界与相邻可表示sort_order真实SQL证伪 |
| 偏好端口 | project声明typed app小值端口，issue提供纯校验，bootstrap转换；每列set标志整体替换 | 词汇单一来源，跨模块边界不导入domain，避免整行覆盖 | 漏值检查或并发丢更新；T13形状/词汇/交错与变异 |
| 完整分组验证 | 项目7种、个人5种以及全部合法subgroup字段均须真实SQL/total/续页通过 | 词汇接受不证明查询分支实现 | created_by_id/project_id/date-none可能仅名义支持；T12完整组合矩阵 |
| 并发完成门 | 新增快照、偏好并发及sort_order复用锁路径的相应交错全部-race -count=5 | 单次成功不足以证实事务ctx、锁后重判与无丢更新 | 偶发泄漏、死锁、丢更新；T8/T12/T13/T14强制记录匹配测试和五遍结果 |

## 4. 逐项验收

T1–T15已完成；T5a/T5b均独立Approved，T5a的事务测试Minor已在T5b关闭。T12a及T12b均经独立评审，T12b排序定义问题经修复及范围复审关闭；T13a迁移/持久化/纯校验和T13b接线/API/并发均已独立Approved，父T13已验收。T14矩阵/读边界/写隔离/完整回归与T15八接口故事/四变异/完整142条e2e均已独立Approved；T16文档/差异/输入交接、内部整分支审查与统一修复复审已完成；2026-10-26 负责人授权后，P5 已以 `--no-ff` 合并为 `ee26db29c8a55cbb034a2da38c794c7f1af88e4e`；[main CI 40257678959](https://github.com/open-nerve/NerveProject/actions/runs/40257678959) 的 server/web/e2e 全绿；6,517 个证据文件已归档并逐个 SHA-256 核对，P5 本地/远端分支及 worktree 已清理。T16 集成门关闭，M4 总体与后续交接仍 open。 勾选依据真实实施与独立评审，详见 [阶段评审](../reviews/P5-issue-list-review.md)。

| 完成 | Task | 完成线 |
|---|---|---|
| [x] | T1 | 真实池多连接参数覆盖URL且UTC读取不退化 |
| [x] | T2 | 页面词汇、19排序与未来字段拒绝的纯表测试/变异 |
| [x] | T3 | 原始query保留全部日期条件、键归一化与canonical游标；共享固定顺序样例（真实SQL全序/日期AND在T5/T12验证） |
| [x] | T4 | 普通目录/Historical区别、全量状态标签/历史负责人姓名与同ctx |
| [x] | T5 | 平面SQL过滤先于limit、数组防御、精确键、空页补total、OwnOnly |
| [x] | T6 | 分组total/每组limit/完整数组、A∩C续页与none |
| [x] | T7 | 泳道总体/组/泳道/格去重计数、格续页与双多值反例 |
| [x] | T8 | sort_order PATCH、state动态/完成时间、skip true/false互斥及锁路径交错五遍race |
| [x] | T9 | 两类错误映射边界、项目活/归档列表权限、三封套与闭合契约 |
| [x] | T10 | A≠B三关系实际SQL、target存在语义、caller OwnOnly与个人主页权限 |
| [x] | T11 | P3关联替换、归档源同项目例外、父项/计数及Historical不退化 |
| [x] | T12 | 19排序、项目7/个人5全部group及全部合法subgroup实际SQL/total/续页；快照-race -count=5；10k/100k最终SQL实测 |
| [x] | T13 | 00026三列默认/CHECK/Down-Up、三偏好闭合校验/整字段替换/用户隔离；并发-race -count=5 |
| [x] | T14 | 矩阵、运行角色、双向契约、全部新增交错强制race五遍、旧功能回归和全量门禁 |
| [x] | T15 | I3–I7/I19/I20 API故事、I15增长、全部API前置及完整e2e |
| [x] | T16 | 文档/差异/交接、实际证据、整分支终审意见处理、分支/main CI及既定合并清理 |

## 5. 后续 Phase

| Phase | 留下的条目与交付接口 |
|---|---|
| P6 | 事件与自动归档；沿已有写锁与审计语义，P5不提前发布事件 |
| P7/P8 | 链接数据P7；link_count字段和两排序码P8；P8搜索复用列表词汇/普通可见目录，草稿另交付 |
| P9 | 清理/异步删除与规模成本；不因P5新增三偏好列建立新删除图 |
| P10b | api/testdata/issue-order.json交生产前端码点比较器与vitest同文件消费；旧/user-properties/列表筛选store/workItemFilters迁移，相关until:M4/P11例外保留到迁移 |
| P11 | 详情store/service换型、JSONContent/comment_json/is_bot等清理与详情分页数据；P5不关闭详情范围 |
| P12/P13 | 列表/看板/日历与I3–I7页面，项目偏好与个人主页页面、浏览器本地归档/个人设置 |
| P14/P15 | 详情与协作页面故事增长，P5接口故事不冒充页面完成 |
| M5–M7 | 附件、迭代、模块等字段/词汇随所属数据加入，不预埋未来枚举 |
| M8 | 超过v0约20k规模、跨项目扫描、默认排序索引特化与更大规模方案按实测决定 |

### P8接收回执（2026-10-27）

保留P5当时已验收19码，T13新增±link_count成当前21码，不倒改历史。全部Issue/Detail required非负int64、0显式；只数活links，不看作者/metadata，不加filter或派生列；三SQL同形、整数/numeric游标和独立child/link大数探针通过。T10–T12词汇/日期/Visible/OwnOnly/快照、T14 I4非零双向limit2/I18、T15 P5四项4/4已有证据。P10b生产比较器/共享fixture与P11–P15页面仍开放，跨请求计数变化不承诺冻结。 实际命令/计数/变异/检查点见[P8 review §8](../reviews/P8-drafts-search-review.md#8-实施验收与首次整审2026-10-27)。产品HEAD `10f44790`、首次整审 `166b9efd..10f44790`；文档独评/最终准确HEAD CI按后续收据追踪。未合并，不改变本页历史验收与集成事实，输入handoff总体仍open。

## 6. 风险

| 风险 | 影响与控制 |
|---|---|
| actor/target混用、目录OwnOnly遗漏 | 返回错误成员条目或泄漏；独立字段、同快照存在验证、A≠B SQL与权限矩阵 |
| 目录与SQL不在同一快照 | 新状态/退出成员导致漏项、total漂移；ReadOnlySnapshot与屏障交错-race -count=5 |
| 数值转换/同键游标方向不一致 | 分页重复遗漏；精度边界/相邻sort_order/逆UUID，不能仅靠原S1小seed |
| 多值展开层错误 | 整体/组/泳道/格total错、续页换筛选；完整字段与双多值矩阵 |
| CASE排序与跨项目扫描成本 | 已测最终32轮10k/100k，仍全排序/有temp与成员相关扫描，名字/计数跨请求可漂移；保留20k目标，M8负责更大目录/关系度/跨项目与索引特化 |
| 偏好整行覆盖、sort_order漏互斥 | 丢并发更新或改变描述免活动语义；每列set标志与true/false反例、交错五遍race |
| 局部HTTP映射放大 | 旧端点/非法path错误码退化；route+typed-error+声明参数+重复值四重限定及契约测试 |

## 7. 前置交接与关闭条件

| 来源 | P5承接与关闭条件 | 仍开放 |
|---|---|---|
| P1a/P1b | sort_order PATCH全链和列表排序、已有写锁/闸门；T8+T12+I6验证后登记处理结果 | 后续事件/页面 |
| P2 | 日期筛选、归档列表、OwnOnly及权限一致；T3/T9/T10/T14 | 后续搜索/页面 |
| P3 | VisibleProjects替换关联逐项目判定，保持归档源同项目例外、父项隐藏/子项数；T4/T11/T14 | P6事件、P9删除、P14页面 |
| P4 | 直接消费PageSize/ReadOnlySnapshot；保留Historical；扩PATCH不破坏skip_activity；T8/T12历史回归 | P15页面与后续清理 |
| [M2收尾交接](../handoffs/M2-closeout.md) | 日期/页大小/游标与闭合schema沿现状；无新开放map schema；锁下时钟与真实验证保留 | 其他Phase条目不提前关闭 |
| [M3收尾交接](../handoffs/M3-closeout.md) | 三偏好列及端口、成员退出恢复保留设置、个人主页规则、列表目录；T10/T13/T14 | P10b旧筛选store、后续页面 |

当前文档收口已逐节关联 review §2 的真实测试/变异/性能及审查证据。2026-10-26 负责人授权后，P5 已以 `--no-ff` 合并为 `ee26db29c8a55cbb034a2da38c794c7f1af88e4e`；[main CI 40257678959](https://github.com/open-nerve/NerveProject/actions/runs/40257678959) 的 server/web/e2e 全绿；6,517 个证据文件已归档并逐个 SHA-256 核对，P5 本地/远端分支及 worktree 已清理。T16 集成门关闭，M4 总体与后续交接仍 open。

## 附录 A. 起步时的证据边界（历史）

起步提交 `8f4b170d` 时仅有静态预检与文档修订；后续实施证据见第4节和阶段评审。四项 Local/Medium 已纳入本版：actor/target独立输入；全部分组字段实际SQL覆盖；所有新增交错强制race五遍；七节规格与方法/证据边界闭合。两项Low也已修正：IssueFields沿app/rows.go扩展；SQL NULL防御与adapter生产参数规整分别验证。来源/裁定与复审结果在 [P5 review](../reviews/P5-issue-list-review.md) 逐条登记；2026-10-25独立聚焦复核已批准本起步计划，产品验收仍未完成。

起步时没有运行完整预制原型，没有P5 TDD红绿、变异、SQL精度证明、S1复测、性能结果、生成物或P5 CI通过证据。已知真实通过仅是表头所列P4合并main基线CI；不得将其当作P5验收。

### M4/P6 接收记录

列表阶段的描述/排序及archive_in消费边界已按P6规格实施，提交、真实测试/变异与独立评审见 [P6 review §2](../reviews/P6-issue-events-review.md#2-验收标准逐条核对)及[§5](../reviews/P6-issue-events-review.md#5-最终实现与-plan-的差异)。P7/P9/P15和M4整体交接仍按原文开放。
