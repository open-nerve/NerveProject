# M3/P10 项目的页面：评审记录

| 项 | 内容 |
|---|---|
| Phase | M3/P10 `web-project-pages` |
| 日期 | 2026-10-11 |
| 结论 | **通过**。<br>11 个 Task 逐个实现、逐个评审，都通过。Task 1–4、8、10、11 各有一次评审之后的修正轮；Task 6–10 在评审之前各有一轮（实现者自己探查出的缺口，控制者裁定之后先补上）；Task 5 没有修正轮。各 Task 的评审共有 11 条 Important：Task 2 的两条是执行中发现的缺陷，Task 3、4 的各三条和 Task 8、10 的各一条是没有会失败的检查的性质（代码本来对），Task 11 的一条是文档；其余是 Minor。Task 2 的分支持续集成在 e2e 失败一次，修正轮找到根因（裁定 T2-f）。<br>整分支评审的结论是"修复轮之后可以合并"：Critical 0、Important 1、Minor 6。<br>修复轮一次分派、十一个提交（`9c3a4e10` 到 `28526539`），落实裁定 F-1–F-9 交给它的发现和搁置项 P2–P11（P1 的 plan 漂移记在本文）。修复的复审 Critical 0、Important 0、Minor 2，同一个实现者用一个提交补上（`621c9390`，裁定 F-10、F-11），它的报告之后再用一个只改 spec 的提交补上（`093d0d91`，裁定 F-12）；两个提交的范围复审通过，没有发现。<br>控制者的浏览器核对 C6–C8 都通过（第 2 节）。本记录的初稿指出 C7、C8 漏了的调用方，控制者补做核对：标签式导航的项目选择和扩展侧边栏的复制通过，另在 P10 的页面上找到 C7 的两个缺陷：创建弹窗中网络的选择 Tab 到不了（D1），焦点从创建弹窗走到项目卡片关着的右键菜单（D2）。修它们时又分出两个：创建表单的正的 `tabIndex` 让 Tab 离开弹窗（缺陷 i），每个经 `ModalCore` 的弹窗后面的页面对辅助技术仍然可见（缺陷 ii）。裁定 F-13–F-15 修好 D1、D2 和缺陷 i（`d30199b2`、`b6344182`、`6db732a1`、`13388d2f`，文档 `9b3ff184`），在修好的构建上再核对通过。<br>缺陷 ii 涉及 M2–M8 的每个弹窗，是本阶段唯一交给负责人的一项：负责人裁定（2026-10-10）放进 M3 收尾，作为一项独立的任务，P10 不等它合并（第 3 节）。<br>这几个提交的复审（opus）Critical 0、Important 0、Minor 4，没有代码的缺陷；跟进照裁定 F-16、F-17 改文档，删去没有读者的 `CREATE_LABEL`（`456904f4`）。实现者在那次跟进中指出创建弹窗的功能一步有同一个缺陷，照裁定 F-18 修好（`03157319`，文档 `3e9d0fe9`）。这三个提交的范围复审（sonnet）通过，没有发现（第 4 节末）。分支在 `3e9d0fe9` 上的持续集成通过。 |
| spec / plan | [spec](../specs/P10-web-project-pages.md) / [plan](../plans/P10-web-project-pages.md) |
| 设计 | [M3 设计](../M3-design.md) 第 12 节 P10 |
| 分支 | `worktree-m3-p10`，从 `main` 的 `6c1a090a`（P9 合并）分出 |

## 1. 评审方式

- **spec 和 plan 由架构子任务产出**（`fca1e01a`）：
  - 设计 12 节 P10 的草稿是 15 个任务，plan 是 11 个 Task、9,344 行（最长的 Task 3 是 1,473 行），附带原型和逐 Task 的快照；从 `6c1a090a` 的新副本照 plan 复现，每个 Task 之后门禁通过，最终与原型逐文件相同。
  - 94 个变异在最终原型上都被发现（93 个在自己的 Task 的树上；`T2.8` 从 Task 4 起）。W18 另写 30 个探查变异：P10 新加或改写的 105 个测试每个至少失败一次。
  - 唯一没有会失败的检查的性质是 general 页的 `key={project.id}`。spec 第 3 节标"裁定"的七条由控制者裁定 R1–R7（`$M3TMP/p10-architect-rulings.md`），都接受（第 3 节）。
- **执行前的预检**（opus，只读，`$M3TMP/p10-preflight/preflight.md`，在 `fca1e01a` 上）：高 0、中 5、低 10。复现干净：每个 Task 之后门禁通过，最终与原型逐文件相同（3,200 个文件）。R4 的 `answeredAt` 在取数、确认、换代的每种先后下都对；R1、R3（键盘）、R5 成立；架构子任务的 23 个变异重跑都被发现；P10 的故事各跑 6 次都通过。
  - M1：R7 照写做不到。应用中没有从一个项目的设置页到另一个项目的设置页而路由不卸载的路，经命令面板的测试写不出；预检的探查（`history.pushState` + `popstate`）在去掉 `key` 时失败。
  - M2：R6 说被拒绝的离开弹窗留着，可成员页自己一行的确认框在 `await onSubmit()` 之后不论结果都关；侧边栏、项目页头的离开弹窗被拒绝时留着，没有检查（PF19 存活）。
  - M3：创建在途时页头的关闭按钮禁用，没有检查（PF6 存活）。
  - M4：`CustomSearchSelect` 用自定义按钮的一支出了 Tab 顺序，没有检查（PF15 在 P5、W3、P1 上存活）：添加成员弹窗的成员选择从没有被键盘到达。
  - M5：成员下拉框改写之后，M4–M6 的调用方得到的不同，第 3 节第 15 条和第 5 节都没有写：`renderByDefault` 只管提示（桌面上每一行都立即渲染 Popover 和 popper），按钮的点击不再自己阻止冒泡，手机上打开时搜索框取得焦点，列表没有 popper 的 12 像素边距，按键改由 Headless UI 处理。
  - L1–L10：R2 的文字漏了 M3 设计 9.7 C7、13.1 和 M2 交接第 14 节的关闭条件；R4 的 7.7 一句"写进每一份"字面上不对（已归档的列表不写）；`T2.8` 的标记；P8 的 `toPass()` 没有期限；2.13 漏了确认框的关闭；plan 写了不存在的 `optionsFilter`；输入法组字时的 Escape 关上列表；`WhileOpen` 在列表打开时记下回调；面包屑的下拉框成了两个 Tab 停留点；Task 11 的两行说的比代码多。
- **修订**（`4a4e7446`）：落实 R1–R7 要的文档和预检的每一条（控制者的裁定见 `$M3TMP/p10-amend-brief.md`，落点见 spec 第 3 节"预检之后"）。
  - 每个弹窗一条规则：被拒绝时留着，只在成功的跟进中关上（A-M2）；成员下拉框对调用方的约定照旧（A-M5，新的 `dropdowns/member/base.test.tsx`）；P3 在同一个路由里移动，守着 general 页的 `key`（A-M1，`T5.7`）。
  - 从 `6c1a090a` 的新副本重新逐 Task 复现：每个 Task 之后门禁通过，最终与原型相同，只差这个提交的四份文档。plan 9,718 行，最长的 Task 3 是 1,478 行。plan 的 106 个变异在最终原型上都被发现（103 个在自己的 Task 的树上；`T2.8`、`T2.9`、`T2.14` 从 Task 4、8、3 起，各在那个 Task 的树上核对过）；预检的 7 行（PF1、PF6、PF7、PF13、PF15、PF18、PF19）都被发现。W18：109 个测试，106 个变异中的 50 个让其中 67 个失败，31 个探查让其余 42 个失败。
  - e2e 93 → 106，vitest 657 → 761，关键词守卫 67 → 68 条规则，web 的 oxlint 上限 356 → 350 → 349，`propel` 16 → 11。
- **执行**：
  - 开始时的裁定（照 plan 执行、E1）见第 3 节。
  - 11 个 Task 各由一个实现子任务完成（不指定模型），再各由一个评审子任务（opus）检查是否符合 spec、质量是否达标。评审之后的修正轮由同一个实现者做，sonnet 做范围复审；Task 1 只改一句注释、Task 11 只改文档的修正轮由控制者逐字核对，代替范围复审；Task 4 的范围复审要的一行文档由控制者写（`46bfd390`）。同一时刻只有一个写入者。
  - 从 Task 6 起，实现者照前几个 Task 反复出现的缺口类别（在途时的忙、换账户之后兑现的回答、`done` 的交接、谁被提供什么）自己写探查变异，把存活的报告出来；控制者裁定全部补上，实现者在评审之前用一轮补上（Task 6–10 各一轮），评审看两个提交。
  - 实现者在每个 Task 里照 plan 的变异表再跑变异（每个 Task 一个新目录 `$M3TMP/p10mut/tN`，裁定 T1-a），并加上自己的探查。每个 Task 的实现和修正轮都跑过 `make lint-web`、`make knip`、`make test-web` 和本地的 `make e2e`（裁定 E1）；Task 1 只改注释的修正轮跑 `make lint-web`、`make test-web`，Task 11 和它的修正轮只改文档，跑 `make lint-web`、`make knip`。
  - Task 7 之后负责人要求暂停（接近速率上限，换账户），之后在新的会话中继续（worktree、SDD 的文件和 `$M3TMP` 都在）。
  - **执行中发现并修好的生产缺陷**（都在分支上修好）：
    - 执行之前（预检；修订改了 plan 的代码）：
      - M2（A-M2）：成员页自己一行的离开被 nerve 拒绝时，确认框也关上，与侧边栏、项目页头的离开弹窗不同；
      - M5（A-M5）：成员下拉框对 M4–M6 的调用方的约定变了：桌面上列表的每一行立即渲染选择和 popper（原来指针经过之前只有按钮），手机上每次打开都弹出软键盘，列表没有离窗口边缘 12 像素。修订照原来的约定恢复；点击不冒泡一项是 Headless UI 的 `Popover.Button` 自己做的，不是缺陷；
      - L7：输入法组字时的 Escape 关上带搜索的选择的列表；
      - L9：面包屑的下拉框在项目、迭代、模块、视图的页头一节有两个 Tab 停留点（原来一个）。
    - T2：
      - 只在持续集成上失败的 P1（裁定 T2-f）：修订照 L7 让面板的 Escape 处理在组字时提前返回，返回在 `stopPropagation` 之前，组字结束的 Escape 于是离开列表、传到 `document`，创建弹窗的 `useKeypress("Escape")` 关上弹窗。本地照持续集成的条件跑没有复现，16 个 worker 下复现；P1 只在它的最后几步落在弹窗 200 毫秒的离场过渡里时才通过；
      - 评审的 I1（裁定 T2-a）：调用方的 `optionsClassName` 落在 z-30 的面板里没有定位的 `<ul>` 上，M4 的三处调用方给成员下拉框的列表的 z-index（`all-properties.tsx`、`draft-issue-properties.tsx` 的 `z-10`，`assignee-column.tsx` 的 `z-[9]`）不起作用，列表都叠在 30 层，破坏 A-M5 恢复的约定；
      - 评审的 I2（裁定 T2-b）：面包屑唯一的 Tab 停留点（`navigation-search-dropdown.tsx`，`outline-none`）和项目页头的选择没有看得见的键盘焦点，BASE 上标题按钮有焦点环；成员下拉框中 Escape 之后焦点在选择自己的按钮上，也看不出（M2；BASE 上焦点落到 `body`，不是回退）；
      - 评审的 M1（裁定 T2-c）：选择的按钮新的悬停底色盖过 6 个调用方的 `hover:bg-transparent`；根新加的 `flex-shrink-0` 让收集箱的指派人一行溢出 8 像素。
    - T2-g（搁置为 P2）：Plane 的 `useKeypress` 不看 `isComposing`，组字结束的 Escape 关上创建项目的表单（Plane 原有）。
    - 整分支评审：P2 说得太窄。探查 PR1 中组字结束的 Escape 也关上经 `ModalCore` 的弹窗（删除项目的弹窗关了；离开弹窗同样有可以组字的文本框）和表情选择器（在它的搜索框中）。修复轮在三个处理器里修好（F-3）。
    - 浏览器核对之后（裁定 F-13–F-15、F-18；第 3 节）：
      - D1（控制者补做的核对，F-13）：创建弹窗中网络的选择 Tab 到不了。正的 `tabIndex` 落在两个 `div` 上（`attributes.tsx` 包它的、`CustomSelect` 的 Listbox 的根），它的按钮（`tabIndex` 0）不在弹窗的 Tab 循环里，只有用别的办法给它焦点时空格才打开它；
      - D2（控制者补做的核对，F-14）：`@nerve/ui` 的 `ContextMenu` 一直渲染它的项（透明、不接指针），不 `inert`，所以项目页上每张卡片的"Settings""Copy link"都是看不见的 Tab 停留点，创建弹窗的 Tab 循环经过它们；同一个组件在 M4、M6 的列表上；
      - 缺陷 i（F-15）：创建表单的正的 `tabIndex` 排在 Headless UI 焦点陷阱的哨兵（0）之前，Tab 在页头的关闭之后离开弹窗，走到后面的页面。F-13、F-14 的实现者在 P1 的 Tab 一圈中遇到它，控制者的探查把它与缺陷 ii 分开；
      - 功能一步（F-18）：创建之后的功能一步的关闭、打开项目是 `tabIndex` 1、2（各有一处 oxlint 的抑制），与缺陷 i 是同一个缺陷。它不是浏览器找到的：实现者在 F-17 的跟进中补全交接时，照 F-15 的机制推出（没有跑）；修好之后，P1 的检查证实了它（变异 `FW18.index` 把关闭的 `tabIndex` 改回 1，Tab 从关闭走到创建成功的提示和侧边栏）。
  - **代码本来对、只有评审或变异守着的性质**（当轮补上测试）：
    - 预检：创建表单页头的关闭按钮在途时禁用（PF6 → `T4.11`）；用自定义按钮的选择在 Tab 顺序里（PF15 → `T2.9`）；离开弹窗被拒绝时留着（PF19 → `T7.15`）；general 页的 `key`（`T5.7`，经 `moveWithinApp`）；
    - T3（评审的 I1–I3，裁定 T3-c–T3-e）：加入对话框在途时的忙和被拒绝时留着（去掉 `loading`，连按两下发两次加入；在 `await join` 之后关上，被拒绝时也关：两个变异都存活）；`decide` 的顺序只钉住一半（已归档挪到读取的错误之前存活：删除了的已归档项目显示已归档而不是找不到）；F-3 的测试钉不住共用的计数和已归档的列表；
    - T4（评审的 I1–I3，裁定 T4-a–T4-c）：换账户之后兑现的 vitest 答以 `undefined`，在会话的核对之外交回项目的变异存活（另一个账户的页面会进入功能一步）；负责人的候选去掉"成员关系有效"存活（nerve 对这样的负责人 422）；创建的按钮在途时的忙没有检查；
    - T5：general 页读 `currentProjectDetails` 而不是 `getProjectById(projectId)` 的探查 P5.1 存活，评审判断只由评审看住（T5-a）；修复轮的关键词规则后来看住它（第 2 节的变异记录）；
    - T6（自查，裁定 T6-a、T6-b）：自动归档把 `done` 交给自定义时长的弹窗的方式没有测试（P6.1：发出之后不论结果调 `done`，被拒绝时和另一个账户的页面上都关上弹窗）；自定义时长的提交按钮在途时的忙（P6.2）；
    - T7（自查，裁定 T7-a）：`roleChoices` 缺两行（工作区管理员对工作区的访客、项目管理员对是工作区成员的项目访客）；离开弹窗的"Leaving..."在途时的忙；成员页把移出的 `done` 交给确认框的方式；列把成员和调用者的工作区角色交给规则；角色列显示的名字退回英文 `ROLE`；P5 的 `toBeEnabled` 在弹窗的淡出中也通过（改用 `closedWithin`）；
    - T8（自查，裁定 T8-a；评审的 I1，裁定 T8-b）：添加按钮在途时的忙；负责人、默认负责人的修改在会话里跟进；"Add member"给谁；被拒绝时表单留着；换账户之后的回答答以 `undefined`。评审的 I1：谁能改成员的默认值没有钉住（测试的权限总是允许，项目成员和项目之外的工作区管理员看到可改的控件时什么都不失败，nerve 对他们 403）；
    - T9（自查，裁定 T9-a）：删除、归档的按钮在途时的忙；归档换账户之后的回答；已归档的卡片对谁提供恢复、删除，未归档的卡片的设置链接给谁；设置页对是工作区管理员的项目成员提供归档、删除；归档忙到跳转做完；删除只在项目自己的页面跳转；删除按钮在两段文字都对之前不可用。评审的 m1（搁置为 P10）：`card.test.tsx` 按 `shouldRender` 为真读菜单，真的 `ContextMenu` 显示每个 `shouldRender !== false` 的项；
    - T10（自查，裁定 T10-a；评审的 I1，裁定 T10-b）：标签栏的修改在调用时决定（PF-M2）；"更多"在设置取到之前的修改和它的"显示"；访客的标签菜单、工作区访客的拖动。评审的 I1：只在调用时决定一个字段的两种（默认的标签、"更多"中的标签）在 vitest 和 P8 上都存活；
    - 整分支评审：features、automations 页的门放宽到项目的成员时什么都不失败（I1，RV1）；`@nerve/utils` 的 `project.test.ts` 不在非空断言的范围（M1，RV2）；P9 的创建工作区换账户之后兑现的一行答以 `undefined`（搁置项 P5，Task 4 的修正轮报告的）。
  - **执行中删除的孤儿**：T3 中两种语言的顶层文案键 `link_copied`（它的读者只有本 Task 重写的两个侧边栏；`i18norphans.py` 因为 `common.link_copied` 含它而看不见，搁置项 P3）；修复轮删除 `fake-toast.ts` 的 `setPromiseToast` 替身和它的 `Outcome`（没有读者，T6-d、F-5），和两个侧边栏总是 `false` 的 `disableDrag`、`disableDrop`（F-4）；F-15 删去创建表单的正的 `tabIndex` 之后，`@nerve/constants` 的 `PROJECT_CREATE` 表和只为它传的三个 `isMobile` props 没有读者，一并删除；F-17 删除同一个文件中自 Plane 的导入起没有读者的 `CREATE_LABEL`（三行；knip 看不到枚举的成员和表的键）。
  - **测试的偶发**：Task 10 评审之前的一轮加的 P8 的访客 Gus 的拖动在负载下 30 次失败 3 次：他设默认之后的提示进来时文档在滚，按下和移动之间的滚动不开始拖动。修正轮让 `dragOnto` 等页面的有限的过渡结束再按下，Gus 核对他的提示（裁定 T10-d：根上的修法）。
  - 其余裁定都是测试的写法、说错的注释和文档（第 3 节）。跨 Task 的去重和收尾搁置到修复轮（P1–P11，`fix-wave-parked.md`）。
- **整分支评审**（opus，`4a4e7446..1cb735d9`，130 个文件，+5,650/−1,959）：
  - 第一次分派在写下任何东西之前停在接口的错误上（一个误报的安全标记 `[reasoning_extraction]`）；用 SendMessage 接着跑，立刻又停在同一个标记上，原来的上下文不能继续。控制者核对它的副本：`web/`、`e2e/` 与 HEAD 相同，没有留下评审的进程和容器。它在 HEAD 上跑过的门禁都通过：关键词 68 条，没有命中；turbo 54/54，web 349 等于上限；knip 空；vitest 16/16；e2e 105 个加预期的 S3（F4）。
  - 第二个 opus 评审子任务用同一份 brief，沿用那份副本和日志，逐节存下评审：结论"修复轮之后可以合并"，Critical 0、Important 1、Minor 6；另指出搁置项 P2、P4 说得太窄，P7 的范围更宽，P11 由 M6 解决，并照问划定 P2、P7、P11（第 4 节）。
