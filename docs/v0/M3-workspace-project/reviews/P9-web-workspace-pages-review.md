# M3/P9 工作区的页面：评审记录

| 项 | 内容 |
|---|---|
| Phase | M3/P9 `web-workspace-pages` |
| 日期 | 2026-10-09 |
| 结论 | **通过**。<br>11 个 Task 逐个实现、逐个评审，都通过。其中 10 个（Task 1、3–11）各有一次当轮的修正轮。Task 4、5、9 的评审各有一条 Important：Task 4、9 的是执行中发现的缺陷，Task 5 的是只有评审守着的会话性质；其余是 Minor。Task 2 的两条 Minor 搁置到修复轮。<br>整分支评审的结论是"修复轮之后可以合并"：Critical 0、Important 1、Minor 6。<br>修复轮一次分派、八个提交（`e1bdc799` 到 `c1e60774`），落实裁定 F-1–F-8 交给它的发现和搁置项 P1–P12。修复的复审 Critical 0、Important 0、Minor 4，同一个实现者用一个提交补上（`a8eed6b0`，裁定 F-11），它报告的同类问题再用一个提交补上（`897235e5`，裁定 F-12）；两个提交的范围复审见第 4 节末。<br>控制者的浏览器核对 C1–C5 都通过（第 2 节）。 |
| spec / plan | [spec](../specs/P9-web-workspace-pages.md) / [plan](../plans/P9-web-workspace-pages.md) |
| 设计 | [M3 设计](../M3-design.md) 第 12 节 P9 |
| 分支 | `worktree-m3-p9`，从 `main` 的 `4b1334a5`（P8b 合并）分出 |

## 1. 评审方式

- **spec 和 plan 由架构子任务产出**（`5684676f`）：
  - 设计 12 节 P9 的草稿是 14 个任务，plan 是 11 个 Task、8,520 行，附带原型和逐 Task 复现（第五次复现与原型逐文件相同，3,166 个文件）。
  - 76 个变异在最终原型上都被发现，每个也在它自己的 Task 的树上跑过，层相同。W18 另写 40 个探查变异：P9 新加或改写的 118 个测试每个至少失败一次。
  - spec 第 3 节标"裁定"的八条由控制者裁定 P1–P8（`$M3TMP/p9-architect-rulings.md`），都接受（第 3 节）。
- **执行前的预检**（opus，只读，`$M3TMP/p9-preflight/preflight.md`）：高 0、中 4、低 10。复现干净：每个 Task 之后门禁通过，最终与原型逐文件相同，只差文档提交的三个文件。裁定 P4 的前提（另一个账户登录时包装层卸载页面）在代码中和探查中成立（3 次）；P1、P3、P5、P6 成立。
  - M1：2.13 中三行没有"那一处不经 `followInSession` 时失败"的测试（邀请列表、邀请页的接受和忽略、新手引导根的 `change`），探查变异 PF2、PF2b、PF3c、PF19 在每一层都存活。这是裁定 P2 自己的条件。
  - M2：资料一步在 `await` 之后把引导交给根，根的完成在新会话里发生，"出错了"显示在另一个账户的页面上（探查 4 次都复现）。裁定 P8 没有看到这一处。
  - M3：项目导航对话框的限制开关发出点击时显示的开或关，快按两下停在关闭（P8b 评审 P14 那一类，总体设计 7.7 的规则）。
  - M4：P9 的取数文件不在 `.oxlintrc.json` 的 `swr` 范围里，新模块不在非空断言的范围里。
  - L1–L10：两个 vitest 的标题说的比它查的多；角色的三条说明成了两种语言都留下的孤儿；S2 的两个新测试缺最后的复查；2.13 的"不跟进的"漏了只因卸载才安全的组件状态；设计还有五处写"组件先核对 `inSession()`"；`T4.6` 写错；general 页复制地址失败什么都不说；第 5 节收尾一行只写了"Success!"；`heldChange` 是 `gate()` 的第二份；"`useWorkspacesFetch` 是唯一的取数"只由评审看住。另外 PF16：两种语言都留下的无读者的键没有常设的检查。
- **修订**（`1f777867`）：落实 P1–P8 要的文档和预检的每一条（控制者的裁定见 `$M3TMP/p9-amend-brief.md`，落点见 spec 第 3 节"预检之后"）。
  - 邀请列表的修改、邀请页的回答移进照 `useMembershipChanges` 形状的两个 hook（`useInvitationChanges`、`useInvitationAnswer`），各有换账户之后兑现的 vitest。
  - 修订让 Task 6 长到 1,687 行，于是邀请表单的 hook 交出邀请的两处改动和 `useCopyInvitationLink` 移到 Task 3。plan 9,105 行，最长的 Task 6 是 1,475 行、Task 3 是 1,372 行。
  - 从 `4b1334a5` 的新副本重新复现（第六次）：每个 Task 之后门禁通过，最终与原型相同，只差这个提交的四份文档。97 个变异都在最终的树上、也在各自 Task 的树上被发现。W18：140 个测试，44 个探查变异，没有存活的。
  - e2e 75 → 93，vitest 504 → 640，关键词守卫 65 条规则，web 的 oxlint 上限 360 → 359。
- **执行**：
  - 开始时的裁定（照 plan 执行、E1）见第 3 节。
  - 11 个 Task 各由一个实现子任务完成（不指定模型），再各由一个评审子任务（opus）检查是否符合 spec、质量是否达标。修正轮由同一个实现者做，sonnet 做范围复审；Task 11 只改三行文档的修正轮由控制者逐字核对，代替范围复审。同一时刻只有一个写入者。
  - 实现者在每个 Task 里照 plan 的变异表再跑变异，并加上自己的探查变异。每个 Task 的实现和修正轮都跑过 `make lint-web`、`make knip`、`make test-web` 和本地的 `make e2e`（裁定 E1）；Task 11 只改文档的修正轮只跑前两个。
  - **执行中发现并修好的生产缺陷**（都在分支上修好）：
    - T4：
      - 评审的 I1：邀请页的 `mismatched` 是 `<AuthenticationWrapper>` 之上的状态，会话变了它还在。Dave 被告知邮箱不一致之后，另一个标签页换成 Carol，Carol 被告知她自己的邀请是发给另一个邮箱的。裁定 P4 的前提在这一页不成立；
      - "接受"没有在路上的状态：连按两下发出第二次接受，nerve 拒绝，页面先提示出错再进入工作区（原来的页面把它藏在 `console.error` 里）；
      - 中文下邀请页的标题和邀请表单用英文的角色名（`ROLE`）；Task 6 把邀请表单带进新手引导时，那里原来译出的角色名也会变成英文；
      - 邀请页的"接受""忽略""重试"是 `<li role="button">`，没有键盘的处理，键盘用不了。
    - T5：
      - 表单在工作区打开之前就不忙了（`onCreated` 不等写上次打开的工作区），再点一次检查的是新工作区的 slug，闪出"已被占用"；
      - 另有一处代码本来对、只有存活的变异看得出：被拒绝的 slug 检查可能被当作"已被占用"（见下一条列表）。
    - T6：
      - 只给自己建工作区（"Just myself"）的人，创建一步在结束引导的请求还在路上时就不忙了（偏离 T5-c(b)，第 5 节）；
      - nerve 拒绝这个人的引导结束时，表单带着新的 slug 回来，再点说"已被占用"，走不下去（Plane 和 plan 原有）。
    - T7：注册关闭的 nerve 上，注册页不给"登录"的链接。P9 经"注册以接受"让这一页能到达，于是成了死路。
    - T9：停用弹窗在请求在路上时能关掉：之后到达的拒绝在再打开时仍显示；再打开时"确认"可点，第二个 `/me/deactivate` 能发出（这一半的根是 `handleClose` 重置忙碌的状态，M2 时就有）。
    - 整分支评审：
      - I1：邀请弹窗、删除工作区的弹窗、离开或移出的确认框在请求在路上时都能关掉。探查中：迟到的拒绝落在再打开的空表单上；第二个 `DELETE` 发出，先提示"已删除"再提示出错；发出两个删除成员关系的 `DELETE`；
      - M1：`/create-workspace` 不再等导航完成，是对 BASE 的回退；新手引导的结束（资料一步、"以后再说"、链接之后的"继续"）在回答之前又可以点，第二次点击再发一次；
      - M2：`/create-workspace` 的草稿在包装层之上，换账户之后留着：B 看到 A 写的草稿；
      - M3：邀请页的"退出登录"失败时没有处理，是未处理的拒绝，什么都不说。
  - **代码本来对、只有评审或变异守着的性质**（当轮补上测试）：
    - T3：W4 不核对每个拒绝在哪一行下，也不核对已忽略的邀请的菜单没有"复制链接"，两个变异在 vitest 和 W4 上都存活；
    - T4：邀请页"无法连接"时的"重试"没有检查；`invitation-view.test.ts` 出错的三行没有 SWR 留在错误旁边的旧数据，`error && !data` 的变异存活；
    - T5：slug 的检查在换账户之后才回答或被拒绝时，页面什么都不做（spec 2.13 的条件），没有 vitest 守着：那一处若不经 `followInSession`，被拒绝的检查的提示显示在另一个账户的页面上，每一层都发现不了（评审的 I1）。被拒绝的检查（429、无法连接）被当作"已被占用"的变异存活；
    - T6：新手引导的根取不到列表时的界面没有测试；
    - T7：从登录页去注册页丢掉查询参数的探查变异存活（没有故事从登录页去注册页）；
    - T8：变异 `T8.4` 照表的写法等价于正确的代码，只因 hook 测试的替身停在旧值才被发现；
    - T9：新的确认清掉旧的拒绝，没有检查；
    - T10：包装层的测试不读"找不到"的说明，没有工作区的人也看到"Visit Profile"的变异存活。
  - **执行中删除的孤儿**：T3 中 `invite-modal.tsx` 一个不起作用的 `clearTimeout`；T4 中 `EmptySpace` 的 `link` prop（邀请页是它唯一的使用者）；T6 中两种语言的顶层文案键 `"email"`（它的读者是旧的邀请一步的页头；`i18norphans.py` 因为别处出现同样的字面而看不见，PF16 的盲点），和 `ProfileStore` 从不读的 `store` 字段（连同两处 `{} as RootStore`）。
  - 其余裁定都是 plan 自己的测试缺口、测试的写法或说错的注释（第 3 节）。跨 Task 的去重和收尾搁置到修复轮（P1–P12，`fix-wave-parked.md`）。
- **整分支评审**（opus，`1f777867..dba0d756`，115 个文件）：结论"修复轮之后可以合并"，Critical 0、Important 1、Minor 6；另外指出 5 条搁置项说得太窄、漏了地方或数目不对，并照问裁定 P3、P7、P9、P12（第 4 节）。
- **修复**：一次分派，八个提交（`e1bdc799` 到 `c1e60774`）；变异的记录（F-7）在 `$M3TMP/p9tools`，不是提交。修复的复审（opus）通过：Critical 0、Important 0、Minor 4。同一个实现者用一个提交（`a8eed6b0`）补上（裁定 F-11），它报告的同类问题由 `897235e5` 补上（裁定 F-12）；两个提交的范围复审（sonnet）通过（第 4 节末）。
- **控制者的浏览器核对** C1–C5（M3 设计 9.7）在修复轮的头 `c1e60774` 的构建上做，五项都通过（第 2 节）。之后的两个提交只改测试、文档和注释。
- **持续集成**：见第 2 节末。

