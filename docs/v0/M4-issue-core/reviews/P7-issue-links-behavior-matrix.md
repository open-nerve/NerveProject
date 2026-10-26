# M4/P7 链接行为与最终本地门禁矩阵

## 证据版本与读法

- Task15 BASE：`7b44d2ef7db03ab922b7a289fabdfe509951bb7d`；本矩阵逐项核对该提交的实际源码、测试和 T1–T14 原始记录。
- 下表每一行的最终正常源码执行版本均为 **tested HEAD `6e28cc1637e99afc754d3af57a583d45e57831c8`**，tree=`fddd6b9d7764aabbcf6f3696ee4ffcc45cd47620`；2026-10-27本地七门禁实测全绿。历史变异/race另标历史版本，不冒称最终HEAD重新制造过红或跑过race。
- 路径缩写：`I=server/internal/modules/issue`，`A=I/app`，`D=I/domain`，`H=I/adapter/http`，`P=I/adapter/postgres`，`C=I/adapter/crawler`，`B=server/internal/bootstrap`。测试名均为真实函数名；`E=.superpowers/sdd/P7-issue-links` 为 ignored 原始台账。
- 权威定义：[spec §2.3–2.8](../specs/P7-issue-links.md)、[03 规则](../../claude_to_gpt/03-rules-and-pitfalls.md)。这里只归档 Task15 核验，不提前勾选 P7 阶段验收或代替 T16 deep-review。

## 四个 operation 与拒绝码

下表列出 operation 自身声明的全部业务码；共通 401/400 另列。H 的 `TestMain → apitest.Main` 在完整包成功后要求每个自身声明码实际答到；每次 `CheckResponse` 同时拒绝未声明码。下表中的拒绝表主要比较 status，**公开 code 的双向结论来自完整、无 `-run/-skip/-short` 的 H 包运行**，不是从相同 status 推断 code。

| operation / 契约 | 成功、输入与存储性质 | 全部业务拒绝码及对应真实断言 | 层 / 测试 |
|---|---|---|---|
| `listIssueLinks`；GET `/api/v0/issues/{issue_id}/links` | 200；空 `data:[]`；创建时间、ID 降序；nullable title/creator；ReadTx 内 ActionRead | `issue.not_found`、`forbidden`；missing/hidden/deleted 父项逐值同答，归档仍按读权限 | H `TestListIssueLinksHTTP`、`TestCreateIssueLinkHTTPRefusals`；B `TestListIssueLinksAPI`、`TestIssueLinkHiddenAnswers`、`TestPermissionMatrix/listIssueLinks` |
| `createIssueLink`；POST 同集合 | 201；`{url,title?}`；2048 Unicode 原文实存；省略/null title→NULL、空串保留；pending `{}`；跨 issue 可复用 URL | `validation_failed`、`issue.not_found`、`forbidden`、`issue.link_exists`、`project.archived`、`issue.archived`、`server_busy` | H `TestCreateIssueLinkHTTPRefusals`；B `TestCreateIssueLinkAPI`、`TestPermissionMatrix/createIssueLink`、`TestIssueLinkHiddenAnswers`、`TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy/a_link` |
| `updateIssueLink`；PATCH `/api/v0/issue-links/{link_id}` | 200；省略 title 保持、null 清除、空串保留；URL 变化清 `{}`；标题独改保留数据库当前 metadata；允许跨作者 | `validation_failed`、`issue.link_not_found`、`forbidden`、`issue.link_exists`、`project.archived`、`issue.archived`、`server_busy` | H `TestUpdateIssueLinkHTTPRefusals`；B `TestUpdateIssueLinkAPI`、`TestPermissionMatrix/updateIssueLink`、`TestIssueLinkHiddenAnswers`、同 busy 测试 `a_link_edit` |
| `deleteIssueLink`；DELETE 同单行 | 204 空 body；soft delete、同刻同写者；再次删除404；释放活行 URL 排重 | `issue.link_not_found`、`forbidden`、`project.archived`、`issue.archived`、`server_busy` | H `TestUpdateIssueLinkHTTPRefusals`；B `TestUpdateIssueLinkEventsAndDeleteIssueLink`、`TestPermissionMatrix/deleteIssueLink`、`TestIssueLinkHiddenAnswers`、同 busy 测试 `a_link_deletion` |

