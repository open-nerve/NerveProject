# M7：协作功能

> 状态：设计稿，待预检与负责人决定（§15）。
> 起点：`main@d5b5134e03436781d2efd6cb9a483f66e3f95a8f`（M6 完成之后 CI 的两项改动合入，main CI 43921013487 attempt 1 全绿）。
> 文档分支 `codex/m7-design`，根目录保持 main。负责人 2026-11-17 指示“先把这些清理干净，然后没有其他问题就启动 M7”；常规设计、预检、实现、deep-review、`--no-ff` 集成、CI 和归档照 M5、M6 的节奏连续推进；架构回归、跨模块协同调整或意外高风险发现，先提交已完成工作，再用大白话上报。

## 1. 目标、范围与输入

M7 让五件协作功能经真实接口工作：站内通知（关注工作项、@提及），需求收集箱（提交、分诊），保存的视图与跨项目的工作区列表，侧栏收藏，最近访问。行为照 Plane 一比一（[总体设计 §1.1](../v0-design.md#11-v0-保留的功能)），只修 Plane 的明显缺陷并登记差异。

不做：邮件与通知偏好（v0 删除，总体设计 §1.2）；收集箱的表单、邮件来源（社区版前端只建应用内条目，见 §3.5）；私有与锁定的视图（社区版写不了，见 §3.7）；工作区级的迭代、模块列表（M6-M7-03，见 §3.7）；首个投递任务的 CLI（M7 不加命令行，M5-04/M6-M7-08 原文转交 M8）；迭代、模块成为资产宿主（M6-M7-09 未触发，原文转交 M8）。

前置依据：[总体设计](../v0-design.md) §3、§5、§6.6、§7.2、§7.7、§8；[执行方法](../claude_to_gpt/02-method.md)；[硬规则](../claude_to_gpt/03-rules-and-pitfalls.md)；八份输入交接（§11）。Plane 的行为以只读的 `plane/` 为准，表结构以 `tools/plane-schema/plane-v1.4.2-schema.sql` 为起点。三份调研笔记（Plane、服务端、前端的事实与行号）在设计分支的台账 `.superpowers/sdd/M7-design/research/`，下文的 Plane 缺陷编号 D1–D39 指 Plane 调研笔记 §6。

### 1.1 设计起点事实

- **后端**：没有 `notification`、`navigation` 模块，也没有 `notifications`、`intake_issues`、`issue_views`、`user_favorites`、`user_recent_visits` 任何一张表；最新迁移 `00048`。已有的是：`issue_subscribers`（只有自动关注在写：创建者、负责人、评论者、能读的被提及者）与 `issue_mentions`（每次写描述时同步，评论的提及不存）；24 种领域事件、7 个订阅者，没有通知的订阅者；评论、链接事件的载荷里没有工作项 id，没有任何事件带“新提及了谁”；迭代转移、迭代/模块的归档、项目的归档不发布事件。每个项目建立时已有一个隐藏的 Triage 状态，工作项的写不能把工作项放进它，列表经 `JOIN states` 排除它，但搜索与单个读对它没有规则（今天没有 Triage 工作项）。`issue_views_view`、`intake_view` 两个开关只存不读。没有 `GET /workspaces/{slug}/issues`；列表引擎已能跨多个项目读（成员列表就是），但查询语言只认项目、成员两种范围。
- **前端**：五个领域都还是 Plane 的旧代码：6 个 service 的 27 个调用全打 Plane 的 `/api/workspaces/...`，没有一条关键词规则守着它们；项目视图列表永远在加载，两个视图详情页只有 `PageHead`，工作区视图、通知、收集箱一挂载就打旧地址 404，收藏菜单从不取数，首页的“最近”已被 M3 删除，未读数从不取数。用户今天就能点出 404 的入口有：侧栏“视图”、顶栏收件箱、项目导航的视图与收集箱、命令面板的视图命令，以及项目页、迭代页、模块页筛选行的“保存为视图”。订阅的界面已在 M4 删除。
- **关键词守卫**：`phase` 为 `M6/closeout`；`copy-hook` 的 4 个例外 `until: "M7/closeout"`；`getTabIndex`/`ETabIndices` 只剩 M7 的 5 个文件在读，`projectMemberIds` 只剩 M7 的 3 个读者——M7 是最后的迁移者，删除全套。
- **Plane 的可沿用与不可沿用**：见 §3 每节的“Plane”一句。不照搬的缺陷：暂缓永不到期（D1、D2、D16）、“关注”类型的负责人排除失效（D3）、全部已读与列表的筛选对不上（D5）、`assigned` 的判定（D14）、提及的遗漏与重复（D8–D10）、看不到工作项的人照收通知（D11）、分诊的前后端权限不一致（D15）与没有状态迁移规则（D17、D18）、视图的 PUT 与新建不鉴权（D24–D26）、收藏写入不校验目标与连带不一致（D31、D32）、最近访问不按访问时间排序与上限失守（D35–D37）。

## 2. 用户故事

`H` 是 M7 的故事编号。每条有接口与数据库证据，页面故事另有浏览器证据；PAT 与浏览器登录对等。

| 编号 | 行为及有区分度的拒绝路径 |
|---|---|
| H1 | 关注、取消关注一个工作项，详情带 `is_subscribed`；读得到才能关注（只看自己的访客只对自己建的）；看不到的 404 |
| H2 | 通知的生成：字段修改、评论的增改删、链接、附件、关系、归档与恢复、自动归档（执行者为 nerve 自己）、创建与负责人；执行者自己不收；描述修改、迭代/模块的加入移出与转移、表情不通知；删除工作项不发通知，它的通知随之不再列出 |
| H3 | @提及：描述或评论里新提及的人收到“提到了你”，同一次写不再收关注通知；同一个人在一次写里不重复收；读不到这个工作项的人不收 |
| H4 | 收件人在发送那一刻按读权限复核：离开项目、成为只看自己的访客、关注后被移出私有项目的人不收 |
| H5 | 通知列表：“全部 / 提及”两页签、类型（分配给我 / 我创建的 / 我关注的）、未读、已归档、已暂缓；游标分页；只列他现在看得到的项目里的通知 |
| H6 | 单条的已读 / 未读、归档 / 取消、暂缓 / 取消；暂缓到期即回到收件箱并计入未读；全部已读只作用于当前页签与筛选 |
| H7 | 未读数（总数、提及数）在侧栏与顶栏，不计已归档和仍在暂缓期的 |
| H8 | 项目管理员、成员、访客向收集箱提交；开关关闭时被拒；条目处在 Triage，项目的列表、看板、搜索、工作区列表都没有它；提交时的负责人、提及照常关注与通知 |
| H9 | 收集箱列表：未处理（待处理 + 暂缓）/ 已处理（接受、拒绝、重复）、筛选与排序、分页；只看自己的访客只看到自己提交的；侧栏的待处理数 |
| H10 | 分诊：接受（移到默认状态、写一条状态动态、关注者收到通知）、拒绝、暂缓（到期回到待处理）、标重复（同项目里看得到的、不是收集箱条目、不是自己）；不合迁移规则的被拒；按 §15 B 的角色 |
| H11 | 条目的内容：详情复用工作项详情（评论、动态、表情、链接、附件、描述版本）；修改只限管理员与提交者（访客提交者只改标题、描述）；经工作项接口改、归档、删除、关联条目被拒；删除条目 |
| H12 | 项目视图：建、改、删、列、取；按 §3.7 的权限；开关关闭时写被拒、读照常；视图页按视图的筛选与显示列出工作项（四种布局、分页），“另存为新视图”与“更新视图” |
| H13 | 工作区跨项目列表：所有、分配给我、我创建的、我关注的四个默认视图与已存的工作区视图；逐项目的可见与只看自己；统一游标；规模 |
| H14 | 工作区视图的建、改、删与权限 |
| H15 | 收藏项目、迭代、模块、视图与文件夹：写入校验目标（存在、同工作区、他看得到）；文件夹只有一层、同名被拒；拖动排序与进出文件夹；看不到的不显示 |
| H16 | 收藏的连带：目标删除或归档时所有人的收藏移除，项目归档移除它里面的全部收藏，恢复不加回 |
| H17 | 最近访问：页面显式记录工作项与项目；每人每工作区最多 20 条、按访问时间、同一目标只一条；只列看得到的；GET 没有副作用 |
| H18 | 删除图：删除工作项、项目、工作区时 M7 的行立即不可见，60 天后有界清理 |
| H19 | 权限矩阵：管理员、成员、访客（含只看自己的访客）、工作区管理员兼访客、私有项目、撤权 |
| H20 | 页面：通知页、收集箱页、视图页、工作区视图、侧栏收藏、首页最近；换账号、工作区、项目，后退前进，迟到的回答不写进新页面；弹窗、下拉框的键盘、读屏、焦点；复制失败有提示 |
| H21 | 收尾：没有旧服务、地址、类型与到期的关键词例外；`getTabIndex`、`projectMemberIds` 全套删除；死成员与告警收敛；完整对等验收 |

## 3. 业务模型与权限

### 3.1 关注与提及

- **自动关注**沿用 M4（M4 设计 3.12，plane-diff“自动关注”行）：创建者、实际被指派的负责人、评论者、被提及而能读这个工作项的人。收集箱的提交走同一条路径（§3.5），所以提交时的负责人与提及照常关注——Plane 因为请求体嵌套而丢掉了它们（D10）。
- **手动关注**：`POST /issues/{id}/subscribe`、`POST /issues/{id}/unsubscribe`，读得到这个工作项（`issue.read`，只看自己的访客只对自己建的）就能做，看不到 404；重复关注、取消没关注的都是成功的空操作。取消关注软删那一行；之后再被指派、评论或被提及会按自动关注重新加入（Plane 是做任何产生通知的事都重新加入，D12）。详情（`IssueDetail`）带 `is_subscribed`。Plane 的“替别人增删关注”的接口前端不用，不做。
- **提及**沿用 M4：描述的提及存 `issue_mentions`，评论的提及不存。通知需要的是“这一次写新提及了谁”：issue 在写描述、写评论时已经算出来，M7 让它进入事件（§6.1），通知模块不解析 HTML。

### 3.2 通知的生成

通知由领域事件触发，在事务提交之后由任务生成（总体设计 §6.6）。一条通知 = 一个收件人 × 一个可通知的变化，照 Plane 的粒度（Plane 是每条动态一行，D6；Nerve 的 `issue.updated` 本来就是一条动态一个变化）。

| 事件 | 可通知的变化 | Plane |
|---|---|---|
| `issue.created` | 创建本身一条；每个负责人一条 | 相同 |
| `issue.updated` | 名字、父工作项、优先级、状态、起止日期、标签、负责人；不含描述（只有它带来的提及）、`sort_order`、迭代、模块 | 相同（Plane 跳过描述、迭代、模块） |
| `issue.archived`、`issue.unarchived` | 一条；自动归档的执行者为空，页面显示站点名 | Plane 以项目创建者为执行者 |
| `issue.comment_created/updated/deleted` | 一条，带评论 id 与发送时的评论正文（§3.4） | 相同 |
| `issue.link_created/updated/deleted`、`asset.attachment_added/deleted` | 一条 | 相同 |
| `issue.relation_added/removed` | 只在发起的那个工作项上一条 | 相同（另一端的动态被跳过） |
| `issue.deleted` | 不通知；这个工作项的通知随之软删（§4.3） | Plane 的任务对已删的工作项抛错，从不生成（D7） |
| 收集箱的接受 | 状态变化一条（§3.5 写了动态并发布 `issue.updated`） | Plane 不写动态、不通知（D18） |
| 迭代转移、迭代/模块删除带来的移出、表情、草稿 | 不通知 | 相同（Plane 跳过迭代、模块、表情、草稿）；M6-M7-06 的回答是“不要” |

### 3.3 收件人与复核

对工作项 I 上执行者 A 的一次写：

- **提及通知**：这一次新被提及的人（描述的新提及、评论的新提及），去掉 A。
- **关注通知**：I 的关注者（含这次写自动加入的），去掉 A、去掉这一次收到提及通知的人。
- **复核**：发送时，候选收件人按那一刻的读权限过滤（`issue.read` 与只看自己），一次批量的端口调用判定全部候选，不逐人调 `Authorize`（逐人是 O(人数) 次查询，关注者没有上限）。离开项目、降为只看自己的访客、被移出私有项目的人不收（Plane 不复核，D11）。
- **类型**（`sender`）：收件人是 I 的创建者为“我创建的”，否则是 I 当前的负责人为“分配给我”，否则为“我关注的”；提及另为“提及”。Plane 的“分配给我”要求创建者不是负责人（D14），不照搬。
- **快照**：通知存发送时的工作项名字、编号与状态（名、组），以及这个变化的字段、动词、旧值、新值（照 Plane 的 `data`）；评论通知另存评论正文的纯文本摘录。显示时不再读工作项。

### 3.4 通知的状态、列表、未读与全部已读

- 三个状态独立：`read_at`、`archived_at`、`snoozed_till`。暂缓按读那一刻的业务时钟算：`snoozed_till` 在将来为“暂缓中”，已过即回到收件箱（Plane 方向反了且永不到期，D1、D2）。不设定时任务。
- 列表 `GET /workspaces/{slug}/notifications`：页签 `tab=all|mentions`；`type` 为 `assigned,created,subscribed` 的子集（`all` 页签用）；`read=false` 只看未读；`archived=true` 只看已归档（默认不含）；`snoozed=true` 只看暂缓中（默认不含）。按 `(created_at, id)` 倒序游标分页。只列收件人此刻是有效成员的项目里的通知；只看自己的项目里只列他建的工作项的通知（读时过滤，通知行存工作项的创建者）。
- 未读数 `GET /workspaces/{slug}/notifications/unread-count`：`{total, mentions}`，不计已归档、暂缓中的，同样按可见过滤。
- 全部已读 `POST /workspaces/{slug}/notifications/mark-all-read`：请求体与列表的筛选同一组（`tab`、`type`、`archived`、`snoozed`），只作用于那组筛选选中的未读通知（Plane 的参数与列表对不上、“全部”页签会连提及一起标掉，D5）。
- 单条 `PATCH /notifications/{id}`：`read`、`archived`（布尔）、`snoozed_till`（将来的时刻或 `null`）。只有收件人自己，别人的 404。
- 保留期：通知不设保留期（Plane 相同）；软删的照 60 天清理。

### 3.5 收集箱：提交与分诊

- **一个项目一个收集箱，不建 `intakes` 表**：社区版每项目只用一个默认收集箱，名字、说明、图标都不显示；条目直接挂在项目上（§15 A）。
- **提交** `POST /projects/{id}/intake-issues`：项目的管理员、成员、访客；`intake_view` 关闭时 409（M6 开关的同一做法：写拒绝、读照常）。请求体照 Plane 前端实际发送的：名字（必填）、描述、优先级、标签、负责人、截止日期；状态由服务端设为项目的 Triage 状态，迭代、模块、父工作项、开始日期不接受。工作项经 issue 的新建路径写入（编号、动态、自动关注、提及、`issue.created` 事件照常），条目状态为“待处理”。只有应用内来源：Plane 的 `source`、`source_email`、`external_source`、`external_id`、`extra` 不建（M7-01，功能说明不改）。
- **状态与迁移**：`pending`（待处理）、`snoozed`（暂缓）、`accepted`（接受）、`declined`（拒绝）、`duplicate`（重复）。只有待处理与暂缓可以分诊：可到接受、拒绝、重复、暂缓（再暂缓改日期），暂缓可取消回到待处理；接受、拒绝、重复是终态，不能再改（Plane 没有任何迁移规则，D17）。暂缓要给将来的 `snoozed_till`，到期按读那一刻算回到待处理（列表、待处理数都把它算作待处理；Plane 永不到期，D16）。重复要给 `duplicate_to_id`：同项目、他看得到、不是收集箱条目、不是自己（Plane 不校验，D19）；只存不建“重复”关系（Plane 相同）。
- **接受**：工作项从 Triage 移到项目的默认状态（项目总有一个：默认状态不能删除，`project.state_default`，只经“设为默认”改变），在项目锁下读它，写一条普通的状态动态并发布 `issue.updated`，关注者收到状态通知（Plane 不写动态，D18）。拒绝、暂缓、重复只改条目，工作项留在 Triage、项目里看不到。
- **动态**：每次分诊写一条 `field = intake` 的动态，`verb` 为新状态的名字（Plane 用数字 `-1`、`0`、`1`、`2`，前端的 `inbox.tsx` 按数字分支；Nerve 用名字，渲染件随之改）；`issue_activities.verb` 的 CHECK 随之放宽（plane-diff 已预告）。分诊不通知（Plane 相同）。
- **删除** `DELETE /intake-issues/{id}`：项目管理员或提交者（提交者须仍是项目的有效成员）。未接受的连工作项一起删（工作项删除的完整连带）；已接受的只删条目，工作项留下（Plane 相同）。
- **待处理数**：一个工作区级的读给出每个看得到、开着收集箱的项目的待处理数（含已到期的暂缓；只看自己的访客只数自己的），供侧栏；不加进项目的回答（项目模块不读 issue 的表）。

### 3.6 收集箱条目作为工作项

条目就是一个处在 Triage 的工作项，所以：

- **读**照常：详情、按编号取、评论、动态、表情、链接、附件、描述版本，按工作项的读权限（只看自己的访客只看自己建的）。详情多带一个 `intake`（状态、暂缓到、重复目标），页面据此决定显示分诊界面；通知预览也凭它决定嵌入分诊界面（不需要 Plane 的 `is_inbox_issue`）。
- **评论、表情、链接、附件**按它们自己的规则照常可写（分诊时的讨论在这里）。
- **工作项的改、归档、删除**对 Triage 工作项一律 409 `issue.in_triage`：内容的修改走 `PATCH /intake-issues/{id}`（管理员与提交者；访客提交者只能改名字、描述，Plane 相同），删除走收集箱。它不能做父工作项、子工作项、关联的另一端或重复的目标，不能加入迭代、模块。
- **不出现**在项目列表、成员列表、工作区列表、搜索、关联与父工作项的选择器、迭代/模块进度里；自动归档只碰已完成、已取消的状态组，天然不碰它。
- **图片**：收集箱的新建弹窗维持现状不带图片（新建工作项的图片经草稿发布迁移，收集箱不走草稿，M5-M7-01）；建成之后条目就是工作项宿主，详情里的描述按工作项宿主照常带图片。登记差异。

### 3.7 视图与工作区跨项目列表

- **视图**：`issue_views` 一张表，`project_id` 为空即工作区视图。存名字、说明、图标、Nerve 格式的 `filters`、`display_filters`、`display_properties`、所有者。删去 Plane 的 `query`、`rich_filters`（plane-diff 已定）以及 `access`、`is_locked`、`sort_order`、`archived_at`、旧 `filters` 格式：社区版的序列化器把 `access`、`is_locked` 设为只读，所有视图都是公开、未锁（D27），前端也不按 `sort_order` 排序。筛选词按页面校验：项目视图用项目列表的词（含迭代、模块），工作区视图用工作区列表的词。
- **权限**：建：项目视图为项目管理员、成员（开关要开着），工作区视图为工作区管理员、成员（Plane 的新建不鉴权，D25）；改：只有所有者（Plane 相同，D29 照搬）；删：所有者或项目 / 工作区管理员；读：项目视图为项目的有效成员，只看自己的访客看不到别人的视图（也就是全部）；工作区视图为工作区的有效成员。Plane 的路由 PUT 绕过一切检查（D24），不做 PUT；看不到的 404（D26）。删除视图时所有人对它的收藏随之移除。
- **视图列表是集合型**，不分页（大小由成员的操作决定，同标签、状态），按 `created_at` 倒序、同值按 `id`；页面在本地排序、筛选（Plane 相同）。所以 M2 交接 §1 的“每种排序一个游标载荷”不适用。
- **视图页的工作项**：前端把视图的筛选与显示作为查询发给项目列表（Plane 相同，不另设“视图的工作项”接口）。改了筛选或显示而未保存时，改动只在这个页面里（Plane 相同）；“更新视图”（所有者）存回视图，“另存为新视图”新建。
- **工作区跨项目列表** `GET /workspaces/{slug}/issues`：工作区的管理员、成员、访客；范围是他是有效成员的全部未归档项目，逐项目执行只看自己；只读快照、统一游标与比较器（M4 交接 M7-03）。四个默认视图是同一接口加 `assignee_id=me`、`created_by=me`、`subscriber_id=me`（Plane 相同）。“我关注的”按关注行算，含自己建的、被指派的（Plane 的工作区视图就这样，D30 的另一个定义只在通知里）。工作区的词：项目、状态、状态组、优先级、标签、负责人、创建者、关注者、提及、起止日期；没有迭代、模块（没有工作区级的迭代、模块目录，M6-M7-03 的答复），目录取看得到的项目的并集。默认视图的显示设置照 Plane 存在浏览器（按账户、工作区、视图）。

### 3.8 收藏

- 只给工作区管理员、成员（Plane 相同，访客不能用）。类型恰为 `project`、`cycle`、`module`、`view`、`folder` 五种（M1-P3 交接）。
- **写入校验**：目标存在、在这个工作区、他此刻看得到（项目须是有效成员；迭代、模块、项目视图随项目；工作区视图须看得到）；同一目标只能收藏一次，再收藏原样返回。`parent` 只能是他自己的文件夹，文件夹不能放进文件夹；同一人在同一工作区的文件夹名不能重复（Plane 只在前端拦，D32）。`sequence` 按这个人算（Plane 按全工作区，D33）。
- **读** `GET /workspaces/{slug}/favorites`：他的全部收藏（集合型，含文件夹里的），每项带目标的名字、图标、所属项目；目标他现在看不到的不列（离开项目的、只看自己的访客、目标已删除的）。资源本身不带 `is_favorite`，页面由收藏列表算出（避免四个模块各自去读收藏表）。
- **连带**：目标删除或归档时，所有人对它的收藏移除——迭代删除、模块删除（Plane 只删当前用户的，D31）、迭代归档、模块归档、视图删除；项目删除或归档时移除这个项目里的全部收藏（项目自己、它的迭代、模块、视图）。恢复不加回（Plane 相同）。删除文件夹连它里面的一起删（Plane 相同）。

### 3.9 最近访问

- 页面显式记录：`POST /me/recent-visits`（`entity_type` 为 `issue` 或 `project`，`entity_id`），看得到才记录，否则 404；所有 GET 都没有副作用（总体设计 §3.2）。只记工作项与项目：Plane 记六种、只读出三种（D37），页面已在 M1 删除。
- 同一人、同一工作区、同一目标只有一行，重访只更新访问时间；按访问时间倒序（Plane 按建立时间，D35）；每人每工作区最多 20 条，写入时在同一个事务里裁掉更旧的（Plane 的 `== 20` 在并发下失守，D36）。
- 读 `GET /workspaces/{slug}/recent-visits?entity_type=`：集合型，每行带目标的摘要（工作项：名字、状态、优先级、负责人、编号、项目标识；项目：名字、图标、标识）；他现在看不到的、已删除的不列（Plane 不过滤，D39）。工作项删除不另外清理最近访问（读时过滤，20 条上限自然淘汰）。

### 3.10 权限

新的动作在 `access/domain/rules.go`：

| 动作 | 级别 | 角色 |
|---|---|---|
| `issue.subscribe`（关注、取消关注自己） | 项目 | 管理员、成员、访客（只看自己的访客只对自己建的） |
| `notification.read`、`notification.write`（自己的通知） | 工作区 | 管理员、成员、访客 |
| `intake.submit`、`intake.read` | 项目 | 管理员、成员、访客（只看自己的访客只看自己提交的） |
| `intake.update`（改条目的内容） | 项目 | 管理员；提交者（访客提交者只改名字、描述） |
| `intake.triage`（接受、拒绝、暂缓、重复） | 项目 | 按 §15 B |
| `intake.delete` | 项目 | 管理员；提交者（`AllowCreator`） |
| `view.read` / `view.create` / `view.update` / `view.delete`（项目视图） | 项目 | 读：三种角色；建：管理员、成员；改：所有者；删：管理员或所有者 |
| `workspace_view.*`、`workspace_issue.list` | 工作区 | 同上，按工作区角色；列表三种角色 |
| `favorite.read`、`favorite.write` | 工作区 | 管理员、成员 |
| `recent_visit.read`、`recent_visit.write` | 工作区 | 管理员、成员、访客 |

看不到的一律 404，看得到但没权限 403。工作区管理员兼项目访客照 M4 的规则（只看自己）。权限矩阵每个新操作一行。

### 3.11 项目开关与入口规则

`intake_view` 关闭：提交、分诊、条目的修改与删除 409 `issue.intake_disabled`，读照常，重新打开数据都在。`issue_views_view` 关闭：项目视图的建、改、删 409 `navigation.views_disabled`，读照常。侧栏、页头标签栏与命令面板用同一个谓词决定是否给出“视图”“收集箱”入口：开关开着、是项目的有效成员（三种角色，照 Plane）；新建视图另要求管理员或成员。仿 `planningEntryOn` 写 `viewsEntryOn`、`intakeEntryOn`，侧栏与命令面板共用，单测钉住访客与成员（M6-M7-07）。

## 4. 数据、模块归属与删除图

### 4.1 模块归属

- **`notification`**（新模块）：`notifications` 表，事件订阅者与分发任务，通知的接口。读工作项、关注者、提及与收件人权限经 issue 与 access 提供的端口。
- **`navigation`**（新模块）：`issue_views`、`user_favorites`、`user_recent_visits` 三张表与它们的接口。视图筛选词的校验经 issue 的 `ListSettings` 端口（M6 先例）；目标的存在、可见与摘要经 project、planning、issue 的批量端口。
- **收集箱放在 `issue` 模块**（§15 A）：`intake_issues` 表、Triage 工作项的新建路径、分诊、收集箱列表（列表引擎的一种新范围）与待处理数。条目与工作项一对一，每一次分诊都写工作项、动态、关注与事件，并在项目锁下进行；放进 issue 免去一条跨模块的写路径，与 M6 把归属表放进 issue 同理。总体设计 §6.2 目录结构里的 `intake/` 不建。
- 工作区跨项目列表是 issue 列表引擎的新范围 `WorkspaceList`。

### 4.2 表（迁移从 `00049` 起，`<v>_<模块>_<表>.sql`）

| 表 | 模块 | 要点 |
|---|---|---|
| `intake_issues` | issue | `workspace_id`、`project_id`、`issue_id`（`CASCADE`）、`status` CHECK 五值、`snoozed_till`、`duplicate_to_id`（`SET NULL`）、审计与软删列；部分唯一 `(issue_id) WHERE deleted_at IS NULL`（Plane 没有任何唯一，D22）；`status` 与暂缓、重复的 CHECK（暂缓须有日期、重复须有目标） |
| `issue_activities`（改） | issue | `verb` 的 CHECK 加入五个收集箱状态名，只在 `field = 'intake'` 时 |
| `notifications` | notification | `workspace_id`、`project_id`（`CASCADE`）、`receiver_id`、`triggered_by_id`（`SET NULL`，空 = nerve 自己）、`issue_id` 与 `issue_created_by_id`（不是外键，M4 设计 4.16）、`sender`、`data`（快照 JSON，形状由 CHECK 钉住）、`read_at`、`archived_at`、`snoozed_till`、`event_id` 与变化序号（部分唯一，任务重试幂等）；索引 `(receiver_id, workspace_id, created_at, id)`、`(workspace_id, issue_id)`。删去 Plane 的 `title`、`message`、`message_html`、`message_stripped`、`entity_name`（前端不读，只有工作项一种） |
| `issue_views` | navigation | 见 §3.7；`workspace_id`、`project_id`（`CASCADE`，可空）、`owned_by_id` |
| `user_favorites` | navigation | `workspace_id`、`project_id`（`CASCADE`，可空）、`user_id`、`entity_type` CHECK 五值、`entity_identifier`（不是外键）、`name`（文件夹）、`parent_id`（自引用 `CASCADE`）、`sequence`；部分唯一 `(user_id, entity_type, entity_identifier)` 与 `(user_id, workspace_id, lower(name)) WHERE entity_type = 'folder'` |
| `user_recent_visits` | navigation | `workspace_id`、`project_id`（`CASCADE`）、`user_id`、`entity_type` CHECK 两值、`entity_identifier`、`visited_at`；部分唯一 `(user_id, workspace_id, entity_type, entity_identifier)` |

每条指向 `workspaces`、`projects`、`issues` 的外键有不带条件的索引；指向账户的外键照 M2 设计 4.7 的图延伸（账户从不物理删除）。plane-diff 逐表登记。

### 4.3 删除图与清理

- **工作项删除**（每个 `issue.deleted`）：它的 `intake_issues` 随工作项在原事务里软删（issue 自己的表）；notification 的订阅者在原事务里软删它的通知（同 asset 的先例，按 `(workspace_id, issue_id)` 索引）；指向它的 `duplicate_to_id` 不动（读时只给看得到的目标），物理清理时 `SET NULL`。
- **`duplicate_to_id` 与删除目录守卫**：它是“引用”不是“拥有”，守卫的 `keysTo("issues")` 会把别的条目当作被删目标的子行。照四个图片引用外键的先例，在目录里登记为豁免并写明理由（一行，不改守卫的机制）。
- **项目、工作区删除**：notification 与 navigation 各加一个订阅者，在原事务里投递自己的分批任务（`notification.delete_rows`、`navigation.delete_rows`，M6 `planning.delete_rows` 的先例：保留原请求的时刻与删除者，提交后立即不可见）；`intake_issues` 随 issue 的分批任务。
- **60 天清理**：每张新的带 `deleted_at` 的表恰有一个清理者，`TestPurgeCatalogCoversTablesAndIncomingKeys` 的表数由 32 改为 37；清理顺序在 asset → issue → planning → project → workspace → identity 中插入 notification、navigation（在 issue 之前，它们不被别的表指向）。`TestPurgeScale` 与删除规模测试在 M7 代码上重跑。

## 5. 接口

一个模块一个文件：`api/modules/notification.yaml`、`api/modules/navigation.yaml`，收集箱、关注与工作区列表进 `api/modules/issue.yaml`。

| 接口 | 说明 |
|---|---|
| `POST /issues/{id}/subscribe`、`POST /issues/{id}/unsubscribe` | 204；`IssueDetail.is_subscribed` |
| `GET /workspaces/{slug}/notifications` | 游标分页，§3.4 的筛选 |
| `GET /workspaces/{slug}/notifications/unread-count` | `{total, mentions}` |
| `POST /workspaces/{slug}/notifications/mark-all-read` | 请求体为列表的筛选，回答标了几条 |
| `PATCH /notifications/{id}` | `read`、`archived`、`snoozed_till`，回答通知 |
| `POST /projects/{id}/intake-issues`、`GET /projects/{id}/intake-issues` | 提交；列表（`status` 与工作项的筛选、排序、游标，列表引擎的 `IntakeList` 范围） |
| `PATCH /intake-issues/{id}`、`DELETE /intake-issues/{id}`、`POST /intake-issues/{id}/triage` | 改内容、删除、分诊（`{status, snoozed_till?, duplicate_to_id?}`）；`{id}` 是工作项 id（Plane 相同） |
| `GET /workspaces/{slug}/intake-counts` | 集合型 `{project_id, pending}` |
| `GET /workspaces/{slug}/issues` | 工作区跨项目列表（`IssueListPage`） |
| `GET/POST /projects/{id}/views`、`GET/POST /workspaces/{slug}/views`、`GET/PATCH/DELETE /views/{id}` | 视图；列表集合型 |
| `GET/POST /workspaces/{slug}/favorites`、`PATCH/DELETE /favorites/{id}` | 收藏；改只接受 `parent_id`、`sequence`、文件夹的 `name` |
| `POST /me/recent-visits`、`GET /workspaces/{slug}/recent-visits` | 最近访问 |

错误按 RFC 9457、模块前缀的问题码，每个操作写明 `x-problem-codes`，每个新码进 `error-messages.ts` 与两份 `errors.json`。

## 6. 跨模块协作、事件与任务

### 6.1 事件的扩展（issue 的公开约定，随 M7/P1 改，测试同改）

- 评论的三个事件、链接的三个事件加一个 `issue` 变化（`NewID` 为工作项 id），与表情事件的写法相同。
- `issue.created`、`issue.updated`、评论的建立与修改，每个这次新被提及的人加一个 `mention` 变化（`NewID` 为账户）；新提及按 M4 的同步结果（描述）与评论的解析结果，未经可见性过滤（通知发送时自己复核）。
- 收集箱的接受发布普通的 `issue.updated`（状态变化）。
- planning 新增 `planning.cycle_archived`、`planning.module_archived`，project 新增 `project.archived`，供收藏的连带；恢复不发布。

### 6.2 通知的订阅者与任务

- 订阅者 `notify_on_issue_events`（notification 模块，在写入事务内）：只看 §3.2 表里的事件；没有可通知变化的（只拖了 `sort_order`、只改了迭代模块、只改描述且无新提及）不投递；有的投递一个 `notification.fanout` 任务，参数只带事件 id、类型、时刻、工作区、项目、工作项、执行者与去掉 HTML 的变化（描述、评论正文最大 50 万字，不进 `river_job`）。
- 任务（`default` 队列，显式 `Timeout` 与有限次数）：在一个只读快照里经端口读关注者、工作项快照、评论摘录，一次批量复核收件人，一条 `INSERT … ON CONFLICT DO NOTHING` 写入全部通知（`event_id` 加变化序号唯一，重试不重复）。按事件一个任务：批量路径（子树删除不通知；子工作项的移动、自动归档逐个事件）各自独立。
- 订阅者 `forget_deleted_issue_notifications`：`issue.deleted` 时在原事务里软删这个工作项的通知。

### 6.3 端口

- issue 给 notification：关注者与提及的读、工作项快照（名字、编号、状态名与组、创建者）、评论摘录、收件人的批量读权限判定。
- issue 给 navigation：视图筛选词的校验（项目页、工作区页两种词表）、最近访问与收藏用的工作项摘要与可见判定。
- project、planning 给 navigation：项目、迭代、模块的存在、可见与摘要（批量）。
- access 给 notification、navigation：调用者此刻看得到的项目集合与每个项目的只看自己。
- 所有端口由 bootstrap 适配，不在模块之间 import；跨模块的 SQL 联表一律不做（`TestSQLCSchemaScope` 不加例外）。

## 7. 前端

- **数据层**：五个领域的 store 与 service 重写为 v0 7.7 的形状：按会话分代、`useSessionSWR`（键带 `loginId`）、写入排队、`Reconciled`、地址只从路由参数、类型只来自 `@nerve/api-client`；删除 6 个旧 service、`intake-description-version.ts`、`TLegacyIssueUpdateRequest`、手写的 `IFavorite`、`TNotification`、`IProjectView` 等类型与两个模块级单例（M2 交接 §2）。旧地址各加一条 `old-…-urls` 关键词规则（M6 先例）。
- **通知**：通知页（页签、筛选、卡片、加载更多、预览：工作项浮层，条目时嵌分诊界面）；卡片按新通知的字段与类型重写，动作文字全部经 `t()`；顶栏与侧栏的未读数；工作项详情、浮层与命令面板的“关注 / 取消关注”。
- **收集箱**：页面、列表与筛选、详情（复用工作项详情的正文、评论、动态）、分诊页头（先拆 470 行的 `inbox-issue-header.tsx` 与 511 行的 store）、新建弹窗（按 `requestClose` 改关闭策略，M6-M7-01）、拒绝 / 删除 / 查重 / 暂缓弹窗；动态的 `inbox.tsx` 改按状态名；侧栏的待处理数。
- **视图**：项目视图列表与详情（列表引擎加 `PROJECT_VIEW` 一行，照 M6 的表驱动）、“另存为新视图”“更新视图”（接上 `enableUpdateView` 或删去）；工作区视图目录、默认四个与已存视图的列表（新的 `WORKSPACE` 范围与 `IssueListCommandHost` 的对话框宿主）；命令面板的视图菜单、搜索的视图组与 `create_view`。在视图接通之前，“保存为视图”这个今天会 404 的按钮在 P6 最先接通。
- **收藏与最近访问**：侧栏收藏（文件夹、拖动、移出）、项目 / 迭代 / 模块 / 视图的星与命令面板的收藏命令（M6-M7-02）；首页的最近访问（工作项、项目）与页面的显式记录（打开工作项详情、进入项目时各记一次，失败不打扰用户）。
- **交接的页面项**：M3 交接 §3、§4 的可访问性与地址规则（弹窗名、字段提示、嵌套按钮、行名控件、下拉框、Escape、复制经 `useCopyLink` 并删 4 个 `copy-hook` 例外）、`getTabIndex` 与两份 `tab-indices.ts` 的全套删除（M6-M7-05）、`projectMemberIds` getter 的删除（M3 §7）、`select-duplicate.tsx` 不再从 `useParams()` 取它所在路由没有的参数。

## 8. 运行与观察

- 新任务：`notification.fanout`（请求触发，`default` 队列）、`notification.delete_rows`、`navigation.delete_rows`；登记 `EnqueuedKinds` 并各有 worker、各自显式 `Timeout`（River 默认 1 分钟不作依据）。不加周期任务（暂缓按读时算）。
- 分发任务与链接抓取、分批删除共用 `default` 的 10 个 worker；不加新队列（加队列是平台改动）。每次写多一个 `InsertTx`，在项目锁与写闸门之内：P1 测量 M4 的写入基准前后差异，写进阶段记录。
- 保留期：通知不设（§3.4），最近访问有 20 条上限，收藏与视图随人的操作；M8 观察通知表的增长。

## 9. 验证策略

单元覆盖收件人、类型、暂缓、迁移规则、文件夹规则等纯规则；adapter 覆盖 SQL、约束、幂等与锁；`bootstrap` 组合覆盖认证、权限矩阵、事件到通知的全链、删除图与清理、全程序接口；Playwright 覆盖真实页面与数据库断言（§2 每条）。负例至少：看不到的工作项与项目、访客与只看自己、工作区管理员兼访客、私有项目、撤权后的通知与收藏、开关关闭、并发的两次提交与两次分诊、分诊与工作项删除交错、最近访问的并发记录与上限、跨工作区的收藏目标、换会话与迟到回答。工作区跨项目列表加规模测试（多项目、总量与项目列表的规模目标同级），通知的分发加大关注者数的规模测试。每个安全、可见、幂等性质至少有一个能让它失败的变异。

e2e 的时间：CI 的两片各 35 分钟、30 分钟报告线按片（负责人 2026-11-17）。每个页面阶段在阶段记录里写本段新增故事的本机时间与两片的预估；任一片的预估接近 25 分钟时，先向负责人提出加一片，不自行改作业结构。

## 10. 风险

| 风险 | 对策 |
|---|---|
| 每次工作项的写多一个通知的 `InsertTx`，在项目锁与写闸门之内 | 订阅者只投递、不读不算；没有可通知变化的不投；P1 测写入基准前后 |
| 工作区跨项目列表要在全部可见项目里先筛选、计数再分页 | P3a 的规模测试与执行计划核对；不达标先在 P3a 内修，不带着进页面阶段 |
| 扩展 issue 的事件改动 M4 的公开约定与它的测试 | 只加变化、不改已有变化；事件形状测试同改；asset、planning 的订阅者不受影响（它们不看新变化） |
| 关注者没有上限，一次分发的收件人可能很多 | 批量复核与一条 INSERT；规模测试钉住 |
| 前端五个领域约 144 个文件几乎全部重写，五个文件过 300 行 | 每个页面阶段先按职责拆大文件；每段 ≤ 16 Task，超限在实施前拆 |
| e2e 时间随页面故事增长 | §9 的预估与报告；加片须负责人同意 |

## 11. 输入交接与后续交接

| 输入 | 条款落点 | 关闭阶段 |
|---|---|---|
| [M1-P2-trim-content](handoffs/M1-P2-trim-content.md) | 收藏、最近访问没有页（§3.8、§3.9）；首页最近访问只列工作项、项目，读新接口；通知没有已删功能的字段 | P4、P7 |
| [M1-P3-trim-platform](handoffs/M1-P3-trim-platform.md) | 收藏的类型恰五种、前端用生成类型；通知没有工时 | P3b、P7 |
| [M1-closeout](handoffs/M1-closeout.md) | 本领域死成员与死 prop（首个页面阶段跑基线）、改到的文件 oxlint 为零并另清一类规则 | P4–P8 |
| [M2-closeout](handoffs/M2-closeout.md) | §1 视图列表是集合型（§3.7）；§2 会话分代与两个单例；§3 下拉框；§4 复制；§5 删除图（§4.3）；§6 不嵌 iframe、不改 CSP | P1–P3（后端）、P4–P8 |
| [M3-closeout](handoffs/M3-closeout.md) | §1 取数按权限与开关、S2 清单；§2 视图 store、通知预览的取数、布局的 key；§3 弹窗与键盘逐项；§4 地址规则、复制、文案、`projectMemberIds`、守卫阶段；§5–§7 回执 | P4–P8 |
| [M4-closeout](handoffs/M4-closeout.md) | M7-01 收集箱来源与系统执行者；M7-02 关注接口、发送时复核、事件消费；M7-03 视图在新引擎、工作区列表；M7-04 旧 DTO 与命令面板；M7-05 新页面的规则 | P1–P3、P4–P8 |
| [M5-closeout](handoffs/M5-closeout.md) | M5-M7-01 收集箱的图片（§3.6）；M5-M7-02 附件事件与只带 id；M5-M7-03 通知卡片的正文；M5-04 原文转交 | P1、P2、P4、P8 |
| [M6-closeout](handoffs/M6-closeout.md) | M6-M7-01 新建弹窗的关闭；M6-M7-02 迭代、模块的收藏与归档连带；M6-M7-03 视图的迭代、模块词与工作区目录（§3.7）；M6-M7-04 视图的 store 与读；M6-M7-05 `getTabIndex`；M6-M7-06 转移不通知（§3.2）；M6-M7-07 入口规则（§3.11）；M6-M7-08、M6-M7-09 原文转交 | P1–P8 |

输入交接各写“处理结果（M7/P<k>）”，保持 open 至收尾关闭。后续交接由 P8 写给 M8：对等验收、通知表的增长与分发耗时的观察、M5-04/M6-M7-08 与 M6-M7-09 原文。

## 12. 分阶段推进

每段先 spec、plan、预检，再逐 Task 实现，结束 deep-review 与修复复审；每段约 16 Task 以内，超限在实施前拆分。后续阶段只在前置事实落实后写可执行 plan。

| Phase | 交付边界 | 必须同时完成的验证 |
|---|---|---|
| P1 | 关注与通知的后端：issue 事件的扩展、关注接口与 `is_subscribed`、`notification` 模块与表、订阅者与分发任务、列表 / 修改 / 全部已读 / 未读数、删除图与清理、权限矩阵；关键词 phase 改为 `M7/P1` | H1–H7 的接口与数据库；幂等、复核、规模；写入基准前后 |
| P2 | 收集箱的后端（issue 模块）：`intake_issues`、Triage 的新建路径、收集箱列表、分诊与迁移规则、动态、条目的读写规则、待处理数、开关、删除图、权限矩阵 | H8–H11 的接口与数据库；交错 |
| P3a | `navigation` 模块与视图：`issue_views`、两种视图的接口与权限、工作区跨项目列表与规模测试 | H12–H14 的接口与数据库 |
| P3b | 收藏与最近访问：两张表与接口、planning / project 的归档事件与连带、可见过滤 | H15–H17 的接口与数据库 |
| P4 | 网页的通知与关注 | H1–H7 的页面、H20 的通知页 |
| P5 | 网页的收集箱 | H8–H11 的页面、H20 的收集箱页 |
| P6 | 网页的视图：项目视图、工作区视图、保存与更新视图、命令面板的视图 | H12–H14 的页面 |
| P7 | 网页的收藏与最近访问 | H15–H17 的页面 |
| P8 | 最终矩阵、清理规模重跑、死成员与告警收敛、`getTabIndex` 与 `projectMemberIds` 的删除核对、文档同步、M8 交接 | H1–H21 最终证据、deep-review |
| closeout | 首次 noff / main 绿 → §13 勾选与总体 §9.4 → 第二次 noff / main 绿 → 归档清理 | §13 每条证据、全部输入关闭 |

P1 起第一个实现提交把关键词 phase 改为 `M7/P1`；纯设计分支不改 phase。

## 13. 完成标准

- [ ] H1–H21 每条有与失败路径相区分的接口、数据库与页面证据。
- [ ] 通知的触发、收件人、复核、类型、快照按 §3.2、§3.3；状态、列表、未读与全部已读按 §3.4。
- [ ] 收集箱的提交、迁移规则、接受、动态、删除按 §3.5；条目作为工作项的读写边界按 §3.6。
- [ ] 视图与工作区跨项目列表按 §3.7，走同一列表引擎，Go 与前端同 fixture；规模测试通过。
- [ ] 收藏的校验、文件夹、连带按 §3.8；最近访问的记录、上限、排序、过滤按 §3.9。
- [ ] 权限矩阵与开关、访客与只看自己、私有项目、撤权全部通过；没有 Plane 的默认 PUT 与不鉴权的新建。
- [ ] 删除图延伸、事件与清理任务、60 天有界清理，`TestPurgeScale` 与删除规模测试在 M7 代码上重跑。
- [ ] 前端只调 `/api/v0`，类型来自 OpenAPI，stores 按会话分代，旧 service、类型、地址删除并有关键词规则守着。
- [ ] 交接中的键盘、弹窗、下拉框、地址、复制逐项在浏览器核对；本领域死成员与改到文件的告警为零，清一类规则。
- [ ] `copy-hook` 的 M7 例外删除，`getTabIndex`、`projectMemberIds` 全套删除；差异清单与前端改动清单同步。
- [ ] 所有输入交接 closed；M8 收到 M7-closeout，M5-04/M6-M7-08、M6-M7-09 原文未丢。
- [ ] 最终本地七项、deep-review 与修复复审、精确分支 / main CI 通过；历史失败诚实保留。
- [ ] 全阶段与收尾 noff 合入，完成标记第二次 noff 后 main 全绿，总体 §9.4 标完成。
- [ ] 原始证据完整归档并核验，分支与 worktree 正常清理、未知文件为零。

## 14. 进度与裁定

| 阶段 | 状态 | spec / plan / review |
|---|---|---|
| 设计 | 草稿，待预检与负责人决定（§15） | — |

## 15. 待负责人决定

预检之后提交负责人；未决定前不进入 P1。

- **A. 收集箱放进 `issue` 模块**（§4.1）：不建总体设计里的 `intake/` 模块，也不建 Plane 的 `intakes` 表，条目挂在项目上。推荐：是。理由：条目就是 Triage 工作项，每次提交、分诊都写工作项、动态、关注与事件并持项目锁；另立模块要为这些写各开一条跨模块的写路径，而它自己只有一张一对一的表。
- **B. 谁能分诊**（接受、拒绝、暂缓、标重复）：Plane 的后端只放项目管理员（以及是项目成员的工作区管理员），前端却给成员显示按钮，成员点了得 403（D15）。选项：(1) 只给项目管理员（照 Plane 后端）；(2) 管理员与成员（照 Plane 前端）。推荐 (1)：权限以后端为准，成员照样能提交、评论、改自己提交的条目。
- **C. 通知的规则**（§3.2–§3.4）：触发照 Plane 的集合；粒度照 Plane 一个变化一条；删除不发通知且旧通知随之隐藏；收集箱的接受写状态动态并通知关注者；发送时按读权限复核收件人；暂缓按读时到期；全部已读只作用于当前筛选；通知不设保留期。推荐：按此推进。
- **D. 其余取舍**按下列各条推进（设计中写在各节）：
  1. 新建 `notification`、`navigation` 两个模块，跨模块只走端口、不联表（§4.1、§6.3）。
  2. 扩展 issue 的事件：评论、链接事件带工作项 id，新提及作为变化发布（§6.1）。
  3. 关注接口：读得到就能关注；取消关注后再被指派、评论、提及会重新关注（§3.1）。
  4. 通知存发送时的快照，读时按项目可见与只看自己过滤（§3.3、§3.4）。
  5. 收集箱只有应用内来源，不建 Plane 的来源五列，功能说明不改（§3.5）。
  6. 收集箱的迁移规则：只有待处理、暂缓可以分诊，其余三种是终态（§3.5）。
  7. 重复的目标限同项目、看得到、不是条目、不是自己，只存不建关系（§3.5）。
  8. 开关关闭时写入 409、读取照常（§3.11，M6 同例）。
  9. Triage 工作项照常可读、可评论，经工作项接口的改、归档、删除与做关联目标一律拒绝（§3.6）。
  10. 收集箱的新建弹窗不带图片，建成之后按工作项宿主（§3.6）。
  11. 侧栏的待处理数由一个工作区级的读给出（§3.5）。
  12. 视图删去私有、锁定、排序、归档等社区版写不了或不用的列（§3.7）。
  13. 视图的建要求管理员或成员，改只限所有者，删为所有者或管理员（§3.7）。
  14. 视图列表是集合型、不分页（§3.7）。
  15. 工作区跨项目列表给三种角色、逐项目只看自己，四个默认视图是它加一个条件，没有迭代、模块词（§3.7）。
  16. 默认视图的显示设置照 Plane 存在浏览器（§3.7）。
  17. 收藏只给管理员、成员；写入校验目标；文件夹一层与同名在服务端拒绝；序号按人（§3.8）。
  18. 收藏的连带统一为移除所有人的，归档也移除，恢复不加回（§3.8）。
  19. 资源不带 `is_favorite`，页面由收藏列表算（§3.8）。
  20. 最近访问只记工作项、项目，一目标一行、按访问时间、写时裁到 20 条，读时按可见过滤（§3.9）。
  21. 视图、收集箱的入口：开关开着、项目的三种角色，侧栏与命令面板共用一个谓词（§3.11）。
