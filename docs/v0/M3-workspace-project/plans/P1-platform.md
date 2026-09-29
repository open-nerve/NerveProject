# M3/P1 权限框架、组合与建工作区 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 权限框架（`shared.Authorizer`、平台码 `forbidden`、`access` 模块的判定和规则表、操作名的完整性）和两段组合定下；`workspace` 模块的第一片可用：任何调用方都能建、列、看工作区、查 slug，服务器管理员能用 `nerve workspaces create` 建工作区；建工作区按 M3 设计 3.6 约定六最先 `FOR SHARE` 锁账户行、锁下要求账户有效，与 M2 的停用两个顺序都在真实数据库上测过；权限矩阵的骨架和本 Phase 的 30 格通过；W1、W10 的接口版本通过。

**Architecture:** `shared` 加权限的值和端口（`Role`、`Action`、`Target`、`Grant`、`Authorizer`、`ErrNotVisible`）、平台码 `forbidden` 和从 `identity/domain` 移来的三条取值规则。新模块 `access`：`domain` 是规则表和纯函数 `Decide`，`app` 是 `Authorizer`（每次调用经 `WorkspaceRoles` 端口读调用者的角色，不缓存），模块入口只有 `New` 和 `RuleKeys`。新模块 `workspace` 按端口与适配器分层：`domain`（校验、保留名单、操作名、错误）、`app`（四个用例，`createWorkspace` 经 `Accounts` 端口锁账户行）、`adapter/postgres`（sqlc 的查询、存储、`ActiveRole`）、`adapter/http`；模块入口有 `Provide`（只用连接池，交出 `WorkspaceRoles`）、`New`、`NewAdmin`、`Actions`、`ReservedSlugs`。`identity` 加 `Provide`（交出 `Accounts`：`FOR SHARE` 锁账户行、交回状态）。`bootstrap` 按 M3 设计 6.6 两段组合：`identity.Provide` → `workspace.Provide` → `access.New` → `workspace.New` → `identity.New`，`identity` 的值在 `bootstrap/ports.go` 转成 `workspace` 的值；命令行的 `Workspaces` 只组合连接池和 `workspace.NewAdmin`。

**Tech Stack:** Go 1.27.1、pgx v5.11.0、sqlc v1.31.1（`CGO_ENABLED=0`）、oapi-codegen v2.8.0、goose v3.28.0、River v0.47.0、golangci-lint 2.13.2、PostgreSQL 18.6（testcontainers）；Node 24、pnpm 11.10.0、Playwright 1.63.0。

**Spec:** `docs/v0/M3-workspace-project/specs/P1-platform.md`（上级：`docs/v0/M3-workspace-project/M3-design.md`）

## Global Constraints

- **Go 版本和依赖**：本 plan 不执行 `go get`，不加任何 Go 模块或 npm 包（M3 设计 6.1）。`server/go.mod` 和 `server/tools/go.mod` 保持 `go 1.27` 和 `toolchain go1.27.1`（golangci-lint 2.13.2 由 go1.27.0 构建，`go` 行更高就拒绝运行）。每个 Task 提交前执行 `grep -n "^go \|^toolchain" server/go.mod server/tools/go.mod`，四行必须是 `go 1.27` 和 `toolchain go1.27.1`；`git diff --stat 1f0e7ed7 -- server/go.mod server/go.sum server/tools pnpm-lock.yaml` 在本 plan 的任何时刻都没有输出。
- **每个 Task 提交前**：`make lint-go` 两段都输出 `0 issues.`，`make test` 全部通过（没有 `FAIL`）。
- **生成的文件不手写、不从本 plan 复制**：有生成物的 Task（3、8、9、11、14）执行 Task 中的生成命令（`make gen`，或只动了 sqlc 的 Task 8、9 执行 `make gen-go`），用 `shasum -a 256` 和 `wc -l` 核对表中的 SHA-256 和行数，对不上时停下来：说明某个输入与本 plan 不一致。这些 Task **提交之后**执行 `make gen-check`（它用 `git status` 判断生成物是否已提交，提交之前必然报差异）。
- **前端检查**：改了接口描述、前端或 `e2e/` 的 Task（3、11、14、16）另执行 `make lint-web`、`make knip`、`make test-web`；Task 16 执行 `make e2e`。
- **容器**：`make test` 和 `make e2e` 用自己的 testcontainers。开发库 `nerve-dev-db-1` 可以用，但不要停止或重建它，不要执行 `make dev-db-down`、`make dev-db-reset`。不要碰其他项目的容器（`agentforge-*`、`plane-app-*`、`opennerve-*`）。
- **git**：每次 Bash 调用只执行一个 git 命令，不用 `;`、`&&`、`|` 串联 git；不用 `git -C`、`stash`、`clean`、`reset --hard`。`cd` 不与别的命令组合。不碰 `plane/`、`refer/`。
- **安装**：除了 Docker、Go、Node 不做任何全局安装；不执行 `corepack enable`（pnpm 已在 PATH 上）。
- **规则**：不写 `init()`，不用全局可变状态，构造函数显式传入依赖；不建 `utils`、`common`、`helpers` 包；一个文件只做一件事，不超过约 400 行；依赖只能向内；模块之间不互相导入，值在 `bootstrap` 中转换（M3 设计 6.5、6.6）；平台包之间不互相导入；不留没有使用者的代码。**例外**：接口描述按模块一个文件（M0-P3 交接 5），`api/modules/*.yaml` 不受约 400 行的限制。本 plan 的其余文件都在 400 行以内（最长的是 `permission_matrix_test.go`，321 行）。
- **注释**：Go、TS 代码、SQL 查询和接口描述用英文；迁移文件、`reserved_slugs.txt`、`server/configs/*.yaml` 的中文注释和中文文档照本 plan 原样。
- **代码块**：每个改动都写成四个反引号围起来的块，块的第一行写明种类和路径，照原样使用（原型中逐字节运行过）：
  - ````` ````file <路径> ````` 新文件，块的内容加一个结尾换行就是整个文件；
  - ````` ````whole <路径> ````` 已有文件的完整新内容（同样加结尾换行）；
  - ````` ````old <路径> ````` 与紧跟着的 ````` ````new <路径> `````：`old` 的文字在文件当前版本中恰好出现一次，把它换成 `new` 的文字；
  - ````` ````delete <路径> ````` 删除这个文件（块是空的）。
  
  一个文件的几个块按出现的顺序依次应用。拼 plan 的脚本已从 `1f0e7ed7` 起按顺序核对过全部块：每个 `old` 恰好出现一次，每个新文件原来不存在，逐 Task 应用之后的文件与原型逐字节相同（spec 附录 A）。
- **过渡版本**：少数文件先在较早的 Task 写成过渡版本，较晚的 Task 再修改（`api/modules/workspace.yaml`、`workspace/module.go`、`shared/error.go`、生成物）；每个过渡版本都在逐 Task 复现中运行过。
- **提交**：提交信息用英文，末尾加一行：`Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`
- 所有命令在仓库根目录下执行，除非步骤中另有说明。

## 文件结构

路径相对于仓库根目录。"生成"表示由命令生成并提交；一个文件由多个 Task 修改时，"Task"一列都列出。

| 文件 | 职责 | Task |
|---|---|---|
| `server/migrations/sql/00006_workspace_workspaces.sql`、`server/migrations/sql/00007_workspace_workspace_members.sql` | 两张表（M3 设计 4.2、4.3） | 1 |
| `deploy/runtime-grants.sql`（修改） | 运行时角色对两张表的权限 | 1 |
| `server/migrations/schema_test.go`（修改） | 7 个迁移 up、down、再 up；约束和索引的名字；CHECK 的反例 | 1 |
| `server/internal/archtest/sqlc_test.go`、`server/internal/archtest/sqlc_cases_test.go`（修改） | `TestSQLCSchemaScope` 的四种写法、带引号的名字、改名 | 2 |
| `server/internal/shared/authorize.go`、`server/internal/shared/authorize_test.go` | `Role`、`Action`、`Target`、`Grant`、`Authorizer`、`ErrNotVisible` | 3 |
| `server/internal/shared/error.go`、`server/internal/shared/error_test.go`（修改） | `CodeForbidden`、`Forbidden()`；包说明 | 3、4 |
| `server/internal/platform/httpserver/apitest/problems.go`、`server/internal/platform/httpserver/apitest/rules_test.go`、`server/internal/platform/httpserver/apitest/rules_cases_test.go`（修改） | `forbidden` 是平台码；前缀是一个模块 | 3 |
| `api/common.yaml`（修改） | `Problem.code` 的说明 | 3 |
| `server/internal/platform/httpserver/apigen/components.gen.go`（生成） | | 3 |
| `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`（生成） | | 3、11、14 |
| `server/internal/shared/email.go`、`server/internal/shared/email_test.go`、`server/internal/shared/url.go`、`server/internal/shared/url_test.go`、`server/internal/shared/timezone.go`、`server/internal/shared/timezone_test.go` | 三条取值规则（M3 设计 3.13） | 4 |
| `server/internal/modules/identity/domain/email.go`、`server/internal/modules/identity/domain/email_test.go`（完整内容）、`server/internal/modules/identity/domain/user.go`（修改）；`server/internal/modules/identity/domain/url.go`、`server/internal/modules/identity/domain/url_test.go`（删除） | `identity` 改用 `shared` 的规则 | 4 |
| `server/internal/modules/identity/app/activate.go`、`server/internal/modules/identity/app/deactivate.go`、`server/internal/modules/identity/app/login.go`、`server/internal/modules/identity/app/reset_password.go`、`server/internal/modules/identity/app/set_email.go`、`server/internal/modules/identity/adapter/http/limits.go`（修改） | 调用方改调 `shared.NormalizeEmail` | 4 |
| `server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`、`server/internal/modules/access/domain/decide.go`、`server/internal/modules/access/domain/decide_test.go` | 规则表；判定 | 5 |
| `server/internal/modules/access/app/ports.go`、`server/internal/modules/access/app/authorizer.go`、`server/internal/modules/access/app/authorizer_test.go`、`server/internal/modules/access/module.go` | `WorkspaceRoles`；`Authorizer`；`New`、`RuleKeys` | 6 |
| `server/internal/modules/workspace/domain/workspace.go`、`server/internal/modules/workspace/domain/workspace_test.go`、`server/internal/modules/workspace/domain/reserved.go`、`server/internal/modules/workspace/domain/reserved_slugs.txt`、`server/internal/modules/workspace/domain/reserved_test.go`、`server/internal/modules/workspace/domain/actions.go`、`server/internal/modules/workspace/domain/errors.go` | 工作区的规则、保留名单、操作名、错误 | 7 |
| `server/internal/modules/identity/adapter/postgres/queries/users.sql`（修改）、`server/internal/modules/identity/adapter/postgres/accounts.go`、`server/internal/modules/identity/adapter/postgres/accounts_test.go`、`server/internal/modules/identity/app/accounts.go`、`server/internal/modules/identity/provide.go` | `ShareAccount`、`ShareAccountByEmail`；`identity.Provide` | 8 |
| `server/internal/modules/identity/adapter/postgres/gen/users.sql.go`（生成） | | 8 |
| `server/sqlc.yaml`（修改） | `workspace` 的 sqlc 条目 | 9 |
| `server/internal/modules/workspace/app/ports.go` | 用例的端口 | 9 |
| `server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql`、`server/internal/modules/workspace/adapter/postgres/queries/members.sql`、`server/internal/modules/workspace/adapter/postgres/store.go`、`server/internal/modules/workspace/adapter/postgres/workspaces.go`、`server/internal/modules/workspace/adapter/postgres/roles.go`、`server/internal/modules/workspace/adapter/postgres/store_test.go` | 查询、存储、`ActiveRole` | 9 |
| `server/internal/modules/workspace/adapter/postgres/gen/db.go`、`server/internal/modules/workspace/adapter/postgres/gen/models.go`、`server/internal/modules/workspace/adapter/postgres/gen/members.sql.go`、`server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go`（生成） | | 9 |
| `server/internal/modules/workspace/app/create_workspace.go`、`server/internal/modules/workspace/app/list_workspaces.go`、`server/internal/modules/workspace/app/get_workspace.go`、`server/internal/modules/workspace/app/check_slug.go`、`server/internal/modules/workspace/app/fakes_test.go`、`server/internal/modules/workspace/app/create_workspace_test.go`、`server/internal/modules/workspace/app/read_test.go` | 四个用例 | 10 |
| `api/modules/workspace.yaml`；`api/openapi.yaml`（修改） | 接口描述 | 11、14 |
| `server/internal/modules/workspace/adapter/http/gen/oapi-codegen.yaml`、`server/internal/modules/workspace/adapter/http/handler.go`、`server/internal/modules/workspace/adapter/http/workspaces.go`、`server/internal/modules/workspace/adapter/http/handler_test.go`、`server/internal/modules/workspace/adapter/http/workspaces_test.go` | HTTP 一侧 | 11 |
| `server/internal/modules/workspace/adapter/http/gen/server.gen.go`、`server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go`（生成） | | 11 |
| `server/internal/modules/workspace/module.go` | `Provide`、`New`、`Register`；`Actions`、`ReservedSlugs` | 11、12 |
| `server/internal/bootstrap/app.go`（修改）、`server/internal/bootstrap/ports.go`、`server/internal/bootstrap/ports_test.go`、`server/internal/bootstrap/workspace_test.go` | 两段组合；`identity` 到 `workspace` 的转换 | 11 |
| `web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`（修改） | 三个新码的文案（spec 第 3 节第 4 条） | 11 |
| `server/internal/bootstrap/actions_test.go`、`server/internal/bootstrap/reserved_test.go` | 操作名的完整性；保留名单的"服务端"一段 | 12 |
| `server/internal/bootstrap/interleaving_test.go` | 建工作区与停用，两个顺序 | 13 |
| `server/internal/modules/workspace/admin.go`、`server/internal/bootstrap/workspaces.go`、`server/internal/bootstrap/workspaces_test.go`；`server/internal/bootstrap/commands.go`、`server/internal/bootstrap/users.go`（修改） | 命令行的最小组合；两个命令共用的错误转换 | 14 |
| `server/cmd/nerve/workspaces.go`、`server/cmd/nerve/workspaces_test.go`；`server/cmd/nerve/commands.go`（修改） | `nerve workspaces create` | 14 |
| `server/internal/archtest/composition_test.go`（修改） | 两个命令的组合都不建服务器、任务和 `Authorizer` | 14 |
| `README.md`、`server/configs/config.yaml`、`server/internal/platform/config/config.go`（修改） | 部署一节；开关的说明 | 14 |
| `server/internal/platform/postgres/pgtest/pgtest.go`、`server/internal/platform/postgres/pgtest/pgtest_test.go`（修改） | `NewDatabaseFrom` | 15 |
| `server/internal/platform/httpserver/apitest/operations.go`、`server/internal/platform/httpserver/apitest/operations_test.go`（修改） | `Operation.ID`、`Operation.Tags` | 15 |
| `server/internal/bootstrap/permission_matrix_test.go` | 权限矩阵 | 15 |
| `e2e/fixtures/api.ts`（修改）、`e2e/fixtures/assert/workspace.ts`、`e2e/fixtures/workspaces.ts` | 端到端的夹具和断言 | 16 |
| `e2e/stories/workspace/w1-create-workspace.spec.ts`、`e2e/stories/workspace/w10-admin-create-workspace.spec.ts` | W1、W10 的接口版本 | 16 |
| `docs/v0/v0-design.md`、`docs/v0/plane-diff.md`、`docs/v0/M2-auth/M2-design.md`（修改） | M3 设计 3.20 中 P1 的各行 | 16 |
| `docs/v0/M3-workspace-project/handoffs/M2-closeout.md`、`docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md`、`docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md`、`docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md`（修改） | 追加"处理结果（M3/P1）" | 16 |

---

### Task 1: 迁移 `00006`、`00007` 与运行时权限

**Files:**
- Create: `server/migrations/sql/00006_workspace_workspaces.sql`、`server/migrations/sql/00007_workspace_workspace_members.sql`
- Modify: `deploy/runtime-grants.sql`、`server/migrations/schema_test.go`

**Interfaces:**
- Produces（spec 2.3，M3 设计 4.2、4.3）：表 `workspaces`、`workspace_members`，约束和索引的名字 `workspaces_{pkey, name_check, slug_check, organization_size_check, slug_key, created_by_id_fkey, updated_by_id_fkey}`、`workspace_members_{pkey, role_check, workspace_id_fkey, member_id_fkey, created_by_id_fkey, updated_by_id_fkey, workspace_id_member_id_key, member_id_idx, workspace_id_idx}`；`nerve_runtime` 对两张表的 `SELECT, INSERT, UPDATE, DELETE`。
- 使用者：Task 9 的查询和存储（`workspaces_slug_key` 由存储映射成 `workspace.slug_taken`）；Task 15 的矩阵。
- `server/sqlc.yaml` 的 `workspace` 条目不在这里：sqlc 对没有查询的条目报错，条目随 Task 9 的第一批查询加入（spec 第 3 节第 2 条）。

**Tests:**（`server/migrations/schema_test.go`，真实数据库）
- `TestMigrationsGoUpDownAndUpAgain`：7 个迁移；up 之后有两张新表，down 之后没有，再 up 仍是 7 个。
- `TestConstraintAndIndexNames`：查询覆盖两张新表，期望加 18 行（每个外键带着它的 `ON DELETE`：`f c` 是 `CASCADE`，`f n` 是 `SET NULL`）。
- `TestChecksRejectCounterexamples`：合法的一个工作区（slug `acme_1-2`、规模 `500+`）和一个成员（角色 20）插入成功；7 个反例各被它的 CHECK 拒绝：空名称、大写的 slug、带点的 slug、空 slug、规模 `1000+`、角色 10、角色 0。
- 不改而覆盖：`TestTheGrantsFileCoversEveryRelationAndFunction`（`bootstrap/runtime_role_test.go`）要求每张表都在 `deploy/runtime-grants.sql` 中。

- [ ] **Step 1: 两个迁移和权限**

`server/migrations/sql/00006_workspace_workspaces.sql`（新文件，26 行）：

````file server/migrations/sql/00006_workspace_workspaces.sql
-- workspaces：Plane 的 workspaces 表（14 列）按 M3 设计 4.2 保留 10 列。约定见 M2 设计 3.13：
-- id 由应用生成（uuid.NewV7），created_at、updated_at 由应用按用例的时钟写入，DEFAULT now() 只是兜底。

-- +goose Up
CREATE TABLE workspaces (
    id uuid PRIMARY KEY,
    -- "至少一个字母或数字、不含网址"在领域层（M3 设计 3.10）
    name varchar(80) NOT NULL CHECK (name <> ''),
    -- 只有小写（3.10）；保留名在领域层
    slug varchar(48) NOT NULL CHECK (slug ~ '^[a-z0-9_-]+$'),
    -- 前端的选项（web/packages/constants/src/workspace.ts:10）
    organization_size varchar(20)
        CHECK (organization_size IN ('Just myself', '2-10', '11-50', '51-200', '201-500', '500+')),
    -- 取值在领域层校验（3.13）
    timezone varchar(255) NOT NULL DEFAULT 'UTC',
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
-- Plane 的全表唯一 workspace_slug_key 改为部分唯一：删除之后 slug 可以再用，不改名（3.10）
CREATE UNIQUE INDEX workspaces_slug_key ON workspaces (slug) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE workspaces;
````

`server/migrations/sql/00007_workspace_workspace_members.sql`（新文件，26 行）：

````file server/migrations/sql/00007_workspace_workspace_members.sql
-- workspace_members：Plane 的 workspace_members 表（17 列）按 M3 设计 4.3 保留 10 列。
-- 移出、离开、停用都只把 is_active 改为假，行不删除（Plane 相同）。

-- +goose Up
CREATE TABLE workspace_members (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    member_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
    -- 访客 5、成员 15、管理员 20（Plane 的 ROLE_CHOICES；快照的 role >= 0 收紧为三个取值）
    role smallint NOT NULL DEFAULT 5 CHECK (role IN (5, 15, 20)),
    is_active boolean NOT NULL DEFAULT true,
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX workspace_members_workspace_id_member_id_key ON workspace_members (workspace_id, member_id)
    WHERE deleted_at IS NULL;
-- 我的工作区、停用（4.3）
CREATE INDEX workspace_members_member_id_idx ON workspace_members (member_id) WHERE deleted_at IS NULL;
-- 物理级联（M4 的 60 天清理）按它找子行（4 节开头）
CREATE INDEX workspace_members_workspace_id_idx ON workspace_members (workspace_id);

-- +goose Down
DROP TABLE workspace_members;
````

`deploy/runtime-grants.sql`（修改，1 处）：

````old deploy/runtime-grants.sql
GRANT SELECT, INSERT, UPDATE, DELETE ON users, profiles, auth_sessions, api_tokens TO nerve_runtime;
````

````new deploy/runtime-grants.sql
GRANT SELECT, INSERT, UPDATE, DELETE ON users, profiles, auth_sessions, api_tokens TO nerve_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON workspaces, workspace_members TO nerve_runtime;
````

- [ ] **Step 2: 迁移的测试**

`server/migrations/schema_test.go`（修改，10 处）：

````old server/migrations/schema_test.go
	if err != nil || len(up) != 5 {
		t.Fatalf("Up() = %d migrations, %v; want 5", len(up), err)
````

````new server/migrations/schema_test.go
	if err != nil || len(up) != 7 {
		t.Fatalf("Up() = %d migrations, %v; want 7", len(up), err)
````

````old server/migrations/schema_test.go
		{tablesQuery, []string{"api_tokens", "auth_sessions", "profiles", "river_job", "river_leader", "river_notification", "river_queue", "users"}},
````

````new server/migrations/schema_test.go
		{tablesQuery, []string{"api_tokens", "auth_sessions", "profiles", "river_job", "river_leader", "river_notification", "river_queue", "users",
			"workspace_members", "workspaces"}},
````

````old server/migrations/schema_test.go
	if again, err := m.Up(ctx); err != nil || len(again) != 5 {
		t.Errorf("Up() again = %d migrations, %v; want 5", len(again), err)
````

````new server/migrations/schema_test.go
	if again, err := m.Up(ctx); err != nil || len(again) != 7 {
		t.Errorf("Up() again = %d migrations, %v; want 7", len(again), err)
````

````old server/migrations/schema_test.go
// Constraint and index names are the stable names of M2 design 3.13: errors
// are mapped by them, and later migrations drop them by name.
````

````new server/migrations/schema_test.go
// Constraint and index names are the stable names of M2 design 3.13 and M3
// design 4 (an unconditional index <table>_<column>_idx under each CASCADE
// foreign key to workspaces): errors are mapped by them, and later
// migrations drop them by name.
````

````old server/migrations/schema_test.go
		WHERE conrelid IN ('users'::regclass, 'profiles'::regclass, 'auth_sessions'::regclass, 'api_tokens'::regclass)
````

````new server/migrations/schema_test.go
		WHERE conrelid IN ('users'::regclass, 'profiles'::regclass, 'auth_sessions'::regclass, 'api_tokens'::regclass,
				'workspaces'::regclass, 'workspace_members'::regclass)
````

````old server/migrations/schema_test.go
		SELECT indexname || ' i' FROM pg_indexes WHERE tablename IN ('users', 'profiles', 'auth_sessions', 'api_tokens')
````

````new server/migrations/schema_test.go
		SELECT indexname || ' i' FROM pg_indexes
		WHERE tablename IN ('users', 'profiles', 'auth_sessions', 'api_tokens', 'workspaces', 'workspace_members')
````

````old server/migrations/schema_test.go
		"users_pkey p",
````

````new server/migrations/schema_test.go
		"users_pkey p",
		"workspace_members_created_by_id_fkey f n",
		"workspace_members_member_id_fkey f c",
		"workspace_members_member_id_idx i",
		"workspace_members_pkey i",
		"workspace_members_pkey p",
		"workspace_members_role_check c",
		"workspace_members_updated_by_id_fkey f n",
		"workspace_members_workspace_id_fkey f c",
		"workspace_members_workspace_id_idx i",
		"workspace_members_workspace_id_member_id_key i",
		"workspaces_created_by_id_fkey f n",
		"workspaces_name_check c",
		"workspaces_organization_size_check c",
		"workspaces_pkey i",
		"workspaces_pkey p",
		"workspaces_slug_check c",
		"workspaces_slug_key i",
		"workspaces_updated_by_id_fkey f n",
````

````old server/migrations/schema_test.go
// (M2 design 4.2, 4.3, 4.5).
````

````new server/migrations/schema_test.go
// (M2 design 4.2, 4.3, 4.5; M3 design 4.2, 4.3).
````

````old server/migrations/schema_test.go
		"INSERT INTO api_tokens (id, user_id, token_hash, label) VALUES ('0199a2b4-0000-7000-8000-000000000004', " + user + ", sha256('t'), 'x')",
````

````new server/migrations/schema_test.go
		"INSERT INTO api_tokens (id, user_id, token_hash, label) VALUES ('0199a2b4-0000-7000-8000-000000000004', " + user + ", sha256('t'), 'x')",
		"INSERT INTO workspaces (id, name, slug, organization_size) VALUES ('0199a2b4-0000-7000-8000-000000000005', 'Acme', 'acme_1-2', '500+')",
		"INSERT INTO workspace_members (id, workspace_id, member_id, role) VALUES ('0199a2b4-0000-7000-8000-000000000006', " +
			"'0199a2b4-0000-7000-8000-000000000005', " + user + ", 20)",
````

````old server/migrations/schema_test.go
		{"empty label", "UPDATE api_tokens SET label = ''", "api_tokens_label_check"},
````

````new server/migrations/schema_test.go
		{"empty label", "UPDATE api_tokens SET label = ''", "api_tokens_label_check"},
		{"empty workspace name", "UPDATE workspaces SET name = ''", "workspaces_name_check"},
		{"upper-case slug", "UPDATE workspaces SET slug = 'Acme'", "workspaces_slug_check"},
		{"slug with a dot", "UPDATE workspaces SET slug = 'acme.io'", "workspaces_slug_check"},
		{"empty slug", "UPDATE workspaces SET slug = ''", "workspaces_slug_check"},
		{"unknown organization size", "UPDATE workspaces SET organization_size = '1000+'", "workspaces_organization_size_check"},
		{"role 10", "UPDATE workspace_members SET role = 10", "workspace_members_role_check"},
		{"role 0", "UPDATE workspace_members SET role = 0", "workspace_members_role_check"},
````

- [ ] **Step 3: 测试、lint**

Run: `go -C server test -count=1 ./migrations/`
Expected: `ok`。

Run: `go -C server test -count=1 -run TestTheGrantsFileCoversEveryRelationAndFunction ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add deploy/runtime-grants.sql server/migrations/schema_test.go server/migrations/sql/00006_workspace_workspaces.sql server/migrations/sql/00007_workspace_workspace_members.sql
```
```bash
git commit -m "feat(M3/P1): the workspaces and workspace_members tables

Migrations 00006 and 00007 keep 10 of Plane's columns each, with CHECKs on
the name, the slug, the organization size and the role, a partial unique
slug index, and the runtime role's grants.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 7 个迁移 up、down、再 up 都通过；18 个名字和 7 个反例由测试核对；运行时角色能读写两张表。

---

### Task 2: `TestSQLCSchemaScope` 认出四种写法

**Files:**
- Modify: `server/internal/archtest/sqlc_cases_test.go`、`server/internal/archtest/sqlc_test.go`

**Interfaces:**
- Produces（spec 2.4，M3 设计 4.1；M2-closeout 第 8 节第 2 件）：`tableName` 认带引号的名字和 `public.` 前缀（`table()` 把带引号的按原样、不带的转小写）；`renameTable`：`ALTER TABLE … RENAME TO t` 让 `t` 属于改名的模块；`tableStatements` 四种写法：`alters`、`indexes`、`puts a trigger on`、`drops`，各报 `migration %s %s %s, which no migration creates` 或 `…, which module %s creates: the migration belongs to %s`。`REFERENCES` 不算：外键可以指向别的模块的表。
- 使用者：以后每个 Phase 的迁移。

**Tests:**（`server/internal/archtest/sqlc_cases_test.go`）
- `TestSQLCScopeOfTheBaseLayoutPasses`（不改名，基准布局扩大）：River 的迁移加上自己表上的触发器、改名再删除、不带名字的唯一索引，`asset` 的迁移加上引用 `users` 的外键、自己表上的索引和 Down 中删自己的表：都不报。
- `TestSQLCScopeReportsViolations` 加 9 个反例，各报恰好一条：`ALTER TABLE of a quoted name`、`… in the schema public`（`ALTER TABLE IF EXISTS ONLY "public"."users"`）、`CREATE INDEX on another module's table`、`CREATE UNIQUE INDEX without a name`（`CONCURRENTLY … ON ONLY public.users`）、`CREATE TRIGGER on another module's table`（`CREATE OR REPLACE TRIGGER`，跨行）、`DROP TABLE of another module's table`（`DROP TABLE IF EXISTS assets, users CASCADE`）、`DROP TABLE of an unknown table`、`a quoted table created twice`、`a rename of another module's table`。

- [ ] **Step 1: 四种写法**

`server/internal/archtest/sqlc_test.go`（修改，7 处）：

````old server/internal/archtest/sqlc_test.go
}

var (
````

````new server/internal/archtest/sqlc_test.go
}

// A table name as a migration writes it: unquoted (folded to lower case) or
// quoted, in the schema public or without one. ident is any other name, e.g.
// an index's or a trigger's.
const (
	ident     = `(?:"[^"]+"|[a-z_][a-z0-9_$]*)`
	tableName = `(?:(?:public|"public")\.)?("[^"]+"|[a-z_][a-z0-9_]*)`
)

var (
````

````old server/internal/archtest/sqlc_test.go
	createTable       = regexp.MustCompile(`(?i)\bCREATE\s+(?:UNLOGGED\s+)?TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(?:public\.)?([a-z_][a-z0-9_]*)`)
	alterTable        = regexp.MustCompile(`(?i)\bALTER\s+TABLE\s+(?:IF\s+EXISTS\s+)?(?:ONLY\s+)?(?:public\.)?([a-z_][a-z0-9_]*)`)
````

````new server/internal/archtest/sqlc_test.go
	createTable       = regexp.MustCompile(`(?i)\bCREATE\s+(?:UNLOGGED\s+)?TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?` + tableName)
	renameTable       = regexp.MustCompile(`(?i)\bALTER\s+TABLE\s+(?:IF\s+EXISTS\s+)?(?:ONLY\s+)?` + tableName + `\s+RENAME\s+TO\s+` + tableName)
````

````old server/internal/archtest/sqlc_test.go
)

// sqlcScopeViolations checks, in M2 design 3.14's words:
````

````new server/internal/archtest/sqlc_test.go
)

// The statements that act on a table, which must be one the migration's
// module creates (M2 design 3.14, M3 design 4.1). REFERENCES is not among
// them: a foreign key to another module's table is allowed, since no query
// can read across modules anyway. A DROP TABLE may name several tables.
var tableStatements = []struct {
	verb string
	re   *regexp.Regexp
}{
	{"alters", regexp.MustCompile(`(?i)\bALTER\s+TABLE\s+(?:IF\s+EXISTS\s+)?(?:ONLY\s+)?` + tableName)},
	{"indexes", regexp.MustCompile(`(?i)\bCREATE\s+(?:UNIQUE\s+)?INDEX\s+(?:CONCURRENTLY\s+)?(?:IF\s+NOT\s+EXISTS\s+)?(?:` + ident +
		`\s+)?ON\s+(?:ONLY\s+)?` + tableName)},
	{"puts a trigger on", regexp.MustCompile(`(?is)\bCREATE\s+(?:OR\s+REPLACE\s+)?(?:CONSTRAINT\s+)?TRIGGER\s+` + ident +
		`\s.*?\bON\s+(?:ONLY\s+)?` + tableName)},
	{"drops", regexp.MustCompile(`(?i)\bDROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?` + tableName + `((?:\s*,\s*` + tableName + `)*)`)},
}

// moreTables reads the ", t2, t3" after the first table of a DROP TABLE.
var moreTables = regexp.MustCompile(`(?i)` + tableName)

// table is a table name as PostgreSQL resolves it: a quoted name as written,
// an unquoted one in lower case.
func table(name string) string {
	if unquoted, ok := strings.CutPrefix(name, `"`); ok {
		return strings.TrimSuffix(unquoted, `"`)
	}
	return strings.ToLower(name)
}

// touched lists the tables each kind of statement in sql acts on, once each
// per kind: Up and Down both act on them.
func touched(sql string) map[string][]string {
	out := map[string][]string{}
	for _, st := range tableStatements {
		for _, m := range st.re.FindAllStringSubmatch(sql, -1) {
			names := []string{m[1]}
			if len(m) > 2 {
				for _, more := range moreTables.FindAllStringSubmatch(m[2], -1) {
					names = append(names, more[1])
				}
			}
			for _, n := range names {
				if t := table(n); !slices.Contains(out[st.verb], t) {
					out[st.verb] = append(out[st.verb], t)
				}
			}
		}
	}
	return out
}

// sqlcScopeViolations checks, in M2 design 3.14's words:
````

````old server/internal/archtest/sqlc_test.go
//   - the target of every ALTER TABLE is created by the file name's module,
//     so a migration belongs to the module that owns the table it changes;
````

````new server/internal/archtest/sqlc_test.go
//   - the target of every ALTER TABLE, CREATE [UNIQUE] INDEX, CREATE TRIGGER
//     and DROP TABLE, quoted or not, is created by the file name's module
//     (M3 design 4.1), so a migration belongs to the module that owns the
//     table it changes; a table renamed into existence belongs to the module
//     that renames it;
````

````old server/internal/archtest/sqlc_test.go
		for _, c := range createTable.FindAllStringSubmatch(sqlComment.ReplaceAllString(m.sql, ""), -1) {
			table := strings.ToLower(c[1])
			if prev, ok := owner[table]; ok && prev != match[1] {
				report("tables: %s is created by both %s and %s", table, prev, match[1])
````

````new server/internal/archtest/sqlc_test.go
		sql := sqlComment.ReplaceAllString(m.sql, "")
		var created []string
		for _, c := range createTable.FindAllStringSubmatch(sql, -1) {
			created = append(created, table(c[1]))
		}
		for _, r := range renameTable.FindAllStringSubmatch(sql, -1) {
			created = append(created, table(r[2]))
		}
		for _, t := range created {
			if prev, ok := owner[t]; ok && prev != match[1] {
				report("tables: %s is created by both %s and %s", t, prev, match[1])
````

````old server/internal/archtest/sqlc_test.go
			owner[table] = match[1]
````

````new server/internal/archtest/sqlc_test.go
			owner[t] = match[1]
````

````old server/internal/archtest/sqlc_test.go
		var altered []string // once per table: Up and Down both alter it
		for _, a := range alterTable.FindAllStringSubmatch(sqlComment.ReplaceAllString(m.sql, ""), -1) {
			if table := strings.ToLower(a[1]); !slices.Contains(altered, table) {
				altered = append(altered, table)
			}
		}
		for _, table := range altered {
			switch tableOwner, known := owner[table]; {
			case !known:
				report("migration %s alters %s, which no migration creates", m.name, table)
			case tableOwner != module:
				report("migration %s alters %s, which module %s creates: the migration belongs to %s", m.name, table, tableOwner, tableOwner)
````

````new server/internal/archtest/sqlc_test.go
		byVerb := touched(sqlComment.ReplaceAllString(m.sql, ""))
		for _, st := range tableStatements {
			for _, t := range byVerb[st.verb] {
				switch tableOwner, known := owner[t]; {
				case !known:
					report("migration %s %s %s, which no migration creates", m.name, st.verb, t)
				case tableOwner != module:
					report("migration %s %s %s, which module %s creates: the migration belongs to %s", m.name, st.verb, t, tableOwner, tableOwner)
				}
````

- [ ] **Step 2: 反例**

`server/internal/archtest/sqlc_cases_test.go`（修改，4 处）：

````old server/internal/archtest/sqlc_cases_test.go
	"slices"
````

````new server/internal/archtest/sqlc_cases_test.go
	"slices"
	"strings"
````

````old server/internal/archtest/sqlc_cases_test.go
// module, asset, creates assets; identity's own later migration alters
// users to reference assets; River's migration belongs to no entry, and
// alters an unlogged table it creates, as its real one does.
````

````new server/internal/archtest/sqlc_cases_test.go
// module, asset, creates assets, which references users and has an index;
// identity's own later migration alters users to reference assets; River's
// migration belongs to no entry, and alters an unlogged table it creates,
// puts a trigger on its table, and renames a table and drops it, as its real
// one does.
````

````old server/internal/archtest/sqlc_cases_test.go
			"ALTER TABLE river_job ADD COLUMN x int;\nALTER TABLE river_leader ADD COLUMN y int;\n"},
		{"00020_asset_assets.sql", "-- +goose Up\nCREATE TABLE IF NOT EXISTS assets (id uuid PRIMARY KEY);\n"},
````

````new server/internal/archtest/sqlc_cases_test.go
			"ALTER TABLE river_job ADD COLUMN x int;\nALTER TABLE river_leader ADD COLUMN y int;\n" +
			"CREATE TRIGGER river_notify\n    AFTER INSERT ON river_job\n    FOR EACH ROW EXECUTE PROCEDURE river_job_notify();\n" +
			"CREATE TABLE river_migration (version bigint);\nALTER TABLE river_migration\n    RENAME TO river_migration_old;\n" +
			"CREATE UNIQUE INDEX ON river_job USING btree(id);\nDROP TABLE river_migration_old;\n"},
		{"00020_asset_assets.sql", "-- +goose Up\nCREATE TABLE IF NOT EXISTS assets (id uuid PRIMARY KEY, created_by_id uuid REFERENCES users);\n" +
			"CREATE INDEX assets_created_by_id_idx ON assets (created_by_id);\n-- +goose Down\nDROP TABLE assets;\n"},
````

````old server/internal/archtest/sqlc_cases_test.go
		}, "tables: users is created by both identity and asset"},
````

````new server/internal/archtest/sqlc_cases_test.go
		}, "tables: users is created by both identity and asset"},
		// The four forms of M3 design 4.1, each on another module's table.
		{"ALTER TABLE of a quoted name", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			m[2].sql += `ALTER TABLE "users" ADD COLUMN asset_id uuid;`
			return e, m, mods
		}, "migration 00020_asset_assets.sql alters users, which module identity creates: the migration belongs to identity"},
		{"ALTER TABLE of a quoted name in the schema public", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			m[2].sql += `ALTER TABLE IF EXISTS ONLY "public"."users" DROP COLUMN x;`
			return e, m, mods
		}, "migration 00020_asset_assets.sql alters users, which module identity creates: the migration belongs to identity"},
		{"CREATE INDEX on another module's table", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			m[2].sql += "CREATE INDEX assets_users_email_idx ON users (email);"
			return e, m, mods
		}, "migration 00020_asset_assets.sql indexes users, which module identity creates: the migration belongs to identity"},
		{"CREATE UNIQUE INDEX without a name", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			m[2].sql += "CREATE UNIQUE INDEX CONCURRENTLY ON ONLY public.users (lower(email));"
			return e, m, mods
		}, "migration 00020_asset_assets.sql indexes users, which module identity creates: the migration belongs to identity"},
		{"CREATE TRIGGER on another module's table", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			m[2].sql += "CREATE OR REPLACE TRIGGER assets_touch\n    AFTER UPDATE OF email ON users\n    FOR EACH ROW EXECUTE FUNCTION touch();"
			return e, m, mods
		}, "migration 00020_asset_assets.sql puts a trigger on users, which module identity creates: the migration belongs to identity"},
		{"DROP TABLE of another module's table", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			m[2].sql = strings.Replace(m[2].sql, "DROP TABLE assets;", "DROP TABLE IF EXISTS assets, users CASCADE;", 1)
			return e, m, mods
		}, "migration 00020_asset_assets.sql drops users, which module identity creates: the migration belongs to identity"},
		{"DROP TABLE of an unknown table", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			m[2].sql += "DROP TABLE people;"
			return e, m, mods
		}, "migration 00020_asset_assets.sql drops people, which no migration creates"},
		{"a quoted table created twice", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			m[2].sql += `CREATE TABLE "users" (id uuid);`
			return e, m, mods
		}, "tables: users is created by both identity and asset"},
		{"a rename of another module's table", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			m[2].sql += "ALTER TABLE users RENAME TO people;"
			return e, m, mods
		}, "migration 00020_asset_assets.sql alters users, which module identity creates: the migration belongs to identity"},
````

- [ ] **Step 3: 测试、lint**

Run: `go -C server test -count=1 -run TestSQLC ./internal/archtest/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/archtest/sqlc_cases_test.go server/internal/archtest/sqlc_test.go
```
```bash
git commit -m "test(M3/P1): TestSQLCSchemaScope reads quoted names and four more statements

ALTER TABLE, CREATE INDEX, CREATE TRIGGER and DROP TABLE on another module's
table, quoted or in the schema public, and a rename, each with its
counterexample (M3 design 4.1).

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 基准布局通过；9 个新反例各报恰好一条；真实的迁移仍通过 `TestSQLCSchemaScope`。

---

### Task 3: `shared` 的权限端口、平台码 `forbidden`、前缀规则

**Files:**
- Create: `server/internal/shared/authorize.go`、`server/internal/shared/authorize_test.go`
- Modify: `api/common.yaml`、`server/internal/platform/httpserver/apitest/problems.go`、`server/internal/platform/httpserver/apitest/rules_cases_test.go`、`server/internal/platform/httpserver/apitest/rules_test.go`、`server/internal/shared/error.go`、`server/internal/shared/error_test.go`
- Generate: `api/dist/openapi.yaml`、`server/internal/platform/httpserver/apigen/components.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.5、2.6，M3 设计 3.4、11.7）：
  - `shared.Role`（`RoleGuest = 5`、`RoleMember = 15`、`RoleAdmin = 20`）、`shared.Action`、`shared.Target{WorkspaceID, ProjectID uuid.UUID}`、`shared.Grant{WorkspaceRole, ProjectRole Role; ProjectAdmin bool}`；
  - `shared.Authorizer` 接口：`Authorize(ctx, actor Actor, action Action, t Target) (Grant, error)`；
  - `shared.ErrNotVisible`（`KindNotFound`，没有码）；`shared.CodeForbidden = "forbidden"`、`shared.Forbidden()`（403，"Your role does not allow this."）；
  - `apitest`：`forbidden` 是平台码；带前缀的码，前缀必须是 `api/modules/` 下的一个文件名，否则 `problem code %q is prefixed with %q, which is not a module: want one of %q`；
  - `api/common.yaml` 的 `Problem.code` 说明写上 `forbidden` 和新的前缀规则。
- 使用者：Task 5、6 的判定和 `Authorizer`；Task 10 的 `getWorkspace`；以后的模块在别的模块的操作上用自己的码。

**Tests:**
- `server/internal/shared/authorize_test.go`：`TestRolesArePlanes`（三个值是 Plane 的 `ROLE_CHOICES`）；`TestErrNotVisible`（包一层仍被 `errors.Is` 认出；带码的模块 404 不被当作它）。
- `server/internal/shared/error_test.go`：`TestConstructors` 加 `Forbidden()` 一行（403、`forbidden`、说明）。
- `server/internal/platform/httpserver/apitest/rules_cases_test.go`：`TestModuleCodesPass`（`things` 的操作上 `things.taken`、别的模块的 `stuff.not_found`、`forbidden`、`validation_failed` 都通过）；`TestAuthoringRulesReportViolations` 加 "code of no module"（`nowhere.taken`）。
- `rules_test.go`：`TestContractFollowsAuthoringRules` 把模块集合交给 `authoringViolations`。

- [ ] **Step 1: 权限的值和端口；`forbidden`**

`server/internal/shared/authorize.go`（新文件，60 行）：

````file server/internal/shared/authorize.go
package shared

import (
	"context"
	"uuid"
)

// Role is a member's role in a workspace or a project (M3 design 3.4): the
// values of Plane's ROLE_CHOICES, which the tables store.
type Role int

// The three roles. Rules compare roles by set membership, never by order, so
// any other value is allowed nothing.
const (
	RoleGuest  Role = 5
	RoleMember Role = 15
	RoleAdmin  Role = 20
)

// Action names an operation the Authorizer decides on, e.g.
// "workspace.read". Each module declares its own as constants, with an
// Actions() that lists them; the access module's rule table is keyed by
// these names (M3 design 3.4).
type Action string

// Target is what an action is on: a workspace, and for a project-level
// action a project of it; ProjectID is the zero UUID for a workspace-level
// action.
type Target struct {
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
}

// Grant is what a decision read: the caller's active roles in the target's
// workspace and project, and whether the caller administers the project (a
// project member who is its admin or the workspace's). Use cases make the
// relative checks with it (M3 design 3.5, 3.7) instead of reading the roles
// again.
type Grant struct {
	WorkspaceRole Role
	ProjectRole   Role
	ProjectAdmin  bool
}

// Authorizer decides whether actor may do action on t (M3 design 3.4). It
// reads the facts on every call, in the transaction ctx carries: a write
// calls it after it has locked the parent row (M3 design 3.6 convention 2).
// A caller who cannot see the target gets ErrNotVisible, which the use case
// turns into its own resource's 404; one who can see it but whose role the
// rule does not allow gets Forbidden. An action without a rule is an
// internal error: nothing is allowed.
type Authorizer interface {
	Authorize(ctx context.Context, actor Actor, action Action, t Target) (Grant, error)
}

// ErrNotVisible is the Authorizer's answer when the caller cannot see the
// target. It has no code: the use case recognizes it with errors.Is and
// answers the 404 code of what the caller named, e.g. workspace.not_found,
// which only the use case knows.
var ErrNotVisible = &Error{Kind: KindNotFound, Detail: "The target is not visible to the caller."}
````

`server/internal/shared/error.go`（修改，3 处）：

````old server/internal/shared/error.go
// authentication puts the Actor in the context, and TxManager carries one
// transaction through the repositories of several modules. It imports only
// the standard library, and the platform does not import it: the platform
// declares the small interfaces these types satisfy by structure.
````

````new server/internal/shared/error.go
// authentication puts the Actor in the context, TxManager carries one
// transaction through the repositories of several modules, and the
// Authorizer decides what a caller may do in a workspace or a project (M3
// design 3.4). It imports only the standard library, and the platform does
// not import it: the platform declares the small interfaces these types
// satisfy by structure.
````

````old server/internal/shared/error.go
	CodeUnauthorized     = "unauthorized"
````

````new server/internal/shared/error.go
	CodeUnauthorized     = "unauthorized"
	CodeForbidden        = "forbidden"
````

````old server/internal/shared/error.go
}

// RateLimited reports a caller over one of a module's rate limits: 429
````

````new server/internal/shared/error.go
}

// Forbidden reports a caller who can see the target but whose role the rule
// table does not allow the action: 403 forbidden (M3 design 3.4). The
// access module's refusal crosses every module, so its code is a platform
// code (M3 design 11.7).
func Forbidden() *Error {
	return &Error{Kind: KindForbidden, Code: CodeForbidden, Detail: "Your role does not allow this."}
}

// RateLimited reports a caller over one of a module's rate limits: 429
````

`server/internal/platform/httpserver/apitest/problems.go`（修改，3 处）：

````old server/internal/platform/httpserver/apitest/problems.go
// platformCodes are the codes without a module prefix (M2 design 3.11).
````

````new server/internal/platform/httpserver/apitest/problems.go
// platformCodes are the codes without a module prefix (M2 design 3.11), and
// forbidden, the access module's refusal (M3 design 11.7).
````

````old server/internal/platform/httpserver/apitest/problems.go
	"bad_request", "unauthorized", "not_found", "payload_too_large", "validation_failed",
````

````new server/internal/platform/httpserver/apitest/problems.go
	"bad_request", "unauthorized", "forbidden", "not_found", "payload_too_large", "validation_failed",
````

````old server/internal/platform/httpserver/apitest/problems.go
// code prefixed with the module, e.g. identity.email_taken.
````

````new server/internal/platform/httpserver/apitest/problems.go
// code prefixed with a module, e.g. identity.email_taken.
````

`api/common.yaml`（修改，1 处）：

````old api/common.yaml
            unauthorized, not_found, payload_too_large, validation_failed,
            rate_limited, server_busy, internal_error, not_ready); module
            codes are prefixed with the module, e.g. identity.email_taken.
            Each operation lists the codes it can answer in x-problem-codes.
````

````new api/common.yaml
            unauthorized, forbidden, not_found, payload_too_large,
            validation_failed, rate_limited, server_busy, internal_error,
            not_ready); module codes are prefixed with the module that
            refuses, e.g. identity.email_taken, whichever module's operation
            answers them. Each operation lists the codes it can answer in
            x-problem-codes.
````

- [ ] **Step 2: 测试**

`server/internal/shared/authorize_test.go`（新文件，35 行）：

````file server/internal/shared/authorize_test.go
package shared_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The roles are the values of Plane's ROLE_CHOICES, which the tables store
// (M3 design 3.4).
func TestRolesArePlanes(t *testing.T) {
	if shared.RoleGuest != 5 || shared.RoleMember != 15 || shared.RoleAdmin != 20 {
		t.Errorf("roles = %d, %d, %d; want 5, 15, 20", shared.RoleGuest, shared.RoleMember, shared.RoleAdmin)
	}
}

// A use case recognizes ErrNotVisible through wrapping, and no module's 404
// is mistaken for it: those have a code.
func TestErrNotVisible(t *testing.T) {
	wrapped := fmt.Errorf("authorize: %w", shared.ErrNotVisible)
	if !errors.Is(wrapped, shared.ErrNotVisible) {
		t.Errorf("errors.Is(%v, ErrNotVisible) = false, want true", wrapped)
	}
	notFound := shared.NewError(shared.KindNotFound, "workspace.not_found", "The workspace does not exist.")
	for _, err := range []error{notFound, shared.Forbidden(), errors.New("not visible")} {
		if errors.Is(err, shared.ErrNotVisible) {
			t.Errorf("errors.Is(%v, ErrNotVisible) = true, want false", err)
		}
	}
	if got := shared.ErrNotVisible.ProblemStatus(); got != 404 || shared.ErrNotVisible.ProblemCode() != "" {
		t.Errorf("ErrNotVisible = %d %q, want 404 without a code", got, shared.ErrNotVisible.ProblemCode())
	}
}
````

`server/internal/shared/error_test.go`（修改，1 处）：

````old server/internal/shared/error_test.go
		{"Unauthenticated", shared.Unauthenticated(), 401, "unauthorized", 0},
````

````new server/internal/shared/error_test.go
		{"Unauthenticated", shared.Unauthenticated(), 401, "unauthorized", 0},
		{"Forbidden", shared.Forbidden(), 403, "forbidden", 0},
````

`server/internal/platform/httpserver/apitest/rules_test.go`（修改，8 处）：

````old server/internal/platform/httpserver/apitest/rules_test.go
	for _, v := range authoringViolations(c.doc, pathOwners(t)) {
````

````new server/internal/platform/httpserver/apitest/rules_test.go
	modules, err := moduleNames()
	if err != nil || len(modules) == 0 {
		t.Fatalf("module files = %q, %v; want at least one", modules, err)
	}
	for _, v := range authoringViolations(c.doc, modules) {
````

````old server/internal/platform/httpserver/apitest/rules_test.go
// 2.4 and M2 design 3.11–3.12; owners maps each path to the module file that
// declares it. It takes the document so that each check is proven on
// hand-built bad documents (rules_cases_test.go) as well as applied to the
// real contract.
func authoringViolations(doc *openapi3.T, owners map[string]string) []string {
	r := &ruleCheck{doc: doc, owners: owners, lowerCamel: regexp.MustCompile(`^[a-z][A-Za-z0-9]*$`)}
````

````new server/internal/platform/httpserver/apitest/rules_test.go
// 2.4 and M2 design 3.11–3.12, as M3 design 11.7 revises them; modules are
// nerve's modules, those with a file in api/modules/. It takes the document
// so that each check is proven on hand-built bad documents
// (rules_cases_test.go) as well as applied to the real contract.
func authoringViolations(doc *openapi3.T, modules []string) []string {
	r := &ruleCheck{doc: doc, modules: modules, lowerCamel: regexp.MustCompile(`^[a-z][A-Za-z0-9]*$`)}
````

````old server/internal/platform/httpserver/apitest/rules_test.go
	owners     map[string]string
````

````new server/internal/platform/httpserver/apitest/rules_test.go
	modules    []string
````

````old server/internal/platform/httpserver/apitest/rules_test.go
		r.operation(method+" "+path, r.owners[path], ops[method])
````

````new server/internal/platform/httpserver/apitest/rules_test.go
		r.operation(method+" "+path, ops[method])
````

````old server/internal/platform/httpserver/apitest/rules_test.go
// top-level ones (M2 design 3.11). A module's code is prefixed with the
// module file that declares the path; a code without prefix is a platform
// code.
func (r *ruleCheck) problemCodes(where, owner string, op *openapi3.Operation) {
````

````new server/internal/platform/httpserver/apitest/rules_test.go
// top-level ones (M2 design 3.11). A module's code is prefixed with a module
// of nerve, the one that refuses, which need not be the file that declares
// the path (M3 design 11.7): a project operation answers
// workspace.not_found. A code without prefix is a platform code.
func (r *ruleCheck) problemCodes(where string, op *openapi3.Operation) {
````

````old server/internal/platform/httpserver/apitest/rules_test.go
		case prefixed && prefix != owner:
			r.report(where, "problem code %q is not prefixed with its module %q", code, owner)
````

````new server/internal/platform/httpserver/apitest/rules_test.go
		case prefixed && !slices.Contains(r.modules, prefix):
			r.report(where, "problem code %q is prefixed with %q, which is not a module: want one of %q", code, prefix, r.modules)
````

````old server/internal/platform/httpserver/apitest/rules_test.go
func (r *ruleCheck) operation(where, owner string, op *openapi3.Operation) {
````

````new server/internal/platform/httpserver/apitest/rules_test.go
func (r *ruleCheck) operation(where string, op *openapi3.Operation) {
````

````old server/internal/platform/httpserver/apitest/rules_test.go
	r.problemCodes(where, owner, op)
````

````new server/internal/platform/httpserver/apitest/rules_test.go
	r.problemCodes(where, op)
````

`server/internal/platform/httpserver/apitest/rules_cases_test.go`（修改，6 处）：

````old server/internal/platform/httpserver/apitest/rules_cases_test.go
// ruleCasesOwners says which module file declares each path of the base.
var ruleCasesOwners = map[string]string{"/api/v0/things": "things"}
````

````new server/internal/platform/httpserver/apitest/rules_cases_test.go
// ruleCasesModules are the modules of the cases: things declares the base's
// path, and stuff is another module.
var ruleCasesModules = []string{"stuff", "things"}
````

````old server/internal/platform/httpserver/apitest/rules_cases_test.go
// A module's own prefixed code passes.
````

````new server/internal/platform/httpserver/apitest/rules_cases_test.go
// A code prefixed with a module passes: the module's own, or another
// module's, which refused (M3 design 11.7); and forbidden, the platform code
// of the access module.
````

````old server/internal/platform/httpserver/apitest/rules_cases_test.go
	doc := parse(t, strings.Replace(ruleCasesBase, "x-problem-codes: [not_found]", "x-problem-codes: [things.taken, validation_failed]", 1))
	if got := authoringViolations(doc, ruleCasesOwners); len(got) != 0 {
````

````new server/internal/platform/httpserver/apitest/rules_cases_test.go
	doc := parse(t, strings.Replace(ruleCasesBase, "x-problem-codes: [not_found]",
		"x-problem-codes: [things.taken, stuff.not_found, forbidden, validation_failed]", 1))
	if got := authoringViolations(doc, ruleCasesModules); len(got) != 0 {
````

````old server/internal/platform/httpserver/apitest/rules_cases_test.go
	if got := authoringViolations(parse(t, ruleCasesBase), ruleCasesOwners); len(got) != 0 {
````

````new server/internal/platform/httpserver/apitest/rules_cases_test.go
	if got := authoringViolations(parse(t, ruleCasesBase), ruleCasesModules); len(got) != 0 {
````

````old server/internal/platform/httpserver/apitest/rules_cases_test.go
		{"code of another module", "x-problem-codes: [not_found]", "x-problem-codes: [stuff.taken]",
			`GET /api/v0/things: problem code "stuff.taken" is not prefixed with its module "things"`},
````

````new server/internal/platform/httpserver/apitest/rules_cases_test.go
		{"code of no module", "x-problem-codes: [not_found]", "x-problem-codes: [nowhere.taken]",
			`GET /api/v0/things: problem code "nowhere.taken" is prefixed with "nowhere", which is not a module: want one of ["stuff" "things"]`},
````

````old server/internal/platform/httpserver/apitest/rules_cases_test.go
			if got := authoringViolations(doc, ruleCasesOwners); !slices.Equal(got, []string{tt.want}) {
````

````new server/internal/platform/httpserver/apitest/rules_cases_test.go
			if got := authoringViolations(doc, ruleCasesModules); !slices.Equal(got, []string{tt.want}) {
````

- [ ] **Step 3: 生成**

Run: `make gen`
Expected: 成功。只有下表三个生成物改变（`Problem.code` 的说明进了三个文件）：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `8d45bde09fb4b95ca82061f72db760b6d74913143589266cbcbd47f07c75c1ce` | 879 | `api/dist/openapi.yaml` |
| `216a66ef6ddf81d5f732afdb22ac308bceda6791798d52183d22a7b39496ca4e` | 89 | `server/internal/platform/httpserver/apigen/components.gen.go` |
| `5cdfcc307781f5e3e7480a0710e493c418ebcf0e53a2d45b1ed09850b5c85ab4` | 911 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/platform/httpserver/apigen/components.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同；`wc -l` 与上表的行数相同。

- [ ] **Step 4: 测试、lint、前端检查**

Run: `go -C server test -count=1 ./internal/shared/ ./internal/platform/httpserver/apitest/`
Expected: 两个 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过（`forbidden` 还没有操作声明，前端的文案表不变）。

- [ ] **Step 5: 提交**

```bash
git add api/common.yaml server/internal/platform/httpserver/apitest/problems.go server/internal/platform/httpserver/apitest/rules_cases_test.go server/internal/platform/httpserver/apitest/rules_test.go server/internal/shared/authorize.go server/internal/shared/authorize_test.go server/internal/shared/error.go server/internal/shared/error_test.go api/dist/openapi.yaml server/internal/platform/httpserver/apigen/components.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P1): shared's Authorizer port, the platform code forbidden and the module prefix rule

The permission values and port live in shared (M3 design 3.4); forbidden is
the platform code of a refusal by role; a problem code's prefix names the
module that refused, which need not declare the operation (11.7).

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**Done when:** `shared` 仍只依赖标准库；`apitest` 的新规则有正反两面的测试；三个生成物的 SHA-256 与表相同。

---

### Task 4: 三条取值规则移到 `shared`

**Files:**
- Create: `server/internal/shared/email.go`、`server/internal/shared/email_test.go`、`server/internal/shared/timezone.go`、`server/internal/shared/timezone_test.go`、`server/internal/shared/url.go`、`server/internal/shared/url_test.go`
- Modify: `server/internal/modules/identity/adapter/http/limits.go`、`server/internal/modules/identity/app/activate.go`、`server/internal/modules/identity/app/deactivate.go`、`server/internal/modules/identity/app/login.go`、`server/internal/modules/identity/app/reset_password.go`、`server/internal/modules/identity/app/set_email.go`、`server/internal/modules/identity/domain/user.go`、`server/internal/shared/error.go`
- Modify（完整内容）: `server/internal/modules/identity/domain/email.go`、`server/internal/modules/identity/domain/email_test.go`
- Delete: `server/internal/modules/identity/domain/url.go`、`server/internal/modules/identity/domain/url_test.go`

**Interfaces:**
- Produces（spec 2.5，M3 设计 3.13）：`shared.NormalizeEmail`、`shared.ValidEmail`、`shared.MaxEmailLength`（原样从 `identity/domain/email.go` 移来）、`shared.ContainsURL`（原样从 `identity/domain/url.go` 移来）、`shared.ValidTimezone`（新：`time.LoadLocation` 接受、且不是 `Local`；二进制内嵌时区数据库，M3 设计 11.4）。`shared/error.go` 的包说明加上"两个以上模块必须一致的纯取值规则"。
- `identity/domain/email.go` 只剩 `DisplayNameFromEmail`；`identity/domain/url.go`、`url_test.go` 删除；`user.go` 和六个调用方改调 `shared`，行为不变。
- 使用者：Task 7 的工作区校验（名称不含网址、时区），Task 10 的 `ExecuteForAdmin`（规范化邮箱），Task 14 的命令输出。

**Tests:**（随规则移动，内容不变）
- `server/internal/shared/email_test.go`：`TestNormalizeEmail`、`TestValidEmail`、`TestValidEmailLengthLimit`。
- `server/internal/shared/url_test.go`：`TestContainsURLAsPlane`（期望值是 Plane 的 `contains_url` 对同样字符串的回答）。
- `server/internal/shared/timezone_test.go`：`TestValidTimezone`（`UTC`、`Asia/Shanghai`、`America/New_York` 接受；`Mars/Olympus`、空（`LoadLocation` 把它当作 UTC）、`Local`（主机的时区）不接受）。
- `server/internal/modules/identity/domain/email_test.go` 只剩 `DisplayNameFromEmail` 的测试；`identity` 的其他测试不改而通过。

- [ ] **Step 1: `shared` 的三条规则**

`server/internal/shared/email.go`（新文件，85 行）：

````file server/internal/shared/email.go
package shared

import (
	"net/netip"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxEmailLength is the length of users.email, varchar(255), in characters.
// The rules of e-mail addresses are shared: an invitation's address must
// compare with an account's by the same rules (M3 design 3.13).
const MaxEmailLength = 255

// NormalizeEmail is what Plane does before it validates or looks up an
// e-mail address (email.strip().lower(), plane/apps/api/plane/
// authentication/views/app/email.py:73).
func NormalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// ValidEmail reports whether a normalized address is acceptable (M2 design
// 4.2): at most 255 characters, no white space or control character
// anywhere, and valid by Django's EmailValidator, which Plane uses
// (plane/apps/api/plane/authentication/adapter/base.py:79).
func ValidEmail(email string) bool {
	if email == "" || utf8.RuneCountInString(email) > MaxEmailLength || !utf8.ValidString(email) {
		return false
	}
	for _, r := range email {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return false
		}
	}
	return djangoEmail(email)
}

// The regular expressions of Django 5.2's EmailValidator
// (django/core/validators.py), with every look-around rewritten for RE2 into
// the equivalent explicit form: a domain label of 1–63 characters neither
// starts nor ends with a hyphen.
//
// Django compiles them with re.IGNORECASE on str patterns, under which
// [A-Z] and [a-z] also match four non-ASCII letters (Python's re
// documentation): İ, ı, ſ and K. After lower-casing only ı (U+0131) and
// ſ (U+017F) are left, so the local part's classes add those two.
const (
	djangoUL          = `\x{00a1}-\x{ffff}` // Django's "Unicode letters" range
	djangoLabelChar   = `[a-z` + djangoUL + `0-9]`
	djangoLabel       = djangoLabelChar + `(?:[a-z` + djangoUL + `0-9-]{0,61}` + djangoLabelChar + `)?`
	djangoTLDChar     = `[a-z` + djangoUL + `]`
	djangoTLD         = `\.(?:` + djangoTLDChar + `[a-z` + djangoUL + `-]{0,61}` + djangoTLDChar + `|xn--[a-z0-9]{1,59})`
	djangoAtom        = "[-!#$%&'*+/=?^_" + "`" + `{}|~0-9a-z\x{0131}\x{017f}]+`
	djangoQuotedChars = `[\x01-\x08\x0b\x0c\x0e-\x1f!#-\[\]-\x7f\x{0131}\x{017f}]|\\[\x01-\x09\x0b\x0c\x0e-\x7f\x{0131}\x{017f}]`
)

var (
	djangoUser    = regexp.MustCompile(`(?i)^(?:` + djangoAtom + `(?:\.` + djangoAtom + `)*|"(?:` + djangoQuotedChars + `)*")$`)
	djangoDomain  = regexp.MustCompile(`(?i)^` + djangoLabel + `(?:\.` + djangoLabel + `)*` + djangoTLD + `$`)
	djangoLiteral = regexp.MustCompile(`(?i)^\[([a-f0-9:.]+)\]$`)
)

// djangoEmail is EmailValidator.__call__ with the default allow list, except
// for the length, which ValidEmail bounds more tightly.
func djangoEmail(email string) bool {
	at := strings.LastIndexByte(email, '@')
	if at < 0 {
		return false
	}
	user, domain := email[:at], email[at+1:]
	if !djangoUser.MatchString(user) {
		return false
	}
	if domain == "localhost" || djangoDomain.MatchString(domain) {
		return true
	}
	// A literal address: validate_ipv46_address.
	m := djangoLiteral.FindStringSubmatch(domain)
	if m == nil {
		return false
	}
	_, err := netip.ParseAddr(m[1])
	return err == nil
}
````

`server/internal/shared/url.go`（新文件，45 行）：

````file server/internal/shared/url.go
package shared

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// pySpace is what Python's \s matches in a str pattern (str.isspace()):
// Go's \s is ASCII only and lacks \v.
const pySpace = `\t-\r\x1c-\x20\x85\xa0\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}`

// pyLetters is [a-zA-Z] under Python's re.IGNORECASE, which also matches
// İ, ı, ſ and K (Python's re documentation). Go's (?i) adds ſ and K by case
// folding; İ and ı are listed.
const pyLetters = `a-zA-Z\x{0130}\x{0131}`

// urlPattern is Plane's URL_PATTERN (plane/apps/api/plane/utils/url.py:12-23)
// for RE2: an http(s) address, a www. host, a dotted host name with a TLD
// of 2–6 letters, or an IPv4 address, anywhere in the text.
var urlPattern = regexp.MustCompile(`(?i)(?:` +
	`https?://[^` + pySpace + `]+` +
	`|www\.[` + pyLetters + `0-9](?:[` + pyLetters + `0-9-]{0,61}[` + pyLetters + `0-9])?(?:\.[` + pyLetters + `0-9](?:[` + pyLetters + `0-9-]{0,61}[` + pyLetters + `0-9])?)*` +
	`|(?:[` + pyLetters + `0-9](?:[` + pyLetters + `0-9-]{0,61}[` + pyLetters + `0-9])?\.)+[` + pyLetters + `]{2,6}` +
	`|(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)` +
	`)`)

// ContainsURL is Plane's contains_url (plane/apps/api/plane/utils/url.py:
// 26-53), which it applies to first and last names and to workspace names
// (M3 design 3.10, 3.13): text of more than 1000 characters is not looked
// at, and each line only up to its 500th character.
func ContainsURL(s string) bool {
	if utf8.RuneCountInString(s) > 1000 {
		return false
	}
	for line := range strings.SplitSeq(s, "\n") {
		if runes := []rune(line); len(runes) > 500 {
			line = string(runes[:500])
		}
		if urlPattern.MatchString(line) {
			return true
		}
	}
	return false
}
````

`server/internal/shared/timezone.go`（新文件，21 行）：

````file server/internal/shared/timezone.go
package shared

import "time"

// ValidTimezone reports whether time.LoadLocation loads name (M2 design
// 4.2), except "" and "Local", which it takes for UTC and for the host's
// zone. Accounts, workspaces and projects share it (M3 design 3.13).
//
// It is the one rule here that reads outside the process: LoadLocation reads
// the host's zone files first and Go's own tzdata after them, so under go
// test one host may accept a name that another refuses ("asia/shanghai" on a
// case-insensitive file system, spec P3a 3 item 10). The nerve binary embeds
// the zone database (archtest's TestNerveBinaryEmbedsTheTimeZoneDatabase),
// so a deployment's answer does not depend on the machine (M3 design 11.4).
func ValidTimezone(name string) bool {
	if name == "" || name == "Local" {
		return false
	}
	_, err := time.LoadLocation(name)
	return err == nil
}
````

`server/internal/shared/error.go`（修改，1 处）：

````old server/internal/shared/error.go
// design 3.4). It imports only the standard library, and the platform does
// not import it: the platform declares the small interfaces these types
// satisfy by structure.
````

````new server/internal/shared/error.go
// design 3.4). It also holds the pure value rules that modules must apply
// alike: e-mail addresses, web addresses in names, and time zones (M3 design
// 3.13). It imports only the standard library, and the platform does not
// import it: the platform declares the small interfaces these types satisfy
// by structure.
````

- [ ] **Step 2: `identity` 改用它们，删掉旧文件**

`server/internal/modules/identity/domain/email.go`（完整内容，12 行）：

````whole server/internal/modules/identity/domain/email.go
package domain

import "strings"

// DisplayNameFromEmail is the display name a new account gets: the part of
// the address before its first @, as Plane's User.save() does
// (plane/apps/api/plane/db/models/user.py:169-187). It is never empty for a
// valid address.
func DisplayNameFromEmail(email string) string {
	name, _, _ := strings.Cut(email, "@")
	return name
}
````

`server/internal/modules/identity/domain/url.go`（删除）：

````delete server/internal/modules/identity/domain/url.go
````

`server/internal/modules/identity/domain/user.go`（修改，8 处）：

````old server/internal/modules/identity/domain/user.go
// functions and values, with one exception: validTimezone calls
// time.LoadLocation, which reads the host's zone files, so the names it
// accepts depend on the host (spec P3a 3 item 10).
````

````new server/internal/modules/identity/domain/user.go
// functions and values. The rules of e-mail addresses, web addresses and
// time zones that other modules share are in internal/shared (M3 design
// 3.13).
````

````old server/internal/modules/identity/domain/user.go
		} else if containsURL(*name.value) {
````

````new server/internal/modules/identity/domain/user.go
		} else if shared.ContainsURL(*name.value) {
````

````old server/internal/modules/identity/domain/user.go
	if p.Timezone != nil && !validTimezone(*p.Timezone) {
````

````new server/internal/modules/identity/domain/user.go
	if p.Timezone != nil && !shared.ValidTimezone(*p.Timezone) {
````

````old server/internal/modules/identity/domain/user.go

// validTimezone reports whether time.LoadLocation loads name (M2 design
// 4.2), except "" and "Local", which it takes for UTC and for the host's
// zone. LoadLocation reads the host's zone files first and Go's own tzdata
// after them, so one host may accept a name that another refuses:
// "asia/shanghai" on a case-insensitive file system, "posixrules" where the
// host has that file (spec P3a 3 item 10).
func validTimezone(name string) bool {
	if name == "" || name == "Local" {
		return false
	}
	_, err := time.LoadLocation(name)
	return err == nil
}

````

````new server/internal/modules/identity/domain/user.go

````

````old server/internal/modules/identity/domain/user.go
func NewAccount(rules *PasswordRules, email, password string) (string, error) {
	email = NormalizeEmail(email)
````

````new server/internal/modules/identity/domain/user.go
func NewAccount(rules *PasswordRules, email, password string) (string, error) {
	email = shared.NormalizeEmail(email)
````

````old server/internal/modules/identity/domain/user.go
func NewEmail(field, email string) (string, error) {
	email = NormalizeEmail(email)
````

````new server/internal/modules/identity/domain/user.go
func NewEmail(field, email string) (string, error) {
	email = shared.NormalizeEmail(email)
````

````old server/internal/modules/identity/domain/user.go
	case utf8.RuneCountInString(email) > MaxEmailLength:
````

````new server/internal/modules/identity/domain/user.go
	case utf8.RuneCountInString(email) > shared.MaxEmailLength:
````

````old server/internal/modules/identity/domain/user.go
	case !ValidEmail(email):
````

````new server/internal/modules/identity/domain/user.go
	case !shared.ValidEmail(email):
````

`server/internal/modules/identity/app/activate.go`（修改，2 处）：

````old server/internal/modules/identity/app/activate.go

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
````

````new server/internal/modules/identity/app/activate.go

````

````old server/internal/modules/identity/app/activate.go
	email = domain.NormalizeEmail(email)
````

````new server/internal/modules/identity/app/activate.go
	email = shared.NormalizeEmail(email)
````

`server/internal/modules/identity/app/deactivate.go`（修改，1 处）：

````old server/internal/modules/identity/app/deactivate.go
	email = domain.NormalizeEmail(email)
````

````new server/internal/modules/identity/app/deactivate.go
	email = shared.NormalizeEmail(email)
````

`server/internal/modules/identity/app/login.go`（修改，2 处）：

````old server/internal/modules/identity/app/login.go
	account, err := l.find(ctx, domain.NormalizeEmail(in.Email))
````

````new server/internal/modules/identity/app/login.go
	account, err := l.find(ctx, shared.NormalizeEmail(in.Email))
````

````old server/internal/modules/identity/app/login.go
	if !domain.ValidEmail(email) {
````

````new server/internal/modules/identity/app/login.go
	if !shared.ValidEmail(email) {
````

`server/internal/modules/identity/app/reset_password.go`（修改，1 处）：

````old server/internal/modules/identity/app/reset_password.go
	email = domain.NormalizeEmail(email)
````

````new server/internal/modules/identity/app/reset_password.go
	email = shared.NormalizeEmail(email)
````

`server/internal/modules/identity/app/set_email.go`（修改，1 处）：

````old server/internal/modules/identity/app/set_email.go
	from := domain.NormalizeEmail(email)
````

````new server/internal/modules/identity/app/set_email.go
	from := shared.NormalizeEmail(email)
````

`server/internal/modules/identity/adapter/http/limits.go`（修改，2 处）：

````old server/internal/modules/identity/adapter/http/limits.go

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
````

````new server/internal/modules/identity/adapter/http/limits.go

````

````old server/internal/modules/identity/adapter/http/limits.go
	sum := sha256.Sum256([]byte(domain.NormalizeEmail(email)))
````

````new server/internal/modules/identity/adapter/http/limits.go
	sum := sha256.Sum256([]byte(shared.NormalizeEmail(email)))
````

- [ ] **Step 3: 测试随规则移动**

`server/internal/shared/email_test.go`（新文件，115 行）：

````file server/internal/shared/email_test.go
package shared_test

import (
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestNormalizeEmail(t *testing.T) {
	for in, want := range map[string]string{
		"  Alice@Corp.COM\t\n": "alice@corp.com",
		"ÉLODIE@EXÄMPLE.COM":   "élodie@exämple.com",
		"\xc2\xa0bob@corp.com": "bob@corp.com", // U+00A0: TrimSpace takes it too
	} {
		if got := shared.NormalizeEmail(in); got != want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", in, got, want)
		}
	}
}

// The cases follow Django 5.2's own tests of EmailValidator
// (tests/validators/tests.py) where they apply to normalized addresses.
var (
	validEmails = []string{
		"email@here.com",
		"weirder-email@here.and.there.com",
		"email@[127.0.0.1]",
		"email@[2001:dB8::1]",
		"email@[2001:db8:0:0:0:0:0:1]",
		"email@[::fffF:127.0.0.1]",
		"example@valid-----hyphens.com",
		"example@valid-with-hyphens.com",
		"test@domain.with.idn.tld.उदाहरण.परीक्षा",
		"email@localhost",
		`"test@test"@example.com`,
		"example@atm." + strings.Repeat("a", 63),
		"example@" + strings.Repeat("a", 63) + ".atm",
		"example@" + strings.Repeat("a", 63) + "." + strings.Repeat("b", 10) + ".atm",
		"a.b+c@sub.example.co.uk",
		"x@xn--80ak6aa92e.xn--p1ai",
		"elodie@exämple.com",
		"ıſ@example.com", // Python's IGNORECASE matches ı and ſ with [A-Z]
	}
	invalidEmails = []string{
		"",
		"abc",
		"abc@",
		"@abc.com",
		"a @x.cz",
		"abc@.com",
		"something@@somewhere.com",
		"email@127.0.0.1",
		"email@[127.0.0.256]",
		"email@[2001:db8::12345]",
		"email@[2001:db8:0:0:0:0:1]",
		"email@[::ffff:127.0.0.256]",
		"email@[2001:dg8::1]",
		"email@[2001:dG8:0:0:0:0:0:1]",
		"email@[::fTzF:127.0.0.1]",
		"example@invalid-.com",
		"example@-invalid.com",
		"example@invalid.com-",
		"example@inv-.alid-.com",
		"example@inv-.-alid.com",
		"test@example.com\n\n<script src=\"x.js\">",
		"\"\\\t\"@here.com", // an escaped tab: Django accepts it, control characters are refused first
		"trailingdot@shouldfail.com.",
		"a@b.com\n",
		"a\n@b.com",
		`"test@test"\n@example.com`,
		"a@[127.0.0.1]\n",
		"example@atm." + strings.Repeat("a", 64),
		"example@" + strings.Repeat("b", 64) + ".atm.localhost",
		"example@atm." + strings.Repeat("a", 59) + "xn--", // a TLD ends in a letter
		"a@b",
		"a@b.c",
		"a@b.c0m",
		"a..b@c.com",
		".a@c.com",
		"a.@c.com",
		"élodie@exämple.com", // Django's local part is ASCII
		"é@x.com",
		"a@😀.com",
		`"a b"@example.com`,
		"a@ex\xe3\x80\x80ample.com", // U+3000: Django accepts it in a domain; white space is refused first
		strings.Repeat("a", 64) + "@" + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 60) + ".com",
	}
)

func TestValidEmail(t *testing.T) {
	for _, e := range validEmails {
		if !shared.ValidEmail(e) {
			t.Errorf("ValidEmail(%q) = false, want true", e)
		}
	}
	for _, e := range invalidEmails {
		if shared.ValidEmail(e) {
			t.Errorf("ValidEmail(%q) = true, want false", e)
		}
	}
}

func TestValidEmailLengthLimit(t *testing.T) {
	address := func(last int) string {
		return strings.Repeat("a", 64) + "@" + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", last) + ".com"
	}
	if at255 := address(58); len(at255) != 255 || !shared.ValidEmail(at255) {
		t.Errorf("a %d-character address: valid = %v, want true", len(at255), shared.ValidEmail(at255))
	}
	// Longer than users.email holds, though Django allows 320.
	if at256 := address(59); len(at256) != 256 || shared.ValidEmail(at256) {
		t.Errorf("a %d-character address is valid, want invalid", len(at256))
	}
}
````

`server/internal/shared/url_test.go`（新文件，77 行）：

````file server/internal/shared/url_test.go
package shared_test

import (
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The expectations are what Plane's contains_url answers for the same
// strings (Python 3, plane/apps/api/plane/utils/url.py).
func TestContainsURLAsPlane(t *testing.T) {
	a := strings.Repeat
	tests := []struct {
		name, s string
		want    bool
	}{
		{"https address", "https://x", true},
		{"scheme only", "http://", false},
		{"scheme then a no-break space", "http://\u00a0x", false},
		{"scheme then a vertical tab", "http://\vx", false},
		{"scheme then an information separator", "http://\x1cx", false},
		{"scheme then a next line", "http://\u0085x", false},
		{"scheme then an Ogham space", "http://\u1680x", false},
		{"scheme then an en quad", "http://\u2000x", false},
		{"scheme then a hair space", "http://\u200ax", false},
		{"scheme then a line separator", "http://\u2028x", false},
		{"scheme then a paragraph separator", "http://\u2029x", false},
		{"scheme then a narrow no-break space", "http://\u202fx", false},
		{"scheme then a medium mathematical space", "http://\u205fx", false},
		{"scheme then an ideographic space", "http://\u3000x", false},
		{"scheme then a zero-width space", "http://\u200bx", true},
		{"upper case", "HTTPS://X", true},
		{"ftp", "ftp://x", false},
		{"dotted name", "John.Smith", true},
		{"plain name", "Ann", false},
		{"hyphen", "Mary-Ann", false},
		{"apostrophe", "O'Brien", false},
		{"www host", "www.x", true},
		{"upper-case www", "WWW.Example", true},
		{"one-letter TLD", "a.b", false},
		{"two-letter TLD", "a.bc", true},
		{"label ending in a hyphen", "a-.bc", false},
		{"TLD with a hyphen", "a.b-c", false},
		{"leading hyphen", "-a.bc", true},
		{"IPv4", "1.2.3.4", true},
		{"IPv4 inside a longer number", "999.1.1.1", true},
		{"decimal", "3.14", false},
		{"three numbers", "1.2.3", false},
		{"dotted capital I", "\u0130.ab", true},
		{"dotless i", "\u0131.ab", true},
		{"long s", "\u017f.ab", true},
		{"Kelvin sign", "\u212a.ab", true},
		{"e acute", "\u00e9.ab", false},
		{"letters before an address", "\u00e9l\u00e8ve.com", true},
		{"after an Ogham space", "\u1680http://x", true},
		{"second line", "x\nhttps://y", true},
		{"after a line separator", "x\u2028https://y", true},
		{"no-break space after it", "x.co\u00a0", true},
		{"address ending at character 500", a("a", 490) + " https://x", true},
		{"address cut at character 500", a("a", 491) + " https://x", false},
		{"1000 characters", "x.io\n" + a("a", 995), true},
		{"1001 characters", "x.io\n" + a("a", 996), false},
		{"1000 characters, not bytes", "x.io\n" + a("\u00e9", 995), true},
		{"address ending at character 500, not byte 500", a("\u00e9", 490) + " https://x", true},
		{"address cut at character 500, not byte 500", a("\u00e9", 491) + " https://x", false},
		{"address within character 500, beyond byte 500", a("\u00e9", 300) + " https://x" + a("a", 300), true},
		{"a long first line, an address on the second", a("a", 450) + "\n" + a("b", 60) + " x.io", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shared.ContainsURL(tt.s); got != tt.want {
				t.Errorf("ContainsURL(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}
````

`server/internal/shared/timezone_test.go`（新文件，22 行）：

````file server/internal/shared/timezone_test.go
package shared_test

import (
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestValidTimezone(t *testing.T) {
	for name, want := range map[string]bool{
		"UTC":              true,
		"Asia/Shanghai":    true,
		"America/New_York": true,
		"Mars/Olympus":     false,
		"":                 false, // LoadLocation takes it for UTC
		"Local":            false, // the host's zone
	} {
		if got := shared.ValidTimezone(name); got != want {
			t.Errorf("ValidTimezone(%q) = %v, want %v", name, got, want)
		}
	}
}
````

`server/internal/modules/identity/domain/email_test.go`（完整内容，16 行）：

````whole server/internal/modules/identity/domain/email_test.go
package domain

import "testing"

func TestDisplayNameFromEmail(t *testing.T) {
	for in, want := range map[string]string{
		"alice@corp.com":          "alice",
		`"a@b"@example.com`:       `"a`, // Plane's email.split("@")[0]
		"élodie@exämple.com":      "élodie",
		"first.last+tag@corp.com": "first.last+tag",
	} {
		if got := DisplayNameFromEmail(in); got != want {
			t.Errorf("DisplayNameFromEmail(%q) = %q, want %q", in, got, want)
		}
	}
}
````

`server/internal/modules/identity/domain/url_test.go`（删除）：

````delete server/internal/modules/identity/domain/url_test.go
````

- [ ] **Step 4: 测试、lint**

Run: `go -C server test -count=1 ./internal/shared/ ./internal/modules/identity/...`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/modules/identity/adapter/http/limits.go server/internal/modules/identity/app/activate.go server/internal/modules/identity/app/deactivate.go server/internal/modules/identity/app/login.go server/internal/modules/identity/app/reset_password.go server/internal/modules/identity/app/set_email.go server/internal/modules/identity/domain/email.go server/internal/modules/identity/domain/email_test.go server/internal/modules/identity/domain/url.go server/internal/modules/identity/domain/url_test.go server/internal/modules/identity/domain/user.go server/internal/shared/email.go server/internal/shared/email_test.go server/internal/shared/error.go server/internal/shared/timezone.go server/internal/shared/timezone_test.go server/internal/shared/url.go server/internal/shared/url_test.go
```
```bash
git commit -m "refactor(M3/P1): move the e-mail, web address and time zone rules to shared

Two modules must agree on them (M3 design 3.13): identity keeps its display
name rule and calls shared for the rest; the time zone rule is new.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `identity/domain` 不再有邮箱、网址的规则；`identity` 的全部测试不改而通过；`TestValidTimezone` 通过。

---

### Task 5: `access` 的规则表和判定

**Files:**
- Create: `server/internal/modules/access/domain/decide.go`、`server/internal/modules/access/domain/decide_test.go`、`server/internal/modules/access/domain/rules.go`、`server/internal/modules/access/domain/rules_test.go`

**Interfaces:**
- Produces（spec 2.7，M3 设计 3.4、6.5）：
  - `domain.Level`（`LevelWorkspace`、`LevelProject`、`LevelVisible`）、`domain.Rule{Level; Roles []shared.Role}`；
  - 规则表 `rules`，本 Phase 一行：`"workspace.read": {LevelWorkspace, [Admin, Member, Guest]}`；`RuleFor(action) (Rule, bool)`、`RuleKeys()`（排序）；
  - `domain.Membership{Active bool; Role shared.Role}`、`domain.Project{Public bool; Member Membership}`、`domain.Facts{Workspace Membership; Project *Project}`；
  - `Decide(rule, facts) (shared.Grant, error)`：工作区级：不是有效成员 → `shared.ErrNotVisible`，角色不在集合中 → `shared.Forbidden()`；项目级和"看得到即可"按 M3 设计 3.4（工作区管理员、项目的有效成员、公开项目的工作区成员或管理员看得到；项目级要求是项目的有效成员，并且项目角色在集合中或是工作区管理员）；角色按集合比较；级别不认识 → 内部错误。
- 使用者：Task 6 的 `Authorizer`；Task 12 的完整性测试（`RuleKeys`）。

**Tests:**
- `decide_test.go`：
  - `TestDecideAtTheWorkspaceLevel`：4 组角色（全部、管理员和成员、管理员、没有）× 7 种身份（管理员、成员、访客、从来不是、已被移出、工作区已删除、角色 10）；允许时 `Grant` 只带工作区角色。
  - `TestTheWorkspaceLevelIgnoresTheProject`：项目的事实不改变工作区级的判定。
  - `TestDecideAtTheProjectLevels`：4 条规则 × 13 种身份（M3 设计 9.2 项目级的各列、两个未知角色），另加每种身份没有项目时是 `ErrNotVisible`。
  - `TestTheGrantCarriesTheRoles`：`Grant` 带两级角色和 `ProjectAdmin`。
  - `TestARuleOfNoKnownLevelIsAnError`：不是 `ErrNotVisible`，也不是 403。
- `rules_test.go`：
  - `TestEveryRuleDecidesItsCells`：`tableCells` 写死规则表每一行对 7 种身份的答案；规则表的键必须恰好是 `tableCells` 的键：删掉、放宽、收窄一行，或加一行而不写它的格子，都失败。
  - `TestAnActionWithoutARowHasNoRule`。

- [ ] **Step 1: 规则表和判定**

`server/internal/modules/access/domain/rules.go`（新文件，57 行）：

````file server/internal/modules/access/domain/rules.go
// Package domain holds the access module's rules (M3 design 3.4, 6.4): the
// rule table and the decision, pure functions over the facts that the ports
// read.
package domain

import (
	"maps"
	"slices"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Level is where a rule decides (M3 design 6.4).
type Level int

// The three levels.
const (
	// LevelWorkspace allows an active member of the target's workspace whose
	// workspace role is in the rule's roles.
	LevelWorkspace Level = iota + 1
	// LevelProject allows a caller who sees the target project and is its
	// active member, with a project role in the rule's roles or as the
	// workspace's admin.
	LevelProject
	// LevelVisible allows every caller who sees the target project: only
	// project.read and project.join.
	LevelVisible
)

// Rule is a row of the rule table: its level and the roles it allows, the
// workspace roles at LevelWorkspace and the project roles at LevelProject.
// LevelVisible has none.
type Rule struct {
	Level Level
	Roles []shared.Role
}

// rules is the rule table (M3 design 3.4): one row per action, keyed by the
// name its module declares (workspace/domain/actions.go and the like). Its
// rows are the rows of the permission matrix (M3 design 9.2); an action
// without a row is refused. Each phase adds the rows of its operations, and
// bootstrap's completeness test holds the keys equal to the modules'
// Actions().
var rules = map[shared.Action]Rule{
	"workspace.read": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
}

// RuleFor returns the row of action; ok is false when the table has none.
func RuleFor(action shared.Action) (rule Rule, ok bool) {
	rule, ok = rules[action]
	return rule, ok
}

// RuleKeys lists the actions the table has a row for, sorted.
func RuleKeys() []shared.Action {
	return slices.Sorted(maps.Keys(rules))
}
````

`server/internal/modules/access/domain/decide.go`（新文件，87 行）：

````file server/internal/modules/access/domain/decide.go
package domain

import (
	"fmt"
	"slices"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Membership is the caller's membership of a workspace or a project as a
// port read it. Active is false when there is none that counts: no row,
// is_active false, the row deleted, or the workspace or project deleted.
type Membership struct {
	Active bool
	Role   shared.Role
}

// Project is what the ports read about the target project.
type Project struct {
	Public bool       // network = 2
	Member Membership // the caller's membership of the project
}

// Facts are what the ports read about the caller and the target (M3 design
// 3.4). Project is nil when the target has no project that counts: a
// workspace-level target, or a project that does not exist, is deleted or
// belongs to another workspace. An archived project counts (M3 design 3.19).
type Facts struct {
	Workspace Membership
	Project   *Project
}

// Decide applies rule to f (M3 design 3.4): the Grant when the caller may
// act, shared.ErrNotVisible when the caller cannot see the target, and
// shared.Forbidden when the caller sees it but the rule does not allow the
// caller's role. A caller who is not an active member of the workspace sees
// nothing in it. Roles are compared by membership in a set, never by order:
// a role outside the three is allowed nothing and sees no project it is not
// a member of. A rule of no known level is an error: nothing is allowed.
func Decide(rule Rule, f Facts) (shared.Grant, error) {
	switch rule.Level {
	case LevelWorkspace:
		return decideWorkspace(rule, f.Workspace)
	case LevelProject, LevelVisible:
		return decideProject(rule, f)
	}
	return shared.Grant{}, fmt.Errorf("access: a rule of unknown level %d", rule.Level)
}

func decideWorkspace(rule Rule, ws Membership) (shared.Grant, error) {
	switch {
	case !ws.Active:
		return shared.Grant{}, shared.ErrNotVisible
	case !slices.Contains(rule.Roles, ws.Role):
		return shared.Grant{}, shared.Forbidden()
	}
	return shared.Grant{WorkspaceRole: ws.Role}, nil
}

func decideProject(rule Rule, f Facts) (shared.Grant, error) {
	ws, p := f.Workspace, f.Project
	if !ws.Active || p == nil || !sees(ws.Role, *p) {
		return shared.Grant{}, shared.ErrNotVisible
	}
	grant := shared.Grant{WorkspaceRole: ws.Role}
	if p.Member.Active {
		grant.ProjectRole = p.Member.Role
		grant.ProjectAdmin = p.Member.Role == shared.RoleAdmin || ws.Role == shared.RoleAdmin
	}
	if rule.Level == LevelVisible {
		return grant, nil
	}
	if p.Member.Active && (slices.Contains(rule.Roles, p.Member.Role) || ws.Role == shared.RoleAdmin) {
		return grant, nil
	}
	return shared.Grant{}, shared.Forbidden()
}

// sees reports whether an active workspace member with role wsRole sees p:
// the workspace's admin sees every project, an active project member sees
// the project, and a workspace member or admin sees a public project. A
// guest sees only the projects he is a member of, public or not (Plane
// views/project/base.py:198-222).
func sees(wsRole shared.Role, p Project) bool {
	return wsRole == shared.RoleAdmin || p.Member.Active ||
		(p.Public && slices.Contains([]shared.Role{shared.RoleMember, shared.RoleAdmin}, wsRole))
}
````

- [ ] **Step 2: 判定表的测试**

`server/internal/modules/access/domain/decide_test.go`（新文件，196 行）：

````file server/internal/modules/access/domain/decide_test.go
package domain_test

import (
	"errors"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/access/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// outcome is a cell of a decision table: allowed, 403 forbidden or 404 not
// visible.
type outcome string

const (
	allowed   outcome = "ok"
	forbidden outcome = "403"
	invisible outcome = "404"
)

// outcomeOf names what Decide answered; any other error fails the test.
func outcomeOf(t *testing.T, err error) outcome {
	t.Helper()
	switch {
	case err == nil:
		return allowed
	case errors.Is(err, shared.ErrNotVisible):
		return invisible
	case errors.Is(err, shared.Forbidden()):
		return forbidden
	}
	t.Fatalf("Decide() = %v, want nil, ErrNotVisible or Forbidden", err)
	return ""
}

var (
	admin  = domain.Membership{Active: true, Role: shared.RoleAdmin}
	member = domain.Membership{Active: true, Role: shared.RoleMember}
	guest  = domain.Membership{Active: true, Role: shared.RoleGuest}
	none   = domain.Membership{}
)

// The identities of M3 design 9.2's workspace-level columns, as the
// WorkspaceRoles port reports them. The three that are not active members
// keep the role of their row: only Active may count.
var workspaceIdentities = []struct {
	name string
	ws   domain.Membership
}{
	{"admin", admin},
	{"member", member},
	{"guest", guest},
	{"never a member", none},
	{"removed", domain.Membership{Active: false, Role: shared.RoleAdmin}},
	{"workspace deleted", domain.Membership{Active: false, Role: shared.RoleMember}},
	{"role outside the three", domain.Membership{Active: true, Role: 10}},
}

func TestDecideAtTheWorkspaceLevel(t *testing.T) {
	all := []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}
	tests := []struct {
		name  string
		roles []shared.Role
		want  []outcome // in the order of workspaceIdentities
	}{
		{"every role", all, []outcome{allowed, allowed, allowed, invisible, invisible, invisible, forbidden}},
		{"admins and members", all[:2], []outcome{allowed, allowed, forbidden, invisible, invisible, invisible, forbidden}},
		{"admins", all[:1], []outcome{allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden}},
		{"no role", nil, []outcome{forbidden, forbidden, forbidden, invisible, invisible, invisible, forbidden}},
	}
	for _, tt := range tests {
		rule := domain.Rule{Level: domain.LevelWorkspace, Roles: tt.roles}
		for i, id := range workspaceIdentities {
			grant, err := domain.Decide(rule, domain.Facts{Workspace: id.ws})
			if got := outcomeOf(t, err); got != tt.want[i] {
				t.Errorf("%s, %s: Decide() = %s, want %s", tt.name, id.name, got, tt.want[i])
			}
			if err == nil && grant != (shared.Grant{WorkspaceRole: id.ws.Role}) {
				t.Errorf("%s, %s: Grant = %+v, want the workspace role %d alone", tt.name, id.name, grant, id.ws.Role)
			}
		}
	}
}

// A workspace-level rule reads no project: facts about one change nothing.
func TestTheWorkspaceLevelIgnoresTheProject(t *testing.T) {
	rule := domain.Rule{Level: domain.LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}}
	p := &domain.Project{Public: true, Member: admin}
	if _, err := domain.Decide(rule, domain.Facts{Workspace: member, Project: p}); outcomeOf(t, err) != forbidden {
		t.Errorf("a member who administers a project: %v, want forbidden", err)
	}
}

// The columns of M3 design 9.2's project-level table, as the ports report
// them: PA, PM and PG are project admin, member and guest (workspace member,
// member, guest); PM+WA a project member who is the workspace's admin; WA-,
// WM- and WG- a workspace admin, member and guest who are not project
// members; P-before a workspace member who left a private project where he
// was its admin; X a caller who is not an active workspace member, though
// his project membership is still active.
var projectIdentities = []struct {
	name   string
	ws     domain.Membership
	public bool
	member domain.Membership
}{
	{"PA", member, false, admin},
	{"PM", member, false, member},
	{"PG", guest, false, guest},
	{"PM+WA", admin, false, member},
	{"WA- private", admin, false, none},
	{"WM- public", member, true, none},
	{"WM- private", member, false, none},
	{"WG- public", guest, true, none},
	{"WG- private", guest, false, none},
	{"P-before", member, false, domain.Membership{Active: false, Role: shared.RoleAdmin}},
	{"X", none, true, admin},
	{"workspace role outside the three, public", domain.Membership{Active: true, Role: 10}, true, none},
	{"project role outside the three", member, false, domain.Membership{Active: true, Role: 10}},
}

func TestDecideAtTheProjectLevels(t *testing.T) {
	all := []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}
	const ok, no, hidden = allowed, forbidden, invisible
	tests := []struct {
		name string
		rule domain.Rule
		want []outcome // in the order of projectIdentities
	}{
		// updateProject, addProjectMembers, createState… (9.2)
		{"project admins", domain.Rule{Level: domain.LevelProject, Roles: all[:1]},
			[]outcome{ok, no, no, ok, no, no, hidden, hidden, hidden, hidden, hidden, hidden, no}},
		{"project admins and members", domain.Rule{Level: domain.LevelProject, Roles: all[:2]},
			[]outcome{ok, ok, no, ok, no, no, hidden, hidden, hidden, hidden, hidden, hidden, no}},
		// listStates, listLabels, getProjectPreferences… (9.2)
		{"every project role", domain.Rule{Level: domain.LevelProject, Roles: all},
			[]outcome{ok, ok, ok, ok, no, no, hidden, hidden, hidden, hidden, hidden, hidden, no}},
		// getProject (9.2)
		{"seeing the project", domain.Rule{Level: domain.LevelVisible},
			[]outcome{ok, ok, ok, ok, ok, ok, hidden, hidden, hidden, hidden, hidden, hidden, ok}},
	}
	for _, tt := range tests {
		for i, id := range projectIdentities {
			facts := domain.Facts{Workspace: id.ws, Project: &domain.Project{Public: id.public, Member: id.member}}
			_, err := domain.Decide(tt.rule, facts)
			if got := outcomeOf(t, err); got != tt.want[i] {
				t.Errorf("%s, %s: Decide() = %s, want %s", tt.name, id.name, got, tt.want[i])
			}
			// A project that does not count (none, deleted, another
			// workspace's) is seen by no one.
			facts.Project = nil
			if _, err := domain.Decide(tt.rule, facts); outcomeOf(t, err) != invisible {
				t.Errorf("%s, %s, no project: Decide() = %v, want not visible", tt.name, id.name, err)
			}
		}
	}
}

// The Grant carries the roles the decision read, and ProjectAdmin for a
// project member who is its admin or the workspace's.
func TestTheGrantCarriesTheRoles(t *testing.T) {
	visible := domain.Rule{Level: domain.LevelVisible}
	tests := []struct {
		name string
		ws   domain.Membership
		p    domain.Project
		want shared.Grant
	}{
		{"PA", member, domain.Project{Member: admin}, shared.Grant{WorkspaceRole: 15, ProjectRole: 20, ProjectAdmin: true}},
		{"PM", member, domain.Project{Member: member}, shared.Grant{WorkspaceRole: 15, ProjectRole: 15}},
		{"PG", guest, domain.Project{Member: guest}, shared.Grant{WorkspaceRole: 5, ProjectRole: 5}},
		{"PM+WA", admin, domain.Project{Member: member}, shared.Grant{WorkspaceRole: 20, ProjectRole: 15, ProjectAdmin: true}},
		{"WA-", admin, domain.Project{}, shared.Grant{WorkspaceRole: 20}},
		{"WM- public", member, domain.Project{Public: true}, shared.Grant{WorkspaceRole: 15}},
	}
	for _, tt := range tests {
		p := tt.p
		grant, err := domain.Decide(visible, domain.Facts{Workspace: tt.ws, Project: &p})
		if err != nil || grant != tt.want {
			t.Errorf("%s: Decide() = %+v, %v; want %+v", tt.name, grant, err, tt.want)
		}
	}
}

// A rule of no known level allows nothing, and is not a refusal the caller
// could act on: an internal error.
func TestARuleOfNoKnownLevelIsAnError(t *testing.T) {
	for _, level := range []domain.Level{0, 4} {
		rule := domain.Rule{Level: level, Roles: []shared.Role{shared.RoleAdmin}}
		_, err := domain.Decide(rule, domain.Facts{Workspace: admin, Project: &domain.Project{Member: admin}})
		var se *shared.Error
		if err == nil || errors.As(err, &se) {
			t.Errorf("level %d: Decide() = %v, want an internal error", level, err)
		}
	}
}
````

`server/internal/modules/access/domain/rules_test.go`（新文件，63 行）：

````file server/internal/modules/access/domain/rules_test.go
package domain_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/access/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// tableCells is every row of the rule table decided for every identity of
// M3 design 9.2, the matrix's cells without HTTP: a row deleted, widened or
// narrowed fails here, and so does a row added to the table without its
// cells here. A workspace-level row has a cell per workspaceIdentities, a
// project-level one per projectIdentities.
var tableCells = map[shared.Action][]outcome{
	// admin, member, guest, never a member, removed, workspace deleted, a role outside the three
	"workspace.read": {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
}

// cells decides rule for each identity of its level.
func cells(t *testing.T, rule domain.Rule) []outcome {
	t.Helper()
	var out []outcome
	if rule.Level == domain.LevelWorkspace {
		for _, id := range workspaceIdentities {
			_, err := domain.Decide(rule, domain.Facts{Workspace: id.ws})
			out = append(out, outcomeOf(t, err))
		}
		return out
	}
	for _, id := range projectIdentities {
		_, err := domain.Decide(rule, domain.Facts{Workspace: id.ws, Project: &domain.Project{Public: id.public, Member: id.member}})
		out = append(out, outcomeOf(t, err))
	}
	return out
}

func TestEveryRuleDecidesItsCells(t *testing.T) {
	if got, want := domain.RuleKeys(), slices.Sorted(maps.Keys(tableCells)); !slices.Equal(got, want) {
		t.Fatalf("the rule table has rows %q; the cells here are for %q", got, want)
	}
	for _, action := range domain.RuleKeys() {
		rule, ok := domain.RuleFor(action)
		if !ok {
			t.Fatalf("RuleFor(%q) found no row, though RuleKeys lists it", action)
		}
		if got := cells(t, rule); !slices.Equal(got, tableCells[action]) {
			t.Errorf("%s decides %q, want %q", action, got, tableCells[action])
		}
	}
}

// An action the table has no row for is found by no lookup: the Authorizer
// refuses it.
func TestAnActionWithoutARowHasNoRule(t *testing.T) {
	for _, action := range []shared.Action{"", "workspace.delete", "WORKSPACE.READ", "workspace.read "} {
		if rule, ok := domain.RuleFor(action); ok {
			t.Errorf("RuleFor(%q) = %+v, want none", action, rule)
		}
	}
}
````

- [ ] **Step 3: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/access/domain/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/access/domain/decide.go server/internal/modules/access/domain/decide_test.go server/internal/modules/access/domain/rules.go server/internal/modules/access/domain/rules_test.go
```
```bash
git commit -m "feat(M3/P1): the access module's rule table and decision

A pure decision over the caller's memberships at the three levels of M3
design 3.4, the roles compared as sets; the table's one row is
workspace.read, and each row's answer for every identity is pinned.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 判定表的 5 个测试和规则表的 2 个测试通过；`access/domain` 只导入 `shared` 和标准库。

---

### Task 6: `access` 的 `Authorizer` 和模块入口

**Files:**
- Create: `server/internal/modules/access/app/authorizer.go`、`server/internal/modules/access/app/authorizer_test.go`、`server/internal/modules/access/app/ports.go`、`server/internal/modules/access/module.go`

**Interfaces:**
- Produces（spec 2.7，M3 设计 6.4–6.6）：
  - `app.WorkspaceRoles` 接口：`ActiveRole(ctx, workspaceID, userID uuid.UUID) (role shared.Role, ok bool, err error)`；
  - `app.NewAuthorizer(roles) *Authorizer`；`Authorize` 先查规则，没有 → `fmt.Errorf("access: no rule for action %q", action)`，什么都不读；再每次调用都读调用者在目标工作区的角色（不缓存），端口的错误包一层返回；判定只用工作区的事实（项目的端口随 P4 加入）；
  - `access.Deps{WorkspaceRoles}`、`access.New(Deps) shared.Authorizer`、`access.RuleKeys() []shared.Action`。
- 使用者：Task 11 的组合（`workspace.Provide(pool).WorkspaceRoles` 交给 `access.New`）；Task 12 的完整性测试。

**Tests:**（`authorizer_test.go`，假端口按参数回答并记下每次调用的参数和 ctx）
- `TestAuthorizeReadsTheCallersRoleInTheTargetsWorkspace`：两个用户 × 两个工作区，每个答案是它自己那一对的；端口收到调用者的 ctx。
- `TestAuthorizeReadsOnEveryCall`：两次调用之间成员关系结束，第二次看不到。
- `TestAuthorizeRefusesAnActionWithoutARule`：内部错误，不是 `ErrNotVisible`、不是 403；端口没有被调用。
- `TestAuthorizeReturnsThePortsError`。

- [ ] **Step 1: 端口、`Authorizer`、模块入口**

`server/internal/modules/access/app/ports.go`（新文件，22 行）：

````file server/internal/modules/access/app/ports.go
// Package app holds the access module's Authorizer (M3 design 6.4): it reads
// the facts through ports, each implemented by the module that owns the
// table, and decides with the rule table of domain.
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// WorkspaceRoles reads a caller's membership of a workspace (M3 design 6.5).
// The workspace module implements it (workspace.Provide); bootstrap wires it.
type WorkspaceRoles interface {
	// ActiveRole returns userID's role in workspaceID when the membership is
	// active: its row is_active and not deleted, and the workspace not
	// deleted. ok is false otherwise. It reads in the transaction ctx
	// carries, so a write that locked the workspace row reads the role
	// committed before its lock (M3 design 3.6 convention 2).
	ActiveRole(ctx context.Context, workspaceID, userID uuid.UUID) (role shared.Role, ok bool, err error)
}
````

`server/internal/modules/access/app/authorizer.go`（新文件，39 行）：

````file server/internal/modules/access/app/authorizer.go
package app

import (
	"context"
	"fmt"

	"github.com/open-nerve/NerveProject/server/internal/modules/access/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Authorizer implements shared.Authorizer (M3 design 3.4): every call reads
// the facts afresh, nothing is cached, so a removal or a change of role
// takes effect at the next request.
type Authorizer struct {
	roles WorkspaceRoles
}

// NewAuthorizer returns the Authorizer that reads through roles.
func NewAuthorizer(roles WorkspaceRoles) *Authorizer {
	return &Authorizer{roles: roles}
}

// Authorize decides whether actor may do action on t. An action without a
// row in the rule table is an error before anything is read. The facts are
// the caller's workspace membership; no project is read, since the rule
// table has no project-level row yet, so a project-level rule would see no
// project and answer shared.ErrNotVisible. The ProjectAccess port adds the
// project's facts with the projects (M3 design 6.5).
func (a *Authorizer) Authorize(ctx context.Context, actor shared.Actor, action shared.Action, t shared.Target) (shared.Grant, error) {
	rule, ok := domain.RuleFor(action)
	if !ok {
		return shared.Grant{}, fmt.Errorf("access: no rule for action %q", action)
	}
	role, active, err := a.roles.ActiveRole(ctx, t.WorkspaceID, actor.UserID)
	if err != nil {
		return shared.Grant{}, fmt.Errorf("access: read the workspace role: %w", err)
	}
	return domain.Decide(rule, domain.Facts{Workspace: domain.Membership{Active: active, Role: role}})
}
````

`server/internal/modules/access/module.go`（新文件，28 行）：

````file server/internal/modules/access/module.go
// Package access decides what a caller may do in a workspace or a project
// (M3 design 3.3, 3.4, 6.4). It has no table and no operation: it reads the
// memberships through ports that the modules owning them implement, and
// every module asks it through shared.Authorizer.
package access

import (
	"github.com/open-nerve/NerveProject/server/internal/modules/access/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/access/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Deps are the ports the Authorizer reads through; bootstrap takes them from
// the other modules' Provide (M3 design 6.6).
type Deps struct {
	WorkspaceRoles app.WorkspaceRoles
}

// New returns the Authorizer (M3 design 6.6, step 3).
func New(d Deps) shared.Authorizer {
	return app.NewAuthorizer(d.WorkspaceRoles)
}

// RuleKeys lists the actions the rule table has a row for; bootstrap's test
// holds them equal to the union of the modules' Actions() (M3 design 3.4).
func RuleKeys() []shared.Action {
	return domain.RuleKeys()
}
````

- [ ] **Step 2: 测试**

`server/internal/modules/access/app/authorizer_test.go`（新文件，117 行）：

````file server/internal/modules/access/app/authorizer_test.go
package app_test

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type ctxKey struct{}

// membership is one (workspace, user) pair of fakeRoles.
type membership struct{ workspace, user uuid.UUID }

// fakeRoles answers the role of each (workspace, user) it holds, and records
// every call with the value of ctxKey in its context.
type fakeRoles struct {
	roles map[membership]shared.Role
	err   error
	calls []string
}

func (f *fakeRoles) ActiveRole(ctx context.Context, workspaceID, userID uuid.UUID) (shared.Role, bool, error) {
	f.calls = append(f.calls, workspaceID.String()+" "+userID.String()+" "+ctx.Value(ctxKey{}).(string))
	if f.err != nil {
		return 0, false, f.err
	}
	role, ok := f.roles[membership{workspaceID, userID}]
	return role, ok, nil
}

var (
	w1, w2 = uuid.NewV7(), uuid.NewV7()
	a, b   = uuid.NewV7(), uuid.NewV7()
)

// Authorize reads the caller's role in the target's workspace, in the
// caller's context: two users, two workspaces, and each answer is the one of
// its own pair.
func TestAuthorizeReadsTheCallersRoleInTheTargetsWorkspace(t *testing.T) {
	roles := &fakeRoles{roles: map[membership]shared.Role{
		{w1, a}: shared.RoleAdmin, {w2, a}: shared.RoleGuest, {w1, b}: shared.RoleMember,
	}}
	auth := app.NewAuthorizer(roles)
	tests := []struct {
		user, workspace uuid.UUID
		want            shared.Role // 0: not visible
	}{
		{a, w1, shared.RoleAdmin},
		{a, w2, shared.RoleGuest},
		{b, w1, shared.RoleMember},
		{b, w2, 0},
	}
	for _, tt := range tests {
		roles.calls = nil
		ctx := context.WithValue(context.Background(), ctxKey{}, "tx")
		grant, err := auth.Authorize(ctx, shared.Actor{UserID: tt.user}, "workspace.read", shared.Target{WorkspaceID: tt.workspace})
		want := tt.workspace.String() + " " + tt.user.String() + " tx"
		if len(roles.calls) != 1 || roles.calls[0] != want {
			t.Errorf("ActiveRole calls = %q, want [%q]", roles.calls, want)
		}
		if tt.want == 0 {
			if !errors.Is(err, shared.ErrNotVisible) {
				t.Errorf("user %s in %s: Authorize() = %+v, %v; want ErrNotVisible", tt.user, tt.workspace, grant, err)
			}
			continue
		}
		if err != nil || grant != (shared.Grant{WorkspaceRole: tt.want}) {
			t.Errorf("user %s in %s: Authorize() = %+v, %v; want role %d", tt.user, tt.workspace, grant, err, tt.want)
		}
	}
}

// Nothing is cached: a membership that ends between two calls is not
// visible at the second.
func TestAuthorizeReadsOnEveryCall(t *testing.T) {
	roles := &fakeRoles{roles: map[membership]shared.Role{{w1, a}: shared.RoleAdmin}}
	auth := app.NewAuthorizer(roles)
	ctx := context.WithValue(context.Background(), ctxKey{}, "request")
	target := shared.Target{WorkspaceID: w1}
	if _, err := auth.Authorize(ctx, shared.Actor{UserID: a}, "workspace.read", target); err != nil {
		t.Fatalf("first Authorize() = %v", err)
	}
	delete(roles.roles, membership{w1, a})
	if _, err := auth.Authorize(ctx, shared.Actor{UserID: a}, "workspace.read", target); !errors.Is(err, shared.ErrNotVisible) {
		t.Errorf("Authorize() after the membership ended = %v, want ErrNotVisible", err)
	}
}

// An action without a row fails closed, before anything is read, and is not
// a refusal the caller could act on: an internal error.
func TestAuthorizeRefusesAnActionWithoutARule(t *testing.T) {
	roles := &fakeRoles{roles: map[membership]shared.Role{{w1, a}: shared.RoleAdmin}}
	auth := app.NewAuthorizer(roles)
	ctx := context.WithValue(context.Background(), ctxKey{}, "request")
	grant, err := auth.Authorize(ctx, shared.Actor{UserID: a}, "workspace.delete", shared.Target{WorkspaceID: w1})
	var se *shared.Error
	if err == nil || errors.As(err, &se) || grant != (shared.Grant{}) {
		t.Errorf("Authorize() = %+v, %v; want an internal error", grant, err)
	}
	if len(roles.calls) != 0 {
		t.Errorf("ActiveRole calls = %q, want none", roles.calls)
	}
}

// The port's failure is Authorize's.
func TestAuthorizeReturnsThePortsError(t *testing.T) {
	failure := errors.New("connection reset")
	auth := app.NewAuthorizer(&fakeRoles{err: failure})
	ctx := context.WithValue(context.Background(), ctxKey{}, "request")
	if _, err := auth.Authorize(ctx, shared.Actor{UserID: a}, "workspace.read", shared.Target{WorkspaceID: w1}); !errors.Is(err, failure) {
		t.Errorf("Authorize() = %v, want %v", err, failure)
	}
}
````

- [ ] **Step 3: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/access/...`
Expected: 两个 `ok`（`access` 的根包没有测试文件）。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/access/app/authorizer.go server/internal/modules/access/app/authorizer_test.go server/internal/modules/access/app/ports.go server/internal/modules/access/module.go
```
```bash
git commit -m "feat(M3/P1): the access module's Authorizer

Authorize reads the caller's role in the target's workspace through the
WorkspaceRoles port on every call and decides by the rule table; an action
without a row is an internal error before anything is read.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 4 个测试通过；`access` 不导入任何别的模块（archtest 通过）。

---

### Task 7: `workspace` 的领域：校验、保留名单、操作名、错误

**Files:**
- Create: `server/internal/modules/workspace/domain/actions.go`、`server/internal/modules/workspace/domain/errors.go`、`server/internal/modules/workspace/domain/reserved.go`、`server/internal/modules/workspace/domain/reserved_slugs.txt`、`server/internal/modules/workspace/domain/reserved_test.go`、`server/internal/modules/workspace/domain/workspace.go`、`server/internal/modules/workspace/domain/workspace_test.go`

**Interfaces:**
- Produces（spec 2.8，M3 设计 3.10、3.11、5.1–5.3）：
  - `domain.Workspace`、`domain.NewWorkspace`、`domain.DefaultTimezone = "UTC"`；
  - `domain.SlugReason`（`SlugInvalid`、`SlugReserved`、`SlugTaken`）、`domain.CheckSlug(slug) SlugReason`；
  - `domain.CheckNewWorkspace(w) error`：一次报出全部字段的 422 `validation_failed`（名称、slug、规模、时区的规则见 spec 2.8）；
  - `reserved_slugs.txt`（嵌入）和 `domain.ReservedSlugs{App, Server, Reserved}`、`All()`、`domain.Reserved()`（副本）；
  - `domain.ActionRead = "workspace.read"`、`domain.Actions()`；
  - `domain.ErrNotFound`（404 `workspace.not_found`）、`ErrCreationDisabled`（403 `workspace.creation_disabled`）、`ErrSlugTaken`（409 `workspace.slug_taken`）、`ErrAccountNotFound`（404 `workspace.account_not_found`）、`ErrAccountDeactivated`（403 `workspace.account_deactivated`）；后两个只给命令行（spec 第 3 节第 7 条）。
- 使用者：Task 9–11、14。

**Tests:**
- `workspace_test.go`：`TestCheckNewWorkspaceAcceptsValidWorkspaces`（一个字符和 80 个汉字的名称、非拉丁的数字、1 和 48 个字符的 slug、六个规模、给或不给时区）；`TestCheckNewWorkspaceReportsEveryField`（每条规则一行，另有一行四个字段同时出错、四条都报）；`TestCheckSlug`（合格、不合格、保留；`login` 不保留）。
- `reserved_test.go`：`TestTheReservedListIsWellFormed`（三段都有、互不重复、都合 slug 的写法，"预留"一段恰好是 `admin`、`docs`、`help`、`static`）；`TestParseReserved`（注释、空行、首尾空白、同一段出现两次时合并；段之前的名字是错误）；`TestReservedReturnsACopy`（改返回值不改名单）。

- [ ] **Step 1: 领域**

`server/internal/modules/workspace/domain/workspace.go`（新文件，143 行）：

````file server/internal/modules/workspace/domain/workspace.go
// Package domain holds the workspace module's rules (M3 design 6.2): pure
// functions and values.
package domain

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Workspace is a workspace as a member sees it (M3 design 5.2): its columns,
// the caller's role and the number of active members.
type Workspace struct {
	ID               uuid.UUID
	Name             string
	Slug             string
	OrganizationSize *string // nil: not given
	Timezone         string
	Role             shared.Role // the caller's
	TotalMembers     int
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// NewWorkspace is what the caller asks for when creating a workspace (M3
// design 5.1). A nil Timezone asks for DefaultTimezone.
type NewWorkspace struct {
	Name             string
	Slug             string
	OrganizationSize *string
	Timezone         *string
}

// DefaultTimezone is a new workspace's time zone when none is given, the
// column's default (M3 design 4.2).
const DefaultTimezone = "UTC"

// The lengths of workspaces.name, varchar(80), and workspaces.slug,
// varchar(48), in characters.
const (
	maxNameLength = 80
	maxSlugLength = 48
)

// organizationSizes are the web form's options (web/packages/constants/src/
// workspace.ts:10), which the column's CHECK holds too (M3 design 4.2).
var organizationSizes = []string{"Just myself", "2-10", "11-50", "51-200", "201-500", "500+"}

// slugPattern is a slug's spelling: lower case only (M3 design 3.10).
var slugPattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

// SlugReason is why a slug cannot name a new workspace (M3 design 5.1, the
// reason of SlugAvailability).
type SlugReason string

// The reasons.
const (
	SlugInvalid  SlugReason = "invalid"  // not 1–48 of a-z, 0-9, - and _
	SlugReserved SlugReason = "reserved" // on the reserved list
	SlugTaken    SlugReason = "taken"    // an undeleted workspace has it
)

// CheckSlug returns why slug cannot name a new workspace without looking at
// the workspaces, invalid or reserved, or "" when it can. Whether it is taken
// is the repository's to say.
func CheckSlug(slug string) SlugReason {
	switch {
	case utf8.RuneCountInString(slug) > maxSlugLength || !slugPattern.MatchString(slug):
		return SlugInvalid
	case isReserved(slug):
		return SlugReserved
	}
	return ""
}

// CheckNewWorkspace checks w (M3 design 3.10, 3.13), as Plane's serializer
// does (serializers/workspace.py:48-67) with the slug in lower case only:
//   - a name of 1–80 characters, with a letter or a digit (Unicode), without
//     a web address, and without NUL, which the database cannot store;
//   - a slug of 1–48 lower-case letters, digits, - and _, not reserved;
//   - an organization size of the web form's options;
//   - a time zone that shared.ValidTimezone accepts.
//
// Every problem is reported at once, as one 422 validation_failed. Whether
// the slug is taken is the database's to say.
func CheckNewWorkspace(w NewWorkspace) error {
	var fields []shared.FieldError
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
	}
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}

func checkName(name string) *shared.FieldError {
	field := "name"
	switch {
	case name == "":
		return &shared.FieldError{Field: field, Code: shared.FieldTooShort, Message: "must not be empty"}
	case utf8.RuneCountInString(name) > maxNameLength:
		return &shared.FieldError{Field: field, Code: shared.FieldTooLong, Message: fmt.Sprintf("must be at most %d characters", maxNameLength)}
	case strings.ContainsRune(name, 0):
		return &shared.FieldError{Field: field, Code: shared.FieldInvalidFormat, Message: "must not contain a NUL character"}
	case !strings.ContainsFunc(name, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) }):
		return &shared.FieldError{Field: field, Code: shared.FieldInvalidFormat, Message: "must contain a letter or a digit"}
	case shared.ContainsURL(name):
		return &shared.FieldError{Field: field, Code: shared.FieldContainsURL, Message: "must not contain a web address"}
	}
	return nil
}

func checkSlug(slug string) *shared.FieldError {
	field := "slug"
	switch {
	case slug == "":
		return &shared.FieldError{Field: field, Code: shared.FieldTooShort, Message: "must not be empty"}
	case utf8.RuneCountInString(slug) > maxSlugLength:
		return &shared.FieldError{Field: field, Code: shared.FieldTooLong, Message: fmt.Sprintf("must be at most %d characters", maxSlugLength)}
	case CheckSlug(slug) == SlugInvalid:
		return &shared.FieldError{Field: field, Code: shared.FieldInvalidFormat, Message: "may hold only lower-case letters, digits, - and _"}
	case CheckSlug(slug) == SlugReserved:
		return &shared.FieldError{Field: field, Code: shared.FieldNotAllowed, Message: "is reserved"}
	}
	return nil
}
````

`server/internal/modules/workspace/domain/reserved_slugs.txt`（新文件，30 行）：

````file server/internal/modules/workspace/domain/reserved_slugs.txt
# 工作区 slug 的保留名单（M3 设计 3.10）。前端的副本 RESTRICTED_URLS 随前端的数据层删除，
# 之后前后端只有这一份。
# 一行一个名字，属于它上面最近的一段；空行和 # 开头的行不算。
#
# [app]：应用用到的顶层路径段，即 web/apps/web/app/routes/core.ts 的顶层静态路由段，
#   加上 web/apps/web/public/ 的顶层目录。核对它的 web vitest 随前端的数据层加入（M3 设计 9.5）。
#   invitations 是 /invitations 页的段：删掉这一页的改动（决策点 2）同时把它移出名单。
# [server]：服务端在前端页面之外自己回答的顶层路径，由 bootstrap 的 Go 测试核对。
# [reserved]：以后最可能加的顶层页面先占住的名字，与另两段不重复。

[app]
create-workspace
invitations
onboarding
settings
sign-up
workspace-invitations
icons

[server]
api
assets
healthz
readyz

[reserved]
admin
docs
help
static
````

`server/internal/modules/workspace/domain/reserved.go`（新文件，75 行）：

````file server/internal/modules/workspace/domain/reserved.go
package domain

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"
)

//go:embed reserved_slugs.txt
var reservedFile string

// ReservedSlugs is the reserved list's three sections (M3 design 3.10).
type ReservedSlugs struct {
	App      []string // the web app's top-level route segments and public/'s top-level directories
	Server   []string // the top-level paths the server answers itself, beside the web app's pages
	Reserved []string // names held for top-level pages to come
}

// All is the three sections together.
func (r ReservedSlugs) All() []string {
	return slices.Concat(r.App, r.Server, r.Reserved)
}

// reserved is the embedded list, parsed once. The file is part of the
// binary, so a list that does not parse stops every nerve at start.
var reserved = mustParseReserved(reservedFile)

// Reserved returns the reserved list.
func Reserved() ReservedSlugs {
	return ReservedSlugs{App: slices.Clone(reserved.App), Server: slices.Clone(reserved.Server), Reserved: slices.Clone(reserved.Reserved)}
}

func mustParseReserved(text string) ReservedSlugs {
	r, err := parseReserved(text)
	if err != nil {
		panic("workspace: reserved_slugs.txt: " + err.Error())
	}
	return r
}

// parseReserved reads the list: sections [app], [server] and [reserved],
// one name per line under each; blank lines and lines starting with # are
// comments.
func parseReserved(text string) (ReservedSlugs, error) {
	var r ReservedSlugs
	var section *[]string
	for i, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		switch line {
		case "":
			continue
		case "[app]":
			section = &r.App
		case "[server]":
			section = &r.Server
		case "[reserved]":
			section = &r.Reserved
		default:
			switch {
			case strings.HasPrefix(line, "#"):
				continue
			case section == nil:
				return ReservedSlugs{}, fmt.Errorf("line %d: %q is in no section", i+1, line)
			}
			*section = append(*section, line)
		}
	}
	return r, nil
}

// isReserved reports whether slug is on the list, in any section.
func isReserved(slug string) bool {
	return slices.Contains(reserved.All(), slug)
}
````

`server/internal/modules/workspace/domain/actions.go`（新文件，16 行）：

````file server/internal/modules/workspace/domain/actions.go
package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The workspace module's actions: the keys of its rows in the access
// module's rule table (M3 design 3.4).
const (
	// ActionRead is reading a workspace: getWorkspace.
	ActionRead shared.Action = "workspace.read"
)

// Actions lists the module's actions. bootstrap's test holds the union of
// every module's Actions equal to the rule table's keys (M3 design 3.4).
func Actions() []shared.Action {
	return []shared.Action{ActionRead}
}
````

`server/internal/modules/workspace/domain/errors.go`（新文件，24 行）：

````file server/internal/modules/workspace/domain/errors.go
package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The workspace module's errors (M3 design 5.3). api/modules/workspace.yaml
// declares the codes of those the API answers with in x-problem-codes; the
// last two are `nerve workspaces create`'s only (M3 design 3.11).
var (
	// ErrNotFound answers a workspace that does not exist, is deleted, or of
	// which the caller is not an active member: the same 404 for all three
	// (M3 design 8.2).
	ErrNotFound = shared.NewError(shared.KindNotFound, "workspace.not_found", "The workspace does not exist.")
	// ErrCreationDisabled answers a creation while workspace.creation_enabled
	// is false (M3 design 3.11).
	ErrCreationDisabled = shared.NewError(shared.KindForbidden, "workspace.creation_disabled", "Creating workspaces is disabled on this instance.")
	// ErrSlugTaken answers a creation with a slug an undeleted workspace has.
	ErrSlugTaken = shared.NewError(shared.KindConflict, "workspace.slug_taken", "A workspace with this slug exists.")
	// ErrAccountNotFound answers `nerve workspaces create` for an address no
	// account has.
	ErrAccountNotFound = shared.NewError(shared.KindNotFound, "workspace.account_not_found", "No account has this e-mail address.")
	// ErrAccountDeactivated answers `nerve workspaces create` for a
	// deactivated account, as read under the lock (M3 design 3.6 convention 6).
	ErrAccountDeactivated = shared.NewError(shared.KindForbidden, "workspace.account_deactivated", "The account is deactivated.")
)
````

- [ ] **Step 2: 测试**

`server/internal/modules/workspace/domain/workspace_test.go`（新文件，112 行）：

````file server/internal/modules/workspace/domain/workspace_test.go
package domain

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func ptr[T any](v T) *T { return &v }

func TestCheckNewWorkspaceAcceptsValidWorkspaces(t *testing.T) {
	for _, w := range []NewWorkspace{
		{Name: "Acme", Slug: "acme"},
		{Name: "a", Slug: "a"},
		{Name: strings.Repeat("工", 80), Slug: strings.Repeat("x", 48)},
		{Name: "研发部", Slug: "rd_team-2", OrganizationSize: ptr("Just myself"), Timezone: ptr("Asia/Shanghai")},
		{Name: "٣", Slug: "0", OrganizationSize: ptr("500+"), Timezone: ptr("UTC")},
		{Name: "-_ Team 7 _-", Slug: "-_-"},
	} {
		if err := CheckNewWorkspace(w); err != nil {
			t.Errorf("CheckNewWorkspace(%+v) = %v, want nil", w, err)
		}
	}
	for _, size := range organizationSizes {
		if err := CheckNewWorkspace(NewWorkspace{Name: "Acme", Slug: "acme", OrganizationSize: &size}); err != nil {
			t.Errorf("organization size %q: %v, want nil", size, err)
		}
	}
}

func TestCheckNewWorkspaceReportsEveryField(t *testing.T) {
	valid := NewWorkspace{Name: "Acme", Slug: "acme"}
	with := func(change func(*NewWorkspace)) NewWorkspace {
		w := valid
		change(&w)
		return w
	}
	name := func(n string) NewWorkspace { return with(func(w *NewWorkspace) { w.Name = n }) }
	slug := func(s string) NewWorkspace { return with(func(w *NewWorkspace) { w.Slug = s }) }
	field := func(f, code, message string) []shared.FieldError {
		return []shared.FieldError{{Field: f, Code: code, Message: message}}
	}
	tests := []struct {
		name string
		w    NewWorkspace
		want []shared.FieldError
	}{
		{"empty name", name(""), field("name", "too_short", "must not be empty")},
		{"name of 81 characters", name(strings.Repeat("工", 81)), field("name", "too_long", "must be at most 80 characters")},
		{"name with NUL", name("Ac\x00me"), field("name", "invalid_format", "must not contain a NUL character")},
		{"name of symbols only", name("-_________-"), field("name", "invalid_format", "must contain a letter or a digit")},
		{"name of spaces only", name("   "), field("name", "invalid_format", "must contain a letter or a digit")},
		{"name with a web address", name("Acme www.acme.io"), field("name", "contains_url", "must not contain a web address")},
		{"name with a dotted host", name("acme.io"), field("name", "contains_url", "must not contain a web address")},
		{"empty slug", slug(""), field("slug", "too_short", "must not be empty")},
		{"slug of 49 characters", slug(strings.Repeat("x", 49)), field("slug", "too_long", "must be at most 48 characters")},
		{"upper-case slug", slug("Acme"), field("slug", "invalid_format", "may hold only lower-case letters, digits, - and _")},
		{"slug with a dot", slug("acme.io"), field("slug", "invalid_format", "may hold only lower-case letters, digits, - and _")},
		{"slug with a space", slug("my team"), field("slug", "invalid_format", "may hold only lower-case letters, digits, - and _")},
		{"slug with a non-ASCII letter", slug("équipe"), field("slug", "invalid_format", "may hold only lower-case letters, digits, - and _")},
		{"slug of the app", slug("create-workspace"), field("slug", "not_allowed", "is reserved")},
		{"slug of the app's public directory", slug("icons"), field("slug", "not_allowed", "is reserved")},
		{"slug of the server", slug("api"), field("slug", "not_allowed", "is reserved")},
		{"slug held for later", slug("admin"), field("slug", "not_allowed", "is reserved")},
		{"unknown organization size", with(func(w *NewWorkspace) { w.OrganizationSize = ptr("1000+") }),
			field("organization_size", "invalid_format", "is not a known organization size")},
		{"unknown time zone", with(func(w *NewWorkspace) { w.Timezone = ptr("Mars/Olympus") }),
			field("timezone", "invalid_format", "is not a known time zone")},
		{"the host's zone", with(func(w *NewWorkspace) { w.Timezone = ptr("Local") }),
			field("timezone", "invalid_format", "is not a known time zone")},
		{"all at once", NewWorkspace{Name: "", Slug: "API", OrganizationSize: ptr(""), Timezone: ptr("")}, []shared.FieldError{
			{Field: "name", Code: "too_short", Message: "must not be empty"},
			{Field: "slug", Code: "invalid_format", Message: "may hold only lower-case letters, digits, - and _"},
			{Field: "organization_size", Code: "invalid_format", Message: "is not a known organization size"},
			{Field: "timezone", Code: "invalid_format", Message: "is not a known time zone"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckNewWorkspace(tt.w)
			var se *shared.Error
			if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || !slices.Equal(se.Fields, tt.want) {
				t.Errorf("CheckNewWorkspace(%+v) = %#v, want validation_failed with %v", tt.w, err, tt.want)
			}
		})
	}
}

func TestCheckSlug(t *testing.T) {
	for slug, want := range map[string]SlugReason{
		"acme":                   "",
		"a":                      "",
		strings.Repeat("x", 48):  "",
		"":                       SlugInvalid,
		strings.Repeat("x", 49):  SlugInvalid,
		"Acme":                   SlugInvalid,
		"acme/x":                 SlugInvalid,
		"sign-up":                SlugReserved,
		"readyz":                 SlugReserved,
		"static":                 SlugReserved,
		"login":                  "", // not a route: /login is a workspace's address (M3 design 3.10)
		"one":                    "", // Plane's product words are not reserved
		"workspace-invitations2": "",
	} {
		if got := CheckSlug(slug); got != want {
			t.Errorf("CheckSlug(%q) = %q, want %q", slug, got, want)
		}
	}
}
````

`server/internal/modules/workspace/domain/reserved_test.go`（新文件，55 行）：

````file server/internal/modules/workspace/domain/reserved_test.go
package domain

import (
	"slices"
	"testing"
)

// The list parses into three sections that do not overlap, every name is
// spelled as a slug, and the held names are exactly the four of M3 design
// 3.10 (9.1). Which names the app and the server sections hold is checked
// against the routes: the server's by bootstrap, the app's by the web app.
func TestTheReservedListIsWellFormed(t *testing.T) {
	r := Reserved()
	for name, section := range map[string][]string{"app": r.App, "server": r.Server, "reserved": r.Reserved} {
		if len(section) == 0 {
			t.Errorf("section [%s] is empty", name)
		}
	}
	seen := map[string]bool{}
	for _, slug := range r.All() {
		if seen[slug] {
			t.Errorf("%q is listed twice", slug)
		}
		seen[slug] = true
		if !slugPattern.MatchString(slug) || len(slug) > maxSlugLength {
			t.Errorf("%q is not spelled as a slug", slug)
		}
		if CheckSlug(slug) != SlugReserved {
			t.Errorf("CheckSlug(%q) = %q, want reserved", slug, CheckSlug(slug))
		}
	}
	if want := []string{"admin", "docs", "help", "static"}; !slices.Equal(r.Reserved, want) {
		t.Errorf("section [reserved] = %q, want %q", r.Reserved, want)
	}
}

func TestParseReserved(t *testing.T) {
	got, err := parseReserved("# a comment\n\n[server]\napi\n  healthz  \n[app]\nsign-up\n# another\n[reserved]\nhelp\n[app]\nicons\n")
	want := ReservedSlugs{App: []string{"sign-up", "icons"}, Server: []string{"api", "healthz"}, Reserved: []string{"help"}}
	if err != nil || !slices.Equal(got.App, want.App) || !slices.Equal(got.Server, want.Server) || !slices.Equal(got.Reserved, want.Reserved) {
		t.Errorf("parseReserved() = %+v, %v; want %+v", got, err, want)
	}
	if _, err := parseReserved("api\n[server]\n"); err == nil || err.Error() != `line 1: "api" is in no section` {
		t.Errorf("a name before every section: %v, want line 1 in no section", err)
	}
}

// Reserved returns a copy: a caller that changes it changes no answer.
func TestReservedReturnsACopy(t *testing.T) {
	r := Reserved()
	r.Server[0] = "not-reserved"
	if CheckSlug("not-reserved") != "" || CheckSlug("api") != SlugReserved {
		t.Error("changing Reserved()'s result changed the list")
	}
}
````

- [ ] **Step 3: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/domain/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/workspace/domain/actions.go server/internal/modules/workspace/domain/errors.go server/internal/modules/workspace/domain/reserved.go server/internal/modules/workspace/domain/reserved_slugs.txt server/internal/modules/workspace/domain/reserved_test.go server/internal/modules/workspace/domain/workspace.go server/internal/modules/workspace/domain/workspace_test.go
```
```bash
git commit -m "feat(M3/P1): the workspace module's domain

The rules of a new workspace, reported all at once; the reserved slugs in
three sections (M3 design 3.10); the action workspace.read; the module's
errors.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 领域的 6 个测试通过；名单文件解析通过。

---

### Task 8: `identity.Provide` 交出 `Accounts`

**Files:**
- Create: `server/internal/modules/identity/adapter/postgres/accounts.go`、`server/internal/modules/identity/adapter/postgres/accounts_test.go`、`server/internal/modules/identity/app/accounts.go`、`server/internal/modules/identity/provide.go`
- Modify: `server/internal/modules/identity/adapter/postgres/queries/users.sql`
- Generate: `server/internal/modules/identity/adapter/postgres/gen/users.sql.go`

**Interfaces:**
- Produces（spec 2.9，M3 设计 3.6 约定一、六，6.5、6.6）：
  - 查询 `ShareAccount`、`ShareAccountByEmail`（`SELECT id, email, is_active FROM users WHERE … FOR SHARE`）；
  - `app.AccountState{ID uuid.UUID; Email string; Active bool}`；存储的 `ShareAccount(ctx, id)`、`ShareAccountByEmail(ctx, email)` 返回 `(AccountState, found bool, error)`，在 ctx 带的事务里执行；
  - `identity.Accounts` 接口、`identity.Provided{Accounts}`、`identity.Provide(pool) Provided`：只用连接池，不建用例、不要签名密钥。
- 使用者：Task 11 的组合和 Task 14 的命令行组合（经 `bootstrap/ports.go` 转给 `workspace`）。

**Tests:**（`accounts_test.go`，真实数据库）
- `TestShareAccountReadsTheState`：两个账户（一个已停用），两种查法各读对状态；两种查法各一个不存在的账户 → `found = false`；地址按原样匹配（大写的查不到：规范化是调用方的事）。
- `TestTheShareLockBlocksDeactivationNotAnotherShare`：两种查法各一个子测试。持 `FOR SHARE` 时，另一个事务的 `LockForCredentials`（M2 停用的 `FOR NO KEY UPDATE`）在 `lock_timeout = 500ms` 下得到 `55P03`；第二个 `FOR SHARE` 不等。持锁的事务 10 秒内结束，测试失败而不挂住。

- [ ] **Step 1: 查询、存储、`Provide`**

`server/internal/modules/identity/adapter/postgres/queries/users.sql`（修改，1 处）：

````old server/internal/modules/identity/adapter/postgres/queries/users.sql
SET is_active = true, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

````

````new server/internal/modules/identity/adapter/postgres/queries/users.sql
SET is_active = true, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: ShareAccount :one
-- Accounts (M3 design 6.5): the first lock of a transaction that gives the account a workspace
-- membership (3.6 conventions 1 and 6). FOR SHARE conflicts with deactivation's FOR NO KEY UPDATE,
-- so the two run one after the other and is_active is read under the lock; two FOR SHARE do not
-- wait for each other.
SELECT id, email, is_active
FROM users
WHERE id = sqlc.arg(id)
FOR SHARE;

-- name: ShareAccountByEmail :one
-- ShareAccount for the server administrator's commands, which name the account by its address.
SELECT id, email, is_active
FROM users
WHERE email = sqlc.arg(email)
FOR SHARE;

````

`server/internal/modules/identity/app/accounts.go`（新文件，12 行）：

````file server/internal/modules/identity/app/accounts.go
package app

import "uuid"

// AccountState is an account's state as another module reads it through
// identity's Accounts, under the FOR SHARE lock the read takes (M3 design
// 6.5): the module that asked decides what a deactivated account may do.
type AccountState struct {
	ID     uuid.UUID
	Email  string
	Active bool
}
````

`server/internal/modules/identity/adapter/postgres/accounts.go`（新文件，38 行）：

````file server/internal/modules/identity/adapter/postgres/accounts.go
package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
)

// ShareAccount locks account id's row FOR SHARE until the transaction ends
// and returns its state; found is false when there is none (M3 design 6.5).
// Call it inside a transaction, as the transaction's first lock (M3 design
// 3.6 convention 1).
func (s *Store) ShareAccount(ctx context.Context, id uuid.UUID) (app.AccountState, bool, error) {
	row, err := s.queries(ctx).ShareAccount(ctx, id)
	return accountState(row.ID, row.Email, row.IsActive, err)
}

// ShareAccountByEmail is ShareAccount for the account with email, a
// normalized address.
func (s *Store) ShareAccountByEmail(ctx context.Context, email string) (app.AccountState, bool, error) {
	row, err := s.queries(ctx).ShareAccountByEmail(ctx, email)
	return accountState(row.ID, row.Email, row.IsActive, err)
}

func accountState(id uuid.UUID, email string, active bool, err error) (app.AccountState, bool, error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return app.AccountState{}, false, nil
	case err != nil:
		return app.AccountState{}, false, fmt.Errorf("lock the account row: %w", err)
	}
	return app.AccountState{ID: id, Email: email, Active: active}, true, nil
}
````

`server/internal/modules/identity/provide.go`（新文件，35 行）：

````file server/internal/modules/identity/provide.go
package identity

import (
	"context"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
)

// Accounts locks an account row FOR SHARE and returns the account's state;
// found is false when there is no such account (M3 design 6.5). It is the
// first lock of the caller's transaction and never a later one (M3 design
// 3.6 conventions 1 and 6): it conflicts with deactivation's lock of the
// row, so the account cannot be deactivated before the caller commits, and
// the state read is the one committed before the lock. An address is
// matched as given: the caller normalizes it.
type Accounts interface {
	ShareAccount(ctx context.Context, id uuid.UUID) (state app.AccountState, found bool, err error)
	ShareAccountByEmail(ctx context.Context, email string) (state app.AccountState, found bool, err error)
}

// Provided are the adapters identity offers the other modules. They depend
// on the pool alone, so bootstrap builds them before any module (M3 design
// 6.6, step 2).
type Provided struct {
	Accounts Accounts
}

// Provide builds identity's adapters for the other modules.
func Provide(pool *pgxpool.Pool) Provided {
	return Provided{Accounts: postgresadapter.New(pool)}
}
````

- [ ] **Step 2: 生成**

Run: `make gen-go`
Expected: 成功；只有 `users.sql.go` 改变：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `d6fb38d39ed45fb514bc236c4c669541473d2dd0adcf2233614897ffd5cc20a1` | 321 | `server/internal/modules/identity/adapter/postgres/gen/users.sql.go` |

Run: `shasum -a 256 server/internal/modules/identity/adapter/postgres/gen/users.sql.go`
Expected: 与上表相同。

- [ ] **Step 3: 测试**

`server/internal/modules/identity/adapter/postgres/accounts_test.go`（新文件，149 行）：

````file server/internal/modules/identity/adapter/postgres/accounts_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// ShareAccount and ShareAccountByEmail read the account's state: two
// accounts, one deactivated, and an unknown one of each kind (M3 design
// 6.5). The address is matched as given: the caller normalizes it.
func TestShareAccountReadsTheState(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)
	exec(t, pool, "UPDATE users SET is_active = false WHERE id = $1", bob.ID)
	tx := postgres.NewTxManager(pool, 2*time.Second)

	type answer struct {
		state app.AccountState
		found bool
	}
	var byID, byEmail []answer
	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		for _, id := range []uuid.UUID{alice.ID, bob.ID, uuid.NewV7()} {
			state, found, err := s.ShareAccount(ctx, id)
			if err != nil {
				return err
			}
			byID = append(byID, answer{state, found})
		}
		for _, email := range []string{"alice@corp.com", "bob@corp.com", "Alice@corp.com", "carol@corp.com"} {
			state, found, err := s.ShareAccountByEmail(ctx, email)
			if err != nil {
				return err
			}
			byEmail = append(byEmail, answer{state, found})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	aliceState := answer{app.AccountState{ID: alice.ID, Email: "alice@corp.com", Active: true}, true}
	bobState := answer{app.AccountState{ID: bob.ID, Email: "bob@corp.com", Active: false}, true}
	if want := []answer{aliceState, bobState, {}}; !slices.Equal(byID, want) {
		t.Errorf("ShareAccount() = %+v, want %+v", byID, want)
	}
	if want := []answer{aliceState, bobState, {}, {}}; !slices.Equal(byEmail, want) {
		t.Errorf("ShareAccountByEmail() = %+v, want %+v", byEmail, want)
	}
}

// FOR SHARE (M3 design 3.6 convention 6): while a transaction holds it,
// deactivation's lock of the row (FOR NO KEY UPDATE, M2's credential lock)
// waits, and a second FOR SHARE does not. lock_timeout turns a wait into a
// failure; the holding transaction ends within 10s, so the test fails, not
// hangs. Both reads take the lock.
func TestTheShareLockBlocksDeactivationNotAnotherShare(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	holders := map[string]func(ctx context.Context) error{
		"ShareAccount": func(ctx context.Context) error {
			_, _, err := s.ShareAccount(ctx, u.ID)
			return err
		},
		"ShareAccountByEmail": func(ctx context.Context) error {
			_, _, err := s.ShareAccountByEmail(ctx, "alice@corp.com")
			return err
		},
	}
	for name, hold := range holders {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			locked, release := make(chan struct{}), make(chan struct{})
			held := make(chan error, 1)
			go func() {
				held <- tx.WithinTx(ctx, func(ctx context.Context) error {
					if err := hold(ctx); err != nil {
						return err
					}
					close(locked)
					select {
					case <-release:
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				})
			}()
			select {
			case <-locked:
			case err := <-held:
				t.Fatalf("taking the lock: %v", err)
			case <-ctx.Done():
				t.Fatal("the lock was not taken within 10s")
			}
			defer func() {
				close(release)
				select {
				case err := <-held:
					if err != nil {
						t.Errorf("the transaction holding the lock: %v", err)
					}
				case <-ctx.Done():
					t.Error("the transaction holding the lock did not end within 10s")
				}
			}()
			withTimeout := func(fn func(ctx context.Context) error) error {
				return tx.WithinTx(context.Background(), func(ctx context.Context) error {
					if _, err := postgres.DB(ctx, pool).Exec(ctx, "SET LOCAL lock_timeout = '500ms'"); err != nil {
						return err
					}
					return fn(ctx)
				})
			}

			share := withTimeout(func(ctx context.Context) error {
				_, _, err := s.ShareAccount(ctx, u.ID)
				return err
			})
			deactivation := withTimeout(func(ctx context.Context) error {
				_, err := s.LockForCredentials(ctx, u.ID)
				return err
			})

			if share != nil {
				t.Errorf("a second FOR SHARE: %v, want no wait", share)
			}
			var pgErr *pgconn.PgError
			if !errors.As(deactivation, &pgErr) || pgErr.Code != "55P03" {
				t.Errorf("deactivation's lock: %v, want lock_not_available after waiting", deactivation)
			}
		})
	}
}
````

- [ ] **Step 4: 测试、lint**

Run: `go -C server test -count=1 -run 'TestShareAccountReadsTheState|TestTheShareLockBlocksDeactivationNotAnotherShare' ./internal/modules/identity/adapter/postgres/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 5: 提交**

```bash
git add server/internal/modules/identity/adapter/postgres/accounts.go server/internal/modules/identity/adapter/postgres/accounts_test.go server/internal/modules/identity/adapter/postgres/queries/users.sql server/internal/modules/identity/app/accounts.go server/internal/modules/identity/provide.go server/internal/modules/identity/adapter/postgres/gen/users.sql.go
```
```bash
git commit -m "feat(M3/P1): identity provides Accounts, the account row locked FOR SHARE

Provide(pool) hands other modules the account's state read under FOR SHARE
(M3 design 3.6 convention 6): deactivation's lock waits for it, another
share does not.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**Done when:** 两个测试在真实数据库上通过；`users.sql.go` 的 SHA-256 与表相同。

---

### Task 9: `workspace` 的端口和存储

**Files:**
- Create: `server/internal/modules/workspace/adapter/postgres/queries/members.sql`、`server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql`、`server/internal/modules/workspace/adapter/postgres/roles.go`、`server/internal/modules/workspace/adapter/postgres/store.go`、`server/internal/modules/workspace/adapter/postgres/store_test.go`、`server/internal/modules/workspace/adapter/postgres/workspaces.go`、`server/internal/modules/workspace/app/ports.go`
- Modify: `server/sqlc.yaml`
- Generate: `server/internal/modules/workspace/adapter/postgres/gen/db.go`、`server/internal/modules/workspace/adapter/postgres/gen/members.sql.go`、`server/internal/modules/workspace/adapter/postgres/gen/models.go`、`server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go`

**Interfaces:**
- Produces（spec 2.10，M3 设计 3.12、6.5）：
  - `app.ErrNotFound`、`app.Clock`、`app.AccountState`、`app.Accounts`（`ShareAccount`、`ShareAccountByEmail`）、`app.WorkspaceRow`、`app.MemberRow`、`app.WorkspaceCreator`（`CreateWorkspace`、`CreateMember`）、`app.WorkspaceLister`、`app.WorkspaceFinder`、`app.SlugChecker`；
  - 查询 `CreateWorkspace`、`ListWorkspaces`（`ORDER BY w.name, w.id`）、`WorkspaceBySlug`、`SlugTaken`、`CreateMember`、`ActiveRole`（spec 2.10 的条件）；
  - `postgresadapter.New(pool) *Store`：实现上面的仓储接口和 `ActiveRole(ctx, workspaceID, userID) (shared.Role, bool, error)`；`workspaces_slug_key` 的唯一冲突 → `domain.ErrSlugTaken`；
  - `server/sqlc.yaml` 的 `workspace` 条目（两个迁移、`queries`、`gen`）。
- 使用者：Task 10 的用例；Task 11 的 `workspace.Provide`（`ActiveRole`）；Task 15 的矩阵准备数据。

**Tests:**（`store_test.go`，真实数据库，账户用 SQL 写入 `identity` 的表作为夹具）
- `TestCreateWorkspaceStoresTheRow`：返回存下的值；审计列是用例的时间，UTC、微秒；`created_by_id = updated_by_id`。
- `TestCreateWorkspaceSlugTaken`：同一个 slug 第二次 → `ErrSlugTaken`；工作区删除之后 slug 可以再用。
- `TestCreateMemberStoresTheRow`。
- `TestListWorkspaces`：三个账户、七个工作区（夹具注释列出每个的状态）：alice 的列表不含别人的、已删除的、被移出的、成员行已删除的；同名的两个按 id；角色和有效成员数；bob 的列表不同。
- `TestWorkspaceBySlug`：找到的带有效成员数；已删除的、大写的、只是前缀的、不存在的 → `app.ErrNotFound`。
- `TestSlugTaken`：未删除的占用，已删除的、大写的、不存在的不占用。
- `TestActiveRole`：12 对（管理员、成员、访客、被移出、成员行已删除、工作区已删除、从来不是成员、不存在的工作区），各自的答案。
- `TestActiveRoleReadsInTheTransaction`：在事务里写的成员关系，`ActiveRole` 在同一个事务里看得到；回滚之后看不到。

- [ ] **Step 1: 端口、查询和 sqlc 条目**

`server/internal/modules/workspace/app/ports.go`（新文件，96 行）：

````file server/internal/modules/workspace/app/ports.go
// Package app holds the workspace module's use cases (M3 design 6.2) and the
// ports they need: small repository interfaces per use case, the clock, and
// the other modules' adapters as bootstrap converts them.
package app

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ErrNotFound is a repository's answer for a row that is not there.
var ErrNotFound = errors.New("workspace: not found")

// Clock gives the time the use cases write into the audit columns (M2
// design 3.13).
type Clock interface {
	Now() time.Time
}

// AccountState is an account's state as Accounts reads it under its lock:
// identity's value, converted in bootstrap/ports.go (M3 design 6.5).
type AccountState struct {
	ID     uuid.UUID
	Email  string
	Active bool
}

// Accounts locks an account row FOR SHARE and returns the account's state;
// found is false when there is no such account (M3 design 6.5). identity
// implements it (identity.Provide). Call it as the transaction's first lock
// and never later (M3 design 3.6 conventions 1 and 6): the lock keeps the
// account from being deactivated until the transaction ends, and the state is
// the one committed before it. An address is matched as given.
type Accounts interface {
	ShareAccount(ctx context.Context, id uuid.UUID) (state AccountState, found bool, err error)
	ShareAccountByEmail(ctx context.Context, email string) (state AccountState, found bool, err error)
}

// WorkspaceRow is a workspace to insert: checked values, its id, its
// creator and the time of the use case's clock.
type WorkspaceRow struct {
	ID               uuid.UUID
	Name             string
	Slug             string
	OrganizationSize *string
	Timezone         string
	CreatedBy        uuid.UUID
	Now              time.Time
}

// MemberRow is a workspace membership to insert, active.
type MemberRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	MemberID    uuid.UUID
	Role        shared.Role
	CreatedBy   uuid.UUID
	Now         time.Time
}

// WorkspaceCreator inserts a workspace and its first member.
type WorkspaceCreator interface {
	// CreateWorkspace inserts w and returns it as stored, without a role and
	// with no member counted; domain.ErrSlugTaken when an undeleted
	// workspace has its slug.
	CreateWorkspace(ctx context.Context, w WorkspaceRow) (domain.Workspace, error)
	// CreateMember inserts m.
	CreateMember(ctx context.Context, m MemberRow) error
}

// WorkspaceLister lists a user's workspaces.
type WorkspaceLister interface {
	// ListWorkspaces returns the undeleted workspaces of which userID is an
	// active member, with his role and the number of active members, by
	// name, then id (M3 design 3.12).
	ListWorkspaces(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error)
}

// WorkspaceFinder reads a workspace by its slug.
type WorkspaceFinder interface {
	// WorkspaceBySlug returns the undeleted workspace with slug and its
	// number of active members, without a role; ErrNotFound when there is
	// none.
	WorkspaceBySlug(ctx context.Context, slug string) (domain.Workspace, error)
}

// SlugChecker tells whether a slug is taken.
type SlugChecker interface {
	// SlugTaken reports whether an undeleted workspace has slug.
	SlugTaken(ctx context.Context, slug string) (bool, error)
}
````

`server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql`（新文件，27 行）：

````file server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql
-- name: CreateWorkspace :one
-- The audit columns come from the use case's clock (M2 design 3.13); RETURNING gives the values as stored.
INSERT INTO workspaces (id, name, slug, organization_size, timezone, created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(slug), sqlc.narg(organization_size), sqlc.arg(timezone),
        sqlc.arg(created_by), sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now))
RETURNING id, name, slug, organization_size, timezone, created_at, updated_at;

-- name: ListWorkspaces :many
-- listWorkspaces (M3 design 3.12): the undeleted workspaces of which the user is an active member, by name,
-- then id, with his role and the number of active members.
SELECT w.id, w.name, w.slug, w.organization_size, w.timezone, w.created_at, w.updated_at, m.role,
       (SELECT count(*) FROM workspace_members c
        WHERE c.workspace_id = w.id AND c.is_active AND c.deleted_at IS NULL) AS total_members
FROM workspaces w
JOIN workspace_members m ON m.workspace_id = w.id
WHERE m.member_id = sqlc.arg(user_id) AND m.is_active AND m.deleted_at IS NULL AND w.deleted_at IS NULL
ORDER BY w.name, w.id;

-- name: WorkspaceBySlug :one
SELECT w.id, w.name, w.slug, w.organization_size, w.timezone, w.created_at, w.updated_at,
       (SELECT count(*) FROM workspace_members c
        WHERE c.workspace_id = w.id AND c.is_active AND c.deleted_at IS NULL) AS total_members
FROM workspaces w
WHERE w.slug = sqlc.arg(slug) AND w.deleted_at IS NULL;

-- name: SlugTaken :one
SELECT EXISTS (SELECT 1 FROM workspaces WHERE slug = sqlc.arg(slug) AND deleted_at IS NULL);
````

`server/internal/modules/workspace/adapter/postgres/queries/members.sql`（新文件，13 行）：

````file server/internal/modules/workspace/adapter/postgres/queries/members.sql
-- name: CreateMember :exec
INSERT INTO workspace_members (id, workspace_id, member_id, role, created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(member_id), sqlc.arg(role),
        sqlc.arg(created_by), sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now));

-- name: ActiveRole :one
-- WorkspaceRoles (M3 design 6.5): the user's role when his membership is active, its row not deleted and
-- the workspace not deleted. The partial unique index holds at most one undeleted row per pair.
SELECT m.role
FROM workspace_members m
JOIN workspaces w ON w.id = m.workspace_id
WHERE m.workspace_id = sqlc.arg(workspace_id) AND m.member_id = sqlc.arg(user_id)
  AND m.is_active AND m.deleted_at IS NULL AND w.deleted_at IS NULL;
````

`server/sqlc.yaml`（修改，1 处）：

````old server/sqlc.yaml
            go_type: {import: time, type: Time, pointer: true}
````

````new server/sqlc.yaml
            go_type: {import: time, type: Time, pointer: true}
  - engine: postgresql
    schema:
      - migrations/sql/00006_workspace_workspaces.sql
      - migrations/sql/00007_workspace_workspace_members.sql
    queries: internal/modules/workspace/adapter/postgres/queries
    gen:
      go:
        package: gen
        out: internal/modules/workspace/adapter/postgres/gen
        sql_package: pgx/v5
        emit_pointers_for_null_types: true
        overrides:
          - db_type: uuid
            go_type: {import: uuid, type: UUID}
          - db_type: uuid
            nullable: true
            go_type: {import: uuid, type: UUID, pointer: true}
          - db_type: timestamptz
            go_type: {import: time, type: Time}
          - db_type: timestamptz
            nullable: true
            go_type: {import: time, type: Time, pointer: true}
````

- [ ] **Step 2: 生成**

Run: `make gen-go`
Expected: 成功；生成 `workspace` 的四个文件：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `1dfb2b6312c3c3db25a8cad84c031995c3b0546c73b3585a0282c98e11397676` | 32 | `server/internal/modules/workspace/adapter/postgres/gen/db.go` |
| `960e00c3d843e167da88676d1b663dfc65871bf0421452c5f8a62164ba8e53bd` | 62 | `server/internal/modules/workspace/adapter/postgres/gen/members.sql.go` |
| `9b86024c0357a2dea3a00590ea79d18f5a3ac78bf797376c7647b10ed8e99950` | 37 | `server/internal/modules/workspace/adapter/postgres/gen/models.go` |
| `34b262aecbe1df467d4833ac7f0898d4a85c9a5cb3e1eccbf6bc8edaa9f339f4` | 164 | `server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go` |

Run: `shasum -a 256 server/internal/modules/workspace/adapter/postgres/gen/db.go server/internal/modules/workspace/adapter/postgres/gen/members.sql.go server/internal/modules/workspace/adapter/postgres/gen/models.go server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go`
Expected: 与上表相同。

- [ ] **Step 3: 存储**

`server/internal/modules/workspace/adapter/postgres/store.go`（新文件，47 行）：

````file server/internal/modules/workspace/adapter/postgres/store.go
// Package postgresadapter is the workspace module's repository adapter: sqlc
// queries (queries/, generated into gen/) over the transaction that the
// context carries, or the pool. It also implements the reads that other
// modules make of workspaces through ports (M3 design 6.5): WorkspaceRoles.
package postgresadapter

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// Store implements the workspace repository ports of app.
type Store struct {
	pool *pgxpool.Pool
}

// New returns the store over pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// queries runs in the context's transaction when there is one.
func (s *Store) queries(ctx context.Context) *gen.Queries {
	return gen.New(postgres.DB(ctx, s.pool))
}

// uniqueViolation reports whether err broke the unique constraint name.
func uniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}

// notFound turns pgx.ErrNoRows into app.ErrNotFound.
func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return app.ErrNotFound
	}
	return err
}
````

`server/internal/modules/workspace/adapter/postgres/workspaces.go`（新文件，83 行）：

````file server/internal/modules/workspace/adapter/postgres/workspaces.go
package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateWorkspace inserts w and returns it as stored. A slug an undeleted
// workspace has is domain.ErrSlugTaken. The domain checked every value, so a
// CHECK violation is a bug: an internal error (500), not a domain error.
func (s *Store) CreateWorkspace(ctx context.Context, w app.WorkspaceRow) (domain.Workspace, error) {
	row, err := s.queries(ctx).CreateWorkspace(ctx, gen.CreateWorkspaceParams{
		ID: w.ID, Name: w.Name, Slug: w.Slug, OrganizationSize: w.OrganizationSize, Timezone: w.Timezone,
		CreatedBy: &w.CreatedBy, Now: w.Now,
	})
	switch {
	case uniqueViolation(err, "workspaces_slug_key"):
		return domain.Workspace{}, domain.ErrSlugTaken
	case err != nil:
		return domain.Workspace{}, fmt.Errorf("create workspace: %w", err)
	}
	return domain.Workspace{
		ID: row.ID, Name: row.Name, Slug: row.Slug, OrganizationSize: row.OrganizationSize, Timezone: row.Timezone,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

// CreateMember inserts m, active.
func (s *Store) CreateMember(ctx context.Context, m app.MemberRow) error {
	err := s.queries(ctx).CreateMember(ctx, gen.CreateMemberParams{
		ID: m.ID, WorkspaceID: m.WorkspaceID, MemberID: m.MemberID, Role: int16(m.Role), CreatedBy: &m.CreatedBy, Now: m.Now,
	})
	if err != nil {
		return fmt.Errorf("create workspace member: %w", err)
	}
	return nil
}

// ListWorkspaces returns the undeleted workspaces of which userID is an
// active member, with his role and the number of active members, by name,
// then id.
func (s *Store) ListWorkspaces(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error) {
	rows, err := s.queries(ctx).ListWorkspaces(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	out := make([]domain.Workspace, len(rows))
	for i, r := range rows {
		out[i] = domain.Workspace{
			ID: r.ID, Name: r.Name, Slug: r.Slug, OrganizationSize: r.OrganizationSize, Timezone: r.Timezone,
			Role: shared.Role(r.Role), TotalMembers: int(r.TotalMembers), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		}
	}
	return out, nil
}

// WorkspaceBySlug returns the undeleted workspace with slug and its number
// of active members, without a role; app.ErrNotFound when there is none.
func (s *Store) WorkspaceBySlug(ctx context.Context, slug string) (domain.Workspace, error) {
	r, err := s.queries(ctx).WorkspaceBySlug(ctx, slug)
	if err != nil {
		return domain.Workspace{}, notFound(err)
	}
	return domain.Workspace{
		ID: r.ID, Name: r.Name, Slug: r.Slug, OrganizationSize: r.OrganizationSize, Timezone: r.Timezone,
		TotalMembers: int(r.TotalMembers), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, nil
}

// SlugTaken reports whether an undeleted workspace has slug.
func (s *Store) SlugTaken(ctx context.Context, slug string) (bool, error) {
	taken, err := s.queries(ctx).SlugTaken(ctx, slug)
	if err != nil {
		return false, fmt.Errorf("check slug: %w", err)
	}
	return taken, nil
}
````

`server/internal/modules/workspace/adapter/postgres/roles.go`（新文件，28 行）：

````file server/internal/modules/workspace/adapter/postgres/roles.go
package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ActiveRole is access's WorkspaceRoles (M3 design 6.5): userID's role in
// workspaceID when the membership is active, its row not deleted and the
// workspace not deleted; ok is false otherwise. It reads in the transaction
// ctx carries.
func (s *Store) ActiveRole(ctx context.Context, workspaceID, userID uuid.UUID) (shared.Role, bool, error) {
	role, err := s.queries(ctx).ActiveRole(ctx, gen.ActiveRoleParams{WorkspaceID: workspaceID, UserID: userID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return 0, false, nil
	case err != nil:
		return 0, false, fmt.Errorf("read the workspace role: %w", err)
	}
	return shared.Role(role), true, nil
}
````

- [ ] **Step 4: 测试**

`server/internal/modules/workspace/adapter/postgres/store_test.go`（新文件，315 行）：

````file server/internal/modules/workspace/adapter/postgres/store_test.go
package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// now is the fixed clock's time: whole microseconds, as timestamptz stores
// them, so the audit columns read back equal to it (M2 design 3.13).
var now = clocktest.At(time.Date(2026, 9, 29, 10, 0, 0, 123456789, time.UTC)).Now()

func newStore(t *testing.T) (*postgresadapter.Store, *pgxpool.Pool) {
	t.Helper()
	pool, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: pgtest.NewDatabase(t), MaxConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return postgresadapter.New(pool), pool
}

func exec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

// newAccount inserts the users row that the workspace tables reference.
// identity owns the table: the test writes it directly, as a fixture.
func newAccount(t *testing.T, pool *pgxpool.Pool, email string) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, $2, 'x', 'x')", id, email)
	return id
}

// newWorkspace creates a workspace named name with slug, and admin as its
// admin, and returns it as stored.
func newWorkspace(t *testing.T, s *postgresadapter.Store, name, slug string, admin uuid.UUID) domain.Workspace {
	t.Helper()
	w, err := s.CreateWorkspace(context.Background(), app.WorkspaceRow{
		ID: uuid.NewV7(), Name: name, Slug: slug, Timezone: "UTC", CreatedBy: admin, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	join(t, s, w.ID, admin, shared.RoleAdmin)
	return w
}

// join makes user a member of workspace with role.
func join(t *testing.T, s *postgresadapter.Store, workspace, user uuid.UUID, role shared.Role) {
	t.Helper()
	m := app.MemberRow{ID: uuid.NewV7(), WorkspaceID: workspace, MemberID: user, Role: role, CreatedBy: user, Now: now}
	if err := s.CreateMember(context.Background(), m); err != nil {
		t.Fatal(err)
	}
}

// CreateWorkspace stores the row with the use case's values and answers it
// as stored: the time is the clock's, in UTC, to the microsecond.
func TestCreateWorkspaceStoresTheRow(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	size := "2-10"
	row := app.WorkspaceRow{ID: uuid.NewV7(), Name: "研发部", Slug: "rd", OrganizationSize: &size, Timezone: "Asia/Shanghai", CreatedBy: alice, Now: now}

	got, err := s.CreateWorkspace(context.Background(), row)

	want := domain.Workspace{ID: row.ID, Name: "研发部", Slug: "rd", OrganizationSize: &size, Timezone: "Asia/Shanghai", CreatedAt: now, UpdatedAt: now}
	if err != nil || !sameWorkspace(got, want) || got.CreatedAt.Location() != time.UTC {
		t.Fatalf("CreateWorkspace() = %+v, %v; want %+v", got, err, want)
	}
	var createdBy, updatedBy uuid.UUID
	var created, updated time.Time
	var deleted *time.Time
	if err := pool.QueryRow(context.Background(),
		"SELECT created_by_id, updated_by_id, created_at, updated_at, deleted_at FROM workspaces WHERE id = $1", row.ID).
		Scan(&createdBy, &updatedBy, &created, &updated, &deleted); err != nil {
		t.Fatal(err)
	}
	if createdBy != alice || updatedBy != alice || !created.Equal(now) || !updated.Equal(now) || deleted != nil {
		t.Errorf("row: created_by %s updated_by %s at %v, %v, deleted %v; want alice at %v", createdBy, updatedBy, created, updated, deleted, now)
	}
}

// sameWorkspace compares two workspaces, the organization size by value.
func sameWorkspace(a, b domain.Workspace) bool {
	sizeA, sizeB := a.OrganizationSize, b.OrganizationSize
	a.OrganizationSize, b.OrganizationSize = nil, nil
	return a == b && (sizeA == nil) == (sizeB == nil) && (sizeA == nil || *sizeA == *sizeB)
}

// A slug is unique among undeleted workspaces only: a deleted workspace's
// slug can be used again (M3 design 3.10).
func TestCreateWorkspaceSlugTaken(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	first := newWorkspace(t, s, "Acme", "acme", alice)

	_, err := s.CreateWorkspace(context.Background(), app.WorkspaceRow{ID: uuid.NewV7(), Name: "Other", Slug: "acme", Timezone: "UTC", CreatedBy: alice, Now: now})
	if !errors.Is(err, domain.ErrSlugTaken) {
		t.Fatalf("a taken slug: %v, want workspace.slug_taken", err)
	}
	exec(t, pool, "UPDATE workspaces SET deleted_at = $2 WHERE id = $1", first.ID, now)
	if _, err := s.CreateWorkspace(context.Background(), app.WorkspaceRow{ID: uuid.NewV7(), Name: "Acme 2", Slug: "acme", Timezone: "UTC", CreatedBy: alice, Now: now}); err != nil {
		t.Errorf("the slug of a deleted workspace: %v, want it free", err)
	}
}

func TestCreateMemberStoresTheRow(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	w := newWorkspace(t, s, "Acme", "acme", alice)
	id := uuid.NewV7()

	if err := s.CreateMember(context.Background(), app.MemberRow{ID: id, WorkspaceID: w.ID, MemberID: bob, Role: shared.RoleGuest, CreatedBy: alice, Now: now}); err != nil {
		t.Fatal(err)
	}

	var workspace, member, createdBy, updatedBy uuid.UUID
	var role int
	var active bool
	var created, updated time.Time
	if err := pool.QueryRow(context.Background(),
		"SELECT workspace_id, member_id, role, is_active, created_by_id, updated_by_id, created_at, updated_at FROM workspace_members WHERE id = $1", id).
		Scan(&workspace, &member, &role, &active, &createdBy, &updatedBy, &created, &updated); err != nil {
		t.Fatal(err)
	}
	if workspace != w.ID || member != bob || role != 5 || !active || createdBy != alice || updatedBy != alice || !created.Equal(now) || !updated.Equal(now) {
		t.Errorf("row = %s %s role %d active %v by %s/%s at %v/%v", workspace, member, role, active, createdBy, updatedBy, created, updated)
	}
}

// fixture is three accounts and seven workspaces, in every state for alice:
//   - beta (alice admin, bob member, carol removed), acme (bob admin, alice
//     guest), beta2 (bob removed, alice member; the same name as beta, so the
//     id breaks the tie);
//   - gone (alice admin, deleted), left (alice removed), dropped (alice's
//     row deleted), bobs (bob alone).
type fixture struct {
	s                                            *postgresadapter.Store
	pool                                         *pgxpool.Pool
	alice, bob, carol                            uuid.UUID
	beta, acme, beta2, gone, left, dropped, bobs domain.Workspace
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	s, pool := newStore(t)
	f := fixture{s: s, pool: pool}
	f.alice, f.bob, f.carol = newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	f.beta = newWorkspace(t, s, "Beta", "beta", f.alice)
	join(t, s, f.beta.ID, f.bob, shared.RoleMember)
	join(t, s, f.beta.ID, f.carol, shared.RoleMember)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2", f.beta.ID, f.carol)
	f.acme = newWorkspace(t, s, "Acme", "acme", f.bob)
	join(t, s, f.acme.ID, f.alice, shared.RoleGuest)
	f.beta2 = newWorkspace(t, s, "Beta", "beta-2", f.bob)
	join(t, s, f.beta2.ID, f.alice, shared.RoleMember)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2", f.beta2.ID, f.bob)
	f.gone = newWorkspace(t, s, "Gone", "gone", f.alice)
	exec(t, pool, "UPDATE workspaces SET deleted_at = $2 WHERE id = $1", f.gone.ID, now)
	f.left = newWorkspace(t, s, "Left", "left", f.bob)
	join(t, s, f.left.ID, f.alice, shared.RoleAdmin)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2", f.left.ID, f.alice)
	f.dropped = newWorkspace(t, s, "Dropped", "dropped", f.bob)
	join(t, s, f.dropped.ID, f.alice, shared.RoleMember)
	exec(t, pool, "UPDATE workspace_members SET deleted_at = $3 WHERE workspace_id = $1 AND member_id = $2", f.dropped.ID, f.alice, now)
	f.bobs = newWorkspace(t, s, "Bob's", "bobs", f.bob)
	return f
}

// ListWorkspaces lists a user's active memberships of undeleted workspaces,
// by name then id, with his role and the active members counted: not another
// account's workspace, not a deleted one, not one he was removed from, not a
// deleted membership (M3 design 3.12).
func TestListWorkspaces(t *testing.T) {
	f := newFixture(t)
	type item struct {
		id      uuid.UUID
		role    shared.Role
		members int
	}
	// beta and beta2 have the same name: the smaller id first.
	betas := []item{{f.beta.ID, shared.RoleAdmin, 2}, {f.beta2.ID, shared.RoleMember, 1}}
	if f.beta2.ID.Compare(f.beta.ID) < 0 {
		betas[0], betas[1] = betas[1], betas[0]
	}
	tests := []struct {
		name string
		user uuid.UUID
		want []item
	}{
		{"alice", f.alice, append([]item{{f.acme.ID, shared.RoleGuest, 2}}, betas...)},
		{"bob", f.bob, []item{{f.acme.ID, shared.RoleAdmin, 2}, {f.beta.ID, shared.RoleMember, 2}, {f.bobs.ID, shared.RoleAdmin, 1},
			{f.dropped.ID, shared.RoleAdmin, 1}, {f.left.ID, shared.RoleAdmin, 1}}},
		{"carol, removed", f.carol, nil},
		{"no account", uuid.NewV7(), nil},
	}
	for _, tt := range tests {
		got, err := f.s.ListWorkspaces(context.Background(), tt.user)
		if err != nil {
			t.Fatal(err)
		}
		var items []item
		for _, w := range got {
			items = append(items, item{w.ID, w.Role, w.TotalMembers})
		}
		if !slices.Equal(items, tt.want) {
			t.Errorf("%s: ListWorkspaces() = %v, want %v", tt.name, items, tt.want)
		}
	}
	// The columns are the stored ones.
	got, _ := f.s.ListWorkspaces(context.Background(), f.alice)
	if w := got[0]; w.Name != "Acme" || w.Slug != "acme" || w.Timezone != "UTC" || w.OrganizationSize != nil || !w.CreatedAt.Equal(now) || !w.UpdatedAt.Equal(now) {
		t.Errorf("ListWorkspaces()[0] = %+v, want acme as stored", w)
	}
}

// WorkspaceBySlug finds an undeleted workspace, with its active members
// counted.
func TestWorkspaceBySlug(t *testing.T) {
	f := newFixture(t)
	got, err := f.s.WorkspaceBySlug(context.Background(), "beta")
	want := f.beta
	want.TotalMembers = 2
	if err != nil || !sameWorkspace(got, want) {
		t.Errorf("WorkspaceBySlug(beta) = %+v, %v; want %+v", got, err, want)
	}
	for _, slug := range []string{"gone", "BETA", "bet", "nothing"} {
		if _, err := f.s.WorkspaceBySlug(context.Background(), slug); !errors.Is(err, app.ErrNotFound) {
			t.Errorf("WorkspaceBySlug(%q) = %v, want app.ErrNotFound", slug, err)
		}
	}
}

func TestSlugTaken(t *testing.T) {
	f := newFixture(t)
	for slug, want := range map[string]bool{"beta": true, "beta-2": true, "gone": false, "BETA": false, "nothing": false} {
		if got, err := f.s.SlugTaken(context.Background(), slug); err != nil || got != want {
			t.Errorf("SlugTaken(%q) = %v, %v; want %v", slug, got, err, want)
		}
	}
}

// ActiveRole is a caller's role in a workspace only while the membership
// counts: every pair of the fixture, each in its own state (M3 design 6.5).
func TestActiveRole(t *testing.T) {
	f := newFixture(t)
	tests := []struct {
		name            string
		workspace, user uuid.UUID
		want            shared.Role // 0: not active
	}{
		{"alice admin of beta", f.beta.ID, f.alice, shared.RoleAdmin},
		{"bob member of beta", f.beta.ID, f.bob, shared.RoleMember},
		{"carol removed from beta", f.beta.ID, f.carol, 0},
		{"alice guest of acme", f.acme.ID, f.alice, shared.RoleGuest},
		{"alice member of beta2", f.beta2.ID, f.alice, shared.RoleMember},
		{"bob removed from beta2", f.beta2.ID, f.bob, 0},
		{"alice admin of the deleted gone", f.gone.ID, f.alice, 0},
		{"alice removed from left", f.left.ID, f.alice, 0},
		{"alice's membership of dropped deleted", f.dropped.ID, f.alice, 0},
		{"alice never in bobs", f.bobs.ID, f.alice, 0},
		{"carol never in acme", f.acme.ID, f.carol, 0},
		{"no workspace", uuid.NewV7(), f.alice, 0},
	}
	for _, tt := range tests {
		role, ok, err := f.s.ActiveRole(context.Background(), tt.workspace, tt.user)
		if err != nil || ok != (tt.want != 0) || role != tt.want {
			t.Errorf("%s: ActiveRole() = %d, %v, %v; want %d", tt.name, role, ok, err, tt.want)
		}
	}
}

// ActiveRole reads in the transaction ctx carries: it sees the membership
// the transaction wrote before committing.
func TestActiveRoleReadsInTheTransaction(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	w := newWorkspace(t, s, "Acme", "acme", alice)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	var inside shared.Role
	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		if err := s.CreateMember(ctx, app.MemberRow{ID: uuid.NewV7(), WorkspaceID: w.ID, MemberID: bob, Role: shared.RoleMember, CreatedBy: alice, Now: now}); err != nil {
			return err
		}
		var err error
		inside, _, err = s.ActiveRole(ctx, w.ID, bob)
		return errors.Join(err, errors.New("roll back"))
	})
	if err == nil || inside != shared.RoleMember {
		t.Errorf("in the transaction: role %d, %v; want 15", inside, err)
	}
	if _, ok, err := s.ActiveRole(context.Background(), w.ID, bob); ok || err != nil {
		t.Errorf("after the rollback: %v, %v; want no membership", ok, err)
	}
}
````

- [ ] **Step 5: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/adapter/postgres/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`（`TestSQLCSchemaScope` 认出 `workspace` 的条目和查询）。

- [ ] **Step 6: 提交**

```bash
git add server/internal/modules/workspace/adapter/postgres/queries/members.sql server/internal/modules/workspace/adapter/postgres/queries/workspaces.sql server/internal/modules/workspace/adapter/postgres/roles.go server/internal/modules/workspace/adapter/postgres/store.go server/internal/modules/workspace/adapter/postgres/store_test.go server/internal/modules/workspace/adapter/postgres/workspaces.go server/internal/modules/workspace/app/ports.go server/sqlc.yaml server/internal/modules/workspace/adapter/postgres/gen/db.go server/internal/modules/workspace/adapter/postgres/gen/members.sql.go server/internal/modules/workspace/adapter/postgres/gen/models.go server/internal/modules/workspace/adapter/postgres/gen/workspaces.sql.go
```
```bash
git commit -m "feat(M3/P1): the workspace module's ports and store

The queries of workspaces and members, the caller's list and the active
role; a taken slug is the store's answer, a deleted workspace frees it.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**Done when:** 8 个存储测试在真实数据库上通过，每个 `WHERE` 都有第二个账户或工作区看得见；四个生成物的 SHA-256 与表相同。

---

### Task 10: 四个用例

**Files:**
- Create: `server/internal/modules/workspace/app/check_slug.go`、`server/internal/modules/workspace/app/create_workspace.go`、`server/internal/modules/workspace/app/create_workspace_test.go`、`server/internal/modules/workspace/app/fakes_test.go`、`server/internal/modules/workspace/app/get_workspace.go`、`server/internal/modules/workspace/app/list_workspaces.go`、`server/internal/modules/workspace/app/read_test.go`

**Interfaces:**
- Produces（spec 2.11，M3 设计 3.4、3.6、3.11、8.2）：
  - `app.CreateWorkspaceDeps{Accounts, Workspaces, Tx, Clock, Logger, Enabled}`、`app.NewCreateWorkspace(d)`；`Execute(ctx, w)`（开关、`RequireActor`、校验、一个事务：`ShareAccount` → `CreateWorkspace` → `CreateMember`）；`ExecuteForAdmin(ctx, email, w)`（不看开关；`shared.NormalizeEmail`；`ShareAccountByEmail`）；成功后记 INFO `workspace created`（`workspace_id`、`user_id`、`by`）；
  - `app.NewListWorkspaces(lister)`、`Execute(ctx)`；
  - `app.NewGetWorkspace(finder, authorizer)`、`Execute(ctx, slug)`：`Authorize(ActionRead, Target{WorkspaceID})`，`ErrNotVisible` → `workspace.not_found`，角色取自 `Grant`；不开事务；
  - `app.NewCheckSlug(checker)`、`Execute(ctx, slug) (domain.SlugReason, error)`：不合格就不查库。
- 使用者：Task 11 的 handler 和组合；Task 14 的 `NewAdmin`。

**Tests:**（`fakes_test.go` 的手写假实现：假事务在 ctx 里做标记；存储和锁的每次调用记下参数和是否在事务里；`fakeAuthorizer` 按调用者和工作区回答）
- `create_workspace_test.go`：
  - `TestExecuteCreatesTheWorkspaceWithTheCallerAsAdmin`：两个调用者；调用顺序 `ShareAccount(调用者)` → `CreateWorkspace` → `CreateMember(20)`，都在同一个事务里（假锁在事务外被调用时记下 `outside tx`）；时间是时钟的；结果角色 20、人数 1；日志恰好一行 `msg="workspace created" workspace_id=… user_id=… by=api`。
  - `TestExecuteStoresTheTimeZoneGiven`。
  - `TestExecuteWhileCreationIsDisabled`：开关关闭时答 `workspace.creation_disabled`，值不合规也一样，什么都不调。
  - `TestExecuteRefusesInvalidValuesBeforeTheTransaction`：422，没有锁、没有写入。
  - `TestExecuteRefusesAnAccountNoLongerActive`：锁下读到已停用、账户不存在：401 `unauthorized`，没有写入。
  - `TestExecuteSlugTaken`：存储的回答原样返回，没有成员写入。
  - `TestExecuteReturnsTheLocksError`：写入之前。
  - `TestExecuteWithoutACaller`：401，什么都不调。
  - `TestExecuteForAdminCreatesForTheAccountOfTheAddress`：开关开和关都建；地址 `"  Bob@Corp.COM "` 规范化之后交给锁；日志一行，`by=cli`。
  - `TestExecuteForAdminRefuses`：没有账户、锁下已停用、保留的 slug：各自的错误，没有写入，没有日志。
  - 拒绝的测试（`TestExecuteRefusesAnAccountNoLongerActive`、`TestExecuteSlugTaken`）也断言没有日志。
- `read_test.go`：`TestListWorkspacesIsTheCallersList`（两个调用者，两个列表）；`TestGetWorkspaceDecidesOnTheWorkspaceFound`（两个调用者、两个工作区；`Authorize` 的参数；不开事务；角色来自 `Grant`）；`TestGetWorkspaceNotFound`（看不到和不存在是同一个 `workspace.not_found`；别的拒绝不被换掉）；`TestCheckSlug`（只查可以用的 slug）。

- [ ] **Step 1: 用例**

`server/internal/modules/workspace/app/create_workspace.go`（新文件，130 行）：

````file server/internal/modules/workspace/app/create_workspace.go
package app

import (
	"context"
	"log/slog"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateWorkspaceDeps are CreateWorkspace's collaborators.
type CreateWorkspaceDeps struct {
	Accounts   Accounts
	Workspaces WorkspaceCreator
	Tx         shared.TxManager
	Clock      Clock
	Logger     *slog.Logger
	// Enabled is workspace.creation_enabled: whether callers of the API may
	// create workspaces. The server's administrator always may (M3 design
	// 3.11).
	Enabled bool
}

// CreateWorkspace creates a workspace with its first admin (M3 design 3.11):
// the caller of POST /api/v0/workspaces, or the account the server's
// administrator names to `nerve workspaces create`. One use case, two
// entries, the same rules, uniqueness and admin membership.
type CreateWorkspace struct {
	d CreateWorkspaceDeps
}

// NewCreateWorkspace returns the use case.
func NewCreateWorkspace(d CreateWorkspaceDeps) *CreateWorkspace {
	return &CreateWorkspace{d: d}
}

// Execute creates w with the caller as its admin: workspace.creation_disabled
// while creation is off, before anything else (Plane views/workspace/
// base.py:83-96); 422 validation_failed; 401 unauthorized when the account,
// read under its lock, is no longer active (M3 design 3.6 convention 6);
// workspace.slug_taken.
func (u *CreateWorkspace) Execute(ctx context.Context, w domain.NewWorkspace) (domain.Workspace, error) {
	if !u.d.Enabled {
		return domain.Workspace{}, domain.ErrCreationDisabled
	}
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Workspace{}, err
	}
	return u.create(ctx, w, byAPI, func(ctx context.Context) (AccountState, error) {
		state, found, err := u.d.Accounts.ShareAccount(ctx, actor.UserID)
		switch {
		case err != nil:
			return AccountState{}, err
		case !found || !state.Active:
			return AccountState{}, shared.Unauthenticated()
		}
		return state, nil
	})
}

// ExecuteForAdmin creates w with the account of email as its admin, for the
// server's administrator: workspace.creation_enabled does not apply. The
// address is normalized as registration does it; no such account is
// workspace.account_not_found, a deactivated one, read under its lock,
// workspace.account_deactivated.
func (u *CreateWorkspace) ExecuteForAdmin(ctx context.Context, email string, w domain.NewWorkspace) (domain.Workspace, error) {
	email = shared.NormalizeEmail(email)
	return u.create(ctx, w, byCLI, func(ctx context.Context) (AccountState, error) {
		state, found, err := u.d.Accounts.ShareAccountByEmail(ctx, email)
		switch {
		case err != nil:
			return AccountState{}, err
		case !found:
			return AccountState{}, domain.ErrAccountNotFound
		case !state.Active:
			return AccountState{}, domain.ErrAccountDeactivated
		}
		return state, nil
	})
}

// The values of "by" in the logs: who asked.
const (
	byAPI = "api"
	byCLI = "cli"
)

// create checks w, then writes in one transaction in the order of M3 design
// 3.6: the admin's account row first, locked FOR SHARE by lock, which also
// decides on the account's state as read under the lock; then the workspace
// and its admin membership. Creating a workspace enqueues nothing: no demo
// data (M3 design 3.11).
func (u *CreateWorkspace) create(ctx context.Context, w domain.NewWorkspace, by string,
	lock func(ctx context.Context) (AccountState, error)) (domain.Workspace, error) {
	if err := domain.CheckNewWorkspace(w); err != nil {
		return domain.Workspace{}, err
	}
	zone := domain.DefaultTimezone
	if w.Timezone != nil {
		zone = *w.Timezone
	}
	now := u.d.Clock.Now()
	var admin AccountState
	var created domain.Workspace
	err := u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if admin, err = lock(ctx); err != nil {
			return err
		}
		created, err = u.d.Workspaces.CreateWorkspace(ctx, WorkspaceRow{
			ID: uuid.NewV7(), Name: w.Name, Slug: w.Slug, OrganizationSize: w.OrganizationSize, Timezone: zone,
			CreatedBy: admin.ID, Now: now,
		})
		if err != nil {
			return err
		}
		return u.d.Workspaces.CreateMember(ctx, MemberRow{
			ID: uuid.NewV7(), WorkspaceID: created.ID, MemberID: admin.ID, Role: shared.RoleAdmin, CreatedBy: admin.ID, Now: now,
		})
	})
	if err != nil {
		return domain.Workspace{}, err
	}
	created.Role, created.TotalMembers = shared.RoleAdmin, 1
	u.d.Logger.InfoContext(ctx, "workspace created", slog.String("workspace_id", created.ID.String()),
		slog.String("user_id", admin.ID.String()), slog.String("by", by))
	return created, nil
}
````

`server/internal/modules/workspace/app/list_workspaces.go`（新文件，31 行）：

````file server/internal/modules/workspace/app/list_workspaces.go
package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListWorkspaces lists the caller's workspaces: GET /api/v0/workspaces, the
// whole collection at once (M3 design 3.12). It is an account-level
// operation: any valid credential may call it, and the list holds only the
// workspaces of which the caller is an active member, so it asks no
// Authorizer (M3 design 6.4).
type ListWorkspaces struct {
	workspaces WorkspaceLister
}

// NewListWorkspaces returns the use case.
func NewListWorkspaces(workspaces WorkspaceLister) *ListWorkspaces {
	return &ListWorkspaces{workspaces: workspaces}
}

// Execute returns the caller's workspaces, by name then id.
func (u *ListWorkspaces) Execute(ctx context.Context) ([]domain.Workspace, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	return u.workspaces.ListWorkspaces(ctx, actor.UserID)
}
````

`server/internal/modules/workspace/app/get_workspace.go`（新文件，47 行）：

````file server/internal/modules/workspace/app/get_workspace.go
package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetWorkspace reads a workspace by its slug: GET /api/v0/workspaces/{slug}.
type GetWorkspace struct {
	workspaces WorkspaceFinder
	auth       shared.Authorizer
}

// NewGetWorkspace returns the use case.
func NewGetWorkspace(workspaces WorkspaceFinder, auth shared.Authorizer) *GetWorkspace {
	return &GetWorkspace{workspaces: workspaces, auth: auth}
}

// Execute returns the workspace with the caller's role. A read opens no
// transaction and decides directly (M3 design 3.4). A workspace that does
// not exist, is deleted, or that the caller cannot see is the same
// workspace.not_found (M3 design 8.2).
func (u *GetWorkspace) Execute(ctx context.Context, slug string) (domain.Workspace, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Workspace{}, err
	}
	w, err := u.workspaces.WorkspaceBySlug(ctx, slug)
	switch {
	case errors.Is(err, ErrNotFound):
		return domain.Workspace{}, domain.ErrNotFound
	case err != nil:
		return domain.Workspace{}, err
	}
	grant, err := u.auth.Authorize(ctx, actor, domain.ActionRead, shared.Target{WorkspaceID: w.ID})
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return domain.Workspace{}, domain.ErrNotFound
	case err != nil:
		return domain.Workspace{}, err
	}
	w.Role = grant.WorkspaceRole
	return w, nil
}
````

`server/internal/modules/workspace/app/check_slug.go`（新文件，36 行）：

````file server/internal/modules/workspace/app/check_slug.go
package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// CheckSlug answers GET /api/v0/workspace-slugs/{slug}: whether a slug can
// name a new workspace, and why not (M3 design 3.10, 5.1). It is an
// account-level operation, asking no Authorizer (M3 design 6.4); it tells
// only whether the name can be used, as Plane's check does (M3 design 8.2).
type CheckSlug struct {
	slugs SlugChecker
}

// NewCheckSlug returns the use case.
func NewCheckSlug(slugs SlugChecker) *CheckSlug {
	return &CheckSlug{slugs: slugs}
}

// Execute returns why slug cannot be used, invalid, reserved or taken, or ""
// when it can. Only a slug that could be used is looked up.
func (u *CheckSlug) Execute(ctx context.Context, slug string) (domain.SlugReason, error) {
	if reason := domain.CheckSlug(slug); reason != "" {
		return reason, nil
	}
	taken, err := u.slugs.SlugTaken(ctx, slug)
	if err != nil {
		return "", err
	}
	if taken {
		return domain.SlugTaken, nil
	}
	return "", nil
}
````

- [ ] **Step 2: 假实现和测试**

`server/internal/modules/workspace/app/fakes_test.go`（新文件，139 行）：

````file server/internal/modules/workspace/app/fakes_test.go
package app_test

import (
	"context"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var now = clocktest.At(time.Date(2026, 9, 29, 10, 0, 0, 123456000, time.UTC)).Now()

// fakeTx runs fn in a context marked as inside the transaction; the fakes
// record whether each call happened there.
type fakeTx struct{ calls int }

type inTxKey struct{}

func (f *fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	f.calls++
	return fn(context.WithValue(ctx, inTxKey{}, true))
}

// callLog records the calls of the fakes that share it, in order, each with
// its arguments, and " outside tx" when it ran outside a transaction.
type callLog struct{ calls []string }

func (l *callLog) add(ctx context.Context, format string, args ...any) {
	call := fmt.Sprintf(format, args...)
	if ctx.Value(inTxKey{}) != true {
		call += " outside tx"
	}
	l.calls = append(l.calls, call)
}

// fakeAccounts answers the state of the accounts it holds, by id and by
// address, and logs each lock.
type fakeAccounts struct {
	log      *callLog
	accounts []app.AccountState
	err      error
}

func (f *fakeAccounts) ShareAccount(ctx context.Context, id uuid.UUID) (app.AccountState, bool, error) {
	f.log.add(ctx, "ShareAccount %s", id)
	i := slices.IndexFunc(f.accounts, func(a app.AccountState) bool { return a.ID == id })
	if f.err != nil || i < 0 {
		return app.AccountState{}, false, f.err
	}
	return f.accounts[i], true, nil
}

func (f *fakeAccounts) ShareAccountByEmail(ctx context.Context, email string) (app.AccountState, bool, error) {
	f.log.add(ctx, "ShareAccountByEmail %s", email)
	i := slices.IndexFunc(f.accounts, func(a app.AccountState) bool { return a.Email == email })
	if f.err != nil || i < 0 {
		return app.AccountState{}, false, f.err
	}
	return f.accounts[i], true, nil
}

// fakeWorkspaces is the repositories: it logs every call with its
// arguments, answers from the workspaces it holds, and fails a call with
// the error set for it.
type fakeWorkspaces struct {
	log        *callLog
	workspaces []domain.Workspace // by slug for WorkspaceBySlug and SlugTaken
	lists      map[uuid.UUID][]domain.Workspace
	createErr  error
	listErr    error
	members    []app.MemberRow
}

func (f *fakeWorkspaces) CreateWorkspace(ctx context.Context, w app.WorkspaceRow) (domain.Workspace, error) {
	size := "<nil>"
	if w.OrganizationSize != nil {
		size = *w.OrganizationSize
	}
	f.log.add(ctx, "CreateWorkspace %s %q %s %s %s by %s at %s", w.ID, w.Name, w.Slug, size, w.Timezone, w.CreatedBy, w.Now.Format(time.RFC3339Nano))
	if f.createErr != nil {
		return domain.Workspace{}, f.createErr
	}
	// As stored: the database's clock has no nanoseconds either.
	return domain.Workspace{ID: w.ID, Name: w.Name, Slug: w.Slug, OrganizationSize: w.OrganizationSize, Timezone: w.Timezone,
		CreatedAt: w.Now, UpdatedAt: w.Now}, nil
}

func (f *fakeWorkspaces) CreateMember(ctx context.Context, m app.MemberRow) error {
	f.log.add(ctx, "CreateMember %s in %s as %d by %s at %s", m.MemberID, m.WorkspaceID, m.Role, m.CreatedBy, m.Now.Format(time.RFC3339Nano))
	f.members = append(f.members, m)
	return nil
}

func (f *fakeWorkspaces) ListWorkspaces(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error) {
	f.log.add(ctx, "ListWorkspaces %s", userID)
	return f.lists[userID], f.listErr
}

func (f *fakeWorkspaces) WorkspaceBySlug(ctx context.Context, slug string) (domain.Workspace, error) {
	f.log.add(ctx, "WorkspaceBySlug %s", slug)
	i := slices.IndexFunc(f.workspaces, func(w domain.Workspace) bool { return w.Slug == slug })
	if i < 0 {
		return domain.Workspace{}, app.ErrNotFound
	}
	return f.workspaces[i], nil
}

func (f *fakeWorkspaces) SlugTaken(ctx context.Context, slug string) (bool, error) {
	f.log.add(ctx, "SlugTaken %s", slug)
	return slices.ContainsFunc(f.workspaces, func(w domain.Workspace) bool { return w.Slug == slug }), nil
}

// grantKey is one (user, workspace) pair of fakeAuthorizer.
type grantKey struct{ user, workspace uuid.UUID }

// fakeAuthorizer answers each (user, workspace) pair its grant, or its
// error, and ErrNotVisible for any other; it logs every call.
type fakeAuthorizer struct {
	log    *callLog
	grants map[grantKey]shared.Grant
	errs   map[grantKey]error
}

func (f *fakeAuthorizer) Authorize(ctx context.Context, actor shared.Actor, action shared.Action, t shared.Target) (shared.Grant, error) {
	f.log.add(ctx, "Authorize %s %s on %s/%s", actor.UserID, action, t.WorkspaceID, t.ProjectID)
	key := grantKey{actor.UserID, t.WorkspaceID}
	if err := f.errs[key]; err != nil {
		return shared.Grant{}, err
	}
	if g, ok := f.grants[key]; ok {
		return g, nil
	}
	return shared.Grant{}, shared.ErrNotVisible
}
````

`server/internal/modules/workspace/app/create_workspace_test.go`（新文件，245 行）：

````file server/internal/modules/workspace/app/create_workspace_test.go
package app_test

import (
	"context"
	"errors"
	"log/slog"
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

var (
	alice = app.AccountState{ID: uuid.NewV7(), Email: "alice@corp.com", Active: true}
	bob   = app.AccountState{ID: uuid.NewV7(), Email: "bob@corp.com", Active: true}
	carol = app.AccountState{ID: uuid.NewV7(), Email: "carol@corp.com", Active: false}
)

type createFixture struct {
	log        *callLog
	tx         *fakeTx
	accounts   *fakeAccounts
	workspaces *fakeWorkspaces
	logs       *strings.Builder // the use case's log, as text
}

func newCreate(enabled bool) (*app.CreateWorkspace, *createFixture) {
	log := &callLog{}
	f := &createFixture{log: log, tx: &fakeTx{}, accounts: &fakeAccounts{log: log, accounts: []app.AccountState{alice, bob, carol}},
		workspaces: &fakeWorkspaces{log: log}, logs: &strings.Builder{}}
	return app.NewCreateWorkspace(app.CreateWorkspaceDeps{
		Accounts: f.accounts, Workspaces: f.workspaces, Tx: f.tx, Clock: clocktest.At(now),
		Logger: slog.New(slog.NewTextHandler(f.logs, nil)), Enabled: enabled,
	}), f
}

// createdLog is the one line a creation logs: the workspace, its admin, and
// whether the API or the command line created it.
func createdLog(workspace uuid.UUID, admin app.AccountState, by string) string {
	return `level=INFO msg="workspace created" workspace_id=` + workspace.String() + " user_id=" + admin.ID.String() + " by=" + by + "\n"
}

// logged is the log without each line's time.
func logged(f *createFixture) string {
	var out strings.Builder
	for line := range strings.Lines(f.logs.String()) {
		_, rest, _ := strings.Cut(line, " ")
		out.WriteString(rest)
	}
	return out.String()
}

func as(user app.AccountState) context.Context {
	return shared.WithActor(context.Background(), shared.Actor{UserID: user.ID})
}

// The caller's account row is locked first, in the transaction, then the
// workspace and the caller's admin membership are written, all with the
// clock's time (M3 design 3.6, its global order and convention 1): for two
// callers, each with his own account. The answer is the workspace as stored,
// the caller its admin and only member; one line is logged.
func TestExecuteCreatesTheWorkspaceWithTheCallerAsAdmin(t *testing.T) {
	for _, user := range []app.AccountState{alice, bob} {
		uc, f := newCreate(true)
		size := "11-50"

		got, err := uc.Execute(as(user), domain.NewWorkspace{Name: "Acme", Slug: "acme", OrganizationSize: &size})

		if err != nil {
			t.Fatalf("%s: Execute() = %v", user.Email, err)
		}
		at := now.Format(time.RFC3339Nano)
		want := []string{
			"ShareAccount " + user.ID.String(),
			"CreateWorkspace " + got.ID.String() + ` "Acme" acme 11-50 UTC by ` + user.ID.String() + " at " + at,
			"CreateMember " + user.ID.String() + " in " + got.ID.String() + " as 20 by " + user.ID.String() + " at " + at,
		}
		if !slices.Equal(f.log.calls, want) || f.tx.calls != 1 {
			t.Errorf("%s: calls = %q in %d transactions, want %q in one", user.Email, f.log.calls, f.tx.calls, want)
		}
		if got.ID == uuid.Nil() || got.Name != "Acme" || got.Slug != "acme" || *got.OrganizationSize != "11-50" || got.Timezone != "UTC" ||
			got.Role != shared.RoleAdmin || got.TotalMembers != 1 || !got.CreatedAt.Equal(now) || !got.UpdatedAt.Equal(now) {
			t.Errorf("%s: Execute() = %+v, want acme as stored, with the caller its admin and only member", user.Email, got)
		}
		if m := f.workspaces.members; len(m) != 1 || m[0].ID == uuid.Nil() || m[0].ID == got.ID {
			t.Errorf("%s: members = %+v, want one with its own id", user.Email, m)
		}
		if line, want := logged(f), createdLog(got.ID, user, "api"); line != want {
			t.Errorf("%s: log = %q, want %q", user.Email, line, want)
		}
	}
}

// A time zone given is stored as given.
func TestExecuteStoresTheTimeZoneGiven(t *testing.T) {
	uc, f := newCreate(true)
	zone := "Asia/Shanghai"
	got, err := uc.Execute(as(alice), domain.NewWorkspace{Name: "研发部", Slug: "rd", Timezone: &zone})
	if err != nil || got.Timezone != zone || got.OrganizationSize != nil {
		t.Fatalf("Execute() = %+v, %v; want the time zone %s and no size", got, err, zone)
	}
	if call := f.log.calls[1]; call != "CreateWorkspace "+got.ID.String()+` "研发部" rd <nil> Asia/Shanghai by `+alice.ID.String()+" at "+now.Format(time.RFC3339Nano) {
		t.Errorf("CreateWorkspace call = %q", call)
	}
}

// While creation is off, the API answers workspace.creation_disabled before
// it looks at anything, the values included (Plane views/workspace/
// base.py:83-96).
func TestExecuteWhileCreationIsDisabled(t *testing.T) {
	uc, f := newCreate(false)
	for _, w := range []domain.NewWorkspace{{Name: "Acme", Slug: "acme"}, {Name: "", Slug: "API"}} {
		if _, err := uc.Execute(as(alice), w); !errors.Is(err, domain.ErrCreationDisabled) {
			t.Errorf("Execute(%+v) = %v, want workspace.creation_disabled", w, err)
		}
	}
	if len(f.log.calls) != 0 || f.tx.calls != 0 {
		t.Errorf("calls = %q in %d transactions, want none", f.log.calls, f.tx.calls)
	}
}

// Invalid values are 422 before any lock or write.
func TestExecuteRefusesInvalidValuesBeforeTheTransaction(t *testing.T) {
	uc, f := newCreate(true)
	_, err := uc.Execute(as(alice), domain.NewWorkspace{Name: "Acme", Slug: "api"})
	var se *shared.Error
	if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || se.Fields[0].Field != "slug" {
		t.Errorf("Execute() = %v, want validation_failed on slug", err)
	}
	if len(f.log.calls) != 0 || f.tx.calls != 0 {
		t.Errorf("calls = %q in %d transactions, want none", f.log.calls, f.tx.calls)
	}
}

// An account that is deactivated, as read under the lock, or gone, gets 401
// and writes nothing (M3 design 3.6 convention 6).
func TestExecuteRefusesAnAccountNoLongerActive(t *testing.T) {
	gone := app.AccountState{ID: uuid.NewV7()}
	for _, user := range []app.AccountState{carol, gone} {
		uc, f := newCreate(true)
		_, err := uc.Execute(as(user), domain.NewWorkspace{Name: "Acme", Slug: "acme"})
		if !errors.Is(err, shared.Unauthenticated()) {
			t.Errorf("%s: Execute() = %v, want 401 unauthorized", user.ID, err)
		}
		if want := []string{"ShareAccount " + user.ID.String()}; !slices.Equal(f.log.calls, want) {
			t.Errorf("%s: calls = %q, want %q", user.ID, f.log.calls, want)
		}
		if f.logs.Len() != 0 {
			t.Errorf("%s: log = %q, want nothing", user.ID, f.logs)
		}
	}
}

// A taken slug is the store's answer, and nothing more is written.
func TestExecuteSlugTaken(t *testing.T) {
	uc, f := newCreate(true)
	f.workspaces.createErr = domain.ErrSlugTaken
	if _, err := uc.Execute(as(alice), domain.NewWorkspace{Name: "Acme", Slug: "acme"}); !errors.Is(err, domain.ErrSlugTaken) {
		t.Errorf("Execute() = %v, want workspace.slug_taken", err)
	}
	if len(f.log.calls) != 2 || len(f.workspaces.members) != 0 || f.logs.Len() != 0 {
		t.Errorf("calls = %q, log %q; want the lock and the insert only, and no log", f.log.calls, f.logs)
	}
}

// The lock's failure is the use case's, before any write.
func TestExecuteReturnsTheLocksError(t *testing.T) {
	uc, f := newCreate(true)
	failure := errors.New("connection reset")
	f.accounts.err = failure
	if _, err := uc.Execute(as(alice), domain.NewWorkspace{Name: "Acme", Slug: "acme"}); !errors.Is(err, failure) {
		t.Errorf("Execute() = %v, want %v", err, failure)
	}
	if len(f.log.calls) != 1 {
		t.Errorf("calls = %q, want the lock only", f.log.calls)
	}
}

func TestExecuteWithoutACaller(t *testing.T) {
	uc, f := newCreate(true)
	if _, err := uc.Execute(context.Background(), domain.NewWorkspace{Name: "Acme", Slug: "acme"}); err == nil {
		t.Error("Execute() without an actor = nil, want an error")
	}
	if len(f.log.calls) != 0 {
		t.Errorf("calls = %q, want none", f.log.calls)
	}
}

// The server's administrator creates a workspace for the account of an
// address, normalized, whatever workspace.creation_enabled says (M3 design
// 3.11).
func TestExecuteForAdminCreatesForTheAccountOfTheAddress(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		uc, f := newCreate(enabled)

		got, err := uc.ExecuteForAdmin(context.Background(), "  Bob@Corp.COM ", domain.NewWorkspace{Name: "Acme", Slug: "acme"})

		if err != nil || got.Role != shared.RoleAdmin || got.TotalMembers != 1 {
			t.Fatalf("enabled %v: ExecuteForAdmin() = %+v, %v", enabled, got, err)
		}
		at := now.Format(time.RFC3339Nano)
		want := []string{
			"ShareAccountByEmail bob@corp.com",
			"CreateWorkspace " + got.ID.String() + ` "Acme" acme <nil> UTC by ` + bob.ID.String() + " at " + at,
			"CreateMember " + bob.ID.String() + " in " + got.ID.String() + " as 20 by " + bob.ID.String() + " at " + at,
		}
		if !slices.Equal(f.log.calls, want) || f.tx.calls != 1 {
			t.Errorf("enabled %v: calls = %q in %d transactions, want %q in one", enabled, f.log.calls, f.tx.calls, want)
		}
		if line, want := logged(f), createdLog(got.ID, bob, "cli"); line != want {
			t.Errorf("enabled %v: log = %q, want %q", enabled, line, want)
		}
	}
}

// No account, a deactivated one read under the lock, invalid values: the
// command's errors, and nothing written.
func TestExecuteForAdminRefuses(t *testing.T) {
	tests := []struct {
		name  string
		email string
		w     domain.NewWorkspace
		want  error
		calls int
	}{
		{"no account", "dave@corp.com", domain.NewWorkspace{Name: "Acme", Slug: "acme"}, domain.ErrAccountNotFound, 1},
		{"a deactivated account", "carol@corp.com", domain.NewWorkspace{Name: "Acme", Slug: "acme"}, domain.ErrAccountDeactivated, 1},
		{"a reserved slug", "alice@corp.com", domain.NewWorkspace{Name: "Acme", Slug: "healthz"}, shared.Invalid(), 0},
	}
	for _, tt := range tests {
		uc, f := newCreate(true)
		if _, err := uc.ExecuteForAdmin(context.Background(), tt.email, tt.w); !errors.Is(err, tt.want) {
			t.Errorf("%s: ExecuteForAdmin() = %v, want %v", tt.name, err, tt.want)
		}
		if len(f.log.calls) != tt.calls || len(f.workspaces.members) != 0 || f.logs.Len() != 0 {
			t.Errorf("%s: calls = %q, log %q; want %d, no write and no log", tt.name, f.log.calls, f.logs, tt.calls)
		}
	}
}
````

`server/internal/modules/workspace/app/read_test.go`（新文件，142 行）：

````file server/internal/modules/workspace/app/read_test.go
package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	acme = domain.Workspace{ID: uuid.NewV7(), Name: "Acme", Slug: "acme", Timezone: "UTC", TotalMembers: 3, CreatedAt: now, UpdatedAt: now}
	beta = domain.Workspace{ID: uuid.NewV7(), Name: "Beta", Slug: "beta", Timezone: "UTC", TotalMembers: 1, CreatedAt: now, UpdatedAt: now}
)

// ListWorkspaces is the caller's list, asked of the store for the caller:
// two callers, two lists.
func TestListWorkspacesIsTheCallersList(t *testing.T) {
	log := &callLog{}
	aliceList := []domain.Workspace{acme, beta}
	store := &fakeWorkspaces{log: log, lists: map[uuid.UUID][]domain.Workspace{alice.ID: aliceList, bob.ID: {beta}}}
	uc := app.NewListWorkspaces(store)
	for user, want := range map[app.AccountState][]domain.Workspace{alice: aliceList, bob: {beta}, carol: nil} {
		log.calls = nil
		got, err := uc.Execute(as(user))
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("%s: Execute() = %v, %v; want %v", user.Email, got, err, want)
		}
		if want := []string{"ListWorkspaces " + user.ID.String() + " outside tx"}; !slices.Equal(log.calls, want) {
			t.Errorf("%s: calls = %q, want %q", user.Email, log.calls, want)
		}
	}
	if _, err := uc.Execute(context.Background()); err == nil {
		t.Error("Execute() without an actor = nil, want an error")
	}
}

// GetWorkspace reads the workspace, then asks the Authorizer for
// workspace.read on it, for the caller, without a transaction: the answer
// carries the caller's role from the grant. Two callers, two workspaces.
func TestGetWorkspaceDecidesOnTheWorkspaceFound(t *testing.T) {
	log := &callLog{}
	store := &fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme, beta}}
	auth := &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{
		{alice.ID, acme.ID}: {WorkspaceRole: shared.RoleGuest},
		{alice.ID, beta.ID}: {WorkspaceRole: shared.RoleAdmin},
		{bob.ID, beta.ID}:   {WorkspaceRole: shared.RoleMember},
	}}
	uc := app.NewGetWorkspace(store, auth)
	tests := []struct {
		user app.AccountState
		w    domain.Workspace
		role shared.Role
	}{
		{alice, acme, shared.RoleGuest},
		{alice, beta, shared.RoleAdmin},
		{bob, beta, shared.RoleMember},
	}
	for _, tt := range tests {
		log.calls = nil
		got, err := uc.Execute(as(tt.user), tt.w.Slug)
		want := tt.w
		want.Role = tt.role
		if err != nil || got != want {
			t.Errorf("%s, %s: Execute() = %+v, %v; want %+v", tt.user.Email, tt.w.Slug, got, err, want)
		}
		wantCalls := []string{
			"WorkspaceBySlug " + tt.w.Slug + " outside tx",
			"Authorize " + tt.user.ID.String() + " workspace.read on " + tt.w.ID.String() + "/" + uuid.Nil().String() + " outside tx",
		}
		if !slices.Equal(log.calls, wantCalls) {
			t.Errorf("%s, %s: calls = %q, want %q", tt.user.Email, tt.w.Slug, log.calls, wantCalls)
		}
	}
}

// A workspace the caller cannot see and one that is not there are the same
// workspace.not_found; a refusal that is not "not visible" is not turned
// into one.
func TestGetWorkspaceNotFound(t *testing.T) {
	log := &callLog{}
	failure := errors.New("connection reset")
	store := &fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme, beta}}
	auth := &fakeAuthorizer{log: log, errs: map[grantKey]error{
		{bob.ID, beta.ID}:   shared.Forbidden(),
		{carol.ID, beta.ID}: failure,
	}}
	uc := app.NewGetWorkspace(store, auth)
	tests := []struct {
		name  string
		user  app.AccountState
		slug  string
		want  error
		calls int
	}{
		{"not visible", bob, "acme", domain.ErrNotFound, 2},
		{"no such workspace", alice, "nothing", domain.ErrNotFound, 1},
		{"forbidden", bob, "beta", shared.Forbidden(), 2},
		{"the Authorizer failed", carol, "beta", failure, 2},
	}
	for _, tt := range tests {
		log.calls = nil
		if _, err := uc.Execute(as(tt.user), tt.slug); !errors.Is(err, tt.want) {
			t.Errorf("%s: Execute() = %v, want %v", tt.name, err, tt.want)
		}
		if len(log.calls) != tt.calls {
			t.Errorf("%s: calls = %q, want %d", tt.name, log.calls, tt.calls)
		}
	}
}

// CheckSlug looks up only a slug that could be used.
func TestCheckSlug(t *testing.T) {
	log := &callLog{}
	uc := app.NewCheckSlug(&fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme}})
	tests := []struct {
		slug   string
		want   domain.SlugReason
		lookup bool
	}{
		{"free", "", true},
		{"acme", domain.SlugTaken, true},
		{"Acme", domain.SlugInvalid, false},
		{"", domain.SlugInvalid, false},
		{"settings", domain.SlugReserved, false},
		{"assets", domain.SlugReserved, false},
	}
	for _, tt := range tests {
		log.calls = nil
		got, err := uc.Execute(context.Background(), tt.slug)
		if err != nil || got != tt.want {
			t.Errorf("Execute(%q) = %q, %v; want %q", tt.slug, got, err, tt.want)
		}
		if lookup := len(log.calls) > 0; lookup != tt.lookup || (lookup && log.calls[0] != "SlugTaken "+tt.slug+" outside tx") {
			t.Errorf("Execute(%q): calls = %q, want a lookup: %v", tt.slug, log.calls, tt.lookup)
		}
	}
}
````

- [ ] **Step 3: 测试、lint**

Run: `go -C server test -count=1 ./internal/modules/workspace/app/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/modules/workspace/app/check_slug.go server/internal/modules/workspace/app/create_workspace.go server/internal/modules/workspace/app/create_workspace_test.go server/internal/modules/workspace/app/fakes_test.go server/internal/modules/workspace/app/get_workspace.go server/internal/modules/workspace/app/list_workspaces.go server/internal/modules/workspace/app/read_test.go
```
```bash
git commit -m "feat(M3/P1): create, list, read a workspace and check a slug

createWorkspace locks the caller's account row first, in its transaction,
and requires it active under the lock (M3 design 3.6 convention 6); the
server's administrator creates for the account of an address whatever the
switch says. getWorkspace asks the Authorizer, a workspace not visible is
not found.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 14 个用例测试通过；`workspace/app` 只导入 `workspace/domain`、`shared` 和标准库。

---

### Task 11: 接口描述、HTTP 一侧和两段组合

**Files:**
- Create: `api/modules/workspace.yaml`、`server/internal/bootstrap/ports.go`、`server/internal/bootstrap/ports_test.go`、`server/internal/bootstrap/workspace_test.go`、`server/internal/modules/workspace/adapter/http/gen/oapi-codegen.yaml`、`server/internal/modules/workspace/adapter/http/handler.go`、`server/internal/modules/workspace/adapter/http/handler_test.go`、`server/internal/modules/workspace/adapter/http/workspaces.go`、`server/internal/modules/workspace/adapter/http/workspaces_test.go`、`server/internal/modules/workspace/module.go`
- Modify: `api/openapi.yaml`、`server/internal/bootstrap/app.go`、`web/apps/web/helpers/authentication.helper.ts`、`web/packages/i18n/src/locales/en/auth.json`、`web/packages/i18n/src/locales/zh-CN/auth.json`
- Generate: `api/dist/openapi.yaml`、`server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go`、`server/internal/modules/workspace/adapter/http/gen/server.gen.go`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.12、2.13，M3 设计 5.1–5.3、6.6、9.4）：
  - `api/modules/workspace.yaml`：`listWorkspaces`（`GET /api/v0/workspaces`）、`createWorkspace`（`POST /api/v0/workspaces`，201）、`getWorkspace`（`GET /api/v0/workspaces/{slug}`）、`checkWorkspaceSlug`（`GET /api/v0/workspace-slugs/{slug}`），结构和码见 spec 2.12；`api/openapi.yaml` 加 `workspace` 标签和三个路径；
  - `adapter/http`：`UseCases{ListWorkspaces, CreateWorkspace, GetWorkspace, CheckSlug}`（四个小接口）、`Register(router, api, uc)`；
  - `workspace` 模块入口：`WorkspaceRoles`、`Provided`、`Provide(pool)`、`AccountState`（`app.AccountState` 的别名）、`Deps{Pool, Tx, Clock, Logger, Authorizer, Accounts, CreationEnabled}`、`New(Deps) *Module`、`(*Module).Register`；
  - `bootstrap/app.go`：M3 设计 6.6 的顺序；`bootstrap/ports.go` 的 `workspaceAccounts`；
  - 前端文案表加 `workspace.not_found`、`workspace.creation_disabled`、`workspace.slug_taken`（`auth` 命名空间，`en`、`zh-CN`；spec 第 3 节第 4 条）。
- 使用者：Task 12（`Actions`、`ReservedSlugs` 加进 `module.go`）、Task 14（`NewAdmin`、`workspaceAccounts`）、Task 15、16。

**Tests:**
- `workspace/adapter/http/handler_test.go`：`TestMain` 调 `apitest.Main(m, "workspace")`：`workspace.yaml` 声明的每个码都要在这个包里经 `CheckResponse` 返回过（M3 设计 9.4）；假用例按参数回答并记下参数。
- `workspaces_test.go`：`TestListWorkspacesAnswersTheCallersList`（两个调用者）；`TestCreateWorkspaceAnswers201`（请求体变成用例的输入，可选字段缺省时为 nil；201 和工作区）；`TestCreateWorkspaceRefusals`（`workspace.creation_disabled`、保留的 slug 的 `validation_failed`、`workspace.slug_taken`、账户在此期间停用的 401，逐字节比较 problem）；`TestCreateWorkspacePassesAnUnknownSizeToTheUseCase`；`TestGetWorkspace`（调用者和路径中的 slug 交给用例；`workspace.not_found`）；`TestCheckWorkspaceSlug`（可用；三种原因；路径中转义的 slug 解码后交给用例）。每个响应都经契约核对。
- `bootstrap/ports_test.go`：`TestWorkspaceAccountsConvertsIdentitysAnswer`（假 `identity.Accounts` 记下参数；状态、`found`、错误原样）。
- `bootstrap/workspace_test.go`：`TestCreatingAWorkspaceAsConfigured`：真实组合和数据库，开关两种值。开：alice 建 acme，是它唯一的管理员；她的列表有它、读得到它；bob 读到 `workspace.not_found`。关：403 `workspace.creation_disabled`，没有写入。
- 不改而覆盖：M2 的整程序测试 `TestAPIRoutesAreTheContractsOperations`、`TestOperationsThatNeedATokenAnswer401WithoutOne`、`TestTheAnswerToABrokenBodyStaysSmall`、`TestParametersThatDoNotBindAnswer400`、`TestFieldCodesAreTheContractsEnum`（`server/internal/bootstrap`）自动覆盖四个新操作；前端的 vitest 核对文案表的键恰好等于契约的全部码。

- [ ] **Step 1: 接口描述**

`api/modules/workspace.yaml`（新文件，218 行）：

````file api/modules/workspace.yaml
# workspace 模块的接口（M3 设计 5.1、5.2）。生成的 Go 代码在
# server/internal/modules/workspace/adapter/http/gen（make gen-go）。
openapi: 3.1.0
info:
  title: Nerve workspace API
  version: v0
paths:
  /api/v0/workspaces:
    get:
      operationId: listWorkspaces
      tags: [workspace]
      summary: List the caller's workspaces
      description: >-
        The workspaces of which the caller is an active member, each with the
        caller's role and the number of active members, by name and then by
        id. The whole collection at once: collections are not paginated.
      security: [{bearer: []}]
      x-problem-codes: []
      responses:
        '200':
          description: The caller's workspaces.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/WorkspaceList'
        default:
          $ref: '#/components/responses/Problem'
    post:
      operationId: createWorkspace
      tags: [workspace]
      summary: Create a workspace with the caller as its admin
      description: >-
        Creates the workspace and makes the caller its admin and only member;
        nothing else is created with it. While the instance's
        workspace_creation_enabled is false, every request answers
        workspace.creation_disabled. The name has 1–80
        characters with a letter or a digit and no web address; the slug has
        1–48 lower-case letters, digits, - and _, and is neither reserved
        (not_allowed) nor another undeleted workspace's
        (workspace.slug_taken). An account deactivated meanwhile answers
        unauthorized.
      security: [{bearer: []}]
      x-problem-codes: [workspace.creation_disabled, validation_failed, workspace.slug_taken]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/WorkspaceCreate'
      responses:
        '201':
          description: The new workspace; the caller is its admin.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Workspace'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspaces/{slug}:
    parameters:
      - $ref: '#/components/parameters/Slug'
    get:
      operationId: getWorkspace
      tags: [workspace]
      summary: Read a workspace
      description: >-
        A workspace that does not exist, is deleted, or of which the caller
        is not an active member answers the same workspace.not_found.
      security: [{bearer: []}]
      x-problem-codes: [workspace.not_found]
      responses:
        '200':
          description: The workspace, with the caller's role.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Workspace'
        default:
          $ref: '#/components/responses/Problem'
  /api/v0/workspace-slugs/{slug}:
    parameters:
      - $ref: '#/components/parameters/Slug'
    get:
      operationId: checkWorkspaceSlug
      tags: [workspace]
      summary: Tell whether a slug can name a new workspace
      description: >-
        Available, or why not: invalid (not 1–48 lower-case letters, digits,
        - and _), reserved (a path of this site, or held for one), or taken
        (an undeleted workspace has it). It tells nothing about the workspace
        that has it.
      security: [{bearer: []}]
      x-problem-codes: []
      responses:
        '200':
          description: Whether the slug is available.
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/SlugAvailability'
        default:
          $ref: '#/components/responses/Problem'
components:
  # 与 api/openapi.yaml 的相同：打包时丢掉模块文件的 securitySchemes，oapi-codegen 读这一份（M0-P3 交接 3）
  securitySchemes:
    bearer:
      type: http
      scheme: bearer
  parameters:
    Slug:
      name: slug
      in: path
      required: true
      description: A workspace's slug, as in the web app's address.
      schema:
        type: string
  responses:
    Problem:
      description: Error (RFC 9457 problem details).
      headers:
        Retry-After:
          description: >-
            Whole seconds to wait before trying again, rounded up; sent with
            rate_limited and server_busy.
          schema:
            type: integer
            minimum: 1
        WWW-Authenticate:
          description: >-
            Sent with every 401 (RFC 9110 15.5.2): Bearer error="invalid_token"
            (RFC 6750 3) when the bearer token sent is refused before the
            operation runs, being invalid or expired; plain Bearer for every
            other 401, including a credential that the operation finds revoked
            while it runs.
          schema:
            type: string
      content:
        application/problem+json:
          schema:
            $ref: '../common.yaml#/components/schemas/Problem'
  schemas:
    WorkspaceRole:
      description: A member's role in a workspace, 5 guest, 15 member, 20 admin.
      type: integer
      enum: [5, 15, 20]
    OrganizationSize:
      description: The size of the organization, as the web app's form offers it.
      type: string
      enum: [Just myself, 2-10, 11-50, 51-200, 201-500, 500+]
    Workspace:
      type: object
      additionalProperties: false
      required: [id, name, slug, organization_size, timezone, logo_url, role, total_members, created_at, updated_at]
      properties:
        id:
          type: string
          format: uuid
        name:
          type: string
        slug:
          description: Lower case; it never changes.
          type: string
        organization_size:
          description: One of OrganizationSize's values; null when none was given.
          type: [string, 'null']
        timezone:
          description: An IANA time zone name.
          type: string
        logo_url:
          description: Null until uploads arrive (M5).
          type: [string, 'null']
        role:
          $ref: '#/components/schemas/WorkspaceRole'
        total_members:
          description: The number of active members.
          type: integer
        created_at:
          type: string
          format: date-time
        updated_at:
          type: string
          format: date-time
    WorkspaceList:
      type: object
      additionalProperties: false
      required: [data]
      properties:
        data:
          type: array
          items:
            $ref: '#/components/schemas/Workspace'
    WorkspaceCreate:
      type: object
      additionalProperties: false
      required: [name, slug]
      properties:
        name:
          description: 1–80 characters, with a letter or a digit, without a web address.
          type: string
        slug:
          description: 1–48 lower-case letters, digits, - and _; not reserved.
          type: string
        organization_size:
          $ref: '#/components/schemas/OrganizationSize'
        timezone:
          description: An IANA time zone name, e.g. from GET /api/v0/timezones; UTC when not given.
          type: string
    SlugAvailability:
      type: object
      additionalProperties: false
      required: [available]
      properties:
        available:
          type: boolean
        reason:
          description: Why the slug is not available; absent when it is.
          type: string
          enum: [invalid, reserved, taken]
````

`api/openapi.yaml`（修改，2 处）：

````old api/openapi.yaml
    description: What this instance runs.
````

````new api/openapi.yaml
    description: What this instance runs.
  - name: workspace
    description: Workspaces and their members.
````

````old api/openapi.yaml
    $ref: 'modules/instance.yaml#/paths/~1api~1v0~1timezones'
````

````new api/openapi.yaml
    $ref: 'modules/instance.yaml#/paths/~1api~1v0~1timezones'
  /api/v0/workspaces:
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspaces'
  /api/v0/workspaces/{slug}:
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspaces~1{slug}'
  /api/v0/workspace-slugs/{slug}:
    $ref: 'modules/workspace.yaml#/paths/~1api~1v0~1workspace-slugs~1{slug}'
````

`server/internal/modules/workspace/adapter/http/gen/oapi-codegen.yaml`（新文件，33 行）：

````file server/internal/modules/workspace/adapter/http/gen/oapi-codegen.yaml
# oapi-codegen 配置：api/modules/workspace.yaml → server.gen.go。
# 由 make gen-go 在 server/ 下执行，路径相对于 server/。
package: gen
output: internal/modules/workspace/adapter/http/gen/server.gen.go
generate:
  models: true
  std-http-server: true
  strict-server: true
compatibility:
  # 枚举常量总是带类型名前缀：以后别的枚举出现同名的值，也不会改掉已有常量的名字
  always-prefix-enum-values: true
# output-options 是每个模块照抄的模板（M2 设计 3.12），bodyshapegen 读同一份 type-mapping
output-options:
  name-normalizer: ToCamelCaseWithInitialisms
  # 可为空又可省略的字段生成 nullable.Nullable[T]：PATCH 能区分"没传"和"传 null"
  nullable-type: true
  type-mapping:
    # 不带 format 的 number 默认生成 float32，bodyshape 查不了它的范围；bodyshapegen 只接受 int、int64 和 float64
    number:
      default:
        type: float64
    string:
      formats:
        # 标准库的 uuid：默认的 openapi_types.UUID 会带进 github.com/google/uuid
        uuid:
          type: uuid.UUID
          import: uuid
        # 邮箱的格式由领域层校验（422）；默认的 openapi_types.Email 在解码时自己校验，绕过这一层
        email:
          type: string
import-mapping:
  # common.yaml 的组件生成在共享包 apigen 里，这里只引用
  ../common.yaml: github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apigen
````

- [ ] **Step 2: 生成**

Run: `make gen`
Expected: 成功；生成物：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `c53d06195061383e59e39bbba26689713f64bdc373ebf174cb89986caec2e523` | 1084 | `api/dist/openapi.yaml` |
| `e6ac2a7b99195d4c0be18c99d26750143226e1f8ec1d12f0b062d651bf33c012` | 23 | `server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go` |
| `77a415bf25672be4409e50e29e0f0a9127cf4731810f65665300120108326c39` | 752 | `server/internal/modules/workspace/adapter/http/gen/server.gen.go` |
| `4933ed593887eb77c81c58315d26276ad880a3ccbd7b53d22087ee290319a3d6` | 1134 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go server/internal/modules/workspace/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 3: HTTP 一侧、模块入口、组合**

`server/internal/modules/workspace/adapter/http/handler.go`（新文件，65 行）：

````file server/internal/modules/workspace/adapter/http/handler.go
// Package httpadapter serves the workspace module's API: it implements the
// strict server that oapi-codegen generates from api/modules/workspace.yaml
// into the gen package, and translates between the generated types and the
// use cases.
package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// ListWorkspacesUseCase is app.ListWorkspaces.
type ListWorkspacesUseCase interface {
	Execute(ctx context.Context) ([]domain.Workspace, error)
}

// CreateWorkspaceUseCase is app.CreateWorkspace's entry for the API.
type CreateWorkspaceUseCase interface {
	Execute(ctx context.Context, w domain.NewWorkspace) (domain.Workspace, error)
}

// GetWorkspaceUseCase is app.GetWorkspace.
type GetWorkspaceUseCase interface {
	Execute(ctx context.Context, slug string) (domain.Workspace, error)
}

// CheckSlugUseCase is app.CheckSlug.
type CheckSlugUseCase interface {
	Execute(ctx context.Context, slug string) (domain.SlugReason, error)
}

// UseCases are the use cases behind the module's operations.
type UseCases struct {
	ListWorkspaces  ListWorkspacesUseCase
	CreateWorkspace CreateWorkspaceUseCase
	GetWorkspace    GetWorkspaceUseCase
	CheckSlug       CheckSlugUseCase
}

// Register mounts the module's routes on router behind api's per-route
// middlewares; api.Errors answers binding, decoding and handler errors.
// Every operation needs a token.
func Register(router *httpserver.Router, api *httpserver.API, uc UseCases) {
	strict := gen.NewStrictHandlerWithOptions(handler{uc: uc}, nil, gen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  api.Errors.BodyError,
		ResponseErrorHandlerFunc: api.Errors.Write,
	})
	var middlewares []gen.MiddlewareFunc
	for _, m := range api.Middlewares(gen.BodyShapes()) {
		middlewares = append(middlewares, m)
	}
	gen.HandlerWithOptions(strict, gen.StdHTTPServerOptions{
		BaseRouter:       router,
		Middlewares:      middlewares,
		ErrorHandlerFunc: api.Errors.BadRequest,
	})
}

// handler implements gen.StrictServerInterface.
type handler struct {
	uc UseCases
}
````

`server/internal/modules/workspace/adapter/http/workspaces.go`（新文件，81 行）：

````file server/internal/modules/workspace/adapter/http/workspaces.go
package httpadapter

import (
	"context"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// ListWorkspaces serves GET /api/v0/workspaces.
func (h handler) ListWorkspaces(ctx context.Context, _ gen.ListWorkspacesRequestObject) (gen.ListWorkspacesResponseObject, error) {
	list, err := h.uc.ListWorkspaces.Execute(ctx)
	if err != nil {
		return nil, err
	}
	out := gen.ListWorkspaces200JSONResponse{Data: make([]gen.Workspace, len(list))}
	for i, w := range list {
		out.Data[i] = workspace(w)
	}
	return out, nil
}

// CreateWorkspace serves POST /api/v0/workspaces.
func (h handler) CreateWorkspace(ctx context.Context, req gen.CreateWorkspaceRequestObject) (gen.CreateWorkspaceResponseObject, error) {
	in := domain.NewWorkspace{Name: req.Body.Name, Slug: req.Body.Slug, Timezone: req.Body.Timezone}
	if size := req.Body.OrganizationSize; size != nil {
		s := string(*size)
		in.OrganizationSize = &s
	}
	w, err := h.uc.CreateWorkspace.Execute(ctx, in)
	if err != nil {
		return nil, err
	}
	return gen.CreateWorkspace201JSONResponse(workspace(w)), nil
}

// GetWorkspace serves GET /api/v0/workspaces/{slug}.
func (h handler) GetWorkspace(ctx context.Context, req gen.GetWorkspaceRequestObject) (gen.GetWorkspaceResponseObject, error) {
	w, err := h.uc.GetWorkspace.Execute(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	return gen.GetWorkspace200JSONResponse(workspace(w)), nil
}

// CheckWorkspaceSlug serves GET /api/v0/workspace-slugs/{slug}.
func (h handler) CheckWorkspaceSlug(ctx context.Context, req gen.CheckWorkspaceSlugRequestObject) (gen.CheckWorkspaceSlugResponseObject, error) {
	reason, err := h.uc.CheckSlug.Execute(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	if reason == "" {
		return gen.CheckWorkspaceSlug200JSONResponse{Available: true}, nil
	}
	r := gen.SlugAvailabilityReason(reason)
	return gen.CheckWorkspaceSlug200JSONResponse{Available: false, Reason: &r}, nil
}

// workspace is w as the API shows it.
func workspace(w domain.Workspace) gen.Workspace {
	size := nullable.NewNullNullable[string]()
	if w.OrganizationSize != nil {
		size = nullable.NewNullableWithValue(*w.OrganizationSize)
	}
	return gen.Workspace{
		ID:               w.ID,
		Name:             w.Name,
		Slug:             w.Slug,
		OrganizationSize: size,
		Timezone:         w.Timezone,
		// Required and null until M5. The zero Nullable is "unspecified" and
		// would marshal as "": set null explicitly.
		LogoURL:      nullable.NewNullNullable[string](),
		Role:         gen.WorkspaceRole(w.Role),
		TotalMembers: w.TotalMembers,
		CreatedAt:    w.CreatedAt,
		UpdatedAt:    w.UpdatedAt,
	}
}
````

`server/internal/modules/workspace/module.go`（新文件，80 行）：

````file server/internal/modules/workspace/module.go
// Package workspace is the workspaces module (M3 design 3.3, 6.2):
// workspaces and their members. It brings creating, listing and reading
// workspaces, and checking a slug, and offers the other modules its reads
// through ports.
package workspace

import (
	"context"
	"log/slog"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/http"
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// WorkspaceRoles reads a caller's active role in a workspace: access's port
// (M3 design 6.5). ok is false unless the membership is active, its row not
// deleted and the workspace not deleted. It reads in the transaction ctx
// carries.
type WorkspaceRoles interface {
	ActiveRole(ctx context.Context, workspaceID, userID uuid.UUID) (role shared.Role, ok bool, err error)
}

// Provided are the adapters workspace offers the other modules. They depend
// on the pool alone, so bootstrap builds them before any module (M3 design
// 6.6, step 2).
type Provided struct {
	WorkspaceRoles WorkspaceRoles
}

// Provide builds workspace's adapters for the other modules.
func Provide(pool *pgxpool.Pool) Provided {
	return Provided{WorkspaceRoles: postgresadapter.New(pool)}
}

// AccountState is the account state the Accounts port hands over: bootstrap
// converts identity's into it (M3 design 6.5).
type AccountState = app.AccountState

// Deps are what bootstrap gives the module (M3 design 6.6, step 5).
type Deps struct {
	Pool       *pgxpool.Pool
	Tx         shared.TxManager
	Clock      app.Clock
	Logger     *slog.Logger
	Authorizer shared.Authorizer
	// Accounts is identity's Accounts, converted (bootstrap/ports.go).
	Accounts app.Accounts
	// CreationEnabled is workspace.creation_enabled (M3 design 3.11).
	CreationEnabled bool
}

// Module is the wired workspace module.
type Module struct {
	uc httpadapter.UseCases
}

// New wires the module's use cases and its HTTP side from d; nothing is
// registered or injected after it (M3 design 6.6).
func New(d Deps) *Module {
	store := postgresadapter.New(d.Pool)
	return &Module{uc: httpadapter.UseCases{
		ListWorkspaces: app.NewListWorkspaces(store),
		CreateWorkspace: app.NewCreateWorkspace(app.CreateWorkspaceDeps{
			Accounts: d.Accounts, Workspaces: store, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger, Enabled: d.CreationEnabled,
		}),
		GetWorkspace: app.NewGetWorkspace(store, d.Authorizer),
		CheckSlug:    app.NewCheckSlug(store),
	}}
}

// Register mounts the module's API on router behind api's middlewares.
func (m *Module) Register(router *httpserver.Router, api *httpserver.API) {
	httpadapter.Register(router, api, m.uc)
}
````

`server/internal/bootstrap/ports.go`（新文件，29 行）：

````file server/internal/bootstrap/ports.go
package bootstrap

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
)

// The values that cross from one module's port to another's: modules do not
// import each other, so where an implementation answers its own type, a few
// lines here convert it (M3 design 6.5).

// workspaceAccounts is identity's Accounts as workspace's port: the same
// locks, identity's AccountState converted into workspace's.
type workspaceAccounts struct {
	accounts identity.Accounts
}

func (a workspaceAccounts) ShareAccount(ctx context.Context, id uuid.UUID) (workspace.AccountState, bool, error) {
	state, found, err := a.accounts.ShareAccount(ctx, id)
	return workspace.AccountState(state), found, err
}

func (a workspaceAccounts) ShareAccountByEmail(ctx context.Context, email string) (workspace.AccountState, bool, error) {
	state, found, err := a.accounts.ShareAccountByEmail(ctx, email)
	return workspace.AccountState(state), found, err
}
````

`server/internal/bootstrap/app.go`（修改，5 处）：

````old server/internal/bootstrap/app.go
	"github.com/jackc/pgx/v5/pgxpool"

````

````new server/internal/bootstrap/app.go
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
````

````old server/internal/bootstrap/app.go
	"github.com/open-nerve/NerveProject/server/internal/modules/instance"
````

````new server/internal/bootstrap/app.go
	"github.com/open-nerve/NerveProject/server/internal/modules/instance"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
````

````old server/internal/bootstrap/app.go
	limiter := ratelimit.New(time.Now)

````

````new server/internal/bootstrap/app.go
	limiter := ratelimit.New(time.Now)
	tx := postgres.NewTxManager(pool, cfg.Database.CommitTimeout)

	// The modules in two steps (M3 design 6.6): first the adapters each
	// offers the others, built from the pool alone; then the modules, each
	// with every port it needs, so no construction waits on another.
	identityPorts := identity.Provide(pool)
	workspacePorts := workspace.Provide(pool)
	authorizer := access.New(access.Deps{WorkspaceRoles: workspacePorts.WorkspaceRoles})
	ws := workspace.New(workspace.Deps{
		Pool: pool, Tx: tx, Clock: clock.System{}, Logger: logger, Authorizer: authorizer,
		Accounts:        workspaceAccounts{accounts: identityPorts.Accounts},
		CreationEnabled: cfg.Workspace.CreationEnabled,
	})
````

````old server/internal/bootstrap/app.go
		Tx:              postgres.NewTxManager(pool, cfg.Database.CommitTimeout),
````

````new server/internal/bootstrap/app.go
		Tx:              tx,
````

````old server/internal/bootstrap/app.go
	inst.Register(a.router, api)
````

````new server/internal/bootstrap/app.go
	inst.Register(a.router, api)
	ws.Register(a.router, api)
````

- [ ] **Step 4: 前端的文案**

`web/apps/web/helpers/authentication.helper.ts`（修改，1 处）：

````old web/apps/web/helpers/authentication.helper.ts
  "identity.api_token_not_found": "auth.errors.api_token_not_found",
````

````new web/apps/web/helpers/authentication.helper.ts
  "identity.api_token_not_found": "auth.errors.api_token_not_found",
  "workspace.not_found": "auth.errors.workspace_not_found",
  "workspace.creation_disabled": "auth.errors.workspace_creation_disabled",
  "workspace.slug_taken": "auth.errors.workspace_slug_taken",
````

`web/packages/i18n/src/locales/en/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/en/auth.json
      "api_token_not_found": "The token does not exist or has been revoked.",
````

````new web/packages/i18n/src/locales/en/auth.json
      "api_token_not_found": "The token does not exist or has been revoked.",
      "workspace_not_found": "The workspace does not exist.",
      "workspace_creation_disabled": "Creating workspaces is turned off on this server.",
      "workspace_slug_taken": "A workspace with this URL already exists.",
````

`web/packages/i18n/src/locales/zh-CN/auth.json`（修改，1 处）：

````old web/packages/i18n/src/locales/zh-CN/auth.json
      "api_token_not_found": "令牌不存在或已被撤销。",
````

````new web/packages/i18n/src/locales/zh-CN/auth.json
      "api_token_not_found": "令牌不存在或已被撤销。",
      "workspace_not_found": "工作区不存在。",
      "workspace_creation_disabled": "本服务器已关闭创建工作区。",
      "workspace_slug_taken": "已有工作区使用这个地址。",
````

- [ ] **Step 5: 测试**

`server/internal/modules/workspace/adapter/http/handler_test.go`（新文件，178 行）：

````file server/internal/modules/workspace/adapter/http/handler_test.go
package httpadapter_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Every problem code the module's operations declare must be answered by a
// test here (M2 design 3.11), whichever module produces it (M3 design 9.4).
func TestMain(m *testing.M) { apitest.Main(m, "workspace") }

var (
	aliceID = uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	bobID   = uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")
	created = time.Date(2026, 9, 29, 10, 0, 0, 123456000, time.UTC)
)

// fakeAuth accepts the tokens "alice" and "bob" as those accounts.
type fakeAuth struct{}

func (fakeAuth) Authenticate(ctx context.Context, token string) (context.Context, string, error) {
	switch token {
	case "alice":
		return shared.WithActor(ctx, shared.Actor{UserID: aliceID, SessionID: uuid.NewV7()}), "session:alice", nil
	case "bob":
		return shared.WithActor(ctx, shared.Actor{UserID: bobID, SessionID: uuid.NewV7()}), "session:bob", nil
	}
	return nil, "", shared.Unauthenticated()
}

// caller is the account the context's actor is, by name.
func caller(ctx context.Context) string {
	actor, err := shared.RequireActor(ctx)
	switch {
	case err != nil:
		return "nobody"
	case actor.UserID == aliceID:
		return "alice"
	case actor.UserID == bobID:
		return "bob"
	}
	return "someone else"
}

// fakes are the use cases behind a test server: each records who called it
// with what, and answers what it is given.
type fakes struct {
	list   *fakeList
	create *fakeCreate
	get    *fakeGet
	check  *fakeCheck
}

type fakeList struct {
	calls []string
	lists map[string][]domain.Workspace
}

func (f *fakeList) Execute(ctx context.Context) ([]domain.Workspace, error) {
	f.calls = append(f.calls, caller(ctx))
	return f.lists[caller(ctx)], nil
}

type fakeCreate struct {
	callers []string
	got     []domain.NewWorkspace
	answer  domain.Workspace
	err     error
}

func (f *fakeCreate) Execute(ctx context.Context, w domain.NewWorkspace) (domain.Workspace, error) {
	f.callers = append(f.callers, caller(ctx))
	f.got = append(f.got, w)
	return f.answer, f.err
}

type fakeGet struct {
	calls      []string
	workspaces map[string]domain.Workspace // "caller slug" → workspace
}

func (f *fakeGet) Execute(ctx context.Context, slug string) (domain.Workspace, error) {
	key := caller(ctx) + " " + slug
	f.calls = append(f.calls, key)
	w, ok := f.workspaces[key]
	if !ok {
		return domain.Workspace{}, domain.ErrNotFound
	}
	return w, nil
}

type fakeCheck struct {
	calls   []string
	reasons map[string]domain.SlugReason
}

func (f *fakeCheck) Execute(ctx context.Context, slug string) (domain.SlugReason, error) {
	f.calls = append(f.calls, caller(ctx)+" "+slug)
	return f.reasons[slug], nil
}

// newServer serves the module with f; a fake left nil is an idle one.
func newServer(t *testing.T, f fakes) http.Handler {
	t.Helper()
	logger := slog.New(slog.DiscardHandler)
	router := httpserver.NewRouter(logger)
	limit := ratelimit.New(time.Now).Bucket("test", ratelimit.Rate{PerMinute: 600, Burst: 100})
	api, err := httpserver.NewAPI(httpserver.APIConfig{
		Logger:         logger,
		Authenticator:  fakeAuth{},
		MaxBodyBytes:   1024,
		RequestTimeout: 5 * time.Second,
		IPv6PrefixLen:  64,
		Anonymous:      limit,
		Authenticated:  limit,
		AuthFailure:    limit,
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.list == nil {
		f.list = &fakeList{}
	}
	if f.create == nil {
		f.create = &fakeCreate{}
	}
	if f.get == nil {
		f.get = &fakeGet{}
	}
	if f.check == nil {
		f.check = &fakeCheck{}
	}
	httpadapter.Register(router, api, httpadapter.UseCases{
		ListWorkspaces: f.list, CreateWorkspace: f.create, GetWorkspace: f.get, CheckSlug: f.check,
	})
	return router
}

// do serves req and checks the answer against the contract.
func do(t *testing.T, h http.Handler, req *http.Request) (*http.Response, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	res := rec.Result()
	apitest.Load(t).CheckResponse(t, req, res)
	body, _ := io.ReadAll(res.Body)
	return res, string(body)
}

// request is a request with token as its bearer and body as JSON, if any.
func request(method, path, token, body string) *http.Request {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}
````

`server/internal/modules/workspace/adapter/http/workspaces_test.go`（新文件，168 行）：

````file server/internal/modules/workspace/adapter/http/workspaces_test.go
package httpadapter_test

import (
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	acmeID = uuid.MustParse("0199a2b4-0000-7000-8000-00000000000a")
	betaID = uuid.MustParse("0199a2b4-0000-7000-8000-00000000000b")
	size   = "2-10"
	acme   = domain.Workspace{ID: acmeID, Name: "Acme", Slug: "acme", OrganizationSize: &size, Timezone: "Asia/Shanghai",
		Role: shared.RoleAdmin, TotalMembers: 3, CreatedAt: created, UpdatedAt: created.Add(time.Hour)}
	beta = domain.Workspace{ID: betaID, Name: "Beta", Slug: "beta", Timezone: "UTC", Role: shared.RoleGuest, TotalMembers: 1,
		CreatedAt: created, UpdatedAt: created}
)

const (
	acmeJSON = `{"created_at":"2026-09-29T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-00000000000a","logo_url":null,` +
		`"name":"Acme","organization_size":"2-10","role":20,"slug":"acme","timezone":"Asia/Shanghai","total_members":3,` +
		`"updated_at":"2026-09-29T11:00:00.123456Z"}`
	betaJSON = `{"created_at":"2026-09-29T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-00000000000b","logo_url":null,` +
		`"name":"Beta","organization_size":null,"role":5,"slug":"beta","timezone":"UTC","total_members":1,` +
		`"updated_at":"2026-09-29T10:00:00.123456Z"}`
)

// The list is the caller's, as the use case answers it: two callers.
func TestListWorkspacesAnswersTheCallersList(t *testing.T) {
	list := &fakeList{lists: map[string][]domain.Workspace{"alice": {acme, beta}, "bob": nil}}
	h := newServer(t, fakes{list: list})
	for token, want := range map[string]string{
		"alice": `{"data":[` + acmeJSON + `,` + betaJSON + `]}` + "\n",
		"bob":   `{"data":[]}` + "\n",
	} {
		res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces", token, ""))
		if res.StatusCode != http.StatusOK || body != want {
			t.Errorf("%s: GET /api/v0/workspaces = %d %s, want 200 %s", token, res.StatusCode, body, want)
		}
	}
	if want := []string{"alice", "bob"}; !slices.Equal(slices.Sorted(slices.Values(list.calls)), want) {
		t.Errorf("callers = %q, want %q", list.calls, want)
	}
}

// The body becomes the use case's input, the optional fields nil when
// absent; the answer is 201 with the workspace.
func TestCreateWorkspaceAnswers201(t *testing.T) {
	create := &fakeCreate{answer: acme}
	h := newServer(t, fakes{create: create})

	res, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces", "bob",
		`{"name":"Acme","slug":"acme","organization_size":"2-10","timezone":"Asia/Shanghai"}`))
	if res.StatusCode != http.StatusCreated || body != acmeJSON+"\n" {
		t.Errorf("POST = %d %s, want 201 %s", res.StatusCode, body, acmeJSON)
	}
	res, _ = do(t, h, request(http.MethodPost, "/api/v0/workspaces", "alice", `{"name":"Beta","slug":"beta"}`))
	if res.StatusCode != http.StatusCreated {
		t.Errorf("POST without the optional fields = %d, want 201", res.StatusCode)
	}

	zone := "Asia/Shanghai"
	want := []domain.NewWorkspace{{Name: "Acme", Slug: "acme", OrganizationSize: &size, Timezone: &zone}, {Name: "Beta", Slug: "beta"}}
	if len(create.got) != 2 || !sameInput(create.got[0], want[0]) || !sameInput(create.got[1], want[1]) {
		t.Errorf("inputs = %+v, want %+v", create.got, want)
	}
	if !slices.Equal(create.callers, []string{"bob", "alice"}) {
		t.Errorf("callers = %q, want bob, then alice", create.callers)
	}
}

func sameInput(a, b domain.NewWorkspace) bool {
	same := func(x, y *string) bool { return (x == nil) == (y == nil) && (x == nil || *x == *y) }
	return a.Name == b.Name && a.Slug == b.Slug && same(a.OrganizationSize, b.OrganizationSize) && same(a.Timezone, b.Timezone)
}

// The use case's refusals, as the contract declares them.
func TestCreateWorkspaceRefusals(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"creation disabled", domain.ErrCreationDisabled, http.StatusForbidden,
			`{"status":403,"code":"workspace.creation_disabled","title":"Forbidden","detail":"Creating workspaces is disabled on this instance."}`},
		{"a reserved slug", shared.Invalid(shared.FieldError{Field: "slug", Code: shared.FieldNotAllowed, Message: "is reserved"}), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"slug","code":"not_allowed","message":"is reserved"}]}`},
		{"a taken slug", domain.ErrSlugTaken, http.StatusConflict,
			`{"status":409,"code":"workspace.slug_taken","title":"Conflict","detail":"A workspace with this slug exists."}`},
		{"an account deactivated meanwhile", shared.Unauthenticated(), http.StatusUnauthorized,
			`{"status":401,"code":"unauthorized","title":"Unauthorized","detail":"Authentication is required."}`},
	}
	for _, tt := range tests {
		h := newServer(t, fakes{create: &fakeCreate{err: tt.err}})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces", "alice", `{"name":"Acme","slug":"acme"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// An unknown organization size is the domain's to refuse: the body's
// structure is fine, and the use case gets it.
func TestCreateWorkspacePassesAnUnknownSizeToTheUseCase(t *testing.T) {
	create := &fakeCreate{err: shared.Invalid(shared.FieldError{Field: "organization_size", Code: shared.FieldInvalidFormat, Message: "is not a known organization size"})}
	h := newServer(t, fakes{create: create})
	res, _ := do(t, h, request(http.MethodPost, "/api/v0/workspaces", "alice", `{"name":"Acme","slug":"acme","organization_size":"1000+"}`))
	if res.StatusCode != http.StatusUnprocessableEntity || len(create.got) != 1 || *create.got[0].OrganizationSize != "1000+" {
		t.Errorf("POST = %d, inputs %+v; want 422 from the use case", res.StatusCode, create.got)
	}
}

// The slug of the path goes to the use case for the caller; a workspace it
// does not find is workspace.not_found.
func TestGetWorkspace(t *testing.T) {
	get := &fakeGet{workspaces: map[string]domain.Workspace{"alice acme": acme, "bob beta": beta}}
	h := newServer(t, fakes{get: get})
	tests := []struct {
		token, path string
		status      int
		want        string
	}{
		{"alice", "/api/v0/workspaces/acme", http.StatusOK, acmeJSON},
		{"bob", "/api/v0/workspaces/beta", http.StatusOK, betaJSON},
		{"bob", "/api/v0/workspaces/acme", http.StatusNotFound,
			`{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist."}`},
	}
	for _, tt := range tests {
		res, body := do(t, h, request(http.MethodGet, tt.path, tt.token, ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s GET %s = %d %s, want %d %s", tt.token, tt.path, res.StatusCode, body, tt.status, tt.want)
		}
	}
	if want := []string{"alice acme", "bob beta", "bob acme"}; !slices.Equal(get.calls, want) {
		t.Errorf("calls = %q, want %q", get.calls, want)
	}
}

// The availability, and the reason when the slug is not available; the slug
// of the path arrives unescaped.
func TestCheckWorkspaceSlug(t *testing.T) {
	check := &fakeCheck{reasons: map[string]domain.SlugReason{
		"acme": domain.SlugTaken, "settings": domain.SlugReserved, "my team": domain.SlugInvalid,
	}}
	h := newServer(t, fakes{check: check})
	tests := []struct{ path, want string }{
		{"/api/v0/workspace-slugs/free", `{"available":true}`},
		{"/api/v0/workspace-slugs/acme", `{"available":false,"reason":"taken"}`},
		{"/api/v0/workspace-slugs/settings", `{"available":false,"reason":"reserved"}`},
		{"/api/v0/workspace-slugs/my%20team", `{"available":false,"reason":"invalid"}`},
	}
	for _, tt := range tests {
		res, body := do(t, h, request(http.MethodGet, tt.path, "alice", ""))
		if res.StatusCode != http.StatusOK || body != tt.want+"\n" {
			t.Errorf("GET %s = %d %s, want 200 %s", tt.path, res.StatusCode, body, tt.want)
		}
	}
	if want := []string{"alice free", "alice acme", "alice settings", "alice my team"}; !slices.Equal(check.calls, want) {
		t.Errorf("calls = %q, want %q", check.calls, want)
	}
}
````

`server/internal/bootstrap/ports_test.go`（新文件，91 行）：

````file server/internal/bootstrap/ports_test.go
package bootstrap

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
)

// fakeIdentityAccounts answers the states it holds, by id and by address, and
// records what it was asked.
type fakeIdentityAccounts struct {
	states []identityapp.AccountState
	err    error
	asked  []string
}

func (f *fakeIdentityAccounts) ShareAccount(_ context.Context, id uuid.UUID) (identityapp.AccountState, bool, error) {
	f.asked = append(f.asked, "id "+id.String())
	i := slices.IndexFunc(f.states, func(s identityapp.AccountState) bool { return s.ID == id })
	if i < 0 {
		return identityapp.AccountState{}, false, f.err
	}
	return f.states[i], true, f.err
}

func (f *fakeIdentityAccounts) ShareAccountByEmail(_ context.Context, email string) (identityapp.AccountState, bool, error) {
	f.asked = append(f.asked, "email "+email)
	i := slices.IndexFunc(f.states, func(s identityapp.AccountState) bool { return s.Email == email })
	if i < 0 {
		return identityapp.AccountState{}, false, f.err
	}
	return f.states[i], true, f.err
}

// workspaceAccounts hands workspace identity's answer to the same question:
// the account asked for, its state converted, found and the error as they
// came.
func TestWorkspaceAccountsConvertsIdentitysAnswer(t *testing.T) {
	alice := identityapp.AccountState{ID: uuid.NewV7(), Email: "alice@corp.com", Active: true}
	bob := identityapp.AccountState{ID: uuid.NewV7(), Email: "bob@corp.com", Active: false}
	fake := &fakeIdentityAccounts{states: []identityapp.AccountState{alice, bob}}
	a := workspaceAccounts{accounts: fake}
	ctx := context.Background()

	type answer struct {
		state workspace.AccountState
		found bool
	}
	var got []answer
	for _, id := range []uuid.UUID{alice.ID, bob.ID, uuid.NewV7()} {
		s, found, err := a.ShareAccount(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, answer{s, found})
	}
	for _, email := range []string{"bob@corp.com", "carol@corp.com"} {
		s, found, err := a.ShareAccountByEmail(ctx, email)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, answer{s, found})
	}
	want := []answer{
		{workspace.AccountState{ID: alice.ID, Email: "alice@corp.com", Active: true}, true},
		{workspace.AccountState{ID: bob.ID, Email: "bob@corp.com", Active: false}, true},
		{},
		{workspace.AccountState{ID: bob.ID, Email: "bob@corp.com", Active: false}, true},
		{},
	}
	if !slices.Equal(got, want) {
		t.Errorf("answers = %+v, want %+v", got, want)
	}
	if len(fake.asked) != 5 || fake.asked[0] != "id "+alice.ID.String() || fake.asked[3] != "email bob@corp.com" {
		t.Errorf("identity was asked %q", fake.asked)
	}

	failure := errors.New("connection reset")
	fake.err = failure
	if _, _, err := a.ShareAccount(ctx, alice.ID); !errors.Is(err, failure) {
		t.Errorf("ShareAccount() = %v, want %v", err, failure)
	}
	if _, _, err := a.ShareAccountByEmail(ctx, "alice@corp.com"); !errors.Is(err, failure) {
		t.Errorf("ShareAccountByEmail() = %v, want %v", err, failure)
	}
}
````

`server/internal/bootstrap/workspace_test.go`（新文件，66 行）：

````file server/internal/bootstrap/workspace_test.go
package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The workspace module as bootstrap wires it (M3 design 6.6), with
// workspace.creation_enabled both ways (3.11). On: the caller creates a
// workspace, through identity's account lock, and is its admin and only
// member; his list holds it; he reads it, through the Authorizer, and
// another account reads it as not found. Off: the API answers
// workspace.creation_disabled and nothing is created.
func TestCreatingAWorkspaceAsConfigured(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("creation_enabled %v", enabled), func(t *testing.T) {
			contract := apitest.Load(t)
			cfg := testConfig(t, pgtest.NewDatabase(t), false)
			cfg.Workspace.CreationEnabled = enabled
			base := startApp(t, cfg, migrations.FS())
			alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
			bob := registerAccount(t, contract, base, "bob@example.com").AccessToken

			created, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice, `{"name":"Acme","slug":"acme"}`)
			_, list := call(t, contract, http.MethodGet, base+"/api/v0/workspaces", alice, "")
			aliceReads, _ := call(t, contract, http.MethodGet, base+"/api/v0/workspaces/acme", alice, "")
			bobReads, bobBody := call(t, contract, http.MethodGet, base+"/api/v0/workspaces/acme", bob, "")

			var w struct {
				Slug         string `json:"slug"`
				Role         int    `json:"role"`
				TotalMembers int    `json:"total_members"`
			}
			var l struct {
				Data []struct {
					Slug string `json:"slug"`
				} `json:"data"`
			}
			if err := json.Unmarshal([]byte(list), &l); err != nil {
				t.Fatalf("decode the list %s: %v", list, err)
			}
			if !enabled {
				if created != http.StatusForbidden || problemCode(t, []byte(body)) != "workspace.creation_disabled" ||
					len(l.Data) != 0 || aliceReads != http.StatusNotFound {
					t.Errorf("create = %d %s, list %s, read %d; want 403 workspace.creation_disabled, nothing created", created, body, list, aliceReads)
				}
				return
			}
			if err := json.Unmarshal([]byte(body), &w); err != nil || created != http.StatusCreated || w.Slug != "acme" || w.Role != 20 || w.TotalMembers != 1 {
				t.Errorf("create = %d %s, want 201 with the caller its admin and only member", created, body)
			}
			if len(l.Data) != 1 || l.Data[0].Slug != "acme" || aliceReads != http.StatusOK {
				t.Errorf("alice's list %s, her read %d; want acme, 200", list, aliceReads)
			}
			if bobReads != http.StatusNotFound || problemCode(t, []byte(bobBody)) != "workspace.not_found" {
				t.Errorf("bob's read = %d %s, want 404 workspace.not_found", bobReads, bobBody)
			}
		})
	}
}
````

- [ ] **Step 6: 测试、lint、前端检查**

Run: `go -C server test -count=1 ./internal/modules/workspace/... ./internal/bootstrap/`
Expected: 全部 `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过（文案表的键恰好等于契约的全部码）。

- [ ] **Step 7: 提交**

```bash
git add api/modules/workspace.yaml api/openapi.yaml server/internal/bootstrap/app.go server/internal/bootstrap/ports.go server/internal/bootstrap/ports_test.go server/internal/bootstrap/workspace_test.go server/internal/modules/workspace/adapter/http/gen/oapi-codegen.yaml server/internal/modules/workspace/adapter/http/handler.go server/internal/modules/workspace/adapter/http/handler_test.go server/internal/modules/workspace/adapter/http/workspaces.go server/internal/modules/workspace/adapter/http/workspaces_test.go server/internal/modules/workspace/module.go web/apps/web/helpers/authentication.helper.ts web/packages/i18n/src/locales/en/auth.json web/packages/i18n/src/locales/zh-CN/auth.json api/dist/openapi.yaml server/internal/modules/workspace/adapter/http/gen/bodyshape.gen.go server/internal/modules/workspace/adapter/http/gen/server.gen.go web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P1): the workspace API and the two-step composition

listWorkspaces, createWorkspace, getWorkspace and checkWorkspaceSlug, with
their HTTP side; bootstrap provides identity's Accounts and workspace's
WorkspaceRoles first, then builds access and the modules (M3 design 6.6),
workspace.creation_enabled wired both ways. The web message table gains the
three codes.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**Done when:** 四个操作经真实组合可用；开关的两种值都有测试看到；`apitest.Main` 在 `workspace` 的 HTTP 包里通过；前端检查通过。

---

### Task 12: 操作名的完整性；保留名单的"服务端"一段

**Files:**
- Create: `server/internal/bootstrap/actions_test.go`、`server/internal/bootstrap/reserved_test.go`
- Modify: `server/internal/modules/workspace/module.go`

**Interfaces:**
- Produces（spec 2.14，M3 设计 3.4、3.10、9.1、9.4）：`workspace.Actions() []shared.Action`、`workspace.ReservedSlugs() domain.ReservedSlugs`（模块入口，给组合根的测试）。
- 使用者：以后每个加操作名的 Phase 在 `moduleActions()` 中加一行。

**Tests:**（`server/internal/bootstrap`）
- `actions_test.go`：`TestEveryActionHasARuleAndEveryRuleAnAction`（各模块的 `Actions()` 的并集恰好是 `access.RuleKeys()`，没有重复，规则表不空）；`TestActionViolationsCatchesEachMismatch`（动作没有规则行、规则行不是任何模块的动作、一个动作声明两次、两个模块声明同一个动作，各报一条）。
- `reserved_test.go`：`TestTheReservedServerSlugsAreTheServersTopLevelPaths`：组合根注册的每个路由的第一段（前端页面的 `/` 除外），加上前端文件中"缺失的文件答 404 而不是页面"的顶层目录，恰好等于名单的 `[server]` 一段（`api`、`assets`、`healthz`、`readyz`）；`icons/` 缺失的文件答页面，不算。

- [ ] **Step 1: 模块入口的两个函数**

`server/internal/modules/workspace/module.go`（修改，2 处）：

````old server/internal/modules/workspace/module.go
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
````

````new server/internal/modules/workspace/module.go
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
````

````old server/internal/modules/workspace/module.go
	httpadapter.Register(router, api, m.uc)
}

````

````new server/internal/modules/workspace/module.go
	httpadapter.Register(router, api, m.uc)
}

// Actions lists the module's actions: bootstrap holds the union of every
// module's equal to access's rule table (M3 design 3.4).
func Actions() []shared.Action {
	return domain.Actions()
}

// ReservedSlugs is the reserved list (M3 design 3.10): bootstrap holds its
// server section equal to the top-level paths the server answers itself.
func ReservedSlugs() domain.ReservedSlugs {
	return domain.Reserved()
}

````

- [ ] **Step 2: 两个测试**

`server/internal/bootstrap/actions_test.go`（新文件，86 行）：

````file server/internal/bootstrap/actions_test.go
package bootstrap

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// moduleActions are the actions each module declares (M3 design 3.4). A
// module with actions adds its line here.
func moduleActions() map[string][]shared.Action {
	return map[string][]shared.Action{
		"workspace": workspace.Actions(),
	}
}

// actionViolations reports where the modules' actions and the rule table's
// keys part (M3 design 3.4): an action without a row, which the Authorizer
// would refuse at run time; a row without an action, which nothing asks
// for; an action declared twice, in one module or in two.
func actionViolations(modules map[string][]shared.Action, rules []shared.Action) []string {
	var found []string
	declared := map[shared.Action]string{}
	for _, module := range slices.Sorted(maps.Keys(modules)) {
		for _, action := range modules[module] {
			if other, ok := declared[action]; ok {
				found = append(found, fmt.Sprintf("action %q is declared by %s and by %s", action, other, module))
				continue
			}
			declared[action] = module
			if !slices.Contains(rules, action) {
				found = append(found, fmt.Sprintf("action %q of %s has no row in the rule table", action, module))
			}
		}
	}
	for _, rule := range rules {
		if _, ok := declared[rule]; !ok {
			found = append(found, fmt.Sprintf("the rule table's row %q is no module's action", rule))
		}
	}
	return found
}

// The union of the modules' Actions() is exactly the rule table's keys, and
// no action is declared twice (M3 design 3.4, 9.1).
func TestEveryActionHasARuleAndEveryRuleAnAction(t *testing.T) {
	if len(access.RuleKeys()) == 0 {
		t.Fatal("the rule table has no row")
	}
	for _, v := range actionViolations(moduleActions(), access.RuleKeys()) {
		t.Error(v)
	}
}

// Each check of actionViolations fails on its counterexample.
func TestActionViolationsCatchesEachMismatch(t *testing.T) {
	const read, update, create = shared.Action("workspace.read"), shared.Action("workspace.update"), shared.Action("project.create")
	if got := actionViolations(map[string][]shared.Action{"workspace": {read}, "project": {create}}, []shared.Action{create, read}); len(got) != 0 {
		t.Fatalf("a matching layout: %q, want none", got)
	}
	tests := []struct {
		name    string
		modules map[string][]shared.Action
		rules   []shared.Action
		want    string
	}{
		{"an action without a row", map[string][]shared.Action{"workspace": {read, update}}, []shared.Action{read},
			`action "workspace.update" of workspace has no row in the rule table`},
		{"a row without an action", map[string][]shared.Action{"workspace": {read}}, []shared.Action{read, update},
			`the rule table's row "workspace.update" is no module's action`},
		{"an action of two modules", map[string][]shared.Action{"workspace": {read}, "project": {read}}, []shared.Action{read},
			`action "workspace.read" is declared by project and by workspace`},
		{"an action twice in one module", map[string][]shared.Action{"workspace": {read, read}}, []shared.Action{read},
			`action "workspace.read" is declared by workspace and by workspace`},
	}
	for _, tt := range tests {
		if got := actionViolations(tt.modules, tt.rules); !slices.Equal(got, []string{tt.want}) {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
}
````

`server/internal/bootstrap/reserved_test.go`（新文件，66 行）：

````file server/internal/bootstrap/reserved_test.go
package bootstrap

import (
	"io/fs"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
)

// serverPaths are the top-level path segments the server answers itself,
// beside the web app's pages: the first segment of every route the
// composition root registers but the web UI's "/", and each top-level
// directory of the web files below which the web UI answers a missing file
// with 404 instead of the page.
func serverPaths(t *testing.T, a *app, webFiles fs.FS) []string {
	t.Helper()
	paths := map[string]bool{}
	for _, pattern := range a.router.Patterns() {
		path := pattern
		if _, rest, ok := strings.Cut(pattern, " "); ok {
			path = rest
		}
		if segment, _, _ := strings.Cut(strings.TrimPrefix(path, "/"), "/"); segment != "" {
			paths[segment] = true
		}
	}
	entries, err := fs.ReadDir(webFiles, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		rec := httptest.NewRecorder()
		a.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/"+e.Name()+"/nerve-reserved-probe.js", nil))
		if rec.Code == http.StatusNotFound {
			paths[e.Name()] = true
		}
	}
	return slices.Sorted(maps.Keys(paths))
}

// The reserved list's server section is exactly what the server answers
// itself (M3 design 3.10, 9.4): a route added without its name on the list,
// or a name the server does not answer, fails here. A workspace named api
// would have its pages under /api/, which the API answers.
func TestTheReservedServerSlugsAreTheServersTopLevelPaths(t *testing.T) {
	a := buildApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})
	got := serverPaths(t, a, testWebUI)
	if want := slices.Sorted(slices.Values(workspace.ReservedSlugs().Server)); !slices.Equal(got, want) {
		t.Errorf("the server answers the top-level paths %q; the reserved list's server section is %q", got, want)
	}
	// The probe tells the web UI's page from its 404: a directory whose
	// missing files get the page is the app's, not the server's.
	pages := fstest.MapFS{"index.html": {Data: []byte(testIndexHTML)}, "icons/logo.svg": {Data: []byte("<svg/>")}}
	if got := serverPaths(t, a, pages); slices.Contains(got, "icons") {
		t.Errorf("serverPaths = %q; icons/ is served the page, want it left out", got)
	}
}
````

- [ ] **Step 3: 测试、lint**

Run: `go -C server test -count=1 -run 'TestEveryActionHasARuleAndEveryRuleAnAction|TestActionViolationsCatchesEachMismatch|TestTheReservedServerSlugsAreTheServersTopLevelPaths' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/bootstrap/actions_test.go server/internal/bootstrap/reserved_test.go server/internal/modules/workspace/module.go
```
```bash
git commit -m "test(M3/P1): every action has a rule; the reserved server slugs are the server's paths

The modules' actions and the rule table must be the same set (M3 design
3.4); the reserved list's server section is what the composed server
answers itself (3.10).

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 3 个测试通过；删掉规则行、加一个未登记的动作、从名单删掉 `healthz` 都让它们失败（spec 附录 A）。

---

### Task 13: 建工作区与 M2 的停用：两个顺序

**Files:**
- Create: `server/internal/bootstrap/interleaving_test.go`

**Interfaces:**
- 没有新的生产代码。两边各跑真实的用例、真实的 Postgres 存储和数据库：`identity` 的停用（`app.Deactivate.ExecuteByEmail`，M2 停用的写入路径：`FOR NO KEY UPDATE` 锁账户行）和 `workspace` 的 `app.CreateWorkspace.Execute`。闸门由包装存储的测试替身放在持锁的事务里：停用在它的最后一次写入 `RevokeSessions` 之前（账户已在事务里停用、未提交），建工作区在它的第一次插入 `CreateWorkspace` 之前；`pgtest.WaitForLockWait` 证明另一方在等这一行之后才打开闸门；每个等待 10 秒为限。

**Tests:**（`interleaving_test.go`）
- `TestDeactivationFirstRefusesTheWorkspace`：停用持锁；建工作区等它的 `FOR SHARE`，锁下读到已停用，401；没有工作区；账户已停用。
- `TestCreationFirstHoldsOffTheDeactivation`：建工作区持 `FOR SHARE`；停用等它的 `FOR NO KEY UPDATE`，等待期间账户仍有效、没有提交的工作区；放开之后两边都成功：一个工作区，账户已停用。

- [ ] **Step 1: 交错测试**

`server/internal/bootstrap/interleaving_test.go`（新文件，228 行）：

````file server/internal/bootstrap/interleaving_test.go
package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	identitydomain "github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Creating a workspace against M2's deactivation of its admin's account, in
// both orders, on a real database (M3 design 3.6 convention 6; the
// interleaving 8 of 9.3 without the membership's end, which the
// deactivation adds with its port). Each side runs its real use case; a
// gate inside its transaction, after its lock of the account row, holds the
// transaction open, and pgtest.WaitForLockWait proves that the other side
// waits on that row before the gate opens. Every wait has a deadline.

// gate holds a transaction open: the first call of wait signals held and
// waits until open is closed.
type gate struct {
	held, open chan struct{}
}

func newGate() *gate { return &gate{held: make(chan struct{}), open: make(chan struct{})} }

func (g *gate) wait(ctx context.Context) error {
	close(g.held)
	select {
	case <-g.open:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// gatedSessions stops deactivation at its last write, holding the account
// row's lock.
type gatedSessions struct {
	identityapp.SessionRevoker
	gate *gate
}

func (s gatedSessions) RevokeSessions(ctx context.Context, userID, keep uuid.UUID, reason identitydomain.RevokeReason, now time.Time) (int, error) {
	if err := s.gate.wait(ctx); err != nil {
		return 0, err
	}
	return s.SessionRevoker.RevokeSessions(ctx, userID, keep, reason, now)
}

// gatedWorkspaces stops the creation before its first insert, holding the
// account row's lock.
type gatedWorkspaces struct {
	*workspacepg.Store
	gate *gate
}

func (w gatedWorkspaces) CreateWorkspace(ctx context.Context, row workspaceapp.WorkspaceRow) (domain.Workspace, error) {
	if err := w.gate.wait(ctx); err != nil {
		return domain.Workspace{}, err
	}
	return w.Store.CreateWorkspace(ctx, row)
}

type race struct {
	pool  *pgxpool.Pool
	alice uuid.UUID
}

// newRace is a database with alice's account.
func newRace(t *testing.T) race {
	t.Helper()
	pool := openPool(t, pgtest.NewDatabase(t))
	users := identitypg.New(pool)
	r := race{pool: pool, alice: uuid.NewV7()}
	now := time.Now()
	if err := users.CreateUser(context.Background(), identityapp.NewUser{
		ID: r.alice, Email: "alice@example.com", PasswordHash: "x", DisplayName: "alice", Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := users.CreateDefaultProfile(context.Background(), uuid.NewV7(), r.alice, now); err != nil {
		t.Fatal(err)
	}
	return r
}

// deactivation is M2's `nerve users deactivate` over sessions.
func (r race) deactivation(sessions identityapp.SessionRevoker) *identityapp.Deactivate {
	store := identitypg.New(r.pool)
	return identityapp.NewDeactivate(identityapp.DeactivateDeps{
		Accounts: store, Users: store, Profiles: store, Sessions: sessions,
		Tx: postgres.NewTxManager(r.pool, 2*time.Second), Clock: clocktest.At(time.Now()), Logger: slog.New(slog.DiscardHandler),
	})
}

// creation is createWorkspace over workspaces, with identity's Accounts as
// bootstrap wires it.
func (r race) creation(workspaces workspaceapp.WorkspaceCreator) *workspaceapp.CreateWorkspace {
	return workspaceapp.NewCreateWorkspace(workspaceapp.CreateWorkspaceDeps{
		Accounts:   workspaceAccounts{accounts: identity.Provide(r.pool).Accounts},
		Workspaces: workspaces,
		Tx:         postgres.NewTxManager(r.pool, 2*time.Second),
		Clock:      clocktest.At(time.Now()),
		Logger:     slog.New(slog.DiscardHandler),
		Enabled:    true,
	})
}

func (r race) state(t *testing.T) (active bool, workspaces int) {
	t.Helper()
	if err := r.pool.QueryRow(context.Background(), "SELECT is_active, (SELECT count(*) FROM workspaces) FROM users WHERE id = $1", r.alice).
		Scan(&active, &workspaces); err != nil {
		t.Fatal(err)
	}
	return active, workspaces
}

// run starts fn and returns the channel of its result.
func run(fn func() error) <-chan error {
	done := make(chan error, 1)
	go func() { done <- fn() }()
	return done
}

func held(t *testing.T, ctx context.Context, g *gate, done <-chan error, what string) {
	t.Helper()
	select {
	case <-g.held:
	case err := <-done:
		t.Fatalf("%s ended before its gate: %v", what, err)
	case <-ctx.Done():
		t.Fatalf("%s did not reach its gate within 10s", what)
	}
}

func result(t *testing.T, ctx context.Context, done <-chan error, what string) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		t.Fatalf("%s did not end within 10s", what)
		return nil
	}
}

// Deactivation first: it holds the account row; the creation waits on its
// FOR SHARE, then reads the account deactivated under the lock and answers
// 401. No workspace.
func TestDeactivationFirstRefusesTheWorkspace(t *testing.T) {
	r := newRace(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	g := newGate()
	deactivated := run(func() error {
		_, err := r.deactivation(gatedSessions{identitypg.New(r.pool), g}).ExecuteByEmail(ctx, "alice@example.com")
		return err
	})
	held(t, ctx, g, deactivated, "the deactivation")
	created := run(func() error {
		_, err := r.creation(workspacepg.New(r.pool)).Execute(shared.WithActor(ctx, shared.Actor{UserID: r.alice}),
			domain.NewWorkspace{Name: "Acme", Slug: "acme"})
		return err
	})
	pgtest.WaitForLockWait(t, r.pool, 5*time.Second)
	close(g.open)

	if err := result(t, ctx, deactivated, "the deactivation"); err != nil {
		t.Fatalf("the deactivation: %v", err)
	}
	if err := result(t, ctx, created, "the creation"); !errors.Is(err, shared.Unauthenticated()) {
		t.Errorf("the creation = %v, want 401 unauthorized", err)
	}
	if active, workspaces := r.state(t); active || workspaces != 0 {
		t.Errorf("alice active %v, %d workspaces; want deactivated and none", active, workspaces)
	}
}

// Creation first: it holds the account row FOR SHARE; the deactivation
// waits on its FOR NO KEY UPDATE until the workspace is committed, then
// deactivates the account. Both succeed.
func TestCreationFirstHoldsOffTheDeactivation(t *testing.T) {
	r := newRace(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	g := newGate()
	created := run(func() error {
		_, err := r.creation(gatedWorkspaces{workspacepg.New(r.pool), g}).Execute(shared.WithActor(ctx, shared.Actor{UserID: r.alice}),
			domain.NewWorkspace{Name: "Acme", Slug: "acme"})
		return err
	})
	held(t, ctx, g, created, "the creation")
	deactivated := run(func() error {
		_, err := r.deactivation(identitypg.New(r.pool)).ExecuteByEmail(ctx, "alice@example.com")
		return err
	})
	pgtest.WaitForLockWait(t, r.pool, 5*time.Second)
	if active, workspaces := r.state(t); !active || workspaces != 0 {
		t.Errorf("while the creation holds the row: alice active %v, %d workspaces; want active and none committed", active, workspaces)
	}
	close(g.open)

	if err := result(t, ctx, created, "the creation"); err != nil {
		t.Errorf("the creation = %v, want it done", err)
	}
	if err := result(t, ctx, deactivated, "the deactivation"); err != nil {
		t.Errorf("the deactivation = %v, want it done", err)
	}
	if active, workspaces := r.state(t); active || workspaces != 1 {
		t.Errorf("alice active %v, %d workspaces; want deactivated after one workspace", active, workspaces)
	}
}
````

- [ ] **Step 2: 测试、lint**

Run: `go -C server test -count=5 -race -run 'TestDeactivationFirstRefusesTheWorkspace|TestCreationFirstHoldsOffTheDeactivation' ./internal/bootstrap/`
Expected: `ok`。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 3: 提交**

```bash
git add server/internal/bootstrap/interleaving_test.go
```
```bash
git commit -m "test(M3/P1): creating a workspace against deactivation, both orders

On a real database: deactivation first, the creation waits on FOR SHARE
and is refused; creation first, the deactivation waits until the
workspace is committed (M3 design 3.6 convention 6).

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 两个测试通过，`-count=5 -race` 也通过；去掉 `FOR SHARE`、把锁挪到事务之外、在锁之前读 `is_active` 都让它们失败（spec 附录 A）。

---

### Task 14: `nerve workspaces create`

**Files:**
- Create: `server/cmd/nerve/workspaces.go`、`server/cmd/nerve/workspaces_test.go`、`server/internal/bootstrap/workspaces.go`、`server/internal/bootstrap/workspaces_test.go`、`server/internal/modules/workspace/admin.go`
- Modify: `README.md`、`api/modules/workspace.yaml`、`server/cmd/nerve/commands.go`、`server/configs/config.yaml`、`server/internal/archtest/composition_test.go`、`server/internal/bootstrap/commands.go`、`server/internal/bootstrap/users.go`、`server/internal/platform/config/config.go`
- Generate: `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`

**Interfaces:**
- Produces（spec 2.16，M3 设计 3.11、6.6）：
  - `workspace.AdminDeps{Pool, Tx, Clock, Logger, Accounts}`、`workspace.NewAdmin(AdminDeps) *Admin`、`(*Admin).CreateWorkspace(ctx, slug, name, adminEmail) (domain.Workspace, error)`；
  - `bootstrap.WorkspaceCommand`、`bootstrap.Workspaces(ctx, cfg, logOut, out, cmd) error`、`bootstrap.CreateWorkspace(slug, name, adminEmail) WorkspaceCommand`（输出 `created workspace <slug> with admin <规范化的地址>`）；
  - `bootstrap/commands.go` 收下 `cliFieldName`（加 `slug` → `--slug`、`name` → `--name`）和 `commandError`，两个命令共用；
  - `nerve workspaces create --slug <slug> --name <name> --admin-email <address>`（三个标志都必填；`nerve workspaces` 打印帮助）；
  - `TestCommandsComposeNoServerAndNoJobs` 取代 `TestUsersComposeNoServerAndNoJobs`；
  - `createWorkspace` 的说明、`README.md` 部署一节、`config.yaml` 和 `config.go` 的注释写上这个命令。
- 使用者：Task 16 的 W10。

**Tests:**
- `server/internal/bootstrap/workspaces_test.go`：`TestWorkspacesCreate`（开关关闭；地址 `" Alice@Corp.COM "` 规范化；一行输出；日志 `msg="workspace created"`、`by=cli`；`workspaces`、`workspace_members` 各一行，值对；`river_job` 为空）；`TestWorkspacesCreateErrors`（slug 被占用、没有账户、已停用、保留的 slug（`--slug is reserved`）、名称和 slug 都不合规：没有输出、一行错误、数据库不变）。
- `server/cmd/nerve/workspaces_test.go`：`TestWorkspacesCreateCommand`（退出码、stdout、stderr 的最后一行；缺标志：`required flag(s) "admin-email" not set`、`required flag(s) "name", "slug" not set`；未知子命令）；`TestBareWorkspacesPrintsHelp`。
- `server/internal/archtest/composition_test.go`：`TestCommandsComposeNoServerAndNoJobs`：`Users`、`Workspaces` 的静态调用都到达各自的 `NewAdmin`，都到达不了任何模块根包的 `New`、`platform/httpserver`、`ratelimit`、`jobs`、River。

- [ ] **Step 1: `NewAdmin` 和命令行的组合**

`server/internal/modules/workspace/admin.go`（新文件，46 行）：

````file server/internal/modules/workspace/admin.go
package workspace

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AdminDeps are what the server administrator's commands need: the pool and
// identity's Accounts, and no Authorizer, signing key or jobs client (M3
// design 6.6).
type AdminDeps struct {
	Pool     *pgxpool.Pool
	Tx       shared.TxManager
	Clock    app.Clock
	Logger   *slog.Logger
	Accounts app.Accounts
}

// Admin is the server administrator's use cases behind `nerve workspaces`
// (M3 design 3.11).
type Admin struct {
	create *app.CreateWorkspace
}

// NewAdmin wires the administrator's use cases on the pool alone: the
// command line builds no HTTP server, so it is the module's own minimal
// composition, not New's.
func NewAdmin(d AdminDeps) *Admin {
	return &Admin{create: app.NewCreateWorkspace(app.CreateWorkspaceDeps{
		Accounts: d.Accounts, Workspaces: postgresadapter.New(d.Pool), Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
	})}
}

// CreateWorkspace is `nerve workspaces create`: the workspace named name
// with slug, with the account of adminEmail as its admin, whatever
// workspace.creation_enabled says.
func (a *Admin) CreateWorkspace(ctx context.Context, slug, name, adminEmail string) (domain.Workspace, error) {
	return a.create.ExecuteForAdmin(ctx, adminEmail, domain.NewWorkspace{Name: name, Slug: slug})
}
````

`server/internal/bootstrap/workspaces.go`（新文件，55 行）：

````file server/internal/bootstrap/workspaces.go
package bootstrap

import (
	"context"
	"io"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/logging"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// WorkspaceCommand is one `nerve workspaces` command on the administrator's
// use cases: it runs and returns the line it prints.
type WorkspaceCommand func(ctx context.Context, admin *workspace.Admin) (string, error)

// Workspaces runs cmd on the minimal composition of M3 design 6.6: a pool,
// identity's Accounts and workspace's administrator use cases; no HTTP
// server, no Authorizer, no signing key, no jobs client. The command's line
// goes to out, the logs to logOut. An error is one line for the
// administrator, and the database is unchanged.
func Workspaces(ctx context.Context, cfg config.Config, logOut, out io.Writer, cmd WorkspaceCommand) error {
	logger, err := logging.New(logOut, cfg.Log)
	if err != nil {
		return err
	}
	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()
	admin := workspace.NewAdmin(workspace.AdminDeps{
		Pool:     pool,
		Tx:       postgres.NewTxManager(pool, cfg.Database.CommitTimeout),
		Clock:    clock.System{},
		Logger:   logger,
		Accounts: workspaceAccounts{accounts: identity.Provide(pool).Accounts},
	})
	line, err := cmd(ctx, admin)
	if err != nil {
		return commandError(err)
	}
	return writeLine(out, line)
}

// CreateWorkspace is `nerve workspaces create` (M3 design 3.11).
func CreateWorkspace(slug, name, adminEmail string) WorkspaceCommand {
	return func(ctx context.Context, admin *workspace.Admin) (string, error) {
		w, err := admin.CreateWorkspace(ctx, slug, name, adminEmail)
		return "created workspace " + w.Slug + " with admin " + shared.NormalizeEmail(adminEmail), err
	}
}
````

`server/internal/bootstrap/commands.go`（修改，3 处）：

````old server/internal/bootstrap/commands.go
	"log/slog"
````

````new server/internal/bootstrap/commands.go
	"log/slog"
	"strings"
````

````old server/internal/bootstrap/commands.go
	"github.com/open-nerve/NerveProject/server/internal/platform/webui"
````

````new server/internal/bootstrap/commands.go
	"github.com/open-nerve/NerveProject/server/internal/platform/webui"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````

````old server/internal/bootstrap/commands.go
	return err
}

````

````new server/internal/bootstrap/commands.go
	return err
}

// cliFieldName names a use case's field as the administrator's commands
// know it; a field it does not know keeps its own name.
func cliFieldName(field string) string {
	switch field {
	case "email":
		return "--email"
	case "new_email":
		return "--new-email"
	case "password":
		return "the password"
	case "slug":
		return "--slug"
	case "name":
		return "--name"
	default:
		return field
	}
}

// commandError is err as one line for the administrator: the invalid
// fields of a domain error, each as "<field> <problem>", or its detail.
func commandError(err error) error {
	var se *shared.Error
	if !errors.As(err, &se) || len(se.Fields) == 0 {
		return err
	}
	problems := make([]string, len(se.Fields))
	for i, f := range se.Fields {
		problems[i] = cliFieldName(f.Field) + " " + f.Message
	}
	return errors.New(strings.Join(problems, "; "))
}

````

`server/internal/bootstrap/users.go`（修改，4 处）：

````old server/internal/bootstrap/users.go
	"context"
	"errors"
````

````new server/internal/bootstrap/users.go
	"context"
````

````old server/internal/bootstrap/users.go
	"io"
	"strings"
````

````new server/internal/bootstrap/users.go
	"io"
````

````old server/internal/bootstrap/users.go
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
````

````new server/internal/bootstrap/users.go
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
````

````old server/internal/bootstrap/users.go

// cliFieldName names a use case's field as the command line knows it; a
// field it does not know keeps its own name.
func cliFieldName(field string) string {
	switch field {
	case "email":
		return "--email"
	case "new_email":
		return "--new-email"
	case "password":
		return "the password"
	default:
		return field
	}
}

// commandError is err as one line for the administrator: the invalid
// fields of a domain error, each as "<field> <problem>", or its detail.
func commandError(err error) error {
	var se *shared.Error
	if !errors.As(err, &se) || len(se.Fields) == 0 {
		return err
	}
	problems := make([]string, len(se.Fields))
	for i, f := range se.Fields {
		problems[i] = cliFieldName(f.Field) + " " + f.Message
	}
	return errors.New(strings.Join(problems, "; "))
}

````

````new server/internal/bootstrap/users.go

````

- [ ] **Step 2: 命令**

`server/cmd/nerve/workspaces.go`（新文件，46 行）：

````file server/cmd/nerve/workspaces.go
package main

import (
	"github.com/spf13/cobra"

	"github.com/open-nerve/NerveProject/server/internal/bootstrap"
)

// newWorkspacesCommand is `nerve workspaces`, the server administrator's
// commands on workspaces (M3 design 3.11).
func newWorkspacesCommand(load configLoader) *cobra.Command {
	workspaces := &cobra.Command{
		Use:   "workspaces",
		Short: "Manage workspaces as the server's administrator",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	workspaces.AddCommand(createWorkspaceCommand(load))
	return workspaces
}

// createWorkspaceCommand is `nerve workspaces create`: it works while
// workspace creation is switched off.
func createWorkspaceCommand(load configLoader) *cobra.Command {
	var slug, name, adminEmail string
	cmd := &cobra.Command{
		Use:   "create --slug <slug> --name <name> --admin-email <address>",
		Short: "Create a workspace with an account as its admin; works while workspace creation is switched off",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := load()
			if err != nil {
				return err
			}
			return bootstrap.Workspaces(cmd.Context(), cfg, cmd.ErrOrStderr(), cmd.OutOrStdout(),
				bootstrap.CreateWorkspace(slug, name, adminEmail))
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "the workspace's slug, its address")
	cmd.Flags().StringVar(&name, "name", "", "the workspace's name")
	cmd.Flags().StringVar(&adminEmail, "admin-email", "", "the e-mail address of the account that becomes its admin")
	_ = cmd.MarkFlagRequired("slug") // the flags exist
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("admin-email")
	return cmd
}
````

`server/cmd/nerve/commands.go`（修改，1 处）：

````old server/cmd/nerve/commands.go
	root.AddCommand(newServeCommand(load), newMigrateCommand(load), newUsersCommand(load, stdin), newVersionCommand())
````

````new server/cmd/nerve/commands.go
	root.AddCommand(newServeCommand(load), newMigrateCommand(load), newUsersCommand(load, stdin), newWorkspacesCommand(load),
		newVersionCommand())
````

- [ ] **Step 3: 说明、README、配置的注释**

`api/modules/workspace.yaml`（修改，1 处）：

````old api/modules/workspace.yaml
        workspace.creation_disabled. The name has 1–80
````

````new api/modules/workspace.yaml
        workspace.creation_disabled, and the server's administrator creates
        workspaces with `nerve workspaces create`. The name has 1–80
````

`README.md`（修改，1 处）：

````old README.md
  `set-email` 和 `activate` 都不是账户被盗后的恢复手段：怀疑账户被盗时，另外执行 `reset-password`。
````

````new README.md
  `set-email` 和 `activate` 都不是账户被盗后的恢复手段：怀疑账户被盗时，另外执行 `reset-password`。
- **建工作区**：`workspace.creation_enabled`（默认 `true`）决定用户能否经接口建工作区；关闭时 `POST /api/v0/workspaces` 答 403 `workspace.creation_disabled`，前端隐藏入口。这时由服务器管理员建：`nerve workspaces create --slug <slug> --name <名称> --admin-email <邮箱>`，不受这个开关限制，建出的工作区与接口建的相同，这个邮箱的账户是它的管理员和唯一成员。它与 `nerve users` 一样直接连数据库，服务不用停；邮箱按注册时的规则规范化。slug 已被占用或是保留名、名称不合规、邮箱没有账户、账户已停用时，退出码为 1，打印一行说明，数据库不变。
````

`server/configs/config.yaml`（修改，1 处）：

````old server/configs/config.yaml
  # 是否允许创建工作区。实例配置接口报告它；创建工作区在 M3 加入时照它执行
````

````new server/configs/config.yaml
  # 是否允许经接口创建工作区。关闭时 POST /api/v0/workspaces 答 403
  # workspace.creation_disabled，服务器管理员仍可用 nerve workspaces create
  # 建工作区；实例配置接口报告它，前端照它隐藏入口
````

`server/internal/platform/config/config.go`（修改，1 处）：

````old server/internal/platform/config/config.go
// clients; creating workspaces arrives, and honours it, in M3 (M2 design 5.3).
````

````new server/internal/platform/config/config.go
// clients; while CreationEnabled is off, POST /api/v0/workspaces answers
// workspace.creation_disabled and `nerve workspaces create` still creates
// (M3 design 3.11).
````

Run: `make gen`
Expected: 成功；只有说明文字改变的两个生成物：

| SHA-256 | 行数 | 文件 |
|---|---|---|
| `71dc05ee43e621e3ebe550a060c347da5ee6493dd822bae86ad2cc527058442f` | 1084 | `api/dist/openapi.yaml` |
| `54c7214da826dd41a05daa1862a21bbbcc7b1d5e82ad54f0cdfa58bc8f05268e` | 1134 | `web/packages/api-client/src/schema.gen.ts` |

Run: `shasum -a 256 api/dist/openapi.yaml web/packages/api-client/src/schema.gen.ts`
Expected: 与上表相同。

- [ ] **Step 4: 测试**

`server/internal/bootstrap/workspaces_test.go`（新文件，105 行）：

````file server/internal/bootstrap/workspaces_test.go
package bootstrap

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// runWorkspaces runs cmd on the database at url with workspace creation
// switched off, and returns its line, its logs and its error.
func runWorkspaces(t *testing.T, url string, cmd WorkspaceCommand) (out, logs string, err error) {
	t.Helper()
	cfg := testConfig(t, url, false)
	cfg.Log.Level = "info"
	cfg.Workspace.CreationEnabled = false
	var stdout, stderr bytes.Buffer
	err = Workspaces(context.Background(), cfg, &stderr, &stdout, cmd)
	return stdout.String(), stderr.String(), err
}

// workspaceRows are the workspaces and their members, one line each, in a
// fixed order.
func workspaceRows(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var rows string
	err := pool.QueryRow(context.Background(), `SELECT coalesce(string_agg(w.slug || ' ' || w.name || ' ' || w.timezone || ' by ' || c.email
			|| ': ' || u.email || ' ' || m.role || ' ' || m.is_active, E'\n' ORDER BY w.slug, u.email), '')
		FROM workspaces w JOIN users c ON c.id = w.created_by_id
		JOIN workspace_members m ON m.workspace_id = w.id JOIN users u ON u.id = m.member_id`).Scan(&rows)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// `nerve workspaces create` runs on the minimal composition while creation
// is switched off (M3 design 3.11, 6.6): the workspace, with the account of
// the address, normalized, as its only member and admin; one line; the log
// says the command line asked. Nothing is enqueued.
func TestWorkspacesCreate(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := openPool(t, url)
	createdAccount(t, url, pool, "alice@corp.com")
	createdAccount(t, url, pool, "bob@corp.com")

	out, logs, err := runWorkspaces(t, url, CreateWorkspace("acme", "Acme", " Alice@Corp.COM "))

	if err != nil || out != "created workspace acme with admin alice@corp.com\n" {
		t.Fatalf("create = %q, %v", out, err)
	}
	if !strings.Contains(logs, `msg="workspace created"`) || !strings.Contains(logs, "by=cli") {
		t.Errorf("logs = %s, want the creation logged by cli", logs)
	}
	if got, want := workspaceRows(t, pool), "acme Acme UTC by alice@corp.com: alice@corp.com 20 true"; got != want {
		t.Errorf("rows = %q, want %q", got, want)
	}
	var jobs int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM river_job").Scan(&jobs); err != nil || jobs != 0 {
		t.Errorf("river_job holds %d rows (%v), want none: creating a workspace enqueues nothing", jobs, err)
	}
}

// A refused command prints no line, says why in one line, and leaves the
// database as it was (M3 design 3.11).
func TestWorkspacesCreateErrors(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := openPool(t, url)
	createdAccount(t, url, pool, "alice@corp.com")
	createdAccount(t, url, pool, "dave@corp.com")
	if _, _, err := runUsers(t, url, DeactivateUser("dave@corp.com")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runWorkspaces(t, url, CreateWorkspace("acme", "Acme", "alice@corp.com")); err != nil {
		t.Fatal(err)
	}
	before := workspaceRows(t, pool)
	tests := []struct {
		name string
		cmd  WorkspaceCommand
		want string
	}{
		{"a taken slug", CreateWorkspace("acme", "Acme again", "alice@corp.com"), "A workspace with this slug exists."},
		{"an unknown account", CreateWorkspace("beta", "Beta", "nobody@corp.com"), "No account has this e-mail address."},
		{"a deactivated account", CreateWorkspace("beta", "Beta", "dave@corp.com"), "The account is deactivated."},
		{"a reserved slug", CreateWorkspace("api", "API", "alice@corp.com"), "--slug is reserved"},
		{"a bad name and slug", CreateWorkspace("Beta!", " ", "alice@corp.com"),
			"--name must contain a letter or a digit; --slug may hold only lower-case letters, digits, - and _"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := runWorkspaces(t, url, tt.cmd)
			if err == nil || err.Error() != tt.want || out != "" {
				t.Errorf("= %q, %v; want no line and %q", out, err, tt.want)
			}
		})
	}
	if got := workspaceRows(t, pool); got != before {
		t.Errorf("rows = %q, want them unchanged: %q", got, before)
	}
}
````

`server/cmd/nerve/workspaces_test.go`（新文件，68 行）：

````file server/cmd/nerve/workspaces_test.go
package main

import (
	"context"
	"strings"
	"testing"
)

// `nerve workspaces create` makes the account of the address the admin of
// a new workspace while creation is switched off, and prints one line; a
// refused one exits 1 with one line on stderr and nothing on stdout (M3
// design 3.11).
func TestWorkspacesCreateCommand(t *testing.T) {
	environ, pool := usersDatabase(t)
	environ = append(environ, "NERVE_WORKSPACE__CREATION_ENABLED=false")
	if code, _, stderr := executeWithInput(context.Background(), environ, "Tr0ub4dor&3\n", "users", "create", "--email", "nia@corp.com"); code != 0 {
		t.Fatalf("create the account = %d: %s", code, stderr)
	}

	code, stdout, stderr := execute(context.Background(), environ,
		"workspaces", "create", "--slug", "acme", "--name", "Acme", "--admin-email", "NIA@corp.com")

	if code != 0 || stdout != "created workspace acme with admin nia@corp.com\n" {
		t.Fatalf("nerve workspaces create = %d %q (stderr %q), want 0 and one line", code, stdout, stderr)
	}
	var role int
	if err := pool.QueryRow(context.Background(), `SELECT m.role FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
		JOIN users u ON u.id = m.member_id WHERE w.slug = 'acme' AND u.email = 'nia@corp.com'`).Scan(&role); err != nil || role != 20 {
		t.Errorf("nia's role in acme = %d (%v), want 20", role, err)
	}

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"a taken slug", []string{"workspaces", "create", "--slug", "acme", "--name", "Acme", "--admin-email", "nia@corp.com"},
			"nerve: A workspace with this slug exists.\n"},
		{"an unknown account", []string{"workspaces", "create", "--slug", "beta", "--name", "Beta", "--admin-email", "may@corp.com"},
			"nerve: No account has this e-mail address.\n"},
		{"a reserved slug", []string{"workspaces", "create", "--slug", "settings", "--name", "Beta", "--admin-email", "nia@corp.com"},
			"nerve: --slug is reserved\n"},
		{"no admin", []string{"workspaces", "create", "--slug", "beta", "--name", "Beta"}, "nerve: required flag(s) \"admin-email\" not set\n"},
		{"no slug and no name", []string{"workspaces", "create", "--admin-email", "nia@corp.com"},
			"nerve: required flag(s) \"name\", \"slug\" not set\n"},
		{"an unknown command", []string{"workspaces", "delete"}, "nerve: unknown command \"delete\" for \"nerve workspaces\"\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := execute(context.Background(), environ, tt.args...)
			if code != 1 || stdout != "" || !strings.HasSuffix(stderr, tt.want) {
				t.Errorf("nerve %s = %d, stdout %q, stderr %q; want 1 and %q", strings.Join(tt.args, " "), code, stdout, stderr, tt.want)
			}
		})
	}
	var workspaces int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM workspaces").Scan(&workspaces); err != nil || workspaces != 1 {
		t.Errorf("%d workspaces (%v), want acme alone", workspaces, err)
	}
}

func TestBareWorkspacesPrintsHelp(t *testing.T) {
	code, stdout, stderr := execute(context.Background(), nil, "workspaces")

	if code != 0 || !strings.Contains(stdout, "create") || stderr != "" {
		t.Errorf("nerve workspaces = %d, stdout %q, stderr %q; want 0 and the help", code, stdout, stderr)
	}
}
````

`server/internal/archtest/composition_test.go`（修改，6 处）：

````old server/internal/archtest/composition_test.go
// The command line's composition, bootstrap.Users, is a pool and identity's
// administrator use cases (M2 design 3.17): nothing it calls builds the HTTP
// side, a rate limiter or a jobs client. The rule follows the static calls
// from bootstrap.Users; the commands are func values it calls dynamically,
// so they are not followed: they only receive the composition. Reaching
// identity.NewAdmin shows the walk sees the composition at all.
func TestUsersComposeNoServerAndNoJobs(t *testing.T) {
````

````new server/internal/archtest/composition_test.go
// The command line's compositions, bootstrap.Users and bootstrap.Workspaces,
// are a pool and the modules' administrator use cases (M2 design 3.17, M3
// design 6.6): nothing they call builds a module's HTTP side or the
// Authorizer (a module's New), the HTTP server, a rate limiter or a jobs
// client. The rule follows the static calls from each; the commands are
// func values it calls dynamically, so they are not followed: they only
// receive the composition. Reaching the module's NewAdmin shows the walk
// sees the composition at all.
func TestCommandsComposeNoServerAndNoJobs(t *testing.T) {
````

````old server/internal/archtest/composition_test.go
	users := built[0].Func("Users")
	if users == nil {
		t.Fatal("bootstrap.Users not found: the rule checks nothing")
	}
	reached, banned := walkCalls(static.CallGraph(prog), users, composesMore)
	if !slices.ContainsFunc(reached, func(chain []*ssa.Function) bool {
		return chain[len(chain)-1].String() == m("internal/modules/identity")+".NewAdmin"
	}) {
		var names []string
		for _, chain := range reached {
			names = append(names, funcName(chain[len(chain)-1]))
````

````new server/internal/archtest/composition_test.go
	graph := static.CallGraph(prog)
	for _, c := range []struct{ root, admin string }{
		{"Users", m("internal/modules/identity") + ".NewAdmin"},
		{"Workspaces", m("internal/modules/workspace") + ".NewAdmin"},
	} {
		root := built[0].Func(c.root)
		if root == nil {
			t.Fatalf("bootstrap.%s not found: the rule checks nothing", c.root)
````

````old server/internal/archtest/composition_test.go
		t.Fatalf("bootstrap.Users does not reach identity.NewAdmin, so the rule checks nothing; it reaches:\n%s",
			strings.Join(names, "\n"))
	}
	for _, chain := range banned {
		names := make([]string, len(chain))
		for i, f := range chain {
			names[i] = funcName(f)
````

````new server/internal/archtest/composition_test.go
		reached, banned := walkCalls(graph, root, composesMore)
		if !slices.ContainsFunc(reached, func(chain []*ssa.Function) bool {
			return chain[len(chain)-1].String() == c.admin
		}) {
			var names []string
			for _, chain := range reached {
				names = append(names, funcName(chain[len(chain)-1]))
			}
			t.Errorf("bootstrap.%s does not reach %s, so the rule checks nothing; it reaches:\n%s",
				c.root, c.admin, strings.Join(names, "\n"))
````

````old server/internal/archtest/composition_test.go
		t.Errorf("bootstrap.Users builds more than the pool and NewAdmin: %s", strings.Join(names, " → "))
````

````new server/internal/archtest/composition_test.go
		for _, chain := range banned {
			names := make([]string, len(chain))
			for i, f := range chain {
				names[i] = funcName(f)
			}
			t.Errorf("bootstrap.%s builds more than the pool and the administrator use cases: %s", c.root, strings.Join(names, " → "))
		}
````

````old server/internal/archtest/composition_test.go
// composesMore reports whether f builds what the command line must not: the
// HTTP side (identity.New, platform/httpserver), a rate limiter or a jobs
// client (platform/jobs, River).
````

````new server/internal/archtest/composition_test.go
// composesMore reports whether f builds what the command line must not: a
// module's HTTP side or the Authorizer (New in a module's root package:
// identity.New, workspace.New, access.New and those to come), the HTTP
// server (platform/httpserver), a rate limiter or a jobs client
// (platform/jobs, River).
````

````old server/internal/archtest/composition_test.go
	if path == m("internal/modules/identity") && f.Name() == "New" {
````

````new server/internal/archtest/composition_test.go
	if module, ok := strings.CutPrefix(path, m("internal/modules")+"/"); ok && !strings.Contains(module, "/") && f.Name() == "New" {
````

- [ ] **Step 5: 测试、lint、前端检查**

Run: `go -C server test -count=1 -run 'TestWorkspacesCreate|TestBareWorkspacesPrintsHelp|TestCommandsComposeNoServerAndNoJobs' ./cmd/nerve/ ./internal/archtest/ ./internal/bootstrap/`
Expected: 三个 `ok`。

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
git add README.md api/modules/workspace.yaml server/cmd/nerve/commands.go server/cmd/nerve/workspaces.go server/cmd/nerve/workspaces_test.go server/configs/config.yaml server/internal/archtest/composition_test.go server/internal/bootstrap/commands.go server/internal/bootstrap/users.go server/internal/bootstrap/workspaces.go server/internal/bootstrap/workspaces_test.go server/internal/modules/workspace/admin.go server/internal/platform/config/config.go api/dist/openapi.yaml web/packages/api-client/src/schema.gen.ts
```
```bash
git commit -m "feat(M3/P1): nerve workspaces create

The server's administrator creates a workspace for the account of an
address, also while creation through the API is switched off (M3 design
3.11). The command composes the pool and workspace's administrator use
case only.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

Run（提交之后）: `make gen-check`
Expected: 通过。

**Done when:** 命令在真实数据库上输出一行并建对数据；错误是一行、退出码 1、数据库不变；命令行的组合里没有服务器、任务和 `Authorizer`。

---

### Task 15: 权限矩阵的骨架

**Files:**
- Create: `server/internal/bootstrap/permission_matrix_test.go`
- Modify: `server/internal/platform/httpserver/apitest/operations.go`、`server/internal/platform/httpserver/apitest/operations_test.go`、`server/internal/platform/postgres/pgtest/pgtest.go`、`server/internal/platform/postgres/pgtest/pgtest_test.go`

**Interfaces:**
- Produces（spec 2.17，M3 设计 9.2）：
  - `pgtest.NewDatabaseFrom(t, prepared string) string`：`CREATE DATABASE … TEMPLATE`，副本在测试结束时删除；复制时不能有连接连着 `prepared`；
  - `apitest.Operation` 加 `ID`（`operationId`）和 `Tags`；
  - `permission_matrix_test.go`：列、准备数据、`matrixRows()`、`matrixModules`、`matrixViolations`（spec 2.17）。以后每个 Phase 给自己的操作加行、在 `matrixModules` 加模块。
- 使用者：P2–P7 的矩阵行。

**Tests:**
- `pgtest_test.go`：`TestNewDatabaseFromCopiesThePreparedDatabase`（两个副本各自独立；写一个不影响另一个和源库）。
- `operations_test.go`：`TestOperations` 核对 `ID` 和 `Tags`。
- `permission_matrix_test.go`：`TestPermissionMatrix`（30 格；读的格子共用一个副本上的 app，写的 12 格各用一个副本、并行；每格断言状态码和 problem 的码，请求和响应都经契约核对）；`TestThePermissionMatrixCoversEveryOperation`；`TestMatrixViolationsCatchesEachGap`（缺行、行指向不存在的操作、行缺一列的格子）。

- [ ] **Step 1: 测试工具**

`server/internal/platform/postgres/pgtest/pgtest.go`（修改，2 处）：

````old server/internal/platform/postgres/pgtest/pgtest.go
	"net/url"
````

````new server/internal/platform/postgres/pgtest/pgtest.go
	"net/url"
	"strings"
````

````old server/internal/platform/postgres/pgtest/pgtest.go
	return newDatabase(t, "template0")
````

````new server/internal/platform/postgres/pgtest/pgtest.go
	return newDatabase(t, "template0")
}

// NewDatabaseFrom returns the URL of a new database copied from prepared, a
// database of this package that the test has filled, so that data prepared
// once serves every case that writes (M3 design 9.2). Nobody may be
// connected to prepared while it is copied: close its pools first. The copy
// is dropped when the test ends.
func NewDatabaseFrom(t testing.TB, prepared string) string {
	t.Helper()
	u, err := url.Parse(prepared)
	if err != nil {
		t.Fatalf("pgtest: %v", err)
	}
	return newDatabase(t, strings.TrimPrefix(u.Path, "/"))
````

`server/internal/platform/httpserver/apitest/operations.go`（修改，2 处）：

````old server/internal/platform/httpserver/apitest/operations.go
type Operation struct {
````

````new server/internal/platform/httpserver/apitest/operations.go
type Operation struct {
	ID     string // operationId
	Tags   []string
````

````old server/internal/platform/httpserver/apitest/operations.go
			o := Operation{Method: strings.ToUpper(method), Path: path, Public: !needsToken(op),
````

````new server/internal/platform/httpserver/apitest/operations.go
			o := Operation{ID: op.OperationID, Tags: op.Tags, Method: strings.ToUpper(method), Path: path, Public: !needsToken(op),
````

`server/internal/platform/postgres/pgtest/pgtest_test.go`（修改，1 处）：

````old server/internal/platform/postgres/pgtest/pgtest_test.go
		t.Error("a table created in one test database is visible in another")
````

````new server/internal/platform/postgres/pgtest/pgtest_test.go
		t.Error("a table created in one test database is visible in another")
	}
}

// A copy holds what the prepared database held, and each copy is its own:
// a write to one reaches neither the other copy nor the prepared database.
func TestNewDatabaseFromCopiesThePreparedDatabase(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prepared := pgtest.NewDatabase(t)
	conn, err := pgx.Connect(ctx, prepared)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	_, err = conn.Exec(ctx, "CREATE TABLE marks (name text); INSERT INTO marks VALUES ('prepared')")
	_ = conn.Close(ctx) // nobody may be connected to the database copied
	if err != nil {
		t.Fatal(err)
	}
	a := connect(t, pgtest.NewDatabaseFrom(t, prepared))
	b := connect(t, pgtest.NewDatabaseFrom(t, prepared))
	if _, err := a.Exec(ctx, "INSERT INTO marks VALUES ('a')"); err != nil {
		t.Fatal(err)
	}

	marks := func(conn *pgx.Conn) string {
		var names string
		if err := conn.QueryRow(ctx, "SELECT string_agg(name, ',' ORDER BY name) FROM marks").Scan(&names); err != nil {
			t.Fatal(err)
		}
		return names
	}
	if got := [3]string{marks(a), marks(b), marks(connect(t, prepared))}; got != [3]string{"a,prepared", "prepared", "prepared"} {
		t.Errorf("marks in copy a, copy b, the prepared database = %q; want the write in a alone", got)
````

`server/internal/platform/httpserver/apitest/operations_test.go`（修改，2 处）：

````old server/internal/platform/httpserver/apitest/operations_test.go
      operationId: getThing
````

````new server/internal/platform/httpserver/apitest/operations_test.go
      operationId: getThing
      tags: [things]
````

````old server/internal/platform/httpserver/apitest/operations_test.go
		t.Errorf("Operations() = %+v", ops)
````

````new server/internal/platform/httpserver/apitest/operations_test.go
		t.Errorf("Operations() = %+v", ops)
	}
	if ops[1].ID != "getThing" || !slices.Equal(ops[1].Tags, []string{"things"}) || ops[0].ID != "listThings" || ops[0].Tags != nil {
		t.Errorf("IDs and tags = %q %q, %q %q; want each operation's", ops[0].ID, ops[0].Tags, ops[1].ID, ops[1].Tags)
````

- [ ] **Step 2: 矩阵**

`server/internal/bootstrap/permission_matrix_test.go`（新文件，321 行）：

````file server/internal/bootstrap/permission_matrix_test.go
package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The permission matrix (M3 design 9.2): each operation of the modules
// below, called over HTTP on the wired app and a real database by each kind
// of caller, its status and problem code asserted cell by cell. The data is
// prepared once, through the modules' stores; the cells that only read
// share one copy of it, and each cell that writes gets a copy of its own
// (pgtest.NewDatabaseFrom), so no cell sees another's writes. A phase that
// adds an operation adds its row, and what the row needs prepared.

// matrixModules are the modules whose every operation has a row: their
// operations are authorized in a workspace, or act on the caller's
// workspaces. project adds itself with its operations.
var matrixModules = []string{"workspace"}

// caller is a column: an account, and how it stands to the workspace a row
// targets.
type caller string

const (
	callerAdmin   caller = "admin"
	callerMember  caller = "member"
	callerGuest   caller = "guest"
	callerNever   caller = "never a member"
	callerRemoved caller = "removed"
	callerDeleted caller = "workspace deleted"
)

// workspaceColumns are the columns of the workspace level, in the order of
// 9.2's table.
var workspaceColumns = []caller{callerAdmin, callerMember, callerGuest, callerNever, callerRemoved, callerDeleted}

// workspaceOf is the slug of the workspace a column's cells target: the
// prepared workspace, or the deleted one its caller was the admin of.
func workspaceOf(c caller) string {
	if c == callerDeleted {
		return "gone"
	}
	return "acme"
}

// cell is an answer: the status and, for a problem, its code.
type cell struct {
	status int
	code   string
}

var (
	cellOK                = cell{status: http.StatusOK}
	cellCreated           = cell{status: http.StatusCreated}
	cellWorkspaceNotFound = cell{http.StatusNotFound, "workspace.not_found"}
	cellCreationDisabled  = cell{http.StatusForbidden, "workspace.creation_disabled"}
)

// matrixRow is an operation's row: the request each caller sends and the
// answer each gets.
type matrixRow struct {
	op      string // operationId
	variant string // what sets the row apart from the operation's other rows
	write   bool   // each cell on a copy of its own
	config  func(*config.Config)
	request func(c caller) (method, path, body string)
	cells   map[caller]cell
}

func (r matrixRow) name() string {
	if r.variant == "" {
		return r.op
	}
	return r.op + ", " + r.variant
}

// every is the same answer in every workspace column.
func every(answer cell) map[caller]cell {
	cells := map[caller]cell{}
	for _, c := range workspaceColumns {
		cells[c] = answer
	}
	return cells
}

// sameRequest is the request of a row whose callers all send the same.
func sameRequest(method, path, body string) func(caller) (string, string, string) {
	return func(caller) (string, string, string) { return method, path, body }
}

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

// matrixData is the prepared database, the signing key every app on a copy
// of it shares, and each column's access token.
type matrixData struct {
	url     string
	keyFile string
	tokens  map[caller]string
}

// config is the configuration of an app on the database at url.
func (d matrixData) config(t *testing.T, url string, change func(*config.Config)) config.Config {
	t.Helper()
	cfg := testConfig(t, url, false)
	cfg.Auth.JWT.PrivateKeyFile = d.keyFile
	if change != nil {
		change(&cfg)
	}
	return cfg
}

// prepareMatrix fills a database for the matrix: an account for each
// column, registered through the API for its token; the workspace acme with
// its admin, member and guest, and the removed member's ended membership;
// the deleted workspace gone with its admin. Everything that connected to
// the database is closed when it returns, so that it can be copied.
func prepareMatrix(t *testing.T) matrixData {
	t.Helper()
	d := matrixData{url: pgtest.NewDatabase(t), keyFile: writeFile(t, testKeyPEM), tokens: map[caller]string{}}
	prepared := t.Run("prepare", func(t *testing.T) {
		contract := apitest.Load(t)
		base := startApp(t, d.config(t, d.url, nil), migrations.FS())
		pool := openPool(t, d.url)
		ids := map[caller]uuid.UUID{}
		for _, c := range workspaceColumns {
			email := strings.ReplaceAll(string(c), " ", "-") + "@example.com"
			d.tokens[c] = registerAccount(t, contract, base, email).AccessToken
			var id uuid.UUID
			if err := pool.QueryRow(context.Background(), "SELECT id FROM users WHERE email = $1", email).Scan(&id); err != nil {
				t.Fatal(err)
			}
			ids[c] = id
		}
		seed := matrixSeed{t: t, store: workspacepg.New(pool), ids: ids, now: time.Now()}
		acme := seed.workspace("acme", callerAdmin)
		seed.join(acme, callerAdmin, shared.RoleAdmin)
		seed.join(acme, callerMember, shared.RoleMember)
		seed.join(acme, callerGuest, shared.RoleGuest)
		seed.join(acme, callerRemoved, shared.RoleMember)
		gone := seed.workspace("gone", callerDeleted)
		seed.join(gone, callerDeleted, shared.RoleAdmin)
		// No store removes a member or deletes a workspace yet: SQL does
		// what they will, until the phases that add them.
		seed.exec(pool, "UPDATE workspace_members SET is_active = false WHERE member_id = $1", ids[callerRemoved])
		seed.exec(pool, "UPDATE workspaces SET deleted_at = now() WHERE id = $1", gone)
	})
	if !prepared {
		t.FailNow()
	}
	return d
}

// matrixSeed writes the prepared data through the modules' stores.
type matrixSeed struct {
	t     *testing.T
	store *workspacepg.Store
	ids   map[caller]uuid.UUID
	now   time.Time
}

func (s matrixSeed) workspace(slug string, admin caller) uuid.UUID {
	s.t.Helper()
	w, err := s.store.CreateWorkspace(context.Background(), workspaceapp.WorkspaceRow{
		ID: uuid.NewV7(), Name: slug, Slug: slug, Timezone: "UTC", CreatedBy: s.ids[admin], Now: s.now,
	})
	if err != nil {
		s.t.Fatal(err)
	}
	return w.ID
}

func (s matrixSeed) join(workspace uuid.UUID, c caller, role shared.Role) {
	s.t.Helper()
	if err := s.store.CreateMember(context.Background(), workspaceapp.MemberRow{
		ID: uuid.NewV7(), WorkspaceID: workspace, MemberID: s.ids[c], Role: role, CreatedBy: s.ids[c], Now: s.now,
	}); err != nil {
		s.t.Fatal(err)
	}
}

func (s matrixSeed) exec(pool *pgxpool.Pool, sql string, args ...any) {
	s.t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		s.t.Fatal(err)
	}
}

// Each cell of the matrix, the writing ones in parallel on their copies.
func TestPermissionMatrix(t *testing.T) {
	d := prepareMatrix(t)
	contract := apitest.Load(t)
	reads := startApp(t, d.config(t, pgtest.NewDatabaseFrom(t, d.url), nil), migrations.FS())
	for _, r := range matrixRows() {
		for _, c := range workspaceColumns {
			want, ok := r.cells[c]
			if !ok {
				continue // TestThePermissionMatrixCoversEveryOperation reports it
			}
			t.Run(r.name()+"/"+string(c), func(t *testing.T) {
				t.Parallel()
				base := reads
				if r.write || r.config != nil {
					base = startApp(t, d.config(t, pgtest.NewDatabaseFrom(t, d.url), r.config), migrations.FS())
				}
				method, path, body := r.request(c)
				status, answer := call(t, contract, method, base+path, d.tokens[c], body)
				got := cell{status: status}
				if status >= http.StatusBadRequest {
					got.code = problemCode(t, []byte(answer))
				}
				if got != want {
					t.Errorf("%s %s = %d %s, want %d %s", method, path, status, answer, want.status, want.code)
				}
			})
		}
	}
}

// matrixViolations reports where the matrix and the contract part: an
// operation of matrixModules without a row, a row that names no operation,
// a row without a cell for a column.
func matrixViolations(ops []apitest.Operation, rows []matrixRow) []string {
	var found []string
	inContract, inMatrix := map[string]bool{}, map[string]bool{}
	for _, r := range rows {
		inMatrix[r.op] = true
		for _, c := range workspaceColumns {
			if _, ok := r.cells[c]; !ok {
				found = append(found, fmt.Sprintf("row %s has no cell for %s", r.name(), c))
			}
		}
	}
	for _, op := range ops {
		inContract[op.ID] = true
		if slices.ContainsFunc(op.Tags, func(tag string) bool { return slices.Contains(matrixModules, tag) }) && !inMatrix[op.ID] {
			found = append(found, fmt.Sprintf("operation %s of %s has no row", op.ID, strings.Join(op.Tags, ", ")))
		}
	}
	for _, r := range rows {
		if !inContract[r.op] {
			found = append(found, fmt.Sprintf("row %s names no operation of the contract", r.name()))
		}
	}
	return found
}

// Every operation of the matrix's modules has a row, each row names an
// operation and has a cell for each column (M3 design 9.2): a new
// operation without a row fails here.
func TestThePermissionMatrixCoversEveryOperation(t *testing.T) {
	for _, v := range matrixViolations(apitest.Load(t).Operations(), matrixRows()) {
		t.Error(v)
	}
}

// Each check of matrixViolations fails on its counterexample.
func TestMatrixViolationsCatchesEachGap(t *testing.T) {
	ops := []apitest.Operation{{ID: "getWorkspace", Tags: []string{"workspace"}}, {ID: "getMe", Tags: []string{"identity"}}}
	row := matrixRow{op: "getWorkspace", cells: every(cellOK)}
	if got := matrixViolations(ops, []matrixRow{row}); len(got) != 0 {
		t.Fatalf("a matching matrix: %q, want none", got)
	}
	partial := row
	partial.cells = map[caller]cell{callerAdmin: cellOK}
	tests := []struct {
		name string
		ops  []apitest.Operation
		rows []matrixRow
		want []string
	}{
		{"an operation without a row", append(ops, apitest.Operation{ID: "updateWorkspace", Tags: []string{"workspace"}}),
			[]matrixRow{row}, []string{"operation updateWorkspace of workspace has no row"}},
		{"a row of no operation", ops, []matrixRow{row, {op: "renameWorkspace", cells: every(cellOK)}},
			[]string{"row renameWorkspace names no operation of the contract"}},
		{"a row without a cell", ops, []matrixRow{partial}, []string{
			"row getWorkspace has no cell for member", "row getWorkspace has no cell for guest", "row getWorkspace has no cell for never a member",
			"row getWorkspace has no cell for removed", "row getWorkspace has no cell for workspace deleted",
		}},
	}
	for _, tt := range tests {
		if got := matrixViolations(tt.ops, tt.rows); !slices.Equal(got, tt.want) {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
}
````

- [ ] **Step 3: 测试、耗时、lint**

Run: `go -C server test -count=1 ./internal/platform/postgres/pgtest/ ./internal/platform/httpserver/apitest/`
Expected: 两个 `ok`。

Run: `go -C server test -count=3 -race -run 'TestPermissionMatrix|TestThePermissionMatrixCoversEveryOperation|TestMatrixViolationsCatchesEachGap' ./internal/bootstrap/`
Expected: `ok`。

Run: `go -C server test -count=1 -v -run 'TestPermissionMatrix$' ./internal/bootstrap/`
Expected: `--- PASS: TestPermissionMatrix`；记下它和 `prepare` 子测试的耗时（原型和复现：整个矩阵 0.24–1.34 秒，准备 0.06–0.07 秒；M3 设计 9.2 的预算是全部 20–30 秒）。

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

- [ ] **Step 4: 提交**

```bash
git add server/internal/bootstrap/permission_matrix_test.go server/internal/platform/httpserver/apitest/operations.go server/internal/platform/httpserver/apitest/operations_test.go server/internal/platform/postgres/pgtest/pgtest.go server/internal/platform/postgres/pgtest/pgtest_test.go
```
```bash
git commit -m "test(M3/P1): the permission matrix and its first rows

Six callers against every operation of workspace, each cell's status and
code; the writing cells run in parallel on copies of a prepared database.
Every operation of the matrix's modules must have a row (M3 design 9.2).

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** 30 格通过，耗时记下；完整性核对通过；三个反例各被发现。

---

### Task 16: 端到端 W1、W10；上级文档

**Files:**
- Create: `e2e/fixtures/assert/workspace.ts`、`e2e/fixtures/workspaces.ts`、`e2e/stories/workspace/w1-create-workspace.spec.ts`、`e2e/stories/workspace/w10-admin-create-workspace.spec.ts`
- Modify: `docs/v0/M2-auth/M2-design.md`、`docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md`、`docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md`、`docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md`、`docs/v0/M3-workspace-project/handoffs/M2-closeout.md`、`docs/v0/plane-diff.md`、`docs/v0/v0-design.md`、`e2e/fixtures/api.ts`

**Interfaces:**
- Produces（spec 2.18、2.19）：
  - `e2e/fixtures/api.ts`：`Workspace`、`WorkspaceCreate` 类型；`slugFor(testInfo, label)`；`createWorkspace(api, token, body)`；
  - `e2e/fixtures/assert/workspace.ts`：`expectWorkspaceCreated(db, adminEmail, w) Promise<string>`、`countWorkspaces(db)`、`expectNoWorkspaceAdded(db, before)`；
  - `e2e/fixtures/workspaces.ts`：`nerveWorkspaces(db, args, env)`、`nerveWorkspacesFails(db, args, message)`；
  - 文档：M3 设计 3.20 中 P1 的各行（spec 2.19 的表）；四份交接追加"处理结果（M3/P1）"（spec 第 7 节：M2-closeout 第 4、5、8、12 节，M1-P2、M1-P3、M1-P4 的保留名单的服务端一侧），照 M2 各 Phase 的写法。
- 使用者：P8 起的页面版本的故事；P1 的 review。

**Tests:**
- `e2e/stories/workspace/w1-create-workspace.spec.ts`：`W1 (API): creating a workspace makes the caller its admin and only member; a taken or reserved slug, or creation switched off, adds nothing`。
- `e2e/stories/workspace/w10-admin-create-workspace.spec.ts`：`W10: nerve workspaces create makes the account of the address the workspace's admin, creation switched off or not; a taken slug, an unknown or a deactivated account adds nothing`。
- 此前的 50 个故事不改而通过。

- [ ] **Step 1: 夹具和断言**

`e2e/fixtures/api.ts`（修改，3 处）：

````old e2e/fixtures/api.ts
import { createClient } from "@nerve/api-client";
````

````new e2e/fixtures/api.ts
import { createHash } from "node:crypto";

import { createClient, type components } from "@nerve/api-client";
import { expect, type TestInfo } from "@playwright/test";

import { bearer } from "./auth";
````

````old e2e/fixtures/api.ts
export type Api = ReturnType<typeof createClient>;
````

````new e2e/fixtures/api.ts
export type Api = ReturnType<typeof createClient>;

export type Workspace = components["schemas"]["Workspace"];
export type WorkspaceCreate = components["schemas"]["WorkspaceCreate"];
````

````old e2e/fixtures/api.ts
}

````

````new e2e/fixtures/api.ts
}

/**
 * A slug of this run of this test, as emailFor gives an address: the tests
 * of a worker share its database, and --repeat-each runs a test again in
 * the same worker. It is label and 16 hexadecimal digits, well within the
 * 48 characters of a slug.
 */
export function slugFor(testInfo: TestInfo, label = "w"): string {
  const run = `${testInfo.testId}-${testInfo.repeatEachIndex}-${testInfo.retry}`;
  return `${label}-${createHash("sha256").update(run).digest("hex").slice(0, 16)}`;
}

/** Creates a workspace with the bearer token given, its caller the admin (M3 design 3.11), and returns it. */
export async function createWorkspace(api: Api, token: string, body: WorkspaceCreate): Promise<Workspace> {
  const { data, error, response } = await api.POST("/api/v0/workspaces", { body, headers: bearer(token) });
  expect(response.status, `create the workspace ${body.slug}: ${JSON.stringify(error)}`).toBe(201);
  if (!data) {
    throw new Error(`createWorkspace ${body.slug} answered 201 without the workspace`);
  }
  return data;
}

````

`e2e/fixtures/assert/workspace.ts`（新文件，85 行）：

````file e2e/fixtures/assert/workspace.ts
import { expect } from "@playwright/test";

import type { Database } from "../db";

// Database assertions of the workspace stories. The page version and the
// API version of a story call the same function (M3 design 2).

/** What a workspace was created with: the values it must hold. */
export interface NewWorkspace {
  name: string;
  slug: string;
  organization_size: string | null;
  timezone: string;
}

/**
 * W1, W10: the workspace of w.slug is new and holds w; the account of
 * adminEmail, a lowercased address, created it and is its only member, an
 * active admin (role 20), since the moment it was created. Returns the
 * workspace's id.
 */
export async function expectWorkspaceCreated(db: Database, adminEmail: string, w: NewWorkspace): Promise<string> {
  const admins = await db.query<{ id: string }>("SELECT id FROM users WHERE email = $1", [adminEmail]);
  expect(admins, `the account of ${adminEmail}`).toHaveLength(1);
  const admin = admins[0]?.id;
  const workspaces = await db.query<{ id: string; created_at: Date; updated_at: Date }>(
    `SELECT id, name, slug, organization_size, timezone, created_by_id, updated_by_id, created_at, updated_at, deleted_at
       FROM workspaces WHERE slug = $1`,
    [w.slug]
  );
  expect(workspaces, `the workspaces of slug ${w.slug}`).toEqual([
    {
      ...w,
      id: expect.any(String),
      created_by_id: admin,
      updated_by_id: admin,
      created_at: expect.any(Date),
      updated_at: workspaces[0]?.created_at,
      deleted_at: null,
    },
  ]);
  const id = workspaces[0]?.id as string;
  expect(
    await db.query(
      `SELECT member_id, role, is_active, created_by_id, updated_by_id, created_at, updated_at, deleted_at
         FROM workspace_members WHERE workspace_id = $1`,
      [id]
    ),
    `the members of ${w.slug}`
  ).toEqual([
    {
      member_id: admin,
      role: 20,
      is_active: true,
      created_by_id: admin,
      updated_by_id: admin,
      created_at: workspaces[0]?.created_at,
      updated_at: workspaces[0]?.created_at,
      deleted_at: null,
    },
  ]);
  return id;
}

/** How many workspaces and memberships there are. */
export interface WorkspaceCounts {
  workspaces: number;
  members: number;
}

export async function countWorkspaces(db: Database): Promise<WorkspaceCounts> {
  const [counts] = await db.query<WorkspaceCounts>(
    `SELECT (SELECT count(*)::int FROM workspaces) AS workspaces,
            (SELECT count(*)::int FROM workspace_members) AS members`
  );
  if (!counts) {
    throw new Error("the counts query returned no row");
  }
  return counts;
}

/** W1, W10: a refused creation added no workspace and no membership. */
export async function expectNoWorkspaceAdded(db: Database, before: WorkspaceCounts): Promise<void> {
  expect(await countWorkspaces(db)).toEqual(before);
}
````

`e2e/fixtures/workspaces.ts`（新文件，29 行）：

````file e2e/fixtures/workspaces.ts
import { expect } from "@playwright/test";

import type { Database } from "./db";
import { runNerve } from "./server";

// The server administrator's commands on workspaces, nerve workspaces (M3
// design 3.11), on a worker's database: the worker's nerve sees what they
// change on its next request.

/** Runs nerve workspaces with args, and the variables of env, and returns its output: the one line it prints. */
export async function nerveWorkspaces(db: Database, args: string[], env: Record<string, string> = {}): Promise<string> {
  return (await runNerve(["workspaces", ...args], db.url, "", env)).stdout;
}

/**
 * Runs nerve workspaces with args and expects it to fail: exit code 1, no
 * output, and "nerve: <message>" as the last line of stderr.
 */
export async function nerveWorkspacesFails(db: Database, args: string[], message: string): Promise<void> {
  const failure = await runNerve(["workspaces", ...args], db.url).then(
    () => undefined,
    (err: { code?: unknown; stdout?: string; stderr?: string }) => err
  );
  const label = `nerve workspaces ${args.join(" ")}`;
  expect(failure, `${label} fails`).toBeDefined();
  expect(failure?.code, label).toBe(1);
  expect(failure?.stdout, label).toBe("");
  expect(failure?.stderr?.endsWith(`nerve: ${message}\n`), `${label}: ${failure?.stderr}`).toBe(true);
}
````

- [ ] **Step 2: 两个故事**

`e2e/stories/workspace/w1-create-workspace.spec.ts`（新文件，59 行）：

````file e2e/stories/workspace/w1-create-workspace.spec.ts
import { createApi, createWorkspace, slugFor, type Api } from "../../fixtures/api";
import { countWorkspaces, expectNoWorkspaceAdded, expectWorkspaceCreated } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// W1, create a workspace (M3 design 2, 3.10, 3.11). The page version comes
// with the onboarding and /create-workspace pages.

/** The answer of GET /api/v0/workspace-slugs/{slug}. */
async function availability(api: Api, token: string, slug: string): Promise<unknown> {
  const { data, error, response } = await api.GET("/api/v0/workspace-slugs/{slug}", {
    params: { path: { slug } },
    headers: bearer(token),
  });
  expect(response.status, `check ${slug}: ${JSON.stringify(error)}`).toBe(200);
  return data;
}

test("W1 (API): creating a workspace makes the caller its admin and only member; a taken or reserved slug, or creation switched off, adds nothing", async ({
  api,
  db,
  nerveWith,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = (await createPAT(api, (await register(api, email)).access_token)).token;
  const slug = slugFor(testInfo);
  expect(await availability(api, pat, slug)).toEqual({ available: true });

  const body = { name: "Acme", slug, organization_size: "2-10", timezone: "Asia/Shanghai" } as const;
  const created = await createWorkspace(api, pat, body);

  expect(created).toMatchObject({ ...body, logo_url: null, role: 20, total_members: 1 });
  expect(await expectWorkspaceCreated(db, email, body)).toBe(created.id);
  expect(await availability(api, pat, slug)).toEqual({ available: false, reason: "taken" });
  expect(await availability(api, pat, "settings")).toEqual({ available: false, reason: "reserved" });

  const before = await countWorkspaces(db);
  const taken = await api.POST("/api/v0/workspaces", { body: { name: "Acme again", slug }, headers: bearer(pat) });
  expect(taken.response.status).toBe(409);
  expect(taken.error?.code).toBe("workspace.slug_taken");
  const reserved = await api.POST("/api/v0/workspaces", {
    body: { name: "Settings", slug: "settings" },
    headers: bearer(pat),
  });
  expect(reserved.response.status).toBe(422);
  expect(reserved.error?.errors?.map((e) => ({ field: e.field, code: e.code }))).toEqual([
    { field: "slug", code: "not_allowed" },
  ]);
  // A nerve with creation switched off, on the same database: the personal
  // access token works there too.
  const closed = createApi((await nerveWith({ NERVE_WORKSPACE__CREATION_ENABLED: "false" })).baseURL);
  const refused = await closed.POST("/api/v0/workspaces", {
    body: { name: "Beta", slug: slugFor(testInfo, "beta") },
    headers: bearer(pat),
  });
  expect(refused.response.status).toBe(403);
  expect(refused.error?.code).toBe("workspace.creation_disabled");
  await expectNoWorkspaceAdded(db, before);
});
````

`e2e/stories/workspace/w10-admin-create-workspace.spec.ts`（新文件，53 行）：

````file e2e/stories/workspace/w10-admin-create-workspace.spec.ts
import { slugFor } from "../../fixtures/api";
import { countWorkspaces, expectNoWorkspaceAdded, expectWorkspaceCreated } from "../../fixtures/assert/workspace";
import { bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";
import { nerveUsers } from "../../fixtures/users";
import { nerveWorkspaces, nerveWorkspacesFails } from "../../fixtures/workspaces";

// W10, the administrator creates a workspace (M3 design 2, 3.11): how
// workspaces come to be while creation is switched off. There is no page
// version.

test("W10: nerve workspaces create makes the account of the address the workspace's admin, creation switched off or not; a taken slug, an unknown or a deactivated account adds nothing", async ({
  api,
  db,
}, testInfo) => {
  const email = emailFor(testInfo);
  const pat = (await createPAT(api, (await register(api, email)).access_token)).token;
  const deactivated = emailFor(testInfo, "deactivated");
  await register(api, deactivated);
  await nerveUsers(db, ["deactivate", "--email", deactivated]);
  const slug = slugFor(testInfo);

  const line = await nerveWorkspaces(
    db,
    ["create", "--slug", slug, "--name", "Acme", "--admin-email", email.toUpperCase()],
    { NERVE_WORKSPACE__CREATION_ENABLED: "false" }
  );

  expect(line).toBe(`created workspace ${slug} with admin ${email}\n`);
  const id = await expectWorkspaceCreated(db, email, { name: "Acme", slug, organization_size: null, timezone: "UTC" });
  // The account sees it through the API, as its admin and only member.
  const { data, response } = await api.GET("/api/v0/workspaces", { headers: bearer(pat) });
  expect(response.status).toBe(200);
  expect(data?.data).toMatchObject([{ id, slug, name: "Acme", role: 20, total_members: 1 }]);

  const before = await countWorkspaces(db);
  await nerveWorkspacesFails(
    db,
    ["create", "--slug", slug, "--name", "Acme again", "--admin-email", email],
    "A workspace with this slug exists."
  );
  await nerveWorkspacesFails(
    db,
    ["create", "--slug", slugFor(testInfo, "beta"), "--name", "Beta", "--admin-email", emailFor(testInfo, "nobody")],
    "No account has this e-mail address."
  );
  await nerveWorkspacesFails(
    db,
    ["create", "--slug", slugFor(testInfo, "beta"), "--name", "Beta", "--admin-email", deactivated],
    "The account is deactivated."
  );
  await expectNoWorkspaceAdded(db, before);
});
````

- [ ] **Step 3: 上级文档**

`docs/v0/M2-auth/M2-design.md`（修改，3 处）：

````old docs/v0/M2-auth/M2-design.md
  | `Forbidden` | 403 | 模块码 |
````

````new docs/v0/M2-auth/M2-design.md
  | `Forbidden` | 403 | `forbidden`（M3 设计 3.4），或模块码 |
````

````old docs/v0/M2-auth/M2-design.md
    1. 写法：码的格式是 `^([a-z]+\.)?[a-z_]+$`；带前缀的码，前缀等于所在模块文件的名字；不带前缀的码必须是下面表中的平台码。
````

````new docs/v0/M2-auth/M2-design.md
    1. 写法：码的格式是 `^([a-z]+\.)?[a-z_]+$`；带前缀的码，前缀必须是 nerve 的一个模块（`api/modules/` 下有它的文件），不必是声明它的那个文件：码说明哪个领域拒绝了，同一个事实在各处用同一个码（M3 设计 11.7）；不带前缀的码必须是下面表中的平台码。
````

````old docs/v0/M2-auth/M2-design.md
  | `unauthorized` | 401 | 新增。认证中间件，以及 `RequireActor` |
````

````new docs/v0/M2-auth/M2-design.md
  | `unauthorized` | 401 | 新增。认证中间件，以及 `RequireActor` |
  | `forbidden` | 403 | M3 新增。`access` 的拒绝：调用者看得到目标，他的角色不允许这个操作（M3 设计 3.4、11.7） |
````

`docs/v0/v0-design.md`（修改，7 处）：

````old docs/v0/v0-design.md
- **分页**：用游标。请求带 `?limit=50&cursor=...`，响应格式为 `{ "data": [...], "next_cursor": "..." }`，最后一页的 `next_cursor` 是 `null`。游标是不透明的字符串：封套（版本号加载荷，base64url 编码）由 `internal/shared` 定义，载荷由各个列表按自己的排序定义，例如 PAT 列表的载荷是这一页最后一行的 `(created_at, id)`（M2 设计 3.12）。解不开、版本不认识、载荷不合这个列表的格式、不是 `EncodeCursor` 原样写出的游标，都是 400 `bad_request`；游标不签名，改成另一个合格的位置照样可用，它只决定从哪里接着读。`limit` 超出 1–100 是 422 `validation_failed`。
````

````new docs/v0/v0-design.md
- **分页**：用游标。请求带 `?limit=50&cursor=...`，响应格式为 `{ "data": [...], "next_cursor": "..." }`，最后一页的 `next_cursor` 是 `null`。游标是不透明的字符串：封套（版本号加载荷，base64url 编码）由 `internal/shared` 定义，载荷由各个列表按自己的排序定义，例如 PAT 列表的载荷是这一页最后一行的 `(created_at, id)`（M2 设计 3.12）。解不开、版本不认识、载荷不合这个列表的格式、不是 `EncodeCursor` 原样写出的游标，都是 400 `bad_request`；游标不签名，改成另一个合格的位置照样可用，它只决定从哪里接着读。`limit` 超出 1–100 是 422 `validation_failed`。
- **集合型的列表不分页**：页面要整个集合、大小由管理员的操作决定的列表（工作区、成员、邀请、项目、项目成员、状态、标签）一次返回全部，响应仍是 `{ "data": [...] }` 封套，没有 `limit`、`cursor`、`next_cursor`；每个列表写明顺序，同值时按 `id`（M3 设计 3.12）。以后要给其中一个分页，是接口的改动，调用方同时改。第一个是 M3/P1 的 `listWorkspaces`。
````

````old docs/v0/v0-design.md
- 平台自己的错误码不带模块前缀：`bad_request`（400）、`unauthorized`（401）、`not_found`（404）、`payload_too_large`（413）、`validation_failed`（422）、`rate_limited`（429，带 `Retry-After`）、`internal_error`（500）、`not_ready`（503，只用于 `/readyz`）、`server_busy`（503，带 `Retry-After`）；模块的错误码带模块前缀，例如 `identity.email_taken`（M2 设计 3.11）。
````

````new docs/v0/v0-design.md
- 平台自己的错误码不带模块前缀：`bad_request`（400）、`unauthorized`（401）、`forbidden`（403，`access` 的拒绝：看得到而角色不允许，M3 设计 3.4）、`not_found`（404）、`payload_too_large`（413）、`validation_failed`（422）、`rate_limited`（429，带 `Retry-After`）、`internal_error`（500）、`not_ready`（503，只用于 `/readyz`）、`server_busy`（503，带 `Retry-After`）；模块的错误码带模块前缀，例如 `identity.email_taken`（M2 设计 3.11）。
````

````old docs/v0/v0-design.md
                            领域错误与错误码、TxManager 端口、分页游标的封套；以后加入领域事件接口、
                            Authorizer 端口。平台不导入它；时钟等其余端口由使用方的 app 层声明
````

````new docs/v0/v0-design.md
                            领域错误与错误码、TxManager 端口、分页游标的封套、Authorizer 端口和它的
                            Role、Action、Target、Grant（M3 设计 3.4），两个以上模块共用的纯取值规则
                            （邮箱、网址、时区，M3 设计 3.13）；以后加入领域事件接口。操作名常量不在这里，
                            在各模块的 domain（actions.go）。平台不导入它；时钟等其余端口由使用方的 app 层声明
````

````old docs/v0/v0-design.md
  module.go          模块入口：New(依赖)；(*Module).Register(router, api) 把模块生成的路由挂到 bootstrap 的根路由上，
````

````new docs/v0/v0-design.md
  module.go          模块入口：Provide(pool) 返回给别的模块用的适配器（6.3 第 4 条）；New(依赖)；(*Module).Register(router, api) 把模块生成的路由挂到 bootstrap 的根路由上，
````

````old docs/v0/v0-design.md
4. **只在组合根接线**：禁止全局可变状态，禁止 `init()` 副作用，禁止服务定位器。所有依赖都通过构造函数显式传入。
````

````new docs/v0/v0-design.md
4. **只在组合根接线**：禁止全局可变状态，禁止 `init()` 副作用，禁止服务定位器。所有依赖都通过构造函数显式传入。
   - 模块之间的端口可以是双向的，所以每个模块分两段构造（M3 设计 6.6）：`Provide(pool)` 只返回只依赖连接池、给别的模块用的适配器，不建用例；`New(Deps)` 建出全部用例和 HTTP 的一侧，`Deps` 是它要的全部端口，构造之后不再登记、不再注入任何东西。组合根先调各模块的 `Provide`，再按依赖的顺序调 `New`，没有构造的环。
````

````old docs/v0/v0-design.md
- **集中定义**：规则表放在 `modules/access` 中，各模块通过 `shared` 里的 `Authorizer` 端口调用：
````

````new docs/v0/v0-design.md
- **集中定义**（M3 设计 3.4）：规则表放在 `modules/access/domain/rules.go`，以操作名为键，每行是级别（工作区级、项目级、看得到即可）和允许的角色；表里没有的操作一律拒绝。各模块通过 `shared` 里的 `Authorizer` 端口调用；操作名常量在各模块的 `domain/actions.go`，`bootstrap` 的测试核对各模块的操作名恰好是规则表的键：
````

````old docs/v0/v0-design.md
  "issue.update":  {Project: [Admin, Member], AllowCreator: true}
  "issue.archive": {Project: [Admin, Member]}
  "issue.delete":  {Project: [Admin],         AllowCreator: true}
  ```
````

````new docs/v0/v0-design.md
  "workspace.read": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}}
  ```
  "创建者本人可以修改或删除自己创建的对象"由 M4 随第一条用到它的规则在规则表加入（`AllowCreator`）。
- **看不到与不能做**：不是有效成员、看不到的项目，判定答 `shared.ErrNotVisible`，用例换成自己资源的 404；看得到而角色不允许，403 `forbidden`。
- **相对的规则在用例中**：改自己的角色、目标的角色更高、唯一管理员这类要比较两个人的规则，由用例用判定交回的 `Grant` 和读到的目标判断。
- **判定的时机**：写操作在事务中取得父行（工作区行、项目行）的锁之后判定；读操作不开事务，直接判定。每个请求都读角色，不缓存。
````

`docs/v0/plane-diff.md`（修改，3 处）：

````old docs/v0/plane-diff.md
| River 的表 | **新增**的基础设施表（M2/P3b，`00005_river_main_v2_to_v7.sql`）：`river_job`、`river_leader`（`UNLOGGED`）、`river_queue`、`river_notification`，枚举 `river_job_state`，函数 `river_job_state_in_bitmask`。内容是 River v0.47.0 主线第 2–7 版迁移的原样导出（`river migrate-get --line main --all --exclude-version 1`），不建 `river_migration`，版本由 goose 管理；表、约束和索引的名字随 River，不按二·全局的约定改 | 后台任务和定时任务改用 River，与业务数据同库，替代 Plane 的 Celery 和 Celery Beat（v0 总体设计 5.2、6.7；M2 设计 3.15） |
| `workspaces` | 删除旧的 `logo` URL 列 | 遗留列 |
| `workspace_members` | 删除 `view_props`、`default_props` | 遗留列 |
````

````new docs/v0/plane-diff.md
| River 的表 | **新增**的基础设施表（M2/P3b，`00005_river_main_v2_to_v7.sql`）：`river_job`、`river_leader`（`UNLOGGED`）、`river_queue`、`river_notification`，枚举 `river_job_state`，函数 `river_job_state_in_bitmask`。内容是 River v0.47.0 主线第 2–7 版迁移的原样导出（`river migrate-get --line main --all --exclude-version 1`），不建 `river_migration`，版本由 goose 管理；表、约束和索引的名字随 River，不按二·全局的约定改 | 后台任务和定时任务改用 River，与业务数据同库，替代 Plane 的 Celery 和 Celery Beat（v0 总体设计 5.2、6.7；M2 设计 3.15） |
| `workspaces` | 14 列保留 10 列（M3/P1，`00006_workspace_workspaces.sql`） | M3 设计 4.2 |
| `workspaces` | `name`：新加 `CHECK (name <> '')`；"1–80 个字符、至少一个字母或数字、不含网址"在领域层 | Plane 只在序列化器中检查 |
| `workspaces` | `slug`：新加 `CHECK (slug ~ '^[a-z0-9_-]+$')`，只有小写；全表唯一的 `workspace_slug_key` 改为部分唯一索引 `workspaces_slug_key ON (slug) WHERE deleted_at IS NULL`（不同于二·全局去掉的 `(…, deleted_at)` 一类：Plane 这里是全表唯一，删除工作区时把 slug 改名腾出它），删除不再改名 | 页面只收小写；部分唯一索引下删除之后 slug 可以重用（M3 设计 3.10） |
| `workspaces` | `organization_size`：新加 CHECK，取值是前端的六个选项（`Just myself`、`2-10`、`11-50`、`51-200`、`201-500`、`500+`） | Plane 的服务端不检查 |
| `workspaces` | `timezone`：新加 `DEFAULT 'UTC'`；取值在领域层按 IANA 名称校验 | 模型的默认值（M3 设计 3.13） |
| `workspaces` | `created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 二·全局 |
| `workspaces` | 删除旧的 `logo` URL 列 | 遗留列 |
| `workspaces` | 删除 `owner_id`、`background_color`；`logo_asset_id` 暂不建，由 M5 随文件存储加入 | `owner_id` 与 `created_by_id` 重复、没有读取者，它的 `CASCADE` 会随账户删掉工作区（M3 设计 3.15）；`background_color` 前端不读；在 M5 之前接口的 `logo_url` 是 `null` |
| `workspace_members` | 17 列保留 10 列（M3/P1，`00007_workspace_workspace_members.sql`） | M3 设计 4.3 |
| `workspace_members` | `workspace_id`、`member_id`：加上 `ON DELETE CASCADE`；`created_by_id`、`updated_by_id`：加上 `ON DELETE SET NULL` | 二·全局（模型的 `on_delete`） |
| `workspace_members` | `role`：`CHECK (role >= 0)` 收紧为 `CHECK (role IN (5, 15, 20))`，新加 `DEFAULT 5` | 只有三种角色（M3 设计 3.4） |
| `workspace_members` | `is_active`：新加 `DEFAULT true`；`created_at`、`updated_at`：新加 `DEFAULT now()`（兜底，见二·全局） | 模型的默认值 |
| `workspace_members` | 部分唯一索引 `workspace_members_workspace_id_member_id_key ON (workspace_id, member_id) WHERE deleted_at IS NULL` 照搬；新加 `workspace_members_member_id_idx ON (member_id) WHERE deleted_at IS NULL` 和不带条件的 `workspace_members_workspace_id_idx ON (workspace_id)` | 我的工作区按账户查；物理级联要不带条件的索引（M3 设计 4） |
| `workspace_members` | 删除 `view_props`、`default_props` | 遗留列 |
| `workspace_members` | 删除 `issue_props`、`company_role`、`getting_started_checklist`、`tips`、`explored_features` | 前端不读不写；公司角色随 M2 删掉的新手引导步骤没有了写入方；后三项是 Plane 已砍功能的状态 |
````

````old docs/v0/plane-diff.md
| 分页 | `每页条数:页码:是否上一页` 形式的偏移游标 | 不透明游标 |
````

````new docs/v0/plane-diff.md
| 分页 | `每页条数:页码:是否上一页` 形式的偏移游标 | 不透明游标 |
| 集合型的列表（工作区、成员、项目等） | 一次返回全部，响应是裸数组 | 同样一次返回全部、不分页，响应是 `{"data": [...]}` 封套，每个列表写明顺序（M3 设计 3.12） |
````

````old docs/v0/plane-diff.md
| 个人访问令牌的编辑 | `PATCH` 可以改名称和说明，响应中带令牌原文 | 不提供；令牌原文只在创建时返回一次 |
````

````new docs/v0/plane-diff.md
| 个人访问令牌的编辑 | `PATCH` 可以改名称和说明，响应中带令牌原文 | 不提供；令牌原文只在创建时返回一次 |
| 工作区的 slug | 服务端接受大写；删除工作区时把 slug 改成 `slug__<时间戳>` 腾出它；`PATCH` 能改 | 只有小写；部分唯一索引，删除时不改名；建好之后不能改（M3 设计 3.10） |
| 保留的工作区名 | 前后端各一份，服务端 45 个以上，含 Plane 的产品词 | 服务端一份（`workspace/domain/reserved_slugs.txt`）：本站用到的顶层路径段（应用的顶层路由段、`public/` 的顶层目录、服务端的 `api`、`assets`、`healthz`、`readyz`）加 4 个预留段（`admin`、`docs`、`help`、`static`）；前端的副本随 M3 的前端改造删除，改问 `GET /api/v0/workspace-slugs/{slug}`（M3 设计 3.10） |
| 建工作区之后 | 投递 `workspace_seed`：建一个名为 "Plane" 的机器人账户做管理员，再建演示项目、状态、标签和工作项 | 什么都不投递，没有演示数据（M3 设计 3.11） |
| 关闭创建工作区时 | 实例管理员在管理后台为自己建工作区 | 服务器管理员用 `nerve workspaces create --slug --name --admin-email` 建，不受开关限制，`--admin-email` 的账户是它的管理员（M3 设计 3.11） |
````

- [ ] **Step 4: 交接的处理结果**

`docs/v0/M3-workspace-project/handoffs/M2-closeout.md`（修改，1 处）：

````old docs/v0/M3-workspace-project/handoffs/M2-closeout.md
来源：[M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M3 的各行；[M2 收尾 spec](../../M2-auth/specs/closeout.md) 第 3 节。

````

````new docs/v0/M3-workspace-project/handoffs/M2-closeout.md
来源：[M2 设计](../../M2-auth/M2-design.md) 13.2 中接收者含 M3 的各行；[M2 收尾 spec](../../M2-auth/specs/closeout.md) 第 3 节。

## 处理结果（M3/P1）

- **第 4 节 `profiles.last_workspace_id` 的外键**（完成）：不补外键，理由写在 M3 设计 3.14；没有代码改动。
- **第 5 节 `workspace_creation_enabled` 的执行**（完成）：开关关闭时，`createWorkspace` 在做任何事之前答 403 `workspace.creation_disabled`（`server/internal/modules/workspace/app/create_workspace.go`），`api/modules/workspace.yaml` 声明这个码；用例测试、`workspace` 的 HTTP 测试、组合测试的两种开关值、权限矩阵中关闭开关的一行和 W1 都核对它。命令写进 M3 设计 3.11 并实现：`nerve workspaces create --slug --name --admin-email` 不看开关（W10）。
- **第 8 节 模块边界**（完成）：第 1 件选端口：`access` 在 `app/ports.go` 声明 `WorkspaceRoles`，由 `workspace` 的存储实现，`bootstrap` 接上（M3 设计 6.5、6.6），不是 `TestSQLCSchemaScope` 的例外。第 2 件：`TestSQLCSchemaScope` 认出带引号的名字和 `public.` 前缀、`ALTER TABLE`、`CREATE [UNIQUE] INDEX`、`CREATE TRIGGER`、`DROP TABLE` 和改名，每种都有反例（`server/internal/archtest/sqlc_cases_test.go`），去掉任何一种的变异都让它失败。第 3 件：`Authorizer` 在 `server/internal/shared/authorize.go` 声明，由 `access` 实现，`shared` 仍只依赖标准库。`ProjectAccess` 在 P4 照同一写法。
- **第 12 节 页大小的规则**：M3 的列表都是集合型的，不分页（M3 设计 3.12；P1 的 `listWorkspaces` 答 `{"data": [...]}`）。按关闭条件由 P1 的 review 写明，本节在 M3 收尾时原样写进 M4 的交接（M3 设计 13.2）。

仍未处理，状态保持 `open`：第 1–3、6、7、9–11、13、14 节，随 M3 设计 13.1 中各自的 Phase；第 12 节等 P1 的 review 和 M3 的收尾。

来源：[M3/P1 spec](../specs/P1-platform.md) 第 7 节。

````

`docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md`（修改，1 处）：

````old docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md
来源：[M1/P2 评审记录](../../M1-frontend-trim/reviews/P2-trim-content-review.md)第 7 节。

````

````new docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md
来源：[M1/P2 评审记录](../../M1-frontend-trim/reviews/P2-trim-content-review.md)第 7 节。

## 处理结果（M3/P1）

- **保留的工作区地址，服务端一侧**（完成）：服务端的保留名单是 `server/internal/modules/workspace/domain/reserved_slugs.txt`，分"应用""服务端""预留"三段（M3 设计 3.10），本文件列出的已去掉的词都不在其中。建工作区和查 slug 都按它拒绝（422 `not_allowed`，`reason: reserved`）；"服务端"一段由 `bootstrap` 的 Go 测试核对。

仍未处理，状态保持 `open`：保留名单的前端一侧（`RESTRICTED_URLS` 删除，前后端一份，有测试核对，P8）；项目字段、侧边栏偏好、个人主页，随 M3 设计 13.1 中各自的 Phase。

来源：[M3/P1 spec](../specs/P1-platform.md) 第 7 节。

````

`docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md`（修改，1 处）：

````old docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md
来源：[M1/P3 评审记录](../../M1-frontend-trim/reviews/P3-trim-platform-review.md)第 7 节。

````

````new docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md
来源：[M1/P3 评审记录](../../M1-frontend-trim/reviews/P3-trim-platform-review.md)第 7 节。

## 处理结果（M3/P1）

- **保留的工作区地址，服务端一侧**（完成）：服务端的名单三段（M3 设计 3.10）；Plane 的产品词逐个有结论：都不保留，名单只收应用的顶层路由段和 `public/` 的顶层目录、服务端自己回答的顶层路径、四个预留的名字。`TestCheckSlug`（`server/internal/modules/workspace/domain/workspace_test.go`）核对产品词可以用作 slug。

仍未处理，状态保持 `open`：`RESTRICTED_URLS` 与后端同源（前端一侧，P8）；本文件的其余几条随 M3 设计 13.1 中各自的 Phase。

来源：[M3/P1 spec](../specs/P1-platform.md) 第 7 节。

````

`docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md`（修改，1 处）：

````old docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md
来源：[M1/P4 评审记录](../../M1-frontend-trim/reviews/P4-router-native-review.md)第 7 节。

````

````new docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md
来源：[M1/P4 评审记录](../../M1-frontend-trim/reviews/P4-router-native-review.md)第 7 节。

## 处理结果（M3/P1）

- **保留的工作区名，服务端一侧**（完成）：名单的"应用"一段是 `web/apps/web/app/routes/core.ts` 的顶层静态路由段加上 `web/apps/web/public/` 的顶层目录（M3 设计 3.10）；`login` 不是路由段，不保留（`TestCheckSlug`）。服务端自己回答的顶层路径是"服务端"一段，由 `bootstrap` 的 Go 测试核对。

仍未处理，状态保持 `open`：前后端用同一份、有测试核对"应用"一段与路由表一致（P8）；离开项目的顺序（P10）。

来源：[M3/P1 spec](../specs/P1-platform.md) 第 7 节。

````

- [ ] **Step 5: 检查和端到端**

Run: `make lint-go`
Expected: 两段都是 `0 issues.`

Run: `make test`
Expected: 全部 `ok`，没有 `FAIL`。

Run: `make lint-web`
Expected: 通过（关键词守卫也查文档）。

Run: `make knip`
Expected: 通过。

Run: `make test-web`
Expected: 通过。

Run: `make e2e`
Expected: 52 个全部通过（M2 的 50 个、W1、W10）。

- [ ] **Step 6: 提交**

```bash
git add docs/v0/M2-auth/M2-design.md docs/v0/M3-workspace-project/handoffs/M1-P2-trim-content.md docs/v0/M3-workspace-project/handoffs/M1-P3-trim-platform.md docs/v0/M3-workspace-project/handoffs/M1-P4-router-native.md docs/v0/M3-workspace-project/handoffs/M2-closeout.md docs/v0/plane-diff.md docs/v0/v0-design.md e2e/fixtures/api.ts e2e/fixtures/assert/workspace.ts e2e/fixtures/workspaces.ts e2e/stories/workspace/w1-create-workspace.spec.ts e2e/stories/workspace/w10-admin-create-workspace.spec.ts
```
```bash
git commit -m "test(M3/P1): W1 and W10 through the API; the P1 rows of the design docs

Creating a workspace through the API and through nerve workspaces create,
each refusal leaving the database as it was. v0-design, M2-design and
plane-diff carry P1's rows of M3 design 3.20; the handoffs P1 closes items
of record the results.

Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>"
```

**Done when:** `make e2e` 52 个全部通过；M3 设计 3.20 中 P1 的各行、8.7 中 P1 的一行（Task 14 的 README）都已写好；四份交接有"处理结果（M3/P1）"。
