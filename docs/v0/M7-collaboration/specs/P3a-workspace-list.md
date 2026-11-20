# M7 / P3a：工作区跨项目列表

> 状态：spec 经预检 [re-88](../../../reviews/re-88-M7P3a预检.md) 修正后通过。基线 main `7dfaa8a1`（P2c 的 noff）。设计见 [M7 设计](../M7-design.md) §2（H13）、§3.7、§3.10、§4.1、§5、§9、§10、§15 D3/D4；M4 交接 [M7-03](../handoffs/M4-closeout.md)、M2 交接 §1、M6 交接 M6-M7-03。现状调研见本阶段台账 `research/p3a-facts.md`（ignored，随阶段归档；下文 DP-n 指它 §9 的决定点）。

## 1. 目标与边界（P3 的再划分）

设计 §12 的 P3a（`navigation` 模块与视图、工作区跨项目列表与规模测试）照原范围约 20–22 个 Task，超过约束 2 的约 16。实施前再划分（R1，DP-1 (b)）：

- **P3a（本 spec）**：工作区跨项目列表 `GET /workspaces/{slug}/issues`——issue 列表引擎的新范围 `WorkspaceList`，只动 issue 模块（与契约、网页客户端的一处）；规模测试。
- **P3b**：`navigation` 模块与 `issue_views`、两种视图的接口与权限、删除图与清理；H13 里“已存的工作区视图”那一半与 H14。工作区视图的词表校验用本段定下的 `WorkspaceList` 词。
- **P3c**：收藏与最近访问（原 P3b；设计 `:277` 等处提到的“P3b”随之改为 P3c）。

依赖方向：先有工作区列表的词，再有校验这些词的工作区视图；设计 §10 的头号风险（规模）先在小段里落地。三段共用关键词的 phase `M7/P3`。

P3a 不做：视图（P3b）；页面（P6b）；工作区标签目录（DP-6 (b)：M4 交接要它给已有的个人主页按标签分组，工作区列表的标签筛选是第二个使用者；设计 §5 的接口表里没有它，随 P6b 的页面定，写进设计 §12 的 P6b 一行）。

**P3a 合入后、P3b 之前 main 的样子**：工作区列表可用，网页还不调用（P6b 接）；没有 navigation 模块、没有 `issue_views` 表；用户看不到变化。

## 2. 交付合同

### 2.1 文件与职责

| 路径 | 职责 |
|---|---|
| `server/internal/modules/issue/domain/` | `ListScope` 加 `WorkspaceList`；它的筛选词、只平铺；`ParseListQuery` 认它 |
| `server/internal/modules/issue/app/` | `ListWorkspaceIssues`（照成员列表，去掉目标成员与 `relation`）；动作 `workspace_issue.list` |
| `server/internal/modules/access/` | `workspace_issue.list` 的规则（工作区：管理员、成员、访客） |
| `server/internal/modules/issue/adapter/http/` | 处理器；`list_query.go` 的**两张**按操作的清单：`withListQuery`（把原始查询交给用例，`:72-75`）与 `listSingleQuery`（单值参数，`:52-70`）都要加新操作——漏了前者，原始查询为空、答 200 却不带筛选 |
| `server/internal/bootstrap/` | 权限矩阵、删除窗口、空创建者；组合测试 H13；`TestIssueListScale` 的工作区场景 |
| `api/modules/issue.yaml`、`api/openapi.yaml` | `listWorkspaceIssues`、它的参数与码 |
| `web/packages/api-client/` | 新路由进 `csvListPaths`（`csvKeys` 已有全部词；`client.test.mjs` 按契约推导，不改期望） |
| `e2e/` | H13 的接口故事（`e2e/stories/collaboration/h13-workspace-list.spec.ts`） |
| `docs/` | 设计 §3.7、§12、§14；plane-diff；M2、M4、M6 交接的处理结果；硬规则；阶段记录 |

### 2.2 范围与词（DP-2、DP-3、DP-4、DP-5）

- 范围：调用者在工作区里是有效成员的全部**未归档**项目（issue 的 `VisibleProjects.Visible`，与成员列表、搜索、待处理数同源；不是成员的项目不在内，工作区管理员与公开项目也一样），逐项目执行只看自己（`OwnOnly`）；排除已删除、已归档的工作项与收集箱里的工作项（非 Triage 的状态目录）。
- 筛选词：`project_id`（只与范围相交）、`state_id`、`state_group`、`priority`、`label_id`、`assignee_id`、`created_by_id`、`mention_id`、`subscriber_id`、`start_date`、`target_date`、`sub_issue`；不收 `cycle_id`、`module_id`（§15 D4）、`archived`、`relation`。四个默认视图就是同一接口加 `assignee_id=me`、`created_by_id=me`、`subscriber_id=me`（设计写的 `created_by` 即 Nerve 的 `created_by_id`）。
- 只平铺：`group_by`、`sub_group_by`、`group_key`、`sub_group_key` 一律 422（已有的码；Plane 的工作区接口只平铺、页面只渲染表格；以后要分组是增量）。
- 次序：沿用不分范围的 `ListOrderAllowed`（它也是游标键的校验，成员列表已如此，M6 P3b R3；所以 `cycle_name`、`module_name` 的次序词照收）。
- 游标：`[order, key, id]` 的共享封套，与项目无关；游标只决定起点，筛选与可见集合每次由查询与快照重算。
- 一个只读的 REPEATABLE READ 快照（M4 交接 M7-03；与项目列表、成员列表相同）。

