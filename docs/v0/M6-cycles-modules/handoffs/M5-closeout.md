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

## 处理结果（M6/P1）：M5-M6-03

`planning` 的清理步骤属于它自己的 `Purger`，排在 issue 之后、project 之前，在 `files.cleanup_budget` 的同一预算内运行，单条语句有 `LIMIT` 批量，被取消的语句回滚并在下一次运行重做；`cycles` 加入 `purgeScaleTables`，`TestPurgeScale`（`NERVE_PURGE_SCALE=1`）认得它的两条分页语句、一次运行完成。`planning` 放到 project 之后的变异让 `TestPurgeDeletedRunsWithRuntimeRole` 失败。删除迭代不影响资产。关系表在 P3 加入；收尾前在 M6 代码上重跑 `TestPurgeScale` 与 `TestTheDeletionsAtTheScaleTarget`，本交接保持 open。证据：[P1 spec](../specs/P1-cycles-backend.md) 2.4、A6。

## 处理结果（M6/P2）：M5-M6-01

P2 的模块不是资产的宿主：`modules.description` 是纯文本的 `text`（只拒 NUL，契约写明 plain text），Plane 的 `description_text`、`description_html` 已删除；模块链接只存地址、标题与抓取的元数据，抓到的图标以 `data:` 地址内嵌在元数据里，不存文件（同工作项链接）。`AssetPurpose`、`asset` 模块与 `file_assets` 都没有改动。本条的触发条件（迭代/模块第一次带文件）没有出现，本交接保持 open。证据：[P2 spec](../specs/P2-modules-backend.md) 2.2。

## 处理结果（M6/P2）：M5-M6-03

`planning` 的 `Purger` 在 `cycles` 之前加三步 `module_links → module_members → modules`（子先父），每张表单语句分页、有 `LIMIT`，在 `files.cleanup_budget` 的同一预算内运行；三张表加入 `purgeScaleTables`，P2 以 `NERVE_PURGE_SCALE=1` 跑 `TestPurgeScale`，追踪器认得三张表的分页语句，一次尝试、一次运行完成。删除模块不影响资产。关系表在 P3 加入；收尾前在 M6 代码上重跑 `TestPurgeScale` 与 `TestTheDeletionsAtTheScaleTarget`，本交接保持 open。证据：[P2 spec](../specs/P2-modules-backend.md) 2.4、A6。

## 处理结果（M6/P3a）：M5-M6-03

两张关系表 `cycle_issues`、`module_issues` 归 `issue` 模块，清理步骤属于 `issue` 的 `Purger`（`issue/module.go` 的 `Provide`）：在 `issues` 之前加 `module_issues → cycle_issues`（子先父；模块次序 asset → issue → planning 使它们先于 `cycles`、`modules`），在 `files.cleanup_budget` 的同一预算内运行。两条新语句 `PurgeModuleIssues`、`PurgeCycleIssues`（`issue/adapter/postgres/queries/purge.sql`）都是单语句分页（`WITH candidates AS MATERIALIZED … LIMIT … DELETE FROM <表>`），被预算取消的语句回滚，下一次运行从游标重做。两张表加入 `purgeScaleTables` 与 `seedPurgeSparseRows`；P3a 以 `NERVE_PURGE_SCALE=1` 跑 `TestPurgeScale`（收据 `t3-purge-scale`，exit 0）：一次尝试、一次运行完成（660,033 行，运行 4.9 秒；M5 的 P8 为 660,026 行 5.0 秒），追踪器认得两张表各 2 条分页语句，表序为 `… issue_subscribers → draft_issues → module_issues → cycle_issues → issues → module_links …`，过期范围各 1 行清掉，59 天前删除的与未删除的各 1 行留下。清理把 `issues` 放到两张表之前、只把 `cycle_issues` 放到 `issues` 之后的变异（`t3-mut-purge-issues-first`、`t3-mut-purge-cycle-after-issues`）让 `TestIssuePurgeAllTables` 的子先父核对失败。删除迭代、模块不影响资产：订阅者只软删归属、写动态、记工作项的最后修改。收尾前在 M6 代码上重跑 `TestPurgeScale` 与 `TestTheDeletionsAtTheScaleTarget`，本交接保持 open。证据：提交 `f9e62906607ba16f2b0a7ee21ced6db539ce03c8`；[P3a spec](../specs/P3a-issue-membership.md) 2.5、A3。

## 处理结果（M6/P3b）：M5-M6-02

