# M4/P7 链接、出站与抓取评审记录

日期：2026-10-27。T1–T15实施及独立规格/质量评审完成；原范围单次deep-review完成，T16单次修复与正式文档已写。本轮准确提交完整本地七门禁已通过；独立fix复审已确认原8项全部 ADDRESSED；T7历史时态文案已纠正，本次纯文档增量确认及最终交付HEAD CI见主控最终记录，未合并main。分支`codex/m4-p7`，基线`5005be985e7ec78727edc6f8fc5c05999b45cc65`。配套：[spec](../specs/P7-issue-links.md)、[plan](../plans/P7-issue-links.md)、[整体审核](../../../reviews/re-05-M4P7整体审核.md)。

## 1. 评审方式

准备期按02§3紧凑spec/plan做独立安全/集成预检，5项Medium修订复审闭环；四列PG/sqlc实验不是完整原型。实施按逐Task真实红绿、生产变异、原字节恢复与独立双结论评审推进，局部修复复审。负责人后来要求有产品变化的检查点push前完整本地门禁，T4/T8/T12/T15已按此执行。

单次整分支deep-review固定`5005be985e7ec78727edc6f8fc5c05999b45cc65..d2a3cf26df220f95ac8f656d9ff8f4eaa2997305`，22 commits/165文件/457chunks；core72/120、network_jobs55/81、contracts_e2e24/144、docs_guards14/112全文覆盖，无重无漏，源字节对Git核验。三lens覆盖逻辑、集成、测试，含全部18项issue请求写及后台写交会锁图。作者只执行本次统一修复，不再派reviewer；独立定向复审由主控后续完成，不冒称第二次整支审查。

## 2. 验收标准逐条核对

`E=.superpowers/sdd/P7-issue-links`为ignored原始证据根；每Task报告/独评记录保存当时BASE、实际命令、源码SHA、RUN/exit及恢复记录。下列为历史交付锚，不冒称变异未提交字节的运行HEAD。四API/三活动/三事件/metadata/subscriber/guard逐值证明、全部命令与层见[行为矩阵](P7-issue-links-behavior-matrix.md)；七类21项真实生产补丁及排除项见[变异矩阵](P7-issue-links-mutation-matrix.md)。

| Task | 交付锚 | 实际交付与证据 |
|---|---|---|
| T1 | `0537a2d9e289cb12ee75fc3059447eeafff8ef05` | 12列/长URL/碰撞/grants/两条删除；`E/task-1-report.md`、对应review/evidence |
| T2 | `5b171386a2117ea8bbbe199ab5acf494e4dd2a4c` | 纯域、标题三态、原文存储及RETURNING；`E/task-2-report.md`、对应review/evidence |
| T3 | `9da12bc1b0881de6c3e04eabff3fc4fbafa88ac9` | 创建/列表/三事件目录/首次码和API种子；`E/task-3-report.md`、对应review/evidence |
| T4 | `4bc28b2b73272a1f1b88932a4e62140ad0be4454` | 改删/条件metadata/活动事件/拒绝回滚；`E/task-4-report.md`、对应review/evidence |
| T5 | `8f05b68aa8b3d564ea5b74e682cb0beb26d45f5d` | 权限矩阵、锁下重判、原子回滚/race；`E/task-5-report.md`、对应review/evidence |
| T6 | `9fc903225b77209081c79925895d4a5cc6483115` | 独立IANA完整性/端点及嵌入地址；`E/task-6-report.md`、对应review/evidence |
| T7 | `128de7b620460d0161776b8dcc633f637cb746d5` | 每跳全候选/固定IP/IDNA/TLS/期限/限额；`E/task-7-report.md`、对应review/evidence |
| T8 | `8a1a6acce00d93846257067cb4902b2de2b7d04a` | test-only配置与CLI隔离；`E/task-8-report.md`、对应review/evidence |
| T9 | `15e2f251d6993491c71f5f0bc698a1b8e5c4beb0` | 标题/图标/共享预算/受控日志；`E/task-9-report.md`、对应review/evidence |
| T10 | `9cb594faa3165b749f3d54637be0ca91d7c1b9d9` | guard SQL/worker/server组合；`E/task-10-report.md`、对应review/evidence |
| T11 | `f2a6569b685c03541da24ec22b51b8a964d269b5` | 首subscriber/原事务InsertTx/完整快照同步；`E/task-11-report.md`、对应review/evidence |
| T12 | `fdeadacf6d39340733d7e42f615229da3fdb6456` | 请求任务目录/受限运行角色/真实completed；`E/task-12-report.md`、对应review/evidence |
| T13 | `a08777740f267ffcef45d0ee93c71a70de2d1b4c` | URL/title/delete真实worker交错/race；`E/task-13-report.md`、对应review/evidence |
| T14 | `7b44d2ef7db03ab922b7a289fabdfe509951bb7d` | I10 API和带links删除目录；`E/task-14-report.md`、对应review/evidence |
| T15 | `d2a3cf26df220f95ac8f656d9ff8f4eaa2997305` | 两矩阵/21历史生产变异/锁观察器修复/七门禁；`E/task-15-report.md`、对应review/evidence |
| T16 | 修复及tested `cd0fce7ed4f18739a60fcf3d0c44f811b809b7b8`；docs-only回填提交见Task16报告 | charset/nullable enum/422 fixture、8项发现、正式文档；原8项独立fix复审已关闭；本次文档增量确认/最终交付HEAD CI见主控记录 |

