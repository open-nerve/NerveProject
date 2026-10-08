# M3/P8b 项目一侧的数据层：评审记录

| 项 | 内容 |
|---|---|
| Phase | M3/P8b `web-project-data` |
| 日期 | 2026-10-09 |
| 结论 | **通过**。<br>11 个 Task 逐个实现、逐个评审，都通过。其中 8 个（Task 1、3–8、10）有当轮的修正轮：Task 1、3、4、5、6、8、10 的 Important 都是 plan 自己的测试缺口或执行中发现的缺陷，按裁定当轮补上或进修复轮。<br>整分支评审的结论是"修复轮之后可以合并"：Critical 0、Important 3、Minor 9。<br>修复轮一次分派、九个提交，落实了裁定 F-1 交给它的全部发现和搁置项；其余按 F-2、F-3 交给 P10、M4（第 6 节）。修复的复审 Critical 0、Important 0、Minor 5，同一个实现者用一个提交补上（`682bbff6`，裁定 F-6）；它的范围复审见第 4 节末。 |
| spec / plan | [spec](../specs/P8b-web-project-data.md) / [plan](../plans/P8b-web-project-data.md) |
| 设计 | [M3 设计](../M3-design.md) 第 12 节 P8b；P8 拆成 P8a、P8b 见 `7cf3a286`（负责人 2026-10-07 确认） |
| 分支 | `worktree-m3-p8b`，从 `main` 的 `7390e012`（P8a 合并）分出 |

## 1. 评审方式

- **拆分**：设计里的 P8 `web-data-layer` 由控制者照第 16 节拆成 P8a（工作区一侧）和 P8b（项目一侧），裁定 R1–R4（`$M3TMP/p8-split-rulings.md`），设计提交 `7cf3a286`，负责人 2026-10-07 确认。P8b 在 P8a 合并（`7390e012`，2026-10-08）之后开始。
- **spec 和 plan 由架构子任务产出**（`763ad47c`）：设计的草稿是 9 个任务，plan 是 11 个 Task、12,491 行，附带原型和逐 Task 复现（最终与原型逐文件相同，3,113 个文件）。132 个变异在最终原型上都被发现；W2 的挂载清单是真实运行的记录。spec 第 3 节标"裁定"的十条由控制者裁定 D1–D10（`$M3TMP/p8b-architect-rulings.md`），其中：
  - D1：11 个 Task；
  - D3：项目封面的上传、28 张预设和随机封面在 Task 1 删除（生成的 `ProjectCreate`、`ProjectUpdate` 没有封面，P8a 的 A2 同理）；设计 7.8、12 节 P10 的任务 3 和 M5 的交接随之改；
  - D5：S2 断言每个账户每一页确切的挂载清单，不从关键词规则推出旧地址的模式；
  - D8：项目 store 的测试 422 行不接受原样：记录和加载移进共用的 `fake-projects.ts`，按关注点分成两个文件；
  - D9：S2 的改写并进 Task 10，与它守着的项目包装层在同一个提交。
- **执行前的预检**（opus，只读，`$M3TMP/p8b-preflight/preflight.md`）：高 0、中 8、低 6。复现干净，架构子任务的变异都在它说的地方被发现。
  - PF-M1：Task 3–10 之间，一个旧地址被写回挂载路径，所有门禁都放过（spec 第 3 节第 7 条不成立）。改为每个 Task 把自己删掉的地址加进关键词规则（P8a 的做法）。
  - PF-M2：请求体在调用时从 nerve 回答之前的 store 算出：标签栏丢掉前一个修改，侧边栏和状态的拖动按旧的两邻放。改为一条规则：轮到它发出时，从 nerve 最近一次回答算。
  - PF-M3：调用者另一个工作区的项目在地址的工作区里打开，页面混着两个工作区。改为项目只在地址的工作区里算数。
  - PF-M4–M8：`useProjectFetch` 的两项判断、项目 store 的四项性质、标签栏 store 在接线处的门、S2 对包装层渲染的检查（只靠时区的请求看出）、子 store 的测试只有一个工作区：都没有检查，补上测试。
  - PF-L1–L6：工作项 store 留着旧的复制；两个 getter 给出一个状态的两个版本；别的写者盖过包装层的读（L3）；次要集合没有自己的交错测试；没有检查发现两种语言都留下的孤儿文案键（L5）；S2 去重、挂起时打印 `null`。
- **修订**（`42775c10`）：落实 D1–D10 和预检的发现（控制者的裁定见 `$M3TMP/p8b-amend-brief.md`）。从 `7390e012` 的新副本重新复现：每个 Task 之后门禁通过，最终与原型逐文件相同（3,115 个文件）；158 个变异都在最终的树上、也在各自 Task 的树上被发现。plan 13,308 行。
- **执行**：
  - 开始时的裁定 E1–E5 见第 3 节。
  - 11 个 Task 各由一个实现子任务完成（不指定模型），再各由一个评审子任务（opus）检查是否符合 spec、质量是否达标。Task 的修正轮用 sonnet 做范围复审。同一时刻只有一个写入者。
  - 实现者在每个 Task 里照 plan 的变异表再跑变异，并加上自己的探查变异。每个 Task 都跑过 `make lint-web`、`knip`、`test-web`。`make e2e` 在本地跑到 Task 8 的实现（`c2f4c102`），每次 70 个通过。从 Task 8 的修正轮（`e7934b13`）起，本机的 Docker Desktop 起不来（"Docker Desktop is unable to start"，控制者核实）；没有人重启它，也没有碰任何容器。此后 e2e 只由分支的持续集成跑（第 2 节）。
  - **执行中发现并修好的生产缺陷**（都在分支上，从不在 `main`）：
    - T3：两个工作项 store 的 `fetchParentStats` 等着重取项目；项目的读被拒绝时，`createIssue` 在 nerve 已建好工作项之后失败（M4 让工作项出现之前看不到）；
    - T5：store 还没有标签栏时，页头的修改作用于手写的默认值，nerve 整份替换，调用者已藏起的标签又回来；
    - T7：工作项详情的标签选择器中"添加标签"一行原是 `Combobox.Option value={query}`。Headless UI 2.2.10 在按下鼠标时就选中，于是在创建标签之前先把输入的文字当作标签 id 写进工作项。这是 Plane 原有的错误，Task 7 的块把这一行改成按钮时去掉，评审查明；
    - T8：修改状态从来不会成功：表单把整个 `State` 放进封闭的 `StateUpdate`，nerve 答 400，页面提示成名称冲突。状态页读 Plane 形状的错误：创建的 `catch` 中抛 `TypeError`、什么都不提示；修改、删除按 400 选文案，而 nerve 答 409；
    - T11 的评审发现、修复轮修好：成员默认值的负责人和默认指派人一起发，另一个取自页面显示的值，连续两次修改时第二次盖掉第一次的回答；功能开关在调用时把显示的值取反（修复轮扩大到五处开关）；
    - 整分支评审：I2，离开的公开项目在这一代再也不显示；I3，nerve 的类型化请求体拒绝的字符串：两个角色下拉框（加成员的弹窗、成员页的改角色）发出字符串的角色，自动归档的月数是字符串。
  - **代码本来对、只有评审守着的性质**（plan 的测试钉不住，当轮补上测试）：T3，项目页删掉已归档列表的取数时 `/projects` 一直转圈，而门禁都通过；T6，成员的修改在轮到它时才找成员关系；T10，nerve 先前给过的项目再取时答 `project.not_found`。
  - **执行中删除的孤儿**：T1 的五条文案键和 `@nerve/utils` 的 `checkURLValidity`（knip 看不穿包的入口）；T3 的 `TFetchStatus`；T4 的 `fetchProjectsWithCreatePermissions`；T5 中迭代、模块 store 从不读的 `projectService` 字段；T7 的 `state_id` 标签页序号和 `create-root.tsx` 中总是 `""` 的两行；T8 的 `stateDetails`、`workspaceStateDetails`；T9 的三条文案键；修复轮的 `workspaceStates`（整分支评审 I1）。
  - 其余裁定都是 plan 自己的测试缺口、说错的注释或文字（第 3 节）。跨 Task 的去重和收尾搁置到修复轮（P1–P17）。
