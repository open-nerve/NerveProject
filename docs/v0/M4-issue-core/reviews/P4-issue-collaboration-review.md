# M4/P4 描述与协作：评审与裁定记录

| 项 | 内容 |
|---|---|
| Phase | M4/P4 `issue-collaboration` |
| 日期 | 2026-10-25 |
| 状态 | **进行中**：A1 已由负责人批准（2026-10-25），按适配器方案继续；不是阶段验收或 Claude 终审通过记录 |
| spec / plan | [spec](../specs/P4-issue-collaboration.md) / [plan](../plans/P4-issue-collaboration.md) |
| 分支 | `codex/m4-p4` |
| main 基线 | `56d575fb8ff6575880d1edef65219728a81f2b12` |
| 工作树 | `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m4-p4`；主目录保持 `main` |

## 1. 评审方式

- 按[交接入口](../../claude_to_gpt/README.md)的顺序读取 01、03、02、04、05、06，核对 M4 设计的 P4–收尾、完成标准、进度表，以及 P3 的 spec、plan、review 样板和此前交给 P4 的事项。
- 依交接 02 §3 调整计划产出方式：保留文件、接口、验收、执行命令与关键性质的变异计划，不把尚未运行的原型或逐块重放写成已验证。架构子任务负责 spec/plan 草案，主任务核对既有代码与基线。暂停时保存了 spec 和 16 个任务纲要（`573e8eb6`）；负责人批准 A1 后恢复计划细化与预检。
- 在准备 Task 1 时发现 A1，曾停止进入预检和产品实现；负责人已批准推荐方案，现恢复推进。T1 已完成独立 Task 评审、F1 修正与定向复核，T2–T9 独立评审通过，T9两项Minor交T10闭合；后续 Task 与 Claude 整分支终审待执行。下表保留起步基线和架构放置探针，不与 T1 结果混淆。

## 2. 验收标准核对

### 2.1 已运行的检查

| 检查 | 实际结果 | 证据范围 |
|---|---|---|
| `pnpm install --frozen-lockfile` | 退出 0；锁文件未变 | 工作树安装；没有全局安装或新增依赖 |
| `make test` | 退出 0；server 和 server/tools 的测试通过 | P3 起步基线；日志 `/tmp/nerve-m4-p4-baseline-test.log` |
| `make test-web` | 退出 0；17/17 个任务成功，全部命中 Turbo 缓存 | 基线缓存结果；日志 `/tmp/nerve-m4-p4-baseline-web.log` |
| `make lint-web` | 退出 0；54/54 个任务成功，其中 37 个命中缓存 | 基线；日志 `/tmp/nerve-m4-p4-baseline-lint-web.log` |
| `node tools/keywords.mjs`（启动时） | 72 条规则、23 条例外，无命中 | 已把 phase 改为 `M4/P4` |
| 架构放置合成图探针 | domain 放置失败；adapter 放置通过 | §3/A1；只验证依赖方向，未安装或运行 bluemonday |
| 移除临时探针后 `go -C server test -count=1 ./internal/archtest` | 退出 0；1.481s | 现有架构守卫未改动 |

### 2.2 T1 验收与独立评审（已完成）

T1 已交付清洗适配器、领域纯值、共享编辑器往返样例，以及局部解析/导出修复。实施范围为 `cbe0a9fc..18381d56`（主实现 `6cd23ca4`、F1 修正 `18381d56`）。独立评审的规格与质量结论通过；随后推进者确认的 F1 已由同一评审者定向复核为 ADDRESSED，修正未引入新问题，当前无遗留发现。过程报告分别为工作树 `.superpowers/sdd/P4-issue-collaboration/task-1-report.md` 与 `task-1-review.md`。

| T1 证据 | 实际结果与边界 |
|---|---|
| 共享样例与真实编辑器 | 14 例，实际 CoreEditorExtensions 的 22 个 node / 8 个 mark；Go 清洗、真实 HTML/Markdown 重序列化、再次清洗及独立投影断言通过；空 TextStyle 的规范化单独断言 |
| 默认标签 | 锁定上游 75 个默认标签加 4 个自定义标签；独立合法上下文/空元素/表结构断言，非允许标签删除；见 F1 |
| 实际可编译变异 | 首轮 14 项，加 F1 移除 ruby 共 15 项，均被业务断言发现并恢复绿；未把构建失败算作检出 |
| 首轮完整门禁 | `make lint-go` 两段 0 issues；`make test`（含 archtest）通过；`make lint-web` 54/54（33 缓存）、`make knip` 退出 0、`make test-web` 17/17（15 缓存） |
| 修正后定向门禁 | domain/content 全包、content 全包 race、lint-go 两段 0 issues、真实 Editor 5 tests 通过；未重跑不受影响的容器全套 |

原始日志在 `/tmp/nerve-p4-t1-logs/` 及其 `fix-1/` 子目录；代码和测试固定在上述提交，日志只佐证实际运行。T3 才由 module.New 接入首个 API 消费者，描述及 skip_activity 的接口行为仍由 T9/T15 验证；T1 完成不等于 P4 完成。

#### T1 / F1：锁定上游基础标签遗漏（Important，第1轮修正）

首轮规格/代码独立评审均给出 Approved，但规格评审把 Plane 放行表一致性列为差异外证据。推进者追溯锁定来源后确认实际存在规格差距：首版清洗策略遗漏25个 nh3 默认标签，且额外允许 `tfoot`；此前 Approved 不代表该差距已闭合，也不据此宣称 T1 完成。