## 2. 验收标准核对（spec 第 4 节，M3 设计 12 节 P9 的完成线）

| # | 标准 | 结果 |
|---|---|---|
| 1 | W1–W9 的页面版本、S1、S2 和此前的全部故事通过：worktree 中 `make e2e` 93 个（`4b1334a5` 是 75 个）；`make test-web` 通过 | **通过。** 本地（裁定 E1）每个 Task 的实现和修正轮（Task 11 只改文档的修正轮除外）都是 plan 的数目：Task 1 77、Task 2 79、Task 3 80、Task 4 82、Task 5 83、Task 6 88、Task 7 90、Task 8 91、Task 9 92、Task 10 和 11 93。修复轮在第 4 步之后和 `c1e60774` 上 93 个，补充 `a8eed6b0` 上 93 个；F-12 只改 e2e 的一个 fixture，跑了用它的 W1、A12、W3、W4、W7（16 个）。web 的 vitest：`4b1334a5` 上 56 个文件、504 个；Task 11 结束时 73 个、646 个；修复轮之后 75 个、657 个（补充没有加 vitest）。整分支评审在副本上 92/93（只有 S3 因为副本不是 git 仓库失败，F4）；修复的复审列出 93 个，跑了修复轮的五个页面故事。持续集成见本节末。 |
| 2 | 9.5 中 P9 的 vitest 通过，每个都有发现它的变异；P9 写或改的每个测试至少被一个变异发现（W18） | **通过。** `delete-workspace-modal.test.tsx` 6 个（`T1.3`、`T1.7`、`T1.8`）、`onboarding/root.test.tsx` 13 个（`T6.1`、`T6.4`、`T6.6`、`T6.8`、`T6.10` 等）、`onboarding-place.test.ts` 8 个（`T6.2`、`T6.3`）、`invitation-link.test.ts` 7 个（`T3.7`、`T4.8`）。W18：修订时 P9 新加或改写的 140 个测试都至少失败一次（spec A.2）。修复轮新加的测试由复审对照修复轮的变异逐个核对：两个没有变异让它失败（复审 M2），补充加 `M3.c`、`P12.f`，各只由那一行发现。修复轮的测试没有另跑一次 W18 的全表。 |
| 3 | 9.6 的变异核对：去掉 `followInSession` 的核对（`T1.1`），W3 的会话切换失败 | **通过。** `T1.1` 去掉 `in-session.ts` 的 `if (!inSession()) return;`。整分支评审在 `dba0d756` 的副本上重跑：W3 的会话切换在 `w3-workspace-settings.spec.ts:389`（`expect(await moved).toBe(false)`，页面跳转了）失败。修复轮的变异记录在修复轮的树上再跑：同一个断言失败（那棵树和 HEAD 上是 :390），另由三个 vitest 文件发现（`in-session.test.ts`、`delete-workspace-modal.test.tsx`、`workspace-details.test.tsx`）。设计 9.6 要 P9 写进评审的就是这一条。C5 是同一性质在浏览器中看得见的一半。 |
| 4 | W4 的成员视角：成员打开成员页，没有失败的请求，也不请求邀请 | **通过。** W4 的页面版本（成员的页面不显示也不请求邀请，`watchPage` 没有失败的请求）；S2 中成员、访客、不是项目成员的成员打开成员页都只有 `APP`、`WORKSPACE`，`T2.8`（成员页对每个角色都取邀请）由 S2 发现。整分支评审逐页核对挂载时的请求，与 S2 的 `REQUESTS` 和 spec 2.12 相同。C3 在浏览器中核对成员 bob 的页面没有发出 `GET …/invitations`。 |
| 5 | `node tools/keywords.mjs` 通过：67 条规则、3 个例外，没有命中；`tsc`、knip 通过 | **通过**（`a8eed6b0` 的 `make lint-web`、`make knip`，`897235e5` 的 `make lint-web`）。P9 加 `workspaces-list-fetch`（Task 10，`T10.6` 由它发现）；修复轮加 `refusal-toast`、`sign-out-toast`（`P12.c`、`P12.d` 由它们发现）。 |
| 6 | 根目录 `.oxlintrc.json` 的范围（2.14），每一组各有一个变异让 `check:lint` 失败 | **通过。** `no-restricted-imports` 加三个取数文件（W4 一类的 `T6.12`、`T10.4`、`T10.5`）；`no-non-null-assertion` 加 P9 的新模块和测试（W12 一类九个），Task 4 的修正轮加 `empty-space.tsx`，修复轮加两个新 hook 和它们的测试（`733af2b4`，`P12.e`）。这些变异都让 `check:lint` 失败。 |
| 7 | 改到的文件按 7.9 没有 oxlint 警告；上限调低；没有新的抑制 | **通过。** 修复的复审在副本上核对：P9 改到的 115 个 web 的 TS 文件都是 0 条，web 356 条等于上限；补充、F-12 改到的文件也是 0 条。上限 360 → 359（Task 3）→ 357（Task 4 的修正轮，`empty-space.tsx` 清零）→ 356（修复轮，确认框去掉正的 `tabIndex`），其余各包不变。没有新的抑制，删去三处，188 → 185（spec A.5）：Task 6、Task 8 各一处；修复轮的 F-4 在 `account-commands.ts` 随 `useCallback` 删去一处 `react-hooks/exhaustive-deps`（评审时补进 spec A.5，原来写两处、186）。 |
| 8 | 没有新的 `as`、`any`、`!`，第 3 节第 15 条裁定的三个 `as const` 除外 | **通过。** 整分支评审在 `dba0d756` 上核对，Task 6 的修正轮另删去两个 `{} as RootStore`。修复轮在 e2e 写了三个新的 `let …!:`（复审 M1）；补充把它们和 e2e 中此前就有的三处都改为 `deferred()`，e2e 的 fixture 和故事中不再有 `let …!:`（范围复审核实）。修复轮没有新的 `as`、`any`（复审）。 |
| 9 | 逐 Task 复现 | **通过（修订时）。** 修订 `1f777867` 时从 `4b1334a5` 的新副本照 plan 重放：每个 Task 之后门禁通过（e2e 76…92 个，加 S3 的 F4），最终与原型逐文件相同，只差四份文档（spec A.10）。执行中各 Task 的修正轮和修复轮照裁定偏离原型，没有再复现（整分支评审的说明）；plan 中受影响的块见第 5 节。 |
| 10 | 控制者的浏览器核对 C1–C5 写进评审（9.7） | **通过**，见下。 |

**控制者的浏览器核对 C1–C5**（9.7，`$M3TMP/p9-browser-checks.md`，截图在 `$M3TMP/p9browser/shots`）：
- **准备**：
  - 构建：`c1e60774` 的 `git archive` 副本，`make build`（`bin/nerve`，内嵌 web）。之后的 `a8eed6b0`、`897235e5` 只改测试、文档和注释，生产代码相同。
  - 数据库：一个临时的 `postgres:18.6` 容器（`p9-browser-db`），核对时起、之后停掉，没有碰别的容器。同一个库上两个 nerve：A 注册开放（`127.0.0.1:8091`），B 注册关闭（`NERVE_AUTH__SIGNUP_ENABLED=false`，`127.0.0.1:8092`）。
  - 数据经接口建（`p9browser/` 下的脚本）：acme（alice 管理员；bob 成员、carol 访客、frank 经页面加入、ivy 经 nerve B 加入）、gamma（alice 管理员，ben 成员）、beta（alice 管理员）；acme 待处理的邀请 erin（已忽略）、frank、grace（访客）；新人 hana，语言是中文。
  - Playwright MCP，Chromium 1200×900，宽窄的核对用 700×900。
- **C1 落点（W2）：通过。** 没有上次的工作区时，alice 落到最早建的 `/acme`。经工作区菜单切到 Beta，刷新 `/` 落到 `/beta`。在 general 页删除 Beta 之后到 `/acme`。在 Gamma 中角色选择勾着 ben 现在的角色，把他提升为管理员，alice 再离开，落到 `/acme`。除核对自己故意的拒绝（错误密码的 401、"已是成员"的 422）外，没有失败的请求。
- **C2 邀请链接（W5、W6）：通过。**
  - 未登录：显示工作区和角色、"登录以接受""注册以接受"，两个链接都带着邀请和 `next_path`，不显示邮箱。登录之后回到链接，接受发出 `POST …/accept`，再写上次打开的工作区，进入 `/acme`。
  - frank 打开 grace 的链接点接受：说发给了另一个邮箱，只给退出登录，不显示任何邮箱。已忽略的链接说已忽略；令牌改一位或没有令牌说无效。
  - 注册关闭的 nerve B：邀请要在 B 上建，因为每个 nerve 用自己的临时密钥签令牌（A 的链接在 B 上无效，与 README 8.7 的单一密钥一致）。ivy 从链接到注册页，看到"Join A Acme"和去登录页的链接（T7-b）；注册之后回到链接、接受，新手引导只有资料一步，落到 `/acme`。
  - 中文（hana）：角色名、邮箱不一致、已忽略、无效的文案都译出（T4-c）。
- **C3 成员页的三种身份：通过。**
  - 管理员 alice：名字、显示名、邮箱；别人有角色选择，自己一行是文字；"添加成员"；待处理的邀请中 erin 标"已忽略"，菜单只能删除、没有复制链接，grace 待接受，有角色选择和菜单。邀请 bob 在那一行下说"已是成员"；邀请 jack 为访客，建成（201）。
  - 成员 bob：成员列表和邮箱，角色是文字，没有"添加成员"、没有邀请、没有 Webhooks；页面没有发出 `GET …/invitations`；自己一行的菜单是离开。
  - 访客 carol：成员页显示没有权限（9.2）。
- **C4 M2 交接第 13 节：通过。**
  - "找不到工作区"界面的退出按钮：Tab 能到达（`aria-label="Sign out"`），空格退出登录到 `/?next_path=%2Fnowhere`，再登录之后 Tab、回车也退出。
  - 加载：工作区的页面在包装层的取数回答之前显示加载；项目的设置页（状态）在项目到达之后才显示。
  - `ProfileSidebar`：700 像素时折叠，拉过 768 像素展开；第二次同样，监听读的是当前的状态。
  - `WorkspaceLogo`：工作区菜单（菜单项和触发按钮）和设置的工作区卡片中是方形的盒子。
  - 中文下的新手引导（三步的标题、邀请一步角色的默认值"成员"、链接一步的文案）和首页导览的文案都译出。
  - "加入工作区"一步不在：新手引导从资料一步到创建（决策点 2），P8a 删除了它。
