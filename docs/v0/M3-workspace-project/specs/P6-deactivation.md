# M3/P6 停用账户与成员关系：设计说明（spec）

| 项 | 内容 |
|---|---|
| Phase | M3/P6 `deactivation` |
| 日期 | 2026-10-04 |
| 状态 | 第 3 节每一条已裁定：第 1、2、8 条和预检的 M1 由负责人 2026-10-04 裁定（"按建议"），其余由控制者同日裁定；裁定和预检的发现（M1、L1–L8；没有高的）都已落实，设计随 `e9289a4d` 改。终审的 I1 由负责人 2026-10-04 裁定"A"（第 20 条），与任务中停放的 28 个测试一侧的条目（P1–P28）一起在修订一轮落实；控制者同日的裁定 F-1（删除只写锁住的 id，补全"A"）至 F-5 在修订一轮的第二步落实 |
| 上级文档 | [M3 设计文档](../M3-design.md) 第 2（W9）、3.3、3.6（加锁表，约定一、二、五、六，方案 E）、3.7 规则 2、3.8、3.9、3.11、3.20（P6 两行）、5.3、6.2、6.3、6.6、6.7、8.7、9.1–9.4、9.6、12（P6 与约束 1–4）、13、13.1、14、16、17.1–17.4（I-1、M-2、M1、M2、M5，复核 spike 15、16）节 |
| 前置交接 | [P5a spec](P5a-memberships.md) 第 5 节 P6 一行，[P5a review](../reviews/P5a-memberships-review.md) 第 6 节；[P5b spec](P5b-project-memberships.md) 第 5 节 P6 一行，[P5b review](../reviews/P5b-project-memberships-review.md) 第 6 节；[P4a review](../reviews/P4a-projects-review.md)、[P4b review](../reviews/P4b-project-members-review.md)、[P3 review](../reviews/P3-invitations-review.md)、[P1 review](../reviews/P1-platform-review.md) 第 6 节中 P6 的条目；[M2 收尾交接](../handoffs/M2-closeout.md)第 6 节；[M4 的 M2 收尾交接](../../M4-issue-core/handoffs/M2-closeout.md)第 1 节（落点见第 3 节第 3 条） |
| 计划 | [P6 plan](../plans/P6-deactivation.md) |

本 spec 只写 M3 设计交给 P6 决定的东西：名字、签名、SQL、测试名，以及原型证明了什么。规则本身以 M3 设计为准，这里引用节号，不重述。P6 依赖 P5b（`6dcc0794` 的 `main`）。

P6 是评审敏感的一段（M3 设计 12 节约束 3）：停用跨越账户所在的每一个工作区，有接口（`deactivateMe`）和服务器管理员的命令（`nerve users deactivate`）两条路，改动 `identity`，并与约定六的每一条增长路径串行。停用是安全操作：停用之后账户没有一个有效的成员关系、没有一份发给它邮箱的邀请，它是唯一有效成员的工作区也没有待接受的邀请（预检 M1 的修法，2.3），没有人能经邀请进入一个没有管理员的工作区。服务器管理员一侧回来只经 `nerve users activate` 再 `nerve workspaces reactivate-member`；工作区管理员的新邀请是另一条路，`reactivate-member` 不看工作区有没有管理员（第 3 节第 8 条，负责人裁定的已知限制）。附录 A 的四十类清扫逐个核对每个这样的测试在它所说的性质去掉之后失败，并记下失败在哪一层（单元、存储、组合、端到端）；只在单元一层失败的安全或加锁的性质算缺口，例外逐条写在第 3 节。

## 1. 目标

按 M3 设计 12 节 P6 和 3.9：停用账户在同一个事务里结束他的全部成员关系、删除发给他邮箱的全部邀请和他留下的空工作区的待接受邀请；唯一管理员时拒绝；约定六的每一条增长路径都与停用串行。具体是：

- `identity`：端口 `MembershipDeactivator`，`LockedAccount` 加 `Email`，`deactivate` 在撤销会话之后、以锁下读到的邮箱调用它（复核 M5）；`deactivateMe` 声明 `workspace.sole_admin`、`project.sole_admin`，`identity` 自己的 HTTP 测试经 `fakeDeactivate.err` 返回这两个码（9.4，`apitest.Main` 两个方向）；
- `workspace` 的 `Deactivator`（`app.Deactivator`，经 `workspace.NewDeactivator` 构造），在 `deactivate` 已持有的账户行之下（约定六）：在调用时找出他是有效成员的未删除工作区，按 id 升序一次锁住（N），上锁时已删除的跳过、不答 404（复核 spike 15）；查 3.7 规则 2 的工作区一侧；以一条语句按 id 升序锁住（N）它要删除的邀请并得到它们的 id：发给这个邮箱的**全部**邀请，已忽略的也算（Plane `views/user/base.py:313`），和这些工作区中他之外没有有效成员的那些的**待接受**邀请（负责人对预检 M1 的裁定 (a)，设计 3.7、3.9）（终审的 I1 和裁定 F-1，第 3 节第 20 条）；在这把锁之后、也就在最后一把工作区锁之后读一次时钟（3.3）；按这些 id 软删除这些邀请，不写别的邀请；结束他在这些工作区的成员关系；调用一次 `ProjectCascade.EndMemberships`，跨这些工作区；它不复用 `membershipEnd`；
- 跨工作区的 `EndMemberships`（P5a 的实现，P6 不改）：在调用时列举，按项目 id 一次锁住全部项目，上锁时已删除的跳过，查规则 2 的项目一侧；`TestLockActiveMemberProjectsLocksInIDOrder` 改为跨两个工作区；
- `project.NewCascade`（随第一个使用者 `nerve users deactivate` 加入，6.3、设计 1043），`project.New` 经它建出同一个实现；`nerve users` 的组合只多建停用要的两件（`workspace.NewDeactivator`、`project.NewCascade`），archtest 钉住每个命令组合建什么、不建什么；
- 两条路（9.3）各测：拒绝时数据库完全不变（每张表每一行）；成功时一个事务、一个时刻、由他本人结束全部成员关系、删除全部邀请；跨两个工作区时项目按 id 升序加锁；复核 spike 15（列举到的工作区在上锁之前被删除，跳过）；改邮箱先提交时删除发给新邮箱的邀请；命令打印问题的说明、退出码 1，没有请求期限，中断之后什么都不改；
- 交错 7（Codex S1 的四种走法，另跑恢复以前的成员行）、8、13、14（各另跑以前的项目成员行）、15、16、19（换成真实的停用），两种顺序，`-count=5 -race`，没有 40P01；另加规则 2 之下两位管理员同时停用；
- 故事 W9 的接口版本（含命令）；README 部署一节的停用、恢复两条（8.7），差异清单第四节"停用账户"一行（3.20），M2 收尾交接第 6 节的处理结果。

## 2. 交付物

### 2.1 文件总览

路径相对于仓库根目录（`server/internal/` 省略）。"Task"是 plan 中负责它的任务（plan 有完整的文件表）。本 Phase 84 个手写的文件（新建 22 个、修改 62 个；9 个 Task 是 59 个，新建 16 个、修改 43 个，修订一轮新建 6 个、另改 19 个，见表的最后一行）、4 个生成物；没有迁移，没有新表，跨模块的端口只有设计 3.9 的 `MembershipDeactivator`（签名见第 3 节第 1 条）。

| 路径 | 内容 | Task |
|---|---|---|
| `workspace/adapter/postgres/queries/deactivation.sql`、`deactivation.go`、`deactivation_test.go`、`deactivation_invitations_test.go`、`failures_test.go`；`workspace/app/ports.go` | 计划的五条语句（锁工作区、工作区的唯一管理员、按邮箱删除邀请、删除他留下的空工作区的待接受邀请、结束工作区成员关系）；修订一轮先加锁住两条删除要写的邀请的一条（终审的 I1），再照裁定 F-1 把两条删除并成按锁住的 id 删除的一条（第 3 节第 20 条），现在五条，邀请的两条的存储测试另在 `deactivation_invitations_lock_test.go`、`deactivation_invitations_window_test.go`；端口 `AllMembershipsEnder` | 1 |
| `project/module.go`；`project/adapter/postgres/end_test.go`；`bootstrap/interleaving_{answers,deletion,endings,growth,leaving,removal,roles}_test.go` | `project.NewCascade`、`CascadeDeps`；跨工作区的 id 顺序；交错测试里只给 `Pool` 的 `project.New(…).Cascade()` 换成它（P4a review S5） | 2 |
| `workspace/app/deactivate_memberships.go` 及测试、`fakes_deactivation_test.go`、`fakes_workspaces_test.go`、`clock_test.go`、`ports.go`、`remove_member_test.go`；`workspace/domain/errors.go`、`project/domain/errors.go`；两个模块的 HTTP 测试 | `app.Deactivator`；两个 `sole_admin` 的说明对每个调用者都成立 | 3 |
| `identity/app/ports.go`、`deactivate.go` 及测试；`identity/adapter/postgres/queries/users.sql`、`users.go`、`credentials_test.go`；`identity/adapter/http/me_test.go`；`identity/module.go`、`admin.go`；`api/modules/identity.yaml`；`workspace/deactivator.go`、`workspace/module.go`；`bootstrap/app.go`、`users.go`、`deactivating_test.go`、`interleaving_test.go`、`interleaving_answers_test.go`；`archtest/composition_test.go` | 端口和锁下的邮箱；两个码；`workspace.NewDeactivator`、`Module.Deactivator()`；两处组合；交错 8、19 换成真实的停用 | 4 |
| `bootstrap/deactivation_world_test.go`、`deactivation_test.go`、`users.go`、`users_test.go`、`reactivation_races_test.go`；`cmd/nerve/users.go`、`users_test.go` | 两条路的世界和组合测试；命令的输出和说明 | 5 |
| `bootstrap/deactivation_locks_test.go`、`deactivation_races_test.go`、`interleaving_answers_test.go` | 每把锁的顺序和强度、事务的连接、命令的中断；等锁期间的删除和移出；锁下的邮箱 | 6 |
| `bootstrap/interleaving_accepting_test.go`、`reactivation_races_test.go`、`interleaving_answers_test.go`、`interleaving_invite_test.go`、`interleaving_growth_test.go` | 交错 7、16 | 7 |
| `bootstrap/interleaving_shrinking_test.go` | 交错 13、14、15；两位管理员同时停用 | 8 |
| `e2e/stories/workspace/w9-deactivation.spec.ts`、`e2e/stories/identity/a12-deactivate.spec.ts`；`README.md`；`docs/v0/plane-diff.md`；`docs/v0/v0-design.md`；`docs/v0/M3-workspace-project/handoffs/M2-closeout.md` | 故事 W9；文档 | 9 |
| 新建：`bootstrap/deactivation_crossed_test.go`、`workspace/adapter/postgres/deactivation_invitations_lock_test.go`、`deactivation_invitations_window_test.go`、`platform/postgres/pgtest/soon.go`、`soon_test.go`、`lockwait_behind_test.go`；另改：`platform/postgres/pgtest/lockwait.go`、`lockwait_test.go`、`pgtest.go`；`identity/app/fakes_test.go`；`workspace/app/fakes_projects_test.go`；`workspace/adapter/postgres/locks_test.go`、`directory_test.go`、`directory_share_test.go`、`update_invitation_test.go`；`project/adapter/postgres/demote_test.go`；`bootstrap/ending_test.go`、`ending_world_test.go`、`membership_world_test.go`、`project_membership_races_test.go`、`project_membership_role_race_test.go`、`project_removal_test.go`、`project_write_locks_test.go`、`reactivation_test.go`、`table_rows_test.go`（连同上面各 Task 已列的 30 个手写的文件和生成的 `gen/deactivation.sql.go`） | 修订一轮：终审的 I1 的锁和裁定 F-1 的按 id 删除（第 3 节第 20 条）和它们的测试；停放项 P1–P28（`pgtest.Soon`、`pgtest.WaitForLockWaitBehind`、`stampWriter` 等）和裁定 F-5 的三处 | 修订 |

生成物：`workspace/adapter/postgres/gen/deactivation.sql.go`（Task 1）、`identity/adapter/postgres/gen/users.sql.go`、`api/dist/openapi.yaml`、`web/packages/api-client/src/schema.gen.ts`（Task 4）。

### 2.2 依赖

不加 Go 模块、npm 包、迁移、表。P5a 的 `EndMemberships`、`LockActiveMemberProjects`、`SoleAdmin`（项目一侧）不改；P5a 的 `membershipEnd`、P5b 的 `EndMember` 不用（停用连已忽略的邀请一起删除，3.6 的表；P5a review 第 6 节）。

### 2.3 存储（Task 1；3.6 约定六、3.7 规则 2、3.8、3.9）

五条语句，写进 `workspace/adapter/postgres/queries/deactivation.sql`，都在 `ctx` 带着的事务里执行（停用的事务，`identity` 开的，已持有账户行的 `FOR NO KEY UPDATE`）。计划的五条里两条删除邀请（`DeleteInvitationsTo`，和预检 M1 的修法、称为第五条语句的 `DeleteInvitationsOfWorkspacesLeftEmpty`）在修订一轮先由一条锁的语句 `LockInvitationsToDelete` 按 id 锁住（终审的 I1），再照裁定 F-1 并成一条只删锁住的 id 的 `DeleteInvitations`：两条删除的谓词搬进锁的语句，只写在那里（第 3 节第 20 条）。下文说的"第五条语句"指那一组谓词：

```sql
-- name: LockMemberWorkspaces :many
SELECT w.id
FROM workspaces w
WHERE w.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM workspace_members m
              WHERE m.workspace_id = w.id AND m.member_id = $member_id AND m.is_active AND m.deleted_at IS NULL)
ORDER BY w.id
FOR NO KEY UPDATE;

-- name: SoleAdmin :one
SELECT EXISTS (
    SELECT 1 FROM workspace_members m
    WHERE m.workspace_id = ANY ($workspace_ids::uuid[]) AND m.member_id = $member_id AND m.role = 20
      AND m.is_active AND m.deleted_at IS NULL
      AND NOT EXISTS (SELECT 1 FROM workspace_members a
                      WHERE a.workspace_id = m.workspace_id AND a.member_id <> m.member_id AND a.role = 20 AND a.is_active
                        AND a.deleted_at IS NULL)
      AND EXISTS (SELECT 1 FROM workspace_members o
                  WHERE o.workspace_id = m.workspace_id AND o.member_id <> m.member_id AND o.is_active
                    AND o.deleted_at IS NULL));

-- name: LockInvitationsToDelete :many
SELECT i.id
FROM workspace_member_invites i
WHERE i.deleted_at IS NULL
  AND (i.email = $email
       OR (i.workspace_id = ANY ($workspace_ids::uuid[]) AND i.responded_at IS NULL
           AND NOT EXISTS (SELECT 1 FROM workspace_members o
                           WHERE o.workspace_id = i.workspace_id AND o.member_id <> $member_id AND o.is_active
                             AND o.deleted_at IS NULL)))
ORDER BY i.id
FOR NO KEY UPDATE;

-- name: DeleteInvitations :exec
UPDATE workspace_member_invites
SET deleted_at = $now::timestamptz, updated_at = $now, updated_by_id = $deleted_by::uuid
WHERE id = ANY ($ids::uuid[]) AND deleted_at IS NULL;

-- name: EndWorkspaceMemberships :exec
UPDATE workspace_members
SET is_active = false, updated_at = $now::timestamptz, updated_by_id = $ended_by::uuid
WHERE workspace_id = ANY ($workspace_ids::uuid[]) AND member_id = $member_id AND is_active
  AND deleted_at IS NULL;
```

