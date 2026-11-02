# M5 / P2 阶段评审记录

> 实施中：T1–T9完成，T10–T14待执行；没有进行整分支终审、没有合并。基线main85dc4087，预检提交72a3e58b，见[re-34](../../../reviews/re-34-M5P2预检.md)。原件在本工作树 `.superpowers/sdd/P2-assets`。

## 1. 方式

主控按executing-plans逐Task实施/TDD/自查，阶段末deep-review三视角与一次新鲜整分支审查。P2完整builder check仅为设计预检，不当产品审查。

## 2. 验收进度

A1–A12均保持未完成。T1提供用途/范围/父组合、metadata≤3候选/16KiB及受限图片校验；它不提供API、数据库或完整恢复链，不能据此勾选阶段标准。

## 3. T1已完成的验证与裁定

- domain/content 的setup编译红单列；最小零行为声明后真实性质红，最终绿。四格式实物头、完整字节hash、同长度异尾、头部≤1MiB、边长/像素限制、过短/过长流、取消均覆盖。1×1 WebP夹具另用官方完整解码确认可读。
- `task-1-mutations.json`记录4个行为变异被杀：取消候选上限、只摘要首块、只信声明图片、遗漏EOF取消。原字节hash恢复核验。
- `task-1-affected-green.log`：asset全包/archtest/config/httpserver的race通过。`task-1-lint.log`：资产相关golangci-lint 0 issues。关键词76规则/14例外无命中。
- `task-1-full-go.log/json`：实际make test完整server/tools通过，退出0；测试输入逐文件hash在task-1-tested-inputs.json，原件没有假记全项目七项已跑。
- x/image0.47的必要传递链为x/sys0.49、x/text0.43，后者要求现有x/tools0.51/x/sync0.24。go mod tidy把原本直接import的rivertype归到直接依赖，版本未变。模块图保存在task-1-dependency-graph.txt。
- validator只消费io.Reader，Close由拥有ObjectReader的app在T7验证；T1不声称证明上层Close。ETag最长1024字节且总metadata≤16KiB，超长显式拒绝不截断，P3真实provider合同继续核对。
- P1 inline类型检查复用同一RasterMediaType，既有签名读回归通过，无新类型放行。图片仅头部验证，不冒称全像素/动画安全证明。

## 4. 整分支审查

未开始；T13执行。

## 5. 与plan差异

T1确认必须的完整Go依赖传递链，补齐spec/plan；T1无ETag输入，原“ETag当摘要”变异具体化为漏算尾部字节，真正ETag版本与SHA不同职责在T7封存组合验证。每条局部裁定及代价另见台账。

## 6. 后续责任

T9已完成对象/临时目录对账与真实worker恢复；接续T10 API故事与首次完整七项检查点。P4/P5绑定、P3 S3、P6/P7页面以及P1遗留C12 Minor都未完成；输入handoff保持open。

## 7. 已知限制

当前已有基础纯规则/校验、本地有限清单/临时回收、宿主授权与锁编排、父清理单页与River持久进度；已有file_assets、同步删除和资产清理worker；已有六个业务操作和真实上传/复制/读取/软删除字节链；已补齐对象对账/重启恢复。完整API故事和交错验收仍待T10/T11，不能根据T1–T9通过声称F1/F3/F13/F14/F15已满足。

## 8. T2本地清单、临时目录与配置

- 实际红/绿在task-2-red.log与task-2-focused-green.log；setup编译红分开保留。10,000个真实对象以乱序创建，10页完整覆盖；最终含内部布局检查的清单读取1.027秒。目录每次读取256项、只保留最小limit+1；另外在10,000目录的3条结果页验证窗口容量≤4。不是任意规模性能承诺。
- 四个变异实际行为失败且源恢复：取消根限制、取消ctx检查、无界保留、全量枚举。年龄来自实物ModTime与调用方cutoff，严格边界/另一个root/部分进度保留已测；业务clock误接线变异需在T9真正调用者存在后完成，不冒充本Task已证明。
- 未知目录、符号链接、损坏metadata、对象内未知文件均失败且不删除；每页只读取本页metadata，非快照，后页损坏到该页失败。临时清理仅合法UUID目录中data/meta.json的已知布局，错误会携带此前已完成的游标和删除数。
- 五项生命周期配置范围、默认、溢出、未知键与日志遮盖通过；test profile继承安全默认，期限故事用显式覆盖。已有P1签名/传输与全部asset/config/configs/archtest race通过。lint首轮仅导入分组错误，修正后0 issues，原失败保留。
- 局部裁定为本页metadata读取、profile默认、对象内部未知布局及业务clock变异层级；结论、理由和代价见台账。