- **整分支评审**（opus，`42775c10..4e71bb73`）：结论"修复轮之后可以合并"，Critical 0、Important 3、Minor 9；另外指出 6 条搁置项说得不对或太窄，并照问裁定 P1、P13、P16（第 4 节）。
- **修复**：一次分派，九个提交（`c3aff07d` 到 `2f6986da`）。修复的复审（opus）通过：Critical 0、Important 0、Minor 5。同一个实现者用一个提交（`682bbff6`）补上（裁定 F-6），范围复审（sonnet）见第 4 节末。
- **持续集成**：见第 2 节末。

## 2. 验收标准核对（spec 第 4 节）

| # | 标准 | 结果 |
|---|---|---|
| 1 | S1、改写的 S2 和此前的全部故事通过：worktree 中 `make e2e` 75 个（S2 7 个，Task 10 起） | **通过（由持续集成）。** Task 1–8 的实现和 Task 1–7 的修正轮在本地每次都是 70 个通过。从 Task 8 的修正轮起本地不能跑（Docker Desktop，第 1 节），e2e 由分支的持续集成跑：改写的 S2 第一次运行在 Task 10 的 `0d505ec9`（run 37800253033），实现者从代码推出的清单在这次运行中成立；之后 Task 10 的修正轮、Task 11 和修复轮每次都通过。整分支评审和修复的复审都没有跑 e2e，依赖这些运行。 |
| 2 | M2 交接第 3 节的 `git grep` 中 M3 的 10 处都已消失 | **通过。** 27 → 22 处：M3 的 2 处（`project/form.tsx` 的 `ProjectService`，Task 5；`ProjectMemberService`，Task 4、6）消失，P8a 的 8 处已在之前消失；另 3 处是封面上传的 `FileService`，随 Task 1 删除。剩下的 22 处都是 M4–M7 的（本记录在 HEAD 上重跑）。 |
| 3 | `node tools/keywords.mjs` 通过：64 条规则、3 个例外，没有命中；Task 3–9 每个 Task 删掉的地址从那个 Task 起由它发现 | **通过。** 64 条规则、3 个例外（M3 加的只有 `plane-workspace-urls` 的 `until: "M4"` 一条），没有命中（`682bbff6` 的门禁）。每个 Task 一个 `tN-…-back` 变异，修订时在各自的树上由守卫发现。修复轮给 `plane-user-urls` 加 `/api/users/me/` 之外的命中样例（P15）：`fw-p15-users-me-only` 在加之前存活，之后被发现。 |
| 4 | 9.5 中 P8b 的 vitest 通过，每个都有一个发现它的变异 | **通过。** web 的 vitest 56 个文件、504 个测试（`682bbff6`；`7390e012` 上 33 个、294 个；Task 11 结束时 47 个、469 个；修复轮之后 54 个、497 个）。变异：<br>• plan 的 158 个在修订时都被发现（最终的树和各自 Task 的树）；<br>• 整分支评审在 `4e71bb73` 上重跑：plan 的 155 个被发现（`t10-cycles-back`、`t10-unread-back` 只由 S2 发现，`t5-pp-change-at-call` 从 Task 5 的修正轮起由 `t5r-pp-change-at-call` 代替），各 Task 修正轮和探查的 108 个中 105 个；<br>• 修复轮在 `6bbcceed` 上把各表一起重跑（P3）：plan 147/158，9 个要改的文字已不在、各有代替，2 个存活（`t10-unread-back` 只由 S2 发现；`t10-pw-children-non-member` 的表只写 `tsc` 和 S2，单独跑时由 `project-wrapper.test.tsx` 发现）；探查 104/108（2 个文字不在，2 个是记下的存活：等价的 `t6r-pm-add-ids-request`、只由 S2 发现的 `t10-page-no-archived-call`）；修复轮自己的 41/42（`fw-rw-t10-cycles-back` 只由 S2 发现）；<br>• F-6 的后续在 `682bbff6` 上重跑修复轮的表：50 个中 49 个（加 8 个 `fw-f6-*`）；<br>• 修复的复审在副本上亲手跑了 I2、I3（三处）、P13、P14、M3、M9、P16 的变异，都被发现。 |
| 5 | 根目录 `.oxlintrc.json` 的两条 `overrides` 覆盖 P8b 的文件，两个包装层都在范围内 | **通过。** `no-restricted-imports` 的范围与总体设计 7.7 的清单恰好相同（23 个条目，Task 11 的评审核对）。W4、W12 的 7 个变异都让 `check:lint` 失败（spec 附录 A.2）；修复轮改写的 `fw-rw-t10-raw-swr-wrapper` 由 oxlint 发现。 |
| 6 | `tsc`、knip 通过 | **通过**（`682bbff6`）。 |
| 7 | 改到的文件按 7.9 没有 oxlint 警告；上限 435 → 360 | **通过。** 有手改的源文件 148 个都是 0 条；修复轮手改的也都是 0 条，其中 7 个是新加入的（spec 第 4 节）。修复的复审核对修复轮改的 53 个 web 文件，F-6 之后的门禁核对 57 个，都是 0 条。只经机械步骤到达的 36 个文件中 5 个共 14 条（spec 附录 A.8），留给 P11 第 4 个任务。web 的上限 435 → 360，`utils` 12 → 7，其余各包不变；没有新的抑制，删去四处。 |
| 8 | 逐 Task 复现 | **通过（修订时）。** 修订 `42775c10` 时，架构子任务从 `7390e012` 的新副本照 plan 重放：每个 Task 之后门禁通过（e2e：Task 1–9 69 个加 S3 的 F4，Task 10、11 74 个加 S3），最终与原型逐文件相同（3,115 个文件）。执行中各 Task 的修正轮照裁定偏离原型，plan 中受影响的后续块随之改（第 5 节）。 |

**持续集成**：分支上每次推送都跑 server、web、e2e 三个任务，每次都通过（两次在途的运行被同一个 SHA 的第二次推送取消，它们的重跑通过，见下）：
- Task 1（`7f7da15e`）：run 37751393037；
- Tasks 2–3（`52918ab8`）：run 37759314345；
- Task 3 的修正轮到 Task 4（`2604422c`）：run 37767714818；
- Task 5（`05802fe2`）：run 37771617181；
- Task 6（`7e85f8e3`）：run 37776156382；
- Task 7（`ba8cb1f4`）：run 37777252944；
- Task 7 的修正轮（`e875364b`）：run 37779334464；
- Task 8（`c2f4c102`）：run 37784028981；
- Task 8 的修正轮（`e7934b13`）：run 37791002275。同一个 SHA 的第一次运行 37791000290 被第二次推送取消（并发组取消在途的运行）；
- Task 9（`2db3bdcf`）：run 37796293191。第一次运行 37795525637 同样在 e2e 中被取消；
- Task 10（`0d505ec9`）：run 37800253033，S2 的第一次运行；
- Task 10 的修正轮（`fe1b45d5`）：run 37803319826，S2 第七个测试新加的检查通过；
- Task 11（`4e71bb73`）：run 37805028675；
- 修复轮（`2f6986da`）：run 37820800889，S2 在 P13 改为读路由的参数之后通过；
- F-6 的后续（`682bbff6`）：run 37825535784，三个任务都通过。