- **修复**：一次分派，十一个提交（`9c3a4e10` 到 `28526539`）；变异的记录（F-8）在 `$M3TMP/p10tools`，不是提交。修复的报告之后，裁定 F-10 接受实现者写明的五处偏离。修复的复审（opus）通过：Critical 0、Important 0、Minor 2。同一个实现者用一个提交（`621c9390`）补上 F-10 的文档项和复审的两条（裁定 F-11）；它的报告把 router store 的其余读法分开之后，其中 29 个不传项目的项目级 `allowPermissions` 由只改 spec 的 `093d0d91` 写进各阶段的行（裁定 F-12）。两个提交的范围复审（sonnet）通过，没有发现（第 4 节）。
- **控制者的浏览器核对** C6–C8（M3 设计 9.7）在修复轮的头 `28526539` 的构建上做，三项都通过（第 2 节）；之后的两个提交只改文档和一处测试的说明。本记录的初稿指出 C7 没有核对标签式导航的项目选择、没有用键盘打开创建弹窗中网络的选择，C8 没有核对扩展侧边栏的复制。控制者用同一个构建补做：项目选择和扩展侧边栏的复制通过，网络的选择和创建弹窗的 Tab 找到 D1、D2（第 2 节）。
- **浏览器核对之后**（第 3、4 节）：
  - 裁定 F-13（D1）、F-14（D2），修复轮的实现者接着做。做完这两个修复之后它停下报告：P1 走一圈 Tab 留不在创建弹窗里，它归因于 `ModalCore`。控制者探查删除项目的弹窗（没有正的 `tabIndex`），分出两个缺陷：缺陷 i 只在创建表单里，裁定 F-15 删去它的正的 `tabIndex`；缺陷 ii 是每个经 `ModalCore` 的弹窗的，交给负责人（下一条）。
  - 五个提交（`d30199b2` 到 `9b3ff184`）之后，控制者在 `9b3ff184` 的构建上再做浏览器核对，通过（第 2 节）。
  - 这五个提交的复审（opus，`093d0d91..9b3ff184`）：Critical 0、Important 0、Minor 4，没有代码的缺陷。裁定 F-16（`FW13.root` 由评审看住）和 F-17（复审的四条，`456904f4`）。实现者在 F-17 的报告中指出功能一步仍有正的 `tabIndex`：裁定 F-18，`03157319` 修好，`3e9d0fe9` 改文档。`9b3ff184..3e9d0fe9` 的范围复审（sonnet）通过，没有发现。
- **唯一交给负责人的一项**（缺陷 ii）：每个经 `ModalCore` 的弹窗打开时，后面的页面对读屏等辅助技术仍然可见。它涉及 M2–M8 的每个弹窗，根本的修正是改写这个共用组件，是跨模块的，所以控制者没有自己定，交给负责人；P10 的合并等负责人的回答，本地的修复、文档、复审、持续集成和浏览器的再核对照常进行。负责人裁定（2026-10-10）"放进 M3 收尾"：改写 `ModalCore` 是 M3 收尾的一项独立的任务，P10 不等它合并（第 3 节）。
- **持续集成**：见第 2 节末。

## 2. 验收标准核对（spec 第 4 节，M3 设计 12 节 P10 的完成线）

| # | 标准 | 结果 |
|---|---|---|
| 1 | P1–P5、P8 的页面版本、S1、S2 和此前的全部故事通过：worktree 中 `make e2e` 106 个（`6c1a090a` 是 93 个）；`make test-web` 通过 | **通过。** 本地（裁定 E1）每个 Task 的实现和修正轮（只改文档、只改注释的除外）都是 plan 的数目：Task 1 94、Task 2 95、Task 3 98、Task 4 99、Task 5 101、Task 6 102、Task 7 103、Task 8 104、Task 9 105、Task 10 106；修正轮把检查加在已有的故事里，数目不变。修复轮的代码的头 `3c974513` 上 106 个，P1、P3、P4、P5、P8、S2 各 3 次；之后的跟进不改故事，没有跑 e2e。浏览器核对之后的修复：`b6344182`、`6db732a1` 和代码的头 `13388d2f` 上 106 个，P1、P2、P4 在 `6db732a1` 上各 3 次（27 个都通过），P1 在 `13388d2f` 上再 3 次；F-18 的 `03157319` 上 106 个，P1 3 次。web 的 vitest：`6c1a090a` 上 75 个文件、657 个；原型 87 个、761 个；Task 11 结束时（`1cb735d9`）93 个、820 个；修复轮之后 95 个、848 个；F-15 在成员下拉框的 vitest 加 1 个，之后 95 个、849 个（`@nerve/utils` 6 个文件、65 个）。整分支评审和第一次复审在各自的副本上都是 105 个加 S3（副本不是 git 仓库，F4）；第二次复审在它的副本上跑 P1、P2 各 3 次和 P4 一次（12、9、2 个都通过），没有跑全部的 e2e。持续集成见本节末。 |
| 2 | 打开表情选择器时没有外部请求、没有 CSP 违规（P1 的页面版本，`watchPage` 和 `requestsElsewhere`） | **通过。** P1 的页面版本打开图标的表情一页，`requestsElsewhere` 为空，控制台安静（`expectQuietConsole`）；P3 的页面版本也打开它。变异 `T1.1`（选择器读 frimousse 默认的 CDN）、`T1.2`（构建不写出数据）由 P1 发现，`T1.3`（读的版本不是构建写出的）由 `emojibase.test.ts` 和 P1 发现。C6 在浏览器中核对：数据从 nerve 读，没有发往别的源的请求，没有 CSP 违规。 |
| 3 | 9.5 中 P10 的 vitest 通过，每个都有发现它的变异；P10 写或改的每个测试至少被一个变异发现（W18） | **通过。** `delete-project-modal.test.tsx`（9.5"删除项目的组件"，`T9.1` 等）、`use-project-membership-changes.test.ts`（离开失败时不跳转：`T7.1` 和拒绝的一行）、`emojibase.test.ts`（`T1.3`）。W18 在修复轮的头上重算（`w18head.py`）：P10 写的 193 个 web 应用的测试和 `@nerve/utils` 的 6 行都至少被一个变异或探查让它失败（147 个由 W18 探查之外的变异，46 个只由 W18 探查）；记录中原有的变异都不让它们失败的 8 个测试，修复轮加 `W18.32`–`W18.38`，`W18.39` 是原型上手工的 utils 探查。F-15 在 `dropdowns/member/base.test.tsx` 加的第 194 个由 `T2.6`、`T2.8`、`T2.18` 和 `W18.31` 让它失败（194 个中 148 个由 W18 探查之外的变异，这样的变异 120 个）；`FW18.index` 不让任何 vitest 失败（vitest 不渲染功能一步），数目不变。 |
| 4 | 2.13 的每一处有一个"不经 `followInSession` 时失败"的 vitest | **通过。** 2.13 的 12 处各有换账户之后兑现的 vitest（`lateSettlings`）；W3 一类 17 个变异都由 vitest 发现（附录 A.2）。修正轮让交回东西的几处答以 nerve 的回答（`lateSettlingsAnswering`，裁定 T4-a：创建 `T4.12`、添加 `T8.16`、归档 `T9.16`）。整分支评审逐行对照代码：12 行"跟进的"与代码相同，各有它自己的测试；"不跟进的"组件状态的清单不全，修复轮补上加入对话框、包装层的加入界面和自定义时长（F-9）。 |
| 5 | 每个弹窗请求在路上时关不掉，被拒绝时留着，只在成功的跟进中关上（第 3 节第 6 条） | **通过。** 加入、创建、移出和离开的确认、离开、添加成员、自定义时长、删除、归档和恢复：在途时取消禁用、`ModalCore` 的 `handleClose` 为空，由故事扣住请求核对（`sentHeld`、`closedByEscape`、`enabledWithin`）；被拒绝时留着、按钮重新可用（`closedWithin`）。记录中"关得掉"一类 21 个（含组字的 Escape 4 个）都由端到端发现，"留着"一类 8 个都被发现。整分支评审逐个弹窗核对（第 4 节）。C6 在浏览器中核对离开的失败和唯一管理员的 409。 |
| 6 | 成员下拉框对 M4–M7 的调用方照旧：推迟渲染、手机上不取焦点、列表离窗口边缘 12 像素（`dropdowns/member/base.test.tsx`）；面包屑的下拉框一节一个 Tab 停留点（P2） | **通过。** `T2.10`–`T2.12` 由 `base.test.tsx` 发现，Task 2 的修正轮加 `T2.18`；`T2.14`（之前一节的标题是自己的 Tab 停留点）由 P2 发现。`T2.8`（成员下拉框丢掉调用方的 `tabIndex`）的 P1 一层随 F-15 通过（创建表单不再给负责人 `tabIndex`），现在由 `base.test.tsx` 的新测试发现（工作项的表单仍给它）。调用方给列表的 z-index 照旧有效（T2-a）只由评审看住：给它的调用方都在 M4 的页面上（附录 A.2）。C7 在浏览器中核对项目页头的面包屑一节只停一次 Tab。M4–M7 页面上的调用方由 spec 第 5 节的 M4、M6、M7 行带去核对。 |
| 7 | P2 的页面版本：不是成员的人直接打开公开项目的地址，只请求项目本身，没有失败的请求；已归档的项目同样（S2） | **通过。** P2 的两个页面版本读 `mounts.ts` 中同一份 `PROJECT_MEMBER`；S2 加"项目的管理员打开已归档的项目"（`APP`、`WORKSPACE`、`PROJECT`）。`T3.6`、`T3.7` 由 vitest 和 P2、S2 发现。整分支评审逐页、逐个角色核对挂载时的请求，与 2.12 和 S2 的 `REQUESTS` 相同。 |
| 8 | `node tools/keywords.mjs` 通过：69 条规则、3 个例外，没有命中；`tsc`、knip 通过 | **通过**（`3c974513`、`621c9390` 的 `make lint-web`、`make knip`，`093d0d91` 的 `make lint-web`；之后浏览器核对之后的修复的头（代码 `13388d2f`，文档 `9b3ff184`）、`456904f4`、`03157319` 的 `make lint-web`、`make knip`，`3e9d0fe9` 的文档之后再跑 `make lint-web`）。P10 加 `project-leave`（Task 7，`T7.13` 由它发现）和 `project-address-route`（修复轮，改回读 router store 的 `FW3` 各行由它发现）。 |
| 9 | P10 的页面从路由参数读地址（`project-address-route`）；features、automations 页的门只给 nerve 允许修改项目的人看开关，门放宽时 `admin-pages.test.tsx` 失败 | **通过。** 修复轮（F-2）：七处读路由参数，`allowPermissions` 明写工作区和项目，规则看住 11 个文件。`admin-pages.test.tsx`（25 个）经 React Router 的 `createRoutesStub` 渲染五页，权限和项目是 store 自己的，router store 指着工作区、没有项目；评审的 RV1（`FW3.rv1`）、三个 `FW3.wide-*` 和每页读 router store 的变异都被发现。复审在自己的副本上重跑 `FW3.rv1`、`FW3.list-gate`、`FW3.store`，结果相同。 |
| 10 | 输入法组字时的 Escape 不关上创建表单、`ModalCore` 的弹窗和表情选择器（P1、P4） | **通过**（Chromium 的事件顺序）。`FW4.keypress`、`FW4.picker`、`FW4.modal-capture` 由 P1 发现，`FW4.modal` 由 P4 发现。C6 在浏览器中对离开弹窗和删除弹窗的名称框发出同样的组字事件，弹窗都留着。Safari 先结束组字，只查 `isComposing` 的守卫在那里拦不住（第 7 节）。 |
| 11 | 创建项目的弹窗里 Tab 按表单的顺序走一圈、不离开弹窗，网络的选择停在它的按钮上，创建之后的功能一步也是；关着的右键菜单不是 Tab 停留点（P1、P2，裁定 F-13–F-15、F-18） | **通过。** P1 用 `tabCycle`（`e2e/fixtures/project-pages.ts`：按 Tab 直到焦点回到起点，最多 40 次，不在弹窗里的一站记作"outside:"）从名称起走一圈，后面有 Ops 的卡片：标识、说明、Public（网络的按钮）、Lead、Cancel、Create project、Close、Project icon、名称，每一站都在弹窗里；从说明按 Tab 到网络的按钮，空格打开；从 Cancel 按 Shift+Tab 到负责人。创建之后从功能一步的 Close 走一圈，都在 `[aria-modal=true]` 里（成功的提示是不模态的 dialog）：Open project、四个功能的开关、Close。P2 右键 Web 的卡片，经打开的、唯一不 `inert` 的菜单复制链接。变异 `FW13.wrapper`、`FW15.index`、`FW18.index` 由 P1 发现，`FW14.inert`、`FW14.open` 由 P2 发现；`FW13.root` 存活，由评审看住（F-16，见下面的变异记录）。第二次复审读过这两步：失败的理由对。控制者在 `9b3ff184` 的构建上再核对（见下）。 |
| 12 | 根目录 `.oxlintrc.json` 的范围（2.14）：P10 新建的模块和改到的 nerve 测试在 `no-non-null-assertion` 的范围里；每一组各有一个变异让 `check:lint` 失败 | **通过。** W12 一类 12 个变异都由静态检查发现：每个 Task 的一组，加修复轮的 `FW7.rv2`（`@nerve/utils` 的 `project.test.ts`，评审的 M1）和 `FW7.toast`（修复轮改到的 `fake-toast.ts`）。复审另写三个（`RR.*`：修复轮的三个新文件中各一个 `!`，含路径中转义了方括号的 `admin-pages.test.tsx`），web 的 `check:lint` 都失败。 |
| 13 | 改到的文件按 7.9 没有 oxlint 警告：web 的 120 个 TS 文件（删除的一个不算）都是 0 条；web 的上限 356 → 348，`propel` 16 → 11，`ui` 19 → 15；没有新的抑制 | **通过。** 修复轮的头上 116 个文件 0 条；第一次复审另核对 e2e 的 19 个 TS 文件，也是 0 条。浏览器核对之后的修复另手改 3 个（`@nerve/ui` 的 `custom-select.tsx`、`context-menu/root.tsx`，`@nerve/constants` 的 `tab-indices.ts`），实现者报告 119 个都是 0 条，第二次复审核对这一轮改的 13 个 TS 文件也是 0 条；F-18 另手改 `project-feature-update.tsx`，共 120 个（spec A.5）。上限 356 → 350（Task 2，`dropdowns/member/` 的六条随旧写法删除）→ 349（Task 7，确认框的正的 `tabIndex`）→ 348（修复轮，`member-select.tsx` 重复的 `options &&` 随它读路由参数删除）；`propel` 16 → 11（Task 1）；`ui` 19 → 15（F-14：`context-menu/root.tsx` 原有的 3 条 `exhaustive-deps` 和 1 条 `jsx-no-constructed-context-values` 随 `useCallback`、`useMemo` 清零，第二次复审在 BASE 的文件上核对过这 4 条）；其余各包不变。抑制 185 → 183（Task 5、Task 7 各删一处）→ 181（F-18 删去功能一步的两处 `jsx-a11y/tabindex-no-positive`；范围复审用 `suppressions.py` 的写法核对），没有新的。 |
| 14 | 没有新的 `as`、`any`、`!` | **通过。** 原型上 `castsall.py` 核对整个 Phase 的 99 个 TS 文件，`w12.cjs` 核对 61 个源文件（`as` 3 → 1、`any` 14 → 10、`!` 0）；各 Task 的评审核对它的提交（Task 3、4、10 的评审写明没有新的）；修复轮加的行由 castsall 的写法和复审核对：没有新的 `as`、`any`、`!` 或抑制；浏览器核对之后和复审之后加的行（`28526539` 到 `13388d2f`、`13388d2f` 到 `03157319`）同样没有（附录 A.7）。`6db732a1` 中 `tabCycle` 的一个 `as const` 由 `13388d2f` 去掉（第二次复审核对）。 |
| 15 | 逐 Task 复现 | **通过（修订时）。** 修订 `4a4e7446` 时从 `6c1a090a` 的新副本照 plan 重放：每个 Task 之后门禁通过（e2e 93…105 个，加 S3 的 F4），最终与原型逐文件相同，只差四份文档（spec A.10）。各 Task 的修正轮、修复轮和浏览器核对之后的几轮照裁定偏离原型，没有再复现；plan 中受影响的块见第 5 节。 |
| 16 | 控制者的浏览器核对 C6–C8 写进评审（9.7），含 `webui` 对表情数据的 `HEAD` 的回答（2.1） | **通过**，见下：C6–C8 通过；补充的核对找到 D1、D2，由 F-13–F-15 修好，修好之后的再核对通过。 |

**控制者的浏览器核对 C6–C8、补充的核对和修好之后的再核对**（9.7，`$M3TMP/p10-browser-checks.md`，截图在 `$M3TMP/p10browser/shots`）：
- **准备**：
  - 构建：`28526539` 的 `git archive` 副本，`make build`（`bin/nerve`，内嵌 web）。之后的 `621c9390`、`093d0d91` 只改文档和一处测试的说明，生产代码相同。补充的核对用同一个构建；修好之后的再核对用 `9b3ff184` 的构建（`$M3TMP/p10browser/src2`）。
  - 数据库：一个临时的 `postgres:18.6` 容器（`p10-browser-db`），核对时起，之后停掉并删除，没有碰别的容器。nerve（`NERVE_ENV=dev`）在 `127.0.0.1:8095` 上做 C6、C7；C8 另起一个 nerve，在局域网地址 `10.10.20.104:8096` 上，用同一个库；之后两个 nerve 停掉，截图移到 `$M3TMP/p10browser/shots/playwright-mcp`。补充的核对另起一个临时的 `p10-browser-db`，nerve 在 `10.10.20.104:8096` 上；再核对用这个库，nerve 在 `127.0.0.1:8095` 上；之后库和 nerve 都停掉并删除，截图移到 `$M3TMP/p10browser/shots/playwright-mcp-2`。worktree 中没有留下浏览器的文件。
  - 数据经接口建（`p10browser/seed.mjs`）：acme（alice 管理员、bob 成员、carol 访客、dan 成员、erin 管理员，frank 一个待接受的邀请）；Web（公开；alice 管理员、bob 成员、carol 访客）、Ops（公开，只有 alice）、Old（私密，alice）。
  - Playwright MCP，Chromium，1280×900。页面的 CSP 是 nerve 给的（`connect-src 'self'`），从 jsDelivr 取数据会是违规。
- **C6 项目：通过。**
  - 创建（alice）：表情选择器从 nerve 读 `/assets/emojibase/15.3.2/en/data.json` 和 `messages.json`（200），没有发往别的源的请求（Resource Timing 和请求清单都没有），控制台没有 CSP 违规和错误（只有 Plane 的 Canvas2D `willReadFrequently` 警告），搜"rocket"找到并选中；图标一页列出图标，选中房子。负责人只用键盘：从说明按 Tab 经过网络的选择的两个停留点（都是 `div`，到不了它的按钮：补充的核对把它记作 D1，见下）到负责人，Enter 打开，列表在按钮旁（面板左边对齐按钮、上边在按钮下边），搜索框有焦点，"bob"之后 Enter 选中、列表关上。`POST …/projects` 201，进入功能一步，打开 Cycles 发 `PATCH` 200；nerve 中是 Launch / LAUNCH、公开、负责人 bob、`logo_props` 是房子的图标、Cycles 开着，侧边栏显示带房子的 Launch。
  - 表情数据的 `HEAD`（spec 2.1，设计 7.7"P10 核对并写进 review"）：`HEAD /assets/emojibase/15.3.2/en/data.json` 答 200，`Cache-Control: public, max-age=31536000, immutable`，`Content-Length: 700120`，没有 `ETag`、没有 `Last-Modified`，与 spec 记的相同；`ETag` 归收尾（第 6 节）。
  - 加入（dan，工作区成员，不在 Ops、Launch）：项目列表中 Ops、Launch 有"Join"，私密的 Old 不在。项目页的加入界面"Join project"发 `POST …/join` 200，页面显示项目；卡片的 Join 打开对话框，加入 200 之后对话框关上，去 Ops 的工作项页。
  - 离开：
    - dan 从成员页自己一行（菜单只有"Leave"）：控制者让离开两次答 500 的 problem，提示"Something went wrong"，弹窗留着、"Leave"可用，页面停在成员页；真的离开 204 之后去 `/acme/projects`，Ops 从侧边栏消失。
    - 访客 carol 从侧边栏离开（Web 的菜单有"Copy link""Settings""Leave project"）：名称框中依次发出组字开始、组字中的 Escape、组字结束，弹窗留着（F-3）；两个字段填好之后 204，去 `/acme/projects`。
    - Ops 唯一的管理员 alice 从自己一行离开：409 `project.sole_admin`，提示给出 nerve 的原因，弹窗和页面留着。
  - 归档、恢复、删除（alice）：在 Old 的 general 设置归档 200，去 `/acme/projects`，Old 从列表和侧边栏消失；已归档的项目页上 Old 的卡片有"Archived""Restore""Delete"，恢复 200，Old 回到侧边栏；在 Old 的设置删除，名称框中组字的 Escape 不关弹窗，两段都填好之后 `DELETE` 204；Ops 在设置归档之后从已归档的卡片删除 204，页面停在已归档的项目页（"No projects archived"）。未归档的卡片的右键菜单只有设置和复制链接。
  - 成员页（alice，Web）：负责人只用键盘（Enter 打开，列表在按钮旁，搜索框有焦点，候选是 alice、bob 和"None"，访客 carol 不在），选 bob 发 `PATCH` 200；默认负责人同样的候选，选 alice 200；nerve 中负责人 bob、默认负责人 alice。bob 的角色选择 Tab 能到，空格打开（Guest、Member、Admin），选 Guest、再选 Member 各 200。添加成员的弹窗 Tab 顺序是"Select co-worker"（一个停留点）、角色、"Add more"、取消；候选是 dan 和 erin（不在 Web 的工作区成员），dan 默认 Member，工作区管理员 erin 只能是 Admin（`addableRoles`）；添加之后列表有他们。
  - general 页（alice，Web）：Tab 顺序名称 → 说明 → 标识 → 可见性（一个停留点）→ 时区（一个停留点）→ 更新 → 归档 → 删除；可见性空格打开，选 Private；时区 Enter 打开、搜索框有焦点，"Shanghai"之后 Enter 选"UTC+08:00 Beijing"（M2 交接第 11 节的页面一侧）；`PATCH` 200，nerve 中 network 0、时区 `Asia/Shanghai`。
  - automations 页（alice，Web）：空格拨动自动归档的开关 200；月数的选择 Tab 能到，空格打开，选"3 months"200。
