# M4/P8 草稿、搜索与链接计数评审记录

日期：2026-10-27。分支 `codex/m4-p8`，基线 `166b9efd342ecd69b0ad8f29e4cd0ae2a65b1f5d`。**独立静态预检已通过，开放 High/Medium/Low 均为0；尚未开始产品实施。** 配套：[spec](../specs/P8-drafts-search.md)、[plan](../plans/P8-drafts-search.md)、[deep-review 预检报告](../../../reviews/re-06-M4P8预检审核.md)。

## 1. 评审方式

按 [02 方法](../../claude_to_gpt/02-method.md) §3 的可调整做法，沿 P7 使用紧凑 spec/plan 和独立静态预检。先盘点设计 §12/P8、前序 Phase 与输入 handoff，再由唯一作者编写规格/计划；主控核范围、接口和事实，独立审者分别追踪草稿与搜索/计数的逻辑、集成和验证方案。没有实现完整原型、逐 Task 重放或运行产品变异，不能用静态结论替代实施期证据。

本地 ignored 原始材料位于项目根目录 `.superpowers/sdd/P8-drafts-search/preparation/`，后文简称 `E`。`requirements-inventory.md`、`code-inventory.md` 是准备输入；作者报告和独立报告各自保存读取范围、源码/文档版本及限制。

## 2. 验收标准逐条核对

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

## 7. 已知限制

静态审查不能证明新增 SQL 的真实执行计划、生成类型、运行角色授权或并发时序；对应验证必须在实施时取得原始证据。真正的架构回归、跨模块协同方案变化或意外高风险仍按负责人纪律先提交已完成工作，再上报决策。

准备提交后的回执只更新预检状态、报告与接续链接，不改已审接口/任务方案；产品、迁移、生成物和测试源码与基线相同。后续产品实施仍须逐Task验证及最终整分支deep-review。