只读来源链为 Plane `requirements/base.txt:75` 的 nh3==0.2.18、`content_validator.py:71–82` 的默认标签与四个自定义标签合并；[nh3 v0.2.18 源码](https://raw.githubusercontent.com/messense/nh3/v0.2.18/src/lib.rs)第232–234行从 ammonia 默认 Builder 导出标签，[其 Cargo.lock](https://raw.githubusercontent.com/messense/nh3/v0.2.18/Cargo.lock)锁定 ammonia 4.0.0，[ammonia v4.0.0 默认表](https://raw.githubusercontent.com/rust-ammonia/ammonia/v4.0.0/src/lib.rs)第359–369行定义75个标签。以该表加四个自定义标签修复；`tfoot`没有真实编辑器必需证据，按裁定移除。本轮只对齐基础标签集合；属性、CSS和值校验仍沿 Nerve 已批准的较窄策略，不宣称与 Plane 任意属性逐字全等。保留原URL约束，不引入 nh3/ammonia 依赖或修改守卫。

新增独立行为回归按合法父子上下文、空元素和表结构逐项覆盖默认标签，并固定 unknown/script/tfoot 删除断言。先看到标签丢失及 tfoot 留存的业务红，再补齐；另明确允许 area/bdo/data/map/rtc 的无属性形式，避免 bluemonday 隐式集合仍丢标签。删除 ruby 的实际可编译变异被 `TestCleanPlaneDefaultTags/ruby` 检出，恢复后 domain/content（含既有 S3 与恶意样例）、content race、lint-go、真实 Editor S3 均通过；具体命令和原始日志在 `task-1-report.md` 的“F1 修正第1轮”追加节。本轮未重跑无关容器全套。原独立评审者复核 `6cd23ca4..18381d56`，确认 F1 ADDRESSED，规格通过、质量 Approved，无新增或范围外发现。

### 2.3 T2 验收与独立评审（已完成）

实施范围为 `3ce74a9b..91be8ecb`。共享页大小从 identity 移入 shared，PAT 仍先检查游标再检查 limit；三个 issue 游标各有领域类型，复用规范封套，严格校验两元素、UTC 与非零 UUID。平台 `ReadOnlySnapshot` 和 `LockTimeout` 在构造末尾组合，不改变原有写事务、嵌套、取消与收尾行为。生产及测试组合均注入独立 ReadTx；新列表自 T3 起消费，未改 P3 已有列表，也未添加空调用。

| T2 证据 | 实际结果与边界 |
|---|---|
| 域与 PAT 回归 | 默认/边界/完整错误字段、三种游标、篡改/规范拼写及原 PAT 分页通过 |
| 真实库快照 | 两种 Option 顺序均保留隔离/只读/锁超时；外部连接提交后旧快照不变，新事务可见；拒绝写入后回滚，同连接回池无参数残留；嵌套写事务不升级隔离 |
| 实际变异 | 去掉隔离、去掉只读、后 Option 覆盖前配置、放过零 UUID、放宽两元素，共 5 项实际编译后触发业务断言，恢复绿 |
| 完整门禁 | `make lint-go` 两段 0 issues；`GOFLAGS=-p=1 make test` 的 server 与 server/tools 全通过；无契约/Web/迁移变化，未跑无关门禁 |
| 独立评审 | Spec Compliance 通过、Task Quality Approved；无 Critical/Important/Minor，不需要修复轮 |

过程文件为 `.superpowers/sdd/P4-issue-collaboration/task-2-report.md`、`task-2-review.md`，日志前缀 `/tmp/nerve-p4-t2-`。首次快照夹具误把事务内 ctx 用作外部写，已改用事务外 ctx；旧错误夹具不作为独立连接证明，修正后的真实库测试和隔离变异提供该证据。评审的两个差异外边界已闭合：推进者核对运行日志及干净的 main/工作树；新列表同 ctx 的消费属于 T3/T5/T10/T12 的明确后续要求，已进入简报，T2 不宣称已交付列表 API。

### 2.4 T3 验收与独立评审（已完成）

评论创建、升序分页、提及成员目录及迁移00021已提交为 `98dcbc3e`，实施基线为 `c380c48c`，文档接线补充为`3a3f0e50`。清洗器由模块组合，列表从首次目标读取起使用独立ReadTx；新表同步授权、三层删除目录、API夹具、权限矩阵、503和事务连接检查。独立审查覆盖`c380c48c..3a3f0e50`，规格通过、质量Approved，Critical/Important/Minor均为0，不需要修复轮。

推进者已核对原始日志：`make lint-go`两段0 issues；`make test`全部通过（含bootstrap107.027s、迁移和tools）；最后`make lint-web`54/54（36缓存）、`make knip`退出0、`make test-web`17/17（全缓存），提交后`make gen-check`无差异。九项实际可编译变异覆盖可见性、成员资格、OwnOnly、评论工作项范围/软删过滤、写表序、删除步骤、grants及外键，均触发具体业务断言并恢复。报告及原始日志位于工作树`.superpowers/sdd/P4-issue-collaboration/task-3-report.md`和`logs/task-3-*.log`。

端到端首轮122通过/4失败，其中I15/I17独立前置未给新评论表造行。补API前置后两故事定位2/2通过，最后完整`make e2e`为124通过/2失败；剩余仅A4的两项Firefox故事，均报本机Firefox可执行文件缺失，符合交接03已记录的环境限制，未安装浏览器。此结果不是端到端全绿；没有将首次业务失败归为环境豁免。

独立报告为`task-3-review.md`。差异外边界由推进者闭合：运行日志与会话记录对应实际生成/变异/串行容器过程，主目录仍为干净的main；`server/configs/config.yaml:17`及`platform/httpserver/api.go`保留1 MiB默认请求体限制，生产`bootstrap/app.go`和测试`issue_writes_test.go`注入T2验证过的独立ReadOnlySnapshot；`bootstrap/gate.go`仍是503映射点，`issue/module.go`的删除批大小仍为1000。后续清洗器复用、评论表情、描述和完整阶段性质分别由T4/T6/T9等任务继续，不计为本任务交付。

### 2.5 T4 验收与独立评审（已完成）

评论编辑/删除及共享Locks的子资源检查已提交为`de2c43ed`，基线为`1e5db490`。预检和锁下都重读评论并执行作者/ProjectAdmin规则；隐藏或缺失统一评论404，作者403先于归档，归档先于无改动。编辑按issues→subscribers→comments写入并使用同一时刻，删除只软删目标活评论，T6再补真实表情及删除连带。新增动作、接口契约、错误文案、矩阵、503、锁形状和事务连接均同步。

推进者核对`task-4-report.md`及`logs/task-4-*`原始证据：app可编译桩和真实HTTP均先出现业务红；最终`make lint-go`两段0 issues，`make test`全部通过（bootstrap96.730s），`make gen`和提交后`make gen-check`通过。最后顺序运行Web lint54/54（38缓存）、knip退出0、test17/17（全缓存）；首轮test-web17/17中1项缓存。初轮lint-go格式提示已修正；误并行Web lint/test引发共享editor构建目录EEXIST，保留失败日志，命令结束后顺序复跑通过，未改构建脚本或清理他人资源。后续生成/Web门禁统一串行。

五项实际可编译变异分别覆盖管理员误获编辑权、作者检查移到gate后、遗漏锁下重读、隐藏父项404形状、无改动仍写审计，均业务红并恢复。第二项另用真实外部事务持项目锁，1秒期限内应直接答403；移除预检后该业务断言失败，恢复后通过，不将它计成第六项变异。本Task未改迁移/furnish/e2e故事，未运行`make e2e`，不宣称端到端全绿。

独立审查范围`1e5db490..6dc99e6b`，报告`task-4-review.md`给出规格通过、质量Approved，Critical/Important均无；复核共享Locks的P3 reach/instead路径、错误顺序和外部锁测试清理后未发现行为回归。一项Minor为`app/lock.go`的precheck注释仍称see/issue.not_found，而实际已用seeAs/w.notFound；已随T5同步，T5独立审查确认闭合，不因注释重跑套件。过程证据已由推进者关联实际日志/会话，主目录仍保持干净main；T6的真实表情读删明确进入后续任务，没有把它算作T4交付。

### 2.6 T5 验收与独立评审（已完成）

工作项表情和迁移00022已提交为`8470a689`，基线为`0eb813d9`。三个操作使用既定动作和同一Locks路径，列表从首次可见性检查起使用独立ReadTx；添加/移除只处理调用者自己的活行，先改主工作项审计再写表情，不新增动态或关注。新表同步授权、部分唯一/CHECK/外键、工作项及项目删除步骤、三层目录和公共/I15/I17接口前置。T4的旧预检注释已同步，随本任务差异复核。

九项实际可编译变异覆盖Add/Remove各漏审计、删除缺actor范围、已删仍存在、部分唯一改全量唯一、去CHECK、工作项/项目漏删步骤、缺grants，全部触发业务断言并恢复。原始证据在`.superpowers/sdd/P4-issue-collaboration/task-5-report.md`和`logs/task-5-*`，推进者已核对业务红/绿、变异和完整门禁，不把编译或夹具错误计入变异。

最终`make gen`、`make lint-go`两段0 issues、`make test`（bootstrap99.734s及tools）和提交后`make gen-check`通过；Web lint54/54（4缓存）、knip退出0、test17/17（10缓存）。`make e2e`为124通过/2失败，只有A4已知Firefox可执行文件缺失；I15/I17和公共目录故事通过，不宣称端到端全绿。一轮迁移/存储定位因测试容器reaper消失失败，保留原日志，未操作已有容器；原命令重跑及最终全套均通过。初轮迁移计数/表名排序、旧删除测试表数与格式问题均修正后复验。

独立报告`task-5-review.md`审查`0eb813d9..f8883581`，规格通过、质量Approved，Critical/Important/Minor均无，T4注释Minor确认闭合。差异外边界由推进者核对日志和执行过程；生产`bootstrap/app.go`的独立ReadOnlySnapshot注入未变，新列表的消费已在差异中确认，主目录保持干净main。T6的评论表情及二层外键明确留后续，没有提前建表或省略本Task已有表的验证。

### 2.7 T6 验收与独立评审（含I1修正，已完成）

评论表情、页内批读及两种PATCH完整响应已提交为`a422cd91`，基线为`900f4c33`。目标检查复用同一Locks预检/锁下再判，反应不要求评论作者，只写自己的表情行，不改工作项/评论审计、动态或关注。评论列表在原ReadTx中批读本页，PATCH在原写事务中读回活表情，读取失败保持原错误并回滚改动。

迁移00023、授权、删除步骤、HTTP契约、权限矩阵、锁形状、503及公共/I15/I17接口前置一并落地。二层目录守卫核实实际外键列端点，未知链仍拒绝。二层软删变异首次存活，暴露workItemStamps未枚举新表；补齐后同一变异被子树删除戳断言检出。另七项实际变异覆盖父审计、评论范围、软删过滤、物理二层FK、PATCH两分支及HTTP映射；八项均为可编译的业务失败并恢复绿，不把首次存活或夹具panic计入杀死。

`make gen`、Go lint两段0 issues、全量Go（bootstrap102.754s及tools）、Web lint54/54（4缓存）、knip、Web test17/17（10缓存）及提交后gen-check通过。E2E为124通过/2项已知A4 Firefox可执行文件缺失，I15/I17通过，没有新增业务失败。首轮端点反例误调用fatalProbe未实现的Fatal及两处goimports均已修正并保留原日志；端点反例实际拒绝断言通过。完整报告`task-6-report.md`与`logs/task-6-*`已由推进者逐项核对；首轮独立审查发现I1，后续修正和定向结论如下。

独立审查`task-6-review.md`对`900f4c33..132da60e`提出一项Important：E2E目录只读源列而隐含目标为id，未核实实际confkey。修复`48bb532c`加入实际目标列/两侧列数核验与两段完整端点断言，保留原递归图。新增两个隔离数据库反例在旧守卫下均未拒绝，修复后与I15/I17共4项通过；Web lint54/54（36缓存）、knip、test17/17（全缓存）及gen-check通过，完整E2E126通过/2已知Firefox缺失。未重复未改动Go全套，既有NO_COLOR/FORCE_COLOR警告如实保留。

定向报告`task-6-fix-1-review.md`审查`132da60e..48bb532c`，确认I1已解决、无新增Critical/Important/Minor，两个反例只改各自临时副本。Task 6规格与质量门由首审和定向复审共同闭合，无遗留发现。差异外边界由推进者核对：既有Locks/HiddenFrom/角色集合、gate唯一映射和独立ReadOnlySnapshot未变；创建评论显式空表情、HTTP映射及两个locale key仍正确消费；实际日志、只读Plane、串行容器与主目录main状态均有执行证据。阶段尚未到Claude终审。

### 2.8 T7 验收与独立评审（已完成）

描述纯模型、字段存在性和已有列读写已提交为`2eb31d8c`，基线为`5e8edee4`，共25个生产/测试/生成文件。内部Issue/IssueRow/IssueFields以指针保留stripped的NULL，创建明确保存默认HTML与空文本；普通字段PATCH从锁下旧值透传两列。仅描述写入只改两列，不动审计。GetIssue/GetIssues/SubIssues共享完整映射；HTTP字段、迁移与完整描述副作用留待T9。

领域、存储和真实旧API回归都有业务红绿；八项实际变异覆盖Unicode误按字节、忽略skip=false出现、漏可清空字段、误改审计、创建零值、普通PATCH丢值、适配器漏映射、NULL归一化，均触发业务断言并恢复。锁下再读回归也检出了PATCH零值变异；邻居和三种真实持久值均有保持断言。完整证据为`task-7-report.md`与`logs/task-7-*`，推进者已核对。

最终gen-go、Go lint两段0 issues、全Go（server63包、bootstrap98.351s、tools）及提交后gen-check通过；没有适用的Web/E2E业务改动。首个临时Postgres端口映射异常及一次仅进程API1.44诊断单独保存，原环境1.55重跑取得业务红、随后定位与全量通过，未改Docker配置或已有容器。全量命令在各包结束后曾较久未退出，只读确认无测试子进程，等待自然退出0，没有杀进程或清理缓存。独立审查结论如下。

独立报告`task-7-review.md`审查`5e8edee4..046e04c8`，规格通过、质量Approved，Critical/Important/Minor均无。对SubIssues、readBack、fake读取复用及锁下write函数作具名聚焦核对，确认新值没有在这些接缝丢失。差异外边界由推进者闭合：Locks、角色集合、gate、生产ReadOnlySnapshot和1 MiB请求体配置/接线均未改；T1清洗与内容标识已独立验收，T9简报明确先清洗后比较/写入。原始执行日志、主目录干净main、工作树整洁、串行临时资源与生成过程均已核对。T7没有公开HTTP描述编辑功能，后续边界不算本任务遗漏。

### 2.9 T8 验收与独立评审（已完成）

描述动态和版本窗口内部能力已提交为`c8f56df9`，基线`12974bd4`，11个代码/测试/生成文件。领域在标题后加入描述行；应用层只在最新活动态是同actor描述时合并，否则插入；存储按`created_at DESC,id DESC`读任意字段活行，合并保留id、旧值、actor与创建者，仅更新新值和指定时刻/更新者。版本纯规则独立使用last_saved_at，同owner且0至600秒含端点；版本表与原子描述API仍由T9交付。

单元与真实存储均取得可编译业务红，合并存储另有真实空写红；绿色回归覆盖端口错误原样传播、事务上下文/回滚、邻居与已删行。五项实际变异分别改用created_at、排除600秒端点、预筛description、漏写new_value、漏id排序，均被业务断言检出并恢复。去id排序首次存活，复合索引偶然提供同序；经本测试事务SET LOCAL关闭索引扫描后，同时间逆id断言检出了缺陷，生产索引/规则未改。首次存活日志保留，同一变异仅计一次。

推进者已核对`task-8-report.md`、`logs/task-8-*`原始失败与恢复、最终门禁及分支状态。最终gen-go、Go lint两段0 issues、唯一一次全Go（server63包及tools，bootstrap98.678s）和提交后gen-check均通过；无本Task适用的Web/E2E改动。早期测试helper编译失败不计业务红；首两条红测试及初次gen-go退出码据当次工具返回，其余相邻.exit为现场捕获，不混淆来源。主目录仍干净main，工作树提交后干净。独立评审结论如下。

独立报告`task-8-review.md`审查`12974bd4..3a82bb8d`，规格通过、质量Approved，Critical/Important/Minor均无。评审定点核对stamp.activity不重读时钟以及Store.queries复用事务上下文。差异外边界由推进者闭合：Locks、权限、gate、架构守卫、迁移和依赖均无改动；T9正式要求已包含标题先写、旧/新清洗、按last_saved_at/id读最新活版本、found=false新建和整个PATCH回滚，当前准备接口由该任务消费。实际日志/生成过程及分支状态已逐项核实，任务完成不代表P4阶段通过。

### 2.10 T9 验收与独立评审（已完成，携带两项Minor）

描述API、提及与版本原子写入已提交为`bd1d5fb9`，基线`f0bb1ed0`，68个代码/测试/生成文件。创建清洗一次并同步初版；更新在锁下清洗比较，描述缺省不读提及目录/版本，普通同HTML无变化零写。skip同标识只保存描述并同步关系资格，不改工作项审计、动态或版本；正常写按全局表序保存，并消费T8的最新动态与last_saved_at滑动规则。新增两表随本次API同时接入迁移、授权、sqlc、删除连带和接口夹具；详情仍经seen，列表不暴露描述。

单元与真实HTTP均先取得可编译业务红。十项实际业务变异覆盖提及差集、成员资格、skip审计、版本写入、事务回滚、seen、滑动窗口、两表删除及grants，均取得业务FAIL并恢复定位通过。首次提及变异为编译失败、首次全局绕事务变异为创建超时，二者保留且不计数；回滚改为仅UpdateIssue主行走pool后，末端版本故障明确检出主行残留。bootstrap目录曾抓到旧夹具缺新表行，改用描述API补齐，未放宽目录守卫。

推进者已核对`task-9-report.md`、`logs/task-9-*`和提交范围。完整Go首次因新增矩阵误把普通项目访客列为可写失败，修正测试预期后全套通过（server63包与tools，bootstrap100.749s）；授权实现未变。最终Go lint两段0 issues，Web lint54/54、knip、test-web17/17、gen及提交后gen-check通过。全Go后仅增强shape测试为完整properties schema及required精确比较，聚焦测试和lint通过，未宣称该测试增强后重跑全套。

端到端首轮125通过、I17夹具业务失败1项、Firefox缺执行文件2项；I17独立创建的三个资源经API补描述提及后定向1/1通过，原目录断言保留，不将分次结果写成一轮126通过。Firefox缺失沿交接03单列。提交前暂存空白检查曾非零但shell继续创建提交，已去掉空白、重新生成并用失败即停命令检查，通过后amend本Task未推送提交；最终范围diff检查和生成检查通过，原错误记录保留。主目录仍干净main，未推送或合并。

独立报告task-9-review.md审查f0bb1ed0..05a78fa3，规格通过、质量Approved，Critical/Important为0，Minor为2：M1的同owner超时组合用例把Bob版本回拨后Alice反成最新，不能独立证明超时分支；M2是HTTP三处注释仍描述旧契约。推进者定点确认，明确交紧接着的T10修正并由其独立评审复核，不进入T9阻断修复循环，也不静默丢弃。差异外边界已闭合：T8窗口纯规则与T1清洗沿已验收结论；Locks、seen公共实现、项目/工作区既有删除编排、gate、权限实现、底层事务、架构守卫和依赖未改，适用全Go及原始日志已核对。

### 2.11 T10 实施与验证（待独立评审）

描述版本列表与详情已提交为`ed472ff3`，基线`2a24ba87`，32个代码/测试/生成文件。读取使用独立小接口，列表从首次目标读取/授权起处于同一只读快照，按last_saved_at/id降序与limit+1分页；详情先定位活版本，再经seeAs隐藏父工作项，普通访客自己403、别人404，管理员兼访客自己的版本可读、别人仍404。两种归档可读，列表五字段、详情多HTML，不另造还原路由。

应用层取得可编译业务红后转绿；真实API/SQL覆盖100活版本、同时间逆id、跨时间主序、页长1/50/100、空页与满末页、邻居和软删、完整Problem一致及端口错误。列表在首次GetIssue验证只读/可重复读属性，另一连接提交删除后仍读取原快照；两个GET在项目FOR UPDATE持有时可完成，不占写事务/闸门。新增动作和version_id只扩现有规则/矩阵登记、种子和断言，未改通用守卫逻辑。

六项实际业务变异分别反向tuple、漏issue条件、漏软删条件、放宽guest、隐藏404错配和移除600秒上限，均业务FAIL并恢复；隐藏映射加强为先比较整个Problem再检查双方契约，同一变异只计一次。T9 M1已保证Bob过期版本仍为latest，再以同owner保存的新ID断言检出漏窗口规则；M2三处旧注释已同步，待本Task独立复核。

推进者已核对报告与原始日志。最终完整Go通过（server63包及tools，bootstrap102.395s），Go lint两段0 issues，Web lint54/54、knip、test-web17/17、gen与提交后gen-check通过。真实失败保留：无schema的GET被误要求拒绝空JSON、InvalidCursor误声明为独立码、import分组、access规则表缺新增动作枚举，均按既有规则修正，不放宽平台或守卫。仅扩Go矩阵种子，无迁移/E2E furnish/故事变化，本Task不适用E2E。

有一次明确的执行纪律偏差：两个bootstrap定位套件在23:08:17–20 UTC重叠约3秒，原因是前一命令返回session_id后未等到退出；二者自然结束，未操作既有容器。报告保留此事，不声称全程串行；后续每个变异/恢复及完整Go都等待退出后再启动下一套。主目录仍干净main，工作树提交后干净，无推送或合并。独立评审待执行。

### 2.12 尚未完成的 P4 验收

T10独立评审及T9两项Minor复核、历史目录与动态读取、跨资源权限矩阵增量、交错 14/15、接口故事、目录驱动删除增量、其他关键性质的实际变异、阶段完整门禁与 Claude 整分支终审尚未完成。M4 设计 §14 不新增勾选；P4 不标已完成。

## 3. 准备阶段的决定与裁定记录

### A1：清洗库应放在哪一层（负责人已批准，2026-10-25）

**发现时的问题。** 当时的 M4 设计要求在 `issue/domain/sanitize.go` 使用 bluemonday；M0 已建立并由测试强制执行的规则，则禁止 domain、app 和 shared 直接或间接依赖任何第三方库。按 M4 的文件放置直接实现会使架构检查失败；若为它放宽守卫，会改变已经建立的纯层边界。清洗库本身已获设计批准，冲突是放置位置，不是依赖许可。

**依据。**

- [M4 设计](../M4-design.md) §3.7、§6.2、§12/P4 Task 1、§16 在准备基线 `573e8eb6` 上明确指向 `issue/domain/sanitize.go` 或 `issue/domain`；§6.1 批准 bluemonday 和其传递依赖，并要求 issue 接受现有 archtest 检查。
- [M0 设计](../../M0-foundation/M0-design.md) §3 的模块边界表和 §3.7 的第 2、10 条及传递检查禁止该依赖。
- 现有 `server/internal/archtest/rules_test.go` 的 `innerLayersArePure` / `isInfrastructure` 拒绝直接导入；`purity_test.go` 的 `TestPureLayersReachNoInfrastructure` 拒绝间接导入。把依赖转藏在 shared 也不成立。

**实际验证。** 临时测试调用现有 `check(graph)` 和传递检查函数，输入以下两张合成依赖图，不增加第三方依赖、不改守卫：

```text
原设计：issue/domain → github.com/microcosm-cc/bluemonday
候选：  issue（module.go）→ issue/app、issue/adapter/content
        issue/adapter/content → issue/app、issue/domain、bluemonday、x/net/html
        issue/app → issue/domain
```

运行 `go -C server test -count=1 ./internal/archtest -run '^TestP4SanitizerPlacementProbe$' -v`，退出 1：

```text
design_domain_placement: FAIL
internal/modules/issue/domain imports github.com/microcosm-cc/bluemonday:
domain and app import only the standard library (not net/http or database/sql),
their own module's inner layers and internal/shared
proposed_adapter_placement: PASS
```

探针随即移除，原有 archtest 重跑通过。源文件副本、完整输出与重跑日志保存在工作树的 `.superpowers/sdd/P4-issue-collaboration/`，分别为 `p4_sanitizer_probe_test.go.txt`、`sanitizer-placement-probe.log`、`archtest-after-probe.log`。这是依赖图证据，不代表清洗实现或安全验证已完成。

**已批准的 A 方案。** 调整 M4 对清洗的层次安排，保持现有架构规则：

1. domain 只保存清洗结果、内容标识等纯值及规则。
2. app 声明由用例消费的 `HTMLSanitizer` 小接口。
3. `issue/adapter/content` 集中实现清洗、解析和内容投影，承接 bluemonday 与 `x/net/html`，放行表仍只有一份。
4. `issue/module.go` 构造实现并注入用例；bootstrap 继续负责模块组合。描述和评论的产品行为不变，S3 和安全验证仍是第一项实现验收。
5. 同步 M4 设计中上述位置及 spec/plan，再进入预检；不修改或豁免现有架构守卫。

这沿用现有 identity 的做法：`app.PasswordHasher` 声明接口，`adapter/argon2` 引用第三方库，`module.go` 组装。代价是增加一个窄接口和适配器，并修订 M4 设计的层次说明。

**备选 B。** 保留 domain 清洗实现，为第三方依赖放宽纯层规则。它需要修改基础架构约束及直接/传递守卫，会影响以后对纯层的判断，**不推荐**。

**负责人裁定（2026-10-25）：批准 A。** 原话：“清洗放到 issue/adapter/content，app 声明小接口，domain 保持纯净，不放宽架构守卫。”同时要求同步设计、spec 和 plan 后继续 P4，完成后先不合并，由负责人安排 Claude 整分支终审。原暂停点已提交为 `573e8eb6`；本次同步层次说明，不改变产品规则。

### 已形成的局部计划调整

- 五张新表按首个可用接口的任务加入，同步删除连带和 `furnishIssueRows`，避免用临时豁免绕过目录驱动删除断言。
- 为满足 P4 读目录与查询在同一快照内，计划把设计已批准的 `ReadOnlySnapshot` 从 P5 前移；它必须与现有 `LockTimeout` 的事务开始路径兼容，交给 P5 去重。
- 提及资格须按被提及成员自身的可见性判定，不能沿用发起人的 `OwnOnly`；在既有 `ProjectMembers` 目录职责内细化，独立预检确认可沿原有模块边界实现，尚未实现。不得改变 P2 对 `ProjectGate.Lock` 返回值的约定，发现额外跨模块协议变化时仍按负责人纪律处理。

### 独立预检与局部修订（2026-10-25）

独立预检以 `391c76e9` 的 spec/plan 为基准，核对 16 个任务、92 对共享文件/接口及证据关系、12 个新操作的权限/锁/404 路径，发现 High 0、Medium 5、Low 1；没有新的架构冲突。过程报告保存在 `.superpowers/sdd/P4-issue-collaboration/preflight-review.md`，不是产品验证或整分支终审。以下均由推进者按负责人授权作局部裁定；统一修订后的定向复核已确认六项发现闭合，可进入 Task 1。

| 编号 | 发现与裁定 | 理由和不修的影响 |
|---|---|---|
| M1 | 五份迁移统一为 `server/migrations/sql/…`，编号不变 | 现有 embed 与 sqlc 消费该目录；错误路径会漏建表 |
| M2 | 表情列表 HTTP 返回 `IssueReactionCollection`，空值为 `{data:[]}`；用例仍可返回切片 | 遵从设计 §5.2，避免实现成裸数组 |
| M3 | T6 复用同一 `CommentReactions` 窄读取能力，在当前写事务内组装更新/无改动评论响应；列表保持批读 | 完整 Comment 响应须保留已有活表情，不能把未知状态当作空集合 |
| M4 | T7 同时修改创建/更新用例构造与存储映射：创建显式默认，普通 PATCH 透传锁下旧描述；增加中间态回归 | 新 SQL 参数若消费零值，会绕过默认或清空旧描述，不能等 T9 才恢复正确行为 |
| M5 | T1 对 callout 现有 Markdown 出口作上下文转义，URL 继续由清洗适配器验证；增加真实导出断言及变异 | data 属性在 HTML 中安全，不代表解码后可直接插入 HTML 片段。保留合法 emoji/icon，阻止生成额外属性/标签；预检时为源码边界缺口，T1随后实际复现了Markdown额外属性，未宣称浏览器执行 |
| L1 | 明确仅 `skip=true` 总写；`skip=false` 且净化 HTML 相同不写，两者都受字段组合约束 | 消除 spec 的文字冲突，不改变已裁定语义 |

另将 T10 对 T4 子资源检查的依赖写入计划；“拒绝零连接”准确限定为不取得写事务连接，预检读取仍会短暂借用池连接。后续提交保留 `feat(M4/P4)` 等前缀，按本会话较新的用户级 AGENTS 要求使用中文说明，已提交历史不改写。

### S3中的局部编辑器解析修复

Task 1 的真实编辑器测试发现：CustomColor 的 `getAttrs` 在缺属性时返回 `null`（表示接受），会把 emoji 的 span 当作颜色；带 style 的颜色 span 又被优先级 101 的通用 TextStyle 先消费，丢掉颜色属性。推进者核对当前源码及锁定版本后裁定：只接受自身非空颜色属性，优先级设为 102，测试分别证明 emoji、文字色和背景色保留并实际变异。范围只增加 `custom-color.ts` 的局部解析修复，不改第三方库或架构。当前没有持久属性的空 TextStyle 可按原生规则归一化，但须单列样例、断言 schema 属性为空并保留文字；未来新增属性不得静默忽略。该项已在T1实现、验证并通过独立 Task 评审。

同一 S3 测试还实际复现了颜色属性的再解释：`data-text-color` / `data-background-color` 保存带分号的值后，editor 会生成额外的 `position`、`background-image` 声明；表格颜色属性有同类入口。这是输出 CSS 的证据，没有宣称脚本执行或网络请求。推进者裁定在适配器现有唯一属性过滤边界验证这些颜色值，保留真实调色板键和合法 CSS 颜色、拒绝注入值，复用颜色校验避免策略漂移；共享真实往返和移除校验变异须证明修复。不改架构、不新增依赖、不放宽 style，T1已完成修复、定位验证和独立评审，结果见§2.2。

T1 共享金样 `api/testdata/issue-richtext.json` 单列文件规模例外：它保存同一条 Go/TS 往返链的固定输入、真实 editor 文档与逐节点/属性期望，是数据契约而非逻辑大文件。保留可读排版和完整证据，避免为压行数压缩或把一份跨语言约束拆散；具体行数以实际提交报告为准，其他逻辑文件仍遵守约 400 行约束。

### T3 中的共同接线与删除故事前置

H文件清单补充`api/openapi.yaml`的新路径根转引，以及`bootstrap/gate_test.go`的逐写503枚举。首次HTTP契约定位实证根转引缺失会使新操作不在文档契约内；闸门测试也须显式枚举新写。这是已要求的API/503接线，没有新增架构或改变守卫。

M同时覆盖公共furnish及已有故事独立构造的目录行。T3首轮端到端实际发现I15/I17缺少评论前置，故新增窄`e2e/fixtures/comments.ts`供公共夹具与两故事复用：I15在转Done前创建评论，I17在归档末级前为目标、两层子树与邻居创建评论；评论者与删除者不同。保持目录断言与API前置纪律，不等T15才修、不新增豁免。plan/spec与任务简报同步记录此局部调整。

### T4 中的错误文案与测试边界

沿设计既定`issue_comment.write`动作实现相对权限。`issue.not_author`文案明确“Only the author can edit a comment; only the author or a project administrator can delete it.”，避免笼统only-author变更误述管理员删除权限；中文同义。`issue.comment_not_found`统一“The comment does not exist, or you cannot see it.”，不增加存在性信息。

带日志gate的失败端口测试先断言成功Enter后恰有一次末尾的预期项目Leave、且已离开事务，再仅去掉该末项复用既有helper；预检拒绝仍要求零Enter/Begin/Now/Lock/Leave。未放宽共享helper。完整工作项会产生初始活动，故用活动整行快照前后相等证明未修改，而非硬编码活动数量或只比ID。G中的迁移/furnish/故事门禁不扩成所有bootstrap种子：本Task矩阵种子由完整Go测试覆盖，不重复无关e2e。

### T5 中的矩阵参数与迁移反例适用范围

`{reaction}`是设计点名的值参数，不是UUID。矩阵目标守卫只在`/api/v0/issues/{issue_id}/reactions/{reaction}`精确路径接受固定种子值`128077`，其他值或未知路径仍拒绝，既有矩阵自检补反例；没有增加notTargets豁免。T6新增评论路径时再按其精确契约扩展，不泛放行未知参数。

M中“漏二层comment外键”的实际变异明确属于T6的`comment_reactions→issue_comments→issues`，该表未建出的T5不提前创建它或重复变异已有comments。每张新表仍同步本表外键、授权、目录与实际漏删/删宽反例；T5自己的两个删除步骤和grants均已实际变异。此为计划测试范围消歧，不改变数据库规则或降低守卫。

T6准备时确认：T5的共享表情错误文案当前限定“on the work item”，对现有消费者正确；新增评论消费者时统一改成资源中性的“You already have this reaction.”与“You do not have this reaction.”并同步中英，保持同一业务码/状态。避免为同一语义新增错误码，或让评论错误指向工作项；这是局部文案适配，待T6验证。

### T6 二层目录与测试职责

新迁移约束测试独立为`server/migrations/comment_reaction_schema_test.go`，复用现有基础设施，避免扩大原约370行的文件。目录守卫只精确支持设计点名的comment_reactions→issue_comments→issues实际FK列端点，行本身参与全部删除/戳/邻居断言，保留未知链和错误端点反例；这是既定二层表的完整接线，没有豁免。首次变异存活促成workItemStamps同样补齐，过程证据保留。首审I1进一步补齐E2E的confkey核验；两个真实错误目标列反例置于独立i17-catalog.spec.ts，各用模板副本、只关闭本例资源，复用现有工厂。

### T7 旧可空列的值保持（后续实施要求）

预核对发现00015的description_stripped为可空text，P1–P3 INSERT省略该列，旧行实际为NULL。原T7的string映射会在普通字段PATCH透传时把NULL改成空串，违背保存两列原值。局部裁定：domain.Issue及app.IssueRow/IssueFields用`*string`保留NULL；新增创建/明确描述保存写已知文本，普通PATCH从锁下旧值原样透传。DescriptionWrite.Text与RichText.Text仍是string，不用COALESCE、不改迁移或公开API。新增NULL/空串分别保留的真实回归和归一化变异；skip结构检查同时覆盖可清空字段的存在性bool。spec、plan与T7/T9简报同步；T7实现已有真实NULL/空串回归及变异，独立评审已确认。SubIssues既有Go转换直接复用扩展后的GetIssueRow/issueOf，因此本Task改查询与生成类型并验证实际读取，无需为原计划文件枚举制造无效Go差异；文件清单同步实际职责。

### T9 描述写入与评论关注接缝（实施要求）

只读核对现有设计3.7/3.12与T7代码后，明确“每次描述保存同步提及”指实际写入：普通相同HTML且其他字段无变化时零写，不额外读目录；skip=true即使内容相同也会写入，所以同步成员资格。设计/spec/plan及简报统一措辞，没有改变既定行为。评论的visibleMentionSubscribers总加入actor，描述更新不能直接照搬；只合并实际新增负责人和可见被提及者，测试非创建者/非负责人/未被提及的编辑者不会自动关注。T3成员目录接口不变，不增加跨模块协议。

实施时将组合测试拆为issue_descriptions_test（详情/seen）、issue_description_atomic_test（原子链/普通版本）和issue_description_skip_test（skip/边界），复用issueWorld并保持各文件职责。描述的descriptionTargets直接消费既有ProjectMembers与HiddenFrom，评论mentions.go没有所需行为变化，因此从计划的修改清单移除，不为预估文件表制造无效差异。这两项局部调整已同步计划和任务简报。

## 4. 整分支评审的发现与处理

尚未送 Claude 终审。A1 是实施前发现，负责人已批准 A 方案。后续完成 P4 后，按用户要求交付分支、基线、spec、plan 和本记录，等待负责人安排 Claude 对整分支终审；处理发现后才能合并。

## 5. 最终实现与 plan 的差异

T1实施中增加了CustomColor既有解析的局部根因修复，以及颜色属性再次生成CSS的值检查，已同步spec/plan，依据见§3。最终阶段差异待后续Task完成后核对。启动变化为 `tools/keywords.json` 的phase和工作树忽略规则（`05841220`）；`573e8eb6`保存A1暂停草案，`391c76e9`统一批准方案，`cbe0a9fc`闭合预检修订。

## 6. 移交事项

- **恢复 P4：** A1 已记录批准；设计/spec/plan已同步，独立预检及修订复核通过；T1–T9已完成独立评审，继续Task 10并闭合T9两项Minor。不能把本文的基线检查算成 P4 的验收结果。
- **P5：** 若采用前移方案，复用 P4 落地的 `ReadOnlySnapshot`，不要重复实现；其验证结果在实际完成后补入正式交接。
- **Claude：** 本分支已完成T1–T9，T9两项Minor已携带到T10，T10起后续任务待推进，尚未到阶段终审节点；T4一项注释Minor已由T5独立审查确认闭合。
- M4 的各份交接维持当前状态，不因计划覆盖而提前关闭；M5–M8 的 M4 收尾交接尚未产生。

## 7. 已知限制

- P4 尚未全部实现，M4 远未达到收尾完成条件；没有推送、PR 或合并，没有新 main CI 结论。
- W11 既有偶发失败按 [06](../../claude_to_gpt/06-ci-w11-flake.md) 的裁定不调查；未来某次合并被它阻塞且重跑仍失败时再上报。
- 独立预检及六项局部修订的定向复核已完成；只代表计划层可执行，不代表产品测试或阶段终审通过。
