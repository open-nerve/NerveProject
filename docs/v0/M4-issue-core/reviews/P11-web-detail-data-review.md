# M4/P11 详情数据层：准备与预检记录

日期：2026-10-30。**结论：文档预检通过，未实施。** 负责人已裁定P11-S1采用A，设计/spec/plan/交接同步后完成复核。[spec](../specs/P11-web-detail-data.md)、[plan](../plans/P11-web-detail-data.md)、[deep-review记录](../../../reviews/re-14-M4P11预检审核.md)、[已裁定记录](P11-project-number-width-decision.md)。本页不证明P11产品完成，也不授权开始实施。

## 1. 前置、身份与范围

P10b最终交付2aaef85c已授权--no-ff合并main `df5d3bbc1887486bd105fe6fc663f57f5b371992`，准确main CI40908365144 attempt1 server/web/e2e全绿，归档后正常清理分支和worktree，详见[P10b review§17](P10b-web-list-data-review.md#17-授权合并main-ci与归档清理2026-10-30)。

从此main创建 `codex/m4-p11` / `.codex/worktress/codex/m4-p11`；主目录一直main。用户只授权准备spec/plan和预检，本轮没有产品改动、依赖安装、原型/代码重放或产品测试。准备原件在主目录 `.superpowers/sdd/P11-preparation-20261030/`，在worktree外，未来归档不能遗漏。

## 2. 验收映射

| spec验收 | 本轮处理 | 产品状态 |
|---|---|---|
| A1–A3 服务/资源/确认 | 对照生成合同、root/permission/Reconciled及旧调用，定义请求/竞态任务并保留既有回归 | 未实施、未运行 |
| A4–A6 协作/版本/草稿 | 校正评论表情读取、cursor请求含义、原生DTO、M7边界和PATCH→publish；草稿清空/无项目/拒绝已落到消费者任务 | 未实施、未运行 |
| A7–A8 实例/提及/拒绝/宿主 | 原7实例+两旧IssueService提前接线，四个真实提及入口、权限key与宿主owner明确 | 未实施、未运行 |
| A9 错误/标签/宽度 | 标签范围按M7边界修正；宽度按A统一/增宽/scope及权限重置 | 未实施、未运行 |
| A10 证据与终审 | 已列Task验证/至少15性质/F/准确CI及终审要求 | 本轮完成文档预检，非阶段终审 |

## 3. 方法与实际覆盖

使用brainstorming/writing-plans准备，按brainstorming要求另派全新只读执行者检查。主控deep-review依次从逻辑、集成、覆盖核文档；独立builder-check不是三位审者，也不是未来产品终审。

全文读取spec/plan及映射服务、store、直接消费者；原consumer-inventory的77条关键词路径在恢复预检后逐批补完全文，另读root权限/写队列、Reconciled/SessionSWR、成员目录、List链路、草稿表单/provider与相关测试。issue.fetch/sub_issues.fetch、issue.store/issue-changes、root.store、reconciled/SessionSWR、base.loading/base.clone、workspace-draft/root、list-data/group-consumers/removal等回归已核。逐路径/行数/hash与读取方式见review-coverage.json；consumer-inventory本身仍只是搜索库存，不是自动产生的阅读证明。

逻辑视角核原生字段、payload清空、列宽与异步失效；集成视角追实际service/DTO读者、M7边界、宿主和删除旧服务的编译依赖；覆盖视角对照既有回归与各Task的请求、拒绝、竞态、确认和变异目标。大设计按相关节、YAML/生成schema/关键词按合同结构核查，不冒称全文审查全部生成物或全仓库。已知旧实现与目标不符的路径是实施输入，已映射Task，预检通过不是旧代码已修好。

API清单来自本地YAML解析30个操作；当前phase=M4/P10、无until=M4/P11，P11旧地址21个exact key期限M4/P12。没有运行未来计划的Vitest/编译/F/变异，也没有用main已有绿色测试替代P11验证。

## 4. 发现、修订与复核

| ID | 级别/根因 | 处理及当前状态 |
|---|---|---|
| PF-1 | Major，文稿局部接口误记 | plan T2的IssueDetail.identifier改为Project.identifier+sequence_id，错误按钮回项目列表。核schema/真实route；文稿已修，产品未实施。 |
| BQ-1 / P11-S1 | Major，上游设计与公开合同冲突 | Project无next_work_item_sequence，先提交c437f6ea再上报。负责人选择A，按已加载可见行统一列宽、scope/权限重算，零列宽专用HTTP；设计/spec/plan/M3交接同步。决策关闭，未修产品、未放宽守卫。 |
| BQ-2 / PF-2 | Major，文稿局部虚构目录 | 仅project标签API、labelMap仅已取目录；spec§2.10/planT12保留缺标签限制，完整恢复M7，无新API。 |
| PF-3 | Major，回归迁移依赖未写实 | issue-changes仍spy旧IssueService，base.loading和draft/root用旧请求；plan全局/T2–T4/T8同步迁原生fixture，保留R31/R32、弹窗加载/克隆及筛选隔离，不删断言换绿。 |
| PF-4 | Major，草稿边界未落到消费者 | payload不接受null，Draft无created_by且可无项目；spec§2.8/planT8补键删除、顶层project_id区别、无项目展示/保存/删除、拒绝留输入及测试。完整页面故事仍P15。 |
| PF-5 | Minor，具体调用表漏项 | T9写实四个searchEntity调用；T4列CycleStore无调用旧服务三处删除；T3/T8明确确认回调不得二次写，M6目录回归保留。未扩M6/M7业务。 |

独立审者 `/root/p11_builder_precheck` 首次全文核spec/plan发现BQ-1/BQ-2；裁定后全文复核A修订稿，随后限定复核PF-3/PF-4和最终调用表，均返回“nothing important left”。原始结论/输入hash在builder-check.md；三次复核及hash在builder-recheck-a/local/callers.json。历史builder-input快照晚于首次PF-1修正，不能冒称历史输入hash都等于快照。最终状态文字与T3既定DOM断言的目标文件补注另存hash桥/静态回执，不冒充独立审者检查了后来的提交SHA。

## 5. 静态验证与证据边界

准备目录保存api-inventory.json、consumer-inventory.json、builder-check.md、修订快照、decision-evidence.json、review-coverage.json及最终静态回执。检查链接/路径与锚点、合同、关键词期限、14Task/10条开放验收、仅docs diff与main不动；源hash与已读基线一致。历史blocked回执保留，不覆盖为通过。实际命令/结果由回执记录；零产品测试不是失败或跳过的产品绿。

P10b证据保持原hash及历史限制，不替历史变异补红绿。准备提交SHA/推送回执放忽略目录，不循环写文档自身SHA。准确main CI已核；准备分支CI按实际提交单独记录，不能把它当P11新功能的产品证明。

## 6. 移交与未完成

P11-S1裁定、修订和预检复核已完成；另待实施安排，从T1开始。P12搜索首调用、P13归档/profile/R36/R38、P14详情与父项搜索、P15协作/草稿/全局搜索、P16守卫终态不关闭。M7标签完整恢复与旧版本DTO最终删除保持所属阶段。

M4§14、总体§9.4、输入handoff顶层open不变；M5–M8最终M4-closeout未创建。P10b已收尾，P11不是已完成阶段，不合并、不清理P11准备worktree。

## 7. 当前下一步

提交并推送通过预检的文档，保留codex/m4-p11分支/worktree，不合并。只有负责人安排实施后才核基线并从T1开始；本轮没有待补答的合同问题。原停下依据和两方案代价保留在决策记录，A裁定关闭该阻塞，不自动授权实现。
