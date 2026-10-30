# M4/P11 详情数据层：预检与实施记录

日期：2026-10-30。**当前：文档预检通过，负责人已授权实施，T1–T14已实现，T15整审修复已完成，最终门禁验证中。** 负责人已裁定P11-S1采用A，设计/spec/plan/交接同步后完成复核。[spec](../specs/P11-web-detail-data.md)、[plan](../plans/P11-web-detail-data.md)、[deep-review记录](../../../reviews/re-14-M4P11预检审核.md)、[已裁定记录](P11-project-number-width-decision.md)。产品验收仍开放；准备阶段证据保持原义，实施证据见§8。

## 1. 前置、身份与范围

P10b最终交付2aaef85c已授权--no-ff合并main `df5d3bbc1887486bd105fe6fc663f57f5b371992`，准确main CI40908365144 attempt1 server/web/e2e全绿，归档后正常清理分支和worktree，详见[P10b review§17](P10b-web-list-data-review.md#17-授权合并main-ci与归档清理2026-10-30)。

从此main创建 `codex/m4-p11` / `.codex/worktress/codex/m4-p11`；主目录一直main。准备阶段用户只授权spec/plan和预检，当时没有产品改动、依赖安装、原型/代码重放或产品测试。准备原件在主目录 `.superpowers/sdd/P11-preparation-20261030/`，在worktree外，未来归档不能遗漏。

## 2. 预检时的验收映射（实施进展见§8）

| spec验收 | 本轮处理 | 产品状态 |
|---|---|---|
| A1–A3 服务/资源/确认 | 对照生成合同、root/permission/Reconciled及旧调用，定义请求/竞态任务并保留既有回归 | 未实施、未运行 |
| A4–A6 协作/版本/草稿 | 校正评论表情读取、cursor请求含义、原生DTO、M7边界和PATCH→publish；草稿清空/无项目/拒绝已落到消费者任务 | 未实施、未运行 |
| A7–A8 实例/提及/拒绝/宿主 | 原7实例+两旧IssueService提前接线，四个真实提及入口、权限key与宿主owner明确 | 未实施、未运行 |
| A9 错误/标签/宽度 | 标签范围按M7边界修正；宽度按A统一/增宽/scope及权限重置 | 未实施、未运行 |
| A10 证据与终审 | 已列Task验证/至少15性质/F/准确CI及终审要求 | 本轮完成文档预检，非阶段终审 |

## 3. 方法与实际覆盖

使用brainstorming/writing-plans准备，按brainstorming要求另派全新只读执行者检查。主控deep-review依次从逻辑、集成、覆盖核文档；独立builder-check不是三位审者，也不是未来产品终审。

全文读取spec/plan及映射服务、store、直接消费者；原consumer-inventory的77条关键词路径在恢复预检后逐批补完全文，另读root权限/写队列、Reconciled/SessionSWR、成员目录、List链路、草稿表单/provider与相关测试。issue.fetch/sub_issues.fetch、issue.store/issue-changes、root.store、reconciled/SessionSWR、base.loading/base.clone、workspace-draft/root、list-data/group-consumers/removal等回归已核。逐路径/行数/hash与读取方式见review-coverage.json；consumer-inventory本身仍只是搜索库存，不是自动产生的阅读证明。

逻辑视角核原生字段、payload清空、列宽与异步失效；集成视角追实际service/DTO读者、M7边界、宿主和删除旧服务的编译依赖；覆盖视角对照既有回归与各Task的请求、拒绝、竞态、确认和变异目标。大设计按相关节、YAML/生成schema/关键词按合同结构核查，不冒称全文审查全部生成物或全仓库。已知旧实现与目标不符的路径是实施输入，已映射Task，预检通过不是旧代码已修好。

API清单来自本地YAML解析30个操作；当前phase=M4/P10、无until=M4/P11，P11旧地址21个exact key期限M4/P12。没有运行未来计划的Vitest/编译/F/变异，也没有用main已有绿色测试替代P11验证。

## 4. 发现、修订与复核

| ID | 级别/根因 | 处理及当前状态 |
|---|---|---|
| PF-1 | Major，文稿局部接口误记 | plan T2的IssueDetail.identifier改为Project.identifier+sequence_id，错误按钮回项目列表。核schema/真实route；文稿已修，产品未实施。 |
| BQ-1 / P11-S1 | Major，上游设计与公开合同冲突 | Project无next_work_item_sequence，先提交c437f6ea再上报。负责人选择A，按已加载可见行统一列宽、scope/权限重算，零列宽专用HTTP；设计/spec/plan/M3交接同步。决策关闭，未修产品、未放宽守卫。 |
| BQ-2 / PF-2 | Major，文稿局部虚构目录 | 仅project标签API、labelMap仅已取目录；spec§2.10/planT12保留缺标签限制，完整恢复M7，无新API。 |
| PF-3 | Major，回归迁移依赖未写实 | issue-changes仍spy旧IssueService，base.loading和draft/root用旧请求；plan全局/T2–T4/T8同步迁原生fixture，保留R31/R32、弹窗加载/克隆及筛选隔离，不删断言换绿。 |
| PF-4 | Major，草稿边界未落到消费者 | payload不接受null，Draft无created_by且可无项目；spec§2.8/planT8补键删除、顶层project_id区别、无项目展示/保存/删除、拒绝留输入及测试。完整页面故事仍P15。 |
| PF-5 | Minor，具体调用表漏项 | T9写实四个searchEntity调用；T4列CycleStore无调用旧服务三处删除；T3/T8明确确认回调不得二次写，M6目录回归保留。未扩M6/M7业务。 |

独立审者 `/root/p11_builder_precheck` 首次全文核spec/plan发现BQ-1/BQ-2；裁定后全文复核A修订稿，随后限定复核PF-3/PF-4和最终调用表，均返回“nothing important left”。原始结论/输入hash在builder-check.md；三次复核及hash在builder-recheck-a/local/callers.json。历史builder-input快照晚于首次PF-1修正，不能冒称历史输入hash都等于快照。最终状态文字与T3既定DOM断言的目标文件补注另存hash桥/静态回执，不冒充独立审者检查了后来的提交SHA。

## 5. 静态验证与证据边界

准备目录保存api-inventory.json、consumer-inventory.json、builder-check.md、修订快照、decision-evidence.json、review-coverage.json及最终静态回执。检查链接/路径与锚点、合同、关键词期限、14Task/10条开放验收、仅docs diff与main不动；源hash与已读基线一致。历史blocked回执保留，不覆盖为通过。实际命令/结果由回执记录；零产品测试不是失败或跳过的产品绿。

P10b证据保持原hash及历史限制，不替历史变异补红绿。准备提交SHA/推送回执放忽略目录，不循环写文档自身SHA。准确main CI已核；准备分支CI按实际提交单独记录，不能把它当P11新功能的产品证明。

## 6. 移交与未完成

P11-S1裁定、修订和预检复核已完成；现已获实施安排，按T1起顺序执行。P12搜索首调用、P13归档/profile/R36/R38、P14详情与父项搜索、P15协作/草稿/全局搜索、P16守卫终态不关闭。M7标签完整恢复与旧版本DTO最终删除保持所属阶段。

M4§14、总体§9.4、输入handoff顶层open不变；M5–M8最终M4-closeout未创建。P10b已收尾，P11不是已完成阶段，不合并、不清理P11准备worktree。

## 7. 当前下一步

按通过预检的计划完成T1–T14，阶段完成后deep-review并修复局部问题，保留codex/m4-p11分支/worktree、暂不合并。架构级问题先提交已完成工作再上报。原停下依据和两方案代价保留在决策记录。

## 8. 实施起点与执行方法

2026-10-30重新核对：main/origin main仍为df5d3bbc，分支/远端为23f715b2，主目录和本worktree干净。仅在本worktree按锁文件安装依赖并强制构建10个共享包（全成功，无缓存）。实施phase进入M4/P11；未延长任何例外，21个P11旧URL exact key仍到M4/P12，另1个搜索key归P12–P15。

使用executing-plans/TDD逐任务执行。本次按02§3的方法调整：每Task主控分别核要求和质量，最终另派全新上下文整分支审者，并完成deep-review三视角；不将自审称为独立评审。原始证据、Task BASE、偏差裁定及共享接口预检留在本worktree `.superpowers/sdd/P11-web-detail-data/`；以后与主目录准备证据一起归档，当前不清理。产品每个检查点按计划严格串行F，准确产品SHA的CI与后续文档SHA分开记录。

### T1 薄服务（已实现，尚未接资源）

getByIdentifier和9个详情方法只调用生成客户端并unwrap，无转换/缓存/排序，无barrel新导出。两文件78项请求/拒绝测试通过，api-client原有5项通过，api/web types与关键词、格式、diff-check通过。T1初始行为红为59失败/15通过；M01将关系ID误换成目标Issue ID，1项失败/8项通过/46项按筛选未运行，精确恢复源码hash后全78项通过。原始文件以t1-/m01-命名。

主控要求/质量复核发现测试错误载荷最初用了非合同fields，已改为生成Problem.errors并给collection加生成类型，复测通过。M01的service仅接relationId、不接Issue ID，因此使用已知不同目标ID模拟误传；不是增加旧签名。T2–T4负责真实接线，当前不声称详情已完成。完整F按T4检查点执行。

### T2 生命周期与核心读取

root集中登记详情资源并响应权限、删除及退役；DetailResource复用Reconciled保存读取证明与确认，实体仍只在canonical。核心、父项、完整编号读取改用原生HTTP，冷项目先目录证明，附属读取受控await；归档页头、ID跳转及编号入口使用SessionSWR权限身份。编号入口最小签名接线属于本Task依赖，不扩后续详情UI。

资源、原生读取、既有fetch、实际入口、核心确认与弹窗回归9文件201项通过，web types通过。M02 getter越权、M03 scope不匹配、M04确认重放均有行为红，精确hash恢复后通过；M03三项失败中错误ID为断言红，另外两项为异常触发权威回查后超时，未把它们算成三个独立性质。原件t2-/m02-/m03-/m04-保留。

主控要求/质量自审发现逐ID嵌套排队会让等待父项的批次被后来单项抢先；新增反例确实失败，改为调用时同步预约全部ID的tail，独立ID仍并行、失败继续、退役后不执行；11项资源测试包含该修复。旧测试Fixture同步迁移未删除拒绝、旧读、克隆或输入断言。FakeNerve增加默认不开启的respond脚本以复用真实客户端。当前尚有未迁附属服务，完整F留在T4。

### T3 子项、关联与真实消费者

子项改用原生集合，仅存ID，状态分布从canonical与State.group派生；关联只存关系ID/type/目标ID/createdAt，目录及权限证明完成后才接收目标。批量与普通父项PATCH共享root队列，删除关系使用relationId，双方计数通过权威Detail回查。未知旧父项的批量回答没有旧版本证据，因此确认后将该范围已持有列表标待刷新；核心PATCH继续沿P10b确认协调，保留较新完整页面的证据，不额外猜计数或放宽守卫。

成功命令与补读错误分开：保留确认，界面显示只读重试；重试不重复POST、不重放已经被新GET删除的关系。并发重试按本次尝试身份归属错误。真实详情根在canonical待核期间仍保留重试入口；删除弹窗拒绝不关闭、成功只关闭一次，旧宿主完成不关闭新宿主；跨项目关联行使用源项目范围删除。已确认实体的编辑回调不再二次PATCH；测试明确模拟子编辑弹窗的确认回调合同，不冒充完整编辑页面E2E。

局部自审修复同步覆盖实际菜单、弹窗、子项展开和选择列表；选择列表的搜索拒绝可见且可重试，旧项目搜索晚答不覆盖新列表。沿同一错误路径提前完成T12删除弹窗的旧error.error清理、T10选择列表的受控读取；其余T10–T12合同仍待完成。资源错误/宿主lease是局部UI状态，权限事实仍只有IssueRootStore。删除旧relation service/types和3条已消失的URL exact例外，未延长其他例外。

18文件330项受影响回归通过（t3-regression-verified），共享包10项强制构建、web types、关键词75规则/42例外、变更文件lint零警告、格式与diff通过。M05b手加父计数得到9而非8；M06移除共享child排队导致提前PATCH；M07用targetId删除导致路径断言失败，三项均恢复原字节/hash，完整回归通过。M05首次未传权限证明，错误写入被既有守卫拦住，未计入有效变异；M06验证共享队列，不能把该变异说成单独验证了排序必要性。

证据限制保留：一次共享包重建与回归重叠导致导入失败，改回构建后串行回归；一次scope测试误以为Reconciled会抛出SessionChanged，核源码后按既有返回undefined合同修正，未声称修复了不存在的生产问题；未知旧父测试首次用了错误group_key，修正为原生key后才记录行为红。T3因真实消费者局部修复超出原行数估计，仍为已定数据接线范围，无新服务端/跨模块协议。阶段完整F和首次产品推送仍在T4，未合并。

### T4 链接、旧服务删除与首个完整门禁

链接集合改用原生IssueLink，nullable title与metadata.error按响应展示；命令共享工作项队列，确认前不改资源，确认后以权威Detail同步计数。成功补读失败保留确认并显示只读重试，弹窗拒绝保留输入，成功才关闭；旧宿主完成不关闭新宿主。删除IssueService、链接旧类型、六条旧URL exact例外及CycleStore三处无用服务声明，M6目录边界回归保留。

原生资源13项及真实消费者4项通过；R31/R32、fetch和弹窗等迁移回归6文件135项通过。M08提前修改缓存和M09把元数据failed当写失败均出现行为断言红，原字节/hash恢复后17项通过。最初红中旧Axios路径与原生路径不同是迁移边界证据；测试泛型类型错误单列，不计行为红。要求/质量自审及完整门禁又发现一个错误翻译键、两个已无读者的类型导出，均已修复；不新增规则豁免。web lint警告上限由217收紧为197，与删除旧实现后的真实数目一致。

首次F严格串行：t4-f1的gen-check/lint-go/test-go通过；lint-web发现翻译键后在t4-f2重跑通过，knip发现死导出后t4-f3的lint-web/knip/test-web/e2e均通过。前端局部修复不改服务端，复用前三项真实结果。web 177文件1,849项、共享包Vitest157项及api-client5项通过，18个test任务强制执行、0缓存；E2E 169项通过（含Firefox A4）。包级完整日志和哈希在t4-test-web-packages，七项raw/json保留准确命令、HEAD与dirty文件hash。首次验证脚本仅识别了未跟踪目录，最终F前已修正为逐文件hash，未补造早期回执。准确提交与分支CI另行记录；本地通过不代表main验收，未合并。

T4提交`5fd567c86a064db804558944b929d0f413eb1364`已用系统git推送，[检查点CI](https://github.com/open-nerve/NerveProject/actions/runs/40943202954)运行中。提交后受影响回归21文件349项通过；首次task-done误写领域边界路径只跑20文件347项，纠正为真实路径后完整通过，未把错误路径当测试成功。

### T5前的任务规模裁定

按spec§6已预留的局部拆分，原T5的评论、评论表情和工作项表情分别涉及三store、服务、真实控件及测试，合在一项超过约1500行估算。拆为T5评论及内嵌表情、T6工作项表情，原T6–T14顺延为T7–T15；完整F检查点相应为T4/T9/T13/T15。服务端/公共合同/阶段范围不变，无需重开架构裁定。旧工作项表情服务仅在T5期间保留，T6必须迁走最后消费者后删除；历史评审、回执和变异编号维持当时含义。设计/spec/plan已同步，阶段仍未合并。

### T5 评论与内嵌表情

评论服务六个方法接本代生成客户端；集合按完整页链原子接收并保存最后请求cursor，失败保留上次完整数据与错误。评论确认和内嵌表情共用父工作项队列，按原生ID去重及updated_at纳秒版本合并；移除表情只删当前用户同emoji，评论删除/重新完整读取后不会被旧表情回答复活。评论表情store仅做派生与委托，没有独立GET或第二份DTO缓存。旧评论/表情类型与三条评论旧URL例外删除，工作项表情按拆分留T6。

实际表单等待确认；拒绝保留输入，旧宿主不清新输入或提示，编辑使用服务器确认HTML；评论卡用actor_id及目录fallback，edited_at=null正确展示。活动区显示读取错误并提供只读重试。已有M5文件操作保持所属接口；确认评论后的上传状态跟进失败单独显示，不把它变成评论创建失败而重复发送。

三文件59项服务/store/界面测试与此前回归合计24文件408项通过；共享包10项强制构建、types、变更文件零lint警告、格式、关键词75规则/33例外通过，web警告上限197→192。M10实际请求方法从POST错误变GET被断言捕获（生成类型也不允许该GET，但本记录是运行时请求断言红）；M11a删掉其他用户表情、M12丢失读中的确认重放均行为红，原字节hash恢复后全回归通过。T5初次51红中两项新方法缺失及其派生失败不计行为证明；后续两项失败为fixture未提供当前用户，补用已有成员fixture构造完整User，不放松生产身份检查。三个UI问题与读取错误入口均有独立红绿；后续类型/静态修订不冒充新的行为bug。

T4准确SHA `5fd567c86a064db804558944b929d0f413eb1364`的CI40943202954 attempt1已server/web/e2e全绿，公开原件在主目录preparation/t4-checkpoint-ci-03；它只证明T4检查点，不能替代T5之后最终分支验证。当前未合并。

### T6 工作项表情

工作项表情服务三方法接生成客户端，集合复用DetailResource与root工作项队列；按原生id确认去重，删除仅当前用户同emoji，旧GET不能撤销已确认删除，退役/权限ABA不接晚答。界面改actor_id目录展示、原生错误码、只读重试和宿主跟进，权限丢失恢复后也释放当前控件忙碌状态。旧表情类型、无读者groupReactions工具及两条旧URL exact例外删除，未放宽任何守卫。

26文件440项受影响回归通过（t6-regression-restored）；共享包10项强制构建/0缓存、types、变更文件零警告/格式、关键词75规则31例外通过。M11b误删他人同emoji、M12b漏重放使删除复活均断言红，精确字节hash恢复后回归全绿；M12b是共享确认性质的追加覆盖，不虚增独立性质数量。实际界面三个问题及权限恢复后的忙碌状态各有红绿。issue.fetch和base.loading最后旧Axios夹具迁到FakeNerve，拒绝/输入保留/旧来源晚答等原断言保留；首次大回归9失败由base.loading仍回答旧路径导致，修夹具后全过。首次types为Promise联合推断问题，改async回调；误用不存在的make check-keywords无测试结论，随后真实node工具通过。主控要求与质量自审完成，无新架构发现；整分支终审仍待T15，未合并。

### T7 动态、增量与写后读取

动态服务/store改原生IssueActivityPage，完整页链原子接收，增量保留最后请求cursor、按ID覆盖重排；活动和评论共用activity-order的纳秒时间/id比较，合并展示增加kind避免key碰撞。actor_id为空显示站点名，其他执行者使用成员目录及缺失fallback；创建者与执行者展示合并，移除source_data、旧details/epoch/is_bot类型和最后动态旧地址例外。所有渲染分支显式传当前workspace/issue，nullable链接/编号不产生null地址，M5–M7对应字段仍保留展示边界。排序工具按职责命名activity-order，原计划timeline会命中现有守卫；只改命名，未加例外。

补齐设计§3.13既定的写后读取：核心、链接、评论、评论表情、工作项表情、关联、子项批量均等待已打开的活动/评论增量读取；没有加载过的附属集合不为列表操作额外全量读取。资源各自记录补读失败，命令成功保持成功，只读重试不重发写。核服务端MarkChanged后补上T5/T6评论写与工作项表情的工作项权威回查，复用canonical既有核验/列表传播；评论表情不更新工作项时间，但若前一评论命令的权威读取失败，仍须完成该待核读取，不能用成功读评论把错误抹掉。最后这个连续操作反例已独立红绿修复，没有第二份待核状态或权限观察器。

30文件492项回归通过（t7-regression-restored），其中七类命令的确认/拒绝/读失败及权威只读重试、连续操作共24项走真实FakeNerve；服务/分页/排序与实际UI均覆盖。T5/T6单资源重放和表单测试隔离新增的补读协作，以继续直接验证读期间确认重放，完整协作由新文件负责；未把这些隔离单测说成整链测试。共享包10项force/0缓存、types、47个变更TS文件零warning/格式、关键词75规则30例外通过，web/utils/types警告上限保持192/6/0。

M13从第一页错误重开、M14只追加不覆盖、M15丢纳秒三项均行为断言红，逐文件hash恢复后回归绿。真实UI执行者/创建/读错误5项、nullable地址4项和连续写后待核1项均有红绿。证据限制：首次实现误从api-client导入本地unwrap，纠正后原生89项通过；一次批量编辑创建展示产生语法错误，修正后实际UI10项通过，这些实现错误不计新的行为证明。要求与质量自审完成，无架构级新发现；T7因补齐原有写后协调扩大了实际调用方数量，但未扩大公共合同/阶段范围。整分支deep-review、完整F及准确最终CI仍在后续任务，未合并。


### T8 描述版本、确认恢复与 M7 类型边界

WorkItemVersionService改为本代ApiClient的list/get，版本列表与逐版本详情分别由DetailResource持有原生DTO及scope证明，列表按last_saved_at/id倒序、保留纳秒精度并按需继续加载。member资源同时核工作区与有效项目角色，guest列表/详情零HTTP；工作区管理员仍按有效项目管理员规则读取。两个M4入口使用IssueDescriptionVersions与本代store，SessionSWR带登录及权限身份；归档详情可查看历史但不能恢复。

共享组件只接id/ownerId/lastSavedAt与HTML等展示输入，不认识任一服务DTO；M7的精确旧DTO移到services/inbox/intake-description-version.ts，旧URL、无body、nullable与原错误原样保留。M7包装仍使用既有编辑器恢复回调，并未接通自身API，不能把共享弹窗等待回调算作M7服务端确认。删除共享旧类型和两条旧M4 URL例外；无生成字段补丁、无新依赖。

M4恢复走root普通description_html PATCH与既有共享队列，确认前保留弹窗/内容，403/422明确显示拒绝，成功才把服务端HTML送入编辑器且emitUpdate=false，避免第二次PATCH。初始空HTML是真实值，未读取不能用空段落冒充。关闭同步清除选择，无延时清理；快速切换旧详情不盖新项。主控要求/质量自审又修复排队期间及答复期间权限ABA、错项目恢复范围、缓存HTML仍在时错误提示被隐藏三类边界；各有行为红→绿，缺失成员显示稳定回退。

35文件538项受影响回归通过（t8-review-fixed），共享10项强制构建、web types、变更25文件lint零警告/格式、关键词75规则28例外通过；web lint上限192→191，types/i18n保持0。M16放开项目guest读取确实出现两次GET；M17提前关闭导致两项失败；M18取消版本身份隔离后旧HTML确实覆盖新选择。原字节/hash精确恢复；对应原件m16–m18与t8-*保留。首轮新store不存在、全详情fixture错误等待附属读取以及三处测试类型错误均单列，不算生产性质证明；修正fixture为fetchIssueCore后真实请求与断言全部通过。本Task按计划不跑F、不推送；T9下一检查点统一执行完整本地门禁与准确SHA的CI，阶段不合并。


### T9 原生个人草稿与发布确认

草稿服务六方法接生成API，store按工作区持有原生Draft/Page、独立loader和member读取证明；每个draft的保存/发布串行，排队前捕获本代权限。payload白名单保留未编辑合法字段，清空可空输入删除键、数组清空保留[]，顶层project_id区分省略与null。发布先确认最新完整PATCH，再无body POST；403/409/422保留已保存草稿和表单。成功IssueDetail复用root创建确认，更新canonical、项目/个人等已持列表与计数，旧GET晚到仍重放；不为编号刷新项目。

页面和实际表单同步消费Draft.payload，无项目目录仍显示草稿、编辑器和工作区权限删除入口；保存/删除拒绝保持输入及弹窗，确认回调无第二次PATCH。保存关闭期间禁重复操作；权限member ABA与宿主变更阻止旧答关闭/提示。草稿页只读重试清除当前SWR错误；M5附件跟进失败独立显示，不使已确认创建重发。草稿移除模块/迭代字段后，最后六个无读者下拉组件、旧service barrel、旧类型/常量和三条URL例外一并删除；M6领域未被接通或扩展。

局部决定及代价：

- total只取服务器：增删/发布使旧total未知；已读过列表时重读第一页，成功确认保持，即使补读失败也只重试GET。代价是已加载后续页回到第一页，需要再次加载；不能用局部页长度伪造总数。
- 草稿命令直接使用自身store，删除伪装成IssueActions的适配；通用拖拽可明确选择无动作，未知store仍报错，草稿拖拽零请求。表单的dirty字段工具改为泛型表单字段，member UI租约复用root权限身份，不增第二份权限事实。
- 无项目编辑器保留图片扩展，避免已有HTML丢图；上传/复制仅对缺项目显式拒绝并零HTTP，不假称M5已接通。M7精确旧版本DTO只去掉无读者export，形状不变。
- T9新增草稿/UI/服务测试795行；加上消费者迁移和彻底删除，实际diff超出初估。保持原子换型并分文件验证，未另拆一个无法独立编译的旧/新消费者过渡Task；不把估算当已满足的行数门禁。后续整审须完整覆盖这一较大范围。

主控要求/质量自审修复了权限ABA后旧表单跟进、只读重试遗留SWR错误、保存关闭过早退出以及拖拽入口遗漏，均有真实行为红绿。M19不等待PATCH、M20局部payload丢未编辑字段、M21只写canonical不更新持有列表、M22发布错误刷新项目均行为红且精确恢复hash；M20采用后补的m20-payload-red，其初版过宽变异仍保留但不代替完整payload性质。额外拒绝删除关窗变异也为红。

T9检查点本地F七项有效结果全部通过：t9-f1的gen-check/lint-go/test-go，后续仅前端修复后t9-f5的lint-web/knip/强制test-web/e2e。前端18任务0缓存：web195文件2109项、共享157项、api-client5项；各包原始日志及hash另存t9-f5-package-logs。E2E169项含两个Firefox A4全部通过。历史失败不删除：F1找到T8错误翻译键，F2要求keywords文件统一格式，F3找到删除后的7个死文件/1个无用export，F4发现草稿拖拽初始化；均根因修复后重跑相关前端门禁。keywords语义只删3条例外，整文件格式差异由现有oxfmt要求；当前75规则25例外，web警告上限191→158。

服务/store/UI聚焦首次失败包含尚未迁移的合同；订阅mock缺失、timer/helper误用、旧加载fixture、两个并发GET夹具与权威核验GET漏答各单列，不计生产性质证明。完整页面故事仍归P15；统一列宽归T13；阶段终审、最终F/CI仍待T15。当前T9准确提交CI另存实施证据，不合并。


### T10 有效成员提及、建议弹层与旧 JSON 收口

四个实际入口（lite-text、description-input、工作项表单描述、收集箱创建描述）统一经两种编辑器包装使用本代成员目录。项目建议取有效项目成员并排除工作区inactive；无项目取当前工作区有效成员。仅按display_name匹配，最多五条，不受成员页面筛选影响，也不把email带入建议。只读零HTTP；每次输入读取用本代成员服务，query/宿主/权限ABA/退役检查在返回前生效。TipTap保留初始扩展回调，因此hook提供稳定函数入口，实际读取跟随最近已提交的宿主范围，避免新项目继续调用旧闭包。

建议弹层在query变化时立即结束上轮请求，包括下一次300ms防抖窗口；旧成功/失败不能改变当前内容或loading，等待时Enter不能选旧条目。当前拒绝显示原码文案和GET重试，过期会话错误不提示。共同handler仅新增可选错误展示函数，不持有领域服务，M7自己的接口未被改变。自审补上另一个目录读者覆盖当前读取时的明确重试错误，不能把它伪装为SessionChanged而隐藏。

删除WorkspaceService.searchEntity、四模块实例、其独占请求/响应DTO和一条旧地址例外；弹层迁走最后一个lodash使用后，移除editor的lodash-es/@types/lodash-es声明并更新锁文件。TSearchEntities仍供已保存mention节点使用，普通工作项搜索合同仍归后续页面。删除共享description_json表单状态/提交字段和无读者JSONContent目录/barrel；剩余description_json只在M7专属精确旧版本DTO及其合同测试，不冒称全web清零。保留editor内部TipTap JSON文档。

43文件635项相关回归、editor7文件45项（含真实编辑器既有测试）、共享10项force/0缓存构建、types、变更16文件零warning/格式及关键词75规则24例外通过；knip首轮指出上述两项死依赖，移除后knip/共享强制构建/types/editor45项复验通过（t10-*-cleanup）；警告上限web158→154、editor64→59。M23放回inactive成员使两个实际成员目录断言失败，M24移除当前query保护使旧答覆盖新词；均逐文件hash恢复后全回归通过。初始hook9红是旧接口尚未迁移，不能算九种独立性质；建议弹层5红为真实行为。首次构建的测试ES2024 Promise.withResolvers与React act全局类型不适配ES2023，改本包局部deferred/显式声明后通过，没有改编译基线。误调用不存在的check-keywords脚本不构成验证，随后复用已记录的正式keywords命令通过。

主控要求及质量自审完成；不声称独立逐Task审核。T9准确87d1bcd7的CI40964469564当前server/web已过、E2E待结束；T10没有新push，不冒用T9 CI证明T10。继续T11，阶段deep-review与最终完整门禁仍待T15，不合并。


### T11：按需读取与会话缓存

| 当前调用 | await / catch / UI 落点 | 本次归属 |
|---|---|---|
| issue.store.fetchIssue / fetchIssueWithIdentifier 的核心、父项、状态及协作资源 | 核心先校验，所有辅助读取 await；当前拒绝 throw，旧权限转 SessionChanged；入口原生错误和只读重试 | store T2/T3已接；T11补三个页面与peek错误落点 |
| sub_issues.readChildren 的本项目状态与父计数权威读取 | 顺序 await；DetailResource.error；子项内容及父同级列表错误/重试 | store T3已闭合；T11补 siblings 会话/权限key |
| relation.directories 的跨项目状态/成员/标签 | 一次工作区状态GET；各目标项目members/labels，Promise.all等待；错误交回资源，不假空数组 | T11；子项合同仍只允许同项目，不恢复旧跨项目子项 |
| StateDropdown / MemberDropdown / IssueLabelSelect | 打开时await原生目录；当前错误可见、loader结束、GET重试；旧宿主/权限回答忽略 | T11，真实组件DOM测试，未只测通用hook |
| 详情标签select / 列表LabelDropdown | 同一读取状态合同；列表组件用observer跟随目录；失败时禁止借Enter创建 | T11；创建动作统一入口仍T13 |
| versions / workspace drafts | SessionSWR业务参数+member permission identity，原生列表错误、重试；版本恢复UI lease为member | T8/T9已闭合，T11补版本恢复的member lease |
| 归档页面及页头 | 单条详情沿issue.read含合法访客，SessionSWR含当前项目permission identity；原生错误与GET重试 | T11；不改变服务端合同与权限守卫 |
| use-workspace-issue-properties | 当前只由M7 workspace-notifications使用，cycles/modules未接合同 | 保持M7边界，不列为M4已迁移或加M4豁免 |
| ParentSelectRoot / detail与peek的update跟进 | 属于修改后续与弹窗owner合同，尚有吞失败/重复补读的旧逻辑 | T12宿主与T14修改矩阵继续处理，不以本表声称阶段已完成 |

T11读取入口的行为红、修复和回归已执行；验证边界与执行中的纠正见下文。


T11局部裁定纠正：曾把归档列表的member规则误套到单条详情并做未提交实验；核设计§3.4/§3.11、server GetIssue.Execute后撤回，root/canonical/versions store恢复本Task基线，未改服务端或权限守卫。单条归档的ID/browse/peek仍按issue.read，合法访客可以读取；只有归档列表按issue.archive。t11-archived-proof-red/entry-green及peek-guest-red反映错误假设，不作为产品安全证明或变异数量。t11-archive-contract-red用正确合同发现三处过度限制，修复后纳入回归。


T11最终验证：t11-web-final为直接未缓存Web全套 **198文件2144项通过**（没有skip）；t11-native-types/types-present均通过，最新source的静态为t11-native-static-fixed（28个TS/TSX文件0warning/0error、格式通过），knip通过，cap按实际153从154收紧。关键词75条/24例外无命中。此前t11-regression-corrected为47文件665项；随后新增按钮属性及两项真实键盘测试，最终全套已覆盖；task-done还会对该聚焦命令留下提交后回执。

M25在关联目录调用catch返回空数组，三类403测试均断言红；M26把siblings的permission key改常量，权限20→5→20的3次合法读取退回1次，断言红。m25/m26原字节SHA恢复核对，最终全套绿；不把归档错误假设的实验计入有效变异。

实际组件测试替换了最初仅三个展示叶mock的版本：当前状态、成员（项目/工作区）、三处标签全部运行真实下拉与RootStore/FakeNerve。自审发现默认状态按钮嵌套原生button，移除嵌套后又通过tabIndex测试发现Headless Combobox会强制-1；最终复用既有CustomSearchSelect的Popover触发器，DropdownButton为嵌入内容提供纯展示span，状态/成员只有一个原生触发器并能按Enter发目录GET。复用现有getButtonStyling，不改共享包、不新增依赖；移除无读者StateOption。成员SSR的旧测试曾把嵌套按钮视为合同，已改为检验唯一触发器的tabIndex及纯展示内容，实际DOM测试守住键盘行为。DateDropdown/R36仍属于原P13移交，本次未冒称全面修复所有下拉。

本Task的要求/质量复核由主控分两遍自审完成，没有冒充独立代理终审；完整fresh-context审者及deep-review仍在T15。局部裁定：每次打开目录读取当前源（代价是增加一次GET）；当前请求被另一读者覆盖而返回undefined会显示可重试错误，不伪造空目录。目录数据与权限事实没有第二份cache/观察者；旧scope结果不提示。ParentSelectRoot及写后跟进的owner仍按T12/T14收口，不能把本Task表述为整个P11已经交付。T9精确提交87d1bcd7的CI40964469564 attempt1三项最终全绿，t9-ci-04为回执；该结果不代替后续产品提交CI。


### T12：弹窗、浮层和工作区命令的实际宿主

T12从`8a57f52d`起实施。完成后仍不合并；T13–T15及整分支deep-review/fresh reviewer尚未完成。

| 状态、原初始值 | 全部写入入口 | 实际宿主与结束点 |
|---|---|---|
| create=false、crud.create/existing={toggle:false,parentIssueId:undefined} | sub-issues/quick-action-button；sub-issues/issues-list/list-item、relations/issue-list-item触发行编辑；widget-modals及两个content关闭 | 每个详情/peek的IssueDetailHost持有自己的IssueDetailModals；widget创建/选择、两content本地行编辑随owner key结束 |
| subIssues=null | sub-issues/quick-action-button、widget-modals | widget选择弹窗；scope/权限变化及卸载复位，ExistingIssuesListModal换目标/关闭重开也重建选择 |
| issueLink=false、issueLinkData=null | links/quick-action-button、issue-detail/links/link-item、create-update-link-modal及widget-modals | 当前宿主的链接弹窗；旧回调不清新编辑数据 |
| relation=null、relationKey=null | relations/quick-action-button、widget-modals | 当前宿主的关联选择弹窗；空relation真正回到null |
| parent=null | issue-detail/parent-select、parent-select-root | 当前宿主的父项选择；旧更新完成不启动新宿主的跟进读取/关闭 |
| delete=null | sub-issues/issues-list/list-item、relations/issue-list-item及各自content | 子项/关联行实际DeleteIssueModal；行局部payload随宿主key销毁 |
| peekIssue=undefined | use-issue-peek-overview-redirection、通知sidebar/notification-card/item、peek关闭回调 | 详情/项目列表/归档列表/profile/通知动态宿主均传hostScope；同引用+所有者检查，旧卸载/晚到删除、归档、恢复不关闭新选择 |
| command create/delete=false、allowedProjectIds=undefined | issues/header、project-issues空态、workspace/sidebar/quick-actions、power-k创建/工作项命令；WorkItemLevelModals关闭 | 原workspace provider内WorkItemLevelModals，workspace、projectId??identifier、具体identifier及permission identity换代/卸载释放拥有的两开关；首页/草稿/profile/browse创建能力保留 |
| peek局部delete/archive/duplicate/edit=false、两个content行payload | peek/header→quick-actions；子项/关联行回调 | 不新增全局状态；宿主子树换代清理。Escape同时检查局部确认框，不能关掉底层peek |

没有恢复P10a已删除的附件宿主或root归档开关。IssueDetail保留资源/服务、peek选择、展开偏好；新的局部UI状态不复制Issue/Project，也不增加权限观察者。useIssueDetailModals必须在真实宿主下使用，不提供全局降级。子组件即使经portal渲染仍继承该owner；详情和peek可以同时拥有不同弹窗，撤下“只要有peek就禁止详情渲染widget modal”的全局开关。

有效红绿：t12-host-contract-red五项宿主断言失败；t12-command-red五项工作区命令断言失败（含删除拒绝不能吞为成功）；t12-selection-red锁定换目标仍保留隐形旧选择；t12-route-epoch-red锁定route改变残留peek与StrictMode恢复旧followup；t12-peek-actions/reopen-red锁定真实View+Header旧删除/恢复/同issue重开及Escape。触发器真实DOM还发现peek布局嵌套button和父项button包含链接/移除交互，分别用t12-peek-button/parent-buttons-red确认，再改为现有共享select的纯展示内容及并列原生按钮。没有放宽守卫。

测试边界：detail-host和work-item-level测试运行实际宿主，展示叶被probe/props收集器替换；peek/host-actions运行真实View与Header，仅隔离领域命令的延迟、页面内容和菜单展示；links/relations仍用真实store/FakeNerve并接入实际宿主。它们证明宿主归属，不替代T14完整命令矩阵或P13/P14浏览器Back故事。最初一次夹具漏fake timers、一轮未应答附属请求中断、一次命令拼成check-types以及两处mock模块缺新context出口，都作为失败保留，不算产品红证据。

最终当前源：t12-web-wired **201文件2171项通过**；恢复变异后t12-regression-restored **50文件695项通过**，包含计划指定clone/loading；t12-types-reviewed、t12-static-reviewed（37个TS/TSX，0 warning/error）、t12-knip、t12-keywords（75规则/24例外）通过。删除四项旧告警后web cap由153收紧到149。M27故意全局retire其它宿主，在旧宿主卸载时把新链接弹窗清掉，断言红；M28漏掉scope换代依赖，只在卸载结束旧状态，换issue断言红。两项均恢复原字节SHA，见t12-mutations-restored.json；有效变异没有编译错误或超时冒充。

G由主控分别完成要求及质量自审，未声称独立逐Task评审。局部裁定：采用显式UI context替代全局字段；用root+scope的独立React宿主身份结束所有子树局部表单，代价是权限水位改变会清空未提交输入；peek真正卸载的选择在同次layout收尾后微任务释放，防止StrictMode误清，跨route仍同步结束旧选择。通知只迁移M4嵌入宿主的显示合同，M7自身未接通接口不在本段变更。T14继续核写后错误传播/重复读取和完整命令矩阵，T15才做阶段终审。

### T13：标签确认、原生拒绝与统一编号宽度

从`b2412db4`起实施。五个标签创建入口/回调归入`useLabelChanges().create`，新增`label-create-direct`守卫；实际表单、详情与行下拉对重名显示name字段错误，拒绝保留输入，不制造Label或提前选中。成功只跟进本会话/宿主，重复Enter在途只发一次。无项目的标签展示只派生当前工作区已持且可见的labelMap，不请求不存在的工作区标签接口，不预取所有项目；完整能力仍归M7。base.tsx两处旧错误取值已在T9清理，本Task补实际Form创建/修改的原生拒绝测试，并以M31验证退化会失败，没有假称再次删除。

List局部数值注册表统一全列宽度，IssueBlockRoot在RenderIfVisible外登记合法完整编号，递归展开子项共用；跨项目按每个实际编号取最大，不把最长前缀和最大序号拼成假编号。普通分页、筛选缩小、删除允许保留已达到宽度；login/workspace/list kind/project/profile user/view/权限identity变化重算，独立root/并存布局各自拥有注册表。未知项目不输出undefined编号，key隐藏为0，空集合不保留可见宽度；全程零编号专用HTTP。provider位于滚动容器内部，切scope不破坏滚动监听；StrictMode及旧注册清理不能影响新owner。

| 证据 | 当前结果/限制 |
|---|---|
| 标签行为 | t13-labels-red：7项真实失败；t13-labels-environment：3文件54项通过（含既有M3统一入口和T11读取拒绝）。初次改写脚本中途退出/ResizeObserver缺失单列为测试环境问题 |
| 宽度与错误 | t13-width-red：5项宽度行为失败；最终12项宽度测试覆盖两种布局、视口外、分页、展开子项、不同前缀、scope/权限ABA、未知目录、隐藏key、StrictMode；t13-contracts-ready：5文件71项通过 |
| 有效变异 | M29直接create被新守卫拒绝；M30固定一位序号令统一列宽断言红，取消permission identity令ABA后错误保留49px而非35px；M31恢复error.error令两个表单错误显示unknown而非forbidden。原字节SHA恢复，t13-label-mutation-restored / t13-width-error-mutations-restored保留；memo依赖与格式的后续修改由最终全套回归桥接 |
| 本地F | t13-f-gen-check/lint-go通过；test-go其余包通过，jobs启动Ryuk超时未进入测试，t13-jobs-retry按同一源码单包25.8秒通过；t13-f-continued-lint-web为54任务0缓存通过，knip通过；test-web为18任务0缓存，Web203文件2202项、editor7文件45项通过，副日志及hash见t13-suite-logs |
| E2E适配 | 首轮168通过/1失败：P1仍断言Lead为嵌套button，而T11已修为单一原生按钮。只更新这处过期DOM断言，保留Tab/焦点/Escape及所有请求和控制台门禁；P1三故事通过，t13-e2e-restored完整169通过（含2Firefox），不是W11偶发 |
| 静态 | t13-e2e-types、t13-static-final：20个TS/TSX零warning/错误且格式正确；web cap149→147，关键词76规则24例外，无新增依赖或豁免 |

局部裁定：P1故事的DOM预期随T11真实控件结构同步，不保留非法嵌套来满足旧测试；代价是测试不再约束已删除的包装元素，单一语义按钮/Tab顺序/焦点恢复得到更直接约束。草稿/列表宽度没有改变项目公开字段或后端。getUserProfileIssues/fetchParentStats已无实现；peek自身的`isArchiveIssueModalOpen`是仍在使用的局部状态，不能声称整个web该名字零命中。

G按要求与质量两遍主控自审，未冒称独立逐Task审者。T14继续处理根详情/peek/标签更新的拒绝传播、父项重复补读和加载收尾、展开子项的受控读取、标题/描述实际保存状态及完整命令矩阵；阶段A1–A10与T15整分支审查仍开放。本检查点推送后的准确SHA/CI回执另存忽略证据目录，不能用T9绿色CI代替。


### T14 修改矩阵与调用方收尾

T13准确提交`100565d27bc7d842a5693d877b84f42ef73e544c`的[CI 40982066769](https://github.com/open-nerve/NerveProject/actions/runs/40982066769)已全绿（attempt 1，server/web/e2e）；这份结论不用于后续T14产品。

详情/peek命令现在传播已报告的拒绝；属性事件明确结束Promise，恢复失败保留peek。父项操作删除重复详情/子项补读，移除失败释放自己取得的loader。列表展开副作用离开setState updater，失败展示原生错误并只读重试。标签创建成功但附加失败保留输入，不伪称附加成功。

标题与描述以本地输入修订和实际命令确认管理自动保存：拒绝保留输入/显式重试，旧确认不覆盖新输入，撤回原值仍排入必要的后续写，不在卸载时重发已尝试的写。标题/描述各自状态汇总，任何失败都不会被另一字段的成功或计时器掩盖；M7仅跟进共享展示props，不改变旧领域服务。提及菜单增加纯UI生命周期信号，scope变更清空已显示建议、关闭菜单，选择时同步阻止旧scope；源hook只读取既有root权限identity。

矩阵复核又发现排队起点权限水位丢失，及目录等待/拖动队列/204删除回包的同类缺口；已在现有IssueRootStore统一加固，资源入口传源范围，涉及多个工作项时合并其已有项目证明。没有第二套权限事实。不同root、另一工作区权限变化不误伤当前队列。

证据目录仍为本worktree `.superpowers/sdd/P11-web-detail-data/`：`t14-command-red` 8行为失败→9绿；`t14-autosave-red` 7行为失败及1未处理拒绝→12项（含撤回/卸载补充）绿；`t14-editor-status-red` 1行为失败→13项相关绿；`t14-parent-children-red` 3行为失败及1未处理拒绝→24绿；`t14-mention-scope-red` 1行为失败→6绿，源hook14绿。`t14-queue-aba-restored-red` 在确认恢复权限后18项失败→完整矩阵57绿；先前fixture自动答复错误的运行不算ABA证明。`t14-queue-scope-drag-red` 真实多发PATCH，`t14-catalog-recovered-red` 在新读恢复实体后真实多发PATCH，`t14-core-permission-red` 的DELETE旧回包确认错误，均在`t14-root-guards-green`的170项及随后回归通过。错误类型/类型检查/fixture错误单列，不充当行为证据。

最终定向`t14-matrix-final`：23文件440项通过，涵盖全部detail资源、核心多副本、拖动及本Task真实消费者；`t14-types-final`通过。静态53文件零告警，原web警告数147→146并收紧cap。共享包10任务强制构建通过；最终F和整分支审核留T15。无新增依赖，无守卫放宽。

#### 完整修改矩阵

下表每一语义分别审阅；“公共矩阵”指`issue-details/detail-mutations.test.ts`的18类命令，各有待确认/403不改，以及退役/权限ABA排队不发送。服务逐方法403/404/409/422测试仅证明原生Problem原样传播，不代表每个端点都会产生全部业务码；无body的删除没有表单422语义。业务409还包括归档工作项/项目，链接冲突为`issue.link_exists`，表情冲突为`issue.reaction_exists`，草稿发布为`issue.draft_changed`。

| 修改语义 | 确认前/拒绝 | 退役/权限/旧读 | 确认及补读失败 |
|---|---|---|---|
| 属性PATCH | issue-changes六写矩阵；实际表单403/422 | issue-changes权限/晚GET、公共队列入口 | canonical+项目/归档/profile；目录失败待核 |
| 描述PATCH/版本恢复 | 公共矩阵description；autosave/versions拒绝保留 | 公共矩阵、旧编辑确认不盖新值 | 原生Detail；版本等待确认后关闭，两个字段各自状态 |
| 父项PATCH | 公共矩阵parent | 排队/ABA；旧父未知不猜计数 | detail-changes核旧/新父权威读；真实父项UI不重读 |
| 批量子项 | 公共矩阵subAdd | 多ID预约、跨资源交错、旧父未知stale | 原生回答子项；两端计数待核不回滚 |
| 移除子项 | 公共矩阵subRemove | 与批量同队列，ABA不发送 | nullable parent确认，loader失败释放 |
| 删除子树 | 公共矩阵subDelete | issue-changes未知后代/旧GET/verify；旧权限204不确认 | root墓碑与父计数；无body删除不捏造422字段 |
| 恢复归档 | 公共矩阵restore；详情/peek拒绝 | 拖动/普通队列起点水位，旧宿主不关新peek | 六写多副本协调 |
| 新增关联 | 公共矩阵relationAdd | source/target目录证明、跨项目ABA | relationId/native目标；补读失败只读重试 |
| 删除关联 | 公共矩阵relationRemove | 旧集合重放、两端scope | DELETE关系ID；拒绝不动，后续GET不重写 |
| 新增链接 | 公共矩阵linkAdd | 旧GET重放/ABA | native链接；元数据failed可确认，计数权威 |
| 修改链接 | 公共矩阵linkEdit | 同上及并存活动增量 | native nullable title；补读失败保留成功 |
| 删除链接 | 公共矩阵linkRemove | 同上及scope失效 | 204后按ID移除，计数不本地减 |
| 新增评论 | 公共矩阵commentAdd | 全页原子/晚GET/纳秒重排 | native评论及内嵌表情；权威补读失败保留 |
| 编辑评论 | 公共矩阵commentEdit | 新版本不被旧页覆盖 | comment_html原生确认；最后请求cursor增量 |
| 删除评论 | 公共矩阵commentRemove | 旧页重放，已删除不被晚表情复活 | 204删除评论及自身表情，无独立表情GET |
| 新增工作项表情 | 公共矩阵reactionAdd | actor维度、同issue队列/ABA | 幂等ID；只读补取不重POST |
| 删除工作项表情 | 公共矩阵reactionRemove | 同上 | 只移除当前actor同reaction，不动别人 |
| 新增评论表情 | 公共矩阵commentReactionAdd | scope/队列/晚回包 | 仅comment.reactions；不清除之前尚未核实的核心错误 |
| 删除评论表情 | 公共矩阵commentReactionRemove | 同上 | 204只移除本人，不建伪表情集合 |
| 草稿新增/更新/删除 | workspace-draft store矩阵，403/422保留 | 独立draft队列、权限、全页晚答 | 原生Draft/204；total待刷新而非猜数 |
| 草稿发布 | workspace-draft publish/真实弹窗矩阵 | 退役/成员失效，单次提交 | PATCH完整payload→无bodyPOST→root创建确认；后续失败不能重发 |
| 标签创建/附加 | use-label-changes、issue-properties、form错误矩阵 | session/host捕获；双Enter只发一次 | nativeLabel成功才选择；附加拒绝不回滚已建标签 |

#### 过渡清单

`t14-transition-audit`逐个对照基线证明9个指定模块实例消失：两个版本消费者（main-content、peek issue-detail）、workspace_draft service模块实例、四个提及入口（lite-text、description-input、issue-modal description-editor、inbox create issue-description）、archives header及issueId page两处IssueService。根按代构造原生薄服务是保留的正确归属，不要求全库`new Service`为零。

M4指定目录的is_bot/description_json/comment_json均零命中，旧IssueService文件删除，keywords仍76规则/24精确例外。剩余M4模块实例为ExistingIssuesListModal（P12）、父项搜索和M7查重选择（P14）、命令面板搜索（P15）；projectIssuesSearch旧URL例外到P16前强制到期，P14若无读者即删。global-view.store的WorkspaceService属于M7自身待接通。M5资产、M6迭代/模块、M7收集箱/视图保留各自领域边界；copy-hook现有例外按各closeout处理，不冒称P11已满足P15/整M4零残留门禁。删除无读者label defaultOptions及其三个构造调用、peek storeType、输入key/状态冗余prop。


### T15 整分支审查与修复（最终验证中）

使用deep-review对阶段基线df5d3bbc至ad0dc1ec逐批审逻辑、集成和覆盖；全新只读审者核283个变更（271全文、6完整结构+变更、6历史文档相关章节+完整diff），另9依赖全文；逐路径、原SHA/行数与阅读方式在`final-reviewer-coverage.json`。不把部分历史正文或结构解析称为逐字全文，不声称审了整仓。主控核真实调用链并自行完成一次局部修复；未重新派第二审者。

固定审核HEAD的6 Major/3 Minor包括父项提前关闭、子项共享加载残留、祖先卸载阻止最后autosave、版本恢复未重置autosave、两处日期拆写、标签旧集合覆盖、普通表单旧scope跟进、四个翻译key和三类只读标签透传。普通表单问题按可能重置新表单/触发旧资产请求的影响升为Major；无Critical或架构阻断。所有发现均纳入局部修复，不放宽架构、权限或关键词守卫。

`t15-regression-ready`的33文件524项、`t15-types-ready`已通过；静态/完整F/最终准确SHA的CI仍待完成，不能据此声称P11已交付。有效RED与夹具失败的区别、修复权衡和全部回执路径保存在台账；阶段A1–A10继续开放，最终整审报告和验收映射在门禁完成后收口。
