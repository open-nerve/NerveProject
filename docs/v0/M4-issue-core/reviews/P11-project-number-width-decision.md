# P11-S1：编号列宽依赖不存在的项目字段

日期：2026-10-30。**负责人已选择方案A（2026-10-30）：按展示数据计算列宽。** 合同缺口的范围决策已关闭，文档预检已通过，见[阶段review](P11-web-detail-data-review.md)；本次仍不开始实现。 这不是P10b回归；P10b已完成集成和准确main CI。发现来自P11独立builder-check，主控已核源码和生成合同。

## 1. 问题与证据

发现时的原设计要求工作项编号列宽读取 `Project.next_work_item_sequence`，创建/发布后刷新项目，以便例如 `WEB-9` 到 `WEB-10` 时多留一位空间。但现有Project接口没有该字段，刷新也取不到。

| 证据 | 当前事实 |
|---|---|
| [M4设计§7.3](../M4-design.md#73-services-与-storesp10ap10bp11)、[M3 handoff§6](../handoffs/M3-closeout.md) | 发现时遗留 next_work_item_sequence/写后项目刷新要求；现已按§4裁定修订 |
| [Project合同](../../../../api/modules/project.yaml) 的 components.schemas.Project | additionalProperties=false；所有properties/required均无该字段 |
| [生成TypeScript](../../../../web/packages/api-client/src/schema.gen.ts)、[项目store](../../../../web/apps/web/core/store/project/project.store.ts) | 直接消费当前生成Project，没有可读取的next序号 |
| [项目HTTP序列化](../../../../server/internal/modules/project/adapter/http/projects.go) | project()未输出该字段 |
| [项目内部SQL](../../../../server/internal/modules/project/adapter/postgres/queries/issue_ports.sql) | 创建工作项时仅内部 last_issue_sequence 加1并返回，用于分配编号；不是公开项目DTO |
| [列表编号宽度](../../../../web/apps/web/core/components/issues/issue-layouts/list/block.tsx) | 当前传固定1位，注释已说明公开项目没有该字段；不是已经存在而漏接的前端getter |

不能增加 TypeScript可选字段、强转any、从当前issue.sequence_id+1伪造项目全局值，或继续用刷新掩盖缺失合同。问题根因是设计/交接要求与已经实现的公开合同不一致。首次依负责人“跨模块调整先提交再上报”的纪律暂停，已提交c437f6ea后上报；下文保留当时两方案及现裁定。

## 2. 方案A：由当前展示数据计算列宽（已采用）

调整设计与M3交接中这一条验收：编号宽度属于当前列表布局，由该布局**已加载且当前仍可见**工作项的完整编号字符串计算统一宽度；同一展示scope加载更多时允许增宽，删除/筛选缩小可维持当前宽度以免抖动；切项目/工作区/登录会话或权限范围改变时重新计算。独立详情按自身编号显示。不把未读取或失权工作项的编号当输入，不扫描不可见canonical作为项目总量。

范围：前端显示helper/布局调用方、对应Vitest/页面回归、设计/spec/plan/M3交接的准确处理结果。不新增项目API字段，不为列宽请求项目，不新增counter/实体缓存；显示宽度局部状态不是服务端序号事实。草稿publish仍须原计划的root创建确认，只有“为了取不存在next序号而刷新项目”这一子项取消。

验收样本：同列9/10/100位数、项目标识不同长度、分页加载更长编号、过滤/删除、权限收回和新会话、空列表。列宽足以容纳当前显示的编号；不承诺提前为尚未加载或未来生成的编号留位。代价：下一页第一次出现更长编号时，列可能变宽。这是明确更改展示验收，不声称实现旧字段。

## 3. 方案B：增补项目公开只读合同（未采用，历史比较）

保留原设计含义，把next序号作为项目公开只读字段：需同步Project OpenAPI、生成物、项目domain/app/adapter读取和HTTP序列化、前端生成DTO/夹具以及创建/发布后的权威刷新。字段不可写，不把它当issue数量；需明确空项目值、最大编号边界、并发创建/草稿发布与数据库计数的一致性。

需同时决定其对各角色的可见性，尤其“仅看自己工作项”的访客是否可读项目全局分配进度；测试不能默认所有角色字段相同。还需核项目读取的缓存/版本语义：内部序号改变不当然改变Project.updated_at，不能让旧项目GET覆盖新字段或用毫秒/客户端时间做版本补丁。

范围：跨project/issue边界、公开API与生成客户端、服务端/前端测试；需在现P11计划前增加合同任务并重做预检，不能作为T12前端小修直接混入。可使用现有数据库计数，不必先假定需要新依赖或迁移。代价是本段扩大为跨模块协议工作，实施及验收明显增加。

## 4. 裁定与后续验证

负责人回复“A：按展示数据计算列宽（推荐）”，明确采用A，不新增项目公开字段。spec§2.11将“可见”明确为权限/筛选/展开后的已加载行，包括视口外行及已展开子项；每个布局统一列宽，权限/scope变化重算。T8不再为列宽刷新项目，T12承担显示规则与测试；M3交接只修这一条，不关闭整个handoff。

已按裁定同步修订M4设计、spec、plan与M3交接这条的处理边界；独立builder-check复核及deep-review映射覆盖已完成，静态验证结果写入阶段review和准备回执。当前所有Task/A1–A10开放、phase仍M4/P10；本轮用户只授权准备，裁定本问题也不自动授权产品实现。
