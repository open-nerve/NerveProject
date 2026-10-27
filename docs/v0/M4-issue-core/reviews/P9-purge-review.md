# M4/P9 物理清理、版本保留与异步删除评审记录

> **当前接续：负责人已授权从T1实施，阶段结束做deep-review，暂不合并。** 下文§1–7保留准备期预检事实；实施结果与决策另行追加，不将预检证据当作产品验证。

日期：2026-10-27。分支 `codex/m4-p9`，基线 `6b9d08d4821f1423c7bb9795557727c206b7880f`。**准备与独立静态预检 Approved；2项局部遗漏已修订复审，开放 High/Medium/Low 均为0；未实现 P9。** 配套：[spec](../specs/P9-purge.md)、[plan](../plans/P9-purge.md)、[deep-review报告](../../../reviews/re-08-M4P9预检审核.md)。

## 1. 评审方式

沿 [02 方法](../../claude_to_gpt/02-method.md) §3 允许的紧凑方法，由唯一作者细化已批准设计，独立审者核查逻辑正确性、集成安全与测试覆盖。准备工作不做完整产品原型或逐 Task 重放，静态审查不替代实施期的真实 SQL、锁交错、River 调度、变异和规模证据。

准备原件在 P9 worktree 的 ignored `.superpowers/sdd/P9-purge/`，下文简称 E9。输入 `design-inventory.md`、`code-inventory.md` 已核既定协议、前序交接和当前源码；作者、独立审者及主控分别记录范围和结论。唯一作者冻结后，两位独立审者均全文读完 v1 spec/plan，再分别追踪删除/锁/事件和清理/保留/调度/故事/规模；主控核因果链、覆盖清单与准确指纹。删除侧101文件/15,568行、清理侧106文件/13,367行有重叠，不相加为独立文件总数；节选另列，不冒称全文。主控已核两份全文清单每项行数、字节和SHA匹配，见 `preflight-coverage-verification.json`。

## 2. 验收标准逐条核对

P8 已按授权以 `--no-ff` 合并为上述基线，[main CI 40520609987](https://github.com/open-nerve/NerveProject/actions/runs/40520609987) 的 server/web/e2e 三任务全部成功。1,598 文件、37,552,117 字节及 1 个空证据目录归档校验后，P8 本地/远端分支和 worktree 已清理，清理后再核哈希通过。完整回执见 [P8 review](P8-drafts-search-review.md) 第 9 节；这些证据证明 P9 基线状态，不证明 P9 产品通过验收。

P9 所有产品验收均未完成，逐条责任以 spec §4 的A1–A13和 plan 的16 Task为准。准备期实际运行 `node tools/keywords.mjs`（72规则、23例外、零命中）和 `go -C server test -count=1 ./internal/archtest`，均 exit0；前后输入状态一致，原始日志哈希已核，分别见 E9 的 `checks/preparation-keywords/`、`checks/preparation-architecture/`。首稿文档检查144处本地链接/锚点无新增失败，`git diff --check`通过；4个既有M5–M8最终closeout目标留待M4收尾，不将其记作新增失败。最终回执后的检查由 `checks/preparation-docs-final/`、`checks/preparation-diff-final/` 和 `final-preparation-receipt.json` 绑定准确文件身份。没有执行P9迁移、生成、产品测试、性能测量或完整本地CI。

## 3. 准备期决定

沿用已批准的四模块清理者、单个清理任务、版本保留项目 N 锁、事务内删除事件及按项目投递；不引入新的模块协议、持久化事件表或依赖。具体局部细化由 spec §3 固定，独立预检发现会在 §4 闭环。账户不做物理删除，旧 P2 空创建者权限边界另以测试数据准备验证；不据历史措辞增加账户清理功能。

## 4. 预检发现与处理

首稿 v1 提交为 `26f004057f3a42b0f880257965f9693f1bb68ec5`，包含P8集成回填与P9准备，phase首次改为M4/P9，72规则/23例外未变。删除侧初审零发现；清理侧2项Medium，经主控核实现有源码后修订，原审者定向复审均为ADDRESSED。没有需要重新裁定的架构问题。

| 原始ID / 级别 | 根因与修订 | 闭环与实施期验证 |
|---|---|---|
| P9-PURGE-M1 / Medium | T1新增23个索引却遗漏既有精确全量schema/index期望。spec/plan补四个migration测试文件、23个`iw`项和draft精确indexdef；不放宽全量相等，T3新目录测试仍保留 | `purge-rereview-v2.md`确认ADDRESSED；未来T1实际执行`TestConstraintAndIndexNames`、`TestDraftSchema`红绿及migrations回归 |
| P9-PURGE-M2 / Medium | T10异步切换后，runtime无项目草稿测试仍在204后立即Scan审计bool，合法未完成窗口会失败。补有限等待准确workspace/At/By的真实workspace_drafts任务completed，保留实际runtime角色、两个草稿三审计字段和邻workspace整行不变 | 同一复审确认ADDRESSED；未来T10精确选择器命中`TestRuntimeRoleDeletesDraftsInWorkspaceWithoutProjects`，不可用owner直调、睡眠或宽松断言替代 |

| 冻结稿 | spec SHA-256 | plan SHA-256 |
|---|---|---|
| v1：提交26f00405，337/259行 | `b4609d278b2a05a4f573881e55aa4ebec18ba6236aea3fba5b998da39b766763` | `ab299bc8ac3c80a16fae77b88ed9f3295044bd4cf133908013c1db8739a626e6` |
| v2：26f00405上两项修订，341/261行 | `17dc388fe4d74052d62a1427a94926e9417224bf8a1490371caa659a42240129` | `879d2a20d760fa80e6d4d5e05ffbd084f55b8948814c82e4e25e1fffe51069c9` |

原始报告/清单为 `deletion-preflight.md`、`deletion-coverage.md`、`purge-preflight.md`、`purge-coverage.md`；作者修订记录 `author-revision-v2.md`，闭环为 `purge-rereview-v2.md`。v1/v2原字节分别保存在 `frozen-v1/`、`frozen-v2/`，`revision-v1-v2.diff`为14增/8删的两文件修订。冻结前已吸收的事件登记顺序、eitherOf两端预检、W3两处即时查询及周期任务防重叠等细化不重复计作这2项发现。v2之后只改预检状态与证据入口，不改实施协议/任务；最终回执定向核验另见 `final-receipt-review.md`，不将其冒充再做一次产品整审。

## 5. 与计划方法的差异

本轮只准备可执行 spec、任务计划和静态预检。没有生成产品实现、预造生成文件指纹或虚构测试红绿、并发/性能数字。实施时应逐 Task 保存真实命令、选择器命中数、输出和恢复证据；已有阶段的运行结果只能证明其各自范围。

## 6. 移交事项

本轮暂不实现 P9，也不合并 P9。预检完成后由负责人安排启动 T1；P10a–P16、M4 总体完成标准、输入 handoff 总状态和 M5–M8 最终交接均保持原阶段归属。P9 实施结束仍须完整整分支 deep-review、局部问题修复及复审，再按负责人的集成安排处理。

## 7. 已知限制

静态预检不能证明运行时锁序、真实 worker 的重试/snooze、索引和 SQL 执行效果或五个目标规模项目的删除性能；计划必须保留相应实施门禁。真正的架构回归、新跨模块协同方案或意外高风险，按负责人纪律先提交已完成工作再上报。
