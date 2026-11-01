# M4/P16 前端清理：预检与后续实施记录

日期：2026-11-01。状态：文档预检通过，尚未实施。基线 main `313728cbc61003d7ed072c1e1e191e1cc2a3d75a`，分支 `codex/m4-p16`；[spec](../specs/P16-web-cleanup.md)、[plan](../plans/P16-web-cleanup.md)、[准备盘点](P16-web-cleanup-inventory.md)、[deep-review预检](../../../reviews/re-26-M4P16预检审核.md)。

## 1. 前置与方法

P15 的授权合并、准确 main CI、证据归档及正常清理见 [P15 review §13](P15-web-collaboration-drafts-search-review.md#13-授权集成main-ci-与归档清理2026-11-01)。本期分支从该 main 基线独立建 worktree，主目录仍 main；原生 worktree 工具不能指定负责人的路径，因此用系统 git worktree add。

本次授权只有准备；按 02 §3 用源码/测试/类型合同、只读 lint 和 P15 AST 输入等价核对替代产品原型/逐 Task 重放。没有安装 P16 依赖、产品改动、P16测试执行或 phase 变更。原 AST 报告的 raw hash、tested input 中三份非文稿 dirty 文件已核实；后续集成修复不改变 web/，最终基线以 integration-baseline-equivalence.json 绑定；不把复用说成新运行。

brainstorming 要求的 fresh builder-check 只读完整 spec/plan/inventory 与相关源码，返回 `nothing important left`；主控按 deep-review 三视角顺序核验，不声称三位独立审者。builder之后由主控明确 no-shadow 根规则键、当前any数量、类型文件不进生产入口及新config的显式lint，核准config忽略项需用相同规则临时配置去除ignorePatterns；进一步对8处旧no-shadow屏蔽作同规则副本对照，补入隐藏8条及无效2处的删除验收，再核最终链接/状态；范围与产品均未改变。阅读范围与静态证据保存在主目录 `.superpowers/sdd/P16-preparation-20261101/`。

## 2. 验收映射

下表是实施后的证据落点，当前全部待实施；准备通过不等于 A 条款完成。

| 标准 | Task与证据落点 | 当前状态 |
|---|---|---|
| A1 | T1完整last-reader表、M6/M7子交接与静态反例 | 待实施/回归 |
| A2 | T2 UI类型正负例、真实值/键盘/portal及所有消费者 | 待实施 |
| A3 | T3全仓诊断/屏蔽0、规则error、合法/非法输入、词法含义 | 待实施 |
| A4 | T4新AST逐行ledger、历史52行、动态读者/删除链 | 待实施 |
| A5 | T5 M4领域/全期改动零告警与各包精确cap | 待实施 |
| A6 | T6关键词、旧搜索具名扫描、正反例与未来边界 | 待实施/回归 |
| A7 | T7隐式getter/显式方法区分、M7三文件与原store测试 | 待实施/回归 |
| A8 | T8 README/两清单/交接子项与实际能力对账 | 待实施 |
| A9 | T9 S2入口/四身份精确请求与拥有者测试 | 待实施 |
| A10 | T10三引擎同范围清单、影响回归、最终C12留收尾 | 待实施 |
| A11 | T11完整F、deep-review/修复复审、准确分支HEAD CI | 待实施 |

## 3. 准备决定

spec D1–D8均为已批准设计内的落地决定，无待裁定项。保持十一Task，当前实际影响为32个选择器生产文件、46条已知no-shadow（38显示+8隐藏，含无效屏蔽共35待核文件）与453个M4 dead候选；历史数量保留来源、不拿它们当实施上限。共享last-reader接口留到M6/M7最后迁移；原生API/权限所有权不变。类型判别与UI运行、dead候选与实死、C12准备与最后验收分开记录。

## 4. 预检发现与修订

[deep-review报告](../../../reviews/re-26-M4P16预检审核.md)逐项列文稿修订。主要是补当前所有消费者和真实菜单event类型、保留主题对象/null/0/多选语义、约束类型负例进入真实tsc但不进入生产入口、全期改动文件告警与原AST逐行审计、0例外合法终态和C12边界。

没有产品缺陷被声称已修复，没有待决 Structural Issue/Needs Decision。T1/T7现存代码已符合“共享接口尚有读者则留”的选择，后续只需当前复核/交接，不能为凑任务重写接口。

## 5. 与原方法和设计的差异

遵用户“暂不实现”，不做产品原型/逐任务重放、不预填生成物或测试通过数；正式实施执行每Task五步与必要反证。设计十一粗项保持，但按基线数明确影响文件；32文件/34直接调用不是新增业务范围。所需共享泛型/no-shadow本就是P16批准范围，未扩大后端协议。

## 6. 移交事项

T1/T7把M6/M7真实读者与最后删除条件写入现有输入，closeout再汇成最终M4-closeout。M7视图服务/版本旧DTO、完整工作区标签、收集箱来源/系统创建者和lazy pending discard仍在各自阶段。全M4终审、控制者全路径C12、§3.23/§14、总体9.4、性能数字与八份输入顶部关闭继续留closeout。

## 7. 已知限制与待验证

静态候选453行尚未逐一完成T4裁定；103条显示+8条隐藏诊断的63文件待核范围不是固定上限，当前TS接口编译绿也不能证明泛型新增合同。类型子元素与父组件的泛型关系不能由独立JSX Option保证。现存M6/M7页面未接通的责任未改变，旧URL规则零命中不能代表全应用零旧请求。三浏览器准备还未运行P16测试，不能把P12–P15的旧矩阵当本期结果。

## 8. 准备交付核验

十一Task/五十五步骤和十一A均未勾，tools/keywords.json仍M4/P15；提交范围限文稿。相对链接、既有/新增路径、基线/主目录、实际盘点数字与未实施状态均由最终静态收据核对。准确准备提交及远端一致性以提交后的 `preparation-delivery.json` 为准，不能将文稿内预填SHA作为自证。

本地证据根 `.superpowers/sdd/P16-preparation-20261101/`：builder-check.json、baseline-inventory-summary.json、ast-input-equivalence.json、dropdown-callers.json、lint-baseline-summary.json及逐包原始diagnostics、planned-lint-scope.json、shadow-suppression-scan.json及原样/去屏蔽对照raw、历史/最后读者清单、source-review-inventory.json、static-preflight.json。产品实现证据另在授权后的worktree `.superpowers/sdd/P16-web-cleanup/`，当前尚未产生。