- **C7 下拉框（键盘）：当时核对的都通过**（补充的核对另找到 D1、D2，见下）。
  - `CustomSearchSelect` 在 M3 的调用方：创建表单的负责人（`MemberDropdownBase`）和成员页的负责人、默认负责人（`member-select`），同 C6；添加成员弹窗的成员选择（一个 Tab 停留点，Enter 打开，输入筛选，Enter 选中）；general 页和个人设置（`/settings/profile/preferences`）的时区（Enter 打开、搜索框有焦点，114 个时区，Escape 关上、焦点回到按钮）；项目页头面包屑的项目一节（`navigation-search-dropdown`：在侧边栏和"Work Items"一节之间只停一次 Tab；Enter 打开，列表在按钮旁，面板 x 281 对按钮 x 280.6、y 75 对按钮下边 74.5；搜索框有焦点，选项是 Launch、Old、Ops、Web；"Ops"之后 Enter 去 Ops 的工作项页）。
  - `CustomSelect` 在 M3 页面上的调用方各自 Tab 能到，空格（或下箭头）打开，Escape 关上、焦点回到按钮：项目的可见性（general）、项目成员的角色、添加成员的角色、自动归档的月数、工作区成员的角色（4 个）和邀请的角色（成员页）、邀请弹窗的角色（Escape 只关列表，弹窗留着：它的 `ModalCore` 没有 `handleClose`，P9）、组织规模（工作区 general、`/create-workspace`）、个人设置的主题、语言和一周的第一天。
  - 当时没有核对的两个 M3 调用方：标签式导航的项目页头的项目选择（`navigation/project-header.tsx`；C8 打开了标签式导航，只核对复制），创建表单的网络的选择（`projects/create/attributes.tsx`；C6 中 Tab 经过它，没有用键盘打开）。补充的核对见下。
  - M4–M7 页面上的调用方（成员下拉框的 12 个文件、4 个筛选、迭代、模块、视图的页头）：它们的页面还不能运行（工作项页到 M4 之前是 404），没有核对；spec 第 5 节的 M4、M6、M7 行带着这项核对（第 6 节）。
- **C8 局域网 HTTP 地址上的复制：通过。**
  - 那里的页面不是安全的上下文（`isSecureContext` 为假，`navigator.clipboard` 是 `undefined`），纯 HTTP 下能登录。复制经 `@nerve/utils` 的 `copyTextToClipboard` 退到 textarea 和 `document.execCommand("copy")`，`useCopyLink` 照它的结果提示。控制者包住 `execCommand`，记下复制了什么、它答什么，再让它答 `false`，代替拒绝的浏览器。
  - 项目卡片的"Copy link"（`/acme/projects`）：复制 `http://10.10.20.104:8096/acme/projects/{web}/issues`，提示"Link copied! Project link copied to clipboard"；拒绝时提示"Error! Something went wrong. Please try again."。
  - 侧边栏的项目菜单的"Copy link"：同样的链接和提示（没有试拒绝）。
  - 邀请的菜单的"Copy link"（工作区成员页，frank）：复制 `http://10.10.20.104:8096/workspace-invitations?invitation_id=…&token=nrv_inv_…`，提示"Link copied! Invite link copied to clipboard"；拒绝时提示出错。
  - 项目标签栏"⋯"的"Copy link"（在"Project navigation"对话框中把导航改为标签式，单选在 nerve 回答之后才变；工作项的标签）：复制工作项页的链接，提示同卡片；拒绝时提示出错。
- **整个核对中的请求和控制台**：API 请求都到 nerve 的 `/api/v0`，只有 M4 的 `user-properties`（工作项页上 404，2.12）；没有发往别的源的请求；没有 CSP 违规；控制台的错误只有那个 404 和故意的拒绝，警告只有 Plane 的 Canvas2D `willReadFrequently`。
- 核对中看到的、不是 P10 缺陷的几处进第 6 节（收尾）和第 7 节。
- **补充的核对**（本记录的初稿指出的漏项）：
  - 标签式导航的项目页头的项目选择（`navigation/project-header.tsx`）：一个 Tab 停留点；Enter 打开，列表在按钮旁（面板 x 287 对按钮 x 286.6，y 81 对按钮下边 80.5），搜索框有焦点；选项是 Old、Ops、Web，"Ops"之后 Enter 去 Ops 的工作项页。**通过。**
  - 扩展侧边栏的复制：把"Show limited projects on sidebar"设为 1，"More"打开扩展侧边栏，Web 的菜单有"Copy link""Archives""Settings"；在局域网的 HTTP 地址上（`execCommand` 的退路）复制 `http://10.10.20.104:8096/acme/projects/{web}/issues`，提示"Link copied! Project link copied to clipboard"。**通过。**
  - 创建弹窗中网络的选择，用键盘：**不通过（D1）**。有了焦点之后它能用：空格打开 Private、Public，上箭头、Enter 选 Private，Escape 关上列表、弹窗留着。可 Tab 到不了它：弹窗的 Tab 循环是 Close[8] →（离开弹窗：Settings、Copy link ×3）→ 名称[1] → 标识[2] → 说明[3] → DIV[4] → DIV[4] → Lead[5] → Cancel[6] → Create[7] → Close[8]。正的 `tabIndex` 在包它的 `div`（`attributes.tsx:45`）和 `CustomSelect` 的 Listbox 的根 `div`（`custom-select.tsx:50`）上，按钮是 0，从不在循环里。
  - 焦点离开创建弹窗：**不通过（D2）**。`ContextMenu` 把每张项目卡片的菜单项一直渲染在一个看不见的固定层里（`opacity-0 pointer-events-none`，不 `inert`），每张卡片的"Settings""Copy link"都是项目页上的 Tab 停留点，创建弹窗的 Tab 循环经过它们。
  - 另找到：经 `ModalCore` 的弹窗打开时，没有一个 `body` 的子元素 `inert` 或 `aria-hidden`。探查的是删除项目的弹窗（没有正的 `tabIndex`）：焦点陷阱守住 Tab（名称 → 确认 → 取消 → 名称），可应用的根、`#context-menu-portal` 和其余的都没有被隐藏。这是缺陷 ii（第 3 节）。
- **修好之后的再核对**（`9b3ff184` 的构建，裁定 F-13–F-15）：**通过。**
  - 项目页上三张卡片关着的右键菜单（`#context-menu-portal > *`）都是 `inert`。
  - 创建弹窗从名称起的 Tab 循环：标识 → 说明 → Public（网络的选择的按钮）→ Lead → Cancel → Create project → Close → Project icon → 名称 → …，每一站都在弹窗里（D1、D2）。
  - 网络的选择：在"Public"上按空格打开 Private、Public，Escape 关上列表、弹窗留着。
  - 右键 Web 的卡片：打开的菜单是唯一不 `inert` 的，有 Settings、Copy link；下箭头、Enter 复制链接，提示"Link copied! Project link copied to clipboard"；菜单关上，三个又都是 `inert`。
  - F-18（`03157319`）在这次再核对之后：功能一步的 Tab 由 P1 核对（上表第 11 行），没有再在浏览器中核对。

**变异的记录**（裁定 F-8，`$M3TMP/p10tools/p10-mutation-record.md`，表 `mutants_p10_record.py`，日志 `$M3TMP/p10mut/fix/logs/`）：
- 一张表收进：plan 的表和各 Task 的修正轮加的行（`mutants_p10.py`）、预检和修订的（`mutants_pf_amend.py`）、各 Task 的探查（`p10mut/t*/probes_*.py`；Task 4 的修正轮的两个原是一个脚本，成了 `P4f.*` 两行）、W18 的探查、修复轮自己的（`mutants_fw_p10.py`）、浏览器核对之后的（`mutants_p10_record.py` 的 `F13_F15`）和复审之后的（同一文件的 `F18`）；`mutants_zh_p10.py` 只是 plan 的行的中文说明，没有变异，记录把它写在每个改坏之下。
- 289 行：35 个是同一组改动的第二个名字（31 个是修正轮把探查收进 plan 的表；`PF6`、`PF15` 是预检的行，与 `T4.11`、`T2.9` 相同；`P9.14` 与 plan 原有的 `T9.11` 相同；`W18.23` 与 `P8.10` 相同），同样的改动只跑一次，两个名字写的层都跑；3 个不适用：`T10.12`（和它的第二个名字 `P10.8`），F-4 删除了它要改的开关，没有可改的地方；`T4.10`（负责人不在创建表单的 Tab 顺序里），F-15 删去了创建表单的每个正的 `tabIndex`，它要改的那一行没有了。行被之后的改动挪过的变异写两种版本的改动（`P8.15`；F-15 之后的 `T4.5`、`T4.11`、`T4.14`）。
- 在修复轮的代码的副本上（第 7 步之后的 `6021048b`；`FW7.toast` 和 `P5.1` 加的关键词一层在 `eee6415c` 上，两者只差 `.oxlintrc.json`）跑了 247 个：247 个被发现，没有等价的，没有不能跑的。浏览器核对之后，`FW14.*` 在 `b6344182` 的副本上跑，`FW13.*`、`FW15.index` 和 F-15 改了行或层的 `T4.5`、`T4.11`、`T4.14`、`T2.8` 在 `6db732a1` 的副本上跑；F-18 的 `FW18.index` 在 `03157319` 的副本上跑。**252 个中 251 个被发现**，没有不能跑的。`FW13.root`（`CustomSelect` 把 `tabIndex` 给 Listbox 的根）存活：自 F-15 起 P10 的页面上没有给 `CustomSelect` `tabIndex` 的，唯一给的是 M6 的 `ModuleStatusSelect`；F-15 之前它让 P1 失败（`b6344182` 上）。它由评审看住（裁定 F-16，第 3 节）。
- 按发现它的层：只有 vitest 的 130 个、只有端到端的 71 个、vitest 和端到端的 22 个、只有静态的 19 个、静态和 vitest 的 9 个（静态 28 个、vitest 161 个、端到端 93 个；一个变异可以算在几层）。按来源：plan 和修正轮的 151 个（`T4.10` 不适用）、预检和修订的 5 个、各 Task 的探查 23 个、W18 的探查 38 个、修复轮的 29 个、浏览器核对之后的 5 个（`FW13.root`、`FW13.wrapper`、`FW14.inert`、`FW14.open`、`FW15.index`）、复审之后的 1 个（`FW18.index`）。29 个变异有一层没有发现它、另一层发现了，理由逐个写在记录里：22 个是探查当时写下、让它通过的那一层（修正轮在另一层加了发现它的检查）；`T2.8` 的 P1 一层随 F-15 通过，`base.test.tsx` 发现它；其余 6 个（`P5.1`、`P6.4`、`P8.2`、`P8.10`、`P10.2`、`P10.5`）各写明自己的理由（例如 `P8.10`：P5 通过，`project-roles` 的 vitest 发现它）。
- W18 在修复轮的头上（上表第 3 行）：P10 写的 193 个测试和 `@nerve/utils` 的 6 行都被某个变异或探查让它失败；F-15 加的第 194 个同样（上表第 3 行）。
- `e2ekills.py` 读过每个算作发现的端到端和 vitest 层的日志：两次端到端的"发现"是 testcontainers 没能绑定端口（`P8.10`、`FW4.keypress`），另有几个变异同类的第二层盖掉了第一层的日志。`mut.py` 改为每层一个日志，这 9 个重跑（`T2.9`、`T6.8`、`T7.22`、`T7.23`、`T9.21`、`P8.14`、`P8.10`、`FW4.keypress`、`P5.1`）：`FW4.keypress` 由 P1 的检查发现，`P8.10` 的 P5 一层存活、由它的 vitest 发现。每个发现现在都是一个故事或一个测试的失败。
- 第一次复审抽查 15 行的日志，在自己的副本上重跑 `FW3.rv1`、`FW3.list-gate`、`FW3.store`、`FW6.taken`、`FW4.modal`、`FW4.picker`，都由记录写的检查发现；`recordclasses.py` 重算出 spec A.2 的类别表，逐格相同。结论：可信。
- 第二次复审读过 `FW15.index`、`FW13.wrapper`（P1:356：Tab 离开弹窗走到侧边栏；包它的 `div` 多出一站）、`FW14.inert`（P2:227：两个不 `inert` 的菜单都有"Copy link"）、`FW14.open`（P2:227：没有不 `inert` 的菜单）、`FW13.root`（`6db732a1` 上存活，`b6344182` 上被发现）和 `T2.8`（`base.test.tsx`）的日志：都因为对的理由失败。范围复审核对 `FW18.index` 在 P1:313 失败，理由对（Tab 离开了弹窗）。

**持续集成**：分支上每次推送都跑 server、web、e2e 三个任务。下面的运行除一次在 e2e 中失败外都通过；其中三次在 GitHub 接口的速率限制之下，从运行的页面上读结果（标"页面"）：
- Task 1（`247529f5`）：run 38008426885；
- Task 1 的修正轮（`80584941`）：run 38009068791；
- Task 2（`209e9bf6`）：run 38009809132 **在 e2e 中失败**（server、web 通过；日志和产物要 GitHub 的认证，这台机器没有，注解只说"exit code 2"；裁定 T2-f）；
- Task 2 的修正轮（`86fe7e11`）：run 38012959310（页面），只在持续集成上的 P1 失败不再出现；
- Task 3（`15f21247`）：run 38013789852；
- Task 3 的修正轮（`3fee585b`）：run 38015331988；
- Task 4（`c3772fa3`）：run 38015935071；
- Task 4 的修正轮（`8ac39a87`）、控制者的 `46bfd390` 和 Task 5（`953c93c9`）：run 38017750336（页面）；控制者停了对 `8ac39a87` 的运行的等待，之后两次推送取代它（接口的速率限制是共用的）；
- Task 6 和它评审之前的一轮（`a1a3cbe5`、`6a028754`）：run 38019112648；
- Task 7 和它评审之前的一轮（`b5ff6a4b`、`b05b42bb`）：run 38021042359；
- Task 8 和它评审之前的一轮（`4079ca22`、`45736939`）：run 38029158560；
- Task 8 的修正轮（`3ff60245`）：run 38029909096；
- Task 9 和它评审之前的一轮（`5781ac46`、`fb38b1a9`）：run 38031549690（页面）；
- Task 10 和它评审之前的一轮（`108585c1`、`cc1ee856`）：run 38033413830；
- Task 10 的修正轮（`bd151a8a`）：run 38036416797；
- Task 11（`213e7b3a`）：run 38036997528；
- Task 11 的修正轮（`1cb735d9`）：run 38037684544；
- 修复轮（十一个提交，头 `28526539`）：run 38049191447；
- 两个跟进提交（`621c9390`、`093d0d91` 一起推送）：run 38051733219；
- 浏览器核对之后的修复（`d30199b2` 到 `9b3ff184`，头 `9b3ff184`）：run 38104491391；
- 复审之后的跟进（`456904f4`、`03157319`、`3e9d0fe9`，头 `3e9d0fe9`）：run 38106559712。

本地的 `make e2e` 每个 Task 都跑（裁定 E1：Docker Desktop 起着），持续集成是第二道。本提交推送之后，分支上再跑一次；合并之后 main 的持续集成在合并时记录。

## 3. 执行中的决定

这一节列控制者在本阶段做的裁定，每条写明理由和裁定错了的代价。完整的记录在执行台账中。

**执行前**
- 架构子任务的 spec 第 3 节，控制者的裁定 R1–R7（`$M3TMP/p10-architect-rulings.md`）：
  - **R1**（15 个任务 → 11 个 Task，接受）：设计的每个任务都有落点（spec 第 3 节第 1 条），页面版本与它的页面同一个提交，每个 Task 之后 e2e 都通过；最长的 Task 3 是 1,473 行，在上限之内（P7、P8、P9 的先例）。代价：太大的 Task 由它的修正轮拆。
  - **R2**（成员下拉框以重写的 `CustomSearchSelect` 为底，`member-options.tsx` 删除，接受）：带搜索的选择的第二份实现（自己的 `Combobox`、自己的打开状态）是负责人的标准不许的重复；`CustomSearchSelect` 现在做到的正是 7.7 要 `member-options` 做的。设计 7.7、9.7 C7 等处的说法随之改，Task 11 的 M2 交接第 14 节的结果写明删除和理由。代价：M3 之内没有；M4–M6 的页面若靠 `member-options` 自己的怪处，在那个 M 的页面任务中显出（预检的 M5 把这一条看得更确切，约定在修订中恢复）。
  - **R3**（`dropdowns/` 的嵌套按钮留着，交给 M4）：改 `DropdownButton` 会改到 M4–M6 页面上的每个下拉框，不是 P10 的；P10 自己的添加成员弹窗不再嵌套。spec 第 5 节 M4 一行写明 HTML 的理由和"M4 随工作项的属性下拉框一起改它"。代价：到 M4 之前这些下拉框的标记不合法，键盘经里面的按钮能用。
  - **R4**（F-3：`getProjectById` 给出 nerve 最后回答的那一份，经 `reconciled.ts` 的 `answeredAt`，一个只增的模块级计数，接受；总体设计 7.7 加一句）：确认的修改已写进每一份，几份只在两次读取之间 nerve 那里变了的地方不同，最后回答的是 web 知道的最好的；丢掉自己的读取会丢掉不是成员的人对公开项目唯一的一份，`updated_at` 不随 `member_role` 变，而这正是 F-3 自己的例子。这是给之后每个 M 的数据层规则（M4 的工作项也会有列表中的一份和自己的读取），照 PF-M2、`followInSession` 的先例进 7.7。代价：nerve 先读、回答却后到的一份在两次读取之间胜出，下一次取数纠正。
  - **R5**（`@nerve/propel` 的 `emoji-icon-picker/index.ts` 多一行导出，接受）：9.5 的守卫（`emojibase.test.ts`）从那里读 `EMOJIBASE_URL`；P9 的裁定 P6 的先例。代价：没有。
  - **R6**（被拒绝的离开在共用的拒绝提示里说原因，弹窗留着，接受；设计 7.6 的说法随之改）：离开的三个入口经一个 hook（规则 `project-leave`），拒绝经 `useRefusalToast`（规则 `refusal-toast`）；停用弹窗把原因写在弹窗里，因为它只有一个入口。代价：原因在提示里，不在弹窗里；P5 核对它显示。预检发现成员页的确认框不是这样（M2），由 A-M2 改为每个弹窗一条规则。
  - **R7**（general 页的 `key={project.id}`：取 (b)，加一个经命令面板的 P3 测试）：没有 `key` 时项目 B 的设置页显示 A 的值，提交把 A 的值写进 B，是写错实体，约 15 行的测试守得住；"只有评审看得出"正是 brief 说的缺口。代价：约 15 行和一个端到端测试。预检证明这样写不出（M1），由 A-M1 改为同一路由内的移动。
  - 架构子任务自己的决定（第 3 节第 8–16 条，含把 S2 的列表移进 `e2e/fixtures/mounts.ts`，S2 的 `REQUESTS` 仍是唯一的清单）照写接受；P10 页面之外的英文 `ROLE` 归收尾的角色名一项（P9 的 P6）；表情数据没有 `ETag` 是 Go 的改动，归收尾。架构子任务的规则失误（`cd` 与别的命令组合三次、`git -C` 一次，都只读）记下，之后的分派重申规则。
