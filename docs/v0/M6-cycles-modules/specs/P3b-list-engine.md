# M6 / P3b：列表引擎的迭代与模块、迭代与模块页的显示设置

> 状态：待预检。基线 main `2648a73940f053940b3f321f241b1a25eaac8209`（P3a 的 noff，main CI 42487991301 首次通过）。设计见 [M6 设计](../M6-design.md) §2 的 G15、§3.7、§4、§5、§6、§7；负责人 2026-11-07 的决定 A/B/C 见设计 §15；P3a 见 [P3a spec](P3a-issue-membership.md)。加锁之路沿用 M4（[硬规则 §4](../../claude_to_gpt/03-rules-and-pitfalls.md)）与 [P1 spec](P1-cycles-backend.md) §3。现状调研见本阶段台账 `research/backend.md`、`research/web.md`（核对过 P3a 前的 `research/list-engine-P3a.md`，ignored，随阶段归档）。

## 1. 目标与边界

P3 按设计 §12 拆为两段，P3a 已合入。P3b 交付设计 §12 P3b 一行：

- **列表引擎**（M4 交接 M6-02 的引擎部分）：工作项列表按迭代、模块筛选（含“无”）与分组；按迭代名、模块名排序（两向）；显示属性“迭代”“模块”。前后端共用同一份排序夹具 `api/testdata/issue-order.json`。
- **项目偏好**：存储的 CHECK 认得新词（新迁移，先例 `00038`）。
- **迭代页与模块页的显示设置**：新表 `cycle_user_properties`、`module_user_properties`（`planning`），接口 `GET/PATCH /api/v0/me/cycles/{cycle_id}/preferences` 与 `/me/modules/{module_id}/preferences`，动作 `cycle.preferences`、`module.preferences`。

覆盖 G15 的接口与数据库证据（页面在 P5–P7），以及 G17 中与列表、偏好相关的矩阵行。

**不做：**
- 网页的筛选、分组、排序、显示属性的界面与迭代/模块目录的取数（P7；P5/P6 建迭代与模块的 store）。
- 迭代/模块页本身（P5/P6）。
- 进度、冻结、转移（P4）。
- 工作区级的迭代/模块视图（M7）。

网页只做五类改动：
1. 重新生成的类型；
2. 共用夹具迫使的排序比较器；
3. 新类型迫使的编译修正；
4. 改名的显示设置类型；
5. 新属性在手写字面量里的补齐（§2.6）。

## 2. 交付合同

### 2.1 列表引擎的词汇

| 种类 | 新词 | 范围 | 语义 |
|---|---|---|---|
| 筛选 | `cycle_id`、`module_id`，值为 uuid 或 `none` | 只限项目列表；“我的工作项”列表回答 422 `not_allowed`（同 `assignee_id`） | 同 `label_id`：一个字段内的多个值取并（`none,<id>` 即“没有迭代，或在该迭代”），不同字段取交；`none` 指没有未删除的归属。不检查 id 是否存在（M4 的规则，`listvocab.go` 的说明） |
| 分组、子分组 | `cycle_id`、`module_id` | 只限项目列表 | `cycle_id` 单值，没有归属的进 `none` 组；`module_id` 多值，同 `label_id`：一个工作项在它的每个模块组各出现一次，没有模块的进 `none` 组；组的总数按组计算，外层 `total` 是去重后的筛选结果。组的次序沿用 M4：`none` 最后，其余按组键的文本（uuid 次序），只回答非空组 |
| 排序 | `cycle_name`、`-cycle_name`、`module_name`、`-module_name` | 两种列表都可用（同 `state_name`、`label_name`，R3） | 键为名字经 `NameOrderKey`；迭代取当前未删除归属的迭代名，模块取未删除归属中最小的键（同标签）；没有的排在最后（两向都如此），同键按工作项 id 随方向排；已归档的迭代、模块照常参与（设计 §3.6）。游标为 `[order, key, id]` 的文本键 |
| 显示属性 | `cycle`、`modules` | 项目、迭代页、模块页的偏好 | 只影响展示；默认 `true`（Plane 前端的实际默认，`packages/utils/src/work-item/base.ts:310-311`） |

