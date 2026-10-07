# M3/P8a 工作区一侧的数据层：评审记录

| 项 | 内容 |
|---|---|
| Phase | M3/P8a `web-workspace-data` |
| 日期 | 2026-10-08 |
| 结论 | **通过**。<br>12 个 Task 逐个实现、逐个评审，都通过。Task 4、7、8 各有多条 Important，都是 plan 自己的测试缺口或执行中发现的缺陷，按裁定当轮补上或进修复轮。<br>整分支评审的结论是"修复轮之后可以合并"：Critical 0、Important 3、Minor 8。<br>修复轮一次分派（第二次只补复审的 7 条 Minor），落实了全部发现和执行中搁置的 P1–P31。复审通过（第 4 节）。 |
| spec / plan | [spec](../specs/P8a-web-workspace-data.md) / [plan](../plans/P8a-web-workspace-data.md) |
| 设计 | [M3 设计](../M3-design.md) 第 12 节 P8a；P8 拆成 P8a、P8b 见 `7cf3a286`（负责人 2026-10-07 确认） |
| 分支 | `worktree-m3-p8`，从 `main` 的 `2474d32e`（P7b 合并）分出 |

## 1. 评审方式

- **拆分**：设计里的 P8 `web-data-layer` 是 16 个任务的草稿。实施之前，按 P1–P7b 的实测估出约 19–21 个任务、约 20,600 行，两个上限都超出（约束 2）。控制者照第 16 节裁定拆成 P8a（工作区一侧）和 P8b（项目一侧），并改正第 16 节里"两个包装层挂载时的取数留在第一段"一句：项目一侧的取数随各自的 store 在 P8b（裁定 R1–R4，`$M3TMP/p8-split-rulings.md`）。设计提交 `7cf3a286`。负责人 2026-10-07 确认："按这个拆法来，继续推进"。
- **spec 和 plan 由架构子任务产出**（`8b553f3c`）：12 个 Task，plan 10,590 行，附带原型和逐 Task 复现。控制者的裁定 A1–A6 见 `$M3TMP/p8a-architect-rulings.md`，其中：
  - A1：Task 4 的 1,567 行算"约 1,500 行"；
  - A4：落点的全部判断进 `useLanding`，由 vitest 守着；
  - A6：会话的取数一律经 `useSessionSWR`，唯一的例外是公开操作（查看邀请），由根目录 `.oxlintrc.json` 的 `overrides` 静态地看住。
- **执行前的预检**（opus，只读，`$M3TMP/p8a-preflight/preflight.md`）：高 1、中 2、低 9。
  - H1：取数的权限条件写在调用处，没有检查。改为由 hook 自己从 store 算出条件，再加 vitest。
  - M1：Task 9 的拒绝邀请不在队列的测试里。
  - M2：Task 12 的文字。
  - L1–L9：标记、样例和文件清单。
- **修订**（`86ba4347`）：重放与原型逐文件相同（3,156 个文件）。125 个变异中 123 个被发现，存活的两个（`t11-cycles-back`、`t11-unread-back`）属于"M6、M7 的取数回到挂载路径"一类，交给 P8b 改写的 S2。
- **执行**：
  - 12 个 Task 各由一个实现子任务完成（不指定模型），再各由一个评审子任务（opus）检查是否符合 spec、质量是否达标。Task 的修复轮用 sonnet 做范围复审。同一时刻只有一个写入者。
  - 实现者在每个 Task 里照 plan 的变异表再跑变异，并加上自己的。每个 Task 都跑过 `make lint-web`、`knip`、`test-web` 和 `make e2e`（70 个）。
  - **执行中发现并修好的生产缺陷**：
    - T7：成员页把角色作为字符串发出，nerve 对每次改角色都答 400；
    - T8：成员页和个人主页取成员时，没有以调用者的列表为条件；
    - T5：只改保留名单时，turbo 重放缓存里的通过结果。
  - **执行中删除的孤儿**：T3 的 `invitation.svg`、T4 的 `EFileAssetType.WORKSPACE_LOGO`、T11 的首页工作项预览。
  - 其余裁定都是 plan 自己的测试缺口、说错的注释或文字（第 3 节）。跨 Task 的去重和收尾搁置到修复轮（P1–P31）。
