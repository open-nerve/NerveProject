# M4/P2 访客规则、创建者、删除、按编号取、归档、起止日期：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M4/P2 `issue-access` |
| 日期 | 2026-10-19；修订 2026-10-20（预检） |
| 状态 | 已裁定：第 3 节第 1–19 条由控制者照建议裁定（2026-10-19；第 3、8、10、14 条带条件，各条写明）；预检（2026-10-20）的发现 M1、L1–L12 按控制者的裁定改进本 spec、plan 和设计（附录 A.2 的"预检之后的修改"）。原型、清扫和逐 Task 复现见附录 A。设计 12 节 P2 的十四个草稿任务在约 16 个、每个约 1,500 行以内重新切成 14 个 Task（第 3 节第 1 条），没有超出，没有停下 |
| 上级文档 | [M4 设计文档](../M4-design.md) 第 2（I1、I2、I15 的"P2"一段）、3.3（`project` 的 `Lock` 回答的字段）、3.4（访客规则、`OwnOnly`、`AllowCreator`、`guest_view_all_features`，规则表中 P2 的行）、3.5（取锁的一步；"修改工作项""删除工作项""归档、恢复"三行；交错 5、12；约定五）、3.6（删除工作项的连带）、3.7（起止日期、按编号取）、3.8（归档、恢复、已归档只读）、3.13（动态的第 8、9、14、16 行）、3.23 与 4.17（P2 的行）、5.1–5.3（`deleteIssue`、`getIssueByIdentifier`、`archiveIssue`、`unarchiveIssue`，三个模式的日期）、6.1（`bodyshape`、`shared.Date`）、6.5、6.7、9.1–9.3（P2 的部分）、12（P2 与约束 1–8）、13.1 节 |
| 前置交接 | [P1a spec](P1a-issues.md) 第 5 节和 [P1a 评审](../reviews/P1a-issues-review.md)第 6 节的 P2 一行；[P1b spec](P1b-issue-update.md) 第 5 节的 P2 一行和收尾一行的最后一句；[P1b 评审](../reviews/P1b-issue-update-review.md)第 6 节；[M1-P3 交接](../handoffs/M1-P3-trim-platform.md)的地址一项；[M2 收尾交接](../handoffs/M2-closeout.md)第 3、4 节；[M3 收尾交接](../handoffs/M3-closeout.md)第 2、3、4 节。落点见第 3 节第 20 条、第 7 节 |
| 计划 | [P2 plan](../plans/P2-issue-access.md) |

本 spec 只写 M4 设计交给 P2 决定的东西：名字、签名、SQL、测试名，以及原型证明了什么。规则本身以 M4 设计为准，这里引用节号，不重述。P2 从 `7db78788`（P1b 的合并提交）开始。

P2 是 M4 权限的一段（设计 12 节约束 3：权限的扩展在这一段，起止日期和 `bodyshape` 的生成器也在，与锁的约定分开）。访客规则在每个指名工作项的读和写上答 404，先于角色的 403，预检和锁下的判定各执行一次；`AllowCreator` 让项目的有效成员删除自己建的工作项，锁下再判定；删除、归档、恢复都走 P1b 的同一条取锁路径（读工作项 → 预检 → 闸门 → 项目 N → 锁下重读 → 判定），不另写第二条（闸门一步由单元的 `TestEachWriteOnAWorkItemEntersItsProjectsGate` 和组合的 `TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy` 钉住）；起止日期从 `shared.Date` 经 `bodyshape`、生成器、HTTP、领域到存储，修改按合并之后的值检查。附录 A 的清扫逐个核对每个这样的测试在它所说的性质去掉之后失败，并记下失败在哪一层（单元、存储、组合、端到端、守卫）；只在单元一层失败的安全或加锁的性质算缺口，按性质只能在单元一层的逐条写在附录 A.3。

## 1. 目标

按 M4 设计 12 节 P2：

- **访客规则和创建者**（3.4）：`shared.Target.CreatorID`、`shared.Grant.OwnOnly`（按项目的角色：在项目里只是访客的工作区管理员同样为真）；`access` 读项目的 `guest_view_all_features`，规则表的 `AllowCreator`（创建者仍是项目的有效成员、项目角色不低于成员时允许）；`issue.read` 加访客，`issue.delete`（管理员 + `AllowCreator`），`issue.archive`（管理员、成员），`issue.search`（工作区的三个角色：按编号取的第一步，第 3 节第 4 条）。`issue/app` 的 `Locks.decide` 在预检和锁下的判定都执行访客规则：看不到的工作项答 404 `issue.not_found`，先于角色的 403；`getIssue`、`updateIssue` 随之。
- **`deleteIssue`**（`DELETE /api/v0/issues/{issue_id}`，3.5、3.6）：取锁的一步照 P1b；锁下项目已归档 409；然后同一时刻、同一个删除者软删除工作项和 P1a 的四张子表（负责人、标签、关注、动态）里它的行，按全局的表序，每张表一条语句；子树在 P3。已归档的工作项照样删除（3.8）。
- **`getIssueByIdentifier`**（`GET /api/v0/workspaces/{slug}/issues/{identifier}`，3.7）：工作区（不是有效成员 404 `workspace.not_found`）→ 解析 → 折叠比较（`foldIdentifier`，样例 `api/testdata/identifier-fold.json`）→ 编号 → 与 `getIssue` 同一个读（`see`）；格式不对、没有这个项目、没有这个编号、看不到项目、访客规则隐藏的，都是同一个 404 `issue.not_found`，回答逐字节相同。
- **`archiveIssue`、`unarchiveIssue`**（`POST /api/v0/issues/{issue_id}/archive`、`/unarchive`，3.8）：锁下项目已归档 409 → 已是所求的样子：不写，200 → 归档要状态组是已完成或已取消（409 `issue.not_archivable`，带文案）→ 时刻 → `archived_at`、最后修改者和时刻 → 动态第 14、16 行；回答完整的 `IssueDetail`。
- **已归档只读**（3.8）：`updateIssue` 对已归档的工作项 409 `issue.archived`（带文案），在项目已归档的 409 之后、在"什么都不改"之前；`project.state_in_use` 的说明、契约和文案加上补救（P1a、P1b 交给 P2）。
- **起止日期**（3.7、6.1）：`shared/date.go`；`bodyshape` 的 `FormatDate`，字段错误的路径按下标的数值排序，不限类型的节点检查数的范围，代价测试加字符串的数组；生成器认 `shared.Date`；五个模块的 `oapi-codegen` 把 `format: date` 映射到它；`IssueCreate`、`IssueUpdate`、`IssueDetail` 的 `start_date`、`target_date`；修改按合并之后的值检查（422）；动态第 8、9 行。
- **测试**：交错 12；交错 5 的访客规则下 404 的一支；删除与修改同一个工作项的两种顺序；等锁期间被降级或移出的创建者的删除；矩阵的工作项表（访客的三列、"管理员·访客"、建了它的项目成员）；整程序测试（含日期的格式）；`apitest` 两个方向；端到端 I15 的接口版本，I1、I2 的起止日期和合并之后的日期 422，I2 的已归档 409。
- **文档**：3.23 中 P2 的行（总体设计 6.5、M2 设计 3.11、差异清单第四节）；三份交接的"处理结果（M4/P2）"。

不加 Go 模块、npm 包；没有新表、新迁移（起止日期的两列和 `issues_dates_check` 在 P1a 的迁移里）。跨模块的端口：`access` 读项目的事实多一列（`guest_view_all_features`）；`project` 给 `issue` 的 `IssueProjects` 加一个读（`ProjectIdentifiers`）；`workspace` 给 `project` 的目录经 `bootstrap` 也交给 `issue`（`WorkspaceBySlug`，只回答 `id`）。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录（`server/internal/` 省略）。"Task"是 plan 中负责它的任务（plan 有完整的文件表）。本 Phase 手写的代码、测试和配置文件 140 个（新建 34 个，修改 106 个），生成物 11 个，文档 8 个（本 spec 和 plan 新建；修改的是总体设计、M2 设计、差异清单和三份交接），另有本提交对 M4 设计的修改（第 15 节的一行；裁定的 3.3、3.4 的两句和 6.5 的 `ProjectGate` 一行）。

| 路径 | 内容 | Task |
|---|---|---|
| `shared/authorize.go`；`access/domain/decide.go`、`rules.go` 及测试、`creator_test.go`（新）；`access/app/authorizer.go`、`ports.go` 及测试；`project/adapter/postgres/queries/access.sql`、`access.go` 及测试；`project/app/access.go`；`issue/domain/actions.go`；`bootstrap/ports_test.go`；`tools/keywords.json` | `Target.CreatorID`、`Grant.OwnOnly`、`Facts.Creator`、`Project.GuestViewAll`、`Rule.AllowCreator`；拒绝时也交回 `Grant`；`ProjectFacts` 读 `guest_view_all_features`；`issue.delete`、`issue.archive` 的规则行和动作；关键词守卫的 `phase` 改为 `M4/P2` | 1 |
| `access/domain/rules.go` 及测试；`issue/domain/issue.go` 及测试；`issue/app/lock.go`、`get_issue.go`、`update_issue.go`、`fakes_test.go`、`guest_rule_test.go`（新）；契约（`getIssue`、`updateIssue` 的描述，`Project.guest_view_all_features` 的说明）；`issue/adapter/http/issues_test.go`；`bootstrap/issue_interleaving_test.go`、`issue_world_test.go`、`permission_matrix_issues_test.go`、`permission_matrix_issue_updates_test.go` | `issue.read` 加访客；`Issue.HiddenFrom`；`decide` 带上工作项，先执行访客规则（404 优先）；交错 5 的访客一支 | 2 |
| `bootstrap/permission_matrix_issue_columns_test.go`（新）、`permission_matrix_seed_issues_test.go`、`permission_matrix_seeded_test.go`、`permission_matrix_targets_test.go`、`permission_matrix_columns_test.go`、`permission_matrix_coverage_test.go`、`permission_matrix_issues_test.go`、`permission_matrix_issue_updates_test.go`、`permission_matrix_test.go`、`permission_matrix_members_test.go`、`permission_matrix_invitations_test.go`、`permission_matrix_workspace_test.go` | 矩阵的工作项表：17 列、各列自己的工作项（`ownIssues`）、"可见全部"的项目（`guestViewAll`）；`getIssue`、`createIssue`、`updateIssue` 的行换到它上面；新身份在别的表里的格子 | 3 |
| `issue/app/delete_issue.go`（新）及测试、`fakes_test.go`、`guest_rule_test.go`、`update_issue_test.go`、`lock_test.go`；`issue/adapter/postgres/queries/delete_issue.sql`（新）、`delete_issue.go`（新）及测试；`issue/adapter/http/handler.go`、`issues.go`、`delete_test.go`（新）、`handler_test.go`；`issue/module.go`；契约；`bootstrap/issue_deletion_test.go`（新）、`gate_test.go`、`issue_write_locks_test.go`、`issue_writes_test.go`、`project_deletion_test.go`、`workspace_deletion_test.go`、`workspace_deletion_catalog_test.go`、`permission_matrix_issue_updates_test.go` | `deleteIssue`：用例、五条语句、接口、接线、矩阵；按目录的删除测试；取锁形状的一行；连接测试的删除；删除进写闸门（单元、组合）；在项目之前已删除的工作项（PF.3） | 4 |
| `issue/domain/identifier.go`（新）及测试、`actions.go`、`errors.go`；`api/testdata/identifier-fold.json`（新）；`access/domain/rules.go` 及测试；`issue/app/get_issue_by_identifier.go`（新）及测试、`lock.go`、`ports.go`、`get_issue_test.go`、`guest_rule_test.go`；`issue/adapter/postgres/queries/issues.sql`、`issues.go` 及测试；`project/adapter/postgres/queries/issue_ports.sql`、`issue_ports.go` 及测试、`failures_test.go`；`project/app/issue_ports.go`；`project/module.go`；`workspace/module.go`；`issue/module.go`；`issue/adapter/http/handler.go`；`bootstrap/ports.go`、`ports_test.go`、`issue_ports_test.go`（新）、`app.go`、`issue_writes_test.go` | 按编号取：解析、折叠、`issue.search`、`Locks.seeWorkspace`、用例、三个端口和两条查询、接线；拒绝的回答逐值比较（访客规则隐藏的与不存在的是同一个回答） | 5 |
| 契约（`getIssueByIdentifier`、参数 `Slug`、`IssueIdentifier`；`api/openapi.yaml`）；`issue/adapter/http/issues.go`、`by_identifier_test.go`（新）、`handler_test.go`；`issue/module.go`；`bootstrap/issue_identifier_test.go`（新）、`permission_matrix_issues_test.go`、`permission_matrix_issue_columns_test.go`、`permission_matrix_columns_test.go`、`permission_matrix_targets_test.go` | 按编号取的接口、矩阵、组合 | 6 |
| `issue/domain/issue.go`、`activity.go`、`errors.go` 及测试；`issue/app/archive_issue.go`（新）及测试、`ports.go`、`rows.go`、`update_issue.go`、`fakes_test.go`、`lock_test.go`；`issue/adapter/postgres/queries/update.sql`、`update.go` 及测试；契约（`api/openapi.yaml`）；`issue/adapter/http/handler.go`、`issues.go`、`archive_test.go`（新）；`issue/module.go`；前端的三个文案文件；`bootstrap/issue_archive_test.go`（新）、`permission_matrix_issue_archive_test.go`（新）、`gate_test.go`、`permission_matrix_test.go`、`permission_matrix_coverage_test.go`、`permission_matrix_issue_updates_test.go`、`issue_world_test.go`、`issue_write_locks_test.go`、`issue_writes_test.go`、`issue_updates_test.go` | 归档、恢复；`issue.not_archivable`；矩阵的 `prepare`；取锁形状的两行；连接测试的归档、恢复；归档、恢复进写闸门（单元、组合）；归档和修改共用的 `issueState`（`stateOf` 改成它）；归档的矩阵的检查建在 `answered` 上 | 7 |
| `issue/domain/errors.go`；`issue/app/update_issue.go` 及测试、`delete_issue_test.go`；契约（`updateIssue`；`project.yaml` 的 `deleteState`）；`issue/adapter/http/update_test.go`；`project/domain/errors.go`、`project/adapter/http/state_writes_test.go`；前端的三个文案文件；`bootstrap/permission_matrix_issue_updates_test.go`、`bootstrap/issue_cascade_test.go` | 已归档只读（409 `issue.archived`）；已归档的工作项照样删除（单元测试、矩阵）；删除标签照样从已归档的工作项上摘下它（组合测试）；`project.state_in_use` 的补救 | 8 |
| `shared/date.go`（新）及测试；`platform/httpserver/bodyshape/bodyshape.go`、`ambiguity.go`、`paths.go`（新）及测试、`bodyshape_test.go`、`cost_test.go`；`server/tools/bodyshapegen/generate.go` 及测试、`testdata` 的三个文件；五个模块的 `adapter/http/gen/oapi-codegen.yaml`；`platform/httpserver/apitest/operations.go`、`bodycases_test.go` | 日期的平台：`shared.Date`、`FormatDate`、路径的次序、数的范围、代价测试、生成器、类型映射、`apitest` 的 `date` | 9 |
| `issue/domain/dates.go`（新）及测试、`issue.go`、`patch.go`、`activity.go` 及测试；`issue/app/create_issue.go`、`update_issue.go`、`rows.go`、`dates_test.go`（新）、`create_issue_test.go`、`fakes_test.go`；`issue/adapter/postgres/queries/issues.sql`、`update.sql`、`issues.go`、`update.go` 及测试；`server/sqlc.yaml` | 起止日期：领域、用例、存储；动态第 8、9 行 | 10 |
| 契约（三个模式的 `start_date`、`target_date`；`createIssue`、`updateIssue` 的描述）；`issue/adapter/http/issues.go`、`issues_test.go`、`update_test.go`、`dates_test.go`（新）；`issue/app/update_issue_test.go`；`platform/httpserver/bodyshape/bodyshape.go`；`bootstrap/issue_dates_test.go`（新）、`issue_updates_test.go`、`permission_matrix_issues_test.go`、`permission_matrix_issue_updates_test.go` | 起止日期：接口、组合、矩阵；用例的单元测试里已归档先于日期的次序、原样给出的日期 | 11 |
| `bootstrap/issue_interleaving_archive_test.go`（新）、`issue_interleaving_deletion_test.go`、`issue_interleaving_test.go` | 交错 12；删除与修改同一个工作项；等锁期间被降级、移出的创建者 | 12 |
| `e2e/fixtures/issues.ts`、`assert/issue.ts`、`assert/catalog.ts`；`e2e/stories/issue/i15-archive-issue.spec.ts`（新）、`i1-create-issue.spec.ts`、`i2-update-issue.spec.ts` | I15 的接口版本；I1、I2 的日期；I2 的已归档 409；`expectIssueDeleted`，它读的外键图（`keysBelow`） | 13 |
| `docs/v0/v0-design.md`、`M2-auth/M2-design.md`、`plane-diff.md`；三份交接 | 3.23 中 P2 的行；交接的处理结果 | 14 |

生成物：`project/adapter/postgres/gen/access.sql.go`（Task 1）、`issue_ports.sql.go`（Task 5）；`issue/adapter/postgres/gen/delete_issue.sql.go`（Task 4）、`issues.sql.go`（Task 5、10）、`update.sql.go`（Task 7、10）、`models.go`（Task 10）；`api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`（Task 2、4、6、7、8、11）；`project/adapter/http/gen/server.gen.go`（Task 2）；`issue/adapter/http/gen/server.gen.go`（Task 4、6、7、11）、`bodyshape.gen.go`（Task 11）。

### 2.2 依赖

不加 Go 模块、npm 包；`shared/date.go` 只用标准库（`cmp`、`fmt`、`time`）。没有模块之间的导入（`archtest`），`issue` 的 sqlc 只读写自己的表（`TestSQLCSchemaScope`）。跨模块的端口：