## 9. T3宿主授权、锁协议与父删除事实

- Access 的真实性质红/绿、三个 owner 门面的红/绿、转接的红/绿各留原件；真实 PostgreSQL 的锁/父事实测试与156项角色矩阵通过。矩阵使用 API 建立宿主，实际调用各 owner 的现有权限判断，不把尚未上线的资产 HTTP 路由说成已测。
- 实际调用次序为 Inspect → 同一项目门 → Tx → credential 时钟与 CallerLock → 排序 W(S)/P(N)/D(U) → Check → 资产 U → 审计时钟 → 包内回调。错误、取消和 panic 释放资源；源目标的语义顺序独立于物理锁排序。
- PostgreSQL 测试绑定请求 PID、SQL 与 blocker，反向 ID 输入证明先持低 ID 再等高 ID；W 锁与 SHARE 兼容、D 锁与 KEY SHARE 冲突。父事实读取不额外锁父行，原时间与删除人逐值保留，创建者不同于删除者。
- `task-3-mutations.json` 六项均命中行为失败：绕过 owner、W 逆序、D 逆序、另建项目门、失去 HiddenFrom、误用审计时钟做 credential 校验。每次恢复原文件并核 SHA256。
- 局部裁定：issue 以 `AssetProjectReader` 读取已持锁项目的归档状态，避免再次锁项目；错误接线会破坏拒绝次序，真实查询及矩阵约束它。T3 提取统一 `projectWrites` 并验证资产转接使用同一实例；实际业务模块在 T5 一次构造，HTTP 交错在 T11 证明，不以转接单测替代最终接线。
- plan 原有回调示意与锁次序笔误已按 spec 和实际符号同步：Check 在父锁后、资产锁前；`AccessHeld` 携带最终锁后的时刻；`ParentKey.Kind` 使用具名 ScopeKind。业务用例仍须完成用途、绑定、父链和 revision 复查。
- `task-3-race.log`：asset/issue/project/workspace 全包、bootstrap 与 archtest 的串行 race 回归通过，bootstrap 653.318秒；源码快照核验未变。lint 首轮发现三个适配器未被使用，真实锁/矩阵改为经过这些适配器后，`task-3-lint-final.log` 为0 issues。真实 P 锁转接、错误原样传递与最终聚焦 race 通过（bootstrap 45.388秒）；没有放宽 lint 或架构守卫。阶段验收仍不勾选。

## 10. T4父清理分页与River持久进度

- 四个owner改为每次一表一页的Page，生产中不再保留无界Purge入口；原清理图、60天边界、独立提交、迟到删除与父CASCADE测试经测试专用drain迁移，未删原断言。issue/project/workspace构造时复制明确保护标识，T4当时生产为空；后续T5已将真实资产约束及精确目录接上，不把空集合当保护已交付。
- `PurgeProtected`仅识别23503与schema/table/constraint全匹配。真实PG父→子→资产及自引用证明整批失败全部回滚，逐候选得到Scanned3/Deleted1/Blocked2，受阻前后邻域继续；取消/未知FK/23001/40P01保留此前独立提交进度并返回错误。
- 普通固定SQL用有限物化候选CTE和DELETE CTE分别返回扫描数、删除数与候选最大ID，DELETE再核cutoff。实际PID/SQL/blocker交错先复现候选被并发删除时错误耗尽，再复现候选被改到精确cutoff仍遭删除；两个方向修复后通过。无跨模块SQL、额外父锁或表名拼接。
- bootstrap预算使用运行时钟，单轮业务cutoff保存在全局模块/表/After游标；worker在snooze和错误前RecordOutput，编码上限16KiB，未知版本/模块/表或损坏输出明确失败。真实River在同一job的snooze后停机，换Runner且业务快进30天仍从原After/cutoff继续，完整页前进后完成；不声称每页恰好一次或计数跨重试恰好一次。
- `task-4-mutations.json`六项变异全部被杀且恢复字节hash一致：Deleted判断耗尽、只看23503、重置cutoff、丢After、漏RecordOutput、扫描数冒充删除数。失败源码/日志分别保留，不把编译错误当性质红。
- 首次受影响race暴露旧请求探针仍等待空页DELETE；新受保护路径空页不发DELETE。夹具先阻住真实前置清理、提交过期候选，再移交issues屏障；日志确认真实DeletePurgeIssues与业务MarkChanged各自PID/SQL，原项目锁/工作区锁/未提前写断言保留。聚焦RED与GREEN原件在task-4-purge-probe-*.log。
- 最终串行race覆盖platform/postgres、四owner的app/adapter及bootstrap清理/父CASCADE回归全部通过，bootstrap52.129秒；archtest1.690秒；全server lint 0 issues。1506个测试输入hash在task-4-final-inputs.json，运行后核验未变。原首次race失败及错误cwd的lint调用保留；正式结果只取final日志。
- 局部裁定及代价：单页接口让遗漏调用者编译失败；保护标识构造复制避免后注入；有限CTE保留真实扫描成本；DELETE cutoff复核避免等待后的过期事实；真实候选夹具保持原锁证明。完整理由与错误代价在台账，阶段A1–A12仍未勾选。

