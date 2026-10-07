# M3/P7b 标签：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M3/P7b `labels` |
| 日期 | 2026-10-06 |
| 状态 | 已完成（[评审记录](../reviews/P7b-labels-review.md)）：执行中的改动按评审记录第 3–5 节改入（2026-10-07）。第 3 节第 1、2、3、11 条由控制者裁定 B1–B4（2026-10-06，B2 不取第 2 条原来的做法）；预检之后的修订一轮（`1b7b8966`）。拆分照负责人 2026-10-05 的裁定与控制者的裁定 S1–S8（`p7-split-rulings.md`） |
| 上级文档 | [M3 设计文档](../M3-design.md) 第 2（P7、W3、P4）、3.4、3.6（加锁表，约定一、二、三、五及其例外）、3.16、3.19、3.20（P7b 两行）、4.10、4.11、5.1–5.3、6.7、9.2、9.3（交错 11）、11.5、12（P7b 与约束 1–4）、13.1 节 |
| 前置交接 | [P7a spec](P7a-states.md) 第 5 节 P7b 一行、[P7a review](../reviews/P7a-states-review.md) 第 6 节（共用路径、竞争和阶梯、矩阵、假实现、已归档的小表、第一次运行的重放、踩过的十个坑）；[P4a review](../reviews/P4a-projects-review.md) 第 6 节（标签进 `DeleteWorkspaceProjects`，W3、P4 断言标签）；[P4b review](../reviews/P4b-project-members-review.md) 第 6 节（每个写经 `Locks`，手工维护的表，`keysTo`，`expectProjectDeleted` 只数之前未删除的行）；[P5b review](../reviews/P5b-project-memberships-review.md) 第 6 节（标签是共用路径的又一种行）；[P6 review](../reviews/P6-deactivation-review.md) 第 6 节（`deleteLabel` 连带子标签与约定五的例外，裁定 S7 的 O3）。落点见第 3 节第 13 条 |
| 计划 | [P7b plan](../plans/P7b-labels.md) |

本 spec 只写 M3 设计交给 P7b 决定的东西：名字、签名、SQL、测试名，以及原型证明了什么。规则本身以 M3 设计为准，这里引用节号，不重述。P7b 依赖 P7a（`0deb8c34` 的 `main`）。

P7b 是评审敏感的一段（M3 设计 12 节约束 3）：标签的两层在并发下由项目行的锁保证（交错 11），新表进入删除项目和删除工作区的连带，删除父标签连带子标签是一条写多行的语句（约定五）。附录 A 的五十类清扫逐个核对每个这样的测试在它所说的性质去掉之后失败，并记下失败在哪一层（单元、存储、组合、端到端）；只在单元一层失败的安全或加锁的性质算缺口，按性质只能在单元一层的逐条写在第 3 节和附录 A。

P7b 的原型从 P7 的架构子任务第一次运行的标签部分（`p7snap/T7`→`T8`→`T9`→`p7proto` 的差异：表和连带、领域和存储、`createLabel` 和 `listLabels` 的用例）接过来，放在 P7a 合并之上，照裁定 S3 重新切成十一个 Task，逐条复审、清扫之后改在根上；接过来的、手工合并的、改了的，见附录 A 的"对第一次运行的复审"。

## 1. 目标

按 M3 设计 12 节 P7b 和 3.16：`labels` 表和它的两个连带，标签的全部操作和 3.16 的规则。具体是：

- 迁移 `00014_project_labels.sql`（Plane 15 列保留 12 列，4.10）和运行时的授权；删除项目、删除工作区的连带加标签一步，在 `deleteProjects` 的最后（3.6、11.5）；
- 标签的规则（`project/domain/label.go`）：名称 1–255 个字符、不空白、不含 NUL；颜色至多 255 个字符、可以为空；`SortOrderAfter`（项目未删除的标签的最大值加 10000，没有时 65535）；`CheckParent`（两层：父标签是同一项目、未删除、没有父标签的标签；有子标签的标签不能有父标签；不能做自己的父标签；各 422 `parent_id` `not_allowed`）；两个标签码；
- 标签的存储：七条语句（`queries/labels.sql`）和连带的一条（`DeleteLabels`），每条的存储测试每个谓词各由一行决定，每个方法有失败测试；名称在项目内唯一、不分大小写，由部分唯一索引 `labels_project_id_name_key ON (project_id, lower(name))` 保证，冲突按索引的确切名字映射为 409 `project.label_name_taken`；
- 四个操作：`listLabels`（父子一起，按 `sort_order`、再按 id；已归档的项目照常列出）、`createLabel`（经 `lockAndDecide`）、`updateLabel`、`deleteLabel`（经共用路径 `lockRowAndDecide`，`labelWrite` 构造它的 `rowWrite`）；`deleteLabel` 一条语句删除这个标签和它下面的标签；
- 每个标签的写：工作区 `FOR SHARE` → 项目 `FOR NO KEY UPDATE` → 判定 → 层级的检查 → 时钟 → 写；按行寻址的两个写锁下重读确认仍属那个项目，否则答 `project.label_not_found`；最先锁工作区的测试里一行，按行寻址的第一步探测它的标签行；
- 矩阵：每个矩阵项目种下三个标签（`matrixLabels`），四个操作 14 行，已归档项目的小表随每个操作；
- 并发：交错 11（两个层级改动）和标签的其余六对（建子标签与把父标签放到别处下、同名不分大小写的两个创建、同一标签的两次移动、删除父标签与往它下面放标签或建标签——约定五的探测、两个都成功的创建），两种顺序；按行寻址的竞争和锁强度推广到标签的行；每个标签的写只写它的行；`-count=5 -race`，没有 40P01；
- 故事 P7 的接口版本；W2、W3、P4 的标签（P4a 的裁定 S4）；`expectProjectDeleted` 只数删除项目之前未删除的行（P4b review 第 6 节 M5）；3.20 的 P7b 两行：总体设计 5.3、`plane-diff.md` 二·按表的 `labels` 和第四节的 P7b 行。

不加 Go 模块、npm 包；跨模块的端口不变；新表一张。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录（`server/internal/` 省略）。"Task"是 plan 中负责它的任务（plan 有完整的文件表）。本 Phase 77 个手写的文件（新建 24 个、修改 53 个）、生成物 6 个；一个迁移、一张新表，跨模块的端口不变。

| 路径 | 内容 | Task |
|---|---|---|
| `server/migrations/sql/00014_project_labels.sql`、`server/sqlc.yaml`、`deploy/runtime-grants.sql`；`server/migrations/schema_test.go`、`project_schema_test.go`；`project/adapter/postgres/queries/cascade.sql`、`cascade.go`、`cascade_test.go`；`project/app/ports.go`、`deletion.go`、`fakes_write_test.go`、`cascade_test.go`、`delete_project_test.go`；`bootstrap/project_deletion_test.go`、`workspace_deletion_test.go`；契约两句；`e2e/fixtures/assert/project.ts` | 表、授权、两个连带 | 1 |
| `project/domain/label.go`、`label_test.go`、`errors.go`、`project.go`、`state.go`、`state_test.go` | 标签的规则，两个标签码；名称的检查与状态共用 | 2 |
| `project/adapter/postgres/queries/labels.sql`、`labels.go`、`labels_test.go`、`label_writes_test.go`、`failures_test.go`、`states_test.go`；`project/app/label_ports.go` | 七条语句、存储方法、存储测试、失败测试 | 3 |
| `access/domain/rules.go`、`rules_test.go`；`project/domain/actions.go`；`project/app/label_ports.go`、`label_parent.go`、`create_label.go` 及测试、`fakes_label_test.go`、`fakes_state_test.go`、`fakes_write_test.go`、`clock_test.go` | `createLabel` 的用例 | 4 |
| 契约；`project/adapter/http/handler.go`、`labels.go` 及测试、`handler_test.go`；`project/module.go`；前端的三个文案文件；`bootstrap/permission_matrix_labels_test.go`、`permission_matrix_seeded_test.go`、`permission_matrix_columns_test.go`、`permission_matrix_test.go`、`project_write_locks_test.go`、`project_writes_test.go`、`project_connection_test.go` | `createLabel` 的接口和组合；矩阵的标签种子和行 | 5 |
| `rules.go`、`actions.go`；`project/app/label_ports.go`、`update_label.go` 及测试、`fakes_label_test.go`、`fakes_write_test.go`、`clock_test.go` | `updateLabel` 的用例 | 6 |
| 契约；HTTP；`module.go`；文案；`bootstrap/permission_matrix_project_test.go`、`permission_matrix_states_test.go`、`permission_matrix_labels_test.go`、`permission_matrix_columns_test.go`、`permission_matrix_targets_test.go`、`project_write_locks_test.go`、`project_writes_test.go`、`project_connection_test.go` | `updateLabel` 的接口和组合；状态和标签的矩阵行共用 `rowsByID`；以标签行为目标的矩阵列；最先锁工作区的测试加 `/api/v0/labels/` | 7 |
| 契约；`rules.go`、`actions.go`、`label_ports.go`、`delete_label.go` 及测试、`fakes_label_test.go`、`clock_test.go`；HTTP；`module.go`；矩阵、最先锁工作区、盖戳、连接 | `deleteLabel` | 8 |
| 契约；`rules.go`、`actions.go`、`label_ports.go`、`list_labels.go` 及测试、`fakes_label_test.go`；HTTP；`module.go`；矩阵 | `listLabels` | 9 |
| `bootstrap/interleaving_labels_test.go`、`label_rows_test.go`、`project_row_races_test.go` | 并发 | 10 |
| `e2e/stories/project/p7-labels.spec.ts`、`p4-archive.spec.ts`、`e2e/stories/workspace/w2-landing.spec.ts`、`w3-workspace-settings.spec.ts`、`e2e/fixtures/api.ts`、`assert/project.ts`、`assert/workspace.ts`；`bootstrap/workspace_deletion_catalog_test.go`；`docs/v0/v0-design.md`、`docs/v0/plane-diff.md` | 故事；文档 | 11 |

生成物：`project/adapter/postgres/gen/cascade.sql.go`、`gen/labels.sql.go`、`gen/models.go`（Task 1、3）；`api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`project/adapter/http/gen/server.gen.go`（Task 1、5、7、8、9）、`project/adapter/http/gen/bodyshape.gen.go`（Task 5、7）。

### 2.2 依赖

不加 Go 模块、npm 包。一个迁移（`00014`）、一张表（`labels`）；`deploy/runtime-grants.sql` 给运行时的角色 `labels` 的读写。`access` 的规则表加四行；`workspace` 模块不改（删除工作区的连带经 `project` 的 `DeleteWorkspaceProjects`，它的步骤就是 `deleteProjects` 的步骤）。

### 2.3 表和两个连带（Task 1；3.6、3.16、4.10、11.5）

- `00014_project_labels.sql`：照 4.10 的 12 列；`project_id NOT NULL`（没有工作区级标签）；`parent_id uuid REFERENCES labels ON DELETE CASCADE`；`CHECK (name <> '')`；`CONSTRAINT labels_not_own_parent_check CHECK (parent_id <> id)`；`color` 默认 `''`，`sort_order double precision NOT NULL DEFAULT 65535`；`labels_project_id_name_key ON (project_id, lower(name)) WHERE deleted_at IS NULL`；不带条件的 `labels_parent_id_idx`、`labels_workspace_id_idx`、`labels_project_id_idx`。Down 删表。迁移的测试：`TestMigrationsGoUpDownAndUpAgain`（14 个）、`TestConstraintAndIndexNames`（`projectNames` 加标签的 13 个名字）、`TestProjectChecksRejectCounterexamples`（空名称、自己做自己的父标签）、`TestProjectUniqueKeysHoldAmongUndeletedRowsOnly`（Web 有 Bug 时 bug 被拒绝，Bug 删除之后可以；Ops 另有它自己的 Bug）。
- 连带：`queries/cascade.sql` 的 `DeleteLabels`（照 `DeleteStates`：工作区的、`project_id` 给出时只那个项目的未删除的标签，父子一起，在给出的时刻、由给出的账户），`Store.DeleteLabels`；`app.ProjectsDeleter` 加 `DeleteLabels`；`deleteProjects` 的步骤是项目、成员关系、成员的显示设置、状态、标签（`deletion.go` 的"P7b adds the labels at the end"半句删去）。删除项目、删除工作区都跑这同一组步骤，谁也漏不了另一个删的表。
- 测试：存储的 `cascade_test.go`（`deletionSteps` 从端口反射出全部步骤，标签一步自动在内；`checkDeletions` 的表加 `labels`，`seedProject` 加一个标签和它的子标签）；单元的 `TestDeleteWorkspaceProjects…`（五步的次序）、`TestDeleteProjectReturnsEachFailure`（标签一步的失败原样返回，之后不运行）；组合的删除项目、删除工作区的目录驱动的测试（`project_deletion_test.go`、`workspace_deletion_test.go`）各种下 Bug 和它的子标签 UI（`seedLabels`，直接写，不依赖哪个写已经有了），由不是删除者的账户写，删除者保留写者的话看得出；删除工作区时最后一步失败的测试改为让标签一步失败（请求期间把 `labels` 改名），之前的步骤都回滚。
- 契约：`deleteProject`、`deleteWorkspace` 的描述加"its labels"。
- 端到端的过渡（第 3 节第 11 条）：`expectProjectDeleted` 读 `projects` 的每个外键表，表有了而故事还写不出标签（`createLabel` 在 Task 5），Task 1 暂时把 `labels` 排除在外，注释写明；Task 11 删去，并让 `expectProjectDeleted` 核对它读的表就是有 `project_id` 列的每张表（裁定 B4 的条件，2.11）。

### 2.4 标签的规则（Task 2；3.16、5.2、5.3）

`project/domain/label.go`（新）：

- `Label{ID, WorkspaceID, ProjectID, ParentID *uuid.UUID, Name, Color, SortOrder, CreatedAt, UpdatedAt}`，`Label.Place()`（共用路径的 `placed`）；`LabelCreate{Name, Color, ParentID *uuid.UUID}`（没有 `sort_order`：新标签的位置由项目给，第 3 节第 2 条）；`LabelPatch{Name, Color *string; SetParent bool; ParentID *uuid.UUID; SortOrder *float64}`（`SetParent` 说父标签给了没有，给了时 `ParentID == nil` 是"移到顶层"）。
- `CheckNewLabel(LabelCreate) error`：名称 `checkRequiredText`（1–255 个字符、不空白、不含 NUL），颜色 `checkLabelColor`（至多 255 个字符、不含 NUL、可以为空）；有问题的字段一次全部报告（一个 422 `validation_failed`）。名称是否已被占用由数据库判断；父标签可不可以由用例在项目的锁下判断（`CheckParent`）。`CheckLabelPatch(LabelPatch) error`：只查给了的字段；`sort_order` 可以是任何数。
- `SortOrderAfter(greatest *float64) float64`：最大值加 10000；`nil` 时 65535（Plane `Label.save()`、列的默认值；用例每次用它，设计 4.10，第 3 节第 2 条）。
- `CheckParent(id, project uuid.UUID, parent *Label, hasChildren bool) error`：`id` 是这个标签（新建时 `uuid.Nil`），`parent` 是锁下读到的父标签（没有或已删除时 `nil`）。依次：不是这个项目的标签（含没有、已删除、别的项目的）"must be a label of the project"；是它自己"must not be the label itself"；父标签自己有父标签"must be a label without a parent: labels have two levels"；这个标签有子标签"must be null: the label has labels under it, and labels have two levels"。每个都是一个 422 `validation_failed`，`parent_id` `not_allowed`。
- 名称的检查与状态共用（第一次运行的 T9 与 P7a 的 T2-d 的手工合并，附录 A）：`project.go` 的 `checkLength`（P7a，不空白、不超长）拆出 `checkMaxLength`（只看长度），加 `checkRequiredText(field, s, limit)`（`checkLength` 再 `checkText`，第一个问题）；`state.go` 的 `checkStateText` 删去，状态的名称和颜色改用 `checkRequiredText(…, maxStateText)`；标签的颜色用 `checkMaxLength` 和 `checkText`。不另写第二个长度检查。
- `project/domain/errors.go`：`ErrLabelNotFound`（404 `project.label_not_found`，"The label does not exist, or you cannot see its project."）、`ErrLabelNameTaken`（409 `project.label_name_taken`，"A label of the project has this name, in this case or another."）。说明对每个收到它的调用者都成立（清扫 30）；它们随第一个回答它们的操作进入契约（Task 5、7）。
- 测试：`TestLabelPlace`、`TestCheckNewLabelAcceptsValidLabels`、`TestCheckLabelReportsEveryField`（名称空、空白、256 个字符、含 NUL；颜色 256 个字符、含 NUL；两个问题的字段报第一个；`CheckLabelPatch` 只查给了的）、`TestSortOrderAfter`、`TestCheckParent`（四种拒绝和两种接受，按切片的顺序）。

### 2.5 存储（Task 3；3.6、3.16、6.7）

七条语句，写进 `project/adapter/postgres/queries/labels.sql`：

```sql
-- name: CreateLabel :one
INSERT INTO labels (id, workspace_id, project_id, parent_id, name, color, sort_order, created_by_id, updated_by_id, created_at,
                    updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.narg(parent_id), sqlc.arg(name), sqlc.arg(color),
        sqlc.arg(sort_order), sqlc.arg(created_by), sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now))