- 预检（高 0、中 5、低 10）和修订 `4a4e7446`（第 1 节）：控制者的裁定都接受（`$M3TMP/p10-amend-brief.md`；除下面写明的，都取预检给的修法）：
  - **A-M1**（R7 照写做不到）：取 (b′)：P3 的页面版本在同一个路由里移动（`history.pushState` + `popstate`，Web 的读取已缓存），去掉 `key` 的 `T5.7` 由 P3 发现；spec 第 3 节第 7 条、第 6 节、A.2 照预检的文字改（应用中今天没有这条路，这一步代替之后加在设置页上的第一个项目切换）。理由同 R7。代价：约 20 行，P3 保持约 400 行。
  - **A-M2**（被拒绝的离开）：每个弹窗一条规则：被拒绝时留着，只在成功的跟进中关上（取预检的"代码的一种"）。成员页的确认框（移出和离开）经 `followInSession` 在成功的跟进中关上；P5 的被拒绝的离开核对确认框还开着；加预检的侧边栏离开弹窗被拒绝的一步（`T7.15`）和确认框自己关上的变异（`T7.17`）；2.13 删去确认框的例外；7.6 改为"失败时留在原页，弹窗留着，原因在共用的拒绝提示里；成功之后才关上、跳转"。理由：离开的三个入口和移出应当一样，一个被拒绝时关上、旁边的却留着的弹窗，正是 F-1、F-2 的规则要防的不一致。代价：修订中改几处代码和测试。
  - **A-M3**、**A-M4**：P1 核对页头的关闭在途时禁用（`T4.11`）；P5 用 Tab 到添加弹窗的成员选择，Enter 打开、搜索框有焦点（`T2.9`）。代价：各几行。
  - **A-M5**（成员下拉框对 M4–M6 的行为）：恢复调用方看得到的约定，不只是列出不同。改写只许改 7.7 要的（Tab 能到、列表在按钮旁、桌面上搜索框有焦点、Escape 只关列表），其余照旧：`renderByDefault` 推迟渲染、按钮的点击不冒泡、手机上不取焦点、popper 的 `preventOverflow` 12 像素；各放在它该在的地方、不重复，每一项有一个会失败的测试，测不了的写进 A.2 并写明理由；按键（第 5 项）和 L8 写进第 3 节第 15 条和第 5 节 M4 一行。理由：P10 不能不打招呼地改 M4–M6 的页面：点击冒泡进工作项的链接、桌面上每一行都挂一个 popper，会成为 M4 没要的回退。代价：`CustomSearchSelect` 两个新 props（`focusSearchOnOpen`、`popperModifiers`）和一个 vitest。
  - **L1–L10**：L1 设计 9.7 C7、13.1 改为"成员下拉框以 `CustomSearchSelect` 为底，列表在按钮旁"，调用方的数取预检的，Task 11 在 M2 交接第 14 节的结果中把关闭条件的 `member-options` 一项改为成员下拉框，`docs/v0` 中其余写 `member-options` 的逐个查；L2 总体设计 7.7 的一句取预检的文字（"持有它的"和已归档的列表）；R3 的第 5 节 M4 一行写明 HTML 的理由；L3 `T2.8` 标"（Task 4 起）"；L4 `toPass({ timeout: 5_000 })`，A.4 第 46 条列出 `toPass`；L5 由 A-M2 代替；L6 删去 `optionsFilter`；L7 组字时的 Escape 不关列表，能便宜地写会失败的测试就写，否则记作只由评审看住；L8 写进第 5 节 M4 一行；L9 便宜就在根上改成一个 Tab 停留点，加一个两个停留点时失败的检查，里面一定要留按钮就写明理由；L10 Task 11 的两行照预检的文字。
  - 修订写明的几处偏离，控制者接受：A-M2 的侧边栏一步由访客 ann 做（侧边栏和项目页头只给访客提供离开，被拒绝的是 nerve 的 403，不是预检写的 pat 的 409）；A-M5 第 2 项不写代码（Headless UI 的 `Popover.Button` 自己阻止点击，我们的 `preventDefault` 反而让它跳过切换），只由评审看住；L9 之前一节的标题仍是 `tabIndex={-1}` 的按钮（只能用鼠标的 `<span>` 加点击被 jsx-a11y 标出，放在选择的按钮旁边会改 M4–M6 页头的布局）；L1 记录中的 `member-options`（M2、P8a 的 spec、plan、评审，M2 交接第 14 节的问题本身）不改。
- 开始执行时：
  - **照 plan 执行**：预检的跨 Task 表已逐对核对，它的发现都在修订中落实，并由再一次的复现证明。代价：错了是一轮 Task 级的修正。
  - **E1**：Docker Desktop 起着；每个 Task 在本地跑 `make e2e`，数目照 plan 的全局约束；Docker 不再回答时实现者报告，控制者依靠分支的持续集成。代价：每轮慢一些。

**执行中**
- **T1-a … T1-d（表情选择器的数据）**：
  - 实现者报告两点：它重用了 P9 留下的 `$M3TMP/t1-mut`，盖掉了 P9 的 T1 变异日志（P9 已合并，没有损失）；`emojibase.ts` 的注释说版本由它的 `package.json` 固定，其实是 `pnpm-workspace.yaml` 的 catalog。T1-a：之后每个 Task 的变异在新目录 `$M3TMP/p10mut/tN` 跑。T1-b：注释交给 Task 的评审判断。代价：没有。
  - 评审：Spec ✅，Quality 通过，C0/I0/M2（18 个文件与快照逐字节相同；锁文件没有加包；构建写出约 700 KB 的两个文件，webui 嵌入、答 `GET` 和 `HEAD`，缺的资源 404；真的 Vite 开发服务器也答；CSP 不变；`requestsElsewhere` 与 S2 原来的记录器等价；propel 的清理不改行为）。m1 那句注释不真：T1-c 当轮用评审给的句子修（`80584941`），控制者逐字核对代替范围复审。m2 plan 第 15 行和 brief 说锁文件多"两处 importer 的条目"，其实一处是 catalog 的：T1-d 记作 plan 文字的漂移，不改（搁置为 P1，P8b、P9 的先例）。代价：一个小提交。
- **T2-a … T2-g（下拉框）**：
  - 实现者报告三点（M4 的三处调用方给列表的 z-index 落在没有定位的 `<ul>` 上；只是样子的不同；Escape 之后焦点看不出），交给评审照 A-M5 判断。分支的持续集成在 e2e 失败，本地两次通过；日志要 GitHub 的认证，这台机器没有。
  - 评审：Spec ✅（15 个文件与快照相同，变异的行与结果相同），Quality 要修，C0/I2/M4。I1、I2、M1 见第 1 节（逐个看过 9 处 `CustomSearchSelect` 的直接调用、5 个页头的面包屑、13 个文件中 15 处成员下拉框）；M2 Escape 之后的焦点；M3 `transitionsEnded` 等无限的动画，自己没有期限；M4 一句注释说的比真的多，第二个 vitest 的标题说到自定义按钮却没核对它（去掉 `customButton` 的变异在 vitest 存活）。
  - 裁定：
    - **T2-a**（I1）：在根上改，`optionsClassName` 到达 popper 定位的 `Popover.Panel`（`cn("z-30", optionsClassName)`），调用方的 z-index 照旧胜出；逐个核对调用方的类照旧渲染。代价：没有，恢复约定。
    - **T2-b**（I2、M2）：看得见的键盘焦点只写在 `CustomSearchSelect` 自己的按钮上（两支都加，Headless UI 的 `data-focus:`，只在键盘时），每个调用方随之显示焦点；第 5 节 M4 一行加 Escape 之后的焦点。代价：一个组件上的类。
    - **T2-c**（M1）：去掉悬停底色；去掉根新加的 `flex-shrink-0`，P10 自己的布局要它时只在调用处加；列表其余的样子写进第 3 节第 15 条和第 5 节 M4 一行。代价：几个类。
    - **T2-d**（M3）：`transitionsEnded` 只等有限的动画。**T2-e**（M4）：注释只说真的；第二个 vitest 核对它标题说的。代价：没有。
    - **T2-f**（持续集成的失败）：修正轮先在本地照持续集成的条件复现（`make build` 之后两个 worker 跑全部两次；Task 2 的故事两个 worker 各五次），复现了就在根上修、不靠多等；复现不了，修正轮的推送是下一次核对，再一次只在持续集成上失败就交给负责人取日志。代价：一轮更长的 e2e。
  - 修正轮（`86fe7e11`）：持续集成的条件没有复现，16 个 worker 在 P1 复现，根因见第 1 节。面板对每个 Escape 都 `stopPropagation`，只在不结束组字的 Escape 上关列表；P1 数传到 `document` 的 Escape，要 0（不靠时间）。之后两个 worker 95/95，×5 70/70，8 个 worker ×10 140/140；16 个 worker 下另几次失败是 Postgres 的"too many clients"。T2-c 没有照字面去掉 `flex-shrink-0`：BASE 上它就在 `CustomSearchSelect` 的根上，直接的调用方留着它（改名 `shrink-0`，同样的 CSS；tailwind-merge 不认识前者，调用方的 `shrink` 盖不过它），成员下拉框传 `shrink`。plan 中 Task 4 的 P1 导入块和 Task 8 的 `fake-controls.ts` 块随之改（第 5 节），加 `T2.16`–`T2.18`。范围复审（sonnet）通过：确认根因是旧代码在组字的 Escape 上于 `stopPropagation` 之前返回，改名不改行为。之后的持续集成通过。
  - **T2-g**：实现者报告 Plane 的 `useKeypress` 不看组字：创建弹窗的任何字段中组字结束的 Escape 都关上弹窗；修 hook 的一行会改到 M4–M7 的调用方。裁定：这是 zh-CN 真实的一类缺陷，结束组字的 Escape 不能关上任何东西，修在根上（`useKeypress` 和 P10 的弹窗用的其他 Escape 处理，查 Headless UI 的 Dialog）；它是 web 全局的 hook，改动只去掉错的行为，不是架构意义上的跨模块。搁置为 P2 给修复轮，加一个 P1 的端到端，写进 W17。代价：到修复轮之前，中文用户在创建弹窗中按 Escape 结束组字会关上弹窗。
- **T3-a … T3-h（列表、卡片、加入；包装层；复制；F-3）**：
  - T3-a：实现者在 plan 之外删除了两种语言的顶层键 `link_copied`（它的读者只有本 Task 重写的侧边栏，Task 4–11 的块照样能应用）：接受，等评审确认没有读者（评审确认）；孤儿检查的盲点（一个键的名字是另一个键路径的末段）搁置为 P3，与 PF-L5、PF16 一起。代价：没有。
  - T3-b：两个只读的 git 命令用 `;` 串起来：规则失误记下，之后的分派重申规则。代价：没有。
  - 评审：Spec ❌（`decide` 的顺序只钉住一半），Quality 要修，C0/I3/M2；代码是对的，三条 Important 都是不会失败的测试（第 1 节）。M1 一句注释缺"已归档的列表"的限定；M2 第 3 节第 15 条没有列出复制的提示的改动，第 8 条的引文写错。
  - 裁定：**T3-c**（I1）P2 的卡片故事扣住加入，核对"Joining..."不可用；一次被拒绝的加入（管理员在对话框开着时删除项目）：原因提示、对话框留着、按钮重新可用；两个变异都要被发现。**T3-d**（I2）决定表加三行（已归档的项目：找不到、不可达、没回答）。**T3-e**（I3）照评审加两个 store 的测试。**T3-f**（M1）注释用设计 7.7 的说法。**T3-g**（M2）一个复制 hook 统一的提示接受（一处复制，一种说法），写进第 3 节第 8、15 条，第 8 条的引文改为"Link copied!"。代价：约 20 行 e2e、几个测试。
  - 修正轮（`3fee585b`）：新的共用 fixture `closedWithin`（一秒之内不关，`closedByEscape` 改为经它），`T3.13`–`T3.17`；P2 的故事标题的漂移记进 P1。范围复审通过，另指出 spec 的 vitest 总数没有随 +5 改。**T3-h**：搁置为 P4，修复轮在最后统一重算 spec 的数目（vitest、e2e、变异、W18；P9 修复轮的做法），代码和测试已通过。代价：spec 的数目到修复轮之前落后。
- **T4-a … T4-e（创建项目）**：
  - 评审：Spec ✅（18 个文件中 17 个与快照相同，P1 只差 `86fe7e11`），Quality 要修，C0/I3/M1（第 1 节）。Task 4 没有让 P2 更糟：每一步只有一个 Escape 的监听起作用。
  - 裁定：**T4-a**（I1）扩展 `fake-tab.ts` 的 `lateSettlings`（不复制），兑现时答以测试给的回答，这里答以 `web`，核对其余调用方照样通过。**T4-b**（I2）在 vitest 钉住：候选的筛选抽成 `attributes.tsx` 旁边的纯函数，表中有成员关系已结束的人和访客；P1 已约 400 行，不再加长。**T4-c**（I3）P1 在扣住创建时核对"Creating"不可用；P1 超过约 400 行就把帮助函数移进共用的 fixture，不拆故事。**T4-d**（M1，再选一次已选的负责人就是没有负责人）：不是会话、权限、取数的性质，记作只由评审看住（A.2）。代价：共用的替身多一个参数，一个小函数和它的表，一行。
  - 修正轮（`8ac39a87`）：`lateSettlingsAnswering(answer)`（其余调用方不变）、`leadCandidates`（4 行）、`named` 移到 `e2e/fixtures/auth.ts`（P1 395 行）、`T4.12`–`T4.14`。实现者另报告 P9 的 `use-create-workspace.test.ts` 有同样的缺口：**T4-e** 搁置为 P5（一行，加一个变异）。代价：到修复轮之前是测试的缺口，不是缺陷。
  - 范围复审：T4-a–T4-c 通过，T4-d 要的 A.2 一条没写：控制者照裁定的文字补上（`46bfd390`，自己的裁定的一行记录，不另复审）。
- **T5-a … T5-d（general 页）**：
  - 评审：Spec ✅，Quality 通过，C0/I0/M3。核对了：请求体是 `ProjectUpdate` 的六个字段；标识的检查在修改之内，检查和 `PATCH` 之间换了账户也写不成错的人（客户端绑定它的 `loginId`，`auth-middleware.ts:37` 在发出之前抛 `SessionChangedError`）；删除的 4 个键没有读者。
  - 实现者的探查 P5.1（读 `currentProjectDetails` 而不是 `getProjectById(projectId)` 通过每道检查）：评审判断不是标准要求检查的缺口（取数和 `isAdmin` 来自参数，`PATCH` 发到显示的项目；同一路由内的移动之后只晚一次提交，用户来不及操作，应用中今天没有这样的移动；相称的检查没有：关键词规则要 9 个例外，页面的 vitest 要约 5 个替身）。**T5-a**：记作只由评审看住（A.2，进修复轮的文档 P4）。代价：没有。
  - **T5-b**（m1）：标识现在保留 ÇŞĞİÖÜ（nerve 接受，原来丢掉），写进 W17（P4）。**T5-c**（m2）：`use-update-project-details.ts:71` 自己写死 `"errors.project_identifier_taken"`，`PROBLEM_MESSAGES` 已有：搁置为 P6。**T5-d**（m3）：general 页控制区的删除读 `currentProjectDetails`，归档读参数：搁置为 P7 给整分支评审和修复轮，清扫 P10 的页面中从 router store 读地址的，移到参数，不是 P10 的设置页列进第 5 节。代价：今天没有（走不到）。
- **T6-a … T6-e（features、automations）**：
  - 实现者的探查 P6.1、P6.2 存活（第 1 节），另报告四个功能页从 `currentProjectDetails` 读开关的值、automations 页读它的标题。
  - 评审之前的裁定：**T6-a**（会话和弹窗一类）：自动归档的 vitest 中自定义时长的替身留下它的属性，用三种方式调 `handleChange({ archive_in: 6 }, done)`（回答了：`done` 调一次；被拒绝：不调，提示 nerve 的原因；换账户之后：都不），P6.1 要被发现。**T6-b**（T3-c、T4-c 一类）：P3 核对"Submitting..."在扣住时不可用。**T6-c**：那几处读法加进 P7 的清扫。代价：约 3 行 vitest、一行。
  - 评审之前的一轮（`6a028754`）之后评审：Spec ✅，Quality 通过，C0/I0/M2（开关在轮到它时决定；自动归档读地址的项目；自定义时长在途时关不掉、忙到跟进做完；请求体只有字段、是数字；每个新测试都能失败）。**T6-d**（m1，`fake-toast.ts:22-37` 的 `setPromiseToast` 替身和 `Outcome` 成了孤儿）：搁置为 P8，修复轮删除。**T6-e**（m2，W17：功能开关的"Updating project feature..."加载提示没了；2.13 的自动归档一行也要写 `T6.8` 的测试）：并进 P4。代价：没有。
- **T7-a、T7-b（成员页）**：
  - 实现者的探查 P7.1–P7.7 存活，另有三处小的（P5 的 `toBeEnabled` 在淡出中也通过；`PROJECT_ROLES` 的注释从 Task 8 起不真；成员页的路由组件读 `currentProjectDetails` 的标题）。
  - **T7-a**（评审之前）：每一条都是已裁定的一类（权限的镜像；T3-c、T4-c、T6-b 的弹窗；T6-a 的 `done` 交接），全部补上：P7.1、P7.2 两行 vitest，对照 nerve 的规则；P7.3 P5 一行 `enabledWithin`；P7.4 成员行的 vitest（回答了关上，被拒绝留着，换账户之后什么都不做）；P7.5、P7.6 `useProjectColumns` 的 vitest；P7.7 `choices={[]}` 一行；P5:354 改用 `closedWithin`；注释照实。每个探查成为一个会被发现的变异行，P5 保持约 400 行（多用 vitest）。路由组件的标题归 P7。代价：评审之前一轮。
  - 评审之前的一轮（`b05b42bb`，`T7.18`–`T7.27`）之后评审：Spec ✅，Quality 通过，C0/I0/M2。权限的镜像逐种对照 nerve 的 `decide.go`、`rules.go`、`membership.go`（`CheckRoleChange`、`CheckRemoval`）和三个用例：页面从不提供 nerve 拒绝的，也不藏 nerve 允许的。**T7-b**（m1 `rowOf` 有三份；m2 `project-roles.test.tsx` 手写已登录的会话）：搁置为 P9（只是测试的重复）。代价：没有。实现者两次把只读的 `git diff` 串起来，规则失误记下。
- **T8-a、T8-b（添加成员、成员的默认值）**：
  - 实现者的探查 P8.1、P8.6、P8.11、P8.3、P8.5 存活；另报告 `member-list.tsx`、`MemberSelect` 读 router store，`a6-sign-out.spec.ts:25` 可以用 `shownNameOf`。
  - **T8-a**（评审之前）：五个全补上（都是已裁定的一类：弹窗的忙、会话、权限的显示、A-M2 的拒绝、T4-a），多用 vitest（P3、P5 约 400 行）；router store 的读法归 P7，`shownNameOf` 归 P9。代价：评审之前一轮。
  - 评审之前的一轮（`45736939`，`T8.12`–`T8.16`；P5 正好 400 行）之后评审：Spec ✅，Quality 要修，C0/I1/M1。权限的镜像与 nerve 相同（`addableRoles` = `CanAdd`，添加的候选 = `CheckTargets`，负责人、默认负责人 = `CanAssign`，"Add member" = `project_member.add`）。I1 见第 1 节；m1 `member-list.test.tsx` 照抄了 `permissions.store.test.ts` 的权限 store 的接线。
  - **T8-b**（I1 和 m1 一起）：权限 store 的接线抽进一个共用的替身（`store/user/fake-permissions.ts`），三个测试用它；成员默认值的 vitest 用 store 自己的权限，`it.each`：项目管理员、是项目成员的工作区管理员可以改，项目成员、项目访客、不是项目成员的工作区管理员不能；读两个选择的 `isDisabled` 和开关的 `disabled`（`fake-controls` 的开关替身加 `disabled?`）；放宽到成员和 `isDisabled={false}` 的变异要被发现（`T8.17`）。代价：一轮修正；不这样就是第三份接线。`assert/identity.ts:111` 自己算的 `shownNameOf` 归 P9。
  - 后台的安全评审标出 `project-settings-member-defaults.tsx` 中的 `isDisabled={false}`：那是实现者证明新测试时临时写的变异 `T8.17`；提交的文件（`45736939`、之后的 `3ff60245`）和工作区都是两处 `isDisabled={!isAdmin}` 和 `disabled={!isAdmin}`（git grep 核对），nerve 的 `project.update` 对不是管理员的人 403，`T8.17` 现在守着页面的门。不是缺陷。
  - 修正轮（`3ff60245`，`T8.17`、`T8.18`）之后范围复审（sonnet）通过：五行与 nerve 的 `rules.go:83` 和 `rules_test.go:41` 的表相同；生产文件没动；三个文件的 vitest 41/41。