- **分组码沿用“分组码等于筛选名”的约定**（设计 §7）。
- **按名字排序的目录**：issue 的语句不能连 `cycles`、`modules`（`TestSQLCSchemaScope`），目录照 M4 §11.5 以数组传入静态查询（R2）。

### 2.2 文件与职责

| 路径 | 职责 |
|---|---|
| `server/internal/modules/issue/domain/` | `listvocab.go`：§2.1 的筛选、值语法、分组、分组键、排序。`list_query.go`：`ListFilters` 加 `Cycles`、`Modules IDFilter`，`canonicalizeListGroupKey` 认两个分组字段。`list_filters.go`：`appendID` 由两路 `else` 改为按字段的 `switch`（P3a 调研发现的陷阱：`none` 会被记到负责人）。`list_cursor_key.go`：可空与文本键两处列表 |
| `server/internal/modules/issue/app/` | `planning_ports.go`：`PlanningDirectory`（R2），并入 `PlanningPorts`。`list_ports.go`：`ListDirectories` 加 `Cycles`、`Modules []ListName`。`listing.go`：只在 `±cycle_name`/`±module_name` 时、在读行的同一个快照里取目录（先例：附件计数，`listing.go:90-95`）。`list_settings.go`：`Validate` 改收共用的显示设置值，显示属性由类型保证恰好 14 个键，删除手写的属性清单（R5） |
| `server/internal/modules/issue/adapter/` | `postgres/list_parameters.go`：成对的数组与 `Filter{Cycles,Modules}`、`{Cycles,Modules}None`。`groupParameters` 与子分组的同名函数逐字段复制，每个新参数改三处。三条列表语句（平铺、分组、子分组），各加：目录 CTE `cycle_names`/`module_names`，不叫 `cycles`/`modules` 以免遮住表名；两个筛选；两个名字键；分组与子分组的 `CASE`。三条语句的列保持逐一相同。`postgres/list_cursor.go:47` 的文本键。`http/list_query.go` 的 `listSingleQuery`：项目列表重复给出两个新参数时回答 422 |
| `server/internal/modules/planning/` | `app`：`PlanningDirectory` 的实现（按项目 id 数组读未删除的迭代、模块的 id 与名字，含已归档，`ORDER BY id`）；偏好的读与改、`Locks` 的共享锁之路（R6）；列表设置端口（消费方声明，R5）。`domain`：两种偏好、默认值、动作 `cycle.preferences`、`module.preferences`（并入 `Actions()`；`module.preferences` 并入 `ModuleAction`，`TestModuleAction` 要求所有 `module.` 动作都在其中，偏好之路不看开关）。`adapter/postgres`：两张表的读、upsert、删迭代/模块时的软删、删除任务两步、清理两步。`adapter/http`：四个处理器。`module.go`：接线、清理步骤 |
| `server/internal/modules/project/` | `domain`：`Filters` 加 `CycleID`、`ModuleID`；显示设置改用共用的类型（R5）。`app/list_settings.go`：转换随之。`adapter/http`、`adapter/postgres` 的映射随之。`PlanningProjects` 加 `Share`（项目行 `FOR SHARE`，R6）与其语句 |
| `server/internal/modules/access/domain/rules.go` 与测试 | 两个新动作：项目级，管理员、成员、访客 |
| `server/internal/shared/` | 新文件：三页共用的显示设置值 `ListDisplayFilters`、`ListCalendar`、`ListDisplayProperties`（14 个布尔）与默认值，以及十个共同的筛选键 `ListFilters`（R5） |
| `server/internal/bootstrap/` | 接线：`issuePlanning` 加目录；`planningProjects.Share`；偏好的列表设置端口。删除 `projectListSettings` 的转换（R5）。组合测试、矩阵、守卫与夹具（§2.5） |
| `server/migrations/sql/` | `00045_project_project_user_properties_planning_display.sql`（R4）、`00046_planning_cycle_user_properties.sql`、`00047_planning_module_user_properties.sql`（§2.4） |
| `server/migrations/` 测试、`server/sqlc.yaml`、`deploy/runtime-grants.sql` | 迁移总数 44→47、清理表数 30→32（`trigram_schema_test`、`purge_indexes_test`、`purge_catalog_test`）；`planningNames` 加约束与索引名；`00045` 的上下行测试（照 `attachment_display_test.go`）；sqlc 的 project 条目加 `00045`、planning 条目加 `00046`、`00047`；运行角色对两张新表的四种 DML |
| `api/common.yaml` | 新增 `IssueListCalendar`、`IssueListDisplayFilters`、`IssueListDisplayProperties`。由 `project.yaml` 的 `ProjectListCalendar`、`ProjectDisplayFilters`、`ProjectDisplayProperties` 搬来改名，旧名删除。显示属性加 `cycle`、`modules` 两个必填布尔（R5） |
| `api/modules/issue.yaml` | `listIssues` 的查询参数 `cycle_id`、`module_id`。`IssueListOrder` 加四个词。`IssueListGroupField` 加两个词（两条路由共用，“我的工作项”在运行时拒绝，同 `project_id` 的先例） |
| `api/modules/project.yaml` | `ProjectIssueFilters` 加可省略的 `cycle_id`、`module_id`；偏好的两个 schema 改引 `common.yaml` |
| `api/modules/planning.yaml` | 四个操作（§2.3）；`CycleIssueFilters`（11 键，无 `cycle_id`）、`ModuleIssueFilters`（11 键，无 `module_id`）、`CyclePreferences(Update)`、`ModulePreferences(Update)` |
| `api/testdata/issue-order.json` | `cycle_name`、`module_name` 两个场景（R7）；四个消费者的个数随之改：Go 的 `domain/list_order_test` 11→13，`bootstrap/issue_list_orders_test` 21→25，适配器的 `list_order_test` 的播种分支，网页的 `issue-order.test.ts` 11→13 |
| `web/` | §2.6 |
| `e2e/` | §2.5、§2.6 |
| `docs/` | P3a 回填（T1）；`plane-diff.md`；输入交接的“处理结果（M6/P3b）”；设计 §6、§14 |

