---
status: open
from: M5/closeout
to: M6
created: 2026-11-06
---

# M5 交给 M6：资产的接入方式、列表中的附件计数、清理的协议与首个投递任务的 CLI

这是接收方尚待完成的工作。转交只表示责任和验收条件已落实到此处，不表示本页功能已经实现或通过验收。
来源为 [M5 设计 §13](../../M5-files/M5-design.md#13-输入交接与后续交接)、[P8 阶段记录](../../M5-files/reviews/P8-final-acceptance-review.md)与其中的 T12 最终矩阵；M5 的最终集成与 main CI 由 M5 收尾记录，不以本页替代。

<a id="M5-M6-01"></a>
## M5-M6-01 迭代与模块现在不是资产的宿主

- **现状**：资产用途由 OpenAPI 的 `AssetPurpose` 给出，共十种（`api/modules/asset.yaml`；Go 为 `asset/domain/scope.go` 的 `Purpose` 与 `ScopeKind`，数据库为 `file_assets_scope_check`）：`USER_AVATAR`、`USER_COVER`（用户本人）、`WORKSPACE_LOGO`（工作区）、`PROJECT_COVER`（项目，或建项目前以工作区为 id 的 `new_project`）、`ISSUE_ATTACHMENT`/`ISSUE_DESCRIPTION`（工作项）、`COMMENT_ATTACHMENT`/`COMMENT_DESCRIPTION`（评论，或发评论前以工作项为 id 的 `new_comment`）、`DRAFT_ISSUE_ATTACHMENT`/`DRAFT_ISSUE_DESCRIPTION`（草稿）。迭代与模块的表单是纯文本（`components/cycles/form.tsx`、`components/modules/form.tsx` 的 `TextArea`），进度侧栏只读，没有图片或附件。
- **若 M6 让迭代/模块带图片或附件**，照现有宿主（工作项、评论、草稿）的接法新增宿主，不另起一套：
  1. 用途与范围加入 `asset/domain/scope.go`、`content.go` 与 YAML 枚举；迁移给 `file_assets` 加 `<host>_id … ON DELETE NO ACTION` 外键与新的范围检查分支（先例 `00034_asset_file_assets.sql`）。
  2. 宿主模块导出 `AssetAccess{Inspect, Check}` 与 `AssetHostNotFound`/`AssetHostChanged`（先例 `issue/module_assets.go`、`issue/app/asset_access.go`），在 `bootstrap/asset_access.go` 加一支；宿主行需要锁时加锁端口（`bootstrap/asset_ports.go`）。
  3. 导出 `Asset<Host>Facts` 返回 `AssetParent{ID, DeletedAt, DeletedByID}`，在 `bootstrap/asset_parent_reader.go` 加一批读取；`asset.cleanup` 的行阶段据此判断父删除。
  4. 删除：宿主删除事件由 `asset/app/deletion_events.go` 订阅（现订 `issue.deleted`、`issue.comment_deleted` 同步软删，`project.deleted`、`workspace.deleted` 在发布者事务内投递 `asset.delete_scope`），或像草稿一样调直接端口；`DeleteScopeRows` 现在只处理 issue/comment/draft，需同步扩展。
  5. 富文本图片：issue 模块从 `image-component[src]` 取规范 UUID，写宿主前 `LockContent`、写后以同一时刻 `BindContent`（`issue/app/create_core.go`）。新宿主须扩展 `asset/app/content_bindings.go` 的宿主检查与 `content_binding_rules.go` 的用途与放置规则；前端 `store/asset/content-view.ts` 的 `ContentHost` 与 EditorImages 能力随之扩展。描述中只存资产 ID，不存签名地址。
  6. 物理清理：外键加入 `ProtectedParentReferences`，宿主的 purge 步骤标 `Protected: true`，模块在 `bootstrap/app.go` 的 `newPurgeDeleted` 中排在 asset 之后、被它引用的父模块之前；`asset_protection_catalog_test.go` 与 `purge_catalog_test.go` 会核对覆盖。
- **不能做的**：跨模块 JOIN `file_assets`；绕开宿主的权限检查读文件；把旧的 Plane 文件地址或 `/api/assets/v2` 带回。

**何时验收：** 迭代/模块第一次带文件之前。上传、拒绝、取消、换宿主与迟到结果、权限矩阵、删除立即不可见、60 天清理与存储重试，均需与 M5 的 F8–F15 同等的 API、字节与页面证据（local 与 S3）。

<a id="M5-M6-02"></a>
## M5-M6-02 迭代与模块列表中的附件计数

M5 给工作项加入真实的 `attachment_count`、排序码与显示属性“附件”（M4 交接 M5-01）。计数由 issue 模块经 asset 的 `AttachmentCounter`/`AttachmentCountCatalog` 端口取得（`bootstrap/asset_attachments.go`），只数已绑定、未删除的工作项附件。M6 在新列表引擎恢复迭代/模块的工作项列表（M4 交接 M6-02）时，附件列与排序随共享的列表设置出现，不另算计数。

**何时验收：** 迭代/模块列表开放时，用真实附件核计数、排序、分页与换会话；附件增删后计数随规范读取更新。

<a id="M5-M6-03"></a>
## M5-M6-03 新关系表的清理走有界、可续跑的协议

- 共享的物理清理 `cleanup.purge_deleted` 自 M5/P2 起按模块顺序 asset → issue → project → workspace → identity 分页执行，每次运行的预算为 `files.cleanup_budget`（默认 10 秒），用尽即把游标写入任务输出并 snooze，下一次运行从同一游标、同一截止时间继续（`bootstrap/purge.go`、`purge_worker.go`）。M6 的 `cycle_issues`、`module_issues` 等新表加入时，其 purge 步骤属于所在模块的 `Purger`，在同一预算内运行；单条语句必须有界（`LIMIT` 批量），被预算取消的语句会回滚并在下一次运行重做。
- 规模验收：`TestPurgeScale`（`NERVE_PURGE_SCALE=1`）的追踪器只认三种语句形状——单语句分页（`WITH candidates AS MATERIALIZED … DELETE FROM <表>`）、受保护表的候选 `SELECT id FROM <表>` 与按 ID 的 `DELETE FROM <表> … = ANY(…)`；新表在项目范围内时加入 `purgeScaleTables`，新语句不合这三种形状会被判为未知。P8 在 M5 代码上的结果：660,026 行一次运行 5.0 秒（M4 同数据 12.2 秒）；统计陈旧时仍有界并续跑完成（6 次运行合计 52 秒）。
- 删除事件：M6 删除迭代/模块不影响资产；工作项、评论、项目、工作区的删除对资产的影响见 M5-M6-01 第 4 步。

**何时验收：** 每张新表合入时同步扩展 purge 目录测试；M6 收尾前在 M6 代码上重跑 `TestPurgeScale` 与 `TestTheDeletionsAtTheScaleTarget`（`NERVE_MEASURE_SCALE=1`）。

<a id="M5-04"></a>
## M5-04 首个投递任务的 CLI（原文转交）

下面完整引用 M4 交给 M5 的 M5-04，其中又完整引用 M2 原第 1 节；相对链接按原出处解释，原文不改。可点击来源：[M4 → M5 原交接](../../M5-files/handoffs/M4-closeout.md#M5-04)。

<!-- M4-M5-04-BEGIN -->
````text
## M5-04 首个投递任务的 CLI

下面完整引用 M2 原第 1 节；其中相对链接按原出处解释，原文不改。可点击来源：[M2 → M4 原交接](../../M4-issue-core/handoffs/M2-closeout.md)。

<!-- M2-SECTION-1-BEGIN -->
```text
## 1. 命令行的"只投递"River 客户端（负责人：做好记录，以后别漏了）

- **负责人的要求**：2026-09-26 批准把它推迟到 M4 时说，"做好记录，以后别漏了"。M4 用不到它，也要把这一节原样转进下一个 M 的交接，不能随本交接关闭而丢掉。
- **做什么**：第一个"用例会投递 River 任务、又能从命令行触发"的命令加入时，`platform/jobs` 提供一个只投递的客户端（不配队列，不调用 `Start`，`river@v0.47.0/client.go:89-95`），`bootstrap` 的命令组合把它交给这些用例。
- **为什么**：命令行的组合只有连接池和 `identity` 的管理用例，没有 River 客户端（`server/internal/bootstrap/users.go:22-26`，M2 设计 3.17）。服务用的客户端（`server/internal/platform/jobs/jobs.go:70` 的 `New`）配队列、启动 worker，命令行不能用它。用例在事务里用 `InsertTx` 投递（M2 设计 3.15）；命令行上没有客户端时，任务要么没投递，要么用例在运行中失败。M2 的命令都不投递任务，所以推迟（M2 设计 3.15、3.17、13.2）。
- **可能早于 M4**：M3 给停用加端口（[M3 的交接](../../M3-workspace-project/handoffs/M2-closeout.md)第 6 节），`nerve users deactivate` 也走这个用例；端口的实现投递任务时，这一项提前到 M3。
- **建议的守法**：投递任务的端口在用例的构造参数里必填，命令的组合漏了它，在构造时就失败，而不是运行到一半；`server/internal/bootstrap/users_test.go` 这样的组合测试覆盖每个命令。
- **关闭条件**：M3 已为停用的端口加入这个客户端时，M4 的 review 核对它合这一节（没有队列、不调用 `Start`，组合测试覆盖每个命令）并写明；否则，M4 加入了这样的命令时，命令的组合有只投递的客户端，一个测试在测试库上运行这条命令，核对 `river_job` 多了这条任务、事务回滚时没有。M4 结束时仍没有这样的命令，M4 的 review 写明，并把本节原样写进下一个 M 的交接，本交接才能改为 `closed`。

```
<!-- M2-SECTION-1-END -->

**当前事实与责任：** 只投递的客户端已在 `platform/jobs`（`NewInserter`，M4/P6）；M3、M4 没有投递任务的命令，现有 CLI 组合无需构造它。**第一个投递任务的命令加入时，把它交给命令行的组合**，按 §3.20 显式声明投递的 kinds，启动时验证对应 worker。

**何时验收：** 首个投递任务的 CLI 提交中完成。真实命令经运行角色执行，验证 `river_job` 的 kind/args、提交留任务、回滚不留任务；少登记一个 worker 的负例必须失败。不能把服务端已有 NewInserter 当 CLI 已接通。
````
<!-- M4-M5-04-END -->

**M5 结束时的事实：** M5 仍没有投递任务的命令。CLI 的组合（`bootstrap/users.go`、`workspaces.go`、`commands.go`）只构造连接池与管理用例，不接事件发布者；`jobs.NewInserter` 唯一的非测试调用在 `bootstrap/app.go`（serve），由 `issueInserter` 与 `assetInserter` 包装。M5 新增的投递均在服务中：`asset.delete_scope` 由 `project.deleted`/`workspace.deleted` 的订阅者在发布者事务内投递。

**何时验收：** 首个投递任务的 CLI 提交中完成，验收同上文原文；M6、M7、M8 谁先加入这样的命令，谁完成并在另两份交接中写明。
