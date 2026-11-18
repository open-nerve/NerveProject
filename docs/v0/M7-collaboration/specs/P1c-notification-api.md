# M7 / P1c：通知的接口与读

> 状态：spec 经预检 [re-84](../../../reviews/re-84-M7P1c预检.md) 修正后通过。基线 main `1a23f6f8c419dc9df6d305b1ed780c7d2cdff54c`（P1b 的 noff）。设计见 [M7 设计](../M7-design.md) §2（H2–H7、H19）、§3.3、§3.4、§3.10、§5、§6.3、§9；P1b 交来的表、快照与后续责任见 [P1b spec](P1b-notification-core.md) §2.2、§2.5、§5 与 R2、R6。现状调研见本阶段台账 `research/p1c-facts.md`（ignored，随阶段归档）。

## 1. 目标与边界

P1c 交付通知的四个接口与读时的过滤：列表、未读数、全部已读、单条修改；读的人此刻看得到的项目（含已归档的）与只看自己；通知里另一端工作项（父、关联）按读的人此刻的可见给出编号与名字；新动作与权限矩阵；H2–H7 的接口证据与 e2e 故事。关键词 phase 仍是 `M7/P1`（P1a、P1b、P1c 共用）。

P1c 不做：网页（P4，旧的 8 个 Plane 地址与 `TNotification` 随 P4 删除并加关键词规则）；通知的保留期（设计 §3.4：不设）；单条读取与删除（`GET`、`DELETE /notifications/{id}`，设计 §5 没有）；迁移与索引（要加先停下报告）。

## 2. 交付合同

### 2.1 文件与职责

| 路径 | 职责 |
|---|---|
| `api/modules/notification.yaml`、`api/openapi.yaml` | 四个操作、`notification` 标签、`x-problem-codes`（§2.4） |
| `server/internal/modules/notification/` | `domain/`（筛选的解析与校验、游标、修改的规则、回答里放行与解析的字段，§2.3–§2.7）；`app/`（四个用例与端口）；`adapter/postgres/`（列表、计数、单行的读与锁、全部已读的一批）；`adapter/http/`（处理器、`Register`、`gen/`）；`module.go` 的 `Deps` 加事务、时钟、授权与读端口 |
| `server/internal/modules/access/` | 动作 `notification.read`、`notification.write`（工作区级，管理员、成员、访客）；`NewReadableProjects`（§2.5，与 `Authorize` 同一个判定函数，照 `NewDecider`） |
| `server/internal/modules/project/` | 上一条要的一次批量读；另一端的项目标识按 id 数组的一次读（§2.6） |
| `server/internal/modules/issue/` | `IssueRefs`：一组工作项 id 按读的人此刻的可见给出编号、名字与所在项目（§2.6） |
| `server/internal/bootstrap/` | 接线、端口转换、`actions_test` 把 `notification` 移出 `actionlessModules`；组合测试、权限矩阵、等价测试、交错 |
| `web/` | 只有生成的类型、`error-messages.ts` 与两份 `errors.json` 的新码（页面在 P4；契约不用会改网页客户端的写法，§2.3） |
| `e2e/` | H2–H7 的接口故事 |
| `docs/` | plane-diff；硬规则 03 §4 的登记（R2、R5）；M4 交接 M7-02 与 M5 交接 M5-M7-02 的“处理结果（M7/P1c）”；设计 §3.4、§5、§14；阶段记录 |

### 2.2 接口

| 操作 | 地址 | 动作 | 回答 |
|---|---|---|---|
| `listNotifications` | `GET /workspaces/{slug}/notifications` | `notification.read` | `NotificationPage`：`{data: Notification[], next_cursor: string \| null}` |
| `getNotificationUnreadCount` | `GET /workspaces/{slug}/notifications/unread-count` | `notification.read` | `{all, mentions}`（R3） |
| `markAllNotificationsRead` | `POST /workspaces/{slug}/notifications/mark-all-read` | `notification.write` | `{marked}`：这一次标为已读的条数 |
| `updateNotification` | `PATCH /notifications/{notification_id}` | `notification.write` | `Notification` |