- **C5 两个账户两个标签页（7.1）：通过。** 标签页 1 以 carol 停在 `/acme/settings/members`（没有权限的界面）；标签页 2 中 carol 退出、alice 登录。标签页 1 不刷新就跟到 alice，渲染她的成员页（管理员、添加成员、待处理的邀请）；它刷新自己的令牌，取新会话的数据（包括管理员的邀请列表），没有失败的请求。自动的一半是 W3 的会话切换（上表第 3 行）。
- 核对中看到的、不是 P9 缺陷的几处进第 6 节（P11、M4、收尾）。

**变异的记录**（裁定 F-7，`$M3TMP/p9tools/p9-mutation-record.md`，结果 `mut-results-fw.json`，日志 `mut-logs-fw/`）：
- plan 的 97 个、各 Task 修正轮和探查的 41 个、修复轮的 27 个，共 165 个，在修复轮的树上各跑一次：163 个被发现。
- 补充加 3 个（`M3.c`、`P12.f`、`I1.f`），`closedByEscape` 改过之后重跑 `I1.d`、`I1.e`、`T9.7`；F-12 改过 `enabledWithin` 之后重跑由它发现的 10 个（`T5f.3`、`T6f.1`、`T6f.2`、`M1.a`–`M1.f`、`P10.b`）。都被发现。全表 168 个中 166 个被发现，没有不能跑的。
- 没有被发现的两个：
  - `T8.4-old`（plan 原来的 `T8.4`）是等价的：它在轮到修改时才读显示的设置，那时与 store 所持的相同。Task 8 的修正轮把 `T8.4` 改写为探查 `T8p.1` 的写法（提出修改时就读），由 hook 的 vitest 和 W8 发现。
  - `T4f.M3a`（邀请页的标题按英文的 `ROLE` 写角色）存活：标题没有渲染的测试，`fake-i18n` 丢掉参数，W5 读英文。Task 4 的修正轮已记下（T4-f），是第 6 节收尾中角色名一项。
- 有一层设计上就看不见的（记录逐个写明，变异由另一层发现）：`T3.13`、`T9p.2`、`T9p.5` 的 vitest 一层（服务端渲染看不到点击之后才设的状态，由 W4、W9 发现）；`T9p.1` 的 vitest 一层（弹窗测试的替身不发第二次请求，由 W9 发现）；`T7p.1`、`T7f.1`、`T7f.2`、`T8p.2`、`T8p.3` 的 `tsc`、oxlint 一层（这些探查写出静态检查，说明它们看不见这个性质）。
- 没有收进来的：`t3-fix-mut` 的 `F3.p1`、`F3.p2` 是 W4 的 fixture 的探查，不是代码的变异（复审认可）。
- 记录引用的行号多数是第 6 步修订之前的树（`46110e0a`）上的，记录的开头写明到补充的树的偏移；补充和 F-12 重跑的引用补充的树上的行。

**持续集成**：分支上每次推送都跑 server、web、e2e 三个任务。下面的运行除一次被取消外都通过；其中五次在 GitHub 接口的速率限制之下，从运行的页面上读结果（标"页面"）：
- Task 1（`4424fcd4`）：run 37892123219；
- Task 1 的修正轮（`7c9d08f7`）：run 37893656125；
- Task 2（`228c5ec6`）：run 37894424150；
- Task 3（`4016450b`）：run 37895805573；
- Task 3 的修正轮（`d30d7170`）：run 37897378634；
- Task 4（`b1779c3b`）：run 37898355845（页面）；
- Task 4 的修正轮（`835a6f51`）：run 37902476685；
- Task 5（`c494a9a1`）：run 37903764497（页面）；
- Task 5 的修正轮（`2fbbd863`）：run 37905781894；
- Task 6（`3d0bcee1`）：run 37908088832；
- Task 6 的修正轮（`1384f634`）：run 37910579079（页面）；
- Task 7（`3babf3f0`）：run 37911625595；
- Task 7 的修正轮（`9db5caac`）：run 37913095067；
- Task 8（`559898b0`）：run 37914372370；
- Task 8 的修正轮（`f581f50c`）：run 37915759806（页面）；
- Task 9（`90421304`）：run 37916859771；
- Task 9 的修正轮（`f250ec2c`）：run 37919002933；
- Task 10（`e1e967a5`）：run 37919983407；
- Task 10 的修正轮（`d8838659`）：run 37921284089（页面）；
- Task 11（`0d5f8ec6`）：run 37922374180 在 e2e 中被 `dba0d756` 的推送取消（并发组取消在途的运行；web、server 已通过）；
- Task 11 的修正轮（`dba0d756`）：run 37923040018；
- 修复轮的前七个提交（到 `733af2b4`，控制者为早一点跑持续集成先推送）：run 37935688203；
- 修复轮的头（`c1e60774`）：run 37939228977；
- 补充和 F-12（`a8eed6b0`、`897235e5` 一起推送）：run 37945957035。

本地的 `make e2e` 每个 Task 都跑（裁定 E1：Docker Desktop 2026-10-09 起又能用），持续集成是第二道。本提交推送之后，分支上再跑一次；合并之后 main 的持续集成在合并时记录。

## 3. 执行中的决定

这一节列控制者在本阶段做的裁定，每条写明理由和裁定错了的代价。完整的记录在执行台账中。

**执行前**
- 架构子任务的 spec 第 3 节，控制者的裁定 P1–P8（都接受，`$M3TMP/p9-architect-rulings.md`）：
  - **P1**（14 个任务 → 11 个 Task）：设计的每个任务都有落点，页面版本与它的页面同一个提交，最长的 Task 6 在上限之内。代价：太大的 Task 由它的修正轮拆。
  - **P2**（会话核对只写在 `followInSession` 一处）：十几个组件各写 `sessionGuard()`、`await`、核对、跟进，是负责人的标准不许的重复；9.6 的变异成为 `T1.1`。条件：每个组件不经它自己跟进时，由它自己的 vitest 发现。设计 7.1、9.6 的句子随之改（修订）。代价：绕过它的组件只由它自己的 vitest 守着，预检核对 2.13 每一行都有。
  - **P3**（M2 的 `UserStore.deactivateAccount` 交回 `endSession` 的 `boolean`）：唯一的调用方是停用弹窗；成功只在结束了这个标签页的会话时跟进（W3 一类）。代价：没有（M2 收尾交接的 P9 处理结果写明新的签名，Task 11）。
  - **P4**（停用被拒绝的原因是弹窗自己的状态，不另加核对）：暂时接受，由预检核对"另一个账户登录时包装层卸载页面"，预检核实（代码和 3 次探查）。代价：另一个账户的页面上一条旧的拒绝。执行中 T4-e 把这一条说得更确切。
  - **P5**（W1 的"含大写"改为 `café`，字段把大写转成小写）：字段从不发出大写；故事的本意由 `café` 保留。代价：没有。
  - **P6**（`members-list.tsx` 89 → 91）：一个地址要的 prop 和格式化的拆行。代价：没有。
  - **P7**（`fake-tab.ts` 的 `let settle!` 和三个 `as const`）：`as const` 不是类型转换；修订让 `heldChange` 用 `gate()` 之后，那个 `!` 也没有了。代价：没有。
  - **P8**（M2 的账户页面自己的提示不核对会话，交给 P11）：它们改的是账户，不是 9.6 列的工作区一侧。代价：P11 之前，迟到的账户页面提示可能显示在另一个账户的页面上（第 7 节）。
- 预检（高 0、中 4、低 10）和修订 `1f777867`（第 1 节）：控制者的裁定都接受（`$M3TMP/p9-amend-brief.md`）：A-M1 两个 hook 和它们换账户之后兑现的测试；A-M2 资料一步的交接经 `followInSession`；A-M3 开关在轮到它时决定（`{ limitToggled: true }`）；A-M4 两条 `overrides` 的范围；L1–L10 照预检给的修法；PF16 仍是 P8b 的 PF-L5 留下的缺口（第 6 节收尾）。spec 第 3 节由这些裁定定下，执行中没有再议，只有第 4 条由 T4-e 说得更确切。
- 开始执行时：
  - **照 plan 执行**：预检的跨 Task 表已逐对核对，它的发现都在修订中落实，并由第六次复现再证明。代价：错了是一轮 Task 级的修正。
  - **E1**：Docker Desktop 又能用了（负责人 2026-10-09 重启了它）；每个 Task 在本地跑 `make e2e`，数目照 plan 的全局约束；Docker 不再回答时实现者报告，控制者像 P8b 那样依靠分支的持续集成。代价：每轮慢一些。

**执行中**
- **T1-a … T1-d（会话里的跟进；general 页和删除；W3）**：
  - 实现者报告 general 页复制失败的提示（预检 L7）没有检查：`.catch(() => undefined)` 照样通过。搁置为 P1：它和邀请链接的复制用一个检查（T1-a）。代价：当时没有。
  - 评审 C0/I0/M3，当轮修（T1-b）：
    - m1：W3 中确认的点击和等回答写在一起，10 秒的期限从点"确认"算起，盖住数据库的轮询、另一个标签页的登录：一个会说错原因的偶发失败。改为只点击的 `confirmDeletion`，会话切换从放行起等回答；
    - m2：一句不真的注释；
    - m3：general 页的 `currentWorkspace` 指的是地址的工作区，改名 `workspace`。
    - 理由：m1 在之后的故事复用的 fixture 里。代价：一个小提交。plan 中 Task 2 的 `e2e/fixtures/workspace-pages.ts` 整文件块随之带上 `confirmDeletion`（`7c9d08f7`）。
  - 给 Task 9 的说明：Task 9 的 `store/user/index.test.ts` 自己写一个带 `endSession` 的 `tab`，`fake-tab.ts` 要不要加它（T1-c，由 T9-pre 决定）。代价：没有。
  - 范围复审：Task 1 自己的 plan 块仍是修正之前的代码。记作 plan 文字的漂移，不改（T1-d，P8b 的 F-4 先例）；之后的块照样能应用（实现者在副本上核对过）。搁置为 P2。代价：没有。
- **T2-a（成员页）**：评审 C0/I0/M2。m1：`useMemberColumns.tsx:23` 读 `useParams()`，而成员行从列表收地址的 slug，一行两次读地址（没有变异），搁置为 P3 给整分支评审。m2：spec 第 3 节第 19 条漏了角色选择现在勾着那一行的角色，搁置为 P4 给修复轮的文档。代价：没有。
- **T3-a（邀请的表单和列表）**：评审 C0/I0/M4，当轮修：
  - m1：W4 的 fixture 按 `name:"Member"` 的第几个找行的角色选择，`[Guest, Guest]` 这样的一批会超时（潜在的，Task 6 可能复用）；
  - m2：W4 在整个弹窗里找拒绝的原因，不在各自的行下，行互换的变异在 vitest 和 W4 上都存活；
  - m3：已忽略的邀请只核对了角色，去掉 `shouldRender: !declined` 存活；
  - m4：两个小处。
  - 理由：m2、m3 是看得见的行为的存活变异，m1 是之后会复用的 fixture。代价：一个小提交，W4 重跑。修正轮的探查变异留在实现者的草稿里，搁置为 P5。
