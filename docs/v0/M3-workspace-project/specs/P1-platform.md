# M3/P1 权限框架、组合与建工作区：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M3/P1 `platform` |
| 日期 | 2026-09-29 |
| 状态 | 进行中（已按 pre-flight 修订） |
| 上级文档 | [M3 设计文档](../M3-design.md) 第 2（W1、W10）、3.3、3.4、3.6（约定一、二、六）、3.10–3.13、3.20（P1 各行）、4.1–4.3、4.11（P1 各行）、5.1–5.3、6.1、6.2、6.4–6.6、8.2、8.3、8.7（P1 一行）、9.1–9.4、9.6、11.4、11.6、11.7、12（P1）、13.1 节；[v0 总体设计](../../v0-design.md) 3.4、3.5、6.2、6.3、6.5 节；[M2 设计](../../M2-auth/M2-design.md) 3.11、3.13、3.14、3.17 节 |
| 前置交接 | [M2-closeout](../handoffs/M2-closeout.md) 第 4、5、8、12 节；[M1-P2](../handoffs/M1-P2-trim-content.md)、[M1-P3](../handoffs/M1-P3-trim-platform.md)、[M1-P4](../handoffs/M1-P4-router-native.md) 的保留名单（服务端一侧；本 Phase 的处理见第 7 节） |
| 计划 | [P1 plan](../plans/P1-platform.md) |

本 spec 只写 M3 设计交给 P1 决定的东西：名字、签名、SQL、测试名，以及原型证明了什么。规则本身以 M3 设计为准，这里引用节号，不重述。P1 是 M3 的第一个 Phase，依赖 M2 的收尾（`1f0e7ed7` 之前的 `main`）。

## 1. 目标

按 M3 设计 12 节 P1：权限框架、两段组合、矩阵测试的骨架定下；任何调用方都能建、列、看工作区，服务器管理员能用命令建工作区。具体是：

- 迁移 `00006`（`workspaces`）、`00007`（`workspace_members`）和它们的运行时权限；`TestSQLCSchemaScope` 认出别的模块的表上的四种写法（4.1）；
- `shared`：`authorize.go`（`Role`、`Action`、`Target`、`Grant`、`Authorizer`、`ErrNotVisible`）、平台码 `forbidden`、从 `identity/domain` 移来的三条取值规则（3.4、3.13）；`apitest` 的前缀规则按 11.7 修订；
- `access`：判定的纯函数和规则表（本 Phase 一行 `workspace.read`）、`WorkspaceRoles` 端口、`Authorizer`；
- `workspace`：领域规则、保留名单、操作名、错误；存储；`createWorkspace`（开关、账户行 `FOR SHARE`、锁下要求有效）、`listWorkspaces`、`getWorkspace`、`checkWorkspaceSlug`；接口描述和 HTTP 一侧；只用连接池的 `NewAdmin`；
- `identity`：`Provide(pool)` 交出 `Accounts`（锁账户行、交回状态）；
- `bootstrap`：两段组合（6.6）、`bootstrap/ports.go`、`nerve workspaces` 的组合；操作名的完整性、保留名单"服务端"一段、与 M2 停用的两个顺序的交错、权限矩阵的骨架和本 Phase 的行；
- `cmd/nerve`：`nerve workspaces create --slug --name --admin-email`；
- 端到端：W1、W10 的接口版本；3.20 中 P1 的各行、8.7 中 P1 的一行。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录。"生成"表示由 `make gen`（或 `make gen-go`）生成并提交，不手改。"Task"是 plan 中负责它的任务。

| 路径 | 内容 | Task |
|---|---|---|
| `server/migrations/sql/00006_workspace_workspaces.sql`、`00007_workspace_workspace_members.sql`；`server/migrations/schema_test.go`；`deploy/runtime-grants.sql` | 两张表；约束和索引的名字、CHECK 的反例；运行时权限 | 1 |
| `server/internal/archtest/sqlc_test.go`、`sqlc_cases_test.go` | 四种写法（带引号的名字、`CREATE [UNIQUE] INDEX`、`CREATE TRIGGER`、`DROP TABLE`）和改名 | 2 |
| `server/internal/shared/authorize.go`、`authorize_test.go`、`error.go`、`error_test.go` | 权限的值和端口；`CodeForbidden`、`Forbidden()` | 3 |
| `server/internal/platform/httpserver/apitest/problems.go`、`rules_test.go`、`rules_cases_test.go`；`api/common.yaml` | `forbidden` 是平台码；前缀是一个模块 | 3 |
| `api/dist/openapi.yaml`、`server/internal/platform/httpserver/apigen/components.gen.go`、`web/packages/api-client/src/schema.gen.ts`（生成） | `forbidden` 与 `Problem.code` 的说明 | 3 |
| `server/internal/shared/email.go`、`url.go`、`timezone.go` 及测试；`identity/domain/email.go`、`user.go`（修改）、`url.go`（删除）；`identity/app`、`identity/adapter/http` 的调用方 | 三条取值规则移到 `shared` | 4 |
| `server/internal/modules/access/domain/rules.go`、`decide.go` 及测试 | 规则表、判定 | 5 |
| `server/internal/modules/access/app/ports.go`、`authorizer.go` 及测试；`access/module.go` | `WorkspaceRoles`、`Authorizer`、`New`、`RuleKeys` | 6 |
| `server/internal/modules/workspace/domain/` | `Workspace`、校验、保留名单、`ActionRead`、错误 | 7 |
| `server/internal/modules/identity/adapter/postgres/queries/users.sql`、`accounts.go`、`accounts_test.go`；`identity/app/accounts.go`；`identity/provide.go` | `ShareAccount`、`ShareAccountByEmail`；`Provide` | 8 |
| `server/internal/modules/identity/adapter/postgres/gen/users.sql.go`（生成） | | 8 |
| `server/sqlc.yaml`；`server/internal/modules/workspace/app/ports.go`；`workspace/adapter/postgres/`（含 `queries/`） | `workspace` 的 sqlc 条目；端口；存储；`ActiveRole` | 9 |
| `server/internal/modules/workspace/adapter/postgres/gen/`（生成） | | 9 |
| `server/internal/modules/workspace/app/` | 四个用例 | 10 |
| `api/modules/workspace.yaml`、`api/openapi.yaml`；`workspace/adapter/http/`（`gen/oapi-codegen.yaml` 和测试）；`workspace/module.go`；`bootstrap/app.go`、`ports.go` 及测试；`web/apps/web/helpers/authentication.helper.ts`、两份 `auth.json` | 接口描述；HTTP 一侧；两段组合；三个新码的文案 | 11 |
| `api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`、`workspace/adapter/http/gen/*.gen.go`（生成） | | 11、14 |
| `server/internal/modules/workspace/module.go`（修改）；`bootstrap/actions_test.go`、`reserved_test.go` | `Actions()`、`ReservedSlugs()`；两个 bootstrap 测试 | 12 |
| `server/internal/bootstrap/interleaving_test.go` | 建工作区与 M2 的停用，两个顺序 | 13 |
| `server/internal/modules/workspace/admin.go`；`bootstrap/workspaces.go`、`commands.go`、`users.go`；`cmd/nerve/workspaces.go`、`commands.go`；`archtest/composition_test.go`；`README.md`；`server/configs/config.yaml`、`platform/config/config.go`（注释）；`api/modules/workspace.yaml`（说明） | `nerve workspaces create` | 14 |
| `server/internal/bootstrap/permission_matrix_test.go`；`platform/postgres/pgtest/pgtest.go`；`platform/httpserver/apitest/operations.go`、`rules_test.go` | 矩阵的骨架；从已准备的库复制；操作的 id 和 tags；每个操作的标签是它的模块 | 15 |
| `e2e/fixtures/api.ts`、`assert/workspace.ts`、`workspaces.ts`；`e2e/stories/workspace/w1-*.spec.ts`、`w10-*.spec.ts`；`docs/v0/v0-design.md`、`plane-diff.md`、`M2-auth/M2-design.md`；`docs/v0/M3-workspace-project/handoffs/` 的 `M2-closeout.md`、`M1-P2-trim-content.md`、`M1-P3-trim-platform.md`、`M1-P4-router-native.md` | W1、W10 的接口版本；3.20 的 P1 各行；交接的处理结果 | 16 |

### 2.2 依赖

没有新依赖（6.1）。`server/go.mod`、`server/tools/go.mod` 和两个 `go.sum` 不变，仍是 `go 1.27` / `toolchain go1.27.1`；不加 npm 包。

### 2.3 迁移（4.1–4.3）

```sql
-- 00006_workspace_workspaces.sql
CREATE TABLE workspaces (
    id uuid PRIMARY KEY,
    name varchar(80) NOT NULL CHECK (name <> ''),
    slug varchar(48) NOT NULL CHECK (slug ~ '^[a-z0-9_-]+$'),
    organization_size varchar(20)
        CHECK (organization_size IN ('Just myself', '2-10', '11-50', '51-200', '201-500', '500+')),
    timezone varchar(255) NOT NULL DEFAULT 'UTC',
    created_by_id uuid REFERENCES users ON DELETE SET NULL,
    updated_by_id uuid REFERENCES users ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);
CREATE UNIQUE INDEX workspaces_slug_key ON workspaces (slug) WHERE deleted_at IS NULL;
-- Down: DROP TABLE workspaces;

-- 00007_workspace_workspace_members.sql
CREATE TABLE workspace_members (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    member_id uuid NOT NULL REFERENCES users ON DELETE CASCADE,
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
CREATE INDEX workspace_members_member_id_idx ON workspace_members (member_id) WHERE deleted_at IS NULL;
CREATE INDEX workspace_members_workspace_id_idx ON workspace_members (workspace_id);
-- Down: DROP TABLE workspace_members;
```

