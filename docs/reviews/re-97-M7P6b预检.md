# M7 / P6b 预检

> 2026-11-27：结论 **修正后通过**（Ready after fixes）。严重 0、重要 5、次要 9、观察 6。合同与服务端在 `6dead17d` 上都撑得住：`listWorkspaceLabels` 照 `listWorkspaceStates` 的每一层都有先例可照（`PRJ:597-622`、`rules.go:86`、`SV/app/list_workspace_states.go`、`states.sql:81-92`、`permission_matrix_states_test.go:111`、`:178`），两个新的顶层测试名的桶是 12 与 1、都在第 0 路；控件矩阵每一格对得上 `rules.go:187`、`:235-238` 与 `view.go:58`、`:65`；工作区列表的 11 个词、`me`、平铺与 422 同 `ISS:384-404` 与生成的类型；工作区视图只收 `spreadsheet` 或 `list`、分组为空（`NAV:635-644`），R8 的“`list` 照表格画、原样存回”因此是完整的。spec、plan 与简报引的行号核了约 200 处，plan 里约 85 个文件的行数逐个量过，都对；偏差七处，各差一两行或是说法（m9）。十一条高风险裁定方向都对，O1 的选法（轮到时整个从草稿算）站得住、`t10-mut-filters-at-click` 杀得死。P6a 晚发现的三类里，m1（迟到的 `NoViewDraftError`）与 O1 对已存视图已经预先堵住；另有两处同类的口子没堵（I2 默认视图的偏好读、I5 删除弹窗的路由参数）。五条重要：I1 引擎的第三臂漏了“行在不在列表里”的判断与三处编译跟随；I2；I3 T5 的允许清单点错了文件；I4 两个旧组件的最后读者在 T9 而不是 T11；I5。
>
> - **目标**：[P6b spec](../v0/M7-collaboration/specs/P6b-web-workspace-views.md)、[P6b plan](../v0/M7-collaboration/plans/P6b-web-workspace-views.md)（未提交的草稿），次要目标台账的 `implementer-brief.md` 与 `final.sh`。依据台账 `rulings.md`（D-1–D-32；D-16、D-8 是负责人的）、`research/scope.md`、`research/questions.md`，[M7 设计](../v0/M7-collaboration/M7-design.md) §2、§3.7、§5、§7、§12，[总体设计](../v0/v0-design.md) §7.7，合同 `api/modules/{project,navigation,issue}.yaml`，先例 [re-96](re-96-M7P6a预检.md)、[P6a spec](../v0/M7-collaboration/specs/P6a-web-project-views.md) 与 [P6a 阶段记录](../v0/M7-collaboration/reviews/P6a-web-project-views-review.md) §7、§9。
> - **基线**：main `6dead17d`（P6a 的 noff，`6dead17d7aa199f8d124cfdb403eacc64aedb1e5`），worktree `codex/m7-p6b`。

下文 `C/` = `web/apps/web/core/`，`W/` = `web/apps/web/`，`WV/` = `W/app/(all)/[workspaceSlug]/(projects)/workspace-views/`，`WC/` = `C/components/workspace/views/`，`VC/` = `C/components/views/`，`IL/` = `C/components/issues/issue-layouts/`，`PKG/` = `web/packages/`，`E/` = `e2e/`，`K` = `tools/keywords.json`，`HO/` = `docs/v0/M7-collaboration/handoffs/`，`D` = M7 设计，`NAV`、`PRJ`、`ISS` = `api/modules/` 的 `navigation.yaml`、`project.yaml`、`issue.yaml`，`SV/` = `server/internal/modules/project/`，`BS/` = `server/internal/bootstrap/`，`P6aR` = P6a 阶段记录，`L/` = 台账 `.superpowers/sdd/P6b-web-workspace-views/`。

## 1. 方法

一次 fresh 只读审查（`fable`）：只写本文件，不动 git 状态（只用了只读的 `git status`、`git grep`、`git log`，另有一次 `git ls-files` 找 `.oxlintrc.json` 的位置），不起容器，不跑测试、构建与安装。跑过的只有 `grep`、`sed`、`cat`、`wc`、`ls` 与台账的 `doc-links.py`（写本文件之前 243 个文档、2491 个链接、0 断，两份草稿在内；写完之后的数字见本节末）；管道里另用 `cut`、`head`、`awk`、`tr` 裁剪输出，`printf %s <名字> | cksum` 算过三个测试名的桶，有一次用 `python3` 只读地在 en 的语言文件里查键在不在，都不写任何东西。`node tools/keywords.mjs`、oxlint、knip、deadsym、vitest、Playwright、Go 测试都没有运行。审查前后 `git status` 只有两份草稿是未跟踪的（之后多本文件）。

worktree 没有装依赖。`swr@2.4.2` 的行为是读主目录 `node_modules/.pnpm/swr@2.4.2_react@19.2.8/` 里装着的源码核的（只读：`dist/index/index.mjs` 的去重标记、成功一支的竞态核对 `:423`、失败一支不核对就写进键 `:471-477`）。`plane/` 没有在里面跑 git。

对照代码核了：

