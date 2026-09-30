# M3 工作区与项目：设计与实施规划

| 项 | 内容 |
|---|---|
| 里程碑 | M3 工作区与项目（`docs/v0/M3-workspace-project`） |
| 日期 | 2026-09-29 |
| 状态 | 第三稿，已按聚焦复核修订（17.3）。第 10 节的四个决策点都已由负责人裁定（2026-09-29：1 选 A+(a) 且查看不显示邮箱，2 选 A，3 选 A，4 选 A）；第 11 节的七个架构问题都已由负责人批准（2026-09-29） |
| 上级文档 | [v0 总体设计](../v0-design.md) 1.1、3、4、5、6、7、8、9 节；[差异清单](../plane-diff.md)；[前端改动清单](../frontend-changes.md)；[M2 设计](../M2-auth/M2-design.md)（它定下的平台约定约束 M3，本文照做，不重新裁定；要修订的一处写在 11.7） |
| 前置交接 | `handoffs/` 中的 6 份：M2-closeout（14 节）、M1-closeout、M1-P2、M1-P3、M1-P4、M0-P3（已完成）。另有别的 M 的交接中点名 M3 的几项（M4 的 M2-closeout 第 1、5、8 节，M6、M7 的删除关系图，M5、M8 的 M1-closeout 各一项），以及 M3 不涉及的 M4 的 M2-closeout 第 6 节。逐条落到 Phase，见第 13 节 |
| 设计评审 | 第一稿经独立评审（Ready with fixes：Important 11、Minor 29），第二稿全部落实，见 17.1；第二稿经 Codex 对抗性评审（[报告](reviews/M3-design-codex-adversarial-review.md)：Critical 0、Important 3、Minor 5），第三稿全部落实，每一条都落为正文的规则和一个指定 Phase 的测试，见 17.2；第三稿经聚焦复核（Ready with fixes：Critical 0、Important 2、Minor 9），全部落实，见 17.3 |

---

## 0. 目标与范围

### 0.1 目标
M3 是第一个有多个业务模块、第一次跨模块协作的里程碑，也是前端第一次整块替换一个领域的 stores。M3 结束时：
1. 任何调用方都能通过接口创建和管理工作区，邀请成员（凭邀请链接接受），管理成员和角色；创建、归档、恢复、删除项目，管理项目成员、状态、标签和个人的显示设置。用 PAT 能做的事和页面一样多。
2. **权限框架**：`internal/shared` 有 `Authorizer` 端口，`access` 模块持有规则表并实现它；每个操作经它判定"看不到（404）、看得到但不能做（403）、能做"。一张完整的"身份 × 操作"矩阵逐格测试（总体设计 6.5）。
3. 第一次跨模块协作，定下以后各 M 照做的写法：
   - 模块之间只经端口协作（总体设计 6.3 第 2 条）；M3 的连带（删除工作区、降为访客、结束成员关系）都经 `workspace` 调 `project` 的一个端口完成，领域事件留给第一个有异步订阅者的 M4（3.3）；
   - 全局加锁顺序延伸到 M3 的表，写事务**先锁父行，再判定权限**；一个账户的成员关系集合只经一个协议增长和收缩，与停用串行（3.6 约定六）；
   - `identity` 的两个扩展点由 M3 接上：注册策略认邀请令牌，停用账户经端口处理成员关系。
4. 前端把工作区、成员、邀请、项目、项目成员、状态、标签、显示设置的 services 和 stores 换成生成的客户端和类型，按总体设计 7.7 分代，没有转换层。M3 能到达的页面挂载时不请求 M4 及以后的旧接口。
5. 本领域的每个用户故事都有端到端测试：页面、数据库两层断言（M3 没有异步结果，见第 2 节）；接口版本用 PAT，调用同一组数据库断言。每个故事的前置数据在它落地的 Phase 都能经接口准备。
6. 持续集成的全部门禁通过；交给 M3 的交接全部关闭；给 M4–M8 的交接写清接收条件。

### 0.2 范围
| 包含 | 不包含（留给后续 M） |
|---|---|
| 工作区：创建（接口和 `nerve workspaces create` 命令）、列出、查看、修改、删除；工作区名（slug）的可用性检查和保留名单 | 工作区图标、项目封面的存储和上传（M5）。M3 的接口中 `logo_url`、`cover_image_url`、成员头像恒为 `null`，上传控件删除，见 3.2 |
| 成员：列出、改角色、移出、离开；访客降级时项目角色随之降为访客；恢复被移出的成员（`nerve workspaces reactivate-member`，照 Plane 的管理命令，3.11） | 工作项、工作项的筛选和显示设置（M4）。个人显示设置中属于工作项列表的部分（筛选条件、显示属性）由 M4 按自己的格式加列，见 3.18 |
| 邀请：管理员邀请、改角色、删除、复制链接；受邀的人凭链接查看、接受、忽略；关闭注册时凭有效邀请注册（第 10 节决策点 1、2、4） | 收藏、最近访问、保存的视图、跨项目的标签列表（M7）；工作区首页的"最近"小部件（M7，见 3.1） |
| 项目：创建（默认 6 个状态）、列出（按可见性）、查看、修改、删除、归档、恢复；项目标识的可用性检查 | 打开 `intake_view` 时建默认收集箱、分诊状态的接口（M7）；迭代、模块（M6） |
| 项目成员：从工作区成员中添加、改角色、移出、加入公开项目、离开 | 删除状态前检查它有没有工作项、删除标签时处理工作项上的标签（M4） |
| 状态：列出、创建、修改、删除、设为默认；工作区内自己所在项目的状态 | 60 天物理清理（M4 的第一个业务定时任务），见 4.12 |
| 标签：列出、创建、修改（含两层的父子关系）、删除 | 规则表的"创建者本人可改"（`AllowCreator`）和 `guest_view_all_features` 对工作项的约束（M4，它们的第一个使用者） |
| 个人显示设置：工作区的项目导航偏好；项目的默认标签页、隐藏的标签页、侧边栏里项目的顺序 | 领域事件和它的订阅者（M4，M2 设计 3.15）；Webhook、变更事件的对外投递（M8） |
| 权限框架（`shared.Authorizer`、`access` 模块、规则表）和完整的权限矩阵 | 邮件（v0 不做）：邀请不发邮件，由管理员复制链接发给对方 |
| 前端：落点、新手引导的创建工作区和邀请成员两步、邀请页、工作区设置、项目列表和项目设置、成员、状态、标签；挂载时的规则延伸到 M3 的页面；表情选择器的数据从本站提供；M3 领域的死成员、死 prop 和 oxlint | 工作项页面、个人主页的工作项列表（M4）。M3 的页面上指向它们的链接是已知的过渡，见 3.1 |

### 0.3 关于文中的 spike
文中的"spike"是写设计时在仓库之外做的验证（放在临时目录，不进仓库）。结论、实测的数字和关键的行号都已写在正文中，读者不需要那些文件。
- **表结构**（4.2–4.10）：在开发库 `nerve-dev-db-1`（Postgres 18.6）里建一个临时库，建出第 4 节的九张表，逐条试约束，试完删库。
- **行锁的冲突**（3.6）：同一个临时库，两个会话交错，一方持锁 2 秒，另一方 `lock_timeout = 500ms`。
- **sqlc**（M2 设计 3.14 的延伸）：用仓库锁定的 sqlc v1.31.1（`CGO_ENABLED=0 go -C server tool -modfile=tools/go.mod sqlc generate -f <临时配置>`）为这九张表和 10 条有代表性的查询生成代码。
- **前端的测量**（第 1 节、7.11）：M1 收尾的 `deadsym.mjs`、`domains.mjs`（全文在 [M1 收尾计划](../M1-frontend-trim/plans/closeout.md)附录 A）；按 `domains.mjs` 分领域统计 oxlint；按文件组数行数；`git grep` 数旧接口的调用和 store 的使用方。
- **表情选择器**（7.7）：读 `frimousse` 0.3.0 的构建产物，确认它怎样拼数据的地址；量 `emojibase-data` 15.3.2 的文件大小。
- **第一稿评审的 spike**（评审者做，临时库都已删除）：S1a 先判定、后锁父行时两位管理员互相降级，结果一个管理员都不剩；S1b 先锁父行、后判定，后到的一方被拒绝；S2 两次改父标签都只持项目的 `FOR SHARE`，结果成环；S3 `CREATE DATABASE … TEMPLATE` 每次约 83 毫秒，16 路并行 205 毫秒（9.2）。
- **第二稿的 spike**（3.6）：一方软删除项目并持锁，另一方 `SELECT … WHERE deleted_at IS NULL FOR SHARE` 等到对方提交之后读到 0 行；一方把状态设为默认并持锁，另一方带 `AND NOT "default"` 的软删除等到之后删除 0 行，两个状态都还在、恰好一个默认。
- **第二稿的 Codex 评审的 spike**（评审者在自己新建、事后删除的容器里做，报告 1.2 节）：S1 停用先列举工作区、再等正在接受的邀请，结果项目成员关系仍有效、工作区没有管理员（3.6 约定六）；S2 恢复成员之后接受旧的访客邀请，结果工作区没有管理员、访客是项目管理员（3.8）；S3、S4 复现了第二稿的两个 spike；S5 `logo_props` 只查对象类型时未知的键和错的类型都能写入（4.6）；S6 sqlc 只装载 `workspace` 的 schema 时 JOIN `users` 失败（6.5）；S7 `bootstrap` 返回的码补不了 `identity` 的 HTTP 测试进程的记录（9.4）。
- **第三稿的 spike**（4.6）：`logo_props` 按 M2 设计 3.13 写的 CHECK。`{}` 和三种合法的值通过；S5 的两个不合法的值（它的第三个值 `{}` 合法，照样通过）、未知的嵌套键、嵌套值的类型不对、数组、字符串、`null` 都得到 `check_violation`（23514）。嵌套的 `->` 和 `-` 要加括号：`-` 的优先级高于 `->`，不加时 `'emoji' - array[…]` 先算，报 JSON 语法错误。
- **第三稿的聚焦复核的 spike**（复核者在 `nerve-dev-db-1` 的临时库里做，事后删除；每次等待都经 `pg_stat_activity` 和 `pg_blocking_pids` 确认）：约定六的每一对增长与收缩在两种顺序下都串行，没有死锁、没有丢失的不变式。另外找到：
  - 9d：B 是有效成员、有一份待接受的访客邀请，A 移出 B 之后，B 凭旧链接以访客回到工作区（3.8"结束的成员关系不留下邀请"）；
  - 12a、12b：降为访客与加入项目在 B 的工作区成员行上串行（交错测试 17）；
  - 14a：两位管理员按相反的顺序创建重叠的两批邀请，在唯一索引上互相等待，死锁；14b：两批都排序之后不死锁，后到的一方得到原始的 23505（3.6 约定五、3.8）；
  - 15：停用列举到的一个工作区在它上锁之前被删除，锁的语句少返回一行（3.9）；
  - 16：忽略邀请在工作区的锁之后才锁账户行，与停用死锁（3.6 约定一）；
  - 17：第三稿的"加入取现在的工作区角色"让被降为访客的项目成员离开再加入，回来是 15（3.5）。

---

## 1. 现状基线（2026-09-29 实测，`main` 为 `f8cb7c2`）

| 项 | 数值与出处 |
|---|---|
| 服务端的模块 | `identity`、`instance`（`server/internal/modules/`）。`internal/shared` 有 `Actor`、领域错误、`TxManager`、游标的封套（`actor.go`、`error.go`、`tx.go`、`cursor.go`），还没有 `Authorizer` |
| 迁移 | 5 个：`00001`–`00004` 是 `identity` 的四张表，`00005` 是 River（`server/migrations/sql/`）。`server/sqlc.yaml` 只有 `identity` 一个条目 |
| 接口 | 15 个操作：`identity` 13 个（`api/modules/identity.yaml`，623 行），`instance` 2 个。平台码 9 个，没有 `forbidden`（`server/internal/platform/httpserver/apitest/problems.go:26-29`）；带前缀的码必须用所在模块文件的名字（`apitest/rules_test.go:153-162`，M2 设计 3.11 第 1 条） |
| 注册策略 | 端口 `SignupPolicy.AllowSignup(ctx)`（`server/internal/modules/identity/app/ports.go:352-355`），`bootstrap` 接成配置开关 `signupSwitch`（`server/internal/bootstrap/app.go:191-194`） |
| 停用账户 | `identity/app/deactivate.go` 的 `deactivate` 依次写 `users`、`profiles`、`auth_sessions`；没有成员关系的端口（M2 决策点 3） |
| 创建工作区的开关 | 配置 `workspace.creation_enabled`，默认 `true`（`server/configs/config.yaml:83`），只出现在 `GET /api/v0/instance`；服务端没有执行它 |
| Plane 的管理命令 | `plane/apps/api/plane/db/management/commands/`：`activate_user.py`、`reactivate_workspace_member.py:13-90`（按 slug 和邮箱恢复无效的工作区成员关系，提示项目成员关系仍无效）、`create_project_member.py:20-62` |
| Plane 的表 | `workspaces` 14 列、`workspace_members` 17 列、`workspace_member_invites` 13 列、`workspace_user_properties` 14 列、`projects` 36 列、`project_members` 16 列、`project_user_properties` 15 列、`states` 18 列、`labels` 15 列（`tools/plane-schema/plane-v1.4.2-schema.sql`）。外键都没有 `ON DELETE`；唯一约束同时有 Django 的 `(…, deleted_at)` 全表唯一和 `WHERE deleted_at IS NULL` 的部分唯一两套；`workspace_slug_key UNIQUE (slug)` 是全表唯一 |
| M3 领域的前端代码 | 229 个文件、23,485 行（不含测试），按文件组的明细见 7.11。其中侧边栏的收藏（M7）、工作区视图（M7）的部分不是 M3 要改的 |
| M3 要替换的旧接口调用 | 按 service 方法数共 56 个：`workspace.service.ts` 23、`project.service.ts` 9、`project-archive.service.ts` 2、`project-member.service.ts` 6、`project-state.service.ts` 7（另 1 个分诊状态属于 M7）、`issue_label.service.ts` 5、`user.service.ts` 4（`/settings/`、离开工作区、加入项目、离开项目）。`/api/users/` 下的调用全部是 M3 的（7 处，`git grep -n "/api/users/" -- web`） |
| 模块级的 service 实例 | M3 领域 10 处（M2 交接第 3 节），都经不带令牌的 axios 基类调 Plane 的地址，nerve 答 404 |
| store 的使用方 | `useWorkspace` 38 个文件、`useProject` 129、`useMember` 70、`useProjectState` 30、`useLabel` 25、`useUserPermissions` 122、`useUserSettings` 6（`grep -rlw` 于 `web/apps/web/{core,app}`） |
| Plane 类型的使用方 | 不含 `packages/types` 自身：`IProject` 20 个文件、`IState` 23、`IIssueLabel` 25、`IWorkspace` 19、`IUserLite` 14、`TProject` 11、`IWorkspaceMemberInvitation` 8、`TProjectMembership` 7、`IWorkspaceMember` 6、`EUserProjectRoles` 27、`EUserWorkspaceRoles` 10 |
| 死成员和死 prop | `deadsym.mjs` → `prop 429, member 836, export 2`；`domains.mjs --rows M3` → 208 行（成员 140、prop 68）。其中 41 行按路径归到 M3、实际属于别的 M：`packages/types/src/workspace-notifications.ts` 17 行（M7）、`workspace-draft-issues/base.ts` 16 行（M4）、`core/sidebar/progress-stats/` 8 行（M6）。M3 自己的是 167 行，最多的是 `packages/types/src/workspace.ts` 39 行、`store/project/project.store.ts` 11 行、`types/src/project/projects.ts` 10 行 |
| oxlint | 共 568 条警告（web 452、editor 65、ui 19、utils 12、propel 16、constants 1、hooks 3，与各包 `check:lint` 的上限相等）。按 `domains.mjs` 的划分，M3 领域 73 条，分布在 39 个文件：`no-shadow` 18、`promise(always-return)` 14、`jsx-a11y(no-static-element-interactions)` 7、`react-hooks(exhaustive-deps)` 6、`click-events-have-key-events` 5、`tabindex-no-positive` 4 等。全仓按规则：`no-shadow` 165、`no-static-element-interactions` 70、`promise(always-return)` 62、`click-events-have-key-events` 61、`exhaustive-deps` 39 |
| 挂载时请求旧接口的地方 | 工作区的每一页：`WorkspaceAuthWrapper` 取收藏（M7，`core/layouts/auth-layout/workspace-wrapper.tsx:100-104`），以及 M3 自己的项目角色（`project-roles`）、项目列表、工作区的状态（`:81-92,106-111`）；`TopNavigationRoot` 取未读通知数（M7，`core/components/navigation/top-navigation-root.tsx:29-32`）；首页的"最近"小部件（M7，`core/components/home/widgets/recents/index.tsx:45-49`）。项目的每一页：`ProjectAuthWrapper` 取迭代、模块（M6）和视图、分诊状态（M7），以及 M3 自己的显示设置、标签、成员、状态（`core/layouts/auth-layout/project-wrapper.tsx:77-96`） |
| 保留的工作区名 | 前端 `RESTRICTED_URLS` 45 个（`web/packages/constants/src/workspace.ts`）；应用的顶层静态路由段 6 个：`create-workspace`、`invitations`、`onboarding`、`settings`、`sign-up`、`workspace-invitations`（`web/apps/web/app/routes/core.ts`）；`web/apps/web/public/` 顶层有 `icons/`；服务端自己的顶层路径 `api`、`assets`、`healthz`、`readyz` |
| 表情选择器 | `frimousse` 0.3.0 没有收到 `emojibaseUrl`，按 `https://cdn.jsdelivr.net/npm/emojibase-data@latest/<语言>/data.json` 和 `messages.json` 取数据，先发 `HEAD` 比对 ETag，数据缓存在 localStorage 的 `frimousse/data/<语言>`（spike，`node_modules/frimousse/dist/index.js`）。页面的 CSP 是 `connect-src 'self'`（`server/internal/platform/webui/csp.go:42`），这个请求被挡住。`emojibase-data` 15.3.2 已在依赖树里（`@tiptap/extension-emoji` 带进），`en/data.json` 700,120 字节、`en/messages.json` 6,500 字节 |

---

## 2. 用户故事（M3 的验收范围）

每个故事一个 Playwright 测试文件：工作区一侧放在 `e2e/stories/workspace/`（W1–W12），项目一侧放在 `e2e/stories/project/`（P1–P9）。按总体设计 8.2 断言。

**约定**：
- **两层断言**：页面和数据库。M3 不投递任何 River 任务，也没有通知和 Webhook，"异步结果"一层在 M3 为空；3.9、3.11 说明为什么停用的端口和建工作区都不投递任务。
- **接口版本**：一律用 PAT；页面版本和接口版本调用同一个断言函数（`e2e/fixtures/assert/workspace.ts`、`assert/project.ts`，按表分文件）。结果因凭证种类而不同的地方由参数表达，写在故事里。
- **前置数据**经接口准备（总体设计 8.2），不写 SQL：`e2e/fixtures/api.ts` 加上建工作区、邀请并接受、建项目、加项目成员的帮助函数，都走生成的客户端。最后一列是故事的接口版本和页面版本各在哪个 Phase 加入：每个故事都排在它的前置数据能经接口准备的 Phase 或之后（第 12 节）。
- **命令行**的故事（W10、W12）没有页面版本。
- 每个有页面的故事都用 `watchPage`（`e2e/fixtures/browser.ts`）断言：没有失败的接口请求、没有发往旧接口的请求（`oldApiRequests`）、控制台安静、没有 CSP 违规（3.1）。故意触发的失败（403、409 等）按请求逐个放行。
- 下表中 403 不带码的都是平台码 `forbidden`（5.3）。

| 编号 | 故事 | 页面 | 数据库 | 接口版本 | Phase（接口 / 页面） |
|---|---|---|---|---|---|
| W1 | 创建工作区 | 新用户完成资料步骤后在新手引导里建工作区（名、slug、规模），再到"邀请成员"一步点跳过，落到工作区首页；已完成引导的用户在 `/create-workspace` 再建一个。slug 已被占用、是保留名、含大写：表单在字段下方提示，不提交。`workspace.creation_enabled = false` 的独立 nerve：两个入口都显示"创建工作区已关闭" | `workspaces` 新增一行：slug 为小写，`created_by_id` 是当前账户；`workspace_members` 新增一行，`role = 20`、`is_active`；`profiles.last_workspace_id` 是新工作区（前端写入，3.14）。没有任何演示数据（3.11） | `POST /api/v0/workspaces`，同一组断言；slug 被占用 409 `workspace.slug_taken`，保留名 422（`slug`，`not_allowed`）；关闭时 403 `workspace.creation_disabled`，数据库不变。`GET /api/v0/workspace-slugs/{slug}` 三种回答（可用、被占用、保留） | P1 / P9 |
| W2 | 登录后的落点 | 有两个工作区的账户：登录后落到上次的工作区；在 general 页删掉上次的工作区，再登录，落到另一个；在另一个工作区先把一位成员提升为管理员，再离开它，之后登录落到 `/create-workspace`。三次都没有失败的请求 | `profiles.last_workspace_id` 随切换工作区而变（工作区菜单切换一次）；删除、离开之后对应的行按 W3、W7 的断言 | `GET /api/v0/workspaces` 只返回仍是有效成员的工作区，含 `role`、`total_members`、`created_at`：删除之后少一个，离开之后再少一个。落点规则是前端的，接口版本只核对列表 | P5 / P9 |
| W3 | 工作区设置 | 管理员在 general 页改名、规模、时区，刷新后仍是新值；成员打开同一页，表单不可编辑。管理员在 general 页删除工作区（输入名称确认），落到下一个工作区或 `/create-workspace`。会话切换：另建一个工作区再删除，测试在 `page.route` 里先经 `route.fetch()` 把删除的请求发到 nerve（此时会话未变，它成功），另一个标签页退出并以另一个账户登录之后，才经 `route.fulfill()` 把这个回答交给页面：原标签页不跳转、不提示（7.1、9.6） | `workspaces` 对应列和 `updated_by_id`；删除后工作区及其成员、邀请、显示设置、项目（和项目之下的行）的 `deleted_at` 在同一时刻写入 | `PATCH`、`DELETE /api/v0/workspaces/{slug}`；成员 403；删除之后原成员 `GET` 得到 404 `workspace.not_found`；slug 不能修改（请求体里带 `slug` 是 400）。P4 起断言加上项目的连带 | P3（P4 补连带）/ P9 |
| W4 | 邀请成员 | 管理员在成员页邀请两个邮箱（一个成员、一个访客），列表出现两条待接受的邀请；改其中一条的角色；复制链接（剪贴板里是 `/workspace-invitations?invitation_id=…&token=…`）；删除另一条。邀请已是成员的邮箱、重复的邮箱：弹窗在对应行提示。一条被忽略的邀请在列表中显示"已忽略"，再邀请这个邮箱被拒绝，删掉它之后才能再邀请。成员打开同一页：只有成员列表，没有邀请的界面，页面也不请求邀请列表（`watchPage` 没有失败的请求，7.1） | `workspace_member_invites` 新增两行：邮箱已规范化、`role` 正确、`accepted = false`、`responded_at` 为空；表中没有令牌；删除的一行 `deleted_at` 已填 | `POST /api/v0/workspaces/{slug}/invitations`（批量）、`PATCH`/`DELETE /api/v0/workspace-invitations/{id}`，同一组断言；成员、访客 403（决策点 4）；已是成员 422（`invitations[i].email`，`not_allowed`），重复或已被忽略 422（`duplicate`） | P3 / P9 |
| W5 | 凭链接接受或忽略邀请 | 已登录、邮箱一致的账户打开链接，看到工作区名和角色（看不到被邀请的邮箱，决策点 1），点"接受"，进入工作区。另一条邀请点"忽略"，页面说明已忽略。用另一个账户打开链接，点"接受"：得到 403 之后，页面说明"这份邀请发给了另一个邮箱"，不说是哪一个，接受和忽略按钮不再可用，提供退出登录（打开时页面无从知道邮箱是否一致：查看不含邮箱）。令牌被改动一位：页面说明链接无效 | 接受：`workspace_members` 新增一行，角色等于邀请的角色；邀请 `accepted = true`、`responded_at` 已填、`deleted_at` 已填。忽略：`accepted = false`、`responded_at` 已填、未删除。邮箱不一致：数据库不变 | `POST /api/v0/workspace-invitations/{id}/accept`、`/decline`，同一组断言；邮箱不一致 403 `workspace.invitation_email_mismatch`，回答里没有被邀请的邮箱；令牌不对、邀请不存在 404 `workspace.invitation_not_found`（两者相同）；已回应 409 `workspace.invitation_responded`。公开的 `GET /api/v0/workspace-invitations/{id}?token=…` 不带令牌是 400，回答里没有 `email` | P3 / P9 |
| W6 | 注册关闭时凭邀请注册（决策点 1） | 注册关闭的独立 nerve（管理员经这个 nerve 建工作区、发邀请，3.8）：未登录打开邀请链接，页面提供"注册以接受"；注册页的标题是"加入 <工作区>"；注册后回到邀请页，接受，完成新手引导的资料一步就进入工作区。不带邀请直接打开注册页：显示"注册已关闭" | 新账户和资料照 A1；接受之后同 W5 | `POST /api/v0/auth/register` 带 `invitation {id, token}`：邮箱与邀请一致时 201；不带、令牌不对、邮箱不一致、邀请已回应或已删除时都是 403 `identity.signup_disabled`（同一句，不说明是哪一种） | P3 / P9 |
| W7 | 成员管理 | 管理员把一个成员改为访客（这个人在两个项目里是成员、管理员），把另一个成员移出。管理员不能改自己的角色（没有入口），不能移出自己。唯一的管理员点"离开工作区"：提示先指定另一位管理员。被移出的成员凭新的邀请再次接受 | 改为访客：`workspace_members.role = 5`，这个人在本工作区全部项目中的 `project_members.role` 都是 5（含已离开的项目）。移出：`workspace_members.is_active = false`，他在本工作区的项目成员关系全部 `is_active = false`，行不删除。再次接受：原来那一行 `is_active` 恢复为真、角色改为邀请的角色 | `PATCH`、`DELETE /api/v0/workspace-members/{workspace_member_id}`、`POST /api/v0/workspaces/{slug}/leave`；改自己、移出自己 409 `workspace.own_membership`；唯一管理员离开 409 `workspace.sole_admin`；移出某个项目唯一的管理员（那个项目还有别的成员）409 `project.sole_admin` | P5 / P9 |
| W8 | 项目导航偏好 | 在工作区侧边栏的"项目导航"对话框里改成标签页式、只显示 3 个项目；刷新后仍生效 | `workspace_user_properties` 新增一行（第一次修改时），两列是新值 | `GET`/`PATCH /api/v0/me/workspaces/{slug}/preferences`；没有这一行时 `GET` 返回默认值，数据库不变 | P2 / P9 |
| W9 | 停用账户与成员关系（M2 交接第 6 节） | 在 general 页停用：这个人是某工作区唯一的管理员、那里还有别的成员时，弹窗里显示"先指定另一位管理员"，账户不变。指定之后再停用，回到登录页 | 被拒绝：数据库完全不变（`users`、`profiles`、会话、成员关系、邀请）。成功：A12 的断言，加上他全部的工作区、项目成员关系 `is_active = false`，发给他邮箱的全部邀请（含已忽略的）`deleted_at` 已填 | PAT 调用 `POST /api/v0/me/deactivate`：409 `workspace.sole_admin` 或 `project.sole_admin`；成功时同一组断言。`nerve users deactivate --email …` 同样被拒绝（退出码 1，输出说明）和成功 | P6 / P9 |
| W10 | 管理员建工作区（3.11） | — | `nerve workspaces create --slug acme --name Acme --admin-email a@…`：同 W1 的 `workspaces`、`workspace_members`；输出一行。slug 被占用、邮箱没有账户、账户已停用：退出码 1，输出说明，数据库不变；`workspace.creation_enabled = false` 时命令照常可用 | 用接口核对：这个账户 `GET /api/v0/workspaces` 能看到它，`role = 20` | P1 / — |
| W11 | 访客的边界（权限矩阵的页面抽样，9.2） | 访客登录：侧边栏只列出他加入的项目；打开工作区设置的成员页，看到"没有权限"；成员列表里的邮箱都不显示，他自己的也不显示；打开项目的状态设置，看到"没有权限" | 数据库不变 | 矩阵的全部格子在 9.2 由后端测试逐格覆盖；这个故事只用 PAT 抽样四格：访客 `GET` 邀请列表 403、`POST /states` 403、`GET` 私密项目 404、成员列表中 `member.email` 为 `null` | P7 / P11 |
| W12 | 恢复被移出的成员（3.11） | — | 管理员移出成员 B（B 是工作区管理员，在一个项目里是项目管理员），再运行 `nerve workspaces reactivate-member --slug acme --email b@…`：`workspace_members` 那一行 `is_active` 恢复为真、角色不变；输出说明 B 的 1 个项目成员关系仍无效。B 的账户已停用时，输出另外提示 `nerve users activate`。已是有效成员：输出说明，退出码 0。工作区不存在、账户不存在、不是这个工作区的成员：退出码 1，数据库不变 | B 用 PAT `POST /api/v0/projects/{id}/join`：原来的项目成员行恢复，角色是 20（原来那一行的 20 与他的工作区角色 20 中较低的，3.5）。**旧邀请**（Codex S2）：B 被移出期间 A 给 B 发了一份访客邀请；恢复、加入项目之后 A 离开，B 成为唯一的管理员；B 接受那份旧邀请：200，回答的 `role` 是 20，邀请 `accepted = true` 并已软删除，`workspace_members.role` 仍是 20，项目角色仍是 20（3.8 规则：邀请从不改变有效的成员关系） | P5 / — |
| P1 | 创建项目 | 成员在项目列表页建项目（名、标识、说明、公开、负责人、图标）；建好后项目出现在列表和侧边栏。标识已被占用、含非法字符：表单提示 | `projects` 新增一行：`identifier` 为大写，`timezone` 等于工作区的时区，`network = 2`，`last_issue_sequence = 0`；`project_members` 新增创建者（`role = 20`）和负责人（`role = 20`，与创建者不同时）；`project_user_properties` 为这两人各一行（`sort_order` 按 3.18）；`states` 新增 6 行，名称、颜色、`sequence`、`group` 照 Plane，`Backlog` 是默认 | `POST /api/v0/workspaces/{slug}/projects`，同一组断言；标识被占用 409 `project.identifier_taken`，名称被占用 409 `project.name_taken`，名称含 `-`、`.` 等 422（`name`，`not_allowed`）；访客 403；负责人是访客或不是成员 422（`project_lead_id`，`not_allowed`）。`GET /api/v0/workspaces/{slug}/project-identifiers/{identifier}` | P4 / P10 |
| P2 | 项目列表与可见性 | 管理员看到全部项目（含私密）；成员看到自己加入的和公开的；访客只看到自己加入的。成员直接打开一个没加入的公开项目的地址：显示"加入项目"，页面只请求项目详情，不请求它的显示设置、标签、成员、状态（`watchPage` 没有失败的请求，7.1）；点"加入"之后进入项目 | 加入：`project_members` 新增一行，角色等于他的工作区角色；`project_user_properties` 新增一行，`sort_order = 65535`（3.18） | `GET /api/v0/workspaces/{slug}/projects`（三种角色三组结果，默认不含已归档）；`POST /api/v0/projects/{id}/join`；成员加入私密项目、访客加入公开项目都是 404 `project.not_found` | P4 / P10 |
| P3 | 项目设置 | 项目管理员在 general 页改名、标识、说明、可见性、时区、图标；在 features 页开关迭代、模块、视图、收集箱；在 members 页设负责人和默认负责人（从项目中不是访客的成员里选），打开"访客可见全部"；在 automations 页设自动归档 3 个月。项目成员打开 general 页，表单不可编辑 | `projects` 对应列、`updated_by_id` | `PATCH /api/v0/projects/{id}`；项目成员 403；`archive_in = 13` 422；负责人、默认负责人是项目访客或不是项目成员 422（`project_lead_id`、`default_assignee_id`，`not_allowed`） | P4 / P10 |
| P4 | 归档与恢复 | 管理员在设置页归档项目：它从侧边栏消失，出现在"已归档的项目"；在那里恢复，再归档，然后删除 | 归档：`archived_at` 已填；恢复：为空；删除：项目及其成员、显示设置、状态、标签的 `deleted_at` 在同一时刻写入 | `POST /archive`、`/unarchive`、`DELETE /api/v0/projects/{id}`；已归档的项目 `PATCH` 409 `project.archived`；`GET /projects?archived=true` 只返回已归档的 | P4 / P10 |
| P5 | 项目成员 | 管理员从工作区成员中添加两人（一个成员、一个访客），把成员改为访客，移出访客。成员点"离开项目"：先等接口成功，再回到项目列表（M1-P4 交接）。唯一的管理员离开：提示 | 添加：`project_members`、`project_user_properties` 各新增两行；改角色；移出：`is_active = false`；离开同 | `POST /api/v0/projects/{id}/members`（批量）、`PATCH`/`DELETE /api/v0/project-members/{project_member_id}`、`POST /leave`；添加不是工作区成员的人、把工作区访客加为成员 422；工作区管理员加为管理员以外的角色 422；项目成员改别人的角色 403；项目管理员（不是工作区管理员）改另一位管理员的角色 403 `project.role_too_high`；唯一管理员离开 409 `project.sole_admin`。被移出的项目管理员（工作区成员）`POST /join` 公开项目：恢复原来那一行，角色是 15（原来那一行的 20 以他现在的工作区角色 15 为上限，3.5） | P5 / P10 |
| P6 | 状态 | 管理员在状态设置里新建"Review"（进行中组），改颜色，拖到"In Progress"之前，设为默认；删除"In Progress"（进行中组还有"Review"，成功）。"Review"（默认）和"Backlog"（积压组唯一的状态）的删除按钮不可用，与 Plane 的页面相同 | `states` 对应列；新建的 `sequence` 等于非分诊状态的最大值加 15000（默认的 6 个之后是 70000）；拖动改 `sequence`；设为默认之后恰好一行 `default`；删除：`deleted_at` | `GET`/`POST /api/v0/projects/{id}/states`、`PATCH`/`DELETE /api/v0/states/{id}`、`POST /mark-default`；删除默认状态 409 `project.state_default`；删除一组中唯一的状态、把它改到别的组 409 `project.state_last_in_group`；`group = triage` 422；项目成员修改 403；已归档的项目 `GET /states` 返回空列表；`GET /api/v0/workspaces/{slug}/states` 不含分诊状态和已归档项目的状态 | P7 / P11 |
| P7 | 标签 | 管理员建标签"Bug""Feature"，把"UI"拖进"Bug"之下，改名，调顺序，删除"Bug"（其下的"UI"一起删除）。建"bug"（大小写不同）：提示已存在 | `labels`：`parent_id`、`sort_order`；删除时父子两行的 `deleted_at` 在同一时刻写入 | `GET`/`POST /api/v0/projects/{id}/labels`、`PATCH`/`DELETE /api/v0/labels/{id}`；同名（不区分大小写）409 `project.label_name_taken`；三层、父标签在别的项目、自己做自己的父标签 422（`parent_id`，`not_allowed`） | P7 / P11 |
| P8 | 项目的个人显示设置 | 在项目页头的标签页导航里把"模块"设为默认标签页、隐藏"视图"；在侧边栏把第三个项目拖到最前。刷新后仍生效 | `project_user_properties.preferences`、`sort_order` | `GET`/`PATCH /api/v0/me/projects/{id}/preferences`；未知的标签页 422 | P4 / P10 |
| P9 | 访客看别人的个人主页（决策点 3） | 负责人裁定 A：访客打开别的成员的个人主页，看到用户卡片，工作项区域显示"没有权限"，不请求工作项列表 | 数据库不变 | 卡片的数据来自成员列表（W11 的接口版本已覆盖）；工作项列表的接口在 M4，由 M4 按矩阵的这一格测试（13.2） | — / P11 |

**冒烟故事的更新**：
- S1：`goose_db_version` 的最新版本等于最后一个迁移文件（照旧，M3 加了九个）。
- S2：登录之后落到工作区首页、打开一个项目的设置时，没有失败的请求，没有发往旧接口的请求，没有 CSP 违规（3.1）。它在前端的第一个 Phase（P8）改写。

---

## 3. 设计裁定

以下裁定都在总体设计和 M2 设计定下的范围内；改写已批准规则的地方列在 3.20，按 Phase 同步到上级文档。与 Plane 不同的地方逐条登记到差异清单（第 4 节、4.11）。3.3、3.4、3.6 是后续各 M 照做的约定。

### 3.1 挂载时的规则延伸到 M3 能到达的页面（M2 交接第 2 节）
**规则：M3 能到达的页面，挂载时不向 M4 及以后的旧接口发任何请求。** 这是 M2 设计 3.1 的同一条规则，范围扩大到 M3 的页面。

- **M3 能到达的页面**：M2 的页面（登录、注册、`/onboarding`、`/create-workspace`、`/settings/profile/*`），加上 `/workspace-invitations`、工作区首页 `/:workspaceSlug`、项目列表 `/:workspaceSlug/projects` 和已归档的项目、工作区设置的 general 和 members、项目设置的全部标签页（general、members、features 下四项、states、labels、automations），以及访客看到的个人主页（决策点 3，卡片之外不挂载工作项列表）。
- **不在其内**（属于各自的 M）：项目的工作项、迭代、模块、视图、收集箱页面（M4、M6、M7），工作区视图、草稿、通知、浏览（M4、M7），成员和管理员看到的个人主页的工作项列表（M4），Webhook 设置（M8）。M3 页面上指向它们的链接是已知的过渡：点进去之后那一页的旧接口得到 404，直到它的 M（与 M2 设计 3.1"提交 M3 的表单得到 404"同理）。
- **删掉的挂载时请求**（第 1 节的清单；各自的 M 随新接口加回，写进 13.2）：
  - `WorkspaceAuthWrapper` 取收藏（M7）：删掉这一个 SWR。侧边栏的收藏区在 M7 之前为空。
  - `TopNavigationRoot` 取未读通知数（M7）：删掉这一个 SWR。通知按钮没有数字。
  - 首页的"最近"小部件（M7）：从 `HomeBody` 移除，它的组件文件随之删除（`core/components/home/widgets/recents/` 四个文件 381 行，以及只为它存在的空状态和骨架），否则 knip 报未使用。M7 从 Plane 的源码加回。首页在 M7 之前只有问候和"还没有项目"的引导。
  - `ProjectAuthWrapper` 取迭代、两次模块（M6）和视图、分诊状态（M7）：删掉这几个 SWR。
  - 项目侧边栏读 `project.intake_count`（M7）：M3 的项目资源不定义它（3.2 的规则 3），读取删掉。
- **M3 自己的挂载时取数一次改完**：两个包装层在每一页挂载时取项目角色、项目列表、工作区的状态、项目的显示设置、标签、成员和状态（第 1 节）。它们的 store 全部在前端的第一个 Phase（P8，数据层）改接新接口，页面的 Phase 在它之后。这样任何页面的故事加入时，挂载路径上已经没有旧接口的请求（第 12 节）。
- **理由**：与 M2 相同。故事的 `watchPage` 断言"没有发往旧接口的请求、没有失败的请求、控制台安静"不需要例外；挂载时注定失败的请求会产生 404 提示和控制台错误，把真正的缺陷淹没。每个后续 M 随自己的接口加回自己的取数，改动落在拥有它的 M。
- **代价**：M7 之前侧边栏没有收藏、通知没有数字、首页没有"最近"。
- **守卫**：第 2 节每个有页面的故事都经 `watchPage` 断言；收尾按页面逐个列出守着它的故事（M2 收尾的做法）。README"前端"一节"M2 中看到的页面"一条改写为 M3 能到达的页面（M2 交接第 2 节）。

### 3.2 后续 M 的字段何时进入接口（照 M2 设计 3.2）
| 字段 | 规则 | 处理 |
|---|---|---|
| `Workspace.logo_url` | 2 | 可为 `null`，M3 恒为 `null`。删除 general 页的图标上传弹窗（`core/components/workspace/settings/workspace-details.tsx:153-190` 一带）；显示图标的地方不改（没有图标时显示首字母）。M5 加 `workspaces.logo_asset_id`，迁移归 `workspace` 模块（`<v>_workspace_workspaces_logo_asset.sql`，M2 设计 3.14） |
| `Project.cover_image_url` | 2 | 同上，恒为 `null`。删除新建项目时上传预设封面的一步（`core/components/projects/create/root.tsx:65-101`）和默认的随机封面（`projects/create/utils.ts:12`），删除项目设置里的封面选择器；显示封面的地方用默认图。M5 加 `projects.cover_image_asset_id`（`<v>_project_projects_cover_image_asset.sql`），并处理它自己的交接"新建项目时的封面值"（M5 的 M1-closeout） |
| `MemberUser.avatar_url`（成员里的用户资料，5.2） | 2 | 可为 `null`，恒为 `null`。生成的 `MemberUser` 取代 `IUserLite`，`avatar_url` 本来就可空（M2 交接第 7 节） |
| `Project.logo_props` | 1 | 真实数据（表情或图标的 JSON），M3 读写（3.19） |
| 工作项的计数：项目的 `intake_count`（M7）、`total_issues` 一类，工作区成员的 `draft_issue_count`（M4） | 3 | 不定义，删掉前端的读取，由产生它的 M 加回 |
| `Project.is_favorite`（M7 的收藏） | 3 | 不定义。项目卡片上的收藏按钮、建项目之后"加入收藏"的一步（`core/components/projects/create/root.tsx:53`）、`project.store.ts` 的 `addProjectToFavorites`、`removeProjectFromFavorites` 和 `favoriteProjectIds` getter（`:228-245`，没有读者，成为死代码），以及 `favorite.store.ts:284-287` 移除收藏时把 `projectMap[…].is_favorite` 置为假的项目分支（`Project` 没有这个字段，不删就编译不过），在 P8 删除，M7 随收藏的接口加回（13.2）。旧 `ProjectService` 中三个收藏方法本来就没有调用方，一并删除 |

### 3.3 模块划分与跨模块的协作
- **三个模块**，照总体设计 6.2：
  - `workspace`：工作区、成员、邀请、工作区的个人显示设置。
  - `project`：项目、项目成员、状态、标签、项目的个人显示设置。
  - `access`：权限规则表，实现 `shared.Authorizer`（3.4）。它**没有表**。
- **`access` 经端口读成员关系**（M2 交接第 8 节第 1 件）：`access/app` 声明两个小接口，`workspace` 和 `project` 模块各实现一个，`bootstrap` 接上。不在 `TestSQLCSchemaScope` 里开例外。
  - 理由：总体设计 6.3"跨模块的读取在 `app` 层声明端口"；例外会让 `access` 依赖别的模块的表结构，守卫也变弱。
  - 代价：两个接口、每次判定两次按索引的查找（与直接查表的次数相同）。
- **跨模块只经端口**（总体设计 6.3 第 2 条），端口声明在调用方的 `app` 层：
  - `project` 问 `workspace`：按 slug 取工作区（目录），锁住若干人的工作区成员行并取回有效角色（约定三，3.6）。
  - `workspace` 让 `project` 做连带：端口 `ProjectCascade`，由 `project/app` 的用例实现，三个方法：
    - `DeleteWorkspaceProjects(ctx, workspaceID, by, now) error`：软删除这个工作区的项目及其下的行（项目 → 项目成员 → 项目显示设置 → 状态 → 标签，3.6 的取锁表），删除者 `by` 与时刻 `now` 由工作区的连带传入，与工作区一侧的每一步相同；
    - `EndMemberships(ctx, workspaceIDs, userID, now) error`：在调用时列举他在这些工作区里有效的项目成员关系（不接收事先列好的项目，3.6 约定六），锁住那些项目、查唯一管理员（可能拒绝 `project.sole_admin`，3.7）、停用他的项目成员关系。移出、离开、停用账户都用它；
    - `DemoteToGuest(ctx, workspaceID, userID, now)`：把他在这个工作区的项目角色都改为访客（Plane `views/workspace/member.py:87-89`，含已离开的项目）。
  - `workspace` 问 `project`：他在这个工作区还有几个无效的项目成员关系（`ProjectMembershipCounts`，只读，`reactivate-member` 的提示）。
  - `workspace` 问 `identity`：按 id 或邮箱锁住账户行、取回它的状态（`Accounts`：是否存在、是否有效、邮箱），由用例决定怎样处理停用的账户；按 id 批量读公开资料，不加锁，停用的账户也返回（`MemberProfiles`：成员列表，以及事务中途要读的邮箱，3.6 约定一）。
  - 全部端口列在 6.5。
- **M3 不引入领域事件**：
  - M2 设计 3.15 和 M4 的交接（M2-closeout 第 6 节）把"事件订阅者怎样写"交给第一个需要它的 M，也就是 M4；总体设计 6.3 第 3 条说典型的订阅者只做一件事：往 River 里投递任务。
  - M3 的三种连带都只有一个接收方、同步、直接写表，其中一种还要能拒绝。它们正是端口的形状；照它们设计事件的形状，M4、M7、M8 会继承一个按反例定下的形状。
  - M4 加入第一个异步订阅者时引入事件，届时可以把删除工作区的连带改挂到一个 `WorkspaceDeleted` 事件上（13.2）。

### 3.4 权限框架：`Authorizer` 与规则表（M2 交接第 8 节第 3 件；总体设计 6.5）
- **`shared` 中的值和端口**（`shared/authorize.go`，只依赖标准库，规则 10 照旧）：
  - `Role`：`RoleGuest = 5`、`RoleMember = 15`、`RoleAdmin = 20`（Plane `db/models/workspace.py` 的 `ROLE_CHOICES`）。
  - `Action`：操作名的类型（`type Action string`）。操作名常量不在 `shared`，见下。
  - `Target{WorkspaceID, ProjectID}`：工作区级的操作 `ProjectID` 为零值。
  - `Grant{WorkspaceRole, ProjectRole Role; ProjectAdmin bool}`：判定时读到的角色，用例拿它做相对的检查（3.5、3.7），不再查一次。
  - `Authorizer.Authorize(ctx, actor Actor, action Action, t Target) (Grant, error)`。
  - 两种拒绝：`shared.ErrNotVisible`（`KindNotFound`，没有码；用例用 `errors.Is` 认出它，换成自己资源的 404 码，例如 `project.state_not_found`：只有用例知道调用方点名的是什么），和平台码 `forbidden`（403，`shared.CodeForbidden`，看得到但不能做）。`access` 没有操作、没有接口描述文件，它的拒绝跨越所有模块，所以是平台码（11.7）。
- **操作名在各模块声明**：`workspace/domain/actions.go`、`project/domain/actions.go` 声明本模块的 `shared.Action` 常量和 `Actions()`；`access/domain/rules.go` 的规则表以操作名为键。`bootstrap` 的一个测试核对：各模块 `Actions()` 的并集恰好等于规则表的键。
  - 为什么：`access` 不能导入别的模块（架构测试），规则表只能以名字为键；组合根看得到全部模块，完整性在那里证明。加一个操作只改本模块和规则表两处，`shared` 不积累各模块的词汇。没有导入环，架构测试和 depguard 的规则都不变。
  - 放在 `shared` 的另一种做法：每加一个操作都要改 `shared`，还是要改规则表，多一处。
- **判定**（`access/domain` 的纯函数，输入是端口读到的事实）：
  - 工作区级：调用方在这个工作区有**有效**成员关系（`is_active`、未删除，工作区未删除）且角色在规则允许的集合里 → 允许；有效成员而角色不允许 → 403；不是有效成员 → 看不到。
  - 项目级：先要是这个工作区的有效成员，否则看不到。然后：
    - **看得到**这个项目：工作区管理员；或项目的有效成员；或项目公开（`network = 2`）且工作区角色不低于成员。其余看不到：访客看不到自己没加入的项目，公开的也一样（Plane 的列表对访客同样隐藏，`views/project/base.py:198-222`）；成员看不到自己没加入的私密项目。
    - **能做**：是项目的有效成员，并且项目角色在允许的集合里，或者他是工作区管理员（总体设计 6.5："项目成员（角色不限），同时是工作区管理员"；Plane `permissions/base.py:19-88` 的 `allow_permission`）。看得到而不是项目成员 → 403（例如工作区管理员要先加入项目）。
  - 已删除的项目、已删除的工作区都看不到；已归档的项目看得到（3.19）。
- **判定的时机**：写操作在事务里、**取得父行的锁之后**判定（3.6 约定二）；读操作不开事务，直接判定。
- **规则表**（`access/domain/rules.go`）：每个操作名一行：级别和允许的角色。M3 的全部行就是 9.2 矩阵的行。表里没有的操作一律拒绝。`AllowCreator`（总体设计 6.5 的示例）没有 M3 的使用者，由 M4 随第一条用到它的规则加入（13.2）。
- **相对的规则在用例里**：改自己的角色、移出自己、目标的角色更高、访客的上限、唯一管理员。它们要比较两个人，用 `Grant` 和用例读到的目标成员判断，码在 5.3。
- **按可见性过滤的列表**（项目列表）：用例先对工作区级的操作判定，再按同一条可见性规则在 SQL 中过滤。一个集成测试对 9.2 的每种身份核对"列表的结果"等于"对每个项目逐个判定 `project.read` 的结果"，两处规则不会走散。工作区的状态列表不按可见性，只含调用者是有效成员的未归档项目的状态（Plane `views/workspace/state.py:20-26`），它等于"对每个项目判定 `state.list`"，同一个测试一并核对。
- **每个请求都读**，不缓存：移出成员、改角色立即生效（总体设计 4.2）。
- **Plane 里服务端比页面宽松的地方**，按页面收紧，逐条登记差异清单第四节：

  | 操作 | Plane 的服务端 | Plane 的页面 | Nerve |
  |---|---|---|---|
  | 归档、恢复项目 | 项目管理员和成员（`views/project/base.py:428,436`） | 只有项目管理员看得到归档和恢复（`settings/projects/[projectId]/page.tsx:30,48`；`core/components/project/card.tsx:166`） | 项目管理员；总体设计 5.5 本来就写"项目由项目管理员归档" |
  | 修改状态 | 项目管理员、成员、访客（`views/state/base.py:61`）；新建、删除、设为默认都只许项目管理员（`:46,105,113`） | 只有项目管理员可编辑（`core/components/project-states/root.tsx:38-43`） | 项目管理员（Plane 的修改一条是缺陷，总体设计 5.1 第 4 类） |
  | 修改、删除项目 | 工作区管理员（即使不是项目成员）或项目管理员（`views/project/base.py:314-336,382-397`） | 项目设置只对项目成员开放（`ProjectAuthWrapper`） | 项目级规则（总体设计 6.5 的原文）：项目管理员，或同时是工作区管理员的项目成员；不是成员的工作区管理员先加入（一次 `join`） |
  | 列出标签 | 任何工作区成员都得到 200：`ProjectBasePermission` 对读取只看工作区成员，查询再按项目成员过滤、不看 `is_active`，不是成员得到空列表，已离开的成员仍能列出（`views/issue/label.py:26-40`、`permissions/project.py:18-22`） | 只在项目内取 | 与列出状态相同：项目的有效成员（`views/state/base.py:77`），其余按可见性 403 或 404 |
  | 项目负责人、默认负责人 | 任何账户 id（只有外键） | 创建时从工作区成员中选；设置页从项目中不是访客的成员中选（`core/components/project/member-select.tsx:32-42`） | 创建：工作区的有效管理员或成员；修改：项目中不是访客的有效成员（3.19） |

- **与 Plane 相同、不是差异的**：成员列表里的邮箱。Plane 给访客的成员列表用不带邮箱的 `UserLiteSerializer`，给管理员和成员用带邮箱的 `UserAdminLiteSerializer`（`views/workspace/member.py:50-54`；`serializers/workspace.py:93-94,109-110`）；Nerve 相同（访客看到的 `member.email` 为 `null`）。
- **谁能邀请**：负责人裁定只有工作区管理员（决策点 4；Plane 的服务端让管理员和成员邀请，页面只给管理员）。

### 3.5 项目的角色规则（照搬 Plane，码见 5.3）
- **添加项目成员**（`views/project/member.py:46-154`，只许项目管理员，`:46`）：只能从工作区的有效成员中添加；工作区管理员只能以管理员加入，工作区访客只能以访客加入（`:69-83`）；以前离开过的，恢复那一行，角色取这次给出的角色（Plane 相同，`:86-94`）。已是有效成员的，Plane 会顺手改掉他的角色（`:86-94`），Nerve 答 422 `duplicate`（页面只列出还不是成员的人，登记差异）。
- **改角色**（`:205-287`）：只有项目管理员，或同时是工作区管理员的项目成员（Plane 先拒绝不是管理员的人，`:234-238`："You do not have permission to update roles"）。然后，不是工作区管理员时：不能改自己的角色，不能改角色不低于自己的人，不能给出不低于自己的角色。所以项目管理员不能提拔别人为管理员、不能改另一位管理员，这两件只有工作区管理员能做。目标是工作区访客时，项目角色不能高于访客。
- **移出**（`:290-321`，只许项目管理员）：不能移出自己（请用"离开"）；不能移出项目角色比自己高的人（这里 Plane 不给工作区管理员例外，照搬）。
- **加入项目**（`views/project/invite.py:131-189`）：工作区的管理员和成员可以加入公开项目，私密项目只有工作区管理员能加入；新的成员关系的角色等于他的工作区角色；以前离开过的恢复那一行，角色取**原来那一行的角色与他现在的工作区角色中较低的一个**（`min`）。已是有效成员时什么都不做，返回项目。
  - Plane 恢复时只改 `is_active`，保留旧行的角色；新行取工作区角色（`views/project/invite.py:157-174`）。一位被移出的项目管理员只要还是工作区成员，就能自己加入公开项目、拿回管理员。
  - Nerve 以他现在的工作区角色为上限（3.6 约定六"恢复不比新授予给得更多，也不比原来那一行更多"）：被移出的项目管理员（原来 20）现在是工作区成员（15），回来是 15；被项目管理员降为访客（5，"改角色"一条）的工作区成员离开再加入，回来仍是 5，不能靠离开再加入给自己升级。前一半是 Nerve 加的上限，后一半与 Plane 相同；登记差异。
  - 第三稿取"他现在的工作区角色"，只做到前一半：被降为访客的人离开再加入就回到 15（复核 I1，spike 17）。
  - 工作区访客的项目行都是访客（`DemoteToGuest`，含无效的行，3.8），所以做过工作区访客的人被改回成员之后再加入，不会带回做访客之前的项目角色。
- **离开项目**：唯一的项目管理员不能离开（3.7）。

### 3.6 加锁顺序与并发（M2 交接第 6 节"加锁顺序"）
M2 设计 3.5 的加锁顺序是全局约定。M3 的表接在它后面，另加六条约定，后续 M 的表照做。

- **全局顺序**：`users` → `profiles` → `auth_sessions` → `api_tokens` → `workspaces` → `workspace_member_invites` → `workspace_members` → `workspace_user_properties` → `projects` → `project_members` → `project_user_properties` → `states` → `labels`。同一张表的多行按 `id` 升序取锁（批量语句见约定五）。
  - 按这个顺序取的**行锁**不会互相等成环。这句话只管行锁：同一张表上并发的插入者在唯一索引上等待对方未提交的索引项，不是行锁，不在这个顺序里。它由约定五的后一半（在共享的父行锁之下批量插入时按键的顺序插入）排除（复核 I2，spike 14a）。
- **约定一：锁账户行的事务最先锁它**（M2 照旧）。M3 有三类：
  - 创建邀请按 M2 的凭证锁（`CredentialLock`，`FOR NO KEY UPDATE`）锁住邀请人的账户行、复核他的凭证（3.8）；
  - 让一个账户的工作区成员关系变多的写（接受邀请、创建工作区、`reactivate-member`）以 `FOR SHARE` 锁住那个账户的行，见约定六；
  - 忽略邀请与接受同一个形状：最先以 `FOR SHARE` 锁住调用者的账户行，在锁下读邮箱与邀请比较（3.8）。
  - **`Accounts` 只作事务的第一把锁；事务中途要读账户的邮箱，一律经不加锁的 `MemberProfiles`（或同等的只读端口）**。在工作区的锁之后再锁账户行，会与停用（先账户行 N、后工作区 N）互相等成环：复核的 spike 16 让忽略邀请先持工作区 S、再取账户行 S，死锁。中途读邮箱的地方：创建邀请的"已是有效成员"，移出、离开时删除发给他的邀请（3.8）。
- **约定二：写事务先锁父行，再判定权限。**
  - "父行"是判定所在的那一行：工作区级的操作是工作区行，项目级的操作是项目行。
  - `FOR NO KEY UPDATE` 用于：
    - 成员关系和角色的一切改变：接受邀请、添加、加入、改角色、移出、离开、恢复，以及它们的连带；
    - 维护项目范围不变式的写：状态和标签的全部写入（3.16、3.17）；
    - 修改父行本身的写：改、删工作区；改、删、归档、恢复项目。
  - `FOR SHARE` 用于其余的写：往父行下面加行、改下面的行。
  - 锁住之后才调用 `Authorize`。等锁的一方在对方提交之后读到已提交的角色（spike S1b）。第一稿先判定、后锁：两位管理员互相降级，各自的判定都在对方提交之前读到"管理员"，结果一个管理员都不剩（spike S1a）。
  - 每个写用例（P2–P7）把检查分两处：只看请求的值的校验（名称、颜色、角色的取值、上限，不合规 422）在事务之前，不开事务、不取锁；依赖行的检查（目标的成员关系已结束、是自己的、一组中唯一的状态）在判定之后（6.7）。
  - 按资源寻址的操作（`PATCH /states/{id}` 这一类）：读资源行得到父行 → 锁父行 → 重读资源行，确认仍存在、仍属于这个父行 → 判定。
  - 锁父行的语句带 `deleted_at IS NULL`：等锁之后 Postgres 按最新的版本重新求值条件，父行已被软删除时读到 0 行，用例答 404（第二稿 spike）。
  - 为什么外键不够：外键检查只取父行的 `FOR KEY SHARE`，它不挡软删除的 `UPDATE … SET deleted_at`（spike：一方持 `FOR KEY SHARE`，另一方软删除父行立即完成；持 `FOR SHARE` 时软删除等到超时；软删除持锁时 `FOR SHARE` 同样等待；两个 `FOR SHARE` 互不阻塞）。
  - 不用"判定时以 `FOR SHARE` 读自己的成员行"这种更细的写法：互相降级时双方各持自己那一行的共享锁、又要改对方的行，死锁（评审 I1）。
- **约定三：让某人成为项目成员的写，先锁这些人的工作区成员行。** 建项目（创建者和负责人）、添加项目成员、加入项目，在锁项目行之前以 `FOR SHARE` 锁住这些人的 `workspace_members` 行（`id` 升序），取回是否有效和角色。移出和停用改这一行，与之互斥，"项目成员 ⊆ 工作区成员"因此不依赖别的路径是否锁了工作区行。
  - **锁目标行只为取锁的顺序；目标的 422 在判定之后给出**。全局顺序要求目标行在项目行之前锁，而判定在项目行的锁之后：先按调用者判定（看不到 404，不能做 403），通过之后才看目标是否有效、角色是否合规（422）。否则不是项目成员的人会拿到 422 而不是 403，看不到项目的人拿到 422 而不是 404，422 与 404 之差透露了项目 id 存在（复核 M4）。
- **约定四：M3 的锁之后可以插入引用别的账户的行。** M2 规定"一个事务在锁住后三张表的行以后，不再插入引用另一个账户的行"（外键检查对那一行 `users` 取 `FOR KEY SHARE`，算一次 `users` 的锁）。M3 放宽为：锁住 M3 的表之后，可以插入引用别的账户的行（添加项目成员、以别人为负责人建项目）。接受邀请插入的成员行引用的是事务最先锁住的那个账户，M2 本来就允许，不靠这一条放宽。
  - 为什么不会死锁：持有 `users` 行、并且与 `FOR KEY SHARE` 冲突的，只有改唯一键的 `UPDATE`，也就是 `set-email`（升级为 `FOR UPDATE`，M2 设计 3.5），而它不取 M3 的任何锁；停用持的 `FOR NO KEY UPDATE` 与 `FOR KEY SHARE` 不冲突（spike 同上，M2 设计 3.5 的 spike 也测过）。v0 不物理删除账户。
- **约定五：批量的写。**
  - **一条语句批量改子行时按扫描顺序加锁。** 删除工作区或项目的连带、降为访客的连带、删除父标签连带子标签，都是一条 `UPDATE` 改多行，行锁按扫描顺序而不是 `id` 顺序取。
    - 这是安全的：这些语句都在持有父行的 `FOR NO KEY UPDATE` 之后执行，同一父行下不会有第二条批量语句与之并发。
    - 与之并发的只有单个资源上的写（例如改一个项目、在一个项目里建状态）。它们只锁自己那一行和它下面的行，不会再去等批量语句已经锁住的行，所以只会一方等另一方。
  - **只持父行的共享锁而批量插入时，按唯一键的顺序插入。** 两个事务都持父行的 `FOR SHARE`，各自插入一批、键有重叠而顺序相反时，各自等对方未提交的索引项，死锁（复核 spike 14a：两位管理员分别邀请 `[x, y]`、`[y, x]`）。按键排序之后，后到的一方只等先到的一方，先到的提交之后它得到唯一约束的冲突（spike 14b），由用例翻译成字段错误。
    - M3 里只有创建邀请是这种写（按规范化后的邮箱排序，3.8）：添加项目成员持项目的 `FOR NO KEY UPDATE`，同一项目一次只有一个；其余的插入要么也在父行的 `FOR NO KEY UPDATE` 之下，要么在一张有唯一索引的表里只插一行（建工作区、建项目的那一行），要么插在本事务刚建出的父行之下。
    - 为什么排序而不是让创建邀请取工作区的 `FOR NO KEY UPDATE`：那样创建邀请会是约定二里唯一在 `FOR NO KEY UPDATE` 之下"往父行下面加行"的写，分类出现例外，同一工作区的邀请也全部排队；排序只改这一个用例插入的顺序。
    - 以后各 M 在共享的父行锁之下批量插入时照做。
- **约定六：成员关系集合的增长与收缩。** 一个账户的有效成员关系（工作区的、项目的）只经下面两种写改变；后续 M 加入新的成员关系时照做。
  - **收缩**（移出、离开、停用）：先改他的工作区成员行，再在同一个事务里用一条新的语句列举他在这些工作区里有效的项目成员关系，锁住那些项目（`EndMemberships` 在调用时列举，不接收事先列好的项目）。停用另外最先锁住他的账户行（`FOR NO KEY UPDATE`，M2），然后才列举他有效成员关系所在的工作区、按 `id` 升序一次锁住，项目同样一次按 `id` 升序锁住：不按工作区逐个锁项目，全局的 `id` 顺序没有例外。停用很少发生，多锁几行的代价可以忽略。
  - **工作区一侧的增长**（接受邀请，含恢复以前的成员关系；创建工作区；`reactivate-member`）：最先以 `FOR SHARE` 锁住得到成员关系的那个账户的行，在锁下重读 `is_active`（接受邀请还重读邮箱，拿锁下的邮箱与邀请比较），持有到事务结束，然后按全局顺序锁工作区。账户已停用时拒绝：接口答 401 `unauthorized`，命令退出码 1。`reactivate-member` 是唯一的例外，照 Plane 允许停用的账户（3.11）。
  - **项目一侧的增长**（建项目时的创建者和负责人、添加项目成员、加入项目，含恢复以前的项目成员行）：约定三，以 `FOR SHARE` 锁住他的工作区成员行、在锁下确认有效；不锁账户行。
  - **恢复不比新授予给得更多，也不比原来那一行更多**：恢复以前的成员关系时，角色不超过这一次新授予会给的，也不超过原来那一行的。
    - 他自己发起的恢复（加入项目）：`min(原来那一行的角色, 他现在的工作区角色)`（3.5）。
    - 管理员这一次授予的恢复，由授予的人决定角色：接受邀请取邀请的角色，添加取请求里的角色。恢复为工作区访客时同时 `DemoteToGuest`：它让他的项目行都是访客，含无效的行；加入按原来那一行封顶，所以这一步是承重的：没有它，做过访客的人被改回成员之后再加入，会带回做访客之前的项目角色。
    - 只有服务器管理员的 `reactivate-member` 保留原来的角色（照 Plane）。
    - **邀请从不改变有效的成员关系；结束的成员关系不留下邀请**（3.8）。
  - **为什么这样够**：
    - 停用在锁住账户行之后才列举工作区；工作区一侧的增长也要这一行，`FOR SHARE` 与停用的 `FOR NO KEY UPDATE` 冲突。增长先提交，停用拿到锁之后的列举就含有新的工作区；停用先提交，增长在锁下读到账户已停用而拒绝。列举之后，他的工作区集合不会再变大。第二稿的接受邀请不锁账户行，停用列举之后集合还能变大：Codex 的 S1 让新工作区漏掉了停用的锁、唯一管理员的检查和项目的连带，结果工作区没有管理员、一个项目里留着他有效的成员关系。
    - 项目一侧的增长要求他的工作区成员关系有效，并锁住那一行；收缩先改这一行、后列举项目。增长先提交，后面的列举看得到新的项目成员关系；收缩先改了这一行，增长等锁之后读到无效而拒绝（添加 422，建项目的负责人 422，加入 404）。
    - 项目一侧不锁账户行：工作区成员行已经把它与一切收缩串起来；而且添加成员、指定负责人是别人发起的，锁别人的账户行会让管理操作与那个人自己的凭证操作排队。
    - 不在锁住成员关系或项目之后再补锁新的工作区：那会违反全局顺序。
  - **账户行上只重读 `is_active` 和邮箱，不复核凭证**：接受邀请不签发凭证，它给的是账户的成员关系；M2 的 `CallerLock` 复核调用者的会话或 PAT，用在以调用者的凭证发出新东西的写上（创建邀请，3.8）。所以这一步不叫凭证复核。重置密码之后被撤销的凭证在下一个请求就被拒绝（M2）。
  - 交错测试 7、8、12–17 逐条核对这些路径（9.3）。
- **各操作的取锁**（S 是 `FOR SHARE`，N 是 `FOR NO KEY UPDATE`，"判定"是调用 `Authorize`）：

  | 操作 | 依次 |
  |---|---|
  | 创建工作区（接口、命令） | 管理员的账户行 S（锁下确认有效，约定六）→ 插入工作区 → 插入成员 |
  | 修改工作区 | 工作区 N → 判定 → 改 |
  | 删除工作区 | 工作区 N → 判定 → 软删除工作区 → 邀请 → 成员 → 显示设置 → `DeleteWorkspaceProjects`：项目 → 项目成员 → 项目显示设置 → 状态 → 标签（批量，约定五） |
  | 改工作区成员的角色 | 读成员行（得到工作区）→ 工作区 N → 重读成员行 → 判定 → 目标的检查（已结束 404、自己 409）→ 改；改为访客时 `DemoteToGuest`：他在这个工作区的项目（N，`id` 升序）→ 项目成员（批量） |
  | 移出成员、离开工作区 | 读成员行（移出按 id 指定成员关系；离开按 slug，没有这一步）→ 工作区 N → 重读成员行（移出）→ 判定 → 查唯一管理员（离开）→ 软删除这个工作区里发给他邮箱的待接受邀请（邮箱经 `MemberProfiles`，不加锁，约定一；3.8）→ 成员行 → `EndMemberships`：列举他在这个工作区有效的项目成员关系（新的语句，约定六）→ 那些项目（N，`id` 升序）→ 查唯一管理员 → 项目成员 |
  | `nerve workspaces reactivate-member` | 账户行 S（取回状态，停用的也允许，3.11）→ 工作区 N → 成员行（恢复，角色不变）→ 数他无效的项目成员关系（不加锁） |
  | 修改工作区的显示设置 | 工作区 S → 判定 → 插入或更新 |
  | 创建邀请 | 邀请人的账户行（N，复核凭证，3.8）→ 工作区 S → 判定 → 校验（"已是有效成员"：有效成员的 id 经 `MemberProfiles` 换成邮箱比较，不加锁，约定一）→ 按规范化后的邮箱排序插入邀请（约定五；唯一索引的冲突翻译为 422 `duplicate`，3.8） |
  | 修改、删除邀请 | 读邀请 → 工作区 S → 重读邀请 → 判定 → 改 |
  | 接受邀请 | 读邀请（不加锁，得到工作区）→ 接收账户的行 S（锁下重读 `is_active`、邮箱，约定六）→ 工作区 N → 邀请行 `FOR UPDATE`（确认未回应、未删除，核对令牌和锁下的邮箱）→ 已是有效成员：成员关系不变；否则插入或恢复成员（恢复为访客时 `DemoteToGuest`：他在这个工作区的项目 N → 项目成员）→ 记接受、软删除邀请 |
  | 忽略邀请 | 读邀请（不加锁，得到工作区）→ 调用者的账户行 S（锁下读 `is_active`、邮箱，约定一）→ 工作区 S → 邀请行 `FOR UPDATE`（确认未回应、未删除，核对令牌和锁下的邮箱）→ 记回应 |
  | 停用账户（3.9） | `users` N（锁下读邮箱）→ `profiles` → `auth_sessions`（M2 的部分）→ 列举他有效成员关系所在的工作区 → 那些工作区（N，`id` 升序；锁时已删除的跳过）→ 查唯一管理员 → 发给他邮箱的全部邀请 → 他的工作区成员行 → `EndMemberships`：列举这些工作区里他有效的项目成员关系（新的语句）→ 那些项目（N，`id` 升序；锁时已删除的跳过）→ 查唯一管理员 → 项目成员 |
  | 创建项目 | 工作区 S → 判定 → 创建者、负责人的工作区成员行（S，`id` 升序）→ 插入项目 → 项目成员 → 项目显示设置 → 6 个状态 |
  | 修改、归档、恢复项目 | 项目 N → 判定 → 改 |
  | 删除项目 | 项目 N → 判定 → 软删除项目 → 项目成员 → 显示设置 → 状态 → 标签（批量） |
  | 添加项目成员、加入项目 | 目标（加入时是调用者自己）的工作区成员行（S，`id` 升序，取回是否有效和角色）→ 项目 N → 判定 → 目标的检查（添加：无效或角色不合规 422，约定三）→ 插入或恢复项目成员（恢复时的角色按约定六）→ 显示设置 |
  | 改项目成员的角色、移出项目成员 | 读项目成员 → 项目 N → 重读 → 判定 → 改 |
  | 离开项目 | 项目 N → 判定 → 查唯一管理员 → 改 |
  | 修改项目的显示设置 | 项目 S → 判定 → 改 |
  | 新建状态 | 项目 N → 判定 → 插入 |
  | 修改、删除状态，设为默认 | 读状态 → 项目 N → 重读 → 判定 → 检查 → 守卫的写（3.17） |
  | 新建、修改、删除标签 | （修改、删除先读标签）→ 项目 N → 重读 → 判定 → 层级检查 → 写（删除连带子标签，批量） |

- **"设为默认"必须两条语句**：spike 中用一条 `UPDATE … SET "default" = (id = $new)` 翻转默认，一个方向成功，反方向因部分唯一索引逐行检查而失败，取决于行的物理顺序。
- **交错测试**（9.3）在真实数据库上确定性地覆盖 19 种交错：
  1. 两位管理员同时离开；
  2. 两位管理员互相降级；
  3. 接受邀请与删除工作区；
  4. 添加项目成员与移出工作区成员；
  5. 移出成员与他正在建项目；
  6. 移出管理员与他正在删除工作区；
  7. 停用与接受邀请（接受的是他原来不在的工作区，含 Codex S1 的中间两步；另跑一次他在那里有以前的成员行，接受恢复它）；
  8. 停用与创建工作区；
  9. 创建邀请与管理员重置密码；
  10. 设为默认与删除状态；
  11. 两次改父标签；
  12. 接受邀请与改邮箱；
  13. 停用与把他加为项目成员（另跑一次他在那里有以前的项目成员行）；
  14. 停用与他加入项目（同上）；
  15. 停用与以他为负责人建项目；
  16. 停用与恢复他的成员关系；
  17. 降为访客与他加入项目、与把他加为项目成员；
  18. 两位管理员创建重叠的两批邀请，以及单个邮箱的竞争；
  19. 忽略邀请与停用。
  - 7、8、13–16 核对约定六的每一条增长路径与停用，其中 7、13、14 的插入新行和恢复以前的行各跑一次；17 核对降为访客的连带与项目一侧的增长；12、19 核对约定一；18 核对约定五的批量插入。凡有两种顺序的（谁先拿到锁），两种都跑。第二稿对 7 的论证（"停用先软删除邀请、后改成员行，改成员行的语句看得到新的成员关系"）只证明了停用会停用新的成员行，补不回漏掉的工作区锁、唯一管理员的检查和项目的连带，已由约定六取代。

### 3.7 唯一的管理员
M2 决策点 3 要求停用"按 Plane 的本意"拒绝唯一的管理员。Plane 在四处检查唯一管理员，写法各不相同，三处有缺陷。Nerve 只用两条规则：

1. **直接离开**：工作区或项目唯一的有效管理员不能离开，哪怕那里只有他一个人（Plane `views/workspace/member.py:164-174`、`views/project/member.py:332-346`，照搬）。409 `workspace.sole_admin`、`project.sole_admin`。
2. **连带结束**：移出工作区成员、离开工作区时，他在这个工作区的项目成员关系随之结束；停用账户时，他全部的成员关系结束。只要他是某个工作区或项目唯一的有效管理员、**而那里还有别的有效成员**，就拒绝；那里只有他一人时允许（Plane 停用的本意，`views/user/base.py:265-305`）。
   - Plane 离开工作区时查的是"只有他一个成员、而他是管理员的项目"（`views/workspace/member.py:176-195`），与它的提示语相反。
   - 移出成员的检查（`:122-135`）有两处错：它拿工作区成员的 `id` 去比项目成员的 `member_id`，永远不会命中；即使命中，它查的也是"只有一个成员"的项目（`total_members=1`），与离开的检查同样反了。
   - 停用的查询只数这个人自己的行，从不拒绝（M2 决策点 3）。
   - 三处都改为规则 2，登记差异。
- **改角色不需要检查**：按 3.6 约定二，改工作区成员角色的人在取得工作区锁之后仍被判定为管理员，而且不能改自己的角色（`workspace.own_membership`）。所以改完之后至少还有他这一位管理员（spike S1b：互相降级时后到的一方被拒绝）。项目里，同时是工作区管理员的项目成员可以把唯一的项目管理员改成成员，这时项目由工作区管理员管理，与 Plane 相同。
- **接受邀请不会减少管理员**：邀请从不改变有效的成员关系（3.8）；恢复以前的成员关系时他原来不是有效成员，不在管理员的人数里。第二稿让接受覆盖有效成员的角色，Codex 的 S2 由此绕过了本节的保证（恢复成员 → 另一位管理员离开 → 他接受旧的访客邀请：工作区没有管理员，访客却是项目管理员）。
- **只有一人的工作区在他停用之后**没有有效成员。恢复的办法与 Plane 相同：`nerve users activate` 恢复账户，`nerve workspaces reactivate-member` 恢复成员关系（3.11）。

### 3.8 邀请：令牌、接受与注册（M2 交接第 1 节；决策点 1、2、4 已裁定）
- **只有链接一条路**（决策点 2，A）：管理员在成员页邀请若干邮箱、各带角色；邀请列表的每一行有"复制链接"。v0 不发邮件，管理员自己把链接交给对方。"系统内接受"（`/invitations` 页、新手引导的"加入工作区"一步、按邮箱批量接受）删除（7.8）。
- **谁能邀请**（决策点 4，A）：只有工作区管理员能列出、创建、修改、删除邀请（Plane 的页面）。
  - 邀请的角色不能高于邀请人的角色。Plane 创建时检查（`views/workspace/invite.py:61-66`），修改时不检查（`urls/workspace.py:72` 继承的 `partial_update`），是缺陷。只有管理员邀请，这条自然成立，不需要单独的检查和码。
  - 以后若让成员邀请，先要定邀请的读取范围（决策点 4 的记录）。
- **令牌**：`nrv_inv_` 加 22 个 base64url 字符，是 `HMAC-SHA256(K, "workspace-invitation" ‖ 邀请 id)` 的前 16 字节；`K = HKDF-SHA256(签名私钥的种子, info = "nerve workspace-invitation mac v1")`，与刷新令牌的 MAC 密钥同法派生（M2 设计 3.4），密钥不离开 `identity/adapter/signing`。
  - **不存令牌**：删除 `token` 列。管理员列出邀请时由服务端重新算出；接受时按常量时间比对。
  - 为什么不存随机令牌的哈希（像 PAT）：Plane 的"复制链接"随时可用（`core/components/workspace/settings/invitations-list-item.tsx:80`，M1 保留的路径），只存哈希就只能在创建时显示一次。为什么不存原文（像 Plane，`views/workspace/invite.py:93-99`）：数据库泄露时待接受的链接一起泄露（总体设计 5.3 对 PAT 的同一个理由）。
  - **同一部署的全部 nerve 进程用同一个密钥文件**：一个进程发的链接在别的进程上才有效（JWT 本来就这样要求）。README 写明。
  - 代价：换签名密钥之后，待接受的链接全部失效，管理员重新复制即可（重新算出的是新链接）；dev、test 没有配置密钥时，重启也使链接失效（与刷新令牌相同）。
  - **让一个泄露的链接失效**：删除这份邀请，重新邀请（新的 id，新的令牌）。
  - `identity` 为此对外提供"按用途派生的 MAC"（6.5），列为架构问题 11.1。
  - 链接：`/workspace-invitations?invitation_id=<id>&token=<令牌>`。Plane 的链接还带 `slug`（`serializers/workspace.py:122`），Nerve 用不到，去掉。
- **签发按账户行锁**（M2 设计 3.5、M2 交接第 1 节）：创建邀请的事务先锁邀请人的账户行、复核他的凭证（`identity` 的 `CredentialLock` 经端口提供，6.5），再插入。与管理员重置密码并发时，重置先提交则创建得到 401，没有新邀请。
  - 已经存在的邀请不随重置撤销：链接属于邀请，不属于邀请人的凭证。账户被盗后的恢复步骤加一条"核对待接受的邀请"（README，8.6）。
- **查看**：公开的 `GET /api/v0/workspace-invitations/{invitation_id}?token=…`。
  - 邀请不存在、已删除、已接受、令牌不对，一律 404 `workspace.invitation_not_found`（Plane 的公开接口不要令牌，`views/workspace/invite.py:227-233`；Nerve 要，登记差异）。
  - 返回工作区的名称、slug、图标，邀请的角色，以及是否已被忽略；**不返回被邀请的邮箱**（决策点 1：拿到链接的人不能从页面上得知该用哪个邮箱注册）。
  - 访问日志只记路径（`server/internal/platform/httpserver/middleware.go:117,165`），不记查询参数里的令牌。
- **接受、忽略**：`POST …/accept`、`POST …/decline`，要登录，请求体带令牌。登录账户的邮箱（规范化后）必须等于邀请的邮箱（Plane `views/workspace/invite.py:161-174`，GHSA-4vj8-p63v-8p24），不等时 403 `workspace.invitation_email_mismatch`，回答只说"这份邀请发给了另一个邮箱"，不含那个邮箱。已回应的邀请 409。
  - **接受最先锁住接收账户的行**（`FOR SHARE`，3.6 约定六）：在锁下重读 `is_active` 和邮箱，已停用答 401；邮箱的比较用锁下读到的邮箱，所以与改邮箱（`FOR UPDATE`）串行。
  - **忽略也最先以 `FOR SHARE` 锁住调用者的账户行**（3.6 约定一），在锁下读 `is_active` 和邮箱，已停用答 401，与邀请比较的是锁下的邮箱。忽略不让成员关系变多，锁账户行是为了两种回应一个形状、邮箱的核对相同，也为了不在工作区的锁之后再锁账户行：那样与停用死锁（复核 spike 16）。
  - **邀请从不改变有效的成员关系**：他已是这个工作区的有效成员时，接受只消费这份邀请（记 `accepted = true`、`responded_at`，软删除），成员关系和角色都不变，回答 200 `Workspace`，`role` 是他不变的角色；页面照常进入工作区。角色只经 `updateWorkspaceMember` 改变，带着它的全部规则（不能改自己、唯一管理员、`DemoteToGuest`）。
    - 为什么：Plane 对已存在的成员行无条件改角色（`views/workspace/invite.py:188-195`），不分有效还是已离开。它的 `reactivate_workspace_member` 只恢复成员关系、不动待接受的邀请，于是"恢复成员 → 另一位管理员离开 → 他接受旧的访客邀请"让工作区没有管理员，访客却仍是项目管理员（Codex S2）。登记差异。
  - **结束的成员关系不留下邀请**：移出成员、离开工作区在同一个事务里软删除这个工作区里发给他邮箱的待接受邀请（未回应、未删除）；停用对他全部的工作区做同样的事，已忽略的也删（3.9）。他要回来，得有一份新的邀请。
    - 这一步在已持有的工作区 N 之下，按全局顺序在改成员行之前（邀请在 `workspace_members` 之前）；他的邮箱经不加锁的 `MemberProfiles` 读（3.6 约定一）。`EndMemberships` 拒绝时它随整个事务回滚。
    - 为什么：`reactivate-member`（3.11）和改邮箱都能留下"有效成员加一份发给他的待接受邀请"；上一条让这份邀请在他有效时无害，可他一旦被移出，旧链接就能让他自己回来（复核 spike 9d：B 以访客恢复，没有管理员的任何新动作）。只在文档里提醒管理员删除邀请，缺口仍在；在成员关系结束的地方删掉，缺口从根上关闭。
    - 已忽略的邀请不删：它不能再被接受（409），留在列表里是管理员看得到的记录。
    - Plane 移出、离开都不动邀请，登记差异。
  - **恢复以前的成员关系**：他在这个工作区有无效的成员行时，接受恢复那一行（`is_active`），角色改为邀请的角色；邀请的角色是访客时，同一个事务里 `DemoteToGuest`（他在这个工作区的项目成员行，含无效的，都改为访客，P4 起）。
    - 为什么可以按邀请定角色：他原来不是有效成员，不在管理员的人数里，任何角色都不会减少管理员；他的项目成员关系仍无效，以后经添加、加入恢复时的角色按 3.6 约定六另定。
    - 访客时的 `DemoteToGuest` 让"工作区访客的项目角色都是访客"对无效的行也成立。它是承重的：加入恢复以前的项目行时角色以原来那一行为上限（3.5），没有这一步，他被改回成员之后加入，会带回做访客之前的高角色。
  - 没有成员行时新建。邀请记 `accepted = true`、`responded_at`，并软删除（Plane 删除它，`:208`）。
  - 忽略：记 `accepted = false`、`responded_at`，不删除；链接之后显示"已忽略"。
  - **已忽略的邀请仍占着这个邮箱**：部分唯一索引下，再邀请这个邮箱得到 422 `duplicate`，直到管理员删除它。Plane 相同（重复的邀请被 `ignore_conflicts` 静默跳过，`:111-114`）。成员页的邀请列表显示"已忽略"，管理员删除之后可以再邀请。
  - 接受不写 `profiles.last_workspace_id`（Plane 写，`:204-206`）：邀请页在接受之后经 `PATCH /api/v0/me/profile` 写入，与 `/create-workspace` 相同（3.14）。
- **注册**（决策点 1，(a)）：`SignupPolicy` 扩展为 `AllowSignup(ctx, email string, invitation *SignupInvitation) (bool, error)`，不另加端口（M2 设计评审 M16）。`auth.signup_enabled` 为真时照旧允许；为假时，只有带着有效邀请、而注册邮箱（规范化后）等于邀请邮箱时允许。
  - `bootstrap` 的实现组合配置开关和 `workspace` 提供的检查（6.5）。
  - `RegisterRequest` 加可选的 `invitation {id, token}`。邀请无效的各种情况都得到 403 `identity.signup_disabled`，与不带邀请相同，不透露是哪一种。
  - 持有链接的人能用注册试邮箱：邮箱对了就注册成功，不对就被拒绝。这一次次尝试受按 IP 的注册限流约束（`ratelimit.register_ip`，每分钟 10 次），决策点 1 接受了这个风险（8.2）。
  - 注册不替用户接受邀请（Plane 也不）：注册页带着 `next_path` 回到邀请页，由用户点"接受"。`identity` 的注册因此不写 M3 的表。
  - Plane 关闭注册时认"这个邮箱有任何一份邀请"（`authentication/adapter/base.py:102-120`），不要令牌，登记差异。
- **邀请的其余规则**：邀请已是有效成员的邮箱 422 `not_allowed`，已有未删除的邀请 422 `duplicate`（Plane 静默忽略，登记差异）；邮箱按注册的规则规范化和校验（3.13）；一次最多 100 个；没有有效期（与 Plane 相同）。
  - **"已是有效成员"的核对**：取这个工作区有效成员的 id，经不加锁的 `MemberProfiles` 换成邮箱，与规范化后的邮箱比较（3.6 约定一）；不经 `Accounts`，它只作事务的第一把锁。
  - **一批全有或全无**：任何一个邮箱不合规，整批都不插入，回答列出每个不合规的 `invitations[i]`。
  - **按规范化后的邮箱排序插入**（3.6 约定五）：两位管理员同时邀请重叠的邮箱时，后到的一方只等先到的一方，不会死锁（复核 spike 14a、14b）。
  - **并发的冲突与事先的校验同一个回答**：先到的一批提交之后，后到的一批插入同一个邮箱时 `workspace_member_invites_workspace_id_email_key` 报 23505；用例把它翻译为 422 `invitations[i].email` 的 `duplicate`，`i` 是这个邮箱在请求里的下标（9.3 的"唯一约束冲突翻译为 409"的例外），整批回滚。

### 3.9 停用账户的端口（M2 交接第 6 节）
- `identity/app` 声明端口 `MembershipDeactivator.DeactivateMemberships(ctx, userID, email, now) error`，由 `workspace` 模块实现，`bootstrap` 接上。`deactivate`（`identity/app/deactivate.go`）在撤销会话之后调用它：自助停用（`Execute`）和 `nerve users deactivate`（`ExecuteByEmail`）都经过这里。它返回错误时整个停用回滚。
- **邮箱取自锁下的账户行**：M2 的 `CredentialLock` 返回的 `LockedAccount` 只有 `{PasswordHash, Active}`（`identity/app/ports.go:95`），P6 给它加上 `Email`；`deactivate` 把锁下读到的邮箱传给 `DeactivateMemberships`（`ExecuteByEmail` 按邮箱锁住那一行，锁下的邮箱就是它）。不用事务之前读到的邮箱：其间提交的改邮箱会让停用删掉发给旧邮箱的邀请、留下发给新邮箱的（复核 M5）。
- 实现按 3.6 的顺序，在 `deactivate` 已经锁住的账户行之下（约定六：列举之后他的工作区集合不会再变大）：
  - 列举他全部有效成员关系所在的工作区，按 `id` 升序锁住，按 3.7 规则 2 检查（`workspace.sole_admin`）；
  - **列举到而上锁时已不存在的工作区、项目跳过**：锁的语句带 `deleted_at IS NULL`，少返回的行就是其间被删除的，那里的成员关系已随删除结束。不答 404：约定二"读到 0 行答 404"只用于调用方点名的父行，`workspace.not_found` 也不是 `deactivateMe` 的码（复核 spike 15）；
  - 软删除发给这个邮箱的**全部**邀请，已忽略的也删（Plane `views/user/base.py:313`）；
  - 停用他的工作区成员关系；
  - 调 `ProjectCascade.EndMemberships`，传入这些工作区：它在调用时列举他在其中有效的项目成员关系（此前提交的添加、加入、建项目都在其中），一次按 `id` 升序锁住那些项目、检查（`project.sole_admin`）、停用项目成员关系。
- `deactivateMe` 的 `x-problem-codes` 加上 `workspace.sole_admin`、`project.sole_admin`（409，前缀规则按 11.7 修订）；命令打印问题的说明，退出码 1。
- **不投递任务**：命令行的组合不需要 River 客户端，M4 交接第 1 节的"只投递的客户端"留在 M4（13.1）。
- **恢复**：`nerve users activate` 只恢复账户（M2），不恢复成员关系；成员关系由 `nerve workspaces reactivate-member` 逐个工作区恢复（3.11）。这与 Plane 相同：它有 `activate_user` 和 `reactivate_workspace_member` 两个管理命令。命令的说明和 README 写明这两步。
- 差异清单第四节"停用账户"一行随之更新。

### 3.10 工作区名：规则与保留名单（M1-P2、M1-P3、M1-P4 交接）
- **slug**：`^[a-z0-9_-]{1,48}$`，只有小写。
  - Plane 的服务端接受大写（`serializers/workspace.py:61-67`，正则在 `:66`），它的页面总是转成小写（`core/components/workspace/create-workspace-form.tsx`）。只差大小写的 `Acme`、`acme` 是两个工作区、两个容易混淆的地址。Nerve 照页面只收小写，登记差异。
  - 在未删除的工作区中唯一（部分唯一索引）。Plane 删除工作区时把 slug 改成 `slug__<时间戳>` 来腾出它（`db/models/workspace.py:156-176`），效果相同，不照搬改名。
  - 建好之后不能改：页面上它是只读的（`workspace-details.tsx:250-270`），Plane 的 `PATCH` 却接受它。登记差异。
- **名称**：1–80 个字符，至少一个字母或数字（Unicode），不含网址（`serializers/workspace.py:48-59`）。
- **保留名单 = 本站用到的顶层路径段，加上一组声明的预留段**：
  - 用到的：应用的顶层静态路由段（`web/apps/web/app/routes/core.ts`）、`web/apps/web/public/` 的顶层目录、服务端自己的顶层路径（`api`、`assets`、`healthz`、`readyz`）。决策点 2 删掉 `/invitations` 之后是 10 个：`create-workspace`、`onboarding`、`settings`、`sign-up`、`workspace-invitations`、`icons`、`api`、`assets`、`healthz`、`readyz`。
  - 预留的：`admin`、`docs`、`help`、`static`。以后最可能加的顶层页面（管理、接口文档、帮助、静态文件）先占住名字，免得加路由时已经有同名的工作区。
  - M1-P4 交接要求"正好是应用的顶层路由段"。服务端的四个也要算上：名为 `api` 的工作区，它的页面 `/api/projects` 会落到接口的路由上。
  - Plane 的产品词（`one`、`business`、`pro`、`license`、`initiatives`、`workflow`、`story`、`disco`、`drive`、`channels` 等，M1-P3 交接）**全部去掉**：它们不是 Nerve 用到的路径，也不是预留段。`sign-in`、`signin`、`login`、`register`、`profile` 也不保留：它们不是路由，名为 `login` 的工作区就在 `/login`（M1-P4 交接说明这些地址本来就匹配 `/:workspaceSlug`）。
- **同一份来源**：名单只有一份，`server/internal/modules/workspace/domain/reserved_slugs.txt`（`embed`），分"应用""服务端""预留"三段。
  - 前端不再有副本：删除 `RESTRICTED_URLS`；创建表单问 `GET /api/v0/workspace-slugs/{slug}`，它回答"可用、已被占用、保留"。
  - web 的 vitest 读这个文件，核对"应用"一段等于 `routes/core.ts` 的顶层静态路由段加上 `public/` 的顶层目录；`bootstrap` 的 Go 测试核对"服务端"一段等于组合根在前端页面之外注册的顶层路径；"预留"一段恰好是上面声明的四个，与另两段不重复。任何一边加了路由而名单没跟上，测试失败。
- **以后加顶层路由时**：
  - 优先放在已保留的段之下（例如 `/settings/…`），或者用一个预留段（把它从"预留"移到"应用"或"服务端"一段）。
  - 两者都不行时，加路由的那次改动同时把新段加进名单，并加一条启动检查：已有同名的未删除工作区时记 ERROR，写明这个工作区的页面被新路由遮住；发布说明告诉管理员把数据迁到新建的工作区（slug 不能改）。

### 3.11 创建工作区与两个管理命令（M2 交接第 5 节）
- **开关**：`workspace.creation_enabled = false` 时，`POST /api/v0/workspaces` 答 403 `workspace.creation_disabled`（Plane `views/workspace/base.py:83-96`）。前端照旧按 `GET /api/v0/instance` 的 `workspace_creation_enabled` 隐藏入口（M2 已做，四处）。
- **`nerve workspaces create --slug <slug> --name <名称> --admin-email <邮箱>`**：
  - 依据：Plane 在关闭创建时，实例管理员仍能在管理后台建工作区，建出的工作区属于他自己（`license/api/views/workspace.py:71-99`，`owner=request.user`）。Nerve 没有实例管理员账户，它的职能由命令行承担（M2 设计 3.16、3.17），所以由 `--admin-email` 指定管理员。这是 Plane 行为的翻译，登记差异（4.11）。
  - 不受开关限制；建出的工作区与接口相同（规则、唯一性、管理员成员）。邮箱按注册的规则规范化；没有这个账户、账户已停用、slug 不可用时退出码 1，输出一句说明，数据库不变。账户行按 3.6 约定六最先以 `FOR SHARE` 锁住，`Accounts` 交回账户的状态，用例在锁下要求有效。
  - 没有这个账户、账户已停用是只给命令行的两个码：`workspace.account_not_found`、`workspace.account_deactivated`。它们不进任何操作的 `x-problem-codes`（5.3 只列接口的码），命令行打印它们的说明，像 M2 的 `identity.account_not_found`（M3/P1 spec 第 3 节第 7 条）。
- **`nerve workspaces reactivate-member --slug <slug> --email <邮箱>`**（照 Plane 的 `reactivate_workspace_member.py:13-90`）：
  - 最先以 `FOR SHARE` 锁住这个账户的行（约定六），取回它的状态；
  - 把这个人在这个工作区无效的成员关系恢复为有效，角色不变；
  - 输出说明他在这个工作区还有几个项目成员关系仍无效（移出和停用都连带结束了它们；个数经 `ProjectMembershipCounts` 读取，6.5）；账户已停用时照样恢复（照 Plane，约定六唯一的例外），另外提示下一步运行 `nerve users activate`：在这两步之间，这个成员关系计入管理员的人数，而他还不能登录。这一状态下项目一侧的增长照常允许（它只看工作区成员行，约定六）：管理员可以把他加为项目成员、指定为负责人，直到 `nerve users activate` 或再次停用；认证拒绝这个账户，这些成员关系在那之前不给账户持有人任何东西；
  - 已是有效成员时输出说明、退出码 0；工作区不存在、账户不存在、从来不是这个工作区的成员时退出码 1，数据库不变。
  - 不需要删除邀请：移出、停用已经删掉了结束之前发给他的待接受邀请（3.8"结束的成员关系不留下邀请"），恢复之后最多还剩一份在移出或停用之后发出的邀请；他有效时它不改变成员关系（3.8），他下一次被移出时它被删除。第三稿说这份邀请"无害"，只在他仍是有效成员时成立：他被移出之后，旧链接能让他自己回来（复核 M3），这由 3.8 的规则在成员关系结束的地方关闭。
  - 不照搬 Plane 的 `create_project_member.py`：恢复出来的工作区管理员可以经接口加入任何项目，加入会恢复原来的项目成员行，角色取原来那一行与他的工作区角色中较低的（3.5）。
- 两个命令的组合与 `nerve users` 相同：连接池、`workspace` 的管理用例、`identity` 提供的 `Accounts`、`project` 提供的 `ProjectMembershipCounts`（6.5）；没有 River 客户端。
- **没有演示数据**：Plane 建工作区之后投递 `workspace_seed`（`views/workspace/base.py:137`），它建一个名为 "Plane" 的机器人账户做管理员，再建演示项目、状态、标签和工作项（`bgtasks/workspace_seed_task.py:505` 起）。机器人账户违背"不区分人和智能体"，演示数据不是保留的功能，不照搬，登记差异。建工作区因此也不投递任务。

### 3.12 集合型的列表不分页，顺序写明
- M3 的列表（工作区、成员、邀请、项目、项目成员、状态、标签、工作区的状态）一次返回全部，响应是 `{"data": [...]}`，没有 `limit`、`cursor`。**这些集合在 v0 中不分页。**
- 理由：页面要的就是整个集合（下拉框、侧边栏、权限表）；集合的大小由管理员的操作决定；Plane 也一次返回全部（`views/workspace/member.py:46-55`、`views/project/base.py:145-223` 等）；分页会让每个 store 都循环取页。以后真要给其中一个分页，那是接口的改动，要同时改调用方（只读 `data` 的调用方会拿到不完整的集合）。
- **每个列表的顺序**（测试和页面都要确定的顺序；同值时按 `id`）：

  | 列表 | 顺序 | 依据 |
  |---|---|---|
  | 工作区 | 名称 | Plane `views/workspace/base.py:73` |
  | 工作区成员、项目成员 | 加入的时间（`created_at`） | Plane 不排序；按加入先后最稳定 |
  | 邀请 | 创建时间，新的在前 | Plane `views/workspace/invite.py:251` |
  | 项目 | 调用者的 `sort_order`（不是成员的排在后面），再按名称 | Plane `views/project/base.py:104` |
  | 状态 | `sequence` | Plane `db/models/state.py:115` |
  | 工作区的状态 | 项目，再按 `sequence` | 同上 |
  | 标签 | `sort_order` | Plane `views/issue/label.py:39` |

- **页大小的规则不动**（M2 交接第 12 节）：M3 没有分页的列表，`PageSize` 留在 `identity/domain`；这一节原样写进 M4 的交接（工作项列表是第二个分页列表，13.2）。游标"不签名、改成另一个合格的位置照样可用"的提醒一并交过去。
- 总体设计 3.4 补一句（3.20），列为架构问题 11.3。

### 3.13 两个模块共用的取值规则移到 `shared`
- M3 需要三条 M2 写在 `identity/domain` 里的规则：
  - 邮箱的规范化和校验（`identity/domain/email.go:17`、`:25`）：邀请的邮箱要与账户的邮箱按同一规则规范化，接受邀请时比较的是两者；
  - 网址检测（`identity/domain/url.go:32`）：工作区名称不能含网址；
  - 时区（`identity/domain/user.go:81`）：工作区和项目的时区与个人时区同一规则。
- 它们移到 `shared`（只用标准库，规则 10 照旧），测试随之移动；`identity` 改为调用 `shared`。这是 M2 对 `PageSize` 定下的"第二个使用者出现时移到 `shared`"的同一做法。
- 这扩大了 `shared` 的内容（从"跨边界的值和端口"到"两个模块必须一致的纯取值规则"），列为架构问题 11.4。

### 3.14 `profiles.last_workspace_id` 不加外键；登录后的落点（M2 交接第 2、4 节）
- **不加外键**：
  1. 它是客户端写入的落点提示（M2 设计 3.2），只有落点规则读它，而落点规则必须和调用者此刻的工作区列表取交集：外键表达不了"仍是有效成员""工作区未软删除"，这个交集才是唯一正确的检查（Plane 同样在读取时判断，`serializers/user.py:98-138`）；
  2. 外键只在物理删除工作区时（M4 的 60 天清理）把它置空，而那时留下的 id 同样无害；
  3. 外键会让 `identity` 翻译一个指向 `workspace` 的约束，并给出"这个 id 是不是一个工作区"的探测手段。
  - 与 Plane 相同，不登记差异。M2 交接第 4 节的关闭条件"写明补或不补和理由"由本节满足。
- **落点**（Plane `serializers/user.py:98-138`）：已完成引导的用户，`last_workspace_id` 仍在 `GET /api/v0/workspaces` 的结果中就去那里；否则去最早创建的工作区（`created_at` 最小）；一个都没有时去 `/create-workspace`。由前端的 `AuthenticationWrapper` 算出。
  - 不重建 Plane 的 `/api/users/me/settings/`：删除 `currentUserSettings`、`IUserSettings` 的工作区部分和设置 store，以及四处 `await fetchCurrentUserSettings()`（M2 交接第 2 节）。
  - 删除工作区不去清别人的 `last_workspace_id`（Plane 清，`views/workspace/base.py:160-172`）：落点规则让它无害，登记差异。

### 3.15 项目负责人、默认负责人改为 `SET NULL`（M2 交接第 9 节）
- Plane 的 `default_assignee`、`project_lead` 是 `CASCADE`（`db/models/project.py:77-90`）：物理删除一个账户，他负责的项目和其下的一切随之删除。改为 `ON DELETE SET NULL`，按总体设计 5.1 第 4 类（明显的缺陷）登记差异。
- v0 从不物理删除账户（M2 设计 4.7），这一改在 v0 中没有运行时的效果，去掉的是一个陷阱。
- `workspaces.owner_id` 同样是 `CASCADE`（`db/models/workspace.py:131`），它与 `created_by_id` 重复、没有代码读它（前端 `git grep "\.owner\b"` 没有工作区的命中），整列删除（4.2）。
- 删除关系图见 4.12。

### 3.16 标签：只有项目的标签，最多两层，名称不分大小写
- **`project_id` 非空**：Plane 的应用只经项目的接口建标签（`views/issue/label.py:43-56`、`:90-110`）；Plane 的应用和 Nerve 的范围里都没有建"工作区级标签"（`project_id` 为空）的路径。所以不保留它：
  - 差异清单二·按表"`labels`：工作区级标签的名称唯一范围改为 `(workspace_id, name)`"一行改为"没有工作区级标签，`project_id` 非空"；
  - 总体设计 5.3"补充约束"中的"标签名在工作区内唯一"改为"标签名在项目内唯一（不分大小写）"（3.20，架构问题 11.5）。
- **名称在项目内唯一，不分大小写**：Plane 的序列化器按 `iexact` 查重（`serializers/issue.py:375-386`），数据库的唯一约束却区分大小写。Nerve 在数据库上建 `(project_id, lower(name))` 的部分唯一索引（spike：`Bug` 之后插入 `bug` 被拒绝），登记差异。
- **层级**：Plane 的服务端不检查父标签；它的页面只允许两层：子标签不能再有子标签，有子标签的标签不能变成子标签（`core/components/labels/label-drag-n-drop-HOC.tsx:110`、`label-utils.ts:47-50,62-72`）。Nerve 在服务端执行：父标签在同一个项目、未删除、自己没有父标签；有子标签的标签不能有父标签；不能以自己为父（数据库另有 `labels_not_own_parent_check`）。登记差异。
- **层级在并发下成立：标签的全部写入先以 `FOR NO KEY UPDATE` 锁住项目行，再判定、检查、写入**（3.6）。
  - 为什么需要：只持项目的 `FOR SHARE` 时，两次改父标签互不阻塞，各自的检查都通过，结果成环（spike S2）。"在 P 下建子标签"与"给 P 设父标签"同理，会出现三层。
  - 为什么锁项目行而不是锁涉及的标签行：层级是整个项目的标签集合的性质，锁归集合的拥有者，与管理员的计数锁工作区行同一个道理。按行锁要为"移动的标签""新的父标签""它的子标签"分别定锁的强度，才能挡住每一种组合，难以核对。标签的写入很少，整个项目排队的代价可以忽略。
  - 交错测试：两次改父标签，后到的一方看到新的层级，得到 422（9.3）。
- 删除父标签时子标签一起软删除（Plane 的 `CASCADE`，软删除照样连带），一条语句，按 3.6 约定五。
- 删除 `description` 列：Plane 的序列化器不读不写它（`serializers/issue.py:361-373`），前端的类型里也没有。

### 3.17 状态
- **默认的 6 个状态**（`db/models/state.py:24-62`）在建项目的事务里建出，含 Triage（总体设计 5.5）。
- **分诊状态只由 `group = 'triage'` 识别**：删除 `is_triage` 列。Plane 的两个标志并不一致：`is_triage` 只由 Plane 的迁移 `0063` 回填过，`State.save()`（`db/models/state.py:117-128`）和建项目时的 `bulk_create` 都不设它，所以之后建出的 Triage 行 `is_triage = false`；而 Plane 的视图按 `is_triage=False` 过滤（`views/state/base.py:39`），默认管理器又按 `group` 排除分诊（`db/models/state.py:65` 起）。
- 分诊状态对 M3 的状态操作不可见（列表不含它，按 id 操作得到 404，与 Plane 的默认管理器相同）；收集箱取它的接口由 M7 加入。
- **每个项目至多一个默认状态、至多一个分诊状态**：部分唯一索引，新加（Plane 在代码里清掉旧的默认）。spike 验证两个索引都挡住第二行。
- **每一组至少一个状态**（分诊组除外）：删除一组中唯一的状态，或者把它改到别的组，409 `project.state_last_in_group`。
  - Plane 的页面这样限制：删除按钮和拖动在组内只剩一个状态时不可用（`core/components/project-states/options/delete.tsx:33`，`totalStates` 是组内的个数，`state-list.tsx:31`；`state-item.tsx:48`）。它的服务端不检查。
  - 按 3.4 的原则，服务端照页面执行，登记差异。
- 新建的 `sequence` = 项目中**非分诊状态**的最大值加 15000（`db/models/state.py:117-128`；它的 `State.objects` 排除分诊，`:65-68`），默认的 6 个状态之后是 70000；`group` 不能是 `triage`（`serializers/state.py:31-34`）；名称在项目内唯一（区分大小写，照搬）。
- 修改只允许项目管理员（3.4 的表）；`group` 不能改成 `triage`。
- 删除默认状态 409 `project.state_default`（Plane `views/state/base.py:113` 起）。"状态下还有工作项时不能删"由 M4 加入（13.2）。
- **锁与守卫的写**：状态的全部写入先以 `FOR NO KEY UPDATE` 锁住项目行（3.6：默认状态、组的非空都是项目范围的不变式）。语句本身另带守卫，不依赖检查与写入之间没有别人：
  - 删除：`UPDATE states SET deleted_at = $now WHERE id = $1 AND NOT "default" AND deleted_at IS NULL`，改了 0 行时重读这一行，按它的状态答 409 `project.state_default` 或 404；
  - 设为默认：第一条语句清掉原来的默认，第二条 `… SET "default" = true WHERE id = $1 AND deleted_at IS NULL`，改了 0 行时让事务失败（404），不留下没有默认状态的项目。
  - 第二稿 spike：设为默认持锁时，带守卫的删除等到它提交之后删除 0 行，恰好一个默认。
- **已归档项目的状态不列出**：`listStates` 对已归档的项目返回空列表，工作区的状态列表也不含它们（Plane `views/state/base.py:37`，照搬）。
- **`order` 不进入接口**：它是状态在所属分组中的位置（Plane 在列表里现算，`views/workspace/state.py:30-38`），前端的状态图标拿它画进度。改由 state store 按 `sortStates` 的顺序算出，5 处读 `state.order` 的组件改读 store（7.3）。
- 删除 `slug` 列：没有代码读它，Plane 自己的默认状态的 `slug` 都是空串（同样因为 `bulk_create`）。

### 3.18 个人显示设置的范围
- **工作区**（`workspace_user_properties`）：`navigation_control_preference`（手风琴或标签页）、`navigation_project_limit`，由 `ProjectNavigationDialog` 读写（M1-P2 交接）。
- **项目**（`project_user_properties`）：`preferences.navigation` 的 `default_tab` 和 `hide_in_more_menu`（`core/components/navigation/use-tab-preferences.ts`），以及侧边栏里项目的顺序 `sort_order`（`core/store/project/project.store.ts:440`）。Plane `preferences` 里的 `pages` 一项随文档页砍掉。
- **不建工作项列表的筛选和显示列**（`filters`、`display_filters`、`display_properties`、`rich_filters`）：差异清单二·按表把它们的列名和格式交给 M3/M4 确定，它们的使用者是 M4 的列表引擎，由 M4 按自己的格式加列，迁移归这两张表的模块（M2 设计 3.14）。
  - M4 的工作项筛选 store（`core/store/issue/project/filter.store.ts:127,174,236,256`）仍调 Plane 的 `/projects/{id}/user-properties/`：`ProjectService` 中这两个旧方法留给它，M4 替换（13.2）。
- **行的生命周期**：
  - 项目的显示设置随项目成员关系一起建（Plane `ProjectMember` 的创建路径）。`sort_order`：建项目、添加成员时是他在这个工作区已有的最小值减 10000，没有时 65535（`views/project/member.py:96-118`）；加入项目时是模型的默认值 65535（Plane 的 `join` 用 `bulk_create`，`views/project/invite.py:175-185`）。都照搬。
  - 工作区的显示设置在第一次修改时建：`PATCH` 经部分唯一索引做 `INSERT … ON CONFLICT (workspace_id, user_id) WHERE deleted_at IS NULL DO UPDATE`（spike：两次写入只有一行，值是后一次的；sqlc 能生成）。没有这一行时 `GET` 返回默认值，不写库：总体设计 3.2 规定 GET 没有副作用，Plane 在 GET 里 `get_or_create`（`views/workspace/user.py:269-277`），登记差异。

### 3.19 项目的读取、归档与字段规则
- **看得到而不是成员**时，`GET /api/v0/projects/{id}` 返回项目，`member_role` 为 `null`。Plane 对这种情况回答 409（公开）或 403（私密）（`views/project/base.py:232-244`），前端据状态码显示"加入项目"的界面。Nerve 的 `ProjectAuthWrapper` 改为按 `member_role` 判断，界面相同。登记差异。
- **工作区访客对没加入的公开项目**得到 404：它不在访客的列表里（Plane 同样隐藏），404 就是"看不到"。Plane 的取单个对访客也回答 409（`views/project/base.py:235-244`），登记差异。
- **已归档的项目看得到**（总体设计 5.5"仍然可以查看"）：`GET` 返回它，带 `archived_at`。Plane 答 404（`views/project/base.py:227`）；前端遇到已归档的项目显示与 Plane 相同的界面。
  - 修改已归档的项目 409 `project.archived`（Plane 400，`:343-347`）；其余写入不因归档而拒绝（Plane 也不）。
  - 列表默认不含已归档的项目，`?archived=true` 只返回已归档的（总体设计 3.3）。Plane 的列表两者都返回，由前端过滤。
- **项目资源带调用者的视角**：`member_role`（他的项目角色）、`sort_order`（他的侧边栏顺序），与 Plane 的列表相同（`views/project/base.py:147-183`）；另带 `member_ids`（有效成员的 id，卡片上的头像用）。
- **负责人、默认负责人**：
  - 创建：负责人是工作区的有效管理员或成员，他随之成为项目管理员（Plane `views/project/base.py:272-278`）；创建时不设默认负责人。
  - 修改：两者都必须是项目中不是访客的有效成员（Plane 设置页的选择范围，`core/components/project/member-select.tsx:32-42`），可以清空（`null`）。修改不会把谁加为成员。
  - 不合规时 422（`project_lead_id`、`default_assignee_id`，`not_allowed`）。Plane 的服务端不检查，登记差异。
  - 页面上它们与"访客可见全部"一起在项目设置的 members 页（`project-settings-member-defaults.tsx`，由 `…/members/page.tsx:50` 渲染），不在 features 页。
- **`logo_props`**：结构是封闭的（5.2），全部字段可选；列的默认值 `{}` 是合法的值，表示没有图标。数据库按 M2 设计 3.13 另有同样的 CHECK：键的集合、出现的每个键的值类型，嵌套的 `emoji`、`icon` 两个对象也查到底（4.6）。不为它编一个默认图标：页面建项目时总会带一个随机的表情（`core/components/projects/create/utils.ts:14-19`），经接口不带图标建出的项目，页面的 `Logo` 组件显示占位（`web/packages/propel/src/emoji-icon-picker/logo.tsx:37`），与 Plane 相同。
- **标识**：转成大写后 1–10 个字符，只能是 `A-Z`、`0-9` 和 `ÇŞĞİÖÜ`，与 Plane 的页面相同（`core/components/project/create/common-attributes.tsx:96-107`、`core/components/project/form.tsx:339`）；Plane 的服务端允许 12 个字符和更多的字符。标识会出现在工作项编号里（M4 的 `GET /workspaces/{slug}/issues/{PROJ-12}`），不能含 `-`。登记差异。列类型仍是 `varchar(12)`。
- **名称**：不能含 `& + , : ; $ ^ } { * = ? @ # | ' < > . ( ) % ! -`（Plane `serializers/project.py:39-44` 的 `FORBIDDEN_IDENTIFIER_CHARS_PATTERN`），数据库另有同样的 CHECK（spike：`Web-2` 被拒绝）。名称和标识在工作区的未删除项目中唯一（照搬）。
- **时区**：不传时取工作区的时区（Plane 的创建路径）。
- **公开发布、anchor、自动关闭、估算、文档页开关**等字段不进入接口（M1-P2、M1-P3 交接），表里也没有（4.6）。Nerve 没有项目动态，所以也不产生 M1-P3 交接列出的几类动态记录。

### 3.20 需要同步到上级文档的地方
**规则**照 M2 设计 3.20：每处偏离，在实现它的那个 Phase 的同一次合并中同步到上级文档和差异清单；建表、改表的 Phase 逐列更新差异清单。收尾只按本表逐行核对。总体设计 9.4 中 M3 的状态随本文的第一稿改为"进行中"（已完成）。负责人已裁定第 10 节、批准第 11 节（2026-09-29），下表每一行的内容都已确定，按标出的 Phase 同步。

| Phase | 文档 | 位置 | 内容 |
|---|---|---|---|
| P1 | M2 设计 | 3.11 第 1 条 | 前缀规则的修订：带前缀的码，前缀必须是 nerve 的一个模块（`api/modules/` 下有它的文件），不必是声明它的那个文件；平台码加 `forbidden`（403）（11.7） |
| P1 | 总体设计 | 3.5 | 平台码的表加 `forbidden` |
| P1 | 总体设计 | 6.2 | `shared` 的内容加上 `Authorizer`、`Role`、`Action` 类型，以及两个模块共用的纯取值规则；操作名常量在各模块（3.4、3.13） |
| P1 | 总体设计 | 6.5 | 规则表的形式、看不到与不能做的判定、相对规则在用例中、判定在父行的锁之后；`AllowCreator` 由 M4 加入（3.4） |
| P1 | 总体设计 | 6.3 | 第 4 条"只在组合根接线"：模块先 `Provide` 后 `New` 的两段组合（6.6、11.6） |
| P1 | 差异清单 | 二·按表 | `workspaces`、`workspace_members` 逐列（4.2、4.3）；`workspaces` 的全表唯一 `workspace_slug_key` 改为部分唯一（二·全局只写了去掉 `(…, deleted_at)` 一类，这一条另写） |
| P1 | 差异清单 | 三、四 | 4.11 中标 P1 的行 |
| P1 | 总体设计 | 3.4 | 集合型的列表在 v0 中不分页，响应仍是 `{data}` 封套（3.12、11.3）。P1 有第一个集合列表（`listWorkspaces`） |
| P2 | 总体设计 | 3.6 | "关联对象只返回 ID"加一个例外：成员资源内嵌用户的公开资料，v0 没有读别的用户的接口（5.2、11.3）。P2 有第一个成员列表 |
| P2 | 总体设计 | 4.2 | 加锁顺序延伸到 M3 的表和六条约定，包括"先锁父行，再判定"、成员关系集合的增长与收缩、`Accounts` 只作事务的第一把锁、共享锁之下的批量插入按唯一键排序（3.6、11.2）。约定六的接受邀请一段在 P3、停用一段在 P6 随实现核对 |
| P2 | 差异清单 | 二·按表、四 | `workspace_user_properties` 逐列（4.5）；4.11 中标 P2 的行 |
| P3 | 总体设计 | 1.1 | 邀请"在系统内接受"改为"凭邀请链接接受，由工作区管理员复制链接交给对方；只有工作区管理员发出邀请"（决策点 2、4） |
| P3 | 总体设计 | 4.2 | "被邀请的邮箱始终可以注册"改为"注册关闭时，持有效邀请链接、且注册邮箱与邀请一致的人仍可注册；公开的查看不显示被邀请的邮箱"（决策点 1） |
| P3 | 差异清单 | 二·按表、四 | `workspace_member_invites` 逐列（4.4）；4.11 中标 P3 的行（邀请的各行，含"接受不改变有效的成员关系"） |
| P4 | 差异清单 | 二·按表、四 | `projects`、`project_members`、`project_user_properties`、`states` 逐列（4.6–4.9），`projects` 的计数列定名为 `last_issue_sequence`；4.11 中标 P4 的行 |
| P5 | 差异清单 | 四 | 4.11 中标 P5 的行（唯一管理员的三处修正、恢复成员的命令、移出和离开删除发给他的待接受邀请） |
| P6 | 差异清单 | 四 | "停用账户"一行（3.9） |
| P7 | 总体设计 | 5.3 | "标签名在工作区内唯一"改为"标签名在项目内唯一（不分大小写）；没有工作区级标签"（3.16） |
| P7 | 差异清单 | 二·按表、四 | `labels` 逐列，原"工作区级标签的名称唯一范围"一行改写（3.16）；4.11 中标 P7 的行 |
| P1、P3、P5、P6 | README | "部署""安全"两节 | 8.7 中各 Phase 的行 |
| P8 | 总体设计 | 7.7 | `RootStore` 有了释放的方法，由 `store-context.tsx` 在换代时调用；`inSession()` 从 `theme-switcher.tsx` 的闭包移到 `core/lib/in-session.ts`，7.7 的引用随之改；加一句"页面按权限决定取数，不只决定显示"（7.1） |
| P8–P11 | 前端改动清单 | 3.1、3.2 | M3 一行的状态；上传控件删除到 M5；`/invitations` 页、`RESTRICTED_URLS`、设置 store 删除；挂载时的取数删除到 M6、M7（3.1） |
| P9 | README | "前端"一节 | "M2 中看到的页面"改写为 M3 能到达的页面（M2 交接第 2 节） |
| 收尾 | 总体设计 | 9.4 | M3 的状态改为"已完成" |

---

## 4. 数据模型

按 M2 设计 3.13 的表结构约定建表：外键的 `ON DELETE` 照搬模型的 `on_delete`，不用 `DEFERRABLE`，默认值和 CHECK 从模型读来、在差异清单中写为"新加"，审计时间由用例的时钟写入，不建 `*_like` 索引，`created_by_id`、`updated_by_id` 不建索引，只用 PG 17 的语法。Plane 的每张表同时有 Django 的 `UNIQUE (…, deleted_at)` 和 `WHERE deleted_at IS NULL` 的部分唯一两套约束；前者在 `deleted_at` 为空时不起作用（空值互不相等），Nerve 只保留部分唯一索引（差异清单二·全局已有这一条）。

M3 对 M2 设计 3.13 的两处补充：
- **索引的名字**：同一列（或同一组列）上有两个部分唯一索引、只靠条件区分时，名字在列名之后加上条件的含义：`<表>_<列>_<含义>_key`，例如 `states_project_id_default_key`、`states_project_id_triage_key`。3.13 的规则只写了一列一个的情形。
- **物理级联用到的外键列有不带条件的索引**：3.13 规定"只给查询或级联真正用到的外键列建索引"。M3 的部分索引都带 `WHERE deleted_at IS NULL`，而 M4 的 60 天清理删除的正是已软删除的行：删除一个父行时，Postgres 按外键在子表上查找 `WHERE <列> = $1`，部分索引用不上，只能顺序扫描。所以指向 `workspaces`、`projects` 和 `labels.parent_id` 的每条 `CASCADE` 外键都有一个不带条件的索引 `<表>_<列>_idx`（下面各表逐个列出）。指向 `users` 的外键不建：v0 不物理删除账户（M2 设计 4.7），这些级联不会触发，查询要用的已有索引。

九张表都经 spike 在 Postgres 18.6 上建出、逐条试过约束，sqlc v1.31.1 能解析（带引号的列名 `"group"`、`"default"`，表达式唯一索引，`ON CONFLICT … WHERE`，`FOR SHARE`、`FOR NO KEY UPDATE`），生成的 `State` 结构有 `Group`、`Default` 字段。

### 4.1 迁移文件
每张表随它的第一个操作所在的 Phase 建出。

| 文件 | 内容 | Phase |
|---|---|---|
| `00006_workspace_workspaces.sql` | `workspaces` | P1 |
| `00007_workspace_workspace_members.sql` | `workspace_members` | P1 |
| `00008_workspace_workspace_user_properties.sql` | `workspace_user_properties` | P2 |
| `00009_workspace_workspace_member_invites.sql` | `workspace_member_invites` | P3 |
| `00010_project_projects.sql` | `projects` | P4 |
| `00011_project_project_members.sql` | `project_members` | P4 |
| `00012_project_project_user_properties.sql` | `project_user_properties` | P4 |
| `00013_project_states.sql` | `states` | P4 |
| `00014_project_labels.sql` | `labels` | P7 |

- `server/sqlc.yaml` 加 `workspace`（P1）、`project`（P4）两个条目，`schema:` 各列本模块的迁移（M2 设计 3.14）。`access` 没有表，没有条目。
- 删除工作区的连带随表加入：P2 连带成员和显示设置，P3 加上邀请，P4 加上项目（经 `ProjectCascade`），P7 加上标签；降为访客的项目连带在 P4 随项目加入。
- 每个建表的 Phase 同时把新表加进 `deploy/runtime-grants.sql`（M2 设计 6.1）。
- **`TestSQLCSchemaScope` 的补充**（M2 交接第 8 节第 2 件，P1）：所有者规则现在只用正则认 `CREATE TABLE` 和 `ALTER TABLE`（`server/internal/archtest/sqlc_test.go`）。M3 是第一个有跨模块外键的 M，补上四种写法，每种一个反例，规则漏掉时测试失败：带引号的标识符（`ALTER TABLE "users"`）；别的模块的表上的 `CREATE [UNIQUE] INDEX … ON users`；`CREATE TRIGGER … ON users`；`DROP TABLE users`。`REFERENCES users` 仍然允许（M2 的 spike：跨模块外键照写，跨模块的读取写不出来）。

### 4.2 `workspaces`（Plane 14 列 → 10 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | 照搬 |
| `name` | `varchar(80) NOT NULL`，`CHECK (name <> '')` | 类型照搬；CHECK 新加。"至少一个字母或数字、不含网址"在领域层 |
| `slug` | `varchar(48) NOT NULL`，`CHECK (slug ~ '^[a-z0-9_-]+$')` | 类型照搬；CHECK 新加（只有小写，3.10）。全表唯一 `workspace_slug_key` 改为部分唯一索引 `workspaces_slug_key ON (slug) WHERE deleted_at IS NULL` |
| `organization_size` | `varchar(20)`，可空，`CHECK (organization_size IN ('Just myself', '2-10', '11-50', '51-200', '201-500', '500+'))` | 类型照搬；CHECK 新加，取值是前端的选项（`web/packages/constants/src/workspace.ts:10`），Plane 的服务端不检查 |
| `timezone` | `varchar(255) NOT NULL DEFAULT 'UTC'` | 类型照搬；默认值新加（模型的 `default="UTC"`）。取值在领域层校验（3.13） |
| `created_by_id`、`updated_by_id` | `uuid REFERENCES users ON DELETE SET NULL` | `ON DELETE` 来自 `UserAuditModel` |
| `created_at`、`updated_at` | `timestamptz NOT NULL DEFAULT now()` | 由应用写入 |
| `deleted_at` | `timestamptz` | 照搬 |

- **删除的列**：`logo`（被 `logo_asset` 取代的旧地址，差异清单已登记）；`logo_asset_id`（M5 加回，3.2）；`owner_id`（与 `created_by_id` 重复、没有读取者、`CASCADE` 的陷阱，3.15）；`background_color`（Plane 建工作区时随机取一个颜色，前端不读）。

### 4.3 `workspace_members`（Plane 17 列 → 10 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | 照搬 |
| `workspace_id` | `uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE` | `ON DELETE` 来自模型 |
| `member_id` | `uuid NOT NULL REFERENCES users ON DELETE CASCADE` | 同上 |
| `role` | `smallint NOT NULL DEFAULT 5`，`CHECK (role IN (5, 15, 20))` | 快照的 `CHECK (role >= 0)` 收紧为三个取值；默认值新加 |
| `is_active` | `boolean NOT NULL DEFAULT true` | 默认值新加。移出、离开、停用都只把它改为假（Plane 相同） |
| `created_by_id`、`updated_by_id`、`created_at`、`updated_at`、`deleted_at` | 同 4.2 | — |

- **索引**：`workspace_members_workspace_id_member_id_key ON (workspace_id, member_id) WHERE deleted_at IS NULL`（部分唯一，照搬）；`workspace_members_member_id_idx ON (member_id) WHERE deleted_at IS NULL`（我的工作区、停用）；`workspace_members_workspace_id_idx ON (workspace_id)`（物理级联）。
- **删除的列**：`view_props`、`default_props`（差异清单已登记）；`issue_props`、`company_role`、`getting_started_checklist`、`tips`、`explored_features`（前端不读不写；公司角色随 M2 删掉的新手引导步骤一起没有了写入方；后三项是 Plane 已砍功能的状态）。

### 4.4 `workspace_member_invites`（Plane 13 列 → 11 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | 照搬；它是链接里的 `invitation_id` |
| `workspace_id` | `uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE` | — |
| `email` | `varchar(255) NOT NULL`，`CHECK (email <> '' AND email = lower(email) AND email !~ '[[:space:]]')` | CHECK 新加：存规范化之后的邮箱（3.13），与 `users.email` 的 CHECK 相同（`server/migrations/sql/00001_identity_users.sql:8`） |
| `role` | `smallint NOT NULL DEFAULT 5`，`CHECK (role IN (5, 15, 20))` | 同 4.3 |
| `accepted` | `boolean NOT NULL DEFAULT false` | 默认值新加 |
| `responded_at` | `timestamptz` | 照搬；`CONSTRAINT workspace_member_invites_responded_check CHECK (responded_at IS NOT NULL OR NOT accepted)` 新加 |
| `created_by_id`、`updated_by_id`、`created_at`、`updated_at`、`deleted_at` | 同 4.2 | — |

- **索引**：`workspace_member_invites_workspace_id_email_key ON (workspace_id, email) WHERE deleted_at IS NULL`（照搬；已忽略的邀请没有删除，仍占着这个邮箱，3.8）；`workspace_member_invites_email_idx ON (email) WHERE deleted_at IS NULL`（注册策略、停用按邮箱查）；`workspace_member_invites_workspace_id_idx ON (workspace_id)`（物理级联）。
- **删除的列**：`token`（不存令牌，3.8）；`message`（没有写入方）。
- spike：`accepted = true` 而 `responded_at` 为空、邮箱含大写，都被 CHECK 拒绝。

### 4.5 `workspace_user_properties`（Plane 14 列 → 10 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | — |
| `workspace_id` | `uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE` | — |
| `user_id` | `uuid NOT NULL REFERENCES users ON DELETE CASCADE` | — |
| `navigation_project_limit` | `integer NOT NULL DEFAULT 10`，`CHECK (navigation_project_limit >= 0)` | 默认值、CHECK 新加（模型 `default=10`） |
| `navigation_control_preference` | `varchar(25) NOT NULL DEFAULT 'ACCORDION'`，`CHECK (… IN ('ACCORDION', 'TABBED'))` | 默认值、CHECK 新加（模型的 choices） |
| `created_by_id`、`updated_by_id`、`created_at`、`updated_at`、`deleted_at` | 同 4.2 | — |

- **索引**：`workspace_user_properties_workspace_id_user_id_key ON (workspace_id, user_id) WHERE deleted_at IS NULL`；`workspace_user_properties_workspace_id_idx ON (workspace_id)`（物理级联）。
- **删除的列**：`filters`、`display_filters`、`display_properties`、`rich_filters`（它们的使用者 M4 按自己的格式加回，3.18；差异清单同）。

### 4.6 `projects`（Plane 36 列 → 23 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | — |
| `workspace_id` | `uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE` | — |
| `name` | `varchar(255) NOT NULL`，`CHECK (name <> '' AND name !~ '[&+,:;$^}{*=?@#\|''<>.()%!-]')` | CHECK 新加（Plane 的禁用字符，3.19） |
| `description` | `text NOT NULL DEFAULT ''` | 默认值新加 |
| `identifier` | `varchar(12) NOT NULL`，`CHECK (identifier ~ '^[A-Z0-9ÇŞĞİÖÜ]{1,10}$')` | CHECK 新加（3.19）；spike：`web`、11 个字符被拒绝，`ÇAY1` 通过 |
| `network` | `smallint NOT NULL DEFAULT 2`，`CHECK (network IN (0, 2))` | 快照的 `>= 0` 收紧；默认值新加（0 私密、2 公开） |
| `project_lead_id` | `uuid REFERENCES users ON DELETE SET NULL` | Plane `CASCADE`，改为 `SET NULL`（3.15） |
| `default_assignee_id` | `uuid REFERENCES users ON DELETE SET NULL` | 同上 |
| `cycle_view`、`module_view`、`issue_views_view`、`intake_view`、`guest_view_all_features` | `boolean NOT NULL DEFAULT false` | 默认值新加（模型） |
| `archive_in` | `integer NOT NULL DEFAULT 0`，`CHECK (archive_in BETWEEN 0 AND 12)` | 默认值、CHECK 新加（模型的验证器 0–12） |
| `logo_props` | `jsonb NOT NULL DEFAULT '{}'`，`CONSTRAINT projects_logo_props_check`，见表下 | 默认值、CHECK 新加。`{}` 表示没有图标；键和值类型与接口的结构相同，字段都可选（3.19、5.2） |
| `timezone` | `varchar(255) NOT NULL DEFAULT 'UTC'` | 默认值新加 |
| `last_issue_sequence` | `integer NOT NULL DEFAULT 0`，`CHECK (last_issue_sequence >= 0)` | **新增**：工作项编号的计数列（总体设计 5.3；差异清单"列名在 M3 建表时确定"）。M4 用 `UPDATE projects SET last_issue_sequence = last_issue_sequence + 1 WHERE id = $1 RETURNING last_issue_sequence` 取号（sqlc spike 能生成）。M3 不读写它，不进入接口 |
| `archived_at` | `timestamptz` | 照搬 |
| `created_by_id`、`updated_by_id`、`created_at`、`updated_at`、`deleted_at` | 同 4.2 | — |

- **`logo_props` 的 CHECK**（M2 设计 3.13 的写法：是对象、键的集合、出现的每个键的值类型，包在 `CASE WHEN jsonb_typeof(…) = 'object'` 里）。每个键都可选，所以类型的条件写成"没有这个键，或者类型对"；嵌套的 `emoji`、`icon` 两个对象同样检查。它们的值都是字符串，到这一层结构就查完了：
  ```sql
  CONSTRAINT projects_logo_props_check CHECK (CASE WHEN jsonb_typeof(logo_props) = 'object' THEN
        (logo_props - array['in_use', 'emoji', 'icon']) = '{}'::jsonb
    AND (NOT logo_props ? 'in_use' OR logo_props -> 'in_use' IN ('"emoji"'::jsonb, '"icon"'::jsonb))
    AND (NOT logo_props ? 'emoji' OR CASE WHEN jsonb_typeof(logo_props -> 'emoji') = 'object' THEN
            ((logo_props -> 'emoji') - array['value', 'url']) = '{}'::jsonb
        AND (NOT (logo_props -> 'emoji') ? 'value' OR jsonb_typeof(logo_props -> 'emoji' -> 'value') = 'string')
        AND (NOT (logo_props -> 'emoji') ? 'url' OR jsonb_typeof(logo_props -> 'emoji' -> 'url') = 'string')
      ELSE false END)
    AND (NOT logo_props ? 'icon' OR CASE WHEN jsonb_typeof(logo_props -> 'icon') = 'object' THEN
            ((logo_props -> 'icon') - array['name', 'color', 'background_color']) = '{}'::jsonb
        AND (NOT (logo_props -> 'icon') ? 'name' OR jsonb_typeof(logo_props -> 'icon' -> 'name') = 'string')
        AND (NOT (logo_props -> 'icon') ? 'color' OR jsonb_typeof(logo_props -> 'icon' -> 'color') = 'string')
        AND (NOT (logo_props -> 'icon') ? 'background_color'
             OR jsonb_typeof(logo_props -> 'icon' -> 'background_color') = 'string')
      ELSE false END)
  ELSE false END)
  ```
  - 第三稿 spike（Postgres 18.6）：`{}`、只有 `emoji`、只有 `icon`、三个键都有的值通过；Codex S5 的 `{"unexpected": true}`、`{"in_use": 17, "emoji": []}`，以及 `{"in_use": "other"}`、`{"emoji": {"value": 1}}`、`{"emoji": "x"}`、`{"icon": {"shape": "round"}}`、`{"icon": {"color": null}}`、`[]`、`"x"`、`null` 都得到 `check_violation`（23514）。这些值在 P4 逐个变成仓储测试（9.3）。
  - 嵌套的 `->` 要加括号：`-` 的优先级高于 `->`，写成 `logo_props -> 'emoji' - array[…]` 时先算 `'emoji' - array[…]`，报 JSON 语法错误。
  - 第二稿只查对象类型，把内部的形状交给接口（Codex M-5）。接口的结构是封闭的，所以不是能经接口写进坏数据的漏洞，但它是 M2 约定没有登记的例外；本稿照约定写全。
- **索引**：`projects_workspace_id_identifier_key ON (workspace_id, identifier) WHERE deleted_at IS NULL`；`projects_workspace_id_name_key ON (workspace_id, name) WHERE deleted_at IS NULL`（照搬；按工作区列出项目也用它）；`projects_workspace_id_idx ON (workspace_id)`（物理级联）。
- **删除的列**（差异清单已登记的 10 列）：`emoji`、`icon_prop`、`cover_image`、`description_text`、`description_html`、`page_view`、`is_time_tracking_enabled`、`is_issue_type_enabled`、`estimate_id`、`close_in`。**另外删除**：`default_state_id`（M1-P2 交接：新工作项的默认状态来自 `states.default`）；`cover_image_asset_id`（M5 加回，3.2）；`external_id`、`external_source`（总体设计 5.3）。

### 4.7 `project_members`（Plane 16 列 → 11 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | — |
| `workspace_id` | `uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE` | — |
| `project_id` | `uuid NOT NULL REFERENCES projects ON DELETE CASCADE` | — |
| `member_id` | `uuid NOT NULL REFERENCES users ON DELETE CASCADE` | Plane 可空，改为非空：没有成员的成员关系没有意义，Plane 也从不写入空值 |
| `role` | `smallint NOT NULL DEFAULT 5`，`CHECK (role IN (5, 15, 20))` | 同 4.3 |
| `is_active` | `boolean NOT NULL DEFAULT true` | 默认值新加 |
| `created_by_id`、`updated_by_id`、`created_at`、`updated_at`、`deleted_at` | 同 4.2 | — |

- **索引**：`project_members_project_id_member_id_key ON (project_id, member_id) WHERE deleted_at IS NULL`（照搬）；`project_members_member_id_idx ON (member_id) WHERE deleted_at IS NULL`（连带结束、停用、我的项目角色）；`project_members_workspace_id_idx ON (workspace_id)`、`project_members_project_id_idx ON (project_id)`（物理级联）。
- **删除的列**：`view_props`、`default_props`、`preferences`（差异清单已登记）；`sort_order`（被 `project_user_properties.sort_order` 取代，前端不读）；`comment`（没有写入方）。

### 4.8 `project_user_properties`（Plane 15 列 → 11 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | — |
| `workspace_id`、`project_id` | 同 4.7 | — |
| `user_id` | `uuid NOT NULL REFERENCES users ON DELETE CASCADE` | — |
| `preferences` | `jsonb NOT NULL DEFAULT '{"navigation": {"default_tab": "work_items", "hide_in_more_menu": []}}'`，CHECK：外层恰好一个键 `navigation`，它恰好有 `default_tab`（字符串）和 `hide_in_more_menu`（数组），按 M2 设计 3.13 包在 `CASE WHEN jsonb_typeof(…) = 'object'` 里 | 默认值来自模型，去掉 `pages`（文档页已砍）；CHECK 新加。spike：数组、字符串、多出 `pages` 键都得到 `check_violation`（23514） |
| `sort_order` | `double precision NOT NULL DEFAULT 65535` | 默认值新加 |
| `created_by_id`、`updated_by_id`、`created_at`、`updated_at`、`deleted_at` | 同 4.2 | — |

- **索引**：`project_user_properties_project_id_user_id_key ON (project_id, user_id) WHERE deleted_at IS NULL`；`project_user_properties_workspace_id_idx ON (workspace_id)`、`project_user_properties_project_id_idx ON (project_id)`（物理级联）。
- **删除的列**：`filters`、`display_filters`、`display_properties`、`rich_filters`（M4 按自己的格式加，3.18）。

### 4.9 `states`（Plane 18 列 → 14 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | — |
| `workspace_id`、`project_id` | 同 4.7 | — |
| `name` | `varchar(255) NOT NULL`，`CHECK (name <> '')` | CHECK 新加 |
| `description` | `text NOT NULL DEFAULT ''` | 默认值新加 |
| `color` | `varchar(255) NOT NULL` | 照搬 |
| `sequence` | `double precision NOT NULL DEFAULT 65535` | 默认值新加 |
| `"group"` | `varchar(20) NOT NULL DEFAULT 'backlog'`，`CHECK ("group" IN ('backlog', 'unstarted', 'started', 'completed', 'cancelled', 'triage'))` | 默认值、CHECK 新加（`StateGroup`）。列名照搬，SQL 中带引号 |
| `"default"` | `boolean NOT NULL DEFAULT false` | 默认值新加 |
| `created_by_id`、`updated_by_id`、`created_at`、`updated_at`、`deleted_at` | 同 4.2 | — |

- **索引**：`states_project_id_name_key ON (project_id, name) WHERE deleted_at IS NULL`（照搬）；`states_project_id_default_key ON (project_id) WHERE "default" AND deleted_at IS NULL`、`states_project_id_triage_key ON (project_id) WHERE "group" = 'triage' AND deleted_at IS NULL`（新加，3.17；名字按本节开头的补充）；`states_workspace_id_idx ON (workspace_id)`、`states_project_id_idx ON (project_id)`（物理级联）。
- **删除的列**：`slug`、`is_triage`（3.17）；`external_id`、`external_source`。

### 4.10 `labels`（Plane 15 列 → 12 列）
| 列 | 类型与约束 | 与 Plane 的差异 |
|---|---|---|
| `id` | `uuid PRIMARY KEY` | — |
| `workspace_id` | `uuid NOT NULL REFERENCES workspaces ON DELETE CASCADE` | — |
| `project_id` | `uuid NOT NULL REFERENCES projects ON DELETE CASCADE` | Plane 可空，改为非空（3.16） |
| `parent_id` | `uuid REFERENCES labels ON DELETE CASCADE`；`CONSTRAINT labels_not_own_parent_check CHECK (parent_id <> id)` | CHECK 新加；两层的规则在领域层，并发下由项目行的锁保证（3.16） |
| `name` | `varchar(255) NOT NULL`，`CHECK (name <> '')` | CHECK 新加 |
| `color` | `varchar(255) NOT NULL DEFAULT ''` | 默认值新加（模型 `blank=True`） |
| `sort_order` | `double precision NOT NULL DEFAULT 65535` | 默认值新加；新建时为最大值加 10000（照搬） |
| `created_by_id`、`updated_by_id`、`created_at`、`updated_at`、`deleted_at` | 同 4.2 | — |

- **索引**：`labels_project_id_name_key ON (project_id, lower(name)) WHERE deleted_at IS NULL`（不分大小写，3.16）；`labels_parent_id_idx ON (parent_id)`（删除父标签时找子标签，也服务物理级联，所以不带条件）；`labels_workspace_id_idx ON (workspace_id)`、`labels_project_id_idx ON (project_id)`（物理级联）。快照中 `project_id` 为空时全表按名称唯一的索引不保留。
- **删除的列**：`description`（3.16）；`external_id`、`external_source`。

### 4.11 行为差异（登记到差异清单第四节）
| 方面 | Plane | Nerve | Phase |
|---|---|---|---|
| 工作区的 slug | 服务端接受大写；删除时改名腾出 slug；`PATCH` 能改 | 只有小写；部分唯一索引，不改名；建好之后不能改（3.10） | P1 |
| 保留的工作区名 | 前后端各一份，服务端 45+ 个含产品词 | 一份：本站用到的 10 个顶层路径加 4 个预留段（3.10） | P1 |
| 建工作区之后 | 投递 `workspace_seed`：机器人管理员、演示项目和工作项 | 什么都不投递（3.11） | P1 |
| 关闭创建工作区时 | 实例管理员在管理后台为自己建 | `nerve workspaces create --admin-email` 指定管理员（3.11） | P1 |
| 集合型的列表 | —（Plane 也一次返回全部） | 一次返回全部，`{data}` 封套；关联字段带 `_id`（差异清单第三节，3.12、5.2） | P1 |
| 工作区的显示设置 | `GET` 时 `get_or_create` | `GET` 不写库，没有行时返回默认值；第一次修改时建（3.18） | P2 |
| 删除工作区 | 清掉别人的 `last_workspace_id`；连带软删除由 Celery 异步完成 | 不清（落点规则让它无害）；连带软删除在同一个事务里（3.14，总体设计 5.5） | P2（P3、P4、P7 补全连带） |
| 邀请的列出、创建、修改、删除 | 工作区管理员和成员；修改不限制角色，成员能把邀请改成管理员 | 只有工作区管理员（决策点 4）；邀请的角色不高于邀请人（3.8） | P3 |
| 邀请令牌 | JWT，原文存库；公开的查看不要令牌，返回被邀请的邮箱；关闭注册时有邀请的邮箱就能注册 | 由签名密钥派生的 MAC，不存库；查看要令牌、不返回邮箱；注册要有效令牌且邮箱一致（3.8、决策点 1） | P3 |
| 重复的邀请 | 静默忽略 | 422 `duplicate`（3.8） | P3 |
| 接受邀请之后 | 服务端写 `last_workspace_id` | 前端写（3.14） | P3 |
| 接受邀请时已有成员行 | 不分有效还是已离开，都把角色改为邀请的角色（`views/workspace/invite.py:188-195`） | 已是有效成员：只消费邀请，成员关系和角色不变；以前的成员行：恢复，角色取邀请的，访客时连带项目角色（3.8） | P3（项目的连带 P4） |
| 加入项目时恢复以前的成员行 | 只改 `is_active`，保留旧的角色（`views/project/invite.py:157-174`）：被移出的项目管理员自己加入就拿回管理员 | 角色取 `min(原来那一行的角色, 他现在的工作区角色)`：不比新加入给得更多（Nerve 加的上限，被移出的项目管理员回来是他的工作区角色），也不比原来那一行更多（与 Plane 相同，被降为访客的人离开再加入仍是访客）（3.5） | P4 |
| 项目负责人、默认负责人 | `CASCADE`；负责人可以是任何账户 | `SET NULL`；创建时是工作区的有效管理员或成员，修改时是项目中不是访客的有效成员（3.15、3.19） | P4 |
| 看得到而不是成员时取项目 | 409（公开）或 403（私密） | 200，`member_role` 为 `null`（3.19） | P4 |
| 工作区访客取没加入的公开项目 | 409 | 404：看不到（3.19） | P4 |
| 已归档的项目 | 取单个 404；列表与未归档的混在一起 | 取单个照常返回；列表默认不含，`?archived=true` 只含（3.19） | P4 |
| 修改、删除项目 | 不是项目成员的工作区管理员也能 | 项目级规则（3.4） | P4 |
| 归档、恢复项目 | 项目管理员和成员 | 项目管理员（3.4） | P4 |
| 项目标识 | 最多 12 个字符，只禁一组符号 | 1–10 个，只能是大写字母、数字和 `ÇŞĞİÖÜ`（3.19） | P4 |
| 添加已是有效成员的人为项目成员 | 顺手改他的角色 | 422 `duplicate`（3.5） | P4 |
| 移出成员、离开工作区时的唯一管理员检查 | 移出：比较了错误的 id，查的又是"只有一个成员"的项目；离开：查"只有他一人的项目" | 3.7 规则 2 | P5 |
| 恢复被移出的成员 | 管理命令 `reactivate_workspace_member`（位置参数） | `nerve workspaces reactivate-member --slug --email`，行为照搬（3.11） | P5 |
| 移出成员、离开工作区时发给他的待接受邀请 | 不动：他凭旧链接就能回来 | 同一个事务里软删除这个工作区里发给他邮箱的待接受邀请，回来要新的邀请（3.8） | P5 |
| 停用账户 | 唯一管理员的检查从不拒绝 | 3.7 规则 2；停用成员关系、删除发给这个邮箱的全部邀请在同一个事务里（3.9） | P6 |
| 修改状态 | 项目的访客也能 | 项目管理员（3.4） | P7 |
| 默认状态、分诊状态 | 在代码里维持唯一；`is_triage` 与 `group` 不一致 | 数据库保证各至多一个；只看 `group`（3.17） | P4、P7 |
| 一组中唯一的状态 | 服务端能删除、能改组（页面不让） | 409 `project.state_last_in_group`（3.17） | P7 |
| 标签的名称 | 数据库区分大小写，序列化器不区分 | 数据库不分大小写（3.16） | P7 |
| 标签的层级 | 服务端不检查 | 服务端执行页面的两层规则，并发下由项目行的锁保证（3.16） | P7 |
| 工作区级标签 | 模型允许 | 没有（3.16） | P7 |
| 列出标签 | 任何工作区成员都得到 200，已离开的成员仍能列出 | 项目的有效成员（3.4） | P7 |

### 4.12 删除关系图（M3 的表）
在 M2 设计 4.7 的图上延伸。M3 的表没有 `DO_NOTHING`（不写 `ON DELETE`）的外键。
```
users
 ├── workspace_members.member_id                   ON DELETE CASCADE
 ├── workspace_user_properties.user_id             ON DELETE CASCADE
 ├── project_members.member_id                     ON DELETE CASCADE
 ├── project_user_properties.user_id               ON DELETE CASCADE
 ├── projects.project_lead_id                      ON DELETE SET NULL   （Plane CASCADE，3.15）
 ├── projects.default_assignee_id                  ON DELETE SET NULL   （Plane CASCADE，3.15）
 └── 九张表的 created_by_id、updated_by_id          ON DELETE SET NULL
workspaces
 ├── workspace_members.workspace_id                ON DELETE CASCADE
 ├── workspace_member_invites.workspace_id         ON DELETE CASCADE
 ├── workspace_user_properties.workspace_id        ON DELETE CASCADE
 ├── projects.workspace_id                         ON DELETE CASCADE
 └── project_members、project_user_properties、states、labels 的 workspace_id   ON DELETE CASCADE
projects
 ├── project_members.project_id                    ON DELETE CASCADE
 ├── project_user_properties.project_id            ON DELETE CASCADE
 ├── states.project_id                             ON DELETE CASCADE
 └── labels.project_id                             ON DELETE CASCADE
labels
 └── labels.parent_id                              ON DELETE CASCADE
（profiles.last_workspace_id 不是外键，3.14）
```
- **允许物理删除的范围**：v0 从不物理删除账户（M2 设计 4.7），所以指向 `users` 的外键在 v0 不会触发。M3 自己只软删除。物理删除由 M4 的 60 天清理做（总体设计 5.5、6.7）：它删除软删除超过 60 天的工作区、项目、成员、邀请、显示设置、状态、标签，`CASCADE` 带走其下的行。
- 图中 `workspaces`、`projects`、`labels` 之下的每条外键都有不带条件的索引（本节开头），清理的级联按索引找子行。M4 自己的表（工作项引用状态、标签）的外键去向和索引由 M4 在这张图上延伸（13.2）。

---

## 5. 接口

### 5.1 操作
两个新模块文件：`api/modules/workspace.yaml`、`api/modules/project.yaml`（模块文件名等于模块目录名，M2 设计 3.12）；`access` 没有操作。路径都不带结尾 `/`（M1-P3 交接：项目标识的地址随新接口统一）。每个操作写 `security` 和 `x-problem-codes`（M2 设计 3.11、3.12）；下表"主要错误"一栏就是后者的内容，不再列出所有操作都可能返回的平台码。带 `bearer` 的操作另隐含 `unauthorized`；带请求体的都可能返回 400 `bad_request`（结构）。平台码 `forbidden` 写作 403 F，`validation_failed` 写作 422 V。一个操作可以声明别的模块的码（例如 `project.yaml` 的操作答 `workspace.not_found`），前缀是拒绝它的那个模块（按 11.7 修订的 M2 设计 3.11）。

**`workspace`**（19 个）：

| 方法与路径 | operationId | 认证 | 请求 | 成功 | 主要错误 |
|---|---|---|---|---|---|
| `GET /api/v0/workspaces` | `listWorkspaces` | bearer | — | 200 `WorkspaceList` | — |
| `POST /api/v0/workspaces` | `createWorkspace` | bearer | `WorkspaceCreate {name, slug, organization_size?, timezone?}` | 201 `Workspace` | 403 `workspace.creation_disabled`；422 V；409 `workspace.slug_taken` |
| `GET /api/v0/workspaces/{slug}` | `getWorkspace` | bearer | — | 200 `Workspace` | 404 `workspace.not_found` |
| `PATCH /api/v0/workspaces/{slug}` | `updateWorkspace` | bearer | `WorkspaceUpdate {name?, organization_size?, timezone?}` | 200 `Workspace` | 404；403 F；422 V |
| `DELETE /api/v0/workspaces/{slug}` | `deleteWorkspace` | bearer | — | 204 | 404；403 F |
| `GET /api/v0/workspace-slugs/{slug}` | `checkWorkspaceSlug` | bearer | — | 200 `SlugAvailability {available, reason?: taken \| reserved \| invalid}` | — |
| `GET /api/v0/workspaces/{slug}/members` | `listWorkspaceMembers` | bearer | — | 200 `WorkspaceMemberList`（含已不是有效成员的，带 `is_active`） | 404 |
| `PATCH /api/v0/workspace-members/{workspace_member_id}` | `updateWorkspaceMember` | bearer | `WorkspaceMemberUpdate {role}` | 200 `WorkspaceMember` | 404 `workspace.member_not_found`；403 F；409 `workspace.own_membership`；422 V |
| `DELETE /api/v0/workspace-members/{workspace_member_id}` | `removeWorkspaceMember` | bearer | — | 204 | 404；403 F；409 `workspace.own_membership`；409 `project.sole_admin` |
| `POST /api/v0/workspaces/{slug}/leave` | `leaveWorkspace` | bearer | — | 204 | 404；409 `workspace.sole_admin`；409 `project.sole_admin` |
| `GET /api/v0/workspaces/{slug}/invitations` | `listWorkspaceInvitations` | bearer | — | 200 `WorkspaceInvitationList` | 404；403 F |
| `POST /api/v0/workspaces/{slug}/invitations` | `createWorkspaceInvitations` | bearer | `WorkspaceInvitationsCreate {invitations: [{email, role}]}`（1–100） | 201 `WorkspaceInvitationList` | 404；403 F；422 V（`invitations[i].email` 的 `invalid_format`、`duplicate`、`not_allowed`） |
| `PATCH /api/v0/workspace-invitations/{invitation_id}` | `updateWorkspaceInvitation` | bearer | `WorkspaceInvitationUpdate {role}` | 200 `WorkspaceInvitation` | 404 `workspace.invitation_not_found`；403 F；422 V；409 `workspace.invitation_responded` |
| `DELETE /api/v0/workspace-invitations/{invitation_id}` | `deleteWorkspaceInvitation` | bearer | — | 204 | 404；403 F |
| `GET /api/v0/workspace-invitations/{invitation_id}` | `getWorkspaceInvitation` | 公开 | 查询参数 `token`（必填） | 200 `InvitationPreview` | 404 `workspace.invitation_not_found` |
| `POST /api/v0/workspace-invitations/{invitation_id}/accept` | `acceptWorkspaceInvitation` | bearer | `InvitationResponse {token}` | 200 `Workspace`（已是有效成员时 `role` 是他不变的角色，3.8） | 404；403 `workspace.invitation_email_mismatch`；409 `workspace.invitation_responded` |
| `POST /api/v0/workspace-invitations/{invitation_id}/decline` | `declineWorkspaceInvitation` | bearer | `InvitationResponse {token}` | 204 | 同上 |
| `GET /api/v0/me/workspaces/{slug}/preferences` | `getWorkspacePreferences` | bearer | — | 200 `WorkspacePreferences` | 404 |
| `PATCH /api/v0/me/workspaces/{slug}/preferences` | `updateWorkspacePreferences` | bearer | `WorkspacePreferencesUpdate` | 200 `WorkspacePreferences` | 404；422 V |

**`project`**（26 个）：

| 方法与路径 | operationId | 认证 | 请求 | 成功 | 主要错误 |
|---|---|---|---|---|---|
| `GET /api/v0/workspaces/{slug}/projects` | `listProjects` | bearer | 查询参数 `archived`（`true` 只要已归档的） | 200 `ProjectList` | 404 `workspace.not_found` |
| `POST /api/v0/workspaces/{slug}/projects` | `createProject` | bearer | `ProjectCreate {name, identifier, description?, network?, project_lead_id?, logo_props?, timezone?}` | 201 `Project` | 404；403 F；422 V；409 `project.identifier_taken`、`project.name_taken` |
| `GET /api/v0/workspaces/{slug}/project-identifiers/{identifier}` | `checkProjectIdentifier` | bearer | — | 200 `IdentifierAvailability {available}` | 404；403 F |
| `GET /api/v0/projects/{project_id}` | `getProject` | bearer | — | 200 `Project` | 404 `project.not_found` |
| `PATCH /api/v0/projects/{project_id}` | `updateProject` | bearer | `ProjectUpdate`（5.2） | 200 `Project` | 404；403 F；409 `project.archived`；422 V；409 `project.identifier_taken`、`project.name_taken` |
| `DELETE /api/v0/projects/{project_id}` | `deleteProject` | bearer | — | 204 | 404；403 F |
| `POST /api/v0/projects/{project_id}/archive` | `archiveProject` | bearer | — | 200 `Project` | 404；403 F |
| `POST /api/v0/projects/{project_id}/unarchive` | `unarchiveProject` | bearer | — | 200 `Project` | 404；403 F |
| `GET /api/v0/projects/{project_id}/members` | `listProjectMembers` | bearer | — | 200 `ProjectMemberList` | 404；403 F |
| `POST /api/v0/projects/{project_id}/members` | `addProjectMembers` | bearer | `ProjectMembersAdd {members: [{member_id, role}]}`（1–100） | 201 `ProjectMemberList` | 404；403 F；422 V（`members[i].member_id` 的 `not_allowed`、`duplicate`，`members[i].role` 的 `not_allowed`） |
| `PATCH /api/v0/project-members/{project_member_id}` | `updateProjectMember` | bearer | `ProjectMemberUpdate {role}` | 200 `ProjectMember` | 404 `project.member_not_found`；403 F；409 `project.own_membership`；403 `project.role_too_high`；422 V |
| `DELETE /api/v0/project-members/{project_member_id}` | `removeProjectMember` | bearer | — | 204 | 404；403 F；409 `project.own_membership`；403 `project.role_too_high` |
| `POST /api/v0/projects/{project_id}/join` | `joinProject` | bearer | — | 200 `Project` | 404；403 F |
| `POST /api/v0/projects/{project_id}/leave` | `leaveProject` | bearer | — | 204 | 404；403 F；409 `project.sole_admin` |
| `GET /api/v0/me/projects/{project_id}/preferences` | `getProjectPreferences` | bearer | — | 200 `ProjectPreferences` | 404；403 F |
| `PATCH /api/v0/me/projects/{project_id}/preferences` | `updateProjectPreferences` | bearer | `ProjectPreferencesUpdate {navigation?, sort_order?}` | 200 `ProjectPreferences` | 404；403 F；422 V |
| `GET /api/v0/projects/{project_id}/states` | `listStates` | bearer | — | 200 `StateList`（已归档的项目为空） | 404；403 F |
| `POST /api/v0/projects/{project_id}/states` | `createState` | bearer | `StateCreate {name, color, group, description?}` | 201 `State` | 404；403 F；422 V；409 `project.state_name_taken` |
| `PATCH /api/v0/states/{state_id}` | `updateState` | bearer | `StateUpdate {name?, color?, group?, description?, sequence?}` | 200 `State` | 404 `project.state_not_found`；403 F；422 V；409 `project.state_name_taken`、`project.state_last_in_group` |
| `DELETE /api/v0/states/{state_id}` | `deleteState` | bearer | — | 204 | 404；403 F；409 `project.state_default`、`project.state_last_in_group` |
| `POST /api/v0/states/{state_id}/mark-default` | `markDefaultState` | bearer | — | 204 | 404；403 F |
| `GET /api/v0/workspaces/{slug}/states` | `listWorkspaceStates` | bearer | — | 200 `StateList`（他是有效成员的、未归档的项目；不含分诊） | 404 `workspace.not_found` |
| `GET /api/v0/projects/{project_id}/labels` | `listLabels` | bearer | — | 200 `LabelList` | 404；403 F |
| `POST /api/v0/projects/{project_id}/labels` | `createLabel` | bearer | `LabelCreate {name, color?, parent_id?, sort_order?}` | 201 `Label` | 404；403 F；422 V；409 `project.label_name_taken` |
| `PATCH /api/v0/labels/{label_id}` | `updateLabel` | bearer | `LabelUpdate {name?, color?, parent_id?, sort_order?}` | 200 `Label` | 404 `project.label_not_found`；403 F；422 V；409 `project.label_name_taken` |
| `DELETE /api/v0/labels/{label_id}` | `deleteLabel` | bearer | — | 204 | 404；403 F |

**`identity` 的修改**：
- `POST /api/v0/auth/register`：`RegisterRequest` 加可选的 `invitation {id, token}`（3.8）；错误码不变。
- `POST /api/v0/me/deactivate`：`x-problem-codes` 加 `workspace.sole_admin`、`project.sole_admin`（3.9）。

**路径的说明**：
- 按总体设计 3.2：列表和创建挂在父资源下（`/workspaces/{slug}/…`、`/projects/{id}/…`），单个资源用短路径（`/workspace-members/{id}`、`/states/{id}`），业务动作用 `POST /资源/{id}/动作名`（`archive`、`unarchive`、`join`、`leave`、`accept`、`decline`、`mark-default`）。
- 工作区用 slug 寻址（与前端的地址一致，Plane 也是），其余资源用 uuid。
- 成员关系的路径参数叫 `workspace_member_id`、`project_member_id`：它们是成员关系行的 id；`member_id` 在别处（`ProjectMember.member_id`、`ProjectMembersAdd`）都指账户的 id，不混用。
- 调用者自己的显示设置放在 `/me/…` 下，路径属于 `workspace`、`project` 的模块文件（M2 设计 3.12："`/me/recent-visits` 属于它自己的模块"）。
- slug 可用性的路径是 `/workspace-slugs/{slug}`，不是 `/workspaces/{slug}/…`：不是成员时 `/workspaces/{slug}` 答 404，两者不混。
- 没有 Plane 的 `workspace-members/me`、`project-members/me`、`project-roles`：调用者的角色在 `Workspace.role` 和 `Project.member_role` 里（Plane 的列表也带着它们），前端的权限 store 从这两处取。
- 每个列表的顺序见 3.12。

### 5.2 结构
- **关联字段只给 id，名字带 `_id`**（总体设计 3.2、3.6）：Plane 的 `project_lead`、`default_assignee`、`parent`、`workspace`、项目成员的 `member` 改为 `project_lead_id`、`default_assignee_id`、`parent_id`、`workspace_id`、`member_id`。登记在差异清单第三节。
- **例外：工作区成员内嵌用户的公开资料**。`WorkspaceMember {id, workspace_id, role, is_active, created_at, member: MemberUser}`，`MemberUser {id, display_name, first_name, last_name, avatar_url, email}`。
  - 理由：v0 没有读别的用户的接口，成员列表是看到同事的名字和头像的唯一来源（Plane 同样内嵌，`serializers/workspace.py:93-116`）。`MemberUser` 取代前端的 `IUserLite`（M2 交接第 7、11 节：`avatar_url` 可空，`is_bot` 删除），用在全部显示人名的地方。
  - `email`：调用者是管理员或成员时是对方的邮箱，访客看到 `null`（与 Plane 相同，3.4）。`avatar_url` 在 M5 之前为 `null`。
  - 总体设计 3.6 补上这个例外（3.20，架构问题 11.3）。
- `Workspace {id, name, slug, organization_size, timezone, logo_url, role, total_members, created_at, updated_at}`：`role` 是调用者的角色，`total_members` 是有效成员数（前端的工作区菜单和首页引导读它）。没有 `owner`（M2 交接第 11 节：`owner` 的类型"按接口再定"，结论是不定义）。
- `WorkspaceInvitation {id, workspace_id, email, role, accepted, responded_at, created_at, created_by_id, token}`：`token` 由服务端算出，只有能管理邀请的人拿得到这个结构；页面用它拼链接。`responded_at` 有值而 `accepted` 为假就是"已忽略"。
- `InvitationPreview {id, role, declined, workspace_name, workspace_slug, workspace_logo_url}`：公开查看的视图，平铺工作区的三项（调用者读不了工作区本身）。**没有被邀请的邮箱**（决策点 1）。
- `WorkspacePreferences {navigation_control_preference, navigation_project_limit}`。
- `Project {id, workspace_id, name, description, identifier, network, project_lead_id, default_assignee_id, cycle_view, module_view, issue_views_view, intake_view, guest_view_all_features, archive_in, archived_at, logo_props, timezone, cover_image_url, member_role, sort_order, member_ids, created_at, updated_at}`：`member_role`、`sort_order` 是调用者的，不是成员时为 `null`。`ProjectUpdate` 是除 `id`、`workspace_id`、`archived_at`、`cover_image_url`、`member_role`、`sort_order`、`member_ids`、时间以外各字段的可选版本；可空的字段（`project_lead_id`、`default_assignee_id`）用 `nullable.Nullable` 区分"没传"和"清空"（M2 设计 3.12），规则见 3.19。
- `logo_props`：`{in_use?: "emoji" | "icon", emoji?: {value?, url?}, icon?: {name?, color?, background_color?}}`，照前端的 `TLogoProps`（`web/packages/types/src/common.ts:21-32`）写成封闭的结构（不允许别的键），字段都可选，`{}` 表示没有图标（3.19）；数据库的 CHECK 与它相同（4.6）。
- `ProjectMember {id, project_id, member_id, role, created_at}`：列表只含有效成员、且仍是工作区有效成员的人（Plane `views/project/member.py:156-169`）。查询只读 `project_members` 的有效行，不 JOIN `workspace_members`（别的模块的表，6.5）：约定三、六的不变式保证有效的项目成员仍是工作区的有效成员（项目一侧的增长锁住并确认工作区成员行，收缩连带结束项目成员关系，3.6）。
- `ProjectPreferences {navigation: {default_tab, hide_in_more_menu}, sort_order}`：`default_tab` 取 `work_items`、`cycles`、`modules`、`views`、`intake` 之一（`core/components/navigation/use-navigation-items.ts:38-78`），`hide_in_more_menu` 是这些值的集合（不含 `work_items`）；`navigation` 整体替换。
- `State {id, workspace_id, project_id, name, description, color, group, default, sequence, created_at, updated_at}`：`group` 在接口里不出现 `triage`（3.17）。没有 `order`。
- `Label {id, workspace_id, project_id, parent_id, name, color, sort_order, created_at, updated_at}`。
- 列表：`{data: [...]}`（3.12）。
- 结构中新的开放 map：没有（M2 设计 3.12 的 `closedObject` 不需要例外）。
- 请求体中第一次出现对象的数组（`invitations[]`、`members[]`）：`apitest` 生成的请求体用例扩展到数组里的对象，数组元素里的未知字段、缺少的必填字段同样 400（M2 设计 3.11 的结构检查，9.4）。

### 5.3 错误码
| 码 | 状态 | 含义 |
|---|---|---|
| `forbidden`（平台码） | 403 | 看得到，但规则表不允许调用者的角色做这件事（3.4）。`access` 的拒绝，跨越所有模块 |
| `workspace.not_found` | 404 | 工作区不存在、已删除，或调用者不是它的有效成员 |
| `workspace.creation_disabled` | 403 | `workspace.creation_enabled = false`（3.11） |
| `workspace.slug_taken` | 409 | slug 已被未删除的工作区占用 |
| `workspace.member_not_found` | 404 | 成员不存在，或不在调用者看得到的工作区里 |
| `workspace.own_membership` | 409 | 改自己的角色，或移出自己（请用"离开"）（Plane 400） |
| `workspace.sole_admin` | 409 | 唯一的管理员离开；或连带结束、停用时，他是某个还有别的成员的工作区唯一的管理员（3.7） |
| `workspace.invitation_not_found` | 404 | 邀请不存在、已删除、已接受，或令牌不对（公开的查看、接受、忽略都一样） |
| `workspace.invitation_email_mismatch` | 403 | 登录账户的邮箱不是邀请的邮箱；回答不含邀请的邮箱（3.8） |
| `workspace.invitation_responded` | 409 | 邀请已被忽略，不能再接受、忽略或改角色 |
| `project.not_found` | 404 | 项目不存在、已删除，或调用者看不到它（3.4） |
| `project.identifier_taken` | 409 | 标识已被本工作区未删除的项目占用 |
| `project.name_taken` | 409 | 名称同上 |
| `project.archived` | 409 | 修改已归档的项目（3.19） |
| `project.member_not_found` | 404 | 项目成员不存在，或项目看不到 |
| `project.own_membership` | 409 | 不是工作区管理员而改自己的项目角色；移出自己 |
| `project.role_too_high` | 403 | 改或移出项目角色不低于（改）或高于（移出）自己的人；给出不低于自己的角色（3.5） |
| `project.sole_admin` | 409 | 唯一的项目管理员离开；或连带结束、停用时他是某个还有别的成员的项目唯一的管理员（3.7） |
| `project.state_not_found` | 404 | 状态不存在、已删除、是分诊状态，或项目看不到 |
| `project.state_name_taken` | 409 | 项目内同名的状态 |
| `project.state_default` | 409 | 删除默认状态 |
| `project.state_last_in_group` | 409 | 删除一组中唯一的状态，或把它改到别的组（3.17） |
| `project.label_not_found` | 404 | 标签不存在、已删除，或项目看不到 |
| `project.label_name_taken` | 409 | 项目内同名的标签（不分大小写） |

- 取值不合规一律 422 `validation_failed`，字段错误的码取自封闭集合（总体设计 3.5），M3 不新增字段错误码：例如保留的 slug 是 `slug` 的 `not_allowed`，把工作区访客加为项目成员是 `members[i].role` 的 `not_allowed`，三层标签是 `parent_id` 的 `not_allowed`，组为 `triage` 是 `group` 的 `not_allowed`。
- `apitest` 核对两个方向（每个返回的码都声明过，每个声明的码都有测试返回过，M2 设计 3.11），并核对每个前缀是 nerve 的一个模块（11.7）。"都有测试返回过"按模块的 HTTP 测试包计：声明在模块 X 的操作上的码，要在 X 自己的 HTTP 测试包里经 `CheckResponse` 返回过，即使产生它的是别的模块（9.4）。`deactivateMe` 的两个新码因此由 `identity` 的 HTTP 测试经它的假实现返回，`bootstrap` 的整程序测试另在真实的组合上返回。

### 5.4 限流
M3 不加新的桶：需要登录的操作按凭证计数（`authenticated`），公开的查看邀请按 IP 计数（`anonymous`）（总体设计 3.6）。邀请令牌是 128 位的 MAC，没有可以猜的空间；按 IP 的额度足够挡住扫描邀请 id。

---

## 6. 后端结构

### 6.1 新增和修改的包
| 包 | 内容 | 依赖 |
|---|---|---|
| `internal/shared` | 新：`authorize.go`（`Role`、`Action` 类型、`Target`、`Grant`、`Authorizer`、`ErrNotVisible`，3.4）；`error.go` 加平台码 `CodeForbidden = "forbidden"` 和它的构造函数；从 `identity/domain` 移来的 `email.go`、`url.go`、`timezone.go`（3.13）。没有事件（3.3） | 只有标准库 |
| `modules/access` | 6.4 | `internal/shared` |
| `modules/workspace` | 6.2 | `internal/shared`；适配器另外依赖平台、pgx |
| `modules/project` | 6.3 | 同上 |
| `modules/identity` | `SignupPolicy` 多两个参数、`RegisterRequest` 的 `invitation`（3.8）；`MembershipDeactivator` 端口，`deactivate` 在撤销会话之后调用它（3.9）；签名密钥由 `bootstrap` 先载入（`LoadKeys`，6.6），多一个按用途派生的 MAC；`Provide(pool)` 对外提供 `Accounts`（锁账户行并交回它的状态）、`CredentialLock` 和 `PublicProfiles`（按 id 批量读公开资料，不加锁，含停用的账户）（6.5）；`LockedAccount` 加 `Email`，停用把锁下的邮箱交给 `MembershipDeactivator`（3.9）；三条取值规则改调 `shared` | 同 M2 |
| `bootstrap` | 接线（6.6）；`ports.go`：把 `identity` 返回的值转成 `workspace` 端口的值（6.5）；注册策略 `signup_policy.go`（配置开关加邀请检查，取代 `signupSwitch`）；`workspaces.go`（`nerve workspaces` 的组合）；`users.go` 的组合加上停用要用的部分；测试：操作名的完整性（3.4）、保留名单"服务端"一段（3.10）、权限矩阵（9.2） | 全部 |
| `cmd/nerve` | `nerve workspaces create --slug --name --admin-email`、`nerve workspaces reactivate-member --slug --email`（3.11） | 同 M2 |
| `internal/archtest` | `TestSQLCSchemaScope` 的四种新写法和反例（4.1）；命令行的组合测试从 `Users` 扩展到 `Workspaces`，禁止的构造加上 `workspace.New`、`project.New`、`access.New`（6.6）；其余规则不变，三个新模块在现有规则下检查 | — |
| `platform/httpserver/apitest` | `platformCodes` 加 `forbidden`；前缀规则改为"前缀是 `api/modules/` 下的一个模块"（11.7）；请求体用例扩展到对象的数组（5.2） | — |
| `deploy/runtime-grants.sql` | 九张新表的读写（M2 设计 6.1：`runtime_role_test.go` 在建表的 Phase 就会要求） | — |

- 没有新的配置项（`workspace.creation_enabled` 已有），没有新的依赖。
- `platform` 的生产代码不变。

### 6.2 `workspace` 模块的结构
```
modules/workspace/
  domain/
    workspace.go            Workspace；名称、slug、规模、时区的规则
    reserved_slugs.txt      保留名单，go:embed，"应用""服务端""预留"三段（3.10）
    reserved.go             读名单；slug 的回答：可用、已被占用、保留、不合规
    member.go               Member；角色；改自己、移出自己的规则（5.3）；唯一管理员的两条规则（3.7）
    invitation.go           Invitation；令牌的格式和 MAC 的消息（3.8）；一批邀请的校验（1–100、重复）
    preferences.go          工作区显示设置的默认值和取值
    actions.go              本模块的操作名（shared.Action 常量）和 Actions()（3.4）
    errors.go               本模块的错误码（5.3）
  app/
    ports.go                仓储（按用例拆成小接口）、Clock、ProjectCascade、ProjectMembershipCounts、Accounts、MemberProfiles、CallerLock、InvitationMAC；跨边界的小值类型（AccountState、PublicProfile）
    create_workspace.go     Execute（调用者）和 ExecuteForAdmin（命令行，按邮箱）两个入口，一个用例（3.11）
    list_workspaces.go  get_workspace.go  update_workspace.go  delete_workspace.go  check_slug.go
    list_members.go  update_member.go  remove_member.go  leave_workspace.go
    end_membership.go       移出、离开共用的一步：查唯一管理员、删除发给他的待接受邀请、停用成员行、ProjectCascade.EndMemberships（3.6、3.7、3.8）
    reactivate_member.go    nerve workspaces reactivate-member（3.11）
    list_invitations.go  create_invitations.go  update_invitation.go  delete_invitation.go
    preview_invitation.go   公开的查看，不含邮箱（3.8）
    respond_invitation.go   接受和忽略
    get_preferences.go  update_preferences.go
    deactivate_memberships.go  identity 的 MembershipDeactivator 的实现（3.9）
    signup_invitation.go    注册策略要的检查：邀请有效、令牌对、邮箱一致（3.8）
  adapter/
    postgres/               仓储；对外提供的读取和锁（6.5）；queries/*.sql；gen/（sqlc）
    http/                   handler，按资源分文件；公开操作的清单；gen/
  module.go                 Provide(pool)；New(Deps)；Register；PublicOperations()；Actions()；Deactivator()；SignupInvitations()
  admin.go                  NewAdmin(AdminDeps)：命令行的建工作区、恢复成员和停用
```

### 6.3 `project` 模块的结构
```
modules/project/
  domain/
    project.go              Project；名称、标识、可见性、archive_in、logo_props、时区、负责人的规则（3.19）
    member.go               ProjectMember；3.5 的相对规则（纯函数，输入是 Grant 和目标）；唯一管理员（3.7）
    preferences.go          导航偏好的取值；新成员的 sort_order（3.18）
    state.go                State；默认的 6 个状态（db/models/state.py:24-62）；组、sequence、一组至少一个（3.17）
    label.go                Label；两层的规则（3.16）
    actions.go              本模块的操作名和 Actions()（3.4）
    errors.go
  app/
    ports.go                仓储、Clock、WorkspaceDirectory、WorkspaceMembers
    create_project.go  list_projects.go  get_project.go  update_project.go  delete_project.go
    archive_project.go      归档和恢复
    check_identifier.go
    list_members.go  add_members.go  update_member.go  remove_member.go  join_project.go  leave_project.go
    cascade.go              workspace 的 ProjectCascade 的实现：DeleteWorkspaceProjects、EndMemberships、DemoteToGuest（3.3、3.6、3.7）
    get_preferences.go  update_preferences.go
    list_states.go  create_state.go  update_state.go  delete_state.go  mark_default_state.go  list_workspace_states.go
    list_labels.go  create_label.go  update_label.go  delete_label.go
  adapter/
    postgres/  http/        同 workspace
  module.go                 Provide(pool)（ProjectAccess、ProjectMembershipCounts）；New(Deps)；NewCascade(CascadeDeps)；Register；Actions()；Cascade()
```
- 一个用例一个文件（总体设计 6.1），每个预计 40–120 行；`domain` 的文件都在 400 行以内。按调用者过滤的列表（项目、工作区的状态）在适配器里写专门的查询（总体设计 6.3"明确不做的事"），它们与规则表一致由 9.3 的测试守住（3.4）。
- `NewCascade` 只建 `cascade.go` 的三个方法（它们只用 `project` 自己的仓储，时间由调用方传入，事务取自调用方的 `ctx`）；`New` 用它建出同一个实现，由 `Cascade()` 交出。命令行的组合只要连带，调用 `NewCascade`，不建 HTTP 的一侧（6.6）。

### 6.4 `access` 模块的结构
```
modules/access/
  domain/
    rules.go                规则表：每个操作名一行，级别和允许的角色（3.4、9.2）；RuleKeys()
    decide.go               判定：输入是端口读到的事实，输出 Grant、看不到或不能做。纯函数
  app/
    ports.go                WorkspaceRoles、ProjectAccess（6.5）
    authorizer.go           Authorizer：读事实 → decide
  module.go                 New(Deps) 返回 shared.Authorizer 的实现；RuleKeys()
```
- 没有 `adapter`，没有表，没有 sqlc 条目（3.3）。
- 规则表的三个级别：
  - **工作区**：调用者在这个工作区的有效角色在集合里；
  - **项目**：调用者看得到这个项目，是它的有效成员且项目角色在集合里，或是它的成员而同时是工作区管理员；
  - **看得到即可**：只有 `project.read` 和 `project.join`（后者另要求工作区角色不低于成员，所以访客加入公开项目也是 404：他看不到它）。
- 账户级的操作（列出自己的工作区、建工作区、检查 slug、邀请的查看、接受、忽略）不经过 `Authorizer`：它们不属于任何工作区，任何有效的凭证都可以调用，规则在用例里（3.8、3.11）。
- 端口的读取在调用方的事务里（`ctx` 带着事务）：写操作在取得父行的锁之后判定，读到的是锁之后已提交的角色（3.6 约定二）。

### 6.5 端口
| 端口 | 声明在 | 实现 | 测试 |
|---|---|---|---|
| `Authorizer`（`Authorize(ctx, actor, action, target) (Grant, error)`） | `internal/shared` | `access/app` | `access/domain` 的判定表测试（每条规则、每种身份）；9.2 的矩阵在真实的组合上逐格 |
| `WorkspaceRoles`（调用者在工作区的有效角色） | `access/app` | `workspace/adapter/postgres`（`workspace.Provide`） | `workspace` 的集成测试；已删除的工作区、`is_active = false` 都算不是成员 |
| `ProjectAccess`（项目是否存在、所属工作区、可见性、调用者的有效项目角色） | `access/app` | `project/adapter/postgres`（`project.Provide`） | 同上；已删除的项目算不存在，已归档的照常 |
| `WorkspaceDirectory`（按 slug 取工作区的 id 和时区） | `project/app` | `workspace/adapter/postgres`（`workspace.Provide`） | 集成测试 |
| `WorkspaceMembers`（按 `id` 升序 `FOR SHARE` 锁住若干人的工作区成员行，返回是否有效和角色；约定三） | `project/app` | 同上 | 集成测试；交错测试 4、5、17（3.6） |
| `ProjectCascade`（`DeleteWorkspaceProjects`、`EndMemberships`、`DemoteToGuest`，3.3；`EndMemberships` 在调用时列举项目，3.6 约定六） | `workspace/app` | `project/app` 的 `cascade.go`（`project.New` 的 `Cascade()`，或命令行的 `project.NewCascade`） | 用例测试；删除工作区、降为访客、移出的集成测试核对连带的行与工作区的写入在同一事务（连带失败时工作区的写入也回滚）；交错测试 1、4、5、6、13–15、17；跨工作区的 `EndMemberships` 按 `id` 升序锁项目、跳过上锁时已删除的项目（P6） |
| `ProjectMembershipCounts`（`CountInactive(ctx, workspaceID, userID) (int, error)`：他在这个工作区无效的项目成员关系个数，不加锁） | `workspace/app` | `project/adapter/postgres`（`project.Provide`） | 集成测试；W12 的输出 |
| `Accounts`（按 id、按邮箱以 `FOR SHARE` 锁住账户行，交回 `AccountState{ID, Email, Active}`；没有这个账户时交回"不存在"。用例决定怎样对待停用的账户：建工作区、接受和忽略邀请要求有效，`reactivate-member` 允许，3.6 约定六。只作事务的第一把锁，约定一） | `workspace/app` | `identity`（`identity.Provide`，存储之上的薄实现；值在 `bootstrap/ports.go` 转换） | 集成测试（有效、停用、不存在三种）；交错测试 7、8、12、16、19 |
| `MemberProfiles`（`PublicProfiles(ctx, ids) ([]PublicProfile, error)`：按 id 批量读显示名、名字、邮箱，不加锁，停用的账户也返回；`avatar_url` 在 M5 之前为空。成员列表用它；事务中途要读邮箱时也只用它，约定一） | `workspace/app` | `identity`（`identity.Provide`，一条查询；值在 `bootstrap/ports.go` 转换） | 集成测试（含停用的账户、不存在的 id）；`listWorkspaceMembers` 的用例测试核对访客看到的 `email` 为 `null` |
| `CallerLock`（`LockCaller(ctx, actor, now) error`：锁调用者的账户行并复核凭证） | `workspace/app` | `identity` 的 `CredentialLock`（`identity.Provide`） | 交错测试 9 |
| `InvitationMAC`（`Tag`、`Verify`） | `workspace/app` | `identity/adapter/signing` 按用途派生的 MAC（`info = "nerve workspace-invitation mac v1"`，经 `identity.LoadKeys` 的结果） | 往返；改动邀请 id 任一字节不成立；换了签名密钥之后不成立；与刷新令牌的 MAC 密钥不同；比较用 `hmac.Equal` |
| `MembershipDeactivator` | `identity/app` | `workspace/app` 的 `deactivate_memberships.go` | 停用的接口和命令两条路（3.9、9.3） |
| `SignupPolicy`（`AllowSignup(ctx, email, invitation)`） | `identity/app` | `bootstrap` 的 `signup_policy.go`：配置开关，关闭时交给 `workspace` 的 `SignupInvitations` | 用例测试；W6 |

- **没有给 `access` 的 sqlc 例外**：两个读取端口由拥有那张表的模块实现（3.3）。
- **跨边界的值**：端口的参数和结果只用标准库和 `shared` 的类型，或者调用方的 `app` 层声明的小值类型（`AccountState`、`PublicProfile`）。实现方返回它自己的类型时，由 `bootstrap/ports.go` 里几行的函数转换；模块之间不互相导入（架构测试）。一个模块的 SQL 不 JOIN 别的模块的表：sqlc 的 schema 按模块装载，JOIN `users` 的查询根本编译不出来（Codex S6），成员列表因此先读 `workspace_members`，再经 `MemberProfiles` 一次读齐资料，在用例里拼。
- **每个端口只有一种真实的实现**：总体设计 6.1 的里氏契约测试针对 `local`、`s3` 这样有两种实现的端口，这里不需要；用例测试用手写的假实现。

### 6.6 组合：没有构造的环
模块之间的端口是双向的：`project` 要 `workspace` 的目录和成员锁，`workspace` 要 `project` 的连带和项目成员关系的个数；`access` 要两者的读取，两者又要 `access`；`identity` 要 `workspace` 的停用实现和邀请检查，`workspace` 要 `identity` 的账户锁、公开资料、凭证复核和 MAC。每个模块分两段构造：
- **`Provide(pool)`** 只返回只依赖连接池的适配器，不建用例。唯一的例外是 `identity.Provide` 的 `CredentialLock`：它是 M2 的协议步骤（锁账户行、复核凭证版本），只用 `identity` 的存储；放进 `identity.New` 会让 `workspace.New` 等 `identity.New`，而 `identity.New` 要 `workspace` 的停用实现，成环。
- **`New(Deps)`** 建出全部用例和 HTTP 的一侧，`Deps` 里是它要的全部端口。构造之后不再登记、不再注入任何东西。

`bootstrap` 按下面的顺序接线：
1. **载入签名密钥**：`identity.LoadKeys(pem)`（原在 `identity.New` 里）。返回的值不导出密钥，只提供 `identity.New` 要的 JWT 和刷新令牌的 MAC，以及按用途派生的 MAC（3.8 的"密钥不离开 `signing`"照旧）。
2. **适配器**：`identity.Provide(pool)`（`Accounts`、`PublicProfiles`、`CredentialLock`）、`workspace.Provide(pool)`（`WorkspaceRoles`、`WorkspaceDirectory`、`WorkspaceMembers`）、`project.Provide(pool)`（`ProjectAccess`、`ProjectMembershipCounts`）。
3. `access.New`（`WorkspaceRoles`、`ProjectAccess`）→ `Authorizer`。
4. `project.New`（`Authorizer`、`WorkspaceDirectory`、`WorkspaceMembers`）→ 它的 `Cascade()`。
5. `workspace.New`（`Authorizer`、`project` 的 `Cascade()` 和 `ProjectMembershipCounts`、`Accounts` 和 `MemberProfiles`（经 `bootstrap/ports.go` 转换）、`CallerLock`、邀请的 MAC、创建开关）→ 它的 `Deactivator()`、`SignupInvitations()`。
6. `identity.New`（密钥、`SignupPolicy` = 配置开关加 `workspace` 的 `SignupInvitations`、`MembershipDeactivator` = `workspace` 的 `Deactivator`）。

- 第三稿加的三个读取（`PublicProfiles`、`Accounts` 交回的状态、`ProjectMembershipCounts`）都是第 2 步的适配器，只依赖连接池，顺序不变、仍然没有环（Codex M-1 的核对）。

- **命令行的组合**（M2 设计 3.17 的最小组合）：
  - `nerve users …`（`bootstrap.Users`）：连接池、`identity.NewAdmin`，加上停用要的 `workspace.NewAdmin`（它的 `Deactivator`）和 `project.NewCascade`。
  - `nerve workspaces …`（`bootstrap.Workspaces`）：连接池、`identity.Provide` 的 `Accounts`、`project.Provide` 的 `ProjectMembershipCounts`、`workspace.NewAdmin`（建工作区、恢复成员）、`project.NewCascade`（`NewAdmin` 的依赖，与上一个组合相同）。
  - 两个组合都没有签名密钥、`Authorizer` 和 River（停用和建工作区都不投递任务，3.9、3.11）。
  - 组合测试（`archtest/composition_test.go`）从 `Users` 扩展到 `Workspaces`：从它们出发的静态调用不到达 `identity.New`、`workspace.New`、`project.New`、`access.New`、HTTP 服务、限流和任务队列，并且到达 `workspace.NewAdmin`（证明检查看到了组合）。
- `bootstrap` 的整程序测试（M2 设计 3.11 的四个）照旧覆盖全部新操作：默认拒绝（每个非公开操作不带令牌 401）、请求体结构、参数绑定、错误码。

### 6.7 一个写请求的权限判定：`PATCH /api/v0/states/{state_id}`
```
请求 → …（M2 设计 6.4 的固定链、路由、参数绑定、认证、限流、请求体结构）
     → handler：RequireActor；把 StateUpdate 转成用例的入参
     → UpdateState.Execute(ctx, actor, stateID, in)：
         0. 值的校验：名称、颜色、group ≠ triage、sequence（不合规 → 422），在事务之前，只看请求
       TxManager.WithinTx {
         1. states.Get(stateID)：未删除、不是分诊状态；没有 → 404 project.state_not_found
         2. projects.LockForNoKeyUpdate(state.ProjectID)          （约定二：状态的写入锁项目行，3.6、3.17）
              SELECT … WHERE id = $1 AND deleted_at IS NULL FOR NO KEY UPDATE
              → 0 行（项目已删除，或等锁时被删除）：404 project.state_not_found
         3. states.Get(stateID) 重读：等锁期间被删除 → 404 project.state_not_found
         4. Authorizer.Authorize(ctx, actor, project.ActionStateUpdate, Target{WorkspaceID, ProjectID})
              access：WorkspaceRoles → 调用者在这个工作区的有效角色（没有 → 看不到）
                      ProjectAccess  → 项目未删除、可见性、调用者的有效项目角色
                      decide(rules["state.update"] = 项目级 {Admin}, 事实)
              → 看不到：shared.ErrNotVisible → 用例换成 404 project.state_not_found（与第 1 步不存在时相同）
              → 看得到、不是项目管理员、也不是同时是工作区管理员的项目成员：403 forbidden
              → 允许：Grant{WorkspaceRole, ProjectRole, ProjectAdmin}
         5. 依赖行的检查：改组而它是原组中唯一的状态 → 409 project.state_last_in_group，在判定之后
         6. states.Update；唯一约束冲突 → 409 project.state_name_taken
       }
     → 200 State；或者 error → APIErrors → problem+json
```
- **要点**：
  - 锁在判定之前（3.6 约定二）：第 4 步读到的角色是取锁之后已提交的，等锁期间被降级、被移出的调用者在这里被拒绝。第一稿把判定放在第 2 步、没有锁，评审的 spike 证明那样两位管理员能互相降级到一个不剩（3.6）。
  - 资源先读、锁父行、再重读：用例要从资源行知道它属于哪个项目。资源不存在与看不到得到同一个 404 和同一个码，不透露私密项目里有没有这个 id（8.2）。
  - 值的校验在事务之前，依赖行的检查在判定之后（3.6 约定二）：值的校验的 422 只看请求，对不存在的 id、没有权限的人答的都一样，不透露任何行，也不为明显错误的请求开事务、取锁；依赖行的检查放在判定之后，包括依赖行的 422（约定三中目标的 422；邀请已是有效成员的 `not_allowed`、已有邀请的 `duplicate`，3.8），没有权限的人得不到目标的任何信息（8.2）。
  - 判定每次都读（3.4），不缓存。
  - `Grant` 交回用例：需要比较两个人的用例（改项目成员的角色、移出）直接用它，不再查调用者的角色。
- **列表**：`GET /workspaces/{slug}/projects` 先 `WorkspaceDirectory`（没有 → 404 `workspace.not_found`），再判定工作区级的 `project.list`（看不到 → 同一个 404），再由适配器的查询按 3.4 的可见性过滤（调用者的工作区角色从 `Grant` 取），`?archived` 决定只要未归档或只要已归档的。
- **事务**：写操作一个事务，锁按 3.6；读操作不开事务，直接判定（M2 设计 6.4 的"单条语句的写入不开事务"同理，读取之间没有要一致的不变式）。

---

## 7. 前端

前端先做数据层（P8），再做页面（P9–P11）：P8 把 M3 的全部类型、service、store 和两个包装层挂载时的取数一次迁到新接口，页面的 Phase 加入故事时，挂载路径上已经没有旧接口的请求（3.1、第 12 节）。

### 7.1 按会话分代（总体设计 7.7；M2 交接第 3 节）
- **service 按代建**：M3 的 service 都是生成的客户端之上的薄封装，构造时接收这一代的 `ApiClient`，由 store 在构造时建（`ApiTokenStore` 的写法，`core/store/user/api-token.store.ts`）。M2 交接列出的 10 处模块级实例在 P8 全部消失；只调公开操作的（注册页和邀请页查看邀请，`getWorkspaceInvitation`）用 `publicClient`，可以是模块级的。
- **`RootStore` 的释放**：M3 是第一个接上注册在沿用对象上的反应的 M，按总体设计 7.7 给 `RootStore` 加上 `dispose()`，由 `store-context.tsx` 在换代时调用旧的一代。要释放的只有 `project_filter.store.ts:63` 观察 `router.workspaceSlug` 的 `reaction`：store 保存它返回的释放函数，`dispose()` 调用它。单元测试：换代之后改路由，旧一代的反应不再运行。`cycle_filter`、`module_filter`、`issue_calendar_view` 的同类反应和 `issue/root.store.ts:170` 的 `autorun` 属于 M4、M6，照同一写法由它们接上（13.2）。
- **SWR 键**：M3 加回和新写的 SWR 键都带 `loginId`（例如 `["workspaces", loginId]`）。fetcher 返回 store 方法的 Promise（M2 交接第 2 节"注意"）。
- **一个资源的修改一个接一个发出**（总体设计 7.7）：每个 store 对同一个资源的修改经 `oneAtATime()`（`core/lib/one-at-a-time.ts`）排队，前一个有了应答或失败之后才发下一个。M3 里连续修改最多的是拖动排序（状态的 `sequence`、标签的父子和 `sort_order`、侧边栏的项目顺序）和显示设置；表单另外在提交期间禁用按钮。只排队修改，取数不排队。
- **`SessionChangedError`**：store 的修改遇到它时不改本代的状态、不提示错误（M2 的做法）。
- **组件在 `await` 之后的页面级副作用先核对会话**（总体设计 7.7 的规则，Codex 4.3 第 2 条）：`SessionChangedError` 只管 store 发出请求之前换了代的情形；已经发出的请求在另一个标签页换了账户之后照样可能迟到地成功，旧的组件闭包仍会拿到结果。所以组件在修改成功之后跳转、提示、改页面状态之前，先核对标签页仍在发出修改时的会话。
  - `inSession()` 从 `core/components/appearance/theme-switcher.tsx:50` 的闭包移到 `core/lib/in-session.ts`：`const inSession = sessionGuard()` 在发出修改时取当前的 `loginId`，之后 `inSession()` 比较它（P8）。
  - M3 的页面里这样的组件逐个列进 P9、P10 的 spec，例如 `project/delete-project-modal.tsx:62-69`、`workspace/delete-workspace-form.tsx:67-75`（删除之后跳转和提示）、离开工作区和项目、创建工作区和项目之后的跳转、接受邀请之后的跳转。
  - 测试：`sessionGuard` 的 vitest（P8）；上面两个删除组件各一个 vitest（修改的 Promise 在 `loginId` 改变之后才兑现，组件不跳转、不提示）（P9、P10）；端到端在 W3 的页面版本里（P9）：
    - 删除工作区的请求在换账户之前经 `route.fetch()` 到达 nerve 并成功，它的回答在另一个标签页换了账户之后才经 `route.fulfill()` 交给页面；原标签页不跳转、不提示。
    - 不能在换账户之后才放行请求：退出撤销了原会话，每个请求都查会话（M2 设计 3.5），放行的请求得到 401，按总体设计 7.7 成为 `SessionChangedError`；页面因为请求失败而不跳转，没有 `inSession()` 也会通过（复核 M6）。
    - 变异核对：去掉组件里 `inSession()` 的核对，这个端到端必须失败。
- **权限决定取数，不只决定显示**（Codex 4.3 第 1 条）：页面只请求调用者有权读的资源；按权限隐藏的区域，它的取数同样按权限启用（SWR 的键在没有权限时为 `null`）。M3 的每一页照此写，后续 M 照做：
  - 工作区设置的成员页：成员列表单独取；邀请列表只在调用者是工作区管理员时取（`workspace/settings/members-list.tsx:46-55` 现在不分角色先取邀请，只在 `:79` 控制显示）。
  - `ProjectAuthWrapper`：先取项目详情；调用者是有效的项目成员（`member_role` 不为 `null`）之后，才取项目的显示设置、标签、成员和状态（`layouts/auth-layout/project-wrapper.tsx:70-95` 现在同时发出，不是成员时后四个都是 403）。
  - 测试：W4 的页面版本（成员打开成员页）和 P2 的页面版本（不是成员的人直接打开公开项目的地址），`watchPage` 断言没有失败的请求（P9、P10）；两处的取数条件各有一个 vitest（P8）。

### 7.2 类型
生成的类型（`@nerve/api-client`，`--root-types`）取代下面的 Plane 类型，没有转换层（总体设计 7.2）：

| Plane 类型（`web/packages/types/src/`） | 生成的类型 | 说明 |
|---|---|---|
| `IWorkspace` | `Workspace` | `owner` 没有了（M2 交接第 11 节）；`logo_url` 可空 |
| `IWorkspaceMember`、`IWorkspaceMemberMe` | `WorkspaceMember` | 调用者自己的角色从 `Workspace.role` 取 |
| `IUserLite` | `MemberUser` | `avatar_url` 可空；`is_bot` 删除（M2 交接第 7、11 节） |
| `IWorkspaceMemberInvitation`、`IWorkspaceBulkInviteFormData` | `WorkspaceInvitation`、`WorkspaceInvitationsCreate`、`InvitationPreview` | `invite_link` 由页面用 `id` 和 `token` 拼出；`InvitationPreview` 没有邮箱 |
| `IWorkspaceUserPropertiesResponse` 的导航部分 | `WorkspacePreferences` | 筛选部分属于 M4、M7 |
| `IProject`、`TProject` | `Project` | 关联字段带 `_id`；`intake_count`、`cover_image`、`inbox_view`、`is_favorite` 没有了（3.1、3.2） |
| `TProjectMembership`、`IProjectBulkAddFormData` | `ProjectMember`、`ProjectMembersAdd` | 显示人名时从工作区成员按 `member_id` 取 |
| `IProjectUserPropertiesResponse` 的导航和顺序、`TProjectNavigationPreferences`、`IProjectMemberNavigationPreferences` | `ProjectPreferences` | 筛选部分属于 M4 |
| `IState` | `State` | `order` 由 store 算（3.17） |
| `IIssueLabel`（`IIssueLabelTree` 由 store 从它算出） | `Label` | `parent` → `parent_id` |
| `IUserSettings` | — | 删除（3.14） |
| `EUserWorkspaceRoles`、`EUserProjectRoles` | 保留为常量 | 取值 5、15、20 与生成的 `Role` 相同；页面用它们比较角色 |

- `packages/types/src/workspace.ts`、`project/`、`state.ts` 中 M3 的部分随之删除；仍被 M4 以后的代码引用的少数类型（例如工作项的筛选、`TProjectAppliedDisplayFilterKeys`）留在原处，由它们的 M 处理。
- 使用方（第 1 节的统计：`IProject` 20 个文件、`IState` 23、`IIssueLabel` 25、`IUserLite` 14 等）在 P8 逐个改用生成的类型：大部分只改导入和字段名（`parent`、`project_lead`、`default_assignee`、`member`），属于 7.11 的"跟随类型的使用方"。

### 7.3 services 与 stores（P8）
| store（`core/store/`） | 做法 |
|---|---|
| `workspace/index.ts`（`WorkspaceRootStore`） | 重写：工作区列表、当前工作区、创建、修改、删除、slug 检查；落点的计算移到 `AuthenticationWrapper` 用的一个纯函数（7.4）；`getWorkspaceRedirectionUrl` 删除 |
| `member/workspace/workspace-member.store.ts` | 重写：成员、邀请；搜索和筛选的部分（`workspace-member-filters.store.ts`）不变 |
| `member/project/project-member.store.ts` | 重写：项目成员、添加、改角色、移出、加入、离开；人名取自工作区成员 |
| `project/project.store.ts` | 重写：项目列表（未归档、已归档两份）、详情、创建、修改、归档、恢复、删除、侧边栏顺序；`is_favorite` 的读写、两个收藏方法和 `favoriteProjectIds` getter 删除，`favorite.store.ts` 的项目分支一起删除（3.2，M7 加回）；`project_filter.store.ts` 只加释放（7.1） |
| `state.store.ts` | 重写：项目的状态、工作区的状态；`sortStates`（`packages/utils/src/work-item/state.ts`）的顺序和分组中的位置 `order` 由 store 算出，读 `state.order` 的组件（`issue-layouts/utils.tsx:234`、`dropdowns/state/base.tsx:120,182` 等；`home/widgets/recents/issue.tsx` 随小部件删除）改读 store 的 getter |
| `label.store.ts` | 重写：项目的标签、层级；工作区级的标签取数（M7 的跨项目列表）删除，M7 加回（13.2） |
| `user/permissions.store.ts` | 重写：工作区角色取自 `Workspace.role`，项目角色取自 `Project.member_role`；Plane 的 `workspace-members/me`、`project-roles` 两次取数和模块级的 `WorkspaceService` 删除；`getProjectRole`（`:123-130`）的规则不变：先要是项目成员，工作区管理员才算项目管理员，与 3.4 的服务端规则一致 |
| `user/settings.store.ts` | 删除（3.14） |

- **services**（`core/services/`）：`workspace.service.ts`、`project/*.service.ts`、`issue/issue_label.service.ts` 中 M3 的 56 个方法改为按资源的新 service（`workspace`、`workspace-member`、`workspace-invitation`、`project`、`project-member`、`state`、`label`、`preferences`），每个方法一次调用生成的客户端。`user.service.ts` 的 `currentUserSettings`、`leaveWorkspace`、`joinProject`、`leaveProject` 移入对应的新 service 或删除（M2 交接第 11 节）。
- **挂载路径上的取数**：`WorkspaceAuthWrapper`、`ProjectAuthWrapper`、`TopNavigationRoot` 挂载时的取数在 P8 一起处理：M3 自己的改调新 store，并按 7.1 由权限启用；M6、M7 的删除（3.1）。
- **P8 的使用方**：store 的方法签名和字段一变，调用它们的页面和组件（第 1 节：`useProject` 129 个文件等）要在同一个 Phase 改到能编译、行为不变；页面的新行为（新手引导、邀请页、设置页）在 P9–P11。
- **旧 `ProjectService`（`core/services/project/project.service.ts`）留下三个方法**，其余的删除：
  - 取、改项目用户属性的两个方法：M4 的筛选 store 使用（`core/store/issue/project/filter.store.ts:127,174,236,256`，3.18）。它们的地址是 M3 替换的地址（`/user-properties/`），关键词守卫为它们登记例外（7.10）。
  - `projectIssuesSearch`：工作项搜索，三个调用方属于后续的 M，要在 M3 之后仍能编译（Codex M-3）：`core/components/issues/parent-issues-list-modal.tsx:68`（选父工作项，M4）、`core/components/core/modals/existing-issues-list-modal.tsx:104`（向迭代、模块加已有的工作项，M6）、`core/components/inbox/modals/select-duplicate.tsx:62`（收集箱查重，M7）。它的地址 `/search-issues/` 不是 M3 替换的地址，不在关键词规则里，不需要例外。
  - 选"留在旧 service"而不是"与调用方一起移到一个新的旧搜索 service"：移动只是换个文件名，三个调用方和它们的 M 都不变，M4 定下工作项搜索的接口时一次替换。
  - 旧 service 在 M4 替换最后一个方法时删除（13.2）。
- **时区**：工作区和项目设置的时区选择用 M2 的 `GET /api/v0/timezones`（M2 交接第 11 节）。

### 7.4 落点、新手引导、邀请页与注册页
- **落点**（3.14，P8）：`AuthenticationWrapper` 取 `GET /api/v0/workspaces`（SWR 键带 `loginId`），按"上次的工作区仍在列表中 → 它；否则最早创建的 → 它；一个都没有 → `/create-workspace`"算出，纯函数有 vitest。M2 为"完成引导的用户直接去 `/create-workspace`"写的分支改为这条规则。四处 `await fetchCurrentUserSettings()` 删除（M2 交接第 2 节）。切换工作区、创建工作区、接受邀请之后经 `PATCH /api/v0/me/profile` 写 `last_workspace_id`（P9）。
- **新手引导**（决策点 2，P9）：
  - 资料步骤之后，已经有工作区的人（受邀的人先接受、再完成引导）直接完成引导，进入落点。这是 `onboarding/root.tsx` 现有的分支（资料一步时 `workspacesList.length > 0` 就 `finishOnboarding()`），它的列表改由 `listWorkspaces` 提供。
  - 还没有工作区的人：资料之后是"创建工作区"，再是"邀请成员"（可跳过）。创建工作区关闭时，这一步显示"创建工作区已关闭，请向工作区的管理员要邀请链接"。邀请成员一步调 `createWorkspaceInvitations`，完成后显示每个邀请的链接供复制（v0 不发邮件）。
  - "加入工作区"一步（`onboarding/steps/workspace/join-invites.tsx`）和 `OnboardingRoot` 的 `invitations` 在 P8 随系统内接受一起删除（7.8）。
- **邀请页** `/workspace-invitations?invitation_id=…&token=…`（P9）：用 `publicClient` 查看（`getWorkspaceInvitation`），按回答显示：
  - 未登录：工作区名、角色和"这份邀请发给一个指定的邮箱"，"登录以接受"和"注册以接受"（都带 `next_path` 回到本页）；
  - 已登录：接受、忽略。接受之后写 `last_workspace_id`，进入工作区（还没完成引导的新账户先完成资料一步，见上）；他本来就是有效成员时接受同样成功，角色不变（3.8），页面同样进入工作区；跳转之前核对会话（7.1）；
  - 接受或忽略得到 403 `workspace.invitation_email_mismatch`：说明"这份邀请发给了另一个邮箱"，不说是哪一个，接受和忽略按钮不再可用，提供退出登录（决策点 1，W5）；
  - 已忽略（`declined`）：说明已忽略，请向管理员要新的邀请；
  - 链接无效（404）：说明，提供回到首页。
- **注册页**（决策点 1，P9）：带着 `invitation_id`、`token` 的注册页（`auth-root.tsx`、`auth-header.tsx` 已读 `invitation_id`）用公开的查看取工作区名，标题"加入 <工作区>"。邮箱不预填：查看不返回被邀请的邮箱。提交时 `RegisterRequest.invitation` 带上两者。注册关闭而没有邀请：照 M2 显示"注册已关闭"。
- **`/invitations` 页删除**（决策点 2，P8）：它的入口（设置的侧边栏、工作区菜单 `workspace-menu-root.tsx:204`、`workspace-options.tsx:47`、命令面板 `account-commands.ts:47`、`workspace-wrapper.tsx:210` 的链接）和路由（`routes/core.ts:28`）一起删除。

### 7.5 工作区的页面（P9）
- **工作区首页**：问候和"还没有项目"的引导；"最近"小部件在 P8 删除（3.1）。
- **侧边栏**：工作区菜单切换工作区；"项目导航"对话框读写 `WorkspacePreferences`（W8）；收藏区在 M7 之前为空（3.1）。项目的顺序在 P10。
- **工作区设置 general**：名称、规模、时区；slug 只读；图标上传删除（3.2）；删除工作区（输入名称确认）之后按落点规则去下一个工作区。
- **工作区设置 members**：
  - 成员列表：角色下拉只对管理员可用，不能改自己；访客看不到任何人的邮箱，`email` 都为 `null`，他自己的也是（与 Plane 相同，9.2）；离开工作区，唯一的管理员时显示 409 的说明。
  - 邀请只对管理员显示，也只为管理员取（决策点 4，与 Plane 的页面相同；7.1）：邀请弹窗（批量，每行邮箱和角色，422 的字段错误落到对应的行）；邀请列表的每一行有复制链接、改角色、删除。
  - 已忽略的邀请在列表中标"已忽略"，只能删除；再邀请这个邮箱得到的 422 `duplicate` 落到那一行，说明"这个邮箱已有一份邀请，先删除它"（3.8）。
- **停用账户**（M2 的 general 页）：409 `workspace.sole_admin`、`project.sole_admin` 在弹窗里显示"先指定另一位管理员"（W9）。
- **`WorkspaceAuthWrapper`**：按 `GET /api/v0/workspaces/{slug}` 显示工作区或"找不到工作区"（404）；取收藏的 SWR 在 P8 删除（3.1）；M2 改过的"找不到工作区"界面的退出按钮、`isLoading` 在这里核对（M2 交接第 13 节，9.7）。

### 7.6 项目的页面（P10、P11）
- **项目列表**：按 `listProjects` 的结果（已按可见性过滤）；卡片上的"加入"对看得到而不是成员的项目显示（`member_role` 为 `null`）；已归档的项目在 archives 页（`?archived=true`），恢复、删除。
- **创建项目**：名、标识（输入时转大写，只接受 3.19 的字符，`common-attributes.tsx:96-107` 的规则不变）、说明、公开或私密、负责人（从工作区的管理员和成员中选）、图标（表情或图标，表情数据见 7.7）；封面一步删除（3.2）。
- **`ProjectAuthWrapper`**：按 `getProject` 的回答：404 → "找不到项目"；`member_role` 为 `null` → "加入项目"的界面（原来按 409、403 分支，`project-access-restriction.tsx`）；`archived_at` 有值 → 已归档的界面。取迭代、模块、视图、分诊状态的 SWR 在 P8 删除（3.1）；显示设置、标签、成员、状态在确认是有效的项目成员之后才取（7.1）。
- **项目设置**：
  - general：名、标识、说明、可见性、时区、图标。
  - members：成员列表（添加、改角色、移出；添加时只列出还不是成员的工作区成员）；负责人、默认负责人（从项目中不是访客的成员里选）、"访客可见全部"（`project-settings-member-defaults.tsx`，由 `…/members/page.tsx:50` 渲染，与 Plane 相同）。
  - features：迭代、模块、视图、收集箱。
  - automations：只剩自动归档。
  - 不能编辑时表单只读，与 Plane 页面的判断相同，角色来自 `member_role` 和 `Workspace.role`。
- **离开项目的顺序**（M1-P4 交接）：**先等接口成功，再跳到项目列表**。失败时留在原页、弹窗显示错误，用户仍在项目里，页面与服务端一致；Plane 先跳后调，失败时用户已离开项目页却仍是成员。`leave-project-modal.tsx` 照此改，故事 P5 断言成功后才跳转。
- **侧边栏的项目顺序**：按 `ProjectPreferences.sort_order`，拖动排序调 `updateProjectPreferences`（经 `oneAtATime()`）。
- **导航偏好**：项目页头的标签页导航（`use-tab-preferences.ts`）读写 `ProjectPreferences.navigation`（P8 故事）。
- **状态设置**（P11）：列表按组、`sequence`；拖动改 `sequence`；设为默认；默认状态和一组中唯一的状态的删除按钮不可用（Plane 的页面已是这样），服务端并发下仍拒绝时显示 409 的说明（3.17）。
- **标签设置**（P11）：两层的拖放（`label-drag-n-drop-HOC.tsx`）照旧，服务端另外检查（3.16）；同名（不分大小写）409 落到名称字段。

### 7.7 下拉框、复制到剪贴板、表情选择器（P10）
- **下拉框**（M2 交接第 14 节）：
  - `CustomSearchSelect`（`web/packages/ui/src/dropdowns/custom-search-select.tsx`）改成像 Popover 那样可以用 Tab 到达的按钮，`Combobox` 放在它的面板里；
  - `dropdowns/member/member-options.tsx`：popper 的 ref 放在列表元素本身，有搜索框的用 `Combobox`、打开时输入框取得焦点；
  - `CustomSelect` 在 M3 页面上的 9 个调用方和 `CustomSearchSelect` 的 4 个调用方逐个在浏览器中核对（9.7）。
- **复制到剪贴板**：3 处未处理拒绝的调用（`workspace/sidebar/projects-list.tsx`、`project/card.tsx`、`extended-project-sidebar.tsx`）加上处理，失败时经 `t()` 提示；邀请行的复制链接（`invitations-list-item.tsx:80`，P9）照同样的写法。
- **表情选择器的数据从本站提供**（M2 交接第 10 节）：
  - `emojibase-data` 固定为 15.3.2（已在依赖树里，由 `@tiptap/extension-emoji` 带进），加为 web 应用的直接依赖；
  - 一个小的 Vite 插件在构建时把 `en/data.json`、`en/messages.json` 输出到 `/assets/emojibase/15.3.2/en/`（`/assets/` 已由 nerve 提供，与其他构建产物同一路径）；
  - `EmojiPicker.Root`（`web/packages/propel/src/emoji-icon-picker/emoji/emoji.tsx`）传 `emojibaseUrl="/assets/emojibase/15.3.2"`，版本固定，不再请求 `cdn.jsdelivr.net`；
  - CSP 不放开任何外部来源；构建产物多约 707 KB 的 JSON（只在打开表情选择器时下载，之后由 `frimousse` 缓存在 localStorage）；
  - 只支持英文的表情名称搜索（与 Plane 相同：它不传 `locale`，`frimousse` 默认取 `en`）；
  - `frimousse` 先发 `HEAD` 比对 ETag 再决定用不用缓存（0.3 的 spike）。`webui` 对嵌入的文件回答 `HEAD`，是否带 ETag 决定它每次打开都重新下载还是用缓存，功能不受影响；P10 核对并写进 review。
  - 守卫：一个 vitest 核对传入的地址和版本；故事 P1、P3 打开表情选择器，`watchPage` 断言没有 CSP 违规、没有发往外部的请求（总体设计 8.2 的 `cspViolations`）。

### 7.8 删除的代码
- 系统内接受（决策点 2，P8）：`/invitations` 页和它的入口、新手引导的"加入工作区"一步、按邮箱列出和批量接受的 service 方法和 store 方法。
- `RESTRICTED_URLS`（`web/packages/constants/src/workspace.ts`）和它的检查（3.10，P8）：创建表单改问 `checkWorkspaceSlug`；`navigation.test.ts` 中守着它的部分改为 3.10 的名单测试。
- `IUserSettings`、`settings.store.ts`、`currentUserSettings`（3.14，P8）。
- 10 处模块级的 service 实例（7.1，P8）。
- 首页的"最近"小部件和挂载时取收藏、未读通知数、迭代、模块、视图、分诊状态的 SWR（3.1，P8）。
- `permissions.store.ts` 中 `workspace-members/me`、`project-roles` 的取数（P8）。
- 项目侧边栏读 `intake_count` 的部分（3.1、3.2 的规则 3，P8）。
- `Project.is_favorite` 的读写（3.2，P8；M7 加回）：项目卡片的收藏按钮、建项目之后加入收藏的一步、`project.store.ts` 的两个收藏方法和 `favoriteProjectIds` getter、`favorite.store.ts:284-287` 的项目分支。
- `IUserLite.is_bot`（M2 交接第 11 节，P8）。
- 工作区图标、项目封面的上传控件和随机封面（3.2，P9、P10）。

### 7.9 死成员、死 prop 和 oxlint（M1 收尾交接）
- **死成员和死 prop**：`domains.mjs --rows M3` 在 `f8cb7c2` 上是 208 行，其中 41 行按路径归到 M3、实际属于别的 M（第 1 节），M3 自己的 167 行在它们的文件被重写或改到时删除，P11 清完剩下的。
  - 关闭条件照交接：M3 合并时 `--rows M3` 的每一行已消失，或写进 review（文件、名称、谁在读或传它）。
  - 那 41 行按归属写进 M4、M6、M7 的交接（13.2），`domains.mjs` 的 `DOMAINS` 不改（它是 M1 收尾计划附录 A 的一部分，改它等于改所有 M 的口径）。
- **oxlint**：
  - 谁改谁清：P8–P11 改到的文件在那个 Phase 合并时没有 oxlint 警告。M3 领域现在 73 条，分布在 39 个文件（第 1 节）。
  - 按规则清一类（P11）：`eslint-plugin-promise(always-return)` 全仓清零（62 条，其中 M3 领域 14 条）。它能机械修复（`then` 回调里补 `return` 或改为 `await`），也常与"复制到剪贴板"一类未处理的拒绝同处出现，修它顺带检查错误处理。`no-shadow`（165 条）数量太多、改名会碰到 M4–M7 的文件，不选。
  - 各包的上限随之调低（`tools/lint-cap.mjs`），review 写明各包的变化。

### 7.10 关键词守卫的新规则（`tools/keywords.json`，P8）
| 规则 | 禁止 |
|---|---|
| `plane-workspace-urls` | M3 替换的 Plane 地址：由 56 个旧方法的地址写出模式（工作区、成员、邀请、slug 检查、`workspace-members/me`、`project-roles`、项目、项目成员、归档、标识检查、状态、标签、显示设置）。一条例外：`core/services/project/project.service.ts` 中 `/user-properties/` 的两处（M4 的筛选 store 用，7.3），`until: "M4"`，原因写"M3/P8 起，M4 的工作项筛选改用新接口时删除"。`/search-issues/` 不是 M3 的地址，不在模式里；规则的不命中样例加上 `/api/workspaces/{slug}/projects/{id}/search-issues/`，守住项目地址的模式不会宽到它 |
| `plane-user-urls`（收紧） | 整个 `/api/users/`（M2 交接第 11 节）：7 处调用都是 M3 的，P8 全部迁走 |
| `project-invitations`（例外删除） | `joinProject` 的例外（`until: "M3"`）随它改用新接口删除（M1-P3 交接） |
| `restricted-urls` | `RESTRICTED_URLS` |
| `in-system-invitations` | `/invitations` 路由、`joinWorkspaces`、`getUserWorkspaceInvitations` 一类名称（决策点 2） |
| `user-settings` | `IUserSettings`、`currentUserSettings`、`fetchCurrentUserSettings` |

- 全部在 P8 加入：数据层一次迁走全部旧调用，规则从第一个前端 Phase 起就是完整的；P11 只核对 M3 加的例外只剩跨 M 的一条（`/user-properties/` 的两处，`until: "M4"`），`project-invitations` 的 `until: "M3"` 已删除。
- 每条规则带 `samples`（命中和不命中）和 `files` 样例，照 M2 设计 7.9 的写法；`node tools/keywords.mjs` 通过。

### 7.11 规模估计（按 M2 收尾附录 B 校准）
**前端**（M3 领域在 `f8cb7c2` 上 229 个文件、23,485 行，不含测试；分组的数法见 0.3 的 spike）：

| 文件组 | 文件 | 预计改到的文件 | 行 | 估计的删除 | 估计的重写 | 估计的新增 |
|---|---|---|---|---|---|---|
| services | 7 | 7 | 695 | 600 | — | 350 |
| stores | 14 | 14 | 3,415 | 1,700 | 400 | 700 |
| store 的 hooks | 6 | 6 | 102 | 20 | 40 | — |
| wrappers | 2 | 2 | 371 | 80 | 150 | — |
| 顶层页面（`create-workspace`、`invitations`、`workspace-invitations`） | 6 | 6 | 475 | 250 | 150 | 50 |
| 设置页的路由 | 32 | 15 | 1,737 | 100 | 300 | — |
| 项目列表和归档的路由 | 6 | 4 | 142 | — | 50 | — |
| 工作区的组件 | 50 | 35 | 5,838 | 450 | 900 | 100 |
| 项目的组件 | 50 | 40 | 5,639 | 350 | 1,000 | 100 |
| 状态、标签的组件 | 25 | 20 | 2,243 | 50 | 500 | — |
| 设置的组件 | 11 | 8 | 482 | — | 100 | — |
| 新手引导的工作区和邀请两步 | 6 | 6 | 911 | 300 | 200 | 50 |
| 成员、项目下拉框 | 7 | 5 | 865 | — | 200 | — |
| 类型 | 5 | 5 | 319 | 250 | — | — |
| 常量 | 2 | 2 | 251 | 60 | — | — |
| **组内小计** | **229** | **约 175** | **23,485** | **约 4,200** | **约 4,000** | **约 1,350** |
| 组外：跟随类型和 store 的使用方（约 140）、`promise(always-return)` 在 M3 领域之外的文件（约 25）、首页"最近"小部件、`ui` 的下拉框、表情插件（约 15） | — | 约 180 | — | 400 | 700 | 150 |
| 测试（vitest） | — | 约 25 | — | — | — | 3,000 |
| **合计** | | **约 380（330–430）** | | **约 4,600** | **约 4,700** | **约 1,500 + 测试 3,000** |

- 第一稿的合计"约 280–320 个文件"算错了：组内的文件数本身就是 229，加上组外和测试约 400（评审 M15）。本稿按"预计改到的文件"数：组内约 175 个（有的文件不用动，例如只显示名称的设置页路由），合计约 380。
- 按附录 B 的教训：领域文件组的大部分行估为"删除"，新写的 service、store 估为"新增"，"重写"只留给组件；测试单列，按新增生产代码的约 2 倍；横向的清理单列在组外。
- 生成的 `schema.gen.ts` 预计增加约 2,500 行（M2 的 13 个操作 803 行）。

**后端**：

| 项 | 估计 | 依据 |
|---|---|---|
| 生产代码（不含生成的代码） | 约 12,000 行（10,000–15,400） | 见下 |
| 测试 | 约 24,000 行（20,000–31,000） | M2 是生产代码的 2 倍；矩阵测试是表驱动的，每格一行 |
| 手写的 SQL | 约 1,500 行（查询约 1,200，迁移约 300） | M2 查询 227 行对 13 个操作 |
| 接口描述 | 约 2,200 行（`workspace.yaml` 约 950，`project.yaml` 约 1,250） | M2 约每个操作 48 行 |

- **上限**：M2 的 `identity` 13 个操作 4,462 行，每个操作约 343 行（M2 收尾附录 B）；M3 的 45 个操作按同一比率约 15,400 行。
- **下调的依据**：M2 的 4,462 行里约 1,091 行是与操作个数无关的机器（argon2 163、签名 186、认证 58、会话 158、令牌的认证 145、凭证锁 51、签发 64、密码 85、限额 79、River 的清理 102），除去之后 17 个用例（13 个操作加 4 个命令）每个约 198 行。M3 约 51 个用例（45 个操作，加上建工作区的命令入口、恢复成员、停用的实现、注册的邀请检查、三个连带）约 10,100 行；M3 自己的机器（`access` 约 400、`shared` 约 300、组合和命令约 600、`identity` 的改动约 250、保留名单和令牌约 150）约 1,700 行；合计约 11,800。M3 的用例更接近增删改查，但每个写用例多了锁父行、重读、判定三步，所以不再往下压。

**端到端和持续集成**：
- 故事 21 个（W1–W12、P1–P9；W10、W12 只有命令和接口，P9 只有页面），约 2,900 行（M2 每个故事约 130 行）；fixture 约 1,400 行（九张表的断言、准备数据的帮助函数；M2 的 fixture 是估计的 2 倍）。
- 测试约多 50 个（多数故事有页面、接口两个版本）；按 M2 实测每个约 0.9 秒，持续集成的 E2E 一步多约 45 秒，仍远低于考虑分片的约 5 分钟。

---

## 8. 安全

### 8.1 邀请令牌
- 128 位的 MAC，绑定邀请 id（3.8）：不能从一个邀请推出另一个的令牌，也不能猜。比较用 `hmac.Equal`（M2 设计 3.9 的理由）。
- 数据库里没有令牌：数据库泄露时待接受的链接不泄露（总体设计 5.3 对 PAT 的理由）。密钥是签名私钥按用途派生的（HKDF 的 `info` 不同），与刷新令牌的 MAC 密钥互不相同；泄露其中一个派生密钥不影响签名和另一个。
- **链接的保密是主要的控制**（决策点 1）：拿到链接的人能看到工作区名和角色；邮箱一致只保护已有账户的受邀者。公开的查看和邮箱不一致的拒绝都不说被邀请的邮箱，拿到链接的人还要另外知道那个邮箱，才能在受邀的人注册之前冒用它。
- 令牌在链接的查询参数里：
  - 服务端的访问日志只记路径，不记查询参数（`middleware.go:117,165`，M2 已是这样），P3 的测试核对访问日志里没有 `token=`；
  - 页面的 `Referrer-Policy: same-origin`（M2 设计 8.3）不把它带给第三方；
  - 浏览器历史里有这个链接，与 Plane 相同；接受之后邀请被软删除，链接随之失效。
- 没有有效期（与 Plane 相同）。让一个链接失效：删除这份邀请，重新邀请（新的 id、新的令牌）；换签名密钥让全部链接失效。同一部署的全部 nerve 进程必须用同一个密钥文件（JWT 本来就这样要求），否则一个进程发的链接在另一个进程上无效（README，8.7）。

### 8.2 枚举
| 入口 | 回答 | 说明 |
|---|---|---|
| 工作区（slug）不存在、已删除、不是成员 | 同一个 404 `workspace.not_found` | 不区分"有这个工作区而你不是成员" |
| 项目、状态、标签、成员的 id 不存在或看不到 | 各自同一个 404 | 6.7；id 是随机的 UUID |
| `GET /workspace-slugs/{slug}` | 被占用、保留、可用 | 需要登录；它回答的只是"这个名字能不能用"，Plane 的 slug 检查同样回答（`views/workspace/base.py:214-225`），创建表单需要它 |
| 公开的查看邀请 | 邀请不存在、已删除、已接受、令牌不对：同一个 404 | 不带令牌 400；按 IP 限流（5.4）；回答里没有被邀请的邮箱 |
| 接受、忽略 | 同上 404；令牌对而邮箱不一致 403 `workspace.invitation_email_mismatch` | 回答不含被邀请的邮箱；要试出邮箱，必须先拥有那个邮箱的账户 |
| 注册时带邀请 | 各种无效都是 403 `identity.signup_disabled` | 与不带邀请相同（3.8）。持有链接的人能用注册试被邀请的邮箱（对了就注册成功），每次尝试受按 IP 的注册限流（`register_ip`）约束；决策点 1 接受这个风险 |

### 8.3 404 与 403 的边界
- 看不到的一律 404，看得到而不能做的 403 `forbidden`（3.4）。规则只有一处（`access/domain/decide.go`），矩阵逐格测试（9.2）；列表的过滤与逐个判定一致（9.3）。
- 默认拒绝：规则表里没有的操作一律拒绝；`bootstrap` 的测试核对各模块的 `Actions()` 与规则表的键一一对应（3.4）；一个新操作忘了调 `Authorize`，矩阵中它那一行的 403、404 格就会失败。
- 写操作在父行的锁之后判定（3.6 约定二）：等锁期间被降级、被移出的人拿不到过期的授权。
- 请求体的结构检查（M2 设计 3.11）挡住越权的字段：`PATCH /workspaces/{slug}` 带 `slug`，`PATCH /projects/{id}` 带 `workspace_id`、`archived_at`，都是 400。数组里的对象同样检查（5.2）。

### 8.4 邮箱与个人信息
- 成员列表：访客看不到任何人的邮箱，`email` 都为 `null`，他自己的也是；管理员、成员看得到每个人的，与 Plane 相同（3.4、9.2）。
- 邀请列表只有管理员看得到（决策点 4），其中有被邀请的邮箱和可用的令牌。
- 公开的查看邀请不显示被邀请的邮箱，邮箱不一致的拒绝也不说（决策点 1）。代价：用错账户打开链接的人不知道该用哪个邮箱，要问发链接的管理员。
- `MemberUser` 只有公开的资料（显示名、名字、头像）和按角色给出的邮箱，没有时区、最后登录等。

### 8.5 CSP
- 表情选择器的数据从本站提供（7.7），CSP 不放开外部来源（M2 交接第 10 节）。
- 工作区图标、项目封面在 M5 之前没有外部图片（3.2）。项目的 `logo_props.emoji.url` 是 Plane 为自定义表情留的字段，M3 的页面不写它；接口按 5.2 的封闭结构接受它，但页面只渲染 `value`，不按它请求图片（P10 核对）。

### 8.6 其他
- **账户被盗之后**：重置密码撤销会话和 PAT（M2），但不撤销他以前发出的邀请（3.8）。README 的"账户被盗之后"加一步：在各工作区的成员页核对待接受的邀请，删除不认识的。
- **无人管理的工作区**：唯一的管理员被停用之后（他是那里唯一的成员时允许，3.7），或者成员被误移出之后，服务器管理员用 `nerve users activate` 恢复账户、`nerve workspaces reactivate-member` 恢复成员关系（3.11），与 Plane 的两个管理命令相同。项目成员关系由恢复出来的管理员经接口加入项目恢复。README 和命令的说明写明这两步。
- **工作区名**：保留名单挡住与本站路径冲突的名字（3.10），名称不能含网址（3.10）。

### 8.7 README 的新增内容
| Phase | 位置 | 内容 |
|---|---|---|
| P1 | 部署 | `nerve workspaces create`；`workspace.creation_enabled = false` 时用它建工作区 |
| P3 | 部署、安全 | 邀请链接由签名密钥派生：同一部署的全部 nerve 进程用同一个密钥文件；换钥之后要重新复制链接；没有配置密钥的 dev、test 重启之后链接失效；让一个泄露的链接失效：删除邀请、重新邀请；账户被盗之后核对待接受的邀请 |
| P5 | 部署 | `nerve workspaces reactivate-member`：恢复被移出的成员，项目成员关系仍无效，由他经接口加入项目恢复 |
| P6 | 部署 | 停用会结束他的全部成员关系；`nerve users activate` 只恢复账户，成员关系按工作区用 `reactivate-member` 恢复 |

- "前端"一节的改写在 P9（3.20）。

---

## 9. 测试

### 9.1 后端单元测试
- **`access/domain`**：判定表：每条规则 × 每种身份（9.2 的列）× 项目公开、私密、已归档、已删除 × 成员关系有效、无效。
- **操作名的完整性**（`bootstrap`）：`workspace.Actions()`、`project.Actions()` 的并集恰好等于 `access.RuleKeys()`，两个模块之间没有重名（3.4）。
- **领域规则**：slug（大小写、长度、字符、保留名）；工作区名称（字母或数字、网址）；项目名称的禁用字符、标识（大写、长度、`ÇŞĞİÖÜ`、`-`）；`archive_in`；`logo_props`（字段都可选、未知的键被拒绝）；负责人、默认负责人（创建、修改两套规则）；状态（组、`triage`、默认、一组至少一个、`sequence` 的算法）；标签（两层、自己做父、别的项目）；3.5 的相对规则（一张表：调用者的角色 × 目标的角色 × 新角色 × 是否工作区管理员）；唯一管理员的两条规则（3.7）；新成员的 `sort_order`（建项目、添加、加入）。
- **邀请令牌**：格式（`nrv_inv_` 加 22 个字符）；往返；改一个字符不成立；另一个邀请的令牌不成立；换密钥不成立。
- **接受邀请的用例**（P3，手写的假实现）：一张表覆盖他在这个工作区的三种状态：已是有效成员（邀请被消费，成员关系和角色不变，回答他不变的角色；邀请的角色高于、低于、等于他的角色各一行）、有无效的成员行（恢复，角色取邀请的）、没有成员行（新建）；账户在锁下读到已停用时 401；锁下的邮箱与邀请不一致时 403，数据库不变。忽略的用例同样按锁下的状态和邮箱回答 401、403。
- **恢复时的角色**（P4，3.6 约定六"恢复不比新授予给得更多，也不比原来那一行更多"）：
  - 加入恢复以前的项目成员行时角色是 `min(原来那一行的角色, 他现在的工作区角色)`，一张表：
    | 原来那一行 | 工作区角色 | 恢复为 | 情形 |
    |---|---|---|---|
    | 5 | 15 | 5 | 被项目管理员降为访客的人离开再加入，不能给自己升级 |
    | 20 | 15 | 15 | 被移出的项目管理员（故事 P5） |
    | 15 | 20 | 15 | 原来是项目成员的工作区管理员 |
    | 20 | 5 | 5 | 纯函数的一格；经接口，工作区访客加入是 404 |
  - 添加恢复时取请求里的角色（管理员这一次的决定）；接受邀请恢复为访客时 `DemoteToGuest` 被调用。
- **保留名单**：三段都能解析、互不重复、都符合 slug 的规则；"预留"一段恰好是 3.10 声明的四个。

### 9.2 权限矩阵（逐格）
**测法**：`server/internal/bootstrap/permission_matrix_test.go`，在真实的组合和真实的 Postgres 上经 HTTP 调用，每格断言状态码和错误码。
- **准备一次数据**（Go 测试，用各模块的仓储写入，不经 e2e 的规则，所以每个 Phase 都能加它的行）：一个工作区，一个公开项目、一个私密项目，各有状态、标签、成员、邀请；第二个已删除的工作区和它的项目；下表的每种身份一个账户（工作区级的列复用项目级的账户），共 11 个。准备好的库作为模板。
- **读的格子**共用这个库；**写的格子**各自从模板复制一个库（`pgtest` 加一个"从已准备的库复制"的帮助函数），互不影响。约 400 格，其中写的约 250 格。评审的 spike：复制一次约 83 毫秒，16 路并行 205 毫秒；250 格复制约 3–5 秒，整个矩阵预计 20–30 秒。
- 表写在测试里，一行一个操作、一列一种身份。测试另外核对：接口描述里除了账户级和公开的操作，每个操作都有一行（新操作不加一行就失败）。

**工作区级**（✓ 成功；403 是平台码 `forbidden`，另注的除外；404 是该资源的 not_found 码）：

| 操作 | 管理员 | 成员 | 访客 | 从来不是成员 | 已被移出 | 工作区已删除 |
|---|---|---|---|---|---|---|
| `getWorkspace` | ✓ | ✓ | ✓ | 404 | 404 | 404 |
| `updateWorkspace`、`deleteWorkspace` | ✓ | 403 | 403 | 404 | 404 | 404 |
| `listWorkspaceMembers` | ✓ | ✓ | ✓，`email` 都为 `null` | 404 | 404 | 404 |
| `updateWorkspaceMember`、`removeWorkspaceMember`（目标是另一位成员） | ✓ | 403 | 403 | 404 | 404 | 404 |
| 同上，目标是自己 | 409 `workspace.own_membership` | 403 | 403 | 404 | 404 | 404 |
| `leaveWorkspace` | ✓（另有管理员时；唯一的 409） | ✓ | ✓ | 404 | 404 | 404 |
| `listWorkspaceInvitations`、`createWorkspaceInvitations`、`updateWorkspaceInvitation`、`deleteWorkspaceInvitation` | ✓ | 403 | 403 | 404 | 404 | 404 |
| `getWorkspacePreferences`、`updateWorkspacePreferences` | ✓ | ✓ | ✓ | 404 | 404 | 404 |
| `listProjects` | ✓ 全部 | ✓ 加入的和公开的 | ✓ 加入的 | 404 | 404 | 404 |
| `createProject`、`checkProjectIdentifier` | ✓ | ✓ | 403 | 404 | 404 | 404 |
| `listWorkspaceStates` | ✓ 他加入的未归档项目的 | ✓ 同左 | ✓ 同左 | 404 | 404 | 404 |

- 邀请的四行按决策点 4（只有管理员）。

**账户级**（不经 `Authorizer`，6.4）：`listWorkspaces`、`checkWorkspaceSlug`、`createWorkspace`（开关关闭时 403 `workspace.creation_disabled`）、`acceptWorkspaceInvitation`、`declineWorkspaceInvitation`（邮箱一致 ✓，不一致 403 `workspace.invitation_email_mismatch`）：任何有效凭证；不带令牌 401。`getWorkspaceInvitation`：公开，令牌对 200，否则 404。

**项目级**（列：PA 项目管理员；PM 项目成员；PG 项目访客；PM+WA 项目成员而同时是工作区管理员；WA- 工作区管理员、不是项目成员；WM-公、WM-私 工作区成员、不是项目成员，项目公开或私密；WG- 工作区访客、不是项目成员（公开、私密的项目答案相同）；P-前 以前是私密项目的成员、已离开（`is_active = false`），仍是工作区成员；X 不是有效的工作区成员（从来不是、已被移出、工作区已删除三个账户，答案相同）。PA、PM 的工作区角色是成员，PG 的是访客）：

| 操作 | PA | PM | PG | PM+WA | WA- | WM-公 | WM-私 | WG- | P-前 | X |
|---|---|---|---|---|---|---|---|---|---|---|
| `getProject` | ✓ | ✓ | ✓ | ✓ | ✓ `member_role: null` | ✓ `member_role: null` | 404 | 404 | 404 | 404 |
| `updateProject`、`deleteProject`、`archiveProject`、`unarchiveProject` | ✓ | 403 | 403 | ✓ | 403 | 403 | 404 | 404 | 404 | 404 |
| `joinProject` | ✓（不变） | ✓（不变） | 403 | ✓（不变） | ✓ 角色 20 | ✓ 角色 15 | 404 | 404 | 404 | 404 |
| `leaveProject` | ✓（另有管理员时；唯一的 409） | ✓ | ✓ | ✓ | 403 | 403 | 404 | 404 | 404 | 404 |
| `listProjectMembers` | ✓ | ✓ | ✓ | ✓ | 403 | 403 | 404 | 404 | 404 | 404 |
| `addProjectMembers` | ✓ | 403 | 403 | ✓ | 403 | 403 | 404 | 404 | 404 | 404 |
| `updateProjectMember`、`removeProjectMember`（目标：项目成员） | ✓ | 403 | 403 | ✓ | 403 | 403 | 404 | 404 | 404 | 404 |
| `getProjectPreferences`、`updateProjectPreferences` | ✓ | ✓ | ✓ | ✓ | 403 | 403 | 404 | 404 | 404 | 404 |
| `listStates`、`listLabels` | ✓ | ✓ | ✓ | ✓ | 403 | 403 | 404 | 404 | 404 | 404 |
| `createState`、`updateState`、`deleteState`、`markDefaultState` | ✓ | 403 | 403 | ✓ | 403 | 403 | 404 | 404 | 404 | 404 |
| `createLabel`、`updateLabel`、`deleteLabel` | ✓ | 403 | 403 | ✓ | 403 | 403 | 404 | 404 | 404 | 404 |

- `addProjectMembers` 的目标无效（不是工作区的有效成员，或把工作区访客加为成员）时，PM 那一格仍是 403、X 那一格仍是 404，只有能做的人得到 422：目标行先锁只为取锁的顺序，目标的 422 在判定之后给出（3.6 约定三）。矩阵为这两格各加一个无效目标的用例。
- `updateProjectMember` 的规则行是项目管理员，加上同时是工作区管理员的项目成员（Plane `views/project/member.py:234-238` 先拒绝不是管理员的人），所以 PM、PG 得到 `forbidden`；`project.role_too_high` 只出现在管理员之间的相对规则里（3.5，单元测试和故事 P5 覆盖）。
- P-前 这一列就是 3.4 表中"列出标签"的情形：Plane 给他 200，Nerve 答 404。以前是公开项目成员的人与 WM-公 相同（看得到、不是成员）。
- **已归档的项目**：另一张小表。PA 对已归档项目：`getProject` ✓（带 `archived_at`）、`updateProject` 409 `project.archived`、`unarchiveProject` ✓、`listStates` ✓ 空列表（3.17）、`listWorkspaceStates` 不含它、`listLabels` 和状态、标签的写入 ✓（3.19）；`listProjects` 默认不含，`?archived=true` 只含。
- **决策点 3 的一格**：访客看别人的个人主页。按裁定 A，页面只显示用户卡片（数据来自 `listWorkspaceMembers`，上表已覆盖），工作项区域"没有权限"；工作项列表的接口在 M4，它在规则表加这一行时按裁定写（13.2）。页面一侧由 P9 故事覆盖。

### 9.3 集成测试（`pgtest`）
- **仓储**：每条查询；CHECK 的反例（4.2–4.10 的 spike 逐条变成测试，含 4.6 `logo_props` 的全部反例：Codex S5 的 `{"unexpected": true}`、`{"in_use": 17, "emoji": []}` 等十个值得到 `check_violation`，四个合法的值通过，P4）；唯一约束冲突翻译为 409 的码，一个例外：邀请的 `workspace_member_invites_workspace_id_email_key` 翻译为 422 `invitations[i].email` 的 `duplicate`，`i` 是冲突的邮箱在请求里的下标，与事先校验的回答相同（3.8，P3）；审计列等于固定时钟；部分唯一索引下删除之后可以重用（slug、标识、名称、邀请的邮箱）；`ON CONFLICT … WHERE` 的显示设置只有一行；锁父行的语句带 `deleted_at IS NULL`，父行已软删除时读到 0 行。
- **连带**：删除工作区、删除项目、删除父标签，被连带的行的 `deleted_at` 与父行相同（同一时刻）；`ProjectCascade` 失败时工作区一侧的写入也回滚。
- **可见性一致**：对 9.2 的每种身份，`listProjects` 的结果等于对每个项目逐个判定 `project.read` 的结果；`listWorkspaceStates` 同理（3.4）。
- **邀请从不改变有效的成员关系**（3.8，P5）：Codex S2 的顺序：A、B 是管理员，A 移出 B，之后给 B 发一份访客邀请，`reactivate-member` 恢复 B，B 加入一个项目，A 离开，B 接受那份旧邀请：B 仍是工作区管理员（工作区有一位管理员），项目角色仍是 20，邀请已消费。故事 W12 的接口版本走同一顺序。
- **结束的成员关系不留下邀请**（3.8，P5）：复核 spike 9d 的顺序：A、B 是管理员，B 有效而有一份发给他的待接受访客邀请（上一条里 `reactivate-member` 之后的状态），A 移出 B；之后 B 用旧链接查看、接受都得到 404 `workspace.invitation_not_found`，他没有回到工作区，邀请的 `deleted_at` 等于移出的时刻。B 自己离开同样跑一次。一份已忽略的邀请不受影响。`EndMemberships` 以 `project.sole_admin` 拒绝时，邀请随之回滚、仍待接受。
- **交错**（3.6 的 19 种，用 `pgtest.WaitForLockWaitOn(t, pool, <被等的父行所在的表>, 时限)` 让一方确定地等在那张表的行锁上，例如 `workspaces`；组装好的应用在同一个库上跑 River，不看表的 `pgtest.WaitForLockWait` 会被别的等待提前满足，它只用在没有别的语句能等的库上，如 `ON CONFLICT` 等同一个键的插入；括号里是加入的 Phase）：
  1. 两位管理员同时离开工作区（P5）：一个成功，另一个等锁之后 409 `workspace.sole_admin`；项目一侧（`leaveProject`）同理。
  2. 两位管理员互相降级（P2）：先拿到工作区锁的一方成功；后到的一方判定时已是成员，403 `forbidden`；仍有一位管理员。
  3. 接受邀请与删除工作区（P3）：接受先拿到工作区锁，删除等待，之后连带删除新成员；删除先提交，接受锁工作区时读到 0 行，404。
  4. 添加项目成员与移出工作区成员（P5）：添加先锁住目标的工作区成员行，移出等待，之后连带停用新的项目成员关系；移出先提交，添加得到 422（`members[i].member_id`，`not_allowed`）。
  5. 移出成员与他正在建项目（P5）：建项目先锁住工作区（`FOR SHARE`），移出等待，之后连带停用他在新项目里的成员关系；移出先提交，建项目判定时他已看不到工作区，404。
  6. 移出管理员与他正在删除工作区（P5）：删除先拿到锁，移出等待，之后锁工作区时读到 0 行，404；移出先提交，删除判定时他已不是成员，404。
  7. 停用与接受邀请（P6；Codex S1，B 原来不在工作区 W，A 是 W 的管理员，W 有项目 P）：
     - (a) 接受先锁住 B 的账户行，停用等待；接受提交之后停用的列举含 W，B 在 W 的成员关系被停用。
     - (b) 在 (a) 的接受提交之后、停用拿到 W 的锁之前，B 的另一个事务锁住 W、把 A 改为成员（S1 的第 5 步）：停用等 W 的锁，之后 B 是 W 唯一的管理员而 W 还有别的成员，409 `workspace.sole_admin`，数据库不变。
     - (c) 同样的位置，B 的另一个事务锁住自己的工作区成员行、加入 P（S1 的第 6 步）：停用改成员行时等它，之后 `EndMemberships` 的列举含 P，B 在 P 的成员关系被停用；W 仍有管理员 A。
     - (d) 停用先锁住账户行：接受等待，之后在锁下读到账户已停用，401，没有新的成员关系。
     - (e) B 在 W 有以前的（无效的）成员行，(a) 和 (d) 各再跑一次：接受恢复那一行而不是插入，结果相同（恢复与插入一样串行在账户行上）。
  8. 停用与创建工作区（P6）：创建先锁住账户行，停用等待，之后停用新的成员关系（他是那里唯一的成员，允许）；停用先提交，创建被拒绝（接口 401，命令行退出码 1），没有新工作区。
  9. 创建邀请与管理员重置密码（P3）：重置先提交，创建得到 401，没有新邀请（M2 的交错测试 1 的同一写法）。
  10. 设为默认与删除状态（P7）：先设为默认，删除得到 409 `project.state_default`；先删除，设为默认得到 404；两种顺序都恰好一个默认状态。
  11. 两次改父标签（P7）：后到的一方看到新的层级，422（`parent_id`，`not_allowed`），没有环。
  12. 接受邀请与改邮箱（P3）：改邮箱先提交（它取 `FOR UPDATE`），接受在锁下读到新邮箱，403 `workspace.invitation_email_mismatch`，数据库不变；接受先锁住账户行，改邮箱等待，接受按原邮箱成功，之后改邮箱照常完成。
  13. 停用与把他加为项目成员（P6）：添加先锁住他的工作区成员行，停用改成员行时等它，之后列举含新的项目成员关系并停用它；停用先改了成员行，添加等锁之后读到无效，422（`members[i].member_id`，`not_allowed`）。他在这个项目有以前的项目成员行时两种顺序各再跑一次（添加恢复那一行），结果相同。
  14. 停用与他加入项目（P6）：同上；停用在先时加入得到 404。同样另跑一次以前的项目成员行（加入恢复它）。
  15. 停用与以他为负责人建项目（P6）：建项目先锁住工作区（`FOR SHARE`），停用等工作区的锁，之后列举含新项目并停用他在那里的成员关系（创建者也是管理员，他不是唯一的）；停用先提交，建项目锁他的成员行时读到无效，422（`project_lead_id`，`not_allowed`）。
  16. 停用与恢复他的成员关系（P6）：`reactivate-member` 先锁住账户行，停用等待，之后列举含恢复的工作区并再次停用它；停用先提交，恢复照样进行（3.11 的例外），输出提示 `nerve users activate`。
  17. 降为访客与他加入项目、与把他加为项目成员（P4；B 是工作区成员）：加入（添加）先锁住 B 的工作区成员行，降为访客改这一行时等它，之后 `DemoteToGuest` 的列举（改成员行之后的新语句）含新的项目成员行，项目角色是 5；降为访客先提交，加入等锁之后读到 B 是访客，404（访客看不到没加入的项目），以成员角色添加他 422（`members[i].role`，`not_allowed`）。加入、添加各跑两种顺序。
  18. 两位管理员创建重叠的两批邀请（P3）：A 请求 `[x, y]`、C 请求 `[y, x]`，都持工作区 S，各自按排序插入；C 等在唯一索引上，A 提交之后 C 得到 422 `invitations[1].email` 的 `duplicate`（`x` 在 C 的请求里的下标），C 的整批都没有插入；没有 40P01。另跑一次单个邮箱的竞争（两批都只有 `x`）：后到的一方 422，下标 0。
  19. 忽略邀请与停用（P3；B 是工作区 W 的有效成员，还有一份发给他的、W 的待接受邀请，即 `reactivate-member` 或改邮箱之后的状态）：忽略先锁住 B 的账户行，停用等待，之后照常停用；停用先锁住账户行，忽略等待，之后在锁下读到账户已停用，401。两种顺序都没有 40P01。P6 之前停用还不锁工作区，停用一方由测试的会话按 3.9 的顺序取锁（账户行 N，再 W 的 N）；P6 换成真实的停用再跑一次。
- **停用的两条路**（M2 交接第 6 节，P6）：`deactivateMe` 和 `nerve users deactivate` 各测：唯一管理员时拒绝、数据库完全不变；成功时成员关系、项目成员关系、邀请（含已忽略的）都在同一事务里改好；跨两个工作区时项目按 `id` 升序加锁。另外两种情形（3.9）：
  - 复核 spike 15：列举到的一个工作区在停用锁住它之前被删除：停用跳过它，照常完成，不答 404；
  - 改邮箱先持账户行，停用等在账户行上，改邮箱提交之后：停用删除发给新邮箱的邀请，发给旧邮箱的不动（邮箱取自锁下的账户行）。

### 9.4 契约测试和整程序测试
- `apitest`：
  - 新错误码两个方向都核对（每个返回的码都声明了，每个声明的码都有测试返回过，M2 设计 3.11）；
  - **规则：声明在模块 X 的操作上的码，在 X 自己的 HTTP 测试包里经 `CheckResponse` 返回过，即使产生它的是别的模块**。`apitest` 记下的"返回过的码"只在一个测试进程之内（`apitest/problems.go:92-106`），每个 HTTP 测试包的 `TestMain` 调 `apitest.Main` 只核对自己的进程；`bootstrap` 是另一个测试进程，它返回过的码补不了 X 的记录（Codex S7）。所以 X 的 HTTP 测试用它已有的假实现让用例返回那个错误，经 handler 透传：
    - `identity`：`deactivateMe` 的 `workspace.sole_admin`、`project.sole_admin`，用 `fakeDeactivate.err`（`identity/adapter/http/me_test.go:43`），P6；
    - `workspace`：`removeWorkspaceMember`、`leaveWorkspace` 的 `project.sole_admin`，P5；
    - `project`：各操作的 `workspace.not_found`，P4。
  - `bootstrap` 的整程序测试另外在真实的组合上返回这些码（产生它的模块确实产生它），不代替上面的测试；
  - `platformCodes` 有 `forbidden`；带前缀的码，前缀必须是 `api/modules/` 下的一个模块，一个反例（不存在的前缀）核对规则会失败（11.7）；
  - `security` 的写法（只有 `getWorkspaceInvitation` 公开）；
  - 请求体用例扩展到对象的数组：数组元素里的未知字段、缺少的必填字段 400（5.2）。
- `bootstrap`：
  - 四个整程序测试覆盖全部新操作：默认拒绝、请求体结构（含 `slug`、`workspace_id` 等不可写的字段 400）、参数绑定（路径的 uuid、`archived` 的布尔、`token` 的必填）、错误码与状态；
  - 操作名的完整性（9.1）；
  - 保留名单"服务端"一段等于组合根在前端页面之外注册的顶层路径（3.10）。
- `archtest`：命令行的组合测试覆盖 `Users` 和 `Workspaces`（6.6）；`TestSQLCSchemaScope` 的四个反例（4.1）。

### 9.5 前端单元测试（vitest）
- 落点函数（3.14）的三种情况（P8）。
- 权限 store：`Workspace.role`、`Project.member_role` 到页面权限的对应（与 9.2 同一组身份，P8）。
- state store 的顺序和分组中的位置（取代 `order`，P8）。
- `RootStore.dispose()`：换代之后改路由，旧一代的反应不再运行（7.1，P8）。
- 一个资源的修改一个接一个：连续两次拖动排序，第二个请求在第一个有了应答之后才发出（7.1，P8）。
- `sessionGuard()`：`loginId` 不变时 `inSession()` 为真，换了之后为假（7.1，P8）。
- 权限决定取数：成员页的邀请列表只在工作区管理员时启用取数；`ProjectAuthWrapper` 在 `member_role` 为 `null` 时不启用子资源的取数（7.1，P8）。
- 删除工作区、删除项目的组件：修改的 Promise 在 `loginId` 改变之后才兑现，组件不跳转、不提示（7.1，P9、P10）。
- 保留名单"应用"一段等于 `routes/core.ts` 的顶层静态路由段加 `public/` 的顶层目录（3.10，P8）。
- 新手引导：已有工作区的人资料一步之后完成（7.4，P9）。
- 邀请链接的拼法（`id`、`token` 编码，P9）。
- 离开项目：接口失败时不跳转（7.6，P10）。
- 表情选择器的数据地址和版本（7.7，P10）。

### 9.6 端到端
- **fixture**：`e2e/fixtures/api.ts` 加建工作区（P1）、邀请并接受（P3）、建项目和加项目成员（P4）、建状态和标签（P7）的帮助函数（生成的客户端，PAT）；`e2e/fixtures/assert/workspace.ts`、`assert/project.ts` 按表的断言函数，页面版本和接口版本共用；`e2e/fixtures/db.ts` 加九张表的读取。
- **故事**：第 2 节的 21 个；S1、S2 的更新。每个故事的接口版本和页面版本在第 2 节标出的 Phase 加入。
- **会话切换**（W3 的页面版本，P9）：在 `page.route` 里先经 `route.fetch()` 把删除工作区的请求发到 nerve，拿到成功的回答（此时会话还是原来的）；在同一个浏览器上下文的另一个标签页退出并以另一个账户登录之后，才经 `route.fulfill()` 把这个回答交给页面；断言原标签页没有跳转、没有提示（7.1）。
  - 不能在切换之后才放行请求：退出撤销了原会话，放行的请求得到 401，页面因为请求失败而不跳转，测试不依赖 `inSession()` 就会通过（复核 M6）。
  - 变异核对：去掉组件里 `inSession()` 的核对，这个测试必须失败（P9 第 11 个任务写进 review）。
  - 这是 C5 的自动化部分，C5 仍由控制者在浏览器里看一遍。
- **注册关闭、创建关闭的独立 nerve**（W1、W6）：沿用 M2 为 A13 起的第二个 nerve 的写法。它以 `NERVE_ENV=test` 运行、没有配置密钥文件（`e2e/fixtures/server.ts:100-105`），签名密钥是它自己的临时密钥，主 nerve 算出的令牌在它那里无效。所以 W6 在第二个 nerve 上经它的接口建工作区、发邀请，链接由它算出。部署时的要求（全部进程共用一个密钥文件）写在 README（8.7）。

### 9.7 控制者的浏览器核对
| 编号 | 内容 | Phase |
|---|---|---|
| C1 | 落点：两个工作区、删掉上次的、提升第二位管理员后离开；切换工作区后刷新；没有失败的请求（W2） | P9 |
| C2 | 邀请链接：未登录、已登录、邮箱不一致（页面不显示任何邮箱）、已忽略、令牌改一位；注册关闭时注册再接受，新手引导只有资料一步；中文界面下的文案（W5、W6） | P9 |
| C3 | 成员页：管理员、成员、访客三种身份（邮箱、角色下拉、邀请列表只对管理员、已忽略的邀请、离开） | P9 |
| C4 | M2 交接第 13 节：`WorkspaceAuthWrapper` 的退出按钮（Tab、Enter、空格）；工作区和项目加载中的显示；`ProfileSidebar` 宽屏折叠后拉过 768 像素；`WorkspaceLogo` 在工作区菜单和卡片中的盒子；新手引导邀请成员一步和导览在中文下的文案。"加入工作区"一步随决策点 2 删除，这一条写明删除 | P9 |
| C5 | 两个账户两个标签页：在工作区页上切换账户，旧的一代不再写入，页面以新账户重新渲染（7.1） | P9 |
| C6 | 项目：创建（表情和图标选择器，Network 面板没有 `cdn.jsdelivr.net`，控制台没有 CSP 违规）、加入、离开（接口失败时停在原页）、归档、恢复、删除；members 页设负责人、默认负责人 | P10 |
| C7 | 下拉框：`CustomSearchSelect` 用 Tab 到达、键盘打开；`member-options` 的列表在按钮旁；9 个 `CustomSelect` 调用方、4 个 `CustomSearchSelect` 调用方逐个 | P10 |
| C8 | 复制到剪贴板：在局域网 HTTP 地址上复制项目链接、邀请链接，失败时有提示 | P10 |
| C9 | 状态和标签的拖动、设为默认、两层标签的拖放；默认状态和组内唯一状态的删除按钮不可用；访客看状态设置"没有权限" | P11 |
| C10 | 访客打开别人的个人主页（决策点 3，A） | P11 |

---

## 10. 负责人决策点

下面四项改变用户能做什么或谁能看到什么，而 Plane 和上级文档都没有定下（或者 Plane 的服务端和页面不一致）。四项都已由负责人裁定（2026-09-29），都选了建议的选项；选项和代价留在下面备查，正文各节都已按裁定写定。

下面几件按 Plane 已有的规则裁定，没有交负责人：
- **创建工作区的命令**（3.11）：Plane 关闭创建时由实例管理员在管理后台为自己建（`license/api/views/workspace.py:71-99`），Nerve 没有实例管理员账户，职能由命令行承担（M2 已这样处理用户），`--admin-email` 指定管理员。这是翻译，登记为差异（4.11）。
- **恢复成员的命令**（3.11）：照 Plane 的 `reactivate_workspace_member`。
- **保留名单里的 Plane 产品词**（3.10）：保留名单的作用是不让工作区名与本站的路径冲突（Plane 同一用途），产品词都不是 Nerve 的路径，全部去掉；另外预留四个以后可能用到的段。
- **已是有效成员时接受邀请**（3.8）：第二稿漏掉的状态转换（Codex I-2）。它不改变用户能做的事，只堵住一条绕过唯一管理员保证的路径，由控制者裁定：邀请从不改变有效的成员关系。

### 决策点 1：接受工作区邀请凭什么；注册关闭时持邀请的人能否注册（M2 交接第 1 节）
**负责人裁定（2026-09-29）：A+(a)，公开的查看和邮箱不一致的拒绝都不显示被邀请的邮箱。** 落在 3.8、5.2、7.4、8.1、8.2、8.4、W5、W6。

**背景**：
- v0 没有邮件，邮箱没有经过验证。注册在 prod 默认关闭（M2 决策点 2），所以邀请是新人进入工作区的主要途径，受邀的人通常还没有账户。
- **链接的保密是主要的控制**。邮箱一致只保护已有账户的受邀者：对还没有账户的人，拿到链接、又知道被邀请的邮箱的人，可以先用那个邮箱注册，再接受。第一稿的公开查看显示被邀请的邮箱，所以只要拿到链接就够了（评审 I10）。

| 选项 | 对用户 | 能防住什么 | 对代码 |
|---|---|---|---|
| A+(a) 照第一稿 | 受邀的人用链接和被邀请的邮箱接受；注册关闭时，持有效邀请、注册邮箱与邀请一致的人可以注册；查看显示被邀请的邮箱 | 已有账户的受邀者的链接被别人拿到，别人用自己的账户接受不了。受邀的人还没有账户时，拿到链接就能冒用 | 查看带 `email` |
| **A+(a)，查看不显示邮箱（已选）** | 同上；公开的查看只说"这份邀请发给一个指定的邮箱"，邮箱不一致的拒绝也不说是哪一个。用错账户打开链接的人要问管理员该用哪个邮箱 | 同上，另外拿到链接的人还要另外知道被邀请的邮箱才能冒用。转发链接的消息里常常就有那个邮箱，所以这只是缩小，不是关上 | 3.8、5.2 |
| A+(b) 注册关闭时不能凭邀请注册 | 管理员先在服务器上用 `nerve users create` 替对方建账户、设初始密码，把密码另外交给对方（v0 没有邮件，同样靠带外传递），再发链接；对方登录后改密码 | 关上冒用：链接只能被那个已有的账户接受。代价是每个新成员都要服务器管理员操作，多一个要传递的秘密（初始密码），受邀的人不能自助 | 去掉 `SignupPolicy` 的邀请分支和注册页的邀请参数 |
| B 只凭令牌 | 拿到链接的任何账户都能接受 | 对新受邀的人与 A+(a) 几乎相同；对已有账户的受邀者更弱 | 去掉邮箱比对，邀请的邮箱只是给管理员看的标签 |

- **理由**：保留自助（管理员只发链接）；链接相当于 Plane 的邀请邮件；不显示邮箱让单独泄露的链接不够用。剩下的风险（链接和邮箱一起泄露、而受邀的人还没注册）由管理员删除邀请、重新邀请来处理（8.1）。Plane 在 GHSA-4vj8-p63v-8p24 之后也要求邮箱一致（`views/workspace/invite.py:161-174`）。
- **(a) 带来的另一个口子**（Codex 4.1）：持有链接的人能拿注册去试被邀请的邮箱，对了就注册成功、错了被拒绝。每次尝试受按 IP 的注册限流约束（`ratelimit.register_ip`，每分钟 10 次、最多攒 5 次）；查看不显示邮箱只是少了直接的披露，并没有让邮箱变成第二个验证过的因素（3.8、8.2）。
- **以后改的代价**：各选项之间的代码改动都小。显示或不显示邮箱是查看的一个字段和拒绝的一句话；A → B 放宽安全边界，要重新评估。(a) → (b) 删掉 `SignupPolicy` 的邀请分支和注册页的邀请参数，但已经凭邀请注册的账户不会因此消失：改的时候要另外决定怎样对待这些存量账户（逐个核对，或者不处理）。

### 决策点 2：保留一条接受的路径还是两条（M1 设计 3.15）
**负责人裁定（2026-09-29）：A，只有链接。** 落在 3.8、7.4、7.8、7.10。

**背景**：M1 保留了两条路：系统内接受（`/invitations` 页、新手引导的"加入工作区"一步，按邮箱列出发给自己的邀请）和凭链接接受（`/workspace-invitations`、成员页的"复制链接"）。用不到的一条删除。

| 选项 | 对用户 | 对代码 |
|---|---|---|
| **A 只有链接（已选）** | 管理员邀请之后把链接发给对方（v0 没有邮件，本来就要这样）；受邀的人打开链接接受 | 删除 `/invitations` 页、新手引导的"加入工作区"一步、按邮箱列出和批量接受；接口少两个操作 |
| B 两条都留 | 登录后还能在 `/invitations` 看到发给自己邮箱的邀请 | 系统内的列表只能凭邮箱匹配。按决策点 1，接受仍要令牌，而列表拿不到令牌（给了就等于只凭邮箱）；列表还会把工作区名告诉任何先注册了这个邮箱的人 |
| C 只有系统内 | 不需要复制链接 | 只凭邮箱接受，与决策点 1 的各选项都不相容 |

- **理由**：v0 没有邮件，"系统内"的一条只能靠邮箱匹配，正是决策点 1 要排除的；留着它只能做成一个"你有一份邀请，请向管理员要链接"的提示，价值很小。
- **以后改的代价**：有了邮件（v0 之后）再加回系统内的列表：一个 `GET /me/workspace-invitations`（按已验证的邮箱）和 Plane 的 `/invitations` 页（从 Plane 的源码取回），中等。那时的列表也不返回接受的凭证。

### 决策点 3：访客能否看别人的个人主页（M1-P2 交接）
**负责人裁定（2026-09-29）：A，照 Plane 的页面。** 落在 9.2、故事 P9、C10；工作项列表的规则行交给 M4（13.2）。

**背景**：个人主页是用户卡片加"分配给他的、他创建的、他关注的"工作项（M1-P2）。Plane 的页面只让工作区管理员和成员看工作项部分，访客看到"没有权限"，自己的主页也一样（`app/(all)/[workspaceSlug]/(projects)/profile/[userId]/layout.tsx` 的 `isAuthorized`）；Plane 的服务端却让任何有效成员取（`views/workspace/user.py:98-99` 的 `WorkspaceViewerPermission`）。

| 选项 | 对用户 | 对代码 |
|---|---|---|
| **A 照 Plane 的页面（已选）** | 访客能看到任何成员的卡片（名字、头像），看不到任何人（包括自己）主页上的工作项 | 页面不变；M4 的规则表中这一行只允许工作区管理员和成员 |
| B 访客只能看自己的 | 访客在自己的主页上看到自己的工作项 | 页面按"是不是自己"分支；M4 的规则加"本人"的条件，查询仍要按他能看到的项目过滤 |
| C 与成员相同 | 访客看别人主页上的工作项，结果按他能看到的项目过滤 | 页面去掉判断；M4 的查询按访客的可见性过滤（还要受 `guest_view_all_features` 约束） |

- **理由**：与 3.4 的"服务端比页面宽松时按页面收紧"一致；访客在 Plane 里是受限的外部协作者，他能看到的工作项已经由项目内的规则约束，个人主页是另一个汇总入口，不开放最保守。
- **以后改的代价**：页面是一个判断、规则表是一行，但这不是全部（Codex 4.1）。改成 B 或 C 时，M4 个人主页的工作项查询必须按访客能看到的项目过滤，并受 `guest_view_all_features` 约束，还要为访客的每种项目成员关系补测试；不能只放开入口、拿入口代替数据的授权。数据不受影响。

### 决策点 4：谁能邀请成员
**负责人裁定（2026-09-29）：A，只有工作区管理员。** 落在 3.4、3.8、7.5、9.2。

**背景**：Plane 的服务端让工作区管理员和成员都能列出、创建、修改、删除邀请（`WorkSpaceAdminPermission` 允许两者，`permissions/workspace.py:61-71`），创建时邀请的角色不能高于邀请人（`views/workspace/invite.py:61-66`），修改时不检查（缺陷：成员能把自己发的邀请改成管理员）。Plane 的页面只对管理员显示邀请。

| 选项 | 对用户 | 对代码 |
|---|---|---|
| **A 只有管理员，照 Plane 的页面（已选）** | 成员要邀请人时找管理员；页面与 Plane 相同 | 规则表的四行只允许管理员；角色的上限自然成立 |
| B 管理员和成员，照 Plane 的服务端 | 成员能邀请成员或访客（不高于自己） | 规则表允许两种角色；创建和修改都检查上限；邀请的读取范围另要限定（见下）；成员页给成员显示邀请的界面（Plane 的页面没有，要新写） |

- **理由**：凭链接接受（决策点 2）之后，邀请是进入工作区的唯一途径，发出一份邀请等于决定让谁进来，这是管理员的事；Plane 的用户在页面上看到的本来就是这样。
- **以后改的代价**（Codex I-3 的条件）：不只是规则表的四行。一份邀请带着可用的令牌（`WorkspaceInvitation.token`），而管理员可能已经邀请了一个还没注册的邮箱做管理员；成员只要能列出这份邀请，就拿到了邮箱和令牌，按决策点 1 注册、接受，得到一个管理员账户，全程不创建、不修改邀请，角色的上限挡不住。所以若以后让成员邀请，列表、单个邀请的回答和复制链接都必须排除角色高于读者的邀请（或者至少不给这些邀请的令牌），成员能改、删哪些邀请也要一起定，并按"成员读不到管理员邀请的令牌"写测试。Plane 的读取范围不提供这层保护（`views/workspace/invite.py:39-49` 只按工作区过滤），不能照搬。

---

## 11. 架构问题

以下七条改变总体设计或 M2 设计的架构约定，都已由负责人批准（2026-09-29）。每条留着问题、做法和另一种做法备查；正文各节都已按此写定，上级文档按 3.20 的 Phase 同步。
- 第一稿的"领域事件的形状"不再是架构问题：M3 不引入事件，连带经端口完成，事件的写法仍按 M2 设计 3.15 由 M4 定（3.3、13.2）。

### 11.1 邀请令牌由签名密钥派生（3.8、6.6；负责人已批准，2026-09-29）
- **问题**：邀请链接里的令牌要能随时重新算出（"复制链接"随时可用），又不能存在数据库里。
- **做法**：令牌是签名私钥按用途派生出的 MAC 算出的，`identity` 为此对别的模块提供"按用途派生的 MAC"，签名密钥的载入从 `identity.New` 移到 `bootstrap` 先做（`identity.LoadKeys`）。它与 M2 的刷新令牌的 MAC 是同一做法。
  - 同一部署的全部 nerve 进程必须用同一个密钥文件（JWT 本来就这样要求），README 写明。
  - 让一个泄露的链接失效：删除这份邀请，重新邀请。换签名密钥让全部待接受的链接失效，管理员重新复制即可。
- **另一种做法**：`workspace` 自己的密钥（新配置 `workspace.invitation_key_file`，prod 必填）。它的好处是能与签名密钥分开轮换，一把密钥出事时影响的范围更小（邀请链接和登录互不牵连），换 JWT 密钥时链接也不失效（Codex 4.2）；代价是多一份要部署、备份、轮换的秘密。v0 取派生的做法：派生密钥之间已经按用途分开，一个派生密钥泄露不影响签名和别的用途；需要单独轮换时再加这份密钥。
- 同一件事的另一部分：创建邀请"按账户行锁"（M2 交接第 1 节）落为 `identity` 经端口提供的 `CallerLock`，在 `workspace` 的事务里第一个调用。

### 11.2 加锁顺序的延伸和六条约定（3.6；负责人已批准，2026-09-29）
- **问题**：M2 设计 3.5 的加锁顺序是全局约定。M3 把九张表接到它后面，并定下以后各 M 照做的写法。
- **做法**：
  - 顺序：`users` → … → `workspaces` → 邀请 → 工作区成员 → 工作区显示设置 → `projects` → 项目成员 → 项目显示设置 → `states` → `labels`。
  - **写事务先锁父行（工作区行或项目行），再判定权限**。成员关系和角色的改变、状态和标签的写入、修改父行本身的写用 `FOR NO KEY UPDATE`，其余的写用 `FOR SHARE`；按 id 寻址的写先读资源、锁父行、再重读。评审的 spike 证明先判定、后锁时两位管理员能互相降级到一个不剩。
  - 让某人成为项目成员的写先锁他的工作区成员行；批量语句按扫描顺序加锁（在父行的锁之下，安全）。
  - 第三稿的聚焦复核在约定之内加了三处收紧，顺序和六条约定不变（17.3）：目标的 422 在判定之后给出（约定三）；只持父行的共享锁而批量插入时按唯一键的顺序插入，因为全局顺序只排除行锁的环，排除不了唯一索引上的互等（约定五）；`Accounts` 只作事务的第一把锁，事务中途读邮箱经不加锁的端口（约定一）。
  - **成员关系集合的增长与收缩**（约定六，第三稿加入）：让一个账户得到工作区成员关系的写最先以 `FOR SHARE` 锁住他的账户行、在锁下重读 `is_active`；收缩先改工作区成员行、再在调用时列举项目；停用在锁住账户行之后才列举。锁的顺序正确不等于锁的集合完整：Codex 的 S1 证明，没有这一条时停用列举之后他的工作区集合还能变大，新工作区漏掉了锁、检查和连带（Codex 4.2）。
  - 放宽 M2 的"锁住后三张表之后不再插入引用另一个账户的行"：M3 的锁之后可以插入（添加项目成员、以别人为负责人建项目）。理由：唯一与外键检查冲突的锁是改邮箱的 `FOR UPDATE`，而它不取 M3 的任何锁。
- **另一种做法**：判定时以 `FOR SHARE` 读调用者自己的成员行，不锁父行。互相降级时双方各持自己那一行、又要改对方的行，死锁；而且约定要按操作逐个推敲，难以核对。
- 总体设计 4.2 在 P2 按此更新（3.20）。

### 11.3 集合型的列表不分页；成员内嵌用户的公开资料（3.12、5.2；负责人已批准，2026-09-29）
- **问题**：总体设计 3.4 写"列表接口统一分页"，3.6 写"关联对象只返回 ID"。M3 的列表一次返回全部，工作区成员内嵌 `MemberUser`。
- **做法**：3.4 补"集合型的列表（成员、项目、状态、标签等由管理员的操作决定大小的集合）在 v0 中不分页，一次返回全部，仍用 `{data}` 封套"；3.6 补"成员资源内嵌用户的公开资料，因为 v0 没有读别的用户的接口"。Plane 两者都是这样；分页会让每个 store 循环取页。
- **代价**：集合没有硬性的上限，"由管理员的操作决定大小"不等于技术上有上限（Codex 4.2）。本文不断言大的工作区够快或不够快：大工作区的成员列表、项目列表的回答大小和加载时间留作实际的测量，交给 M8 的性能测量（16、13.2）。以后要给其中一个分页，是接口的改动，调用方要同时改：只读 `data` 的调用方会拿到不完整的集合。
- **另一种做法**：现在就返回 `next_cursor: null`，调用方照分页写。每个 store 多一段循环，换来的是以后加分页不改接口；v0 里用不到。

### 11.4 `shared` 可以放两个模块必须一致的纯取值规则（3.13；负责人已批准，2026-09-29）
- **问题**：总体设计 6.2 把 `shared` 定为跨边界的值和端口。M3 把邮箱规范化、网址检测、时区校验移进去，因为 `identity` 和 `workspace` 必须一致（邀请的邮箱要与账户的邮箱按同一规则比较）。
- **做法**：照此做，并在 6.2 写明：`shared` 还可以放"两个以上模块必须一致的纯取值规则"，只用标准库，不放业务流程；只放真正共用的三条。
  - 时区的校验用 `time.LoadLocation`，它读时区数据库，严格说不是不依赖环境的纯函数（Codex 4.2）。它先读主机的时区文件，主机没有这个名字的文件时才用 nerve 的二进制内嵌的时区数据库（`archtest` 的 `TestNerveBinaryEmbedsTheTimeZoneDatabase`），所以不同的机器可能对个别名字答得不同（大小写不敏感的文件系统上的 `asia/shanghai`，有这个文件的主机上的 `posixrules`）；主机的时区文件更新之后，同一个名字的回答也可能随之改变。3.13 要的只是各模块问同一个函数、同一时刻对一个名字的回答相同，这一点成立，所以它仍放在 `shared`，不为它建端口。
- **另一种做法**：`workspace` 经端口问 `identity`"这两个邮箱是否相同"，为纯函数建端口是过度设计。

### 11.5 标签只属于项目（3.16；负责人已批准，2026-09-29）
- **问题**：总体设计 5.3 写"标签名在工作区内唯一"，差异清单为工作区级标签定了唯一范围。Plane 的应用没有建工作区级标签的路径。
- **做法**：没有工作区级标签，标签名在项目内唯一、不分大小写；照此改 5.3 和差异清单（P7）。M7 的跨项目标签列表只是把各项目的标签列在一起，不需要工作区级标签。
- **另一种做法**：保留 `project_id` 可空的工作区级标签。没有使用者，只多一种要测的情形。

### 11.6 模块的两段组合：`Provide` 与 `New`（6.6；负责人已批准，2026-09-29）
- **问题**：模块之间的端口第一次成为双向的（`workspace` 与 `project` 互相要，`access` 与两者互相要，`identity` 与 `workspace` 互相要），组合根要能无环地建出它们。
- **做法**：给别的模块提供只依赖连接池的适配器的模块，先由 `Provide(pool)` 建出这些适配器（唯一的例外是 `identity` 的凭证锁，理由在 6.6）；每个模块再由 `New(Deps)` 建出全部用例（`access` 的 `Authorizer`、`project` 的连带这类依赖别的端口的实现也在 `New` 中建出），构造之后不再登记或注入任何东西。顺序：密钥 → 各模块的 `Provide` → `access.New` → `project.New` → `workspace.New`（收到 `project` 的连带）→ `identity.New`。第三稿按 Codex M-1 加的三个跨模块读取都是 `Provide` 的适配器，顺序不变（6.6）。以后的模块照做，写进总体设计 6.3 第 4 条"只在组合根接线"的说明（P1）。
- **另一种做法**：模块之间用 setter 事后注入（有半初始化的对象），或延迟求值的闭包（接线的顺序藏在运行时），都比两段组合难检查。

### 11.7 错误码前缀规则的修订和平台码 `forbidden`（3.4、5.3；负责人已批准，2026-09-29）
- **问题**：M2 设计 3.11 规定带前缀的错误码必须用声明它的那个模块文件的名字。M3 有跨模块的拒绝：项目的操作答"工作区不存在"（`workspace.not_found`），移出工作区成员、停用账户答"他是某个项目唯一的管理员"（`project.sole_admin`）；权限的拒绝来自没有接口文件的 `access`。
- **做法**：
  - 加平台码 `forbidden`（403），它是 `access` 的拒绝，跨越所有模块。
  - 前缀规则改为"前缀必须是 nerve 的一个模块（`api/modules/` 下有它的文件）"，不必是声明它的那个文件。码说明的是哪个领域拒绝了，同一个事实在各处用同一个码。`apitest` 核对前缀是真实的模块。
  - 这样的码由声明它的模块在自己的 HTTP 测试包里返回过（9.4，Codex M-2）；这是测试的位置，不改变这条规则。
  - M2 设计 3.11 的原文在 P1 同步修订（3.20）。
- **另一种做法**：把这些拒绝在每个调用方的模块里重新编码（例如 `workspace.project_sole_admin`、`identity.project_sole_admin`）。同一个事实有几个名字，客户端要认几遍。

---

## 12. Phase 划分与实施规划

所有 Phase 依次推进，每个 Phase 按 M2 的节奏（worktree → spec 和 plan → 按任务实现、逐个评审 → 整分支评审 → 修复 → review、交接 → `--no-ff` 合并、推送 → 持续集成通过 → 清理 worktree 和分支）。每个 Phase 合并时：持续集成的全部门禁、此前已加入的全部故事都通过；3.20 和 8.7 中标着这个 Phase 的行已在同一次合并中同步。

**约束**：
1. **每个故事的前置数据在它加入时都能经接口准备**（总体设计 8.2），不写 SQL。第二个工作区成员只能由接受邀请产生，所以凡是断言"成员 403""访客看不到"的故事都排在邀请之后（第 2 节最后一列）。权限矩阵是 Go 测试，用仓储准备数据，每个 Phase 都能加它的行（9.2）。
2. **每段约 16 个任务以内，plan 不超过约 1,500 行**。M2 的 P3 在设计里是一段，实施时超过了上限，由负责人批准拆成 P3a、P3b：设计时没有按任务数估每段的大小。M3 在这里按任务数分好，下面每段的任务列表就是 plan 的草稿。
3. **评审敏感的内容不在同一段**（M2 的教训：两件都需要细审的事放在一起会互相稀释）：权限框架（P1）、加锁约定和连带（P2 起步，P4、P5 延伸）、邀请令牌（P3）、停用（P6）。
4. **新的错误码随它的文案进来**：M2 的 vitest 要求 `PROBLEM_MESSAGES` 的键恰好等于契约的全部 `x-problem-codes`（M2 设计 3.11、7.3），后端的 Phase 也不例外（M3/P1 spec 第 3 节第 4 条）。P2–P7 中声明新错误码的任务，在同一个任务里把它加进 `PROBLEM_MESSAGES`（`web/apps/web/helpers/authentication.helper.ts`）和 `en`、`zh-CN` 两份 `auth.json`，并运行 `make test-web`。平台码 `forbidden` 随第一个声明它的操作（P2）进表。这张表和它的文案在 P8 移到通用的位置（P8 任务 13）。

**顺序**：
- **后端七段，前端四段**（先后端、后前端，M2 的做法：前端对接真实的接口）。
- **P1 平台**带上建、列、看工作区三个操作和建工作区的命令，让它合并的是有故事验收（W1、W10）的接口，而不只是基础设施；这三个操作不改成员关系，不牵涉加锁约定。
- **P2 工作区**让"先锁父行，再判定"第一次落地（修改、删除工作区，改成员角色）。
- **P3 邀请**之后才有第二个成员。
- **P4 项目**带上添加、加入项目成员，以及删除工作区和降为访客的连带：项目一出现，这两个连带就要成立，否则 P4 到 P5 之间的主干上会有"工作区访客是项目管理员"。
- **P5 成员关系**：移出、离开、改项目角色、唯一管理员、恢复成员的命令，共用 `EndMemberships`、3.7 的规则和交错测试 1、4、5、6。
- **P6 停用**单独一段：它跨越工作区、有接口和命令两条路、改动 `identity`，是单独的评审敏感内容；与 P5 合在一起约 22 个任务。
- **P7 状态和标签**最后：它们只依赖项目，状态的表在 P4 随默认状态建出。
- **P8 数据层**先于全部页面（3.1）：页面的故事加入时，挂载路径上已经没有旧接口。
- **没有空实现**：连带随表加入（删除工作区的连带：P2 显示设置和成员，P3 邀请，P4 项目，P7 标签；降为访客的连带在 P4）。

**规模**：148 个任务（后端 94、前端 54），每段 9–16 个。第三稿为 Codex 的发现加的测试让 P5、P6、P9 各多一个任务，其余加进已有的任务，每段仍在上限之内。第三稿的聚焦复核（17.3）加的规则和测试都并进已有的任务（P3、P4、P5、P6、P8、P9），各段的任务数不变。

### P1 `platform`：权限框架、组合与建工作区（后端，15 个任务）
- **目标**：权限框架、两段组合、矩阵测试的骨架定下；任何调用方都能建、列、看工作区，管理员能用命令建工作区。
- **任务**：
  1. 迁移 `00006`、`00007`，`sqlc.yaml` 的 `workspace` 条目，`deploy/runtime-grants.sql`。
  2. `TestSQLCSchemaScope` 的四种写法和反例（4.1）。
  3. `shared/authorize.go`；平台码 `forbidden`（`shared`、`apitest` 的 `platformCodes`）（3.4）。
  4. `apitest` 的前缀规则修订和反例（11.7）。
  5. 三条取值规则移到 `shared`（3.13）。
  6. `access` 的 domain：判定的纯函数（工作区级、项目级、看得到即可三个级别，事实是输入）、规则表（本 Phase 的行）、判定表测试。
  7. `access` 的 app：`WorkspaceRoles` 端口、`Authorizer`（`ProjectAccess` 端口随 `project` 在 P4 加入）。
  8. 操作名：`workspace/domain/actions.go`，`bootstrap` 的完整性测试（3.4）。
  9. 两段组合：`identity.Provide`、`workspace.Provide`、`access.New`、`workspace.New` 的接线（6.6）。
  10. `workspace` 的 domain 和保留名单（三段，"服务端"一段的 Go 测试，3.10）。
  11. `createWorkspace`（开关、账户行 `FOR SHARE`，`Accounts` 交回账户的状态，锁下要求有效，3.6 约定六）、`listWorkspaces`、`getWorkspace`。
  12. `checkWorkspaceSlug`。
  13. `nerve workspaces create` 和 `bootstrap.Workspaces` 的组合，组合测试扩展（6.6）。
  14. 矩阵测试的骨架（`pgtest` 从已准备的库复制）和本 Phase 的行；整程序测试覆盖新操作。
  15. 端到端：`api.ts` 的建工作区、`assert/workspace.ts`，W1、W10 的接口版本；3.20、8.7 中 P1 的行；review。
- **关闭**：M2 交接第 4 节（3.14）、第 5 节（3.11）、第 8 节（端口而不是例外、`TestSQLCSchemaScope`、`Authorizer`；`ProjectAccess` 在 P4 照同一写法）、第 12 节（review 写明 M3 没有分页的列表）；M1-P2、M1-P3、M1-P4 的保留名单的服务端一侧（前端一侧在 P8）。
- **完成线**：W1、W10 的接口版本通过；本 Phase 的矩阵格子通过；判定表测试、完整性测试通过；`TestSQLCSchemaScope` 的四个反例和前缀规则的反例在规则漏掉时失败；架构测试通过。

### P2 `workspaces`：工作区的管理和加锁约定（后端，12 个任务）
- **目标**：修改、删除工作区，列出成员、改角色，工作区的显示设置；"先锁父行，再判定"第一次落地。
- **任务**：
  1. 迁移 `00008`，grants。
  2. 锁父行的写法：工作区行的 `FOR SHARE`、`FOR NO KEY UPDATE`（带 `deleted_at IS NULL`），写用例的固定步骤（锁 → 判定 → 写）。
  3. `updateWorkspace`。
  4. `deleteWorkspace`（连带成员、显示设置）。
  5. `listWorkspaceMembers`：`MemberProfiles` 端口（`identity.Provide` 的 `PublicProfiles`，不加锁，含停用的账户，在 `bootstrap/ports.go` 转换，6.5）；`MemberUser`，邮箱按角色。
  6. `updateWorkspaceMember`（改自己 409；降为访客的项目连带随项目在 P4）。
  7. `getWorkspacePreferences`、`updateWorkspacePreferences`（`ON CONFLICT`，3.18）。
  8. 交错测试 2：两位管理员互相降级。
  9. 矩阵：本 Phase 的行。
  10. 整程序测试覆盖新操作。
  11. 端到端：W8 的接口版本。
  12. 3.20 中 P2 的行（总体设计 3.6、4.2）；review。
- **关闭**：M1-P2 侧边栏偏好（没有 `/sidebar-preferences/`，经 `WorkspacePreferences`）。
- **完成线**：W8 通过；交错测试 2 在真实数据库上通过；本 Phase 的矩阵格子通过。

### P3 `invitations`：邀请与凭邀请注册（后端，14 个任务）
- **目标**：邀请的全部操作；关闭注册时凭邀请注册；第二个成员从这里开始存在。
- **任务**：
  1. 迁移 `00009`，grants；删除工作区的连带加上邀请。
  2. `identity.LoadKeys` 移到 `bootstrap`，按用途派生的 MAC（6.6、11.1）。
  3. `identity.Provide` 加 `CredentialLock`，`workspace` 的 `CallerLock` 端口。
  4. 令牌的领域：格式、消息、比对（3.8）。
  5. `createWorkspaceInvitations`（账户行锁、批量校验、已忽略的邀请占着邮箱；"已是有效成员"经 `MemberProfiles` 比较邮箱；一批全有或全无，按规范化后的邮箱排序插入，唯一索引的 23505 翻译为 422 `invitations[i].email` 的 `duplicate`，3.8）。
  6. `listWorkspaceInvitations`、`updateWorkspaceInvitation`、`deleteWorkspaceInvitation`。
  7. `getWorkspaceInvitation`（公开，不含邮箱）；访问日志没有 `token=` 的测试（8.1）。
  8. `acceptWorkspaceInvitation`、`declineWorkspaceInvitation`：两者都最先以 `FOR SHARE` 锁调用者的账户行（锁下重读 `is_active`、邮箱，3.6 约定一、六），再锁工作区（接受 `FOR NO KEY UPDATE`，忽略 `FOR SHARE`）、邀请行 `FOR UPDATE`；已是有效成员时只消费邀请，恢复以前的成员行时取邀请的角色（3.8）；接受的用例测试（9.1）。
  9. `SignupPolicy` 扩展、`RegisterRequest.invitation`、`bootstrap` 的注册策略（3.8）。
  10. `apitest` 的请求体用例扩展到对象的数组（5.2）。
  11. 交错测试 3（接受与删除工作区）、9（创建邀请与重置密码）、12（接受与改邮箱）、18（两批重叠的邀请、单个邮箱的竞争）、19（忽略与停用，停用一方按 3.9 的取锁顺序）。
  12. 矩阵：邀请的行、账户级的操作。
  13. 端到端：`api.ts` 的邀请并接受；W3、W4、W5、W6 的接口版本（W6 在第二个 nerve 上建邀请，9.6）。
  14. 3.20、8.7 中 P3 的行（总体设计 1.1、4.2 按决策点 1、2、4 的裁定）；review。
- **关闭**：M2 交接第 1 节的接口一侧（页面一侧在 P9）。
- **完成线**：上述故事通过；交错测试 3、9、12、18、19 通过，都没有 40P01；接受的用例测试覆盖已是有效成员、以前的成员行、没有成员行三种；邀请的唯一冲突翻译为 422（下标正确）的测试通过；数组请求体的用例通过。

### P4 `projects`：项目、项目成员的加入与两个连带（后端，16 个任务）
- **目标**：项目的建、列（按可见性）、看、改、删、归档、恢复，项目成员的列出、添加、加入，项目的显示设置；删除工作区和降为访客的连带。
- **任务**：
  1. 迁移 `00010`–`00013`，`sqlc.yaml` 的 `project` 条目，grants；`projects_logo_props_check` 和它的十个反例、四个合法值的仓储测试（4.6，含 Codex S5 的值）。
  2. `project` 的 domain：名称、标识、可见性、`archive_in`、`logo_props`、负责人、默认的 6 个状态、`sort_order`；`actions.go`。
  3. `ProjectAccess`（`project.Provide`）、`access` 的项目级端口和规则行；`WorkspaceDirectory`、`WorkspaceMembers`（`workspace.Provide`）。
  4. `createProject`（工作区 `FOR SHARE`、成员行 `FOR SHARE`、默认 6 个状态、创建者和负责人、显示设置）。
  5. `listProjects`（可见性、`archived`、顺序）和可见性一致的测试（9.3）。
  6. `getProject`、`checkProjectIdentifier`。
  7. `updateProject`（负责人、默认负责人的规则，3.19）。
  8. `deleteProject`、`archiveProject`、`unarchiveProject`。
  9. `getProjectPreferences`、`updateProjectPreferences`。
  10. `listProjectMembers`、`addProjectMembers`（目标的 422 在判定之后给出，3.6 约定三）。
  11. `joinProject`（`sort_order = 65535`；恢复以前的行时角色取 `min(原来那一行的角色, 他现在的工作区角色)`，3.5，9.1 的四行表）。
  12. `ProjectCascade` 端口和 `DeleteWorkspaceProjects`、`DemoteToGuest`（含接受邀请恢复为访客时的调用，3.8，它让加入的上限成立）；`project.New` 在 `workspace.New` 之前（6.6）；交错测试 17（降为访客与加入、添加，两种顺序）。
  13. 矩阵：本 Phase 的项目级的行，含 `addProjectMembers` 无效目标的 PM、X 两格（9.2）；整程序测试；`project` 的 HTTP 测试返回各操作声明的 `workspace.not_found`（9.4）。
  14. 端到端的 fixture：`api.ts` 的建项目、加项目成员，`assert/project.ts`。
  15. 端到端：P1、P2、P3、P4、P8 的接口版本；W3 加上项目的连带。
  16. 3.20 中 P4 的行；review。
- **关闭**：M2 交接第 7 节的接口一侧（`logo_url`、`cover_image_url`、`avatar_url` 可为 `null`；`IUserLite` 在 P8）、第 9 节（4.12）；M1-P2 项目字段；M1-P3 不再读的字段、地址（`project-identifiers` 不带结尾 `/`）。
- **完成线**：上述故事通过；可见性一致测试、连带的回滚测试、`logo_props` 的 CHECK 反例、恢复时角色的四行表、交错测试 17 通过；`project` 的 `apitest.Main` 两个方向核对通过；本 Phase 的矩阵格子通过。

### P5 `memberships`：结束成员关系与恢复（后端，15 个任务）
- **目标**：移出、离开工作区和项目，改项目成员的角色，唯一管理员的两条规则，恢复成员的命令。
- **任务**：
  1. `ProjectCascade.EndMemberships`（在调用时列举、锁他的项目、查唯一管理员、停用，3.6 约定六）。
  2. `end_membership.go`：移出、离开共用的一步（3.7 规则 2；软删除这个工作区里发给他邮箱的待接受邀请，邮箱经 `MemberProfiles`，3.8）。
  3. `removeWorkspaceMember`；`workspace` 的 HTTP 测试为它和 `leaveWorkspace` 返回 `project.sole_admin`（9.4）。
  4. `leaveWorkspace`（规则 1）。
  5. `updateProjectMember`（`{Admin}` 加工作区管理员的例外、3.5 的相对规则）。
  6. `removeProjectMember`。
  7. `leaveProject`。
  8. `nerve workspaces reactivate-member`（账户行 `FOR SHARE`、停用的账户照样恢复并提示、`ProjectMembershipCounts` 端口，3.11）。
  9. 交错测试 1（两位管理员同时离开）、4（添加项目成员与移出）。
  10. 交错测试 5（移出与建项目）、6（移出管理员与删除工作区）。
  11. 矩阵：剩下的成员关系的行，已不是成员的身份（9.2 的"已被移出"、P-前）。
  12. 端到端：W2、W7、P5 的接口版本（P5 含被移出的项目管理员重新加入得到 15）。
  13. 端到端：W12（含旧邀请的一段）。
  14. 邀请从不改变有效的成员关系：Codex S2 顺序的集成测试；结束的成员关系不留下邀请：复核 spike 9d 的顺序，移出、离开各一次，之后旧链接 404（9.3）。
  15. 3.20、8.7 中 P5 的行；review。
- **完成线**：上述故事通过；交错测试 1、4、5、6，S2 顺序和 9d 顺序的集成测试通过；唯一管理员的两条规则各有正反例；`workspace` 的 `apitest.Main` 两个方向核对通过。

### P6 `deactivation`：停用账户与成员关系（后端，9 个任务）
- **目标**：停用账户在同一个事务里结束他的全部成员关系、删除发给他邮箱的全部邀请；唯一管理员时拒绝；约定六的每一条增长路径都与停用串行。
- **任务**：
  1. `identity` 的 `MembershipDeactivator` 端口，`LockedAccount` 加 `Email`、`deactivate` 传入锁下的邮箱，`deactivateMe` 的两个码；`identity` 的 HTTP 测试经 `fakeDeactivate.err` 返回这两个码（3.9、9.4）。
  2. `workspace` 的 `Deactivator`：在已锁的账户行之下列举并锁他的全部工作区（上锁时已删除的跳过）、查唯一管理员、删除全部邀请、停用成员行、一次调用 `EndMemberships`（3.6 约定六、3.9）。
  3. 跨工作区的 `EndMemberships`：在调用时列举，按 `id` 升序一次锁住全部项目，上锁时已删除的跳过（3.6、3.9）。
  4. `nerve users` 的组合加上停用的部分，组合测试（6.6）。
  5. 交错测试 7（停用与接受邀请，Codex S1 的四种走法，另跑恢复以前的成员行）、8（停用与创建工作区）；19 换成真实的停用再跑一次。
  6. 交错测试 13–16（停用与添加项目成员、加入项目、以他为负责人建项目、恢复成员；13、14 另跑恢复以前的项目成员行）。
  7. 停用的两条路的测试（9.3），含复核 spike 15（列举到的工作区在上锁之前被删除，跳过而不是 404）和改邮箱先提交时删除发给新邮箱的邀请。
  8. 端到端：W9 的接口版本（含命令）。
  9. 3.20、8.7 中 P6 的行；review。
- **关闭**：M2 交接第 6 节；M4 的 M2 收尾交接第 1 节在 M3 的部分（停用不投递任务，所以不提前，review 写明，13.1）。
- **完成线**：W9 通过；交错测试 7、8、13–16 和真实停用下的 19 通过；`identity` 的 `apitest.Main` 两个方向核对通过（`deactivateMe` 的两个码由它自己的 HTTP 测试返回），`bootstrap` 的整程序测试在真实的组合上返回它们。

### P7 `states-and-labels`：状态与标签（后端，13 个任务）
- **目标**：状态和标签的全部操作，锁项目行、守卫的写、两层的标签。
- **任务**：
  1. 迁移 `00014`，grants；删除项目、删除工作区的连带加上标签。
  2. 状态的规则：组、`sequence`、一组至少一个（3.17）。
  3. `listStates`（已归档的为空）、`createState`。
  4. `updateState`。
  5. `deleteState`、`markDefaultState`（守卫的写，两条语句）。
  6. `listWorkspaceStates`。
  7. 标签的规则：两层、不分大小写（3.16）。
  8. `listLabels`、`createLabel`。
  9. `updateLabel`、`deleteLabel`（连带子标签）。
  10. 交错测试 10（设为默认与删除）、11（两次改父标签）。
  11. 矩阵：状态、标签的行，已归档项目的小表；整程序测试。
  12. 端到端：P6、P7 的接口版本，W11 的四格抽样。
  13. 3.20 中 P7 的行（总体设计 5.3、差异清单的 `labels`）；review。
- **完成线**：上述故事通过；交错测试 10、11 通过；项目级矩阵全部格子通过（后端的矩阵至此完整）。

### P8 `web-data-layer`：前端的数据层（16 个任务）
- **目标**：M3 的全部类型、service、store、权限 store 和挂载路径上的取数迁到新接口，按会话分代；使用方改到能编译、行为不变；系统内接受删除；关键词规则完整。
- **任务**：
  1. `RootStore.dispose()` 和 `project_filter` 反应的释放（7.1）。
  2. 工作区的类型、service、store；`RESTRICTED_URLS` 删除，保留名单"应用"一段的 vitest（3.10）。
  3. 工作区成员、邀请的 store；`MemberUser` 取代 `IUserLite` 的使用方。
  4. 系统内接受的删除：`/invitations` 页、入口、路由、新手引导的"加入工作区"一步和它们的 service、store 方法（7.8）。
  5. 项目 store，`IProject`、`TProject` 的使用方；`is_favorite` 的读取、收藏的两个 store 方法、`favoriteProjectIds` getter 和 `favorite.store.ts:284-287` 的项目分支删除（3.2）。
  6. 项目成员 store。
  7. 状态 store（顺序、组中的位置），`IState` 的使用方。
  8. 标签 store（层级），`IIssueLabel` 的使用方。
  9. 显示设置的 store（工作区、项目）。
  10. 权限 store（`Workspace.role`、`Project.member_role`）。
  11. 设置 store 删除、落点函数、`AuthenticationWrapper`（3.14、7.4）。
  12. 两个包装层和 `TopNavigationRoot` 挂载时的取数：M3 的改调新 store，按权限启用（`ProjectAuthWrapper` 的子资源在确认是项目成员之后才取，7.1），M6、M7 的删除，首页"最近"小部件删除（3.1）。
  13. service 按代建（10 处模块级实例消失）、修改经 `oneAtATime()` 的核对；`sessionGuard()` 移到 `core/lib/in-session.ts`（7.1）；problem 码的文案表 `PROBLEM_MESSAGES` 和它的文案从 `authentication.helper.ts` 和 `auth` 命名空间移到通用的位置，核对它与契约的码一致的 vitest 随它移动（第 12 节约束 4）。
  14. 关键词规则和 `/user-properties/` 的例外（`until: "M4"`，7.10），`/projects/{id}/search-issues/` 是规则的不命中样例；旧 `ProjectService` 只留三个方法（7.3）。
  15. vitest（9.5 中 P8 的各项）。
  16. S2 的改写；3.20 中 P8 的行；review。
- **关闭**：M2 交接第 3 节、第 7 节（`IUserLite` → `MemberUser`）；M1-P3 项目成员（`project-invitations` 的例外删除）；M1-P2、M1-P3、M1-P4 保留名单的前端一侧（前后端一份，有测试）。
- **完成线**：S1、S2 通过，此前的全部故事仍通过；M2 交接第 3 节的 `git grep` 中 M3 的 10 处消失；`node tools/keywords.mjs` 通过；9.5 中 P8 的 vitest 通过；`tsc`、knip 通过。

### P9 `web-workspace-pages`：工作区的页面（14 个任务）
- **目标**：落点、新手引导、邀请页、注册页、工作区首页和侧边栏、工作区设置对接新接口，行为按第 2 节的故事。
- **任务**：
  1. 落点的页面一侧：切换、创建工作区、接受邀请之后写 `last_workspace_id`；删除、离开之后的去向。
  2. 新手引导：创建工作区、邀请成员两步；已有工作区的人资料一步之后完成（7.4）。
  3. 邀请页（7.4）。
  4. 注册页带邀请（7.4）。
  5. 首页、侧边栏的工作区部分、项目导航对话框（7.5）。
  6. 工作区设置 general（图标上传删除）。
  7. 工作区设置 members：成员列表、改角色、移出、离开；邀请只为管理员取（7.1）。
  8. 邀请的界面：批量邀请、复制链接、改角色、删除、已忽略（7.5）。
  9. 停用账户的弹窗显示 409 的说明（7.5）。
  10. `WorkspaceAuthWrapper` 的界面（M2 交接第 13 节）。
  11. 页面级副作用的会话核对：删除工作区、离开工作区、创建工作区、接受邀请之后的跳转和提示先核对 `inSession()`（spec 逐个列出组件）；删除工作区组件的 vitest；W3 的会话切换端到端：`route.fetch()` 在换账户之前，`route.fulfill()` 在之后；变异核对：去掉 `inSession()` 时它失败，写进 review（7.1、9.6）。
  12. 端到端：W1–W5 的页面版本（W4 含成员打开成员页没有失败的请求）。
  13. 端到端：W6–W9 的页面版本。
  14. 浏览器核对 C1–C5；3.20 中 P9 的行（README"前端"一节）；review。
- **关闭**：M2 交接第 1 节（页面一侧）、第 2 节、第 13 节。
- **完成线**：上述故事通过，含 W3 的会话切换和 W4 的成员视角；9.5 中 P9 的 vitest 通过；C1–C5 写进 review。

### P10 `web-project-pages`：项目的页面（15 个任务）
- **目标**：项目列表、创建、设置、成员、归档、导航偏好对接新接口；下拉框、复制、表情选择器的交接完成。
- **任务**：
  1. 项目列表、卡片、加入。
  2. 已归档的项目页。
  3. 创建项目（去掉封面，负责人从工作区的管理员和成员中选）。
  4. `ProjectAuthWrapper` 的界面（7.6）；不是成员时只取项目详情（7.1，P2 故事的页面版本核对）。
  5. 项目设置 general。
  6. 项目设置 members：成员、添加、改角色、移出；负责人、默认负责人、访客可见全部。
  7. features、automations。
  8. 离开项目的顺序（M1-P4 交接）；删除、离开、创建项目之后的跳转和提示先核对 `inSession()`，删除项目组件的 vitest（7.1）。
  9. 侧边栏的项目顺序、项目页头的导航偏好。
  10. 下拉框（M2 交接第 14 节）。
  11. 复制到剪贴板。
  12. 表情选择器的数据（7.7）。
  13. 端到端：P1–P5、P8 的页面版本。
  14. 浏览器核对 C6–C8。
  15. 3.20 中 P10 的行；review。
- **关闭**：M2 交接第 10、11、14 节；M1-P4 离开项目的顺序。
- **完成线**：上述故事通过；打开表情选择器时没有外部请求、没有 CSP 违规；C6–C8 写进 review。

### P11 `web-states-labels-and-cleanup`：状态、标签的页面与清理（9 个任务）
- **目标**：状态、标签的设置页；M3 领域的死成员、死 prop 和 oxlint 清完；关键词守卫收尾。
- **任务**：
  1. 状态设置页（7.6）。
  2. 标签设置页（7.6）。
  3. 死成员和死 prop：`--rows M3` 剩下的行（7.9）。
  4. oxlint：M3 改到的文件清零。
  5. `promise(always-return)` 全仓清零，各包上限调低（7.9）。
  6. 关键词守卫的最终核对：M3 加的例外只剩 `/user-properties/` 的两处（`until: "M4"`），`until: "M3"` 的例外都已删除；旧 `ProjectService` 只剩 7.3 的三个方法。
  7. 端到端：P6、P7、P9、W11 的页面版本。
  8. 浏览器核对 C9、C10。
  9. 前端改动清单的 M3 一行（3.20）；review。
- **关闭**：M1-closeout 交接（死成员、oxlint）；M1-P2 个人主页的页面一侧（C10）。
- **完成线**：全部 21 个故事的页面版本和接口版本通过；`domains.mjs --rows M3` 没有输出，或剩下的每一行写进 review；各包的 oxlint 上限已调低；C9、C10 写进 review。

### 收尾 `closeout`
- 6 份交接逐项写下结论，`status` 改为 `closed`（13.1）。
- 按 3.20 逐行核对上级文档和差异清单都已在各 Phase 同步；差异清单按第 4 节逐列总核对。
- 写好给 M4–M8 的交接（13.2），包括 7.9 中按路径误归到 M3 的 41 行。
- 规模估计与实际的对比（7.11），照 M2 收尾附录 B 的量法。
- 总体设计 9.4 中 M3 的状态改为"已完成"。

---

## 13. 交接的落点

### 13.1 交给 M3 的交接
**M3 自己的 6 份**（`docs/v0/M3-workspace-project/handoffs/`）：

| 交接与节 | 落在哪里 | 关闭条件（在哪个 Phase 满足） |
|---|---|---|
| M2-closeout §1 邀请与注册 | 3.8；决策点 1、2、4；7.4 | 负责人已裁定（第 10 节）；只凭邮箱的接受被拒绝、凭有效令牌且邮箱一致的成功；接受最先锁住接收账户的行，已是有效成员时不改成员关系（交错测试 12、9.1 的用例测试）；注册关闭时带有效邀请的注册成功、不带的 `identity.signup_disabled`；W5、W6 两个版本（P3 接口、P9 页面） |
| M2-closeout §2 落点与新手引导的取数 | 3.1、3.14、7.4 | 落点规则（上次的、最早的、`/create-workspace`）；新手引导用新接口；A3、A10 的断言不改仍通过；取数失败时没有未处理的拒绝（SWR fetcher 返回 Promise，vitest）；`currentUserSettings` 和四处 `await` 删除（P8）；README"前端"一节改写；挂载时的规则延伸到 M3 的页面（3.1）（P9） |
| M2-closeout §3 stores 按会话分代 | 7.1 | M3 的 10 处模块级实例消失（`publicClient` 除外）；`project_filter` 的反应随退役的一代释放，有单元测试；新 SWR 键带 `loginId`（P8） |
| M2-closeout §4 `last_workspace_id` 的外键 | 3.14 | 不加，理由写在 3.14（本文满足；P1 的 review 复述） |
| M2-closeout §5 `workspace_creation_enabled` | 3.11 | 关闭时 403 `workspace.creation_disabled`，有测试；提供 `nerve workspaces create`（P1） |
| M2-closeout §6 停用的端口 | 3.6、3.7、3.9 | 三件事在停用的事务里；唯一管理员时拒绝；接口和命令两条路都有测试；`deactivateMe` 的码声明、两个方向核对（`identity` 自己的 HTTP 测试返回过它们，9.4）；差异清单"停用账户"一行；加锁顺序和成员关系集合的增长与收缩写在 3.6，每条增长路径与停用的交错测试（7、8、13–16）和忽略与停用的交错测试（19）通过（P6） |
| M2-closeout §7 可空的引用字段 | 3.2、5.2、7.2 | `logo_url`、`cover_image_url`、`MemberUser.avatar_url` 可为 `null`（P1、P2、P4）；`IUserLite` 由 `MemberUser` 取代（P8） |
| M2-closeout §8 模块边界 | 3.3、3.4、4.1 | 端口而不是例外（3.3）；`TestSQLCSchemaScope` 四种写法各一个反例；`Authorizer` 在 `shared`，规则 10 照旧（P1） |
| M2-closeout §9 删除关系图 | 3.15、4.12 | 图写在 4.12，每条指向 `users` 的外键有去向；项目负责人改为 `SET NULL`，登记差异（P4） |
| M2-closeout §10 表情选择器的数据 | 7.7、8.5 | 不请求 `cdn.jsdelivr.net`，没有 CSP 违规（故事 P1、P3 的 `watchPage`，C6）；CSP 不放开外部来源（P10） |
| M2-closeout §11 M2 留下的调用和类型 | 7.2、7.3、7.4、7.10 | `leaveWorkspace`、`joinProject`、`leaveProject` 改用生成的客户端，`is_bot` 删除，`owner` 不再存在，`plane-user-urls` 收紧为整个 `/api/users/`（P8）；新手引导的三步：创建、邀请用新接口，加入一步随决策点 2 删除（P8、P9）；时区用 `GET /api/v0/timezones`（P9、P10） |
| M2-closeout §12 页大小的规则 | 3.12 | M3 没有分页的列表：P1 的 review 写明，本节原样写进 M4 的交接（收尾，13.2） |
| M2-closeout §13 P5 改到、M2 走不到的页面 | 9.7 C4 | 逐条的浏览器核对写进 review；"加入工作区"一步删除，写明（P9） |
| M2-closeout §14 下拉框和复制 | 7.7、9.7 C7、C8 | `CustomSearchSelect` 能用 Tab 到达、键盘打开；`member-options` 的列表在按钮旁；9 个和 4 个调用方核对；3 处复制处理失败（P10） |
| M1-closeout 死成员和死 prop | 7.9 | `--rows M3` 的每一行消失或写进 review；按路径误归的 41 行写进 M4、M6、M7 的交接（P11、收尾） |
| M1-closeout oxlint | 7.9 | 改到的文件 0 条；`promise(always-return)` 全仓清零；各包上限调低，review 写明（P8–P11） |
| M1-P2 项目字段 | 4.6、5.2 | 接口没有 `close_in`、`default_state`、`page_view`、`estimate_id`（P4） |
| M1-P2 侧边栏偏好 | 3.18、5.1 | 没有 `/sidebar-preferences/`；项目导航偏好经 `WorkspacePreferences`（P2） |
| M1-P2 保留的工作区地址 | 3.10 | 与 M1-P3、M1-P4 的同一项一起关闭（P1、P8） |
| M1-P2 个人主页 | 决策点 3；9.2 | 裁定写进矩阵（本文 9.2 的一格；页面在 P11 的 C10，接口的一行交给 M4） |
| M1-P3 不再读的字段 | 4.6、5.2、3.19 | 项目接口没有 `anchor`、发布设置；Nerve 没有项目动态，不产生那几类记录（P4）。视图、收集箱的接口不在 M3，由 M7 的同名交接（`docs/v0/M7-collaboration/handoffs/M1-P3-trim-platform.md`）约束，M3 的 review 写明 |
| M1-P3 项目成员 | 3.5、7.10 | 只有"从工作区成员中添加"（P4）；`joinProject` 改用新接口，`project-invitations` 的例外删除，`node tools/keywords.mjs` 通过（P8） |
| M1-P3 保留的工作区地址 | 3.10 | 与后端同源；产品词逐个有结论：全部去掉，另加四个预留段（3.10）（P1、P8） |
| M1-P3 地址 | 5.1 | `project-identifiers` 等新地址都不带结尾 `/`（P4） |
| M1-P4 保留的工作区名 | 3.10 | 名单"应用"一段正好是 `routes/core.ts` 的顶层静态路由段（加 `public/` 的顶层目录），前后端一份，vitest 和 Go 测试各守一段（P1、P8） |
| M1-P4 离开项目的顺序 | 7.6 | 先等接口成功再跳转，vitest 和故事 P5（P10） |
| M0-P3 分页的公共组件 | — | 已在 M2/P3a 完成（`status: done`）。M3 的列表不分页（3.12），不引用这些组件；收尾时状态改为 `closed` |

**别的 M 的交接中点名 M3 的**：

| 交接与节 | M3 的部分 | 落点 |
|---|---|---|
| M4 的 M2-closeout §1 只投递的 River 客户端 | "端口的实现投递任务时提前到 M3" | M3 的停用端口不投递任务（3.9），不提前；P6 的 review 写明，M4 照原计划 |
| M4 的 M2-closeout §6 事件订阅者的写法 | **M3 不涉及**：M3 不引入事件（3.3），这一节原样留给 M4 | 13.2 给 M4 一条说明：删除工作区的连带以后可以改挂到事件上 |
| M4 的 M2-closeout §5、M6 的 §2、M7 的 §5 删除关系图 | 各自延伸自己的表，M3 延伸 M3 的 | 4.12 |
| M4 的 M2-closeout §8 callout 的表情图 | 表情回应"用表情选择器，数据由 M3 改为本站提供" | 7.7；M4 核对表情回应 |
| M8 的 M1-closeout"AGPL 的三项义务和发往第三方的请求" | 表情选择器的数据请求 | 7.7 完成之后，M8 对这一项只剩核对 |
| M5 的 M1-closeout"新建项目时的封面值" | M3 删掉了上传预设封面的一步（3.2） | M5 加回封面时按它的关闭条件做；13.2 |
| M5、M8 的 M2-closeout 开头"M3 及以后有 stores 的 M" | M3 的部分就是 M2-closeout §3 | 7.1 |

### 13.2 M3 交给后续 M 的事项（收尾时写成交接）
| 接收者 | 事项 |
|---|---|
| M4 | **事件**：M3 没有领域事件，删除工作区、降为访客、结束成员关系的连带经 `ProjectCascade` 同步完成（3.3）。M4 随第一个异步订阅者引入事件（M2 设计 3.15、M4 的 M2 收尾交接第 6 节不变），届时可以把删除工作区的连带改挂到 `WorkspaceDeleted` 上，并为工作项加上删除项目、删除工作区的连带。**约定**：先锁父行再判定、加锁顺序、成员关系集合的增长与收缩（3.6 约定六，以后加入新的成员关系时照它写，恢复"不比新授予给得更多，也不比原来那一行更多"）、共享锁之下的批量插入按唯一键排序（约定五）、账户行只作第一把锁（约定一）；页面按权限取数、`await` 之后先核对会话（7.1）；声明在一个模块上的跨模块错误码由这个模块的 HTTP 测试返回（9.4）；规则表和操作名在各模块、完整性测试（3.4）；权限矩阵的写法（9.2）；错误码前缀规则的修订和 `forbidden`（11.7）；规则表的 `AllowCreator` 和 `guest_view_all_features` 对工作项的约束（3.4）。**数据**：删除状态前检查它的工作项、删除标签时处理 `issue_labels`；工作项编号取 `projects.last_issue_sequence`（4.6）；工作项的筛选和显示列加在 `workspace_user_properties`、`project_user_properties`（3.18），旧 `ProjectService` 的 `/user-properties/` 两个方法和关键词例外（`until: "M4"`）；旧 `ProjectService` 的 `projectIssuesSearch` 和它的三个调用方（M4 的选父工作项、M6 的添加已有工作项、M7 的收集箱查重）随 M4 的工作项搜索接口替换，之后删除旧 service（7.3）；60 天清理包括 M3 的表，指向 `workspaces`、`projects`、`labels.parent_id` 的外键已有不带条件的索引，工作项的表照做（4.12）；工作项引用状态、标签、项目的外键在 4.12 的图上延伸。**其他**：页大小的规则移到 `shared`、游标不签名的提醒（M2 交接第 12 节原样，3.12）；决策点 3 的规则行（个人主页的工作项列表）；`issue_calendar_view` 的反应和 `issue/root.store.ts` 的 `autorun` 的释放（7.1）；`workspace-draft-issues/base.ts` 的 16 行死成员；M3 页面上指向工作项页面的链接接上之后，这些页面进入 `watchPage` 的范围（3.1） |
| M5 | `workspaces.logo_asset_id`、`projects.cover_image_asset_id` 的迁移归各自模块，图标和封面的上传控件从 Plane 的源码加回，`logo_url`、`cover_image_url`、`MemberUser.avatar_url` 有真值（3.2）；新建项目的封面值（M5 的 M1-closeout，M3 已删掉上传预设封面的一步） |
| M6 | `ProjectAuthWrapper` 取迭代、模块（3.1，按 7.1 由权限启用）；`existing-issues-list-modal.tsx` 改用 M4 的工作项搜索（7.3）；`cycle_filter`、`module_filter` 的反应的释放（7.1）；`core/sidebar/progress-stats/` 的 8 行死成员；迭代、模块的外键在 4.12 的图上延伸 |
| M7 | 侧边栏的收藏、未读通知数、首页的"最近"小部件、项目的视图和分诊状态的取数、项目侧边栏的 `intake_count`（3.1）；打开 `intake_view` 时建默认收集箱、分诊状态的接口（3.17）；跨项目的标签列表（3.16、7.3）；项目的收藏（`is_favorite`、卡片上的收藏按钮、建项目之后加入收藏、`favorite.store.ts` 的项目分支、`favoriteProjectIds` getter，3.2）和归档项目时的收藏处理；收集箱查重 `select-duplicate.tsx` 改用 M4 的工作项搜索（7.3）；`workspace-notifications.ts` 的 17 行死成员；工作区视图的筛选列（3.18） |
| M8 | 工作区、项目、成员的 Webhook 事件：在 M4 引入的事件机制上加（M3 没有事件）；表情选择器的数据请求只剩核对（7.7）；README 中邀请链接与签名密钥的说明随发布核对（8.7）；性能测量加上大工作区的成员列表、项目列表（不分页的集合，11.3）：回答的大小和页面的加载时间，按测到的数字决定要不要给其中一个分页 |

---

## 14. 完成标准
- [ ] 第 2 节的 21 个故事（页面版本、接口版本）和 S1、S2 在持续集成中通过。
- [ ] 权限矩阵的全部格子（含已不是成员的身份）、可见性一致的测试、19 个交错测试、"邀请从不改变有效的成员关系"和"结束的成员关系不留下邀请"的集成测试在真实数据库上通过。
- [ ] 3.6 约定六的每一条增长路径（接受邀请、创建工作区、`reactivate-member`、建项目的负责人、添加项目成员、加入项目、恢复以前的成员关系）都有与停用的交错测试（恢复的走法在 7、13、14 里）；项目一侧的增长另有与降为访客的交错测试（17）。
- [ ] 每个模块的 `apitest.Main` 两个方向核对通过：声明在它的操作上的跨模块码由它自己的 HTTP 测试返回过（9.4）。
- [ ] `access/domain` 的判定表测试通过；各模块的操作名与规则表一一对应；每个操作在矩阵里有一行。
- [ ] 持续集成的全部门禁通过（含架构测试、`TestSQLCSchemaScope`、`apitest` 的前缀规则、命令行的组合测试、关键词守卫、oxlint 上限、knip）。
- [x] 第 10 节的四个决策点有负责人的裁定，本文按裁定修改；第 11 节的七个架构问题已由负责人批准（2026-09-29）。
- [ ] 13.1 的每一项已关闭并写进相应 Phase 的 review，6 份交接 `status: closed`；13.2 的交接已写好。
- [ ] 3.20 的每一行已在对应的 Phase 同步（含 M2 设计 3.11 的修订）；差异清单按第 4 节逐列核对过；README 按 8.7 更新。
- [ ] 浏览器核对 C1–C10 写进各 Phase 的 review。
- [ ] `domains.mjs --rows M3` 没有输出（或剩下的每一行写明是误报）；oxlint 上限已调低。
- [ ] M3 能到达的页面挂载时没有旧接口请求，也没有调用者无权读的请求（故事的 `watchPage` 覆盖每一页，7.1）。
- [ ] M3 页面的组件在 `await` 之后的跳转、提示先核对会话（7.1；W3 的会话切换端到端）。
- [ ] 规模估计与实际的对比写进收尾的 review。
- [ ] 总体设计 9.4 中 M3 的状态改为"已完成"。

## 15. Phase 进度表
| Phase | 内容 | 状态 |
|---|---|---|
| 设计 | 本文（第三稿，按聚焦复核修订） | 第一稿经独立评审、第二稿经 Codex 对抗性评审、第三稿经聚焦复核，都已落实（17.1–17.3）；决策点已裁定，架构问题已批准（2026-09-29） |
| P1 `platform` | 权限框架、组合与建工作区（后端，15） | 已完成：[spec](specs/P1-platform.md)、[plan](plans/P1-platform.md)、[review](reviews/P1-platform-review.md)（执行时 16 个 Task） |
| P2 `workspaces` | 工作区的管理和加锁约定（后端，12） | 已完成：[spec](specs/P2-workspaces.md)、[plan](plans/P2-workspaces.md)、[review](reviews/P2-workspaces-review.md)（执行时 15 个 Task） |
| P3 `invitations` | 邀请与凭邀请注册（后端，14） | 未开始 |
| P4 `projects` | 项目、项目成员的加入与两个连带（后端，16） | 未开始 |
| P5 `memberships` | 结束成员关系与恢复（后端，15） | 未开始 |
| P6 `deactivation` | 停用账户与成员关系（后端，9） | 未开始 |
| P7 `states-and-labels` | 状态与标签（后端，13） | 未开始 |
| P8 `web-data-layer` | 前端的数据层（16） | 未开始 |
| P9 `web-workspace-pages` | 工作区的页面（14） | 未开始 |
| P10 `web-project-pages` | 项目的页面（15） | 未开始 |
| P11 `web-states-labels-and-cleanup` | 状态、标签的页面与清理（9） | 未开始 |
| 收尾 `closeout` | 交接、同步核对、规模对比 | 未开始 |

## 16. 风险
| 风险 | 影响 | 应对 |
|---|---|---|
| M3 的规模约是 M2 的三倍（45 个操作，前端约 380 个文件） | 周期拉长；某个 Phase 超出规模上限 | 按任务数分成 11 段，每段 9–16 个任务（第 12 节）；某个 Phase 的 plan 仍超出约 1,500 行时，照 M2 拆分并请负责人批准 |
| 前端的数据层（P8）一次改到的文件多（store 的使用方约 140 个） | plan 超出上限 | 已按领域分成 16 个任务；仍超出时拆成工作区一侧和项目一侧两段，两个包装层挂载时的取数留在第一段，页面的 Phase 仍在两段之后 |
| 规则表与列表的 SQL 过滤走散 | 列表里出现看不到的项目，或漏掉看得到的 | 可见性一致的集成测试（9.3）；规则只在 `access/domain` 一处 |
| 无人管理的工作区（唯一的成员被停用，或成员被误移出） | 数据留在库里，没有人能进入 | `nerve users activate` 和 `nerve workspaces reactivate-member`，与 Plane 的两个管理命令相同；README 写明（3.11、8.6） |
| 换签名密钥使全部邀请链接失效；进程之间密钥不同使链接时好时坏 | 待接受的人拿到的链接不能用 | README 写明同一部署共用一个密钥文件；换钥之后管理员重新复制（3.8、11.1） |
| 父行的锁让同一工作区、同一项目的管理变更排队 | 移出、离开、改角色、状态和标签的写入互相等待 | 这些操作很少，持锁时间只有一个事务；往父行下面加行的 `FOR SHARE` 互不阻塞（3.6 的 spike） |
| 类型的改动波及 M4 以后的组件（`IProject`、`IState`、`IIssueLabel` 的使用方） | 这些页面在 M3 里走不到，故事测不到 | `tsc`、knip、vitest 守编译和单元；改动只限类型和字段名；M4 接上时在它的故事里核对（13.2） |
| 集合型的列表没有硬性的上限（11.3） | 很大的工作区里，成员列表、项目列表的回答变大、页面变慢 | 本文不断言够快或不够快；交给 M8 的性能测量，量大工作区的回答大小和加载时间，按数字决定要不要给其中一个分页（13.2） |
| 以后加入新的成员关系时漏掉约定六 | 停用、移出之后留下有效的成员关系，或绕过唯一管理员的保证（第二稿的 Codex I-1、I-2 就是这一类） | 约定六写成全局约定（3.6、11.2），交给 M4（13.2）；每条增长路径都要有与停用的交错测试（第 14 节） |
| 测试时间 | 矩阵约 400 格、E2E 多约 50 个测试 | 矩阵并行、写的格子各用复制的库（9.2，spike 实测复制 3–5 秒）；E2E 一步预计多约 45 秒（7.11） |
| M3 页面上指向 M4 以后页面的链接 | 点进去得到旧接口的 404 | 已知的过渡（3.1），各 M 接上时消失 |

## 17. 评审的落实

### 17.1 第一稿的独立评审（2026-09-29）
评审结论：Ready with fixes（Critical 0、Important 11、Minor 29），带三个竞态的 spike（S1a、S1b、S2）和一个耗时的 spike（S3）。控制者接受全部发现并逐条裁定，第二稿照裁定落实：

| 编号 | 问题 | 落实 |
|---|---|---|
| I1 | 先判定、后锁父行：两位管理员能互相降级到一个不剩；被移出的人还能建项目、删工作区 | 3.6 约定二"先锁父行，再判定"，成员关系和角色的一切改变用 `FOR NO KEY UPDATE`，按 id 寻址的写先读、锁父行、再重读，锁语句带 `deleted_at IS NULL`；6.7 改写；3.7 删去"改的人自己是管理员"的旧理由，改由约定二推出；交错测试 2、5、6；不用按行的写法（它死锁） |
| I2 | 后端 Phase 的顺序让故事的前置数据到不了 | 第 12 节重排为 platform → workspaces → invitations → projects → memberships → deactivation → states-and-labels；第 2 节每个故事标出接口和页面的 Phase；W2 改写为"删除一个，提升第二位管理员后离开另一个" |
| I3 | 前端 Phase 让挂载时的旧接口请求留在页面上 | P8 `web-data-layer` 先迁走全部类型、service、store、权限 store 和挂载路径上的取数，加上关键词规则；页面在 P9–P11；W11 的页面版本移到 P11；"10 处实例消失"写在 P8 的完成线 |
| I4 | 跨模块的错误码违反 `apitest` 的前缀规则 | 平台码 `forbidden`（3.4、5.3）；M2 设计 3.11 的前缀规则修订为"前缀是 nerve 的一个模块"，`apitest` 核对（11.7、3.20 的 P1、9.4） |
| I5 | "与 Plane 相同、没有恢复"不实 | `nerve workspaces reactivate-member`（3.11、W12）；3.7、3.9、8.6、16 和 README（8.7）改正；不照搬 `create_project_member`，理由在 3.11 |
| I6 | 标签的层级在并发下能成环 | 标签的全部写入先以 `FOR NO KEY UPDATE` 锁项目行（3.16、3.6 的表）；交错测试 11 |
| I7 | 3.6 的锁表有缺口 | 3.6 的表覆盖每个写操作；约定三加上建项目；约定五（批量语句按扫描顺序加锁）；停用一次锁住全部项目；3.17 的守卫的写；交错测试 5、10 |
| I8 | M3 的事件抢在 M2 设计 3.15 和 M4 的交接之前 | M3 不引入事件（3.3）；三个连带经 `ProjectCascade`（6.5）；第一稿的 11.1 改为第 11 节开头的说明；13.1 列出 M4 交接第 6 节"M3 不涉及"；13.2 给 M4 的说明 |
| I9 | 3.4 的收紧表混了三类 | 邮箱一行删除（与 Plane 相同，3.4、4.11）；谁能邀请交决策点 4；邀请角色的上限写进 3.8 |
| I10 | 决策点 1 夸大了 A 与 B 的差别 | 决策点 1 重写：链接的保密是主要的控制，四个选项，建议 A+(a) 且查看不显示邮箱，(b) 的代价写实；3.8、5.2、8.1、8.4 随之 |
| I11 | P1 超过规模上限、混了两个评审敏感的内容；后端估计偏低 | P1 拆为 platform 和 workspaces（第 12 节）；7.11 以 M2 每个操作约 343 行为上限重估（约 12,000 行），写明下调的依据 |
| M1 | 引文的行号 | 按评审改正（`views/workspace/invite.py`、`serializers/workspace.py`、`db/models/state.py` 各处） |
| M2 | React Router 的说法不对 | 3.10 改为：Plane 的页面总转小写，只差大小写的地址容易混淆 |
| M3 | `is_triage` 的理由不对 | 3.17 改为 `State.save()` 和建项目的 `bulk_create` 都不设它，只有迁移 `0063` 回填 |
| M4 | 一组中最后一个状态 | 409 `project.state_last_in_group`（3.17、5.3、4.11）；P6 故事改写 |
| M5 | 新状态的 `sequence` | "非分诊状态的最大值加 15000"，默认之后是 70000（3.17、P6） |
| M6 | 已归档项目的状态 | 照 Plane 不列出（3.17、P6、9.2） |
| M7 | 修改时的负责人、默认负责人 | 3.19 的规则、4.11；P3 故事在 members 页设它们；页面位置照 Plane（7.6） |
| M8 | `logo_props` 的默认值与封闭结构 | 字段都可选，`{}` 表示没有图标（3.19、5.2、4.6） |
| M9 | 对象数组的请求体 | `apitest` 的请求体用例扩展到数组里的对象（5.2、9.4，P3） |
| M10 | 物理级联只有部分索引 | 不带条件的外键索引（第 4 节开头、4.3–4.10、4.12） |
| M11 | 同一列两个部分唯一索引的命名 | `<表>_<列>_<含义>_key`（第 4 节开头） |
| M12 | 邀请邮箱的 CHECK | 加上空白的条件（4.4） |
| M13 | "读 `data` 的调用方不受影响" | 改为"v0 中不分页"，并写明以后分页是接口的改动（3.12、11.3） |
| M14 | 列表的顺序 | 3.12 的顺序表 |
| M15 | 7.11 的合计 | 加"预计改到的文件"一列，合计重算为约 380（7.11） |
| M16 | 受邀者的新手引导 | 资料一步之后完成（7.4、W6） |
| M17 | 修改的串行 | `core/lib/one-at-a-time.ts`（7.1） |
| M18 | 组合中的事后登记，`Provide` 里放了用例 | `Provide` 只返回适配器，用例在 `New`，`project.New` 在 `workspace.New` 之前，构造之后不登记（6.6、11.6） |
| M19 | 操作名常量的位置 | 各模块声明，`bootstrap` 的完整性测试（3.4） |
| M20 | 矩阵缺少已不是成员的身份 | 9.2 加"已被移出""工作区已删除""P-前" |
| M21 | 访客对公开项目 404 | 保留 404，登记差异（3.19、4.11） |
| M22 | 第二个 nerve 上的 W6 | 邀请经第二个 nerve 创建（9.6、W6）；README 写明同一部署共用密钥文件（3.8、8.7） |
| M23 | 停用删除的邀请 | 全部，含已忽略的（3.9、W9） |
| M24 | 加入项目的 `sort_order` | 65535（3.18、P2） |
| M25 | 路径参数的名字 | `{workspace_member_id}`、`{project_member_id}`（5.1） |
| M26 | 保留名单不向前兼容 | 预留 4 个段、三段的测试、冲突的处理（3.10） |
| M27 | `updateProjectMember` 的规则行 | `{Admin}` 加工作区管理员的例外（3.5、9.2） |
| M28 | 已忽略的邀请 | 占着邮箱直到删除，与 Plane 相同；页面显示（3.8、W4、7.5） |
| M29 | Plane 移出检查的两处错误 | 3.7 两处都写 |

**与评审的建议不同或做了选择的地方**：
- **I6**：评审给了两种做法（锁涉及的标签行，或锁项目行），选锁项目行，并扩展到状态的全部写入：默认状态、组的非空、标签的层级都是项目范围的不变式，锁归集合的拥有者（3.16）。
- **I7**：停用在"写明例外"和"一次锁住全部"之间选后者，全局的 `id` 顺序没有例外（3.6）。
- **I8**：端口名取 `ProjectCascade`，三个方法照评审。
- **I2、I11**：评审的起点是五段，本稿是七段后端：停用单独一段（评审敏感，与成员关系合在一起约 22 个任务）；平台一段带上建、列、看工作区，使它合并的是有故事验收的接口；降为访客的连带随项目在 P4，不让主干在 P4、P5 之间出现"工作区访客是项目管理员"。
- **M10**：在"加索引"和"M4 的清理显式删除子行"之间选加索引，M4 只需照做（13.2）。
- **M18**：`identity.Provide` 除了适配器还返回 `CredentialLock`：它是 M2 的协议步骤，只用 `identity` 的存储；放进 `identity.New` 会成环（6.6）。
- **M22**：选"经第二个 nerve 创建邀请"，另外在 README 写明共用密钥文件。
- **M26**：预留段和冲突的处理两者都写。
- **M8**：选字段可选，不为 `logo_props` 编一个默认图标。
- **评审的隐含决定 4**（`--admin-email`）：不再说"照搬"，登记为差异（3.11、4.11）。
- **评审给负责人的问题**：Q1 → 决策点 1（重写）；Q2 → 决策点 4（新）；Q3、Q4 按控制者的裁定不交负责人（I5 加命令，I8 用端口）。

### 17.2 第二稿的 Codex 对抗性评审（2026-09-29）
第二稿 `0d0f6ee5` 经 Codex 对抗性评审（[报告](reviews/M3-design-codex-adversarial-review.md)，`4627831b`）：Critical 0、Important 3（其中 I-3 只在决策点 4 选 B 时成立）、Minor 5，另有对决策点和架构问题的意见（报告 4.1、4.2）和前端的两条约束（报告 4.3），带七个 spike（S1–S7）。负责人裁定了四个决策点、批准了七个架构问题，并要求把评审中有价值的内容吸收进设计，为编码加固方向。控制者逐条核对之后全部接受；第三稿把每一条落成正文的规则，加上一个指定 Phase 的测试。"测试和 Phase"一栏是实现并验证它的地方。

**负责人的裁定**（2026-09-29）：

| 事项 | 裁定 | 落点 |
|---|---|---|
| 决策点 1 | A+(a)，公开的查看和邮箱不一致的拒绝都不显示被邀请的邮箱 | 3.8、5.2、7.4、8.1、8.2、8.4、第 10 节、W5、W6 |
| 决策点 2 | A，只有链接 | 3.8、7.4、7.8、7.10、第 10 节 |
| 决策点 3 | A，照 Plane 的页面 | 9.2、故事 P9、C10、第 10 节、13.2（M4） |
| 决策点 4 | A，只有工作区管理员 | 3.4、3.8、7.5、9.2、第 10 节 |
| 11.1–11.7 | 全部批准 | 第 11 节；3.20 的各行都已确定 |

**Codex 的发现**：

| 编号 | 问题 | 规则的落点 | 测试和 Phase |
|---|---|---|---|
| I-1 | 停用列举工作区之后，接受邀请还能让他的工作区集合变大；新工作区漏掉停用的锁、唯一管理员的检查和项目的连带，结果工作区没有管理员、项目里留着有效的成员关系（S1） | 3.6 约定六：工作区一侧的增长最先以 `FOR SHARE` 锁住账户行、在锁下重读 `is_active`；项目一侧的增长锁工作区成员行；收缩先改工作区成员行、在调用时列举项目；账户行只重读 `is_active` 和邮箱，不叫凭证复核。3.6 的取锁表、3.8、3.9、3.11、6.5 随之 | 交错测试 7（S1 的四种走法）、8、13–16（P6）；12（P3）；第 14 节的核对项 |
| I-2 | 恢复成员之后接受旧的访客邀请，覆盖了有效成员的角色：工作区没有管理员，访客仍是项目管理员（S2） | 3.8"邀请从不改变有效的成员关系"：已是有效成员时只消费邀请；恢复以前的成员行时取邀请的角色，访客时 `DemoteToGuest`；3.6 约定六"恢复不比新授予给得更多，也不比原来那一行更多"（加入恢复时取 `min(原来那一行, 现在的工作区角色)`，3.5；后一半由 17.3 的 I1 补上）；3.7；3.11 说明 `reactivate-member` 不删旧邀请（移出、离开删除它，17.3 的 M3）；4.11 两行 | 9.1 接受的用例测试（P3）；9.3 S2 顺序的集成测试和 W12 的旧邀请一段（P5）；9.1 恢复时的角色（P4）；故事 P5 的重新加入（P5） |
| I-3 | 若让成员邀请，邀请列表会把管理员邀请的令牌交给成员，成员注册、接受就是管理员（只在决策点 4 选 B 时成立） | 决策点 4 的"以后改的代价"写明读取范围的条件：列表、单个邀请、复制链接都要排除高于读者角色的邀请 | 决策点 4 裁定 A，风险不存在；9.2 的矩阵中成员对邀请的四个操作 403（P3） |
| M-1 | 跨模块的读取没有覆盖成员列表的资料和恢复命令；`Accounts` 只锁有效账户，与恢复停用账户的命令矛盾 | 6.5 `MemberProfiles`（无锁批量读，含停用的账户）、`Accounts` 交回状态、`ProjectMembershipCounts`；值在 `bootstrap/ports.go` 转换，不跨模块 JOIN（S6）；6.6 的顺序不变 | 三个端口的集成测试（P1 `Accounts`、P2 `MemberProfiles`、P5 `ProjectMembershipCounts`）；W12（P5） |
| M-2 | `bootstrap` 返回过的码补不了 `identity` 的 HTTP 测试进程的记录，`identity` 的 `apitest.Main` 会失败（S7） | 9.4 规则：声明在模块 X 的操作上的码，在 X 的 HTTP 测试包里经 `CheckResponse` 返回过；5.3 | `identity` 用 `fakeDeactivate.err` 的两条 409 透传测试（P6）；`workspace` 的 `project.sole_admin`（P5）；`project` 的 `workspace.not_found`（P4） |
| M-3 | 旧 `ProjectService` 缩到两个方法会丢掉 `projectIssuesSearch` 和它的三个调用方 | 7.3 旧 service 留三个方法及理由；7.10 例外 `until: "M4"`，`/search-issues/` 不需要例外；13.2（M4、M6、M7）；P11 的最终核对 | P8 的 `tsc`、`node tools/keywords.mjs`；P11 的第 6 个任务 |
| M-4 | W5 要求打开链接时就认出账户不对，而查看里没有邮箱 | W5 改为点"接受"得到 403 之后说明、按钮不再可用、提供退出；与 7.4 一致 | W5 的页面版本（P9） |
| M-5 | `logo_props` 的 CHECK 只查对象类型，不合 M2 设计 3.13 | 4.6 按 3.13 写全：键的集合、出现的每个键的类型、嵌套的 `emoji`、`icon`；第三稿 spike；3.19、5.2 | 9.3 的十个反例（含 S5 的值）和四个合法值（P4） |
| 4.3 第 1 条 | 页面按权限显示，却不按权限取数：成员页给成员取邀请，项目包装层给不是成员的人取子资源，都得到 403 | 7.1 规则"权限决定取数，不只决定显示"；7.3、7.5、7.6 | W4、P2 的页面版本（P9、P10）；9.5 的两个取数条件的 vitest（P8） |
| 4.3 第 2 条 | 已发出的修改在换了账户之后迟到地成功，旧页面仍会跳转、提示 | 7.1 规则：`inSession()` 移到 `core/lib/in-session.ts`，组件在 `await` 之后先核对；3.20 的 P8 行 | 9.5 的 `sessionGuard` 和两个删除组件的 vitest（P8、P9、P10）；W3 的会话切换端到端（P9） |
| 4.1 决策点 1 | 注册能用来试邮箱；(a) 改 (b) 之后留下已注册的账户 | 第 10 节决策点 1；3.8；8.2 | 写明的风险；`register_ip` 的限流沿用 M2 的测试 |
| 4.1 决策点 3 | B、C 的代价漏了 M4 的查询过滤和测试 | 第 10 节决策点 3 | 若以后改选，由 M4 的规则行和查询测试覆盖（13.2） |
| 4.2 11.1 | 另一种做法的好处写得太窄 | 11.1：分开轮换、影响范围更小 | — |
| 4.2 11.2 | 锁的顺序正确不等于锁的集合完整 | 11.2 加约定六 | 同 I-1 |
| 4.2 11.3 | 集合没有硬性的上限 | 11.3；第 16 节的新风险行；13.2（M8 的测量） | M8 的性能测量 |
| 4.2 11.4 | 时区校验读时区数据库 | 11.4：二进制内嵌时区数据库，不建端口 | 已有的 `TestNerveBinaryEmbedsTheTimeZoneDatabase` |

**控制者的核对**（在第三稿之前，对着第二稿和代码）：I-1 核对了第二稿 3.6 接受一行不锁账户、停用在开头列举；I-2 核对了第二稿 3.8 照搬 Plane `views/workspace/invite.py:193-194` 的角色覆盖；M-2 核对了 `apitest/problems.go` 的 `answered` 只在一个测试进程之内；M-3 核对了 `projectIssuesSearch` 有三个调用方；M-5 核对了 M2 设计 3.13 要求键集合和类型的 CHECK。其余各条的说法与第二稿的原文和代码一致。

**做了选择或超出评审的地方**：
- **I-1**：接受邀请照评审取接收账户行的 `FOR SHARE`；项目一侧的增长不锁账户行，靠工作区成员行和收缩的顺序串行，3.6 约定六说明为什么够；账户行只重读 `is_active` 和邮箱，不复核凭证（接受不签发凭证）。
- **I-2**：选评审推荐的"保留现角色并消费邀请"，不选"拒绝"；`reactivate-member` 不删除旧邀请（3.11；聚焦复核的 M3 证明通用的规则只在他有效时够，移出、离开现在删除发给他的待接受邀请，17.3）。另外把同一条原则推到两处评审没有提的地方，都是约定六"恢复不比新授予给得更多，也不比原来那一行更多"的推论：加入项目恢复以前的行时，角色取 `min(原来那一行的角色, 现在的工作区角色)`（第三稿取现在的工作区角色，聚焦复核的 I1 补上后一半，17.3；Plane 保留旧角色，登记差异，3.5）；接受邀请恢复为工作区访客时 `DemoteToGuest`，在 `min` 之下它是承重的。
- **M-1**：`reactivate-member` 照 Plane 允许停用的账户，是约定六唯一的例外，命令提示下一步 `nerve users activate`。
- **M-3**：选"留在旧 service"，不选"移到一个新的旧搜索 service"，理由在 7.3。核对时连带发现：新的 `Project` 没有 `is_favorite`，收藏的读取也要在 P8 删除（3.2，13.2 的 M7）。
- **M-5**：CHECK 查到嵌套的两个对象为止，它们的值都是字符串，结构到此查完。

### 17.3 第三稿的聚焦复核（2026-09-29）
第三稿 `34a89d21` 经聚焦复核：Ready with fixes，Critical 0、Important 2、Minor 9。复核带 Postgres 18.6 上的 spike（复核者在 `nerve-dev-db-1` 的临时库里做，事后删除）。结论是约定六的协议成立：它点名的每一对增长与收缩，两种顺序都串行，没有死锁，也没有丢失的不变式；两条 Important 在协议旁边，不在协议里。控制者接受全部发现并逐条裁定，没有一条是架构层面的：每一条收紧 M3 自己模块里的一条规则，或补全一张清单；已批准的加锁顺序、六条约定和组合都不变。每一条落成正文的规则，加上一个指定 Phase 的测试。

| 编号 | 问题 | 规则的落点 | 测试和 Phase |
|---|---|---|---|
| I1 | 第三稿的"加入恢复时取现在的工作区角色"让被项目管理员降为访客的人离开再加入，自己升回工作区角色（spike 17） | 原则"恢复不比新授予给得更多，也不比原来那一行更多"：加入取 `min(原来那一行, 现在的工作区角色)`；添加取请求的角色、接受取邀请的角色不变；恢复为访客时的 `DemoteToGuest` 写明是承重的。3.5、3.6 约定六、3.8、3.11、4.11（两半都写）、W12、故事 P5、17.2 | 9.1 恢复时角色的四行表（P4 第 11 个任务） |
| I2 | 创建邀请只持工作区 S，两位管理员按相反的顺序邀请重叠的邮箱时在唯一索引上死锁；排序之后后到的一方得到原始的 23505，9.3 的通则会把它翻成 409（spike 14a、14b） | 3.6 全局顺序注明只排除行锁的环；约定五的后一半"只持父行的共享锁而批量插入时按唯一键的顺序插入"，写明为什么排序而不是 N；3.8 一批全有或全无、按规范化后的邮箱排序插入、23505 翻译为 422 `invitations[i].email` 的 `duplicate`；9.3 通则的例外；3.6 取锁表；11.2 | 交错测试 18（P3 第 11 个任务）；翻译和下标（P3 第 5 个任务） |
| M1 | 忽略邀请怎样读邮箱没有写；在工作区的锁之后经 `Accounts` 读，与停用死锁（spike 16） | 3.6 约定一：`Accounts` 只作事务的第一把锁，中途读邮箱经不加锁的 `MemberProfiles`；忽略最先以 S 锁调用者的账户行；创建邀请的"已是有效成员"经 `MemberProfiles` 比较。3.3、3.6 取锁表、3.8、6.5 | 交错测试 19（P3 第 11 个任务；P6 第 5 个任务换成真实的停用再跑） |
| M2 | 降为访客与加入、添加的串行，恢复以前的行与停用的串行，都没有点名的测试（spike 12a、12b、5a、5b） | 3.6、9.3 的交错清单；第 14 节 | 交错测试 17（P4 第 12 个任务）；7 另跑恢复以前的成员行（P6 第 5 个任务），13、14 另跑恢复以前的项目成员行（P6 第 6 个任务） |
| M3 | 待接受的旧邀请让刚被移出的成员凭旧链接自己回来；3.11 的"旧邀请无害"只在他有效时成立（spike 9d） | 3.8"结束的成员关系不留下邀请"：移出、离开软删除这个工作区里发给他的待接受邀请（在已持有的工作区 N 之下、改成员行之前，邮箱经 `MemberProfiles`）；3.6 取锁表；3.11 改正，并写明 `reactivate-member` 为什么不需要改；4.11 和 3.20 的 P5 行 | 9.3 的 9d 顺序，移出、离开各一次（P5 第 14 个任务）；实现在 P5 第 2 个任务 |
| M4 | 添加在判定之前锁并确认目标，实现可能把 422 给了该得 403、404 的人，而 422 与 404 之差透露项目 id 存在 | 3.6 约定三"锁目标行只为取锁的顺序；目标的 422 在判定之后给出"；取锁表 | 9.2 `addProjectMembers` 无效目标的 PM、X 两格（P4 第 13 个任务） |
| M5 | 停用的邮箱从哪里来没有写，`LockedAccount` 没有邮箱；列举到而上锁时已删除的工作区会落进"0 行答 404"（spike 15） | 3.9：邮箱取自锁下的账户行（`LockedAccount` 加 `Email`）；上锁时已删除的工作区、项目跳过，不答 404。6.1、6.5、取锁表 | P6 第 7 个任务：spike 15 的情形，改邮箱先提交时删除发给新邮箱的邀请；实现在 P6 第 1–3 个任务 |
| M6 | W3 的会话切换端到端在换账户之后才放行请求，请求得到 401，没有 `inSession()` 也会通过 | W3、7.1、9.6：`route.fetch()` 在换账户之前，`route.fulfill()` 在之后 | P9 第 11 个任务，变异核对：去掉 `inSession()` 时它失败 |
| M7 | `is_favorite` 的删除清单漏了 `favorite.store.ts:284-287` 和 `favoriteProjectIds` getter，7.2、7.3、7.8 没提 | 3.2、7.2、7.3、7.8、13.2 的 M7 行 | P8 第 5 个任务；`tsc`、knip |
| M8 | `reactivate-member` 的例外同样让项目一侧的增长接纳停用的账户 | 3.11 写明这一状态下项目一侧的增长照常允许，直到 `nerve users activate` 或再次停用 | 写明的行为，没有新测试 |
| M9 | 六处文字 | 0.3 S5 的计数；4.6"三个键"；约定四和 11.2 的例子（添加项目成员、以别人为负责人建项目）；5.2 `ProjectMember` 只读 `project_members`；7.4 与 W5 一致（按钮不再可用）；7.10 `search-issues` 的不命中样例 | P8 第 14 个任务（关键词的样例）；其余是文字 |

**控制者的核对**：控制者读了复核的全文，逐条对照第三稿的原文确认问题存在；I1、I2、M1、M3、M5 另有复核者的 spike（17、14a 和 14b、16、9d、15），M2 的串行由 spike 12a、12b、5a、5b 证实；M6 对照了 M2 设计 3.5"每个请求都查一次会话"，M7 对照了 `favorite.store.ts` 和 `project.store.ts` 的代码。

**做了选择的地方**：
- **I2**：选排序，不选让创建邀请取工作区的 N：N 会让它成为约定二里唯一在 N 之下往父行下面加行的写。排序作为约定五的后一半写进约定，不加第七条约定，已批准的六条约定不变（11.2）。
- **M1**：忽略取 `Accounts` 的 S（控制者的裁定），不选"经 `MemberProfiles` 读"：两种回应一个形状，邮箱的核对相同。
- **M3**：选根上的修正（移出、离开删除待接受的邀请），不选只改文字；已忽略的邀请不删，它不能再被接受，留在列表里是管理员看得到的记录。
- **交错测试 19 在 P3**：P3 时停用还不锁工作区，停用一方由测试的会话按 3.9 的顺序取锁，P6 换成真实的停用再跑。
- **M5**：P6 第 7 个任务另加"改邮箱先提交"的一例，守住"邮箱取自锁下"。
- **任务数**：全部并进已有的任务，各段的任务数不变（P3 14、P4 16、P5 15、P6 9，合计 148）。
