# M4 最终完成标记与集成记录

日期：2026-11-02。负责人已明确授权T14的两次`--no-ff`集成、准确main CI核对、证据归档及正常清理。

## 1. 已成立的验收事实

验收内容已以`--no-ff`合入main `83c83a9447ddd6282d8c1d52ab680d5ae67c7e10`；[main CI 41389469695](https://github.com/open-nerve/NerveProject/actions/runs/41389469695) attempt 1的server/web/e2e全部success。

收尾候选`c0e470717b044284bcd72d2fe56dca6eb7f5ac2e`的[分支CI 41386161653](https://github.com/open-nerve/NerveProject/actions/runs/41386161653) attempt 1三个作业成功，远端E2E336/336。第一次main合并的两个父提交分别是`01e40e113e11365a4eeacc2794973de2751dde61`与该候选；候选和main的完整tree均为`f099ca57dc7b4e45ae6343bf38e7022f53787a14`。

本地七门禁、Web3392项、Go1893顶层/10191含子例、默认E2E336项及三引擎各完整167项C12已通过；五个默认Go skip、规模独立实测和浏览器实际操作边界保持[阶段review](closeout-review.md)的原说明。全期[R01–R40](../../../reviews/re-29-M4整体审核.md)与[独立整分支审查及修复](../../../reviews/re-30-M4收尾整体审核.md)已闭环，没有未关闭Critical/Important或延期Minor。

## 2. 完成标记与C12绑定

[M4设计第14节](../M4-design.md#14-完成标准)十八条逐项勾选，每条指向[验收台账](closeout-acceptance.md)的D01–D18证据；设计头部/第15节及[总体设计9.4](../../v0-design.md#94-里程碑进度表)标M4已完成。SYNC44由本次完成标记关闭，其他历史同步和差异记录保留。

C12实际运行的build10产品输入SHA256为`f727834b81272f061e8125a9ddd0d36b1878563288da72e033018be2956ce6cb`，已验二进制SHA256为`fbb6e969949790ba54f018d270126bb0456f569e4ba9879d133a096ba114f3ac`。候选到第一次main tree完全相同；本记录分支只改文档，逐文件核产品、配置、测试及生成物无变化。合并产生的版本/提交标识由各准确main CI的S3验证，不声称不同构建的二进制字节相同。第二次main的双parent/tree及相同产品输入在外置收据中再绑定。

八份输入交接均closed，M5–M8的四份`handoffs/M4-closeout.md`保持open；205原条款、43处理结论、36阶段移交和M2/M0原文保护继续按[交接索引](closeout-handoffs.md)核对。输出open代表接收方未来工作，未扩成本次功能实现。

## 3. 文档收口审核

按deep-review的逻辑、集成和覆盖三个视角检查本次完成记录：完成标记须有已成功的准确main依据；十八条与原验收标准一一对应；C12产品输入不变；历史失败/限制/执行裁定保留；后续交接不冒称实现。全文核本记录及被改的当前状态、设计完成段、同步行和验收行；其他历史正文通过精确差异保护，不冒称重新进行了全M4产品审查或浏览器实测。

规模估计沿原口径更新到本记录内容，手写与生成物数不变，文档数计入T14；逐Phase对照见[阶段review第5节](closeout-review.md#5-最终实现与计划的差异规模)。

运行静态校验：仅预定文档路径有变化；M4第14节18/18勾选且D编号证据完整；总体9.4状态正确；新锚点与本地链接可达；8输入/4输出及原文保护不变；产品和测试相对验收main无差异。具体输出保存在record证据根的`record-verification.json`，发现必须先修正再提交。

## 4. 最终CI与归档收据

本次文档提交补齐完成标记；它的准确分支CI、第二次main合并/CI及归档清理事实由主目录`.superpowers/sdd/M4-closeout-integration-20261102/final-integration.json`事后签结，不在提交中预报自身CI。

以下Git勾选和叙述记录文档提交时点；提交后的CI、第二次合并与清理以该外置收据为准，不为更新自身CI再制造提交循环。只有两次main CI均成功、全部归档复验和正常清理完成后，才向负责人宣告M4全部收官。

| 原始来源 | 归档目标（主目录下） |
|---|---|
| 收尾worktree的`.superpowers/`、浏览器报告、运行日志及`bin/nerve` | `.superpowers/sdd/M4-closeout-integration-20261102/worktree-evidence/`，保留原相对路径 |
| 主目录`.superpowers/sdd/M4-closeout-preparation-20261101/` | 同集成根`preparation-evidence/M4-closeout-preparation-20261101/` |
| `codex/m4-closeout-record` worktree的独有证据 | 同集成根`record-evidence/` |
| 两次main与记录分支CI、合并/清理命令、最终台账 | 同集成根及`closure-ledger/` |

逐成员验证源/目标的SHA256、bytes、mode、symlink目标和目录集合，unknown=0；依赖和具名可再生成构建缓存单列排除，独有证据全部保留。源冻结后再执行归档；清理前确认两分支都在准确origin/main、remote-only=0、工作树干净，再删除远端分支、普通`git worktree remove`和`git branch -d`，不用force。清理后重新核归档及main与origin/最终CI一致。实际文件数、最终SHA/run/attempt/jobs和清理结果只取外置收据。

M5及以后仍由负责人另行安排，输入见各里程碑的M4交接。