| 共通或边界 | 真实断言 / 观察边界 | 层 / 测试 |
|---|---|---|
| 身份401、坏UUID400、body结构400 | 四op无token/坏UUID；POST缺url、null/错误类型、metadata/creator未知字段；PATCH null url/类型/未知字段；用例 calls=0。DELETE没有body schema，不外推PATCH结构规则 | H `TestCreateIssueLinkHTTPBinding`、`TestUpdateIssueLinkHTTPBinding` |
| 422字段语义 | URL `not_allowed`，title `too_long/invalid_format`，多字段同报；2048/2049 URL 与255/256 title按Unicode字符；scheme/path/query原文、title三态。域字段表不是逐个HTTP字段码运行证明 | D `TestLinkURLNormalization`、`TestLinkTitlePatch`；A `TestUpdateIssueLinkValidationBeforeResources`；B 两个 API 测试；H 拒绝表实际回答 `validation_failed` |
| 三写繁忙503 | code=`server_busy`、`Retry-After: 1`；饱和时隐藏404、403和合法域422仍优先，不把资源拒绝改成繁忙 | B `TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy` 内refusal/invalid子例；H两拒绝表验证header |
| 角色和归档交叉 | 四op普通/归档；三写both archived/archived issue；普通guest own读200写403、他人隐藏404；all-features读他人仍写403；admin兼guest按可见性/管理员写；跨作者可改删 | B `permission_matrix_links_test.go` 的14行/140格；`TestPermissionMatrix`含prepare，`permission_matrix_test.go`逐格status+code；`TestIssueLinkHiddenAnswers` |
| 原文排重与数据库错误 | 活行精确相等、软删复用、独立hash碰撞原文并存、长高熵原文；仅23P01+指定constraint映射link_exists，其他数据库错保链 | P `TestLinkStore`、`TestLinkConflictMapping`；迁移/存储测试及变异矩阵的原始证据；不把物理constraint改成短索引 |

## 三活动：四槽逐值

槽顺序固定为 `(old_value,new_value,old_identifier,new_identifier)`；`∅` 表示 nil，`""` 为非nil空串。Activity 的 workspace/project/issue/actor/at 与业务写一致；comment按存储比较，HTTP不增加该字段。

| 活动 / 触发 | 四槽 | comment / 例外 | 层 / 测试 |
|---|---|---|---|
| created / 新link | `(∅,url,∅,linkID)` | `created a link` | D `link_activity.go`；A `TestCreateIssueLink` 三种title深比较完整Activity与审计；B `TestCreateIssueLinkEventComposition` SQL四槽、comment、同刻 |
| updated / 实际URL变化 | `(oldURL,newURL,linkID,linkID)` | `updated a link`；title-only不写URL活动 | A `TestUpdateIssueLink/URL_title_omitted,URL_and_title`；B `TestUpdateIssueLinkEventsAndDeleteIssueLink` SQL全槽 |
| deleted / 显式删除link | `(oldURL,"",∅,∅)` | `deleted the link`；新值不能是nil，两个ID必须nil | A `TestDeleteIssueLink` 全结构深比较；B 同上 SQL空串/两个nil、soft delete审计 |
| title-only / noop / refusal / worker / cascade | 无额外link活动 | title-only仍审计+updated事件；noop/refusal不推进审计；worker只metadata；cascade仅已有issue/project事件 | A `TestUpdateIssueLinkNoopAndArchiveOrder`；B `TestUpdateIssueLinkEventsAndDeleteIssueLink`、`TestIssueEventRefusalsAreUnobservable`、T13三交错测试；E2E I10/I17与目录故事 |