- `Store.LockMemberWorkspaces(ctx, userID) ([]uuid.UUID, error)`：锁在排好序的行到来时取，所以顺序是 id 的；等过锁之后 PostgreSQL 在行的最新版本上重新判断 `deleted_at IS NULL`，其间被删除的工作区少返回一行，不是错误（复核 spike 15；那里的成员关系已随删除结束，3.9）。
- `Store.SoleAdmin(ctx, workspaceIDs, userID) (bool, error)`：他是其中某个"还有别的有效成员"的工作区唯一的有效管理员。`role = 20` 是"管理员"这个集合；只有他一人的、另有有效管理员的不算；已结束、已删除的管理员和成员都不算。
- `Store.LockInvitationsToDelete(ctx, workspaceIDs, userID, email) ([]uuid.UUID, error)`（修订一轮：终审的 I1 加它，裁定 F-1 让它回答 id；第 3 节第 20 条）：以一条语句按 id 升序锁住（`FOR NO KEY UPDATE`）停用要删除的全部邀请，持有到事务结束，按这个顺序回答它们的 id。谓词是计划的两条删除的并集，邀请未删除，并且：(1) 发给这个邮箱，每个工作区、待接受的和已忽略的都算（已接受的在接受时已删除；Plane `views/user/base.py:313`），按邮箱查用 P3 为此建的 `workspace_member_invites_email_idx`（P3 review 第 6 节）；或 (2) 问到的工作区中他之外没有有效、未删除成员的那些的待接受（未回应）邀请（第五条语句的谓词，负责人对预检 M1 的裁定 (a)）：在他的成员关系结束之前执行，"之外"不算他自己；只有他一人的工作区在他停用之后没有有效成员，也没有一份邀请能让人进来（设计 3.7）：被接受的邀请会给一个没有管理员的工作区一位成员；已忽略的不算（不能再被接受，3.8 留着它的理由同样成立）。(2) 的谓词逐个：问到的工作区（锁住的）、未回应；别的成员的相关（同一工作区）、"之外"、有效、未删除。锁在排好序的行到来时取，所以顺序是 id 的；等过锁之后 PostgreSQL 在行的最新版本上重新判断谓词，其间被删除的邀请跳过，不是错误。为什么一条语句、为什么回答 id：发给他邮箱的邀请多在他不锁的工作区里，另一个停用可能持有它们（那个停用清空的工作区的待接受邀请）；每个停用都在写邀请之前按 id 升序取它要写的全部，之后只写这些，两个停用不在邀请的行上互相等成环。按两条删除的先后各取各的，正是 I1 的环；锁住之后仍按谓词删除，会写到锁之后才提交、没有按 id 取过的邀请，正是修订一轮第一步余下的环（裁定 F-1）。(2) 的那些在工作区的 N 之下：创建邀请持工作区的 S、接受持 N，停用持着这组工作区的锁时，它们和他之外的有效成员都不会变。发给他邮箱的不在这组工作区的锁之下：第一稿的一句"这组工作区和它们的邀请在停用持锁时不会变"只对这组工作区的邀请成立，终审的 I1 由此而来；锁之后才提交的一份不在回答里，留着，与停用之后才建立的一份等价（第 3 节第 8 条 (a)）。
- `Store.DeleteInvitations(ctx, ids, by, now) error`（裁定 F-1，替代计划的 `DeleteInvitationsTo` 和第五条语句）：软删除这些 id 中未删除的邀请，由给的账户、在给的时刻，不写别的行；已删除的保留原来的时刻（锁住的行在停用持锁时不会被别人删除，这一条只在给了已删除的 id 时起作用）。它在工作区的 N 之下、按全局顺序在成员行之前（邀请在 `workspace_members` 之前）；`Deactivator` 给它的正是锁的回答。
- `Store.EndWorkspaceMemberships(ctx, workspaceIDs, userID, by, now) error`：只结束有效、未删除的；行和角色留着，已结束的（等锁时被移出的）保留原来的结束者和时刻；`workspaceIDs` 里他已没有有效成员关系的工作区不是错误。
- 端口 `workspace/app.AllMembershipsEnder`（`ports.go`）有这五个方法（`LockInvitationsToDelete`、`DeleteInvitations` 是修订一轮的，替代计划的两条删除），是 `Deactivator` 的存储；`*postgres.Store` 实现它。
- 测试（`adapter/postgres/deactivation_test.go`）：`TestLockMemberWorkspaces`（bob 在 acme、gamma 有效：锁住这两个，`FOR SHARE` 等它们、外键的 `FOR KEY SHARE` 不等；beta 已结束、delta 已删除的成员关系、gone 已删除的工作区、epsilon 只有 carol、zeta 他不是成员的，都不锁）、`TestLockMemberWorkspacesLocksInIDOrder`（id 小的 web 在表里排在 alpha 之后；alpha 被持有时 `LockMemberWorkspaces` 已持有 web）、`TestLockMemberWorkspacesLeavesOutAWorkspaceDeletedWhileItWaited`（acme 在它等锁时被删除：回答只有 beta）、`TestSoleAdminOfAWorkspace`（17 个情形，三个一次问两个工作区，一个不问任何工作区；修订一轮加"一位访客，另有一位成员、没有管理员"（停放项 P2：`m.role = 20` 对访客也有决定它的一行）；各情形的工作区并排存在，问到不该问的工作区的实现答错）、`TestDeleteInvitations`（`deactivation_invitations_test.go`，裁定 F-1：给它 bob 的 acme 待接受、beta 已忽略、acme 以前已删除的三份，前两份由给的账户（grace，不是写过它们的 alice）、在给的时刻删除，已删除的保留时刻；没给它的、gamma 那一份发给他邮箱的，和 carol 的，每一列不动；先在一个回滚的事务里运行，每一行都不变：它在调用者的事务里）；邀请的锁的测试（`deactivation_invitations_lock_test.go`）在一个库里用计划的两条删除的世界，修订一轮把它们提成 `invitationsToBob`（他三份：acme 待接受、beta 已忽略、gamma 他不是成员的；以前已接受、已删除的两份；carol 的、`rebob@` 的）和 `newLeftEmpty`（一次问 acme（只有他）、beta（carol 是有效的访客，修订一轮由成员改为访客：访客也是有效成员，停放项 P1）、gamma（carol 已结束）、delta（carol 的成员关系已删除），每处一份 alice 发给 dave 的待接受邀请；acme 已忽略的、先前已删除的，和不问的 zeta（没有有效成员）的待接受邀请）：`TestLockInvitationsToDeleteLocksWhatItReturns`（回答按 id 升序恰是应删的那些，每个谓词各由一行决定：发给他邮箱的已删除的和 frank 的（邀请未删除）、carol 的和 `rebob@` 的（邮箱）、zeta 的（问到的工作区）、erin 已忽略的（待接受）、beta 的（别的成员的相关、访客也算）、acme 的（之外）、gamma 的（有效）、delta 的（未删除）；集合形式在 acme 答错；它在自己的事务里恰持有回答的那些，`FOR SHARE` 等它们、外键的 `FOR KEY SHARE` 不等，别的邀请都不锁）、`TestLockInvitationsToDeleteLocksInIDOrder`（id 小的是 acme 发给 dave 的、(2) 的，在表里排在 omega 发给他邮箱的、(1) 的之后；后者被持有时它已持有前者：按表的顺序或先取 (1) 的都先碰到后者；回答也按 id）、`TestLockInvitationsToDeleteLeavesOutAnInvitationDeletedWhileItWaited`（omega 的那一份在它等锁时被删除：回答只有 sigma 的那一份，`DeleteInvitations` 删除它，omega 的照删除留下）、`TestAnInvitationCreatedAfterADeactivationsLockIsNotItsToDelete`（`deactivation_invitations_window_test.go`，裁定 F-1 关掉的窗口：bob 的停用锁住 acme 发给 carol 的 x 之后，一份发给他邮箱、id 比 x 小的 z 在 gamma 提交；carol 的停用按 id 锁住 z、等 x；bob 的只删 x，提交；carol 的跳过其间被删除的 x、删 z：两个都完成，x 由 bob、z 由 carol 删除；按谓词删除时 bob 会等 z，40P01）、`TestEndWorkspaceMemberships`（acme 的成员、gamma 的管理员各结束，角色不变，由 bob，虽然这两行之前由 alice 写（清扫 14）；beta 已由 alice 结束的保留结束者；已删除的、carol 的、delta 不问的每一列不动）；`failures_test.go` 的 `TestAFailedReadIsAnErrorNotAnAnswer` 加两个读，修订一轮加 `LockInvitationsToDelete`（失败时不回答 id，正确的回答不是空的），`TestAFailedWriteIsAnError` 加计划的三个写（清扫 19），修订一轮之后是 `DeleteInvitations`、`EndWorkspaceMemberships` 两个；两个等锁的测试的 goroutine 在 10 秒期限的 context 上运行（清扫 29）。五条语句都是整个集合上的 `UPDATE`、`EXISTS` 或带 `ORDER BY` 的读，没有"取第一行"：物理行序不决定结果（清扫 1 的"两种行序"不适用，锁的顺序另由两个 `…LocksInIDOrder` 在逆序的堆上核对）。

### 2.4 `project.NewCascade`（Task 2；6.3、6.6，设计 1043）

```go
type CascadeDeps struct {
	Pool *pgxpool.Pool
}

func NewCascade(d CascadeDeps) Cascade
```

- `project/module.go`：`NewCascade` 在连接池上建 `app.NewCascade(store, store, store)`，返回模块已导出的 `Cascade` 接口；`New` 经 `NewCascade(CascadeDeps{Pool: d.Pool})` 建出同一个实现，`Module.cascade` 的类型是 `Cascade`。没有别的构造：命令行只要连带，不建 HTTP 一侧（6.6）。
- `Cascade.EndMemberships` 的说明补上"跨这些工作区按项目 id 锁住"。`TestLockActiveMemberProjectsLocksInIDOrder`（`project/adapter/postgres/end_test.go`）改为跨两个工作区：Web（acme）、Alpha（beta）、Zed（acme）按 id 升序，在表和名称、标识的索引里排成 Alpha、Zed、Web；Alpha 被持有时，它已持有 Web、还没有 Zed（按行序的会先碰到 Alpha、什么都不持有；按工作区先后的会连 Zed 也持有）。
- 七个交错测试文件里只给 `Pool` 的 `project.New(project.Deps{Pool: …}).Cascade()` 换成 `project.NewCascade(project.CascadeDeps{Pool: …})`（P4a review 第 6 节 S5：P5、P6 的命令行不照抄这种部分构造；测试同样不再用）。

### 2.5 `workspace` 的 `Deactivator`（Task 3；3.3、3.6 约定六、3.7 规则 2、3.8、3.9）

`workspace/app/deactivate_memberships.go`：

```go
type Deactivator struct { /* memberships AllMembershipsEnder; projects ProjectCascade; clock Clock */ }

func NewDeactivator(memberships AllMembershipsEnder, projects ProjectCascade, clock Clock) *Deactivator
func (d *Deactivator) DeactivateMemberships(ctx context.Context, userID uuid.UUID, email string) error
```

- 顺序（调用记录由单元测试核对整个序列）：`LockMemberWorkspaces(userID)` → `SoleAdmin(workspaces, userID)`（为真时 `domain.ErrSoleAdmin`，什么都不写，邀请也不锁）→ `LockInvitationsToDelete(workspaces, userID, email)`（要删除的邀请，按 id 升序，回答它们的 id；修订一轮加的，第 3 节第 20 条）→ `clock.Now()`（一次，在这把锁之后，也就在最后一把工作区锁之后，3.3）→ `DeleteInvitations(锁的回答, userID, now)`（裁定 F-1：只删锁住的那些）→ `EndWorkspaceMemberships(workspaces, userID, userID, now)` → `projects.EndMemberships(workspaces, userID, userID, now)`（一次，跨这些工作区；`project.sole_admin` 原样返回）。每一步的失败原样返回，之后什么都不再运行；`identity` 的事务随之回滚，前面的写一起撤销。
- 写者是他本人（`by = userID`），两条路相同：服务器管理员不是账户（第 3 节第 11 条）。
- 邀请在成员行之前、项目在工作区之后：全局加锁顺序 `users` → `workspaces` → `workspace_member_invites` → `workspace_members` → `projects` → `project_members`（3.6）；邀请的行由一条语句按 id 升序取，在写任何一份之前，之后只写取到的那些（第 3 节第 20 条，裁定 F-1）。工作区一侧没有任何增长能在他的账户行被持有时提交（约定六），所以在调用时列举就够；项目一侧的增长先取工作区的 S（方案 E），在工作区的 N 之下也不能提交，`EndMemberships` 在调用时列举（第 3 节第 5 条）。
- 两个码的说明（`workspace/domain/errors.go`、`project/domain/errors.go`）改为不说"谁去做"，对每个收到它的调用者都成立（清扫 30；P5b review 第 6 节）：

  | 码 | 说明 |
  |---|---|
  | `workspace.sole_admin` | The workspace would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has other active members. It must first be given another admin, or be deleted. |
  | `project.sole_admin` | The project would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has other active members. It must first be given another admin, or be deleted. |

  收到它们的：离开的本人（规则 1）、移出他的工作区管理员（P5a）、停用的本人（接口）、执行 `nerve users deactivate` 的服务器管理员（他不是任何工作区的成员，"把另一位成员设为管理员"他做不到，原来的工作区一句对他不成立）。说明不替调用者选路：给另一位管理员有几条路（提拔、添加、加入），删除也是。前端的两份文案（`auth.json`）说给页面的调用者，不变（第 3 节第 17 条）。
- 单元测试（`app/deactivate_memberships_test.go`）：`TestDeactivateMembershipsEndsEveryMembership`（bob 在 acme 是成员、在 beta 是访客，两个工作区按 id；carol 已没有有效成员关系，调用照样到达，邀请照样删除；假的锁回答测试自己取的两个 id，删除收到的正是它们，按那个顺序：删除的参数是锁的回答，不是另算的）、`TestDeactivateMembershipsRefusesTheOnlyAdmin`（alice，acme 唯一的管理员：`workspace.sole_admin` 本身，在邀请的锁之前，时钟没有读、什么都没写；修订一轮去掉了"事务的函数返回的正是它"：`deactivating` 返回的就是 `WithinTx` 的回答，这一句不会失败，停放项 P5）、`TestDeactivateMembershipsFailsAtEachStep`（六步各失败一次，邀请的锁和删除在内，另有项目一侧的 `project.sole_admin`：回答的第一个 `*shared.Error` 就是注入的那个，调用记录停在那一步，没有重试）；`clock_test.go` 的 `TestEachWriteReadsTheClockUnderItsLock` 加一行，时钟在邀请的锁之后读；假实现 `fakes_deactivation_test.go` 照存储的包装返回失败；`project.sole_admin` 的替身在 `fakes_projects_test.go` 只有一份，停用和移出的测试共用（停放项 P6）。

### 2.6 `identity` 与接线（Task 4；3.9、6.2、6.6、9.4）

- `identity/app/ports.go`：`LockedAccount` 加 `Email`（锁下读到的地址）；新端口

  ```go
  type MembershipDeactivator interface {
  	DeactivateMemberships(ctx context.Context, userID uuid.UUID, email string) error
  }
  ```

  `LockUserForCredentials` 多读 `email`（`queries/users.sql`），`Store.LockForCredentials` 把它放进 `LockedAccount`。
- `identity/app/deactivate.go`：`DeactivateDeps.Memberships`；`Execute` 把 `Lock` 回答的 `account.Email` 传下去，`ExecuteByEmail` 传它锁住那一行用的规范化地址（按邮箱锁住，锁下的邮箱就是它：其间改了地址的，按旧地址的锁读不到行，`identity.account_not_found`）；`deactivate(ctx, id, email, now)` 依次写账户、新手引导、会话，最后调用 `Memberships.DeactivateMemberships(ctx, id, email)`，它的拒绝或失败原样返回、整个停用回滚，成功时才记日志。`identity` 自己的时刻照 M2 在事务之前读，写账户、新手引导、会话；成员关系、邀请的时刻由 `Deactivator` 在锁之后读（第 3 节第 1 条）。
- `identity/module.go` 的 `Deps.Memberships`、`admin.go` 的 `AdminDeps.Memberships`：两处构造都把它交给 `NewDeactivate`。
- `api/modules/identity.yaml`：`deactivateMe` 的 `x-problem-codes: [workspace.sole_admin, project.sole_admin]`（11.7 的前缀规则），描述写明删除发给他地址的全部邀请和他是唯一有效成员的工作区的待接受邀请、结束全部成员关系（行和角色留着）、这些在一个时刻一个事务、两个拒绝什么都不改，以及成员关系怎样回来：服务器管理员的 `reactivate-member` 逐个工作区，或接受工作区管理员的新链接，项目成员关系随加入、添加回来（预检 L3.2：第一稿只写了 `reactivate-member`，还读得像连项目成员关系一起恢复）。描述在 `schema.gen.ts` 里是一行，关键词守卫的 `project-invitations` 规则按行匹配 `project.*invit`（M1 设计 2.2）：`project` 之后不再出现 `invit…`。
- 测试：`identity/app/deactivate_test.go`（`TestDeactivate` 的调用记录以锁下的地址 `alice@locked.example` 调用成员关系的一步，在会话之后；`TestDeactivateWhenAWriteFails`、`TestDeactivateByEmailWhenAWriteFails` 加成员关系的拒绝和失败：原样返回、第一个 `*shared.Error` 不变、一个事务、不记日志，调用记录是 `deactivateCalls`、`deactivateByEmailCalls` 截到失败那一步（失败的假实现不记它自己；成员关系一步记它的一次），之后没有调用、没有重试（预检 L1：第一稿不比较调用记录，`pf-id-runs-after-failure` 活下来）；修订一轮 `identity` 的 `fakeTx` 记下函数的回答（`returned`，照 `workspace` 的），两个测试的失败各行核对回答就是事务回滚所依的那个（停放项 P8：审查的 `commit_then_err` 先提交、再答出那一步的失败，第一稿在单元一层活下来，现在失败）；`TestDeactivateByEmail` 以规范化的地址调用）；`identity/adapter/http/me_test.go` 的 `TestDeactivateMeProblems`（401、两个 409 经 `fakeDeactivate.err` 透传、包装过的照样、失败 500，回答逐字；`identity` 的 `apitest.Main` 两个方向通过，9.4）；`credentials_test.go` 的 `TestLockForCredentialsReadsTheRow` 读出 alice 的地址（先存的 bob 的不是）。
- `workspace/deactivator.go`：导出的接口 `workspace.Deactivator`（一个方法）、`DeactivatorDeps{Pool, Clock, Projects}`、`NewDeactivator(DeactivatorDeps) Deactivator`；`workspace.New` 经它建出自己的，`Module.Deactivator()` 交出（第 3 节第 2 条）。
- 组合：`bootstrap/app.go` 给 `identity.New` 的 `Memberships: ws.Deactivator()`（6.6 的第 5、6 步，顺序不变）；`bootstrap/users.go` 给 `identity.NewAdmin` 的 `Memberships: workspace.NewDeactivator(workspace.DeactivatorDeps{Pool: pool, Clock: clock.System{}, Projects: project.NewCascade(project.CascadeDeps{Pool: pool})})`。`archtest/composition_test.go` 的 `TestCommandsComposeNoServerAndNoJobs`：`Users` 到达 `identity.NewAdmin`、`workspace.NewDeactivator`、`project.NewCascade`，到达不了 `workspace.NewAdmin`；`Workspaces` 到达 `workspace.NewAdmin`，到达不了停用的两件（`reactivate-member` 不调连带，6.6"只建用得到的"）。
- 交错 8（`bootstrap/interleaving_test.go`）和 19（`interleaving_answers_test.go`）换成真实的停用：`deactivating(pool, sessions, memberships)`（`bootstrap/deactivating_test.go`）是 `nerve users deactivate` 照 `users.go` 的接法（`identity` 的存储、`workspace.app.NewDeactivator`、`project.NewCascade`），会话、成员关系两处可以换成被 gate 停住的；`endedHoldingAll` 停在结束他的工作区成员关系之后、项目一步之前。P3 的 `gatedDeactivation`（测试的会话按 3.9 的顺序取锁）删除，不与真实的并存（P3 review 第 6 节）。19 的两种顺序都核对：停用最后写邀请和成员关系，由他、在一个时刻。忽略在先时，邀请最后一次之前也由 bob 写（忽略的写），"由停用写"由时刻看出：邀请的 `updated_at` 等于它的 `deleted_at`，即他的成员关系结束的时刻（预检 L7；`l7-dit-keeps-declined` 在这里失败）。修订一轮：bob 的成员关系由 `join` 作为他自己的建出（停放项 P19），测试先盖上 alice 的写者戳，成员关系上的"由他"可以失败（`EndWorkspaceMemberships` 保留写者的变体在这里失败）；没有邀请被删除时时刻一列是 false 而不是 NULL，失败由测试自己的诊断说出（停放项 P9）；回答比较链上第一个问题（`sameOutcome` 比较它的类别、码和说明，交错 3、7、12 同样，停放项 P17）。

### 2.7 两条路的组合测试（Task 5；9.3）