- 合同与服务端：`PRJ:597-646`、`:1381`，`api/openapi.yaml:120-121`，`NAV` 的七个操作与 `View`、`ViewFilters`、`ViewCreate`、`ViewUpdate`，`ISS:384-404` 与 `IssueListOrder`、生成的 `listWorkspaceIssues` 查询类型，`PKG/api-client/src/index.ts`（工作区列表在 CSV 的路径名单里，`:35-40`）；`access/domain/rules.go`、`rules_test.go`，`navigation/domain/{view,content}.go`；project 模块的 `app/{list_workspace_states,state_ports,label_ports}.go`、`domain/actions.go`、`adapter/postgres/{queries/states.sql,labels.go,labels_test.go,states_test.go:279-332,failures_test.go}`、`adapter/http/{labels,states,handler,handler_test,labels_test,states_test}.go`、`module.go`；`BS/` 的三份矩阵测试、`project_visibility_test.go:60-135`、`membership_world_test.go`；`tools/go-test-shard.sh` 全文。`listWorkspaceStates` 在仓库里被点名的每一处都用 `git grep` 列过（m2 的出处）。
- 列表引擎：`PKG/types/src/issues/issue.ts`，`C/store/issue/helpers/{issue-list-read,issue-list-binding,issue-list-catalog,base-issues.store,page-preferences-filter.store,page-preferences-issues.store,issue-filter-helper.store}.ts`，`C/lib/{paged-groups,issue-membership,issue-filters,planning-switch,local-storage}.ts`，`C/store/issue/{root.store,permission-reads}.ts`、`profile/`、`project-view/` 的全部，`C/services/issue/issue-list.service.ts`；分派点 `C/hooks/store/{use-issues,use-page-issue-filters}.ts`、`C/hooks/{use-issues-actions.tsx,use-issue-layout-store.ts,use-issue-list-mount.ts,use-route-list-words.ts}`、`C/components/issues/use-page-display.ts`、`issue-modal/base.tsx:50-100`、`IL/{command-host,utils,row-expansion,group-columns}.tsx`、`IL/empty-states/index.tsx`、`IL/spreadsheet/{base-spreadsheet-root,spreadsheet-view}.tsx`、`IL/filters/header/display-filters/display-properties.tsx`、`PKG/constants/src/issue/{filter,common}.ts`、`PKG/constants/src/workspace.ts`。读 `scope.query.*`、`ListContext`、`EIssuesStoreType.PROFILE`、`isWorkspaceLevel` 的每一处用 `git grep` 列过。
- 视图的数据层与组件：`C/store/views/{view.store,view-changes}.ts`、`C/services/views/views.service.ts`、`C/lib/{views-entry,reconciled,use-session-swr}.ts`、`VC/{modal,quick-actions,delete-view-modal,view-content,use-view-page-read,actions-host}.ts(x)`、`VC/` 里读路由参数的每一处；`filters-hoc/` 的全部产品文件、`fake-view-saving.tsx`、`update-view.test.tsx:60-203`、`update-view-busy.test.tsx:146-165`，`PKG/shared-state/src/store/rich-filters/filter.ts` 的 `canSaveView`、`canUpdateView`、`resetExpression`、`updateView`。
- 旧的工作区视图层：`WV/` 的五个文件、`WC/` 的八个文件、`C/store/global-view.store.ts`、`C/hooks/store/use-global-view.ts`、`C/services/workspace.service.ts`、`PKG/types/src/{workspace-views,views,view-props}.ts`；每个旧组件的导入者、`globalView`、`TStaticViewTypes`、`my_issues`、`useViewMenuItems` 的读者（`git grep`）；`W/app/routes/core.ts:75-82`、`:148-161`。
- 标签：`C/store/{label,state}.store.ts`、`C/hooks/use-project-labels-read.ts`、`C/layouts/auth-layout/{use-workspace-fetch,use-project-fetch}.ts`，`getProjectLabels`、`getProjectLabelIds`、`fetchProjectLabels` 的全部读者，`C/components/profile/profile-issues.tsx`、`C/components/issues/workspace-draft/root.tsx`；渲染个人页与草稿页的每一份单测（`git grep` 列出六份，逐份读了按调用断言的地方）。
- 门禁与 e2e：`L/{gates,final}.sh`、`Makefile:124-128`、`knip.jsonc`、`tools/keywords.mjs`（过期与 stale 例外的检查）、`K` 的 `project-address-route`、`workspace-address-route`、`old-view-urls`、`plane-workspace-urls`、`copy-hook` 两个例外、phase，`.oxlintrc.json:121-232`，`W/package.json:10`；`E/playwright.m7-pages.config.ts`、`E/fixtures/{mounts,views}.ts`、`E/stories/smoke/{s2-web-app,s2-issue-list-removal}.spec.ts`、`E/stories/issue/i19-member-issues.spec.ts:330-340`、`:455-495`；e2e 里能匹配到新目录地址的断言（只有 I19，正则只认项目的）；en 的语言文件里 spec §2.14 列的 15 个已有的键。
- 交接与文档：`HO/M3-closeout.md`、`M4-closeout.md`、`M6-closeout.md`、`M2-closeout.md` 被引的每一行，设计 `:40-47`、`:118-121`、`:192-211`、`:240-248`、`:309`、`:353`，`P6aR:3`、`:51`、`:177-188`、`:244-247`，P6a spec `:335`，`specs/P4-web-notifications.md:270`、`:379-381`，`v0-design.md:638`，`frontend-changes.md:206`，`re-19:164`。

没有核实的：

- 任何要运行的东西：oxlint 的 14 条落在哪些文件、knip 与死成员在各 Task 之后的结果、vitest 与页面故事的时长、关键词的实际结果。I4 的“T9 的 knip 必红”是按 `knip.jsonc`、`Makefile:126-128` 与导入表推出的。
- I1、I2、I5 的后果是按引擎、swr 与组件的源码推出的，没有在浏览器里看。
- P6a 的四份 HOC 测试在 O1 换来源之后是否不改断言全过：读了 `update-view.test.tsx:80-162` 的两格，判断成立（草稿的筛选与点击时的表达式相同），没有运行；plan 已有“请求体因此变就停下报告”的出口。
- `research/questions.md` 只读了目录，`research/scope.md` 读了 §1.5–§1.10；P6a 阶段记录的 §3 没有逐条读。
- plan T1 的回填数字（终检、CI 的编号与秒数、归档名）没有对归档的台账；main CI `45778376995` 没有读（不许用 `gh`）。
- spec §7 的 Plane 行号一处没核（plan 写 T15 核）；zh-CN 的文案键；`workspace_empty_state.views.cta_primary` 的全文。
- `use-issues.ts` 的泛型 `switch` 上 `satisfies never` 的收窄（同 re-96，留给 `t8-exhaustive-probe`）。

写完之后 `doc-links.py`：244 个文档、2498 个链接、0 断。

## 2. 发现

### 严重

无。

### 重要

