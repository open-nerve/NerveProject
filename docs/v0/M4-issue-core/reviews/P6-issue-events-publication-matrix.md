# T15 全发布点与无事件例外核验

核验基线：565fad12cf14460ebc2d8e627ebcd67fab9ce1ab 加 Task15 测试/fixture 差异。生产 app 无永久变动。`rg 'Events.Publish' server/internal/modules/issue/app` 共15处源调用，archive共享方法对应两个手动用例，因此16个实际发布用例；目录为12种事件。下表是逐行代码核对和既有真实测试索引，非声称每格新跑。最终完整 `06-make-test` 重新执行所有 Go 包，`11-make-e2e` 重新执行完整故事。

## 共通约束

- 请求路径全部经 Locks.run 的预检→gate→原事务→锁下重判；拒绝不发布。十五请求入口的隐藏目标与不存在目标由新 TestIssueEventRefusalsAreUnobservable 比较 HTTP状态、完整body和headers，publisher/任务/业务clock均0；既有权限/guest/archived/依赖/gate顺序测试由全量套件回归。
- 每次业务写一次Clock，OccurredAt复用stamp.at；系统每项目一次。Event.ID各次uuid.NewV7，非零、不等Entity.ID，批量互异；dispatcher对各订阅者保留相同ID。issue_events.go/event、collaboration_events.go和AutoArchive字面量已逐一核对；app字面量期望与真实组合断言独立核对。T15三个subscriber实测同ID、原ctx值与同pgx.Tx。
- 每行均在完整业务写（含附属行/版本/动态）完成后、同ctx发布，错误返回原事务。所有四槽未指定的值均nil，下面显式列出，空字符串`""`绝不等同nil。ID不填入文本槽。真实PG/River测试均有提交或回滚快照，而非只靠fake调用日志。

## 16个用例 → 12种类型