- `bootstrap/deactivation_world_test.go`：`deactivationWorld` 是 P5a 的 `endingWorld` 之后经接口建出：gamma（carol 的工作区），dave 接受她的邀请、再被她移出；她邀请 bob 的地址到 gamma，bob 忽略；alice 把 dave 改为 acme 的管理员、再移出他（已结束的管理员）；alice 把 bob 改为 beta 的管理员（他的角色在各处不同）；bob 归档 Solo；Docs 是 alice 在 acme 的项目，建在 beta 的 Lab 之后，bob 加入它（他的项目 id 跨两个工作区交错：Web、Ops、Solo 属 acme，Lab 属 beta，Docs 又属 acme）。`preconditions` 核对决定停用结果的行：`acme bob 15, acme dave 20 ended, beta bob 20, gamma dave 15 ended; Solo archived; gamma declined`。
- `deactivationPath{name, sent}` 和 `byAPI`、`byCommand`：一个是带 bob 的令牌的 `deactivateMe`（核对契约，问题的状态与码相配），一个是 `Users(…, DeactivateUser(addr))`，地址以大写、两边带空格输入（命令规范化它，清扫 22）；`sent` 在后台开始、不在它的 goroutine 上调用测试（清扫 33），回答在测试的 goroutine 上等。
- `TestARefusedDeactivationChangesNothing`（两条路 × 五个情形，各一个世界）：alice（acme 唯一的有效管理员，dave 已结束的管理员不算）`workspace.sole_admin`；bob（Ops、Lab 唯一的管理员）`project.sole_admin`；alice 加入 Lab 之后照样（Ops 里 erin 已结束的管理员不算）；她改为加入 Ops 之后照样（Lab 属 beta，检查跨工作区）。每次每张表每一行不变（`tableRows(riversOwn)`）；第五个情形：alice 离开 beta 之后，bob 是 beta 唯一的有效管理员（carol 是成员），`workspace.sole_admin`，虽然他按 id 的第一个工作区 acme 有 alice：检查问他的每一个工作区，工作区的拒绝在项目的之前（清扫 24）。拒绝在账户、资料、会话的写之后，项目一侧的拒绝在删除邀请、结束工作区成员关系之后，所以每一处都显示为回滚。
- `TestADeactivationRefusedAtItsCommitChangesNothing`：对停用写的每张表（`users`、`profiles`、`auth_sessions`、`workspace_member_invites`、`workspace_members`、`project_members`）各让提交失败一次（`refusingCommits`）：回答是失败（不是契约的问题），每张表不变：没有一步在自己的事务里写。
- `TestADeactivationEndsEveryMembership`：bob 的 2 个工作区成员关系、5 个项目成员关系（Solo 已归档也在）、3 份邀请（acme、beta 待接受，gamma 已忽略），每行先由测试盖上 dave 写的戳（清扫 14 的"由谁"可以失败）；停用之后每一行恰好变了 `is_active`（或 `deleted_at`）、`updated_at`、`updated_by_id` 三列，时刻是同一个、不早于请求，写者是 bob，角色不变（他的角色在各处不同，清扫 26）；他的账户无效；别人的每一行每一列不变（`rowsBut`）；账户的 `updated_at`（`identity` 在事务之前读的时刻）不晚于成员关系的时刻（第 3 节第 1 条的两个时刻的先后，预检 L2）。再停用一次：接口 401（会话已撤销）、命令照样完成，都不再写他已结束、已删除的行，也不写别人的任何一行（`rowsBut`，修订一轮加，停放项 P10：工作区集合为空时结束所有有效成员关系的变体在这里失败）。之后 carol 的停用完成：她是 gamma 唯一的有效管理员，也是唯一的有效成员（dave 已结束，规则 2 不拒绝）。
- `TestADeactivationDeletesTheInvitationsOfAWorkspaceItEmpties`（两条路；预检 M1 的反例，由第五条语句的谓词关闭）：erin 接受 carol 的邀请、再被她移出，gamma 只剩 carol；她再邀请 dave、erin（两份由测试盖上 alice 的写者戳），又邀请 frank 并删除那一份；她的停用之后：两份邀请在她的成员关系结束的时刻由她删除，其余每一列不变；她之外的每一行不变（bob 已忽略的、frank 已删除的、acme 和 beta 的邀请都在内）；dave、erin 的接受答 404 `workspace.invitation_not_found`，gamma 没有有效成员。之后服务器管理员 `reactivate-member` 两人：gamma 有两位成员、没有管理员（第 3 节第 8 条 (b) 的已知限制），dave 的停用照样完成：规则 2 只数管理员（`s1-sole-role` 在这里失败）。
- 命令的输出：`deactivated <邮箱>: revoked <n> sessions and ended its memberships; to bring it back, run nerve users activate, then nerve workspaces reactivate-member in each workspace`（`bootstrap/users.go` 的 `DeactivateUser`，8.7），拒绝时打印问题的说明、退出码 1（`cmd/nerve/users_test.go` 的 `TestUsersCommandsFail` 加 nia 一行）；`nerve users deactivate`、`activate` 的说明写明结束成员关系和回来的路：`activate` 的一句是 "its memberships stay ended: nerve workspaces reactivate-member restores one workspace's, as does accepting a new invitation to it"（`cmd/nerve/users.go`；预检 L3.1：第一稿说成员关系一直结束到 `reactivate-member`，而接受新邀请同样恢复）。`commandInBackground` 是命令在后台运行的共用一步（`reactivation_races_test.go`，P5a 的 `reactivatingBobWith` 改用它）。

### 2.8 锁、竞争、连接、中断（Task 6；清扫 8、9、12、13、29、31）

- `TestEachLockOfADeactivationIsItsStrength`（两条路）：carol 另建 able（bob 加入，在 acme、beta 之后建、slug 最前）和 delta（bob 加入后被她移出）；bob 建 kappa（只有他）并邀请 erin（`bobsAlone`）；修订一轮（终审的 I1、停放项 P12、P13）：之后 carol 再邀请他的地址到 delta，这一份 id 比 kappa 的大，经 `writtenAgain` 重写 kappa 的一份之后在表里排在 delta 的之前；七个事务以 `FOR SHARE` 持有他的账户、beta、kappa 的邀请、delta 的邀请、他在 beta 的成员关系、Lab、他在 Docs 的成员关系（gamma 的邀请，他忽略的那一份，id 最小，不持有）；acme 的行、Lab 的行和成员行、kappa 的邀请先重写一遍，每条 `UPDATE` 核对写了几行，之后按 `ctid` 读出的顺序作为前提核对：beta 在 acme 之前、Docs 在 Lab 之前、他在 Docs 的成员行在 Lab 的之前、delta 的邀请在 kappa 的之前（不带 `ORDER BY` 的语句按表的顺序会先碰到 beta、Docs、delta 的邀请，按 slug 的索引会先碰到 able：原型上的计划经 slug 的部分索引读工作区，`EXPLAIN` 核对过）。逐个放开，停用依次等在 `users`、`workspaces`、kappa 的邀请、delta 的邀请、`workspace_members`、`projects`、`project_members` 上；每一步的探针 `pgtest.WaitForLockWaitBehind` 认的是被那个持有者挡住的等待（`pg_blocking_pids`），等的是哪一行由持有者定，不会被上一步残留的、同一张表上的等待满足（停放项 P12）。每一步 `lockOn` 读出十六行的最强的锁，与 `state(took, shared)` 比较：停用已取的各是 `FOR NO KEY UPDATE`，不更强、不更弱，持有者还持着的是 `FOR SHARE`，其余没有锁；gamma、delta 两个工作区从不加锁（他不是 gamma 的成员，在 delta 的成员关系已结束）；等 beta 时已持有 acme、还没有 able、kappa，也没有任何一份邀请；等 kappa 的邀请时已持有 beta、able、kappa 和 gamma 的邀请（id 更小）；等 delta 的邀请时已持有 kappa 的（一条语句按 id 取要删的邀请，之后的删除只写它们；计划的实现里写 delta 那份的 `DeleteInvitationsTo` 在写 kappa 那份的第五条语句之前，按删除的先后取的实现在这里先等 delta 的）；等他的成员关系时已持有 delta 的（邀请在成员行之前，全局顺序）；等 Lab 时已持有 Web、Ops、Solo（id 在 Lab 之前；第一稿的说明写了 Ops、Solo 而 `rows` 不读它们，预检 L3.5），还没有 Docs。完成的时刻不早于 delta 的邀请放开的时刻（时钟在邀请的锁之后读）、也不早于 beta 放开的时刻（最后一把工作区锁，3.3）。
- `TestTwoDeactivationsWithCrossedInvitationsBothGoThrough`（`deactivation_crossed_test.go`，修订一轮，终审的 I1；两种先后 × 两条路）：`newCrossedWorld` 全经接口：bob 建 acme、carol 建 gamma，各有一位接受了邀请又被移出的成员（dave、erin，已结束的成员不算有效成员）；bob 邀请 carol 的地址到 acme，carol 邀请 bob 的地址到 gamma。每个停用都要删两份：发给自己地址的（对方的工作区，它不锁）和自己工作区的待接受邀请（它留下的空工作区）；两份正是对方也要删的。第一个停用（`deactivating`）删除它的邀请之后停在 gate（`deletedInvitations`，修订一轮第二步之前停在删除发给自己地址的那一条之后）；第二个在一条路上开始，等第一个持有的邀请（`WaitForLockWaitOn`）；放开之后两个都完成：两个账户停用、成员关系结束、没有邀请留下；第二个的时刻不早于 gate 放开（它在等到的邀请锁之后读时钟）。按删除的先后取邀请锁的旧实现（`f-I1-dropped`：邀请的锁去掉，两条按谓词的删除放回，gate 照旧停在删除发给自己地址的邀请之后），每一次都有一个停用回滚（40P01，或第二个答 500），`-count=5` 二十次都失败。
- `TestAnInvitationCreatedAfterADeactivationsLockIsLeftToTheOther`（`deactivation_crossed_test.go`，修订一轮，裁定 F-1；carol 的停用走两条路）：在 `newCrossedWorld` 里 carol 先撤回她发给 bob 的邀请；bob 的停用（`deactivating`）锁住 acme 发给 carol 的 x 之后停在 gate（`lockedInvitations`）；测试接着提交 z，carol 发给 bob 地址的一份 gamma 的邀请，它的 id 在建世界之前取，比 x 小（前提核对）；carol 的停用按 id 锁住 z、等 x（`WaitForLockWaitOn`）；放开之后两个都完成，两个账户停用、成员关系结束、没有邀请留下，z 在 carol 的成员关系结束的时刻由她删除，x 由 bob。放回按谓词删除的实现（`f1-dropped`：2eb20f77 的存储、端口和 `Deactivator`）里 bob 会写 z、等 carol：每一次都有 carol 的停用回滚，`-count=5` 十次都失败。
- `TestAnInterruptedDeactivationChangesNothing`：他在 Lab 的成员行被持有，命令的配置给服务的请求期限 200 毫秒；命令过了两倍期限仍在等（没有自己的期限，17.4 G1 (a)）；中断（它的 context 结束，SIGINT、SIGTERM 的做法）之后失败、没有输出，每张表不变；放开之后再执行，完成。
- `TestTheDeactivationRunsOnItsTransactionsConnection`：命令的连接池只有一个连接：任何一条走池的语句都会等第二个连接到 5 秒的期限，命令失败；实际完成（清扫 8）。
- `TestADeactivationFindsWhatChangedMeanwhile`（三个情形 × 两条路）：测试先把 `endings` 读的每一行盖上 dave 的写者戳（数着行数），之后每个"由谁"都是这一回的写，不是这一行先前的写者（修订一轮，停放项 P11：结束、删除他的行的每条语句，停用的、移出的、删除的，保留写者的变体都在这里失败）；alice 的写持有一个工作区的锁、等一个被持有的行，停用等那个工作区的行（探针只由那一把锁的等待满足：她的写等的是另一张表的行，不持 `workspaces` 的元组锁，一条语句只在等的时候持元组锁；删除 acme 先写了 acme 的行，没有等它，停放项 P14 改了第一稿"两方都不写 `workspaces`"的说法）；放开之后：删除 acme（持 acme 的 N）：停用跳过 acme，不答 404、不失败，结束其余的，acme 和它的项目、邀请照删除留下的（复核 spike 15）；删除 Docs（持 acme 的 S，方案 E）：停用拿到 acme 之后才列举项目，Docs 已删除，照删除留下的；移出 bob（持 beta 的 N，已结束他在 beta 的成员关系、删除 beta 的邀请）：停用等到 beta 之后读到它们已结束、已删除，结束者仍是 alice（`endings` 给每一行写明最后写它的人，清扫 40）。
- `TestADeactivationReadsTheAddressUnderItsLock`：`nerve users set-email` 把 bob 的地址改为 `robert@example.com`，在撤销会话之前停在 gate、持着他的账户行；停用等账户行；改地址提交之后：`deactivateMe`（以改地址不撤销的 PAT）在锁下读到新地址，删除发给 `robert@` 的两份邀请，发给 `bob@` 的三份不动（`endings` 读的行同样先由 dave 盖戳，"由 robert"是停用的写）；`nerve users deactivate --email bob@example.com` 按旧地址锁不到行，`identity.account_not_found`，他什么都不变（复核 M5）。`setBobsEmail` 是这一步的共用写法（`interleaving_answers_test.go`，P3 的交错 12 同用）。

### 2.9 交错 7、16（Task 7；9.3，Codex S1）

- `bootstrap/interleaving_accepting_test.go`：`s1Race` 是 P3 的 `answerRace`（bob 被邀请到 acme 作管理员）加 Web（acme 的公开项目，alice 是管理员）；`former` 时 bob 在 acme 有以前的成员行（他自己的，接受时建的，修订一轮由 alice 改为他，停放项 P19；被 alice 移出过）；`newS1Race` 的四条种子语句在 `pgtest.Soon(t)` 的期限之内运行（清扫 29，预检 L5；修订一轮 bootstrap 的 `soon` 移到 `pgtest.Soon`，每个包的测试共用这一个有界的 context，停放项 P3、P4、P9、P16）。`standing` 一行写出他的账户、他在 acme 每一行（角色、结束者）、在 Web 的、邀请（已接受/未回答、删除者）和 alice 的角色；他在 Web 的一行在和他某一行 acme 的同一时刻结束时写"ended with acme"（修订一轮，停放项 P15：那一行由他自己的加入写，"由他"在那里不会失败，看的是时刻）。
  - `TestAcceptingAndDeactivating`（(a)、(d)、(e)，四个子测试）：接受先持账户行 `FOR SHARE` 和 acme 的 N 停在 gate，停用等账户行；提交之后停用停在拿到账户行之后自己的 gate（修订一轮加），测试把接受写的成员行盖上 alice 的写者戳，再放开；停用列举到 acme，结束他的成员关系（由他，`EndWorkspaceMemberships` 保留写者的变体在这里失败；alice 仍是管理员）。停用先持账户行停在 gate，接受等它，之后在锁下读到停用，401，没有成员关系；邀请由停用删除。以前的成员行：接受恢复那一行，(a) 结束它，(d) 照 alice 的移出留下它。接受的回答比较链上第一个问题（`sameOutcome`，停放项 P17）。
  - `TestAnAdmittedAdminsWriteAndHisDeactivation`（(b)、(c)）：接受先提交；停用停在账户行之后自己的 gate，测试这时把他在 acme 的成员行盖上 alice 的写者戳（修订一轮）；bob 另一个事务（b）把 alice 改为成员，持 acme 的 N 停在 gate，（c）加入 Web，持 acme 的 S 和他的成员关系停在 gate；停用继续，等 acme 的行（第 3 节第 6 条）。之后（b）bob 是 acme 唯一的有效管理员、alice 是成员：`workspace.sole_admin`；每张表在两方都停在 gate、都还没有提交写的时候读（`workspace_members` 除了 alice 那一行，改角色写它），拒绝之后不变（预检 L4：第一稿在放开停用之后才读）；（c）停用列举到他在 Web 的成员关系（锁住他的账户之后建的），和 acme 的在一个时刻结束（"ended with acme"，项目一侧保留 `updated_at` 的变体在这里失败），acme 的由他；acme 仍有 alice。
- `reactivation_races_test.go` 的 `TestReactivatingAndDeactivating`（交错 16）：恢复先（acme 被别的事务持有，恢复持他的账户 `FOR SHARE` 等它，停用等账户行；放开之后恢复完成，停用列举到 acme、再次结束，Web、Ops 仍结束）；停用先（停用持账户行停在 gate，恢复等它；停用提交之后恢复照样进行，3.11 的例外，输出提示 `nerve users activate`）。两种顺序的结果都由 `endersOf(t, pool, id)`（`reactivation_races_test.go`：一个账户每一个已结束的成员关系，工作区按 slug、项目按名字，各写明最后写它的人，按字节排序，同名的工作区在项目之前、再按 id，修订一轮加的次序，停放项 P18；清扫 39、40；交错 13–16 和两位管理员共用）核对：恢复先是 acme 由 bob 结束，Web、Ops 仍是 alice 的；停用先 Web、Ops 仍是 alice 的。
- `interleaving_answers_test.go`：`newAnswerRace(t, role)`（邀请的角色）、`deactivateBob(ctx, sessions, memberships)`；`interleaving_accepting_test.go`：`changeRole(ctx, pool, by, id, role, members, cascade)` 是 (b) 和 P4b 的降为访客（`interleaving_growth_test.go` 的 `demote`）共用的改角色。

### 2.10 交错 13、14、15 与两位管理员（Task 8；9.3，P5b review 第 6 节）

`bootstrap/interleaving_shrinking_test.go`，照 P5b 的 `TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize` 写：第二方等工作区的行，两方都不写 `workspaces`，探针只由那一把锁的等待满足；每个等待有期限；goroutine 上不调用测试。

修订一轮：`growthRace` 的两行成员关系各是成员自己的（建工作区、接受邀请那样，停放项 P19），交错 13、14、15 在动作之前把他在 acme 的成员行盖上 alice 的写者戳（`lastWrittenByAlice`，经 `stampWriter`），"acme bob"是停用的写；增长、建项目停在 gate 时 `sharesAcme` 读出 acme 和他的成员关系都是 `FOR SHARE`，不更强（`lockOn`，停放项 P25；`sharesAcme` 是这项检查唯一的一份，交错 4、5、17 同用）；动作之后另读 alice 的已结束成员关系，应为无（停放项 P22：项目一侧的 `EndMemberships` 去掉 `member_id` 的 `end_any_member` 在这里失败）；项目一侧的三种拒绝由一个解码 `refusedAt` 判断，`refusedAsAGuest`、`refusedAsNoMember` 调用它（停放项 P23）。

- `TestADeactivationAndTheProjectSidesGrowthSerialize`（交错 13、14：添加 / 加入 × 新的 / 以前的项目成员行 × 两种顺序，八个子测试）：增长先持 acme 和他的成员关系 `FOR SHARE` 停在 gate，停用等 acme 的行；增长提交之后停用在项目一步列举到 Web，结束（由他，时刻在 gate 打开之后读），acme 的也由他（`endersOf`）。加入的格子里 Web 的一行由他自己的加入写，"Web bob"的写者一半不会失败，那里由"Web 的时刻等于 acme 的"看出停用的写（停放项 P20，说明照此收窄；添加的格子由 alice 写，写者一半可以失败）。停用先持 acme 的 N、已结束他的成员关系（`endedHoldingAll`）；增长等 acme 的行，之后读到他已结束：添加 422 `members[0].member_id` `not_allowed`，加入 404；以前的 Web 行照 alice 的移出留下。两种顺序里第二方等待时都没有事务持有 Web，alice 的成员关系都不结束。
- `TestADeactivationAndACreationHeLeadsSerialize`（交错 15）：动作之前读出 bob 是 acme 的成员、在 Web 没有成员关系（停放项 P24）；建项目先持 acme 和他的成员关系 `FOR SHARE`；停用等 acme，之后列举到 Ops 并结束他在那里的（那一行由 alice 的建项目写）；停用先：建项目 422 `project_lead_id`，没有 Ops；两种顺序里 acme 的结束都由他（`endersOf`），alice 的成员关系都不结束。第一稿说明里"alice 是创建者也是管理员，所以他不是唯一的管理员"测试不读，也不是规则 2 不拒绝的理由（他独自在 Ops 也可以），删去（停放项 P25）。
- `TestTwoAdminsDeactivatedAtOnceLeaveAnAdmin`（规则 2 之下的两次停用，`memberWorld`，四个子测试）：测试先把 alice、gina 的成员关系（工作区 2 行、项目 3 行）盖上 dave 的写者戳（修订一轮，停放项 P20：alice 建 acme、Web、Ops 时写的是她自己）；acme 的两位管理员 alice、gina 同时停用；第一位在两个位置之一停住：她的规则 2 检查之后、写之前（`soleAdminCheckedHolding`，清扫 25、31：检查在一个事务、写在另一个的实现，第二位在这里不等 acme，探针失败），或她的成员关系结束之后；第二位等 acme 的行；第一位提交之后第二位是 acme 唯一的有效管理员，`workspace.sole_admin`，账户仍有效；acme 仍有管理员；第一位每一个已结束的成员关系（acme 和它的项目）都由她本人结束（`endersOf`，清扫 40；两边保留写者的变体在这里失败）。

