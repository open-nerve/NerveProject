# M5 / P2 资产用例与生命周期实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> 状态：预检通过，见[re-34](../../../reviews/re-34-M5P2预检.md)，T1–T4已完成，接续T5。负责人已连续授权 M5 常规推进与集成；新的重大风险/分歧先提交再上报。

**Goal：** 在 P1 local 实物合同上提供10种用途的私有准备件、六个业务操作和不丢恢复记录的删除/清理。

**Architecture：** asset 拥有 file_assets/对象状态与小存储端口；owner 持有权限和父事实；bootstrap 唯一接线。外部 I/O 在事务/连接/门之外；即时 NO ACTION 与精确受保护父批次协议兜底，不关闭外键、不放宽守卫。

**Tech Stack：** 既有 Go1.27/PostgreSQL/sqlc/River0.47/local；新增仅官方 golang.org/x/image v0.47.0 WebP 及必要间接依赖。

**Spec：** [P2-assets](../specs/P2-assets.md)及其[P2-access](../specs/P2-access.md)、[P2-recovery](../specs/P2-recovery.md)、[P2-protected-purge](../specs/P2-protected-purge.md)，四份一起阅读。

## Global Constraints

- worktree：`/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m5-p2`，分支 codex/m5-p2；基线85dc4087，main CI41471671630已全绿。主目录永远main。
- 文档中文、提交详细中文并保留feat/fix/docs(M5/P2)前缀，系统/usr/bin/git推送。常规noff已获连续授权，不重复索权。
- plane/refer只读且不在其内跑git；不动既有容器/Docker，不stash/clean/reset--hard/force，不取凭据，不全局安装。容器套件一次一个。
- 手写文件约400行内；现有gen、集中契约/目录和已存在大文件按原例外，新增逻辑按职责拆分，不为行数把耦合隐藏。
- T1改tools/keywords.json phase=M5/P2，核到期例外；不可只延长until。P6/P7尚未到期的旧文件消费者不提前删除。
- 四份spec所有固定值通用：候选≤3、metadata≤16KiB；上传15m(10s–15m)、外部段10s(1s–10s)、lease30s+grace30s；准备件7d、墓碑严格>60d；cleanup1h/100/10s；orphan/tmp真实24h；图片头≤1MiB、边≤16384、像素≤40,000,000。
- scope/user_id/父列与权限矩阵逐值一致；P2不绑定、不加attachment_count、不发假附件动态，不修改existingM4权限谓词。
- 新错误同Task进`web/apps/web/core/lib/error-messages.ts`及en/zh-CN errors.json，运行make test-web；前端实体仅用生成类型。
- SQL仅本模块sqlc；端口由消费方声明；Tx由ctx传递，不在owner间传业务闭包；同一实例write gate注入issue和asset。
- 尽量复用issueWorld/newWorkItem/heldTx/pgtest.RowWaits、已有资源世界/存储合同、e2e/furnish/assert/catalog，不复制测试框架。前置故事数据仅API；SQL只用于断言/隔离存储层fixture。
- 每Task记录实际红/绿/变异、命令退出码、测试tree/提交和生成物SHA256/行数；全部RED命令预期非零且命中具名性质断言，全部GREEN命令预期0及所选用例PASS（不能0 tests）；没运行不写通过。各Task聚焦回归，产品检查点推送前串行完整七项，最终新变化按实际范围重验。
- T5首表就落实server删除/清理/守卫；API阶段结束的T10把e2e furnish接真实API后首次整段检查点。中间只做本地分Task提交，不将未完成的整段宣称七项全绿或推产品检查点。

## Review Focus

