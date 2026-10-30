# 04 下一步与遗留

**当前接续点见§18：P12实现、本地验收及整审六项修复通过，按准确最终文稿HEAD CI后置回执核准交付；先不合并。§6–17为历史快照。**

## 1. 已知的问题
### 1.1 持续集成上 W11 页面故事偶发失败（不阻塞主线）
main 的持续集成第 340 次（P3 的合并 `dfc9378b`）在 M3 的 W11 页面故事上失败：管理员切到另一个工作区、按 Ctrl+K，命令面板 5 秒内没有出现。与 P3 的改动无关，本机复现不出来，根因未明。

**负责人 2026-10-25 的决定**：不阻塞主线推进，不专门去查。等它阻塞主线时（某次合并的持续集成因它失败、重跑也不过），停下来告诉负责人，由负责人协调 Codex 一起解决。

问题背景、证据、排除过的方向、没证实的假设和下次要收集的东西都在 [06-ci-w11-flake.md](06-ci-w11-flake.md)。

### 1.2 空的工作树 `m4-p1b`：已删除
它是客户端重启时自动建出来的空工作树（分支 `worktree-m4-p1b`，在 `7db78788`，已在 `main` 里，远端没有）。负责人同意之后，2026-10-25 删除了工作树和本地分支。

## 2. P4 `issue-collaboration` 的起步清单
- **设计**：第 12 节 P4 一段：16 个 Task，目标、任务、关闭、完成线都写好了。还要读它点名的 3.7（清洗、`skip_activity`）、3.12（评论、表情、关注、提及）、3.13（动态）、3.14（描述的历史，D5）、5.x、8.2（富文本）、8.6（游标的篡改），以及 D6（不做回复）、D8（项目管理员删别人的评论）。
- **此前各段交给 P4 的**：
  - P1b spec §5、评审 §6：描述、`skip_activity` 进 `IssuePatch`；动态的第 4 行（描述，合并）；I2 的描述的断言。
  - P2 spec §5、评审 §6：
    - 已归档的工作项上评论答 409 `issue.archived`；
    - 访客规则用到评论、表情、描述版本的读写；
    - M1-P3 交接里描述历史的地址；
    - 动态的字段名、评论的 `access`。
  - P3 spec §5、评审 §6：
    - `listIssueActivities` 对按标识指名别的工作项的行（第 5、20–23 行）执行访客规则；
    - `IssueDetail` 加 `description_html` 之后与 `Issue` 分开：HTTP 的 `issueDetail → issue` 转换在编译时故意断开，`apitest` 核对"`IssueDetail` 是 `Issue` 加一个字段"；分开之后的回答仍都经 `seen`；
    - M2 交接第 3 节：P4 的写在取得全部锁之后读时钟。
  - [M2 收尾交接](../M4-issue-core/handoffs/M2-closeout.md)、[M3 收尾交接](../M4-issue-core/handoffs/M3-closeout.md)里"处理结果（M4/P3）"末尾的"仍未处理"一行，落在 P4 的条目；设计 13.1 是全部交接的落点表。
- **评审敏感的**：
  - HTML 的清洗（bluemonday，往返的样例 S3）；
  - 提及的集合同步和关注；
  - 描述版本的 600 秒合并；
  - 评论的权限（D8）；
  - 动态列表的访客规则；
  - 游标的载荷和篡改的测试。
- **第一个提交**把 `tools/keywords.json` 的 `phase` 改成 `M4/P4`。

## 3. P5 以后的顺序
后端：
- P5 列表引擎与显示设置；
- P6 领域事件、只投递的客户端、自动归档、测试的时钟；
- P7 链接、出站客户端与抓取；
- P8 草稿、搜索与 `link_count`；
- P9 物理清理、描述版本的保留、删除项目和工作区改为任务。

前端：
- P10a 换型与删除 M6、M7 的工作项列表；
- P10b 列表一侧的数据层；
- P11 详情一侧的数据层；
- P12 列表、看板、表格、日历；
- P13 筛选、显示设置、归档页、个人主页；
- P14 详情、创建弹窗、父子、关联、链接；
- P15 时间线、评论、表情、描述的历史、草稿、搜索、命令面板；
- P16 死成员、`no-shadow` 与守卫的终态。