- **T9-a、T9-b（删除、归档、恢复）**：
  - 实现者的探查 P9.1、P9.2、P9.7、P9.8、P9.9、P9.12、P9.6、P9.10、P9.11 存活；W17：未归档的卡片的设置链接现在也给是工作区管理员的项目访客（nerve 允许）。两次探查的端到端碰上 testcontainers 的端口超时，重跑干净。
  - **T9-a**（评审之前）：全补上（在途时的忙 T3-c、T4-c、T6-b，T4-a，权限的显示 T8-a、T8-b，F-2 的忙到跟进做完）：P4 两行 `enabledWithin`；`lateSettlingsAnswering(web)`；卡片的 vitest 经 `callerInWeb` 对照 nerve 的规则；P4 中 wes 在设置页的归档、删除；归档的弹窗的 vitest 扣住跳转（做不便宜才记作只由评审看住）；删除在另一个项目的页面上不跳转的一行；提交按钮的 `disabled`（便宜就钉住）；W17 一行进 P4。代价：评审之前一轮。
  - 评审之前的一轮（`fb38b1a9`，测试、共用的替身、plan 和 spec，没有生产代码；`T9.14`–`T9.23`；P4 304 行）。归档的 vitest 用 `vi.mock("react")` 包住 React 的 `useState`，在服务端渲染之后看到忙：一种新的替身，留在这个测试里。
  - 评审：Spec ✅，Quality 通过，C0/I0/M4。权限的镜像与 nerve 的 `rules.go:83-87`、`decide.go:81` 相同，放宽和收窄都会失败。保留 `vi.mock("react")` 的写法：它委托给真的 `useState`，只记下组件自己的 setter，是最强的便宜检查（`settledYet` 看不到、要改生产代码，e2e 要截脚本块），出现第二个状态时明确地失败；第二个测试用到它之前留在本地。
  - **T9-b**：m1（`card.test.tsx` 读菜单的方式与 `ContextMenu` 不同，第 1 节）搁置为 P10，排在修复轮的第一个；m2（`Caller`、`its` 的第三份）进 P9；m3（第 3 节第 14 条说归档"不关就跳转"，代码先关再跳转）和 m4（2.13 的"不跟进的"漏了归档的 `setIsLoading(false)`、删除和添加的 `isSubmitting`）进 P4。代价：到修复轮之前，那一种变异形状只由评审看住。
- **T10-a … T10-d（侧边栏的拖动、标签栏）**：
  - 实现者的探查 P10.1、P10.6、P10.9、P10.7、P10.8 存活；另报告 P8 第三次手写工作区设置的 `PATCH`（api.ts、W3 之外）。
  - **T10-a**（评审之前）：全补上：P10.1 每个修改的回答经 `made()` 串起来（PF-M2 是 v0 7.7 的规则）；P10.6、P10.9 一个"更多"的 vitest，`Menu` 的替身扩展 `fake-controls`，不复制；P10.7、P10.8（T8-a、T8-b 一类）访客的标签菜单和工作区访客的拖动各有会失败的检查（`callerInWeb` 的 vitest 或 P8 中的访客，取便宜而真的，P8 保持约 400 行）；api.ts 一个工作区设置的帮助函数，P8、W3 和现在的调用方共用（便宜，现在做）。代价：评审之前一轮。
  - 评审之前的一轮（`cc1ee856`，`T10.8`–`T10.12`；P8 中的访客 Gus）之后评审：Spec ✅，Quality 要修，C0/I1/M3。I1 见第 1 节；m1 `changeWorkspacePreferences` 与 W8 自己的 `change` 重复；m2 扩展侧边栏的拖动（`disableDrag={false}`）没有故事，收窄它存活；m3 spec 2.10、A.4 第 46 条把 5 秒的 `toPass` 说成拖动的（是 `menuOf` 的，拖动的期限是 `sentTo` 的 10 秒），`tab-navigation-overflow-menu.tsx:31` 的注释说的比真的多。
  - **T10-b**：I1 照评审给的次序（两次设 Modules 为默认、显示 Views、收起 Modules、显示 Modules），两个变异行都要被发现；m1 一个交回 nerve 的回答的帮助函数，删去 W8 的那份；m3 的注释照实。代价：一轮修正，只改测试和一句注释。**T10-c**：m2 搁置为 P11（一个 `callerInWeb` 上会失败的检查，或写明理由的只由评审看住）；m3 的 spec 句子进 P4。代价：到修复轮之前没有。
  - 修正轮（`bd151a8a`，`T10.13`、`T10.14`；唯一的生产改动是一句注释）另修好一个偶发（第 1 节）；api.ts 404 行（Task 10 开始时 390），约等于 400，接受。范围复审（sonnet）通过：确认偶发的修法在根上（`transitionsEnded(body)` 去掉了提示进来时的滚动），期限保留（只等有限的过渡，T2-d），fixture 共用。
  - **T10-d**：偶发的修法作为根上的修法接受。代价：没有。
- **T11-a、T11-b（文档）**：
  - 实现者照实改了 plan 的块的九处（每处都照 HEAD 为真：前端改动清单的下拉框、复制、创建、已归档、标签栏几行，M2 交接第 10、11、14 节，M1-P4 的结果）。另报告两点：spec 2.3 说页头的复制经 `useCopyProjectLink`，代码经 `useCopyLink`；错误一行删去"P11"，可标签设置页仍把 `project.label_name_taken` 放在提示里。**T11-a**：两点都进 P4。代价：没有。
  - 评审：Spec ✅（九处偏离都照 HEAD 为真；关闭的说法诚实：第 10、11 节关闭，第 14 节"代码一侧完成"到 C7、C8 之前留着），Quality 要修，C0/I1/M3：I1 前端改动清单没有第 3 节第 15 条的弹窗一条（README 说每个改动都列出）；m1 "M4–M6"漏了 M7 收集箱的两个指派人；m2 一行暗示唯一的管理员能从侧边栏或页头离开；m3 三处小的（`WhileOpen`、新的可访问的名字、general 页被拒绝的标识检查）。
  - **T11-b**：一轮只改文档的修正全部补上，外加 P4 中给前端改动清单的两条 W17（T6 的加载提示、T9 的设置链接），一次把改动清单写全，P4 只留 spec 的文字。理由：这些文档的用处是在 HEAD 上为真。代价：一个小的文档提交（`1cb735d9`），由控制者逐字核对代替范围复审。

**整分支评审之后**
- **F-1**（搁置的 P10，修复轮的第一步）：`card.test.tsx:58` 照 `ContextMenu`（`packages/ui/src/dropdowns/context-menu/root.tsx:79`）筛 `item.shouldRender !== false`，加一个变异（`shouldRender: isArchived && hasAdminRole ? true : undefined`）。理由：读菜单与应用不同的测试会藏起一个提供。代价：小。
- **F-2**（I1 和搁置的 P7，改的是同几行，一步）：评审的 P7 清单中 P10 的七处从路由参数读地址（控制区、四个功能页、automations、成员页、`member-list`、`member-select`、项目设置侧边栏的头部），这些地方的 `allowPermissions` 明写工作区和项目；一条关键词规则不让这些文件读 router store（68 → 69 条，有命中和不命中的样例）；一个 vitest 用 store 自己的权限（`callerInWeb`，P9 搁置项的共用调用者行从 `fake-permissions.ts` 导出），router store 指着别处，渲染五页的门：项目的管理员、是工作区管理员的项目成员和项目访客看到开关，项目的成员、访客看到 `NotAuthorizedView`；渲染路由模块不相称时退回 P3 的步骤，写明理由。不是 P10 的地方按 M 写进 spec 第 5 节，各自移它们的地方时扩大规则的范围。理由：I1 是只有评审守着的权限性质；P7 是"读地址只有一种写法"。代价：一步。
- **F-3**（搁置的 P2，照评审放宽）：组字的 Escape 什么都不关：`useKeypress` 在 `event.isComposing` 时返回；`ModalCore` 面板上捕获阶段的 `onKeyDownCapture` 对组字的 Escape `preventDefault()`（Headless UI 自己的退出方式，38 个使用者都覆盖）；propel 的 `emoji-picker.tsx:122` 在 `handleToggle(false)` 之前返回；照仓库的写法和包的依赖图，不写共用的函数；端到端：P1 的名称框和表情选择器的搜索框，P4 删除弹窗的名称框；每个处理器一个变异；W17 和前端改动清单记下（看得见的改动：输入法用户的 Escape 只结束组字，不再关上弹窗和选择器）。理由：根在三个处理器，只修 `useKeypress` 会留下删除、离开和选择器。代价：一步。裁定之前控制者核对：Headless UI 2.2.10 的 `useEscape` 在 window 上听 keydown、跳过 `defaultPrevented` 的；仓库在 6 处只查 `isComposing`；propel 不依赖 ui、utils。（捕获阶段的机制不对，修复轮用冒泡阶段，F-10 接受。）
- **F-4**（M6，解决搁置的 P11）：删除 `SidebarProjectsListItem` 和两个调用方的 `disableDrag`、`disableDrop`：加入的列表的拖动和放下没有条件，nerve 的 `project_preferences.update` 允许每个有效成员。理由：负责人的"不要开关"，变异也就没有地方落；P8 中 Gus 的拖动核对剩下的这一条路。代价：小。
- **F-5**（M5，加搁置的 P6、P8）：页头的复制是 `activeItem ? copyLink(activeItem.href, …) : copyProjectLink(projectId)`，spec 2.3 由此为真（T11-a 的 spec 改动改为"两种都写"）；P6 标识被占用的键一个来源，加一行 vitest；P8 删除孤儿替身。代价：小。
- **F-6**（测试的重复：搁置的 P9 全部、M5 的 `underFields`、搁置的 P5）：`fake-projects.ts` 的 `projectMemberDetailsOf`；`project-roles.test` 用 `fake-tab`；`a6-sign-out.spec.ts:25` 和 `assert/identity.ts:111` 用共用的 `shownNameOf`；`Caller`、`its` 从 `fake-permissions.ts` 导出（F-2 用）；`fake-refusal.ts` 旁边一个 `underFields`；P5：P9 的创建工作区换账户之后的一行答以工作区（`lateSettlingsAnswering`），加一个变异。代价：小。
- **F-7**（M1）：`web/packages/utils/src/project.test.ts` 进非空断言的范围、2.14 的 Task 4 清单和一个 W12 行（RV2 被发现）。代价：小。
- **F-8**（变异的记录，照 P9 的 F-7）：`mutants_p10.py` 的每一行（含修正轮加的）、每个 Task 的探查、W18 和中文的表、修复轮自己的，合成一张表，在修复轮的头上用 `mut.py` 在后台跑；之后的改动让它等价或不适用的，写明理由，不硬跑；结果写进 `$M3TMP/p10tools/p10-mutation-record.md`，spec A.2 的总数取自它。代价：跑得久，在后台。
- **F-9**（文档：搁置的 P4 全部、M2、M3、M4、过时的句子）：spec 的每个数目在修复轮的头上重数（vitest、e2e、变异和它的层、W18、上限、规则数）；P4 的每一项（T5-a、T5-b、T6-e、T9-a、T9-b 和评审的三个不跟进的状态、T10-c、T11-a；T7-a 记下的成员路由的读法现在由 F-2 移了，照实写）；M2（第 5 节收尾的两处复制）、M3（M4 一行写出 8 个文件）、M4（M7 一行加收集箱的两个下拉框，"同 M4 一行"）；成员下拉框的调用方"M4–M6"改为"M4–M7"，spec 和 M3 设计 7.7、9.7 C7、13.1 都改；第 3 节第 15 条加 `CustomSearchSelect` 开着时卸载调 `onClose`；第 3 节、W17 加 F-3、F-4 的改动；第 5 节加 F-2 的交接；第 5 节收尾加搁置项 P3（`i18norphans` 按完整的键路径，并进 PF-L5 的工具缺口；评审按完整的键路径扫过，没有孤儿）；前端改动清单加 F-3 一行；M8 的 M1 收尾交接写"处理结果（M3/P10）"（`:53`：选择器读 nerve 固定的一份，只剩标注块的图标 `:54`）；P1 的 plan 漂移记在评审里，不改 plan；不碰 M3 设计 §15 的 P10 一行。代价：一步文档。
- 次序：F-1、F-6（F-2 要的共用调用者行）、F-2、F-3、F-4、F-5、F-7、F-8（从一开始就在后台）、F-9 最后；一个实现者（不指定模型），一次分派（`fix-wave-brief.md`，九步），之后 opus 复审。没有一条是架构层面的：F-3 只为组字的按键改一个共用组件的 Escape，F-2 不改用户看得到的东西。分派之前控制者改了 brief 中的两个路径（propel 的 `emoji-icon-picker/`、`e2e/stories/identity/`）。
- **F-10**（修复轮的报告之后）：
  - 接受五处偏离（第 4 节）。
  - 关注 1（地址规则的模式在范围之外还命中 109 个文件，报告说多是工作区一级的读法）：没有已知的缺陷；应用是否不再从 router store 读工作区是跨 Phase 的决定，写进 spec 第 5 节收尾一行（带数目和 `routerreads.py`），也写进给负责人的收尾报告；现在不升级。
  - 关注 2：`api.ts` 404 行约等于 400 行（P9 的 F-10 先例）；`assert/identity.ts` 472 行是 M2 的，P10 之前 471 行，拆分进收尾一行。
  - 关注 3：凡写"14 个文件"的地方改为"12 个文件（M4 8、M6 2、M7 2）"（控制者照 `git grep -w MemberDropdown` 在它的目录之外核对：12 个文件；P10 的创建用 `MemberDropdownBase`）。
  - 关注 4：用 `e2ekills.py` 重查之前各 Phase 的端到端的发现，进收尾一行。
  - 关注 5：控制者清理时删除 `$M3TMP/p10mut/fix/base`。
  - 这些文档项和复审的发现在复审之后一起用一个提交（P9 的 F-11 先例）。控制者另记下：`dropdowns/member/base.test.tsx:11` 的注释仍写"M4–M6's rows"，也进这个提交。代价：一个小的文档提交。
- **F-11**（复审之后）：修复轮的实现者用一个提交补上：F-10 的各项；复审的 M1、M2；`base.test.tsx:11` 的"M4–M6"；Safari 的说明：收尾一行记下 9 处输入法的守卫只查 `isComposing`、Safari 的顺序躲过它们，`@nerve/ui`、`@nerve/propel`、web 共用一个加上 MDN 的 `keyCode === 229` 的判断是跨 Phase 的决定（它改到 M4 的 Enter 的守卫），P10 不做，写进给负责人的报告。门禁 `make lint-web`、`make knip`、`make test-web`，sonnet 做范围复审。代价：一个小提交。
- **F-12**（跟进的报告之后）：报告把 109 个文件按读的是什么分开（第 4 节的"F-10、F-11 的跟进"），其中 29 个不传项目的项目级 `allowPermissions` 写进它们的阶段的行（M4 16、M6 11、P11 2），每个交接是一份完整的清单；收尾只留 36 个只读工作区的。一个只改 spec 的提交，之后 sonnet 对 `28526539..093d0d91` 做范围复审。代价：很小。

**浏览器核对之后**
- 本记录的初稿指出 C7 没有核对标签式导航的项目选择、没有用键盘打开创建弹窗中网络的选择，C8 没有核对扩展侧边栏的复制。控制者用同一个构建、一个新的临时库补做（第 2 节），找到 P10 页面上 C7 的两个生产缺陷 D1、D2。C7 是 P10 的完成线，这些是 P10 的页面：
  - **F-13**（D1，网络的选择 Tab 到不了）：`CustomSelect` 把 `tabIndex` 给它的按钮（`customButton` 和 `label` 两支），不给 Listbox 的根，与 P10 对 `DropdownButton` 的规则相同（spec 第 3 节第 3 条）；`attributes.tsx` 包它的 `div` 不再收 `tabIndex`。另一个给 `tabIndex` 的调用方是 M6 的 `ModuleStatusSelect`，写进 spec 第 3 节第 15 条和第 5 节 M6 一行。检查：P1 从说明按 Tab 到网络的按钮，空格打开；变异把 `tabIndex` 放回根上。
  - **F-14**（D2，焦点走到卡片关着的右键菜单）：`ContextMenu` 关着时 `inert`，不在 Tab 顺序里，也不在无障碍树里；布局照旧，右键时它仍量得到菜单的大小。检查：创建弹窗的 Tab 一圈不离开弹窗，另加一个打开的菜单照旧能用的检查；变异去掉 `inert`。spec 第 3 节第 15 条、第 5 节 M4、M6 行（它们的列表的菜单）和前端改动清单记下。
  - 两条的理由：在共用组件的根上修，改动局部，只去掉坏的停留点，不是架构层面的。代价：一轮小的修复（修复轮的实现者接着做）、复审、持续集成，在新的构建上再做浏览器核对；本记录等这些之后再改。
- 实现者做完 F-13（`d30199b2`）、F-14（`b6344182`）之后停下报告两点：
  - P1 走一圈 Tab 留不在创建弹窗里。它归因于 `ModalCore`：`Transition.Root` 挂载 `Dialog` 时它已经打开，Headless UI 让其余部分 `inert` 的处理从不隐藏应用。
  - 它对 spec 的改动会碰到本记录尚未提交的状态一行。控制者把本记录的三处已跟踪的改动存进 `$M3TMP/p10-review-doc-tracked.patch` 并恢复为 HEAD，初稿备份到 `$M3TMP/p10-review-doc-v1.md`；修好之后在新的 HEAD 上重做（不套用那份补丁：状态一行之后由实现者改过）。
- 控制者在 `28526539` 的构建上探查删除项目的弹窗（没有正的 `tabIndex`）：焦点陷阱守住 Tab（名称 → 确认 → 取消 → 名称），可弹窗打开时没有一个 `body` 的子元素 `inert` 或 `aria-hidden`。所以是两个缺陷：
  - 缺陷 i：Tab 离开创建弹窗。它的正的 `tabIndex` 排在 Headless UI 焦点陷阱的哨兵（0）之前，只在 P10 的创建表单里。
  - 缺陷 ii：每个经 `ModalCore` 的弹窗后面的页面对辅助技术仍然可见，是整个应用的、跨模块的。
- **F-15**（缺陷 i）：删去 P10 的创建表单（`projects/create/`、`project/create/` 下的字段、网络、负责人、按钮和页头的关闭）的正的 `tabIndex`，Tab 的顺序就是 DOM 的顺序，本来要的就是这个顺序；`PROJECT_CREATE` 这张表没有别的读者就删除；P1 核对走一圈 Tab 留在创建弹窗里（`tabCycle`）；spec 第 3 节第 12 条重写：P10 删去创建表单的正的 `tabIndex`，因为它让 Tab 离开弹窗，交给 M4、M6 的表单的那一句加上这个理由。理由：在根上修，局部，删代码。代价：小。
- **缺陷 ii 交给负责人**（本阶段唯一的一项）：它涉及 M2–M8 的每个弹窗，修它要把 `ModalCore` 改写为 Headless UI v2 的 `Dialog`（`DialogPanel` 和它的过渡），是跨模块的，不是控制者能自己定的。P10 的合并等负责人的回答；本地的工作（F-15、文档、复审、持续集成、浏览器的再核对）照常进行。
  - **负责人的裁定（2026-10-10）**："放进 M3 收尾"。改写 `ModalCore`（Headless UI v2 的 `Dialog`，`DialogBackdrop`、`DialogPanel` 的过渡，每个弹窗后面的页面 `inert`、`aria-hidden`，加一个会失败的检查，全部回归）是 M3 收尾的一项独立的任务；P10 不等它合并。到那时之前，弹窗打开时读屏软件仍读得到后面的页面（第 7 节）。