### 2.3 偏好的接口与错误

| 操作 | 路径 | `x-problem-codes` |
|---|---|---|
| `getCyclePreferences` | `GET /api/v0/me/cycles/{cycle_id}/preferences` | `planning.cycle_not_found`、`forbidden` |
| `updateCyclePreferences` | `PATCH` 同上 | `validation_failed`、`planning.cycle_not_found`、`forbidden`、`server_busy` |
| `getModulePreferences`、`updateModulePreferences` | `/api/v0/me/modules/{module_id}/preferences` | 同上，`planning.module_not_found` |

- **回答**：`{filters, display_filters, display_properties}`。没有存过时回答默认值，读不写入（同项目偏好）。
- **PATCH**：三个顶层字段都可省略；给出的整字段替换，没给的保持；第一次修改存下默认值加这次修改。`filters` 为空对象即不筛选，空数组照存。
- **只看自己的行**：每个调用者只读写自己的一行（部分唯一 `(cycle_id, user_id) WHERE deleted_at IS NULL`）。
- **拒绝次序**：
  1. 结构 400：多余的键，包括迭代页的 `cycle_id`、模块页的 `module_id`（R8）；
  2. 词的 422 `validation_failed`：字段路径同项目偏好，如 `filters.state_id[0]`、`display_filters.group_by`；
  3. 404：项目不存在或看不见、迭代/模块不存在或已删除；
  4. 403：不是项目的有效成员。