P3b 让附件列随共用的列表设置出现在迭代页、模块页：两页的偏好（`cycle_user_properties`、`module_user_properties`，`GET/PATCH /api/v0/me/cycles|modules/{id}/preferences`）的显示属性就是三页共用的 14 个键（`common.yaml` 的 `IssueListDisplayProperties`），`attachment_count` 在其中、默认 `true`（迁移 `00046`、`00047` 的默认值与 CHECK，`TestPlanningPreferencesSchema`；接口 `TestPlanningPreferencesAPI`；e2e `g15-planning-preferences.spec.ts` 读到的默认值）。迭代页、模块页的列表就是项目列表加 P3b 的 `cycle_id`/`module_id` 筛选（M6 设计 §7），不另开列表接口：行的 `attachment_count` 仍经 M5 的 `AttachmentCounter` 填，`±attachment_count` 的目录仍经 `AttachmentCountCatalog` 取，与迭代名、模块名的目录在 `issue/app/listing.go` 的同一处按同一批项目取；筛选只加条件，不另算计数。M5 的 `TestListAttachmentCount`、`TestListAttachmentCountPreferences` 在 P3b 的回归中照常通过；没有另加“按迭代筛选并按附件数排序”的组合测试。用真实附件核计数、排序、分页与换会话在迭代页、模块页的列表开放时（P5、P6）。本交接保持 open。证据：提交 `acc1f8a9fd0ae43513d5b81262edd744015c7323`（共用的显示设置）、`e1ce8ea24bc4467919f4012ff1c7d2400bb88ee6`（按需目录并为一处）、`e239a557471a048f56453f94676f2ecec22bf238`（两张表）、`79a33c7c83c078ac4664eb0379c0decca4795226`（接口）；[P3b spec](../specs/P3b-list-engine.md) 2.1、2.4、R5。

## 处理结果（M6/P3b）：M5-M6-03

两张新表 `cycle_user_properties`、`module_user_properties` 归 `planning`，清理步骤属于 `planning` 的 `Purger`：`module_user_properties` 在 `module_links` 之前、`cycle_user_properties` 在 `cycles` 之前（子先父，次序为 `module_user_properties → module_links → module_members → modules → cycle_user_properties → cycles`），在 `files.cleanup_budget` 的同一预算内运行。两条新语句 `PurgeCycleUserProperties`、`PurgeModuleUserProperties`（`planning/adapter/postgres/queries/purge.sql`）都是单语句分页（`WITH candidates AS MATERIALIZED … LIMIT … DELETE FROM <表>`），被预算取消的语句回滚，下一次运行从游标重做。两张表加入 `purgeScaleTables` 与 `seedPurgeSparseRows`；P3b 以 `NERVE_PURGE_SCALE=1` 跑 `TestPurgeScale`（收据 `t8-purge-scale`，exit 0，33 秒）：一次尝试、一次运行完成（运行 4.9 秒），旧范围的各表由自己的语句清掉，59 天前删除的与未删除的行不变。把 `cycles` 放到 `cycle_user_properties` 之前的变异（`t8-mut-purge-cycles-first`）让 `TestModulePurgeChildrenFirst` 与 `TestPurgeCycles` 的表序核对失败。删除迭代、模块及其偏好行不影响资产。收尾前在 M6 代码上重跑 `TestPurgeScale` 与 `TestTheDeletionsAtTheScaleTarget`，本交接保持 open。证据：提交 `e239a557471a048f56453f94676f2ecec22bf238`；[P3b spec](../specs/P3b-list-engine.md) 2.5、A6。

## 处理结果（M6/P5b）：M5-M6-02（迭代）

迭代页的工作项列表在 P5b 开放，附件列随共用的列表设置出现，网页不另算计数：迭代页偏好的 `display_properties` 就是三页共用的 14 个键（`attachment_count` 在其中、默认 `true`），行的 `attachment_count` 与 `±attachment_count` 的排序、续页都由 nerve 给出，网页只按回答显示（迭代页的列表是项目列表加 `cycle_id`，[P5b spec](../specs/P5b-cycle-list.md) R1）。“何时验收”由 G15 页面故事二（`e2e/stories/planning/g15-cycle-list-page.spec.ts`，提交 `f4a3700732eeda348ebf160b6aa8062fb168b1f1`）以真实附件核对：

- **计数**：迭代里 51 个工作项，其中两个各有真实上传的附件（1 个、3 个），列表行显示“Attachments1”“Attachments3”，表格的附件列显示“1 attachment”“3 attachments”，没有附件的显示“0 attachments”。
- **排序与分页**：接口按 `-attachment_count`、`attachment_count` 两个方向续页，每项一次、最后一页没有游标（`expectPages`）；页面以列表布局按状态分组（每组 50），两个方向各先列出 50 项、滚到底以组的游标续页，每项一次、之后没有“Load more”，续页的查询带 `cycle_id`、`group_key`、`order_by`；表格的表头两个方向排序（`aria-sort`）。
- **换会话与增删**：经接口给一个工作项加一个附件、给另一个删一个，另一个标签页重新登录同一个人，页面跟随新会话（换用了新令牌）读到 nerve 现在的数（2、2）。
- 每一段以整页安静的断言收尾。e2e 变异“附件列恒为 0”让故事失败。表格自身的续页（每页 100，要 101 项）不在故事里，续页由列表布局核对；附件部件本身的增删由 M5 F8 覆盖。

模块页的附件列在 P6。本交接保持 open。证据：提交 `f4a3700732eeda348ebf160b6aa8062fb168b1f1`；[P5b spec](../specs/P5b-cycle-list.md) §2.10。