- **T4-a … T4-g（邀请页）**：
  - 评审：Spec ✅，Quality 要修，C0/I1/M5。
  - **I1**（第 1 节）根上改（T4-a）：页面有状态的主体（预览、回答、`mismatched`、在路上的标志）移进 `<AuthenticationWrapper>` 之下渲染的子组件，裁定 P4 的前提对它成立。核对：W5 的故事 2 在 Dave 的不一致之后接着让 Carol 在另一个标签页登录，标签页 A 显示"接受"（在 `b1779c3b` 上失败，修正之后通过）。若包装层在 PUBLIC 页面换会话时不卸载子组件，实现者停下报告（那会是跨页面的升级问题）。代价：一轮修正。
  - M1、M2、M4、M5 同一轮修（T4-b），每条都是这一页的存活变异或看得见的缺陷：
    - M1：出错的三行带上旧数据；
    - M2：W5 让第一次取预览中断一次，之后"重试"显示"接受"；
    - M4：回答在路上时显示加载、丢掉第二次点击，除接受成功之外每次兑现都清掉；核对连按两下"接受"只发一次、没有提示；
    - M5：删掉 `link`，选项是真正的 `<button type="button">` 或链接，文件 0 条警告，上限降到新的总数（357）。
    - e2e 仍是 82 个（加在 W5 已有的故事里）。代价：同一轮。
  - **M3**（T4-c）：邀请页的标题和 `invite-modal/fields.tsx`（Task 6 带进新手引导的那一行）按 `t(ROLE_DETAILS[r].i18n_title)` 写角色，Task 6 不会让中文退回英文；英文不变，e2e 不变。其余读 `ROLE` 的地方（成员列表、邀请列表、项目成员的页面、提及）搁置为 P6：全应用一个角色名的来源，进 spec 第 5 节的收尾。代价：这一轮之外没有。
  - Task 4 自己的 plan 文字记作漂移（T4-d）；之后的块在副本上核对过，只有 Task 7 的 W6 一行要改（`835a6f51`：按 role 的链接点"注册以接受"）。代价：没有。
  - **T4-e**：实现者报告，PUBLIC 页面上退出登录时包装层仍挂着子组件（`authentication-wrapper.tsx:71`、`:77` 渲染同一个片段），而任何账户的登录都经过加载、重新挂载它们。邀请页是唯一的 PUBLIC 页面，它所持的只在登录之后的视图里读（不一致在登录时决定；加载只在回答的视图里；每次失败的回答都重读，标志总在它的会话里清掉）：接受。spec 第 3 节第 4 条要说得更确切，搁置为 P7；整分支评审权衡包装层是否按会话给 PUBLIC 页面的子组件加 key。不是升级：一个页面，已经正确，不跨模块。代价：当时没有。
  - **T4-f**：plan 中 Task 5–11 的文字仍写上限 359（:5383、:6851、:7135、:8161、:8577、:8962、:9083）；之后的 Task 在分派时写明 357，plan 不改（P2 一类）。`M3a`（标题退回 `ROLE`）存活，记给评审（没有页面的渲染测试，P1 一类）。代价：没有。
  - 范围复审通过，另有一个小处：`EmptySpaceItem` 的按钮里放了 `<div>`，搁置为 P8（T4-g）。代价：没有。
- **T5-a … T5-c（创建工作区）**：
  - 评审：Spec ❌（只是测试：2.13 中 `use-create-workspace.ts` 一行要求 slug 检查的跟进经 `followInSession`，它不经时没有 vitest 失败；代码本来对），Quality 要修，C0/I1/M3。
  - **I1**（T5-a）：照评审在共用的 `lateSettlings` 上加两行（检查在换账户之后才回答、才被拒绝：不创建、不放字段错误、不提示）。理由：spec 2.13 的条件；Task 6 不加创建一步的换账户测试，这个文件是两个使用者唯一的守卫。代价：一轮修正。
  - m1、m2、m3 同一轮修（T5-b）：
    - m1：`onCreated` 交回 `Promise` 并被等待，页面传 `openWorkspace`；W1 扣住 `PATCH /me/profile`，核对按钮一直忙；
    - m2：被拒绝的检查（429）提示 `errors.rate_limited`，不当作已被占用；
    - m3：`creationRefusal` 改用 `needsErrorBanner`、`fieldErrorKeys`，不另写一份。
    - 理由：m1 看得见的闪烁，m2 错误路径的存活变异，m3 负责人的标准不许的重复。Task 5 自己的 plan 文字记作漂移；之后的块不用改。代价：同一轮。
  - **T5-c**：说明 (a)：`useOpenWorkspace` 不等 `navigate`，新页面加载时按钮可能又显示"创建"，搁置为 P9 给整分支评审。说明 (b)：Task 6 的新手引导在工作区只给创建者自己时有 m1 同样的窗口，带进 Task 6 的分派和评审：创建一步要忙到它的跟进兑现。代价：当时没有。
- **T6-a、T6-b（新手引导）**：
  - 实现者照 T5-c(b) 偏离 plan（第 5 节）：窗口确实存在。根为单人的工作区等引导的结束，创建一步等它，`onCreated` 交回 `Promise`；W1 的故事 2 加一个"Just myself"的新人 Bob，扣住 `PATCH /me/profile`，在 plan 的代码上失败。评审认可：与 `2fbbd863` 同一个形状，每个跟进都在会话里，忙碌总会结束（`followInSession` 从不拒绝）。
  - 评审 C0/I0/M4，当轮修（T6-a）：
    - m1：根取不到列表时的界面没有测试（`T6p.1` 存活），`root.test.tsx` 加一行；
    - m2：nerve 拒绝单人的结束时回到带着新 slug 的表单。根上改：结束有自己的失败跟进，先提示，再进入那个工作区的邀请一步（重新加载也会去那里）；W1 让 Bob 的结束答一次 503；
    - m3：两种语言的顶层 `"email"` 成了孤儿，删除；
    - m4：`ProfileStore` 从不读的 `store` 字段删除。
    - 理由：m1 是取数性质的缺口（Task 4 的 M2 一类）；m2 是看得见的死路，修法是这一步自己的跟进约 5 行，没有新机制；m3、m4 是负责人的标准要删的孤儿。代价：一轮修正。
  - 评审的两条说明（W1 故事 2、3 中 Bob 的页面不核对失败的请求和控制台；单人结束的等待只经队列的顺序钉住）搁置为 P10（T6-b）。代价：当时没有。
- **T7-a … T7-d（带邀请的注册）**：
  - 评审 C0/I0/M1。m1：从登录页去注册页丢掉查询参数的探查 `T7p.1` 存活；W6 的故事 2 改成往返（T7-a）。往返三次中失败两次：地址先变、页面后换，两页都显示"Join A Acme"，邮箱填进了旧表单（plan 原来的单向故事也潜在着）；改为等只有登录表单才有的"Go to workspace"之后再填。代价：一个小提交。
  - 评审的说明（第 1 节 T7 的死路）同一轮修（T7-b）：页头的 `enableSignUpConfig` 只管去注册页的链接，注册页的"登录"总显示，带着查询参数；W6 的故事 1 在注册关闭的 nerve 上核对。代价：同一轮。
  - 注册关闭的登录页没有"注册"由 A2（页面）:54–61 核对，不再写一份（T7-c）。范围复审另指出 A2 的 `toHaveCount(0)` 可能在实例的回答和渲染之间就通过（原有），搁置进 P10（T7-d）。代价：没有。
- **T8-a、T8-b（导航设置）**：
  - 评审 C0/I0/M2，每个处理都照 PF-M2 在轮到时算。m1：`T8.4` 照表的写法在轮到它时读，等价于正确的代码，只因 hook 测试的替身停在旧值才被发现。m2：W8 自己又写了一份记录请求体的路由。
  - 当轮修 m2 和 m1 代码的一侧（T8-a）：`settings-pages.ts` 的 `bodiesSentTo(page, method, path)`，`sentTo` 建在它上面，W8 用它；hook 测试的替身跟着轮次；变异工具的 `T8.4` 改为 `T8p.1` 的写法，由 hook 的 vitest 和 W8 发现。m1 文档的一侧（plan 的 T8.4 一行、spec A.2 的数目 3 → 4）搁置为 P11。代价：一个小提交。
  - 评审的说明：`setToast({…t(errorMessageKey(error))})` 全仓库约 30 份，搁置为 P12，由整分支评审决定范围（T8-b）。代价：当时没有。
- **T9-pre、T9-a … T9-c（停用弹窗）**：
  - **T9-pre**（T1-c 的决定）：`fake-tab.ts` 不加 `endSession`。它的 `tokenManager` 是会话的核对所读的标签页会话（13 个测试文件用它替 `api-client`）；Task 9 的 store 测试是另一种读者（store 结束它发出时的会话，替身还要 `publicClient`），为一个读者把 `vi.fn` 和 `publicClient` 放进共用的替身会扩大它的用途。plan 照旧，Task 9 的评审核对只复制了 `state` 的形状。代价：没有。
  - 评审：Quality 要修，C0/I1/M2。**I1**（第 1 节）和 M1、M2 同一轮修，用评审的根因修法（T9-a）：请求在路上时弹窗关不掉：取消按钮禁用；`ModalCore` 的 `handleClose` 为空，Escape 和背景不关；`handleClose` 不再重置忙碌的状态，`finally` 照管。它也去掉第二次发出。评审否决的另外三种：打开时清掉（effect，打开之后的迟到兑现照样写进）、关闭时卸载（丢掉过渡、要改 Plane 的 `form.tsx`，第二次发出仍在）、只在打开时跟进（计数器或 ref 的接线）。M1：W9 中扣住的第二次确认在"停用中"时没有提示，放行之后显示 409 的原因（`T9p.1` 由它发现）。M2：store 测试的行写明回答时的 `loginId`。代价：一轮修正。
  - 实现者另加 A12 的最后一步（取消、可用、关闭弹窗），因为 `T9.7` 在 Escape 之后的检查上存活（约 200 毫秒的离场过渡中内容还看得见）：接受，它说的是故事自己的话（停在原处，可以再确认或取消），通过的方向不靠时间（T9-b）。范围复审说失败的方向靠变异的步骤长过过渡（实测 5/5），更直接的核对搁置进 P10（T9-c）。代价：没有。
- **T10-a、T10-b（工作区列表的唯一取数；包装层的界面）**：
  - 评审 C0/I0/M2。m1：包装层测试的"找不到"两行不读说明，"没有工作区"一行只核对没有 `go_home`；两个变异（说明退回 Plane 的英文；没有工作区的人也给"Visit Profile"）在 vitest 和 W3 上存活。m2：S2 三个单次加载的故事各写一遍"恰好的清单"的结尾。当轮修（T10-a）：两行读说明，`not` 是数组；S2 用一个 `expectExactly`。理由：m1 看得见的行为的存活变异，m2 重复的测试逻辑（与 T3-a、T8-a 一致）。代价：一个小提交。
  - 评审另说退出登录失败的一支在五处都没有检查（`workspace-wrapper.tsx`、`user-menu-root.tsx`、`workspace-menu-root.tsx`、`account-commands.ts`、`switch-account-modal.tsx`，`.catch(() => undefined)` 处处存活）：P1 扩大到这五处；P12 的辅助函数会合并这五份相同的提示（T10-b）。代价：当时没有。