- 两个文件的中文注释写明来源（Plane 的列数、保留几列）、M2 设计 3.13 的约定、留在领域层的规则和每个索引的用途（上面省略）。`deploy/runtime-grants.sql` 加一行 `GRANT SELECT, INSERT, UPDATE, DELETE ON workspaces, workspace_members TO nerve_runtime;`（`bootstrap/runtime_role_test.go` 的 `TestTheGrantsFileCoversEveryRelationAndFunction` 在建表的同一个 Task 就要求它）。
- `schema_test.go`：`TestMigrationsGoUpDownAndUpAgain` 数到 7 个迁移，两张表 up 之后在、down 之后不在；`TestConstraintAndIndexNames` 的查询覆盖两张新表，期望加 18 行（16 个名字，主键既是约束也是索引：`workspaces_{pkey, name_check, slug_check, organization_size_check, slug_key, created_by_id_fkey, updated_by_id_fkey}`，`workspace_members_{pkey, role_check, workspace_id_fkey, member_id_fkey, created_by_id_fkey, updated_by_id_fkey, workspace_id_member_id_key, member_id_idx, workspace_id_idx}`，每个外键带着它的 `ON DELETE`）；`TestChecksRejectCounterexamples` 加 7 个反例（空名称、大写的 slug、带点的 slug、空 slug、规模 `1000+`、角色 10、角色 0）和合法的插入。
- `server/sqlc.yaml` 的 `workspace` 条目在 Task 9 随第一批查询加入（第 3 节第 2 条）。

### 2.4 `TestSQLCSchemaScope` 的四种写法（4.1；M2 交接第 8 节第 2 件）

- 表名的写法 `tableName = (?:(?:public|"public")\.)?("[^"]+"|[a-z_][a-z0-9_]*)`，带引号的名字按原样、不带的转小写（`table()`）。建表之外，`ALTER TABLE … RENAME TO t` 也让 `t` 属于改名的模块。
- 作用于表的语句 `tableStatements`：`alters`（`ALTER TABLE [IF EXISTS] [ONLY] t`）、`indexes`（`CREATE [UNIQUE] INDEX [CONCURRENTLY] [IF NOT EXISTS] [名字] ON [ONLY] t`）、`puts a trigger on`（`CREATE [OR REPLACE] [CONSTRAINT] TRIGGER 名字 … ON [ONLY] t`）、`drops`（`DROP TABLE [IF EXISTS] t[, t2 …]`），整分支修复时加上 `drops a trigger on`（`DROP TRIGGER [IF EXISTS] 名字 ON t`）、`alters a trigger on`（`ALTER TRIGGER 名字 ON t`）。每种都报"which no migration creates"或"which module %s creates: the migration belongs to %s"。`REFERENCES` 不在其中：外键可以指向别的模块的表。`tableStatements` 的注释列出规则不查的写法（策略、`COMMENT ON`、`TRUNCATE`、`LIKE`/`INHERITS`/`PARTITION OF`、只写名字不写表的索引语句、数据语句、`DO` 块）。
- `sqlc_cases_test.go`：基准布局中 River 的迁移加上自己表上的触发器、改名再删除、不带名字的唯一索引（照它真实的迁移），`asset` 的迁移加上引用 `users` 的外键、自己表上的索引和删自己的表，这些都必须通过；`TestSQLCScopeReportsViolations` 加 14 个反例：带引号的 `ALTER TABLE`、`public` 下带引号的、别的模块的表上的 `CREATE INDEX CONCURRENTLY IF NOT EXISTS 名字`、不带名字的 `CREATE UNIQUE INDEX ON ONLY public.users`、`CREATE TRIGGER`（`ON users` 单独一行，规则去掉 `(?s)` 时失败）、`CREATE CONSTRAINT TRIGGER`、`DROP TABLE IF EXISTS assets, users CASCADE`、只在大小写上与 `users` 不同的带引号名字（`DROP TABLE "Users"`，没有迁移建它）、大写的不带引号名字（`ALTER TABLE USERS`）、同一个带引号的表建两次、改别的模块的表的名字、`DROP TRIGGER IF EXISTS … ON public.users CASCADE`、带引号且没有 `IF EXISTS` 的 `DROP TRIGGER`、`ALTER TRIGGER … ON users RENAME TO`（执行中的补充见评审记录第 3 节 T2-a、第 4 节）。

### 2.5 `shared`（3.4、3.13、11.4、11.7）

```go
type Role int
const (RoleGuest Role = 5; RoleMember Role = 15; RoleAdmin Role = 20)
type Action string
type Target struct{ WorkspaceID, ProjectID uuid.UUID }
type Grant struct{ WorkspaceRole, ProjectRole Role; ProjectAdmin bool }
type Authorizer interface {
	Authorize(ctx context.Context, actor Actor, action Action, t Target) (Grant, error)
}
var ErrNotVisible = &Error{Kind: KindNotFound, Detail: "The target is not visible to the caller."}
const CodeForbidden = "forbidden"
func Forbidden() *Error // KindForbidden, CodeForbidden, "Your role does not allow this."
```

- `ErrNotVisible` 没有码：用例用 `errors.Is` 认出它，换成自己资源的 404 码；它若漏到接口上，没有声明的码会让 `apitest` 失败。
- 取值规则（Task 4）：`NormalizeEmail`、`ValidEmail`、`MaxEmailLength`（`email.go`，Django 的正则照旧），`ContainsURL`（`url.go`），`ValidTimezone`（`timezone.go`：`time.LoadLocation`，`Local` 除外；二进制内嵌时区数据库，11.4）。`identity/domain/email.go` 只剩 `DisplayNameFromEmail`，`url.go` 删除，`user.go` 和六个调用方改调 `shared`；测试随规则移动（`TestNormalizeEmail`、`TestValidEmail`、`TestValidEmailLengthLimit`、`TestContainsURLAsPlane`、`TestValidTimezone`），内容不变。`shared` 的包说明加上"两个以上模块必须一致的纯取值规则"。
- 测试：`TestRolesArePlanes`（三个值）；`TestErrNotVisible`（包一层仍认得出；模块的 404 不被当作它）；`TestConstructors` 加 `Forbidden()` 一行。

### 2.6 `apitest` 的前缀规则（11.7）

- `platformCodes` 加 `forbidden`。`authoringViolations(doc, modules)` 的模块集合是 `api/modules/` 下的文件名（`moduleNames()`）；带前缀的码，前缀不是其中之一时报 `problem code %q is prefixed with %q, which is not a module: want one of %q`。
- `api/common.yaml` 的 `Problem.code` 说明写上 `forbidden` 和"前缀是拒绝它的模块，不必是声明它的操作所在的文件"。
- 测试：`TestModuleCodesPass`（`things.taken`、`stuff.not_found`、`forbidden`、`validation_failed` 在 `things` 的操作上都通过）；`TestAuthoringRulesReportViolations` 加"code of no module"（`nowhere.taken`）。

### 2.7 `access`（3.4、6.4、6.5）

- `domain/rules.go`：`Level`（`LevelWorkspace`、`LevelProject`、`LevelVisible`）、`Rule{Level; Roles []shared.Role}`、`rules`（本 Phase 一行：`"workspace.read": {LevelWorkspace, [Admin, Member, Guest]}`）、`RuleFor(action) (Rule, bool)`（交回这一行的副本，`Roles` 用 `slices.Clone`：调用方写它改不了表）、`RuleKeys()`（排序）。
- `domain/decide.go`：`Membership{Active bool; Role shared.Role}`、`Project{Public bool; Member Membership}`、`Facts{Workspace Membership; Project *Project}`、`Decide(rule, facts) (shared.Grant, error)`：
  - 工作区级：不是有效成员 → `ErrNotVisible`；角色不在集合中 → `Forbidden()`；否则 `Grant{WorkspaceRole}`；
  - 项目级、看得到即可：不是有效的工作区成员、没有项目、看不到 → `ErrNotVisible`；`sees` = 工作区管理员，或项目的有效成员，或项目公开而工作区角色是成员或管理员；看得到之后，工作区角色或有效的项目角色不是三个值之一（`knownRoles`）→ `Forbidden()`，两个级别都一样；看得到即可 → 允许；项目级 → 是项目的有效成员，并且项目角色在集合中或是工作区管理员 → 允许，否则 `Forbidden()`；`Grant` 带上两级角色和 `ProjectAdmin`；
  - 角色按集合比较，从不按大小：三个值以外的角色什么都不允许；级别不认识 → 错误（500）。
  - 项目级在 P1 没有规则行使用，判定已按 3.4 写全并由表测试覆盖（第 3 节第 9 条）。
