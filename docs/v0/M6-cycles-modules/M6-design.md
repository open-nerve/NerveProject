# M6：迭代与模块

> 状态：设计与 P1 预检通过（2026-11-07，[re-54](../../reviews/re-54-M6设计与P1预检.md)）；P1–P5 已完成（P5 拆为 P5a/P5b），P6 实施中（拆为 P6a/P6b，进度见 §14）。
> 起点：`main@ba3fd6e0b65e91e8bcf5ba32ad02ebff0b7bd939`（M5 最终 main CI 42251522430，三个作业成功）。
> 当前文档分支为`codex/m6-design`，根目录保持main。负责人2026-11-07指示“启动 M6，按照之前的推进节奏，一步步推进”，并在设计预检后决定三项（见 §15）：归属表放 `issue` 模块、草稿照 Plane 带迭代与模块、其余取舍按 §15 推进。常规设计、预检、实现、deep-review、`--no-ff`集成、CI和归档照 M5 的节奏连续推进；架构回归、跨模块协同调整或意外高风险发现，先提交已完成工作，再用大白话上报。

## 1. 目标、范围与输入

M6 让迭代（Sprint）和模块经真实接口工作：迭代的建立、日期与状态、进度、燃尽图、把未完成的工作项转到别的迭代；模块的建立、状态、负责人与成员、链接；两者的归档与恢复；工作项上的“迭代”“模块”字段及它们在列表引擎里的筛选、分组、排序和显示。行为照 Plane 一比一（[总体设计 §1.1](../v0-design.md#11-v0-保留的功能)），只修 Plane 的明显缺陷并登记差异。

不做：收藏、最近访问、跨项目的工作区视图与通知（M7，见 §11 的交接）；估算点数（M1 已删）；迭代/模块的文件（M5-M6-01：两者的描述是纯文本，不成为资产宿主）；首个投递任务的 CLI（M6 不加命令行，M5-04 原样转交）。

前置依据：[总体设计](../v0-design.md) §3、§5、§6、§7.2、§7.7、§8；[执行方法](../claude_to_gpt/02-method.md)；[硬规则](../claude_to_gpt/03-rules-and-pitfalls.md)；七份输入交接（§11）。Plane 的行为以只读的 `plane/`（提交 `02c19e134`，与前端迁入同一版）为准，表结构以 `tools/plane-schema/plane-v1.4.2-schema.sql` 为起点。

### 1.1 设计起点事实

- **后端**：没有 `planning` 模块，也没有任何迭代、模块的表；只有 `projects.cycle_view`、`module_view` 两个开关（`00010_project_projects.sql`），没有接口受它们约束。最新迁移是 `00038`。`Issue`、`IssueDetail` 没有 `cycle_id`、`module_ids`；列表引擎（`issue/domain/listvocab.go`）没有迭代、模块的筛选、分组、排序；M4 在设计里为它们留了位置（M4 设计 §3、`:597`、`:652`、`:1097`）。
- **前端**：迭代/模块的页面、store、旧服务都还在并挂着路由，约 164 个文件、1.46 万行；全部调用旧 Plane 地址，没有一处走 `/api/v0`。两个列表页从不取数（没有调用方调 `fetchAllCycles`、`fetchModules`），详情页只显示侧栏；侧栏、标签页、归档页的入口按开关显示，点进去就是这些坏页面。工作项一侧的迭代/模块（列表的 `EIssuesStoreType.CYCLE`/`MODULE`、属性下拉框、转移弹窗）已被 M4 删除，只剩动态的两种渲染（`issue-activity/activity/actions/{cycle,module}.tsx`）。
- **关键词守卫**：`analytics` 规则对 `core/services/cycle.service.ts` 与其测试的例外 `until: "M6"`，phase 一改为 `M6/P1` 即到期；`copy-hook` 的 5 个例外 `until: "M6/closeout"`。
- **Plane 的可沿用与不可沿用**：见 §3 每节的“Plane”一句；Plane 的授权漏洞（未覆写的 DRF 默认 PUT/GET、公共接口的跨项目 IDOR）、转移不写动态的崩溃、快照被每次转移覆盖、无人分组恒为 0、燃尽按 UTC 切天，都不照搬。

## 2. 用户故事

`G` 是 M6 的故事编号。每条有接口与数据库证据，页面故事另有浏览器证据；PAT 与浏览器登录对等。

| 编号 | 行为及有区分度的拒绝路径 |
|---|---|
| G1 | 成员建迭代：只给名字成草稿；给起止日期时两者都要、开始不晚于结束；与本项目其他未删除迭代的日期重叠被拒；访客、开关关闭时被拒 |
| G2 | 迭代的状态（草稿/未开始/进行中/已结束）按项目时区的“今天”由服务端推算；列表、详情、进行中的迭代卡片读同一个状态 |
| G3 | 改迭代的名字、说明、日期、排序；已结束的迭代只能改排序；已归档的迭代不能改；删迭代只限项目管理员或创建者 |
| G4 | 工作项加入、移出迭代（工作项详情、列表行、迭代页“添加已有”）；一个工作项最多在一个迭代，加入别的迭代就移过去；已结束、已归档的迭代不能加入；每次变化一条动态 |
| G5 | 迭代的进度：按状态组计数，进度 = 完成 ÷（总数 − 取消）；负责人、标签分布含“无负责人”“无标签”；燃尽图按项目时区逐日，未来为空 |
| G6 | 迭代结束后统计冻结：之后工作项改状态、移出，已结束迭代的数字不变；冻结只写一次 |
| G7 | 把已结束迭代里未完成（积压、未开始、进行中）的工作项转到草稿/未开始/进行中的迭代；已完成、已取消的留下；每个转走的工作项一条动态 |
| G8 | 归档已结束的迭代、恢复；归档目录；已归档的迭代不出现在下拉框和工作项的可选项里，但工作项上的值照样显示 |
| G9 | 建模块：名字项目内唯一（不分大小写）、状态六种、起止日期、负责人与成员只能是项目中不是访客的有效成员 |
| G10 | 模块链接的增改删：地址校验、同一模块内地址不重复、元数据由后台抓取（沿用 M4 工作项链接的抓取协议） |
| G11 | 工作项加入、移出模块（可同时属于多个）；替换整个集合；已归档的模块不能加入；每次变化一条动态 |
| G12 | 模块的进度、分布、燃尽图（开始到目标日期），口径同 G5 |
| G13 | 归档状态为已完成或已取消的模块、恢复；删除模块只限项目管理员或创建者 |
| G14 | 删除迭代/模块：工作项本身不受影响，归属随之移除并各记一条动态；删除工作项、项目、工作区时迭代/模块及归属立即不可见，60 天后有界清理 |
| G15 | 项目的工作项列表按迭代、模块筛选（含“无”）、分组，按模块名排序，显示属性“迭代”“模块”；迭代/模块页的工作项列表走同一引擎、四种布局、双向分页 |
| G16 | 自动归档跳过仍在未结束迭代或未到目标日期模块里的工作项 |
| G17 | 权限矩阵：管理员、成员、访客（含只看自己的访客）、工作区管理员兼访客、私有项目、撤权；访客读到的计数只算他能看见的工作项 |
| G18 | 项目的迭代/模块开关关闭时写入被拒、读取照常，再打开数据都在 |
| G19 | 页面：换账号、工作区、项目、迭代、模块，后退前进，迟到的回答不写进新页面；弹窗与下拉框的键盘、读屏、焦点；复制链接失败有提示 |
| G20 | 收尾：无旧迭代/模块服务与地址、到期的关键词例外、死成员与告警；完整对等验收 |

## 3. 业务模型与权限

### 3.1 迭代

- **字段**：名字（必填，≤255）、说明（纯文本）、开始与结束日期、负责人（建立者）、排序、冻结的统计（§3.5）、归档时刻。
- **日期是项目时区下的日历日**（`date`），两者都空或都有、开始 ≤ 结束。Plane 存时刻并把开始规整为当地 00:00:01、当天创建时为“现在”，结束为 23:59，状态判断又不真用项目时区；Nerve 存日、按项目时区判断，登记差异（我的决定）。
- **状态由服务端推算、不入库**：今天（项目时区）记为 T；有空日期为草稿，T < 开始为未开始，开始 ≤ T ≤ 结束为进行中，结束 < T 为已结束。接口返回 `status`，前端不自己推算。
- **日期重叠**：两者都有时，与本项目其他未删除（含已归档）迭代的闭区间相交即拒绝，409 `planning.cycle_dates_overlap`。Plane 只有一个前端先调的检查接口、创建与修改都不检查；Nerve 在写入时检查，删除 `date-check` 接口，登记差异（我的决定）。并发两次写入以项目范围的锁串行（§6.3）。
- **已归档的项目**：其下的迭代/模块不能写（409 `project.archived`，同 M4）。
- **可改的范围**：已结束的迭代只能改排序（Plane 的本意，Plane 因传错变量实际全可改，不照搬），其他字段 409 `planning.cycle_completed`；已归档的迭代任何修改 409 `planning.archived`。
- **负责人**：建立者，信息性；外键 `ON DELETE SET NULL`（Plane `CASCADE`，同 M3 对项目负责人的改法，登记差异）。

### 3.2 模块

- **字段**：名字（必填，≤255，项目内未删除者不分大小写唯一，409 `planning.module_name_taken`）、说明（纯文本；删除 Plane 的 `description_text`、`description_html` 遗留列，差异清单已登记）、开始与目标日期（`date`，开始 ≤ 目标）、状态（`backlog`、`planned`、`in-progress`、`paused`、`completed`、`cancelled`，默认 `planned`，数据库 CHECK）、负责人、成员、排序、归档时刻。
- **负责人与成员只有信息作用**，不影响权限。写入时须是项目中不是访客的有效成员（422 `lead_id`/`member_ids[i]` `not_allowed`）；Plane 的应用接口接受任何账户、公共接口过滤为项目成员，Nerve 统一校验（同 M3 的项目负责人规则，我的决定）。成员离开项目不回写模块。`member_ids` 整体替换。
- **链接**：标题（可空，≤255）、地址（http/https，补协议、校验同 M4 工作项链接）、元数据（后台抓取，沿用 M4 的协议：P2 把地址规整校验（原 `issue/domain/link.go`）与抓取适配器（原 `issue/adapter/crawler/`）移出 `issue`、两个模块共用，不复制一份，我的决定；落点受架构测试的导入规则约束（`domain`/`app` 只能导入标准库、本模块内层与 `internal/shared`，`platform` 不导入 `shared` 与业务模块），所以地址与标题检查和元数据类型在 `internal/shared/link.go`，抓取在 `platform/outbound/linkpreview`（返回自有的 `Metadata`），由 `bootstrap` 一个适配器转成 `shared.LinkMetadata` 交给两个模块；仍是一份代码，工作项链接的行为与测试不变，见 [P2 spec](specs/P2-modules-backend.md) R12）；同一模块内未删除链接的地址唯一（409）。

### 3.3 工作项的归属

- **一个工作项最多在一个迭代**（`cycle_issues` 部分唯一 `(issue_id)`，总体设计 §5.3 已定）；可属于多个模块。
- **只经工作项字段修改**（总体设计 §3.2、差异清单“迭代和模块归属”一行）：创建与 `PATCH /issues/{id}` 的 `cycle_id`（`null` 移出）与 `module_ids`（整体替换）。
- **草稿也带迭代与模块**（负责人2026-11-07决定，照 Plane）：`IssueDraftPayload` 加可省略的 `cycle_id`、`module_ids`，保存草稿只做结构检查（同其他九个字段，M4 §5.4）；发布时按普通创建的规则检查（迭代已结束、已归档、已删除或开关关闭则发布被拒，草稿留着可改），在发布事务里写归属与动态。差异清单“草稿的关联表”一行随之更新（P3 后端、P7 新建弹窗）。没有单独的“加入迭代”接口；迭代/模块页的“添加已有”逐个发 PATCH，结果逐项报告（我的决定，保持 v0 的单一写入口）。
- **约束**：只检查这次新加入的目标：须同项目、未删除、未归档，迭代不能已结束（Plane 同），模块不限状态（Plane 同）；已在其中的保持不动，移出不受归档、结束的限制（包括移出已归档的迭代/模块）。前端下拉框隐藏已归档的模块，PATCH 时保留工作项上已有的隐藏 id，不会因整体替换而误移出。工作项须可写（`issue.update` 加 OwnOnly）。开关关闭时 409（§3.8）；已归档的项目 409 `project.archived`。
- **动态**：照 Plane 的字段名写进 `issue_activities`，前端现有渲染器直接读：加入迭代 `created`/`cycles`/`""→名字`、移到别的迭代 `updated`/`cycles`/旧名→新名、移出或迭代被删 `deleted`/`cycles`/名字→`""`；模块同理用 `modules`。标识列放迭代/模块 id。重复加入不写动态（Plane 会重复写，不照搬）。

### 3.4 转移

`POST /cycles/{id}/transfer-issues {target_cycle_id}`：源迭代须已结束且未归档（Plane 的公共接口规则；应用接口不查，不照搬），目标须同项目、未删除、未归档、不是已结束、不是源本身。只要源里还有未完成的工作项就能再转（可以多次转移）。转走源里状态组为积压、未开始、进行中且未删除、未归档的工作项；已完成、已取消的留下。转移前若源尚未冻结，先冻结（§3.5）。每个转走的工作项一条 `updated`/`cycles` 动态（Plane 因崩溃不写，不照搬）。返回转走的数量。

### 3.5 进度、冻结、燃尽与分布

- **口径**（M1 收尾交接要求本 M 定）：服务端按状态组计数（积压、未开始、进行中、已完成、已取消）与总数；进度 = 已完成 ÷（总数 − 已取消），分母为 0 时为 0。迭代、模块各一个前端函数（`@nerve/utils`），卡片、列表、排序、侧栏都调用它（我的决定，取活动迭代卡片与迭代列表已有的口径）。
- **冻结**：迭代第一次被看到已结束时写一次 `progress_snapshot`（计数、分布、燃尽序列与冻结时刻），之后不覆盖：周期任务 `planning.freeze_cycles`（启动即运行，之后按配置间隔，默认 1 小时）与转移都会写，先写者为准。已结束的迭代只能改排序，所以冻结不会被日期修改作废。读取：列表行与 `/progress` 总是返回实时计数，另带冻结（有时）；已结束迭代的统计展示（进度、分布、燃尽）读冻结，没有冻结（任务尚未运行）时读实时；“转移”按钮与转移本身看实时的未完成数（积压、未开始、进行中 > 0），不看冻结是否为空：这指转哪些，没有可转的时转移回答 0、照常 200（[P4 spec](specs/P4-progress-transfer.md) R7）。冻结写下后不清除；改项目时区让已冻结的迭代暂时回到进行中、其间又有变化的边角接受，读者只在已结束时用冻结（R5）。Plane 以“快照为空”判断能否转移、只在转移时写快照并每次覆盖，Nerve 改为上述规则（我的决定）。模块不冻结（Plane 同）。
- **燃尽**：从开始日到结束日（项目时区）逐日一点，今天之后为 `null`（有值的是开始日到 min(结束日, 今天)），值 = 范围 − 当日结束前已完成的数量；范围 = 当前归属里除已取消外的工作项，已完成按 `completed_at` 换算到项目时区的日期；今天之后为 `null`。在开始前已完成的从第一天就计入（与 Plane 同）。模块用开始日到目标日。理想线由前端画。已完成按状态组：状态改组后缺完成时刻的从第一天起算，过时的完成时刻不算；缺日期或跨度超过 3660 天时没有燃尽（[P4 spec](specs/P4-progress-transfer.md) §2.1、R4）。Plane 按 UTC 切天、取消的永远不烧，不照搬。
- **分布**：负责人与标签两种，没有负责人、标签的计入“无”一行（只回答总数大于 0 的行，按 id、“无”最后，只给 id，[P4 spec](specs/P4-progress-transfer.md) R3），每行总数、已完成、未完成（积压/未开始/进行中）；已取消不计入。Plane 的“无负责人”恒为 0，不照搬。
- **谁能看到什么**：所有数字只算调用者能看见、未归档、未删除的工作项（已归档的工作项不计，同 Plane）；只看自己的访客（OwnOnly）总是实时计算他自己的范围，不读冻结。

### 3.6 归档

迭代只有已结束的能归档（409 `planning.cycle_not_completed`），模块只有状态已完成或已取消的能归档（409 `planning.module_not_closed`）；恢复不看迭代/模块自身的状态（开关、已归档项目、权限的规则照常）；重复归档与恢复照 M4 回答 200、不写入。归档的对象不在默认列表、下拉框和工作项可选项里，在归档目录里能看、能恢复，工作项上已有的归属照样显示和计数。归档迭代时移除指向它的收藏（总体设计 §5.5）由 M7 落实（收藏表属于 M7，§11）。

### 3.7 权限

新的动作在 `access/domain/rules.go`，按集合判定：

| 动作 | 级别 | 角色 |
|---|---|---|
| `cycle.read`、`module.read` | 项目 | 管理员、成员、访客 |
| `cycle.write`、`module.write`（建、改、归档、恢复、转移、链接） | 项目 | 管理员、成员 |
| `cycle.delete`、`module.delete` | 项目 | 管理员；成员只限建立者（`AllowCreator`） |
| `cycle.preferences`、`module.preferences`（自己的列表显示设置） | 项目 | 管理员、成员、访客 |

看不到的项目、迭代、模块一律 404，看得到但没权限 403。规则级别都是 `LevelProject`（`LevelVisible` 只给 `project.read`/`project.join`，不能放进公开项目的非成员）。工作区管理员兼项目访客按 M4 的规则（OwnOnly 只看自己）；迭代/模块的写会动到他看不见的工作项（转移、删除）或把他看不见的数量写进回答，所以 Grant 为 OwnOnly 时所有 `planning` 的写 403（我的决定）。自己的列表显示设置（`cycle.preferences`、`module.preferences`）不在“所有 `planning` 的写”之内：它不动任何工作项、不把数量写进回答，项目偏好也照常让 OwnOnly 的人改，所以 OwnOnly 照常读写；已归档的项目与迭代/模块、开关关闭也不拒绝它（[P3b spec](specs/P3b-list-engine.md) R6）。Plane 中访客能列迭代、看进度却不能看详情，Nerve 让访客读迭代/模块本身，工作项与数字按 OwnOnly 过滤（我的决定）。

### 3.8 项目的开关

`cycle_view` 关闭时，迭代的建、改、归档、恢复、转移、删除及工作项的 `cycle_id` 变化 409 `planning.cycles_disabled`；读取照常，重新打开后数据都在。`module_view` 同理（`planning.modules_disabled`）。Plane 的应用接口不看开关，只有公共接口在建改时看；Nerve 统一在写入时看（我的决定，接口与页面行为一致）。

## 4. 数据、模块归属与删除图

### 4.1 模块归属

| 模块 | 表 | 理由 |
|---|---|---|
| `planning`（新） | `cycles`、`cycle_user_properties`、`modules`、`module_members`、`module_links`、`module_user_properties` | 总体设计 §6.2 |
| `issue` | `cycle_issues`、`module_issues` | 负责人2026-11-07决定（总体设计 §6.2 的表清单随之由 13 张改为 15 张）。归属是工作项的字段（总体设计 §3.2）：工作项的创建、PATCH、详情、列表筛选/分组/排序、删除都只碰本模块的表；外键 `REFERENCES` 指向 `planning` 的表，`TestSQLCSchemaScope` 允许。先例：`issue_labels` 归 `issue`、引用 `project` 的 `labels` |

`TestSQLCSchemaScope` 不加例外：所有跨模块读写经端口（§6），列表需要的迭代/模块目录照 M4 §11.5 以数组传入静态查询。

### 4.2 表（迁移从 `00039` 起，`<v>_<模块>_<表>.sql`）

以 Plane 快照为起点，差异全部登记进差异清单：

- `cycles`：保留 `id, name, description, start_date, end_date, owned_by_id, project_id, workspace_id, sort_order, archived_at` 与审计列、`deleted_at`（P1）；`progress_snapshot` 在 P4 随冻结加入。`start_date`、`end_date` 改 `date`；`owned_by_id` 可空 `SET NULL`；删除前端与后端都不读的 `view_props`、`logo_props`、`timezone`（用项目的时区）、`version`、`external_source`、`external_id`。CHECK：日期都空或都有且开始 ≤ 结束。
- `modules`：保留 `id, name, description, start_date, target_date, status, lead_id, project_id, workspace_id, sort_order, archived_at` 与审计列、`deleted_at`；删除 `view_props`、`logo_props`、`external_*`、`description_text`、`description_html`。部分唯一 `(project_id, lower(name)) WHERE deleted_at IS NULL`；状态 CHECK；`lead_id SET NULL`。
- `module_members`、`module_links`、`cycle_issues`、`module_issues`：照快照；`cycle_issues` 加部分唯一 `(issue_id)`；`module_links` 的地址唯一照 M4 `issue_links` 的写法。
- `cycle_user_properties`、`module_user_properties`：只存 Nerve 的列表显示设置三字段（`filters`、`display_filters`、`display_properties`，形状与 CHECK 同 `project_user_properties`，M4 §3 / `00026`、`00038`），删除旧 `filters`/`rich_filters` 格式（差异清单已登记“筛选格式改变”）。
- 每条指向 `workspaces`、`projects`、`issues`、`users` 的外键有无条件索引；`sort_order` 初值与拖拽算法照 Plane（总体设计 §5.5）。

### 4.3 删除图

在 M2 §4.7、M3 §4.12、M4 §4.16 的图上延伸（物理删除时的去向）：

```
workspaces ── cycles.workspace_id / modules.workspace_id / 各关联表.workspace_id   CASCADE
projects ──── cycles.project_id / modules.project_id / 各关联表.project_id         CASCADE
issues ────── cycle_issues.issue_id / module_issues.issue_id                       CASCADE（M4 §4.16 预留）
cycles ────── cycle_issues.cycle_id / cycle_user_properties.cycle_id               CASCADE
modules ───── module_issues / module_members / module_links / module_user_properties CASCADE
users ─────── cycles.owned_by_id、modules.lead_id                                   SET NULL（Plane CASCADE / SET NULL）
              module_members.member_id、*_user_properties.user_id                    CASCADE
              各表 created_by_id / updated_by_id                                     照 M2 §3.13
```

软删除与清理：
- 删迭代/模块：软删本行，在同一事务发布 `planning.cycle_deleted` / `planning.module_deleted`；`issue` 订阅，软删归属并写动态（§3.3）。
- 删工作项：`issue` 在 `issueRowSteps` 里软删本模块的两张归属表。
- 删项目、工作区：`planning` 订阅 `project.deleted`/`workspace.deleted`，在原事务投递 `planning.delete_rows`（沿用请求的 At/By，先例 `issue/app/delete_rows_on_deletion.go`）；父范围立即隐藏。
- 60 天物理清理：`planning` 加入 `cleanup.purge_deleted` 的模块顺序（asset → issue → planning → project → workspace → identity），每条语句有界并合 `TestPurgeScale` 的三种形状（M5-M6-03）；`purge_catalog_test` 的表数、删除目录守卫、`furnishIssueRows` 随每张表同步扩展。

## 5. 接口

新文件 `api/modules/planning.yaml`、标签 `planning`。列表挂在父资源下，单个资源短路径，集合型列表一次返回 `{data}` 并写明顺序（总体设计 §3.2、§3.4）：

| 方法与路径 | 说明 |
|---|---|
| `GET/POST /projects/{project_id}/cycles` | 默认未归档迭代，按 `sort_order, created_at, id`；`?archived=true` 只返回已归档的，按 `archived_at DESC, id`（总体设计 §3.3）；行内带状态、实时计数与冻结：列表行为 `CycleListItem`（六个实时数、`assignee_ids` 与冻结的计数），`Cycle` 与各写入的回答不带计数（[P4 spec](specs/P4-progress-transfer.md) R9） |
| `GET/PATCH/DELETE /cycles/{cycle_id}` | 详情、修改、软删 |
| `POST /cycles/{cycle_id}/archive`、`/unarchive` | §3.6 |
| `POST /cycles/{cycle_id}/transfer-issues` | §3.4；体 `{target_cycle_id}`，答 `{transferred_issues}` |
| `GET /cycles/{cycle_id}/progress` | `{status, live, progress_snapshot}`：实时的计数、分布、燃尽与冻结（有时）（替换旧 `analytics?type=issues` 与 `progress` 两个地址，[P4 spec](specs/P4-progress-transfer.md) §2.3） |
| `GET/PATCH /me/cycles/{cycle_id}/preferences` | 迭代页的工作项列表显示设置（同项目的偏好结构） |
| 模块：`GET/POST /projects/{project_id}/modules`（`?archived=true` 同上）、`GET/PATCH/DELETE /modules/{module_id}`、`archive`/`unarchive`、`GET /modules/{module_id}/progress`、`GET/PATCH /me/modules/{module_id}/preferences` | 同上（列表行为 `ModuleListItem`，模块的 `/progress` 答 `Progress`，不冻结）；`PATCH` 的 `member_ids` 整体替换 |
| `GET/POST /modules/{module_id}/links`、`PATCH/DELETE /module-links/{link_id}` | §3.2 |
| `issue.yaml`：`Issue`/`IssueDetail` 加 `cycle_id`、`module_ids`；创建与 PATCH 接受它们；列表加筛选 `cycle_id`、`module_id`（含 `none`）、分组 `cycle_id`、`module_id`、排序 `cycle_name`、`module_name` 两向、显示属性 `cycle`、`modules` | §7 |

错误码前缀 `planning.`，每个操作声明 `x-problem-codes`；请求体校验沿用 bodyshape（400）与字段错误（422）的分工。没有工作区级的迭代/模块列表（Plane 的 `/workspaces/{slug}/cycles/` 只给工作区视图与通知用，属 M7）。

## 6. 跨模块协作、事务与锁

M2 交接第 1 节要求列出每个跨模块的写入与查询；全部走端口，没有 SQL 例外：

| 方向 | 用途 | 端口（由使用方声明，在 `bootstrap` 接线） |
|---|---|---|
| `issue` → `planning` | 写 `cycle_id`/`module_ids` 时读取目标的事实（项目、是否删除/归档、迭代是否已结束、开关） | `PlanningTargets`：在工作项写已取得的项目行锁下读取 |
| `issue` → `planning` | 按名字排序时的迭代/模块名字（含已归档、不含已删除），只在 `±cycle_name`/`±module_name` 时于读行的同一快照按项目 id 数组取得；分组头由前端给、筛选值只做语法检查，都不用目录（[P3b](specs/P3b-list-engine.md) R2） | `PlanningDirectory`：`CycleNames`、`ModuleNames` 按项目 id 数组返回 id 与名字，数组传入静态查询（M4 §11.5） |
| `issue` → `planning` | 自动归档要排除的未结束迭代、未到目标日期模块 | `OpenPlanning.OpenTargets`：在项目锁下按项目时区返回 id 数组（P3a） |
| `planning` → `issue` | 计数、分布、燃尽（按调用者的可见范围）；列表行的批量计数 | `IssueStats`（planning/app 声明，即原写的 `PlanningIssueStats`）：按状态 id 回答工作项事实与批量计数，状态组经项目的读取，计算在 planning 的纯函数（[P4 spec](specs/P4-progress-transfer.md) R2） |
| `planning` → `issue` | 转移：在同一事务移动归属并写动态 | `IssueTransfer.TransferCycleIssues`：经 ctx 在 planning 的写事务里，软删加插入、每个一条动态、`MarkChanged`（[P4 spec](specs/P4-progress-transfer.md) §2.6） |
| `planning` → `project` | 项目事实（时区、开关、成员角色）、项目锁；改迭代/模块页的偏好时项目行 `FOR SHARE`（[P3b](specs/P3b-list-engine.md) R6）；状态组（P4） | 复用/扩展现有项目端口（状态组复用 `IssueStates`） |
| `planning` → `issue` | 迭代/模块页偏好里的筛选与显示筛选的词（纯校验，不读库） | `IssueListSettings`：与项目偏好同一个 `issue.ListSettings`（[P3b](specs/P3b-list-engine.md) R5） |
| `planning` → 成员 | 负责人、成员是否为项目中不是访客的有效成员 | 项目成员端口 |
| 事件 | `planning.cycle_deleted`/`module_deleted` → `issue` 软删归属；`project.deleted`/`workspace.deleted` → `planning` 投递清理任务 | 同步分发器，发布者事务内（`bootstrap/events.go`） |

**锁序**：`planning` 的每个写沿用 M4 的加锁之路（[硬规则 §4](../claude_to_gpt/03-rules-and-pitfalls.md)）：不加锁预检全部 404/403 → 进入项目写闸门（无连接、无锁时）→ 开事务、项目行 `FOR NO KEY UPDATE` → 锁下重读与再判定 → 迭代/模块行 → 最后一把锁之后读时钟 → 同一时刻、同一写者。工作项的写已经先取项目行锁，因此同一项目里的工作项写与迭代/模块写按项目行串行，归属、日期重叠、模块重名、转移、冻结都在这把锁下检查与写入；`planning` 复用 `platform/gate` 与项目锁端口，不复制工作项的判定。周期任务（冻结）按项目逐个取同一把锁；项目/工作区删除后的清理任务不取项目锁（项目已删除，同 `issue/app/delete_rows.go`）。具体推导与交错测试在 P1/P3/P4 的 spec。

## 7. 工作项列表、动态与自动归档

- **列表引擎**（M4 交接 M6-02）：`listvocab.go` 加筛选 `cycle_id`、`module_id`（值为 id 或 `none`，同 `label_id` 的语法）、分组 `cycle_id`、`module_id`（沿用分组码等于筛选名的约定；模块是多值分组，同标签）、排序 `cycle_name`、`module_name` 两向（Plane 的表头提供迭代排序而后端允许表只有模块名，Nerve 两者都支持，登记差异）、显示属性 `cycle`、`modules`；`api/testdata/issue-order.json`、游标键、`list_settings.go`、项目偏好的 CHECK（新迁移，先例 `00038`）同步。筛选与分组的 `cycle_id`、`module_id` 只限项目列表，“我的工作项”回答 422；四个排序词两种列表都可用（[P3b spec](specs/P3b-list-engine.md) R1、R3）。迭代/模块页的列表就是项目列表加 `cycle_id`/`module_id` 筛选，读本迭代/模块的偏好（偏好里的筛选不含本域的 `cycle_id`/`module_id` 键），不另开列表接口。附件列随共享设置出现（M5-M6-02）。
- **动态**：§3.3。
- **自动归档**（M4 §3 `:431`）：照 Plane，工作项不在未结束的迭代（结束日为空或 ≥ 今天）且不在未到目标日的模块（目标日为空或 ≥ 今天）时才可自动归档；id 数组经 `OpenPlanning.OpenTargets` 在项目锁下取得，保持每项目事务、固定业务时钟。“未结束”的 SQL（结束日为空或 ≥ 今天）是 `StatusOf` 的“已结束”取反，两者都在 planning，由适配器测试 `TestOpenCycleIDs` 对照钉住（P3a）。
- **搜索**：M4 的 `issue-search` 不加 Plane 的迭代/模块变体（“添加已有”用普通搜索并排除已在其中的工作项，见 §8）。

## 8. 前端

- **按总体设计 §7.2 重写本领域**：类型来自 OpenAPI 生成的类型，删除 `@nerve/types` 的手写 `ICycle`、`IModule` 等；service 只调 `/api/v0`；store 照 §7.7 按会话分代（SWR 键带 `loginId`；迭代的筛选 store 按项目 id 存取、不跟随 `router`，没有要释放的反应（[P5a spec](specs/P5a-web-cycles.md) R4）；模块的筛选 store 在 P6a 同样按项目 id 存取、不跟随 `router`、没有反应，筛选与搜索只在会话里、不写 `localStorage`（筛选里有成员的 id），列表的版式与排序是纯展示的设置、按项目存在浏览器（[P6a spec](specs/P6a-web-modules.md) §2.4、R4）；迭代页偏好的筛选 store（`CycleIssuesFilter`）按迭代 id 存取、不读 `router`，与项目页共用按页的基类（[P5b spec](specs/P5b-cycle-list.md) §2.4、R3），模块页照用（P6b）；删除 2 个模块级 `CycleService` 实例）。旧 service、store 方法、手写类型、`date-check`、工作区级取数（`use-workspace-issue-properties.ts` 的两处：迭代的在 P5a，模块的连同整个 hook 在 P6a）一并删除。
- **页面**：迭代列表（进行中卡片、“即将开始”（含草稿）、“已结束”，照 Plane 的三段，[P5a spec](specs/P5a-web-cycles.md) R5）、详情（工作项列表 + 进度侧栏；列表是项目列表加 `cycle_id`，已结束、已归档迭代的列表只读，[P5b spec](specs/P5b-cycle-list.md) R1、R5）、归档目录；模块列表（列表/卡片）、详情（列表 + 侧栏、链接、成员；页头与侧栏在 P6a，列表在 P6b，[P6a spec](specs/P6a-web-modules.md) §1）、归档目录。列表页真正取数，详情页以 `cycleId`/`moduleId` 给子树 key（M3 交接 §1）；项目级弹窗宿主移到项目的 key 之下并在卸载时关上开关（M3 交接 §1）。两域共用一个取数门（`usePlanningReads`：可读、列表可取，开关的谓词与项目包装层、写的门共用）、读失败与停用空态（`core/components/planning/`）、分布的行与进度页签（`core/components/core/sidebar/`，页签与面板一致）（[P6a spec](specs/P6a-web-modules.md) §2.3、§2.5）。
- **图表**：`BurndownChart` 覆盖 0、1、多个点，坐标有限（M4 交接 M6-04，P5a）；模块侧栏在 P6a 改用它之后，`ProgressChart` 与 `AreaChart` 没有调用方的 `comparisonLine`（它的单点除零）删除（[P6a spec](specs/P6a-web-modules.md) §2.5）。
- **工作项一侧**：恢复 `dropdowns/cycle`、`dropdowns/module`（popper 的 ref 在列表元素上、`Combobox`/`Listbox` 二选一、弹窗里 `modal={false}`，M2/M3 交接）；工作项详情、预览、列表行的迭代/模块属性；列表的筛选、分组、显示属性；`existing-issues-list-modal.tsx` 的迭代/模块调用方；日历的 `addIssuesToView`；转移弹窗。
- **键盘与可访问性**：M3 交接 §3 逐项（`ModalTitle`、正 `tabIndex` 删除、嵌套按钮、每行控件以行命名、字段提示与 `useRefusedFocus`、模块状态选择的提示、筛选按钮）；迭代/模块的表单与归档按钮迁到自然 Tab 顺序（`getTabIndex` 的最后读者在 M7 的视图与收集箱表单，M6 不删它）。
- **地址与复制**：本领域 22 个读 router store 地址的文件改读路由参数并纳入 `project-address-route`；5 处复制改经 `useCopyLink`/`useCopyText`，删除 `copy-hook` 的例外。迭代的文件与两处复制、迭代的收藏入口、项目级弹窗宿主的 key 在 P5a 随页面的重写完成（P5a spec R2），模块的文件、复制与收藏入口在 P6a 完成（[P6a spec](specs/P6a-web-modules.md) §2.9、§2.10）。
- **收藏**：迭代/模块的收藏按钮属于 M7 的收藏功能；M6 删除本领域里调用旧收藏接口的入口与 store 逻辑，交 M7 随收藏一起恢复（§11）。
- **命令面板**：迭代、模块的打开与切换在已取得的项目列表里本地过滤（同项目的做法，M4 设计 `:40`、`:671` 交下）；工作区范围的搜索属 M7。
- **交给 M6 的视图文件**：M3 交接把 4 个视图文件的地址读取（`views/(detail)/[viewId]/header.tsx`、`views/(list)/page.tsx`、`views/view-list-item-action.tsx`、`views/views-list.tsx`）、视图的跟进经 `followInSession`、`isCreateViewModalOpen` 随项目弹窗宿主一起分派给 M6，M6 照做；保存的视图自己的页面仍属 M7。
- **动态渲染**：`issue-activity/activity/actions/cycle.tsx` 的移出分支读 `new_value`，按 §3.3 的写法会显示空名，改为读 `old_value`（模块的渲染器已如此）。

## 9. 运行与观察

- 周期任务 `planning.freeze_cycles`（§3.5）：配置 `planning.freeze_interval`，默认 1h（测试环境 2s，bootstrap 测试 1h、只有启动时的一次），`RunOnStart`；进入 `all_periodic_jobs_test` 与其余五处周期种类清单；不进写闸门，每项目一个事务、先锁后读时钟（[P4 spec](specs/P4-progress-transfer.md) §2.5）。
- 请求触发的任务：`planning.delete_rows`（项目/工作区删除）、`planning.crawl_link`（模块链接元数据），登记 `EnqueuedKinds` 并各有 worker。
- 启动时运行的任务可能插到持锁的测试后面（M5 P8 的教训）：新的周期任务进入 `waitForStartupRuns`。

## 10. 验证策略

单元覆盖日期、状态、重叠、口径、燃尽等纯规则；adapter 覆盖 SQL、约束与锁；`bootstrap` 组合覆盖认证、权限矩阵、事务、事件、删除图、清理与全程序接口；Playwright 覆盖真实页面与数据库断言（§2 每条）。负例至少：跨项目的迭代/模块、访客与 OwnOnly、工作区管理员兼访客、已归档/已删除/已结束、开关关闭、并发两次建重叠迭代、写归属与删迭代交错、转移与写归属交错、冻结与转移交错、换会话与迟到回答。每个安全、加锁、访客性质至少有一个能让它失败的变异。完整七项本地门禁先于有产品变化的检查点推送；分支 CI 与 `--no-ff` 后的 main CI 分开记录（M5 的规则，含环境失败与测试失败的区分）。

## 11. 输入交接与后续交接

| 输入 | 条款落点 | 关闭阶段 |
|---|---|---|
| [M1-P2-trim-content](handoffs/M1-P2-trim-content.md) | §3.5 只按状态组计数、无点数字段；`analytics` 例外随 P1 删除旧地址 | P1（例外）/ P4（接口）/ P5（前端） |
| [M1-P3-trim-platform](handoffs/M1-P3-trim-platform.md) | 进度改用新接口、无结尾斜杠问题、无已删类型 | P5 |
| [M1-closeout](handoffs/M1-closeout.md) | 84 行死成员/prop、改到文件 oxlint 为零并清一类规则、进度口径三处 | P5–P8 |
| [M2-closeout](handoffs/M2-closeout.md) | §1 跨模块写入表（§6）、§2 删除图（§4.3）、§3 会话分代、§4 下拉框取数三点、§5 下拉框、§6 复制 | P1/P3（后端）、P5–P7（前端） |
| [M3-closeout](handoffs/M3-closeout.md) | §1 取数与 stores、弹窗宿主与 key；§2 删除图、8 行死成员；§3 键盘与弹窗；§4 地址、复制、oxlint、守卫阶段；§5 焦点索引 | P1、P5–P8 |
| [M4-closeout](handoffs/M4-closeout.md) | M6-01 领域页面验收、M6-02 新引擎列表、M6-03 关系表与删除、M6-04 表单焦点与图表 | P3、P5–P8 |
| [M5-closeout](handoffs/M5-closeout.md) | M5-M6-01 不成为资产宿主、M5-M6-02 附件计数、M5-M6-03 有界清理、M5-04 原文转交 | P3、P8 |

输入交接各写“处理结果（M6/P<k>）”，保持 open 至收尾关闭。后续交接由 P8 写给 M7（收藏与归档时移除收藏、工作区级迭代/模块列表与视图筛选、`getTabIndex` 的最后删除、M5-04 原文）与 M8（对等验收、观察项）。

## 12. 分阶段推进

每段先 spec、plan、预检，再逐 Task 实现，结束 deep-review 与修复复审；每段约 16 Task 以内，超限在实施前拆分。后续阶段只在前置事实落实后写可执行 plan。

| Phase | 交付边界 | 必须同时完成的验证 |
|---|---|---|
| P1 | `planning` 模块、`cycles` 表与迭代的建/读/改/删/归档/恢复/归档目录、日期与状态、重叠、开关、权限矩阵、删除图（项目/工作区删除、清理任务、60 天清理）；删除前端旧 `analytics` 调用路径（例外到期） | G1–G3、G8、G18 的接口与数据库；锁与交错、删除目录守卫与清理规模 |
| P2 | `modules`、`module_members`、`module_links` 与模块的建/读/改/删/归档/恢复、成员与负责人校验、链接与元数据抓取 | G9、G10、G13 的接口与数据库 |
| P3a | `cycle_issues`、`module_issues`（`issue` 模块）、工作项与草稿的 `cycle_id`/`module_ids`（含回答）、动态、删除迭代/模块与工作项的归属清理、自动归档 | G4、G11、G14、G16 的接口与数据库；交错 |
| P3b | 列表引擎的筛选/分组/排序/显示、迭代与模块的偏好 | G15 的接口与数据库；同 fixture 的 Go 与前端顺序测试 |
| P4 | 进度/分布/燃尽接口、冻结任务、转移、列表行的计数 | G5–G7、G12、G17 的接口与数据库；冻结与转移交错 |
| P5a | 前端迭代：service/store 按会话分代、列表/详情/归档、表单与弹窗、进度侧栏与进行中卡片、图表、转移弹窗；迭代文件的地址/复制/收藏入口与项目级弹窗宿主（P5 因超出一段的上限拆为 P5a/P5b，[P5a spec](specs/P5a-web-cycles.md) §1、R1、R2） | G1–G3、G5–G8、G19 的页面 |
| P5b | 迭代页工作项列表（`EIssuesStoreType.CYCLE`、本页偏好、四种布局、本页新建、已结束只读、附件列）、进行中卡片的优先级页签、卡片与侧栏的行点击 | G15、G19 的列表部分 |
| P6a | 前端模块：service/store 按会话分代、迭代与模块共用的取数门与件、列表/卡片与预览、归档目录、详情页的页头与侧栏（状态、负责人、成员、日期、链接、进度）、新建/修改弹窗、快捷操作、删除与归档；模块文件的地址/复制/收藏入口，跨领域读者改用新 store、旧代码删除（P6 因超出一段的上限拆为 P6a/P6b，[P6a spec](specs/P6a-web-modules.md) §1、R1） | G9、G10、G12、G13 的页面，G19 的模块页面 |
| P6b | 模块页工作项列表（`EIssuesStoreType.MODULE`、本页偏好、四种布局、本页新建与“移出模块”的 `module_ids` 整集合、附件列、侧栏的行切换本页筛选）；加入之前先按职责拆开工作项根 store（M6-M8-08，[P6a spec](specs/P6a-web-modules.md) R16） | G15、G19 的模块页列表部分（[P6a spec](specs/P6a-web-modules.md) R15） |
| P7 | 前端工作项一侧：属性与下拉框（含新建弹窗与草稿）、列表筛选/分组/显示、添加已有、日历、命令面板（迭代、模块菜单的本地过滤与切换）、迭代页的移动端页头（[P5b spec](specs/P5b-cycle-list.md) §1、§5）；迭代、模块文件的地址与复制、收藏入口与项目级弹窗宿主已在 P5a、P6a 完成 | G4、G11、G15、G19 的页面；交接中的键盘逐项 |
| P8 | 最终矩阵、清理规模重跑、死成员与告警收敛、文档同步、M7/M8 交接 | G1–G20 最终证据、deep-review |
| closeout | 首次 noff/main 绿 → §13 勾选与总体 §9.4 → 第二次 noff/main 绿 → 归档清理 | §13 每条证据、全部输入关闭 |

P1 起第一个实现提交把关键词 phase 改为 `M6/P1`；纯设计分支不改 phase。

## 13. 完成标准

仅在收尾首次 main CI 成功后逐条勾选，并附最终证据；本节现在全未完成。

- [ ] G1–G20 每条有与失败路径相区分的接口、数据库与页面证据。
- [ ] 迭代的日期、状态、重叠、可改范围与归档按 §3.1、§3.6；模块的名字、状态、成员、链接与归档按 §3.2、§3.6。
- [ ] 工作项归属只经工作项字段、一个迭代、多个模块，动态完整；转移与冻结按 §3.4、§3.5。
- [ ] 进度口径由每域一个函数计算，各处显示一致；燃尽与分布按项目时区与可见范围。
- [ ] 列表引擎的筛选、分组、排序、显示与迭代/模块页的列表走同一引擎，Go 与前端同 fixture。
- [ ] 权限矩阵与开关、访客与 OwnOnly、私有项目、撤权全部通过；无 Plane 的默认 PUT 或跨项目漏洞。
- [ ] 删除图延伸、事件与清理任务、60 天有界清理、`TestPurgeScale` 与 `TestTheDeletionsAtTheScaleTarget` 在 M6 代码上重跑。
- [ ] 前端只调 `/api/v0`，类型来自 OpenAPI，stores 按会话分代，旧 service/类型/地址删除。
- [ ] 交接中的键盘、弹窗、下拉框、地址、复制逐项在浏览器核对；本领域死成员与改到文件的告警为零，清一类规则。
- [ ] `analytics`、`copy-hook` 的例外删除，无到期例外；差异清单与前端改动清单同步。
- [ ] 所有输入交接 closed；M7/M8 收到 M6-closeout，M5-04 原文未丢。
- [ ] 最终本地七项、deep-review 与修复复审、精确分支/main CI 通过；历史失败诚实保留。
- [ ] 全阶段与收尾 noff 合入，完成标记第二次 noff 后 main 全绿，总体 §9.4 标完成。
- [ ] 原始证据完整归档并核验，分支与 worktree 正常清理、未知文件为零。

## 14. 进度与裁定

| 阶段 | 状态 | spec / plan / review |
|---|---|---|
| 设计与 P1 预检 | 已合入 main（d1ec8c69）；e2e 作业期限改为 35 分钟（910a55cd，负责人同意），main CI 42332404481 第二次运行通过 | [预检](../../reviews/re-54-M6设计与P1预检.md) / [P1 spec](specs/P1-cycles-backend.md) / [P1 plan](plans/P1-cycles-backend.md) |
| P1 迭代后端 | 已完成：noff `0a601f21`，分支 CI 42340200507 与 main CI 42343073949 首次通过，证据归档于 `.superpowers/sdd/M6-P1-integration-20261107` | [P1 spec](specs/P1-cycles-backend.md) / [P1 plan](plans/P1-cycles-backend.md) / [P1 阶段记录](reviews/P1-cycles-backend-review.md) / [re-55](../../reviews/re-55-M6P1整体审核.md) |
| P2 模块后端 | 已完成：noff `861ea1b5` 的 main CI 两次未过（与 P2 无关），根因修复（含负责人 2026-11-08 批准的 server 任务分两路）第二次 noff `ddd79a9f`，main CI 42458454710 首次通过；证据归档于 `.superpowers/sdd/M6-P2-integration-20261108` | [预检](../../reviews/re-56-M6P2预检.md) / [P2 spec](specs/P2-modules-backend.md) / [P2 plan](plans/P2-modules-backend.md) / [P2 阶段记录](reviews/P2-modules-backend-review.md) |
| P3a 工作项的归属 | 已完成：noff `2648a739`，分支 CI 42484965936 与 main CI 42487991301 首次通过（检查点 CI 42478883539 第一次因 GitHub 504 未过，只重跑失败的作业后通过），证据归档于 `.superpowers/sdd/M6-P3a-integration-20261108`；P3 因超出一段的上限拆为 P3a/P3b（§12） | [预检](../../reviews/re-58-M6P3a预检.md) / [P3a spec](specs/P3a-issue-membership.md) / [P3a plan](plans/P3a-issue-membership.md) / [P3a 阶段记录](reviews/P3a-issue-membership-review.md) |
| P3b 列表引擎与偏好 | 已完成：noff `ed8f0df4`，检查点 CI 42594813667、分支 CI 42616111421 与 main CI 42621199100 首次通过（最终门禁第一次因本机 Docker API 超时未过，整段重跑一次通过），证据归档于 `.superpowers/sdd/M6-P3b-integration-20261109` | [预检](../../reviews/re-60-M6P3b预检.md) / [P3b spec](specs/P3b-list-engine.md) / [P3b plan](plans/P3b-list-engine.md) / [P3b 阶段记录](reviews/P3b-list-engine-review.md) |
| P4 进度、冻结与转移 | 已完成：noff `c3ba5a40`。检查点 CI 42775845319 的 e2e 第一次因 W11 的老问题未过，只重跑失败作业后通过，W11 据 trace 找到根因并修正；最终候选的分支 CI 第一次因测试取的时刻带纳秒未过，修正后 42803535189 通过；合入后 main CI 42809569341 的 e2e 因 C12 未过，据 trace 查实为产品缺陷（编辑器建成之前就挂引用），修复分支第二次 noff `a2b50624`，main CI 42824266166 首次通过；证据归档于 `.superpowers/sdd/M6-P4-integration-20261110`、`M6-P4-ci-20261110` | [预检](../../reviews/re-62-M6P4预检.md) / [P4 spec](specs/P4-progress-transfer.md) / [P4 plan](plans/P4-progress-transfer.md) / [P4 阶段记录](reviews/P4-progress-transfer-review.md) |
| P5a 网页的迭代（数据层与页面） | 已完成：noff `0b8efcd7`；最终候选的分支 CI 42984392888 的 e2e 因 G18 在冻结任务前后比较读失败，根因修正后 42990196723 首次通过，main CI 42995581409 首次通过；本机 `make test` 两次因 Docker Desktop 与 macOS 的端口范围重叠失败，负责人 2026-11-11 决定以分支 CI 为准（M6-M8-05，并延续到 P5b–P7 中不改 Go 的阶段）；证据归档于 `.superpowers/sdd/M6-P5a-integration-20261111`。P5 因超出一段的上限拆为 P5a/P5b（§12） | [预检](../../reviews/re-64-M6P5a预检.md) / [P5a spec](specs/P5a-web-cycles.md) / [P5a plan](plans/P5a-web-cycles.md) / [P5a 阶段记录](reviews/P5a-web-cycles-review.md) / [re-65](../../reviews/re-65-M6P5a整体审核.md) |
| P5b 迭代页的工作项列表 | 已完成：noff `9675bb31`。检查点 `00cf6c38` 的分支 CI 43047769115 的 e2e 两次因 runner 解析不了 Ubuntu 的镜像而取消（M6-M8-06），T13 之后 `f4a37007` 的 43070304124 全绿；整分支审查 [re-67](../../reviews/re-67-M6P5b整体审核.md) 没有 Critical、Important，修复轮 F 另经限定复审；最终候选的本机门禁中 build+e2e 一次 F11 (page) 的中止读取（M6-M8-07，与本段无关，单独重跑通过），本机 `make test` 通过；记录提交的分支 CI 43098103752 一直排队、没有派发作业，补记后的 43354077443 的 e2e 因 I21 在受保护的行留给下一轮时就核对物理删除（M4 起潜伏的测试缺陷）未过，根因修正后最终候选 `ded24135` 的 43358881107 首次通过；合入后 main CI 43362963122 的 e2e 因 G15 故事三跳转后以两个页面都有的行为锚（本段故事的缺陷）未过，据 trace 修正，修复分支第二次 noff `aadaf61a`，main CI 43373039561 首次通过；证据归档于 `.superpowers/sdd/M6-P5b-integration-20261113` | [预检](../../reviews/re-66-M6P5b预检.md) / [P5b spec](specs/P5b-cycle-list.md) / [P5b plan](plans/P5b-cycle-list.md) / [P5b 阶段记录](reviews/P5b-cycle-list-review.md) / [re-67](../../reviews/re-67-M6P5b整体审核.md) |
| P6a 网页的模块（数据层与页面） | 已实现，待整审与合入：T1–T15 已提交（P5b 的回填 `0a7313f1`，实现 `d71ab782`–`dc9172b0`，G9、G10、G13 与 G12、G19 的页面故事 `b1d2578e`、`b8ed29df`，交接回执与差异清单在 T15）；检查点 `82dac1ef` 的本机七项中六项通过（含本机 `make test`），build+e2e 451 条通过、W6 (API) 因 M6-M8-03 失败，只重跑这一项即通过；它的分支 CI 43418051413 全绿，e2e 作业 25 分 18 秒（基线 main CI 43373039561 为 19 分 14 秒，逐条比对是 runner 的速度差异，不是产品的回归）；T16 的门禁、整审与合入未做。P6 因超出一段的上限拆为 P6a/P6b（§12，[P6a spec](specs/P6a-web-modules.md) §1），P6b（模块页的工作项列表）在 P6a 合入后写 spec 与 plan | [预检](../../reviews/re-68-M6P6a预检.md) / [P6a spec](specs/P6a-web-modules.md) / [P6a plan](plans/P6a-web-modules.md) |
| P6b、P7、P8、closeout | 未开始 | — |

## 15. 决定记录

负责人2026-11-07的决定：

- **A. 归属表**：`cycle_issues`、`module_issues` 放 `issue` 模块（§4.1）；总体设计 §6.2 同步改为 15 张。
- **B. 草稿**：照 Plane 带迭代与模块（§3.3）。
- **C. 其余取舍**：按下列 17 条推进（设计中标为“我的决定”）；实施中发现不合适的，仍按规则先提交再上报。


1. 迭代/模块本身归 `planning`，全部跨模块走端口（§4.1、§6；归属表见 A）。
2. 迭代日期存项目时区的日历日，状态按项目时区推算（§3.1）。
3. 日期重叠由服务端拒绝，删除 `date-check`（§3.1）。
4. 已结束迭代只能改排序（§3.1）。
5. 进度 = 完成 ÷（总数 − 取消），迭代与模块同口径（§3.5）。
6. 冻结在迭代结束后写一次、不覆盖（§3.5）。
7. 燃尽按项目时区，范围不含已取消（§3.5）。
8. 转移只从已结束的迭代转出（§3.4）。
9. 模块负责人与成员须是项目中不是访客的有效成员（§3.2）。
10. 开关关闭时拒绝写入、读取照常（§3.8）。
11. 访客能读迭代/模块本身，数字只算他能看见的工作项（§3.5、§3.7）。
12. 归属只经工作项 PATCH，“添加已有”逐个 PATCH（§3.3）。
13. 迭代/模块的收藏入口在 M6 删除，交 M7 随收藏恢复（§8）。
14. 冻结后照样返回实时计数，转移看实时未完成数、可多次转移（§3.4、§3.5）。
15. OwnOnly 的调用者不能做 `planning` 的写（§3.7）。
16. 排序加 `cycle_name`（§7）。
17. 链接的规整与抓取移出 `issue`、两个模块共用一份（§3.2）。落点见 §3.2 / P2 spec R12：检查与元数据类型在 `internal/shared`，抓取在 `platform/outbound/linkpreview`，`bootstrap` 一次适配（原写“移到 `platform`”，因架构测试的导入规则而改）。