| # | 位置 | 发现 | 证据 | 须做的修正 |
|---|---|---|---|---|
| I1 | spec §2.3“范围臂”“绑定”、§2.16 的 T8 一行、R1 的代价；plan T8 的 Files、行数约束、Step 1、性质表 | **第三个范围臂在 spec 列的四处之外还要过引擎的三处，其中一处是语义：行在不在这张列表里。照 spec 的“同成员臂”写，“分配给我”“我关注的”两个默认视图，以及任何带状态、负责人、创建者、提及、关注条件的工作区页，在行内编辑或新建之后列错。** ① 类型：`IssueListBinding.context` 把范围整个展开成 `ListContext`，而 `ListContext` 是另一处独立的两臂联合，加臂不改它编译不过；`WorkspaceIssueQuery` 没有 `group_by`，直接读 `scope.query.group_by` 的六处与读 `p.relation` 的一处也都编译不过。这些文件不在 T8 的 Files 与 §2.16 里（其中 `paged-groups.ts` 已 445 行，过了 400）。② 语义：`issueMembership` 只有两支。项目支核 `state_id`、`assignee_id`、`created_by_id`（认 `me`），`mention_id`/`subscriber_id` 有值记“未知”；另一支（成员臂）只核工作区、归档、`project_id` 与 `relation`。工作区臂落进后一支，自己的 `me` 词与 `state_id` 等都不参与：在“分配给我”把一行的负责人改成别人（表格可行内编辑，R7），确认之后仍判它在列表里，行留着、总数不变；在“分配给我”上新建一个没有分给自己的工作项，它被直接插进列表；“我关注的”本该是“未知、重读”，成了“在”；页面自己加的 `state_id`、`assignee_id`、`created_by_id` 条件同样不参与。到下一次重取才对。③ 没有格或变异钉这件事：T8 的“编辑一行，工作区列表跟着变”（`t8-mut-covers-false`）只证写得进来。 | `C/store/issue/helpers/issue-list-binding.ts:54-61`；`C/lib/paged-groups.ts:53-56`、`:332`（只认 `"member"`）、`:320-338`；`C/lib/issue-membership.ts:54-75`（两支）、`:72-74`（`relation`）、`:90`、`:103`；`issue-list-read.ts:76`、`:135`；`issue-list-catalog.ts:92`；`base-issues.store.ts:96`；测试替身 `C/store/issue/fake-issues.ts:107` | spec §2.3 写全：① `ListContext` 加工作区臂；臂的查询类型写成 `WorkspaceIssueQuery & { group_by?: never; sub_group_by?: never }`（读 `group_by` 的六处不改即得 `undefined`），或逐处写改法，二选一写明。② `issueMembership` 加工作区支：工作区、未归档、`project_id`（同成员支）＋ `state_id`、`assignee_id`、`created_by_id`（同项目支，认 `me`）＋ `mention_id`/`subscriber_id` 有值即“未知”；没有 `relation`。`paged-groups.ts:332` 改为“不是项目臂”。③ plan T8 的 Files 与 §2.16 加 `C/lib/paged-groups.ts`（只改类型与一行，写明不变长）、`C/lib/issue-membership.ts`（112）、需要时 `fake-issues.ts`。④ Step 1 加格、性质表加行：“分配给我”上改负责人，行离开、总数减一（`t8-mut-membership-no-assignee`）；`state_id` 条件下改状态同理；“我关注的”上的确认走重读、不直接插入（`t8-mut-membership-subscriber-known`）；别的工作区的确认不动这张表（`t8-mut-foreign-workspace`）。H13 (page) 的“行的属性可编辑”一步放在“分配给我”上改负责人，断言行离开。 |
| I2 | spec §2.8“默认视图”、§2.4 的 `discard`；plan T9 Step 1、性质表的 `t9-mut-mount-read-shared` | **默认视图的偏好“挂载时读、卸下时丢、偏好在了才画”没有写机制；照个人页的先例写，两秒内再进来页面是空的（re-96 I1 的同一类）。** 个人页以普通的 `useSessionSWR([...], fetchFilters)` 读偏好，它卸下时不丢，所以没事。默认视图卸下要 `discard(pageId)`（D-3）；同样经 SWR 的键读的话，离开之后在去重窗口（2000 毫秒，这里没有缩短）里回到同一个默认视图（切换器来回、后退前进），取数函数不被调用，`filters[pageId]` 一直是空的，布局根不画，没有加载也没有错误。plan 只为已存视图写了“卸下立刻再挂上”的格与变异；默认视图没有，H13 (page) 的“离开再进来还在”落不落在两秒里看机器。 | `C/components/profile/profile-issues.tsx:49-51`；`VC/use-view-page-read.ts:30-36`（P6a 为同一件事写的说明）、`:40-58`；`C/lib/use-session-swr.ts:26`、`:35-46`；swr 2.4.2 `dist/index/index.mjs` 的去重标记 | spec §2.8 写明机制，二选一：(a) 不经 SWR——布局里一个 effect 调 `fetchFilters(slug, pageId)`、清理时 `discard(pageId)`（读的是浏览器，服务不抛错）；(b) 与已存视图同一个“每次挂载读自己的”写法。plan T9 Step 1 加格“默认视图卸下立刻再挂上，照常画出，显示是浏览器里的”，性质表加 `t9-mut-default-read-shared`（换成普通的 `useSessionSWR` 读，该格失败）；H13 (page) 经切换器从“所有工作项”到“分配给我”立刻回来，断言列表在。 |
| I3 | plan 允许清单的 T5 一条与“不改、不受影响”一条、T5 的 Files；spec §6“既有测试” | **T5 的允许清单点错了文件：个人页读目录之后必红的三份 M4 单测不在清单里，清单里唯一的 `profile-route.test.tsx` 反而不受影响。** `profile-route.test.tsx:44` 把 `ProfileIssuesPage` 换成替身，目录的读不发生。真正渲染个人页、按调用断言的是：① `profile-page.test.tsx`（经路由的页面挂真的 `ProfileIssuesPage`）的三格——`:253-263`（“without prefetching directories”，`:262` 断言成员列表之外的调用数为 0）、`:264-294`（“offers only already held labels…”，`:293` 断言没有以 `/labels` 结尾的调用）、`:296-313`（“…even while the label catalog is cold”，`:313` 同）：这三格钉的正是 R18 要翻过来的行为；② `IL/list-data.test.tsx:248-262` 的个人页一行（等到恰一个调用，`calls[0]` 须是成员列表）；③ `IL/attachment-count.test.tsx:300-311`（`:310` 的 `reads()` 恰等于一个成员列表）。照“清单之外的红先停下报告”，T5 做到一半要停；照“M4 的列表与筛选测试不改”，没有合规的做法。草稿页的 `page.test.tsx` 清单只写到 `:113`，`:119-129`（等到第三个调用、`calls[2]`、`:129` 的“每个调用都以 `/drafts` 结尾”）同样要改。 | 上列行号；`W/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/[profileViewId]/page.tsx:10`、`:29`。另两份渲染这两页的测试按路径取调用，不受影响：`IL/properties/row-planning-properties.test.tsx:269-283`、`C/components/issues/workspace-draft/draft-planning.test.tsx:108-134` | plan 的 T5 清单改为：`profile-page.test.tsx` 三格照 R18 改写（`:262` 只数目录之外的调用，另断言恰一个目录的读；`:264` 一格改为“选项是目录里成员的未归档项目的标签”；`:296` 一格改为“目录答到之前照旧列出，答到之后组名齐全”）；`list-data.test.tsx:248-262` 与 `attachment-count.test.tsx:300-311` 按路径取成员列表（断言的意思不变）；`profile-route.test.tsx` 一条删去或写明不受影响；“M4 的列表与筛选测试不改”加这两处例外。T4 的 `page.test.tsx` 写成 `:98`、`:104-129`。T5 的 Files 与 spec §6 同步。 |
| I4 | spec §2.5 末句、§2.11“删去”、§2.13 的表与“死成员”；plan Global Constraints 的 knip 清单与守卫的 `copy-hook` 一条、T9 的 Files 与 Step 2、T11 的 Files | **`WC/quick-action.tsx` 与 `WC/default-view-quick-action.tsx` 的最后一个读者是 T9 删掉的 `WV/header.tsx`，plan 到 T11 才删它们：T9 的 knip 必红；新页头要用的默认视图菜单也晚了两个 Task。** 这两个文件只被 `WV/header.tsx` 导入，另有 `WV/layout.test.tsx` 的两个替身（同在 T9 删）。T9 之后它们没有导入者：`make knip`（不带排除、逐 Task 的门禁）报未使用的文件，照“删掉最后一个读者的 Task 同一提交删去”也该在 T9。删了它们，两个 `copy-hook` 例外当场成了 stale，关键词检查报错，也得同一提交删——plan 写的是 T11。另外 spec §2.8 的新页头在 T9 就要“默认视图只有复制链接与新标签页打开”的菜单，而 `WC/default-view-menu.tsx` 在 plan 里是 T11 的新文件（第一个读者写的是列表页）：T9 要么继续用旧的 `DefaultWorkspaceViewQuickActions`（写死的英文、只处理成功的复制，与 §2.8“字面量都经 `t()`”相悖，而且 T11 的 Files 里没有页头），要么提前建新菜单，两种 plan 都没写。 | `WV/header.tsx:19`、`:21`；`WV/layout.test.tsx:29-30`；`Makefile:126-128`、`knip.jsonc`；`K:3424-3437`（路径在 `:3426`、`:3433`）、`tools/keywords.mjs:182-211`；`WC/default-view-quick-action.tsx:14`、`:22`、`:34-41`；`PKG/types/src/workspace-views.ts:41-43` | T9 建 `WC/default-view-menu.tsx`（第一个读者是视图页页头，T11 的默认行是第二个），同一提交删 `WC/quick-action.tsx`、`WC/default-view-quick-action.tsx` 与两个 `copy-hook` 例外；`TStaticViewTypes`/`STATIC_VIEW_TYPES` 的最后读者随之在 T9 走掉（另两处 T8 已去：`issue-filter-helper.store.ts:18`、`:136` 与 `PKG/constants/src/workspace.ts:8`、`:36`），那三行在 T9 删，文件的其余部分照旧到 T11；死成员 `default-view-quick-action.tsx:23` 一行的去处写 T9。spec §2.5、§2.11、§2.13 的表（“八个”拆成 T9 两个、T11 六个）、plan 的 knip 清单、守卫一条、T9 与 T11 的 Files 及提交说明照改；T9 留 oxlint 计数的收据（这一步删五个旧文件）。 |
| I5 | spec §2.11“删除”、§2.8“删除之后”；plan T7 Step 1、性质表 T7 | **删除弹窗靠路由参数 `viewId` 判断“删的是不是这一页显示的视图”；工作区的路由参数叫 `globalViewId`。按容器泛化时这一处不改，在工作区视图自己的页头删除就不设 `pageLeaving`、不跳转，页面停在“视图不存在”——P6a 整审 O3 与限定复审 n1 的同一类。** 弹窗读 `useParams().viewId`，以它判 `shownByPage`、设 `pageLeaving`、决定跳不跳。spec 只写“删的是这一页显示的视图时转到 `viewsPath(container)`”，没有说“这一页显示的视图”从哪里读；T7 的格若照 P6a 的替身路由（读 `viewId`）挂工作区页，单测全绿而产品错，要到 T13 的 H14 (page)（检查点 2 之后）才露出来。 | `VC/delete-view-modal.tsx:43`、`:58-59`、`:69`；`W/app/routes/core.ts:78-81`；`VC/fake-view-pages.tsx:52`；`VC/` 的产品文件里读路由参数的只有这一处（`git grep useParams`） | spec §2.11 写明：页面显示的视图 id 按容器取（项目 `viewId`，工作区 `globalViewId`），收进 `VC/view-paths.ts` 或由宿主的 `ViewsPage` 给出，删除弹窗不直接读 `viewId`。plan T7 Step 1：工作区的格挂在真实形状的路由（`:workspaceSlug/workspace-views/:globalViewId`）上，断言删除发出之后页面不画、成功之后到 `/{slug}/workspace-views`、被拒之后照常画；性质表加 `t7-mut-delete-param-viewid`（工作区容器仍读 `viewId`，该格失败）。T9 的“只在自己的宿主删除时不画”一格同样挂真实路由。 |