- 三个工作区级的操作按 slug 找工作区，不是有效成员的 404 `workspace.not_found`（各模块共用的码）；三种角色都放行，`forbidden` 不可达，照 `getWorkspacePreferences` 不声明。
- 单条修改见 §2.7 的次序：不是调用者的、已删除的、所在工作区他不是成员的、项目此刻读不到的、只看自己而不是他建的工作项的，一律 404 `notification.not_found`，与随机 id 逐字节相同（体、头、码）。

### 2.3 列表与筛选

查询参数全部声明为字符串（由 domain 解析；`type` 是逗号分隔的字符串而不是数组，网页客户端的序列化不必改）：

| 参数 | 取值 | 含义 |
|---|---|---|
| `tab` | `all`（默认）、`mentions` | `all` 列 `sender` 不是 `mentioned` 的；`mentions` 只列 `mentioned` 的（照 Plane 的两个页签，R3） |
| `type` | `assigned`、`created`、`subscribed` 的非空子集，逗号分隔、成员不重复 | 只在 `tab=all` 用：只列 `sender` 在其中的；`tab=mentions` 给了它 422 `not_allowed`（字段 `type`） |
| `read` | `true`、`false` | `false` 只列未读，`true` 只列已读；省略两者都列 |
| `archived` | `true`、`false`（默认） | `false` 只列未归档的，`true` 只列已归档的 |
| `snoozed` | `true`、`false`（默认） | 按读那一刻的业务时钟 `now`：`snoozed_till > now` 为暂缓中；`false` 只列不在暂缓中的（含已到期的），`true` 只列暂缓中的 |
| `cursor`、`limit` | `shared` 的游标与页长（默认 50，最多 100） | 按 `(created_at DESC, id DESC)`；游标是这一页最后一行的 `[created_at, id]`（模块自己的游标类型，`shared.EncodeCursor`） |

- 各条件合取，另加 §2.5 的可见过滤（在 SQL 的 `LIMIT` 之前）。
- 参数的值不合法（非法值、重复的 `type` 成员）422（字段错误 `invalid_format` 或 `not_allowed`）；未知参数、重复的标量参数照草稿列表的口径 422；游标不合法 400（`shared.InvalidCursor`）；`limit` 越界 422。
- 一次请求之内，可见的集合与行在同一个只读快照里取（`ReadTx`，`listDrafts` 的先例）。跨页是键集分页：行的 `created_at` 是事件的时刻，迟到的分发写出的行可能落在已翻过的位置（交给 P4：不能靠第一页的游标判断有没有新通知）。同一事件的几条 `created_at` 相同，彼此按 id 稳定排序，不保证按变化的位置。

### 2.4 回答的形状与码

`Notification`：`id`、`workspace_id`、`project_id`、`issue_id`、`sender`（四种）、`triggered_by_id`（`null` 为 nerve 自己）、`data`、`read_at`、`archived_at`、`snoozed_till`、`created_at`。演员只给 id（同动态、评论）。

`data` 是发送时的快照（P1b spec §2.5）：`issue{id,name,sequence_id,identifier,state_name,state_group}` 照存。`change{field,verb,old_value?,new_value?,old_id?,new_id?,comment_id?,excerpt?}` 的 `old_id`/`new_id` 按白名单放行（R7）：

- `assignees`、`labels`、`state` 的照存（账户、标签、状态的 id）；
- 指向别的工作项的（`parent` 与关联的四种 `field`，§2.6）经 `IssueRefs` 解析：读的人此刻读得到的给出 id，并另带 `old_issue`/`new_issue`：`{id, sequence_id, identifier, name}`；读不到的这两对都是 `null`；
- 其余字段的 id 一律不给。

码（逐操作的 `x-problem-codes`，每个有处理器测试回答）：