- **整分支评审**（opus，`86ba4347..6df5853a`）：结论"修复轮之后可以合并"，Critical 0、Important 3、Minor 8；另外指出 8 条搁置项说得不对或太窄（第 4 节）。
- **修复**：一次分派，五步五个提交（`d390a05a`、`ae8b24c1`、`3290d3d1`、`5cf948cb`、`a0790d62`）。修复的复审（opus）通过，提出 7 条 Minor；同一个实现者用一个提交（`32622655`）补上，范围复审（sonnet）通过。
- **持续集成**：见第 2 节末。

## 2. 验收标准核对（spec 第 4 节）

| # | 标准 | 结果 |
|---|---|---|
| 1 | S1 和此前的全部故事通过：worktree 中 `make e2e` 70 个全部通过 | **通过。** 每个 Task 都是 70 passed；修复轮的五步之后（`a0790d62`）和第二次之后（`32622655`）都是 70 passed。复审在 HEAD 的副本上：69 个通过，只有 S3 因为读不到提交而失败（F4）。复审还用四个账户重跑了每一页挂载时的请求，与整分支评审的清单相同（第 4 节）。 |
| 2 | M2 交接第 3 节的 `git grep` 中 `WorkspaceService` 的 8 处消失 | **通过。** 37 → 27 处，M3 的 8 处都不在。`permissions.store.ts` 的 `workspaceService` 字段仍是旧的 axios service，P8b 删除（spec 第 3 节第 4 条）。剩下的 5 个模块级实例是 M4 的（编辑器和搜索）。 |
| 3 | `node tools/keywords.mjs` 通过：64 条规则，没有命中 | **通过。** 64 条规则、3 个例外，没有命中（`32622655`）。 |
| 4 | 9.5 中 P8a 的 vitest 通过，每个都有一个发现它的变异 | **通过。** web 的 vitest 33 个文件、294 个测试（Task 12 结束时 268 个）。plan 的变异表（spec 附录 A.2）在各 Task 里核对过；修复轮另有 67 个变异（第一次 62 个，第二次 5 个），都被发现，在 HEAD 上重跑过；复审重跑 20 个，又写了 22 个自己的。 |
| 5 | `.oxlintrc.json` 的 `overrides` 生效 | **通过。** A6（`no-restricted-imports`）和 L4（`no-non-null-assertion`）的每个条目各有一个变异，`swr/*` 的 `patterns` 一个（P24）；还有两个对照：改掉 `patterns` 之后 `swr/immutable` 不再被发现，只导入类型时不报。 |
| 6 | `tsc`、knip 通过 | **通过。** |
| 7 | 改到的文件按 7.9 没有 oxlint 警告；上限 452 → 435 | **通过。** 有手改的源文件都是 0 条，修复轮手改的 57 个 web 文件也都是 0 条，其中 10 个原来只经机械步骤到达（P5 的 9 个和 `issue/root.store.ts`）。只经机械步骤到达的 19 个文件共 15 条，列在 spec 附录 A.8，留给 P11 第 4 个任务。web 的上限 452 → 435，其余各包不变，没有新的抑制。 |
| 8 | 逐 Task 复现 | **通过。** 修订 `86ba4347` 时，架构子任务重放：每个 Task 之后门禁通过，最终与原型逐文件相同（3,156 个文件）。 |

**持续集成**：分支上每次推送都跑 server、web、e2e 三个任务，每次都通过：
- Tasks 1–3（`c596f8b1`）：run 37600158736；
- Tasks 4–7（`d91df7ed`）：run 37614474661；
- Tasks 8–12（`6df5853a`）：run 37633136541；
- 修复轮（`a0790d62`）：run 37645812371；
- 第二次（`32622655`）：run 37650426815。

本提交推送之后，分支上再跑一次；合并之后 main 的持续集成在合并时记录。

## 3. 执行中的决定

这一节列控制者在本阶段做的裁定，每条写明理由和裁定错了的代价。完整的记录在执行台账中。

**执行前**
- 拆分（负责人确认）、R1–R4、A1–A6，以及预检的 H1、M1、M2、L1–L9（第 1 节）。spec 第 3 节由这些裁定定下，执行中没有再议。