## 三事件：Change四槽与Envelope

每个Change槽为 `(Old,New,OldID,NewID)`；**所有link Change的两个ID槽均nil**。Entity=`{Kind:"issue_link",ID:linkID}`；Event.ID独立非零且不同于Entity.ID，workspace/project/actor/time来自当前锁下写，不能把活动的ID槽复制进Change。

| 事件 / 条件 | Changes（固定url→title，只包含变化字段） | nil/空串/值的实际覆盖 | 层 / 测试 |
|---|---|---|---|
| `issue.link_created` | url `(∅,url,∅,∅)`；title `(∅,title,∅,∅)`，两项恒在 | 创建title nil、`""`、值三种；url原文及顺序 | A `TestCreateIssueLink/null_title,empty_title,title_value` 与 `wantCollaborationEvent` 完整Envelope/Changes深比较；B `TestCreateIssueLinkEventComposition` |
| `issue.link_updated` / URL变化 | url `(oldURL,newURL,∅,∅)`；title仅实际变化时追加 | URL+title值、URL且title省略（无title Change）；两项顺序不变 | A `TestUpdateIssueLink/URL_title_omitted,URL_and_title`；B `TestUpdateIssueLinkEventsAndDeleteIssueLink` |
| 同事件 / title-only | title `(oldTitle,newTitle,∅,∅)`；没有url Change | 值→nil、值→空串；B覆盖title更名/清除；不增加URL活动或crawl job | A `TestUpdateIssueLink/title_null,title_empty`；B 同上、`TestIssueLinkSubscriberRegistration`；E2E I10 |
| `issue.link_deleted` | url `(oldURL,∅,∅,∅)`；title `(oldTitle,∅,∅,∅)`，两项恒在 | A旧title为值，B旧title为nil；完整Envelope与两ID槽均深比较 | A `TestDeleteIssueLink`；B `TestUpdateIssueLinkEventsAndDeleteIssueLink` |
| 目录和原子性 | 独立15类准确序列/无重复/切片不可变；未知未来类型拒绝 | 目录dispatcher无subscriber，只证类型；真实subscriber另测事务payload | B `TestIssueEventDirectory`、`TestLinkJobCommitRollback`；A `collaboration_events_test.go` / B `issue_comment_events_test.go`的Envelope helper |

## metadata、订阅与worker

