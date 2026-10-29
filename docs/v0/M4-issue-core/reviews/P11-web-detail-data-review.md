# M4/P11 详情数据层：准备与预检记录

日期：2026-10-30。**结论：预检被P11-S1阻塞，未实施。** [spec](../specs/P11-web-detail-data.md)、[plan](../plans/P11-web-detail-data.md)、[deep-review记录](../../../reviews/re-14-M4P11预检审核.md)、[需负责人决定的合同缺口](P11-project-number-width-decision.md)。不得引用本页作为通过预检或产品完成证明。

## 1. 前置、身份与范围

P10b最终交付2aaef85c已授权--no-ff合并main `df5d3bbc1887486bd105fe6fc663f57f5b371992`，准确main CI40908365144 attempt1 server/web/e2e全绿，归档后正常清理分支和worktree，详见[P10b review§17](P10b-web-list-data-review.md#17-授权合并main-ci与归档清理2026-10-30)。

从此main创建 `codex/m4-p11` / `.codex/worktress/codex/m4-p11`；主目录一直main。用户只授权准备spec/plan和预检，本轮没有产品改动、依赖安装、原型/代码重放或产品测试。准备原件在主目录 `.superpowers/sdd/P11-preparation-20261030/`，在worktree外，未来归档不能遗漏。

## 2. 验收映射

| spec验收 | 本轮处理 | 产品状态 |
|---|---|---|
| A1–A3 服务/资源/确认 | 对照生成合同、既有root/permission/Reconciled与旧store调用，定义资源任务和请求/竞态测试 | 未实施、未运行 |
| A4–A6 协作/版本/草稿 | 校正无评论表情GET、cursor请求含义、原生DTO、M7边界和PATCH→publish | 未实施；A6项目刷新子项待P11-S1 |
| A7–A8 实例/提及/拒绝/宿主 | 原7实例+两旧IssueService提前接线，明确权限key与宿主owner | 未实施、未运行 |
| A9 错误/标签/宽度 | 标签“已有workspace目录”假设按M7边界修正；编号公开字段不存在 | 未实施；宽度合同待决 |
| A10 证据与终审 | 已列Task验证/至少15性质/F/准确CI及终审要求 | 本轮仅部分静态预检，非阶段终审 |

## 3. 方法与实际覆盖

使用brainstorming/writing-plans准备，按brainstorming明确要求另派全新只读执行者检查。主控deep-review依次从逻辑、集成、覆盖核文档；独立builder-check不是三位审者，也不是未来产品终审。

全文读取spec/plan；相关业务源已读root权限/写队列、Reconciled/SessionSWR、详情核心及子项/关联/link/comment/reaction/activity/draft stores、版本共享组件/M7服务、提及入口/部分宿主，以及现有issue.fetch/sub_issues.fetch/root.store/SessionSWR回归。大设计/生成schema/关键词按相关节与结构读取；77路径consumer-inventory是搜索库存，不能当77文件全文覆盖。项目字段缺口出现后暂停完整预检；尚未完成所有映射消费者/其余大测试文件的逐份全文覆盖，不能声称全分支审核无遗漏。

API清单来自本地YAML解析30个操作，不由网络文档推断；关键词期限静态核当前phase=M4/P10、无until=M4/P11，P11旧地址21个exact key期限M4/P12。没有运行未来计划里的Vitest/编译/F/变异，也没有用main已有绿色测试替代P11验证。

## 4. 发现、修订与复核

| ID | 级别/根因 | 处理及当前状态 |
|---|---|---|
| PF-1 | Major，文稿局部接口误记 | plan T2曾写IssueDetail.identifier，实际只有project_id/sequence_id；改为同代Project.identifier+sequence_id，失败按钮到项目列表，不保留旧workspace-views地址。主控核schema/真实route；未实施。 |
| BQ-1 / P11-S1 | Major，上游设计与公开合同冲突 | Project无next_work_item_sequence，不能仅前端刷新实现；两方案与代价已写决策记录。**待负责人裁定**，未修产品、未放宽守卫。 |
| BQ-2 / PF-2 | Major，文稿局部虚构目录 | 当前仅project标签API、labelMap仅已取目录；按现有设计/M3交接明确P11保留缺标签限制，完整恢复M7。spec§2.10/planT12修订，无新API或跨模块协议。 |

独立审者 `/root/p11_builder_precheck` 全文核spec/plan与必要源，发现BQ-1/BQ-2；其它权限令牌/队列、编号后置scope证明、M7版本DTO边界及publish确认未提出必须先回答的问题。原始结论/当时输入hash在builder-check.md；保存的builder-input快照晚于首次PF-1修正，不能冒称两份历史输入hash都等于快照。修订后的阻塞稿尚不称独立复审通过，待负责人裁定后一起限定复核。

## 5. 静态验证与证据边界

准备目录保存api-inventory.json、consumer-inventory.json、builder-check.md、修订快照、decision-evidence.json和最终文档静态检查回执。检查内容是链接/路径、生成合同缺口、关键词期限、14Task/开放验收、仅docs diff与main不动。实际执行命令/结果由回执记录；零产品测试是本轮边界，不是失败或跳过的产品绿。

P10b证据保持原hash与历史限制；本轮只增加集成事实，不重写已归档原件、不替历史变异补红绿。准备提交的SHA/推送回执放忽略目录，不循环写文档自身SHA。准确main CI已核；准备分支CI若触发仅记录其真实状态，不影响“尚未实施/预检阻塞”的结论。

## 6. 移交与未完成

先由负责人裁定P11-S1，再同步spec/plan/设计并完成预检复核；之后另待实施安排，从T1开始。P12搜索首调用、P13归档/profile/R36/R38、P14详情与父项搜索、P15协作/草稿/全局搜索、P16守卫终态不关闭。M7标签完整恢复与旧版本DTO的最终删除保持所属阶段。

M4§14、总体§9.4、输入handoff顶层open不变；M5–M8最终M4-closeout未创建。P10b已收尾，P11不是已完成阶段，不合并、不清理P11准备worktree。

## 7. 当前下一步

本轮提交可评审的准备稿与决策记录后停止。需负责人选择：A只调整列宽验收，B扩展项目公开只读协议。主控建议A。问题来源是设计/交接和生成合同的真实冲突，停下依据是负责人明确的跨模块纪律，不是技能额外增加的审批流程。