- **T11-a（文档）**：
  - 实现者照实改了 plan 的两句（原型中也不真）：M2 收尾交接第 2 节（W2 只核对切换的写和落点）、前端改动清单 3.2（general 页把字段错误说成提示）。
  - 评审：Quality 要修，C0/I0/M4：m1 3.2 把落点列在按代码显示 nerve 错误的页面中；m2 README 的"快速开始"应是"快速入门指南"；m3 README 说没有工作区的人接着邀请成员，"Just myself"没有这一步；m4 Webhook 是 M8 的。另两处可选的也改：M2 收尾交接的引用加 2.4，个人主页在 P10、P11 对接。裁定当轮修（T11-a）。理由：这些文档的用处是在 HEAD 上为真。代价：一个小的文档提交（`dba0d756`），由控制者逐字核对代替范围复审。

**整分支评审之后**
- **F-1**（I1）：修复轮中逐个弹窗用 Task 9 的根因修法：请求在路上时取消禁用，有 `handleClose` 的 `ModalCore` 在路上时不给它；W4、W3、W7（页面）扣住请求核对；变异 `I1.a`–`I1.c`。理由：负责人要逐个清扫的那一类，看得见的缺陷，合并之前要修。代价：修复轮的一步。
- **F-2**（M1 和搁置项 P9）：在根上改。`followInSession` 在修改和它在会话里的跟进都兑现之后才兑现（跟进可以交回 `Promise<void>`）；"从不拒绝"不变，跟进自己的拒绝不悄悄吞掉，说明写明去向，没有调用方看到未处理的拒绝。`useOpenWorkspace` 交回 `navigate` 的 `Promise`；`named()` 交回 `finish()`；资料一步的 `done` 交回 `onDone()`；"以后再说"和链接之后的"继续"在 `onDone()` 在路上时显示忙碌。理由：这是 P9 自己的基本件，不跨模块；总体设计 7.7 的 P9 一条只在写了旧的兑现点时才改（它没有写，不改）。代价：修复轮的一步。
- **F-3**（M2）：删掉提到包装层之上的草稿（props、页面的状态、卸载时的 effect）；W1（页面）在另一个标签页登录 B 之后核对表单是空的；变异 `M2.a`。代价：修复轮的一步。
- **F-4**（搁置项 P12，照评审的裁定，全应用）：
  - `core/hooks/use-refusal-toast.ts` 的 `useRefusalToast(titleKey = "toast.error")` 取代 26 份相同的和 3 份自己标题的提示；在字段和提示之间决定的几处照旧决定，提示的一支调它；停用弹窗里的原因不变。
  - `core/hooks/use-sign-out.ts` 的 `useSignOut(): () => Promise<boolean>` 取代五处退出登录和邀请页的一处（评审 M3）；一个 hook 的 vitest 发现去掉失败一支的变异（P1 的退出登录一半）。
  - 两条关键词规则让两者各只在一处（65 → 67 条，各有样例）。机械的改动：已有的测试不改照样通过。
  - 理由：P10 接着会再加约十份；只改 web，除 M3 外不改行为，不跨模块。代价：修复轮的一步。
- **F-5**：P3 照裁定修（`useMemberColumns(workspaceSlug)`，去掉两个判断）；M4 修：`use-create-workspace.ts` 中 `useCreateWorkspace` 旁边一个 `useCreationForm()`（规则、`onNameChange`、提交），两个表单各自排版；P8 修（`EmptySpaceItem` 的按钮里是 span）。代价：修复轮的一步。
- **F-6**（测试）：P1 复制失败的一半，两处复制用一个检查（让 `writeText` 拒绝，看到提示）；M6(a) `auth.ts` 中一个"另一个标签页以 X 登录"的 fixture，交回标签页 B，五处都用它；M6(b) P8b 的四个手写的 `ApiError` 改用 `refusal()`；P10：W1 故事 2、3 中 Bob 的页面经 `watchPage` 核对，A2 的 count-0 放在证明页面已渲染的正向等待之后，A12 直接发现 `T9.7`（过渡之后弹窗仍看得见）。代价：修复轮的一步。
- **F-7**（搁置项 P5）：各修正轮的探查和变异与修复轮自己的都收进 `$M3TMP/p9tools` 的变异表（之后的改动挪了位置的给出两种版本），在修复轮的头上跑；本文记录结果（第 2 节）。代价：修复轮的一步，跑得久。
- **F-8**（文档）：
  - 评审 M5（spec 第 2 节的总览、上限、2.14 加 `empty-space.tsx`、vitest 的数目和 A.1、A.6 的大小、第 3 节第 12 条的 `bodiesSentTo`、第 19 条的修正轮的不同、第 5 节收尾：L8 漏的地方、P6、PF16 的盲点）；P7 的一句（spec 第 3 节第 4 条、2.13 的"不跟进的"、`AuthenticationWrapper` 的说明）；P11（plan 的 T8.4 一行、A.2 的数目 3 → 4）；`user/index.ts:104` 的注释；M1-P2 交接的"处理结果（M3/P9）"。
  - 其余不真的"将来"句（M3 设计 :1176、:1232、:1276，`plane-diff.md:242`）进 spec 第 5 节的收尾（P8b 的 P17 先例）；§15 的 P9 一行由评审的提交改（本提交）。
  - plan 文字的漂移（P2）记在本文，plan 不改。
  - 代价：修复轮的一步。
- **F-9**：general 页照旧发出它的三个字段（spec 2.1、A.4：表单编辑的字段；表单的保存以用户看到的为准，最后写的算）。记作已知的行为，不是缺陷（第 7 节）。代价：没有。
- **F-10**（修复轮的报告之后）：接受关注 1、2、4、5：W1 403 行约等于 400 行的上限；更宽的规则模式是更真的规则；离开、接受没有看得见的忙碌窗口（由复审核对）；记录的行号的偏移在它的开头写明。关注 3（主题切换的 `setPromiseToast`）由复审判断归 `useRefusalToast` 还是 spec 第 5 节的收尾。代价：当时没有。
- **F-11**（复审之后）：M1–M4 和小处在合并之前由修复轮的实现者用一个提交补上（P8b 的 `682bbff6` 先例），sonnet 做范围复审。代价：一个小提交。
- **F-12**（补充的报告之后）：`enabledWithin` 同样加固：只把超时算作"一直不能用"；定位到两个按钮的探查必须失败。同一类，一个小提交。代价：小。

## 4. 整分支评审的发现与处理

结论"修复轮之后可以合并"：Critical 0、Important 1、Minor 6。评审（opus）的范围是 `1f777867..dba0d756`（115 个文件，+5,220/−1,642），在 `dba0d756` 的副本上（`git archive`，`pnpm install --frozen-lockfile`）跑过每一道门禁：关键词 65 条、3 个例外，没有命中；turbo 的 `check:types`、`check:lint`、`check:format`、`check:sync`（web 357 条，等于上限）；改到的 82 个 web 文件和三个包文件 0 条警告；knip；vitest 73 个文件 646 个；`make build` 加完整的 e2e 92/93（S3，F4）；9.6 的 `T1.1`（第 2 节）。它还核对了：
- 三类缺陷逐个清扫：每个请求体（封闭、类型生成、角色是数字，都有测试钉住）；每条出错的路径（邀请页的退出登录之外，都经 `errorMessageKey` 或 `FIELD_ERROR_MESSAGES`；没有不结束的加载）；从所持的值做请求体的（只有导航对话框，已在轮到时算）；
- 2.13 的每一行在 `dba0d756` 上的代码和它自己的换账户之后兑现的 vitest；
- 两类：(a) 请求在路上时能关掉的弹窗（I1）；(b) 跟进做完之前就不忙的地方（M1）；
- 会话的模型逐页：裁定 P4 的前提只在 `/create-workspace` 的草稿处不成立（M2）；
- 每一页挂载时的请求，与 S2 的 `REQUESTS` 和 spec 2.12 相同；
- 删除：deadsym 在 HEAD 上多 42 行，都是测试中的成员或一个用 `in` 读的判别字段，没有生产代码的死成员（M1 收尾交接要 M3 的评审记下的就是这一份）；删掉的名字没有留下；三种扫描都没有孤儿的文案键；
- 重复：M6，和一处小的（`workspace-details.test.tsx:49`、`:56` 两次写同一个表单类型）。

| 发现 | 处理 |
|---|---|
| **I1** 三个弹窗在请求在路上时能关掉（Task 9 的 I1 那一类）：邀请弹窗、删除工作区的弹窗、离开或移出的确认框（成员页和邀请列表都用）。探查 A：迟到的 422 落在再打开的空表单上；B：第二个 `DELETE`，先"已删除"再出错；C：两个删除成员关系的 `DELETE`，第一个迟到的兑现关掉别的确认框 | F-1，`e1bdc799`；W4、W3、W7（页面）和 A12 扣住请求核对；`I1.a`–`I1.e`，补充加 `I1.f` |
| **M1** 跟进做完之前就不忙的三处：`/create-workspace`（`useOpenWorkspace` 不等 `navigate`，对 BASE 的回退）；已有工作区的人的资料一步（探查 D：第二次点击再发一次 `PATCH /me` 和结束）；邀请一步的"以后再说"和链接之后的"继续"（没有忙碌的状态） | F-2，`3524be57`；`M1.a`–`M1.g` |
| **M2** `/create-workspace` 的草稿在 `AuthenticationWrapper` 之上：另一个标签页登录 B 之后，B 的表单带着 A 写的（探查 E） | F-3，`b772671f`；`M2.a` |
| **M3** 邀请页的"退出登录"没有失败的一支（未处理的拒绝） | F-4 的 `useSignOut`，`f1b62547`；`M3.a`、`M3.b`，补充加 `M3.c` |
| **M4** 两个创建表单各写一遍规则和"名称给出 slug"的接线 | F-5 的 `useCreationForm`，`9ff34271`；`M4.a` |
| **M5** spec 的数目和清单早于各修正轮：文件总览、上限、2.14、vitest 的数目、A.6、第 3 节第 12、19 条；第 5 节收尾漏了 L8 的地方、P6、PF16 的盲点 | F-8，`c1e60774` |
| **M6** 测试的重复：(a) "另一个标签页以 X 登录"的三行写在五处（W3、W5 和 M2 的 A6、A9、A11）；(b) P8b 的测试手写 `ApiError` | F-6 的 `anotherTabSignsIn`、`refusal()`，`b400e015` |
| 搁置项说得不对或太窄：P1 漏了邀请页的退出登录（没有失败的一支，比没有检查更糟）；P4 漏了另五处修正轮的不同；P9 太窄（同一根因也在新手引导的结束，且是对 BASE 的回退）；P12 的数目（26 份相同的、3 份自己标题的；退出登录的提示五份，第六处没有）；P6 准确，还要写进 spec 第 5 节 | F-4、F-6（P1）；F-8（P4、P6）；F-2（P9）；F-4（P12） |
| 照问裁定 P3、P7、P9、P12 | P3：修，`useMemberColumns(workspaceSlug)`，一行只读一次地址，方向是 spec 2.2 选的（列表读路由、往下传）；`invitations-list-item.tsx` 一行读一次，不改。P7：把 spec 的句子说确切，不按会话给 PUBLIC 页面的子组件加 key：唯一的 PUBLIC 页面所持的两个值都只在登录之后的视图里读、都有钉住，加 key 只多一次重挂载和预览的重取，什么也不修好；真正破坏前提的是包装层之上的状态（M2）。P9：在根上修，经 `followInSession` 等跟进（M1）。P12：在本阶段的修复轮中全应用做成两个 hook，关键词规则可选（F-4 加了） |
| 不真的"将来"句七处：M3 设计 :1176、:1232、:1276、§15 :2116；`M1-P2-trim-content.md:44`（没有 P9 的处理结果）；`plane-diff.md:242`；`user/index.ts:104` 的注释 | F-8：注释和 M1-P2 交接在修复轮改；设计的三句和 `plane-diff.md` 进 spec 第 5 节的收尾；§15 由本提交改 |
| 没有判断的：general 页发出没改的字段（时区的保存写回另一位管理员同时改的名称）；`confirm-workspace-member-remove.tsx` 中 Plane 的英文；中文的措辞（只抽查了加的 26 条）；e2e 的偶发（完整地跑了一次）；变异表（只重跑了 `T1.1`）；plan 的漂移；C1–C5 | F-9（第 7 节）；英文进第 6 节收尾；变异表由 F-7 重跑；漂移记在第 5 节；C1–C5 在第 2 节 |