- **不拒绝的情形**：只看自己的访客（OwnOnly）、已归档的项目、已归档的迭代/模块、开关关闭，都照常读写（R6）。
- **错误码**：`server_busy` 是 planning 写事务的锁超时（`busyTx`，同 planning 其他写）。不新增错误码，前端无新文案。每个（操作，码）在 planning 的 apitest 有一个回答它的处理器测试。

### 2.4 表

`cycle_user_properties`、`module_user_properties` 照 `project_user_properties`（`00012`、`00026`、`00038`）去掉 `preferences`、`sort_order`。

| 表 | 列 | 约束与索引 |
|---|---|---|
| `cycle_user_properties` | `id`、`workspace_id`、`project_id`、`cycle_id`（`REFERENCES cycles ON DELETE CASCADE`）、`user_id`（`REFERENCES users ON DELETE CASCADE`）、`filters`、`display_filters`、`display_properties`、审计四列、`deleted_at` | 部分唯一 `(cycle_id, user_id) WHERE deleted_at IS NULL`；`workspace_id`、`project_id`、`cycle_id`、`user_id` 各一个无条件索引（审计列照 `00041` 的写法）；`<表>_deleted_at_idx … WHERE deleted_at IS NOT NULL`（`purge_indexes_test` 的要求） |
| `module_user_properties` | 同上，`module_id REFERENCES modules ON DELETE CASCADE` | 同上 |

**三个 jsonb 列的默认值与 CHECK：**

| 列 | 默认值 | CHECK |
|---|---|---|
| `filters` | `{}` | 键属于本页允许的 11 个：项目的 12 个去掉本页自己的键，值为字符串数组 |
| `display_filters` | 同 `project_user_properties` | 同 `project_user_properties` |
| `display_properties` | 14 个键都为 `true` | 恰好 14 个键，都是布尔 |

`00045` 把项目偏好的两个 CHECK 扩为 12 个筛选键与 14 个显示属性（R4）。

### 2.5 删除、清理与守卫

- **删迭代/删模块**：在删除事务里、同一时刻、同一写者软删它未删除的偏好行（同删模块时的成员与链接，`delete_module.go`）。`CycleDeleter` 加一个方法，`ModuleDeleter` 再加一个。
- **删项目、工作区**：
  - `planning.delete_rows` 的 `DeletionRowSteps` 加 `cycle_user_properties`、`module_user_properties`，父先于子，接在 `module_links` 之后。
  - `ProjectRows` 加两个方法，剩余探查的 `CASE` 加两支。
  - 沿用请求的 At/By，按批有界，可续跑。
- **成员离开或被移出**：偏好行保留（Plane 同；项目偏好同，`EndMember` 只清 `is_active`）。在此期间读写被规则的 403 挡住，重新加入后行还在。
- **60 天清理**：`planning` 的清理步骤子先父：`module_user_properties`、`module_links`、`module_members`、`modules`、`cycle_user_properties`、`cycles`。每条语句有界，符合 `TestPurgeScale` 的三种形状（M5-M6-03）。
- **守卫与夹具**：
  - bootstrap 的 `projectPlanningTable` 与 e2e 的 `asynchronousDeletionTable` 认得两张表；
  - `seedCycles`、`seedModules`（`planning_deletion_fixture_test.go`）以 SQL 各种一行偏好，覆盖项目、工作区、迟到删除与清理的守卫；
  - `purgeScaleTables`、`seedPurgeSparseRows` 加两张表：这里列出的表必须有过期行（`purge_scale_fixture_test.go:106-109`）；
  - `purge_runtime_test` 的 planning 组加两张表；
  - e2e 的 `furnishPlanningRows` 以管理员的令牌 PATCH 两种偏好，让删除故事有行可断言。
- **用户删除**：外键 `CASCADE`；v0 不物理删除账户。

### 2.6 网页与 e2e 的随动改动