**执行中**
- **T1-a … T1-c（会话分代）**：
  - `dispose()`、`store-context` 的注释和测试名说"退役的一代什么都不再运行"，而迭代、模块的筛选的反应和 `issue/root.store.ts` 的 `autorun` 仍跟随共享的 `router`。改为只说释放的是项目筛选的反应，并点名 M4、M6 的观察者（T1-a）。
  - hook 的测试只有一个会话，固定 `loginId` 的变异要到 e2e 的 A11 才失败。改为加第二个会话 y，并用 hook 交给 SWR 的键调用 fetcher（T1-b）。
  - 测试里两条 MobX 严格模式的警告来自 M4 的 `autorun`，不在 P8a 修。移交 M4，并把"释放"扩大为"不在 action 之外写 `router` 的副本"（T1-c，P4）。
- **T2-a、T2-b（文案表的搬移）**：11 个机械编辑的文件中，新的导入在旧的 `// helpers` 注释下（P5）；`frontend-changes.md:217` 交给 Task 12。
- **T3-a、T3-b（系统内接受的删除）**：
  - 孤儿 `invitation.svg` 删除；两处说错的注释改正；关键词规则补上不命中的样例和各自的命中样例（T3-a）。
  - `creation_disabled` 的文案交给 P9（T3-b，P7）。
- **T4-a … T4-c（工作区的 store，评审 Important 6 条）**：
  - **I1**：取数与修改重叠时有三种错误：
    - 重复：建了之后，一个已经列出它的取数回来；
    - 过时：确认改名之后，较旧的取数回来；
    - 被超过：较旧的取数在较新的之后回来。

    M2 的 `ApiTokenStore`（总体设计 7.7 的范例）处理了这三种。**控制者裁定在修复轮的一步里从根上修**（T4-a，P8）：写一个共用的辅助，`ApiTokenStore` 也改用它，每个 store 测三种交错。
    - 理由：Task 7、8、10 由 plan 的块建出其余的 store，后面的锚点依赖 plan 的文字，中途重写会连锁；所有 store 都建好之后一次改，是一个连贯的改动，评审也只要一次。
    - 代价：缺陷在分支上（从不在 main）留到修复轮。
  - I2–I5 当轮补上（T4-b）：
    - 改名不是乐观的（`rv-update-optimistic`）；
    - 取数时回答一个与已载入的不同的列表（`rv-update-stale-rollback`）；
    - `RootStore` 给工作区 store 的客户端（`rv-root-public-client` 之前过得了一切）；
    - `WORKSPACE_LOGO` 删除。
  - 搁置的（T4-c，P9–P12）：I6（plan 的 Expected 行说 W1、W10、W3 覆盖页面，而这三个故事只走接口）以及 m1、m2、m6。
- **T5-a、T5-b（落点、设置 store、保留名单、离开）**：
  - 一次修复补上五处（T5-a）：
    - turbo 的 `web#test` 输入加上 `reserved_slugs.txt`，之前只改名单会重放缓存里的通过结果；
    - `useLanding` 的 SWR 配置；
    - slug 检查不排队；
    - `isOnboarded` 的两支；
    - `HandedFetch` 的说明。
  - 权限 store 的项目一半不能活过离开和删除，交给 P8b（T5-b，P14）。
- **T6-a … T6-c（权限 store 的工作区一半；包装层取列表）**：
  - 包装层的判断（取不到、加载中、找不到、就绪，8.3）没有测试，两个变异存活。作为修复轮的必做步骤 P15：`useWorkspaceFetch` 返回判断的联合类型，包装层只负责渲染（T6-a）。理由：Task 8、10 照 plan 的锚点重写这个 hook 和包装层，现在改会连锁。
  - 两个按角色大小过滤选项的地方，对每个能到达的调用者都是死的（只有管理员能到，决策点 4）。作为必做步骤 P16 删除（T6-b）。
  - 其余（T6-c）：P17 搁置；P18 交给 P8b；P19 交给 P9。
- **T7-a … T7-c（`MemberUser` 和工作区成员，评审 Important 5 条）**：
  - **I1**：成员页把角色作为字符串发出，nerve 对每次改角色都答 400。当轮改为发生成的 `WorkspaceRole`（数字），并去掉两处 `as unknown as`（T7-a）。理由：页面接上了 nerve，却发出 nerve 必定拒绝的请求，这是根上的缺陷。
  - I2–I5 当轮补上（T7-b）：nerve 的回答与请求分不开；名称排序的一半不会失败；只有一个工作区；会话；旧的资料。
  - 搁置的（T7-c，P20、P21）：Task 4 的同一个缺口；页面的提示读 `err.error`。