评审没有提到的搁置项：P2 记在第 5 节；P5 由 F-7；P8 由 F-5；P10 由 F-6；P11 由 F-8。

**修复轮**（一次分派，`dba0d756` 之上，八个提交）：
- **请求在路上时弹窗关不掉**（`e1bdc799`，F-1）：
  - 删除工作区的弹窗和确认框的取消按钮在 `isSubmitting`、`isRemoving` 时禁用，`ModalCore` 的 `handleClose` 为空；邀请弹窗的取消禁用（它的 `ModalCore` 本来没有 `handleClose`）。确认框去掉正的 `tabIndex`，web 的上限 357 → 356。
  - `settings-pages.ts` 加 `closedByEscape(page)`：有期限的一秒，长过 200 毫秒的离场过渡。
  - W4、W3、W7、A12 扣住请求（`holdAnswer`）核对取消禁用，W3、W7、A12 另核对 Escape 关不掉；A12 由此直接发现 `T9.7`（P10 的 A12 一项），最后的取消一步留着。变异 `I1.a`–`I1.e`。
- **忙碌到跟进做完**（`3524be57`，F-2）：
  - `in-session.ts`：跟进可以交回 `void` 或 `Promise<void>`；`followInSession` 在修改和跟进交回的 `Promise` 都兑现之后兑现；仍从不拒绝，跟进抛出或拒绝时交给 `reportError`（控制台和窗口的 `error` 事件），说明写明。M2 的调用者没有一个坏掉。
  - `useOpenWorkspace` 交回 `navigate` 的 `Promise`；新手引导的根中 `named()`、`onDone` 交回 `finish()`；资料一步的 `done` 交回 `onDone()`；邀请一步和链接在 `onDone()` 在路上时显示忙碌。T5-c(b) 和 `1384f634` 的行为不变。`fake-tab.ts` 加 `settledYet`。
  - 测试：`in-session.test.ts` 加 4 个，`use-open-workspace.test.ts`、资料一步各加 1 个；W1 用 `sentHeld` 扣住结束，在三处核对 `enabledWithin(…)` 为假（故事 3 中 Ada 的"继续"、故事 2 的"以后再说"、故事 1 链接之后的"继续"）；`/create-workspace` 扣住下一页的脚本（`holdScripts`），按钮一直不能用到导航完成。变异 `M1.a`–`M1.g`。
  - 总体设计 7.7 的 P9 一条没有写兑现点，不改。
- **不跨会话留草稿**（`b772671f`，F-3）：`create-workspace/page.tsx`、`create-workspace-form.tsx` 中提上去的草稿删除。`e2e/fixtures/auth.ts` 的 `anotherTabSignsIn` 先在这一步写出，W1（页面）用它：填"Secret plans"，另一个标签页登录另一个账户，两个字段都是空的。`M2.a` 照第 5 步的 `useCreationForm` 写，在第 3 步的树上也同样被发现。
- **一处失败提示、一处退出登录**（`f1b62547`，F-4）：
  - 两个 hook 和它们的 vitest（3 个、2 个），取代 26 份相同的和 3 份自己标题的提示（`delete-workspace-modal`、`security.tsx`、`delete-token-modal` 保留各自的标题）。做决定的几处照旧决定：`use-create-workspace`、`invite-modal/refusal`、`use-workspace-invitation`、资料一步、`security.tsx`，以及个人设置的 general 表单和 API 令牌的表单。
  - 六处退出登录各保留之后的动作（`switch-account-modal` 只在交回 `true` 时关闭）。
  - 关键词规则 `refusal-toast`、`sign-out-toast`：67 条规则、3 个例外，没有命中。
  - 两个决定表（`refusal.test.ts`、`use-create-workspace.test.ts`）的 `toast` 行不再带文案的键，形状是 `{ kind: "toast" }`，文案由 hook 按错误给出；其余测试不改照样通过。
  - 这一步之后完整的 `make e2e` 93 个。变异 `M3.a`、`M3.b`、`P12.a`–`P12.d`。
- **小的生产改动**（`9ff34271`，F-5）：`useMemberColumns(workspaceSlug)`（`P3.a`）、`useCreationForm(onCreated)`（`M4.a`）、`EmptySpaceItem` 中的 span。
- **测试**（`b400e015`，F-6）：
  - `e2e/fixtures/browser.ts` 的 `refuseClipboardWrites(page)`（初始化脚本加一次 `evaluate`）；W4（页面）中 dave 的链接和 general 页的地址复制失败都提示（`P1.a`、`P1.b`）。
  - `anotherTabSignsIn` 用在 W3、W5、A6、A9、A11（和 W1），`newRecord`、`writeRecord` 成为模块私有；P8b 的四个 `ApiError` 改用 `refusal()`。
  - W1 故事 2、3 中 Bob 的页面经 `watchPage`：故事 2 按因果的顺序是 503 的失败、它的控制台错误、工作区首页的警告，故事 3 没有。故事 2 直接扣住结束（`FINISHED`）本身，不再靠排在它前面的那次写（`P10.b`、`P10.c`、`P10.d`；`T6f.2` 在新的直接核对上被发现，实现者自己与它重复的 `P10.a` 去掉）。
  - A2 不用改：`showSignIn` 已经等实例的回答和"Go to workspace"，`InstanceWrapper` 在有实例的设置之前不渲染页面；只把这一点写进说明。
- **非空断言的范围**（`733af2b4`）：两个新 hook 和它们的测试加进 `.oxlintrc.json` 的 `no-non-null-assertion`，照 2.14 对 P9 每个新模块的做法（`P12.e`）。
- **变异的记录**（F-7，没有提交）：第 2 节。
- **文档**（`c1e60774`，F-8）：spec 的第 2 节总览（评审时 114 个文件，修复轮之后 154 个）、`followInSession` 的新约定、共用的部分、2.1–2.6 的改动和数目、2.10 的退出登录、2.13（P7 的一句和 `useSignOut`）、2.14、新的 2.15、第 3 节第 4、12、19 条、第 4 节、第 5 节收尾、A.1、A.2、A.5、A.6、预检表的 L7；plan 的 T8.4 一行（P11）；`AuthenticationWrapper` 的说明（P7）；`store/user/index.ts:104` 的注释；M1-P2 交接的"处理结果（M3/P9）"。M3 设计 §15 不改（本提交改）。
- 门禁（`c1e60774`）：`make lint-web`（67 条规则、3 个例外、没有命中；web 356 等于上限）、`make knip`、`make test-web`（web 75 个文件、657 个）、`make e2e` 93 个；改到的故事各跑三次，129/129；修复轮改到的每个 web 的 TS 文件 0 条警告。
- 实现者写明的偏离：`refusal-toast` 的模式是 `setToast({…errorMessageKey(`，比 brief 的字面宽（`security.tsx` 的 `setError` 用那个字面写字段错误，是正当的，不加例外）；两个决定表的测试改了（上面）；A2 只改说明；`P10.a` 去掉；多出的 `733af2b4`。关注五条：(1) W1 403 行；(2) 规则的模式更宽；(3) 主题切换的 `setPromiseToast` 不在 26 份之中，规则看不见它；(4) 别的不交回 `Promise` 的跟进（成员页的离开 `void navigate("/")`、邀请的接受 `void openWorkspace(…)`）在导航之前就不忙，没有找到看得见的窗口；(5) 记录的 W1 行号来自第 6 步修订之前的树。裁定 F-10。

**修复的复审**（opus，`dba0d756..c1e60774`，八个提交、86 个文件，+917/−565）：通过，Critical 0、Important 0、Minor 4，另有小处。
- 它在副本上：
  - 跑过门禁：关键词 67 条、3 个例外，没有命中；turbo 54/54；web 356 条等于上限，P9 改到的 115 个 web 的 TS 文件 0 条（少的一条是正的 `tabIndex={1}`）；knip；vitest 75 个文件 657 个；`playwright test --list` 93 个、42 个文件；修复轮的五个页面故事（W1、W3、W4、W7、A12）11/11；
  - 亲手跑了 `I1.e`，在 W7:299 被发现（`closedByEscape` 在通过的方向不靠时间）；
  - 文件的数目和大小与 spec 相同。