## 11. T5资产首表、同步删除与受保护恢复

- 00034建立23列file_assets、六个具名即时NO ACTION父引用及无条件索引，审计SET NULL。SQL约束拒绝metadata缺字段、JSON null、未知版本、错误类型、第四候选、超16KiB及错误范围组合；完整迁移上下往返、精确约束/索引目录和24张墓碑表目录已扩展。原守卫拒绝新表/新边的红保存在task-5-old-catalog-red.log，没有先排除asset再写断言。
- 资产仓储只用自有sqlc固定查询；SaveAsset比较StorageRevision并保留申请者/用途/父范围/大小，读取非法metadata明确失败。purge SQL在候选和DELETE两处均要求严格60天及全部key记录为空。原生成物重新生成后逐hash相等，未手改生成结果。
- issue子树、comment同步事件和真正DeleteDraft在原事务使用原At/By标记资产；三层子树/评论/草稿与未删邻域均通过。后续订阅者失败会连同资产和父行整体回滚。PublishDraft内部消费草稿不调用删除资产端口，负例有真实程序回归。
- project/workspace事件只InsertTx范围任务。实际事务内可见、事务外不可见，rollback后任务为0；真实运行中的asset.delete_scope任务按ID/完整参数/一次成功attempt观察，原时间和删除人不变。scope分页第二批失败只回滚本批，已提交批次的游标保留并恢复。
- 已知对象回收先读owner原始父删除事实，再锁资产核scope/revision；最早父时间及原删除人不后移。先提交reclaiming，再在无数据库连接/事务时Delete，成功后短事务核同一key/phase移记录。真实本地文件失败时文件、metadata和父保护都保留；重试成功后资产行与父行逐步收敛。候选文件已删但确认事务失败的场景保留reclaiming并幂等重试，旧父事实遇新revision会跳过。
- 真实本地对象覆盖live final、59天、精确60天、61天、pending staging、7天过期件与业务快进但真实上传安全期未过。T5只交付已知key，孤立对象/tmp对账、迟到落盘和真实asset worker跨进程恢复仍是T9责任。
- 目录从全库真实FK图推导所有受保护可清理祖先，包括states→issues及issue自引用，与声明步骤集合精确相等；user虽不purge仍核FK/index。删声明、漏祖先、CASCADE/SET NULL/RESTRICT/deferred变体均拒绝。六类父表加issue自引用，各以batch1/3验证最小/最大受阻、中间可删，先Scanned3/Deleted1/Blocked2，再经实际资产purge释放后收敛。
- 原项目/工作区/工作项删除目录夹具已补真实资产行，不过滤新表。旧迟到父CASCADE先证明受保护资产保留，再用实际owner事实恢复后验证原级联清空；未改近期/存活邻域及账户不变断言。资产近期删除审计按新的原父时间合同单独逐值核对。此处存储夹具不冒称T10公开API故事。
- 实际runtime账号暴露遗漏file_assets权限，deploy/runtime-grants.sql已按既有四项DML补齐，未增加DDL/TRUNCATE等权限；原全目录权限检查与受限账号删除/清理回归通过。部分旧测试专用Runner漏注册新kind也已补齐真实资产范围worker，保留原issue事件观测。运行账号的DELETE观察器实测asset→issue→project→workspace→identity；完整应用的周期清理遇受保护项目仍能完成并清理无关邻项。
- task-5-mutations.json中七项行为变异全被杀且恢复字节hash一致：CASCADE、漏states、Deleted零提前耗尽、metadata未空硬删、Delete失败清记录、PublishDraft误删资产、非事务入队。task-5-wiring-mutation.json另证明生产保护标识为空时完整程序失败；首次untyped nil仅编译失败，另存compile-attempt，不计性质证据。全部历史失败和修复日志保留。
- 实际符号较plan原示意精简为FindAsset/CreateAsset/SaveAsset与原Asset审计、revision；SweepCursor同时保存业务/runtime cutoff，Scanned和Removed分开。候选DB确认失败不前移游标，对象删除失败保留记录并让下一次从邻域继续，失败行由下轮周期重看。详细理由及错误代价见台账各Ruling。
- 最终验证通过：输入1542文件快照为task-5-final-inputs.json，运行后逐hash未变；资产全包、迁移、archtest race及父模块/组合层扩大race均通过，bootstrap188.804秒，lint为0 issues。收据在task-5-verification.json；无完整七项或远端CI通过声明，按计划T10前不推产品检查点。