1. 候选外部成功但DB失败、暂停超过lease或旧PUT迟到：final不可被替换，恢复依据与真正回收同时证明（T7/T9/T11）。
2. 草稿换项目、评论作者/管理员、guest管理员、源目标正反ID：授权次序和锁序不能由同一单例数据假通过（T3/T8/T11）。
3. 首批全受阻但后面可删，以及直接/传递CASCADE、自引用：Scanned驱动游标，未知23503不吞（T4/T5）。
4. 业务快进61d而真实传输未完、调小配置、跨实例同名、坏清单项：不得误删活数据或借假时间证明真实过期（T2/T9/T10）。
5. recordOutput/snooze/进程退出、原At/By重试、阶段末重复GET：真实River恢复，内容签发零DML、父删除失败回滚全链（T5/T6/T9/T11）。

## 文件结构与接口衔接

详细位置见spec§2.1；新增文件命名按本计划。每层自己的值由bootstrap显式转换。下列共享数据指asset自身domain/app类型，不给其他模块增加asset import。

- domain.Asset：spec表字段的具名业务值；domain.NewAsset/Scope/Purpose/StorageMetadata，ContentProof{SHA256 string; Width,Height *int}。
- app.AssetReader.Get(ctx,id)(domain.Asset,bool,error)；app.AssetLocker.LockAssets(ctx,ids)([]domain.Asset,error)；app.AssetCreator.Insert(ctx,AssetWrite)(domain.Asset,error)；app.MetadataWriter.Update(ctx,MetadataWrite)(domain.Asset,error)。MetadataWrite含ID/ExpectedRevision/新metadata/可选uploaded与原始Stamp，SQL校验revision受影响行。
- app.Stamp{At time.Time; By *uuid.UUID}；app.AssetWrite{Asset domain.Asset; Stamp Stamp}；app.DeletionStamp同Stamp；app.ParentKey{Kind domain.ScopeKind; ID uuid.UUID}。用途/父链/大小等immutable字段只有后续owner绑定协议能改，P2通用metadata写不开放它们。
- app.ContentValidator.Validate(ctx,io.Reader,size int64,contentType string,imageRequired bool)(domain.ContentProof,error)。取消/超限error保留分类；完整hash，图片仅头。
- app.Access.Within(ctx,actor,[]HostRequest,[]uuid.UUID,func(context.Context,AccessHeld)error)error 是asset包内方法；AccessHeld含Hosts/Assets/最终锁后时刻At，各owner只实现Inspect/Check和小锁端口。内部回调不得导出为模块间接口。
- app.ParentDeletionReader.Deletions(ctx,[]Host)(map[ParentKey]DeletionStamp,error)；自有Rows的按scope、回收、purge分页由T5定义固定查询，不拼外部表名。
- owner app各自PurgeCursor{Table string; After uuid.UUID}、PurgeProgress{Cursor;Scanned,Deleted,Blocked int;Done bool}；Purger.Page(ctx,cutoff,batch,cursor)(PurgeProgress,error)。bootstrap消费方同形值经具名转换，不让owner import bootstrap/platform app类型。
- app.SweepCursor{Version int;Phase string;After uuid.UUID;Token string;Cutoff time.Time}；SweepResult{Cursor;Scanned,Deleted,Blocked int;Complete bool}；Cleanup.Execute(ctx,cursor)(SweepResult,error)。River adapter保存≤16KiB output；错误也带已安全提交游标。

---

## Task 1（T1）：用途、对象状态与受限图片校验

**Files：** 新建asset/domain/{asset,scope,metadata}.go及测试、asset/adapter/content/validate.go及测试、asset/app/content_ports.go；修改server/go.mod/sum、tools/keywords.json；新增schema所需业务错误等到T6同时登记契约。

**Interfaces：** 消费P1 ObjectKey/MaxObjectSize；产出本计划domain值、ContentValidator及ContentProof。domain内部非法输入先返回具名纯错误，HTTP公共码在T6映射。