| 操作 | 码 |
|---|---|
| `listNotifications` | `workspace.not_found`、`validation_failed`、`bad_request`（游标） |
| `getNotificationUnreadCount` | `workspace.not_found`、`validation_failed`（未知参数） |
| `markAllNotificationsRead` | `workspace.not_found`、`validation_failed` |
| `updateNotification` | `notification.not_found`、`validation_failed`、`server_busy` |

以及共有的认证码与请求体的 400（bodyshape：`read`、`archived` 给 `null`、类型不对）。新码 `notification.not_found` 进 `error-messages.ts` 与两份 `errors.json`。

### 2.5 读的人看得到的项目（读时的可见过滤）

- access 的 `NewReadableProjects(deps)`（照 `NewDecider`）给出 `Readable(ctx, actor, workspaceID) → []Grant{ProjectID, OwnOnly}`：调用者此刻对哪些项目有 `issue.read`、每个项目是否只看自己。事实由一次批量读给出：调用者在这个工作区里**有有效项目成员资格的、未删除的项目（含已归档的）**的事实与成员资格，加他的工作区角色；判定由 `access/domain` 里与 `Authorize` 同一个函数逐项目作出（P1b `DecideMany` 的做法，R1）。只接受不放行非成员的规则（`issue.read` 没有有效成员资格不放行，`decide.go` 的项目分支；照 `DecideMany` 拒绝 `AllowCreator` 的写法拒绝别的规则）。工作区管理员不是项目成员的读不到；在项目里是访客的工作区管理员照 M4 只看自己（项目没开 `guest_view_all_features` 时）。
- 等价测试照 `TestDecideManyIsAuthorizeOnTheMatrixSeed` 跑两遍：矩阵的原始种子上，对每个调用者、每个工作区，`Readable` 与逐项目调 `Authorize(issue.read)` 逐格一致；第二遍在副本上用 SQL 造出请求造不出的状态（项目单独删除而成员行还在、工作区成员行删除而项目成员行还在、撤权的人的项目成员行又变为有效、已归档的项目），造状态的辅助与 P1b 的那一遍共用。每一遍都要出现“放行、只看自己、拒绝”三种答案。
- 列表、未读数、全部已读三条 SQL 只看 `project_id` 在这个集合里的行（`(project_id, own_only)` 两个数组传入）；只看自己的项目里只看 `issue_created_by_id` 是调用者的行（R2）。行的工作区必须是 slug 的工作区。单条修改不写这条 SQL，经端口判定（§2.7）。
- 已删除的项目不在集合里（它的通知在 `notification.delete_rows` 软删之前也不列）；已删除的工作项的通知在删除事务里已软删（P1b）。

### 2.6 另一端的工作项

- 指向别的工作项的变化：`field = parent`，以及关联的四种 `field`（`relates_to`、`duplicate`、`blocking`、`blocked_by`，P1b spec §2.3）。一个 bootstrap 测试钉住 issue 的每一种关联类型都在这个集合里（变异：去掉一种，测试红）。
- issue 交出 `IssueRefs(ctx, reader, grants, ids) → map[id]IssueRef{ID, ProjectID, SequenceID, Name}`（照 `module_notification_issues.go` 的交出方式）：`ids` 中未删除的、在 `grants` 的项目里、`HiddenFrom` 不隐藏的工作项；已归档的工作项照样给出（它读得到，R4）。另一端在别的工作区的不在集合里，读不到。
- 项目标识：bootstrap 对一页里去重后的项目 id 做一次批量读（project 加一条按 id 数组读未删除项目标识的查询）；页里没有引用时不读。一页一次 `IssueRefs`、一次标识的读。
- 列表与单条修改的回答经它解析；未读数、全部已读不解析。

### 2.7 修改与全部已读

**单条修改** `PATCH /notifications/{notification_id}`，请求体 `{read?, archived?, snoozed_till?}`，省略的不改；`{}` 合法（不写，回答当前行）。次序（R5，不透露存在）：

