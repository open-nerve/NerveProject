# M7：协作功能

> 状态：设计已预检（2026-11-17，[re-81](../../reviews/re-81-M7设计预检.md)，Ready after fixes，修正已改入本文）；负责人 2026-11-17 决定 §15 的 A–D 都按推荐（见 §15），其余取舍 E 照设计推进。
> 起点：`main@d5b5134e03436781d2efd6cb9a483f66e3f95a8f`（M6 完成之后 CI 的两项改动合入，main CI 43921013487 attempt 1 全绿）。
> 文档分支 `codex/m7-design`，根目录保持 main。负责人 2026-11-17 指示“先把这些清理干净，然后没有其他问题就启动 M7”；常规设计、预检、实现、deep-review、`--no-ff` 集成、CI 和归档照 M5、M6 的节奏连续推进；架构回归、跨模块协同调整或意外高风险发现，先提交已完成工作，再用大白话上报。

## 1. 目标、范围与输入

M7 让五件协作功能经真实接口工作：站内通知（关注工作项、@提及），需求收集箱（提交、分诊），保存的视图与跨项目的工作区列表，侧栏收藏，最近访问。行为照 Plane 一比一（[总体设计 §1.1](../v0-design.md#11-v0-保留的功能)），只修 Plane 的明显缺陷并登记差异。

不做：邮件与通知偏好（v0 删除，总体设计 §1.2）；收集箱的表单、邮件来源（社区版前端只建应用内条目，见 §3.5）；私有与锁定的视图（社区版写不了，见 §3.7）；工作区级的迭代、模块列表，以及依赖它们的工作区视图的迭代、模块词和命令面板在工作区范围搜迭代、模块（M6-M7-03，§15 D4）；首个投递任务的 CLI（M7 不加命令行，M5-04/M6-M7-08 原文转交 M8）；迭代、模块成为资产宿主（M6-M7-09 未触发，原文转交 M8）。

前置依据：[总体设计](../v0-design.md) §3、§5、§6.6、§7.2、§7.7、§8；[执行方法](../claude_to_gpt/02-method.md)；[硬规则](../claude_to_gpt/03-rules-and-pitfalls.md)；八份输入交接（§11）。Plane 的行为以只读的 `plane/` 为准，表结构以 `tools/plane-schema/plane-v1.4.2-schema.sql` 为起点。三份调研笔记（Plane、服务端、前端的事实与行号）在设计分支的台账 `.superpowers/sdd/M7-design/research/`，下文的 Plane 缺陷编号 D1–D39 指 Plane 调研笔记 §6。

### 1.1 设计起点事实

- **后端**：没有 `notification`、`navigation` 模块，也没有 `notifications`、`intake_issues`、`issue_views`、`user_favorites`、`user_recent_visits` 任何一张表；最新迁移 `00048`。已有的是：`issue_subscribers`（只有自动关注在写：创建者、负责人、评论者、能读的被提及者）与 `issue_mentions`（每次写描述时同步，评论的提及不存）；21 种领域事件（issue 15、planning 2、asset 2、project 1、workspace 1）、7 个订阅者，没有通知的订阅者；评论、链接事件的载荷里没有工作项 id，没有任何事件带“新提及了谁”；迭代转移、迭代/模块的归档、项目的归档不发布事件。每个项目建立时已有一个隐藏的 Triage 状态，工作项的写不能把工作项放进它（`pickState`），列表经调用方给的状态目录排除它，但搜索与单个读对它没有规则（今天没有 Triage 工作项）。`issue_views_view`、`intake_view` 两个开关只存不读。没有 `GET /workspaces/{slug}/issues`；列表引擎已能跨多个项目读（成员列表就是），但查询语言只认项目、成员两种范围，并且先筛选、计数再分页。
- **前端**：五个领域都还是 Plane 的旧代码：6 个 service 的 27 个调用全打 Plane 的 `/api/workspaces/...`，没有一条关键词规则守着它们；项目视图列表永远在加载，两个视图详情页只有 `PageHead`，工作区视图、通知、收集箱一挂载就打旧地址 404，收藏菜单从不取数，首页的“最近”已被 M3 删除，未读数从不取数。用户今天就能点出 404 的入口有：侧栏“视图”、顶栏收件箱、项目导航的视图与收集箱、命令面板的视图命令，以及项目页、迭代页、模块页筛选行的“保存为视图”。订阅的界面已在 M4 删除；分诊状态的前端已在 M3 删除。
- **关键词守卫与测试预算**：`phase` 为 `M6/closeout`；`copy-hook` 的 4 个例外 `until: "M7/closeout"`；`getTabIndex`/`ETabIndices` 只剩 M7 的 5 个文件在读，`projectMemberIds` 只剩 M7 的 3 个读者——M7 是最后的迁移者，删除全套。CI 的 Go 测试分两路，`bootstrap` 包在运行器上已用 919–1185 秒、逼近 20 分钟的整包期限；e2e 分两片、每片 35 分钟；改路数、片数都须负责人同意。
- **Plane 的可沿用与不可沿用**：见 §3 每节的“Plane”一句。不照搬的缺陷：暂缓永不到期（D1、D2、D16）、“关注”类型的负责人排除失效（D3）、全部已读与列表的筛选对不上（D5）、`assigned` 的判定（D14）、提及的遗漏与重复（D8–D10）、看不到工作项的人照收通知（D11）、分诊的前后端权限不一致（D15）与没有状态迁移规则（D17、D18）、视图的 PUT 与新建不鉴权（D24–D26）、收藏写入不校验目标与连带不一致（D31、D32）、最近访问不按访问时间排序与上限失守（D35–D37）。

## 2. 用户故事

`H` 是 M7 的故事编号。每条有接口与数据库证据，页面故事另有浏览器证据；PAT 与浏览器登录对等。

| 编号 | 行为及有区分度的拒绝路径 |
|---|---|
| H1 | 关注、取消关注一个工作项，详情带 `is_subscribed`；读得到才能关注（只看自己的访客只对自己建的）；看不到的 404 |
| H2 | 通知的生成：字段修改、评论的增改删、链接、附件、关系、归档与恢复、自动归档（执行者为 nerve 自己）、创建与负责人；执行者自己不收；描述修改、迭代/模块的加入移出与转移、表情不通知；删除工作项不发通知，它的通知随之不再列出（含与分发任务交错时） |
| H3 | @提及：描述或评论里新提及的人收到“提到了你”，同一次写不再收关注通知；同一个人在一次写里不重复收；读不到这个工作项的人不收 |
| H4 | 收件人在发送那一刻按读权限复核：离开项目、成为只看自己的访客、关注后被移出私有项目的人不收；通知里另一端的工作项（父、关联）按读的人此刻的可见显示 |
| H5 | 通知列表：“全部 / 提及”两页签、类型（分配给我 / 我创建的 / 我关注的）、未读、已归档、已暂缓；游标分页；只列他现在看得到的项目里的通知 |
| H6 | 单条的已读 / 未读、归档 / 取消、暂缓 / 取消；暂缓到期即回到收件箱并计入未读；全部已读只作用于当前页签与筛选 |
| H7 | 未读数（“全部”页签的、提及的）在侧栏与顶栏，不计已归档和仍在暂缓期的 |
| H8 | 项目管理员、成员、访客向收集箱提交；开关关闭时被拒；条目处在 Triage，项目的列表、看板、搜索、工作区列表都没有它；提交时的负责人、提及照常关注与通知 |
| H9 | 收集箱列表：未处理（待处理 + 暂缓）/ 已处理（接受、拒绝、重复）、筛选与排序、分页；只看自己的访客只看到自己提交的；侧栏的待处理数 |
| H10 | 分诊：接受（移到默认状态、写一条状态动态、关注者收到通知）、拒绝、暂缓（到期回到待处理）、标重复（同项目里看得到的、不是收集箱条目、不是自己）；不合迁移规则的被拒；只有项目管理员能分诊（§15 B） |
| H11 | 条目的内容：详情复用工作项详情（评论、动态、表情、链接、附件）；修改只限待处理、暂缓时的管理员与提交者（访客提交者只改标题、描述），描述的图片同此；经工作项接口改、归档、删除、关联条目被拒；删除条目 |
| H12 | 项目视图：建、改、删、列、取；按 §3.7 的权限；开关关闭时写被拒、读照常；视图页按视图的筛选与显示列出工作项（四种布局、分页），“另存为新视图”与“更新视图” |
| H13 | 工作区跨项目列表：所有、分配给我、我创建的、我关注的四个默认视图与已存的工作区视图；逐项目的可见与只看自己；统一游标；规模 |
| H14 | 工作区视图的建、改、删与权限（访客看不到已存的工作区视图） |
| H15 | 收藏项目、迭代、模块、视图与文件夹：写入校验目标（存在、同工作区、他看得到、未归档）；文件夹只有一层、同名被拒；拖动排序与进出文件夹；看不到或已归档的不显示 |
| H16 | 收藏的连带：目标删除或归档时所有人的收藏移除，项目归档移除它里面的全部收藏，恢复不加回 |
| H17 | 最近访问：页面显式记录工作项与项目；每人每工作区最多 20 条、按访问时间、同一目标只一条；工作项删除时随之移除；只列看得到的；GET 没有副作用 |
| H18 | 删除图：删除工作项、项目、工作区时 M7 的行立即不可见，60 天后有界清理 |
| H19 | 权限矩阵：管理员、成员、访客（含只看自己的访客）、工作区管理员兼访客、私有项目、撤权 |
| H20 | 页面：通知页、收集箱页、视图页、工作区视图、侧栏收藏、首页最近；换账号、工作区、项目，后退前进，迟到的回答不写进新页面；弹窗、下拉框的键盘、读屏、焦点；复制失败有提示 |
| H21 | 收尾：没有旧服务、地址、类型与到期的关键词例外；`getTabIndex`、`projectMemberIds` 全套删除；死成员与告警收敛；完整对等验收 |

## 3. 业务模型与权限

### 3.1 关注与提及

- **自动关注**沿用 M4（M4 设计 3.12，plane-diff“自动关注”行）：创建者、实际被指派的负责人、评论者、被提及而能读这个工作项的人。收集箱的提交走同一条路径（§3.5），所以提交时的负责人与提及照常关注——Plane 因为请求体嵌套而丢掉了它们（D10）。
- **手动关注**：`POST /issues/{id}/subscribe`、`POST /issues/{id}/unsubscribe`，判定用 `issue.read`（只看自己的访客只对自己建的），看不到 404；重复关注、取消没关注的都是成功的空操作。取消关注软删那一行；之后再被指派、评论或被新提及会按自动关注重新加入（Plane 是做任何产生通知的事都重新加入，D12）；描述的保存、评论的修改只让新被提及的人自动关注，仍被提及的人不会因此被加回（[P1a spec](specs/P1a-subscriptions-events.md) R7）。不进写闸门、不取项目锁：在一个事务里以 `FOR SHARE` 锁住工作项行、确认未删除再写，与工作项的删除串行（删除的事务软删关注行，迟到的关注不会漏在已删工作项上）；这是硬规则 §4 的一个写明理由的例外（[P1a spec](specs/P1a-subscriptions-events.md) §2.4）。详情（`IssueDetail`）带 `is_subscribed`。Plane 的“替别人增删关注”的接口前端不用，不做。
- **提及**沿用 M4：描述的提及存 `issue_mentions`，评论的提及不存。通知需要的是“这一次写新提及了谁”，由 issue 算出并放进事件（§6.1），通知模块不解析 HTML：描述是 M4 同步时已有的新增集合；评论的建立是正文里的全部提及，评论的修改是新正文的提及减去旧正文的提及。

### 3.2 通知的生成

通知由领域事件触发，在事务提交之后由任务生成（总体设计 §6.6）。一条通知 = 一个收件人 × 一个可通知的变化，照 Plane 的粒度（Plane 是每条动态一行，D6；Nerve 的 `issue.updated` 本来就是一条动态一个变化）。

| 事件 | 可通知的变化 | Plane |
|---|---|---|
| `issue.created` | 创建本身一条；每个负责人一条 | 相同 |
| `issue.updated` | 名字、父工作项、优先级、状态、起止日期、标签、负责人；不含描述（只有它带来的提及）、`sort_order`、迭代、模块 | 相同（Plane 跳过描述、迭代、模块） |
| `issue.archived`、`issue.unarchived` | 一条；自动归档的执行者为空，页面显示站点名 | Plane 以项目创建者为执行者 |
| `issue.comment_created`、`issue.comment_updated` | 一条，带评论 id 与发送时评论正文的纯文本摘录 | 相同 |
| `issue.comment_deleted` | 一条，不带摘录（评论已不在） | 相同 |
| `issue.link_created/updated/deleted`、`asset.attachment_added/deleted` | 一条 | 相同 |
| `issue.relation_added/removed` | 只在发起的那个工作项上一条 | 相同（另一端的动态被跳过） |
| `issue.deleted` | 不通知；这个工作项的通知随之软删（§4.3） | Plane 的任务对已删的工作项抛错，从不生成（D7） |
| 收集箱的接受 | 状态变化一条（§3.5 写了动态并发布 `issue.updated`） | Plane 不写动态、不通知（D18） |
| 迭代转移、迭代/模块删除带来的移出、表情、草稿 | 不通知 | 相同（Plane 跳过迭代、模块、表情、草稿）；M6-M7-06 的回答是“不要” |

### 3.3 收件人与复核

对工作项 I 上执行者 A 的一次写：

- **提及通知**：这一次新被提及的人（§3.1），去掉 A。
- **关注通知**：I 的关注者（含这次写自动加入的），去掉 A、去掉这一次收到提及通知的人。
- **复核**：发送时，候选收件人按那一刻的读权限过滤。判定只有一个来源：access 给出“这些账户在这个项目里的角色与只看自己”，由 `access/domain` 里与 `Authorize` 同一个函数对批量读来的事实作出（一次查询，不逐人调 `Authorize`；关注者没有上限）；issue 再对只看自己的人执行 `HiddenFrom`。一个等价测试在权限矩阵的种子上逐格比对批量判定与 `Authorize`。离开项目、降为只看自己的访客、被移出私有项目的人不收（Plane 不复核，D11）。
- **类型**（`sender`）：收件人是 I 的创建者为“我创建的”，否则是 I 当前的负责人为“分配给我”，否则为“我关注的”；提及另为“提及”。发送时定下、之后不变（Plane 的筛选在读时按当时的负责人重算，并且排除失效，D3；登记差异）。Plane 的“分配给我”要求创建者不是负责人（D14），不照搬。
- **快照**：通知存发送时的工作项名字、编号与状态（名、组），以及这个变化的字段、动词、旧值、新值（照 Plane 的 `data`）；评论通知另存摘录。指向另一个工作项的变化（父工作项、关联）只存 id，读时经 issue 的端口按读的人此刻的可见换成编号与名字，看不到的不显示名字（M4 交接 M7-02：不引用接收者看不到的另一端）。

### 3.4 通知的状态、列表、未读与全部已读

- 三个状态独立：`read_at`、`archived_at`、`snoozed_till`。暂缓按读那一刻的业务时钟算：`snoozed_till` 在将来为“暂缓中”，已过即回到收件箱（Plane 方向反了且永不到期，D1、D2）。不设定时任务。
- 列表 `GET /workspaces/{slug}/notifications`：页签 `tab=all|mentions`；`type` 为 `assigned,created,subscribed` 的子集（`all` 页签用）；`read=false` 只看未读；`archived=true` 只看已归档（默认不含）；`snoozed=true` 只看暂缓中（默认不含）。按 `(created_at, id)` 倒序游标分页。只列收件人此刻是有效成员的项目里的通知；只看自己的项目里只列他建的工作项的通知（读时过滤，通知行存工作项的创建者）。项目归档不影响（项目照样读得到）。
- 未读数 `GET /workspaces/{slug}/notifications/unread-count`：`{all, mentions}`（P1c R3 由 `total` 改名：两个页签各自的数，“全部”不含提及，照 Plane），不计已归档、暂缓中的，同样按可见过滤；有一个只含未读、未归档行的部分索引。
- 全部已读 `POST /workspaces/{slug}/notifications/mark-all-read`：请求体与列表的筛选同一组（`tab`、`type`、`archived`、`snoozed`），只作用于那组筛选选中的未读通知（Plane 的参数与列表对不上、“全部”页签会连提及一起标掉，D5）；分批更新，回答标了几条。
- 单条 `PATCH /notifications/{id}`：`read`、`archived`（布尔）、`snoozed_till`（将来的时刻或 `null`）。只有收件人自己，别人的 404。
- 保留期：通知不设保留期（Plane 相同）；软删的照 60 天清理；M8 观察增长。

### 3.5 收集箱：提交与分诊

- **一个项目一个收集箱，不建 `intakes` 表**：社区版每项目只用一个默认收集箱，名字、说明、图标都不显示；条目直接挂在项目上（§15 A）。
- **提交** `POST /projects/{id}/intake-issues`：项目的管理员、成员、访客；`intake_view` 关闭时 409（M6 开关的同一做法：写拒绝、读照常）。请求体照 Plane 前端实际发送的：名字（必填）、描述、优先级、标签、负责人、截止日期；状态由服务端设为项目的 Triage 状态，迭代、模块、父工作项、开始日期不接受。工作项经 issue 的新建核心写入（编号、动态、自动关注、提及、`issue.created` 事件照常），条目状态为“待处理”。只有应用内来源：Plane 的 `source`、`source_email`、`external_source`、`external_id`、`extra` 不建（M7-01，功能说明不改，`issue-root.tsx` 的 FORMS 分支删除）。
- **状态与迁移**：`pending`（待处理）、`snoozed`（暂缓）、`accepted`（接受）、`declined`（拒绝）、`duplicate`（重复）。只有待处理与暂缓可以分诊：可到接受、拒绝、重复、暂缓（再暂缓改日期），暂缓可取消回到待处理；接受、拒绝、重复是终态，不能再改（Plane 没有任何迁移规则，D17）。暂缓要给将来的 `snoozed_till`，到期按读那一刻算回到待处理（列表、待处理数都把它算作待处理；Plane 永不到期，D16）。重复要给 `duplicate_to_id`：同项目、他看得到、不是收集箱条目、不是自己（Plane 不校验，D19）；只存不建“重复”关系（Plane 相同）。
- **接受**：工作项从 Triage 移到项目的默认状态（项目总有一个：默认状态不能删除，`project.state_default`，只经“设为默认”改变），在项目锁下读它，写一条普通的状态动态并发布 `issue.updated`，关注者收到状态通知（Plane 不写动态，D18）。拒绝、暂缓、重复只改条目，工作项留在 Triage、项目里看不到。
- **动态**：每次分诊写一条 `field = intake` 的动态，`verb` 为新状态的名字（Plane 用数字 `-1`、`0`、`1`、`2`，前端的 `inbox.tsx` 按数字分支；Nerve 用名字，渲染件随之改）；`issue_activities.verb` 的 CHECK 随之放宽（plane-diff 已预告）。分诊不通知（Plane 相同）。
- **删除** `DELETE /intake-issues/{id}`：项目管理员，或仍是项目中成员以上的提交者（M4 的 `issue.delete` 同一规则：访客提交者不能删，Plane 能，登记差异）。未接受的连工作项一起删（工作项删除的完整连带）；已接受的只删条目，工作项留下（Plane 相同）。
- **待处理数**：一个工作区级的读给出每个看得到（有效成员、未归档）、开着收集箱的项目的待处理数（含已到期的暂缓；只看自己的访客只数自己的），供侧栏；不加进项目的回答（项目模块不读 issue 的表）。

### 3.6 收集箱条目作为工作项

条目就是一个处在 Triage 的工作项，所以：

- **读**照常：详情、按编号取、评论、动态、表情、链接、附件，按工作项的读权限（只看自己的访客只看自己建的）。描述版本照 M4 的 `issue_description_version.read`（管理员、成员；Plane 让只看自己的访客也读收集箱条目的版本，登记差异）。详情多带一个 `intake`（状态、暂缓到、重复目标），页面据此显示分诊界面而不显示状态字段（`state_id` 是隐藏的 Triage 状态，状态接口不列它）；通知预览也凭它判断（不需要 Plane 的 `is_inbox_issue`）。M3 交接 §1.2 的分诊状态的接口与前端（store、下拉框、类型）不加回：条目的状态就是收集箱的状态，接受之后才有普通状态。
- **评论、表情、链接**按它们自己的规则照常可写（分诊时的讨论在这里）。资产（描述图片、附件）的写对 Triage 宿主按 `intake.update` 判定（管理员与提交者，访客提交者也算），不按 `issue.update`，并且只在待处理、暂缓且收集箱开着时（拒绝、重复 409 `issue.intake_closed`，关着 409 `issue.intake_disabled`，管理员也一样）；资产的读照常（P2c）。
- **内容的修改** `PATCH /intake-issues/{id}`：只在待处理、暂缓时；它就是工作项修改的核心（提及同步、描述版本、内容绑定、动态、事件照常），换了字段白名单（名字、描述、优先级、标签、负责人、截止日期；访客提交者只有名字、描述）与授权。接受之后它答 409，改工作项走普通的工作项接口与权限（访客提交者从此不能改，同 Plane）；拒绝、重复之后同样 409。
- **工作项的改、归档、删除**对 Triage 工作项一律 409 `issue.in_triage`；它不能做父工作项、子工作项、关联的另一端或重复的目标，不能加入迭代、模块，草稿不能以它为父工作项。
- **不出现**在项目列表、成员列表、工作区列表、搜索、关联与父工作项的选择器、迭代/模块进度、列表的计数（子工作项数等）里；自动归档只碰已完成、已取消的状态组，天然不碰它。
- **图片**：收集箱的新建弹窗维持现状不带图片（新建工作项的图片经草稿发布迁移，收集箱不走草稿，M5-M7-01）；建成之后条目就是工作项宿主，按上面的规则带图片；M5-M7-01 要求的三浏览器、本机与 S3 的证据在 P5。登记差异。

### 3.7 视图与工作区跨项目列表

- **视图**：`issue_views` 一张表，`project_id` 为空即工作区视图。存名字、说明、图标、Nerve 格式的 `filters`、`display_filters`、`display_properties`、所有者。删去 Plane 的 `query`、`rich_filters`（plane-diff 已定）以及 `access`、`is_locked`、`sort_order`、`archived_at`、旧 `filters` 格式：社区版的序列化器把 `access`、`is_locked` 设为只读，所有视图都是公开、未锁（D27），前端也不按 `sort_order` 排序。筛选词按页面校验：项目视图用项目列表的词（含迭代、模块），工作区视图用工作区列表的词。
- **权限**：建：项目视图为项目管理员、成员（开关要开着），工作区视图为工作区管理员、成员（Plane 的新建不鉴权，D25）；改：只有所有者（Plane 相同，D29 照搬）；删：所有者或项目 / 工作区管理员；读：项目视图为项目的有效成员，只看自己的访客看不到别人的视图（也就是全部）；工作区视图为工作区管理员、成员，工作区访客的列、取、建、改、删都是 403（§3.10 的表、H14；访客只有四个默认视图，D3；P3b R1）。Plane 的路由 PUT 绕过一切检查（D24），不做 PUT；看不到的 404（D26）。删除视图时所有人对它的收藏随之移除（随 P3c）。实现见 [P3b spec](specs/P3b-views.md)：已存的筛选键项目视图 12 个、工作区视图 11 个（`sub_issue` 只在显示里）；`/views/{view_id}` 上的每个 404 都是 `navigation.view_not_found`；PATCH 给出的顶层字段整体替换，没有改动不写；已归档项目上的视图读写照常（P3b R2）。
- **视图列表是集合型**，不分页（大小由成员的操作决定，同标签、状态），按 `created_at` 倒序、同值按 `id`；页面在本地排序、筛选（Plane 相同）。所以 M2 交接 §1 的“每种排序一个游标载荷”不适用。
- **视图页的工作项**：前端把视图的筛选与显示作为查询发给项目列表（Plane 相同，不另设“视图的工作项”接口）。改了筛选或显示而未保存时，改动只在这个页面里（Plane 相同）；“更新视图”（所有者）存回视图，“另存为新视图”新建。
- **工作区跨项目列表** `GET /workspaces/{slug}/issues`：工作区的管理员、成员、访客；范围是他是有效成员的全部未归档项目，逐项目执行只看自己；只读快照、统一游标与比较器（M4 交接 M7-03）。四个默认视图是同一接口加 `assignee_id=me`、`created_by_id=me`、`subscriber_id=me`（Plane 相同）。不是成员的项目不在范围内（工作区管理员与公开项目也一样），`project_id` 只与范围相交；排除收集箱里的、已归档的、已删除的工作项。“我关注的”按关注行算，含自己建的、被指派的（Plane 的工作区视图就这样，D30 的另一个定义只在通知里）。工作区的词（12 个）：`project_id`、`state_id`、`state_group`、`priority`、`label_id`、`assignee_id`、`created_by_id`、`mention_id`、`subscriber_id`、`start_date`、`target_date`、`sub_issue`；没有迭代、模块（§15 D4），也不收 `archived`、`relation`（422）。只平铺：`group_by`、`sub_group_by`、`group_key`、`sub_group_key` 一律 422（Plane 的工作区接口只平铺、页面只渲染表格）；次序同项目列表。目录取看得到的项目的并集（工作区标签目录随 P6b）。实现见 [P3a spec](specs/P3a-workspace-list.md)。默认视图的显示设置照 Plane 存在浏览器（按账户、工作区、视图）。

### 3.8 收藏

- 只给工作区管理员、成员（Plane 相同，访客不能用）。类型恰为 `project`、`cycle`、`module`、`view`、`folder` 五种（M1-P3 交接）。
- **写入校验**：目标存在、在这个工作区、他此刻看得到、未归档（项目须是有效成员；迭代、模块、项目视图随项目；工作区视图须看得到）；同一目标只能收藏一次，再收藏原样返回。`parent` 只能是他自己的文件夹，文件夹不能放进文件夹；同一人在同一工作区的文件夹名不能重复（Plane 只在前端拦，D32）。`sequence` 按这个人算（Plane 按全工作区，D33）。
- **读** `GET /workspaces/{slug}/favorites`：他的全部收藏（集合型，含文件夹里的），每项带目标的名字、图标、所属项目；目标他现在看不到的、已删除的、已归档的不列（离开项目、只看自己的访客、写入与归档交错留下的行都由读时过滤兜住）。资源本身不带 `is_favorite`，页面由收藏列表算出（避免四个模块各自去读收藏表）。
- **连带**：目标删除或归档时，所有人对它的收藏移除——迭代删除、模块删除（Plane 只删当前用户的，D31）、迭代归档、模块归档、视图删除；项目删除或归档时移除这个项目里的全部收藏（项目自己、它的迭代、模块、视图）。恢复不加回（Plane 相同；读时过滤保证归档期间不显示，连带保证恢复后不回来）。删除文件夹连它里面的一起删（Plane 相同）。实现见 [P3c spec](specs/P3c-favorites.md)：`view` 兼指两种视图（工作区视图可收藏）；接口的字段 `entity_id`，建新 201、已有 200；开关不看；全部软删；三种归档事件由 planning、project 在归档的事务里发布（§6.1）；收藏行加锁的全序是文件夹先于条目。

### 3.9 最近访问

- 页面显式记录：`POST /me/recent-visits`（`entity_type` 为 `issue` 或 `project`，`entity_id`），看得到才记录，否则 404；所有 GET 都没有副作用（总体设计 §3.2）。只记工作项与项目：Plane 记六种、只读出三种（D37），页面已在 M1 删除。
- 同一人、同一工作区、同一目标只有一行，重访只更新访问时间；按访问时间倒序（Plane 按建立时间，D35）；每人每工作区最多 20 条：写入时以这个人、这个工作区的事务级咨询锁串行，在同一事务里物理删除第 20 条之后的行（Plane 的 `== 20` 在并发下失守，D36）。
- 读 `GET /workspaces/{slug}/recent-visits?entity_type=`：集合型，每行带目标的摘要（工作项：名字、状态、优先级、负责人、编号、项目标识；项目：名字、图标、标识）；他现在看不到的不列（Plane 不过滤，D39）。工作项删除时，指向它的最近访问在原事务里软删（M4 交接 M7-04；Plane 单个删除时硬删所有人的行、批量删除不清，D44）。实现见 [P3d spec](specs/P3d-recents.md)：`project_id` 非空；记录 204（新记与重访同），重访同一行、`visited_at` 与审计的 `updated_at`、`updated_by_id` 同一时刻；看不到答目标自己的 404、与不存在的逐值相同；工作区访客能记能列；已归档的能记、照列，收集箱里的条目能记、接受之前不列；上限数未删的全部行、先过滤后取 20；记录的锁序是工作区 S → 项目 S → 咨询锁，裁剪 `SKIP LOCKED`；连带的订阅者 `navigation_forget_deleted_issue_visits`，项目、工作区删除经 `navigation.delete_rows` 的第三步。

### 3.10 权限

新的动作在 `access/domain/rules.go`。表只放角色；“提交者”“所有者”这类相对规则由用例判定（`workspace_member.update` 的约定），`AllowCreator` 只用在它原有的含义上（成员以上的创建者）。

| 动作 | 级别 | 表里的角色 | 用例的相对规则 |
|---|---|---|---|
| 关注、取消关注 | 项目 | 用 `issue.read` | 只看自己的访客只对自己建的 |
| `notification.read`、`notification.write` | 工作区 | 管理员、成员、访客 | 只对自己的通知 |
| `intake.submit`、`intake.read` | 项目 | 管理员、成员、访客 | 只看自己的访客只看自己提交的 |
| `intake.update` | 项目 | 管理员、成员、访客 | 非管理员须是提交者；访客提交者只改名字、描述；收集箱里的工作项上资产的写也按它判定（P2c） |
| `intake.triage` | 项目 | 管理员（§15 B） | — |
| `intake.delete` | 项目 | 管理员，`AllowCreator` | — |
| `intake.count` | 工作区 | 管理员、成员、访客 | 各项目的待处理数（P2b）；只看自己的访客只数自己的 |
| `view.read` / `view.create` / `view.update` / `view.delete` | 项目 | 读三种角色；建、改、删管理员与成员 | 改只限所有者；成员删只限所有者 |
| `workspace_view.read` / `.create` / `.update` / `.delete` | 工作区 | 管理员、成员 | 同上，按工作区角色 |
| `workspace_issue.list` | 工作区 | 管理员、成员、访客 | 逐项目可见与只看自己 |
| `favorite.read`、`favorite.write` | 工作区 | 管理员、成员 | 只对自己的收藏 |
| `recent_visit.read`、`recent_visit.write` | 工作区 | 管理员、成员、访客 | 只对自己的；目标须看得到 |

看不到的一律 404，看得到但没权限 403。工作区管理员兼项目访客照 M4 的规则（只看自己）。权限矩阵每个新操作一行。

### 3.11 项目开关与入口规则

`intake_view` 关闭：提交、分诊、条目的修改与删除、条目上资产的写 409 `issue.intake_disabled`，读照常，重新打开数据都在。`issue_views_view` 关闭：项目视图的建、改、删 409 `navigation.views_disabled`，读照常。侧栏、页头标签栏与命令面板用同一个谓词决定是否给出“视图”“收集箱”入口：开关开着、是项目的有效成员（三种角色，照 Plane）；新建视图另要求管理员或成员。仿 `planningEntryOn` 写 `viewsEntryOn`、`intakeEntryOn`，侧栏与命令面板共用，单测钉住访客与成员（M6-M7-07）。

## 4. 数据、模块归属与删除图

### 4.1 模块归属

- **`notification`**（新模块）：`notifications` 表，事件订阅者与分发任务，通知的接口。读工作项、关注者、提及经 issue 的端口，收件人的判定经 access 的端口。
- **`navigation`**（新模块）：`issue_views`、`user_favorites`、`user_recent_visits` 三张表与它们的接口。视图筛选词的校验经 issue 的 `ListSettings` 端口（M6 先例）；目标的存在、可见与摘要经 project、planning、issue 的批量端口。
- **收集箱放在 `issue` 模块**（§15 A）：`intake_issues` 表、Triage 工作项的新建路径、分诊、收集箱列表与待处理数。条目与工作项一对一，每一次分诊都写工作项、动态、关注与事件，并在项目锁下、经写闸门进行；放进 issue 免去一条跨模块的写路径，与 M6 把归属表放进 issue 同理。总体设计 §6.2 目录结构里的 `intake/` 不建。收集箱列表是一个专门的平铺查询，复用列表的筛选解析与游标封套，不进列表引擎的分组机器（列表引擎的状态目录按设计排除 Triage）。
- 工作区跨项目列表是 issue 列表引擎的新范围 `WorkspaceList`。

### 4.2 表（迁移从 `00049` 起，`<v>_<模块>_<表>.sql`）

五张新表都带 `deleted_at` 与审计列（`created_by_id`、`updated_by_id` 指向账户 `ON DELETE SET NULL`）。账户从不物理删除（M2 设计 4.7），下面写的是指向账户的其余外键的动作。

| 表 | 模块 | 要点 |
|---|---|---|
| `intake_issues` | issue | `workspace_id`、`project_id`、`issue_id`（`CASCADE`）、`status` CHECK 五值、`snoozed_till`、`duplicate_to_id`（不是外键，§15 D2）；部分唯一 `(issue_id) WHERE deleted_at IS NULL`（Plane 没有任何唯一，D22）；CHECK：暂缓须有日期、重复须有目标、其余状态两者皆空 |
| `issue_activities`（改） | issue | `verb` 的 CHECK 加入五个收集箱状态名，只在 `field = 'intake'` 时 |
| `notifications` | notification | `workspace_id`、`project_id`（`CASCADE`）、`receiver_id`（`CASCADE`，Plane 相同）、`triggered_by_id`（`SET NULL`，空 = nerve 自己）、`issue_id` 与 `issue_created_by_id`（不是外键，M4 设计 4.16）、`sender`、`data`（快照 JSON，形状由 CHECK 钉住）、`read_at`、`archived_at`、`snoozed_till`、`event_id`、`change_seq`；部分唯一 `(event_id, change_seq, receiver_id)`（任务重试幂等）；索引 `(receiver_id, workspace_id, created_at, id)`、`(workspace_id, issue_id)`、未读未归档的部分索引。删去 Plane 的 `title`、`message`、`message_html`、`message_stripped`、`entity_name`（前端不读，只有工作项一种） |
| `issue_views` | navigation | 见 §3.7；`workspace_id`、`project_id`（`CASCADE`，可空）、`owned_by_id`（`CASCADE`，Plane 相同）、`created_by_id`、`updated_by_id`（`SET NULL`）；不指向工作项 |
| `user_favorites` | navigation | `workspace_id`、`project_id`（`CASCADE`，可空）、`user_id`（`CASCADE`）、`entity_type` CHECK 五值、`entity_identifier`（不是外键）、`name`（文件夹）、`parent_id`（自引用 `CASCADE`）、`sequence`；部分唯一 `(user_id, entity_type, entity_identifier)` 与 `(user_id, workspace_id, lower(name)) WHERE entity_type = 'folder'`，两条都带 `deleted_at IS NULL`（软删之后能重建）；`sequence double precision`；`created_by_id`、`updated_by_id` `SET NULL`（P3c） |
| `user_recent_visits` | navigation | `workspace_id`、`project_id`（`CASCADE`，非空：两种目标都在项目里）、`user_id`（`CASCADE`）、`entity_type` CHECK 两值、`entity_identifier`（不是外键）、`visited_at`；CHECK 项目的访问 `project_id = entity_identifier`；三条部分索引都带 `deleted_at IS NULL`：唯一 `(user_id, workspace_id, entity_type, entity_identifier)`（软删之后能重建）、连带 `(workspace_id, entity_identifier)`、列表与裁剪 `(user_id, workspace_id, visited_at DESC, id DESC)`；`created_by_id`、`updated_by_id` `SET NULL`（P3d） |

每条指向 `workspaces`、`projects`、`issues` 的外键有不带条件的索引。plane-diff 逐表登记。

### 4.3 删除图与清理

- **工作项删除**（每个 `issue.deleted`）：它的 `intake_issues` 随工作项在原事务里软删（issue 自己的表）；notification 的订阅者在原事务里软删它的通知，navigation 的订阅者在原事务里软删指向它的最近访问（同 asset 的先例，按 `(workspace_id, issue_id)`、`(workspace_id, entity_identifier)` 索引）。别的条目里指向它的 `duplicate_to_id` 不动：读时只给看得到的目标，看不到或已删的不显示。
- **项目、工作区删除**：notification 与 navigation 各加一个订阅者，在原事务里投递自己的分批任务（`notification.delete_rows`、`navigation.delete_rows`，M6 `planning.delete_rows` 的先例：保留原请求的时刻与删除者，提交后立即不可见）；navigation 另有工作区范围的一步，处理没有项目的行（工作区视图、文件夹，issue 的 `workspace_drafts` 先例）；`intake_issues` 随 issue 的分批任务。
- **删除目录守卫**：五张表都在 `keysTo` 的范围内按原有机制核对，没有新的豁免（`duplicate_to_id` 不是外键）。
- **60 天清理**：每张新的带 `deleted_at` 的表恰有一个清理者，`TestPurgeCatalogCoversTablesAndIncomingKeys` 的表数由 32 改为 37（P3b 之后 35，P3c 之后 36，P3d 之后 37，已到）；清理顺序在 asset → issue → planning → project → workspace → identity 中插入 notification、navigation（在 issue 之前，它们不被别的表指向）。`TestPurgeScale` 与删除规模测试在 M7 代码上重跑。

## 5. 接口

一个模块一个文件：`api/modules/notification.yaml`、`api/modules/navigation.yaml`，收集箱、关注与工作区列表进 `api/modules/issue.yaml`。

| 接口 | 说明 |
|---|---|
| `POST /issues/{id}/subscribe`、`POST /issues/{id}/unsubscribe` | 204；`IssueDetail.is_subscribed` |
| `GET /workspaces/{slug}/notifications` | 游标分页，§3.4 的筛选 |
| `GET /workspaces/{slug}/notifications/unread-count` | `{all, mentions}` |
| `POST /workspaces/{slug}/notifications/mark-all-read` | 请求体为列表的筛选，回答标了几条 |
| `PATCH /notifications/{id}` | `read`、`archived`、`snoozed_till`，回答通知 |
| `POST /projects/{id}/intake-issues`、`GET /projects/{id}/intake-issues` | 提交；列表（`status`、优先级、标签、负责人、创建者、建立与更新日期，排序为建立、更新时间与编号，游标） |
| `PATCH /intake-issues/{id}`、`DELETE /intake-issues/{id}`、`POST /intake-issues/{id}/triage` | 改内容、删除、分诊（`{status, snoozed_till?, duplicate_to_id?}`）；`{id}` 是工作项 id（Plane 相同） |
| `GET /workspaces/{slug}/intake-counts` | 集合型 `{project_id, pending}` |
| `GET /workspaces/{slug}/issues` | 工作区跨项目列表（`IssueListPage`） |
| `GET/POST /projects/{id}/views`、`GET/POST /workspaces/{slug}/views`、`GET/PATCH/DELETE /views/{id}` | 视图；列表集合型 |
| `GET/POST /workspaces/{slug}/favorites`、`PATCH/DELETE /favorites/{favorite_id}` | 收藏（P3c）：建新 201、已有 200；改只接受 `parent_id`、`sequence`、文件夹的 `name` |
| `POST /me/recent-visits`、`GET /workspaces/{slug}/recent-visits` | 最近访问（P3d）：记录 204（新记与重访同）；列表集合型，`entity_type` 可选 |

错误按 RFC 9457、模块前缀的问题码，每个操作写明 `x-problem-codes`，每个新码进 `error-messages.ts` 与两份 `errors.json`。新码至少有：issue 的 `issue.in_triage`、`issue.intake_disabled`、`issue.intake_not_found`、`issue.intake_transition_not_allowed`（409）、`duplicate_to_id` 不合格（422 `not_allowed`，字段错误）；notification 的 `notification.not_found`；navigation 的 `navigation.view_not_found`、`navigation.views_disabled`、`navigation.favorite_not_found`、`navigation.favorite_folder_name_taken`（409）、目标不合格（422 `not_allowed`）。P 阶段的 spec 定全集。

## 6. 跨模块协作、事件与任务

### 6.1 事件的扩展（issue 的公开约定，§15 D1）

- 评论的三个事件、链接的三个事件加一个 `issue` 变化：建立、修改在 `NewID`，删除在 `OldID`（与表情、关联的移除、`asset.attachment_deleted` 同一写法，[P1a spec](specs/P1a-subscriptions-events.md) R1）。
- `issue.created`、`issue.updated`、评论的建立与修改，每个这次新被提及的人加一个 `mention` 变化（`NewID` 为账户），集合按 §3.1 由 issue 算出，未经可见性过滤（通知发送时自己复核）。
- 收集箱的接受发布普通的 `issue.updated`（状态变化）。
- planning 新增 `planning.cycle_archived`、`planning.module_archived`，project 新增 `project.archived`，供收藏的连带；恢复不发布。
- 关联的两个事件追加 `requested_on`：建立时是请求所在的工作项，移除时是存储为 `issue_id` 的那一端（移除按关联 id），值为那一端读到的类型（[P1b spec](specs/P1b-notification-core.md) §2.3、R11），通知只发到这一端。
- 只加变化与类型，不改已有变化；事件形状测试同改；asset、planning 现有的订阅者不看新变化。

### 6.2 通知的订阅者与任务

- 订阅者 `notify_on_issue_events`（notification 模块，在写入事务内）：只看 §3.2 表里的事件；没有可通知变化的（只拖了 `sort_order`、只改了迭代模块、只改描述且无新提及）不投递；有的投递一个 `notification.fanout` 任务，参数只带事件 id、类型、时刻、工作区、项目、工作项、执行者与去掉 HTML 的变化（描述、评论正文最大 50 万字，不进 `river_job`）。
- 任务（`default` 队列，显式 `Timeout` 与有限次数，次数用尽时记日志）：在一个读写事务里先经 issue 的端口以 `FOR SHARE` 锁住工作项行（已删除则什么都不写、完成），再读关注者、工作项快照、评论摘录，经 access 一次判定收件人，一条 `INSERT … ON CONFLICT DO NOTHING` 写入全部通知。`FOR SHARE` 与删除的行锁冲突，所以分发与删除串行：删除先提交，任务看到已删除；任务先提交，删除事务里的订阅者软删掉刚写的通知。按事件一个任务：批量路径（子树删除不通知；子工作项的移动、自动归档逐个事件）各自独立。
- 订阅者 `forget_deleted_issue_notifications`：`issue.deleted` 时在原事务里软删这个工作项的通知。

### 6.3 端口

- issue 给 notification：关注者与提及的读、工作项快照（名字、编号、状态名与组、创建者）与行锁、评论摘录、只看自己的 `HiddenFrom`；读通知时把父工作项、关联另一端的 id 按读的人的可见换成编号与名字（一页一次批量）。
- access 给 notification：一组账户在一个项目里的角色与只看自己（§3.3，与 `Authorize` 同一个判定函数）；给 notification、navigation：调用者此刻看得到的项目集合与每个项目的只看自己。
- issue 给 navigation：视图筛选词的校验（项目页、工作区页两种词表）、最近访问用的工作项摘要与可见判定。
- project、planning 给 navigation：项目、迭代、模块的存在、可见、归档与摘要（批量）。
- 所有端口由 bootstrap 适配，不在模块之间 import；跨模块的 SQL 联表一律不做（`TestSQLCSchemaScope` 不加例外）。

## 7. 前端

- **数据层**：五个领域的 store 与 service 重写为 v0 7.7 的形状：按会话分代、`useSessionSWR`（键带 `loginId`）、写入排队、`Reconciled`、地址只从路由参数、类型只来自 `@nerve/api-client`；删除 6 个旧 service、`intake-description-version.ts`、`TLegacyIssueUpdateRequest`、手写的 `IFavorite`、`TNotification`（连同 M3 交接 §2.4 的 17 行死成员）、`IProjectView` 等类型与两个模块级单例（M2 交接 §2）。旧地址各加一条 `old-…-urls` 关键词规则（M6 先例）。
  - **处理结果（M7/P4，通知的部分）：** 通知的 service 与 store 照此重写（`core/services/notification.service.ts`、`core/store/notification/`），通知的旧 service（模块级单例）、store、hook、`TNotification` 等手写类型连同 17 行死成员删除，加 `old-notification-urls`（[P4 spec](specs/P4-web-notifications.md) §2.1–§2.4、§2.9、§2.10）。通知的部分关闭；收集箱、视图、收藏、最近访问的部分随 P5–P7。
- **先收起会 404 的入口**：第一个页面阶段（P4）去掉项目页、迭代页、模块页筛选行的“保存为视图”（`enableSaveView` 的两处），P6 随视图接口接回；其余旧入口随各自的阶段接通。
  - **处理结果（M7/P4）：** 两处 `enableSaveView` 已去掉（`project-layout-root.tsx`、`planning-page-layout-root.tsx`，单测 `roots/save-view-collapsed.test.tsx`）；HOC 的保存与更新分支到 P6a/P6b 之前没有传入者，登记在 [P4 阶段记录](reviews/P4-web-notifications-review.md) §5（[P4 spec](specs/P4-web-notifications.md) §2.8、R17）。“保存为视图”的收起关闭；接回随 P6a/P6b。
- **通知（P4）**：通知页（页签、筛选、卡片、加载更多、预览）；卡片按新通知的字段与类型重写，动作文字全部经 `t()`；顶栏与侧栏的未读数；工作项详情、浮层与命令面板的“关注 / 取消关注”。收集箱条目的通知在 P4 以普通浮层打开，P5 再加嵌入的分诊界面与它的故事。
- **收集箱（P5）**：页面、列表与筛选、详情（复用工作项详情的正文、评论、动态；`intake` 在时隐去状态字段）、分诊页头（先拆 470 行的 `inbox-issue-header.tsx` 与 511 行的 store）、新建弹窗（按 `requestClose` 改关闭策略，M6-M7-01）、拒绝 / 删除 / 查重 / 暂缓弹窗；动态的 `inbox.tsx` 改按状态名；侧栏的待处理数；通知预览里嵌入的分诊界面。命令面板没有收集箱的搜索结果（Plane 也没有），保留“转到收集箱”。
- **视图（P6a、P6b）**：项目视图列表与详情（列表引擎加 `PROJECT_VIEW` 一行，照 M6 的表驱动）、“另存为新视图”“更新视图”（接上 `enableUpdateView`）；工作区视图目录、默认四个与已存视图的列表（新的 `WORKSPACE` 范围与 `IssueListCommandHost` 的对话框宿主）；命令面板的视图菜单、搜索的视图组与 `create_view`。
- **收藏与最近访问（P7）**：侧栏收藏（文件夹、拖动、移出，失败经提示而不是 `console.error`）、项目 / 迭代 / 模块 / 视图的星与命令面板的收藏命令（M6-M7-02、M3 交接 §1.4 的项目收藏）；首页的最近访问（工作项、项目）与页面的显式记录（打开工作项详情、进入项目时各记一次，失败不打扰用户）。
- **交接的页面项**：M3 交接 §3、§4 的可访问性与地址规则（弹窗名、字段提示、嵌套按钮、行名控件、下拉框、Escape、复制经 `useCopyLink` 并删 4 个 `copy-hook` 例外）、`getTabIndex` 与两份 `tab-indices.ts` 的全套删除（M6-M7-05）、`projectMemberIds` getter 的删除（M3 §7）、`select-duplicate.tsx` 不再从 `useParams()` 取它所在路由没有的参数。M3 交接 §2 的其余三项（命令面板在工作区页外的工作区菜单、`createWorkspace` 被拒后的重试、`useWorkspaceFetch` 换 slug）不在 M7 的页面上，P8 逐项核对现状并写结论。

## 8. 运行与观察

- 新任务：`notification.fanout`（请求触发，`default` 队列）、`notification.delete_rows`、`navigation.delete_rows`；登记 `EnqueuedKinds` 并各有 worker、各自显式 `Timeout`（River 默认 1 分钟不作依据）。不加周期任务（暂缓按读时算）。
- 分发任务与链接抓取、分批删除共用 `default` 的 10 个 worker；不加新队列（加队列是平台改动）。每次写多一个 `InsertTx`，在项目锁与写闸门之内：P1b 测量 M4 的写入基准与自动归档一轮在前后的差异，写进阶段记录。
- 保留期：通知不设（§3.4），最近访问有 20 条上限，收藏与视图随人的操作；M8 观察通知表的增长与分发耗时。

## 9. 验证策略

单元覆盖收件人、类型、暂缓、迁移规则、文件夹规则等纯规则；adapter 覆盖 SQL、约束、幂等与锁；`bootstrap` 组合覆盖认证、权限矩阵、事件到通知的全链、删除图与清理、全程序接口；Playwright 覆盖真实页面与数据库断言（§2 每条）。负例至少：看不到的工作项与项目、通知里另一端的工作项、访客与只看自己、工作区管理员兼访客、私有项目、撤权后的通知与收藏、开关关闭、并发的两次提交与两次分诊、分诊与工作项删除交错、分发任务与工作项删除交错、关注与工作项删除交错、收藏与归档交错、最近访问的并发记录与上限、跨工作区的收藏目标、重复目标被清理之后的读、换会话与迟到回答。批量复核与 `Authorize` 的等价测试（§3.3）。工作区跨项目列表加规模测试（多项目、总量与项目列表的规模目标同级），通知的分发加大关注者数的规模测试。每个安全、可见、幂等性质至少有一个能让它失败的变异。M5-M7-01 的收集箱条目图片在 P5 有三浏览器、本机与 S3 的证据。

测试时间：CI 的 Go 两路（每路作业 25 分钟；第 0 路另跑生成检查、lint 与 `bootstrap` 以外的全部包；每次 `go test` 20 分钟）与 e2e 两片（每片 35 分钟、30 分钟报告线按片，负责人 2026-11-17）都在刀口附近。每个阶段在阶段记录里写本段新增测试的本机时间与对两路、两片的预估；任一路或任一片的预估接近期限的八成时，先向负责人提出加一路或加一片，不自行改作业结构。

## 10. 风险

| 风险 | 对策 |
|---|---|
| 每次工作项的写多一个通知的 `InsertTx`，在项目锁与写闸门之内 | 订阅者只投递、不读不算；没有可通知变化的不投；P1b 测写入基准与自动归档 |
| 工作区跨项目列表要在全部可见项目里先筛选、计数再分页 | P3a 的规模测试与执行计划核对；不达标先在 P3a 内修，不带着进页面阶段 |
| 扩展 issue 的事件改动 M4 的公开约定与它的测试 | 只加变化、不改已有变化（§6.1） |
| 关注者没有上限，一次分发的收件人可能很多 | 批量判定与一条 INSERT；规模测试钉住 |
| 分发、关注、收藏的写与删除、归档交错 | `FOR SHARE` 串行（§3.1、§6.2）与读时过滤（§3.8）；交错测试 |
| 前端五个领域约 144 个文件几乎全部重写，五个文件过 300 行 | 每个页面阶段先按职责拆大文件；每段 ≤ 16 Task，超限在实施前拆 |
| Go 测试与 e2e 的时间随 M7 增长 | §9 的逐段预估与报告；加路、加片须负责人同意 |

## 11. 输入交接与后续交接

| 输入 | 条款落点 | 关闭阶段 |
|---|---|---|
| [M1-P2-trim-content](handoffs/M1-P2-trim-content.md) | 收藏、最近访问没有页（§3.8、§3.9）；首页最近访问只列工作项、项目，读新接口；通知没有已删功能的字段 | P3c、P3d、P4、P7 |
| [M1-P3-trim-platform](handoffs/M1-P3-trim-platform.md) | 收藏的类型恰五种、前端用生成类型；通知没有工时 | P3c、P7 |
| [M1-closeout](handoffs/M1-closeout.md) | 本领域死成员与死 prop（首个页面阶段跑基线）、改到的文件 oxlint 为零并另清一类规则（`no-shadow` 已清零，另选） | P4–P8 |
| [M2-closeout](handoffs/M2-closeout.md) | §1 视图列表是集合型（§3.7）；§2 会话分代与两个单例（§7）；§3 下拉框的浏览器核对；§4 复制；§5 删除图与指向账户的外键（§4.2、§4.3）；§6 不嵌 iframe、不改 CSP | P1–P3（后端）、P4–P8 |
| [M3-closeout](handoffs/M3-closeout.md) | §1 取数按权限与开关、S2 清单、§1.2 分诊状态不加回（§3.6）、§1.4 项目收藏（§7）；§2 视图 store、通知预览的取数、布局的 key、17 行死成员（§7）、收藏的 `console.error`、其余三项 P8 结论；§3 弹窗与键盘逐项；§4 地址规则、复制、文案、`projectMemberIds`、守卫阶段；§5–§7 回执 | P4–P8 |
| [M4-closeout](handoffs/M4-closeout.md) | M7-01 收集箱来源与系统执行者（§3.5）；M7-02 关注接口、发送时复核、事件消费、另一端（§3.1–§3.3），动态的 `comment` 列不进公开接口（页面按字段与值渲染）；M7-03 视图在新引擎、工作区列表（§3.7）；M7-04 旧 DTO、删工作项时清最近访问（§3.9）、命令面板的视图结果（收集箱没有，§7）；M7-05 新页面的规则 | P1–P3、P4–P8 |
| [M5-closeout](handoffs/M5-closeout.md) | M5-M7-01 收集箱的图片（§3.6、§9）；M5-M7-02 附件事件与只带 id；M5-M7-03 通知卡片的正文（摘录为纯文本，不显示图片）；M5-04 原文转交 | P1、P2、P4、P5、P8 |
| [M6-closeout](handoffs/M6-closeout.md) | M6-M7-01 新建弹窗的关闭；M6-M7-02 迭代、模块的收藏与归档连带；M6-M7-03 项目视图有迭代、模块词，工作区级的不做（§15 D4，关闭）；M6-M7-04 视图的 store 与读；M6-M7-05 `getTabIndex`；M6-M7-06 转移不通知（§3.2）；M6-M7-07 入口规则（§3.11）；M6-M7-08、M6-M7-09 原文转交 | P1–P8 |

输入交接各写“处理结果（M7/P<k>）”，保持 open 至收尾关闭。后续交接由 P8 写给 M8：对等验收、通知表的增长与分发耗时的观察、M5-04/M6-M7-08 与 M6-M7-09 原文。

## 12. 分阶段推进

每段先 spec、plan、预检，再逐 Task 实现，结束 deep-review 与修复复审；每段约 16 Task 以内，超限在实施前拆分。后续阶段只在前置事实落实后写可执行 plan。

| Phase | 交付边界 | 必须同时完成的验证 |
|---|---|---|
| P1a | 工作项一侧的关注与事件：issue 事件的扩展（§6.1 的前两条：评论、链接事件的工作项 id，新提及）、关注接口与 `is_subscribed`；关键词 phase 改为 `M7/P1`（P1a、P1b 共用，关键词工具只认 `M<n>/P<k>`，M6/P3 先例） | H1 的接口与数据库；事件形状测试 |
| P1b | `notification` 模块的核心：表、订阅者与分发任务、通知要用的 issue 与 access 端口（快照与行锁、批量判定，随消费方一起到）、关联事件追加请求端、删除图与清理（[P1b spec](specs/P1b-notification-core.md) R1 拆段） | H2–H4 的数据库证据；幂等、复核、交错、规模；批量判定的等价测试；写入基准与自动归档前后 |
| P1c | 通知的接口：列表 / 修改 / 全部已读 / 未读数、读时的可见过滤（含已归档项目）与另一端、权限矩阵 | H2–H7 的接口证据与 e2e 故事 |
| P2a | 收集箱条目的生命周期（issue 模块）：`intake_issues` 与删除图、清理；Triage 的新建路径与开关；分诊（迁移规则、接受、动态）；条目作为工作项的写入边界（§3.6，含搜索等排除）；详情的 `intake`；两个操作的权限矩阵（P2a 实施前按删除守卫重新划分，见 [P2a spec](specs/P2a-intake-submit-triage.md) §1） | H8、H10 的接口与数据库；H11 的读写边界；交错 |
| P2b | 列表与其余：收集箱列表、内容的修改与删除、待处理数、它们的权限矩阵；P2a 审查留下的 m2、m3、n1（实施前再划分，见 [P2b spec](specs/P2b-intake-list-edit-delete.md) §1） | H9、H11 其余的接口与数据库（描述图片除外） |
| P2c | Triage 宿主的资产判定：资产的四个操作按 `intake.update` 与同一时段、资产路径读开关、先读工作项再选动作；`asset.yaml` 与资产处理器测试 | H11 里“描述的图片同此”的接口与数据库 |
| P3a | 工作区跨项目列表：issue 列表引擎的新范围 `WorkspaceList`、`GET /workspaces/{slug}/issues` 与规模测试（实施前再划分，见 [P3a spec](specs/P3a-workspace-list.md) §1） | H13 的列表一半的接口与数据库 |
| P3b | `navigation` 模块与视图：`issue_views`、两种视图的接口与权限、删除图与清理 | H12、H14 与 H13 的已存工作区视图的接口与数据库 |
| P3c | 收藏：拆 `bootstrap/app.go`、planning / project 的三种归档事件、`user_favorites` 与接口、连带、可见过滤（实施前再划分，见 [P3c spec](specs/P3c-favorites.md) §1） | H15、H16 的接口与数据库 |
| P3d | 最近访问：`user_recent_visits` 与接口、工作项删除时的连带、可见过滤（见 [P3d spec](specs/P3d-recents.md) §1） | H17 的接口与数据库 |
| P4 | 网页的通知与关注；收起“保存为视图”（见 [P4 spec](specs/P4-web-notifications.md) §1） | H1–H7 的页面、H20 的通知页 |
| P5 | 网页的收集箱，通知预览里的分诊 | H8–H11 的页面、H20 的收集箱页、M5-M7-01 的证据 |
| P6a | 网页的项目视图：列表、详情、保存与更新视图、命令面板的视图 | H12 的页面、H20 的视图页 |
| P6b | 网页的工作区视图与跨项目列表；工作区标签目录（M4 交接 M7-03：个人主页按标签分组与工作区列表的标签筛选，P3a R5） | H13、H14 的页面、H20 的工作区视图 |
| P7 | 网页的收藏与最近访问 | H15–H17 的页面、H20 的收藏与首页 |
| P8 | 最终矩阵、清理规模重跑、死成员与告警收敛、`getTabIndex` 与 `projectMemberIds` 的删除核对、M3 交接 §2 其余三项的结论、文档同步（总体设计 §3.2 的 `mark-all-read` 路径、§5.2 的 `intakes` 与表数、§6.2 的目录、§6.6、§9.4，plane-diff，前端改动清单）、M8 交接 | H1–H21 最终证据、deep-review |
| closeout | 首次 noff / main 绿 → §13 勾选与总体 §9.4 → 第二次 noff / main 绿 → 归档清理 | §13 每条证据、全部输入关闭 |

P1a 的第一个实现提交把关键词 phase 改为 `M7/P1`（P1a、P1b 共用；之后每段改为 `M7/P<k>`，拆开的段共用一个号）；纯设计分支不改 phase。

## 13. 完成标准

- [ ] H1–H21 每条有与失败路径相区分的接口、数据库与页面证据。
- [ ] 通知的触发、收件人、复核、类型、快照与另一端按 §3.2、§3.3；状态、列表、未读与全部已读按 §3.4。
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
| 设计 | 已合入 main（noff `8d17eed7`，与 P1a 的 spec、plan 一起）；预检 Ready after fixes，修正已改入；负责人 2026-11-17 决定 §15 A–D 按推荐 | [re-81](../../reviews/re-81-M7设计预检.md) |
| P1a 关注接口与通知要用的事件 | 已完成：合入 main（noff `9a3a388a`），main CI 44034770817 全绿；整分支审查 Ready to merge（五处次要由修复轮处理，限定复审 Ready to merge） | [P1a spec](specs/P1a-subscriptions-events.md) / [P1a plan](plans/P1a-subscriptions-events.md) / [P1a 阶段记录](reviews/P1a-subscriptions-events-review.md) |
| P1b 通知模块的核心 | 已完成：合入 main（noff `1a23f6f8`），main CI 44179902213 全绿；整分支审查 Ready after fixes，修复轮后限定复审 Ready to merge；spec 与 plan 经预检 [re-83](../../reviews/re-83-M7P1b预检.md) 修正后通过，基线 main `9a3a388a`（P1a 的 noff）；原 P1b 拆为 P1b、P1c；负责人 2026-11-18 决定自动归档的通知批量投递（spec R12） | [P1b spec](specs/P1b-notification-core.md) / [P1b plan](plans/P1b-notification-core.md) / [P1b 阶段记录](reviews/P1b-notification-core-review.md) |
| P1c 通知的接口与读 | 已完成：合入 main（noff `31f38580`），main CI 44337846563 全绿；整分支审查与限定复审都是 Ready to merge；本机最终门禁的第 0 路因 M6-M8-05 失败，负责人 2026-11-18 决定服务端以分支 CI 为准；spec 与 plan 经预检 [re-84](../../reviews/re-84-M7P1c预检.md) 修正后通过，基线 main `1a23f6f8`（P1b 的 noff）；两个检查点七项全过，检查点 1 的分支 CI 全绿；阶段记录 [P1c review](reviews/P1c-notification-api-review.md) | [P1c spec](specs/P1c-notification-api.md) / [P1c plan](plans/P1c-notification-api.md) |
| P2a 收集箱条目的生命周期 | 已完成：合入 main（noff `670399c4`；合入后 main CI 暴露的 M5 测试竞态另以 `057176e5` 修复），审查与集成见阶段记录 §9；spec 与 plan 经预检 [re-85](../../reviews/re-85-M7P2a预检.md) 修正后通过，基线 main `31f38580`（P1c 的 noff）；实施前按删除守卫与 P2b 重新划分（spec §1、R1）；阶段记录 [P2a 记录](reviews/P2a-intake-submit-triage-review.md) | [P2a spec](specs/P2a-intake-submit-triage.md) / [P2a plan](plans/P2a-intake-submit-triage.md) |
| P2b 收集箱的列表、内容修改、删除与待处理数 | 已完成：合入 main（noff `f25cccad`），审查与集成见阶段记录 §9；spec 与 plan 经预检 [re-86](../../reviews/re-86-M7P2b预检.md) 修正后通过，基线 main `057176e5`；实施前把 Triage 宿主的资产判定再划到 P2c（spec §1、R1） | [P2b spec](specs/P2b-intake-list-edit-delete.md) / [P2b plan](plans/P2b-intake-list-edit-delete.md) / [P2b 记录](reviews/P2b-intake-list-edit-delete-review.md) |
| P2c Triage 宿主的资产判定 | 已完成：合入 main（noff `7dfaa8a1`），审查与集成见阶段记录 §9；spec 与 plan 经预检 [re-87](../../reviews/re-87-M7P2c预检.md) 修正后通过，基线 main `f25cccad` | [P2c spec](specs/P2c-intake-assets.md) / [P2c plan](plans/P2c-intake-assets.md) / [P2c 记录](reviews/P2c-intake-assets-review.md) |
| P3a 工作区跨项目列表 | 已完成：合入 main（noff `17e0b1ac`），main CI 44743168952 全绿；审查与集成见阶段记录 §9；spec 与 plan 经预检 [re-88](../../reviews/re-88-M7P3a预检.md) 修正后通过；实施前把 P3 再划为 P3a（工作区列表）、P3b（视图）、P3c（收藏与最近访问）（spec §1、R1）；T7 后暂停，先做下面两个插入段，合并 main（`32ea59b8`）后继续 | [P3a spec](specs/P3a-workspace-list.md) / [P3a plan](plans/P3a-workspace-list.md) / [P3a 记录](reviews/P3a-workspace-list-review.md) |
| 插入段：bootstrap 测试按名字的哈希分路 | 已完成：负责人 2026-11-20 批准；合入 main（noff `33a19ee1`），main CI 44720865989 全绿；p0=46 的依据与 M4 周期整理测试的修复见记录 | [分路记录](reviews/shard-hash-review.md) |
| 插入段：M4 泳道查询的计划翻转 | 已完成：负责人 2026-11-20 裁定在 M7 里单独修；合入 main（noff `2083c0b7`），main CI 44724861021 全绿；预检 [re-89](../../reviews/re-89-M4泳道计划预检.md) | [spec 与 plan](specs/M4fix-swimlanes-plan.md) / [记录](reviews/M4fix-swimlanes-plan-review.md) |
| P3b navigation 模块与视图 | 已完成：合入 main（noff `1168fd25`），main CI 44820762025 全绿；审查与集成见阶段记录 §9；spec 与 plan 经预检 [re-90](../../reviews/re-90-M7P3b预检.md) 修正后通过，基线 main `17e0b1ac`；16 个 Task | [P3b spec](specs/P3b-views.md) / [P3b plan](plans/P3b-views.md) / [P3b 记录](reviews/P3b-views-review.md) |
| P3c 收藏与三种归档事件 | 已完成：合入 main（noff `9d26b1c3`），main CI 44947828240 全绿；审查与集成见阶段记录 §9；spec 与 plan 经预检 [re-91](../../reviews/re-91-M7P3c预检.md) 修正后通过，基线 main `1168fd25`；实施前把 P3c 再划为 P3c（收藏）与 P3d（最近访问）（spec §1、R1）；16 个 Task | [P3c spec](specs/P3c-favorites.md) / [P3c plan](plans/P3c-favorites.md) / [P3c 记录](reviews/P3c-favorites-review.md) |
| P3d 最近访问 | 已完成：合入 main（noff `3d9293bd`），main CI 44989539530 全绿；审查与集成见阶段记录 §9；spec 与 plan 经预检 [re-92](../../reviews/re-92-M7P3d预检.md) 修正后通过，基线 main `9d26b1c3`（P3c 的 noff）；15 个 Task；第 1 路以本机检查点 0 为基线（spec §6）；三个检查点的本地七项与分支 CI 见阶段记录 §5 | [P3d spec](specs/P3d-recents.md) / [P3d plan](plans/P3d-recents.md) / [P3d 记录](reviews/P3d-recents-review.md) |
| P4 网页的通知与关注 | 实施完成，待整分支审查：spec 与 plan 经预检 [re-93](../../reviews/re-93-M7P4预检.md) 修正后通过，基线 main `3d9293bd`（P3d 的 noff）；16 个 Task；M7 第一个页面阶段，页面故事进新的三引擎配置 `playwright.m7-pages.config.ts`；三个检查点在 T2、T11、T14 之后，本机七项与分支 CI 44999385236、45018858067、45033492263 皆 attempt 1 全绿（检查点 1 本机的 F11 (page) 以 M6-M8-07 的签名失败一次、只重跑该文件即过），见阶段记录 §5；R6（未读的点不在聚焦时重取）与 R25 的缺口进 M7 收官给负责人的报告（阶段记录 §4） | [P4 spec](specs/P4-web-notifications.md) / [P4 plan](plans/P4-web-notifications.md) / [P4 记录](reviews/P4-web-notifications-review.md) |

## 15. 决定记录

负责人 2026-11-17 的决定（四个问题都选推荐）：A 收集箱放进 `issue` 模块；B 只有项目管理员能分诊（以及按总体设计 6.5 的约定，是项目有效成员的工作区管理员）；C 通知按下述规则；D 四项都按推荐，其余取舍 E 一并按设计推进。实施中发现不合适的，仍按规则先提交再上报。下面是当时提交决定的原文。

- **A. 收集箱放进 `issue` 模块**（§4.1）：不建总体设计里的 `intake/` 模块，也不建 Plane 的 `intakes` 表，条目挂在项目上（总体设计 §5.2 的表清单在 P8 同步）。推荐：是。理由：条目就是 Triage 工作项，每次提交、分诊都写工作项、动态、关注与事件并持项目锁；另立模块要为这些写各开一条跨模块的写路径，而它自己只有一张一对一的表。
- **B. 谁能分诊**（接受、拒绝、暂缓、标重复）：Plane 的后端只放项目管理员（以及是项目成员的工作区管理员），前端却给成员显示按钮，成员点了得 403（D15）。选项：(1) 只给项目管理员（照 Plane 后端）；(2) 管理员与成员（照 Plane 前端）。推荐 (1)：权限以后端为准，成员照样能提交、评论、改自己提交的条目。
- **C. 通知的规则**（§3.2–§3.4）：触发照 Plane 的集合；粒度照 Plane 一个变化一条；删除不发通知且旧通知随之隐藏；收集箱的接受写状态动态并通知关注者；发送时按读权限复核收件人；另一端的工作项按读的人的可见显示；暂缓按读时到期；全部已读只作用于当前筛选；通知不设保留期。推荐：按此推进。
- **D. 四项跨模块或范围上的取舍**（推荐都按此推进）：
  1. 扩展 issue 已公开的事件：评论、链接事件带工作项 id，新提及作为变化发布；planning、project 新增三种归档事件（§6.1）。
  2. 收集箱的 `duplicate_to_id` 不建外键（M4 设计 4.16 原定 `SET NULL` 外键）：外键会让删除目录守卫把别的条目当作被删目标的子行，只能为它另开一类豁免、削弱守卫；不建外键则与通知、收藏、最近访问对工作项的引用同一做法，读时只给看得到的目标（§4.2、§4.3）。
  3. 已存的工作区视图照 Plane 不给工作区访客看（访客只有四个默认视图，§3.7）。
  4. 不做工作区级的迭代、模块：工作区视图没有迭代、模块词，命令面板在工作区范围不搜迭代、模块（M6-M7-03 关闭，§1）。
- **E. 其余取舍**照先例或照 Plane 推进，写在各节：新建 `notification`、`navigation` 模块且只走端口（§4.1、§6.3）；关注的接口与重新关注（§3.1）；快照与读时过滤（§3.3、§3.4）；收集箱只有应用内来源、迁移规则、重复目标的条件、访客提交者不能删、条目的读写边界与资产判定、新建不带图片、侧栏待处理数（§3.5、§3.6）；视图删去社区版写不了的列、权限、集合型列表、默认视图的显示设置存浏览器（§3.7）；收藏的校验、文件夹规则、连带、读时过滤、不加 `is_favorite`（§3.8）；最近访问的记录、上限、排序、过滤与随工作项删除（§3.9）；开关与入口（§3.11）。