**网页：**
1. **重新生成** `api-client`。
2. **排序比较器**：`lib/issue-order.ts` 的 `IssueCatalog` 加 `cycles`、`modules` 两个 `ReadonlyMap`（值为生成的 `Cycle`、`Module`），`issueSortKey` 加两种排序（R7）。`issue-order.test.ts` 的 `fixtureRow` 两个分支，场景数 13。测试用的 `cycleOf`、`moduleOf` 放在 `store/project/fake-projects.ts`，照 `labelOf`。
3. **目录字面量**：`lib/paged-groups.test.ts:12`、`store/issue/fake-issues.ts:110` 补两个空 `Map`。生产的 `store/issue/helpers/issue-list-binding.ts:80-84` 同样给两个空 `Map`，并在 `prepareCatalog` 里照标签的写法加检查：排序为 `±cycle_name`/`±module_name` 时，工作项的迭代/模块不在目录里就抛出 `Required issue directory is unavailable`（R9）。
4. **分组字段的编译修正**：`lib/issue-membership.ts` 的 `groupKeys` 加 `module_id`（多值，同 `label_id`），`lib/issue-filters.ts` 的 `issueFieldForGroup` 加 `module_id → module_ids`。`cycle_id` 落在既有的 `default`，行为正确。
5. **显示设置**：
   - 改名：`ProjectDisplayFilters` → `IssueListDisplayFilters`，`store/issue/project/filter.store.ts:8`。
   - 补两个键：`store/project/fake-projects.ts:81-94` 与 `components/issues/issue-layouts/roots/project-lifecycle.test.tsx:92-105` 的显示属性字面量补 `cycle`、`modules`。
   - 保存路径不用改：切换显示属性时，网页回传服务端上次给的整个对象（`filter.store.ts:249-250`），新键随之往返，不会 400。

**e2e：**
- `stories/issue/i4-list-order-calendar.spec.ts`：四种排序，`satisfies Record<IssueListOrder,…>` 迫使与枚举同一提交；打开两个开关并经接口加入迭代与模块；排序个数 23→27。
- `stories/issue/i5-list-preferences.spec.ts`：显示属性补两个键；拒绝词由 `group_by:"cycle_id"` 改为仍被拒的 `"project_id"`。
- `stories/project/p8-project-preferences.spec.ts:40-54`：默认值补两个键。
- 新的 G15 接口与数据库故事放 `stories/planning/`。

## 3. 局部裁定

- **R1 筛选与分组只限项目列表**：迭代与模块按项目存在，“我的工作项”跨项目，没有可选的迭代目录。Plane 的个人工作项页也没有这两种筛选与分组（`plane/packages/constants/src/issue/filter.ts:113-121`）。工作区级视图的迭代/模块筛选属 M7。
- **R2 目录只为按名字排序**：
  - **端口**：`issue/app` 声明 `PlanningDirectory`，有两个方法 `CycleNames(ctx, projectIDs)`、`ModuleNames(ctx, projectIDs)`，回答 id 与名字，包括已归档的，不包括已删除的；名字的键在 Go 里用 `NameOrderKey` 算（同状态与标签）。实现在 `planning/app`，由 `planning.Provide` 交出，`bootstrap/ports.go` 转换。
  - **取数时机**：只在 `±cycle_name` 时取迭代、`±module_name` 时取模块，在读行的同一个只读快照里（先例：附件计数），其余排序不多一条语句。
  - **用途收窄**：分组头由网页按自己的目录给出，筛选值只做语法检查（M4 的规则），都不需要目录。设计 §6 的 `PlanningDirectory` 一行据此收窄。
  - **跨项目**：“我的工作项”的目录覆盖调用者全部可见项目，同状态、标签、负责人的现行做法。这些项目都是调用者的有效成员项目，`cycle.read`/`module.read` 对访客开放，不泄露名字；开关关闭的项目照样取，读取不受开关约束（设计 §3.8）。