- 裁定：F-1–F-3、F-5–F-7、F-9 在根上做到，没有带来新的缺陷；F-4 做到，带 M2、M3；F-8 大部分做到（M3、M4）。`followInSession` 的 12 处调用中只有打算交回 `Promise` 的跟进交回它，其余交回 `undefined`（`setToast` 交回字符串，不是 thenable），没有忙碌会卡在它自己的请求之外。记录有 165 行，与结果文件相同；两个没被发现的解释正确；抽查和 W1 的行号偏移成立。
- 关注 1、2、4、5 可以接受（离开：store 的修改在队列里先把工作区从列表中去掉，包装层显示"找不到"，页面上没有了离开的控件；接受：`answering` 在成功时不清掉，加载一直到卸载）；关注 3 归 spec 第 5 节的收尾（M3）。
- 四条 Minor：
  - **M1**：修复轮在 e2e 写了三个新的 `let x!:`（`settings-pages.ts:112`、`:116`，W1:296），是"先建 `Promise` 再取 `resolve`"的写法的第四到第六份（`holdAnswer`、A9、A12 已有三份），违反"没有新的 `!`"，A.7 的一句也不成立；
  - **M2**：`use-sign-out.test.ts:24`、`use-refusal-toast.test.ts:28` 两个测试没有一个有名字的变异让它失败；修复轮的测试没有重新建立第 4 节 W18 的一句；
  - **M3**：`theme-switcher.tsx:58` 是第二处按 `errorMessageKey` 的失败提示，可四句说 hook 是唯一的一处（hook 的说明、规则的 `why`、spec 2.14、2.15）；它是提示的 `Promise` 的错误状态，hook 的 `(error) => void` 表达不了，归 spec 第 5 节的收尾；
  - **M4**：spec 2.3、2.5 仍写 `{ kind: "toast"; message }`；2.15 说 W4 用 `closedByEscape`，可 W4 没有那一步（给邀请弹窗的 `ModalCore` 加 `handleClose` 的变异处处存活）；第 5 节的角色名一项没写 `T4f.M3a`；小处：`user/index.ts:105` 一行 175 个字符，2.15 做决定的页面漏了两个，A.6 变长的 Plane 文件仍是原型的。
- 说明（不计入）：`sign-out-toast` 只看提示的键；一个新的地方再从 store 解构 `signOut`（M3 那一类）每道检查都通过；可选的第二个模式能发现它（第 6 节收尾）。
- 结论：可以写评审记录、合并，之前用一个小提交补上四条。

**F-11 的补充**（`a8eed6b0`）：
- M1：e2e 的 lib 是 ES2023，没有 `Promise.withResolvers`，不改 lib。新文件 `e2e/fixtures/deferred.ts` 的 `deferred<T = void>()` 交回 `{ promise, resolve }`，没有 `!`，没有 oxlint 的警告；六处（`holdAnswer`、`holdScripts` 两处、A9、A12、W1）都用它，e2e 的 fixture 和故事中不再有 `let …!:`，没有断言变。A4、A11 的 `new Promise` 是别的形状，不改。
- M2：`M3.c`（退出成功时 hook 答 `false`）、`P12.f`（不是 nerve 的拒绝时用 Plane 的旧文案）各只由那一行发现，两行都留着。
- M3：四句改为"`setToast` 中按 `errorMessageKey` 的失败提示都经它；主题切换的 `setPromiseToast` 自己说失败"；spec 第 5 节加 `theme-switcher.tsx:56-59` 一项；代码不改。
- M4：spec 2.3、2.5 写 `{ kind: "toast" }`；第 5 节的角色名一项写明 `T4f.M3a` 存活；W4 在 POST 被扣住时加 `expect(await closedByEscape(page)).toBe(false)`，变异 `I1.f`。
  - **证明它时发现这个检查是瞎的**：`I1.f` 起初在 W4 上存活。页面上有两个 `role="dialog"`：弹窗，和删除 erin 的邀请留下的提示（`aria-modal="false"`）。`closedByEscape` 的定位因严格模式抛错，什么都接的 `() => false` 把它读成"没关"，所以不论 Escape 关没关，检查都通过。
  - 根上改在 fixture：只定位 `[role="dialog"][aria-modal="true"]`，只把 `TimeoutError` 算作没关，别的失败照常抛出。之后 `I1.f` 在 W4:291 被发现；用这个 fixture 的 `I1.d`（W3:370）、`I1.e`（W7:299）、`T9.7`（A12:131）重跑，仍被发现。
- 小处：2.15 做决定的页面加个人设置的 general 表单和 API 令牌的表单；A.6 列出修复轮让 11 个 Plane 文件变长（复审说 4 个），各为它自己的行为；`user/index.ts:105` 照 120 列换行；spec 的数目（155 个文件、e2e 22 个、修复轮的变异 30 个、全表 168 个中 166 个），A.7 记下三个 `let …!:` 和它们的去掉。
- 门禁：`make lint-web`（67 条；356）、`make knip`、`make test-web`（657，没有新的 vitest）、改到的故事各三次 105/105、`make e2e` 93 个、改到的文件 0 条警告。记录重新生成：168 个中 166 个。
- 关注：`enabledWithin` 有同样的什么都接的形状，交给 F-12。

**F-12**（`897235e5`）：`enabledWithin` 先 `expect(button).toHaveRole("button")`（严格：定位到零个、几个或不是按钮都失败），再一秒等它按 `getByRole("button", { disabled: false })` 可用；只有这次等待的 `TimeoutError` 算"一直不能用"，规则是与 `closedByEscape` 共用的模块私有 `timedOut()`。W1 的五处调用都恰好定位一个按钮，不用改。探查：W1:313 的定位改成十个按钮或没有按钮时，旧的 `enabledWithin` 都通过，新的都失败。由它发现的十个变异重跑仍被发现。门禁：`make lint-web`（67 条；356），W1、A12、W3、W4、W7 各一次 16/16，e2e 的 oxlint 0 条。spec 第 2 节的共用部分写明它现在核对什么。

**补充和 F-12 的范围复审**（sonnet，`c1e60774..897235e5`）：通过，只有说明。M1：e2e 中没有 `!:` 了，`deferred` 的类型成立，含义不变，e2e 的 `tsc` 干净；M2：`M3.c`、`P12.f` 由它们的行发现；M3 成立；M4：`timedOut` 把 `TimeoutError` 之外的都重新抛出，`aria-modal` 来自 Headless UI 的 Dialog（`I1.f` 被发现就是证据）；F-12：`toHaveRole` 会重试，W1 的五处调用不变；spec 的数目成立（159 减 4 份文档是 155；e2e 22；168/166；30；A.6 的 11 个）；R3（e2e 的 lint、格式、类型干净；web 356 等于上限）。

## 5. 最终实现与 plan 的差异

plan 的块记录的是当时的文字。执行中只改了 plan 的三处（9,105 → 9,112 行）：
- `7c9d08f7`：Task 2 的 `e2e/fixtures/workspace-pages.ts` 整文件块带上 `confirmDeletion`（T1-b），让之后的块照样能应用；
- `835a6f51`：Task 7 的 W6 一行按 role 的链接点"注册以接受"（T4-d）；
- `c1e60774`：Task 8 变异表的 `T8.4` 一行（P11，F-8）。

各 Task 自己的文字和修复轮的不同不改（P2；T1-d、F-8，P8b 的 F-4 先例），spec 在说规则的地方已改为与代码一致（F-8、F-11）。与 plan 不同的地方都来自第 3、4 节的裁定：
- **生产代码**：
  - T1-b：general 页的 `currentWorkspace` 改名 `workspace`；
  - T3-a：`invite-modal.tsx` 一个不起作用的 `clearTimeout` 删除；
  - T4-a … T4-c：邀请页有状态的主体是 `<AuthenticationWrapper>` 之下的子组件 `WorkspaceInvitation`；`answering` 的加载只在回答的视图里，连按两下只发一次；页面的标题和 `invite-modal/fields.tsx` 按 `t(ROLE_DETAILS[r].i18n_title)` 写角色；`EmptySpace` 的 `link` prop 删除，`EmptySpaceItem` 是按钮或链接，上限 359 → 357；
  - T5-a、T5-b：`onCreated` 交回 `Promise` 并被等待，`/create-workspace` 传 `openWorkspace`；`creationRefusal` 建在 `needsErrorBanner`、`fieldErrorKeys` 上；`availability` 改名；
  - **偏离 T5-c(b)**（Task 6，`3d0bcee1`）：新手引导的根为单人的工作区等引导的结束，创建一步等它，`onCreated` 交回 `Promise`。四个文件（根、根的测试、`steps/root.tsx`、`steps/workspace/create.tsx`）只多 `await` 和 `Promise` 的类型；W1 的故事 2 加 Bob；
  - T6-a：`finish(onFailed = failed)`，单人的结束被拒绝时先提示、再进入邀请一步；顶层的 `"email"` 键删除；`ProfileStore(api)`；
  - T7-b：`auth-screens/header.tsx` 的 `showsLink = type === SIGN_UP || signup_enabled`；
  - T9-a：停用弹窗在请求在路上时关不掉，`handleClose` 不再重置忙碌的状态；
  - 修复轮：F-1–F-5 的每一项和 F-8 的两段注释（第 4 节）。`followInSession` 的约定改为等跟进交回的 `Promise`；新的 `useRefusalToast`、`useSignOut`、`useCreationForm`；关键词规则 65 → 67 条；上限 356。
- **测试**：plan 之外加的：
  - 各 Task 的修正轮：
    - T1：W3 的 `confirmDeletion`，会话切换从放行起等回答；
    - T3：W4 中行内的角色选择（`invitationFormRow`）、每个拒绝在自己的行下、已忽略的邀请没有复制链接；
    - T4：W5 的故事 2 接着让 Carol 登录、中断一次之后的重试、连按两下只发一次；`invitation-view.test.ts` 出错的行带旧数据；`e2e/fixtures/auth.ts` 的 `removeRecord`；
    - T5：`use-create-workspace.test.ts` 加 3 个（两个换账户、一个被拒绝的检查）；W1 扣住 `PATCH /me/profile`；
    - T6：`root.test.tsx` 加 3 个（列表取不到；单人结束的两种换账户）；W1 让 Bob 的结束答一次 503；
    - T7：W6 的往返，和注册关闭的 nerve 上注册页的"登录"；
    - T8：`bodiesSentTo`；hook 测试的替身跟着轮次；
    - T9：A12 的取消、Escape 和最后的取消一步；W9 中扣住的第二次确认；store 测试的行带 `loginId`；
    - T10：包装层测试读说明；S2 的 `expectExactly`。
  - 修复轮：`use-refusal-toast.test.ts`、`use-sign-out.test.ts`；`in-session.test.ts` 加 4 个，`use-open-workspace.test.ts`、`steps/profile/root.test.tsx` 各加 1 个；两个决定表的 `toast` 行不带文案的键；e2e 的 `sentHeld`、`holdScripts`、`enabledWithin`、`closedByEscape`、`anotherTabSignsIn`、`refuseClipboardWrites` 和补充的 `deferred()`；W1、W3、W4、W7、A12 的新核对，A2 的说明，A6、A9、A11 改用共用的 fixture；P8b 的四个测试改用 `refusal()`。
  - 数目：vitest 在 `4b1334a5` 上 56 个文件、504 个；原型 73 个、640 个；Task 11 结束时 73 个、646 个（修正轮加的 6 个）；修复轮之后 75 个、657 个。e2e 93 个不变（修复轮的检查都加在已有的故事里）。