- `app/ports.go`：`WorkspaceRoles.ActiveRole(ctx, workspaceID, userID) (role shared.Role, ok bool, err error)`。`app/authorizer.go`：`NewAuthorizer(roles)`；`Authorize` 先查规则，没有 → `fmt.Errorf("access: no rule for action %q")`，什么都不读；再每次调用都读角色（不缓存），端口的错误包一层返回；判定只用工作区的事实（`ProjectAccess` 随 P4 加入）。
- `module.go`：`Deps{WorkspaceRoles}`、`New(Deps) shared.Authorizer`、`RuleKeys()`。
- 测试：`TestDecideAtTheWorkspaceLevel`（四组角色 × 7 种身份：管理员、成员、访客、从来不是、已被移出、工作区已删除、角色 10；允许时 `Grant` 只带工作区角色）；`TestTheWorkspaceLevelIgnoresTheProject`；`TestDecideAtTheProjectLevels`（4 条规则 × 17 种身份，含 9.2 项目级的各列、"以前是公开项目的成员""被降为项目访客的工作区成员"和四个未知角色：公开项目上工作区角色 10、25 的非成员看不到，项目角色 10、工作区角色 10 的项目管理员在每个级别都是 403；另加没有项目的一行）；`TestRolesOutsideTheThreeAreAllowedNothing`（性质测试：三个值以外的每个区间和两端，在四个位置——不在项目中的调用者的工作区角色、项目管理员的工作区角色、工作区成员的项目角色、工作区管理员的项目角色——对每条规则、公开与否都不允许）；`TestTheGrantCarriesTheRoles`（每个放行的级别都核对 `Grant` 带的两个角色）；`TestARuleOfNoKnownLevelIsAnError`；`TestEveryRuleDecidesItsCells`（规则表的每一行对 9.2 的每种身份的答案写成 `tableCells`：删掉、放宽、收窄一行，或加一行而不写它的格子，都失败）；`TestRuleForReturnsACopy`（调用方改写交回的 `Roles` 之后再查，角色不变）；`TestAnActionWithoutARowHasNoRule`；`TestAuthorizeReadsTheCallersRoleInTheTargetsWorkspace`（两个用户 × 两个工作区，假端口按参数回答并记下调用和 ctx）；`TestAuthorizeReadsOnEveryCall`；`TestAuthorizeRefusesAnActionWithoutARule`（不是 `ErrNotVisible`、不是 403，端口没有被调用）；`TestAuthorizeReturnsThePortsError`。

### 2.8 `workspace` 的领域（3.10、3.11、5.2、5.3）

- `workspace.go`：`Workspace{ID, Name, Slug, OrganizationSize *string, Timezone, Role shared.Role, TotalMembers int, CreatedAt, UpdatedAt}`、`NewWorkspace{Name, Slug, OrganizationSize, Timezone *string}`、`DefaultTimezone = "UTC"`；`CheckSlug(slug) SlugReason`（`invalid`、`reserved`、""；`taken` 由存储回答）；`CheckNewWorkspace(w) error`：一次报出全部字段，422 `validation_failed`：
  - `name`：空 → `too_short` "must not be empty"；超过 80 个字符 → `too_long`；含 NUL → `invalid_format`；没有字母或数字 → `invalid_format` "must contain a letter or a digit"；含网址 → `contains_url`；
  - `slug`：空 → `too_short`；超过 48 → `too_long`；不合 `^[a-z0-9_-]+$` → `invalid_format` "may hold only lower-case letters, digits, - and _"；保留 → `not_allowed` "is reserved"；
  - `organization_size` 不是六个选项之一、`timezone` 不是时区 → `invalid_format`。
- `reserved_slugs.txt`（`embed`）：`[app]` `create-workspace`、`invitations`、`onboarding`、`settings`、`sign-up`、`workspace-invitations`、`icons`；`[server]` `api`、`assets`、`healthz`、`readyz`；`[reserved]` `admin`、`docs`、`help`、`static`（第 3 节第 3 条说明 `invitations`）。`reserved.go`：`ReservedSlugs{App, Server, Reserved}`、`All()`、`Reserved()`（返回副本）；文件解析不了时 nerve 启动即失败。
- `actions.go`：`ActionRead shared.Action = "workspace.read"`、`Actions()`。
- `errors.go`：`ErrNotFound`（404 `workspace.not_found`）、`ErrCreationDisabled`（403 `workspace.creation_disabled`）、`ErrSlugTaken`（409 `workspace.slug_taken`），以及只给命令行的 `ErrAccountNotFound`（`workspace.account_not_found`）、`ErrAccountDeactivated`（`workspace.account_deactivated`）（第 3 节第 7 条）。
- 测试：`TestCheckNewWorkspaceAcceptsValidWorkspaces`、`TestCheckNewWorkspaceReportsEveryField`、`TestCheckSlug`（`login` 不保留，3.10）；`TestTheReservedListIsWellFormed`（三段都有、互不重复、都合 slug 的写法，"预留"一段恰好是四个）、`TestParseReserved`（注释、空行、首尾空白、同一段出现两次时合并；段之前的名字是错误）、`TestReservedReturnsACopy`。

### 2.9 `identity` 交出 `Accounts`（3.6 约定一、六；6.5；6.6）

```sql
-- name: ShareAccount :one
SELECT id, email, is_active FROM users WHERE id = sqlc.arg(id) FOR SHARE;
-- name: ShareAccountByEmail :one
SELECT id, email, is_active FROM users WHERE email = sqlc.arg(email) FOR SHARE;
```

- `identity/app/accounts.go`：`AccountState{ID uuid.UUID; Email string; Active bool}`。存储的 `ShareAccount`、`ShareAccountByEmail` 返回 `(AccountState, found bool, error)`，`pgx.ErrNoRows` → `found = false`。
- `identity/provide.go`：`Accounts` 接口、`Provided{Accounts}`、`Provide(pool) Provided`（只用连接池，不建用例）。
- 测试：`TestShareAccountReadsTheState`（两个账户，一个已停用；两种查法各一个不存在的；地址按原样匹配）；`TestTheShareLockBlocksDeactivationNotAnotherShare`（两种查法各一个子测试：持 `FOR SHARE` 时，另一个事务的 `FOR NO KEY UPDATE`（M2 的凭证锁）在 `lock_timeout` 500 毫秒下得到 `55P03`，第二个 `FOR SHARE` 不等；每个等待 10 秒为限）。

### 2.10 `workspace` 的端口与存储（6.5、3.12）

- `app/ports.go`：`ErrNotFound`、`Clock`、`AccountState`、`Accounts{ShareAccount, ShareAccountByEmail}`、`WorkspaceRow`、`MemberRow`、`WorkspaceCreator{CreateWorkspace, CreateMember}`、`WorkspaceLister`、`WorkspaceFinder`、`SlugChecker`。
- 查询（`queries/workspaces.sql`、`members.sql`）：`CreateWorkspace`（`RETURNING` 存下的值，`created_by_id = updated_by_id`，两个时间都是用例的时钟）；`ListWorkspaces`（有效成员、行未删除、工作区未删除；带角色和有效成员数；`ORDER BY w.name, w.id`）；`WorkspaceBySlug`（未删除，带有效成员数）；`SlugTaken`（`EXISTS`，未删除）；`CreateMember`；`ActiveRole`（`m.is_active AND m.deleted_at IS NULL AND w.deleted_at IS NULL`）。
- 存储：`workspaces_slug_key` 的唯一冲突 → `domain.ErrSlugTaken`；其余错误（含 CHECK）是 500，领域已校验过。`roles.go` 的 `ActiveRole` 在 ctx 带的事务里读（写操作在父行的锁之后判定时读到的是事务内的状态）。
- 测试（真实数据库，账户用 SQL 建）：`TestCreateWorkspaceStoresTheRow`（审计列、UTC、微秒）、`TestCreateWorkspaceSlugTaken`（删除之后 slug 可以再用）、`TestCreateMemberStoresTheRow`、`TestListWorkspaces`（三个账户、七个工作区，`alice` 在每一种状态里：别人的、已删除的、被移出的、成员行已删除的都不在；顺序按名称再按 id；角色和人数）、`TestWorkspaceBySlug`、`TestSlugTaken`、`TestActiveRole`（夹具中的每一对，各自的状态）、`TestActiveRoleReadsInTheTransaction`。

### 2.11 用例（3.4、3.6、3.11、8.2）

- `CreateWorkspace`（`CreateWorkspaceDeps{Accounts, Workspaces, Tx, Clock, Logger, Enabled}`）：
  - `Execute(ctx, w)`：开关关闭 → `ErrCreationDisabled`，在一切之前（Plane `views/workspace/base.py:83-96`）；`RequireActor`；校验在事务之前；一个事务里：`ShareAccount(调用者)`，没有或锁下已停用 → 401 `unauthorized`；`CreateWorkspace`；`CreateMember`（角色 20）。
  - `ExecuteForAdmin(ctx, email, w)`：不看开关；邮箱按注册的规则规范化；同一个事务，锁是 `ShareAccountByEmail`：没有 → `ErrAccountNotFound`，锁下已停用 → `ErrAccountDeactivated`。
  - 两个入口共用 `create`：结果 `Role = 20`、`TotalMembers = 1`；成功后记 INFO "workspace created"（`workspace_id`、`user_id`、`by` = `api` 或 `cli`）；不投递任务、不建演示数据（3.11）。
- `ListWorkspaces`：调用者的列表，不经 `Authorizer`（账户级，6.4）。`GetWorkspace`：按 slug 读，没有 → `workspace.not_found`；`Authorize(ActionRead, Target{WorkspaceID})`，`ErrNotVisible` → `workspace.not_found`，别的错误原样；`Role` 取自 `Grant`；读不开事务（3.4）。`CheckSlug`：`CheckSlug` 不合格就不查库，否则问 `SlugTaken`。
- 测试（手写的假实现：假事务在 ctx 里做标记，存储和锁的每次调用记下参数和是否在事务里）：`TestExecuteCreatesTheWorkspaceWithTheCallerAsAdmin`（两个调用者；调用顺序和参数；一个事务）、`TestExecuteStoresTheTimeZoneGiven`、`TestExecuteWhileCreationIsDisabled`（值不合规也先答关闭，什么都不调）、`TestExecuteRefusesInvalidValuesBeforeTheTransaction`、`TestExecuteRefusesAnAccountNoLongerActive`（已停用、不存在：401，没有写入）、`TestExecuteSlugTaken`、`TestExecuteReturnsTheLocksError`、`TestExecuteWithoutACaller`、`TestExecuteForAdminCreatesForTheAccountOfTheAddress`（开关两种值；地址规范化；日志 `by=cli`）、`TestExecuteForAdminRefuses`；成功的测试断言日志恰好一行（`by=api` 或 `by=cli`），拒绝的测试断言没有日志；`TestListWorkspacesIsTheCallersList`、`TestGetWorkspaceDecidesOnTheWorkspaceFound`、`TestGetWorkspaceNotFound`、`TestCheckSlug`。