RETURNING id, workspace_id, project_id, parent_id, name, color, sort_order, created_at, updated_at;

-- name: LabelByID :one
SELECT … FROM labels WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: ListLabels :many
SELECT … FROM labels WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL ORDER BY sort_order, id;

-- name: GreatestSortOrder :one
SELECT sort_order FROM labels WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL ORDER BY sort_order DESC LIMIT 1;

-- name: HasChildren :one
SELECT EXISTS (SELECT 1 FROM labels WHERE parent_id = sqlc.arg(id)::uuid AND deleted_at IS NULL);

-- name: UpdateLabel :one
UPDATE labels l
SET name          = coalesce(sqlc.narg(name)::text, l.name),
    color         = coalesce(sqlc.narg(color)::text, l.color),
    parent_id     = CASE WHEN sqlc.arg(set_parent)::boolean THEN sqlc.narg(parent_id)::uuid ELSE l.parent_id END,
    sort_order    = coalesce(sqlc.narg(sort_order)::double precision, l.sort_order),
    updated_by_id = sqlc.arg(updated_by)::uuid,
    updated_at    = sqlc.arg(now)
WHERE l.id = sqlc.arg(id) AND l.deleted_at IS NULL
RETURNING …;

-- name: DeleteLabel :exec
UPDATE labels
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE (id = sqlc.arg(id) OR parent_id = sqlc.arg(id)) AND deleted_at IS NULL;
```

（`…` 是每条读标签的语句共同的九列；完整的文件在 plan Task 3，每条语句带说明它是谁的、在什么锁下。）

- 存储方法（`labels.go`）：`CreateLabel(ctx, app.LabelRow) (domain.Label, error)`、`LabelByID(ctx, id) (domain.Label, bool, error)`、`ListLabels(ctx, projectID) ([]domain.Label, error)`、`GreatestSortOrder(ctx, projectID) (*float64, error)`、`HasChildren(ctx, id) (bool, error)`、`UpdateLabel(ctx, id, domain.LabelPatch, by, now) (domain.Label, error)`、`DeleteLabel(ctx, id, by, now) error`，每条经 `s.queries(ctx)`（调用者的事务）。`labelTaken`：唯一冲突只在约束名恰好是 `labels_project_id_name_key` 时答 `domain.ErrLabelNameTaken`（P7a review 第 6 节坑 8）；别的冲突（重复的 id）、CHECK、外键的违反都是内部错误，不是领域错误。`app.LabelRow{ID, WorkspaceID, ProjectID, ParentID, Name, Color, SortOrder, CreatedBy, Now}`（`label_ports.go`，Task 3 只有它：存储的 `CreateLabel` 收它）。
- 测试（`labels_test.go`、`label_writes_test.go`，在 P7a 的 `stateWorld` 上：acme 的 Web、Ops（已归档），beta 的 Site；`addLabel` 经 `CreateLabel` 存）：`TestCreateLabel`（每一列，父标签，别的标签每一列不变）、`TestCreateLabelNameTaken`（bug、BUG 在 Bug 旁边各 409，是 API 会答的第一个问题，什么都不存；已删除的标签的名称、别的项目的名称是空着的）、`TestCreateLabelBreakingAnotherConstraintIsInternal`（重复的 id、空名称、自己做自己的父标签、不存在的父标签，各是那个约束的违反，不是领域错误）、`TestLabelByID`（问哪个答哪个，两行各问一次，读第一行的会答错其中一个；已删除的、没有的不答）、`TestListLabels`（父子一起，按 `sort_order` 再按 id：Feature 最后加，`sort_order` 与 UI 相同而 id 更小，排在 UI 之前，行在表里的次序给不出；不含已删除的 Old，不含别的项目的；已归档的 Ops 照常）、`TestGreatestSortOrder`（子标签的也算；已删除的 99999、Ops 的 80000 不算；没有标签时 `nil`）、`TestHasChildren`（Bug 有 UI；UI 没有；Feature 唯一的子标签已删除；没有的 id 没有）；`TestUpdateLabel`（Web 的 UI：给了的字段改、审计列改为给出的时刻和账户，别的列、别的标签不变；父标签给了就设、给 `nil` 就清、没给就留；什么都不给的补丁跟在给了全部字段的补丁之后，它保留的父标签、颜色、`sort_order` 既不是空也不是列的默认值，清扫 26）、`TestUpdateLabelNameTaken`、`TestUpdateLabelWritesNoDeletedLabel`、`TestDeleteLabel`（Web 的 Bug 和它下面的 UI 在给出的时刻、由给出的账户删除，`deleted_at` 与 `updated_at` 相同，别的列不动；在 Bug 下、之前已删除的 Gone 保留它的时刻；Feature、Feature 的 Docs、Ops 的 Bug 每一列不变；再删 Bug 什么都不改；删除子标签 Docs 只删它自己）。`failures_test.go`：七个方法在取消的 context 上各原样返回 `context.Canceled`、不是 `*shared.Error`（P7a review 第 6 节坑 7），读不答"没有这个标签""没有标签""没有 `sort_order`""没有子标签"这些零值（`seedLabel` 给 Web 一个标签和它的子标签）。
- 每条语句都没有"取第一行"之外的行序依赖：`GreatestSortOrder` 带 `ORDER BY sort_order DESC LIMIT 1`，`ListLabels` 带完整的 `ORDER BY`，它的次序由行在表里的次序给不出的一行核对（清扫 1 的"两种行序"、清扫 48）。

### 2.6 `createLabel`（Task 4、5；3.4、3.6、3.16、3.19、5.1–5.3、9.2）

- Task 4（用例）：`access` 的 `label.create`（项目级，项目管理员；同时是工作区管理员的项目成员由 Authorizer 的通则得到，3.4；不给项目访客，9.2）；`ActionLabelCreate`。`label_ports.go`：`LabelFinder`（`LabelByID`：按行寻址的写不加锁地读、锁下重读，`checkParent` 在项目的锁下读父标签）、`LabelCreator`（`LabelFinder`、`GreatestSortOrder`、`CreateLabel`）。`label_parent.go` 的 `checkParent(ctx, labels LabelFinder, id, project, parentID uuid.UUID, hasChildren bool) error`：在项目的 `FOR NO KEY UPDATE` 之下读 `parentID`（每个标签的写都取这把锁，读到的标签在提交之前不变），回答的 id 不是问的那个时是错误（清扫 22），再 `domain.CheckParent`。`CreateLabel`：`NewCreateLabel(locks Locks, labels LabelCreator, tx shared.TxManager, clock Clock)`；`Execute(ctx, projectID, domain.LabelCreate) (domain.Label, error)`：没有调用者、领域拒绝的值，在事务之前；事务里 `lockAndDecide`（工作区 S → 项目 N → 判定 `label.create`）→ 给了父标签时 `checkParent(…, uuid.Nil, projectID, *in.ParentID, false)` → `sort_order`（`GreatestSortOrder` 和 `SortOrderAfter`，每次都在锁下读：设计 4.10，第 3 节第 2 条）→ 时钟 → `CreateLabel`；回答的 id 不是插入的 id 时是错误。已归档的项目照常建（3.19）。
- 假实现（`fakes_label_test.go`）：`fakeLabels` 嵌入 `*fakeStore`，共用它的调用记录、失败（`errs`）、锁下的重读（`reread`）、`answersAs`、`changedAs`；按 id 的读与状态的共用一个 `readRow[R]`（`fakes_write_test.go`：记调用、按 `errs` 失败、锁下的重读照 `reread` 失败、消失或换项目、回答 `answersAs`），`LabelByID` 和 P7a 的 `StateByID` 都只是取出这一行再交给它，P7a 的状态测试的子测试、名称和断言不变（附录 A 的 PF-L3）；`of`、`GreatestSortOrder` 两份各自留着：合起来要为两种不相干的行各传取项目和次序的函数，比两段各九行左右的循环难读；`newLabels`：web 的 Bug（顶层）、它下面的 UI、Feature（顶层），ops 的 Docs；`of` 按名称排，不是按 `sort_order` 的两种次序，也不是按 id 的（清扫 16）；`CreateLabel` 照存储查名称（不分大小写）。
- 测试：`TestCreateLabel`（四行：web 顶层在 Feature 之后 95535；在 Bug 下照样；归档的 ops 在 Docs 之后 75535；ops 没有标签时 65535；调用记录、由调用者、时刻到微秒）、`TestCreateLabelRefuses`（14 行：没有调用者、领域拒绝的值，在事务之前；没有项目、等锁时 acme 或 web 被删除、web 移到别的工作区、看不到 web 的调用者，各 `project.not_found`；成员 403；之后各 422 `parent_id not_allowed`：父标签不是标签、是 ops 的、是 UI（在 Bug 下）；与 Bug 同名不同大小写，存储的 409，在插入之后；回答另一个 id 的标签、读成另一个 id 的父标签，写自己的错误）、`TestCreateLabelReturnsEachFailure`（每个端口调用的失败各原样返回，之前的调用照旧、之后什么都不运行）；`TestEachWriteReadsTheClockUnderItsLock` 加 `createLabel`。
- Task 5（接口和组合）：契约 `POST /api/v0/projects/{project_id}/labels`（`createLabel`，201 `Label`；`x-problem-codes: [validation_failed, project.not_found, forbidden, project.label_name_taken]`）；`Label`、`LabelCreate`（`name` 必填，`color`、`parent_id` 可选；没有 `sort_order`，`additionalProperties: false` 让给了它的请求答 400，钉在 `TestCreateLabelHoldsTheBodyToItsStructure` 原有的多余字段里：`{"name":"A","sort_order":1}`，不另加行）。`Label` 的说明不说工作项（P7a 的第 3 节第 7 条）。HTTP：`UseCases.CreateLabel`、`CreateLabel` 处理函数（没给的颜色为空，没给的父标签为 `nil`）、`label(domain.Label) gen.Label`（父标签在顶层时 `null`）；每个映射的字段在这个 Task 由 HTTP 测试钉住（坑 5）。接线；包说明（`moddoc`）。前端文案 `project.label_name_taken`。
- 矩阵（`permission_matrix_labels_test.go`，新）：`matrixLabels` 是每个矩阵项目的 Bug（65535）、它下面的 UI（75535）、Feature（85535）；`prepareMatrix` 经存储（`labels`，由工作区管理员，父标签在前）建出，`seeded.label(key, name)` 回答它们的 id（没有种下的名称立即失败）；`seededLabels` 按名称读回每个矩阵项目的标签，前提核对恰好是这些（父标签、`sort_order`；gone 的随它删除，别的未删除，清扫 10）。`createLabel` 的五行：成功（`createsTheLabel`：QA 在列的项目的顶层，Feature 之后 95535）；"a name taken"（bug：409 在判定之后，不能建标签的人不知道项目有哪些标签）；"a parent with a parent"（UI 做父标签）、"a parent of another project"（同一工作区的已归档项目的 Bug：查的是父标签的项目，不只是它的工作区，`r-parent-same-workspace`），各 422 `parent_id not_allowed`，在判定之后；已归档项目的小表。种子写在新文件里（`permission_matrix_seed_test.go` 389 行，没有余地，文件的说明写明）。
- 组合：`TestEachWriteOnAProjectSharesItsWorkspaceFirst` 加 `createLabel`；`TestTheWritesOnAProjectStampTheirRequest` 加 `createLabel`（Web 归档期间：已归档项目的标签照常写，坑 6）；`TestTheWritesOnAProjectRunOnTheirTransactionsConnection` 加建 Bug、在 Bug 下建 UI、在 UI 下建 Icons（422：父标签的读在事务的连接上被拒绝，坑 3）。

### 2.7 `updateLabel`（Task 6、7；3.4、3.6、3.16、3.19、6.7、9.2）

- Task 6（用例）：`label.update`（同 `label.create`）；`ActionLabelUpdate`。`label_ports.go`：`labelWrite(id, action, labels LabelFinder) rowWrite[domain.Label]`（照状态的 `stateWrite`，P7a F3：`find` 是 `LabelByID`，`notFound` 是 `domain.ErrLabelNotFound`）；`LabelUpdater`（`LabelFinder`、`HasChildren`、`UpdateLabel`；说明只说 `HasChildren`、`UpdateLabel` 在项目的锁下，`LabelByID` 照 `LabelFinder` 说的，坑 2）。`UpdateLabel`：`NewUpdateLabel(locks, labels LabelUpdater, tx, clock)`；`Execute(ctx, id, domain.LabelPatch) (domain.Label, error)`：没有调用者、领域拒绝的值，在事务之前（6.7）；事务里 `lockRowAndDecide(…, labelWrite(id, ActionLabelUpdate, u.labels))`；给了父标签、且不是"移到顶层"时，`HasChildren(l.ID)` 和 `checkParent(…, l.ID, l.ProjectID, *p.ParentID, hasChildren)`（判定之后、写之前）；时钟；`UpdateLabel`；回答的 id 不是这一行的时是错误。已归档项目的标签照常改（3.19）。
- 假实现：`HasChildren`、`UpdateLabel`（照存储：给了的字段改，名称不分大小写查重、自己除外，`changedAs`）、`labelPatch`；`failsFor`（只让这一个标签的读失败：父标签的读失败而这个标签自己的读照常，清扫 2）。
- 测试：`TestUpdateLabel`（六行：Feature 改名改色、放到 Bug 下、给 `sort_order`；Bug 改成另一种大小写（自己的名称不算占用）；UI 移到顶层，不读子标签、父标签；归档的 ops 的 Docs 照常；调用记录、由调用者、时刻到微秒）、`TestUpdateLabelRefuses`（16 行：没有调用者、领域拒绝的值，在事务之前；没有这一行、等锁时 acme 或 web 被删除、这一行被删除或移到 ops、看不到 web 的调用者，各 `project.label_not_found`；成员 403；之后各 422 `parent_id not_allowed`：Feature 放到它自己下、放到 UI（在 Bug 下）下、放到 ops 的 Docs 下、放到不存在的标签下、Bug（有 UI）放到 Feature 下；与 Bug 同名不同大小写，存储的 409；回答另一个标签的改动，写自己的错误；重读成另一个 id 的标签是共用路径的，在 `lock_test.go` 里钉过一次，不再加行，S6）、`TestUpdateLabelReturnsEachFailure`（Feature 放到 Bug 下，子标签和父标签都读；每个调用的失败各原样返回）；时钟一行。
- Task 7（接口和组合）：`PATCH /api/v0/labels/{label_id}`（`updateLabel`，200 `Label`；`[validation_failed, project.label_not_found, forbidden, project.label_name_taken]`）；`LabelID`、`LabelUpdate`（`parent_id` 是 `[string, 'null']`：`null` 移到顶层，没给不动）。HTTP（`named(req.Body.ParentID)` 给出 `SetParent`、`ParentID`）、接线；文案 `project.label_not_found`。矩阵：`rowsByID{path, notFound, find}`（`permission_matrix_project_test.go`：按 id 指名项目下一行的写的格子 `of`、`ofArchived` 和请求 `to`），P7a 的 `ofState`、`ofArchivedState`、`toState` 换成它的 `stateRows`（行、格子、断言不变），标签的是 `labelRows`；`updateLabel` 五行（Feature 改名 Story：`renamesTheLabel`；"a name taken"；"a label with labels under it given a parent"（Bug 放到 Feature 下）；"a value refused"（空名称，在看标签之前，每一列都是 422 `name too_short`）；已归档）；列的核对（`permission_matrix_columns_test.go`、`permission_matrix_targets_test.go`）认得瞄准标签的格子（`{label_id}`，标签从不在 `seeded` 的成员关系里）。`project_write_locks_test.go`：`labelNamed`，`rowPaths` 加 `/api/v0/labels/`，`updateLabel` 一行（QA 改名 Checked），第一步探测它的标签行没有被持有。盖戳加 `updateLabel`（QA 先由 bob 写过，`bobs`）；连接加 Bug 放到 UI 下（422）、UI 改名 Widgets 并移到顶层、再放回 Bug 下。

### 2.8 `deleteLabel`（Task 8；3.6、3.16、3.19、约定五）

- 契约 `DELETE /api/v0/labels/{label_id}`（204；`[project.label_not_found, forbidden]`）。`label.delete`（同 `label.create`）；`ActionLabelDelete`；`LabelDeleter`（`LabelFinder`、`DeleteLabel`；说明写明项目的锁覆盖它下面的标签：它们是这个项目的，3.16、约定五）。
- `DeleteLabel`：`lockRowAndDecide(…, labelWrite(id, ActionLabelDelete, …))` → 时钟 → `DeleteLabel(l.ID, actor.UserID, now)`（一条语句：这个标签和它下面的标签）。
- 测试：`TestDeleteLabel`（Bug 连同 UI；UI（在 Bug 下）、Feature 各只删自己；已归档的 ops 的 Docs 照常）、`TestDeleteLabelRefuses`（8 行，同 `updateLabel` 的锁和判定的一半）、`TestDeleteLabelReturnsEachFailure`；时钟一行；HTTP；矩阵两行（Bug：PA、PM+WA 204，没有 `check`：204 没有回答体，删除的效果由盖戳在 Web 归档期间核对，坑 6；已归档）；最先锁工作区（Checked）、盖戳（bob 写过的 Bug 和它下面的 UI，两行都由 alice 在请求的时刻删除，`deleted_at = updated_at`）、连接（删除 Bug，Widgets 随它）各加一行。

### 2.9 `listLabels`（Task 9；3.4、3.12、3.16、3.19、9.2）

- 契约 `GET /api/v0/projects/{project_id}/labels`（`listLabels`，200 `LabelList`；`[project.not_found, forbidden]`）；`LabelList{data: [Label]}`。`label.list`（项目级，项目的每个有效成员，访客也能，同 `state.list`；Plane 让任何工作区成员、甚至已离开的成员列出，4.11）。
- `ListLabels`：`findAndDecide`（`label.list`）→ `ListLabels`，存储的顺序原样；不开事务。已归档的项目照常列出（3.19：与 `listStates` 不同，标签不随归档隐去，设计 2 节 P7 和 4.11 没有说要隐去）。
- 测试：`TestListLabels`（存储的次序是任何排序都给不出的，用例原样回答；归档的 ops 照常；不开事务）、`TestListLabelsRefuses`；HTTP（`data` 是数组、不是 `null`）；矩阵两行（每个有效成员 200：`listsTheLabels` 读出 UI 在 Bug 下、其余在顶层，按 `sort_order`；`PM+WA` 作为成员、`WA-` 不是成员 403；已归档）。

### 2.10 并发（Task 10；3.6 约定二、三、五，3.16，9.3 交错 11）

- `project_row_races_test.go` 的 `rowWrite` 加 `label`，`row()` 加标签的一支，`labelsFor` 在写一个标签时给 Web 种下 `worldLabels`；`rowWrites` 加 `updateLabel`、`deleteLabel`（bob 改 Web 的 Feature）和 `createLabel`（按项目寻址，`creates`，同 `createState`）：
  - `TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`：另一个事务持有 Web 的 `FOR NO KEY UPDATE`（或 acme 的，如连带），其间这一行删除、移到 Ops（直接改 `project_id`，接口改不了：锁下重读确认项目的那一半），调用者的成员关系结束，Web 删除，acme 删除；写在通过认证和不加锁的读之后等它，提交之后答 404，什么都不改。`createLabel` 只跑后三种。
  - `TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`：逐个放开 acme 的 N、Web 和这一行的 S，`lockOn` 读出每一步此时最强的锁；标签的两个写等这一行时持有 Web 的 `FOR NO KEY UPDATE`；写的时刻不早于 Web 的放开、早于这一行的放开。`createLabel` 由 `creates` 跳过：它的顺序和强度由最先锁工作区的测试和交错钉住。
  - 跳过的条件加上标签："这一行结束"只对成员关系（`c.membership && (m.state != "" || m.label != "")`：状态、标签不结束）；离开由 `m.member == m.by` 认出（它的行是调用者自己的成员关系），状态、标签的写也把调用者放在 `member`，条件另要 `m.state == "" && m.label == ""`，否则标签的"这一行删除""移到 Ops"会被当成离开跳过（附录 A）。
- `TestLabelWritesOnOneProjectSerialize`（`interleaving_labels_test.go`，新）：在 `memberWorld` 上，Web 和 Ops 各种下 `worldLabels`（Bug、它下面的 UI、Feature、Docs；`seedLabelsOf` 经存储由 alice（两个项目的管理员）建出、读回、核对，坑 4）。第一个写持有 acme 的 S、Web 的 N 和它写的行，停在它写之后的门（`labelWrittenHolding`）；第二个共享 acme、等 Web 的行（两者都不写 `projects` 的行，只有这把锁的等待满足探测）。第一个提交之后，第二个照第一个提交的判定（它的父标签和这个标签的子标签都在锁下读）。七对、两种顺序：
  1. Feature 放到 Docs 下与 Docs 放到 Feature 下（交错 11）：第二个的父标签此时有父标签，422 "must be a label without a parent"；没有三层，没有环；
  2. 在 Feature 下建 Icons 与把 Feature 放到 Docs 下：先建，Feature 有子标签、不能有父标签（422 "must be null…"）；先移，Feature 有父标签、不能做父标签（422 "must be a label without a parent"）；
  3. 建 QA 与建 qa：第二个 409 `project.label_name_taken`（同名的并发创建两种顺序，完成线）；
  4. Feature 放到 Docs 下与放到 Bug 下：第二个写自己的标签在等锁时变了（坑 1），第二个照样成功，从第一个放的地方移到它要的地方；
  5. 删除 Bug 与把 Feature 放到 Bug 下、6. 删除 Bug 与在 Bug 下建 Icons（约定五的探测，第 3 节第 4 条）：先删除，Bug 不再是项目的标签（422 "must be a label of the project"）；另一个先，删除的一条语句连同刚放到 Bug 下的标签和 UI 一起删除，在每个标签的写都取的那把锁下；
  7. 在顶层建 QA 与在 Bug 下建 Icons：两个都成功；第二个在第一个之后（它的位置在锁下读），时刻是它拿到锁之后时钟给的（`createLabel` 在锁之后读时钟，组合一层只有这一对的第二个写是成功的创建）。
  之后 Web 的标签是第一个留下的样子（第二个成功时加上它的），时刻不早于门打开的时刻；Web 未删除的标签没有两个在同一个 `sort_order`（新标签的位置在项目的锁下读，设计 4.10；在锁之前读最大值的 `s25-create-greatest-unlocked` 在第 7 对的两种顺序失败，清扫 25）；Ops 的不变（清扫 49）。
- `TestEachLabelWriteChangesItsRowsAlone`（`label_rows_test.go`，新）：在 `memberWorld` 上，Web 和 Ops 各种下 `worldLabels`；bob（Web 的管理员）在 Web 顶层建 QA、在 Feature 下建 Icons、在 Docs 下建 Fonts、改 Feature 的名称、把 UI 移到顶层、删除 Fonts、把 Docs（它下面唯一的标签已删除）放到 Bug 下、删除 Feature（Icons 随它）、删除 Docs（之前删除的 Fonts 保留它的删除）；其间他把 Bug 放到 Docs 下（422：Bug 有 UI）、建 bug（409：Bug 的另一种大小写），各什么都不写。每个写之后，它写的行之外每张表每一行不变，已删除的行也在内（`rowsBut`、`tableRows`，清扫 23、44、49）。

### 2.11 端到端与文档（Task 11；第 2 节 P7、W3、P4，3.20、4.11）

- `e2e/stories/project/p7-labels.spec.ts`（P7 的接口版本）：Web 的管理员 ann 建 Bug（65535）、Feature（75535）；工作区管理员 admin（Web 的另一位管理员）建 UI（85535，没有颜色）；ann 把 UI 拖到 Bug 下（名称、颜色、`sort_order` 不变）；admin 把 UI 改名 Widgets（父标签、`sort_order` 不变），把 Feature 移到 Bug 之前（55535）；mem（Web 的成员）按 `sort_order` 列出（Feature、Bug、Widgets：不是建的次序）；ann 建 Ops，它有自己的 Bug（名称只在项目内唯一）；之后九个拒绝，各什么都不改：QA 的颜色 256 个字符，422 `{field: color, code: too_long}`；bug（Bug 的另一种大小写）、admin 把 Feature 改名 WIDGETS，各 409 `project.label_name_taken`；在 Widgets 下建 Icons（第三层）、把 Feature 放到 Ops 的 Bug 下（别的项目）、放到它自己下、把 Bug（有 Widgets）放到 Feature 下，各 422 `{field: parent_id, code: not_allowed}`；mem 改 Feature、建 QA，各 403；ann 删除 Bug，Widgets 随它，同一时刻、由她；之后再删除 Bug、改 Widgets 的名称，各 404 `project.label_not_found`，在已删除的 Bug 下建 Icons 422；bug 又空出来，ann 建它，在 Feature 之后（65535：删除了的标签的 `sort_order` 不再算，其中 Widgets 的 85535 最大）；mem 的列表；ann 归档 Ops，它的标签照常列出、在 Bug 下建 Runbook（在 Bug 之后的 75535：新标签在项目的标签之后）、把 Bug 改名 Incident、把 Runbook 移到顶层（`parent_id: null`）；最后 admin 在 Elsewhere 的两个项目（`amidAnotherWorkspace` 建在 Web 前后）没有标签。每一步之后、每组拒绝之前之后用 SQL 读出标签的行（`expectLabels`：名称、颜色、父标签的名称、`sort_order`、删除、最后写它的人、行在它的项目的工作区、父标签在同一项目、删除的时刻就是它最后一次写的时刻；清扫 27、38）。
- W3：两个工作区各有的 Web 由成员（Web 的负责人，因而是它的管理员）建 Bug 和它下面的 UI；Old 由成员建一个标签，删除 Old 时由管理员重写；Acme 删除之后 Web 的两个标签由管理员在同一时刻删除，Other 的 Web 的照旧（`expectLabels`）；`deletedAloneTables` 加 `labels`（Old 的标签随 Old 先删除）。W2：First 的 Web 建一个标签（`expectWorkspaceDeleted` 要每张表都有一行随工作区删除）。P4：管理员在归档的 Web 里建 Bug、它下面的 UI、Feature，删除 Feature；删除 Web 之后 Bug、UI 随它删除，Feature 保留它自己的、更早的时刻。
- 夹具：`api.ts` 的 `Label`、`LabelCreate`、`LabelUpdate`、`createLabel`；`assert/project.ts` 的 `LabelRow`、`expectLabels`；`expectProjectDeleted` 不再排除 `labels`（Task 1 的过渡），只数删除项目之前未删除的行（P4b review 第 6 节 M5：之前删除的行保留它的时刻，不算随项目删除，也不算没删），并核对它从目录读出的表就是有 `project_id` 列的每张表（说明写明；留着排除或少一张表在这里失败：裁定 B4 的条件，`e2e-exclusion-kept`）；`assert/workspace.ts` 的 `workspaceTables`、`deletedAloneTables` 加 `labels`，说明的"P7b"半句删去。
- `workspace_deletion_catalog_test.go` 的 `keysTo` 的说明：表指向它自己的外键（`labels.parent_id`）照别的外键跟随，这样的表凭它自己指向父表的外键通过（P4b review 第 6 节；`labels` 不需要 P12 说的 `AND tab <> $1` 和反例：`projects`、`workspaces` 都不指向它下面的表）。
- 文档（3.20 的 P7b 两行）：总体设计 5.3 的"标签名在工作区内唯一"改为"标签名在项目内唯一（不分大小写）；没有工作区级标签（M3 设计 3.16）"；`plane-diff.md` 二·按表的 `labels` 五行（原"工作区级标签的名称唯一范围"一行改写为索引一行），第四节的标签的名称、层级、工作区级标签、列出标签、新建标签的顺序五行（新建标签的顺序：Plane 让第一个标签用请求给的 `sort_order`；Nerve 的请求不给它，给了答 400，一律在最后，位置用 `updateLabel` 改），删除工作区一行的"标签由 M3/P7b 加入这个事务"改为事务里的一项。

### 2.12 矩阵

标签的 14 行（`permission_matrix_labels_test.go`）：`listLabels` 2 行、`createLabel` 5 行（成功、名称已占用、父标签有父标签、父标签在同一工作区的别的项目、已归档）、`updateLabel` 5 行（改名、名称已占用、有子标签的标签给父标签、值被拒绝、已归档）、`deleteLabel` 2 行（Bug、已归档）。写的格子各在自己的数据库副本上运行。已归档项目的小表（`archivedColumns`：PA、WM、X）随每个操作，在加这个操作的 Task。矩阵一共 853 格（P7a 结束时 721 格，P7b 的 14 行加 132 格：10 行各 12 格、4 个已归档的小表各 3 格），1.53 秒。

## 3. 与设计的差异和补充

第 1、2、3、11 条由控制者裁定（2026-10-06：B1、B3、B4 照本条，B2 不取第 2 条的做法）；其余是本 spec 对设计交给 P7b 的事的决定，和承接条目的落点。

1. **契约文件不拆（裁定 B1：照本条）**。`api/modules/project.yaml` 在 P7b 之后 1,350 行（P7a 结束时 1,139 行）。仓库的约定是一个模块一个描述文件：`Makefile`（"每个模块一个描述文件 api/modules/<模块>.yaml"）由 `api/modules/*.yaml` 的文件名得出模块名，生成到 `internal/modules/<模块>/adapter/http/gen`；`apitest`（`rules_test.go`）要求 `api/modules/<m>.yaml` 的每个操作的标签恰好是 `<m>`，`apitest.Main` 读 `api/modules/<module>.yaml` 核对码的两个方向（M0-P3 交接 5）。按资源拆出 `labels.yaml` 会被当成一个叫 `labels` 的模块，要改这三处的约定，是跨模块的约定，不是 P7b 能定的。P7b 不拆。留着的代价：这一个文件约 1,350 行，超过约 400 行的规则（接口描述按模块一个文件，P7a 起已是写明的例外）；`project` 模块的接口到 P7b 就齐了（后面的 P8–P11 是前端），它不再长；M4 的工作项是别的模块、别的文件。要拆，是一个单独的任务：约定改成"一个模块一个目录或多个文件"，`Makefile`、`apitest`、打包（`api/openapi.yaml` 的引用）一起改。
2. **新建标签的 `sort_order`：不收（不取，裁定 B2）**。本条原来提议照第一次运行"给了就用它，没给时最大值加 10000，没有标签时 65535"；控制者不取，照设计 4.10"新建时为最大值加 10000（照搬）"。Plane 的 `Label.save()` 在项目已有标签时一律用最大值加 10000，请求给的 `sort_order` 被忽略，只有项目的第一个标签用它（`db/models/label.py`）；Plane 的网页建标签不发 `sort_order`（只有拖动的修改发，`label.store.ts`）。P7b：`LabelCreate` 没有 `sort_order`（契约、HTTP、领域都没有），新标签一律在项目未删除的标签的最大值加 10000，没有标签时 65535，与 Plane 的网页一致；位置由 `updateLabel` 改，同 `createState` 不收 `sequence`、`updateState` 改它（P7a）。给了 `sort_order` 的请求答 400（`additionalProperties: false`），钉在 HTTP 测试原有的多余字段里（2.6），不另加行。`plane-diff.md` 第四节"新建标签的顺序"一行保留、改写（Plane 让第一个标签用给的值，Nerve 不收）。设计不改。代价：要把新标签放到别处的调用者多一次 `PATCH`；以后要收这个字段是只加不改的。
3. **任务的切分：十一个（裁定 B3：照本条）**（裁定 S3：在约 16 个任务、每个任务约 1,500 行以内重新切分）。设计 12 节 P7b 的八个任务草稿：1 表和连带、2 领域和存储、3 `listLabels` 与 `createLabel`、4 `updateLabel`、5 `deleteLabel`、6 并发、7 端到端、8 文档和评审。P7b 的十一个：
   - 设计的任务 2 分成 Task 2（领域）和 Task 3（存储），照 P7a 的 Task 2、3；
   - 设计的任务 3 分成 Task 4（`createLabel` 的用例）和 Task 5（`createLabel` 的接口和组合：契约、HTTP、接线、矩阵的种子和行、最先锁工作区、盖戳、连接），`listLabels` 移到 Task 9：照 P7a 的 Task 5、6 和 Task 8。契约的码要在同一个 Task 由 HTTP 测试返回（`apitest.Main`），矩阵的行要接好的 app，所以接口和组合跟着同一个操作；矩阵的标签种子（Task 5）和它的第一批行一起；Task 5–8 都不需要 `listLabels`（组合的测试用 SQL 读行）。Task 5 在 plan 里 926 行，再加 `listLabels`（Task 9 在 plan 里 612 行，其中代码块 468 行），合在一起约 1,500 行，到了上限；
   - 设计的任务 4 分成 Task 6（用例）和 Task 7（接口和组合），同上；
   - 设计的任务 5 是 Task 8（用例、接口、组合一起：`deleteLabel` 的用例只有一步写，plan 里 655 行）；
   - 设计的任务 6、7 是 Task 10、11；任务 8 的文档并进 Task 11（端到端和文档），"评审"是 Phase 的评审，不是 plan 的 Task。
   - 已归档项目的小表中标签的行随每个操作（Task 5、7、8、9），不集中在后面（设计 12 节 P7b 的"每个加操作的任务同时加它的……矩阵行"）；`project.label_name_taken` 随 `createLabel` 的接口（Task 5），`project.label_not_found` 随第一个回答它的 `updateLabel` 的接口（Task 7），文案随之（约束 4）。
4. **清扫 41 和约定五的例外：`deleteLabel` 连带子标签（裁定 S7 的 O3；P6 review 第 6 节）**。P7b 写多行的语句：`DeleteLabel`（这个标签和它下面的标签，`WHERE (id = $1 OR parent_id = $1) AND deleted_at IS NULL`）和连带的 `DeleteLabels`（工作区的或一个项目的全部标签）；其余的按主键写至多一行（`UpdateLabel`），或插入新行（`CreateLabel`）。这些行（项目 P 的标签）的全部写者和它们持有的锁：
   - `createLabel`、`updateLabel`、`deleteLabel`：工作区 `FOR SHARE` → 项目 `FOR NO KEY UPDATE`，之后才碰标签的行（后两个经共用路径，`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength` 读出每一步的锁；`createLabel` 经 `lockAndDecide`，顺序和强度由最先锁工作区的测试钉住，`p-create-share-project` 在交错的第 3、7 对失败）；
   - 删除项目（`deleteProjects` 的标签一步）：工作区 `FOR SHARE` → 项目 `FOR NO KEY UPDATE`；
   - 删除工作区的连带（`DeleteWorkspaceProjects` 的同一组步骤）：工作区 `FOR NO KEY UPDATE`，与任何标签的写持有的工作区 `FOR SHARE` 冲突；
   - 停用（P6）、结束和恢复成员关系（P5a）、项目成员的写（P4b、P5b）、状态的写（P7a）不写标签。
   一个标签的子标签都在它的项目里（3.16：父标签是同一项目的，`CheckParent` 在项目的锁下检查；标签不能换项目，没有改 `project_id` 的写）。所以 `DeleteLabel` 写的每一行都在它持有 N 的那个项目下：能写到这些行的写都先持有这个项目的 N 或它的工作区的 N，父行的锁让它们先后进行，不会有一个写不经父行的锁碰到这一批行；约定五的例外（按 id 先锁、只写锁住的）不适用，不加按 id 的锁。`DeleteLabels` 同理（项目的或工作区的 N 之下）。最坏的交错在真实数据库上展示：删除 Bug 与把 Feature 放到 Bug 下、与在 Bug 下建 Icons，两种顺序（`TestLabelWritesOnOneProjectSerialize` 的第 5、6 对）：另一个先时，删除的一条语句连同刚放到 Bug 下的标签一起删除（它在 Web 的锁上等到另一个提交，再读、再写）；删除先时，另一个读不到 Bug，答 422。`createLabel` 改取项目的 `FOR SHARE`（`p-create-share-project`）时两个创建不再互斥（`FOR SHARE` 与 `FOR SHARE` 相容，与删除和修改的 N 仍冲突），两个创建的第 3、7 对在组合一层失败（附录 A）。说明写在 `DeleteLabel` 的语句、`LabelDeleter` 的端口上（"A label's children are of its project (3.16), so that lock covers every row the statement writes"）。
5. **清扫 36：别的操作或两步到达同样的结果**（brief：两层的规则、没有环、不分大小写的唯一）。
   - 两层：设父标签的只有 `createLabel`（给了父标签）和 `updateLabel`（给了父标签），两者都在项目的 N 之下经 `checkParent`：父标签没有父标签、这个标签没有子标签；移到顶层不受限。两步：先给 A 一个子标签、再给 A 父标签，第二步 422；先给 A 父标签、再在 A 下建标签，第二步 422；并发的两步由项目的 N 串行（交错 11 和第 2 对）。父标签删除时子标签随它删除（`DeleteLabel` 一条语句），没有挂在已删除的父标签下的未删除的标签；已删除的标签做父标签 422（`LabelByID` 不回答它）。标签不能换项目。没有绕过。
   - 没有环：自己做自己的父标签由领域（422）和 `labels_not_own_parent_check`（存储一层，纵深防御）拒绝；两个标签互为父标签要求其中一个先有父标签再有子标签（或反过来），两层的规则拒绝。没有绕过。
   - 不分大小写的唯一：唯一索引在数据库上（`lower(name)`），建、改名都经它；删除之后名称空出来（部分索引）；两个并发的创建由索引拒绝第二个（第 3 对）。没有绕过。`lower()` 的大小写规则是数据库的 `LC_CTYPE`（第 7 节）。
6. **哪个检查在判定之前、哪个在之后**（6.7）。值的检查（名称、颜色）在事务之前，比项目、标签的 404 和 403 先答（矩阵"a value refused"：每一列都是 422）；父标签和名称的冲突在判定之后（矩阵"a parent with a parent""a name taken"：只有能写的列是 422、409，别的列 403 或 404：不能写标签的人不知道项目有哪些标签）。
7. **契约不说工作项**（P7a 第 3 节第 7 条）。第一次运行的 `Label` 说"which its work items carry"，P7b 删去；"标签下有工作项时"的行为由 M4 加入。
8. **`listLabels` 对已归档的项目照常列出**（3.19）。`listStates` 对已归档的项目答空列表（设计 2 节 P6），标签没有这样的规定：设计 2 节 P7、3.16、3.19 都没有说归档的项目的标签要隐去，Plane 也照常列出。契约、矩阵的已归档行（200，`listsTheLabels`）和故事 P7（Ops 归档之后列出、建、改）钉住。
9. **与停用、连带没有新的交错**（P6 spec 第 5 节 P7 一行）。标签的写是项目级的写（工作区 S → 项目 N），不改成员关系，不是约定六的增长或收缩；停用（工作区 N → 项目 N）、删除工作区的连带（工作区 N）与它们在工作区行上串行，不成环。
10. **清扫 37：已归档的项目、已删除的工作区、停用的调用者**。已归档项目的标签照常列出、建、改、删除（3.19），契约逐个写明，矩阵的已归档小表每个操作一行，盖戳的测试在 Web 归档期间跑三个标签的写（坑 6）。工作区在等锁时被删除：写答它自己的 404（`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile` 的"acme deleted"）；已删除的工作区下的标签在矩阵的 X 列（gone 的项目随它删除）。停用的调用者在认证一步被拒绝，到不了标签的操作。
11. **端到端的过渡（裁定 B4：照本条，第一种做法）**（Task 1 到 Task 11）。Task 1 建表之后，`expectProjectDeleted` 从目录读出 `labels`，要求它有随项目删除的行；故事要到 Task 5 才写得出标签，W3、P4 到 Task 11 才写。Task 1 在 `expectProjectDeleted` 的查询里排除 `labels`，注释说明为什么、到哪里；Task 11 删去排除，同时加 M5 的改动（之前删除的行不算）和 W2、W3、P4 的标签。中间的 Task 都不跑 `make e2e`（plan 只在 Task 1、11 跑），Task 1 的 `make e2e` 由这个排除通过。这是目录驱动的检查上的一个豁免（常设规则是不加），只在 Task 1–10 的树上，Task 11 删去；所以列出待裁定。B4 的条件：Task 11 的变异表显示排除的删去承重，`k-no-labels-step`（项目的标签不随它删除）和 `e2e-exclusion-kept`（留着排除）都在 P4、W3 失败；后者靠 Task 11 加的自查：`expectProjectDeleted` 从目录读出的表要等于有 `project_id` 列的每张表，留着排除的列表少了 `labels`（预检 PF-M1）。Task 2–10 不跑 `make e2e`，也不用 `expectProjectDeleted`（预检核对过）。另一种做法：排除在 Task 5（`createLabel` 的接口）删去，W2、W3、P4 的标签和 M5 的改动随之移到 Task 5，Task 5 加跑 `make e2e`；代价是 Task 5（plan 里 926 行）再加约 280 行的故事和夹具（W3 90、P4 78、W2 20、`assert/workspace.ts` 44 行，`api.ts`、`assert/project.ts` 的一部分），接口和故事混在一个 Task 里。
12. **矩阵的种子在新文件里**。`permission_matrix_seed_test.go` 389 行，没有余地；标签的种子（`matrixLabels`、`labels`、`seededLabels`）和行一起放在 `permission_matrix_labels_test.go`，文件的说明写明；`prepareMatrix`（`permission_matrix_test.go`）在 `seededStates` 之后调用它们。
13. **承接的条目**：

    | 来处 | 条目 | 落点 |
    |---|---|---|
    | P7a spec 第 5 节、review 第 6 节 | 共用路径：`updateLabel`、`deleteLabel` 经 `lockRowAndDecide`，`labelWrite` 照 `stateWrite`（F3）；`createLabel` 经 `lockAndDecide`，`listLabels` 经 `findAndDecide`；`Label.Place()`；路径的键的核对不再加行（S6） | 2.6、2.7、2.8、2.9 |
    | 同上 | 竞争和阶梯：`rowWrite` 的 `label`、`row()` 的标签一支，`createLabel` 照 `createState` 进 `rowWrites`（`creates`）；`underRow`、`rowPaths`（`/api/v0/labels/`）、`labelNamed`；交错 11 照 `stateWrittenHolding`、门和探测 | 2.10；Task 5、7、8、10 |
    | 同上 | 矩阵：标签的种子和查找（`seeded.label`、`seededLabels` 的前提），`rowsByID` 的 `labelRows`（与 `stateRows` 共用，PF-L3），每个操作一行 `variant: "archived"`；列的核对认得瞄准标签的格子 | 2.6、2.7、2.12 |
    | 同上 | 假实现：`fakeLabels` 嵌入 `*fakeStore`，按 id 的读与状态共用 `readRow`（PF-L3），列表的次序不是任何排序给得出的 | 2.6 |
    | 同上 | 第一次运行 T8–T10 的重放和手工合并：`project.yaml`、`deletion.go`、`fakes_write_test.go`、`state.go`/`project.go`（`checkLength`）；生成物重新生成；第一次运行没有改到的共用测试文件（盖戳、最先锁工作区、连接、矩阵、竞争、`fakes_write_test.go`） | 附录 A"对第一次运行的复审" |
    | P7a review 第 6 节，坑 1 | 第二个写自己的标签在等锁时变了 | 2.10 第 4 对 |
    | 坑 2 | 嵌入了查找的端口，说明不说整个在锁下 | `LabelFinder`、`LabelUpdater`、`LabelDeleter` 的说明（2.6、2.7、2.8） |
    | 坑 3 | 连接测试真的跑每一个决定所依赖的读，含被拒绝的层级改动 | 2.6、2.7（Icons 在 UI 下、Bug 放到 UI 下，各 422） |
    | 坑 4 | 承重的世界读回自己的设置 | `seedLabelsOf`（2.10）；`seededLabels`（2.6） |
    | 坑 5 | 每个映射到 HTTP 的字段在映射它的 Task 钉住 | Task 5（`createLabel` 的四个字段、`Label` 的九个字段）、Task 7（`updateLabel` 的四个字段，`parent_id` 的三种：给 id、给 `null`、不给） |
    | 坑 6 | 204 的写在矩阵里看不到效果；已归档项目上的效果由盖戳在归档期间跑 | 2.8；盖戳的三个标签的写在 Web 归档期间 |
    | 坑 7 | 失败的行要求 `errors.Is(err, context.Canceled)`、不是 `*shared.Error` | `failures_test.go` 的 `failed`（2.5） |
    | 坑 8 | 按唯一索引的确切名字映射冲突 | `labelTaken`（2.5）；`TestCreateLabelBreakingAnotherConstraintIsInternal`；`s19-any-key-taken` |
    | 坑 9 | 变异的锚点在它的 Task 的树上 | `mutanchors.py`：每个变异的锚点从它所在的 Task 起在每一份快照上恰好出现一次（附录 A） |
    | 坑 10 | 契约文件是否按资源拆 | 第 1 条 |
    | P7a review 第 6 节，其余的教训 | 每个列表的说明照写；问题的回答体每个包只写一次（`labelNotFoundJSON` 等常量）；端到端的账户经 `newAccount`；"工作区的每一行"经 `workspaceTables`（加 `labels`） | Task 9、5/7、11 |
    | P4a review 第 6 节 | 标签经 `deletion()` 的最后一步进 `DeleteWorkspaceProjects`，不是 `cascade()` 的一步；W3、P4 断言标签（裁定 S4） | 2.3；2.11 |
    | P4b review 第 6 节 | 每个标签的写经 `Locks`，在最先锁工作区的测试里有一行，按行寻址的第一步探测它的标签行；`deleteProjects`、`ProjectsDeleter`；手工维护的表（存储测试的表和种子、`deleteAll`（从端口反射，自动在内）、`failures_test.go` 的取消的步骤、`delete_project_test.go` 的每步失败、e2e 的 `workspaceTables`）；`keysTo` 的说明；`expectProjectDeleted` 只数之前未删除的行 | 2.3、2.6–2.8、2.11 |
    | P5b review 第 6 节 | 标签是共用路径的又一种行；竞争和锁强度的标签行 | 2.7、2.8、2.10 |
    | P6 review 第 6 节 | `deleteLabel` 连带子标签与约定五的例外，带最坏交错的探测 | 第 4 条 |
    | P2 review 第 6 节 | `pgtest` 的探测有期限，探测不存在的表立即失败：P7b 的探测都经 `pgtest.WaitForLockWaitOn`（有期限，按表名），没有新的探测函数；连续的探测各在不同的表上（清扫 47） | Task 7、10 |
    | P1–P7a 的常设规则 | 时钟在最后一把锁之后（`TestEachWriteReadsTheClockUnderItsLock` 三行）；锁下重读确认父行；`apitest.Main` 两个方向；端口的错误原样返回；角色按集合；关键词守卫（5 个命中都有例外，P7b 不加）；矩阵不加新的列表（`labels` 的写瞄准 `projectTables` 的列）；目录驱动的检查不加豁免 | 各 Task |

## 4. 验收标准（完成线，M3 设计 12 节 P7b）

- [ ] P7 的接口版本、断言标签的 W3 和 P4 通过，此前的每个故事仍然通过（`make e2e` 共 70 个：此前的 69 个，加 P7；原型和复现的副本不是仓库的检出，S3 读不到构建的提交号，单独失败，与 P7b 无关）。
- [ ] 交错 11 和同名的并发创建两种顺序 `-count=5 -race` 通过，没有 40P01（`race5.sh`：14 个测试各 5 次，670 个子测试，245 秒）。
- [ ] 迁移 `00014` 升、降、再升通过（`TestMigrationsGoUpDownAndUpAgain`）。
- [ ] 每个标签的写在 `TestEachWriteOnAProjectSharesItsWorkspaceFirst` 里有一行（按行寻址的第一步探测它的标签行）；少一行时完整性核对失败（`s34-first-lock-row`）。
- [ ] `project` 的 `apitest.Main` 两个方向核对通过（含两个标签码）。
- [ ] 整程序测试覆盖本 Phase 的操作（矩阵、最先锁工作区、盖戳、连接、竞争、每个写只写它的行）。
- [ ] 项目级矩阵的全部格子通过（后端的矩阵至此完整），已归档项目的小表含每个状态和标签的操作；格子数和耗时记下（853 格，1.53 秒）。
- [ ] `make gen-check`、`make lint-go`、`make test`、`make lint-web`、`make knip`、`make test-web`、`make e2e` 通过。

## 5. 不在 P7b 范围内

- `PROBLEM_MESSAGES` 的搬迁：P8。P7 的页面版本（标签设置的两层拖放、同名 409 落到名称字段）：P11。标签与工作项（`issue_labels`、"标签下有工作项时"）：M4。收集箱：M7。

**留给后面的 Phase**（各 Phase 的 spec 接过去；P7b 的 review 第 6 节照录）：

| Phase | 条目 |
|---|---|
| P8 | 两个标签码的前端文案（`PROBLEM_MESSAGES`、两份 `auth.json`）随搬迁照旧；标签的类型（`Label`、`LabelList`、`LabelCreate`、`LabelUpdate`）来自 `schema.gen.ts`，前端的 label store 改用它们（M1 留下的 `IIssueLabel` 等由 P8 的类型换掉） |
| P11 | P7 的页面版本：标签设置里建、拖进父标签（`label-drag-n-drop-HOC.tsx` 的两层规则照旧，服务端另外检查，3.16）、改名、调顺序（拖动发 `sort_order`）、删除（子标签随它）；同名（不分大小写）409 落到名称字段，层级的 422 显示服务端的说明（7.6）。网页建标签不发 `sort_order`，契约也不收（第 3 节第 2 条） |
| M4 | 工作项带标签（`issue_labels`，部分唯一约束 `(issue_id, label_id)`）；删除标签时工作项上的标签怎么办；工作项的写是否取工作区的 S 由 M4 决定（负责人 2026-10-02）；`labels` 的物理删除（60 天清理）按不带条件的三个索引找行 |
| M7 | 跨项目的标签列表只是把各项目的标签列在一起，不需要工作区级标签（11.5） |
| 收尾 | 3.20 的 P7b 两行（总体设计 5.3、`plane-diff.md`）由 Task 11 写好，收尾逐行核对；第 3 节第 1 条（契约文件）的裁定落到设计（第 2 条照设计 4.10，没有要落的）；部署说明写明数据库的 `LC_CTYPE`（第 7 节） |

## 6. 风险

| 风险 | 应对 |
|---|---|
| 两层的规则只在领域，数据库只挡自己做自己的父标签 | 每个设父标签的写都在项目的 N 之下读父标签和子标签（`checkParent`），交错 11 和第 2 对两种顺序在真实数据库上展示；`s25-*` 两个在锁前检查的变异在组合一层失败（附录 A） |
| `deleteLabel` 一条语句写多行 | 第 3 节第 4 条：全部写者都先持有项目的 N 或工作区的 N；第 5、6 对的探测 |
| 新表进入两个连带 | 目录驱动的组合测试（`keysTo` 读出全部外键）不加豁免；存储的 `deletionSteps` 从端口反射；W3、P4 在故事里读出标签 |
| 一个大的种子（矩阵每个项目三个标签） | `prepareMatrix` 经存储建出，`seededLabels` 核对每个项目恰好这些；写的格子各在自己的数据库副本上 |
| 第一次运行的代码没有清扫 | 全部复审、五十类清扫、变异逐个跑过（附录 A） |
| 持续集成（ubuntu）上的运行 | 原型和复现都在 macOS 上；合并之后看持续集成，P1–P7a 都是这样合并的 |

## 7. 已知的限制、交接和关闭条件

- **不分大小写跟随数据库的 `LC_CTYPE`**：`lower(name)` 按数据库的字符分类折叠大小写。官方的 PostgreSQL 镜像以 `en_US.utf8` 初始化（开发库 `nerve-dev-db-1`：`lower('BUG ÄÖÜ')` 是 `bug äöü`），非 ASCII 的大小写也折叠；以 `C` 初始化的数据库只折叠 ASCII，`Ä` 与 `ä` 是两个名称。Plane 的 `iexact` 同样跟随数据库。部署说明（M8）写明用 UTF-8 的 locale 初始化。
- **`createProject` 锁工作区的回答没有键可核对**（P7a 第 3 节第 1 条）：照旧。
- **标签的锁下重读只在组合一层钉住**（P7a review 第 7 节的 P6′ 同类）：单元的假实现回答不了等锁时变了的父标签；交错的七对在组合一层。

| 交接 | P7b 处理的条目 | 留下的条目 |
|---|---|---|
| P7a spec 第 5 节、review 第 6 节的 P7b 一行 | 第 3 节第 13 条 | 无 |
| P4a review 第 6 节 | 第 3 节第 13 条 | 无 |
| P4b review 第 6 节 | 第 3 节第 13 条 | 无 |
| P5b review 第 6 节 | 第 3 节第 13 条 | 无 |
| P6 review 第 6 节 | 第 3 节第 4 条 | 无 |

**M3 设计 13.1 的关闭条件**：13.1 没有落在 P7（P7a、P7b）的一项（设计 12 节 P7b"关闭：没有"）。没有放不下的条件。

## 附录 A：原型验证记录（2026-10-06）

原型在 `$M3TMP/p7bproto`：照裁定 S4，第一次运行的标签部分（`p7snap/T7`→`T8`→`T9`→`p7proto` 的三段差异，`p7btools/logs/run1-*.diff`）放在 P7a 的合并（`0deb8c34`）之上重放（`p7proto` 照裁定 S5 留着没动）；`node_modules` 由仓库的安装目标装好（不用符号链接）；Go 1.27.1（`toolchain`）、Node 24、pnpm 11.10.0、Docker；`bin/golangci-lint` 2.13.2。复现的基础是 `0deb8c34` 的一份新的副本（`p7bbase-0deb8c34.tar`，快照 T0）。做法照 P7a：每个 Task 做完时存一份源文件的快照（`$M3TMP/p7bsnap/T1`…`T11`），plan 的代码块由脚本从相邻两份快照的差异生成（`p7btools/mkblocks.py`），生成的文件不进块，按 SHA-256 核对（`gensha.py`）。每个 Task 的改动记成步骤（`capture.py`：新文件整份、删除的文件、改动的文件逐处替换），`build.py` 按步骤从前一份快照建出这个 Task（`gofmt`、`make gen`、门禁、快照）；之后的修改写进拥有那个文件的 Task 的步骤，从那个 Task 起重建。`assemble.py` 组装 plan 并核对每个块放了一次、每个文件在文件表里、每个 Task 在约 1,500 行以内、没有 HTML 实体。

**第一次运行的重放和手工合并**（`run1_overlap.py`：第一次运行改的每个源文件，P7a 的合并里是否还是第一次运行的 T7 的样子）：39 个文件。18 个与 T7 相同，差异照原样应用（迁移的两个测试、`sqlc.yaml`、授权、连带的语句和存储方法、`ports.go`、单元和存储的连带测试、`delete_project_test.go`、组合的两个删除测试、`workspace.yaml`、`openapi.yaml`、`access` 的规则和测试、`actions.go`、`authentication.helper.ts`）；12 个是新文件，照原样取来（迁移、`labels.sql`、存储方法和两个存储测试、领域和它的测试、`label_ports.go`、`label_parent.go`、`create_label.go`、`list_labels.go`、`fakes_label_test.go`）；9 个与 P7a 改过的文件相交，手工合并，都是相邻的文字：

- `api/modules/project.yaml`：P7a 改了状态操作的描述和 `State` 的两句；标签的路径、参数和结构照旧接在状态的之后（照新的切分分到 Task 1、5、7、8、9）。
- `project/app/deletion.go`：P7a 的说明写着"P7b adds the labels at the end"；P7b 加上那一步，删去这半句。
- `project/app/fakes_write_test.go`：P7a 改了 `rowReads` 的说明；P7b 加 `DeleteLabels`（Task 1），之后接上标签的假实现（Task 4、6）；Task 4 加 `readRow`，状态和标签的按 id 的读共用它（修订的 PF-L3，下文）。
- `project/adapter/postgres/failures_test.go`：P7a 加了状态的方法；标签的方法接在之后。
- `project/domain/errors.go`、两份 `auth.json`：P7a 加了状态码；标签码接在之后。
- `project/domain/project.go`、`state.go`：第一次运行的 T9 把 `checkShortText` 从 `state.go` 移出给标签用；P7a 的 T2-d 已把状态的检查改成 `checkLength`（不空白、不超长）。P7b 拆出 `checkMaxLength`（只看长度）、加 `checkRequiredText`（`checkLength` 再 `checkText`），状态的名称和颜色、标签的名称都用它，删去 `checkStateText`；标签的颜色用 `checkMaxLength` 和 `checkText`。没有第二个长度检查（2.4）。

生成物都重新生成。第一次运行没有的：`updateLabel`、`deleteLabel` 的用例，四个操作的 HTTP 和接线、矩阵、最先锁工作区、盖戳、连接、竞争、交错 11 和其余六对、每个写只写它的行、故事、文档。

**对第一次运行的复审**（第一次运行的标签部分没有清扫；这里逐条复审之后在根上改了的，按缺陷类别；`run1diff.py` 列出每个接过来的文件改了几行）：

- **迁移、存储方法**：照旧（`00014`、`labels.go` 一字未改）。`DeleteLabel` 的语句说明加上约定五的理由（第 3 节第 4 条）。
- **清扫 22**：`checkParent` 原来不核对读到的父标签是不是问的那个；现在读成另一个 id 是错误（`s22-parent-id`）。
- **清扫 21**：存储的 `TestCreateLabelNameTaken`、`TestUpdateLabelNameTaken` 原来只看 `errors.Is`，包在别的错误之后也会通过；现在要求第一个 `*shared.Error` 就是 `project.label_name_taken`（`s21-taken-wrapped`）。`TestUpdateLabelWritesNoDeletedLabel` 原来只排除名称被占用；现在要求不是任何领域错误。
- **清扫 26**：存储的 `TestUpdateLabel` 的空补丁原来是第一行，它保留的父标签、颜色、`sort_order` 正是种子的值，也可能是列的默认值；现在跟在给了全部字段的补丁之后（`s26-ul-*`）。
- **清扫 45**：同一行里"回答的 `updated_at` 是给的时刻"已由"回答与存下的相同"和审计列的核对蕴含，删去。
- **清扫 16**：单元假实现的列表原来按 `sort_order` 倒序，按 id 排的用例也会通过，`GreatestSortOrder` 的假实现靠这个次序取第一个；`fakeLabels.of` 改按名称（既不是 `sort_order` 的两种次序，也不是 id 的），`GreatestSortOrder` 的假实现自己找最大的；`TestListLabels` 先核对存储的次序不是任何一种排序给得出的。
- **清扫 50**：`TestCheckParent` 的说明原来说"有子标签又以自己为父标签的标签因后者被拒绝"，表里没有这一行，删去；`TestCheckLabelReportsEveryField` 加"两个问题的字段报第一个：太长先于 NUL"两行（名称、颜色），说明随之；`CreateLabel`、`ListLabels`、`checkParent`、`LabelFinder` 的说明照代码改写。
- **端口**：`label_ports.go` 照用到的 Task 分开：`LabelRow`（Task 3）、`LabelFinder`、`LabelCreator`（Task 4）、`labelWrite`、`LabelUpdater`（Task 6）、`LabelDeleter`（Task 8）、`LabelLister`（Task 9）；存储测试的 `addLabel` 改用 `states_test.go` 的 `workspaceOf`。

**清扫之后的加强**：变异按层跑完之后（单元、结构、存储、组合、端到端，`stage-*-before-hardening.json`），只在单元或存储一层被发现、而组合一层或故事本可以展示的，改测试，不改产品代码；改动写进拥有那个文件的 Task 的步骤，快照由 `build.py` 从 Task 10 起重建，最终原型的门禁重跑，单元、组合和 P7 三层整层重跑（`stage-unit.json`、`stage-composed.json`、`stage-e2e-P7.json`）：

- `TestEachLabelWriteChangesItsRowsAlone`（Task 10）加四步：在 Docs 下建 Fonts、只删除 Fonts、把 Docs（它下面唯一的标签已删除）放到 Bug 下、删除 Docs（之前删除的 Fonts 保留它的删除）。`s1-hc-deleted`（`HasChildren` 把已删除的子标签也算上：Docs 会被当成有子标签，422）和 `s1-dl-deleted`（`DeleteLabel` 把之前删除的子标签再删一次：Fonts 的行会变）原来只在存储一层（清扫 44）。
- 交错的第 7 对（Task 10）：在顶层建 QA 与在 Bug 下建 Icons，两个都成功，第二个的时刻不早于门打开的时刻。`c-create-early`（`createLabel` 在锁之前读时钟）原来只在单元一层：交错里第二个写是创建的那几对，第二个都被拒绝，读时刻的锁强度测试由 `creates` 跳过 `createLabel`。加锁的性质只在单元一层，是缺口，现在补上。
- `seedLabelsOf`（Task 10）原来由 bob 写，他是 Web 的管理员，却只是 Ops 的成员，规则不让他写 Ops 的标签（清扫 35、43）；改为 alice，两个项目的管理员。
- 故事 P7（Task 11）：
  - 原来 admin 把 Widgets 移到 Bug 之前（55535），删除的 Bug、Widgets 都在 Feature 之前，"删除了的标签的位置不再算"展示不出（清扫 15、50），`s1-gs-deleted` 只在存储一层。现在 admin 只改 UI 的名称，再把 Feature 移到 Bug 之前（55535）：删除的 Widgets 在 85535，最大，bug 建在 Feature 之后的 65535。成员的列表照旧不是建的次序（Feature、Bug、Widgets）。admin 只给 `sort_order` 的修改留着 Feature 的颜色（`s26-ul-color-empty` 原来只在存储一层），只给名称的修改留着 Widgets 的父标签。
  - 拒绝里加 QA 的颜色 256 个字符，422 `color too_long`（`v-color-unchecked` 原来只在单元一层）。
  - 已归档的 Ops 里，Runbook 建在 Bug 下，最后移到顶层，`parent_id: null`（`s4-update-null-ignored` 原来只在单元一层，`s26-ul-null-kept` 只在存储一层）。Runbook 原来建在 ann 给的 70000.5；裁定 B2 之后请求不给位置，它在 Bug 之后的 75535（修订）。
- 裁定 S6（复审时发现）：原型的 `TestUpdateLabelRefuses`、`TestDeleteLabelRefuses` 各有一行"重读成另一个 id 的标签"，那是共用路径的核对，P7a 已在 `lock_test.go` 钉过一次；两行删去，说明随之，`TestDeleteLabelRefuses` 的 `set` 不再需要标签的假实现。这两个文件是 Task 6、8 的新文件，之后的 Task 不改它们，`gofmt`、`make gen` 也不碰测试文件，所以 Task 6–11 的快照照原样换上新的文件（`f1_snaps.py`），与 `build.py` 重建的结果相同；复现逐 Task 跑过它们。

组合一层的"（Task n 起）"：`markers.py` 把每个在组合一层被发现的变异放在它所在的 Task 起的快照上（Task 1、5、7、8、9、10：连带、四个操作的接口、并发的测试各自加进来的地方），只跑最终的树上发现它的那些测试，记下每个测试从哪个 Task 起失败（`logs/markers.json`）；plan 的变异表照这个标。

**修订（2026-10-06：控制者的裁定 B2，预检的 H0/M2/L4）**：预检扫的是 `c2943340`；修订在原型上改、按 Task 的步骤从改动的第一个 Task 起重建快照，之后全部重跑（变异按层、`markers.py`、`mutanchors.py`、`race5.sh`、门禁和 `make e2e`、逐 Task 复现）。

- **裁定 B2**：`createLabel` 不收 `sort_order`（第 3 节第 2 条）。契约的 `LabelCreate` 去掉它，`createLabel` 的描述写明新标签在项目的其余标签之后（Task 5；生成物重新生成，Task 5、7、8、9 的 SHA-256 表重新钉住）；`domain.LabelCreate` 去掉 `SortOrder`（Task 2）；用例在锁下每次读 `GreatestSortOrder`、`SortOrderAfter`，删去 `sortOrder`，`labelCreated` 不再有没给的一支（Task 4）；处理函数不再传它（Task 5）。`TestCreateLabel` 的"at its sort order"一行删去；HTTP 测试原有的 `{"name":"A","sort_order":"1"}`（类型不对）改为 `{"name":"A","sort_order":1}`，是多余字段的 400，不另加行。故事 P7 的标题改为"listed, created after its labels, and changed"，Runbook 在 Bug 之后的 75535。`plane-diff.md` 的"新建标签的顺序"一行保留、改写（PF-L4）。变异 `q-given-ignored`、`s4-create-drops-sort` 删去，锚在 `u.sortOrder` 上的 `s2-create-greatest`、`q-greatest-ignored`、`s25-create-parent-unlocked` 和 HTTP 的 `s4-create-drops-parent` 改锚。
- **PF-M1（裁定 B4 的条件）**：Task 11 的 `expectProjectDeleted` 核对它从目录读出的表就是有 `project_id` 列的每张表（说明写明）。`e2e-exclusion-kept`（留着 Task 1 的 `AND c.conrelid <> 'labels'::regclass`）在 P4、W3 失败，失败的正是这条自查（"the tables under projects, as their project_id columns name them"）；`k-no-labels-step` 也在 P4、W3 失败。Task 11 有了变异表，列这两个。
- **PF-M2（清扫 25）**：交错的每一对之后核对 Web 未删除的标签没有两个同在一个 `sort_order`。`s25-create-greatest-unlocked`（在锁之前读最大值，新标签照它放）在单元（调用次序）和组合一层（第 7 对的两种顺序）失败；这条核对也让 `q-greatest-ignored`、`s1-gs-asc`、`s26-cr-sort` 多在交错里失败。
- **PF-L1**：`mut_tables.py` 按跑出这个变异的那一层的包找测试（同名的存储、用例、HTTP 测试分开；用例和 HTTP 同名时，变异自己的包有它就是它的，否则是用例的），Task 3 原来标错的 23 行不再标（存储的测试在 Task 3 就有），`s4-label-parent-null` 的 `TestUpdateLabel` 标 Task 7；组合一层的"（Task n 起）"由 `markers.py` 在修订之后的快照上重新量出（68 个变异，`logs/markers.json`），没有手标的。
- **PF-L2**：矩阵的"a parent of another project"改用 acme 的已归档项目的 Bug（同一工作区的另一个项目）；`r-parent-same-workspace`（`createLabel` 只看父标签的工作区）在单元和组合一层（`createLabel, a parent of another project` 的 PA、PM+WA 两格）失败。格子数不变。
- **PF-L3**：`fakes_write_test.go` 的 `readRow[R]`（记调用、按 `errs` 失败、锁下的重读照 `reread` 失败、消失或换项目，回答 `answersAs`），`fakeStates.StateByID`、`fakeLabels.LabelByID` 都只取出这一行交给它（Task 4，标签的那份第一次出现的地方）；`permission_matrix_project_test.go` 的 `rowsByID{path, notFound, find}` 的 `of`、`ofArchived`、`to`，`stateRows` 代替 P7a 的 `ofState`、`ofArchivedState`、`toState`，标签的是 `labelRows`（Task 7）。`of`、`GreatestSortOrder` 两份不合并：合起来要为两种不相干的行各传取项目和次序的函数，比两段各九行左右的循环难读。P7a 的测试不变，由碰到这些帮手的 P7a 变异在改之前的树（`0deb8c34`）和之后的原型上各跑一次核对（`p7btools/amend/pfl3.py`、`pfl3_cmp.py`，`logs/pfl3-*.json`）：单元一层（`project/app` 包）之前跑的 393 个测试和子测试之后都在（之后共 488 个），`s2-delete-reread`、`s2-path-reread-404`、`s22-row-id`、`p-no-reread`、`p-project-unchecked`、`p-row-404-forbidden` 和状态的写的 404 换成项目的 404（矩阵的格子 `notFound`）各让 P7a 的同样的测试失败（5、12、5、100、10、16、22 个），之后多出的只是标签的测试；组合一层（`TestPermissionMatrix`）之前跑的 723 个测试和子测试（行和格子）之后都在（之后共 855 个），`p-row-404-forbidden` 和状态的 404 各让 P7a 的同样的 52、82 个失败（之后多出的 4 个是标签的行），其余五个在这一层两边都不失败（它们只在单元一层被发现）。
- **清扫的审计**：清扫 1 的列表加上 `ListLabels` 同值时按 id 的次序（`s1-ll-tiebreak`，存储的 `TestListLabels`）和名称的 CHECK（`s1-check-name`，结构和存储一层），两个原来就被发现。
- 变异 105 个减 2、加 5，共 108 个，都被发现；按层：单元 43/43、结构 4/4、存储 49/57、组合 68/88、P7 46/54、P4 4/7、W3 7/7（存储、组合和故事上没发现的都在别的层被发现，按设计的见上文）；`mutanchors.py`：每个变异的锚点从它所在的 Task 起在每一份快照上恰好出现一次。

**最终的原型**（`gates.sh`，2026-10-06 13:11–13:13，修订之后在最终的树上跑的门禁（`logs/amend-final-*.log`、`amend-final.summary`）；之后原型只被变异、`markers.py` 和 PF-L3 的运行临时改动（Go 的文件经 `-overlay`，别的文件跑完复原），逐 Task 复现的终态与它逐个文件相同）：`make gen` 之后生成物没有差异；`make lint-go` 两段 `0 issues.`；`make test` 42 个 `ok`；`make lint-web`（关键词守卫 5 个命中都有例外，54 个任务）；`make knip`；`make test-web`（16 个任务）；`make e2e` 70 个故事中 69 个通过，P7 和断言标签的 W2、W3、P4 在其中；S3 因原型不是仓库的检出、构建没有提交号而失败（P4b 的 F4，读 `commit`，与 P7b 无关，P7a 的原型同样）。

**迁移**：`TestMigrationsGoUpDownAndUpAgain`（14 个迁移，降到零、再升）通过；`00014` 的 Down 删表。

**矩阵**：`TestPermissionMatrix` 853 格全部通过（P7a 结束时 721 格，P7b 的 14 行加 132 格：10 行各 12 格、4 个已归档的小表各 3 格），1.53 秒（修订之后的 `race5.sh` 的第二段，`logs/matrix.log`）。

**交错和竞争**：修订之后的 `race5.sh`（`logs/race5.log`）：14 个测试各 `-count=5 -race`，70 次顶层、670 个子测试全部通过，245 秒，没有 40P01，没有数据竞争。其中：交错 11 和其余六对（`TestLabelWritesOnOneProjectSerialize`，14 个子测试：两层的交错、建子标签与移走父标签、同名不分大小写的两个创建、同一标签移两次、约定五的探测两对、两个都成功的创建，各两种顺序；每一对之后核对 Web 的标签各在自己的 `sort_order`）；按行寻址的竞争和锁强度（成员关系、状态、标签的写）；最先锁工作区；状态和标签的写各只写它的行；P7a 的交错 10 和"组里最后一个"的两对；P5b 的七个交错。P7a 的状态测试和 P5b 的成员测试没有改（P7b 只给存储的 `states_test.go` 抽出一个共用的 `workspaceOf`、给三处说明加上标签；`project_row_races_test.go` 加标签的行，成员关系、状态的行照旧），全部通过。

**逐 Task 复现**：修订之后的 `replay_all.sh`（`p7btools/replay-logs`；修订之前的留在 `logs/pre-amend/replay-logs`）：`0deb8c34` 的一份新的副本（`p7bbase-0deb8c34.tar`），按 plan 的顺序应用 11 个 Task 的块，照原样运行每个 Task 的 `Run:` 命令，一共 1,276 秒，没有重跑。每个 Task 之后 `make lint-go` 两段 `0 issues.`、`make test` 42 个 `ok`；`make gen`（Task 1、5、7、8、9）和 `make gen-go`（Task 3）之后生成物与快照相同，每个 SHA-256 和行数与 plan 的表相同；`make lint-web`、`make knip`、`make test-web` 在 Task 1、5、7、8、9、11 通过；Task 10 的 `-count=5 -race` 208 秒通过，没有 40P01，没有数据竞争；`make e2e` 在 Task 1 是 69 个中 68 个、在 Task 11 是 70 个中 69 个通过，失败的只有 S3（F4）。最后：`treediff` 与原型比较 3,130 个文件，没有差异；`planapply check` 从 `0deb8c34` 起放下全部 291 个块（267 处替换、24 个新文件）。复现的副本不是 git 检出，`make gen-check` 在那里问不了 git 生成物是否已提交（git 失败，目标仍然返回 0），由 gen-same 代替：Task 11 之后 `make gen` 的每个生成物逐字节相同（0 个差异）。

**五十类清扫**（brief 的缺陷类别；每个变异一个 `go test -overlay` 或 `go build -overlay`，树不动，Go 以外的文件（迁移、契约和打包的契约）原地改、跑完复原；`mutlevels.py` 在变异写的每一层各跑一次：单元（`project` 的 `app`、`domain`、`adapter/http`，`access/domain`）、结构（`server/migrations`）、存储（`project/adapter/postgres`）、组合（`bootstrap`）、端到端（单独运行的故事 P7、P4、W3）；"层"是它被发现的每一层）：

| 清扫 | 大小 | 结果 | 层 |
|---|---|---|---|
| 1 每个 SQL 谓词 | 七条语句和连带的一条的 19 个谓词和次序（`LabelByID` 2、`ListLabels` 4（含次序和同值时按 id 的次序）、`GreatestSortOrder` 3（含取最小的）、`HasChildren` 2、`UpdateLabel` 2、`DeleteLabel` 3（含不带子标签）、`DeleteLabels` 3），每个去掉，参数照旧绑定；迁移的唯一索引 2（区分大小写、算已删除的）、CHECK 2（自己做自己的父标签、空名称）；同值的次序和名称的 CHECK 由预检的清扫审计补进列表 | 23/23。组合一层 13 个（`s1-hc-deleted`、`s1-dl-deleted` 在加强之后），端到端 P7 14 个、P4 或 W3 3 个；只在存储一层的 `s1-ul-deleted`（组合一层等价）、`s1-ll-tiebreak`（同值的次序只有 `updateLabel` 给得出），只在结构和存储一层的 `s1-check-own-parent`、`s1-check-name`（纵深防御），按设计（下文） | 结构；存储；组合；端到端 |
| 2 端口调用的错误 | 四个用例和 `checkParent` 的每个端口调用注入失败（`Test…ReturnsEachFailure`），7 个吞掉或越过的变异 | 7/7，都只在单元一层（按性质，下文） | 单元 |
| 3 写不动的行 | 存储的写比较被写的行之外每一行每一列（`tableRows`、`columns`）；`UpdateLabel` 改写 `created_at`、三个写保留原来的 `updated_at`，4 个变异 | 4/4：`s3-ul-keeps-moment`、`s3-dl-keeps-moment` 在存储和组合一层，`s3-cl-keeps-moment` 在存储和 P4、W3；`s3-ul-created` 只在存储一层（审计列，按设计，下文） | 存储；组合；端到端 |
| 4 组合根的接线 | `project.New` 接上的三个写：停住的时钟 3 个；HTTP 的映射：建的丢掉父标签、颜色，改的丢掉父标签、null 的父标签当作没给，回答的标签一律在顶层，5 个（`createLabel` 不收 `sort_order`，裁定 B2） | 8/8：停住的时钟 3 个在组合一层（盖戳的测试）；HTTP 的映射 5 个在单元一层（HTTP 的测试），其中 4 个也在组合一层和 P7，`s4-update-null-ignored` 在单元和 P7（加强之后） | 单元；组合；端到端 |
| 5 安全性质在真实环境上 | 四行规则各放宽或收紧一格（4 个），`listLabels` 不判定（1 个） | 5/5，都在单元和组合一层（矩阵），`s5-update-members`、`s5-create-members` 也在 P7 | 单元；组合；端到端 |
| 6 每句文档 | 每句说明、契约描述、故事标题、差异清单的行对照代码 | 改了的见"对第一次运行的复审"的清扫 50 一条和"清扫之后的加强" | 审阅 |
| 7 反例里没有随机 | P7b 的测试 | 表都是切片；`worldLabels` 是 map，只做"名称 → 父标签"的查找和比较，种子按固定的切片次序建出 | 审阅 |
| 8 决定所依赖的读在调用者的事务里 | 存储的七个方法（连带的一个在内）各改为走池 | 7/7，都在组合一层（`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`；连带的一个也在提交时被拒绝的删除的测试） | 组合 |
| 9 锁顺序在真实环境上 | 每个标签的写：工作区 S → 项目 N（`createLabel` 经 `lockAndDecide`，两个按行寻址的经共用路径） | `p-create-share-project`、`s32-wait-on-label` 被发现；最先锁工作区的测试（`NOWAIT`）和锁强度的测试（`lockOn`）各有标签的行 | 组合 |
| 10 承重的种子行有前提 | 矩阵的标签种子（每个项目恰好 Bug、UI、Feature，UI 在 Bug 下，`sort_order` 65535、75535、85535）、`memberWorld` 的 `worldLabels`、存储测试的次序 | 都有前提：`seededLabels`、`seedLabelsOf` 读回核对、存储测试的 `ctid` 核对 | 审阅；组合 |
| 11 每条拒绝路径 | 没有调用者什么都不读；判定失败原样返回；不能写的人对无效的目标得到 403 或 404（矩阵"名称已占用""父标签有父标签""父标签在别的项目""有子标签的标签给父标签"各行的非管理员列） | 每个用例的拒绝表第一行；矩阵 | 审阅 |
| 12 与结束、删除的竞争（组合） | 标签的三个写：按行寻址的两个各五种（这一行删除、移到 Ops，调用者的成员关系结束，Web 删除，acme 删除），`createLabel` 后三种 | 都答 404，不答 403；什么都不改；等待由探测确认（2.10）；`p-label-404-state` 被发现 | 组合 |
| 13 锁强度在组合一层看得到 | `lockOn` 读出每一步的锁：项目 N、工作区 S；按行寻址的写不锁项目以外的行 | `p-create-share-project` 被发现（交错的第 3、7 对，两个创建） | 组合 |
| 14 每个"由谁"可以失败 | 存储的写之前由 maker 写过；组合的盖戳先由 bob 写过（`bobs`）；3 个保留写者的变异 | 3/3，都在存储和组合一层（盖戳的测试、连带的删除的测试），也在 P7 或 W3 | 存储；组合；端到端 |
| 15 标题的每个说法都有展示 | P7、P4、W3、W2 的标题逐句 | 每一句都有步骤；P7 的"the deleted labels' places no longer counted"原来展示不出（删除的标签的位置都在 Feature 之前），改为 Feature 移到最前、Widgets 留在最后（"清扫之后的加强"） | 审阅 |
| 16 单元假对象的回答顺序 | 假实现的列表、`GreatestSortOrder` | 改按名称，测试先核对不是任何排序给得出的（"对第一次运行的复审"） | 审阅 |
| 17 判定之前的代价有界 | 每个写在判定之前：领域的检查（不读库）、至多一行的读和两把锁；读：一次查找 | 有界 | 审阅 |
| 18 契约描述只说代码做的 | 四个操作的描述和 `Label` 的字段；3 个变异（少声明、多声明） | 3/3：少声明的 2 个在单元（`apitest.Main`）和组合一层（读问题码的测试）；多声明的 `s18-update-extra-default` 只在单元一层（按性质） | 单元；组合 |
| 19 每个新的存储方法有失败测试 | 八个方法（连带的一个在内），10 个变异（名称被占用答成内部错误、重复的 id 答成名称被占用） | 10/10，都在存储一层；`s19-name-taken-internal` 也在组合一层和 P7；其余 9 个按性质只在存储一层（下文） | 存储；组合；端到端 |
| 20 相关谓词的集合形式、"别的"算进自己 | `HasChildren` 的 `EXISTS` 只看这个标签的子标签；`UpdateLabel` 改名时自己的名称：唯一索引只比较别的行（自己换大小写可以，存储测试一行） | 不适用于集合形式；"别的"由索引保证 | 审阅 |
| 21 拒绝和失败钉住第一个 `*shared.Error` | 每个拒绝表用 `outcome.check`；名称被占用包在 404 之后、`updateLabel` 在找到标签之后才查值（2 个） | 2/2：`s21-taken-wrapped` 在存储、组合一层和 P7，`s21-patch-checked-late` 在单元和组合一层（矩阵"值被拒绝"的行） | 单元；存储；组合；端到端 |
| 22 端口的回答对得上所问 | 两个写的回答、`checkParent` 读到的父标签（3 个）；重读的 id 由共用路径钉过一次（裁定 S6） | 3/3，只在单元一层（按性质，裁定 S6） | 单元 |
| 23 组合的夹具跨第二个工作区、第二个项目 | 矩阵（acme、gone、other）、`memberWorld`（acme 的 Web、Ops，beta 的 Lab），Web、Ops 各有同名的标签 | 交错、每个写只写它的行、竞争都核对 Ops 的标签不变；故事 P7 读 Elsewhere 的两个项目 | 组合；端到端 |
| 24 每条规则的每一半在组合一层有反例 | 两层的四种拒绝和"移到顶层不受限"、父标签是同一项目的（不只是同一工作区：矩阵的同一工作区另一项目的 Bug，修订的 PF-L2）、不分大小写的唯一（建、改名，已删除的、别的项目的不算）、`sort_order`（最大值加 10000、65535） | 规则 10/10、`sort_order` 3/3；其中 10 个在组合一层（矩阵的行、交错），`r-parent-self`、`v-color-unchecked`、`q-first` 在单元一层和 P7（故事的拒绝、Ops 的第一个标签）：真实环境的反例在端到端 | 单元；组合；端到端 |
| 25 判定和执行在同一把锁下 | 父标签和子标签在锁前读（2 个）；最大的 `sort_order` 在锁前读（1 个，修订的 PF-M2：交错的每一对之后核对 Web 的标签各在自己的 `sort_order`） | 3/3，都在组合一层（交错 11 的第 1、2、5、6 对；最大值的那个在第 7 对的两种顺序，也在单元一层的调用次序） | 单元；组合 |
| 26 "不变"用不会碰巧得出的值 | `UpdateLabel` 没给的名称、颜色、`sort_order`、父标签，给 null 的父标签；`CreateLabel` 丢掉颜色、父标签、`sort_order`（8 个） | 8/8，都在存储一层；5 个也在组合一层，`s26-ul-color-empty`、`s26-ul-parent-top`、`s26-ul-null-kept` 在存储一层和 P7（加强之后） | 存储；组合；端到端 |
| 27 故事里的"不动"在动作之前读、之后核对 | P7 每一步之后 `expectLabels`，每组拒绝之前（上一步之后）和之后各一次；P4、W3 删除之前、之后 | 都有 | 审阅 |
| 28 只在端到端被发现的性质另有 Go 的测试 | 变异表里只在端到端失败的 | 在端到端被发现的 53 个（P7 46 个，P4、W3 7 个）中 52 个另在单元、结构、存储或组合一层被发现；`e2e-exclusion-kept` 改的是端到端的夹具自己，只有故事看得到，它守的性质（项目下的每张表随项目删除）在组合一层由目录驱动的删除测试核对 | 审阅 |
| 29 每个能等锁的等待都有期限 | 交错的 context 10 秒；竞争的 `receiveWithin`；探测 5 秒；`pgtest.Soon` | 都有：交错的 context 10 秒，竞争的回答 `receiveWithin` 10 秒，探测 `WaitForLockWaitOn` 5 秒，持有事务时的语句和种子在 `pgtest.Soon`（5 秒）上；没有不带期限的等待 | 审阅 |
| 30 每个输出、问题的细节和补救对每个收到它的调用者都成立 | 两个标签码的说明、四种 `parent_id` 的说明 | 收到 409、422 的都是判定通过的项目管理员；404 的说明"或者你看不到它的项目"对看不到项目的调用者成立 | 审阅 |
| 31 检查和执行之间的 gate 在检查的事务之外 | 交错的门在第一个写的写之后、提交之前 | `s25-*` 在门上失败 | 组合 |
| 32 每个探针有反例 | 等待从项目行挪到标签行（结果不变） | 1/1，在组合一层：`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength` 的探测失败 | 组合 |
| 33 goroutine 上不调用 `t.Fatal` | P7b 的 `run(...)` | 闭包里只调用用例、只送回答案 | 审阅 |
| 34 "没有替身"、"完成"的说法全包 grep | 完整性核对：契约的每个项目级的写在最先锁工作区的测试里有一行；矩阵覆盖每个操作；`expectProjectDeleted` 读的表就是有 `project_id` 列的每张表（修订的 PF-M1，裁定 B4 的条件） | 2/2：`s34-first-lock-row` 在组合一层（`TestEachWriteOnAProjectSharesItsWorkspaceFirst` 的完整性核对）；`e2e-exclusion-kept`（留着 Task 1 的排除）在 P4、W3（自查"the tables under projects, as their project_id columns name them"） | 组合；端到端 |
| 35 "由 X 写"的种子是规则允许的 | `worldLabels`；矩阵的种子；组合的删除测试的 Bug、UI | `worldLabels` 原来由 bob 写，他是 Web 的管理员，却只是 Ops 的成员，规则不让他写 Ops 的标签；改为 alice（两个项目的管理员）。删除测试的由 alice（项目的管理员）。矩阵的种子照 P7a 的状态种子由工作区的管理员写，他不是矩阵项目的成员：标签的写者可以是之后离开项目的管理员，这是规则到得了的状态，矩阵也不读种子行的写者 | 审阅 |
| 36 别的操作或两步到达同样的结果 | 两层、没有环、不分大小写的唯一 | 没有绕过（第 3 节第 5 条） | 审阅 |
| 37 归档项目、删除的工作区、停用的调用者 | 每个操作 | 第 3 节第 10 条 | 审阅 |
| 38 故事的每个拒绝之前先读 | P7 的九个拒绝；删除之后的两个 404 和一个 422 | 每组之前（上一步之后）`expectLabels`，之后再读一次 | 审阅 |
| 39 测试里的排序与排序规则无关 | `expectLabels`（`sort_order`，再按名称 `COLLATE "C"`）；`labelsOf` 是 map | 与排序规则无关 | 审阅 |
| 40 每个交错的结果写明每个结束由谁 | 七对：两个写都由 bob；结果读出每个标签的父标签 | 没有成员关系的结束；写者由盖戳的测试钉住 | 审阅 |
| 41 每个写多行的语句与同一些行的其他写者 | `DeleteLabel`（这个标签和它下面的）、`DeleteLabels`（连带） | 父行的锁排除它们，约定五的例外不适用；最坏的交错由第 5、6 对展示（第 3 节第 4 条） | 组合 |
| 42 锁语句之后的写只写锁住的行 | P7b 没有先锁一批行再写的语句 | 不适用 | 审阅 |
| 43 每个"由 X"可以失败 | 同清扫 14 | 3/3，都在存储和组合一层（盖戳的测试、连带的删除的测试），也在 P7 或 W3 | 存储；组合；端到端 |
| 44 "保留"、"还有"读状态 | "有子标签"读未删除的；"之前删除的保留它的删除"读 `deleted_at` 和 `updated_at` | `TestHasChildren`、`TestDeleteLabel`（存储）、`TestEachLabelWriteChangesItsRowsAlone`、P4 | 存储；组合；端到端 |
| 45 没有不会失败的断言 | 共用路径的核对在各用例里的重复（裁定 S6）；存储 `TestUpdateLabel` 的回答时刻 | 删去重复和被蕴含的核对（"对第一次运行的复审"） | 审阅 |
| 46 测试一侧的等待都有期限 | 种子、读回、持有事务时的语句 | 都在 `pgtest.Soon` 的期限上 | 审阅 |
| 47 连续的探测在同一张表上要指名 | 交错每个子测试一次探测（`projects`）；竞争照 P7a | 没有连续的同表探测 | 审阅 |
| 48 堆序或索引序的夹具有前提 | `ListLabels` 的存储测试 | `ctid` 前提；`s1-ll-order` 被发现 | 存储；端到端 |
| 49 组合的夹具看得到过宽的写 | 标签的三个写 | `TestEachLabelWriteChangesItsRowsAlone`；交错核对 Ops；故事 P7 读 Ops、Elsewhere | 组合；端到端 |
| 50 每句注释和文档对照它说的代码 | 每个 Task 末尾 | 见清扫 6、15 | 审阅 |

**只在单元一层被发现的变异（按性质）**：端口失败的 7 个（`s2-*`：真实的存储在这些步骤不失败，存储方法自己的失败由清扫 19 在存储一层核对）；两个写的回答和 `checkParent` 读到的父标签对不对得上所问（`s22-*` 3 个：真实的存储回答的就是问的那一行，裁定 S6 的同类）；`s18-update-extra-default`（多声明、从不回答的码：只有 `apitest.Main` 的覆盖核对看得出，P5b、P7a 的同类也只在单元一层）。

**只在结构、存储一层（按性质）**：存储方法自己的失败（`s19-*` 中的 8 个，清扫 19：取消的 context）；`s19-any-key-taken`（重复的 id 答成名称被占用：用例每次新生成 id，组合出的 app 里到不了）；`s1-check-own-parent`（迁移去掉 `labels_not_own_parent_check`：领域先拒绝自己做自己的父标签，`r-parent-self` 在单元和端到端失败，CHECK 是纵深防御，app 里到不了它）、`s1-check-name`（迁移去掉 `CHECK (name <> '')`：领域先拒绝空白的名称，`CheckNewLabel`、`CheckLabelPatch`）；两个都由 `TestProjectChecksRejectCounterexamples`、`TestConstraintAndIndexNames` 和存储的 `TestCreateLabelBreakingAnotherConstraintIsInternal` 发现。

**组合一层等价、只在存储一层（按设计，指名的测试发现它们）**：`s1-ul-deleted`（`UpdateLabel` 去掉 `deleted_at IS NULL`：写的 id 来自项目的 N 之下的 `LabelByID`，它不回答已删除的，之后没有别的写能删除它，语句自己的守卫是纵深防御；由存储的 `TestUpdateLabelWritesNoDeletedLabel` 发现）；`s3-ul-created`（`UpdateLabel` 改写 `created_at`：审计列，在存储一层每列比较；组合一层和故事读写者和时刻，不读 `created_at`，同 P7a 的 `s3-us-created`；由存储的 `TestUpdateLabel` 发现）。

**只在存储一层、组合一层和故事里没有反例的（按设计）**：`s1-ll-tiebreak`（`ListLabels` 只按 `sort_order` 排，同值的次序不定）：建的标签每次在最大值之后（交错现在核对没有两个同值），同值只有 `updateLabel` 给得出；列表的次序是语句自己的性质，用例原样回答存储的次序（单元的 `TestListLabels`），由存储的 `TestListLabels` 的同值行钉住（清扫 48）。

**组合一层没有、端到端有反例的**（14 个，每个也在单元或存储一层被发现；`e2e-exclusion-kept` 改的是故事的夹具，见清扫 28）：列表的范围和次序（`s1-ll-deleted`、`s1-ll-order`）、最大的 `sort_order` 的范围（`s1-gs-project`、`s1-gs-deleted`）、唯一索引算上已删除的（`s1-idx-deleted`）、连带再删已删除的（`s1-cl-deleted`、`s3-cl-keeps-moment`，P4、W3）、修改留着没给的值（`s26-ul-color-empty`、`s26-ul-parent-top`、`s26-ul-null-kept`、`s4-update-null-ignored`）、自己做自己的父标签和颜色的长度（`r-parent-self`、`v-color-unchecked`）、项目的第一个标签的 `sort_order`（`q-first`）。组合一层的测试读锁、行、写者、时刻和问题，这些是 API 的回答里的值和故事走到的拒绝；故事 P7、P4、W3 是它们在真实环境上的反例。

**变异表（按缺陷类别）**：变异的定义在 `$M3TMP/p7btools/mutants_p7b.py`（108 个），分层的结果在 `p7btools/logs`（`stage-unit.json`、`stage-schema.json`、`stage-store.json`、`stage-composed.json`、`stage-e2e-P7.json`、`stage-e2e-P4.json`、`stage-e2e-W3.json`，修订之前的七层和标记留在 `logs/pre-amend/`；加强之前的单元、组合和 P7 三层留在 `stage-unit-before-hardening.json`、`stage-composed-before-hardening.json`、`stage-e2e-P7-before-hardening.json`；修订之后七层整层重跑；组合一层的"（Task n 起）"由 `markers.py` 在修订之后的快照上重新量出（`logs/markers.json`、`markers-*.log`），每个变异的 `mutant-*.log`），`mut_tables.py` 把重跑按层合进去，写出 plan 每个 Task 的变异表和下表。108 个变异，108 个被发现；只在单元一层被发现的 11 个，都是按性质的（上文）。

| 缺陷类别 | 变异 | 结果 | 失败的测试 | 层 |
|---|---|---|---|---|
| 清扫 1：SQL 谓词 | `s1-lb-id`、`s1-lb-deleted`、`s1-ll-project`、`s1-ll-deleted`、`s1-ll-order`、`s1-gs-project`、`s1-gs-deleted`、`s1-gs-asc`、`s1-hc-parent`、`s1-hc-deleted`、`s1-ul-id`、`s1-ul-deleted`、`s1-dl-id`、`s1-dl-children`、`s1-dl-deleted`、`s1-cl-workspace`、`s1-cl-project`、`s1-cl-deleted`、`s1-idx-case`、`s1-idx-deleted`、`s1-check-own-parent`、`s1-check-name`、`s1-ll-tiebreak` | 23/23 | `TestLabelByID`、`TestUpdateLabel`、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`、`TestEachLabelWriteChangesItsRowsAlone`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`、`TestLabelWritesOnOneProjectSerialize`、等 25 个 | 结构；存储；组合；端到端 |
| 清扫 3：写不动的列 | `s3-ul-created`、`s3-ul-keeps-moment`、`s3-dl-keeps-moment`、`s3-cl-keeps-moment` | 4/4 | `TestUpdateLabel`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`、`TestLabelWritesOnOneProjectSerialize`、`TestTheWritesOnAProjectStampTheirRequest`、`TestDeleteLabel`、P7、`TestDeletingAProjectSoftDeletesItsRowsAlone`、等 3 个 | 存储；组合；端到端 |
| 清扫 14：由谁 | `s14-ul-keeps-writer`、`s14-dl-keeps-writer`、`s14-cl-keeps-writer` | 3/3 | `TestUpdateLabel`、`TestTheWritesOnAProjectStampTheirRequest`、P7、`TestDeleteLabel`、`TestDeletingAProjectSoftDeletesItsRowsAlone`、`TestDeletingAWorkspaceSoftDeletesItsProjects`、`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、等 2 个 | 存储；组合；端到端 |
| 清扫 26：不会碰巧的值 | `s26-ul-name-kept`、`s26-ul-color-empty`、`s26-ul-sort-zero`、`s26-ul-parent-top`、`s26-ul-null-kept`、`s26-cr-sort`、`s26-cr-parent`、`s26-cr-color` | 8/8 | `TestUpdateLabel`、`TestUpdateLabelNameTaken`、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`、`TestPermissionMatrix`、`TestTheWritesOnAProjectStampTheirRequest`、P7、`TestCreateLabel`、等 13 个 | 存储；组合；端到端 |
| 清扫 8：事务的连接 | `s8-create`、`s8-byid`、`s8-greatest`、`s8-children`、`s8-update`、`s8-delete`、`s8-cascade` | 7/7 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`、`TestADeletionRefusedAtItsCommitChangesNoRow`、`TestAProjectDeletionRefusedAtItsCommitChangesNoRow` | 组合 |
| 清扫 19：存储方法的失败 | `s19-create`、`s19-byid`、`s19-list`、`s19-greatest`、`s19-children`、`s19-update`、`s19-delete`、`s19-cascade`、`s19-name-taken-internal`、`s19-any-key-taken` | 10/10 | `TestAFailedWriteIsAnError`、`TestCreateLabelBreakingAnotherConstraintIsInternal`、`TestCreateLabelNameTaken`、`TestAFailedReadIsAnErrorNotAnAnswer`、`TestUpdateLabelNameTaken`、`TestUpdateLabelWritesNoDeletedLabel`、`TestEachLabelWriteChangesItsRowsAlone`、等 3 个 | 存储；组合；端到端 |
| 清扫 2：端口的错误 | `s2-create-greatest`、`s2-create-create`、`s2-parent-read`、`s2-update-children`、`s2-update-update`、`s2-delete-delete`、`s2-list` | 7/7 | `TestCreateLabelReturnsEachFailure`、`TestCreateLabelRefuses`、`TestUpdateLabelReturnsEachFailure`、`TestUpdateLabelRefuses`、`TestDeleteLabelReturnsEachFailure`、`TestListLabelsRefuses` | 单元 |
| 清扫 22：回答对得上所问（裁定 S6） | `s22-created-id`、`s22-updated-id`、`s22-parent-id` | 3/3 | `TestCreateLabelRefuses`、`TestUpdateLabelRefuses` | 单元 |
| 标签的路径：404、锁的强度（清扫 13、41） | `p-label-404-state`、`p-create-share-project` | 2/2 | `TestDeleteLabelRefuses`、`TestUpdateLabelRefuses`、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`、`TestBodiesThatBreakTheStructureAnswer400`、`TestPermissionMatrix`、P7、`TestLabelWritesOnOneProjectSerialize` | 单元；组合；端到端 |
| 时钟在锁下（3.3） | `c-create-early`、`c-update-early`、`c-delete-early` | 3/3 | `TestCreateLabel`、`TestCreateLabelRefuses`、`TestCreateLabelReturnsEachFailure`、`TestEachWriteReadsTheClockUnderItsLock`、`TestLabelWritesOnOneProjectSerialize`、`TestUpdateLabel`、`TestUpdateLabelRefuses`、等 5 个 | 单元；组合 |
| 两层和父标签的规则、值的检查（清扫 36） | `r-parent-any-project`、`r-parent-self`、`r-parent-third-level`、`r-parent-children`、`r-update-children-unread`、`r-update-unchecked`、`r-create-unchecked`、`r-parent-same-workspace`、`v-color-unchecked`、`v-patch-name-unchecked` | 10/10 | `TestCheckParent`、`TestCreateLabelRefuses`、`TestUpdateLabelRefuses`、`TestPermissionMatrix`、P7、`TestLabelWritesOnOneProjectSerialize`、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`、等 7 个 | 单元；组合；端到端 |
| `sort_order` | `q-step`、`q-first`、`q-greatest-ignored` | 3/3 | `TestCreateLabel`、`TestCreateLabelRefuses`、`TestCreateLabelReturnsEachFailure`、`TestEachWriteReadsTheClockUnderItsLock`、`TestSortOrderAfter`、`TestPermissionMatrix`、P7、等 1 个 | 单元；组合；端到端 |
| 清扫 21：第一个问题 | `s21-taken-wrapped`、`s21-patch-checked-late` | 2/2 | `TestCreateLabelNameTaken`、`TestUpdateLabelNameTaken`、`TestEachLabelWriteChangesItsRowsAlone`、`TestLabelWritesOnOneProjectSerialize`、`TestPermissionMatrix`、P7、`TestUpdateLabelRefuses` | 单元；存储；组合；端到端 |
| 清扫 25、31：判定和执行在同一把锁下（交错 11） | `s25-update-parent-unlocked`、`s25-create-parent-unlocked`、`s25-create-greatest-unlocked` | 3/3 | `TestLabelWritesOnOneProjectSerialize`、`TestPermissionMatrix`、`TestCreateLabel`、`TestCreateLabelRefuses`、`TestCreateLabelReturnsEachFailure`、`TestEachWriteReadsTheClockUnderItsLock` | 单元；组合 |
| 清扫 32：探针的反例 | `s32-wait-on-label` | 1/1 | `TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`、`TestLabelWritesOnOneProjectSerialize` | 组合 |
| 清扫 4：接线 | `s4-create-frozen-clock`、`s4-update-frozen-clock`、`s4-delete-frozen-clock`、`s4-update-drops-parent`、`s4-update-null-ignored`、`s4-create-drops-parent`、`s4-create-drops-color`、`s4-label-parent-null` | 8/8 | `TestTheWritesOnAProjectStampTheirRequest`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`、`TestUpdateLabel`、`TestEachLabelWriteChangesItsRowsAlone`、`TestPermissionMatrix`、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`、P7、等 2 个 | 单元；组合；端到端 |
| 清扫 5：规则表和判定 | `s5-update-members`、`s5-create-members`、`s5-delete-guests`、`s5-list-no-guests`、`s5-list-undecided` | 5/5 | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix`、P7、`TestListLabels`、`TestListLabelsRefuses` | 单元；组合；端到端 |
| 清扫 18：契约 | `s18-delete-no-not-found`、`s18-create-no-taken`、`s18-update-extra-default` | 3/3 | `TestDeleteLabel`、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`、`TestPermissionMatrix`、`TestCreateLabelRefusals`、`TestEachLabelWriteChangesItsRowsAlone`、`apitest.Main` | 单元；组合 |
| 清扫 34：完整性的核对 | `s34-first-lock-row`、`e2e-exclusion-kept` | 2/2 | `TestEachWriteOnAProjectSharesItsWorkspaceFirst`、P4、W3 | 组合；端到端 |
| 连带：删除项目、工作区（3.6、11.5） | `k-no-labels-step` | 1/1 | `TestDeleteProject`、`TestDeleteProjectReturnsEachFailure`、`TestDeleteWorkspaceProjects`、`TestEachWriteReadsTheClockUnderItsLock`、`TestAFailedProjectsStepRollsTheDeletionBack`、`TestAProjectDeletionRefusedAtItsCommitChangesNoRow`、`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、等 7 个 | 单元；组合；端到端 |

**Phase 的大小**（裁定 S3：约 16 个任务、每个任务约 1,500 行以内）：11 个 Task（约 16 个之内）；plan 7,842 行（`wc -l`；修订之前 7,614 行）；每个 Task 都在约 1,500 行以内，最大的是 Task 5（926 行），其次 Task 7（923 行）。各 Task：1 629、2 430、3 863、4 748、5 926、6 577、7 923、8 655、9 612、10 594、11 799（`assemble.py` 量的，从 Task 的标题到下一个标题）。没有超出，没有停下。