### 2.11 端到端与文档（Task 9；第 2 节 W9，3.20、8.7）

- `e2e/stories/workspace/w9-deactivation.spec.ts`（W9 的接口版本，含命令）：B 建 acme、A 是成员；A 建 beta，B、C 是成员；B 在 beta 建 Lab、添加 C；C 建 gamma、delta，邀请 B，B 忽略 gamma 的；B 建 epsilon（只有他），邀请 C。B 是 acme 唯一的管理员：接口 409 `workspace.sole_admin`、命令同一句说明，每次之前读、之后核对六张表（清扫 38）。B 把 A 改为 acme 的管理员：仍是 Lab 唯一的管理员，`project.sole_admin`，同样不变。A（beta 的管理员）加入 Lab：接口停用 B，204；A12 的断言；他的六行（Lab、acme、beta、epsilon、两份邀请）都结束或删除，由他，在一个时刻（按 `COLLATE "C"` 排序，清扫 39）；他发给 C 的 epsilon 的邀请在同一时刻由他删除，C 接受它答 404 `workspace.invitation_not_found`（第五条语句）；别人的每一行不变。修订一轮（停放项 P27）：A 邀请 D 到 beta（B 离开之后仍有成员），C 邀请 D 到 gamma（B 不在那里），两份待接受的邀请发给另一个地址；"别人的每一行"连 B 已接受的 beta 的邀请（已删除的行）一起读：删除发给任何地址的邀请的、重写已删除邀请的变体在 W9 失败。`nerve users activate` 只恢复账户（成员关系仍结束），`reactivate-member` 恢复 acme 一处（B 读得到 acme，读不到 beta；他在 Lab 的成员关系仍结束，停放项 P28）；命令再停用一次：acme 的又结束，别人的不变。epsilon 的注释只说那份邀请（停放项 P26）：他在 epsilon 的成员关系由结束 acme、beta 的同一条语句结束，W9 已看到它的写者；邀请由第五条语句删除，它的写者由 Go 的组合测试核对。
- `a12-deactivate.spec.ts`：命令的新一行。
- README（8.7）：`deactivate` 一条加上结束成员关系、删除邀请（含他留下的空工作区的待接受邀请）、拒绝的情形和说明、没有请求期限、中断可以重试；拒绝时的补救由那里的管理员做，服务器管理员没有对应的命令（预检 L3.3）；`activate` 一条加"不恢复成员关系"；"恢复被移出的成员"加"账户停用"、回来的两步，和它不看工作区有没有管理员（第 3 节第 8 条 (b)）。差异清单第四节"停用账户"一行（3.20；第五条语句是与 Plane 的差异）。总体设计 4.2 约定六的停用一段写成实现之后的样子（3.20 的 P2 一行"停用一段在 M3/P6 随实现核对"）。M2 收尾交接第 6 节的处理结果（关闭；M4 交接第 1 节不提前）。

### 2.12 矩阵

不加矩阵的格子：`deactivateMe` 是 `/me` 的操作，没有工作区、项目的作用域，规则表里没有它的操作名，"谁可以"只有调用者本人（第 3 节第 10 条）。新表没有；`projectTables`、`workspaceLevelTables` 和完整性核对照旧成立，`table_rows_test.go` 和组合出的删除测试不加豁免照旧通过。

## 3. 与设计的差异和补充（已裁定）

每一条都已裁定。第 1、2 条改了设计写下的跨模块端口的签名和命令行组合的写法，第 8 条是清扫 36 报告的绕过，连同预检的 M1（第 8 条 (d)），由负责人 2026-10-04 裁定（"按建议"），设计 3.3、3.7、3.9、3.11、6.2、6.6、9.3 随 `e9289a4d` 改；第 20 条是终审的 I1，负责人 2026-10-04 裁定"A"，修订一轮落实，设计 3.6、3.9 随它改；其余各条由控制者 2026-10-04 裁定接受（第 17、19 条先暂定、待预检：第 17 条预检确认每个页面的调用者仍得到成立的补救；第 19 条见那一条）。预检的每一条发现（M1、L1–L8；没有高的）都由控制者同日接受、已落实，落点写在各条和附录 A。没有一条改变 6.6 的组合顺序、平台与 `shared` 的边界、负责人此前的裁定（第 10、11 节，17.4，P5b 的 I1 取 A）或 Phase 的划分。

1. **`MembershipDeactivator` 不带 `now`；一个停用有两个时刻**（负责人 2026-10-04 接受；设计 3.9 第一条和 3.3 随 `e9289a4d` 改）：3.9 写的端口是 `DeactivateMemberships(ctx, userID, email, now)`，而 3.3（第 176 行）和 P5a spec 第 5 节 P6 一行要求停用"先按 id 锁住他的工作区，之后读 `now`"。`identity` 不取工作区的锁，它的时刻照 M2（3.5）在事务之前读；把它传下去，成员关系的结束就带着锁之前的时刻：一个在停用等工作区的锁时提交的增长（交错 13、14 的增长先），会在停用结束它时显得比它晚建立。所以端口不带 `now`，由 `Deactivator` 在最后一把工作区锁之后读一次（2.5）。后果：一个停用有两个时刻，都在一个事务里：账户、新手引导、会话是 `identity` 的（M2 不变），邀请、工作区和项目成员关系是 `Deactivator` 的，不早于前者。另外两种写法：(a) 端口带 `now`，用 `identity` 事务之前的时刻：违反 3.3，`o-clock-early`（读在锁之前）在组合一层由 `TestEachLockOfADeactivationIsItsStrength`（时刻不早于 beta 放开）、`TestADeactivationAndTheProjectSidesGrowthSerialize`（结束晚于 gate 打开）失败，这种写法同样失败；(b) `identity` 在锁账户行之后再读时钟并传下去：仍在工作区的锁之前，同 (a)。建议（负责人接受）：采用不带 `now` 的端口，设计 3.9 第一条改为 `MembershipDeactivator.DeactivateMemberships(ctx, userID, email) error`，并加一句"它在锁住他的全部工作区之后读时刻（3.3）；账户、新手引导、会话的时刻照 M2 在事务之前读"。两者都读 `clock.System{}`；墙钟不回拨时，后者在前者之后读，所以不早于前者。这个先后由 `TestADeactivationEndsEveryMembership` 钉住：账户的 `updated_at` 不晚于成员关系的时刻（预检 L2；`identity` 的时钟快一秒的 `l2-cli-identity-ahead`、`l2-api-identity-ahead` 在组合一层失败）。
2. **命令行的组合用 `workspace.NewDeactivator`，不建 `workspace.NewAdmin`**（负责人 2026-10-04 接受；设计 6.2、6.6 随 `e9289a4d` 改）：6.2 的 `admin.go` 一行写"`NewAdmin(AdminDeps)`：命令行的建工作区、恢复成员和停用"，6.6 第 1101 行写 `nerve users` 加上"`workspace.NewAdmin`（它的 `Deactivator`）"。但 `workspace.NewAdmin`（P1、P5a）要 `identity.Provide` 的 `Accounts`、`project.Provide` 的 `ProjectMembershipCounts`、事务管理器和日志：在 `nerve users` 里建它，会建出没有一个 `users` 命令用得到的用例，违反同一节的"一个组合只构造它的命令用得到的部分"（6.6 第 1103 行）。停用的一步只要连接池、时钟和项目的连带，所以 `workspace` 导出 `NewDeactivator(DeactivatorDeps{Pool, Clock, Projects}) Deactivator`（`workspace/deactivator.go`），`workspace.New` 经它建出自己的（与 `project.New` 经 `NewCascade` 同一种写法），`Module.Deactivator()` 照 6.2 交出。服务器的组合顺序（6.6 第 5、6 步）不变。`TestCommandsComposeNoServerAndNoJobs` 钉住 `Users` 到达 `NewDeactivator`、`NewCascade`，到达不了 `workspace.NewAdmin`；`Workspaces` 反过来（`s4-users-builds-workspace-admin`、`s4-workspaces-builds-cascade`）。建议（负责人接受）：接受；设计 6.2 的文件树加 `deactivator.go  NewDeactivator(DeactivatorDeps)：停用的成员关系一步，命令行和 New 共用`，`admin.go` 一行改为"命令行的建工作区、恢复成员"；6.6 第 1101 行改为"加上停用要的 `workspace.NewDeactivator` 和 `project.NewCascade`"。
3. **移交的落点**（说明）：

   | 移交 | 落点 |
   |---|---|
   | P5a spec 第 5 节：`EndMemberships` 跨几个工作区调用一次 | Task 3（2.5）：`Deactivator` 一次调用，传他全部的工作区；Task 6 的锁的测试核对项目跨工作区按 id |
   | 同上：时刻在这些锁之后读 | Task 3（`TestEachWriteReadsTheClockUnderItsLock` 一行、调用记录）、Task 6（时刻不早于 beta 放开）；端口的签名见第 1 条 |
   | 同上：`TestLockActiveMemberProjectsLocksInIDOrder` 的顺序跨工作区同样成立 | Task 2（2.4）：改为跨两个工作区，按工作区先后的实现在它失败（`o-prj-per-workspace`） |
   | 同上：规则 2 跨工作区，另查工作区一级的唯一管理员 | Task 1（`SoleAdmin`）、Task 3；组合一层 Task 5 的四个拒绝（Lab 属 beta 一条） |
   | 同上：`project.NewCascade` 随 `nerve users deactivate` 加入 | Task 2（构造）、Task 4（命令行的组合，archtest） |
   | 同上：停用之后 `reactivate-member` 照样恢复（交错 16） | Task 7 |
   | 同上：`nerve users deactivate` 没有请求期限（G1 (a)），README 写明 | Task 6（`TestAnInterruptedDeactivationChangesNothing`）、Task 9（README） |
   | 同上、P5a review 第 6 节：账户行锁保持 `FOR NO KEY UPDATE` | 不改（`LockAccount`、`LockUserForCredentials` 照 M2）；Task 6 的锁的测试读出它（`o-account-share`、`o-account-update`）；结束经 `updated_by_id` 的外键取他自己 `users` 行的 `FOR KEY SHARE`，与 `FOR NO KEY UPDATE` 不冲突 |
   | 同上：停用不复用 `membershipEnd` | Task 1、3：自己的语句（计划的五条，修订一轮加邀请的锁，第 20 条）；`DeleteInvitationsTo` 连已忽略的一起删（`s15-dit-pending-only`：只删待接受的，在存储、组合、W9 失败） |
   | 同上：`EndMember` 要求有效、`ReactivateMember` 要求无效，调用者在工作区 N 下读到状态 | 停用不调用它们；`EndWorkspaceMemberships` 只结束有效的，在工作区 N 之下（`s1-ewm-active`：等锁时被移出的那一行被再写一次，组合一层失败） |
   | P5b spec 第 5 节、review 第 6 节：停用经 `EndMemberships`，不经 `EndMember` | Task 3 |
   | 同上：P5b 的写先取工作区 S，停用按 id 取工作区 N，在工作区行上串行；交错 13–15 照 `TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize` 写 | Task 8（2.10）；探针的反例 `s32-wait-moved` |
   | 同上：`project.sole_admin` 的文字对停用命令的调用者同样成立；命令的输出照 8.7 写下一步 | Task 3（两个码改写，2.5）、Task 5（输出） |
   | P4a review 第 6 节：`project.NewCascade` 替换交错测试里只给 `Pool` 的部分构造（S5） | Task 2 |
   | P4a、P4b review 第 6 节：P6 之前停用的账户仍可被添加为项目成员、指定为负责人 | 不再成立：停用结束他全部的工作区成员关系，添加、建项目在锁下读到无效（交错 13、15） |
   | P3 review 第 6 节：交错 19 换成真实的停用（删除 `gatedDeactivation`）；按邮箱删除用 `workspace_member_invites_email_idx`；交错 7 | Task 4（19）、Task 1（2.3）、Task 7（7） |
   | P1 review 第 6 节：交错 8 的成员关系一半 | Task 4（`interleaving_test.go`：停用在后时结束新工作区的成员关系） |
   | 设计的测试清单（9.3，第 1503–1511 行）：`identity` 的两个 409 透传；每个拒绝数据库完全不变；3.9 的另外两种情形 | Task 4（`TestDeactivateMeProblems`）；Task 5；Task 6 |
   | 一直有效的规则：锁之后读时钟；锁下重读确认父行；目录驱动的检查；`apitest.Main` 两个方向；端口的错误原样返回；角色按集合；关键词守卫；新的矩阵表 | `TestEachWriteReadsTheClockUnderItsLock` 一行（Task 3）；停用不按点名的父行写，锁的语句自己带 `deleted_at IS NULL`，上锁时已删除的不在回答里（2.3）；没有新表，`table_rows_test.go` 和删除测试不加豁免；`identity` 的 `apitest.Main`（Task 4），`workspace`、`project` 的照旧；清扫 2；`role = 20` 只在两条"管理员"的查询里；契约的新文字没有新的命中；`deactivateMe` 不进矩阵（第 10 条） |
   | M2 收尾交接第 6 节 | 关闭（Task 9 写处理结果） |
   | M4 的 M2 收尾交接第 1 节（"只投递"的 River 客户端） | 不提前：停用不投递任务（3.9、13.1），命令行的组合仍没有 River 客户端；Task 9 的处理结果和 review 写明 |

4. **任务的划分：9 个**（说明）：设计的任务与 plan 的对应：1（`identity` 的端口、邮箱、两个码、HTTP 测试）→ 4；2（`Deactivator`）→ 1（存储）、3（用例）：存储一个 Task，用例和它改写的两个码一个 Task；3（跨工作区的 `EndMemberships`）→ 2（P5a 的实现不改，Task 2 把它的锁顺序测试改为跨工作区，并加 `NewCascade`）；4（`nerve users` 的组合、组合测试）→ 4（接线、archtest）、5（两条路的组合测试）；5（交错 7、8，19）→ 4（8、19）、7（7）：`identity` 的改动让交错 8、19 原来手写的停用一方构造不出 `Deactivate`（少了 `Memberships`），它们只能在同一个 Task 里换成真实的停用，不写过渡版本；6（13–16）→ 7（16）、8（13–15）；7（两条路，spike 15，改邮箱）→ 5、6；8（W9）→ 9；9（文档；review）→ 9（review 是控制者的）。最大的 Task 是 Task 4（1,336 行），每个都在约 1,500 行以内；plan 共 5,617 行。
5. **项目上锁时已删除的跳过，在组合一层到不了**（说明）：方案 E 之下删除项目先取工作区的 S（P4b），停用先持他全部工作区的 N、再列举项目，所以组合出的 app 里没有"列举之后、上锁之前被删除的项目"。组合一层只有它的前一半：停用等工作区时被删除的项目，列举时已不在（`TestADeactivationFindsWhatChangedMeanwhile` 的 Docs 一条）；上锁时的跳过由 P5a 的存储测试 `TestLockActiveMemberProjectsLeavesOutAProjectDeletedWhileItWaited` 钉住（`o-prj-locked-deleted`，附录 A）。
6. **交错 7 (c) 的等待在工作区行上**（说明）：9.3 的 7 (c) 写"停用改成员行时等它"；方案 E（17.4）之下加入先取 acme 的 S、再锁他的成员行，停用在取 acme 的 N 时就等，到不了成员行。`TestAnAdmittedAdminsWriteAndHisDeactivation` 的探针认 `workspaces`。建议设计 9.3 的 7 (c) 改为"停用锁工作区时等它"（与 13、14 同一种说法）：控制者接受，设计随 `e9289a4d` 改。
7. **组合一层的失败注入在提交**（说明）：组合出的 app 里存储的语句不会失败；"每个端口调用注入失败"在单元一层（`TestDeactivateMembershipsFailsAtEachStep`、`TestDeactivateWhenAWriteFails`、`TestDeactivateByEmailWhenAWriteFails`，比较到失败那一步为止的调用记录：`identity` 的两个是预检 L1 之后才比较的），存储一层的失败原样返回由 `failures_test.go` 钉住；组合一层对停用写的六张表各让提交失败一次（`TestADeactivationRefusedAtItsCommitChangesNothing`）：每条语句都已执行，回滚撤销全部，没有一步在自己的事务里写。
8. **清扫 36：别的操作或两步操作到达同样的结果**（第一稿报告了 (a)–(c)，预检找出漏报的 (d)；负责人 2026-10-04 裁定"按建议"：(a)、(b) 是已知的限制，(c) 不处理，(d) 照修法 (a) 由第五条语句关闭）：
   - **(a) 回来的第三条路**（已知的限制，与 Plane 相同）：停用之后，工作区管理员可以再向他的地址发一份邀请；`nerve users activate` 之后他接受它，就回到那个工作区（以前那一行恢复，角色取邀请的，3.8），不经 `reactivate-member`。这要工作区管理员的一次邀请，与 Plane 相同；"回来只有 activate 加 reactivate-member"作为安全性质不成立，作为"服务器管理员一侧的命令"成立。邀请的建立与停用不串行（它锁工作区的 S 和邀请人的账户行，不锁被邀请的账户；停用只锁他是成员的工作区）：与停用同时建立的邀请，在停用的邀请锁的语句之后提交时留下（第一稿是 `DeleteInvitationsTo` 的语句之后，裁定 F-1 提前到锁语句之后，第 20 条），与停用提交之后才建立的等价。
   - **(b) `reactivate-member` 不看工作区有没有有效的管理员**（已知的限制）：它照 3.11 恢复停用账户的成员关系；服务器管理员在 `activate` 之前执行它，一个不能登录的账户就有了有效的管理员成员关系，别的管理员可以离开（规则 1 把他算作另一位），工作区只剩这个停用的管理员（P5a、P5b 第 7 节写过同一种状态）。它同样能把一个以前的非管理员成员恢复到 (d) 那样被停用清空的工作区：工作区有有效成员而没有有效管理员，没有人能邀请、移出、改角色（预检 M1 的第二条路、L8）；`TestADeactivationDeletesTheInvitationsOfAWorkspaceItEmpties` 的最后一步展示它。两者都是服务器管理员自己的操作，没有权限的提升，命令照旧提示 `nerve users activate`；P6 之后，停用本身不再留下这两种状态，`reactivate-member` 是它们唯一的来源。另一种写法（`reactivate-member` 在 `activate` 之前拒绝停用的账户，或在工作区没有管理员时拒绝）要改 P5a 的命令、3.11 和交错 16，负责人没有选。设计 3.11 和 README 的"恢复被移出的成员"写明：要让工作区有人管理，先恢复以前的管理员。
   - **(c) 规则 2 的项目一侧经降级绕过**（不处理）：同时是工作区管理员的项目成员可以把项目唯一的管理员改为成员（3.7"项目由工作区管理员管理"，P5b 第 3 节第 11 条），之后那人的停用不再被项目拒绝，项目有成员、没有管理员。设计认可的状态，不是新的绕过。工作区一侧没有经降级的对应的路（没有人能改自己的角色，改别人的管理员角色的人自己仍是管理员）；第一稿据此写"工作区一侧没有这样的路"，不对：(d) 是一条。
   - **(d) 一人的工作区经邀请回到无管理员**（预检 M1，第一稿漏报；负责人裁定修法 (a)）：他是工作区唯一的有效成员（规则 2 因而允许）时，第一稿的停用只删发给他邮箱的邀请，他（或以前在那里的管理员）发出的待接受邀请留着；被邀请的人接受之后，工作区有有效成员、没有有效管理员（Plane 相同）。P6 加第五条语句 `DeleteInvitationsOfWorkspacesLeftEmpty`（修订一轮先由邀请的锁按 id 锁住它要写的行，再照裁定 F-1 把它的谓词并入那条锁的语句，删除由 `DeleteInvitations` 按锁住的 id 做；2.3、第 20 条）：在工作区的 N 之下、他的成员关系结束之前，删除锁住的工作区中他之外没有有效成员的那些的待接受邀请。`TestADeactivationDeletesTheInvitationsOfAWorkspaceItEmpties`（组合，两条路）、存储的锁的回答（`TestLockInvitationsToDeleteLocksWhatItReturns`，修订一轮之前是 `TestDeleteInvitationsOfWorkspacesLeftEmpty`）、锁的阶梯的 kappa（顺序和强度）、W9 的 epsilon 钉住它；`m1-dropped` 在单元、组合、端到端失败。这条路关闭之后，"有有效成员而没有有效管理员"只由 (b) 的服务器管理员操作到达。设计 3.7、3.9 随 `e9289a4d` 改；差异清单的"停用账户"一行写明与 Plane 的差异。
   - **"没有有效的成员关系"**：停用的账户不能认证（接受、加入、建工作区都 401）；添加项目成员、指定负责人要有效的工作区成员关系（422）；`nerve workspaces create --admin-email` 拒绝停用的账户（P1）。只有 (b) 的 `reactivate-member` 例外。
   - **"没有发给他邮箱的邀请"**：改邮箱之前发给旧地址的邀请，停用不删（它们已不是他的地址），他也接受不了（403 `workspace.invitation_email_mismatch`），注册了旧地址的另一个账户可以接受：那不是他。