- [x] Step 1：写TestAssetPurposeScopes/TestStorageMetadataBounds/TestContentValidation。用10种合法组合和跨用途/多余父字段；JSON null/版本2/重复key/别的assetID/第四候选；真实4格式、SVG伪PNG、1MiB+1头、16385边/40,000,001像素、同大小异字节、过长/少1字节与取消。
- [x] Step 2：运行 `go -C server test ./internal/modules/asset/domain ./internal/modules/asset/adapter/content -count=1`，保存真正性质红；先不存在包时的setup错误不能替代断言红。
- [x] Step 3：实现纯规则与流式validator；直接调用png/jpeg/gif及webp DecodeConfig，不用init注册解码器。锁定x/image v0.47.0，说明必要x/sys0.49.0/x/text0.43.0及后者要求的x/tools0.51.0/x/sync0.24.0，其他依赖不升级。修改phase并核关键词例外。
- [x] Step 4：同命令绿及既有asset合同；变异去候选上限、只摘要首块、图片仅信声明分别杀死；验证reader取消与有限读。validator消费io.Reader不拥有Close，T7用例在所有出口关闭ObjectReader，不能在本Task假称已证明该层资源释放。
- [x] Step 5：中文提交，记录实际依赖diff与未运行范围。

## Task 2（T2）：local有界清单与临时残留回收

