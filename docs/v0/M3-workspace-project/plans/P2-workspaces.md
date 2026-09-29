# M3/P2 工作区的管理和加锁约定 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 工作区的管理可用：管理员修改、删除工作区（删除在同一个事务里连带成员和显示设置），任何有效成员列出成员（邮箱按角色给出），管理员改别人的角色（改自己 409），每个成员读写自己在工作区的显示设置（第一次修改时建行）。"先锁父行，再判定"第一次落地：每个写用例在事务里先锁工作区行（带 `deleted_at IS NULL`），再判定，再写；两位管理员互相降级在真实数据库上两个顺序都测过。权限矩阵加本 Phase 的 7 行（42 格）；W8 的接口版本通过；总体设计 3.6、4.2 和差异清单中 P2 的各行写好。

**Architecture:** 新迁移 `00008`（`workspace_user_properties`）。`workspace` 模块加：领域的 `WorkspacePatch`、`Preferences`、`Membership`/`Member`/`MemberUser`、`SeesEmails`、`CheckMemberRole`；端口 `WorkspaceLocker`、`WorkspaceSharer`、`WorkspaceUpdater`、`WorkspaceDeleter`、`MemberLister`、`MemberUpdater`、`PreferencesReader`、`PreferencesWriter`、`MemberProfiles`；存储的三把父行锁（`LockWorkspaceBySlug`、`LockWorkspace` 是 `FOR NO KEY UPDATE`，`ShareWorkspaceBySlug` 是 `FOR SHARE`）；用例共用的 `lockAndDecide`、`decide`（`app/lock.go`）；六个用例；接口描述的 6 个操作。`identity.Provide` 加 `PublicProfiles`（不加锁，含停用的账户），`bootstrap/ports.go` 把它转成 `workspace` 的 `MemberProfiles`。`access` 的规则表加 5 行。测试一侧：`archtest` 的"模块只经 sqlc 执行 SQL"，`pgtest.WaitForLockWaitOn`（只算等某张表的行锁），矩阵的行按模块分文件、写格子限并行、行可以核对答案、请求可以指名准备好的行，交错 2。

**Tech Stack:** Go 1.27.1、pgx v5.11.0、sqlc v1.31.1（`CGO_ENABLED=0`）、oapi-codegen v2.8.0、goose v3.28.0、River v0.47.0、golangci-lint 2.13.2、PostgreSQL 18.6（testcontainers）；Node 24、pnpm 11.10.0、Playwright 1.63.0。

**Spec:** `docs/v0/M3-workspace-project/specs/P2-workspaces.md`（上级：`docs/v0/M3-workspace-project/M3-design.md`）

## Global Constraints

- **Go 版本和依赖**：本 plan 不执行 `go get`，不加任何 Go 模块或 npm 包。`server/go.mod` 和 `server/tools/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`（golangci-lint 2.13.2 由 go1.27.0 构建，`go` 行更高就拒绝运行）。每个 Task 提交前执行 `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod`，四行必须是 `go 1.27` 和 `toolchain go1.27.1`；`git diff --stat d247b554 -- server/go.mod server/go.sum server/tools pnpm-lock.yaml` 在本 plan 的任何时刻都没有输出。
- **每个 Task 提交前**：`make lint-go` 两段都输出 `0 issues.`，`make test` 全部通过（没有 `FAIL`）。
- **生成的文件不手写、不从本 plan 复制**：有生成物的 Task（1、4、6–12）执行 Task 中的生成命令（改了接口描述的 Task 6、8、9、10、12 执行 `make gen`，只动了 sqlc 的 Task 1、4、7、11 执行 `make gen-go`），用 `shasum -a 256` 和 `wc -l` 核对表中的 SHA-256 和行数，对不上时停下来：说明某个输入与本 plan 不一致。这些 Task **提交之后**执行 `make gen-check`（它用 `git status` 判断生成物是否已提交，提交之前必然报差异）。
- **前端检查**：改了接口描述、前端或 `e2e/` 的 Task（6、8、9、10、12、14）另执行 `make lint-web`、`make knip`、`make test-web`；Task 15 只改文档，执行 `make lint-web`（关键词守卫也查文档）；Task 14 执行 `make e2e`。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers。开发库 `nerve-dev-db-1` 可以用，但不要停止或重建它，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`）。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。
- **规则**：不写 `init()`，不用全局可变状态，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行；依赖只能向内；模块之间不互相导入，值在 `bootstrap` 中转换（M3 设计 6.5、6.6）；模块的 SQL 只经 sqlc（Task 2 起由 `TestModulesRunSQLOnlyThroughSQLC` 核对）；不留没有使用者的代码。**例外**：接口描述按模块一个文件（M0-P3 交接 5），`api/modules/*.yaml` 不受约 400 行的限制。本 plan 的其余文件都在 400 行以内（最长的是 `bootstrap/permission_matrix_test.go`，354 行）。
- **注释**：Go、TS 代码、SQL 查询和接口描述用英文；迁移文件的中文注释和中文文档照本 plan 原样。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的）。

  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从 `d247b554` 起按顺序核对过全部块：每个 `old` 恰好出现一次（在它之前的块应用之后的文件中），每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同（spec 附录 A）。可以用 `node <planapply.mjs> <本 plan> apply <仓库根> <n>` 写入第 n 个 Task 的块，也可以手工照抄。
- **过渡版本**：一些文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改（`api/modules/workspace.yaml`、`workspace/module.go`、`app/ports.go`、`app/lock.go`、`app/fakes_test.go`、`adapter/http/handler.go`、`access/domain/rules.go`、`bootstrap/permission_matrix_test.go`、`adapter/postgres/failures_test.go`、生成物）；每个过渡版本都在逐 Task 复现中运行过。
- **变异**：每个 Task 末尾的"变异"表列出：把代码改坏的方式，和必须因此失败的测试。它们在原型上逐个跑过（`$M3TMP/p2tools/mutants.py`，spec 附录 A）；实现者可以照表抽查，改坏之后必须恢复。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录。"生成"表示由命令生成并提交；一个文件由多个 Task 修改时，"Task"一列都列出。

| 文件 | 职责 | Task |
|---|---|---|
| `server/migrations/sql/00008_workspace_workspace_user_properties.sql` | 显示设置的表（M3 设计 4.5） | 1 |
| `deploy/runtime-grants.sql`、`server/sqlc.yaml`（修改） | 运行时角色的权限；sqlc 读新迁移 | 1 |
| `server/migrations/schema_test.go`（修改） | 8 个迁移 up、down、再 up；名字和种类；CHECK 的反例；部分唯一键 | 1 |
| `server/internal/modules/workspace/adapter/postgres/gen/models.go`（生成） | | 1 |
| `server/internal/archtest/rawsql_test.go`、`server/internal/archtest/rawsql_cases_test.go` | 模块只经 sqlc 执行 SQL（P1 的移交，spec 第 3 节第 2 条表的第 7 行） | 2 |
| `server/internal/platform/postgres/pgtest/lockwait.go`、`server/internal/platform/postgres/pgtest/lockwait_test.go`（修改） | `WaitForLockWaitOn`：只算等某张表的行锁 | 3 |
| `server/internal/bootstrap/workspace_test.go`（修改） | P1 的整程序测试改用它；`TestAnAdminDemotedMeanwhileCannotUpdateTheWorkspace`；`TestAMembershipEndedMeanwhileIsNotFound` | 3、6、12 |
| `server/internal/modules/workspace/adapter/postgres/locks.go`、`server/internal/modules/workspace/adapter/postgres/locks_test.go` | 工作区行的三把父行锁 | 4、11 |
| `server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql`（修改） | 锁、修改、删除工作区的查询 | 4、6、9、11 |
| `server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go`（生成） | | 4、6、9、11 |
| `server/internal/bootstrap/permission_matrix_test.go`（修改） | 矩阵的形状：按模块分文件、写格子的并行上限、行的核对、指名准备好的行 | 5、6、8、9、12 |
| `server/internal/bootstrap/permission_matrix_workspace_test.go` | `workspace` 的矩阵行 | 5、6、8、9、10、12 |
| `server/internal/bootstrap/permission_matrix_coverage_test.go`（修改） | 完整性核对随请求的新签名 | 12 |
| `api/modules/workspace.yaml`（修改）、`api/openapi.yaml`（修改） | 6 个操作和它们的结构 | 6、8、9、10、12 |
| `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`server/internal/modules/workspace/adapter/http/gen/server.gen.go`（生成） | | 6、8、9、10、12 |
| `server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go`（生成） | | 6、8、12 |
| `server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`（修改） | 规则表的 5 行和它们的格子 | 6、8、9、10、12 |
| `server/internal/modules/workspace/domain/actions.go`（修改） | 5 个操作名 | 6、8、9、10、12 |
| `server/internal/modules/workspace/domain/workspace.go`、`server/internal/modules/workspace/domain/workspace_test.go`（修改） | `WorkspacePatch`、`CheckWorkspacePatch` | 6 |
| `server/internal/modules/workspace/app/ports.go`（修改） | 本 Phase 的端口 | 6、7、9、10、12 |
| `server/internal/modules/workspace/app/lock.go` | `lockAndDecide`、`decide`：锁 → 判定 | 6、12 |
| `server/internal/modules/workspace/app/update_workspace.go`、`server/internal/modules/workspace/app/update_workspace_test.go` | `updateWorkspace` | 6 |
| `server/internal/modules/workspace/app/fakes_test.go`（修改） | 假实现 | 6、8、9、10、12 |
| `server/internal/modules/workspace/adapter/postgres/workspaces.go`（修改） | 存储的新读写 | 6、9、10、11 |
| `server/internal/modules/workspace/adapter/postgres/update_workspace_test.go` | `UpdateWorkspace` 的存储测试 | 6 |
| `server/internal/modules/workspace/adapter/postgres/failures_test.go`、`server/internal/modules/workspace/adapter/postgres/store_test.go`（修改） | 每个读写失败时答错误（P1 的读的测试移来） | 6、7、9、10、11 |
| `server/internal/modules/workspace/adapter/http/handler.go`、`server/internal/modules/workspace/adapter/http/handler_test.go`（修改） | 用例的接口；测试的假实现 | 6、8、9、10、12 |
| `server/internal/modules/workspace/adapter/http/workspaces.go`、`server/internal/modules/workspace/adapter/http/workspaces_test.go`（修改） | 修改、删除工作区的 handler | 6、9 |
| `server/internal/modules/workspace/module.go`（修改） | 接上用例 | 6、8、9、10、12 |
| `web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`（修改） | `forbidden`、`workspace.member_not_found`、`workspace.own_membership` 的文案 | 6、12 |
| `server/internal/modules/workspace/domain/preferences.go`、`server/internal/modules/workspace/domain/preferences_test.go` | 显示设置的值、默认值、校验 | 7 |
| `server/internal/modules/workspace/adapter/postgres/queries/preferences.sql`、`server/internal/modules/workspace/adapter/postgres/preferences.go`、`server/internal/modules/workspace/adapter/postgres/preferences_test.go` | 显示设置的存储 | 7、9 |
| `server/internal/modules/workspace/adapter/postgres/gen/preferences.sql.go`（生成） | | 7、9 |
| `server/internal/modules/workspace/app/preferences.go`、`server/internal/modules/workspace/app/preferences_test.go`、`server/internal/modules/workspace/adapter/http/preferences.go`、`server/internal/modules/workspace/adapter/http/preferences_test.go` | 读、改显示设置的用例和 handler | 8 |
| `server/internal/modules/workspace/app/delete_workspace.go`、`server/internal/modules/workspace/app/delete_workspace_test.go`、`server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go` | `deleteWorkspace` 和它的连带 | 9 |
| `server/internal/modules/workspace/adapter/postgres/queries/members.sql`（修改） | 成员的查询 | 9、10、11 |
| `server/internal/modules/workspace/adapter/postgres/gen/members.sql.go`（生成） | | 9、10、11 |
| `server/internal/modules/identity/adapter/postgres/queries/users.sql`、`server/internal/modules/identity/adapter/postgres/accounts.go`、`server/internal/modules/identity/app/accounts.go`、`server/internal/modules/identity/provide.go`（修改）；`server/internal/modules/identity/adapter/postgres/public_profiles_test.go` | `identity.Provide` 交出 `PublicProfiles` | 10 |
| `server/internal/modules/identity/adapter/postgres/gen/users.sql.go`（生成） | | 10 |
| `server/internal/bootstrap/ports.go`、`server/internal/bootstrap/ports_test.go`、`server/internal/bootstrap/app.go`（修改） | `PublicProfiles` 转成 `MemberProfiles` 并接上 | 10 |
| `server/internal/modules/workspace/domain/member.go`、`server/internal/modules/workspace/domain/member_test.go` | 成员关系、`MemberUser`、`SeesEmails`、`CheckMemberRole` | 10、12 |
| `server/internal/modules/workspace/app/list_members.go`、`server/internal/modules/workspace/app/list_members_test.go`、`server/internal/modules/workspace/adapter/postgres/list_members_test.go`、`server/internal/modules/workspace/adapter/http/members.go`、`server/internal/modules/workspace/adapter/http/members_test.go` | `listWorkspaceMembers` | 10、12 |
| `server/internal/modules/workspace/adapter/postgres/update_member_test.go` | `MemberByID`、`UpdateMemberRole` 的存储测试 | 11 |
| `server/internal/modules/workspace/app/update_member.go`、`server/internal/modules/workspace/app/update_member_test.go`、`server/internal/modules/workspace/domain/errors.go`（修改） | `updateWorkspaceMember`；两个新码 | 12 |
| `server/internal/bootstrap/interleaving_roles_test.go` | 交错 2：两位管理员互相降级 | 13 |
| `e2e/fixtures/api.ts`、`e2e/fixtures/assert/workspace.ts`（修改）；`e2e/stories/workspace/w8-navigation-preferences.spec.ts` | W8 的接口版本 | 14 |
| `docs/v0/v0-design.md`、`docs/v0/plane-diff.md`、`docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md`（修改） | M3 设计 3.20 中 P2 的各行；交接的处理结果 | 15 |

---

### Task 1: 迁移 `00008` 与运行时权限

**Files:**
- Create: `server/migrations/sql/00008_workspace_workspace_user_properties.sql`
- Modify: `deploy/runtime-grants.sql`、`server/migrations/schema_test.go`、`server/sqlc.yaml`
- Generate: `server/internal/modules/workspace/adapter/postgres/gen/models.go`

**Interfaces:**
- Produces（spec 2.3，M3 设计 4.5）：表 `workspace_user_properties`（10 列），约束和索引 `workspace_user_properties_{pkey, workspace_id_fkey, user_id_fkey, created_by_id_fkey, updated_by_id_fkey, navigation_project_limit_check, navigation_control_preference_check, workspace_id_user_id_key, workspace_id_idx}`；`workspace_id_user_id_key` 是 `WHERE deleted_at IS NULL` 的部分唯一索引，是显示设置的 `ON CONFLICT` 的目标（3.18）；`nerve_runtime` 对它的 `SELECT, INSERT, UPDATE, DELETE`；`server/sqlc.yaml` 的 `workspace` 条目读 `00008`。
- 使用者：Task 7 的查询和存储；Task 9 的连带。

**Tests:**（`server/migrations/schema_test.go`，真实数据库）
- `TestMigrationsGoUpDownAndUpAgain`：8 个迁移；up 之后有 `workspace_user_properties`，down 之后没有，再 up 仍是 8 个。
- `TestConstraintAndIndexNames`：查询覆盖新表，期望加 10 行，每行带种类（`f c`：`ON DELETE CASCADE` 的外键；`f n`：`SET NULL`；`iuw`：带条件的唯一索引；`i`：普通索引），所以部分唯一索引变成普通的唯一索引、外键丢了 `CASCADE` 都会失败。
- `TestChecksRejectCounterexamples`：合法的一行（上限 0、`TABBED`）插入成功；三个反例各被它的 CHECK 拒绝：上限 -1、`SIDEBAR`、小写的 `tabbed`。
- `TestUniqueKeysHoldAmongUndeletedRowsOnly`：同一账户、同一工作区第二行未删除的被拒绝；第一行软删除之后可以再插入。
- 不改而覆盖：`TestTheGrantsFileCoversEveryRelationAndFunction`（每张表都在 `deploy/runtime-grants.sql` 中）。

- [ ] **Step 1: 迁移、权限和 sqlc 条目**

`server/migrations/sql/00008_workspace_workspace_user_properties.sql`（新文件，29 行）：

````file server/migrations/sql/00008_workspace_workspace_user_properties.sql
-- workspace_user_properties：Plane 的 workspace_user_properties 表（14 列）按 M3 设计 4.5 保留 10 列。
-- 一个账户在一个工作区的项目导航偏好（3.18）。第一次修改时建出（INSERT … ON CONFLICT），没有这一行时
-- 接口答默认值，默认值与这里的 DEFAULT 相同；删除工作区时随之软删除。
-- 工作项列表的筛选和显示列（filters、display_filters、display_properties、rich_filters）由 M4、M7 按自己的格式加。

-- +goose Up
CREATE TABLE workspace_user_properties (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    user_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    -- 侧边栏显示的项目数，0 表示不限（模型的 default=10）
    navigation_project_limit integer NOT NULL DEFAULT 10 CHECK (navigation_project_limit >= 0),
    -- 模型的 choices
    navigation_control_preference varchar(25) NOT NULL DEFAULT 'ACCORDION'
        CHECK (navigation_control_preference IN ('ACCORDION', 'TABBED')),
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
-- 一个账户在一个工作区至多一行未删除的；INSERT … ON CONFLICT 以它为冲突的目标（3.18）
CREATE UNIQUE INDEX workspace_user_properties_workspace_id_user_id_key ON workspace_user_properties (workspace_id, user_id)
    WHERE deleted_at IS NULL;
-- 物理级联（M4 的 60 天清理）按它找子行（4 节开头）
CREATE INDEX workspace_user_properties_workspace_id_idx ON workspace_user_properties (workspace_id);

-- +goose Down
DROP TABLE workspace_user_properties;
````

`deploy/runtime-grants.sql`（修改，1 处）：

````old deploy/runtime-grants.sql
GRANT SELECT, INSERT, UPDATE, DELETE ON workspaces, workspace_members TO nerve_runtime;
````

````new deploy/runtime-grants.sql
GRANT SELECT, INSERT, UPDATE, DELETE ON workspaces, workspace_members, workspace_user_properties TO nerve_runtime;
````

`server/sqlc.yaml`（修改，1 处）：

````old server/sqlc.yaml
      - migrations/sql/00007_workspace_workspace_members.sql
````

````new server/sqlc.yaml
      - migrations/sql/00007_workspace_workspace_members.sql
      - migrations/sql/00008_workspace_workspace_user_properties.sql
````

- [ ] **Step 2: 生成**

Run: `make gen-go`
Expected: 成功；`workspace` 的 `models.go` 加上新表的结构：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `2cf44e1eda02c7d53805d69a5560ef6fd7a131cba9ee00048e05122dafc9c42a` | 50 | `server/internal/modules/workspace/adapter/postgres/gen/models.go` |

Run: `shasum -a 256 server/internal/modules/workspace/adapter/postgres/gen/models.go`
Expected: 与上表相同。

- [ ] **Step 3: 迁移的测试**

`server/migrations/schema_test.go`（修改，13 处）：

````old server/migrations/schema_test.go
	if err != nil || len(up) != 7 {
		t.Fatalf("Up() = %d migrations, %v; want 7", len(up), err)
````

````new server/migrations/schema_test.go
	if err != nil || len(up) != 8 {
		t.Fatalf("Up() = %d migrations, %v; want 8", len(up), err)
````

````old server/migrations/schema_test.go
			"workspace_members", "workspaces"}},
````

````new server/migrations/schema_test.go
			"workspace_members", "workspace_user_properties", "workspaces"}},
````

````old server/migrations/schema_test.go
	if again, err := m.Up(ctx); err != nil || len(again) != 7 {
		t.Errorf("Up() again = %d migrations, %v; want 7", len(again), err)
````

````new server/migrations/schema_test.go
	if again, err := m.Up(ctx); err != nil || len(again) != 8 {
		t.Errorf("Up() again = %d migrations, %v; want 8", len(again), err)
````

````old server/migrations/schema_test.go
		WHERE conrelid IN ('users'::regclass, 'profiles'::regclass, 'auth_sessions'::regclass, 'api_tokens'::regclass,
				'workspaces'::regclass, 'workspace_members'::regclass)
````

````new server/migrations/schema_test.go
		WHERE conrelid IN ('users'::regclass, 'profiles'::regclass, 'auth_sessions'::regclass, 'api_tokens'::regclass,
				'workspaces'::regclass, 'workspace_members'::regclass, 'workspace_user_properties'::regclass)
````

````old server/migrations/schema_test.go
		WHERE i.indrelid IN ('users'::regclass, 'profiles'::regclass, 'auth_sessions'::regclass, 'api_tokens'::regclass,
				'workspaces'::regclass, 'workspace_members'::regclass)
````

````new server/migrations/schema_test.go
		WHERE i.indrelid IN ('users'::regclass, 'profiles'::regclass, 'auth_sessions'::regclass, 'api_tokens'::regclass,
				'workspaces'::regclass, 'workspace_members'::regclass, 'workspace_user_properties'::regclass)
````

````old server/migrations/schema_test.go
		"workspace_members_workspace_id_member_id_key iuw",
````

````new server/migrations/schema_test.go
		"workspace_members_workspace_id_member_id_key iuw",
		"workspace_user_properties_created_by_id_fkey f n",
		"workspace_user_properties_navigation_control_preference_check c",
		"workspace_user_properties_navigation_project_limit_check c",
		"workspace_user_properties_pkey iu",
		"workspace_user_properties_pkey p",
		"workspace_user_properties_updated_by_id_fkey f n",
		"workspace_user_properties_user_id_fkey f c",
		"workspace_user_properties_workspace_id_fkey f c",
		"workspace_user_properties_workspace_id_idx i",
		"workspace_user_properties_workspace_id_user_id_key iuw",
````

````old server/migrations/schema_test.go
// (M2 design 4.2, 4.3, 4.5; M3 design 4.2, 4.3).
````

````new server/migrations/schema_test.go
// (M2 design 4.2, 4.3, 4.5; M3 design 4.2, 4.3, 4.5).
````

````old server/migrations/schema_test.go
			"'0199a2b4-0000-7000-8000-000000000005', " + user + ", 20)",
````

````new server/migrations/schema_test.go
			"'0199a2b4-0000-7000-8000-000000000005', " + user + ", 20)",
		"INSERT INTO workspace_user_properties (id, workspace_id, user_id, navigation_project_limit, navigation_control_preference) " +
			"VALUES ('0199a2b4-0000-7000-8000-000000000007', '0199a2b4-0000-7000-8000-000000000005', " + user + ", 0, 'TABBED')",
````

````old server/migrations/schema_test.go
		{"role 0", "UPDATE workspace_members SET role = 0", "workspace_members_role_check"},
````

````new server/migrations/schema_test.go
		{"role 0", "UPDATE workspace_members SET role = 0", "workspace_members_role_check"},
		{"negative project limit", "UPDATE workspace_user_properties SET navigation_project_limit = -1", "workspace_user_properties_navigation_project_limit_check"},
		{"unknown navigation control", "UPDATE workspace_user_properties SET navigation_control_preference = 'SIDEBAR'",
			"workspace_user_properties_navigation_control_preference_check"},
		{"lower-case navigation control", "UPDATE workspace_user_properties SET navigation_control_preference = 'tabbed'",
			"workspace_user_properties_navigation_control_preference_check"},
````

````old server/migrations/schema_test.go
// (M3 design 3.10, 4.2, 4.3).
````

````new server/migrations/schema_test.go
// (M3 design 3.10, 4.2, 4.3, 4.5).
````

````old server/migrations/schema_test.go
		"INSERT INTO workspaces (id, name, slug) VALUES (" + workspace + ", 'Acme', 'acme')",
		"INSERT INTO workspace_members (id, workspace_id, member_id) VALUES (gen_random_uuid(), " + workspace + ", " + user + ")",
````

````new server/migrations/schema_test.go
		"INSERT INTO workspaces (id, name, slug) VALUES (" + workspace + ", 'Acme', 'acme')",
		"INSERT INTO workspace_members (id, workspace_id, member_id) VALUES (gen_random_uuid(), " + workspace + ", " + user + ")",
		"INSERT INTO workspace_user_properties (id, workspace_id, user_id) VALUES (gen_random_uuid(), " + workspace + ", " + user + ")",
````

````old server/migrations/schema_test.go
		},
		{
````

````new server/migrations/schema_test.go
		},
		{
			"an account's preferences in a workspace",
			"INSERT INTO workspace_user_properties (id, workspace_id, user_id) VALUES (gen_random_uuid(), " + workspace + ", " + user + ")",
			"UPDATE workspace_user_properties SET deleted_at = now()",
			"workspace_user_properties_workspace_id_user_id_key",
		},
		{
````

````old server/migrations/schema_test.go
	// Either case can run first: each soft-deletes rows of its own table
	// only, and neither key reads the other table (a membership of a deleted
	// workspace still holds its key).
````

````new server/migrations/schema_test.go
	// Any case can run first: each soft-deletes rows of its own table only,
	// and no key reads another table (a membership or a preferences row of a
	// deleted workspace still holds its key).
````

- [ ] **Step 4: 测试、lint**

Run: `go -C server test -count=1 ./migrations/`
Expected: `ok`。

Run: `go -C server test -count=1 -run TestTheGrantsFileCoversEveryRelationAndFunction ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add deploy/runtime-grants.sql server/migrations/schema_test.go server/migrations/sql/00008_workspace_workspace_user_properties.sql server/sqlc.yaml server/internal/modules/workspace/adapter/postgres/gen/models.go
```
```bash
git commit -m "feat(M3/P2): the workspace_user_properties table

Migration 00008 keeps 10 of Plane's 14 columns: an account's project
navigation settings in a workspace, with the model's defaults and choices
as CHECKs and a partial unique key per account and workspace, the target
of the settings' upsert (M3 design 3.18, 4.5); the runtime role's grants.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（spec 附录 A）：

| 改坏 | 必须失败的测试 |
|---|---|
| 唯一索引去掉 `WHERE deleted_at IS NULL` | `TestConstraintAndIndexNames`、`TestUniqueKeysHoldAmongUndeletedRowsOnly`（Task 7 起另有 `TestUpsertPreferencesAfterTheRowIsDeleted`） |
| 去掉上限的 CHECK | `TestConstraintAndIndexNames`、`TestChecksRejectCounterexamples` |
| 去掉导航方式的 CHECK | `TestConstraintAndIndexNames`、`TestChecksRejectCounterexamples` |
| `workspace_id` 的外键去掉 `ON DELETE CASCADE` | `TestConstraintAndIndexNames` |

**Done when:** 8 个迁移 up、down、再 up 都通过；10 个名字和种类、3 个反例、部分唯一键由测试核对；运行时角色能读写新表。

---

### Task 2: 模块只经 sqlc 执行 SQL

**Files:**
- Create: `server/internal/archtest/rawsql_cases_test.go`、`server/internal/archtest/rawsql_test.go`

**Interfaces:**
- Produces（spec 2.4；P1 review 第 6 节，整分支评审 M7）：架构测试 `TestModulesRunSQLOnlyThroughSQLC`：`internal/modules` 下不是测试、不在 `gen` 包里的 Go 文件，不调用两个参数以上的 `Exec`、`Query`、`QueryRow`、`SendBatch`、`CopyFrom`、`Prepare`，也不含大写的 SQL 字符串（`SELECT … FROM`、`INSERT INTO`、`UPDATE … SET`、`DELETE FROM`、`TRUNCATE`、`MERGE INTO`）；解析失败的文件也报。找不到任何 `adapter/postgres/store.go` 时失败，不让"什么都没查"冒充通过。
- 使用者：以后每个 Phase 的存储（`TestSQLCSchemaScope` 只看得到 sqlc 的查询，直接经 pgx 执行的语句能读别的模块的表）。

**Tests:**（`server/internal/archtest/rawsql_cases_test.go`）
- `TestRawSQLOfTheBaseLayoutPasses`：照现有写法的模块不报：经 sqlc 查询的存储（注释里的 SQL 不算）、`r.URL.Query()` 这样没有参数的调用、小写的"select a workspace"文案、sqlc 生成的 `gen` 包、用 SQL 写夹具的测试文件。
- `TestRawSQLViolationsAreReported`：八个反例各报它的每一处，行号、列号都对：在事务上 `Query` 一条 JOIN `users` 的语句（调用和字符串两处）、常量里的 `UPDATE` 经 `Exec`（两处）、SQL 从别处来的 `QueryRow`、排进批次的 `DELETE` 和 `SendBatch`（两处）、`CopyFrom`、`Prepare`、`app` 层的一个 `INSERT` 字符串、解析失败的文件。

- [ ] **Step 1: 规则和反例**

`server/internal/archtest/rawsql_test.go`（新文件，105 行）：

````file server/internal/archtest/rawsql_test.go
package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// TestModulesRunSQLOnlyThroughSQLC holds all of a module's SQL to M2 design
// 3.14, not only the queries sqlc sees (M3 design 6.5; P1 final review M7):
// outside the gen packages that sqlc writes, a module's production code
// neither runs a statement of its own nor holds one. TestSQLCSchemaScope
// sees sqlc's queries only, so a statement run through pgx directly could
// read another module's tables, e.g. a member list that joins users instead
// of asking MemberProfiles. Tests are left out: they seed fixtures with SQL.
func TestModulesRunSQLOnlyThroughSQLC(t *testing.T) {
	registerSources(t)
	files := map[string]string{}
	root := filepath.Join(moduleRoot, "internal", "modules")
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(moduleRoot, path)
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// A walk that finds no store must not pass as "no raw SQL".
	if !slices.ContainsFunc(slices.Collect(maps.Keys(files)), func(p string) bool {
		return strings.HasSuffix(p, "/adapter/postgres/store.go")
	}) {
		t.Fatalf("read %d files under internal/modules and no adapter/postgres/store.go: the rule checks nothing", len(files))
	}
	for _, v := range rawSQLViolations(files) {
		t.Error(v)
	}
}

// statementMethods are the pgx methods that run SQL they are given: pgx's
// Conn, Tx and Batch results, pgxpool's Pool and Conn, and
// platform/postgres's Querier all have them. The call rule counts only
// calls with at least two arguments, a context and the SQL (or the batch,
// or the table): a method of one of these names that takes fewer, such as
// url.URL.Query, runs no statement.
var statementMethods = []string{"Exec", "Query", "QueryRow", "SendBatch", "CopyFrom", "Prepare"}

// sqlText matches a string literal that holds a statement: upper-case SQL,
// as every query of this repository is written. Lower-case prose, such as a
// message that tells a user to "select a workspace from the list", is not
// matched.
var sqlText = regexp.MustCompile(`\b(?:SELECT\b[\s\S]*\bFROM|INSERT\s+INTO|UPDATE\s+\S+\s+SET|DELETE\s+FROM|TRUNCATE|MERGE\s+INTO)\b`)

// rawSQLViolations checks the Go files of internal/modules, by path
// relative to server/: in a file that is neither a test nor in a gen
// package, no call of a statementMethods method with two arguments or more,
// and no string literal that sqlText matches. A file that does not parse is
// reported too, so that it cannot hide either.
func rawSQLViolations(files map[string]string) []string {
	var found []string
	for _, path := range slices.Sorted(maps.Keys(files)) {
		if strings.HasSuffix(path, "_test.go") || slices.Contains(strings.Split(path, "/"), "gen") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, files[path], parser.SkipObjectResolution)
		if err != nil {
			found = append(found, fmt.Sprintf("%s does not parse: %v", path, err))
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				if sel, ok := n.Fun.(*ast.SelectorExpr); ok && slices.Contains(statementMethods, sel.Sel.Name) && len(n.Args) >= 2 {
					found = append(found, fmt.Sprintf("%s: calls %s, which runs SQL outside sqlc's queries", fset.Position(n.Pos()), sel.Sel.Name))
				}
			case *ast.BasicLit:
				if n.Kind != token.STRING {
					return true
				}
				if s, err := strconv.Unquote(n.Value); err == nil && sqlText.MatchString(s) {
					found = append(found, fmt.Sprintf("%s: holds SQL outside sqlc's queries", fset.Position(n.Pos())))
				}
			}
			return true
		})
	}
	return found
}
````

`server/internal/archtest/rawsql_cases_test.go`（新文件，84 行）：

````file server/internal/archtest/rawsql_cases_test.go
package archtest

import (
	"slices"
	"testing"
)

// rawSQLBase is a module as the rule wants it: a store that runs sqlc's
// queries, a handler that reads a URL's query and words its messages in
// lower case, sqlc's generated code and a test that seed with SQL.
func rawSQLBase() map[string]string {
	return map[string]string{
		"internal/modules/workspace/adapter/postgres/store.go": `package postgresadapter

func (s *Store) Members(ctx context.Context, id uuid.UUID) ([]gen.ListMembersRow, error) {
	// SELECT m.id FROM workspace_members m: a comment is not SQL
	return s.queries(ctx).ListMembers(ctx, id)
}
`,
		"internal/modules/workspace/adapter/http/members.go": `package httpadapter

const hint = "select a workspace from the list, then update your settings"

func page(r *http.Request) string { return r.URL.Query().Get("page") }
`,
		"internal/modules/workspace/adapter/postgres/gen/members.sql.go": "package gen\n\nconst listMembers = `-- name: ListMembers :many\n" +
			"SELECT id FROM workspace_members WHERE workspace_id = $1`\n\n" +
			"func (q *Queries) ListMembers(ctx context.Context, id uuid.UUID) (pgx.Rows, error) { return q.db.Query(ctx, listMembers, id) }\n",
		"internal/modules/workspace/adapter/postgres/store_test.go": "package postgresadapter_test\n\n" +
			"func seed(pool *pgxpool.Pool) { pool.Exec(context.Background(), \"INSERT INTO users (id) VALUES ($1)\", 1) }\n",
	}
}

func TestRawSQLOfTheBaseLayoutPasses(t *testing.T) {
	if got := rawSQLViolations(rawSQLBase()); len(got) != 0 {
		t.Errorf("violations = %q, want none", got)
	}
}

// Each way a module could run or hold SQL of its own is reported, at its
// line.
func TestRawSQLViolationsAreReported(t *testing.T) {
	const file = "internal/modules/workspace/adapter/postgres/members.go"
	tests := []struct {
		name, source string
		want         []string
	}{
		{"a member list that joins users, run on the context's transaction", "package postgresadapter\n\n" +
			"func (s *Store) Members(ctx context.Context, id uuid.UUID) (pgx.Rows, error) {\n" +
			"\treturn postgres.DB(ctx, s.pool).Query(ctx, `SELECT m.id, u.email\n\t\tFROM workspace_members m JOIN users u ON u.id = m.member_id\n" +
			"\t\tWHERE m.workspace_id = $1`, id)\n}\n",
			[]string{file + ":4:9: calls Query, which runs SQL outside sqlc's queries", file + ":4:45: holds SQL outside sqlc's queries"}},
		{"a statement in a constant, run on the pool", "package postgresadapter\n\n" +
			"const deactivate = \"UPDATE users SET is_active = false WHERE id = $1\"\n\n" +
			"func (s *Store) Off(ctx context.Context, id uuid.UUID) error {\n\t_, err := s.pool.Exec(ctx, deactivate, id)\n\treturn err\n}\n",
			[]string{file + ":3:20: holds SQL outside sqlc's queries", file + ":6:12: calls Exec, which runs SQL outside sqlc's queries"}},
		{"a row read on a transaction, the SQL built elsewhere", "package postgresadapter\n\n" +
			"func (s *Store) Email(ctx context.Context, tx pgx.Tx, q string) error {\n\treturn tx.QueryRow(ctx, q).Scan(nil)\n}\n",
			[]string{file + ":4:9: calls QueryRow, which runs SQL outside sqlc's queries"}},
		{"a batch", "package postgresadapter\n\n" +
			"func (s *Store) Many(ctx context.Context, b *pgx.Batch) {\n\tb.Queue(\"DELETE FROM workspace_members WHERE id = $1\", 1)\n" +
			"\ts.pool.SendBatch(ctx, b).Close()\n}\n",
			[]string{file + ":4:10: holds SQL outside sqlc's queries", file + ":5:2: calls SendBatch, which runs SQL outside sqlc's queries"}},
		{"a copy", "package postgresadapter\n\n" +
			"func (s *Store) Load(ctx context.Context, c *pgx.Conn, rows pgx.CopyFromSource) {\n" +
			"\tc.CopyFrom(ctx, pgx.Identifier{\"users\"}, []string{\"id\"}, rows)\n}\n",
			[]string{file + ":4:2: calls CopyFrom, which runs SQL outside sqlc's queries"}},
		{"a prepared statement", "package postgresadapter\n\n" +
			"func (s *Store) Ready(ctx context.Context, c *pgx.Conn, q string) { c.Prepare(ctx, \"members\", q) }\n",
			[]string{file + ":3:69: calls Prepare, which runs SQL outside sqlc's queries"}},
		{"an insert in the app layer", "package app\n\n" +
			"var insert = `INSERT INTO workspace_user_properties (id) VALUES ($1)`\n",
			[]string{file + ":3:14: holds SQL outside sqlc's queries"}},
		{"a file that does not parse", "package postgresadapter\n\nfunc (\n",
			[]string{file + " does not parse: " + file + ":3:8: expected ')', found 'EOF'"}},
	}
	for _, tt := range tests {
		files := rawSQLBase()
		files[file] = tt.source
		if got := rawSQLViolations(files); !slices.Equal(got, tt.want) {
			t.Errorf("%s: violations =\n%q\nwant\n%q", tt.name, got, tt.want)
		}
	}
}
````

- [ ] **Step 2: 测试、lint**

Run: `go -C server test -count=1 -run 'TestRawSQL|TestModulesRunSQLOnlyThroughSQLC' ./internal/archtest/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/archtest/rawsql_cases_test.go server/internal/archtest/rawsql_test.go
```
```bash
git commit -m "test(M3/P2): modules run SQL only through sqlc

Outside the gen packages, a module's production code neither calls a pgx
method that runs a statement nor holds one: TestSQLCSchemaScope sees sqlc's
queries only, and a statement run directly could read another module's
tables (P1 final review, M7).

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**：

| 改坏 | 必须失败的测试 |
|---|---|
| 规则不查 `Exec` | `TestRawSQLViolationsAreReported` |
| 规则不认 `SELECT … FROM` | `TestRawSQLViolationsAreReported` |
| 规则跳过所有 `adapter/` | `TestRawSQLViolationsAreReported` |

**Done when:** 基准写法通过，八个反例各报它的每一处，真实的仓库通过。

---

### Task 3: `pgtest.WaitForLockWaitOn`

**Files:**
- Modify: `server/internal/bootstrap/workspace_test.go`、`server/internal/platform/postgres/pgtest/lockwait.go`、`server/internal/platform/postgres/pgtest/lockwait_test.go`

**Interfaces:**
- Produces（spec 2.5；P1 review 第 6 节）：`pgtest.WaitForLockWaitOn(t, pool, table string, limit time.Duration)`：只在 `pool` 的数据库里有连接在等 `table` 某一行的行锁时返回（`pg_locks` 中 `locktype = 'tuple'`、`relation = to_regclass(table)` 的一行，与 `pg_stat_activity` 的 `wait_event_type = 'Lock'` 相连），到期失败，失败信息是 `no statement waited for a row lock of <table> within <limit>`。`WaitForLockWait` 不变（两者共用 `waitFor`）。
- 使用者：P1 的 `TestAnAccountDeactivatedMeanwhileCannotCreateAWorkspace` 改等 `users`（组合出的 app 在同一个数据库上跑 River 的任务，它的锁等待会让不分表的探测提前返回）；Task 4、6、12、13 的锁测试等 `workspaces`。

**Tests:**
- `lockwait_test.go`：`TestWaitForLockWaitOnSeesOnlyWaitsForItsTablesRows`：等 `a` 的一行时对 `a` 返回；对 `b`（等待的事务读过 `b`）、对只等咨询锁的数据库都在 300 毫秒到期失败，信息如上。
- `bootstrap/workspace_test.go`：`TestAnAccountDeactivatedMeanwhileCannotCreateAWorkspace` 改用 `WaitForLockWaitOn(…, "users", …)`，断言不变。

- [ ] **Step 1: 探测和它的测试**

`server/internal/platform/postgres/pgtest/lockwait.go`（修改，3 处）：

````old server/internal/platform/postgres/pgtest/lockwait.go
	t.Helper()
````

````new server/internal/platform/postgres/pgtest/lockwait.go
	t.Helper()
	waitFor(t, pool, limit, "a lock",
		"SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'")
}

// WaitForLockWaitOn returns once a backend connected to pool's database
// waits for a row lock of table, and fails the test when none has within
// limit. While a backend waits for a row, it holds or awaits that row's
// tuple lock on the table, so a wait for another table's rows, or for a
// lock of another kind, such as an advisory lock, does not count. Use it
// where more than one statement could wait, or where the table waited on is
// what the test asserts, as with the parent-row locks of M3 design 3.6.
func WaitForLockWaitOn(t testing.TB, pool *pgxpool.Pool, table string, limit time.Duration) {
	t.Helper()
	waitFor(t, pool, limit, "a row lock of "+table, `
		SELECT count(DISTINCT a.pid) FROM pg_stat_activity a JOIN pg_locks l ON l.pid = a.pid
		WHERE a.datname = current_database() AND a.wait_event_type = 'Lock'
			AND l.locktype = 'tuple' AND l.relation = to_regclass($1)`, table)
}

// waitFor polls query, a count of the waiting backends, until it is
// positive; what names the wait in the failure.
func waitFor(t testing.TB, pool *pgxpool.Pool, limit time.Duration, what, query string, args ...any) {
	t.Helper()
````

````old server/internal/platform/postgres/pgtest/lockwait.go
		if err := pool.QueryRow(context.Background(),
			"SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'").Scan(&waiting); err != nil {
````

````new server/internal/platform/postgres/pgtest/lockwait.go
		if err := pool.QueryRow(context.Background(), query, args...).Scan(&waiting); err != nil {
````

````old server/internal/platform/postgres/pgtest/lockwait.go
	t.Fatalf("no statement waited for a lock within %v", limit)
````

````new server/internal/platform/postgres/pgtest/lockwait.go
	t.Fatalf("no statement waited for %s within %v", what, limit)
````

`server/internal/platform/postgres/pgtest/lockwait_test.go`（修改，1 处）：

````old server/internal/platform/postgres/pgtest/lockwait_test.go
		t.Errorf("WaitForLockWait on the other database failed with %q, want it to fail at its deadline", failed)
	}
````

````new server/internal/platform/postgres/pgtest/lockwait_test.go
		t.Errorf("WaitForLockWait on the other database failed with %q, want it to fail at its deadline", failed)
	}
}

// WaitForLockWaitOn counts a wait for a row of its table only: not a wait
// for another table's row, though the waiting transaction has read that
// table; not a wait for an advisory lock; not a wait in another database.
func TestWaitForLockWaitOnSeesOnlyWaitsForItsTablesRows(t *testing.T) {
	t.Parallel()
	rows, advisory := pgtest.NewDatabase(t), pgtest.NewDatabase(t)
	for _, url := range []string{rows, advisory} {
		for _, stmt := range []string{"CREATE TABLE a (id int PRIMARY KEY)", "CREATE TABLE b (id int PRIMARY KEY)", "INSERT INTO a VALUES (1)"} {
			if _, err := connect(t, url).Exec(context.Background(), stmt); err != nil {
				t.Fatal(err)
			}
		}
	}
	holdRowAndWait(t, rows)
	holdAndWait(t, advisory)
	rowsPool, advisoryPool := newPool(t, rows), newPool(t, advisory)

	pgtest.WaitForLockWaitOn(t, rowsPool, "a", 10*time.Second)
	for _, tt := range []struct {
		name  string
		pool  *pgxpool.Pool
		table string
	}{
		{"another table, which the waiter read", rowsPool, "b"},
		{"an advisory lock", advisoryPool, "a"},
	} {
		failed := fatalOf(func(tb testing.TB) { pgtest.WaitForLockWaitOn(tb, tt.pool, tt.table, 300*time.Millisecond) })
		if want := "no statement waited for a row lock of " + tt.table + " within 300ms"; failed != want {
			t.Errorf("%s: WaitForLockWaitOn failed with %q, want %q", tt.name, failed, want)
		}
	}
}

// holdRowAndWait makes a second connection to url wait for the row of a
// that a first one holds FOR NO KEY UPDATE, after reading b in the same
// transaction, until the test ends.
func holdRowAndWait(t *testing.T, url string) {
	t.Helper()
	holder, waiter := connect(t, url), connect(t, url)
	held, err := holder.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := held.Exec(context.Background(), "SELECT id FROM a WHERE id = 1 FOR NO KEY UPDATE"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	done := make(chan error, 1)
	go func() {
		tx, err := waiter.Begin(ctx)
		if err == nil {
			_, err = tx.Exec(ctx, "SELECT count(*) FROM b")
		}
		if err == nil {
			_, err = tx.Exec(ctx, "SELECT id FROM a WHERE id = 1 FOR NO KEY UPDATE")
		}
		if err == nil {
			err = tx.Rollback(ctx)
		}
		done <- err
	}()
	// Runs before the connections close: the waiter gets the row, or its
	// context ends the wait.
	t.Cleanup(func() {
		defer cancel()
		if err := held.Rollback(context.Background()); err != nil {
			t.Error(err)
		}
		if err := <-done; err != nil {
			t.Errorf("the waiting connection: %v", err)
		}
	})
````

- [ ] **Step 2: P1 的整程序测试改用它**

`server/internal/bootstrap/workspace_test.go`（修改，1 处）：

````old server/internal/bootstrap/workspace_test.go
	pgtest.WaitForLockWait(t, pool, 5*time.Second) // the creation waits on alice's row
````

````new server/internal/bootstrap/workspace_test.go
	// The creation waits on alice's row. The app runs River's jobs on the
	// same database, so only a wait for a row of users counts.
	pgtest.WaitForLockWaitOn(t, pool, "users", 5*time.Second)
````

- [ ] **Step 3: 测试、lint**

Run: `go -C server test -count=1 ./internal/platform/postgres/pgtest/`
Expected: `ok`。

Run: `go -C server test -count=3 -run TestAnAccountDeactivatedMeanwhileCannotCreateAWorkspace ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/bootstrap/workspace_test.go server/internal/platform/postgres/pgtest/lockwait.go server/internal/platform/postgres/pgtest/lockwait_test.go
```
```bash
git commit -m "test(M3/P2): WaitForLockWaitOn counts a wait for one table's rows

A wait for another table's row, or for an advisory lock, does not satisfy
it: the composed app runs River's jobs on the same database. P1's
whole-program test of the account lock waits on users now.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**：

| 改坏 | 必须失败的测试 |
|---|---|
| 探测不看表（`relation = … OR true`） | `TestWaitForLockWaitOnSeesOnlyWaitsForItsTablesRows` |
| 探测不看锁的种类（任何锁等待都算） | `TestWaitForLockWaitOnSeesOnlyWaitsForItsTablesRows` |

**Done when:** 探测的三种情形由测试核对；P1 的测试等 `users`。

---

### Task 4: 工作区行的父行锁

**Files:**
- Create: `server/internal/modules/workspace/adapter/postgres/locks.go`、`server/internal/modules/workspace/adapter/postgres/locks_test.go`
- Modify: `server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql`
- Generate: `server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go`

**Interfaces:**
- Produces（spec 2.6，M3 设计 3.6 约定二）：
  - 查询 `LockWorkspaceBySlug`（`FOR NO KEY UPDATE`）、`ShareWorkspaceBySlug`（`FOR SHARE`），都带 `deleted_at IS NULL`；
  - `(*Store).LockWorkspaceBySlug(ctx, slug) (uuid.UUID, error)`、`(*Store).ShareWorkspaceBySlug(ctx, slug) (uuid.UUID, error)`：锁住未删除的工作区行直到 ctx 带的事务结束，答它的 id；没有这一行（包括等锁期间被删除）时 `app.ErrNotFound`；别的失败原样包装返回，不是 `ErrNotFound`。
- 使用者：Task 6、8、9 的写用例；Task 11 加按 id 的第三把锁。

**Tests:**（`locks_test.go`，真实数据库）
- `TestTheWorkspaceLocksConflictAsConvention2Says`：持有一把锁时另一把在 `lock_timeout` 下得到 `55P03` 或者立即答 id：`FOR NO KEY UPDATE` 等任何一把，`FOR SHARE` 等 `FOR NO KEY UPDATE`、不等另一个 `FOR SHARE`，都不等另一个工作区的锁。
- `TestTheWorkspaceLocksFindOnlyAnUndeletedWorkspace`：找到的答 id；已删除的、不存在的、大小写不同的 slug：`app.ErrNotFound`。
- `TestTheWorkspaceLocksSkipAWorkspaceDeletedWhileTheyWait`：一个事务删除工作区并持着行锁，锁在等（`WaitForLockWaitOn(…, "workspaces", …)` 确认），删除提交之后锁读到 0 行：`app.ErrNotFound`。
- `TestAFailedWorkspaceLockIsAnErrorNotAnAnswer`：已取消的 ctx：`context.Canceled`，不是 `app.ErrNotFound`。

- [ ] **Step 1: 查询**

`server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql
SELECT EXISTS (SELECT 1 FROM workspaces WHERE slug = sqlc.arg(slug) AND deleted_at IS NULL);

````

````new server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql
SELECT EXISTS (SELECT 1 FROM workspaces WHERE slug = sqlc.arg(slug) AND deleted_at IS NULL);

-- name: LockWorkspaceBySlug :one
-- The parent lock of a write that changes the workspace row itself or a membership (M3 design 3.6 convention 2):
-- FOR NO KEY UPDATE waits for another FOR NO KEY UPDATE and for FOR SHARE. After a wait, Postgres evaluates
-- deleted_at IS NULL again on the row's newest version, so a workspace deleted meanwhile reads no row.
SELECT id
FROM workspaces
WHERE slug = sqlc.arg(slug) AND deleted_at IS NULL
FOR NO KEY UPDATE;

-- name: ShareWorkspaceBySlug :one
-- The parent lock of a write that adds or changes a row under the workspace (M3 design 3.6 convention 2): FOR SHARE
-- does not wait for another FOR SHARE, and it holds off the workspace's deletion, which the FOR KEY SHARE of a
-- foreign key check does not.
SELECT id
FROM workspaces
WHERE slug = sqlc.arg(slug) AND deleted_at IS NULL
FOR SHARE;

````

- [ ] **Step 2: 生成**

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `b04897e5cf1d9415048f8447b97df5c4ae760fc2d4dd3ffbc66fc1e227428e8a` | 198 | `server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go` |

Run: `shasum -a 256 server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go`
Expected: 与上表相同。

- [ ] **Step 3: 存储和测试**

`server/internal/modules/workspace/adapter/postgres/locks.go`（新文件，42 行）：

````file server/internal/modules/workspace/adapter/postgres/locks.go
package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
)

// The parent locks of the writes on a workspace (M3 design 3.6 convention
// 2). Each locks the undeleted workspace row until the transaction ctx
// carries ends and returns the workspace's id; app.ErrNotFound when there is
// none, also when it was deleted while the lock waited. Outside a
// transaction the lock would end with its statement: call them inside one.

// LockWorkspaceBySlug locks the workspace with slug FOR NO KEY UPDATE: for
// a write of the workspace row itself or of a membership.
func (s *Store) LockWorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	id, err := s.queries(ctx).LockWorkspaceBySlug(ctx, slug)
	return lockedWorkspace(id, err)
}

// ShareWorkspaceBySlug locks the workspace with slug FOR SHARE: for a write
// that adds or changes a row under the workspace.
func (s *Store) ShareWorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	id, err := s.queries(ctx).ShareWorkspaceBySlug(ctx, slug)
	return lockedWorkspace(id, err)
}

func lockedWorkspace(id uuid.UUID, err error) (uuid.UUID, error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return uuid.UUID{}, app.ErrNotFound
	case err != nil:
		return uuid.UUID{}, fmt.Errorf("lock the workspace row: %w", err)
	}
	return id, nil
}
````

`server/internal/modules/workspace/adapter/postgres/locks_test.go`（新文件，235 行）：

````file server/internal/modules/workspace/adapter/postgres/locks_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// lock is one of the store's parent locks of a workspace named by its slug
// (M3 design 3.6 convention 2).
type lock struct {
	name string
	take func(ctx context.Context, s *postgresadapter.Store, slug string) (uuid.UUID, error)
}

var (
	noKeyUpdate = lock{"LockWorkspaceBySlug", func(ctx context.Context, s *postgresadapter.Store, slug string) (uuid.UUID, error) {
		return s.LockWorkspaceBySlug(ctx, slug)
	}}
	forShare = lock{"ShareWorkspaceBySlug", func(ctx context.Context, s *postgresadapter.Store, slug string) (uuid.UUID, error) {
		return s.ShareWorkspaceBySlug(ctx, slug)
	}}
	locks = []lock{noKeyUpdate, forShare}
)

// hold runs fn in a transaction of its own and keeps it open until end is
// called, which commits it and returns its error. fn not done within 10s
// fails the test. The test's cleanup calls end too, so a failure still ends
// the transaction: the pool's Close waits for its connection.
func hold(t *testing.T, tx *postgres.TxManager, fn func(ctx context.Context) error) (end func() error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	done, release := make(chan struct{}), make(chan struct{})
	held := make(chan error, 1)
	go func() {
		held <- tx.WithinTx(ctx, func(ctx context.Context) error {
			if err := fn(ctx); err != nil {
				return err
			}
			close(done)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	var once sync.Once
	var ended error
	end = func() error {
		once.Do(func() {
			close(release)
			select {
			case ended = <-held:
			case <-ctx.Done():
				ended = errors.New("the transaction holding the row did not end within 10s")
			}
			cancel()
		})
		return ended
	}
	t.Cleanup(func() { _ = end() })
	select {
	case <-done:
	case err := <-held:
		once.Do(cancel) // the transaction has ended: nothing to wait for
		t.Fatalf("taking the lock: %v", err)
	case <-ctx.Done():
		t.Fatal("the lock was not taken within 10s")
	}
	return end
}

// withLockTimeout runs fn in a transaction whose lock waits end after
// 300ms with lock_not_available (55P03).
func withLockTimeout(tx *postgres.TxManager, pool *pgxpool.Pool, fn func(ctx context.Context) error) error {
	return tx.WithinTx(context.Background(), func(ctx context.Context) error {
		if _, err := postgres.DB(ctx, pool).Exec(ctx, "SET LOCAL lock_timeout = '300ms'"); err != nil {
			return err
		}
		return fn(ctx)
	})
}

// The two locks conflict as convention 2 wants: FOR NO KEY UPDATE waits for
// either, FOR SHARE waits for FOR NO KEY UPDATE and not for another FOR
// SHARE, and neither waits for a lock of another workspace. A lock that
// waits ends with lock_not_available under a lock_timeout; one that does not
// answers its workspace's id.
func TestTheWorkspaceLocksConflictAsConvention2Says(t *testing.T) {
	tests := []struct {
		held, then lock
		slug       string // then's
		waits      bool
	}{
		{noKeyUpdate, noKeyUpdate, "acme", true},
		{noKeyUpdate, forShare, "acme", true},
		{forShare, noKeyUpdate, "acme", true},
		{forShare, forShare, "acme", false},
		{noKeyUpdate, noKeyUpdate, "beta", false},
		{noKeyUpdate, forShare, "beta", false},
	}
	for _, tt := range tests {
		t.Run(tt.held.name+" held, "+tt.then.name+" of "+tt.slug, func(t *testing.T) {
			s, pool := newStore(t)
			alice := newAccount(t, pool, "alice@corp.com")
			ids := map[string]uuid.UUID{"acme": newWorkspace(t, s, "Acme", "acme", alice).ID, "beta": newWorkspace(t, s, "Beta", "beta", alice).ID}
			tx := postgres.NewTxManager(pool, 2*time.Second)
			hold(t, tx, func(ctx context.Context) error {
				_, err := tt.held.take(ctx, s, "acme")
				return err
			})

			var got uuid.UUID
			err := withLockTimeout(tx, pool, func(ctx context.Context) error {
				var err error
				got, err = tt.then.take(ctx, s, tt.slug)
				return err
			})

			var pgErr *pgconn.PgError
			switch {
			case tt.waits && (!errors.As(err, &pgErr) || pgErr.Code != "55P03"):
				t.Errorf("%s() = %s, %v; want lock_not_available after waiting", tt.then.name, got, err)
			case !tt.waits && (err != nil || got != ids[tt.slug]):
				t.Errorf("%s() = %s, %v; want %s's id %s without waiting", tt.then.name, got, err, tt.slug, ids[tt.slug])
			}
		})
	}
}

// A lock finds the undeleted workspace with the slug and answers its id;
// app.ErrNotFound for a deleted workspace, a slug no workspace has, or a
// slug of another case.
func TestTheWorkspaceLocksFindOnlyAnUndeletedWorkspace(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	gone := newWorkspace(t, s, "Gone", "gone", alice)
	exec(t, pool, "UPDATE workspaces SET deleted_at = $2 WHERE id = $1", gone.ID, now)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	for _, l := range locks {
		for slug, want := range map[string]uuid.UUID{"acme": acme.ID, "beta": beta.ID, "gone": {}, "nothing": {}, "ACME": {}} {
			var got uuid.UUID
			err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
				var err error
				got, err = l.take(ctx, s, slug)
				return err
			})
			if want == (uuid.UUID{}) && (!errors.Is(err, app.ErrNotFound) || got != want) {
				t.Errorf("%s(%q) = %s, %v; want app.ErrNotFound", l.name, slug, got, err)
			}
			if want != (uuid.UUID{}) && (err != nil || got != want) {
				t.Errorf("%s(%q) = %s, %v; want %s", l.name, slug, got, err, want)
			}
		}
	}
}

// A lock that waits for the transaction deleting the workspace reads no row
// once that one commits (convention 2): Postgres evaluates deleted_at IS
// NULL again on the committed version. WaitForLockWaitOn sees the lock
// waiting on the workspace row before the deletion commits.
func TestTheWorkspaceLocksSkipAWorkspaceDeletedWhileTheyWait(t *testing.T) {
	for _, l := range locks {
		t.Run(l.name, func(t *testing.T) {
			s, pool := newStore(t)
			alice := newAccount(t, pool, "alice@corp.com")
			acme := newWorkspace(t, s, "Acme", "acme", alice)
			tx := postgres.NewTxManager(pool, 2*time.Second)
			commit := hold(t, tx, func(ctx context.Context) error {
				if _, err := s.LockWorkspaceBySlug(ctx, "acme"); err != nil {
					return err
				}
				_, err := postgres.DB(ctx, pool).Exec(ctx, "UPDATE workspaces SET deleted_at = $2 WHERE id = $1", acme.ID, now)
				return err
			})
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			type answer struct {
				id  uuid.UUID
				err error
			}
			answered := make(chan answer, 1)
			go func() {
				var a answer
				a.err = tx.WithinTx(ctx, func(ctx context.Context) error {
					var err error
					a.id, err = l.take(ctx, s, "acme")
					return err
				})
				answered <- a
			}()
			pgtest.WaitForLockWaitOn(t, pool, "workspaces", 5*time.Second)
			if err := commit(); err != nil {
				t.Fatalf("the deletion: %v", err)
			}
			select {
			case a := <-answered:
				if !errors.Is(a.err, app.ErrNotFound) || a.id != (uuid.UUID{}) {
					t.Errorf("%s() after the deletion = %s, %v; want app.ErrNotFound", l.name, a.id, a.err)
				}
			case <-ctx.Done():
				t.Fatal("the lock did not end within 10s")
			}
		})
	}
}

// A lock that fails answers its error, never app.ErrNotFound, which the use
// case would turn into workspace.not_found: a cancelled context.
func TestAFailedWorkspaceLockIsAnErrorNotAnAnswer(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	newWorkspace(t, s, "Acme", "acme", alice)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, l := range locks {
		if id, err := l.take(cancelled, s, "acme"); !errors.Is(err, context.Canceled) || errors.Is(err, app.ErrNotFound) || id != (uuid.UUID{}) {
			t.Errorf("%s() = %s, %v; want context.Canceled, not app.ErrNotFound", l.name, id, err)
		}
	}
}
````

- [ ] **Step 4: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/adapter/postgres/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/modules/workspace/adapter/postgres/locks.go server/internal/modules/workspace/adapter/postgres/locks_test.go server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go
```
```bash
git commit -m "feat(M3/P2): the workspace row's parent locks

LockWorkspaceBySlug takes the undeleted workspace FOR NO KEY UPDATE and
ShareWorkspaceBySlug FOR SHARE, until the transaction ends; a workspace
deleted while the lock waits is not found (M3 design 3.6 convention 2).

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**（改生成的查询常量，等于改 `.sql` 再生成）：

| 改坏 | 必须失败的测试 |
|---|---|
| `LockWorkspaceBySlug` 去掉 `deleted_at IS NULL` | `TestTheWorkspaceLocksFindOnlyAnUndeletedWorkspace`、`TestTheWorkspaceLocksSkipAWorkspaceDeletedWhileTheyWait` |
| `ShareWorkspaceBySlug` 去掉 `deleted_at IS NULL` | 同上 |
| `LockWorkspaceBySlug` 换成 `FOR SHARE` | `TestTheWorkspaceLocksConflictAsConvention2Says` |
| `ShareWorkspaceBySlug` 换成 `FOR NO KEY UPDATE` | `TestTheWorkspaceLocksConflictAsConvention2Says` |

**Done when:** 四个锁测试在真实数据库上通过，每种冲突都在 `lock_timeout` 之下确定地得到；生成物的 SHA-256 与表相同。

---

### Task 5: 矩阵的形状

**Files:**
- Create: `server/internal/bootstrap/permission_matrix_workspace_test.go`
- Modify: `server/internal/bootstrap/permission_matrix_test.go`

**Interfaces:**
- Produces（spec 2.7，M3 设计 9.2；P1 review 第 6 节，整分支评审 M8）：
  - 行按模块分文件：`permission_matrix_workspace_test.go` 的 `workspaceMatrixRows()`；`matrixRows()` 把各模块的行接起来；P1 的 5 行和 `inWorkspace` 等辅助函数移过去；
  - `matrixApps = 8`：写格子（和带 `config` 的格子）各开一个 app，同时最多 8 个（每个 app 的连接池最多 4 个连接，容器的 `max_connections` 是 100）；
  - `matrixRow.check func(t, c caller, answer string)`：格子的状态码和码对了、而且不是 problem 时核对答案的内容；`TestPermissionMatrix` 数一共核对了多少个答案，与应核对的格子数不等时失败（所以跳过核对的矩阵不能通过）；
  - `decodeAnswer(t, answer, v)`；P1 的两行加上核对：`listWorkspaces`（每列列出的 slug）、`getWorkspace`（`acme` 和调用者自己的角色）。
- 使用者：Task 6–12 的矩阵行。

**Tests:**
- `TestPermissionMatrix`：P1 的 30 格不变，两行多了答案的核对；
- `TestThePermissionMatrixCoversEveryOperation`、`TestMatrixViolationsCatchesEachGap` 不改而通过。

- [ ] **Step 1: 形状和移过去的行**

`server/internal/bootstrap/permission_matrix_test.go`（修改，12 处）：

````old server/internal/bootstrap/permission_matrix_test.go
	"context"
````

````new server/internal/bootstrap/permission_matrix_test.go
	"context"
	"encoding/json"
````

````old server/internal/bootstrap/permission_matrix_test.go
	"net/http"
	"strings"
````

````new server/internal/bootstrap/permission_matrix_test.go
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
````

````old server/internal/bootstrap/permission_matrix_test.go
// by cell. The data is
````

````new server/internal/bootstrap/permission_matrix_test.go
// by cell, and where a row says so, what the answer holds. The data is
````

````old server/internal/bootstrap/permission_matrix_test.go
// (pgtest.NewDatabaseFrom), so no cell sees another's writes. A phase that
// adds an operation adds its row, and what the row needs prepared.
````

````new server/internal/bootstrap/permission_matrix_test.go
// (pgtest.NewDatabaseFrom), so no cell sees another's writes. Each module's
// rows are in a file of their own (permission_matrix_<module>_test.go): a
// phase that adds an operation adds its row there, and what the row needs
// prepared here.
````

````old server/internal/bootstrap/permission_matrix_test.go
	cellOK                = cell{status: http.StatusOK}
	cellCreated           = cell{status: http.StatusCreated}
	cellWorkspaceNotFound = cell{http.StatusNotFound, "workspace.not_found"}
	cellCreationDisabled  = cell{http.StatusForbidden, "workspace.creation_disabled"}
````

````new server/internal/bootstrap/permission_matrix_test.go
	cellOK      = cell{status: http.StatusOK}
	cellCreated = cell{status: http.StatusCreated}
````

````old server/internal/bootstrap/permission_matrix_test.go
	cells   map[caller]cell
````

````new server/internal/bootstrap/permission_matrix_test.go
	cells   map[caller]cell
	// check, when set, runs on each answer that is not a problem and is its
	// cell's: what the answer holds for that caller.
	check func(t *testing.T, c caller, answer string)
````

````old server/internal/bootstrap/permission_matrix_test.go
// matrixRows are the rows, a phase's operations added by that phase.
func matrixRows() []matrixRow {
	return []matrixRow{
		// The account level: any valid credential (6.4).
		{op: "listWorkspaces", request: sameRequest(http.MethodGet, "/api/v0/workspaces", ""), cells: every(cellOK)},
		{op: "checkWorkspaceSlug", request: sameRequest(http.MethodGet, "/api/v0/workspace-slugs/acme", ""), cells: every(cellOK)},
		{op: "createWorkspace", write: true, request: sameRequest(http.MethodPost, "/api/v0/workspaces", `{"name":"New","slug":"new"}`),
			cells: every(cellCreated)},
		{op: "createWorkspace", variant: "creation switched off", write: true,
			config:  func(cfg *config.Config) { cfg.Workspace.CreationEnabled = false },
			request: sameRequest(http.MethodPost, "/api/v0/workspaces", `{"name":"New","slug":"new"}`), cells: every(cellCreationDisabled)},
		// The workspace level.
		{op: "getWorkspace",
			request: func(c caller) (string, string, string) {
				return http.MethodGet, "/api/v0/workspaces/" + workspaceOf(c), ""
			},
			cells: map[caller]cell{callerAdmin: cellOK, callerMember: cellOK, callerGuest: cellOK,
				callerNever: cellWorkspaceNotFound, callerRemoved: cellWorkspaceNotFound, callerDeleted: cellWorkspaceNotFound}},
	}
}

````

````new server/internal/bootstrap/permission_matrix_test.go
// decodeAnswer decodes a cell's answer into v for a row's check.
func decodeAnswer(t *testing.T, answer string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(answer), v); err != nil {
		t.Fatalf("the answer %s: %v", answer, err)
	}
}

// matrixRows are the rows, each module's from its file.
func matrixRows() []matrixRow {
	return slices.Concat(workspaceMatrixRows())
}

// matrixApps is how many writing cells may run an app at once. Each app's
// pool opens up to testConfig's MaxConns (4) connections, besides the reads
// app's and pgtest's admin pool (4 each), against the container's
// max_connections of 100.
const matrixApps = 8

````

````old server/internal/bootstrap/permission_matrix_test.go
// Each cell of the matrix, the writing ones in parallel on their copies.
````

````new server/internal/bootstrap/permission_matrix_test.go
// Each cell of the matrix, the writing ones in parallel on their copies, at
// most matrixApps of them with an app at once. Every answer a row's check is
// for is checked, and counted: a harness that skipped the checks would fail.
````

````old server/internal/bootstrap/permission_matrix_test.go
	reads := startApp(t, d.config(t, pgtest.NewDatabaseFrom(t, d.url), nil), migrations.FS())
````

````new server/internal/bootstrap/permission_matrix_test.go
	reads := startApp(t, d.config(t, pgtest.NewDatabaseFrom(t, d.url), nil), migrations.FS())
	apps := make(chan struct{}, matrixApps)
	var checked atomic.Int64
	toCheck := 0
	t.Cleanup(func() {
		if !t.Failed() && checked.Load() != int64(toCheck) {
			t.Errorf("%d answers checked, want %d", checked.Load(), toCheck)
		}
	})
````

````old server/internal/bootstrap/permission_matrix_test.go
				continue // TestThePermissionMatrixCoversEveryOperation reports it
			}
			t.Run(r.name()+"/"+string(c), func(t *testing.T) {
				// Connections: at most -parallel cells (GOMAXPROCS by
				// default) run at once, and each that starts an app of its
				// own (a writing cell) opens a pool of up to testConfig's
				// MaxConns (4), besides the reads app's pool and pgtest's
				// admin pool (4 each), against the container's
				// max_connections of 100. P2 bounds the cells that start an
				// app with a semaphore.
````

````new server/internal/bootstrap/permission_matrix_test.go
				continue // TestThePermissionMatrixCoversEveryOperation reports it
			}
			if r.check != nil && want.code == "" {
				toCheck++
			}
			t.Run(r.name()+"/"+string(c), func(t *testing.T) {
````

````old server/internal/bootstrap/permission_matrix_test.go
				if r.write || r.config != nil {
````

````new server/internal/bootstrap/permission_matrix_test.go
				if r.write || r.config != nil {
					apps <- struct{}{}
					// Registered before the app's: cleanups run last first,
					// so the slot is freed once the app is closed.
					t.Cleanup(func() { <-apps })
````

````old server/internal/bootstrap/permission_matrix_test.go
					t.Errorf("%s %s = %d %s, want %s", method, path, status, strings.TrimSpace(answer), want)
````

````new server/internal/bootstrap/permission_matrix_test.go
					t.Errorf("%s %s = %d %s, want %s", method, path, status, strings.TrimSpace(answer), want)
					return
				}
				if r.check != nil && got.code == "" {
					r.check(t, c, answer)
					checked.Add(1)
````

`server/internal/bootstrap/permission_matrix_workspace_test.go`（新文件，83 行）：

````file server/internal/bootstrap/permission_matrix_workspace_test.go
package bootstrap

import (
	"net/http"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// The workspace module's rows of the permission matrix (M3 design 9.2).

var (
	cellWorkspaceNotFound = cell{http.StatusNotFound, "workspace.not_found"}
	cellCreationDisabled  = cell{http.StatusForbidden, "workspace.creation_disabled"}
)

// inWorkspace are the cells of a workspace-level row: the answer of the
// workspace's admin, member and guest, and workspace.not_found for the
// callers the workspace is not visible to.
func inWorkspace(admin, member, guest cell) map[caller]cell {
	return map[caller]cell{callerAdmin: admin, callerMember: member, callerGuest: guest,
		callerNever: cellWorkspaceNotFound, callerRemoved: cellWorkspaceNotFound, callerDeleted: cellWorkspaceNotFound}
}

// toWorkspace is the request of a row whose callers each send method to the
// path under the workspace their column targets.
func toWorkspace(method, path, body string) func(caller) (string, string, string) {
	return func(c caller) (string, string, string) {
		return method, "/api/v0/workspaces/" + workspaceOf(c) + path, body
	}
}

func workspaceMatrixRows() []matrixRow {
	return []matrixRow{
		// The account level: any valid credential (6.4).
		{op: "listWorkspaces", request: sameRequest(http.MethodGet, "/api/v0/workspaces", ""), cells: every(cellOK),
			check: listsItsWorkspaces},
		{op: "checkWorkspaceSlug", request: sameRequest(http.MethodGet, "/api/v0/workspace-slugs/acme", ""), cells: every(cellOK)},
		{op: "createWorkspace", write: true, request: sameRequest(http.MethodPost, "/api/v0/workspaces", `{"name":"New","slug":"new"}`),
			cells: every(cellCreated)},
		{op: "createWorkspace", variant: "creation switched off", write: true,
			config:  func(cfg *config.Config) { cfg.Workspace.CreationEnabled = false },
			request: sameRequest(http.MethodPost, "/api/v0/workspaces", `{"name":"New","slug":"new"}`), cells: every(cellCreationDisabled)},
		// The workspace level.
		{op: "getWorkspace", request: toWorkspace(http.MethodGet, "", ""), cells: inWorkspace(cellOK, cellOK, cellOK),
			check: readsItsRole},
	}
}

// listsItsWorkspaces: each caller's list is the workspaces it is an active
// member of, never acme for the callers it is not visible to, and not the
// deleted gone.
func listsItsWorkspaces(t *testing.T, c caller, answer string) {
	var list struct {
		Data []struct {
			Slug string `json:"slug"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	want := map[caller][]string{callerAdmin: {"acme"}, callerMember: {"acme"}, callerGuest: {"acme"},
		callerNever: {"other"}, callerRemoved: {"other"}, callerDeleted: {}}[c]
	got := []string{}
	for _, w := range list.Data {
		got = append(got, w.Slug)
	}
	if !slices.Equal(got, want) {
		t.Errorf("%s lists %q, want %q", c, got, want)
	}
}

// readsItsRole: the workspace read is acme with the caller's own role.
func readsItsRole(t *testing.T, c caller, answer string) {
	var w struct {
		Slug string `json:"slug"`
		Role int    `json:"role"`
	}
	decodeAnswer(t, answer, &w)
	want := map[caller]int{callerAdmin: 20, callerMember: 15, callerGuest: 5}[c]
	if w.Slug != "acme" || w.Role != want {
		t.Errorf("%s reads %s as role %d, want acme as role %d", c, w.Slug, w.Role, want)
	}
}
````

- [ ] **Step 2: 测试、lint**

Run: `go -C server test -count=1 -run 'TestPermissionMatrix|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEachGap' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/permission_matrix_workspace_test.go
```
```bash
git commit -m "test(M3/P2): the matrix's rows by module, bounded writes, checked answers

Each module's rows are in a file of their own; at most eight writing cells
run an app at once; a row can check what each answer holds, and the matrix
counts the answers it checked, so a harness that skipped them fails.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**：

| 改坏 | 必须失败的测试 |
|---|---|
| 从不运行行的核对（`if false && r.check != nil …`） | `TestPermissionMatrix`（"answers checked, want …"） |
| 只对 problem 运行核对（`got.code != ""`） | `TestPermissionMatrix` |

写格子的并行上限不是正确性的性质：去掉它，格子数在默认并行度（GOMAXPROCS）之下仍通过，只是连接数没有上限（spec 第 6 节）。

**Done when:** 矩阵的 30 格通过，两行的答案被核对并计数；行在自己模块的文件里。

---

### Task 6: `updateWorkspace`：锁 → 判定 → 写

**Files:**
- Create: `server/internal/modules/workspace/adapter/postgres/failures_test.go`、`server/internal/modules/workspace/adapter/postgres/update_workspace_test.go`、`server/internal/modules/workspace/app/lock.go`、`server/internal/modules/workspace/app/update_workspace.go`、`server/internal/modules/workspace/app/update_workspace_test.go`
- Modify: `api/modules/workspace.yaml`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/permission_matrix_workspace_test.go`、`server/internal/bootstrap/workspace_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/workspace/adapter/http/handler.go`、`server/internal/modules/workspace/adapter/http/handler_test.go`、`server/internal/modules/workspace/adapter/http/workspaces.go`、`server/internal/modules/workspace/adapter/http/workspaces_test.go`、`server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql`、`server/internal/modules/workspace/adapter/postgres/store_test.go`、`server/internal/modules/workspace/adapter/postgres/workspaces.go`、`server/internal/modules/workspace/app/fakes_test.go`、`server/internal/modules/workspace/app/ports.go`、`server/internal/modules/workspace/domain/actions.go`、`server/internal/modules/workspace/domain/workspace.go`、`server/internal/modules/workspace/domain/workspace_test.go`、`server/internal/modules/workspace/module.go`、`web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/workspace/adapter/http/gen/server.gen.go`、`server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.8，M3 设计 3.4、3.6 约定二、5.1–5.3）：
  - 接口描述：`updateWorkspace`（`PATCH /api/v0/workspaces/{slug}`，`WorkspaceUpdate{name?, organization_size?, timezone?}`，`additionalProperties: false`，带 `slug` 的请求体是 400 `bad_request`），200 答 `Workspace`；码 `[validation_failed, workspace.not_found, forbidden]`；
  - `domain.WorkspacePatch{Name, OrganizationSize, Timezone *string}`、`domain.CheckWorkspacePatch(p) error`：按建工作区的规则逐个检查设了的字段，全部问题一次 422（`CheckNewWorkspace` 与它共用 `checkOrganizationSize`、`checkTimezone`、`invalid`）；`domain.ActionUpdate = "workspace.update"`；
  - `access` 规则表加 `workspace.update`（管理员）；
  - `app.WorkspaceLocker{LockWorkspaceBySlug}`、`app.WorkspaceUpdater{WorkspaceLocker; UpdateWorkspace(ctx, id, p, by, now)}`；
  - `app/lock.go` 的 `lockAndDecide(ctx, lock, auth, actor, slug, action) (uuid.UUID, shared.Grant, error)`：在 ctx 带的事务里先锁、再判定；锁读到 0 行和 `ErrNotVisible` 都是 `workspace.not_found`，别的失败原样返回；
  - `app.NewUpdateWorkspace(workspaces, auth, tx, clock)`，`Execute(ctx, slug, p)`：`RequireActor` → `CheckWorkspacePatch`（事务之前，只看值）→ 一个事务：`lockAndDecide(LockWorkspaceBySlug, workspace.update)` → `UpdateWorkspace`；答案的角色取自 `Grant`；
  - 查询 `UpdateWorkspace`（`CASE WHEN set_x THEN x ELSE 原值`，`RETURNING` 连同有效成员数）；`(*Store).UpdateWorkspace` 答存下的值；没有这一行是错误，不是 `ErrNotFound`（调用者持着这一行的锁）；
  - HTTP：`UpdateWorkspaceUseCase`、handler `UpdateWorkspace`；`module.go` 接上；
  - 前端文案表加 `forbidden`（M3 设计第 12 节约束 4：第一个声明它的操作）。
- 使用者：Task 8、9 复用 `lockAndDecide`；Task 12 从中分出 `decide`。

**Tests:**
- `domain/workspace_test.go`：`TestCheckWorkspacePatch`：空的补丁通过；每个字段的合法、非法值；三个字段都错时三条一起报。
- `app/update_workspace_test.go`（假实现记下每次调用、参数和是否在事务里）：
  - `TestUpdateWorkspaceLocksThenDecidesThenWrites`：两个调用者、两个工作区、三种补丁（含空补丁）；调用顺序恰好是 `LockWorkspaceBySlug` → `Authorize(workspace.update)` → `UpdateWorkspace`，都在一个事务里；时间是时钟的；角色 20。
  - `TestUpdateWorkspaceRefusals`：非法值（没有事务、没有调用）；不存在、看不到（都是 `workspace.not_found`，看不到的也先锁）；成员的 `forbidden`；锁、判定、写入失败原样返回，不是 404。
  - `TestUpdateWorkspaceWithoutACaller`：401，什么都不调。
- `adapter/postgres/update_workspace_test.go`：`TestUpdateWorkspace`（五种补丁；只改设了的字段；`updated_by_id`、`updated_at`；有效成员数；另一个工作区不变）；`TestUpdateWorkspaceWithoutARowIsAnError`。
- `adapter/postgres/failures_test.go`（新文件）：P1 的 `TestAFailedReadIsAnErrorNotAnAnswer` 从 `store_test.go` 原样移来（`store_test.go` 留在 400 行以内）；`TestAFailedWriteIsAnError`：每个写在已取消的 ctx 上答 `context.Canceled`、没有行（本 Task 是 `UpdateWorkspace`，Task 7、9、11 各加它们的写）。
- `adapter/http/workspaces_test.go`：`TestUpdateWorkspace`（请求体成为补丁，没给的字段是 `nil`；200）；`TestUpdateWorkspaceRefusals`（三个码，经契约核对）；`TestUpdateWorkspaceRefusesTheSlug`（带 `slug` 是 400，用例不被调用）。
- `access/domain/rules_test.go`：`tableCells` 加 `workspace.update` 一行。
- 矩阵：`updateWorkspace` 一行（管理员 200，成员、访客 `forbidden`，另三列 `workspace.not_found`），`renamesIt` 核对管理员的答案（`acme` 改名、角色 20）。
- `bootstrap/workspace_test.go`：`TestAnAdminDemotedMeanwhileCannotUpdateTheWorkspace`（真实数据库、组合出的 app）：一个事务 `FOR NO KEY UPDATE` 锁住 `acme` 并把 alice 降为成员；她的 `PATCH` 等在工作区行上（`WaitForLockWaitOn(…, "workspaces", …)`）；降级提交之后她得到 403 `forbidden`，名字不变。先判定的实现在这里读到她未提交之前的角色（管理员），改了名字（spec 第 3 节第 5 条的变异）。`answer`、`sendInBackground` 是它和 Task 12 共用的辅助。

- [ ] **Step 1: 接口描述**

`api/modules/workspace.yaml`（修改，2 处）：

````old api/modules/workspace.yaml
          description: The workspace, with the caller's role.
````

````new api/modules/workspace.yaml
          description: The workspace, with the caller's role.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Workspace'
        default:
          $ref: '#/components/responses/Problem'
    patch:
      operationId: updateWorkspace
      tags: [workspace]
      summary: Change a workspace's name, organization size or time zone
      description: >-
        For the workspace's admins. The fields given change and the others
        stay; the values follow createWorkspace's rules (validation_failed),
        checked before the workspace is looked at. The slug never changes: a
        body with slug is refused (bad_request). A workspace that does not
        exist, is deleted, or of which the caller is not an active member
        answers workspace.not_found; a member or a guest, forbidden. The role
        is decided after the workspace row is locked, so a caller demoted
        meanwhile is refused.
      security: [{bearer: []}]
      x-problem-codes: [validation_failed, workspace.not_found, forbidden]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/WorkspaceUpdate'
      responses:
        '200':
          description: The workspace as changed, with the caller's role.
````

````old api/modules/workspace.yaml
          description: An IANA time zone name, e.g. from GET /api/v0/timezones; UTC when not given.
          type: string
````

````new api/modules/workspace.yaml
          description: An IANA time zone name, e.g. from GET /api/v0/timezones; UTC when not given.
          type: string
    WorkspaceUpdate:
      type: object
      additionalProperties: false
      properties:
        name:
          description: 1–80 characters, with a letter or a digit, without a web address.
          type: string
        organization_size:
          $ref: '#/components/schemas/OrganizationSize'
        timezone:
          description: An IANA time zone name.
          type: string
````

- [ ] **Step 2: 领域、规则表**

`server/internal/modules/workspace/domain/workspace.go`（修改，4 处）：

````old server/internal/modules/workspace/domain/workspace.go
type NewWorkspace struct {
	Name             string
	Slug             string
	OrganizationSize *string
````

````new server/internal/modules/workspace/domain/workspace.go
type NewWorkspace struct {
	Name             string
	Slug             string
	OrganizationSize *string
	Timezone         *string
}

// WorkspacePatch is a partial update of a workspace (M3 design 5.1): a nil
// field stays as it is. The slug is not in it: it never changes (M3 design
// 3.10).
type WorkspacePatch struct {
	Name             *string
	OrganizationSize *string
````

````old server/internal/modules/workspace/domain/workspace.go
func CheckNewWorkspace(w NewWorkspace) error {
````

````new server/internal/modules/workspace/domain/workspace.go
func CheckNewWorkspace(w NewWorkspace) error {
	return invalid(checkName(w.Name), checkSlug(w.Slug), checkOrganizationSize(w.OrganizationSize), checkTimezone(w.Timezone))
}

// CheckWorkspacePatch checks the fields p sets by CheckNewWorkspace's
// rules, every problem at once, as one 422 validation_failed.
func CheckWorkspacePatch(p WorkspacePatch) error {
	var name *shared.FieldError
	if p.Name != nil {
		name = checkName(*p.Name)
	}
	return invalid(name, checkOrganizationSize(p.OrganizationSize), checkTimezone(p.Timezone))
}

// invalid is the 422 of the problems found, in order, or nil when there is
// none.
func invalid(found ...*shared.FieldError) error {
````

````old server/internal/modules/workspace/domain/workspace.go
	if f := checkName(w.Name); f != nil {
		fields = append(fields, *f)
	}
	if f := checkSlug(w.Slug); f != nil {
		fields = append(fields, *f)
	}
	if w.OrganizationSize != nil && !slices.Contains(organizationSizes, *w.OrganizationSize) {
		fields = append(fields, shared.FieldError{Field: "organization_size", Code: shared.FieldInvalidFormat, Message: "is not a known organization size"})
	}
	if w.Timezone != nil && !shared.ValidTimezone(*w.Timezone) {
		fields = append(fields, shared.FieldError{Field: "timezone", Code: shared.FieldInvalidFormat, Message: "is not a known time zone"})
````

````new server/internal/modules/workspace/domain/workspace.go
	for _, f := range found {
		if f != nil {
			fields = append(fields, *f)
		}
````

````old server/internal/modules/workspace/domain/workspace.go
		return shared.Invalid(fields...)
````

````new server/internal/modules/workspace/domain/workspace.go
		return shared.Invalid(fields...)
	}
	return nil
}

func checkOrganizationSize(size *string) *shared.FieldError {
	if size != nil && !slices.Contains(organizationSizes, *size) {
		return &shared.FieldError{Field: "organization_size", Code: shared.FieldInvalidFormat, Message: "is not a known organization size"}
	}
	return nil
}

func checkTimezone(zone *string) *shared.FieldError {
	if zone != nil && !shared.ValidTimezone(*zone) {
		return &shared.FieldError{Field: "timezone", Code: shared.FieldInvalidFormat, Message: "is not a known time zone"}
````

`server/internal/modules/workspace/domain/workspace_test.go`（修改，1 处）：

````old server/internal/modules/workspace/domain/workspace_test.go
}

func TestCheckSlug(t *testing.T) {
````

````new server/internal/modules/workspace/domain/workspace_test.go
}

// A patch is checked by the rules of a new workspace, field by field: a
// field left nil is not checked, so the empty patch passes.
func TestCheckWorkspacePatch(t *testing.T) {
	for _, p := range []WorkspacePatch{
		{},
		{Name: ptr("研发部")},
		{Name: ptr(strings.Repeat("工", 80)), OrganizationSize: ptr("500+"), Timezone: ptr("Asia/Shanghai")},
		{OrganizationSize: ptr("Just myself")},
		{Timezone: ptr("UTC")},
	} {
		if err := CheckWorkspacePatch(p); err != nil {
			t.Errorf("CheckWorkspacePatch(%+v) = %v, want nil", p, err)
		}
	}
	tests := []struct {
		name string
		p    WorkspacePatch
		want []shared.FieldError
	}{
		{"empty name", WorkspacePatch{Name: ptr("")}, []shared.FieldError{{Field: "name", Code: "too_short", Message: "must not be empty"}}},
		{"name with a web address", WorkspacePatch{Name: ptr("acme.io"), Timezone: ptr("UTC")},
			[]shared.FieldError{{Field: "name", Code: "contains_url", Message: "must not contain a web address"}}},
		{"unknown organization size", WorkspacePatch{OrganizationSize: ptr("1000+")},
			[]shared.FieldError{{Field: "organization_size", Code: "invalid_format", Message: "is not a known organization size"}}},
		{"the host's zone", WorkspacePatch{Name: ptr("Acme"), Timezone: ptr("Local")},
			[]shared.FieldError{{Field: "timezone", Code: "invalid_format", Message: "is not a known time zone"}}},
		{"all at once", WorkspacePatch{Name: ptr("-"), OrganizationSize: ptr(""), Timezone: ptr("")}, []shared.FieldError{
			{Field: "name", Code: "invalid_format", Message: "must contain a letter or a digit"},
			{Field: "organization_size", Code: "invalid_format", Message: "is not a known organization size"},
			{Field: "timezone", Code: "invalid_format", Message: "is not a known time zone"},
		}},
	}
	for _, tt := range tests {
		err := CheckWorkspacePatch(tt.p)
		var se *shared.Error
		if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || !slices.Equal(se.Fields, tt.want) {
			t.Errorf("%s: CheckWorkspacePatch(%+v) = %#v, want validation_failed with %v", tt.name, tt.p, err, tt.want)
		}
	}
}

func TestCheckSlug(t *testing.T) {
````

`server/internal/modules/workspace/domain/actions.go`（修改，2 处）：

````old server/internal/modules/workspace/domain/actions.go
	ActionRead shared.Action = "workspace.read"
````

````new server/internal/modules/workspace/domain/actions.go
	ActionRead shared.Action = "workspace.read"
	// ActionUpdate is changing a workspace: updateWorkspace.
	ActionUpdate shared.Action = "workspace.update"
````

````old server/internal/modules/workspace/domain/actions.go
	return []shared.Action{ActionRead}
````

````new server/internal/modules/workspace/domain/actions.go
	return []shared.Action{ActionRead, ActionUpdate}
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"workspace.read": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
````

````new server/internal/modules/access/domain/rules.go
	"workspace.read":   {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	"workspace.update": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"workspace.read": {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
````

````new server/internal/modules/access/domain/rules_test.go
	"workspace.read":   {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
	"workspace.update": {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
````

- [ ] **Step 3: 查询和生成**

`server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql
WHERE w.slug = sqlc.arg(slug) AND w.deleted_at IS NULL;

````

````new server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql
WHERE w.slug = sqlc.arg(slug) AND w.deleted_at IS NULL;

-- name: UpdateWorkspace :one
-- updateWorkspace, under the workspace's FOR NO KEY UPDATE: only the fields that are set change (M2 design 3.14).
-- RETURNING gives the values as stored and the number of active members.
UPDATE workspaces w
SET name              = CASE WHEN sqlc.arg(set_name)::boolean THEN sqlc.arg(name)::text ELSE w.name END,
    organization_size = CASE WHEN sqlc.arg(set_organization_size)::boolean THEN sqlc.arg(organization_size)::text
                        ELSE w.organization_size END,
    timezone          = CASE WHEN sqlc.arg(set_timezone)::boolean THEN sqlc.arg(timezone)::text ELSE w.timezone END,
    updated_by_id     = sqlc.arg(updated_by),
    updated_at        = sqlc.arg(now)
WHERE w.id = sqlc.arg(id)
RETURNING w.id, w.name, w.slug, w.organization_size, w.timezone, w.created_at, w.updated_at,
          (SELECT count(*) FROM workspace_members c
           WHERE c.workspace_id = w.id AND c.is_active AND c.deleted_at IS NULL) AS total_members;

````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `caf1a5bf50e4cafd71e62a723321e895974669c75a9b2b730af2250f799708cf` | 1123 | `api/dist/openapi.yaml` |
| `cfd77ee2c644e0fa7f26b37328e20338b149431a9f670602cb674ba5c97ecd02` | 27 | `server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go` |
| `d0e1c5af1a497350903628e6f54a2ab9c1f87d01d6761b2701b4332f9fb776c0` | 880 | `server/internal/modules/workspace/adapter/http/gen/server.gen.go` |
| `c56599c998f8495a25ddbcae992ed303acdc5df8fe03e546ae9388b6ec16908b` | 263 | `server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go` |
| `3b40d19d28553f3987c5a8e577d39ed6c35a4bc0ddbde68e11674371c9dcaf7f` | 1174 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go server/internal/modules/workspace/adapter/http/gen/server.gen.go server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 4: 端口、锁 → 判定、用例**

`server/internal/modules/workspace/app/ports.go`（修改，1 处）：

````old server/internal/modules/workspace/app/ports.go
}

// SlugChecker tells whether a slug is taken.
````

````new server/internal/modules/workspace/app/ports.go
}

// WorkspaceLocker takes the parent locks of the writes on a workspace (M3
// design 3.6 convention 2), in the transaction ctx carries: each locks the
// undeleted workspace with slug until the transaction ends and returns its
// id; ErrNotFound when there is none, also when it was deleted while the
// lock waited.
type WorkspaceLocker interface {
	// LockWorkspaceBySlug locks FOR NO KEY UPDATE: for a write of the
	// workspace row itself or of a membership.
	LockWorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, error)
}

// WorkspaceUpdater changes a workspace row under its lock.
type WorkspaceUpdater interface {
	WorkspaceLocker
	// UpdateWorkspace applies p to the workspace id, by the account by at
	// now, and returns it as stored with its number of active members,
	// without a role.
	UpdateWorkspace(ctx context.Context, id uuid.UUID, p domain.WorkspacePatch, by uuid.UUID, now time.Time) (domain.Workspace, error)
}

// SlugChecker tells whether a slug is taken.
````

`server/internal/modules/workspace/app/lock.go`（新文件，36 行）：

````file server/internal/modules/workspace/app/lock.go
package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// lockAndDecide is the first two steps of every write on a workspace (M3
// design 3.6 convention 2), in the transaction ctx carries: lock locks the
// undeleted workspace with slug, then the Authorizer decides action on it
// for actor and reads the role committed before the lock was granted. It
// returns the workspace's id and the grant. A workspace that is not there,
// deleted, or not visible to actor is domain.ErrNotFound; a role the rule
// does not allow is the Authorizer's shared.Forbidden.
func lockAndDecide(ctx context.Context, lock func(ctx context.Context, slug string) (uuid.UUID, error),
	auth shared.Authorizer, actor shared.Actor, slug string, action shared.Action) (uuid.UUID, shared.Grant, error) {
	id, err := lock(ctx, slug)
	switch {
	case errors.Is(err, ErrNotFound):
		return uuid.UUID{}, shared.Grant{}, domain.ErrNotFound
	case err != nil:
		return uuid.UUID{}, shared.Grant{}, err
	}
	grant, err := auth.Authorize(ctx, actor, action, shared.Target{WorkspaceID: id})
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return uuid.UUID{}, shared.Grant{}, domain.ErrNotFound
	case err != nil:
		return uuid.UUID{}, shared.Grant{}, err
	}
	return id, grant, nil
}
````

`server/internal/modules/workspace/app/update_workspace.go`（新文件，53 行）：

````file server/internal/modules/workspace/app/update_workspace.go
package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateWorkspace changes a workspace's name, organization size or time
// zone: PATCH /api/v0/workspaces/{slug}.
type UpdateWorkspace struct {
	workspaces WorkspaceUpdater
	auth       shared.Authorizer
	tx         shared.TxManager
	clock      Clock
}

// NewUpdateWorkspace returns the use case.
func NewUpdateWorkspace(workspaces WorkspaceUpdater, auth shared.Authorizer, tx shared.TxManager, clock Clock) *UpdateWorkspace {
	return &UpdateWorkspace{workspaces: workspaces, auth: auth, tx: tx, clock: clock}
}

// Execute checks p, then in one transaction (M3 design 3.6): the workspace
// row FOR NO KEY UPDATE, the decision on workspace.update, the change. The
// answer carries the caller's role from the grant. A check of the values
// alone comes first: it tells nothing about the workspace.
func (u *UpdateWorkspace) Execute(ctx context.Context, slug string, p domain.WorkspacePatch) (domain.Workspace, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Workspace{}, err
	}
	if err := domain.CheckWorkspacePatch(p); err != nil {
		return domain.Workspace{}, err
	}
	now := u.clock.Now()
	var updated domain.Workspace
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		id, grant, err := lockAndDecide(ctx, u.workspaces.LockWorkspaceBySlug, u.auth, actor, slug, domain.ActionUpdate)
		if err != nil {
			return err
		}
		if updated, err = u.workspaces.UpdateWorkspace(ctx, id, p, actor.UserID, now); err != nil {
			return err
		}
		updated.Role = grant.WorkspaceRole
		return nil
	})
	if err != nil {
		return domain.Workspace{}, err
	}
	return updated, nil
}
````

`server/internal/modules/workspace/app/fakes_test.go`（修改，4 处）：

````old server/internal/modules/workspace/app/fakes_test.go
	workspaces []domain.Workspace // by slug for WorkspaceBySlug and SlugTaken
````

````new server/internal/modules/workspace/app/fakes_test.go
	workspaces []domain.Workspace // by slug for WorkspaceBySlug, SlugTaken and the locks; by id for UpdateWorkspace
````

````old server/internal/modules/workspace/app/fakes_test.go
	memberErr  error               // for CreateMember, which then stores nothing
````

````new server/internal/modules/workspace/app/fakes_test.go
	memberErr  error               // for CreateMember, which then stores nothing
	updateErr  error               // for UpdateWorkspace
````

````old server/internal/modules/workspace/app/fakes_test.go
	slugErrs   map[string]error    // by slug, for WorkspaceBySlug and SlugTaken
````

````new server/internal/modules/workspace/app/fakes_test.go
	slugErrs   map[string]error    // by slug, for WorkspaceBySlug and SlugTaken
	lockErrs   map[string]error    // by slug, for the locks
````

````old server/internal/modules/workspace/app/fakes_test.go
}

// grantKey is one (user, workspace) pair of fakeAuthorizer.
````

````new server/internal/modules/workspace/app/fakes_test.go
}

func (f *fakeWorkspaces) LockWorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	f.log.add(ctx, "LockWorkspaceBySlug %s", slug)
	return f.lock(slug)
}

// lock answers the id of the workspace with slug, app.ErrNotFound when it
// holds none, and the error set for slug wrapped as the store wraps it.
func (f *fakeWorkspaces) lock(slug string) (uuid.UUID, error) {
	if err := f.lockErrs[slug]; err != nil {
		return uuid.UUID{}, fmt.Errorf("lock the workspace row: %w", err)
	}
	i := slices.IndexFunc(f.workspaces, func(w domain.Workspace) bool { return w.Slug == slug })
	if i < 0 {
		return uuid.UUID{}, app.ErrNotFound
	}
	return f.workspaces[i].ID, nil
}

func (f *fakeWorkspaces) UpdateWorkspace(ctx context.Context, id uuid.UUID, p domain.WorkspacePatch, by uuid.UUID, now time.Time) (domain.Workspace, error) {
	f.log.add(ctx, "UpdateWorkspace %s name=%s size=%s timezone=%s by %s at %s", id, show(p.Name), show(p.OrganizationSize), show(p.Timezone),
		by, now.Format(time.RFC3339Nano))
	if f.updateErr != nil {
		return domain.Workspace{}, fmt.Errorf("update workspace: %w", f.updateErr)
	}
	i := slices.IndexFunc(f.workspaces, func(w domain.Workspace) bool { return w.ID == id })
	if i < 0 {
		return domain.Workspace{}, fmt.Errorf("update workspace %s: no such row", id)
	}
	w := f.workspaces[i]
	if p.Name != nil {
		w.Name = *p.Name
	}
	if p.OrganizationSize != nil {
		w.OrganizationSize = p.OrganizationSize
	}
	if p.Timezone != nil {
		w.Timezone = *p.Timezone
	}
	w.UpdatedAt = now
	return w, nil
}

// show is *s quoted, or <nil>.
func show(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%q", *s)
}

// grantKey is one (user, workspace) pair of fakeAuthorizer.
````

`server/internal/modules/workspace/app/update_workspace_test.go`（新文件，147 行）：

````file server/internal/modules/workspace/app/update_workspace_test.go
package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func ptr[T any](v T) *T { return &v }

// updateFixture is UpdateWorkspace over fakes sharing one log: alice is
// acme's admin, bob beta's.
type updateFixture struct {
	log        *callLog
	tx         *fakeTx
	workspaces *fakeWorkspaces
	auth       *fakeAuthorizer
}

func newUpdate() (*app.UpdateWorkspace, *updateFixture) {
	log := &callLog{}
	f := &updateFixture{log: log, tx: &fakeTx{}, workspaces: &fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme, beta}},
		auth: &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{
			{alice.ID, acme.ID}: {WorkspaceRole: shared.RoleAdmin},
			{bob.ID, beta.ID}:   {WorkspaceRole: shared.RoleAdmin},
		}}}
	return app.NewUpdateWorkspace(f.workspaces, f.auth, f.tx, clocktest.At(now)), f
}

// lockedDecision is the calls of the first two steps on w for user, in the
// transaction: the lock, then the decision on action.
func lockedDecision(user app.AccountState, w domain.Workspace, lock string, action shared.Action) []string {
	return []string{
		lock + " " + w.Slug,
		"Authorize " + user.ID.String() + " " + string(action) + " on " + w.ID.String() + "/" + uuid.Nil().String(),
	}
}

// UpdateWorkspace locks the workspace FOR NO KEY UPDATE, then decides, then
// writes, all in one transaction (M3 design 3.6): the patch is applied by
// the caller at the clock's now, and the answer carries the caller's role.
// Two callers, two workspaces.
func TestUpdateWorkspaceLocksThenDecidesThenWrites(t *testing.T) {
	size := "11-50"
	tests := []struct {
		user app.AccountState
		w    domain.Workspace
		p    domain.WorkspacePatch
		call string
	}{
		{alice, acme, domain.WorkspacePatch{Name: ptr("Acme Inc"), OrganizationSize: &size, Timezone: ptr("Asia/Shanghai")},
			`name="Acme Inc" size="11-50" timezone="Asia/Shanghai"`},
		{bob, beta, domain.WorkspacePatch{Timezone: ptr("Europe/Paris")}, `name=<nil> size=<nil> timezone="Europe/Paris"`},
		{alice, acme, domain.WorkspacePatch{}, `name=<nil> size=<nil> timezone=<nil>`},
	}
	for _, tt := range tests {
		uc, f := newUpdate()
		got, err := uc.Execute(as(tt.user), tt.w.Slug, tt.p)
		want := tt.w
		if tt.p.Name != nil {
			want.Name = *tt.p.Name
		}
		if tt.p.OrganizationSize != nil {
			want.OrganizationSize = tt.p.OrganizationSize
		}
		if tt.p.Timezone != nil {
			want.Timezone = *tt.p.Timezone
		}
		want.Role, want.UpdatedAt = shared.RoleAdmin, now
		if err != nil || got != want {
			t.Errorf("%s, %s: Execute() = %+v, %v; want %+v", tt.user.Email, tt.w.Slug, got, err, want)
		}
		wantCalls := append(lockedDecision(tt.user, tt.w, "LockWorkspaceBySlug", domain.ActionUpdate),
			"UpdateWorkspace "+tt.w.ID.String()+" "+tt.call+" by "+tt.user.ID.String()+" at "+now.Format(time.RFC3339Nano))
		if !slices.Equal(f.log.calls, wantCalls) || f.tx.calls != 1 {
			t.Errorf("%s, %s: calls = %q in %d transactions, want %q in one", tt.user.Email, tt.w.Slug, f.log.calls, f.tx.calls, wantCalls)
		}
	}
}

// Each refusal, and each failure, is the answer, with nothing written: the
// values' check first, without a transaction; then a workspace that is not
// there and one the caller cannot see, both workspace.not_found; a member's
// forbidden; a failed lock, decision or write, never turned into a 404.
func TestUpdateWorkspaceRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	invalid := shared.Invalid(shared.FieldError{Field: "name", Code: shared.FieldTooShort, Message: "must not be empty"})
	rename := domain.WorkspacePatch{Name: ptr("Renamed")}
	tests := []struct {
		name   string
		user   app.AccountState
		slug   string
		p      domain.WorkspacePatch
		set    func(f *updateFixture)
		want   error
		calls  []string
		inTxes int
	}{
		{"invalid values", alice, "acme", domain.WorkspacePatch{Name: ptr("")}, nil, invalid, nil, 0},
		{"no such workspace", alice, "nothing", rename, nil, domain.ErrNotFound, []string{"LockWorkspaceBySlug nothing"}, 1},
		{"not visible", bob, "acme", rename, nil, domain.ErrNotFound, lockedDecision(bob, acme, "LockWorkspaceBySlug", domain.ActionUpdate), 1},
		{"forbidden", carol, "acme", rename, func(f *updateFixture) { f.auth.errs = map[grantKey]error{{carol.ID, acme.ID}: shared.Forbidden()} },
			shared.Forbidden(), lockedDecision(carol, acme, "LockWorkspaceBySlug", domain.ActionUpdate), 1},
		{"the lock failed", alice, "acme", rename, func(f *updateFixture) { f.workspaces.lockErrs = map[string]error{"acme": failure} },
			failure, []string{"LockWorkspaceBySlug acme"}, 1},
		{"the Authorizer failed", alice, "acme", rename, func(f *updateFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: failure} },
			failure, lockedDecision(alice, acme, "LockWorkspaceBySlug", domain.ActionUpdate), 1},
		{"the write failed", alice, "acme", rename, func(f *updateFixture) { f.workspaces.updateErr = failure }, failure,
			append(lockedDecision(alice, acme, "LockWorkspaceBySlug", domain.ActionUpdate),
				"UpdateWorkspace "+acme.ID.String()+` name="Renamed" size=<nil> timezone=<nil> by `+alice.ID.String()+" at "+now.Format(time.RFC3339Nano)), 1},
	}
	for _, tt := range tests {
		uc, f := newUpdate()
		if tt.set != nil {
			tt.set(f)
		}
		got, err := uc.Execute(as(tt.user), tt.slug, tt.p)
		if !errors.Is(err, tt.want) || got != (domain.Workspace{}) {
			t.Errorf("%s: Execute() = %+v, %v; want no workspace and %v", tt.name, got, err, tt.want)
		}
		if tt.want == failure && errors.Is(err, domain.ErrNotFound) {
			t.Errorf("%s: Execute() = %v, which is also workspace.not_found", tt.name, err)
		}
		if !slices.Equal(f.log.calls, tt.calls) || f.tx.calls != tt.inTxes {
			t.Errorf("%s: calls = %q in %d transactions, want %q in %d", tt.name, f.log.calls, f.tx.calls, tt.calls, tt.inTxes)
		}
	}
}

// Without a caller, UpdateWorkspace is 401 and touches nothing, whatever the
// values.
func TestUpdateWorkspaceWithoutACaller(t *testing.T) {
	for _, p := range []domain.WorkspacePatch{{Name: ptr("Renamed")}, {Name: ptr("")}} {
		uc, f := newUpdate()
		if _, err := uc.Execute(context.Background(), "acme", p); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 || f.tx.calls != 0 {
			t.Errorf("Execute(%+v) without an actor = %v, calls %q in %d transactions; want 401 unauthorized and nothing", p, err, f.log.calls, f.tx.calls)
		}
	}
}
````

- [ ] **Step 5: 存储**

`server/internal/modules/workspace/adapter/postgres/workspaces.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/postgres/workspaces.go
	"fmt"
````

````new server/internal/modules/workspace/adapter/postgres/workspaces.go
	"fmt"
	"time"
````

````old server/internal/modules/workspace/adapter/postgres/workspaces.go
}

// SlugTaken reports whether an undeleted workspace has slug.
````

````new server/internal/modules/workspace/adapter/postgres/workspaces.go
}

// UpdateWorkspace applies p to the workspace id, by the account by at now,
// and returns it as stored with its number of active members, without a
// role. The caller holds the workspace's lock, so the row is there; its
// absence is an error, not app.ErrNotFound. The domain checked every value.
func (s *Store) UpdateWorkspace(ctx context.Context, id uuid.UUID, p domain.WorkspacePatch, by uuid.UUID, now time.Time) (domain.Workspace, error) {
	r, err := s.queries(ctx).UpdateWorkspace(ctx, gen.UpdateWorkspaceParams{
		SetName: p.Name != nil, Name: deref(p.Name),
		SetOrganizationSize: p.OrganizationSize != nil, OrganizationSize: deref(p.OrganizationSize),
		SetTimezone: p.Timezone != nil, Timezone: deref(p.Timezone),
		UpdatedBy: &by, Now: now, ID: id,
	})
	if err != nil {
		return domain.Workspace{}, fmt.Errorf("update workspace: %w", err)
	}
	return domain.Workspace{
		ID: r.ID, Name: r.Name, Slug: r.Slug, OrganizationSize: r.OrganizationSize, Timezone: r.Timezone,
		TotalMembers: int(r.TotalMembers), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, nil
}

// deref is the value p points at, or the zero value for nil.
func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

// SlugTaken reports whether an undeleted workspace has slug.
````

`server/internal/modules/workspace/adapter/postgres/update_workspace_test.go`（新文件，87 行）：

````file server/internal/modules/workspace/adapter/postgres/update_workspace_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateWorkspace changes the fields the patch sets and no other, of that
// workspace only; it writes the updater and the clock's time, and answers
// the row as stored with the number of active members.
func TestUpdateWorkspace(t *testing.T) {
	size, other := "2-10", "500+"
	tests := []struct {
		name string
		p    domain.WorkspacePatch
		want func(w domain.Workspace) domain.Workspace
	}{
		{"every field", domain.WorkspacePatch{Name: ptr("Acme Inc"), OrganizationSize: &other, Timezone: ptr("Europe/Paris")},
			func(w domain.Workspace) domain.Workspace {
				w.Name, w.OrganizationSize, w.Timezone = "Acme Inc", &other, "Europe/Paris"
				return w
			}},
		{"the name", domain.WorkspacePatch{Name: ptr("研发部")}, func(w domain.Workspace) domain.Workspace { w.Name = "研发部"; return w }},
		{"the organization size", domain.WorkspacePatch{OrganizationSize: &other},
			func(w domain.Workspace) domain.Workspace { w.OrganizationSize = &other; return w }},
		{"the time zone", domain.WorkspacePatch{Timezone: ptr("Asia/Tokyo")},
			func(w domain.Workspace) domain.Workspace { w.Timezone = "Asia/Tokyo"; return w }},
		{"nothing", domain.WorkspacePatch{}, func(w domain.Workspace) domain.Workspace { return w }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
			acme, err := s.CreateWorkspace(context.Background(), app.WorkspaceRow{
				ID: uuid.NewV7(), Name: "Acme", Slug: "acme", OrganizationSize: &size, Timezone: "UTC", CreatedBy: alice, Now: now,
			})
			if err != nil {
				t.Fatal(err)
			}
			join(t, s, acme.ID, alice, shared.RoleAdmin)
			join(t, s, acme.ID, bob, shared.RoleMember)
			join(t, s, acme.ID, carol, shared.RoleGuest)
			exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE member_id = $1", carol)
			beta := newWorkspace(t, s, "Beta", "beta", alice)
			later := now.Add(time.Hour)

			got, err := s.UpdateWorkspace(context.Background(), acme.ID, tt.p, bob, later)

			want := tt.want(acme)
			want.TotalMembers, want.UpdatedAt = 2, later
			if err != nil || !sameWorkspace(got, want) {
				t.Fatalf("UpdateWorkspace() = %+v, %v; want %+v", got, err, want)
			}
			if read, err := s.WorkspaceBySlug(context.Background(), "acme"); err != nil || !sameWorkspace(read, want) {
				t.Errorf("read back: %+v, %v; want %+v", read, err, want)
			}
			var updatedBy, createdBy uuid.UUID
			if err := pool.QueryRow(context.Background(), "SELECT updated_by_id, created_by_id FROM workspaces WHERE id = $1", acme.ID).
				Scan(&updatedBy, &createdBy); err != nil || updatedBy != bob || createdBy != alice {
				t.Errorf("updated_by_id %s, created_by_id %s, %v; want bob, alice", updatedBy, createdBy, err)
			}
			beta.TotalMembers = 1
			if read, err := s.WorkspaceBySlug(context.Background(), "beta"); err != nil || !sameWorkspace(read, beta) {
				t.Errorf("the other workspace: %+v, %v; want it unchanged, %+v", read, err, beta)
			}
		})
	}
}

// A workspace id without a row is an error, not app.ErrNotFound: the caller
// holds the row's lock, so its absence is not an answer to give.
func TestUpdateWorkspaceWithoutARowIsAnError(t *testing.T) {
	s, _ := newStore(t)
	got, err := s.UpdateWorkspace(context.Background(), uuid.NewV7(), domain.WorkspacePatch{Name: ptr("Acme")}, uuid.NewV7(), now)
	if err == nil || errors.Is(err, app.ErrNotFound) || got != (domain.Workspace{}) {
		t.Errorf("UpdateWorkspace() = %+v, %v; want an error that is not app.ErrNotFound", got, err)
	}
}

func ptr[T any](v T) *T { return &v }
````

`server/internal/modules/workspace/adapter/postgres/store_test.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/store_test.go

// A read that fails answers its error, never a plausible answer: not "not a
// member", which the Authorizer would turn into workspace.not_found; not
// "free", "none" or app.ErrNotFound. Each read runs on a cancelled context
// against a workspace alice administers, so that the right answer is none
// of the zero values.
func TestAFailedReadIsAnErrorNotAnAnswer(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	w := newWorkspace(t, s, "Acme", "acme", alice)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	failed := func(err error) bool { return errors.Is(err, context.Canceled) }

	if role, ok, err := s.ActiveRole(cancelled, w.ID, alice); !failed(err) || ok || role != 0 {
		t.Errorf("ActiveRole() = %d, %v, %v; want context.Canceled, not a member", role, ok, err)
	}
	if taken, err := s.SlugTaken(cancelled, "acme"); !failed(err) || taken {
		t.Errorf("SlugTaken() = %v, %v; want context.Canceled", taken, err)
	}
	if list, err := s.ListWorkspaces(cancelled, alice); !failed(err) || list != nil {
		t.Errorf("ListWorkspaces() = %v, %v; want context.Canceled, no list", list, err)
	}
	if got, err := s.WorkspaceBySlug(cancelled, "acme"); !failed(err) || errors.Is(err, app.ErrNotFound) || !sameWorkspace(got, domain.Workspace{}) {
		t.Errorf("WorkspaceBySlug() = %+v, %v; want context.Canceled, not app.ErrNotFound", got, err)
	}
}

````

````new server/internal/modules/workspace/adapter/postgres/store_test.go

````

`server/internal/modules/workspace/adapter/postgres/failures_test.go`（新文件，54 行）：

````file server/internal/modules/workspace/adapter/postgres/failures_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// A read that fails answers its error, never a plausible answer: not "not a
// member", which the Authorizer would turn into workspace.not_found; not
// "free", "none" or app.ErrNotFound. Each read runs on a cancelled context
// against a workspace alice administers, so that the right answer is none
// of the zero values.
func TestAFailedReadIsAnErrorNotAnAnswer(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	w := newWorkspace(t, s, "Acme", "acme", alice)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	failed := func(err error) bool { return errors.Is(err, context.Canceled) }

	if role, ok, err := s.ActiveRole(cancelled, w.ID, alice); !failed(err) || ok || role != 0 {
		t.Errorf("ActiveRole() = %d, %v, %v; want context.Canceled, not a member", role, ok, err)
	}
	if taken, err := s.SlugTaken(cancelled, "acme"); !failed(err) || taken {
		t.Errorf("SlugTaken() = %v, %v; want context.Canceled", taken, err)
	}
	if list, err := s.ListWorkspaces(cancelled, alice); !failed(err) || list != nil {
		t.Errorf("ListWorkspaces() = %v, %v; want context.Canceled, no list", list, err)
	}
	if got, err := s.WorkspaceBySlug(cancelled, "acme"); !failed(err) || errors.Is(err, app.ErrNotFound) || !sameWorkspace(got, domain.Workspace{}) {
		t.Errorf("WorkspaceBySlug() = %+v, %v; want context.Canceled, not app.ErrNotFound", got, err)
	}
}

// A write that fails answers its error, never nil, which a use case would
// take for done, and never a row. Each write runs on a cancelled context
// against a workspace alice administers.
func TestAFailedWriteIsAnError(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	w := newWorkspace(t, s, "Acme", "acme", alice)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	failed := func(err error) bool { return errors.Is(err, context.Canceled) }

	if got, err := s.UpdateWorkspace(cancelled, w.ID, domain.WorkspacePatch{Name: ptr("Renamed")}, alice, now); !failed(err) ||
		!sameWorkspace(got, domain.Workspace{}) {
		t.Errorf("UpdateWorkspace() = %+v, %v; want context.Canceled", got, err)
	}
}
````

- [ ] **Step 6: HTTP 一侧和接线**

`server/internal/modules/workspace/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/http/handler.go
}

// CheckSlugUseCase is app.CheckSlug.
````

````new server/internal/modules/workspace/adapter/http/handler.go
}

// UpdateWorkspaceUseCase is app.UpdateWorkspace.
type UpdateWorkspaceUseCase interface {
	Execute(ctx context.Context, slug string, p domain.WorkspacePatch) (domain.Workspace, error)
}

// CheckSlugUseCase is app.CheckSlug.
````

````old server/internal/modules/workspace/adapter/http/handler.go
	GetWorkspace    GetWorkspaceUseCase
````

````new server/internal/modules/workspace/adapter/http/handler.go
	GetWorkspace    GetWorkspaceUseCase
	UpdateWorkspace UpdateWorkspaceUseCase
````

`server/internal/modules/workspace/adapter/http/workspaces.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/http/workspaces.go
}

// CheckWorkspaceSlug serves GET /api/v0/workspace-slugs/{slug}.
````

````new server/internal/modules/workspace/adapter/http/workspaces.go
}

// UpdateWorkspace serves PATCH /api/v0/workspaces/{slug}.
func (h handler) UpdateWorkspace(ctx context.Context, req gen.UpdateWorkspaceRequestObject) (gen.UpdateWorkspaceResponseObject, error) {
	p := domain.WorkspacePatch{Name: req.Body.Name, Timezone: req.Body.Timezone}
	if size := req.Body.OrganizationSize; size != nil {
		s := string(*size)
		p.OrganizationSize = &s
	}
	w, err := h.uc.UpdateWorkspace.Execute(ctx, req.Slug, p)
	if err != nil {
		return nil, err
	}
	return gen.UpdateWorkspace200JSONResponse(workspace(w)), nil
}

// CheckWorkspaceSlug serves GET /api/v0/workspace-slugs/{slug}.
````

`server/internal/modules/workspace/adapter/http/handler_test.go`（修改，4 处）：

````old server/internal/modules/workspace/adapter/http/handler_test.go
	get    *fakeGet
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
	get    *fakeGet
	update *fakeUpdate
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
}

type fakeCheck struct {
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
}

type fakeUpdate struct {
	calls  []string // "caller slug"
	got    []domain.WorkspacePatch
	answer domain.Workspace
	err    error
}

func (f *fakeUpdate) Execute(ctx context.Context, slug string, p domain.WorkspacePatch) (domain.Workspace, error) {
	f.calls = append(f.calls, caller(ctx)+" "+slug)
	f.got = append(f.got, p)
	return f.answer, f.err
}

type fakeCheck struct {
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
		f.get = &fakeGet{}
	}
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
		f.get = &fakeGet{}
	}
	if f.update == nil {
		f.update = &fakeUpdate{}
	}
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
		ListWorkspaces: f.list, CreateWorkspace: f.create, GetWorkspace: f.get, CheckSlug: f.check,
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
		ListWorkspaces: f.list, CreateWorkspace: f.create, GetWorkspace: f.get, UpdateWorkspace: f.update, CheckSlug: f.check,
````

`server/internal/modules/workspace/adapter/http/workspaces_test.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/http/workspaces_test.go
}

// The availability, and the reason when the slug is not available; the slug
````

````new server/internal/modules/workspace/adapter/http/workspaces_test.go
}

// The body becomes the patch, a field left out nil, for the caller and the
// slug of the path; the answer is 200 with the workspace.
func TestUpdateWorkspace(t *testing.T) {
	update := &fakeUpdate{answer: acme}
	h := newServer(t, fakes{update: update})
	for _, tt := range []struct{ token, path, body string }{
		{"alice", "/api/v0/workspaces/acme", `{"name":"Acme","organization_size":"2-10","timezone":"Asia/Shanghai"}`},
		{"bob", "/api/v0/workspaces/beta", `{"timezone":"UTC"}`},
		{"alice", "/api/v0/workspaces/acme", `{}`},
	} {
		res, body := do(t, h, request(http.MethodPatch, tt.path, tt.token, tt.body))
		if res.StatusCode != http.StatusOK || body != acmeJSON+"\n" {
			t.Errorf("%s PATCH %s %s = %d %s, want 200 %s", tt.token, tt.path, tt.body, res.StatusCode, body, acmeJSON)
		}
	}
	zone, utc := "Asia/Shanghai", "UTC"
	want := []domain.WorkspacePatch{{Name: &acme.Name, OrganizationSize: &size, Timezone: &zone}, {Timezone: &utc}, {}}
	if len(update.got) != len(want) {
		t.Fatalf("patches = %+v, want %+v", update.got, want)
	}
	for i := range want {
		if !samePatch(update.got[i], want[i]) {
			t.Errorf("patch %d = %+v, want %+v", i, update.got[i], want[i])
		}
	}
	if calls := []string{"alice acme", "bob beta", "alice acme"}; !slices.Equal(update.calls, calls) {
		t.Errorf("calls = %q, want %q", update.calls, calls)
	}
}

func samePatch(a, b domain.WorkspacePatch) bool {
	same := func(x, y *string) bool { return (x == nil) == (y == nil) && (x == nil || *x == *y) }
	return same(a.Name, b.Name) && same(a.OrganizationSize, b.OrganizationSize) && same(a.Timezone, b.Timezone)
}

// The use case's refusals, as the contract declares them.
func TestUpdateWorkspaceRefusals(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"invalid values", shared.Invalid(shared.FieldError{Field: "name", Code: shared.FieldTooShort, Message: "must not be empty"}),
			http.StatusUnprocessableEntity, `{"status":422,"code":"validation_failed","title":"Unprocessable Entity",` +
				`"detail":"The request has invalid values.","errors":[{"field":"name","code":"too_short","message":"must not be empty"}]}`},
		{"not visible", domain.ErrNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`},
		{"a member", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
	}
	for _, tt := range tests {
		h := newServer(t, fakes{update: &fakeUpdate{err: tt.err}})
		res, body := do(t, h, request(http.MethodPatch, "/api/v0/workspaces/acme", "alice", `{"name":""}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: PATCH = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// The slug never changes: a body that names it is refused before the use
// case, like any field the contract does not have.
func TestUpdateWorkspaceRefusesTheSlug(t *testing.T) {
	update := &fakeUpdate{answer: acme}
	h := newServer(t, fakes{update: update})
	res, body := do(t, h, request(http.MethodPatch, "/api/v0/workspaces/acme", "alice", `{"name":"Acme","slug":"acme-2"}`))
	if res.StatusCode != http.StatusBadRequest || len(update.calls) != 0 {
		t.Errorf("PATCH with a slug = %d %s, calls %q; want 400 and no call", res.StatusCode, body, update.calls)
	}
}

// The availability, and the reason when the slug is not available; the slug
````

`server/internal/modules/workspace/module.go`（修改，2 处）：

````old server/internal/modules/workspace/module.go
// workspaces and their members. It brings creating, listing and reading
// workspaces, and checking a slug, and offers the other modules its reads
// through ports.
````

````new server/internal/modules/workspace/module.go
// workspaces and their members. It brings creating, listing, reading and
// changing workspaces, and checking a slug, and offers the other modules its
// reads through ports.
````

````old server/internal/modules/workspace/module.go
		GetWorkspace: app.NewGetWorkspace(store, d.Authorizer),
		CheckSlug:    app.NewCheckSlug(store),
````

````new server/internal/modules/workspace/module.go
		GetWorkspace:    app.NewGetWorkspace(store, d.Authorizer),
		UpdateWorkspace: app.NewUpdateWorkspace(store, d.Authorizer, d.Tx, d.Clock),
		CheckSlug:       app.NewCheckSlug(store),
````

- [ ] **Step 7: 矩阵和整程序测试**

`server/internal/bootstrap/permission_matrix_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_test.go
	cellOK      = cell{status: http.StatusOK}
	cellCreated = cell{status: http.StatusCreated}
````

````new server/internal/bootstrap/permission_matrix_test.go
	cellOK        = cell{status: http.StatusOK}
	cellCreated   = cell{status: http.StatusCreated}
	cellForbidden = cell{http.StatusForbidden, "forbidden"}
````

`server/internal/bootstrap/permission_matrix_workspace_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_workspace_test.go
			check: readsItsRole},
````

````new server/internal/bootstrap/permission_matrix_workspace_test.go
			check: readsItsRole},
		{op: "updateWorkspace", write: true, request: toWorkspace(http.MethodPatch, "", `{"name":"Renamed"}`),
			cells: inWorkspace(cellOK, cellForbidden, cellForbidden), check: renamesIt},
	}
}

// renamesIt: the admin's answer is acme renamed, with the admin's role.
func renamesIt(t *testing.T, c caller, answer string) {
	var w struct {
		Slug string `json:"slug"`
		Name string `json:"name"`
		Role int    `json:"role"`
	}
	decodeAnswer(t, answer, &w)
	if w.Slug != "acme" || w.Name != "Renamed" || w.Role != 20 {
		t.Errorf("%s's update answers %+v, want acme named Renamed, role 20", c, w)
````

`server/internal/bootstrap/workspace_test.go`（修改，2 处）：

````old server/internal/bootstrap/workspace_test.go
	type answer struct {
		res  *http.Response
		body []byte
		err  error
	}
	answered := make(chan answer, 1)
	go func() {
		res, err := client.Do(req)
		if err != nil {
			answered <- answer{err: err}
			return
		}
		body, err := io.ReadAll(res.Body)
		_ = res.Body.Close()
		res.Body = io.NopCloser(bytes.NewReader(body))
		answered <- answer{res, body, err}
	}()
````

````new server/internal/bootstrap/workspace_test.go
	answered := sendInBackground(req)
````

````old server/internal/bootstrap/workspace_test.go
			a.res.StatusCode, a.res.Header.Get("WWW-Authenticate"), a.body, workspaces)
	}
}

````

````new server/internal/bootstrap/workspace_test.go
			a.res.StatusCode, a.res.Header.Get("WWW-Authenticate"), a.body, workspaces)
	}
}

// answer is what a request sent in the background got.
type answer struct {
	res  *http.Response
	body []byte
	err  error
}

// sendInBackground sends req with the tests' client and hands over its
// answer, the body read and put back.
func sendInBackground(req *http.Request) <-chan answer {
	answered := make(chan answer, 1)
	go func() {
		res, err := client.Do(req)
		if err != nil {
			answered <- answer{err: err}
			return
		}
		body, err := io.ReadAll(res.Body)
		_ = res.Body.Close()
		res.Body = io.NopCloser(bytes.NewReader(body))
		answered <- answer{res, body, err}
	}()
	return answered
}

// The composed updateWorkspace decides after it has locked the workspace
// row (M3 design 3.6 convention 2). A transaction holds the row FOR NO KEY
// UPDATE and demotes alice, acme's admin, to member; her PATCH, authenticated,
// waits on the row. Once the demotion commits she is refused forbidden and
// the name stays: a decision taken before the lock would have read her
// uncommitted role, admin, and renamed the workspace.
func TestAnAdminDemotedMeanwhileCannotUpdateTheWorkspace(t *testing.T) {
	contract := apitest.Load(t)
	base, pool := sessionApp(t)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice, `{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("create = %d %s", status, body)
	}

	demotion, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = demotion.Rollback(context.Background()) }()
	for _, sql := range []string{
		"SELECT id FROM workspaces WHERE slug = 'acme' FOR NO KEY UPDATE",
		"UPDATE workspace_members SET role = 15 WHERE member_id = (SELECT id FROM users WHERE email = 'alice@example.com')",
	} {
		if _, err := demotion.Exec(context.Background(), sql); err != nil {
			t.Fatal(err)
		}
	}
	req := newRequest(t, http.MethodPatch, base+"/api/v0/workspaces/acme", alice, []byte(`{"name":"Renamed"}`))
	contract.CheckRequest(t, req)
	answered := sendInBackground(req)
	// The app runs River's jobs on the same database: only a wait for a row
	// of workspaces counts.
	pgtest.WaitForLockWaitOn(t, pool, "workspaces", 5*time.Second)
	if err := demotion.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}

	a := receiveWithin(t, answered, 10*time.Second, "answer to the update")
	if a.err != nil {
		t.Fatal(a.err)
	}
	contract.CheckResponse(t, req, a.res)
	var name string
	if err := pool.QueryRow(context.Background(), "SELECT name FROM workspaces WHERE slug = 'acme'").Scan(&name); err != nil {
		t.Fatal(err)
	}
	if a.res.StatusCode != http.StatusForbidden || problemCode(t, a.body) != "forbidden" || name != "Acme" {
		t.Errorf("update = %d %s, name %q; want 403 forbidden and the name unchanged", a.res.StatusCode, a.body, name)
	}
}

````

- [ ] **Step 8: 前端的文案**

`web/apps/web/helpers/authentication.helper.ts`（修改，1 处）：

````old web/apps/web/helpers/authentication.helper.ts
  server_busy: "auth.errors.server_busy",
````

````new web/apps/web/helpers/authentication.helper.ts
  server_busy: "auth.errors.server_busy",
  forbidden: "auth.errors.forbidden",
````

`web/packages/i18n/src/locales/en/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/auth.json
      "workspace_slug_taken": "A workspace with this URL already exists.",
````

````new web/packages/i18n/src/locales/en/auth.json
      "workspace_slug_taken": "A workspace with this URL already exists.",
      "forbidden": "Your role does not allow this.",
````

`web/packages/i18n/src/locales/zh-CN/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/auth.json
      "workspace_slug_taken": "已有工作区使用这个地址。",
````

````new web/packages/i18n/src/locales/zh-CN/auth.json
      "workspace_slug_taken": "已有工作区使用这个地址。",
      "forbidden": "你的角色不能做这件事。",
````

- [ ] **Step 9: 测试、检查**

Run: `go -C server test -count=1 ./internal/modules/workspace/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestAnAdminDemotedMeanwhileCannotUpdateTheWorkspace|TestPermissionMatrix|TestThePermissionMatrixCoversEveryOperation|TestEveryActionHasARuleAndEveryRuleAnAction' ./internal/bootstrap/`
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
Expected: 通过（文案表的键等于契约的码，含 `forbidden`）。

- [ ] **Step 10: 提交**

```bash
git add api/modules/workspace.yaml server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/permission_matrix_workspace_test.go server/internal/bootstrap/workspace_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/workspace/adapter/http/handler.go server/internal/modules/workspace/adapter/http/handler_test.go server/internal/modules/workspace/adapter/http/workspaces.go server/internal/modules/workspace/adapter/http/workspaces_test.go server/internal/modules/workspace/adapter/postgres/failures_test.go server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql server/internal/modules/workspace/adapter/postgres/store_test.go server/internal/modules/workspace/adapter/postgres/update_workspace_test.go server/internal/modules/workspace/adapter/postgres/workspaces.go server/internal/modules/workspace/app/fakes_test.go server/internal/modules/workspace/app/lock.go server/internal/modules/workspace/app/ports.go server/internal/modules/workspace/app/update_workspace.go server/internal/modules/workspace/app/update_workspace_test.go server/internal/modules/workspace/domain/actions.go server/internal/modules/workspace/domain/workspace.go server/internal/modules/workspace/domain/workspace_test.go server/internal/modules/workspace/module.go web/apps/web/helpers/authentication.helper.ts web/packages/i18n/src/locales/en/auth.json web/packages/i18n/src/locales/zh-CN/auth.json api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go server/internal/modules/workspace/adapter/http/gen/server.gen.go server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P2): updateWorkspace locks the workspace row, then decides

PATCH /api/v0/workspaces/{slug} changes the name, organization size or
time zone for the workspace's admins. The values are checked first; then,
in one transaction, the workspace row FOR NO KEY UPDATE, the decision, the
write (M3 design 3.6 convention 2): an admin demoted meanwhile is refused.
The web app words forbidden.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**：

| 改坏 | 必须失败的测试 |
|---|---|
| 先读工作区、`Authorize`，再锁（先判定） | `TestUpdateWorkspaceLocksThenDecidesThenWrites`、`TestAnAdminDemotedMeanwhileCannotUpdateTheWorkspace`（真实数据库：200 而不是 403） |
| 锁和判定挪到事务之前 | `TestUpdateWorkspaceLocksThenDecidesThenWrites`（假实现记下 `outside tx`） |
| 用 `ShareWorkspaceBySlug`（`FOR SHARE`）锁 | `TestUpdateWorkspaceLocksThenDecidesThenWrites`（锁的种类在存储一侧由 Task 4 的 `TestTheWorkspaceLocksConflictAsConvention2Says` 核对） |
| `workspace.update` 也给成员 | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix/updateWorkspace/member` |
| 存储的 `UpdateWorkspace` 吞掉错误 | `TestAFailedWriteIsAnError`、`TestUpdateWorkspaceWithoutARowIsAnError` |

**Done when:** `updateWorkspace` 的用例、存储、HTTP、矩阵（6 格）和整程序测试通过；`make test-web` 通过。

---

### Task 7: 显示设置的领域、端口和存储

**Files:**
- Create: `server/internal/modules/workspace/adapter/postgres/preferences.go`、`server/internal/modules/workspace/adapter/postgres/preferences_test.go`、`server/internal/modules/workspace/adapter/postgres/queries/preferences.sql`、`server/internal/modules/workspace/domain/preferences.go`、`server/internal/modules/workspace/domain/preferences_test.go`
- Modify: `server/internal/modules/workspace/adapter/postgres/failures_test.go`、`server/internal/modules/workspace/app/ports.go`
- Generate: `server/internal/modules/workspace/adapter/postgres/gen/preferences.sql.go`

**Interfaces:**
- Produces（spec 2.9，M3 设计 3.18、4.5）：
  - `domain.Preferences{NavigationControl string; NavigationProjectLimit int}`、`domain.PreferencesPatch`（两个指针）、`domain.DefaultPreferences()`（`ACCORDION`、10，等于列的默认值）、`(Preferences).Apply(patch)`、`domain.CheckPreferencesPatch(p)`（方式是两个值之一，上限 0 到 2147483647，一次 422）；
  - `app.WorkspaceSharer{ShareWorkspaceBySlug}`、`app.PreferencesRow{ID, WorkspaceID, UserID, Patch, Now}`、`app.PreferencesReader{WorkspaceFinder; Preferences(ctx, workspaceID, userID) (p, found, err)}`、`app.PreferencesWriter{WorkspaceSharer; UpsertPreferences(ctx, r) (domain.Preferences, error)}`；
  - 查询 `Preferences`、`UpsertPreferences`（`INSERT … ON CONFLICT (workspace_id, user_id) WHERE deleted_at IS NULL DO UPDATE`，冲突时只改设了的字段和 `updated_at`、`updated_by_id`）；`(*Store).Preferences`（没有行是 `found=false`，失败是错误）、`(*Store).UpsertPreferences`（插入的值是默认值加上补丁）。
- 使用者：Task 8 的用例；Task 9 的连带；矩阵的准备数据。

**Tests:**
- `domain/preferences_test.go`：`TestPreferencesApply`（设了的字段改变，别的不变；`DefaultPreferences()` 是 `ACCORDION`、10）；`TestCheckPreferencesPatch`（合法的五种，含空补丁、0 和 2147483647；小写的方式、Plane 的 `SIDEBAR`、空串、-1、2147483648、两个都错）。
- `adapter/postgres/preferences_test.go`（真实数据库）：
  - `TestDefaultPreferencesAreTheColumnDefaults`：不带两列插入的一行读出来等于 `DefaultPreferences()`；
  - `TestUpsertPreferences`：第一次修改建行（默认值加补丁，创建者、时间）；之后只改设了的字段、`updated_at`、`updated_by_id`，行的 id 不变；空补丁不改值；每个账户、每个工作区各自一行；
  - `TestUpsertPreferencesAfterTheRowIsDeleted`：已删除的行不算冲突，下一次修改从默认值建新行，读到的是它；
  - `TestPreferencesReadsTheAccountsOwnRow`：第一次修改之前没有；别的账户的行、别的工作区的行都读不到；
  - `TestConcurrentFirstChangesLeaveOneRow`：两个事务同时第一次修改：第二个等第一个的索引项（`WaitForLockWait`），第一个提交后它改那一行；只有一行，两次设的字段都在。
- `failures_test.go`：读的测试加 `Preferences`（alice 先有设置，所以"没有行"不是正确答案）；写的测试加 `UpsertPreferences`。

- [ ] **Step 1: 领域**

`server/internal/modules/workspace/domain/preferences.go`（新文件，61 行）：

````file server/internal/modules/workspace/domain/preferences.go
package domain

import (
	"fmt"
	"math"
	"slices"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Preferences are an account's display settings in a workspace (M3 design
// 3.18, 5.2): how the sidebar shows the projects, and how many before
// "more".
type Preferences struct {
	NavigationControl      string // one of navigationControls
	NavigationProjectLimit int    // 0 shows every project
}

// PreferencesPatch is a partial update of Preferences: a nil field stays as
// it is.
type PreferencesPatch struct {
	NavigationControl      *string
	NavigationProjectLimit *int
}

// DefaultPreferences are an account's settings in a workspace until it first
// changes them: the columns' defaults (M3 design 4.5), which the store's
// test holds equal.
func DefaultPreferences() Preferences {
	return Preferences{NavigationControl: "ACCORDION", NavigationProjectLimit: 10}
}

// navigationControls are the web app's two modes (TProjectNavigationMode),
// which the column's CHECK holds too.
var navigationControls = []string{"ACCORDION", "TABBED"}

// Apply returns p with the fields patch sets changed.
func (p Preferences) Apply(patch PreferencesPatch) Preferences {
	if patch.NavigationControl != nil {
		p.NavigationControl = *patch.NavigationControl
	}
	if patch.NavigationProjectLimit != nil {
		p.NavigationProjectLimit = *patch.NavigationProjectLimit
	}
	return p
}

// CheckPreferencesPatch checks the fields p sets: a mode of the two, and a
// limit from 0 to the column's integer maximum. Every problem is reported at
// once, as one 422 validation_failed.
func CheckPreferencesPatch(p PreferencesPatch) error {
	var control, limit *shared.FieldError
	if p.NavigationControl != nil && !slices.Contains(navigationControls, *p.NavigationControl) {
		control = &shared.FieldError{Field: "navigation_control_preference", Code: shared.FieldInvalidFormat, Message: "is not ACCORDION or TABBED"}
	}
	if p.NavigationProjectLimit != nil && (*p.NavigationProjectLimit < 0 || *p.NavigationProjectLimit > math.MaxInt32) {
		limit = &shared.FieldError{Field: "navigation_project_limit", Code: shared.FieldOutOfRange,
			Message: fmt.Sprintf("must be between 0 and %d", math.MaxInt32)}
	}
	return invalid(control, limit)
}
````

`server/internal/modules/workspace/domain/preferences_test.go`（新文件，66 行）：

````file server/internal/modules/workspace/domain/preferences_test.go
package domain

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The fields a patch sets change, the others stay.
func TestPreferencesApply(t *testing.T) {
	base := Preferences{NavigationControl: "TABBED", NavigationProjectLimit: 3}
	tests := []struct {
		p    PreferencesPatch
		want Preferences
	}{
		{PreferencesPatch{}, base},
		{PreferencesPatch{NavigationControl: ptr("ACCORDION")}, Preferences{"ACCORDION", 3}},
		{PreferencesPatch{NavigationProjectLimit: ptr(0)}, Preferences{"TABBED", 0}},
		{PreferencesPatch{NavigationControl: ptr("ACCORDION"), NavigationProjectLimit: ptr(20)}, Preferences{"ACCORDION", 20}},
	}
	for _, tt := range tests {
		if got := base.Apply(tt.p); got != tt.want {
			t.Errorf("Apply(%+v) = %+v, want %+v", tt.p, got, tt.want)
		}
	}
	if DefaultPreferences() != (Preferences{"ACCORDION", 10}) {
		t.Errorf("DefaultPreferences() = %+v, want ACCORDION, 10", DefaultPreferences())
	}
}

// A patch has a mode of the two and a limit from 0 to 2147483647, the
// column's range; each field is checked only when set.
func TestCheckPreferencesPatch(t *testing.T) {
	for _, p := range []PreferencesPatch{
		{}, {NavigationControl: ptr("ACCORDION")}, {NavigationControl: ptr("TABBED")},
		{NavigationProjectLimit: ptr(0)}, {NavigationProjectLimit: ptr(math.MaxInt32)},
	} {
		if err := CheckPreferencesPatch(p); err != nil {
			t.Errorf("CheckPreferencesPatch(%+v) = %v, want nil", p, err)
		}
	}
	mode := shared.FieldError{Field: "navigation_control_preference", Code: "invalid_format", Message: "is not ACCORDION or TABBED"}
	limit := shared.FieldError{Field: "navigation_project_limit", Code: "out_of_range", Message: "must be between 0 and 2147483647"}
	tests := []struct {
		name string
		p    PreferencesPatch
		want []shared.FieldError
	}{
		{"a lower-case mode", PreferencesPatch{NavigationControl: ptr("tabbed")}, []shared.FieldError{mode}},
		{"a mode of Plane's sidebar", PreferencesPatch{NavigationControl: ptr("SIDEBAR")}, []shared.FieldError{mode}},
		{"no mode", PreferencesPatch{NavigationControl: ptr("")}, []shared.FieldError{mode}},
		{"a negative limit", PreferencesPatch{NavigationProjectLimit: ptr(-1)}, []shared.FieldError{limit}},
		{"a limit past the column", PreferencesPatch{NavigationProjectLimit: ptr(math.MaxInt32 + 1)}, []shared.FieldError{limit}},
		{"both", PreferencesPatch{NavigationControl: ptr("TABS"), NavigationProjectLimit: ptr(-5)}, []shared.FieldError{mode, limit}},
	}
	for _, tt := range tests {
		err := CheckPreferencesPatch(tt.p)
		var se *shared.Error
		if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || !slices.Equal(se.Fields, tt.want) {
			t.Errorf("%s: CheckPreferencesPatch(%+v) = %#v, want validation_failed with %v", tt.name, tt.p, err, tt.want)
		}
	}
}
````

- [ ] **Step 2: 端口和查询**

`server/internal/modules/workspace/app/ports.go`（修改，2 处）：

````old server/internal/modules/workspace/app/ports.go
// WorkspaceLocker takes the parent locks of the writes on a workspace (M3
// design 3.6 convention 2), in the transaction ctx carries: each locks the
// undeleted workspace with slug until the transaction ends and returns its
// id; ErrNotFound when there is none, also when it was deleted while the
// lock waited.
type WorkspaceLocker interface {
	// LockWorkspaceBySlug locks FOR NO KEY UPDATE: for a write of the
	// workspace row itself or of a membership.
	LockWorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, error)
````

````new server/internal/modules/workspace/app/ports.go
// The parent locks of the writes on a workspace (M3 design 3.6 convention
// 2), in the transaction ctx carries: each locks the undeleted workspace
// with slug until the transaction ends and returns its id; ErrNotFound when
// there is none, also when it was deleted while the lock waited.

// WorkspaceLocker locks FOR NO KEY UPDATE: for a write of the workspace row
// itself or of a membership.
type WorkspaceLocker interface {
	LockWorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, error)
}

// WorkspaceSharer locks FOR SHARE: for a write that adds or changes a row
// under the workspace.
type WorkspaceSharer interface {
	ShareWorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, error)
````

````old server/internal/modules/workspace/app/ports.go
}

// SlugChecker tells whether a slug is taken.
````

````new server/internal/modules/workspace/app/ports.go
}

// PreferencesRow is a change of an account's display settings in a
// workspace, and the id of the row if the change inserts one.
type PreferencesRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	Patch       domain.PreferencesPatch
	Now         time.Time
}

// PreferencesReader reads an account's display settings in a workspace.
type PreferencesReader interface {
	WorkspaceFinder
	// Preferences returns userID's settings in workspaceID; found is false
	// while there is no undeleted row.
	Preferences(ctx context.Context, workspaceID, userID uuid.UUID) (p domain.Preferences, found bool, err error)
}

// PreferencesWriter writes an account's display settings in a workspace
// under the workspace's lock.
type PreferencesWriter interface {
	WorkspaceSharer
	// UpsertPreferences applies r.Patch to the account's undeleted row, or
	// inserts one with domain.DefaultPreferences and the patch applied, and
	// returns the settings as stored.
	UpsertPreferences(ctx context.Context, r PreferencesRow) (domain.Preferences, error)
}

// SlugChecker tells whether a slug is taken.
````

`server/internal/modules/workspace/adapter/postgres/queries/preferences.sql`（新文件，25 行）：

````file server/internal/modules/workspace/adapter/postgres/queries/preferences.sql
-- name: Preferences :one
-- getWorkspacePreferences: the account's undeleted row, if any (M3 design 3.18).
SELECT navigation_control_preference, navigation_project_limit
FROM workspace_user_properties
WHERE workspace_id = sqlc.arg(workspace_id) AND user_id = sqlc.arg(user_id) AND deleted_at IS NULL;

-- name: UpsertPreferences :one
-- updateWorkspacePreferences, under the workspace's FOR SHARE (M3 design 3.18): the first change inserts the row with
-- the values given, the defaults with the change applied; later ones change only the fields that are set. The
-- conflict target is the partial unique index, so a deleted row does not count.
INSERT INTO workspace_user_properties AS p (id, workspace_id, user_id, navigation_control_preference,
                                             navigation_project_limit, created_by_id, updated_by_id, created_at,
                                             updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(user_id), sqlc.arg(navigation_control_preference),
        sqlc.arg(navigation_project_limit), sqlc.arg(user_id), sqlc.arg(user_id), sqlc.arg(now), sqlc.arg(now))
ON CONFLICT (workspace_id, user_id) WHERE deleted_at IS NULL DO UPDATE
SET navigation_control_preference = CASE WHEN sqlc.arg(set_navigation_control)::boolean
                                         THEN EXCLUDED.navigation_control_preference
                                         ELSE p.navigation_control_preference END,
    navigation_project_limit      = CASE WHEN sqlc.arg(set_navigation_project_limit)::boolean
                                         THEN EXCLUDED.navigation_project_limit
                                         ELSE p.navigation_project_limit END,
    updated_by_id                 = EXCLUDED.updated_by_id,
    updated_at                    = EXCLUDED.updated_at
RETURNING navigation_control_preference, navigation_project_limit;
````

- [ ] **Step 3: 生成**

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `95cd80b75695227e679b7c19aa5f4bdae279b2a9ffe0b5f5bb68e5c9b235083b` | 90 | `server/internal/modules/workspace/adapter/postgres/gen/preferences.sql.go` |

Run: `shasum -a 256 server/internal/modules/workspace/adapter/postgres/gen/preferences.sql.go`
Expected: 与上表相同。

- [ ] **Step 4: 存储和测试**

`server/internal/modules/workspace/adapter/postgres/preferences.go`（新文件，45 行）：

````file server/internal/modules/workspace/adapter/postgres/preferences.go
package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// Preferences returns userID's settings in workspaceID; found is false while
// there is no undeleted row.
func (s *Store) Preferences(ctx context.Context, workspaceID, userID uuid.UUID) (domain.Preferences, bool, error) {
	r, err := s.queries(ctx).Preferences(ctx, gen.PreferencesParams{WorkspaceID: workspaceID, UserID: userID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Preferences{}, false, nil
	case err != nil:
		return domain.Preferences{}, false, fmt.Errorf("read workspace preferences: %w", err)
	}
	return domain.Preferences{NavigationControl: r.NavigationControlPreference, NavigationProjectLimit: int(r.NavigationProjectLimit)}, true, nil
}

// UpsertPreferences applies r.Patch to the account's undeleted row, or
// inserts one with domain.DefaultPreferences and the patch applied, by the
// account at r.Now, and returns the settings as stored. The domain checked
// the values, so the limit fits the column.
func (s *Store) UpsertPreferences(ctx context.Context, r app.PreferencesRow) (domain.Preferences, error) {
	inserted := domain.DefaultPreferences().Apply(r.Patch)
	row, err := s.queries(ctx).UpsertPreferences(ctx, gen.UpsertPreferencesParams{
		ID: r.ID, WorkspaceID: r.WorkspaceID, UserID: r.UserID,
		NavigationControlPreference: inserted.NavigationControl, NavigationProjectLimit: int32(inserted.NavigationProjectLimit),
		SetNavigationControl: r.Patch.NavigationControl != nil, SetNavigationProjectLimit: r.Patch.NavigationProjectLimit != nil,
		Now: r.Now,
	})
	if err != nil {
		return domain.Preferences{}, fmt.Errorf("write workspace preferences: %w", err)
	}
	return domain.Preferences{NavigationControl: row.NavigationControlPreference, NavigationProjectLimit: int(row.NavigationProjectLimit)}, nil
}
````

`server/internal/modules/workspace/adapter/postgres/preferences_test.go`（新文件，215 行）：

````file server/internal/modules/workspace/adapter/postgres/preferences_test.go
package postgresadapter_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// prefsRow is a stored row of workspace_user_properties, as the tests read it.
type prefsRow struct {
	id                   uuid.UUID
	mode                 string
	limit                int
	createdBy, updatedBy uuid.UUID
	createdAt, updatedAt time.Time
}

// undeletedPrefs reads the undeleted rows of user in workspace.
func undeletedPrefs(t *testing.T, pool *pgxpool.Pool, workspace, user uuid.UUID) []prefsRow {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT id, navigation_control_preference, navigation_project_limit, created_by_id, updated_by_id, created_at, updated_at
		FROM workspace_user_properties WHERE workspace_id = $1 AND user_id = $2 AND deleted_at IS NULL`, workspace, user)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []prefsRow
	for rows.Next() {
		var r prefsRow
		if err := rows.Scan(&r.id, &r.mode, &r.limit, &r.createdBy, &r.updatedBy, &r.createdAt, &r.updatedAt); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func upsert(t *testing.T, s *postgresadapter.Store, r app.PreferencesRow) domain.Preferences {
	t.Helper()
	p, err := s.UpsertPreferences(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// domain.DefaultPreferences are the columns' defaults: a row inserted
// without the two columns reads as them.
func TestDefaultPreferencesAreTheColumnDefaults(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	exec(t, pool, "INSERT INTO workspace_user_properties (id, workspace_id, user_id) VALUES ($1, $2, $3)", uuid.NewV7(), acme.ID, alice)
	if p, found, err := s.Preferences(context.Background(), acme.ID, alice); err != nil || !found || p != domain.DefaultPreferences() {
		t.Errorf("Preferences() = %+v, %v, %v; want the defaults %+v", p, found, err, domain.DefaultPreferences())
	}
}

// The first change inserts the account's row: the defaults with the change
// applied, by the account, at the clock's time. Later ones change the fields
// they set on the same row, and only its updated_at and updated_by_id
// besides; an empty change leaves the values. Each account and workspace
// has its own row.
func TestUpsertPreferences(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	first := uuid.NewV7()
	later, latest := now.Add(time.Hour), now.Add(2*time.Hour)

	steps := []struct {
		r    app.PreferencesRow
		want domain.Preferences
		at   time.Time
	}{
		{app.PreferencesRow{ID: first, WorkspaceID: acme.ID, UserID: alice, Patch: domain.PreferencesPatch{NavigationProjectLimit: ptr(3)}, Now: now},
			prefs("ACCORDION", 3), now},
		{app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: alice, Patch: domain.PreferencesPatch{NavigationControl: ptr("TABBED")}, Now: later},
			prefs("TABBED", 3), later},
		{app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: alice, Now: latest}, prefs("TABBED", 3), latest},
	}
	for i, step := range steps {
		if got := upsert(t, s, step.r); got != step.want {
			t.Errorf("step %d: UpsertPreferences() = %+v, want %+v", i, got, step.want)
		}
		rows := undeletedPrefs(t, pool, acme.ID, alice)
		want := prefsRow{first, step.want.NavigationControl, step.want.NavigationProjectLimit, alice, alice, now, step.at}
		if len(rows) != 1 || rows[0].id != want.id || rows[0].mode != want.mode || rows[0].limit != want.limit || rows[0].createdBy != alice ||
			rows[0].updatedBy != alice || !rows[0].createdAt.Equal(now) || !rows[0].updatedAt.Equal(step.at) {
			t.Errorf("step %d: rows %+v, want one: %+v", i, rows, want)
		}
	}

	if got := upsert(t, s, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: bob,
		Patch: domain.PreferencesPatch{NavigationProjectLimit: ptr(0)}, Now: now}); got != prefs("ACCORDION", 0) {
		t.Errorf("bob's first change = %+v, want ACCORDION, 0", got)
	}
	if got := upsert(t, s, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: beta.ID, UserID: alice, Now: now}); got != domain.DefaultPreferences() {
		t.Errorf("alice's first change in beta = %+v, want the defaults", got)
	}
	if p, _, err := s.Preferences(context.Background(), acme.ID, alice); err != nil || p != prefs("TABBED", 3) {
		t.Errorf("alice's settings in acme after bob's and beta's = %+v, %v; want TABBED, 3", p, err)
	}
}

// A deleted row is no conflict (the partial unique index): the next change
// inserts a new row from the defaults, and it is the one read.
func TestUpsertPreferencesAfterTheRowIsDeleted(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	upsert(t, s, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: alice,
		Patch: domain.PreferencesPatch{NavigationControl: ptr("TABBED"), NavigationProjectLimit: ptr(3)}, Now: now})
	exec(t, pool, "UPDATE workspace_user_properties SET deleted_at = $1", now)
	if _, found, err := s.Preferences(context.Background(), acme.ID, alice); err != nil || found {
		t.Fatalf("Preferences() of a deleted row: found %v, %v; want none", found, err)
	}

	second := uuid.NewV7()
	if got := upsert(t, s, app.PreferencesRow{ID: second, WorkspaceID: acme.ID, UserID: alice,
		Patch: domain.PreferencesPatch{NavigationProjectLimit: ptr(7)}, Now: now}); got != prefs("ACCORDION", 7) {
		t.Errorf("UpsertPreferences() after the deletion = %+v, want ACCORDION, 7", got)
	}
	if rows := undeletedPrefs(t, pool, acme.ID, alice); len(rows) != 1 || rows[0].id != second {
		t.Errorf("undeleted rows %+v, want the new one, %s", rows, second)
	}
	if p, found, err := s.Preferences(context.Background(), acme.ID, alice); err != nil || !found || p != prefs("ACCORDION", 7) {
		t.Errorf("Preferences() = %+v, %v, %v; want ACCORDION, 7", p, found, err)
	}
}

// Preferences reads the account's own undeleted row in that workspace:
// none before the first change, none for another account's or another
// workspace's row.
func TestPreferencesReadsTheAccountsOwnRow(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	if _, found, err := s.Preferences(context.Background(), acme.ID, alice); err != nil || found {
		t.Errorf("before any change: found %v, %v; want none", found, err)
	}
	upsert(t, s, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: bob, Patch: domain.PreferencesPatch{NavigationProjectLimit: ptr(1)}, Now: now})
	upsert(t, s, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: beta.ID, UserID: alice, Patch: domain.PreferencesPatch{NavigationProjectLimit: ptr(2)}, Now: now})
	if _, found, err := s.Preferences(context.Background(), acme.ID, alice); err != nil || found {
		t.Errorf("with bob's row in acme and alice's in beta: found %v, %v; want none for alice in acme", found, err)
	}
	for _, tt := range []struct {
		workspace, user uuid.UUID
		limit           int
	}{{acme.ID, bob, 1}, {beta.ID, alice, 2}} {
		if p, found, err := s.Preferences(context.Background(), tt.workspace, tt.user); err != nil || !found || p.NavigationProjectLimit != tt.limit {
			t.Errorf("Preferences(%s, %s) = %+v, %v, %v; want the limit %d", tt.workspace, tt.user, p, found, err, tt.limit)
		}
	}
}

// Two first changes at once leave one row (M3 design 3.18): the second
// insert waits on the first's index entry, and once that commits it changes
// the row instead, the fields it sets over the first's.
func TestConcurrentFirstChangesLeaveOneRow(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	tx := postgres.NewTxManager(pool, 5*time.Second)
	commit := hold(t, tx, func(ctx context.Context) error {
		_, err := s.UpsertPreferences(ctx, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: alice,
			Patch: domain.PreferencesPatch{NavigationProjectLimit: ptr(3)}, Now: now})
		return err
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	type answer struct {
		p   domain.Preferences
		err error
	}
	answered := make(chan answer, 1)
	go func() {
		p, err := s.UpsertPreferences(ctx, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: alice,
			Patch: domain.PreferencesPatch{NavigationControl: ptr("TABBED")}, Now: now})
		answered <- answer{p, err}
	}()
	pgtest.WaitForLockWait(t, pool, 5*time.Second)
	if err := commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case a := <-answered:
		if a.err != nil || a.p != prefs("TABBED", 3) {
			t.Errorf("the second change = %+v, %v; want TABBED, 3", a.p, a.err)
		}
	case <-ctx.Done():
		t.Fatal("the second change did not end within 10s")
	}
	if rows := undeletedPrefs(t, pool, acme.ID, alice); len(rows) != 1 {
		t.Errorf("rows %+v, want one", rows)
	}
}

// prefs is the settings mode and limit.
func prefs(mode string, limit int) domain.Preferences {
	return domain.Preferences{NavigationControl: mode, NavigationProjectLimit: limit}
}
````

`server/internal/modules/workspace/adapter/postgres/failures_test.go`（修改，4 处）：

````old server/internal/modules/workspace/adapter/postgres/failures_test.go
	"testing"
````

````new server/internal/modules/workspace/adapter/postgres/failures_test.go
	"testing"
	"uuid"
````

````old server/internal/modules/workspace/adapter/postgres/failures_test.go
// against a workspace alice administers, so that the right answer is none
// of the zero values.
func TestAFailedReadIsAnErrorNotAnAnswer(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	w := newWorkspace(t, s, "Acme", "acme", alice)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
````

````new server/internal/modules/workspace/adapter/postgres/failures_test.go
// against a workspace alice administers and has display settings in, so
// that the right answer is none of the zero values.
func TestAFailedReadIsAnErrorNotAnAnswer(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	w := newWorkspace(t, s, "Acme", "acme", alice)
	tabbed := "TABBED"
	upsert(t, s, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: w.ID, UserID: alice,
		Patch: domain.PreferencesPatch{NavigationControl: &tabbed}, Now: now})
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
````

````old server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("WorkspaceBySlug() = %+v, %v; want context.Canceled, not app.ErrNotFound", got, err)
````

````new server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("WorkspaceBySlug() = %+v, %v; want context.Canceled, not app.ErrNotFound", got, err)
	}
	if p, found, err := s.Preferences(cancelled, w.ID, alice); !failed(err) || found || p != (domain.Preferences{}) {
		t.Errorf("Preferences() = %+v, %v, %v; want context.Canceled, not no row", p, found, err)
````

````old server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("UpdateWorkspace() = %+v, %v; want context.Canceled", got, err)
	}
````

````new server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("UpdateWorkspace() = %+v, %v; want context.Canceled", got, err)
	}
	if got, err := s.UpsertPreferences(cancelled, app.PreferencesRow{ID: uuid.NewV7(), WorkspaceID: w.ID, UserID: alice, Now: now}); !failed(err) ||
		got != (domain.Preferences{}) {
		t.Errorf("UpsertPreferences() = %+v, %v; want context.Canceled", got, err)
	}
````

- [ ] **Step 5: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/domain/ ./internal/modules/workspace/adapter/postgres/`
Expected: 两个 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 6: 提交**

```bash
git add server/internal/modules/workspace/adapter/postgres/failures_test.go server/internal/modules/workspace/adapter/postgres/preferences.go server/internal/modules/workspace/adapter/postgres/preferences_test.go server/internal/modules/workspace/adapter/postgres/queries/preferences.sql server/internal/modules/workspace/app/ports.go server/internal/modules/workspace/domain/preferences.go server/internal/modules/workspace/domain/preferences_test.go server/internal/modules/workspace/adapter/postgres/gen/preferences.sql.go
```
```bash
git commit -m "feat(M3/P2): the display settings' domain and store

An account's project navigation settings in a workspace: the defaults are
the columns', the first change inserts the row through the partial unique
key (INSERT ... ON CONFLICT), later ones change the fields they set; two
first changes at once leave one row (M3 design 3.18).

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**：

| 改坏 | 必须失败的测试 |
|---|---|
| 冲突的目标换成 `(id)`（等于普通的插入） | `TestUpsertPreferences`、`TestConcurrentFirstChangesLeaveOneRow` |
| 冲突时不看是否设了方式（总是覆盖） | `TestUpsertPreferences` |
| `Preferences` 不看 `user_id` | `TestPreferencesReadsTheAccountsOwnRow` |
| `Preferences` 不看 `workspace_id` | `TestPreferencesReadsTheAccountsOwnRow` |
| `Preferences` 读已删除的行 | `TestUpsertPreferencesAfterTheRowIsDeleted` |
| `DefaultPreferences` 的上限改成 5；或迁移的默认值改成 5 | `TestDefaultPreferencesAreTheColumnDefaults` |
| 校验放行 -1 | `TestCheckPreferencesPatch` |
| `Preferences` 失败时答"没有行" | `TestAFailedReadIsAnErrorNotAnAnswer` |
| `UpsertPreferences` 吞掉错误 | `TestAFailedWriteIsAnError` |

**Done when:** 5 个存储测试在真实数据库上通过（两个账户、两个工作区、一行已删除、一次并发）；生成物的 SHA-256 与表相同。

---

### Task 8: 读、改显示设置

**Files:**
- Create: `server/internal/modules/workspace/adapter/http/preferences.go`、`server/internal/modules/workspace/adapter/http/preferences_test.go`、`server/internal/modules/workspace/app/preferences.go`、`server/internal/modules/workspace/app/preferences_test.go`
- Modify: `api/modules/workspace.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/permission_matrix_workspace_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/workspace/adapter/http/handler.go`、`server/internal/modules/workspace/adapter/http/handler_test.go`、`server/internal/modules/workspace/app/fakes_test.go`、`server/internal/modules/workspace/domain/actions.go`、`server/internal/modules/workspace/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/workspace/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.9，M3 设计 3.18、5.1）：
  - 接口描述：`getWorkspacePreferences`（`GET /api/v0/me/workspaces/{slug}/preferences`，码 `[workspace.not_found]`）、`updateWorkspacePreferences`（`PATCH`，`WorkspacePreferencesUpdate`，码 `[validation_failed, workspace.not_found]`），都答 `WorkspacePreferences{navigation_control_preference, navigation_project_limit}`；`api/openapi.yaml` 加路径；
  - `domain.ActionPreferencesRead = "workspace_preferences.read"`、`ActionPreferencesUpdate = "workspace_preferences.update"`；规则表两行：任何有效成员；
  - `app.NewGetWorkspacePreferences(reader, auth)`：不开事务：`WorkspaceBySlug` → `Authorize` → `Preferences`，没有行时答 `DefaultPreferences()`，不写库；
  - `app.NewUpdateWorkspacePreferences(writer, auth, tx, clock)`：校验（事务之前）→ 一个事务：`lockAndDecide(ShareWorkspaceBySlug, workspace_preferences.update)` → `UpsertPreferences`（每次一个新的行 id，插入时用）；
  - HTTP：两个 handler；`module.go` 接上。
- 使用者：W8（Task 14）；P9 的 `ProjectNavigationDialog`。

**Tests:**
- `app/preferences_test.go`：
  - `TestGetWorkspacePreferencesReadsTheCallersOwn`：alice 的设置；bob 没改过，两个工作区都是默认值；不开事务，判定在读之前，参数是调用者和找到的工作区；
  - `TestGetWorkspacePreferencesRefusals`：不存在、看不到都是 `workspace.not_found`，不读设置；失败原样返回，不是默认值；
  - `TestUpdateWorkspacePreferencesLocksThenDecidesThenWrites`：`ShareWorkspaceBySlug` → `Authorize` → `UpsertPreferences`，在一个事务里；alice 的修改保留她以前设的，bob 的第一次从默认值开始；两次写的行 id 不同；
  - `TestUpdateWorkspacePreferencesRefusals`：非法值（没有事务）；不存在、看不到；锁、判定、写入失败。
- `adapter/http/preferences_test.go`：`TestGetWorkspacePreferences`、`TestUpdateWorkspacePreferences`、`TestWorkspacePreferencesRefusals`（码经契约核对；上限不是整数时 400，用例不被调用）。
- 矩阵：两行，每列都是调用者自己的设置：管理员 200、成员 200、访客 200，另三列 `workspace.not_found`；`preferencesAre` 核对答案：读的一行管理员是准备好的 `TABBED`、3，别人是默认值；改的一行（上限改成 5）管理员 `TABBED`、5，别人 `ACCORDION`、5。准备数据经存储写入管理员在 `acme` 的设置。

- [ ] **Step 1: 接口描述**

`api/modules/workspace.yaml`（修改，2 处）：

````old api/modules/workspace.yaml
        default:
          $ref: '#/components/responses/Problem'
components:
````

````new api/modules/workspace.yaml
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/me/workspaces/{slug}/preferences:
    parameters:
      - $ref: '#/components/parameters/Slug'
    get:
      operationId: getWorkspacePreferences
      tags: [workspace]
      summary: Read the caller's display settings in a workspace
      description: >-
        The caller's own settings of the sidebar's project navigation in the
        workspace, for any active member. Until the caller first changes
        them they are the defaults, ACCORDION and 10, and reading them writes
        nothing. A workspace that does not exist, is deleted, or of which the
        caller is not an active member answers workspace.not_found.
      security: [{bearer: []}]
      x-problem-codes: [workspace.not_found]
      responses:
        '200':
          description: The caller's settings in the workspace.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/WorkspacePreferences'
        default:
          $ref: '#/components/responses/Problem'
    patch:
      operationId: updateWorkspacePreferences
      tags: [workspace]
      summary: Change the caller's display settings in a workspace
      description: >-
        The fields given change and the others stay; the first change stores
        the caller's settings, the defaults with the change applied. The
        values are checked before the workspace is looked at
        (validation_failed). A workspace that does not exist, is deleted, or
        of which the caller is not an active member answers
        workspace.not_found.
      security: [{bearer: []}]
      x-problem-codes: [validation_failed, workspace.not_found]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/WorkspacePreferencesUpdate'
      responses:
        '200':
          description: The caller's settings in the workspace, as changed.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/WorkspacePreferences'
        default:
          $ref: '#/components/responses/Problem'
components:
````

````old api/modules/workspace.yaml
          description: An IANA time zone name.
          type: string
    SlugAvailability:
````

````new api/modules/workspace.yaml
          description: An IANA time zone name.
          type: string
    NavigationControlPreference:
      description: How the sidebar shows the projects, as sections one under another or as tabs.
      type: string
      enum: [ACCORDION, TABBED]
    NavigationProjectLimit:
      description: How many projects the sidebar shows before "more"; 0 shows them all.
      type: integer
      minimum: 0
      maximum: 2147483647
    WorkspacePreferences:
      type: object
      additionalProperties: false
      required: [navigation_control_preference, navigation_project_limit]
      properties:
        navigation_control_preference:
          $ref: '#/components/schemas/NavigationControlPreference'
        navigation_project_limit:
          $ref: '#/components/schemas/NavigationProjectLimit'
    WorkspacePreferencesUpdate:
      type: object
      additionalProperties: false
      properties:
        navigation_control_preference:
          $ref: '#/components/schemas/NavigationControlPreference'
        navigation_project_limit:
          $ref: '#/components/schemas/NavigationProjectLimit'
    SlugAvailability:
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspace-slugs~1{slug}'
````

````new api/openapi.yaml
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspace-slugs~1{slug}'
  /api/v0/me/workspaces/{slug}/preferences:
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1me~1workspaces~1{slug}~1preferences'
````

- [ ] **Step 2: 生成**

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `6a66063755d659529081b488a662a6eecffc01cdefd3fde8ebbc358182d6c473` | 1201 | `api/dist/openapi.yaml` |
| `9b86a0dd01d8741090362c5c663afc1ac3b3cbe987b93f46d8f7949766c5cc16` | 31 | `server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go` |
| `1e9205c25b0dece9a141d02b6eb82aafbecb54c25eeae547a6fd1dfa2760cc96` | 1143 | `server/internal/modules/workspace/adapter/http/gen/server.gen.go` |
| `3901f55a5779df2a9e8470f125fde533258c1573e7a9fa8f3241306469914a63` | 1272 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go server/internal/modules/workspace/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 3: 操作名、规则表**

`server/internal/modules/workspace/domain/actions.go`（修改，2 处）：

````old server/internal/modules/workspace/domain/actions.go
	ActionUpdate shared.Action = "workspace.update"
````

````new server/internal/modules/workspace/domain/actions.go
	ActionUpdate shared.Action = "workspace.update"
	// ActionPreferencesRead is reading one's display settings in a
	// workspace: getWorkspacePreferences.
	ActionPreferencesRead shared.Action = "workspace_preferences.read"
	// ActionPreferencesUpdate is changing them: updateWorkspacePreferences.
	ActionPreferencesUpdate shared.Action = "workspace_preferences.update"
````

````old server/internal/modules/workspace/domain/actions.go
	return []shared.Action{ActionRead, ActionUpdate}
````

````new server/internal/modules/workspace/domain/actions.go
	return []shared.Action{ActionRead, ActionUpdate, ActionPreferencesRead, ActionPreferencesUpdate}
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"workspace.update": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
````

````new server/internal/modules/access/domain/rules.go
	"workspace.update": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	// One's own display settings: every active member (M3 design 9.2).
	"workspace_preferences.read":   {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	"workspace_preferences.update": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"workspace.read":   {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
	"workspace.update": {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
````

````new server/internal/modules/access/domain/rules_test.go
	"workspace.read":               {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
	"workspace.update":             {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
	"workspace_preferences.read":   {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
	"workspace_preferences.update": {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
````

- [ ] **Step 4: 用例**

`server/internal/modules/workspace/app/preferences.go`（新文件，97 行）：

````file server/internal/modules/workspace/app/preferences.go
package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetWorkspacePreferences reads the caller's display settings in a
// workspace: GET /api/v0/me/workspaces/{slug}/preferences.
type GetWorkspacePreferences struct {
	preferences PreferencesReader
	auth        shared.Authorizer
}

// NewGetWorkspacePreferences returns the use case.
func NewGetWorkspacePreferences(preferences PreferencesReader, auth shared.Authorizer) *GetWorkspacePreferences {
	return &GetWorkspacePreferences{preferences: preferences, auth: auth}
}

// Execute returns the caller's settings, or domain.DefaultPreferences while
// he has changed none. A read opens no transaction and writes nothing (M3
// design 3.18): the row is created by the first change.
func (u *GetWorkspacePreferences) Execute(ctx context.Context, slug string) (domain.Preferences, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Preferences{}, err
	}
	w, err := u.preferences.WorkspaceBySlug(ctx, slug)
	switch {
	case errors.Is(err, ErrNotFound):
		return domain.Preferences{}, domain.ErrNotFound
	case err != nil:
		return domain.Preferences{}, err
	}
	_, err = u.auth.Authorize(ctx, actor, domain.ActionPreferencesRead, shared.Target{WorkspaceID: w.ID})
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return domain.Preferences{}, domain.ErrNotFound
	case err != nil:
		return domain.Preferences{}, err
	}
	p, found, err := u.preferences.Preferences(ctx, w.ID, actor.UserID)
	switch {
	case err != nil:
		return domain.Preferences{}, err
	case !found:
		return domain.DefaultPreferences(), nil
	}
	return p, nil
}

// UpdateWorkspacePreferences changes the caller's display settings in a
// workspace: PATCH /api/v0/me/workspaces/{slug}/preferences.
type UpdateWorkspacePreferences struct {
	preferences PreferencesWriter
	auth        shared.Authorizer
	tx          shared.TxManager
	clock       Clock
}

// NewUpdateWorkspacePreferences returns the use case.
func NewUpdateWorkspacePreferences(preferences PreferencesWriter, auth shared.Authorizer, tx shared.TxManager, clock Clock) *UpdateWorkspacePreferences {
	return &UpdateWorkspacePreferences{preferences: preferences, auth: auth, tx: tx, clock: clock}
}

// Execute checks p, then in one transaction (M3 design 3.6): the workspace
// row FOR SHARE, the decision, the caller's row changed or inserted
// (M3 design 3.18).
func (u *UpdateWorkspacePreferences) Execute(ctx context.Context, slug string, p domain.PreferencesPatch) (domain.Preferences, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Preferences{}, err
	}
	if err := domain.CheckPreferencesPatch(p); err != nil {
		return domain.Preferences{}, err
	}
	now := u.clock.Now()
	var stored domain.Preferences
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		id, _, err := lockAndDecide(ctx, u.preferences.ShareWorkspaceBySlug, u.auth, actor, slug, domain.ActionPreferencesUpdate)
		if err != nil {
			return err
		}
		stored, err = u.preferences.UpsertPreferences(ctx, PreferencesRow{
			ID: uuid.NewV7(), WorkspaceID: id, UserID: actor.UserID, Patch: p, Now: now,
		})
		return err
	})
	if err != nil {
		return domain.Preferences{}, err
	}
	return stored, nil
}
````

`server/internal/modules/workspace/app/fakes_test.go`（修改，2 处）：

````old server/internal/modules/workspace/app/fakes_test.go
	members    []app.MemberRow
}
````

````new server/internal/modules/workspace/app/fakes_test.go
	members    []app.MemberRow
	prefs      map[prefsKey]domain.Preferences
	prefsErr   error // for Preferences and UpsertPreferences
	upserts    []app.PreferencesRow
}

// prefsKey is one (workspace, user) pair of fakeWorkspaces' preferences.
type prefsKey struct{ workspace, user uuid.UUID }
````

````old server/internal/modules/workspace/app/fakes_test.go
}

// show is *s quoted, or <nil>.
````

````new server/internal/modules/workspace/app/fakes_test.go
}

func (f *fakeWorkspaces) ShareWorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	f.log.add(ctx, "ShareWorkspaceBySlug %s", slug)
	return f.lock(slug)
}

func (f *fakeWorkspaces) Preferences(ctx context.Context, workspaceID, userID uuid.UUID) (domain.Preferences, bool, error) {
	f.log.add(ctx, "Preferences %s %s", workspaceID, userID)
	if f.prefsErr != nil {
		return domain.Preferences{}, false, fmt.Errorf("read workspace preferences: %w", f.prefsErr)
	}
	p, found := f.prefs[prefsKey{workspaceID, userID}]
	return p, found, nil
}

// UpsertPreferences logs the row without its id, which the use case makes
// anew each time; upserts keeps it whole.
func (f *fakeWorkspaces) UpsertPreferences(ctx context.Context, r app.PreferencesRow) (domain.Preferences, error) {
	var limit *string
	if r.Patch.NavigationProjectLimit != nil {
		limit = ptr(fmt.Sprint(*r.Patch.NavigationProjectLimit))
	}
	f.log.add(ctx, "UpsertPreferences %s %s mode=%s limit=%s at %s", r.WorkspaceID, r.UserID, show(r.Patch.NavigationControl), show(limit),
		r.Now.Format(time.RFC3339Nano))
	f.upserts = append(f.upserts, r)
	if f.prefsErr != nil {
		return domain.Preferences{}, fmt.Errorf("write workspace preferences: %w", f.prefsErr)
	}
	key := prefsKey{r.WorkspaceID, r.UserID}
	p, found := f.prefs[key]
	if !found {
		p = domain.DefaultPreferences()
	}
	if f.prefs == nil {
		f.prefs = map[prefsKey]domain.Preferences{}
	}
	f.prefs[key] = p.Apply(r.Patch)
	return f.prefs[key], nil
}

// show is *s quoted, or <nil>.
````

`server/internal/modules/workspace/app/preferences_test.go`（新文件，210 行）：

````file server/internal/modules/workspace/app/preferences_test.go
package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// prefsFixture is the preference use cases over fakes sharing one log:
// alice is a guest of acme and has settings there, bob a member of acme and
// of beta without any; carol sees neither.
type prefsFixture struct {
	log        *callLog
	tx         *fakeTx
	workspaces *fakeWorkspaces
	auth       *fakeAuthorizer
	get        *app.GetWorkspacePreferences
	update     *app.UpdateWorkspacePreferences
}

var tabbed = domain.Preferences{NavigationControl: "TABBED", NavigationProjectLimit: 3}

func newPrefs() *prefsFixture {
	log := &callLog{}
	f := &prefsFixture{log: log, tx: &fakeTx{},
		workspaces: &fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme, beta},
			prefs: map[prefsKey]domain.Preferences{{acme.ID, alice.ID}: tabbed}},
		auth: &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{
			{alice.ID, acme.ID}: {WorkspaceRole: shared.RoleGuest},
			{bob.ID, acme.ID}:   {WorkspaceRole: shared.RoleMember},
			{bob.ID, beta.ID}:   {WorkspaceRole: shared.RoleMember},
		}}}
	f.get = app.NewGetWorkspacePreferences(f.workspaces, f.auth)
	f.update = app.NewUpdateWorkspacePreferences(f.workspaces, f.auth, f.tx, clocktest.At(now))
	return f
}

// The caller's own settings in the workspace named, read without a
// transaction after the decision: alice's, and the defaults for bob, who has
// changed none, in either workspace.
func TestGetWorkspacePreferencesReadsTheCallersOwn(t *testing.T) {
	tests := []struct {
		user app.AccountState
		w    domain.Workspace
		want domain.Preferences
	}{
		{alice, acme, tabbed},
		{bob, acme, domain.DefaultPreferences()},
		{bob, beta, domain.DefaultPreferences()},
	}
	for _, tt := range tests {
		f := newPrefs()
		got, err := f.get.Execute(as(tt.user), tt.w.Slug)
		if err != nil || got != tt.want {
			t.Errorf("%s, %s: Execute() = %+v, %v; want %+v", tt.user.Email, tt.w.Slug, got, err, tt.want)
		}
		want := []string{
			"WorkspaceBySlug " + tt.w.Slug + " outside tx",
			"Authorize " + tt.user.ID.String() + " workspace_preferences.read on " + tt.w.ID.String() + "/" + uuid.Nil().String() + " outside tx",
			"Preferences " + tt.w.ID.String() + " " + tt.user.ID.String() + " outside tx",
		}
		if !slices.Equal(f.log.calls, want) || f.tx.calls != 0 || len(f.workspaces.upserts) != 0 {
			t.Errorf("%s, %s: calls = %q, %d transactions, %d writes; want %q, none", tt.user.Email, tt.w.Slug, f.log.calls, f.tx.calls,
				len(f.workspaces.upserts), want)
		}
	}
}

// A workspace that is not there and one the caller cannot see are the same
// workspace.not_found, and the settings are not read; a failure is the
// answer, never the defaults.
func TestGetWorkspacePreferencesRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	tests := []struct {
		name  string
		user  app.AccountState
		slug  string
		set   func(f *prefsFixture)
		want  error
		calls int
	}{
		{"no such workspace", alice, "nothing", nil, domain.ErrNotFound, 1},
		{"not visible", carol, "acme", nil, domain.ErrNotFound, 2},
		{"alice cannot see beta", alice, "beta", nil, domain.ErrNotFound, 2},
		{"forbidden", bob, "acme", func(f *prefsFixture) { f.auth.errs = map[grantKey]error{{bob.ID, acme.ID}: shared.Forbidden()} },
			shared.Forbidden(), 2},
		{"the store failed", alice, "gamma", func(f *prefsFixture) { f.workspaces.slugErrs = map[string]error{"gamma": failure} }, failure, 1},
		{"the Authorizer failed", alice, "acme", func(f *prefsFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: failure} },
			failure, 2},
		{"the read failed", alice, "acme", func(f *prefsFixture) { f.workspaces.prefsErr = failure }, failure, 3},
	}
	for _, tt := range tests {
		f := newPrefs()
		if tt.set != nil {
			tt.set(f)
		}
		if got, err := f.get.Execute(as(tt.user), tt.slug); !errors.Is(err, tt.want) || got != (domain.Preferences{}) {
			t.Errorf("%s: Execute() = %+v, %v; want no settings and %v", tt.name, got, err, tt.want)
		}
		if len(f.log.calls) != tt.calls {
			t.Errorf("%s: calls = %q, want %d", tt.name, f.log.calls, tt.calls)
		}
	}
	f := newPrefs()
	if _, err := f.get.Execute(context.Background(), "acme"); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and no call", err, f.log.calls)
	}
}

// UpdateWorkspacePreferences locks the workspace FOR SHARE, then decides,
// then writes the caller's row, in one transaction (M3 design 3.6): alice's
// change keeps what she set before, bob's first one starts from the
// defaults. Each write names a new row id.
func TestUpdateWorkspacePreferencesLocksThenDecidesThenWrites(t *testing.T) {
	tests := []struct {
		user app.AccountState
		w    domain.Workspace
		p    domain.PreferencesPatch
		call string
		want domain.Preferences
	}{
		{alice, acme, domain.PreferencesPatch{NavigationProjectLimit: ptr(0)}, `mode=<nil> limit="0"`, prefs("TABBED", 0)},
		{bob, acme, domain.PreferencesPatch{NavigationControl: ptr("TABBED")}, `mode="TABBED" limit=<nil>`, prefs("TABBED", 10)},
		{bob, beta, domain.PreferencesPatch{}, `mode=<nil> limit=<nil>`, domain.DefaultPreferences()},
	}
	for _, tt := range tests {
		f := newPrefs()
		got, err := f.update.Execute(as(tt.user), tt.w.Slug, tt.p)
		if err != nil || got != tt.want {
			t.Errorf("%s, %s: Execute() = %+v, %v; want %+v", tt.user.Email, tt.w.Slug, got, err, tt.want)
		}
		want := append(lockedDecision(tt.user, tt.w, "ShareWorkspaceBySlug", domain.ActionPreferencesUpdate),
			"UpsertPreferences "+tt.w.ID.String()+" "+tt.user.ID.String()+" "+tt.call+" at "+now.Format(time.RFC3339Nano))
		if !slices.Equal(f.log.calls, want) || f.tx.calls != 1 {
			t.Errorf("%s, %s: calls = %q in %d transactions, want %q in one", tt.user.Email, tt.w.Slug, f.log.calls, f.tx.calls, want)
		}
	}
	f := newPrefs()
	for range 2 {
		if _, err := f.update.Execute(as(bob), "beta", domain.PreferencesPatch{}); err != nil {
			t.Fatal(err)
		}
	}
	if ids := f.workspaces.upserts; ids[0].ID[6]>>4 != 7 || ids[0].ID == ids[1].ID {
		t.Errorf("row ids %s, %s; want two different version 7 ids", ids[0].ID, ids[1].ID)
	}
}

// Each refusal and failure is the answer, with nothing written: the values'
// check first, without a transaction; a workspace not there or not visible,
// workspace.not_found; a failed lock, decision or write, never a 404.
func TestUpdateWorkspacePreferencesRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	limit := domain.PreferencesPatch{NavigationProjectLimit: ptr(5)}
	shared403 := func(f *prefsFixture) { f.auth.errs = map[grantKey]error{{bob.ID, acme.ID}: shared.Forbidden()} }
	tests := []struct {
		name   string
		user   app.AccountState
		slug   string
		p      domain.PreferencesPatch
		set    func(f *prefsFixture)
		want   error
		calls  []string
		inTxes int
	}{
		{"invalid values", alice, "acme", domain.PreferencesPatch{NavigationProjectLimit: ptr(-1)}, nil,
			shared.Invalid(shared.FieldError{Field: "navigation_project_limit", Code: shared.FieldOutOfRange, Message: "must be between 0 and 2147483647"}), nil, 0},
		{"no such workspace", alice, "nothing", limit, nil, domain.ErrNotFound, []string{"ShareWorkspaceBySlug nothing"}, 1},
		{"not visible", carol, "acme", limit, nil, domain.ErrNotFound, lockedDecision(carol, acme, "ShareWorkspaceBySlug", domain.ActionPreferencesUpdate), 1},
		{"forbidden", bob, "acme", limit, shared403, shared.Forbidden(), lockedDecision(bob, acme, "ShareWorkspaceBySlug", domain.ActionPreferencesUpdate), 1},
		{"the lock failed", alice, "acme", limit, func(f *prefsFixture) { f.workspaces.lockErrs = map[string]error{"acme": failure} }, failure,
			[]string{"ShareWorkspaceBySlug acme"}, 1},
		{"the write failed", alice, "acme", limit, func(f *prefsFixture) { f.workspaces.prefsErr = failure }, failure,
			append(lockedDecision(alice, acme, "ShareWorkspaceBySlug", domain.ActionPreferencesUpdate),
				"UpsertPreferences "+acme.ID.String()+" "+alice.ID.String()+` mode=<nil> limit="5" at `+now.Format(time.RFC3339Nano)), 1},
	}
	for _, tt := range tests {
		f := newPrefs()
		if tt.set != nil {
			tt.set(f)
		}
		got, err := f.update.Execute(as(tt.user), tt.slug, tt.p)
		if !errors.Is(err, tt.want) || got != (domain.Preferences{}) {
			t.Errorf("%s: Execute() = %+v, %v; want no settings and %v", tt.name, got, err, tt.want)
		}
		if tt.want == failure && errors.Is(err, domain.ErrNotFound) {
			t.Errorf("%s: Execute() = %v, which is also workspace.not_found", tt.name, err)
		}
		if !slices.Equal(f.log.calls, tt.calls) || f.tx.calls != tt.inTxes {
			t.Errorf("%s: calls = %q in %d transactions, want %q in %d", tt.name, f.log.calls, f.tx.calls, tt.calls, tt.inTxes)
		}
	}
	f := newPrefs()
	if _, err := f.update.Execute(context.Background(), "acme", limit); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 || f.tx.calls != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and nothing", err, f.log.calls)
	}
}

// prefs is the settings mode and limit.
func prefs(mode string, limit int) domain.Preferences {
	return domain.Preferences{NavigationControl: mode, NavigationProjectLimit: limit}
}
````

- [ ] **Step 5: HTTP 一侧和接线**

`server/internal/modules/workspace/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/http/handler.go
}

// CheckSlugUseCase is app.CheckSlug.
````

````new server/internal/modules/workspace/adapter/http/handler.go
}

// GetPreferencesUseCase is app.GetWorkspacePreferences.
type GetPreferencesUseCase interface {
	Execute(ctx context.Context, slug string) (domain.Preferences, error)
}

// UpdatePreferencesUseCase is app.UpdateWorkspacePreferences.
type UpdatePreferencesUseCase interface {
	Execute(ctx context.Context, slug string, p domain.PreferencesPatch) (domain.Preferences, error)
}

// CheckSlugUseCase is app.CheckSlug.
````

````old server/internal/modules/workspace/adapter/http/handler.go
	ListWorkspaces  ListWorkspacesUseCase
	CreateWorkspace CreateWorkspaceUseCase
	GetWorkspace    GetWorkspaceUseCase
	UpdateWorkspace UpdateWorkspaceUseCase
	CheckSlug       CheckSlugUseCase
````

````new server/internal/modules/workspace/adapter/http/handler.go
	ListWorkspaces    ListWorkspacesUseCase
	CreateWorkspace   CreateWorkspaceUseCase
	GetWorkspace      GetWorkspaceUseCase
	UpdateWorkspace   UpdateWorkspaceUseCase
	CheckSlug         CheckSlugUseCase
	GetPreferences    GetPreferencesUseCase
	UpdatePreferences UpdatePreferencesUseCase
````

`server/internal/modules/workspace/adapter/http/preferences.go`（新文件，39 行）：

````file server/internal/modules/workspace/adapter/http/preferences.go
package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// GetWorkspacePreferences serves GET /api/v0/me/workspaces/{slug}/preferences.
func (h handler) GetWorkspacePreferences(ctx context.Context, req gen.GetWorkspacePreferencesRequestObject) (gen.GetWorkspacePreferencesResponseObject, error) {
	p, err := h.uc.GetPreferences.Execute(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	return gen.GetWorkspacePreferences200JSONResponse(preferences(p)), nil
}

// UpdateWorkspacePreferences serves PATCH /api/v0/me/workspaces/{slug}/preferences.
func (h handler) UpdateWorkspacePreferences(ctx context.Context, req gen.UpdateWorkspacePreferencesRequestObject) (gen.UpdateWorkspacePreferencesResponseObject, error) {
	patch := domain.PreferencesPatch{NavigationProjectLimit: req.Body.NavigationProjectLimit}
	if mode := req.Body.NavigationControlPreference; mode != nil {
		m := string(*mode)
		patch.NavigationControl = &m
	}
	p, err := h.uc.UpdatePreferences.Execute(ctx, req.Slug, patch)
	if err != nil {
		return nil, err
	}
	return gen.UpdateWorkspacePreferences200JSONResponse(preferences(p)), nil
}

// preferences is p as the API shows it.
func preferences(p domain.Preferences) gen.WorkspacePreferences {
	return gen.WorkspacePreferences{
		NavigationControlPreference: gen.NavigationControlPreference(p.NavigationControl),
		NavigationProjectLimit:      p.NavigationProjectLimit,
	}
}
````

`server/internal/modules/workspace/adapter/http/handler_test.go`（修改，4 处）：

````old server/internal/modules/workspace/adapter/http/handler_test.go
	check  *fakeCheck
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
	check  *fakeCheck
	prefs  *fakePrefs
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
}

type fakeCheck struct {
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
}

// fakePrefs is both preference use cases: each call is recorded as
// "caller slug", and a PATCH's patch; it answers what it is given.
type fakePrefs struct {
	calls   []string
	patches []domain.PreferencesPatch
	answer  domain.Preferences
	err     error
}

type fakeGetPrefs struct{ *fakePrefs }

func (f fakeGetPrefs) Execute(ctx context.Context, slug string) (domain.Preferences, error) {
	f.calls = append(f.calls, "GET "+caller(ctx)+" "+slug)
	return f.answer, f.err
}

type fakeUpdatePrefs struct{ *fakePrefs }

func (f fakeUpdatePrefs) Execute(ctx context.Context, slug string, p domain.PreferencesPatch) (domain.Preferences, error) {
	f.calls = append(f.calls, "PATCH "+caller(ctx)+" "+slug)
	f.patches = append(f.patches, p)
	return f.answer, f.err
}

type fakeCheck struct {
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
		f.check = &fakeCheck{}
	}
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
		f.check = &fakeCheck{}
	}
	if f.prefs == nil {
		f.prefs = &fakePrefs{}
	}
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
		ListWorkspaces: f.list, CreateWorkspace: f.create, GetWorkspace: f.get, UpdateWorkspace: f.update, CheckSlug: f.check,
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
		ListWorkspaces: f.list, CreateWorkspace: f.create, GetWorkspace: f.get, UpdateWorkspace: f.update, CheckSlug: f.check,
		GetPreferences: fakeGetPrefs{f.prefs}, UpdatePreferences: fakeUpdatePrefs{f.prefs},
````

`server/internal/modules/workspace/adapter/http/preferences_test.go`（新文件，88 行）：

````file server/internal/modules/workspace/adapter/http/preferences_test.go
package httpadapter_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GET answers the use case's settings for the caller and the slug of the
// path.
func TestGetWorkspacePreferences(t *testing.T) {
	prefs := &fakePrefs{answer: domain.Preferences{NavigationControl: "TABBED", NavigationProjectLimit: 0}}
	h := newServer(t, fakes{prefs: prefs})
	for _, token := range []string{"alice", "bob"} {
		res, body := do(t, h, request(http.MethodGet, "/api/v0/me/workspaces/acme/preferences", token, ""))
		if want := `{"navigation_control_preference":"TABBED","navigation_project_limit":0}` + "\n"; res.StatusCode != http.StatusOK || body != want {
			t.Errorf("%s: GET = %d %s, want 200 %s", token, res.StatusCode, body, want)
		}
	}
	if want := []string{"GET alice acme", "GET bob acme"}; !slices.Equal(prefs.calls, want) {
		t.Errorf("calls = %q, want %q", prefs.calls, want)
	}
}

// The body becomes the patch, a field left out nil; the answer is 200 with
// the settings as stored.
func TestUpdateWorkspacePreferences(t *testing.T) {
	prefs := &fakePrefs{answer: domain.Preferences{NavigationControl: "ACCORDION", NavigationProjectLimit: 3}}
	h := newServer(t, fakes{prefs: prefs})
	for _, body := range []string{`{"navigation_control_preference":"TABBED","navigation_project_limit":3}`, `{"navigation_project_limit":0}`, `{}`} {
		res, got := do(t, h, request(http.MethodPatch, "/api/v0/me/workspaces/beta/preferences", "bob", body))
		if want := `{"navigation_control_preference":"ACCORDION","navigation_project_limit":3}` + "\n"; res.StatusCode != http.StatusOK || got != want {
			t.Errorf("PATCH %s = %d %s, want 200 %s", body, res.StatusCode, got, want)
		}
	}
	tabbed, three, zero := "TABBED", 3, 0
	want := []domain.PreferencesPatch{{NavigationControl: &tabbed, NavigationProjectLimit: &three}, {NavigationProjectLimit: &zero}, {}}
	if len(prefs.patches) != len(want) {
		t.Fatalf("patches = %+v, want %+v", prefs.patches, want)
	}
	for i := range want {
		got := prefs.patches[i]
		sameMode := (got.NavigationControl == nil) == (want[i].NavigationControl == nil) &&
			(got.NavigationControl == nil || *got.NavigationControl == *want[i].NavigationControl)
		sameLimit := (got.NavigationProjectLimit == nil) == (want[i].NavigationProjectLimit == nil) &&
			(got.NavigationProjectLimit == nil || *got.NavigationProjectLimit == *want[i].NavigationProjectLimit)
		if !sameMode || !sameLimit {
			t.Errorf("patch %d = %+v, want %+v", i, got, want[i])
		}
	}
	if calls := []string{"PATCH bob beta", "PATCH bob beta", "PATCH bob beta"}; !slices.Equal(prefs.calls, calls) {
		t.Errorf("calls = %q, want %q", prefs.calls, calls)
	}
}

// The use cases' refusals, as the contract declares them; a limit that is
// not a whole number is refused before them.
func TestWorkspacePreferencesRefusals(t *testing.T) {
	notFound := `{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`
	tests := []struct {
		name, method, body string
		err                error
		status             int
		want               string
	}{
		{"GET, not visible", http.MethodGet, "", domain.ErrNotFound, http.StatusNotFound, notFound},
		{"PATCH, not visible", http.MethodPatch, `{"navigation_project_limit":3}`, domain.ErrNotFound, http.StatusNotFound, notFound},
		{"PATCH, invalid values", http.MethodPatch, `{"navigation_project_limit":-1}`,
			shared.Invalid(shared.FieldError{Field: "navigation_project_limit", Code: shared.FieldOutOfRange, Message: "must be between 0 and 2147483647"}),
			http.StatusUnprocessableEntity, `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"navigation_project_limit","code":"out_of_range","message":"must be between 0 and 2147483647"}]}`},
	}
	for _, tt := range tests {
		h := newServer(t, fakes{prefs: &fakePrefs{err: tt.err}})
		res, body := do(t, h, request(tt.method, "/api/v0/me/workspaces/acme/preferences", "alice", tt.body))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
	prefs := &fakePrefs{}
	h := newServer(t, fakes{prefs: prefs})
	if res, _ := do(t, h, request(http.MethodPatch, "/api/v0/me/workspaces/acme/preferences", "alice", `{"navigation_project_limit":2.5}`)); res.StatusCode != http.StatusBadRequest || len(prefs.calls) != 0 {
		t.Errorf("PATCH with a fraction = %d, calls %q; want 400 and no call", res.StatusCode, prefs.calls)
	}
}
````

`server/internal/modules/workspace/module.go`（修改，2 处）：

````old server/internal/modules/workspace/module.go
// changing workspaces, and checking a slug, and offers the other modules its
// reads through ports.
````

````new server/internal/modules/workspace/module.go
// changing workspaces, checking a slug, and each member's display settings,
// and offers the other modules its reads through ports.
````

````old server/internal/modules/workspace/module.go
		GetWorkspace:    app.NewGetWorkspace(store, d.Authorizer),
		UpdateWorkspace: app.NewUpdateWorkspace(store, d.Authorizer, d.Tx, d.Clock),
		CheckSlug:       app.NewCheckSlug(store),
````

````new server/internal/modules/workspace/module.go
		GetWorkspace:      app.NewGetWorkspace(store, d.Authorizer),
		UpdateWorkspace:   app.NewUpdateWorkspace(store, d.Authorizer, d.Tx, d.Clock),
		CheckSlug:         app.NewCheckSlug(store),
		GetPreferences:    app.NewGetWorkspacePreferences(store, d.Authorizer),
		UpdatePreferences: app.NewUpdateWorkspacePreferences(store, d.Authorizer, d.Tx, d.Clock),
````

- [ ] **Step 6: 矩阵**

`server/internal/bootstrap/permission_matrix_test.go`（修改，4 处）：

````old server/internal/bootstrap/permission_matrix_test.go
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
````

````new server/internal/bootstrap/permission_matrix_test.go
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
````

````old server/internal/bootstrap/permission_matrix_test.go
// later removed; the workspace gone with its admin; and the workspace
// other, whose admin was never a member of acme and where the removed
// member is still active, so that a role read in the wrong workspace lets
// either into acme. Through SQL, until the stores of P5 and P2 replace it:
// the removed member's membership of acme ended, and gone's workspace row
// alone soft-deleted. Everything that connected to the database is closed
// when it returns, so that it can be copied.
````

````new server/internal/bootstrap/permission_matrix_test.go
// later removed, and the admin's display settings there; the workspace gone
// with its admin; and the workspace other, whose admin was never a member of
// acme and where the removed member is still active, so that a role read in
// the wrong workspace lets either into acme. Through SQL, until the stores
// of P5 and P2 replace it: the removed member's membership of acme ended,
// and gone's workspace row alone soft-deleted. Everything that connected to
// the database is closed when it returns, so that it can be copied.
````

````old server/internal/bootstrap/permission_matrix_test.go
		seed.join(acme, callerRemoved, shared.RoleMember)
````

````new server/internal/bootstrap/permission_matrix_test.go
		seed.join(acme, callerRemoved, shared.RoleMember)
		tabbed, three := "TABBED", 3
		seed.preferences(acme, callerAdmin, workspacedomain.PreferencesPatch{NavigationControl: &tabbed, NavigationProjectLimit: &three})
````

````old server/internal/bootstrap/permission_matrix_test.go
		ID: uuid.NewV7(), WorkspaceID: workspace, MemberID: s.ids[c], Role: role, CreatedBy: s.ids[c], Now: s.now,
````

````new server/internal/bootstrap/permission_matrix_test.go
		ID: uuid.NewV7(), WorkspaceID: workspace, MemberID: s.ids[c], Role: role, CreatedBy: s.ids[c], Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
}

func (s matrixSeed) preferences(workspace uuid.UUID, c caller, p workspacedomain.PreferencesPatch) {
	s.t.Helper()
	if _, err := s.store.UpsertPreferences(context.Background(), workspaceapp.PreferencesRow{
		ID: uuid.NewV7(), WorkspaceID: workspace, UserID: s.ids[c], Patch: p, Now: s.now,
````

`server/internal/bootstrap/permission_matrix_workspace_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_workspace_test.go
			cells: inWorkspace(cellOK, cellForbidden, cellForbidden), check: renamesIt},
````

````new server/internal/bootstrap/permission_matrix_workspace_test.go
			cells: inWorkspace(cellOK, cellForbidden, cellForbidden), check: renamesIt},
		{op: "getWorkspacePreferences", request: toPreferences(http.MethodGet, ""), cells: inWorkspace(cellOK, cellOK, cellOK),
			check: preferencesAre(navigation{"TABBED", 3}, navigation{"ACCORDION", 10})},
		{op: "updateWorkspacePreferences", write: true, request: toPreferences(http.MethodPatch, `{"navigation_project_limit":5}`),
			cells: inWorkspace(cellOK, cellOK, cellOK), check: preferencesAre(navigation{"TABBED", 5}, navigation{"ACCORDION", 5})},
	}
}

// toPreferences is the request of a row whose callers each send method to
// their own display settings in the workspace their column targets.
func toPreferences(method, body string) func(caller) (string, string, string) {
	return func(c caller) (string, string, string) {
		return method, "/api/v0/me/workspaces/" + workspaceOf(c) + "/preferences", body
	}
}

// navigation is a caller's display settings as an answer holds them.
type navigation struct {
	Mode  string `json:"navigation_control_preference"`
	Limit int    `json:"navigation_project_limit"`
}

// preferencesAre: each caller reads and changes his own settings; the
// admin's answer is admin, the member's and the guest's others.
func preferencesAre(admin, others navigation) func(t *testing.T, c caller, answer string) {
	return func(t *testing.T, c caller, answer string) {
		var got navigation
		decodeAnswer(t, answer, &got)
		want := others
		if c == callerAdmin {
			want = admin
		}
		if got != want {
			t.Errorf("%s's settings are %+v, want %+v", c, got, want)
		}
````

- [ ] **Step 7: 测试、检查**

Run: `go -C server test -count=1 ./internal/modules/workspace/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix|TestThePermissionMatrixCoversEveryOperation|TestEveryActionHasARuleAndEveryRuleAnAction' ./internal/bootstrap/`
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

- [ ] **Step 8: 提交**

```bash
git add api/modules/workspace.yaml api/openapi.yaml server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/permission_matrix_workspace_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/workspace/adapter/http/handler.go server/internal/modules/workspace/adapter/http/handler_test.go server/internal/modules/workspace/adapter/http/preferences.go server/internal/modules/workspace/adapter/http/preferences_test.go server/internal/modules/workspace/app/fakes_test.go server/internal/modules/workspace/app/preferences.go server/internal/modules/workspace/app/preferences_test.go server/internal/modules/workspace/domain/actions.go server/internal/modules/workspace/module.go api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go server/internal/modules/workspace/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P2): read and change one's display settings in a workspace

GET answers the caller's settings, the defaults until the first change,
and writes nothing; PATCH takes the workspace FOR SHARE, decides, and
upserts the caller's row. Any active member, for his own settings only.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**：

| 改坏 | 必须失败的测试 |
|---|---|
| 锁挪到事务之前 | `TestUpdateWorkspacePreferencesLocksThenDecidesThenWrites` |
| 改用 `LockWorkspaceBySlug`（`FOR NO KEY UPDATE`） | `TestUpdateWorkspacePreferencesLocksThenDecidesThenWrites` |
| 写入时的账户不是调用者 | `TestUpdateWorkspacePreferencesLocksThenDecidesThenWrites` |
| `workspace_preferences.update` 只给管理员 | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix/updateWorkspacePreferences/member`、`…/guest` |
| 存储的 `Preferences` 不看 `user_id` | `TestPermissionMatrix/getWorkspacePreferences/member`、`…/guest`（加上 Task 7 的存储测试） |
| 冲突的目标换成 `(id)` | `TestPermissionMatrix/updateWorkspacePreferences/admin`（加上 Task 7 的存储测试） |

`GET` 不写库由端口保证：`PreferencesReader` 没有写的方法。

**Done when:** 两个用例、两个 handler、矩阵的 12 格通过；`GET` 在没有行时答默认值、不写库。

---

### Task 9: `deleteWorkspace` 和它的连带

**Files:**
- Create: `server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go`、`server/internal/modules/workspace/app/delete_workspace.go`、`server/internal/modules/workspace/app/delete_workspace_test.go`
- Modify: `api/modules/workspace.yaml`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/permission_matrix_workspace_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/workspace/adapter/http/handler.go`、`server/internal/modules/workspace/adapter/http/handler_test.go`、`server/internal/modules/workspace/adapter/http/workspaces.go`、`server/internal/modules/workspace/adapter/http/workspaces_test.go`、`server/internal/modules/workspace/adapter/postgres/failures_test.go`、`server/internal/modules/workspace/adapter/postgres/preferences.go`、`server/internal/modules/workspace/adapter/postgres/queries/members.sql`、`server/internal/modules/workspace/adapter/postgres/queries/preferences.sql`、`server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql`、`server/internal/modules/workspace/adapter/postgres/workspaces.go`、`server/internal/modules/workspace/app/fakes_test.go`、`server/internal/modules/workspace/app/ports.go`、`server/internal/modules/workspace/domain/actions.go`、`server/internal/modules/workspace/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/workspace/adapter/http/gen/server.gen.go`、`server/internal/modules/workspace/adapter/postgres/gen/members.sql.go`、`server/internal/modules/workspace/adapter/postgres/gen/preferences.sql.go`、`server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.10，M3 设计 3.6、3.14、4.12）：
  - 接口描述：`deleteWorkspace`（`DELETE /api/v0/workspaces/{slug}`，204，码 `[workspace.not_found, forbidden]`）；
  - `domain.ActionDelete = "workspace.delete"`；规则表一行：管理员；
  - 查询 `DeleteWorkspace`、`DeleteWorkspaceMembers`、`DeleteWorkspacePreferences`：一条语句软删除这个工作区的未删除的行（`deleted_at`、`updated_at` 同一时刻，`updated_by_id` 是删除者），已删除的行保持原来的时间；
  - `app.WorkspaceDeleter{WorkspaceLocker; DeleteWorkspace; DeleteWorkspaceMembers; DeleteWorkspacePreferences}`；
  - `app.NewDeleteWorkspace(workspaces, auth, tx, clock, logger)`：一个事务：`lockAndDecide(LockWorkspaceBySlug, workspace.delete)` → `cascade()` 的每一步（工作区行、成员、显示设置），同一个时刻；提交之后记 INFO `workspace deleted`（`workspace_id`、`user_id`）；不清任何人的 `last_workspace_id`（3.14）。`cascade()` 是连带的唯一列表：P3 在工作区行之后加邀请，P4 在最后加项目（经 `ProjectCascade`），P7 加标签；
  - HTTP：handler `DeleteWorkspace`；`module.go` 传入 `d.Logger`。
- 使用者：矩阵的"工作区已删除"一列改为经这个操作准备（P1 的移交，spec 第 3 节第 2 条表的第 1 行）；P3、P4、P7 的连带。

**Tests:**
- `app/delete_workspace_test.go`：`TestDeleteWorkspaceLocksThenDecidesThenCascades`（两个调用者、两个工作区；锁 → 判定 → 三步，都在一个事务里，同一个时刻、同一个删除者；日志恰好一行）；`TestDeleteWorkspaceRefusals`（不存在、看不到、`forbidden`、锁失败，三步中任何一步失败：之后的步骤不执行，事务以错误结束，不记日志，失败不是 404；没有调用者 401）。
- `adapter/postgres/delete_workspace_test.go`（真实数据库）：
  - `TestDeletingAWorkspaceSoftDeletesItsRows`：三步在一个事务里：`acme`、它的三个成员关系（一个已结束）、两个成员的设置在同一时刻由 bob 删除；dave 的成员关系、carol 的设置在此之前已删除，时间不变；`beta` 的每一行不变；`is_active` 不变；slug 立即可以再用；
  - `TestAFailedDeletionLeavesTheWorkspace`：真实的用例、最后一步失败：工作区和两个成员都还在，没有删除的成员关系。
- `failures_test.go`：写的测试加三步。
- `adapter/http/workspaces_test.go`：`TestDeleteWorkspace`（204、没有正文；两个码）。
- 矩阵：`deleteWorkspace` 一行（管理员 204，成员、访客 `forbidden`，另三列 `workspace.not_found`）；准备数据中 `gone` 改为由它的管理员经 `DELETE` 删除（连带删除成员关系），代替 P1 的 SQL。

- [ ] **Step 1: 接口描述、操作名、规则表**

`api/modules/workspace.yaml`（修改，1 处）：

````old api/modules/workspace.yaml
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Workspace'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspace-slugs/{slug}:
````

````new api/modules/workspace.yaml
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Workspace'
        default:
          $ref: '#/components/responses/Problem'
    delete:
      operationId: deleteWorkspace
      tags: [workspace]
      summary: Delete a workspace
      description: >-
        For the workspace's admins. The workspace, its members and their
        display settings are soft-deleted in one transaction, at the same
        moment; the slug can name a new workspace at once. Nobody's
        last_workspace_id is cleared. A workspace that does not exist, is
        deleted, or of which the caller is not an active member answers
        workspace.not_found; a member or a guest, forbidden.
      security: [{bearer: []}]
      x-problem-codes: [workspace.not_found, forbidden]
      responses:
        '204':
          description: The workspace is deleted.
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspace-slugs/{slug}:
````

`server/internal/modules/workspace/domain/actions.go`（修改，2 处）：

````old server/internal/modules/workspace/domain/actions.go
	ActionUpdate shared.Action = "workspace.update"
````

````new server/internal/modules/workspace/domain/actions.go
	ActionUpdate shared.Action = "workspace.update"
	// ActionDelete is deleting a workspace: deleteWorkspace.
	ActionDelete shared.Action = "workspace.delete"
````

````old server/internal/modules/workspace/domain/actions.go
	return []shared.Action{ActionRead, ActionUpdate, ActionPreferencesRead, ActionPreferencesUpdate}
````

````new server/internal/modules/workspace/domain/actions.go
	return []shared.Action{ActionRead, ActionUpdate, ActionDelete, ActionPreferencesRead, ActionPreferencesUpdate}
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"workspace.update": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
````

````new server/internal/modules/access/domain/rules.go
	"workspace.update": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	"workspace.delete": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"workspace.update":             {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
````

````new server/internal/modules/access/domain/rules_test.go
	"workspace.update":             {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
	"workspace.delete":             {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
````

- [ ] **Step 2: 查询和生成**

`server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql
           WHERE c.workspace_id = w.id AND c.is_active AND c.deleted_at IS NULL) AS total_members;

````

````new server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql
           WHERE c.workspace_id = w.id AND c.is_active AND c.deleted_at IS NULL) AS total_members;

-- name: DeleteWorkspace :exec
-- deleteWorkspace's first step, under the workspace's FOR NO KEY UPDATE: the slug is free again at once (the partial
-- unique index). The rows under it are soft-deleted at the same moment by the steps that follow (M3 design 3.6).
UPDATE workspaces
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

````

`server/internal/modules/workspace/adapter/postgres/queries/members.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/members.sql
        sqlc.arg(created_by), sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now));
````

````new server/internal/modules/workspace/adapter/postgres/queries/members.sql
        sqlc.arg(created_by), sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now));

-- name: DeleteWorkspaceMembers :exec
-- deleteWorkspace's cascade: every undeleted membership of the workspace, active or not, one statement in scan order
-- under the workspace's FOR NO KEY UPDATE (M3 design 3.6 convention 5). A row deleted before keeps its time.
UPDATE workspace_members
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;
````

`server/internal/modules/workspace/adapter/postgres/queries/preferences.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/preferences.sql
WHERE workspace_id = sqlc.arg(workspace_id) AND user_id = sqlc.arg(user_id) AND deleted_at IS NULL;
````

````new server/internal/modules/workspace/adapter/postgres/queries/preferences.sql
WHERE workspace_id = sqlc.arg(workspace_id) AND user_id = sqlc.arg(user_id) AND deleted_at IS NULL;

-- name: DeleteWorkspacePreferences :exec
-- deleteWorkspace's cascade: every member's display settings in the workspace (M3 design 3.6 convention 5).
UPDATE workspace_user_properties
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;
````

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `d05e4b2dedb5b4a1ce55230254beb06e211b064fe90d9834c0ed67020803e263` | 1217 | `api/dist/openapi.yaml` |
| `310d4770501d77927a24f363e297d888c1dc5fd48c91586869b2d5e8fb72d1f6` | 1242 | `server/internal/modules/workspace/adapter/http/gen/server.gen.go` |
| `efc58c277da9e2e53e4c55d74ba29362a0ba18d8023e594a7d39ab3847dc474f` | 81 | `server/internal/modules/workspace/adapter/postgres/gen/members.sql.go` |
| `f0a4aeb7e356ff84d554eab669c82bdd2176e340efe442c2978daa81c88e722d` | 108 | `server/internal/modules/workspace/adapter/postgres/gen/preferences.sql.go` |
| `dac7e04ed88362447bdc0a6a022fe8423fa6ef34c1d571f595e80dd30c3f751b` | 282 | `server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go` |
| `5ecd58f05192c56b033ef938b6972e498aaa15bcd5cf3719a2dff53493d612d9` | 1298 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/server.gen.go server/internal/modules/workspace/adapter/postgres/gen/members.sql.go server/internal/modules/workspace/adapter/postgres/gen/preferences.sql.go server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 3: 端口和用例**

`server/internal/modules/workspace/app/ports.go`（修改，1 处）：

````old server/internal/modules/workspace/app/ports.go
}

// PreferencesRow is a change of an account's display settings in a
````

````new server/internal/modules/workspace/app/ports.go
}

// WorkspaceDeleter soft-deletes a workspace and the rows under it, under its
// lock. Each step sets deleted_at and updated_at to now and updated_by_id to
// by, on the undeleted rows only.
type WorkspaceDeleter interface {
	WorkspaceLocker
	// DeleteWorkspace soft-deletes the workspace row.
	DeleteWorkspace(ctx context.Context, id, by uuid.UUID, now time.Time) error
	// DeleteWorkspaceMembers soft-deletes its memberships, active or not.
	DeleteWorkspaceMembers(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
	// DeleteWorkspacePreferences soft-deletes its members' display
	// settings.
	DeleteWorkspacePreferences(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
}

// PreferencesRow is a change of an account's display settings in a
````

`server/internal/modules/workspace/app/delete_workspace.go`（新文件，68 行）：

````file server/internal/modules/workspace/app/delete_workspace.go
package app

import (
	"context"
	"log/slog"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteWorkspace deletes a workspace: DELETE /api/v0/workspaces/{slug}.
type DeleteWorkspace struct {
	workspaces WorkspaceDeleter
	auth       shared.Authorizer
	tx         shared.TxManager
	clock      Clock
	logger     *slog.Logger
}

// NewDeleteWorkspace returns the use case.
func NewDeleteWorkspace(workspaces WorkspaceDeleter, auth shared.Authorizer, tx shared.TxManager, clock Clock, logger *slog.Logger) *DeleteWorkspace {
	return &DeleteWorkspace{workspaces: workspaces, auth: auth, tx: tx, clock: clock, logger: logger}
}

// cascade is what deleting a workspace soft-deletes, in the order of M3
// design 3.6: the workspace row, then the rows under it, each step one
// statement at the same moment. Later phases add theirs here: P3 the
// invitations after the workspace row, P4 the projects at the end, through
// ProjectCascade (M3 design 3.3).
func (u *DeleteWorkspace) cascade() []func(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return []func(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error{
		u.workspaces.DeleteWorkspace,
		u.workspaces.DeleteWorkspaceMembers,
		u.workspaces.DeleteWorkspacePreferences,
	}
}

// Execute deletes the workspace in one transaction (M3 design 3.6): the
// workspace row FOR NO KEY UPDATE, the decision on workspace.delete, then
// the cascade. Nobody's last_workspace_id is cleared (M3 design 3.14).
func (u *DeleteWorkspace) Execute(ctx context.Context, slug string) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	now := u.clock.Now()
	var deleted uuid.UUID
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		id, _, err := lockAndDecide(ctx, u.workspaces.LockWorkspaceBySlug, u.auth, actor, slug, domain.ActionDelete)
		if err != nil {
			return err
		}
		for _, step := range u.cascade() {
			if err := step(ctx, id, actor.UserID, now); err != nil {
				return err
			}
		}
		deleted = id
		return nil
	})
	if err != nil {
		return err
	}
	u.logger.InfoContext(ctx, "workspace deleted", slog.String("workspace_id", deleted.String()), slog.String("user_id", actor.UserID.String()))
	return nil
}
````

`server/internal/modules/workspace/app/fakes_test.go`（修改，2 处）：

````old server/internal/modules/workspace/app/fakes_test.go
	updateErr  error               // for UpdateWorkspace
````

````new server/internal/modules/workspace/app/fakes_test.go
	updateErr  error               // for UpdateWorkspace
	deleteErrs map[string]error    // by step, e.g. "DeleteWorkspaceMembers"
````

````old server/internal/modules/workspace/app/fakes_test.go
}

// show is *s quoted, or <nil>.
````

````new server/internal/modules/workspace/app/fakes_test.go
}

func (f *fakeWorkspaces) DeleteWorkspace(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	return f.deleteStep(ctx, "DeleteWorkspace", id, by, now)
}

func (f *fakeWorkspaces) DeleteWorkspaceMembers(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.deleteStep(ctx, "DeleteWorkspaceMembers", workspaceID, by, now)
}

func (f *fakeWorkspaces) DeleteWorkspacePreferences(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.deleteStep(ctx, "DeleteWorkspacePreferences", workspaceID, by, now)
}

// deleteStep logs a step of the deletion and fails it with the error set
// for it, wrapped as the store wraps it.
func (f *fakeWorkspaces) deleteStep(ctx context.Context, step string, id, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "%s %s by %s at %s", step, id, by, now.Format(time.RFC3339Nano))
	if err := f.deleteErrs[step]; err != nil {
		return fmt.Errorf("%s: %w", step, err)
	}
	return nil
}

// show is *s quoted, or <nil>.
````

`server/internal/modules/workspace/app/delete_workspace_test.go`（新文件，120 行）：

````file server/internal/modules/workspace/app/delete_workspace_test.go
package app_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// deleteFixture is DeleteWorkspace over fakes sharing one log: alice is
// acme's admin, bob beta's.
type deleteFixture struct {
	log        *callLog
	tx         *fakeTx
	workspaces *fakeWorkspaces
	auth       *fakeAuthorizer
	logs       *strings.Builder
}

func newDelete() (*app.DeleteWorkspace, *deleteFixture) {
	log := &callLog{}
	f := &deleteFixture{log: log, tx: &fakeTx{}, workspaces: &fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme, beta}},
		auth: &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{
			{alice.ID, acme.ID}: {WorkspaceRole: shared.RoleAdmin},
			{bob.ID, beta.ID}:   {WorkspaceRole: shared.RoleAdmin},
		}}, logs: &strings.Builder{}}
	return app.NewDeleteWorkspace(f.workspaces, f.auth, f.tx, clocktest.At(now), slog.New(slog.NewTextHandler(f.logs, nil))), f
}

// cascadeCalls are the deletion's steps on w by user, in order.
func cascadeCalls(user app.AccountState, w domain.Workspace) []string {
	var calls []string
	for _, step := range []string{"DeleteWorkspace", "DeleteWorkspaceMembers", "DeleteWorkspacePreferences"} {
		calls = append(calls, step+" "+w.ID.String()+" by "+user.ID.String()+" at "+now.Format(time.RFC3339Nano))
	}
	return calls
}

// DeleteWorkspace locks the workspace FOR NO KEY UPDATE, decides, then
// soft-deletes the workspace, its members and their settings, by the caller
// at one moment, in one transaction (M3 design 3.6), and logs it. Two
// callers, two workspaces.
func TestDeleteWorkspaceLocksThenDecidesThenCascades(t *testing.T) {
	for _, tt := range []struct {
		user app.AccountState
		w    domain.Workspace
	}{{alice, acme}, {bob, beta}} {
		uc, f := newDelete()
		if err := uc.Execute(as(tt.user), tt.w.Slug); err != nil {
			t.Errorf("%s, %s: Execute() = %v", tt.user.Email, tt.w.Slug, err)
		}
		want := append(lockedDecision(tt.user, tt.w, "LockWorkspaceBySlug", domain.ActionDelete), cascadeCalls(tt.user, tt.w)...)
		if !slices.Equal(f.log.calls, want) || f.tx.calls != 1 {
			t.Errorf("%s, %s: calls = %q in %d transactions, want %q in one", tt.user.Email, tt.w.Slug, f.log.calls, f.tx.calls, want)
		}
		wantLog := `level=INFO msg="workspace deleted" workspace_id=` + tt.w.ID.String() + " user_id=" + tt.user.ID.String() + "\n"
		if _, got, _ := strings.Cut(f.logs.String(), " "); got != wantLog {
			t.Errorf("%s, %s: log %q, want %q", tt.user.Email, tt.w.Slug, got, wantLog)
		}
	}
}

// Each refusal and failure is the answer: nothing is deleted after it, and
// nothing is logged. A step that fails ends the cascade there, and its
// transaction ends with the error, so the database rolls back the steps
// before it; never a 404 for a failure.
func TestDeleteWorkspaceRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	decided := lockedDecision(alice, acme, "LockWorkspaceBySlug", domain.ActionDelete)
	steps := cascadeCalls(alice, acme)
	tests := []struct {
		name  string
		user  app.AccountState
		slug  string
		set   func(f *deleteFixture)
		want  error
		calls []string
	}{
		{"no such workspace", alice, "nothing", nil, domain.ErrNotFound, []string{"LockWorkspaceBySlug nothing"}},
		{"not visible", bob, "acme", nil, domain.ErrNotFound, lockedDecision(bob, acme, "LockWorkspaceBySlug", domain.ActionDelete)},
		{"forbidden", carol, "acme", func(f *deleteFixture) { f.auth.errs = map[grantKey]error{{carol.ID, acme.ID}: shared.Forbidden()} },
			shared.Forbidden(), lockedDecision(carol, acme, "LockWorkspaceBySlug", domain.ActionDelete)},
		{"the lock failed", alice, "acme", func(f *deleteFixture) { f.workspaces.lockErrs = map[string]error{"acme": failure} }, failure,
			[]string{"LockWorkspaceBySlug acme"}},
		{"the workspace row failed", alice, "acme", func(f *deleteFixture) { f.workspaces.deleteErrs = map[string]error{"DeleteWorkspace": failure} },
			failure, append(slices.Clone(decided), steps[0])},
		{"the members failed", alice, "acme", func(f *deleteFixture) { f.workspaces.deleteErrs = map[string]error{"DeleteWorkspaceMembers": failure} },
			failure, append(slices.Clone(decided), steps[:2]...)},
		{"the settings failed", alice, "acme",
			func(f *deleteFixture) {
				f.workspaces.deleteErrs = map[string]error{"DeleteWorkspacePreferences": failure}
			},
			failure, append(slices.Clone(decided), steps...)},
	}
	for _, tt := range tests {
		uc, f := newDelete()
		if tt.set != nil {
			tt.set(f)
		}
		err := uc.Execute(as(tt.user), tt.slug)
		if !errors.Is(err, tt.want) || (tt.want == failure && errors.Is(err, domain.ErrNotFound)) {
			t.Errorf("%s: Execute() = %v, want %v", tt.name, err, tt.want)
		}
		if !slices.Equal(f.log.calls, tt.calls) || f.tx.calls != 1 || f.logs.Len() != 0 {
			t.Errorf("%s: calls = %q in %d transactions, log %q; want %q in one, no log", tt.name, f.log.calls, f.tx.calls, f.logs, tt.calls)
		}
	}
	uc, f := newDelete()
	if err := uc.Execute(context.Background(), "acme"); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 || f.tx.calls != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and nothing", err, f.log.calls)
	}
}
````

- [ ] **Step 4: 存储**

`server/internal/modules/workspace/adapter/postgres/workspaces.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/workspaces.go
}

// deref is the value p points at, or the zero value for nil.
````

````new server/internal/modules/workspace/adapter/postgres/workspaces.go
}

// DeleteWorkspace soft-deletes the workspace row id, by the account by at
// now.
func (s *Store) DeleteWorkspace(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).DeleteWorkspace(ctx, gen.DeleteWorkspaceParams{ID: id, DeletedBy: by, Now: now}); err != nil {
		return fmt.Errorf("delete workspace: %w", err)
	}
	return nil
}

// DeleteWorkspaceMembers soft-deletes the undeleted memberships of the
// workspace, active or not, by the account by at now.
func (s *Store) DeleteWorkspaceMembers(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).DeleteWorkspaceMembers(ctx, gen.DeleteWorkspaceMembersParams{WorkspaceID: workspaceID, DeletedBy: by, Now: now})
	if err != nil {
		return fmt.Errorf("delete workspace members: %w", err)
	}
	return nil
}

// deref is the value p points at, or the zero value for nil.
````

`server/internal/modules/workspace/adapter/postgres/preferences.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/postgres/preferences.go
	"fmt"
````

````new server/internal/modules/workspace/adapter/postgres/preferences.go
	"fmt"
	"time"
````

````old server/internal/modules/workspace/adapter/postgres/preferences.go
}

// UpsertPreferences applies r.Patch to the account's undeleted row, or
````

````new server/internal/modules/workspace/adapter/postgres/preferences.go
}

// DeleteWorkspacePreferences soft-deletes the undeleted display settings of
// the workspace's members, by the account by at now.
func (s *Store) DeleteWorkspacePreferences(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).DeleteWorkspacePreferences(ctx, gen.DeleteWorkspacePreferencesParams{WorkspaceID: workspaceID, DeletedBy: by, Now: now})
	if err != nil {
		return fmt.Errorf("delete workspace preferences: %w", err)
	}
	return nil
}

// UpsertPreferences applies r.Patch to the account's undeleted row, or
````

`server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go`（新文件，178 行）：

````file server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// deletion is a row's audit columns after a deletion, as the tests read
// them.
type deletion struct {
	deletedAt *time.Time
	updatedAt time.Time
	updatedBy *uuid.UUID
}

// deletions reads the audit columns of the rows sql selects (id first),
// keyed by id.
func deletions(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) map[uuid.UUID]deletion {
	t.Helper()
	rows, err := pool.Query(context.Background(), sql, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[uuid.UUID]deletion{}
	for rows.Next() {
		var id uuid.UUID
		var d deletion
		if err := rows.Scan(&id, &d.deletedAt, &d.updatedAt, &d.updatedBy); err != nil {
			t.Fatal(err)
		}
		out[id] = d
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// deletedAtBy reports whether d was deleted at when by by.
func (d deletion) deletedAtBy(when time.Time, by uuid.UUID) bool {
	return d.deletedAt != nil && d.deletedAt.Equal(when) && d.updatedAt.Equal(when) && d.updatedBy != nil && *d.updatedBy == by
}

// The three steps, in one transaction, soft-delete the workspace, every
// membership of it, active or not, and every member's settings in it, at
// the same moment and by the same account; a membership and a settings row
// deleted before keep their time, and another workspace keeps everything.
// The slug is free again.
func TestDeletingAWorkspaceSoftDeletesItsRows(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	dave := newAccount(t, pool, "dave@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	join(t, s, acme.ID, bob, shared.RoleMember)
	join(t, s, acme.ID, carol, shared.RoleGuest)
	join(t, s, acme.ID, dave, shared.RoleMember)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE member_id = $1", carol)
	join(t, s, beta.ID, bob, shared.RoleMember)
	for _, r := range []app.PreferencesRow{
		{ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: alice, Now: now}, {ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: bob, Now: now},
		{ID: uuid.NewV7(), WorkspaceID: acme.ID, UserID: carol, Now: now}, {ID: uuid.NewV7(), WorkspaceID: beta.ID, UserID: alice, Now: now},
	} {
		upsert(t, s, r)
	}
	earlier := now.Add(-time.Hour)
	exec(t, pool, "UPDATE workspace_user_properties SET deleted_at = $1 WHERE user_id = $2", earlier, carol)
	exec(t, pool, "UPDATE workspace_members SET deleted_at = $1 WHERE member_id = $2", earlier, dave)
	later := now.Add(time.Hour)

	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		for _, step := range []func(ctx context.Context, id, by uuid.UUID, now time.Time) error{
			s.DeleteWorkspace, s.DeleteWorkspaceMembers, s.DeleteWorkspacePreferences,
		} {
			if err := step(ctx, acme.ID, bob, later); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for what, rows := range map[string]map[uuid.UUID]deletion{
		"acme": deletions(t, pool, "SELECT id, deleted_at, updated_at, updated_by_id FROM workspaces WHERE id = $1", acme.ID),
		"members": deletions(t, pool, `SELECT id, deleted_at, updated_at, updated_by_id FROM workspace_members
			WHERE workspace_id = $1 AND member_id <> $2`, acme.ID, dave),
		"settings": deletions(t, pool, `SELECT id, deleted_at, updated_at, updated_by_id FROM workspace_user_properties
			WHERE workspace_id = $1 AND user_id <> $2`, acme.ID, carol),
	} {
		want := map[string]int{"acme": 1, "members": 3, "settings": 2}[what]
		if len(rows) != want {
			t.Errorf("%s: %d rows, want %d", what, len(rows), want)
		}
		for id, d := range rows {
			if !d.deletedAtBy(later, bob) {
				t.Errorf("%s %s: %+v, want deleted at %v by bob", what, id, d, later)
			}
		}
	}
	var active []bool
	if err := pool.QueryRow(context.Background(), `SELECT array_agg(is_active ORDER BY is_active) FROM workspace_members
		WHERE workspace_id = $1 AND member_id <> $2`, acme.ID, dave).Scan(&active); err != nil || len(active) != 3 || active[0] || !active[1] || !active[2] {
		t.Errorf("is_active of acme's members: %v, %v; want false, true, true as they were", active, err)
	}
	var carolDeleted, daveDeleted time.Time
	if err := pool.QueryRow(context.Background(), `SELECT (SELECT deleted_at FROM workspace_user_properties WHERE user_id = $1),
		(SELECT deleted_at FROM workspace_members WHERE member_id = $2)`, carol, dave).
		Scan(&carolDeleted, &daveDeleted); err != nil || !carolDeleted.Equal(earlier) || !daveDeleted.Equal(earlier) {
		t.Errorf("carol's settings deleted at %v, dave's membership at %v, %v; want %v, as before", carolDeleted, daveDeleted, err, earlier)
	}
	for id, d := range deletions(t, pool, `
		SELECT id, deleted_at, updated_at, updated_by_id FROM workspaces WHERE id = $1
		UNION ALL SELECT id, deleted_at, updated_at, updated_by_id FROM workspace_members WHERE workspace_id = $1
		UNION ALL SELECT id, deleted_at, updated_at, updated_by_id FROM workspace_user_properties WHERE workspace_id = $1`, beta.ID) {
		if d.deletedAt != nil || !d.updatedAt.Equal(now) {
			t.Errorf("beta's row %s: %+v, want it untouched", id, d)
		}
	}
	if _, err := s.WorkspaceBySlug(context.Background(), "acme"); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("WorkspaceBySlug(acme) after the deletion: %v, want app.ErrNotFound", err)
	}
	newWorkspace(t, s, "Acme again", "acme", bob)
}

// failingSettings is the store with its last deletion step failing.
type failingSettings struct {
	*postgresadapter.Store
}

var errDiskFull = errors.New("disk full")

func (failingSettings) DeleteWorkspacePreferences(context.Context, uuid.UUID, uuid.UUID, time.Time) error {
	return errDiskFull
}

// allowAll allows every action, as the workspace's admin.
type allowAll struct{}

func (allowAll) Authorize(context.Context, shared.Actor, shared.Action, shared.Target) (shared.Grant, error) {
	return shared.Grant{WorkspaceRole: shared.RoleAdmin}, nil
}

// The use case's cascade is one transaction on the database: when its last
// step fails, the workspace and its members are not deleted either.
func TestAFailedDeletionLeavesTheWorkspace(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	join(t, s, acme.ID, bob, shared.RoleMember)
	uc := app.NewDeleteWorkspace(failingSettings{s}, allowAll{}, postgres.NewTxManager(pool, 2*time.Second), clocktest.At(now),
		slog.New(slog.DiscardHandler))

	err := uc.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: alice}), "acme")

	if !errors.Is(err, errDiskFull) {
		t.Fatalf("Execute() = %v, want %v", err, errDiskFull)
	}
	if w, err := s.WorkspaceBySlug(context.Background(), "acme"); err != nil || w.TotalMembers != 2 {
		t.Errorf("acme after the failure: %+v, %v; want it there with its 2 members", w, err)
	}
	if got := deletions(t, pool, "SELECT id, deleted_at, updated_at, updated_by_id FROM workspace_members WHERE deleted_at IS NOT NULL"); len(got) != 0 {
		t.Errorf("deleted memberships after the failure: %+v, want none", got)
	}
}
````

`server/internal/modules/workspace/adapter/postgres/failures_test.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/postgres/failures_test.go
	"testing"
````

````new server/internal/modules/workspace/adapter/postgres/failures_test.go
	"testing"
	"time"
````

````old server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("UpsertPreferences() = %+v, %v; want context.Canceled", got, err)
	}
````

````new server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("UpsertPreferences() = %+v, %v; want context.Canceled", got, err)
	}
	for name, write := range map[string]func(context.Context, uuid.UUID, uuid.UUID, time.Time) error{
		"DeleteWorkspace": s.DeleteWorkspace, "DeleteWorkspaceMembers": s.DeleteWorkspaceMembers,
		"DeleteWorkspacePreferences": s.DeleteWorkspacePreferences,
	} {
		if err := write(cancelled, w.ID, alice, now); !failed(err) {
			t.Errorf("%s() = %v; want context.Canceled", name, err)
		}
	}
````

- [ ] **Step 5: HTTP 一侧和接线**

`server/internal/modules/workspace/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/http/handler.go
}

// GetPreferencesUseCase is app.GetWorkspacePreferences.
````

````new server/internal/modules/workspace/adapter/http/handler.go
}

// DeleteWorkspaceUseCase is app.DeleteWorkspace.
type DeleteWorkspaceUseCase interface {
	Execute(ctx context.Context, slug string) error
}

// GetPreferencesUseCase is app.GetWorkspacePreferences.
````

````old server/internal/modules/workspace/adapter/http/handler.go
	UpdateWorkspace   UpdateWorkspaceUseCase
````

````new server/internal/modules/workspace/adapter/http/handler.go
	UpdateWorkspace   UpdateWorkspaceUseCase
	DeleteWorkspace   DeleteWorkspaceUseCase
````

`server/internal/modules/workspace/adapter/http/workspaces.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/http/workspaces.go
}

// CheckWorkspaceSlug serves GET /api/v0/workspace-slugs/{slug}.
````

````new server/internal/modules/workspace/adapter/http/workspaces.go
}

// DeleteWorkspace serves DELETE /api/v0/workspaces/{slug}.
func (h handler) DeleteWorkspace(ctx context.Context, req gen.DeleteWorkspaceRequestObject) (gen.DeleteWorkspaceResponseObject, error) {
	if err := h.uc.DeleteWorkspace.Execute(ctx, req.Slug); err != nil {
		return nil, err
	}
	return gen.DeleteWorkspace204Response{}, nil
}

// CheckWorkspaceSlug serves GET /api/v0/workspace-slugs/{slug}.
````

`server/internal/modules/workspace/adapter/http/handler_test.go`（修改，4 处）：

````old server/internal/modules/workspace/adapter/http/handler_test.go
	update *fakeUpdate
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
	update *fakeUpdate
	del    *fakeDelete
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
	f.got = append(f.got, p)
	return f.answer, f.err
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
	f.got = append(f.got, p)
	return f.answer, f.err
}

type fakeDelete struct {
	calls []string // "caller slug"
	err   error
}

func (f *fakeDelete) Execute(ctx context.Context, slug string) error {
	f.calls = append(f.calls, caller(ctx)+" "+slug)
	return f.err
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
		f.update = &fakeUpdate{}
	}
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
		f.update = &fakeUpdate{}
	}
	if f.del == nil {
		f.del = &fakeDelete{}
	}
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
		ListWorkspaces: f.list, CreateWorkspace: f.create, GetWorkspace: f.get, UpdateWorkspace: f.update, CheckSlug: f.check,
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
		ListWorkspaces: f.list, CreateWorkspace: f.create, GetWorkspace: f.get, UpdateWorkspace: f.update, DeleteWorkspace: f.del, CheckSlug: f.check,
````

`server/internal/modules/workspace/adapter/http/workspaces_test.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/http/workspaces_test.go
}

// The availability, and the reason when the slug is not available; the slug
````

````new server/internal/modules/workspace/adapter/http/workspaces_test.go
}

// DELETE asks the use case for the caller and the slug of the path, and
// answers 204 without a body; its refusals as the contract declares them.
func TestDeleteWorkspace(t *testing.T) {
	del := &fakeDelete{}
	h := newServer(t, fakes{del: del})
	for _, tt := range []struct{ token, path string }{{"alice", "/api/v0/workspaces/acme"}, {"bob", "/api/v0/workspaces/beta"}} {
		if res, body := do(t, h, request(http.MethodDelete, tt.path, tt.token, "")); res.StatusCode != http.StatusNoContent || body != "" {
			t.Errorf("%s DELETE %s = %d %q, want 204 and no body", tt.token, tt.path, res.StatusCode, body)
		}
	}
	if want := []string{"alice acme", "bob beta"}; !slices.Equal(del.calls, want) {
		t.Errorf("calls = %q, want %q", del.calls, want)
	}
	for _, tt := range []struct {
		err    error
		status int
		want   string
	}{
		{domain.ErrNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`},
		{shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
	} {
		h := newServer(t, fakes{del: &fakeDelete{err: tt.err}})
		if res, body := do(t, h, request(http.MethodDelete, "/api/v0/workspaces/acme", "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("DELETE refused with %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// The availability, and the reason when the slug is not available; the slug
````

`server/internal/modules/workspace/module.go`（修改，2 处）：

````old server/internal/modules/workspace/module.go
// workspaces and their members. It brings creating, listing, reading and
// changing workspaces, checking a slug, and each member's display settings,
// and offers the other modules its reads through ports.
````

````new server/internal/modules/workspace/module.go
// workspaces and their members. It brings creating, listing, reading,
// changing and deleting workspaces, checking a slug, and each member's
// display settings, and offers the other modules its reads through ports.
````

````old server/internal/modules/workspace/module.go
		UpdateWorkspace:   app.NewUpdateWorkspace(store, d.Authorizer, d.Tx, d.Clock),
````

````new server/internal/modules/workspace/module.go
		UpdateWorkspace:   app.NewUpdateWorkspace(store, d.Authorizer, d.Tx, d.Clock),
		DeleteWorkspace:   app.NewDeleteWorkspace(store, d.Authorizer, d.Tx, d.Clock, d.Logger),
````

- [ ] **Step 6: 矩阵**

`server/internal/bootstrap/permission_matrix_test.go`（修改，6 处）：

````old server/internal/bootstrap/permission_matrix_test.go
// memberships through the workspace store, and the two states no store
// writes yet through SQL (prepareMatrix). The cells that only read share
// one copy of it, and each cell that writes gets a copy of its own
// (pgtest.NewDatabaseFrom), so no cell sees another's writes. Each module's
// rows are in a file of their own (permission_matrix_<module>_test.go): a
// phase that adds an operation adds its row there, and what the row needs
// prepared here.
````

````new server/internal/bootstrap/permission_matrix_test.go
// memberships through the workspace store, the deleted workspace through
// the API, and the state no store writes yet through SQL (prepareMatrix).
// The cells that only read share one copy of it, and each cell that writes
// gets a copy of its own (pgtest.NewDatabaseFrom), so no cell sees
// another's writes. Each module's rows are in a file of their own
// (permission_matrix_<module>_test.go): a phase that adds an operation adds
// its row there, and what the row needs prepared here.
````

````old server/internal/bootstrap/permission_matrix_test.go
	cellCreated   = cell{status: http.StatusCreated}
````

````new server/internal/bootstrap/permission_matrix_test.go
	cellCreated   = cell{status: http.StatusCreated}
	cellNoContent = cell{status: http.StatusNoContent}
````

````old server/internal/bootstrap/permission_matrix_test.go
// the wrong workspace lets either into acme. Through SQL, until the stores
// of P5 and P2 replace it: the removed member's membership of acme ended,
// and gone's workspace row alone soft-deleted. Everything that connected to
// the database is closed when it returns, so that it can be copied.
````

````new server/internal/bootstrap/permission_matrix_test.go
// the wrong workspace lets either into acme. Through the API, gone deleted
// by its admin, which soft-deletes its memberships with it. Through SQL,
// until P5's store replaces it, the removed member's membership of acme
// ended. Everything that connected to the database is closed when it
// returns, so that it can be copied.
````

````old server/internal/bootstrap/permission_matrix_test.go
		// No store removes a member (P5) or deletes a workspace (P2) yet, so
		// SQL stands in until those phases replace it. The first statement
		// ends the removed member's membership of acme. The second
		// soft-deletes the workspace row of gone alone, which leaves its
		// admin's membership active in a deleted workspace; P2's delete
		// also soft-deletes the memberships, invitations and preferences,
		// so replacing it changes the state this column is asked about.
````

````new server/internal/bootstrap/permission_matrix_test.go
		// No store removes a member yet (P5), so SQL stands in until that
		// phase replaces it: it ends the removed member's membership of acme.
````

````old server/internal/bootstrap/permission_matrix_test.go
		seed.exec(pool, "UPDATE workspaces SET deleted_at = now() WHERE id = $1", gone)
````

````new server/internal/bootstrap/permission_matrix_test.go
		// The column's caller deletes gone as deleteWorkspace does it: its
		// membership goes with the workspace row, so every cell of the column
		// is asked about a workspace deleted the one way there is. A
		// membership left active in a deleted workspace is ActiveRole's
		// store test (P1).
		if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/workspaces/gone", d.tokens[callerDeleted], ""); status != http.StatusNoContent {
			t.Fatalf("deleting gone = %d %s", status, body)
		}
````

````old server/internal/bootstrap/permission_matrix_test.go
// matrixSeed writes the prepared workspaces and memberships through the
// workspace store; exec runs the SQL that stands in for the stores P2 and
// P5 add.
````

````new server/internal/bootstrap/permission_matrix_test.go
// matrixSeed writes the prepared workspaces, memberships and settings
// through the workspace store; exec runs the SQL that stands in for the
// store P5 adds.
````

`server/internal/bootstrap/permission_matrix_workspace_test.go`（修改，1 处）：

````old server/internal/bootstrap/permission_matrix_workspace_test.go
			cells: inWorkspace(cellOK, cellForbidden, cellForbidden), check: renamesIt},
````

````new server/internal/bootstrap/permission_matrix_workspace_test.go
			cells: inWorkspace(cellOK, cellForbidden, cellForbidden), check: renamesIt},
		{op: "deleteWorkspace", write: true, request: toWorkspace(http.MethodDelete, "", ""),
			cells: inWorkspace(cellNoContent, cellForbidden, cellForbidden)},
````

- [ ] **Step 7: 测试、检查**

Run: `go -C server test -count=1 ./internal/modules/workspace/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestPermissionMatrix|TestThePermissionMatrixCoversEveryOperation|TestEveryActionHasARuleAndEveryRuleAnAction' ./internal/bootstrap/`
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

- [ ] **Step 8: 提交**

```bash
git add api/modules/workspace.yaml server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/permission_matrix_workspace_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/workspace/adapter/http/handler.go server/internal/modules/workspace/adapter/http/handler_test.go server/internal/modules/workspace/adapter/http/workspaces.go server/internal/modules/workspace/adapter/http/workspaces_test.go server/internal/modules/workspace/adapter/postgres/delete_workspace_test.go server/internal/modules/workspace/adapter/postgres/failures_test.go server/internal/modules/workspace/adapter/postgres/preferences.go server/internal/modules/workspace/adapter/postgres/queries/members.sql server/internal/modules/workspace/adapter/postgres/queries/preferences.sql server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql server/internal/modules/workspace/adapter/postgres/workspaces.go server/internal/modules/workspace/app/delete_workspace.go server/internal/modules/workspace/app/delete_workspace_test.go server/internal/modules/workspace/app/fakes_test.go server/internal/modules/workspace/app/ports.go server/internal/modules/workspace/domain/actions.go server/internal/modules/workspace/module.go api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/server.gen.go server/internal/modules/workspace/adapter/postgres/gen/members.sql.go server/internal/modules/workspace/adapter/postgres/gen/preferences.sql.go server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P2): deleteWorkspace and its cascade

DELETE /api/v0/workspaces/{slug}, for the workspace's admins: under the
workspace row's lock, the workspace, its memberships and their display
settings are soft-deleted in one transaction at one moment; the slug is
free at once. The matrix's deleted workspace is now deleted this way.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**：

| 改坏 | 必须失败的测试 |
|---|---|
| 连带挪到事务之后、各步各自提交 | `TestDeleteWorkspaceLocksThenDecidesThenCascades`、`TestAFailedDeletionLeavesTheWorkspace` |
| `cascade()` 少了成员一步 | `TestDeleteWorkspaceLocksThenDecidesThenCascades` |
| `cascade()` 少了显示设置一步 | `TestDeleteWorkspaceLocksThenDecidesThenCascades`、`TestAFailedDeletionLeavesTheWorkspace` |
| `DeleteWorkspaceMembers` 不看 `workspace_id` | `TestDeletingAWorkspaceSoftDeletesItsRows`（`beta` 的行被删） |
| `DeleteWorkspacePreferences` 不看 `workspace_id` | `TestDeletingAWorkspaceSoftDeletesItsRows` |
| `DeleteWorkspaceMembers` 不看 `deleted_at IS NULL` | `TestDeletingAWorkspaceSoftDeletesItsRows`（dave 的时间被改） |
| 在事务里、提交之前多记一行日志 | `TestDeleteWorkspaceLocksThenDecidesThenCascades` |
| 三步中任何一步吞掉错误 | `TestAFailedWriteIsAnError` |
| 矩阵的 `gone` 不删除 | `TestPermissionMatrix/…/workspace_deleted`（全部 10 行的这一列） |

**Done when:** 删除在一个事务里、同一时刻连带成员和显示设置；另一个工作区不受影响；矩阵的 6 格通过，"工作区已删除"一列经删除准备。

---

### Task 10: `listWorkspaceMembers` 与 `MemberProfiles`

**Files:**
- Create: `server/internal/modules/identity/adapter/postgres/public_profiles_test.go`、`server/internal/modules/workspace/adapter/http/members.go`、`server/internal/modules/workspace/adapter/http/members_test.go`、`server/internal/modules/workspace/adapter/postgres/list_members_test.go`、`server/internal/modules/workspace/app/list_members.go`、`server/internal/modules/workspace/app/list_members_test.go`、`server/internal/modules/workspace/domain/member.go`、`server/internal/modules/workspace/domain/member_test.go`
- Modify: `api/modules/workspace.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/app.go`、`server/internal/bootstrap/permission_matrix_workspace_test.go`、`server/internal/bootstrap/ports.go`、`server/internal/bootstrap/ports_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/identity/adapter/postgres/accounts.go`、`server/internal/modules/identity/adapter/postgres/queries/users.sql`、`server/internal/modules/identity/app/accounts.go`、`server/internal/modules/identity/provide.go`、`server/internal/modules/workspace/adapter/http/handler.go`、`server/internal/modules/workspace/adapter/http/handler_test.go`、`server/internal/modules/workspace/adapter/postgres/failures_test.go`、`server/internal/modules/workspace/adapter/postgres/queries/members.sql`、`server/internal/modules/workspace/adapter/postgres/workspaces.go`、`server/internal/modules/workspace/app/fakes_test.go`、`server/internal/modules/workspace/app/ports.go`、`server/internal/modules/workspace/domain/actions.go`、`server/internal/modules/workspace/module.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/identity/adapter/postgres/gen/users.sql.go`、`server/internal/modules/workspace/adapter/http/gen/server.gen.go`、`server/internal/modules/workspace/adapter/postgres/gen/members.sql.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.11，M3 设计 3.4、5.1、5.2、6.5、6.6）：
  - `identity`：查询 `PublicProfiles`（`WHERE id = ANY($1)`，不加锁，不看 `is_active`，按 id）；`identityapp.PublicProfile{ID, Email, FirstName, LastName, DisplayName}`；`identity.Provided.PublicProfiles`（`identity.Provide` 交出）；
  - `workspace`：`domain.Membership{ID, WorkspaceID, MemberID, Role, IsActive, CreatedAt}`、`domain.MemberUser{ID, DisplayName, FirstName, LastName, Email *string}`、`domain.Member{Membership; User}`、`domain.SeesEmails(role)`（管理员、成员；按集合比较）；`app.PublicProfile`、`app.MemberProfiles{PublicProfiles(ctx, ids)}`、`app.MemberLister{WorkspaceFinder; ListMembers}`；查询 `ListMembers`（未删除的，含已结束的，`ORDER BY created_at, id`）；`workspace.PublicProfile`（`app.PublicProfile` 的别名）、`workspace.Deps.Profiles`；
  - `app.NewListWorkspaceMembers(members, profiles, auth)`：不开事务：`WorkspaceBySlug` → `Authorize(workspace_member.list)` → `ListMembers` → 一次 `PublicProfiles`（列出的成员）；邮箱按调用者的角色，调用者自己的也一样；成员没有账户是错误（外键保证它不发生）；
  - 接口描述：`listWorkspaceMembers`（`GET /api/v0/workspaces/{slug}/members`，`WorkspaceMemberList{data}`，码 `[workspace.not_found]`）；`MemberUser`（`avatar_url` 在 M5 之前总是 `null`，`email` 必有、可为 `null`）；`WorkspaceMember{id, workspace_id, role, is_active, created_at, member}`；
  - `bootstrap/ports.go` 的 `workspaceProfiles`：`identity.PublicProfiles` 转成 `workspace.MemberProfiles`；`app.go` 接上 `Profiles`；
  - 规则表 `workspace_member.list`：任何有效成员。
- 使用者：Task 12 的答案（改角色之后的成员）；P9 的成员页、个人主页。

**Tests:**
- `identity/adapter/postgres/public_profiles_test.go`：`TestPublicProfilesReadsTheAccountsAskedFor`（停用的也在；不存在的 id 略过；重复的 id 一次；没有 id 时没有）；`TestPublicProfilesDoesNotWaitForTheRowsLock`（停用持着账户行的 `FOR NO KEY UPDATE` 时立即读到提交过的值，约定一）；`TestAFailedProfilesReadIsAnError`（已取消的 ctx：`context.Canceled`，没有资料）。
- `bootstrap/ports_test.go`：`TestWorkspaceProfilesConvertsIdentitysAnswer`（参数原样交给 `identity`，每个字段转换，顺序是 `identity` 的；错误原样、没有资料）。
- `workspace/domain/member_test.go`：`TestSeesEmails`（三个角色，外加 0、10、25）。
- `app/list_members_test.go`：`TestListWorkspaceMembersShowsAddressesByRole`（存储的顺序；已结束的、停用账户的都在；资料一次读完，参数是列出的成员；不开事务；管理员、成员看到每个邮箱，访客一个也看不到，自己的也看不到）；`TestListWorkspaceMembersRefusals`（不存在、看不到：`workspace.not_found`，不列；列、读资料失败原样返回；成员没有账户是错误，不是部分列表）。
- `adapter/postgres/list_members_test.go`：`TestListMembers`（按开始的时间、再按 id；含已结束的；已删除的行、另一个工作区的行都不在）。
- `failures_test.go`：读的测试加 `ListMembers`。
- `adapter/http/members_test.go`：`TestListWorkspaceMembers`（一个邮箱、一个 `null`、`avatar_url` 是 `null`、已结束的照原样）。
- 矩阵：`listWorkspaceMembers` 一行（管理员、成员、访客 200，另三列 `workspace.not_found`）；`listsTheMembers` 核对 `acme` 的四个成员关系（被移出的 `is_active: false`），管理员和成员看到邮箱，访客都是 `null`。

- [ ] **Step 1: `identity` 交出 `PublicProfiles`**

`server/internal/modules/identity/adapter/postgres/queries/users.sql`（修改，1 处）：

````old server/internal/modules/identity/adapter/postgres/queries/users.sql
WHERE email = sqlc.arg(email)
FOR SHARE;

````

````new server/internal/modules/identity/adapter/postgres/queries/users.sql
WHERE email = sqlc.arg(email)
FOR SHARE;

-- name: PublicProfiles :many
-- MemberProfiles (M3 design 6.5): the public profile of each account of ids, deactivated ones too, by id. No lock: a
-- transaction that holds a workspace's lock reads an address this way (3.6 convention 1).
SELECT id, email, first_name, last_name, display_name
FROM users
WHERE id = ANY (sqlc.arg(ids)::uuid[])
ORDER BY id;

````

`server/internal/modules/identity/app/accounts.go`（修改，1 处）：

````old server/internal/modules/identity/app/accounts.go
import "uuid"
````

````new server/internal/modules/identity/app/accounts.go
import "uuid"

// PublicProfile is an account's public profile as another module reads it
// through identity's MemberProfiles, without a lock and whatever the
// account's state (M3 design 6.5).
type PublicProfile struct {
	ID          uuid.UUID
	Email       string
	FirstName   string
	LastName    string
	DisplayName string
}
````

`server/internal/modules/identity/adapter/postgres/accounts.go`（修改，1 处）：

````old server/internal/modules/identity/adapter/postgres/accounts.go
}

func accountState(id uuid.UUID, email string, active bool, err error) (app.AccountState, bool, error) {
````

````new server/internal/modules/identity/adapter/postgres/accounts.go
}

// PublicProfiles returns the public profile of each account of ids that
// exists, deactivated ones too, by id, without a lock (M3 design 6.5).
func (s *Store) PublicProfiles(ctx context.Context, ids []uuid.UUID) ([]app.PublicProfile, error) {
	rows, err := s.queries(ctx).PublicProfiles(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("read public profiles: %w", err)
	}
	out := make([]app.PublicProfile, len(rows))
	for i, r := range rows {
		out[i] = app.PublicProfile{ID: r.ID, Email: r.Email, FirstName: r.FirstName, LastName: r.LastName, DisplayName: r.DisplayName}
	}
	return out, nil
}

func accountState(id uuid.UUID, email string, active bool, err error) (app.AccountState, bool, error) {
````

`server/internal/modules/identity/provide.go`（修改，3 处）：

````old server/internal/modules/identity/provide.go
}

// Provided are the adapters identity offers the other modules. They depend
````

````new server/internal/modules/identity/provide.go
}

// PublicProfiles reads the public profile of each account of ids that
// exists, deactivated ones too, by id, without a lock (M3 design 6.5): the
// workspace module's MemberProfiles. A transaction that holds a lock of a
// later table reads an account's address this way and never through
// Accounts (M3 design 3.6 convention 1).
type PublicProfiles interface {
	PublicProfiles(ctx context.Context, ids []uuid.UUID) ([]app.PublicProfile, error)
}

// Provided are the adapters identity offers the other modules. They depend
````

````old server/internal/modules/identity/provide.go
	Accounts Accounts
````

````new server/internal/modules/identity/provide.go
	Accounts       Accounts
	PublicProfiles PublicProfiles
````

````old server/internal/modules/identity/provide.go
	return Provided{Accounts: postgresadapter.New(pool)}
````

````new server/internal/modules/identity/provide.go
	store := postgresadapter.New(pool)
	return Provided{Accounts: store, PublicProfiles: store}
````

`server/internal/modules/identity/adapter/postgres/public_profiles_test.go`（新文件，79 行）：

````file server/internal/modules/identity/adapter/postgres/public_profiles_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// PublicProfiles answers the accounts of ids that exist, by id: a
// deactivated one too, an unknown id left out, a repeated id once, and
// nothing for no ids.
func TestPublicProfilesReadsTheAccountsAskedFor(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newUser("alice@corp.com"), newUser("bob@corp.com"), newUser("carol@corp.com")
	for _, u := range []app.NewUser{alice, bob, carol} {
		mustCreate(t, s, u)
	}
	exec(t, pool, "UPDATE users SET first_name = 'Alice', last_name = 'Liddell', display_name = 'al' WHERE id = $1", alice.ID)
	exec(t, pool, "UPDATE users SET is_active = false WHERE id = $1", bob.ID)

	got, err := s.PublicProfiles(context.Background(), []uuid.UUID{bob.ID, uuid.NewV7(), alice.ID, bob.ID})

	want := []app.PublicProfile{
		{ID: alice.ID, Email: "alice@corp.com", FirstName: "Alice", LastName: "Liddell", DisplayName: "al"},
		{ID: bob.ID, Email: "bob@corp.com", DisplayName: bob.DisplayName},
	}
	slices.SortFunc(want, func(a, b app.PublicProfile) int { return slices.Compare(a.ID[:], b.ID[:]) })
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("PublicProfiles() = %+v, %v; want %+v", got, err, want)
	}
	if got, err := s.PublicProfiles(context.Background(), nil); err != nil || len(got) != 0 {
		t.Errorf("PublicProfiles(nil) = %+v, %v; want none", got, err)
	}
}

// A read of the profiles that fails answers its error, never no profiles,
// which the member list would take for a member without an account.
func TestAFailedProfilesReadIsAnError(t *testing.T) {
	s, _ := newStore(t)
	alice := newUser("alice@corp.com")
	mustCreate(t, s, alice)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := s.PublicProfiles(cancelled, []uuid.UUID{alice.ID}); !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("PublicProfiles() = %+v, %v; want context.Canceled and no profile", got, err)
	}
}

// PublicProfiles takes no lock: it reads an account whose row a
// deactivation holds FOR NO KEY UPDATE without waiting, as it was
// committed (M3 design 3.6 convention 1).
func TestPublicProfilesDoesNotWaitForTheRowsLock(t *testing.T) {
	s, pool := newStore(t)
	alice := newUser("alice@corp.com")
	mustCreate(t, s, alice)
	end := hold(t, postgres.NewTxManager(pool, 5*time.Second), func(ctx context.Context) error {
		_, err := postgres.DB(ctx, pool).Exec(ctx, "SELECT 1 FROM users WHERE id = $1 FOR NO KEY UPDATE", alice.ID)
		if err != nil {
			return err
		}
		_, err = postgres.DB(ctx, pool).Exec(ctx, "UPDATE users SET display_name = 'changing' WHERE id = $1", alice.ID)
		return err
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got, err := s.PublicProfiles(ctx, []uuid.UUID{alice.ID})
	if err != nil || len(got) != 1 || got[0].DisplayName != alice.DisplayName {
		t.Errorf("PublicProfiles() under the lock = %+v, %v; want alice as committed, at once", got, err)
	}
	if err := end(); err != nil {
		t.Fatal(err)
	}
}
````

- [ ] **Step 2: 接口描述、领域、规则表**

`api/modules/workspace.yaml`（修改，2 处）：

````old api/modules/workspace.yaml
          description: The workspace is deleted.
````

````new api/modules/workspace.yaml
          description: The workspace is deleted.
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspaces/{slug}/members:
    parameters:
      - $ref: '#/components/parameters/Slug'
    get:
      operationId: listWorkspaceMembers
      tags: [workspace]
      summary: List a workspace's members
      description: >-
        Every membership of the workspace, those that ended too (is_active
        false), in the order they began, then by id, each with the member's
        public profile. For any active member. The addresses are shown to
        admins and members; to a guest every address is null, his own too.
        A workspace that does not exist, is deleted, or of which the caller
        is not an active member answers workspace.not_found. The whole
        collection at once: collections are not paginated.
      security: [{bearer: []}]
      x-problem-codes: [workspace.not_found]
      responses:
        '200':
          description: The workspace's memberships.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/WorkspaceMemberList'
````

````old api/modules/workspace.yaml
          description: An IANA time zone name, e.g. from GET /api/v0/timezones; UTC when not given.
          type: string
````

````new api/modules/workspace.yaml
          description: An IANA time zone name, e.g. from GET /api/v0/timezones; UTC when not given.
          type: string
    MemberUser:
      description: >-
        A member's public profile, embedded in the membership: the one way
        v0 shows other accounts (M3 design 5.2).
      type: object
      additionalProperties: false
      required: [id, display_name, first_name, last_name, avatar_url, email]
      properties:
        id:
          type: string
          format: uuid
        display_name:
          type: string
        first_name:
          type: string
        last_name:
          type: string
        avatar_url:
          description: Null until uploads arrive (M5).
          type: [string, 'null']
        email:
          description: The member's address for a caller who is an admin or a member; null for a guest.
          type: [string, 'null']
    WorkspaceMember:
      type: object
      additionalProperties: false
      required: [id, workspace_id, role, is_active, created_at, member]
      properties:
        id:
          description: The membership's id, which /workspace-members/{workspace_member_id} names.
          type: string
          format: uuid
        workspace_id:
          type: string
          format: uuid
        role:
          $ref: '#/components/schemas/WorkspaceRole'
        is_active:
          description: False once the membership has ended.
          type: boolean
        created_at:
          type: string
          format: date-time
        member:
          $ref: '#/components/schemas/MemberUser'
    WorkspaceMemberList:
      type: object
      additionalProperties: false
      required: [data]
      properties:
        data:
          type: array
          items:
            $ref: '#/components/schemas/WorkspaceMember'
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspaces~1{slug}'
````

````new api/openapi.yaml
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspaces~1{slug}'
  /api/v0/workspaces/{slug}/members:
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspaces~1{slug}~1members'
````

`server/internal/modules/workspace/domain/member.go`（新文件，46 行）：

````file server/internal/modules/workspace/domain/member.go
package domain

import (
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Membership is a row of workspace_members: an account's membership of a
// workspace, active or ended (M3 design 5.2).
type Membership struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	MemberID    uuid.UUID
	Role        shared.Role
	IsActive    bool
	CreatedAt   time.Time
}

// MemberUser is a member's public profile as another member sees it (M3
// design 5.2): Email is nil for a caller whose role may not see it.
type MemberUser struct {
	ID          uuid.UUID
	DisplayName string
	FirstName   string
	LastName    string
	Email       *string
}

// Member is a membership with its member's profile: WorkspaceMember.
type Member struct {
	Membership
	User MemberUser
}

// emailReaders are the workspace roles that see the members' addresses:
// admins and members, not guests (Plane views/workspace/member.py:50-54,
// M3 design 3.4). Roles are compared by set, never by order.
var emailReaders = []shared.Role{shared.RoleAdmin, shared.RoleMember}

// SeesEmails reports whether a caller of role sees the members' addresses.
func SeesEmails(role shared.Role) bool {
	return slices.Contains(emailReaders, role)
}
````

`server/internal/modules/workspace/domain/member_test.go`（新文件，19 行）：

````file server/internal/modules/workspace/domain/member_test.go
package domain

import (
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Admins and members see the addresses; guests, and a role outside the
// three, do not.
func TestSeesEmails(t *testing.T) {
	for role, want := range map[shared.Role]bool{
		shared.RoleAdmin: true, shared.RoleMember: true, shared.RoleGuest: false, 0: false, 10: false, 25: false,
	} {
		if got := SeesEmails(role); got != want {
			t.Errorf("SeesEmails(%d) = %v, want %v", role, got, want)
		}
	}
}
````

`server/internal/modules/workspace/domain/actions.go`（修改，2 处）：

````old server/internal/modules/workspace/domain/actions.go
	ActionDelete shared.Action = "workspace.delete"
````

````new server/internal/modules/workspace/domain/actions.go
	ActionDelete shared.Action = "workspace.delete"
	// ActionMemberList is listing a workspace's members:
	// listWorkspaceMembers.
	ActionMemberList shared.Action = "workspace_member.list"
````

````old server/internal/modules/workspace/domain/actions.go
	return []shared.Action{ActionRead, ActionUpdate, ActionDelete, ActionPreferencesRead, ActionPreferencesUpdate}
````

````new server/internal/modules/workspace/domain/actions.go
	return []shared.Action{ActionRead, ActionUpdate, ActionDelete, ActionMemberList, ActionPreferencesRead, ActionPreferencesUpdate}
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"workspace.read":   {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	"workspace.update": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	"workspace.delete": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
````

````new server/internal/modules/access/domain/rules.go
	"workspace.read":        {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	"workspace.update":      {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	"workspace.delete":      {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	"workspace_member.list": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"workspace.delete":             {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
````

````new server/internal/modules/access/domain/rules_test.go
	"workspace.delete":             {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
	"workspace_member.list":        {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
````

- [ ] **Step 3: 端口、查询、用例、存储**

`server/internal/modules/workspace/app/ports.go`（修改，2 处）：

````old server/internal/modules/workspace/app/ports.go
}

// WorkspaceCreator inserts a workspace and its first member.
````

````new server/internal/modules/workspace/app/ports.go
}

// PublicProfile is an account's public profile as MemberProfiles reads it:
// identity's value, converted in bootstrap/ports.go (M3 design 6.5).
type PublicProfile struct {
	ID          uuid.UUID
	Email       string
	FirstName   string
	LastName    string
	DisplayName string
}

// MemberProfiles reads the public profile of each account of ids that
// exists, deactivated ones too, without a lock (M3 design 6.5): the way a
// use case reads another account, also inside a transaction that holds a
// workspace's lock (M3 design 3.6 convention 1). identity implements it
// (identity.Provide).
type MemberProfiles interface {
	PublicProfiles(ctx context.Context, ids []uuid.UUID) ([]PublicProfile, error)
}

// WorkspaceCreator inserts a workspace and its first member.
````

````old server/internal/modules/workspace/app/ports.go
// there is none, also when it was deleted while the lock waited.
````

````new server/internal/modules/workspace/app/ports.go
// there is none, also when it was deleted while the lock waited.

// MemberLister reads a workspace and lists its memberships.
type MemberLister interface {
	WorkspaceFinder
	// ListMembers returns the undeleted memberships of the workspace, active
	// or not, by created_at, then id (M3 design 3.12).
	ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]domain.Membership, error)
}
````

`server/internal/modules/workspace/adapter/postgres/queries/members.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/members.sql
        sqlc.arg(created_by), sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now));
````

````new server/internal/modules/workspace/adapter/postgres/queries/members.sql
        sqlc.arg(created_by), sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now));

-- name: ListMembers :many
-- listWorkspaceMembers: every undeleted membership, active or not, by the time it began (M3 design 3.12).
SELECT id, workspace_id, member_id, role, is_active, created_at
FROM workspace_members
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL
ORDER BY created_at, id;
````

`server/internal/modules/workspace/app/list_members.go`（新文件，81 行）：

````file server/internal/modules/workspace/app/list_members.go
package app

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListWorkspaceMembers lists a workspace's members: GET
// /api/v0/workspaces/{slug}/members.
type ListWorkspaceMembers struct {
	members  MemberLister
	profiles MemberProfiles
	auth     shared.Authorizer
}

// NewListWorkspaceMembers returns the use case.
func NewListWorkspaceMembers(members MemberLister, profiles MemberProfiles, auth shared.Authorizer) *ListWorkspaceMembers {
	return &ListWorkspaceMembers{members: members, profiles: profiles, auth: auth}
}

// Execute returns every undeleted membership of the workspace, ended ones
// too, each with its member's public profile, deactivated accounts' too (M3
// design 5.1). The addresses are there for a caller whose role sees them
// (domain.SeesEmails), the caller's own included. A read opens no
// transaction and decides directly (M3 design 3.4).
func (u *ListWorkspaceMembers) Execute(ctx context.Context, slug string) ([]domain.Member, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	w, err := u.members.WorkspaceBySlug(ctx, slug)
	switch {
	case errors.Is(err, ErrNotFound):
		return nil, domain.ErrNotFound
	case err != nil:
		return nil, err
	}
	grant, err := u.auth.Authorize(ctx, actor, domain.ActionMemberList, shared.Target{WorkspaceID: w.ID})
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return nil, domain.ErrNotFound
	case err != nil:
		return nil, err
	}
	memberships, err := u.members.ListMembers(ctx, w.ID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(memberships))
	for i, m := range memberships {
		ids[i] = m.MemberID
	}
	profiles, err := u.profiles.PublicProfiles(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]PublicProfile, len(profiles))
	for _, p := range profiles {
		byID[p.ID] = p
	}
	seesEmails := domain.SeesEmails(grant.WorkspaceRole)
	out := make([]domain.Member, len(memberships))
	for i, m := range memberships {
		p, ok := byID[m.MemberID]
		if !ok {
			// The foreign key keeps every member's account: its absence is a bug.
			return nil, fmt.Errorf("workspace member %s: no account %s", m.ID, m.MemberID)
		}
		user := domain.MemberUser{ID: p.ID, DisplayName: p.DisplayName, FirstName: p.FirstName, LastName: p.LastName}
		if seesEmails {
			user.Email = &p.Email
		}
		out[i] = domain.Member{Membership: m, User: user}
	}
	return out, nil
}
````

`server/internal/modules/workspace/app/fakes_test.go`（修改，2 处）：

````old server/internal/modules/workspace/app/fakes_test.go
	log        *callLog
	workspaces []domain.Workspace // by slug for WorkspaceBySlug, SlugTaken and the locks; by id for UpdateWorkspace
	lists      map[uuid.UUID][]domain.Workspace
	createErr  error
	memberErr  error               // for CreateMember, which then stores nothing
	updateErr  error               // for UpdateWorkspace
	deleteErrs map[string]error    // by step, e.g. "DeleteWorkspaceMembers"
	listErrs   map[uuid.UUID]error // by user, for ListWorkspaces
	slugErrs   map[string]error    // by slug, for WorkspaceBySlug and SlugTaken
	lockErrs   map[string]error    // by slug, for the locks
	members    []app.MemberRow
	prefs      map[prefsKey]domain.Preferences
	prefsErr   error // for Preferences and UpsertPreferences
	upserts    []app.PreferencesRow
````

````new server/internal/modules/workspace/app/fakes_test.go
	log         *callLog
	workspaces  []domain.Workspace // by slug for WorkspaceBySlug, SlugTaken and the locks; by id for UpdateWorkspace
	lists       map[uuid.UUID][]domain.Workspace
	createErr   error
	memberErr   error               // for CreateMember, which then stores nothing
	updateErr   error               // for UpdateWorkspace
	deleteErrs  map[string]error    // by step, e.g. "DeleteWorkspaceMembers"
	listErrs    map[uuid.UUID]error // by user, for ListWorkspaces
	slugErrs    map[string]error    // by slug, for WorkspaceBySlug and SlugTaken
	lockErrs    map[string]error    // by slug, for the locks
	members     []app.MemberRow
	memberships map[uuid.UUID][]domain.Membership // by workspace, for ListMembers
	membersErr  error                             // for ListMembers
	prefs       map[prefsKey]domain.Preferences
	prefsErr    error // for Preferences and UpsertPreferences
	upserts     []app.PreferencesRow
````

````old server/internal/modules/workspace/app/fakes_test.go
}

// show is *s quoted, or <nil>.
````

````new server/internal/modules/workspace/app/fakes_test.go
}

func (f *fakeWorkspaces) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]domain.Membership, error) {
	f.log.add(ctx, "ListMembers %s", workspaceID)
	if f.membersErr != nil {
		return nil, fmt.Errorf("list workspace members: %w", f.membersErr)
	}
	return f.memberships[workspaceID], nil
}

// fakeProfiles answers the profiles it holds of the ids asked for, in the
// order it holds them, and logs each call with its ids.
type fakeProfiles struct {
	log      *callLog
	profiles []app.PublicProfile
	err      error
}

func (f *fakeProfiles) PublicProfiles(ctx context.Context, ids []uuid.UUID) ([]app.PublicProfile, error) {
	f.log.add(ctx, "PublicProfiles %v", ids)
	if f.err != nil {
		return nil, fmt.Errorf("read public profiles: %w", f.err)
	}
	var out []app.PublicProfile
	for _, p := range f.profiles {
		if slices.Contains(ids, p.ID) {
			out = append(out, p)
		}
	}
	return out, nil
}

// show is *s quoted, or <nil>.
````

`server/internal/modules/workspace/app/list_members_test.go`（新文件，157 行）：

````file server/internal/modules/workspace/app/list_members_test.go
package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// membersFixture is ListWorkspaceMembers over fakes sharing one log. acme's
// memberships: alice (admin), bob (member), carol's ended one (guest, her
// account deactivated); beta's: bob alone. The Authorizer gives alice admin
// in acme, bob member in acme and guest in beta.
type membersFixture struct {
	log        *callLog
	workspaces *fakeWorkspaces
	profiles   *fakeProfiles
	auth       *fakeAuthorizer
	uc         *app.ListWorkspaceMembers
}

var (
	aliceInAcme = domain.Membership{ID: uuid.NewV7(), WorkspaceID: acme.ID, MemberID: alice.ID, Role: shared.RoleAdmin, IsActive: true, CreatedAt: now}
	bobInAcme   = domain.Membership{ID: uuid.NewV7(), WorkspaceID: acme.ID, MemberID: bob.ID, Role: shared.RoleMember, IsActive: true,
		CreatedAt: now.Add(time.Minute)}
	carolInAcme = domain.Membership{ID: uuid.NewV7(), WorkspaceID: acme.ID, MemberID: carol.ID, Role: shared.RoleGuest, CreatedAt: now.Add(time.Hour)}
	bobInBeta   = domain.Membership{ID: uuid.NewV7(), WorkspaceID: beta.ID, MemberID: bob.ID, Role: shared.RoleGuest, IsActive: true, CreatedAt: now}
	profiles    = []app.PublicProfile{
		{ID: carol.ID, Email: carol.Email, DisplayName: "carol"},
		{ID: alice.ID, Email: alice.Email, FirstName: "Alice", LastName: "Liddell", DisplayName: "al"},
		{ID: bob.ID, Email: bob.Email, DisplayName: "bob"},
	}
)

func newMembers() *membersFixture {
	log := &callLog{}
	f := &membersFixture{log: log,
		workspaces: &fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme, beta}, memberships: map[uuid.UUID][]domain.Membership{
			acme.ID: {aliceInAcme, bobInAcme, carolInAcme}, beta.ID: {bobInBeta},
		}},
		profiles: &fakeProfiles{log: log, profiles: profiles},
		auth: &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{
			{alice.ID, acme.ID}: {WorkspaceRole: shared.RoleAdmin},
			{bob.ID, acme.ID}:   {WorkspaceRole: shared.RoleMember},
			{bob.ID, beta.ID}:   {WorkspaceRole: shared.RoleGuest},
		}}}
	f.uc = app.NewListWorkspaceMembers(f.workspaces, f.profiles, f.auth)
	return f
}

// withUser is m with its member's profile from profiles, the address only
// when seen.
func withUser(m domain.Membership, seen bool) domain.Member {
	i := slices.IndexFunc(profiles, func(p app.PublicProfile) bool { return p.ID == m.MemberID })
	p := profiles[i]
	user := domain.MemberUser{ID: p.ID, DisplayName: p.DisplayName, FirstName: p.FirstName, LastName: p.LastName}
	if seen {
		user.Email = &p.Email
	}
	return domain.Member{Membership: m, User: user}
}

// sameMembers compares two lists, the addresses by value.
func sameMembers(a, b []domain.Member) bool {
	return slices.EqualFunc(a, b, func(x, y domain.Member) bool {
		ex, ey := x.User.Email, y.User.Email
		x.User.Email, y.User.Email = nil, nil
		return x == y && (ex == nil) == (ey == nil) && (ex == nil || *ex == *ey)
	})
}

// The list is the workspace's memberships in the store's order, the ended
// one too, each with its member's profile, a deactivated account's too; the
// profiles are read once, for the listed members, without a transaction.
// An admin and a member see every address, a guest none, his own neither.
func TestListWorkspaceMembersShowsAddressesByRole(t *testing.T) {
	acmeIDs := fmt.Sprint([]uuid.UUID{alice.ID, bob.ID, carol.ID})
	tests := []struct {
		name string
		user app.AccountState
		w    domain.Workspace
		ids  string
		want []domain.Member
	}{
		{"acme's admin", alice, acme, acmeIDs, []domain.Member{withUser(aliceInAcme, true), withUser(bobInAcme, true), withUser(carolInAcme, true)}},
		{"acme's member", bob, acme, acmeIDs, []domain.Member{withUser(aliceInAcme, true), withUser(bobInAcme, true), withUser(carolInAcme, true)}},
		{"beta's guest", bob, beta, fmt.Sprint([]uuid.UUID{bob.ID}), []domain.Member{withUser(bobInBeta, false)}},
	}
	for _, tt := range tests {
		f := newMembers()
		got, err := f.uc.Execute(as(tt.user), tt.w.Slug)
		if err != nil || !sameMembers(got, tt.want) {
			t.Errorf("%s: Execute() = %+v, %v; want %+v", tt.name, got, err, tt.want)
		}
		want := []string{
			"WorkspaceBySlug " + tt.w.Slug + " outside tx",
			"Authorize " + tt.user.ID.String() + " workspace_member.list on " + tt.w.ID.String() + "/" + uuid.Nil().String() + " outside tx",
			"ListMembers " + tt.w.ID.String() + " outside tx",
			"PublicProfiles " + tt.ids + " outside tx",
		}
		if !slices.Equal(f.log.calls, want) {
			t.Errorf("%s: calls = %q, want %q", tt.name, f.log.calls, want)
		}
	}
}

// A workspace that is not there and one the caller cannot see are the same
// workspace.not_found, and nothing is listed; a failure is the answer,
// never a partial list, and so is a member without an account.
func TestListWorkspaceMembersRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	tests := []struct {
		name  string
		user  app.AccountState
		slug  string
		set   func(f *membersFixture)
		want  error
		calls int
	}{
		{"no such workspace", alice, "nothing", nil, domain.ErrNotFound, 1},
		{"not visible", carol, "acme", nil, domain.ErrNotFound, 2},
		{"alice cannot see beta", alice, "beta", nil, domain.ErrNotFound, 2},
		{"forbidden", alice, "acme", func(f *membersFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: shared.Forbidden()} },
			shared.Forbidden(), 2},
		{"the Authorizer failed", alice, "acme", func(f *membersFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: failure} }, failure, 2},
		{"the list failed", alice, "acme", func(f *membersFixture) { f.workspaces.membersErr = failure }, failure, 3},
		{"the profiles failed", alice, "acme", func(f *membersFixture) { f.profiles.err = failure }, failure, 4},
	}
	for _, tt := range tests {
		f := newMembers()
		if tt.set != nil {
			tt.set(f)
		}
		if got, err := f.uc.Execute(as(tt.user), tt.slug); !errors.Is(err, tt.want) || got != nil {
			t.Errorf("%s: Execute() = %+v, %v; want no list and %v", tt.name, got, err, tt.want)
		}
		if len(f.log.calls) != tt.calls {
			t.Errorf("%s: calls = %q, want %d", tt.name, f.log.calls, tt.calls)
		}
	}
	f := newMembers()
	f.profiles.profiles = profiles[1:]
	if got, err := f.uc.Execute(as(alice), "acme"); err == nil || errors.Is(err, domain.ErrNotFound) || got != nil {
		t.Errorf("a member without an account: Execute() = %+v, %v; want an error that is not a 404", got, err)
	}
	f = newMembers()
	if _, err := f.uc.Execute(context.Background(), "acme"); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and no call", err, f.log.calls)
	}
}
````

`server/internal/modules/workspace/adapter/postgres/workspaces.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/workspaces.go
}

// DeleteWorkspaceMembers soft-deletes the undeleted memberships of the
````

````new server/internal/modules/workspace/adapter/postgres/workspaces.go
}

// ListMembers returns the undeleted memberships of the workspace, active or
// not, by created_at, then id.
func (s *Store) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]domain.Membership, error) {
	rows, err := s.queries(ctx).ListMembers(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace members: %w", err)
	}
	out := make([]domain.Membership, len(rows))
	for i, r := range rows {
		out[i] = domain.Membership{ID: r.ID, WorkspaceID: r.WorkspaceID, MemberID: r.MemberID, Role: shared.Role(r.Role),
			IsActive: r.IsActive, CreatedAt: r.CreatedAt}
	}
	return out, nil
}

// DeleteWorkspaceMembers soft-deletes the undeleted memberships of the
````

`server/internal/modules/workspace/adapter/postgres/list_members_test.go`（新文件，64 行）：

````file server/internal/modules/workspace/adapter/postgres/list_members_test.go
package postgresadapter_test

import (
	"context"
	"slices"
	"testing"
	"time"
	"uuid"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// joinAt makes user a member of workspace with role at the time at, and
// returns the membership as stored.
func joinAt(t *testing.T, s *postgresadapter.Store, workspace, user uuid.UUID, role shared.Role, at time.Time) domain.Membership {
	t.Helper()
	m := app.MemberRow{ID: uuid.NewV7(), WorkspaceID: workspace, MemberID: user, Role: role, CreatedBy: user, Now: at}
	if err := s.CreateMember(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	return domain.Membership{ID: m.ID, WorkspaceID: workspace, MemberID: user, Role: role, IsActive: true, CreatedAt: at}
}

// ListMembers is the workspace's undeleted memberships, the ended ones too,
// by the time they began, then by id; not a deleted row, not another
// workspace's.
func TestListMembers(t *testing.T) {
	s, pool := newStore(t)
	var ids []uuid.UUID
	for _, email := range []string{"alice@corp.com", "bob@corp.com", "carol@corp.com", "dave@corp.com", "eve@corp.com"} {
		ids = append(ids, newAccount(t, pool, email))
	}
	alice, bob, carol, dave, eve := ids[0], ids[1], ids[2], ids[3], ids[4]
	acme, err := s.CreateWorkspace(context.Background(), app.WorkspaceRow{ID: uuid.NewV7(), Name: "Acme", Slug: "acme", Timezone: "UTC", CreatedBy: alice, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	beta := newWorkspace(t, s, "Beta", "beta", bob)
	carolIn := joinAt(t, s, acme.ID, carol, shared.RoleGuest, now.Add(time.Minute))
	bobIn := joinAt(t, s, acme.ID, bob, shared.RoleMember, now.Add(2*time.Minute))
	aliceIn := joinAt(t, s, acme.ID, alice, shared.RoleAdmin, now)
	eveIn := joinAt(t, s, acme.ID, eve, shared.RoleMember, now)
	joinAt(t, s, acme.ID, dave, shared.RoleMember, now)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE member_id = $1", carol)
	exec(t, pool, "UPDATE workspace_members SET deleted_at = $2 WHERE member_id = $1", dave, now)
	joinAt(t, s, beta.ID, alice, shared.RoleGuest, now)

	got, err := s.ListMembers(context.Background(), acme.ID)

	carolIn.IsActive = false
	first, second := aliceIn, eveIn
	if slices.Compare(eveIn.ID[:], aliceIn.ID[:]) < 0 {
		first, second = eveIn, aliceIn
	}
	if want := []domain.Membership{first, second, carolIn, bobIn}; err != nil || !slices.Equal(got, want) {
		t.Errorf("ListMembers(acme) = %+v, %v; want %+v", got, err, want)
	}
	if got, err := s.ListMembers(context.Background(), uuid.NewV7()); err != nil || len(got) != 0 {
		t.Errorf("ListMembers() of no workspace = %+v, %v; want none", got, err)
	}
}
````

`server/internal/modules/workspace/adapter/postgres/failures_test.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("Preferences() = %+v, %v, %v; want context.Canceled, not no row", p, found, err)
	}
````

````new server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("Preferences() = %+v, %v, %v; want context.Canceled, not no row", p, found, err)
	}
	if list, err := s.ListMembers(cancelled, w.ID); !failed(err) || list != nil {
		t.Errorf("ListMembers() = %v, %v; want context.Canceled, no list", list, err)
	}
````

- [ ] **Step 4: HTTP 一侧、接线、转换**

`server/internal/modules/workspace/adapter/http/handler.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/http/handler.go
}

// GetPreferencesUseCase is app.GetWorkspacePreferences.
````

````new server/internal/modules/workspace/adapter/http/handler.go
}

// ListMembersUseCase is app.ListWorkspaceMembers.
type ListMembersUseCase interface {
	Execute(ctx context.Context, slug string) ([]domain.Member, error)
}

// GetPreferencesUseCase is app.GetWorkspacePreferences.
````

````old server/internal/modules/workspace/adapter/http/handler.go
	DeleteWorkspace   DeleteWorkspaceUseCase
````

````new server/internal/modules/workspace/adapter/http/handler.go
	DeleteWorkspace   DeleteWorkspaceUseCase
	ListMembers       ListMembersUseCase
````

`server/internal/modules/workspace/adapter/http/members.go`（新文件，48 行）：

````file server/internal/modules/workspace/adapter/http/members.go
package httpadapter

import (
	"context"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// ListWorkspaceMembers serves GET /api/v0/workspaces/{slug}/members.
func (h handler) ListWorkspaceMembers(ctx context.Context, req gen.ListWorkspaceMembersRequestObject) (gen.ListWorkspaceMembersResponseObject, error) {
	list, err := h.uc.ListMembers.Execute(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	out := gen.ListWorkspaceMembers200JSONResponse{Data: make([]gen.WorkspaceMember, len(list))}
	for i, m := range list {
		out.Data[i] = member(m)
	}
	return out, nil
}

// member is m as the API shows it.
func member(m domain.Member) gen.WorkspaceMember {
	// Required and nullable: the zero Nullable is "unspecified" and would
	// marshal as "", so null is set explicitly.
	email := nullable.NewNullNullable[string]()
	if m.User.Email != nil {
		email = nullable.NewNullableWithValue(*m.User.Email)
	}
	return gen.WorkspaceMember{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		Role:        gen.WorkspaceRole(m.Role),
		IsActive:    m.IsActive,
		CreatedAt:   m.CreatedAt,
		Member: gen.MemberUser{
			ID:          m.User.ID,
			DisplayName: m.User.DisplayName,
			FirstName:   m.User.FirstName,
			LastName:    m.User.LastName,
			AvatarURL:   nullable.NewNullNullable[string](), // M5
			Email:       email,
		},
	}
}
````

`server/internal/modules/workspace/adapter/http/handler_test.go`（修改，4 处）：

````old server/internal/modules/workspace/adapter/http/handler_test.go
	del    *fakeDelete
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
	del    *fakeDelete
	member *fakeMembers
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
	return f.err
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
	return f.err
}

type fakeMembers struct {
	calls []string // "caller slug"
	lists map[string][]domain.Member
	err   error
}

func (f *fakeMembers) Execute(ctx context.Context, slug string) ([]domain.Member, error) {
	f.calls = append(f.calls, caller(ctx)+" "+slug)
	return f.lists[slug], f.err
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
		f.del = &fakeDelete{}
	}
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
		f.del = &fakeDelete{}
	}
	if f.member == nil {
		f.member = &fakeMembers{}
	}
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
		GetPreferences: fakeGetPrefs{f.prefs}, UpdatePreferences: fakeUpdatePrefs{f.prefs},
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
		ListMembers: f.member, GetPreferences: fakeGetPrefs{f.prefs}, UpdatePreferences: fakeUpdatePrefs{f.prefs},
````

`server/internal/modules/workspace/adapter/http/members_test.go`（新文件，58 行）：

````file server/internal/modules/workspace/adapter/http/members_test.go
package httpadapter_test

import (
	"net/http"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	aliceEmail  = "alice@corp.com"
	aliceMember = domain.Member{
		Membership: domain.Membership{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000a1"), WorkspaceID: acmeID, MemberID: aliceID,
			Role: shared.RoleAdmin, IsActive: true, CreatedAt: created},
		User: domain.MemberUser{ID: aliceID, DisplayName: "al", FirstName: "Alice", LastName: "Liddell", Email: &aliceEmail},
	}
	bobMember = domain.Member{
		Membership: domain.Membership{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000b1"), WorkspaceID: acmeID, MemberID: bobID,
			Role: shared.RoleGuest, CreatedAt: created},
		User: domain.MemberUser{ID: bobID, DisplayName: "bob"},
	}
)

const (
	aliceMemberJSON = `{"created_at":"2026-09-29T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000a1","is_active":true,` +
		`"member":{"avatar_url":null,"display_name":"al","email":"alice@corp.com","first_name":"Alice",` +
		`"id":"0199a2b4-0000-7000-8000-000000000001","last_name":"Liddell"},"role":20,"workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	bobMemberJSON = `{"created_at":"2026-09-29T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000b1","is_active":false,` +
		`"member":{"avatar_url":null,"display_name":"bob","email":null,"first_name":"","id":"0199a2b4-0000-7000-8000-000000000002",` +
		`"last_name":""},"role":5,"workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
)

// The list is the use case's for the caller and the slug of the path: an
// address shown, one null, the avatar null, an ended membership as it is.
func TestListWorkspaceMembers(t *testing.T) {
	members := &fakeMembers{lists: map[string][]domain.Member{"acme": {aliceMember, bobMember}}}
	h := newServer(t, fakes{member: members})
	for _, tt := range []struct{ token, slug, want string }{
		{"alice", "acme", `{"data":[` + aliceMemberJSON + `,` + bobMemberJSON + `]}`},
		{"bob", "beta", `{"data":[]}`},
	} {
		res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/"+tt.slug+"/members", tt.token, ""))
		if res.StatusCode != http.StatusOK || body != tt.want+"\n" {
			t.Errorf("%s GET %s's members = %d %s, want 200 %s", tt.token, tt.slug, res.StatusCode, body, tt.want)
		}
	}
	if want := []string{"alice acme", "bob beta"}; !slices.Equal(members.calls, want) {
		t.Errorf("calls = %q, want %q", members.calls, want)
	}
	h = newServer(t, fakes{member: &fakeMembers{err: domain.ErrNotFound}})
	res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/acme/members", "bob", ""))
	if want := `{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`; res.StatusCode != http.StatusNotFound || body != want+"\n" {
		t.Errorf("GET refused = %d %s, want 404 %s", res.StatusCode, body, want)
	}
}
````

`server/internal/modules/workspace/module.go`（修改，4 处）：

````old server/internal/modules/workspace/module.go
// changing and deleting workspaces, checking a slug, and each member's
// display settings, and offers the other modules its reads through ports.
````

````new server/internal/modules/workspace/module.go
// changing and deleting workspaces, checking a slug, listing the members,
// and each member's display settings, and offers the other modules its
// reads through ports.
````

````old server/internal/modules/workspace/module.go
type AccountState = app.AccountState

````

````new server/internal/modules/workspace/module.go
type AccountState = app.AccountState

// PublicProfile is the profile the MemberProfiles port hands over: bootstrap
// converts identity's into it (M3 design 6.5).
type PublicProfile = app.PublicProfile

````

````old server/internal/modules/workspace/module.go
	Accounts app.Accounts
````

````new server/internal/modules/workspace/module.go
	Accounts app.Accounts
	// Profiles is identity's PublicProfiles, converted (bootstrap/ports.go).
	Profiles app.MemberProfiles
````

````old server/internal/modules/workspace/module.go
		DeleteWorkspace:   app.NewDeleteWorkspace(store, d.Authorizer, d.Tx, d.Clock, d.Logger),
````

````new server/internal/modules/workspace/module.go
		DeleteWorkspace:   app.NewDeleteWorkspace(store, d.Authorizer, d.Tx, d.Clock, d.Logger),
		ListMembers:       app.NewListWorkspaceMembers(store, d.Profiles, d.Authorizer),
````

`server/internal/bootstrap/ports.go`（修改，1 处）：

````old server/internal/bootstrap/ports.go
	state, found, err := a.accounts.ShareAccountByEmail(ctx, email)
	return workspace.AccountState(state), found, err
}

````

````new server/internal/bootstrap/ports.go
	state, found, err := a.accounts.ShareAccountByEmail(ctx, email)
	return workspace.AccountState(state), found, err
}

// workspaceProfiles is identity's PublicProfiles as workspace's
// MemberProfiles: the same read, each profile converted.
type workspaceProfiles struct {
	profiles identity.PublicProfiles
}

func (p workspaceProfiles) PublicProfiles(ctx context.Context, ids []uuid.UUID) ([]workspace.PublicProfile, error) {
	profiles, err := p.profiles.PublicProfiles(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]workspace.PublicProfile, len(profiles))
	for i, profile := range profiles {
		out[i] = workspace.PublicProfile(profile)
	}
	return out, nil
}

````

`server/internal/bootstrap/ports_test.go`（修改，1 处）：

````old server/internal/bootstrap/ports_test.go
	}
}

````

````new server/internal/bootstrap/ports_test.go
	}
}

// fakeIdentityProfiles answers the profiles it holds of the ids asked for,
// in the order it holds them, and records the ids.
type fakeIdentityProfiles struct {
	profiles []identityapp.PublicProfile
	err      error
	asked    [][]uuid.UUID
}

func (f *fakeIdentityProfiles) PublicProfiles(_ context.Context, ids []uuid.UUID) ([]identityapp.PublicProfile, error) {
	f.asked = append(f.asked, ids)
	var out []identityapp.PublicProfile
	for _, p := range f.profiles {
		if slices.Contains(ids, p.ID) {
			out = append(out, p)
		}
	}
	return out, f.err
}

// workspaceProfiles asks identity for the ids workspace asks for and hands
// over each profile, every field converted, in identity's order; an error
// as it came, without profiles.
func TestWorkspaceProfilesConvertsIdentitysAnswer(t *testing.T) {
	alice := identityapp.PublicProfile{ID: uuid.NewV7(), Email: "alice@corp.com", FirstName: "Alice", LastName: "Liddell", DisplayName: "al"}
	bob := identityapp.PublicProfile{ID: uuid.NewV7(), Email: "bob@corp.com", DisplayName: "bob"}
	fake := &fakeIdentityProfiles{profiles: []identityapp.PublicProfile{alice, bob}}
	p := workspaceProfiles{profiles: fake}
	ids := []uuid.UUID{bob.ID, uuid.NewV7(), alice.ID}

	got, err := p.PublicProfiles(context.Background(), ids)

	want := []workspace.PublicProfile{
		{ID: alice.ID, Email: "alice@corp.com", FirstName: "Alice", LastName: "Liddell", DisplayName: "al"},
		{ID: bob.ID, Email: "bob@corp.com", DisplayName: "bob"},
	}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("PublicProfiles() = %+v, %v; want %+v", got, err, want)
	}
	if len(fake.asked) != 1 || !slices.Equal(fake.asked[0], ids) {
		t.Errorf("identity was asked %v, want %v", fake.asked, ids)
	}
	failure := errors.New("connection reset")
	fake.err = failure
	if got, err := p.PublicProfiles(context.Background(), ids); !errors.Is(err, failure) || got != nil {
		t.Errorf("PublicProfiles() = %+v, %v; want no profiles and %v", got, err, failure)
	}
}

````

`server/internal/bootstrap/app.go`（修改，1 处）：

````old server/internal/bootstrap/app.go
		Accounts:        workspaceAccounts{accounts: identityPorts.Accounts},
````

````new server/internal/bootstrap/app.go
		Accounts:        workspaceAccounts{accounts: identityPorts.Accounts},
		Profiles:        workspaceProfiles{profiles: identityPorts.PublicProfiles},
````

- [ ] **Step 5: 矩阵**

`server/internal/bootstrap/permission_matrix_workspace_test.go`（修改，3 处）：

````old server/internal/bootstrap/permission_matrix_workspace_test.go
import (
````

````new server/internal/bootstrap/permission_matrix_workspace_test.go
import (
	"fmt"
````

````old server/internal/bootstrap/permission_matrix_workspace_test.go
			cells: inWorkspace(cellNoContent, cellForbidden, cellForbidden)},
````

````new server/internal/bootstrap/permission_matrix_workspace_test.go
			cells: inWorkspace(cellNoContent, cellForbidden, cellForbidden)},
		{op: "listWorkspaceMembers", request: toWorkspace(http.MethodGet, "/members", ""), cells: inWorkspace(cellOK, cellOK, cellOK),
			check: listsTheMembers},
````

````old server/internal/bootstrap/permission_matrix_workspace_test.go
			cells: inWorkspace(cellOK, cellOK, cellOK), check: preferencesAre(navigation{"TABBED", 5}, navigation{"ACCORDION", 5})},
````

````new server/internal/bootstrap/permission_matrix_workspace_test.go
			cells: inWorkspace(cellOK, cellOK, cellOK), check: preferencesAre(navigation{"TABBED", 5}, navigation{"ACCORDION", 5})},
	}
}

// listsTheMembers: acme's four memberships, the removed member's ended; the
// admin and the member see every address, the guest none, his own neither
// (M3 design 3.4, 9.2).
func listsTheMembers(t *testing.T, c caller, answer string) {
	var list struct {
		Data []struct {
			IsActive bool `json:"is_active"`
			Member   struct {
				DisplayName string  `json:"display_name"`
				Email       *string `json:"email"`
			} `json:"member"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	var got, want []string
	for _, m := range list.Data {
		email := "null"
		if m.Member.Email != nil {
			email = *m.Member.Email
		}
		got = append(got, fmt.Sprintf("%s %s active %v", m.Member.DisplayName, email, m.IsActive))
	}
	for _, name := range []string{"admin", "member", "guest", "removed"} {
		email := name + "@example.com"
		if c == callerGuest {
			email = "null"
		}
		want = append(want, fmt.Sprintf("%s %s active %v", name, email, name != "removed"))
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("%s sees the members %q, want %q", c, got, want)
````

- [ ] **Step 6: 生成、测试、检查**

Step 1–3 写入了两个查询和接口描述，生成之前 Go 代码编译不过（`gen` 包还没有新方法）。

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `b7ba0fd4bf56d0a289796fd2a17c45212ad5cd5585bce516f386624ba46e28f0` | 1308 | `api/dist/openapi.yaml` |
| `affa236712b8bd1a0b12f8f0f672585db78ab25c0af6400847d14be1e0fda65b` | 364 | `server/internal/modules/identity/adapter/postgres/gen/users.sql.go` |
| `277b0091d14b50aa542b27f8dfc77bac703432a6dc176b8d1741e57457e39e56` | 1383 | `server/internal/modules/workspace/adapter/http/gen/server.gen.go` |
| `91546b096a1dd60fb9a59bfe202f86d1975735563567fae01ef3c9d72fdd0e65` | 125 | `server/internal/modules/workspace/adapter/postgres/gen/members.sql.go` |
| `858e8108545de06a2d3b73c4c90023e484655681d13d75b55680af7cce80028a` | 1378 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/identity/adapter/postgres/gen/users.sql.go server/internal/modules/workspace/adapter/http/gen/server.gen.go server/internal/modules/workspace/adapter/postgres/gen/members.sql.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

Run: `go -C server test -count=1 ./internal/modules/workspace/... ./internal/modules/identity/adapter/postgres/ ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestWorkspaceProfilesConvertsIdentitysAnswer|TestPermissionMatrix|TestThePermissionMatrixCoversEveryOperation|TestEveryActionHasARuleAndEveryRuleAnAction' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`（`TestModulesRunSQLOnlyThroughSQLC`、`TestSQLCSchemaScope` 核对成员列表不跨模块读表）。

Run: `make lint-web`
Expected: 通过。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

- [ ] **Step 7: 提交**

```bash
git add api/modules/workspace.yaml api/openapi.yaml server/internal/bootstrap/app.go server/internal/bootstrap/permission_matrix_workspace_test.go server/internal/bootstrap/ports.go server/internal/bootstrap/ports_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/identity/adapter/postgres/accounts.go server/internal/modules/identity/adapter/postgres/public_profiles_test.go server/internal/modules/identity/adapter/postgres/queries/users.sql server/internal/modules/identity/app/accounts.go server/internal/modules/identity/provide.go server/internal/modules/workspace/adapter/http/handler.go server/internal/modules/workspace/adapter/http/handler_test.go server/internal/modules/workspace/adapter/http/members.go server/internal/modules/workspace/adapter/http/members_test.go server/internal/modules/workspace/adapter/postgres/failures_test.go server/internal/modules/workspace/adapter/postgres/list_members_test.go server/internal/modules/workspace/adapter/postgres/queries/members.sql server/internal/modules/workspace/adapter/postgres/workspaces.go server/internal/modules/workspace/app/fakes_test.go server/internal/modules/workspace/app/list_members.go server/internal/modules/workspace/app/list_members_test.go server/internal/modules/workspace/app/ports.go server/internal/modules/workspace/domain/actions.go server/internal/modules/workspace/domain/member.go server/internal/modules/workspace/domain/member_test.go server/internal/modules/workspace/module.go api/dist/openapi.yaml server/internal/modules/identity/adapter/postgres/gen/users.sql.go server/internal/modules/workspace/adapter/http/gen/server.gen.go server/internal/modules/workspace/adapter/postgres/gen/members.sql.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P2): list a workspace's members with their public profiles

GET /api/v0/workspaces/{slug}/members, for any active member: every
membership, the ended ones too, each with the member's profile read
through MemberProfiles, identity's PublicProfiles converted in bootstrap
(no lock, deactivated accounts too). Guests see no address.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**：

| 改坏 | 必须失败的测试 |
|---|---|
| `SeesEmails` 也给访客 | `TestSeesEmails`、`TestListWorkspaceMembersShowsAddressesByRole`、`TestPermissionMatrix/listWorkspaceMembers/guest` |
| `SeesEmails` 按大小比较（`role >= 15`） | `TestSeesEmails`（角色 25） |
| 用例按管理员的角色给邮箱 | `TestListWorkspaceMembersShowsAddressesByRole`、`TestPermissionMatrix/listWorkspaceMembers/guest` |
| `ListMembers` 列出已删除的行 | `TestListMembers` |
| `ListMembers` 只列有效的 | `TestListMembers`、`TestPermissionMatrix/listWorkspaceMembers/admin` |
| `ListMembers` 不看 `workspace_id` | `TestListMembers`、`TestPermissionMatrix/listWorkspaceMembers/admin` |
| `ListMembers` 只按 id 倒序 | `TestListMembers` |
| `PublicProfiles` 只读有效的账户 | `TestPublicProfilesReadsTheAccountsAskedFor` |
| `PublicProfiles` 加 `FOR SHARE` | `TestPublicProfilesDoesNotWaitForTheRowsLock` |
| 转换丢掉邮箱 | `TestWorkspaceProfilesConvertsIdentitysAnswer`、`TestPermissionMatrix/listWorkspaceMembers/admin` |
| HTTP 一侧总答 `null` | `TestListWorkspaceMembers` |
| `workspace_member.list` 去掉访客 | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix/listWorkspaceMembers/guest` |
| `ListMembers` 吞掉错误 | `TestAFailedReadIsAnErrorNotAnAnswer` |
| `PublicProfiles` 吞掉错误 | `TestAFailedProfilesReadIsAnError` |

**Done when:** 成员列表的用例、存储、HTTP、转换和矩阵的 6 格通过；邮箱按角色；成员列表只经 `MemberProfiles` 读账户。

---

### Task 11: 成员关系的存储：按 id 的锁、读、改角色

**Files:**
- Create: `server/internal/modules/workspace/adapter/postgres/update_member_test.go`
- Modify: `server/internal/modules/workspace/adapter/postgres/failures_test.go`、`server/internal/modules/workspace/adapter/postgres/locks.go`、`server/internal/modules/workspace/adapter/postgres/locks_test.go`、`server/internal/modules/workspace/adapter/postgres/queries/members.sql`、`server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql`、`server/internal/modules/workspace/adapter/postgres/workspaces.go`
- Generate: `server/internal/modules/workspace/adapter/postgres/gen/members.sql.go`、`server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go`

**Interfaces:**
- Produces（spec 2.12，M3 设计 3.6 约定二）：
  - 查询 `LockWorkspace`（按 id，`FOR NO KEY UPDATE`，带 `deleted_at IS NULL`）、`MemberByID`（未删除的，有效与否都读）、`UpdateMemberRole`（`RETURNING` 存下的值）；
  - `(*Store).LockWorkspace(ctx, id) error`（没有这一行：`app.ErrNotFound`）、`(*Store).MemberByID(ctx, id) (domain.Membership, error)`（没有：`app.ErrNotFound`；失败是错误）、`(*Store).UpdateMemberRole(ctx, id, role, by, now) (domain.Membership, error)`（没有这一行是错误：调用者在锁下读过它）。
- 使用者：Task 12 的用例（按资源寻址的写：读成员关系得到工作区，锁住工作区，再读）。

**Tests:**（真实数据库）
- `locks_test.go`：锁的表加上第三把 `LockWorkspace`（`named` 带 slug 和 id）；`TestTheWorkspaceLocksConflictAsConvention2Says` 加 5 种组合（同一个工作区上，按 id 的锁与另两把的四种都互相等待；与另一个工作区的锁不等）；另三个锁测试对三把锁各跑一遍。
- `update_member_test.go`：`TestMemberByID`（有效的、已结束的都读到；已删除的、不存在的：`app.ErrNotFound`）；`TestUpdateMemberRole`（只改那一行的角色、`updated_by_id`、`updated_at`，答存下的值；同一个人在另一个工作区的成员关系不变；没有这一行是错误，不是 `app.ErrNotFound`）。
- `failures_test.go`：读的测试加 `MemberByID`，写的测试加 `UpdateMemberRole`。

- [ ] **Step 1: 查询和生成**

`server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql
FOR NO KEY UPDATE;

````

````new server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql
FOR NO KEY UPDATE;

-- name: LockWorkspace :one
-- LockWorkspaceBySlug for a write addressed by a row under the workspace (M3 design 3.6 convention 2): the use case
-- read the row for the workspace's id, and reads it again under this lock.
SELECT id
FROM workspaces
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR NO KEY UPDATE;

````

`server/internal/modules/workspace/adapter/postgres/queries/members.sql`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/queries/members.sql
ORDER BY created_at, id;
````

````new server/internal/modules/workspace/adapter/postgres/queries/members.sql
ORDER BY created_at, id;

-- name: MemberByID :one
-- updateWorkspaceMember reads the membership before the workspace's lock, for the workspace, and again under it.
SELECT id, workspace_id, member_id, role, is_active, created_at
FROM workspace_members
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: UpdateMemberRole :one
-- updateWorkspaceMember, under the workspace's FOR NO KEY UPDATE.
UPDATE workspace_members
SET role = sqlc.arg(role), updated_by_id = sqlc.arg(updated_by), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id)
RETURNING id, workspace_id, member_id, role, is_active, created_at;
````

Run: `make gen-go`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `f0883d084a99d24843aecf86bf3a6546cfdf618c354cdc01de1a26a03cf78f4d` | 198 | `server/internal/modules/workspace/adapter/postgres/gen/members.sql.go` |
| `e55c542e25f5cfaf6bbd7424ada92f593e63c38f4d625008f6ad20cf71b29733` | 298 | `server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go` |

Run: `shasum -a 256 server/internal/modules/workspace/adapter/postgres/gen/members.sql.go server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go`
Expected: 与上表相同。

- [ ] **Step 2: 存储和测试**

`server/internal/modules/workspace/adapter/postgres/locks.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/locks.go
}

// ShareWorkspaceBySlug locks the workspace with slug FOR SHARE: for a write
````

````new server/internal/modules/workspace/adapter/postgres/locks.go
}

// LockWorkspace locks the workspace id FOR NO KEY UPDATE: for a write
// addressed by a row under the workspace, which the use case reads again
// under the lock.
func (s *Store) LockWorkspace(ctx context.Context, id uuid.UUID) error {
	_, err := lockedWorkspace(s.queries(ctx).LockWorkspace(ctx, id))
	return err
}

// ShareWorkspaceBySlug locks the workspace with slug FOR SHARE: for a write
````

`server/internal/modules/workspace/adapter/postgres/workspaces.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/postgres/workspaces.go
}

// DeleteWorkspaceMembers soft-deletes the undeleted memberships of the
````

````new server/internal/modules/workspace/adapter/postgres/workspaces.go
}

// MemberByID returns the undeleted membership id; app.ErrNotFound when
// there is none.
func (s *Store) MemberByID(ctx context.Context, id uuid.UUID) (domain.Membership, error) {
	r, err := s.queries(ctx).MemberByID(ctx, id)
	if err != nil {
		return domain.Membership{}, notFound(err)
	}
	return domain.Membership{ID: r.ID, WorkspaceID: r.WorkspaceID, MemberID: r.MemberID, Role: shared.Role(r.Role),
		IsActive: r.IsActive, CreatedAt: r.CreatedAt}, nil
}

// UpdateMemberRole sets the role of the membership id, by the account by at
// now, and returns the membership as stored. The caller holds the
// workspace's lock and read the row under it: its absence is an error.
func (s *Store) UpdateMemberRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Membership, error) {
	r, err := s.queries(ctx).UpdateMemberRole(ctx, gen.UpdateMemberRoleParams{ID: id, Role: int16(role), UpdatedBy: &by, Now: now})
	if err != nil {
		return domain.Membership{}, fmt.Errorf("update workspace member: %w", err)
	}
	return domain.Membership{ID: r.ID, WorkspaceID: r.WorkspaceID, MemberID: r.MemberID, Role: shared.Role(r.Role),
		IsActive: r.IsActive, CreatedAt: r.CreatedAt}, nil
}

// DeleteWorkspaceMembers soft-deletes the undeleted memberships of the
````

`server/internal/modules/workspace/adapter/postgres/locks_test.go`（修改，18 处）：

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
// lock is one of the store's parent locks of a workspace named by its slug
// (M3 design 3.6 convention 2).
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
// lock is one of the store's parent locks of a workspace (M3 design 3.6
// convention 2), which names it by its slug or by its id: take answers the
// id it locked.
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
	take func(ctx context.Context, s *postgresadapter.Store, slug string) (uuid.UUID, error)
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
	take func(ctx context.Context, s *postgresadapter.Store, w named) (uuid.UUID, error)
}

// named is how a lock names a workspace.
type named struct {
	slug string
	id   uuid.UUID
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
	noKeyUpdate = lock{"LockWorkspaceBySlug", func(ctx context.Context, s *postgresadapter.Store, slug string) (uuid.UUID, error) {
		return s.LockWorkspaceBySlug(ctx, slug)
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
	noKeyUpdate = lock{"LockWorkspaceBySlug", func(ctx context.Context, s *postgresadapter.Store, w named) (uuid.UUID, error) {
		return s.LockWorkspaceBySlug(ctx, w.slug)
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
	forShare = lock{"ShareWorkspaceBySlug", func(ctx context.Context, s *postgresadapter.Store, slug string) (uuid.UUID, error) {
		return s.ShareWorkspaceBySlug(ctx, slug)
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
	forShare = lock{"ShareWorkspaceBySlug", func(ctx context.Context, s *postgresadapter.Store, w named) (uuid.UUID, error) {
		return s.ShareWorkspaceBySlug(ctx, w.slug)
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
	locks = []lock{noKeyUpdate, forShare}
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
	noKeyUpdateByID = lock{"LockWorkspace", func(ctx context.Context, s *postgresadapter.Store, w named) (uuid.UUID, error) {
		if err := s.LockWorkspace(ctx, w.id); err != nil {
			return uuid.UUID{}, err
		}
		return w.id, nil
	}}
	locks = []lock{noKeyUpdate, forShare, noKeyUpdateByID}
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
// The two locks conflict as convention 2 wants: FOR NO KEY UPDATE waits for
// either, FOR SHARE waits for FOR NO KEY UPDATE and not for another FOR
// SHARE, and neither waits for a lock of another workspace. A lock that
// waits ends with lock_not_available under a lock_timeout; one that does not
// answers its workspace's id.
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
// The locks conflict as convention 2 wants: FOR NO KEY UPDATE, by slug or
// by id, waits for any of them, FOR SHARE waits for FOR NO KEY UPDATE and
// not for another FOR SHARE, and none waits for a lock of another
// workspace. A lock that waits ends with lock_not_available under a
// lock_timeout; one that does not answers its workspace's id.
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
		{forShare, forShare, "acme", false},
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
		{forShare, forShare, "acme", false},
		{noKeyUpdate, noKeyUpdateByID, "acme", true},
		{noKeyUpdateByID, noKeyUpdate, "acme", true},
		{forShare, noKeyUpdateByID, "acme", true},
		{noKeyUpdateByID, forShare, "acme", true},
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
		{noKeyUpdate, forShare, "beta", false},
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
		{noKeyUpdate, forShare, "beta", false},
		{noKeyUpdate, noKeyUpdateByID, "beta", false},
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
				_, err := tt.held.take(ctx, s, "acme")
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
				_, err := tt.held.take(ctx, s, named{"acme", ids["acme"]})
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
				got, err = tt.then.take(ctx, s, tt.slug)
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
				got, err = tt.then.take(ctx, s, named{tt.slug, ids[tt.slug]})
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
// A lock finds the undeleted workspace with the slug and answers its id;
// app.ErrNotFound for a deleted workspace, a slug no workspace has, or a
// slug of another case.
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
// A lock finds the undeleted workspace it names and answers its id;
// app.ErrNotFound for a deleted workspace, a slug or an id no workspace has,
// or a slug of another case.
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
		for slug, want := range map[string]uuid.UUID{"acme": acme.ID, "beta": beta.ID, "gone": {}, "nothing": {}, "ACME": {}} {
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
		for _, tt := range []struct {
			w    named
			want uuid.UUID
		}{
			{named{"acme", acme.ID}, acme.ID}, {named{"beta", beta.ID}, beta.ID}, {named{"gone", gone.ID}, uuid.UUID{}},
			{named{"nothing", uuid.NewV7()}, uuid.UUID{}}, {named{"ACME", uuid.NewV7()}, uuid.UUID{}},
		} {
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
				got, err = l.take(ctx, s, slug)
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
				got, err = l.take(ctx, s, tt.w)
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
			if want == (uuid.UUID{}) && (!errors.Is(err, app.ErrNotFound) || got != want) {
				t.Errorf("%s(%q) = %s, %v; want app.ErrNotFound", l.name, slug, got, err)
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
			if tt.want == (uuid.UUID{}) && (!errors.Is(err, app.ErrNotFound) || got != tt.want) {
				t.Errorf("%s(%+v) = %s, %v; want app.ErrNotFound", l.name, tt.w, got, err)
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
			if want != (uuid.UUID{}) && (err != nil || got != want) {
				t.Errorf("%s(%q) = %s, %v; want %s", l.name, slug, got, err, want)
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
			if tt.want != (uuid.UUID{}) && (err != nil || got != tt.want) {
				t.Errorf("%s(%+v) = %s, %v; want %s", l.name, tt.w, got, err, tt.want)
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
					a.id, err = l.take(ctx, s, "acme")
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
					a.id, err = l.take(ctx, s, named{"acme", acme.ID})
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
	newWorkspace(t, s, "Acme", "acme", alice)
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
	acme := newWorkspace(t, s, "Acme", "acme", alice)
````

````old server/internal/modules/workspace/adapter/postgres/locks_test.go
		if id, err := l.take(cancelled, s, "acme"); !errors.Is(err, context.Canceled) || errors.Is(err, app.ErrNotFound) || id != (uuid.UUID{}) {
````

````new server/internal/modules/workspace/adapter/postgres/locks_test.go
		if id, err := l.take(cancelled, s, named{"acme", acme.ID}); !errors.Is(err, context.Canceled) || errors.Is(err, app.ErrNotFound) || id != (uuid.UUID{}) {
````

`server/internal/modules/workspace/adapter/postgres/update_member_test.go`（新文件，70 行）：

````file server/internal/modules/workspace/adapter/postgres/update_member_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// MemberByID reads an undeleted membership, an ended one too;
// app.ErrNotFound for a deleted row and an id no row has.
func TestMemberByID(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	bobIn := joinAt(t, s, acme.ID, bob, shared.RoleMember, now)
	carolIn := joinAt(t, s, acme.ID, carol, shared.RoleGuest, now)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE id = $1", carolIn.ID)
	carolIn.IsActive = false
	gone := joinAt(t, s, newWorkspace(t, s, "Beta", "beta", alice).ID, bob, shared.RoleGuest, now)
	exec(t, pool, "UPDATE workspace_members SET deleted_at = $2 WHERE id = $1", gone.ID, now)

	for _, want := range []domain.Membership{bobIn, carolIn} {
		if got, err := s.MemberByID(context.Background(), want.ID); err != nil || got != want {
			t.Errorf("MemberByID(%s) = %+v, %v; want %+v", want.ID, got, err, want)
		}
	}
	for _, id := range []uuid.UUID{gone.ID, uuid.NewV7()} {
		if got, err := s.MemberByID(context.Background(), id); !errors.Is(err, app.ErrNotFound) || got != (domain.Membership{}) {
			t.Errorf("MemberByID(%s) = %+v, %v; want app.ErrNotFound", id, got, err)
		}
	}
}

// UpdateMemberRole sets the role of that membership only, with the updater
// and the time, and answers it as stored; a missing row is an error, not
// app.ErrNotFound.
func TestUpdateMemberRole(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	bobIn := joinAt(t, s, acme.ID, bob, shared.RoleMember, now)
	bobInBeta := joinAt(t, s, beta.ID, bob, shared.RoleMember, now)
	later := now.Add(time.Hour)

	got, err := s.UpdateMemberRole(context.Background(), bobIn.ID, shared.RoleGuest, alice, later)

	want := bobIn
	want.Role = shared.RoleGuest
	if err != nil || got != want {
		t.Errorf("UpdateMemberRole() = %+v, %v; want %+v", got, err, want)
	}
	var updatedBy uuid.UUID
	var updatedAt time.Time
	if err := pool.QueryRow(context.Background(), "SELECT updated_by_id, updated_at FROM workspace_members WHERE id = $1", bobIn.ID).
		Scan(&updatedBy, &updatedAt); err != nil || updatedBy != alice || !updatedAt.Equal(later) {
		t.Errorf("updated_by_id %s at %v, %v; want alice at %v", updatedBy, updatedAt, err, later)
	}
	if other, err := s.MemberByID(context.Background(), bobInBeta.ID); err != nil || other != bobInBeta {
		t.Errorf("bob in beta: %+v, %v; want it unchanged", other, err)
	}
	if _, err := s.UpdateMemberRole(context.Background(), uuid.NewV7(), shared.RoleGuest, alice, later); err == nil || errors.Is(err, app.ErrNotFound) {
		t.Errorf("UpdateMemberRole() of no row = %v, want an error that is not app.ErrNotFound", err)
	}
}
````

`server/internal/modules/workspace/adapter/postgres/failures_test.go`（修改，3 处）：

````old server/internal/modules/workspace/adapter/postgres/failures_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
````

````new server/internal/modules/workspace/adapter/postgres/failures_test.go
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````

````old server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("ListMembers() = %v, %v; want context.Canceled, no list", list, err)
	}
````

````new server/internal/modules/workspace/adapter/postgres/failures_test.go
		t.Errorf("ListMembers() = %v, %v; want context.Canceled, no list", list, err)
	}
	bob := joinAt(t, s, w.ID, newAccount(t, pool, "bob@corp.com"), shared.RoleMember, now)
	if got, err := s.MemberByID(cancelled, bob.ID); !failed(err) || errors.Is(err, app.ErrNotFound) || got != (domain.Membership{}) {
		t.Errorf("MemberByID() = %+v, %v; want context.Canceled, not app.ErrNotFound", got, err)
	}
````

````old server/internal/modules/workspace/adapter/postgres/failures_test.go
		}
	}
````

````new server/internal/modules/workspace/adapter/postgres/failures_test.go
		}
	}
	bob := joinAt(t, s, w.ID, newAccount(t, pool, "bob@corp.com"), shared.RoleMember, now)
	if got, err := s.UpdateMemberRole(cancelled, bob.ID, shared.RoleGuest, alice, now); !failed(err) || got != (domain.Membership{}) {
		t.Errorf("UpdateMemberRole() = %+v, %v; want context.Canceled", got, err)
	}
````

- [ ] **Step 3: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/adapter/postgres/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/workspace/adapter/postgres/failures_test.go server/internal/modules/workspace/adapter/postgres/locks.go server/internal/modules/workspace/adapter/postgres/locks_test.go server/internal/modules/workspace/adapter/postgres/queries/members.sql server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql server/internal/modules/workspace/adapter/postgres/update_member_test.go server/internal/modules/workspace/adapter/postgres/workspaces.go server/internal/modules/workspace/adapter/postgres/gen/members.sql.go server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go
```
```bash
git commit -m "feat(M3/P2): the membership's store: lock by id, read, change the role

LockWorkspace takes the undeleted workspace FOR NO KEY UPDATE by its id,
for a write addressed by a membership; MemberByID reads a membership,
ended or not; UpdateMemberRole answers the row as stored.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**：

| 改坏 | 必须失败的测试 |
|---|---|
| `LockWorkspace` 去掉 `deleted_at IS NULL` | `TestTheWorkspaceLocksFindOnlyAnUndeletedWorkspace`、`TestTheWorkspaceLocksSkipAWorkspaceDeletedWhileTheyWait/LockWorkspace` |
| `LockWorkspace` 换成 `FOR SHARE` | `TestTheWorkspaceLocksConflictAsConvention2Says`（Task 13 起另有 `TestTwoAdminsDemotingEachOtherLeaveAnAdmin`） |
| `MemberByID` 把失败答成 `app.ErrNotFound` | `TestAFailedReadIsAnErrorNotAnAnswer` |
| `UpdateMemberRole` 吞掉错误 | `TestAFailedWriteIsAnError`、`TestUpdateMemberRole` |

**Done when:** 三把锁的 11 种组合、两个存储测试在真实数据库上通过；生成物的 SHA-256 与表相同。

---

### Task 12: `updateWorkspaceMember`；矩阵的请求指名准备好的行

**Files:**
- Create: `server/internal/modules/workspace/app/update_member.go`、`server/internal/modules/workspace/app/update_member_test.go`
- Modify: `api/modules/workspace.yaml`、`api/openapi.yaml`、`server/internal/bootstrap/permission_matrix_coverage_test.go`、`server/internal/bootstrap/permission_matrix_test.go`、`server/internal/bootstrap/permission_matrix_workspace_test.go`、`server/internal/bootstrap/workspace_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/workspace/adapter/http/handler.go`、`server/internal/modules/workspace/adapter/http/handler_test.go`、`server/internal/modules/workspace/adapter/http/members.go`、`server/internal/modules/workspace/adapter/http/members_test.go`、`server/internal/modules/workspace/app/fakes_test.go`、`server/internal/modules/workspace/app/list_members.go`、`server/internal/modules/workspace/app/ports.go`、`server/internal/modules/workspace/domain/actions.go`、`server/internal/modules/workspace/domain/errors.go`、`server/internal/modules/workspace/domain/member.go`、`server/internal/modules/workspace/domain/member_test.go`、`server/internal/modules/workspace/module.go`、`web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`
- Modify（完整内容）: `server/internal/modules/workspace/app/lock.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/workspace/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.12、2.7，M3 设计 3.4、3.6、5.1–5.3）：
  - 接口描述：`updateWorkspaceMember`（`PATCH /api/v0/workspace-members/{workspace_member_id}`，`WorkspaceMemberUpdate{role}`，200 答 `WorkspaceMember`），码 `[validation_failed, workspace.member_not_found, forbidden, workspace.own_membership]`；
  - `domain.ErrMemberNotFound`（404 `workspace.member_not_found`，"The member does not exist, or you cannot see the workspace."）、`domain.ErrOwnMembership`（409 `workspace.own_membership`，"You cannot change your own membership."）；`domain.CheckMemberRole(role)`（5、15、20 之一，按集合；否则 422 `role`：`invalid_format`，"is not 5, 15 or 20"）；`domain.ActionMemberUpdate = "workspace_member.update"`；规则表一行：管理员；
  - `app.MemberUpdater{MemberByID; LockWorkspace; UpdateMemberRole}`；`app/lock.go` 分出 `decide(ctx, auth, actor, action, workspaceID, notFound)`：`lockAndDecide` 和本用例共用，`ErrNotVisible` 换成调用方给的 404；
  - `app.NewUpdateWorkspaceMember(members, profiles, auth, tx, clock)`，`Execute(ctx, id, role)`：`CheckMemberRole`（事务之前）→ 一个事务：`MemberByID` → `LockWorkspace(它的工作区)` → `MemberByID` 再读 → `decide(workspace_member.update)` → 目标的检查（已结束：`member_not_found`；自己的：`own_membership`）→ `UpdateMemberRole` → 读他的公开资料（在提交之前：读失败时事务回滚），邮箱按调用者的角色。降为访客的项目连带在 P4 加在这里；
  - `list_members.go` 与它共用 `memberUser`；
  - HTTP：handler `UpdateWorkspaceMember`；前端文案表加两个码；
  - 矩阵：`seeded{memberships}` 和 `seeded.membership(slug, caller)`：准备数据经存储写入成员关系时记下 id；`matrixRow.request` 改为 `func(c caller, s seeded) (method, path, body string)`，完整性核对随之；两行：改别人（`anotherMember`：成员改访客的，别人改成员的；管理员 200 并由 `demotesTheMember` 核对答案，成员、访客 `forbidden`，另三列 `workspace.member_not_found`）、改自己（`ownMembership`：管理员 409 `workspace.own_membership`，成员、访客 `forbidden`，另三列 404）。
- 使用者：Task 13 的交错；P4 的降级连带；P9 的成员页。

**Tests:**
- `domain/member_test.go`：`TestCheckMemberRole`（三个角色通过；0、4、10、16、21、25、-5 各是一个 422，按集合而不是范围）。
- `app/update_member_test.go`：
  - `TestUpdateWorkspaceMemberLocksThenDecidesThenWrites`：降为访客、升为管理员两次：调用恰好是读 → 锁工作区 → 再读 → 判定 → 写 → 读资料，都在一个事务里；答案带成员的资料和邮箱（管理员看得到）。
  - `TestUpdateWorkspaceMemberRefusals`（14 种）：三个值以外的角色（没有事务）；不存在的成员关系；工作区不存在；等锁期间被删除；看不到；成员的 `forbidden`（对别人的、对已结束的、对自己的都是 403：没有权限的人得不到目标的任何信息）；管理员对已结束的：`member_not_found`；等锁期间结束的：`member_not_found`；自己的：`own_membership`；读、锁、判定失败原样返回，不是 404。
  - `TestUpdateWorkspaceMemberFailsWithinTheTransaction`：写入失败、读资料失败、成员没有账户：都是错误，在事务里，不是成员；没有调用者 401。
- `adapter/http/members_test.go`：`TestUpdateWorkspaceMember`、`TestUpdateWorkspaceMemberRefusals`（四个码经契约核对；没有 `role` 的请求体 400，用例不被调用）。
- `bootstrap/workspace_test.go`：`TestAMembershipEndedMeanwhileIsNotFound`（真实数据库、组合出的 app）：一个事务锁住 `acme` 并结束 bob 的成员关系，alice 的 `PATCH` 已读到有效、等在工作区行上；提交之后她得到 404 `workspace.member_not_found`，bob 的角色不变。
- 矩阵：`TestPermissionMatrix` 的两行；`TestMatrixViolationsCatchesEachGap`、`TestThePermissionMatrixCoversEveryOperation` 随新的请求签名。

- [ ] **Step 1: 接口描述、领域、规则表**

`api/modules/workspace.yaml`（修改，2 处）：

````old api/modules/workspace.yaml
                $ref: '#/components/schemas/WorkspaceMemberList'
````

````new api/modules/workspace.yaml
                $ref: '#/components/schemas/WorkspaceMemberList'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspace-members/{workspace_member_id}:
    parameters:
      - name: workspace_member_id
        in: path
        required: true
        description: A membership's id (WorkspaceMember.id), not the member's account id.
        schema:
          type: string
          format: uuid
    patch:
      operationId: updateWorkspaceMember
      tags: [workspace]
      summary: Change a member's role
      description: >-
        For the workspace's admins. The role is checked first
        (validation_failed). A membership that does not exist, is deleted or
        has ended, or whose workspace the caller cannot see, answers
        workspace.member_not_found; a member or a guest, forbidden; the
        caller's own membership, workspace.own_membership: nobody changes his
        own role. The role is decided after the workspace row is locked, so
        of two admins who demote each other at once only the first succeeds
        and the workspace keeps an admin.
      security: [{bearer: []}]
      x-problem-codes: [validation_failed, workspace.member_not_found, forbidden, workspace.own_membership]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/WorkspaceMemberUpdate'
      responses:
        '200':
          description: The membership with its new role.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/WorkspaceMember'
````

````old api/modules/workspace.yaml
            $ref: '#/components/schemas/WorkspaceMember'
    WorkspaceUpdate:
````

````new api/modules/workspace.yaml
            $ref: '#/components/schemas/WorkspaceMember'
    WorkspaceMemberUpdate:
      type: object
      additionalProperties: false
      required: [role]
      properties:
        role:
          $ref: '#/components/schemas/WorkspaceRole'
    WorkspaceUpdate:
````

`api/openapi.yaml`（修改，1 处）：

````old api/openapi.yaml
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspaces~1{slug}~1members'
````

````new api/openapi.yaml
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspaces~1{slug}~1members'
  /api/v0/workspace-members/{workspace_member_id}:
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspace-members~1{workspace_member_id}'
````

`server/internal/modules/workspace/domain/errors.go`（修改，1 处）：

````old server/internal/modules/workspace/domain/errors.go
	ErrCreationDisabled = shared.NewError(shared.KindForbidden, "workspace.creation_disabled", "Creating workspaces is disabled on this instance.")
````

````new server/internal/modules/workspace/domain/errors.go
	ErrCreationDisabled = shared.NewError(shared.KindForbidden, "workspace.creation_disabled", "Creating workspaces is disabled on this instance.")
	// ErrMemberNotFound answers a membership that does not exist, is deleted
	// or has ended, or whose workspace the caller cannot see: the same 404
	// for all (M3 design 5.3, 8.2).
	ErrMemberNotFound = shared.NewError(shared.KindNotFound, "workspace.member_not_found", "The member does not exist, or you cannot see the workspace.")
	// ErrOwnMembership answers a change of the caller's own membership (M3
	// design 3.4, 5.3): nobody changes his own role.
	ErrOwnMembership = shared.NewError(shared.KindConflict, "workspace.own_membership", "You cannot change your own membership.")
````

`server/internal/modules/workspace/domain/member.go`（修改，1 处）：

````old server/internal/modules/workspace/domain/member.go
}

// emailReaders are the workspace roles that see the members' addresses:
````

````new server/internal/modules/workspace/domain/member.go
}

// roles are the three workspace roles (Plane's ROLE_CHOICES).
var roles = []shared.Role{shared.RoleGuest, shared.RoleMember, shared.RoleAdmin}

// CheckMemberRole checks that role is one of the three, as one 422
// validation_failed.
func CheckMemberRole(role shared.Role) error {
	if !slices.Contains(roles, role) {
		return shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldInvalidFormat, Message: "is not 5, 15 or 20"})
	}
	return nil
}

// emailReaders are the workspace roles that see the members' addresses:
````

`server/internal/modules/workspace/domain/member_test.go`（修改，2 处）：

````old server/internal/modules/workspace/domain/member_test.go
import (
````

````new server/internal/modules/workspace/domain/member_test.go
import (
	"errors"
	"slices"
````

````old server/internal/modules/workspace/domain/member_test.go
	"github.com/open-nerve/NerveProject/server/internal/shared"
)
````

````new server/internal/modules/workspace/domain/member_test.go
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The three roles pass; any other value is one 422 on role, whatever its
// size (roles are a set, not a scale).
func TestCheckMemberRole(t *testing.T) {
	for _, role := range []shared.Role{shared.RoleGuest, shared.RoleMember, shared.RoleAdmin} {
		if err := CheckMemberRole(role); err != nil {
			t.Errorf("CheckMemberRole(%d) = %v, want nil", role, err)
		}
	}
	want := []shared.FieldError{{Field: "role", Code: "invalid_format", Message: "is not 5, 15 or 20"}}
	for _, role := range []shared.Role{0, 4, 10, 16, 21, 25, -5} {
		err := CheckMemberRole(role)
		var se *shared.Error
		if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || !slices.Equal(se.Fields, want) {
			t.Errorf("CheckMemberRole(%d) = %#v, want validation_failed with %v", role, err, want)
		}
	}
}
````

`server/internal/modules/workspace/domain/actions.go`（修改，2 处）：

````old server/internal/modules/workspace/domain/actions.go
	ActionMemberList shared.Action = "workspace_member.list"
````

````new server/internal/modules/workspace/domain/actions.go
	ActionMemberList shared.Action = "workspace_member.list"
	// ActionMemberUpdate is changing a member's role: updateWorkspaceMember.
	ActionMemberUpdate shared.Action = "workspace_member.update"
````

````old server/internal/modules/workspace/domain/actions.go
	return []shared.Action{ActionRead, ActionUpdate, ActionDelete, ActionMemberList, ActionPreferencesRead, ActionPreferencesUpdate}
````

````new server/internal/modules/workspace/domain/actions.go
	return []shared.Action{ActionRead, ActionUpdate, ActionDelete, ActionMemberList, ActionMemberUpdate, ActionPreferencesRead, ActionPreferencesUpdate}
````

`server/internal/modules/access/domain/rules.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules.go
	"workspace_member.list": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
````

````new server/internal/modules/access/domain/rules.go
	"workspace_member.list": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// The relative rules (one's own role) are the use case's (M3 design 3.4).
	"workspace_member.update": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
````

`server/internal/modules/access/domain/rules_test.go`（修改，1 处）：

````old server/internal/modules/access/domain/rules_test.go
	"workspace_member.list":        {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
````

````new server/internal/modules/access/domain/rules_test.go
	"workspace_member.list":        {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
	"workspace_member.update":      {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
````

- [ ] **Step 2: 生成**

Run: `make gen`
Expected: 成功：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `8b33e05bda5cbf6bb144ae45690faa1912f77a6f8acb2003fc66c57153a9bc18` | 1353 | `api/dist/openapi.yaml` |
| `a27c79bd359d63583e6dfa7814c78efc59a34c97bda48aa0cd18a900780f903d` | 34 | `server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go` |
| `476f1cfde00fed8af63fe4b06e829f68f73a22939e8f884d4ba1ad616f13bd1e` | 1505 | `server/internal/modules/workspace/adapter/http/gen/server.gen.go` |
| `332e98a0c767752e13cd22d5adbcfec605d8084b7f3c106a2cd310764b241452` | 1433 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go server/internal/modules/workspace/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 3: 端口、判定、用例**

`server/internal/modules/workspace/app/ports.go`（修改，1 处）：

````old server/internal/modules/workspace/app/ports.go
}

// WorkspaceLocker locks FOR NO KEY UPDATE: for a write of the workspace row
````

````new server/internal/modules/workspace/app/ports.go
}

// MemberUpdater changes a membership's role under its workspace's lock (M3
// design 3.6: read the row, lock the workspace, read the row again).
type MemberUpdater interface {
	// MemberByID returns the undeleted membership id, active or not;
	// ErrNotFound when there is none.
	MemberByID(ctx context.Context, id uuid.UUID) (domain.Membership, error)
	// LockWorkspace locks the undeleted workspace id FOR NO KEY UPDATE until
	// the transaction ends; ErrNotFound when there is none, also when it was
	// deleted while the lock waited.
	LockWorkspace(ctx context.Context, id uuid.UUID) error
	// UpdateMemberRole sets the membership's role, by the account by at now,
	// and returns it as stored.
	UpdateMemberRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Membership, error)
}

// WorkspaceLocker locks FOR NO KEY UPDATE: for a write of the workspace row
````

`server/internal/modules/workspace/app/lock.go`（完整内容，48 行）：

````whole server/internal/modules/workspace/app/lock.go
package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// lockAndDecide is the first two steps of every write on a workspace named
// by its slug (M3 design 3.6 convention 2), in the transaction ctx carries:
// lock locks the undeleted workspace with slug, then decide. It returns the
// workspace's id and the grant. A workspace that is not there, deleted, or
// not visible to actor is domain.ErrNotFound; a role the rule does not
// allow is the Authorizer's shared.Forbidden.
func lockAndDecide(ctx context.Context, lock func(ctx context.Context, slug string) (uuid.UUID, error),
	auth shared.Authorizer, actor shared.Actor, slug string, action shared.Action) (uuid.UUID, shared.Grant, error) {
	id, err := lock(ctx, slug)
	switch {
	case errors.Is(err, ErrNotFound):
		return uuid.UUID{}, shared.Grant{}, domain.ErrNotFound
	case err != nil:
		return uuid.UUID{}, shared.Grant{}, err
	}
	grant, err := decide(ctx, auth, actor, action, id, domain.ErrNotFound)
	if err != nil {
		return uuid.UUID{}, shared.Grant{}, err
	}
	return id, grant, nil
}

// decide asks the Authorizer for action on the workspace for actor, who
// reads the role committed before the workspace's lock was granted: a write
// calls it under that lock. A workspace not visible to actor is notFound,
// the 404 of what the caller named.
func decide(ctx context.Context, auth shared.Authorizer, actor shared.Actor, action shared.Action, workspaceID uuid.UUID,
	notFound error) (shared.Grant, error) {
	grant, err := auth.Authorize(ctx, actor, action, shared.Target{WorkspaceID: workspaceID})
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return shared.Grant{}, notFound
	case err != nil:
		return shared.Grant{}, err
	}
	return grant, nil
}
````

`server/internal/modules/workspace/app/list_members.go`（修改，2 处）：

````old server/internal/modules/workspace/app/list_members.go
		user := domain.MemberUser{ID: p.ID, DisplayName: p.DisplayName, FirstName: p.FirstName, LastName: p.LastName}
		if seesEmails {
			user.Email = &p.Email
		}
		out[i] = domain.Member{Membership: m, User: user}
````

````new server/internal/modules/workspace/app/list_members.go
		out[i] = domain.Member{Membership: m, User: memberUser(p, seesEmails)}
````

````old server/internal/modules/workspace/app/list_members.go
	return out, nil
}

````

````new server/internal/modules/workspace/app/list_members.go
	return out, nil
}

// memberUser is p as a caller sees it: the address only when seesEmails.
func memberUser(p PublicProfile, seesEmails bool) domain.MemberUser {
	user := domain.MemberUser{ID: p.ID, DisplayName: p.DisplayName, FirstName: p.FirstName, LastName: p.LastName}
	if seesEmails {
		user.Email = &p.Email
	}
	return user
}

````

`server/internal/modules/workspace/app/update_member.go`（新文件，113 行）：

````file server/internal/modules/workspace/app/update_member.go
package app

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateWorkspaceMember changes a member's role: PATCH
// /api/v0/workspace-members/{workspace_member_id}.
type UpdateWorkspaceMember struct {
	members  MemberUpdater
	profiles MemberProfiles
	auth     shared.Authorizer
	tx       shared.TxManager
	clock    Clock
}

// NewUpdateWorkspaceMember returns the use case.
func NewUpdateWorkspaceMember(members MemberUpdater, profiles MemberProfiles, auth shared.Authorizer, tx shared.TxManager,
	clock Clock) *UpdateWorkspaceMember {
	return &UpdateWorkspaceMember{members: members, profiles: profiles, auth: auth, tx: tx, clock: clock}
}

// Execute checks role, then in one transaction (M3 design 3.6): the
// membership read for its workspace, the workspace row FOR NO KEY UPDATE,
// the membership read again under the lock, the decision on
// workspace_member.update, then the checks on the target, which only a
// caller allowed to change roles gets to see: an ended membership is
// workspace.member_not_found, the caller's own workspace.own_membership.
// The answer carries the member's profile, read without a lock (M3 design
// 3.6 convention 1) before the commit, so a failed read changes nothing.
// Demoting to guest does not touch projects yet: P4 adds that cascade here.
func (u *UpdateWorkspaceMember) Execute(ctx context.Context, id uuid.UUID, role shared.Role) (domain.Member, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Member{}, err
	}
	if err := domain.CheckMemberRole(role); err != nil {
		return domain.Member{}, err
	}
	now := u.clock.Now()
	var updated domain.Member
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		m, err := u.lockedMember(ctx, id)
		if err != nil {
			return err
		}
		grant, err := decide(ctx, u.auth, actor, domain.ActionMemberUpdate, m.WorkspaceID, domain.ErrMemberNotFound)
		if err != nil {
			return err
		}
		switch {
		case !m.IsActive:
			return domain.ErrMemberNotFound
		case m.MemberID == actor.UserID:
			return domain.ErrOwnMembership
		}
		if m, err = u.members.UpdateMemberRole(ctx, m.ID, role, actor.UserID, now); err != nil {
			return err
		}
		updated, err = u.withProfile(ctx, m, grant.WorkspaceRole)
		return err
	})
	if err != nil {
		return domain.Member{}, err
	}
	return updated, nil
}

// lockedMember reads the membership id, locks its workspace, and reads it
// again under the lock: a membership or workspace deleted meanwhile is
// domain.ErrMemberNotFound.
func (u *UpdateWorkspaceMember) lockedMember(ctx context.Context, id uuid.UUID) (domain.Membership, error) {
	m, err := u.members.MemberByID(ctx, id)
	if err != nil {
		return domain.Membership{}, memberNotFound(err)
	}
	if err := u.members.LockWorkspace(ctx, m.WorkspaceID); err != nil {
		return domain.Membership{}, memberNotFound(err)
	}
	m, err = u.members.MemberByID(ctx, id)
	if err != nil {
		return domain.Membership{}, memberNotFound(err)
	}
	return m, nil
}

// memberNotFound turns ErrNotFound into domain.ErrMemberNotFound.
func memberNotFound(err error) error {
	if errors.Is(err, ErrNotFound) {
		return domain.ErrMemberNotFound
	}
	return err
}

// withProfile is m with its member's public profile, the address as a
// caller of role sees it.
func (u *UpdateWorkspaceMember) withProfile(ctx context.Context, m domain.Membership, role shared.Role) (domain.Member, error) {
	profiles, err := u.profiles.PublicProfiles(ctx, []uuid.UUID{m.MemberID})
	if err != nil {
		return domain.Member{}, err
	}
	if len(profiles) != 1 || profiles[0].ID != m.MemberID {
		// The foreign key keeps every member's account: its absence is a bug.
		return domain.Member{}, fmt.Errorf("workspace member %s: no account %s", m.ID, m.MemberID)
	}
	return domain.Member{Membership: m, User: memberUser(profiles[0], domain.SeesEmails(role))}, nil
}
````

`server/internal/modules/workspace/app/fakes_test.go`（修改，2 处）：

````old server/internal/modules/workspace/app/fakes_test.go
	memberships map[uuid.UUID][]domain.Membership // by workspace, for ListMembers
	membersErr  error                             // for ListMembers
````

````new server/internal/modules/workspace/app/fakes_test.go
	memberships map[uuid.UUID][]domain.Membership // by workspace, for ListMembers and MemberByID
	membersErr  error                             // for ListMembers and MemberByID
	roleErr     error                             // for UpdateMemberRole
	onLock      func()                            // run by LockWorkspace once it has locked: what changed while it waited
````

````old server/internal/modules/workspace/app/fakes_test.go
}

// fakeProfiles answers the profiles it holds of the ids asked for, in the
````

````new server/internal/modules/workspace/app/fakes_test.go
}

func (f *fakeWorkspaces) MemberByID(ctx context.Context, id uuid.UUID) (domain.Membership, error) {
	f.log.add(ctx, "MemberByID %s", id)
	if f.membersErr != nil {
		return domain.Membership{}, fmt.Errorf("read workspace member: %w", f.membersErr)
	}
	for _, list := range f.memberships {
		if i := slices.IndexFunc(list, func(m domain.Membership) bool { return m.ID == id }); i >= 0 {
			return list[i], nil
		}
	}
	return domain.Membership{}, app.ErrNotFound
}

// LockWorkspace answers as the slug locks do, for the workspace with id.
func (f *fakeWorkspaces) LockWorkspace(ctx context.Context, id uuid.UUID) error {
	f.log.add(ctx, "LockWorkspace %s", id)
	i := slices.IndexFunc(f.workspaces, func(w domain.Workspace) bool { return w.ID == id })
	if i < 0 {
		return app.ErrNotFound
	}
	if _, err := f.lock(f.workspaces[i].Slug); err != nil {
		return err
	}
	if f.onLock != nil {
		f.onLock()
	}
	return nil
}

func (f *fakeWorkspaces) UpdateMemberRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Membership, error) {
	f.log.add(ctx, "UpdateMemberRole %s to %d by %s at %s", id, role, by, now.Format(time.RFC3339Nano))
	if f.roleErr != nil {
		return domain.Membership{}, fmt.Errorf("update workspace member: %w", f.roleErr)
	}
	for _, list := range f.memberships {
		if i := slices.IndexFunc(list, func(m domain.Membership) bool { return m.ID == id }); i >= 0 {
			list[i].Role = role
			return list[i], nil
		}
	}
	return domain.Membership{}, fmt.Errorf("update workspace member %s: no such row", id)
}

// fakeProfiles answers the profiles it holds of the ids asked for, in the
````

`server/internal/modules/workspace/app/update_member_test.go`（新文件，158 行）：

````file server/internal/modules/workspace/app/update_member_test.go
package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newUpdateMember is UpdateWorkspaceMember over membersFixture's fakes: in
// acme alice is the admin, bob a member, carol's membership has ended.
func newUpdateMember() (*app.UpdateWorkspaceMember, *membersFixture, *fakeTx) {
	f := newMembers()
	tx := &fakeTx{}
	return app.NewUpdateWorkspaceMember(f.workspaces, f.profiles, f.auth, tx, clocktest.At(now)), f, tx
}

// lockedMemberCalls are the calls up to the decision on the membership m
// for user: the read, the workspace's lock, the read again, the decision.
func lockedMemberCalls(user app.AccountState, m domain.Membership) []string {
	return []string{
		"MemberByID " + m.ID.String(),
		"LockWorkspace " + m.WorkspaceID.String(),
		"MemberByID " + m.ID.String(),
		"Authorize " + user.ID.String() + " workspace_member.update on " + m.WorkspaceID.String() + "/" + uuid.Nil().String(),
	}
}

// UpdateWorkspaceMember reads the membership, locks its workspace FOR NO
// KEY UPDATE, reads it again, decides, then writes the role and reads the
// member's profile, all in one transaction (M3 design 3.6); the admin sees
// the address.
func TestUpdateWorkspaceMemberLocksThenDecidesThenWrites(t *testing.T) {
	for _, role := range []shared.Role{shared.RoleGuest, shared.RoleAdmin} {
		uc, f, tx := newUpdateMember()
		got, err := uc.Execute(as(alice), bobInAcme.ID, role)
		want := bobInAcme
		want.Role = role
		if err != nil || !sameMembers([]domain.Member{got}, []domain.Member{withUser(want, true)}) {
			t.Errorf("to %d: Execute() = %+v, %v; want %+v", role, got, err, withUser(want, true))
		}
		wantCalls := append(lockedMemberCalls(alice, bobInAcme),
			fmt.Sprintf("UpdateMemberRole %s to %d by %s at %s", bobInAcme.ID, role, alice.ID, now.Format(time.RFC3339Nano)),
			fmt.Sprintf("PublicProfiles %v", []uuid.UUID{bob.ID}))
		if !slices.Equal(f.log.calls, wantCalls) || tx.calls != 1 {
			t.Errorf("to %d: calls = %q in %d transactions, want %q in one", role, f.log.calls, tx.calls, wantCalls)
		}
	}
}

// Each refusal and failure is the answer, and no role is written. The
// role's check comes first. A membership that is not there, deleted
// meanwhile, of a workspace not there or not visible is
// workspace.member_not_found; a member's forbidden comes before any check
// of the target, so he learns nothing about it; then an ended membership
// is workspace.member_not_found and the caller's own
// workspace.own_membership; a failure is never a 404.
func TestUpdateWorkspaceMemberRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	forbidBob := func(f *membersFixture) { f.auth.errs = map[grantKey]error{{bob.ID, acme.ID}: shared.Forbidden()} }
	stranger := domain.Membership{ID: uuid.NewV7(), WorkspaceID: uuid.NewV7(), MemberID: bob.ID, Role: shared.RoleMember, IsActive: true}
	decided := lockedMemberCalls(alice, bobInAcme)
	tests := []struct {
		name  string
		user  app.AccountState
		id    uuid.UUID
		role  shared.Role
		set   func(f *membersFixture)
		want  error
		calls []string
	}{
		{"a role outside the three", alice, bobInAcme.ID, 10, nil,
			shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldInvalidFormat, Message: "is not 5, 15 or 20"}), nil},
		{"no such membership", alice, stranger.ID, shared.RoleGuest, nil, domain.ErrMemberNotFound, []string{"MemberByID " + stranger.ID.String()}},
		{"no such workspace", alice, stranger.ID, shared.RoleGuest,
			func(f *membersFixture) {
				f.workspaces.memberships[stranger.WorkspaceID] = []domain.Membership{stranger}
			},
			domain.ErrMemberNotFound, []string{"MemberByID " + stranger.ID.String(), "LockWorkspace " + stranger.WorkspaceID.String()}},
		{"deleted while the lock waited", alice, bobInAcme.ID, shared.RoleGuest,
			func(f *membersFixture) {
				f.workspaces.onLock = func() { f.workspaces.memberships[acme.ID] = []domain.Membership{aliceInAcme, carolInAcme} }
			},
			domain.ErrMemberNotFound, decided[:3]},
		{"not visible", carol, bobInAcme.ID, shared.RoleGuest, nil, domain.ErrMemberNotFound, lockedMemberCalls(carol, bobInAcme)},
		{"a member", bob, aliceInAcme.ID, shared.RoleGuest, forbidBob, shared.Forbidden(), lockedMemberCalls(bob, aliceInAcme)},
		{"a member, of an ended membership", bob, carolInAcme.ID, shared.RoleGuest, forbidBob, shared.Forbidden(), lockedMemberCalls(bob, carolInAcme)},
		{"a member, of his own", bob, bobInAcme.ID, shared.RoleGuest, forbidBob, shared.Forbidden(), lockedMemberCalls(bob, bobInAcme)},
		{"an ended membership", alice, carolInAcme.ID, shared.RoleMember, nil, domain.ErrMemberNotFound, lockedMemberCalls(alice, carolInAcme)},
		{"ended while the lock waited", alice, bobInAcme.ID, shared.RoleGuest,
			func(f *membersFixture) {
				f.workspaces.onLock = func() { f.workspaces.memberships[acme.ID][1].IsActive = false }
			},
			domain.ErrMemberNotFound, decided},
		{"his own", alice, aliceInAcme.ID, shared.RoleMember, nil, domain.ErrOwnMembership, lockedMemberCalls(alice, aliceInAcme)},
		{"the read failed", alice, bobInAcme.ID, shared.RoleGuest, func(f *membersFixture) { f.workspaces.membersErr = failure }, failure,
			[]string{"MemberByID " + bobInAcme.ID.String()}},
		{"the lock failed", alice, bobInAcme.ID, shared.RoleGuest, func(f *membersFixture) { f.workspaces.lockErrs = map[string]error{"acme": failure} },
			failure, decided[:2]},
		{"the Authorizer failed", alice, bobInAcme.ID, shared.RoleGuest,
			func(f *membersFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: failure} }, failure, decided},
	}
	for _, tt := range tests {
		uc, f, tx := newUpdateMember()
		if tt.set != nil {
			tt.set(f)
		}
		got, err := uc.Execute(as(tt.user), tt.id, tt.role)
		if !errors.Is(err, tt.want) || got != (domain.Member{}) {
			t.Errorf("%s: Execute() = %+v, %v; want no member and %v", tt.name, got, err, tt.want)
		}
		if tt.want == failure && errors.Is(err, domain.ErrMemberNotFound) {
			t.Errorf("%s: Execute() = %v, which is also workspace.member_not_found", tt.name, err)
		}
		wantTx := 1
		if tt.calls == nil {
			wantTx = 0
		}
		if !slices.Equal(f.log.calls, tt.calls) || tx.calls != wantTx {
			t.Errorf("%s: calls = %q in %d transactions, want %q in %d", tt.name, f.log.calls, tx.calls, tt.calls, wantTx)
		}
	}
}

// A failed write, a failed read of the profile, and a member without an
// account each fail the transaction, which the database then rolls back:
// the answer is the error, never a member.
func TestUpdateWorkspaceMemberFailsWithinTheTransaction(t *testing.T) {
	failure := errors.New("connection reset")
	for name, set := range map[string]func(f *membersFixture){
		"the write":            func(f *membersFixture) { f.workspaces.roleErr = failure },
		"the profile":          func(f *membersFixture) { f.profiles.err = failure },
		"a member without one": func(f *membersFixture) { f.profiles.profiles = profiles[:2] },
	} {
		uc, f, tx := newUpdateMember()
		set(f)
		got, err := uc.Execute(as(alice), bobInAcme.ID, shared.RoleGuest)
		if err == nil || errors.Is(err, domain.ErrMemberNotFound) || got != (domain.Member{}) || tx.calls != 1 {
			t.Errorf("%s failing: Execute() = %+v, %v in %d transactions; want an error that is not a 404", name, got, err, tx.calls)
		}
		if slices.ContainsFunc(f.log.calls, func(c string) bool { return strings.HasSuffix(c, " outside tx") }) {
			t.Errorf("%s failing: calls %q, want all in the transaction", name, f.log.calls)
		}
	}
	uc, f, _ := newUpdateMember()
	if _, err := uc.Execute(context.Background(), bobInAcme.ID, shared.RoleGuest); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and no call", err, f.log.calls)
	}
}
````

- [ ] **Step 4: HTTP 一侧和接线**

`server/internal/modules/workspace/adapter/http/handler.go`（修改，4 处）：

````old server/internal/modules/workspace/adapter/http/handler.go
	"context"
````

````new server/internal/modules/workspace/adapter/http/handler.go
	"context"
	"uuid"
````

````old server/internal/modules/workspace/adapter/http/handler.go
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
````

````new server/internal/modules/workspace/adapter/http/handler.go
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````

````old server/internal/modules/workspace/adapter/http/handler.go
}

// GetPreferencesUseCase is app.GetWorkspacePreferences.
````

````new server/internal/modules/workspace/adapter/http/handler.go
}

// UpdateMemberUseCase is app.UpdateWorkspaceMember.
type UpdateMemberUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, role shared.Role) (domain.Member, error)
}

// GetPreferencesUseCase is app.GetWorkspacePreferences.
````

````old server/internal/modules/workspace/adapter/http/handler.go
	ListMembers       ListMembersUseCase
````

````new server/internal/modules/workspace/adapter/http/handler.go
	ListMembers       ListMembersUseCase
	UpdateMember      UpdateMemberUseCase
````

`server/internal/modules/workspace/adapter/http/members.go`（修改，2 处）：

````old server/internal/modules/workspace/adapter/http/members.go
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
````

````new server/internal/modules/workspace/adapter/http/members.go
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````

````old server/internal/modules/workspace/adapter/http/members.go
	return out, nil
````

````new server/internal/modules/workspace/adapter/http/members.go
	return out, nil
}

// UpdateWorkspaceMember serves PATCH /api/v0/workspace-members/{workspace_member_id}.
func (h handler) UpdateWorkspaceMember(ctx context.Context, req gen.UpdateWorkspaceMemberRequestObject) (gen.UpdateWorkspaceMemberResponseObject, error) {
	m, err := h.uc.UpdateMember.Execute(ctx, req.WorkspaceMemberID, shared.Role(req.Body.Role))
	if err != nil {
		return nil, err
	}
	return gen.UpdateWorkspaceMember200JSONResponse(member(m)), nil
````

`server/internal/modules/workspace/adapter/http/handler_test.go`（修改，5 处）：

````old server/internal/modules/workspace/adapter/http/handler_test.go
	"context"
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
	"context"
	"fmt"
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
	member *fakeMembers
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
	member *fakeMembers
	role   *fakeUpdateMember
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
}

// fakePrefs is both preference use cases: each call is recorded as
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
}

type fakeUpdateMember struct {
	calls  []string // "caller id role"
	answer domain.Member
	err    error
}

func (f *fakeUpdateMember) Execute(ctx context.Context, id uuid.UUID, role shared.Role) (domain.Member, error) {
	f.calls = append(f.calls, fmt.Sprintf("%s %s %d", caller(ctx), id, role))
	return f.answer, f.err
}

// fakePrefs is both preference use cases: each call is recorded as
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
		f.member = &fakeMembers{}
	}
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
		f.member = &fakeMembers{}
	}
	if f.role == nil {
		f.role = &fakeUpdateMember{}
	}
````

````old server/internal/modules/workspace/adapter/http/handler_test.go
		ListMembers: f.member, GetPreferences: fakeGetPrefs{f.prefs}, UpdatePreferences: fakeUpdatePrefs{f.prefs},
````

````new server/internal/modules/workspace/adapter/http/handler_test.go
		ListMembers: f.member, UpdateMember: f.role, GetPreferences: fakeGetPrefs{f.prefs}, UpdatePreferences: fakeUpdatePrefs{f.prefs},
````

`server/internal/modules/workspace/adapter/http/members_test.go`（修改，1 处）：

````old server/internal/modules/workspace/adapter/http/members_test.go
)

// The list is the use case's for the caller and the slug of the path: an
````

````new server/internal/modules/workspace/adapter/http/members_test.go
)

// PATCH changes the role of the membership of the path for the caller and
// answers the membership; the role is the use case's to check.
func TestUpdateWorkspaceMember(t *testing.T) {
	role := &fakeUpdateMember{answer: aliceMember}
	h := newServer(t, fakes{role: role})
	for _, body := range []string{`{"role":20}`, `{"role":10}`} {
		res, got := do(t, h, request(http.MethodPatch, "/api/v0/workspace-members/"+aliceMember.ID.String(), "bob", body))
		if res.StatusCode != http.StatusOK || got != aliceMemberJSON+"\n" {
			t.Errorf("PATCH %s = %d %s, want 200 %s", body, res.StatusCode, got, aliceMemberJSON)
		}
	}
	id := aliceMember.ID.String()
	if want := []string{"bob " + id + " 20", "bob " + id + " 10"}; !slices.Equal(role.calls, want) {
		t.Errorf("calls = %q, want %q", role.calls, want)
	}
}

// The use case's refusals, as the contract declares them; a body without a
// role is refused before it.
func TestUpdateWorkspaceMemberRefusals(t *testing.T) {
	tests := []struct {
		err    error
		status int
		want   string
	}{
		{shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldInvalidFormat, Message: "is not 5, 15 or 20"}), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"role","code":"invalid_format","message":"is not 5, 15 or 20"}]}`},
		{domain.ErrMemberNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.member_not_found","title":"Not Found","detail":"The member does not exist, or you cannot see the workspace."}`},
		{shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{domain.ErrOwnMembership, http.StatusConflict,
			`{"status":409,"code":"workspace.own_membership","title":"Conflict","detail":"You cannot change your own membership."}`},
	}
	path := "/api/v0/workspace-members/" + bobMember.ID.String()
	for _, tt := range tests {
		h := newServer(t, fakes{role: &fakeUpdateMember{err: tt.err}})
		if res, body := do(t, h, request(http.MethodPatch, path, "alice", `{"role":5}`)); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("PATCH refused with %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
	}
	role := &fakeUpdateMember{}
	h := newServer(t, fakes{role: role})
	if res, _ := do(t, h, request(http.MethodPatch, path, "alice", `{}`)); res.StatusCode != http.StatusBadRequest || len(role.calls) != 0 {
		t.Errorf("PATCH without a role = %d, calls %q; want 400 and no call", res.StatusCode, role.calls)
	}
}

// The list is the use case's for the caller and the slug of the path: an
````

`server/internal/modules/workspace/module.go`（修改，2 处）：

````old server/internal/modules/workspace/module.go
// changing and deleting workspaces, checking a slug, listing the members,
// and each member's display settings, and offers the other modules its
// reads through ports.
````

````new server/internal/modules/workspace/module.go
// changing and deleting workspaces, checking a slug, listing the members
// and changing their roles, and each member's display settings, and offers
// the other modules its reads through ports.
````

````old server/internal/modules/workspace/module.go
		ListMembers:       app.NewListWorkspaceMembers(store, d.Profiles, d.Authorizer),
````

````new server/internal/modules/workspace/module.go
		ListMembers:       app.NewListWorkspaceMembers(store, d.Profiles, d.Authorizer),
		UpdateMember:      app.NewUpdateWorkspaceMember(store, d.Profiles, d.Authorizer, d.Tx, d.Clock),
````

- [ ] **Step 5: 矩阵和整程序测试**

`server/internal/bootstrap/permission_matrix_test.go`（修改，17 处）：

````old server/internal/bootstrap/permission_matrix_test.go
// matrixRow is an operation's row: the request each caller sends and the
// answer each gets.
````

````new server/internal/bootstrap/permission_matrix_test.go
// matrixRow is an operation's row: the request each caller sends, which can
// name a row prepareMatrix seeded, and the answer each gets.
````

````old server/internal/bootstrap/permission_matrix_test.go
	request func(c caller) (method, path, body string)
````

````new server/internal/bootstrap/permission_matrix_test.go
	request func(c caller, s seeded) (method, path, body string)
````

````old server/internal/bootstrap/permission_matrix_test.go
func sameRequest(method, path, body string) func(caller) (string, string, string) {
	return func(caller) (string, string, string) { return method, path, body }
````

````new server/internal/bootstrap/permission_matrix_test.go
func sameRequest(method, path, body string) func(caller, seeded) (string, string, string) {
	return func(caller, seeded) (string, string, string) { return method, path, body }
}

// seeded are the ids of the rows prepareMatrix seeded that a request can
// name: each membership, by the workspace's slug and the column.
type seeded struct {
	memberships map[string]uuid.UUID
}

// membership is the id of c's membership of the workspace slug; the zero id
// when there is none, which no row has.
func (s seeded) membership(slug string, c caller) uuid.UUID {
	return s.memberships[slug+"/"+string(c)]
````

````old server/internal/bootstrap/permission_matrix_test.go
// of it shares, and each column's access token.
````

````new server/internal/bootstrap/permission_matrix_test.go
// of it shares, each column's access token, and the seeded ids.
````

````old server/internal/bootstrap/permission_matrix_test.go
	tokens  map[caller]string
````

````new server/internal/bootstrap/permission_matrix_test.go
	tokens  map[caller]string
	seeded  seeded
````

````old server/internal/bootstrap/permission_matrix_test.go
// with its admin; and the workspace other, whose admin was never a member of
// acme and where the removed member is still active, so that a role read in
// the wrong workspace lets either into acme. Through the API, gone deleted
// by its admin, which soft-deletes its memberships with it. Through SQL,
// until P5's store replaces it, the removed member's membership of acme
// ended. Everything that connected to the database is closed when it
// returns, so that it can be copied.
````

````new server/internal/bootstrap/permission_matrix_test.go
// with its admin and the member; and the workspace other, whose admin was
// never a member of acme and where the removed member is still active, so
// that a role read in the wrong workspace lets either into acme. Through the
// API, gone deleted by its admin, which soft-deletes its memberships with
// it. Through SQL, until P5's store replaces it, the removed member's
// membership of acme ended. Everything that connected to the database is
// closed when it returns, so that it can be copied.
````

````old server/internal/bootstrap/permission_matrix_test.go
		seed := matrixSeed{t: t, store: workspacepg.New(pool), ids: ids, now: time.Now()}
		acme := seed.workspace("acme", callerAdmin)
		seed.join(acme, callerAdmin, shared.RoleAdmin)
		seed.join(acme, callerMember, shared.RoleMember)
		seed.join(acme, callerGuest, shared.RoleGuest)
		seed.join(acme, callerRemoved, shared.RoleMember)
````

````new server/internal/bootstrap/permission_matrix_test.go
		seed := matrixSeed{t: t, store: workspacepg.New(pool), ids: ids, now: time.Now(),
			workspaces: map[string]uuid.UUID{}, memberships: map[string]uuid.UUID{}}
		seed.workspace("acme", callerAdmin)
		seed.join("acme", callerAdmin, shared.RoleAdmin)
		seed.join("acme", callerMember, shared.RoleMember)
		seed.join("acme", callerGuest, shared.RoleGuest)
		seed.join("acme", callerRemoved, shared.RoleMember)
````

````old server/internal/bootstrap/permission_matrix_test.go
		seed.preferences(acme, callerAdmin, workspacedomain.PreferencesPatch{NavigationControl: &tabbed, NavigationProjectLimit: &three})
		gone := seed.workspace("gone", callerDeleted)
		seed.join(gone, callerDeleted, shared.RoleAdmin)
		other := seed.workspace("other", callerNever)
		seed.join(other, callerNever, shared.RoleAdmin)
		seed.join(other, callerRemoved, shared.RoleMember)
````

````new server/internal/bootstrap/permission_matrix_test.go
		seed.preferences("acme", callerAdmin, workspacedomain.PreferencesPatch{NavigationControl: &tabbed, NavigationProjectLimit: &three})
		seed.workspace("gone", callerDeleted)
		seed.join("gone", callerDeleted, shared.RoleAdmin)
		seed.join("gone", callerMember, shared.RoleMember)
		seed.workspace("other", callerNever)
		seed.join("other", callerNever, shared.RoleAdmin)
		seed.join("other", callerRemoved, shared.RoleMember)
		d.seeded = seeded{memberships: seed.memberships}
````

````old server/internal/bootstrap/permission_matrix_test.go
		seed.exec(pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2",
			acme, ids[callerRemoved])
````

````new server/internal/bootstrap/permission_matrix_test.go
		seed.exec(pool, "UPDATE workspace_members SET is_active = false WHERE id = $1", d.seeded.membership("acme", callerRemoved))
````

````old server/internal/bootstrap/permission_matrix_test.go
// through the workspace store; exec runs the SQL that stands in for the
// store P5 adds.
````

````new server/internal/bootstrap/permission_matrix_test.go
// through the workspace store, and keeps their ids by slug (and column);
// exec runs the SQL that stands in for the store P5 adds.
````

````old server/internal/bootstrap/permission_matrix_test.go
	t     *testing.T
	store *workspacepg.Store
	ids   map[caller]uuid.UUID
	now   time.Time
````

````new server/internal/bootstrap/permission_matrix_test.go
	t           *testing.T
	store       *workspacepg.Store
	ids         map[caller]uuid.UUID
	now         time.Time
	workspaces  map[string]uuid.UUID // by slug
	memberships map[string]uuid.UUID // by slug/column, as seeded.membership reads them
````

````old server/internal/bootstrap/permission_matrix_test.go
func (s matrixSeed) workspace(slug string, admin caller) uuid.UUID {
````

````new server/internal/bootstrap/permission_matrix_test.go
func (s matrixSeed) workspace(slug string, admin caller) {
````

````old server/internal/bootstrap/permission_matrix_test.go
	return w.ID
````

````new server/internal/bootstrap/permission_matrix_test.go
	s.workspaces[slug] = w.ID
````

````old server/internal/bootstrap/permission_matrix_test.go
func (s matrixSeed) join(workspace uuid.UUID, c caller, role shared.Role) {
	s.t.Helper()
````

````new server/internal/bootstrap/permission_matrix_test.go
func (s matrixSeed) join(slug string, c caller, role shared.Role) {
	s.t.Helper()
	id := uuid.NewV7()
````

````old server/internal/bootstrap/permission_matrix_test.go
		ID: uuid.NewV7(), WorkspaceID: workspace, MemberID: s.ids[c], Role: role, CreatedBy: s.ids[c], Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
}

func (s matrixSeed) preferences(workspace uuid.UUID, c caller, p workspacedomain.PreferencesPatch) {
````

````new server/internal/bootstrap/permission_matrix_test.go
		ID: id, WorkspaceID: s.workspaces[slug], MemberID: s.ids[c], Role: role, CreatedBy: s.ids[c], Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
	s.memberships[slug+"/"+string(c)] = id
}

func (s matrixSeed) preferences(slug string, c caller, p workspacedomain.PreferencesPatch) {
````

````old server/internal/bootstrap/permission_matrix_test.go
		ID: uuid.NewV7(), WorkspaceID: workspace, UserID: s.ids[c], Patch: p, Now: s.now,
````

````new server/internal/bootstrap/permission_matrix_test.go
		ID: uuid.NewV7(), WorkspaceID: s.workspaces[slug], UserID: s.ids[c], Patch: p, Now: s.now,
````

````old server/internal/bootstrap/permission_matrix_test.go
				method, path, body := r.request(c)
````

````new server/internal/bootstrap/permission_matrix_test.go
				method, path, body := r.request(c, d.seeded)
````

`server/internal/bootstrap/permission_matrix_coverage_test.go`（修改，3 处）：

````old server/internal/bootstrap/permission_matrix_coverage_test.go
			method, path, _ := r.request(c)
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
			method, path, _ := r.request(c, seeded{})
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
		r.request = func(c caller) (string, string, string) {
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
		r.request = func(c caller, s seeded) (string, string, string) {
````

````old server/internal/bootstrap/permission_matrix_coverage_test.go
			return get(c)
````

````new server/internal/bootstrap/permission_matrix_coverage_test.go
			return get(c, s)
````

`server/internal/bootstrap/permission_matrix_workspace_test.go`（修改，5 处）：

````old server/internal/bootstrap/permission_matrix_workspace_test.go
	cellCreationDisabled  = cell{http.StatusForbidden, "workspace.creation_disabled"}
````

````new server/internal/bootstrap/permission_matrix_workspace_test.go
	cellCreationDisabled  = cell{http.StatusForbidden, "workspace.creation_disabled"}
	cellMemberNotFound    = cell{http.StatusNotFound, "workspace.member_not_found"}
	cellOwnMembership     = cell{http.StatusConflict, "workspace.own_membership"}
````

````old server/internal/bootstrap/permission_matrix_workspace_test.go
func toWorkspace(method, path, body string) func(caller) (string, string, string) {
	return func(c caller) (string, string, string) {
		return method, "/api/v0/workspaces/" + workspaceOf(c) + path, body
	}
````

````new server/internal/bootstrap/permission_matrix_workspace_test.go
func toWorkspace(method, path, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, _ seeded) (string, string, string) {
		return method, "/api/v0/workspaces/" + workspaceOf(c) + path, body
	}
}

// ofMember are the cells of a row that names a membership: the answers of
// the workspace's admin, member and guest, and workspace.member_not_found
// for the callers the workspace is not visible to.
func ofMember(admin, member, guest cell) map[caller]cell {
	return map[caller]cell{callerAdmin: admin, callerMember: member, callerGuest: guest,
		callerNever: cellMemberNotFound, callerRemoved: cellMemberNotFound, callerDeleted: cellMemberNotFound}
}

// toMembership is the request of a row whose callers each PATCH body to the
// membership target names for their column: a workspace's slug and whose
// membership of it.
func toMembership(target func(caller) (string, caller), body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		slug, who := target(c)
		return http.MethodPatch, "/api/v0/workspace-members/" + s.membership(slug, who).String(), body
	}
}

// anotherMember is, for each column, a membership of another account in the
// workspace the column targets: the member's, or the guest's for the member
// himself.
func anotherMember(c caller) (string, caller) {
	switch c {
	case callerMember:
		return "acme", callerGuest
	case callerDeleted:
		return "gone", callerMember
	}
	return "acme", callerMember
}

// ownMembership is, for each column, the caller's own membership of the
// workspace the column targets: ended for the removed member, deleted with
// gone. Never a member has none in acme: his cell names the member's.
func ownMembership(c caller) (string, caller) {
	switch c {
	case callerNever:
		return "acme", callerMember
	case callerDeleted:
		return "gone", callerDeleted
	}
	return "acme", c
````

````old server/internal/bootstrap/permission_matrix_workspace_test.go
			check: listsTheMembers},
````

````new server/internal/bootstrap/permission_matrix_workspace_test.go
			check: listsTheMembers},
		{op: "updateWorkspaceMember", variant: "another member", write: true, request: toMembership(anotherMember, `{"role":5}`),
			cells: ofMember(cellOK, cellForbidden, cellForbidden), check: demotesTheMember},
		{op: "updateWorkspaceMember", variant: "one's own", write: true, request: toMembership(ownMembership, `{"role":15}`),
			cells: ofMember(cellOwnMembership, cellForbidden, cellForbidden)},
````

````old server/internal/bootstrap/permission_matrix_workspace_test.go
			cells: inWorkspace(cellOK, cellOK, cellOK), check: preferencesAre(navigation{"TABBED", 5}, navigation{"ACCORDION", 5})},
````

````new server/internal/bootstrap/permission_matrix_workspace_test.go
			cells: inWorkspace(cellOK, cellOK, cellOK), check: preferencesAre(navigation{"TABBED", 5}, navigation{"ACCORDION", 5})},
	}
}

// demotesTheMember: the admin's answer is the member's membership, now a
// guest's, with the member's address.
func demotesTheMember(t *testing.T, c caller, answer string) {
	var m struct {
		Role   int `json:"role"`
		Member struct {
			Email string `json:"email"`
		} `json:"member"`
	}
	decodeAnswer(t, answer, &m)
	if m.Role != 5 || m.Member.Email != "member@example.com" {
		t.Errorf("%s's change answers %+v, want the member as a guest", c, m)
````

````old server/internal/bootstrap/permission_matrix_workspace_test.go
func toPreferences(method, body string) func(caller) (string, string, string) {
	return func(c caller) (string, string, string) {
````

````new server/internal/bootstrap/permission_matrix_workspace_test.go
func toPreferences(method, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, _ seeded) (string, string, string) {
````

`server/internal/bootstrap/workspace_test.go`（修改，2 处）：

````old server/internal/bootstrap/workspace_test.go
	"time"
````

````new server/internal/bootstrap/workspace_test.go
	"time"
	"uuid"
````

````old server/internal/bootstrap/workspace_test.go
		t.Errorf("update = %d %s, name %q; want 403 forbidden and the name unchanged", a.res.StatusCode, a.body, name)
	}
}

````

````new server/internal/bootstrap/workspace_test.go
		t.Errorf("update = %d %s, name %q; want 403 forbidden and the name unchanged", a.res.StatusCode, a.body, name)
	}
}

// The composed updateWorkspaceMember reads the membership again once it
// holds the workspace's lock (M3 design 3.6 convention 2). A transaction
// holds acme's row FOR NO KEY UPDATE and ends bob's membership; alice's
// PATCH has read it active and waits on the row. Once the end commits she
// gets workspace.member_not_found and bob's role stays: a use case that
// went on with the row read before the lock would have changed an ended
// membership.
func TestAMembershipEndedMeanwhileIsNotFound(t *testing.T) {
	contract := apitest.Load(t)
	base, pool := sessionApp(t)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	registerAccount(t, contract, base, "bob@example.com")
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice, `{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("create = %d %s", status, body)
	}
	bob := uuid.NewV7()
	if _, err := pool.Exec(context.Background(), `INSERT INTO workspace_members (id, workspace_id, member_id, role)
		SELECT $1, w.id, u.id, 15 FROM workspaces w, users u WHERE w.slug = 'acme' AND u.email = 'bob@example.com'`, bob); err != nil {
		t.Fatal(err)
	}

	end, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = end.Rollback(context.Background()) }()
	if _, err := end.Exec(context.Background(), "SELECT id FROM workspaces WHERE slug = 'acme' FOR NO KEY UPDATE"); err != nil {
		t.Fatal(err)
	}
	if _, err := end.Exec(context.Background(), "UPDATE workspace_members SET is_active = false WHERE id = $1", bob); err != nil {
		t.Fatal(err)
	}
	req := newRequest(t, http.MethodPatch, base+"/api/v0/workspace-members/"+bob.String(), alice, []byte(`{"role":5}`))
	contract.CheckRequest(t, req)
	answered := sendInBackground(req)
	pgtest.WaitForLockWaitOn(t, pool, "workspaces", 5*time.Second)
	if err := end.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}

	a := receiveWithin(t, answered, 10*time.Second, "answer to the change")
	if a.err != nil {
		t.Fatal(a.err)
	}
	contract.CheckResponse(t, req, a.res)
	var role int
	if err := pool.QueryRow(context.Background(), "SELECT role FROM workspace_members WHERE id = $1", bob).Scan(&role); err != nil {
		t.Fatal(err)
	}
	if a.res.StatusCode != http.StatusNotFound || problemCode(t, a.body) != "workspace.member_not_found" || role != 15 {
		t.Errorf("change = %d %s, role %d; want 404 workspace.member_not_found and the role unchanged", a.res.StatusCode, a.body, role)
	}
}

````

- [ ] **Step 6: 前端的文案**

`web/apps/web/helpers/authentication.helper.ts`（修改，1 处）：

````old web/apps/web/helpers/authentication.helper.ts
  "workspace.slug_taken": "auth.errors.workspace_slug_taken",
````

````new web/apps/web/helpers/authentication.helper.ts
  "workspace.slug_taken": "auth.errors.workspace_slug_taken",
  "workspace.member_not_found": "auth.errors.workspace_member_not_found",
  "workspace.own_membership": "auth.errors.workspace_own_membership",
````

`web/packages/i18n/src/locales/en/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/auth.json
      "forbidden": "Your role does not allow this.",
````

````new web/packages/i18n/src/locales/en/auth.json
      "forbidden": "Your role does not allow this.",
      "workspace_member_not_found": "The member does not exist, or you cannot see the workspace.",
      "workspace_own_membership": "You cannot change your own membership.",
````

`web/packages/i18n/src/locales/zh-CN/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/auth.json
      "forbidden": "你的角色不能做这件事。",
````

````new web/packages/i18n/src/locales/zh-CN/auth.json
      "forbidden": "你的角色不能做这件事。",
      "workspace_member_not_found": "成员不存在，或你看不到这个工作区。",
      "workspace_own_membership": "不能修改自己的成员身份。",
````

- [ ] **Step 7: 测试、耗时、检查**

Run: `go -C server test -count=1 ./internal/modules/workspace/... ./internal/modules/access/...`
Expected: 全部 `ok`。

Run: `go -C server test -count=1 -run 'TestAMembershipEndedMeanwhileIsNotFound|TestAnAdminDemotedMeanwhileCannotUpdateTheWorkspace|TestPermissionMatrix|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEachGap|TestEveryActionHasARuleAndEveryRuleAnAction' ./internal/bootstrap/`
Expected: `ok`。

Run: `go -C server test -count=1 -v -run 'TestPermissionMatrix$' ./internal/bootstrap/`
Expected: `--- PASS: TestPermissionMatrix`；记下它和 `prepare` 子测试的耗时（原型和复现：72 格 0.21–1.60 秒，准备 0.05–0.07 秒，spec 附录 A；M3 设计 9.2 的预算是全部 20–30 秒）。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过（文案表含 `workspace.member_not_found`、`workspace.own_membership`）。

- [ ] **Step 8: 提交**

```bash
git add api/modules/workspace.yaml api/openapi.yaml server/internal/bootstrap/permission_matrix_coverage_test.go server/internal/bootstrap/permission_matrix_test.go server/internal/bootstrap/permission_matrix_workspace_test.go server/internal/bootstrap/workspace_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go server/internal/modules/workspace/adapter/http/handler.go server/internal/modules/workspace/adapter/http/handler_test.go server/internal/modules/workspace/adapter/http/members.go server/internal/modules/workspace/adapter/http/members_test.go server/internal/modules/workspace/app/fakes_test.go server/internal/modules/workspace/app/list_members.go server/internal/modules/workspace/app/lock.go server/internal/modules/workspace/app/ports.go server/internal/modules/workspace/app/update_member.go server/internal/modules/workspace/app/update_member_test.go server/internal/modules/workspace/domain/actions.go server/internal/modules/workspace/domain/errors.go server/internal/modules/workspace/domain/member.go server/internal/modules/workspace/domain/member_test.go server/internal/modules/workspace/module.go web/apps/web/helpers/authentication.helper.ts web/packages/i18n/src/locales/en/auth.json web/packages/i18n/src/locales/zh-CN/auth.json api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go server/internal/modules/workspace/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P2): updateWorkspaceMember decides under the workspace's lock

PATCH /api/v0/workspace-members/{id}, for the workspace's admins: the
membership is read, its workspace locked, the membership read again, the
role decided; then an ended membership is not found and one's own is
workspace.own_membership. The matrix's requests can name a seeded row.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**变异**：

| 改坏 | 必须失败的测试 |
|---|---|
| 按第一次读到的判定，再锁、再读（先判定） | `TestUpdateWorkspaceMemberLocksThenDecidesThenWrites`（Task 13 起另有 `TestTwoAdminsDemotingEachOtherLeaveAnAdmin`） |
| 锁下不再读 | `TestUpdateWorkspaceMemberLocksThenDecidesThenWrites`、`TestAMembershipEndedMeanwhileIsNotFound` |
| 读、锁、再读挪到事务之前 | `TestUpdateWorkspaceMemberLocksThenDecidesThenWrites`、`TestUpdateWorkspaceMemberFailsWithinTheTransaction` |
| 允许改自己的 | `TestUpdateWorkspaceMemberRefusals`、`TestPermissionMatrix/updateWorkspaceMember,_one's_own/admin` |
| 允许改已结束的 | `TestUpdateWorkspaceMemberRefusals`、`TestAMembershipEndedMeanwhileIsNotFound` |
| 目标的检查挪到判定之前 | `TestUpdateWorkspaceMemberRefusals`、`TestPermissionMatrix/updateWorkspaceMember,_one's_own/member`、`…/guest` |
| 只把"已结束"挪到判定之前 | `TestUpdateWorkspaceMemberRefusals`（"a member, of an ended membership"） |
| `CheckMemberRole` 按范围（5 到 20） | `TestCheckMemberRole`、`TestUpdateWorkspaceMemberRefusals` |
| `workspace.own_membership` 答 403 | `TestUpdateWorkspaceMemberRefusals`（HTTP）、`TestPermissionMatrix/updateWorkspaceMember,_one's_own/admin` |
| HTTP 测试不再答 `workspace.own_membership` | `apitest.Main`：`api/modules/workspace.yaml declares problem codes that no test answered through CheckResponse` |
| 规则表加一行而 `tableCells` 没有 | `TestEveryRuleDecidesItsCells`、`TestEveryActionHasARuleAndEveryRuleAnAction` |

**Done when:** 用例、HTTP、整程序测试和矩阵的 12 格通过；矩阵共 72 格，耗时记下；`make test-web` 通过。

---

### Task 13: 交错 2：两位管理员互相降级

**Files:**
- Create: `server/internal/bootstrap/interleaving_roles_test.go`

**Interfaces:**
- Produces（spec 2.13，M3 设计 9.3 交错 2）：`interleaving_roles_test.go`：`gatedMembers`（存储，`UpdateMemberRole` 之前停在闸门上：判定之后、写入之前，持着工作区的锁）、`adminRace`（`acme` 有两位管理员 alice、bob）、`change`（真实的用例，`identity` 的资料和 `access` 的 `Authorizer` 照 `bootstrap` 的接法）；复用 P1 的 `race`、`gate`、`run`、`held`、`result`。
- 使用者：P5 的交错 1（两位管理员同时离开）照同样的写法。

**Tests:**
- `TestTwoAdminsDemotingEachOtherLeaveAnAdmin`，两个顺序（alice 先、bob 先）：先的一方降另一方为成员，停在闸门上；后的一方开始降先的一方；`WaitForLockWaitOn(…, "workspaces", …)` 确认它等在工作区行上，此时两人都还是管理员；闸门打开：先的一方成功，后的一方 403 `forbidden`（它在锁下判定时已是成员）；先的一方仍是管理员。每个等待 10 秒为限。

- [ ] **Step 1: 交错**

`server/internal/bootstrap/interleaving_roles_test.go`（新文件，150 行）：

````file server/internal/bootstrap/interleaving_roles_test.go
package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The interleaving 2 of M3 design 9.3: two admins demote each other at once,
// in both orders, on a real database, through the real use case and the
// Authorizer as bootstrap wires them. The first side's gate, inside its
// transaction after its decision and before its write, holds the workspace's
// lock; pgtest.WaitForLockWaitOn proves that the other side waits on the
// workspace row before the gate opens. The first side succeeds; the other
// decides once the first has committed, as a member, and is refused
// forbidden: the workspace keeps an admin (3.6 convention 2, spike S1b).

// gatedMembers stops a role change before its write, after its decision,
// holding the workspace row's lock.
type gatedMembers struct {
	*workspacepg.Store
	gate *gate
}

func (m gatedMembers) UpdateMemberRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (
	workspacedomain.Membership, error) {
	if err := m.gate.wait(ctx); err != nil {
		return workspacedomain.Membership{}, err
	}
	return m.Store.UpdateMemberRole(ctx, id, role, by, now)
}

// adminRace is a database with acme, whose admins are alice and bob, and the
// ids of their accounts and memberships.
type adminRace struct {
	race
	bob                  uuid.UUID
	aliceIn, bobIn, acme uuid.UUID
}

func newAdminRace(t *testing.T) adminRace {
	t.Helper()
	r := adminRace{race: newRace(t), bob: uuid.NewV7(), aliceIn: uuid.NewV7(), bobIn: uuid.NewV7()}
	now := time.Now()
	if err := identitypg.New(r.pool).CreateUser(context.Background(), identityapp.NewUser{
		ID: r.bob, Email: "bob@example.com", PasswordHash: "x", DisplayName: "bob", Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	store := workspacepg.New(r.pool)
	w, err := store.CreateWorkspace(context.Background(), workspaceapp.WorkspaceRow{
		ID: uuid.NewV7(), Name: "Acme", Slug: "acme", Timezone: "UTC", CreatedBy: r.alice, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	r.acme = w.ID
	for id, user := range map[uuid.UUID]uuid.UUID{r.aliceIn: r.alice, r.bobIn: r.bob} {
		if err := store.CreateMember(context.Background(), workspaceapp.MemberRow{
			ID: id, WorkspaceID: w.ID, MemberID: user, Role: shared.RoleAdmin, CreatedBy: r.alice, Now: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

// change is updateWorkspaceMember over members, with identity's profiles and
// the Authorizer as bootstrap wires them.
func (r adminRace) change(members workspaceapp.MemberUpdater) *workspaceapp.UpdateWorkspaceMember {
	return workspaceapp.NewUpdateWorkspaceMember(members, workspaceProfiles{profiles: identity.Provide(r.pool).PublicProfiles},
		access.New(access.Deps{WorkspaceRoles: workspace.Provide(r.pool).WorkspaceRoles}),
		postgres.NewTxManager(r.pool, 2*time.Second), clocktest.At(time.Now()))
}

// roles are alice's and bob's roles in acme.
func (r adminRace) roles(t *testing.T) (alice, bob shared.Role) {
	t.Helper()
	if err := r.pool.QueryRow(context.Background(),
		"SELECT (SELECT role FROM workspace_members WHERE id = $1), (SELECT role FROM workspace_members WHERE id = $2)", r.aliceIn, r.bobIn).
		Scan(&alice, &bob); err != nil {
		t.Fatal(err)
	}
	return alice, bob
}

func TestTwoAdminsDemotingEachOtherLeaveAnAdmin(t *testing.T) {
	for _, aliceFirst := range []bool{true, false} {
		name := "bob first"
		if aliceFirst {
			name = "alice first"
		}
		t.Run(name, func(t *testing.T) {
			r := newAdminRace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			first, second := r.alice, r.bob
			firstTarget, secondTarget := r.bobIn, r.aliceIn
			if !aliceFirst {
				first, second, firstTarget, secondTarget = second, first, secondTarget, firstTarget
			}
			g := newGate()
			demoted := run(func() error {
				_, err := r.change(gatedMembers{workspacepg.New(r.pool), g}).Execute(shared.WithActor(ctx, shared.Actor{UserID: first}),
					firstTarget, shared.RoleMember)
				return err
			})
			held(t, ctx, g, demoted, "the first demotion")
			refused := run(func() error {
				_, err := r.change(workspacepg.New(r.pool)).Execute(shared.WithActor(ctx, shared.Actor{UserID: second}),
					secondTarget, shared.RoleMember)
				return err
			})
			pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
			if alice, bob := r.roles(t); alice != shared.RoleAdmin || bob != shared.RoleAdmin {
				t.Errorf("while the first holds the lock: alice %d, bob %d; want both admins", alice, bob)
			}
			close(g.open)

			if err := result(t, ctx, demoted, "the first demotion"); err != nil {
				t.Errorf("the first demotion = %v, want it done", err)
			}
			if err := result(t, ctx, refused, "the second demotion"); !errors.Is(err, shared.Forbidden()) {
				t.Errorf("the second demotion = %v, want 403 forbidden", err)
			}
			wantAlice, wantBob := shared.RoleAdmin, shared.RoleMember
			if !aliceFirst {
				wantAlice, wantBob = shared.RoleMember, shared.RoleAdmin
			}
			if alice, bob := r.roles(t); alice != wantAlice || bob != wantBob {
				t.Errorf("alice %d, bob %d; want %d, %d: the first one's change, and an admin left", alice, bob, wantAlice, wantBob)
			}
		})
	}
}
````

- [ ] **Step 2: 测试、lint**

Run: `go -C server test -count=5 -race -run TestTwoAdminsDemotingEachOtherLeaveAnAdmin ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/bootstrap/interleaving_roles_test.go
```
```bash
git commit -m "test(M3/P2): two admins demoting each other leave an admin

Interleaving 2 of M3 design 9.3, both orders, on a real database through
the use case as bootstrap wires it: the first holds the workspace's lock
between its decision and its write, the second waits on the workspace row
and, deciding once the first has committed, is refused forbidden.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**：

| 改坏 | 必须失败的测试 |
|---|---|
| 不锁工作区 | `TestTwoAdminsDemotingEachOtherLeaveAnAdmin`（两个顺序：没有等待，两人都降为成员） |
| 按第一次读到的判定（在锁之前） | `TestTwoAdminsDemotingEachOtherLeaveAnAdmin` |
| 读、锁、再读挪到事务之前（锁随语句结束） | `TestTwoAdminsDemotingEachOtherLeaveAnAdmin` |
| `LockWorkspace` 换成 `FOR SHARE` | `TestTwoAdminsDemotingEachOtherLeaveAnAdmin` |

**Done when:** 两个顺序在真实数据库上 `-count=5 -race` 通过；四个变异都让它失败，不挂住。

---

### Task 14: 端到端 W8 的接口版本

**Files:**
- Create: `e2e/stories/workspace/w8-navigation-preferences.spec.ts`
- Modify: `e2e/fixtures/api.ts`、`e2e/fixtures/assert/workspace.ts`

**Interfaces:**
- Produces（spec 2.14，M3 设计 2 的 W8、9.6）：`e2e/fixtures/api.ts` 的 `WorkspacePreferences`、`WorkspacePreferencesUpdate`（生成的类型）；`e2e/fixtures/assert/workspace.ts` 的 `expectPreferences(db, slug, email, want | null) Promise<string | null>`：这个账户在这个工作区的设置行：`want` 为 `null` 时没有，否则恰好一行、未删除、值等于 `want`、由这个账户写入，答它的 id。
- 使用者：P9 的 W8 页面版本。

**Tests:**
- `e2e/stories/workspace/w8-navigation-preferences.spec.ts`：`W8 (API): the settings are the defaults and nothing is stored until the first change, which stores one row; later changes change it; each workspace has its own`：读到默认值、库里没有行；改成 `TABBED`、3：答案和库里的一行；再读相同；只改上限为 0：同一行，另一个字段不变；另一个工作区仍是默认值、没有行；不是成员的账户读到 404 `workspace.not_found`。
- 此前的 52 个故事不改而通过。

- [ ] **Step 1: 夹具、断言和故事**

`e2e/fixtures/api.ts`（修改，1 处）：

````old e2e/fixtures/api.ts
export type WorkspaceCreate = components["schemas"]["WorkspaceCreate"];
````

````new e2e/fixtures/api.ts
export type WorkspaceCreate = components["schemas"]["WorkspaceCreate"];
export type WorkspacePreferences = components["schemas"]["WorkspacePreferences"];
export type WorkspacePreferencesUpdate = components["schemas"]["WorkspacePreferencesUpdate"];
````

`e2e/fixtures/assert/workspace.ts`（修改，2 处）：

````old e2e/fixtures/assert/workspace.ts
import { expect } from "@playwright/test";

````

````new e2e/fixtures/assert/workspace.ts
import { expect } from "@playwright/test";

import type { WorkspacePreferences } from "../api";
````

````old e2e/fixtures/assert/workspace.ts
  expect(await countWorkspaces(db)).toEqual(before);
}

````

````new e2e/fixtures/assert/workspace.ts
  expect(await countWorkspaces(db)).toEqual(before);
}

/**
 * W8: the settings rows of the account of email in the workspace of slug:
 * none while want is null, else exactly one, undeleted, holding want and
 * written by that account. Returns the row's id, or null.
 */
export async function expectPreferences(
  db: Database,
  slug: string,
  email: string,
  want: WorkspacePreferences | null
): Promise<string | null> {
  const rows = await db.query<{ id: string }>(
    `SELECT p.id, p.navigation_control_preference, p.navigation_project_limit, p.deleted_at,
            p.created_by_id = u.id AND p.updated_by_id = u.id AS written_by_the_account
       FROM workspace_user_properties p
       JOIN workspaces w ON w.id = p.workspace_id
       JOIN users u ON u.id = p.user_id
      WHERE w.slug = $1 AND u.email = $2`,
    [slug, email]
  );
  if (want === null) {
    expect(rows, `the settings of ${email} in ${slug}`).toEqual([]);
    return null;
  }
  expect(rows, `the settings of ${email} in ${slug}`).toEqual([
    { ...want, id: expect.any(String), deleted_at: null, written_by_the_account: true },
  ]);
  return rows[0]?.id ?? null;
}

````

`e2e/stories/workspace/w8-navigation-preferences.spec.ts`（新文件，77 行）：

````file e2e/stories/workspace/w8-navigation-preferences.spec.ts
import {
  createWorkspace,
  slugFor,
  type Api,
  type WorkspacePreferences,
  type WorkspacePreferencesUpdate,
} from "../../fixtures/api";
import { expectPreferences } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// W8, the project navigation's settings (M3 design 2, 3.18). The page
// version comes with the sidebar's "project navigation" dialog (P9).

const defaults: WorkspacePreferences = { navigation_control_preference: "ACCORDION", navigation_project_limit: 10 };

/** The answer of GET /api/v0/me/workspaces/{slug}/preferences, which must be 200. */
async function read(api: Api, token: string, slug: string): Promise<unknown> {
  const { data, error, response } = await api.GET("/api/v0/me/workspaces/{slug}/preferences", {
    params: { path: { slug } },
    headers: bearer(token),
  });
  expect(response.status, `read the settings in ${slug}: ${JSON.stringify(error)}`).toBe(200);
  return data;
}

/** The answer of PATCH /api/v0/me/workspaces/{slug}/preferences, which must be 200. */
async function change(api: Api, token: string, slug: string, body: WorkspacePreferencesUpdate): Promise<unknown> {
  const { data, error, response } = await api.PATCH("/api/v0/me/workspaces/{slug}/preferences", {
    params: { path: { slug } },
    body,
    headers: bearer(token),
  });
  expect(response.status, `change the settings in ${slug}: ${JSON.stringify(error)}`).toBe(200);
  return data;
}

test("W8 (API): the settings are the defaults and nothing is stored until the first change, which stores one row; later changes change it; each workspace has its own", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = (await createPAT(api, (await register(api, email)).access_token)).token;
  const slug = slugFor(testInfo);
  const other = slugFor(testInfo, "other");
  await createWorkspace(api, pat, { name: "Acme", slug });
  await createWorkspace(api, pat, { name: "Other", slug: other });

  expect(await read(api, pat, slug)).toEqual(defaults);
  await expectPreferences(db, slug, email, null);

  const tabbed: WorkspacePreferences = { navigation_control_preference: "TABBED", navigation_project_limit: 3 };
  expect(await change(api, pat, slug, tabbed)).toEqual(tabbed);
  const row = await expectPreferences(db, slug, email, tabbed);
  // After a refresh the page reads them again.
  expect(await read(api, pat, slug)).toEqual(tabbed);

  // Showing every project: one field changes, the other stays, on the same row.
  expect(await change(api, pat, slug, { navigation_project_limit: 0 })).toEqual({
    ...tabbed,
    navigation_project_limit: 0,
  });
  expect(await expectPreferences(db, slug, email, { ...tabbed, navigation_project_limit: 0 })).toBe(row);

  // The other workspace keeps the defaults, and nothing is stored for it.
  expect(await read(api, pat, other)).toEqual(defaults);
  await expectPreferences(db, other, email, null);

  // Another account, not a member, reads nothing there.
  const stranger = (await register(api, emailFor(testInfo, "stranger"))).access_token;
  const refused = await api.GET("/api/v0/me/workspaces/{slug}/preferences", {
    params: { path: { slug } },
    headers: bearer(stranger),
  });
  expect(refused.response.status).toBe(404);
  expect(refused.error?.code).toBe("workspace.not_found");
});
````

- [ ] **Step 2: 检查和端到端**

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
Expected: 53 个全部通过（此前的 52 个、W8）。

- [ ] **Step 3: 提交**

```bash
git add e2e/fixtures/api.ts e2e/fixtures/assert/workspace.ts e2e/stories/workspace/w8-navigation-preferences.spec.ts
```
```bash
git commit -m "test(M3/P2): W8 through the API

The project navigation's settings: the defaults and nothing stored until
the first change, one row after it, changed in place by later ones, each
workspace its own, and nothing for an account that is not a member.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**变异**：W8 单独运行时看得到 `Preferences` 不看 `workspace_id`（另一个工作区读到 `TABBED`）和 `GET` 写库（没有行的断言）；不看 `user_id` 要第二个成员，P2 没有加成员的接口（邀请在 P3），由 Task 7 的存储测试和 Task 8 的矩阵看到（spec 第 6 节）。

**Done when:** `make e2e` 53 个全部通过。

---

### Task 15: 上级文档和交接

**Files:**
- Modify: `docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md`、`docs/v0/plane-diff.md`、`docs/v0/v0-design.md`

**Interfaces:**
- Produces（spec 2.15，M3 设计 3.20 中 P2 的各行）：
  - `docs/v0/v0-design.md` 3.6：关联字段的名字带 `_id`；例外：工作区成员内嵌成员的公开资料（`MemberUser`）；4.2：加锁的全局顺序和六条约定，"先锁父行，再判定"从 M3/P2 起落地；
  - `docs/v0/plane-diff.md`：二·按表 `workspace_user_properties` 的五行；三：关联字段带 `_id`（P1 的移交，spec 第 3 节第 2 条表的第 3 行）、显示设置的路径；四：显示设置的 `GET` 不写库、删除工作区的连带；
  - `docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md`：追加"处理结果（M3/P2）"（侧边栏偏好的接口一侧、个人主页的数据来源），状态保持 `open`。
- 使用者：P3 起的各 Phase；P2 的 review。

**Tests:** 关键词守卫（`make lint-web`）也查文档。

- [ ] **Step 1: 文档**

`docs/v0/v0-design.md`（修改，2 处）：

````old docs/v0/v0-design.md
- **关联对象只返回 ID**：以后按需增加 `?expand=`。
````

````new docs/v0/v0-design.md
- **关联对象只返回 ID**：以后按需增加 `?expand=`。关联字段的名字带 `_id`（例如 `WorkspaceMember.workspace_id`）。
  - **例外：工作区成员内嵌成员的公开资料**。`WorkspaceMember.member` 是 `MemberUser`（`id`、显示名、名字、头像、邮箱）：v0 没有读别的用户的接口，成员列表是看到同事名字和头像的唯一来源（Plane 同样内嵌）。邮箱按调用者的角色给出，访客看到 `null`（M3 设计 5.2、11.3）。第一个是 M3/P2 的 `listWorkspaceMembers`。
````

````old docs/v0/v0-design.md
- **账户行锁**：签发和变更凭证的事务（登录、创建 PAT、修改密码、停用）先锁账户行，再确认调用者的凭证仍然有效；服务器管理员改动已有账户的命令（`reset-password`、`set-email`、`deactivate`、`activate`）按邮箱锁同一行。所以并发的修改不会让已被撤销的凭证再签发或变更凭证；续期靠会话行上的条件更新（M2 设计 3.5）。
````

````new docs/v0/v0-design.md
- **账户行锁**：签发和变更凭证的事务（登录、创建 PAT、修改密码、停用）先锁账户行，再确认调用者的凭证仍然有效；服务器管理员改动已有账户的命令（`reset-password`、`set-email`、`deactivate`、`activate`）按邮箱锁同一行。所以并发的修改不会让已被撤销的凭证再签发或变更凭证；续期靠会话行上的条件更新（M2 设计 3.5）。
- **加锁顺序**（M3 设计 3.6）：全局顺序是 `users` → `profiles` → `auth_sessions` → `api_tokens` → `workspaces` → `workspace_member_invites` → `workspace_members` → `workspace_user_properties` → `projects` → `project_members` → `project_user_properties` → `states` → `labels`，同一张表的多行按 `id` 升序。另有六条约定，后续里程碑的表照做：
  1. 锁账户行的事务最先锁它；事务中途要读别的账户（例如邮箱），只经不加锁的 `MemberProfiles`。
  2. **写事务先锁父行，再判定权限**：父行是判定所在的那一行（工作区级是工作区行，项目级是项目行），锁它的语句带 `deleted_at IS NULL`；成员关系和角色的改变、维护项目范围不变式的写、修改父行本身的写用 `FOR NO KEY UPDATE`，其余的写用 `FOR SHARE`。按资源寻址的写先读资源行得到父行，锁住父行之后重读。
  3. 让某人成为项目成员的写，先以 `FOR SHARE` 锁住他的工作区成员行；目标的 422 在判定之后给出。
  4. M3 的锁之后可以插入引用别的账户的行。
  5. 一条语句批量改子行时按扫描顺序加锁（都在父行的 `FOR NO KEY UPDATE` 之下）；只持父行的共享锁而批量插入时，按唯一键的顺序插入。
  6. 成员关系集合的增长先以 `FOR SHARE` 锁住得到它的账户行，收缩先改工作区成员行、再列举他的项目成员关系；恢复以前的成员关系不比新授予给得更多，也不比原来那一行更多。

  "先锁父行，再判定"从 M3/P2 起落地（修改、删除工作区，改成员的角色，显示设置）：两位管理员互相降级时，先拿到工作区锁的一方成功，后到的一方判定时已是成员而被拒绝。约定六的接受邀请一段在 M3/P3、停用一段在 M3/P6 随实现核对。
````

`docs/v0/plane-diff.md`（修改，3 处）：

````old docs/v0/plane-diff.md
| `workspace_members` | 删除 `issue_props`、`company_role`、`getting_started_checklist`、`tips`、`explored_features` | 前端不读不写；公司角色随 M2 删掉的新手引导步骤没有了写入方；后三项是 Plane 已砍功能的状态 |
````

````new docs/v0/plane-diff.md
| `workspace_members` | 删除 `issue_props`、`company_role`、`getting_started_checklist`、`tips`、`explored_features` | 前端不读不写；公司角色随 M2 删掉的新手引导步骤没有了写入方；后三项是 Plane 已砍功能的状态 |
| `workspace_user_properties` | 14 列保留 10 列（M3/P2，`00008_workspace_workspace_user_properties.sql`） | M3 设计 4.5 |
| `workspace_user_properties` | `workspace_id`、`user_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`） |
| `workspace_user_properties` | `navigation_project_limit`：新加 `DEFAULT 10`、`CHECK (navigation_project_limit >= 0)`；`navigation_control_preference`：新加 `DEFAULT 'ACCORDION'`、`CHECK (navigation_control_preference IN ('ACCORDION', 'TABBED'))`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值和 choices；0 表示显示全部项目 |
| `workspace_user_properties` | 部分唯一索引照搬，改名为 `workspace_user_properties_workspace_id_user_id_key ON (workspace_id, user_id) WHERE deleted_at IS NULL`（Plane 叫 `workspace_user_properties_unique_workspace_user_when_deleted_at`）；新加不带条件的 `workspace_user_properties_workspace_id_idx ON (workspace_id)`；`user_id` 不单独建索引 | 显示设置的写入经这个索引 `ON CONFLICT`（M3 设计 3.18）；物理级联要不带条件的索引（M3 设计 4）；按账户查的都带着工作区 |
| `workspace_user_properties` | 删除 `filters`、`display_filters`、`display_properties`、`rich_filters` | 工作项列表的筛选和显示列由它们的使用者 M4 按自己的格式加回（M3 设计 3.18） |
````

````old docs/v0/plane-diff.md
| 集合型的列表（工作区、成员、项目等） | 一次返回全部，响应是裸数组 | 同样一次返回全部、不分页，响应是 `{"data": [...]}` 封套，每个列表写明顺序（M3 设计 3.12） |
````

````new docs/v0/plane-diff.md
| 集合型的列表（工作区、成员、项目等） | 一次返回全部，响应是裸数组 | 同样一次返回全部、不分页，响应是 `{"data": [...]}` 封套，每个列表写明顺序（M3 设计 3.12） |
| 关联字段 | 名字不带 `_id`（`workspace`、`member`、`project_lead`），有的内嵌对象 | 只给 id，名字带 `_id`（`WorkspaceMember.workspace_id`）；唯一的例外是工作区成员内嵌成员的公开资料 `member`（`MemberUser`，v0-design 3.6；M3 设计 5.2） |
| 工作区的显示设置的路径 | `/api/workspaces/{slug}/user-properties/`（与工作项的筛选合在一起） | `GET`、`PATCH /api/v0/me/workspaces/{slug}/preferences`，只有项目导航的两项（M3 设计 3.18）；没有 `/sidebar-preferences/` |
````

````old docs/v0/plane-diff.md
| 关闭创建工作区时 | 实例管理员在管理后台为自己建工作区 | 服务器管理员用 `nerve workspaces create --slug --name --admin-email` 建，不受开关限制，`--admin-email` 的账户是它的管理员（M3 设计 3.11） |
````

````new docs/v0/plane-diff.md
| 关闭创建工作区时 | 实例管理员在管理后台为自己建工作区 | 服务器管理员用 `nerve workspaces create --slug --name --admin-email` 建，不受开关限制，`--admin-email` 的账户是它的管理员（M3 设计 3.11） |
| 工作区的显示设置 | `GET` 时 `get_or_create`：读取就建行 | `GET` 不写库，没有行时返回默认值（`ACCORDION`、10）；第一次修改时经部分唯一索引 `INSERT … ON CONFLICT` 建行（M3 设计 3.18） |
| 删除工作区 | 清掉别人的 `last_workspace_id`；成员、显示设置等的连带软删除由 Celery 异步完成 | 不清 `last_workspace_id`（登录后的落点规则让它无害，M3 设计 3.14）；工作区、成员、显示设置在同一个事务里、同一时刻软删除，删除之后 slug 立即可以重用。邀请由 M3/P3、项目由 M3/P4、标签由 M3/P7 加入这个事务 |
````

`docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md`（修改，1 处）：

````old docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md
来源：[M3/P1 spec](../specs/P1-platform.md) 第 7 节。

````

````new docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md
来源：[M3/P1 spec](../specs/P1-platform.md) 第 7 节。

## 处理结果（M3/P2）

- **侧边栏偏好**（接口一侧完成）：没有 `/sidebar-preferences/`。项目导航偏好是 `GET`、`PATCH /api/v0/me/workspaces/{slug}/preferences`，结构 `WorkspacePreferences {navigation_control_preference, navigation_project_limit}`，任何有效成员读写自己的；没有这一行时 `GET` 返回默认值（`ACCORDION`、10），不写库，第一次修改时建行（M3 设计 3.18）。W8 的接口版本核对这两件。`ProjectNavigationDialog` 改调它在 P9（W8 的页面版本）。
- **个人主页的数据来源**：`listWorkspaceMembers` 在权限矩阵中有了行：管理员、成员、访客都得到 200，含已离开的成员（`is_active: false`），访客看到的邮箱都是 `null`（M3 设计 9.2）。页面一侧在 P9。

仍未处理，状态保持 `open`：保留名单的前端一侧（P8）；项目字段（P4）；个人主页的页面（P9）。

来源：[M3/P2 spec](../specs/P2-workspaces.md) 第 7 节。

````

- [ ] **Step 2: 检查**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过。

- [ ] **Step 3: 提交**

```bash
git add docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md docs/v0/plane-diff.md docs/v0/v0-design.md
```
```bash
git commit -m "docs(M3/P2): the P2 rows of the design docs; the M1-P2 handoff's result

v0-design gains the exception to 'relations are ids' (a membership embeds
its member's public profile) and the lock order with its six conventions;
plane-diff carries workspace_user_properties and P2's behaviour rows; the
M1-P2 handoff records what P2 closes.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** M3 设计 3.20 中 P2 的各行（总体设计 3.6、4.2，差异清单）写好；M1-P2 交接有"处理结果（M3/P2）"。