- **plan 自己的文字**（P2，记下、不改）：
  - Task 1 在 `7c9d08f7` 之后：W3 的块（约 :1165–1340：导入、会话切换中的 `deleteFromGeneralPage`、两句注释）、`workspace-details.tsx` 的块（:826、:848、:863 的 `currentWorkspace`）、:133 的 fixture 导出清单没有 `confirmDeletion`；plan 说 W3 是 395 行（修正轮之后 396，现在 397）；
  - Task 3–10 各自的块是修正轮之前的代码（T4-d、T5-b、T6-a 等；之后的块都在副本上核对过照样能应用）；
  - Task 6 的偏离的四个文件和 W1 不在 plan 的文字里（T5-c(b)）；
  - Task 5–11 的文字仍写上限 359（:5383、:6851、:7135、:8161、:8577、:8962、:9083，T4-f），全局约束的上限链停在 359（现在 356）；
  - Task 11 的块中两句不真的文字（实现者改了），和 T11-a 改的六处；
  - 修复轮第一次改到的 40 个文件和补充的 `deferred.ts` 不在 plan 的文件表里，列在 spec 第 2 节。
- **文件大小**：
  - 分支改过的 web 文件都在 400 行以内。最长的是 F-4 改到的 P8b 的 `label-dropdown.tsx` 345 行、`issues/select/base.tsx` 326 行；web 的新文件中最长的是 `use-create-workspace.test.ts` 189 行、`onboarding/root.test.tsx` 180 行；P9 改写的 Plane 文件中，删除工作区的弹窗 182 行（表单并入）、邀请页 174 行。
  - e2e：W1 403 行（裁定 F-10：约等于 400 行的上限；修复轮在它的五个页面版本中加了忙碌、草稿和 Bob 的页面的核对；复审说若要剪，把 W1 自己的 `availability()` 移进 `fixtures/api.ts`，不用拆）；W3 397 行、`assert/workspace.ts` 390 行、S2 383 行（都是原有的故事和 fixture，P9 加了页面版本和行）；新文件中最长的是 `workspace-pages.ts` 184 行。
  - 变长的 Plane 文件（spec A.6）：原型的 17 个中 16 个是本 Phase 的对象，只为使用方改到的是 `members-list.tsx` 89 → 91（裁定 P6）；修复轮和补充又让 11 个变长，各为它自己的行为，没有只为使用方改到的。
  - `tools/keywords.json` 2,209 → 2,241 行（Task 10）→ 2,307 行（修复轮的两条规则和样例）。

## 6. 移交事项

本 Phase 关闭的（spec 第 7 节）：
- M2 收尾交接第 1、2 节的页面一侧（这两节至此关闭）；第 13 节随本评审的 C4 关闭（第 2 节）；第 11 节的页面一侧做完工作区的部分，随 P10 关闭。
- P8a review 第 6 节 P9 一行；P8b review 第 6 节 P9 一行（E5：Task 8；M2 交接第 2、11 节的页面一侧：W2 的页面版本、新手引导的两步、general 页的时区）。
- P2、P3、P4b（P26）、P6 交给 P9 的行；M1-P2 交接的侧边栏偏好（修复轮写了它的"处理结果（M3/P9）"）；Codex 设计评审 M-4、4.3 第 2 条的页面一侧。
- 每一项的落点见 spec 第 3 节的交接表。

同一个里程碑之内的后续 Phase 和之后的里程碑（写进它们的 spec；spec 第 5 节各行照录，下面是执行之后的补充）：
- **P10（项目的页面）**：
  - spec 第 5 节 P10 一行照录（含 P8b spec 第 5 节 P10 一行，和 P9 的清扫看到的项目成员页的 `err.error`、`use-tab-preferences.ts` 的 `console.error`）。
  - 执行之后加的：修复轮定下的写法，P10 的页面照它们写：
    - 页面发出修改之后的跟进经 `followInSession`，跟进可以交回 `Promise`，按钮忙到它做完（F-2）；
    - 弹窗在请求在路上时关不掉：取消禁用，`ModalCore` 的 `handleClose` 为空（F-1，Task 9 的根因修法）；
    - 失败的提示经 `useRefusalToast`，退出登录经 `useSignOut`；关键词规则 `refusal-toast`、`sign-out-toast` 也看 P10 的页面（F-4）；
    - 两个标签页的故事用 `anotherTabSignsIn`；扣住回答用 `holdAnswer`、`sentHeld`；有期限的"一直不能用""关不掉"用 `enabledWithin`、`closedByEscape`；测试决定何时兑现的 `Promise` 用 `deferred()`（F-6、F-11、F-12）。
  - M2 交接第 11 节的页面一侧（项目 general 页的时区），本节随 P10 关闭。
- **P11（状态、标签的页面与清理）**：
  - spec 第 5 节 P11 一行照录：M2 的账户页面自己的提示（五个文件，裁定 P8）；个人主页的页面（故事 P9 的页面版本，C10）；P8a、P8b 留给 P11 的照旧。
  - 浏览器核对看到的：
    - 访客在界面上没有办法离开工作区：成员页对访客没有权限（W11 的页面一半）；
    - 照录的个人主页一项现在的样子：`/{slug}/profile/{id}` 仍调 Plane 的 `/api/workspaces/{slug}/user-issues/…`（404），工作项的列表一直是骨架（工作项是 M4 的，C10）。
- **M4**：spec 第 5 节 M4 一行照录（`issues/issue-modal/base.tsx:233`、`:323`；P8a、P8b 交给 M4 的照旧）。浏览器核对：个人主页上工作项的列表（上一条）。
- **M6**：spec 第 5 节照录（迭代、模块的弹窗读 `error?.error`；视图的修改之后的跟进）。
- **M7**：spec 第 5 节照录（P19：已挂载的包装层换 slug 时不重取列表；收藏的 `console.error`）。
- **M8**：spec 第 5 节照录（webhooks 的三处 `error?.error`）。
- **契约**：P9 没有新的。
- **收尾**：
  - spec 第 5 节照录：P9 改到的文件中仍写死的英文（预检 L8 和评审 M5 的清单）；PF-L5、PF16 的工具缺口和评审找到的盲点；角色名只有一个来源（搁置项 P6，含邀请页标题的 `T4f.M3a`）；主题切换的第二处失败提示（`theme-switcher.tsx:56-59`，F-11）；P9 让它们不再成立的"将来"句（M3 设计 :1176、:1232、:1276，`plane-diff.md:242`）。
  - 执行之后加的：
    - `confirm-workspace-member-remove.tsx` 的英文（"Leave workspace?"、"Remove {名字}?"和移出的正文）：整分支评审说归 P11 或收尾；修复轮（F-1）之后它也是 P9 改到的文件。
    - 浏览器核对看到的：
      - 中文首页的问候是"早上 晚上, 花子"：`common.good` 在中文是"早上"（Plane 的译文）；日期（"Friday, Oct 9"）没有本地化；
      - 中文下仍是 Plane 的英文：设置侧边栏的"Workspace settings""Project settings"、"Search commands..."、规模的选项（"Just myself"…）、邀请页的页面标题"Workspace Invitations"、工作区菜单中的角色"Admin"（加进 P6 的地方）；
      - "找不到工作区"和访客的没有权限的设置页不设页面标题，前一个标题留着（例如"Sign in - Nerve"）；
      - "You declined this invitation."对打开已忽略的链接的任何人都显示，不论是不是被邀请的人。
    - 关键词规则 `sign-out-toast` 只看提示的键：再从 store 解构 `signOut` 的新地方每道检查都通过。复审给出可选的第二个模式（`\{[^}]*\bsignOut\b[^}]*\}\s*=\s*useUser\(\)|useUser\(\)\.signOut`，在 HEAD 上只命中 hook，在 BASE 上命中六处），收尾决定是否加。

## 7. 已知限制

- **general 页发出它的三个字段**（F-9）：保存时间区也写回名称和规模，另一位管理员在此之间的修改被盖掉（最后写的算）。spec 2.1、A.4 把"编辑的字段"定为表单的字段，W3 钉住它。
- **PUBLIC 页面在退出登录时留着子组件**（P7，T4-e）：登录和换账户都重新挂载子组件，退出登录不。邀请页是唯一的 PUBLIC 页面，它所持的（`mismatched`、`answering`）只在登录之后的视图里读，由 `invitation-view.test.ts` 和 W5 的故事 2 钉住。整分支评审决定不按会话加 key；将来的 PUBLIC 页面照 `AuthenticationWrapper` 说明中的一句写。
- **邀请页标题中的角色名没有核对**（`T4f.M3a` 存活）：标题没有渲染的测试，`fake-i18n` 丢掉参数，W5 读英文，英文的 `ROLE` 与译文相同。收尾的角色名一项（第 6 节）。
- **主题切换的失败提示不经 `useRefusalToast`**：`setPromiseToast` 的错误状态是加载中的提示变成拒绝，hook 的 `(error) => void` 表达不了；规则 `refusal-toast` 只看 `setToast`。收尾决定（第 6 节）。
- **两条新规则看不到的形状**（复审）：`refusal-toast` 漏掉提示对象中消息之前有嵌套 `{…}` 的（例如标题是 `t("…", { name })`），和在调用之外拼出的消息；`sign-out-toast` 漏掉再从 store 解构 `signOut` 的（第 6 节收尾）。
- **跟进自己出错时交给 `reportError`**：`followInSession` 从不拒绝。`reportError` 只在浏览器中有，Node 24 没有；只有 `in-session.test.ts` 让跟进失败，它替换了 `reportError`（复审：不是缺陷）。
- **不交回 `Promise` 的跟进**：成员页的离开（`void navigate("/")`）和邀请的接受（`void openWorkspace(…)`）在导航之前就不忙。没有看得见的窗口：离开时 store 的修改先把工作区从列表去掉，包装层显示"找不到"，没有离开的控件；接受时加载一直到卸载。新的约定允许它们交回 `Promise`，没有改（F-10）。
- **有期限的 e2e 核对**：`closedByEscape`、`enabledWithin` 等一秒，通过的方向不靠时间；失败的方向要离场的过渡（约 200 毫秒）在一秒内结束，五倍的余量。W3 的会话切换在放行之后等 2 秒（spec 第 6 节）。
- **只由端到端核对的**：邀请表单行下的原因（W4）、停用弹窗里的原因和新的确认清掉旧的（W9）。组件的 vitest 是服务端渲染，看不到点击之后才设的状态，那几个变异的 vitest 一层存活（第 2 节）。
- **修复轮的测试没有另跑一次 W18**：复审逐个对照修复轮的变异，找到的两个由补充补上；没有对它们跑 W18 的全表。
- **复现只到修订**：逐 Task 复现是修订时的（spec A.10）；修正轮和修复轮改了复现没有见过的代码，没有再复现。
- **M2 的账户页面自己的提示不核对会话**（裁定 P8）：P11 之前，迟到的提示可能显示在另一个账户的页面上。
- **每次挂载都重取、聚焦时不取**（P8a 的 F-1）；**已挂载的工作区包装层换 slug 时不重取列表**（P19，M7）。
- **整分支评审的说明**：直接打开个人设置时，列表取不到则侧边栏是空的、什么都不说（页面本身的内容不靠它）；`workspace-details.test.tsx:49`、`:56` 两次写同一个表单类型。都不改。
- 浏览器核对看到的几处都不是 P9 的缺陷，已列在第 6 节（P11、M4、收尾），这里不重复。