- **T8-a … T8-c（邀请；成员页按角色取数）**：
  - 邀请 store 的测试重复了四种已知的缺口，当轮补上。同时统一规则：被拒绝的再取保留 store 原有的数据，所有 store 都按这一条（T8-a）。
  - **I2**：成员页和个人主页取成员时没有以调用者的列表为条件（H1、spec 2.13），当轮改正（T8-b）。
  - 搁置的（T8-c，P24、P25）：两个过渡的文件和 L4 的三个文件。
- **T9-a、T9-b（邀请链接）**：
  - `useInvitationPreview` 的键和 `publicClient` 没有检查，从键里去掉 token 的变异过得了一切。新的 `use-invitation-preview.test.ts` 补上，service 测试里不会失败的断言删除（T9-a）。理由：公开的例外（A6）要有检查守着。
  - 其余（T9-b）：P8、P9、P24、P27。
- **T10-a、T10-b（工作区的显示设置）**：
  - 显示设置的测试重复了三种已知的缺口，"一个 store 被两代共用"的变异存活。当轮补上，成员的同一个缺口也补上（T10-a）。
  - 侧边栏的映射抽成纯函数 `navigation-preferences.ts`，用 `it.each` 测试，照 `landingPath` 的写法。理由：没有 DOM 的 view-model 测试是根上的修法；只靠 P9 的页面版本，它要由评审守一整个 Phase。
  - 对话框的几条交给 P9（T10-b，P28）。
- **T11-a、T11-b（M6、M7 的挂载时取数）**：
  - 首页的工作项预览随"最近"小部件失去了唯一的触发，删除（T11-a）。
  - 文档的部分，和 `ProjectViewStore` 没有写者的两个字段，搁置（T11-b，P29、P30）。
- **T12（文档）**：通过。措辞的小处搁置为 P31。P2（`issue_calendar_view` 只观察本代的资料，不用释放）在三处核实。

## 4. 整分支评审的发现与处理

结论"修复轮之后可以合并"：Critical 0、Important 3、Minor 8。评审在 HEAD 的副本上跑过每一道门禁，并用四个账户（管理员、成员、访客、非成员）实际运行了每一页挂载时的请求：没有一个角色请求它无权读的资源，每个 M3 的请求都发往 `/api/v0`。它还核对了：
- 换代的全过程；
- 删除的完整性：deadsym 没有新的生产代码行，每个删除的名字 0 处；
- 完成线的每一项。

| 发现 | 处理 |
|---|---|
| **I1** 工作区成员的取数写了三遍（包装层、成员页、个人主页），同一个 SWR 键上有三种配置；个人主页的配置没有检查 | F-1、F-2：配置只有 `useSessionSWR` 的一份；成员的取数一个 hook（`core/hooks/use-workspace-members-fetch.ts`），条件、键、fetcher 在一处，一个测试 |
| **I2** store 测试的辅助在 Task 之间重复（5 个载入辅助、3 份内联的"会话已换"、4 份 store hook 的 mock），与 spec 2.4 的"各只有一处"不符 | F-3：`FakeNerve.replaceSession`；一个核对方法和路径的 `answered`，每个 store 的载入都用它（含 M2 的 `loadList`）；一个 store hook 的假实现 `fake-store-hooks.ts`；`invitationOf` 并入共用的构造 |
| **I3** `ONCE` 名不副实：全局的 `revalidateOnMount: true` 决定每次挂载（SWR 2.4.2），于是每次挂载都重新取数；P15 的改写和 P15、P25、P28 要钉的配置都建在这个错误的前提上 | F-1：选"每次挂载都取"，见下 |
| **M1** 按 slug 存的状态活过工作区的删除和离开（slug 可以再用） | F-6：按工作区的 id 存放；删除再建同名、离开，两种情况各有测试 |
| **M2** spec 第 2 节与 HEAD 不符的地方多于 P23 | spec §2、2.8–2.10 同步 |
| **M3** 第二个新的 `as` | `in-session.test.ts` 给 hoisted 工厂写返回类型 |
| **M4** 新手引导的邀请一步把数组当成 Plane 的映射，选错工作区 | 去掉 `Object.values`；"邀请进刚建的工作区"交给 P9 |
| **M5** 两处注释过时 | 改正（`onboarding/page.tsx`、`preferences.go:33`） |
| **M6** `settings/mobile/nav.tsx` 把 R4 的名字改回去 | 用 store 的名字 |
| **M7** spec 第 5 节 P9 一行漏了四处列表的读者 | 写进 spec 第 5 节和本文第 6 节 |
| **M8** 判断的联合类型的标签不一致 | 都用 `kind`（`Landing`、`WorkspaceAccess`、个人主页的判断） |
| 搁置项说得不对或太窄：P8（7 处）、P14（工作区一侧）、P15、P21、P23、P24、P25、P28 | F-1、F-4–F-8 |