**Files：** 新建asset/adapter/local/{list,temporary}.go及测试、app/cleanup_storage_ports.go；修改config/files.go/defaults与server/configs/*.yaml及配置测试。

**Interfaces：** 消费P1 Store.operation/os.Root；产出ObjectLister.List(ctx,token,limit)(ObjectPage,error)、TemporaryCleaner.SweepTemporary(ctx,before,limit,token)(next,removed,error)；files新增参数按recovery§2。

- [x] Step 1：TestLocalInventoryPages用乱序key、三页、10,000个有效最小对象；TestTemporaryCleanupIsolation用旧tmp/新tmp/未知项/符号链接/另一个root；TestFileLifecycleConfig覆盖每个边界、整数溢出、未知键和日志。
- [x] Step 2：运行 `go -C server test ./internal/modules/asset/adapter/local ./internal/platform/config -run 'Inventory|Temporary|FileLifecycle' -count=1`，记录红。
- [x] Step 3：实现ReadDir(256)+有限堆limit+1，校验token、ctx和不越根；临时目录仅生成UUID布局。对坏项明确失败且不删，不能“空清单成功”。配置默认15m/10s/1h/100/10s，含上界与遮盖。
- [x] Step 4：运行聚焦及asset/local/config全包；记录10,000目录页耗时和资源上界，取消测试有有限上界。变异取消根限制、忽略ctx、全量积累与全量枚举已拦截；业务clock作年龄由T9真实调用层拦截，T2只证明物理年龄与调用方cutoff；不能用探针声称任意规模。
- [x] Step 5：提交；本段仍无新业务API/表，既有P1传输合同通过。

## Task 3（T3）：owner权限门面与一次排序锁协议

**Files：** 新建issue/project/workspace的app/asset_access.go及测试、module_assets.go；workspace/issue自有asset查询与sqlc生成；bootstrap/asset_access.go/asset_ports.go；asset/app/{access_ports,access}.go及测试；bootstrap/app.go共享gate接线。

**Interfaces：** 精确实现access§2全部方法；owner定义自己的输入输出，bootstrap转换；ParentDeletionReader分组所需owner墓碑批量查询也在本Task，返回原At/By，不缺失猜删除。

- [x] Step 1：TestAssetHostAccessRealMatrix用每用途/角色；TestAssetAccessOrdering记录precheck→gate→tx→credential→sortedW/P/D→Check→sortedA及退出。评论管理员不能edit、能按原规则delete；guestHiddenFrom的404先于403；草稿作者私有/归档允许/删除拒绝；两个scope次序与变更复查。
- [x] Step 2：运行 `go -C server test ./internal/modules/asset/app ./internal/modules/issue/app ./internal/modules/project/app ./internal/modules/workspace/app -run 'Asset|CommentWrite|Draft' -count=1`；未实现断言红留原件。
- [x] Step 3：复用原decide/commentCheck/DraftLocks，不复制权限；添加排序单语句W(S)/D(U)，P(N)复用IssueProjects；新权限门面不取另一套gate/tx。统一bootstrap的同一write gate，所有错误原样或具名not-found转换。
- [x] Step 4：单元矩阵绿及自有postgres真实锁查询/archtest；变异绕过owner、倒序锁、多new一个gate、失去guest谓词，分别用能区分的层测。真正HTTP交错在T11补齐，不把单元记录当PG证明。
- [x] Step 5：提交并更新锁图实际符号。bootstrap最末asset业务构造依赖接续T5/T6，不后注入。

## Task 4（T4）：父清理前进与可持续任务游标

**Files：** 新建platform/postgres/protected_purge.go及测试；修改issue/project/workspace/identity的app/purge.go、adapter/postgres/purge.go/queries/purge.sql及测试；bootstrap/purge.go/purge_test.go及各module facade、相关目录测试。

**Interfaces：** ProtectedReference/ProtectedForeignKey/PurgePage和固定Candidates/Delete小操作见protected-purge；各owner产出Page而非无界Purge循环，bootstrap转成全局模块/表/After/cutoff游标，River RecordOutput+Snooze。

- [x] Step 1：真实PG TestProtectedPurgeProgress：父→issue→asset模拟、自引用、最小/最大受阻中间成功，要求Scanned3/Deleted1/Blocked2/After3；精确schema/table/constraint，未知23503/23001/40P01不吞。TestPurgeCursorResume验证一页与固定cutoff/错误进度；原M4无资产行为保持。
- [x] Step 2：串行 `go -C server test ./internal/platform/postgres ./internal/bootstrap -run 'ProtectedPurge|PurgeCursor|Purge.*Module' -count=1`，保存红，不能借更改期望掩盖skip。
- [x] Step 3：实现整批失败整体回滚再逐候选独立短事务；After来自扫描位置，已提交count随error返回。unknown cursor/过时table返回error；普通步骤保持单条自有固定SQL，以有限候选/删除CTE分别返回扫描和删除进度，DELETE再核cutoff；受保护步骤只对明确声明祖先开启。此时资产表尚未存在，生产保护集合为空，真实模拟测试覆盖算法。
- [x] Step 4：真实River进度保存/snooze一次重入测试及所有owner purge回归；变异Deleted==0停止、只看23503、倒退cutoff/丢After、Snooze不RecordOutput须失败。目录不加NO ACTION白名单。
- [x] Step 5：提交，记录每个受影响caller已迁移；正式资产祖先精确集合由T5实际catalog推导。

## Task 5（T5）：首张资产表、删除协作与已知对象回收

**Files：** 新建`server/migrations/sql/00034_asset_file_assets.sql`、migrations/asset_schema_test.go、asset/adapter/postgres/{store,rows,metadata,deletion,purge}.go与queries及gen；asset/app/{cleanup,delete_rows,purge}.go/adapter/river、asset/provide.go/module_events.go/module_jobs.go；修改owner墓碑查询、issue/app/draft_writes.go、bootstrap/assets.go/app.go/事件目录/删除目录/purge_catalog_test.go、server/sqlc.yaml、plane-diff.md。

**Interfaces：** 产出AssetReader/Locker/Creator/MetadataWriter，删除同步端口、ParentDeletionReader，asset.Provided，ProtectedParentReferences()副本、Purger.Page，DeleteScope.Execute(ctx,scope,id,Stamp,cursor)(SweepResult,error)。NewInserter适配器仅InsertTx；任务kind asset.delete_scope / asset.cleanup，现有cleanup.purge_deleted继续。

- [ ] Step 1：写TestAssetSchemaConstraints、TestAssetDeletionGraph、TestAssetPurgePreservesObjects、TestAssetParentDeletionRollback。构造真实本地对象+自有存储层行，三个层级、活/59d邻域、对象Delete失败、60d精确边界；原issue/delete/PG父CASCADE与目录guard都保留。SQL非法JSON缺字段/null、scope/check、metadata非空硬删必须失败。
- [ ] Step 2：聚焦 `go -C server test ./migrations ./internal/modules/asset/... ./internal/bootstrap -run 'AssetSchema|AssetDeletion|AssetPurge|AssetParent|PurgeCatalog|ParentCascades' -count=1`；记录红和原guard拒绝新受保护边，不能先豁免再写实现。
- [ ] Step 3：迁移/固定查询/模块所有权同时落地；SQL metadata-empty保护purge。同步issue.deleted/issue.comment_deleted及真正DeleteDraft同原事务标记；project/workspace事件只InsertTx范围任务。原At/By不后移。asset扫父事实/过期件/已知key，认领reclaiming→事务外Delete→锁下移记录，物理行严格>60d且全空。
- [ ] Step 4：按access构造顺序拆newAssets为早期storage资源与一次完整asset.New；close所有失败路径保持。目录真实读全FK，精确验证六个资产NO ACTION边，沿CASCADE反向推全部可清理祖先与步骤集合相等；user虽不purge仍验FK/index。将动态表数更新为真实24并保留非purge来源负例，不筛asset。
- [ ] Step 5：聚焦绿、全部asset/受影响owner/archtest、原删除/清理组合回归；变异CASCADE、漏states祖先、首页0提前停、未空metadata删行、失败Delete清记录、PublishDraft误调用删除、任务InsertTx改非事务，分别杀死。fixture是真实存储层证明，API与e2e扩展仍在T10，不写成已交付。
- [ ] Step 6：提交；首次表/同步删除/父任务/已知对象恢复/保护目录一个交付单元，不单独推半成品检查点。

## Task 6（T6）：申请、元数据与读取签发

**Files：** 新建api/modules/asset.yaml、http/gen/oapi-codegen.yaml及生成，asset/app/{prepare,read}.go、http/{assets,handler}.go；修改api/openapi.yaml、module.go、bootstrap/actions/routes/error目录测试、两语错误文案及配置样例。

**Interfaces：** Prepare.Execute(ctx,domain.NewAsset)(PreparedAsset,error)，PreparedAsset{Asset,Upload}；Get.Execute(ctx,id)(domain.Asset,error)；ResolveContent.Execute(ctx,id)(domain.SignedRequest,error)。路由只先登记本Task三个实际操作，不为T7/T8返回桩；共有DTO及错误一次定义。OpenAPI组件名为NewAsset、Asset、PreparedAsset、Upload、Content、DuplicateAsset，PreparedAsset含asset/upload；无手写TS镜像。

- [ ] Step 1：TestAssetPrepareReadContent：10种用途API申请、严格bodyshape/422与scope解析、大小限额、签名必要headers、DB失败不返回URL、持久expiry/safeAfter、元数据无key；pending content409。TestAssetContentReadNoWrites用实际SQL审计/行快照断言零UPDATE，并覆盖整点+不足5m换下一整点。
- [ ] Step 2：组合/HTTP测试红，契约source/response精确核验；纯返回固定DTO不能让字节相关断言通过。
- [ ] Step 3：先契约再生成实现；public business错误按spec精确码和状态登记、两语同步。Prepare在授权事务记录staging后CPU签名，成功commit才返回；resolve不写DB。完整asset.New接已创建owner门面/CallerLock，更新actionless理由但不改guard判词。
- [ ] Step 4：运行 `make gen`、`go -C server test ./internal/modules/asset/... ./internal/bootstrap -run 'AssetPrepare|AssetContent|AssetRead|Actions|Routes|Problem|Errors' -count=1`、`make test-web`；生成物随提交记录hash/行数。变异遗漏required header、完整query日志、无CallerLock、JSON限额放宽至少有对应失败证据。
- [ ] Step 5：提交。普通read URL在P7完成实物后增加真GET字节证据，当前不能假称F1闭环。

## Task 7（T7）：确认与唯一不可变发布

**Files：** asset/app/complete.go、complete_storage.go与测试；http/complete.go、api/modules/asset.yaml及生成；bootstrap/assets_complete_test.go。

**Interfaces：** Complete.Execute(ctx,id)(domain.Asset,error)；消费T1validator/P1六端口/T5metadata与T3Access。候选与结果提交用同一路径，中间没有gate/tx/连接。

- [ ] Step 1：TestAssetCompleteRealBytes：真实HTTP PUT/GET，完整hash、重复complete同ID/final/审计；TestAssetCompleteRaces与TestAssetSealFailureRecovery：双独立候选、同大小不同字节、旧PUT重放、封存成功DB rollback、暂停>lease、另一请求已发布后本请求失败，保留每个对象和记录断言。
- [ ] Step 2：运行 `go -C server test ./internal/modules/asset/... ./internal/bootstrap -run 'AssetComplete|AssetSeal' -count=1`，红与失效原因留原件。
- [ ] Step 3：新增complete契约、生成；严格两段短事务和总外部deadline；完整双次流式校验，只有一个final胜者；lease/phase核验，不把timeout当不存在。storage error/no object/version change区分。
- [ ] Step 4：聚焦绿及validator/local合同；变异固定seal key、去expectedETag、信源metadata不验目的、最后不重验权限、超lease仍晋升、败者覆盖整份JSON，分别在实际层杀死。更新API错误两语若新增，已声明码不重复扩张。
- [ ] Step 5：提交，记录真实对象keys只入私有证据，不放日志/DTO。

## Task 8（T8）：复制与删除准备件

**Files：** app/{duplicate,delete}.go与测试、http/对应handler、API/生成、bootstrap/assets_duplicate_test.go/assets_delete_test.go。

**Interfaces：** Duplicate.Execute(ctx,sourceID,domain.DuplicateAsset)(domain.Asset,error)，Delete.Execute(ctx,id)error。目标Metadata Origin=copy，无staging；消费同一Access与存储操作。

- [ ] Step 1：TestAssetDuplicateAuthorizationAndBytes：两账号、源404目标可写/源可读目标403、跨workspace及正反projectID、相同文件名不同字节、附件转图片验证失败、成功target与source key不同；TestAssetDeletePrepared：重复404/原戳、不删除bound、删除与complete两方向。
- [ ] Step 2：聚焦 `go -C server test ./internal/modules/asset/... ./internal/bootstrap -run 'AssetDuplicate|AssetDeletePrepared' -count=1` 红。
- [ ] Step 3：先源后目标预检，两段一次排序多范围；目标candidate先commit、外部copy后重验两端；失败新目标由恢复规则接管。删除准备件不做同步对象I/O、不清metadata，已绑定返回asset.bound；真正绑定暂不开放。
- [ ] Step 4：绿及六API完整契约/错误目录；变异仅验目标、反转拒绝顺序、删后迟到发布、复用sourcekey而非独立对象，组合数据可杀。不能声称重试duplicate完全幂等，额外准备件边界写证据。
- [ ] Step 5：提交；六API无桩齐备。

## Task 9（T9）：孤立对象、迟到结果与真实River恢复

**Files：** app/cleanup_inventory.go/cleanup_cursor.go与测试、adapter/river/cleanup.go/delete_scope.go及真实worker测试、bootstrap/assets_cleanup_test.go/purge_cursor_test.go；完善T5已知对象清理与目录。

**Interfaces：** Cleanup.Execute与SweepCursor/Result按文件结构区；ObjectLister与TemporaryCleaner按T2；RecordOutput不是app依赖，只在river adapter。父cleanup固定cutoff与模块顺序同T4。

- [ ] Step 1：TestAssetCleanupClocksAndLateObjects：7d/60d相等不删、+epsilon才入选，business+61d且runtime上传/lease未过保留、配置变小仍按记录安全期、后到u/o、活final、跨root邻域。TestAssetSweepRestarts：真实job至少3页，小预算snooze、进程重新构造、失败output/实际storage失败及重试，最终无丢页。
- [ ] Step 2：串行 `go -C server test ./internal/modules/asset/... ./internal/bootstrap -run 'AssetCleanup|AssetSweep|PurgeCursor' -count=1` 红；River假实现只证明调用，不能替代持久状态。
- [ ] Step 3：完善未知对象/tmp核验与runtime24h、有限游标和错误计数；最后读自己的metadata，不在asset锁后读带锁owner；RecordOutput后才snooze，错误保留安全游标和key。阶段完整才Complete，不以0Deleted推断耗尽。
- [ ] Step 4：绿、故障再试与下一周期重新从头找旧受阻/新低ID；变异用业务时钟判断grant/age、runtime门无视、删除前不认领、output丢失、key被重新认领、错误当Complete，都由永久测试拦截。
- [ ] Step 5：提交；记录清理失败仍可观察且相关父保留，不能只报最后成功。

## Task 10（T10）：真实API故事与完整删除fixture

**Files：** 新建e2e/fixtures/storage.ts（普通函数再薄fixture）、e2e/stories/files/{f1-upload,f3-publication,f13-permissions,f14-deletion,f15-cleanup}.spec.ts；修改fixtures/furnish.ts/assert/catalog.ts/相关M3M4删除故事与clock/server配置入口。

**Interfaces：** `prepareFile(api,input):Promise<PreparedAsset>`、`uploadFile(upload,bytes):Promise<void>`、`completeFile(api,id):Promise<Asset>`、`readFile(api,id):Promise<Buffer>`；TS来自生成schema，raw传输不带API bearer/cookies。

- [ ] Step 1：故事只经API建workspace/project/issue/comment/draft和真实file；furnishIssueRows每类新parent FK至少一个有字节的资产，删除多层/反向关系/活邻域齐备，createdBy与deletedBy不同。API无bound，因此不制造假附件计数/页面断言。
- [ ] Step 2：运行选定files/M3M4删除故事的Playwright命令（先make build，使用现有e2e命令入口）；保存新增性质红；SQL仅断言，禁止前置插资产冒充API。
- [ ] Step 3：实现fixture/完整故事，短upload_ttl显式10s、正常transfer budget，条件等待runtime expiry（不是sleep）；Clock只快进业务保留，刷新有效凭据后继续，不能用过期session测试伪装文件故障。
- [ ] Step 4：聚焦故事绿、至少读字节摘要/数据库恢复状态/实际目录邻域；借平台clock重启验证root复用与清理恢复。knownW11若出现按一次重跑规则，不专项查。
- [ ] Step 5：提交。首次产品检查点前串行完整七项，记录每项实际输入/生成物已提交后gen-check；全通过才system git push当前HEAD并核准确CI，继续独立T11而不将远端未知写成通过。

## Task 11（T11）：跨模块交错与资源故障验收

**Files：** bootstrap/assets_interleaving_test.go/assets_resources_test.go、复用现有pgtest/helper；必要app/adapter局部根因修复同Task。

**Interfaces：** 不增加生产测试回调或假service；观察实际query PID/blocker，存储接口注入受控阻塞/错误但完整程序事务真实。

- [ ] Step 1：测试credential reset→新grant401；workspace/project撤权两方向；issue/comment删除/归档；draft改项目/发布；source失效和target改范围；两个项目正反UUID与重复同项目；source404不进gate；最终检查仍在锁下。
- [ ] Step 2：TestAssetStorageDoesNotHoldResources：Inspect/Open/Seal分别卡住，另一个请求能取同账号/项目/资产锁并拿到连接/门；取消所有reader/资源释放，最终parent/asset变化让迟到提交拒绝。TestAssetScopeDeletionRollback验证事务event/metadata/job统一回滚。
- [ ] Step 3：运行 `go -C server test -race ./internal/bootstrap -run 'Asset.*(Interleav|Resource|Rollback|Lock)' -count=5 -timeout=15m`，必要分组仍串行，正常写零40P01。物理purge允许重试与普通写分列。
- [ ] Step 4：反例去caller首锁、错分gate、移final重验到锁外、asset后追project，必须有真实PG/程序性质红。任何新架构级环路先提交再上报，局部不足直接根因修。
- [ ] Step 5：提交，按准确最终锁图更新spec事实，不把计划当已运行证据。

## Task 12（T12）：性质变异、七项门禁与文档证据

**Files：** 本phase相关产品/测试/plane-diff、docs/v0/M5-files/reviews/P2-assets-review.md、input handoff处理结果，ignored证据台账。

- [ ] Step 1：逐A1–A11对应实际测试；汇总至少15个有效定向变异（权限/锁/gate/版本/lease/回收/purge/runtime/目录/错误）。变异前备份原字节，单项运行后恢复hash，禁止stash/reset/clean；编译红单列，不冒充性质红。
- [ ] Step 2：串行完整 `make gen`→提交生成物→`make gen-check`、`make lint-go`、`make test`、`make lint-web`、`make knip`、`make test-web`、`make e2e`。记录raw命令、准确tree、退出码、数字及历史失败；容器单套件。
- [ ] Step 3：review逐条证据与局部裁定/代价；更新M5进度为实施完成待整审，M4/M5联合合同标实际落点；输入handoff只追加处理结果保持open。
- [ ] Step 4：提交推送检查点，核准确SHA/run/三个job，不能用旧绿替代。现有工具能读日志直接读，不取凭据；真实失败走根因，W11一次重跑仍失败停报。

## Task 13（T13）：deep-review与修复闭环

**Files：** docs/reviews下下一编号的M5P2整体审核、阶段review、实际发现对应文件。

- [ ] Step 1：按deep-review完整读取基线85dc4087至最终候选，三视角顺序核逻辑、集成、测试；包括全部purge caller、父目录、shared gate/账户首锁、暂停后对象、权限矩阵和生成协议。
- [ ] Step 2：按executing-plans/requesting-code-review要求安排一次新鲜完整分支审查；按范围给资料，不重复发相同未改动范围。主控裁定后局部问题修复/必要回归/复审，原始失败不删；重大新分歧先提交上报。
- [ ] Step 3：有新产品改动重验受影响与必要完整门禁；最终准确branch CI全绿。阶段review补全七节，A1–A11据实勾选，A12 main/归档部分待T14。
- [ ] Step 4：提交最终文档并核其准确CI；不把文档前产品CI写成最后HEAD通过。

## Task 14（T14）：已授权集成、main CI和下一阶段

- [ ] Step 1：核分支干净、准确main与基线关系及全绿HEAD；`--no-ff`合并main，详细中文说明，系统git正常推送。
- [ ] Step 2：准确main SHA的server/web/e2e全绿；失败按根因，已知W11重跑仍不过停报。分支绿绝不替代main。
- [ ] Step 3：主目录ignored归档完整raw/审查/变异/失败/Playwright产物，manifest核SHA256/bytes/mode/membership，未知0；核远端全合并后常规删除local/remote分支和worktree，清理后再验归档。
- [ ] Step 4：P2标完成、A12收据、M5进度更新；P3 S3 spec/plan/镜像版本与预检先行，普通推进遵从连续授权，重大新发现例外。M5总体完成项保持未勾选。

## 预检与覆盖索引

| 合同 | Tasks |
|---|---|
| spec§2.2 表/纯规则、2.4图片 | T1/T5 |
| spec§2.3六API/生成/两语 | T6–T8 |
| access全部权限/锁/构造 | T3/T5/T6/T8/T11 |
| recovery§1–4状态/运行期 | T1/T5/T7–T9 |
| recovery§5–6清单/游标 | T2/T4/T9 |
| protected-purge全套DB/目录 | T4/T5/T9/T11 |
| A1/A2/A3/A4 | T6–T8/T10，T7/T10，T3/T8/T10，T3/T11 |
| A5/A6/A7/A8 | T7/T11，T8/T10，T5/T10/T11，T2/T5/T9/T10 |
| A9/A10/A11/A12 | T4/T5，T2/T4/T9，T1/T7，T12–T14 |

14 Tasks，执行前自检每个步骤的签名/值/验收可落地、Review Focus均有测试；builder check问题归入已回答/局部裁定/负责人决策，完成后另写预检记录。独立builder check与主控自检已通过，原件和范围限制见re-34。