### 2.3 操作 `GET /workspaces/{slug}/issues`（`listWorkspaceIssues`）

- 授权新动作 `workspace_issue.list`（工作区：管理员、成员、访客；设计 §3.10 已列）。
- 拒绝次序：查询参数 400/422（未知参数、重复的非日期参数、不收的词）→ 工作区看不到 404 `workspace.not_found`。三种工作区角色都许可（`workspace_members.role` 只有这三种），所以没有 403。
- 回答 `IssueListPage` 的平铺形状 `{total, data, next_cursor}`，恒为平铺（契约说明写明），行与成员列表相同（看不到的父为 `null`、子数按项目只看自己）；越过末尾的游标答空页。
- `x-problem-codes`：`validation_failed`、`workspace.not_found`（处理器测试逐码回答）；不声明 `forbidden`。

### 2.4 规模（DP-7）

- 场景放在 `TestIssueListScale` 之下（`NERVE_LIST_SCALE=1` 才跑，CI 上是一次跳过，不新增顶层名）；路由与预期范围的选择从现在的一个整数扩为三种（项目、成员、工作区）。
- 夹具：判定的规模为**总量 20000 个工作项**，分在种子的多个项目里（另加一个多项目均匀分布的子种子）；请求的人在一部分行上是负责人与关注者，`assignee_id=me`、`subscriber_id=me` 的场景不为空（或改用别的账号的 id）。
- 配对的场景：工作区列表的默认页、`assignee_id=me`、`subscriber_id=me`、按标签名排序、末页游标，各与同一次运行里项目列表在同等总量上的对应场景配对。
- 度量：列表 SQL 的中位（照 M4 P5：1 次预热 + 5 次实测），留 EXPLAIN。
- **通过线**：工作区列表每个场景的中位不超过配对的项目列表场景的 1.5 倍，且不超过 M4 实测、负责人已接受的 100k 默认页代价 105.53 ms（M4 设计 `:639-642`）。
- 不达标先在本段查明；要靠加索引或特化 SQL 才达标（M4 把这类特化留给 M8 按数字决定），**停下上报**。
- 时间：M4 的规模跑用 `-timeout=40m`；加场景与子种子后重估，写进台账。

## 3. 局部裁定

- **R1 P3 再划分**：见 §1；设计 §12、§14 随之改，阶段记录向负责人说明。
- **R2 只平铺**；**R3 次序不分范围**；**R4 范围用 `Visible`**。
- **R5 工作区标签目录留给 P6b**（DP-6 (b)）：写进设计 §12 的 P6b 一行，M4 交接 M7-03 写处理结果。
- **R6 设计 §3.7 与 §3.10 的不一致**（工作区访客读已存的工作区视图）属 P3b，P3b 的 spec 定（推荐照 §3.10 与 H14：403）。
- **R7 没有 403**：三种工作区角色都许可，契约不声明 `forbidden`。

## 4. 验收标准

- 契约、生成物、处理器逐码；`make gen-check` 无差异；网页测试通过。
- 单测：解析（新范围的词、只平铺的四个词、`relation` 与 `cycle_id`/`module_id`/`archived` 的 422）、用例（拒绝次序、`Visible` 为空的空页）。
- 拒绝次序的交叉格：坏的查询 × 看不到的工作区 → 422；重复的单值参数 × 没有凭证 → 422（参数先于认证，照已有的列表）。
- 守卫：权限矩阵一行（工作区级的读）；删除窗口读 20 → 21（项目删除时工作区列表 200、不含被删项目的工作项）；空创建者的跳过与泄漏目录（空创建者的工作项对只看自己的访客不出现）。
- 组合测试（真实的 app，一个新的顶层名 `TestWorkspaceIssueList…`，其余为子测试）：H13 的列表一半——四个默认视图；逐项目只看自己（同一个人在一个项目里是只看自己的访客、在另一个里是成员）；不是成员的项目不出现（含工作区管理员、公开项目）；收集箱里的、已归档的工作项与已归档的项目、已离开的项目不出现；跨项目的游标翻页、`total` 与末页之后的空页；改过的游标换不出看不到的行；各筛选词；工作区管理员兼项目访客；只读快照（与项目列表、成员列表的快照测试同一种格）。
- 每个性质至少一个能让测试失败的变异（plan 的变异表）。
- 规模：§2.4 的场景、度量、通过线与证据（本机 `NERVE_LIST_SCALE=1`）。
- e2e：H13 的接口故事（PAT 与会话对等）。
- 测试时间：CI 每一路 Test 步的增量 ≤ 90 秒，两路各自记；新顶层名只有一个（按当时的轮流分在第 764 位、落第 1 路，换 7 个，检查点对照；合并 main 后分路改为按名字的哈希，加测试不再换路，这一条作废，见阶段记录 §4）。

## 5. 后续责任

- P3b：navigation 与视图；工作区视图的词表校验用 `WorkspaceList` 的词（新增 `WorkspaceListSettings`）；H13 的已存工作区视图、H14；M2 交接 §1 的视图列表与跨模块联表的结论；R6。
- P6b：工作区列表与工作区视图的页面、工作区标签目录（个人主页的按标签分组与工作区列表的标签筛选）、旧 `WorkspaceService` 四个方法的删除。

## 6. 风险

- 规模不达标：在本段内查明；要靠索引或特化 SQL 时停下上报。
- 换路：只加一个顶层名并预先算好。