| 写用例 / 生产发布点 | 类型 / Entity.Kind / Entity.ID | actor、次数、完整写之后 | 精确payload表 | 真实PG/River与独立字面量测试 |
|---|---|---|---|---|
| CreateIssue（快速添加同一路径），create_issue.go:224 | issue.created / issue / 新issue ID | 请求者；1；issue/序号/关联/初始版本全部写完 | C | TestIssueCreatedEvent、TestIssueCreatedEventComposition；后者subscriber失败还核项目序号及附属行回滚 |
| UpdateIssue，update_issue.go:298 | issue.updated / issue / 当前issue ID | 请求者；实际变更1；关系/活动/版本完成 | U | TestIssueUpdatedEventEachField、SortOrder、DescriptionWindow、NoopAndSkip、Parent；TestIssueUpdatedEventComposition及HiddenOldParent；真实失败回滚七类表和job |
| DeleteIssue，delete_issue.go:112 | issue.deleted / issue / 每个子树issue ID | 请求者；子树ID升序各1；所有子树及附属软删完 | Changes空（四槽无元素） | TestIssueDeletionEvents，隐藏/归档后代、末项subscriber失败，所有issue/附属/活动快照回滚 |
| ArchiveIssue，archive_issue.go:111 | issue.archived / issue / 当前issue ID | 请求者；改变时1；归档和动态写完 | A手动 | TestIssueArchivalEvents（app+bootstrap）；真实失败回滚归档/活动/job |
| UnarchiveIssue，同共享归档发布点 | issue.unarchived / issue / 当前issue ID | 请求者；改变时1；恢复和动态写完 | A恢复 | 同上，原状态已满足时0 |
| AddSubIssues，sub_issues.go:140 | issue.updated / issue / 每个实际moved子项ID | 请求者；每moved各1；所有SetParents/动态后循环 | U.parent | TestHierarchyEvents；TestHierarchyEventsComposition核完整batch、末subscriber失败回滚 |
| CreateIssueRelations，create_relations.go:97 | issue.relation_added / issue_relation / 每个关系行ID | 请求者；每关系1（两端动态不双发）；全批完成后 | R+ | TestRelationEvents、TestRelationEventsComposition：两种ID序/各方向/跨项目，末项失败真实回滚 |
| DeleteIssueRelation，delete_relation.go:99 | issue.relation_removed / issue_relation / 原关系行ID | 请求者；1；删除/两端审计/动态后 | R− | 同上；ProjectID和两端取存储方向，与备用授权端无关；真实回滚 |
| CreateIssueComment，comments.go:63 | issue.comment_created / issue_comment / 新评论行ID | 请求者；1；评论/mention/subscriber/issue审计后 | H+ | TestCommentEventsComposition、TestCommentEventsPublicationRollback/create_comment |
| UpdateIssueComment，comment_writes.go:78 | issue.comment_updated / issue_comment / 原评论行ID | 请求者；真实清洗后内容改变1；写后，保存锁下旧HTML | H更新 | 同上/update_comment；相同清洗HTML0 |
| DeleteIssueComment，comment_writes.go:133 | issue.comment_deleted / issue_comment / 原评论行ID | 实际删除者（可为管理员而非作者）；1；评论及表情软删后 | H− | 同上/delete_comment；连带表情无独立事件 |
| AddIssueReaction，reactions.go:54 | issue.reaction_added / issue_reaction / 新表情行ID | 请求者；1；写表情和issue审计后 | Q+ issue | TestReactionEventsComposition/comment=false，PublicationRollback/add_issue_reaction |
| RemoveIssueReaction，reactions.go:102 | issue.reaction_removed / issue_reaction / 查询返回的原表情行ID | 请求者；1；软删和issue审计后 | Q− issue | 同上/remove_issue_reaction；T15错ID变异专门只破坏删除路径 |
| AddCommentReaction，comment_reactions.go:63 | issue.reaction_added / comment_reaction / 新表情行ID | 请求者；1；表情写后，不推进issue/comment审计 | Q+ comment | TestReactionEventsComposition/comment=true及PublicationRollback/add_comment_reaction |
| RemoveCommentReaction，comment_reactions.go:116 | issue.reaction_removed / comment_reaction / 查询返回的原表情行ID | 请求者；1；软删后，不推进issue/comment审计 | Q− comment | 同上/remove_comment_reaction |
| AutoArchive，auto_archive.go:85 | issue.archived / issue / 实际归档issue ID升序 | 事件 ActorID=nil；活动 actor_id 为 SQL NULL；每项目一次Clock/每issue一个新Event.ID；本项目全UPDATE/全活动后 | A系统 | TestAutoArchiveDatabaseBoundariesAndNullActor、PublishFailureRollsBackRowsActivitiesAndJob、DatabasePartialCommitAndRetry；新增ThreeSubscribersAndBatchRollback证明首/末失败原ctx整批回滚及成功3job提交 |

## 四槽逐字段核验

元组顺序为 `(Old, New, OldID, NewID)`。`文本`表示实际值，不是占位文字写入事件；`nil`为nil指针。

