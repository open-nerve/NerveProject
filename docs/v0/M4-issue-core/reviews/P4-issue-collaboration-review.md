# M4/P4 描述与协作：评审与裁定记录

| 项 | 内容 |
|---|---|
| Phase | M4/P4 `issue-collaboration` |
| 日期 | 2026-10-25 |
| 状态 | **实现/适用验证完成，待终审**：T1–T16独立评审通过，T15两项Minor已由T16复核关闭；本地Firefox两项环境失败；分支CI 40170883949中Web/服务端通过、E2E的S2失败待定位；Claude终审未执行，尚未合并 |
| spec / plan | [spec](../specs/P4-issue-collaboration.md) / [plan](../plans/P4-issue-collaboration.md) |
| 分支 | `codex/m4-p4` |
| main 基线 | `56d575fb8ff6575880d1edef65219728a81f2b12` |
| 工作树 | `/Users/xiaoruan/project/nerve-project/.codex/worktress/codex/m4-p4`；主目录保持 `main` |

## 1. 评审方式

- 按[交接入口](../../claude_to_gpt/README.md)的顺序读取 01、03、02、04、05、06，核对 M4 设计的 P4–收尾、完成标准、进度表，以及 P3 的 spec、plan、review 样板和此前交给 P4 的事项。
- 依交接 02 §3 调整计划产出方式：保留文件、接口、验收、执行命令与关键性质的变异计划，不把尚未运行的原型或逐块重放写成已验证。架构子任务负责 spec/plan 草案，主任务核对既有代码与基线。暂停时保存了 spec 和 16 个任务纲要（`573e8eb6`）；负责人批准 A1 后恢复计划细化与预检。
- 在准备 Task 1 时发现 A1，曾停止进入预检和产品实现；负责人已批准推荐方案，现恢复推进。T1 已完成独立 Task 评审、F1 修正与定向复核，T2–T16 独立评审通过，T9/T13两项Minor分别在T10/T14关闭；T15两项Minor已由T16独立复核关闭，Claude整分支终审待执行。下表保留起步基线和架构放置探针，不与 T1 结果混淆。

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

### 2.10 T9 验收与独立评审（已完成，两项Minor已由T10关闭）

描述API、提及与版本原子写入已提交为`bd1d5fb9`，基线`f0bb1ed0`，68个代码/测试/生成文件。创建清洗一次并同步初版；更新在锁下清洗比较，描述缺省不读提及目录/版本，普通同HTML无变化零写。skip同标识只保存描述并同步关系资格，不改工作项审计、动态或版本；正常写按全局表序保存，并消费T8的最新动态与last_saved_at滑动规则。新增两表随本次API同时接入迁移、授权、sqlc、删除连带和接口夹具；详情仍经seen，列表不暴露描述。

单元与真实HTTP均先取得可编译业务红。十项实际业务变异覆盖提及差集、成员资格、skip审计、版本写入、事务回滚、seen、滑动窗口、两表删除及grants，均取得业务FAIL并恢复定位通过。首次提及变异为编译失败、首次全局绕事务变异为创建超时，二者保留且不计数；回滚改为仅UpdateIssue主行走pool后，末端版本故障明确检出主行残留。bootstrap目录曾抓到旧夹具缺新表行，改用描述API补齐，未放宽目录守卫。

推进者已核对`task-9-report.md`、`logs/task-9-*`和提交范围。完整Go首次因新增矩阵误把普通项目访客列为可写失败，修正测试预期后全套通过（server63包与tools，bootstrap100.749s）；授权实现未变。最终Go lint两段0 issues，Web lint54/54、knip、test-web17/17、gen及提交后gen-check通过。全Go后仅增强shape测试为完整properties schema及required精确比较，聚焦测试和lint通过，未宣称该测试增强后重跑全套。

端到端首轮125通过、I17夹具业务失败1项、Firefox缺执行文件2项；I17独立创建的三个资源经API补描述提及后定向1/1通过，原目录断言保留，不将分次结果写成一轮126通过。Firefox缺失沿交接03单列。提交前暂存空白检查曾非零但shell继续创建提交，已去掉空白、重新生成并用失败即停命令检查，通过后amend本Task未推送提交；最终范围diff检查和生成检查通过，原错误记录保留。主目录仍干净main，未推送或合并。

独立报告task-9-review.md审查f0bb1ed0..05a78fa3，规格通过、质量Approved，Critical/Important为0，Minor为2：M1的同owner超时组合用例把Bob版本回拨后Alice反成最新，不能独立证明超时分支；M2是HTTP三处注释仍描述旧契约。推进者定点确认，明确交紧接着的T10修正并由其独立评审复核，不进入T9阻断修复循环，也不静默丢弃。差异外边界已闭合：T8窗口纯规则与T1清洗沿已验收结论；Locks、seen公共实现、项目/工作区既有删除编排、gate、权限实现、底层事务、架构守卫和依赖未改，适用全Go及原始日志已核对。