本提交推送之后，分支上再跑一次；合并之后 main 的持续集成在合并时记录。

## 3. 执行中的决定

这一节列控制者在本阶段做的裁定，每条写明理由和裁定错了的代价。完整的记录在执行台账中。

**执行前**
- 拆分（负责人确认）、R1–R4、D1–D10，以及预检的 PF-M1–M8、PF-L1–L6 和修订 `42775c10`（第 1 节）。spec 第 3 节由这些裁定定下，执行中没有再议。
- 开始执行时：
  - **照 plan 执行**：预检的跨 Task 表已逐对核对，它的发现都在修订中落实，并由复现再证明。代价：错了是一轮 Task 级的修正。
  - **E1**：接受 Task 3 的 1,875 行和 Task 8 的 1,726 行。理由：多出的是预检要的测试和共用的替身；拆 Task 3 要在中间的树上留一个一半是 Plane 写法的项目 store，拆 Task 8 要一个过渡的状态页；P8a 的 1,5xx 行的 Task 一轮修正就过了。代价：两份长的 brief 和评审。
  - **E2**：PF-L1 落在 Task 8、9（状态、标签 store 重写的地方），各有一个 `root.store.test.ts` 的测试和变异。代价：没有。
  - **E3**：接受 brief 列表之外的关键词（Task 7 的工作区 `labels/` 和 `intake-state/`，Task 5 的 `user-favorite-projects/`）：用它们的方法都在那个 Task 删除。代价：没有。
  - **E4**：放在第一个之前的标签取它减 10000（共用的 `placeBetween`，服务端的步长）。理由：`labels.sort_order` 是双精度、没有 CHECK，契约没有下限，负数合法。代价：只是发出的位置值变了。
  - **E5**：P8a 的 `use-navigation-preferences` 的 `preferencesChangeOf` 在调用时读显示的 `limitedProjectsCount`（PF-M2 的形状），交给 P9 与 P28 一起改；本文第 6 节和 spec 第 5 节写它。另外 Task 11 在总体设计 7.7 的排队一条之后写上 PF-M2 的总规则：它和排队一样约束之后的 M。代价：plan 的文字之外多一句，记作 Task 11 的偏离。
  - 带到本评审：PF-L5（没有检查发现两种语言都留下的孤儿文案键）给收尾；PF-L3 给 M4、M7（spec 第 5 节）。

**执行中**
- **T1-a … T1-c（生成的 `Project` 没有的）**：
  - 实现者报告三条文案键没有了读者（另两条在 BASE 上已是孤儿）。在修正轮中删除，改 spec A.7（T1-a）。理由：删除就删干净，之后没有块再碰这两个文件。
  - 评审：Spec ❌，Important 3 条、Minor 3 条，当轮一并修（T1-b）：
    - I1、I2：五条孤儿文案键和 `checkURLValidity`（唯一的读者已删，knip 看不穿包的入口，之后没有 Task 删它）；
    - I3：封面的辅助函数留着死的分支（`getFileURL` 原样返回）和一句不真的注释；
    - m4–m6：创建弹窗的页头改用 `showDefaultWhenEmpty`；三处说错的文字；第 21 条加上清 oxlint 警告带来的两处可见的不同。
  - 页头用 `src={undefined}`（T1-c）。理由：`src={null}` 过不了 `tsc`（`CoverImage` 的 props 与 `<img>` 的相交），放宽它会让 Task 2 的 `t2-cover-null` 看不见。代价：没有。
- **Task 2（生成的类型取代 Plane 的四个项目类型）**：评审通过，C0/I0/M2。机械步骤重跑逐字节相同，31 个手改的文件 0 条警告。五个只经机械步骤到达的文件把 `Partial<Project>` 当作请求体或 getter 的类型，搁置为 P1，由整分支评审裁定。
- **T3-a … T3-d（项目的 service 和 store）**：
  - 实现者删除 `TFetchStatus`（唯一的读者在本 Task 删掉，knip 看不见），评审认可，记进 spec A.9。
  - 评审 C0/I3/M5。
  - **I1**：项目页的已归档取数没有任何检查：删掉它，`/projects` 和 `/archives` 一直转圈，门禁都通过。改为移进 `useArchivedProjectsFetch`，用 `fake-session-swr`、`fake-store-hooks` 测键、"工作区不是他的"时为 `null` 和 fetcher 的参数（T3-a）。理由：P8a 对每个挂载时取数的写法。代价：没有。
  - I2、m1–m5 当轮补上（T3-b）：已归档列表和恢复的"重复"交错；侧边栏的移动写进项目自己的读；拒绝从加载好的 store 开始；不真的标题和 spec 2.3；第 21 条的 `tabIndex`；A.9。理由：每一条都是只有评审守着的性质或不真的句子。
  - **I3**：`fetchParentStats` 在根上改（T3-c）：项目、已归档两个工作项 store 不再重取项目（`() => {}`，同工作区和个人页的 store）。理由：Task 1 删掉了工作项的修改会动的那两个字段；重取还与包装层的读争同一个项目自己的读（PF-L3）。代价：将来若有项目的字段随工作项变化，M4 再加一个不等的重取；spec 第 5 节 M4 一行去掉 PF-L3 中工作项 store 的一半。
  - 接受的过渡：`ProjectMemberStore.projectRoot` 写了、到 Task 6 重写之前不读；旧 `ProjectService` 的方法留到 Task 5。
  - 项目页去掉这个 hook 的调用，到 P10 之前没有门禁守着（knip 把测试当作入口）。交给 Task 10：S2 的管理员也打开 `/{slug}/projects`，清单含已归档的列表（T3-d）。代价：S2 多几行。
  - 范围复审：hook 测试的"地址没有工作区"一行不会失败，搁置为 P4。
- **T4-pre … T4-b（加入、离开；权限 store 的项目一半）**：
  - Task 4 进行中，一次自动的安全评审说 `getProjectRole` 是开放的：store 没有的项目给工作区管理员管理员、给其余的人访客，也不比较项目的工作区。控制者裁定不开放（T4-pre）：四条都成立才给角色；表中加两行和两个变异。理由：页面按权限 store 决定显示什么、取什么（7.1）。
  - 前提不成立（T4-pre-b）：评审读到的是变异 `t4-pf-unlisted-guest` 在 `mut.py` 运行时临时写进文件的样子，plan 的代码本来就不开放。两行留下，它们发现真的开放写法。第二次自动的发现（每个加入的项目都给成员）同样是正在验证的变异 `rv-create-any-role`；控制者核对提交的代码仍有 `role >= MEMBER`，不改。
  - 评审 C0/I3/M3，当轮修（T4-a）：
    - I1：每个加入的测试加入的都是 acme 的项目（第三条教训），加一个加入 beta 的项目的；
    - I2：拒绝的 `it.each` 从没读过项目自己的读开始（第一条教训），改为先读；
    - I3：`projectsWithCreatePermissions` 重写了却没有测试，加测试；
    - M1：`member-list-item.tsx` 在 store 没有这个项目时把自己的移出走成管理员的一支，改为认人；
    - M2：`| null` 和没有调用者的 `fetchProjectsWithCreatePermissions` 删除；
    - M3：测试自己又写了一份 `loadProjects`，改用共用的；plan 中 Task 5 的两个块随之改。
  - 两处提示读 `ApiError` 没有的 `err?.error`，nerve 的 `project.sole_admin` 从不显示：交给 P10，与 P8a 的 P21 同类（T4-b，P5）。代价：P10 之前是固定的文字。
