# M5/P7b 阶段评审记录

> 2026-11-05：正式预检通过后已获准实施；预检历史结论保留，当前执行见§8。分支`codex/m5-p7b`，基线main`cf07c9fbe663b2e2c821c5988bb06933325b8ac1`。P7a main CI 41869045846及归档/正常清理已核；

## 1. 评审方式与范围

[主spec](../specs/P7b-content-assets.md)、[内容资源](../specs/P7b-content-lifecycle.md)、[编辑器](../specs/P7b-editor-images.md)与[plan](../plans/P7b-content-assets.md)共同界定16个Task。一次fresh builder对照冻结P7a实际代码逐Task核查，主控处理1项局部编排遗漏；最终main产品树与审查候选一致。正式报告见[re-49](../../../reviews/re-49-M5P7b预检.md)。

这是实施前预检，尚未开始P7b整分支deep-review；原P7、P7a审查与当前前端范围分别记录，不冒充同一轮完成。

## 2. 验收状态

A1–A8全部开放，T1–T2已实现，尚待页面与阶段联合验收。P5/P7a的API、P6的上传及权限回归仅为前置；P7b真实页面、三浏览器、local/S3、有效变异和完整七门禁必须在本阶段新运行。

## 3. 局部裁定

主spec§3的11项为绑定合同：唯一UploadSession/消费方小策略、无new_issue、私有null项目草稿、编辑器只处理图片、评论普通附件与正文分离、节点撤销不删资产、有限原始下载、可选图片能力/未来领域不启用、普通副本Issue与草稿副本Draft、遵守现有草稿owner寿命、权威重读保留本地输入。

两个预检补充的理由与代价：

- 草稿owner现为工作区集合；保留它在同工作区项目变化下的保守失效。资产不得私存DTO绕过，代价是可能需重读草稿；不顺带改M4权限寿命。
- IssueDetailHost在条件子树外持有本地输入/基准/已确认命令结果，解决普通附件权威重读引起的表单卸载。旧view/URL/请求照常结束；新owner及资产核验后才编辑，冲突提示、不重发未知写，真实权限/宿主消失永久清空。代价是增加小型受控状态与联合回归，传输可能需显式恢复。权限观察器、owner和通用读取算法不改。

原已批准P7恢复协议保持：仅作者分页查询、每页重新授权；明确选择丢弃，普通附件与绑定描述图删除行为不同；列表缺席不等于删除成功。没有新的架构待决项。

## 4. 发现与处理

唯一builder发现：IssueVerification.pending使规范Issue getter返回undefined，详情/peek移除正文，组件本地未发送评论及失败autosave可能丢失。按§3同步资源spec§4.1、主spec A2、编辑器保存、T3/T5/T6/T9/T13及联合变异要求。

主控额外核实际title-input与host嵌套，明确标题也纳入；宿主读取401/403/404与临时故障分开，前者退休状态；写422仍保留。审者没有看修订后的产品（本轮无产品修改），因此不冒称独立复审通过；主控合同自检与main输入一致性检查闭环。

## 5. 与原草案的差异

正式以P7a最终main为基线；补准备件生成DTO/端点/cursor规则，T13使用实际stories/files与fixtures/file-story.ts；原恢复/副本接收义务保留。明确草稿owner寿命与权威刷新输入所有者，不放宽架构守卫或新增依赖。phase继续M5/P7，不改工具支持的数字格式。

## 6. 移交事项

P7b T1–T13实现与证明F8–F12/F16/F17页面、准备件恢复、旧消费者删除；T14完整门禁、T15一次整审与修复、T16 noff/准确mainCI/全量归档及P8仅预检。P8/closeout承接最终故事矩阵、默认部署/容量、维护项、warning/死成员及全部handoffs关闭。

## 7. 限制与证据

本轮没有实施、安装依赖或运行P7b产品测试，48个Step/8个产品验收仍空。预检只证明合同与实际入口可衔接，页面交错必须在实施中验证；若需要更改M4通用权限/读取语义，先提交证据上报。P7a完整归档/清理与准确CI见[P7a§13](P7a-draft-preparations-review.md#13-t7-集成归档与接续)。P7b原始builder与准备资料、当前preflight输入清单/receipt保存在本阶段ignored目录，实施后随阶段归档。

## 8. 实施记录

2026-11-05负责人授权从T1开始，P7b完成整审后按原节奏集成并继续P8/M5最终收尾。根目录仍main，现有codex/m5-p7b worktree执行；此前“暂不实施”是预检交付时点。阶段验收仍逐条等待实际证据。

T1实现与验证：已有ApiClient消费生成Asset/Content/AttachmentPage/DraftAssetPreparationPage，原始字节下载独立fetch且有取消/期限/大小上限。首轮有效接口/下载断言RED，随后115/115聚焦测试通过；下载成功测试曾因jsdom FileReader任务与fake timers等待方式不符超时，修正测试为既有有期限settle，原失败完整保留。类型检查通过；修改8文件oxlint零warning；完整make test-web为18/18成功（17缓存，web实际重跑）。T1门禁收据在t1-types/t1-lint-2/t1-test-web及task-1-tests，产品输入前后不变。

原件位于本worktree `.superpowers/sdd/P7b-content-assets/`：run-check.py记录每次命令、真实exit、耗时、准确HEAD及前后输入SHA清单。安装使用锁定pnpm且输入未改；基线make test-web为18/18全部缓存命中，不冒充新运行。既有phase=M5/P7按P7a/P7b共同数字阶段合同保持。

T2实现与验证：UploadSession只消费小策略，资料策略保留P6权限；内容view使用唯一PermissionReads与规范owner。六用途、私有null项目草稿、guest评论、作者限制和永久撤回共187项通过；三项变异（取消后发布、误套封面管理员角色、ID返回复活）均触发AssertionError并核SHA恢复。替换选择时clear的同步observer可销毁会话，补重入后current检查根因修复，专用RED保留。初版四个评论用例失败源于fixture作者actor与实际账号u-actor不一致，修正fixture，不放宽权限。类型、14个修改源码文件零warning通过；全前端18/18（15缓存）通过。原件t2-selection-red、t2-green-2、t2-mutation-receipt、t2-types、t2-lint-2、t2-test-web。