| 表/字段 | 精确四槽 |
|---|---|
| C.name | (nil, 清洗/trim后的实际名称, nil, nil) |
| C.description | (nil, 清洗HTML（可空串）, nil, nil) |
| C.state | (nil, 锁下状态名称, nil, 状态ID) |
| C.priority | (nil, 实际priority字符串, nil, nil) |
| C.start_date、C.target_date | (nil, YYYY-MM-DD或nil, nil, nil) |
| C.parent | (nil, 锁下父编号或nil, nil, 父ID或nil) |
| C.sort_order | (nil, float64无损g/-1/64字符串, nil, nil) |
| C.sequence_id | (nil, 十进制序号, nil, nil) |
| C.completed_at | (nil, UTC RFC3339Nano或nil, nil, nil) |
| C.assignees（按ID） | (nil, 实际显示名含默认负责人, nil, 用户ID) |
| C.labels（按ID） | (nil, 锁下label名称, nil, label ID) |
| U.name、description、priority | (真实旧文本, 真实新文本, nil, nil) |
| U.parent（含批量子项） | (真实旧父编号或`""`, 新父编号或`""`, 旧父ID或nil, 新父ID或nil)；隐藏旧父不从响应推导 |
| U.state | (旧状态名, 新状态名, 旧状态ID, 新状态ID)；不另加completed_at |
| U.start_date、target_date | (旧日期YYYY-MM-DD或`""`, 新日期YYYY-MM-DD或`""`, nil, nil) |
| U.labels / assignees增加 | (`""`, 名称/显示名, nil, 新ID) |
| U.labels / assignees删除 | (名称/显示名, `""`, 旧ID, nil) |
| U.sort_order | (旧无损数字字符串, 新无损数字字符串, nil, nil) |
| A手动 archived_at | (nil, `manual_archive`, nil, nil) |
| A系统 archived_at | (nil, `archive`, nil, nil) |
| A恢复 archived_at | (`archive`, `restore`, nil, nil) |
| R+ relation_type | (nil, 规范存储类型, nil, nil) |
| R+ issue、related_issue | (nil, nil, nil, 各自规范存储端ID) |
| R− relation_type | (规范存储类型, nil, nil, nil) |
| R− issue、related_issue | (nil, nil, 各自规范存储端ID, nil) |
| H+ comment_html | (nil, 清洗后HTML, nil, nil) |
| H更新 comment_html | (锁下旧HTML, 清洗后新HTML, nil, nil) |
| H− comment_html | (锁下旧HTML, nil, nil, nil) |
| Q+ reaction | (nil, 规范表情字符串, nil, nil) |
| Q+ issue/comment | (nil, nil, nil, 所属容器ID) |
| Q− reaction | (规范表情字符串, nil, nil, nil) |
| Q− issue/comment | (nil, nil, 所属容器ID, nil) |

创建顺序固定C表，再assignees/labels各ID序；更新按domain.UpdateActivity（name、description、parent、priority、state、dates、labels增删、assignees增删）最后sort_order。关系固定relation_type/issue/related_issue；表情固定reaction/container。四槽检查实际使用reflect.DeepEqual独立字面量，非只对字段名/数量。

## 无事件例外逐项

| 已实现写路径/分支 | 不发布事实与证据 |
|---|---|
| UpdateIssue无语义变动、同排序、同HTML | NoopAndSkip及真实UpdatedEventComposition，event/job/clock次数不增长 |
| skip_activity=true且描述标识同，仅HTML ID/关系补写 | 同测试；T15 skip-mispublication额外事件被7≠6精确捕获。skip标识变更仍正常updated，不属于例外 |
| 手动归档/恢复目标状态已满足 | app/真实TestIssueArchivalEvents幂等0事件/clock；T4实际幂等变异红 |
| AddSubIssues中已在父下的子项、父节点 | TestHierarchyEventsComposition只moved两项；不会给unchanged或父节点发事件 |
| 关联/评论/表情导致issue审计改变 | 相应Composition订阅全部12类型、数量和类型精确；不附加issue.updated |
| Comment清洗后同HTML | TestCommentEventsComposition内容不变、count/clock不增长 |
| 重复表情、缺失、权限/归档拒绝 | TestReactionEventsComposition duplicate409/absent404；app Refusals 及真实权限/全量矩阵；新增15入口隐藏404严格同不存在且publisher0 |
| issue删除连带评论、两表情、关系 | TestIssueDeletionEvents整子树仅issue.deleted；完整附属快照。无连带collaboration事件 |
| project/workspace级联删除issue及附属 | Cascade不依赖publisher；TestIssueDeletionEventsCascade真实删除、job0；P9才发高层事件 |
| DetachLabel | Cascade.DetachLabel仅调用自有Store.DetachLabels，不持有publisher，不增issue.updated；边界静态核验+原标签回归，并非伪称专有事件测试 |
| AutoArchive无候选/策略关闭/归档删除项目/无终态/无匹配行/重试 | app RechecksPolicyUnderLock；真实Boundaries、PartialCommitAndRetry、T12双向交错，0事件、无多余clock/活动 |
| 当前Read/list/settings验证 | 无业务发布写入口；列表偏好写属project/workspace既有模块，本期不接publisher；P8草稿、P7链接尚未实现，不能列为已实现发布点 |

真实回滚对应T3创建/更新、T4生命周期、T5批量父子/关系、T6七种评论/表情、T10系统归档均有数据库与River任务快照。本Task补三subscriber及首/末失败；历史证据不以fake日志替代。