T10已完成两项修正，独立报告task-10-review.md逐项确认M1/M2均为ADDRESSED，闭合证据见下一节。

### 2.11 T10 验收与独立评审（已完成）

描述版本列表与详情已提交为`ed472ff3`，基线`2a24ba87`，32个代码/测试/生成文件。读取使用独立小接口，列表从首次目标读取/授权起处于同一只读快照，按last_saved_at/id降序与limit+1分页；详情先定位活版本，再经seeAs隐藏父工作项，普通访客自己403、别人404，管理员兼访客自己的版本可读、别人仍404。两种归档可读，列表五字段、详情多HTML，不另造还原路由。

应用层取得可编译业务红后转绿；真实API/SQL覆盖100活版本、同时间逆id、跨时间主序、页长1/50/100、空页与满末页、邻居和软删、完整Problem一致及端口错误。列表在首次GetIssue验证只读/可重复读属性，另一连接提交删除后仍读取原快照；两个GET在项目FOR UPDATE持有时可完成，不占写事务/闸门。新增动作和version_id只扩现有规则/矩阵登记、种子和断言，未改通用守卫逻辑。

六项实际业务变异分别反向tuple、漏issue条件、漏软删条件、放宽guest、隐藏404错配和移除600秒上限，均业务FAIL并恢复；隐藏映射加强为先比较整个Problem再检查双方契约，同一变异只计一次。T9 M1已保证Bob过期版本仍为latest，再以同owner保存的新ID断言检出漏窗口规则；M2三处旧注释已同步，两项均经本Task独立复核关闭。

推进者已核对报告与原始日志。最终完整Go通过（server63包及tools，bootstrap102.395s），Go lint两段0 issues，Web lint54/54、knip、test-web17/17、gen与提交后gen-check通过。真实失败保留：无schema的GET被误要求拒绝空JSON、InvalidCursor误声明为独立码、import分组、access规则表缺新增动作枚举，均按既有规则修正，不放宽平台或守卫。仅扩Go矩阵种子，无迁移/E2E furnish/故事变化，本Task不适用E2E。

有一次明确的执行纪律偏差：两个bootstrap定位套件在23:08:17–20 UTC重叠约3秒，原因是前一命令返回session_id后未等到退出；二者自然结束，未操作既有容器。报告保留此事，不声称全程串行；后续每个变异/恢复及完整Go都等待退出后再启动下一套。主目录仍干净main，工作树提交后干净，无推送或合并。

独立报告task-10-review.md审查2a24ba87..8dc2ed01，规格通过、质量Approved，Critical/Important/Minor均为0，T9 M1/M2均ADDRESSED。评审定点核对seeAs隐藏优先顺序、生产ReadTx独立注入和M1 latest辅助查询。差异外边界由推进者闭合：T1清洗、T2游标和快照底层、T8窗口纯规则、T9普通PATCH还原及删除链沿既有验收；Locks、platform、project模块、gate、生产组合、shared、架构守卫与依赖在本Task范围无改动，最终完整Go与原始日志已核对。GET无requestBody与本次不适用E2E的边界已明确；阶段剩余任务和Claude终审不在本次Approved内。

### 2.12 T11 验收与独立评审（已完成）

历史项目目录已提交为`80d76256`，基线`990428bd`，11个代码/测试/生成文件。issue/app声明只含Historical的小端口，project只查询自有projects/project_members，bootstrap转换值并原样传播错误。查询限定workspace、actor和is_active，按项目id排序；不看项目/成员软删除或归档，OwnOnly只由项目guest角色与guest_view_all_features决定。T12消费目录过滤引用；没有改变目标工作项授权、ArchivedProjects或添加普通目录方法。

先取得存储及转换/真实组合的可编译业务红，再实现转绿。数据覆盖私有/公开、项目和成员分别软删、归档、退出、跨工作区、其他actor、无成员关系、id顺序、普通guest和管理员兼guest。四项实际SQL变异分别增加软删过滤、漏is_active、漏workspace、错误置OwnOnly=false；前三由存储测试、最后一项由真实管理员兼guest组合断言检出，均以业务FAIL退出1。每项SQL经gen-go生成，最终恢复原SQL并重新生成，两组定位测试均恢复通过。

推进者核对task-11-report.md和原始日志：Go lint两段0 issues，唯一一次完整Go通过（server63包及tools，bootstrap106.601s），提交后gen-check通过，工作树干净、主目录仍干净main。所有容器命令串行等自然退出，无容器/Docker操作；本Task无HTTP/Web/迁移/E2E furnish/故事变化，不适用对应门禁。首轮组合夹具尝试直接把workspace管理员作为project guest加入被既有API拒绝422，改为正常创建后在隔离测试库构造历史窄角色，再取得真实目录业务红；该前置失败不计变异。