最后是收尾。每段的输入照第 2 节的方法收集：设计第 12 节的条目，加上此前各段 spec §5 和评审 §6 里写给它的行。P3 交给后面的见 P3 评审 §6：
- P5：`VisibleProjects` 来了之后替换 `seeEach` 和 `ArchivedProjects`；`force_custom_plan`。
- P6：事件。
- P7：链接表在全局表序里的位置。
- P8：选择器排除自己和后代。
- P9：`DeletedAt`，交错 24、25，T7.5 和 T13.4 的组合测试。
- P14：页面只在用户改了父工作项时发 `parent_id`。

## 4. M4 收尾的清单
设计第 12 节收尾一段有 14 个 Task，第 14 节是完成标准。此外，P1a–P3 的评审还留下这些：
- W11 的偶发失败（[06](06-ci-w11-flake.md)）若到收尾仍未解决，归收尾第 2 条"不稳定的故事修正"。
- P1b spec §5 "收尾"一行：测试里手写的 `55P03` 改用 `postgres.LockNotAvailable`。HEAD 上剩 13 个测试文件里的 16 处，用 `git grep '"55P03"'` 找，去掉 `tx.go` 自己、它的测试和 `gate_test.go` 造的错误。
- `workspace` 的 `deactivation_test.go` 的 `waitsFor` 是 `pgtest.RowWaits` 的近似副本，合并掉。
- M3 的几个测试世界（`project_*`、`membership_world`、`ending_world`、`interleaving_*`、`permission_matrix_*`）手拼项目成员的请求，可以收进共享的帮手。这是测试整理，可选。
- 全部交接逐条写结论，`status` 改为 `closed`。给 M5–M8 写 `handoffs/M4-closeout.md`：设计 13.2 已有四个链接指向它们，内容见那张表。
- 设计 3.23 逐行核对；[plane-diff.md](../plane-diff.md)、[frontend-changes.md](../frontend-changes.md) 定稿；README 按设计 8.8 更新。
- 浏览器核对 C12 要 Chromium、Firefox、WebKit。早期环境只有Chromium；P7已用项目内缓存实际跑过Chromium/Firefox，不能沿用“本机只有Chromium”判断。WebKit与C12的三浏览器核对仍到收尾按实际环境处理。
- 性能的数字写进收尾的评审，交给 M8：列表 1 万、2 万、10 万个工作项；搜索；删除；清理。
- 设计第 15 节、总体设计 9.4 标 M4 已完成。

## 5. 阶段终审与集成（负责人 2026-10-26 更新）
- **Codex 推进**：每段的设计细化、spec/plan 预检、实现、测试、逐任务评审和记录。
- **每个 Phase 完成后使用 `deep-review` 整分支终审**：由我们自行完成，不再等待 Claude 审查；此项替代 2026-10-25 的 Claude 终审安排。审查必须固定分支、基线与 HEAD，并覆盖逻辑正确性、集成安全和测试覆盖；报告与阶段 review 都可追溯。
- **修复闭环**：局部问题由 Codex 根因修复并完成相应验证、复审；没有阻断发现后，向负责人提供分支、基线、最终 HEAD、spec/plan/review 和 deep-review 报告路径。
- **集成门**：终审通过不自动授权合并，仍按负责人对该阶段的安排执行；已授权不再重复请示。用 `--no-ff` 合并后确认精确 main SHA 的 CI 通过，归档并核验本地证据，再清理分支和 worktree。下一段从完成集成的 main 起步。
- **上报**：架构回归、跨模块协同调整或意外高风险发现，按 [03](03-rules-and-pitfalls.md) 第 1 节先提交已完成工作再上报；审查方式调整不改变这个边界。

## 6. 当前落点（2026-10-26，P6 集成后）