- **T5-a、T5-b（标签栏、标识检查、旧 `ProjectService`）**：
  - 评审 C0/I2/M4。
  - **I1**：store 没有标签栏时，修改作用于手写的 `DEFAULT_NAVIGATION`，nerve 整份替换，藏起的标签丢掉。根上改（T5-a）：没有标签栏就不发、直接失败（状态、标签的 `held()` 规则），`DEFAULT_NAVIGATION` 删除。理由：plan 的"没有取过时是默认的"与 spec 第 15 条相抵，以 spec 为准。代价：那一个往返里控件提示失败；取到之前藏起控件是 P10 的页面的事。
  - 其余当轮补上（T5-b）：
    - I2：两次隐藏的测试中 nerve 的回答与请求相同，改为不同；排队的测试补上"被拒绝的修改不是下一个的起点"；
    - M1：三个修改抽成 `tab-navigation-utils.ts`，用 `it.each` 测；
    - M2：加载的标签栏不是默认的；
    - M3：迭代、模块 store 从不读的 `projectService` 字段删除，`cycle.store.ts` 的 3 条 `always-return` 清零，上限 408 → 405，plan 中之后各 Task 的上限随之改；
    - M4：第 21 条加"被拒绝的隐藏、显示提示失败"。
  - `form.tsx` 的 `!available` 一支交给 P10（P6）。
- **T6-a、T6-b（项目成员）**：
  - 实现者自己的 6 个探查变异在 plan 的测试上存活，加了五个测试（一个测试发现两个）。
  - 评审 C0/I1/M3。I1：没有测试钉住"成员关系在轮到它时才找"（`t6r-pm-membership-at-call` 存活）。M1：调用者自己的角色只由回答与请求相同的测试守着，改为请求 15、回答 5。当轮补上（T6-a）；加入的 id 取自请求的变异在 nerve 的契约下等价，记进 A.2。代价：没有。
  - M2 扩大 P5（P10）；M3 进 P2、P3（T6-b）。
  - 范围复审核实等价：`add_members.go` 一个事务、按请求的顺序回答。
- **T7-a … T7-e（M7 的部分先走；标签页面清零）**：
  - 收集箱创建表单的 `state_id` 标签页序号删除，它唯一的读者在本 Task 删掉（T7-a，Task 1 的 `cover_image` 先例）。代价：没有；其余各项的序号各减一。
  - 三个标签选择器的"添加标签"一行成为按钮，spec 2.7 和第 21 条照实写（T7-b）。理由：这是 plan 自己的块；以查询文字为值的 `Combobox.Option` 在多选中是"标签名当 id"的路径。plan 的"行为不变"一句进 P2。评审确认：Headless UI 在按下鼠标时就选中，原来的写法把输入的文字当作标签 id 写进工作项；搜索框为空时方向键下、回车写进 `""`。
  - 拖放的 effect 依赖 `isEditable`，接受（T7-c）。
  - 评审 C0/I0/M4，删除完整。m1–m4 当轮补上（T7-d）：工作区一级显示的工作项不再有标签（第 21 条、spec 第 5 节 M7 一行和 M4 的指向）；按钮修掉的错误照实写；A.10 不真的一句；`create-root.tsx` 总是 `""` 的两行 `state_id`。
  - 标签设置页的拖动把手只因 `onDrop` 重建才重新登记，交给 P10（T7-e，P7）。
- **T8-a … T8-i（状态）**：
  - 实现者的 18 个探查变异中 15 个在 plan 的测试上存活，另加 `state.store.changes.test.ts` 的五个测试。放在另一个文件里，接受（T8-a，Task 6 先例，约 400 行）。
  - 新 service 抛 `ApiError`，旧的抛 axios 的 `error.response`：`create.tsx` 的 `else` 支读 `data.error`，在 `catch` 中抛 `TypeError`、不提示；修改、删除按 400 选文案，nerve 答 409。这是 Task 8 引入的缺陷，不是 P5 那一类。修正轮里三个状态页改读 `errorMessageKey`（T8-b）。
  - spec 的句子照代码：只有空的组只发 `group`（T8-c）。
  - 评审 C0/I1/M4。**I1**：修改状态从来不会成功（第 1 节）。根上改：表单的值只是编辑的三个字段，`create-update.test.tsx` 钉住请求体（T8-d）。
  - T8-b 扩大到状态页的每一条出错路径：创建、修改、删除，设默认（原来在请求发出之前就结束加载、什么都不提示），被拒绝的拖动（原来只在控制台记一条）（T8-e）。代价：五个文件各几行。
  - M1（另一个项目中的移动）、M2（`t8r-st-chain-on-request` 不等价：被拒绝的移动什么都不改）、M4（第 21 条的措辞）照评审改（T8-f）。
  - M3：`issue/root.store.ts` 的 `stateDetails`、`workspaceStateDetails` 没有读者，删除；plan 中 Task 9 的两个块和散列行随之改（T8-g）。
  - 带给 Task 9：标签页的修改请求体和出错的提示做同样的检查；项目的页面已扫过，干净（T8-h）。
  - 修正轮中会话被切断，没有提交；照同样的指示恢复实现者。修正轮另外让 `state-item.tsx` 的拖放 effect 交回 `combine` 的撤销。
  - 范围复审：plan 中 Task 8 自己的块和散列行仍带着 `stateDetails`。控制者核实 Task 9 的块照样能应用于真实的树；Task 8 自己的文字是 P2 那一类的漂移（T8-i）。
- **T9-a … T9-d（标签）**：
  - 实现者的 13 个探查变异中 10 个在 plan 的测试上存活，另加 `label.store.changes.test.ts` 的三个测试，放在另一个文件里，接受（T9-a）。
  - T8-h 的结果（T9-b）：`LabelUpdate` 是封闭的，plan 的表单本来只发名称和颜色，没有请求体的缺陷。标签的值移进 `useForm` 的 `defaultValues`，测试由此钉住请求体；标签设置页和三个选择器的每条出错路径读 `errorMessageKey`；被拒绝的创建之后加载会停下；三条孤儿文案键删除。评审核对 `defaultValues` 的移动不改行为。
  - 两个选择器的取标签被拒绝时一直"加载中"或被吞掉，Plane 原有，交给 P10（T9-c，P9）。
  - 评审 C0/I0/M3，没有 Task 8 那一类的缺陷。M1（状态、标签的测试复制了记录和"nerve 之后的回答"）、M3（`|| undefined` 已不起作用）进修复轮（P10、P11）；M2（第三个选择器 `issues/select/dropdown.tsx`）进 P9（T9-d）。
- **T10-a … T10-c（项目包装层和 S2）**：
  - S2 的清单记下查询串（T10-a）。理由：已归档的列表和包装层的列表只差 `?archived=`，只记路径分不开"少了已归档的取数"和"未归档的取了两次"（T3-d 的变异）。代价：之后的 Phase 也要写查询串。
  - 评审 C0/I1/M3：
    - I1：没有一行是"nerve 先前给过项目、再取时答 `project.not_found`"；
    - M1：`project-wrapper.tsx` 从判断到渲染没有检查（`t10r-pw-unavailable-blank` 存活）；
    - M2：`use-project-fetch.ts` 读路由 store 的工作区，它比地址晚一次渲染；
    - M3：plan 的文字、spec 2.10 的一句，S2 第七个测试缺 CSP 的检查和最后的复查。
  - I1 的一行、M1 的服务端渲染测试 `project-wrapper.test.tsx`、M3 当轮补上（T10-b）。代价：一个测试文件，S2 由持续集成重新核对。
  - 工作区包装层同样的缺口进修复轮（P12）；M2 交整分支评审权衡（P13）（T10-c）。代价：一次渲染的窗口，不泄露数据。
