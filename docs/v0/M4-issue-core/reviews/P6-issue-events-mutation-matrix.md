# T15 最低绑定生产变异矩阵

不是任选五项/十五项：逐项映射spec §2.16全部最低集合。共20项（14项复用历史真实证据、6项T15新执行），安全、事务、锁、完整性均有实际变异；第1项明确为隐藏404边界，不把payload重命名安全。每项变异退出1、恢复退出0。原日志/patch/恢复记录已逐个读取；`task-15-evidence/prior-evidence-audit.json`保存核验所读文件和哈希。

下文证据路径相对本 worktree 的 ignored `.superpowers/sdd/P6-issue-events/`，不在正式 `reviews/` 目录内；表中 T15 新证据的裸文件名另相对该根下的 `task-15-evidence/`。同一格省略目录的历史文件名，沿用该格已点名的 `task-N-evidence/` 目录。归档后按归档清单中的同名相对路径定位。T15所有新命令/实际RUN计数/退出/HEAD/环境在`task-15-evidence/{mutation,restored}-名称.json`，原stdout/stderr合流保存`.log`，精确生产补丁`名称.patch`，逐字恢复SHA256在`名称-restoration.json`。历史项沿原任务报告及原始命令文件引用，不冒称当前新跑；当前完整门禁重新跑恢复后的整仓行为。