**控制者的裁定**（台账里每条都写了理由和代价）：
- **F-1**（I3）：选"每次挂载都取"。`useSessionSWR` 自己持有每个会话取数的唯一配置：不在聚焦时取，拒绝不重试，不涉及挂载。于是应用的 `revalidateOnMount: true` 让每次挂载都取，`revalidateIfStale: true` 让每次换键都取。它的 `config` 参数删除。
  - 理由：
    - M2 有意写下的三处配置（令牌列表、当前用户、落点）正是这一份；
    - 同一个键上配置不一的问题（I1）从构造上不可能再出现；
    - 挂载时取数能看到别处改了的角色和成员关系；
    - 变宽的竞争窗口由 F-4 处理。
  - 代价：以后某一页要"一个会话只取一次"时，再加回参数；每次进入工作区的页面多发几个 GET。
- **F-2**（I1、P26、M8）：成员的取数只有一个 hook。包装层的判断是"这是不是他的工作区"的唯一判断；成员页只再加管理员的邀请。个人主页拿到包装层给的工作区，只从成员的取数得出加载中、失败、是成员、不是成员。
- **F-3**（I2）：测试辅助合成一套。
- **F-4**（P8 扩大）：一个辅助 `core/lib/reconciled.ts`，按键记住最新的取数，并把取数在途时 nerve 确认的修改作为幂等的函数按序重放到取数的回答上。它不假定队列。它覆盖评审列出的七处：
  1. 按键的最新取数；
  2. 显示设置是一份文档；
  3. 移出成员是改为 `is_active: false`；
  4. 成员的取数从调和之后的结果写共享的 `users`，`updateMember` 用 nerve 的回答刷新它；
  5. 删除和离开按 id；
  6. 每个集合各测三种交错；
  7. `ApiTokenStore` 改用它。

  P20 在这一步：每个 store 的会话测试先载入，再换会话、再取，状态不变。
- **F-5**：`ApiTokenStore` 也经 `oneAtATime()` 排队。
  - 理由：总体设计 7.7 的规则不限范围；M2 的 codex-fixes 只给有更新的两个 store 排了队。
  - 代价：同时发出的建令牌和撤销令牌要等一个往返。
- **F-6**（M1）：工作区内的状态（成员、邀请、显示设置）按工作区的 id 存放，SWR 的键带 id；getter 仍收 slug，经调用者的列表找到 id。
  - 离开或删除的工作区的状态立刻读不到；再建同名的从空开始。
  - 代价：每次读多一次列表查找；丢掉的 id 的条目留到这一代结束。
- **F-7、F-8**：Minor 和其余搁置项（见下）。
- **F-9**：P22（让 `tsc` 挡住以字符串发出角色）不做。
  - 理由：角色下拉框用的是 `@nerve/ui` 的 `CustomSelect`，它的 `value`、`onChange` 是 `any`，web 有 20 个文件在用；改它是跨包的改动，对 M3 只换来一个下拉框。P9 的 W7 页面版本断言 PATCH 的角色是数字，已写进 spec 第 5 节 P9 一行。
  - 代价：P9 之前，字符串的角色回来时 `tsc` 看不见。
- **F-10**：复审的 N1–N7 合并之前补上（见下）。