### 2.12 接口描述和 HTTP 一侧（5.1–5.3、9.4）

- `api/modules/workspace.yaml`：四个操作（`listWorkspaces`、`createWorkspace`、`getWorkspace`、`checkWorkspaceSlug`），`x-problem-codes` 依次为 `[]`、`[workspace.creation_disabled, validation_failed, workspace.slug_taken]`、`[workspace.not_found]`、`[]`；结构 `WorkspaceRole`（`enum [5, 15, 20]`）、`OrganizationSize`、`Workspace`（10 个必填字段，`organization_size`、`logo_url` 可为 `null`；`logo_url` 在 M5 之前恒为 `null`）、`WorkspaceList {data}`、`WorkspaceCreate`、`SlugAvailability {available, reason?}`。`api/openapi.yaml` 加 `workspace` 标签和三个路径。
- `adapter/http`：`UseCases` 四个接口、`Register(router, api, uc)`；handler 只做类型转换，`workspace()` 把领域值转成响应（可空字段用 `nullable`）。`workspace` 在 P1 没有公开的操作。
- 测试：`TestMain(m) { apitest.Main(m, "workspace") }`（声明的每个码都要在这个包里经 `CheckResponse` 返回过，9.4）；`TestListWorkspacesAnswersTheCallersList`、`TestCreateWorkspaceAnswers201`、`TestCreateWorkspaceRefusals`（四种拒绝）、`TestCreateWorkspacePassesAnUnknownSizeToTheUseCase`（枚举由领域校验）、`TestGetWorkspace`、`TestCheckWorkspaceSlug`（路径里转义的 slug 解码后交给用例）。
- 前端的错误文案表（`authentication.helper.ts` 的 `PROBLEM_MESSAGES`）加三个码，`en`、`zh-CN` 的 `auth.json` 各加三条（第 3 节第 4 条）。

### 2.13 两段组合（6.6、11.6）

- `workspace/module.go`：`WorkspaceRoles` 接口、`Provided{WorkspaceRoles}`、`Provide(pool)`；`type AccountState = app.AccountState`；`Deps{Pool, Tx, Clock, Logger, Authorizer, Accounts, CreationEnabled}`；`New(Deps) *Module`、`Register`。Task 12 加 `Actions()`、`ReservedSlugs()`。
- `bootstrap/app.go` 按 6.6 的顺序：`identity.Provide(pool)` → `workspace.Provide(pool)` → `access.New({WorkspaceRoles})` → `workspace.New(…)`（`Accounts` 经 `bootstrap/ports.go` 的 `workspaceAccounts` 转换，`CreationEnabled: cfg.Workspace.CreationEnabled`）→ `identity.New`（不变）。`TxManager` 只建一个，两个模块共用。
- `bootstrap/ports.go`：`workspaceAccounts{accounts identity.Accounts}` 的两个方法把 `identity` 的 `AccountState` 转成 `workspace.AccountState`，`found` 和错误原样。
- 测试：`TestWorkspaceAccountsConvertsIdentitysAnswer`（假 `identity.Accounts` 记下参数）；`TestCreatingAWorkspaceAsConfigured`（开关两种值，真实组合和数据库：开时 alice 建 acme，列表有它，她读得到，bob 读到 `workspace.not_found`；关时 403 `workspace.creation_disabled`，什么都没建）。M2 的整程序测试 `TestAPIRoutesAreTheContractsOperations`、`TestOperationsThatNeedATokenAnswer401WithoutOne`、`TestTheAnswerToABrokenBodyStaysSmall`、`TestParametersThatDoNotBindAnswer400`、`TestFieldCodesAreTheContractsEnum` 不改就覆盖四个新操作。

### 2.14 完整性与保留名单"服务端"一段（3.4、3.10、9.1、9.4）

- `bootstrap/actions_test.go`：`moduleActions()`（`"workspace": workspace.Actions()`）；`actionViolations(modules, rules)` 报三种不一致：动作没有规则行、规则行不是任何模块的动作、一个动作声明两次；`TestEveryActionHasARuleAndEveryRuleAnAction`（规则表不能为空）；`TestActionViolationsCatchesEachMismatch`（四个反例）。整分支修复时改为默认要求：`actionlessModules`（`identity`、`instance`、`access`，各写明理由）；`TestEveryModuleDeclaresItsActionsOrHasNone` 在测试时读 `internal/modules` 的目录，每个模块要么在 `moduleActions` 中、要么在 `actionlessModules` 中，两张表里没有目录的名字也报出；`TestModuleViolationsCatchesEachGap`（三个反例）。
- `bootstrap/reserved_test.go`：`serverPaths(t, app, webFiles)` = 组合根注册的每个路由的第一段（前端页面的 `/` 除外），加上前端文件中缺失的文件答 404（而不是页面）的顶层目录（`assets`）；`TestTheReservedServerSlugsAreTheServersTopLevelPaths`：等于名单的"服务端"一段；`icons/` 缺失的文件答页面，不算（第 3 节第 6 条）。

### 2.15 建工作区与 M2 的停用（3.6 约定六；9.3 交错 8 的前一半）

`bootstrap/interleaving_test.go`：两边各跑真实的用例（`identity` 的 `Deactivate.ExecuteByEmail`，与 `deactivateMe` 同一条 `FOR NO KEY UPDATE` 的写入路径；`workspace` 的 `CreateWorkspace.Execute`）、真实的存储和数据库；闸门由包装存储的测试替身放在持锁的事务里（停用：最后一次写入 `RevokeSessions` 之前，账户已在事务里停用；建工作区：第一次插入 `CreateWorkspace` 之前）；`pgtest.WaitForLockWait` 证明另一方在等这一行之后才打开闸门；每个等待 10 秒为限。

- `TestDeactivationFirstRefusesTheWorkspace`：停用持锁；建工作区等它的 `FOR SHARE`，锁下读到已停用，401；没有工作区，账户已停用。
- `TestCreationFirstHoldsOffTheDeactivation`：建工作区持 `FOR SHARE`；停用等它的 `FOR NO KEY UPDATE`，等待期间账户仍有效、没有提交的工作区；放开之后两边都成功：一个工作区，账户已停用。停用成员关系是 P6 的端口（交错 8 的后一半）。

### 2.16 `nerve workspaces create`（3.11、6.6）

- `workspace/admin.go`：`AdminDeps{Pool, Tx, Clock, Logger, Accounts}`、`NewAdmin(AdminDeps) *Admin`、`(*Admin).CreateWorkspace(ctx, slug, name, adminEmail) (domain.Workspace, error)`（`ExecuteForAdmin`，不看开关）。
- `bootstrap/workspaces.go`：`WorkspaceCommand`；`Workspaces(ctx, cfg, logOut, out, cmd)`：连接池、`identity.Provide(pool).Accounts`（经 `workspaceAccounts`）、`workspace.NewAdmin`；没有签名密钥、`Authorizer`、River；`CreateWorkspace(slug, name, adminEmail)` 打印 `created workspace <slug> with admin <规范化的邮箱>`。`cliFieldName`、`commandError` 从 `users.go` 移到 `commands.go`（两个命令共用），加上 `slug` → `--slug`、`name` → `--name`。
- `cmd/nerve/workspaces.go`：`nerve workspaces`（无子命令时打印帮助），`create --slug <slug> --name <name> --admin-email <address>`，三个标志都必填。
- `archtest/composition_test.go`：`TestUsersComposeNoServerAndNoJobs` 改为 `TestCommandsComposeNoServerAndNoJobs`，对 `Users`、`Workspaces` 各走一遍静态调用：必须到达各自的 `NewAdmin`（`identity`、`workspace`）；不能到达任何模块根包的 `New`（`identity.New`、`workspace.New`、`access.New` 和以后的）、`platform/httpserver`、`ratelimit`、`jobs`、River。
- 测试：`TestWorkspacesCreate`（开关关闭；地址规范化；一行；日志 `msg="workspace created"`、`by=cli`；`workspaces`、`workspace_members` 各一行；`river_job` 为空）；`TestWorkspacesCreateErrors`（slug 被占用、没有账户、已停用、保留的 slug（`--slug is reserved`）、名称和 slug 都不合规：没有输出、一行错误、数据库不变）；`TestWorkspacesCreateCommand`（`cmd/nerve`：退出码、stdout、stderr 的最后一行；缺标志；未知子命令）；`TestBareWorkspacesPrintsHelp`。
- `createWorkspace` 的说明、`README.md` 部署一节（8.7）、`config.yaml` 和 `config.go` 的注释写上这个命令。

### 2.17 权限矩阵的骨架（9.2）