1. 不加锁读行：`id`、`receiver_id = 调用者`、`deleted_at IS NULL`；没有即 404 `notification.not_found`。
2. 以行的工作区 `Authorize(notification.write)`；`ErrNotVisible` 即同一个 404。
3. 取 §2.5 的可见集合；行的项目不在其中，或只看自己而 `GuestRule.HiddenFrom(issue_created_by_id, 调用者, true)` 为真（P1b 的端口，不写 SQL 副本），即同一个 404。
4. 开事务（带锁的期限，与工作项的写同一个 `issue.write_lock_timeout`；等锁超时经 bootstrap 的映射回答 503 `server_busy`），`FOR NO KEY UPDATE` 锁这一行并复核收件人与未删除（没有即同一个 404：期间被删除）。
5. 读业务时钟 `now`（锁之后）。
6. 校验：`snoozed_till` 是时刻时须晚于 `now`（截到微秒后比较，identity 的 `CheckAPIToken` 先例），否则 422 `must_be_future`（字段 `snoozed_till`）；`null` 清空。所以别人的通知配过去的 `snoozed_till` 回答 404，不是 422。
7. 算新值：`read: true` 时 `read_at` 若为空写 `now`、已读的不变，`false` 清空；`archived` 同理作用于 `archived_at`。什么都没变的不写，回答锁下读到的行；变了的写 `updated_at = now`、`updated_by_id = 调用者`（P1b 的迁移头）。回答经 §2.6 解析。

**全部已读** `POST /workspaces/{slug}/notifications/mark-all-read`，请求体 `{tab?, type?, archived?, snoozed?}`（请求体里是真正的枚举与布尔，由 bodyshape 把关），含义与列表的同名参数相同；`read` 不在其中：只作用于未读的。

- 请求开始时读一次业务时钟 `now` 与 §2.5 的可见集合（R5：`now` 同时是界，各批共用一个时刻），之后的批不重算。
- 只标可见、未读、未删除、事件时刻 `created_at <= now`、符合筛选的行。事件早于 `now` 而分发在批循环结束前才写出的行可能被标（与用户点击前一刻才到的通知同一类，登记为已知性质）；事件晚于 `now` 的不标。
- 分批：每批一个事务，带锁的子查询取 `id > 上一批最大 id` 的至多 1000 行（常量，不进配置）按 `ORDER BY id` `FOR NO KEY UPDATE SKIP LOCKED`，条件（未读、未删除、筛选）写在带锁的子查询里，再更新 `read_at = now`、`updated_at = now`、`updated_by_id = 调用者`，`RETURNING id`；一批不足 1000 行即止。语句头写明不变式：全部已读从不等待行锁（R5）。被跳过的行只有两种：正被软删的（不必标）、正被单条修改或另一次全部已读持有的（毫秒级，不保证标到）。
- 回答 `{marked}`：各批实际更新的行数之和。某一批失败时回答错误，已提交的批不回滚（再请求一次标其余的）。

### 2.8 动作与权限矩阵

- `notification.read`、`notification.write`：工作区级，管理员、成员、访客（设计 §3.10）；`notification` 移出 `actionlessModules`，`moduleActions()` 加它（P1b R2）。规则表的单元测试加两行。
- 权限矩阵的行只用 `workspaceColumns`（管理员、成员、访客、从未加入、已移除、已删除工作区）：项目级的列会让非 GET 的行被当作“项目上的写”去查项目锁（`writesOnAProject`）。
  - 种子：每个工作区列一条属于该列调用者的通知（`newSeeded` 里按列命名，`prepareMatrix` 插行，`workspaceOfRow` 认得 `notifications`）。访客的那条种在他读得到的工作项上（acme/private，或 acme/public 且 `issue_created_by_id` 是他）；管理员、成员的那条种在 acme/public。已删除工作区那一列的行在删除 gone 之前插入（草稿循环的位置），由删除等待它软删。
  - 单条修改两行：(a) 带 `prepare`（`prepareDraftPublisher` 的先例：在写格自己的库上给管理员、成员补 acme/public 的有效成员资格）：200、200、200、404、404、404；(b) 不带 `prepare` 的变体：管理员、成员 404 `notification.not_found`（工作区管理员与成员不是项目成员读不到），访客 200，后三列 404。
  - 列表、未读数、全部已读：`workspaceCells`（三种角色 200，后三列 404 `workspace.not_found`），各加 `check` 断言各列看到的条数、两个未读数与 `marked`（否则 200 的格子没有区分度）；全部已读的格子在各自的库副本上写。