**修复轮**（一次分派，五步）：
- **测试辅助**（`d390a05a`，F-3）。
- **调和与按 id 存放**（`ae8b24c1`，F-4、F-5、F-6、P8、P20、M1、P23 的代码部分）：
  - 新文件 `core/lib/reconciled.ts`（127 行），`ApiTokenStore` 自己的一份删除；
  - 测试按集合各有三种交错，"重复"不适用的（成员、显示设置是按 id 覆盖或一份文档）在测试文件里用一句话说明；
  - `issue/root.store.ts` 唯一的外部读者 `workSpaceMemberRolesMap` 原来没有任何读者，删除。
- **取数的 hook**（`3290d3d1`，F-1、F-2、P15、P26、I1、I3、M8）：
  - `useWorkspaceFetch(slug)` 返回 `WorkspaceAccess`（取不到并可重试、加载中、找不到、就绪），包装层只负责渲染；
  - `it.each` 有五行，"slug 不在列表上"分成"列表上有别的"和"列表是空的"，于是 `hasWorkspaces` 也钉住。
- **小项**（`5cf948cb`）：
  - P16：先核实管理员的门槛，再删除两处按大小过滤的代码和喂它们的读、守卫、`hasRoleChangeAccess`、`isAdmin`、`shouldRender`；
  - P17、P24（A6 加三个文件和 `swr/*`；L4 加十三个文件）；
  - P5：11 个文件都到 0 条，没有留给 P11 的；
  - M3–M6。
- **文档**（`a0790d62`）：
  - spec 在说规则或清单的地方与代码一致；
  - 总体设计 7.7（P31，F-1、F-4、F-5、F-6）；
  - `frontend-changes.md`；
  - M5 的交接一行（P13）。

  plan 的文字不改，与代码的不同记在修复轮的报告里（P7 的先例；见第 5 节）。
- 实现者的偏离，控制者都接受：
  1. 删除、离开和三个按工作区的取数改收工作区（id 和 slug），免得再查一次、免得 id 与 slug 对不上。
  2. 从未取过显示设置时做的修改不再写入，由下一次取数带来（F-1 之下每一页挂载都取）。
  3. 辅助自己处理 `SessionChangedError`。
  4. 没有调用者的 `replacedSessionClient()` 删除。
  5. P24 的两个假实现在 `core/store/` 而不是简报写的 `core/lib/`。

**修复的复审**（opus，`6df5853a..a0790d62`）：通过，Critical 0、Important 0、Minor 7。
- 结论：F-1 到 F-9、I1–I3、M1–M8 都在根上落实；`reconciled.ts` 覆盖七处，没有特例，也没有 store 留着自己的一份。
- 它在副本上：
  - 跑过每一道门禁（vitest 290、oxlint 435、e2e 69 加上 S3 的 F4）；
  - 重跑了报告里的 20 个变异；
  - 写了 22 个自己的变异：存活的 5 个中，3 个成了 N1–N3，1 个是说明 B，1 个是应当存活的对照。

  F-10 裁定 7 条都在合并之前补上（`32622655`）：
  - **N1**：包装层第一次取列表就失败时没有一行，把"加载中"的判断移到"取不到"之前的变异存活。加一行，显示可重试的"取不到"。
  - **N2**：个人主页"再取失败、store 里仍有这个成员"没有一行。加一行，显示成员。
  - **N3**：文档把"换键时取"归功于 `revalidateOnMount`，实际是 `revalidateIfStale`。测试钉住 `revalidateIfStale: true`，文档写明两条规则。
  - **N4**：7.7 新加的两条写成以后每个 M 都照做的规则，`reconciled.ts` 是它唯一的实现，分页或分组的集合扩展它而不是另写一份（M4）。
  - **N5**：被超过的取数无论回答什么都给 `undefined`，由较新的那个决定，有测试。
  - **N6**：个人主页读给它的工作区的成员关系，不经路由；有测试，路由的工作区与给的不同。
  - **N7**：spec 附录 A.7 的 W12 按 HEAD 重量（171 个文件，`as` 61 → 45，`any` 38 → 25，`!` 0 → 0）；2.14 写"P8a 的四个集合"并点名令牌的那个测试。

  两条说明不用改：
  - **A**：跨工作区共享的 `users` 资料可能被另一个工作区较旧的取数写回旧名字；那不是本标签页确认的修改，下一次挂载的取数改正它。
  - **B**：`provider.tsx` 把 `WEB_SWR_CONFIG` 交给 SWR 这一环只有评审能看到；测试的注释点名了它。