- `pgtest.NewDatabaseFrom(t, prepared) string`：`CREATE DATABASE … TEMPLATE <prepared>`，副本在测试结束时删除；复制时不能有连接连着 `prepared`。`TestNewDatabaseFromCopiesThePreparedDatabase`：两个副本各自独立，写一个不影响另一个和源库。
- `apitest.Operation` 加 `ID`（`operationId`）和 `Tags`；`TestOperations` 核对。
- 规则：`api/modules/<m>.yaml` 的每个操作的标签恰好是 `[<m>]`。矩阵按标签找一个模块的操作，标签写错（例如复制来的 `tags: [identity]`）的操作会悄悄漏掉它的行；`TestEveryOperationIsTaggedWithItsModule`（`apitest/rules_test.go`，Task 15）核对每个模块文件的每个操作。
- `bootstrap/permission_matrix_test.go`：
  - 列（`caller`）：管理员、成员、访客、从来不是成员、已被移出、工作区已删除；最后一列的请求指向已删除的工作区 `gone`（它的管理员），其余指向 `acme`。
  - 准备（`prepareMatrix`，在子测试 `prepare` 里，结束时关掉它的 app 和连接池）：每列一个账户，经接口注册拿访问令牌（所有 app 用同一个密钥文件，令牌在每个副本上都有效）；工作区和成员关系经 `workspace` 的存储写入；移出和删除还没有存储，用 SQL（第 3 节第 10 条）。
  - 行（`matrixRows()`）：`listWorkspaces`、`checkWorkspaceSlug`：各列 200；`createWorkspace`：各列 201（写，每格一个副本）；`createWorkspace, creation switched off`：各列 403 `workspace.creation_disabled`（每格一个副本和一个关闭开关的 app）；`getWorkspace`：管理员、成员、访客 200，其余 404 `workspace.not_found`。共 30 格，写的 12 格。
  - `TestPermissionMatrix`：读的格子共用一个副本上的 app，写的格子并行；每格断言状态码和 problem 的码，请求和响应都经契约核对。
  - 列"从来不是""已被移出"的账户是另一个工作区 `other` 的管理员：把角色读错了工作区的实现在这两列失败。
  - `TestThePermissionMatrixCoversEveryOperation`（`permission_matrix_coverage_test.go`）：除了 `matrixExempt`（`identity` 账户级、`instance` 公开，各写明理由）的操作，契约的每个操作都有一行；豁免表中没有操作的名字报出；每行指向一个存在的操作，每行对每列都有一格；每格的请求是它那一行的操作（方法和路径）；发 GET 以外的请求的行必须标 `write`（它的格子不能跑在读的格子共用的副本上）。`TestMatrixViolationsCatchesEachGap`：每种缺口各有反例，含没有列在任何表中的模块的操作、过期和拼错的豁免名。
  - 耗时（附录 A）：整个矩阵 0.24–1.34 秒（准备 0.07 秒）。

### 2.18 端到端（2、9.6）

- `e2e/fixtures/api.ts`：`Workspace`、`WorkspaceCreate` 类型；`slugFor(testInfo, label)`（`label-` 加 16 位十六进制，按测试、重复、重试区分）；`createWorkspace(api, token, body)`（生成的客户端，期望 201）。
- `e2e/fixtures/assert/workspace.ts`：`expectWorkspaceCreated(db, adminEmail, w)`（`workspaces` 一行：值、`created_by_id = updated_by_id` = 这个账户、`updated_at = created_at`、未删除；`workspace_members` 一行：这个账户、20、有效、同一时刻；返回工作区 id）；`countWorkspaces`、`expectNoWorkspaceAdded`。
- `e2e/fixtures/workspaces.ts`：`nerveWorkspaces(db, args, env)`、`nerveWorkspacesFails(db, args, message)`。
- `stories/workspace/w1-create-workspace.spec.ts`（W1 的接口版本，PAT）：slug 的三种回答（可用、被占用、保留）；建工作区，响应与数据库一致；被占用 409 `workspace.slug_taken`；保留 422（`slug`，`not_allowed`）；同一个库上关闭创建的第二个 nerve 答 403 `workspace.creation_disabled`（PAT 在那里同样有效）；三次拒绝之后数据库不变。
- `stories/workspace/w10-admin-create-workspace.spec.ts`（W10）：开关关闭时命令照常建，输出一行；数据库断言与 W1 相同；这个账户经接口列出它，`role = 20`；slug 被占用、没有账户、账户已停用：退出码 1、一行说明、数据库不变。

### 2.19 文档（3.20 的 P1 各行、8.7）

| 文档 | 位置 | 内容 |
|---|---|---|
| M2 设计 | 3.11 | 核对第 1 条：前缀是 nerve 的一个模块；`Forbidden` 一行和平台码表加 `forbidden` |
| 总体设计 | 3.4 | 集合型的列表不分页，`{data}` 封套，第一个是 `listWorkspaces` |
| 总体设计 | 3.5 | 平台码加 `forbidden` |
| 总体设计 | 6.2 | `shared` 的内容；`module.go` 的 `Provide` |
| 总体设计 | 6.3 | 第 4 条：两段组合 |
| 总体设计 | 6.5 | 规则表的形式、看不到与不能做、相对规则在用例中、判定在父行的锁之后；`AllowCreator` 由 M4 加入 |
| 差异清单 | 二·按表 | `workspaces`、`workspace_members` 逐列；`workspaces_slug_key` 的部分唯一另写一行 |
| 差异清单 | 三、四 | 4.11 中标 P1 的行（第 3 节第 11 条） |
| README | 部署 | `nerve workspaces create`；`workspace.creation_enabled = false` 时用它（Task 14） |
| M3 的四份交接 | 文末 | 追加"处理结果（M3/P1）"，照 M2 各 Phase 的写法（第 7 节） |

## 3. 与设计的差异和补充（请控制者裁定）

以下都没有改变 M3 设计的架构。

