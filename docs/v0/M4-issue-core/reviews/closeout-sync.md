# M4 收尾文档同步逐项核对

2026-11-02；T11 文档/静态检查点。当前产品 `5bd935d8`；最终门禁、C12影响格和集成签结另由T13/T14完成。

## 1. 方法与边界

以T1冻结的M4§3.23原44行、§4.17原45行为键。目标完整段落已阅读并与当前实现、表/索引和测试行为核对；脚本只绑定手工结论与原文/hash，不用关键字命中代替审核。原行全文、SHA、目标文件SHA及差异行/测试SHA在 `t11-sync-mapping.json`；M2/M3目标完整段落在 `t11-target-sections.json`，其余全文阅读见 `source-read-ledger.json`。

43项同步核对完成，SYNC44保留集成待办；45项差异有当前行和行为证据。这里只证明文档一致，不把历史阶段通过当作最新产品全量验收。T2/T3服务端回归和并发原件见[合同复核](closeout-server-contracts.md)，T8/T9规模证据见[验收台账](closeout-acceptance.md)。

## 2. 44条同步

| 原文键（M4§3.23顺序） | 当前目标 | 逐项事实及结论 | 对应实施证据 |
|---|---|---|---|
| SYNC01 · P1a / 总体设计 / 4.2 | [4.2](../../v0-design.md) | 先项目锁后其他锁；拒绝预检不锁工作区；删除工作区先锁项目再读时刻；同项目写/工作区删除的串行代价；**已核对** | [closeout-server-contracts](closeout-server-contracts.md) |
| SYNC02 · P1b / 总体设计 / 4.2 | [4.2](../../v0-design.md) | 连接前进项目gate；K/排队上限/lock_timeout与503 Retry-After；热点仍消耗池连接，T9实测另列；**已核对** | [closeout-server-contracts](closeout-server-contracts.md) |
| SYNC03 · P1b / 总体设计 / 6.2 | [6.2](../../v0-design.md) | platform/gate只给按键context信号量，业务资格在模块内；**已核对** | [closeout-server-contracts](closeout-server-contracts.md) |
| SYNC04 · P1a / 总体设计 / 6.2 | [6.2](../../v0-design.md) | issue独占13表；app端口与adapter/content清洗分工已补齐；**已核对** | [P4-issue-collaboration-review](P4-issue-collaboration-review.md) |
| SYNC05 · P1a / M3 设计 / 3.6 的取锁表、17.4 | [3.6、17.4](../../M3-workspace-project/M3-design.md) | 项目N锁和工作区删除项目锁顺序一致；M4不继承全部工作区锁；**已核对** | [P9-purge-review](P9-purge-review.md) |
| SYNC06 · P1a / M3 设计 / 3.17 | [3.17](../../M3-workspace-project/M3-design.md) | 含已归档工作项仍阻止删状态，409 project.state_in_use；**已核对** | [P1a-issues-review](P1a-issues-review.md) |
| SYNC07 · P1a / 差异清单 / 二·按表 | [二·按表](../../plane-diff.md) | issues/assignees/labels/subscribers/activities逐列、索引、审计及软删约束已登记；**已核对** | [P1a-issues-review](P1a-issues-review.md) |
| SYNC08 · P1a / 差异清单 / 四 | [四](../../plane-diff.md) | P1a编号/引用/默认指派关注/状态/写锁行为，见本表后DIFF逐条映射；**已核对** | [P1a-issues-review](P1a-issues-review.md) |
| SYNC09 · P1b / 差异清单 / 四 | [四](../../plane-diff.md) | P1b连接前闸门/200详情/负责人行返回已登记；**已核对** | [P1b-issue-update-review](P1b-issue-update-review.md) |
| SYNC10 · P2 / 总体设计 / 6.5 | [6.5](../../v0-design.md) | AllowCreator须有效项目成员且非访客；OwnOnly使用项目角色，workspace-admin例外不绕过可见性；**已核对** | [closeout-server-contracts](closeout-server-contracts.md) |
| SYNC11 · P2 / M2 设计 / 3.11 | [3.11](../../M2-auth/M2-design.md) | date格式、按数值下标排序、多类型节点的数范围；通用验证不由issue域复制；**已核对** | [P2-issue-access-review](P2-issue-access-review.md) |
| SYNC12 · P2 / 差异清单 / 四 | [四](../../plane-diff.md) | P2日期/创建者/OwnOnly/归档/编号折叠与错误优先级，DIFF05/15–20；**已核对** | [P2-issue-access-review](P2-issue-access-review.md) |
| SYNC13 · P3 / 总体设计 / 3.4 | [3.4](../../v0-design.md) | 子项/关联/链接/工作项表情是data集合，非游标页；**已核对** | [P3-issue-hierarchy-review](P3-issue-hierarchy-review.md) |
| SYNC14 · P3 / 总体设计 / 3.6 | [3.6](../../v0-design.md) | IssueRelation.issue内嵌列表行是第二项ID-only例外；**已核对** | [P3-issue-hierarchy-review](P3-issue-hierarchy-review.md) |
| SYNC15 · P3 / 总体设计 / 4.2 | [4.2](../../v0-design.md) | 跨项目按ID升序锁项目；关联行可跨项目的约定五例外；**已核对** | [closeout-server-contracts](closeout-server-contracts.md) |
| SYNC16 · P3 / 差异清单 / 二·按表、四 | [二·按表、四](../../plane-diff.md) | relations列/唯一对/方向/删除/访问与父子规则，DIFF21–26；**已核对** | [P3-issue-hierarchy-review](P3-issue-hierarchy-review.md) |
| SYNC17 · P4 / 总体设计 / 1.1 | [1.1](../../v0-design.md) | 评论无回复；只保留可查看和还原的描述历史；**已核对** | [P4-issue-collaboration-review](P4-issue-collaboration-review.md) |
| SYNC18 · P4 / 总体设计 / 5.2、5.3、6.6、6.7 | [5.2、5.3、6.6、6.7](../../v0-design.md) | 44表，无issue_versions；描述活版本保留最新20，墓碑另由purge；**已核对** | [P9-purge-review](P9-purge-review.md) |
| SYNC19 · P4 / M2 设计 / 3.12 | [3.12](../../M2-auth/M2-design.md) | shared统一DefaultPageSize/MaxPageSize，模块消费而不另写规则；**已核对** | [P4-issue-collaboration-review](P4-issue-collaboration-review.md) |
| SYNC20 · P4 / 差异清单 / 一、二·按表、四 | [一、二·按表、四](../../plane-diff.md) | issue_versions删除及5协作表逐列；内容清洗/历史/时间线/表情，DIFF08–10/27–34；**已核对** | [P4-issue-collaboration-review](P4-issue-collaboration-review.md) |
| SYNC21 · P5 / 总体设计 / 3.3 | [3.3](../../v0-design.md) | none/重复日期/单日；格式和词值422，不支持否定/OR；**已核对** | [P5-issue-list-review](P5-issue-list-review.md) |
| SYNC22 · P5 / 总体设计 / 3.4 | [3.4](../../v0-design.md) | total/subgroups/subgroup_totals/RO-RR快照/组格游标；当前21排序码，P5历史19保留；**已核对** | [P8-drafts-search-review](P8-drafts-search-review.md) |
| SYNC23 · P5 / 总体设计 / 6.2 | [6.2](../../v0-design.md) | issue/domain词汇；project/app声明IssueListSettings，Provide纯校验，未引入反向app依赖；**已核对** | [P5-issue-list-review](P5-issue-list-review.md) |
| SYNC24 · P5 / M3 设计 / 3.18 | [3.18](../../M3-workspace-project/M3-design.md) | 三列filters/display_filters/display_properties无rich_filters；原生前端已迁移；link_count由P8提供；**已核对** | [P10b-web-list-data-review](P10b-web-list-data-review.md) |
| SYNC25 · P5 / 差异清单 / 二·按表、三、四 | [二·按表、三、四](../../plane-diff.md) | preferences三列/14列表，原生路径、排序分组名；不存在旧适配层；**已核对** | [P5-issue-list-review](P5-issue-list-review.md) |
| SYNC26 · P6 / 总体设计 / 6.2、6.3 | [6.2、6.3](../../v0-design.md) | shared值事件+EventPublisher，同事务同步订阅；异步经InsertTx，失败回滚；**已核对** | [closeout-server-contracts](closeout-server-contracts.md) |
| SYNC27 · P6 / 总体设计 / 6.7 | [6.7](../../v0-design.md) | default/periodic两队列、三周期与RunOnStart；业务偏移不改变租约/限时真实钟；**已核对** | [P6-issue-events-review](P6-issue-events-review.md) |
| SYNC28 · P6 / M2 设计 / 3.15、3.17 | [3.15、3.17](../../M2-auth/M2-design.md) | NewInserter在模块构造前；声明请求kind/periodic缺worker启动失败；**已核对** | [closeout-server-contracts](closeout-server-contracts.md) |
| SYNC29 · P6 / 差异清单 / 四 | [四](../../plane-diff.md) | 自动归档系统actor为空，配置间隔非每日固定时刻；**已核对** | [P6-issue-events-review](P6-issue-events-review.md) |
| SYNC30 · P7 / 总体设计 / 6.2、8（安全） | [6.2、8.1](../../v0-design.md) | outbound单一SSRF边界，拨号前固定DNS、重定向/本机/特殊网段限制与M8复用；**已核对** | [P7-issue-links-review](P7-issue-links-review.md) |
| SYNC31 · P7 / 差异清单 / 二·按表、四 | [二·按表、四](../../plane-diff.md) | links 12列/约束；1MiB HTML/64KiB位图、metadata blocked/failed及地址变化清空；**已核对** | [P7-issue-links-review](P7-issue-links-review.md) |
| SYNC32 · P7 / README / "安全"一节 | [部署、链接抓取与出站安全](../../../../README.md) | 并发/间隔/锁预算、IANA与本机网卡、端口扫描边界/网络策略均声明；**已核对** | [P7-issue-links-review](P7-issue-links-review.md) |
| SYNC33 · P8 / 总体设计 / 5.4 | [5.4](../../v0-design.md) | draft project_id列，发布锁保存payload且issue.create、draft_changed 409；**已核对** | [P8-drafts-search-review](P8-drafts-search-review.md) |
| SYNC34 · P8 / 总体设计 / 6.9 | [6.9](../../v0-design.md) | 只查标题/编号；空词最近；精确编号/折叠/日期、选择器排除环，不搜stripped；**已核对** | [P8-drafts-search-review](P8-drafts-search-review.md) |
| SYNC35 · P8 / 差异清单 / 二·按表、四 | [二·按表、四、五](../../plane-diff.md) | draft九列/三个索引；活issues.name trigram部分GIN；搜索/发布/活link_count；**已核对** | [P8-drafts-search-review](P8-drafts-search-review.md) |
| SYNC36 · P8 / README / "部署"一节 | [部署](../../../../README.md) | pg_trgm trusted extension需数据库CREATE；runtime显式函数授权边界；**已核对** | [P8-drafts-search-review](P8-drafts-search-review.md) |
| SYNC37 · P9 / 总体设计 / 5.5、6.7 | [5.5、6.7](../../v0-design.md) | 23表单任务逐模块清理，严格60天cutoff；活描述版本与父CASCADE/迟到40P01边界；**已核对** | [P9-purge-review](P9-purge-review.md) |
| SYNC38 · P9 / 总体设计 / 5.5、6.3 | [5.5、6.3](../../v0-design.md) | 项目/工作区事件原事务投递、请求At/By；立即不可见与每项目约2万工作项目标；**已核对** | [P9-purge-review](P9-purge-review.md) |
| SYNC39 · P9 / M3 设计 / 3.6、13.2 | [3.6、13.2](../../M3-workspace-project/M3-design.md) | 删除发布事件，去掉IssueCascade同步删除；保留StateInUse/DetachLabel及M3同步侧；**已核对** | [P9-purge-review](P9-purge-review.md) |
| SYNC40 · P9 / 差异清单 / 四 | [四](../../plane-diff.md) | 活动/评论链物理CASCADE；issue/state/label同步，项目/工作区issue侧异步；**已核对** | [P9-purge-review](P9-purge-review.md) |
| SYNC41 · P10b / 总体设计 / 7.7 | [7.7](../../v0-design.md) | canonical单实体/updated_at；Reconciled.fetchPart/paged-groups；IssueRootStore统一范围失效，R06源事实修正；**已核对** | [P10b-web-list-data-review](P10b-web-list-data-review.md) |
| SYNC42 · P10a–P16 / 前端改动清单 / 3.1、3.2 | [3.1、3.2](../../frontend-changes.md) | P10a–P16已集成；M6/M7领域页面未接通与列表撤下仍明示，不冒充可用；**已核对** | [P16-web-cleanup-review](P16-web-cleanup-review.md) |
| SYNC43 · P12 / README / "前端"一节 | [前端](../../../../README.md) | 四布局/原生搜索及详情协作草稿、附件关闭和后续页面边界；**已核对** | [P12-web-list-pages-review](P12-web-list-pages-review.md) |
| SYNC44 · 收尾 / 总体设计 / 9.4 | [9.4](../../v0-design.md) | 保持进行中；只在T14获准集成且main CI通过后改已完成；**待T14集成** | [closeout-acceptance](closeout-acceptance.md) |