| 性质 | 准确结果 / 例外 | 层 / 测试 |
|---|---|---|
| pending | 非nil空map `{}`；创建和URL改变即返回/落库；无title/favicon/error键 | B `TestCreateIssueLinkAPI`、`TestIssueLinkRuntimeJob`；E2E I10受控首响应前API和全行DB |
| success | 恰四键 `{title,favicon,url,favicon_url}`；title/icon可nil；url为原任务URL，favicon_url为成功图标最终URL；不带error | C `TestCrawlerMetadata`、`fixture_test.go` map深比较；B `TestIssueLinkRuntimeJob` DB/HTTP完整map；E2E I10四键和内嵌PNG严格等值 |
| 页面blocked / failed | 恰两键 `{error:"blocked"或"failed",url}`；网络失败写结果后任务完成；日志细分不公开 | C `TestCrawlerLimitsAndFailures`、`TestCrawlerLogsRedacted`；A `TestCrawlLink/blocked_result_completes,failed_result_completes`；E2E blocked只看结果，不声称0连接 |
| 图标失败 | 页面仍为四键success；favicon/favicon_url均nil；仅一次候选，SVG/64KiB+1拒绝，页面1MiB限额，共享5秒总预算 | C `TestCrawlerLimitsAndFailures` 的body_boundaries/icon_mime/page_failures/blocked_before_socket/one_icon_attempt/deadline子例；production均经outbound |
| JSON边界与schema | 缺键/null/string可区分；其他JSON类型拒绝；schema封闭五个允许键，但没有把三种shape设为互斥oneOf | D `TestLinkMetadata`；`api/modules/issue.yaml` IssueLinkMetadata；H nullable五键夹具只证序列化，不冒充生产success shape |
| created订阅 | `crawl_on_link_change`仅created/updated；created一个url.New→一次 `{link_id:Entity.ID,url}` | A `TestCrawlOnLinkChange/created`；B `TestIssueLinkSubscriberRegistration`、`TestLinkJobCommitRollback`；真实app生产bridge |
| updated / 不订阅 | 仅url Change投一次；title-only/empty updated不投；deleted根本不订阅；A→B→A三个不同任务，不去重 | A同测试的updated/title_only/updated_empty及Types精确值；B `TestIssueLinkSubscriberRegistration`；T13 `A_to_B_to_A_accepts_old_A_as_approved` |
| malformed / 投递失败 | created缺url、created/updated重复url或New=nil均error；Enqueue错原链返回，使原事务回滚；不读link、不抓网络、不取锁 | A同测试所有invalid及enqueue_error；B `TestLinkJobCommitRollback` 坏payload、取消InsertTx、第二subscriber失败 |
| 真实事务 / 注册 | 原ctx/Tx传InsertTx，未提交独立连接不可见；commit业务+活动+job，rollback全无；default队列、3attempt、无周期crawl；目录独立预期crawl kind | B `TestLinkJobCommitRollback`、`TestNewAppCrawlJobCatalog`、`TestNewAppCrawlJobRuns`；I `module_events.go/module_jobs.go`；真实B/app.go组合 |
| metadata SQL guard | 单句id+精确URL+未删除；0行成功；不写审计/活动/事件，不取项目锁；DB/context错保链供重试 | P `TestLinkMetadataGuard`；A `TestCrawlLink`；River `TestCrawlLinkWorker`；SQL及生成物均对照原SHA |
| 真实并发 | A暂停→B提交→旧A0写→B成功与反序；标题锁下读后crawler先提交，UPDATE RETURNING保留新metadata与反序；删后旧任务/同URL新ID不改；完整非metadata审计/活动不变 | B `TestLinkCrawlURLInterleaving`、`TestLinkTitlePreservesConcurrentMetadata`、`TestLinkCrawlAfterDeletion`；T13最终变异及 `-race -count=5` 历史15顶层/50RUN、每名5次，无40P01 |
| 批准限制 | A→B→A时旧A可满足id+url，最后完成者覆盖同URL结果；B不能写入A。不宣称generation严格新鲜 | T13独立A→B→A子例按精确三个job IDs等待；spec§2.7明确接受 |
| 运行角色 / 注册拒绝 | 实际非owner非superuser无bypassRLS角色CRUD+job；请求快照保留所有普通表完整行，另按id记录请求job id/kind/args；periodic/attempt噪声排除，新增/同ID替换可见 | B `TestIssueLinkRuntimeJob`、`TestRegistrationRowsSeeRequestJobs`、既有拒绝注册；runtime_scratch四条SQL精确42501；T12 M32/M33仅测试守卫反例 |

## 无副作用、锁序与故事