### 次要

| # | 位置 | 发现 | 须做的修正 |
|---|---|---|---|
| m1 | spec §2.4“词表”、§2.8“表格根”、§5 的 `HO/M6-closeout.md:36-41` 一行；plan T8 | **Display 面板在工作区页上会给“Cycle”“Modules”两个属性开关，而表格永远不出这两列。** 面板按 `useRouteListWords` 给词：路由没有项目时每个词都给（`C/hooks/use-route-list-words.ts:17-22`，`display-properties.tsx:36-42`；注释 `:28-30` 写的是为个人页的列表与看板）；表格按路由的项目过滤列，没有项目就没有这两列（`IL/spreadsheet/spreadsheet-view.tsx:61-66`）。`my_issues.layoutOptions.spreadsheet.display_properties` 是全部 14 个键（`PKG/constants/src/issue/filter.ts:200`，`common.ts:90-105`）。spec §5 写“词表与显示不给迭代、模块”，照 §2.4 的改名做不到。 | §2.4 写明 `workspace_issues.layoutOptions.spreadsheet.display_properties` 是 14 个键去掉 `cycle`、`modules`（存着的值与 `WORKSPACE_VIEW_CONTENT` 照旧 14 个，原样存回）；T8 加格“工作区页的 Display 不给 Cycle、Modules”，性质表一行（`t8-mut-display-planning-offered`）。 |
| m2 | spec §2.1“查询”“可见性”；plan 允许清单的 T2 一条、T2 Step 2 | **T2 的测试清单少一处，另有两处与 states 的先例不同要写明。** ① `SV/adapter/postgres/failures_test.go` 的 `TestAFailedReadIsAnErrorNotAnAnswer` 对 store 的每个读各有一行（`ListWorkspaceStates` 在 `:135-136`，头注释 `:37-42` 逐个点名）；新的 `ListWorkspaceLabels` 不在允许清单里，“失败的读是错误、不是空列表”就没有人钉。② 可见性测试照 `project_visibility_test.go:75` 写时，先例把已归档的项目也接进来（states 对它答空）；`listLabels` 对已归档的项目照常列出（`permission_matrix_labels_test.go:71-74`），对不是成员的账户答 403（`:69-70`）。spec 写了“已归档的不在”，没写 403 一半。③ spec 的查询写 `@workspace_id`、`@user_id`，本仓的写法是 `sqlc.arg(…)`（`states.sql:89-90`）。 | 允许清单的 T2 加 `failures_test.go`：`ListWorkspaceLabels` 一行与头注释一句；性质表加“失败的读不答空列表”（适配器吞掉错误的变异）。§2.1 的可见性一句写成“各未归档项目的 `listLabels` 接起来，答 403 的算没有”。查询照 `sqlc.arg` 写。 |
| m3 | spec §2.9“请求体整个在轮到时从草稿算”；plan T10 Step 1 | **O1 的写法少两句边界。** ① spec 的 `body` 直接解 `drafts.confirmed(viewId)`，它的类型可以是 `undefined`（P6a 是 `draft?.…`，`use-project-view-saving.ts:121-128`）；筛选也从草稿取之后，“没有草稿”时发什么要写。② 基类的 `values` 不随子类的 `discard` 清掉（`page-preferences-filter.store.ts:155-157`，`project-view/filter.store.ts:27-28`）：页面在轮到之前离开，`confirmed` 给的是离开前的草稿，发出去的仍是用户点击时看到的；离开又进来（重新播种）则是视图自己的内容，等于一次空改。 | §2.9 加两句：`confirmed` 为空时不发、照失败处理（或写明不可能的理由）；离开之后轮到的体是离开前的草稿。T10 的 O1 一格已写“经 `updateFilterExpression` 改草稿”（不经停用的控件），保留这个写法。 |
| m4 | spec §2.4“词表”、§2.13 的 `my_issues` 一行；plan T8 允许清单、Step 2 的 `t8-readers` | **`my_issues` 改名不是“编译跟随”。** `ISSUE_DISPLAY_FILTERS_BY_PAGE` 的类型是索引签名（`PKG/constants/src/issue/filter.ts:38-40`），`WC/form.tsx:137` 的 `.my_issues.layoutOptions.spreadsheet` 改名之后照样编译，运行时是 `undefined.layoutOptions`：T8 到 T11 之间旧列表页的“Edit View”一点就抛。同理 `TStaticViewTypes` 今天就是 `string`（`STATIC_VIEW_TYPES` 没有 `as const`，`PKG/types/src/workspace-views.ts:41-43`），“键类型若随之换（编译跟随）”不会发生。 | 两处改成“按 `git grep` 跟随”，`t8-readers` 的收据写“改名之后 `my_issues` 零处、`workspace_issues` 的读者是 `form.tsx:137` 与 `ISSUE_STORE_TO_FILTERS_MAP`”；`default-view-quick-action.tsx` 一条删去（照 I4 它在 T9 删）。 |
| m5 | plan“T9 与 T11 之间的窗口”、T9 Step 2、T10、T11 的 Files；spec §2.13 的 `workspace-address-route` | **窗口的三处接缝没写。** ① 原样移过去的 `WV/(list)/page.tsx` 读 `currentWorkspace`（`WV/page.tsx:25`、`:28`）；spec 把 `WV/(list)/*` 整个列进 `workspace-address-route`，T9 若照此收入，关键词就红。② T9 的视图页用的还是旧的工作区 HOC：它不传状态的选项（`workspace-level.tsx:44-60` 只有成员、项目、标签），也不给保存与更新，所以 T9 这一个提交里 `state_id` 的条件没有选项可选。③ 列表页的搜索状态由 `(list)/layout.tsx` 提供（spec §2.10），`WC/page-state.tsx` 是 T11 的新文件，T11 的 Files 却没有 `(list)/layout.tsx`。 | “窗口”一段补：`(list)/page.tsx` 在 T11 重写时才进 `workspace-address-route`（T9 只收它的 `layout`、`header`）；T9 到 T10 之间工作区页没有状态的选项与保存；T11 的 Files 加 `WV/(list)/layout.tsx`。 |
| m6 | spec §2.2 的 `getProjectLabels`、R32；plan T4 Step 1、性质表的 `t4-mut-fallback-archived` | **回落的范围判断有两半，格只有一半。** “在目录的范围里”是成员且未归档；T4 只列了“已归档的项目不回落”。不是成员而看得到的项目（`member_role === null`，如公开项目）没有格：去掉这一半，`getProjectLabels` 对它给 `[]`，行上的标签（`C/components/issues/use-issue-labels.ts:11`）与命令面板的标签菜单（`power-k/…/labels-menu.tsx:29-33`）就把“没读过”当成“没有标签”。另外目录是一次读的快照，而“在范围里”按项目 store 此刻判：读之后才加入的项目回落到 `[]`，到三页之一再挂载重读才对；`prepareListCatalog` 自己兜得住（行上有不认识的标签就读项目的，`issue-list-catalog.ts:113-119`），别的读者没有。 | T4 加格“不是成员的项目不回落（给 `undefined`）”，`t4-mut-fallback-archived` 拆成两行或写明一个变异两格都失败；§2.2 写一句快照的限制（读之后加入的项目到下一次目录的读之前给 `[]`）。 |
| m7 | spec §5 的 `HO/M4-closeout.md:51-58` 一行、§2.15 的 H13 (page) | **M7-03 写“关闭”，它的验收词有两个没对到故事。** `HO/M4-closeout.md:58` 列的是详情、列表、筛选、跨项目访客、无权项计数、失败重试、会话切换。H13 (page) 有“不是成员的项目、已归档的项目不出”，没有计数的断言；“读失败重试”只有已存视图的读，列表的读没有。 | §2.15 的 H13 (page) 加：默认视图上页面显示的总数不含不是成员的与已归档的项目的工作项（或写明由接口故事 `h13-workspace-list.spec.ts` 承担、页面只核行）；列表读失败之后重试一步（只在 Chromium，同视图读的那一步）。§5 的那一行逐词写去处。 |
| m8 | spec §2.3“分派点（每处单测一格、变异一行）”；plan T8 Step 1“分派点各一格（spec §2.3 的 1–11）”、T9 的 Files、性质表 | **第 9 点（空态）在 T9，不在 T8；第 9、10 点各缺一个变异。** `IL/empty-states/index.tsx` 的 `switch` 有 `default: return null`（`:35-36`），T8 加枚举不动它也编译，plan 把它放在 T9 是对的，但 T8 的“1–11”与此不符。性质表里空态只有 `t9-mut-empty-guest`（按钮给谁），去掉 `case GLOBAL`（什么都不画）没有一行；第 10 点的“项目集取这一页筛选的 `project_id`”也没有。 | T8 的那一句改成“1–8、10、11（第 9 点在 T9）”；性质表加 `t9-mut-empty-case-missing` 与 `t8-mut-mount-projects-ignored`（后者的格：筛到一个项目之后，别的项目的角色变化不让这一页重取）。 |
| m9 | spec 与 plan 的行号、说法 | **一批偏差（都不影响做法）。** ① spec §2.3 第 5 点与 §3 补记 8 的 `use-page-display.ts:48`：读布局在 `:47`，按布局取选项在 `:55`。② “三份钉 `GLOBAL` 的测试”（spec §2.3、R1、A3、§6，plan T8）是两份文件三处：`use-issues.test.ts:26`，`use-issue-layout-store.test.tsx:127`、`:75-116`。③ spec §2.2 的 S2 访客行 `:207-219` 应为 `:208-219`。④ plan 三处写 `P/views`，P6b 的 spec 没有定义 `P`（P6a 的有），plan 说“缩写同 spec”。⑤ plan T1 的 `final.sh` 头注释 `:3-9`：P6a 的理由在 `:4-9`。⑥ I19 的两处正则在 `:485`、`:491`（spec §2.12 与 §3 补记 10 写 `:486`、`:492-494`；plan T14 的 `:486-494` 应从 `:485` 起）。⑦ spec §6 的 `page.test.tsx:98-113` 见 I3。 | 照改；④ 在 P6b spec 的缩写里加 `P`，或 plan 写全路径。 |

