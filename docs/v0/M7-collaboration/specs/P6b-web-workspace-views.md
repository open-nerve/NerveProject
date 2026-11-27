# M7 / P6b：网页的工作区视图与跨项目列表——工作区标签目录、视图页、列表页、另存与更新视图

> 状态：草稿，已照主控对起草时 §8 四件待定的裁定（D-29–D-32）与补记的照收修正；预检 [re-97](../../../reviews/re-97-M7P6b预检.md)（修正后通过：重要 5、次要 9）主控全部采纳，I1–I5、m1–m9 已照它的修法并入（另并入观察 o1、o3、o4、o6 的建议），待主控核对。基线 main `6dead17d`（P6a 的 noff，`6dead17d7aa199f8d124cfdb403eacc64aedb1e5`）。设计见 [M7 设计](../M7-design.md) §2（H13 `:40`、H14 `:41`、H20 `:47`）、§3.7（`:118-121`）、§5（`:192-211`）、§7（`:240`、`:242-243`、`:246`、`:248`）、§12（P6b 一行 `:309`）、§14；网页的长期规则见[总体设计 §7.2、§7.7](../../v0-design.md#77-stores-按会话分代长期有效m2-起)（请求体轮到时才算 `:638`）。调研与裁定在台账 `.superpowers/sdd/P6b-web-workspace-views/`：`research/scope.md`、`research/questions.md`（Q1–Q26）、`rulings.md`（D-1–D-32，本 spec 的 R1–R32 逐条对应；D-29–D-32 是主控 2026-11-27 对起草时 §8 四件待定的裁定，已并入）。负责人 2026-11-27 批了两件：D-16 加接口 `listWorkspaceLabels`（P6b 唯一的合同改动），D-8 不改合同（工作区视图的合同默认布局仍是 `list`）。P4–P6a 的裁定照旧约束 P6b。新写的代码注释一律写 “M7/P6b spec”（代码里已有 76 处 “P6b spec” 指 M6/P6b）。

缩写：`C` = `web/apps/web/core`；`W` = `web/apps/web`；`P` = `W/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]`（同 P6a）；`WV` = `W/app/(all)/[workspaceSlug]/(projects)/workspace-views`；`WC` = `C/components/workspace/views`；`VC` = `C/components/views`；`IL` = `C/components/issues/issue-layouts`；`PKG` = `web/packages`；`E` = `e2e`；`K` = `tools/keywords.json`；`HO` = `docs/v0/M7-collaboration/handoffs`；`D` = M7 设计；`NAV` = `api/modules/navigation.yaml`；`PRJ` = `api/modules/project.yaml`；`ISS` = `api/modules/issue.yaml`；`SV` = `server/internal/modules/project`；`BS` = `server/internal/bootstrap`；`PW` = 只读的 `plane/apps/web`；`P6aS` = [P6a spec](P6a-web-project-views.md)；`P6aR` = [P6a 阶段记录](../reviews/P6a-web-project-views-review.md)；`L` = 台账目录。

## 1. 目标与边界

P6 在设计 §12 分两段，P6b 是工作区一半（`D:309`）。P6b 交付：

- **接口**（§2.1，R16，负责人）：`GET /api/v0/workspaces/{slug}/labels`，照 `listWorkspaceStates` 的范围与形状；project 模块的用例、查询、处理器与测试，权限矩阵一行，可见性测试，接口故事一份。
- **工作区标签目录**（§2.2，R17，**L!**）：标签 store 按工作区持有目录、按项目给出；工作区视图页、个人页、草稿页挂载时读，不进外壳。
- **列表引擎的 `GLOBAL`**（§2.3，R1，**L!**）：新 store 类型与新范围臂 `"workspace"`，穷尽的分派，逐个列出的静默分派点。
- **工作区页的筛选与两种偏好**（§2.4，R2、R3、R5，**L!**）：一个筛选 store，默认视图的偏好存浏览器、已存视图用泛化的内存草稿；契约的 11 个筛选词。
- **默认视图**（§2.5，R4）：四行的一张表，各自的 `me` 词固定在查询里。
- **视图的数据层按容器**（§2.6，R9，**L!**）：P6a 的 service、store、写扩到工作区视图，删 `globalView`。
- **入口与控件矩阵**（§2.7，R12、R13）：`workspaceViewsOn`、`workspaceViewsCreatableBy`；访客只有默认四个。
- **视图页**（§2.8，R6、R7、R8、R11、R24）：以 `globalViewId` 为 key，默认视图不读、已存视图四种状态；只渲染表格（按行的项目判可编辑、不给快速新建）；Plane 的页头；空态。
- **另存与更新视图**（§2.9，R10，**L!**）：共用的更新一半、按容器的另存一半；“更新视图”的请求体整个在轮到时从草稿算（O1 的结构守护）。
- **列表页**（§2.10，R13）与**弹窗、宿主、行菜单、删除按容器**（§2.11，R14、R15，**L!**）。
- **个人页的按标签分组与标签筛选**读目录（§2.12，R18，**L!**）。
- **删除**（§2.13，R20、R21，**L!**）：旧 store、旧 service、旧类型、旧帮手、旧组件；`old-view-urls` 扩到工作区旧地址。
- **e2e**（§2.15，R22）：H13 (API) 一份；H13 (page)、H14 (page)、H20 (page)；S2 的移除冒烟整份删；I19 两处改写。
- **P6a 的收尾回填**（T1，R27）。

**P6b 不做**：
- 命令面板（R19）：四条导航命令照旧，不加工作区视图的菜单、搜索与新建命令。
- 收藏与最近访问（P7）；视图的收藏星。
- 合同的其余部分、CI 配置：不改；CI 余量只报告（R23）。工作区视图的合同默认布局（`NAV:637-644` 的 `list`）不改（R8，负责人）。
- 富筛选 `invalid.tsx` 的英文：仍是 M8 的候选（P6aR §7）。
- `project/applied-filters/{members,date}.tsx` 与 `common/applied-filters/` 的重复（P6aR §7 的 O4）：`common/` 的两份仍有读者，留着，交 M8（§2.13）。
- 状态筛选里同名状态的项目标识（R31）、状态 store 回落的归档守护（R32）：交 M8（§5）。

**P6b 之前 main 的样子**：工作区视图的页面仍读 Plane 的旧地址（`C/services/workspace.service.ts:13`、`:21`、`:29`、`:37`，经 `C/store/global-view.store.ts`），布局用裸的 `useSWR`（`WV/layout.tsx:22-26`）、不以 id 为 key、读整份列表判存在，“不存在”是写死的英文（`:37-48`）；视图页只有 `PageHead`（`WV/[globalViewId]/page.tsx:14-29`），路由判为 `DEFAULT`（`C/hooks/use-issue-layout-store.ts:28-30`），不列工作项；页头的 “Add view” 不看角色（`WV/header.tsx:100-102`），面包屑 “Views” 不是链接（`:72-76`）；行菜单的编辑删除给所有人（`WC/view-list-item.tsx`）、复制只处理成功（`WC/quick-action.tsx`、`WC/default-view-quick-action.tsx`，`K:3424-3437` 两个 `copy-hook` 例外）；工作区 HOC 照搬 Plane 的锁、所有者与 `rich_filters`，只有个人页读它、不给保存（`C/components/work-item-filters/filters-hoc/workspace-level.tsx`、`C/components/profile/profile-issues.tsx:59-68`）；个人页按标签分组时组名冷则 “—”、标签筛选只有已持有的标签（`IL/group-columns.tsx:266-284`、`workspace-level.tsx:55-58`）；合同没有工作区级的标签操作。

**P6b 之后 main 的样子**：工作区的管理员、成员、访客从侧栏进“所有工作项”，四个默认视图跨调用者是有效成员的全部未归档项目列出工作项（只渲染表格，每行按自己的项目判可编辑），改显示存浏览器、再进来还在，改筛选只在这次进入；管理员与成员另见列表页与切换器里的已存视图、新建、另存（默认视图另存带它的 `me` 词），所有者“更新视图”（等回答、在途停用、请求体轮到时从草稿算），所有者与工作区管理员删；访客打开已存视图的地址看到“不存在”、不发请求；标签的筛选选项、按标签分组的组名与行上的标签来自一次工作区标签目录的读；旧的工作区视图层全部删除。

## 2. 交付合同

### 2.1 接口 `listWorkspaceLabels`（R16，负责人）

- **契约**（`PRJ`，紧接 `listWorkspaceStates` 的路径 `PRJ:597-622`）：`/api/v0/workspaces/{slug}/labels`，`get`，`operationId: listWorkspaceLabels`，`tags: [project]`，`summary: List the labels of the projects the caller is a member of`。描述照 `listWorkspaceStates` 与 `listLabels`（`PRJ:624-646`）写：工作区的有效成员；他是有效成员的、未删除未归档的项目的标签，顶层与其下的都在；按项目 id、再按 `sort_order`（小的在前）、再按 `id`；每个项目的与 `listLabels` 列的相同；他看得到而不是成员的项目不加任何标签；工作区不存在、已删除或他不是有效成员答 `workspace.not_found`；集合型，不分页。`security: [{bearer: []}]`，`x-problem-codes: [workspace.not_found]`（没有 `forbidden`：三种角色都可读），`200` 回 `LabelList`（`PRJ:1381`），`default` 回 `Problem`。`api/openapi.yaml` 在 `:120-121`（states 的 `$ref`）旁加一行 `$ref`。`make gen` 之后 `make gen-check` 过。
- **规则**：`SV/domain/actions.go` 加 `ActionWorkspaceLabelList shared.Action = "workspace_label.list"`（注释照 `:57-59`），并进 `Actions()`（`:74-80`）；access 的规则表 `server/internal/modules/access/domain/rules.go:86` 旁加 `"workspace_label.list": {Level: LevelWorkspace, Roles: Admin, Member, Guest}`（注释照 `:84-85`）；`rules_test.go:37` 旁加一行 `{allowed, allowed, allowed, invisible, invisible, invisible, forbidden}`。
- **用例**：新 `SV/app/list_workspace_labels.go`（照 `list_workspace_states.go`，40 行）：`ListWorkspaceLabels{workspaces WorkspaceDirectory; labels WorkspaceLabelLister; auth shared.Authorizer}`，`Execute(ctx, slug)` = `shared.RequireActor` → `findWorkspaceAndDecide(…, domain.ActionWorkspaceLabelList)` → `labels.ListWorkspaceLabels(ctx, ws.ID, actor.UserID)`；端口 `WorkspaceLabelLister` 进 `SV/app/label_ports.go`（照 `state_ports.go:23-29`）。新测试 `list_workspace_labels_test.go`（照 `list_workspace_states_test.go`）：没有身份、工作区找不到、规则拒绝、成功各一格。
- **查询**：`SV/adapter/postgres/queries/labels.sql` 加 `ListWorkspaceLabels :many`，照 `states.sql:81-92`：`labels l JOIN projects p ON p.id = l.project_id JOIN project_members m ON m.project_id = p.id`，`p.workspace_id = sqlc.arg(workspace_id) AND p.deleted_at IS NULL AND p.archived_at IS NULL AND m.member_id = sqlc.arg(user_id) AND m.is_active AND m.deleted_at IS NULL AND l.deleted_at IS NULL`（本仓的写法，同 `states.sql:89-90`），`ORDER BY l.project_id, l.sort_order, l.id`。`SV/adapter/postgres/labels.go`（122）加 `ListWorkspaceLabels`；`labels_test.go`（208）加一格：次序、已归档项目的不在、不是成员的项目的不在、已删除的标签不在、不在职的成员身份不算。`SV/adapter/postgres/failures_test.go`（308）的 `TestAFailedReadIsAnErrorNotAnAnswer` 对 store 的每个读各有一行（`ListWorkspaceStates` 在 `:135-136`）：加 `ListWorkspaceLabels` 一行（失败的读是错误、不是空列表），头注释 `:41-42` 的 “nor no labels, which listLabels would answer as none” 补上 `listWorkspaceLabels`。
- **处理器**：新 `SV/adapter/http/workspace_labels.go`（`labels_test.go` 已 290 行，测试另起 `workspace_labels_test.go`）：`ListWorkspaceLabels` 回 `gen.ListWorkspaceLabels200JSONResponse`，与 `labels.go:12-21` 同一种换算；`handler.go` 加 `ListWorkspaceLabelsUseCase`（`:118-124` 旁）与字段（`:166-167` 旁）；`handler_test.go` 的替身加一个字段（`:83-84`、`:260-264`、`:280` 旁）；`SV/module.go:351` 旁接线。
- **权限矩阵**：`BS/permission_matrix_labels_test.go` 的 `labelMatrixRows()`（`:66`）加一行 `{op: "listWorkspaceLabels", request: toWorkspace(http.MethodGet, "/labels", ""), cells: inWorkspace(cellOK, cellOK, cellOK), check: listsTheWorkspaceLabels}`，检查照 `permission_matrix_states_test.go:174-200` 的 `listsTheWorkspaceStates`：各账户列到的恰是他是成员的未归档项目的 `matrixLabels`，按项目 id。不加顶层测试名：它是 `TestPermissionMatrix` 的子测试，而 `TestPermissionMatrix` 在桶 46、第 1 路（`tools/go-test-shard.sh:15-18`），新行给它加三个并行的子测试与一个检查，量很小，检查点 0 的第 1 路基线已含它。
- **可见性**：新 `BS/workspace_labels_visibility_test.go`，两个顶层测试照 `project_visibility_test.go:75`、`:108`：`TestListingWorkspaceLabelsListsEachProjects`（矩阵的每个账户：工作区的列表等于各未归档项目的 `listLabels` 按项目 id 接起来，答 403 的算没有；与先例 `project_visibility_test.go:75` 的两处不同要照实写：`listLabels` 对已归档的项目照常列出（`BS/permission_matrix_labels_test.go:71-74`），所以已归档的项目不接进来，而不是像 states 那样接进来答空；对不是成员的账户 `listLabels` 答 403（`:69-70`））与 `TestWorkspaceLabelsKeepToTheirWorkspace`（`newMemberWorld`：acme 的列表只有 Web、Ops 的，beta 的只有 Lab 的；已删除的标签不在）。名字的桶（`printf %s <name> | cksum` 的第一个数对 100 取余）分别是 12 与 1，都在第 0 路（桶 0–45，`tools/go-test-shard.sh:18-23`）：第 1 路是紧的一路（P6a 最终分支 CI 的 bootstrap 第 1 路 760.8 秒），不往里加顶层测试。
- **接口故事**：新 `E/stories/collaboration/h13-workspace-labels.spec.ts`，标题 “H13 (API): …”（m7-pages 不收：它只收 `-page(s).spec.ts`，`E/playwright.m7-pages.config.ts:10-11`；默认配置在 chromium 跑）：成员列到他两个项目的标签、次序同上，等于两次 `listLabels` 接起来；一个项目归档之后它的标签不在；他不是成员的项目不在；访客只列他所在项目的；别的工作区答 404 `workspace.not_found`。不过 200 行。
- **设计**：§5 的接口表加一行（主控在 T1 改）。

### 2.2 工作区标签目录（R17，L!）

- **service**：`C/services/project/labels.service.ts` 加 `listInWorkspace(slug)`（`GET /api/v0/workspaces/{slug}/labels`，回答原样）。
- **store**（`C/store/label.store.ts`，208）：照状态 store 的先例（`C/store/state.store.ts:63-64`、`:93-99`、`:114-121`、`:153-155`、`:220-224`）：
  - 新 `workspaces = new ReconciledByKey<Label[]>()`，按工作区 id；`fetchWorkspaceLabels(workspace: WorkspaceRef)` 以 slug 读、以 id 存。
  - `labelMap` 先放各工作区的目录、再放各项目自己的列表（同一个标签取项目列表的那一份，同 `stateMap`），只给项目 store 给得出项目的。
  - `getProjectLabels(projectId)`：项目自己的列表；没有时，若项目在目录的范围里（调用者是成员：`member_role !== null`，且未归档），取它所在工作区的目录里这个项目的那些；都没有是 `undefined`。（比状态 store 多一个范围的判断：目录不含已归档的项目与不是成员而看得到的项目（`member_role === null`，如公开项目），不能把“目录里没有”说成“这个项目没有标签”；两半各有一格。）目录是一次读的快照，而“在范围里”按项目 store 此刻判：读目录之后才加入的项目，到下一次目录的读（三页之一再挂载）之前回落得 `[]`；`prepareListCatalog` 自己兜得住（行上有不认识的标签就读项目的，`issue-list-catalog.ts:113-119`），行上的标签（`C/components/issues/use-issue-labels.ts:11`）与命令面板的标签菜单（`C/components/power-k/ui/pages/context-based/work-item/labels-menu.tsx:29-33`）没有，这是已知的限制。
  - 写（建、改、移、删）在 nerve 确认之后同时写进项目的列表与工作区的目录（`label.project_id`、`label.workspace_id`），同 `state.store.ts:220-224`。类的说明（`:65-72` 的“the workspace's across its projects are M7's”）照实改写。
  - 读者都不变：行上的标签（`C/components/issues/use-issue-labels.ts`）、分组的列（`IL/group-columns.tsx:266-284`）、筛选的选项、`prepareListCatalog`（`C/store/issue/helpers/issue-list-catalog.ts:113-119`：目录在时不再逐项目读）都经 `labelMap`、`getProjectLabels` 拿到目录里的标签。
- **读**：新 `C/hooks/use-workspace-label-catalog.ts` 的 `useWorkspaceLabelCatalog(workspaceSlug, enabled)`：`useSessionSWR(enabled && workspace ? ["WORKSPACE_LABELS", workspace.id, workspace.slug] : null, (id, slug) => fetchWorkspaceLabels({ id, slug }))`（键的形状同 `C/layouts/auth-layout/use-workspace-fetch.ts:49-50` 的 `WORKSPACE_STATES`），工作区按路由的 slug 取。三个读者，各自的门：
  - 工作区视图页的布局根（§2.8，三种角色；只在列表画出时，也就是默认视图与读到的已存视图）；
  - 个人页 `C/components/profile/profile-issues.tsx`（门是它已有的 `allowed`：管理员或成员，`:46-47`）；
  - 草稿页 `C/components/issues/workspace-draft/root.tsx`（门是 `drafts.canRead(ws)`，`:23`、`:27`）。
  不进工作区外壳（`use-workspace-fetch.ts`），通知的预览照旧按项目读（P5b T9）。`.oxlintrc.json` 的 swr 名单（`:205-219` 那一组）收入这个文件。
- **挂载清单**：`E/fixtures/mounts.ts` 加 `WORKSPACE_LABELS = ["GET /api/v0/workspaces/{slug}/labels"]`（注释：工作区视图页、个人页的工作项、草稿页各在挂载时读一次，按各页的门），头注释 `:5-16` 的已读、不读清单照实补一句 M7 P6b。
- **S2**（`E/stories/smoke/s2-web-app.spec.ts`）：管理员与成员的 `profileVisits`（`:128-137`）与 `draftsVisit`（`:148-151`）各加 `...WORKSPACE_LABELS`，`:128` 的注释（“no preference or label-catalog fetch”）照实改；访客的行（`:208-219`，个人页与草稿页都不读）不变。

### 2.3 列表引擎的 `GLOBAL` 与 `"workspace"` 范围（R1，L!）

- **两个名字**：页面的 store 类型叫 `GLOBAL`（Plane 与 `HO/M4-closeout.md:54` 的原文，路由参数 `globalViewId` 同源）；引擎的范围臂叫 `"workspace"`（设计 `D:246` 的 “`WORKSPACE` 范围”，对服务端 `WorkspaceList`）。两份文档说的是两层，不矛盾。
- **枚举**：`EIssuesStoreType.GLOBAL = "GLOBAL"`（`PKG/types/src/issues/issue.ts:14-24`）。
- **范围臂**（`C/store/issue/helpers/issue-list-read.ts:16-32`）：第三臂 `{ scope: "workspace"; workspaceSlug; workspaceID; query: WorkspaceIssueQuery }`。`C/services/issue/issue-list.service.ts`（28，`:5-6` 两个查询类型旁）导出 `WorkspaceIssueQuery = NonNullable<operations["listWorkspaceIssues"]["parameters"]["query"]> & { group_by?: never; sub_group_by?: never }`（二选一里选这一种，注释写明理由：生成的查询没有分组，`never` 的两个字段让读 `scope.query.group_by` 的六处——`issue-list-read.ts:76`、`:135`，`issue-list-catalog.ts:92`，`base-issues.store.ts:96`，`C/lib/issue-membership.ts:90`、`:103`——不改即得 `undefined`，`"sub_group_by" in query` 的两处——`issue-membership.ts:41`、`issue-list-catalog.ts:93`——同理），`IssueListService.listWorkspace(slug, query)`（`GET /api/v0/workspaces/{slug}/issues`，`ISS:384-404`）收它。
  - `scopeRole`（`:42-49`）：不是项目臂时取工作区角色，写法不变。
  - `allowsRole`（`:52-56`）：条件改为 `scope.scope === "member" || (scope.scope === "project" && scope.query.archived === "true")` 要管理员或成员，其余（项目的未归档列表、工作区列表）有角色就过（工作区列表三种角色都可读，`server/internal/modules/access/domain/rules.go:187`）。
  - `readPage`（`:82-107`）：工作区臂调 `listWorkspace`，答的是平铺的一页（今天 `:82-88` 的二选一改成按三臂分；`relation` 只在成员臂）；“当前”的项目集是 `query.project_id ?? []`（同成员臂）；`captureRead` 的 `member`（`:71`）对工作区臂为假（同身份）；行的范围核对：`workspace_id` 相同，`project_id` 给了时行的项目在其中。
  - 绑定（`C/store/issue/helpers/issue-list-binding.ts`）：`covers`（`:62-69`）对工作区臂同成员臂（没给 `project_id` 即整个工作区）；取数的项目集与身份（`:108-112`）同成员臂，`identity` 的 `member` 为假（工作区列表不随调用者的成员身份水位变，`C/store/issue/permission-reads.ts:107-114`）。`context`（`:54-61`）把范围整个展开成 `ListContext`，所以下一条的 `ListContext` 必须同时加臂。
- **`ListContext` 与“行在不在这张列表里”**（I1：漏了它，“分配给我”“我关注的”与任何带状态、负责人、创建者、提及、关注条件的工作区页在行内编辑或新建之后列错，到下一次重取才对）：
  - `C/lib/paged-groups.ts`（445，已过 400，只动两处、别的不碰）：`ListContext`（`:53-56`，独立的两臂联合）加第三臂 `{ scope: "workspace"; workspaceSlug: string; workspaceID: string; query: WorkspaceIssueQuery }`（+1 行）；`applyGroupChange` 里“别的工作区的确认不影响这张表”的短路（`:332`，今天只认 `"member"`）改为 `context.scope !== "project"`，上面 `:329-330` 的注释照实改（“member scope” → 跨项目的范围）。除这一臂的一行之外文件不变长。
  - `C/lib/issue-membership.ts`（112）的 `issueMembership` 今天只有两支（`:54-66` 项目支、`:67-75` 成员支）。加工作区支：工作区相同、未归档、`project_id`（同成员支，`:69-71`）；加上 `state_id`、`assignee_id`、`created_by_id`（同项目支 `:59-61`，认 `me`）；`mention_id`/`subscriber_id` 有值即“未知”（同项目支 `:66`，确认之后走重读、不直接插入或移出）；没有 `relation`（`:72-74` 只在成员支）。项目支与工作区支共用的三条收成文件里的一个小函数，不复制。之后的公共部分（优先级、标签、日期、`sub_issue`、`state_group`）照旧对三臂都算。默认视图自己的词（§2.5）本来就在范围的查询里，所以“分配给我”的 `assignee_id: ["me"]` 由这一支判。
  - 测试替身 `C/store/issue/fake-issues.ts`（208）的 `listContext`（`:105-112`）是项目臂，不改；工作区的格需要时在它旁边追加一个工作区的帮手（只追加）。
- **列表 store**（新 `C/store/issue/workspace/`）：
  - `issue.store.ts` 的 `GlobalIssues extends BaseIssuesStore<GlobalIssuesFilter>`，先例 `ProfileIssues`（`C/store/issue/profile/issue.store.ts`）。不继承 `PagePreferencesIssues`：它的 `fetchPage` 只造项目的范围（`C/store/issue/helpers/page-preferences-issues.store.ts:30-43`）。`viewFlags = { enableQuickAdd: false, enableIssueCreation: false, enableInlineEditing: true }`。`fetchIssues(slug, globalViewId, loader, options)`：工作区 id 由工作区 store 按 slug 给（没有即不读）；页面 id 见 §2.4；查询 = `{ ...workspaceListQuery(toWorkspaceFilters(filters.richFilters), filters.displayFilters), ...ownTerm(globalViewId), limit: options.limit }`（默认视图的词放最后，§2.5）；`select({ scope: "workspace", … }, options).fetchFirst()`。另有 `fetchNextIssues(slug, group?)`、`fetchIssuesWithExistingPagination(slug, globalViewId, loader)`、`isPreferenceQueryCurrent(slug, globalViewId)`（当前列表是这个工作区的工作区臂、上次读用的是这一页此刻的偏好；先例 `C/store/issue/project-view/issue.store.ts`）。
  - `filter.store.ts` 的 `GlobalIssuesFilter`（§2.4）。
  - 在 `C/store/issue/root.store.ts:220-221` 旁构造。上限由 340 放宽到 350（R1，只为这两个字段与它们的构造）。
- **分派点**（每处单测一格、变异一行；漏了多数是静默的；第 9 点在 T9，其余在 T8）：
  1. `C/hooks/store/use-issues.ts`（118）：`TStoreIssues` 加一行、穷尽的 `switch` 加 `case`。
  2. `C/hooks/use-issues-actions.tsx`（353）：一个导入、一行无条件的调用（`useGlobalIssueActions()`）、一个 `case`；不多别的。
  3. `C/hooks/use-issue-layout-store.ts:28-30`：`globalViewId` → `GLOBAL`（注释改写）。
  4. `C/hooks/store/use-page-issue-filters.ts`（42）：联合与 `switch` 加 `GLOBAL`，以路由的 `workspaceSlug`、`globalViewId` 算页面 id 读 `globalIssuesFilter.getIssueFilters`。
  5. `C/components/issues/use-page-display.ts`（62）：`PageDisplayStoreType` 加 `GLOBAL`；显示的选项对 `GLOBAL` 一律按表格取（`layoutOptions.spreadsheet`），不按持有的布局（读布局在 `:47`，按布局取选项在 `:55`；R8，§2.8）。
  6. `PKG/constants/src/issue/filter.ts:240-246` 的 `ISSUE_STORE_TO_FILTERS_MAP` 加 `GLOBAL` → `workspace_issues`（§2.4）。
  7. `PKG/constants/src/issue/common.ts:55-60` 的 `TCreateModalStoreTypes` 加 `GLOBAL`（空态的新建以它打开，§2.8）。
  8. `IL/spreadsheet/base-spreadsheet-root.tsx:29-33` 的联合加 `GLOBAL`。
  9. `IL/empty-states/index.tsx:22-37` 加 `case GLOBAL`（§2.8 的空态）。这个 `switch` 有 `default: return null`（`:35-36`），加枚举不动它也编译，漏了就什么都不画：所以它在 T9（空态的组件同一提交），单测一格、变异一行（去掉 `case`）。
  10. `C/hooks/use-issue-list-mount.ts`（68）：对 `GLOBAL`，`page` 一段取路由的 `globalViewId`（同一工作区两个页面不共用一个键）；角色取工作区角色（`getWorkspaceRoleByWorkspaceSlug`）；`allowed` 为有角色；项目集取这一页筛选的 `toWorkspaceFilters(…).project_id ?? []`；`identity(workspaceID, projectIDs, false)`。
  11. 行的快捷操作 `ProjectIssueQuickActions`（`IL/quick-action-dropdowns/project-issue.tsx:178-182`）：**不改**。工作区的行照个人页走 `PROJECT`（编辑与复制经项目页的对话框写，写入经 `covers` 进工作区列表）；单测一格钉住“在工作区页上编辑一行，工作区列表里那一行跟着变”。
  - **不接**（各写一句理由，不是遗漏）：`IL/command-host.tsx:30-36` 的 `DIALOG_STORES`（只管分组头的“+”，`IL/list/headers/group-by-card.tsx:95`、`IL/kanban/headers/group-by-card.tsx:110`；表格不分组）；`C/hooks/use-group-dragndrop.ts`（表格不拖）；列表、看板、日历三个根的联合（只渲染表格）；`IL/utils.tsx:24-25` 的 `isWorkspaceLevel`（表格的路径上没有读者）；M6 的两张表；`usePageFilterExpression` 的联合（同 P6a R10，§2.4）。
- **命令**：新 `C/hooks/use-global-issue-actions.ts`，成员写全：`fetchIssues`、`fetchNextIssues`、`createIssue`、`createIssueAsGiven`、`updateIssue`、`removeIssue`、`archiveIssue`、`updateFilters`。建、改、删、归档同个人页（在给定的项目里，写进每一个显示它的列表）；`updateFilters` 不看调用方给的 id（表格传的是路由的项目 id，`base-spreadsheet-root.tsx:97-104`，工作区路由上是 `""`），用路由算出的页面 id，经新 `C/hooks/use-global-preferences-follow.ts` 跟进（`usePagePreferencesFollow` 的 `capture`、`listed`、`relist`，先例 `C/hooks/use-project-view-preferences-follow.ts`）。故意不给：`addIssuesToView`（没有日历的某一天）、`dropIssue`（表格不拖）。
- **穷尽**：两个 `switch` 照 P6a 的写法（`satisfies never`），漏一个值是编译错；运行时的未知值仍抛 “Unsupported issue store”。
- **钉 `GLOBAL` 退役的测试（两份文件三处）**：`C/hooks/store/use-issues.test.ts:26` 与 `C/hooks/use-issue-layout-store.test.tsx:127` 的拒绝表去掉 `"GLOBAL"`（`"unknown"` 照旧被拒）；`use-issue-layout-store.test.tsx:75-116` 的 `globalViewId` 一格期望 `GLOBAL`（三元式的末支 `DEFAULT` → `GLOBAL`；“在选定的项目建、不取列表”的断言不变）。

### 2.4 工作区页的筛选与两种偏好（R2、R3、R5，L!）

- **词与换算**（`C/lib/issue-filters.ts`，349，不过 400）：
  - `ListPageFilters = ProjectIssueFilters & Pick<ViewFilters, "project_id">`（列表页的筛选的公共形状）；`WorkspacePageFilters = Omit<ListPageFilters, "cycle_id" | "module_id"> & { cycle_id?: never; module_id?: never }`（照 `:198-199` 的写法）：契约的 11 个词——`priority`、`state_group`、`state_id`、`label_id`、`assignee_id`、`created_by_id`、`mention_id`、`subscriber_id`、`project_id`、`start_date`、`target_date`（`NAV:566-634`、`ISS:405-493`）。
  - `toWorkspaceFilters(expression)`：十个词经 `toProjectFilters` 的同一套校验，`project_id` 经从 `toMemberFilters` 抽出的 `projectIdsOf(expression)`（`:214-235` 的那一段，两处共用）。
  - `workspaceListQuery(filters, display)`：`{ project_id?, …十个词, order_by: order(display.order_by), sub_issue }`，永不发分组参数（工作区接口对它们 422，`ISS:396-402`）。
  - `toFilterExpression(filters: ListPageFilters)`（`:237-252`）收 `project_id`（`project_id__in`）；个人页的绕行随之删去：`C/store/issue/profile/filter.store.ts:27-32` 的 `cleanExpression` 变为 `toFilterExpression(toMemberFilters(expression))`，个人页的行为不变（它的单测不改）。
- **`PagePreferencesFilter` 的类型界**（`C/store/issue/helpers/page-preferences-filter.store.ts`，298，不过 300）：`F extends ProjectIssueFilters` 放宽为 `F extends ListPageFilters`（`:23`、`:33`、`:41`、`:123`、`:292`），不加方法、不改行为、显示的换算仍不导出（P6a R28）。项目页、迭代页、模块页、项目视图页的既有测试不改全过。
- **`GlobalIssuesFilter`**（`C/store/issue/workspace/filter.store.ts`）：`extends PagePreferencesFilter<WorkspacePageFilters>`，转换器 `toWorkspaceFilters`，store 类型 `GLOBAL`，服务是按页面 id 分路的 `WorkspacePagePreferences`（同文件，私有）：默认视图的页面 id → `DefaultViewPreferences`，其余（已存视图的 id）→ 泛化的 `ViewDraftService<WorkspacePageFilters>`。只多两个方法：`discard(pageId)`（转给那一路的 `discard`，并删 `filters[pageId]`，同 `C/store/issue/project-view/filter.store.ts` 的 `discard`）与 `adopt(viewId, view)`（转给草稿）。
- **页面 id**（新 `C/store/issue/workspace/page-id.ts`）：`workspacePageId(slug, globalViewId)`：默认视图的键是 `JSON.stringify([slug, key])`（同 `profile/filter.store.ts` 的范围键写法），已存视图是视图 id；`defaultViewOfPage(pageId)` 反过来给 `{ slug, key }` 或 `undefined`。理由：`PagePreferencesService.get(id)` 只收 id（`page-preferences-filter.store.ts:33-36`），默认视图的四个键在每个工作区都一样，浏览器里的显示要按工作区分。
- **默认视图的偏好**（新 `C/store/issue/workspace/default-view-preferences.ts`，`DefaultViewPreferences implements PagePreferencesService<WorkspacePageFilters>`，线上词，R3）：
  - `get(pageId)`：`filters` 是这次进入在内存里持有的（没有即 `{}`）；`display_filters`、`display_properties` 是浏览器里存的（没有即默认：`WORKSPACE_VIEW_CONTENT` 的显示与属性，§2.11）。
  - 存储：`@/lib/local-storage` 的 `storage`，键 `workspace_view_display`，条目按（账户 id、页面 id）；账户 id 取 `rootIssueStore.currentUserId`，没有就不读不写。读回逐字段校验：布局一律 `spreadsheet`、`group_by`/`sub_group_by` 一律 `null`、`order_by` 是字符串（否则默认；不合法的词由 `show` 与 `workspaceListQuery` 各自回到 `-created_at`）、`sub_issue`/`show_empty_groups` 是布尔、`calendar` 用默认；属性按 14 个键取布尔，缺的用默认。解析失败或存储不可用一律用默认；写失败吞掉（值仍在内存）。不复用 `handleIssuesLocalFilters`（`C/store/issue/helpers/issue-filter-helper.store.ts:206-305`）：它按控件词校验显示（`localDisplayFilters`），线上词会被丢。
  - `update(pageId, body)`：`filters` 只进内存；显示与属性进内存并写浏览器；返回整份。
  - `discard(pageId)`：丢这次进入的筛选（D-3：筛选只在本次进入）。
- **草稿服务的泛化**：`C/store/issue/project-view/view-draft.service.ts` 移到 `C/store/issue/helpers/view-draft.service.ts`（`git mv`，`NoViewDraftError` 同文件），按筛选类型参数化：`ViewDraftService<F extends ListPageFilters>`，构造时多收一个 `filtersOf: (filters: ViewFilters) => F`（项目视图原样，工作区视图丢掉 `cycle_id`、`module_id`——合法的工作区视图本来没有这两个键）；行为不变（P6a 的单测只改导入路径）。
- **筛选不推进基线**（同 P6a R10）：工作区页的富筛选回调只调 `GlobalIssuesFilter.updateFilterExpression`（写这一页的偏好）再经跟进重取，不用 `usePageFilterExpression`。
- **筛选行的选项**（工作区 HOC，§2.9）：成员 `getWorkspaceMemberIds(slug)`；项目 = 工作区里调用者是成员、未归档的项目（今天 `workspace-level.tsx:48-54` 的判断）；标签 = `labelMap` 里这些项目的（目录读到之后是全部）；状态 = 这些项目各自的 `getProjectStateIds`（状态 store 对没读过自己列表的项目回落到外壳读的工作区目录，`state.store.ts:114-121`；没有工作区级的状态 getter，不新加）。不同项目的同名状态是各自的选项（筛选按 id）、不带项目标识（R31：不改 M4 的筛选配置，带标识交 M8；§7）。
- **词表**：`ISSUE_DISPLAY_FILTERS_BY_PAGE.my_issues`（`PKG/constants/src/issue/filter.ts:187-218`，唯一的读者 `WC/form.tsx:137`）改名 `workspace_issues`：筛选是上面的 11 个词，布局只有 `spreadsheet`（`list` 一项删去，R8）。它的类型是索引签名（`filter.ts:38-40`），改名之后 `.my_issues.layoutOptions` 照样编译、运行时是 `undefined.layoutOptions`：所以读者按 `git grep` 跟随，不是编译跟随——`WC/form.tsx:137` 在同一提交改成 `workspace_issues`（否则 T8 到 T11 之间旧列表页的 “Edit View” 一点就抛），到它被删（T11）。各默认视图的筛选词表是它去掉自己的键（§2.5）。
- **Display 的属性**（m1）：`workspace_issues.layoutOptions.spreadsheet.display_properties` 是 14 个键（`PKG/constants/src/issue/common.ts:90-105`）去掉 `cycle`、`modules`。理由：Display 面板按 `useRouteListWords` 给词，路由没有项目时每个词都给（`C/hooks/use-route-list-words.ts:17-22`，`IL/filters/header/display-filters/display-properties.tsx:36-42`），而表格在没有项目的路由上不出这两列（`IL/spreadsheet/spreadsheet-view.tsx:61-66`）；今天 `my_issues` 给的是全部 14 个（`filter.ts:200`），照搬就有两个什么都不做的开关。存着的值与 `WORKSPACE_VIEW_CONTENT` 照旧是 14 个键（原样存回，`hasAdditionalChanges` 照旧比全部）。

### 2.5 默认视图（R4）

- **一张表**：`PKG/constants/src/workspace.ts:35-55` 的 `DEFAULT_GLOBAL_VIEWS_LIST` 每行加 `own?: "assignee_id" | "created_by_id" | "subscriber_id"`，键的类型改为同文件的 `DefaultWorkspaceViewKey`（`"all-issues" | "assigned" | "created" | "subscribed"`，代替 `@nerve/types` 的 `TStaticViewTypes`）：

| 键 | 名字（已有的键） | 自己的词 | 筛选词表 |
|---|---|---|---|
| `all-issues` | `default_global_view.all_issues` | 无 | 11 个 |
| `assigned` | `default_global_view.assigned` | `assignee_id: ["me"]` | 去掉 `assignee_id` |
| `created` | `default_global_view.created` | `created_by_id: ["me"]` | 去掉 `created_by_id` |
| `subscribed` | `default_global_view.subscribed` | `subscriber_id: ["me"]` | 去掉 `subscriber_id` |

- **查询**：页面的筛选加自己的词，自己的词放最后（`GlobalIssues.fetchIssues`）；词表里没有自己的键，用户就不会选出同名的条件（同一个键重复发是 422，`ISS:398-400`；Plane 的覆盖写法 `PW/core/store/issue/workspace/filter.store.ts:139-145` 不照搬）。“我关注的”按关注行算（`D:121`）。
- **偏好**：显示与属性存浏览器（按账户、工作区、视图，`D:121`），筛选只在这次进入（§2.4）。
- **删除**：`C/store/issue/helpers/issue-filter-helper.store.ts` 的 `getFilterConditionBasedOnViews`（`:134-137`、`:164-183`，没有调用者）；`TStaticViewTypes`、`STATIC_VIEW_TYPES`（`PKG/types/src/workspace-views.ts:41-43`）：另两处读者 T8 已去（`issue-filter-helper.store.ts:18`、`:136` 随帮手删，`PKG/constants/src/workspace.ts:8`、`:36` 换成 `DefaultWorkspaceViewKey`），最后的读者 `WC/default-view-quick-action.tsx:14`、`:22` 在 T9 删（§2.13），这三行同一提交删，文件的其余部分到 T11。（`STATIC_VIEW_TYPES` 没有 `as const`，`TStaticViewTypes` 今天就是 `string`，换键类型不会让谁编译不过：读者按 `git grep` 核。）

### 2.6 视图的数据层按容器（R9，L!）

- **容器**：新 `C/store/views/view-container.ts`：`ViewContainer = { kind: "project"; projectId: string } | { kind: "workspace"; workspaceId: string; workspaceSlug: string }`；`containerKey(c)` 给项目 id 或工作区 id（视图的 `project_id ?? workspace_id`）。
- **service**（`C/services/views/views.service.ts`，54）：加 `listWorkspaceViews(slug)`、`createWorkspaceView(slug, body)`（`NAV:77-133`），回答原样。
- **store**（`C/store/views/view.store.ts`，110）：
  - `lists` 的键是容器（项目视图按项目 id，工作区视图按工作区 id）。
  - 新 `getWorkspaceViews(workspaceSlug)`：工作区 store 按 slug 给出工作区、调用者的工作区角色过 `workspaceViewsOn`（§2.7）时给 `lists.get(workspace.id)` 的各行（各取 `getViewById` 的最新一份），否则 `undefined`；新 `fetchWorkspaceViews(workspace: WorkspaceRef)`。
  - `getViewById`（`:72-81`）：最新的一份按种类判可见——项目视图照旧（`projectOf(project_id)`）；工作区视图（`project_id` 为空）要工作区 store 给得出 `workspace_id` 的工作区（`workspaceRoot.workspaces` 里按 id 找，私有的 `workspaceOf`）且角色过 `workspaceViewsOn`。
  - `rowsOf`（`:104-109`）：每行的 `answeredAt` 取 `row.project_id ?? row.workspace_id` 那个键的。
- **写**（`C/store/views/view-changes.ts`，134，不过 220）：`create(container, body)`，队列键 `new:${containerKey}`，按种类调 service，回答 `prepended` 进 `lists[containerKey]`；`update`、`remove` 写进 `view.project_id ?? view.workspace_id`（今天 `:78`、`:95` 的 `?? undefined` 让工作区视图的改删不进任何列表）。`IViewStore.create` 的签名随之变，P6a 的调用处（`VC/modal.tsx:101`、测试与替身）编译跟随。
- **读的位置**（R20 的规则照旧）：键 `["WORKSPACE_VIEWS", workspace.id, slug]`，门是 `workspaceViewsOn`；读者是列表页（§2.10）与视图页页头的切换器（打开时，§2.8）。外壳不读。
- **删**：`RootStore.globalView`（`C/store/root.store.ts:59`、`:97`）、`GlobalViewStore`、`useGlobalView`、旧 `WorkspaceService` 与它的测试，随最后的读者在 T11 删（§2.13）。`C/store/root.store.test.ts:106` 的 store 数 20 → 19，`:113` 与 `:14` 删；`C/lib/store-context.test.ts:11`、`:313-318` 的例子换成 `x.issue`（`IssueRootStore.rootStore`，`C/store/issue/root.store.ts:88`，经根找兄弟的同一性质）。

### 2.7 入口规则与控件矩阵（R12、R13）

- **规则**（`C/lib/views-entry.ts` 加两个函数）：`workspaceViewsOn(role)` = 角色是工作区管理员或成员（工作区视图没有开关，`NAV:103-133` 没有 `views_disabled`）；`workspaceViewsCreatableBy(role)` = 同上（单列，读者是新建、另存的门，将来与读分开时只改一处）。角色一律 `getWorkspaceRoleByWorkspaceSlug(路由的 slug)`（`C/store/user/permissions.store.ts:56-59`）。名字里不得有不分大小写的 “sentry”（`K:1224` 的 `unsplash-telemetry`；P6a 的教训：`viewsEntry` 一类的名字会撞）。
- **矩阵**（“—”＝不给；每格对到服务端的一条规则；“主”＝所有者且角色是管理员或成员）：

| 控件 | 管理员 | 成员 | 访客 | 服务端的规则 |
|---|---|---|---|---|
| 侧栏 “Views”、命令面板的四条导航（R19 不改） | 给 | 给 | 给 | `workspace_issue.list`（`rules.go:187`） |
| 四个默认视图的页面：列表、筛选、Display（显示存浏览器）、表格列头的排序 | 给 | 给 | 给 | `workspace_issue.list` |
| 列表页的默认四行 | 给 | 给 | 给 | 同上 |
| 列表页与切换器里的已存视图（读 `GET …/views`） | 给 | 给 | —，不读 | `workspace_view.read`（`rules.go:235`） |
| 已存视图的页面 | 给 | 给 | “不存在”，不读 | `getView` 对工作区视图只给管理员、成员（`NAV:141-151`） |
| “Add view”（列表页与视图页页头）、“Save as”（五种工作区页） | 给 | 给 | — | `workspace_view.create`（`rules.go:236`） |
| “Update view”、行与页头的“编辑” | 主 | 主 | — | `View.UpdatableBy`（`server/internal/modules/navigation/domain/view.go:58`）且 `workspace_view.update`（`rules.go:237`） |
| 行与页头的“删除” | 全部 | 主 | — | `View.DeletableBy`（`view.go:65`）且 `workspace_view.delete`（`rules.go:238`） |
| 已存行与已存视图页头的“复制链接”“新标签页打开” | 给 | 给 | — | `workspace_view.read` |
| 默认行与默认视图页头的“复制链接”“新标签页打开” | 给 | 给 | 给 | 页面的地址，不读视图 |
| 工作区页空态的“新建项目”“新建工作项” | 给 | 给 | — | `project.create`（`rules_test.go` 的 `project.create` 行）、M4 的工作项规则 |
| 行的属性可编辑 | 按行的项目角色 | 同左 | 同左 | M4 的工作项规则（访客的项目角色只读） |
| 视图页的筛选、Display（已存视图只改草稿；“更新视图”在途时停用，§2.9） | 给 | 给 | 给（只在默认视图） | 只读 |

  被降为访客的所有者不给“编辑”“更新视图”；所有者离开工作区之后视图照常列出，没有人能改，工作区管理员能删。今天的偏差改正：“Add view” 不看角色（`WV/header.tsx:100-102`）、行的编辑删除给所有人（`WC/view-list-item.tsx:52-73`）、访客照样读已存视图（`WV/layout.tsx:22-26`、`WC/views-list.tsx:26-31`）。

### 2.8 视图页（R6、R7、R8、R11、R24）

- **路由与布局**：照 P6a 把 `WV` 拆成 `(list)`、`(detail)` 两个布局（`W/app/routes/core.ts:75-82` 改成两个 `layout(…)`，先例 `:148-161`）：`WV/(list)/{layout,page,header}.tsx`、`WV/(detail)/layout.tsx`、`WV/(detail)/[globalViewId]/{page,header}.tsx`。旧的 `WV/layout.tsx`、`WV/header.tsx` 与 `WV/layout.test.tsx` 删去（新测试接上它的各格）；`WV/[globalViewId]/page.tsx` 移到 `(detail)` 之下重写；`WV/page.tsx` 在 T9 原样移到 `WV/(list)/page.tsx`（列表的正文仍是旧组件，页头与布局是新的；T9 的 `(list)/layout.tsx` 只有 key 与宿主，页面状态在 T11 随 `WC/page-state.tsx` 加），T11 重写它的正文（§2.10）。`WV/(detail)/layout.tsx` 以 `globalViewId` 为 key（`<Fragment key={globalViewId}>`，`HO/M3-closeout.md:41`），`ViewActionsHost`（容器为工作区，§2.11）在 key 之内；后退、前进换视图时整棵重挂。
- **默认视图**（四个键）：不读任何视图；挂载时 `globalIssuesFilter.fetchFilters(slug, pageId)`（读浏览器的显示），卸下时 `discard(pageId)`；偏好在了才画布局根。
  - **机制**（预检 I2，主控取“不经 SWR”）：新 `WC/use-default-view-read.ts` 的 `useDefaultViewRead(workspaceSlug, key)`：一个 `useEffect`（依赖 store、slug、页面 id）在挂载时调 `fetchFilters(slug, pageId)`、清理时 `discard(pageId)`，返回 `getIssueFilters(pageId) !== undefined`。偏好在浏览器存储与内存里，`DefaultViewPreferences.get` 同步读完、立即兑现，所以每次挂载都读一次自己的，不进 SWR 的去重窗。不能照个人页的先例经 `useSessionSWR` 读（`C/components/profile/profile-issues.tsx:49-51`）：个人页卸下不丢偏好，这里卸下要 `discard`；同一个键在去重窗（2000 毫秒）里再挂上，取数函数不被调用，`filters[pageId]` 一直是空的，布局根不画，没有加载也没有错误（re-96 I1 的同一类，P6a 为已存视图写的说明见 `VC/use-view-page-read.ts:30-36`）。服务不抛错，`fetchFilters` 只会因会话换了而拒绝（`assertActive`）：effect 接住 `SessionChangedError`、不提示不记录（会话换了页面随之重挂）。卸下之前发出的读若在 `discard` 之后才落定，写回的是空筛选与发出时浏览器里的显示（`get` 在发出时已读完），与再进来的那一次读相同，再进来的读照常覆盖。
- **已存视图**：
  - 门：`workspaceViewsOn(role)` 不过（访客）→ 直接“不存在”，不发请求（R11）。
  - 读：P6a 的 `VC/use-view-page-read.ts` 泛化为 `useViewPageRead(storeType, workspaceSlug, viewId)`（`storeType` 是 `PROJECT_VIEW` 或 `GLOBAL`；键 `[storeType === PROJECT_VIEW ? "PROJECT_VIEW_PAGE" : "WORKSPACE_VIEW_PAGE", slug, viewId]`，项目的键不变），每次挂载读自己的（`mutate()` 一次，关掉挂载与重连的再验证），取数函数接住 `NoViewDraftError`、照常兑现（`noDraftIsNoFailure`，P6a 修复轮 m1，`VC/use-view-page-read.ts:42`、`:74-76`）；卸下时 `discard`。对 `GLOBAL`，`forbidden` 也算“不存在”（中途被降为访客；项目的照旧只认 `navigation.view_not_found`）。
  - 显示的视图：同文件加 `useShownWorkspaceView(workspaceSlug, viewId)`：读答到、草稿在、`project_id === null` 且 `workspace_id` 是路由的工作区才给；项目视图的 id（`GET /views/{id}` 对它也答）同“不存在”（R11）。
  - 四种状态：读在途 → 加载；不存在 → “视图不存在”（R29 的新键 `workspace_views.not_found.{title,description,action}`，动作回到工作区视图的列表页 `/{slug}/workspace-views`、文字 “Go to views”，同 P6a 项目视图的“不存在”回列表页，R30；Plane 回所有工作项，§7）；别的失败 → 失败与重试（`PlanningReadFailure` 的形状）；读到 → 布局根。读的组件以会话为 key（同 P6a 的 `ViewVisit`）。
  - 删除之后：只有这一页自己的宿主发出删除（删的是这一页的视图）时，跳转之前什么都不画（`ViewsPage.pageLeaving`，P6a 限定复审 n1）；别的挂载与再进来读答 404 的一律说“不存在”。“这一页显示的视图”由删除弹窗按容器从路由取（工作区的参数是 `globalViewId`，§2.11）：照 P6a 读 `viewId` 的话，在工作区视图自己的页头删除就不设 `pageLeaving`、不跳转，页面停在“视图不存在”（预检 I5）。
- **布局根**（新 `IL/roots/workspace-view-layout-root.tsx`，先例 `IL/roots/project-view-layout-root.tsx`，110）：`IssuesStoreContext` = `GLOBAL`；工作区 HOC（`entityType` = `GLOBAL`、`entityId` = 页面 id、`filtersToShowByLayout` = 这一页的词表、`updateFilters` = §2.4 的回调、`enableSaveView`、已存视图另给 `enableUpdateView`，§2.9）；`WorkItemFiltersRow`（`disabled` = `isUpdating(viewId)`，已存视图）；`DraftBusyContext` 以 `isUpdating(viewId)` 提供（默认视图恒假）；只渲染表格根；`IssuePeekOverview` 的 `hostScope` = `["workspace-view", slug, globalViewId]`；读工作区标签目录（§2.2）。
- **表格根**（新 `IL/spreadsheet/roots/workspace-root.tsx`，先例 `IL/spreadsheet/roots/project-root.tsx`）：`BaseSpreadsheetRoot` 收 `QuickActions = ProjectIssueQuickActions` 与 `canEditPropertiesBasedOnProject = (projectId) => 行的项目角色是管理员或成员`。这个接缝表格根已有（`base-spreadsheet-root.tsx:37`、`:80-90`，调研 Q7 说没有，是错的），R7 因此只是传它；路由没有项目，`isEditingAllowed` 为假，`disableIssueCreation` 随之为真，不给快速新建（`viewFlags` 也关）。迭代、模块两列在工作区路由上本来不出（`IL/spreadsheet/spreadsheet-view.tsx:61-66`，§15 D4），Display 也不给这两个属性（§2.4）。
- **布局**（R8，负责人）：工作区页只渲染表格。存着 `list` 布局的视图（经接口不给显示建的，合同默认 `list`，`NAV:637-644`）照表格画；草稿保留它存着的 `layout`（`PagePreferencesFilter.show` 不改写它，`page-preferences-filter.store.ts:167`），页头的 Display 一律给表格的选项（§2.3 第 5 点），没有布局切换、没有任何写 `layout` 的入口，“更新视图”原样存回。网页新建与另存明写 `spreadsheet`（§2.9、§2.11）。
- **页头**（`WV/(detail)/[globalViewId]/header.tsx`，Plane 的形状，R6）：面包屑（“Views” 链到 `/{slug}/workspace-views`；切换器 `BreadcrumbNavigationSearchDropdown` 列默认四个，`workspaceViewsOn` 时另列已存视图，打开时以 `["WORKSPACE_VIEWS", …]` 读，用 P6a 加的 `onOpen`）；筛选开关（`WorkItemFiltersToggle`，`GLOBAL` 与页面 id）；Display（`FiltersDropdown` + `DisplayFiltersSelection`，经 `usePageDisplay(GLOBAL, pageId)`，以 `t("common.display")` 为名，没有布局切换）；“Add view”（`workspaceViewsCreatableBy`，打开宿主的新建）；快捷操作（已存视图同行菜单，`VC/quick-actions.tsx`；默认视图只有复制链接与新标签页打开，新 `WC/default-view-menu.tsx` 在 T9 建，这个页头是它第一个读者，T11 的默认行是第二个，§2.10）。已存视图“更新视图”在途时筛选开关里的添加筛选与 Display 停用（`disabled = isUpdating(viewId)`，同 P6a R29）。四种状态下只画面包屑；窄屏照 Plane 不另设页头，800 宽时页头的控件仍可达（H20 (page) 核一次）。英文字面量都经 `t()`。
- **页面标题**：列表页 `${工作区名} - ${t("views")}`；视图页 `${工作区名} - ${视图名或默认视图的名字}`（今天写死的 “- All Views”，`WV/page.tsx:28`、`WV/[globalViewId]/page.tsx:21`）。
- **空态**（R24，新 `IL/empty-states/global-view.tsx`，`HO/M3-closeout.md:84` 点名的位置；先例 `PW/core/components/issues/issue-layouts/empty-states/global-view.tsx` 与 `IL/empty-states/project-issues.tsx` 的 `ProjectListEmptyState`）：有条件而筛到空 → 搜索的空态与“清除筛选”（`DraftBusyContext` 为真时停用）；没有条件、工作区里调用者没有在职的未归档项目 → `workspace_projects.empty_state.no_projects.*` 与“新建项目”；否则 → `workspace_empty_state.views.*` 与“新建工作项”（以 `GLOBAL` 打开新建弹窗）。两个按钮只给 `workspaceViewsCreatableBy`（照矩阵不给访客；Plane 给访客一个停用的按钮）。都是已有的键。

### 2.9 另存与更新视图（R10，L!）

- **拆分**：`C/components/work-item-filters/filters-hoc/use-project-view-saving.ts`（170）拆成：新 `use-view-update.ts`（容器无关的“更新视图”，下面）与按容器的另存——`use-project-view-saving.ts` 只剩项目页的另存并组合 `useViewUpdate`（名字与导出不变，P6a 的 HOC 不改），新 `use-workspace-view-saving.ts` 是工作区的另存加 `useViewUpdate`。每份不过 150 行。
- **“更新视图”（共用）**：`useViewUpdate({ storeType, workspaceSlug, viewId, canUpdate })`，`storeType` 是 `PROJECT_VIEW` 或 `GLOBAL`（取那一页的筛选 store：`confirmed`、`adopt`、`fetchFilters`，与富筛选实例）。
  - **请求体整个在轮到时从草稿算**（O1 的结构守护，选定这一种）：`body = () => { const draft = drafts.confirmed(viewId); return { filters: draft.filters, display_filters: draft.display_filters, display_properties: draft.display_properties } }`；点击交进来的表达式（`PKG/shared-state/src/store/rich-filters/filter.ts:501`）不用。不变量：草稿是这一页列出什么的唯一来源（列表的查询、富筛选的显示都由它来，`page-preferences-filter.store.ts:160-172`），发出去的就是轮到那一刻页面列出的；成功之后草稿、页面的筛选与基线都是 nerve 的回答。从轮到到回答之间草稿不变，仍由在途停用（P6a R29）守着；回答 `adopt` 进草稿，与当前表达式不等价时 `resetExpression`（P6a 的做法），所以在途中漏进来的改动不会被记成已存。
  - **两句边界**（预检 m3）：① `confirmed(viewId)` 的类型可以是 `undefined`：为空时体抛错、不发请求（同 `remove` 在轮到时找不到视图的写法，`C/store/views/view-changes.ts:91-92`），HOC 的 `failed` 一支照失败处理（提示、按钮留着）。它在这条路上不会为空：基类的 `values` 只增不删（`page-preferences-filter.store.ts:128`、`:195`、`:229`，没有删除），而“更新视图”只在布局根画出之后可点，布局根要 `getIssueFilters(viewId)`，它与 `values` 在同一次 `fetchFilters` 里写（`:195-196`）；一格钉住（下一句的那一格同时证它不为空）。② 页面在轮到之前离开：子类的 `discard` 不清基类的 `values`（`:155-157`，`C/store/issue/project-view/filter.store.ts:27-28`），所以离开之后才轮到的体是离开前的草稿，发出去的仍是用户点击时看到的；离开又进来（重新播种）之后才轮到，体是视图读回的内容，等于一次空改。单测一格：扣住前一笔、点击、卸下页面、放开，体等于卸下之前的草稿。
  - 其余同 P6a R9：等 nerve 的回答；成功 → 提示（按容器的键：`project_views.action.update.success` / `workspace_views.action.update.success`）、`adopt`、`fetchFilters`、必要时 `resetExpression`，然后 resolve；失败 → 按码提示、reject，草稿与基线不动、按钮留着；跟进只在 HOC 还挂着时、经 `followInSession`；成功之后焦点落在筛选行的添加筛选。`hasAdditionalChanges` 比 `confirmed(viewId)` 的显示与属性和视图存着的（线上词）。
  - `canUpdate`：视图在、所有者是当前用户、项目视图过 `viewsCreatableBy(project, role)` / 工作区视图过 `workspaceViewsCreatableBy(role)`。不看 `is_locked`。
- **另存（工作区）**：五种工作区页（四个默认视图与已存视图页）都给，给谁照 `workspaceViewsCreatableBy`，按钮在富筛选有条件时出现（shared-state 的 `canSaveView`，`filter.ts:271-273`；所以“分配给我”先加一个条件才有“Save as”）。弹窗（容器为工作区，§2.11）预填 `{ filters: { ...toWorkspaceFilters(expression), ...自己的词 }, display_filters: { ...confirmed.display_filters, layout: "spreadsheet" }, display_properties: confirmed.display_properties }`，名字为空（P6a R30）；成功之后转到新视图的页面。项目页的另存不变（P6a R7、R8）。
- **工作区 HOC**（`filters-hoc/workspace-level.tsx`，208，改后不过 150）：去掉 Plane 的分支（`useGlobalView`、`is_locked`、`owned_by`、`access`、“{name} 2”/“Untitled” 的预填、`rich_filters` 的体、英文提示，`:39-160`）；只剩筛选行的宿主：选项（§2.4）、`useWorkspaceViewSaving`、另存的弹窗。个人页照旧不给保存（行为不变）。`shared.ts`（33）里只给 HOC 覆盖标签与 `isDisabled` 的类型若随之没有读者，同一提交删去（核读者写进台账）。P4 R17 的工作区一半关闭（`specs/P4-web-notifications.md:379-381`）。
- **在途停用**：工作区已存视图页照 P6a R29：来源只有 `ViewStore.isUpdating(viewId)`；筛选行、页头的筛选开关与 Display、表格列头的排序、空态的清除筛选都停用（后两处经 `DraftBusyContext`）。默认视图没有“更新视图”，不停用。

### 2.10 列表页（R13）

- **布局**：`WV/(list)/layout.tsx` 以 `workspaceSlug` 为 key，提供页面状态（只有搜索词，新 `WC/page-state.tsx`，先例 `VC/page-state.tsx`），宿主 `ViewActionsHost`（容器为工作区）在 key 之内；离开页面、换工作区即丢。布局在 T9 建（key 与宿主），页面状态在 T11 加。
- **页面**（`WV/(list)/page.tsx`）：搜索框（`InputGroup`，以 `t("common.search.label")` 为名）；默认四行（按名字的译文不分大小写的子串过滤）；`workspaceViewsOn` 时另列已存行（读在途 → 加载；失败 → 失败与重试；没有已存视图时只有默认四行；搜索筛掉全部时给“没有匹配”，`common.search.no_matches_found`）。访客只见默认四行、不读。没有排序与筛选（Plane 相同）。
- **页头**（`WV/(list)/header.tsx`）：面包屑（`t("views")`，不是链接）与 “Add view”（`workspaceViewsCreatableBy`，打开宿主的新建）。
- **行**：已存行用 P6a 的 `VC/view-list-item.tsx`、`view-list-item-action.tsx` 按容器泛化（§2.11），链到 `/{slug}/workspace-views/{id}`；默认行是新 `WC/default-view-row.tsx`（T11），链到 `/{slug}/workspace-views/{key}`，菜单只有复制链接与新标签页打开：`WC/default-view-menu.tsx`（T9 为视图页页头建，§2.8；以同一工厂的两项组成，`C/components/common/quick-actions-helper.tsx`；复制经 `useCopyLink`，成功与失败各自提示；菜单按钮以 `t("common.item_options", { name })` 为名、一行一个右键菜单）。

### 2.11 弹窗、宿主、行菜单与删除按容器（R14、R15，L!）

- **弹窗**（`VC/modal.tsx`，133）：`projectId` 换成 `container: ViewContainer`；新建经 `views.create(container, …)`，成功之后转到对应的视图页（项目：今天的地址；工作区：`/{slug}/workspace-views/{id}`）；工作区容器新建的默认内容是 `WORKSPACE_VIEW_CONTENT`（`VC/view-content.ts` 加，`NEW_VIEW_CONTENT` 的表格版：`layout: "spreadsheet"`、不分组、`-created_at`、显示子工作项与空组、日历按月不含周末，14 个属性全开）；提示按容器取键（键写全的一张表，`tools/i18n-keys.mjs` 才认得：项目 `project_views.action.{create,update}.success`，工作区 `workspace_views.action.{create,update}.success`）。关闭政策、`ModalTitle`、`FieldError`、被拒取得焦点、自然 Tab、只管名字说明图标都不变（P6a R11、R12、R28、R30）。`C/components/modals/project-level.tsx` 的 `create_view` 只换 props 的形状。
- **宿主**（`VC/actions-host.tsx`，102）：`projectId` 换成 `container`；`pageMounted`、`pageLeaving` 照旧。
- **行菜单**（`VC/quick-actions.tsx`，106）：角色按容器（项目角色 / 工作区角色），`isAdmin` 是那一级的管理员，`isOwner` 是所有者且那一级的角色是管理员或成员；链接按容器；复制经 `useCopyLink`（`C/hooks/use-copy-link.ts:42`）。容器的角色与写权限收成一个 hook（新 `VC/use-view-rights.ts`，读者是行菜单与两份另存）；地址收成新 `VC/view-paths.ts`（`viewPath`、`viewsPath`，与路由的形状同一处的 `pageViewParam`，下一条），读者是弹窗、行、菜单、删除。
- **删除**（`VC/delete-view-modal.tsx`，92）：确认之后，删的是这一页显示的视图时转到 `viewsPath(container)`（工作区：列表页 `/{slug}/workspace-views`）；文案按容器（工作区用已有的 `workspace_views.delete_view.{title,content}`、`workspace_views.action.delete.success`）；`pageLeaving` 的规则照旧。“这一页显示的视图”的 id 按容器从路由取（预检 I5）：今天弹窗读 `useParams().viewId`（`:43`），以它判 `shownByPage`、设 `pageLeaving`（`:58-59`）、决定跳不跳（`:69`），而工作区的路由参数叫 `globalViewId`（`W/app/routes/core.ts:78-81`）。`VC/view-paths.ts` 给 `pageViewParam(container)`（项目 `"viewId"`、工作区 `"globalViewId"`），弹窗读 `useParams()[pageViewParam(container)]`，不直接读 `viewId`（`VC/` 的产品文件里读路由参数的只有这一处）。默认视图的键永远不等于视图 id，所以默认视图页上不会误判。
- **行**（`VC/view-list-item.tsx`、`view-list-item-action.tsx`）：`projectId` 换成 `container`。
- **删去**（在最后的读者走掉的 Task 里）：`WC/quick-action.tsx`、`WC/default-view-quick-action.tsx` 只被 T9 删掉的 `WV/header.tsx`（`:19`、`:21`）导入，另有 `WV/layout.test.tsx:29-30` 的两个替身（同在 T9 删）：在 T9 删，`K` 的两个 `copy-hook` 例外（`K:3424-3437`，路径在 `:3426`、`:3433`）同一提交删（文件没了例外即成 stale，`tools/keywords.mjs:182-211` 报错）；`WC/{modal,form,delete-view-modal,view-list-item,default-view-list-item,views-list}.tsx` 的最后读者在 T11 走掉（`modal.tsx` 还被工作区 HOC 读到 T10、被 `view-list-item.tsx` 读到 T11），在 T11 删。

### 2.12 个人页的按标签分组与标签筛选（R18，L!）

- 个人页挂载时读目录（§2.2，门是 `allowed`）；分组的列（`IL/group-columns.tsx:266-284`）与筛选的选项（工作区 HOC 的 `labelIds`）都经 `labelMap` 拿到目录里的标签，组名齐全、空组齐全，“—”只剩真的不认识的 id（服务端答了、目录里没有：读之后被删的）。`group-columns.tsx` 不改（它读 `labelMap` 的写法本来就对），`:277` 的注释照实改；HOC 的注释（`workspace-level.tsx:55` 的 “a cold profile does not fetch a catalog”）随 §2.9 的改写删去。
- I19（`E/stories/issue/i19-member-issues.spec.ts`，528，M4 的故事）两处改写（允许清单，不让文件变长）：`:336` 改为“这一路恰一个目录的读、没有项目标签的读”；`:460` 的标题与 `:485-493` 改为“重载前后各一个目录的读（`[目录, 项目标签, 目录]`，项目标签那一次是详情的标签下拉打开时读的），筛选的标签选项在冷的重载之后仍在”。三处的正则（`:336`、`:485`、`:491`，今天只认 `/projects/{id}/labels`，目录的读匹配不上）收成文件里的一个常量。

### 2.13 删除与读者（R20、R21）

每一项在删之前 `git grep` 核读者，阶段记录写下结果；删掉最后一个读者的 Task 同一提交删去它。

| 删除 | 最后的读者（基线） | 何时 |
|---|---|---|
| `getFilterConditionBasedOnViews`（`C/store/issue/helpers/issue-filter-helper.store.ts:134-137`、`:164-183`） | 没有调用者 | T8 |
| `WV/layout.tsx`、`WV/header.tsx`、`WV/layout.test.tsx`（`WV/page.tsx`、`WV/[globalViewId]/page.tsx` 移进新布局） | 路由 | T9（两个新布局；列表页的正文在 T11 换） |
| `E/stories/smoke/s2-issue-list-removal.spec.ts`（整份，R22） | — | T9（它唯一的一行钉着今天的视图页，T9 重写之后必红） |
| `ISSUE_DISPLAY_FILTERS_BY_PAGE.my_issues`（改名，§2.4） | `WC/form.tsx:137` | T8 改名、读者按 `git grep` 跟随（索引签名，编译查不出，§2.4）；T11 随旧表单 |
| 工作区 HOC 的旧分支（`workspace-level.tsx:39-160`） | 个人页 | T10 |
| `WC/quick-action.tsx`（107）、`WC/default-view-quick-action.tsx`（103）与 `K` 的两个 `copy-hook` 例外（`K:3424-3437`） | `WV/header.tsx:19`、`:21`（另有 `WV/layout.test.tsx:29-30` 的替身），都在 T9 删 | T9（预检 I4） |
| `WC/{modal,form,delete-view-modal,view-list-item,default-view-list-item,views-list}.tsx` 六个旧组件 | `WV/page.tsx:17-18`（列表页，T11 换）、`filters-hoc/workspace-level.tsx:17`（T10 换） | T11 |
| `C/store/global-view.store.ts`（179）、`C/hooks/store/use-global-view.ts`（17）、`RootStore.globalView`、`C/services/workspace.service.ts`（43）与 `workspace.service.test.ts`（53） | `WC/` 的旧组件、`WV` 的旧页头与布局、工作区 HOC | T11 |
| `STATIC_VIEW_TYPES`/`TStaticViewTypes`（`PKG/types/src/workspace-views.ts:41-43`） | `WC/default-view-quick-action.tsx:14`、`:22`（另两处 T8 已去，§2.5） | T9 |
| `PKG/types/src/workspace-views.ts` 的其余（`IWorkspaceView` `:15-39`，T11 删整份）与桶导出；`IWorkspaceViewProps`（`PKG/types/src/view-props.ts:170-174`）；`EViewAccess`（`PKG/types/src/views.ts:7-10`；`TViewFiltersSortKey`、`TViewFiltersSortBy` 被 P6a 的列表页读，留） | 旧 store、旧组件、旧 HOC（`WV/layout.test.tsx` 在 T9 已去） | T11 |
| `common/applied-filters/{members,date}.tsx` | `VC/applied-filters/root.tsx:11-12` 仍读 | 不删（条件不成立）；与 `project/applied-filters/` 的重复交 M8 |

- **`old-view-urls`**（`K:3257`）：内容规则扩到工作区的旧地址（`/api/workspaces/…/views…`，项目一段可选），说明改写；两条 miss 样例（`/api/workspaces/${workspaceSlug}/views/`、`…/views/${viewId}/`）翻成 hit；miss 另加 `/api/v0/workspaces/{slug}/views`、页面地址 `` `/${workspaceSlug}/workspace-views/${viewId}` ``；文件 hit 加 `web/apps/web/core/services/workspace.service.ts`（只是路径）。与旧 service 同一提交（T11）。`plane-workspace-urls`（`K:2053`）的 miss 样例不动。
- **`workspace-address-route`**（`K:2844`）：P6b 新建或改写的、读工作区的文件收入文件名单并带 hit 样例（`WV/(list)/{layout,header}.tsx`（T9）；`WV/(list)/page.tsx` 只在 T11 重写之后收——T9 原样移过去的它还读 `currentWorkspace`（`WV/page.tsx:25`、`:28`），收进来关键词就红（预检 m5）；`WV/(detail)/*`、`IL/roots/workspace-view-layout-root.tsx`、`IL/empty-states/global-view.tsx`、`WC/` 的新文件里读工作区的、`filters-hoc/workspace-level.tsx`、`use-workspace-view-saving.ts`、`VC/use-view-rights.ts`、`C/hooks/use-workspace-label-catalog.ts`）；内容规则照旧（不读 `currentWorkspace`、工作区级的 `allowPermissions` 要传 slug）。P6a 已在 `project-address-route` 的名单里的 `VC` 文件留在那里：那条规则把两参、三参的 `allowPermissions(…)` 都算 hit（`K:2342`，名单 `K:2639-2655`），所以 `VC` 按容器取角色一律走 `getWorkspaceRoleByWorkspaceSlug` / `getProjectRoleByWorkspaceSlugAndProjectId`（§2.7），不换成 `allowPermissions`，规则不放宽（预检 o3）。
- **死成员**：P6aR §7 留给 P6b 的 19 行（`workspace-views.ts` 12 行、`view-props.ts:171-173`、`WV/layout.test.tsx:24`、`:26`、`WC/default-view-quick-action.tsx:23`、`global-view.store.ts:19`）在 T11 之后清零，逐行写去处：其中 `WV/layout.test.tsx:24`、`:26` 与 `WC/default-view-quick-action.tsx:23`（`i18n_label`）三行随文件在 T9 走掉，其余 16 行在 T11。

### 2.14 守卫与告警（R25）

- **oxlint**：web 的上限 14（`W/package.json:10`）是等式；新文件零告警；删掉的文件若带告警，同一提交照实改上限并留计数的收据。e2e 的上限 0。
- **knip 与第一个读者**：新文件、新导出在第一个读者出现的 Task 里建立（plan 列出）；包的桶导出与只被测试读的产品导出 knip 查不出，死成员的整份比对兜底。
- **swr 名单**（`.oxlintrc.json:205-219` 那一组）：收入 `C/hooks/use-workspace-label-catalog.ts`、`WV/(list)/page.tsx`（或它读已存视图的组件）、`WV/(detail)/[globalViewId]/header.tsx`；`VC/use-view-page-read.ts` 已在。删掉的 `WV/layout.tsx` 不在名单里（它直接用 `swr`，今天就违反 v0 7.7，随删除了结）。
- **文案**：P6b 动到或新建的文件里的字面量都经 `t()`，en 的文字不变；新键只有 R29 的六个（`workspace_views.not_found.{title,description,action}`、`workspace_views.action.{create,update,delete}.success`；en 与 zh-CN 同一提交，写完整的字面量，不许一个键是另一个键的前缀）；别的一律用已有的键（`views`、`common.display`、`common.filters`、`common.search.label`、`common.search.no_matches_found`、`common.item_options`、`common.link_copied_to_clipboard`、`toast.success`、`default_global_view.*`、`workspace_views.add_view`、`workspace_views.delete_view.*`、`workspace_projects.empty_state.no_projects.*`、`workspace_empty_state.views.*`、`common_empty_state.search.*`、`project_issues.empty_state.issues_empty_filter.secondary_button.text`）。
- **iframe**：工作区视图的页面不嵌 iframe（`git grep -i iframe -- web/apps/web/core web/apps/web/app` 在终检时再核一次，阶段记录写一句），CSP 不改（`HO/M2-closeout.md:78`）。

### 2.15 e2e（R22、R23）

- **配置**：页面故事都在 `E/stories/collaboration/`，名字合 m7-pages 的 `testMatch`（`-page.spec.ts` 或 `-pages.spec.ts`）与 `grep`（标题带 `H13 (page)`、`H14 (page)` 或 `H20 (page)`），三引擎；CI 以默认配置在 chromium 跑一次。接口故事 `h13-workspace-labels.spec.ts`（§2.1）不进 m7-pages。
- **检查点与终检的页面配置**（R23）：同 P6a 的六个——m7-pages、m6-pages、m6-acceptance、m4-list（引擎与表格）、m4-filters（I19 与 `PagePreferencesFilter`）、m4-detail（行的命令与浮层）；`gates.sh` 本来就跑 Go 两路与 `gen-check`。第 1 路以 P6b 检查点 0 的本机值为基线，+30 秒停查；CI 的八成、web 960 秒、e2e 30 分钟只报告。本机若被 WindowServer 注销，同一 HEAD 只补跑没跑到的配置。
- **夹具**：新 `E/fixtures/workspace-view-pages.ts`（工作区视图页与列表页的定位、行、切换器、扣住视图的回答、工作区列表的读表、`WORKSPACE_VIEW_CONTENT` 的期望）；前置条件只经接口（`E/fixtures/views.ts` 的 `createWorkspaceView` `:34`、`viewDefaults` `:52`、`viewRow` `:158`、`savedQuery` `:184`）；`E/fixtures/mounts.ts` 另加 `WORKSPACE_VIEWS = ["GET /api/v0/workspaces/{slug}/views"]`（T12 加，注释同 `PROJECT_VIEWS`）。`view-pages.ts`、`cycle-pages.ts` 不长。
- **故事**：

| 文件 | 标题 | 覆盖 |
|---|---|---|
| `h13-workspace-list-page.spec.ts` | `H13 (page)` | 四个默认视图各进一次：跨两个项目的行；请求的查询精确（`assignee_id=me` 等自己的词、`order_by`、`sub_issue`、`limit`，没有分组参数）；“Load more” 翻页一次；只看自己的访客在只看自己的项目里只见自己的、在另一个项目里见全部；不是成员的项目、已归档的项目的行不出（总数里不含它们由接口故事 `h13-workspace-list.spec.ts:101` 按数据库逐条钉住，页面只核行，M7-03 的“无权项计数”）；改显示（表格列头排序、Display 的属性，Display 没有 Cycle、Modules）存浏览器，离开再进来还在、换工作区不跟过去；经切换器从“所有工作项”到“分配给我”再立刻回来，列表照常画出、显示是浏览器里的（预检 I2：默认视图的偏好每次挂载读自己的）；改筛选（含项目与标签）只在这次进入，离开再进来没了；标签的筛选选项来自目录（进页面恰一个 `GET /api/v0/workspaces/{slug}/labels`），行上的标签名齐全；行的属性按行的项目可编辑（访客的行只读），这一步放在“分配给我”上：把一行的负责人改成别人，确认之后行离开（预检 I1：工作区支的成员判断）；访客能读、没有 “Save as”；列表的读失败之后重试（只在 Chromium、目标页代码载入之后装路由，同视图读的那一步）；空态（有条件筛到空的“清除筛选”；没有项目的“新建项目”） |
| `h13-workspace-view-page.spec.ts` | `H13 (page)` | 已存工作区视图页：恰一个 `GET /api/v0/views/{id}`，列表的查询包含 `savedQuery` 的各项；改筛选不发 `PATCH`、“Update view” 出现；所有者“更新视图”：请求体精确（筛选与显示都是那一刻的草稿），扣住回答时筛选行、筛选开关、Display、表格列头都停用，放开之后恢复、经接口读回与回答相同；被拒（另一会话经接口删掉视图，404 写进期望）提示、按钮留着；成员看别人的视图没有“更新视图”与编辑；从默认视图“分配给我”加一个条件之后 “Save as”，请求体带 `assignee_id: ["me"]` 与 `layout: "spreadsheet"`、转到新视图；经接口建的 `list` 布局视图照表格画、改排序之后“更新视图”的请求体里 `layout` 仍是 `list`；不存在的三种（不存在的 id、项目视图的 id、访客打开已存视图的地址——没有 `GET /api/v0/views/{id}`）都是“视图不存在”，动作回到列表页（R30）；读失败重试（只在 Chromium、目标页代码载入之后装路由） |
| `h14-workspace-views-list-page.spec.ts` | `H14 (page)` | 列表页：默认四行与已存行、搜索、“没有匹配”；管理员、成员 “Add view”：弹窗以 “Create View” 为名，名字空白与超长（`FieldError`、`aria-invalid`、焦点、不发请求），请求体精确（`WORKSPACE_VIEW_CONTENT`、没有 `filters`），`viewRow` 的 `project_id` 为空、所有者是他，行出现、转到新视图；访客没有 “Add view”、没有已存行、没有 `GET …/views`；所有者改名只发 `name`；成员删不了别人的、工作区管理员删别人的（`viewRow` 的 `deleted`）；在视图自己的页头删除之后转到列表页、不闪“不存在”；复制链接成功与被拒（`refuseClipboardWrites`）各有提示；菜单按钮名为 “{name} options”、右键打开右键菜单 |
| `h20-workspace-view-pages.spec.ts` | `H20 (page)` | 迟到的回答：已存视图列表的读在换工作区之后答到不写进新工作区的列表页；视图的读在经切换器换视图之后答到不写进新视图，回到原视图又一个 `GET /api/v0/views/{id}`；“更新视图”与删除在离开页面之后答到，没有提示与跳转；另一个标签页换了账号之后答到的“更新视图”不在新会话里提示；后退前进之后上一个视图的弹窗与草稿不跟过去（`HO/M3-closeout.md:41`）；键盘：弹窗的 Tab 次序（图标、名字、说明、Cancel、提交）、嵌套 Escape（图标选择器）、切换器以键盘选、行菜单以键盘打开与关上之后焦点回到按钮、以键盘“更新视图”之后焦点落在添加筛选（三引擎记焦点）；“Display” 的按钮以它为名、`aria-expanded`、Escape 之后焦点回到它（`HO/M3-closeout.md:65`）；800 宽一次（页头控件可达） |

- **I19**（m4-filters）：§2.12 的两处改写，在 T14。
- **S2**：`s2-issue-list-removal.spec.ts` 整份删（T9）；`s2-web-app.spec.ts` 随目录的读改（T4、T5）。
- **故事的写法**：每个“在途”都靠扣住回答（`holdAnswer`），不靠时间，不断言墙上的时钟；一律放行或拒绝的路由只在 Chromium 装，而且只在目标页的代码载入之后装；换页、离开、改服务端的状态之前先等在途的读答完（动作之前登记 `waitForResponse`）；每个故事以 `expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []])`（预期的拒绝写进去）与 `expectQuietConsole`（不放宽）结束。
- **时长**：每份不过 400 行；本机 chromium 每个文件约 8 秒是先例的上限，超了写原因；任何一个 `test()` 单引擎超过 30 秒先报主控。

### 2.16 文件与职责

| 文件 | 职责 | Task |
|---|---|---|
| `PRJ`、`api/openapi.yaml`、生成物；`SV/domain/actions.go`、`rules.go`/`rules_test.go`；`SV/app/{list_workspace_labels,label_ports}.go` 与测试；`SV/adapter/postgres/{queries/labels.sql,labels.go,labels_test.go,failures_test.go}`；`SV/adapter/http/{workspace_labels,workspace_labels_test,handler,handler_test}.go`；`SV/module.go`；`BS/permission_matrix_labels_test.go`、`BS/workspace_labels_visibility_test.go`（新） | 接口 | T2 |
| `E/stories/collaboration/h13-workspace-labels.spec.ts`（新） | 接口故事 | T3 |
| `C/services/project/labels.service.ts`、`C/store/label.store.ts`、`C/hooks/use-workspace-label-catalog.ts`（新）、`C/components/issues/workspace-draft/root.tsx`、`E/fixtures/mounts.ts`、`s2-web-app.spec.ts` 的草稿一行 | 目录与草稿页的读 | T4 |
| `C/components/profile/profile-issues.tsx`、`s2-web-app.spec.ts` 的个人页三行、`IL/group-columns.tsx` 的注释 | 个人页读目录 | T5 |
| `C/store/views/{view-container,view.store,view-changes}.ts`、`C/services/views/views.service.ts`、`C/lib/views-entry.ts` | 数据层按容器、入口规则 | T6 |
| `VC/{modal,actions-host,quick-actions,delete-view-modal,view-list-item,view-list-item-action,view-content}.tsx/ts`、`VC/{use-view-rights,view-paths}.ts`（新）、`C/components/modals/project-level.tsx`、en/zh-CN | 弹窗、宿主、行菜单、删除按容器 | T7 |
| `PKG/types/src/issues/issue.ts`；`C/store/issue/workspace/*`（新）、`helpers/view-draft.service.ts`（移来）、`C/store/issue/root.store.ts`；`issue-list-read.ts`、`issue-list-binding.ts`、`issue-list.service.ts`；`C/lib/paged-groups.ts`（`ListContext` 一臂与 `:332` 一行）、`C/lib/issue-membership.ts`（工作区支）、需要时 `C/store/issue/fake-issues.ts`（只追加）；`C/lib/issue-filters.ts`、`page-preferences-filter.store.ts`、`profile/filter.store.ts`；§2.3 的分派点；`use-global-issue-actions.ts`、`use-global-preferences-follow.ts`（新）；`PKG/constants/src/{workspace,issue/filter,issue/common}.ts`；`issue-filter-helper.store.ts` | 引擎 | T8 |
| `W/app/routes/core.ts`、`WV/(list)/{layout,header}.tsx`（新）、`WV/(list)/page.tsx`（移来）、`WV/(detail)/*`（新）、`VC/use-view-page-read.ts`、`IL/roots/workspace-view-layout-root.tsx`、`IL/spreadsheet/roots/workspace-root.tsx`、`IL/empty-states/{index,global-view}.tsx`（新）、`WC/{default-view-menu.tsx,use-default-view-read.ts}`（新）；删旧 `WV` 文件、S2 移除冒烟、`WC/{quick-action,default-view-quick-action}.tsx`、`TStaticViewTypes` 的三行与两个 `copy-hook` 例外 | 视图页 | T9 |
| `filters-hoc/{use-view-update,use-workspace-view-saving}.ts`（新）、`use-project-view-saving.ts`、`workspace-level.tsx`、`shared.ts` | 另存与更新视图 | T10 |
| `WV/(list)/{page,layout}.tsx`、`WC/{page-state,default-view-row}.tsx`（新）；删 §2.13 的旧层（T11 的几行）；`K`、`.oxlintrc.json`；`root.store.test.ts`、`store-context.test.ts` | 列表页与删除 | T11 |
| `E/fixtures/workspace-view-pages.ts`（新）、`h13-workspace-list-page.spec.ts`、`h13-workspace-view-page.spec.ts`、`mounts.ts` 的 `WORKSPACE_VIEWS` | H13 (page) | T12 |
| `h14-workspace-views-list-page.spec.ts` | H14 (page) | T13 |
| `h20-workspace-view-pages.spec.ts`、`i19-member-issues.spec.ts` 两处 | H20 (page) 与 I19 | T14 |

## 3. 局部裁定

主控的裁定在台账 `rulings.md`（D-1–D-32），本节逐条写成 R1–R32：**R-n 即 D-n**；R29–R32 是主控对起草时 §8 四件待定的裁定（“起草之后的裁定”一节）。十一条是**高风险的局部裁定**（**L!**，动到 M3、M4、M6、P6a 的网页共用代码），阶段记录写明、收官报告列出：R1、R2、R5、R7、R9、R10、R14、R15、R17、R18、R20；R10 里 O1 的选法（P6a 项目视图的“更新视图”一并改为从草稿取体）经主控确认为 L!。**R16、R8 是负责人的**（2026-11-27）。

| D | R | D | R | D | R | D | R |
|---|---|---|---|---|---|---|---|
| D-1 | R1（L!） | D-8 | R8（负责人） | D-15 | R15（L!） | D-22 | R22 |
| D-2 | R2（L!） | D-9 | R9（L!） | D-16 | R16（负责人） | D-23 | R23 |
| D-3 | R3 | D-10 | R10（L!） | D-17 | R17（L!） | D-24 | R24 |
| D-4 | R4 | D-11 | R11 | D-18 | R18（L!） | D-25 | R25 |
| D-5 | R5（L!） | D-12 | R12 | D-19 | R19 | D-26 | R26 |
| D-6 | R6 | D-13 | R13 | D-20 | R20（L!） | D-27 | R27 |
| D-7 | R7（L!） | D-14 | R14（L!） | D-21 | R21 | D-28 | R28 |
| D-29 | R29 | D-30 | R30 | D-31 | R31 | D-32 | R32 |

- **R1 新 store 类型 `GLOBAL`、新范围臂 `"workspace"`**（D-1，**L!**）：§2.3。理由：工作区列表是三种角色都能读的平铺列表，范围是“他是成员的全部项目”，与个人页的成员列表（带 `relation`、可分组、只给管理员成员）不是一回事；借 `PROFILE` 要在偏好、分组、键上处处分支。代价：`IssueListScope` 与 `ListContext` 各多一臂，`readPage`、`covers`、`allowsRole`、身份各多一个分支，`issueMembership` 多一支（工作区支：同成员支的范围加项目支的状态、负责人、创建者，提及与关注记“未知”；预检 I1），`paged-groups.ts:332` 改一行；约 10 个分派点；钉退役的测试（两份文件三处）翻过来；`C/store/issue/root.store.ts` 的上限 340 → 350（只为构造）。动到 M4 的 `use-issues.ts`、`use-issues-actions.tsx`、`use-issue-layout-store.ts`、`issue-list-*`、表格根，M6 的 `use-page-display.ts`、`use-page-issue-filters.ts`。
- **R2 放宽 `PagePreferencesFilter` 的界，一个筛选 store、两种偏好服务**（D-2，**L!**）：§2.4。理由：线上词与控件词的换算只有一份（P6a R28），工作区页必须经它；`toFilterExpression` 收 `project_id` 让个人页的绕行可以删。代价：`issue-filters.ts` 约 +30 行；`page-preferences-filter.store.ts` 只改类型；个人页的 `cleanExpression` 变一行。
- **R3 默认视图存浏览器、已存视图用内存草稿**（D-3）：§2.4。理由：设计 `D:120-121` 与 Plane（`PW/core/store/issue/workspace/filter.store.ts:161-170`、`:173-181`）。代价：一个约 80 行的服务与单测；草稿服务泛化、移到 `helpers/`。
- **R4 默认视图自己的词固定在查询里**（D-4）：§2.5。理由：同迭代页的 `withoutOwn`（P5b R9），页面自己的条件不让用户改，就不会被悄悄换掉（Plane 的覆盖会）。代价：表多一列；删无读者的帮手。
- **R5 契约的 11 个筛选词**（D-5，**L!**）：§2.4。理由：经接口建的视图里有 `state_id`、`mention_id` 时，网页要能显示、原样存回（产品对人与 agent 不分）。代价：M4 的词表改名（索引签名，读者按 `git grep` 跟随）、多三个词；Display 的属性去掉 `cycle`、`modules`（预检 m1）；同名状态的选项（§7）。
- **R6 Plane 的页头，面包屑 “Views” 链到列表页**（D-6）：§2.8。理由：列表页没有别的入口。代价：页头重写约 130 行；plane-diff 一行。
- **R7 表格根按行的项目判可编辑，行的命令照个人页，不给快速新建**（D-7，**L!**）：§2.8。起草时核出：接缝已在 `base-spreadsheet-root.tsx:37`、`:80-90`，所以只是新的工作区根传它、联合加 `GLOBAL`。代价：一个约 30 行的根；M4 的表格根只改联合。
- **R8 只渲染表格，网页新建与另存明写 `spreadsheet`，`list` 视图照表格画、原样存回**（D-8，**负责人**：不改合同）：§2.8。代价：`usePageDisplay` 对 `GLOBAL` 一律取表格的选项；接口默认（`list`）与页面（表格）的这点不一致写进 plane-diff 与阶段记录。
- **R9 P6a 的数据层按容器扩到工作区视图，删 `globalView`**（D-9，**L!**）：§2.6。理由：一个实体一个 store，`/views/{id}` 的回答只写一处。代价：三个文件各 +20–30 行；P6a 的单测加工作区的格、`create` 的调用处编译跟随；`root.store.test.ts` 20 → 19、`store-context.test.ts` 换例子（M2/M3 的会话测试）。
- **R10 更新一半共用、另存按容器，请求体整个在轮到时从草稿算**（D-10，**L!**）：§2.9。O1 两种写法里选“轮到时算”：理由是草稿本来就是这一页列出什么的唯一来源，体从它取就不再需要“点击时的快照与轮到时的显示是两个来源”的不变量；“点击时冻结”还得另写一条“发出的与当前的相同”的核对。主控接受，确认为 L!（改的是 P6a 项目视图“更新视图”的行为来源）。代价：P6a 的“更新视图”跟着换来源（等价：草稿的筛选就是由同一个表达式换算写进去的），一格钉住“扣住前一笔、点击、改草稿、放开：体是新草稿、基线是回答”；两句边界（`confirmed` 为空不发、照失败处理，在这条路上不会为空；离开之后才轮到的体是离开前的草稿，预检 m3）各有说法、一格；HOC 的工作区一半重写；P4 R17 关闭。
- **R11 视图页以 `globalViewId` 为 key，默认视图不读，已存视图四种状态**（D-11）：§2.8。工作区视图没有开关、没有停用页（`NAV:103-133`、`:137-152` 核过）；项目视图的 id、访客都算“不存在”，访客不发请求；每次挂载读自己的，迟到的 `NoViewDraftError` 兑现；自己的宿主发出删除才不画。代价：P6a 的读 hook 加一个参数（项目的键不变）。
- **R12 控件矩阵**（D-12）：§2.7。代价：每格一个单测，H13、H14 (page) 三种角色各走一遍。
- **R13 入口规则 `workspaceViewsOn`，列表页只有搜索**（D-13）：§2.7、§2.10。名字不含 “sentry”。代价：两个函数与表格式单测。
- **R14 P6a 的弹窗按容器泛化，删旧弹窗与旧表单**（D-14，**L!**）：§2.11。代价：`VC/modal.tsx` 约 ±25 行，P6a 的弹窗测试加工作区的格、`projectId` 的 props 编译跟随。
- **R15 宿主、行菜单与删除弹窗按容器泛化，复制经 `useCopyLink`**（D-15，**L!**）：§2.11。代价：四个 P6a 文件各 ±10–20 行，两个小的新文件（权利与地址），单测加工作区的格。
- **R16 新接口 `listWorkspaceLabels`**（D-16，**负责人**）：§2.1。理由：与 `listWorkspaceStates` 同构、一个一致的快照、Plane 相同、对 agent 也是一次调用。代价：服务端约 400–600 行（含测试与故事）；Go 两路与 `gen-check` 进检查点；设计 §5 加一行。
- **R17 目录在三页挂载时读，写进标签 store**（D-17，**L!**）：§2.2。持有的形状照状态 store（按工作区持有，按项目给出），R32 确认。代价：标签 store 约 +30 行；S2 两组访问各加一行；草稿页与个人页的测试的读数照允许清单改。
- **R18 个人页的分组与筛选读目录，I19 两处改写**（D-18，**L!**）：§2.12。理由：M4/P13 的“目录归 M7”（`docs/reviews/re-19-M4P13整体审核.md:164`），P6b 正是那个“M7 补其领域验收”。代价：I19 两格改写，不变长。
- **R19 命令面板不改**（D-19）。
- **R20 旧层全删**（D-20，**L!**）：§2.13。`common/applied-filters` 有读者，不删（条件不成立）。代价：删约 900 行；M2/M3 的两份会话测试换例子。
- **R21 关键词**（D-21）：§2.13。
- **R22 S2 的移除冒烟整份删**（D-22）：视图不存在的页面进 H13 (page)。删除在 T9（视图页的同一提交），不在 H14 的 Task：它唯一的一行（`s2-issue-list-removal.spec.ts:15`）钉着今天的视图页，T9 之后必红，没有红的提交。
- **R23 检查点与终检同 P6a 的六个页面配置**（D-23）：§2.15。
- **R24 空态照 Plane**（D-24）：§2.8；按钮照矩阵只给管理员与成员（Plane 给访客一个停用的）。
- **R25 文案经 `t()`，不嵌 iframe**（D-25）：§2.14；新键见 R29。
- **R26 16 个 Task，检查点在引擎改动之前、视图页与另存之后、删除之后**（D-26）：plan。
- **R27 P6a 的回填**（D-27）：T1。
- **R28 P6a 的教训进实施者简报**（D-28）。
- **R29 新增六个 `workspace_views.*` 键**（D-29）：`workspace_views.not_found.{title,description,action}`（en：“View does not exist”、“The view you are looking for does not exist, or you may not see it.”、“Go to views”）与 `workspace_views.action.{create,update,delete}.success`（en 同 `project_views.action.*`），en 与 zh-CN 同一提交；工作区的页面不读 `project_views.*`。理由：键名跟着容器，项目视图以后改文案不牵连工作区页。代价：两份语言文件各六个键；按容器取键的一张表（§2.11）。
- **R30 “不存在”回到工作区视图的列表页**（D-30）：§2.8。动作是 `/{slug}/workspace-views`、文字 “Go to views”。理由：与 P6a 项目视图的“不存在”回列表页一致。代价：plane-diff 一行（Plane 回所有工作项，§7 第 11 条）。
- **R31 同名状态各是一项**（D-31）：§2.4。不改 M4 的筛选配置；带项目标识交 M8（§5，阶段记录 §7 写一条）。
- **R32 目录的持有形状**（D-32）：§2.2。接受起草者的做法：按工作区持有，`getProjectLabels` 对调用者是有效成员、未归档的项目回落到它，不逐项目写列表（理由见本节末补记第 3 条）。状态 store 的回落没有归档守护，交 M8（§5）。

**起草时的补记**（起草者核出的与调研、裁定的出入，各自的定法；主控 2026-11-27 照收，台账 `rulings.md` 的“起草之后的裁定”一节）：
1. **表格根的接缝已在**（调研 Q7 说没有）：`base-spreadsheet-root.tsx:37` 的 `canEditPropertiesBasedOnProject?` 与 `:80-90` 的使用。R7 只传它。
2. **`GlobalIssues` 不继承 `PagePreferencesIssues`**（调研 Q1 (a) 与 scope §3 这样写）：后者的 `fetchPage` 只造项目范围（`page-preferences-issues.store.ts:30-43`），照 `ProfileIssues` 继承 `BaseIssuesStore`。
3. **目录的持有形状**（调研 Q17 (a) 写“对每个项目写一份完整的项目列表，没有标签的写空列表”）：照状态 store 的先例按工作区持有、`getProjectLabels` 回落到它（§2.2）。理由：先例同构（`state.store.ts:63-64`、`:114-121`、`:220-224`）；对每个项目写列表要拿项目 store 的成员身份去补“空列表”，那一刻的项目 store 与服务端的快照可以不一致；按项目读的读者看到的相同。
4. **`DIALOG_STORES` 不加 `GLOBAL`**（调研 Q7 (a) 写加）：它只管分组头的“+”，表格不分组；拖放、列表看板日历三个根同样不接（§2.3）。
5. **S2 的删除在 T9**（R22 的补记）。
6. **默认视图的浏览器存储不复用 `handleIssuesLocalFilters`**（调研 Q3 写沿用它的四元组）：它按控件词校验（§2.4）。
7. **页面 id 是复合的**：服务的 `get(id)` 收不到 slug（§2.4）。
8. **`usePageDisplay` 要按表格取选项**：它按持有的布局取（读布局 `use-page-display.ts:47`、取选项 `:55`），`list` 视图会没有 Display 的选项（§2.3 第 5 点）。
9. **状态的选项**（调研 Q5 写“工作区状态目录已在外壳”）：状态 store 没有工作区级的 getter（`state.store.ts:32-49`），选项取各项目的 `getProjectStateIds`（回落到外壳读的目录）；同名状态各是一项（§7）。
10. **I19 的正则匹配不到目录的读**（`:336`、`:485`、`:491` 只认 `/projects/{id}/labels`）：T4、T5 之后它们照旧过，只是 `:460` 的标题不再是真的；改写按 R18 放在 T14（不是红的提交）。
11. **`common/applied-filters` 有读者**（`VC/applied-filters/root.tsx:11-12`）：R20 的条件不成立，不删。
12. **`WV/layout.tsx` 直接用 `swr`**（`:22-26`），不在 swr 名单里：v0 7.7 的偏差，随 T9 删除了结。

## 4. 验收标准

- [ ] **A1 接口**：契约、生成物、`gen-check`；用例、查询、处理器各有单测；规则表与 `Actions()` 一致（bootstrap 的并集测试）；权限矩阵一行三格；可见性两个测试在第 0 路（桶号写进台账）；H13 (API) 一份；设计 §5 一行。
- [ ] **A2 目录**：标签 store 按工作区持有、按项目给出、写入两处都写；三页挂载时各读一次（各自的门），外壳不读；S2 与 `mounts.ts` 照实；swr 名单。
- [ ] **A3 引擎**：`GLOBAL` 在 §2.3 的每个分派点（每处单测一格、变异一行）；范围臂的 `allowsRole`、`readPage`、`covers`、身份；`ListContext` 的工作区臂与 `issueMembership` 的工作区支（“分配给我”改负责人行离开、“我关注的”走重读、别的工作区的确认不动这张表）；命令写全、不给的两个写明；两个 `switch` 穷尽；钉退役的测试（两份文件三处）照 §2.3 改；项目页、迭代页、模块页、项目视图页、个人页不回归（m4-list、m4-filters、m4-detail、m6-pages、m7-pages）。
- [ ] **A4 筛选与偏好**：11 个词；`toFilterExpression` 收 `project_id`、个人页的绕行删去；`PagePreferencesFilter` 只改类型；默认视图的显示存浏览器（按账户、工作区、视图）、筛选只在这次进入、读回的校验、每次挂载读自己的（不经 SWR，卸下立刻再挂上照常画出）；Display 没有 `cycle`、`modules`；已存视图的草稿卸下即丢、迟到的播种不建回。
- [ ] **A5 默认视图**：四行的表；自己的词在查询里、不在词表里；死帮手与旧类型删去。
- [ ] **A6 数据层**：工作区视图的列与建；按容器的列表；`getViewById` 按种类判可见；改删写进工作区的列表；`globalView` 删去，`root.store.test.ts` 19、`store-context.test.ts` 换例子。
- [ ] **A7 入口与矩阵**：§2.7 的每一格有单测，H13、H14 (page) 三种角色核过；名字不含 “sentry”。
- [ ] **A8 视图页**：以 `globalViewId` 为 key；默认视图不读视图；已存视图的四种状态、访客不发请求、项目视图的 id 算不存在、每次挂载读自己的；只渲染表格、按行判可编辑、不给快速新建；`list` 视图照表格画、原样存回；页头与面包屑链接；空态。
- [ ] **A9 另存与更新视图**：五种工作区页给另存，默认视图另存带它的词、明写表格；“更新视图”的请求体整个在轮到时从草稿算（单测一格、变异一行；两句边界各一格）；P6a 的“更新视图”不回归（m7-pages 的 H12）；在途停用照 P6a；工作区 HOC 去掉 Plane 的分支。
- [ ] **A10 列表页、弹窗、宿主、菜单、删除**：按容器泛化、项目一侧不回归；只有搜索；默认行的菜单；复制经 `useCopyLink`、两个 `copy-hook` 例外随旧组件在 T9 删去；删除弹窗按容器取页面的视图 id，在工作区视图自己的页头删除之后转到列表页、不闪“不存在”。
- [ ] **A11 个人页与删除**：分组的组名与筛选的选项来自目录；I19 两处改写不变长；§2.13 的表逐项删去、读者写进阶段记录；`old-view-urls` 有样例；`workspace-address-route` 的名单；oxlint 等于上限、新文件零告警；knip；死成员 19 行清零；plane-diff；前端改动清单；交接的处理结果（§5）。
- [ ] **A12 e2e 与收尾**：§2.15 的新故事三引擎各一次、chromium `--repeat-each=3`；六个页面配置与整套默认 e2e 不回归；S2 移除冒烟删去；终检、最终分支 CI、合并、main CI 与归档（由下一段的 T1 补记）。

## 5. 交接与后续责任

**本段处理的输入交接**：都写“处理结果（M7/P6b）”；写关闭的照写关闭，其余照设计 §11 保持 open 至收尾。T15 写。

| 交接 | 条款 | 本段 | Task |
|---|---|---|---|
| `D:240`（§7 数据层） | 五个领域的 store 与 service 照 v0 7.7 重写、删手写类型、旧地址各一条 `old-…-urls` | 视图一半关闭：§2.6、§2.13 | T6、T11 |
| `D:242-243`（§7 先收起） | P4 收起的工作区 HOC 分支 | 关闭：§2.9（P4 R17 的工作区一半） | T10 |
| `D:246`（§7 视图） | 工作区视图目录、默认四个与已存视图的列表（`WORKSPACE` 范围与对话框宿主） | 关闭：§2.3、§2.8、§2.10（对话框宿主不接 `GLOBAL`，理由见 §2.3） | T8–T11 |
| `D:248`（§7 交接的页面项） | 弹窗名、字段提示、嵌套按钮、行名控件、下拉框、Escape、复制经 hook | 工作区视图的部分逐项见下 | T7、T9、T11 |
| `D:309`（§12 P6b） | H13、H14 的页面、H20 的工作区视图；工作区标签目录 | §2.2、§2.15 | T2–T5、T12–T14 |
| `HO/M4-closeout.md:51-58`（M7-03） | `GLOBAL`、布局根、`workspace-views/[globalViewId]`；跨项目列表；完整的工作区标签目录；旧 `WorkspaceService` 的四个方法；`:58` 的验收词 | 关闭。验收词逐个的去处：详情——H13 (page) 的已存视图页；列表——H13 (page) 的四个默认视图；筛选——H13 (page) 的筛选（含项目与标签）；跨项目访客——H13 (page) 只看自己的访客；无权项计数——接口故事 `h13-workspace-list.spec.ts:101`（总数按数据库逐条），页面只核行；失败重试——H13 (page) 的列表读与视图读各一步；会话切换——H20 (page) 另一个标签页换账号 | T2、T4、T8、T9、T11、T12、T14 |
| `HO/M4-closeout.md:103`（M7-04 第五条的工作区一半） | 工作区 HOC 的写后关闭、工作区视图的弹窗 | 关闭：§2.9、§2.11 | T7、T10 |
| `HO/M3-closeout.md:20` | 跨项目的标签列表、工作区级标签的前端（筛选、按标签分组）、工作区一级显示的工作项上的标签 | 关闭：§2.1、§2.2、§2.12（通知预览照旧按项目读，P5b T9） | T2、T4、T5 |
| `HO/M3-closeout.md:23` | 工作区视图的筛选列 | 页面一半关闭：§2.4 | T8 |
| `HO/M3-closeout.md:26` | 挂载时的取数按权限启用，S2 的清单随之改 | 目录与已存视图的读按门；S2 照实 | T4、T5、T11 |
| `HO/M3-closeout.md:37`、`:41` | 界面状态在以 id 为 key 的布局之下；`workspace-views/[globalViewId]` 的 key | 关闭：§2.8、§2.10 | T9、T11 |
| `HO/M3-closeout.md:47-48` | 弹窗的 Tab、Escape、点外面、焦点、portal；弹窗的名字 | 工作区视图的弹窗关闭（P6a 的弹窗，§2.11；H14、H20 (page)） | T7、T13、T14 |
| `HO/M3-closeout.md:56`、`:59`、`:61` | 嵌套按钮；行的控件以行为名；右键菜单以右键核 | 工作区视图的行关闭 | T11、T13 |
| `HO/M3-closeout.md:65` | `FiltersDropdown` 的 M7 调用方 | 工作区视图页的 “Display” 一步 | T9、T14 |
| `HO/M3-closeout.md:70` | `workspace/views/form.tsx` 的名字提示 | 关闭：旧表单删去，P6a 的表单已照 `FieldError` | T11 |
| `HO/M3-closeout.md:84`、`:88`、`:91` | 地址只从路由；复制只经 hook；文案经 `t()` | 工作区视图的部分关闭 | T9、T11 |
| `HO/M2-closeout.md:45`、`:56`、`:78` | 切换器 `BreadcrumbNavigationSearchDropdown` 的核对；工作区视图两处复制；是否嵌 iframe | 关闭：切换器以键盘选（H20 (page)）、复制经 hook、不嵌 iframe | T9、T11、T14 |
| `HO/M6-closeout.md:36-41`（M6-M7-03 的工作区一半） | 工作区视图没有迭代、模块词（§15 D4） | 页面照它：词表不给迭代、模块词，Display 的属性去掉 `cycle`、`modules`（§2.4，预检 m1） | T8 |
| `HO/M6-closeout.md:50-62`、`:76`（M6-M7-04 的工作区一半） | 工作区视图的 store、页面与 M3 交给它的项 | 关闭 | T6–T11 |
| `specs/P4-web-notifications.md:379-381`（P4 R17） | 工作区 HOC 的保存与更新分支 | 关闭 | T10 |
| P6aR §7 的 P6b 各条（`:179-187`） | 旧类型；`useViewMenuItems` 的工作区读者；`old-view-urls`、两例外；S2 最后一行；`[globalViewId]` 的 key；工作区 HOC；O1；`applied-filters` 的重复；19 行死成员 | 逐条关闭；`applied-filters` 的重复交 M8 | T8–T11 |

**后续**：
- **M8**：`project/applied-filters` 与 `common/applied-filters` 的重复；`invalid.tsx` 的英文（P6a 的那一条照旧）；同名状态的选项不带项目标识（R31：要带就是 M4 共用的筛选配置 `C/hooks/work-item-filters/use-work-item-filters-config.tsx` 的改动，阶段记录 §7 写一条，M7 收官时进 M8 的交接）；状态 store 的 `getProjectStates` 对已归档项目回落到工作区目录得 `[]`（R32：标签 store 在 §2.2 加了范围判断，状态 store 没有，同形状，不是 P6b 引入的）。
- **M7 收官报告**：十一条 L!（§3）；R16、R8 是负责人的；合同默认布局与页面的不一致（R8）。

## 6. 风险

- **引擎的分派**（R1）：漏一个分派点，两个 `switch` 是编译错，其余多是静默的（§2.3 的清单）；“行在不在这张列表里”漏了工作区支，行内编辑与新建之后列错（预检 I1）。缓解：每处单测一格一变异，成员判断三格三变异；“不接”的各写理由；T8 之后 chromium 的 m4-list、m4-filters、m4-detail 与检查点 0 比对。
- **共用的偏好基类**（R2）：类型界放宽、`toFilterExpression` 收 `project_id`。缓解：项目页、迭代页、模块页、项目视图页、个人页的既有单测不改全过；m4-filters。
- **“更新视图”的来源换了**（R10）：P6a 的“更新视图”从“点击的表达式 + 草稿的显示”变为“整个草稿”。缓解：两者在正常流程里相等（草稿的筛选是同一个表达式换算写进去的）；P6a 的单测不改断言全过、H12 (page) 不回归；新格钉住结构上的性质。
- **目录的回落**（R17）：`getProjectLabels` 回落到目录之后，`prepareListCatalog` 与各读者不再逐项目读。缓解：只对目录范围里的项目回落；单测“目录在时按标签排序不再读项目标签”“已归档的项目不回落”；I19 与 H13 (page) 的读表。
- **数据层的容器**（R9）：`getViewById` 的可见按种类判，工作区视图对访客不给。缓解：单测四格（项目视图、工作区视图 × 角色），P6a 的格不改断言。
- **浏览器存储**（R3）：坏的存储值。缓解：逐字段校验与默认；单测“坏的 JSON、错的类型、存储不可用”。
- **既有测试**：按调用数断言、目录的读会让它们变的单测——草稿页 `workspace-draft/root.test.tsx:59-75`、`page.test.tsx:98`、`:104-129`（T4）；个人页 `profile-page.test.tsx` 的三格 `:253-263`、`:264-294`、`:296-326`（`:262`、`:293`、`:313` 钉的正是 R18 要翻过来的行为），以及渲染个人页的两份 M4 列表单测 `IL/list-data.test.tsx:248-262`、`IL/attachment-count.test.tsx:300-311`（T5）。`profile-route.test.tsx` 把个人页换成替身（`:44`），不受影响；`IL/properties/row-planning-properties.test.tsx:269-283`、`workspace-draft/draft-planning.test.tsx:108-134` 按路径取调用，也不受影响。另有 S2 两份；I19；钉退役的两份文件三处；两份会话测试。全部写进允许清单（预检 I3）。
- **WebKit 的预载与本机注销**：同 P6a（路由只在 Chromium、目标页代码载入之后装；注销之后同一 HEAD 补跑）。
- **时长与 CI**：单测约 1,500–2,000 行新增；web 作业在慢 runner 上已越 960 秒（P6a 最终分支 CI 1046 秒），照 R23 只报告；e2e 加 5 个文件，chromium 合计估 60–90 秒（估算，未测），分到两片各约 +30–45 秒；Go 第 0 路加两个可见性测试（估 10–20 秒）；第 1 路不加顶层测试，只有 `TestPermissionMatrix`（桶 46）多三个并行的子测试与一个检查（量很小，检查点 0 的基线已含）。

## 7. 与 Plane 的差异（T15 登记进 `docs/v0/plane-diff.md` 的行为差异一节，与 M7/P6a 的行同处）

1. 工作区视图页只渲染表格：存着 `list` 布局的视图照表格画、原样存回，网页新建与另存写 `spreadsheet`；接口不给显示时默认 `list`（负责人：不改合同）；Plane 对非表格布局什么都不画（`PW/core/components/views/helper.tsx:25-56`）（R8）。
2. 默认视图自己的条件固定在查询里、不在筛选词表里；Plane 以它覆盖同名的筛选（`PW/core/store/issue/workspace/filter.store.ts:139-145`）（R4）。
3. 从默认视图“另存”带它的 `me` 条件；Plane 只存页面的筛选（`PW/core/components/issues/issue-layouts/roots/all-issue-layout-root.tsx:135-147`）（R10）。
4. 工作区页的筛选词是契约的 11 个（含 `state_id`、`mention_id`、`subscriber_id`）；Plane 9 个（R5）。不同项目的同名状态各是一个选项、不带项目标识（R31，交 M8）。
5. 访客只有四个默认视图，不读已存视图，打开已存视图的地址看到“不存在”、不发请求；“Add view”、编辑、删除照矩阵；Plane 都给（R11、R12）。
6. 面包屑 “Views” 链到列表页；Plane 不链（R6）。
7. 工作区标签目录在工作区视图页、个人页的工作项、草稿页挂载时读，一个接口；Plane 在工作区视图、草稿、通知页读，个人页不读（`PW/core/hooks/use-workspace-issue-properties.ts:40-45`）（R16、R17）。
8. “更新视图”的请求体整个在轮到时从草稿算（另见 P6a 的第 2 条）（R10）。
9. 工作区页的空态按钮只给管理员与成员；Plane 给访客一个停用的（R24）。
10. 视图的弹窗只管名字、说明、图标；行一个右键菜单、菜单按钮以视图名为名、复制失败有提示（P6a 的第 4、7 条扩到工作区视图）（R14、R15）。
11. 已存视图不存在（含访客打开、项目视图的 id）时，动作回到工作区视图的列表页；Plane 回所有工作项（`PW/core/components/issues/issue-layouts/roots/all-issue-layout-root.tsx:115-130`）（R30）。

（Plane 的行号由 T15 对照主目录的只读 `plane/` 逐一核过。）

## 8. 起草时的待定（已裁定）

起草时交主控的四件，主控 2026-11-27 已裁定（台账 `rulings.md` 的“起草之后的裁定”一节），各自并入正文：
- 新文案键 → R29（§2.8、§2.11、§2.14）。
- “不存在”的动作 → R30：回到列表页 `/{slug}/workspace-views`（§2.8、§2.15、§7 第 11 条）。
- 同名状态的选项 → R31：不改，带标识交 M8（§2.4、§5、§7 第 4 条）。
- 目录的持有形状 → R32：照起草（§2.2、§3 补记第 3 条、§5）。