| # | 最低集合/分类 | 生产变异与检出层 | 实际红/恢复证据 |
|---|---|---|---|
| 1 | 安全：隐藏404 | Locks.decide把ErrNotVisible映为Forbidden；真实HTTP15写入口全部403而不存在404，完整body/headers比较失败。publisher仍0不代表安全允许改变 | **T15新** `hidden-404-to-403.patch`；`mutation-hidden-404-to-403.log`/`restored-hidden-404-to-403.log`，16 RUN（含父），15叶均红→绿 |
| 2 | Publish移出事务 / 事务 | T4将DeleteIssue批量Publish移到Locks.run返回后；真实subscriber无Tx，三项子树rows/activities已经错误提交 | `task-4-evidence/09-mutant-outside-tx.txt`内含完整patch/命令/EXIT1；`10-restore-all.txt` EXIT0 |
| 3 | Publish吞错 / 事务 | T6创建评论忽略Publish错误；app原错误链断言nil；真实PG用例HTTP201≠500，在回滚快照前即检出 | `task-6-evidence/17-swallow-publish.patch/.log`；`18-restored-swallow-publish.log`，命令/退出见各原始日志首部 |
| 4 | 只发根 / 事件 | T4子树循环替换为仅请求id；真实组合events1≠完整子树3 | `task-4-evidence/03-mutant-root.txt`（patch+EXIT1）；`04-restore-root.txt` EXIT0。Cascade父节点被selector访问但未跑其子项，不算覆盖 |
| 5 | 关系双发 / 事件 | T5每关系Publish两次；app和真实组合4 events≠2关系（两端动态4不是理由） | `task-5-evidence/08-double-publish.patch`、`08-double-publish-red.log`；`09-double-publish-restored.log`；原commands.txt给退出 |
| 6 | sort_order漏发 / 事件 | T3屏蔽sortOrder Change；字面量完整payload及真实组合SortOrder缺失 | `task-3-evidence/mutation-ignore-sort.diff/.log`；`restore-ignore-sort.log`；mutation-results.json红1/恢复0、逐字一致 |
| 7 | skip误发 / 事件 | 同内容身份HTML补写分支writeMentions后强行Publish；真实组合events/jobs7≠6 | **T15新** `skip-mispublication.patch`，`mutation-skip-mispublication.log`/`restored-skip-mispublication.log`；实际1顶层红→绿 |
| 8 | 删除表情错ID / 事件 | 只把RemoveIssueReaction的Entity.ID改成current.IssueID，新增路径不变；真实subscriber按实体读已删表情行失败，DELETE500≠204（HTTP层先检出，非伪称到达最终envelope） | **T15新** `deleted-reaction-wrong-id.patch`，对应mutation/restored日志；1父+comment=false叶，1→0 |
| 9 | NULL换零UUID / 系统身份 | T10 ActivityRow.ActorID从nil变eventPtr(uuid.UUID{})；app逐行system activity actor不是nil | `task-10-evidence/mutation-zero-actor.log`含patch、实际命令/EXIT1；`18-affected-regression.log`恢复广回归0；当前长度guard改进不改变ActorID断言 |
| 10 | 漏worker：periodic / 完整性 | 删除periodic Args.Kind独立存在性检查；New错误地返回nil错误，期望has no worker | `task-7-evidence/periodic-integrity.patch`、`mutation-periodic-integrity.log`、`restored-periodic-integrity.log`；mutations.json 1→0 |
| 11 | 漏worker：enqueued / 完整性 | 删除EnqueuedKinds独立检查；同样New错误地成功；保留另一worker使River自身不掩盖检查 | `task-7-evidence/enqueued-integrity.patch`、mutation/restored-enqueued-integrity.log；mutations.json 1→0 |
| 12 | <=换< / 阈值 | 自有SQL updated_at<=cutoff变<，sqlc成功生成；真实PG exact cutoff行缺失/archived_at nil/审计未变 | `task-9-evidence/mutation-cutoff-exclusive.patch/.log`；`restored-cutoff-exclusive.log/.txt`及generate/restore-gen日志；1→0 |
| 13 | 锁前读策略 / 锁 | T12 SQL先MATERIALIZED prior读旧项目，再锁，返回prior；真实request_first的阈值改大/禁用/归档仍错误归档，archived1≠0，业务/活动/任务错误保留 | `task-12-evidence/mutation-no-reread.patch/.log`；`restored-race-5.log`和run-summary.json。只采用sqlc成功后的此patch，不采用not-evidence-no-reread-before-generation日志 |
| 14 | 锁前读clock / 锁/时钟 | T12把Now移至事务/锁前；等待期间29天推进31天后却归档0≠1，活动/事件/job均0 | `task-12-evidence/mutation-early-clock.patch/.log`；`restored-race-5.log`及run-summary.json |
| 15 | 状态过滤遗漏 / 状态 | T14移除domain.Archivable过滤；旧started误归档，后续API更新completed409≠200 | `task-14-evidence/mutant-state.diff`；mutant-state-build.json构建0、mutant-state-i16.json/stdout测试1（1匹配）；restored-build/restored-i16 JSON分别0，final-i16/full-e2e也0 |
| 16 | fixture误认历史完成 / 时钟 | T14等待器删除attempted_at和finalized_at两个界线（保留参数类型）；真实准备期historical completed被返回，违反must not satisfy freshness | `task-14-evidence/mutant-historical.diff`、mutant-historical-build/i16.json/stdout；恢复同第15项。1匹配，1→0 |
| 17 | T10 Minor诊断 | 自动归档重复InsertActivities行；新增failfast精确长度立即报activities/events=6/3 want3/3；不再先assert再index panic | **T15新** `duplicate-activities.patch`；对应mutation/restored日志，1 RUN，1→0，红日志无panic |
| 18 | T12 Minor / 真实锁 | 给Execute外围加whole-batch WithinTx；第二States屏障时第一项目锁仍被持有，RowWaits实测失败；恢复后第一锁释放/第二N仍持有、IDs顺序和count1/job1正确 | **T15新** `whole-batch-transaction.patch`；对应mutation/restored日志，-race -count=1；最终05-race-five也绿 |
| 19 | T13 Minor / 真实进程 | advance省略旧进程stop；对确切child.exitCode立即断言0，实际null；不靠不同URL或端口轮询，finally仍停第一进程 | **T15新** `clock-omit-stop.patch`；对应mutation/restored日志，Playwright实际1失败→1通过 |
| 20 | fixture跨写入旧扫描 / 时钟 | T14等待器attempted_at>watermark误改finalized_at>watermark，保留spawn界线；真实扫描开始早于最后写入、完成晚于写入仍被误接受 | `task-14-evidence/mutant-finalized-only.diff`、mutant-finalized-only-i16.json/stdout；实际crossing attempt must not satisfy freshness，1→0；恢复同15 |

## 证据边界与修复

- 六个T15生产变异正常运行后产生上述业务红，没有INFRA/编译失败充作检出，也没有幸存变异。T13 dump/attach探针是独立故障诊断，不充作上表生产变异数量。
- 第17项改善的是测试诊断，原生产实现正确；baseline绿→生产duplicate变异红→恢复绿。第18项增强证明第一项目提交释放锁，不把旧的“第二项目尚未开始”替代它。
- 旧T3初次600秒期望错误、T12最初未成功生成SQL、T14探索actor类型错误均不计入。旧日志NO_COLOR/FORCE_COLOR冲突保留原样；新命令只设NO_COLOR且移除FORCE_COLOR。
- 三subscriber和batch-last测试是新的真实事务证据（第一InsertTx，第二sentinel，第三零调用；业务/全部活动/前项job全回滚，成功对照3个job提交），不是把app fake记录重新命名为PG证据。

## T16 证据补记

上文“新命令只设NO_COLOR且移除FORCE_COLOR”准确记录T15原执行环境，不表示其Playwright子进程无警告。T15独评后的定向验证同时移除两个变量，在`4a489125`上实际1/1、退出0、无颜色警告；原日志仍照旧保留。详见正式[P6 review §2](P6-issue-events-review.md#2-验收标准逐条核对)。
