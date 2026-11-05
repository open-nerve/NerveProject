# Deep Review：M5/P7a 草稿准备件恢复协议

审查日期：2026-11-04。正式main基线`963636757fe5c5a8c10d5781b43d1481ba6f32a2`，冻结候选`ed1e2a496d6ee990e0dd08816364fa37d4b10ca0`。按deep-review与executing-plans执行一次fresh whole-branch审查，审者`m5_p7a_final_review`（gpt-6-astra）；主控处理结论与验证。没有逐Task派审者或重复同范围复审。

## 1. 范围、证据与限制

完整审查41个变更文件：37个手写文件完整读取，4个生成物对源和完整diff核查。包括新增domain/app/仓储/SQL/HTTP、原组合根、权限矩阵、四份bootstrap测试及local/S3 API故事；M5设计、P5/P6接续记录、P7决策、P7a合同、P7b延后合同及全部局部裁定。额外追踪既有Host/DraftLocks、只读事务、shared cursor、DTO/codec和prepare/duplicate/complete/delete/publish调用链。

审者执行只读证据核验：T4 race、P5回归、T5 gen-check/Go lint的raw SHA/字节和4544个当前非文档输入逐项相同；T4为65个顶层PASS，无skip/data race；T3原件为local/S3四个API E2E通过，后续新增测试输入差异明确记录。没有在另一套件运行时启动数据库测试，也没有声称完整CI已经通过。完整审者结论转存、变异/恢复SHA、review package和原始日志在本阶段ignored证据目录，最终归档入口由阶段记录回填。

## 2. 三视角与五项重点

| 视角 | 判断 |
|---|---|
| 逻辑 | UUID分页、默认50/最大100、多取一条、空数组、cursor规范化/跨draft、事务/行错误不返回部分成功均正确；发现下述全零UUID边界。 |
| 集成 | YAML总入口→生成handler→窄应用端口→既有Host/ReadTx→asset自身SQL一致，无新权限、跨模块JOIN、存储I/O或写副作用；旧403及DELETE合同保持。 |
| 测试 | 未知ID恢复、同名隔离、真实PID快照/撤权、取消释放、写入交错、动态分页及有效变异有覆盖；需要补全零UUID用例。 |

五项Review Focus均已逐项核对：未知ID只从API发现；新私有404映射只限Host forbidden；每页授权与rows同一RR只读快照；列表不提供后续删除/发布许可，普通附件和绑定描述图分开；只复用现有owner入口且P7a不领取页面完成。

SOLID：消费方声明Inspect-only和List-only小接口，domain只处理纯游标，asset SQL只处理自身资产，owner仍决定宿主授权。未发现新增架构回归、数据结构或模块协议调整需求。

## 3. Local Fix：F1 全零草稿UUID返回500

审者评级Minor；主控复定为Important功能边界，按新增私有入口“不可存在资源同形拒绝”的实际合同，将其作为必须修复的功能边界处理，不因无越权就保留错误分类。

位置：`server/internal/modules/asset/app/draft_preparations.go:47`（审查候选行号）。生成绑定可解析全零UUID；无cursor时domain decoder早退成功；应用用例未拦截，进入现有issue AssetAccess.read的内部前置检查，返回原始`invalid issue asset request`，最终HTTP为500。普通不可能存在的ID被报成系统故障，可能引起无意义重试及日志噪声；未发现泄露或数据更改路径。

根因属于新应用入口的语义校验，不是owner或全局错误映射。修复限定为：认证及分页/cursor校验后、进入ReadTx前，全零draftID返回既有asset.not_found。无actor仍401，非法参数仍400/422，真实owner/rows数据库错误仍500。

必要回归已完成：`TestDraftPreparationsNilDraft`的有效actor首先触发“nil draft opened a transaction”失败；`TestDraftAssetPreparationsNilDraft`真实HTTP首先得到500而非404。随后在新用例进入ReadTx前增加三行检查，两个反例及准备件应用/HTTP回归通过；全零与随机不存在的Problem及头一致，无会话401、非法cursor400、owner与rows故障500均保持。原件`t6-f1-red`/`t6-f1-green`；定向正则没有选择domain测试，输出的`no tests to run`如实保留，domain完整验证由全门禁覆盖。

## 4. Declined to judge 与阶段界线

审者唯一暂不判断项为P7b页面和完整编辑器/复制交互：当前分支未交付页面。主控裁定保持这项明确的阶段边界；P7a只交付真实API恢复能力，P7b正式预检及后续页面测试承接，不以API证明代替页面。代价是用户可操作的恢复界面仍待P7b。

## 5. 合并判断

审者原结论With fixes；没有Structural或NeedsDecision项。主控F1局部修复与有效RED/GREEN已闭环，没有延后minor。修复前七门禁全通过（E2E385/385）；修复后完整门禁与准确最终分支CI仍为集成前置，不能以本报告替代它们或main CI。这份审查结论形成时尚未合并；最终集成证据见下。

主控最终本地复验：F1修复后七门禁全部通过，完整E2E385/385，证据见[P7a阶段记录§12](../v0/M5-files/reviews/P7a-draft-preparations-review.md#12-最终本地交付收据)。没有待修审查项；当时准确最终分支CI和main集成仍待分别核验，最终结果见下。

2026-11-05集成回填：最终候选756b90314009f00e487483729b5d19f76dbeb110的CI41859931318、noff maincf07c9fbe663b2e2c821c5988bb06933325b8ac1的CI41869045846分别三作业全绿；归档核验与正常清理完成，P7b正式预检完成且暂不实施。[完成证据](../v0/M5-files/reviews/P7a-draft-preparations-review.md#13-t7-集成归档与接续)。此为主控交付事实回填，没有增加第二次同范围审查。