- `access/app` 的 `ProjectFacts` 多回答 `GuestViewAll`（`project` 的 `access.sql` 多读 `guest_view_all_features`，`bootstrap` 转换）。
- `issue/app` 新声明 `WorkspaceDirectory`（`WorkspaceBySlug(ctx, slug) (id uuid.UUID, found bool, err error)`）、`ProjectIdentifiers`（`Identifiers(ctx, workspaceID) ([]ProjectIdentifier, error)`）、`IssueNumbers`（`IssueByNumber(ctx, projectID, number) (id, found, err)`）、`IssueRowsDeleter`、`IssueArchiver`。前两个由 `bootstrap/ports.go` 接成：`issueWorkspaces`（`workspace` 交给 `project` 的同一个 `WorkspaceDirectory`，只取 `id`）和 `issueProjects.Identifiers`（`project` 的 `IssueProjects.ProjectIdentifiers`，`[]IssueProjectIdentifier` 换成 `[]issue.ProjectIdentifier`）；后三个是 `issue` 自己的存储。
- `workspace/module.go` 的 `WorkspaceDirectory` 的说明写上 `issue` 也用它（只改说明）。
- 五个模块的 `oapi-codegen.yaml` 把 `format: date` 映射到 `shared.Date`（今天只有 `issue` 的契约有日期，第 3 节第 10 条）；`server/sqlc.yaml` 的 `issue` 把 `date` 映射到 `time.Time`（可为空的是指针），适配器换成 `shared.Date`。

### 2.3 规则的扩展（Task 1；3.4）

- `shared/authorize.go`：`Target.CreatorID`（目标工作项的创建者；别的动作和创建者不明的工作项是零值）；`Grant.OwnOnly`（项目角色是访客、项目的 `guest_view_all_features` 为假时为真，按项目的角色，不按工作区的）。`Authorizer` 的说明：角色不允许时回答 `shared.Forbidden` **和**这次判定读到的 `Grant`，用例因此能先答访客规则的 404（第 3 节第 2 条）。
- `access/domain/decide.go`：`Project.GuestViewAll`、`Facts.Creator`；`decideProject` 先算出 `Grant`（含 `OwnOnly`），再判定角色，拒绝时交回它；`decideWorkspace` 同样。`creatorRoles`（管理员、成员，按集合）：规则带 `AllowCreator`、调用者是创建者、项目的有效成员、项目角色在 `creatorRoles` 里时允许。
- `access/domain/rules.go`：`Rule.AllowCreator`；`issue.delete`（项目级，管理员，`AllowCreator`）、`issue.archive`（项目级，管理员、成员）。`issue/domain/actions.go`：`ActionDelete`、`ActionArchive`。
- `access/app/authorizer.go`：`Creator: t.CreatorID == actor.UserID`，`GuestViewAll` 从 `ProjectFacts` 来。
- 测试：`TestEveryRuleDecidesItsCells`（规则表的每一行逐格，对没建和建了目标工作项的调用者各一遍：带 `AllowCreator` 的行用它的 `creatorCells`，别的行两遍相同）；`TestAllowCreatorAllowsAMemberWhoCreatedIt`（成员、管理员的创建者允许；作为访客不论项目的标志、成员关系结束之后、三个之外的项目角色都不允许；没建的调用者只看角色；不带 `AllowCreator` 的规则不看创建者）；`TestARefusalCarriesTheGrant`；`TestTheGrantCarriesTheRoles`（`OwnOnly` 按项目角色，不论工作区的角色）；`TestAuthorizeKnowsTheCreator`、`TestAuthorizeReadsTheTargetsProject`；存储的 `TestProjectFacts`（读出标志）；`bootstrap` 的 `TestAccessProjectsConvertsProjectsAnswer`。

### 2.4 访客规则（Task 2；3.4、3.5、6.7）

- `access/domain/rules.go`：`issue.read` 加项目的访客。
- `issue/domain/issue.go`：`Issue.HiddenFrom(actor, ownOnly)`：`ownOnly` 的调用者看不到别人建的工作项，也看不到创建者不明的（`created_by_id` 为空：账户被物理删除之后，P9）。
- `issue/app/lock.go`：`decide(ctx, actor, action, workspaceID, projectID, i *domain.Issue, notFound)`：`i` 不为空时 `Target.CreatorID` 取它的创建者；`ErrNotVisible` 答 `notFound`；`i.HiddenFrom(actor.UserID, grant.OwnOnly)` 也答 `notFound`，在角色的 403 之前（`Authorize` 拒绝时也交回 `Grant`，第 3 节第 2 条）；其余照 `Authorize` 的回答。`see`（读、预检）传读到的工作项，`lockAndDecide` 传锁下重读的工作项；按项目的写（创建）传 `nil`。
- `getIssue`、`updateIssue` 的说明和契约的描述写上访客规则；`Project.guest_view_all_features` 的说明（设计 12 节 P2 的第 2 个任务）。
- 测试：领域的 `TestTheGuestRuleHidesAnothersWorkItem`；单元的 `TestADecisionOnAWorkItemNamesItsCreator`（预检和锁下各把创建者交给判定，按项目的判定不指名谁）、`TestTheGuestRuleHidesAnothersWorkItemFromAReader`（自己的读得到，别人的 404，角色被拒绝时也先答 404；项目"可见全部"时读得到别人的；创建者不明的工作项也是 404：假存储把零值的创建者读成没有，`creatorOf`）、`TestTheGuestRuleHidesAnothersWorkItemFromAWriter`（预检答 404，在闸门和锁之前，角色被拒绝与否；等锁期间成了这样的访客的，锁下答 404，不写；对自己的，角色的 403）；HTTP 的 `TestGetIssueRefusals`；交错 5 的访客一支（`TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`：bob 改 alice 的 Fix 时被降为"只看自己"的项目里的访客，锁下的判定答 404 `issue.not_found`，改他自己的工作项答 403）。

### 2.5 矩阵的工作项表（Task 3；9.2）

`permission_matrix_issue_columns_test.go`（新）：`issueColumns` 17 列，照 9.2 的次序：PA、PM、建了它的项目成员（`callerCreator`）、访客·全部（`callerGuestAll`）、访客·自己（`callerGuestOwn`）、访客·别人（`callerGuestOthers`）、管理员·访客（`callerAdminGuest`，对别人的）和它对自己的（`callerAdminGuestOwn`）、PM+WA、WA-、WM-公、WM-私、WG-、P-前、X 三列。种子（`permission_matrix_seed_issues_test.go`）：acme 的私有项目"可见全部"（`guestViewAll`，由它的管理员 PA 经项目的存储设），公开项目照默认（`guest_view_all_features` 为假），所以访客的三种情形是同一个账户；每个项目的 `matrixIssue` 之后，`ownIssues` 经存储写下，创建者是各列的账户：PM 的；工作区访客的（访客不能创建，`createIssue`，所以访客自己的工作项是他从前作为成员建的；种子经存储把它记在他名下：判定只读创建者和他现在的角色，不读它怎样来的）；在项目里是访客的工作区管理员作为管理员建的；已离开私有项目的成员离开之前建的（9.2"建了它、已离开项目"）。`seededIssues` 读回每个工作项的编号、名称、创建者，以及工作项表自己的每一列靠的事实（列的账户在它的项目里的角色、标志，工作区管理员的角色）。`onAWorkItem(admin, member, creator, guest)`、`onTheIssues(admin, member, guest)` 给出一行的格子；每格指名它的列自己的工作项（`issueOf`、`toWorkItem`），`matrixViolations` 核对（`TestMatrixViolationsCatchesAnotherCasesWorkItem`）。`getIssue`、`createIssue`、`updateIssue` 的行换到这张表上。9.2 的"创建者的两行"（"建了它、后来被降为访客"403，"建了它、已离开项目"404）是这张表的列：访客·自己（种子记在他名下的）和 P-前（离开之前建的）的 `deleteIssue` 格子（第 3 节第 15 条）。

### 2.6 `deleteIssue`（Task 4；3.5、3.6、5.1）