## 12. T6申请、元数据与内容签发

- 三个实际操作 createAsset/getAsset/resolveAssetContent 按契约生成Go/bodyshape/TS后接入完整asset模块。没有complete/duplicate/delete桩；新五个错误码与两语文案同Task登记，未改变动作或错误目录守卫。413/415仅在HTTP边界映射，其余owner错误原样保留。
- `task-6-prepare-red.log`与`task-6-read-red.log`为真实路由缺失的HTTP性质红。十种用途、十二种用途/范围组合经过API申请并真实PUT，检查完整父范围、必要Content-Type、持久expiry和本次传输预算+30s的SafeAfter；业务时钟+24h仍签运行时期限。无效结构/字段、实例限额与100MiB上限、非图片用途声明、非申请者逐值404以及pending内容409均验证。
- 延后到COMMIT才失败的真实数据库约束证明申请行回滚、API只返回500而无URL。元数据不返回key/上传capability；普通读取在同一只读快照授权。真实HTTP在FindAsset结束后停住，第二请求删除issue并提交，恢复后的原读仍见同一快照，新读404；记录生产BEGIN为REPEATABLE READ READ ONLY。
- 内容签发使用同一Access/CallerLock/owner门面和最终锁下复查，不UPDATE资产。三轮GET内容/元数据的真实语句级DML审计为0，整行快照不变。精确UTC整点、恰好5m、少1ns、非UTC输入和跨日由实际签发用例测试；声明image/png的附件仍强制attachment。
- `task-6-mutations.json`七项行为变异失败且原字节哈希恢复：漏required header、完整query进入访问日志、移除CallerLock、放宽普通JSON限额、5m边界误含相等、生产读取缺失快照、上传误用业务clock。第一版query变异落到panic日志，访问日志测试未失败；原件单列保留，改为真正访问日志语句后命中失败，不把第一次存活说成拦截。
- 最终生成输入3967文件不变，91生成物的SHA256/行数在`task-6-generated.json`；最终生成无Problem重名警告。扩大race回归通过（bootstrap58.229秒），完整asset HTTP套件及错误目录闭合通过，server lint 0 issues，`make test-web`18/18任务通过。命令/输入/日志见`task-6-verification.json`。仍未运行本Phase完整七项和远端产品检查点。
- 首次读取校验失败因测试额外store在空PEM配置下另生成密钥；同一测试PEM后通过，未改签名逻辑。HTTP单测起初断言全局缓存头，已将该断言归到真实应用调用；快照SQL大小写断言也按实际语义修正。原失败/编译尝试均保留。
- 局部裁定：未来T7/T8错误随其首次消费者登记，以保持精确目录而不虚报可返回错误；HTTP拒绝前置顺序明示沿用M2路径绑定→认证→bodyshape→字段，不另建资产认证链。代价和理由在台账，未修改宿主可见性/角色/归档顺序。raw仍为P1最终`/static/local/`。
- 上传字节已真实落盘，普通内容URL当前用显式storage fixture核验签名；完成实物后的GET字节与完整hash仍是T7责任，不能据此关闭F1。A1–A12及输入handoff继续开放。

## 13. T7确认、不可变发布与失败恢复依据