独立报告task-11-review.md审查990428bd..8d03317c，规格通过、质量Approved，Critical/Important/Minor均为0。评审定点核对Store.queries透传事务、成员移出与重入沿用原行的生命周期，以及wanda真实workspace管理员前提。差异外边界由推进者闭合：平台事务、共享授权、Locks、gate、架构守卫及依赖均无改动，原始验证和VCS已核对；目标授权、历史引用过滤及首次读取起同一快照的实际消费已明确进入紧接的T12正式要求，本Task不宣称交付动态API。

### 2.13 T12 验收与独立评审（已完成）

动态分页已提交为`1a74d4ac`，基线`67b77b39`，20个代码/测试/契约/生成文件新增1406行；根任务另有计划拆分提交`9d98788a`。列表首次see、T11历史目录和活动查询共用独立ReadTx。SQL只访问issue模块自有表，目标和活动软删条件始终生效；五种引用field的非NULL旧/新端各须在历史目录范围且通过OwnOnly，不过滤引用issue软删/归档，物理缺失则整行略去。引用条件先于LIMIT，游标只有位置语义；DTO严格十一字段，空值与字符串verb照设计，无新错误码或写操作。

应用层与存储先有可编译业务红，定位转绿；真实API覆盖隐藏旧父编号、四种关系、管理员兼guest、项目删除前后历史不变、成员失活后隐藏和物理清理。分页覆盖同时间逆id、1/50/100、邻居/软删、实际末行cursor与末页重读；增量按id覆盖合并描述行并重排，与全量结果一致。首次目标读取、目录和查询均检查只读/可重复读，外连接提交成员失活及活动删除后当前页仍见旧快照，下一请求见新状态；持项目FOR UPDATE时仍可读取。组合测试按引用/项目删除与分页/增量/快照拆成151/216行，复用既有帮手。

四项实际SQL变异为漏新端、误加引用软删/归档条件、先LIMIT后过滤、持cursor绕过目录。前三由真实存储历史集合/limit1遍历断言失败，第四由真实项目删除后撤销成员的cursor granted reference access断言失败；每项均经gen-go生成、可编译业务FAIL、恢复生成并同定位测试通过，不计构建/夹具失败。

推进者已核对task-12-report.md、现场日志与干净VCS。最终完整Go通过（server63包及tools，bootstrap101.308s），Go lint两段0 issues，Web lint54/54、knip、test-web17/17、gen和提交后gen-check均通过。无迁移/E2E furnish/故事变更，仅Go矩阵种子扩展，本Task不适用E2E；仅Go测试修正后未无故重跑已绿Web。

真实失败保留：OpenAPI3.1空值写法、HTTP假端口误报未声明码、关系唯一/父同项目/删父级联目标的夹具、import样式均已按现有规则纠正。首轮完整Go拒绝两条使用prepare修改共享读取副本的矩阵行；保留标准归档项目行，把issue/both归档移入独立issueWorld，不改守卫。第二轮完整Go因首轮后补入的Begin失败注入测试误用共享fakeTx而失败；该替身不消费callLog.fail，现用本Task专用失败ReadTx验证原错误及零后续读取，定位通过后最终完整Go通过。未先定位新增测试便跑第二轮全套的执行偏差如实保留；前两轮make均未执行到tools，不冒称全套成功。

独立报告task-12-review.md审查67b77b39..a29450e2，规格通过、质量Approved，Critical/Important/Minor均为0。差异外边界由推进者闭合：既有Locks.see、T11目录和平台快照沿已验收实现，生成来源及全部原始门禁已核对，所列共享边界在本Task无改动；本Task已实际消费T11目录并验证首次读取起的同一快照。局部Approved不代替后续交错、安全、故事及Claude终审。

### 2.14 T13 验收与独立评审（已完成）

协作交错测试已提交为`53f1b479`，基线`f99ebe14`，七个测试/帮手文件新增695行、删除6行，没有最终生产改动。新增删除工作项与发评论、同/不同actor描述保存、八类写的锁后时钟、活动时间不倒退、成员移出/降级、评论编辑/删除与表情、评论写与项目/工作项归档七组。两种提交方向均以实际项目N锁等待驱动，时钟回调在确认等待后、放行前运行；旧入口委托同一个调度器，失败路径放行并限时回收后台写，不用固定sleep控制顺序。