- **R3 排序不分范围**：`ListOrderAllowed` 现在不带范围，OpenAPI 的 `IssueListOrder` 两条路由共用，`TestIssueListAllOrders` 覆盖“我的工作项”的形态。按 R2 的跨项目目录，四个新词在两种列表都可用，不加范围参数。Plane 的允许表只有模块名，没有迭代名，按迭代名排序时退回 `-created_at`；Nerve 两者都支持（设计 §15 第 16 条），登记差异。
- **R4 项目偏好的 CHECK**：`00045` 照 `00038`：
  - 删去并重建同名的 `project_user_properties_filters_check`，改为 12 个键。旧行仍然合法，不用 UPDATE。
  - 显示属性：删去 CHECK；UPDATE 现有行 `|| '{"cycle":true,"modules":true}'`；改默认值；重建同名的 CHECK，改为 14 个键。
  - Down：先从行里去掉 `cycle`、`modules` 与筛选的 `cycle_id`、`module_id` 两个键，再恢复旧的 CHECK 与默认值。否则旧 CHECK 会因存过的新键而建不起来。
  - `display_filters` 的 CHECK 只管形状，分组与排序的词本来就是自由字符串，不改。
- **R5 显示设置的形状三页共用**：
  - **为什么共用**：显示筛选、日历、显示属性在项目、迭代、模块三页完全相同。各存一份，14 个键的列表就要写三遍，容易不一致。照 P2 把链接检查放进 `internal/shared` 的先例（设计 §15 第 17 条），一份共用。
  - **契约**：放在 `common.yaml`，名字中性：`IssueListCalendar`、`IssueListDisplayFilters`、`IssueListDisplayProperties`。先例 `LinkMetadata`，issue 与 planning 共用。Go 生成为 `apigen.X`，TS 保留 schema 名。
  - **值类型**：放在 `internal/shared`，带 JSON 标签与默认值。项目的 `domain.DisplayFilters`、`DisplayCalendar`、`DisplayProperties` 与 `app.ListDisplayFilters`、`ListCalendar` 删除，改用共用类型。
  - **校验端口**：issue 的 `ListSettings.Validate(filters map[string][]string, display *shared.ListDisplayFilters, properties *shared.ListDisplayProperties)` 改收共用类型；显示属性的键由类型保证，issue 里手写的 12 个属性清单删除（它与项目里的转换清单是同一事实的两份）。项目与 planning 各自声明的 `IssueListSettings` 端口签名相同，`bootstrap` 直接交给 `issue.ListSettings`，删除 `projectListSettings` 的逐字段转换。
  - **筛选按页各自**：十个共同的键在 `shared.ListFilters`（嵌入），连同转为 `map[string][]string` 的方法。项目再加两个键，迭代页、模块页各再加对方的一个键。
  - **契约层面**：每页一个封闭的 OpenAPI schema，因为 `additionalProperties:false` 不能与组合同用。
  - **验证**：`bodyshape` 对请求体里引用 `common.yaml` 的 schema 是第一次（`LinkMetadata` 只在回答里）。`util.LoadSwagger` 会解析外部引用，在该 Task 内以 `make gen` 与 bodyshape 的测试核实。
- **R6 偏好的加锁之路**：
  - **GET**：用 planning 现有的读之路（`Reads.run`，一个只读快照），决定 `cycle.preferences`/`module.preferences`；没有写时的那些拒绝。
  - **PATCH**：`Locks` 新开一条共享锁之路：在一个事务里，先不加锁找到项目（`see` 经 `Facts`），再以 `Projects.Share` 把项目行加 `FOR SHARE` 并在锁下重读迭代/模块，决定动作后读时钟、upsert。
  - **不进项目写闸门，不拒绝写时的那三种情形**：OwnOnly 403、已归档项目 409、开关 409。这些是迭代/模块本身的写的规则（设计 §3.7、§3.8），个人显示设置不是这类写；项目偏好与草稿保存也是这样做的。
  - **为什么要锁**：`FOR SHARE` 与删迭代/模块、删项目的 `FOR NO KEY UPDATE`/`FOR UPDATE` 冲突，所以不会在一个正被删除的迭代下写入新行：先删则 PATCH 404，先写则行随删除软删。
  - **开关**：偏好之路不经 `turnedOff`。`module.preferences` 虽在 `ModuleAction` 里，也不受开关约束。
  - **重构**：`write` 与 `named` 的字段重复（`lock.go:78`、`decide.go:15`），改为 `write` 嵌入 `named`。改到 `lock.go` 时顺手完成，行为不变。