- **第二次的范围复审**（sonnet，`a0790d62..32622655`）：通过。N1–N7 都照裁定在根上落实，每个新测试都能在它的变异上失败（按代码推理）；文档的每一句在 HEAD 上成立；没有新的问题。`fake-store-hooks.ts` 保留跟随路由的 `getWorkspaceMemberDetails`，它是接口上仍有别处使用的成员，不是只为测试留的。

## 5. 最终实现与 plan 的差异

plan 的块记录的是当时的文字，保留原样，本提交不改 plan。spec 在说规则的地方已改为与代码一致（P7 的先例）。与 plan 不同的地方都来自第 3、4 节的裁定：
- **生产代码**：
  - T7-a：角色作为数字发出；
  - T8-b：取成员以列表为条件；
  - T3-a、T4-b、T11-a：删除孤儿；
  - T5-a：turbo 的输入；
  - T10-a：侧边栏的映射抽成纯函数；
  - 修复轮：
    - F-1：`useSessionSWR` 的一份配置，删除 `config` 参数和 `ONCE`；
    - F-2：一个成员取数的 hook，包装层的判断；
    - F-4：`core/lib/reconciled.ts`；
    - F-5：`ApiTokenStore` 排队；
    - F-6：按工作区的 id 存放；
    - P16 的删除。
- **测试**：各 Task 的修复轮（第 3 节）和修复轮。
- **plan 自己的文字**：修复轮的报告记下了与代码不同的地方：
  - P1：释放的范围（plan:34、:244、:493、:505、:534-535、:580-582）；
  - P9：Expected 行说 W1、W10、W3 覆盖页面（:3512、:4834、:5809、:10053）；
  - P23：Task 7 的测试清单；
  - P25：:7343 说 `IWorkspaceMemberInvitation` 在 Task 8 删除，实际是 Task 9；
  - 全局约束 :21 的"最长的文件"；
  - 修复轮的新文件不在 plan 的文件表里，列在 spec 第 2 节。
- **文件大小**：分支改过的源文件都在约 400 行以内。最大的是：
  - `core/store/workspace/index.test.ts` 382 行；
  - `core/store/user/api-token.store.test.ts` 368 行；
  - `core/store/member/workspace/workspace-member.store.ts` 367 行（成员和邀请仍是一个 store，没有拆）。

## 6. 移交事项

同一个里程碑之内的后续 Phase 和之后的里程碑（写进它们的 spec；spec 第 5 节各行照录，下面是执行之后的补充）：
- **P8b（项目一侧的数据层）**：
  - spec 第 5 节 P8b 一行照录。
  - **P14**：权限 store 的项目一半（`projectUserInfo`、`workspaceProjectsPermissions`）不能活过离开和删除工作区，照 F-6 按工作区的 id 存放。`workspace-wrapper.tsx` 在 `currentWorkspaceInfo` 有缓存时直接渲染子组件。
  - **P18**：项目一半的权限表包含 PM+WA（项目成员兼工作区管理员即项目管理员）和 WA-，杀掉 `permissions.store.ts` 中读列表的变异（`pf-project-admin-any`）。
  - **S2 的改写**要发现"M6、M7 的取数回到挂载路径"这一类：迭代、模块、视图、分诊状态、收藏、未读通知数、"最近"；`t11-cycles-back`、`t11-unread-back` 是样例。
  - 两个包装层的取数改经 `useSessionSWR` 之后，加进 A6 的范围。
  - 项目一侧的 store 照总体设计 7.7：
    - 会话的取数不传配置（F-1）；
    - 取数与修改经 `reconciled.ts` 调和（F-4）；
    - 修改排队（F-5）；
    - 按工作区和项目的 id 存放（F-6）。
