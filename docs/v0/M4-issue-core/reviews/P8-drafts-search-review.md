# M4/P8 草稿、搜索与链接计数评审记录

日期：2026-10-27。**P8已完成集成**：最终分支 `cde70d9735994cb6a579e0b800469f32631e9981` 经授权 `--no-ff` 合并为 main `6b9d08d4821f1423c7bb9795557727c206b7880f`；[main CI 40520609987](https://github.com/open-nerve/NerveProject/actions/runs/40520609987) attempt1的server/web/e2e均completed/success，归档核验及分支/worktree清理完成，完整原件见§9。实施基线 `166b9efd342ecd69b0ad8f29e4cd0ae2a65b1f5d`、产品HEAD `10f44790ccb842e758f0ab50ecb4d294dc15a6e6` 与原审查范围不变。配套：[spec](../specs/P8-drafts-search.md)、[实际plan](../plans/P8-drafts-search.md)、[首次整体审核](../../../reviews/re-07-M4P8整体审核.md)。

§1–7保留准备期记录及当时的实施接收追加，§8保留实施/首次整审与文档独评时的历史快照；其中“待最终CI”“未授权合并”“不清理”描述当时边界，由§9关闭相应待办，不改写原审查结论。[re-06预检](../../../reviews/re-06-M4P8预检审核.md)不替代§8实施证据，re-07和T16独评也不冒充对本次集成回填的审查。

## 1. 评审方式

按 [02 方法](../../claude_to_gpt/02-method.md) §3 的可调整做法，沿 P7 使用紧凑 spec/plan 和独立静态预检。先盘点设计 §12/P8、前序 Phase 与输入 handoff，再由唯一作者编写规格/计划；主控核范围、接口和事实，独立审者分别追踪草稿与搜索/计数的逻辑、集成和验证方案。没有实现完整原型、逐 Task 重放或运行产品变异，不能用静态结论替代实施期证据。

准备期本地 ignored 原始材料位于主项目根目录 `.superpowers/sdd/P8-drafts-search/preparation/`，后文简称 `E`；该准备目录仍在原处，从未位于P8 worktree，不属于§9本次归档。`requirements-inventory.md`、`code-inventory.md` 是准备输入；作者报告和独立报告各自保存读取范围、源码/文档版本及限制。

## 2. 验收标准逐条核对

本节保留准备期责任表和当时未实施状态，当前逐项实施结果在§8。

P7 已按授权 `--no-ff` 合并，准确 [main CI 40451632280](https://github.com/open-nerve/NerveProject/actions/runs/40451632280) 全绿；5,443 个证据文件归档校验后，P7 本地/远端分支及 worktree 已清理。此事实只证明 P8 基线可用，完整集成回执见 [P7 review §8](P7-issue-links-review.md#8-授权合并main-ci-与归档清理2026-10-27)。P7 收尾文档另经独立只读审核，`E/p7-receipt-review.md` 无 High/Medium/Low 发现。

| P8 验收项 | 准备期状态；实施责任 |
|---|---|
| A1 草稿表、grants、项目/工作区级联 | 未实施；T1/4/8，含无项目草稿、零项目工作区和旧删除审计 |
| A2 保存结构、替换/noop、NUL、游标/total | 未实施；T2–4，不提前执行创建业务验证 |
| A3 六操作权限与不可见性 | 未实施；T3/4/6/7，逐值404、guest/owner顺序与拒绝零锁 |
| A4 发布与直接创建一致、完整副作用及回滚 | 未实施；T5/6/8，锁下存储载荷、唯一核心和同一时刻 |
| A5 交错16/17及完整锁图 | 未实施；T7，真实等待链和两种次序 |
| A6 扩展、迁移/runtime角色和生成类型 | 未实施；T9/11，非超级用户 owner 与实际 sqlc 输出 |
| A7 搜索语义、排除、授权和快照 | 未实施；T10–12，递归穿过隐藏中间节点但不返回它们 |
| A8 全部 Issue 表示的 link_count 与两方向排序 | 未实施；T13/14，三列表 SQL、游标、成员列表和偏好 |
| A9 I14/I18 API 与 I4 两方向故事 | 未实施；T8/14，前置仅经 API |
| A10 10 万英文/中文查询测量 | 未执行；T15，记录最终 SQL/参数、locale、EXPLAIN 与实际耗时 |
| A11 检查点完整本地门禁、整分支 deep-review | 未执行；T4/8/12/15 检查点及 T16 终审 |
| A12 正式同步和阶段交付 | 未实施；T16，先不合并，交接总体 status 保持 open |

准备期实际运行：`node tools/keywords.mjs` 通过（72规则、23例外、零命中）；`go -C server test -count=1 ./internal/archtest` 通过。原始输出与命令/时刻在 `E/checks/guard-results.json`、`keywords.log`、`architecture.log`。首次文档检查遍历118个本地链接/锚点，114个有效，4个是基线已存在且约定留M4收尾创建的 M5–M8 `M4-closeout.md` 目标；没有新增坏链接。`git diff --check`通过。仅docs/phase变化，未运行P8产品测试、迁移、生成、性能或完整本地CI，不冒称实现验收。最终回执的文档检查见同目录`docs-check.json`。

## 3. 准备期决定

设计的 13 个粗项细化为 16 个 Task，保持 D2 范围；不通过减少权限、交错、变异或完整检查点来缩小任务。具体局部结论、理由和错判代价在 spec §3。草稿的项目 S 锁与 workspaceID 下传都是已批准交接的兑现，未引入新的跨模块协议。

准备期同步设计/硬规则的适用边界：普通工作项写仍首先取项目 N；草稿采用已批准的 W/P/draft 顺序并复用集中锁依赖。保存与搜索的 NUL 表示限制、旧项目删除404/新参数422和 getDraft 的403按具体契约记录；未放宽架构守卫或新增依赖。

## 4. 预检发现与处理

两位独立审者均全文读完冻结spec的306行和plan的257行，并追读对应现有源码/测试。草稿分区报告 `E/drafts-preflight.md`、`drafts-coverage.md`；搜索分区 `E/search-preflight.md`、`search-preflight-coverage.md`。两分区均 **Approved（静态计划范围）**，开放 High/Medium/Low 均为0；无 Structural Issue/Needs Decision。

准确审查锚为准备提交 `d4391bffc6fe22b7aea6db7288c5ea039fc45bfd`，spec SHA-256 `4feaacbf9bdbad3b481d88ca1ba7f3d58a03475f077c796b73f3ce8d0814aeaf`、plan SHA-256 `59ceff068545bd53d4e2f807b7330ad65884e8daab729a8a502a4d42233a2913`。合并重复NUL记录后共11组局部修订（9组Medium、2组Low），全部关闭；原始编号保留在独立报告，汇总见deep-review报告。

| 汇总项 | 根因修订；实施验证责任 |
|---|---|
| D1 / Medium | 创建核心公开构造、私有锁内方法，module单实例共享；T5/6验证旧创建与发布副作用。 |
| D2 / Medium | W后无额外锁重读存在性，明确删除/降guest判定时点；D重读后归档先于decoder；T6/7交叉反例。 |
| D3 / Medium | T4及时扩普通issue/draft CRUD/publish三类锁形状，union覆盖全写操作，无豁免；T6补发布行。 |
| D4 / Medium | publish-only HTTP小守卫拒非零body，认证先行、读首字节、不改bodyshape；T6真实handler链。 |
| D5 / Medium | 保存显式项目用既有project.read/LevelVisible，分别保留草稿存在目录和搜索成员目录；T3 public非成员/归档/private反例。 |
| S1 / Medium | 日期none互斥与索引错误路径明确；T10/12两字段与三种组合。 |
| S2 / Medium | generated binding ErrorHandler先解析raw，再处理重复标量等；T12畸形编码与重复参数同现。 |
| S3 / Medium | numeric CASE测试源补link_count独立列，两个计数精度及10→11样例/19→21码同步；T13实际PG/生成验证。 |
| S4 / Medium | 三种存储关系×双向六候选及软删边对照；T11/14集合和变异。 |
| DOC1 / Low（D6/S5合并） | 修正不存在的测试路径、Cyrillic字母、原始NUL；冻结稿字节与路径已核。 |
| DOC2 / Low（D7） | 上级设计明确guest403与他人404、D重读先于显式新项目验证，03同步草稿锁形式；定向复核关闭。 |

另将非超级用户扩展/运行角色、pg_depend区分扩展成员与严格应用函数目录、预存扩展OID保持、显式空for拒绝等落实到T9/T10。未更换模块协议、锁顺序或依赖；尚未取得这些未来测试的执行证据。

最终纯回执定向复核另有1项Low（FINAL-DOC-1）：预检报告把“原型/逐Task重放未做”与“实施必须取得的证据”连写，可能误读成未来须补完整原型。现已分开表述，仅要求计划列明的生成、交错、变异和性能证据；独立审者确认ADDRESSED、开放发现0。报告为 `E/final-receipt-review.md`，此项不改写冻结准备稿的11组发现统计。

## 5. 与计划方法的差异

本轮仅做可执行规格、任务拆分和静态独立预检；不生成产品文件、不预造生成物 SHA、不写虚构的测试红绿/性能数字。实施时按 Task 记录实际命令、测试选择器命中数、生成 SHA 和变异恢复证据。P7 的测试/CI 不作为 P8 实现验证。

## 6. 移交事项

预检已关闭，由负责人安排启动 T1；本轮不自动实施或合并 P8。P9 接异步删除/清理，P10b/P11/P12/P14 接前端数据层及选择器，P15 接草稿页与命令面板。输入 handoff 仅在实施完成时追加真实处理结果；M4 总体仍进行中，M5–M8 最终 `M4-closeout.md` 留 M4 收尾。

**P8实施接收回执（2026-10-27，集成前历史）：** T1–T15已完成，四份输入及P1a–P7 spec§5/review§6只追加本期承接；M0扩展服务端验证、M1无page、M2游标/存储结构、M3锁/矩阵/级联子项已有§8实证。四份status仍open，P9/P10b/P11/P12/P14/P15/M5–M8及M4最终收尾保持原归属。P5当时19码历史不改写，当前21码另附P8回执。未合并、不清理分支/worktree。

### 处理结果（M4/P9，2026-10-28）

T10实际runtime角色等唯一workspace_drafts任务completed后核两个草稿原At/By/邻workspace不变；W3 Old未完成与零项目直接目录两处均迁移issue有限poll、M3即时断言。T11六draft操作/search窗口、T12交错16/24/25本次P/Q与Old双审计、T13删除故事接续；目录无新豁免。

证据：[P9 review §8–12](P9-purge-review.md)、[P9 spec附录B/C/D](../specs/P9-purge.md)；产品身份 `e003e96625ba51264d3aeb77729729d53a770103`，独立整审与两次局部复审结果按上述入口，最终检查点/文档独评分别记账。

仍开放：P10b–P15数据层/选择器/草稿页/命令面板、M5附件同事务/内容标识、M6/M7真实字段和M8扩展维护仍开放；P8集成/归档事实不倒改。 本回执只承接P9后端子项，不关闭整份交接或M4，不改变本页历史运行/集成身份。

## 7. 已知限制

静态审查不能证明新增 SQL 的真实执行计划、生成类型、运行角色授权或并发时序；对应验证必须在实施时取得原始证据。真正的架构回归、跨模块协同方案变化或意外高风险仍按负责人纪律先提交已完成工作，再上报决策。

准备提交后的回执只更新预检状态、报告与接续链接，不改已审接口/任务方案；产品、迁移、生成物和测试源码与基线相同。后续产品实施仍须逐Task验证及最终整分支deep-review。

## 8. 实施验收与首次整审（2026-10-27）

本节为集成前历史快照，保留当时的证据范围与待办；最终分支CI、授权合并、main CI和清理结果见§9。

### 范围、身份与证据入口

首次整审准确范围为 `166b9efd342ecd69b0ad8f29e4cd0ae2a65b1f5d..10f44790ccb842e758f0ab50ecb4d294dc15a6e6`：18提交，186文件、55171行、3062898字节。基线是已集成P7的main；产品HEAD包含T1–T15及T6修复，T16文档提交不冒充产品测量运行时HEAD。分支 `codex/m4-p8`、工作树 `.codex/worktress/codex/m4-p8`，主目录仍main。

下文执行时原件短名 `E8=.superpowers/sdd/P8-drafts-search/`（当时本地ignored、尚未归档；当前归档根见§9）；每个 `checks/<name>/result.json` 和 `output.log` 保存准确argv/cwd、前后HEAD/树、源码hash、时刻、exit和原日志hash。`task-N-author-report.md`、`task-N-review.md`（N=1..15）分别为作者与独立要求/质量审查；不能以作者自审代独评。Go RUN含父/子测试，非独立性质数量；编译0 RUN、零匹配、lint/生成0 RUN均不当行为通过。

### A1–A12逐项结果

| 条款 | 实际证据、结果与限制 |
|---|---|
| A1 | T1九列/三索引/FK/JSONobject/全授权、workspaceID贯穿及真实批量/回滚；真实schema缺表红，M1无项目/M2旧审计红→恢复绿。T4 furnish全部5调用点走API、动态目录无豁免；T8 I14三故事、P4/W2/W3六故事、bootstrap28/store13 RUN通过，零项目与旧删除审计有HTTP/DB证据。T1影响6602 RUN含原有3 skip。 |
| A2 | T2 domain5/PG1 RUN，个人页/total/未知或他人cursor、整替换/JSONB等值；M3 owner/M4 NUL/M5 merge真实红绿。T4 HTTP Shapes三态/null/封闭结构/noop/三NUL精确路径且snapshot零变化，保存业务坏值仍成功且无issue副作用；影响2188 RUN（1既有numeric probe skip）、最终bootstrap聚焦86/矩阵32。 |
| A3 | T3最终聚焦60、access/arch70 RUN，真实project.read可见public非成员/归档/退出/private边界；T4五操作30角色格及完整404对照，T6第六发布行/OwnOnly/adminGuest；T7 W/P/D终态与预检零gate/写锁、成功CRUD不入gate。M6 SHARE→N、M7先owner等反例有真实锁/授权证明。 |
| A4 | T5 core baseline22直接绿，decoder12，恢复后app/HTTP/arch1327、组合24；M11绕结构、M12漏version/event均真实红。T6恢复组合127、app/HTTP/arch1353、最终HTTP201；M13锁前payload/M14漏软删/M15二次时钟红→恢复。三真实DB/事件故障逐表snapshot、project counter和jobs全回滚；I14重复发布404。同canonical存储字节比较400，未外推原始键顺序。 |
| A5 | T7聚焦82 RUN、交错16/17 `-count=5 -race`65、既有writer/gate影响116、app1104。记录请求PID→holder、两级等待链、旧新UUID两方向、W后存在性/权限、P/D后404/409、不追新项目、gate连接0、D后唯一时钟。M16–19逐个行为红→恢复；T15最终源再race65/65。完整图见 `task-7-lock-graph.md`。 |
| A6 | T9 runtime1/migration3/type1 RUN及影响4709（3旧skip），非superuser owner+数据库CREATE安装、runtime真实similarity/草稿CRUD且DDL42501；预存扩展Down/Up OID保持，应用同名函数仍在严格目录。M20漏schema为实际生成interface{}→float32编译约束红，M21漏grants为真实权限红。T11同名查询扩完整SQL后再次验证原生float32和runtime。 |
| A7 | T10完整聚焦102、domain/shared580 RUN；M22 rune→byte、M23溢出/宽编号、M24重复末值红绿。T11采用 `t11-final-M25–M29`，最终聚焦52、影响1945（1944PASS/1旧skip）、快照26/arch52；真实全深度穿隐藏中间节点、双向三类关系、过滤先LIMIT、目录/source/seen同RO/RR。T12 HTTP31/HTTP包225、完整矩阵2219、真实契约468、影响6952 RUN；M30–32逐值HTTP红绿。命令中不存在的TestContract不计证据，真实7组契约另有收据。 |
| A8 | T13全表示活links/metadata、当前21码、独立child/link numeric sentinel（2^53及bigint）与伪造游标权限；M33–36在真实PG/HTTP分页层红→恢复。影响2518/2517PASS/1旧skip，最终PG355/354PASS/1旧skip，bootstrap260/259PASS/1旧scale skip；强制web18/18零缓存。P5历史19码原样保留，仅追加当前21。 |
| A9 | T8 I14三故事；T14先build后I18四+I4两共6故事、I14再3故事通过，API前置、独立0/1/2/2双向limit2期望。M26反向关系漏排与M33缺活link谓词均gen/build真实binary红→源恢复/gen/build同故事绿；最终58输入hash与88ba09fa及binary相符。T13原I3/I4两故事适配与T14非零计数故事分开。 |
| A10 | spec附录保留13场景+父测试14/14、精确100000=5万英文/5万中文、17typed实参、metadata/OID/EXPLAIN与每场景first-observed+5warm。PG18.6 Debian aarch64/libc/en_US.utf8，非superuser runtime，owner seed/ANALYZE；计时为Store+RO/RR，不含HTTP/目录。dirty88ba09fa+测试字节→最终10f44790的20项源码hash映射成立，不伪称clean提交后实测；八标题场景顺扫、单项目login bitmap，不造cold-cache或固定延迟承诺。 |
| A11 | 下表四次本地完整七门禁均准确提交/clean且先于push；两码/双语集合、架构/关键词未放宽。首次整审四分区全覆盖，产品无发现；文档独评10f44790..a65e8605已Approved，task-16-review.md确认历史D1 ADDRESSED，A11现有真实闭环证据。最终准确HEAD CI仍待主控另核。 |
| A12 | T16同步30文档：P8三件、正式re-07、M4/v0/Plane/README、03/04、四输入与P1a–P7十六份追加回执。Plane对照实际本地源码；四输入保持open，M4/后续未关闭。文档提交与独评、最终准确HEAD CI由 ignored `task-16-final-checkpoint-receipt.json` 及最终交付消息绑定，未合并。 |

### 检查点与CI

每次按 `make gen-check`、`make lint-go`、`GOFLAGS=-p=1 make test`、`make lint-web`、`make knip`、`TURBO_FORCE=true make test-web`、`make e2e` 串行，全部exit0；前后HEAD/工作树不变。make test非-v，只报告51个测试包通过，不能捏造case数；强制web每次18/18任务、0缓存。

| 检查点准确HEAD | 原件入口（E8） | E2E实际通过 | 产品分支CI |
|---|---|---:|---|
| T4 `9b4cfbe9d758f06106a232f53ba4e7deff7952ad` | `t4-f-9b4cfbe9-full-checkpoint{,-summary}.json`、`task-4-checkpoint-receipt.json` | 153（151 Chromium+2 Firefox） | [40467017174](https://github.com/open-nerve/NerveProject/actions/runs/40467017174)，attempt1 server/web/e2e success |
| T8 `297755a870ebea9ee86338f90ccec3df2e80afe5` | `t8-f-297755a8-full-checkpoint{,-summary}.json`、`task-8-checkpoint-receipt.json` | 157（155+2） | [40480068316](https://github.com/open-nerve/NerveProject/actions/runs/40480068316)，attempt1三任务success |
| T12 `a1f78432daecc589ccbf13108489f7275c56c50f` | `t12-f-a1f78432-full-checkpoint{,-summary}.json`、`task-12-checkpoint-receipt.json` | 157（155+2） | [40493082409](https://github.com/open-nerve/NerveProject/actions/runs/40493082409)，attempt2三任务success；首轮浏览器安装18分10秒后job超时保留历史，不改写为产品失败或通过 |
| T15 `10f44790ccb842e758f0ab50ecb4d294dc15a6e6` | `t15-f-10f44790-full-checkpoint{,-summary}.json`、`task-15-checkpoint-receipt.json` | 162（160+2） | [40508120315](https://github.com/open-nerve/NerveProject/actions/runs/40508120315)，attempt1、准确HEAD/branch，server/web/e2e均completed/success；最终原件 `ci/t15-10f44790/20261027T131510624014Z/` |

T15各门禁耗时分别2.725/1.535/358.547/3.783/1.800/13.870/44.601秒；E2E故事自身43.1秒。system `/usr/bin/git` push及准确remote SHA原件在各receipt；本T16作者不push、不重跑已通过产品套件。

### 审查、根因修复与真实方法差异

T6独评Important I1发现发布初读后第二次项目预读消失被混成project404。`7f970ebd..8b8d71c6`两文件局部修复区分物理消失draft404与仍存在但create不可见project404；新增用例3 RUN先真实红再绿，app1104/组合9与独立定向复审关闭。未改普通NewLocks五参、锁图、gate或契约。前序各Task报告中的已解释fixture/编译/lint错误原样保留；它们不充当性质证明。

首次整审仅一次，逻辑/集成安全/测试覆盖三个镜头按职责覆盖全部186文件及必要调用链，生成物/迁移/fixtures/旧回归均未跳过。drafts64/10437行、search-list71/12676、contracts-e2e39/24655、docs-guards12/7403；`whole-review-*-report.md`/`coverage.json`与 `whole-review-coverage-verification.json`全范围/hash核验。三个产品分区Critical/Major/Minor均0；文档分区新增0，历史Minor D1一项（M4§11.2摘要误列版本保留无项目锁），本次按既定§3.5/3.14修正，P9待实现。D1与T16文档增量已在10f44790..a65e8605独评Approved，task-16-review.md确认D1 ADDRESSED，未虚设第二轮产品整审或产品修复。正式结论见[re-07](../../../reviews/re-07-M4P8整体审核.md)。

性质原件独立审计见 `mutation-audit-through-t14.md`，覆盖M1–36及T14真实M26/M33二进制变体；T15新增100k OwnOnly真红→原字节恢复14/14绿。保留T2恢复SQL+生成备份的历史、T3恢复时序、T7未跟踪hash局限、T9交叠收据不采而用稳定恢复后收据、T13 PG后import-only差异经精确hash重建关闭，不为表面统一重造历史。

T5/T7/T8/T14正常产品初轮本就绿；缺接口编译红与测试作者错误不冒充行为红。T8零项目HTTP证据在W3，复用已有bootstrap/目录；T13已承担全表示Go计数，T14只四测试/fixture文件。T15无需优化，条件性优化未运行；性能默认skip不是实测通过，受影响PG+arch408 RUN/406PASS/2skip分别为旧numeric probe和性能opt-in，最终race65/65、P5四项4/4。实际Files、任务归属与计划勾选见plan执行回执；没有无关产品修改或重复全F。

### 剩余边界

文档独评准确范围 `10f44790ccb842e758f0ab50ecb4d294dc15a6e6..a65e86056b1fe60b7dcef88b8867e7acfaacb703`；独立审者 `/root/p8_design_full_context` 的 `E8/task-16-review.md` 为Spec/Quality Approved，新增Critical/Important/Minor均0，历史D1 ADDRESSED；覆盖和原件核验在 `task-16-review-coverage.json`。据此勾A11/T16复审。当前仅补录这些已发生状态，仍由同审者快速核本增量；最终准确提交gen-check/push/CI由主控负责，集成须另有授权。

本次文档增量10f44790..a65e8605已独评Approved、D1 ADDRESSED；后续状态补录的窄复核与最终准确HEAD CI另核，不预宣称通过。性能仅所列受控分布、稀疏assignee/label/relation和Store路径，first-observed非物理冷缓存；跨请求计数/游标弱快照仍在。Ryuk既有Shell警告与受支持回退保留，不称pristine。P9异步删除/保留/清理、前端P10b–P15、M5附件及M8部署维护均未实施；M4§14/总体§9.4与最终M5–M8 closeout不提前完成。用户未授权合并，分支与证据保留。


## 9. 授权合并、main CI 与归档清理（2026-10-27）

### 集成身份与准确CI

负责人授权后，最终分支 `cde70d9735994cb6a579e0b800469f32631e9981` 从基线 `166b9efd342ecd69b0ad8f29e4cd0ae2a65b1f5d` 经 `--no-ff` 合并为 main `6b9d08d4821f1423c7bb9795557727c206b7880f`。两个父提交依次为上述基线和最终分支HEAD；合并tree `68572b15efee737383da9be8d9d93df289d555d6` 与已审分支树相同，合并后main干净。系统 `/usr/bin/git push origin main` exit0；原件为 `premerge.json`、`merge.json`、`push.json`。

最终分支的状态增量已由 `task-16-status-review.md` 独立确认Approved；准确 [分支CI 40515294403](https://github.com/open-nerve/NerveProject/actions/runs/40515294403) attempt1的server/web/e2e全部completed/success，绑定上述 `cde70d9735994cb6a579e0b800469f32631e9981`。分支交付时的 `branch-final-receipt.json` 保留 `complete_unmerged`、`merged:false` 等历史值；它证明最终分支交付，不代替后来的集成回执。

推送后的 [main CI 40520609987](https://github.com/open-nerve/NerveProject/actions/runs/40520609987) attempt1精确绑定 `head_sha=6b9d08d4821f1423c7bb9795557727c206b7880f`、branch=`main`，run为completed/success；[server 122246361805](https://github.com/open-nerve/NerveProject/actions/runs/40520609987/job/122246361805)、[web 122246361484](https://github.com/open-nerve/NerveProject/actions/runs/40520609987/job/122246361484)、[e2e 122250107435](https://github.com/open-nerve/NerveProject/actions/runs/40520609987/job/122250107435)均completed/success，无active/failed steps。最终公开元数据原件位于集成目录的 `ci/main-6b9d08d4/20261027T143732860611Z/{runs,jobs,summary}.json`；`verified-success.json`记录其哈希、本地main与已通过CI的origin/main一致且干净，以及远端P8分支相对origin/main无独有提交。分支CI、本地七门禁和main CI保持为独立事实；本次纯文档回填没有重跑产品套件。

### 证据归档与正常清理

清理前将P8工作树的1,598个证据文件、37,552,117字节和1个空目录归档，逐文件核对来源与副本SHA-256。manifest同时分类77,503个可再生文件及2个可再生空目录，未分类材料为0；不把依赖和构建产物算入保留证据数。归档根为主项目目录下 `.superpowers/sdd/P8-integration-20261027/worktree-evidence/`，完整原件新根为：

`.superpowers/sdd/P8-integration-20261027/worktree-evidence/.superpowers/sdd/P8-drafts-search/`

§8的实施证据短名 `E8` 和plan实施回执的 `E` 对应上述新根；它们执行时位于P8 worktree内的 `.superpowers/sdd/P8-drafts-search/`。归档保留原字节、原命令和cwd，旧路径不表示材料仍在已移除的 `.codex/worktress/codex/m4-p8` 中。§1的准备短名 `E` 则仍指主项目根目录 `/Users/xiaoruan/project/nerve-project/.superpowers/sdd/P8-drafts-search/preparation/`；它从未在P8 worktree，本次1,598文件归档不包含该准备目录。

确认main/远端main与成功CI一致、工作树干净，且 `origin/main..origin/codex/m4-p8` 为0提交后，按正常删除流程清理远端分支、本地已合并分支及P8 worktree，没有使用force。`cleanup.json`确认清理后再次逐文件校验归档哈希，主目录HEAD仍为main且干净。manifest SHA-256为 `85ed510a85ff67ef04388c731fe6eec565443a1445dc677791a4ea2b4ff7d30a`。

### 集成原件索引与边界

以下路径均相对主项目的集成证据目录 `/Users/xiaoruan/project/nerve-project/.superpowers/sdd/P8-integration-20261027/`，不相对当前P9工作树；本次回填已读取原件并核对下列SHA-256，manifest只读取结构摘要，不将其内容展开到正式文档。

| 原件 | SHA-256 |
|---|---|
| `premerge.json` | `48d820af01b73cfba668bfc4c6a94a05ac1a91202f37e2e799469fa97ae016ae` |
| `merge.json` | `6cec22c9e9176854709918526073808e13ab0983d3178e0705ac31603a3e36e4` |
| `push.json` | `c562135ae6f2be7b8a46be16148004509dd01a411d8c85175c04571c01acf313` |
| `verified-success.json` | `4ad73e7a301fe85a87bc3b43d7e5833b035d9720e0250009392220ea76538c8c` |
| `archive-manifest.json` | `85ed510a85ff67ef04388c731fe6eec565443a1445dc677791a4ea2b4ff7d30a` |
| `cleanup.json` | `e4c7a6be62c2c308af47881e501889a7b730684ec4dcce4fd88e801291c5f3b4` |
| `branch-final-receipt.json` | `b51e21da72c15a3f873bf390f94db21c770bcf27a93f009c7220df2011a5d11e` |

本节关闭原历史快照中的最终状态复核、最终分支CI、main集成和归档清理待办，**P8已完成**。原首次整审与T16独评的范围和结论不变，未宣称它们覆盖本次集成回填。P9从上述main另开 `codex/m4-p9`，当前仅做spec/plan准备与静态预检，未实施；M4§14、总体设计§9.4、输入handoff的总体status及M5–M8最终closeout保持原状。