- `issue/app/delete_issue.go`（新）：`IssueDeletion{IssueID, By, At}`；`IssueRowsDeleter`（`DeleteIssueRow`、`DeleteIssueAssigneeRows`、`DeleteIssueLabelRows`、`DeleteIssueSubscriberRows`、`DeleteIssueActivityRows`）；`issueRowSteps`：全局的表序（工作项 → 负责人 → 标签 → 关注 → 动态，与删除项目的 `rowSteps` 同一个次序）；`DeleteIssueDeps{Locks, Rows, Clock}`、`NewDeleteIssue`、`Execute(ctx, id) error`：`Locks.run` 的 `write{issue: &id, action: issue.delete}`；锁下项目已归档 409 `project.archived`；`Clock.Now()`；逐张表一条语句，第一个失败结束它。不写动态（3.6：删除不写"已删除"的动态，差异已登记）。
- `queries/delete_issue.sql`（新）：五条 `UPDATE … SET deleted_at = $at, updated_at = $at, updated_by_id = $by WHERE issue_id = $id AND deleted_at IS NULL`（工作项的一条按 `id`）；已删除的行保留它的删除。约定五：第 3 节第 7 条。
- 契约：`DELETE /api/v0/issues/{issue_id}`，204，`x-problem-codes: [issue.not_found, forbidden, project.archived, server_busy]`；HTTP 的 `DeleteIssue`；`issue.New` 的 `DeleteIssue`。
- 测试：单元的 `TestDeleteIssueDeletesTheWorkItemAndItsRows`（3.5 的次序：`Locks` 的调用、`issue.delete` 的判定、时刻、逐张表、都是 alice 在那一刻的）、`TestDeleteIssueRefusals`（锁之前和锁下：不在、看不到项目 404，角色 403，已归档的项目锁下 409，都不删一行）、`TestDeleteIssueStopsAtAFailingPort`；`guest_rule_test.go` 的 `TestADecisionOnAWorkItemNamesItsCreator` 带上删除（访客规则的两个表经修改核对 `Locks.run`，删除走同一条路），也带上创建者不明的工作项的删除（两次判定都不指名创建者：`AllowCreator` 不把调用者当作它的创建者）；`lock_test.go` 的 `TestEachWriteOnAWorkItemEntersItsProjectsGate` 的删除一行（预检之后、任何事务之外进项目的写闸门，事务结束之后最后离开）；存储的 `TestTheStatementsOfADeletedWorkItemDeleteItsRows`（每条语句只删这个工作项未删除的行：一小时前已删除的行、另一个已删除的工作项的行保留原来的删除，别的工作项的行和每一列都不动）；HTTP 的 `TestDeleteIssue`、`TestDeleteIssueRefusals`；组合的 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`（按目录：`keysTo(issues)` 的每个外键下都有一行，删除之后没有未删除的；`subItems`（`issues.parent_id`）豁免到 P3，第 3 节第 14 条）；`project_deletion_test.go` 的种子加一个在项目之前已删除的工作项，项目删除之后它的行照旧（PF.3）；取锁形状的 `deleteIssue` 一行（第 3 节第 13 条）；`TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection` 在一个连接的池上也删除一个工作项（删除的每条语句在它的事务的连接上，附录 A.2）；组合的 `TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy` 的删除一行（项目的一个位置被占、不排队时，删除立刻答 503 `server_busy`、`Retry-After: 1`）；矩阵的 `deleteIssue` 行。

### 2.7 按编号取（Task 5、6；3.7、5.1）

- `issue/domain/identifier.go`（新）：`ParseIdentifier(s) (project, number, ok)`（最后一个连字符；项目的标识不空；编号是不带符号、不以 0 开头的十进制，1 到 2147483647）；`foldIdentifier`（`a`–`z` 成 `A`–`Z`，`çşğöü` 成 `ÇŞĞÖÜ`，`i`、`ı`、`I`、`İ` 成 `I`）；`MatchIdentifier(asked, identifiers) (index, ok)`：折叠之后相同的只有一个时取它；两个以上时取逐字等于 `strings.ToUpper(asked)` 的那个，没有就没有。`api/testdata/identifier-fold.json`（`issue-order.json` 的写法）的样例由 `TestMatchIdentifierFindsWhatTheSamplesSay` 读。
- `access`：`issue.search`（工作区级，管理员、成员、访客，3.4 的行）；`issue/domain` 的 `ActionSearch`、`ErrWorkspaceNotFound`（404 `workspace.not_found`，`workspace` 模块的码，操作声明它，第 3 节第 4 条）。
- `issue/app`：`Locks.seeWorkspace(ctx, actor, action, workspaceID)`（`ErrNotVisible` 答 `workspace.not_found`）；`GetIssueByIdentifier{Locks, Workspaces, Projects, Issues}`、`Execute(ctx, slug, identifier)`：`WorkspaceBySlug` → `seeWorkspace(issue.search)` → `find`（`ParseIdentifier` → `Identifiers` → `MatchIdentifier` → `IssueByNumber`，任何一步找不到都是 `issue.not_found`）→ `Locks.see(issue.read)`。不取锁、不开事务、不进闸门。
- 查询：`project` 的 `ProjectIdentifiers`（工作区未删除的项目，归档的也在，`id` 的次序；已删除的项目的标识可能已是别人的，M3 设计 3.19）；`issue` 的 `IssueByNumber`（项目的、未删除的、归档的也在）。
- 契约（Task 6）：`GET /api/v0/workspaces/{slug}/issues/{identifier}`，`x-problem-codes: [workspace.not_found, issue.not_found, forbidden]`，参数 `Slug`、`IssueIdentifier`；HTTP 的 `GetIssueByIdentifier`。
- 测试：领域的 `TestParseIdentifier`、`TestMatchIdentifierFindsWhatTheSamplesSay`；单元的 `TestGetIssueByIdentifierReadsTheWorkItem`、`TestGetIssueByIdentifierRefusals`；存储的 `TestIssueByNumber`、`TestProjectIdentifiers`；`bootstrap` 的转换 `TestIssueProjectsConvertsProjectIdentifiers`、`TestIssueWorkspacesConvertsWorkspacesAnswer`；HTTP 的 `TestGetIssueByIdentifier`、`TestGetIssueByIdentifierRefusals`；组合的 `TestAWorkItemIsFoundByItsIdentifier`（IŞ 和 İŞ、İI 和 Iİ 两对折叠相同的项目、ŞI 和已删除的 Şİ、Sec（私有）、已删除的和在 Done 里已归档的工作项、另一个工作区 beta 里同一个标识的项目和它唯一带点的 İŞ；任何大小写、土耳其字母；两对各取逐字等于大写的那个，大写不是其中任何一个的（`ii-1`）404；已删除的项目不算：`Şİ-1` 找到的是 ŞI 的（已删除的 Şİ 与 ŞI 折叠相同，又正是所问的大写）；比较的是折叠：beta 的 `iş-1` 找到 İŞ 的；每种 404 的回答逐字节相同；wanda 403，olga 和不存在的 slug `workspace.not_found`；附录 A.2 加了 beta、İI 和 Iİ，已归档的工作项从 Backlog 移到 Done，预检之后加了 ŞI、Şİ 和 beta 的 İŞ）；矩阵的 `getIssueByIdentifier` 行（`byIdentifier`、`inItsWorkspace`，`TestMatrixViolationsCatchesAnotherCasesIdentifier`）。

### 2.8 归档、恢复（Task 7；3.5、3.8、3.13）

- `issue/domain`：`GroupCancelled`、`Archivable(group)`（已完成、已取消）；`ErrNotArchivable`（409 `issue.not_archivable`）；`ArchiveActivity()`（第 14 行：`archived_at`、`updated`，`NULL` → `manual_archive`）、`UnarchiveActivity()`（第 16 行：`archive` → `restore`）。
- `issue/app/archive_issue.go`（新）：`ArchiveIssueDeps{Locks, Issues IssueArchiver, States StateCatalog, Clock}`；`ArchiveIssue`、`UnarchiveIssue`（`NewArchiveIssue`、`NewUnarchiveIssue`），共用 `archive(ctx, id, on)`：`Locks.run` 的 `write{issue: &id, action: issue.archive}` → 项目已归档 409 → 已是所求的样子：回答锁下读到的工作项，不写 → 归档时 `archivable`（锁下读项目的状态，由 `issueState` 取出工作项的状态：工作项的状态不是项目的是错误；`issueState` 与 `updateIssue` 的 `check` 共用，`update_issue.go` 的 `stateOf` 改成它）→ 时刻 → `SetArchivedAt`（`archived_at`、`updated_at`、`updated_by_id`）→ 一行动态 → 在事务里读回。`IssueArchiver`（嵌入 `IssueReader`；`SetArchivedAt`、`InsertActivities`），`IssueArchival{ID, ArchivedAt, By, At}`。
- `queries/update.sql`：`SetArchivedAt`（按 `id`，写 `archived_at`、`updated_at`、`updated_by_id`）。
- 契约：`POST …/archive`（`x-problem-codes: [issue.not_found, forbidden, project.archived, issue.not_archivable, server_busy]`）、`POST …/unarchive`（同上，没有 `issue.not_archivable`），回答 `IssueDetail`。前端：`error-messages.ts` 和两份 `errors.json` 的 `issue_not_archivable`（约束 4；措辞第 3 节第 9 条）。
- 矩阵：`permission_matrix_issue_archive_test.go`（新）的九行（`archiveIssue`：照常、Todo 里的、已归档的、已归档的项目、已归档的项目里已归档的；`unarchiveIssue`：照常、没归档的、已归档的项目、已归档的项目里已归档的），格子的 `prepare` 在格子自己的副本上把它的列的工作项移到 Done（和归档）；`matrixRow.prepare`（新的设施：在写的格子的副本上、它的 app 启动之前运行；没有 `write` 的行带 `prepare` 是违规，`TestMatrixViolationsCatchesEachGap`；第 3 节第 15 条）。
- 测试：领域的 `TestArchivable`、`TestArchiveActivity`；单元的 `TestArchiveIssueArchivesTheWorkItem`、`TestUnarchiveIssueUnarchivesTheWorkItem`、`TestArchivingAsItIsWritesNothing`、`TestArchiveIssueRefusals`、`TestArchivingAWorkItemWhoseStateIsNotItsProjectsFails`（锁下读到的状态里没有工作项的状态：是错误，不是 409，读完就不再写；附录 A.2）；存储的 `TestArchivingAWorkItemWritesItsRow`、`TestAFailedUpdateStatementIsAnError`；HTTP 的 `TestArchiveAndUnarchiveIssue`、`TestArchiveIssueRefusals`；组合的 `TestArchivingAWorkItemStampsItsRows`（alice 在 Done 里建工作项，bob 归档、再归档，alice 恢复、再恢复：第一次的归档、恢复各是做的人在请求之内的一个时刻，工作项和它的一行动态同一时刻；第二次什么都不写；一开始就建好的 Todo 里的工作项 409 `issue.not_archivable`，它自己被拒绝的归档和另一个工作项的归档、恢复都不写它的任何一行，移到 Cancelled 之后 bob 归档它（附录 A.2）；Cancelled 改到"进行中"组之后（另一个已取消的状态 Dropped 留住那一组），bob 再归档它照样回答、什么都不写：已归档的工作项先于状态组回答（执行中由评审加的），alice 恢复它：恢复不看状态组（预检之后加的）；`issueWorld.doneWorkItem`）、`TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection` 在一个连接的池上也归档、恢复（附录 A.2）；取锁形状的两行（第 3 节第 13 条）；单元的 `TestEachWriteOnAWorkItemEntersItsProjectsGate` 和组合的 `TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy` 的归档、恢复两行（预检之后加的）。

### 2.9 已归档只读；`state_in_use` 的补救（Task 8；3.5、3.8）

- `issue/domain`：`ErrIssueArchived`（409 `issue.archived`）。`UpdateIssue` 锁下：项目已归档 409 → 工作项已归档 409 `issue.archived`（什么都不改的修改也是）→ `ChangesOf`。契约：`updateIssue` 声明 `issue.archived`。前端：`issue_archived` 的文案。删除标签照样从已归档的工作项上摘下它（`DetachLabel`，项目一级的写，不是对工作项的写；留下一个已删除的标签没有意义；Plane 的级联相同），由组合的 `TestDeletingALabelTakesItOffTheWorkItems` 钉住：alice 删除 Bug 之前，带 Bug 的工作项已归档，它的标签行照样删除（变异 T8.9，执行中由评审发现）。
- `project.state_in_use`：Go 的说明、`project.yaml` 的 `deleteState`、两份文案加上补救（"先恢复已归档的，再把它们都移到别的状态"），P1b 的修改和本 Phase 的恢复都有了才做得到（P1a 评审第 6 节）；`state_writes_test.go` 的 `stateInUseJSON`。措辞第 3 节第 9 条。
- 测试：单元的 `TestUpdateIssueRefusesUnderTheLock`（已归档的项目里已归档的工作项答项目的 409；已归档的工作项什么都不改也 409）、`TestDeleteIssueDeletesTheWorkItemAndItsRows` 的 `an archived work item` 一例（已归档的工作项照样删除：同样的调用和删除）；HTTP 的 `TestUpdateIssueRefusals`；矩阵的 `updateIssue` 的"已归档的项目里已归档的工作项"（`archived, its work item archived`：项目的 409 在先）和"已归档的工作项"（`archived work item`：判定之后 409 `issue.archived`）两行，体都是什么都不改的 `{}`；`deleteIssue` 的同样两行（项目的 409 在先；已归档的工作项照样删除，变异 T8.8，执行中由实现者发现）；组合的 `TestDeletingALabelTakesItOffTheWorkItems`（见上，变异 T8.9）。

### 2.10 日期的平台（Task 9；6.1，M2 交接第 4 节）

- `shared/date.go`（新）：`Date{year, month, day}`（零值不是任何一天）、`ParseDate(s)`（`time.DateOnly`，年份从 0001 起；别的写法、带时间或时区、前后的空白都是错误）、`DateOf(t)`、`Time()`（UTC 的零点）、`String()`、`Compare(e)`、`MarshalText`、`UnmarshalText`（JSON 是字符串）。测试 `TestParseDate`、`TestDateCompare`、`TestDateAndTime`、`TestDateInJSON`。
- `bodyshape`：`FormatDate`，`checkDate` 照 `shared.ParseDate` 的规则读（平台不能导入 `shared`，`archtest`；两者一致由 `issue` 的 HTTP 适配器的 `TestDatesPassTheBodyCheckAsTheyDecode` 钉住，Task 11）；问题按路径排序时下标按数值（`comparePaths`：`tags[2]` 在 `tags[10]` 之前，全序）；`problems` 和路径的代码移进 `paths.go`（新，`bodyshape.go` 因此不过约 400 行，第 3 节第 12 条）；不限类型的节点（`Any` 的值、开放对象的未声明成员、没有 `items` 的数组的元素）里的数由 `numbers` 一遍读完，放不进 `float64` 的报 `invalid_format`。测试：`TestFormatDate`、`TestComparePaths`、`TestCheck`（`ids[2]` 在 `ids[10]` 之前，开放节点的数）、`TestCheckCostsAboutTheBody`（加字符串的数组和开放节点的数）。
- 生成器：`checkers` 加 `shared.Date` → `FormatDate`；`testdata` 加日期的字段（`things.yaml`、`oapi-codegen.yaml`、`things.golden`）；`TestGenerateFollowsTheTypeMapping`（映射之外的 `date` 仍被拒绝）、`TestGenerateAcceptsTheStringsItCanCheck`、`TestGenerateRejectsWhatItCannotCheck`。
- 五个模块的 `oapi-codegen.yaml`：`format: date` → `shared.Date`（第 3 节第 10 条）。
- `apitest`：`date` 进 `checkedFormat`、`validValue`；`TestBodyCasesOfDates`（`date` 是检查的格式：合法的体带一个合法的日期，别的用例都留着它，每个日期的字段各有一个写法不对的用例）。

### 2.11 起止日期：领域、用例、存储（Task 10；3.7、3.13）

- `issue/domain`：`Issue.StartDate`、`TargetDate`，`NewIssue` 的两个日期；`dates.go`（新）：`DateChange{To}`、`checkNewDates`（创建：两个都给且开始晚于截止时 `target_date` `not_allowed`）、`DatesOutOfOrder(i, p)`（修改：合并之后的值，只给了开始日期时在 `start_date`，否则在 `target_date`；相等允许）、`sameDay`、`dayText`；`IssuePatch.SetStartDate`、`StartDate`、`SetTargetDate`、`TargetDate`（给出 `null` 是清空）；`Changes.StartDate`、`TargetDate`（与工作项的相同时不算改动）；`Update.StartDate`、`TargetDate` 和动态第 8、9 行（`YYYY-MM-DD` 或 `""`，`comment` 照 Plane："updated the start date to "（带结尾的空格）、"updated the target date to"）。
- `issue/app`：`CheckNewIssue` 在任何读之前查创建的两个日期；`UpdateIssue.check` 在锁下、在状态、负责人、标签之后查合并之后的日期，与它们一次报出；`IssueRow`、`IssueFields` 带两个日期。
- 存储：`InsertIssue`、`UpdateIssue` 写两列，`GetIssue` 读两列（sqlc 的 `date` → `time.Time`，`dayOf`、`timeOf` 换成 `shared.Date`）。
- 测试：领域的 `TestDatesOutOfOrder`、`TestCheckNewIssueAccepts`、`TestCheckNewIssueReportsEveryProblem`、`TestChangesOf`、`TestChangesNone`、`TestCheckIssuePatchAccepts`、`TestUpdateActivity`；单元的 `TestUpdateIssueHoldsTheMergedDatesInOrder`（工作项从 12 日到 19 日：只给晚于留着的截止日期的开始日期，只看请求会放过、合并之后不对，在 `start_date`；只给早于留着的开始日期的截止日期，在 `target_date`；两个都给、各自与原来的另一个日期次序对而彼此不对，在 `target_date`；都在依赖行的问题之后、同一个 422，什么都不写）、`TestUpdateIssueWritesTheDates`（两个都移过原来的截止日期：与原来的值比较会拒绝、合并之后对的，写下；清空开始日期并给早于原来开始日期的截止日期；只给与留着的截止日期同一天的开始日期；清空截止日期；每个改了的日期一行动态）、`TestCreateIssueRefusesBeforeAnyLock`、`TestCreateIssueWritesTheWorkItemAndItsRows`；存储的 `TestCreatingAWorkItemWritesItsRows`、`TestUpdatingAWorkItemWritesItsRows`。

### 2.12 起止日期：接口（Task 11；3.7、5.2）

- 契约：`IssueDetail` 的 `start_date`、`target_date`（必有，可为 `null`，`format: date`）；`IssueCreate` 的（可省略，不可为 `null`）；`IssueUpdate` 的（可为 `null`：清空）；`createIssue`、`updateIssue` 的描述写上日期的规则（`updateIssue` 的"值照 `createIssue` 的规则"一句除去日期的次序，附录 A.2）。
- HTTP：`given[T](nullable.Nullable[T]) (bool, *T)`：没给、给 `null`、给值三种；创建、修改、回答的映射。`TestDatesPassTheBodyCheckAsTheyDecode`（`bodyshape` 的 `checkDate` 放过的与生成的类型（`shared.Date`）解码得了的是同一组，作为创建的和修改的开始日期各发一次：2026-10-19、0001-01-01、9999-12-31、闰年的 2 月 29 日、用转义写的日期，与平年的 2 月 29 日、4 月 31 日、0 月、13 月、0000 年、少一位的月或日、两位的年、斜线、带时间、带时区、前后的空白、带符号、五位的年、全角数字、空串）；`TestCreateIssue`、`TestCreateIssueHoldsTheBodyToItsStructure`、`TestUpdateIssue`、`TestUpdateIssueHoldsTheBodyToItsStructure`。
- `issue/app`（执行中由评审加的）：单元的 `TestUpdateIssueRefusesUnderTheLock` 加"已归档的工作项给次序不对的日期"一行（409 `issue.archived`，先于日期的次序，变异 T11.6）；`TestUpdateIssueThatChangesNothingWritesNothing` 的"每个字段原样"也给出两个日期，照工作项的（没有；变异 T11.5）。
- `bootstrap`：`TestDatesAreKeptInOrder`（alice 建 10 月 12 日到 19 日的工作项；bob 只给晚于截止的开始日期（11 月 1 日：月份晚、日子早）、只给早于开始的截止日期，各 422，在 `start_date`、`target_date`；两个都移过它原来的截止日期、再清空截止日期，各写一行动态；再给它已有的日期，什么都不改、不写一行；alice 把它移进 Done、归档之后，bob 给早于开始日期的截止日期，409 `issue.archived`，先于日期的次序，什么都不写（执行中由评审加的，变异 T11.6）；开始晚于截止的创建（11 月 1 日到 10 月 19 日）422、不占编号，同一天的允许；附录 A.2 加了已有的日期一步和跨月的两个日期）；`TestAWorkItemReadsAsStored` 加两个日期；矩阵的 `createIssue`、`updateIssue` 的"每个字段都给"带日期，`updateIssue` 加"日期的次序不对"一行。

### 2.13 交错（Task 12；3.5，9.3）

| 交错 | 测试 | 断言 |
|---|---|---|
| 12 | `TestArchivingAWorkItemAndMovingItSerialize`（新文件 `issue_interleaving_archive_test.go`，两种顺序） | alice 归档 Done 里的 Fix，bob 把它移到 In Progress：先移：归档过了预检、在锁下读到它在开始组，409 `issue.not_archivable`，没有归档；先归档：修改过了预检、在锁下读到它已归档，409 `issue.archived`，它留在 Done。动态里只有先过锁的那个写的一行 |
| 删除与修改 | `TestDeletingAWorkItemAndChangingItSerialize`（`issue_interleaving_deletion_test.go`，`deletesIssue`，两种顺序） | 先删：修改过了预检、锁下找不到 Fix，404 `issue.not_found`，没有写，Fix 的三行已删除；先改：删除等 Web 的行，然后把工作项和它的行（含 bob 的一行动态，共四行）同一时刻由 alice 删除，时刻不早于修改的提交 |
| 4、5（创建者的删除，访客一支） | `TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused` 加"删除他自己的" | 预检放行之后被移出：404 `issue.not_found`；被降为"只看自己"的项目里的访客：改 alice 的 Fix 404（访客规则），改、删他自己的 403；都没有写（`AllowCreator` 在锁下再判定） |

`wroteNothing` 也数未删除的行（删除的写什么都没写，行数照旧）。`-race -count=5`：附录 A.4。

### 2.14 端到端（Task 13；第 2 节 I1、I2、I15）

- 夹具：`e2e/fixtures/issues.ts` 的 `issueArchived`、`issueUnarchived`、`issueDeleted`（状态和回答）；`assert/issue.ts`：`archivedRow`、`restoredRow`（第 14、16 行，经共用的 `changed` 一类的形状写）、`CreatedIssue`、`UpdatedIssue` 加 `start_date`、`target_date`，`expectIssueArchival`，`expectIssueDeleted`（从目录读出外键图（`assert/catalog.ts` 的 `keysBelow`，与 `tablesUnder` 同在一处；走到的外键多于一列时失败，同 Go 的 `keysTo`），自 `issues` 向下递归：两层以下的行（评论的表情一类）也算在它下面；每个外键下都要有一行，`subItems` 豁免到 P3；在工作项之前已删除的行不算，也不在这里核对：它保留原来的删除由 `TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt` 核对）。
- I15 的接口版本（新 `i15-archive-issue.spec.ts`）：ann 在 In Progress 的工作项 409 `issue.not_archivable`；移到 Done 后归档（一行动态），再归档不写；已归档时修改 409 `issue.archived`；gus（项目的访客，项目"可见全部"）恢复、归档都 403；恢复（一行动态），再恢复不写；再归档之后，管理员删除它：`expectIssueDeleted`。
- I1 加起止日期和开始晚于截止的 422（与别的值的问题一次报出）；I2 加日期的修改（两行动态）、只给开始日期晚于留着的截止日期和只给截止日期早于留着的开始日期各 422、两个都移过原来的截止日期（只看请求会拒绝的一步不被拒绝）、清空、已归档 409。访客取别人的工作项的 404 在本 Phase 只由矩阵（Go）核对，端到端的那一格随 I20 在 P5（设计 12 节 P2 的第 13 个任务）。

### 2.15 文档（Task 14；3.23）

总体设计 6.5（`AllowCreator` 一句改写：创建者要是项目的有效成员、角色不低于成员；`OwnOnly` 和一条访客规则；Plane 的两条照原样）；M2 设计 3.11（下标按数值排序、不限类型的节点的数、`shared.Date` 的检查器，`date` 从失败的例子里去掉）；差异清单第四节 4.17 中 P2 的七行（"归档、恢复"一行写明 Plane 恢复没归档的也答 404，动态的 `comment` 不同，第 3 节第 8 条）；三份交接（M1-P3、M2 收尾、M3 收尾）写下"处理结果（M4/P2）"，`status` 照旧 `open`（第 7 节）。

## 3. 与设计的差异和补充

第 1 条是控制者允许的重新切分；第 2–19 条是本 spec 对设计交给 P2 的事的决定和补充；第 1–19 条都已由控制者照建议裁定（2026-10-19），带条件的（第 3、8、10、14 条）把条件写在各条里，预检（2026-10-20）的发现按控制者的裁定改在各条（附录 A.2）；第 20 条是承接条目的落点。

1. **重新切分**（已裁定：照本条；规则的行分在 Task 1、2，访客在两个 Task 之间读不到别人的工作项）。设计 12 节 P2 的十四个草稿任务与 plan 的 14 个 Task：草稿 1 是 Task 1，草稿 2 分在 Task 1（`issue.delete`、`issue.archive` 的行）和 Task 2（`issue.read` 加访客、`guest_view_all_features` 的说明：访客的读要访客规则同时到位，否则访客在 Task 1 之后读得到别人的工作项）；草稿 3 是 Task 2，交错 5 的访客一支（草稿 10 的一半）随它；草稿 11（矩阵）的列和种子提前成 Task 3，此后每个操作的行随它的 Task；草稿 4 是 Task 4（`expectIssueDeleted` 随端到端在 Task 13）；草稿 5 拆成 Task 5（领域、用例、端口、存储）和 Task 6（接口、矩阵、组合）；草稿 6、7、8 是 Task 7、8、9；草稿 9 拆成 Task 10（领域、用例、存储）和 Task 11（接口、组合、矩阵）；草稿 10 的交错 12 是 Task 12；草稿 12（整程序测试、`apitest` 两个方向）分在 Task 9（`TestBodyCasesOfDates`）、Task 11（`TestDatesPassTheBodyCheckAsTheyDecode`、`TestDatesAreKeptInOrder`）和每个加操作的 Task（`apitest.Main` 两个方向随契约）；草稿 13、14 是 Task 13、14。设计里"P2 的任务 n"一类的引用（17 节的表）仍指 12 节的草稿列表，不改；本 spec 引用它们时写"设计 12 节 P2 的第 n 个任务"。每个 Task 在 plan 里的行数见附录 A.1。
2. **访客规则在 `decide` 里，404 优先**（已裁定：照本条；P1a 第 5 节"判定的 `Grant` 重新带出"的落点）。`Authorizer` 角色不允许时回答 `shared.Forbidden` 和这次判定读到的 `Grant`（`shared/authorize.go` 的说明、`TestARefusalCarriesTheGrant`），`Locks.decide` 在 `ErrNotVisible` 之后、在角色的拒绝之前看 `HiddenFrom`：一个只看自己的访客对别人的工作项，不论他的角色是否允许这个动作，都是 404（3.4、6.7："看不到的工作项和挂在它下面的一切都答 404"；8.4：别人的工作项对他与不存在的同样不可知）。预检（`see`）和锁下的判定（`lockAndDecide` 传锁下重读的工作项）走同一个 `decide`，访客规则因此在锁之前答，锁下再答一次，以锁下的为准（交错 5 的访客一支）。另一种写法是 `Authorize` 先看访客规则：它不读工作项，要把 `Target` 再加一个"创建者是否就是调用者"的事实，`access` 就要知道工作项的语义；本条让 `access` 只算 `OwnOnly`，过滤留在用例（设计 3.4"用例拿它过滤"）。创建者不明（`created_by_id` 为空，P9 物理删除账户之后）的工作项对只看自己的访客隐藏，对 `AllowCreator` 不是谁的：单元的假存储把零值的创建者读成没有（`creatorOf`），读者的表和 `TestADecisionOnAWorkItemNamesItsCreator` 的删除各有这样一行（变异 T2.3、T2.9；P9 之前没有接口造得出这样的工作项，按性质只在单元一层，附录 A.3；P9 加组合一层的用例，第 5 节）。设计 3.4 的"单个工作项"一句照裁定写明角色不允许时也先看访客规则（404 先于 403，6.7），本提交已改。
3. **`ProjectGate.Lock` 不读 `guest_view_all_features`**（已裁定：照本条；P1a 第 5 节"由 P2 定"，依据是已裁定的 P1a 第 3 节第 5 条：端口只带它的读者用的东西；条件：修订设计 3.3 和 6.5 的 `ProjectGate`）。锁下的判定经 `Authorize` 在事务里重读 `ProjectFacts`（含这个标志），`OwnOnly` 从 `Grant` 来，用例不再读项目（设计 3.4）；`Lock` 的回答没有这一列的读者。设计 6.5 的 `ProjectGate` 一行列出 `Lock` 回答 `guest_view_all_features`（和 `identifier`、`archive_in`），与 P1a 第 3 节第 5 条同一个道理（端口只带它的读者用的东西）：P2 不加。本提交照裁定修订设计两处：3.3 的 `ProjectGate` 一句和 6.5 的 `ProjectGate` 一行都写明 `Lock` 不回答 `guest_view_all_features`，锁下的判定经 `Authorize` 读它（6.5 的一行另写列表在 P5 经 `VisibleProjects` 读它，3.4），那一行的其余各列不动。
4. **按编号取先判定工作区**（已裁定：照本条，新端口的名字和 `workspace/module.go` 的说明在内）。设计 3.4 的 `issue.search` 一行说"按编号取按目标项目判定 `issue.read`"，5.1 的表给它 404 `workspace.not_found` 和 404 `issue.not_found`，9.2 的矩阵给"看得到而不是项目成员的工作区管理员"403。本 spec：先找工作区（`WorkspaceBySlug`），以 `issue.search`（工作区级，三个角色）判定调用者是它的有效成员，不是的、工作区不存在或已删除的 404 `workspace.not_found`（与 M3 每个按 slug 寻址的操作同一个答案）；然后找工作项、以 `issue.read` 在它的项目上判定（`see`，访客规则在内）。工作区一步的判定交给 `Authorizer`，而不是在用例里另读成员关系（M3 的约定：判定只经 `Authorize`）；`issue.search` 的行在 P2 加入（3.4 的表本来就有它，P8 的搜索照用）。契约声明 `workspace.not_found`、`issue.not_found`、`forbidden`：5.1 的表没写 403 F，9.2 的矩阵有（公开项目里不是成员的工作区成员、不是项目成员的工作区管理员）。项目一侧的读：工作区未删除的项目，归档的也在（已归档的项目里的工作项照样读，3.8），`id` 的次序；已删除的项目的标识可能已被别的项目用（M3 设计 3.19），不在其中。新端口的名字（`WorkspaceDirectory`、`ProjectIdentifiers` 和 `ProjectIdentifier`、`IssueNumbers`；`project` 一侧的 `IssueProjects.ProjectIdentifiers` 和 `IssueProjectIdentifier`）和 `workspace/module.go` 的说明一并照用。
5. **拒绝的次序**（已裁定：照本条，这就是设计 3.5 的表，由测试钉住）。修改：项目已归档 409 → 工作项已归档 409 `issue.archived`（什么都不改的也是：P1b 第 3 节第 4 条）→ 什么都不改时不写 → 依赖行和合并之后的日期，一次 422。归档：项目已归档 409 → 已归档：不写，200 → 状态组 409 `issue.not_archivable`；恢复：项目已归档 409 → 没归档：不写，200（不看状态组，3.8）。删除：项目已归档 409；已归档的工作项照样删除（3.8 的"除恢复和删除之外"）。这些都在判定之后：不能做这个动作的人得不到工作项的状态（矩阵"Todo 里的""已归档的"两行的 403、404 格子）。钉住它们的：单元的拒绝表；矩阵的"已归档的项目里已归档的工作项"（`archiveIssue`、`updateIssue` 各一行，变异 T7.2、T8.3）和 `unarchiveIssue` 的"已归档的项目"一行（工作项没归档：恢复的次序，变异 T7.2；`unarchiveIssue` 的"已归档的项目里已归档的工作项"一行看不到这个次序，恢复已归档的工作项不是"已是所求的样子"，那一行钉住的是 T7.1）；删除照样删除已归档的工作项：单元的 `TestDeleteIssueDeletesTheWorkItemAndItsRows` 的 `an archived work item` 一例和矩阵 `deleteIssue` 的"已归档的工作项"一行（变异 T8.8，执行中由实现者发现；`deleteIssue` 的"已归档的项目里已归档的工作项"一行钉住项目的 409 在先）；交错 12 的两种顺序。归档的"已归档先于状态组"另在组合一层钉住：已归档的工作项所在的状态改到"进行中"组之后再归档，照样回答、什么都不写（`TestArchivingAWorkItemStampsItsRows`，变异 T7.25，执行中由评审发现：别的组合测试再归档的都是 Done 里的工作项）；恢复不看状态组也在那里钉住：之后照样恢复（变异 T7.20，预检之后加的）。
6. **删除工作项的步骤**（已裁定：照本条）。用例编排五条语句（`issueRowSteps`），每张表一条，在项目 N 之下、同一个时刻和删除者；表序是全局的（工作项 → 负责人 → 标签 → 关注 → 动态，与删除项目的 `rowSteps` 同一个次序，3.5 的约定）：工作项在先由取锁形状的 `deleteIssue` 一行钉住（变异 T4.8 先写负责人的行，在等 `issues` 时已写了 `issue_assignees`，执行中由评审发现），四张子表之间的次序只在单元一层。不写"已删除"的动态（3.6 已定）。语句是不分批的 `UPDATE … WHERE issue_id = $1 AND deleted_at IS NULL`：一个工作项的行有界（负责人、标签各至多 100 个；动态随它的修改增长，没有上限，但只是一个工作项的），与删除项目的键集分批不同。编号不再使用（序号在项目上，`issues_project_id_sequence_id_key` 含已删除的）。
7. **约定五和死锁**（已裁定：照本条，以预检对 HEAD 的每个写的重推为条件，设计 12 节约束 3 要 P2 重推；预检重推过：约定五对每个写者成立，算上闸门的位置也没有等待环）。P2 的批量写：`DeleteIssueAssigneeRows`、`DeleteIssueLabelRows`、`DeleteIssueSubscriberRows`、`DeleteIssueActivityRows`（按谓词软删除一个工作项的若干行，按扫描的次序锁行）和单行的 `DeleteIssueRow`、`SetArchivedAt`；修改的日期只是 `UpdateIssue` 多写两列。这些行的每个写者都先持这个项目的 N：工作项的每个写（创建、修改、删除、归档、恢复，都经 `Locks.run`）、删除标签的 `DetachLabel`、删除项目和工作区的连带（P1–P8 同步，在项目 N 之下）；P9 的任务只在项目删除之后写，那时已没有写到得了这些行（3.6）。所以这些语句在 N 之下没有第二个写者，不必按 `id` 先锁（约定五的前提，与 P1b 第 3 节第 12 条同一个论证）。**死锁**，把闸门的位置也算作资源：P2 的每个写依次取闸门的位置（等的时候不持连接、不持锁）→ 连接 → 项目行 N（第一把行锁）→ 这个项目的工作项一侧的行（一次一张表、全局的表序）；它从不持着行锁再等闸门或连接，也从不持着项目 N 再等工作区的行（插入的外键取的 `FOR KEY SHARE` 与工作区上的任何写锁都不冲突，S5 的 E3）；工作项的写也可能持着 N 等一行 `users`（外键的 `FOR KEY SHARE` 与 `users set-email` 改邮箱时的 `FOR UPDATE`）：那个写除了这一行不持别的锁、不等任何别的写，所以没有环。在 HEAD 的每个写里：同一个项目的写在 N 上排队，持 N 的不会等另一个写持的这个项目的行（那个写也得先持 N）；删除工作区先锁工作区、再按 `id` 升序锁它的项目（`LockWorkspaceProjects`），它等一个持 N 的工作项的写，而那个写不等工作区；M3 的项目一级的写（成员、状态、标签）持项目 N，工作项的写在 N 上排在它后面；等连接的写不持任何锁，持连接的写只等持连接的写持的锁，锁的等待按"工作区 → 项目（`id` 升序）→ 工作项一侧的行（表序）"无环。所以没有新的等待环；万一有，`lock_timeout`（5 秒）和闸门的 `max_wait` 让它答 503，不是挂住。附录 A.4：P2 的交错和 P1b、P1a、M3 的交错、竞争、取锁形状的测试 `-race -count=5`，没有 40P01。
8. **归档、恢复的动态和回答**（已裁定：照本条；`comment` 是产品的文字，照 Plane 的文字、除非它对 Nerve 是错的；条件：差异清单的"归档、恢复"一行写上"动态的 `comment` 不同"，在 Task 14）。第 14、16 行的 `field`、`verb`、值照 3.13 的表；`comment` 是 "archived the work item"、"restored the work item"。Plane 的是 "Actor has archived the issue"（手动归档，`Actor` 是没有替换的字面量，`bgtasks/issue_activities_task.py:509`）和 "has restored the issue"（`:495`）；P1b 的行照 Plane 的文字（P1b 第 3 节第 6 条），这两行改了：一是 "Actor" 不是任何人，二是 Nerve 说"work item"。页面的渲染器不读 `comment`（3.13）。错了的代价是两句文字、它们的单元测试和端到端的两行。差异清单的"归档、恢复"一行写上"动态的 `comment` 不同"（Task 14 的块，预检之后补上）。归档、恢复回答完整的 `IssueDetail`（3.8）；已是所求的样子时回答锁下读到的工作项，不写任何行（`updated_at` 不变，没有动态，`TestArchivingAsItIsWritesNothing`、矩阵"已归档的""没归档的"两行）。
9. **文案和补救的措辞**（已裁定：照下面的措辞，Go、契约、en 和 zh-CN；产品的文字，是本地的决定）。Task 7、8 照下面的措辞实现，日后改措辞只改这几处（Go 的说明、契约、两份 `errors.json`；比较整个回答体的 HTTP 测试：`state_writes_test.go` 的 `stateInUseJSON`，`issue/adapter/http` 的 `update_test.go` 的 `an archived work item` 一行和 `archive_test.go` 的 `a work item in progress` 一行（`issue.not_archivable`））：
   - `issue.not_archivable`：Go "Only a completed or cancelled work item can be archived; move it to such a state first."；en "Only a completed or cancelled work item can be archived. Move it to such a state first."；zh-CN "只有已完成或已取消的工作项才能归档。请先把它移到已完成或已取消的状态。"
   - `issue.archived`：Go "The work item is archived; unarchive it to change it."；en "The work item is archived. Restore it to change it."；zh-CN "工作项已归档，恢复之后才能修改。"
   - `project.state_in_use`：Go "Work items are in the state, archived ones too, so it cannot be deleted; unarchive the archived ones and move them all to another state first."；en "… Restore the archived ones and move them all to another state first."；zh-CN "……请先恢复已归档的，再把它们都移到别的状态。"
   
   接口的说明用操作的名字（unarchive），页面的文案用页面的词（Restore、恢复，Plane 的按钮）。两条补救对调用者都做得到：移动在 P1b，恢复在本 Phase；访客做不到（他不能改、不能恢复），与别的码的补救同一个写法（对能做这个动作的人说）。在项目里是访客的工作区管理员能删除状态，却看不到别人的工作项：补救要他先把自己在项目里的角色改为成员或管理员（M3），这一点照样成立。已归档的项目的状态照样能删除（M3 设计 3.19），补救的恢复和移动在那里却先答 `project.archived`：它的文字（"unarchive it to change it"）补上它们之前的一步，这一点照样成立（执行中由实现者发现）。
10. **日期的类型**（已裁定：照本条，含 `bodyshape.checkDate` 和 `shared.ParseDate` 各带一份的两行规则（`time.DateOnly`，年份不小于 1）；条件：两处的说明互相指名，改一处的人找得到另一处：`ParseDate` 的说明指名 `bodyshape` 的 `FormatDate`，`checkDate` 的说明写"by shared.ParseDate's rule"，已是这样）。一个 `shared.Date`：领域、HTTP 生成的类型、请求体的检查用它；存储一侧 sqlc 把 `date` 映射到 `time.Time`（UTC 的零点），适配器换成 `shared.Date`（`dayOf`、`timeOf`），不让 `shared` 实现 pgx 的接口。五个模块的 `oapi-codegen.yaml` 都映射（今天只有 `issue` 有日期）：默认的 `openapi_types.Date` 带进 `github.com/google/uuid`（M2 交接第 4 节），哪个模块先加日期都不会悄悄用它。`bodyshape` 不能导入 `shared`，`checkDate` 照 `ParseDate` 的规则另写一份，`TestDatesPassTheBodyCheckAsTheyDecode` 钉住两者接受同一组字符串（变异 T9.2 在它上被发现，Task 11 起）。不加 `archtest` 的规则禁止 `openapi_types.Date`：生成器已拒绝请求体里没有映射的 `date`（`TestGenerateFollowsTheTypeMapping`），回答一侧的类型在每个模块的映射里。
11. **合并之后的日期**（已裁定：照本条，设计 3.7 的规则，这里定位置）。创建：两个都给且开始晚于截止，在任何读之前 422（`target_date`），与别的值的问题一次报出（`CheckNewIssue`）。修改：合并之后的值（给出的，没给的取工作项锁下重读的）在锁下检查，在状态、负责人、标签之后，与它们一次报出（`check`）；只给了开始日期时问题在 `start_date`，否则在 `target_date`（3.7）；相等允许；两个都给、次序不对的修改也在锁下答（所以发给不存在的工作项是 404，不是 422）：一条规则一个地方。`updateIssue` 的描述"值照 `createIssue` 的规则"一句因此写明除去日期的次序（附录 A.2）。动态第 8 行的 `comment` 照 Plane 带结尾的空格（"updated the start date to "，`bgtasks/issue_activities_task.py:283`），第 9 行不带（`:253`）：P1b 的"照 Plane 的文字"。测试数据两个方向都有：只看请求会放过、合并之后不对的（只给开始日期晚于留着的截止日期），和只看请求会拒绝、合并之后对的（两个都给、开始晚于原来的截止日期）（`TestUpdateIssueHoldsTheMergedDatesInOrder`、`TestDatesAreKeptInOrder`、I2）。
12. **`bodyshape` 的改动**（已裁定：照本条）。`problems` 和路径的代码移到新的 `paths.go`（`bodyshape.go` 加了日期之后超过约 400 行）。`comparePaths` 把路径切成名字和下标的段，下标按数值比较，其余逐字节，是全序（同一组问题总是同一个次序）。`numbers`：不限类型的节点的值由已有的 `scanner` 带 `numbers` 一遍读完（不论多深），每个放不进 `float64` 的数在它的路径报 `invalid_format`；它在已有的歧义扫描之后、同一个值只读一次，代价测试（`TestCheckCostsAboutTheBody`）的体加字符串的数组和开放节点里的数。map 型（`additionalProperties` 是模式）和递归的结构在契约里没有用到：M2 交接第 4 节的这两项写明"没有用到"（Task 14）。
13. **取锁形状的测试**（已裁定：照本条，P1b 评审第 6 节的"P2 会加同一个方法的行"）。`issueWrite` 的 `reads` 改为 `waits` 和 `mode`：写在锁下、它自己的任何写之前等的表——锁下第一个读、预检没读的表（别的事务持 `ACCESS EXCLUSIVE`），或锁下什么都不多读时它第一个写的表（持 `SHARE`，预检的普通的读不等它）——和别的事务持它的模式。读表的写（创建、修改、归档：`states`）用 `ACCESS EXCLUSIVE`（普通的读也等它）；锁下什么都不多读的写（删除、恢复）等它第一个写的表 `issues`，用 `SHARE`（写等它、普通的读不等，所以预检的读不被它挡住）。每次发送用新建的目标（`target` 每次建一个）：删除、归档之后同一个目标不能再发。`issueWorld.doneWorkItem`（Done 里的工作项）共用。
14. **按目录的删除测试**（已裁定：照本条；条件：两处豁免写进第 5 节 P3 一行，作为有约束力的交接，已写）。`TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt` 照删除项目的写法从目录读 `keysTo(issues)`；`issues.parent_id`（子工作项，`subItems`）豁免到 P3（P2 的工作项都没有子工作项，3.9），P3 去掉豁免（第 5 节）。`expectIssueDeleted`（端到端）沿外键图递归、同一个豁免。PF.3（P1a 评审第 6 节）落在删除项目的测试：种子加一个在项目之前由 alice 删除的工作项，项目删除之后它的行照旧是她的删除；删除工作区的测试只改辅助的签名（`noRowLastWrittenBy` 取外键的列表）：分批语句的 `deleted_at IS NULL` 是同一条，一处钉住。
15. **矩阵的工作项表和 `prepare`**（已裁定：照本条；`permission_matrix_test.go` 406 行，在约 400 行以内，Global Constraints 点名它）。9.2 的"创建者的两行"写成列的格子（2.5）：一个访客自己的工作项只能是他从前作为成员建的（访客不能创建，`createIssue`）；种子经存储把它记在他名下：判定只读创建者和他现在的角色，不读它怎样来的；"已离开项目"的创建者在私有项目里，离开之后看不到它。acme 的公开项目照默认不"可见全部"、私有项目"可见全部"：访客的三种情形是同一个账户的。`matrixRow.prepare` 是新的设施（P1b 的矩阵没有）：格子要的状态（Done、已归档）是工作项的，只能在格子自己的副本里准备，否则一行的格子互相影响；它在副本上、app 启动之前运行，只给写的行（读的格子共用模板库）。
16. **交错的补充**（已裁定：照本条）。设计 3.5 的交错表没有"删除与修改同一个工作项"：本 spec 补上两种顺序（`TestDeletingAWorkItemAndChangingItSerialize`），删除也是按资源寻址的写，锁下重读决定修改看到的是不是还在。交错 4、5 加"删除他自己的"：`AllowCreator` 在锁下再判定，等锁期间被移出 404、被降为访客 403（3.4 的"创建者仍是有效成员、角色不低于成员"）。删除工作项与删除项目、工作区（交错 2、3 的形状）不另写：两者都先取项目 N，与修改一支同一个论证（第 7 条），先删项目的，删除工作项在锁下读不到项目，404。
17. **I15 删除已归档的工作项**（已裁定：照本条）。I15 的接口版本以删除结束：已归档的工作项照样删除（3.8），`expectIssueDeleted` 第一次被用到（I17 在 P3）。故事里的访客在"可见全部"的项目里：他看得到工作项，归档、恢复都 403（I15 的"项目访客 403"）；"访客·别人"的 404 只由矩阵核对（设计 12 节 P2 的第 13 个任务）。
18. **`wroteNothing` 加强**（已裁定：照本条）。P1a、P1b 的交错用它核对被拒绝的写什么都没写；删除只改 `deleted_at`，行数不变，所以它也数未删除的行（Task 12）。
19. **从 P2 起不手写 `55P03`**（P1b 第 5 节收尾一行的最后一句）。P2 的新测试没有手写的 `55P03`（`grep` 核对）；交错和取锁形状的测试经 `pgtest.WaitForLockWaitBehind` 等，不用 `lock_timeout`。已裁定：确认。
20. **承接的条目**：

    | 来处 | 条目 | 落点 |
    |---|---|---|
    | P1a spec 第 5 节、评审第 6 节 | 判定的 `Grant` 由 `decide`、`precheck`、`lockAndDecide` 带出，`OwnOnly` 的拒绝在预检里答 | 2.4、第 2 条 |
    | 同上 | `OwnOnly` 由 `access` 的 `ProjectFacts` 算出；`ProjectGate.Lock` 要不要 `guest_view_all_features` | 2.3、第 3 条（不要） |
    | 同上 | `project.state_in_use` 的补救 | 2.9、第 9 条 |
    | 同上 | `expectIssueDeleted` 从目录自查、沿外键图递归 | 2.14、第 14 条 |
    | 同上 | 按目录的删除测试的种子加一个在项目之前已删除的工作项（PF.3） | 2.6、第 14 条 |
    | 同上 | `bodyshape` 的代价测试加字符串的数组 | 2.10、第 12 条 |
    | 同上 | 起止日期进接口；按编号取 | 2.7、2.10–2.12 |
    | P1b spec 第 5 节 P2 一行 | `issue.read`、`issue.update` 的访客；`see` 和预检在锁之前答访客规则的 404 | 2.4、第 2 条 |
    | 同上 | 已归档的工作项的修改 409 `issue.archived` | 2.9、第 5 条 |
    | 同上 | 起止日期进 `IssuePatch`（合并之后的值） | 2.11、第 11 条 |
    | 同上 | `project.state_in_use` 的补救 | 2.9、第 9 条 |
    | P1b spec 第 5 节收尾一行 | P2 起新的测试用平台的帮手，不手写 `55P03` | 第 19 条 |
    | P1b 评审第 6 节 | `getIssue` 判定的动作（T4.7）在 P2 之后组合一层才看得到 | 变异 T2.8（`getIssue` 判定 `issue.create`）在 Task 3 起的矩阵被发现（附录 A.6） |
    | 同上 | 取锁形状的行随新的写加（归档、恢复） | 2.6、2.8、第 13 条 |
    | M1-P3 交接的地址一项 | 按编号取统一到 `/workspaces/{slug}/issues/{identifier}` | 2.7（另一半：`browse` 的路由在 P14） |
    | M2 收尾交接第 3 节 | P2 的用例用时钟写 `updated_at` | 2.6、2.8、2.11（单元测试的 `clockAt`，存储测试断言交给它的时刻） |
    | M2 收尾交接第 4 节 | 路径按下标排序、数的范围、`date`、代价测试；map 型、递归的结构 | 2.10、第 10、12 条（后两项"没有用到"） |
    | M3 收尾交接第 2 节 | P2 的批量写逐个按约定五核对 | 第 7 条 |
    | M3 收尾交接第 3 节 | `AllowCreator`、`guest_view_all_features`、矩阵、码 | 2.3–2.5（个人主页的一行在 P5） |
    | M3 收尾交接第 4 节 | `expectIssueDeleted`；`guest_view_all_features` 的说明 | 2.14、2.4 |
    | P1a、P1b 的教训（brief） | 数据分不出性质、复制的测试帮手、任务结束时不成立的句子、计时按条件 | 附录 A.2、A.6（清扫 57–60） |

## 4. 验收标准（完成线，M4 设计 12 节 P2）

- [ ] I15 的接口版本（归档、恢复、两条动态、`issue.not_archivable`、已归档时改字段 409、访客 403、删除已归档的）、I1 和 I2 的起止日期（含合并之后的日期 422 的两个方向）、I2 的已归档 409 通过；`make e2e` 123 个：此前的 122 个，加 I15。
- [ ] 判定表、矩阵（P2 的格子）、按目录的删除测试、`bodyshape` 和生成器的测试通过。
- [ ] 一个看不到工作项的访客对它的每个读和写得到 404、不是 403：矩阵的"访客·别人""管理员·访客"两列（`getIssue`、`getIssueByIdentifier`、`updateIssue`、`deleteIssue`、`archiveIssue`、`unarchiveIssue`），交错 5 的访客一支。
- [ ] 交错 12、删除与修改的两种顺序、交错 4、5 的删除和访客一支 `-count=5 -race` 通过，没有 40P01；P1b、P1a、M3 的交错照旧通过（附录 A.4）。
- [ ] `make gen-check`、`make lint-go`、`make test`、`make lint-web`、`make knip`、`make test-web`、`make e2e` 通过。

## 5. 不在 P2 范围内

- 子树的删除、父子与关联（P3），评论和它的 409（P4），列表里的访客规则、`?archived=true` 的列表和 `Issue` 的列表行（P5），事件（P6：设计 3.5 的"删除""归档、恢复"两行末尾的"事件"），自动归档（P6），搜索（P8），I15、I1、I2 的页面版本（P13、P14）。

**交给后面的 Phase**：

| Phase | 条目 |
|---|---|
| P3 | 删除工作项连带子树（3.6、3.9）：`DeleteIssue` 的步骤加子树和关联行，`TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt` 和 `expectIssueDeleted` 都去掉 `subItems`（`issues.parent_id`）的豁免（第 3 节第 14 条的裁定：有约束力的交接），I17 用 `expectIssueDeleted`；已归档的父工作项、关联的两端 409 `issue.archived`，已归档的不能成为父工作项、关联的对方（3.8）；删除工作项与关联的交错（11、13）重推第 3 节第 7 条的死锁论证（关联行按 `id` 升序，约定五的例外） |
| P4 | 已归档的工作项上评论 409 `issue.archived`；访客规则用到评论、表情、描述版本的读写（挂在工作项下面的一切） |
| P5 | 列表按访客规则过滤（`VisibleProjects` 带 `guest_view_all_features`，3.4）；`?archived=true` 的列表（规则 `issue.archive`）；`Issue` 列表行的 `start_date`、`target_date`；端到端里访客取别人的工作项的 404（I20 的接口版本，设计 12 节 P2 的第 13 个任务） |
| P6 | 删除、归档、恢复、修改发布事件（`issue.deleted`、`issue.archived`、`issue.unarchived`、`issue.updated`，3.19）；自动归档（第 15 行）；`ProjectGate.Lock` 加 `archive_in` |
| P8 | `searchIssues` 照用 `issue.search` 和 `foldIdentifier`（3.18） |
| P9 | 账户物理删除之后 `created_by_id` 为空的工作项：访客规则隐藏它，`AllowCreator` 不认任何人（第 3 节第 2 条）；P2 只在单元一层核对这两半（假存储的 `creatorOf`，变异 T2.3、T2.9，附录 A.3），P9 加组合一层的用例 |
| P13、P14 | I15 的页面版本、已归档的详情只读；I1、I2 的页面版本带日期；`/:workspaceSlug/browse/:workItem` 经 `getIssueByIdentifier`（M1-P3 交接的地址一项的另一半） |
| 收尾 | 3.23 中 P2 的行由 Task 14 写好，收尾逐行核对；三份交接的 `status` 在 M4 收尾随 review 关闭；第 3 节第 3、8 条的修订已做：设计 3.3 的 `ProjectGate` 一句和 6.5 的 `ProjectGate` 一行（本提交），差异清单的"归档、恢复"一行（Task 14），收尾核对它们 |

## 6. 风险

| 风险 | 应对 |
|---|---|
| 某个读写漏了访客规则 | 每个指名工作项的判定只经 `Locks.decide`（`see`、预检、锁下的判定）；矩阵的"访客·别人""管理员·访客"两列覆盖每个操作（`matrixViolations` 要求每个操作都有行）；变异 T2.1、T2.5、T2.6 在组合一层被发现 |
| 预检与锁下的判定不一致（降级的访客、离开的创建者） | 锁下的判定是最终的（交错 4、5 的访客和删除一支；T12.1 在组合一层） |
| 删除工作项与别的写死锁 | 第 3 节第 7 条的论证；`-race -count=5` 的交错、竞争、取锁形状的测试没有 40P01（附录 A.4）；`lock_timeout` 和 `max_wait` 兜底 |
| 日期在 `bodyshape` 与 `shared.Date` 之间不一致 | `TestDatesPassTheBodyCheckAsTheyDecode`（T9.2 Task 11 起被发现）；生成器拒绝没有映射的 `date` |
| 按编号取的 404 透露项目或工作项存在 | 每种 404 的回答逐字节相同（`TestAWorkItemIsFoundByItsIdentifier`）；看不到的项目、访客规则隐藏的与不存在的同一个码；时间上不同：看不到的比不存在的多一次判定的读（与 M3 看不到的项目相同），不处理 |
| 一个工作区的项目很多时按编号取慢 | 每次读工作区的全部项目的标识（v0 的规模下是几十个）；需要时 P8 的搜索一起加索引 |
| 文案、`comment` 的措辞 | 第 3 节第 8、9 条已裁定；日后改措辞只动几处 |
| 持续集成（ubuntu）上的运行 | 原型和复现都在 macOS 上；合并之后看持续集成 |

## 7. 已知的限制、交接和关闭条件

- **访客取别人的工作项的端到端格子不在 P2**：本 Phase 由矩阵（Go）核对，端到端的那一格随 I20 的接口版本在 P5（设计 12 节 P2 的第 13 个任务）。
- **删除的语句不分批**：一个工作项的动态随它的修改增长；目标规模（一个项目约 2 万个工作项、每个约 20 行动态）上一条语句是几十行，没有问题；极端的工作项（几万行动态）由 `lock_timeout` 之外的请求期限兜底。
- **事件**：删除、归档、恢复不发布事件，P6 加（设计 3.19 的用例的发布点）。

| 交接 | P2 处理的条目 | 留下的条目 |
|---|---|---|
| M1-P3 交接的地址一项 | 按编号取统一到 `/workspaces/{slug}/issues/{identifier}`（契约和用例） | `browse` 的路由经它（P14）；旧的工作项地址的守卫（P10a） |
| M2 收尾交接第 3 节 | P2 的用例（删除、归档、恢复、日期的修改）用时钟写 `updated_at`、`deleted_at` | P3、P4、P7 的其余写工作项的用例；自动归档推进固定时钟的测试（P6） |
| M2 收尾交接第 4 节 | 路径按下标的数值排序、开放节点的数的范围、`date`、代价测试；map 型、递归的结构"没有用到" | 无 |
| M3 收尾交接第 2 节 | 约定五：P2 的批量写（删除工作项的五条语句，归档、恢复的 `SetArchivedAt`）都在项目 N 之下，不落在例外里（第 3 节第 7 条） | P2 之后各 Phase 的批量写（到 P9） |
| M3 收尾交接第 3 节 | `AllowCreator`、`guest_view_all_features`、矩阵（访客的三列、"管理员·访客"、创建者）、码 | 个人主页的一行（P5） |
| M3 收尾交接第 4 节 | `expectIssueDeleted`；`guest_view_all_features` 的说明 | 删除断言的限时轮询（P9） |

三份交接的"处理结果（M4/P2）"由 Task 14 写下，`status` 照旧 `open`，在 M4 收尾随 review 关闭（P1a、P1b 的做法）。

**M4 设计 13.1 的关闭条件**：落在 P2 的（M1-P3 的地址一项的一半，M2 交接第 3 节 P2 的用例和第 4 节，M3 交接第 2 节 P2 的批量写、第 3 节和第 4 节的两项）都在上表。没有放不下的条件。

## 附录 A：原型验证记录（2026-10-19；预检之后，2026-10-20）

### A.1 做法和 Phase 的大小

原型在 `$M4TMP/p2proto`：`7db78788`（P1b 的合并提交）的一份 `git archive`（`p2tools/base-7db78788.tar`，快照 `p2snap/T0`）之上逐 Task 做，每个 Task 做完存一份源文件的快照（`p2snap/T1`…`T14`）；`node_modules` 由 `pnpm install --frozen-lockfile` 装好（不用符号链接），`bin/golangci-lint` 2.13.2；Go 1.27.1（`toolchain`）、Node 24.15.0、pnpm 11.10.0、Docker 29.7.2。plan 的代码块由脚本从相邻两份快照的差异生成（`p2tools/mkblocks.py`），每个块从基点起按顺序核对过（`problems: 0`；`planapply.mjs check`：547 个块都对）；生成的文件不进块，按 SHA-256 和行数钉住（`gensha.py`），plan 在每次生成之后有一步 `shasum -a 256` 与表核对（Task 9 的 `make gen` 不改任何生成物，没有表）；`compose_plan.py` 组装 plan，核对每个块放了一次、每个文件在文件表里（`filetablecheck.py`）、没有遗留的标记，并从变异的结果写出每个 Task 的变异表（被发现的层、"（Task n 起）"）。之后的修改（A.2，清扫之后的和预检之后的）都写回拥有那个文件的 Task 的快照，从那个 Task 起每一份快照都改（`p2tools/edits/` 的编辑说明），`mkblocks.py` 重新生成全部块；最后一份快照与原型逐个文件相同（`treediff.mjs`：3,408 个文件）。逐 Task 复现（A.5）的每个 Task 之后的树与快照逐个文件比较。

**切分**（第 3 节第 1 条）：设计 12 节 P2 的十四个草稿任务成为 14 个 Task。原型做的时候拆过两次：按编号取（草稿 5）先是一个 Task，拆成领域、用例、端口和存储（Task 5）与接口、矩阵和组合（Task 6）；起止日期（草稿 9）先是一个 Task，plan 里 1,755 行，拆成领域、用例和存储（Task 10）与接口、组合和矩阵（Task 11）；两次都由脚本把快照分开（`split56.sh`、`split1011.sh`），分开之后每个 Task 的门禁由逐 Task 复现照 plan 跑过（A.5）。

**Phase 的大小**（约 16 个任务、每个约 1,500 行以内）：14 个 Task，plan 11,973 行（`wc -l`），每个 Task 都在约 1,500 行以内：最大的是 Task 7（1,519 行：归档、恢复的用例、单元测试和矩阵的 `prepare`，预检之后又加了写闸门的两行和恢复的一步，比 1,500 行多 19 行），其次 Task 4（1,430 行）、Task 5（1,296 行）。各 Task 从它的标题到下一个标题（Task 14 到文件末尾）：1 782、2 729、3 859、4 1,430、5 1,296、6 582、7 1,519、8 346、9 1,126、10 1,147、11 800、12 241、13 731、14 168，Task 1 之前（开头的约束、脚本和文件表）175 行。手写的文件 138 个（新建 34 个、修改 104 个），生成物 11 个，文档 8 个（2.1），另有本提交对设计的修改（2.1）。切分时没有超出，没有停下。

### A.2 清扫之后的修改和加强

**清扫之后的修改**（变异按 Task 跑完之后复审时发现，改在拥有那个文件的 Task 的快照上，从那个 Task 起每一份快照都改，`p2tools/edits/` 的编辑说明逐条记下，受影响的变异重跑）：

- 说得比代码宽、或在任务结束时为假的句子（清扫 6、53、59）：`updateIssue` 的契约描述"值照 `createIssue` 的规则"写明除去日期的次序（修改按合并之后的值在锁下检查，第 3 节第 11 条；`T11-dates-desc`，生成的打包结果和 `schema.gen.ts` 随之，`server.gen.go` 不变）；`expectIssueDeleted` 的说明"I15、I17"改成"I15（I17 在 P3）"（`T13-fix-i17`）；端到端的 `changed` 的说明写明 `comment` 照 Plane、归档和恢复的两行除外（`T13-changed-doc`、`T13-changed-doc2`，第 3 节第 8 条）；连接测试（`TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`）说"每个写"，原来只有创建和修改：加上删除第二个工作项（`T4-txconn`）、归档和恢复第一个（`T7-txconn`），新加的变异 T4.20、T7.16、T7.17（删除的语句、归档的读回、归档读状态经连接池）只在它上被发现（清扫 8）。
- 数据分不出的性质（清扫 51、57；清扫 24：每条规则的每一半在组合一层有反例）：按编号取的组合测试原来只有一个工作区，标识不按工作区找的变异 T5.8 在 Task 6 的树上没被发现：加了工作区 beta 和它的项目 WEB、它的 1（`T6-scope`），T5.8 的层从矩阵换成这个测试；它原来也没有两个折叠相同、大写却不是其中任何一个的标识（T5.3 只在单元一层）：加了项目 İI、Iİ 和它们的 1，`ii-1` 是 404，`İi-1`、`iİ-1` 各找到自己的（`T6-fold-none`，样例文件的同一条）。日期的组合测试原来没有一步给它已有的日期（"什么都不改时不写"的 T10.7 只在单元一层）：加了这一步（`T11-same-dates`）；它的日期都在 10 月（不比较月份的 T9.4 只在单元一层）：只给开始日期的一步和开始晚于截止的创建改用 11 月 1 日（月份晚、日子早，`T11-dates-month`）。归档的盖戳测试原来在归档、恢复之后才建 Todo 里的工作项（把每个工作项都归档的 T7.15 分不出）：改为一开始就建，归档、恢复之后核对它的每一行（`T7-archive-other`）；它也没有归档已取消的工作项（T7.5 只在单元一层）：最后把它移到 Cancelled，bob 归档它（`T7-archive-cancelled`）。
- 种子要是规则允许的（清扫 35）：按编号取的已归档的工作项原来在 Backlog 里由数据库归档（Task 6 还没有归档的操作），规则到不了这个状态；改成在 Done 里、已完成、已归档（`T6-archived-done`）。
- 端口的回答对得上所问（清扫 22）：归档在锁下读项目的状态，工作项的状态不在其中时是错误（不是 409）；原来没有测试。加了 `TestArchivingAWorkItemWhoseStateIsNotItsProjectsFails`（`T7-foreign-state`）和它的变异 T7.18（当作不能归档的状态答 409），按性质只在单元一层（A.3）。
- 新语句的每个谓词（清扫 1、55）：原型的变异只有删除的负责人、标签两条语句的谓词和按编号取的三个（T4.6、T4.7、T5.8–T5.10）；补上其余 11 个：删除的五条语句的工作项和未删除（T4.12–T4.19）、`IssueByNumber` 的项目和编号（T5.14、T5.15）、`SetArchivedAt` 的 `id`（T7.15）。
- 没有被任何变异弄失败的新测试（清扫 45，`checkkills.py`）：原来有 7 个，各补一个变异：T3.1、T6.2（矩阵的格子指名它的列的工作项、它的标识）、T4.11（`DELETE` 不问用例就答 204）、T5.12、T5.13（`bootstrap` 的两个转换吞掉失败）、T9.12（`shared.DateOf` 取 UTC 的那一天）、T9.13（`Date` 的 JSON 是时刻）。
- 变异本身（清扫 56）：T2.1、T5.13 原来编译不过，是坏的变异而不是发现，改写成编译得过的同一个缺陷；T2.1 原来记了 I15 一层，I15 的项目"可见全部"，访客规则在那里不隐藏任何工作项，去掉这一层；契约的变异（T4.10、T6.1、T7.14）原来只跑 `make gen-go`，而 HTTP 的测试读 `api/dist` 的打包结果，`mut.py` 改为在副本里 `make gen`（含 Web 一侧）之后再跑；生成器和映射的变异（T9.9、T9.10）的 `make gen-go` 一层在 Task 9 的树上没有可拒绝的日期（那时契约里还没有），改为"Task 11 起"；T10.14（新工作项把截止日期存成开始日期）原来记在 `TestAWorkItemReadsAsStored` 上，它建的工作项没有日期，改为日期的组合测试和 I1；新加 T2.8（`getIssue` 判定 `issue.create`，P1b 评审第 6 节：访客让两条规则分得出，Task 3 起在矩阵被发现）。
- 文件的大小：`access/domain/decide_test.go` 加了创建者的表之后 407 行，`TestAllowCreatorAllowsAMemberWhoCreatedIt` 移进新的 `creator_test.go`（`T1-creator-file`，367 行）；`permission_matrix_test.go` 的种子说明重新折行（`T3-seed-doc`，406 行；基点已是 399 行，Global Constraints 把它列为例外）。

**清扫之后的加强**都在上面（数据、谓词、连接、端口）：每处改了测试或补上更高一层的检查，没有改产品代码。改过的测试在它的 Task 的树上先不带变异跑一次（全部通过），以它为一层的变异在它的 Task、"Task n 起"的和最终的树上重跑（`rerun-dates.sh`、`rerun-predicates.sh`、`rerun-conn.sh`、`rerun-foreign.sh`、`rerun-data.sh`），全部被发现。

**预检之后的修改**（2026-10-20：预检 `$M4TMP/p2-preflight/preflight.md` 的 M1、L1–L12 和控制者对它们的裁定；同样改在拥有那个文件的 Task 的快照上，从那个 Task 起每一份快照都改，`p2tools/edits/*-pf-*.txt`，`mkblocks.py` 重新生成全部块，547 个；没有改产品代码）：

- 写闸门（M1，清扫 9）：删除、归档、恢复进项目的写闸门原来没有测试钉住，跳过闸门的变异在每一层都活着。单元：`lock_test.go` 的 `TestEachWriteOnAWorkItemEntersItsProjectsGate`（预检写的样子；删除一行在 Task 4，用 `newUpdate` 的世界，归档、恢复两行在 Task 7，换成 `newArchive` 的世界；`T4-pf-gate`、`T7-pf-gate`）。组合（裁定第 1 条加的）：`gate_test.go` 的 `TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy`：在只让一个写进、不排队的 app 里，一个事务持 Web 的行，alice 的修改占着闸门的位置，删除（Task 4）、归档和恢复（Task 7）各立刻答 503 `server_busy`、`Retry-After: 1`；行放开之后她的修改和这三个写都照常。新变异 T4.21（删除不进闸门，预检的 PF.D5）、T7.19（归档、恢复不进闸门），各在单元和组合两层被发现：跳过闸门的写去等 Web 的行，过了锁超时才答 503（时间的断言失败），或排在 alice 的修改之后、5 秒内没有回答（锁超时对每次等锁分别计时）。
- 创建者不明的工作项（L1）：假存储把零值的创建者读成没有（`fakes_test.go` 的 `creatorOf`，文件 400 行），读者的表加"创建者不明的"一行（Task 2），`TestADecisionOnAWorkItemNamesItsCreator` 加创建者不明的工作项的删除（Task 4；`T2-pf-creator`、`T4-pf-creator`）。新变异 T2.9（创建者不明的工作项在 `AllowCreator` 上算调用者的，原来在每一层都活着）Task 4 起在单元一层被发现，T2.3 也在 `issue/app` 被发现；两个都按性质只在单元一层，到 P9（A.3）。
- 恢复不看状态组（L2，清扫 24）：`TestArchivingAWorkItemStampsItsRows` 的最后加一步：Cancelled 改到"进行中"组（另一个已取消的状态 Dropped 留住那一组），alice 恢复移到 Cancelled 之后归档的那个工作项（`T7-pf-restore`）。新变异 T7.20（恢复也检查状态组）在单元和组合两层被发现。
- 按编号取的数据（L3，清扫 51、57）：`TestAWorkItemIsFoundByItsIdentifier` 加 acme 的 ŞI 和建了又删掉的 Şİ、beta 唯一带点的 İŞ：`Şİ-1` 找到 ŞI 的，`iş-1` 在 beta 找到 İŞ 的（`T6-pf-identifiers`）。T5.9（已删除的项目也算）原来记作按性质只在存储一层，那个论证是错的：折叠让已删除的 Şİ 与 ŞI 相撞，而 Şİ 正是所问的大写；它现在 Task 6 起在组合一层被发现。新变异 T5.16（标识只按大写逐个字母比较，不折叠）在单元一层、Task 6 起也在组合一层被发现。
- 说得不对的句子（L4–L8、L12，清扫 6、30、35、59）：矩阵的种子说明（`T3-pf-seed-doc`）、plan Task 3、2.5、第 3 节第 15 条、清扫 35 和 M3 收尾交接的"访客·自己"（`T14-pf-docs`）写明种子经存储把访客的工作项记在他名下，判定只读创建者和他现在的角色；plan Task 4 和 2.6 不再说访客规则的两个表带上删除（带上删除的是 `TestADecisionOnAWorkItemNamesItsCreator`）；`expectIssueDeleted` 的说明（`T13-pf-deleted-doc`）和 M3 收尾交接不再说它核对更早的删除保持原样（那由按目录的删除测试核对）；2.9 加删除标签照样从已归档的工作项上摘下它；第 3 节第 9 条和清扫 30 加在项目里是访客的工作区管理员的一句；差异清单的"归档、恢复"一行（`T14-pf-docs`）的 Plane 一列加"恢复没归档的 404"，Nerve 一列加"动态的 `comment` 不同"（第 3 节第 8 条的条件）。
- 别的（L9–L11、裁定第 14、15 条）：设计 3.3 和 6.5 的 `ProjectGate`（第 3 节第 3 条）、设计 3.4 的"单个工作项"一句（404 先于 403）、第 3 节第 7 条的 `users` 一句、第 6 节的时间差一句、2.9 的矩阵两行用它们自己的名字。
- 工具：`mut.py` 的日志按层在变异里的序号命名，只跑几层（`MUT_ONLY`）时只换掉那几层的日志。

改过的测试先在带来它的 Task 的树上不带变异跑过（Task 2、4、6、7，全部通过）；以改过的测试为一层的变异和五个新变异在各自的 Task、"Task n 起"的和最终的树上重跑（`pf-rerun.py`：只跑改过的层，即 `issue/app` 的单元一层、`TestArchivingAWorkItemStampsItsRows`、`TestAWorkItemIsFoundByItsIdentifier`，和新变异的每一层；Pass A 39 次、Pass B 13 次、Pass C 49 次，这 101 次一共跑了 116 层），全部被发现。只改了说明的测试（矩阵的种子、`expectIssueDeleted`、写闸门的组合测试）不重跑。

### A.3 只在单元、存储、守卫一层被发现的变异

**只在单元一层被发现的（按性质，25 个）**：

- 端口的失败和对不上的回答（清扫 2、22）3 个：T5.12（`issueProjects.Identifiers` 把项目的存储的失败答成没有项目）、T5.13（`issueWorkspaces` 丢掉工作区的存储的失败）：同 P1a、P1b，组合出的 app 里真实的存储在这些读上不失败，`bootstrap` 的转换测试由失败的假存储核对失败照原样传出。T7.18（归档把对不上工作项的状态当作不能归档的状态，409）：经接口的工作项的状态总是它的项目的（创建、修改检查状态属于项目，状态不在项目之间移动，在用的状态删不掉：`project.state_in_use`），锁下读到的状态里总有它；只有假的存储答得出对不上的状态。
- 规则 1 个：T1.2（没有 `AllowCreator` 的规则也允许创建者）：今天没有一条规则分得出。带创建者的目标只有工作项的动作，`AllowCreator` 要创建者在项目里是管理员或成员，而 `issue.read`、`issue.update`、`issue.archive` 本来就允许这两个角色；规则表的单元测试逐格核对每条规则对创建者和别人的回答（`TestEveryRuleDecidesItsCells`）。
- 创建者不明的工作项 2 个：T2.3（访客规则把创建者不明的工作项给访客看）、T2.9（创建者不明的工作项在 `AllowCreator` 上算调用者的）：`created_by_id` 为空只在 P9 物理删除账户之后（第 3 节第 2 条），经接口建的工作项都有创建者。单元的假存储把零值的创建者读成没有（`creatorOf`），`issue/app` 的读者的表和 `TestADecisionOnAWorkItemNamesItsCreator` 各有一行（T2.3 也在 `issue/domain` 被发现）；P9 加组合一层的用例（第 5 节）。
- 次序的约定 1 个：T4.8（删除先删负责人的行、再删工作项的）。同 P1b 的 T5.11：工作项一侧的每个写者都先持项目 N，表的次序在它之下等不起来；全局的表序是约定（第 3 节第 6 条），由单元的调用次序钉住。
- 契约 4 个：T4.10、T6.1、T7.14、T8.6（契约不声明 `deleteIssue` 的 `project.archived`、`getIssueByIdentifier` 的 `workspace.not_found`、`archiveIssue` 的 `issue.not_archivable`、`updateIssue` 的 `issue.archived`）。`x-problem-codes` 由 HTTP 的测试经 `apitest.Main` 两个方向核对（每个答出的码都声明了，每个声明的码都有测试答出），驱动的是生成的路由和真实的处理函数，这条路上没有替身（同 P1a 的 T9.11）；组合一层和端到端不读契约的声明。
- 文案 3 个：T7.12、T8.4（页面没有 `issue.not_archivable`、`issue.archived` 的文案）：`error-messages.test.ts` 读的是真实的文案表，显示它们的页面在 P13、P14（同 P1b 的 T4.4）；中文的一份缺键由 `check:sync` 发现（T7.13、T8.5，守卫）。T8.7（`project.state_in_use` 不再说怎样才能删除状态）：回答的文字由 `project` 的 HTTP 测试按整个回答体核对（`stateInUseJSON`），真实的处理函数和错误的映射，没有替身。
- 矩阵的自检 2 个：T3.1、T6.2（矩阵的一格可以指名它的列的项目里任何种下的工作项、任何标识）。它们改的是测试的设施（`matrixViolations`），由喂给它违规的行的测试核对（`TestMatrixViolationsCatchesAnotherCasesWorkItem`、`TestMatrixViolationsCatchesAnotherCasesIdentifier`）；矩阵自己的格子都照规矩写，查不出。
- 日期的平台 9 个。平台的组件由测试直接驱动（真实的 `shared.Date`、`bodyshape`、`apitest`），没有替身；组合出的 app 只加它们的接线。各自：T9.1（`ParseDate` 读年份 0）：请求体的检查先于解码拒绝年份 0，经接口到不了 `ParseDate`。T9.2、T9.3（请求体的检查读年份 0、不解开转义）：Task 11 起在 `issue` 的 HTTP 测试（`TestDatesPassTheBodyCheckAsTheyDecode`：生成的路由、真实的 `bodyshape` 和生成的类型）上被发现，组合一层走的是同一条路。T9.5（问题按路径逐字节排序）：`bodyshape` 的测试有 `ids[2]`、`ids[10]`；组合一层的请求体没有十个以上同一个数组的问题。T9.6–T9.8（开放节点的数）：今天的契约里没有不限类型的节点（生成的 `bodyshape.gen.go` 里没有开放的对象、没有 `items` 的数组、不限类型的值），经接口到不了。T9.11（`apitest` 的请求体用例不把 `date` 算作检查的格式）：测试的设施，`TestBodyCasesOfDates` 核对它。T9.12（`DateOf` 取 UTC 的那一天）：存储交回的 `date` 是 UTC 的零点，两种取法相同。

原来只在单元一层、组合一层的数据本可以展示的四个（T5.3、T7.5、T9.4、T10.7）加了数据之后也在组合一层（A.2）；预检找出的两个同类的性质也加了数据，恢复不看状态组（T7.20）和标识按折叠比较（T5.16）都在组合一层被发现。写闸门（T4.21、T7.19）在单元和组合两层被发现，不在这里（A.2）。

**只在存储一层被发现的（按性质，4 个；都在真实的数据库上）**：

- T4.13（删除再删一次早已删除的工作项，盖掉它的戳）：经接口，已删除的工作项在预检就是 404，语句碰不到它。
- T4.17、T4.19（删除再删一次早已删除的关注行、动态）：P2 没有单独删除关注行（M7）或动态的操作，经接口一个未删除的工作项下没有更早删除的这两种行；存储测试种下它们。负责人、标签的同一种变异（T4.6、T4.15）在按目录的删除被发现（P1b 的修改去掉负责人、标签）。
- T5.10（编号也指已删除的工作项）：找到之后 `see` 照 `getIssue` 再读工作项，已删除的读不到，同样是 404。

T5.9（标识也在工作区已删除的项目里找）原来在这里，按性质的论证是错的：已删除的项目的标识与活着的项目的折叠相同、又正是所问的大写时，它取已删除的那个，答 404。预检之后的数据（ŞI 和已删除的 Şİ）在组合一层发现它（A.2）。

**只在守卫一层被发现的（3 个）**：T7.13、T8.5（中文的一份没有 `issue_not_archivable`、`issue_archived`）由 `check:sync` 发现（同 P1b 的 T4.5）；T9.10（`issue` 的映射把 `date` 留给 `openapi_types.Date`）：`make gen-go` 时生成器拒绝请求体里没有映射的 `date`（T9.9 的检查器），生成之前就失败，Task 11 起契约里有了日期才触发。

**只在组合一层被发现的（3 个）**：T4.20、T7.16、T7.17（删除的语句、归档的读回、归档读状态经连接池）：只有一个连接的池看得到语句不在事务的连接上（同 P1b 的 T7.1），单元的假实现没有连接。

**组合一层没有、只有端到端发现的**：没有。端到端发现的 32 个变异每个另在单元、存储或组合一层被发现（`e2ealso.py`：31 个在单元或存储一层，T7.10 在组合一层）。

### A.4 最终的原型、矩阵和交错

**最终的原型**（`gates.sh`，2026-10-20，预检之后的修改做完之后：七个门禁在最终的树上，`logs/amend-*.log`、`amend.summary`；2026-10-19 跑过三次，`final`、`final2`、`final3`，结果相同）：`make gen` 之后生成物没有差异；`make lint-go` 两段 `0 issues.`；`make test` 47 个 `ok`；`make lint-web`（关键词守卫 72 条规则、23 个例外、没有命中，54 个任务）；`make knip`；`make test-web`（17 个任务）；`make e2e` 123 个中 120 个通过，I1、I2、I15 在其中；失败的三个与 P2 无关：S3 因原型不是仓库的检出、构建没有提交号而失败（M3/P4b 的 F4），A4 的两个 firefox 测试因为机器上没有 Firefox（基点的副本上是同样的三个）。

**矩阵**（`matrixcount.sh`，`logs/matrix.log`，预检之后的修改做完之后；之前的一次相同）：`TestPermissionMatrix` 的 1,259 个叶子全部通过：模板的 `prepare` 一个，格子 1,258 个，其中工作项的操作 393 格（`getIssue` 20、`getIssueByIdentifier` 20、`createIssue` 80、`updateIssue` 136、`deleteIssue` 40、`archiveIssue` 57、`unarchiveIssue` 40），其余 865 格是 M3 的；顶层 1.53 秒。

**交错和竞争**（`race.sh`，`logs/race-amend.log`，预检之后的修改做完之后）：`bootstrap` 的 63 个交错、竞争、取锁形状和连接池的测试（`racenames.py`：文件名带 interleaving、race、locks 或 issue_pool 的每个测试：M3、P1a、P1b 的，和 P2 的交错 12、删除与修改、改过的交错 4、5 和取锁形状；写闸门的测试不在其中，它们的文件是 `gate_test.go`）各 `-race -count=5`，315 次顶层全部通过，459 秒，没有 40P01、没有死锁、没有数据竞争。修改之前在最终的树上（`logs/race-final.log`，444 秒）和 Task 12 的树上（`logs/race-T12.log`，448 秒）各跑过一次，同样 63 个测试、315 次顶层全部通过，同样没有。plan 的 Task 12 跑其中工作项一侧的 15 个（P2 的三个，P1b、P1a 的工作项的交错，`TestTwoWritesOnAProjectSerialize`，连接池的两个，取锁形状），逐 Task 复现里同样通过（90 秒，A.5）。

### A.5 逐 Task 复现

**逐 Task 复现**（本提交的 plan，预检之后的修改做完之后，2026-10-20；`replay-all.sh`、`replay.py`，日志在 `p2tools/replay-logs`、`logs/replay-all.log`，`replaysum.py`）：`7db78788` 的一份新的副本（`base-7db78788.tar`），本提交的 spec、plan 和设计照它们在 worktree 里的样子放进去，`pnpm install --frozen-lockfile`，`bin/golangci-lint` 取原型的（2.13.2；`make tools` 会下载它）。先在基点的另一份新的副本（复现存下的 `p2replay-snaps/T0`）上 `planapply.mjs check`：547 个块按顺序都对。按 plan 的顺序应用 14 个 Task 的 547 个块，照原样运行各 Task 的 103 个 `Run:` 命令（其中 10 个是提交之后的 `make gen-check`），一共 1,839 秒，没有重跑，没有起不来的容器。Task 1–12 之后 `make lint-go` 两段 `0 issues.`；Task 1–12 和 14 之后 `make test` 47 个 `ok`；`make gen-go`（Task 1、5、10）和 `make gen`（Task 2、4、6、7、8、11）之后，生成物的每个 SHA-256 和行数与 plan 的表相同，Task 9 的 `make gen` 之后 50 个生成物逐字节不变，十次提交之后的 gen-same 都没有差异；`make lint-web` 在 Task 1、2、4、6、7、8、11、13、14 通过（关键词守卫 72 条规则、23 个例外、没有命中），`make knip`、`make test-web` 在 Task 2、4、6、7、8、11、13 通过；Task 12 的 `-race -count=5`（工作项一侧的 15 个测试）90 秒，没有 40P01、死锁和数据竞争；`make e2e` 在 Task 2、11 是 122 个中 119 个通过，在 Task 13 是 123 个中 120 个通过（I1、I2、I15 在其中），失败的只有 S3（F4）和 A4 的两个 firefox 测试（机器上没有 Firefox）；Task 14 的 `doclinks.py` 没有断的链接。每个 Task 之后的树存成快照（`p2replay-snaps/T1`…`T14`），与 `p2snap` 的十四份逐个文件相同（`snapcmp.py`）。最后 `treediff` 比较复现的树与原型：3,410 个文件对 3,408 个，差的只有本提交的三份文档（spec 和 plan 只在复现里；设计差第 15 节的一行和预检之后改的 3.3、3.4、6.5 三处）。复现的副本不是 git 检出：`make gen-check` 由 gen-same 代替（`make gen` 前后生成物逐字节相同），`make lint-web` 的关键词守卫经 `kwreal.py` 列出文件。第一阶段的复现（538 个块，1,796 秒，结果相同）的树、快照和日志移在 `p2replay-phase1/`。

**复现之后**：plan 和设计与复现读的逐字节相同；spec 在复现读过之后只改了表头的两处（日期，上级文档加上 3.3）和本附录（复现读的是第一阶段的附录）。关键词守卫在 worktree 里对写好的三份文档再跑一次（`node tools/keywords.mjs`），没有命中；spec 和 plan 的相对链接（`doclinks.py`）都在。

### A.6 清扫和变异表

**六十类清扫**（P1b 的五十六类（P1b spec 附录 A.6）和 brief 加的四类（57–60）；每个变异在一份新的副本里（`p2mut/<id>`），Go 以外的文件（契约、故事、文案）也在副本里改，契约的变异在副本里 `make gen` 之后再跑；`mut.py` 在变异写的每一层各跑一次：单元（各模块的 `app`、`domain`、`adapter/http`，`access/domain`、`access/app`，`shared`，`platform/httpserver/bodyshape`、`apitest`，`server/tools` 的生成器，`bootstrap` 的转换和矩阵的自检，Web 的 `error-messages.test.ts`）、存储（各模块的 `adapter/postgres`）、组合（`bootstrap`）、端到端（单独运行的故事 I1、I2、I15）、守卫（`check:sync`、`make gen-go`）；每个变异在它的 Task 的快照上跑（Pass A），后面的 Task 才有的检查在那个 Task 的快照上跑（"（Task n 起）"，Pass B），全部的 Go 一层在最终的树上再跑一次（Pass C）；"层"是它在最终的树上被发现的每一层（端到端在 Task 13 的树上：Task 14 只改文档）；构建失败、崩溃不算发现，端到端的发现由 `e2ekills.py` 核对是断言失败，不是崩溃或超时）：

| 清扫 | 大小 | 结果 | 层 |
|---|---|---|---|
| 1 每个 SQL 谓词 | `delete_issue.sql` 的五条语句各两个谓词：工作项（`id` 或 `issue_id`）和未删除（T4.6、T4.7、T4.12–T4.19）；`IssueByNumber` 的项目、编号、未删除（T5.14、T5.15、T5.10）；`ProjectIdentifiers` 的工作区、未删除（T5.8、T5.9）；`SetArchivedAt` 的 `id`（T7.15）。每个都改成不限制（`OR TRUE`，保留 `sqlc.arg` 让生成的代码照样编译）或去掉 | 全部被发现，都在存储一层；工作项、项目、工作区的谓词也在组合一层（按目录的删除、按编号取、归档的盖戳）；T4.13、T4.17、T4.19（已删除的工作项、关注、动态再被盖戳）、T5.10 只在存储一层（按性质，A.3），T5.9 预检之后也在组合一层（A.2）。原型先只有 T4.6、T4.7、T5.8–T5.10，清扫时补上其余 11 个（A.2） | 存储；组合 |
| 2 端口调用的错误 | 用例吞掉端口的失败：`issueProjects.Identifiers`、`issueWorkspaces` 的转换（T5.12、T5.13）；归档把读到的、对不上工作项的状态当作拒绝（T7.18，A.2）；用例的失败表（`TestDeleteIssueStopsAtAFailingPort`、`TestArchiveIssueRefusals`、`TestGetIssueByIdentifierRefusals` 的每个端口） | T5.12、T5.13、T7.18 只在单元一层（按性质，A.3） | 单元 |
| 3 写不动的行 | 删除的存储测试比较别的工作项和更早删除的行的每一列；归档的存储测试比较另一个工作项；按目录的删除比较同一个项目另一个工作项下的每一行（含它自己的行）；归档的盖戳测试的 Todo 工作项从一开始就在（A.2）；交错 12 的动态只有过了锁的那个写的一行 | T4.12、T4.14、T4.16、T4.18 在按目录的删除，T7.15 在归档的盖戳测试 | 存储；组合 |
| 4 组合根的接线 | `bootstrap` 的 `issueWorkspaces`、`issueProjects.Identifiers`、`ProjectFacts` 的转换（T1.8、T1.9、T5.12、T5.13）；HTTP 的映射和路由（T4.11、T7.10、T7.11、T11.1–T11.4） | 都在组合一层或端到端；T5.12、T5.13 只在单元一层（A.3） | 单元；组合；端到端 |
| 5 安全性质在真实环境上 | 访客规则（T2.1、T2.2、T2.5、T2.6）、`AllowCreator`（T1.1、T1.3、T1.4、T2.4）、`OwnOnly` 按项目角色（T1.7）、`guest_view_all_features`（T1.5、T1.8、T1.9）、按编号取先判定工作区（T5.6、T5.7）、锁下再判定（T12.1） | 都在组合一层（矩阵、交错 4、5 的访客和删除一支、按编号取的组合测试）；T1.2（没有 `AllowCreator` 的规则也认创建者）只在单元一层：今天没有一条规则分得出（A.3）；创建者不明的工作项的两半（T2.3、T2.9）到 P9 只在单元一层（A.3） | 单元；组合；端到端 |
| 6 每句文档 | 每句说明、契约描述、故事标题对照代码；`prophecy.py` 列出每个 Task 加的往后看、计数的注释（19 行，逐行读过：说 P3、P4、P5 的都是设计的交接，"until"是条件） | 改了四处（A.2：`updateIssue` 的"值照 `createIssue` 的规则"、`expectIssueDeleted` 的"I15、I17"、`changed` 的"Plane 的 comment"、连接测试的"每个写"）；`TestAWorkItemReadsAsStored` 的"P1b 没有归档的操作"由 Task 7 照计划改写；预检又找出说得不对的几处（矩阵的种子说明、`expectIssueDeleted` 的说明、Task 4 的访客表一句、差异清单的一行，A.2） | 审阅 |
| 7 反例里没有随机 | P2 的测试 | 表都是切片；"没有的工作项""没有的状态、标签"用随机的 id，是不存在的 id，不是取样 | 审阅 |
| 8 决定所依赖的读在调用者的事务里 | 锁下的重读、判定、归档读的状态、读回都在事务的连接上；连接测试原来只有创建和修改，加上删除、归档、恢复（A.2）；按编号取是读，不进事务 | T4.20、T7.16、T7.17（语句经连接池）在组合一层（一个连接的池：`TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection`） | 组合 |
| 9 锁顺序在真实环境上 | 删除、归档、恢复最先锁项目（取锁形状的三行：`deleteIssue`、`unarchiveIssue` 等 `issues`，`archiveIssue` 等 `states`）；删除的表序（T4.8）；删除在项目的锁之外写（T4.9）；删除、归档、恢复进项目的写闸门（T4.21、T7.19：单元的调用次序，组合的闸门满时立刻 503；预检之后加的，A.2） | T4.9、T4.21、T7.19 在组合一层；T4.8 只在单元一层（按性质，A.3） | 单元；组合 |
| 10 承重的种子行有前提 | 矩阵的 `ownIssues`、`guestViewAll`（`seededIssues` 读回每个工作项的编号、创建者和访客规则读的事实）；按目录的删除的工作项（每张表都有一行、负责人和标签各有一行更早删除的：缺了就 `t.Fatalf`）；交错的 `bobsWorkItem`、`doneWorkItem`；`wroteNothing` 先核对准备的行数 | 都读回核对 | 审阅；组合 |
| 11 每条拒绝路径 | 预检（404、403）、锁下（404、403、409 `project.archived`、`issue.archived`、`issue.not_archivable`、日期的 422）、按编号取（`workspace.not_found`、`issue.not_found`、403） | 单元的拒绝表；矩阵的 P2 行（已归档的项目、已归档的工作项、Todo 里的、日期的次序不对）；交错 4、5、12、删除与修改；按编号取的组合测试；I1、I2、I15 | 审阅 |
| 12 与结束、删除的竞争（组合） | 删除与修改（两种顺序）；归档与移动（交错 12，两种顺序）；等锁期间被移出、降级的创建者的删除（交错 4、5）；删除工作项与删除项目、工作区不另写（第 3 节第 16 条） | 都答对；T1.1、T1.4、T2.4、T4.4、T7.4、T8.1、T12.1 在这些交错里被发现 | 组合 |
| 13 锁强度在组合一层看得到 | 取锁形状的测试的 `waits`、`mode`（第 3 节第 13 条） | T4.9 在组合一层 | 组合 |
| 14 每个"由谁"可以失败 | 删除的语句由 bob 写在 alice 写过的行上（存储）；按目录的删除由 alice 删 bob 写的行（`noRowLastWrittenBy`）；归档由 bob、恢复由 alice（盖戳测试）；日期的动态是 bob 的 | T4.5、T7.9 在存储和组合一层，T4.5 也在 I15 | 存储；组合；端到端 |
| 15 标题的每个说法都有展示 | I15 的标题逐句；组合测试的说明逐句 | 每个说法一步 | 审阅 |
| 16 单元假对象的回答顺序 | 假的项目标识按给的次序回答，存储按 `id`；`MatchIdentifier` 不靠次序（两个以上时取等于大写的那个） | T5.4 在组合一层 | 审阅；组合 |
| 17 判定之前的代价有界 | 删除、归档、恢复的预检：一次不加锁的读、一次判定；按编号取：工作区一次、判定一次、项目的标识一次、编号一次、读一次 | 有界 | 审阅 |
| 18 契约描述只说代码做的 | 四个新操作的描述；`updateIssue` 的日期一句（A.2）；`x-problem-codes`（T4.10、T6.1、T7.14、T8.6） | 码由 `apitest.Main` 两个方向核对，只在单元一层（按性质，A.3） | 单元 |
| 19 每个新的存储方法有失败测试 | 五条删除语句（取消的 `context`）、`SetArchivedAt`（`TestAFailedUpdateStatementIsAnError`）、`IssueByNumber`（`TestAFailedIssueStatementIsAnError`）、`ProjectIdentifiers`（`TestAFailedReadIsAnErrorNotAnAnswer`） | 都有 | 存储 |
| 20 相关谓词的集合形式、"别的"算进自己 | 删除的语句一次按 `issue_id` 写它的行；`ProjectIdentifiers` 一次读工作区的全部项目 | 存储测试的别的工作项、别的工作区 | 存储 |
| 21 拒绝和失败钉住第一个 `*shared.Error` | 单元的拒绝表用 `errors.Is` 比较码；HTTP 的拒绝表按码；矩阵按格子的码 | 审阅 | 审阅 |
| 22 端口的回答对得上所问 | 归档在锁下读的项目的状态里没有工作项的状态：是错误，不是 409（原来没有测试，加了 `TestArchivingAWorkItemWhoseStateIsNotItsProjectsFails`，A.2）；`issueProjects` 问的是用例给的工作区（`TestIssueProjectsConvertsProjectIdentifiers`）；按编号取的 `IssueByNumber` 问的是找到的项目（T5.14） | T7.18 只在单元一层（经接口的工作项的状态总是它的项目的，A.3）；T5.14 在存储和组合一层 | 单元；存储；组合 |
| 23 组合的夹具跨第二个工作区、第二个项目 | 按编号取加了 beta 工作区和它的 WEB（A.2）；矩阵的各项目；按目录的删除的另一个工作项 | T5.8 加了 beta 之后在组合一层（A.2） | 组合 |
| 24 每条规则的每一半在组合一层有反例 | `issue.read` 加访客、`issue.delete`（管理员和创建者）、`issue.archive`（管理员、成员）、`issue.search`（三个角色）；归档的状态组（已完成、已取消），恢复不看状态组；合并之后的日期两个方向（T10.3–T10.6） | 都在矩阵或组合一层；已取消的一半（T7.5）、折叠相同而大写不是其中任何一个的一半（T5.3）原来只在单元一层，组合的测试加了这两种数据；恢复不看状态组（T7.20）是预检之后加的（A.2） | 单元；组合；端到端 |
| 25 判定和执行在同一把锁下 | 删除、归档的时刻在锁下读（T4.4）；锁下再判定（T12.1、T2.5） | 都在组合一层 | 单元；组合 |
| 26 "不变"用不会碰巧得出的值 | 删除的存储测试的行一小时前由 alice 删、再由 bob 删；归档的存储测试的另一个工作项；日期从 12 日到 19 日（只看请求和合并之后的结论不同） | 逐列比较 | 存储；组合 |
| 27 故事里的"不动"在动作之前读、之后核对 | I15 的再归档、再恢复、409、403 各在前后核对（`expectIssueArchival`）；I2 的日期的 422 之前一步的 `expectIssueUpdated` | 都有 | 审阅 |
| 28 只在端到端被发现的性质另有 Go 的测试 | 变异表里只在端到端失败的 | 没有：每个有端到端一层的变异另在单元、存储或组合一层被发现（`e2ealso.py`） | 审阅 |
| 29 每个能等锁的等待都有期限 | P2 的写都经 `Locks.run`：闸门的 `max_wait`、事务里每次等锁的 `issue.write_lock_timeout`（P1b）；测试一侧：交错的 context、探测 | 没有新的等待 | 审阅 |
| 30 每个输出、问题的细节和补救对每个收到它的调用者都成立 | `issue.not_archivable`（先移到已完成或已取消）、`issue.archived`（先恢复）、`project.state_in_use`（先恢复、再移走）：收到它们的都是能做这个动作的人（访客得到 403、404），补救对他们都做得到（第 3 节第 9 条）；在项目里是访客的工作区管理员能删除状态，却看不到别人的工作项：补救要他先把自己在项目里的角色改为成员或管理员（M3），这一点照样成立（预检）；已归档的项目的状态照样能删除（M3 设计 3.19），补救的恢复和移动在那里却先答 `project.archived`：它的文字补上它们之前的一步，这一点照样成立（执行中由实现者发现） | 成立 | 审阅 |
| 31 检查和执行之间的 gate 在检查的事务之外 | 交错照用 P1b 的 `inBothOrders`、`serializeOn` | 没有新写 | 组合 |
| 32 每个探针有反例 | 没有新写的探针（`pgtest.WaitForLockWaitBehind`） | 没有新写 | 审阅 |
| 33 goroutine 上不调用 `t.Fatal` | 交错的写只发请求、送回回答 | 照 P1b | 审阅 |
| 34 "没有替身"、"完成"的说法全包 grep | `issueWrites` 与契约中工作项一侧的写一一对应（删除、归档、恢复各一行）；矩阵覆盖每个操作（`TestThePermissionMatrixCoversEveryOperation`）；`apitest.Main` 两个方向；连接测试的"每个写"（A.2） | 都核对 | 单元；组合 |
| 35 "由 X 写"的种子是规则允许的 | 矩阵的 `ownIssues` 经存储写下，创建者是各列的账户：访客不能创建（`createIssue`），种子经存储把访客的工作项记在他名下，判定只读创建者和他现在的角色，不读它怎样来的（预检之后改的说法：原来说他在成为访客之前作为成员建，种子不是这样写的）；离开的成员的在他离开之前建；`guestViewAll` 由 acme 的管理员经项目的存储设；按编号取的已归档的工作项原来在 Backlog 里由事务归档（Task 6 还没有归档的操作），改成在 Done 里已完成、已归档（A.2）；矩阵的 `prepare` 照同样的样子写 | 规则到得了的状态 | 审阅 |
| 36 别的操作或两步到达同样的结果 | 再归档、再恢复（不写）；删除已归档的工作项（照常，I15、矩阵） | 都有 | 组合；端到端 |
| 37 归档项目、删除的工作区、停用的调用者 | 已归档的项目 409（矩阵的 P2 行，各写一行，已归档的工作项在已归档的项目里也是项目的 409）；按编号取的不存在的工作区；停用的调用者由认证在请求之前拒绝 | T4.1、T7.1、T8.3 在组合一层 | 组合 |
| 38 故事的每个拒绝之前先读 | I15 的 409、403；I2 的日期的 422、已归档的 409 | 之前一步的核对，之后再核对 | 审阅 |
| 39 测试里的排序与排序规则无关 | 按目录的删除按 `r::text` 比较；`archivalRows` 按 `id` | 与排序规则无关 | 审阅 |
| 40 每个交错的结果写明每个结束由谁 | 删除与修改：删除由 alice，bob 的一行动态由 alice 删；交错 12：动态是 bob 的或 alice 的 | 结果读出写者 | 审阅 |
| 41 每个写多行的语句与同一些行的其他写者 | 四条按 `issue_id` 的删除语句（第 3 节第 7 条） | 都在项目 N 之下 | 审阅 |
| 42 锁语句之后的写只写锁住的行 | 删除、归档写的是锁下重读、确认仍在锁住的项目里的工作项 | P1b 的"工作项不在了"同一条路 | 组合 |
| 43 每个"由 X"可以失败 | 同清扫 14 | 同上 | 存储；组合；端到端 |
| 44 "保留"、"还有"读状态 | 更早删除的行保留它的删除（T4.6、T4.13、T4.15、T4.17、T4.19）；已归档的工作项照样读、照样删除 | 都被发现 | 存储；组合 |
| 45 没有不会失败的断言 | P2 的断言 | 原型的每个新的 Go 测试至少被一个变异弄失败（`checkkills.py`）：原来有 7 个没有，补上 7 个变异（A.2） | 审阅 |
| 46 测试一侧的等待都有期限 | 种子、读回、持有事务时的语句 | 都在 `pgtest.Soon`、context 或测试自己的期限上 | 审阅 |
| 47 连续的探测在同一张表上要指名 | 每个交错一次探测 | 没有连续的同表探测 | 审阅 |
| 48 堆序或索引序的夹具有前提 | 删除、按编号取的查询没有次序；`ProjectIdentifiers` 按 `id`（`MatchIdentifier` 不靠它） | 没有依赖堆序的断言 | 审阅 |
| 49 组合的夹具看得到过宽的写 | 按目录的删除的另一个工作项；归档的盖戳测试的 Todo 工作项（A.2） | T4.12、T4.14、T4.16、T4.18、T7.15 在组合一层 | 组合 |
| 50 每句注释和文档对照它说的代码 | 每个 Task 末尾 | 见清扫 6、15、59 | 审阅 |
| 51 数据分不出性质 | 每个谓词、次序、过滤的测试数据里有变异会错拿的行 | 按编号取原来只有一个工作区（T5.8 分不出），加了 beta；也没有两个折叠相同、大写却不是其中任何一个的标识（T5.3），加了 İI、Iİ；日期原来没有一步给它已有的日期（T10.7），加了一步，日期都在一个月里（T9.4），改用 11 月 1 日；归档的组合测试原来没有别的工作项（T7.15），Todo 的工作项改为一开始就建，也没有已取消的（T7.5），最后归档它（都在 A.2）；预检之后：按编号取加了 ŞI 和已删除的 Şİ（T5.9）、beta 唯一带点的 İŞ（T5.16），归档的盖戳测试加了状态组改了之后的恢复（T7.20）（A.2） | 存储；组合 |
| 52 复制的测试辅助 | 扩展已有的：`issueWorld`（`newWorkItemBy`、`doneWorkItem`）、`itemWrite`、`inBothOrders`、`wroteNothing`、`issueWrites`、矩阵的帮手、`changed`、`byId` | 没有重写的辅助；`expectIssueDeleted` 照删除项目的目录的写法 | 审阅 |
| 53 任务结束时为假的注释和文档 | `prophecy.py` 列出的每一行 | 见清扫 6 | 审阅 |
| 54 文案给出没人能做的补救 | `issue.not_archivable`、`issue.archived`、`project.state_in_use` 的补救 | 见清扫 30 | 审阅 |
| 55 新的存储查询的谓词没有钉住 | P2 的每个 `WHERE` | 每个有一个存储测试的变异（清扫 1） | 存储 |
| 56 只从后面的 Task 才被发现的变异；容器起不来不算 | "（Task n 起）"的行在那个 Task 的树上跑；构建失败、崩溃不算发现（`mut.py`），端到端的发现核对是断言失败（`e2ekills.py`） | 见变异表和各 Task 的表；两个原来编译不过的变异改写（T2.1、T5.13，A.2） | 审阅 |
| 57 数据分不出性质（brief） | 同清扫 51 | 同上 | 存储；组合 |
| 58 复制的测试帮手（brief：扩展，不重写） | 同清扫 52 | 同上 | 审阅 |
| 59 比代码宽的句子（brief） | 每句说明、描述、标题 | 改了四处（清扫 6，A.2）：说的比代码宽的三句（`updateIssue` 的规则一句、`changed` 的 `comment`、连接测试的"每个写"），一句在任务结束时为假（I17 在 P3）；预检之后又改了几处（A.2） | 审阅 |
| 60 计时按条件（brief） | 交错按探测到的等锁、请求的回答；盖戳测试按请求前后的时刻比较；没有固定的睡眠 | 没有按固定时间的等待 | 审阅 |

**变异表（按缺陷类别）**：变异的定义在 `$M4TMP/p2tools/mutants_p2.py`（119 个，按 Task 分在 `mutants_p2_a.py`（Task 1–4）、`mutants_p2_b.py`（Task 5–8）、`mutants_p2_c.py`（Task 9–12），层在 `mutants_common.py`；中文描述在 `mutants_zh_p2.py`），结果在 `mut-results-a.json`（各自的 Task 的树）、`mut-results-b.json`（"Task n 起"）、`mut-results-c.json`（最终的树），每次运行的日志在 `mut-logs-a`、`mut-logs-b/T<n>`、`mut-logs-c`；清扫之后的修改（A.2）改过的测试，以它为一层的变异在改过的快照上重跑，结果按层并进同样的文件（预检之后的同样，`pf-rerun.py`），表从它们写出（`classtable.py`）。119 个变异都被发现，没有等价的；跑过而没有发现的层没有。在最终的原型上（端到端一层在 Task 13 的树上），单元一层的检查发现 90 个，组合 84 个，端到端（I1、I2、I15）32 个，存储 22 个，守卫 4 个（一个变异可以在几层被发现）。只有一层发现的：单元 25 个、存储 4 个、守卫 3 个（都在 A.3），组合 3 个（T4.20、T7.16、T7.17），端到端没有。"（Task n 起）"的：Task 2 起 1 个、Task 3 起 6 个、Task 4 起 7 个、Task 6 起 13 个、Task 7 起 1 个、Task 11 起 18 个、Task 12 起 6 个、Task 13 起 32 个（端到端）。端到端的 33 次发现（T8.1 在 I2 和 I15 各一次）和 `error-messages.test.ts` 的 2 次由 `e2ekills.py` 核对都是断言失败。原型上 P2 新加的 51 个 Go 测试每个至少被一个变异弄失败（`checkkills.py`）。

| 缺陷类别 | 变异 | 结果 | 失败的检查 | 层 |
|---|---|---|---|---|
| 访客规则（3.4：`OwnOnly`、404 先于 403、预检和锁下各一次） | T1.5、T1.6、T1.7、T2.1、T2.2、T2.3、T2.5、T2.6 | 8/8 | `TestTheGrantCarriesTheRoles`、`TestARefusalCarriesTheGrant`、`TestPermissionMatrix`、I15、`TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`、`TestGetIssueByIdentifierRefusals`、`TestTheGuestRuleHidesAnothersWorkItemFromAReader`、等 3 个 | 单元；组合；端到端 |
| `AllowCreator`（3.4：创建者仍是有效成员、角色不低于成员） | T1.1、T1.2、T1.3、T1.4、T2.4、T2.9 | 6/6 | `TestAllowCreatorAllowsAMemberWhoCreatedIt`、`TestEveryRuleDecidesItsCells`、`TestPermissionMatrix`、`TestACallerRemovedOrDemotedWhileHisWriteWaitsIsRefused`、`TestAuthorizeKnowsTheCreator`、`TestADecisionOnAWorkItemNamesItsCreator` | 单元；组合 |
| `guest_view_all_features` 的读和它进判定 | T1.8、T1.9 | 2/2 | `TestAuthorizeReadsTheTargetsProject`、`TestPermissionMatrix`、I15、`TestProjectFacts` | 单元；存储；组合；端到端 |
| 规则表的行，归档的状态组（3.4、3.8） | T1.10、T1.11、T2.7、T5.11、T7.4、T7.5、T7.20 | 7/7 | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix`、I15、`TestArchivable`、`TestArchiveIssueRefusals`、`TestArchivingAWorkItemAndMovingItSerialize`、`TestArchivingAWorkItemStampsItsRows`、等 1 个 | 单元；组合；端到端 |
| 清扫 5、25：预检和锁下的判定、判定的动作 | T2.8、T4.1、T5.7、T7.1、T8.1、T12.1 | 6/6 | `TestGetIssueReadsTheWorkItem`、`TestGetIssueRefusals`、`TestTheGuestRuleHidesAnothersWorkItemFromAReader`、`TestPermissionMatrix`、`TestDeleteIssueRefusals`、`TestGetIssueByIdentifierReadsTheWorkItem`、`TestGetIssueByIdentifierRefusals`、等 7 个 | 单元；组合；端到端 |
| 拒绝的次序（第 3 节第 5 条）、表序 | T4.8、T7.2、T8.2、T8.3 | 4/4 | `TestDeleteIssueDeletesTheWorkItemAndItsRows`、`TestArchiveIssueRefusals`、`TestPermissionMatrix`、`TestUpdateIssueRefusesUnderTheLock` | 单元；组合 |
| 删除工作项的连带（3.6） | T4.2、T4.3 | 2/2 | `TestDeleteIssueDeletesTheWorkItemAndItsRows`、`TestDeleteIssueStopsAtAFailingPort`、`TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`、I15 | 单元；组合；端到端 |
| 清扫 1、55：新语句的每个谓词 | T4.6、T4.7、T4.12、T4.13、T4.14、T4.15、T4.16、T4.17、T4.18、T4.19、T5.14、T5.15、T7.15 | 13/13 | `TestTheStatementsOfADeletedWorkItemDeleteItsRows`、`TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`、`TestIssueByNumber`、`TestAWorkItemIsFoundByItsIdentifier`、`TestArchivingAWorkItemWritesItsRow`、`TestArchivingAWorkItemStampsItsRows` | 存储；组合 |
| 清扫 14：由谁、何时 | T4.5、T7.6、T7.9 | 3/3 | `TestTheStatementsOfADeletedWorkItemDeleteItsRows`、`TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`、I15、`TestArchiveIssueArchivesTheWorkItem`、`TestArchivingAWorkItemStampsItsRows`、`TestArchivingAWorkItemWritesItsRow` | 单元；存储；组合；端到端 |
| 清扫 25：时刻在锁下读 | T4.4 | 1/1 | `TestDeleteIssueDeletesTheWorkItemAndItsRows`、`TestDeleteIssueRefusals`、`TestEachWriteOnAWorkItemEntersItsProjectsGate`、`TestDeletingAWorkItemAndChangingItSerialize` | 单元；组合 |
| 清扫 9：写在项目的锁下 | T4.9 | 1/1 | `TestDeleteIssueDeletesTheWorkItemAndItsRows`、`TestDeleteIssueRefusals`、`TestEachWriteOnAWorkItemEntersItsProjectsGate`、`TestEachIssueWriteLocksItsProjectFirst` | 单元；组合 |
| 清扫 9：写进项目的写闸门（3.5、11.11） | T4.21、T7.19 | 2/2 | `TestEachWriteOnAWorkItemEntersItsProjectsGate`、`TestEachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy` | 单元；组合 |
| 清扫 8：语句在事务的连接上 | T4.20、T7.16、T7.17 | 3/3 | `TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection` | 组合 |
| 按编号取的范围（工作区、已删除的） | T5.6、T5.8、T5.9、T5.10 | 4/4 | `TestGetIssueByIdentifierReadsTheWorkItem`、`TestGetIssueByIdentifierRefusals`、`TestAWorkItemIsFoundByItsIdentifier`、`TestPermissionMatrix`、`TestProjectIdentifiers`、`TestIssueByNumber` | 单元；存储；组合 |
| 标识的折叠（3.7） | T5.1、T5.2、T5.3、T5.4、T5.16 | 5/5 | `TestMatchIdentifierFindsWhatTheSamplesSay`、`TestAWorkItemIsFoundByItsIdentifier` | 单元；组合 |
| 标识的解析（3.7） | T5.5 | 1/1 | `TestParseIdentifier`、`TestAWorkItemIsFoundByItsIdentifier` | 单元；组合 |
| 什么都不改时不写 | T7.3、T10.7、T10.8 | 3/3 | `TestArchivingAsItIsWritesNothing`、`TestPermissionMatrix`、`TestArchivingAWorkItemStampsItsRows`、I15、`TestChangesOf`、`TestDatesAreKeptInOrder`、I2 | 单元；组合；端到端 |
| 动态的第 8、9、14、16 行（3.13） | T7.7、T7.8、T10.10、T10.11 | 4/4 | `TestUnarchiveIssueUnarchivesTheWorkItem`、`TestArchivingAWorkItemStampsItsRows`、I15、`TestArchiveActivity`、`TestUpdateActivity`、I2 | 单元；组合；端到端 |
| 合并之后的日期（第 3 节第 11 条） | T10.1、T10.2、T10.3、T10.4、T10.5、T10.6 | 6/6 | `TestCheckNewIssueReportsEveryProblem`、`TestDatesAreKeptInOrder`、I1、`TestDatesOutOfOrder`、`TestCheckNewIssueAccepts`、`TestUpdateIssueHoldsTheMergedDatesInOrder`、I2 | 单元；组合；端到端 |
| 改了的日期写进工作项 | T10.9 | 1/1 | `TestUpdateIssueWritesTheDates`、`TestDatesAreKeptInOrder`、I2 | 单元；组合；端到端 |
| 日期的两列（存储） | T10.12、T10.13、T10.14 | 3/3 | `TestUpdatingAWorkItemWritesItsRows`、`TestAWorkItemReadsAsStored`、`TestDatesAreKeptInOrder`、I2、`TestCreatingAWorkItemWritesItsRows`、I1 | 存储；组合；端到端 |
| `shared.Date` 和请求体的日期 | T9.1、T9.2、T9.3、T9.4、T9.12、T9.13 | 6/6 | `TestParseDate`、`TestFormatDate`、`TestDatesPassTheBodyCheckAsTheyDecode`、`TestCheckCostsAboutTheBody`、`TestDateCompare`、`TestDatesAreKeptInOrder`、`TestDateAndTime`、等 1 个 | 单元；组合 |
| 问题的路径的次序 | T9.5 | 1/1 | `TestCheck`、`TestComparePaths` | 单元 |
| 不限类型的节点里的数 | T9.6、T9.7、T9.8 | 3/3 | `TestCheck`、`TestCheckCostsAboutTheBody` | 单元 |
| 生成器的检查器 | T9.9 | 1/1 | `TestGenerateMatchesTheGoldenFile`、`TestGenerateIsDeterministic`、`TestGenerateAcceptsTheStringsItCanCheck`、`make gen-go` | 单元；守卫 |
| 五个模块的类型映射 | T9.10 | 1/1 | `make gen-go` | 守卫 |
| `apitest` 的请求体用例 | T9.11 | 1/1 | `TestBodyCasesOfDates` | 单元 |
| HTTP 的映射和回答 | T4.11、T7.11、T11.1、T11.2、T11.3、T11.4 | 6/6 | `TestDeleteIssue`、`TestDeleteIssueRefusals`、`TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt`、I15、`TestArchivingAsItIsWritesNothing`、`TestPermissionMatrix`、`TestUpdateIssue`、等 8 个 | 单元；组合；端到端 |
| 清扫 18：契约声明的码 | T4.10、T6.1、T7.14、T8.6 | 4/4 | `TestDeleteIssueRefusals`、`TestGetIssueByIdentifierRefusals`、`TestArchiveIssueRefusals`、`TestUpdateIssueRefusals` | 单元 |
| 清扫 4：路由的接线 | T7.10 | 1/1 | `TestPermissionMatrix`、`TestArchivingAWorkItemStampsItsRows`、I15 | 组合；端到端 |
| 清扫 2、22：失败的传递，对不上所问的回答 | T5.12、T5.13、T7.18 | 3/3 | `TestIssueProjectsConvertsProjectIdentifiers`、`TestIssueWorkspacesConvertsWorkspacesAnswer`、`TestArchivingAWorkItemWhoseStateIsNotItsProjectsFails` | 单元 |
| 错误码的文案和补救（约束 4） | T7.12、T7.13、T8.4、T8.5、T8.7 | 5/5 | `error-messages.test.ts`、`check:sync`、`TestDeleteState` | 单元；守卫 |
| 矩阵的格子指名它的列的工作项 | T3.1、T6.2 | 2/2 | `TestMatrixViolationsCatchesAnotherCasesWorkItem`、`TestMatrixViolationsCatchesAnotherCasesIdentifier` | 单元 |