- `task-7-http-red.log`和`task-7-failure-red.log`记录真实complete路由缺失的性质红。契约先生成再接用例；初次联调漏总OpenAPI路径引用，虽然实际行为已返回正确结果，契约检查仍失败，原件保留，补齐引用后通过。当前四个真实业务操作，无复制/删除桩。
- 真实HTTP PUT→complete→content→GET覆盖普通附件、头像及声明PNG的附件：完整SHA256、唯一o-key、已验证图片尺寸、附件download、重复确认DTO/整行审计逐值不变，以及旧PUT409后已发布字节不变。staging仍留存到已持久化安全期，候选只在最终事务移为final。
- 两次真实请求各自封存后受控暂停，第二次先发布，第一请求成功或外部报错均重新授权后返回同一胜者。胜者不丢另一个候选，败者不覆写final/审计；两个实物仍可查。封存后最终COMMIT约束失败保留候选与实物，重试另用新key并保留旧记录。runtime快进31秒的迟到结果拒绝晋升；同长度异字节的目的读取拒绝发布。
- source/target读取失败、Close失败和独立外部超时均为503而非缺失；每个Open均Close。客户端取消使受阻reader实际结束并关闭，候选仍在。一次完整外部段的两个Open使用相同deadline。缺staging为409、伪PNG为422，三个候选满额503并有Retry-After:1。
- `task-7-mutations.json`七项最终均为行为失败且恢复原SHA256：固定候选key、缺Seal版本、只信目的metadata、最终只验读权限、超lease发布、旧metadata覆盖、遗漏reader.Close。第一次权限变异存活，因为移除成员让读写均403；改为自己创建的issue上降为guest并先证明元数据GET200后，错误ReadAsset意图被实际HTTP拒绝性质拦截。最初存活记录独立保留。
- 首次扩大race发现旧零DML夹具与启动全局purge交错；十次诊断中两次抓到确切`PurgeAssets`空DELETE。夹具现在等待两个实际启动清理任务完成再装审计，不关闭worker、不排除任何DML；额外保留SQL诊断。十次race复跑通过（30.236秒），既有行快照/零语句断言仍在。
- 最终受影响race通过（bootstrap35.495秒、archtest1.464秒），asset HTTP完整目录通过（4.696秒），server lint 0 issues，`make test-web`18/18（1缓存，40.671秒）。3851个选定源码输入哈希运行后未变；生成物逐文件SHA256/bytes/行数在`task-7-generated.json`，命令和退出码见`task-7-verification.json`。完整七项/远端产品检查点、T10故事和T11完整资源交错尚未执行，不把本Task说成阶段通过。
- 局部裁定：并发候选修改允许revision推进，但保存仍按当前锁定revision CAS，最终必须核自己的key/phase/lease及完整父范围；metadata预约不改业务审计，成功发布才改At/By。请求取消沿用context，独立预算超时/后端故障给具名503，不把私有路径或签名URL带进公共错误日志。代价与理由已记台账；API A1–A12及输入handoff保持开放。

## 14. T8复制与删除准备件

- `task-8-http-red.log`和`task-8-copy-red.log`为真实接口未实现的性质红；先契约/生成再接用例。duplicate与delete加入后六个真实业务操作齐备，`asset.bound`与两语同Task接入；新增错误没有放宽精确目录。所有普通JSON、认证和owner角色/归档判定沿用原路径。
- 真实API覆盖三个项目、两个工作区、项目ID正反方向及同项目复制；同文件名不同字节不混淆。每次duplicate返回新资产、新final key、独立实物且没有staging/上传授权，读回字节、SHA256与源一致，源整行不变。相同请求再执行仍创建新的准备件，不声明跨请求幂等。
- 双账号与私有项目撤权覆盖：他人准备件/自己已不可见的源到可写目标均404；源可读但目标准备无权403；不可见源优先于目标403，并且不创建目标行。普通附件转头像重新验真实图片：PNG成功且有尺寸/inline，伪PNG422并保留pending候选，text/plain声明415。
- 目标行及候选提交后才进入实际Seal；受控暂停期间分别删除源/目标，最后都404且不发布目标，已封存实物和候选仍存在。准备件pending和uploaded都能软删，保留metadata/实物和原审计；重复404不后移删除戳。先删除后迟到complete被拒绝，先complete再删除保留final。bound仅为显式存储夹具证明拒绝，不冒称绑定API上线。
- complete/duplicate共用纯候选晋升及事务外双流封存，复制在Seal前核源已发布摘要，最终还核源key/摘要/不透明版本。复用时曾误信恢复记录中的重复size：故意制造声明4字节/实物3字节的DB夹具后两接口错误成功，原RED已保留；现以资产声明构造校验预期，两者422并保留恢复依据。此为边界故障注入，不表示公开API能改声明。
- `task-8-mutations.json`五项行为失败且恢复hash：只验目标（错误201）、目标先拒绝（403替代404）、允许已删资产迟到发布（错误200）、复用源key（目标所有权结构校验拒绝，API500）、跳过源已发布SHA（错误201）。测试不以编译失败代替性质证据。
- 最终受影响race通过（bootstrap39.525秒、archtest1.824秒），完整asset HTTP与错误目录4.778秒通过，server lint 0 issues，`make test-web`18/18（1缓存，39.622秒）。3860个选定源码输入哈希未变；84个生成文件的SHA256/bytes/行数在`task-8-generated.json`，命令与退出码在`task-8-verification.json`。完整七项、E2E故事、远端检查点和阶段整审仍按T10–T13执行。
- 局部裁定：源实物缺失统一为asset.object_missing，与后端异常分开；复制没有上传能力，失败目标由7天准备件规则和候选安全期回收；共用封存变化始终带回T7回归。原因与代价已记台账，A1–A12和输入handoff仍开放。