### 观察

- **o1 第 1 路不是一点不加。** `TestPermissionMatrix` 在桶 46（`cksum` 727573146），是第 1 路的（`tools/go-test-shard.sh:15-18`）；矩阵的新行给它加三个并行的子测试与一个检查。量很小，spec §2.1 自己也写了“`TestPermissionMatrix` 的子测试”，只是 §6 的“第 1 路不加”说得过满；检查点 0 的第 1 路基线已经含它。
- **o2 `["WORKSPACE_VIEWS", …]` 的迟到失败。** 列表页与切换器以普通的 `useSessionSWR` 读已存视图；swr 的失败一支不核对是不是当前的请求就写进键，换工作区再回来时一个迟到的失败能让当前的列表页显示“失败”（有重试）。与 P6a 整审 O5 同形状（`P6aR:244`，已交 M8），不是 P6b 引入的；H20 (page) 的第一步只钉成功的迟到回答。
- **o3 `project-address-route` 名单里的 `VC` 文件。** 这条规则的内容把两参、三参的 `allowPermissions(…)` 都算 hit（`K:2342`；hit 样例 `K:2352` 就是带 slug 的工作区级写法）；`VC/quick-actions.tsx`、`modal.tsx`、`delete-view-modal.tsx`、`actions-host.tsx` 在它的名单里（`K:2639-2655`）。spec §2.7 让角色一律走 `getWorkspaceRoleByWorkspaceSlug`，正好避开；简报写一句，免得 T7 换成 `allowPermissions` 之后去放宽规则。
- **o4 T4 的单配置只跑 `s2-web-app`。** 草稿页另有 `i14-drafts.spec.ts` 与 `f10-draft-files-page.spec.ts` 等故事，它们不按整张读表断言（e2e 里能匹配到目录地址的断言只有 I19，正则只认项目的），照理不受影响，但要到检查点 0 的 `make e2e` 才第一次跑到。T4 顺手单跑这两份更稳。
- **o5 `D:246` 的“对话框宿主”。** 设计那一行把“`WORKSPACE` 范围与对话框宿主”并列；spec 不把 `GLOBAL` 接进 `DIALOG_STORES` 的理由成立（`isDialogStore` 的读者只有两个分组头，`IL/list/headers/group-by-card.tsx:95`、`IL/kanban/headers/group-by-card.tsx:110`，表格不分组；行的编辑、复制经 `PROJECT` 走）。§5 已写，阶段记录照抄即可。
- **o6 T8 仍是最大的 Task**：引擎的臂、两种偏好服务、十个分派点、八个命令与三十来个变异在一个提交里；I1 与 m1、m8 之后还要多五六行变异。照 P6a T8 的先例先写方案（Step 0 已有），建议在那一步写明提交的分界（引擎的臂与换算一份，偏好与分派点一份），整审的范围照此分。