T4额外三个修复提交`ed205cad`/`a2211c8f`/`be3b8d4e`分别补活动回滚、完整迁移/历史FS边界及I15 link种子，不追记为T1/T3最初已覆盖。T7初稿`2a4a2196`后IDNA修复为表中`128de7b6`，只使用修复后的网络变异。T15初稿`958f2e95`完整Go失败后修正观察器fixture，tested为`6e28cc16`，最终`d2a3cf26`仅两矩阵文档、3639非docs字节相同；不是在d2另跑全套。

T15准确本地七门禁：Go1751顶层/8725RUN（3已披露skip）、强制Web1170、E2E153=151Chromium+2Firefox；lint-go/gen-check/lint-web/knip全绿。T15准确d2 [CI40387396589](https://github.com/open-nerve/NerveProject/actions/runs/40387396589)由主控核公开原件全绿：server9m11、web3m45、e2e7m03，`E/checkpoint-t15/verified-success.json`保存摘要。T12的40373692243在浏览器安装16m36成功、Build成功、E2E已开始后触20分钟job上限取消，历史保留，不算全绿。

T16局部真实红：charset原始非UTF8标题及href损坏；schema原任意error被错误接受。修复后charset25场景、schema省略/null/blocked/failed与3非法值、完整crawler/apitest/HTTP/archtest回归有raw。首次含null的enum被既有authoring guard拒绝，改为独立enum组件+单字段nullable oneOf/ref，生成物无`<nil>`常量；guard未弱化。C1按静态调度推导，未概率复现或伪造红；完整快照前只等待got的唯一crawl。本轮 tested `cd0fce7ed4f18739a60fcf3d0c44f811b809b7b8`（tree=`25f1520301bd2471ad4892d859b97256faff884c`）七门禁全部 exit0：Go 1753 顶层/8759 RUN、3 skip，强制 Web 1170 测试，E2E 153=151 Chromium+2 Firefox。 原始门禁在 `E/checkpoint-t16-local`，汇总 `E/task-16-evidence/gate-summary.json`；不复用T15SHA当新执行。最终仅docs回填，3643个tracked非docs路径（含README及symlink文本）与tested字节一致，提交后另跑gen-check。

### 2.1 准备期长URL实验（历史）


原设计同时允许 2048 Unicode 字符、要求 `(issue_id,url)` 原文 B-tree 部分唯一。在本实验 PostgreSQL 18.6 / 8192 字节页上，2048 字符、8135 UTF-8 字节的不可压缩合法 URL 被拒绝：SQLSTATE `54000`，索引行 8168 字节超过 2704 字节上限。这个结果与 [PostgreSQL 的 B-tree 容量约束](https://www.postgresql.org/docs/18/btree.html) 一致。

替代方案是单列表达式 hash exclusion：`EXCLUDE USING hash ((issue_id::text || ':' || url) WITH =) WHERE (deleted_at IS NULL)`。UUID 规范文本长度固定，组合表达式无歧义；排斥比较仍对原文做等号重检，哈希相同的不同 URL 可以共存。它不是“摘要唯一”，也不是 PostgreSQL 不支持的 hash unique index。官方依据：[hash 索引](https://www.postgresql.org/docs/18/hash-index.html)、[EXCLUDE 约束](https://www.postgresql.org/docs/18/sql-createtable.html)。

真实数据库结果：同一长 URL 成功保存且原文不变；同 issue 的重复 INSERT/UPDATE 报指定约束 `23P01`；不同 issue、不同 URL、软删后重建均成功；2049 字符由 CHECK 拒绝；真实 hash 碰撞的两个不同原文同时存在。完整代码、命令、原始输出与 SHA 在 `index-probe/`；固定 issue UUID `0199a2b4-0000-7000-8000-000000000001` 的可复验碰撞对为 `http://collision.example/271150` 与 `http://collision.example/83166`，完整表达式 hash 均为 `-1669361772`，补验证据单列 `index-probe-v2/`，未覆盖前轮。实验自建的 PostgreSQL/Ryuk 容器均已自动清理，没有操作既有容器。

实验不覆盖正式 API、权限/grants、业务事务或生产并发。T1/T3/T4/T5/T13 随后已将长 URL、精确错误映射、碰撞对照和锁串行验证变为正式产品测试，详见T1–T5/T13证据。


## 3. 局部决定

完整发生顺序与每条理由/若错代价保存在[整体审核附录](../../../reviews/re-05-M4P7整体审核.md#附录裁定台账)。重点：hash exclusion保2048原文/12列；标题SQL保留库内metadata；标准库Transport规范IDNA；A→B→A沿已批准id+URL守卫；fixture先完成真实异步writer再取完整快照。旧“全套留T15”已被负责人提前检查点要求取代，不能当当前规则。

T16解码只在adapter，复用既有charset及tokenizer/Lookup；全ASCII前缀超过1024后UTF8保留，显式BOM/header/meta优先。公开error枚举收紧在API边界，DB仍五键string/null。历史变异审计冻结T15 source ref，当前crawler改变另列差异，不能改历史恢复SHA。理由与风险见整体审核，不引入generation、新锁、依赖或共享协议。

## 4. 发现与处理

### 准备期预检（历史）

首审累计 High 0、Medium 5、Low 0，全部是既定范围内的局部修订，未要求放宽守卫或改变模块协议。

| 编号 | 问题、修订及验证归属 | 当前结论 |
|---|---|---|
| C1 / Medium | 访客可见性必须依 Grant.OwnOnly；补 guest_view_all_features=true/false、管理员兼访客与普通访客的差别，T3/T5 按真实判定覆盖，并修正交接 03 的条件省略。 | v2 修订，主控核对闭合。 |
| S1 / Medium | title-only updated 无 url Change 是合法不投递，不能按“缺 url”返回错误；created 缺 url、重复 url、nil New 仍是错误，T11 补真实提交/审计/无 job。 | v2 独立复审闭合。 |
| S2 / Medium | 仅检查公开 metadata 无法证明日志脱敏；T9 捕获完整 slog 输出，用不同 query/userinfo/body 哨兵、阶段/分类非空及原始 URL/error 日志生产变异证明。 | v3 独立复审闭合。 |
| I1 / Medium | 2048 字符与原文 B-tree 容量冲突；采纳 §2.1/§3 的精确 hash exclusion，T3 只映射约束名+23P01，补长 URL、软删、碰撞和 API 边界。 | v4 独立定向复审闭合。 |
| I2 / Medium | 既有 TestIssueEventDirectory 固定 12 类、把 link 视 future、向生产订阅者发布空 Event；T3 改独立 15 类目录，T11 分离真实 payload/事务订阅测试。 | v4 独立定向复审闭合。 |

安全分区 v3、集成分区 v4 均已 Approved；未关闭 High/Medium/Low 均为 0，预检通过。所有文件清单/冻结稿哈希与定向结论均保留，不将作者自查代替独评。


### 本次整审与历史Minor收敛

5新Minor：NJ-1非UTF8页面解码、C-E2E-01公开error枚举、core C1完整快照前crawl同步、D1写顺序、D2资源/请求/队列上限表述；另3旧Minor：T8/T11 selector、T15 M35–37恢复argv。全部已在单波实施局部修复，证据与逐项状态见[整体审核](../../../reviews/re-05-M4P7整体审核.md)。独立定向复审准确覆盖`d2a3cf26..7e903ecb`的28路径/36块，原8项全部 ADDRESSED，未发现新增产品Critical/Important/Minor。仅T16-DOC-1指出plan的T7两段历史时态遗漏，作者已补原2026-10-26节点及后来IDNA独立复审通过事实，保留原测试/变异数量；本次纯文档增量的独立确认见主控最终记录。

## 5. 最终实现与plan的差异

四CRUD按职责拆create/list/update/delete_issue_link.go；原事务桥复用bootstrap/issue_inserter.go；registrationRows实际在invitations_test.go，新request_jobs_test.go与issue_link_runtime_test.go复用现有世界/等待，无第二套框架。newOutbound消费config.Config，UpdateLink返回domain.Link，Crawl返回domain.LinkMetadata，SetLinkMetadata返回(bool,error)均与实际接口一致。

T8原selector没匹配configs，T11原selector漏LinkCrawlsEnqueue；推荐命令已补，历史0匹配不计覆盖，原完整回归足够，无为文本重跑。T15脚本red/green统一argv数组，M35–37恢复同命令1/2RUN；新派生索引保留旧raw和T15冻结源，与T16当前兼容分开。T16新增charset.go/charset_test.go、schema行为测试、nullable enum生成/HTTP类型转换及C1等待，未改DB CHECK、域/app或网络策略。

## 6. 移交事项

P6首生产订阅/3link事件/请求任务、M2§3审计与§6worker/grants、M3§1请求快照均追加接收回执；P6当时12事件/空订阅及集成事实不改写。M4§3.11/4.8/3.23/14/15、v0§6.2/6.7/8、plane-diff12列/约束/行为和README出站责任同步，02/03更新完整本地CI先于检查点与Firefox事实，04保留历史快照另加当前接续点。

P8继续link_count/草稿/搜索；P9继续异步删除/保留/清理；P11/P14继续详情数据层、I10/I17页面及可达性/卸载清理；P15协作页面；M8复用outbound并负责IANA维护和部署网络策略。M2/M3 status仍open，M4总体§14/v0§9.4仍进行中，不创建M5–M8最终closeout。产品检查点由主控推送，作者未push/merge；本期未合并main，集成另获授权。

## 7. 已知限制与后续门

启动时接口快照不能覆盖NAT/LB自公网或后来网卡变化；普通公网任意端口仍可请求，网络层出站规则负责这些风险。每crawl最多两个资源、每资源5跳，共享5秒；固定并发/入口额度不是队列长度硬上限。HTML解析没有硬CPU deadline；集合不分页，无每link速率桶；A→B→A同URL旧结果可晚覆盖。没有以worker错误原链/default3配置声称实测DB故障三次重试，没有将I10 API故事声称页面交付。

准确原范围审查完成、修复作者自审、本轮本地门禁、独立定向复审、最终分支HEAD CI和授权main集成是不同状态。原8项独立关闭已经发生；本次纯文档增量确认和最终交付HEAD CI由主控最后记录，不提前勾main集成。保留旧INFRA、T15初次观察器误归因、Ryuk/React提示及三skip原始口径，不称输出pristine，不以缓存数替代测试执行数。

### 2026-10-27 同次定向复审回执

`E/task-16-fix-review.md`与`task-16-fix-review-coverage.md`记录准确`d2a3cf26..7e903ecb`的独立结论；主控已全文核阅。原8项全部ADDRESSED；T16-DOC-1仅历史时态文案，未打开新产品波次。此次`7e903ecb`之后只改docs，既有报告/coverage/raw不覆盖，前次作者报告与证据先封存在`E/task-16-evidence/receipt/prior`，新回执另存；完整门禁仍对应tested cd0，未因纯文档重跑产品套件。

产品检查点准确`cd0fce7ed4f18739a60fcf3d0c44f811b809b7b8`的[CI40397086823](https://github.com/open-nerve/NerveProject/actions/runs/40397086823)已由主控核全绿：server7m14、web3m46、e2e7m25；公开原件`E/checkpoint-t16/run-public.json`记录head_verified=true/Success。E2E job API另核相同head、浏览器安装28秒成功、Build成功、E2E于2026-10-26 20:45:23–20:50:42 UTC成功。`run-public.html`、`watch-history.jsonl`、`e2e-job-public.json`原件保留，receipt/ci-verified.json记录其SHA；作者没有重跑远端测试或推分支。这个结果属于已测产品提交cd0，不能冒称后续纯文档交付HEAD的CI成功；最终准确链接由主控最后返回。