## 15. T9完整对账、迟到对象与持久恢复

- 产品提交423281e6。`task-9-inventory-red.log`实际复现旧代码扫完rows即Complete，孤立对象和旧临时目录仍在；补齐rows→objects→tmp→done，只有全部阶段结束才Complete。T2清单/临时端口在bootstrap注入同一本地store，domain保持纯净，没有新SQL、跨模块锁或守卫豁免。
- `TestAssetCleanupClocksAndLateObjects`与`StrictBusinessBoundaries`验证7d/60d/24h严格边界及+1微秒；业务+61d不越过持久grant/lease，当前外部预算1s不重算记录safe_after。真实旧u/o、同assetID不同key、活final、未过安全期的staging/candidate、另一root与未知布局各自有实物断言。隔离存储fixture的bound状态只证明保护规则，不声称P2提供绑定API。
- `TestAssetCleanupLateBytesAfterAcknowledgement`先确认不存在对象并移除恢复记录，再在存储边界注入同一旧u/o的迟到实物；记录不会重新认领，真实24小时相等时保留、+1微秒后回收。这是故障fixture，不声称过期公共上传授权能重新签发。发布纯状态测试分别拒绝已移除/已认领key且不改审计。
- 对账最后只在自身资产表短事务读全部metadata引用，外部Delete不占连接/事务；对象列表必须升序且有限，Token为不透明页起点、ObjectAfter为页内安全位置，未完整处理不采用Next。临时失败保留部分进度/实物；所有已知key安全期仍用记录值。预算到期重放当前项，普通存储失败可越过该项继续邻域并由下周期重看。
- `TestAssetSweepRestarts`运行真实River与local：5个对象、batch2，chmod造成真实Delete失败；相邻项继续且snooze游标落入metadata.output；停止runner、关闭store，按同根重建实例，业务+61d/真实+49h仍沿原双cutoff续扫；保留错误历史与累计Failed=1；新job从头回收旧失败项、后来低ID及上一轮年轻对象。未把重建runner说成杀死宿主OS进程。
- output支持objects/tmp/done并严格核阶段字段、编码token≤8KiB/总output≤16KiB；`RecordOutput`失败原样返回，不以snooze错误冒充验证。首次River夹具在空errors数组上遇到cardinality=NULL，改只读断言coalesce；原件保留，非产品失败。
- `task-9-mutations.json`九项均行为失败且SHA恢复：业务时钟裁孤立年龄、忽略持久runtime安全期、未提交认领就Delete、丢River output、重认领已移除key、重认领reclaiming key、存储错误当Complete、预算中途跳过整页、忽略活引用。该轮没有编译红或幸存项。
- `task-9-focused-green.log`、`task-9-verification.json`：受影响race通过（bootstrap45.367秒、archtest1.669秒），完整HTTP/错误目录4.697秒通过，server lint 0 issues；3867个选定源码输入哈希未变。本Task没有API/生成物/前端错误改动，T8生成物收据仍适用。完整七项与首次推送检查点留T10，不把尚未跑的E2E或远端状态写成通过。局部裁定/代价见台账与recovery§6；A1–A12及handoff继续开放。