## 3. 逐项回答

### 3.1 合同与服务端的事实

都对。`listWorkspaceStates` 的路径、标签、描述、`x-problem-codes: [workspace.not_found]`、`200` 的集合型回答与 spec 照抄的形状一致（`PRJ:597-622`）；`LabelList` 在 `PRJ:1381`；`api/openapi.yaml:120-121` 是 states 的 `$ref`。规则：`workspace_state.list` 三种角色（`rules.go:86`、`rules_test.go:37`），`workspace_issue.list` 三种角色（`:187`），`workspace_view.{read,create,update,delete}` 只给管理员、成员（`:235-238`），`View.UpdatableBy`、`DeletableBy`（`view.go:58`、`:65`）。`getView` 对工作区视图只给管理员、成员，码是 `navigation.view_not_found` 与 `forbidden`（`NAV:138-152`），spec 对 `GLOBAL` 把 `forbidden` 也算“不存在”与此相合。`createWorkspaceView` 没有 `views_disabled`（`NAV:104-133`）。工作区视图的显示只收 `spreadsheet` 或 `list`、分组为空（`NAV:640-644`），不给显示时默认 `list`（`shared/list_settings.go:88`，`content.go` 的 `CheckNewView`）：R8 的三句与合同一致，页面只需处理 `list` 这一种非表格布局。存着的筛选可以带 `me`（`NAV:566-576`，“值同 `listWorkspaceIssues` 收的”），另存带 `assignee_id: ["me"]` 放得进。

### 3.2 新接口（T2–T3）

每一层的先例都在、行号都对：动作 `actions.go:57-59`、`:74-80`；端口 `state_ports.go:23-29`；用例 `list_workspace_states.go`（40 行）；查询 `states.sql:81-92`；适配器测试 `states_test.go:279-332`（次序、已归档、已删除、不在职、别的工作区，新的一格照它写）；处理器 `states.go:65-71`、`labels.go:12-21`，`handler.go:118-124`、`:166-167`，替身 `handler_test.go:83-84`、`:260-264`、`:280`；接线 `module.go:351`；矩阵 `permission_matrix_states_test.go:111`、`:178`（`toWorkspace`、`inWorkspace`、`cellOK`、`matrixLabels` 都在）；可见性 `project_visibility_test.go:75`、`:108`，`newMemberWorld` 在 `membership_world_test.go:45`。桶：`TestListingWorkspaceLabelsListsEachProjects` 的 `cksum` 是 375195812、桶 12，`TestWorkspaceLabelsKeepToTheirWorkspace` 是 998197501、桶 1，都在第 0 路（桶 0–45）；project 模块自己的单测也只在第 0 路跑（`go-test-shard.sh:43-46`）。接口故事的名字不合 m7-pages 的 `testMatch`，标题不合 `grep`，默认配置收它。要补的见 m2（`failures_test.go` 与两处先例的差别）与 o1。

### 3.3 引擎的改动

- **范围臂**：`allowsRole`（`issue-list-read.ts:52-56`）的新条件对；`readPage` 的接口选择与行的范围核对对（`:82-107`）；`covers` 同成员臂对（`issue-list-binding.ts:62-69`）；身份 `member` 为假对——“member” 这个水位是调用者的工作区角色在管理员与成员之间的变化（`permission-reads.ts:35-39`），工作区列表不随它变，项目一级的变化经 `workspaceProjects` 兜住（`:105`、`:113`）。漏的是 `ListContext`、`issueMembership` 与读 `group_by` 的六处（I1）。
- **穷尽**：两个 `switch` 的 `default` 都抛（`use-issues-actions.tsx:76-96`）；`usePageIssueFilters` 的 `switch` 没有 `default`、开在自己的联合上（`use-page-issue-filters.ts:9-15`、`:28-41`），加一臂即可；空态的 `switch` 有 `default: null`，是静默的一处（m8）。
- **`GlobalIssues` 不继承 `PagePreferencesIssues`**：对（它的 `fetchPage` 只造项目的范围，`page-preferences-issues.store.ts:29-43`）。
- **两种偏好服务与页面 id**：`PagePreferencesService.get(id)` 确实只收 id；`JSON.stringify([slug, key])` 与视图的 UUID 不会相撞；浏览器存储不复用 `handleIssuesLocalFilters` 的理由成立（它按控件词过滤，`issue-filter-helper.store.ts:206-305`）。`storage.set`、`storage.get` 自己不接异常（`C/lib/local-storage.ts:10-27`），spec 写的“存储不可用用默认、写失败吞掉”要由新服务自己接，写法已在 spec。
- **11 个词**：与 `ViewFilters`、生成的查询类型一致；`"me"` 的选项在筛选配置里已有。`toFilterExpression` 收 `project_id`、个人页的绕行变一行，类型上成立（`WorkspacePageFilters`、`MemberIssueFilters` 都可赋给 `ListPageFilters`）。
- **默认视图自己的词**：放最后、词表去掉自己的键，避开重复键的 422（`ISS:397-401`）。语义的一半在 I1（自己的词要进成员判断）。
- **`usePageDisplay` 一律取表格的选项**：必要（它按持有的布局取，`use-page-display.ts:47`、`:55`）。Display 的属性另见 m1。
- **`DIALOG_STORES`、拖放、`isWorkspaceLevel` 不接**：三处的理由都核过，成立（`isWorkspaceLevel` 的读者是列表、看板的分组列与子工作项的列表，后者收的是自己的 `storeType`，默认 `PROJECT`）。
- **表格根的接缝**：`canEditPropertiesBasedOnProject` 已在（`base-spreadsheet-root.tsx:37`、`:80-90`），写显示时传路由的项目 id（`:97-104`），工作区路由上是空的，`updateFilters` 不看给的 id 是对的。