- **T11-a、T11-b（关键词的收尾、死行、文档）**：
  - 评审通过。它另外发现：
    - I1：`project-settings-member-defaults.tsx` 从所持的项目取另一个字段，与 P8b 先等回答的 `updateProject` 一起，连续两次修改时第二次盖掉第一次的回答；功能开关在调用时发 `!held[feature]`；
    - m1：`plane-user-urls` 的命中样例都在 `/api/users/me/` 之下，收窄的变异能过；
    - m2：7.7 的"按工作区的 id 存"一条与按 slug 存的 `project_filter.store.ts` 不符；
    - 几处过时的句子。
  - I1 和功能开关是 P8b 先等回答的项目 store 带来的 W17 缺陷（Plane 原来是乐观的），属于页面上的 PF-M2。进修复轮（T11-a，P14）：成员默认值只发改的那一个字段；开关的新值由 store 的方法在轮到它时决定（`toggleDefaultTab` 先例），各有测试；整分支评审扫一遍从所持的值做请求体的每一页。代价：一个 store 方法、两个测试。
  - m1 进修复轮（P15）；m2 由整分支评审裁定（P16）；过时的句子给文档的收尾一步（P17）（T11-b）。

## 4. 整分支评审的发现与处理

结论"修复轮之后可以合并"：Critical 0、Important 3、Minor 9。评审（opus）的范围是 `42775c10..4e71bb73`（231 个文件）。它在 HEAD 的副本上跑过每一道门禁：关键词 64 条、turbo 54/54、knip、vitest 47 个文件 469 个。它没有跑 e2e（Docker），依赖 run 37805028675。它还核对了：
- 死成员：deadsym 新加的 21 行中，20 行是测试替身的类型成员，1 行是生产代码 `workspaceStates`（I1）；
- 文案键：在 `7390e012` 上有读者、在 HEAD 上没有的 10 条，都已在两种语言中删除；
- 三类缺陷逐个清扫：每个 P8b store 修改的调用处的请求体，每个 P8b 调用的 `catch`，页面上从所持的值做出的请求体；
- 五个项目一侧 store 的一致性（和 P8a 的）；换代的全过程；
- 每一页挂载时的请求，与 S2 的 `REQUESTS` 和 spec A.5 相同；
- 完成线：在代码中达到，带 I1–I3。

| 发现 | 处理 |
|---|---|
| **I1** `StateStore.workspaceStates` 只有测试读：唯一的生产读者在 T8-g 删掉 | F-1：删除接口、computed 和 getter，连同只为它的 `rootStore` 字段；测试经 `stateMap`、`getProjectStates` 读工作区的列表；`fw-i1-*` |
| **I2** 离开、删除之后项目自己的读是 `null`，被当作最终的回答：离开的公开项目在这一代再也不显示，项目页计数却没有卡片；spec 2.4、第 21 条不成立 | F-1：`null` 算作忘了（`if (read)`）；"删除、离开之后"的测试也核对之后的列表有它时显示为不是成员；`fw-i2-null-final` |
| **I3** nerve 的类型化请求体拒绝的字符串：加成员弹窗手选的角色（`add-project-members-modal.tsx:244`）、自定义的自动归档月数（`select-month-modal.tsx:72`）；成员页的改角色（`member-columns.tsx:172`）同类，spec 第 5 节原交给 P10 | F-1：三处都改，成员页也在内；一份 `PROJECT_ROLES: ProjectRole[]`，`archive_in` 是数字；服务端渲染的测试 |
| **M1** 项目的创建、设置表单读 Plane 形状的错误（`projects/create/root.tsx:60-62`、`project/form.tsx:84-92`）；`form.tsx:153` 的标识检查不在 `try` 之内 | F-2：交给 P10（spec 第 5 节、本文第 6 节） |
| **M2** 加入的界面中，被拒绝的加入没有 `catch` | F-1：提示 `errorMessageKey`；`project-wrapper.test.tsx` 一行 |
| **M3** `updateProjectSortOrder` 对轮到它时已不在调用者项目中的项目照样发出 | F-1：轮到时找它，没有就不发、直接失败；测试 |
| **M4** 项目自己的读盖过较新的列表 | F-3：设计问题，交给 P10，带评审的说明 |
| **M5** 侧边栏的拖放处理写了两遍 | F-1：一个 hook `use-project-drop.ts` |
| **M6** `TTabPreferences` 把 nerve 的标签栏适配成 Plane 的形状 | F-1：删除；五个读者读生成的 `ProjectNavigation` |
| **M7** 提示、文案的替身和 store 的构造在测试文件之间复制；P10 太窄 | F-1（连同 P10）：共用的 `fake-toast.ts`、`fake-i18n.ts`，状态、标签 store 测试的 `fake-project-lists.ts` |
| **M8** spec 第 5 节 P9 一行漏了 E5 | F-1：补上 |
| **M9** "放下处 → 两邻"写了三遍，spec 第 15 条说只写一份 | F-1：`placeAt` |
| 搁置项说得不对或太窄：P3（`t3-ps-sort-read` 的文字已不在）、P5（`leave-project-modal.tsx` 一条说错，加 M1 的两处）、P9（状态、成员的下拉框和工作项 store 不等的取数归 M4）、P10（加 store 的构造）、P14（加访客开关、自动归档开关，共五处）、P17（漏了 spec 的状态行、第 4 节、第 5 节 P9 一行，M3 设计 `:1943` 的任务数；7.7 的 PF-M2 一条要等 P14 落实才对页面成立） | F-1、F-2 |
| 照问裁定：P1、P13、P16 | P1：三个自动化文件的请求体是 `Pick<ProjectUpdate, "archive_in">`，`dropdowns/project/base.tsx` 的 getter 交给 P11，`fake-projects.ts` 留着；P13：`useProjectFetch` 读路由的参数；P16：项目的筛选按工作区的 id 存，不写例外 |

**控制者的裁定**（台账里每条都写了理由和代价）：
- **F-1**：修复轮做 I1、I2、I3（三处都做：Task 6 起成员页的每次改角色都是 400，是 W17 的缺陷，与 T8-d 同类，不推迟），照评审的 P1、P13、P16，P14 扩大到五处，M2、M3、M5、M6（负责人的标准不许适配层）、M7 连同 P10、M8、M9，P4、P11、P12、P15、P3，以及 P17 和评审补充的文档（各 Phase 的状态行除外，由本提交写）。
  - 理由：每一项都是 P8b 自己文件里的根上的修法、带测试的几行，或让文档成立；没有跨模块的。
  - 代价：修复轮很长，按步骤分提交。
- **F-2**：M1、更正后的 P5、P6、P7 和 P9（加第三个选择器）是 P10 的，写进本文第 6 节 P10 一行和 spec 第 5 节；状态、成员的下拉框和工作项 store 的取数进 spec 第 5 节的 M4 一行。理由：都不是 P8b 引入的可见不同（评审的第二类表）。代价：没有。
- **F-3**：M4（项目自己的读盖过较新的列表）是两份副本、固定先后的设计问题。I2 去掉它最尖锐的一种之后，连同评审的说明交给 P10。代价：项目的页面再打开之前，卡片上是旧的名字。
- **F-4**：plan 文字的漂移（P2）记在本文，plan 不改（P8a 先例）。代价：没有。
- **F-5**（修复轮的报告之后）：接受收集箱标签筛选的两个 `div` 改为按钮（手改的文件按 R3 清零；T7-b 先例；记进第 21 条），也接受交接的处理结果连带记下 P8a 的部分（M2-closeout 原来没有）。代价：没有。
- **F-6**（修复的复审之后）：m1–m5 在合并之前由同一个实现者用一个提交补上（P8a 的 F-10 先例）：一个功能开关的 hook 给两处页面用，hook 的测试发现接线的变异；自动归档的接线同样钉住；m3–m5 三处文档照实写。之后 sonnet 做范围复审。代价：一个提交。