P6 已由 deep-review 终审通过并经授权 `--no-ff` 合并为 `5005be985e7ec78727edc6f8fc5c05999b45cc65`；[main CI 40305247546](https://github.com/open-nerve/NerveProject/actions/runs/40305247546) 重跑三任务全绿。921 个本地证据文件归档核 hash 后，P6 本地/远端分支和 worktree 已清理，见 [P6 review §2.2](../M4-issue-core/reviews/P6-issue-events-review.md#22-授权合并main-ci-与归档清理)。

当前独立分支 `codex/m4-p7` 仅准备[链接 spec](../M4-issue-core/specs/P7-issue-links.md)、[plan](../M4-issue-core/plans/P7-issue-links.md)与[预检记录](../M4-issue-core/reviews/P7-issue-links-review.md)。预检及修订复审已通过，等待负责人安排启动 T1，不自动实现或合并 P7；M4 总体完成标准和交接仍保持开放。

## 7. P7交付前快照（2026-10-27，集成前）

§6保留2026-10-26 P6集成后的冻结快照。现在`codex/m4-p7`已完成T1–T15实施和独评，原范围`5005be98..d2a3cf26`整审全文覆盖165文件/457chunks；5新增Minor及3历史Minor经T16单次修复后，独立定向复审`d2a3cf26..7e903ecb`确认全部ADDRESSED；唯一T7历史时态文案已由作者纠正，本次纯文档增量独立确认和最终交付HEAD CI由主控最后记录。本期产品检查点cd0的[CI40397086823](https://github.com/open-nerve/NerveProject/actions/runs/40397086823)已核准确HEAD全绿，后续纯文档交付HEAD CI另核。T15准确d2的[CI40387396589](https://github.com/open-nerve/NerveProject/actions/runs/40387396589)三任务全绿；本轮完整本地门禁、独立fix定向复审和最终新HEAD CI状态见[P7 review](../M4-issue-core/reviews/P7-issue-links-review.md)与[整体审核](../../reviews/re-05-M4P7整体审核.md)。未授权合并，主目录仍main，工作树/证据保留。

下一接收方继续P8列表link_count/草稿/搜索，P9异步删除/保留/清理，P11/P14详情数据层和I10/I17页面，P15协作页面；M4整体与输入handoff保持open，不创建M5–M8最终closeout。

## 8. P8交付前快照（2026-10-27，P7 集成后）

P7 已经授权以 `--no-ff` 合并为 `166b9efd342ecd69b0ad8f29e4cd0ae2a65b1f5d`，[main CI 40451632280](https://github.com/open-nerve/NerveProject/actions/runs/40451632280) 三任务全绿。5,443个证据文件归档并逐SHA核验后，P7本地/远端分支与worktree均已清理，完整事实见 [P7 review](../M4-issue-core/reviews/P7-issue-links-review.md) 第8节；§6/§7保留较早的冻结快照。

P8 `drafts-search` 已完成T1–T15实施与逐Task独立评审，产品HEAD `10f44790ccb842e758f0ab50ecb4d294dc15a6e6`；[spec](../M4-issue-core/specs/P8-drafts-search.md)、[实际plan](../M4-issue-core/plans/P8-drafts-search.md)、[review](../M4-issue-core/reviews/P8-drafts-search-review.md)与[首次整体审核](../../reviews/re-07-M4P8整体审核.md)列验收、四次本地七门禁、产品checkpoint CI和整审证据。基线仍为上述main，工作树 `.codex/worktress/codex/m4-p8`、分支 `codex/m4-p8`；实施、首次整审和T16文档独评已完成，准确文档a65e8605及task-16-review.md确认Approved/D1 ADDRESSED；最终准确HEAD CI及集成由后续提交和 ignored `task-16-final-checkpoint-receipt.json` 追踪。未合并、不清理，M4总体和后续handoff保持open。准备提交 `d4391bff` 的静态预检是先前历史，不当作产品验证。


## 9. 当前接续点（2026-10-27，P8 集成后）

P8 `drafts-search` 最终分支 `cde70d9735994cb6a579e0b800469f32631e9981` 的[分支 CI 40515294403](https://github.com/open-nerve/NerveProject/actions/runs/40515294403)已通过，随后经负责人授权 `--no-ff` 合并为 main `6b9d08d4821f1423c7bb9795557727c206b7880f`；准确[main CI 40520609987](https://github.com/open-nerve/NerveProject/actions/runs/40520609987) attempt1的server/web/e2e全部completed/success。1,598文件、37,552,117字节及1空目录归档逐hash核验后，P8本地/远端分支与worktree已正常清理，清理后再次核验归档。完整集成原件、manifest和新证据根见[P8 review §9](../M4-issue-core/reviews/P8-drafts-search-review.md#9-授权合并main-ci-与归档清理2026-10-27)；上文未合并/待CI文字保留为当时事实。

当前独立分支 `codex/m4-p9` 位于 `.codex/worktress/codex/m4-p9`，基线为上述main，[物理清理 spec](../M4-issue-core/specs/P9-purge.md)、[plan](../M4-issue-core/plans/P9-purge.md)与[静态预检记录](../M4-issue-core/reviews/P9-purge-review.md)已准备完成。[deep-review预检](../../reviews/re-08-M4P9预检审核.md)的2项局部文档遗漏均已修订复审，开放发现为0；13设计粗项细化为16 Task，负责人现已授权从T1开始实施，阶段完成做deep-review后保留分支等待安排，不合并；产品验收依真实执行证据关闭，不将既有P8运行证据当成P9验证。M4总体§14/总体设计§9.4、输入handoff总体status和M5–M8最终closeout保持原状。

## 10. 当前接续点（2026-10-28，P9文档收尾）

P9本机ignored证据已随授权集成归档清理；本节历史收据链接现指向P9 review§14稳定归档入口，原收据名/身份与历史结论保留，不把已清理本机路径当现存文件。

主目录仍main，P9分支 `codex/m4-p9` / worktree `.codex/worktress/codex/m4-p9` 保留。基线 `6b9d08d4821f1423c7bb9795557727c206b7880f`；当前产品 `e003e96625ba51264d3aeb77729729d53a770103`；文档交付HEAD：首次文档7f5625bf身份保留在原 `task-16-final-receipt.json`；最终交付HEAD与分支CI见主控[交付收据](../M4-issue-core/reviews/P9-purge-review.md#14-授权合并main-ci-与归档清理2026-10-28)，不倒填产品或规模测量HEAD。

入口：[spec](../M4-issue-core/specs/P9-purge.md)、[plan](../M4-issue-core/plans/P9-purge.md)、[review](../M4-issue-core/reviews/P9-purge-review.md)、[整体审核](../../reviews/re-09-M4P9整体审核.md)。T1–T15已独评，首次整审238文件60363行及d253/e003四项发现修复复审闭环，未留产品/架构待决；T16两席独评与CORE-1/2修复复审已闭环，最终文档HEAD CI独立核对。

最终产品F：准确e003七项串行F均exit0；Go51包+16无test包（非verbose逐test skip未知），web138files/1165Vitest+5Node且0cache/skip，E2E163=161Cr+2Fx/0skip；[主控最终F收据](../M4-issue-core/reviews/P9-purge-review.md#14-授权合并main-ci-与归档清理2026-10-28)。准确产品分支CI：准确产品e003的CI40630029402 attempt1已由主控核验server/web/e2e全部success，见[产品CI收据](../M4-issue-core/reviews/P9-purge-review.md#14-授权合并main-ci-与归档清理2026-10-28)；不替代本轮文档提交或未来main CI。文档静态/独立复审：首次31文档静态核对见[静态收据](../M4-issue-core/reviews/P9-purge-review.md#14-授权合并main-ci-与归档清理2026-10-28)；两席文档独评及[CORE-1/2修复复审](../M4-issue-core/reviews/P9-purge-review.md#14-授权合并main-ci-与归档清理2026-10-28)已闭环，见[主控核证](../M4-issue-core/reviews/P9-purge-review.md#14-授权合并main-ci-与归档清理2026-10-28)；最终状态/注释小diff仍交主控限定核验，最终文档HEAD及其准确分支CI由主控[交付收据](../M4-issue-core/reviews/P9-purge-review.md#14-授权合并main-ci-与归档清理2026-10-28)记录；该可变指针当前待核，须按准确HEAD核验，不代替未授权的main集成/CI。T12/T15旧检查点分别有准确分支CI成功收据；T12旧attempt1 W11失败仍是历史事实，未声称查明/修复根因。SP2仅引用4c7379f1+dirty源码绑定d26的原测量，不倒填当前HEAD。

本轮只完成P9后端与文档交接，不启动P10，不合并/清理。M4总体§14/总体设计§9.4、输入handoff status、C12三浏览器及M5–M8最终closeout继续待所属收尾。负责人未来明确授权后才--no-ff合并，核准确main SHA CI及归档哈希，再清理分支/worktree。


## 11. 历史接续点（2026-10-28，P9集成后、P10a启动时）

本节“当前T1/禁止push”均为当时快照；后来实施及root分支CI推送授权见§12，不覆盖最新状态。

P9经授权--no-ff合并main `4599a9cfc5be90b2c242977f194157778a5f6ec9`，[main CI 40642368845](https://github.com/open-nerve/NerveProject/actions/runs/40642368845)三个任务全绿；3,317文件/136,629,586字节/1空目录归档核hash，清理P9本地/远端分支与worktree后再次核验通过，见[P9 review §14](../M4-issue-core/reviews/P9-purge-review.md#14-授权合并main-ci-与归档清理2026-10-28)。以上未集成快照均不覆盖本节。

当前 `codex/m4-p10a` / `.codex/worktress/codex/m4-p10a` 已有既定两段的准备文档，当前已获 P10a 实施授权：[P10a spec](../M4-issue-core/specs/P10a-web-issue-types.md)、[plan](../M4-issue-core/plans/P10a-web-issue-types.md)；[P10b spec](../M4-issue-core/specs/P10b-web-list-data.md)、[plan](../M4-issue-core/plans/P10b-web-list-data.md)。静态预检结果由各阶段review及[deep-review](../../reviews/re-10-M4P10预检审核.md)接续；当前按通过预检的 spec/plan 推进 P10a T1，关键词 phase 按既定守卫语法设 M4/P10；阶段验收仍开放。P10b当前是依赖预检，实施前必须在P10a已集成且main CI绿的基线重核。M4总体§14/总体设计§9.4、handoff总状态和最终closeout不提前关闭。

**负责人裁定 P10A-S1（2026-10-28）：** 已选择A，保持只删工作项列表的范围，保留领域自身加载；M6/M7自身页面完整可用与零失败验收归回所属里程碑，精确记录未接通请求和可见代价。设计、spec、plan及交接责任已同步，独立定向静态预检通过、无开放发现，证据见上述报告§5；M4页面门禁/守卫不变。定向预检完成后，负责人已授权实施 P10a，从 T1 逐任务推进；禁止 push/merge，集成须另获授权。


## 12. P10a实施后接续（2026-10-28）

当前在codex/m4-p10a worktree：T1–T8已各Task独评、修复复审及精确提交，最后产品HEAD `5f7085bce4aa5103a0133fa982847d274efa6e47`，tree `d0b4ea997e3dd7cdd61e9cbc662b75b064636364`；主项目main仍4599a9cf。A/B原全文与A-01/A-02修复独评已完成，无剩余代码发现；C原22路径全文/guard delta通过，C-r1已读14增量，唯一C-r1-01经C-r2复审ADDRESSED，无新增。见[P10a spec](../M4-issue-core/specs/P10a-web-issue-types.md)、[plan](../M4-issue-core/plans/P10a-web-issue-types.md)、[phase review](../M4-issue-core/reviews/P10a-web-issue-types-review.md)、[整体审核（实施与终审完成）](../../reviews/re-11-M4P10a整体审核.md)。最终交付收据尚未生成，不预填准确文档HEAD/CI或通过结论。

root已有本阶段分支CI推送授权，完整门禁后已使用系统git推T8及最后产品5f708；808准确CI run40703659542 attempt1三job已核success；5f708准确产品CI run40712479050 attempt1的server122901158718/web122901159223/e2e122907311901均completed/success，receipt已保存；T9最终交付文档HEAD CI仍待提交后root核，另列。旧T3/T6成功只代表各自检查点，不是最终/main CI。本次仍无合并授权，不merge、不清理分支/worktree/X。未来另授权后才记录真实merge SHA、准确mainCI及全量归档逐hash/清理回执。

T8原round0本地F6=1277、F7=169；R19最终仅新故事finally变化，由最终正常六入口6/6及e2e static补齐，F1–3另有keywords测试精确登记差异，不称整套在r1逐字重跑。六入口75API、八最低项及有效实体组件/service层限制、23段真实stderr归属见phase review；M6/M7前缺列表与领域自身未接通两代价保留，归档404后持续loader及有效实体网络/整页零失败留M6/M7。

A-01/A-02统一修复已提交；最后本地F 1275测试/169E2E通过（F1–3六web删除桥，F4–7同源），C-r1唯一文稿Minor已修且C-r2 Approved，随后root提交并核最终交付HEAD分支CI；不得以本轮文档自查结束Phase。P10b仍未实施，必须等P10a真实main集成且准确mainCI通过后重核spec§2.3/实际文件、消费者与未完成过渡。M4§14/总体§9.4、五份输入总open及M5–M8最终closeout保持，不提前关闭或预建。


最后代码覆盖ready集合355路径=A93/B240/C22，准确覆盖源/行数/哈希与限制见re-11；实施与终审已完成，最后文稿C-r2真实Approved。root提交后生成X/final-delivery-receipt.json，核准确HEAD/run/attempt/ref/jobs全部success（必须最终文稿HEADCI全绿才结束任务），不为正文自指SHA循环提交；未来mainCI/归档清理另授权。旧T9静态inventory误写且原bytes不可证明的限制见phase review§9，最后库存另具名重核，产品与原F/CI raw未改。

## 13. P10a完成、P10b执行前预检（2026-10-29）

**历史时点：本节保留执行前状态，当前接续以§14为准。**

P10a 最终 d1a19e78 授权 --no-ff 合入 main `a06e3f70547b755b03bf8e6b33caafd52725d0f6`，[main CI 40723697041](https://github.com/open-nerve/NerveProject/actions/runs/40723697041) attempt 1 三任务全绿。5,898 证据文件 / 356,803,778 字节 / 22 空目录归档并逐哈希核验，随后正常清理 P10a 分支/worktree，再核归档一致；现址、历史限制及准确收据见 [P10a review §10](../M4-issue-core/reviews/P10a-web-issue-types-review.md#10-授权合并main-ci-与归档清理2026-10-29)。上节待授权/待最终 CI/待归档均已由此接续。

`codex/m4-p10b` / `.codex/worktress/codex/m4-p10b` 从上述准确 main 建立，只准备 [spec](../M4-issue-core/specs/P10b-web-list-data.md)、[plan](../M4-issue-core/plans/P10b-web-list-data.md) 与 [执行前预检](../M4-issue-core/reviews/P10b-web-list-data-review.md)，[deep-review](../../reviews/re-12-M4P10b预检审核.md) 的文档局部问题已修，独立执行者检查无待回答问题。T5 根协调入口前置、T6 canonical 旧写者收口、会话退役接线与 P10a 实际回归/旧接口期限均已明确。当前没有 P10b 实施授权，14 Task 和十项产品验收全部开放。

下一步待负责人安排后从 T1 执行；每阶段完成仍先整分支 deep-review 和修复，未经授权不合并。主目录保持 main；P10b 准备证据位于根 `.superpowers/sdd/P10b-preparation-20261029/`，未来归档不得只收 worktree 而遗漏它。M4 总验收/总体 §9.4、输入 handoff 总状态及 M5–M8 最终 closeout 不关闭。

## 14. P10b最终验收与交付边界（2026-10-29）

分支`codex/m4-p10b`，worktree `.codex/worktress/codex/m4-p10b`，实际main基线a06e3f70547b755b03bf8e6b33caafd52725d0f6；产品候选2e1264aa5711f4699546bb64728151da2f802a0e。T1–T13独评、原whole整审与两次限定修复复审完成；原5I/3M、修复期N1、限定新增N2已处理。真实证据见[阶段review](../M4-issue-core/reviews/P10b-web-list-data-review.md)、[正式re-13](../../reviews/re-13-M4P10b整体审核.md)、[spec](../M4-issue-core/specs/P10b-web-list-data.md)、[plan](../M4-issue-core/plans/P10b-web-list-data.md)。

产品候选完整F七项通过：Go51包/16无测试、web1870（8个fresh包日志）、E2E169；原件X/task-14-root-final-f-attempt1及摘要保留13段真实stderr观察，不称零warning。候选CI40896656404 attempt1三任务成功，绑定上述2e1264aa，准确原件X/task-14-candidate-ci-09.json；纯文稿HEAD还需主控push后核准确CI，不将候选CI冒作新文稿提交的CI。

**下一步只由主控核最终纯文稿与准确交付HEAD CI并向用户报告，保持未合并，不开始P11。** M4§14/总体§9.4、输入handoff顶层open和M5–M8最终closeout保持原范围。R36/R38移交P13，M6领域loader/action/嵌套button与其它既有翻译/key、act/Headless观察按阶段review保留。

主目录始终main，worktree及所有证据保留。根 `.superpowers/sdd/P10b-preparation-20261029/` 在worktree外，未来授权归档时必须一并保存；不得只归档worktree而遗漏准备根。当前没有合并或清理授权。

## 15. P10b完成、P11文档预检（2026-10-30）

P10b最终交付2aaef85c经授权 --no-ff 合并 main `df5d3bbc1887486bd105fe6fc663f57f5b371992`，[准确main CI 40908365144](https://github.com/open-nerve/NerveProject/actions/runs/40908365144) attempt1三任务全绿。worktree3247证据文件/632572496字节/2空目录、外部准备根33文件/1182444字节归档逐hash核验；分支及worktree正常清理后再次核验通过，见[P10b review§17](../M4-issue-core/reviews/P10b-web-list-data-review.md#17-授权合并main-ci与归档清理2026-10-30)。

P11准备分支 `codex/m4-p11`，worktree `.codex/worktress/codex/m4-p11`，基线即上述main；[spec](../M4-issue-core/specs/P11-web-detail-data.md)、[plan](../M4-issue-core/plans/P11-web-detail-data.md)、[预检记录](../M4-issue-core/reviews/P11-web-detail-data-review.md)、[deep-review](../../reviews/re-14-M4P11预检审核.md)。本轮只写文档，14 Task/A1–A10全部开放，phase仍M4/P10；[编号宽度合同P11-S1](../M4-issue-core/reviews/P11-project-number-width-decision.md)已裁定A，按已加载可见数据统一列宽；修订和文档预检复核通过，仍待负责人另行安排T1，当前不实施。

P11沿IssueRootStore方案A管理详情资源权限失效；按资源同步迁服务/store/类型/消费者，普通链接非M5，M7旧版本DTO只归属化不伪造新API。旧IssueService的归档页头/issueId跳转数据接线前移P11，完整页面验收仍P13/P14；搜索wrapper随P12首个生产消费者创建，旧搜索最后清理仍P15。准备证据在主目录 `.superpowers/sdd/P11-preparation-20261030/`，未来归档不能遗漏。M4总验收、输入handoff顶层和最终closeout不提前关闭。

## 16. P11实施、整审及待集成（2026-10-30）

分支codex/m4-p11，基线main df5d3bbc；产品e84acb4b已推送，完整本地F通过，Web210文件2315项、E2E169项，整分支252个TS/TSX零警告。首次整审283变更中271全文/6结构/6相关章节，九项问题及后续死成员/静态补查已修；不宣称每Task独评或283全文。见[P11 spec](../M4-issue-core/specs/P11-web-detail-data.md)、[plan](../M4-issue-core/plans/P11-web-detail-data.md)、[review](../M4-issue-core/reviews/P11-web-detail-data-review.md)、[re-15](../../reviews/re-15-M4P11整体审核.md)。

主控只完成文稿与准确最终HEAD的server/web/e2e核验，再交负责人安排；后置t15-final-delivery.json必须与HEAD一致，不冒用产品候选CI。当前没有合并/清理/P12实施授权。准备证据位于主目录.superpowers/sdd/P11-preparation-20261030，实施证据在worktree的.superpowers/sdd/P11-web-detail-data，均保留。M4§14、总体§9.4、输入handoff顶层与M5–M8最终closeout保持开放。


## 17. P11完成、P12文档预检（2026-10-30）

P11交付42afa24f已按授权--no-ff合并main `ce3c39abef99a290157b282c23180983f8746bcb`；[main CI 41005896280](https://github.com/open-nerve/NerveProject/actions/runs/41005896280) attempt1 server/web/e2e全绿。两父与tree、归档哈希、远端/本地分支及worktree清理见[P11 review§15](../M4-issue-core/reviews/P11-web-detail-data-review.md#15-授权合并main-ci与归档清理2026-10-30)。全部实施原件映射到主目录.superpowers/sdd/P11-integration-20261030/worktree-evidence，准备34文件另存preparation-evidence；清理后再次校验通过。

当前 `codex/m4-p12` / `.codex/worktress/codex/m4-p12` 以该main为基线，只有[spec](../M4-issue-core/specs/P12-web-list-pages.md)、[plan](../M4-issue-core/plans/P12-web-list-pages.md)及[预检记录](../M4-issue-core/reviews/P12-web-list-pages-review.md)。16Task，A1–A12全部开放；phase保持M4/P11，不安装依赖/原型/实施/产品测试。准备证据在主目录.superpowers/sdd/P12-preparation-20261030，后续归档不得遗漏。

P12用已有root统一权限/canonical/确认写入，原生搜索同Task迁日历/子项/关联三个消费者；原22路由清单14存在8已删；DateDropdown和七下拉、四布局、I3/I4/I6/I7以及C1三浏览器/C3列表侧是实际完成线。P13–P16范围保留；M4§14/总体§9.4、输入handoff顶层及最终closeout不提前关闭。预检结论以当前review为准，获负责人下一步实施安排后再从T1开始。

## 18. 历史接续：P12实施、整审修复与剩余验证（2026-10-30）

分支 `codex/m4-p12`，worktree `.codex/worktress/codex/m4-p12`，基线 main `ce3c39abef99a290157b282c23180983f8746bcb`。T1–T15完成，T16六项重要问题已修复为 `49b628fb`；[spec](../M4-issue-core/specs/P12-web-list-pages.md)、[plan](../M4-issue-core/plans/P12-web-list-pages.md)、[阶段记录](../M4-issue-core/reviews/P12-web-list-pages-review.md)、[整分支审核](../../reviews/re-17-M4P12整体审核.md)。

最终产品的前端255文件2697测试、默认E2E180、三浏览器各11通过，静态/knip通过；服务端三项以输入hash核准复用T15。WebKit首次因环境中断5通过/6失败另保留，负责人恢复Docker后同产品输入11/11；没有放宽断言。候选bbd92dd0的CI41060487600三项全绿；准确最终文稿HEAD按review§26核server/web/e2e，后置final-delivery.json放忽略证据目录，回执与HEAD一致且通过才报告交付。

后置0cdd315d CI首次I3失败，attempt2三任务成功；本地CPU8复现导航后应用尚未挂载便开始行序预算，已仅修正夹具为先等分组attached，再按原5秒断言完整ID。新输入静态/knip、完整E2E180、三浏览器各11均通过，未变输入的前端单测/服务端门禁按明确域复用；原远端trace下载失败，不把本地机制冒作其唯一根因。新提交仍另核准确CI，详情见[review§27](../M4-issue-core/reviews/P12-web-list-pages-review.md#27-后置ci的i3诊断与测试同步修正2026-10-30)。

主目录保持main。当前授权为完成P12、deep-review后保留分支；不合并、不清理、不开始P13。准备证据在主目录 `.superpowers/sdd/P12-preparation-20261030/`，实施证据在worktree `.superpowers/sdd/P12-web-list-pages/`，未来归档均须保留。M4§14、总体§9.4、输入交接顶层open和M5–M8最终closeout仍开放。

## 19. 当前接续：P12完成、P13文档预检（2026-10-31）

P12最终e6d1329b已授权--no-ff合并main ff89d0a1219a8d09ff924f5bd5257c486cd7e2b8，准确[main CI41087022719](https://github.com/open-nerve/NerveProject/actions/runs/41087022719)三任务成功。1786份实施/运行证据及14份准备材料归档逐hash通过，正常删除远端/本地分支及worktree后再次复验。主目录main干净，证据根.superpowers/sdd/P12-integration-20261031，见[P12 review§28](../M4-issue-core/reviews/P12-web-list-pages-review.md#28-授权合并main-ci与归档清理2026-10-31)。Git合并、push及远端删除均自动完成；公开API限流后已通过授权Chrome核CI，无需取凭据或负责人代操作。

P13分支codex/m4-p13、worktree .codex/worktress/codex/m4-p13，以ff89d0a1为基线。只有[spec](../M4-issue-core/specs/P13-web-filters-archives-profile.md)、[plan](../M4-issue-core/plans/P13-web-filters-archives-profile.md)、[阶段review](../M4-issue-core/reviews/P13-web-filters-archives-profile-review.md)、[deep-review预检](../../reviews/re-18-M4P13预检审核.md)及集成/设计状态同步。14Task，A1–A12全部开放，phase保持M4/P12；未实现、未装依赖、未跑产品测试。准备证据在主项目.superpowers/sdd/P13-preparation-20261031，未来必须一并归档。

预检通过，下一步等负责人授权从T1实施。沿原root权限/canonical/确认写入；区分项目服务端偏好与归档/profile本地设置；合法开区间和未知已保存标签不被新控件清掉；完整工作区标签仍M7。阶段结束须deep-review和修复复审，先不合并。M4§14/总体§9.4、输入handoff顶层open及M5–M8最终closeout保持开放。
