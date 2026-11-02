# M4 / closeout 文档预检记录

状态：文档预检通过，实施未开始。基线 `01e40e113e11365a4eeacc2794973de2751dde61`。权威为 [spec](../specs/closeout.md)、[plan](../plans/closeout.md)、M4设计§12收尾/§14；本记录不是M4最终review。

## 1. 预检方式

主控依deep-review顺序检查逻辑完整性、集成安全、验收覆盖；按brainstorming技能要求另派全新上下文的builder只读检查规格可执行性，结论 `nothing important left`。原文与范围保存在主目录 `.superpowers/sdd/M4-closeout-preparation-20261101/builder-check.md`；[正式报告](../../../reviews/re-28-M4收尾预检审核.md)记录发现及裁定。

用户仅授权准备；依02§3，以静态源码/合同/消费者和证据清单核验代替整段原型与逐Task重放。没有实现closeout，没有安装依赖或运行本阶段产品测试。P16精确main CI是本轮集成事实，不当作C12/性能已经完成。

## 2. 核对范围与已有基础

| 来源 | 核对结果与本轮限制 |
|---|---|
| M4§12/§14 | 14粗项重排为14Task；18条D逐项对应任务/证据/集成后置项，既有2个已勾子项保持原状 |
| §2/§3.5/§9.7 | 22故事及增长、26交错、C1–C12明确纳入实施目录；I21无页面/API、I22只有页面、I16PAT读任务结果，不造不存在的版本 |
| §3.23/§4.17/§8.8 | 当前44同步行/45行为差异行与README要求；预检核来源/目标，不声称已逐行签收 |
| 18Phase spec§5/review§6、8输入 | 抽取原段和source hash；逐后续责任分类纳入T12，不仅复制§13.2。8份顶部仍open，4份M4-closeout尚未创建 |
| P16 C12准备/最终review | 21组入口、原配置110唯一项+10 S2+5 cleanup=125/引擎；自动回归与最终控制者逐格截图判读分开 |
| 列表/搜索源码 | TestIssueListScale只有10k/100k，拟补20k；每规模13场景；搜索100k/13场景含中文两字。保留独立oracle/typed参数/SQL、模块与EXPLAIN不同口径 |
| 删除/池/清理源码 | 删除六项目各20k/660k行、真实job状态；池10/K2/queue64；runtime purge已有61/59天正确性但缺独立计时。T9新增opt-in测量，生产Runner和runtime角色不换成直调/mock |
| 旧锁断言 | 13文件16直接55P03分类、3个workspace等待helper消费者。实际函数是postgres.LockNotAvailable和pgtest.RowWaits；后者NOWAIT不可无差别代替等待超时 |
| dead/lint/关键词 | P16终态和完整证据为历史起点；T11仍用原AST/current typegen逐真实读者、全期改动范围/原14cap范围重核，不照抄353条候选为零 |
| 最终集成 | 分支验收先留集成待办；验收内容main绿后记录完成标记，再核最终main绿。收据外置，不循环为了写CI结果而制造新的“最终提交” |

原始盘点 `requirements-inventory.json`、`phase-handoffs.json`/分段全文、`legacy-lock-assertions.json`、`source-review-inventory.json`、`preflight-check.json` 在主目录准备根；静态检查与实际运行分列。

## 3. 局部裁定

spec C01–C05无待决项：14Task按验收单元重排；本轮静态预检；规模显式opt-in；最后完成标记分两次精确main事实；锁帮手归并保留原观测语义。理由、代价和受影响验证在spec§3，未改变架构/协议/权限owner或M6/M7验收归属。

## 4. 预检发现与修订

| 编号 | 级别 | 问题与已完成的文稿修订 |
|---|---|---|
| R1 | Major | 只复用P16自动125项无法满足最终C12：21组×3按角色/路径细分，成功截图、控制者真实判读/请求和最终main等价绑定明确到T5–T7/T14 |
| R2 | Major | 旧列表测量没有20k、普通F默认跳过规模、清理只有正确性：T8/T9明确env/精确测试名/独立oracle/样本/job计时和非空工作量；不得拿总test时间充数 |
| R3 | Major | 仅审closeout差异会漏整个M4集成风险：全期1eb9686a..候选与本分支01e40e11..候选两范围、全文分批清单和至少15有效性质反证各有归属 |
| R4 | Major | 先改总体“已完成”再等main绿违反阶段事实：T13保留集成待办，T14获准后先验收内容mainCI，再文档完成标记与最终mainCI；最后收据外置 |
| R5 | Major | 只改8个status会丢未来责任：逐条双向索引、M2§1/M0部署原文比较、M1收集箱细节、M6/M7自身页面与列表恢复条件、所有Phase移交纳入4输出 |
| R6 | Minor | 锁分类函数命名与NOWAIT/超时边界须准确：更正为postgres.LockNotAvailable；等待时长本身被验的路径保留真实等待，原专测字面量不清零 |
| R7 | Minor | “22故事都要页面/API”或“21码再各补升降”会扩大设计：按各故事适用版本，21为已有合法码（sort_order仅升序），不添加虚构故事/排序 |

以上为既定范围内的预检澄清，未实施产品修复。独立builder无新增实质问题；主控按最终文稿复核，无未解决的Structural Issue/Needs Decision。

## 5. 与实施计划的差异

本轮没有原型、Task重放、C12、规模测试或产品F。预检只验证计划是否可执行、源路径/帮手/边界是否准确；新增purge_scale_test.go和各最终验收台账明确为未来产物。实施发现的实际缺口仍须先红后修和复审。

## 6. 下一步与移交

分支 `codex/m4-closeout`，基线见头部；本轮只提交文档、保留worktree。负责人未来授权实施才按T1起步；完成后deep-review且先不合并。准备证据和未来实施证据须一同归档，P16稳定归档入口见 [P16 review §23](P16-web-cleanup-review.md#23-授权合并main-ci-与归档清理2026-11-01)。

## 7. 已知限制

未验的C12格、性能数字、八份输入关闭、四份输出和总体§9.4不提前完成；预检不担保运行时没有新问题。局部自主修，架构/跨模块责任/意外高风险先提交再上报；W11合并重跑仍败则停报。只在需要新验证时扩大重跑，不为文档改动重复已等价的整套产品测试。