1. **任务的划分：16 个，不是 15 个**。设计的任务与 plan 的对应：1 → 1（sqlc 条目移到 9，见下条）；2 → 2；3、4 → 3（两者改同一组 `apitest` 文件，`forbidden` 进 `platformCodes` 和前缀规则一起测）；5 → 4；6 → 5；7 → 6；8 → 7（`actions.go`）和 12（完整性测试要 `workspace.Actions()`，在模块入口之后）；9 → 11（下面说明）；10 → 7（领域和名单）和 12（"服务端"一段的 Go 测试要组合根）；11 → 8（`identity.Provide`）、9（存储）、10（用例）；12 → 10、11（`checkWorkspaceSlug` 的用例和 handler 与另三个同一批）；13 → 14；14 → 15；15 → 16。另加 13（交错，第 5 条）。组合（设计的任务 9）与接口描述、HTTP 一侧放在同一个 Task：模块没有 HTTP 一侧时，组合根没有东西可接；只加接口描述而不注册路由，M2 的整程序测试（路由等于契约的操作）失败。最大的 Task 11 在 plan 中 1,275 行，在约 1,500 行的上限之内；plan 共 16 个 Task。
2. **`server/sqlc.yaml` 的 `workspace` 条目在 Task 9，不在 Task 1**：sqlc 对没有查询的条目报错（`error parsing queries: no queries contained in paths …`，附录 A），`make gen` 在 Task 1 就会失败。`TestSQLCSchemaScope` 只要求"有查询的模块有条目"，Task 1 到 8 之间 `workspace` 没有查询，规则照旧成立。
3. **保留名单的"应用"一段在 P1 含 `invitations`**：3.10 的名单是决策点 2 删掉 `/invitations` 页之后的状态（P8、P9）。这一页在 P1 仍在，名单不含它就会放行一个与现有页面冲突的 slug；删掉这一页的改动同时把它移出名单（文件的注释写明）。"应用"一段与路由表的核对（web vitest）随前端数据层在 P8 加入（9.5）。
4. **前端的错误文案表在 Task 11 加三个码**：M2 设计 3.11、7.3 的 vitest 要求 `PROBLEM_MESSAGES` 的键恰好等于 `api/dist/openapi.yaml` 中的全部 `x-problem-codes`；`workspace.yaml` 一加上三个码，`make test-web` 就失败。所以后端的 Phase 也要改这张表和两份 `auth.json`（设计 12 节的 P1 没有前端的任务，这是设计的遗漏）。文案放在 `auth` 命名空间，与 M2 的码一起。`forbidden` 在 P1 没有操作声明它，不进表（P2 第一次声明时加）。**已裁定，写进设计**：M3 设计第 12 节约束 4（P2–P7 声明新码的任务在同一个任务里加文案并运行 `make test-web`），这张表和它的文案在 P8 任务 13 移到通用的位置。
5. **P1 加了建工作区与 M2 停用的交错（Task 13）**：9.3 把交错 8 标为 P6（那时停用有了结束成员关系的端口）。约定六的"账户行最先 `FOR SHARE`、锁下要求有效"是 P1 的代码，P1 的变异（去掉锁、锁挪到事务外、在锁之前读 `is_active`）只有真实的争锁才能发现（附录 A）；P6 在同一个文件里补上成员关系的一半。
6. **"服务端"一段的核对方式**：3.10 写"组合根在前端页面之外注册的顶层路径"。`assets` 不是组合根注册的路由，是 `webui` 对 `/assets/` 下缺失的文件答 404 而不是页面（M2/P4）。测试把"路由的第一段"和"缺失文件答 404 的前端顶层目录"合起来，等于名单的这一段；`platform` 的生产代码不变（6.1）。
7. **两个只给命令行的错误码**：`workspace.account_not_found`、`workspace.account_deactivated`（3.11 的"没有这个账户、账户已停用"）。它们不在 5.3 的表里，不进任何 `x-problem-codes`；命令行打印它们的说明（照 M2/P3b 的 `identity.account_not_found`）。T16 的块不改 M3 设计；两个码由本 spec 的提交写进 M3 设计 3.11。
8. **规则表之外的两种失败都是 500**：没有规则行的动作（`Authorize`）、级别不认识的规则（`Decide`）都是内部错误，不是 403 或 404：调用方无从补救，这是代码的缺陷，默认拒绝仍然成立。未知的角色什么都不允许（按集合比较）。
9. **项目级的判定在 P1 写全**：设计 12 节任务 6 要三个级别；P1 没有项目级的规则行，`Authorizer` 也还不读项目（`ProjectAccess` 在 P4）。`Decide` 的项目级分支由表测试（17 种身份 × 4 条规则）和三个值以外的角色的性质测试覆盖，P4 加规则行和端口时不改判定。pre-flight（M1）发现第一版的项目级放行三个值以外的角色（工作区角色 10 的项目管理员、看得到即可时的项目角色 10）；修订后在看得到之后先查 `knownRoles`，两个级别都失败关闭（附录 A 的两个变异）。
10. **矩阵的准备数据**：9.2 写"用各模块的仓储写入"。账户经接口注册（为了拿到令牌，注册按 IP 的限流在 test 配置中很宽）；工作区和成员关系经 `workspace` 的存储；"已被移出""工作区已删除"两列的状态还没有存储能写（移出在 P5、删除在 P2），用两条 SQL，写明由那两个 Phase 换成存储。另外，矩阵要求 `workspace`（以后加 `project`）的**每个**操作都有一行，账户级的也要（9.2 只要求工作区级和项目级的）：账户级的三行在本 Phase 本来就有，这样新操作不会因为被归为"账户级"而漏掉。整分支修复把核对改为默认要求：除了 `matrixExempt` 中的 `identity`、`instance`，每个模块的操作都要有行，新模块不必登记就在其中。公开的 `getWorkspaceInvitation`（P3）需要在矩阵中另加一列"不带令牌"，或给矩阵加按操作的豁免（`matrixExempt` 按模块豁免，把 `workspace` 加进去会豁免它的全部操作）：P3 决定。矩阵按标签找一个模块的操作，标签由 `TestEveryOperationIsTaggedWithItsModule` 钉住（2.17，pre-flight L2），标签写错的操作不能绕过完整性核对。
11. **差异清单"集合型的列表"一行不写"关联字段带 `_id`"**：4.11 这一行含"关联字段带 `_id`（5.2）"。P1 的 `Workspace` 没有关联字段，写上就是描述不存在的行为；由第一个带关联字段的资源（P2 的 `WorkspaceMember.workspace_id`）写上。
12. **`platform` 的两处测试工具和一处注释**：`pgtest.NewDatabaseFrom`（9.2 要求）和 `apitest.Operation` 的 `ID`、`Tags`（矩阵的完整性要按操作名和模块核对）都是只给测试用的包；`pgtest`、`apitest` 只给测试用由架构测试保证（`testHelpersOnlyInTests`：它们只被测试导入）。`platform/config/config.go` 中 `WorkspaceConfig` 的注释原来写"创建工作区在 M3 加入并执行它"，改为现在的行为，只改注释、不改代码。`platform` 的生产代码的行为不变，在 6.1 之内（pre-flight Q1）。
13. **"先锁父行，再判定"在 P1 没有可测的地方**：P1 唯一经过 `Authorizer` 的是读（`getWorkspace`，不开事务）；`createWorkspace` 是账户级的，不经过 `Authorizer`。缺陷类别"在父行的锁之前判定"的变异从 P2 的 `updateWorkspace` 起才有对象；`Authorizer` 的端口在 ctx 带的事务里读（`TestActiveRoleReadsInTheTransaction`），为那时做好准备。
14. **命令与接口同一套校验（补充）**：`nerve workspaces create` 与接口共用 `CheckNewWorkspace`，大写的 slug 在命令行同样被拒绝（`--slug may hold only …`），不像 Plane 的前端那样先转成小写；保留的 slug 答 `--slug is reserved`。
15. **建工作区的日志有测试看到（补充）**：设计没有写测试怎样看到 `workspace created` 的日志。用例测试断言成功时恰好一行（`by=api` 或 `by=cli`）、拒绝时没有；命令的测试另核对 `by=cli`（附录 A 的两个日志变异）。

## 4. 验收标准（完成线，M3 设计 12 节 P1）

- [ ] W1、W10 的接口版本通过，此前的 50 个故事仍然通过。
- [ ] 本 Phase 的矩阵格子（30 个）通过；矩阵的完整性核对通过。
- [ ] 判定表测试、`TestEveryRuleDecidesItsCells`、操作名的完整性测试通过。
- [ ] `TestSQLCSchemaScope` 的四种写法的反例、前缀规则的反例在规则漏掉时失败（附录 A 的变异）。
- [ ] 交错的两个测试在真实数据库上通过，`-count=5`、`-race` 也通过。
- [ ] 架构测试通过（含命令行的组合）。
- [ ] 7 个迁移 up、down、再 up 通过。
- [ ] `make lint`、`make test`、`make gen-check`、`make knip`、`make test-web`、`make e2e` 通过。
- [ ] 3.20 中 P1 的各行、8.7 中 P1 的一行在同一次合并中写好；四份交接有"处理结果（M3/P1）"。

## 5. 不在 P1 范围内

- 修改、删除工作区，成员，显示设置（P2）；邀请（P3）；项目、`ProjectAccess`（P4）；移出、离开、`reactivate-member`（P5）；停用的端口（P6）；状态、标签（P7）；前端（P8–P11）。
- 保留名单"应用"一段的 vitest 和删除 `RESTRICTED_URLS`（P8）。
- `identity.LoadKeys` 和 `identity.New` 的新依赖（签名密钥先载入、`MembershipDeactivator`、`SignupInvitations`）：随用到它们的 Phase（P3、P6）。

**留给后面的 Phase**（控制者对 pre-flight 的裁定；各 Phase 的 spec 接过去）：

| Phase | 条目 |
|---|---|
| P2 | 矩阵中"工作区已删除"一列的准备数据从 SQL 改为经删除工作区的存储写入（第 3 节第 10 条）；"在父行的锁之前判定"的变异（第 3 节第 13 条）；差异清单"集合型的列表"一行写上"关联字段带 `_id`"（第 3 节第 11 条）；`forbidden` 随第一个声明它的操作进前端的文案表（设计第 12 节约束 4） |
| P3 | 公开的 `getWorkspaceInvitation` 在矩阵中的一列"不带令牌"，或按操作的豁免（不是把 `workspace` 加进 `matrixExempt`）（第 3 节第 10 条） |
| P5 | 矩阵中"已被移出"一列的准备数据从 SQL 改为经移出成员的存储写入（第 3 节第 10 条） |
| P6 | 交错 8 的后一半：停用结束新的成员关系（2.15） |
| P8 | 问题码的文案表 `PROBLEM_MESSAGES` 和它的文案移出 `authentication.helper.ts` 和 `auth` 命名空间（设计 P8 任务 13） |

## 6. 风险

| 风险 | 应对 |
|---|---|
| 矩阵在 P2–P7 长到约 400 格，时间超过 9.2 的预算（20–30 秒） | P1 的 30 格 0.24–1.34 秒，写的格子每格一个副本加一个 app，约 0.1 秒、并行；按这个速度 250 个写格约 2–4 秒。后面的 Phase 记下自己的耗时 |
| 并行的写格子各开一个 app 和它的连接池（测试配置最多 4 个连接），格子多时连接数可能超过测试容器的 `max_connections`（100） | `go test` 默认的并行度是 GOMAXPROCS；P1 有 12 个并行格子。格子多的 Phase 看到连接错误时，给写格子加一个并行度的上限 |
| 保留名单的"服务端"一段靠探测前端文件判断 `assets` | 探测用一个不存在的文件名；前端改变缺失文件的回答时测试失败而不是悄悄通过（`icons/` 的对照） |
| `reserved_slugs.txt` 解析失败时 nerve 启动即 panic | 文件嵌入二进制；`TestTheReservedListIsWellFormed` 在每次 `make test` 解析它 |
| 前端文案放在 `auth` 命名空间 | 只是位置；P8 任务 13 把表和文案移到通用的位置（设计第 12 节约束 4） |

## 7. 交接和关闭条件

| 交接 | P1 处理的条目 | 留下的条目 |
|---|---|---|
| M2-closeout 第 4 节 | `last_workspace_id` 不补外键：理由在 M3 设计 3.14，P1 不改代码 | — |
| M2-closeout 第 5 节 | 关闭时 403 `workspace.creation_disabled`，有测试（Task 10、11、15、16）；命令写进设计 3.11 并实现（Task 14） | — |
| M2-closeout 第 7 节 | 接口一侧的一部分：`Workspace.logo_url` 必有、可为 `null`，M5 之前总是 `null`（Task 11） | `cover_image_url`、`MemberUser.avatar_url`（P2、P4）；`IUserLite`（P8）；本节保持 `open` |
| M2-closeout 第 8 节 | 第 1 件：端口（`access/app/ports.go` 的 `WorkspaceRoles`，不是例外）；第 2 件：四种写法和反例（Task 2）；第 3 件：`Authorizer` 在 `shared`，`shared` 仍只依赖标准库（Task 3） | `ProjectAccess` 在 P4 照同一写法 |
| M2-closeout 第 12 节 | P1 的列表是集合型的，不分页（3.12）；按关闭条件由 P1 的 review 写明 | 本节在 M3 收尾时原样写进 M4 的交接（13.2） |
| M1-P2、M1-P3、M1-P4 的保留名单 | 服务端一侧：一份名单，三段，"服务端"一段有 Go 测试（Task 7、12）；Plane 的产品词去掉（3.10） | 前端一侧（删 `RESTRICTED_URLS`、vitest）：P8 |