### 3.4 数据层按容器

`getViewById`（`view.store.ts:72-81`）与 `rowsOf`（`:104-109`）的改法对；`view-changes.ts:78`、`:95` 的 `?? undefined` 今天确实让工作区视图的改删不进任何列表。没读过的列表上 `prepended`、`confirm` 是空操作（`reconciled.ts`），所以从视图页“Add view”“Save as”建出的视图不会造出一个半截的列表。读的位置（列表页与切换器打开时、外壳不读）与 R20 一致。迟到的失败见 o2。n1 这一类见 I5。

### 3.5 另存与更新视图（O1）

选法站得住：`_notifyExpressionChange` 同步通知（`filter.ts:538-540`），页面的回调把筛选写进草稿，草稿的服务在内存里、一个微任务落定，点击时草稿的筛选就是点击时的表达式；`updateView` 等 `onViewUpdate` 落定之后才推进基线（`:499-506`）。`t10-mut-filters-at-click` 杀得死：格在 HOC 加假队列的那一层扣住前一笔、点击、经 store 改草稿的筛选、放开，变异发出的是点击时的筛选。这是确定的，不算竞态；界面上因为 R29 的停用看不到（同 re-96 的 O1）。P6a 的两格（`update-view.test.tsx:80-121`、`:147-162`）在换来源之后断言仍成立（没有运行）。要补的两句边界见 m3。

### 3.6 标签目录（R32）

照状态 store 的形状（`state.store.ts:63-64`、`:93-99`、`:114-121`、`:153-155`、`:220-224`）成立，比它多一个范围判断是对的。写入两处都写：没读过的键上 `confirm` 是空操作，目录读在途时的写由 `changesDuringFetch` 重放，不必另写。项目页不受回落影响：项目的外壳每次进项目都为成员读项目自己的标签（`use-project-fetch.ts:65`，`use-project-labels-read.ts:13-16`），下拉框打开时总是读。受回落影响的是工作区一级的页面，口子见 m6（不是成员的项目、快照之后加入的项目）。I19 在 T5 之后照旧过（两处正则只认项目的标签读，标签下拉打开时总读一次），T14 的改写不是红的提交；T5 真正会红的是三份单测（I3）。

### 3.7 删除与关键词

导入表（`git grep`，基线）：

| 文件 | 导入者 | 最后的读者在 |
|---|---|---|
| `WC/quick-action.tsx` | `WV/header.tsx:21`（另有 `WV/layout.test.tsx:29` 的替身） | **T9** |
| `WC/default-view-quick-action.tsx` | `WV/header.tsx:19`（另有 `WV/layout.test.tsx:30` 的替身） | **T9** |
| `WC/modal.tsx` | `WV/header.tsx:20`、`filters-hoc/workspace-level.tsx:17`、`WC/quick-action.tsx:21`、`WC/view-list-item.tsx:18` | T11 |
| `WC/delete-view-modal.tsx` | `WC/quick-action.tsx:20`、`WC/view-list-item.tsx:17` | T11 |
| `WC/form.tsx` | `WC/modal.tsx:16` | T11 |
| `WC/view-list-item.tsx` | `WC/views-list.tsx:15` | T11 |
| `WC/default-view-list-item.tsx`、`WC/views-list.tsx` | `WV/page.tsx:17-18` | T11 |
| `TStaticViewTypes` | `issue-filter-helper.store.ts:18`、`:136`（T8 删），`PKG/constants/src/workspace.ts:8`、`:36`（T8 换），`WC/default-view-quick-action.tsx:14`、`:22` | **T9** |
| `my_issues` | `WC/form.tsx:137` | T8 改名（m4） |

前两行与 `TStaticViewTypes` 是 I4。其余与 spec §2.13 的表一致：`getFilterConditionBasedOnViews` 没有调用者；`globalView` 的读者是旧页头、旧布局、HOC 与 `WC/` 的四个组件，测试两份（`root.store.test.ts:14`、`:113`，`store-context.test.ts:11`、`:315`、`:318`）；`store-context.test.ts:313-318` 的例子换成 `x.issue` 成立（`IssueRootStore.rootStore`，`C/store/issue/root.store.ts:88`）；`root.store.test.ts:106` 是 20。关键词：`K:2` 的 phase 是 `M7/P6`；`old-view-urls` 在 `K:3257`；`plane-workspace-urls`（`K:2053`）里 `workspace.service.ts` 只是文件的 hit 样例（`:2124`），删文件不动它；例外没有匹配就报错、过期也报错（`tools/keywords.mjs:182-218`）。S2 的移除冒烟到 T8 都是绿的、T9 重写视图页之后必红，随 T9 删是对的。窗口里的三处接缝见 m5。

### 3.8 plan 的机制

- **允许清单**：T2、T4、T6–T11、T14 的逐条都对到了文件与行；T5 一条错（I3）；T2 少一份（m2）。
- **行数**：约 85 个文件逐个量过，与 plan 写的相同。上限 `root.store.ts` 350、`view-changes.ts` 220、`label.store.ts` 260、`issue-filters.ts` 400、`use-issue-list-mount.ts` 85 都够用；没写上限而要动的有 `paged-groups.ts`（445，I1）。
- **第一个读者**：T2、T4、T6、T7、T8、T10 的清单成立；T9 少 `WC/default-view-menu.tsx`、多留两个没有读者的旧文件（I4）；T11 的 Files 少 `(list)/layout.tsx`（m5）。
- **RED 在前、变异**：性质表的行都能编译、都对到一格；要加的行在 I1、I2、I5、m1、m2、m6、m8。
- **检查点**：0 在 T7 之后、1 在 T10 之后、2 在 T11 之后，plan 的头注、Global Constraints 与三个 Task 的最后一步一致，简报同。
- **没有红的提交**：三处守不住——T5（I3）、T9 的 knip 与关键词（I4、m5 ①）。
- **oxlint**：上限 14（`W/package.json:10`），是等式；T9、T11 删旧文件时照 Global Constraints 留计数的收据即可（I4 之后 T9 也要）。
- **`final.sh`**：六个配置与 plan 写的相同（`:28`），只差头注释（T1 改）。新故事的四个文件名合 m7-pages 的 `testMatch`（`E/playwright.m7-pages.config.ts:10-11`），接口故事不合。
- **e2e 夹具**：`createWorkspaceView` `:34`、`viewDefaults` `:52`、`viewRow` `:158`、`savedQuery` `:184`，`mounts.ts` 的头注释 `:5-16`，都对。

### 3.9 行号

核了约 200 处：合同、服务端、交接、设计与 P6a 记录被引的全部，网页一侧 Task 依赖的全部。偏差见 m9 的七处（⑦ 即 I3 里的 `page.test.tsx`）。简报与 `final.sh` 没有另外的偏差。

### 3.10 文档之间的矛盾