- **F-16**（`FW13.root` 由评审看住）：F-13–F-15 之后，记录中只有这个变异没有被发现：自 F-15 起 P10 的页面上没有给 `CustomSelect` `tabIndex` 的，唯一给的是 M6 的 `ModuleStatusSelect`。`@nerve/ui` 没有测试的设置，只为一个属性加一套是过度设计；M6 一行已带着它的核对，M6 删去它的表单的正的 `tabIndex` 之后这个属性没有调用方，随之删除。spec 的 A.2 要写明（复审的 M2 指出没写，F-17 补上）。代价：没有。
- **F-17**（第二次复审的四条，第 4 节）：实现者一个提交：M1–M3 是文档，用复审给的文字；M4 删去 `CREATE_LABEL` 的三行，让 spec 第 3 节第 12 条为真（负责人的"不留孤儿"；它早于 P10，可 P10 改了这张表）。门禁 `make lint-web`、`make knip`、`make test-web`。代价：很小。
- **F-18**（实现者在 F-17 的报告中的关注）：创建弹窗的功能一步（`project/project-feature-update.tsx:48`、`:56`）的关闭、打开项目是 `tabIndex` 1、2，各有一处抑制，照 F-15 的机制 Tab 在打开项目之后离开弹窗（推出的，没有跑）；P10 的页面之外还有写死的正的 `tabIndex`。裁定：功能一步删去两个正的 `tabIndex` 和两处抑制（spec A.5 的数随之减少）；P1 的 Tab 一圈也走功能一步；一个变异放回一个正的，要失败；spec 第 3 节第 12 条和前端改动清单写上功能一步；第 5 节交出 P10 之外的那几处：M4、M6 各 2 处，收尾 2 处（`AlertModalCore` 的确认按钮归进 `ModalCore` 的任务，是同一份共用弹窗的工作；命令面板的快捷键弹窗）。实现者的报告写"七处"，F-18 的文档提交照 grep 改为六处（8 行中 2 行在 P10 的功能一步）。理由：与 F-15 是同一个缺陷，在 P10 自己的弹窗里。代价：小。之后持续集成，sonnet 对 `9b3ff184` 之后的提交做范围复审。

## 4. 整分支评审的发现与处理

结论"修复轮之后可以合并"：Critical 0、Important 1、Minor 6。评审（opus）的范围是 `4a4e7446..1cb735d9`（130 个文件，+5,650/−1,959），在 `1cb735d9` 的副本上（`git archive`，装好依赖、构建了包）跑过每一道门禁：关键词 68 条、3 个例外，没有命中；turbo 的 `check:types`、`check:lint`、`check:format`、`check:sync` 54/54（web 349、`propel` 11，都等于上限）；改到的 TS 文件不在任何包的 oxlint 清单里；knip 空；vitest 16/16，web 93 个文件、820 个；`make build` 加完整的 e2e，105 个通过，只有 S3 失败（F4）。它另跑了自己的变异和探查：RV1（cycles 页和 automations 页的门放宽到项目的成员）存活；RV2（`utils/src/project.test.ts` 中一个非空断言）存活；PR1（P10 的弹窗中组字的 Escape，对真的 nerve）。它还核对了：
- 七类缺陷逐处清扫（请求体、错误、所持的值、关闭、被拒绝时留着、忙、会话），每处与 spec 相同；2.13 的 12 行"跟进的"与代码相同，各有它自己的换账户之后兑现的测试；
- 权限表：每个角色在每一页被提供什么，对照 nerve 的 `decide.go`、`rules.go`、`membership.go`、`member.go`：提供的没有不一致；没有会失败的检查的只有 features、automations 页的门（I1）和扩展侧边栏的拖动（P11）；
- 挂载时的请求逐页、逐个角色，与 2.12 和 S2 的 `REQUESTS` 相同；
- 共用部分的约定：`CustomSearchSelect` 调用方传的每个 prop 都起作用（`onClose` 在开着时卸载也调一次，第 3 节第 15 条没写）；成员下拉框的约定照 A-M5；`answeredAt` 在每种先后下都对，是"这个客户端最后写下的回答"（R4 接受的定义）；`followInSession`、`useRefusalToast`、`useCopyLink` 处处一种用法；
- 删除：deadsym 在 HEAD 上没有新的死的生产符号；删掉的名字没有留下；按完整的键路径没有孤儿的文案键；开关见 M6；
- 重复：M5 的两处；
- 文档对照代码：M2、M3 和"M4–M6"不真；README、前端改动清单、M2 交接和 M1-P4 交接的结果、v0 7.7 的 `answeredAt` 一句为真。

| 发现 | 处理 |
|---|---|
| **I1** features 的四页和 automations 页的门（`allowPermissions([ADMIN], PROJECT)` → `NotAuthorizedView`）没有会失败的检查：没有 vitest 渲染这几页，P3 只以管理员打开它们；RV1 把两页的门放宽到项目的成员，vitest 820/820 和完整的 e2e 都通过。门今天与 nerve 的 `project.update` 相同 | F-2，`d3ef10f5`；`admin-pages.test.tsx`；`FW3.rv1`、`FW3.wide-*`、`FW3.gate-*` |
| **M1** `@nerve/utils` 的 `project.test.ts`（Task 4 新写的）不在非空断言的范围，2.14 也没列（RV2 存活） | F-7，`6021048b`；`FW7.rv2` |
| **M2** spec 第 5 节收尾说 API 令牌弹窗和命令面板的复制没有失败的一支，两者都有 | F-9，`28526539` |
| **M3** spec 第 5 节 M4 一行说 `issues/` 下 7 个文件自己写复制，是 8 个；写出它们，第 3 节第 8 条的 21 个才都有落点 | F-9 |
| **M4** spec 第 5 节只把成员下拉框改了的约定交给 M4，M7 收集箱的两个指派人也用它 | F-9 |
| **M5** 两份副本：`underFields` 在两个测试里；`use-project-actions.ts:29-32` 自己再拼一份 `useCopyProjectLink` 的链接和文案 | F-6（`underFields`，`667ba1e4`）；F-5（`d51a1678`，`use-project-actions.test.tsx`，`FW6.copy-*`） |
| **M6** 两个侧边栏的 `disableDrag`、`disableDrop` 总是 `false`，是开关 | F-4，`2bfde3d5`；`T10.12` 不适用 |
| 搁置项说得太窄：P2 只写 `useKeypress`，PR1 中 `ModalCore` 的弹窗（删除弹窗关了）和表情选择器也关；P4 的 2.13 一项漏了加入对话框的 `setIsJoiningLoading(false)`、包装层的 `setIsJoiningProject(false)`、自定义时长的 `isSubmitting`；P4 漏了 T11-b 交来的"M4–M6 … 14 个文件"，同样的字在 M3 设计 7.7、9.7 C7、13.1，spec 的改动够不到 | F-3（P2）；F-9（P4 的几项和设计的三行） |
| 照问划定 P2、P7、P11：P2 的根在三个处理器（`useKeypress`、`ModalCore`、propel 的 `EmojiPicker`），测试在 P1、P4；P7 是 P10 的七处，另列出 P11、M4、M6、M7 和收尾的地方（不传工作区和项目的 `allowPermissions` 退回 router store，成员 store 的 `projectMemberIds` getter 也算）；P11 不要检查也不要只由评审看住，删除开关（M6） | F-3；F-2 和 spec 第 5 节；F-4 |
| 第 5 节的交接：P11 还缺 `project.label_name_taken` 和 P7 的地方；M7 缺成员下拉框的改动（M4）；P2 修在共用的 hook 和组件时，它们在 P10 之外的使用者只要一行 W17 | F-9；第 3 节第 15 条写出 `useKeypress`、`ModalCore` 的其余使用者（复审的 M1 补全） |
| 过时的句子：M8 的 M1 收尾交接 `:53`（表情选择器读 jsDelivr）自 Task 1 起不真 | F-9：那里写"处理结果（M3/P10）"，交接保持 `open` |
| 完成线：除 C6–C8（控制者的）都做到 | C6–C8 在第 2 节 |
| 没有判断的：C6–C8 和 `HEAD`、`ETag`；spec 的数目；`ModalCore` 上 `preventDefault()` 是否不碰 Chromium 中输入法的组字（要真的输入法）；成员下拉框在 M4–M7 页面上的样子 | `HEAD` 在第 2 节的 C6；数目由 F-9 重数；真的输入法和 Safari 进第 7 节和 spec 第 5 节收尾；样子由 M4、M6、M7 行带去 |

评审没有提到的搁置项：P1 记在第 5 节；P3 由 F-9 并进收尾；P5、P9 由 F-6；P6、P8 由 F-5；P10 由 F-1。

**修复轮**（一次分派，`1cb735d9` 之上，十一个提交）：
- **卡片的测试照 `ContextMenu` 读菜单**（`9c3a4e10`，F-1）：`item.shouldRender !== false`；没有一行的结论改变（没有藏起的提供）；`FW1.1`（已归档的卡片的恢复对 nerve 不让恢复的人是 `undefined`）由 card.test 发现。
- **测试的重复**（`667ba1e4`，F-6）：新文件 `core/store/member/fake-users.ts` 的 `userOf(name)`，`fake-members.ts` 的 `membershipOf` 用它；`fake-projects.ts` 的 `projectMemberDetailsOf`（三个测试用）；`project-roles.test.tsx` 用 `fake-tab`；`fake-permissions.ts` 导出 `Caller`、`its`（卡片、成员页、成员默认值和第 3 步的测试用）；`fake-refusal.ts` 的 `underFields(setError)`；`use-create-workspace.test.ts` 换账户之后的一行是 `it.each(lateSettlingsAnswering(acme))`（`FW2.1`）；`shownNameOf` 移进 `e2e/fixtures/auth.ts`（A6、`assert/identity.ts`、P3、P5 用）。没有行为的改动：9 个改到的文件的 106 个测试名字和结论不变（`vitestcmp.py`），A1、A6、A17、P3、P5 通过。
- **项目的设置页从路由读地址**（`d3ef10f5`，F-2）：七处读路由参数，`allowPermissions` 明写工作区和项目；`project-member.store.ts` 新的 `getFilteredProjectMemberIds(projectId)`（成员页的筛选和排序），`projectMemberIds` getter 以 router store 的项目调它，store 测试 2 行；关键词规则 `project-address-route`（69 条，样例的不命中中有状态、标签页），它的 `why` 说之后的阶段扩大它的文件；新的 `admin-pages.test.tsx`（25 个，经 `createRoutesStub`，不用退路）；web 的上限 349 → 348；S2、P3 各 3 次通过，挂载清单不变。变异 `FW3.*`。
- **组字的 Escape 不关弹窗和表情选择器**（`99cfabd7`、`77175dae`，F-3）：`use-keypress.tsx` 在 `event.isComposing` 时返回；`modal-core.tsx` 面板的 `onKeyDown`（冒泡阶段）对组字的 Escape 阻止默认，一句注释说为什么不在捕获阶段；`emoji-picker.tsx` 在 `handleToggle(false)` 之前返回。e2e 在已有的测试里（仍是 106 个）：`settings-pages.ts` 加 `composingEscape`、`closedByComposingEscape`、`hiddenWithin`；P1 的名称框和选择器的搜索框，负责人的测试用 `composingEscape`；P4 在归档之前打开删除弹窗，组字的 Escape 不关，一次普通的 Escape 照旧关上。第二个提交让 `composingEscape` 在一次 `evaluate` 中依次发出组字开始、组字中的 keydown、组字结束，关上选择器的变异失败在"留着"的检查上，而不是失败在字段没了上。变异 `FW4.keypress`、`FW4.modal`、`FW4.picker`、`FW4.modal-capture`。
- **侧边栏的拖动没有开关**（`2bfde3d5`，F-4）：props、`canDrag`、拖动柄的条件和两个调用方的 `false` 删除；`canDrop` 留着自己的条件（另一个项目、`PROJECTS` 的实例）。P8 3 次通过。没有可改的地方，`T10.12`（和 `P10.8`）记作不适用。
- **页头的复制、标识被占用的键、孤儿替身**（`d51a1678`，F-5）：`use-project-actions.ts` 照裁定，它的 `workspaceSlug` prop 去掉（`tab-navigation-root.tsx` 随之改），新的 `use-project-actions.test.tsx`（2 行：开着标签页时复制它的地址；没有时复制项目的工作项页）；`use-update-project-details.ts` 用 `t(PROBLEM_MESSAGES["project.identifier_taken"])`，加一行"检查说被占用时与 nerve 拒绝时同一个说法"；`fake-toast.ts` 的 `setPromiseToast`、`Outcome` 删除。变异 `FW6.copy-tab`、`FW6.copy-project`、`FW6.taken`。
- **utils 的测试在非空断言的范围里**（`6021048b`，F-7）：`FW7.rv2` 让 utils 的 `check:lint` 失败（`mut.py` 加了 `lint:<pkg>` 一层）。
- **变异的记录**（F-8，没有提交）：第 2 节。
- **第 9 步另加的两个提交**：`eee6415c`（2.14 让每次改动到的 nerve 文件进范围，第 6 步改了 `fake-toast.ts`；`FW7.toast` 让 web 的 `check:lint` 失败，没有这一条的对照通过）；`3c974513`（第 4 步让 P1 到 405 行；创建的测试把表单的字段写一份，请求体和数据库的一行共用，回到 398 行；手工的变异（请求体没有说明）在那一处让 P1 失败，P1 3 次通过）。
- **文档**（`28526539`，F-9）：spec 的数目在头上重数（第 2 节总览 157 个文件、第 4 节、A.1、A.2、A.4 第 19、36、45、46 条、A.5–A.8）；新的 2.15 和第 3 节的"修复轮"表；P4 的各项和 M2–M4；凡成员下拉框的调用方写"M4–M6"的改为"M4–M7"（也改 2.11 和 L10 引用的前端改动清单）；第 3 节第 15 条的 `onClose`、F-3、F-4、T5-b、T6-e、T9-a；2.14；第 5 节按阶段的交接（P11、M4、M6、M7 各自扩大 `project-address-route`）、收尾的 `projectMemberIds` getter、T3-a 的工具缺口；第 7 节的 M8 交接。M3 设计 7.7、9.7 C7、13.1 改为"M4–M7"；前端改动清单加组字的 Escape 一行；M8 的 M1 收尾交接写"处理结果（M3/P10）"（`:61` 的关闭条件做到一半，留着标注块的图标，状态保持 `open`）。plan 和 M3 设计 §15 的 P10 一行不改。
- 门禁（代码的头 `3c974513`）：`make lint-web`（69 条规则、3 个例外、没有命中；turbo 54/54；web 348 等于上限，utils 7、propel 11、ui 19、constants 1、types 0、i18n 0；P10 的 116 个 TS 文件 0 条；抑制 183，不变）；`make knip`；`make test-web`（16/16；web 95 个文件、848 个，utils 6 个文件、65 个）；`make e2e` 106 个；P1、P3、P4、P5、P8、S2 各 3 次（P3、P4、P5、P8、S2 在 `6021048b` 上，之后它们的代码没变；P1 在 `3c974513` 上）。
- 实现者写明的偏离，F-10 都接受：
  1. `ModalCore` 用冒泡阶段的 `onKeyDown`，不是捕获阶段：捕获阶段先阻止时，Headless UI 的 `mergeProps` 对默认已被阻止的事件跳过合进面板里控件的处理器，`CustomSearchSelect` 自己的 Escape 处理被跳过，组字的 Escape 传到 `document`，P1 的负责人测试这样失败（`FW4.modal-capture`）。冒泡阶段里面的控件先处理，window 上的 `useEscape` 仍看到被阻止的事件。裁定写的机制不对。
  2. `member-list.tsx`、`member-select.tsx` 不用 `getProjectMemberIds(projectId, …)`：它把调用者排在前面、再按显示名，不看成员页的角色筛选和 `order_by`，成员页的列表会变；改为新的 `getFilteredProjectMemberIds(projectId)`（原来 getter 的算法，按给它的项目），router 的 getter 委托给它（控制者核对：一份实现）；成员的选择照旧自己去掉访客，它列出的与 `getProjectMemberIds(projectId, false)` 不同。
  3. `shownNameOf` 在 `e2e/fixtures/auth.ts`，不在 `project-pages.ts`（A6 和 `assert/identity.ts` 是身份的 fixture）。
  4. 新的 `fake-users.ts`：`fake-projects` 要一个不建 store 的用户，`fake-members.ts` 引入 `UserStore` 和真的客户端，会破坏节点上的测试。
  5. 地址规则也看 general 页（Task 5 起已读参数，不让它退回）；规则不分权限的级别，所以成员页工作区一级的 `allowPermissions` 也传工作区和项目（工作区一级不读项目，多传无害）。
- 关注五条（F-10 的处理在第 3 节）：(1) 地址规则的模式在范围之外命中 109 个文件；(2) 超过约 400 行的两个原有文件（`assert/identity.ts` 472、`api.ts` 404），另有 Plane 的 `projects-list-item.tsx` 460 → 452；(3) 设计中"14 个文件"是预检的数，评审数出 12 个；(4) 后台的一次跑中两次假的发现（已重跑），之前各 Phase 的记录也许有；(5) W18 的基础副本可以删。

**修复的复审**（opus，`1cb735d9..28526539`，十一个提交、54 个文件）：通过，Critical 0、Important 0、Minor 2。
- 它在副本上（`git archive`，`--frozen-lockfile` 安装，构建了包）：
  - 跑过门禁：关键词 69 条、3 个例外，没有命中；turbo 54/54，web 348，其余各包都等于上限；knip 空；vitest 16/16，web 95 个文件、848 个，utils 6 个、65 个；完整的 `make e2e` 105 个通过，只有 S3 失败（F4）；重新构建之后 P1、P3、P4、P5、P8、S2 各 3 次，12、15、6、6、6、30 个通过，没有失败或偶发；修复轮手改的 47 个 TS 文件和 P10 的 116 个 web、19 个 e2e 的 TS 文件 0 条；修复轮加的行没有新的 `as`、`any`、`!`、抑制；
  - 用 BASE 和 HEAD 的 JSON 报告比较十个测试文件：只有 F-5 的新一行不同；
  - 抽查变异的记录（第 2 节）：可信；spec 的每个数目与它重数的相同。
- 逐条裁定：F-1–F-8 在根上做到；五处偏离成立（`getFilteredProjectMemberIds` 是原 getter 的函数体，一份实现；冒泡阶段的 `onKeyDown` 只碰组字的 Escape，让里面的控件先处理，在它之后读 `defaultPrevented` 的只有 Headless UI 的 `useEscape` 和只看指针的 `use-outside-click`）；F-9 除下面两条都做到。
- 两条 Minor：
  - **M1**：spec 第 3 节第 15 条（F-3 的 W17 一行）说 `ModalCore` 的其余调用方是 M4–M8 的弹窗；它们还有 M2 的（停用账户、API 令牌、个人资料）、P9 的（删除工作区、移出成员、邀请）、`project-navigation-dialog.tsx` 和每个 `AlertModalCore` 的使用者。照写，就没有人会在浏览器核对或收尾中看 M2、M3 的弹窗。
  - **M2**：两处数目：2.6 说"四个功能页各一个 `FW3.wide-*`"，是三个，cycles 页的是 `FW3.rv1` 的；A.2 说 35 个别名都是修正轮把探查收进 plan 的表，是 31 个，`PF6`、`PF15` 是预检的行，`P9.14` 与 `T9.11`、`W18.23` 与 `P8.10` 相同。
- 说明（不计入）：没有测试用真的输入法，e2e 发出的合成事件照 Chromium 的顺序；Safari 在那个 keydown 之前结束组字、`isComposing` 为假，只查 `isComposing` 的守卫（仓库在六处的写法，也是裁定选的）在那里拦不住，2.15 和前端改动清单只对 Chromium 为真，MDN 的写法是 `event.isComposing || event.keyCode === 229`；`project-member.store.test.ts:43` 的 `profile = (name) => membershipOf(name).member` 早于修复轮，现在换一条路到 `userOf`，可以顺手整理，不是新的重复。
- 结论：浏览器核对之前，F-10 计划中的跟进提交一并补上两条。