## 3. 局部裁定

- **R1 读得到的项目由 access 给出**：没有现成端口（`VisibleIssueProjects` 不含已归档的项目，`HistoricalIssueProjects` 含已删除的项目与已删除的成员行）。照设计 §6.3 放在 access，与 `Authorize` 同一个判定函数，两遍的等价测试钉住；project 新加一条批量读（有效成员资格、未删除、含已归档）。
- **R2 只看自己在 SQL 里过滤**：分页与计数必须在 SQL 里过滤；硬规则 03 §4 说判定只有 `domain.Issue.HiddenFrom` 一个谓词。依据 M4 设计 3.4“列表和搜索：同一条规则，SQL 按各项目 `OwnOnly` 加 `created_by_id = $actor`”（issue 的 `list_issues.sql`、`search_issues.sql` 如此）：通知的三条 SQL 以 `(project_id, own_only)` 两个数组传入可见的集合（`ROWS FROM(unnest…)` 的先例），过滤 `NOT own_only OR issue_created_by_id = 调用者`（`issue_created_by_id` 为空时得 NULL、不选中，与 `HiddenFrom` 对未知创建者一致）。一个 bootstrap 等价测试在 {创建者为空、是调用者、是别人} × {只看自己、不是} 六格上比对三条 SQL 的取舍与经 issue 交出的 `HiddenFrom`；三条语句各一个变异（去掉过滤）。这是 issue 之外的第一处，T15 在硬规则 03 §4 登记为例外。单条修改走端口，不写副本。
- **R3 页签与未读数**：`tab=all` 不含提及、`mentions` 只含提及（照 Plane 的两个页签；网页按页签分别显示两个数，`top-navigation-root.tsx`）。设计 §3.4、§5 的 `{total, mentions}` 改名为 `{all, mentions}`：各是那个页签里未读、未归档、不在暂缓中的可见条数（Plane 的 `total` 也不含提及，名字有歧义）；只改名字，行为照 Plane，阶段记录向负责人说明。全部已读只作用于请求的页签与筛选（Plane 在“全部”页签连提及一起标，D5，不照搬）。
- **R4 另一端的可见**：读的人此刻读得到的另一端都给出编号与名字，已归档的工作项、已归档项目里的工作项也给（通知是历史，读得到就该认得出；与关联列表隐去已归档的另一端不同，那是工作项详情的现状视图）；读不到的给 `null`，连 id 也不给。
- **R5 通知的写与锁序**：两个写都不进写闸门、不取工作项与项目的锁，只锁通知行。单条修改锁一行，带锁的期限（`issue.write_lock_timeout`，超时 503）。全部已读用 `SKIP LOCKED`、从不等待：工作项删除在一个事务里对子树逐个工作项按 id 升序软删它的通知，整个事务的取锁次序不是全局升序，全部已读若按全局升序等锁会与它互等（预检 re-84 I1）。全部已读的时刻在请求开始时读一次（它同时是界，各批共用），不在最后一把锁之后读，这是硬规则 03 §4“锁后读时钟”的登记例外。`notifications` 的写者与锁序在 T15 登记进硬规则 03 §4：分发只插入；两种软删按 id 升序（同一事务内逐工作项）；单条修改锁一行；全部已读跳过被锁的行；清理照 P1b。
- **R6 暂缓不设定时任务**：读时按业务时钟算（设计 §3.4）；`snoozed=false` 的列表与未读数含已到期的暂缓行。
- **R7 回答里的 id 按白名单放行**：只有 `assignees`、`labels`、`state` 的 id 照存；工作项引用经解析；其余字段的 id 一律不给（新增一种关联类型而通知没跟上时不漏出 id）。已读、归档与暂缓三个状态相互独立，各自修改，没有迁移规则（Plane 相同）。