**修复轮**（一次分派，九个提交）：
- **共用的测试替身**（`c3aff07d`，M7、P10）：
  - `core/lib/fake-toast.ts`、`fake-i18n.ts`，三个渲染测试和之后新加的都用；
  - `core/store/fake-project-lists.ts`：状态、标签 store 的记录和请求体、`changed()`、`listTab()` 和加载；只合并完全相同的，两个状态测试文件有意用不同的 `sequence`；
  - 没有一个 `expect` 变，测试数仍是 469（复审读过整个提交核实）。
- **store**（`3889eb43`，I1、I2、M3、M9、P16）：
  - I1、I2、M3 见上表；
  - M9：`place-between.ts` 的 `placeAt(items, key, droppedOnId, at, step)`，`placeBetween` 成为私有；三个 store 的移动测试不改就通过；`fw-m9-project-end-arg` 在原来的测试上存活，加"放到侧边栏最后"的测试；
  - P16：`ProjectFilterStore` 按工作区的 id 存，四个修改仍收 slug、经 `getWorkspaceBySlug` 找 id；新 `project_filter.store.test.ts`：同一个 slug 重建的工作区从默认的筛选开始。
- **页面的开关在轮到它时决定**（`7bbd69c0`，P14 扩大）：
  - `ProjectStore.toggleProject(projectId, field)`，`field` 是 `ProjectUpdate` 的布尔字段（`ProjectToggleField`）；`toggleAutoArchive(projectId)` 在 0 和 1 之间切换；都从 nerve 最近一次回答的项目算，store 不给出这个项目时不发；与 `updateProject` 共用私有的 `send()`；
  - 功能列表、功能页、访客开关和自动归档的开关改用它们；成员默认值只发改的那一个字段；两处 `as keyof Project` 和没有接住拒绝的 `void updateProjectPromise.then(…)` 删除。
- **nerve 的类型化请求体**（`33587894`，I3、P1）：`core/components/project/project-roles.ts` 的 `PROJECT_ROLES` 给两个角色下拉框用；`archive_in` 是数字；三个自动化文件用 `Pick<ProjectUpdate, "archive_in">`；新 `fake-controls.ts`；`projectMemberOf` 移进 `fake-projects.ts`。
- **包装层和侧边栏**（`41b64b27`、`76b7b220`，P13、M2、P12、M5、M6）：
  - P13：`useProjectFetch(workspaceSlug, projectId)`，`ProjectAuthWrapper` 传 `useParams().workspaceSlug`；新的一行：路由 store 仍是 beta、地址是 acme，beta 的项目是 `not-found`，四个键都是 `null`；
  - M2、P12（`workspace-wrapper.test.tsx` 5 个，每种判断一个变异）、M5（`use-project-drop.ts` 和它的测试）；
  - M6：`TTabPreferences` 删除，`DEFAULT_NAVIGATION: ProjectNavigation` 是还没取到时显示的默认，五个读者读 `default_tab`、`hide_in_more_menu`。
- **小项**（`f31a2c4b`，P15、P11、P4）：
  - P15：`plane-user-urls` 加命中样例 `/api/users/last-visited-workspace/`；
  - P11：去掉 `|| undefined`。这让收集箱的标签筛选成为手改的文件，R3 之下它的两个可点击的 `div` 改为按钮，清掉 4 条，web 的上限 364 → 360（F-5）；
  - P4：那一行删去：守着它的是参数的类型，`tsc` 守着，没有运行时的变异与它不同。
- **P3 的重跑发现的回退**（`6bbcceed`）：I2 之后项目自己的读为 `null` 时给出列表中的，而拒绝的测试中两份相同，于是 `probe-leave-forgets-read-first`、`rv-delete-forgets-read-first` 存活。拒绝的每一行改为从比列表中的新的 web 自己的读开始，两个都再被发现。
- **文档**（`2f6986da`）：spec 第 5 节（P9 加 E5；P10 加 P5、P6、P7、三个选择器和 M4 的说明，去掉已做的角色下拉框；M4 加下拉框和工作项 store 的取数），第 4 节和各节被改得不真的句子，附录 A.2、A.7、A.8、A.10；M2-closeout、M1-P3 的处理结果；M3 设计 `:1943` 的任务数；总体设计 7.7 的三条；前端改动清单。plan 不改（F-4）。
- 实现者在报告里写明的其余选择（复审没有提出异议）：I1 连同只为它的 `rootStore` 字段一起删；状态的两个测试文件只合并相同的记录；spec 附录 A.9 不重量（修复轮只删成员）；`mutants_p8b_t5before.py` 跑的是已不在的测试副本，不再适用。

**修复的复审**（opus，`4e71bb73..2f6986da`，九个提交、61 个文件）：通过，Critical 0、Important 0、Minor 5。
- 它在副本上：
  - 跑过门禁：vitest 54 个文件 497 个，turbo 54/54，web 360 等于上限，knip，修复轮改的 53 个 web 文件 0 条；
  - 亲手跑了 I2、I3（三处）、P13、P14、M3、M9、P16 的变异，都被发现；重新汇总 P3 的运行，与报告相同；
  - 读过 `c3aff07d` 的整个改动，没有断言变了；核实 `6bbcceed` 是对的。
- 五条 Minor：
  - **m1**：P14 五处中的三处，开关到 store 的接线没有测试（功能列表、功能页、自动归档的两处，四个变异在 vitest、`tsc`、oxlint 上都存活；S2 经接口改这些字段，也看不见）；
  - **m2**：功能开关的处理写了两遍；
  - **m3**：spec A.10 说修复轮手改的 Plane 文件都在 400 行以内，`projects-list-item.tsx` 是 460 行；
  - **m4**：M2-closeout 的结果说记下 P8a 的部分，却没有第 2 节的一条；
  - **m5**：M2-closeout 与 M3 设计对谁关闭 M2 交接第 11 节说法不一。
- 合并时的说明（不是发现）：本文第 6 节 P10 一行要写 P8a 的角色下拉框一项已在 P8b 做了；成员默认值的请求体类型和提示替身的重置，两个小处（第 7 节）。

F-6 的后续（`682bbff6`）：
- m1、m2：新 `core/components/project/settings/use-feature-toggle.ts`，`useFeatureToggle(workspaceSlug, projectId)` 是功能列表和功能页唯一的开关处理；`AutoArchiveAutomation` 自己做它的修改（`updateProject`、`toggleAutoArchive` 和拒绝的提示），自动化页只放它。共用的替身加 `Switch`（`fake-controls.ts`）和 `setPromiseToast`（`fake-toast.ts`）。新测试 `use-feature-toggle.test.tsx` 4 个、`auto-archive-automation.test.tsx` 3 个。8 个 `fw-f6-*` 变异（复审的四个照新的代码重写，加四个）都过得了 `tsc`，都由 vitest 发现。
- m3：A.10 点名 `projects-list-item.tsx`（460 行，修复轮只改两行，行数不变）。
- m4：M2-closeout 的结果加第 2 节中 P8a 的部分。
- m5：M2-closeout 的结果写明第 11 节的代码一侧已完成、页面一侧在 P9、P10，随 P10 关闭；M3 设计 13.1 同样写，`:1998` 本来就这样说。
- 门禁：`make lint-web`（web 360）、`make knip`、`make test-web`（web 56 个文件、504 个），`tsc` 通过；修复轮的变异表重跑 50 个中 49 个被发现，`fw-rw-t10-cycles-back` 只由 S2 发现。