| 场景 | 实际观察与必要例外 | 层 / 测试 |
|---|---|---|
| 预检拒绝 | 域校验先于资源；隐藏404先于403；拒绝不开gate/tx、单连接不被持有；完整业务快照、事件/probe job、clock不变 | A links/link_writes拒绝；B `TestIssueLinkHiddenAnswers`、`TestIssueLinkWritesLockAndRecheck/precheck_uses_no_write_resources`、`TestIssueEventRefusalsAreUnobservable` |
| 锁下重判与顺序 | project先于link，项目N锁；重读成员/OwnOnly/删除移动后重新授权；归档项目→归档issue→noop→唯一→锁后一次Clock→issue→link→activity→event | B `TestIssueLinkWritesLockAndRecheck`、该测试duplicate/*与clock/*子例；A精确calls与归档/noop表；拒绝和noop不写、不读clock、不查唯一 |
| 中途故障 | 三写×MarkChanged/link SQL/InsertActivities/Publish故障：完整业务/活动快照回滚；生产crawl job以T11真实事务证明，不拿早期probe充数 | B `TestIssueLinkWriteRollback`、`TestCreateIssueLinkPublicationRollback`、`TestUpdateIssueLinkPublicationRollback`、`TestLinkJobCommitRollback` |
| I10全链 | typed API种子、无协议URL、受控pending→标题/icon/dataURL、精确args唯一job completed、title不增任务、换URL重抓、409无变、204活动链；只本机fixture | `e2e/stories/issue/i10-links.spec.ts` 两case；T14 `stories-verified`为7/7 Chromium组合历史；M30故事红201→500恢复1/1，不重复计新变异 |
| I17 / I15 / 项目与工作区 | 两层子树三个活link、不同creator/deleter；已归档末级、先删/邻居全行保护；API furnish含links；目录没有绕过issue_links | I17 delete/catalog、I15 archive、project P4、workspace W2/W3 API/page；T14主组合7/7、cascade7/7历史；最终完整E2E另列 |

最终源码接线核查：B/app.go构造 `newOutbound`、真实 `issue.Subscribers(issueInserter)` 和模块Jobs；I/module.go构造真实crawler与CrawlLink；I/module_jobs.go声明worker和EnqueuedKinds；registrationRows消费请求kind；所有删除目录与furnish均含links。定向检索生产源码无临时 `emptySubscriber` / `fakecrawler` / `fakeCrawler`，本Task另修复既有锁序测试的异步fixture接缝，产品源码保持不变。

## 最终门禁记录

首轮clean `958f2e95070fa6db32f684eb98a7ea9fbb9fb588` 全仓Go实际1,744顶层/8,700总RUN，exit2，唯一失败为 `TestEachIssueWriteLocksItsProjectFirst/addSubIssues`：观察器选到`issue_links`而非`projects`；后六门禁未启动。原始输出保留 `E/checkpoint-t15-local`，不是绿，也不覆盖。真实本机受控HTTP诊断同时看到请求PID92执行LockIssueProject等待projects、worker PID93执行SetLinkMetadata等待issue_links；holder94下旧观察器先选后者。精确link/URL完成同步后同场景绿，请求实际200且原worker completed。诊断首次把SQL名误写LockProject的失败和随后正确的确定性红均保留，不把误期望当性质红。

`issue_write_locks_test.go`保持原锁序、强度、无写与无workspace锁断言，只等待所有新seed及每次成功create201的精确生产crawl完成，并在每个全表锁窗口前断言pending crawl=0、auto-archive候选=0。当前矩阵没有URL更换，标题独改/删除不投任务；唯一issue异步写源是上述crawl；RunOnStart auto-archive仍运行但fixture没有archive_in=1..12的候选，identity cleanup不写issueTables。观察器仍只认holder，前提明确为窗口中只有目标请求写，未按SQL文本过滤、停worker或sleep。未来fixture新增URL改动/可归档项目必须同步其真实任务，否则前提断言失败。

本次另对真实生产LockIssueProject查询删除FOR NO KEY UPDATE、生成后，原矩阵最窄createIssueLink选择实际红（先等issues、project无N锁）；query和生成物按原SHA逐字节恢复，同选择绿，说明fixture同步没有遮蔽真实请求锁序回归。最终测试源码完整矩阵1顶层/19RUN绿，随后准确新clean提交的七门禁在独立 `E/checkpoint-t15-local-retry1` 全部执行成功。历史 T12 本地 `fdeadacf6d39340733d7e42f615229da3fdb6456` 七门禁通过，不作为本次结果。T12远端CI `40373692243`浏览器安装耗时16分36秒后成功、Build成功、E2E已开始，随后整job20分钟上限取消；不能写成远端全绿或E2E未开始。最终push/CI归主控T16处理。


### 本次最终七门禁（全部exit0）

执行时间2026-10-27 03:04:35–03:11:32（Asia/Shanghai），所有stage开始/结束HEAD、源码SHA与clean状态均一致。Go用 `GOFLAGS='-p=1 -v'`，其余 `GOFLAGS=-p=1`；cwd为本worktree。测试与容器串行，命令无 `-run/-skip/-short`。

| 门禁 / 实际命令 | 实际结果 | raw stem / 秒 |
|---|---|---|
| `make test` | server+tools 51通过包、16个无测试包；1,751顶层/8,725总RUN，含3明确SKIP，0cache；bootstrap376/3934，issue完整HTTP包60/143（11.800秒），公开码双向守卫实际启用 | retry1/test；333.569 |
| `make lint-go BIN_DIR=/Users/xiaoruan/project/nerve-project/bin` | server与tools两段各0 issues | retry1/lint-go；11.049 |
| `make gen-check` | Go/sqlc、契约bundle、TS类型全部重新生成，git无差异；生成物已提交 | retry1/gen-check；3.695 |
| `make lint-web` | 54/54任务，36cache；类型/lint/format与既有关键词等守卫通过；缓存数不当作测试数量 | retry1/lint-web；4.074 |
| `make knip` | typegen后knip，配置hint也按error，0退出 | retry1/knip；2.100 |
| `make test-web 'TURBO_QUIET=--output-logs=full --force'` | 138文件/1,165 Vitest +5 Node=1,170；18/18任务、0cache；四条既有React key提示 | retry1/test-web；15.355 |
| `make e2e VERSION=0.0.0-local.p7.t15` | build 11/11任务（11cache），Go二进制真实重建；9 workers，153case/153attempt：Chromium151+Firefox2；0失败/flaky/skip，HTML机器报告同值 | retry1/e2e；46.784 |

`retry1`完整路径为 `E/checkpoint-t15-local-retry1`；每stem都有原始`.stdout/.stderr`、精确命令/cwd/退出码/时间/HEAD/tree/源码稳定性与日志SHA的`.json`，`execution.log`末尾为ALL_LOCAL_GATES_PASSED。`source-manifest.json`关联准确源码；`E/task-15-evidence/gate-summary.json`、`package-run-counts.json`保留实际包/测试/browser统计，矩阵引用的49个顶层测试均在本次RUN输出实际出现。

- 三个Go skip如实保留：`TestIssueListScale`要求NERVE_LIST_SCALE=1；`TestTheDeletionsAtTheScaleTarget`要求NERVE_MEASURE_SCALE；`TestFlatIssueListOrders/sub_issues_count`的huge count numeric probe跳过，bounded counts另有测试。未声称规模性能验收。
- Go日志本轮三次Ryuk `Shell not found in container`，随后各ready且套件继续成功；Web四条React key提示保留，不改产品/Docker降噪。生成器stderr是正常bundle/类型输出，Turbo版本提示保留。T13日志记录Minor已关闭，T8/T11两条selector文档Minor仍归T16。
- 旧E2E在开跑前归档79文件/1,440,353字节，新E2E归档79文件/1,453,856字节，分别为 `E/task-15-evidence/e2e-before` / `e2e-after`；每文件复制前后SHA一致，各有manifest。新HTML中机器报告另提取为 `playwright-report.json`，统计153且无errors。旧归档不倒推运行HEAD。
- 最后提交仅补实测文档，**没有在最终文档HEAD另跑完整套件**。`tested-nondoc-manifest.json`记录tested HEAD全部3,639个非docs文件的SHA，最终逐项证明源码、测试、生成物与tested HEAD字节一致；相对Task BASE唯一非docs差异是上述已有锁测试。最终文档HEAD及证明见Task15报告，不用相同源码冒称新的执行。