## 3. 45条行为差异

以下测试均在全期全文审查范围；测试文件只是可追溯入口，不能代替运行收据。T2/T3/T9已有实际运行；最终产品仍待T13完整F。原差异的Plane侧原文保留不改。

| 原文键/行为 | 差异清单现行落点 | 真实行为测试 | 结论 |
|---|---|---|---|
| DIFF01 工作项的写的取锁 | [§四，第308行](../../plane-diff.md) | [modules/issue/app/lock_test.go](../../../../server/internal/modules/issue/app/lock_test.go) | 一致 |
| DIFF02 一个项目上的并发写 | [§四，第309行](../../plane-diff.md) | [platform/gate/gate_test.go](../../../../server/internal/platform/gate/gate_test.go) | 一致 |
| DIFF03 删除项目、工作区时工作项一侧的行 | [§四，第302行](../../plane-diff.md) | [modules/issue/adapter/postgres/delete_relations_test.go](../../../../server/internal/modules/issue/adapter/postgres/delete_relations_test.go) | 一致 |
| DIFF04 负责人、标签不合格 | [§四，第312行](../../plane-diff.md) | [modules/issue/adapter/http/create_payload_test.go](../../../../server/internal/modules/issue/adapter/http/create_payload_test.go) | 一致 |
| DIFF05 起止日期 | [§四，第317行](../../plane-diff.md) | [modules/issue/adapter/http/dates_test.go](../../../../server/internal/modules/issue/adapter/http/dates_test.go) | 一致 |
| DIFF06 默认负责人 | [§四，第313行](../../plane-diff.md) | [modules/issue/app/issue_events_test.go](../../../../server/internal/modules/issue/app/issue_events_test.go) | 一致 |
| DIFF07 负责人的动态 | [§四，第314行](../../plane-diff.md) | [modules/issue/app/issue_events_test.go](../../../../server/internal/modules/issue/app/issue_events_test.go) | 一致 |
| DIFF08 只补块 id 的保存 | [§四，第334行](../../plane-diff.md) | [modules/issue/adapter/postgres/description_test.go](../../../../server/internal/modules/issue/adapter/postgres/description_test.go) | 一致 |
| DIFF09 描述的动态的合并 | [§四，第338行](../../plane-diff.md) | [modules/issue/adapter/postgres/description_activity_test.go](../../../../server/internal/modules/issue/adapter/postgres/description_activity_test.go) | 一致 |
| DIFF10 自动关注 | [§四，第315行](../../plane-diff.md) | [modules/issue/adapter/postgres/description_test.go](../../../../server/internal/modules/issue/adapter/postgres/description_test.go) | 一致 |
| DIFF11 编号 | [§四，第307行](../../plane-diff.md) | [modules/issue/adapter/http/create_payload_test.go](../../../../server/internal/modules/issue/adapter/http/create_payload_test.go) | 一致 |
| DIFF12 删除状态 | [§四，第316行](../../plane-diff.md) | [modules/project/app/delete_state_test.go](../../../../server/internal/modules/project/app/delete_state_test.go) | 一致 |
| DIFF13 修改的响应 | [§四，第310行](../../plane-diff.md) | [modules/issue/adapter/http/update_test.go](../../../../server/internal/modules/issue/adapter/http/update_test.go) | 一致 |
| DIFF14 负责人的读取 | [§四，第311行](../../plane-diff.md) | [modules/issue/adapter/http/issues_test.go](../../../../server/internal/modules/issue/adapter/http/issues_test.go) | 一致 |
| DIFF15 创建者的权限 | [§四，第318行](../../plane-diff.md) | [modules/issue/app/guest_rule_test.go](../../../../server/internal/modules/issue/app/guest_rule_test.go) | 一致 |
| DIFF16 访客只看自己的工作项 | [§四，第319行](../../plane-diff.md) | [modules/issue/app/guest_rule_test.go](../../../../server/internal/modules/issue/app/guest_rule_test.go) | 一致 |
| DIFF17 访客删除自己建的工作项 | [§四，第320行](../../plane-diff.md) | [modules/issue/app/guest_rule_test.go](../../../../server/internal/modules/issue/app/guest_rule_test.go) | 一致 |
| DIFF18 已归档的工作项 | [§四，第321行](../../plane-diff.md) | [modules/issue/adapter/http/archive_test.go](../../../../server/internal/modules/issue/adapter/http/archive_test.go) | 一致 |
| DIFF19 归档、恢复 | [§四，第322行](../../plane-diff.md) | [modules/issue/adapter/http/archive_test.go](../../../../server/internal/modules/issue/adapter/http/archive_test.go) | 一致 |
| DIFF20 按编号取 | [§四，第323行](../../plane-diff.md) | [modules/issue/adapter/http/by_identifier_test.go](../../../../server/internal/modules/issue/adapter/http/by_identifier_test.go) | 一致 |
| DIFF21 删除工作项 | [§四，第324行](../../plane-diff.md) | [modules/issue/adapter/http/delete_test.go](../../../../server/internal/modules/issue/adapter/http/delete_test.go) | 一致 |
| DIFF22 父工作项 | [§四，第325行](../../plane-diff.md) | [modules/issue/adapter/http/sub_issues_test.go](../../../../server/internal/modules/issue/adapter/http/sub_issues_test.go) | 一致 |
| DIFF23 加子工作项的动态 | [§四，第326行](../../plane-diff.md) | [modules/issue/app/issue_hierarchy_events_test.go](../../../../server/internal/modules/issue/app/issue_hierarchy_events_test.go) | 一致 |
| DIFF24 子工作项数 | [§四，第327行](../../plane-diff.md) | [modules/issue/adapter/postgres/sub_issues_test.go](../../../../server/internal/modules/issue/adapter/postgres/sub_issues_test.go) | 一致 |
| DIFF25 关联 | [§四，第328行](../../plane-diff.md) | [modules/issue/adapter/http/relations_test.go](../../../../server/internal/modules/issue/adapter/http/relations_test.go) | 一致 |
| DIFF26 关联的动态 | [§四，第329行](../../plane-diff.md) | [modules/issue/app/issue_hierarchy_events_test.go](../../../../server/internal/modules/issue/app/issue_hierarchy_events_test.go) | 一致 |
| DIFF27 空评论 | [§四，第335行](../../plane-diff.md) | [modules/issue/adapter/http/comment_writes_test.go](../../../../server/internal/modules/issue/adapter/http/comment_writes_test.go) | 一致 |
| DIFF28 改、删别人的评论 | [§四，第336行](../../plane-diff.md) | [modules/issue/adapter/http/comment_writes_test.go](../../../../server/internal/modules/issue/adapter/http/comment_writes_test.go) | 一致 |
| DIFF29 表情的值 | [§四，第337行](../../plane-diff.md) | [modules/issue/adapter/http/reactions_test.go](../../../../server/internal/modules/issue/adapter/http/reactions_test.go) | 一致 |
| DIFF30 评论、表情的动态 | [§四，第339行](../../plane-diff.md) | [modules/issue/app/collaboration_events_test.go](../../../../server/internal/modules/issue/app/collaboration_events_test.go) | 一致；DIFF30仅不写活动，已发布评论删除等值事件 |
| DIFF31 提及的同步 | [§四，第340行](../../plane-diff.md) | [modules/issue/adapter/postgres/description_test.go](../../../../server/internal/modules/issue/adapter/postgres/description_test.go) | 一致 |
| DIFF32 描述的历史 | [§四，第341行](../../plane-diff.md) | [modules/issue/adapter/postgres/description_versions_test.go](../../../../server/internal/modules/issue/adapter/postgres/description_versions_test.go) | 一致 |
| DIFF33 动态的列表 | [§四，第342行](../../plane-diff.md) | [modules/issue/adapter/http/activities_test.go](../../../../server/internal/modules/issue/adapter/http/activities_test.go) | 一致 |
| DIFF34 时间线的增量读取 | [§四，第343行](../../plane-diff.md) | [modules/issue/adapter/http/activities_test.go](../../../../server/internal/modules/issue/adapter/http/activities_test.go) | 一致 |
| DIFF35 列表的分组、排序 | [§四，第293行](../../plane-diff.md) | [modules/issue/adapter/http/list_issues_test.go](../../../../server/internal/modules/issue/adapter/http/list_issues_test.go) | 一致 |
| DIFF36 已归档的项目的工作项列表 | [§四，第294行](../../plane-diff.md) | [modules/issue/adapter/http/list_issues_test.go](../../../../server/internal/modules/issue/adapter/http/list_issues_test.go) | 一致 |
| DIFF37 个人主页的工作项 | [§四，第295行](../../plane-diff.md) | [modules/issue/adapter/http/list_member_issues_test.go](../../../../server/internal/modules/issue/adapter/http/list_member_issues_test.go) | 一致 |
| DIFF38 显示设置 | [§四，第296行](../../plane-diff.md) | [modules/project/adapter/http/issue_preferences_test.go](../../../../server/internal/modules/project/adapter/http/issue_preferences_test.go) | 一致 |
| DIFF39 自动归档的执行者 | [§四，第289行](../../plane-diff.md) | [modules/issue/adapter/postgres/auto_archive_test.go](../../../../server/internal/modules/issue/adapter/postgres/auto_archive_test.go) | 一致 |
| DIFF40 定时任务的时刻 | [§四，第297行](../../plane-diff.md) | [platform/jobs/registration_test.go](../../../../server/internal/platform/jobs/registration_test.go) | 一致 |
| DIFF41 链接的地址 | [§四，第291行](../../plane-diff.md) | [modules/issue/adapter/http/link_writes_test.go](../../../../server/internal/modules/issue/adapter/http/link_writes_test.go) | 一致 |
| DIFF42 链接的抓取 | [§四，第292行](../../plane-diff.md) | [platform/outbound/policy_registry_test.go](../../../../server/internal/platform/outbound/policy_registry_test.go) | 一致 |
| DIFF43 草稿 | [§四，第298行](../../plane-diff.md) | [modules/issue/adapter/http/drafts_test.go](../../../../server/internal/modules/issue/adapter/http/drafts_test.go) | 一致 |
| DIFF44 搜索 | [§四，第299行](../../plane-diff.md) | [modules/issue/adapter/http/search_issues_test.go](../../../../server/internal/modules/issue/adapter/http/search_issues_test.go) | 一致 |
| DIFF45 物理清理 | [§四，第300行](../../plane-diff.md) | [modules/issue/adapter/postgres/purge_test.go](../../../../server/internal/modules/issue/adapter/postgres/purge_test.go) | 一致 |

## 4. 本次文档修正

- 将P10b/P11/P16已集成事实与历史快照区分；README、M3前端消费链、差异清单和前端改动表同步。M6/M7自身接口和已撤下列表继续明确不可用边界。
- 总体列表排序写当前21码并保留P5的19码历史；自动归档改为配置间隔及RunOnStart；补adapter/content清洗端口，domain保持纯净。
- 评论/表情不写活动不等于不发事件：`issue.comment_deleted`由app删除事务发布，domain/events.go和collaboration_events_test.go已有证据；删除审计与通知订阅入口保留。
- 未改M4§14复合项为已完成，未改总体§9.4为已完成。

## 5. 静态终数入口

原AST当前354候选、102来源组与历史52逐名追溯见[死成员审计](closeout-dead-members.md)。14个lint范围实际86警告，M4域/全期改动路径零诊断，no-shadow为0且error；原始raw、全期触及路径并集收据和13例外逐实例保留在T11证据根。