## 4. 验收标准

- 接口：四个操作的契约、生成物与处理器；逐操作的码各有处理器测试回答；`make gen-check` 无差异；网页客户端的测试照旧（参数都是字符串）。
- 组合测试（真实的 app，经接口）：
  - H2–H4 的接口证据：P1b 的各类写在列表里以 `Notification` 出现（字段、类型、快照、`triggered_by_id`），执行者不收，提及与关注的去重。
  - H5：两个页签、`type` 子集与 422、`read`、`archived`、`snoozed`、跨页无重无漏、游标稳定、`limit` 的边界与 422、坏游标 400、未知参数 422。
  - 可见：撤出项目、结束工作区成员、降为只看自己的访客（只列他建的工作项的通知）、管理员兼访客、私有项目的移出、已归档的项目照列、已删除的项目不列；另一端：读得到的给编号与名字（含已归档的）、读不到的两对都为 `null`、别的工作区的为 `null`；白名单之外的字段不给 id。
  - H6：单条的已读 / 未读、归档 / 取消、暂缓（将来、过去 422、`null`）、`{}`；四种 404（别人的、已删除的、别的工作区的、项目读不到的）与随机 id 逐字节相同；别人的通知配过去的 `snoozed_till` 是 404；暂缓到期（推进业务时钟）回到列表并计入未读；全部已读只作用于页签与筛选、事件晚于 `now` 的不标、跨批（造足 1000 行以上或批长可调）计数正确、迟到的行的行为写死。
  - H7：`{all, mentions}` 不计已归档、暂缓中、不可见的。
- 等价：`Readable` 与 `Authorize` 两遍逐格一致；R2 的六格乘三条 SQL；关联类型全在引用集合里。
- 交错：子树删除持有通知行时全部已读立即返回、标了其余的、`marked` 不含被持有的行、没有 40P01（变异：去掉 `SKIP LOCKED` 让它失败）；全部已读与项目删除的分批软删；单条修改与工作项删除（删除先提交则 404，修改先提交则随后被软删）；单条修改等锁超时 503。
- 权限矩阵：§2.8 的行与覆盖守卫；动作的规则表测试。
- e2e：H2–H7 的接口故事（PAT 与浏览器会话对等，H1 的写法），本机每条约 3 秒以内。
- 每个可见、幂等、锁的性质至少一个能让测试失败的变异。
- 测试时间：CI 第 0 路 Test 步的总增量（含 bootstrap 以外的包）≤ 90 秒，折合本机约 60 秒；两个检查点记本机两路与分支 CI 两路的实测。P1c 之后第 0 路若超过 870 秒，在阶段记录里向负责人提出加一路（设计 §9）。

## 5. 后续责任

- P4：通知页、卡片、未读数、关注按钮按本 spec 的形状重写；删除旧的 8 个地址与 `TNotification`，加 `old-notification-urls` 关键词规则；新通知的提示不能靠第一页的游标（§2.3）。
- 附件通知不带附件 id（P1b 的交接）：P4 若要链到文件再议。

## 6. 风险

- 测试时间：权限矩阵与组合测试会让 bootstrap 包更慢（第 0 路已在 780 秒左右）；按 §4 的预算，检查点各量一次。
- 可见集合每次读一次：调用者在一个工作区里的项目数有界，一次查询。
- 全部已读的批：每批对剩余未读按 id 排序一次（未读索引不含 id）；T5 记一次 1 万条未读时单批的耗时，不设门槛。