断言包括锁下最终内容、描述old_value与版本last_saved_at窗口、同时间逆id的活动顺序、18个权限拒绝的五表与审计零写、完整404响应等价、删除时间/actor及旧删除行和邻居保持、活表情无孤儿。首次七组直接通过，如实记录测试先补齐的经过，不伪造首次业务红；随后三项可编译生产变异分别将Clock移到锁前、复用锁前issue/grant、用created_at代替last_saved_at判版本窗口，均取得业务FAIL并逐字恢复后定位通过。第二项的具体结果包括删issue先时后评论500、删comment先时添加表情201活孤儿、删除表情错误Problem码及成员拒绝错误成功，不能笼统写成所有场景都错误成功。

推进者已核对task-13-report.md、三份变异差异及原始日志：最终七组定位通过；七组`-race -count=5`通过（138.722s，无DATA RACE或40P01）；首次lint唯一QF1001已等价简化测试条件，修正后Go lint两段0 issues；随后完整Go通过（server63包及tools，bootstrap113.493s），提交后gen-check通过。race在该测试条件简化前运行，最终完整Go覆盖简化后代码。所有容器命令串行自然退出，无Web/契约/迁移/故事变化，不追加对应门禁。

既有P3关系写入的锁序疑点已按§3核对原裁定并记录实际资源顺序，没有更改生产锁协议或守卫。平台、架构守卫、issue生产、project/access、bootstrap生产及依赖清单相对本Task基线均未变；工作树干净、主目录仍main。独立报告task-13-review.md审查f99ebe14..7e3f3a68，规格通过、质量Approved，Critical/Important均0、Minor两项。M1为失败清理重复消费已结束请求的channel；M2为两个同时间title候选均会打断描述合并，不能判别id排序方向。两项当时登记并纳入紧接T14，现已由T14修正并独立复核ADDRESSED。差异外事项由推进者闭合：保留全部15种写的锁图及原P3裁定核对，沿用T1–T12已验收守卫和本次完整Go原始证据，恢复比对/生成检查/主目录与提交状态已核对；不以本Task验收代替T14安全补证或Claude终审。

### 2.15 T14 验收与独立评审（已完成）

权限、安全、快照及回滚补证已提交为`db7cfc10`，基线`2f89db6f`，六个bootstrap测试文件新增591行、删除18行，无最终生产/SQL/契约/Web改动。十二新操作逐一比较隐藏/不存在的完整404响应与稳定头，整份状态含五新表及审计、旧删除行；真实SQL/Authorizer配合外部EXCLUSIVE表锁及可观察gate/Tx，证明非作者403的预检不取得写资源。既有矩阵/连接/gate集合复用，issueTables精确补齐mentions与description_versions。

三组独立ReadOnlySnapshot从首次GetIssue开始：外连接提交目标权限降级及行变更后，本次应用请求仍按旧授权成功、页面逐值旧快照；下一HTTP及应用请求按新权限404；只恢复目标权限后，新请求读到确切新行。评论主查询和表情批读、活动历史引用目录和目标授权、版本列表分别有判别数据，直接应用组合与后续HTTP证据不混称。另在七个新写的十六个实际SQL写入/回读端口返回同一sentinel，验证原错误及完整状态回滚；读取/授权/锁故障沿既有fake的停止/传播证据，不冒称都执行过真实回滚。

T13 M1已通过发送后关闭结果channel修正，纯调度回归有原实现业务红与恢复绿；M2同时间候选改为合并资格不同的描述/title，精确检查id、old_value及最终值，真实SQL改id ASC后业务失败，恢复生成后绿。九类性质共十次实际实验，历史引用OR/仅旧端放行是同一性质两个变体；其余覆盖缺矩阵行、作者判断过晚、pool绕事务、ReadTx晚于首次授权、cursor改变结果域、自定义src未清洗、漏版本表连带及排序。每项均可编译业务FAIL、恢复定位绿，不把编译或环境失败计入。

推进者已读task-14-report.md、原始命令账本与十份变异源差异/失败断言。全部定位27.278s通过，Go lint两段0 issues，唯一完整Go通过（server63包及tools，bootstrap113.121s），提交后gen-check通过且工作树干净。容器命令串行自然退出；只有测试/helper变化，不重跑无变化Web/E2E，也未重跑T13整套race，受影响交错已定位及完整Go覆盖。首次M1绿运行因新security调用少参数构建失败、首次非法HTTP用例误用CheckRequest失败均已修正并原样保留；早期口头17注错已校正为实际16。无架构/平台/模块生产/依赖改动。独立报告task-14-review.md审查2f89db6f..08326569，规格通过、质量Approved，Critical/Important/Minor均0，T13 M1/M2均ADDRESSED。差异外约束由推进者闭合：沿用T1–T13已验收的操作集合/锁图/503及fake故障证据，保留本Task实际恢复范围、最终Go与生成检查、主目录main和提交状态的核对；既有首读包装、历史目录ctx、完整答复规整及清理依赖均已由评审定点确认。

### 2.16 T15 验收与独立评审（已完成，Minor由T16关闭）