每一行的结果由 Task 16 追加到对应交接的"处理结果（M3/P1）"（M2-closeout、M1-P2、M1-P3、M1-P4 四份），状态保持 `open`：这几份交接都还有别的 Phase 的条目。

## 附录 A：原型验证记录（2026-09-29）

原型在 `$M3TMP/p1proto`（`1f0e7ed7` 的副本，Go 1.27.1、Node 24.15.0、pnpm 11.10.0、Docker 29.7.2；`bin/golangci-lint` 2.13.2）。每个 Task 做完时把源文件存一份快照（`$M3TMP/p1snap/T1`…`T16`），plan 的代码块由脚本从相邻两份快照的差异生成：新文件给完整内容（````file`），删除的文件给 ````delete`，改动的文件给最少上下文的 ````old`/````new` 对，改动大于文件本身时给完整内容（````whole`）；脚本把每个 Task 的块应用到前一份快照上，结果与这一份快照逐字节相同。生成的文件不进块，按 SHA-256 核对。

**逐 Task 复现**（`$M3TMP/p1tools/replay.py`）：在 `$M3TMP/p1replay`（`1f0e7ed7` 的另一个副本，`pnpm install --frozen-lockfile`）按 plan 的 16 个 Task 依次执行：`planapply.mjs` 从 plan 的文本中取出这个 Task 的块写入；然后按顺序执行这个 Task 的每一条 `Run:` 命令，原样照 plan。例外只有：`make gen`、`make gen-go` 之后核对全部生成物与这个 Task 的快照逐字节相同；`shasum -a 256` 的输出与 plan 表中的 SHA-256 和行数核对；副本不是 git 仓库时 `make lint-web` 的关键词守卫由 `kwcheck.mjs` 代替（同样的规则、同样的文件）；`make e2e` 之前把副本初始化为 git 仓库并提交（F3）；提交之后的 `make gen-check` 由生成物的核对代替，复现结束时在已提交的副本上另跑一次。每个 Task 另核对 `go.mod`、`go.sum`、`server/tools` 和 `pnpm-lock.yaml` 不变。

| Task | 复现的结果 |
|---|---|
| 1 | `./migrations/`、`TestTheGrantsFileCoversEveryRelationAndFunction` ok；lint 2 × `0 issues.`；`make test` 32 个 `ok` |
| 2 | `TestSQLC*` ok；lint、`make test` 同上 |
| 3 | `make gen`：13 个生成物与快照相同，3 个 SHA-256 与表相同；`shared`、`apitest` ok；lint；`make test` 32 `ok`；`make lint-web`（54 个 turbo 任务）、`make knip`、`make test-web`（16 个）通过 |
| 4 | `shared`、`identity/...` 的 10 个包 ok；lint；`make test` 32 `ok` |
| 5–7 | 各自的包 ok；lint；`make test` 33、34、35 `ok` |
| 8 | `make gen-go`：13 个生成物相同，`users.sql.go` 的 SHA-256 相同；两个锁测试 ok；lint；`make test` 35 `ok` |
| 9 | `make gen-go`：17 个生成物相同，4 个 SHA-256 相同；存储 ok；lint；`make test` 36 `ok` |
| 10 | `workspace/app` ok；lint；`make test` 37 `ok` |
| 11 | `make gen`：19 个生成物相同，4 个 SHA-256 相同；`workspace/...`、`bootstrap` ok；lint；`make test` 38 `ok`；前端检查、`make knip`、`make test-web` 通过 |
| 12 | 3 个测试 ok；lint；`make test` 38 `ok` |
| 13 | 交错 `-count=5 -race` ok；lint；`make test` 38 `ok` |
| 14 | `make gen`：19 个生成物相同，2 个 SHA-256 相同；3 个包 ok；lint；`make test` 38 `ok`；前端检查、`make knip`、`make test-web` 通过 |
| 15 | `pgtest`、`apitest`（含 `TestEveryOperationIsTaggedWithItsModule`）ok；矩阵 `-count=3 -race` ok；`-v`：`TestPermissionMatrix` 1.32 秒（此前两次完整复现 1.34、1.29 秒）、`prepare` 0.07 秒；lint；`make test` 38 `ok` |
| 16 | 12 个文件（含四份交接的处理结果）；lint；`make test` 38 `ok`；前端检查、`make knip`、`make test-web` 通过；`make e2e` 52 个通过 |
| 结束 | 复现的树与原型、与 T16 快照逐文件相同（2,737 个文件，0 个差异）；已提交的副本上 `make gen-check` 和 `make lint-web`（关键词守卫 60 条规则、3 个例外，没有命中）通过；`planapply.mjs check` 从 `1f0e7ed7` 起 192 个块全部通过 |

| 核对 | 命令 | 结果 |
|---|---|---|
| 生成物 | `make gen-check`（原型在 Task 16 之后初始化为 git 仓库并提交，所以 `gen-check` 可用）；各 Task 的 `make gen` 与快照比较 | 没有差异 |
| Go 静态检查 | `make lint-go` | server 0 issues；server/tools 0 issues |
| Go 测试 | `make test`（testcontainers，`postgres:18.6`） | 全部通过（37 个包加 `server/tools` 的 1 个） |
| 前端检查 | `make lint-web`（关键词守卫 60 条规则、3 个例外，没有命中；turbo 54 个任务）；`make knip`；`make test-web`（16 个任务） | 全部通过 |
| 端到端 | `make e2e` | 52 个全部通过（M2 的 50 个、W1、W10） |
| 迁移 | `TestMigrationsGoUpDownAndUpAgain`（7 个迁移） | up、down、再 up 通过 |
| 矩阵 | `go test -count=5 -v -run 'TestPermissionMatrix$' ./internal/bootstrap/` | 每次 0.24–1.28 秒（三次完整复现中 1.34、1.29、1.32 秒），准备 0.06–0.07 秒；`-race -count=3` 通过 |
| 复现 | plan 的 16 个 Task 依次在 `$M3TMP/p1replay` 执行 | 见上面的表 |

**原型中定下的事实**：

- **F1** sqlc 对没有查询的条目：`error parsing queries: no queries contained in paths …`（只含 `workspace` 两个迁移、`queries` 为空目录的配置，`go tool -modfile=tools/go.mod sqlc generate`）。所以条目随第一批查询加入（第 3 节第 2 条）。
- **F2** 前端文案表的 vitest（`authentication.helper.test.ts` 的 "have a message for every problem code of the contract, and no other"）在 Task 11 之后、加文案之前失败，缺 `workspace.creation_disabled`、`workspace.not_found`、`workspace.slug_taken`（第 3 节第 4 条）。
- **F3** S3（实例信息里的 `commit`）要求副本是 git 仓库：不是时 `commit` 是 `unknown`，S3 失败（M2 的各 Phase 相同）。原型在 Task 16 之后 `git init` 并提交（只在副本里，用 `--git-dir`、`--work-tree`），之后 52 个故事全部通过。
- **F4** `pgtest.NewDatabaseFrom` 并行复制同一个模板：12 个写格子同时 `CREATE DATABASE … TEMPLATE`，没有冲突；准备数据的子测试结束时关掉它的 app 和连接池，复制时模板上没有连接。

**变异核对**（`$M3TMP/p1tools/mutants.py`：改一处代码，跑点名的测试，恢复；期望这些测试失败，并且输出含点名的失败行）。47 个变异全部被发现（在按 pre-flight 修订之后的原型上一次跑完，之后原型与复现的树仍逐文件相同；其中 5 个是修订加的）：

| 变异 | 被哪些测试发现 |
|---|---|
| 规则表删掉 `workspace.read` 一行 | `TestEveryRuleDecidesItsCells`、`TestEveryActionHasARuleAndEveryRuleAnAction`、`TestPermissionMatrix` |
| 规则行去掉访客（收窄） | `TestEveryRuleDecidesItsCells`、`TestPermissionMatrix/getWorkspace/guest` |
| 规则行加上角色 10（放宽） | `TestEveryRuleDecidesItsCells` |
| 判定不看 `Active`（被移出的仍算） | `TestDecideAtTheWorkspaceLevel`、`TestPermissionMatrix/getWorkspace/removed` |
| 判定不看规则的角色 | `TestDecideAtTheWorkspaceLevel` |
| 角色按大小比较（未知的角色 10 通过） | `TestDecideAtTheWorkspaceLevel` |
| 访客看得到没加入的公开项目 | `TestDecideAtTheProjectLevels` |
| 没有规则行的动作被允许 | `TestAuthorizeRefusesAnActionWithoutARule` |
| `ActiveRole` 的两个参数对调 | `TestAuthorizeReadsTheCallersRoleInTheTargetsWorkspace`、`TestPermissionMatrix/getWorkspace/admin` |
| 完整性：动作没有规则行也通过 | `TestActionViolationsCatchesEachMismatch` |
| 完整性：规则行没有动作也通过 | `TestActionViolationsCatchesEachMismatch` |
| 矩阵删掉 `getWorkspace` 一行 | `TestThePermissionMatrixCoversEveryOperation` |
| 矩阵的完整性不查缺行 | `TestMatrixViolationsCatchesEachGap` |
| 开关不接（恒开） | `TestCreatingAWorkspaceAsConfigured/creation_enabled_false`、`TestPermissionMatrix/createWorkspace,_creation_switched_off` |
| 开关不接（恒关） | `TestCreatingAWorkspaceAsConfigured/creation_enabled_true`、`TestPermissionMatrix/createWorkspace/admin` |
| `ShareAccount` 去掉 `FOR SHARE`（生成的代码） | `TestTheShareLockBlocksDeactivationNotAnotherShare/ShareAccount`、`TestCreationFirstHoldsOffTheDeactivation` |
| `ShareAccountByEmail` 去掉 `FOR SHARE` | `TestTheShareLockBlocksDeactivationNotAnotherShare/ShareAccountByEmail` |
| `FOR SHARE` 换成 `FOR UPDATE`（两个建工作区互相等） | `TestTheShareLockBlocksDeactivationNotAnotherShare/ShareAccount` |
| 账户行在事务之前锁 | `TestExecuteCreatesTheWorkspaceWithTheCallerAsAdmin`（假事务记下 "outside tx"）、`TestCreationFirstHoldsOffTheDeactivation` |
| 在锁之前、不加锁地读 `is_active`，用它回答 | `TestDeactivationFirstRefusesTheWorkspace` |
| 交错的闸门永不打开 | `TestCreationFirstHoldsOffTheDeactivation` 在 10 秒的期限失败（"did not end within 10s"），不挂住 |
| 名单的"服务端"一段少 `healthz` | `TestTheReservedServerSlugsAreTheServersTopLevelPaths` |
| 名单的"预留"一段少 `admin` | `TestTheReservedListIsWellFormed` |
| 领域不查保留名 | `TestCheckSlug`、`TestCheckNewWorkspaceReportsEveryField`、`TestWorkspacesCreateErrors/a_reserved_slug` |
| `getWorkspace` 不把 `ErrNotVisible` 换成 404 | `TestGetWorkspaceNotFound`、`TestPermissionMatrix/getWorkspace/never_a_member` |
| `ActiveRole` 不看 `is_active`（生成的代码） | `TestActiveRole`、`TestPermissionMatrix/getWorkspace/removed` |
| `ActiveRole` 不看工作区已删除 | `TestActiveRole` |
| `ListWorkspaces` 列出已删除的工作区 | `TestListWorkspaces` |
| 存储认不出 slug 的唯一冲突 | `TestCreateWorkspaceSlugTaken` |
| `workspace` 的 HTTP 测试不再返回 `workspace.slug_taken`（9.4） | `apitest.Main`：`api/modules/workspace.yaml declares problem codes that no test answered …`，`createWorkspace: workspace.slug_taken` |
| `bootstrap.Workspaces` 建 `Authorizer` | `TestCommandsComposeNoServerAndNoJobs`（`bootstrap.Workspaces builds more …`） |
| 迁移放行角色 10 | `TestChecksRejectCounterexamples` |
| 迁移放行大写的 slug | `TestChecksRejectCounterexamples` |
| grants 少两张表 | `TestTheGrantsFileCoversEveryRelationAndFunction` |
| `TestSQLCSchemaScope` 不查 `ALTER TABLE` | `TestSQLCScopeReportsViolations/ALTER_TABLE_of_a_quoted_name` |
| 不查 `CREATE INDEX` | `…/CREATE_INDEX_on_another_module's_table`、`…/CREATE_UNIQUE_INDEX_without_a_name` |
| 不查 `CREATE TRIGGER` | `…/CREATE_TRIGGER_on_another_module's_table` |
| 不查 `DROP TABLE` | `…/DROP_TABLE_of_another_module's_table` |
| 不认带引号的名字 | `…/ALTER_TABLE_of_a_quoted_name` |
| 前缀规则放行不是模块的前缀 | `TestAuthoringRulesReportViolations` |
| 日志的 `by` 恒为 `api`（命令行建的也写 `api`） | `TestExecuteForAdminCreatesForTheAccountOfTheAddress` |
| 在事务里、写入成功之前记日志（被拒绝的也记） | `TestExecuteSlugTaken` |
| 项目级不查工作区角色是否三个值之一（pre-flight M1） | `TestDecideAtTheProjectLevels`："project admins, workspace role outside the three, project admin: Decide() = ok, want 403" |
| 项目级不查项目角色是否三个值之一（pre-flight M1） | `TestDecideAtTheProjectLevels`："seeing the project, project role outside the three: Decide() = ok, want 403" |
| 触发器的规则去掉 `(?s)`（pre-flight L1） | `TestSQLCScopeReportsViolations/CREATE_TRIGGER_on_another_module's_table` |
| `workspace.yaml` 的一个操作标成 `identity`（pre-flight L2） | `TestEveryOperationIsTaggedWithItsModule`：`GET /api/v0/workspaces in api/modules/workspace.yaml: tags = ["identity"], want ["workspace"]` |
| `RuleFor` 交出表自己的 `Roles`（pre-flight L3） | `TestRuleForReturnsACopy` |

