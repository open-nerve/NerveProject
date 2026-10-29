# M4/P11 详情数据层：预检与实施记录

日期：2026-10-30。**当前：文档预检通过，负责人已授权实施，T1–T4已实现，继续T5。** 负责人已裁定P11-S1采用A，设计/spec/plan/交接同步后完成复核。[spec](../specs/P11-web-detail-data.md)、[plan](../plans/P11-web-detail-data.md)、[deep-review记录](../../../reviews/re-14-M4P11预检审核.md)、[已裁定记录](P11-project-number-width-decision.md)。产品验收仍开放；准备阶段证据保持原义，实施证据见§8。

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