- **P9（落点、新手引导、邀请页、工作区页面）**：
  - **P7**：`onboarding.workspace.creation_disabled` 的文案照设计 7.4、决策点 2。
  - **P9**：W1、W3、W10 只走接口，页面版本是 P9 的。
  - **P12**：改名被拒绝只有 `console.error`。
  - **P19**：设计 7.5 已改为由调用者的列表决定"找不到"（本提交）；P9 不加第二种判断。挂载着的包装层换 slug 时不重取列表，在别的标签页加入的工作区要到下一次挂载才显示（M7）；P9 若要改，改这一个判断。
  - **P21（扩大）**：下面这些提示读 `ApiError` 没有的 `err.error`，nerve 的拒绝（`sole_admin` 等）显示成通用的文字，要改用文案表：
    - `member-columns.tsx:149`；
    - `members-list-item.tsx:55`、`:70`；
    - `invitations-list-item.tsx:59`、`:148`；
    - `members/page.tsx:69`；
    - `onboarding/steps/team/root.tsx:315`。
  - **P22**（F-9）：W7 的页面版本断言 PATCH 的角色是数字。
  - **P27**：
    - 缺 `token` 或 `invitation_id` 的链接一直转圈；
    - 接受、拒绝之后，缓存的查看仍是 `declined: false`；
    - 没登录时接受得到 401，进了 `console.error`；
    - W5、W6 的页面版本（plan:9217、plan:8492 误写成 A9）。
  - **P28**：`project-navigation-dialog.tsx`：
    - 拒绝改走文案表（`SessionChangedError` 不提示）；
    - 数量框失焦或防抖之后再发；
    - 挂载时捕获的旧上限；
    - 侧边栏在显示设置到达前后的切换。
  - **M4**：新手引导的邀请发往刚建的工作区，而不是排在第一个的。
  - **M7**：
    - 读列表而没有取数的三处：新手引导的 `hasWorkspaces`、个人设置侧边栏的工作区、power-K 在工作区页面之外打开工作区的菜单。直接打开 `/settings/profile/*` 时，三处都没有工作区。
    - 创建成功之后再取列表被拒绝，被当成"创建失败"，重试时遇到"slug 已占用"。根在 `createWorkspace` 只写进已经取到的列表。
- **P10**：spec 第 5 节照录。
- **P11**：只经机械步骤到达的 19 个文件的 15 条警告（spec 附录 A.8）。P5 没有留下文件。
- **M4**：
  - spec 第 5 节照录。
  - **P4**：不在 action 之外写 `router` 的副本：把 `workspaceSlug`、`projectId` 等做成 `rootStore.router` 上的计算属性，或改成 `reaction`，由 `RootStore.dispose()` 调用它的释放函数。在那之前 `store-context.test.ts` 有两条 MobX 严格模式的警告。
  - 工作项的分页、分组列表扩展 `reconciled.ts`，不另写一份（总体设计 7.7）。
- **M5**：spec 第 5 节照录；P13 的一行已同步。
- **M6**：spec 第 5 节照录。**P3**：退役的模块筛选的反应在每次换路由时仍写 localStorage，由 M6 释放。
- **M7**：spec 第 5 节照录。**P30**：`ProjectViewStore.loader` 和 `fetchedMap` 没有写者了；M7 的视图取数要设置它们，否则删除它们。
- **契约**：**P11**：下一个改 `api/modules/workspace.yaml` 的人，把 `Workspace.organization_size` 收紧到数据库 CHECK 的值。
- **收尾**：
  - M2 的 `ApiTokenStore` 在本 Phase 改用 `reconciled.ts`，并且排队（F-4、F-5）：M2/P4 spec 2.8 中关于它的描述以总体设计 7.7 为准。
  - 核对 3.20 的 P8a 两行（总体设计 7.7、前端改动清单）。

## 7. 已知限制

- **P8a 与 P8b 之间，工作区的页面仍有项目一侧的 404**（spec 附录 A.5）：拆分的已知过渡，P8b 的 S2 改写之后核对清单为空。
- **每次挂载都取数**（F-1）：进入工作区的每一页都重取列表、成员和显示设置。别处的改动在下一次挂载时可见；两次挂载之间，聚焦时不重取。
- **离开或删除的工作区，它的状态留到这一代结束**（F-6）：按 id 存放，读不到，不再显示。
- **跨工作区共享的 `users` 资料**（复审说明 A）：另一个工作区较旧的成员取数可能写回一个人旧的名字，下一次挂载的取数改正它。
- **`provider.tsx` 把 `WEB_SWR_CONFIG` 交给 SWR 这一环只有评审能看到**（复审说明 B）：vitest 不渲染 provider，测试自己合并这份配置，注释点名 `provider.tsx`。