9. **命令的一行不列出工作区**（说明）：`deactivated <邮箱>: revoked <n> sessions and ended its memberships; …` 不列出结束了哪些工作区；服务器管理员要恢复时需要知道 slug（`reactivate-member` 对他从来不是成员的工作区答 `workspace.never_a_member`）。列出它们要让端口回答工作区（跨模块的端口多一个回答），P6 不做，记在第 7 节。
10. **`deactivateMe` 不进矩阵**（说明）：矩阵的格子是"谁对哪个工作区、项目的资源能做什么"；`deactivateMe` 只作用于调用者本人，规则表里没有它的操作名，没有工作区、项目的作用域，"别人能不能停用他"不成立（路径里没有账户）。它的拒绝、成功由两条路的组合测试（Task 5、6）在每张表上核对。
11. **写者是他本人**（说明）：邀请和成员关系的 `updated_by_id` 是被停用的账户，两条路相同。服务器管理员不是账户（M2 的命令没有操作者行，日志记 `"by":"cli"`）；`deactivateMe` 的调用者就是他。
12. **命令的一条路与改邮箱**（说明）：命令按邮箱锁账户行（M2），改邮箱先提交时，旧地址锁不到行，`identity.account_not_found`，什么都不改（`TestADeactivationReadsTheAddressUnderItsLock` 的命令一行）；服务器管理员用新地址再执行一次。
13. **再停用一次**（说明）：已停用的账户的会话已撤销，`deactivateMe` 401；命令照样完成（M2：账户、新手引导再写一次，撤销 0 个会话），P6 的语句不再写任何一行，他的不写，别人的也不写（都已结束、已删除，他的工作区集合为空；`TestADeactivationEndsEveryMembership` 的"again"一步，修订一轮另核对别人的每一行，停放项 P10；W9 的最后一步只结束 `reactivate-member` 恢复的那一行）。
14. **清扫 16**（说明）：假实现的 `LockMemberWorkspaces` 从一个 map 收集、按 id 排序回答，重新排序不改变它；`Deactivator` 原样传下存储的顺序（`s16-workspaces-reversed` 在单元一层失败；组合一层 SQL 的 `ANY` 不看顺序，等价）。
15. **S3 在副本里失败**（说明）：原型和复现的副本不是 git 仓库，构建没有提交号，S3 读 `commit` 失败（P4b 的 F4），与 P6 无关；其余故事全部通过。
16. **测试盖的"由 dave 写"的戳**（清扫 35，说明）：`TestADeactivationEndsEveryMembership` 先用 SQL 把 bob 的十行的 `updated_by_id` 改成 dave，让"由 bob 写"的核对可以失败（清扫 14）；说明写的是"由测试盖上"，不说成 dave 的一次写（dave 写不了这些行）。世界里其余的行都由规则允许的人经接口写出（`deactivationWorld` 的每一步：carol 建 gamma、邀请、移出 dave；alice 改角色、移出 dave；bob 归档 Solo、加入 Docs），矩阵对这些写本来就有格子。
17. **前端的两份文案不变**（说明）：`auth.json` 的 `workspace_sole_admin`、`project_sole_admin` 是页面说给页面的调用者（停用自己、离开、移出的成员）的，原来的补救对他们成立；服务器的说明改为不说"谁去做"，是为了命令的调用者（第 2.5 节）。W9 的页面（P9）用它们。
18. **`byCommand` 以大写、带空格输入地址**（清扫 22，说明）：命令规范化地址再锁、再传给成员关系一步；传入原样输入的地址的变异（`s22-cli-raw-address`）原来只在单元一层失败，组合的世界改为以大写、两边带空格输入之后在组合一层失败（邀请留着）。
19. **故事看不到的谓词**、**只在单元一层**和**只在存储一层被发现的变异**（控制者 2026-10-04 接受，先暂定、待预检；预检找出一处不对，照负责人和控制者同日的裁定已改）：见附录 A 的三段，每个的理由逐条写出。第一稿据不变式 (c)"有有效成员的工作区总有有效的管理员"把 `s1-sole-role` 算作组合一层等价；预检 M1 证明 (c) 不成立，它已删去，`s1-sole-role` 现在由 `TestADeactivationDeletesTheInvitationsOfAWorkspaceItEmpties` 在组合一层发现（`reactivate-member` 之后两位成员、没有管理员的 gamma，(b) 的已知限制）。照现在的代码和测试成立的是：只在单元一层被发现的，没有一个是组合一层可以展示的安全性质或锁的性质；只在存储一层被发现的，每个或由附录 A 写出的一条不变式（(a)、(b)、(c')）在组合一层等价，或在组合一层到不了（第 5 条），或是组合出的 app 里不会发生的语句失败（清扫 19）（清扫之后加强的测试见附录 A 开头，这一轮的修改见其后一段）。
20. **两个停用在交叉的邀请上互相等**（终审的 I1；负责人 2026-10-04 裁定"A"，控制者同日的裁定 F-1 补全，修订一轮的两步落实；设计 3.6 的表和 3.9 随它改）：第一稿两条删除各在自己的语句里取邀请的行。`DeleteInvitationsTo` 写发给他地址的邀请，它们多在他不锁的工作区里；第五条语句写他留下的空工作区的待接受邀请。bob 的 acme 邀请了 carol 的地址，carol 的 gamma 邀请了 bob 的地址，两人各是自己工作区唯一的有效成员（别的成员已结束），同时停用：各自先删发给自己地址的、对方工作区的那一份，再去删自己工作区的待接受邀请，正是对方已持有的那一份，40P01，一个停用回滚。两个停用锁的工作区不相交，约定六的串行不覆盖它，规则 2 也不拒绝。裁定"A"：两条删除之前，以一条语句按 id 升序锁住（`FOR NO KEY UPDATE`）两者要写的全部邀请（`LockInvitationsToDelete`，谓词是两条删除的并集，2.3）；`Deactivator` 在 `SoleAdmin` 之后、读时钟之前调用它，时钟因而在这把锁之后读（2.5，3.3）。每个停用都在写任何一份邀请之前按 id 取全部，两个停用不在邀请上成环。测试：存储三个（锁住的恰是应删的、按 id、等锁时被删除的跳过）和失败测试；单元的调用记录、失败注入、拒绝停在它之前、时钟在它之后；组合的 `TestTwoDeactivationsWithCrossedInvitationsBothGoThrough`（两种先后 × 两条路，2.8；去掉这把锁的 `f-I1-dropped` 在 `-count=5` 下二十次都有一个停用回滚）和锁的阶梯（kappa 的、delta 的邀请按 id 取）。

    **余下的一种环，由裁定 F-1 关掉**（修订一轮的探针在存储一层做出过；控制者 2026-10-04 的裁定 F-1，补全裁定"A"）：邀请的 id 在建它的事务之前取（v7）。第一步之后，一份发给他地址的邀请若在他的锁语句之后、`DeleteInvitationsTo` 之前提交，`DeleteInvitationsTo` 照写它，而它没有按 id 次序被锁过；此时另一个停用若已持有它、又在等他锁住的一行，仍是 40P01，一个停用回滚。要到达它：那份邀请在另一人清空的工作区里（只有那里的唯一有效成员发得出），在第一人的锁语句和 `DeleteInvitationsTo` 之间、第二人锁工作区之前提交，它的 id 又比第一人锁住的那一行小（建立邀请的请求在事务之前等了更久）。裁定"A"的前提是停用写的每一份邀请都按 id 取过；按谓词删除也写锁之后才提交的那一份，前提不成立。裁定 F-1：锁的语句回答它锁住的 id（按 id 升序，谓词不变，仍是两条删除的并集），两条删除并成一条 `DeleteInvitations(ids, by, now)`，`WHERE id = ANY (ids) AND deleted_at IS NULL`，写的列不变（2.3）；`Deactivator` 把锁的回答交给它（2.5）。停用写的邀请都是它按 id 锁住的，环从根上没有了；谓词只写在锁的语句里一次。代价：锁之后才提交的、发给他地址的邀请留下，与第 8 条 (a) 已接受的"停用提交之后才建立的"等价，窗口只从"`DeleteInvitationsTo` 之后"提前到"锁语句之后"；锁的顺序和强度不变。测试：存储的 `TestDeleteInvitations`（恰写给它的 id）、锁的回答（`TestLockInvitationsToDeleteLocksWhatItReturns`，每个谓词各由一行决定）、`TestAnInvitationCreatedAfterADeactivationsLockIsNotItsToDelete`（窗口里的 z 不由他写，没有 40P01）；单元的调用记录核对删除收到的正是锁的回答；组合的 `TestAnInvitationCreatedAfterADeactivationsLockIsLeftToTheOther`（2.8；放回按谓词删除的 `f1-dropped` 在 `-count=5` 下十次都有 carol 的停用回滚），交叉的测试照旧（`f-I1-dropped` 照旧二十次都失败）。

## 4. 验收标准（完成线，M3 设计 12 节 P6）

- [ ] W9 的接口版本（含命令）通过，此前的每个故事仍然通过（`make e2e` 共 67 个：此前的 66 个，加 W9）。
- [ ] 交错 7、8、13–16 和真实停用下的 19 两种顺序通过，另加两位管理员同时停用，`-count=5 -race`，没有 40P01。修订一轮：两个停用在交叉的邀请上（第 3 节第 20 条）两种先后、两条路都完成，没有 40P01。
- [ ] `identity` 的 `apitest.Main` 两个方向核对通过（`deactivateMe` 的两个码由它自己的 HTTP 测试返回）；`bootstrap` 的整程序测试在真实的组合上返回它们（Task 5）。
- [ ] 两条路的拒绝每张表不变、成功一个事务一个时刻（不早于账户的时刻）、项目跨工作区按 id、spike 15、改邮箱先提交（Task 5、6）；清空一个工作区的停用删除那里的待接受邀请（第五条语句的谓词，修订一轮并入邀请的锁的语句；Task 1、3、5、6、9）。
- [ ] 命令没有请求期限，中断之后不变；连接池只有一个连接时照样完成（Task 6）。
- [ ] `TestCommandsComposeNoServerAndNoJobs` 钉住两个命令组合各建什么、不建什么。
- [ ] 矩阵照旧（P6 不加格子）通过，耗时记下。
- [ ] `make lint`、`make test`、`make gen-check`、`make knip`、`make test-web`、`make e2e` 通过。

## 5. 不在 P6 范围内

- 状态、标签：P7。`PROBLEM_MESSAGES` 的搬迁：P8。页面（W9 的页面版本：general 页的停用弹窗）：P9。M4 的工作项写入是否取工作区的 S：M4（负责人 2026-10-02）。

**留给后面的 Phase**（各 Phase 的 spec 接过去；P6 的 review 第 6 节照录）：

| Phase | 条目 |
|---|---|
| P7 | 状态、标签的写不改成员关系，不是约定六的增长或收缩，与停用没有新的交错；它们是项目级的写（工作区 S → 项目 N），与停用（工作区 N → 项目 N）在工作区行上串行，不成环。连带不改状态和标签：停用不碰它们 |
| P8 | `workspace.sole_admin`、`project.sole_admin` 的前端文案（`auth.json`）说给页面的调用者，与服务器的说明（P6 改为不说"谁去做"）不同是有意的（第 3 节第 17 条）；搬迁时照旧 |
| P9 | W9 的页面版本：general 页停用，他是某个工作区唯一的管理员、那里还有别的成员时，弹窗显示 `workspace.sole_admin` 的页面文案，账户不变；项目一侧照 `project.sole_admin`；成功之后回到登录页（设计第 2 节 W9）。接口的两个 409 和它们的说明已由 P6 钉住 |
| M4 | 工作项若加入新的成员关系增长（例如指派时自动加入项目），照约定六与停用串行（项目一侧的增长取工作区的 S，3.6），并加一条与停用的交错；"只投递"的 River 客户端照原计划留在 M4（停用不投递任务） |
| 收尾 | 3.20 的 P6 两行（差异清单"停用账户"、README 8.7）和 P2 一行的"停用一段在 P6 随实现核对"（总体设计 4.2）由 Task 9 写好；收尾逐行核对 |

## 6. 风险

| 风险 | 应对 |
|---|---|
| 停用取他全部工作区的 N，方案 E 之下同一工作区连续重叠的项目级的写会让它一直等（17.4 的代价落在停用上） | 接口照服务的请求期限，超时回滚、什么都不改；命令没有期限，等到操作者中断，中断回滚（`TestAnInterruptedDeactivationChangesNothing`），README 写明可以重试 |
| 一个账户在很多工作区、项目：一个事务锁很多行 | 每条语句是整个集合上的一条（约定五），锁按 id 一次取；与别的写只在工作区行、项目行上串行，没有环（附录 A 的交错和两位管理员）。终审找出第一稿的一个环：两个停用在交叉的邀请上互相等（I1），邀请的行现在也由一条语句按 id 一次取，删除只写取到的那些（第 3 节第 20 条：裁定"A"加锁，裁定 F-1 让删除只写锁住的 id，关掉了锁之后才提交的邀请带来的余下的一种环） |
| 端口签名、命令行组合与设计的文字不同（第 3 节第 1、2 条） | 负责人 2026-10-04 接受，设计已改（`e9289a4d`）；两条都有反例在组合一层失败 |
| 故事看不到的谓词 | 存储测试看得到；组合一层另有世界；理由逐条在附录 A |
| plan 的最大 Task 接近上限 | 最大的 Task 4 是 1,336 行 |
| 持续集成（ubuntu）上的运行 | 原型和复现都在 macOS 上；合并之后看持续集成，P1–P5b 都是这样合并的 |

## 7. 已知的限制、交接和关闭条件

- **一个停用两个时刻**（第 3 节第 1 条，负责人接受）：账户、新手引导、会话的时刻在事务之前（M2），邀请和成员关系的在最后一把工作区锁之后；都在一个事务里。两者都读系统时钟：墙钟回拨时后者可以早于前者，只关审计的先后，没有规则读这两个时刻。
- **命令的一行不列出工作区**（第 3 节第 9 条，控制者接受）：服务器管理员要恢复时，需要从别处知道他在哪些工作区：各工作区成员列表里的已结束成员，或数据库里由他本人结束的成员关系（`workspace_members` 中 `member_id`、`updated_by_id` 都是他、`is_active` 为假的行）。
- **回来的第三条路**（第 3 节第 8 条 (a)，负责人裁定的已知限制）：工作区管理员向他的地址发一份新邀请，`activate` 之后他接受，以前的成员行恢复，与 Plane 相同；服务器管理员一侧回来只经 `activate` 加 `reactivate-member`。
- **`reactivate-member` 不看工作区有没有有效的管理员**（第 3 节第 8 条 (b)，负责人裁定的已知限制）：在 `activate` 之前恢复一个停用的管理员，他计入管理员的人数而不能登录；把一个以前的非管理员成员恢复到停用清空的工作区，工作区有成员而没有管理员。两者都是服务器管理员自己的操作；设计 3.11、README 写明先恢复以前的管理员。
- **规则 2 的项目一侧经降级绕过**（第 3 节第 8 条 (c)）：设计认可的状态，不处理。
- **停用不删除显示设置**：`workspace_user_properties`、`project_user_properties` 留着，恢复之后照旧（P5a、P5b 第 7 节同一条，与 Plane 相同）。
- **方案 E 的代价**（17.4）：见第 6 节。

| 交接 | P6 处理的条目 | 留下的条目 |
|---|---|---|
| P5a spec 第 5 节、review 第 6 节的 P6 一行 | 第 3 节第 3 条 | 无 |
| P5b spec 第 5 节、review 第 6 节的 P6 一行 | 第 3 节第 3 条 | 无 |
| P4a、P4b、P3、P1 review 第 6 节中 P6 的条目 | 第 3 节第 3 条 | 无 |
| M2 收尾交接第 6 节 | 关闭（处理结果在交接里） | 无 |
| M4 的 M2 收尾交接第 1 节 | 不提前（第 3 节第 3 条） | 照原计划在 M4 |

**M3 设计 13.1 的关闭条件**：13.1 的 M2-closeout §6 一行（"三件事在停用的事务里；唯一管理员时拒绝；接口和命令两条路都有测试；`deactivateMe` 的码声明、两个方向核对；差异清单'停用账户'一行；加锁顺序和成员关系集合的增长与收缩写在 3.6，每条增长路径与停用的交错测试（7、8、13–16）和忽略与停用的交错测试（19）通过"）全部落在 P6 的 Task 1–9；M4 的 M2-closeout §1 一行由 P6 的 review 写明不提前。没有放不下的条件。

## 附录 A：原型验证记录（2026-10-04）

原型在 `$M3TMP/p6proto`（`6dcc0794` 的副本，Go 1.27.1（`toolchain`）、Node 24、pnpm 11.10.0、Docker；`bin/golangci-lint` 2.13.2）。做法照 P5b：每个 Task 做完时存一份源文件的快照（`$M3TMP/p6snap/T1`…`T9`），plan 的代码块由脚本从相邻两份快照的差异生成（`p6tools/mkblocks.py`），生成的文件不进块，按 SHA-256 核对（`gensha.py`）。Task 5–9 在原型上一起写成，快照 T5–T9 由 `build_snaps.py` 从 T4 和原型按文件归属拼出（T5 取当时存下的 `p6snap-T5-taken`），每一份都 `go vet` 过（`snapcheck.sh`），并在逐 Task 复现中运行过。`assemble.py` 组装 plan 并核对每个块放了一次、每个文件在文件表里、每个 Task 在约 1,500 行以内、没有 HTML 实体。

**清扫之后的加强**（全部变异跑完之后，2026-10-04）：变异表里只在存储或单元一层被发现、而组合一层本可以展示的，改测试，不改产品代码：

- `TestEachLockOfADeactivationIsItsStrength`：不带 `ORDER BY` 的锁工作区（`o-ws-unordered`）原来只在存储一层失败。`EXPLAIN` 在 `deactivationWorld` 上显示计划经 slug 的部分索引读工作区、再逐个探他的成员关系，acme、beta 的 slug 顺序与 id 顺序相同；只重写 acme 的行（第一版的做法）不改变它。现在 carol 另建 able（bob 加入，id 最后、slug 最前）和 delta（bob 加入后被她移出，`carolsWithBob`），acme 的行仍重写一遍（顺序扫描会先碰到 beta）：两个变异都在组合一层失败；他已结束的成员关系也算进去的 `s1-lmw-active` 同时在这里失败（delta 被锁）。
- `TestEndWorkspaceMemberships`：被结束的两行原来由 bob 自己写（`joinAt`），保留写者的结束（`s14-ewm-keeps-writer`）在存储一层看不出；现在两行之前由 alice 写（清扫 14）。
- `TestARefusedDeactivationChangesNothing`：只在他按 id 的第一个工作区检查规则 2 的实现（`s24-check-first-workspace`）原来只在单元一层失败；加第五个情形：alice 离开 beta 之后 bob 是 beta 唯一的管理员（2.7）。
- 交错 13–16 和两位管理员的结果：原来只写明被点名的那一行的结束者（项目的一行、第一位在 acme 的一行）；现在 `endersOf` 读出账户每一个已结束的成员关系和它的写者（清扫 40；2.9、2.10）。
- `changeRole` 从 `interleaving_growth_test.go`（404 → 398 行）移到交错 7 的文件；`cmd/nerve/users_test.go` 写入 oto 的成员关系的种子语句加 5 秒的期限（清扫 29）；`api/modules/identity.yaml` 中 `deactivateMe` 的描述一行超长，重新折行（折叠块的值不变，`make gen` 之后生成物没有差异）。

之后快照由 `sync_after_mut.py`、`build_snaps.py` 改进，块和生成物的表重新生成；受影响的变异重跑（`mutants_p6_rerun.py`：锁的九个在组合一层、`EndWorkspaceMemberships` 的十一个在存储一层、上面的两个在组合一层、契约的一个在单元一层）。

**裁定和预检之后的修订**（2026-10-04，设计随 `e9289a4d` 改）：负责人对预检 M1 的裁定 (a) 加第五条语句 `DeleteInvitationsOfWorkspacesLeftEmpty`（第 2.3 节），带它的存储测试（`TestDeleteInvitationsOfWorkspacesLeftEmpty`，每个谓词各由一行决定）、失败测试（清扫 19）、调用记录和失败注入（清扫 2）、组合的反例 `TestADeactivationDeletesTheInvitationsOfAWorkspaceItEmpties`（由预检的演示测试改成，两条路）和 W9 的一步（B 独自一人的 epsilon，他发给 C 的邀请）；下面的不变式 (c) 删去，换成第五条语句建立的 (c')，`s1-sole-role` 在新的组合测试里失败。预检的 L1–L8：`identity` 的两个失败测试比较截到失败那一步的调用记录（L1）；账户的时刻不晚于成员关系的时刻，由 `TestADeactivationEndsEveryMembership` 核对（L2）；`activate` 的说明、`deactivateMe` 的描述、README 的补救、plan 的全局约束、锁的阶梯关于 Ops 和 Solo 的说明，五句改成代码做的（L3）；交错 7 (b) 在放开停用之前读每张表（L4）；清扫 29 的说法改成下面逐处归类的样子，两个存储测试等锁的 goroutine 和 `newS1Race` 的种子加期限（L5）；清扫 1 的表加上 `identity` 改过的 `LockUserForCredentials`（L6）；交错 19 的"由 bob"要求邀请最后由停用写（L7）；第 3 节第 8 条第二点写全 `reactivate-member` 能到达的（L8）。之后快照、块和生成物的表重新生成；杀死它的测试改过的变异（`amend_affected.py` 按测试的改动挑出，93 个）、`s1-sole-role` 和新的 22 个由 `mutants_p6_amend.py` 在它写的每一层重跑。下面的数字都是修订之后的。

**终审之后的修订一轮**（2026-10-04；负责人对终审 I1 的裁定"A"，和 9 个 Task 里停放的 28 个测试一侧的条目 P1–P28）：在分支上直接改，不经原型和 plan 的块（I1 的修法改产品代码，其余都在测试和文档）。I1：第六条语句 `LockInvitationsToDelete`、`Deactivator` 的新的一步、它们的存储和单元测试、交叉邀请的组合测试、锁的阶梯的两份邀请（2.3、2.5、2.8，第 3 节第 20 条）。停放项各在第 2 节写它的测试处注明（"停放项 Pn"）。变异：邀请的锁的 26 个和表的 141 个重跑见下面"变异表"之前一段；停放项各自的变异在改过的测试上失败，只有一处做不到：P27 要 `s1-dil-workspaces` 在 W9 失败，它在组合一层等价（"故事看不到的谓词"）。门禁在修订一轮的最后一个提交之后重跑（报告）。

**修订一轮的第二步**（2026-10-04；控制者的裁定 F-1 至 F-5）：F-1 让删除只写锁住的 id（第 3 节第 20 条、2.3、2.5、2.8）：`LockInvitationsToDelete` 回答它锁住的 id，计划的两条删除并成 `DeleteInvitations`，存储加 `TestDeleteInvitations` 和窗口的测试，锁的测试核对回答，单元核对删除收到的正是锁的回答，组合加 `TestAnInvitationCreatedAfterADeactivationsLockIsLeftToTheOther`。F-2、F-3、F-4 接受第一步的做法。F-5 是停放项复审的三处测试一侧的修改：两位管理员同时停用的测试里，每个管理员在 acme 的成员行改由另一位管理员盖戳，alice 在 Ops 的由 carol（Ops 的另一位管理员），Web 的两行仍由 dave（Web 的管理员），规则允许的写者（停放项 P19 的同一类）；交错 7 (c) 的说明不再说 Web 的结束"由他"，那里只有时刻看得出；`lastWrittenBy` 由三次 `stampWriter` 组成，各核对行数，结束和停用测试里另写的盖戳循环也改用它。变异（`$M3TMP/p6fix/mutants_r2.py`）：表的 141 行在每行写的每一层重跑，其中 33 行原来改的是计划的两条删除或 `Deactivator` 里它们的调用，重新映射到锁的语句（地址的一支、空工作区的一支的每个谓词，走池，失败被吞掉）、`DeleteInvitations`（写的列、未删除、走池、失败）或 `Deactivator` 的新的两步（吞掉锁或删除的失败、顺序）；`m1-after-members` 照 F-3 改成"邀请的锁在结束成员关系之后"，阶梯和交叉的测试都发现它。141 个都被发现（`archtest` 的 2 个照旧由 `archmut.py`），层变了的 6 行都是重新映射的：`s1-dit-deleted`（`DeleteInvitations` 的未删除）、`s1-dil-deleted`（锁的语句的未删除）只在存储一层，另一条的同一谓词仍在，锁住的行在停用持锁时不会被别人删除：组合一层等价；两条一起去掉的 `f-deleted-both` 在存储、组合、W9 都失败。`s14-dil-keeps-writer` 多了 W9（现在与 `s14-dit-keeps-writer` 同是删除的写者）；`s8-dil-pool` 多了存储；`m1-dropped`（锁的语句里空工作区的一支去掉）不再是 `Deactivator` 的调用，单元一层看不到，组合、W9 照旧；`m1-after-members` 多了组合。另 15 个新的或第一步之后没有表行的（锁的语句的交集、访客不算、无序、倒序、`FOR SHARE`、`FOR UPDATE`、不加锁，`DeleteInvitations` 写全部邀请，`f1-dropped`、`f-I1-dropped`，时钟在锁之前，锁不传工作区、不传邮箱，删除收到工作区的 id、不删）都被发现：`f1-dropped`（2eb20f77 的存储、端口和 `Deactivator`，按谓词删除）在窗口的组合测试 `-count=5` 十次都有 carol 的停用回滚，`f-I1-dropped`（再去掉锁）在交叉的测试二十次都有一个停用回滚。只在存储一层的锁的变异（裁定 F-4 问的四个多锁的）：发给别的地址的（`s1-dit-email`）和已忽略的（`s1-dil-responded`）现在也多删，组合测试读到（锁下的邮箱的测试、清空工作区的测试），`s1-dit-email` 另在 W9；`s1-dil-deleted` 不多删（删除的未删除仍在）；访客不算成员的 `f-lid-guest-ignored` 多删的是只剩访客的工作区的待接受邀请，组合的世界里没有这样的工作区，仍只在存储一层。F-5 的保留写者的变异在盖戳的每个测试失败（两位管理员的测试 2 个，等锁期间的变化和锁下的邮箱 8 个，结束 3 个，停用 3 个，清空工作区 1 个）；交错 7 里项目一侧保留写者的变异照旧通过（说明不再那样说），保留时刻的失败。结果在 `p6fix/logs/mutants_r2-all.json`、`mutants_r2b-all.json`、`mutants_f3-all.json`、`mutants_f5-all.json`、`archmut.json`，逐个写在修订一轮的报告的"Round 2"里。下面清扫表里说到 `DeleteInvitationsTo`、第五条语句的行，说的是计划的语句和当时的变异；第二步之后那些变异改的是哪一条语句、在哪一层被发现，以这一段和报告为准。门禁在第二步的最后一个提交之后重跑（报告）。

**逐 Task 复现**（`$M3TMP/p6tools/replay.py`、`replay_all.sh`，日志在 `replay-logs`）：从 `6dcc0794` 的一份新副本开始，照 plan 的顺序应用 9 个 Task 的块并运行每个 Task 写明的命令（每个 Task 的定向测试、`make lint-go`、`make test`，有的 Task 另有生成、`-race`、前端检查和端到端）。每个 Task 之后 `make lint-go` 两段 `0 issues.`、`make test` 42 个 `ok`；Task 1 的 `make gen-go`、Task 4 的 `make gen` 之后生成物与快照没有差异，SHA-256 和行数与 plan 的表相同；Task 4、9 的 `make lint-web`（关键词守卫 5 个命中都有例外）、`make knip`、`make test-web` 通过；Task 4、6、7、8 的 `-race` 运行（Task 6 `-count=3`，其余 `-count=5`）通过；Task 9 的 `make e2e` 67 个故事中 66 个通过，失败的只是 S3 的 F4（`replay.py` 只接受这一个失败）。最后的树与原型逐个文件相同（`treediff.mjs`：3,066 个文件，0 处差异）；`planapply.mjs` 从基线核对 plan 的 146 个块（129 处替换、16 个新文件、1 个整文件）都放得上；复现的副本最后 `make gen-check` 通过。共 762 秒（修订之后，从 `6dcc0794` 的一份新副本重做）。

**最终的原型**（`gates.sh`）：`make gen` 之后生成物没有差异；`make lint-go` 两段 `0 issues.`；`make test` 42 个 `ok`；`make lint-web`（关键词守卫 5 个命中都有例外，54 个任务）；`make knip`；`make test-web`（16 个任务）；`make e2e` 67 个故事中 66 个通过，S3 因原型不是 git 仓库、构建没有提交号而失败（P4b 的 F4；它读 `commit`，与 P6 无关；第 3 节第 15 条）；W9 单独运行通过。P6 没有迁移。（修订之后重跑。第一次的 `make test` 有两个包的测试容器因 Docker 的接口超时没有起来，与代码无关，重跑通过。）

**矩阵**：`TestPermissionMatrix -v` 532 格（P6 不加格子，与 P5b 结束时相同），1.50 秒，全部通过。

**交错和竞争**（`race5.sh`）：`go test -count=5 -race -v -run '…' ./internal/bootstrap/` 跑 18 个测试：交错 7（2 个）、8（2 个）、13–15（2 个）、16、19，两位管理员同时停用，锁的顺序和强度，命令的中断，事务的连接，等锁期间的变化，锁下的邮箱，P3 的交错 12（同用 `setBobsEmail`），两条路的拒绝、提交被拒和成功：`ok`，93.6 秒，18 个测试各 5 次全部通过（270 个子测试）；输出里没有 40P01，没有数据竞争（修订之后重跑）。

**四十类清扫**（brief 的缺陷类别；每个变异一个 `go test -overlay` 或 `go build -overlay`，树不动，Go 以外的文件（打包的契约、W9）原地改、跑完复原；`mutlevels.py` 在变异写的每一层各跑一次：单元（`identity`、`workspace`、`project` 的单元包和 `archtest`）、存储（三个模块的 `adapter/postgres`）、组合（`bootstrap`、`cmd/nerve`）、端到端（单独运行的故事 W9）；"层"是它被发现的每一层。`archtest` 从磁盘读源文件，`-overlay` 到不了它，它的两个变异由 `archmut.py` 写进服务端模块的一份副本再跑）：

| 清扫 | 大小 | 结果 | 层 |
|---|---|---|---|
| 1 每个 SQL 谓词 | 存储一半：P6 的 5 条新语句的 32 个谓词（`LockMemberWorkspaces` 5：工作区未删除、成员关系相关、成员、有效、未删除；`SoleAdmin` 14：问到的工作区、他、管理员、有效、未删除，另一位管理员的相关、"别人"、管理员、有效、未删除，别的成员的相关、"别人"、有效、未删除；`DeleteInvitationsTo` 2；`DeleteInvitationsOfWorkspacesLeftEmpty` 7：问到的工作区、未回应、未删除，别的成员的相关、"之外"、有效、未删除；`EndWorkspaceMemberships` 4），每个去掉，参数照旧绑定。`identity` 改过的一条 `LockUserForCredentials`（选择列表加 `email`，锁下的地址从这一行来）：它的 `id = $1` 和 `FOR NO KEY UPDATE` 由已有的 `TestLockForCredentialsReadsTheRow`（bob 存在 alice 之前）、`TestLockAccountIsTheAccountRowLock` 和锁的阶梯（他的账户的 `FOR NO KEY UPDATE`）钉住，不另加变异（预检 L6）。故事一半：这 32 个和清扫 20 的五个集合形式，每个只运行 W9 | 存储一半 32/32，每个存储测试都有只由那个谓词决定的行（五条语句都是集合上的 `UPDATE`、`EXISTS` 或带 `ORDER BY` 的读，没有"取第一行"，物理行序不决定结果；锁的顺序另由 `…LocksInIDOrder` 在逆序的堆和索引上核对）。组合一层 21/32，另 11 个在组合一层等价，理由在下面"只在存储一层被发现的变异"。故事一半 14/37：W9 发现 `SoleAdmin` 的他、另一位管理员的相关、"别人"、管理员，别的成员的相关、"别人"（修订加的 epsilon：B 独自一人，停用照样通过），`EndWorkspaceMemberships` 的成员，第五条语句别的成员的相关、"之外"，`s20-sole-admin-any`、`s20-sole-other-any`、`s20-dil-other-any`，修订一轮加的 `DeleteInvitationsTo` 的邮箱、未删除（`s1-dit-email`、`s1-dit-deleted`：D 的两份待接受邀请、B 已接受的那一份，停放项 P27）；看不到的 23 个理由逐条在下面"故事看不到的谓词"，每个都由存储测试发现，12 个另由组合测试发现。修订一轮的第六条语句的谓词见"修订一轮：邀请的锁"一行 | 存储；组合；端到端 |
| 2 端口调用的错误 | `Deactivator` 的六个端口调用（锁工作区、问唯一管理员、删邀请、删他留下的空工作区的待接受邀请、结束工作区成员关系、结束项目成员关系）：失败被吞掉 6、判定的失败答 404、项目一步失败之后重试；`identity` 不管成员关系一步的回答照样提交、会话一步失败之后照样调用成员关系一步再答失败、成员关系一步失败之后重试（后两个是预检 L1）；11 个 | 11/11；每个单元测试（`TestDeactivateMembershipsFailsAtEachStep`、`TestDeactivateWhenAWriteFails`、`TestDeactivateByEmailWhenAWriteFails`）比较到失败那一步为止的整个调用记录 | 单元；组合、端到端（吞掉的是项目一侧的拒绝的 2 个）；组合（`pf-id-retries-memberships`：项目一侧拒绝之后，重试的一步找不到他有效的工作区成员关系，什么都不结束就答成功，`TestARefusedDeactivationChangesNothing` 看到提交） |
| 3 写不动的行 | 3 个写（`DeleteInvitationsTo`、`DeleteInvitationsOfWorkspacesLeftEmpty`、`EndWorkspaceMemberships`），存储测试有另一个工作区、另一个成员、另一个地址、已结束、已删除、已忽略、不问的工作区的行，核对每行每列；9 个变异（三个写各改写 `created_at`、保留原来的时刻，两条删邀请的语句标成已接受，结束时也删除） | 9/9 | 存储；组合；端到端（2 个） |
| 4 组合根的接线 | 服务和命令各自的 `Memberships` 换成什么都不结束的、`Deactivator` 的项目连带换成什么都不结束的、时钟停在 2001 年；`project.NewCascade` 建出什么都不结束的连带；`nerve users` 另建 `workspace.NewAdmin`、`nerve workspaces` 另建 `project.NewCascade`；9 个 | 9/9 | 组合（7 个；3 个另在 W9）；单元（`archtest` 的 2 个） |
| 5 安全性质在真实环境上 | 规则 2 的工作区一半、项目一半，它们的每个限定（清扫 24、20）；`deactivateMe` 不进矩阵（第 3 节第 10 条） | 每个都在组合出的 app 上被发现：`TestARefusedDeactivationChangesNothing`（两条路 × 五个情形）、`TestADeactivationEndsEveryMembership`、`TestTwoAdminsDeactivatedAtOnceLeaveAnAdmin`；W9 两个拒绝 | 组合；端到端 |
| 6 每句文档 | P6 加或改的注释 145 块（768 行，36 个文件）；契约 `deactivateMe` 的描述；两个码的文字；命令的说明和输出；README、差异清单、总体设计的段落；4 个变异（两个码的说明改回"去做什么"、命令的一行少了第二步、成员关系另读一次时钟） | 4/4。清扫中改了：`project.sole_admin`、`workspace.sole_admin` 的说明（对命令的调用者也成立，2.5）；锁的测试的说明（able、delta）；交错 13–16 的说明（每个结束都写明由谁） | 单元；组合；端到端 |
| 7 反例里没有随机 | P6 的全部测试 | 决定变异是否被发现的输入都固定：世界的 id 顺序（acme 在 beta 之前、Lab 在 Docs 之前、able 在最后）在 `newDeactivationWorld` 和锁的测试里核对；物理顺序由重写决定；`time.Now` 只用来框住时刻；不用 `math/rand` | — |
| 8 决定所依赖的读在调用者的事务里 | 五个存储方法各改成走池；5 个（第六条语句的 `f-lid-pool` 见"修订一轮"一行） | 5/5 | 组合（`TestTheDeactivationRunsOnItsTransactionsConnection`：命令的池只有一个连接，走池的语句等到期限）；存储（2 个：修订一轮之后 `s8-dit-pool` 也在存储一层失败，`TestLockInvitationsToDeleteLeavesOutAnInvitationDeletedWhileItWaited` 里走池的删除等它自己的事务锁住的行）；端到端（1 个） |
| 9 锁顺序在真实环境上 | 账户 → 工作区（按 id）→ 邀请 → 工作区成员关系 → 项目（跨工作区按 id）→ 项目成员关系；项目一步在前、先结束成员关系再删邀请、写在检查之前、`identity` 先写成员关系、第五条语句在结束成员关系之后（第一稿：邀请的行在成员行之后才锁，`m1-after-members`）；另有 P6 自己的顺序变异（第 P6 行）；修订一轮的邀请的锁挪到两条删除之后、成员关系之后见"修订一轮"一行 | 4/4（变异 5 个：`s9-end-before-invitations`、`s9-check-after-end`、`s9-memberships-first`、`m1-after-members`，`s11-refusal-after-invitations` 在第 11 行） | 单元（调用记录）；组合（`TestEachLockOfADeactivationIsItsStrength` 的阶梯）。修订一轮之后 `s9-end-before-invitations`、`m1-after-members` 只在单元一层：邀请的行现在由第六条语句在任何一条删除、成员行之前按 id 取，删除和结束的先后不再改变加锁的顺序，写的行和时刻相同（组合一层等价，理由在"只在单元一层被发现的变异"） |
| 10 承重的种子行有前提 | `deactivationWorld` 的 6 行（dave 是 acme 的管理员、他已结束；bob 是 beta 的管理员；dave 在 gamma 已结束；Solo 已归档；gamma 的邀请已忽略）；每行一个变异 | 6/6（`preconditions`，每个用这个世界的测试） | 组合 |
| 11 每条拒绝路径 | 判定在删邀请之后（拒绝之前已写）；1 个 | 1/1 | 单元（组合一层等价：拒绝回滚整个事务，`TestARefusedDeactivationChangesNothing` 核对每张表） |
| 12 与结束、删除的竞争（组合） | `TestADeactivationFindsWhatChangedMeanwhile` 三个情形 × 两条路（acme 删除、Docs 删除、bob 被移出 beta），探针证明在等锁；spike 15 的 404、500 两个变异（P6 行） | 2/2 | 存储；组合 |
| 13 锁强度在组合一层看得到 | 账户、工作区、项目的锁各改成 `FOR SHARE`、`FOR UPDATE`，工作区不加锁；7 个（P6 行） | 7/7 | 存储；组合 |
| 14 每个"由谁"可以失败 | 存储测试里被写的行之前由另一个账户写（加强之后 `EndWorkspaceMemberships` 也是，第五条语句的也是）；组合的世界里 bob 的十行先盖上 dave 的戳；交错 19 的忽略先：他已忽略的邀请之前由他写，"由 bob"另要它最后的时刻是删除的时刻（预检 L7）；5 个变异（三个写保留写者、删邀请写成邀请人、删邀请时已忽略的保留写者和时刻）。修订一轮（停放项 P11、P15、P19、P20）：组合测试里"由 X"的写者一半原来不会失败的地方，行在动作之前存在的，先盖上第三个写者的戳（`stampWriter`；等锁期间的变化和锁下的邮箱、交错 7 和 19、13–15、两位管理员）；行在动作里由他自己建出的（交错 7 (c) 和 13、14 加入的 Web），改看时刻；每个改过的说法，它的保留写者的变异在那个测试失败（报告逐个列出） | 5/5 | 存储；组合；端到端（3 个：修订一轮的重跑里 `s14-ewm-keeps-writer` 也在 W9 失败，W9 在 `fcaf8f17` 由 A 重写 acme、beta、Lab 的三行之后） |
| 15 标题的每个说法都有展示 | W9 的标题 8 个分句（修订加"他留下的空工作区的待接受邀请"，epsilon 的一步）；P6 的 Go 测试名和说明；1 个变异（只删待接受的邀请，`membershipEnd` 的一步） | 每个分句由故事自己的一步展示；1/1 | 存储；组合；端到端 |
| 16 单元假对象的回答顺序 | 假的 `LockMemberWorkspaces` 从 map 收集、按 id 排序回答；`Deactivator` 原样传下；1 个（`Deactivator` 倒转工作区） | 1/1 | 单元（组合一层等价：SQL 的 `ANY` 不看顺序，锁的顺序由语句的 `ORDER BY` 决定，第 3 节第 14 条） |
| 17 判定之前的代价有界 | 两条路的输入：调用者本人或一个地址；判定之前锁一行账户、他的工作区（一条语句）、一个 `EXISTS` | 没有调用者给的无界输入；锁的行数是他的成员关系数 | — |
| 18 契约描述只说代码做的 | `deactivateMe` 的描述和码；3 个变异（少声明两个码各 1，多声明 `workspace.not_found`） | 3/3。多声明的那个第一次改的是打包的契约，而 `apitest.Main` 读模块自己的文件，活下来；改在 `api/modules/identity.yaml` 之后被发现（`s18-extra-code-module`） | 单元（`TestDeactivateMeProblems`、`apitest.Main`）；组合（2 个） |
| 19 每个新的存储方法有失败测试 | 5 个新方法在 `failures_test.go` 里；5 个变异（失败被读成回答） | 5/5 | 存储 |
| 20 相关谓词的集合形式、"别的"算进自己 | `SoleAdmin` 的另一位管理员、别的成员改成"问到的任何工作区"；项目的 `SoleAdmin` 跨工作区调用时同样；第五条语句的别的成员同样；"别的"算进自己的 3 个在清扫 1；5 个 | 5/5 | 存储；组合；端到端（3 个） |
| 21 拒绝和失败钉住第一个 `*shared.Error` | `identity` 把拒绝与 `identity.account_not_found` 一起包；`Deactivator` 把拒绝与 `workspace.not_found` 一起 `errors.Join`；2 个 | 2/2 | 单元；组合 |
| 22 端口的回答对得上所问 | 接口传空的地址；命令传原样输入的地址（组合的世界以大写、两边带空格输入，第 3 节第 18 条）；2 个 | 2/2 | 单元；组合 |
| 23 组合的夹具跨第二个工作区、第二个项目 | `deactivationWorld`：acme、beta、gamma（锁的测试另有 able、delta），Web、Ops、Solo、Lab、Docs，他的项目 id 跨工作区交错；`memberWorld` | 清扫 1、20 的跨范围变异在组合一层被发现（`s1-*-correlated`、`s20-*`） | 组合 |
| 24 每条规则的每一半在组合一层有反例 | 规则 2 的工作区一半（不问另一位管理员、独自一人也拒绝）、项目一半（已结束的管理员算数）、项目一步只走第一个工作区、检查只问第一个工作区；5 个 | 5/5（`s24-check-first-workspace` 原来只在单元一层，加强之后组合一层也失败） | 单元（2 个）；存储（3 个）；组合（每个）；端到端（3 个：独自一人也拒绝的，修订加 epsilon 之后 W9 也发现） |
| 25 判定和执行在同一把锁下 | 在锁工作区之前在另一个事务里检查规则 2（与清扫 31 同一个变异） | 1/1 | 存储；组合（`TestTwoAdminsDeactivatedAtOnceLeaveAnAdmin` 的检查之后的 gate） |
| 26 "不变"用一个变异不会碰巧得出的值 | 结束写角色 20、15、列的默认（bob 的角色在各处不同：acme 15、beta 20、Lab 20、Ops 15…）；3 个 | 3/3 | 存储；组合；端到端 |
| 27 故事里的"不动"、"留着"在动作之前读、之后核对 | W9 的 4 处：两个拒绝前后的六张表、成功前后别人的每一行、再停用前后别人的每一行 | 每处之前、之后各一次 | 端到端 |
| 28 只在端到端被发现的性质另有 Go 的测试 | 全部 141 个变异 | 只在端到端被发现的只有 `s39-w9-collation`（改的是 W9 自己的查询），它的 Go 一半 `s39-endings-collation` 在组合一层失败 | — |
| 29 每个能等锁的等待都有期限 | P6 新改的测试里 87 处等待（`soon(t)` 16、`receiveWithin` 8、`WaitForLockWaitOn` 18、`result` 17、`held` 14、`context.WithTimeout` 14）；新改的行里仍有 `context.Background()` 的，`sweep29.py` 逐处归类：由它做出期限或取消的 15 处；放开持有者的 `Rollback` 5 处（回滚不等锁）；单元测试 11 处（假实现，没有数据库）；存储、命令测试里没有别的事务时的调用 6 处（`SoleAdmin`、`DeleteInvitationsTo`、`DeleteInvitationsOfWorkspacesLeftEmpty`、`EndWorkspaceMemberships` 的直接调用，命令测试的两个种子命令，照各包原有的写法），没有可等的锁。第一稿说新改的行里没有在 `context.Background()` 上执行的语句，不对（预检 L5）：两个存储测试的 goroutine 等锁、`newS1Race` 的四条种子语句，现在各有期限；池照 `openPool`；`cmd/nerve` 的种子语句加强之后有期限 | 每个能等锁的等待都有期限；说法收窄到"能等锁的"（预检 L5）：留在 `context.Background()` 上的都归了类，没有一个等得到锁。修订一轮（停放项 P3、P4、P9、P16）：bootstrap 的 `soon` 移到 `pgtest.Soon`，各包的测试共用这一个；`withLockTimeout`、项目存储测试的 `Begin` 和 `waits`、`tableRows`、交错 19 的 `answered`、`growthRace` 的两个读改用它，池的等待也有期限 | — |
| 30 每个输出、问题的细节和补救对每个收到它的调用者都成立 | 两个码的说明（接口的调用者、命令的调用者、离开和移出的调用者）；命令的一行；README | 两个码改为不说"谁去做"（2.5）；命令的一行写出回来的两步（`s6-cli-line`） | 单元；组合；端到端 |
| 31 检查和执行之间的 gate 在检查的事务之外 | `soleAdminCheckedHolding` 停在规则 2 的检查之后、写之前；检查在一个事务、写在另一个的变异 | 1/1，在这个 gate 由探针失败（第二位不等 acme） | 存储；组合 |
| 32 每个探针有反例 | 停用的等待挪到成员关系的行上（`s32-wait-moved`）；同一个变异，交错 13–15 和两位管理员的探针换成不看表的（`s32-wait-moved-blind`） | 2/2：前者在 6 个测试由探针失败；后者在那 3 个测试通过（换成不看表的探针就看不出），只在别的、探针仍看表的 3 个测试失败 | 组合 |
| 33 goroutine 上不调用 `t.Fatal` | P6 新改的 bootstrap 测试里 74 个交给 `run(…)` 或 `go` 的闭包（`sweep33.py`，P6 加、改的 34 个测试文件） | 没有一处在 goroutine 上调用测试；命名的闭包（`create`、`deactivate`、`route.send`）在测试的 goroutine 上解析 | — |
| 34 "没有替身"、"完成"的说法全包 grep | `gatedDeactivation`；手写的 `Deactivate` 构造；`"UPDATE "+表名` 和手写的停用语句 | `gatedDeactivation` 0 处；`identityapp.NewDeactivate` 在测试里只有 `deactivating` 一处；`UPDATE users SET is_active = false` 留两处（P1 的 `workspace_test.go`：alice 没有工作区，真实的停用写的就是这一行；P5a 的 `reactivation_races_test.go`：竞争的另一方，他的成员关系已结束），都是真实停用会到达的状态 | — |
| 35 "由 X 写"的种子是规则允许的 | `deactivationWorld` 每一步经接口由允许的人写；dave 的戳说成"由测试盖上"（第 3 节第 16 条）；`cmd/nerve` 的 oto 由 SQL 写入、不说成谁的写 | 没有说成某人的写而规则不允许的种子 | — |
| 36 别的操作或两步到达同样的结果 | 唯一管理员的两条规则；"停用之后没有有效的成员关系"；"没有发给他地址的邀请"；"没有人能经邀请进入没有管理员的工作区"；回来的路 | 四条路（第 3 节第 8 条）：(a)、(b) 负责人裁定为已知的限制，(c) 不处理，(d) 是预检 M1 找出的第一稿漏报的一条，由第五条语句关闭；去掉它的 `m1-dropped` 1/1 | 单元；组合；端到端 |
| 37 每个写对归档项目、删除的工作区、停用的调用者 | 归档的项目照样结束（Solo；`s37-archived-left`）；删除的工作区跳过（spike 15）；停用的调用者：`deactivateMe` 401，命令照样完成、P6 的五条语句不再写（第 3 节第 13 条） | 1/1；其余由组合测试钉住 | 存储；组合 |
| 38 故事的每个拒绝之前先读 | W9 的两个拒绝（接口、命令各一次） | 每次之前读六张表、之后核对 | 端到端 |
| 39 测试里的排序与排序规则无关 | `endings`、`endersOf`、`preconditions` 按 `COLLATE "C"`；W9 同样；2 个 | 2/2 | 组合；端到端 |
| 40 每个交错的结果写明每个结束由谁 | 交错 7、13–16、19 和两位管理员 | 加强之后每个结果读出账户每一个已结束的成员关系和它的写者（`endersOf`、`s1Race.standing`、交错 19 的"由 bob"）；交错 8 的创建先：新工作区只有她一个成员，她的成员关系只能由她结束 | 组合 |
| 修订一轮：邀请的锁（终审的 I1；第二步的重新映射和新的 15 个见"修订一轮的第二步"一段） | 第六条语句 `LockInvitationsToDelete` 的 9 个谓词各去掉（邀请未删除、邮箱、问到的工作区、未回应、两条删除的并集换成交集，别的成员的相关、"之外"、有效、未删除），别的成员改成"问到的任何工作区"（清扫 20）、访客不算成员；无序、倒序、`FOR SHARE`、`FOR UPDATE`、不加锁（清扫 9、13）；走池（清扫 8）；失败被读成"已锁住"（清扫 19）；`Deactivator` 的这一步去掉（I1 本身）、挪到 `DeleteInvitationsTo` 之后、第五条语句之后、成员关系结束之后，时钟在它之前读，失败被吞掉（清扫 2），不传工作区、不传邮箱；26 个 | 26/26；`f-I1-dropped` 在交叉邀请的测试 `-count=5` 的二十次里每次都有一个停用回滚（40P01，或第二个答 500） | 存储（语句的 18 个）；组合（18 个：语句的 11 个，`Deactivator` 的 7 个）；单元（`Deactivator` 的 8 个）。只在存储一层的 7 个、只在单元一层的 1 个，理由在下面两段 |
| P6 自己的 | 锁的顺序和强度（工作区 5、项目 4、账户 2，时钟在锁之前、项目一步在前）、spike 15（404、500，上锁时已删除的工作区、项目）、锁下的邮箱、命令的期限和中断；两个时刻的先后（`identity` 的时钟在两条路上各快一秒，账户的时刻晚于成员关系的，预检 L2）；22 个 | 22/22；只在存储一层被发现的两个是上锁时已删除的工作区（组合一层等价）和项目（组合一层到不了，第 3 节第 5 条） | 单元；存储；组合 |

141 个变异，141 个被发现，没有等价的；修订一轮之后 141 个在它写的每一层重跑、仍都被发现，另加邀请的锁的 26 个，167 个都被发现（下面"终审之后的修订一轮"一段）；第二步（裁定 F-1）之后 141 个再重跑（33 个重新映射）、另 15 个，156 个都被发现（"修订一轮的第二步"一段）。`s32-wait-moved-blind` 是清扫 32 要的反例：在探针换成不看表的 3 个测试里它通过（那 3 个测试只由探针看出等待挪了地方），只由别处探针仍看表的测试发现。第一轮里"活下来"的 3 个都不是真的：`archtest` 的 2 个是工具到不了（`-overlay` 改不了它从磁盘读的源文件），`s18-extra-code` 改的是打包的契约；重做之后都被发现。只在单元一层、只在存储一层被发现的，和故事看不到的，理由逐条在下面三段。

**只在单元一层被发现的变异**（每个都有理由，没有一个是组合一层可以展示的安全性质或锁的性质）：

- 清扫 2 的 8 个（锁、检查、删邀请、删空工作区的待接受邀请、结束的失败被吞掉，检查的失败答 404，项目一步重试；`identity` 在会话一步失败之后照样调用成员关系一步，`pf-id-runs-after-failure`，预检 L1）：组合出的 app 里语句不会失败；组合一层的失败注入在提交（`TestADeactivationRefusedAtItsCommitChangesNothing`，第 3 节第 7 条），存储一层的失败原样返回由清扫 19 钉住。
- `s11-refusal-after-invitations`：拒绝在删除邀请之后；拒绝回滚整个事务，组合一层每张表不变，等价。
- `s16-workspaces-reversed`：`Deactivator` 倒转工作区的顺序；后面的语句用 `ANY`，锁的顺序由语句自己的 `ORDER BY` 决定，等价。
- 修订一轮之后的 `s9-end-before-invitations`（先结束成员关系再删邀请）：它在组合一层原来只由锁的阶梯看出（邀请的行在成员行之后才锁）。现在邀请的行由锁的语句在删除和成员行之前按 id 取，删除只写已锁住的行；删除和结束的先后不再改变加锁的顺序，写的行和时刻相同：组合一层等价（裁定 F-3）。第一步里 `m1-after-members`（第五条语句在结束成员关系之后）同样只在单元一层；第二步之后它是"邀请的锁在结束成员关系之后"（F-3），破坏全局顺序，阶梯和交叉的测试发现它。
- `archtest` 的 2 个、`s18-extra-code-module`：组合的结构和契约的声明本来就在单元一层核对（`TestCommandsComposeNoServerAndNoJobs`、`apitest.Main`）。
- 修订一轮的 `f-lock-swallowed`（邀请的锁失败，停用照样进行；第二步之后是 `s2-dil-swallowed` 的重新映射）、第二步的 `s2-invitations-swallowed`（删除失败）：同清扫 2，组合出的 app 里语句不会失败。

**只在存储一层被发现的变异**：

- 组合一层等价的 11 个谓词，依据组合出的 app 的三条不变式：(a) 成员关系的 `deleted_at` 只由删除工作区写，一个工作区的全部成员关系随它删除（`DeleteWorkspaceMembers`）：`s1-lmw-mdeleted`（工作区已删除的被 `w.deleted_at` 排除）、`s1-sole-mdeleted`、`s1-sole-a-deleted`、`s1-sole-o-deleted`、`s1-ewm-deleted`、`s1-dil-mdeleted`（只问锁住的、未删除的工作区）；(b) 他有效、未删除的成员关系正是 `LockMemberWorkspaces` 锁住的工作区的，增长与停用在账户行、工作区行上串行，每个工作区他至多一行未删除的：`s1-sole-workspaces`、`s1-sole-active`、`s1-ewm-workspaces`；(c') 没有有效成员的未删除工作区没有待接受的邀请：建工作区的人是它的管理员；唯一的有效管理员不能离开（规则 1），那里有别的有效成员时不能被停用（规则 2），移出只由另一位管理员做，所以一个有管理员的工作区失去最后一位有效成员只经他的停用，第五条语句在同一个事务里删除它的待接受邀请；没有有效成员就没有管理员发新的邀请，`reactivate-member` 恢复到这样的工作区的非管理员（第 3 节第 8 条 (b)）之后离开，工作区同样没有待接受的邀请：`s1-dil-workspaces`（不问的工作区里，没有有效成员的那些没有可删的待接受邀请）。第一稿的不变式 (c)"有有效成员的工作区总有有效的管理员"不成立：预检 M1 的路（第五条语句之前）和第 3 节第 8 条 (b) 的 `reactivate-member` 都到达"有成员、没有管理员"，删去；第一稿据它算作等价的 `s1-sole-role` 在 `TestADeactivationDeletesTheInvitationsOfAWorkspaceItEmpties` 的最后一步失败（`reactivate-member` 恢复的两位成员中，dave 不是管理员，他的停用照样通过）。另 `o-spike15-locked-deleted`：等锁时被删除的工作区留在回答里，之后的每条语句都跳过它（成员关系已删除、项目已删除），写的行相同。
- `o-prj-locked-deleted`：组合一层到不了（第 3 节第 5 条）。
- 修订一轮的第六条语句（"修订一轮"一行）只在存储一层被发现的 7 个：`f-lid-mdeleted` 照不变式 (a) 等价（问到的工作区都锁住、未删除，成员关系的 `deleted_at` 只随工作区的删除写）；`f-lid-workspaces` 照 (c') 等价（不问的工作区里，没有有效成员的那些没有待接受的邀请可锁）；`f-lid-swallowed` 同清扫 19；`f-lid-deleted`、`f-lid-email`、`f-lid-responded`、`f-lid-guest-ignored` 多锁两条删除不写的行（已删除的、发给别的地址的、已忽略的、只有访客作为别的成员的工作区的待接受邀请）：写的行不变，只多出等待，组合的世界里没有别的事务持有这些行；存储测试读出每一行的锁，它们在那里失败。组合一层的阶梯要展示多锁，需要再加四种邀请行，修订一轮没有加（报告写明）。第二步之后其中两个也多删，组合测试读到，见"修订一轮的第二步"一段；那里另有 `s1-dit-deleted`、`s1-dil-deleted` 两个各只去掉一处未删除的，只在存储一层。
- 清扫 19 的 5 个：组合出的 app 里语句不会失败。

**故事看不到的谓词**（W9 单独运行；每个都由存储测试发现，标 * 的另由组合测试发现）：锁工作区时多锁、少锁不改变写的行（`s1-lmw-correlated`*、`s1-lmw-member`*、`s1-lmw-active`*）；上面组合一层等价的 11 个；W9 里 B 唯一管理的 acme 没有已结束的管理员（`s1-sole-a-active`*），他独自一人的 epsilon 没有已结束的成员（`s1-sole-o-active`*）；他不是管理员的 beta 另有有效的管理员 A（`s1-sole-role`*：反例要一个有成员、没有管理员的工作区，只有第 3 节第 8 条 (b) 的 `reactivate-member` 到得了）；他在锁住的工作区里没有已结束的成员关系（`s1-ewm-active`*）；epsilon 没有已忽略的、已删除的邀请，也没有已结束的成员（`s1-dil-responded`*、`s1-dil-deleted`*、`s1-dil-active`*）；项目的 `SoleAdmin` 跨工作区的集合形式要两个工作区各有一个他管理的项目、一处另有管理员（`s20-prj-admin-any`*、`s20-prj-other-any`*）。W9 要同时展示这些会变成另一个世界；组合的世界（`deactivationWorld`、`memberWorld`）正是为它们写的。修订一轮的 W9 加了 D 的两份待接受邀请（beta 的、gamma 的）和 B 已接受的那一份（停放项 P27），第一稿看不到的 `s1-dit-email`、`s1-dit-deleted` 现在在 W9 失败（第二步之后 `s1-dit-deleted` 是 `DeleteInvitations` 的未删除，锁的语句的未删除仍在，W9 又看不到；两处一起去掉的在 W9 失败）；停放项 P27 也要 `s1-dil-workspaces` 在 W9 失败，它是 (c') 之下组合一层等价的：没有一份待接受的邀请在一个没有有效成员、又不在他的锁之下的工作区里，接口造不出那样的状态，W9 照样看不到。第六条语句的变异不在 W9 运行。

**变异表（按缺陷类别）**：

变异的定义在 `$M3TMP/p6tools/mutants_p6.py`（119 个）、`mutants_p6_rerun.py`（加强之后的重跑，23 个）和 `mutants_p6_amend.py`（修订之后的 116 个：杀死它的测试改过的 93 个重跑，`s1-sole-role`，新的 22 个）；`archmut.py` 跑 `archtest` 的 2 个；结果在 `p6tools/logs`（`mutants_p6.out`、`mutants_p6-all.json`、`mutants_p6_rerun-all.json`、`archmut.json`、`mutants_p6_amend.out`、`mutants_p6_amend-all.json`、每个变异的 `mutant-*.log`），`mut_tables.py` 把重跑按层合进第一轮。修订的重跑里没有一个变异少了它原来被发现的层；多了的五个：`s1-sole-role` 在组合一层，`s1-sole-o-correlated`、`s1-sole-o-other`、`s20-sole-other-any`、`s24-ws-alone-refused` 在 W9（epsilon）。

终审之后的修订一轮：`$M3TMP/p6fix/mutants_table.py` 把上面三份合成表的 141 行（重跑的层按层合进，修订的文字优先；`s9-check-after-end`、`s11-refusal-after-invitations` 的文字按新的一步重写），`fixrun.py` 在每行写的每一层各跑一次（`-failfast`：一层由它的第一个失败确认，失败的测试一栏只记那一个；`archtest` 的 2 个照旧由 `archmut.py` 在副本上跑），结果在 `p6fix/logs/mutants_table-all.json`、`archmut.json`：141 个都被发现；135 行的层不变，6 行变了：`s1-dit-email`、`s1-dit-deleted` 多了 W9（停放项 P27），`s14-ewm-keeps-writer` 多了 W9（`fcaf8f17` 里 A 重写 B 的三行之后），`s8-dit-pool` 多了存储（邀请的锁的存储测试），`s9-end-before-invitations`、`m1-after-members` 少了组合，只在单元一层（理由在"只在单元一层被发现的变异"）。邀请的锁的 26 个定义在 `p6fix/mutants_fix.py`，在存储、组合（整个包）、单元各跑（`FULL=1`），结果在 `p6fix/logs/mutants_fix-*.json`。停放项各自的变异在 `p6fix/mutants_partb.py`，在改过的那个测试上跑，逐个写在修订一轮的报告里，不进这张表。

| 缺陷类别 | 变异 | 结果 | 失败的测试 | 层 |
|---|---|---|---|---|
| P6：锁的顺序和强度 | `o-ws-{desc,unordered,share,update,unlocked}`、`o-prj-per-workspace`、`o-prj-{unordered,share,update}`、`o-clock-early`、`o-projects-first`、`o-account-{share,update}` | 13/13 | `TestLockMemberWorkspaces`、`TestLockMemberWorkspacesLocksInIDOrder`、`TestEachLockOfADeactivationIsItsStrength`、`TestADeactivationAndACreationHeLeadsSerialize`、`TestADeactivationAndTheProjectSidesGrowthSerialize`、`TestADeactivationFindsWhatChangedMeanwhile`、`TestAnAdmittedAdminsWriteAndHisDeactivation`、等 19 个 | 存储；组合；单元 |
| P6：spike 15（等锁时被删除的跳过） | `o-spike15-{404,500}`、`o-spike15-locked-deleted`、`o-prj-locked-deleted` | 4/4 | `TestLockMemberWorkspacesLeavesOutAWorkspaceDeletedWhileItWaited`、`TestADeactivationFindsWhatChangedMeanwhile`、`TestLockMemberWorkspaces`、`TestEndingAMembersProjectMemberships`、`TestLockActiveMemberProjectsLeavesOutAProjectDeletedWhileItWaited` | 存储；组合 |
| 修订一轮：邀请的锁（终审的 I1） | `f-lid-{deleted,email,workspaces,responded,and,correlated,other,active,mdeleted,other-any,guest-ignored,unordered,desc,share,update,unlocked,pool,swallowed}`、`f-I1-dropped`、`f-lock-after-{dit,dil,members}`、`f-clock-before-lock`、`f-lock-swallowed`、`f-lock-no-{workspaces,address}` | 26/26 | `TestLockInvitationsToDeleteLocksWhatTheDeletesWrite`、`TestLockInvitationsToDeleteLocksInIDOrder`、`TestAFailedWriteIsAnError`、`TestTwoDeactivationsWithCrossedInvitationsBothGoThrough`、`TestEachLockOfADeactivationIsItsStrength`、`TestEachWriteReadsTheClockUnderItsLock`、`TestDeactivateMembershipsFailsAtEachStep` | 存储；组合；单元 |
| P6：锁下的邮箱；命令的期限和中断 | `o-email-before-lock`、`o-cli-{deadline,uninterruptible}` | 3/3 | `TestADeactivationReadsTheAddressUnderItsLock`、`TestAnInterruptedDeactivationChangesNothing` | 组合 |
| P6：两个时刻的先后（预检 L2） | `l2-cli-identity-ahead`、`l2-api-identity-ahead` | 2/2 | `TestADeactivationEndsEveryMembership` | 组合 |
| 清扫 36：第五条语句（预检 M1） | `m1-dropped` | 1/1 | `TestDeactivateMembershipsEndsEveryMembership`、`TestDeactivateMembershipsFailsAtEachStep`、`TestEachWriteReadsTheClockUnderItsLock`、`TestADeactivationDeletesTheInvitationsOfAWorkspaceItEmpties`、`TestEachLockOfADeactivationIsItsStrength`、W9 | 单元；组合；端到端 |
| 清扫 1：谓词 | `s1-lmw-{correlated,member,active,mdeleted}`、`s1-sole-{workspaces,member,role,active,mdeleted}`、`s1-sole-a-{correlated,other,role,active,deleted}`、`s1-sole-o-{correlated,other,active,deleted}`、`s1-dit-{email,deleted}`、`s1-ewm-{workspaces,member,active,deleted}`、`s1-dil-{workspaces,responded,deleted,correlated,other,active,mdeleted}` | 31/31 | `TestLockMemberWorkspaces`、`TestEachLockOfADeactivationIsItsStrength`、`TestSoleAdminOfAWorkspace`、`TestADeactivationAndACreationHeLeadsSerialize`、`TestADeactivationAndTheProjectSidesGrowthSerialize`、`TestADeactivationDeletesTheInvitationsOfAWorkspaceItEmpties`、`TestADeactivationEndsEveryMembership`、等 16 个 | 存储；组合；端到端 |
| 清扫 2：端口的错误（预检 L1 在内） | `s2-lock-swallowed`、`s2-sole-swallowed`、`s2-sole-failure-404`、`s2-invitations-swallowed`、`s2-end-swallowed`、`s2-projects-{swallowed,retried}`、`s2-id-memberships-swallowed`、`s2-dil-swallowed`、`pf-id-runs-after-failure`、`pf-id-retries-memberships` | 11/11 | `TestDeactivateMembershipsFailsAtEachStep`、`TestARefusedDeactivationChangesNothing`、W9、`TestDeactivateByEmailWhenAWriteFails`、`TestDeactivateWhenAWriteFails`、`TestARestoredMembershipGivesNoMoreThanItHad`、`TestAnAdmittedAdminsWriteAndHisDeactivation`、等 2 个 | 单元；组合；端到端 |
| 清扫 3、14、26：写不动的行、由谁、不会碰巧的值（预检 L7 在内） | `s3-dit-writes-created`、`s3-dit-answers`、`s3-dit-keeps-time`、`s14-dit-keeps-writer`、`s14-dit-by-inviter`、`s3-ewm-keeps-time`、`s14-ewm-keeps-writer`、`s3-ewm-writes-created`、`s3-ewm-deletes`、`s26-ewm-role-{20,15,default}`、`s3-dil-keeps-time`、`s14-dil-keeps-writer`、`s3-dil-writes-created`、`s3-dil-answers`、`l7-dit-keeps-declined` | 17/17 | `TestDeleteInvitationsTo`、`TestADeactivationEndsEveryMembership`、`TestDecliningAndDeactivating`、`TestADeactivationFindsWhatChangedMeanwhile`、`TestADeactivationReadsTheAddressUnderItsLock`、`TestAcceptingAndDeactivating`、W9、等 8 个 | 存储；组合；端到端 |
| 清扫 4：接线 | `s4-app-skips-memberships`、`s4-users-skips-memberships`、`s4-users-no-project-end`、`s4-users-frozen-clock`、`s4-server-no-project-end`、`s4-server-frozen-clock`、`s4-newcascade-ends-nothing`、`s4-users-builds-workspace-admin`、`s4-workspaces-builds-cascade` | 9/9 | `TestADeactivationDeletesTheInvitationsOfAWorkspaceItEmpties`、`TestADeactivationEndsEveryMembership`、`TestADeactivationFindsWhatChangedMeanwhile`、`TestADeactivationReadsTheAddressUnderItsLock`、`TestADeactivationRefusedAtItsCommitChangesNothing`、`TestARefusedDeactivationChangesNothing`、`TestEachLockOfADeactivationIsItsStrength`、等 20 个 | 组合；端到端；单元 |
| 清扫 6、30：文档和说明 | `s6-ws-detail`、`s6-prj-detail`、`s6-cli-line`、`s6-two-moments` | 4/4 | `TestLeaveWorkspaceRefusals`、`TestARefusedDeactivationChangesNothing`、`TestUsersCommandsFail`、W9、`TestLeaveProject`、`TestADeactivationDeletesTheInvitationsOfAWorkspaceItEmpties`、`TestADeactivationEndsEveryMembership`、等 11 个 | 单元；组合；端到端 |
| 清扫 8：事务的连接 | `s8-lmw-pool`、`s8-sole-pool`、`s8-dit-pool`、`s8-ewm-pool`、`s8-dil-pool` | 5/5 | `TestLockMemberWorkspaces`、`TestADeactivationAndACreationHeLeadsSerialize`、`TestADeactivationAndTheProjectSidesGrowthSerialize`、`TestEachLockOfADeactivationIsItsStrength`、`TestTheDeactivationRunsOnItsTransactionsConnection`、`TestTwoAdminsDeactivatedAtOnceLeaveAnAdmin`、`TestADeactivationRefusedAtItsCommitChangesNothing`、等 3 个 | 存储；组合；端到端 |
| 清扫 9、11：顺序、拒绝路径 | `s9-end-before-invitations`、`s9-check-after-end`、`s11-refusal-after-invitations`、`s9-memberships-first`、`m1-after-members` | 5/5 | `TestDeactivateMembershipsEndsEveryMembership`、`TestDeactivateMembershipsFailsAtEachStep`、`TestEachWriteReadsTheClockUnderItsLock`、`TestEachLockOfADeactivationIsItsStrength`、`TestDeactivateMembershipsRefusesTheOnlyAdmin`、`TestARefusedDeactivationChangesNothing`、`TestAnAdmittedAdminsWriteAndHisDeactivation`、等 7 个 | 单元；组合；端到端 |
| 清扫 10：种子行 | `s10-dave-{member,stays}`、`s10-bob-member-of-beta`、`s10-gamma-dave-stays`、`s10-solo-standing`、`s10-gamma-pending` | 6/6 | `TestADeactivationDeletesTheInvitationsOfAWorkspaceItEmpties`、`TestADeactivationEndsEveryMembership`、`TestADeactivationFindsWhatChangedMeanwhile`、`TestADeactivationReadsTheAddressUnderItsLock`、`TestADeactivationRefusedAtItsCommitChangesNothing`、`TestARefusedDeactivationChangesNothing`、`TestAnInterruptedDeactivationChangesNothing`、等 2 个 | 组合 |
| 清扫 15、16、37：标题、假对象的顺序、归档项目 | `s15-dit-pending-only`、`s16-workspaces-reversed`、`s37-archived-left` | 3/3 | `TestDeleteInvitationsTo`、`TestADeactivationEndsEveryMembership`、`TestADeactivationFindsWhatChangedMeanwhile`、`TestDecliningAndDeactivating`、`TestEachLockOfADeactivationIsItsStrength`、W9、`TestDeactivateMembershipsEndsEveryMembership`、等 4 个 | 存储；组合；端到端；单元 |
| 清扫 18：契约 | `s18-undeclared-{workspace,project}`、`s18-extra-code-module` | 3/3 | `TestDeactivateMeProblems`、`TestARefusedDeactivationChangesNothing` | 单元；组合 |
| 清扫 19：存储方法的失败 | `s19-lmw-swallowed`、`s19-sole-swallowed`、`s19-dit-swallowed`、`s19-ewm-swallowed`、`s19-dil-swallowed` | 5/5 | `TestAFailedReadIsAnErrorNotAnAnswer`、`TestAFailedWriteIsAnError` | 存储 |
| 清扫 20：集合形式 | `s20-sole-admin-any`、`s20-sole-other-any`、`s20-prj-admin-any`、`s20-prj-other-any`、`s20-dil-other-any` | 5/5 | `TestSoleAdminOfAWorkspace`、`TestARefusedDeactivationChangesNothing`、W9、`TestADeactivationDeletesTheInvitationsOfAWorkspaceItEmpties`、`TestADeactivationEndsEveryMembership`、`TestEachLockOfADeactivationIsItsStrength`、`TestSoleAdmin`、等 9 个 | 存储；组合；端到端 |
| 清扫 21、22：第一个问题、回答对得上所问 | `s21-id-wrapped-404`、`s21-ws-joined-404`、`s22-api-no-address`、`s22-cli-raw-address` | 4/4 | `TestDeactivateByEmailWhenAWriteFails`、`TestDeactivateWhenAWriteFails`、`TestARefusedDeactivationChangesNothing`、`TestAnAdmittedAdminsWriteAndHisDeactivation`、`TestTwoAdminsDeactivatedAtOnceLeaveAnAdmin`、`TestUsersCommandsFail`、`TestDeactivateMembershipsRefusesTheOnlyAdmin`、等 7 个 | 单元；组合 |
| 清扫 24：每一半 | `s24-ws-only-ignored`、`s24-ws-alone-refused`、`s24-prj-ended-admin-counts`、`s24-projects-first-workspace`、`s24-check-first-workspace` | 5/5 | `TestSoleAdminOfAWorkspace`、`TestADeactivationEndsEveryMembership`、`TestADeactivationFindsWhatChangedMeanwhile`、`TestADeactivationReadsTheAddressUnderItsLock`、`TestADeactivationRefusedAtItsCommitChangesNothing`、`TestARefusedDeactivationChangesNothing`、`TestAcceptingAndDeactivating`、等 15 个 | 存储；组合；端到端；单元 |
| 清扫 25、31、32：检查和执行在一把锁下、探针的反例 | `s31-check-before-lock`、`s32-wait-moved`、`s32-wait-moved-blind` | 3/3 | `TestLockMemberWorkspaces`、`TestLockMemberWorkspacesLeavesOutAWorkspaceDeletedWhileItWaited`、`TestLockMemberWorkspacesLocksInIDOrder`、`TestADeactivationAndTheProjectSidesGrowthSerialize`、`TestADeactivationFindsWhatChangedMeanwhile`、`TestAnAdmittedAdminsWriteAndHisDeactivation`、`TestEachLockOfADeactivationIsItsStrength`、等 2 个 | 存储；组合 |
| 清扫 39：排序规则 | `s39-endings-collation`、`s39-w9-collation` | 2/2 | `TestADeactivationFindsWhatChangedMeanwhile`、`TestADeactivationReadsTheAddressUnderItsLock`、W9 | 组合；端到端 |