**F-10、F-11 的跟进**（`621c9390`，spec、M3 设计、前端改动清单、`base.test.tsx` 的注释，都是 P10 已改的文件）：
- 成员下拉框的调用方："14 个文件"改为"12 个文件（M4 8、M6 2、M7 2）"，在 M3 设计 7.7、9.7 C7、13.1 和 spec 的 2.2、第 3 节第 2 条、预检的 L1 一行；第 3 节第 2 条和第 5 节收尾的设计一条记下这次改动。`base.test.tsx:11` 的注释写"M4–M7"。
- 复审的 M1：第 3 节第 15 条照 `git grep -lw ModalCore` 和 `AlertModalCore` 写出 `ModalCore` 的其余使用者（M2 的；P9 的，含项目导航；每个 `AlertModalCore` 的确认框；M4–M8 的）。复审的 M2：2.6 写 `FW3.rv1`（cycles 页和 automations 页一起）和三个 `FW3.wide-*`；A.2 照 `mutants_p10_record.SAME` 分开 35 个别名。
- 第 5 节收尾加四条：router store 的其余读法；拆分 `assert/identity.ts`；用 `e2ekills.py` 重查之前各 Phase 的端到端的发现；输入法的守卫（9 处只查 `isComposing`：Enter 的 5 处，`issues/` 下的三个标签选择和 web、`@nerve/ui` 各一个 `useDropdownKeyDown`；Escape 和其他按键的 4 处，`useKeypress`、`ModalCore`、`CustomSearchSelect`、表情选择器）。A.6 记下 `api.ts` 404 行按约 400 行接受。2.15、第 3 节第 15 条和前端改动清单写明是 Chromium 的顺序，Safari 的在第 5 节。
- 改正关注 1：`routerkinds.py` 按命中的是什么分：55 个从 router store 读项目（26 个已在 P11、M4、M6、M7 各行和 `projectMemberIds` 一条中；其余 29 个都是不传项目的项目级 `allowPermissions`，`issues/` 下 16 个，`cycles/`、`modules/`、`views/` 和 power-k 的迭代、模块命令 11 个，`labels/` 2 个，没有阶段的行写它们）、36 个只读工作区（工作区的页面和组件，和 4 个 store）、18 个不是读（同名的局部变量、props、参数，getter 自己的定义，测试的替身，传了工作区的工作区级调用）。
- 门禁：`make lint-web`（69 条、3 个例外、没有命中；turbo 54/54；web 348）、`make knip`、`make test-web`（web 95 个文件、848 个）；不改故事，没有跑 e2e。

**F-12**（`093d0d91`，只改 spec）：M4 行写出 `issues/` 下的 16 个文件，M6 行写出 11 个（路由 `modules/(list)/header.tsx`，`cycles/`、`modules/`、`views/` 和 power-k 的迭代、模块命令），P11 行写出 `labels/` 的 2 个，都带行号（`$M3TMP/p10tools/sitelines.py`）；收尾一条：55 个读项目的文件都在一个阶段的行或 `projectMemberIds` 一条中，跨 Phase 的决定只留给 36 个只读工作区的，18 个不是读的不变；状态行和第 3 节的跟进一段写 F-10–F-12。门禁 `make lint-web`（关键词守卫扫每个被跟踪的文件，spec 在内）：69 条、3 个例外、没有命中；turbo 54/54。

**两个跟进提交的范围复审**（sonnet，`28526539..093d0d91`）：通过，没有发现。它核对了成员下拉框的 12 个文件、9 处输入法的守卫、`ModalCore` 的使用者、35 个别名和 F-12 的全部 29 个地方（30 行）。

**浏览器核对之后的修复**（裁定 F-13–F-15，修复轮的实现者，`093d0d91` 之上，五个提交）：
- `d30199b2`（F-13）：`CustomSelect` 把 `tabIndex` 给按钮（两支），不给 Listbox 的根；包网络的 `div` 不收它；P1 从说明按 Tab 到"Public"按钮，空格打开。
- `b6344182`（F-14）：`ContextMenu` 的层 `inert={!isOpen}`；这个文件原有的 4 条 oxlint 警告清零（`useCallback`、`useMemo`），`ui` 的上限 19 → 15；P2 经卡片的菜单（唯一不 `inert` 的一个）复制链接。
- `6db732a1`（F-15）：创建表单没有正的 `tabIndex`（字段、网络、负责人、取消、创建、页头的关闭）；`PROJECT_CREATE` 的表和只为它传的三个 `isMobile` props 删除；P1 的弹窗测试在后面有项目卡片时走一圈 Tab（`tabCycle`，`project-pages.ts`）；`base.test.tsx` 加一个测试（调用方的 `tabIndex` 在按钮上，选择自己的是 −1）；P1 是 394 行（Ops、Docs 的回答经它们的行写）。
- `13388d2f`：`tabCycle` 交给 `page.evaluate` 的两个句柄是一个对象，不是加 `as const` 的元组。
- `9b3ff184`：文档：spec（2.2–2.4、2.11、第 3 节第 3 条、重写的第 12 条、第 15 条、"控制者的浏览器核对之后"一表、第 4 节、第 5 节 M4、M6、M7 各行和收尾的 `ModalCore` 一条、A.1、A.2、A.4–A.7、状态一行）和前端改动清单（三行）。
- 门禁（代码的头 `13388d2f`）：`make lint-web`（69 条规则、没有命中；turbo 54/54；web 348、`ui` 15）；`make knip`；`make test-web`（16/16；web 95 个文件、849 个，utils 6 个、65 个）；`make e2e` 106 个（`b6344182`、`6db732a1` 上也是）；P1、P2、P4 各 3 次（`6db732a1` 上，27 个），P1 在 `13388d2f` 上再 3 次（12 个）；P10 的 119 个 TS 文件 0 条警告。
- 实现者的关注：`FW13.root` 是记录中唯一没有被发现的变异；P10 一侧要一个检查，只能是渲染真的 `CustomSelect` 的 web 的 vitest，那会是第一个引入真的 `@nerve/ui` 的 web 测试（裁定 F-16，第 3 节）。缺陷 ii 没有改，写进 spec 第 5 节收尾（之后改为负责人的裁定）。

**第二次复审**（opus，`093d0d91..9b3ff184`）：Critical 0、Important 0、Minor 4，没有代码的缺陷；结论"一个文档的跟进之后可以合并"，M1、M2 必须做，M3、M4 小，可以一起。
- 它在副本上（`9b3ff184` 的 `git archive`，`--frozen-lockfile` 安装）跑过门禁：关键词 69 条、3 个例外，没有命中；turbo 54/54，web 348、`ui` 15 都等于上限；这一轮改的 13 个 TS 文件 0 条警告，BASE 的 `context-menu/root.tsx` 4 条；knip 空；vitest 16/16，web 95 个文件、849 个；P1、P2 各 3 次，P4 一次，12、9、2 个通过。
- 逐条裁定在根上做到：
  - F-13：20 个 `CustomSelect` 调用方的文件中只有 `ModuleStatusSelect`（`modules/select/status.tsx:53`）给 `tabIndex`，其余 19 个不失去什么（根上原是 `tabIndex={undefined}`），M6 的模块状态从"正的位置上一个不能用的 `div`"变为"按钮在正的位置上"；Headless UI 2.2.10 的 `ListboxButton` 不设自己的 `tabIndex`。
  - F-14：11 个 `ContextMenu` 的使用者（M3 的项目卡片；M4 的 `quick-action-dropdowns/` 六个和草稿的；M6 的迭代、模块、视图）都不传 `portalContainer`；右键、窗口上的按键、嵌套的子菜单照旧；`useCallback`、`useMemo` 不改行为。
  - F-15：创建表单下没有 `tabIndex`；`PROJECT_CREATE` 和三个 props 没有读者；手机上不变（`getTabIndex` 在那里本来就给 `undefined`）；P1 的一圈就是 DOM 的顺序。
- P1 的 Tab 一圈和 P2 的菜单一步因为对的理由失败（第 2 节的变异记录）。P2 读剪贴板可以改用 `expect.poll` 让读法明确，是可选的，没有改。
- 一处副作用是改进：点了菜单的项之后，菜单关上，获得焦点的项变为 `inert`，浏览器把焦点移到 `body`；之前焦点留在看不见的按钮上，之后的空格或 Enter 会不声不响地再做一次那一项。

| 发现 | 处理 |
|---|---|
| **M1** 缺陷 ii 写成仍等负责人决定（spec 的状态一行、"控制者的浏览器核对之后"一表之后的一段、第 5 节收尾一条），负责人已裁定放进 M3 收尾 | F-17，`456904f4`：三处照复审的文字改 |
| **M2** F-16 没有写进 A.2（"只由评审看住的"中没有 `FW13.root` 和它的理由），第 5 节 M6 一行也没写 M6 删去模块表单的正的 `tabIndex` 之后这个属性随之删除 | F-17：A.2 的记录一句和"只由评审看住的"、F-13 一行、第 1 节、M6 一行；记录的说明也写（重新生成，288 / 251 / 250） |
| **M3** 正的 `tabIndex` 的交接不全：`baseTabIndex`（桌面上是 1）的三个读者（两个经 `ModalCore` 的弹窗的搜索框 `existing-issues-list-modal.tsx:156`、`parent-issues-list-modal.tsx:101`，和 `label-select.tsx:162`），`CustomMenu` 把 `tabIndex` 给 Menu 的根（`custom-menu.tsx:225`，调用方是 M4 的 `default-properties.tsx:262` 和 M7 收集箱的 `issue-properties.tsx:164`）；两个弹窗的照机制推出，没有跑 | F-17：写进 M4、M7 行；第 3 节第 12 条写明这些都删去之后 `getTabIndex` 和两个 `tab-indices.ts` 随之删除 |
| **M4** `ETabIndices.CREATE_LABEL` 和它的表（`tab-indices.ts:43`、`:63`、`:72`）自 Plane 的导入起没有读者（knip 看不到枚举的成员和表的键），第 3 节第 12 条"`getTabIndex` 和其余的表留着，工作项、收集箱、迭代、模块、视图的表单用"对它不真 | F-17：三行删除，`tab-indices.ts` 72 行 |

**复审之后的跟进**（同一个实现者）：
- `456904f4`（F-16、F-17）：上表的四条。门禁 `make lint-web`（69 条、54/54；web 348、`ui` 15）、`make knip`、`make test-web`（16/16）。实现者另报告功能一步的正的 `tabIndex` 和 P10 之外写死的几处（裁定 F-18，第 3 节）。
- `03157319`（F-18）：`project-feature-update.tsx` 删去 `tabIndex` 1、2 和两处抑制（65 → 61 行，抑制 183 → 181）；P1 的创建测试从功能一步的 Close 按 Tab 走一圈，都在模态的弹窗里：Open project、四个开关、Close；P1 399 行。`FW18.index` 由 P1 发现（Tab 走到提示和侧边栏），不让任何 vitest 失败；记录 289 行、跑 252 个、发现 251 个。门禁：`make lint-web`（69 条、54/54）、`make knip`、`make test-web`（16/16；95 个文件、849 个；utils 6 个、65 个）、`make e2e` 106 个、P1 3 次（12 个）。
- `3e9d0fe9`（F-18 的文档）：spec 2.4、第 3 节第 12 条、"F-17 之后"一段、第 4 节、A.1、A.2、A.5、A.6、A.7，文件数 162；前端改动清单一行；第 5 节交出 P10 之外的六处写死的正的 `tabIndex`（M4 2 处、M6 2 处、收尾 2 处）。之后再跑 `make lint-web`。

**三个提交的范围复审**（sonnet，`9b3ff184..3e9d0fe9`）：通过，没有发现。第二次复审的 M1–M4 和 F-18 都做到；只改了预期的 5 个文件；`FW18.index` 在 P1:313 失败，理由对（Tab 离开了弹窗）；用 `suppressions.py` 的写法核对抑制是 181；六处写死的 `tabIndex` 都在写出的行上。

## 5. 最终实现与 plan 的差异

plan 的块记录的是当时的文字。执行中 plan 只在各 Task 的修正轮中改（9,718 → 9,803 行），修复轮和浏览器核对之后的几轮不改它：
- 每个修正轮和评审之前的一轮（`86fe7e11`、`3fee585b`、`8ac39a87`、`6a028754`、`b05b42bb`、`45736939`、`3ff60245`、`fb38b1a9`、`cc1ee856`、`bd151a8a`）在它的 Task 的变异表加它的行和一段说明（`T2.16`–`T2.18`、`T3.13`–`T3.17`、`T4.12`–`T4.14`、`T6.8`、`T6.9`、`T7.18`–`T7.27`、`T8.12`–`T8.18`、`T9.14`–`T9.23`、`T10.8`–`T10.14`；`T4.6` 一行加 `lead-candidates.test.ts`）；改到 plan 之外的文件的，在文件表加上它们；
- `86fe7e11`：Task 4 中 P1 的导入块加 `keydownsReachingDocument`，Task 8 的 `fake-controls.ts` 块留着 `customButton`，让之后的块照样能应用（在副本上照 plan 应用 Task 3–11 核对过）；
- `3fee585b`：全局约束的"过渡版本"一条中 `settings-pages.ts` 加 Task 3。

各 Task 自己的文字和修复轮、浏览器核对之后的几轮的不同不改（搁置项 P1；T1-d、F-9，P8b 的 F-4 先例），spec 在说规则的地方已改为与代码一致（F-9、F-11、F-12，浏览器核对之后的 `9b3ff184`、`456904f4`、`3e9d0fe9`）。与 plan 不同的地方都来自第 3、4 节的裁定：
- **生产代码**：
  - T1-c：`emojibase.ts` 的版本注释说 catalog 固定它；
  - T2-a、T2-b、T2-c、T2-e、T2-f：`custom-search-select.tsx` 的面板收 `cn("z-30", optionsClassName)`，`<ul>` 不再收；两支按钮加 `KEYBOARD_FOCUS`（`data-focus:` 的轮廓）；根的 `flex-shrink-0` 改名 `shrink-0`；面板的 Escape 处理先 `stopPropagation`，组字时不关列表；注释照实。成员下拉框 `base.tsx` 的选择按钮加 `hover:bg-transparent`，根传 `shrink`；
  - T3-a：两种语言的顶层键 `link_copied` 删除；T3-f：`project.store.ts` 中 `getProjectById` 的注释；
  - T4-b：负责人的候选抽进 `projects/create/lead-candidates.ts` 的 `leadCandidates`；
  - T7-a：`PROJECT_ROLES` 的注释照实；
  - T10-b：`tab-navigation-overflow-menu.tsx` 的注释照实；
  - 其余几轮不改生产代码：Task 5 没有修正轮；Task 6、8、9、10 评审之前的一轮和 Task 8 的修正轮只改测试和它们的替身、`.oxlintrc.json` 的范围、plan 和 spec；
  - 修复轮：F-2（七处读路由参数；`getFilteredProjectMemberIds`；规则 `project-address-route`；上限 348）、F-3（三个处理器）、F-4（删除两个开关）、F-5（页头的复制、标识被占用的键）、F-7 和 `eee6415c`（`.oxlintrc.json` 的范围）。关键词规则 68 → 69 条。
  - 浏览器核对之后：F-13（`@nerve/ui` 的 `custom-select.tsx` 两支都把 `tabIndex` 给按钮，Listbox 的根不收；`projects/create/attributes.tsx` 包网络的 `div` 不收）；F-14（`context-menu/root.tsx` 的层 `inert={!isOpen}`，`handleClose` 经 `useCallback`、上下文的值经 `useMemo`，`ui` 的上限 19 → 15）；F-15（创建表单的五个文件，`projects/create/` 的 `attributes.tsx`、`root.tsx` 和 `project/create/` 的 `header.tsx`、`common-attributes.tsx`、`project-create-buttons.tsx`，没有正的 `tabIndex`；`@nerve/constants` 的 `PROJECT_CREATE` 和页头、按钮、`attributes.tsx` 只为它收的 `isMobile` 删除）；F-17（`CREATE_LABEL` 的三行删除）；F-18（`project/project-feature-update.tsx` 的 `tabIndex` 1、2 和两处抑制删除）。
- **测试**：plan 之外加的：
  - 各 Task 的修正轮：
    - T2：P1 数传到 `document` 的 Escape（`keydownsReachingDocument`，代替靠时间的可见性检查），核对 Escape 之后选择按钮的轮廓；`base.test.tsx` 的第二个测试渲染替身收到的 `customButton`；`transitionsEnded` 只等有限的动画；
    - T3：P2 卡片故事中扣住的加入和被拒绝的加入，`settings-pages.ts` 的 `closedWithin`；`use-project-fetch.test.ts` 决定表加 3 行；`project.store.test.ts` 加 2 个；
    - T4：`fake-tab.ts` 的 `lateSettlingsAnswering`，创建的换账户两行答以项目；`lead-candidates.test.ts`（4 行）；P1 扣住时"Creating"禁用，`named` 移到 `auth.ts`；
    - T6：`auto-archive-automation.test.tsx` 加 4 个（`done` 的交接）；P3 的"Submitting..."；
    - T7：`project-roles.test.tsx` 加两行和 3 个；新的 `member-list-item.test.tsx`（8 个）、`useProjectColumns.test.tsx`（3 个）；P5 的 `closedWithin` 和"Leaving..."；
    - T8：添加被拒绝之后再提交同样的请求体；换账户的两行答以成员关系；成员默认值的换账户 5 → 9 个；新的 `member-list.test.tsx`（5 个）；P5 的"Add members..."；成员默认值按 store 自己的权限 5 行；新的 `store/user/fake-permissions.ts`；`fake-controls` 的按钮替身留文字、开关替身留 `disabled`；
    - T9：`archive-restore-modal.test.tsx` 答以项目、加 1 个（`vi.mock("react")`）；`delete-project-modal.test.tsx` 停在原页的一条变为两种；新的 `card.test.tsx`（12 个）；`fake-controls` 的 `ContextMenu`；P4 的两个忙、wes 在设置页、只填了名称时不可用；
    - T10：`use-tab-preferences.test.ts` 的 `made()` 和五个修改的次序；新的 `tab-navigation-overflow-menu.test.tsx`（2 个）；`fake-controls` 的 `Menu`、`buttonsIn`；P8 中的访客 Gus；api.ts 的 `changeWorkspacePreferences`（交回 nerve 的回答，P8、W3、W8、`furnishWorkspace` 共用）；`dragOnto` 等过渡结束。
  - 修复轮：card.test 的筛法；`fake-users.ts`、`projectMemberDetailsOf`、`Caller`、`its`、`underFields`、e2e 的 `shownNameOf`；P9 的创建工作区换账户的一行答以工作区；新的 `admin-pages.test.tsx`（25 个）、`use-project-actions.test.tsx`（2 行）；成员 store 的测试 2 行；general 页标识被占用的一行；e2e 的 `composingEscape`、`closedByComposingEscape`、`hiddenWithin`，P1、P4 的组字的 Escape；P1 把表单的字段写一份。
  - 浏览器核对之后：e2e 的 `tabCycle`（`project-pages.ts`）；P1 在创建弹窗里走一圈 Tab（后面有 Ops 的卡片），从说明 Tab 到网络的按钮、空格打开，F-18 加功能一步的一圈；P1 的接口测试经 Ops、Docs 的行写它们的回答；P2 经卡片的右键菜单复制链接；`base.test.tsx` 加一个（调用方的 `tabIndex` 在按钮上，选择自己的是 −1），`fake-controls.ts` 的带搜索的选择的替身留下 `tabIndex`。
  - 数目：vitest 在 `6c1a090a` 上 75 个文件、657 个；原型 87 个、761 个；Task 11 结束时 93 个、820 个（修正轮加的 6 个文件、59 个）；修复轮之后 95 个、848 个；浏览器核对之后 95 个、849 个。e2e 从 Task 10 起 106 个，修正轮、修复轮和之后几轮的检查都加在已有的故事里。
- **plan 自己的文字**（搁置项 P1，记下、不改）：
  - Task 1 的块中 `emojibase.ts` 的注释是修正之前的（"its package.json pins it"，`80584941` 改了）；
  - 第 15、44、139 行说锁文件多"两处 importer 的条目"，其实一处是 catalog 的条目、一处是 web 应用 importer 的（T1-d）；
  - Task 2–4、6–10 各自的块是修正轮、评审之前的一轮之前的代码（之后的块都在副本上核对过照样能应用），修正轮的改动只记在各 Task 变异表之后的说明里；Task 3 的修正轮改的 P2 的故事标题不在块里；
  - 全局约束的上限链停在 349（修复轮之后 348），写其余各包不变（F-14 之后 `ui` 19 → 15），删掉的抑制写两处（F-18 又删两处，185 → 181）；文件大小的清单是原型的（`api.ts` 390，现在 404；P1 397，现在 399；P5 396，现在 400）；
  - Task 4 中 `attributes.tsx` 照旧用 `getTabIndex(ETabIndices.PROJECT_CREATE, isMobile)`，负责人留在它正的 `tabIndex` 上（第 3244 行的说明，第 3836、3840 行的块），F-15 删去了创建表单的每个正的 `tabIndex`；Task 4 的变异表的 `T4.10` 不适用，`T4.5`、`T4.11`、`T4.14` 改的行变了（记录写两种版本）；
  - 成员下拉框的调用方在第 7、700 行写"M4–M6"，第 1219 行（`base.test.tsx` 块的注释，代码中已改为 M4–M7）和第 9763 行（Task 11 的前端改动清单一行，T11-b 改为 M4–M7）也是；
  - Task 11 的块中实现者照 HEAD 改的九处和 T11-b 的一轮；
  - 修复轮改到的 54 个文件不在 plan 的文字里；其中 25 个（新文件 `admin-pages.test.tsx`、`use-project-actions.test.tsx`、`fake-users.ts` 在内）也不在 plan 的文件表里，列在 spec 第 2 节。各 Task 的修正轮另改的 12 个文件，修正轮当轮加进了 plan 的文件表。浏览器核对之后的几轮另改的 5 个文件（`custom-select.tsx`、`context-menu/root.tsx`、`@nerve/ui` 的 `package.json`、`tab-indices.ts`、`project-feature-update.tsx`）也不在 plan 的文件表里，列在 spec 第 2 节。