故事与fixture提交为`f331174b`，基线`3bf89f02`；十二个e2e文件新增567行、删除7行，无最终Go/SQL/契约/Web产品改动。新增七个真实API用例：I11评论与两类表情，I12多页增量联合时间线及隐藏引用，I13版本分页/detail与普通还原，I2普通描述和独立href/checked审计；已有I1/I15/I17增长。所有前置经API，数据库仅事后断言；跨600秒边界沿用T8/T9固定时钟证据，不以故事等待或SQL造时间替代。

I12分别保留评论、动态末页的请求cursor，合并描述沿原id更新created_at后，相对新评论重排；按id覆盖后的联合结果与重新全量读取逐值相等。I17父/子/孙由既有删除目录验证五表同写者/时刻，额外旧删除分支和邻居的完整快照保持，timestamp text保留微秒。专用API帮手按comments/issues职责扩展；描述动态复用已有changed工厂，独立协作断言文件避免继续扩大原assert/issue.ts。

推进者已核对task-15-report.md、62条命令台账、五份源变异差异及业务红/恢复绿日志。skip误审计、忽略href、忽略checked、隐藏引用过滤失效、漏comment_reactions连带五个独立变体均先重建当前bin再运行；SQL两侧经gen-go。删除变体在旧分支的五表已删除断言被杀，没有冒称到达后续父树目录断言。首次定位直接15/15通过，恢复后定位15/15；首轮lint的十处no-await-in-loop修正后受影响故事4/4、Web lint54/54、knip、test-web17/17任务通过，后者全部缓存。完整make e2e为133通过、两个Firefox A4因可执行文件缺失无法启动，exit2；本轮无业务失败，不能写全绿。提交后gen-check exit0，工作树干净，容器套件无重叠，临时生产变异全部恢复。独立报告task-15-review.md审查3bf89f02..ff4845e6，规格通过、质量Approved，Critical/Important均0、Minor两项：I2/I13原始RFC3339Nano字符串大小比较可能因尾零截断偶发误判；通过日志仍有NO_COLOR与FORCE_COLOR冲突警告。两项登记并纳入T16，时间比较使用保留微秒的定宽值，日志警告如实记录并在已需运行时统一颜色变量，不为消除噪声单独重跑。差异外约束由推进者沿T1–T14已验收的固定时钟、锁图、快照、权限、五表目录和守卫闭合；本Task生产范围为空、变异逐字恢复、最终门禁与main状态已核对。

### 2.17 T16 文档交接及 T15 Minor 独立复核（已完成）

基线为 `499cf108246a21bcff9aa3ee20a5f34a2f17723f`，时间修复 `960b00d37fdcf3de7e79832f6c2d31df41aada79`。本Task只承接授权的12份文档、一个fixture和三个API故事，不改生产/其他测试。完整执行报告为工作树 `.superpowers/sdd/P4-issue-collaboration/task-16-report.md`，原始命令台账 `logs/task-16-commands.jsonl`，新过程文件/日志均用task16-/task-16-前缀，未覆盖前序证据。

| 项 | 实际处理与证据 | 边界 |
|---|---|---|
| T15/M1 | 先以纯值复现 `.12Z→.123Z`、整秒→`.000001Z` 原字符串比较反向（`task-16-instant-red[-boundaries].log` exit1）；抽取I12的九位补零为 `e2e/fixtures/instant.ts` 的instantKey，三故事复用。七对正/反序、微秒/纳秒、同语义时刻定点通过（`task-16-instant-green.log`），定位三故事六项通过 | UTC API RFC3339Nano窄排序键，不是通用日期解析器，不使用Date、不丢微秒、不改生产时间；独立评审已确认ADDRESSED |
| T15/M2 | 原 `task-15-lint-fix-focus.log:4/6/8`、`task-15-final-focus.log:4` 的NO_COLOR被FORCE_COLOR忽略警告保留；本已需要的定位/Web/e2e以 `env -u NO_COLOR` 运行，当前这些日志未再检出该冲突 | 不改全局环境、不为消噪单独重跑，不把历史通过日志说成从无警告 |
| 上级同步 | 总体无回复/仅描述历史/业务表44=Plane43+新增1；差异清单Plane保留43/不留53（96业务+14系统=110不变），五表逐列对照快照/迁移；PageSize共享、500000/skip/作者/动态分页等已登记 | 每工作项最新20版任务仍P9待实现；M4总体9.4及设计§14不提前完成 |
| 交接 | 五份handoff追加处理结果且status=open；spec §5/7、本文§6列出后续接收内容 | 不产生M5–M8真正收尾交接，不关闭既有handoff |
| Web G | lint-web54/54（36缓存）exit0、knip修正后exit0、test-web17/17任务（17缓存）exit0；分别 `task-16-lint-web.log`、`task-16-knip-fixed.log`、`task-16-test-web.log` | 首次knip exit2发现本Task忽略目录两个纯值.mjs过程脚本unused，原脚本改存.mjs.txt并保留失败日志，不改knip配置/守卫 |
| 故事 | `task-16-focus.log` 6 passed；完整 `make e2e` 的 `task-16-full-e2e.log` 133 passed/2 failed，make exit2 | 两失败分别Firefox A4 navigator.locks/localStorage lease缺firefox-1543可执行文件、均launch前止；没有新增业务失败；未安装浏览器或重跑全量 |
| 收口门 | `git diff --check`及相对链接/新fixture文件名核对，最终提交3c0446ea后gen-check自然exit0（UTC01:16:36–39）；Go lint/test沿最近T14、race沿T13 | 本Task无Go变化，不重复Go/race；T16独立审查已通过；分支CI的E2E的S2失败待定位、Claude终审待完成，不以自查冒充 |