**F-6 的范围复审**（sonnet，`2f6986da..682bbff6`）：通过。`useFeatureToggle` 是两个功能开关页面唯一的处理，经 store 的 `toggleProject` 在轮到时决定；测试断言调用的是 toggle、不是 `updateProject`；`AutoArchiveAutomation` 自己发出修改，月数是数字；没有留下孤儿，改到的 10 个文件 0 条警告；A.10、M2 的交接和 M3 设计 13.1 的说法都成立。

## 5. 最终实现与 plan 的差异

plan 的块记录的是当时的文字。执行中 Task 3、4、5、8 的修正轮为让之后的块照样能应用，改了 plan 中之后 Task 的块（Task 10 的 `useProject` 替身、Task 5 的两个块、Task 6–11 的上限和全局约束、Task 9 的两个块和散列行；plan 13,308 → 13,329 行）；各 Task 自己的文字和修复轮的不同不改（F-4，P8a 的先例），spec 在说规则的地方已改为与代码一致。与 plan 不同的地方都来自第 3、4 节的裁定：
- **生产代码**：
  - T1-b：五条文案键、`checkURLValidity`、封面辅助函数的死分支；
  - T3：`TFetchStatus` 删除；T3-a：`useArchivedProjectsFetch`；T3-c：两个工作项 store 的 `fetchParentStats` 不再重取项目；
  - T4-a：`projectsWithCreatePermissions` 是普通的 computed，`fetchProjectsWithCreatePermissions` 删除，`member-list-item.tsx` 认人；
  - T5-a：store 没有标签栏时修改不发；T5-b：`tab-navigation-utils.ts` 的三个修改，迭代、模块 store 的死字段，`cycle.store.ts` 的三处 `await`；
  - T7-a、T7-d：`state_id` 的标签页序号，`create-root.tsx` 的两行；
  - T8-b、T8-e：状态页的每条出错路径读 `errorMessageKey`；T8-d：状态表单只有编辑的三个字段；T8-g：`stateDetails`、`workspaceStateDetails` 删除；`state-item.tsx` 的拖放 effect 交回撤销；
  - T9-b：标签表单的值在 `defaultValues`，标签页和三个选择器的每条出错路径读 `errorMessageKey`，三条文案键删除；
  - T10-a、T3-d、T10-b：S2 记下查询串，管理员打开项目页，第七个测试的 CSP 检查和最后的复查；
  - E5：总体设计 7.7 的 PF-M2 一条；
  - 修复轮：F-1 的每一项（第 4 节），F-6 的 `useFeatureToggle` 和自己做修改的 `AutoArchiveAutomation`。
- **测试**：plan 之外加的：
  - 各 Task：Task 6 的五个测试；`state.store.changes.test.ts`（T8-a）、`label.store.changes.test.ts`（T9-a）；`use-archived-projects-fetch.test.ts`（T3-a）；`tab-navigation-utils.test.ts`（T5-b）；渲染测试 `create-update.test.tsx`（T8-d）、`create-update-label-inline.test.tsx`（T9-b）、`project-wrapper.test.tsx`（T10-b）；
  - 修复轮：`project_filter.store.test.ts`、`project-roles.test.tsx`、`select-month-modal.test.tsx`、`project-settings-member-defaults.test.tsx`、`use-tab-preferences.test.ts`、`use-project-drop.test.ts`、`workspace-wrapper.test.tsx`；F-6 的 `use-feature-toggle.test.tsx`、`auto-archive-automation.test.tsx`；共用的 `fake-toast.ts`、`fake-i18n.ts`、`fake-controls.ts`、`fake-project-lists.ts`。
- **plan 自己的文字**（P2，记下、不改）：
  - Task 1 的两个整文件块、`header.tsx` 的两个块、"完成时"一行（plan:1472），和 Task 2 中 `header.tsx` 的散列行（plan:1591），在 `7f7da15e` 之后过时；
  - Task 3 自己的文字（修正轮之后）；
  - Task 6 的测试清单（16 个）；
  - Task 7 说标签选择器"行为不变"（T7-b）；
  - Task 8 自己的 `issue/root.store.ts`、`root.store.test.ts` 的块和散列行仍带着 `stateDetails`（约 :9191、:9335、:9425、:9431；T8-i）；
  - Task 9 中 T8-h 改到的 8 个文件和 `.oxlintrc.json` 的块；
  - Task 10 的 S2 块和测试标题（T10-a、T3-d、T10-b）；
  - Task 11 的文字（E5 和四处更正）；
  - 全局约束：:21 的"最长的文件"（现在最长的是 `project.store.ts` 374 行），:22 的上限链停在 364（修复轮到 360）；
  - 修复轮的新文件不在 plan 的文件表里，列在 spec 第 2 节。
- **文件大小**：
  - P8b 写或重写的文件都在约 400 行以内。最大的是 `core/store/project/project.store.ts` 374 行，P8a 的 `core/lib/store-context.test.ts` 360 行（P8b 改过，350 → 360），`project-member.store.test.ts` 358 行，`state.store.test.ts` 352 行，`project.store.changes.test.ts` 351 行；S2 330 行（Task 10 的提交是 327，修正轮加 CSP 的检查和最后的复查）。
  - 分支改过的、原来超过 400 行的 Plane 文件都没有变长（例如 `cycle.store.ts` 642 → 635、`project-inbox.store.ts` 524 → 511、`projects-list-item.tsx` 476 → 460、`project/form.tsx` 462 → 411），除了裁定 D7 的三个（`issue-layouts/utils.tsx` 812、`power-k/config/navigation/commands.ts` 466、`work-item/commands.ts` 415）。
  - `tools/keywords.json` 2,189 → 2,209 行（Task 11 时 2,208；修复轮的 P15 加一个样例）。

## 6. 移交事项

本 Phase 关闭的（spec 第 7 节）：M2 交接第 3 节（M3 的 10 处模块级实例全部消失）和第 11 节中项目一侧的代码（`joinProject`、`leaveProject`、`plane-user-urls` 收紧）；M1-P3 的项目成员（`project-invitations` 的例外删除）；P8a review 第 6 节 P8b 一行（P14、P18、S2 的改写、两个包装层进 A6 的范围、F-1、F-4、F-5、F-6）。P8a spec 第 5 节 P8b 一行的落点见 spec 第 3 节的交接表。

同一个里程碑之内的后续 Phase 和之后的里程碑（写进它们的 spec；spec 第 5 节各行照录，下面是执行之后的补充）：
- **P9（工作区的页面）**：
  - spec 第 5 节 P9 一行照录（P8a spec 第 5 节 P9 一行，加 E5）。
  - **E5**：P8a 的 `use-navigation-preferences.ts` 经 `preferencesChangeOf`（`core/hooks/navigation-preferences.ts`）在调用时读显示的 `limitedProjectsCount`；P9 改写侧边栏的项目导航对话框时（P8a review 第 6 节的 P28）一并改为轮到它时从 nerve 最近一次回答算（总体设计 7.7 的 PF-M2 一条）。
  - M2 交接第 2、11 节的页面一侧：W2 的页面版本、新手引导的两步、工作区 general 页的时区选择的核对（M2-closeout 的 P8b 处理结果）。
