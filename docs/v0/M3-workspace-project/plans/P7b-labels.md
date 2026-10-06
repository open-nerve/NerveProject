# M3/P7b 标签 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `labels` 表（迁移 `00014`）和它的两个连带（删除项目、删除工作区，`deleteProjects` 的最后一步），标签的全部操作（`listLabels`、`createLabel`、`updateLabel`、`deleteLabel`）和 3.16 的规则：两层（父标签是同一项目、未删除、没有父标签的标签；有子标签的标签不能有父标签；不能做自己的父标签，各 422 `parent_id` `not_allowed`）；名称在项目内唯一、不分大小写（`lower(name)` 的部分唯一索引，409 `project.label_name_taken`）；新标签的 `sort_order` 是给的，或者项目未删除的标签的最大值加 10000，没有标签时 65535；删除一个标签同时删除它下面的标签，一条语句。每个标签的写：工作区 `FOR SHARE` → 项目 `FOR NO KEY UPDATE` → 判定 → 层级的检查 → 时钟 → 写；`updateLabel`、`deleteLabel` 经 P7a 的共用路径（锁下重读，各答 `project.label_not_found`）。交错 11 和同名的并发创建两种顺序串行，删除父标签与往它下面放标签的最坏交错由探测展示（约定五的分析）；故事 P7 的接口版本通过，W2、W3、P4 断言标签。

**Architecture:** `server/migrations`：`00014_project_labels.sql`。`project/domain`：`label.go`（`Label`、`Place`、`CheckNewLabel`、`CheckLabelPatch`、`SortOrderAfter`、`CheckParent`），两个标签码；名称的检查与状态共用（`checkRequiredText`、`checkMaxLength`）。`project/adapter/postgres`：`queries/labels.sql` 的七条语句和存储方法，`cascade.sql` 的 `DeleteLabels`。`project/app`：`label_ports.go`（`LabelRow`、`LabelFinder`、`labelWrite`、四个端口）、`label_parent.go`（`checkParent`）、四个用例；`deletion.go` 加标签一步。`project/adapter/http`：四个处理函数。`access`：四行规则。`bootstrap`：矩阵的标签种子和行（已归档项目的小表随每个操作），最先锁工作区的测试、盖戳、连接各加标签的写，按资源寻址的竞争和锁强度推广到标签的行，交错 11，每个标签的写只写它的行。不加 Go 模块、npm 包；跨模块的端口不变。

**Tech Stack:** Go 1.27.1、pgx v5.11.0、sqlc v1.31.1（`CGO_ENABLED=0`）、oapi-codegen v2.8.0、goose v3.28.0、River v0.47.0、golangci-lint 2.13.2、PostgreSQL 18.6（testcontainers）；Node 24、pnpm 11.10.0、Playwright 1.63.0。

**Spec:** `docs/v0/M3-workspace-project/specs/P7b-labels.md`（上级：`docs/v0/M3-workspace-project/M3-design.md`）

## Global Constraints

- **Go 版本和依赖**：本 plan 不执行 `go get`，不加任何 Go 模块或 npm 包。`server/go.mod` 和 `server/tools/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`（golangci-lint 2.13.2 由 go1.27.0 构建，`go` 行更高就拒绝运行）。每个 Task 提交前执行 `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod`，四行必须是 `go 1.27` 和 `toolchain go1.27.1`；`git diff --stat 0deb8c34 -- server/go.mod server/go.sum server/tools pnpm-lock.yaml` 在本 plan 的任何时刻都没有输出。
- **每个 Task 提交前**：`make lint-go` 两段都输出 `0 issues.`，`make test` 全部通过（没有 `FAIL`）。
- **生成的文件不手写、不从本 plan 复制**：有生成物的 Task（1、3、5、7、8、9）执行 Task 中的生成命令（只动了 sqlc 的 Task 3 执行 `make gen-go`，改了迁移、连带的语句和接口描述的 Task 1，改了接口描述的 Task 5、7、8、9 执行 `make gen`），用 `shasum -a 256` 和 `wc -l` 核对表中的 SHA-256 和行数，对不上时停下来：说明某个输入与本 plan 不一致。这些 Task **提交之后**执行 `make gen-check`（它用 `git status` 判断生成物是否已提交，提交之前必然报差异）。
- **前端检查和新的错误码**（M3 设计 12 节约束 4）：声明新码的 Task 在同一个 Task 里把它加进 `PROBLEM_MESSAGES`（`web/apps/web/helpers/authentication.helper.ts`）和两份 `auth.json`：Task 5 `project.label_name_taken`，Task 7 `project.label_not_found`。改了接口描述的 Task 1、5、7、8、9 和改了 `e2e/` 的 Task 1、11 另执行 `make lint-web`、`make knip`、`make test-web`；Task 1、11 执行 `make e2e`。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers；机器忙时偶尔起不来，等 Docker 空闲之后重跑一次再当作失败。容器测试一次只跑一套。开发库 `nerve-dev-db-1` 可以用，但不要停止或重建它，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`、`nervewiki-*`）。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合，只读的命令也不行。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。
- **规则**：不写 `init()`，不用全局可变状态，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行；依赖只能向内；模块之间不互相导入，不跨模块的表 JOIN；模块的 SQL 只经 sqlc；角色只按集合判断，不按大小比较；不留没有使用者的代码。**例外**：接口描述按模块一个文件（M0-P3 交接 5），`api/modules/project.yaml`（1350 行）、`api/modules/workspace.yaml`（871 行）不受约 400 行的限制（spec 第 3 节第 1 条）。本 plan 的其余代码文件都在约 400 行以内（最终原型上量的）：最长的是 `server/migrations/schema_test.go`（399 行，本 plan 只改三处数字和表名）、`bootstrap/permission_matrix_test.go`（392 行）、`bootstrap/project_write_locks_test.go`（382 行）、`e2e/fixtures/assert/workspace.ts`（380 行）、`bootstrap/project_writes_test.go`（377 行）、`e2e/fixtures/assert/project.ts`（372 行）、`bootstrap/permission_matrix_columns_test.go`（360 行）、`project/app/ports.go`（352 行）、`bootstrap/project_row_races_test.go`（343 行）、`project/adapter/postgres/states_test.go`（334 行）、`bootstrap/permission_matrix_seeded_test.go`（331 行）、`bootstrap/workspace_deletion_test.go`（327 行）、`server/migrations/project_schema_test.go`（315 行）、`project/adapter/postgres/cascade_test.go`（314 行）、`project/adapter/http/handler_test.go`（306 行）。`bootstrap/permission_matrix_seed_test.go`（389 行）本 plan 不改：标签的种子写在新的 `permission_matrix_labels_test.go` 里（spec 第 3 节第 12 条）。
- **注释**：Go、TS 代码、SQL 查询和接口描述用英文；迁移的注释和中文文档照本 plan 原样。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的）。

  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从 `0deb8c34` 起按顺序核对过全部块：每个 `old` 恰好出现一次（在它之前的块应用之后的文件中），每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同（spec 附录 A）。可以用 `node <planapply.mjs> <本 plan> apply <仓库根> <n>` 写入第 n 个 Task 的块，也可以手工照抄。
- **过渡版本**：一些文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改：契约 `api/modules/project.yaml`、`api/openapi.yaml`，`project/app/label_ports.go`、`fakes_label_test.go`、`fakes_write_test.go`、`clock_test.go`，`project/adapter/http/handler.go`、`handler_test.go`、`labels.go`、`labels_test.go`，`project/module.go`、`project/domain/actions.go`，`access/domain/rules.go`、`rules_test.go`，`bootstrap/permission_matrix_labels_test.go`、`permission_matrix_columns_test.go`、`project_write_locks_test.go`、`project_writes_test.go`、`project_connection_test.go`，前端的三个文案文件，`e2e/fixtures/assert/project.ts`（Task 1 暂时排除 `labels`，Task 11 删去，spec 第 3 节第 11 条）。每个过渡版本都在逐 Task 复现中运行过。
- **变异**：每个 Task 末尾的"变异"表列出：把代码改坏的方式、必须因此失败的测试和它所在的层（单元：假实现；结构：迁移的测试；存储：真实数据库；组合：`bootstrap` 组合出的 app；端到端：单独运行的故事 P7、P4、W3）。它们在最终的原型上逐个跑过（`$M3TMP/p7btools/mutants_p7b.py`，由 `mutlevels.py` 在它写的每一层各跑一次；spec 附录 A：105 个变异，105 个被发现；只在单元一层被发现的 11 个，都是按性质的（附录 A））；一个变异列在它改的代码第一次出现的 Task（它的锚点从那个 Task 起在每一份快照上恰好出现一次，`mutanchors.py` 核对）。实现者可以照表抽查，改坏之后必须恢复。表中"（Task n 起）"标出的测试从那个 Task 起才发现这个变异：单元、结构、存储一层的，是测试在那个 Task 才有或才改成最终的样子；组合一层的，是把变异放在各个 Task 的快照上跑这个测试量出来的（`markers.py`：`bootstrap` 的测试的行常在函数之外，例如矩阵的行，只看函数体会漏标）；端到端的故事都从 Task 11 起。没有标的测试在这一行所在的 Task 的树上就失败。**安全或加锁的性质只由单元一层发现的，算缺口**（brief 的缺陷类别）；表中每一条这类性质都另有存储、组合或端到端一层的测试，例外（"按性质只在单元一层"）写在 spec 附录 A。
- **评审敏感**（M3 设计 12 节约束 3）：两层的规则只在领域，它在并发下由项目行的 `FOR NO KEY UPDATE` 保证：每个设父标签的写都在这把锁下读父标签和这个标签的子标签（`checkParent`）；`deleteLabel` 一条语句删除这个标签和它下面的标签，它们都是这个项目的，项目的锁覆盖这一批行，约定五的例外不适用（spec 第 3 节第 4 条）；新表进入删除项目、删除工作区的连带。改动这些锁、检查、规则或测试之前，先照"变异"表确认它在所说的性质去掉之后失败。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录。"生成"表示由命令生成并提交；一个文件由多个 Task 修改时，"Task"一列都列出。

| 文件 | 职责 | Task |
|---|---|---|
| `server/migrations/sql/00014_project_labels.sql` | `labels` 表（12 列）、`labels_not_own_parent_check`、`labels_project_id_name_key ON (project_id, lower(name)) WHERE deleted_at IS NULL`、三个不带条件的索引 | 1 |
| `server/sqlc.yaml`、`deploy/runtime-grants.sql`（修改） | sqlc 读到新的迁移；运行时的角色读写 `labels` | 1 |
| `server/migrations/schema_test.go`、`server/migrations/project_schema_test.go`（修改） | 14 个迁移、`labels` 在表里；约束和索引的名字；两个 CHECK 的反例；名称不分大小写的唯一键 | 1 |
| `server/internal/modules/project/adapter/postgres/queries/cascade.sql`、`server/internal/modules/project/adapter/postgres/cascade.go`（修改） | `DeleteLabels`、`Store.DeleteLabels` | 1 |
| `server/internal/modules/project/adapter/postgres/gen/cascade.sql.go`、`server/internal/modules/project/adapter/postgres/gen/models.go`（生成） | | 1 |
| `server/internal/modules/project/adapter/postgres/cascade_test.go`（修改） | 种子和核对的表加标签；五步 | 1 |
| `server/internal/modules/project/app/ports.go`、`server/internal/modules/project/app/deletion.go`（修改） | `ProjectsDeleter.DeleteLabels`；`deleteProjects` 的最后一步 | 1 |
| `server/internal/modules/project/app/fakes_write_test.go`（修改） | `fakeStore.DeleteLabels`（Task 1）；标签的假实现接进 `writeFixture`（Task 4、6） | 1、4、6 |
| `server/internal/modules/project/app/cascade_test.go`、`server/internal/modules/project/app/delete_project_test.go`（修改） | 五步的次序；标签一步的失败 | 1 |
| `server/internal/bootstrap/project_deletion_test.go`、`server/internal/bootstrap/workspace_deletion_test.go`（修改） | `seedLabels`；最后一步（标签）失败时整个回滚 | 1 |
| `api/modules/workspace.yaml`（修改） | `deleteWorkspace` 的描述加标签 | 1 |
| `api/modules/project.yaml`、`api/openapi.yaml`（修改） | `deleteProject` 的描述加标签（Task 1）；四个操作、`LabelID`、`Label`、`LabelCreate`、`LabelUpdate`、`LabelList`（Task 5、7、8、9） | 1、5、7、8、9（`openapi.yaml`：5、7） |
| `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`server/internal/modules/project/adapter/http/gen/bodyshape.gen.go`（生成） | | 1、5、7、8、9（`server.gen.go`：5、7、8、9；`bodyshape.gen.go`：5、7） |
| `e2e/fixtures/assert/project.ts`（修改） | Task 1：`expectProjectDeleted` 暂时排除 `labels`；Task 11：删去排除，只数之前未删除的行，`LabelRow`、`expectLabels` | 1、11 |
| `server/internal/modules/project/domain/label.go`、`server/internal/modules/project/domain/label_test.go` | 标签、`Place`、`CheckNewLabel`、`CheckLabelPatch`、`SortOrderAfter`、`CheckParent` | 2 |
| `server/internal/modules/project/domain/errors.go`、`server/internal/modules/project/domain/project.go`、`server/internal/modules/project/domain/state.go`、`server/internal/modules/project/domain/state_test.go`（修改） | 两个标签码；`checkMaxLength`、`checkRequiredText`，状态改用它 | 2 |
| `server/internal/modules/project/adapter/postgres/queries/labels.sql`、`server/internal/modules/project/adapter/postgres/labels.go` | 七条语句和存储方法 | 3 |
| `server/internal/modules/project/adapter/postgres/gen/labels.sql.go`（生成） | | 3 |
| `server/internal/modules/project/adapter/postgres/labels_test.go`、`server/internal/modules/project/adapter/postgres/label_writes_test.go`；`server/internal/modules/project/adapter/postgres/failures_test.go`、`server/internal/modules/project/adapter/postgres/states_test.go`（修改） | 每条语句的存储测试；新方法的失败测试；`workspaceOf` | 3 |
| `server/internal/modules/project/app/label_ports.go` | `LabelRow`（Task 3），`LabelFinder`、`LabelCreator`（Task 4），`labelWrite`、`LabelUpdater`（Task 6），`LabelDeleter`（Task 8），`LabelLister`（Task 9） | 3、4、6、8、9 |
| `server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/domain/actions.go`（修改） | `label.create`（Task 4）、`label.update`（Task 6）、`label.delete`（Task 8）、`label.list`（Task 9） | 4、6、8、9 |
| `server/internal/modules/project/app/label_parent.go` | `checkParent`：在项目的锁下读父标签并检查 | 4 |
| `server/internal/modules/project/app/create_label.go`、`server/internal/modules/project/app/create_label_test.go` | `createLabel` | 4 |
| `server/internal/modules/project/app/fakes_label_test.go`；`server/internal/modules/project/app/fakes_state_test.go`（修改） | 标签的假实现（`fakeLabels`）；`stateSince` 的说明 | 4、6、8、9（`fakes_state_test.go`：4） |
| `server/internal/modules/project/app/clock_test.go`（修改） | 每个标签的写在锁之后读时钟的一行 | 4、6、8 |
| `server/internal/modules/project/adapter/http/labels.go`、`server/internal/modules/project/adapter/http/labels_test.go`；`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`（修改） | 四个处理函数和它们的测试；`UseCases` 的四个字段 | 5、7、8、9 |
| `server/internal/modules/project/module.go`（修改） | 四个用例的接线；包说明 | 5、7、8、9 |
| `web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`（修改） | 两个标签码的文案 | 5、7 |
| `server/internal/bootstrap/permission_matrix_labels_test.go` | 标签的矩阵行（已归档项目的小表随每个操作）、它们的核对，标签的种子（`matrixLabels`、`labels`、`seededLabels`） | 5、7、8、9 |
| `server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_test.go`（修改） | `seeded.label`；`prepareMatrix` 种下并读回标签，矩阵加标签的行 | 5 |
| `server/internal/bootstrap/permission_matrix_columns_test.go`、`server/internal/bootstrap/permission_matrix_targets_test.go`（修改） | 标签从不在 `seeded` 的成员关系里；瞄准标签行的格子（`{label_id}`） | 5、7（`targets`：7） |
| `server/internal/bootstrap/project_write_locks_test.go`、`server/internal/bootstrap/project_writes_test.go`、`server/internal/bootstrap/project_connection_test.go`（修改） | 最先锁工作区的测试、盖戳、连接各加标签的写（`labelNamed`、`rowPaths` 加 `/api/v0/labels/`、`labelID`） | 5、7、8 |
| `server/internal/modules/project/app/update_label.go`、`server/internal/modules/project/app/update_label_test.go` | `updateLabel` | 6 |
| `server/internal/modules/project/app/delete_label.go`、`server/internal/modules/project/app/delete_label_test.go` | `deleteLabel` | 8 |
| `server/internal/modules/project/app/list_labels.go`、`server/internal/modules/project/app/list_labels_test.go` | `listLabels` | 9 |
| `server/internal/bootstrap/interleaving_labels_test.go`、`server/internal/bootstrap/label_rows_test.go`；`server/internal/bootstrap/project_row_races_test.go`（修改） | 交错 11 和标签的其余六对（约定五的探测在其中）；每个标签的写只写它的行；按资源寻址的竞争和锁强度加标签的行 | 10 |
| `e2e/stories/project/p7-labels.spec.ts`；`e2e/stories/project/p4-archive.spec.ts`、`e2e/stories/workspace/w2-landing.spec.ts`、`e2e/stories/workspace/w3-workspace-settings.spec.ts`（修改） | 故事 P7 的接口版本；P4、W2、W3 的标签 | 11 |
| `e2e/fixtures/api.ts`、`e2e/fixtures/assert/workspace.ts`（修改） | `Label*` 类型、`createLabel`；`workspaceTables`、`deletedAloneTables` 加 `labels` | 11 |
| `server/internal/bootstrap/workspace_deletion_catalog_test.go`（修改） | `keysTo` 的说明：指向自己的外键 | 11 |
| `docs/v0/v0-design.md`、`docs/v0/plane-diff.md`（修改） | 总体设计 5.3；二·按表的 `labels`、第四节的 P7b 行（3.20） | 11 |

---

### Task 1: `labels` 表、授权和两个连带：删除项目、删除工作区的最后一步

**Files:**
- Create: `server/migrations/sql/00014_project_labels.sql`
- Modify: `api/modules/project.yaml`、`api/modules/workspace.yaml`、`deploy/runtime-grants.sql`、`e2e/fixtures/assert/project.ts`、`server/internal/bootstrap/project_deletion_test.go`、`server/internal/bootstrap/workspace_deletion_test.go`、`server/internal/modules/project/adapter/postgres/cascade.go`、`server/internal/modules/project/adapter/postgres/cascade_test.go`、`server/internal/modules/project/adapter/postgres/queries/cascade.sql`、`server/internal/modules/project/app/cascade_test.go`、`server/internal/modules/project/app/delete_project_test.go`、`server/internal/modules/project/app/deletion.go`、`server/internal/modules/project/app/fakes_write_test.go`、`server/internal/modules/project/app/ports.go`、`server/migrations/project_schema_test.go`、`server/migrations/schema_test.go`、`server/sqlc.yaml`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/postgres/gen/cascade.sql.go`、`server/internal/modules/project/adapter/postgres/gen/models.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.3；M3 设计 3.6、3.16、4.10、11.5）：
  - `00014_project_labels.sql`：`labels`（Plane 15 列保留 12 列：`id`、`workspace_id`、`project_id NOT NULL`、`parent_id REFERENCES labels ON DELETE CASCADE`、`name varchar(255) NOT NULL CHECK (name <> '')`、`color varchar(255) NOT NULL DEFAULT ''`、`sort_order double precision NOT NULL DEFAULT 65535`、审计列和 `deleted_at`）、`labels_not_own_parent_check CHECK (parent_id <> id)`、`labels_project_id_name_key ON (project_id, lower(name)) WHERE deleted_at IS NULL`、不带条件的 `labels_parent_id_idx`、`labels_workspace_id_idx`、`labels_project_id_idx`；Down 删表。`server/sqlc.yaml` 读它；`deploy/runtime-grants.sql` 给运行时的角色 `labels` 的读写。
  - `queries/cascade.sql` 的 `DeleteLabels`（照 `DeleteStates`：工作区的、`project_id` 给出时只那个项目的未删除的标签，父子一起）、`Store.DeleteLabels(ctx, app.Deletion) error`；`app.ProjectsDeleter` 加 `DeleteLabels`；`deleteProjects` 的步骤是项目、成员关系、成员的显示设置、状态、标签（说明的"P7b adds the labels at the end"半句删去）。删除项目、删除工作区（`DeleteWorkspaceProjects`）都跑这一组。
  - 契约：`deleteProject`、`deleteWorkspace` 的描述加标签。
  - 端到端：`expectProjectDeleted` 暂时排除 `labels`，注释写明为什么、到哪个 Task（spec 第 3 节第 11 条）；Task 11 删去。

**Tests:**
- 迁移：`TestMigrationsGoUpDownAndUpAgain`（14 个迁移、`labels` 在表的列表里，降到零、再升）、`TestConstraintAndIndexNames`（`projectNames` 加标签的 13 个名字和种类）、`TestProjectChecksRejectCounterexamples`（空的标签名称、自己做自己的父标签，各由它的 CHECK 拒绝）、`TestProjectUniqueKeysHoldAmongUndeletedRowsOnly`（Web 有 Bug 时 bug 被 `labels_project_id_name_key` 拒绝，Bug 删除之后可以；Ops 有它自己的 Bug：不折叠大小写的键会收下 bug，少了 `project_id` 的键会拒绝 Ops 的 Bug）。
- 存储：`cascade_test.go` 的 `checkDeletions` 的表加 `labels`，`seedProject` 加一个标签和它的子标签：`TestDeletingAWorkspaceSoftDeletesItsProjects`、`TestDeletingAProjectSoftDeletesItsRowsAlone`、`TestDeletingAnotherWorkspacesProjectChangesNothing` 的"五步"（`deletionSteps` 从端口反射出全部步骤，标签一步自动在内）。
- 单元：`TestDeleteWorkspaceProjects`（五步的次序）、`TestDeleteProjectReturnsEachFailure`（标签一步的失败原样返回，提交之前；之后什么都不运行）。
- 组合：`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`（目录驱动：`keysTo` 读出 `labels` 的外键，种下的 Bug 和 UI（`seedLabels`，由不是删除者的账户直接写）随项目、工作区在同一时刻由删除者删除）；`TestAFailedProjectsStepRollsTheDeletionBack`（请求期间把 `labels` 改名，最后一步失败：500，之前的步骤都回滚）；`TestAProjectDeletionRefusedAtItsCommitChangesNoRow`、`TestADeletionRefusedAtItsCommitChangesNoRow` 照旧通过（标签的行也不变）。

- [ ] **Step 1: 迁移、sqlc 配置、授权和迁移的测试**

`server/migrations/sql/00014_project_labels.sql`（新文件，30 行）：

````file server/migrations/sql/00014_project_labels.sql
-- labels：Plane 的 labels 表（15 列）按 M3 设计 4.10 保留 12 列。
-- 只有项目的标签，project_id 非空；最多两层，名称在项目内唯一、不分大小写（3.16）。description 没有读取者，删除。

-- +goose Up
CREATE TABLE labels (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
    -- 两层的规则在领域层，并发下由项目行的锁保证（3.16）；自己不能做自己的父标签
    parent_id uuid REFERENCES labels ON DELETE CASCADE,
    name varchar(255) NOT NULL CHECK (name <> ''),
    color varchar(255) NOT NULL DEFAULT '',
    sort_order double precision NOT NULL DEFAULT 65535,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CONSTRAINT labels_not_own_parent_check CHECK (parent_id <> id)
);
-- 名称在项目内唯一，不分大小写（3.16）
CREATE UNIQUE INDEX labels_project_id_name_key ON labels (project_id, lower(name)) WHERE deleted_at IS NULL;
-- 删除父标签时找子标签，也服务物理级联，所以不带条件（4 节开头）
CREATE INDEX labels_parent_id_idx ON labels (parent_id);
-- 物理级联（M4 的 60 天清理）按它们找子行（4 节开头）
CREATE INDEX labels_workspace_id_idx ON labels (workspace_id);
CREATE INDEX labels_project_id_idx ON labels (project_id);

-- +goose Down
DROP TABLE labels;
````

`server/sqlc.yaml`（修改，1 处）：

````old server/sqlc.yaml
      - migrations/sql/00013_project_states.sql
````
````new server/sqlc.yaml
      - migrations/sql/00013_project_states.sql
      - migrations/sql/00014_project_labels.sql
````

`deploy/runtime-grants.sql`（修改，1 处）：

````old deploy/runtime-grants.sql
GRANT SELECT, INSERT, UPDATE, DELETE ON projects, project_members, project_user_properties, states TO nerve_runtime;
````
````new deploy/runtime-grants.sql
GRANT SELECT, INSERT, UPDATE, DELETE ON projects, project_members, project_user_properties, states, labels TO nerve_runtime;
````

`server/migrations/schema_test.go`（修改，3 处）：

````old server/migrations/schema_test.go
	if err != nil || len(up) != 13 {
		t.Fatalf("Up() = %d migrations, %v; want 13", len(up), err)
````
````new server/migrations/schema_test.go
	if err != nil || len(up) != 14 {
		t.Fatalf("Up() = %d migrations, %v; want 14", len(up), err)
````

````old server/migrations/schema_test.go
		{tablesQuery, []string{"api_tokens", "auth_sessions", "profiles", "project_members", "project_user_properties", "projects", "river_job",
````
````new server/migrations/schema_test.go
		{tablesQuery, []string{"api_tokens", "auth_sessions", "labels", "profiles", "project_members", "project_user_properties", "projects", "river_job",
````

````old server/migrations/schema_test.go
	if again, err := m.Up(ctx); err != nil || len(again) != 13 {
		t.Errorf("Up() again = %d migrations, %v; want 13", len(again), err)
````
````new server/migrations/schema_test.go
	if again, err := m.Up(ctx); err != nil || len(again) != 14 {
		t.Errorf("Up() again = %d migrations, %v; want 14", len(again), err)
````

`server/migrations/project_schema_test.go`（修改，8 处）：

````old server/migrations/project_schema_test.go
// tables (M3 design 4.6–4.9), as TestConstraintAndIndexNames reads them:
// the part of its want that the migrations 00010–00013 add.
var projectNames = []string{
````
````new server/migrations/project_schema_test.go
// tables (M3 design 4.6–4.10), as TestConstraintAndIndexNames reads them:
// the part of its want that the migrations 00010–00014 add.
var projectNames = []string{
	"labels_created_by_id_fkey f n",
	"labels_name_check c",
	"labels_not_own_parent_check c",
	"labels_parent_id_fkey f c",
	"labels_parent_id_idx i",
	"labels_pkey iu",
	"labels_pkey p",
	"labels_project_id_fkey f c",
	"labels_project_id_idx i",
	"labels_project_id_name_key iuw",
	"labels_updated_by_id_fkey f n",
	"labels_workspace_id_fkey f c",
	"labels_workspace_id_idx i",
````

````old server/migrations/project_schema_test.go
// bypasses it (M3 design 3.17, 3.19, 4.6–4.9). projects_logo_props_check
````
````new server/migrations/project_schema_test.go
// bypasses it (M3 design 3.16, 3.17, 3.19, 4.6–4.10). projects_logo_props_check
````

````old server/migrations/project_schema_test.go
		"INSERT INTO states (id, workspace_id, project_id, name, color) VALUES (gen_random_uuid(), " + workspace + ", " + project + ", 'Backlog', '#60646C')",
````
````new server/migrations/project_schema_test.go
		"INSERT INTO states (id, workspace_id, project_id, name, color) VALUES (gen_random_uuid(), " + workspace + ", " + project + ", 'Backlog', '#60646C')",
		"INSERT INTO labels (id, workspace_id, project_id, name) VALUES (gen_random_uuid(), " + workspace + ", " + project + ", 'Bug')",
````

````old server/migrations/project_schema_test.go
		{"upper-case group", `UPDATE states SET "group" = 'Backlog'`, "states_group_check"},
````
````new server/migrations/project_schema_test.go
		{"upper-case group", `UPDATE states SET "group" = 'Backlog'`, "states_group_check"},
		{"empty label name", "UPDATE labels SET name = ''", "labels_name_check"},
		{"a label its own parent", "UPDATE labels SET parent_id = id", "labels_not_own_parent_check"},
````

````old server/migrations/project_schema_test.go
// first frees the key (M3 design 3.17, 3.19, 4.6–4.9). Another workspace,
// project or account holds keys of its own: a key short of a column
// refuses one of the seeds. A project's or a state's name that differs
// from another in case only is another name, as in Plane (3.17, 3.19): a
// key that folds case refuses one of the seeds too.
````
````new server/migrations/project_schema_test.go
// first frees the key (M3 design 3.16, 3.17, 3.19, 4.6–4.10). Another
// workspace, project or account holds keys of its own: a key short of a
// column refuses one of the seeds. A project's or a state's name that
// differs from another in case only is another name, as in Plane (3.17,
// 3.19): a key that folds case refuses one of the seeds too. A label's is
// the same name (3.16): a key that does not fold case takes it.
````

````old server/migrations/project_schema_test.go
			acme, project, name, group, isDefault)
	}
````
````new server/migrations/project_schema_test.go
			acme, project, name, group, isDefault)
	}
	label := func(project, name string) string {
		return "INSERT INTO labels (id, workspace_id, project_id, name) VALUES (gen_random_uuid(), " + acme + ", " + project + ", '" + name + "')"
	}
````

````old server/migrations/project_schema_test.go
		state(ops, "Backlog", "backlog", true), state(ops, "Triage", "triage", false), state(ops, "Todo", "unstarted", false),
````
````new server/migrations/project_schema_test.go
		state(ops, "Backlog", "backlog", true), state(ops, "Triage", "triage", false), state(ops, "Todo", "unstarted", false),
		// web's Bug; ops has its own.
		label(web, "Bug"), label(ops, "Bug"),
````

````old server/migrations/project_schema_test.go
			`UPDATE states SET deleted_at = now() WHERE "group" = 'triage' AND project_id = ` + web, "states_project_id_triage_key"},
````
````new server/migrations/project_schema_test.go
			`UPDATE states SET deleted_at = now() WHERE "group" = 'triage' AND project_id = ` + web, "states_project_id_triage_key"},
		{"a label's name in a project, in another case", label(web, "bug"),
			"UPDATE labels SET deleted_at = now() WHERE name = 'Bug' AND project_id = " + web, "labels_project_id_name_key"},
````

- [ ] **Step 2: 连带的语句和契约的两句**

`server/internal/modules/project/adapter/postgres/queries/cascade.sql`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/queries/cascade.sql
UPDATE states
````
````new server/internal/modules/project/adapter/postgres/queries/cascade.sql
UPDATE states
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND (sqlc.narg(project_id)::uuid IS NULL OR project_id = sqlc.narg(project_id))
  AND deleted_at IS NULL;

-- name: DeleteLabels :exec
-- The parents and their children alike.
UPDATE labels
````

`api/modules/project.yaml`（修改，1 处）：

````old api/modules/project.yaml
        deleted with its memberships, its members' display settings and its
        states, all at one moment: it is no longer read, listed or
        changed, and its name and identifier are free again in the
````
````new api/modules/project.yaml
        deleted with its memberships, its members' display settings, its
        states and its labels, all at one moment: it is no longer read,
        listed or changed, and its name and identifier are free again in the
````

`api/modules/workspace.yaml`（修改，1 处）：

````old api/modules/workspace.yaml
        their memberships, display settings and states are soft-deleted in one
        transaction, at the same moment; the members' accounts stay. The slug
        can name a new workspace at once. Nobody's last_workspace_id is
        cleared. A workspace that does not exist, is deleted, or of which the
        caller is not an active member answers workspace.not_found; a member
        or a guest, forbidden.
````
````new api/modules/workspace.yaml
        their memberships, display settings, states and labels are
        soft-deleted in one transaction, at the same moment; the members'
        accounts stay. The slug can name a new workspace at once. Nobody's
        last_workspace_id is cleared. A workspace that does not exist, is
        deleted, or of which the caller is not an active member answers
        workspace.not_found; a member or a guest, forbidden.
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `b6905e4155dd6e2f374e6226ff50568ffb16c5b7596da063b7b406b26819aae3` | 2752 | `api/dist/openapi.yaml` |
| `dd5ff4a6c413da536450dc9db13e65cd9110d43aa445effa71d269b2b42cfce5` | 297 | `server/internal/modules/project/adapter/postgres/gen/cascade.sql.go` |
| `eb4d21b20dfba323fc8badf3548f6fbb23c825914d2180c5e94a9516cf6452ca` | 97 | `server/internal/modules/project/adapter/postgres/gen/models.go` |
| `e97322be8e555303bc73eaeb86b1691312b637d8d77b6b088b922fdd1f63b979` | 3057 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/postgres/gen/cascade.sql.go server/internal/modules/project/adapter/postgres/gen/models.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 3: 端口、步骤、存储方法和它们的测试**

`server/internal/modules/project/app/ports.go`（修改，1 处）：

````old server/internal/modules/project/app/ports.go
	DeleteStates(ctx context.Context, d Deletion) error
````
````new server/internal/modules/project/app/ports.go
	DeleteStates(ctx context.Context, d Deletion) error
	DeleteLabels(ctx context.Context, d Deletion) error
````

`server/internal/modules/project/app/deletion.go`（修改，2 处）：

````old server/internal/modules/project/app/deletion.go
// display settings, the states; P7b adds the labels at the end. Deleting a
// workspace (Cascade) and deleting a project both run it, so neither can
// leave out a table the other deletes. A failing step comes back as itself
// and the steps after it do not run.
````
````new server/internal/modules/project/app/deletion.go
// display settings, the states, the labels. Deleting a workspace (Cascade)
// and deleting a project both run it, so neither can leave out a table the
// other deletes. A failing step comes back as itself and the steps after it
// do not run.
````

````old server/internal/modules/project/app/deletion.go
		p.DeleteProjects, p.DeleteProjectMembers, p.DeleteProjectPreferences, p.DeleteStates,
````
````new server/internal/modules/project/app/deletion.go
		p.DeleteProjects, p.DeleteProjectMembers, p.DeleteProjectPreferences, p.DeleteStates, p.DeleteLabels,
````

`server/internal/modules/project/adapter/postgres/cascade.go`（修改，1 处）：

````old server/internal/modules/project/adapter/postgres/cascade.go
}

// LockMemberProjects locks FOR NO KEY UPDATE, in id order, the workspace's
````
````new server/internal/modules/project/adapter/postgres/cascade.go
}

// DeleteLabels soft-deletes the projects' labels, parents and children
// alike.
func (s *Store) DeleteLabels(ctx context.Context, d app.Deletion) error {
	err := s.queries(ctx).DeleteLabels(ctx, gen.DeleteLabelsParams{WorkspaceID: d.WorkspaceID, ProjectID: d.ProjectID, DeletedBy: d.By, Now: d.Now})
	if err != nil {
		return fmt.Errorf("delete the labels: %w", err)
	}
	return nil
}

// LockMemberProjects locks FOR NO KEY UPDATE, in id order, the workspace's
````

`server/internal/modules/project/app/fakes_write_test.go`（修改，1 处）：

````old server/internal/modules/project/app/fakes_write_test.go
}

func (f *fakeStore) GetProject(ctx context.Context, id, userID uuid.UUID) (domain.Project, bool, error) {
````
````new server/internal/modules/project/app/fakes_write_test.go
}

func (f *fakeStore) DeleteLabels(ctx context.Context, d app.Deletion) error {
	return f.deleting(ctx, "DeleteLabels", d)
}

func (f *fakeStore) GetProject(ctx context.Context, id, userID uuid.UUID) (domain.Project, bool, error) {
````

`server/internal/modules/project/app/cascade_test.go`（修改，2 处）：

````old server/internal/modules/project/app/cascade_test.go
	for _, name := range []string{"DeleteProjects", "DeleteProjectMembers", "DeleteProjectPreferences", "DeleteStates"} {
````
````new server/internal/modules/project/app/cascade_test.go
	for _, name := range []string{"DeleteProjects", "DeleteProjectMembers", "DeleteProjectPreferences", "DeleteStates", "DeleteLabels"} {
````

````old server/internal/modules/project/app/cascade_test.go
// DeleteWorkspaceProjects runs the four steps in the order of M3 design
````
````new server/internal/modules/project/app/cascade_test.go
// DeleteWorkspaceProjects runs the five steps in the order of M3 design
````

`server/internal/modules/project/app/delete_project_test.go`（修改，2 处）：

````old server/internal/modules/project/app/delete_project_test.go
// decision, the clock, the four steps on that project alone.
````
````new server/internal/modules/project/app/delete_project_test.go
// decision, the clock, the five steps on that project alone.
````

````old server/internal/modules/project/app/delete_project_test.go
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 10},
````
````new server/internal/modules/project/app/delete_project_test.go
		{"the labels", func(f *writeFixture) { f.store.errs = map[string]error{"DeleteLabels": errDisk} }, 11},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 11},
````

`server/internal/modules/project/adapter/postgres/cascade_test.go`（修改，7 处）：

````old server/internal/modules/project/adapter/postgres/cascade_test.go
	{"projects", "id", 1}, {"project_members", "project_id", 2}, {"project_user_properties", "project_id", 2}, {"states", "project_id", 2},
````
````new server/internal/modules/project/adapter/postgres/cascade_test.go
	{"projects", "id", 1}, {"project_members", "project_id", 2}, {"project_user_properties", "project_id", 2}, {"states", "project_id", 2},
	{"labels", "project_id", 2},
````

````old server/internal/modules/project/adapter/postgres/cascade_test.go
// with alice's and bob's memberships and display settings in it, and a
// backlog state and the triage state, and returns its id. The rows are
// written directly: the delete statements read no column the inserts of
// the use cases would set otherwise.
````
````new server/internal/modules/project/adapter/postgres/cascade_test.go
// with alice's and bob's memberships and display settings in it, a backlog
// state and the triage state, and a label and its child, and returns its
// id. The rows are written directly: the delete statements read no column
// the inserts of the use cases would set otherwise.
````

````old server/internal/modules/project/adapter/postgres/cascade_test.go
			VALUES ($1, $2, $3, $4, '#60646C', $4, $5, $5, $6, $6)`, uuid.NewV7(), workspace, id, group, alice, now)
	}
````
````new server/internal/modules/project/adapter/postgres/cascade_test.go
			VALUES ($1, $2, $3, $4, '#60646C', $4, $5, $5, $6, $6)`, uuid.NewV7(), workspace, id, group, alice, now)
	}
	exec(t, pool, `INSERT INTO labels (id, workspace_id, project_id, parent_id, name, created_by_id, updated_by_id, created_at, updated_at)
		VALUES ($1, $2, $3, NULL, 'Bug', $4, $4, $5, $5), ($6, $2, $3, $1, 'UI', $4, $4, $5, $5)`, uuid.NewV7(), workspace, id, alice, now, uuid.NewV7())
````

````old server/internal/modules/project/adapter/postgres/cascade_test.go
// The four steps, on every project of a workspace, in one transaction,
````
````new server/internal/modules/project/adapter/postgres/cascade_test.go
// The five steps, on every project of a workspace, in one transaction,
````

````old server/internal/modules/project/adapter/postgres/cascade_test.go
// of them, active or not, every member's display settings in them and
// every state, the triage ones too, at the same moment and by the same
// account; a row deleted before keeps its time, and another workspace
// keeps everything. Running the steps again changes nothing.
````
````new server/internal/modules/project/adapter/postgres/cascade_test.go
// of them, active or not, every member's display settings in them, every
// state, the triage ones too, and every label, parents and children alike,
// at the same moment and by the same account; a row deleted before keeps
// its time, and another workspace keeps everything. Running the steps again
// changes nothing.
````

````old server/internal/modules/project/adapter/postgres/cascade_test.go
// The four steps, on one project, soft-delete it and the rows under it
````
````new server/internal/modules/project/adapter/postgres/cascade_test.go
// The five steps, on one project, soft-delete it and the rows under it
````

````old server/internal/modules/project/adapter/postgres/cascade_test.go
// The four steps, on a project of another workspace than the deletion's,
````
````new server/internal/modules/project/adapter/postgres/cascade_test.go
// The five steps, on a project of another workspace than the deletion's,
````

- [ ] **Step 4: 组合的连带测试**

`server/internal/bootstrap/project_deletion_test.go`（修改，2 处）：

````old server/internal/bootstrap/project_deletion_test.go
// settings in it and its states, through createProject.
````
````new server/internal/bootstrap/project_deletion_test.go
// settings in it and its states, through createProject, and a label of
// hers, Bug, with its child, UI, written directly (seedLabels).
````

````old server/internal/bootstrap/project_deletion_test.go
	return contract, base, pool, alice, aliceID, createdProject(t, contract, base, alice, "acme", "Web", "WEB"),
		createdProject(t, contract, base, alice, "acme", "Ops", "OPS")
````
````new server/internal/bootstrap/project_deletion_test.go
	web, ops = createdProject(t, contract, base, alice, "acme", "Web", "WEB"), createdProject(t, contract, base, alice, "acme", "Ops", "OPS")
	seedLabels(t, pool, web, aliceID)
	seedLabels(t, pool, ops, aliceID)
	return contract, base, pool, alice, aliceID, web, ops
}

// seedLabels writes a label of project, Bug, and its child, UI, by by,
// directly: the seed does not depend on which of the project module's
// writes exist.
func seedLabels(t *testing.T, pool *pgxpool.Pool, project, by uuid.UUID) {
	t.Helper()
	if tag, err := pool.Exec(pgtest.Soon(t), `INSERT INTO labels (id, workspace_id, project_id, parent_id, name, created_by_id, updated_by_id)
		SELECT $1::uuid, workspace_id, id, NULL::uuid, 'Bug', $2::uuid, $2::uuid FROM projects WHERE id = $3
		UNION ALL SELECT $4::uuid, workspace_id, id, $1::uuid, 'UI', $2::uuid, $2::uuid FROM projects WHERE id = $3`,
		uuid.NewV7(), by, project, uuid.NewV7()); err != nil || tag.RowsAffected() != 2 {
		t.Fatalf("seeding the labels of %s: %d rows, %v; want 2", project, tag.RowsAffected(), err)
	}
````

`server/internal/bootstrap/workspace_deletion_test.go`（修改，7 处）：

````old server/internal/bootstrap/workspace_deletion_test.go
// display settings in it and a state (seedProject). A phase that adds a
// table under workspaces seeds a row of it here. It returns the workspace's
// id.
````
````new server/internal/bootstrap/workspace_deletion_test.go
// display settings in it, a state and two labels (seedProject). A phase
// that adds a table under workspaces seeds a row of it here. It returns the
// workspace's id.
````

````old server/internal/bootstrap/workspace_deletion_test.go
// with member's membership, his display settings in it and one state,
// directly: the seed does not depend on which of the project module's
// writes exist, nor on what they write besides.
````
````new server/internal/bootstrap/workspace_deletion_test.go
// with member's membership, his display settings in it, one state, and a
// label with its child by member (seedLabels), directly: the seed does not
// depend on which of the project module's writes exist, nor on what they
// write besides. No row under the project is admin's last, so a deletion
// by admin that kept a row's writer would show.
````

````old server/internal/bootstrap/workspace_deletion_test.go
			t.Fatal(err)
		}
	}
}
````
````new server/internal/bootstrap/workspace_deletion_test.go
			t.Fatal(err)
		}
	}
	seedLabels(t, pool, project, member)
}
````

````old server/internal/bootstrap/workspace_deletion_test.go
// workspace's own. The states table, renamed while the request runs, fails
````
````new server/internal/bootstrap/workspace_deletion_test.go
// workspace's own. The labels table, renamed while the request runs, fails
````

````old server/internal/bootstrap/workspace_deletion_test.go
	exec("ALTER TABLE states RENAME TO states_away")
````
````new server/internal/bootstrap/workspace_deletion_test.go
	exec("ALTER TABLE labels RENAME TO labels_away")
````

````old server/internal/bootstrap/workspace_deletion_test.go
	exec("ALTER TABLE states_away RENAME TO states")
````
````new server/internal/bootstrap/workspace_deletion_test.go
	exec("ALTER TABLE labels_away RENAME TO labels")
````

````old server/internal/bootstrap/workspace_deletion_test.go
		t.Fatalf("deleting the workspace with the states' step failing = %d %s, want 500", status, body)
````
````new server/internal/bootstrap/workspace_deletion_test.go
		t.Fatalf("deleting the workspace with the labels' step failing = %d %s, want 500", status, body)
````

- [ ] **Step 5: 端到端的过渡**

`e2e/fixtures/assert/project.ts`（修改，2 处）：

````old e2e/fixtures/assert/project.ts
  expect(project, `the project ${projectId}`).toEqual({ deleted: true, by: adminEmail });
````
````new e2e/fixtures/assert/project.ts
  expect(project, `the project ${projectId}`).toEqual({ deleted: true, by: adminEmail });
  // The labels are left out until the stories can write one: P7b's createLabel gives them the way, and P4 and W3 then
  // write and check them.
````

````old e2e/fixtures/assert/project.ts
      WHERE c.contype = 'f' AND c.confrelid = 'projects'::regclass ORDER BY 1`
````
````new e2e/fixtures/assert/project.ts
      WHERE c.contype = 'f' AND c.confrelid = 'projects'::regclass AND c.conrelid <> 'labels'::regclass ORDER BY 1`
````

- [ ] **Step 6: 测试和 lint**

Run: `go -C server test -count=1 ./migrations/ ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestDeletingAProjectLeavesNoUndeletedRowUnderIt$|TestAProjectDeletionRefusedAtItsCommitChangesNoRow$|TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt$|TestAFailedProjectsStepRollsTheDeletionBack$|TestADeletionRefusedAtItsCommitChangesNoRow$|TestDeletionViolationsCatchesEachGap$|TestKeysRefuseATableWithoutItsParent$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过（关键词守卫 5 个命中都有例外）。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 69 个全部通过（P4、W3 删除项目、工作区时，`expectProjectDeleted` 暂时不读 `labels`）。

- [ ] **Step 7: 提交**

```bash
git add api/modules/project.yaml api/modules/workspace.yaml deploy/runtime-grants.sql e2e/fixtures/assert/project.ts server/internal/bootstrap/project_deletion_test.go server/internal/bootstrap/workspace_deletion_test.go server/internal/modules/project/adapter/postgres/cascade.go server/internal/modules/project/adapter/postgres/cascade_test.go server/internal/modules/project/adapter/postgres/queries/cascade.sql server/internal/modules/project/app/cascade_test.go server/internal/modules/project/app/delete_project_test.go server/internal/modules/project/app/deletion.go server/internal/modules/project/app/fakes_write_test.go server/internal/modules/project/app/ports.go server/migrations/project_schema_test.go server/migrations/schema_test.go server/migrations/sql/00014_project_labels.sql server/sqlc.yaml api/dist/openapi.yaml server/internal/modules/project/adapter/postgres/gen/cascade.sql.go server/internal/modules/project/adapter/postgres/gen/models.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P7b): the labels table, and the deletions of projects and workspaces take their labels

Migration 00014 adds labels: a project's alone, its parent a label of
the same table, never itself, the name unique in its project in any
case among undeleted labels, and the indexes the physical cascade and
a parent's deletion look rows up by. Deleting a project, and deleting a
workspace, delete the labels last, at the deletion's moment and by its
account, in the one list of steps both run. The end-to-end check of a
deleted project leaves labels out until the stories can write one.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p7b.py` 的编号；"层"是它被发现的每一层，端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `s1-cl-workspace` | `DeleteLabels` 去掉 `workspace_id`（删除每个工作区的标签） | `TestDeletingAWorkspaceSoftDeletesItsProjects`、`TestDeletingAnotherWorkspacesProjectChangesNothing`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`、`TestListingProjectsIsReadingEach`（Task 5 起）、`TestListingWorkspaceStatesIsListingEachProjects`（Task 5 起）、`TestPermissionMatrix`（Task 5 起） 等 5 个、W3（Task 11 起） | 存储；组合；端到端 |
| `s1-cl-project` | `DeleteLabels` 去掉 `project_id`（删除项目时删除工作区的全部标签） | `TestDeletingAProjectSoftDeletesItsRowsAlone`、`TestDeletingAnotherWorkspacesProjectChangesNothing`、`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、W3（Task 11 起） | 存储；组合；端到端 |
| `s1-cl-deleted` | `DeleteLabels` 去掉 `deleted_at IS NULL`（已删除的标签改写时刻） | `TestDeletingAProjectSoftDeletesItsRowsAlone`、`TestDeletingAWorkspaceSoftDeletesItsProjects`、P4（Task 11 起）、W3（Task 11 起） | 存储；端到端 |
| `s1-idx-case` | 名称的唯一索引区分大小写（`name` 代替 `lower(name)`） | `TestProjectUniqueKeysHoldAmongUndeletedRowsOnly`、`TestCreateLabelNameTaken`（Task 3 起）、`TestUpdateLabelNameTaken`（Task 3 起）、`TestPermissionMatrix`（Task 5 起）、`TestEachLabelWriteChangesItsRowsAlone`（Task 10 起）、`TestLabelWritesOnOneProjectSerialize`（Task 10 起）、P7（Task 11 起） | 结构；存储；组合；端到端 |
| `s1-idx-deleted` | 名称的唯一索引去掉 `WHERE deleted_at IS NULL` | `TestConstraintAndIndexNames`、`TestProjectUniqueKeysHoldAmongUndeletedRowsOnly`、`TestCreateLabelNameTaken`（Task 3 起）、`TestUpdateLabelNameTaken`（Task 3 起）、P7（Task 11 起） | 结构；存储；端到端 |
| `s1-check-own-parent` | 迁移去掉 `labels_not_own_parent_check` | `TestConstraintAndIndexNames`、`TestProjectChecksRejectCounterexamples`、`TestCreateLabelBreakingAnotherConstraintIsInternal`（Task 3 起） | 结构；存储 |
| `s14-cl-keeps-writer` | `DeleteLabels` 保留原来的 `updated_by_id` | `TestDeletingAProjectSoftDeletesItsRowsAlone`、`TestDeletingAWorkspaceSoftDeletesItsProjects`、`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt`、W3（Task 11 起） | 存储；组合；端到端 |
| `s3-cl-keeps-moment` | `DeleteLabels` 保留原来的 `updated_at` | `TestDeletingAProjectSoftDeletesItsRowsAlone`、`TestDeletingAWorkspaceSoftDeletesItsProjects`、P4（Task 11 起）、W3（Task 11 起） | 存储；端到端 |
| `s8-cascade` | `Store.DeleteLabels` 在连接池上执行 | `TestADeletionRefusedAtItsCommitChangesNoRow`、`TestAProjectDeletionRefusedAtItsCommitChangesNoRow`、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection` | 组合 |
| `s19-cascade` | `Store.DeleteLabels` 的失败答成成功 | `TestAFailedWriteIsAnError` | 存储 |
| `k-no-labels-step` | `deleteProjects` 去掉标签一步（删除项目、工作区留下标签） | `TestDeleteProject`、`TestDeleteWorkspaceProjects`、`TestEachWriteReadsTheClockUnderItsLock`、`TestDeleteProjectReturnsEachFailure`、`TestAFailedProjectsStepRollsTheDeletionBack`、`TestAProjectDeletionRefusedAtItsCommitChangesNoRow`、`TestDeletingAProjectLeavesNoUndeletedRowUnderIt`、`TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt` 等 8 个、P4（Task 11 起）、W3（Task 11 起） | 单元；组合；端到端 |

---

### Task 2: 标签的规则：`Label`、`CheckNewLabel`、`CheckLabelPatch`、`SortOrderAfter`、`CheckParent`、两个标签码

**Files:**
- Create: `server/internal/modules/project/domain/label.go`、`server/internal/modules/project/domain/label_test.go`
- Modify: `server/internal/modules/project/domain/errors.go`、`server/internal/modules/project/domain/project.go`、`server/internal/modules/project/domain/state.go`、`server/internal/modules/project/domain/state_test.go`

**Interfaces:**
- Produces（spec 2.4；M3 设计 3.16、4.10、5.2、5.3）：
  - `project/domain/label.go`（新）：`Label{ID, WorkspaceID, ProjectID, ParentID *uuid.UUID, Name, Color, SortOrder, CreatedAt, UpdatedAt}`、`Label.Place()`；`LabelCreate{Name, Color, ParentID *uuid.UUID, SortOrder *float64}`；`LabelPatch{Name, Color *string; SetParent bool; ParentID *uuid.UUID; SortOrder *float64}`（`SetParent` 说父标签给了没有，给了且 `ParentID == nil` 是移到顶层）；`CheckNewLabel(LabelCreate) error`、`CheckLabelPatch(LabelPatch) error`（名称 1–255 个字符、不空白、不含 NUL；颜色至多 255 个字符、不含 NUL、可以为空；有问题的字段一次全部报告）；`SortOrderAfter(greatest *float64) float64`（加 10000；`nil` 时 65535）；`CheckParent(id, project uuid.UUID, parent *Label, hasChildren bool) error`（依次：不是这个项目的标签、是它自己、父标签有父标签、这个标签有子标签，各一个 422 `parent_id` `not_allowed`，说明各不相同）。
  - `project/domain/errors.go`：`ErrLabelNotFound`（404 `project.label_not_found`）、`ErrLabelNameTaken`（409 `project.label_name_taken`）。它们进契约在 Task 5、7。
  - `project.go`：`checkLength` 拆出 `checkMaxLength`（只看长度，标签的颜色用它），加 `checkRequiredText(field, s, limit)`（`checkLength` 再 `checkText`，第一个问题）；`state.go` 的 `checkStateText` 删去，状态的名称和颜色改用 `checkRequiredText(…, maxStateText)`（第一次运行的 T9 与 P7a 的 T2-d 的手工合并，spec 附录 A：没有第二个长度检查）。

**Tests:**
- `TestLabelPlace`（id、工作区、项目各在它的位置）、`TestCheckNewLabelAcceptsValidLabels`（1 个和 255 个字符的名称、空的和 255 个字符的颜色；父标签和 `sort_order` 不在这里查）、`TestCheckLabelReportsEveryField`（名称空、空白、256 个字符、含 NUL；颜色 256 个字符、含 NUL；两个问题的字段报第一个：太长先于 NUL；`CheckLabelPatch` 只查给了的字段：空补丁、只给 `sort_order`、给父标签或移到顶层都通过）、`TestSortOrderAfter`（10000 之后；没有时 65535）、`TestCheckParent`（顶层的标签做新标签、没有子标签的标签的父标签；四种拒绝，各一个 422 `parent_id not_allowed`，按切片的顺序）。
- `state_test.go` 的说明：状态的检查与标签共用名称的检查；`TestCheckNewStateReportsEveryField` 等照旧通过（行为不变）。

- [ ] **Step 1: 规则、码和测试**

`server/internal/modules/project/domain/project.go`（修改，2 处）：

````old server/internal/modules/project/domain/project.go
	switch {
	case strings.TrimSpace(s) == "":
````
````new server/internal/modules/project/domain/project.go
	if strings.TrimSpace(s) == "" {
````

````old server/internal/modules/project/domain/project.go
	case utf8.RuneCountInString(s) > limit:
		return &shared.FieldError{Field: field, Code: shared.FieldTooLong, Message: fmt.Sprintf("must be at most %d characters", limit)}
	}
	return nil
}
````
````new server/internal/modules/project/domain/project.go
	}
	return checkMaxLength(field, s, limit)
}

// checkMaxLength refuses the text of field, a column of varchar(limit),
// when it is longer than limit characters; it may be empty.
func checkMaxLength(field, s string, limit int) *shared.FieldError {
	if utf8.RuneCountInString(s) > limit {
		return &shared.FieldError{Field: field, Code: shared.FieldTooLong, Message: fmt.Sprintf("must be at most %d characters", limit)}
	}
	return nil
}

// checkRequiredText refuses the text of field, a column of varchar(limit)
// that may not be blank: blank or too long (checkLength), or with NUL
// (checkText), the first of them.
func checkRequiredText(field, s string, limit int) *shared.FieldError {
	if f := checkLength(field, s, limit); f != nil {
		return f
	}
	return checkText(field, s)
}
````

`server/internal/modules/project/domain/state.go`（修改，4 处）：

````old server/internal/modules/project/domain/state.go
	return invalid(checkStateText("name", s.Name), checkStateText("color", s.Color), checkGroup(s.Group),
````
````new server/internal/modules/project/domain/state.go
	return invalid(checkRequiredText("name", s.Name, maxStateText), checkRequiredText("color", s.Color, maxStateText), checkGroup(s.Group),
````

````old server/internal/modules/project/domain/state.go
		found = append(found, checkStateText("name", *p.Name))
````
````new server/internal/modules/project/domain/state.go
		found = append(found, checkRequiredText("name", *p.Name, maxStateText))
````

````old server/internal/modules/project/domain/state.go
		found = append(found, checkStateText("color", *p.Color))
````
````new server/internal/modules/project/domain/state.go
		found = append(found, checkRequiredText("color", *p.Color, maxStateText))
````

````old server/internal/modules/project/domain/state.go

func checkStateText(field, s string) *shared.FieldError {
	if f := checkLength(field, s, maxStateText); f != nil {
		return f
	}
	return checkText(field, s)
}

````
````new server/internal/modules/project/domain/state.go

````

`server/internal/modules/project/domain/label.go`（新文件，129 行）：

````file server/internal/modules/project/domain/label.go
package domain

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Label is a label of a project as stored and as the API shows it (M3
// design 3.16, 5.2): a project's alone, of two levels at most, its parent
// nil at the top.
type Label struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	ParentID    *uuid.UUID
	Name        string
	Color       string
	SortOrder   float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Place is the label's id, its project and the project's workspace: where
// a write on it takes its locks.
func (l Label) Place() (id, workspaceID, projectID uuid.UUID) {
	return l.ID, l.WorkspaceID, l.ProjectID
}

// LabelCreate is what the caller asks for when creating a label (M3 design
// 5.1): its parent nil for a label at the top, its sort order nil for the
// project's to give (SortOrderAfter).
type LabelCreate struct {
	Name      string
	Color     string
	ParentID  *uuid.UUID
	SortOrder *float64
}

// LabelPatch is what updateLabel changes (M3 design 5.1): each field nil
// when it is not given, and kept; SetParent says whether the parent is
// given, ParentID nil then for none.
type LabelPatch struct {
	Name      *string
	Color     *string
	SetParent bool
	ParentID  *uuid.UUID
	SortOrder *float64
}

// maxLabelText is the length of labels.name and labels.color,
// varchar(255), in characters.
const maxLabelText = 255

// sortOrderStep is the gap between a project's last label and a new one,
// and firstSortOrder the sort order of a project's first label: Plane's
// Label.save() and the column's default (M3 design 4.10).
const (
	sortOrderStep  = 10000
	firstSortOrder = 65535
)

// CheckNewLabel checks l (M3 design 3.16): a name of 1–255 characters, not
// blank, without NUL; a color of at most 255 characters, empty for none,
// without NUL. Every field with a problem is reported at once, in one 422
// validation_failed. Whether the name is taken, in any case, is the
// database's to say; whether the parent may be, the use case's, under the
// project's lock (CheckParent).
func CheckNewLabel(l LabelCreate) error {
	return invalid(checkRequiredText("name", l.Name, maxLabelText), checkLabelColor(l.Color))
}

// CheckLabelPatch checks the fields p gives by CheckNewLabel's rules; a
// sort order is any number. Every field with a problem is reported at
// once.
func CheckLabelPatch(p LabelPatch) error {
	var found []*shared.FieldError
	if p.Name != nil {
		found = append(found, checkRequiredText("name", *p.Name, maxLabelText))
	}
	if p.Color != nil {
		found = append(found, checkLabelColor(*p.Color))
	}
	return invalid(found...)
}

// SortOrderAfter is the sort order of a new label of a project whose labels
// have greatest as their greatest sort order, nil when it has none: one
// step after it, or the column's default (Plane's Label.save(); M3 design
// 4.10).
func SortOrderAfter(greatest *float64) float64 {
	if greatest == nil {
		return firstSortOrder
	}
	return *greatest + sortOrderStep
}

// CheckParent refuses parent as the parent of the label id of project,
// uuid.Nil for a new label, which has children when hasChildren (M3 design
// 3.16): the parent of a label is an undeleted label of its project, not
// itself, that has no parent of its own; a label with children has none.
// parent is nil for a parent that is not there or is deleted. Each refusal
// is one 422 validation_failed, parent_id not_allowed.
func CheckParent(id, project uuid.UUID, parent *Label, hasChildren bool) error {
	var why string
	switch {
	case parent == nil || parent.ProjectID != project:
		why = "must be a label of the project"
	case parent.ID == id:
		why = "must not be the label itself"
	case parent.ParentID != nil:
		why = "must be a label without a parent: labels have two levels"
	case hasChildren:
		why = "must be null: the label has labels under it, and labels have two levels"
	default:
		return nil
	}
	return shared.Invalid(shared.FieldError{Field: "parent_id", Code: shared.FieldNotAllowed, Message: why})
}

// checkLabelColor refuses a color longer than labels.color or with NUL; it
// may be empty, for none.
func checkLabelColor(c string) *shared.FieldError {
	if f := checkMaxLength("color", c, maxLabelText); f != nil {
		return f
	}
	return checkText("color", c)
}
````

`server/internal/modules/project/domain/errors.go`（修改，1 处）：

````old server/internal/modules/project/domain/errors.go
		"The state is the only one of its group, and every group keeps one; add another to the group first.")
````
````new server/internal/modules/project/domain/errors.go
		"The state is the only one of its group, and every group keeps one; add another to the group first.")
	// ErrLabelNotFound answers a label that does not exist, is deleted, or
	// whose project the caller does not see: the same 404 for all (M3 design
	// 3.16, 5.3, 8.2).
	ErrLabelNotFound = shared.NewError(shared.KindNotFound, "project.label_not_found",
		"The label does not exist, or you cannot see its project.")
	// ErrLabelNameTaken answers a name another undeleted label of the
	// project has, in any case (M3 design 3.16).
	ErrLabelNameTaken = shared.NewError(shared.KindConflict, "project.label_name_taken",
		"A label of the project has this name, in this case or another.")
````

`server/internal/modules/project/domain/label_test.go`（新文件，120 行）：

````file server/internal/modules/project/domain/label_test.go
package domain

import (
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Place answers the label's own id, workspace and project, each in its
// place.
func TestLabelPlace(t *testing.T) {
	l := Label{ID: uuid.NewV7(), WorkspaceID: uuid.NewV7(), ProjectID: uuid.NewV7()}
	if id, workspace, project := l.Place(); id != l.ID || workspace != l.WorkspaceID || project != l.ProjectID {
		t.Errorf("Place() = %s, %s, %s; want %s, %s, %s", id, workspace, project, l.ID, l.WorkspaceID, l.ProjectID)
	}
}

// CheckNewLabel accepts a name of one and of 255 characters, a color empty
// or of 255 characters, and leaves the parent and the sort order to others.
func TestCheckNewLabelAcceptsValidLabels(t *testing.T) {
	parent := uuid.NewV7()
	for _, l := range []LabelCreate{
		{Name: "Bug", Color: "#FF0000"},
		{Name: "B"},
		{Name: strings.Repeat("标", 255), Color: strings.Repeat("c", 255), ParentID: &parent, SortOrder: ptr(-1.0)},
	} {
		if err := CheckNewLabel(l); err != nil {
			t.Errorf("CheckNewLabel(%+v) = %v, want nil", l, err)
		}
	}
}

// CheckNewLabel and CheckLabelPatch report every field with a problem at
// once: a name empty, blank, of 256 characters or with NUL; a color of 256
// characters or with NUL, which may be empty. A field with two problems
// reports its first: too long before NUL. CheckLabelPatch checks only the
// fields given: an empty patch, a sort order, a parent or none pass.
func TestCheckLabelReportsEveryField(t *testing.T) {
	for _, tt := range []struct {
		name string
		l    LabelCreate
		want []shared.FieldError
	}{
		{"empty name", LabelCreate{Name: ""}, []shared.FieldError{nameEmpty}},
		{"blank name", LabelCreate{Name: " \n"}, []shared.FieldError{nameEmpty}},
		{"name of 256 characters", LabelCreate{Name: strings.Repeat("标", 256)}, []shared.FieldError{nameLong}},
		{"name with NUL", LabelCreate{Name: "B\x00ug"}, []shared.FieldError{nameNUL}},
		{"name of 256 characters with NUL", LabelCreate{Name: strings.Repeat("标", 255) + "\x00"}, []shared.FieldError{nameLong}},
		{"color of 256 characters", LabelCreate{Name: "Bug", Color: strings.Repeat("c", 256)}, []shared.FieldError{colorLong}},
		{"color with NUL", LabelCreate{Name: "Bug", Color: "#\x00"}, []shared.FieldError{colorNUL}},
		{"color of 256 characters with NUL", LabelCreate{Name: "Bug", Color: strings.Repeat("c", 255) + "\x00"},
			[]shared.FieldError{colorLong}},
		{"all at once", LabelCreate{Name: "", Color: "\x00"}, []shared.FieldError{nameEmpty, colorNUL}},
	} {
		if got := fieldsOf(t, CheckNewLabel(tt.l)); !slices.Equal(got, tt.want) {
			t.Errorf("%s: CheckNewLabel() fields %+v, want %+v", tt.name, got, tt.want)
		}
		p := LabelPatch{Name: &tt.l.Name, Color: &tt.l.Color}
		if got := fieldsOf(t, CheckLabelPatch(p)); !slices.Equal(got, tt.want) {
			t.Errorf("%s: CheckLabelPatch() fields %+v, want %+v", tt.name, got, tt.want)
		}
	}
	parent := uuid.NewV7()
	for _, p := range []LabelPatch{{}, {SortOrder: ptr(-1.0)}, {SetParent: true}, {SetParent: true, ParentID: &parent},
		{Name: ptr("Bug"), Color: ptr("")}} {
		if err := CheckLabelPatch(p); err != nil {
			t.Errorf("CheckLabelPatch(%+v) = %v, want nil", p, err)
		}
	}
}

// A new label comes 10000 after the greatest sort order of its project's
// labels, and at the column's default, 65535, when the project has none.
func TestSortOrderAfter(t *testing.T) {
	for _, tt := range []struct {
		greatest *float64
		want     float64
	}{{nil, 65535}, {ptr(65535.0), 75535}, {ptr(-1.5), 9998.5}, {ptr(0.0), 10000}} {
		if got := SortOrderAfter(tt.greatest); got != tt.want {
			t.Errorf("SortOrderAfter(%v) = %v, want %v", tt.greatest, got, tt.want)
		}
	}
}

// CheckParent takes a label of the project at the top as the parent of a
// new label, or of a label without children, and refuses, each as one 422
// parent_id not_allowed: no parent, a label of another project, the label
// itself, a label that has a parent, and any parent of a label with
// children (M3 design 3.16).
func TestCheckParent(t *testing.T) {
	web, ops, bug, ui := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	top := &Label{ID: bug, ProjectID: web}
	child := &Label{ID: ui, ProjectID: web, ParentID: &bug}
	refused := func(why string) []shared.FieldError {
		return []shared.FieldError{{Field: "parent_id", Code: "not_allowed", Message: why}}
	}
	for _, tt := range []struct {
		name        string
		id          uuid.UUID
		parent      *Label
		hasChildren bool
		want        []shared.FieldError
	}{
		{"a new label under a label at the top", uuid.Nil(), top, false, nil},
		{"a label without children under a label at the top", uuid.NewV7(), top, false, nil},
		{"no parent", uuid.Nil(), nil, false, refused("must be a label of the project")},
		{"a label of another project", uuid.Nil(), &Label{ID: uuid.NewV7(), ProjectID: ops}, false, refused("must be a label of the project")},
		{"the label itself", bug, top, false, refused("must not be the label itself")},
		{"a label that has a parent", uuid.Nil(), child, false, refused("must be a label without a parent: labels have two levels")},
		{"a label with children", uuid.NewV7(), top, true,
			refused("must be null: the label has labels under it, and labels have two levels")},
	} {
		if got := fieldsOf(t, CheckParent(tt.id, web, tt.parent, tt.hasChildren)); !slices.Equal(got, tt.want) {
			t.Errorf("%s: CheckParent() fields %+v, want %+v", tt.name, got, tt.want)
		}
	}
}
````

`server/internal/modules/project/domain/state_test.go`（修改，1 处）：

````old server/internal/modules/project/domain/state_test.go
// The problems CheckNewState and CheckStatePatch report: one per field,
// each field's first.
````
````new server/internal/modules/project/domain/state_test.go
// The problems CheckNewState and CheckStatePatch report, and the label
// checks (label_test.go) of the fields they share: one per field, each
// field's first.
````

- [ ] **Step 2: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/domain/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`（`Label` 等在 Task 3 起才有使用者；它们是导出的，lint 不报未使用）

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/modules/project/domain/errors.go server/internal/modules/project/domain/label.go server/internal/modules/project/domain/label_test.go server/internal/modules/project/domain/project.go server/internal/modules/project/domain/state.go server/internal/modules/project/domain/state_test.go
```
```bash
git commit -m "feat(M3/P7b): the rules of a project's labels

A label is a project's, at the top or under a label at the top. Its
name has 1-255 characters, not blank, without NUL; its color at most
255, empty for none. A new label comes 10000 after the greatest sort
order of its project's labels, or at 65535. A parent is a label of the
project, not the label itself, without a parent of its own, and a label
with labels under it takes none: each refusal is parent_id not_allowed.
States and labels share one check of a required text.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p7b.py` 的编号；"层"是它被发现的每一层，端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `r-parent-any-project` | `CheckParent` 收下另一个项目的标签做父标签 | `TestCheckParent`、`TestCreateLabelRefuses`（Task 4 起）、`TestUpdateLabelRefuses`（Task 6 起）、`TestPermissionMatrix`（Task 5 起）、P7（Task 11 起） | 单元；组合；端到端 |
| `r-parent-self` | `CheckParent` 收下标签自己做父标签 | `TestCheckParent`、`TestUpdateLabelRefuses`（Task 6 起）、P7（Task 11 起） | 单元；端到端 |
| `r-parent-third-level` | `CheckParent` 收下有父标签的标签做父标签（第三层） | `TestCheckParent`、`TestCreateLabelRefuses`（Task 4 起）、`TestUpdateLabelRefuses`（Task 6 起）、`TestPermissionMatrix`（Task 5 起）、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 5 起）、`TestLabelWritesOnOneProjectSerialize`（Task 10 起）、P7（Task 11 起） | 单元；组合；端到端 |
| `r-parent-children` | `CheckParent` 让有子标签的标签有父标签 | `TestCheckParent`、`TestUpdateLabelRefuses`（Task 6 起）、`TestPermissionMatrix`（Task 7 起）、`TestEachLabelWriteChangesItsRowsAlone`（Task 10 起）、`TestLabelWritesOnOneProjectSerialize`（Task 10 起）、P7（Task 11 起） | 单元；组合；端到端 |
| `q-step` | 新标签在最大值之后 15000 | `TestEachWriteReadsTheClockUnderItsLock`、`TestSortOrderAfter`、`TestCreateLabel`（Task 4 起）、`TestCreateLabelRefuses`（Task 4 起） 等 5 个、`TestPermissionMatrix`（Task 5 起）、P7（Task 11 起） | 单元；组合；端到端 |
| `q-first` | 项目的第一个标签在 0 | `TestSortOrderAfter`、`TestCreateLabel`（Task 4 起）、P7（Task 11 起） | 单元；端到端 |
| `v-color-unchecked` | `CheckNewLabel` 不查颜色 | `TestCheckLabelReportsEveryField`、P7（Task 11 起） | 单元；端到端 |
| `v-patch-name-unchecked` | `CheckLabelPatch` 不查名称 | `TestCheckLabelReportsEveryField`、`TestUpdateLabelRefuses`（Task 6 起）、`TestPermissionMatrix`（Task 7 起） | 单元；组合 |

---

### Task 3: 标签的存储：七条语句、存储方法、存储测试和失败测试

**Files:**
- Create: `server/internal/modules/project/adapter/postgres/label_writes_test.go`、`server/internal/modules/project/adapter/postgres/labels.go`、`server/internal/modules/project/adapter/postgres/labels_test.go`、`server/internal/modules/project/adapter/postgres/queries/labels.sql`、`server/internal/modules/project/app/label_ports.go`
- Modify: `server/internal/modules/project/adapter/postgres/failures_test.go`、`server/internal/modules/project/adapter/postgres/states_test.go`
- Generate: `server/internal/modules/project/adapter/postgres/gen/labels.sql.go`

**Interfaces:**
- Produces（spec 2.5；M3 设计 3.6、3.16、6.7）：`queries/labels.sql`（新，spec 2.5 有全文）：`CreateLabel`、`LabelByID`、`ListLabels`、`GreatestSortOrder`、`HasChildren`、`UpdateLabel`、`DeleteLabel`。
- `postgresadapter.Store` 的方法（`labels.go`），都在 `ctx` 带着的事务里执行（写的都在项目的 `FOR NO KEY UPDATE` 之下，Task 4–8）：
  - `CreateLabel(ctx, app.LabelRow) (domain.Label, error)`：同名（不分大小写）是 `domain.ErrLabelNameTaken`，只认约束名 `labels_project_id_name_key`（`labelTaken`）；别的唯一键、CHECK、外键是内部错误。
  - `LabelByID(ctx, id) (domain.Label, bool, error)`：未删除的。
  - `ListLabels(ctx, projectID) ([]domain.Label, error)`：未删除的，父子一起，按 `sort_order`、再按 `id`；已归档的项目照常。
  - `GreatestSortOrder(ctx, projectID) (*float64, error)`：未删除的标签（子标签也算）的最大值；没有时 `nil`。
  - `HasChildren(ctx, id) (bool, error)`：有没有未删除的标签以它为父标签。
  - `UpdateLabel(ctx, id, domain.LabelPatch, by, now) (domain.Label, error)`：只改给了的字段（`coalesce`；父标签在 `SetParent` 时改，`nil` 是顶层），审计列改为给的账户和时刻；同名是 `ErrLabelNameTaken`；已删除的不写，是错误。
  - `DeleteLabel(ctx, id, by, now) error`：一条语句，这个未删除的标签和它下面未删除的标签，`deleted_at = updated_at = now`、`updated_by_id = by`。
- `app.LabelRow{ID, WorkspaceID, ProjectID, ParentID, Name, Color, SortOrder, CreatedBy, Now}`（`label_ports.go`，新；Task 4 起加端口）。

**Tests:**（`labels_test.go`、`label_writes_test.go`；P7a 的 `stateWorld`：acme 的 Web、Ops（已归档），beta 的 Site；`addLabel` 经 `CreateLabel` 存，由 maker 在 `earlier`，测试的写都由 alice；`tableRows`、`columns` 比较被写的行之外的每一行每一列；持有另一个事务时运行的语句在 `pgtest.Soon(t)` 的期限上）
- `TestCreateLabel`（每一列与给的相同，父标签也是，由 alice、在给的时刻，未删除；别的标签每一列不变）、`TestCreateLabelNameTaken`（Bug 旁的 bug、BUG 各 409，是 API 会答的第一个问题，什么都不存；已删除的标签的名称、别的项目的名称可以）、`TestCreateLabelBreakingAnotherConstraintIsInternal`（重复的 id、空名称、自己做自己的父标签、不存在的父标签，各是那个约束的违反（`*pgconn.PgError` 的约束名），不是 `*shared.Error`）。
- `TestLabelByID`（每列与 `CreateLabel` 的回答相同，父标签也是；两行各问一次，读表里第一行的实现会答错其中一个；已删除的、不存在的没有）、`TestListLabels`（Feature 最后加，`sort_order` 与 UI 相同而 id 更小：排在 UI 之前，行在表里的次序给不出（清扫 48）；不含已删除的 Old、别的项目的；已归档的 Ops 照常列出）、`TestGreatestSortOrder`（Web 的 UI 75535：子标签也算，不是已删除的 99999、Ops 的 80000；没有标签的项目 `nil`）、`TestHasChildren`（Bug 有 UI；UI 没有；Feature 唯一的子标签已删除；不存在的 id 没有）。
- `TestUpdateLabel`（Web 的 UI：给了的字段改，审计列改为给的；别的列、别的标签不变；父标签给了就设、给 `nil` 就清、没给就留；什么都不给的补丁跟在给了全部字段的补丁之后，它保留的父标签、颜色、`sort_order` 既不是空也不是列的默认值（清扫 26））、`TestUpdateLabelNameTaken`（Bug 的另一种大小写 409，什么都不改；已删除的、别的项目的名称可以，自己的名称换大小写可以）、`TestUpdateLabelWritesNoDeletedLabel`（内部错误，不是领域错误，每个标签每一列不变）、`TestDeleteLabel`（Web 的 Bug 和它下面的 UI 在给的时刻、由给的账户删除，`deleted_at` 与 `updated_at` 相同，别的列不动；在 Bug 下、之前已删除的 Gone 保留它的时刻；Feature、Feature 的 Docs、Ops 的 Bug 每一列不变；再删 Bug 什么都不改；删除子标签 Docs 只删它自己）。
- `failures_test.go`：`TestAFailedReadIsAnErrorNotAnAnswer` 加 `LabelByID`、`ListLabels`、`GreatestSortOrder`、`HasChildren`；`TestAFailedWriteIsAnError` 加 `CreateLabel`、`UpdateLabel`、`DeleteLabel`（清扫 19；`failed`：`errors.Is(err, context.Canceled)` 且不是 `*shared.Error`；`seedLabel` 给 Web 一个标签和它的子标签，读的零值都不是对的答案）。`states_test.go`：`workspaceOf`（标签的测试也用）。

- [ ] **Step 1: 查询**

`server/internal/modules/project/adapter/postgres/queries/labels.sql`（新文件，58 行）：

````file server/internal/modules/project/adapter/postgres/queries/labels.sql
-- name: CreateLabel :one
-- createLabel, under the project's FOR NO KEY UPDATE (M3 design 3.16): the row as stored.
INSERT INTO labels (id, workspace_id, project_id, parent_id, name, color, sort_order, created_by_id, updated_by_id, created_at,
                    updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.narg(parent_id), sqlc.arg(name), sqlc.arg(color),
        sqlc.arg(sort_order), sqlc.arg(created_by), sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now))
RETURNING id, workspace_id, project_id, parent_id, name, color, sort_order, created_at, updated_at;

-- name: LabelByID :one
-- A write on a label named by its id (M3 design 3.6 convention 2, 6.7), read first without a lock for its project and
-- the project's workspace, then again under their locks; and a label named as a parent, under the project's lock: the
-- undeleted label.
SELECT id, workspace_id, project_id, parent_id, name, color, sort_order, created_at, updated_at
FROM labels
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: ListLabels :many
-- listLabels (M3 design 3.16, 3.19): the project's undeleted labels, parents and children alike, by sort order, then id;
-- an archived project's too.
SELECT id, workspace_id, project_id, parent_id, name, color, sort_order, created_at, updated_at
FROM labels
WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL
ORDER BY sort_order, id;

-- name: GreatestSortOrder :one
-- createLabel, under the project's FOR NO KEY UPDATE (M3 design 4.10): the greatest sort order of the project's
-- undeleted labels, parents and children alike, as Plane's Label.save() reads it; no row when it has none.
SELECT sort_order
FROM labels
WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL
ORDER BY sort_order DESC
LIMIT 1;

-- name: HasChildren :one
-- updateLabel, under the project's FOR NO KEY UPDATE (M3 design 3.16): whether an undeleted label has the label as its
-- parent.
SELECT EXISTS (SELECT 1 FROM labels WHERE parent_id = sqlc.arg(id)::uuid AND deleted_at IS NULL);

-- name: UpdateLabel :one
-- updateLabel, under the project's FOR NO KEY UPDATE (M3 design 3.16, 6.7): a field left out, null here, keeps its value;
-- the parent changes when set_parent is true, to none for a null parent_id. A deleted label is not written.
UPDATE labels l
SET name          = coalesce(sqlc.narg(name)::text, l.name),
    color         = coalesce(sqlc.narg(color)::text, l.color),
    parent_id     = CASE WHEN sqlc.arg(set_parent)::boolean THEN sqlc.narg(parent_id)::uuid ELSE l.parent_id END,
    sort_order    = coalesce(sqlc.narg(sort_order)::double precision, l.sort_order),
    updated_by_id = sqlc.arg(updated_by)::uuid,
    updated_at    = sqlc.arg(now)
WHERE l.id = sqlc.arg(id) AND l.deleted_at IS NULL
RETURNING l.id, l.workspace_id, l.project_id, l.parent_id, l.name, l.color, l.sort_order, l.created_at, l.updated_at;

-- name: DeleteLabel :exec
-- deleteLabel, one statement under the project's FOR NO KEY UPDATE (M3 design 3.16, 3.6 convention 5): the undeleted
-- label and the undeleted labels under it, at the moment and by the account given. A label's children are of its
-- project (3.16), so that lock covers every row the statement writes.
UPDATE labels
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE (id = sqlc.arg(id) OR parent_id = sqlc.arg(id)) AND deleted_at IS NULL;
````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `c5f77e61421f78599790339ab493af7fd1649b962733f8e71b34098685994e3b` | 275 | `server/internal/modules/project/adapter/postgres/gen/labels.sql.go` |

Run: `shasum -a 256 server/internal/modules/project/adapter/postgres/gen/labels.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 存储方法和测试**

`server/internal/modules/project/app/label_ports.go`（新文件，23 行）：

````file server/internal/modules/project/app/label_ports.go
package app

import (
	"time"
	"uuid"
)

// The project store's labels, as the label operations read and write them
// (M3 design 3.16).

// LabelRow is a label to insert: its values as the use case decided them,
// its parent checked and its sort order given.
type LabelRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	ParentID    *uuid.UUID
	Name        string
	Color       string
	SortOrder   float64
	CreatedBy   uuid.UUID
	Now         time.Time
}
````

`server/internal/modules/project/adapter/postgres/labels.go`（新文件，120 行）：

````file server/internal/modules/project/adapter/postgres/labels.go
package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// labelOf is a label row as the domain has it. The queries that read a
// label each read the same columns, so their rows convert to
// gen.LabelByIDRow.
func labelOf(r gen.LabelByIDRow) domain.Label {
	return domain.Label{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, ParentID: r.ParentID, Name: r.Name, Color: r.Color,
		SortOrder: r.SortOrder, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

// labelTaken is err, a write's of a label, as a name another undeleted
// label of the project has, in any case: domain.ErrLabelNameTaken; any
// other error is the write's.
func labelTaken(write string, err error) error {
	if postgres.UniqueViolation(err, "labels_project_id_name_key") {
		return domain.ErrLabelNameTaken
	}
	return fmt.Errorf("%s: %w", write, err)
}

// CreateLabel inserts r and returns it as stored (app.LabelCreator). A
// name another undeleted label of the project has, in any case, is
// domain.ErrLabelNameTaken. The domain and the use case checked every
// value, so a CHECK or a foreign key's violation is a bug: an internal
// error, not a domain error.
func (s *Store) CreateLabel(ctx context.Context, r app.LabelRow) (domain.Label, error) {
	row, err := s.queries(ctx).CreateLabel(ctx, gen.CreateLabelParams{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID,
		ParentID: r.ParentID, Name: r.Name, Color: r.Color, SortOrder: r.SortOrder, CreatedBy: &r.CreatedBy, Now: r.Now})
	if err != nil {
		return domain.Label{}, labelTaken(fmt.Sprintf("create label %q", r.Name), err)
	}
	return labelOf(gen.LabelByIDRow(row)), nil
}

// LabelByID is the undeleted label id; found is false when there is none
// (app.LabelFinder).
func (s *Store) LabelByID(ctx context.Context, id uuid.UUID) (l domain.Label, found bool, err error) {
	r, err := s.queries(ctx).LabelByID(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Label{}, false, nil
	case err != nil:
		return domain.Label{}, false, fmt.Errorf("read label %s: %w", id, err)
	}
	return labelOf(r), true, nil
}

// ListLabels lists projectID's undeleted labels, by sort order, then id
// (app.LabelLister).
func (s *Store) ListLabels(ctx context.Context, projectID uuid.UUID) ([]domain.Label, error) {
	rows, err := s.queries(ctx).ListLabels(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list labels of project %s: %w", projectID, err)
	}
	out := make([]domain.Label, len(rows))
	for i, r := range rows {
		out[i] = labelOf(gen.LabelByIDRow(r))
	}
	return out, nil
}

// GreatestSortOrder is the greatest sort order of projectID's undeleted
// labels, nil when it has none (app.LabelCreator).
func (s *Store) GreatestSortOrder(ctx context.Context, projectID uuid.UUID) (*float64, error) {
	greatest, err := s.queries(ctx).GreatestSortOrder(ctx, projectID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("read the greatest sort order of project %s: %w", projectID, err)
	}
	return &greatest, nil
}

// HasChildren reports whether an undeleted label has the label id as its
// parent (app.LabelUpdater).
func (s *Store) HasChildren(ctx context.Context, id uuid.UUID) (bool, error) {
	has, err := s.queries(ctx).HasChildren(ctx, id)
	if err != nil {
		return false, fmt.Errorf("read the children of label %s: %w", id, err)
	}
	return has, nil
}

// UpdateLabel changes the fields p gives of the undeleted label id, by the
// account by at now, and returns it as stored (app.LabelUpdater). A name
// another undeleted label of the project has, in any case, is
// domain.ErrLabelNameTaken; a deleted label is an error, not written.
func (s *Store) UpdateLabel(ctx context.Context, id uuid.UUID, p domain.LabelPatch, by uuid.UUID, now time.Time) (domain.Label, error) {
	r, err := s.queries(ctx).UpdateLabel(ctx, gen.UpdateLabelParams{ID: id, Name: p.Name, Color: p.Color, SetParent: p.SetParent,
		ParentID: p.ParentID, SortOrder: p.SortOrder, UpdatedBy: by, Now: now})
	if err != nil {
		return domain.Label{}, labelTaken(fmt.Sprintf("update label %s", id), err)
	}
	return labelOf(gen.LabelByIDRow(r)), nil
}

// DeleteLabel deletes the undeleted label id and the undeleted labels
// under it, by the account by at now, in one statement (app.LabelDeleter).
func (s *Store) DeleteLabel(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).DeleteLabel(ctx, gen.DeleteLabelParams{ID: id, DeletedBy: by, Now: now}); err != nil {
		return fmt.Errorf("delete label %s: %w", id, err)
	}
	return nil
}
````

`server/internal/modules/project/adapter/postgres/states_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/postgres/states_test.go
// maker. alice writes.
````
````new server/internal/modules/project/adapter/postgres/states_test.go
// maker. alice writes. The label tests add their labels the same way
// (addLabel).
````

````old server/internal/modules/project/adapter/postgres/states_test.go
}

// addState stores a state of w's project, made by maker at earlier, with
````
````new server/internal/modules/project/adapter/postgres/states_test.go
}

// workspaceOf is the workspace of w's project: beta's for Site, acme's
// for the others.
func (w stateWorld) workspaceOf(project uuid.UUID) uuid.UUID {
	if project == w.site {
		return w.beta
	}
	return w.acme
}

// addState stores a state of w's project, made by maker at earlier, with
````

````old server/internal/modules/project/adapter/postgres/states_test.go
	workspace := w.acme
	if project == w.site {
		workspace = w.beta
	}
	got, err := w.s.CreateState(context.Background(), app.StateRow{ID: id, WorkspaceID: workspace, ProjectID: project, State: st,
````
````new server/internal/modules/project/adapter/postgres/states_test.go
	got, err := w.s.CreateState(context.Background(), app.StateRow{ID: id, WorkspaceID: w.workspaceOf(project), ProjectID: project, State: st,
````

`server/internal/modules/project/adapter/postgres/labels_test.go`（新文件，221 行）：

````file server/internal/modules/project/adapter/postgres/labels_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// addLabel stores a label of w's project, made by maker at earlier, with
// the id, name, parent and sort order given, and returns it as CreateLabel
// answers it.
func (w stateWorld) addLabel(t *testing.T, project, id uuid.UUID, name string, parent *uuid.UUID, sortOrder float64) domain.Label {
	t.Helper()
	got, err := w.s.CreateLabel(context.Background(), app.LabelRow{ID: id, WorkspaceID: w.workspaceOf(project), ProjectID: project, ParentID: parent,
		Name: name, Color: "#111", SortOrder: sortOrder, CreatedBy: w.maker, Now: earlier})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// labelNames are the names of labels, in order.
func labelNames(labels []domain.Label) []string {
	out := make([]string, len(labels))
	for i, l := range labels {
		out[i] = l.Name
	}
	return out
}

// CreateLabel stores the row with its values, its parent too, by the
// account and at the moment given, never deleted, and answers it as
// stored; every other label keeps every column.
func TestCreateLabel(t *testing.T) {
	w := newStateWorld(t)
	bug := w.addLabel(t, w.web, uuid.NewV7(), "Bug", nil, 65535)
	id := uuid.NewV7()
	others := tableRows(t, w.pool, "labels", id)

	got, err := w.s.CreateLabel(context.Background(), app.LabelRow{ID: id, WorkspaceID: w.acme, ProjectID: w.web, ParentID: &bug.ID, Name: "UI",
		Color: "#F59E0B", SortOrder: 75535, CreatedBy: w.alice, Now: now})

	want := domain.Label{ID: id, WorkspaceID: w.acme, ProjectID: w.web, ParentID: &bug.ID, Name: "UI", Color: "#F59E0B", SortOrder: 75535,
		CreatedAt: now, UpdatedAt: now}
	if err != nil || jsonOf(t, got) != jsonOf(t, want) {
		t.Errorf("CreateLabel() = %+v, %v; want %+v", got, err, want)
	}
	cols := columns(t, w.pool, "labels", id)
	if a := `"` + w.alice.String() + `"`; cols["created_by_id"] != a || cols["updated_by_id"] != a || cols["deleted_at"] != "null" ||
		cols["created_at"] != jsonTime(now) || cols["updated_at"] != jsonTime(now) || cols["parent_id"] != `"`+bug.ID.String()+`"` {
		t.Errorf("the stored row: %v; want it under Bug, made and written by alice at now, undeleted", cols)
	}
	if after := tableRows(t, w.pool, "labels", id); after != others {
		t.Errorf("the other labels:\n%s\nwant\n%s", after, others)
	}
}

// A name is taken by another undeleted label of the same project in any
// case: bug and BUG beside Bug are each project.label_name_taken, the
// first problem the API would answer, and store nothing; a deleted label's
// name and a name of another project's label are free.
func TestCreateLabelNameTaken(t *testing.T) {
	w := newStateWorld(t)
	w.addLabel(t, w.web, uuid.NewV7(), "Bug", nil, 1)
	old := w.addLabel(t, w.web, uuid.NewV7(), "Old", nil, 2)
	exec(t, w.pool, "UPDATE labels SET deleted_at = $2 WHERE id = $1", old.ID, earlier)
	w.addLabel(t, w.ops, uuid.NewV7(), "Feature", nil, 1)
	create := func(name string) error {
		_, err := w.s.CreateLabel(context.Background(), app.LabelRow{ID: uuid.NewV7(), WorkspaceID: w.acme, ProjectID: w.web, Name: name,
			SortOrder: 3, CreatedBy: w.alice, Now: now})
		return err
	}
	before := tableRows(t, w.pool, "labels")
	for _, taken := range []string{"bug", "BUG"} {
		var se *shared.Error
		if err := create(taken); !errors.As(err, &se) || !errors.Is(se, domain.ErrLabelNameTaken) {
			t.Errorf("%s: %v, want project.label_name_taken", taken, err)
		}
	}
	if after := tableRows(t, w.pool, "labels"); after != before {
		t.Errorf("the labels after the refusals:\n%s\nwant\n%s", after, before)
	}
	for _, free := range []string{"old", "Feature", "Bugs"} {
		if err := create(free); err != nil {
			t.Errorf("%s: %v, want it free", free, err)
		}
	}
}

// Only the name's unique key is a 409: another unique key (a duplicate
// id), a CHECK the domain should have kept (an empty name, a label its own
// parent) and a parent of none are each an internal error, never a domain
// error.
func TestCreateLabelBreakingAnotherConstraintIsInternal(t *testing.T) {
	w := newStateWorld(t)
	bug := w.addLabel(t, w.web, uuid.NewV7(), "Bug", nil, 1)
	own, none := uuid.NewV7(), uuid.NewV7()
	for _, tt := range []struct {
		name       string
		row        app.LabelRow
		constraint string
	}{
		{"a duplicate id", app.LabelRow{ID: bug.ID, Name: "Feature"}, "labels_pkey"},
		{"an empty name", app.LabelRow{ID: uuid.NewV7(), Name: ""}, "labels_name_check"},
		{"its own parent", app.LabelRow{ID: own, ParentID: &own, Name: "Feature"}, "labels_not_own_parent_check"},
		{"a parent of none", app.LabelRow{ID: uuid.NewV7(), ParentID: &none, Name: "Feature"}, "labels_parent_id_fkey"},
	} {
		tt.row.WorkspaceID, tt.row.ProjectID, tt.row.CreatedBy, tt.row.Now = w.acme, w.web, w.alice, now
		_, err := w.s.CreateLabel(context.Background(), tt.row)
		var se *shared.Error
		var pgErr *pgconn.PgError
		if errors.As(err, &se) || !errors.As(err, &pgErr) || pgErr.ConstraintName != tt.constraint {
			t.Errorf("%s: CreateLabel() = %v; want the violation of %s, not a domain error", tt.name, err, tt.constraint)
		}
	}
}

// LabelByID reads the undeleted label the id names, every column as
// CreateLabel answered it, its parent too: not another label, which a read
// of whichever row lies first would answer for one of the two asked about;
// not a deleted label, nor an id of none.
func TestLabelByID(t *testing.T) {
	w := newStateWorld(t)
	bug := w.addLabel(t, w.web, uuid.NewV7(), "Bug", nil, 1)
	ui := w.addLabel(t, w.web, uuid.NewV7(), "UI", &bug.ID, 2.5)
	old := w.addLabel(t, w.web, uuid.NewV7(), "Old", nil, 3)
	exec(t, w.pool, "UPDATE labels SET deleted_at = $2 WHERE id = $1", old.ID, earlier)
	for _, want := range []domain.Label{bug, ui} {
		if got, found, err := w.s.LabelByID(context.Background(), want.ID); err != nil || !found || jsonOf(t, got) != jsonOf(t, want) {
			t.Errorf("LabelByID(%s) = %+v, %v, %v; want %+v", want.Name, got, found, err, want)
		}
	}
	for name, id := range map[string]uuid.UUID{"the deleted Old": old.ID, "no label": uuid.NewV7()} {
		if got, found, err := w.s.LabelByID(context.Background(), id); err != nil || found || got.ID != (uuid.UUID{}) {
			t.Errorf("LabelByID() of %s = %+v, %v, %v; want none", name, got, found, err)
		}
	}
}

// ListLabels lists the project's undeleted labels, parents and children
// alike, by sort order, then id: Feature, added last at UI's sort order
// with an id below its, comes before it, which the order the rows lie in
// does not give; not the deleted Old, nor another project's labels. An
// archived project lists its labels as any other (M3 design 3.19).
func TestListLabels(t *testing.T) {
	w := newStateWorld(t)
	bug := w.addLabel(t, w.web, uuid.NewV7(), "Bug", nil, 3)
	w.addLabel(t, w.web, uuid.NewV7(), "UI", &bug.ID, 2)
	w.addLabel(t, w.web, w.early, "Feature", nil, 2)
	old := w.addLabel(t, w.web, uuid.NewV7(), "Old", nil, 1)
	exec(t, w.pool, "UPDATE labels SET deleted_at = $2 WHERE id = $1", old.ID, earlier)
	w.addLabel(t, w.ops, uuid.NewV7(), "Docs", nil, 1)
	var heap []string
	if err := w.pool.QueryRow(context.Background(), "SELECT array_agg(name ORDER BY ctid) FROM labels WHERE project_id = $1 AND sort_order = 2",
		w.web).Scan(&heap); err != nil || !slices.Equal(heap, []string{"UI", "Feature"}) {
		t.Fatalf("the rows of sort order 2 lie as %q, %v; the test needs UI first", heap, err)
	}

	got, err := w.s.ListLabels(context.Background(), w.web)
	if want := []string{"Feature", "UI", "Bug"}; err != nil || !slices.Equal(labelNames(got), want) {
		t.Errorf("ListLabels(Web) = %q, %v; want %q", labelNames(got), err, want)
	}
	if len(got) > 1 && (got[1].ParentID == nil || *got[1].ParentID != bug.ID) {
		t.Errorf("Web's UI listed as %+v; want it under Bug", got[1])
	}
	exec(t, w.pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", w.ops, now)
	if got, err := w.s.ListLabels(context.Background(), w.ops); err != nil || !slices.Equal(labelNames(got), []string{"Docs"}) {
		t.Errorf("ListLabels(Ops, archived) = %q, %v; want Docs", labelNames(got), err)
	}
	if got, err := w.s.ListLabels(context.Background(), w.site); err != nil || got == nil || len(got) != 0 {
		t.Errorf("ListLabels(Site) = %+v, %v; want an empty list", got, err)
	}
}

// GreatestSortOrder is the greatest sort order of the project's undeleted
// labels, a child's too: Web's UI, 75535, not a deleted label's 99999, nor
// Ops's 80000. A project without labels has none.
func TestGreatestSortOrder(t *testing.T) {
	w := newStateWorld(t)
	bug := w.addLabel(t, w.web, uuid.NewV7(), "Bug", nil, 65535)
	w.addLabel(t, w.web, uuid.NewV7(), "UI", &bug.ID, 75535)
	old := w.addLabel(t, w.web, uuid.NewV7(), "Old", nil, 99999)
	exec(t, w.pool, "UPDATE labels SET deleted_at = $2 WHERE id = $1", old.ID, earlier)
	w.addLabel(t, w.ops, uuid.NewV7(), "Docs", nil, 80000)

	if got, err := w.s.GreatestSortOrder(context.Background(), w.web); err != nil || got == nil || *got != 75535 {
		t.Errorf("GreatestSortOrder(Web) = %s, %v; want 75535", jsonOf(t, got), err)
	}
	if got, err := w.s.GreatestSortOrder(context.Background(), w.site); err != nil || got != nil {
		t.Errorf("GreatestSortOrder(Site) = %s, %v; want none", jsonOf(t, got), err)
	}
}

// HasChildren is whether an undeleted label is under the label: Bug has
// UI; UI has none; Feature's only child is deleted; an id of none has
// none.
func TestHasChildren(t *testing.T) {
	w := newStateWorld(t)
	bug := w.addLabel(t, w.web, uuid.NewV7(), "Bug", nil, 1)
	ui := w.addLabel(t, w.web, uuid.NewV7(), "UI", &bug.ID, 2)
	feature := w.addLabel(t, w.web, uuid.NewV7(), "Feature", nil, 3)
	gone := w.addLabel(t, w.web, uuid.NewV7(), "Gone", &feature.ID, 4)
	exec(t, w.pool, "UPDATE labels SET deleted_at = $2 WHERE id = $1", gone.ID, earlier)
	for _, tt := range []struct {
		name string
		id   uuid.UUID
		want bool
	}{{"Bug", bug.ID, true}, {"UI", ui.ID, false}, {"Feature", feature.ID, false}, {"no label", uuid.NewV7(), false}} {
		if got, err := w.s.HasChildren(context.Background(), tt.id); err != nil || got != tt.want {
			t.Errorf("HasChildren(%s) = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
}
````

`server/internal/modules/project/adapter/postgres/label_writes_test.go`（新文件，143 行）：

````file server/internal/modules/project/adapter/postgres/label_writes_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateLabel changes exactly the fields the patch gives of Web's UI, and
// the audit columns to the moment and the account given, and answers the
// label as stored; every other column keeps its value, and every other
// label every column. A parent given sets it, a parent given as none
// clears it, and a parent not given keeps it. A patch that gives nothing
// changes the audit columns alone: it comes after one that gave every
// field, so that the parent, the color and the sort order it keeps are
// neither none nor the columns' defaults.
func TestUpdateLabel(t *testing.T) {
	w := newStateWorld(t)
	bug := w.addLabel(t, w.web, uuid.NewV7(), "Bug", nil, 1)
	ui := w.addLabel(t, w.web, uuid.NewV7(), "UI", nil, 2).ID
	others := tableRows(t, w.pool, "labels", ui)
	for _, tt := range []struct {
		name   string
		patch  domain.LabelPatch
		change map[string]string
	}{
		{"every field", domain.LabelPatch{Name: ptr("Front"), Color: ptr("#123456"), SetParent: true, ParentID: &bug.ID, SortOrder: ptr(-2.5)},
			map[string]string{"name": `"Front"`, "color": `"#123456"`, "parent_id": `"` + bug.ID.String() + `"`, "sort_order": "-2.5"}},
		{"nothing", domain.LabelPatch{}, nil},
		{"the name alone, its parent kept", domain.LabelPatch{Name: ptr("UI")}, map[string]string{"name": `"UI"`}},
		{"its parent cleared", domain.LabelPatch{SetParent: true}, map[string]string{"parent_id": "null"}},
		{"the sort order alone", domain.LabelPatch{SortOrder: ptr(5.0)}, map[string]string{"sort_order": "5"}},
		{"an empty color", domain.LabelPatch{Color: ptr("")}, map[string]string{"color": `""`}},
	} {
		before := columns(t, w.pool, "labels", ui)
		got, err := w.s.UpdateLabel(context.Background(), ui, tt.patch, w.alice, now)
		want := changed(before, changed(audit(w.alice, now), tt.change))
		if after := columns(t, w.pool, "labels", ui); err != nil || !maps.Equal(after, want) {
			t.Errorf("%s: UpdateLabel() = %v, the row %v\nwant %v", tt.name, err, after, want)
		}
		if stored, _, _ := w.s.LabelByID(context.Background(), ui); jsonOf(t, got) != jsonOf(t, stored) {
			t.Errorf("%s: UpdateLabel() answered %+v; want the row as stored, %+v", tt.name, got, stored)
		}
	}
	if after := tableRows(t, w.pool, "labels", ui); after != others {
		t.Errorf("the other labels:\n%s\nwant\n%s", after, others)
	}
}

// A name another undeleted label of the project has, in any case, is
// project.label_name_taken, the first problem the API would answer, and
// changes nothing; a deleted label's name and another project's are free,
// as is the label's own in another case.
func TestUpdateLabelNameTaken(t *testing.T) {
	w := newStateWorld(t)
	ui := w.addLabel(t, w.web, uuid.NewV7(), "UI", nil, 1).ID
	w.addLabel(t, w.web, uuid.NewV7(), "Feature", nil, 2)
	old := w.addLabel(t, w.web, uuid.NewV7(), "Old", nil, 3)
	exec(t, w.pool, "UPDATE labels SET deleted_at = $2 WHERE id = $1", old.ID, earlier)
	w.addLabel(t, w.ops, uuid.NewV7(), "Docs", nil, 1)
	before := tableRows(t, w.pool, "labels")
	var se *shared.Error
	if _, err := w.s.UpdateLabel(context.Background(), ui, domain.LabelPatch{Name: ptr("FEATURE")}, w.alice, now); !errors.As(err, &se) ||
		!errors.Is(se, domain.ErrLabelNameTaken) {
		t.Errorf("FEATURE: %v, want project.label_name_taken", err)
	}
	if after := tableRows(t, w.pool, "labels"); after != before {
		t.Errorf("the labels after the refusal:\n%s\nwant\n%s", after, before)
	}
	for _, free := range []string{"ui", "Old", "Docs"} {
		if _, err := w.s.UpdateLabel(context.Background(), ui, domain.LabelPatch{Name: ptr(free)}, w.alice, now); err != nil {
			t.Errorf("%s: %v, want it free", free, err)
		}
	}
}

// UpdateLabel writes no deleted label: an internal error, no domain error
// (project.label_name_taken or any other), and every label keeps every
// column.
func TestUpdateLabelWritesNoDeletedLabel(t *testing.T) {
	w := newStateWorld(t)
	old := w.addLabel(t, w.web, uuid.NewV7(), "Old", nil, 1)
	exec(t, w.pool, "UPDATE labels SET deleted_at = $2 WHERE id = $1", old.ID, earlier)
	before := tableRows(t, w.pool, "labels")
	var se *shared.Error
	if got, err := w.s.UpdateLabel(context.Background(), old.ID, domain.LabelPatch{Name: ptr("Other")}, w.alice, now); err == nil ||
		errors.As(err, &se) || got.ID != (uuid.UUID{}) {
		t.Errorf("UpdateLabel() of the deleted Old = %+v, %v; want an internal error", got, err)
	}
	if after := tableRows(t, w.pool, "labels"); after != before {
		t.Errorf("the labels after the refusal:\n%s\nwant\n%s", after, before)
	}
}

// DeleteLabel deletes Web's Bug and UI under it, at the moment and by the
// account given, deleted_at and updated_at alike, and touches no other
// column: Gone, under Bug and deleted before, keeps its moment; Feature,
// Feature's Docs, and Ops's Bug keep every column. Deleted again, Bug
// changes nothing. Deleting a child, Docs, deletes it alone.
func TestDeleteLabel(t *testing.T) {
	w := newStateWorld(t)
	bug := w.addLabel(t, w.web, uuid.NewV7(), "Bug", nil, 1)
	ui := w.addLabel(t, w.web, uuid.NewV7(), "UI", &bug.ID, 2)
	gone := w.addLabel(t, w.web, uuid.NewV7(), "Gone", &bug.ID, 3)
	exec(t, w.pool, "UPDATE labels SET deleted_at = $2 WHERE id = $1", gone.ID, earlier)
	feature := w.addLabel(t, w.web, uuid.NewV7(), "Feature", nil, 4)
	docs := w.addLabel(t, w.web, uuid.NewV7(), "Docs", &feature.ID, 5)
	w.addLabel(t, w.ops, uuid.NewV7(), "Bug", nil, 1)
	others := tableRows(t, w.pool, "labels", bug.ID, ui.ID)
	before := map[uuid.UUID]map[string]string{bug.ID: columns(t, w.pool, "labels", bug.ID), ui.ID: columns(t, w.pool, "labels", ui.ID)}

	if err := w.s.DeleteLabel(context.Background(), bug.ID, w.alice, now); err != nil {
		t.Errorf("DeleteLabel(Bug) = %v", err)
	}
	for id, cols := range before {
		if got, want := columns(t, w.pool, "labels", id), changed(cols, changed(audit(w.alice, now),
			map[string]string{"deleted_at": jsonTime(now)})); !maps.Equal(got, want) {
			t.Errorf("%s: %v\nwant %v", id, got, want)
		}
	}
	if after := tableRows(t, w.pool, "labels", bug.ID, ui.ID); after != others {
		t.Errorf("the other labels:\n%s\nwant\n%s", after, others)
	}
	all := tableRows(t, w.pool, "labels")
	if err := w.s.DeleteLabel(context.Background(), bug.ID, w.alice, now.Add(1)); err != nil {
		t.Errorf("DeleteLabel(Bug) again = %v", err)
	}
	if after := tableRows(t, w.pool, "labels"); after != all {
		t.Errorf("the labels after Bug deleted again:\n%s\nwant\n%s", after, all)
	}
	all = tableRows(t, w.pool, "labels", docs.ID)
	if err := w.s.DeleteLabel(context.Background(), docs.ID, w.alice, now); err != nil {
		t.Errorf("DeleteLabel(Docs) = %v", err)
	}
	if after := tableRows(t, w.pool, "labels", docs.ID); after != all || columns(t, w.pool, "labels", docs.ID)["deleted_at"] != jsonTime(now) {
		t.Errorf("the labels after Docs deleted:\n%s\nwant Docs deleted and the others\n%s", after, all)
	}
}
````

`server/internal/modules/project/adapter/postgres/failures_test.go`（修改，7 处）：

````old server/internal/modules/project/adapter/postgres/failures_test.go
	"github.com/jackc/pgx/v5/pgconn"

````
````new server/internal/modules/project/adapter/postgres/failures_test.go
	"github.com/jackc/pgx/v5/pgconn"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
````

````old server/internal/modules/project/adapter/postgres/failures_test.go
// project.state_last_in_group. Each read runs on a cancelled context
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
// project.state_last_in_group; not "no such label", which a write on it
// would answer as project.label_not_found, nor no labels, which listLabels
// would answer as none, nor "no sort order", which createLabel would take
// for the first label's, nor "no children", which updateLabel would take
// for a label free to take a parent. Each read runs on a cancelled context
````

````old server/internal/modules/project/adapter/postgres/failures_test.go
// has display settings in, with its six default states, bob's membership
// of another project ended, so that the right answer is none of the zero
// values.
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
// has display settings in, with its six default states and a label with a
// child, bob's membership of another project ended, so that the right
// answer is none of the zero values.
````

````old server/internal/modules/project/adapter/postgres/failures_test.go
	}
	todo := seedDefaultStates(t, s, acme, web, alice)["Todo"]
	cancelled, cancel := context.WithCancel(ctx)
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
	}
	todo := seedDefaultStates(t, s, acme, web, alice)["Todo"]
	bug := seedLabel(t, s, acme, web, nil, "Bug", alice)
	seedLabel(t, s, acme, web, &bug, "UI", alice)
	cancelled, cancel := context.WithCancel(ctx)
````

````old server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("ListWorkspaceStates() = %+v, %v; want context.Canceled, not none", list, err)
	}
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("ListWorkspaceStates() = %+v, %v; want context.Canceled, not none", list, err)
	}
	if l, found, err := s.LabelByID(cancelled, bug); !failed(err) || found || l.ID != (uuid.UUID{}) {
		t.Errorf("LabelByID() = %+v, %v, %v; want context.Canceled, not no label", l, found, err)
	}
	if list, err := s.ListLabels(cancelled, web); !failed(err) || list != nil {
		t.Errorf("ListLabels() = %+v, %v; want context.Canceled, not none", list, err)
	}
	if greatest, err := s.GreatestSortOrder(cancelled, web); !failed(err) || greatest != nil {
		t.Errorf("GreatestSortOrder() = %s, %v; want context.Canceled, not none", jsonOf(t, greatest), err)
	}
	if has, err := s.HasChildren(cancelled, bug); !failed(err) || has {
		t.Errorf("HasChildren() = %v, %v; want context.Canceled, not an answer", has, err)
	}
}

// seedLabel stores a label of project, under parent, made by by at now, and
// returns its id.
func seedLabel(t *testing.T, s *postgresadapter.Store, workspace, project uuid.UUID, parent *uuid.UUID, name string, by uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	if _, err := s.CreateLabel(context.Background(), app.LabelRow{ID: id, WorkspaceID: workspace, ProjectID: project, ParentID: parent,
		Name: name, SortOrder: 1, CreatedBy: by, Now: now}); err != nil {
		t.Fatal(err)
	}
	return id
````

````old server/internal/modules/project/adapter/postgres/failures_test.go
// project's or a state's taken name or identifier; a deletion or a
// marking of a state, never "not written", which the use case would
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
// project's, a state's or a label's taken name or identifier; a deletion
// or a marking of a state, never "not written", which the use case would
````

````old server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("MarkDefaultState() = %v, %v; want context.Canceled", marked, err)
	}
````
````new server/internal/modules/project/adapter/postgres/failures_test.go
		t.Errorf("MarkDefaultState() = %v, %v; want context.Canceled", marked, err)
	}
	bug := seedLabel(t, s, acme, web, nil, "Bug", alice)
	if l, err := s.CreateLabel(cancelled, app.LabelRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, ParentID: &bug, Name: "UI",
		SortOrder: 2, CreatedBy: alice, Now: now}); !failed(err) || l.ID != (uuid.UUID{}) {
		t.Errorf("CreateLabel() = %+v, %v; want context.Canceled", l, err)
	}
	if l, err := s.UpdateLabel(cancelled, bug, domain.LabelPatch{Name: ptr("Feature")}, alice, now); !failed(err) || l.ID != (uuid.UUID{}) {
		t.Errorf("UpdateLabel() = %+v, %v; want context.Canceled", l, err)
	}
	if err := s.DeleteLabel(cancelled, bug, alice, now); !failed(err) {
		t.Errorf("DeleteLabel() = %v; want context.Canceled", err)
	}
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/project/adapter/postgres/failures_test.go server/internal/modules/project/adapter/postgres/label_writes_test.go server/internal/modules/project/adapter/postgres/labels.go server/internal/modules/project/adapter/postgres/labels_test.go server/internal/modules/project/adapter/postgres/queries/labels.sql server/internal/modules/project/adapter/postgres/states_test.go server/internal/modules/project/app/label_ports.go server/internal/modules/project/adapter/postgres/gen/labels.sql.go
```
```bash
git commit -m "feat(M3/P7b): the project store reads, creates, changes and deletes a project's labels

Seven statements on labels: a label created and answered as stored;
one read by its id; a project's labels by sort order, then id, parents
and children alike; the greatest sort order of its labels; whether a
label has labels under it; a change of the fields given, the parent
set, cleared or kept; and a label deleted with the labels under it, in
one statement. A name another undeleted label of the project has, in
any case, is project.label_name_taken, by the unique key's exact name;
any other violation is an internal error.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p7b.py` 的编号；"层"是它被发现的每一层，端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `s1-lb-id` | `LabelByID` 去掉 `id`（读到任意一个标签） | `TestLabelByID`、`TestUpdateLabel`（Task 6 起）、`TestPermissionMatrix`（Task 5 起）、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 5 起）、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 7 起）、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`（Task 10 起） 等 7 个、P7（Task 11 起） | 存储；组合；端到端 |
| `s1-lb-deleted` | `LabelByID` 去掉 `deleted_at IS NULL` | `TestLabelByID`、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`（Task 10 起）、`TestLabelWritesOnOneProjectSerialize`（Task 10 起）、P7（Task 11 起） | 存储；组合；端到端 |
| `s1-ll-project` | `ListLabels` 去掉 `project_id`（列出每个项目的标签） | `TestListLabels`（Task 9 起）、`TestPermissionMatrix`（Task 9 起）、P7（Task 11 起） | 存储；组合；端到端 |
| `s1-ll-deleted` | `ListLabels` 去掉 `deleted_at IS NULL` | `TestListLabels`（Task 9 起）、P7（Task 11 起） | 存储；端到端 |
| `s1-ll-order` | `ListLabels` 只按 id 排 | `TestListLabels`（Task 9 起）、P7（Task 11 起） | 存储；端到端 |
| `s1-gs-project` | `GreatestSortOrder` 去掉 `project_id` | `TestGreatestSortOrder`、P7（Task 11 起） | 存储；端到端 |
| `s1-gs-deleted` | `GreatestSortOrder` 去掉 `deleted_at IS NULL` | `TestGreatestSortOrder`、P7（Task 11 起） | 存储；端到端 |
| `s1-gs-asc` | `GreatestSortOrder` 取最小的 | `TestGreatestSortOrder`、`TestPermissionMatrix`（Task 5 起）、P7（Task 11 起） | 存储；组合；端到端 |
| `s1-hc-parent` | `HasChildren` 去掉 `parent_id`（有任何子标签就答有） | `TestHasChildren`、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 7 起）、`TestEachLabelWriteChangesItsRowsAlone`（Task 10 起）、`TestLabelWritesOnOneProjectSerialize`（Task 10 起）、P7（Task 11 起） | 存储；组合；端到端 |
| `s1-hc-deleted` | `HasChildren` 去掉 `deleted_at IS NULL` | `TestHasChildren`、`TestEachLabelWriteChangesItsRowsAlone`（Task 10 起） | 存储；组合 |
| `s1-ul-id` | `UpdateLabel` 去掉 `id`（写每个标签） | `TestUpdateLabelNameTaken`、`TestUpdateLabel`（Task 6 起）、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 7 起）、`TestPermissionMatrix`（Task 7 起）、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 7 起）、`TestEachLabelWriteChangesItsRowsAlone`（Task 10 起） 等 6 个、P7（Task 11 起） | 存储；组合；端到端 |
| `s1-ul-deleted` | `UpdateLabel` 去掉 `deleted_at IS NULL` | `TestUpdateLabelWritesNoDeletedLabel` | 存储 |
| `s1-dl-id` | `DeleteLabel` 去掉 `id` 和 `parent_id`（删除每个标签） | `TestDeleteLabel`（Task 8 起）、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 8 起）、`TestEachLabelWriteChangesItsRowsAlone`（Task 10 起）、`TestLabelWritesOnOneProjectSerialize`（Task 10 起）、P7（Task 11 起） | 存储；组合；端到端 |
| `s1-dl-children` | `DeleteLabel` 去掉 `parent_id`（下面的标签留下） | `TestDeleteLabel`（Task 8 起）、`TestTheWritesOnAProjectStampTheirRequest`（Task 8 起）、`TestLabelWritesOnOneProjectSerialize`（Task 10 起）、P7（Task 11 起） | 存储；组合；端到端 |
| `s1-dl-deleted` | `DeleteLabel` 去掉 `deleted_at IS NULL`（已删除的再删一次） | `TestDeleteLabel`（Task 8 起）、`TestEachLabelWriteChangesItsRowsAlone`（Task 10 起） | 存储；组合 |
| `s3-ul-created` | `UpdateLabel` 改写 `created_at` | `TestUpdateLabel`（Task 6 起） | 存储 |
| `s14-ul-keeps-writer` | `UpdateLabel` 保留原来的 `updated_by_id` | `TestUpdateLabel`（Task 6 起）、`TestTheWritesOnAProjectStampTheirRequest`（Task 7 起）、P7（Task 11 起） | 存储；组合；端到端 |
| `s3-ul-keeps-moment` | `UpdateLabel` 保留原来的 `updated_at` | `TestUpdateLabel`（Task 6 起）、`TestTheWritesOnAProjectStampTheirRequest`（Task 7 起）、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 10 起）、`TestLabelWritesOnOneProjectSerialize`（Task 10 起） | 存储；组合 |
| `s26-ul-name-kept` | `UpdateLabel` 从不写给的名称（`coalesce` 的次序反了） | `TestUpdateLabelNameTaken`、`TestUpdateLabel`（Task 6 起）、`TestPermissionMatrix`（Task 7 起）、`TestTheWritesOnAProjectStampTheirRequest`（Task 7 起）、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 8 起）、P7（Task 11 起） | 存储；组合；端到端 |
| `s26-ul-color-empty` | `UpdateLabel` 把没给的颜色写成空 | `TestUpdateLabel`（Task 6 起）、P7（Task 11 起） | 存储；端到端 |
| `s26-ul-sort-zero` | `UpdateLabel` 把没给的 `sort_order` 写成 0 | `TestUpdateLabel`（Task 6 起）、`TestPermissionMatrix`（Task 7 起）、P7（Task 11 起） | 存储；组合；端到端 |
| `s26-ul-parent-top` | `UpdateLabel` 在没给父标签时移到顶层 | `TestUpdateLabel`（Task 6 起）、P7（Task 11 起） | 存储；端到端 |
| `s26-ul-null-kept` | `UpdateLabel` 把给成 null 的父标签当作没给 | `TestUpdateLabel`（Task 6 起）、P7（Task 11 起） | 存储；端到端 |
| `s14-dl-keeps-writer` | `DeleteLabel` 保留原来的 `updated_by_id` | `TestDeleteLabel`（Task 8 起）、`TestTheWritesOnAProjectStampTheirRequest`（Task 8 起）、P7（Task 11 起） | 存储；组合；端到端 |
| `s3-dl-keeps-moment` | `DeleteLabel` 保留原来的 `updated_at` | `TestDeleteLabel`（Task 8 起）、`TestTheWritesOnAProjectStampTheirRequest`（Task 8 起）、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 10 起）、`TestLabelWritesOnOneProjectSerialize`（Task 10 起）、P7（Task 11 起） | 存储；组合；端到端 |
| `s26-cr-sort` | `Store.CreateLabel` 写 65535，不写给的 `sort_order` | `TestGreatestSortOrder`、`TestCreateLabel`（Task 4 起）、`TestListLabels`（Task 9 起）、`TestListingProjectsIsReadingEach`（Task 5 起）、`TestListingWorkspaceStatesIsListingEachProjects`（Task 5 起）、`TestPermissionMatrix`（Task 5 起）、`TestTheInvitationLinkAnswersEveryCallerAlike`（Task 5 起）、P7（Task 11 起） | 存储；组合；端到端 |
| `s26-cr-parent` | `Store.CreateLabel` 丢掉父标签 | `TestCreateLabelBreakingAnotherConstraintIsInternal`、`TestHasChildren`、`TestCreateLabel`（Task 4 起）、`TestDeleteLabel`（Task 8 起） 等 5 个、`TestListingProjectsIsReadingEach`（Task 5 起）、`TestListingWorkspaceStatesIsListingEachProjects`（Task 5 起）、`TestPermissionMatrix`（Task 5 起）、`TestTheInvitationLinkAnswersEveryCallerAlike`（Task 5 起） 等 9 个、P7（Task 11 起） | 存储；组合；端到端 |
| `s26-cr-color` | `Store.CreateLabel` 丢掉颜色 | `TestCreateLabel`（Task 4 起）、`TestPermissionMatrix`（Task 5 起）、P7（Task 11 起） | 存储；组合；端到端 |
| `s8-create` | `Store.CreateLabel` 在连接池上执行，不在事务的连接上 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 5 起） | 组合 |
| `s8-byid` | `Store.LabelByID` 在连接池上执行 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 5 起） | 组合 |
| `s8-greatest` | `Store.GreatestSortOrder` 在连接池上执行 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 5 起） | 组合 |
| `s8-children` | `Store.HasChildren` 在连接池上执行 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 7 起） | 组合 |
| `s8-update` | `Store.UpdateLabel` 在连接池上执行 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 7 起） | 组合 |
| `s8-delete` | `Store.DeleteLabel` 在连接池上执行 | `TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 8 起） | 组合 |
| `s19-create` | `Store.CreateLabel` 的失败答成成功 | `TestAFailedWriteIsAnError`、`TestCreateLabelBreakingAnotherConstraintIsInternal`、`TestCreateLabelNameTaken` | 存储 |
| `s19-byid` | `Store.LabelByID` 的失败答成"没有" | `TestAFailedReadIsAnErrorNotAnAnswer` | 存储 |
| `s19-list` | `Store.ListLabels` 的失败答成空列表 | `TestAFailedReadIsAnErrorNotAnAnswer` | 存储 |
| `s19-greatest` | `Store.GreatestSortOrder` 的失败答成"没有标签" | `TestAFailedReadIsAnErrorNotAnAnswer` | 存储 |
| `s19-children` | `Store.HasChildren` 的失败答成"没有子标签" | `TestAFailedReadIsAnErrorNotAnAnswer` | 存储 |
| `s19-update` | `Store.UpdateLabel` 的失败答成成功 | `TestAFailedWriteIsAnError`、`TestUpdateLabelNameTaken`、`TestUpdateLabelWritesNoDeletedLabel` | 存储 |
| `s19-delete` | `Store.DeleteLabel` 的失败答成成功 | `TestAFailedWriteIsAnError` | 存储 |
| `s19-name-taken-internal` | 名称被占用答成内部错误（`labelTaken` 不认约束名） | `TestCreateLabelNameTaken`、`TestUpdateLabelNameTaken`、`TestPermissionMatrix`（Task 5 起）、`TestEachLabelWriteChangesItsRowsAlone`（Task 10 起）、`TestLabelWritesOnOneProjectSerialize`（Task 10 起）、P7（Task 11 起） | 存储；组合；端到端 |
| `s19-any-key-taken` | 重复的 id 也答成名称被占用（`labels_pkey`） | `TestCreateLabelBreakingAnotherConstraintIsInternal` | 存储 |
| `s21-taken-wrapped` | 名称被占用包在 `project.label_not_found` 之后 | `TestCreateLabelNameTaken`、`TestUpdateLabelNameTaken`、`TestPermissionMatrix`（Task 5 起）、`TestEachLabelWriteChangesItsRowsAlone`（Task 10 起）、`TestLabelWritesOnOneProjectSerialize`（Task 10 起）、P7（Task 11 起） | 存储；组合；端到端 |
| `s32-wait-on-label` | 标签的写在第一次读时锁标签的行（`FOR UPDATE`，在工作区之前） | `TestEachWriteOnAProjectSharesItsWorkspaceFirst`（Task 7 起）、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`（Task 10 起）、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 10 起）、`TestLabelWritesOnOneProjectSerialize`（Task 10 起） | 组合 |

---

### Task 4: `createLabel` 的用例：锁、判定、父标签的检查、`sort_order`

**Files:**
- Create: `server/internal/modules/project/app/create_label.go`、`server/internal/modules/project/app/create_label_test.go`、`server/internal/modules/project/app/fakes_label_test.go`、`server/internal/modules/project/app/label_parent.go`
- Modify: `server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/app/clock_test.go`、`server/internal/modules/project/app/fakes_state_test.go`、`server/internal/modules/project/app/fakes_write_test.go`、`server/internal/modules/project/app/label_ports.go`、`server/internal/modules/project/domain/actions.go`

**Interfaces:**
- Consumes：Task 2 的 `domain.CheckNewLabel`、`SortOrderAfter`、`CheckParent`；Task 3 的存储方法；P7a 的 `Locks.lockAndDecide`、`write{project, action}`、`writeFixture`、`outcome`。
- Produces（spec 2.6；M3 设计 3.6、3.16、3.19）：
  - `label_ports.go` 加 `LabelFinder`（`LabelByID`：一个写指名的标签，先不加锁读出它的项目和工作区、再在锁下读，以及一个写指名的父标签，在项目的锁下读）和 `LabelCreator`（`LabelFinder`、`GreatestSortOrder`、`CreateLabel`）。
  - `label_parent.go`（新）：`checkParent(ctx, labels, id, project, parentID, hasChildren) error`：在项目的 `FOR NO KEY UPDATE` 下读父标签（每个标签的写都拿这把锁，读到的标签保持到提交），读到的不是问的 id 是错误；再 `domain.CheckParent`。
  - `create_label.go`（新）：`NewCreateLabel(locks, labels LabelCreator, tx, clock)`、`Execute(ctx, projectID, domain.LabelCreate) (domain.Label, error)`：调用者 → `CheckNewLabel`（事务之前）→ 一个事务：`lockAndDecide`（工作区 `FOR SHARE`、项目 `FOR NO KEY UPDATE`、`label.create` 的判定；已归档的项目照常，3.19）→ 给了父标签时 `checkParent(…, uuid.Nil(), projectID, parent, false)` → `sort_order`：给的，否则 `SortOrderAfter(GreatestSortOrder)` → 时钟 → `CreateLabel`（由调用者、在那个时刻）；回答的 id 不是插入的是错误。
  - `label.create`：项目级、只有管理员（同 `state.create`，M3 设计 9.2）；`domain.ActionLabelCreate`、`Actions()` 加它。
  - `fakes_label_test.go`（新）：`fakeLabels`（Web 的 Bug、它下面的 UI、Feature，已归档的 Ops 的 Docs；`answersAs`、`changedAs`、`failsFor`、`errs`，调用记在 `writeFixture` 的日志里）、`newCreateLabel`、`labelCreated`；`fakes_write_test.go` 的 `writeFixture` 接上它。

**Tests:**
- `TestCreateLabel`：Web 的顶层（95535：Feature 85535 之后）、Bug 下面（同）、给的 `sort_order`（-1.5，不读最大值）、已归档的 Ops（75535）、没有标签的 Ops（65535）；每一行核对调用的次序（锁、判定、父标签、最大值、时钟、插入）和存下的每一列。
- `TestCreateLabelRefuses`（14 行）：没有调用者、空白的名称（事务之前，什么都不调用）；没有项目、工作区或项目在锁等待期间删除、项目移到别的工作区、看不见 Web 的调用者，各 `project.not_found`；成员 403（判定在锁之后）；父标签不是标签、是 Ops 的 Docs、是 Bug 下面的 UI，各 422 `parent_id not_allowed`（在判定之后，`lockedDecision` 之后读父标签）；父标签读成另一个 id、插入回答另一个 id 是这个写自己的错误；Bug 的名称换大小写是存储的 409。每一行之后标签不变。
- `TestCreateLabelReturnsEachFailure`：每个端口的失败（项目的工作区、两把锁、判定、父标签、最大值、插入、提交）原样返回，之前的调用都在、之后的都不在。
- `TestEachWriteReadsTheClockUnderItsLock` 加 `createLabel` 一行（时钟在父标签和最大值之后、插入之前）；`TestEveryRuleDecidesItsCells` 加 `label.create` 的 17 格；`TestAnActionWithoutARowHasNoRule` 照旧通过（`Actions()` 的每个动作都有规则）。

- [ ] **Step 1: 规则和动作**

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"state.mark_default": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````
````new server/internal/modules/access/domain/rules.go
	"state.mark_default": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	// As state.create (M3 design 3.4, 9.2).
	"label.create": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"state.mark_default": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible,
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"state.mark_default": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible,
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
	// As state.create.
	"label.create": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````

`server/internal/modules/project/domain/actions.go`（修改，2 处）：

````old server/internal/modules/project/domain/actions.go
	ActionWorkspaceStateList shared.Action = "workspace_state.list"
````
````new server/internal/modules/project/domain/actions.go
	ActionWorkspaceStateList shared.Action = "workspace_state.list"
	// ActionLabelCreate is creating a label in a project: createLabel.
	ActionLabelCreate shared.Action = "label.create"
````

````old server/internal/modules/project/domain/actions.go
		ActionStateList, ActionStateCreate, ActionStateUpdate, ActionStateDelete, ActionStateMarkDefault, ActionWorkspaceStateList}
````
````new server/internal/modules/project/domain/actions.go
		ActionStateList, ActionStateCreate, ActionStateUpdate, ActionStateDelete, ActionStateMarkDefault, ActionWorkspaceStateList,
		ActionLabelCreate}
````

- [ ] **Step 2: 端口、父标签的检查、用例**

`server/internal/modules/project/app/label_ports.go`（修改，3 处）：

````old server/internal/modules/project/app/label_ports.go
import (
````
````new server/internal/modules/project/app/label_ports.go
import (
	"context"
````

````old server/internal/modules/project/app/label_ports.go
	"uuid"
````
````new server/internal/modules/project/app/label_ports.go
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
````

````old server/internal/modules/project/app/label_ports.go
}

````
````new server/internal/modules/project/app/label_ports.go
}

// LabelFinder reads a label by its id: a label a write names as the
// parent, under the project's lock (checkParent).
type LabelFinder interface {
	// LabelByID is the undeleted label id; found is false when there is
	// none.
	LabelByID(ctx context.Context, id uuid.UUID) (l domain.Label, found bool, err error)
}

// LabelCreator is createLabel's repository. Each method runs in the
// transaction ctx carries, under the project's FOR NO KEY UPDATE, which
// every write of its labels takes, so that its labels stay as read (M3
// design 3.16).
type LabelCreator interface {
	LabelFinder
	// GreatestSortOrder is the greatest sort order of projectID's undeleted
	// labels, nil when it has none.
	GreatestSortOrder(ctx context.Context, projectID uuid.UUID) (*float64, error)
	// CreateLabel inserts r and answers it as stored. A name another
	// undeleted label of the project has, in any case, is
	// domain.ErrLabelNameTaken.
	CreateLabel(ctx context.Context, r LabelRow) (domain.Label, error)
}

````

`server/internal/modules/project/app/label_parent.go`（新文件，30 行）：

````file server/internal/modules/project/app/label_parent.go
package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// checkParent reads the label parentID, the parent a write asks for the
// label id of project (uuid.Nil for a new label), under the project's FOR
// NO KEY UPDATE, which every write of the project's labels takes, so that
// the labels stay as read until the commit (M3 design 3.16); and checks it
// (domain.CheckParent), hasChildren saying whether the label has labels
// under it. A parent read for another id than asked is an error.
func checkParent(ctx context.Context, labels LabelFinder, id, project, parentID uuid.UUID, hasChildren bool) error {
	parent, found, err := labels.LabelByID(ctx, parentID)
	if err != nil {
		return err
	}
	var read *domain.Label
	if found {
		if parent.ID != parentID {
			return fmt.Errorf("parent label %s read as %s", parentID, parent.ID)
		}
		read = &parent
	}
	return domain.CheckParent(id, project, read, hasChildren)
}
````

`server/internal/modules/project/app/create_label.go`（新文件，84 行）：

````file server/internal/modules/project/app/create_label.go
package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateLabel creates a label in a project: POST
// /api/v0/projects/{project_id}/labels (M3 design 3.16).
type CreateLabel struct {
	locks  Locks
	labels LabelCreator
	tx     shared.TxManager
	clock  Clock
}

// NewCreateLabel returns the use case.
func NewCreateLabel(locks Locks, labels LabelCreator, tx shared.TxManager, clock Clock) *CreateLabel {
	return &CreateLabel{locks: locks, labels: labels, tx: tx, clock: clock}
}

// Execute checks in (domain.CheckNewLabel), then in one transaction, in the
// order of M3 design 3.6: the project's locks (its workspace FOR SHARE, the
// project FOR NO KEY UPDATE, which every write of its labels takes) and the
// decision on label.create; an archived project's labels are created as
// any other's (3.19). Then, under the locks, the parent given
// (checkParent); the sort order given, or one after the greatest of the
// project's labels (domain.SortOrderAfter); the clock; and the label, by
// the caller at that time. The answer is the label as stored.
func (u *CreateLabel) Execute(ctx context.Context, projectID uuid.UUID, in domain.LabelCreate) (domain.Label, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Label{}, err
	}
	if err := domain.CheckNewLabel(in); err != nil {
		return domain.Label{}, err
	}
	var created domain.Label
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := u.locks.lockAndDecide(ctx, actor, write{project: projectID, action: domain.ActionLabelCreate})
		if err != nil {
			return err
		}
		if in.ParentID != nil {
			if err := checkParent(ctx, u.labels, uuid.Nil(), projectID, *in.ParentID, false); err != nil {
				return err
			}
		}
		sortOrder, err := u.sortOrder(ctx, projectID, in.SortOrder)
		if err != nil {
			return err
		}
		row := LabelRow{ID: uuid.NewV7(), WorkspaceID: h.project.WorkspaceID, ProjectID: projectID, ParentID: in.ParentID, Name: in.Name,
			Color: in.Color, SortOrder: sortOrder, CreatedBy: actor.UserID, Now: u.clock.Now()}
		if created, err = u.labels.CreateLabel(ctx, row); err != nil {
			return err
		}
		if created.ID != row.ID {
			return fmt.Errorf("label %s created as %s", row.ID, created.ID)
		}
		return nil
	})
	if err != nil {
		return domain.Label{}, err
	}
	return created, nil
}

// sortOrder is a new label's sort order: the one given, or one after the
// greatest of projectID's labels.
func (u *CreateLabel) sortOrder(ctx context.Context, projectID uuid.UUID, given *float64) (float64, error) {
	if given != nil {
		return *given, nil
	}
	greatest, err := u.labels.GreatestSortOrder(ctx, projectID)
	if err != nil {
		return 0, err
	}
	return domain.SortOrderAfter(greatest), nil
}
````

- [ ] **Step 3: 假实现和测试**

`server/internal/modules/project/app/fakes_label_test.go`（新文件，151 行）：

````file server/internal/modules/project/app/fakes_label_test.go
package app_test

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// The labels of web and ops, by id, the same in every fixture: made in
// this order, so their ids are too.
var webBug, webUI, webFeature, opsDocs = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()

// fakeLabels is the project store's labels as the label operations' tests
// hold them, by id, beside fakeStore's projects, whose log, failures,
// reads of a row by its id and changedAs it shares.
type fakeLabels struct {
	*fakeStore
	labels map[uuid.UUID]domain.Label
}

// newLabels is newWrites with web's labels, Bug at the top with UI under
// it and Feature at the top, in this order of sort order; and ops's Docs at
// the top. Each was made when the fakes' states were (stateSince).
func newLabels() (*writeFixture, *fakeLabels) {
	f := newWrites()
	l := &fakeLabels{fakeStore: f.store, labels: map[uuid.UUID]domain.Label{}}
	for _, label := range []domain.Label{
		{ID: webBug, ProjectID: webID, Name: "Bug", SortOrder: 65535},
		{ID: webUI, ProjectID: webID, ParentID: &webBug, Name: "UI", SortOrder: 75535},
		{ID: webFeature, ProjectID: webID, Name: "Feature", SortOrder: 85535},
		{ID: opsDocs, ProjectID: opsID, Name: "Docs", SortOrder: 65535},
	} {
		label.WorkspaceID, label.Color, label.CreatedAt, label.UpdatedAt = acme.ID, "#111111", stateSince, stateSince
		l.labels[label.ID] = label
	}
	return f, l
}

// of are the labels of project, in the store's order: by name, which
// neither a sort by sort order, either way, nor one by id gives, so that a
// use case that sorted them would answer another.
func (f *fakeLabels) of(project uuid.UUID) []domain.Label {
	var out []domain.Label
	for _, l := range f.labels {
		if l.ProjectID == project {
			out = append(out, l)
		}
	}
	slices.SortFunc(out, func(a, b domain.Label) int { return cmp.Compare(a.Name, b.Name) })
	return out
}

func (f *fakeLabels) GreatestSortOrder(ctx context.Context, projectID uuid.UUID) (*float64, error) {
	f.log.add(ctx, "GreatestSortOrder %s", projectID)
	if err := f.fail("GreatestSortOrder"); err != nil {
		return nil, err
	}
	var greatest *float64
	for _, l := range f.of(projectID) {
		if greatest == nil || l.SortOrder > *greatest {
			greatest = &l.SortOrder
		}
	}
	return greatest, nil
}

// CreateLabel stores r's label, as stored, unless its project has a label
// of its name in any case: domain.ErrLabelNameTaken.
func (f *fakeLabels) CreateLabel(ctx context.Context, r app.LabelRow) (domain.Label, error) {
	f.log.add(ctx, "CreateLabel %s", labelRow(r))
	if err := f.fail("CreateLabel"); err != nil {
		return domain.Label{}, err
	}
	if f.taken(r.ProjectID, uuid.Nil(), r.Name) {
		return domain.Label{}, domain.ErrLabelNameTaken
	}
	at := r.Now.Truncate(time.Microsecond)
	l := domain.Label{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, ParentID: r.ParentID, Name: r.Name, Color: r.Color,
		SortOrder: r.SortOrder, CreatedAt: at, UpdatedAt: at}
	f.labels[r.ID] = l
	if f.changedAs != (uuid.UUID{}) {
		l.ID = f.changedAs
	}
	return l, nil
}

// LabelByID is the label id; the reads after the first, under the locks,
// answer as f.reread says.
func (f *fakeLabels) LabelByID(ctx context.Context, id uuid.UUID) (domain.Label, bool, error) {
	f.log.add(ctx, "LabelByID %s", id)
	f.rowReadCount++
	again := f.rowReadCount > 1
	if err := f.fail("LabelByID"); err != nil {
		return domain.Label{}, false, err
	}
	if again && f.reread.err != nil {
		return domain.Label{}, false, fmt.Errorf("LabelByID: %w", f.reread.err)
	}
	l, ok := f.labels[id]
	if !ok || (again && f.reread.gone) {
		return domain.Label{}, false, nil
	}
	if again && f.reread.project != (uuid.UUID{}) {
		l.ProjectID = f.reread.project
	}
	if f.answersAs != (uuid.UUID{}) {
		l.ID = f.answersAs
	}
	return l, true, nil
}

// taken reports whether a label of project but except has name, in any
// case.
func (f *fakeLabels) taken(project, except uuid.UUID, name string) bool {
	return slices.ContainsFunc(f.of(project), func(l domain.Label) bool { return l.ID != except && strings.EqualFold(l.Name, name) })
}

// labelRow is r as CreateLabel logs it: every field but its id, which the
// use case makes.
func labelRow(r app.LabelRow) string {
	return fmt.Sprintf("%s/%s %q %q under %s at %v by %s at %s", r.WorkspaceID, r.ProjectID, r.Name, r.Color, parentOf(r.ParentID), r.SortOrder,
		r.CreatedBy, r.Now.Format(timeFormat))
}

// labelJSON is l as the tests compare labels: its parent by its id, not by
// the pointer.
func labelJSON(t *testing.T, l domain.Label) string {
	t.Helper()
	b, err := json.Marshal(l)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// parentOf is a parent as the fakes log it: its id, or "none".
func parentOf(id *uuid.UUID) string {
	if id == nil {
		return "none"
	}
	return id.String()
}
````

`server/internal/modules/project/app/fakes_state_test.go`（修改，1 处）：

````old server/internal/modules/project/app/fakes_state_test.go
// stateSince is when the fakes' states were made, as stored: no clock's
// time.
````
````new server/internal/modules/project/app/fakes_state_test.go
// stateSince is when the fakes' states and labels were made, as stored: no
// clock's time.
````

`server/internal/modules/project/app/fakes_write_test.go`（修改，1 处）：

````old server/internal/modules/project/app/fakes_write_test.go
	// (fakes_member_test.go) or a state's (fakes_state_test.go): how many
	// ran, how the second one answers, and answersAs, when set, the id each
	// answers for the one asked; changedAs, when set, is the id the write of
	// the row answers for the one it wrote (UpdateMemberRole, CreateState,
	// UpdateState).
````
````new server/internal/modules/project/app/fakes_write_test.go
	// (fakes_member_test.go), a state's (fakes_state_test.go) or a label's
	// (fakes_label_test.go): how many ran, how those after the first answer,
	// and answersAs, when set, the id each answers for the one asked;
	// changedAs, when set, is the id the write of the row answers for the
	// one it wrote (UpdateMemberRole, CreateState, UpdateState,
	// CreateLabel).
````

`server/internal/modules/project/app/create_label_test.go`（新文件，192 行）：

````file server/internal/modules/project/app/create_label_test.go
package app_test

import (
	"context"
	"maps"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newCreateLabel is CreateLabel over newLabels' fakes, its clock logged.
func newCreateLabel() (*app.CreateLabel, *writeFixture, *fakeLabels) {
	f, l := newLabels()
	return app.NewCreateLabel(f.locks(), l, f.tx, clockAt{clockNow, f.log}), f, l
}

// qa is the label the tests create: QA, at the top, a name no label of web
// or ops has, in any case; underBug is QA under web's Bug.
var (
	qa       = domain.LabelCreate{Name: "QA", Color: "#0EA5E9"}
	underBug = domain.LabelCreate{Name: "QA", Color: "#0EA5E9", ParentID: &webBug}
)

// labelCreated are the calls of user's creation of in in project at
// sortOrder: its locks and decision; the parent's read, when in gives one;
// the greatest sort order, when in gives none; the clock; the insert of the
// label, by user at that time.
func labelCreated(user, project uuid.UUID, in domain.LabelCreate, sortOrder float64) []string {
	calls := lockedDecision(user, project, domain.ActionLabelCreate)
	if in.ParentID != nil {
		calls = append(calls, "LabelByID "+in.ParentID.String())
	}
	if in.SortOrder == nil {
		calls = append(calls, "GreatestSortOrder "+project.String())
	}
	return append(calls, "Now", "CreateLabel "+labelRow(app.LabelRow{WorkspaceID: acme.ID, ProjectID: project, ParentID: in.ParentID,
		Name: in.Name, Color: in.Color, SortOrder: sortOrder, CreatedBy: user, Now: clockNow}))
}

// CreateLabel, in one transaction and in the order of M3 design 3.6, locks
// the project, decides, reads the parent given, the greatest sort order of
// its labels when none is given, and the clock, then inserts the label by
// the caller at that time; it answers the label as stored, the time to the
// microsecond: web's at the top after its Feature, 95535; under its Bug,
// the same; at the sort order given, without the greatest read; archived
// ops's, as any other's (3.19), after its Docs, 75535; and ops's, its
// labels gone, at 65535. The use case makes each label's id: a second
// label's differs from the first's, and neither is the nil id, the
// project's, or a label's the fixture had.
func TestCreateLabel(t *testing.T) {
	for _, tt := range []struct {
		name      string
		project   uuid.UUID
		in        domain.LabelCreate
		set       func(l *fakeLabels)
		sortOrder float64
	}{
		{"web, at the top", webID, qa, nil, 95535},
		{"under web's Bug", webID, underBug, nil, 95535},
		{"at its sort order", webID, domain.LabelCreate{Name: "QA", SortOrder: ptr(-1.5)}, nil, -1.5},
		{"archived ops", opsID, qa, nil, 75535},
		{"ops without labels", opsID, qa, func(l *fakeLabels) { delete(l.labels, opsDocs) }, 65535},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, l := newCreateLabel()
			if tt.set != nil {
				tt.set(l)
			}
			before := maps.Clone(l.labels)
			got, err := uc.Execute(as(bob), tt.project, tt.in)
			want := domain.Label{ID: got.ID, WorkspaceID: acme.ID, ProjectID: tt.project, ParentID: tt.in.ParentID, Name: tt.in.Name,
				Color: tt.in.Color, SortOrder: tt.sortOrder, CreatedAt: now, UpdatedAt: now}
			if err != nil || labelJSON(t, got) != labelJSON(t, want) || labelJSON(t, l.labels[got.ID]) != labelJSON(t, want) {
				t.Errorf("Execute() = %+v, %v, stored %+v; want %+v", got, err, l.labels[got.ID], want)
			}
			if want := labelCreated(bob, tt.project, tt.in, tt.sortOrder); !slices.Equal(f.log.calls, want) {
				t.Errorf("calls\n%q\nwant\n%q", f.log.calls, want)
			}
			second := tt.in
			second.Name = "QA again"
			again, err := uc.Execute(as(bob), tt.project, second)
			if err != nil || again.ID == got.ID {
				t.Errorf("a second label = %s, %v; want another id than the first's, %s", again.ID, err, got.ID)
			}
			for _, id := range []uuid.UUID{got.ID, again.ID} {
				if _, had := before[id]; had || id == uuid.Nil() || id == tt.project {
					t.Errorf("a label created as %s; want a new id: not the nil id, the project's or a label's the fixture had", id)
				}
			}
		})
	}
}

// Refusals, each in its place, and no label stored: no caller, and values
// the domain refuses, before the transaction; a project that is not there,
// a workspace or project deleted while its lock waited, a project moved
// meanwhile, and a caller who does not see the project, each
// project.not_found; a member, the Authorizer's 403. A parent that is no
// label, another project's, or one under another, after the decision,
// each 422 parent_id not_allowed. A name another label of the project has,
// in another case, is the store's project.label_name_taken, after the
// insert. The parent read for another id, and the label answered for
// another id, are each the write's own error.
func TestCreateLabelRefuses(t *testing.T) {
	upTo := func(n int) []string { return lockedDecision(bob, webID, domain.ActionLabelCreate)[:n] }
	under := func(parent uuid.UUID) domain.LabelCreate { return domain.LabelCreate{Name: "QA", ParentID: &parent} }
	parentRefused := shared.Invalid(shared.FieldError{Field: "parent_id", Code: shared.FieldNotAllowed})
	none := uuid.NewV7()
	taken := domain.LabelCreate{Name: "bug"}
	for _, tt := range []struct {
		name    string
		ctx     context.Context
		project uuid.UUID
		in      domain.LabelCreate
		set     func(f *writeFixture, l *fakeLabels)
		want    error
		calls   []string
	}{
		{"no caller", context.Background(), webID, qa, nil, shared.Unauthenticated(), nil},
		{"a blank name", as(bob), webID, domain.LabelCreate{Name: " "}, nil,
			shared.Invalid(shared.FieldError{Field: "name", Code: shared.FieldTooShort}), nil},
		{"no project", as(bob), uuid.Nil(), qa, nil, domain.ErrNotFound, noProject},
		{"acme deleted while its lock waited", as(bob), webID, qa, func(f *writeFixture, _ *fakeLabels) { f.workspaces.gone = true },
			domain.ErrNotFound, upTo(3)},
		{"web deleted while its lock waited", as(bob), webID, qa, func(f *writeFixture, _ *fakeLabels) { f.store.deleted = true },
			domain.ErrNotFound, upTo(4)},
		{"web moved to another workspace", as(bob), webID, qa, func(f *writeFixture, _ *fakeLabels) { f.store.moved = uuid.NewV7() },
			domain.ErrNotFound, upTo(4)},
		{"a caller who does not see web", as(erin), webID, qa, nil, domain.ErrNotFound,
			lockedDecision(erin, webID, domain.ActionLabelCreate)},
		{"a member", as(alice), webID, under(webBug), nil, shared.Forbidden(), lockedDecision(alice, webID, domain.ActionLabelCreate)},
		{"a parent that is no label", as(bob), webID, under(none), nil, parentRefused,
			append(lockedDecision(bob, webID, domain.ActionLabelCreate), "LabelByID "+none.String())},
		{"ops's Docs as the parent", as(bob), webID, under(opsDocs), nil, parentRefused,
			append(lockedDecision(bob, webID, domain.ActionLabelCreate), "LabelByID "+opsDocs.String())},
		{"web's UI, under Bug, as the parent", as(bob), webID, under(webUI), nil, parentRefused,
			append(lockedDecision(bob, webID, domain.ActionLabelCreate), "LabelByID "+webUI.String())},
		{"the parent read for another id", as(bob), webID, under(webBug), func(_ *writeFixture, l *fakeLabels) { l.answersAs = webFeature },
			nil, append(lockedDecision(bob, webID, domain.ActionLabelCreate), "LabelByID "+webBug.String())},
		{"a name web's Bug has, in another case", as(bob), webID, taken, nil, domain.ErrLabelNameTaken,
			labelCreated(bob, webID, taken, 95535)},
		{"the label answered for another id", as(bob), webID, qa, func(_ *writeFixture, l *fakeLabels) { l.changedAs = uuid.NewV7() },
			nil, labelCreated(bob, webID, qa, 95535)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, l := newCreateLabel()
			if tt.set != nil {
				tt.set(f, l)
			}
			before := maps.Clone(l.labels)
			_, err := uc.Execute(tt.ctx, tt.project, tt.in)
			outcome{tt.name, tt.want, tt.calls}.check(t, err, f)
			if tt.name != "the label answered for another id" && !maps.Equal(l.labels, before) {
				t.Errorf("the labels after the refusal: %v; want them as they were, %v", l.labels, before)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after.
func TestCreateLabelReturnsEachFailure(t *testing.T) {
	all := labelCreated(bob, webID, underBug, 95535)
	fail := func(method string) func(f *writeFixture) {
		return func(f *writeFixture) { f.store.errs = map[string]error{method: errDisk} }
	}
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of the creation's calls ran
	}{
		{"the project's workspace", fail("ProjectWorkspace"), 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"the project's lock", fail("LockProject"), 4},
		{"the decision", func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, 5},
		{"the parent", fail("LabelByID"), 6},
		{"the greatest sort order", fail("GreatestSortOrder"), 7},
		{"the insert", fail("CreateLabel"), 9},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 9},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, _ := newCreateLabel()
			tt.fail(f)
			_, err := uc.Execute(as(bob), webID, underBug)
			outcome{tt.name, errDisk, all[:tt.calls]}.check(t, err, f)
		})
	}
}
````

`server/internal/modules/project/app/clock_test.go`（修改，2 处）：

````old server/internal/modules/project/app/clock_test.go
// never stamps an earlier time than the one it waited for. So does
// createState, which only inserts, after it reads the project's states
// under its lock. deleteState counts the states its group keeps after it
// deletes the state, the clock read before. The clock logs its read among
// the fakes' calls.
````
````new server/internal/modules/project/app/clock_test.go
// never stamps an earlier time than the one it waited for. So do
// createState and createLabel, which only insert, after they read the
// project's states, or its labels and the parent given, under its lock.
// deleteState counts the states its group keeps after it deletes the
// state, the clock read before. The clock logs its read among the fakes'
// calls.
````

````old server/internal/modules/project/app/clock_test.go
		}, defaultMarked(bob, webTodo, webID)},
````
````new server/internal/modules/project/app/clock_test.go
		}, defaultMarked(bob, webTodo, webID)},
		{"createLabel", func() ([]string, error) {
			uc, f, _ := newCreateLabel()
			_, err := uc.Execute(as(bob), webID, underBug)
			return f.log.calls, err
		}, labelCreated(bob, webID, underBug, 95535)},
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`（`NewCreateLabel` 在 Task 5 起才接线；它是导出的）

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/app/clock_test.go server/internal/modules/project/app/create_label.go server/internal/modules/project/app/create_label_test.go server/internal/modules/project/app/fakes_label_test.go server/internal/modules/project/app/fakes_state_test.go server/internal/modules/project/app/fakes_write_test.go server/internal/modules/project/app/label_parent.go server/internal/modules/project/app/label_ports.go server/internal/modules/project/domain/actions.go
```
```bash
git commit -m "feat(M3/P7b): creating a label locks its project, then checks its parent under the lock

createLabel checks the values first, then locks the workspace FOR SHARE
and the project FOR NO KEY UPDATE, which every write of its labels
takes, and decides label.create, an admin's; an archived project's
labels are created as any other's. Under the lock it reads the parent
given and checks it: a label of the project at the top. The sort order
is the one given, or 10000 after the greatest of the project's labels,
or 65535; the time is the clock's under the lock.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p7b.py` 的编号；"层"是它被发现的每一层，端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `s2-create-greatest` | `createLabel` 越过 `GreatestSortOrder` 的失败 | `TestCreateLabelReturnsEachFailure` | 单元（按性质只在单元一层：真实的存储在这一步不失败） |
| `s2-create-create` | `createLabel` 吞掉 `CreateLabel` 的失败 | `TestCreateLabelRefuses`、`TestCreateLabelReturnsEachFailure` | 单元（按性质只在单元一层：真实的存储在这一步不失败） |
| `s2-parent-read` | `checkParent` 把父标签的读失败当作"没有" | `TestCreateLabelReturnsEachFailure`、`TestUpdateLabelReturnsEachFailure`（Task 6 起） | 单元（按性质只在单元一层：真实的存储在这一步不失败） |
| `s22-created-id` | `createLabel` 收下回答为另一个 id 的标签 | `TestCreateLabelRefuses` | 单元（按性质只在单元一层：真实的存储回答不了别的键（裁定 S6）） |
| `s22-parent-id` | `checkParent` 收下读成另一个 id 的父标签 | `TestCreateLabelRefuses` | 单元（按性质只在单元一层：真实的存储回答不了别的键（裁定 S6）） |
| `p-create-share-project` | `createLabel` 对项目只加 `FOR SHARE`（清扫 13、41：O3） | `TestLabelWritesOnOneProjectSerialize`（Task 10 起） | 组合 |
| `c-create-early` | `createLabel` 在锁之前读时钟 | `TestEachWriteReadsTheClockUnderItsLock`、`TestCreateLabel`、`TestCreateLabelRefuses`、`TestCreateLabelReturnsEachFailure`、`TestLabelWritesOnOneProjectSerialize`（Task 10 起） | 单元；组合 |
| `r-create-unchecked` | `createLabel` 收下任何父标签（不调用 `checkParent`） | `TestEachWriteReadsTheClockUnderItsLock`、`TestCreateLabel`、`TestCreateLabelRefuses`、`TestCreateLabelReturnsEachFailure`、`TestPermissionMatrix`（Task 5 起）、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 5 起）、`TestLabelWritesOnOneProjectSerialize`（Task 10 起）、P7（Task 11 起） | 单元；组合；端到端 |
| `q-given-ignored` | `createLabel` 不理给的 `sort_order` | `TestCreateLabel`、P7（Task 11 起） | 单元；端到端 |
| `q-greatest-ignored` | `createLabel` 不理最大的 `sort_order`（每个新标签都在 65535） | `TestEachWriteReadsTheClockUnderItsLock`、`TestCreateLabel`、`TestCreateLabelRefuses`、`TestCreateLabelReturnsEachFailure`、`TestPermissionMatrix`（Task 5 起）、P7（Task 11 起） | 单元；组合；端到端 |
| `s25-create-parent-unlocked` | `createLabel` 在锁之前读父标签并检查 | `TestPermissionMatrix`（Task 5 起）、`TestLabelWritesOnOneProjectSerialize`（Task 10 起） | 组合 |
| `s5-create-members` | 规则表：`label.create` 给成员 | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix`（Task 5 起）、P7（Task 11 起） | 单元；组合；端到端 |

---

### Task 5: `createLabel` 的接口：契约、处理函数、接线、文案、矩阵和组合的测试

**Files:**
- Create: `server/internal/bootstrap/permission_matrix_labels_test.go`、`server/internal/modules/project/adapter/http/labels.go`、`server/internal/modules/project/adapter/http/labels_test.go`
- Modify: `api/modules/project.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_columns_test.go`、`server/internal/bootstrap/permission_matrix_seeded_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/project_connection_test.go`、`server/internal/bootstrap/project_write_locks_test.go`、`server/internal/bootstrap/project_writes_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/module.go`、`web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.7、2.10；M3 设计 4.10、5.3、9.2）：
  - 契约（`api/modules/project.yaml`）：`POST /api/v0/projects/{project_id}/labels`（`createLabel`，201 `Label`；`x-problem-codes: [validation_failed, project.not_found, forbidden, project.label_name_taken]`）；`Label`（9 个字段都必有，`parent_id` 可以为 null，`additionalProperties: false`）、`LabelCreate`（只有 `name` 必有）；`api/openapi.yaml` 的路径和码 `project.label_name_taken`（409）。
  - `adapter/http/labels.go`（新）：`CreateLabel`（颜色没给时为空，父标签、`sort_order` 没给时 `nil`）、`label`（顶层的父标签答 null）；`handler.go` 的 `UseCases` 加 `CreateLabel`；`module.go` 接上 `app.NewCreateLabel(locks, store, d.Tx, d.Clock)`。
  - 前端：`PROBLEM_MESSAGES` 和两份 `auth.json` 加 `project.label_name_taken`。
  - 矩阵：`permission_matrix_labels_test.go`（新）：`matrixLabels`（每个项目的 Bug、Feature 在顶层，UI 在 Bug 下，`sort_order` 65535、75535、85535；gone 的随它删除）、`projectSeed.labels`（经项目的存储，由工作区的管理员）、`seededLabels`（按名称读回、核对父标签和 `sort_order`）、`ofLabel`、`ofArchivedLabel`、`toLabel`（Task 7 起用）、`withParent`；`createLabel` 的 5 行：QA 在顶层（`createsTheLabel`：在 Feature 之后，95535）、名称被占用（`bug`，409）、父标签有父标签（UI，422）、父标签是另一个项目的（422）、已归档的项目。`seeded.label`、`newSeeded` 给每个项目的每个标签一个 id；`prepareMatrix` 种下、读回。
  - 组合：最先锁工作区的测试（`writesOnAProject` 加 `createLabel`）、盖戳（`createLabel` 在 Web 归档期间：`created_by_id`、`updated_by_id` 是 bob，`updated_at = created_at`）、连接（alice 建 Bug、它下面的 UI，在 UI 下建 Icons 被拒绝 422）。

**Tests:**
- 处理函数：`TestCreateLabel`（给全部字段和只给名称两种，用例收到的值和 201 的回答）、`TestCreateLabelHoldsTheBodyToItsStructure`（没有名称、多余的字段、类型不对、`parent_id` 为 null 或不是 id：400 `bad_request`，不调用用例）、`TestCreateLabelRefusals`（422、404、403、409、500 各按契约答）；`apitest.Main` 核对契约的每个操作在处理函数的测试里都有。
- 矩阵：`TestPermissionMatrix` 的 5 个 `createLabel` 行；`TestThePermissionMatrixCoversEveryOperation`（新操作有行）；`TestEveryColumnCallsAsARegisteredAccount`（从未种下的标签让测试立即失败）。
- 组合：`TestWritesOnAProjectAreEachShape`、`TestEachWriteOnAProjectSharesItsWorkspaceFirst`、`TestTheWritesOnAProjectStampTheirRequest`、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`。

- [ ] **Step 1: 契约**

`api/modules/project.yaml`（修改，2 处）：

````old api/modules/project.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/me/projects/{project_id}/preferences:
````
````new api/modules/project.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/projects/{project_id}/labels:
    parameters:
      - $ref: '#/components/parameters/ProjectID'
    post:
      operationId: createLabel
      tags: [project]
      summary: Create a label in a project
      description: >-
        For the project's admins, and its members who are the workspace's
        admins; an archived project's labels are created as any other's.
        The name has 1–255 characters, not blank, without NUL; the color has
        at most 255, without NUL, and is empty when it is left out
        (validation_failed). The values are checked before the project is
        looked at. The parent, when it is given, is a label of the project
        at the top, not one under another: labels have two levels
        (parent_id not_allowed). The sort order is the one given, or the
        greatest of the project's labels plus 10000, or 65535 when the
        project has none. The name may not be another undeleted label's of
        the project, in any case (project.label_name_taken). A project that
        does not exist, is deleted, or that the caller does not see answers
        project.not_found; one he sees but may not change, forbidden. The
        role is decided after the workspace and project rows are locked, and
        the parent is checked after it.
      security: [{bearer: []}]
      x-problem-codes: [validation_failed, project.not_found, forbidden, project.label_name_taken]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/LabelCreate'
      responses:
        '201':
          description: The new label.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Label'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/me/projects/{project_id}/preferences:
````

````old api/modules/project.yaml
          type: string
        sequence:
          description: The state's place among the project's states, the lowest first.
          type: number

````
````new api/modules/project.yaml
          type: string
        sequence:
          description: The state's place among the project's states, the lowest first.
          type: number
    Label:
      description: A label of a project, at the top or under a label at the top.
      type: object
      additionalProperties: false
      required: [id, workspace_id, project_id, parent_id, name, color, sort_order, created_at, updated_at]
      properties:
        id:
          type: string
          format: uuid
        workspace_id:
          type: string
          format: uuid
        project_id:
          type: string
          format: uuid
        parent_id:
          description: The label it is under; null for a label at the top.
          type: [string, 'null']
          format: uuid
        name:
          type: string
        color:
          description: As the web app's color picker gives it, e.g. "#F59E0B"; empty for none.
          type: string
        sort_order:
          description: The label's place among the project's labels, the lowest first.
          type: number
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time
    LabelCreate:
      type: object
      additionalProperties: false
      required: [name]
      properties:
        name:
          description: 1–255 characters, not blank; another undeleted label of the project may not have it, in any case.
          type: string
        color:
          description: At most 255 characters; empty when left out.
          type: string
        parent_id:
          description: A label of the project at the top, which the new label goes under; left out for a label at the top.
          type: string
          format: uuid
        sort_order:
          description: The label's place among the project's labels; after the others when left out.
          type: number

````

`api/openapi.yaml`（修改，2 处）：

````old api/openapi.yaml
    description: Projects, their members and their states.
````
````new api/openapi.yaml
    description: Projects, their members, their states and their labels.
````

````old api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1workspaces~1{slug}~1states'
````
````new api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1workspaces~1{slug}~1states'
  /api/v0/projects/{project_id}/labels:
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1labels'
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `34ac00ffb1dc033ccff7e9fd4e842242bbecbae2f4351a4f63d7aa12523a5457` | 2846 | `api/dist/openapi.yaml` |
| `a18c31a8c113e3e9b3917c0548c85f57c49d7811c0b10ef1935b54fed9787d99` | 80 | `server/internal/modules/project/adapter/http/gen/bodyshape.gen.go` |
| `0f10629d92ecc82ea082211739f25eb533bf73622679adce0261caefd17cb102` | 3165 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `a474bef282c82734e75712821ad98458034d5bacfb1e2dab63bd1c017e1e1f8c` | 3146 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/bodyshape.gen.go server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 处理函数、接线和文案**

`server/internal/modules/project/adapter/http/labels.go`（新文件，28 行）：

````file server/internal/modules/project/adapter/http/labels.go
package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// CreateLabel serves POST /api/v0/projects/{project_id}/labels: the color
// empty when not given, the parent and the sort order nil when not given.
func (h handler) CreateLabel(ctx context.Context, req gen.CreateLabelRequestObject) (gen.CreateLabelResponseObject, error) {
	in := domain.LabelCreate{Name: req.Body.Name, ParentID: req.Body.ParentID, SortOrder: req.Body.SortOrder}
	if req.Body.Color != nil {
		in.Color = *req.Body.Color
	}
	l, err := h.uc.CreateLabel.Execute(ctx, req.ProjectID, in)
	if err != nil {
		return nil, err
	}
	return gen.CreateLabel201JSONResponse(label(l)), nil
}

// label is l as the API shows it: its parent null at the top.
func label(l domain.Label) gen.Label {
	return gen.Label{ID: l.ID, WorkspaceID: l.WorkspaceID, ProjectID: l.ProjectID, ParentID: orNull(l.ParentID), Name: l.Name, Color: l.Color,
		SortOrder: l.SortOrder, CreatedAt: l.CreatedAt, UpdatedAt: l.UpdatedAt}
}
````

`server/internal/modules/project/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// CreateLabelUseCase is app.CreateLabel.
type CreateLabelUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID, in domain.LabelCreate) (domain.Label, error)
}

// UseCases are the use cases behind the module's operations.
````

````old server/internal/modules/project/adapter/http/handler.go
	ListWorkspaceStates ListWorkspaceStatesUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
	ListWorkspaceStates ListWorkspaceStatesUseCase
	CreateLabel         CreateLabelUseCase
````

`server/internal/modules/project/adapter/http/labels_test.go`（新文件，126 行）：

````file server/internal/modules/project/adapter/http/labels_test.go
package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeCreateLabel is createLabel: each call is recorded as "caller id",
// with what it got; it answers answer, or err.
type fakeCreateLabel struct {
	calls  []string
	got    []domain.LabelCreate
	answer domain.Label
	err    error
}

func (f *fakeCreateLabel) Execute(ctx context.Context, projectID uuid.UUID, in domain.LabelCreate) (domain.Label, error) {
	f.calls = append(f.calls, caller(ctx)+" "+projectID.String())
	f.got = append(f.got, in)
	return f.answer, f.err
}

// web's labels as the use cases answer them: Bug at the top, with a color;
// UI under it, without one, at a sort order with a fraction.
var (
	webBugLabel = domain.Label{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000d1"), WorkspaceID: acmeID, ProjectID: webID, Name: "Bug",
		Color: "#FF0000", SortOrder: 65535, CreatedAt: created, UpdatedAt: created}
	webUILabel = domain.Label{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000d2"), WorkspaceID: acmeID, ProjectID: webID,
		ParentID: &webBugLabel.ID, Name: "UI", SortOrder: 75535.5, CreatedAt: created, UpdatedAt: created.Add(1)}
	webBugLabelJSON = `{"color":"#FF0000","created_at":"2026-10-01T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000d1","name":"Bug",` +
		`"parent_id":null,"project_id":"0199a2b4-0000-7000-8000-0000000000a1","sort_order":65535,"updated_at":"2026-10-01T10:00:00.123456Z",` +
		`"workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	webUILabelJSON = `{"color":"","created_at":"2026-10-01T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000d2","name":"UI",` +
		`"parent_id":"0199a2b4-0000-7000-8000-0000000000d1","project_id":"0199a2b4-0000-7000-8000-0000000000a1","sort_order":75535.5,` +
		`"updated_at":"2026-10-01T10:00:00.123456001Z","workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
)

// The answers of the label operations, as the contract declares them.
const (
	labelNameTakenJSON = `{"status":409,"code":"project.label_name_taken","title":"Conflict",` +
		`"detail":"A label of the project has this name, in this case or another."}`
	parentRefusedJSON = `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
		`"errors":[{"field":"parent_id","code":"not_allowed","message":"must be a label without a parent: labels have two levels"}]}`
)

// parentRefused is the use case's refusal of a parent with a parent of its
// own, as parentRefusedJSON answers it.
var parentRefused = shared.Invalid(shared.FieldError{Field: "parent_id", Code: shared.FieldNotAllowed,
	Message: "must be a label without a parent: labels have two levels"})

// POST goes to the use case for the caller and the path's project, with
// the body's fields: the color empty, the parent and the sort order nil,
// when not given; the answer is 201 with the label the use case answers,
// its parent's id, or null at the top.
func TestCreateLabel(t *testing.T) {
	path := "/api/v0/projects/" + webID.String() + "/labels"
	for _, tt := range []struct {
		body   string
		want   domain.LabelCreate
		answer domain.Label
		json   string
	}{
		{`{"name":"UI","color":"#F59E0B","parent_id":"0199a2b4-0000-7000-8000-0000000000d1","sort_order":-2.5}`,
			domain.LabelCreate{Name: "UI", Color: "#F59E0B", ParentID: &webBugLabel.ID, SortOrder: ptr(-2.5)}, webUILabel, webUILabelJSON},
		{`{"name":"Bug"}`, domain.LabelCreate{Name: "Bug"}, webBugLabel, webBugLabelJSON},
	} {
		create := &fakeCreateLabel{answer: tt.answer}
		h := newServer(t, fakes{createLabel: create})
		if res, body := do(t, h, request(http.MethodPost, path, "alice", tt.body)); res.StatusCode != http.StatusCreated || body != tt.json+"\n" {
			t.Errorf("POST %s = %d %s, want 201 %s", tt.body, res.StatusCode, body, tt.json)
		}
		if want := []string{"alice " + webID.String()}; !slices.Equal(create.calls, want) || !reflect.DeepEqual(create.got,
			[]domain.LabelCreate{tt.want}) {
			t.Errorf("POST %s: calls %q with %+v; want %q with %+v", tt.body, create.calls, create.got, want, tt.want)
		}
	}
}

// A body without its name, with a field it may not have, a field of
// another type, or a parent that is null or no id: refused as bad_request
// before the use case.
func TestCreateLabelHoldsTheBodyToItsStructure(t *testing.T) {
	create := &fakeCreateLabel{}
	h := newServer(t, fakes{createLabel: create})
	for _, body := range []string{`{}`, `{"color":"#000"}`, `{"name":"A","parent":"0199a2b4-0000-7000-8000-0000000000d1"}`, `{"name":1}`,
		`{"name":"A","sort_order":"1"}`, `{"name":"A","parent_id":null}`, `{"name":"A","parent_id":"Bug"}`} {
		res, got := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/labels", "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("POST %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(create.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", create.calls)
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestCreateLabelRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"a parent refused", parentRefused, http.StatusUnprocessableEntity, parentRefusedJSON},
		{"no project", domain.ErrNotFound, http.StatusNotFound, projectNotFoundJSON},
		{"a project member", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"a name taken", domain.ErrLabelNameTaken, http.StatusConflict, labelNameTakenJSON},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{createLabel: &fakeCreateLabel{err: tt.err}})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/labels", "alice", `{"name":"Bug"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	listWorkspaceStates *fakeListWorkspaceStates
````
````new server/internal/modules/project/adapter/http/handler_test.go
	listWorkspaceStates *fakeListWorkspaceStates
	createLabel         *fakeCreateLabel
````

````old server/internal/modules/project/adapter/http/handler_test.go
		f.listWorkspaceStates = &fakeListWorkspaceStates{}
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		f.listWorkspaceStates = &fakeListWorkspaceStates{}
	}
	if f.createLabel == nil {
		f.createLabel = &fakeCreateLabel{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
		MarkDefaultState: f.markDefault, ListWorkspaceStates: f.listWorkspaceStates})
````
````new server/internal/modules/project/adapter/http/handler_test.go
		MarkDefaultState: f.markDefault, ListWorkspaceStates: f.listWorkspaceStates, CreateLabel: f.createLabel})
````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// their members, their states and each member's display settings. It
// brings listing, creating, reading, changing, archiving and deleting
// projects, checking an identifier, listing, adding and joining the
// members, changing a member's role, removing a member and leaving, each
// member's display settings, listing, creating, changing and deleting a
// project's states and making one its default, listing the states of a
// workspace's projects one is a member of, carries out the workspace
// module's cascades on the projects (ProjectCascade), and offers the
// access module its reads of a project (ProjectAccess) and the workspace
// module its count of an account's ended project memberships
// (ProjectMembershipCounts).
````
````new server/internal/modules/project/module.go
// their members, their states, their labels and each member's display
// settings. It brings listing, creating, reading, changing, archiving and
// deleting projects, checking an identifier, listing, adding and joining
// the members, changing a member's role, removing a member and leaving,
// each member's display settings, listing, creating, changing and deleting
// a project's states and making one its default, listing the states of a
// workspace's projects one is a member of, creating a project's labels,
// carries out the workspace module's cascades on the projects
// (ProjectCascade), and offers the access module its reads of a project
// (ProjectAccess) and the workspace module its count of an account's ended
// project memberships (ProjectMembershipCounts).
````

````old server/internal/modules/project/module.go
		ListWorkspaceStates: app.NewListWorkspaceStates(d.Workspaces, store, d.Authorizer),
````
````new server/internal/modules/project/module.go
		ListWorkspaceStates: app.NewListWorkspaceStates(d.Workspaces, store, d.Authorizer),
		CreateLabel:         app.NewCreateLabel(locks, store, d.Tx, d.Clock),
````

`web/apps/web/helpers/authentication.helper.ts`（修改，1 处）：

````old web/apps/web/helpers/authentication.helper.ts
  "project.state_default": "auth.errors.project_state_default",
````
````new web/apps/web/helpers/authentication.helper.ts
  "project.state_default": "auth.errors.project_state_default",
  "project.label_name_taken": "auth.errors.project_label_name_taken",
````

`web/packages/i18n/src/locales/en/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/auth.json
      "project_state_default": "The default state cannot be deleted. Make another state the default first.",
````
````new web/packages/i18n/src/locales/en/auth.json
      "project_state_default": "The default state cannot be deleted. Make another state the default first.",
      "project_label_name_taken": "A label of this project already has this name, in this case or another.",
````

`web/packages/i18n/src/locales/zh-CN/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/auth.json
      "project_state_default": "默认状态不能删除。请先把另一个状态设为默认。",
````
````new web/packages/i18n/src/locales/zh-CN/auth.json
      "project_state_default": "默认状态不能删除。请先把另一个状态设为默认。",
      "project_label_name_taken": "这个项目里已有同名的标签（不区分大小写）。",
````

- [ ] **Step 3: 矩阵**

`server/internal/bootstrap/permission_matrix_labels_test.go`（新文件，150 行）：

````file server/internal/bootstrap/permission_matrix_labels_test.go
package bootstrap

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
)

// The rows of the permission matrix of a project's labels (M3 design 9.2):
// creating them, under the project of each column; and the labels they
// rest on, matrixLabels, which prepareMatrix seeds in each project through
// the project store (labels) and reads back (seededLabels). They are here
// rather than in permission_matrix_seed_test.go, which has no room for
// them.

var cellLabelNameTaken = cell{http.StatusConflict, "project.label_name_taken"}

// matrixLabels are the labels prepareMatrix seeds in each project of
// matrixProjects, by its workspace's admin, at the sort orders createLabel
// gives labels created one after another: Bug and Feature at the top, UI
// under Bug. gone's are deleted with it.
var matrixLabels = []struct {
	name, parent string // parent: the name of the label it is under, "" at the top
	sortOrder    float64
}{
	{"Bug", "", 65535}, {"UI", "Bug", 75535}, {"Feature", "", 85535},
}

// newLabel is the body of the label the rows create: QA, at the top, a
// name no seeded label has in any case.
const newLabel = `{"name":"QA","color":"#0EA5E9"}`

// withParent is the request of a row whose callers each create QA under
// the label parent names for their column.
func withParent(parent func(c caller, s seeded) uuid.UUID) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return toProject(http.MethodPost, "/labels", `{"name":"QA","parent_id":"`+parent(c, s).String()+`"}`)(c, s)
	}
}

func labelMatrixRows() []matrixRow {
	return []matrixRow{
		// The project's admins, and its members who are the workspace's
		// admins (M3 design 3.4), as createState.
		{op: "createLabel", write: true, columns: projectColumns, request: toProject(http.MethodPost, "/labels", newLabel),
			cells: ofProject(cellCreated, cellForbidden, cellForbidden, cellCreated, cellForbidden, cellForbidden), check: createsTheLabel},
		// Bug in another case: the 409 comes after the decision, so who may
		// not create labels learns nothing of the project's.
		{op: "createLabel", variant: "a name taken", write: true, columns: projectColumns,
			request: toProject(http.MethodPost, "/labels", `{"name":"bug"}`),
			cells:   ofProject(cellLabelNameTaken, cellForbidden, cellForbidden, cellLabelNameTaken, cellForbidden, cellForbidden)},
		// UI, under Bug, as the parent: labels have two levels (M3 design
		// 3.16), and the parent is checked after the decision.
		{op: "createLabel", variant: "a parent with a parent", write: true, columns: projectColumns,
			request: withParent(func(c caller, s seeded) uuid.UUID { return s.label(projectOf(c), "UI") }),
			cells:   ofProject(cellValidationFailed, cellForbidden, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "parent_id not_allowed"},
		// other's project's Bug, a label of a project of another workspace.
		{op: "createLabel", variant: "a parent of another project", write: true, columns: projectColumns,
			request: withParent(func(_ caller, s seeded) uuid.UUID { return s.label("other/project", "Bug") }),
			cells:   ofProject(cellValidationFailed, cellForbidden, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "parent_id not_allowed"},
		// An archived project's labels are created as any other's (M3 design
		// 3.19).
		{op: "createLabel", variant: "archived", write: true, columns: archivedColumns, request: toProject(http.MethodPost, "/labels", newLabel),
			cells: ofArchived(cellCreated, cellForbidden), check: createsTheLabel},
	}
}

// createsTheLabel: QA at the top of the column's project, after its
// Feature, the greatest sort order of its labels.
func createsTheLabel(t *testing.T, c caller, s seeded, answer string) {
	var l struct {
		ProjectID uuid.UUID  `json:"project_id"`
		ParentID  *uuid.UUID `json:"parent_id"`
		Name      string     `json:"name"`
		Color     string     `json:"color"`
		SortOrder float64    `json:"sort_order"`
	}
	decodeAnswer(t, answer, &l)
	if l.ProjectID != s.project(projectOf(c)) || l.ParentID != nil || l.Name != "QA" || l.Color != "#0EA5E9" || l.SortOrder != 95535 {
		t.Errorf("%s creates %s; want QA in %s, at the top, at 95535", c, answer, projectOf(c))
	}
}

// labels stores each project's matrixLabels, each parent before the labels
// under it, with the ids sd names, by its workspace's admin.
func (s projectSeed) labels(sd seeded) {
	s.t.Helper()
	for _, p := range matrixProjects {
		slug, _, _ := strings.Cut(p.key, "/")
		for _, l := range matrixLabels {
			var parent *uuid.UUID
			if l.parent != "" {
				id := sd.label(p.key, l.parent)
				parent = &id
			}
			if _, err := s.store.CreateLabel(context.Background(), projectapp.LabelRow{ID: sd.label(p.key, l.name), WorkspaceID: s.workspaces[slug],
				ProjectID: s.projects[p.key], ParentID: parent, Name: l.name, SortOrder: l.sortOrder, CreatedBy: s.ids[matrixAdmins[slug]],
				Now: s.now}); err != nil {
				s.t.Fatal(err)
			}
		}
	}
}

// seededLabel is a label of a matrix project as seededLabels reads it: its
// parent by name, "" at the top.
type seededLabel struct {
	Name, Parent string
	SortOrder    float64
	Deleted      bool
}

// seededLabels checks the labels the cells of each matrix project rest on,
// read back by name: those of matrixLabels exactly, each under its parent,
// at its sort order; undeleted, but gone's, deleted with gone. A label
// missing or seeded otherwise would let a cell answer as it wants for
// another reason.
func (s projectSeed) seededLabels(pool *pgxpool.Pool) {
	s.t.Helper()
	for _, p := range matrixProjects {
		gone := strings.HasPrefix(p.key, "gone/")
		var want []seededLabel
		for _, l := range matrixLabels {
			want = append(want, seededLabel{Name: l.name, Parent: l.parent, SortOrder: l.sortOrder, Deleted: gone})
		}
		slices.SortFunc(want, func(a, b seededLabel) int { return strings.Compare(a.Name, b.Name) })
		rows, err := pool.Query(context.Background(), `SELECT l.name, coalesce(p.name, ''), l.sort_order, l.deleted_at IS NOT NULL
			FROM labels l LEFT JOIN labels p ON p.id = l.parent_id WHERE l.project_id = $1 ORDER BY l.name COLLATE "C"`, s.projects[p.key])
		if err != nil {
			s.t.Fatal(err)
		}
		got, err := pgx.CollectRows(rows, pgx.RowToStructByPos[seededLabel])
		if err != nil {
			s.t.Fatal(err)
		}
		if !slices.Equal(got, want) {
			s.t.Fatalf("%s's labels by name = %+v; want %+v", p.key, got, want)
		}
	}
}
````

`server/internal/bootstrap/permission_matrix_seeded_test.go`（修改，6 处）：

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// by the project's key and the column; each state, by the project's key
// and its name; and each account, by its name in matrixAccounts, which
// prepareMatrix registers. t is the test that asks for them (in).
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// by the project's key and the column; each state and each label, by the
// project's key and its name; and each account, by its name in
// matrixAccounts, which prepareMatrix registers. t is the test that asks
// for them (in).
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
	states         map[string]uuid.UUID
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
	states         map[string]uuid.UUID
	labels         map[string]uuid.UUID
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
// of matrixProjectMembers and each of matrixStates in each project before
// prepareMatrix writes them, so that matrixViolations, without a database,
// sees the keys and the targets the cells will.
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
// of matrixProjectMembers and each of matrixStates and matrixLabels in each
// project before prepareMatrix writes them, so that matrixViolations,
// without a database, sees the keys and the targets the cells will.
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
		projects: map[string]uuid.UUID{}, projectMembers: map[string]uuid.UUID{}, states: map[string]uuid.UUID{}, accounts: map[caller]uuid.UUID{}}
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
		projects: map[string]uuid.UUID{}, projectMembers: map[string]uuid.UUID{}, states: map[string]uuid.UUID{}, labels: map[string]uuid.UUID{},
		accounts: map[caller]uuid.UUID{}}
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
			s.states[p.key+"|"+st.Name] = uuid.NewV7()
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
			s.states[p.key+"|"+st.Name] = uuid.NewV7()
		}
		for _, l := range matrixLabels {
			s.labels[p.key+"|"+l.name] = uuid.NewV7()
````

````old server/internal/bootstrap/permission_matrix_seeded_test.go
}

// projectOfRow is the key of the project of id, a row of rows, which are
````
````new server/internal/bootstrap/permission_matrix_seeded_test.go
}

// label is the id of the label name of the project key; one never seeded
// fails the test at once, as membership's does.
func (s seeded) label(key, name string) uuid.UUID {
	id, ok := s.labels[key+"|"+name]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no label %s of %s is seeded", name, key)
	}
	return id
}

// projectOfRow is the key of the project of id, a row of rows, which are
````

`server/internal/bootstrap/permission_matrix_test.go`（修改，7 处）：

````old server/internal/bootstrap/permission_matrix_test.go
// memberships through the workspace store, the projects, their memberships
// and their states through the project store, the deleted workspace
// through the API, and the membership states no store makes alone through
// SQL (prepareMatrix).
````
````new server/internal/bootstrap/permission_matrix_test.go
// memberships through the workspace store, the projects, their
// memberships, their states and their labels through the project store,
// the deleted workspace through the API, and the membership states no
// store makes alone through SQL (prepareMatrix).
````

````old server/internal/bootstrap/permission_matrix_test.go
// permission_matrix_seeded_test.go.
````
````new server/internal/bootstrap/permission_matrix_test.go
// permission_matrix_seeded_test.go; the labels' writer and check are
// beside their rows.
````

````old server/internal/bootstrap/permission_matrix_test.go
	return slices.Concat(workspaceMatrixRows(), projectMatrixRows(), memberMatrixRows(), membershipMatrixRows(), stateMatrixRows())
````
````new server/internal/bootstrap/permission_matrix_test.go
	return slices.Concat(workspaceMatrixRows(), projectMatrixRows(), memberMatrixRows(), membershipMatrixRows(), stateMatrixRows(),
		labelMatrixRows())
````

````old server/internal/bootstrap/permission_matrix_test.go
// projects, project memberships and states of matrixProjects,
// matrixProjectMembers and matrixStates, and acme's archived project
// archived. Through both stores, the removed member's removal; then,
// through the workspace store, the invitations of matrixInvitations.
````
````new server/internal/bootstrap/permission_matrix_test.go
// projects, project memberships, states and labels of matrixProjects,
// matrixProjectMembers, matrixStates and matrixLabels, and acme's archived
// project archived. Through both stores, the removed member's removal;
// then, through the workspace store, the invitations of matrixInvitations.
````

````old server/internal/bootstrap/permission_matrix_test.go
// memberships and its project, with its states, with it; then the checks
// that the rows the cells rest on are there (preconditions), the states
// among them (seededStates). Everything that connected to the database is
// closed when it returns, so that it can be copied. A -run that leaves out
// prepare fails here, not with a 401 in every cell.
````
````new server/internal/bootstrap/permission_matrix_test.go
// memberships and its project, with its states and labels, with it; then
// the checks that the rows the cells rest on are there (preconditions), the
// states and the labels among them (seededStates, seededLabels).
// Everything that connected to the database is closed when it returns, so
// that it can be copied. A -run that leaves out prepare fails here, not
// with a 401 in every cell.
````

````old server/internal/bootstrap/permission_matrix_test.go
		projects.states(s)
````
````new server/internal/bootstrap/permission_matrix_test.go
		projects.states(s)
		projects.labels(s)
````

````old server/internal/bootstrap/permission_matrix_test.go
		projects.seededStates(pool)
````
````new server/internal/bootstrap/permission_matrix_test.go
		projects.seededStates(pool)
		projects.seededLabels(pool)
````

`server/internal/bootstrap/permission_matrix_columns_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_columns_test.go
	// And a state never seeded.
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
	// A state never seeded.
````

````old server/internal/bootstrap/permission_matrix_columns_test.go
		t.Errorf("a state never seeded: failed with %q, want %q", failed, want)
	}
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
		t.Errorf("a state never seeded: failed with %q, want %q", failed, want)
	}
	// And a label never seeded.
	failed = fatalOf(func(tb testing.TB) { newSeeded().in(tb).label("acme/public", "Bugs") })
	if want := "no label Bugs of acme/public is seeded"; failed != want {
		t.Errorf("a label never seeded: failed with %q, want %q", failed, want)
	}
````

- [ ] **Step 4: 组合的测试**

`server/internal/bootstrap/project_write_locks_test.go`（修改，1 处）：

````old server/internal/bootstrap/project_write_locks_test.go
	{op: "markDefaultState", method: http.MethodPost, path: "/api/v0/states/%s/mark-default", want: http.StatusNoContent, row: stateNamed("Done")},
````
````new server/internal/bootstrap/project_write_locks_test.go
	{op: "markDefaultState", method: http.MethodPost, path: "/api/v0/states/%s/mark-default", want: http.StatusNoContent, row: stateNamed("Done")},
	{op: "createLabel", method: http.MethodPost, path: "/api/v0/projects/%s/labels", body: `{"name":"QA"}`, want: http.StatusCreated},
````

`server/internal/bootstrap/project_writes_test.go`（修改，3 处）：

````old server/internal/bootstrap/project_writes_test.go
// Web, deletes the state and makes Done the default, unarchives Web, and
// leaves it. Each row the write writes again is first made bob's, as last
// written by him, and checked so: a write that kept its row's writer would
// pass for alice's otherwise, she having made it.
````
````new server/internal/bootstrap/project_writes_test.go
// Web, deletes the state, makes Done the default and creates a label,
// unarchives Web, and leaves it. Each row the write writes again is first
// made bob's, as last written by him, and checked so: a write that kept its
// row's writer would pass for alice's otherwise, she having made it.
````

````old server/internal/bootstrap/project_writes_test.go
		// Web archived for the two state writes after it, then unarchived: an
		// archived project's states are deleted, and its default made, as any
		// other's (M3 design 3.19), and their rows read the effect back.
		{"archiveProject, before the state writes", http.MethodPost, "/api/v0/projects/" + web.String() + "/archive", "", nil,
````
````new server/internal/bootstrap/project_writes_test.go
		// Web archived for the state and label writes after it, then
		// unarchived: an archived project's states are deleted, its default
		// made, and its labels written, as any other's (M3 design 3.19), and
		// their rows read the effect back.
		{"archiveProject, before the state and label writes", http.MethodPost, "/api/v0/projects/" + web.String() + "/archive", "", nil,
````

````old server/internal/bootstrap/project_writes_test.go
		{"unarchiveProject, after the state writes", http.MethodPost, "/api/v0/projects/" + web.String() + "/unarchive", "", nil,
````
````new server/internal/bootstrap/project_writes_test.go
		// The label QA, made.
		{"createLabel", http.MethodPost, "/api/v0/projects/" + web.String() + "/labels", `{"name":"QA"}`, nil, http.StatusCreated, "",
			"SELECT created_at, created_by_id = $2 AND updated_by_id = $2 AND updated_at = created_at FROM labels WHERE project_id = $1 AND name = 'QA'",
			0, 1},
		{"unarchiveProject, after the state and label writes", http.MethodPost, "/api/v0/projects/" + web.String() + "/unarchive", "", nil,
````

`server/internal/bootstrap/project_connection_test.go`（修改，2 处）：

````old server/internal/bootstrap/project_connection_test.go
// counts the group again, and makes Done Web's default; carol, an admin,
// leaves it; alice deletes both projects.
````
````new server/internal/bootstrap/project_connection_test.go
// counts the group again, and makes Done Web's default; she creates the
// label Bug in it, UI under Bug, and Icons under UI, which the parent's
// read refuses (422: labels have two levels); carol, an admin, leaves it;
// alice deletes both projects.
````

````old server/internal/bootstrap/project_connection_test.go
	send(http.MethodPost, "/api/v0/states/"+done.ID.String()+"/mark-default", r.alice, "", http.StatusNoContent)
````
````new server/internal/bootstrap/project_connection_test.go
	send(http.MethodPost, "/api/v0/states/"+done.ID.String()+"/mark-default", r.alice, "", http.StatusNoContent)
	var bug, ui struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, send(http.MethodPost, web+"/labels", r.alice, `{"name":"Bug"}`, http.StatusCreated), &bug)
	decodeAnswer(t, send(http.MethodPost, web+"/labels", r.alice, `{"name":"UI","parent_id":"`+bug.ID.String()+`"}`, http.StatusCreated), &ui)
	send(http.MethodPost, web+"/labels", r.alice, `{"name":"Icons","parent_id":"`+ui.ID.String()+`"}`, http.StatusUnprocessableEntity)
````

- [ ] **Step 5: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation$|TestEveryColumnCallsAsARegisteredAccount$|TestMatrixViolationsCatchesEachColumnGap$|TestWritesOnAProjectAreEachShape$|TestEachWriteOnAProjectSharesItsWorkspaceFirst$|TestTheWritesOnAProjectStampTheirRequest$|TestTheWritesOnAProjectRunOnTheirTransactionsConnection$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

- [ ] **Step 6: 提交**

```bash
git add api/modules/project.yaml api/openapi.yaml server/internal/bootstrap/permission_matrix_columns_test.go server/internal/bootstrap/permission_matrix_labels_test.go server/internal/bootstrap/permission_matrix_seeded_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/project_connection_test.go server/internal/bootstrap/project_write_locks_test.go server/internal/bootstrap/project_writes_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/labels.go server/internal/modules/project/adapter/http/labels_test.go server/internal/modules/project/module.go web/apps/web/helpers/authentication.helper.ts web/packages/i18n/src/locales/en/auth.json web/packages/i18n/src/locales/zh-CN/auth.json api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/bodyshape.gen.go server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P7b): POST /api/v0/projects/{project_id}/labels creates a label

The contract declares createLabel and the label: its parent null at the
top. The handler hands the body's fields to the use case, the color
empty and the parent and the sort order nil when left out. The
permission matrix seeds three labels in each project and asks each
column to create one, with a name taken, under a parent with a parent,
under another project's label, and in the archived project; the writes
on a project share its workspace first, stamp their request and run on
their transaction's connection with createLabel among them.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p7b.py` 的编号；"层"是它被发现的每一层，端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `s4-create-frozen-clock` | `createLabel` 接上停在 2001 年的时钟 | `TestTheWritesOnAProjectStampTheirRequest` | 组合 |
| `s4-create-drops-parent` | `POST …/labels` 丢掉父标签 | `TestCreateLabel`、`TestPermissionMatrix`、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`、P7（Task 11 起） | 单元；组合；端到端 |
| `s4-create-drops-sort` | `POST …/labels` 丢掉 `sort_order` | `TestCreateLabel`、P7（Task 11 起） | 单元；端到端 |
| `s4-create-drops-color` | `POST …/labels` 丢掉颜色 | `TestCreateLabel`、`TestPermissionMatrix`、P7（Task 11 起） | 单元；组合；端到端 |
| `s4-label-parent-null` | 回答的标签一律在顶层（`parent_id` 为 null） | `TestCreateLabel`、`TestUpdateLabel`（Task 6 起）、`TestListLabels`（Task 9 起）、`TestPermissionMatrix`（Task 9 起）、P7（Task 11 起） | 单元；组合；端到端 |
| `s18-create-no-taken` | `createLabel` 的契约不声明 `project.label_name_taken` | `TestCreateLabelRefusals`、`TestPermissionMatrix`、`TestEachLabelWriteChangesItsRowsAlone`（Task 10 起） | 单元；组合 |

---

### Task 6: `updateLabel` 的用例：共用的行路径、子标签、父标签

**Files:**
- Create: `server/internal/modules/project/app/update_label.go`、`server/internal/modules/project/app/update_label_test.go`
- Modify: `server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/app/clock_test.go`、`server/internal/modules/project/app/fakes_label_test.go`、`server/internal/modules/project/app/fakes_write_test.go`、`server/internal/modules/project/app/label_ports.go`、`server/internal/modules/project/domain/actions.go`

**Interfaces:**
- Consumes：P7a 的 `lockRowAndDecide[R placed]`、`rowWrite{id, action, find, targets, notFound}`；Task 2 的 `CheckLabelPatch`、`Label.Place()`；Task 4 的 `checkParent`、`fakeLabels`。
- Produces（spec 2.6；M3 设计 3.6、3.16、3.19）：
  - `label_ports.go`：`labelWrite(id, action, labels LabelFinder) rowWrite[domain.Label]`（`find` 是 `labels.LabelByID`，404 是 `project.label_not_found`）；`LabelUpdater`（`LabelFinder`、`HasChildren`、`UpdateLabel`）；`LabelFinder` 的说明写明两种读。
  - `update_label.go`（新）：`NewUpdateLabel(locks, labels LabelUpdater, tx, clock)`、`Execute(ctx, id, domain.LabelPatch) (domain.Label, error)`：调用者 → `CheckLabelPatch`（事务之前）→ 一个事务：`lockRowAndDecide(…, labelWrite(id, ActionLabelUpdate, labels))`（读标签得到项目和工作区、工作区 `FOR SHARE`、项目 `FOR NO KEY UPDATE`、锁下重读、`label.update` 的判定；已归档的项目照常）→ 给了父标签（不是 null）时 `HasChildren`，再 `checkParent(…, l.ID, l.ProjectID, parent, hasChildren)`；移到顶层两者都不需要 → 时钟 → `UpdateLabel`；回答的 id 不是这个标签的是错误。
  - `label.update`：同 `label.create`。
  - `fakeLabels` 加 `HasChildren`、`UpdateLabel`、`newUpdateLabel`、`labelLocked`、`labelUpdated`；`writeFixture` 的重读（`reread`）也答标签。

**Tests:**
- `TestUpdateLabel`：Feature 改名、改色；Feature 放到 Bug 下；给 `sort_order`；Bug 换大小写改名；UI 移到顶层；已归档的 Ops 的 Docs 改名。每一行核对调用的次序（给了父标签才读子标签和父标签）和存下的标签。
- `TestUpdateLabelRefuses`：没有调用者、空的名称（事务之前）；没有标签、工作区或项目在锁等待期间删除、标签在锁等待期间删除或移到 Ops、看不见 Web 的调用者，各 `project.label_not_found`；成员 403；Feature 在它自己下面、在 Bug 下面的 UI 下面、在 Ops 的 Docs 下面、在不存在的标签下面，有 UI 的 Bug 在 Feature 下面，各 422 `parent_id not_allowed`（判定之后，先读子标签、再读父标签）；Feature 改成 Bug 的名称（另一种大小写）是存储的 409；修改回答另一个标签是这个写自己的错误（重读的 id 的检查在 P7a 的 `lock_test.go` 里钉过一次，这里不再加行，S6）。每一行之后标签不变（最后一行除外：假实现已写）。
- `TestUpdateLabelReturnsEachFailure`：标签的读、两把锁、锁下的重读、判定、子标签、父标签、修改、提交，各原样返回。
- `TestEachWriteReadsTheClockUnderItsLock` 加 `updateLabel`；`TestEveryRuleDecidesItsCells` 加 `label.update`。

- [ ] **Step 1: 规则和动作**

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"label.create": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````
````new server/internal/modules/access/domain/rules.go
	"label.create": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	"label.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"label.create": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"label.create": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"label.update": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````

`server/internal/modules/project/domain/actions.go`（修改，2 处）：

````old server/internal/modules/project/domain/actions.go
	ActionLabelCreate shared.Action = "label.create"
````
````new server/internal/modules/project/domain/actions.go
	ActionLabelCreate shared.Action = "label.create"
	// ActionLabelUpdate is changing a label: updateLabel.
	ActionLabelUpdate shared.Action = "label.update"
````

````old server/internal/modules/project/domain/actions.go
		ActionLabelCreate}
````
````new server/internal/modules/project/domain/actions.go
		ActionLabelCreate, ActionLabelUpdate}
````

- [ ] **Step 2: 端口和用例**

`server/internal/modules/project/app/label_ports.go`（修改，4 处）：

````old server/internal/modules/project/app/label_ports.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
````
````new server/internal/modules/project/app/label_ports.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````

````old server/internal/modules/project/app/label_ports.go
// LabelFinder reads a label by its id: a label a write names as the
````
````new server/internal/modules/project/app/label_ports.go
// LabelFinder reads a label by its id: a label a write names, first
// without a lock, for its project and the project's workspace, then again
// under their locks (lockRowAndDecide); and a label a write names as the
````

````old server/internal/modules/project/app/label_ports.go
	LabelByID(ctx context.Context, id uuid.UUID) (l domain.Label, found bool, err error)
````
````new server/internal/modules/project/app/label_ports.go
	LabelByID(ctx context.Context, id uuid.UUID) (l domain.Label, found bool, err error)
}

// labelWrite is the write action on the label id, addressed by its
// resource (rowWrite): labels reads it, and project.label_not_found is its
// 404.
func labelWrite(id uuid.UUID, action shared.Action, labels LabelFinder) rowWrite[domain.Label] {
	return rowWrite[domain.Label]{id: id, action: action, find: labels.LabelByID, notFound: domain.ErrLabelNotFound}
````

````old server/internal/modules/project/app/label_ports.go
	CreateLabel(ctx context.Context, r LabelRow) (domain.Label, error)
}

````
````new server/internal/modules/project/app/label_ports.go
	CreateLabel(ctx context.Context, r LabelRow) (domain.Label, error)
}

// LabelUpdater is updateLabel's repository. HasChildren and UpdateLabel run
// in the transaction ctx carries, under the project's FOR NO KEY UPDATE;
// LabelByID's reads are as LabelFinder says.
type LabelUpdater interface {
	LabelFinder
	// HasChildren reports whether an undeleted label has the label id as
	// its parent.
	HasChildren(ctx context.Context, id uuid.UUID) (bool, error)
	// UpdateLabel changes the fields p gives of the undeleted label id, by
	// the account by at now, and answers it as stored. A name another
	// undeleted label of the project has, in any case, is
	// domain.ErrLabelNameTaken.
	UpdateLabel(ctx context.Context, id uuid.UUID, p domain.LabelPatch, by uuid.UUID, now time.Time) (domain.Label, error)
}

````

`server/internal/modules/project/app/update_label.go`（新文件，71 行）：

````file server/internal/modules/project/app/update_label.go
package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateLabel changes a label: PATCH /api/v0/labels/{label_id} (M3 design
// 3.16).
type UpdateLabel struct {
	locks  Locks
	labels LabelUpdater
	tx     shared.TxManager
	clock  Clock
}

// NewUpdateLabel returns the use case.
func NewUpdateLabel(locks Locks, labels LabelUpdater, tx shared.TxManager, clock Clock) *UpdateLabel {
	return &UpdateLabel{locks: locks, labels: labels, tx: tx, clock: clock}
}

// Execute checks p (domain.CheckLabelPatch), then in one transaction, in
// the order of M3 design 3.6: the label's locks (lockRowAndDecide: the
// label read for its project and workspace, the workspace FOR SHARE, the
// project FOR NO KEY UPDATE, which every write of its labels takes, the
// label read again) and the decision on label.update; an archived
// project's labels change as any other's (3.19). Then, when p gives a
// parent, whether the label has labels under it and the parent
// (checkParent); a parent given as none, to the top, needs neither. Then
// the change, at the time the clock gives under the locks. The answer is
// the label as stored.
func (u *UpdateLabel) Execute(ctx context.Context, id uuid.UUID, p domain.LabelPatch) (domain.Label, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Label{}, err
	}
	if err := domain.CheckLabelPatch(p); err != nil {
		return domain.Label{}, err
	}
	var updated domain.Label
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		_, l, err := lockRowAndDecide(ctx, u.locks, actor, labelWrite(id, domain.ActionLabelUpdate, u.labels))
		if err != nil {
			return err
		}
		if p.SetParent && p.ParentID != nil {
			hasChildren, err := u.labels.HasChildren(ctx, l.ID)
			if err != nil {
				return err
			}
			if err := checkParent(ctx, u.labels, l.ID, l.ProjectID, *p.ParentID, hasChildren); err != nil {
				return err
			}
		}
		if updated, err = u.labels.UpdateLabel(ctx, l.ID, p, actor.UserID, u.clock.Now()); err != nil {
			return err
		}
		if updated.ID != l.ID {
			return fmt.Errorf("label %s changed as %s", l.ID, updated.ID)
		}
		return nil
	})
	if err != nil {
		return domain.Label{}, err
	}
	return updated, nil
}
````

- [ ] **Step 3: 假实现和测试**

`server/internal/modules/project/app/fakes_label_test.go`（修改，5 处）：

````old server/internal/modules/project/app/fakes_label_test.go
// reads of a row by its id and changedAs it shares.
````
````new server/internal/modules/project/app/fakes_label_test.go
// reads of a row by its id and changedAs it shares. failsFor, when set, is
// the label whose read fails (errDisk), the others read as they are: how a
// test fails the read of a parent and not the label's own.
````

````old server/internal/modules/project/app/fakes_label_test.go
	labels map[uuid.UUID]domain.Label
````
````new server/internal/modules/project/app/fakes_label_test.go
	labels   map[uuid.UUID]domain.Label
	failsFor uuid.UUID
````

````old server/internal/modules/project/app/fakes_label_test.go
		return domain.Label{}, false, err
	}
````
````new server/internal/modules/project/app/fakes_label_test.go
		return domain.Label{}, false, err
	}
	if f.failsFor != (uuid.UUID{}) && id == f.failsFor {
		return domain.Label{}, false, fmt.Errorf("LabelByID %s: %w", id, errDisk)
	}
````

````old server/internal/modules/project/app/fakes_label_test.go
}

// taken reports whether a label of project but except has name, in any
````
````new server/internal/modules/project/app/fakes_label_test.go
}

// HasChildren reports whether a label has id as its parent.
func (f *fakeLabels) HasChildren(ctx context.Context, id uuid.UUID) (bool, error) {
	f.log.add(ctx, "HasChildren %s", id)
	if err := f.fail("HasChildren"); err != nil {
		return false, err
	}
	for _, l := range f.labels {
		if l.ParentID != nil && *l.ParentID == id {
			return true, nil
		}
	}
	return false, nil
}

// UpdateLabel changes the fields p gives of the label id, as stored,
// unless another label of its project has the name p gives, in any case:
// domain.ErrLabelNameTaken.
func (f *fakeLabels) UpdateLabel(ctx context.Context, id uuid.UUID, p domain.LabelPatch, by uuid.UUID, now time.Time) (domain.Label, error) {
	f.log.add(ctx, "UpdateLabel %s %s by %s at %s", id, labelPatch(p), by, now.Format(timeFormat))
	if err := f.fail("UpdateLabel"); err != nil {
		return domain.Label{}, err
	}
	l, ok := f.labels[id]
	if !ok {
		return domain.Label{}, fmt.Errorf("UpdateLabel: no label %s", id)
	}
	if p.Name != nil {
		if f.taken(l.ProjectID, id, *p.Name) {
			return domain.Label{}, domain.ErrLabelNameTaken
		}
		l.Name = *p.Name
	}
	if p.Color != nil {
		l.Color = *p.Color
	}
	if p.SetParent {
		l.ParentID = p.ParentID
	}
	if p.SortOrder != nil {
		l.SortOrder = *p.SortOrder
	}
	l.UpdatedAt = now.Truncate(time.Microsecond)
	f.labels[id] = l
	if f.changedAs != (uuid.UUID{}) {
		l.ID = f.changedAs
	}
	return l, nil
}

// taken reports whether a label of project but except has name, in any
````

````old server/internal/modules/project/app/fakes_label_test.go
		r.CreatedBy, r.Now.Format(timeFormat))
````
````new server/internal/modules/project/app/fakes_label_test.go
		r.CreatedBy, r.Now.Format(timeFormat))
}

// labelPatch is p as UpdateLabel logs it: the fields it gives, the parent
// "none" for a label moved to the top.
func labelPatch(p domain.LabelPatch) string {
	out := "{"
	if p.Name != nil {
		out += fmt.Sprintf(" name %q", *p.Name)
	}
	if p.Color != nil {
		out += fmt.Sprintf(" color %q", *p.Color)
	}
	if p.SetParent {
		out += " parent " + parentOf(p.ParentID)
	}
	if p.SortOrder != nil {
		out += fmt.Sprintf(" sort order %v", *p.SortOrder)
	}
	return out + " }"
````

`server/internal/modules/project/app/fakes_write_test.go`（修改，1 处）：

````old server/internal/modules/project/app/fakes_write_test.go
	// CreateLabel).
````
````new server/internal/modules/project/app/fakes_write_test.go
	// CreateLabel, UpdateLabel).
````

`server/internal/modules/project/app/update_label_test.go`（新文件，185 行）：

````file server/internal/modules/project/app/update_label_test.go
package app_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newUpdateLabel is UpdateLabel over newLabels' fakes, its clock logged.
func newUpdateLabel() (*app.UpdateLabel, *writeFixture, *fakeLabels) {
	f, l := newLabels()
	return app.NewUpdateLabel(f.locks(), l, f.tx, clockAt{clockNow, f.log}), f, l
}

// labelLocked are the calls of a write by caller on the label id of
// project up to its decision on action: the transaction, the label read,
// acme's row FOR SHARE, the project FOR NO KEY UPDATE, the label read
// again, the decision.
func labelLocked(id, project, caller uuid.UUID, action shared.Action) []string {
	return []string{"Begin", "LabelByID " + id.String(), "ShareWorkspaceByID " + acme.ID.String(), "LockProject " + project.String(),
		"LabelByID " + id.String(), fmt.Sprintf("Authorize %s %s on %s/%s", caller, action, acme.ID, project)}
}

// labelUpdated are the calls of user's change p of the label id of
// project: its locks and decision; whether it has labels under it and the
// parent's read, when p gives a parent; the clock; the change by user at
// that time.
func labelUpdated(user, id, project uuid.UUID, p domain.LabelPatch) []string {
	calls := labelLocked(id, project, user, domain.ActionLabelUpdate)
	if p.SetParent && p.ParentID != nil {
		calls = append(calls, "HasChildren "+id.String(), "LabelByID "+p.ParentID.String())
	}
	return append(calls, "Now", fmt.Sprintf("UpdateLabel %s %s by %s at %s", id, labelPatch(p), user, clockNow.Format(timeFormat)))
}

// UpdateLabel, in one transaction and in the order of M3 design 3.6, locks
// the label's project, decides, reads whether the label has labels under
// it and the parent when a parent is given, reads the clock, then changes
// the fields given, by the caller at that time; it answers the label as
// stored, the time to the microsecond. Feature is renamed and recolored,
// moved under Bug, and given a sort order; Bug is renamed in another case,
// its own name; UI goes to the top, which reads neither its children nor a
// parent; and ops's Docs, of an archived project, changes as any other
// (3.19).
func TestUpdateLabel(t *testing.T) {
	for _, tt := range []struct {
		name    string
		id      uuid.UUID
		p       domain.LabelPatch
		changes func(l *domain.Label) // what p changes of the label as it was
	}{
		{"Feature renamed and recolored", webFeature, domain.LabelPatch{Name: ptr("Story"), Color: ptr("")},
			func(l *domain.Label) { l.Name, l.Color = "Story", "" }},
		{"Feature under Bug", webFeature, domain.LabelPatch{SetParent: true, ParentID: &webBug},
			func(l *domain.Label) { l.ParentID = &webBug }},
		{"Feature given a sort order", webFeature, domain.LabelPatch{SortOrder: ptr(-0.5)}, func(l *domain.Label) { l.SortOrder = -0.5 }},
		{"Bug renamed in another case", webBug, domain.LabelPatch{Name: ptr("BUG")}, func(l *domain.Label) { l.Name = "BUG" }},
		{"UI to the top", webUI, domain.LabelPatch{SetParent: true}, func(l *domain.Label) { l.ParentID = nil }},
		{"archived ops's Docs", opsDocs, domain.LabelPatch{Name: ptr("Guides")}, func(l *domain.Label) { l.Name = "Guides" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, l := newUpdateLabel()
			before := l.labels[tt.id]
			got, err := uc.Execute(as(bob), tt.id, tt.p)
			want := before
			tt.changes(&want)
			want.UpdatedAt = now
			if err != nil || labelJSON(t, got) != labelJSON(t, want) || labelJSON(t, l.labels[tt.id]) != labelJSON(t, want) {
				t.Errorf("Execute() = %+v, %v, stored %+v; want %+v", got, err, l.labels[tt.id], want)
			}
			if calls := labelUpdated(bob, tt.id, before.ProjectID, tt.p); !slices.Equal(f.log.calls, calls) {
				t.Errorf("calls\n%q\nwant\n%q", f.log.calls, calls)
			}
		})
	}
}

// Refusals, each in its place, and no label changed: no caller, and values
// the domain refuses, before the transaction; a label that is not there, a
// workspace or project deleted while its lock waited, a label deleted or
// moved to ops meanwhile, and a caller who does not see web, each
// project.label_not_found; a member, the Authorizer's 403. Then, after the
// decision, each a 422 parent_id not_allowed: Feature under itself, under
// UI, which is under Bug, under ops's Docs, or under a label that is not
// there; and Bug, which has UI under it, under Feature. Feature renamed to
// Bug's name in another case is the store's project.label_name_taken. The
// change answered for another label is the write's own error; the label
// read for another id is the shared path's, pinned once in lock_test.go.
func TestUpdateLabelRefuses(t *testing.T) {
	upTo := func(n int) []string { return labelLocked(webFeature, webID, bob, domain.ActionLabelUpdate)[:n] }
	rename := domain.LabelPatch{Name: ptr("Story")}
	under := func(parent uuid.UUID) domain.LabelPatch { return domain.LabelPatch{SetParent: true, ParentID: &parent} }
	parentRead := func(id, parent uuid.UUID) []string {
		return append(labelLocked(id, webID, bob, domain.ActionLabelUpdate), "HasChildren "+id.String(), "LabelByID "+parent.String())
	}
	parentRefused := shared.Invalid(shared.FieldError{Field: "parent_id", Code: shared.FieldNotAllowed})
	none := uuid.NewV7()
	for _, tt := range []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		p     domain.LabelPatch
		set   func(f *writeFixture, l *fakeLabels)
		want  error
		calls []string
	}{
		{"no caller", context.Background(), webFeature, rename, nil, shared.Unauthenticated(), nil},
		{"a blank name", as(bob), webFeature, domain.LabelPatch{Name: ptr("")}, nil,
			shared.Invalid(shared.FieldError{Field: "name", Code: shared.FieldTooShort}), nil},
		{"no label", as(bob), uuid.Nil(), rename, nil, domain.ErrLabelNotFound, []string{"Begin", "LabelByID " + uuid.Nil().String()}},
		{"acme deleted while its lock waited", as(bob), webFeature, rename, func(f *writeFixture, _ *fakeLabels) { f.workspaces.gone = true },
			domain.ErrLabelNotFound, upTo(3)},
		{"web deleted while its lock waited", as(bob), webFeature, rename, func(f *writeFixture, _ *fakeLabels) { f.store.deleted = true },
			domain.ErrLabelNotFound, upTo(4)},
		{"the label deleted while the locks waited", as(bob), webFeature, rename,
			func(f *writeFixture, _ *fakeLabels) { f.store.reread.gone = true }, domain.ErrLabelNotFound, upTo(5)},
		{"the label moved to ops", as(bob), webFeature, rename, func(f *writeFixture, _ *fakeLabels) { f.store.reread.project = opsID },
			domain.ErrLabelNotFound, upTo(5)},
		{"a caller who does not see web", as(erin), webFeature, rename, nil, domain.ErrLabelNotFound,
			labelLocked(webFeature, webID, erin, domain.ActionLabelUpdate)},
		{"a member", as(alice), webFeature, rename, nil, shared.Forbidden(), labelLocked(webFeature, webID, alice, domain.ActionLabelUpdate)},
		{"Feature under itself", as(bob), webFeature, under(webFeature), nil, parentRefused, parentRead(webFeature, webFeature)},
		{"Feature under UI, which is under Bug", as(bob), webFeature, under(webUI), nil, parentRefused, parentRead(webFeature, webUI)},
		{"Feature under ops's Docs", as(bob), webFeature, under(opsDocs), nil, parentRefused, parentRead(webFeature, opsDocs)},
		{"Feature under a label that is not there", as(bob), webFeature, under(none), nil, parentRefused, parentRead(webFeature, none)},
		{"Bug, with UI under it, under Feature", as(bob), webBug, under(webFeature), nil, parentRefused, parentRead(webBug, webFeature)},
		{"Feature renamed to Bug's name in another case", as(bob), webFeature, domain.LabelPatch{Name: ptr("bug")}, nil,
			domain.ErrLabelNameTaken, labelUpdated(bob, webFeature, webID, domain.LabelPatch{Name: ptr("bug")})},
		{"the change answered for another label", as(bob), webFeature, rename,
			func(_ *writeFixture, l *fakeLabels) { l.changedAs = uuid.NewV7() }, nil, labelUpdated(bob, webFeature, webID, rename)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, l := newUpdateLabel()
			if tt.set != nil {
				tt.set(f, l)
			}
			before := maps.Clone(l.labels)
			_, err := uc.Execute(tt.ctx, tt.id, tt.p)
			outcome{tt.name, tt.want, tt.calls}.check(t, err, f)
			if tt.name != "the change answered for another label" && !maps.Equal(l.labels, before) {
				t.Errorf("the labels after the refusal: %v; want them as they were, %v", l.labels, before)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after: Feature moved under Bug, so that the
// children and the parent are read.
func TestUpdateLabelReturnsEachFailure(t *testing.T) {
	move := domain.LabelPatch{SetParent: true, ParentID: &webBug}
	all := labelUpdated(bob, webFeature, webID, move)
	fail := func(method string) func(f *writeFixture, _ *fakeLabels) {
		return func(f *writeFixture, _ *fakeLabels) { f.store.errs = map[string]error{method: errDisk} }
	}
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture, l *fakeLabels)
		calls int // how many of the change's calls ran
	}{
		{"the label's read", fail("LabelByID"), 2},
		{"the workspace's lock", func(f *writeFixture, _ *fakeLabels) { f.workspaces.err = errDisk }, 3},
		{"the project's lock", fail("LockProject"), 4},
		{"the read under the locks", func(f *writeFixture, _ *fakeLabels) { f.store.reread.err = errDisk }, 5},
		{"the decision", func(f *writeFixture, _ *fakeLabels) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, 6},
		{"the children", fail("HasChildren"), 7},
		{"the parent", func(_ *writeFixture, l *fakeLabels) { l.failsFor = webBug }, 8},
		{"the change", fail("UpdateLabel"), 10},
		{"the commit", func(f *writeFixture, _ *fakeLabels) { f.tx.commitErr = errDisk }, 10},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, l := newUpdateLabel()
			tt.fail(f, l)
			_, err := uc.Execute(as(bob), webFeature, move)
			outcome{tt.name, errDisk, all[:tt.calls]}.check(t, err, f)
		})
	}
}
````

`server/internal/modules/project/app/clock_test.go`（修改，1 处）：

````old server/internal/modules/project/app/clock_test.go
		}, labelCreated(bob, webID, underBug, 95535)},
````
````new server/internal/modules/project/app/clock_test.go
		}, labelCreated(bob, webID, underBug, 95535)},
		{"updateLabel", func() ([]string, error) {
			uc, f, _ := newUpdateLabel()
			_, err := uc.Execute(as(bob), webFeature, domain.LabelPatch{SetParent: true, ParentID: &webBug})
			return f.log.calls, err
		}, labelUpdated(bob, webFeature, webID, domain.LabelPatch{SetParent: true, ParentID: &webBug})},
````

- [ ] **Step 4: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/app/clock_test.go server/internal/modules/project/app/fakes_label_test.go server/internal/modules/project/app/fakes_write_test.go server/internal/modules/project/app/label_ports.go server/internal/modules/project/app/update_label.go server/internal/modules/project/app/update_label_test.go server/internal/modules/project/domain/actions.go
```
```bash
git commit -m "feat(M3/P7b): changing a label takes the shared row path, then checks a parent given

updateLabel checks the fields given first, then reads the label for its
project, locks the workspace FOR SHARE and the project FOR NO KEY
UPDATE, reads the label again and decides label.update, through the
path every write on a row under a project takes. A parent given is
read under the lock, with whether the label has labels under it: a
label of the project at the top, not the label itself, and none for a
label with labels under it. A null parent moves the label to the top.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**（spec 附录 A；`mutants_p7b.py` 的编号；"层"是它被发现的每一层，端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `s2-update-children` | `updateLabel` 越过 `HasChildren` 的失败 | `TestUpdateLabelReturnsEachFailure` | 单元（按性质只在单元一层：真实的存储在这一步不失败） |
| `s2-update-update` | `updateLabel` 吞掉 `UpdateLabel` 的失败 | `TestUpdateLabelRefuses`、`TestUpdateLabelReturnsEachFailure` | 单元（按性质只在单元一层：真实的存储在这一步不失败） |
| `s22-updated-id` | `updateLabel` 收下回答为另一个标签的修改 | `TestUpdateLabelRefuses` | 单元（按性质只在单元一层：真实的存储回答不了别的键（裁定 S6）） |
| `p-label-404-state` | 标签的 404 答 `project.state_not_found`（`labelWrite` 的 `notFound`） | `TestUpdateLabelRefuses`、`TestDeleteLabelRefuses`（Task 8 起）、`TestBodiesThatBreakTheStructureAnswer400`（Task 7 起）、`TestPermissionMatrix`（Task 7 起）、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`（Task 10 起）、P7（Task 11 起） | 单元；组合；端到端 |
| `c-update-early` | `updateLabel` 在锁之前读时钟 | `TestEachWriteReadsTheClockUnderItsLock`、`TestUpdateLabel`、`TestUpdateLabelRefuses`、`TestUpdateLabelReturnsEachFailure`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 10 起）、`TestLabelWritesOnOneProjectSerialize`（Task 10 起） | 单元；组合 |
| `r-update-children-unread` | `updateLabel` 不看这个标签有没有子标签 | `TestUpdateLabelRefuses`、`TestPermissionMatrix`（Task 7 起）、`TestEachLabelWriteChangesItsRowsAlone`（Task 10 起）、`TestLabelWritesOnOneProjectSerialize`（Task 10 起）、P7（Task 11 起） | 单元；组合；端到端 |
| `r-update-unchecked` | `updateLabel` 收下任何父标签（不调用 `checkParent`） | `TestEachWriteReadsTheClockUnderItsLock`、`TestUpdateLabel`、`TestUpdateLabelRefuses`、`TestUpdateLabelReturnsEachFailure`、`TestPermissionMatrix`（Task 7 起）、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`（Task 7 起）、`TestEachLabelWriteChangesItsRowsAlone`（Task 10 起）、`TestLabelWritesOnOneProjectSerialize`（Task 10 起）、P7（Task 11 起） | 单元；组合；端到端 |
| `s21-patch-checked-late` | `updateLabel` 在找到标签之后才查值 | `TestUpdateLabelRefuses`、`TestPermissionMatrix`（Task 7 起） | 单元；组合 |
| `s25-update-parent-unlocked` | `updateLabel` 在锁之前读子标签和父标签并检查 | `TestPermissionMatrix`（Task 7 起）、`TestLabelWritesOnOneProjectSerialize`（Task 10 起） | 组合 |
| `s5-update-members` | 规则表：`label.update` 给成员 | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix`（Task 7 起）、P7（Task 11 起） | 单元；组合；端到端 |

---

### Task 7: `updateLabel` 的接口：契约、处理函数、文案、矩阵和组合的测试

**Files:**
- Modify: `api/modules/project.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_columns_test.go`、`server/internal/bootstrap/permission_matrix_labels_test.go`、`server/internal/bootstrap/permission_matrix_targets_test.go`、`server/internal/bootstrap/project_connection_test.go`、`server/internal/bootstrap/project_write_locks_test.go`、`server/internal/bootstrap/project_writes_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/labels.go`、`server/internal/modules/project/adapter/http/labels_test.go`、`server/internal/modules/project/module.go`、`web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.7、2.10）：
  - 契约：`PATCH /api/v0/labels/{label_id}`（`updateLabel`，200 `Label`；`x-problem-codes: [validation_failed, project.label_not_found, forbidden, project.label_name_taken]`）、参数 `LabelID`、`LabelUpdate`（四个字段都可选，只有 `parent_id` 可以为 null：移到顶层）；`api/openapi.yaml` 的码 `project.label_not_found`（404）。
  - 处理函数 `UpdateLabel`：给了的字段交给用例，null 的父标签是"给了、为顶层"（`SetParent: true, ParentID: nil`）；`UseCases.UpdateLabel`；`module.go` 接上 `app.NewUpdateLabel`。
  - 前端：`project.label_not_found` 的文案。
  - 矩阵：`updateLabel` 的 5 行（Feature 改名 Story：`renamesTheLabel`；名称被占用；有子标签的 Bug 设父标签：422；值被拒绝：空的名称，每一列都是 422 `name too_short`，在看项目之前；已归档的项目）；`permission_matrix_targets_test.go` 的 `{label_id}`；`permission_matrix_columns_test.go` 的瞄准标签行的列。
  - 组合：`rowPaths` 加 `/api/v0/labels/`；`labelNamed`；`writesOnAProject` 加 `updateLabel`（Task 5 建的 QA 改名 Checked）；盖戳加 `updateLabel`（`labelID`）；连接加三个 PATCH（Bug 移到 UI 下被拒绝，UI 改名 Widgets 到顶层，再放回 Bug 下）。

**Tests:**
- 处理函数：`TestUpdateLabel`（各字段、null 的父标签、空的补丁）、`TestUpdateLabelHoldsTheBodyToItsStructure`（多余的字段、类型不对、父标签之外的 null、不是 id 的父标签：400 `bad_request`，不调用用例）、`TestUpdateLabelRefusals`（422、404、403、409、500）。
- 矩阵：`TestPermissionMatrix` 的 5 个 `updateLabel` 行；`TestThePermissionMatrixCoversEveryOperation`；`TestMatrixViolationsCatchesEachColumnGap`（瞄准标签行的格子必须是那一列的项目的标签）。
- 组合：同 Task 5 的四个测试，加标签的行写。

- [ ] **Step 1: 契约**

`api/modules/project.yaml`（修改，3 处）：

````old api/modules/project.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/me/projects/{project_id}/preferences:
````
````new api/modules/project.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/labels/{label_id}:
    parameters:
      - $ref: '#/components/parameters/LabelID'
    patch:
      operationId: updateLabel
      tags: [project]
      summary: Change a label
      description: >-
        For the project's admins, and its members who are the workspace's
        admins; an archived project's labels change as any other's. The
        fields given change and the others stay; the values follow
        createLabel's rules, and the sort order is any number
        (validation_failed), checked before the label is looked at. A label
        that does not exist or is deleted, and a label whose project the
        caller does not see, answer project.label_not_found; a caller who
        sees the project but may not change its labels, forbidden. A parent
        given is a label of the project at the top, not the label itself,
        and a label with labels under it takes none: labels have two levels
        (parent_id not_allowed); a null parent moves the label to the top.
        The name may not be another undeleted label's of the project, in any
        case (project.label_name_taken). The role is decided after the
        workspace and project rows are locked, and the parent is checked
        after it.
      security: [{bearer: []}]
      x-problem-codes: [validation_failed, project.label_not_found, forbidden, project.label_name_taken]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/LabelUpdate'
      responses:
        '200':
          description: The label as changed.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Label'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/me/projects/{project_id}/preferences:
````

````old api/modules/project.yaml
      description: A project membership's id (ProjectMember.id), not the member's account id.
````
````new api/modules/project.yaml
      description: A project membership's id (ProjectMember.id), not the member's account id.
      schema:
        type: string
        format: uuid
    LabelID:
      name: label_id
      in: path
      required: true
      description: A label's id (Label.id).
````

````old api/modules/project.yaml
          description: The label's place among the project's labels; after the others when left out.
          type: number
````
````new api/modules/project.yaml
          description: The label's place among the project's labels; after the others when left out.
          type: number
    LabelUpdate:
      description: >-
        Changes the fields it names; a field left out keeps its value. Only
        parent_id can be null, which moves the label to the top.
      type: object
      additionalProperties: false
      properties:
        name:
          description: 1–255 characters, not blank; another undeleted label of the project may not have it, in any case.
          type: string
        color:
          description: At most 255 characters; empty for none.
          type: string
        parent_id:
          description: A label of the project at the top, not this one, which the label goes under; null for the top.
          type: [string, 'null']
          format: uuid
        sort_order:
          description: The label's place among the project's labels, the lowest first.
          type: number
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1labels'
````
````new api/openapi.yaml
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1projects~1{project_id}~1labels'
  /api/v0/labels/{label_id}:
    $ref: 'modules/project.yaml#/paths/~1api~1v0~1labels~1{label_id}'
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `ac72c911c49efa46c06df2b35dc22d8e91b0a007221cf6d80f8014c2add928a0` | 2905 | `api/dist/openapi.yaml` |
| `9059accc72257d17cffc8b84cee51a8d1179e6e308e6eb360a046299dcd9aacf` | 86 | `server/internal/modules/project/adapter/http/gen/bodyshape.gen.go` |
| `e7dd9ccff491058ad7b9cace9065d08a1c352df0ce3403d6b2b6d93f8d4bd0ca` | 3299 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `a17594b28aa10eb0802946cf018da4b63561663247691f09de93728d59766cc8` | 3215 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/bodyshape.gen.go server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 2: 处理函数、接线和文案**

`server/internal/modules/project/adapter/http/labels.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/labels.go
}

// label is l as the API shows it: its parent null at the top.
````
````new server/internal/modules/project/adapter/http/labels.go
}

// UpdateLabel serves PATCH /api/v0/labels/{label_id}: the fields given go
// to the use case, a null parent as one given as none.
func (h handler) UpdateLabel(ctx context.Context, req gen.UpdateLabelRequestObject) (gen.UpdateLabelResponseObject, error) {
	p := domain.LabelPatch{Name: req.Body.Name, Color: req.Body.Color, SortOrder: req.Body.SortOrder}
	p.SetParent, p.ParentID = named(req.Body.ParentID)
	l, err := h.uc.UpdateLabel.Execute(ctx, req.LabelID, p)
	if err != nil {
		return nil, err
	}
	return gen.UpdateLabel200JSONResponse(label(l)), nil
}

// label is l as the API shows it: its parent null at the top.
````

`server/internal/modules/project/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// UpdateLabelUseCase is app.UpdateLabel.
type UpdateLabelUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, p domain.LabelPatch) (domain.Label, error)
}

// UseCases are the use cases behind the module's operations.
````

````old server/internal/modules/project/adapter/http/handler.go
	CreateLabel         CreateLabelUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
	CreateLabel         CreateLabelUseCase
	UpdateLabel         UpdateLabelUseCase
````

`server/internal/modules/project/adapter/http/labels_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/labels_test.go
}

// web's labels as the use cases answer them: Bug at the top, with a color;
````
````new server/internal/modules/project/adapter/http/labels_test.go
}

// fakeUpdateLabel is updateLabel: each call is recorded as "caller id",
// with what it got; it answers answer, or err.
type fakeUpdateLabel struct {
	calls  []string
	got    []domain.LabelPatch
	answer domain.Label
	err    error
}

func (f *fakeUpdateLabel) Execute(ctx context.Context, id uuid.UUID, p domain.LabelPatch) (domain.Label, error) {
	f.calls = append(f.calls, caller(ctx)+" "+id.String())
	f.got = append(f.got, p)
	return f.answer, f.err
}

// web's labels as the use cases answer them: Bug at the top, with a color;
````

````old server/internal/modules/project/adapter/http/labels_test.go
		`"detail":"A label of the project has this name, in this case or another."}`
````
````new server/internal/modules/project/adapter/http/labels_test.go
		`"detail":"A label of the project has this name, in this case or another."}`
	labelNotFoundJSON = `{"status":404,"code":"project.label_not_found","title":"Not Found",` +
		`"detail":"The label does not exist, or you cannot see its project."}`
````

````old server/internal/modules/project/adapter/http/labels_test.go
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

````
````new server/internal/modules/project/adapter/http/labels_test.go
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// PATCH goes to the use case for the caller and the path's label, with the
// fields the body gives and nil for each it does not; a parent given as
// null is set, to none; the answer is 200 with the label the use case
// answers.
func TestUpdateLabel(t *testing.T) {
	path := "/api/v0/labels/" + webUILabel.ID.String()
	for _, tt := range []struct {
		body string
		want domain.LabelPatch
	}{
		{`{}`, domain.LabelPatch{}},
		{`{"name":"UI","color":"","parent_id":"0199a2b4-0000-7000-8000-0000000000d1","sort_order":75535.5}`,
			domain.LabelPatch{Name: ptr("UI"), Color: ptr(""), SetParent: true, ParentID: &webBugLabel.ID, SortOrder: ptr(75535.5)}},
		{`{"parent_id":null}`, domain.LabelPatch{SetParent: true}},
	} {
		update := &fakeUpdateLabel{answer: webUILabel}
		h := newServer(t, fakes{updateLabel: update})
		if res, body := do(t, h, request(http.MethodPatch, path, "alice", tt.body)); res.StatusCode != http.StatusOK || body != webUILabelJSON+"\n" {
			t.Errorf("PATCH %s = %d %s, want 200 %s", tt.body, res.StatusCode, body, webUILabelJSON)
		}
		if want := []string{"alice " + webUILabel.ID.String()}; !slices.Equal(update.calls, want) || !reflect.DeepEqual(update.got,
			[]domain.LabelPatch{tt.want}) {
			t.Errorf("PATCH %s: calls %q with %+v; want %q with %+v", tt.body, update.calls, update.got, want, tt.want)
		}
	}
}

// A body with a field it may not have, a field of another type, a null
// other than the parent's, or a parent that is no id: refused as
// bad_request before the use case.
func TestUpdateLabelHoldsTheBodyToItsStructure(t *testing.T) {
	update := &fakeUpdateLabel{}
	h := newServer(t, fakes{updateLabel: update})
	for _, body := range []string{`{"project_id":"0199a2b4-0000-7000-8000-0000000000a1"}`, `{"name":1}`, `{"sort_order":"1"}`, `{"name":null}`,
		`{"color":null}`, `{"sort_order":null}`, `{"parent_id":"Bug"}`} {
		res, got := do(t, h, request(http.MethodPatch, "/api/v0/labels/"+webUILabel.ID.String(), "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("PATCH %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(update.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", update.calls)
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestUpdateLabelRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"a parent refused", parentRefused, http.StatusUnprocessableEntity, parentRefusedJSON},
		{"no label", domain.ErrLabelNotFound, http.StatusNotFound, labelNotFoundJSON},
		{"a project member", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"a name taken", domain.ErrLabelNameTaken, http.StatusConflict, labelNameTakenJSON},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{updateLabel: &fakeUpdateLabel{err: tt.err}})
		res, body := do(t, h, request(http.MethodPatch, "/api/v0/labels/"+webUILabel.ID.String(), "alice", `{"name":"UI"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: PATCH = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	createLabel         *fakeCreateLabel
````
````new server/internal/modules/project/adapter/http/handler_test.go
	createLabel         *fakeCreateLabel
	updateLabel         *fakeUpdateLabel
````

````old server/internal/modules/project/adapter/http/handler_test.go
		f.createLabel = &fakeCreateLabel{}
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		f.createLabel = &fakeCreateLabel{}
	}
	if f.updateLabel == nil {
		f.updateLabel = &fakeUpdateLabel{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
		MarkDefaultState: f.markDefault, ListWorkspaceStates: f.listWorkspaceStates, CreateLabel: f.createLabel})
````
````new server/internal/modules/project/adapter/http/handler_test.go
		MarkDefaultState: f.markDefault, ListWorkspaceStates: f.listWorkspaceStates, CreateLabel: f.createLabel, UpdateLabel: f.updateLabel})
````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// workspace's projects one is a member of, creating a project's labels,
// carries out the workspace module's cascades on the projects
// (ProjectCascade), and offers the access module its reads of a project
// (ProjectAccess) and the workspace module its count of an account's ended
// project memberships (ProjectMembershipCounts).
````
````new server/internal/modules/project/module.go
// workspace's projects one is a member of, creating and changing a
// project's labels, carries out the workspace module's cascades on the
// projects (ProjectCascade), and offers the access module its reads of a
// project (ProjectAccess) and the workspace module its count of an
// account's ended project memberships (ProjectMembershipCounts).
````

````old server/internal/modules/project/module.go
		CreateLabel:         app.NewCreateLabel(locks, store, d.Tx, d.Clock),
````
````new server/internal/modules/project/module.go
		CreateLabel:         app.NewCreateLabel(locks, store, d.Tx, d.Clock),
		UpdateLabel:         app.NewUpdateLabel(locks, store, d.Tx, d.Clock),
````

`web/apps/web/helpers/authentication.helper.ts`（修改，1 处）：

````old web/apps/web/helpers/authentication.helper.ts
  "project.label_name_taken": "auth.errors.project_label_name_taken",
````
````new web/apps/web/helpers/authentication.helper.ts
  "project.label_name_taken": "auth.errors.project_label_name_taken",
  "project.label_not_found": "auth.errors.project_label_not_found",
````

`web/packages/i18n/src/locales/en/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/auth.json
      "project_label_name_taken": "A label of this project already has this name, in this case or another.",
````
````new web/packages/i18n/src/locales/en/auth.json
      "project_label_name_taken": "A label of this project already has this name, in this case or another.",
      "project_label_not_found": "The label does not exist, or you cannot see its project.",
````

`web/packages/i18n/src/locales/zh-CN/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/auth.json
      "project_label_name_taken": "这个项目里已有同名的标签（不区分大小写）。",
````
````new web/packages/i18n/src/locales/zh-CN/auth.json
      "project_label_name_taken": "这个项目里已有同名的标签（不区分大小写）。",
      "project_label_not_found": "标签不存在，或者你看不到它所在的项目。",
````

- [ ] **Step 3: 矩阵**

`server/internal/bootstrap/permission_matrix_labels_test.go`（修改，4 处）：

````old server/internal/bootstrap/permission_matrix_labels_test.go
// creating them, under the project of each column; and the labels they
// rest on, matrixLabels, which prepareMatrix seeds in each project through
// the project store (labels) and reads back (seededLabels). They are here
// rather than in permission_matrix_seed_test.go, which has no room for
// them.
````
````new server/internal/bootstrap/permission_matrix_labels_test.go
// creating them, under the project of each column, and the writes on one,
// naming it by its id (/labels/{label_id}) among its column's project's
// seeded labels; and the labels they rest on, matrixLabels, which
// prepareMatrix seeds in each project through the project store (labels)
// and reads back (seededLabels). They are here rather than in
// permission_matrix_seed_test.go, which has no room for them.
````

````old server/internal/bootstrap/permission_matrix_labels_test.go
var cellLabelNameTaken = cell{http.StatusConflict, "project.label_name_taken"}
````
````new server/internal/bootstrap/permission_matrix_labels_test.go
var (
	cellLabelNameTaken = cell{http.StatusConflict, "project.label_name_taken"}
	cellLabelNotFound  = cell{http.StatusNotFound, "project.label_not_found"}
)

// ofLabel are the cells of a row of a write on a label: the answers of PA,
// PM, PG, PM+WA, WA- and WM-公, and project.label_not_found for the
// columns that do not see their project: WM-私, WG-, P-前 and X, whose
// label in gone's project is deleted with it.
func ofLabel(pa, pm, pg, pmwa, wa, wm cell) map[caller]cell {
	return map[caller]cell{callerProjectAdmin: pa, callerProjectMember: pm, callerProjectGuest: pg, callerMemberAndAdmin: pmwa,
		callerAdminOnly: wa, callerMemberPublic: wm, callerMemberPrivate: cellLabelNotFound, callerGuestOnly: cellLabelNotFound,
		callerBefore: cellLabelNotFound, callerNever: cellLabelNotFound, callerRemoved: cellLabelNotFound, callerDeleted: cellLabelNotFound}
}

// ofArchivedLabel are the cells of a row of a write on a label of the
// archived project: the answers of PA and of the workspace's member, who
// sees the project, and project.label_not_found for X, who does not.
func ofArchivedLabel(pa, wm cell) map[caller]cell {
	return map[caller]cell{callerArchivedAdmin: pa, callerArchivedMember: wm, callerArchivedNever: cellLabelNotFound}
}

// toLabel is the request of a row whose callers each send method, with
// body, to the label name of their column's project.
func toLabel(method, name, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return method, "/api/v0/labels/" + s.label(projectOf(c), name).String(), body
	}
}
````

````old server/internal/bootstrap/permission_matrix_labels_test.go
			cells: ofArchived(cellCreated, cellForbidden), check: createsTheLabel},
````
````new server/internal/bootstrap/permission_matrix_labels_test.go
			cells: ofArchived(cellCreated, cellForbidden), check: createsTheLabel},
		// As createLabel: Feature renamed.
		{op: "updateLabel", write: true, columns: projectColumns, request: toLabel(http.MethodPatch, "Feature", `{"name":"Story"}`),
			cells: ofLabel(cellOK, cellForbidden, cellForbidden, cellOK, cellForbidden, cellForbidden), check: renamesTheLabel},
		{op: "updateLabel", variant: "a name taken", write: true, columns: projectColumns,
			request: toLabel(http.MethodPatch, "Feature", `{"name":"bug"}`),
			cells:   ofLabel(cellLabelNameTaken, cellForbidden, cellForbidden, cellLabelNameTaken, cellForbidden, cellForbidden)},
		// Bug, which has UI under it, under Feature: the parent is checked
		// after the decision.
		{op: "updateLabel", variant: "a label with labels under it given a parent", write: true, columns: projectColumns,
			request: func(c caller, s seeded) (string, string, string) {
				return toLabel(http.MethodPatch, "Bug", `{"parent_id":"`+s.label(projectOf(c), "Feature").String()+`"}`)(c, s)
			},
			cells:   ofLabel(cellValidationFailed, cellForbidden, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "parent_id not_allowed"},
		// A value refused before the label is looked at: the same 422 in
		// every column.
		{op: "updateLabel", variant: "a value refused", write: true, columns: projectColumns,
			request: toLabel(http.MethodPatch, "Feature", `{"name":""}`), cells: func() map[caller]cell {
				cells := map[caller]cell{}
				for _, c := range projectColumns {
					cells[c] = cellValidationFailed
				}
				return cells
			}(), refusal: "name too_short"},
		{op: "updateLabel", variant: "archived", write: true, columns: archivedColumns,
			request: toLabel(http.MethodPatch, "Feature", `{"name":"Story"}`), cells: ofArchivedLabel(cellOK, cellForbidden),
			check: renamesTheLabel},
````

````old server/internal/bootstrap/permission_matrix_labels_test.go
		t.Errorf("%s creates %s; want QA in %s, at the top, at 95535", c, answer, projectOf(c))
````
````new server/internal/bootstrap/permission_matrix_labels_test.go
		t.Errorf("%s creates %s; want QA in %s, at the top, at 95535", c, answer, projectOf(c))
	}
}

// renamesTheLabel: the column's project's Feature, at the top, renamed
// Story, as stored.
func renamesTheLabel(t *testing.T, c caller, s seeded, answer string) {
	var l struct {
		ID        uuid.UUID  `json:"id"`
		ProjectID uuid.UUID  `json:"project_id"`
		ParentID  *uuid.UUID `json:"parent_id"`
		Name      string     `json:"name"`
		SortOrder float64    `json:"sort_order"`
	}
	decodeAnswer(t, answer, &l)
	if l.ID != s.label(projectOf(c), "Feature") || l.ProjectID != s.project(projectOf(c)) || l.ParentID != nil || l.Name != "Story" ||
		l.SortOrder != 85535 {
		t.Errorf("%s renames %s; want %s's Feature, at the top, at 85535, named Story", c, answer, projectOf(c))
````

`server/internal/bootstrap/permission_matrix_targets_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_targets_test.go
	"{state_id}":          {"state", func(s seeded) map[string]uuid.UUID { return s.states }},
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
	"{state_id}":          {"state", func(s seeded) map[string]uuid.UUID { return s.states }},
	"{label_id}":          {"label", func(s seeded) map[string]uuid.UUID { return s.labels }},
````

````old server/internal/bootstrap/permission_matrix_targets_test.go
// under a project (underProject: a project membership, a state) one seeded
// in projectOf(c), each from a column of a project table (projectTables):
// a project's operation in a workspace-level row, or the only admin's,
// would leave the project level's own columns unasked.
````
````new server/internal/bootstrap/permission_matrix_targets_test.go
// under a project (underProject: a project membership, a state, a label)
// one seeded in projectOf(c), each from a column of a project table
// (projectTables): a project's operation in a workspace-level row, or the
// only admin's, would leave the project level's own columns unasked.
````

`server/internal/bootstrap/permission_matrix_columns_test.go`（修改，2 处）：

````old server/internal/bootstrap/permission_matrix_columns_test.go
	// membership, a state) is one seeded in the column's project, from a
	// column of a project table.
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
	// membership, a state, a label) is one seeded in the column's project,
	// from a column of a project table.
````

````old server/internal/bootstrap/permission_matrix_columns_test.go
			func(c caller, s seeded) uuid.UUID { return s.state(projectOf(c), "Todo") }, s.state("acme/public", "Todo")},
````
````new server/internal/bootstrap/permission_matrix_columns_test.go
			func(c caller, s seeded) uuid.UUID { return s.state(projectOf(c), "Todo") }, s.state("acme/public", "Todo")},
		{apitest.Operation{ID: "updateLabel", Tags: []string{"project"}, Method: http.MethodPatch, Path: "/api/v0/labels/{label_id}"}, "label",
			func(c caller, s seeded) uuid.UUID { return s.label(projectOf(c), "Bug") }, s.label("acme/public", "Bug")},
````

- [ ] **Step 4: 组合的测试**

`server/internal/bootstrap/project_write_locks_test.go`（修改，6 处）：

````old server/internal/bootstrap/project_write_locks_test.go
// (member_id), a state's name for a state (name).
````
````new server/internal/bootstrap/project_write_locks_test.go
// (member_id), a state's or a label's name for a state or a label (name).
````

````old server/internal/bootstrap/project_write_locks_test.go
func stateNamed(name string) underRow { return underRow{"states", "name", [2]string{name, name}} }

````
````new server/internal/bootstrap/project_write_locks_test.go
func stateNamed(name string) underRow { return underRow{"states", "name", [2]string{name, name}} }

// labelNamed is the label name of each phase's project.
func labelNamed(name string) underRow { return underRow{"labels", "name", [2]string{name, name}} }

````

````old server/internal/bootstrap/project_write_locks_test.go
var rowPaths = map[string]string{"/api/v0/project-members/": "{project_member_id}", "/api/v0/states/": "{state_id}"}
````
````new server/internal/bootstrap/project_write_locks_test.go
var rowPaths = map[string]string{"/api/v0/project-members/": "{project_member_id}", "/api/v0/states/": "{state_id}",
	"/api/v0/labels/": "{label_id}"}
````

````old server/internal/bootstrap/project_write_locks_test.go
	{op: "createLabel", method: http.MethodPost, path: "/api/v0/projects/%s/labels", body: `{"name":"QA"}`, want: http.StatusCreated},
````
````new server/internal/bootstrap/project_write_locks_test.go
	{op: "createLabel", method: http.MethodPost, path: "/api/v0/projects/%s/labels", body: `{"name":"QA"}`, want: http.StatusCreated},
	// The label made before, renamed.
	{op: "updateLabel", method: http.MethodPatch, path: "/api/v0/labels/%s", body: `{"name":"Checked"}`, want: http.StatusOK, row: labelNamed("QA")},
````

````old server/internal/bootstrap/project_write_locks_test.go
// /states/{state_id}) whose row names a project table's columns; a column
// set of a row's own counts only once it is listed in projectTables, so a
// new table of the project level goes there, one of the workspace level in
// workspaceLevelTables, and neither into matrixTables alone
// (TestEachMatrixTableIsOfOneLevel); not a workspace's write that the only
// admin's table asks (leaveWorkspace).
````
````new server/internal/bootstrap/project_write_locks_test.go
// /states/{state_id}, P7b's /labels/{label_id}) whose row names a project
// table's columns; a column set of a row's own counts only once it is
// listed in projectTables, so a new table of the project level goes there,
// one of the workspace level in workspaceLevelTables, and neither into
// matrixTables alone (TestEachMatrixTableIsOfOneLevel); not a workspace's
// write that the only admin's table asks (leaveWorkspace).
````

````old server/internal/bootstrap/project_write_locks_test.go
//     membership or a state: a FOR UPDATE NOWAIT of each succeeds.
````
````new server/internal/bootstrap/project_write_locks_test.go
//     membership, a state or a label: a FOR UPDATE NOWAIT of each succeeds.
````

`server/internal/bootstrap/project_writes_test.go`（修改，5 处）：

````old server/internal/bootstrap/project_writes_test.go
// Web, deletes the state, makes Done the default and creates a label,
// unarchives Web, and leaves it. Each row the write writes again is first
// made bob's, as last written by him, and checked so: a write that kept its
// row's writer would pass for alice's otherwise, she having made it.
````
````new server/internal/bootstrap/project_writes_test.go
// Web, deletes the state, makes Done the default, creates a label and
// renames it, unarchives Web, and leaves it. Each row the write writes
// again is first made bob's, as last written by him, and checked so: a
// write that kept its row's writer would pass for alice's otherwise, she
// having made it.
````

````old server/internal/bootstrap/project_writes_test.go
	// membership is the id of user's membership of Web; state, of Web's
	// state name.
````
````new server/internal/bootstrap/project_writes_test.go
	// membership is the id of user's membership of Web; state and label, of
	// Web's state or label name.
````

````old server/internal/bootstrap/project_writes_test.go
	state := func(name string) func() uuid.UUID { return func() uuid.UUID { return stateID(t, pool, web, name) } }
````
````new server/internal/bootstrap/project_writes_test.go
	state := func(name string) func() uuid.UUID { return func() uuid.UUID { return stateID(t, pool, web, name) } }
	label := func(name string) func() uuid.UUID { return func() uuid.UUID { return labelID(t, pool, web, name) } }
````

````old server/internal/bootstrap/project_writes_test.go
			"SELECT created_at, created_by_id = $2 AND updated_by_id = $2 AND updated_at = created_at FROM labels WHERE project_id = $1 AND name = 'QA'",
			0, 1},
````
````new server/internal/bootstrap/project_writes_test.go
			"SELECT created_at, created_by_id = $2 AND updated_by_id = $2 AND updated_at = created_at FROM labels WHERE project_id = $1 AND name = 'QA'",
			0, 1},
		// The label QA, renamed Checked.
		{"updateLabel", http.MethodPatch, "/api/v0/labels/%s", `{"name":"Checked"}`, label("QA"), http.StatusOK,
			bobs("labels", "project_id = $1 AND name = 'QA'"),
			"SELECT updated_at, updated_by_id = $2 AND name = 'Checked' FROM labels WHERE project_id = $1 AND name IN ('QA', 'Checked')", 1, 1},
````

````old server/internal/bootstrap/project_writes_test.go
		t.Fatalf("the state %s of %s: %v", name, project, err)
````
````new server/internal/bootstrap/project_writes_test.go
		t.Fatalf("the state %s of %s: %v", name, project, err)
	}
	return id
}

// labelID is the id of project's undeleted label name.
func labelID(t *testing.T, pool *pgxpool.Pool, project uuid.UUID, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT id FROM labels WHERE project_id = $1 AND name = $2 AND deleted_at IS NULL", project,
		name).Scan(&id); err != nil {
		t.Fatalf("the label %s of %s: %v", name, project, err)
````

`server/internal/bootstrap/project_connection_test.go`（修改，2 处）：

````old server/internal/bootstrap/project_connection_test.go
// read refuses (422: labels have two levels); carol, an admin, leaves it;
// alice deletes both projects.
````
````new server/internal/bootstrap/project_connection_test.go
// read refuses (422: labels have two levels); she moves Bug under UI,
// which the same read refuses, renames UI Widgets at the top, and moves it
// back under Bug; carol, an admin, leaves it; alice deletes both projects.
````

````old server/internal/bootstrap/project_connection_test.go
	send(http.MethodPost, web+"/labels", r.alice, `{"name":"Icons","parent_id":"`+ui.ID.String()+`"}`, http.StatusUnprocessableEntity)
````
````new server/internal/bootstrap/project_connection_test.go
	send(http.MethodPost, web+"/labels", r.alice, `{"name":"Icons","parent_id":"`+ui.ID.String()+`"}`, http.StatusUnprocessableEntity)
	send(http.MethodPatch, "/api/v0/labels/"+bug.ID.String(), r.alice, `{"parent_id":"`+ui.ID.String()+`"}`, http.StatusUnprocessableEntity)
	send(http.MethodPatch, "/api/v0/labels/"+ui.ID.String(), r.alice, `{"name":"Widgets","parent_id":null}`, http.StatusOK)
	send(http.MethodPatch, "/api/v0/labels/"+ui.ID.String(), r.alice, `{"parent_id":"`+bug.ID.String()+`"}`, http.StatusOK)
````

- [ ] **Step 5: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation$|TestEveryColumnCallsAsARegisteredAccount$|TestMatrixViolationsCatchesEachColumnGap$|TestWritesOnAProjectAreEachShape$|TestEachWriteOnAProjectSharesItsWorkspaceFirst$|TestTheWritesOnAProjectStampTheirRequest$|TestTheWritesOnAProjectRunOnTheirTransactionsConnection$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

- [ ] **Step 6: 提交**

```bash
git add api/modules/project.yaml api/openapi.yaml server/internal/bootstrap/permission_matrix_columns_test.go server/internal/bootstrap/permission_matrix_labels_test.go server/internal/bootstrap/permission_matrix_targets_test.go server/internal/bootstrap/project_connection_test.go server/internal/bootstrap/project_write_locks_test.go server/internal/bootstrap/project_writes_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/labels.go server/internal/modules/project/adapter/http/labels_test.go server/internal/modules/project/module.go web/apps/web/helpers/authentication.helper.ts web/packages/i18n/src/locales/en/auth.json web/packages/i18n/src/locales/zh-CN/auth.json api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/bodyshape.gen.go server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P7b): PATCH /api/v0/labels/{label_id} changes a label

The contract declares updateLabel and project.label_not_found; a null
parent moves the label to the top. The permission matrix asks each
column to rename its project's Feature, to take Bug's name, to give Bug,
which has a label under it, a parent, to send a blank name, and to
change the archived project's; the writes on a project rename a label
among them, the row read again by its own path.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p7b.py` 的编号；"层"是它被发现的每一层，端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `s4-update-frozen-clock` | `updateLabel` 接上停在 2001 年的时钟 | `TestTheWritesOnAProjectStampTheirRequest`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 10 起） | 组合 |
| `s4-update-drops-parent` | `PATCH /labels/{id}` 丢掉父标签 | `TestUpdateLabel`、`TestPermissionMatrix`、`TestTheWritesOnAProjectRunOnTheirTransactionsConnection`、`TestEachLabelWriteChangesItsRowsAlone`（Task 10 起）、P7（Task 11 起） | 单元；组合；端到端 |
| `s4-update-null-ignored` | `PATCH /labels/{id}` 把 null 的父标签当作没给 | `TestUpdateLabel`、P7（Task 11 起） | 单元；端到端 |
| `s18-update-extra-default` | `updateLabel` 的契约多声明从不回答的 `project.state_default` | `apitest.Main` | 单元（按性质只在单元一层：多声明、从不回答的码，只有 `apitest.Main` 的覆盖核对看得出） |

---

### Task 8: `deleteLabel`：用例、契约、处理函数、矩阵和组合的测试

**Files:**
- Create: `server/internal/modules/project/app/delete_label.go`、`server/internal/modules/project/app/delete_label_test.go`
- Modify: `api/modules/project.yaml`、`server/internal/bootstrap/permission_matrix_labels_test.go`、`server/internal/bootstrap/project_connection_test.go`、`server/internal/bootstrap/project_write_locks_test.go`、`server/internal/bootstrap/project_writes_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/labels.go`、`server/internal/modules/project/adapter/http/labels_test.go`、`server/internal/modules/project/app/clock_test.go`、`server/internal/modules/project/app/fakes_label_test.go`、`server/internal/modules/project/app/label_ports.go`、`server/internal/modules/project/domain/actions.go`、`server/internal/modules/project/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.6、2.7；M3 设计 3.6 约定五、3.16）：
  - `label_ports.go`：`LabelDeleter`（`LabelFinder`、`DeleteLabel`；说明：写在项目的 `FOR NO KEY UPDATE` 之下，它覆盖这个标签下面的标签，它们是同一个项目的）。
  - `delete_label.go`（新）：`NewDeleteLabel(locks, labels LabelDeleter, tx, clock)`、`Execute(ctx, id) error`：调用者 → 一个事务：`lockRowAndDecide(…, labelWrite(id, ActionLabelDelete, labels))` → 时钟 → `DeleteLabel`（这个标签和它下面的标签，一条语句，由调用者）。没有检查：删除有子标签的标签不是拒绝，是连带。
  - `label.delete`：同 `label.create`。
  - 契约：`DELETE /api/v0/labels/{label_id}`（`deleteLabel`，204；`x-problem-codes: [project.label_not_found, forbidden]`）；处理函数 `DeleteLabel`；`UseCases.DeleteLabel`；`module.go` 接上。没有新码。
  - 矩阵：`deleteLabel` 的 2 行（Bug 删除，它下面的 UI 一起；已归档的项目）。组合：`writesOnAProject` 加 `deleteLabel`（Task 7 改名的 Checked）；盖戳加 `deleteLabel`（bob 的 Bug 和它下面的 UI 一起删除：两行，`deleted_at = updated_at`，`updated_by_id` 是 alice 的）；连接加删除 Bug（Widgets 一起）。

**Tests:**
- `TestDeleteLabel`：有 UI 的 Bug（两行）、Bug 下面的 UI（一行）、Feature、已归档的 Ops 的 Docs；每一行核对调用的次序和被删除的标签，其余不变。
- `TestDeleteLabelRefuses`：没有调用者；没有标签、工作区或项目在锁等待期间删除、标签在锁等待期间删除或移到 Ops、看不见 Web 的调用者，各 `project.label_not_found`；成员 403。每一行之后标签不变。（重读的 id 的检查在 P7a 的 `lock_test.go` 里钉过一次，这里不再加行，S6。）
- `TestDeleteLabelReturnsEachFailure`：标签的读、两把锁、重读、判定、删除、提交。
- 处理函数 `TestDeleteLabel`（204 没有正文；404、403、500）；`TestEachWriteReadsTheClockUnderItsLock` 加 `deleteLabel`；`TestEveryRuleDecidesItsCells` 加 `label.delete`；矩阵、组合同 Task 7。

- [ ] **Step 1: 规则、动作、端口和用例**

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"label.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````
````new server/internal/modules/access/domain/rules.go
	"label.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	"label.delete": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"label.update": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"label.update": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"label.delete": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````

`server/internal/modules/project/domain/actions.go`（修改，2 处）：

````old server/internal/modules/project/domain/actions.go
	ActionLabelUpdate shared.Action = "label.update"
````
````new server/internal/modules/project/domain/actions.go
	ActionLabelUpdate shared.Action = "label.update"
	// ActionLabelDelete is deleting a label and the labels under it:
	// deleteLabel.
	ActionLabelDelete shared.Action = "label.delete"
````

````old server/internal/modules/project/domain/actions.go
		ActionLabelCreate, ActionLabelUpdate}
````
````new server/internal/modules/project/domain/actions.go
		ActionLabelCreate, ActionLabelUpdate, ActionLabelDelete}
````

`server/internal/modules/project/app/label_ports.go`（修改，1 处）：

````old server/internal/modules/project/app/label_ports.go
	UpdateLabel(ctx context.Context, id uuid.UUID, p domain.LabelPatch, by uuid.UUID, now time.Time) (domain.Label, error)
}

````
````new server/internal/modules/project/app/label_ports.go
	UpdateLabel(ctx context.Context, id uuid.UUID, p domain.LabelPatch, by uuid.UUID, now time.Time) (domain.Label, error)
}

// LabelDeleter is deleteLabel's repository. Its write runs in the
// transaction ctx carries, under the project's FOR NO KEY UPDATE, which
// covers the labels under the label: they are of its project (M3 design
// 3.16, 3.6 convention 5).
type LabelDeleter interface {
	LabelFinder
	// DeleteLabel deletes the undeleted label id and the undeleted labels
	// under it, by the account by at now, in one statement.
	DeleteLabel(ctx context.Context, id, by uuid.UUID, now time.Time) error
}

````

`server/internal/modules/project/app/delete_label.go`（新文件，44 行）：

````file server/internal/modules/project/app/delete_label.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteLabel deletes a label and the labels under it: DELETE
// /api/v0/labels/{label_id} (M3 design 3.16).
type DeleteLabel struct {
	locks  Locks
	labels LabelDeleter
	tx     shared.TxManager
	clock  Clock
}

// NewDeleteLabel returns the use case.
func NewDeleteLabel(locks Locks, labels LabelDeleter, tx shared.TxManager, clock Clock) *DeleteLabel {
	return &DeleteLabel{locks: locks, labels: labels, tx: tx, clock: clock}
}

// Execute, in one transaction, in the order of M3 design 3.6: the label's
// locks (lockRowAndDecide: the label read for its project and workspace,
// the workspace FOR SHARE, the project FOR NO KEY UPDATE, which every
// write of its labels takes, the label read again) and the decision on
// label.delete; an archived project's labels are deleted as any other's
// (3.19). Then the label and the labels under it, in one statement, by the
// caller at the time the clock gives under the locks.
func (u *DeleteLabel) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		_, l, err := lockRowAndDecide(ctx, u.locks, actor, labelWrite(id, domain.ActionLabelDelete, u.labels))
		if err != nil {
			return err
		}
		return u.labels.DeleteLabel(ctx, l.ID, actor.UserID, u.clock.Now())
	})
}
````

- [ ] **Step 2: 假实现和用例的测试**

`server/internal/modules/project/app/fakes_label_test.go`（修改，1 处）：

````old server/internal/modules/project/app/fakes_label_test.go
}

// taken reports whether a label of project but except has name, in any
````
````new server/internal/modules/project/app/fakes_label_test.go
}

// DeleteLabel deletes the label id and the labels under it.
func (f *fakeLabels) DeleteLabel(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "DeleteLabel %s by %s at %s", id, by, now.Format(timeFormat))
	if err := f.fail("DeleteLabel"); err != nil {
		return err
	}
	for key, l := range f.labels {
		if key == id || (l.ParentID != nil && *l.ParentID == id) {
			delete(f.labels, key)
		}
	}
	return nil
}

// taken reports whether a label of project but except has name, in any
````

`server/internal/modules/project/app/delete_label_test.go`（新文件，136 行）：

````file server/internal/modules/project/app/delete_label_test.go
package app_test

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newDeleteLabel is DeleteLabel over newLabels' fakes, its clock logged.
func newDeleteLabel() (*app.DeleteLabel, *writeFixture, *fakeLabels) {
	f, l := newLabels()
	return app.NewDeleteLabel(f.locks(), l, f.tx, clockAt{clockNow, f.log}), f, l
}

// labelDeleted are the calls of user's deletion of the label id of
// project: its locks and decision, the clock, the deletion by user at that
// time.
func labelDeleted(user, id, project uuid.UUID) []string {
	return append(labelLocked(id, project, user, domain.ActionLabelDelete), "Now",
		fmt.Sprintf("DeleteLabel %s by %s at %s", id, user, clockNow.Format(timeFormat)))
}

// DeleteLabel, in one transaction and in the order of M3 design 3.6 and
// 3.16, locks the label's project, decides, reads the clock, then deletes
// the label and the labels under it by the caller at that time: Bug takes
// UI with it; UI, under Bug, and Feature go alone; and ops's Docs, of an
// archived project, goes as any other (3.19).
func TestDeleteLabel(t *testing.T) {
	for _, tt := range []struct {
		name    string
		id      uuid.UUID
		project uuid.UUID
		gone    []uuid.UUID // the labels the deletion takes
	}{
		{"Bug, with UI under it", webBug, webID, []uuid.UUID{webBug, webUI}},
		{"UI, under Bug", webUI, webID, []uuid.UUID{webUI}},
		{"Feature", webFeature, webID, []uuid.UUID{webFeature}},
		{"archived ops's Docs", opsDocs, opsID, []uuid.UUID{opsDocs}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, l := newDeleteLabel()
			want := maps.Clone(l.labels)
			for _, id := range tt.gone {
				delete(want, id)
			}
			err := uc.Execute(as(bob), tt.id)
			if err != nil || !maps.Equal(l.labels, want) {
				t.Errorf("Execute() = %v, the labels after it %v; want nil, %v", err, l.labels, want)
			}
			if calls := labelDeleted(bob, tt.id, tt.project); !slices.Equal(f.log.calls, calls) {
				t.Errorf("calls\n%q\nwant\n%q", f.log.calls, calls)
			}
		})
	}
}

// Refusals, each in its place, and no label deleted: no caller, before the
// transaction; a label that is not there, a workspace or project deleted
// while its lock waited, a label deleted or moved to ops meanwhile, and a
// caller who does not see web, each project.label_not_found; a member, the
// Authorizer's 403. The label read for another id is the shared path's,
// pinned once in lock_test.go.
func TestDeleteLabelRefuses(t *testing.T) {
	upTo := func(n int) []string { return labelLocked(webBug, webID, bob, domain.ActionLabelDelete)[:n] }
	for _, tt := range []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		set   func(f *writeFixture)
		want  error
		calls []string
	}{
		{"no caller", context.Background(), webBug, nil, shared.Unauthenticated(), nil},
		{"no label", as(bob), uuid.Nil(), nil, domain.ErrLabelNotFound, []string{"Begin", "LabelByID " + uuid.Nil().String()}},
		{"acme deleted while its lock waited", as(bob), webBug, func(f *writeFixture) { f.workspaces.gone = true },
			domain.ErrLabelNotFound, upTo(3)},
		{"web deleted while its lock waited", as(bob), webBug, func(f *writeFixture) { f.store.deleted = true },
			domain.ErrLabelNotFound, upTo(4)},
		{"the label deleted while the locks waited", as(bob), webBug, func(f *writeFixture) { f.store.reread.gone = true },
			domain.ErrLabelNotFound, upTo(5)},
		{"the label moved to ops", as(bob), webBug, func(f *writeFixture) { f.store.reread.project = opsID },
			domain.ErrLabelNotFound, upTo(5)},
		{"a caller who does not see web", as(erin), webBug, nil, domain.ErrLabelNotFound,
			labelLocked(webBug, webID, erin, domain.ActionLabelDelete)},
		{"a member", as(alice), webBug, nil, shared.Forbidden(), labelLocked(webBug, webID, alice, domain.ActionLabelDelete)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, l := newDeleteLabel()
			if tt.set != nil {
				tt.set(f)
			}
			before := maps.Clone(l.labels)
			err := uc.Execute(tt.ctx, tt.id)
			outcome{tt.name, tt.want, tt.calls}.check(t, err, f)
			if !maps.Equal(l.labels, before) {
				t.Errorf("the labels after the refusal: %v; want them as they were, %v", l.labels, before)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after.
func TestDeleteLabelReturnsEachFailure(t *testing.T) {
	all := labelDeleted(bob, webBug, webID)
	fail := func(method string) func(f *writeFixture) {
		return func(f *writeFixture) { f.store.errs = map[string]error{method: errDisk} }
	}
	for _, tt := range []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of the deletion's calls ran
	}{
		{"the label's read", fail("LabelByID"), 2},
		{"the workspace's lock", func(f *writeFixture) { f.workspaces.err = errDisk }, 3},
		{"the project's lock", fail("LockProject"), 4},
		{"the read under the locks", func(f *writeFixture) { f.store.reread.err = errDisk }, 5},
		{"the decision", func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, 6},
		{"the deletion", fail("DeleteLabel"), 8},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 8},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, f, _ := newDeleteLabel()
			tt.fail(f)
			err := uc.Execute(as(bob), webBug)
			outcome{tt.name, errDisk, all[:tt.calls]}.check(t, err, f)
		})
	}
}
````

`server/internal/modules/project/app/clock_test.go`（修改，1 处）：

````old server/internal/modules/project/app/clock_test.go
		}, labelUpdated(bob, webFeature, webID, domain.LabelPatch{SetParent: true, ParentID: &webBug})},
````
````new server/internal/modules/project/app/clock_test.go
		}, labelUpdated(bob, webFeature, webID, domain.LabelPatch{SetParent: true, ParentID: &webBug})},
		{"deleteLabel", func() ([]string, error) {
			uc, f, _ := newDeleteLabel()
			err := uc.Execute(as(bob), webBug)
			return f.log.calls, err
		}, labelDeleted(bob, webBug, webID)},
````

- [ ] **Step 3: 契约**

`api/modules/project.yaml`（修改，1 处）：

````old api/modules/project.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/me/projects/{project_id}/preferences:
````
````new api/modules/project.yaml
        default:
          $ref: '#/components/responses/Problem'
    delete:
      operationId: deleteLabel
      tags: [project]
      summary: Delete a label and the labels under it
      description: >-
        For the project's admins, and its members who are the workspace's
        admins; an archived project's labels are deleted as any other's. A
        label that does not exist or is deleted, and a label whose project
        the caller does not see, answer project.label_not_found; a caller
        who sees the project but may not change its labels, forbidden. The
        label and the labels under it are deleted at the moment of the
        request, by the caller, and their names are free again in the
        project. The role is decided after the workspace and project rows
        are locked.
      security: [{bearer: []}]
      x-problem-codes: [project.label_not_found, forbidden]
      responses:
        '204':
          description: The label and the labels under it are deleted.
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/me/projects/{project_id}/preferences:
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `7df50b7b62a2f1a1cd68fab099ab0c15f00d4eb03a3014bdbfc2106675e9ca9f` | 2921 | `api/dist/openapi.yaml` |
| `b2044622d4d0f4f59f7fbf3410a744480bc8d411287ff6e31816bc277f0bf4c0` | 3398 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `1a31360a0bc8e63bfa09e03770a18fc0be8f4e0428a3f46a87415fe273bae62e` | 3241 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 4: 处理函数和接线**

`server/internal/modules/project/adapter/http/labels.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/labels.go
}

// label is l as the API shows it: its parent null at the top.
````
````new server/internal/modules/project/adapter/http/labels.go
}

// DeleteLabel serves DELETE /api/v0/labels/{label_id}.
func (h handler) DeleteLabel(ctx context.Context, req gen.DeleteLabelRequestObject) (gen.DeleteLabelResponseObject, error) {
	if err := h.uc.DeleteLabel.Execute(ctx, req.LabelID); err != nil {
		return nil, err
	}
	return gen.DeleteLabel204Response{}, nil
}

// label is l as the API shows it: its parent null at the top.
````

`server/internal/modules/project/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/handler.go
}

// UseCases are the use cases behind the module's operations.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// DeleteLabelUseCase is app.DeleteLabel.
type DeleteLabelUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// UseCases are the use cases behind the module's operations.
````

````old server/internal/modules/project/adapter/http/handler.go
	UpdateLabel         UpdateLabelUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
	UpdateLabel         UpdateLabelUseCase
	DeleteLabel         DeleteLabelUseCase
````

`server/internal/modules/project/adapter/http/labels_test.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/labels_test.go
			t.Errorf("%s: PATCH = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

````
````new server/internal/modules/project/adapter/http/labels_test.go
			t.Errorf("%s: PATCH = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// DELETE goes to the use case for the caller and the path's label, and
// answers 204 with no body; the use case's refusals, as the contract
// declares them, and its failure.
func TestDeleteLabel(t *testing.T) {
	remove := &fakeDelete{}
	h := newServer(t, fakes{deleteLabel: remove})
	path := "/api/v0/labels/" + webBugLabel.ID.String()
	if res, body := do(t, h, request(http.MethodDelete, path, "alice", "")); res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("DELETE = %d %q, want 204 and no body", res.StatusCode, body)
	}
	if want := []string{"alice " + webBugLabel.ID.String()}; !slices.Equal(remove.calls, want) {
		t.Errorf("calls = %q, want %q", remove.calls, want)
	}
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no label", domain.ErrLabelNotFound, http.StatusNotFound, labelNotFoundJSON},
		{"a project member", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{deleteLabel: &fakeDelete{err: tt.err}})
		if res, body := do(t, h, request(http.MethodDelete, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: DELETE = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	updateLabel         *fakeUpdateLabel
````
````new server/internal/modules/project/adapter/http/handler_test.go
	updateLabel         *fakeUpdateLabel
	deleteLabel         *fakeDelete
````

````old server/internal/modules/project/adapter/http/handler_test.go
		f.updateLabel = &fakeUpdateLabel{}
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		f.updateLabel = &fakeUpdateLabel{}
	}
	if f.deleteLabel == nil {
		f.deleteLabel = &fakeDelete{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
		MarkDefaultState: f.markDefault, ListWorkspaceStates: f.listWorkspaceStates, CreateLabel: f.createLabel, UpdateLabel: f.updateLabel})
````
````new server/internal/modules/project/adapter/http/handler_test.go
		MarkDefaultState: f.markDefault, ListWorkspaceStates: f.listWorkspaceStates, CreateLabel: f.createLabel, UpdateLabel: f.updateLabel,
		DeleteLabel: f.deleteLabel})
````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// workspace's projects one is a member of, creating and changing a
// project's labels, carries out the workspace module's cascades on the
````
````new server/internal/modules/project/module.go
// workspace's projects one is a member of, creating, changing and deleting
// a project's labels, carries out the workspace module's cascades on the
````

````old server/internal/modules/project/module.go
		UpdateLabel:         app.NewUpdateLabel(locks, store, d.Tx, d.Clock),
````
````new server/internal/modules/project/module.go
		UpdateLabel:         app.NewUpdateLabel(locks, store, d.Tx, d.Clock),
		DeleteLabel:         app.NewDeleteLabel(locks, store, d.Tx, d.Clock),
````

- [ ] **Step 5: 矩阵和组合的测试**

`server/internal/bootstrap/permission_matrix_labels_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_labels_test.go
			check: renamesTheLabel},
````
````new server/internal/bootstrap/permission_matrix_labels_test.go
			check: renamesTheLabel},
		// As createLabel: Bug deleted, and UI under it with it.
		{op: "deleteLabel", write: true, columns: projectColumns, request: toLabel(http.MethodDelete, "Bug", ""),
			cells: ofLabel(cellNoContent, cellForbidden, cellForbidden, cellNoContent, cellForbidden, cellForbidden)},
		{op: "deleteLabel", variant: "archived", write: true, columns: archivedColumns, request: toLabel(http.MethodDelete, "Bug", ""),
			cells: ofArchivedLabel(cellNoContent, cellForbidden)},
````

`server/internal/bootstrap/project_write_locks_test.go`（修改，1 处）：

````old server/internal/bootstrap/project_write_locks_test.go
	{op: "updateLabel", method: http.MethodPatch, path: "/api/v0/labels/%s", body: `{"name":"Checked"}`, want: http.StatusOK, row: labelNamed("QA")},
````
````new server/internal/bootstrap/project_write_locks_test.go
	{op: "updateLabel", method: http.MethodPatch, path: "/api/v0/labels/%s", body: `{"name":"Checked"}`, want: http.StatusOK, row: labelNamed("QA")},
	// The label renamed before, deleted.
	{op: "deleteLabel", method: http.MethodDelete, path: "/api/v0/labels/%s", want: http.StatusNoContent, row: labelNamed("Checked")},
````

`server/internal/bootstrap/project_writes_test.go`（修改，3 处）：

````old server/internal/bootstrap/project_writes_test.go
// renames it, unarchives Web, and leaves it. Each row the write writes
// again is first made bob's, as last written by him, and checked so: a
// write that kept its row's writer would pass for alice's otherwise, she
// having made it.
````
````new server/internal/bootstrap/project_writes_test.go
// renames it, deletes bob's label Bug and the label under it, unarchives
// Web, and leaves it. Each row the write writes again is first made bob's,
// as last written by him, and checked so: a write that kept its row's
// writer would pass for alice's otherwise, she having made it.
````

````old server/internal/bootstrap/project_writes_test.go
	carol := carolID.String()
````
````new server/internal/bootstrap/project_writes_test.go
	carol, bug := carolID.String(), uuid.NewV7().String()
````

````old server/internal/bootstrap/project_writes_test.go
			"SELECT updated_at, updated_by_id = $2 AND name = 'Checked' FROM labels WHERE project_id = $1 AND name IN ('QA', 'Checked')", 1, 1},
````
````new server/internal/bootstrap/project_writes_test.go
			"SELECT updated_at, updated_by_id = $2 AND name = 'Checked' FROM labels WHERE project_id = $1 AND name IN ('QA', 'Checked')", 1, 1},
		// Bug and UI under it, made by bob, deleted together: two rows.
		{"deleteLabel", http.MethodDelete, "/api/v0/labels/%s", "", label("Bug"), http.StatusNoContent,
			"INSERT INTO labels (id, workspace_id, project_id, parent_id, name, created_by_id, updated_by_id) SELECT '" + bug + "'::uuid, " +
				"workspace_id, id, NULL::uuid, 'Bug', $3::uuid, $3::uuid FROM projects WHERE id = $1 AND created_by_id = $2 UNION ALL SELECT '" +
				uuid.NewV7().String() + "'::uuid, workspace_id, id, '" + bug + "'::uuid, 'UI', $3::uuid, $3::uuid FROM projects WHERE id = $1 " +
				"AND created_by_id = $2",
			"SELECT updated_at, updated_by_id = $2 AND deleted_at = updated_at FROM labels WHERE project_id = $1 AND name IN ('Bug', 'UI')", 2, 2},
````

`server/internal/bootstrap/project_connection_test.go`（修改，2 处）：

````old server/internal/bootstrap/project_connection_test.go
// which the same read refuses, renames UI Widgets at the top, and moves it
// back under Bug; carol, an admin, leaves it; alice deletes both projects.
````
````new server/internal/bootstrap/project_connection_test.go
// which the same read refuses, renames UI Widgets at the top, moves it
// back under Bug, and deletes Bug, Widgets with it; carol, an admin,
// leaves Web; alice deletes both projects.
````

````old server/internal/bootstrap/project_connection_test.go
	send(http.MethodPatch, "/api/v0/labels/"+ui.ID.String(), r.alice, `{"parent_id":"`+bug.ID.String()+`"}`, http.StatusOK)
````
````new server/internal/bootstrap/project_connection_test.go
	send(http.MethodPatch, "/api/v0/labels/"+ui.ID.String(), r.alice, `{"parent_id":"`+bug.ID.String()+`"}`, http.StatusOK)
	send(http.MethodDelete, "/api/v0/labels/"+bug.ID.String(), r.alice, "", http.StatusNoContent)
````

- [ ] **Step 6: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation$|TestMatrixViolationsCatchesEachColumnGap$|TestWritesOnAProjectAreEachShape$|TestEachWriteOnAProjectSharesItsWorkspaceFirst$|TestTheWritesOnAProjectStampTheirRequest$|TestTheWritesOnAProjectRunOnTheirTransactionsConnection$' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

- [ ] **Step 7: 提交**

```bash
git add api/modules/project.yaml server/internal/bootstrap/permission_matrix_labels_test.go server/internal/bootstrap/project_connection_test.go server/internal/bootstrap/project_write_locks_test.go server/internal/bootstrap/project_writes_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/labels.go server/internal/modules/project/adapter/http/labels_test.go server/internal/modules/project/app/clock_test.go server/internal/modules/project/app/delete_label.go server/internal/modules/project/app/delete_label_test.go server/internal/modules/project/app/fakes_label_test.go server/internal/modules/project/app/label_ports.go server/internal/modules/project/domain/actions.go server/internal/modules/project/module.go api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P7b): DELETE /api/v0/labels/{label_id} deletes a label and the labels under it

deleteLabel takes the shared row path and decides label.delete, then
deletes the label and the labels under it in one statement, by the
caller at the time the clock gives under the locks. The labels under it
are of its project, whose lock it holds, so the deletion needs no lock
of its own on them. The permission matrix asks each column to delete
its project's Bug, UI with it, and the archived project's.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p7b.py` 的编号；"层"是它被发现的每一层，端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `s2-delete-delete` | `deleteLabel` 吞掉 `DeleteLabel` 的失败 | `TestDeleteLabelReturnsEachFailure` | 单元（按性质只在单元一层：真实的存储在这一步不失败） |
| `c-delete-early` | `deleteLabel` 在锁之前读时钟 | `TestEachWriteReadsTheClockUnderItsLock`、`TestDeleteLabel`、`TestDeleteLabelRefuses`、`TestDeleteLabelReturnsEachFailure`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 10 起）、`TestLabelWritesOnOneProjectSerialize`（Task 10 起） | 单元；组合 |
| `s4-delete-frozen-clock` | `deleteLabel` 接上停在 2001 年的时钟 | `TestTheWritesOnAProjectStampTheirRequest`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength`（Task 10 起） | 组合 |
| `s5-delete-guests` | 规则表：`label.delete` 给访客 | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix` | 单元；组合 |
| `s18-delete-no-not-found` | `deleteLabel` 的契约不声明 `project.label_not_found` | `TestDeleteLabel`、`TestPermissionMatrix`、`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`（Task 10 起） | 单元；组合 |
| `s34-first-lock-row` | 最先锁工作区的测试去掉 `deleteLabel` 一行 | `TestEachWriteOnAProjectSharesItsWorkspaceFirst` | 组合 |

---

### Task 9: `listLabels`：用例、契约、处理函数、矩阵

**Files:**
- Create: `server/internal/modules/project/app/list_labels.go`、`server/internal/modules/project/app/list_labels_test.go`
- Modify: `api/modules/project.yaml`、`server/internal/bootstrap/permission_matrix_labels_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/project/adapter/http/handler.go`、`server/internal/modules/project/adapter/http/handler_test.go`、`server/internal/modules/project/adapter/http/labels.go`、`server/internal/modules/project/adapter/http/labels_test.go`、`server/internal/modules/project/app/fakes_label_test.go`、`server/internal/modules/project/app/label_ports.go`、`server/internal/modules/project/domain/actions.go`、`server/internal/modules/project/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/project/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.6、2.7；M3 设计 3.16、3.19）：
  - `label_ports.go`：`LabelLister`（`ProjectFinder`、`ListLabels`）。
  - `list_labels.go`（新）：`NewListLabels(labels LabelLister, auth)`、`Execute(ctx, projectID) ([]domain.Label, error)`：调用者 → `findAndDecide`（项目的工作区、`label.list` 的判定；不开事务，不加锁）→ `ListLabels`（父子一起，按 `sort_order`、再按 `id`；已归档的项目照常）。
  - `label.list`：项目级、项目的有效成员（管理员、成员、访客，同 `state.list`）。
  - 契约：`GET /api/v0/projects/{project_id}/labels`（`listLabels`，200 `LabelList`；`x-problem-codes: [project.not_found, forbidden]`）、`LabelList`；处理函数 `ListLabels`（`data`）；`module.go` 接上 `app.NewListLabels(store, d.Authorizer)`。
  - 矩阵：`listLabels` 的 2 行（每一列的项目的标签，`listsTheLabels` 按 `sort_order` 核对、UI 在 Bug 下；已归档的项目）。

**Tests:**
- `TestListLabels`：Web 的标签按存储的次序原样返回（假实现的次序既不是按 `sort_order` 的正序、倒序，也不是按 id：在用例里排序的实现会答出另一个次序）；已归档的 Ops 照常；调用都在事务之外（项目的工作区、判定、`ListLabels`）。
- `TestListLabelsRefuses`：没有调用者；没有项目、看不见（404）；看得见而不是成员（403）；项目、判定、列出各自的失败原样返回，之后的调用都不在。
- 处理函数 `TestListLabels`（200 的 `data` 按用例的次序、每个字段，没有标签时是 `[]`；404、403、500）；`TestEveryRuleDecidesItsCells` 加 `label.list`；`TestPermissionMatrix` 的 2 行。

- [ ] **Step 1: 规则、动作、端口和用例**

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"state.mark_default": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
````
````new server/internal/modules/access/domain/rules.go
	"state.mark_default": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	// As state.list (M3 design 9.2).
	"label.list": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"state.mark_default": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible,
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
````
````new server/internal/modules/access/domain/rules_test.go
	"state.mark_default": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible,
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
	// As state.list.
	"label.list": {allowed, allowed, allowed, allowed, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
````

`server/internal/modules/project/domain/actions.go`（修改，2 处）：

````old server/internal/modules/project/domain/actions.go
	ActionWorkspaceStateList shared.Action = "workspace_state.list"
````
````new server/internal/modules/project/domain/actions.go
	ActionWorkspaceStateList shared.Action = "workspace_state.list"
	// ActionLabelList is listing a project's labels: listLabels.
	ActionLabelList shared.Action = "label.list"
````

````old server/internal/modules/project/domain/actions.go
		ActionLabelCreate, ActionLabelUpdate, ActionLabelDelete}
````
````new server/internal/modules/project/domain/actions.go
		ActionLabelList, ActionLabelCreate, ActionLabelUpdate, ActionLabelDelete}
````

`server/internal/modules/project/app/label_ports.go`（修改，1 处）：

````old server/internal/modules/project/app/label_ports.go
// (M3 design 3.16).
````
````new server/internal/modules/project/app/label_ports.go
// (M3 design 3.16).

// LabelLister is listLabels' repository.
type LabelLister interface {
	ProjectFinder
	// ListLabels lists projectID's undeleted labels, parents and children
	// alike, by sort order, then id; an archived project's too.
	ListLabels(ctx context.Context, projectID uuid.UUID) ([]domain.Label, error)
}
````

`server/internal/modules/project/app/list_labels.go`（新文件，35 行）：

````file server/internal/modules/project/app/list_labels.go
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListLabels lists a project's labels: GET
// /api/v0/projects/{project_id}/labels.
type ListLabels struct {
	labels LabelLister
	auth   shared.Authorizer
}

// NewListLabels returns the use case.
func NewListLabels(labels LabelLister, auth shared.Authorizer) *ListLabels {
	return &ListLabels{labels: labels, auth: auth}
}

// Execute lists the project's labels, parents and children alike, by sort
// order, an archived project's as any other's (M3 design 3.16, 3.19), after
// the decision on label.list. A read opens no transaction.
func (u *ListLabels) Execute(ctx context.Context, projectID uuid.UUID) ([]domain.Label, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	if err := findAndDecide(ctx, u.labels, u.auth, actor, projectID, domain.ActionLabelList); err != nil {
		return nil, err
	}
	return u.labels.ListLabels(ctx, projectID)
}
````

- [ ] **Step 2: 假实现和用例的测试**

`server/internal/modules/project/app/fakes_label_test.go`（修改，1 处）：

````old server/internal/modules/project/app/fakes_label_test.go
	slices.SortFunc(out, func(a, b domain.Label) int { return cmp.Compare(a.Name, b.Name) })
	return out
````
````new server/internal/modules/project/app/fakes_label_test.go
	slices.SortFunc(out, func(a, b domain.Label) int { return cmp.Compare(a.Name, b.Name) })
	return out
}

func (f *fakeLabels) ListLabels(ctx context.Context, projectID uuid.UUID) ([]domain.Label, error) {
	f.log.add(ctx, "ListLabels %s", projectID)
	if err := f.fail("ListLabels"); err != nil {
		return nil, err
	}
	return f.of(projectID), nil
````

`server/internal/modules/project/app/list_labels_test.go`（新文件，91 行）：

````file server/internal/modules/project/app/list_labels_test.go
package app_test

import (
	"cmp"
	"context"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// labelsListed are the calls of user's list of project's labels: the
// project's workspace, the decision, the list; none in a transaction.
func labelsListed(user, project uuid.UUID) []string {
	return []string{"ProjectWorkspace " + project.String() + " outside tx",
		fmt.Sprintf("Authorize %s %s on %s/%s outside tx", user, domain.ActionLabelList, acme.ID, project),
		fmt.Sprintf("ListLabels %s outside tx", project)}
}

// ListLabels decides label.list on the project, then answers the store's
// list as it is, in the store's order, which no sort by sort order, either
// way, nor by id gives: a use case that sorted the labels would answer
// another. ops, archived, has its labels listed as any other's (3.19). A
// read opens no transaction.
func TestListLabels(t *testing.T) {
	for _, project := range []uuid.UUID{webID, opsID} {
		f, l := newLabels()
		want := l.of(project)
		if project == webID {
			for _, by := range []struct {
				name  string
				order func(a, b domain.Label) int
			}{
				{"the sort order's", func(a, b domain.Label) int { return cmp.Compare(a.SortOrder, b.SortOrder) }},
				{"the sort order's, reversed", func(a, b domain.Label) int { return cmp.Compare(b.SortOrder, a.SortOrder) }},
				{"the id's", func(a, b domain.Label) int { return slices.Compare(a.ID[:], b.ID[:]) }},
			} {
				if slices.IsSortedFunc(want, by.order) {
					t.Fatalf("the store's order %v is %s: a use case that sorted so would pass", want, by.name)
				}
			}
		}
		got, err := app.NewListLabels(l, f.auth).Execute(as(bob), project)
		if err != nil || len(got) == 0 || !reflect.DeepEqual(got, want) || !slices.Equal(f.log.calls, labelsListed(bob, project)) {
			t.Errorf("Execute(%s) = %+v, %v, calls %q; want %+v, calls %q", project, got, err, f.log.calls, want, labelsListed(bob, project))
		}
	}
}

// Refusals, each in its place: no caller; a project not there, or not
// visible: 404; the Authorizer's 403 for one who sees it and is not its
// member. Every port's failure comes back as itself, after the calls
// before it and none after.
func TestListLabelsRefuses(t *testing.T) {
	tests := []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		fail  func(f *writeFixture)
		want  error
		calls []string
	}{
		{"no caller", context.Background(), webID, nil, shared.Unauthenticated(), nil},
		{"no project", as(bob), uuid.Nil(), nil, domain.ErrNotFound, []string{"ProjectWorkspace " + uuid.Nil().String() + " outside tx"}},
		{"not seen", as(erin), webID, nil, domain.ErrNotFound, labelsListed(erin, webID)[:2]},
		{"forbidden", as(alice), webID, nil, shared.Forbidden(), labelsListed(alice, webID)[:2]},
		{"the project failing", as(bob), webID, func(f *writeFixture) { f.store.errs = map[string]error{"ProjectWorkspace": errDisk} }, errDisk,
			labelsListed(bob, webID)[:1]},
		{"the decision failing", as(bob), webID, func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, errDisk,
			labelsListed(bob, webID)[:2]},
		{"the list failing", as(bob), webID, func(f *writeFixture) { f.store.errs = map[string]error{"ListLabels": errDisk} }, errDisk,
			labelsListed(bob, webID)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, l := newLabels()
			if tt.fail != nil {
				tt.fail(f)
			}
			got, err := app.NewListLabels(l, f.auth).Execute(tt.ctx, tt.id)
			if !refusedAs(err, tt.want) || got != nil || !slices.Equal(f.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, f.log.calls, tt.want, tt.calls)
			}
		})
	}
}
````

- [ ] **Step 3: 契约**

`api/modules/project.yaml`（修改，2 处）：

````old api/modules/project.yaml
  /api/v0/projects/{project_id}/labels:
    parameters:
      - $ref: '#/components/parameters/ProjectID'
    post:
````
````new api/modules/project.yaml
  /api/v0/projects/{project_id}/labels:
    parameters:
      - $ref: '#/components/parameters/ProjectID'
    get:
      operationId: listLabels
      tags: [project]
      summary: List a project's labels
      description: >-
        For the project's active members: its labels, those at the top and
        those under them alike, by sort order, the lowest first, then by id;
        an archived project's as any other's. A project that does not
        exist, is deleted, or that the caller does not see answers
        project.not_found; one he sees but is not a member of, forbidden.
        The whole collection at once: collections are not paginated.
      security: [{bearer: []}]
      x-problem-codes: [project.not_found, forbidden]
      responses:
        '200':
          description: The project's labels.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/LabelList'
        default:
          $ref: '#/components/responses/Problem'
    post:
````

````old api/modules/project.yaml
          type: string
          format: date-time
    LabelCreate:
````
````new api/modules/project.yaml
          type: string
          format: date-time
    LabelList:
      type: object
      additionalProperties: false
      required: [data]
      properties:
        data:
          type: array
          items:
            $ref: '#/components/schemas/Label'
    LabelCreate:
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `51386f7a5b38de50e767fa685f4f0c8ccc84f8ba81481c0e81c96772e2cd3c49` | 2951 | `api/dist/openapi.yaml` |
| `5898a2d94b8ebb9914d562f697efb6b02f749e25158314e12e497627084910c6` | 3508 | `server/internal/modules/project/adapter/http/gen/server.gen.go` |
| `ea2fe0baf614ed579977772a4a7e57e1c430de5d48a354cbdae7a1232c425fa4` | 3273 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 4: 处理函数、接线和矩阵**

`server/internal/modules/project/adapter/http/labels.go`（修改，1 处）：

````old server/internal/modules/project/adapter/http/labels.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)
````
````new server/internal/modules/project/adapter/http/labels.go
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// ListLabels serves GET /api/v0/projects/{project_id}/labels: data an
// array, never null.
func (h handler) ListLabels(ctx context.Context, req gen.ListLabelsRequestObject) (gen.ListLabelsResponseObject, error) {
	list, err := h.uc.ListLabels.Execute(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	out := gen.LabelList{Data: make([]gen.Label, len(list))}
	for i, l := range list {
		out.Data[i] = label(l)
	}
	return gen.ListLabels200JSONResponse(out), nil
}
````

`server/internal/modules/project/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/handler.go
}

// CreateLabelUseCase is app.CreateLabel.
````
````new server/internal/modules/project/adapter/http/handler.go
}

// ListLabelsUseCase is app.ListLabels.
type ListLabelsUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID) ([]domain.Label, error)
}

// CreateLabelUseCase is app.CreateLabel.
````

````old server/internal/modules/project/adapter/http/handler.go
	ListWorkspaceStates ListWorkspaceStatesUseCase
````
````new server/internal/modules/project/adapter/http/handler.go
	ListWorkspaceStates ListWorkspaceStatesUseCase
	ListLabels          ListLabelsUseCase
````

`server/internal/modules/project/adapter/http/labels_test.go`（修改，2 处）：

````old server/internal/modules/project/adapter/http/labels_test.go
	"github.com/open-nerve/NerveProject/server/internal/shared"
)
````
````new server/internal/modules/project/adapter/http/labels_test.go
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeListLabels is listLabels: each call is recorded as "caller id"; it
// answers list, or err.
type fakeListLabels struct {
	calls []string
	list  []domain.Label
	err   error
}

func (f *fakeListLabels) Execute(ctx context.Context, projectID uuid.UUID) ([]domain.Label, error) {
	f.calls = append(f.calls, caller(ctx)+" "+projectID.String())
	return f.list, f.err
}
````

````old server/internal/modules/project/adapter/http/labels_test.go
	Message: "must be a label without a parent: labels have two levels"})
````
````new server/internal/modules/project/adapter/http/labels_test.go
	Message: "must be a label without a parent: labels have two levels"})

// GET goes to the use case for the caller and the path's project; the
// answer is 200 with its list, in its order, every field of each label,
// and [] for none. Its refusals and its failure, as the contract declares
// them.
func TestListLabels(t *testing.T) {
	path := "/api/v0/projects/" + webID.String() + "/labels"
	for _, tt := range []struct {
		list []domain.Label
		want string
	}{{[]domain.Label{webUILabel, webBugLabel}, `{"data":[` + webUILabelJSON + `,` + webBugLabelJSON + `]}`}, {nil, `{"data":[]}`}} {
		list := &fakeListLabels{list: tt.list}
		h := newServer(t, fakes{listLabels: list})
		if res, body := do(t, h, request(http.MethodGet, path, "bob", "")); res.StatusCode != http.StatusOK || body != tt.want+"\n" {
			t.Errorf("GET = %d %s, want 200 %s", res.StatusCode, body, tt.want)
		}
		if want := []string{"bob " + webID.String()}; !slices.Equal(list.calls, want) {
			t.Errorf("calls = %q, want %q", list.calls, want)
		}
	}
	for _, tt := range []struct {
		err    error
		status int
		want   string
	}{
		{domain.ErrNotFound, http.StatusNotFound, projectNotFoundJSON},
		{shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{listLabels: &fakeListLabels{err: tt.err}})
		if res, body := do(t, h, request(http.MethodGet, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("GET = %d %s, want %d %s", res.StatusCode, body, tt.status, tt.want)
		}
	}
}
````

`server/internal/modules/project/adapter/http/handler_test.go`（修改，3 处）：

````old server/internal/modules/project/adapter/http/handler_test.go
	listWorkspaceStates *fakeListWorkspaceStates
````
````new server/internal/modules/project/adapter/http/handler_test.go
	listWorkspaceStates *fakeListWorkspaceStates
	listLabels          *fakeListLabels
````

````old server/internal/modules/project/adapter/http/handler_test.go
		f.listWorkspaceStates = &fakeListWorkspaceStates{}
	}
````
````new server/internal/modules/project/adapter/http/handler_test.go
		f.listWorkspaceStates = &fakeListWorkspaceStates{}
	}
	if f.listLabels == nil {
		f.listLabels = &fakeListLabels{}
	}
````

````old server/internal/modules/project/adapter/http/handler_test.go
		MarkDefaultState: f.markDefault, ListWorkspaceStates: f.listWorkspaceStates, CreateLabel: f.createLabel, UpdateLabel: f.updateLabel,
		DeleteLabel: f.deleteLabel})
````
````new server/internal/modules/project/adapter/http/handler_test.go
		MarkDefaultState: f.markDefault, ListWorkspaceStates: f.listWorkspaceStates, ListLabels: f.listLabels, CreateLabel: f.createLabel,
		UpdateLabel: f.updateLabel, DeleteLabel: f.deleteLabel})
````

`server/internal/modules/project/module.go`（修改，2 处）：

````old server/internal/modules/project/module.go
// workspace's projects one is a member of, creating, changing and deleting
// a project's labels, carries out the workspace module's cascades on the
// projects (ProjectCascade), and offers the access module its reads of a
// project (ProjectAccess) and the workspace module its count of an
````
````new server/internal/modules/project/module.go
// workspace's projects one is a member of, listing, creating, changing and
// deleting a project's labels, carries out the workspace module's cascades
// on the projects (ProjectCascade), and offers the access module its reads
// of a project (ProjectAccess) and the workspace module its count of an
````

````old server/internal/modules/project/module.go
		ListWorkspaceStates: app.NewListWorkspaceStates(d.Workspaces, store, d.Authorizer),
````
````new server/internal/modules/project/module.go
		ListWorkspaceStates: app.NewListWorkspaceStates(d.Workspaces, store, d.Authorizer),
		ListLabels:          app.NewListLabels(store, d.Authorizer),
````

`server/internal/bootstrap/permission_matrix_labels_test.go`（修改，3 处）：

````old server/internal/bootstrap/permission_matrix_labels_test.go
// creating them, under the project of each column, and the writes on one,
// naming it by its id (/labels/{label_id}) among its column's project's
// seeded labels; and the labels they rest on, matrixLabels, which
// prepareMatrix seeds in each project through the project store (labels)
// and reads back (seededLabels). They are here rather than in
// permission_matrix_seed_test.go, which has no room for them.
````
````new server/internal/bootstrap/permission_matrix_labels_test.go
// listing and creating them, under the project of each column, and the
// writes on one, naming it by its id (/labels/{label_id}) among its
// column's project's seeded labels; and the labels they rest on,
// matrixLabels, which prepareMatrix seeds in each project through the
// project store (labels) and reads back (seededLabels). They are here
// rather than in permission_matrix_seed_test.go, which has no room for
// them.
````

````old server/internal/bootstrap/permission_matrix_labels_test.go
	return []matrixRow{
````
````new server/internal/bootstrap/permission_matrix_labels_test.go
	return []matrixRow{
		// Every active member of the project (M3 design 9.2), as listStates.
		{op: "listLabels", columns: projectColumns, request: toProject(http.MethodGet, "/labels", ""),
			cells: ofProject(cellOK, cellOK, cellOK, cellOK, cellForbidden, cellForbidden), check: listsTheLabels},
		// An archived project's labels are listed as any other's (M3 design
		// 3.19).
		{op: "listLabels", variant: "archived", columns: archivedColumns, request: toProject(http.MethodGet, "/labels", ""),
			cells: ofArchived(cellOK, cellForbidden), check: listsTheLabels},
````

````old server/internal/bootstrap/permission_matrix_labels_test.go
			cells: ofArchivedLabel(cellNoContent, cellForbidden)},
````
````new server/internal/bootstrap/permission_matrix_labels_test.go
			cells: ofArchivedLabel(cellNoContent, cellForbidden)},
	}
}

// listsTheLabels: the column's project's labels, by sort order, as seeded:
// UI under Bug, the others at the top; the archived project's too.
func listsTheLabels(t *testing.T, c caller, s seeded, answer string) {
	var list struct {
		Data []struct {
			ID        uuid.UUID  `json:"id"`
			ProjectID uuid.UUID  `json:"project_id"`
			ParentID  *uuid.UUID `json:"parent_id"`
			Name      string     `json:"name"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	var got, want []string
	for _, l := range list.Data {
		if l.ProjectID != s.project(projectOf(c)) {
			t.Errorf("%s lists %s, not its project's", c, answer)
		}
		parent := "the top"
		if l.ParentID != nil {
			parent = l.ParentID.String()
		}
		got = append(got, l.Name+" "+l.ID.String()+" under "+parent)
	}
	for _, l := range matrixLabels {
		parent := "the top"
		if l.parent != "" {
			parent = s.label(projectOf(c), l.parent).String()
		}
		want = append(want, l.name+" "+s.label(projectOf(c), l.name).String()+" under "+parent)
	}
	if !slices.Equal(got, want) {
		t.Errorf("%s lists %q, want %q", c, got, want)
````

- [ ] **Step 5: 测试和 lint**

Run: `go -C server test -count=1 ./internal/modules/project/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix$|TestThePermissionMatrixCoversEveryOperation$' ./internal/bootstrap/`
Expected: `ok`（矩阵 853 格：P7a 结束时 721 格，P7b 的 14 行加 132 格；原型上 1.50 秒）。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

- [ ] **Step 6: 提交**

```bash
git add api/modules/project.yaml server/internal/bootstrap/permission_matrix_labels_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/project/adapter/http/handler.go server/internal/modules/project/adapter/http/handler_test.go server/internal/modules/project/adapter/http/labels.go server/internal/modules/project/adapter/http/labels_test.go server/internal/modules/project/app/fakes_label_test.go server/internal/modules/project/app/label_ports.go server/internal/modules/project/app/list_labels.go server/internal/modules/project/app/list_labels_test.go server/internal/modules/project/domain/actions.go server/internal/modules/project/module.go api/dist/openapi.yaml server/internal/modules/project/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P7b): GET /api/v0/projects/{project_id}/labels lists a project's labels

listLabels decides label.list, for the project's active members, and
lists its labels, parents and children alike, by sort order, then id,
an archived project's as any other's, without a transaction. The
permission matrix asks each column for its project's labels and the
archived project's.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A；`mutants_p7b.py` 的编号；"层"是它被发现的每一层，端到端是单独运行的故事）：

| 变异 | 改坏 | 必须失败的测试 | 层 |
|---|---|---|---|
| `s2-list` | `listLabels` 把 `ListLabels` 的失败答成空列表 | `TestListLabelsRefuses` | 单元（按性质只在单元一层：真实的存储在这一步不失败） |
| `s5-list-no-guests` | 规则表：`label.list` 不给访客 | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix` | 单元；组合 |
| `s5-list-undecided` | `listLabels` 不判定就列出 | `TestListLabels`、`TestListLabelsRefuses`、`TestPermissionMatrix` | 单元；组合 |

---

### Task 10: 并发：交错 11、同名的创建、约定五的探测；每个标签的写只写它的行；竞争和锁强度加标签的行

**Files:**
- Create: `server/internal/bootstrap/interleaving_labels_test.go`、`server/internal/bootstrap/label_rows_test.go`
- Modify: `server/internal/bootstrap/project_row_races_test.go`

**Interfaces:**
- Produces（spec 2.9；M3 设计 3.6、3.16、9.3）：
  - `interleaving_labels_test.go`（新）：`labelWrittenHolding`（项目的存储，写之后停在闸门上，持有工作区的 `FOR SHARE`、项目的 `FOR NO KEY UPDATE` 和它写的行）；`seedLabelsOf`（Web 或 Ops 的 `worldLabels`：Bug、它下面的 UI、Feature、Docs，由 alice（两个项目的管理员）经项目的存储建出、读回、核对）、`labelsOf`（未删除的标签：名称 → 父标签的名称）；`labelMoves`、`labelCreates`、`labelDeletes`；`TestLabelWritesOnOneProjectSerialize` 的 7 对，各两种顺序：
    1. Feature 移到 Docs 下与 Docs 移到 Feature 下（交错 11：后一个 422，父标签有父标签）；
    2. 在 Feature 下建 Icons 与 Feature 移到 Docs 下（先建：Feature 有子标签，不能有父标签；先移：Feature 有父标签，不能做父标签）；
    3. 建 QA 与建 qa（后一个 409）；
    4. Feature 移到 Docs 下与移到 Bug 下（都成功，后一个从前一个放的地方移走）；
    5. 删除 Bug 与把 Feature 移到 Bug 下（约定五的探测：先删，Bug 不再是项目的标签，422；后删，删除的一条语句把刚放到 Bug 下的 Feature 和 Bug、UI 一起带走）；
    6. 删除 Bug 与在 Bug 下建 Icons（同上）；
    7. 在顶层建 QA 与在 Bug 下建 Icons（都成功：后一个的时刻是它拿到锁之后时钟给的）。
    之后 Web 的标签是先到的写留下的（后到的成功时也加上它的），时刻不早于闸门打开；Ops 的不变。
  - `label_rows_test.go`（新）：`TestEachLabelWriteChangesItsRowsAlone`：bob 依次建 QA、在 Feature 下建 Icons、在 Docs 下建 Fonts、Feature 改名、UI 移到顶层、删除 Fonts、把 Docs（它下面唯一的标签已删除）放到 Bug 下、删除 Feature（Icons 一起）、删除 Docs（之前删除的 Fonts 保留它的删除）；其间有子标签的 Bug 设父标签（422）、建 bug（409）什么都不写；每个写之后，它写的行之外的每一行每一列不变，已删除的行也在内。
  - `project_row_races_test.go`：`rowWrite` 加 `label`；`rowWrites` 加 `updateLabel`、`deleteLabel`（bob 改 Web 的 Feature）和按项目寻址的 `createLabel`（同 `createState`：没有自己的行可等，由 `creates` 跳过）；`row` 回答 `labels` 的行；`labelsFor` 给写标签的行种下 `seedLabelsOf`。

**Tests:**
- `TestLabelWritesOnOneProjectSerialize`（14 个子测试）；`TestEachLabelWriteChangesItsRowsAlone`；`TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile`、`TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength` 加标签的三个写（行结束这一行只有成员关系有；移到 Ops、删除、调用者的成员关系结束、Web 删除、acme 删除，写答 404、什么都不改；锁强度：等 acme 时什么都不持有，等 Web 时持有 acme 的 `FOR SHARE`，等这一行时持有 Web 的 `FOR NO KEY UPDATE`）。

- [ ] **Step 1: 交错**

`server/internal/bootstrap/interleaving_labels_test.go`（新文件，277 行）：

````file server/internal/bootstrap/interleaving_labels_test.go
package bootstrap

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Interleaving 11, the two levels of labels (M3 design 9.3, 3.16), on
// memberWorld's real database, in both orders: project's CreateLabel,
// UpdateLabel and DeleteLabel over projectLocks, with the Authorizer as
// bootstrap wires them, the first held at a gate past its write. Every
// wait has a deadline.

// labelStore is what a write on a label reads and writes through:
// project's store, or one held at a gate.
type labelStore interface {
	projectapp.LabelCreator
	projectapp.LabelUpdater
	projectapp.LabelDeleter
}

// labelWrittenHolding stops a write on a label past its write: once it
// holds the workspace FOR SHARE, the project FOR NO KEY UPDATE and the rows
// it wrote, before its commit.
type labelWrittenHolding struct {
	*projectpg.Store
	gate *gate
}

func (s labelWrittenHolding) CreateLabel(ctx context.Context, r projectapp.LabelRow) (projectdomain.Label, error) {
	created, err := s.Store.CreateLabel(ctx, r)
	if err != nil {
		return projectdomain.Label{}, err
	}
	return created, s.gate.wait(ctx)
}

func (s labelWrittenHolding) UpdateLabel(ctx context.Context, id uuid.UUID, p projectdomain.LabelPatch, by uuid.UUID,
	now time.Time) (projectdomain.Label, error) {
	changed, err := s.Store.UpdateLabel(ctx, id, p, by, now)
	if err != nil {
		return projectdomain.Label{}, err
	}
	return changed, s.gate.wait(ctx)
}

func (s labelWrittenHolding) DeleteLabel(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := s.Store.DeleteLabel(ctx, id, by, now); err != nil {
		return err
	}
	return s.gate.wait(ctx)
}

// worldLabels are the labels seedLabelsOf gives Web and Ops, by name, each
// with the name of the label it is under, "" at the top: Bug with UI under
// it, Feature and Docs at the top. Ops's are Web's names in another project,
// which no write on Web's touches.
var worldLabels = map[string]string{"Bug": "", "UI": "Bug", "Feature": "", "Docs": ""}

// seedLabelsOf stores worldLabels in project through the project store, by
// alice, an admin of Web and of Ops, whom the rules let write their labels,
// each parent before the labels under it, reads them back (labelsOf), and
// answers their ids by name.
func (w memberWorld) seedLabelsOf(t *testing.T, project uuid.UUID) map[string]uuid.UUID {
	t.Helper()
	var workspace uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT workspace_id FROM projects WHERE id = $1", project).Scan(&workspace); err != nil {
		t.Fatal(err)
	}
	ids, store := map[string]uuid.UUID{}, projectpg.New(w.pool)
	for _, name := range []string{"Bug", "UI", "Feature", "Docs"} {
		ids[name] = uuid.NewV7()
		var parent *uuid.UUID
		if p := worldLabels[name]; p != "" {
			id := ids[p]
			parent = &id
		}
		if _, err := store.CreateLabel(pgtest.Soon(t), projectapp.LabelRow{ID: ids[name], WorkspaceID: workspace, ProjectID: project, ParentID: parent,
			Name: name, SortOrder: float64(len(ids)), CreatedBy: w.ids["alice"], Now: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	if got := w.labelsOf(t, project); !maps.Equal(got, worldLabels) {
		t.Fatalf("the labels seeded in %s read back as %v; want %v", project, got, worldLabels)
	}
	return ids
}

// labelsOf is project's undeleted labels by name, each with the name of the
// label it is under, "" at the top.
func (w memberWorld) labelsOf(t *testing.T, project uuid.UUID) map[string]string {
	t.Helper()
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT l.name, coalesce(p.name, '') FROM labels l LEFT JOIN labels p ON p.id = l.parent_id
		WHERE l.project_id = $1 AND l.deleted_at IS NULL`, project)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	labels := map[string]string{}
	for rows.Next() {
		var name, parent string
		if err := rows.Scan(&name, &parent); err != nil {
			t.Fatal(err)
		}
		labels[name] = parent
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return labels
}

// labelWrite is bob's write on Web's labels, through project's use case
// over a store, with projectLocks, as project.New wires it, given the ids
// of Web's labels by name; apply is what it does to Web's labelsOf.
type labelWrite struct {
	name  string
	run   func(ctx context.Context, w memberWorld, labels labelStore, ids map[string]uuid.UUID) error
	apply func(labels map[string]string)
}

// labelCreates is the creation of the label name under parent, "" at the
// top.
func labelCreates(name, parent string) labelWrite {
	where := "under " + parent
	if parent == "" {
		where = "at the top"
	}
	return labelWrite{"creates " + name + " " + where, func(ctx context.Context, w memberWorld, labels labelStore,
		ids map[string]uuid.UUID) error {
		in := projectdomain.LabelCreate{Name: name}
		if parent != "" {
			id := ids[parent]
			in.ParentID = &id
		}
		_, err := projectapp.NewCreateLabel(w.projectLocks(), labels, w.tx(), clock.System{}).Execute(w.asBob(ctx), w.web, in)
		return err
	}, func(labels map[string]string) { labels[name] = parent }}
}

// labelMoves puts the label name under parent.
func labelMoves(name, parent string) labelWrite {
	return labelWrite{"moves " + name + " under " + parent, func(ctx context.Context, w memberWorld, labels labelStore,
		ids map[string]uuid.UUID) error {
		id := ids[parent]
		_, err := projectapp.NewUpdateLabel(w.projectLocks(), labels, w.tx(), clock.System{}).Execute(w.asBob(ctx), ids[name],
			projectdomain.LabelPatch{SetParent: true, ParentID: &id})
		return err
	}, func(labels map[string]string) { labels[name] = parent }}
}

// labelDeletes deletes the label name, and the labels under it with it.
func labelDeletes(name string) labelWrite {
	return labelWrite{"deletes " + name, func(ctx context.Context, w memberWorld, labels labelStore, ids map[string]uuid.UUID) error {
		return projectapp.NewDeleteLabel(w.projectLocks(), labels, w.tx(), clock.System{}).Execute(w.asBob(ctx), ids[name])
	}, func(labels map[string]string) {
		for n, parent := range labels {
			if n == name || parent == name {
				delete(labels, n)
			}
		}
	}}
}

// parentRefused is the 422 a parent the rules of 3.16 refuse answers, for
// why.
func parentRefused(why string) error {
	return shared.Invalid(shared.FieldError{Field: "parent_id", Code: shared.FieldNotAllowed, Message: why})
}

// sameLabelOutcome is sameOutcome, the refused fields too.
func sameLabelOutcome(err, want error) bool {
	var got, refused *shared.Error
	return sameOutcome(err, want) && (want == nil || errors.As(err, &got) && errors.As(want, &refused) && slices.Equal(got.Fields, refused.Fields))
}

// Two writes on Web's labels at once serialize on Web's row (M3 design
// 3.6, 3.16; 9.3, interleaving 11), in both orders. The first holds acme
// FOR SHARE, Web FOR NO KEY UPDATE and the rows it wrote, and waits at its
// gate past its write. The second shares acme and waits for Web's row:
// neither writes a row of projects, so only that lock's wait satisfies the
// probe. Once the first has committed, the second decides on what it
// committed, its parent and the label's children read under the lock:
//   - Feature under Docs and Docs under Feature: the second's parent has a
//     parent now; no label is ever under a label under another, and none
//     under itself;
//   - Icons created under Feature and Feature moved under Docs: created
//     first, Feature has a label under it and takes no parent; moved
//     first, Feature has a parent and takes no label under it;
//   - QA and qa created: the second's name is taken, in any case;
//   - Feature moved under Docs and under Bug: the second's own label moved
//     while it waited, and the second moves it again, from where the first
//     put it, to where it asks;
//   - Bug deleted, and Feature moved or Icons created under it (convention
//     5's probe): deleted first, Bug is no label of the project any more;
//     the other first, the deletion's one statement takes the label just
//     put under Bug with Bug and UI, under the lock that every write of
//     Web's labels takes;
//   - QA created at the top and Icons under Bug: both are made, the second
//     at the time its clock gives once it holds the lock.
//
// Web's labels are then as the first left them, and the second's too when
// it succeeded, at a moment no earlier than the gate's opening; Ops's as
// they were.
func TestLabelWritesOnOneProjectSerialize(t *testing.T) {
	ofTheProject := parentRefused("must be a label of the project")
	withoutParent := parentRefused("must be a label without a parent: labels have two levels")
	childless := parentRefused("must be null: the label has labels under it, and labels have two levels")
	for _, tt := range []struct {
		a, b               labelWrite
		ifAFirst, ifBFirst error // the second's answer
	}{
		{labelMoves("Feature", "Docs"), labelMoves("Docs", "Feature"), withoutParent, withoutParent},
		{labelCreates("Icons", "Feature"), labelMoves("Feature", "Docs"), childless, withoutParent},
		{labelCreates("QA", ""), labelCreates("qa", ""), projectdomain.ErrLabelNameTaken, projectdomain.ErrLabelNameTaken},
		{labelMoves("Feature", "Docs"), labelMoves("Feature", "Bug"), nil, nil},
		{labelDeletes("Bug"), labelMoves("Feature", "Bug"), ofTheProject, nil},
		{labelDeletes("Bug"), labelCreates("Icons", "Bug"), ofTheProject, nil},
		{labelCreates("QA", ""), labelCreates("Icons", "Bug"), nil, nil},
	} {
		for _, aFirst := range []bool{true, false} {
			first, second, want := tt.a, tt.b, tt.ifAFirst
			if !aFirst {
				first, second, want = tt.b, tt.a, tt.ifBFirst
			}
			t.Run(first.name+", then "+second.name, func(t *testing.T) {
				w := newMemberWorld(t)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				ids := w.seedLabelsOf(t, w.web)
				w.seedLabelsOf(t, w.ops)
				labels, ops := w.labelsOf(t, w.web), w.labelsOf(t, w.ops)
				g := newGate()
				done := run(func() error { return first.run(ctx, w, labelWrittenHolding{projectpg.New(w.pool), g}, ids) })
				held(t, ctx, g, done, "the first write")
				answered := run(func() error { return second.run(ctx, w, projectpg.New(w.pool), ids) })
				pgtest.WaitForLockWaitOn(t, w.pool, "projects", 5*time.Second)
				opened := time.Now().Truncate(time.Microsecond)
				close(g.open)

				if err := result(t, ctx, done, "the first write"); err != nil {
					t.Errorf("%s = %v, want it done", first.name, err)
				}
				if err := result(t, ctx, answered, "the second write"); !sameLabelOutcome(err, want) {
					t.Errorf("%s = %v, want %v as its first problem", second.name, err, want)
				}
				first.apply(labels)
				if want == nil {
					second.apply(labels)
					var latest time.Time
					err := w.pool.QueryRow(pgtest.Soon(t), "SELECT max(updated_at) FROM labels WHERE project_id = $1", w.web).Scan(&latest)
					if err != nil || latest.Before(opened) {
						t.Errorf("Web's last write at %v (%v); want one no earlier than the gate's opening, %v", latest, err, opened)
					}
				}
				if got := w.labelsOf(t, w.web); !maps.Equal(got, labels) {
					t.Errorf("Web's labels after both: %v; want %v", got, labels)
				}
				if got := w.labelsOf(t, w.ops); !maps.Equal(got, ops) {
					t.Errorf("Ops's labels after both: %v; want them as they were, %v", got, ops)
				}
			})
		}
	}
}
````

- [ ] **Step 2: 每个写只写它的行；竞争和锁强度**

`server/internal/bootstrap/label_rows_test.go`（新文件，90 行）：

````file server/internal/bootstrap/label_rows_test.go
package bootstrap

import (
	"maps"
	"net/http"
	"testing"
	"uuid"
)

// Each write on a label, as bootstrap wires it, changes no row but the
// rows it writes (M3 design 3.16): on memberWorld, whose acme has Web and
// Ops, each with worldLabels (seedLabelsOf), bob, Web's admin, creates QA
// at the top of Web, Icons under Feature and Fonts under Docs, renames
// Feature, moves UI to the top, deletes Fonts, moves Docs, whose one label
// under it is deleted, under Bug, and deletes Feature, which takes Icons
// with it, and Docs, whose Fonts, deleted before, keeps its deletion; one
// write after another. Between them, his move of Bug, which has UI under
// it, under Docs (422 parent_id not_allowed) and his creation of a label
// named as Bug in another case (409 project.label_name_taken) write
// nothing. After each write, every row of every table but the ones it
// writes is as it was before it: Web's other labels, the deleted ones
// among them, and Ops's, of the same names. The rows left out are its new
// label, the label it names, and the undeleted labels under the one it
// deletes; a refused write leaves none out.
func TestEachLabelWriteChangesItsRowsAlone(t *testing.T) {
	w := newMemberWorld(t)
	ids := w.seedLabelsOf(t, w.web)
	w.seedLabelsOf(t, w.ops)
	labels, feature := "/api/v0/projects/"+w.web.String()+"/labels", "/api/v0/labels/"+ids["Feature"].String()
	docs := "/api/v0/labels/" + ids["Docs"].String()
	for _, s := range []struct {
		name, method string
		path         string // "" for the label of the first row it writes, one an earlier write made
		body         string
		status       int
		writes       func() []uuid.UUID // the rows it writes that are there before it
		creates      string             // the name of the label it creates, a row it writes too
		// refusal, when set, is the one problem (field and code) a refused
		// write answers.
		refusal string
	}{
		{"createLabel", http.MethodPost, labels, `{"name":"QA"}`, http.StatusCreated, nil, "QA", ""},
		{"createLabel, under Feature", http.MethodPost, labels, `{"name":"Icons","parent_id":"` + ids["Feature"].String() + `"}`,
			http.StatusCreated, nil, "Icons", ""},
		{"createLabel, under Docs", http.MethodPost, labels, `{"name":"Fonts","parent_id":"` + ids["Docs"].String() + `"}`,
			http.StatusCreated, nil, "Fonts", ""},
		{"updateLabel", http.MethodPatch, feature, `{"name":"Story"}`, http.StatusOK, func() []uuid.UUID { return []uuid.UUID{ids["Feature"]} }, "",
			""},
		{"updateLabel, a label with labels under it given a parent", http.MethodPatch, "/api/v0/labels/" + ids["Bug"].String(),
			`{"parent_id":"` + ids["Docs"].String() + `"}`, http.StatusUnprocessableEntity, nil, "", "parent_id not_allowed"},
		{"updateLabel, to the top", http.MethodPatch, "/api/v0/labels/" + ids["UI"].String(), `{"parent_id":null}`, http.StatusOK,
			func() []uuid.UUID { return []uuid.UUID{ids["UI"]} }, "", ""},
		{"createLabel, a name taken", http.MethodPost, labels, `{"name":"bug"}`, http.StatusConflict, nil, "", ""},
		{"deleteLabel, a label under another", http.MethodDelete, "", "", http.StatusNoContent,
			func() []uuid.UUID { return []uuid.UUID{labelID(t, w.pool, w.web, "Fonts")} }, "", ""},
		{"updateLabel, a label whose one label under it is deleted given a parent", http.MethodPatch, docs,
			`{"parent_id":"` + ids["Bug"].String() + `"}`, http.StatusOK, func() []uuid.UUID { return []uuid.UUID{ids["Docs"]} }, "", ""},
		{"deleteLabel", http.MethodDelete, feature, "", http.StatusNoContent,
			func() []uuid.UUID { return []uuid.UUID{ids["Feature"], labelID(t, w.pool, w.web, "Icons")} }, "", ""},
		{"deleteLabel, a label whose label under it is deleted", http.MethodDelete, docs, "", http.StatusNoContent,
			func() []uuid.UUID { return []uuid.UUID{ids["Docs"]} }, "", ""},
	} {
		var writes []uuid.UUID
		if s.writes != nil {
			writes = s.writes()
		}
		path := s.path
		if path == "" {
			path = "/api/v0/labels/" + writes[0].String()
		}
		before := rowsBut(t, w.pool, writes)
		var body []byte
		if s.body != "" {
			body = []byte(s.body)
		}
		req := newRequest(t, s.method, w.base+path, w.tokens["bob"], body)
		w.contract.CheckRequest(t, req)
		res, answer := send(t, req)
		w.contract.CheckResponse(t, req, res)
		if res.StatusCode != s.status || s.refusal != "" && oneError(t, answer) != s.refusal {
			t.Fatalf("%s = %d %s, want %d %s", s.name, res.StatusCode, answer, s.status, s.refusal)
		}
		if s.creates != "" {
			writes = append(writes, labelID(t, w.pool, w.web, s.creates))
		}
		if after := rowsBut(t, w.pool, writes); !maps.Equal(after, before) {
			t.Errorf("%s: every other row after it:\n%v\nwant them as they were:\n%v", s.name, after, before)
		}
	}
}
````

`server/internal/bootstrap/project_row_races_test.go`（修改，11 处）：

````old server/internal/bootstrap/project_row_races_test.go
// bob renames Web's QA, deletes it, or makes it Web's default; or bob
// creates a state in Web, a write addressed by its project, as leaving is.
// gina is Web's member and acme's admin; alice, Web's other admin, stays;
// QA is of the completed group, beside Done; nothing else refuses each
// write.
````
````new server/internal/bootstrap/project_row_races_test.go
// bob renames Web's QA, deletes it, or makes it Web's default; bob renames
// Web's label Feature or deletes it, of the labels seedLabelsOf gives Web
// for these writes alone; or bob creates a state or a label in Web, a write
// addressed by its project, as leaving is. gina is Web's member and acme's
// admin; alice, Web's other admin, stays; QA is of the completed group,
// beside Done; Feature is at the top, with no label under it; nothing else
// refuses each write.
````

````old server/internal/bootstrap/project_row_races_test.go
	// the caller's own for leaving. A write on a state changes none: it
	// carries its caller, whose membership of acme the lock test reads, and
	// state is the name of Web's state it changes instead. A creation
	// changes neither and carries neither; the races end its caller's
	// membership through by, as every write's.
	member, state      string
	method, path, body string // path: %s the row's id for a path of one (rowPaths), else Web's
	status             int    // its answer, alone
	notFound           string // the code of its 404
	target             bool   // it changes the member's role: it shares his membership of acme (convention 3)
	clears             bool   // it makes its state Web's default: it writes Web's default, Backlog, before its state
	creates            bool   // it creates a state of Web: it has no row of its own to wait on
````
````new server/internal/bootstrap/project_row_races_test.go
	// the caller's own for leaving. A write on a state or a label changes
	// none: it carries its caller, whose membership of acme the lock test
	// reads, and state or label is the name of Web's state or label it
	// changes instead. A creation changes neither and carries neither; the
	// races end its caller's membership through by, as every write's.
	member, state, label string
	method, path, body   string // path: %s the row's id for a path of one (rowPaths), else Web's
	status               int    // its answer, alone
	notFound             string // the code of its 404
	target               bool   // it changes the member's role: it shares his membership of acme (convention 3)
	clears               bool   // it makes its state Web's default: it writes Web's default, Backlog, before its state
	creates              bool   // it creates a state or a label of Web: it has no row of its own to wait on
````

````old server/internal/bootstrap/project_row_races_test.go
		creates: true},
````
````new server/internal/bootstrap/project_row_races_test.go
		creates: true},
	{op: "updateLabel", by: "bob", member: "bob", label: "Feature", method: http.MethodPatch, path: "/api/v0/labels/%s",
		body: `{"name":"Story"}`, status: http.StatusOK, notFound: "project.label_not_found"},
	{op: "deleteLabel", by: "bob", member: "bob", label: "Feature", method: http.MethodDelete, path: "/api/v0/labels/%s",
		status: http.StatusNoContent, notFound: "project.label_not_found"},
	{op: "createLabel", by: "bob", method: http.MethodPost, path: "/api/v0/projects/%s/labels", body: `{"name":"QA"}`,
		status: http.StatusCreated, notFound: "project.not_found", creates: true},
````

````old server/internal/bootstrap/project_row_races_test.go
		t.Fatalf("%s creates a state: it has no row of its own", m.op)
````
````new server/internal/bootstrap/project_row_races_test.go
		t.Fatalf("%s creates a row of Web: it has no row of its own", m.op)
````

````old server/internal/bootstrap/project_row_races_test.go
		return "states", stateID(t, w.pool, w.web, m.state)
	}
	return "project_members", w.membership(t, w.web, m.member)
````
````new server/internal/bootstrap/project_row_races_test.go
		return "states", stateID(t, w.pool, w.web, m.state)
	}
	if m.label != "" {
		return "labels", labelID(t, w.pool, w.web, m.label)
	}
	return "project_members", w.membership(t, w.web, m.member)
}

// labelsFor gives Web seedLabelsOf's labels when m writes one of them.
func (m rowWrite) labelsFor(t *testing.T, w memberWorld) {
	t.Helper()
	if m.label != "" {
		w.seedLabelsOf(t, w.web)
	}
````

````old server/internal/bootstrap/project_row_races_test.go
// lock, of the row it names or, leaving or creating a state, of Web's
// workspace, and waits for the row the other holds. Once the other commits,
// the write is 404; the row the other changed is as it left it, and every
// other row as it was, no state created among them. Web is private: bob or
// dave, his membership ended, does not see it.
````
````new server/internal/bootstrap/project_row_races_test.go
// lock, of the row it names or, leaving or creating a state or a label, of
// Web's workspace, and waits for the row the other holds. Once the other
// commits, the write is 404; the row the other changed is as it left it,
// and every other row as it was, no state or label created among them. Web
// is private: bob or dave, his membership ended, does not see it.
````

````old server/internal/bootstrap/project_row_races_test.go
			case c.ofRow && m.state == "" && m.member == m.by:
````
````new server/internal/bootstrap/project_row_races_test.go
			case c.ofRow && m.state == "" && m.label == "" && m.member == m.by:
````

````old server/internal/bootstrap/project_row_races_test.go
			case c.membership && m.state != "":
				continue // a state does not end
````
````new server/internal/bootstrap/project_row_races_test.go
			case c.membership && (m.state != "" || m.label != ""):
				continue // a state or a label does not end
````

````old server/internal/bootstrap/project_row_races_test.go
				w := newPrivateWorld(t)
````
````new server/internal/bootstrap/project_row_races_test.go
				w := newPrivateWorld(t)
				m.labelsFor(t, w)
````

````old server/internal/bootstrap/project_row_races_test.go
			// and the two creations' of TestStateWritesOnOneProjectSerialize.
````
````new server/internal/bootstrap/project_row_races_test.go
			// and the two creations' of TestStateWritesOnOneProjectSerialize and
			// TestLabelWritesOnOneProjectSerialize.
````

````old server/internal/bootstrap/project_row_races_test.go
			w := newMemberWorld(t)
````
````new server/internal/bootstrap/project_row_races_test.go
			w := newMemberWorld(t)
			m.labelsFor(t, w)
````

- [ ] **Step 3: 测试和 lint**

Run: `go -C server test -count=5 -race -run 'TestLabelWritesOnOneProjectSerialize$|TestStateWritesOnOneProjectSerialize$|TestAWriteOnARowUnderAProjectFindsWhatChangedMeanwhile$|TestEachLockOfAWriteOnARowUnderAProjectIsItsStrength$|TestEachWriteOnAProjectSharesItsWorkspaceFirst$|TestEachLabelWriteChangesItsRowsAlone$' ./internal/bootstrap/`
Expected: `ok`，输出里没有 `40P01`（原型上 `race5.sh` 跑这六个和另外八个，各 5 次全部通过：670 个子测试，233 秒）。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/bootstrap/interleaving_labels_test.go server/internal/bootstrap/label_rows_test.go server/internal/bootstrap/project_row_races_test.go
```
```bash
git commit -m "test(M3/P7b): the writes on a project's labels serialize on its row

Two writes on one project's labels serialize on the project's row in
either order: two labels each moved under the other leave two levels
(interleaving 11), a label moved under another and a label created
under it, two labels of one name in different cases, a label moved
twice, and two labels created, the second at its clock's time under the
lock. A parent deleted while a label is moved or created under it is no
label any more, or takes the label with it in its one statement, under
the lock every write of the project's labels takes. The races and
lock strengths of the writes on a row under a project take the label
writes in, and each label write changes its rows alone.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 没有产品代码；它的测试在别的 Task 的变异表里（"（Task 10 起）"）。

---

### Task 11: 故事 P7 的接口版本；P4、W2、W3 的标签；端到端的过渡结束；文档

**Files:**
- Create: `e2e/stories/project/p7-labels.spec.ts`
- Modify: `docs/v0/plane-diff.md`、`docs/v0/v0-design.md`、`e2e/fixtures/api.ts`、`e2e/fixtures/assert/project.ts`、`e2e/fixtures/assert/workspace.ts`、`e2e/stories/project/p4-archive.spec.ts`、`e2e/stories/workspace/w2-landing.spec.ts`、`e2e/stories/workspace/w3-workspace-settings.spec.ts`、`server/internal/bootstrap/workspace_deletion_catalog_test.go`

**Interfaces:**
- Produces（spec 2.11；M3 设计 2、11.5）：
  - `e2e/fixtures/api.ts`：`Label`、`LabelCreate`、`LabelUpdate` 类型（来自 `schema.gen.ts`）、`createLabel(api, token, projectId, body)`。
  - `e2e/fixtures/assert/project.ts`：删去 Task 1 的排除：`expectProjectDeleted` 照旧读外键指向 `projects` 的每个表（目录的列表），只是不数项目之前已经删除的行（如故事删除的标签）；`LabelRow`、`expectLabels(db, projectId, want)`（删除的也在内，按 `sort_order`、再按名称：名称、颜色、父标签的名称、`sort_order`、是否删除、最后写它的账户；每一行在它的项目的工作区、父标签是同一个项目的，删除的行在它最后一次写的时刻删除）。
  - `e2e/fixtures/assert/workspace.ts`：`workspaceTables`（随工作区删除的表）、`deletedAloneTables`（项目删除时随它删除的表）加 `labels`。
  - `p7-labels.spec.ts`（新）：故事 P7 的接口版本（项目的管理员建 Bug、Feature，各在之前的之后；另一个管理员建 UI，前者把它放到 Bug 下；后者把它改名 Widgets、把 Feature 移到 Bug 之前；bug 被占用；颜色太长、第三层、另一个项目的父标签、自己的父标签、有子标签的标签设父标签、成员的写，各被拒绝、什么都不改；Bug 和 Widgets 在同一时刻删除，之后都找不到，bug 又可以用，在 Feature 之后：删除了的标签的位置不再算；成员按次序列出；另一个项目有自己的 Bug；已归档的项目的标签照常列出、创建（也在给的位置）、修改）。页面版本随 P11 的标签设置页。
  - P4：已归档的 Web 加标签（Bug、Bug 下的 UI、Feature），Feature 先删；Web 删除时 Bug、UI 随它在同一时刻删除，Feature 保留它的时刻。W2：First 的 Web 有标签 Bug，First 的删除连带它。W3：成员（Web 的负责人，因此是它的管理员）给 Acme 和 Other 的 Web 各加 Bug、Bug 下的 UI，给 Old 加 Bug；Acme 删除时它的 Web 的标签由管理员删除，Old 的标签保留 Old 删除的时刻，Other 的不变。
  - `workspace_deletion_catalog_test.go`：`keysTo` 的说明：指向自己的外键（如 `labels.parent_id`）与别的外键一样跟随。
  - `docs/v0/v0-design.md` 5.3 的标签行；`docs/v0/plane-diff.md`：二·按表的 `labels`、第四节的 P7b 行（3.20）。

**Tests:**
- `make e2e`：70 个故事全部通过，P7 和标签的 P4、W2、W3 在内。
- `TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt` 等照旧通过（`keysTo` 只改说明）。

- [ ] **Step 1: 端到端的夹具**

`e2e/fixtures/api.ts`（修改，2 处）：

````old e2e/fixtures/api.ts
export type StateUpdate = components["schemas"]["StateUpdate"];
````
````new e2e/fixtures/api.ts
export type StateUpdate = components["schemas"]["StateUpdate"];
export type Label = components["schemas"]["Label"];
export type LabelCreate = components["schemas"]["LabelCreate"];
export type LabelUpdate = components["schemas"]["LabelUpdate"];
````

````old e2e/fixtures/api.ts
  return data;
}

/**
 * Runs make, which makes a story's projects, between two projects of another workspace of the caller of token, one
````
````new e2e/fixtures/api.ts
  return data;
}

/** Creates a label in the project of projectId with the bearer token given, an admin's of the project, and returns it. */
export async function createLabel(api: Api, token: string, projectId: string, body: LabelCreate): Promise<Label> {
  const { data, error, response } = await api.POST("/api/v0/projects/{project_id}/labels", {
    params: { path: { project_id: projectId } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `create the label ${body.name} in ${projectId}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`createLabel ${body.name} answered 201 without the label`);
  }
  return data;
}

/**
 * Runs make, which makes a story's projects, between two projects of another workspace of the caller of token, one
````

`e2e/fixtures/assert/project.ts`（修改，5 处）：

````old e2e/fixtures/assert/project.ts
 * same account, every row under it: of each table whose foreign key names projects (the catalog's list, so a table a
 * later phase adds is read too), by its project_id. Each table has such a row, and none is left undeleted.
````
````new e2e/fixtures/assert/project.ts
 * same account, every row under it that was not deleted before: of each table whose foreign key names projects (the
 * catalog's list, so a table a later phase adds is read too), by its project_id. Each table has such a row, and none
 * is left undeleted; a row deleted before the project, as a label a story deleted, is not counted.
````

````old e2e/fixtures/assert/project.ts
  expect(project, `the project ${projectId}`).toEqual({ deleted: true, by: adminEmail });
  // The labels are left out until the stories can write one: P7b's createLabel gives them the way, and P4 and W3 then
  // write and check them.
````
````new e2e/fixtures/assert/project.ts
  expect(project, `the project ${projectId}`).toEqual({ deleted: true, by: adminEmail });
````

````old e2e/fixtures/assert/project.ts
      WHERE c.contype = 'f' AND c.confrelid = 'projects'::regclass AND c.conrelid <> 'labels'::regclass ORDER BY 1`
````
````new e2e/fixtures/assert/project.ts
      WHERE c.contype = 'f' AND c.confrelid = 'projects'::regclass ORDER BY 1`
````

````old e2e/fixtures/assert/project.ts
                count(*) FILTER (WHERE t.deleted_at IS DISTINCT FROM p.deleted_at
                                    OR t.updated_by_id IS DISTINCT FROM p.updated_by_id)::int AS other
````
````new e2e/fixtures/assert/project.ts
                count(*) FILTER (WHERE (t.deleted_at IS NULL OR t.deleted_at >= p.deleted_at)
                                   AND (t.deleted_at IS DISTINCT FROM p.deleted_at
                                        OR t.updated_by_id IS DISTINCT FROM p.updated_by_id))::int AS other
````

````old e2e/fixtures/assert/project.ts
      }))
  );
}

````
````new e2e/fixtures/assert/project.ts
      }))
  );
}

/** A label of a project as expectLabels reads it. */
export interface LabelRow {
  name: string;
  color: string;
  /** The name of the label it is under; null at the top. */
  parent: string | null;
  sort_order: number;
  deleted: boolean;
  /** The address of the account that wrote the label last: created, changed or deleted it. */
  by: string;
}

/**
 * P4, P7, W3: the labels of the project of projectId, deleted ones too, are exactly want, by sort order, then name.
 * Each is a row of the project's workspace, its parent a label of the same project, and a deleted one was deleted at
 * the moment of its last write, by its writer (M3 design 3.16).
 */
export async function expectLabels(db: Database, projectId: string, want: LabelRow[]): Promise<void> {
  expect(
    await db.query(
      `SELECT l.name, l.color, pl.name AS parent, l.sort_order, l.deleted_at IS NOT NULL AS deleted, b.email AS by,
              l.workspace_id = p.workspace_id AND (pl.id IS NULL OR pl.project_id = l.project_id) AS in_its_project,
              coalesce(l.deleted_at = l.updated_at, true) AS deleted_with_its_last_write
         FROM labels l
         JOIN projects p ON p.id = l.project_id
         JOIN users b ON b.id = l.updated_by_id
         LEFT JOIN labels pl ON pl.id = l.parent_id
        WHERE l.project_id = $1 ORDER BY l.sort_order, l.name COLLATE "C"`,
      [projectId]
    ),
    `the labels of ${projectId}`
  ).toEqual(
    want
      .toSorted((a, b) => a.sort_order - b.sort_order || (a.name < b.name ? -1 : 1))
      .map(({ name, color, parent, sort_order, deleted, by }) => ({
        name,
        color,
        parent,
        sort_order,
        deleted,
        by,
        in_its_project: true,
        deleted_with_its_last_write: true,
      }))
  );
}

````

`e2e/fixtures/assert/workspace.ts`（修改，5 处）：

````old e2e/fixtures/assert/workspace.ts
/** The tables whose rows belong to a workspace and are deleted with it (M3 design 3.6, 4.12); P7b adds the labels. */
````
````new e2e/fixtures/assert/workspace.ts
/** The tables whose rows belong to a workspace and are deleted with it (M3 design 3.6, 4.12). */
````

````old e2e/fixtures/assert/workspace.ts
  "project_user_properties",
  "states",
] as const;
````
````new e2e/fixtures/assert/workspace.ts
  "project_user_properties",
  "states",
  "labels",
] as const;
````

````old e2e/fixtures/assert/workspace.ts
 * memberships, its members' display settings and its states, when the project is deleted (P4b).
````
````new e2e/fixtures/assert/workspace.ts
 * memberships, its members' display settings, its states and its labels, when the project is deleted (P4b, P7b).
````

````old e2e/fixtures/assert/workspace.ts
  "project_user_properties",
  "states",
];
````
````new e2e/fixtures/assert/workspace.ts
  "project_user_properties",
  "states",
  "labels",
];
````

````old e2e/fixtures/assert/workspace.ts
 * settings, its projects, their memberships, their members' display settings and their states. Each table has
 * such a row; none is left undeleted; each table of deletedAlone, whose rows the story deleted alone, has rows
 * deleted earlier, which kept their moment, and every row of the others carries the workspace's.
````
````new e2e/fixtures/assert/workspace.ts
 * settings, its projects, their memberships, their members' display settings, their states and their labels. Each
 * table has such a row; none is left undeleted; each table of deletedAlone, whose rows the story deleted alone, has
 * rows deleted earlier, which kept their moment, and every row of the others carries the workspace's.
````

- [ ] **Step 2: 故事**

`e2e/stories/project/p7-labels.spec.ts`（新文件，250 行）：

````file e2e/stories/project/p7-labels.spec.ts
import {
  addProjectMembers,
  amidAnotherWorkspace,
  answer,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type Label,
  type LabelCreate,
  type LabelUpdate,
} from "../../fixtures/api";
import { expectLabels, type LabelRow } from "../../fixtures/assert/project";
import { bearer, newAccount } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// P7, a project's labels (M3 design 2, 3.16): creating them, putting one
// under another, renaming and moving one, deleting one with the label under
// it, with what the names and the two levels keep. The page version comes
// with the labels' settings page (P11).

/** The labels of the project of projectId that the caller of token lists, in their order. */
async function listLabels(api: Api, token: string, projectId: string): Promise<Label[]> {
  const { data, error, response } = await api.GET("/api/v0/projects/{project_id}/labels", {
    params: { path: { project_id: projectId } },
    headers: bearer(token),
  });
  expect(response.status, `list ${projectId}'s labels: ${JSON.stringify(error)}`).toBe(200);
  return data?.data ?? [];
}

/** The writes on labels, each by the caller of token: creating one in a project, changing and deleting one named by its id. */
function writes(api: Api) {
  return {
    create: async (token: string, projectId: string, body: LabelCreate) => {
      const { error, response } = await api.POST("/api/v0/projects/{project_id}/labels", {
        params: { path: { project_id: projectId } },
        body,
        headers: bearer(token),
      });
      return answer(response, error);
    },
    update: async (token: string, labelId: string, body: LabelUpdate) => {
      const { error, response } = await api.PATCH("/api/v0/labels/{label_id}", {
        params: { path: { label_id: labelId } },
        body,
        headers: bearer(token),
      });
      return answer(response, error);
    },
    remove: async (token: string, labelId: string) => {
      const { error, response } = await api.DELETE("/api/v0/labels/{label_id}", {
        params: { path: { label_id: labelId } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
  };
}

/** rows with the label of name changed as to says. */
function changed(rows: LabelRow[], name: string, to: Partial<LabelRow>): LabelRow[] {
  return rows.map((l) => (l.name === name ? { ...l, ...to } : l));
}

/** The refusal of a parent_id that 3.16's rules do not allow. */
const parentRefused = { status: 422, code: "validation_failed", errors: [{ field: "parent_id", code: "not_allowed" }] };

test("P7 (API): an admin of a project creates Bug and Feature, each after the labels before it; another admin creates UI, which the first puts under Bug; the other renames it Widgets and moves Feature before Bug; bug, in another case, is taken, and a color too long, a third level, a parent of another project, a label its own parent, a parent for a label with a label under it and a member's writes are refused, each changing nothing; Bug is deleted with Widgets at one moment, after which neither is found, and bug is free again, after Feature, the deleted labels' places no longer counted; a member lists the labels by their order, another project has a Bug of its own, and an archived project's labels are listed, created, at a place given too, and changed as any other's", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  // ann and mem, acme's members: ann Web's other admin, mem its member.
  const [ann, mem] = await Promise.all(["ann", "mem"].map((label) => newAccount(api, testInfo, label)));
  if (!ann || !mem) {
    throw new Error("the accounts were not registered");
  }
  await Promise.all([ann, mem].map((who) => inviteAndAccept(api, admin.token, slug, who, 15)));
  const web = await amidAnotherWorkspace(api, admin.token, testInfo, () =>
    createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" })
  );
  await addProjectMembers(api, admin.token, web.id, [
    { member_id: ann.id, role: 20 },
    { member_id: mem.id, role: 15 },
  ]);
  const { create, update, remove } = writes(api);
  // A new project has no labels.
  expect(await listLabels(api, mem.token, web.id), "Web's labels as made").toEqual([]);

  // ann creates Bug, Web's first label, at 65535, then Feature after it, at 65535 + 10000; the admin creates UI at the
  // top, after both, without a color (M3 design 4.10).
  const bug = await createLabel(api, ann.token, web.id, { name: "Bug", color: "#EF4444" });
  expect(
    [bug.project_id, bug.workspace_id === web.workspace_id, bug.name, bug.color, bug.parent_id, bug.sort_order],
    "Bug as created"
  ).toEqual([web.id, true, "Bug", "#EF4444", null, 65535]);
  const feature = await createLabel(api, ann.token, web.id, { name: "Feature", color: "#22C55E" });
  const ui = await createLabel(api, admin.token, web.id, { name: "UI" });
  expect([feature.sort_order, ui.sort_order, ui.color], "Feature and UI as created").toEqual([75535, 85535, ""]);
  let labels: LabelRow[] = [
    { name: "Bug", color: "#EF4444", parent: null, sort_order: 65535, deleted: false, by: ann.email },
    { name: "Feature", color: "#22C55E", parent: null, sort_order: 75535, deleted: false, by: ann.email },
    { name: "UI", color: "", parent: null, sort_order: 85535, deleted: false, by: admin.email },
  ];
  await expectLabels(db, web.id, labels);

  // ann drags UI under Bug, as the settings page does: its parent changes, its name, color and place kept.
  expect(await update(ann.token, ui.id, { parent_id: bug.id }), "ann puts UI under Bug").toEqual({ status: 200 });
  labels = changed(labels, "UI", { parent: "Bug", by: ann.email });
  await expectLabels(db, web.id, labels);
  // The admin renames UI Widgets, its parent and place kept, and moves Feature before Bug.
  expect(await update(admin.token, ui.id, { name: "Widgets" }), "the admin renames UI").toEqual({ status: 200 });
  labels = changed(labels, "UI", { name: "Widgets", by: admin.email });
  await expectLabels(db, web.id, labels);
  expect(await update(admin.token, feature.id, { sort_order: 55535 }), "the admin moves Feature").toEqual({
    status: 200,
  });
  labels = changed(labels, "Feature", { sort_order: 55535, by: admin.email });
  await expectLabels(db, web.id, labels);
  // mem lists them by their order, not the order they were made in, Widgets under Bug.
  expect(
    (await listLabels(api, mem.token, web.id)).map((l) => [l.name, l.parent_id]),
    "Web's labels as mem lists them"
  ).toEqual([
    ["Feature", null],
    ["Bug", null],
    ["Widgets", bug.id],
  ]);

  // Ops, ann's other project of acme, has a Bug of its own: a name is unique in its project alone.
  const ops = await createProject(api, ann.token, slug, { name: "Ops", identifier: "OPS" });
  const opsBug = await createLabel(api, ann.token, ops.id, { name: "Bug" });
  const opsLabels: LabelRow[] = [
    { name: "Bug", color: "", parent: null, sort_order: 65535, deleted: false, by: ann.email },
  ];

  // Refused, each changing nothing of what was just read: ann's creation of QA with a color of 256 characters, one
  // more than a color has; her creation of bug, Bug in another case, and the admin's renaming of Feature WIDGETS;
  // ann's creation of Icons under Widgets, a third level; her moves of Feature under Ops's Bug, of another project,
  // and under itself; her move of Bug, which has Widgets under it, under Feature; mem's change of Feature and his
  // creation of QA, a member's.
  expect(
    [
      await create(ann.token, web.id, { name: "QA", color: "#".repeat(256) }),
      await create(ann.token, web.id, { name: "bug" }),
      await update(admin.token, feature.id, { name: "WIDGETS" }),
      await create(ann.token, web.id, { name: "Icons", parent_id: ui.id }),
      await update(ann.token, feature.id, { parent_id: opsBug.id }),
      await update(ann.token, feature.id, { parent_id: feature.id }),
      await update(ann.token, bug.id, { parent_id: feature.id }),
      await update(mem.token, feature.id, { name: "Story" }),
      await create(mem.token, web.id, { name: "QA" }),
    ],
    "a long color, bug, WIDGETS, Icons under Widgets, Feature under Ops's Bug and under itself, Bug under Feature, mem's writes"
  ).toEqual([
    { status: 422, code: "validation_failed", errors: [{ field: "color", code: "too_long" }] },
    { status: 409, code: "project.label_name_taken" },
    { status: 409, code: "project.label_name_taken" },
    parentRefused,
    parentRefused,
    parentRefused,
    parentRefused,
    { status: 403, code: "forbidden" },
    { status: 403, code: "forbidden" },
  ]);
  await expectLabels(db, web.id, labels);
  await expectLabels(db, ops.id, opsLabels);

  // ann deletes Bug, and with it Widgets, under it: both at one moment, by her.
  expect(await remove(ann.token, bug.id), "ann deletes Bug").toEqual({ status: 204 });
  labels = changed(changed(labels, "Bug", { deleted: true, by: ann.email }), "Widgets", {
    deleted: true,
    by: ann.email,
  });
  await expectLabels(db, web.id, labels);
  expect(
    await db.query(
      `SELECT count(DISTINCT deleted_at)::int AS moments FROM labels WHERE project_id = $1 AND deleted_at IS NOT NULL`,
      [web.id]
    ),
    "the moments of Bug's and Widgets's deletion"
  ).toEqual([{ moments: 1 }]);

  // Not found, changing nothing: ann's deletion of Bug again and her renaming of Widgets; Icons under Bug, a parent
  // deleted, is refused as one of no project.
  expect(
    [
      await remove(ann.token, bug.id),
      await update(ann.token, ui.id, { name: "UI" }),
      await create(ann.token, web.id, { name: "Icons", parent_id: bug.id }),
    ],
    "ann's deletion of Bug again, her renaming of Widgets, Icons under Bug"
  ).toEqual([
    { status: 404, code: "project.label_not_found" },
    { status: 404, code: "project.label_not_found" },
    parentRefused,
  ]);
  await expectLabels(db, web.id, labels);

  // bug is free again: ann creates it after Feature, at 55535 + 10000; the places of the deleted labels, Widgets's
  // 85535 the greatest of all, no longer count.
  expect((await createLabel(api, ann.token, web.id, { name: "bug" })).sort_order, "bug as created").toBe(65535);
  labels = [...labels, { name: "bug", color: "", parent: null, sort_order: 65535, deleted: false, by: ann.email }];
  await expectLabels(db, web.id, labels);
  expect(
    (await listLabels(api, mem.token, web.id)).map((l) => l.name),
    "Web's labels as mem lists them, Bug deleted"
  ).toEqual(["Feature", "bug"]);

  // ann archives Ops: its labels are listed, created, Runbook at the place she gives, and changed, Runbook moved to the
  // top at last, as any other project's (M3 design 3.19).
  const archived = await api.POST("/api/v0/projects/{project_id}/archive", {
    params: { path: { project_id: ops.id } },
    headers: bearer(ann.token),
  });
  expect(archived.response.status, `ann archives Ops: ${JSON.stringify(archived.error)}`).toBe(200);
  expect(
    (await listLabels(api, ann.token, ops.id)).map((l) => l.name),
    "the labels of Ops, archived"
  ).toEqual(["Bug"]);
  const runbook = await createLabel(api, ann.token, ops.id, {
    name: "Runbook",
    parent_id: opsBug.id,
    sort_order: 70000.5,
  });
  expect(await update(ann.token, opsBug.id, { name: "Incident" }), "ann renames Ops's Bug").toEqual({ status: 200 });
  const archivedOps: LabelRow[] = [
    { name: "Incident", color: "", parent: null, sort_order: 65535, deleted: false, by: ann.email },
    { name: "Runbook", color: "", parent: "Incident", sort_order: 70000.5, deleted: false, by: ann.email },
  ];
  await expectLabels(db, ops.id, archivedOps);
  expect(await update(ann.token, runbook.id, { parent_id: null }), "ann moves Runbook to the top").toEqual({
    status: 200,
  });
  await expectLabels(db, ops.id, changed(archivedOps, "Runbook", { parent: null }));

  // The admin's projects of Elsewhere, made around Web, have no labels: no write on Web's labels wrote another
  // project's.
  const elsewhere = await api.GET("/api/v0/workspaces/{slug}/projects", {
    params: { path: { slug: slugFor(testInfo, "elsewhere") } },
    headers: bearer(admin.token),
  });
  expect(elsewhere.data?.data.length, `Elsewhere's projects: ${JSON.stringify(elsewhere.error)}`).toBe(2);
  await Promise.all((elsewhere.data?.data ?? []).map((project) => expectLabels(db, project.id, [])));
});
````

`e2e/stories/project/p4-archive.spec.ts`（修改，5 处）：

````old e2e/stories/project/p4-archive.spec.ts
  amidAnotherWorkspace,
````
````new e2e/stories/project/p4-archive.spec.ts
  amidAnotherWorkspace,
  createLabel,
````

````old e2e/stories/project/p4-archive.spec.ts
import { expectMember, expectProjectCreated, expectProjectDeleted } from "../../fixtures/assert/project";
````
````new e2e/stories/project/p4-archive.spec.ts
import {
  expectLabels,
  expectMember,
  expectProjectCreated,
  expectProjectDeleted,
  type LabelRow,
} from "../../fixtures/assert/project";
````

````old e2e/stories/project/p4-archive.spec.ts
test("P4 (API): the admin archives a project, which leaves the list for the archived ones and whose fields cannot be updated (409) until it is unarchived; unarchives it and archives it again; then deletes it with its members, settings and states at one moment, after which it is not found and its identifier is free", async ({
````
````new e2e/stories/project/p4-archive.spec.ts
test("P4 (API): the admin archives a project, which leaves the list for the archived ones and whose fields cannot be updated (409) until it is unarchived; unarchives it and archives it again; then labels it and deletes it with its members, settings, states and labels at one moment, a label deleted before keeping its own, after which it is not found and its identifier is free", async ({
````

````old e2e/stories/project/p4-archive.spec.ts
  await expectMember(db, web.id, memberEmail, { role: 15, is_active: true, sort_order: 25535, by: memberEmail });
````
````new e2e/stories/project/p4-archive.spec.ts
  await expectMember(db, web.id, memberEmail, { role: 15, is_active: true, sort_order: 25535, by: memberEmail });
  // The admin labels the archived Web, as any project (M3 design 3.19): Bug, UI under Bug, and Feature, which he
  // deletes before Web.
  const bug = await createLabel(api, admin, web.id, { name: "Bug" });
  await createLabel(api, admin, web.id, { name: "UI", parent_id: bug.id });
  const feature = await createLabel(api, admin, web.id, { name: "Feature" });
  const removed = await api.DELETE("/api/v0/labels/{label_id}", {
    params: { path: { label_id: feature.id } },
    headers: bearer(admin),
  });
  expect(removed.response.status, `delete Feature: ${JSON.stringify(removed.error)}`).toBe(204);
  const label = (name: string, parent: string | null, sort_order: number, deleted: boolean): LabelRow => ({
    name,
    color: "",
    parent,
    sort_order,
    deleted,
    by: adminEmail,
  });
  await expectLabels(db, web.id, [
    label("Bug", null, 65535, false),
    label("UI", "Bug", 75535, false),
    label("Feature", null, 85535, true),
  ]);
````

````old e2e/stories/project/p4-archive.spec.ts
  await expectProjectDeleted(db, web.id, adminEmail);
````
````new e2e/stories/project/p4-archive.spec.ts
  await expectProjectDeleted(db, web.id, adminEmail);
  // Its labels are deleted with it; Feature, deleted before, keeps its moment.
  await expectLabels(db, web.id, [
    label("Bug", null, 65535, true),
    label("UI", "Bug", 75535, true),
    label("Feature", null, 85535, true),
  ]);
  expect(
    await db.query(
      `SELECT l.name, l.deleted_at < p.deleted_at AS before_web
         FROM labels l JOIN projects p ON p.id = l.project_id WHERE p.id = $1 ORDER BY l.sort_order`,
      [web.id]
    ),
    "when Web's labels were deleted"
  ).toEqual([
    { name: "Bug", before_web: false },
    { name: "UI", before_web: false },
    { name: "Feature", before_web: true },
  ]);
````

`e2e/stories/workspace/w2-landing.spec.ts`（修改，3 处）：

````old e2e/stories/workspace/w2-landing.spec.ts
  addProjectMembers,
````
````new e2e/stories/workspace/w2-landing.spec.ts
  addProjectMembers,
  createLabel,
````

````old e2e/stories/workspace/w2-landing.spec.ts
  // the workspace); her display settings; and her project Web with its own rows.
````
````new e2e/stories/workspace/w2-landing.spec.ts
  // the workspace); her display settings; and her project Web with its own rows, its label Bug among them.
````

````old e2e/stories/workspace/w2-landing.spec.ts
  await createProject(api, alice, first, { name: "Web", identifier: "WEB" });
````
````new e2e/stories/workspace/w2-landing.spec.ts
  const web = await createProject(api, alice, first, { name: "Web", identifier: "WEB" });
  await createLabel(api, alice, web.id, { name: "Bug" });
````

`e2e/stories/workspace/w3-workspace-settings.spec.ts`（修改，8 处）：

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
import { createProject, createWorkspace, invite, inviteAndAccept, slugFor, type Workspace } from "../../fixtures/api";
import { expectProjectCreated, expectProjectDeleted } from "../../fixtures/assert/project";
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
import {
  createLabel,
  createProject,
  createWorkspace,
  invite,
  inviteAndAccept,
  slugFor,
  type Workspace,
} from "../../fixtures/api";
import { expectLabels, expectProjectCreated, expectProjectDeleted, type LabelRow } from "../../fixtures/assert/project";
````

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
  // project the admin created with the member its lead; only Acme is deleted. Both exist before either is
  // furnished, and the member is Other's admin and Acme's member: each answer, role and row must be the
  // workspace's own, whichever row the database reads first.
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
  // project the admin created with the member its lead, which the member labelled; only Acme is deleted. Both exist
  // before either is furnished, and the member is Other's admin and Acme's member: each answer, role and row must be
  // the workspace's own, whichever row the database reads first.
````

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
    expect(
      await createProject(api, admin, target, { name: "Web", identifier: "web", project_lead_id: memberId })
    ).toMatchObject({
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
    const project = await createProject(api, admin, target, {
      name: "Web",
      identifier: "web",
      project_lead_id: memberId,
    });
    expect(project).toMatchObject({
````

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
      member_ids: [adminId, memberId],
    });
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
      member_ids: [adminId, memberId],
    });
    // The member, Web's lead and so its admin, labels it: Bug, and UI under Bug.
    const bug = await createLabel(api, member, project.id, { name: "Bug", color: "#EF4444" });
    await createLabel(api, member, project.id, { name: "UI", parent_id: bug.id });
    return project;
````

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
  await furnish(otherWorkspace, 20);
  await furnish(acme, 15);
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
  const otherWeb = await furnish(otherWorkspace, 20);
  const acmeWeb = await furnish(acme, 15);
  /** Web's labels, as the member made them; deleted by the admin, as Acme's deletion writes them. */
  const webLabels = (deleted: boolean): LabelRow[] => [
    { name: "Bug", color: "#EF4444", parent: null, sort_order: 65535, deleted, by: deleted ? adminEmail : memberEmail },
    { name: "UI", color: "", parent: "Bug", sort_order: 75535, deleted, by: deleted ? adminEmail : memberEmail },
  ];
  await expectLabels(db, acmeWeb.id, webLabels(false));
````

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
  // The member moves Old in his sidebar: his display settings in it are his own writing, which Old's deletion must
  // write again as the admin's.
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
  // The member moves Old in his sidebar and labels it: his display settings in it and its label are his own writing,
  // which Old's deletion must write again as the admin's.
  await createLabel(api, member, old.id, { name: "Bug" });
````

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
  await expectWorkspaceDeleted(db, slug, adminEmail, deletedAloneTables);
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
  await expectWorkspaceDeleted(db, slug, adminEmail, deletedAloneTables);
  await expectLabels(db, acmeWeb.id, webLabels(true));
````

````old e2e/stories/workspace/w3-workspace-settings.spec.ts
    navigation_project_limit: 3,
  });
````
````new e2e/stories/workspace/w3-workspace-settings.spec.ts
    navigation_project_limit: 3,
  });
  await expectLabels(db, otherWeb.id, webLabels(false));
````

- [ ] **Step 3: 说明和文档**

`server/internal/bootstrap/workspace_deletion_catalog_test.go`（修改，1 处）：

````old server/internal/bootstrap/workspace_deletion_catalog_test.go
// would a cascade that deletes by the parent's key column.
````
````new server/internal/bootstrap/workspace_deletion_catalog_test.go
// would a cascade that deletes by the parent's key column. A key of a table
// to itself, as labels.parent_id, is followed as any other: such a table
// passes on its own key to parent, and its rows under another of its rows
// are found by that key.
````

`docs/v0/v0-design.md`（修改，1 处）：

````old docs/v0/v0-design.md
| 补充约束：项目内编号唯一、同一工作项同一标签唯一、**一个工作项最多属于一个迭代**、标签名在工作区内唯一 | Plane 只在代码里检查，或者本身就有缺陷 |
````
````new docs/v0/v0-design.md
| 补充约束：项目内编号唯一、同一工作项同一标签唯一、**一个工作项最多属于一个迭代**、标签名在项目内唯一（不分大小写）；没有工作区级标签（M3 设计 3.16） | Plane 只在代码里检查，或者本身就有缺陷 |
````

`docs/v0/plane-diff.md`（修改，4 处）：

````old docs/v0/plane-diff.md
| `states` | 删除 `slug`、`is_triage` | `slug` 没有读取者；分诊状态只由 `"group" = 'triage'` 识别（M3 设计 3.17） |
````
````new docs/v0/plane-diff.md
| `states` | 删除 `slug`、`is_triage` | `slug` 没有读取者；分诊状态只由 `"group" = 'triage'` 识别（M3 设计 3.17） |
| `labels` | 15 列保留 12 列（M3/P7b，`00014_project_labels.sql`） | M3 设计 4.10 |
| `labels` | `workspace_id`、`parent_id`：加上 `ON DELETE CASCADE`；`project_id`：改为 `NOT NULL`（Plane 可为空），加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`）；没有工作区级标签（M3 设计 3.16） |
| `labels` | `name`：新加 `CHECK (name <> '')`；`color`：新加 `DEFAULT ''`；`sort_order`：新加 `DEFAULT 65535`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局）；新加 `labels_not_own_parent_check CHECK (parent_id <> id)` | 模型的默认值；标签不能做自己的父标签（M3 设计 3.16） |
| `labels` | 两个部分唯一索引（`project_id` 为空时按 `name` 的 `unique_name_when_project_null_and_not_deleted`、不为空时按 `(project_id, name)` 的 `unique_project_name_when_not_deleted`）改为一个 `labels_project_id_name_key ON (project_id, lower(name)) WHERE deleted_at IS NULL`；新加不带条件的 `labels_parent_id_idx`、`labels_workspace_id_idx`、`labels_project_id_idx` | 名称在项目内唯一，不分大小写：Plane 的数据库区分大小写，序列化器不区分；没有工作区级标签（M3 设计 3.16）；删除父标签时按 `parent_id` 找子标签；物理级联要不带条件的索引（M3 设计 4） |
| `labels` | 删除 `description`、`external_source`、`external_id` | `description` 没有读取者（M3 设计 3.16）；v0 不做导入 |
````

````old docs/v0/plane-diff.md
| `issue_comments` | 删除 `description_id` 及其唯一约束 | 它是指向 `descriptions`（不保留，见一 B）的一对一外键 |
| `labels` | 工作区级标签的名称唯一范围改为 `(workspace_id, name)` | Plane 的约束没有限定在工作区内，是个缺陷 |
````
````new docs/v0/plane-diff.md
| `issue_comments` | 删除 `description_id` 及其唯一约束 | 它是指向 `descriptions`（不保留，见一 B）的一对一外键 |
````

````old docs/v0/plane-diff.md
| 删除工作区 | 清掉别人的 `last_workspace_id`；成员、显示设置等的连带软删除由 Celery 异步完成 | 不清 `last_workspace_id`（登录后的落点规则让它无害，M3 设计 3.14）；工作区、成员、邀请、显示设置，项目和它们的成员关系、成员在项目里的显示设置、状态，在同一个事务里、同一时刻软删除，删除之后 slug 立即可以重用。标签由 M3/P7b 加入这个事务 |
````
````new docs/v0/plane-diff.md
| 删除工作区 | 清掉别人的 `last_workspace_id`；成员、显示设置等的连带软删除由 Celery 异步完成 | 不清 `last_workspace_id`（登录后的落点规则让它无害，M3 设计 3.14）；工作区、成员、邀请、显示设置，项目和它们的成员关系、成员在项目里的显示设置、状态和标签，在同一个事务里、同一时刻软删除，删除之后 slug 立即可以重用 |
````

````old docs/v0/plane-diff.md
| 默认状态、分诊状态 | 在代码里维持唯一；`is_triage` 可以与 `group` 不一致 | 数据库保证每个项目各至多一个（部分唯一索引）；分诊状态只看 `group`；状态的操作同样按 `group` 认出分诊状态（列表不含它，按 id 修改、删除、设为默认答 404 `project.state_not_found`），设为默认在同一个事务里先清掉原来的默认，项目恰好一个默认状态（M3 设计 3.17） |
````
````new docs/v0/plane-diff.md
| 默认状态、分诊状态 | 在代码里维持唯一；`is_triage` 可以与 `group` 不一致 | 数据库保证每个项目各至多一个（部分唯一索引）；分诊状态只看 `group`；状态的操作同样按 `group` 认出分诊状态（列表不含它，按 id 修改、删除、设为默认答 404 `project.state_not_found`），设为默认在同一个事务里先清掉原来的默认，项目恰好一个默认状态（M3 设计 3.17） |
| 标签的名称 | 数据库区分大小写，序列化器不区分（"bug" 与 "Bug" 并发时都能写入） | 数据库不分大小写：`labels_project_id_name_key ON (project_id, lower(name))`，同名答 409 `project.label_name_taken`（M3 设计 3.16） |
| 标签的层级 | 服务端不检查：接口能建三层、把有子标签的标签放到别的标签下、以别的项目的标签做父标签 | 服务端执行页面的两层规则：父标签须是同一项目、未删除、没有父标签的标签，有子标签的标签不能有父标签，标签不能做自己的父标签，否则 422（`parent_id`，`not_allowed`）；每个标签的写都先锁项目行，并发下规则仍成立（M3 设计 3.16） |
| 工作区级标签 | 模型允许（`project_id` 可空） | 没有：`project_id` 非空（M3 设计 3.16） |
| 列出标签 | 任何工作区成员都得到 200，已离开的成员仍能列出 | 项目的有效成员，访客也能；已归档的项目照常列出（M3 设计 3.4、3.19） |
| 新建标签的顺序 | 项目已有标签时一律是它们的最大值加 10000，请求给的 `sort_order` 被忽略；没有标签时是请求给的，或者 65535 | 请求给了 `sort_order` 就用它；没给时是项目未删除的标签的最大值加 10000，没有标签时 65535 |
````

- [ ] **Step 4: 测试和 lint**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 70 个全部通过。

- [ ] **Step 5: 提交**

```bash
git add docs/v0/plane-diff.md docs/v0/v0-design.md e2e/fixtures/api.ts e2e/fixtures/assert/project.ts e2e/fixtures/assert/workspace.ts e2e/stories/project/p4-archive.spec.ts e2e/stories/project/p7-labels.spec.ts e2e/stories/workspace/w2-landing.spec.ts e2e/stories/workspace/w3-workspace-settings.spec.ts server/internal/bootstrap/workspace_deletion_catalog_test.go
```
```bash
git commit -m "test(M3/P7b): story P7 through the API, and the stories that delete projects count their labels

Story P7 creates labels, puts one under another, renames, moves and
deletes them, with what the names and the two levels keep, an archived
project's as any other's. P4 labels the archived project before its
deletion, W2 and W3 label the projects their deletions take: each label
is deleted with its project at its moment, by its account, and a label
deleted before keeps its own. The end-to-end check of a deleted project
counts labels again.

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

**变异**：本 Task 没有产品代码；它的测试在别的 Task 的变异表里（"（Task 11 起）"）。