- spec §2.8（T9 的页头就有默认视图的菜单）与 plan 的 knip 清单（菜单是 T11 的）——I4。
- spec §5（显示不给迭代、模块）与 §2.4（词表只改筛选与布局）——m1。
- spec §2.13（`WV/(list)/*` 整个进规则）与 plan 的窗口（`(list)/page.tsx` 原样移）——m5 ①。
- spec §6“第 1 路不加”与 §2.1“矩阵一行是 `TestPermissionMatrix` 的子测试”——o1。
- spec、plan、裁定、设计与交接之间没有发现别的矛盾。调研与 spec 不同的十二处已在 spec §3 的补记里逐条定了，核过的几条（表格根的接缝、`common/applied-filters` 有读者、`DIALOG_STORES` 的读者、状态 store 没有工作区级的 getter、I19 的正则）都是 spec 对。

### 3.11 P6a 晚发现的三类

| 类 | P6b 里的对应 | 预先堵住了吗 |
|---|---|---|
| m1：迟到的 `NoViewDraftError` 经 swr 的失败一支把活着的页面刷白 | 已存工作区视图复用泛化的 `useViewPageRead`（每次挂载读自己的、取数函数接住），`t9-mut-no-draft-thrown`、`t9-mut-mount-read-shared` | 是。同一类的另一处没堵：默认视图的偏好读（I2） |
| O3 / n1：在视图自己的页上删除，闪“不存在”；第一次修又把别的挂载刷白 | `pageLeaving` 照旧、`t9-mut-leaving-any` | 机制在；但弹窗判“这一页的视图”读的是 `viewId`，工作区路由上永远对不上（I5） |
| O1：“更新视图”的筛选是点击那一刻的 | 整个体在轮到时从草稿算，`t10-mut-filters-at-click` | 是（m3 补两句边界） |

## 4. 核实为正确的事实（摘要）

- 枚举今天没有 `GLOBAL`（`PKG/types/src/issues/issue.ts:14-24`）；`globalViewId` 的路由判为 `DEFAULT`（`use-issue-layout-store.ts:28-30`）；`ISSUE_STORE_TO_FILTERS_MAP` 在 `PKG/constants/src/issue/filter.ts:240-246`，`TCreateModalStoreTypes` 在 `common.ts:55-60`；`my_issues` 的 8 个词与两种布局在 `filter.ts:187-218`。
- 列表挂载：键的 `page` 一段、角色、`allowed`、项目集与身份在 `use-issue-list-mount.ts:25-52`，去重窗口为 0（`:67`）。
- `PagePreferencesFilter`：类型界五处（`:23`、`:33`、`:41`、`:123`、`:292`），`confirmed` `:155-157`，`show` 不改写布局（`:160-175`），读序号 `:181-206`，整字段替换 `:212-233`。
- 草稿：`project-view/filter.store.ts`（36 行）的 `discard` `:27-28`，`view-draft.service.ts`（79 行）的 `discard` `:75`。
- 视图页的读：`use-view-page-read.ts` 的键与选项 `:40-44`、每次挂载的 `mutate` `:48-58`、`noDraftIsNoFailure` `:74-76`、`useShownView` `:85-102`。
- 更新视图：`use-project-view-saving.ts` 的 `onViewUpdate` `:118-154`（体 `:121-128`）、`hasAdditionalChanges` `:101-104`；`fake-view-saving.tsx`（113 行）以真的 store、SWR 与 HTTP 替身挂视图页。
- 旧层：`WV/layout.tsx:22-26` 的裸 `useSWR`、`:37-48` 的英文；`WV/header.tsx:72-76`、`:100-102`；`WV/page.tsx:25`、`:28`；`WC/view-list-item.tsx:52-73`；`WC/views-list.tsx:27-30`；工作区 HOC 的 Plane 分支 `workspace-level.tsx:39-160`、选项 `:48-58`。
- 标签：`label.store.ts` 的 `labelMap` `:99-105`、`getProjectLabels` `:116-120`；`group-columns.tsx:266-284`；个人页的门 `profile-issues.tsx:46-47`；草稿页的门 `workspace-draft/root.tsx:23`、`:27`；`WORKSPACE_STATES` 的键 `use-workspace-fetch.ts:49`。
- 会话与取数：`SESSION_FETCH` 不随焦点再验证、不重试（`use-session-swr.ts:26`），选项只有三个（`:38`）；`useCopyLink` 在 `use-copy-link.ts:42`；`getWorkspaceRoleByWorkspaceSlug` 在 `permissions.store.ts:56-59`。
- 门禁：swr 名单 `.oxlintrc.json:121-219`（末项是 `VC/use-view-page-read.ts`）；knip 不带排除（`Makefile:128`）；`gates.sh` 的七项含 `gen-check`、Go 两路与 knip。
- e2e：S2 的 `profileVisits` `:129-137`、注释 `:128`、`draftsVisit` `:148-151`、访客的行 `:208-219`；I19 的三处正则 `:336`、`:485`、`:491`；`E/stories/collaboration/` 里 P6a 的故事名同 plan 写的（`h12-project-view-page`、`h12-project-view-saving-page`、`h20-view-pages`）。
- 文案：spec §2.14 列的已有的键都在；`workspace_views.action.*`、`workspace_views.not_found.*` 今天没有（R29 的六个是新的）；`workspace_views.add_view`、`workspace_views.delete_view.{title,content}` 在 `workspace.json:135-141`。
- 交接：`HO/M3-closeout.md` 被引的 15 行、`HO/M4-closeout.md:51-58`、`:103`、`HO/M6-closeout.md:36-41`、`:50`、`:76`、`HO/M2-closeout.md:45`、`:56`、`:78` 说的都是 spec §5 所称的内容；`P6aR:179-187` 的九条在 spec §5 末行逐条有去处。
- T1：`P6aR:3`、`:51`、`:247`、P6a spec `:335`、`D:309`、`D:353`、`frontend-changes.md:206` 都是 plan 所称的那几句。

## 5. 需要负责人的事

无新的。五条重要发现都是网页一侧的写法与计划，属主控的局部裁定；没有一条要改合同、服务端的范围或 CI 的预算与结构。D-16（加 `listWorkspaceLabels`）与 D-8（不改合同的默认布局）照旧，核下来与合同、服务端的先例都相合。照已有的约定，R1、R2、R5、R7、R9、R10、R14、R15、R17、R18、R20 十一条高风险的局部裁定在阶段记录与 M7 收官报告里向负责人列出；I1 的修正落在 R1 的范围里（引擎的成员判断多一支），m1 落在 R5 的范围里，写进各自的那一条。

## 6. 处理（主控）

2026-11-27，主控全部采纳，交起草者照本文的修法改 spec、plan 与简报，改完由主控核对：

- I1–I5：照修法改。I2 取“不经 SWR”：默认视图的偏好在浏览器存储里，挂载时的副作用同步读、卸载时 `discard`，不进 SWR 的去重窗；T9 加“卸载之后马上再挂”的一格与 `t9-mut-default-read-shared`，H13 (page) 加一步。I4：T9 建 `default-view-menu.tsx`，同一提交删两个旧组件、`copy-hook` 的两个例外与 `TStaticViewTypes` 的几行。I5：删除弹窗按容器取页面的视图 id，T7、T9 的格挂真的路由形状，加 `t7-mut-delete-param-viewid`。
- m1–m9：照修法改。m3 的两句边界：`confirmed` 为空时不发、照失败处理（若起草者核实不可能为空，写明理由并钉一格）；离开之后轮到的体是离开前的草稿。
- 不需要负责人。
