# M4/P4 描述与协作：实施计划（暂停草案）

> **执行说明：** A1 获裁定且计划补齐、通过预检后，使用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans` 逐任务执行，以复选框（`- [ ]`）记录进度。

**目标：** 完整交付M4设计§12/P4的描述、清洗、提及、版本、评论、两类表情、动态分页及对应测试/交接。

**架构：** 所有写复用Locks.run，跨模块仅经消费方小端口，sqlc拥有模块SQL，bootstrap唯一组合根。列表使用同一只读快照。清洗的层放置存在A1冲突，等待负责人裁定，本稿不能直接执行。

**技术栈：** 当前仓库锁定的Go 1.27/toolchain go1.27.1、pgx v5.11.0、sqlc v1.31.1、oapi-codegen v2.8.0、PostgreSQL 18.6、Node 24、pnpm 11.10.0、TipTap/vitest/Playwright；仅计划增加已批准bluemonday及设计点名的传递依赖，未解析或安装其版本。

**规格：** [P4规格](../specs/P4-issue-collaboration.md)；上级[M4设计](../M4-design.md)；代码基线`56d575fb8ff6575880d1edef65219728a81f2b12`。

## 全局约束

- **状态：A1待负责人裁定，暂停。** 设计3.7/6.2/P4 Task1让domain直接依赖bluemonday；`server/internal/archtest/{rules,purity}_test.go`禁止domain/app/shared直接或间接第三方依赖。已知identity的PasswordHasher/argon2模式可供裁定参考，不据此自行授权架构调整。
- 推荐候选：domain保留RichText/内容标识纯值，app声明HTMLSanitizer小端口，issue/adapter/content实现bluemonday及x/net/html，module.New组合；保留现有架构守卫。该候选未获批准，不建立确定生产签名。备选放宽纯层守卫不推荐。
- 本稿保存暂停前已完成的任务划分；未细化的Files/Interfaces/TDD命令待裁定恢复后补齐、独立预检，不把任务纲要当可直接执行计划。没有原型、逐Task全量重放、S3、变异或P4门禁通过记录。
- 工作树`.codex/worktress/codex/m4-p4`、分支`codex/m4-p4`；主目录保持main。不用stash/clean/reset --hard，不动plane/refer、不在其中git，不操作已有容器，不全局安装。
- 启动提交已把keywords phase改为M4/P4，不重复。未来提交按本次明确指令使用英文`feat(M4/P4): ...`；push用`/usr/bin/git`。草案作者不提交，由主任务统一核对并保存准备工作。
- TDD、关键性质实际变异、独立预检、逐Task独立评审保留；Claude整分支终审由负责人安排。接口/文件/测试/命令驱动的紧凑plan替代原型和代码逐字重放（交接02§3允许），不得伪造通过数字、哈希、生成物或杀死率。
- 预检404/403不取事务/锁/闸门；Locks.run下再读/再判，访客404先于角色/作者403；项目归档→工作项归档→无改动→依赖检查→时钟→写。新写声明503，只有bootstrap/gate映射server_busy。
- 全局表序：issues→assignees→labels→relations→links→subscribers→mentions→comments→issue_reactions→comment_reactions→activities→description_versions。同次改动同一锁下时刻、同一写者；业务描述里的“先评论后updated_at”不意味着SQL逆序。
- 角色按集合、可见性复用HiddenFrom；模块不互相import、不跨模块JOIN、SQL只经sqlc、模块SQL不写SET；端口错误原样传播，不留无消费者代码。新文件约400行以内，契约/生成物沿既有例外。
- 新错误码同Task更新PROBLEM_MESSAGES和en/zh-CN errors.json并跑make test-web；每个新API当Task加入矩阵、HTTP双向契约、锁形状/闸门测试。
- 新表随首个可用API建出，同时grants/sqlc、删除连带、目录断言及furnishIssueRows；不新增临时目录豁免。迁移顺序00021→00022→00023→00024–25保持原编号。
- 实现阶段生成物只经make gen/gen-go，提交后make gen-check；server改动跑lint-go/test，契约/web/e2e改动加lint-web/knip/test-web，迁移/fixture/故事跑e2e；一次只跑一套容器套件。当前暂停不运行这些步骤。
- 测试复用issueWorld、activityBy、workItemStamps、listIn、matrixRow.prepare、itemWrite、inBothOrdersAt、pgtest.RowWaits；e2e复用activityRow/byId/madeBy/tablesUnder/keysBelow。等待有条件、有期限，不sleep同步；逆id、同时间、两方向、两层子树、邻居/已删行让数据区分性质。

## 评审重点

1. 自定义image-component[src]不受bluemonday固定标签URL策略自动覆盖；style属性放行不等于CSS安全。T1需要真实往返及恶意src/CSS变异。
2. 管理员兼项目访客对自己的工作项可删别人评论，别人的工作项仍404；关注按被提及者自己的OwnOnly算。T3/T4/T14。
3. 动态合并“最新一行”与版本600秒滑动窗口是两套规则；同时标题/描述、不同actor、同时间逆id、600秒端点必须区分。T8/T9/T13。
4. 历史目录忽略deleted_at但必须保留is_active；过滤在LIMIT前，隐藏引用不能影响页长/cursor。T11/T12。
5. 新表与嵌入表情扩展删除图；邻居评论/子树、旧删除戳、目录与查询同快照均需要反例。T2–6/T9/T14。

---

## 文件结构与16个Task纲要

以下路径均相对于仓库根目录。I表示`server/internal/modules/issue`，P表示`server/internal/modules/project`，B表示`server/internal/bootstrap`，只为表格缩写。逐任务精确Files、签名和红绿命令尚未完整展开，A1恢复后补齐，不能把它们标为已经预检。

| Task | 可独立评审交付与主文件 | 依赖与必须证明的测试 |
|---|---|---|
| 1 | S3清洗；`api/testdata/issue-richtext.json`、`web/packages/editor/src/sanitization-roundtrip.test.ts`；生产实现放置A1未决 | A1；真实CoreEditorExtensions每种保留节点/mark输出，API无块id列表/标题/嵌套mark重序列化，href/checked内容变化，危险URL/on*/CSS/custom-src去掉；无全局可变policy |
| 2 | `shared/page.go`及identity迁移；`platform/postgres/tx.go`的ReadOnlySnapshot；`I/domain/collaboration_cursor.go` | PageSize默认50、1–100；三种严格两元素载荷/UTC/UUID；ReadOnlySnapshot与LockTimeout任意Option顺序仍同时成立，真实库只读/同快照/配置不泄漏 |
| 3 | 评论创建/列表+00021；`I/app/comments.go`、`I/adapter/{postgres,http}/comments.go`；现有ProjectMembers窄扩展，P的issue_ports及B转换 | T1/T2；清洗后空422、mention/image-only、作者和可见被提及者关注；评论不存mentions不写动态；分页同刻逆id/篡改；迁移/删除/furnish一起交付 |
| 4 | 作者编辑/管理员删除；`I/app/comment_writes.go`与`lock.go`同路径检查；HTTP与存储 | T3；作者预检不占gate，锁下重读；404先于not_author；管理员不能编辑别人；无改动不改edited_at但归档仍拒绝 |
| 5 | 工作项表情+00022；`I/domain/reaction.go`、`I/app/reactions.go`、两个adapter及SQL | T3；码位正则/128长度、部分唯一、自己的重复409、只删自己404、MarkChanged与表情同刻、无动态；目录删除/furnish |
| 6 | 评论表情+00023；`I/app/comment_reactions.go`、adapter/SQL；评论内嵌及删除连带 | T4/T5；不碰issue/comment审计；页内批读不N+1、不能混入页外或软删表情；删除评论连带全部actor表情但邻居不动；目录/furnish |
| 7 | 描述纯模型与现有两列读写；`I/domain/{description,issue,patch}.go`、`I/adapter/postgres/description.go` | T1；不提前公开HTTP；500000字符、默认p、skip存在性（false也受字段组合限制）、内容标识四维；读映射一致、只写两列保留审计；T9为实际消费者 |
| 8 | 描述动态合并+版本纯决策；`I/domain/{activity,description_version}.go`、已有activity表查询/合并 | T7；最新活行是同actor description才合并、old/id保持；标题先写会阻断合并；版本600秒含端点/滑动/不同actor/未来时间不合并；T9消费 |
| 9 | 描述API连同提及和版本原子写+00024–25；`I/app/{description,mentions,create_issue,update_issue}.go`、adapter/SQL、HTTP/apitest | T3/T6/T7/T8；默认创建初版、有效成员mentions集合同步、隐藏guest不关注、状态patch不删mentions；skip同标识不改issue审计/动态/版本；失败全回滚；IssueDetail恰好多一字段且全部仍seen；两表目录/furnish |
| 10 | 版本列表/详情；`I/app/description_versions.go`、adapter/SQL、actions/rules | T2/T9；降序saved_at/id、自己的guest403/隐藏404、adminGuest边界、篡改cursor；详情自身404，列表不带HTML，无专用restore接口 |
| 11 | 历史项目目录；P的issue_ports/queries、B转换、I消费接口 | T2；同workspace/actor/is_active，忽略项目/成员deleted_at和归档；OwnOnly按项目guest标志；端口错不当空目录；T12消费，P5再补普通目录 |
| 12 | 动态分页；`I/app/activities.go`、adapter/SQL | T8/T11；parent及4种relation字段old/new任一不可见整行消失；软删issue/已删项目历史保留、物理清理略去；其他类型identifier不可错当issue；SQL过滤在limit前，cursor来自可见行 |
| 13 | 交错14/15、时钟单调；`B/issue_collaboration_interleaving_test.go`及clock测试 | T9/T12；评论/删除两序，连续描述同actor合并/不同actor分版；先探测等待再推进clock；-race -count=5，无死锁；clock移锁前变异失败 |
| 14 | 跨操作完整矩阵/整程序补证；`B/permission_matrix_collaboration*_test.go`，既有issueWrites/gate/locks帮手 | T3–13；所有13个新操作，作者/别人/adminGuest自己与别人，归档两交叉，隐藏逐值同不存在，全部写取锁/gate/连接/回滚；实际安全变异至少组合层补证 |
| 15 | I11/I12/I13新接口故事，I1/I2/I15/I17增长；`e2e/stories/issue`与fixtures/assert | T14；全部前置经API；600秒边界用组合固定clock，故事不睡10分钟/SQL改时间；I17五表两层子树与邻居；S3/skip/过滤/删除实际故事变异 |
| 16 | 设计3.23/P4、差异、五份handoff及review | T15及真实终审证据；状态open留到M4收尾；P5复用快照、P6事件、P9保留/清理、P11/14/15前端、M5图片投影移交；不伪造验收勾选 |

## 暂定接口决策（待裁定后统一自检）

- `shared.PageSize(limit *int) (int,error)`，`DefaultPageSize=50`、`MaxPageSize=100`；从identity移动，不留副本。`postgres.ReadOnlySnapshot() Option`保留设计名，不能让LockTimeout的BeginQuery覆盖隔离选项；issue.Deps增加ReadTx，现有写Tx不改语义。
- 现有ProjectMembers职责内增加`MentionMembers(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) ([]MemberVisibility,error)`；值含ID/Role/OwnOnly，project同一批读guest标志、bootstrap转换、issue用HiddenFrom。不新增重叠顶层目录，不让ProjectGate.Lock返回guest标志；当前只是设计候选，未实证。
- 子资源相对权限建议通过Locks.run的可选`check(context.Context,domain.Issue,shared.Grant) error`在预检和锁下各执行，统一seeAs/notFound源头映射；每次重读评论确认所属issue，不能另建事务/授权路径。
- 三个分页Execute统一`(ctx context.Context, issueID uuid.UUID, limit *int, cursor *string)`返回对应Page，`Data`空数组、`NextCursor string`空映射null；评论/动态载荷各自`[created_at,id]`，版本`[last_saved_at,id]`，复用shared封套。
- 描述的清洗端口、构造与文件位置全部受A1裁定；本稿不确定domain.NewSanitizer等违反现有守卫的生产签名。纯值RichText/ContentIdentity承载HTML/Text、顺序提及UUID、链接href、checked序列，集合去重只用于关系写入。
- 详细行为、错误码/路径、版本窗口、动态合并、HTTP字段和交接见spec§2/3；这些是草案，不能代替恢复后的完整接口签名自检。

## 每个Task恢复后必须补齐的执行结构

- [ ] 列出准确Create/Modify/Test文件和消费/产出的Go签名，引用上游Interfaces，不另创近义类型。
- [ ] 写明确测试名与关键断言（值取自spec），运行定位命令确认红；若已有实现已绿，用可编译性质变异证明测试有效，不伪造红记录。
- [ ] 实现最小改动；按Task适用范围生成、测试、lint；记录真实输出。
- [ ] 实际变异并核对是业务断言失败，恢复绿；安全/锁不能仅凭单元假实现声称证明，存储/组合/故事至少相应补证。
- [ ] 独立评审、局部裁定/修复、提交；生成物提交后gen-check；更新台账但不写未知hash。

## A1恢复步骤与自检状态

1. 等负责人裁定清洗放置；推荐端口/adapter方案需同步设计3.7/6.2/P4原Task1和本spec文件表/签名，不能靠顶部一句阻塞掩盖下面的确定错误路径。
2. 根据实际裁定补齐16Task逐步Files/Interfaces/测试断言/Run与预期；保留S3先行，别把API字段提前到提及/版本完整之前。
3. 自检覆盖、类型一致性、表序/拒绝序、游标/快照、新表迁移与fixture；独立预检重推HEAD全部写的锁图并抽查变异计划。
4. 重新评估T3/T9任务大小：短文档不代表实现小；修复余量5–10%。若无法约16Task且单Task约1500行plan内评审，按D2先重切，仍超则拆P4a协作/P4b描述与动态并更新设计15/16，不能缩scope。
5. 通过预检后才进入实现；Claude终审由负责人安排，review记录真实结果后再谈合并/main CI/清理。

当前自检状态：已读设计/交接/核心代码，已完成范围与任务映射；未完成可执行命令逐条核对、接口全量一致性检查、独立预检、原型或验证。本次按架构问题纪律保存草案后停止。旧plan行数的只读实测在spec附录A，本稿不冒称门禁通过。