- **R7 夹具的两个场景**：行与期望的 id 数组沿用现有的名字场景。
  - **`cycle_name`**：同 `state_name` 的七行，名字分别为 `Alpha`、`ALPHA`、`ÄBC`、`äBC`、`～`、`😀`，另有一行没有迭代，`key:null`。没有迭代的那一行是真实的工作项，Go 要种它，不能跳过；迭代名不要求唯一，两个草稿迭代可以叫 `Alpha` 与 `ALPHA`。
  - **`module_name`**：同 `label_name` 的七行。工作项 07 与 08 共用同一个模块 `Alpha`；`äBC` 改为 `äBD`，因为模块名按 `lower(name)` 在项目内唯一，PG 的 `lower` 会把 Ä 折叠，这样 Go 只需按名字去重，不用像标签测试那样改名。
  - **共同点**：期望的 id 数组是写死的字面量，不由比较器算出。
- **R8 本页的键靠形状拒绝**：迭代页的筛选 schema 没有 `cycle_id`（封闭），bodyshape 回答 400；数据库的 CHECK 也不认它；planning 的筛选类型没有这个字段，它到不了 `Validate`。按本页的键分组（迭代页 `group_by:"cycle_id"`）接受，结果只有一组。Plane 的后端也不拦，只在本页界面上隐藏这个选项；P5/P6 照做。
- **R9 网页在 P7 之前的状态**：
  - **生产的目录**：两个空 `Map`，加上会大声失败的检查。网页的排序、分组、筛选清单（`filter.store.ts` 的 `orderValues`/`groupValues`、`issue-filters.ts`）在 P7 之前不收这些词，这条路走不到。P7 把空 `Map` 换成 P5/P6 的 store，并补上取数。
  - **服务端存的新词**：经接口存进项目偏好的 `group_by:"cycle_id"`、`order_by:"cycle_name"`，网页显示为不分组、`-created_at`，回传时原样保留。
  - **筛选的缺口**：经接口存的 `cycle_id`/`module_id` 筛选会被网页忽略（列表比存的筛选宽），下次在网页改筛选时被覆盖。这个缺口与表格加载行宽出两列的外观问题都留给 P7（§5）。
- **R10 Plane 差异**（登记进 `plane-diff.md`）：
  - **筛选**：`none,<id>` 取并（Plane 取交，结果为空）。
  - **排序**：按迭代名排序（R3）。
  - **偏好的读写**：
    - GET 不写入（Plane 用 `get_or_create`）；
    - PATCH 不要求先 GET（Plane 用 `objects.get`，没有行即 404）；
    - PATCH 校验词，回答 200（Plane 不校验，回答 201）；
    - 本页的键被拒（Plane 收下后由网页删去）；
    - 筛选是 Nerve 的格式（差异清单已有“筛选格式改变”一行）。
  - **相同的行为**：删迭代/模块时偏好随之软删、成员离开时保留、访客可用。

## 4. 验收标准

- [ ] A1 G15 筛选：
  - 在平铺、分组、子分组三种形态，按迭代、模块与 `none` 筛选；同一字段内的多个值取并，不同字段取交；
  - 已删除的归属不算，已归档的迭代/模块照常算；
  - “我的工作项”回答 422；项目列表里重复给出参数回答 422；
  - 访客与 OwnOnly 只看到自己能看见的工作项，组的总数也只算这些。