- **文件大小**：
  - 超过约 400 行的有三个，都是原有的文件：`e2e/fixtures/assert/identity.ts` 472 行（M2 的 fixture，P10 之前 471 行；F-6 让它用共用的 `shownNameOf`，多一行导入；拆分交收尾，裁定 F-10）；Plane 的 `workspace/sidebar/projects-list-item.tsx` 452 行（P10 之前 460 行，F-4 删除两个开关）；`e2e/fixtures/api.ts` 404 行（`6c1a090a` 上 306 行，Task 3、8 的帮助函数到 390 行，Task 10 评审之前的一轮和修正轮到 404 行；按约 400 行接受，裁定 F-10，照 P9 的 W1 403 行的先例，P9 的裁定 F-10）。
  - e2e：P5 400 行、P3 399 行、P1 399 行（修复轮加的组字的步骤让它到 405 行，`3c974513` 回到 398 行；浏览器核对之后加了 Tab 的一圈和网络的选择，Ops、Docs 的回答经它们的行写，394 行；F-18 加功能一步的一圈，399 行）、`assert/project.ts` 397 行、W3 393 行、`settings-pages.ts` 387 行（都是原有的故事和 fixture，P10 加了页面版本和行）；P2 280 行，`project-pages.ts` 62 行。
  - web：`project.store.ts` 377 行、`card.tsx` 327 行、`project/form.tsx` 292 行（原来 411 行）；新文件中最长的是 `use-update-project-details.test.ts` 151 行。
  - 变长的 Plane 文件（spec A.6）：原型的 22 个中 21 个是本 Phase 的对象，只为使用方改到的是 `emoji-icon-picker/index.ts` 11 → 12（裁定 R5）；修复轮变长的都是它的对象（features 的四页、automations 页、members 页、`member-list.tsx`、`project-member.store.ts`、`use-keypress.tsx`、`emoji-picker.tsx`、`modal-core.tsx`），行为改在它们里面；浏览器核对之后变长的 `@nerve/ui` 的 `custom-select.tsx` 153 → 155、`context-menu/root.tsx` 243 → 250（`inert` 和清零警告的 `useCallback`、`useMemo`）也是；创建表单的几个文件变短，`@nerve/constants` 的 `tab-indices.ts` 90 → 72（F-15 删去 `PROJECT_CREATE`，F-17 删去 `CREATE_LABEL`），功能一步的 `project-feature-update.tsx` 65 → 61（F-18）。
  - `tools/keywords.json` 2,307 → 2,338 行（Task 7 的 `project-leave`）→ 2,387 行（修复轮的 `project-address-route` 和样例）；两种语言的 `common.json` 532 → 527 行。

## 6. 移交事项

本 Phase 关闭的（spec 第 7 节）：
- M2 收尾交接第 10、11 节（至此关闭）；第 14 节随本评审的 C7、C8 关闭（第 2 节：C6–C8、补充的核对和修好之后的再核对）；M1-P4 交接的离开项目的顺序（交接在 M3 的收尾改为 `closed`）。
- P8b review 第 6 节 P10 一行（其中 P7 交 P11、P9 交 M4，见 spec 第 5 节）；P9 review 第 6 节 P10 一行；P8a 交给 P10 的行；P4a review 的 P10；P5b 交给 P10 的行；Codex 设计评审 4.3 第 1、2 条和 5.3 的 M1-P4、表情两句的页面一侧。
- M8 的 M1 收尾交接中表情选择器发往 jsDelivr 的一项（`M1-closeout.md:53`，处理结果写在那里；那一节只剩标注块的默认图标，交接保持 `open`）。
- 每一项的落点见 spec 第 3 节的交接表。

同一个里程碑之内的后续 Phase 和之后的里程碑（写进它们的 spec 或收尾写的交接；spec 第 5 节各行照录，下面是执行之后的补充）：
- **P11（状态、标签的页面与清理）**：
  - spec 第 5 节 P11 一行照录：标签设置页的拖动把手（P8b 的 P7）；标签同名的 409 `project.label_name_taken` 落到名称字段下（T11-a）；状态、标签两页和标签的两个组件从 router store 读地址，P11 改成读路由参数并把这些文件加进 `project-address-route`（F-2、F-12）；P8a、P8b、P9 留给 P11 的照旧。
  - 执行之后加的：P10 定下的写法，P11 的页面照它们写：被拒绝时弹窗留着，只在成功的跟进中关上（A-M2）；页面读地址只从路由参数，`allowPermissions` 明写工作区和项目（F-2）；谁被提供什么由 store 自己的权限的 vitest 钉住（`callerInWeb`、`its`，如 `admin-pages.test.tsx`、`card.test.tsx`）；交回东西的修改换账户之后的测试答以 nerve 的回答（`lateSettlingsAnswering`）；扣住时按钮的忙由 `enabledWithin` 核对，被拒绝之后弹窗留着由 `closedWithin` 核对。
- **M4**：
  - spec 第 5 节 M4 一行照录：嵌套的按钮（裁定 R3）和面包屑之前一节的标题按钮；成员下拉框以 `CustomSearchSelect` 为底之后看得到的两处（按键、`WhileOpen`），点击不冒泡由 Headless UI 做（升级时重看），Escape 之后的焦点，列表的新样子；工作项页的三个标签选择器的取数被拒绝；`user-properties` 的 404；工作项页面的复制（`issues/` 下 8 个文件和 power-k 的工作项命令）；工作项的表单的正的 `tabIndex`（Tab 在最后一个正的之后离开弹窗，P10 的创建表单因此删去了它的，F-15），同一个机制的另几处（F-17：`baseTabIndex` 的三个读者，两个经 `ModalCore` 的弹窗的搜索框和工作项详情的标签选择；父工作项的 `CustomMenu` 把 `tabIndex` 给 Menu 的根）和写死的 `tabIndex={1}` 两处（F-18：归档工作项的弹窗、描述的版本的弹窗）；工作项和草稿的列表的右键菜单关着时 `inert` 之后（F-14），在 M4 的页面上核对右键打开它、点它的项；工作项的页面从 router store 读地址（含 F-12 补上的 16 个文件）；P8a、P8b、P9 交给 M4 的照旧。
  - 执行之后加的：C7 没有核对 M4 页面上的 `CustomSearchSelect` 调用方（成员下拉框的 8 个文件、4 个筛选），它们的页面还不能运行；M4 对接这些页面时逐个用键盘核对。
- **M5**：P10 没有交给它的（spec 第 5 节没有 M5 一行，执行中也没有加的）。
- **M6**：
  - spec 第 5 节 M6 一行照录：迭代、模块、视图的复制；它们的页面从 router store 读地址（含 F-12 补上的 11 个文件）；迭代、模块、视图的表单的正的 `tabIndex`（同 M4）和迭代、模块的归档弹窗写死的 `tabIndex={1}`（F-18）；模块的状态 `ModuleStatusSelect` 给 `CustomSelect` 的 `tabIndex` 自 F-13 落在它的按钮上，在模块的表单上核对，M6 删去模块表单的正的 `tabIndex` 之后它不再收 `tabIndex`，`CustomSelect` 不再读它（F-16）；迭代、模块、视图的列表的右键菜单同 M4；P9 交给 M6 的照旧。
  - 执行之后加的：C7 没有核对 M6 页面上的调用方（成员下拉框的 2 个文件，迭代、模块、视图的页头），M6 对接时核对。
- **M7**：spec 第 5 节 M7 一行照录：收集箱的复制；收集箱的两个指派人的成员下拉框（看得到的不同同 M4 一行，在 M7 的页面上核对，C7 没有核对）；收集箱的页面从 router store 读地址；收集箱的创建表单的正的 `tabIndex`（同 M4），其中父工作项的 `CustomMenu` 的一处落在 Menu 的根上（F-17）；P9 交给 M7 的照旧。
- **正的 `tabIndex` 的交接**分在 M4、M6、M7 和收尾（spec 第 3 节第 12 条）：最后一处删去之后，`getTabIndex`（`@nerve/utils` 的 `tab-indices.ts`）和 `@nerve/constants` 的 `tab-indices.ts` 没有读者，随之删除，`CustomMenu` 不再读 `tabIndex`。
- **M8**：spec 第 5 节 M8 一行照录：Webhook 的密钥的复制；P9 交给 M8 的照旧。M8 的 M1 收尾交接见上。
- **契约**：P10 没有新的。
- **收尾**：
  - spec 第 5 节收尾各条照录：P10 改到的文件中仍写死的英文；复制的另两处（API 令牌弹窗、命令面板）改用 `useCopyLink`，全部改用之后加一条只在 hook 里复制的规则；成员 store 的 `projectMemberIds` getter 在 M4、M6、M7 的读者改完之后删除；从 router store 读地址的其余地方（36 个只读工作区的文件，是否改成读路由参数是跨 Phase 的决定）；拆分 `assert/identity.ts`；用 `e2ekills.py` 重查之前各 Phase 的端到端的发现；输入法的 9 处守卫（Safari 的顺序；共用的判断加 `keyCode === 229` 是跨 Phase 的决定）；`i18norphans.py` 按完整的键路径找读者（T3-a，与 PF-L5、PF16 一起）；角色名只有一个来源（P9 的工作区成员页、邀请行和编辑器的提及卡片仍用英文 `ROLE`）；`webui` 给嵌入的文件加 `ETag`；`@nerve/ui` 下拉框的 `any`；M3 设计 §15 的 P10 一行（本提交改）。
  - spec 第 5 节收尾中浏览器核对之后加的两条：
    - 经 `ModalCore` 的弹窗后面的页面（缺陷 ii）：负责人裁定（2026-10-10）放进 M3 收尾，作为一项独立的任务。`ModalCore` 改用 Headless UI v2 的写法，`<Dialog open={isOpen}>` 加 `DialogBackdrop`、`DialogPanel` 的 `transition`，代替 `Transition.Root`、`Transition.Child`，弹窗打开时后面的页面 `inert`、`aria-hidden`，加一个会失败的检查，全部回归；它涉及 M2–M8 的每个弹窗（`AlertModalCore` 的在内）。`AlertModalCore` 的确认按钮的 `tabIndex={1}`（`modals/alert-modal.tsx:96`）归进这项任务（F-18）。
    - 命令面板的快捷键弹窗（`power-k/ui/modal/shortcuts-root.tsx:86`）的搜索框 `tabIndex={1}`，不属于 M4–M8 的哪一个，由收尾删去（F-18）。
  - 执行之后加的：
    - 浏览器核对看到的：
      - `CustomSelect` 在表单中按 Enter 提交表单：Headless UI 2.2.10 的 `ListboxButton` 在 Enter 上调 `attemptSubmit`，像原生的 select（空格、下箭头打开）；添加成员的弹窗中键盘用户在角色选择上按 Enter，就立即添加。是每个有 `CustomSelect` 的表单的库的行为；`CustomSelect` 是否在 Enter 上打开，与 M2 交接第 14 节的下拉框的说明一起定。
      - 工作区侧边栏的两个展开、收起按钮的名字是没有定义的键 `aria_labels.app_sidebar.close_workspace_menu`、`open_workspace_menu`（`workspace/sidebar/sidebar-menu-items.tsx:51`、`:65`），两种语言都没有这个键（M1 修剪过的 Plane 的侧边栏）。i18n 的检查找孤儿的键，不找读了却没有定义的键：与 PF-L5 的工具缺口和侧边栏一起。
      - 图标选择器设的图标用字体的连字画出，连字的词并进旁边的文字（面包屑中"home Launch"，项目选择的一项"home Launch"，卡片的可访问文字中的"home"）：Plane 的图标渲染，进可访问性的清单。
    - 整分支评审的说明：M4 的 M2 收尾交接 `:90` 的关闭条件（"像 `api-token/modal/generated-token-details.tsx`"那样处理失败）和 M6、M7 的复制一节照写仍为真，可 P10 的第 5 节现在要它们经 `useCopyLink`：收尾写 13.2 的交接时一并改。
    - 第一次复审的说明：`project-member.store.test.ts:43` 的 `profile` 早于修复轮，现在换一条路到 `userOf`，可以顺手整理。
    - C6–C8 当时记作观察的"创建表单中网络的选择有两个不起作用的 Tab 停留点"，补充的核对把它定为缺陷 D1，F-13、F-15 修好，不再交出。

## 7. 已知限制

- **经 `ModalCore` 的弹窗后面的页面对辅助技术可见**（缺陷 ii；负责人裁定放进 M3 收尾，P10 不等它）：弹窗打开时没有一个 `body` 的子元素 `inert` 或 `aria-hidden`，读屏软件仍读得到后面的页面；没有正的 `tabIndex` 的弹窗，Tab 由焦点陷阱守住。原因（spec 第 5 节收尾）：`ModalCore` 用 `Transition.Root show={isOpen}` 包 Headless UI 的 `Dialog`，弹窗打开时才挂载它；Headless UI 2.2.10 的 `Dialog` 把应用的树设为 `inert`、`aria-hidden` 的 layout effect 先于它找到主树所在的 `body` 子元素就运行，之后依赖不变，不再运行（第二次复审对照 Headless UI 的代码核对过）。它涉及 M2–M8 的每个弹窗，到收尾的任务做完之前都是这样。
- **Safari 中组字结束的 Escape**（第一次复审的说明，裁定 F-11）：仓库中查组字的 9 处守卫都只查 `isComposing`（Escape 和其他按键的 4 处：`useKeypress`、`ModalCore`、`CustomSearchSelect`、表情选择器；Enter 的 5 处）。Safari 先结束组字、再交出那个 keydown（`isComposing` 为假），这些守卫在那里拦不住（spec 2.15）。改成 `@nerve/ui`、`@nerve/propel`、web 共用的一个判断（加上 MDN 的 `keyCode === 229`）会改到 M4 的 Enter 的守卫，是跨 Phase 的决定，由收尾定。另外没有测试用真的输入法：e2e 和 C6 照 Chromium 的顺序发出合成的组字事件。
- **表单中的 `CustomSelect` 上按 Enter 提交表单**：Headless UI 2.2 的 `ListboxButton` 的行为，像原生的 select；添加成员的弹窗中在角色选择上按 Enter 就添加（C6 的观察，第 6 节收尾）。
- **表情数据没有 `ETag`**：`webui` 答 `HEAD` 没有 `ETag`、`Last-Modified`，frimousse 在每个新的标签页会话第一次打开选择器时重新下载约 700 KB；文件是 `immutable`，浏览器的 HTTP 缓存通常直接答它，功能不受影响。加 `ETag` 是 Go 的改动，由收尾定。
- **36 个只读工作区的 router store 读法**：router store 由 `StoreWrapper` 的 effect 在渲染之后写入，同一个路由换了地址时晚一次提交（spec 2.6）；没有已知的缺陷由这些读而来。是否改成读路由参数是跨 Phase 的决定，由收尾定（F-10、F-12）。读项目的 55 个各在它们的阶段的行里，到那时之前同样晚一次提交。
- **`answeredAt` 是"这个客户端最后写下的回答"**，不是 nerve 读的先后（R4）：nerve 先读、回答却后到的一份在两次读取之间胜出，下一次取数纠正；项目的几份副本在两次取数之间可能不同（spec 第 3 节第 4 条、第 7 节）。
- **只由评审看住的**（spec 附录 A.2，都不是会话、权限或取数的性质）：被推迟的成员下拉框在指针经过之后换成选择；`CustomSearchSelect` 照 `focusSearchOnOpen={false}` 不取焦点、把 `popperModifiers` 交给 popper；成员下拉框的按钮的点击不冒泡（Headless UI 的 `Popover.Button` 自己做，升级 Headless UI 时重看）；面包屑最后一节的标题是 `<span>`；调用方给列表的 z-index 到达面板（T2-a）；成员下拉框的 `shrink`、`hover:bg-transparent`（T2-c）；`transitionsEnded` 不等无限的动画（T2-d）；再选一次已选的负责人就是没有负责人（T4-d）；`CustomSelect` 把调用方的 `tabIndex` 给按钮、不给 Listbox 的根（`FW13.root` 存活，F-16：P10 的页面上没有给它 `tabIndex` 的调用方，唯一的是 M6 的 `ModuleStatusSelect`；`@nerve/ui` 没有测试的设置；M6 在模块的表单上核对，删去模块表单的正的 `tabIndex` 之后这个读取随之删除）。
- **嵌套的按钮**（裁定 R3）：`dropdowns/` 的下拉框把 `DropdownButton` 放在 Headless UI 的按钮里，HTML 不许按钮里有按钮；面包屑之前一节的标题按钮也在选择的按钮里（`tabIndex={-1}`，键盘从侧边栏到那一节的页）。键盘经里面的按钮能用；M4 随工作项的属性下拉框和页头一起改。
- **成员下拉框中 Escape 之后焦点在选择自己的按钮上**（不在 Tab 顺序里，键盘取得焦点时有轮廓），不回到里面的 `DropdownButton`（原来落到 `body`）：M4 改嵌套的按钮时一起定。
- **P10 之外的正的 `tabIndex`**（spec 第 3 节第 12 条，第 5 节 M4、M6、M7 行和收尾）：P10 的页面上已没有（F-15、F-18）。工作项、收集箱、迭代、模块、视图的表单，`baseTabIndex` 的三个读者，`CustomMenu` 的根，和 P10 之外写死的 6 处仍有；Tab 在最后一个正的之后离开它们的弹窗。表单的照 F-15 的机制（P10 的创建表单在浏览器中显出）；`baseTabIndex` 的两个弹窗、`AlertModalCore` 的确认按钮和快捷键弹窗是照这个机制推出的，没有在浏览器中跑。各由它的阶段删去。
- **`CustomSearchSelect` 开着时卸载也调 `onClose`**（`WhileOpen` 的清理）：现在的调用方的回调是 setter 或什么都不做（整分支评审核对）。`WhileOpen` 在列表打开时记下 `onOpen`、`onClose`，列表开着时调用方换掉的回调这一次关上时不用（预检的 L8）；现在的调用方都传 setter 或不变的回调。
- **general 页的 `key` 守着一条今天走不到的路**：P3 的同一路由内的移动（`moveWithinApp`）代替之后加在设置页上的项目切换；React Router 不再照 `popstate` 移动时这一步先失败（spec 第 6 节）。
- **成员页对项目访客显示没有权限**：nerve 让他列出成员，页面比 nerve 窄；里面的门（`roleChoices`、`canRemove`、"Add member"）各有检查，放宽它无害（整分支评审的权限表）。访客的侧边栏菜单给"Settings"，打开只读的 general 页（Plane 的分法，不是与 nerve 的不一致）。
- **创建表单的负责人按钮显示头像和占位的"Lead"，不显示名字**（`showUserDetails` 关着），P10 之前就这样（C6 的观察）。
- **每次挂载都重取、聚焦时不取**（P8a 的 F-1）。
- **复现只到修订**：逐 Task 复现是修订时的（spec A.10）；修正轮、修复轮和浏览器核对之后的几轮改了复现没有见过的代码，没有再复现。
- **`@nerve/ui` 下拉框的 `any`**：`dropdowns/helper.tsx` 7 个、`custom-search-select.tsx` 2 个，Plane 原有，P10 重写这个组件时没有改它们的类型（给它们类型是这些组件的泛型，改到 13 个调用方；收尾）。
- 浏览器核对看到的其余几处不是 P10 的缺陷，已列在第 6 节的收尾，这里不重复。