首次定位的过程包装器因从e2e cwd推导日志目录错误，在启动Playwright前即FileNotFoundError；修正记录器根目录后仅启动一次定位套件，自然退出后才执行完整e2e。首次纯值脚本遇首个反例即退出，随后保留原日志并另跑两边界合并红；均是纯值业务断言，未把包装器错误当测试红。所有真实开始/结束/退出见task-16-report，不删除历史失败。

推进者已读完整task-16-report.md、原始红绿/18条命令台账及实际提交，最终文档后lint-web54/54（38缓存）、232处相对路径/11个新增锚点检查、提交后gen-check均通过，工作树干净。独立task-16-review.md审查499cf108..3c0446ea，规格通过、质量Approved，Critical/Important/Minor均0；T15 M1/M2逐项ADDRESSED，当前无未关闭任务级发现。差异外的spec前八项沿本文T1–T15逐项验收、原始变异/运行记录与T14最终Go/T13 race闭合；本Task五表文档静态对应及时间修正证据已由审查者定点核对。本Task最终无新依赖或生产差异，架构守卫未改，原始失败与Firefox限制仍保留。

### 2.18 尚未完成的 P4 验收

T1–T16实现及独立Task评审已完成；负责人已推送a0526ade，分支CI 40170883949的Web/服务端通过、E2E的S2失败待定位；root自主推送授权尚未恢复，Claude整分支终审及其发现修复轮未执行。完整e2e的两项Firefox环境失败未通过。M4设计§14不新增勾选，总体9.4仍进行中，P4不标最终完成；没有PR、merge或清理。

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
- 为满足 P4 读目录与查询在同一快照内，已把设计批准的 `ReadOnlySnapshot` 从 P5 前移到T2，并以真实库证明与现有 `LockTimeout` 的开始路径兼容，T3/T10/T12/T14已消费，交P5复用去重。
- 提及资格须按被提及成员自身的可见性判定，不能沿用发起人的 `OwnOnly`；在既有 `ProjectMembers` 目录职责内细化，独立预检确认可沿原有模块边界实现，现已由T3实现、T9复用并经T14补证。不得改变 P2 对 `ProjectGate.Lock` 返回值的约定，发现额外跨模块协议变化时仍按负责人纪律处理。

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

### T13 测试时钟回调与既有P3锁关系

在既有serializeOnTable调度层增加afterWait回调，确认真实项目行锁等待后、放行前推进测试时钟；上层serializeAt及路由装配沿单一实现委托，不在生产Deps增加测试接缝。原调度器失败路径也须打开闸门并限时回收正在等待的后台请求，避免失败测试遗留写入。

锁关系复核发现既有create_relations/delete_relation先写relations再MarkChanged(issues)，没有把它误记成全局表序的子序列。该顺序在[P3 spec](../specs/P3-issue-hierarchy.md) §3第3条已获负责人裁定，[P3 review](P3-issue-hierarchy-review.md)记录双方项目N先行的证明：会连带同一关系行的写在任一端项目N先相遇，不会各持关系锁再反向等待另一端项目。设计§3.5中单端连带删除按id预锁是另一条既定证明，不能扩成新写逆序许可。推进者核对原裁定及相对P4原基线未变的生产文件后，确认本Task没有引入新锁边，更新plan的复核措辞并继续验证；P4新写仍按全局表序，架构守卫不变。

## 4. 整分支评审的发现与处理

尚未送 Claude 终审。A1 是实施前发现，负责人已批准 A 方案。T16独立审查已通过；负责人已推送a0526ade，分支CI的E2E的S2失败待定位；root自主推送仍受钥匙串授权阻塞。按用户要求交付分支 `codex/m4-p4`、原基线 `56d575fb8ff6575880d1edef65219728a81f2b12`、spec/plan/本review和106次变异台账，等待负责人安排Claude整分支终审。终审发现按一个修复轮处理，必要时复审；未获批准前不合并、不push main或清理。