- [ ] A2 G15 分组：迭代单值、模块多值；`none` 组；组总数与去重后的外层 `total`；子分组；双向分页与组内续页；“我的工作项”回答 422。
- [ ] A3 G15 排序：
  - 两种列表、四种形态，覆盖四个词；
  - 没有的排在最后；同键按 id 随方向排；
  - 游标跨页稳定；目录与行在同一个快照里；
  - 其余排序不取 planning 的目录；
  - 同一份夹具驱动 Go 与网页的测试（场景数 13）。
- [ ] A4 项目偏好：12 个筛选键、14 个显示属性；`00045` 的上行、下行、再上行；旧行补上两个属性；默认值相同；网页与 e2e 的随动改动通过。
- [ ] A5 迭代/模块偏好：
  - 四个操作的回答与拒绝次序（§2.3）；
  - 本页的键 400；只读写自己的一行；
  - OwnOnly、已归档的项目/迭代/模块、开关关闭照常；
  - 权限矩阵覆盖四个操作；
  - 与删迭代/模块的两种次序交错各有测试。
- [ ] A6 删除与清理：
  - 删迭代/模块、删项目/工作区、60 天清理都覆盖两张表；
  - 运行角色可执行；`TestPurgeScale` 认得新语句；
  - 全部守卫覆盖两张表，且没有被放宽。
- [ ] A7 契约与文档：
  - 共用的显示设置 schema 在 `common.yaml`，bodyshape 照常；每个新声明的码在 planning 的 apitest 有回答；
  - P3a 的阶段记录 §7、spec A8 与设计 §14 已回填；
  - 差异清单与输入交接的“处理结果（M6/P3b）”已写；设计 §6 的目录一行已收窄。
- [ ] A8 七项本地门禁、deep-review 修复闭环、准确的分支 CI 与 noff 后的 main CI 全绿、归档清理。

## 5. 后续责任

**P7**（网页的列表一侧）：
- 筛选、分组、排序、显示属性的界面与词表（`filter.store.ts`、`issue-filters.ts`、`issue-filter-helper.store.ts`、常量与 `group-columns.tsx`）；
- 目录取数，换掉生产目录的空 `Map`；
- R9 的筛选缺口与表格加载行宽出两列。

**P5/P6**：迭代页与模块页接上本阶段的偏好接口，并在本页隐藏“按本页分组”（R8）。

**P4**：进度、冻结、转移。

**M7**：工作区级视图的迭代/模块筛选。

## 6. 风险

- **三条列表语句**：每个新参数要改三条语句、三处参数复制，三条语句的列须逐一相同。以 sqlc 的生成与行类型互转为守卫：一处不一致即编译失败。每个 Task 跑平铺、分组、子分组三种形态的适配器测试与 HTTP 测试。
- **共用夹具跨前后端**：加场景的同一个提交必须同时改 Go 的三个消费者与网页的比较器，否则一边的测试失败。e2e 的 `i4` 因 `satisfies Record<IssueListOrder,…>` 也要在同一个提交改。
- **共用显示设置的重构**：
  - 影响面：项目偏好的 Go 类型与 TS 类型名都变。
  - 守卫：先做“不改任何回答与存储”的重构，M4 的项目偏好测试只改类型名，断言不动；再加新词。
  - 不确定的一点：bodyshape 对外部引用的请求体能否正确解析，在重构的 Task 内先核实。若不行，停下上报，不绕开。
- **按目录驱动的守卫**：两张新表在建表的 Task 就触发多道守卫（清理表数、删除目录、清理规模必须有过期行）。Go 的夹具以 SQL 种行。e2e 的删除故事要等偏好接口落地、`furnishPlanningRows` 能经接口写行后才转绿，所以整段的七项检查点安排在列表引擎与项目偏好完成之后、建新表之前一次，收尾再一次。