- **P10（项目的页面）**：
  - spec 第 5 节 P10 一行照录。其中执行之后加的（F-2、F-3）：
    - **P5**（整分支评审更正，加 M1）：成员的离开、移出（`member-list-item.tsx:60`、`:68`）和改角色（`member-columns.tsx:151-152`，另有一条多余的 `console.log`）读 `ApiError` 没有的 `err?.error`、`err.error`；离开项目的弹窗（`leave-project-modal.tsx:66-72`）不读错误，总是固定的文字；加成员被拒绝只在控制台记一条（`add-project-members-modal.tsx:83-85`）；创建项目和项目设置读 Plane 的字段错误码（`projects/create/root.tsx:60-64`、`project/form.tsx:90-94`），`project.name_taken`、`project.identifier_taken` 从不显示；`form.tsx:153` 的 `checkProjectIdentifier` 不在 `try` 之内，被拒绝时加载不结束，是未处理的拒绝。照文案表（`errorMessageKey`）改。
    - **P6**：`project/form.tsx:154` 的 `!available` 一支没有检查（变异 `if (available)` 存活），设置页的测试钉住它。
    - **P7**：标签设置页的拖动把手只因 `onDrop` 在 `isUpdating` 变化时重建才重新登记；记住 `onDrop` 会让 pragmatic-dnd 留在已卸下的把手上。
    - **P9**：三个标签选择器的取数被拒绝时一直"加载中"或被吞掉（`label-dropdown.tsx:149-153`、`label-select.tsx:57`、`issues/select/dropdown.tsx:38`），Plane 原有。
    - **F-3**：项目的两份副本谁先（整分支评审 M4）：项目自己的读盖过较新的列表，卡片、侧边栏和权限 store 的角色显示旧的那份，直到再打开项目的页面；一般的规则由 P10 与架构一起定。
  - 标签栏还没取到时，页头的切换默认、隐藏、显示仍显示、点了提示失败（T5-a 的代价）：取到之前藏起它们是 P10 的页面的事。
  - P8a spec 第 5 节 P10 一行中两项已在 P8b 做了，P10 不再做：项目封面的上传和随机封面的删除（Task 1，裁定 D3）；项目成员的角色下拉框发出字符串（P8a spec 第 3 节第 12 条、:175、:236；修复轮，F-1 的 I3）。
  - M2 交接第 11 节的页面一侧（项目 general 页的时区选择的核对），本节随 P10 关闭。
- **P11（状态、标签的页面与清理）**：
  - spec 第 5 节 P11 一行照录：只经机械步骤到达的 36 个文件中 5 个的 14 条警告（spec 附录 A.8），`--rows M3` 剩下的行（A.9），核对 M3 加的关键词例外只剩 `until: "M4"` 一条。
  - **P1**（整分支评审的裁定）：`dropdowns/project/base.tsx:32` 的 getter 类型 `Partial<Project>` 改为 `Project`（唯一的调用者传项目 store 的 `getProjectById`），与它的 5 条警告一起在第 4 个任务做。
- **M4**：
  - spec 第 5 节 M4 一行照录。其中执行之后加的：
    - 按需的取数被拒绝时（F-2）：状态下拉框（`dropdowns/state/dropdown.tsx:29-35`）加载不结束、是未处理的拒绝，成员下拉框（`dropdowns/member/dropdown.tsx:35`）是未处理的拒绝；工作项 store 不等结果的取数（`issue/issue-details/issue.store.ts:132`、`:310`，`sub_issues.store.ts:338-342`）是未处理的拒绝；
    - 工作区一级显示的工作项不显示没有取过标签的项目的标签（T7-d）：M4 让工作项显示出来时就看得到，到 M7 加回工作区级的标签为止；
    - `fetchParentStats` 不再重取项目（T3-c）：工作项的修改将来若动了项目的字段，M4 再加一个不等的重取。
  - P8a 交给 M4 的各项照旧（P4：不在 action 之外写 `router` 的副本；分页、分组的列表扩展 `reconciled.ts`）。
- **M5**：spec 第 5 节照录：项目封面的上传连同 Task 1 删除的 28 张预设封面（M5 的 M1-closeout 已在修订中改，裁定 D3）。
- **M6**：spec 第 5 节照录。
- **M7**：spec 第 5 节照录：项目的收藏、`intake_count`，收集箱的分诊状态和工作区级的标签（Task 7 删除，M7 随各自的接口加回，连同工作区一级显示的工作项上的标签）；**PF-L3**：通知页的收集箱项经原来的 `useSWR` 读项目，写项目自己的读，可以盖过包装层的读，M7 照 7.1 改成按权限的取数（PF-L3 中工作项 store 的一半已由 T3-c 去掉）。
- **契约**：P8b 没有新的；P8a 的一条（`Workspace.organization_size`）照旧。
- **收尾**：
  - spec 第 5 节照录：核对 3.20 的 P8b 两行（总体设计 7.7、前端改动清单）。
  - **PF-L5**：没有检查发现两种语言都留下、已没有读者的文案键（`check:sync` 只比两种语言的键是否相同）；P8b 删除的由各 Task 的"完成时"和 spec 附录 A.7 的键表人工核对，收尾决定是否加一个检查。
  - M1-P3 交接的关闭条件在 M3 中都已做到，`status` 在收尾时改为 `closed`（它的 P8b 处理结果）。

## 7. 已知限制

- **项目自己的读先于列表**（F-3）：较新的列表输给较旧的项目自己的读，直到再打开项目的页面。不泄露什么：nerve 拒绝旧角色允许的操作。I2 之后，离开、删除的项目不再受它影响。
- **一个设置资源、两个队列**：`ProjectStore`（侧边栏的位置）和 `ProjectPreferencesStore`（标签栏）各建一个 `ProjectPreferencesService`，`PATCH me/projects/{id}/preferences` 走两个队列。无害：修改是部分的，每个 store 只写回答中自己的字段（spec 2.5、第 13 条）。
- **"store 没有"的拒绝都显示 `errors.unknown`**：`held()`、`membership()` 抛普通的 `Error`，四个 store 一致。
- **每次挂载都取数、聚焦时不取**（P8a 的 F-1）。SWR 有缓存的成员读时，项目页再挂载会先发出四个子资源的取数，再等 nerve 较新的读；调用者其间被移出时 nerve 答 403（整分支评审没有判断，属 F-1 的设计）。
- **项目一侧的状态留到这一代结束**：按 id 存，读不到，不再显示（P14，与 P8a 的 F-6 同）。
- **只由 S2 发现的变异**：`t10-unread-back`、`fw-rw-t10-cycles-back`、`t10-page-no-archived-call`。本机的 Docker 起不来期间只在持续集成中跑；`t10-page-no-archived-call` 让 S2 失败是从代码推出的，没有在端到端上跑过（spec A.2）。
- **没有自动检查的提示**：状态页的删除、设默认和拖动的提示（Task 8），标签设置页的删除、移动、"移出分组"和三个选择器中创建的提示（Task 9），`member-list-item.tsx` 认人的一支（T4-a，没有组件的渲染测试）。
- **修复的复审留下的小处**：成员默认值的请求体类型是 `Partial<Pick<Project, …>>`，不是 P1 规则的 `Pick<ProjectUpdate, …>`；`fake-toast.ts` 没有重置的辅助，六个测试文件各写 `toasts.length = 0`；"加入"一行中 ops 自己的读与列表中的相同，没有变异针对它；收集箱标签筛选的按钮的 `aria-label` 是英文字面（收集箱已有先例）。都不改。
- **工作项的根 store 先于状态、标签的 store 建**：它的 `autorun` 在第一次重跑时才绑上它们的 map，由新用户的数据触发。与 Plane 相同，测试覆盖结果（整分支评审的说明）。
- **标签拖放"放到自己组的最后"的语义**：store 照 spec 第 15 条做；HOC 是否给出这种指令要在浏览器中判断，整分支评审没有判断。