**缺陷类别**（brief 列出的类别，对本 plan 逐类核对）：

| 类别 | 核对了什么 | 结果 |
|---|---|---|
| 规则行删掉、放宽、收窄 | `tableCells` 对每行每种身份写死答案；矩阵在 HTTP 上再测一遍；`RuleFor` 交出副本，调用方改不了表 | 四个变异都被发现 |
| 在父行的锁之前判定 | P1 没有经过 `Authorizer` 的写（第 3 节第 13 条） | 不适用；端口在事务里读，为 P2 准备 |
| 账户行没有锁、锁在事务之外 | 存储的锁测试（`55P03`）；用例的假事务；两个顺序的交错 | 四个变异都被发现 |
| 在锁之前读 `is_active` | 交错"停用在先" | 被发现 |
| 服务端缺保留名 | "服务端"一段的 Go 测试；三段的格式测试；领域和命令的测试 | 三个变异都被发现 |
| `TestSQLCSchemaScope`、前缀规则被削弱 | 每种写法一个反例（触发器的 `ON` 单独一行）；"code of no module"；每个操作的标签是它的模块 | 八个变异都被发现 |
| 完整性测试漏掉未登记的动作 | `actionViolations` 三种不一致各有反例；矩阵的完整性同样 | 四个变异都被发现 |
| 未知的动作、角色不按默认拒绝 | `Authorize` 的无规则；角色 10 在工作区级和项目级的判定表中都是列（项目级有工作区角色 10 和项目角色 10 两种） | 五个变异都被发现 |
| 断言不可能失败（403 被 401 掩盖） | 矩阵每格断言状态码和码；令牌在每个副本上有效（同一个密钥），否则每格都是 401 而失败 | 矩阵的"恒开""恒关"变异按格失败，不是整体 401 |
| 假实现忽略参数 | `fakeRoles`、`fakeAccounts`、`fakeWorkspaces`、`fakeIdentityAccounts` 按参数回答并记下参数；两个用户 × 两个工作区 | 参数对调的变异被发现 |
| 只有一行 | 存储测试有三个账户、七个工作区；命令的测试有第二个账户（交错只有一个账户：它测的是同一行上的锁） | `WHERE` 的三个变异都被发现 |
| 测试挂住 | 交错、锁测试每个等待 10 秒为限，`lock_timeout` 500 毫秒 | 闸门不开的变异在期限失败 |
| 闸门在争用区段之外 | 两个闸门都在持锁的事务里（停用在最后一次写入之前，建工作区在第一次插入之前）；`WaitForLockWait` 在放开闸门之前确认对方在等 | 去掉锁、锁挪出事务都让交错失败 |
| 说明与代码不符 | 接口描述、命令的 `Short`、README、`config.yaml`、`config.go`、名单文件的注释、代码注释中的设计节号 | 修正四处：名单文件原来写"前后端只有这一份"（前端的副本在 P8 才删）；`get_workspace.go` 引用的节号；`config.go` 的注释；存储测试的夹具注释（原来写两个账户，实际三个） |
| 接线没人看 | 开关的两种值；`Accounts` 的转换；命令行的组合；建工作区的日志 | 开关、组合的变异被发现；核对时发现 `by=api` 的日志没有测试看到（用例测试丢弃日志），补上断言之后两个日志变异都被发现 |
| 时间不是存下的值 | 存储按 `RETURNING` 回答，测试比较 UTC 和微秒；e2e 从数据库读时间比较 | 时钟本身给 UTC 微秒，"回显用例的时间"与存下的值相同，这个变异是等价的，不计 |
| 码只在别的模块的测试包里返回（9.4） | `workspace` 的 HTTP 测试返回它声明的每个码；删掉一个的变异被 `apitest.Main` 发现 | 被发现 |

**pre-flight**（控制者另派的 opus，`$M3TMP/p1-preflight/preflight.md`）：从 `1f0e7ed7` 独立复现 16 个 Task，结果等于原型；它自己的 25 个变异杀死 24 个，活下来的是触发器规则的 `(?s)`（L1）；本附录的 42 个变异全部再次被杀死。发现 High 0、Medium 1（M1：项目级放行三个值以外的角色）、Low 6，控制者全部接受。修订之后：原型改好，plan 从 `1f0e7ed7` 再完整复现一次（上面的表），树与原型逐文件相同，`make e2e` 52 个通过，47 个变异全部被发现；生成物的 SHA-256 一个都没有变。

**没有证明的**：

- 持续集成（ubuntu）上的运行：原型和复现都在 macOS 上。
- 交错 8 的后一半（停用结束新的成员关系）：P6。
- 矩阵到约 400 格时的耗时和连接数（第 6 节）。
- "先锁父行，再判定"：P1 没有这样的写（第 3 节第 13 条），P2 的第一个变异才证明它。
