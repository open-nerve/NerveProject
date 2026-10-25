# M4/P5 工作项列表与显示设置：评审记录

日期：2026-10-25。状态：**实施中；T1–T2完成，T3进行中；阶段尚未完成，不合并**。

| 项目 | 当前证据 |
|---|---|
| 分支 | `codex/m4-p5` |
| 基线 | main `e9782b8f7e7571b39a4eb8924b4b72a6e60eac51` |
| 工作树 | `.codex/worktress/codex/m4-p5`；主目录保持 main |
| 规格与计划 | [spec](../specs/P5-issue-list.md)、[plan](../plans/P5-issue-list.md)，16 Task |
| P4 接收 | [整分支审核](../../../reviews/re-01-M4P4整体审核.md)的三项 Minor 已修复并独立复核；最终分支 `a37f82ec146efca84793b6620aa3e777532bd75a` 的 [CI 40179571485](https://github.com/open-nerve/NerveProject/actions/runs/40179571485) 通过 |
| P4 合并 | `--no-ff` 双亲为 `589d9f01d4f8c30e55f449e430fe1104e2036866` 和 `a37f82ec146efca84793b6620aa3e777532bd75a`；合并后的 [main CI 40180521025](https://github.com/open-nerve/NerveProject/actions/runs/40180521025) server/web/e2e 全部成功 |
| P4 清理 | main CI通过后归档897个过程文件并逐文件SHA核对；旧工作树、本地及远端 `codex/m4-p4` 已删除，主目录保持main且干净 |

## 1. 评审方式

起步规格作者与独立预检者分开。独立者全文审查首版规格、计划和重点源码接缝，核对上级设计及 P1–P4 交接，不将签名核对说成编译或运行通过。按 [02 方法 §3](../../claude_to_gpt/02-method.md) 允许调整的范围，本阶段采用可执行的任务边界、真实 TDD/变异、逐任务独立评审和最终集成证据，替代一次性完成整段原型及逐字节重放。各项完成线不变。

原始预检记录、文件指纹和后续复核放在 ignored 的 `.superpowers/sdd/P5-issue-list/`；P4 完整过程证据在主仓库 `.superpowers/sdd/P4-issue-collaboration/` 归档。正式文档包含全部需要执行的结论，不依赖该本地归档才能理解规格。

P5 完成后仍先交负责人安排整分支终审，处理意见后才合并；P4 本轮的特别安排不扩展为后续阶段免审。架构回归、跨模块语义变动或意外高风险仍按负责人纪律处理。

## 2. 验收标准逐条核对

产品验收清单以 spec §4 为准；T1–T2已有下列实施证据，其他任务尚未完成。下表保留起步检查，后面追加逐任务证据。

| 检查 | 结果 |
|---|---|
| 绿色 main 基线 | 精确 SHA 和三个 CI 任务见上表；没有用 P4 分支 CI 代替 main CI |
| 工作树与依赖 | 从上述 main 创建独立工作树；`pnpm install --frozen-lockfile` exit0，使用现有锁文件，未增加依赖 |
| 本地架构基线 | `go -C server test ./internal/archtest -count=1` exit0，包耗时1.563s；没有放宽守卫 |
| 阶段关键词 | `phase=M4/P5`；没有在 P4/P5 到期的例外，`/user-properties/` 的 M4/P11 例外保留；`node tools/keywords.mjs` 为72规则、23例外、无命中 |
| 起步文档自检 | 16个任务顺序完整、产品验收均未勾选；53处本地链接无新增缺失。M4设计原有4个M5–M8收尾文件链接仍是未来交付，不提前创建或标完成 |
| 独立预检 | 首轮4项 Medium、2项 Low，均为局部计划/文档缺口；修订聚焦复核Approved，新增L3也已关闭，见§4 |
| P5 产品测试、性能、交错及变异 | T1–T2证据见下文；后续任务不能以起步检查或T1结果代替 |

### T1：连接运行参数，已完成

- 实施提交 `a306368fde93c0573b0049f5b413b0c4986bad54`；独立审查结论 Spec compliant、Approved，无未解决问题。
- ParseConfig之后强制两项运行参数，保留其他URL参数及UTC AfterConnect。两条同时借出的真实连接读回两项值，并验证application_name与UTC。
- 初始真实RED为两连接均读到on/auto；加入设置后GREEN。分别删除jit及plan_cache_mode赋值，均仅被相应业务断言杀死，原字节恢复后的控制组均绿。
- postgres整包非缓存通过，`make lint-go`两段0 issues，`make test`完整server/tools非缓存通过；没有契约/生成物/web变动。原日志、变异diff及与提交对应的源文件哈希在ignored `task-1-evidence/`，独立记录为 `task-1-review.md`。

### T2：共用列表词汇，已完成

- 实施提交 `6a6f87677072115c44fe14208ab5fff67d34eedd`；独立评审 Spec compliant、Approved，无待修问题。新增四个纯领域文件，DateFilter与唯一日期原子解析提前到T2，T3/T13复用。
- 测试覆盖19排序、项目/个人字段及分组、me/none、未知scope和日期边界。缺API的初始编译红单列；放行link_count正负码及个人assignee分组的两类实际变异被业务断言杀死，恢复后绿。
- domain整包与archtest非缓存通过，`make lint-go`两段0 issues。纯领域尚未接线，完整`make test`按实际影响留T3链末；没有把T1基线冒称T2全量。未单独捕获的四项早期命令由原实施者从真实工具返回原文补证，明确标为“工具输出转录”，未伪称当时原始日志；reviewer已核实。

## 3. 执行中的决定

1. 按既有 D2 授权切成16个任务，单列 `sort_order` 全链和普通目录替换，保留原15项的全部验收范围。任务与代价见 spec §3，预计规模与重切条件见 plan。
2. `shared.PageSize`、`ReadOnlySnapshot()` 与 Historical 已在 P4 交付，直接消费；普通 Visible 单独实现，不改变历史引用的宽目录语义。
3. HTTP 保留 M2 绑定先于认证的既定顺序。使用生成器已有标准扩展点，只映射 P5 两条 GET 上真实重复的已声明非日期 query；其他错误回到既有400。原始 query 经 strict middleware 深拷贝传入，不重写 URL、不修改平台或生成器。真实接口矩阵仍待 T9 证明。
4. 个人主页的目标成员与调用者分开进入引擎：关系过滤用目标，权限/OwnOnly 用调用者；目标来自已核验的路径对象，不能由 query 覆盖。
5. M4 §5.1 子项排序残留改回 P3 已裁定并交付的 `(created_at, id)` 升序。P5 排序 PATCH 不改变该集合接口行为。

6. T3不增加只有测试调用的生产全序比较器。纯领域验证真实键归一化、类型、canonical及日期条件保留；null方向、UUID方向、多值min和日期AND/OR的实际变异移至T5真实SQL，T12收口，全部完成要求保留。
7. 游标拼写按spec第2.5节固定：微秒UTC、按排序定型及可空性、精确int64、有限float64及正负零统一。不能把RawMessage直接回写当唯一规范拼写。SQL位序数值转换仍待T5实证。

## 4. 预检发现与处理

| 编号/级别 | 原问题 | 修订要求及证据落点 | 当前结论 |
|---|---|---|---|
| M1 / Medium | 目标成员 ID 未进入 Listing/SQL 输入，可能把调用者误当查询目标 | 显式 `ListingInput.TargetMemberID`、`ListRead.TargetMemberID`、独立 `relation_user_id`；T10 的 A≠B 三关系及 OwnOnly 反例/变异 | 已修订，独立复核关闭 |
| M2 / Medium | 新交错只列普通运行，`-race` 被写成可选 | T12/T13 与最终门明确 `-race -count=5`、真实匹配函数及无40P01；包含快照、偏好并发与受影响排序锁路径 | 已修订，独立复核关闭 |
| M3 / Medium | 有词汇表接受测试，未明确全部分组码的真实 SQL/续页覆盖 | T12 覆盖项目7码、个人5码、每个合法子分组字段；独立期望集合、计数、完整数组和翻页 | 已修订，独立复核关闭 |
| M4 / Medium | 缺标准 spec 验收表、局部裁定代价、逐来源交接与方法说明 | 标准7节及附录；验收不打勾；明确静态预检与运行证据的界线 | 已修订，独立复核关闭 |
| L1 / Low | 排序字段文件表漏掉 IssueFields 真正所在的 rows.go | T8 补实际文件路径，不重复声明类型 | 已修订，独立复核关闭 |
| L2 / Low | app fake Rows 无法证明 Store 构造出的 sqlc 数组非nil | T5 核对生产参数映射或真实执行边界，直接NULL的SQL防御另测 | 已修订，独立复核关闭 |
| L3 / Low（复核发现） | 文档修订误把本仓库标准库 `"uuid"` 写成第三方包 | 对照现有 rows.go 等导入，spec/plan恢复标准库；未改产品或依赖 | 已定点复读关闭 |

2026-10-25独立聚焦复核结论 **Approved**；首轮4项Medium、2项Low及修订中新发现的L3全部关闭。批准仅限起步规格与计划，后续任务的产品证据仍待实施。

本轮没有发现需负责人重新裁定的架构问题。SQL数值键的无损转换仍是 T5/T12 必须实证的实施风险，不能将候选 `numeric` 转换写成已验证结果。

## 5. 最终实现与 plan 的差异

阶段尚未结束。T1与计划一致，生产仅增加两项参数；测试额外核对无关application_name保留。后续差异随任务登记。

## 6. 移交事项

| 接收方 | 范围 |
|---|---|
| P5 实施者 | T1–T2已完成；从T3继续按spec/plan逐任务交付、测试与评审 |
| P8 | `link_count` 字段与两个排序码；列表/搜索共用词汇 |
| P10b | 共用排序样例、生产前端码点比较器与vitest；旧筛选store和 `/user-properties/` 迁移 |
| P12–P15 | 接口故事增长为页面故事，列表/看板/日历/详情与访客页面 |
| M8 | 大规模与跨项目扫描成本、是否需要排序索引特化；接收最终生产SQL性能证据 |
| M4 收尾 | handoffs仍open，按各阶段处理结果最终关闭；不能用P4完成代替M4整体完成 |

## 7. 已知限制

本阶段只完成起步时，新的 SQL、API、迁移、性能和测试尚不存在。数值转换、生成器具体接线及100k成本须以实施证据落定；这里不预报通过。

本地Firefox缺失和已知W11偶发仍按既定纪律处理；绿色P4 main CI已覆盖该基线的完整浏览器任务，不代表未来P5变更已经获得同样证据。