**分支CI推送阻塞（2026-10-25）。** 推进者在可审提交3c0446ea使用系统`/usr/bin/git push -u origin codex/m4-p4`，UTC01:20:59–01:21:00自然exit128，错误为`failed to get: -25308`及`could not read Username for https://github.com: Device not configured`。当次无成功推送、无分支CI运行，不能归因W11或借用原main CI成功。原始记录在工作树`.superpowers/sdd/P4-issue-collaboration/logs/task-16-root-push-first.log`及`task-16-root-ci-commands.jsonl`。已请负责人自行恢复终端GitHub认证，不发送密码/token；遵守[03规则§3](../../claude_to_gpt/03-rules-and-pitfalls.md)“不去钥匙串或别处取凭据”。在认证恢复前保留本地分支、完整提交和终审材料，尚未合并。

**分支CI实际运行与自主推送复核（2026-10-25）。** 负责人通过SSH终端推送后，远端分支确认为 `a0526ade19f123b526b78a44dfd4f6bce9b3c99a`，触发 [CI 40170883949](https://github.com/open-nerve/NerveProject/actions/runs/40170883949)（push、attempt 1、UTC06:56:05开始，07:10:38结束状态failure）。Web、server成功；[e2e作业](https://github.com/open-nerve/NerveProject/actions/runs/40170883949/job/121102658672)仅E2E步骤失败，公开annotation只给 `Process completed with exit code 2.`，没有失败用例名。未自行读取作业日志/产物、未重跑；当时已按03规则请负责人提供失败用例与错误文本；不能把这次失败预判为W11或本地Firefox缺失。公开摘要和annotation分别保存于工作树证据目录的 `git-auth-ci-check-public-summary.json`、`git-auth-ci-check-public-annotations.json`。

**负责人提供的失败日志。** 本轮E2E为134通过/1失败；唯一失败是Chromium的 `e2e/stories/smoke/s2-web-app.spec.ts:319`：工作区菜单中“Leave workspace”已可见，按Escape后5秒内该menuitem仍为1，预期0。Firefox A4通过。因此本轮不能归为W11，也不是本地缺Firefox；菜单未关闭的根因待定位，尚不判断是产品行为还是测试时序。已启动只读诊断，未扩展为无关W11调查；原始用户日志节选保存于 `logs/task-16-ci-s2-owner-excerpt.txt`。

同轮root使用系统git的普通push、PTY dry-run和显式指定系统osxkeychain助手的dry-run均报 `-25308` / exit128；`ls-remote`和fetch成功，公开读取不证明认证推送恢复。未改变Git配置、未读取凭据；实际命令与起止记录见 `logs/task-16-root-ci-commands.jsonl`、`logs/git-auth-fetch-check-20261025.json`、`logs/git-auth-explicit-system-helper-check.json`。负责人说明SSH操作没有看到授权弹窗，已指导通过本机或屏幕共享的图形桌面为系统Git助手完成正常钥匙串授权，之后必须由root实际推送验证；当前不能声称自主推送已恢复。此后本地文档提交不在上述CI的精确SHA覆盖范围内，恢复推送后另记最新HEAD的运行。

## 5. 最终实现与 plan 的差异

T1实施中增加了CustomColor既有解析的局部根因修复，以及颜色属性再次生成CSS的值检查，已同步spec/plan，依据见§3。T1–T15实际交付见§2；本次未做原型/代码逐字重放。其余最终差异与规模如下。启动变化为 `tools/keywords.json` 的phase和工作树忽略规则（`05841220`）；`573e8eb6`保存A1暂停草案，`391c76e9`统一批准方案，`cbe0a9fc`闭合预检修订。

### 实际交付规模与依赖

统计为列出实现/修复提交的numstat累计（含生成物/契约），文件数取并集；root证据文档提交不混入T1–T15，不当作最终净差异、工作量或变异覆盖率。

| Task | 实现/修复提交 | 文件数 | 新增/删除 | 依赖变化 |
|---|---|---:|---:|---|
| T1 | 6cd23ca4、18381d56 | 19 | +1594/−32 | bluemonday v1.0.27、x/net v0.60.0；间接douceur v0.2.0、gorilla/css v1.0.1 |
| T2 | 91be8ecb | 12 | +394/−47 | 无 |
| T3 | 98dcbc3e | 69 | +2270/−52 | 无 |
| T4 | de2c43ed | 34 | +1387/−33 | 无 |
| T5 | 8470a689 | 64 | +2192/−44 | 无 |
| T6 | a422cd91、48bb532c | 68 | +2210/−126 | 无 |
| T7 | 2eb31d8c | 25 | +617/−170 | 无 |
| T8 | c8f56df9 | 11 | +503/−2 | 无 |
| T9 | bd1d5fb9 | 68 | +1742/−118 | 无 |
| T10 | ed472ff3 | 32 | +1648/−57 | 无 |
| T11 | 80d76256 | 11 | +299/−13 | 无 |
| T12 | 1a74d4ac | 20 | +1406/−0 | 无 |
| T13 | 53f1b479 | 7 | +695/−6 | 无 |
| T14 | db7cfc10 | 6 | +591/−18 | 无 |
| T15 | f331174b | 12 | +567/−7 | 无 |
| T16 | 960b00d3、3c0446ea（精确SHA/numstat见task-16-report.md） | 16（4代码/故事、12文档） | +219/−68（代码+12/−9；文档+207/−59） | 无 |

T7/8内部类型与决策由T9消费，T11 Historical由T12消费，无空端口。T3/T9实际规模较大包含契约/生成物，独立评审未要求再拆；保持16Task，没有新增第17Task或缩减scope。新时间fixture仅3行；既有契约/生成物和共享S3数据契约的大小例外沿原裁定，未放宽其他逻辑约400行规则。

## 6. 移交事项

| 接收方 | 已交付及后续责任 |
|---|---|
| P5 | 复用shared.PageSize、已与LockTimeout组合的独立ReadOnlySnapshot、Historical历史目录；普通VisibleProjects/列表引擎、force_custom_plan、替换P3 seeEach/ArchivedProjects仍归P5，不重复实现前移能力 |
| P6 | 描述/评论/表情事件与任务尚未实现；skip=true且内容标识相同不发布，其他每次写事件共用当前锁下时刻；不得为P4增加空事件端口；只投递客户端/worker完整性按M2交接继续 |
| P9 | 最新20版保留、60天物理清理（含五表/comment FK/PAT）、项目/工作区异步删除、删除断言限时轮询；P4只交同步连带/键集批量语句，不声称任务正在运行 |
| P11 | 新DTO与service/store换型、description_json/comment_json及JSONContent一起清理；历史列表/单版本新地址、评论内嵌表情、分页契约 |
| P14 | 保存撤销再重做、创建/编辑弹窗isMigrationUpdate、只在用户改父项时发parent_id；I1/I2/I15/I17页面版本 |
| P15 | I11–I13页面、作者编辑/管理员删除；保留产生末页的请求cursor重取，按id覆盖/重排；别端旧评论编辑/删除须整体重读 |
| M5（经M4收尾） | 图片开启才把按顺序image-component[src]纳入内容标识并扩S3；标签放行不代表P4开启上传 |
| M6（经M4收尾） | 继续消费furnishIssueRows与目录删除断言，新表建出同步API前置；后续异步删除需限时轮询。P4五表已接入，未提前完成迭代/模块前端 |
| M7/M8（经M4收尾） | 关注只在加入瞬间可读，通知发送前再判权限；评论/表情不写动态，通知从P6事件；stripped只做摘要/Webhook、不作搜索。收集箱来源和旧显示分支按M1-P3原交接继续 |
| M4真正收尾 | 五份交接及其余输入共八份统一逐项核对closed，浏览器C12、整体性能及M5–M8交接；main CI与W11按06，当前不得关闭或冒称M4完成 |
| root/负责人 | T16独立复核M1/M2及12文档已通过；已核验a0526ade的分支CI，E2E的S2失败待定位；恢复root自主推送授权后继续验证，不必建PR；给负责人整分支材料并等待Claude终审。后续只追加真实run/审查记录 |

原spec §7的P1b/P2/P3输入已分别由T7–9/15、T3–6/10/12/14/15、T9/11–14消费。五份handoff只追加“处理结果（M4/P4）”及明确余项，均保持open；原关闭条件未删除，M4§14和总体§9.4不提前完成。

## 7. 已知限制

- P4实现/适用本地验证及T1–T16独立评审完成，分支CI 40170883949的Web/服务端通过、E2E的S2失败待定位，Claude终审待安排；本地完整e2e两项Firefox缺失仍未通过。M4仍有P5–收尾，负责人已推送a0526ade，root自主推送仍报钥匙串-25308；没有PR、合并或清理，也没有P4合并main CI。原main CI342/343成功仅是历史证据，不能替代本分支CI。
- W11 既有偶发失败按 [06](../../claude_to_gpt/06-ci-w11-flake.md) 的裁定不调查；未来某次合并被它阻塞且重跑仍失败时再上报。
- 独立预检及六项局部修订的定向复核已完成；只代表计划层可执行，不代表产品测试或阶段终审通过。

- 文档相对路径核对232处，无新增缺失目标；M4设计原有四个M5–M8 `handoffs/M4-closeout.md` 链接指向未来收尾产物，当前尚未创建，按本Task边界保留，不能据此声称已完成转交。
