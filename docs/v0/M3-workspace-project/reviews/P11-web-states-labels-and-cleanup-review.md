# M3/P11 状态、标签的页面与清理：评审记录

| 项 | 内容 |
|---|---|
| Phase | M3/P11 `web-states-labels-and-cleanup` |
| 日期 | 2026-10-12 |
| 结论 | **通过**。<br>10 个 Task 逐个实现、逐个评审，都通过。Task 1、3、5、7、10 各有一次评审之后的修正轮（Task 5 的一轮是三个提交），Task 7 在评审之前另有一轮（裁定 T7-a）；Task 2、4、9 的评审发现由裁定带进之后的 Task（T2-a–T2-c 进 Task 3，T4-a–T4-d 进 Task 5，T9-a 进 Task 10），Task 6、8 的 Minor 交给修复轮或交出去。各 Task 的评审共有 8 条 Important：Task 3、4 的各一条是执行中发现的缺陷（状态一行的删除按钮看不出焦点；标签表单的颜色按钮没有名字、看不出焦点），Task 2、5 的各一条和 Task 3、4 的各另一条是没有会失败的检查的性质（代码本来对），Task 9、10 的各一条是文档（`then()` 的分法、只记在台账里的收尾事项）；其余 20 条是 Minor。分支的持续集成到 `83f30b36` 每次都通过（Task 1 的第一次运行的 e2e 被下一次推送取消；第 2 节末）。<br>整分支评审的结论是"修复轮之后可以合并"：Critical 0、Important 0、Minor 8。修复轮一次分派、三个提交（`c9e71088`、`0d2061f5`、`83f30b36`），落实裁定 F-1–F-13（评审的 M1–M8 和搁置项 P1–P5）。修复的复审（opus）Critical 0、Important 0、Minor 3：裁定 F-14 的四处 spec 文字改在本记录的提交里，F-15 交给收尾（第 6 节）。<br>控制者的浏览器核对 C9、C10 在 `83f30b36` 的构建上都通过（第 2 节）。<br>负责人在本阶段答了一项产品决定：Q5（2026-10-11，"菜单加入口，收尾做"），访客离开工作区的入口是 M3 收尾的一项任务（第 3 节）。 |
| spec / plan | [spec](../specs/P11-web-states-labels-and-cleanup.md) / [plan](../plans/P11-web-states-labels-and-cleanup.md) |
| 设计 | [M3 设计](../M3-design.md) 第 12 节 P11 |
| 分支 | `worktree-m3-p11`，从 `main` 的 `7d6a416b`（P10 合并）分出 |

## 1. 评审方式

- **spec 和 plan 由架构子任务产出**（`2bdb1536`）：
  - 设计 12 节 P11 的草稿是 9 个任务，plan 是 10 个 Task、6,719 行（最长的 Task 2 是 1,458 行），spec 489 行；附带原型（`$M3TMP/p11proto`）和逐 Task 的快照（`$M3TMP/p11snap`）。从 `7d6a416b` 的新副本照 plan 复现，每个 Task 之后门禁通过，最终与原型逐文件相同（3,228 个文件）。
  - 63 个变异在最终原型上都被发现。W18：P11 新加或改写的 112 个测试每个至少失败一次（plan 的变异让 50 个失败，28 个探查让其余 62 个）。
  - spec 第 3 节有 8 条待裁定。控制者裁定 Q1–Q4、Q6–Q8，Q5 是产品决定，交给负责人，负责人 2026-10-11 答复"菜单加入口，收尾做"（`$M3TMP/p11-architect-rulings.md`；第 3 节）。
- **执行前的预检**（opus，只读，`$M3TMP/p11-preflight/preflight.md`，在 `2bdb1536` 上）：高 1、中 1、低 9。
  - 复现干净：每个 Task 之后门禁通过，最终与原型逐文件相同（3,228 个文件）。P6、P7、P8、P9、W11、S2 的故事各跑 3 次，57 个都通过。
  - 它自己写的 31 个变异中，28 个由 plan 的检查发现。存活的三个：`PF17`（删去的 prop 的默认值，只由评审看住，接受）、`PF18`（下面的 L9）、`PF29`（Q8 还没有落实；用它写的原型测试就被发现）。
  - Q2 的核对：`AlertModalCore` 的其余 11 个调用方在每条被拒绝的路上都把忙碌复位；两个工作项的删除确认框的 `onSubmit` 可以不传，是一条今天走不到的路。Q8 的 (b) 写得出：它写了原型的测试。
  - H1：Task 8 删去工作区设置侧边栏的 `className`，可移动端的设置导航经 `hamburgerContent` 传它，删了就是看得见的回退，没有门禁看得到（第 3 节 A-H1）。
  - M1：security 页的 API 令牌的两个弹窗不经会话的核对：换账户之后兑现的新建在另一个账户的页面上显示新令牌、把它下载成文件。P11 的文档和 W3 的清扫却说 M2 的账户页面做完了（第 3 节 A-M1）。
  - L1–L9：标签的删除确认框"原来先关"不实；2.12 对 `lateSettlings` 两种的说法不实、拼错；nerve 有四种 `not_allowed`，移出组的那一行走不到；A.6 把三处复制链接错分为"交回调用方"；页面 vitest 的替身不看项目的 id、状态的替身是空的；行内表单的一个测试标题说的比断言多；标签一行的把手和组的展开按钮没有名字；顶部的表单在编辑别的标签时显示"Update"、提交的是新建；写错位置的 `return;` 过得了每道检查。
- **修订**（`1fa8f921`、`18b5ce41`）：落实 Q1–Q8 和预检的每一条（控制者的裁定见 `$M3TMP/p11-amend-brief.md`，落点见 spec 第 3 节末"预检之后的修订"）。
  - `1fa8f921`：M3 设计第 12 节收尾加负责人的两项：`ModalCore` 的重写（2026-10-10）和离开工作区的入口（2026-10-11）。
  - `18b5ce41`：spec 和 plan。plan 7,485 行，10 个 Task（Task 2 1,458 行、Task 7 1,218 行、Task 5 1,212 行）。72 个变异在最终原型上都被发现：62 个在自己的 Task 的树上，10 个由 Task 3 或 Task 5 的页面版本才发现。W18：122 个测试，32 个探查，每个至少失败一次。
  - 从 `7d6a416b` 的新副本重新逐 Task 复现：每个 Task 之后门禁通过，最终与原型逐文件相同（3,231 个文件，0 处差异）。
  - e2e 106 → 110，vitest 95 个文件、849 个 → 106 个、964 个，web 的 oxlint 上限 348 → 346 → 337 → 303，editor 65 → 64；`--rows M3` 删去 26 行（A-H1 留下 `className`），产品代码的行 80 → 54。
- **执行**：
  - 开始时的裁定（S1–S3）见第 3 节。
  - 10 个 Task 各由一个实现子任务完成（不指定模型），再各由一个评审子任务（opus）检查是否符合 spec、质量是否达标。评审之后的修正轮由同一个实现者做，sonnet 做范围复审；Task 7 在评审之前另有一轮（裁定 T7-a），评审看两个提交。同一时刻只有一个写入者。
  - 实现者在每个 Task 里照 plan 的变异表跑变异（每个 Task 一个新目录 `$M3TMP/p11mut/tN`，裁定 S3），并加上自己的探查。每个 Task 的实现和修正轮都跑过 `make lint-web`、`make knip`、`make test-web` 和本地的 `make e2e`（裁定 S2：Docker Desktop 起着）；Task 10 和它的修正轮只改文档：Task 10 跑 `make lint-web`、`make knip`、`make test-web`（取缓存），修正轮跑 `make lint-web`，都没有跑 e2e。
  - **执行中发现并修好的生产缺陷**（都在分支上修好）：
    - 执行之前（预检；修订改了 plan 的代码）：
      - H1（A-H1）：Task 8 原来要删去工作区设置侧边栏（`settings/workspace/sidebar/root.tsx`）的 `className`。移动端的设置导航把这个组件作为值交给 `hamburgerContent`，以 `className` 渲染它（`settings/mobile/nav.tsx:38`）；删了它，手机宽度上工作区设置的菜单就没有高度的限制、圆角的边框和底部的留白。`tsc`、knip 和每个测试都通过：`deadsym.mjs` 看不到经组件类型的变量的读，这一处"死"的清扫会把它删掉；
      - M1（A-M1）：security 页的 API 令牌的两个弹窗在会话之外跟进：另一个标签页换了账户之后才兑现的新建，在另一个账户的页面上显示生成的密钥、把它下载成文件；撤销同样在那里提示、关上。照 7.7 的弹窗规则核对时又发现新建的弹窗在请求路上时取消照样能按（违反总体设计 7.7 的 F-1：请求在路上时弹窗关不掉）；
      - L7（A-L7）：标签一行的拖动把手是没有名字、看不见（`opacity-0`）的 Tab 停留点，按 Enter 什么都不做；标签组的展开按钮没有名字；
      - L8（A-L8）：plan 的代码让顶部的新建表单在编辑别的标签时显示"Update"、提交的却是新建（原来按了什么都不做）；
      - Q2 的加固：两个工作项的删除确认框的 `onSubmit` 可以不传，有了 F-1 之后，有权限而不传时确认框一直忙、再也关不掉（今天 11 个调用处都传）。
    - T1（评审的 m1，裁定 T1-b）：`form-refusal.ts` 直接读 `PROBLEM_MESSAGES[code]`，一个没有文案的已被占用的码会无声地消失（`useFormRefusal` 丢掉 `undefined`）；改为经 `errorMessageKey`，读错误的唯一的函数（`716b8351`）。
    - T3（实现者的关注 1、评审的 I1，修正轮 `39222646`）：状态一行的删除按钮留着 Plane 的 `focus:outline-none`，Tab 到达时看不出焦点。
    - T4（评审的 I1、M1，裁定 T4-a、T4-c，改在 Task 5 的 `4e7285e0`）：
      - 标签表单的颜色按钮是它的第一个 Tab 停留点，没有名字，`focus:outline-none`；状态表单的颜色按钮也没有名字；
      - 改名一个没有颜色的标签时发出 `color: "#000"`，顺带把它改成黑色（Plane 原有，P8b 留着）。
    - T5（裁定 T5-a、T5-b、T5-e、T5-f）：
      - `@nerve/ui` 的 `CustomMenu` 的省略号按钮带 `outline-none`，标签一行的"Options"Tab 到达时看不出（实现者在 Task 5 的提交里改在根上，T5-a 接受；它的 17 个文件 17 处随之一样）；
      - propel 的 `Button` 的基础类 `focus-visible:outline-none`，没有别的焦点样式：应用中每个 `Button` 在键盘取得焦点时都看不出（T5-b，`5f178ceb`）；
      - 之后浏览器的轮廓紧贴实心的按钮、颜色几乎相同，看起来只是宽了 1 像素的边；propel 的 `IconButton` 同样 `focus-visible:outline-none`（T5-e，`9ebc375e`：两者的基础类都改为 `focus-visible:outline-offset-2`）；
      - 项目设置侧边栏的返回箭头（`IconButton`）没有名字（T5-f，`baae317a`：`common.go_back`）。
    - 整分支评审（修复轮 `c9e71088`）：负责人的标准不许的死代码和重复，不是今天看得到的缺陷：
      - M1：`groupedStates` 从组列表一层层传到每一行，没有读者（只出现在副作用的依赖里）；
      - M2：标签的行内表单留着 Plane 的显示和隐藏的接线：`labelForm` 总是真，`setLabelForm` 让每次关上设两次，`forwardRef` 落到没有读者的 `scrollToRef`；
      - M3：状态的根组件再取一次项目的状态，不经权限的门（与项目的包装层是同一个键，今天只因它挂载的位置才无害）；
      - 修复轮另删去 `dashed-circle.tsx` 每个调用方都传、从不读的 `radius`（Task 8 评审的 m2，F-6）。
  - **代码本来对、只有评审或变异守着的性质**（当轮或之后的 Task 补上检查）：
    - 预检：页面 vitest 的替身不看项目的 id（L5 → `T3.10`）；行内表单的 vitest 不看名称下的文案（L6 → `T4.11` 在 vitest 一层也被发现）；写错位置的 `return;`（L9 → `no-unreachable`、`T9.3`）；标签的把手以状态登记（Q8 → `label-drag-n-drop-HOC.test.tsx`、`T5.14`、`T5.15`）。
    - T2（评审的 I1、m1，裁定 T2-a、T2-b，补在 Task 3 的 P6）："Mark as default"忙到跟进做完没有检查：代码是对的（`await markDefault` 和 `disabled={isDefault || isLoading}`），可改成 `void markDefault` 或去掉 `|| isLoading` 的变异都存活，那样连按两下会发两次（nerve 两次都答 204）→ `T2.14`、`T2.15`；组标题收起没有检查（`aria-expanded={true}` 存活）→ `T2.16`。
    - T3（评审的 I2）：项目成员拖不动状态没有检查（`canDrag: () => isDraggable` 存活）→ P6 中成员把一个状态拖到另一个上，一秒之内不发请求，顺序不变（`sentWithin`，`T3.12`）。
    - T4（评审的 I2、M2，裁定 T4-b、T4-d，补在 Task 5 的 P7）：删除标签被 nerve 拒绝时确认框留着（`T4.15`）；一行的"Remove from group"发出 `{ parent_id: null }`（`T4.17`）。
    - T5（评审的 I1，裁定 T5-c）：两个 getter 读给它的项目没有检查（读 `this.projectLabels` 的树、读 router store 的按组分的状态都存活）→ 两个 store 测试的"各项目分开"一条各加一行，`T5.19`、`T3.13`。
    - T6（裁定 T6-b）：`T6.1`、`T6.2` 都停在 P9 的 :33；探查 `P6.1`–`P6.4` 证明之后的检查也会失败，它们进变异的记录。
    - T7（实现者的关注 1，裁定 T7-a，评审之前的一轮 `746c1300`）：新建令牌的请求在路上时"Generate"不可用、Escape 和点外面不关，撤销的确认框收它的忙：浏览器中成立，plan 没有检查 → A11，`T7.10`–`T7.12`。
  - **文档中会把错的交给之后的阶段的**（当轮或下一个 Task 改正）：
    - T9（评审的 I1、m1，裁定 T9-a，改在 Task 10）：spec 附录 A.6 和第 5 节 M6 一行把模块的两处加入收藏算作拒绝没有处理，原因写成"菜单的 `action` 是 `() => void`"。它们是按钮的处理函数，交给 `setPromiseToast`，拒绝已被接住：照写就交给 M6 一个不存在的缺陷。真的分法是链后 23、交回调用方 6、没有处理 6；"命令面板的五处复制"是六处；`snooze/modal.tsx` 是通知的暂缓，不是收集箱的。
    - T10（实现者的报告，裁定 T10-a）：Task 10 的携带项中三个数在 HEAD 上不真：`CustomMenu` 是 17 个文件 17 处，不是 18；项目下拉框只有工作项的表单给正的 `tabIndex`；`IconButton` 没有 M5 的调用处。文档写真的数。
    - T10（评审的 I1，裁定 T10-b，修正轮 `0c98207d`）：四项收尾事项只记在不进 git 的台账里（T5-d、T5-g、T8-a、T8-c），spec 第 7 节说收尾照录第 5 节，它们会丢。修正轮把它们写进 spec 第 5 节收尾的键盘清单和 `showTooltip` 一条。
    - 整分支评审的 M5–M8（修复轮 `83f30b36`，第 4 节）：前端改动清单说侧边栏的外层手柄不再是停留点；`Button` 的轮廓一行漏了 `getButtonStyling` 的调用方；M4–M8 的 M1 收尾交接仍把 `always-return` 列为要清的规则；收尾的写死的英文清单有不存在的、漏了 P11 自己的。
  - **执行中删除的（plan 之外）**：F-1 的 `groupedStates` 一串；F-2 的 `labelForm`、`setLabelForm`、`forwardRef`、`scrollToRef` 和 `reset` 的导入；F-3 的根组件的取数和它的导入、替身；F-6 的 `radius`（`backlog-group-icon.tsx` 的 `radius` 常量随之没有读者，一并删除）；T7-b 的 `fake-form.ts` 代替三份 react-hook-form 的替身；F-4 之后空了的 `e2e/fixtures/downloads.ts`。
  - **规则的失误**（都只读，没有写入任何东西）：预检一次把 `cd` 与 `grep` 组合，几次在路径中用了 shell 变量；Task 2 一次调用跑了两个 git 命令；Task 7 多次把 `cd` 与别的命令组合，提交的那次跑了两个 git 命令，评审之前的一轮用了一次 `git -C … status`；修复轮一次把 `cd` 与命令（含一个 git 命令）组合，一次单独的 `cd`；整分支评审两次把 `cd` 与只读的命令组合；架构子任务的修订一次 `cd … && tail`。裁定 T7-c 之后每次分派都重申规则。
- **整分支评审**（opus，`18b5ce41..0c98207d`，129 个文件，+3,253/−1,054；第 4 节）：Critical 0、Important 0、Minor 8；结论"修复轮之后可以合并，之后做 C9、C10"。
- **修复**：一次分派（不指定模型），三个提交（`c9e71088` 代码、`0d2061f5` e2e、`83f30b36` 文档），落实 F-1–F-13。修复的复审（opus，`0c98207d..83f30b36`）：Critical 0、Important 0、Minor 3，结论"可以做浏览器核对和评审记录"；裁定 F-14、F-15（第 3、4 节）。
- **控制者的浏览器核对** C9、C10（M3 设计 9.7）在修复轮的头 `83f30b36` 的构建上做，两项都通过（第 2 节）。
- **持续集成**：见第 2 节末。

## 2. 验收标准核对（spec 第 4 节，M3 设计 12 节 P11 的完成线）

设计的完成线是四项：全部 21 个故事的页面版本和接口版本通过；`domains.mjs --rows M3` 没有输出，或剩下的每一行写进评审；各包的 oxlint 上限已调低；C9、C10 写进评审。spec 第 4 节把它们和 spec 自己的标准写成下表的 10 项。

| # | 标准 | 结果 |
|---|---|---|
| 1 | 全部 21 个故事（W1–W12、P1–P9）的页面版本和接口版本通过：worktree 中 `make e2e` 110 个（`7d6a416b` 是 106 个）；`make test-web` 通过（web 应用 106 个测试文件、965 个测试） | **通过。** 本地（裁定 S2）每个 Task 的实现和修正轮（只改文档的除外）都是 plan 的数目：Task 1、2 106 个，Task 3、4 108 个，Task 5 109 个，Task 6–9 110 个；修正轮把检查加在已有的故事里，数目不变。修复轮的头上 110 个；P6、P7、P9、W11、S2、A11 的页面版本和 P8 各 3 次，23 个测试 69 次都通过。web 的 vitest：`7d6a416b` 上 95 个文件、849 个；各 Task 提交时 849、888、893、921、932、932、965、965、965、965 个（spec A.1）；修复轮之后 106 个文件、965 个。写 plan 时是 964 个：Task 5 的提交照裁定 T4-c 加了没有颜色的改名一行。整分支评审在它的副本上跑完整的 e2e：109 个通过，只有 S3 失败（副本不是 git 仓库，F4）。持续集成见本节末。 |
| 2 | P6、P7、P9、W11 的页面版本：管理员的修改（请求体、扣住时的忙碌、被拒绝时的说明）、成员只能看、是工作区管理员的项目成员能改、访客看到没有权限、访客打开个人主页不取工作项（S2） | **通过。** 新的故事 `p6-states-page.spec.ts`、`p7-labels-page.spec.ts`、`p9-guest-profile.spec.ts`；W11 的页面版本加状态、标签的设置；S2 的四个账户各打开两页，访客另打开两个个人主页（2.11）。整分支评审的权限表：每个角色在两页和个人主页上被提供什么，对照 nerve 的 `rules.go:112-125`、`decide.go:63-85`，每一格有一个会失败的检查；挂载时的请求逐页、逐个角色与 2.11 和 S2 的 `REQUESTS` 相同。C9、C10 在浏览器中核对（见下）。 |
| 3 | 2.12 的每一处有一个"不经 `followInSession` 时失败"的 vitest | **通过。** 2.12 的 8 行各有换账户之后兑现的 vitest（`lateSettlings`；交回东西的 API 令牌的新建用 `lateSettlingsAnswering(created)`）；W3 一类 17 个变异都由 vitest 发现（spec A.2）。整分支评审逐行对照代码：每一处在表写的地方调 `followInSession`；应用中 49 处 `followInSession` 没有一处包在 `try` 里，没有一处的 `failed` 什么都不做。 |
| 4 | `domains.mjs --rows M3` 剩下的每一行写明文件、名字、谁读或传它（spec 附录 A.9），按路径误归的 41 行列出（spec 第 5 节）；写进评审是控制者的 | **通过**，见下面的"`--rows M3` 的 218 行"：产品代码的 54 行和误归的 41 行在 spec 附录 A.9 逐行或逐文件写出，测试文件中的 123 行在本节按文件列出。 |
| 5 | oxlint：P11 手改的文件 0 条警告，除了 `alert-modal.tsx` 的一条和 Task 9 的文件的其余 14 条；P8a、P8b 留下的只经机械步骤到达的文件 0 条；`promise(always-return)` 全仓 0 条；上限 web 348 → 303、editor 65 → 64，其余不变；没有新的抑制 | **通过。** 整分支评审对每个改到的文件跑 oxlint：115 个 TS、JS 文件共 15 条，在 6 个文件中，正是裁定留着的那些（`alert-modal.tsx` 1 条，第 3 节第 3 条；`cycles/modal.tsx` 1、`issue-activity/helper.tsx` 2、`module-card-item.tsx` 5、`snooze/modal.tsx` 5、`webhook.store.ts` 1，第 3 节第 8 条）。修复轮手改的文件 0 条。上限 348 → 346（Task 2）→ 337（Task 8）→ 303（Task 9），editor 65 → 64，ui 15、propel 11、utils 7、hooks 3、constants 1 都等于条数。plan 改到的 105 个 web 的 TS 文件 61 → 15 条（`always-return` 35 → 0）；P8a、P8b 留下的 6 个文件 15 → 0；抑制 181 → 181（spec A.5）。变异 `T8.1`、`T8.2`、`T9.1`、`T9.2` 由上限发现。 |
| 6 | `node tools/keywords.mjs` 通过：69 条规则、3 个例外，没有 `until: "M3"`；旧 `ProjectService` 只有三个方法；`tsc`、knip 通过 | **通过。** Task 10 的核对输出 `69 [["analytics","M6",1],["brand","M9",3],["plane-workspace-urls","M4",2]]`，`ProjectService` 的三个方法在 `:18`、`:26`、`:38`；每个 Task 和修复轮的 `make lint-web`、`make knip`；整分支评审的副本上没有命中。`project-address-route` 加上 P11 的五个文件（Task 3、5），`T3.1`、`T5.1`、`T5.3` 由它发现。 |
| 7 | 根目录 `.oxlintrc.json`：P11 新建的模块和测试在 `no-non-null-assertion` 的范围里，每一组有一个变异让 `check:lint` 失败；`no-unreachable` 是 error（`T9.3`） | **通过。** W12 的 6 个变异（`T1.4`、`T2.13`、`T3.9`、`T4.12`、`T5.13`、`T7.6`）都由 `check:lint` 发现；`fake-form.ts` 在范围里（裁定 T7-b），`T7.6` 在修复轮的重跑中照旧被发现。`T9.3` 由 `no-unreachable` 发现。 |
| 8 | 没有新的 `as`、`any`、`!` | **通过。** spec A.7：P11 改到的 88 个 web 源文件中 `as` 26 → 26、`any` 7 → 7、`!` 2 → 2；`castsall.py` 对整个 Phase 的 TS 文件没有找到新的。修复轮之后另有 `dialogs.ts` 的 `import { setTimeout as sleep }`（导入的别名），A11 改写的两行各带的 `Response.json()` 的类型使文件中这样的 5 处之前之后相同。整分支评审扫过差异：只有 `use-work-item-filters-config.tsx` 原有的 `as Project[]`（Task 8 只给那一行的参数改名）。 |
| 9 | 逐 Task 复现：从 `7d6a416b` 照 plan 应用，每个 Task 之后门禁通过，最终与原型逐文件相同 | **通过（修订时）。** 修订 `18b5ce41` 的 plan 从 `7d6a416b` 的新副本逐 Task 应用：每个 Task 之后门禁通过，最终与原型 3,231 个文件 0 处差异（spec A.10）。各 Task 的修正轮和修复轮改了原型没有的代码；修复轮照裁定 F-12 由各 Task 的提交重新写出 plan 的块，核对 440 个块都恰好应用、逐 Task 应用之后与那个 Task 的树相同；复审在各 Task 结束的提交上重放了全部 440 个块（第 5 节）。修订之后没有再逐 Task 跑门禁。 |
| 10 | 控制者的浏览器核对 C9、C10 写进评审（9.7） | **通过**，见下。 |

**`--rows M3` 的 218 行**（完成线的第二项；`deadsym.mjs`、`domains.mjs` 的 M3 筛选）：
- 修复轮的副本（同步到 `0d2061f5`；`83f30b36` 只改文档）上的结果是 `$M3TMP/p11mut/fw/dead/fw.m3`：218 行；`deadsym` 是死的 prop 362、成员 975、导出 2。整分支评审在 `0c98207d` 上的 `$M3TMP/p11final/dead/head.m3` 也是 218 行，两份只差 `states-labels.test.tsx` 的 12 行的行号（F-3 删去了它的两行替身）。`7d6a416b` 上是 217 行。
- 分成三份：产品代码的 54 行（spec 附录 A.9 的表逐行写出谁读它们，或它们属于 M4）；按路径误归到 M3 的 41 行（`@nerve/types` 的 `workspace-notifications.ts` 17 行归 M7，`workspace-draft-issues/base.ts` 16 行归 M4，`progress-stats/label.tsx` 5 行和 `state_group.tsx` 3 行归 M6；spec 附录 A.9 和第 5 节）；测试文件中的 123 行（下表）。54 + 41 + 123 = 218。
- 删去的 26 行见 spec 附录 A.9；没有新出现的产品代码的行。
- 测试文件中的 123 行在 30 个文件里。每一行是一个测试自己的东西，由同一个测试读：`it.each` 的行的类型（每一行的列）、`vi.hoisted` 或模块级的持有对象的字段（替身写入、断言读出）、`vi.mock` 替身组件的 props，以及测试自己的表和辅助类型。`7d6a416b` 上是 27 个文件 96 行；P11 的 4 个新测试文件加 30 行，`create-update.test.tsx` 4 → 3 行，`create-update-label-inline.test.tsx` 2 → 0 行（Task 4 重写了它）。路径相对于 `web/apps/web/`：

| 文件 | 行 | 名字（`:行号`） | 是什么 |
|---|---|---|---|
| `app/…/settings/projects/[projectId]/states-labels.test.tsx`（P11 新） | 15 | `permissions`、`projects`、`projectId`（:28）；`label`、`name`、`isEditable`（:56）；`groupedStates`、`name`（:66）、`isEditable`（:67）；`title`（:75）、`children`（:77）、`control`（:79）、`children`（:80）、`actions`、`label`（:82） | `vi.hoisted` 的调用者；标签一行、组列表、页头、内容、标题、按钮、空状态的替身的 props |
| `app/…/settings/projects/[projectId]/admin-pages.test.tsx` | 9 | `permissions`、`projects`（:26）；`title`（:30）、`children`（:32）、`value`（:36）；`page`、`route`、`Component`、`shows`（:42） | `vi.hoisted` 的调用者；替身的 props；测试的页面表 `Page` |
| `core/components/labels/label-drag-n-drop-HOC.test.tsx`（P11 新） | 8 | `value`、`set`（:23）、`current`（:24）、`deps`、`cleanup`（:25）；`dragHandle`（:65）；`grip`、`ref`（:83） | 测试照 React 的语义写的 hook 的模型；pragmatic-dnd 的替身；最后一次提交的把手 |
| `core/components/labels/project-setting-label-group.test.tsx`（P11 新） | 6 | `label`、`name`、`disabled`（:20）；`label`、`name`、`isEditable`（:24） | `LabelItemBlock`、`ProjectSettingLabelItem` 的替身的 props |
| `core/components/project-states/create-update/create-update.test.tsx` | 3 | `data`、`onSubmit`（:25-26）；`forms`（:28） | 表单的替身记下的 props；`vi.hoisted` |
| `core/components/project-states/options/delete.test.tsx`（P11 新） | 1 | `children`（:26） | `Tooltip` 的替身 |
| `core/components/project/card.test.tsx` | 3 | `permissions`（:25）；`offered`（:66）、`linked`（:79） | 模块级的持有对象；`it.each` 的列 |
| `core/components/project/delete-project-modal.test.tsx` | 2 | `workspaceSlug`、`projectId`（:22） | `vi.hoisted` 的路由参数 |
| `core/components/project/logo-props.test.ts` | 2 | `picked`、`kept`（:14） | `it.each` 的列 |
| `core/components/project/member-list-item.test.tsx` | 1 | `of`（:31） | `vi.hoisted` |
| `core/components/project/member-list.test.tsx` | 2 | `permissions`（:21）；`offered`（:55） | 模块级的持有对象；`it.each` 的列 |
| `core/components/project/project-refusal.test.ts` | 3 | `when`、`error`、`shows`（:13） | `it.each` 的列 |
| `core/components/project/project-roles.test.tsx` | 8 | `who`、`workspaceRole`、`offered`（:123）；`who`、`caller`、`membership`、`choices`、`removable`（:135-139） | `it.each` 的列；另一张表的行的类型 |
| `core/components/project/project-settings-member-defaults.test.tsx` | 4 | `onChange`、`isDisabled`（:26）；`permissions`（:37）；`changes`（:146） | 选择的替身的 props；模块级的持有对象；`it.each` 的列 |
| `core/components/project/use-update-project-details.test.ts` | 1 | `available`（:143） | 扣住的回答的类型 |
| `core/components/projects/settings/useProjectColumns.test.tsx` | 8 | `workspaceRole`、`projectRole`（:25）；`columns`（:58）；`who`、`workspaceRole`、`projectRole`、`row`、`offers`（:81-85） | `vi.hoisted` 的调用者；记下渲染的列；表的行的类型 |
| `core/components/projects/use-archived-projects-fetch.test.ts` | 3 | `when`、`slug`、`workspaces`（:36） | `it.each` 的列 |
| `core/components/workspace/invitation-view.test.ts` | 3 | `when`、`inputs`、`view`（:28） | `it.each` 的列 |
| `core/components/workspace/invite-modal/refusal.test.ts` | 3 | `refusal`、`error`、`shown`（:17） | `it.each` 的列 |
| `core/components/workspace/members/invite-modal.test.tsx` | 1 | `children`（:30） | `Dialog.Title` 的替身 |
| `core/components/workspace/settings/workspace-details.test.tsx` | 8 | `onChange`（:21、:35）；`name`、`size`、`zone`（:49、:56） | 时区选择的替身和记下它的数组；测试的表单的值 |
| `core/components/workspace/use-create-workspace.test.ts` | 3 | `refused`、`error`、`shown`（:143） | `it.each` 的列 |
| `core/store/label.store.test.ts` | 3 | `change`、`send`、`refusal`（:182） | 被拒绝的修改的表 |
| `core/store/member/project/project-member.store.test.ts` | 3 | 同上（:207） | 同上 |
| `core/store/member/workspace/workspace-invitations.test.ts` | 3 | 同上（:171） | 同上 |
| `core/store/member/workspace/workspace-member.store.test.ts` | 3 | 同上（:178） | 同上 |
| `core/store/project/project.store.changes.test.ts` | 6 | `field`、`turn`、`bodies`（:249）；`change`、`send`、`refusal`（:286） | 开关的表；被拒绝的修改的表 |
| `core/store/state.store.changes.test.ts` | 2 | `list`、`load`（:55） | 列表的表 |
| `core/store/state.store.test.ts` | 3 | `change`、`send`、`refusal`（:228） | 被拒绝的修改的表 |
| `core/store/workspace/index.test.ts` | 3 | 同上（:245） | 同上 |
| 合计：30 个文件 | 123 | | |

**控制者的浏览器核对 C9、C10**（9.7，`$M3TMP/p11-browser-checks.md`，截图在 `$M3TMP/p11browser/`）：
- **准备**：
  - 构建：`83f30b36` 的 `git archive` 副本（`$M3TMP/p11browser/src`），`--frozen-lockfile` 安装，`make build`（`bin/nerve`，内嵌 web）。
  - 数据库：一个临时的 `postgres:18.6` 容器 `p11-browser-db`（`127.0.0.1:55511`），核对时起、之后删除，没有碰别的容器。nerve（`NERVE_ENV=dev`）在 `127.0.0.1:8097` 上，之后停掉。Playwright MCP 的输出移出了 worktree。
  - 数据经接口建（`p11browser/seed.mjs`）：acme（alice 管理员、bob 成员、carol 访客、erin 管理员）；公开的 Web（alice、erin 是项目管理员，bob 成员，carol 访客；nerve 只让工作区管理员以管理员加入，给 erin 角色 15 时答 422 `members[2].role not_allowed`）；标签 Bug、Feature、Area（下有 UI、API）、Docs；状态是 nerve 的默认，Backlog 是默认，每组一个。
  - Playwright MCP，Chromium，1280×900。
- **C9 状态和标签：通过。**
  - 状态（alice）：默认的 Backlog 显示"Default"、不可用；每个删除都不可用（每组的状态是默认的或组中唯一的）。started 组的"Add"打开表单（Color、Name、说明、Create、Cancel），"Review"发 `POST /api/v0/projects/{id}/states`，行出现在 started 下；之后 In Progress、Review 的删除可用。Review 拖到 In Progress 的上方发一个 `PATCH /api/v0/states/{review}`，顺序变为 Backlog、Todo、Review、In Progress、Done、Cancelled。Review 的"Mark as default"发 `POST …/mark-default`，之后 Review 显示"Default"、它的删除不可用，Backlog 显示"Mark as default"、删除仍不可用（组中唯一的）。重新载入之后顺序和默认照 nerve 的。键盘：从 In Progress 的 Edit 按 Tab 到它的 Delete，显示浏览器的焦点轮廓（`outline: auto`，`c9-delete-focus.png`），行里有焦点时控件显示。
  - 标签（alice）：两层的拖动：Bug 放到 Area 的中间 `PATCH` 200，Bug 在 Area 之下；Docs 放到 UI（Area 之下的）上 200，Docs 进 Area、在 UI 旁边，仍是两层；有子标签的 Area 放到 Feature 上 200，Area 留在顶层、移到 Feature 之前。nerve 拒绝第三层（P7 的情形）：另一个会话（`p11browser/elsewhere.mjs`）把 Feature 放到 Area 之下，页面仍把 Feature 显示在顶层；把 UI、再把 API 放到 Feature 的中间，`PATCH` 答 422 `parent_id not_allowed`，提示"Error! A label can go only under a label at the top of its project, and a label with labels under it stays at the top: labels have two levels."（`c9-two-levels.png`）；控制台只有浏览器自己的"Failed to load resource … 422"和 Plane 的 Canvas2D 警告。
  - 谁能改：项目成员 bob 在状态页上没有行的控件、没有把手，每组的"Add"不可用；状态的行带 pragmatic-dnd 的 `draggable`，可把 In Progress 拖到 Review 上不发 `/api/v0/states` 的请求、顺序不变（`canDrag` 的门）；标签页没有控件、没有把手、没有可拖的，组的展开按钮（"Area"，展开着）是唯一的按钮。项目访客 carol 在状态、标签的设置上看到"Oops! You are not authorized to view this page"（`c9-guest-states.png`）；挂载时项目的包装层取项目的标签、成员和状态（nerve 让访客读它们），页面自己的内容不渲染，没有发往别的源的请求。
- **C10 访客打开别的成员的个人主页（决策点 3 的 A）：通过。**
  - carol 打开 `/acme/profile/{alice}` 和她自己的 `/acme/profile/{carol}`，都转到 `…/assigned`。卡片："Alice Tester (alice)"或"Carol Tester (carol)"和"Joined on"。
  - 页面的 HTML 里没有任何账户的邮箱（alice、carol、bob、erin 的）。工作项的位置说"You do not have the permission to access this page."，没有 Assigned、Created、Subscribed 的链接。
  - 接口请求恰好是 `auth/refresh`、`instance`、`me`、`me/profile`、`workspaces`、`workspaces/acme/members`、`me/workspaces/acme/preferences`、`workspaces/acme/projects?archived=false`、`workspaces/acme/states`：没有工作项的请求，没有发往别的源的请求（`c10-alice.png`、`c10-carol.png`）。
- 核对中看到的、不是 P11 缺陷的三处：组标题以小写的组键作为文字和名字（第 6 节收尾）；被拒绝的移动不刷新列表（第 7 节）；访客的个人主页的页头仍有布局、筛选和 Display 的控件（spec 第 3 节第 6 条，已交给 M4；第 7 节）。

**变异的记录**（spec 附录 A.2，修复轮照裁定 F-12 改为与代码一致）：
- **98 行**：plan 的 72 个和任务评审加的 26 行（`T2.14`–`T2.16`、`T3.11`–`T3.13`、`T4.13`–`T4.17` 和 `T4.14b`、`T5.18`–`T5.24`、`T7.10`–`T7.12`，以及 Task 6 的探查 `P6.1`–`P6.4`）。没有存活的。
  - plan 的 72 个在最终原型上都被发现：62 个在自己的 Task 的树上，10 个由之后的 Task 的页面版本（`T1.2`、`T1.3`、`T2.7`、`T2.8`、`T2.9`、`T2.11`、`T2.12` 由 Task 3 的 P6，`T4.8`–`T4.10` 由 Task 5 的 P7）。
  - 评审加的 26 行各在它的轮次的树上被发现（`$M3TMP/p11mut/t3`、`t5`、`t6`、`t7`）；其中 10 行由之后的 Task 的检查才发现（`T2.14`–`T2.16` 由 Task 3 的 P6，`T3.13`、`T4.13`–`T4.17`、`T4.14b` 由 Task 5 的检查）。
- **按类别**：会话 17、权限的门 18、键盘 22、请求在路上和被拒绝时 15、错误的读法 6、W12 6、清理 5、getter 4、拖动 3、请求体 1、接线 1，合计 98。各层的列：静态 15、vitest 40、端到端 58（一个变异可以算在几层）。按发现它的层：只有端到端的 47、只有 vitest 的 25、只有静态的 11、vitest 和端到端的 11、静态和 vitest 的 4，合计 98。按 Task：4、16、13、18、24、6、12、2、3（Task 10 只改文档），合计 98。本记录逐格加过一遍，与 spec 的合计相同；复审也核对了类别表的 98。
- **修复轮的重跑**（`$M3TMP/p11mut/fw`，`mutants_fw.py`，`run.log`、`results.json`）：检查被 F-1–F-5 移动或改写的 71 行（plan 的、评审加的，和 Task 6、Task 7 的探查），加上 F-2、F-3、F-7 改写的 vitest，在修复之后的树上再跑：87 层中 85 层发现它。存活的两层是 `P7.1`、`P7.2` 在 `create-token-modal.test.tsx` 的 vitest 一层，与 Task 7 的轮次相同，它们由 A11 发现。`T4.8`、`T4.9`、`T5.11`、`T4.14` 以修复之后的文字为第二种版本（F-2 改了那几行的缩进）。`P6.3`（卡片显示看的人的邮箱）由 `expectNoAddress` 发现。
- **整分支评审在 `0c98207d` 上的抽查**：plan 的 72 个变异要改的文字在 HEAD 上都恰好出现一次；有 vitest、oxlint、关键词或 editor 一层的 48 个，52 层都被发现；端到端只发现的 24 个中抽 6 个（`T1.2`、`T1.3`、`T2.11`、`T4.10`、`T5.17`、`T7.9`），都被发现。它自己的两个探查存活：R1（删去根组件的取数）、R2（删去 `groupedStates` 一串），成为它的 M3、M1，修复轮删除了那些代码（不是变异：改动是删除，由 `tsc` 和测试核对）。
- **W18**：P11 新加或改写的 122 个测试都至少被一个变异或探查让它失败（plan 的变异 56 个，32 个探查其余 66 个）；任务评审之后加的没有颜色的改名一行由 `T4.16` 发现。
- **只由评审看住的**：删除确认框的 Tab 循环（`AlertModalCore` 的 `tabIndex={1}` 属于收尾的 `ModalCore` 重写，第 3 节 Q3）。

**持续集成**：分支上每次推送都跑 server、web、e2e 三个任务。下面的运行都通过，除了第一次的 e2e 被取消：
- Task 1（`8313aa18`）：run 38155247991：server、web 通过，e2e 被 `716b8351` 的推送取消（工作流取消同一分支上还在跑的运行）；
- Task 1 的修正轮（`716b8351`）：run 38155787149；
- Task 2（`45e4e1f7`）：run 38156325297；
- Task 3（`9425050d`）：run 38157685606；
- Task 3 的修正轮（`39222646`）：run 38159185271；
- Task 4（`812e41b8`）：run 38159882450；
- Task 5（`4e7285e0`）：run 38162163106；
- Task 5 的修正轮（`5f178ceb`、`9ebc375e`、`baae317a`，头 `baae317a`）：run 38164428211；
- Task 6（`fc9d55bf`）：run 38165393989；
- Task 7 和它评审之前的一轮（`b691d42f`、`746c1300`）：run 38170129256；
- Task 7 的修正轮（`d870fe7d`）：run 38171239388；
- Task 8（`cd5d1bd1`）：run 38172214747；
- Task 9（`9d4709ab`）：run 38173190551；
- Task 10（`49749f5d`）：run 38174450923；
- Task 10 的修正轮（`0c98207d`）：run 38175555154；
- 修复轮（三个提交，头 `83f30b36`）：run 38181207767。

本地的 `make e2e` 每个 Task 都跑（裁定 S2：Docker Desktop 起着），持续集成是第二道。本记录的提交推送之后，分支上再跑一次；合并之后 main 的持续集成在合并时记录。

## 3. 执行中的决定

这一节列控制者在本阶段做的裁定，每条写明理由和裁定错了的代价。完整的记录在执行台账中。

**执行前**
- 架构子任务的 spec 第 3 节，控制者的裁定 Q1–Q8（`$M3TMP/p11-architect-rulings.md`，2026-10-11）；第 3 节其余各条（第 6、7、11–17 条）照架构子任务写的接受：
  - **Q1**（第 1 条，10 个 Task，接受）：在约 16 个、约 1,500 行之内，P7–P10 的先例。代价：太大的 Task 要由它的修正轮拆。（修复轮由提交的代码重新写出块之后，Task 5、7 超过约 1,500 行，第 5 节。）
  - **Q2**（第 2 条，`AlertModalCore` 在 `isSubmitting` 时关不掉，改在组件里，接受）：这是总体设计 7.7 已有规则（P9 的 F-1）的根因修复，不是新规则；P10 修共用组件的先例（`ModalCore` 的组字 Escape、`CustomSelect` 的 `tabIndex`、`ContextMenu` 的 `inert`）。条件：逐个核对其余 11 个调用方在每条被拒绝的路上（store 抛出、提示的路、卸载、提前返回）都把 `isSubmitting` 复位，写进 spec A.4，由预检核对；一个被拒绝之后再也关不掉的弹窗比今天更坏，不复位的在调用方修、加 W17；收尾的 `ModalCore` 重写保持这一点，写进第 5 节收尾。代价：M2、M4–M8 的 11 个确认框随之在请求路上时关不掉（W17），它们的页面没有端到端看它们。
  - **Q3**（第 3 条，`tabIndex={1}` 和它的警告留到收尾的重写，接受，负责人 2026-10-10 的裁定）：收尾的一行写明重写之后 P6、P7 核对删除确认框的 Tab 循环。代价：到那时之前这个 Tab 循环不核对，`alert-modal.tsx` 是手改的文件中唯一留着警告的。
  - **Q4**（第 4 条，放到第三层的 422 说"标签只有两层"，接受）：预检的 L3 之后改为 nerve 的四种 `not_allowed`（父标签不存在或在别的项目、自己做自己的父标签、父标签已在别的标签之下、下面有标签的标签放到别的标签之下），用更完整的文案；移出组得不到 `parent_id` 的 422，它的拒绝经 `useRefusalToast`。理由：通用的"有字段不合要求"在没有 `parent_id` 字段的页面上没有用。代价：页面的话不是 nerve 的原话，两种语言多一条文案。
  - **Q5**（第 5 条，访客在界面上没有离开工作区的路）：产品决定，交给负责人。**负责人的答复（2026-10-11）："菜单加入口，收尾做"**：侧边栏的工作区菜单加"离开工作区"，每个角色都有；唯一的管理员离开时 nerve 拒绝，页面说明原因；是 M3 收尾的一个小任务，P11 不改。修订把它记进 spec 第 3 节第 5 条、第 5 节收尾、第 7 节，和 M3 设计第 12 节收尾（`1fa8f921`）。代价：到收尾之前访客经 PAT 离开。
  - **Q6**（第 8 条，Task 9 的文件其余 14 条警告留着，接受）：Task 9 是照 oxlint 的输出写出的一个机械步骤，正是 7.9 的 R3（控制者 2026-10-07 的裁定）；第 5 节把它们交给 M4、M6、M7、M8。代价：这 5 个文件的 14 条留到它们的 M。
  - **Q7**（第 9 条，上限调到新的条数，接受）：7.9 写的是每次合并时调到新的条数：web 348 → 303、editor 65 → 64，其余条数不变、上限不变。代价：没有。
  - **Q8**（第 10 条，标签的把手以状态登记：取 (b)）：加一个 vitest，P8b 的 P7 要的修法才有会失败的检查。条件：不加依赖和测试环境，用仓库已有的 vitest 写法；回到 ref 的 `T5.ref` 由 vitest 发现，进 Task 5 的变异表；做不到就退回 (a)，在第 3 节第 10 条写明为什么。代价：一个 vitest（架构子任务估计约 40 行，提交的是 137 行），它自己照 React 的语义写 hook（spec 第 6 节的风险）。
- 预检（高 1、中 1、低 9）和修订（`1fa8f921`、`18b5ce41`，第 1 节）：控制者的裁定（`$M3TMP/p11-amend-brief.md`）：
  - **A-H1**（照预检写的接受）：Task 8 留着 `WorkspaceSettingsSidebarRoot` 的 `className`，这一行进 A.9 留下的行，写明它的读者；凡引用的地方改为删去 26 行、产品代码 80 → 54。根因是 `deadsym.mjs` 看不到经组件类型的变量的读，所以其余 25 处删除照同一形状逐个扫过（组件作为值传入、`component=`、`as=`、渲染 prop、组件的表），写进 A.5 的 C-b：没有第二处（`sweep_h1.py`）。代价：没有（这个文件 0 条警告，上限不受影响）。
  - **A-M1**（修法 (a)）：Task 7 也改 security 页的两个 API 令牌的弹窗：各经 `followInSession`（新建在 `done` 里显示、下载密钥，撤销在 `done` 里提示、关上；`failed` 照 M2 显示），各两行 `lateSettlingsAnswering` 的 vitest、2.12 一行、一个变异（`T7.7`、`T7.8`）；照 7.7 的弹窗规则核对它们（在途时关不掉、忙到跟进做完、连按两下不发第二次），违反的在同一个 Task 修，加测试和 W17。理由：同一个页面上同一类缺陷（P9 的裁定 P8 交给 P11 的），生成的密钥是这一页最敏感的跟进。代价：约 40 行和两个小测试（预检的估计）；核对找到新建的取消在途时可按，改为禁用（`T7.9`），M2 的 A11 第二个故事随之改为核对取消不可用、经浏览器的后退离开页面（spec 第 3 节第 18 条、第 6 节的风险）。
  - **A-L1**、**A-L2**（接受）：历史的句子（标签的删除确认框原来就只在 nerve 删除之后关，原来的缺陷是在途时关得掉、跟进不经会话的核对）和 `lateSettlings` 的说法照预检的文字。代价：没有。
  - **A-L3**（接受）：四种 `not_allowed`（也改在裁定的文件里）；预检给的更完整的文案，两种语言；`removeFromGroup` 的拒绝直接经 `useRefusalToast`，没有 `parent_id` 的分支，删去走不到的测试行（hook 的测试 27 → 26）。理由：没有死的分支、死的测试。代价：没有。
  - **A-L4**（接受）：A.6 的分法改为"交回调用方 6、没有处理 8"；迭代、模块的复制链接交给 M6，默认视图的交给 M7。不改代码。（Task 9 的评审又改正一次，裁定 T9-a。）
  - **A-L5**（接受）：页面 vitest 的替身只对页面的项目回答，读了别的 id 的页面显示加载、测试失败；状态的替身有真的组；加 `T3.10`。代价：几行。
  - **A-L6**（先加强）：行内表单的测试像状态表单的那样断言名称下的文案；SSR 看不到时才改标题、记作 P7（`T4.11`）守着。结果是加强做到了：测试记下 react-hook-form 的 `setError`，`T4.11` 在 vitest 一层也被发现。
  - **A-L7**（修法 (a)，在根上）：pragmatic-dnd 的拖动只认指针，把手不能是没有名字、看不见的 Tab 停留点。`@nerve/ui` 的 `DragHandle` 的 4 个调用方都只把它当指针的把手，所以改组件：`tabIndex={-1}`、`aria-hidden`，加一行 W17 写它在 P11 之外的调用方；标签组的展开按钮以标签的名字为名（与 Task 2 的状态组标题一致）；P7 按名字和 `expanded` 找它，并核对把手不在 Tab 顺序里，`T5.16`、`T5.17`。代价：一个共用组件的两个属性。（侧边栏的三个调用方中 `DragHandle` 只是外层按钮里的图标，外层的按钮照旧是停留点，最终评审的 M5 才看出，第 4 节。）
  - **A-L8**（更好的修法）：表单只由 `labelToUpdate` 决定按钮的文字和发出的修改，删去它的 `isUpdating` prop；列表和组照旧以自己的 `isUpdating` 排版；看得到的不同写 W17。代价：没有。
  - **A-L9**（接受）：Task 9 在根目录 `.oxlintrc.json` 把 `no-unreachable` 设为 error（今天 0 处，上限不动），加 `T9.3`。代价：没有。
  - **Q2 的核对和加固**：其余 11 个调用方的核对表写进 A.4；两个工作项的删除确认框的 `onSubmit` 改为必传，条件是两个文件只因这一处改动就结束于 0 条警告（7.9），否则把这条路写进第 5 节 M4。修订核对：它们在 Task 9 之后都是 0 条（Task 1 之后各留一条 `always-return`，裁定 T1-a）。
  - **Q8 (b)**：Task 5 加 `label-drag-n-drop-HOC.test.tsx`（预检的原型，删短），文件头写明它自己照 React 的语义写 hook；`T5.14`（即 `T5.ref`）、`T5.15` 进 Task 5 的变异表。修订中发现删短的测试每次渲染给新的 `onDrop`，`T5.14` 因此存活（W18 的运行看出），改为一个模块级的 `onDrop`（列表把它包进 `useCallback` 时的样子）。
  - 设计的提交（`1fa8f921`）在 spec 和 plan 的提交之前，照之前的修订轮的先例。
- 开始执行时（台账的冲突扫描）：
  - **S1**（照 plan 执行）：预检的跨 Task 表逐对核对过，每条发现都在修订中落实，由再一次的复现证明。代价：错了是一轮 Task 级的修正。
  - **S2**：Docker Desktop 起着；每个 Task 在本地跑 `make e2e`，数目照全局约束；Docker 不再回答时实现者报告，控制者依靠分支的持续集成。代价：每轮慢一些。
  - **S3**：每个 Task 的变异在新目录 `$M3TMP/p11mut/tN` 跑（P10 的 T1-a 的教训）。代价：没有。

**执行中**
- **T1-a … T1-c（表单的拒绝、确认框）**：
  - **T1-a**：实现者报告两个工作项的删除确认框在 Task 1 结束时各留一条 `always-return`，全局约束的例外没有写它们。接受：plan 让 Task 9 在全仓一次的机械步骤里清掉它，两个文件从 Task 9 起都是 0 条；把 `return;` 挪进 Task 1 会把那一步拆开。代价：合并时没有；最终评审要早一点时改一行。
  - 评审：Spec ✅，Quality 通过，C0/I0/M2（6 个文件与快照逐字节相同；其余 11 个调用方逐个读过，不是抽查：每个在被拒绝的路上都复位忙碌，spec A.4 的说法成立；接口合 Task 2、4 用）。m1 见第 1 节；m2 `form-refusal.ts:30` 的注释难懂。
  - **T1-b**：m1、m2 现在一轮小的修正：每个读错误的地方都经 `errorMessageKey`（评审的标准第 2 项），之后没有块改 `form-refusal.ts`。代价：一个小提交；之后的树在这个文件上与快照不同。修正轮 `716b8351`（注释写明为什么留着类型标注：没有它是 TS2322），范围复审（sonnet）通过。留给修复轮的 Minor：`form-refusal.ts:27` 和 `errorMessageKey` 各自从 `ApiError` 取码（搁置项 B 的 T1）。
  - **T1-c**：评审指出 spec 第 3 节第 2 条要 W17、A.4 却说"没有 W17 的行"：带进 Task 10，前端改动清单的确认框一行写明改动到达全部 13 个调用方，A.4 的一句与第 3 节第 2 条一致。代价：没有。
- **T2-a … T2-d（状态的修改）**：
  - 实现者报告"Mark as default"忙到跟进做完没有检查（探查 P2.1 存活）；规则失误：一次调用跑了两个 git 命令（只读和 `git add -N`，无害）。
  - 评审：Spec ✅（19 个文件与快照相同），Quality 通过，C0/I1/M2。移动照 PF-M2，store 在轮到它时算；`followInSession` 不拒绝，每个忙碌都复位；请求体封闭；没有孤儿。I1 见第 1 节；m1 P6 从不收起组标题；m2 `fake-controls.ts:9` 的说明说弹窗渲染它的子元素，新的 `AlertModalCore` 替身渲染 null。
  - **T2-a**（I1）：是真的缺口，但不是 Task 2 的代码缺陷（SSR 的 vitest 看不到点击之后的状态，A.2 中 `T2.7`、`T2.9`、`T2.11` 的理由）：带进 Task 3 的 P6，第一次设默认被扣住，`T2.14`（P2.1）、`T2.15`（去掉 `|| isLoading`），标"（Task 3 起）"。代价：故事多几行。
  - **T2-b**（m1）：同样带进 Task 3 的 P6，`T2.16`。**T2-c**（m2）：一句说明，带进 Task 3 的提交（`fake-controls.ts` 只由 Task 2 的块改），不另开修正轮。代价：没有。
  - **T2-d**：plan 的变异表和 spec A.2 的数目在最终的修复轮一次加上评审加的行，不逐个 Task 改（P10 的 P1、P4 的先例；搁置项 P1）。Task 3 当轮改正了数目（扣住的 9 → 11、键盘 11 → 12、合计 72 → 75）。代价：到修复轮之前文档落后于代码。
- **Task 3（状态设置页；没有编号的裁定）**：
  - 实现者带上 T2-a–T2-c（P6 :196-203 和 :237 的第二次设默认、:106-115 的收起、`fake-controls.ts:9`），`T2.14`–`T2.16` 被发现；关注 1 是删除按钮看不出焦点，关注 2 是 T2-d 的数目。
  - 评审：Spec ✅，Quality 要修，C0/I2/M1。门与 nerve 的规则相同（`rules.go:112-119`、`decide.go:79-82`、`permissions.store.ts:66`），`states-labels.test` 用 store 自己的权限；S2 的行、关键词的样例、`dragOnto`/`tabCycle` 各一份；T2-a、T2-b 成立，T2-c 为真。I1、I2 见第 1 节；M1 P6 的 `name: "Default"` 没有 `exact`，也匹配"Mark(ing) as default"。
  - 控制者把 I1、I2、M1 交给修正轮，`T3.11`、`T3.12` 并进 T2-d 的数目（P1）。修正轮（`39222646`）：删除按钮去掉 `focus:outline-none`，P6 核对 Edit、Delete 的轮廓和 Shift+Tab；新的 `sentWithin`（`e2e/fixtures/settings-pages.ts`），P6 中成员经 `dragOnto` 拖动，不发 `PATCH`、顺序不变（成员重新 `login`：沿用他注册的令牌会让之后的浏览器登出）；`exact: true` 两处。范围复审（sonnet）通过。留下的 Minor（搁置项 B 的 T3）：`sentWithin` 的计时器在请求先到时不清除；成员的页面在拖动时没有 `watchPage`。实现者另指出标签的行可能有同样的两个缺口，带进 Task 5。
- **T4-a … T4-f（标签的修改）**：
  - 实现者报告 C1–C4（第 1 节的颜色按钮、确认框被拒绝时留着、没有颜色的改名、移出组），交给评审判断它们的去处。
  - 评审：Spec ✅（11 个文件与快照逐字节相同），Quality 通过，C0/I2/M3，都是测试的缺口或原有的，带进 Task 5。422 的映射按 nerve 指名的字段（`parent_id`，只有 `CheckParent` 指名它；四种 `not_allowed` 与 `label.go:104-118` 相同）；PF-M2（`label.store.ts:172-191` 在轮到它时算）；忙碌在每条路上复位；清理完整。spec A.4"被拒绝时弹窗留着"一句把标签的引成 `T4.10`（那是扣住的确认框）。
  - **T4-a**（I1）：带进 Task 5：两个颜色按钮以一个新的键为名，标签的那个恢复轮廓，P6、P7 核对，两个键盘的变异 `T4.13`、`T4.14`（Task 5 起），Task 10 写 W17。理由：这些文件之后没有块改。代价：一个键，两个表单和两个故事的几行。
  - **T4-b**（I2）：带进 Task 5：P7 中一次被拒绝的删除，照 P6 的 :231-248，`T4.15`（即 P4.2）。代价：P7 几行。
  - **T4-c**（M1）：与 I1 在同一个文件里一起修（根上的修法：修改从 nerve 存的颜色开始），一行 vitest 和变异 `T4.16`；色块的样子变了就写 W17。代价：一行和一行测试。
  - **T4-d**（M2）：带进 Task 5 的 P7，`T4.17`（即 P4.3）。代价：P7 几行。
  - **T4-e**（M3，移动被以别的字段的 422 拒绝没有测试行）：推迟，给最终评审；nerve 在移动上答不出别的 422，变异今天等价。代价：今天没有。
  - **T4-f**：A.4 那一句和数目在修复轮加上 `T4.13`–`T4.17`（T2-d、P1）。代价：到那时之前文档落后。
- **T5-a … T5-g（标签设置页、共用组件的焦点）**：
  - 实现者带上 T4-a–T4-d 和 Task 3 的提醒（`T4.13`、`T4.14`、`T4.14b`、`T4.15`、`T4.16`、`T4.17`、`T5.18` 都被发现）。关注 1：标签一行的"Options"看不出焦点，它在根上去掉了 `CustomMenu` 省略号按钮的 `outline-none`（窄的修法是给标签一行一个 `buttonClassName`）。关注 2（没有修）：propel 的 `Button` 的 `focus-visible:outline-none`，这一页和每一页上的"Add label"、Add、Cancel、弹窗的按钮都看不出键盘的焦点。
  - **T5-a**：省略号按钮保留根上的修法：与 `DragHandle`（A-L7）、P10 的 T2-b 和 F-13–F-15 是同一条规则和先例；`:focus-visible` 只对键盘显示，用指针的人看不到不同；Task 10 写 W17。代价：错了就改回一个类，或每个调用方一个类。
  - **T5-b**：propel 的 `Button` 在 Task 5 的修正轮里在根上去掉 `focus-visible:outline-none`（P11 自己的页面经它违反"到达时看得见"；共用组件的一行，不是架构的改动；P10 修共用组件的先例），P7 核对"Add label"一类的 `Button` 在 Tab 时有轮廓，加一个变异；Task 10 写 W17（每个 `Button` 在键盘取得焦点时显示浏览器的轮廓）。代价：错了就改回一行，键盘用户又看不到。
  - 评审：Spec ✅（plan 的 17 个文件与快照相同，带进的文件照写明的不同），Quality 要修，C0/I1/M2。权限与 nerve 相同（`rules.go:121-125`、`decide.go:79-81`），在 store 自己的权限上；回调 ref 成立，HOC 的测试让 `T5.14`、`T5.15` 失败；`DragHandle` 的调用方都只用指针；手改的 18 个文件 0 条警告；每个带进的检查失败在它的那一行。I1 见第 1 节；M1 P7 的标题过时；M2 `CustomMenu` 的改动到达 18 个文件 18 处（不是约 22），成员页的 `member-columns.tsx:72` 自己的 `opacity-0` 让轮廓照旧看不见（P10 的成员页原有的）。另外看不出焦点的只有 propel 的 `Button`（T5-b）和 react-color 的 `TwitterPicker` 的十六进制输入框（第三方的内联样式）。
  - **T5-c**：I1、M1 与 T5-b 一起进修正轮；M2 的数进 Task 10 的 W17（`member-columns` 的轮廓被它自己的透明藏住，原有的）。代价：没有。
  - **T5-d**：`TwitterPicker` 的输入框记进 M3 收尾的键盘清单，不修。代价：它的焦点只显示为光标。
  - 修正轮 (a)（`5f178ceb`）：`Button` 去掉 `focus-visible:outline-none`，P7 核对表单的 Add 的轮廓（`T5.20`）；两个 store 测试各一行（`T5.19`、`T3.13`，在 HEAD 原来的测试上存活）；P7 的标题照实。实现者又报告：实心的蓝色按钮上浏览器的轮廓（`rgb(0,95,204)`，偏移 0）几乎与按钮同色，只像宽了 1 像素的边；propel 的 `IconButton`（约 32 个 web 文件）也有 `focus-visible:outline-none`。
  - **T5-e**：两者在同一轮里做完，再交给复审：`Button`、`IconButton` 的基础类加 `focus-visible:outline-offset-2`（轮廓离开填充，每种样式上都看得见），`IconButton` 也去掉 `focus-visible:outline-none`；P11 的页面上 Tab 能到一个 `IconButton` 就核对它，不要只为类名写 propel 或 web 的 vitest；它的到达范围记给 Task 10 的 W17。代价：错了是轮廓外 2 像素的空隙，改回一个类。修正轮 (b)（`9ebc375e`）：实心的 Add 的轮廓离开 2 像素、中间有白边；侧边栏的返回箭头（P11 的页面上 Tab 能到的 `IconButton`）显示轮廓；P7 核对返回箭头的轮廓和偏移、Add 的偏移（`T5.20`–`T5.23`）。实现者再报告：37 处 `IconButton` 只有 7 处有 `aria-label`，返回箭头没有名字；四处调用方藏住轮廓或 Tab 到不了。
  - **T5-f**：返回箭头在这一轮以 `common.go_back` 为名（P11 的页面上的控件，第八类缺陷），P7 按名字找它，加一个变异。代价：一行。修正轮 (c)（`baae317a`），`T5.24`。三个提交的范围复审（sonnet）通过。
  - **T5-g**：其余没有名字的 `IconButton` 和四处藏住轮廓或 Tab 到不了的调用方进 M3 收尾的键盘清单（带它们的 M），评审记录第 6 节照录。代价：到收尾或它们的 M 之前照旧。（Task 10 的修正轮按 HEAD 改正为 27 处、五处。）
  - 留下的 Minor（搁置项 B 的 T5）：P7 按名字找"Go back"，有第二个时会有歧义；从 General 按 Shift+Tab 假定中间没有可聚焦的。plan 的文字漂移（P1）：Task 6 的接口一条仍说第 3 节第 5 条"待裁定"。
- **T6-a … T6-d（访客看到的个人主页）**：
  - 实现者报告：plan 的 P9 块把共用的 `named`（`e2e/fixtures/auth.ts`）抄成一个局部的闭包；它改为扩展共用的那个，让它收要设的名字，P1 的两处调用传 `{ display_name }`（函数体相同），所以多了 plan 之外的两个文件，P9 是 48 行。`T6.1`、`T6.2` 都停在 P9 的 :33。
  - **T6-a**：接受：扩展共用的 fixture 而不是复制，是负责人的不重复的标准；plan 的块落后（P1）。代价：没有。**T6-b**：探查 `P6.1`–`P6.4` 在修复轮进变异的记录（T2-d）。代价：没有。
  - 评审（opus 的评审子任务因 ECONNRESET 中断，接着跑完）：Spec ✅，Quality 通过，C0/I0/M1。web 的代码不变（只改 4 个 e2e 文件）；隐私在页面此刻的 DOM 上（`page.content()`，两个个人主页、两个地址）和每个 `/api/` 请求上核对；`named` 的函数体不变；S2 与快照相同。m1：`p9-guest-profile.spec.ts:36-40` 重复 `w11-guest-bounds.spec.ts:114-115` 的邮箱检查（来自 plan 的块）。另指出 spec 第 2 节的文件数和 plan 的文件结构表在 T6-a 之后少两个。
  - **T6-c**：m1 交给最终的修复轮（负责人的不重复的标准；只是质量，不要变异）。代价：之后一个共用的函数。**T6-d**：文件数和表在修复轮加上 T6-a 的两个文件（T2-d、P1）。代价：没有。
- **T7-a … T7-e（M2 的账户页面）**：
  - 实现者：web 106 个文件、965 个，e2e 110 个，A 系列和 W1 各 3 次两轮都通过；`T7.1`–`T7.9` 都被发现。关注 1 见第 1 节（三条弹窗规则在浏览器中成立、没有检查）；关注 2：plan 之外共用的 `core/lib/fake-form.ts` 代替三份 react-hook-form 的替身（在非空断言的范围里），A11 的令牌名"cancelled"改为"abandoned"；关注 3：规则失误。
  - **T7-a**：关注 1 在评审之前补上（P10 的评审之前一轮的先例）：A11 加探查的那几行，`T7.10`（P7.6）、`T7.11`（P7.7）、`T7.12`（P7.9）。代价：A11 几行。评审之前的一轮（`746c1300`）：A11 沿用扣住的新建，撤销经 `sentHeld` 扣住；新的共用 fixture `closedByBackdropClick`（`closedByEscape` 的孪生）；探查的故事删去、没有提交。规则失误：一次只读的 `git -C … status`。
  - **T7-b**：关注 2 接受（不重复的标准；名字照实）；文件数和表在修复轮（T2-d、P1）。代价：没有。**T7-c**：关注 3 记下，之后每次分派重申规则（`cd` 不与任何命令组合，每次调用一个 git 命令）。代价：没有。
  - 评审：Spec ✅（11 个产品和 plan 的文件与快照相同，不同处已裁定），Quality 通过，C0/I0/M2。密钥只在弹窗自己的状态里（每次打开、关上之后 350 ms 清空）和一个短暂的闭包里；store 存的是去掉密钥的 `ApiToken`；`csvDownload` 释放它的地址；`T7.7` 的 `lateSettlingsAnswering` 两行在任何会话之外的 `done`、`failed` 上失败；A11 记下整个文档的事件和下载的数，直到故事结束。m1：`a11-api-tokens.spec.ts:181-183` 的说明不再为真：`746c1300` 的三个一秒的"不出现"之后，后退在 tokens 的 `GET` 之后两秒多才到（SWR 的默认 `dedupingInterval`），security 页的列表重取，在放开之前已显示"abandoned"，:188 的等待不再证明后续已运行（:202 同样）；泄漏仍由 :222-224 抓住，可 `P7.3` 在第二个故事的发现大概没了。m2：`e2e/fixtures/settings-pages.ts` 418 行。
  - **T7-d**：m1 现在修（一个安全故事中不真的说明和变弱的同步点）。代价：几行。修正轮（`d870fe7d`）：新的 `e2e/fixtures/downloads.ts` 的 `downloadedWithin(page, act)`（`act` 之前就听，之后等一秒），两个迟到的回答都经它；`P7.1`–`P7.3`、`T7.7`–`T7.12` 重跑，第二个故事果然不再发现 `P7.3`（证实 m1）。范围复审通过。留下的 Minor（B 的 T7）：一秒是界限，不是证明。
  - **T7-e**：m2 与 T6-c 一起交给修复轮（两者都改 e2e 的 fixture：`dialogs.ts`、`expectNoAddress`）。代价：到那时之前文件是 418 行。
- **T8-a … T8-d（死成员和死 prop）**：
  - 实现者：每个块照原样应用，14 个文件与快照相同；web 337 等于新的上限；每一行的"死"在当前的树上重新核对（没有组件作为值传入、没有展开、没有字符串或 `keyof` 或测试的读；清扫找得到 A-H1 那样的形状）。另报告：项目下拉框的根照旧收它在 M4、M6 的调用方给的 `tabIndex`，浏览器在那里不理会 `role="presentation"`；`deadsym` 数出 981 个成员（spec 写 980），多的是 Task 7 的 `fake-form.ts` 的 `submit`。
  - 评审：Spec ✅，Quality 通过，C0/I0/M3。在 HEAD^ 上用 git grep 重新核对每一处"死"；常量等于原来的默认值；改名是局部的；`key={group.state}` 唯一。m1：`base.tsx:197` 的 `role="presentation"` 无害，可它的前提在桌面上不成立：三个调用方都给根正的 `tabIndex`，根是没有名字、按 Enter 打开的 Tab 停留点（M4、M6 原有的缺陷；`issues/select/base.tsx:156-159` 同样的写法）。m2：每个调用方都传、没有读者的 prop（`deadsym` 看不到）：`dashed-circle.tsx:11` 的 `radius`、`dropdowns/member/avatar.tsx:17` 的 `showTooltip`。m3：`fake-form.ts:17` 的 `submit` 不是死的，可以导出 `KeptForm` 给三个 hoisted 对象做类型（去掉三份重述的形状，数目回到 980）。
  - **T8-a**：m1：Task 10 的 W17 写真的理由（根可以聚焦的地方不理会这个角色）；根上的修法进 M3 收尾的键盘清单或 M4 的交接。代价：原有的缺陷留到那时。
  - **T8-b**：m2 的 `radius`（P11 自己的 Task 8 改到的文件，只在 propel）和 `base.tsx:141` 的说明在最终的修复轮做。代价：没有。
  - **T8-c**：m2 的 `showTooltip`（M4、M6、M7 的 9 个文件 10 处；7.9 的 0 条警告的规则会连到它们）交给收尾或 M4 的清单。代价：到那时一个死的必传 prop。
  - **T8-d**：m3 的 `KeptForm` 在修复轮做（不手写重述，负责人的标准）。代价：没有。
- **T9-a（`then()` 的回调）**：
  - 实现者：照当前的树从 oxlint 重新列出的 35 处与 plan 的相同（web 34、editor 1；同样的文件和行），38 个块照原样应用，每个 `return;` 在最后；保留的警告与第 3 节第 8 条的 14 条相同；两个工作项的删除确认框 0 条（T1-a）。
  - 评审：Spec ✅，Quality 通过。35 处都看过：每个 `return;` 紧接在它的回调的结尾之前，在顶层；完成值之前之后都是 `undefined`；两个读者（收藏的 `setPromiseToast` 的文案，`deleteGlobalView`、`removeWebhook`）不受影响。I1、m1 见第 1 节。
  - **T9-a**：I1、m1 是 Task 10 的文档和收尾的交接要照录的 spec 文字，所以在 Task 10 改正（进 `task-10-carries.md`）：A.6 的表（23 / 6 / 6、"六处"、"通知的暂缓"），第 5 节 M6 一行（只有两个 quick-actions，原因照实）。代价：Task 10 的提交中 spec 的一段。
- **T10-a、T10-b（文档）**：
  - 实现者先照 plan 的块写四个文件，再加带进的各项：前端改动清单五行扩展加一行键盘焦点；M1 收尾交接的数（80 → 54、删去 26、误归 41）和 23 / 6 / 6；两份交接保持 `open`；spec 的 A.4 一句、A.6 的表、第 5 节 M6 一行。只核对的两项输出与 plan 相同。另报告三个携带项在 HEAD 上不真（第 1 节）和留给修复轮的过时的 spec、plan 句子。
  - **T10-a**：文档写真的数；过时的 spec、plan 句子交给修复轮重数（T2-d）。代价：没有。
  - 评审：Spec ❌（几处 Minor），Quality 要修，C0/I1/M3。文档的每个数在 HEAD 上为真（含 T10-a 的三处：`IconButton` 37 / 32 逐处核对、`CustomMenu` 17 / 17、上限、保留的警告、23 / 6 / 6、W17 各行、README 对照 `rules.go`、`decide.go`）。I1 见第 1 节；m1 前端改动清单把视图放在 M6 下，视图是 M7（`v0-design.md:708`、`M3-design.md:143`）；m2 漏了状态组的加状态按钮名为"Add"和没有把手的标签行不再能整行拖动；m3 一句读起来像 `base.tsx` 还有警告。
  - **T10-b**：I1、m1、m2、m3 现在一轮只改文档的修正（收尾事项必须在提交的文档里）：spec 第 5 节收尾加一条键盘清单，`showTooltip` 一条；spec 第 3 节第 2 条和 A.4 的"视图在 M6"交给修复轮（P1）。代价：几行。修正轮（`0c98207d`）另照 HEAD 改正：没有名字的 `IconButton` 是 27 处（不是 30）；藏住轮廓的是五处（加上 `projects-list-item.tsx:346`）；在 `7d6a416b` 上成员本来就拖不动（只有正在编辑的标签整行可拖）；`profile/sidebar.tsx:104`（编辑个人资料，没有名字）列在收尾的 M3 下。范围复审（sonnet）通过，核对了这些改正。

**整分支评审之后**（`fix-wave-brief.md`，台账；"错了的代价：再一轮小的文档"）
- **F-1**（M1）：删去 `groupedStates` 一串：`GroupList` 的传递，`GroupItem`、`StateList` 的转交，`StateItem` 只在依赖里的读；`tsc`、oxlint、`project-states/` 的 vitest、P6。理由：负责人的标准：不用的代码删干净。
- **F-2**（M2）：`CreateUpdateLabelInline` 去掉 `labelForm`、`setLabelForm`、`forwardRef` 和列表的 `scrollToRef`；`handleClose` 成为必传的 `onClose`，每个调用方只关一次自己的表单；新建的默认颜色就是 `getRandomLabelColor()`；`create-update-label-inline.test.tsx` 看 `onClose`。理由：死代码和重复的关法。
- **F-3**（M3）：删去状态的根组件第二个、不经门的 `useSessionSWR(["PROJECT_STATES", projectId], …)`，和它的导入、`fetchProjectStates` 的解构；包装层按成员关系开的取数是唯一的一个；`states-labels.test.tsx`、P6、S2、W11，请求清单不变。理由：权限驱动的取数只有一份（总体设计 7.7）。
- **F-4**（M4 和搁置项 P3、B 的 T3/T7）：一个共用的 `happenedWithin(page, event, matches, act)` 代替 `sentWithin`、`downloadedWithin`：`act` 之前就听，与 `act` 之后一秒的窗口赛跑，事件先到时结束计时器，之后不再听；调用处读起来自然，可以有薄的名字，机制不留两份。同一步把 `settings-pages.ts`（418 行）的弹窗检查移进新的 `e2e/fixtures/dialogs.ts`；空了的 `downloads.ts` 删除；重跑用到它们的故事和它们发现的变异。
- **F-5**（搁置项 P2）：`e2e/fixtures/assert/identity.ts` 加共用的 `expectNoAddress(page, emails, where)`，W11、P9 都经它；`P6.3` 仍要被发现。
- **F-6**（搁置项 P4）：删去 propel 的 `dashed-circle.tsx` 的 `radius`（接口和三个状态图标的 JSX；图标自己的 `radius` 留着）；`dropdowns/project/base.tsx:141` 的说明写上多选、没有选时的项目图标。
- **F-7**（搁置项 P5）：导出 `fake-form.ts` 的表单类型（`KeptForm`），三个 hoisted 的测试对象以它为类型，去掉重述的形状；报告 `deadsym` 的成员数。
- **F-8**（M5）：前端改动清单的把手一句照实：标签一行的把手只给指针；侧边栏里只有里面的图标是，项目一行（`projects-list-item.tsx:301-314`）和收藏夹（`favorite-folder.tsx:196-207`）的外层 `<button>` 照旧是显示时的停留点，现在没有名字；加上正在编辑的标签一行也不再是放下的目标（`label-drag-n-drop-HOC.tsx:73`）；两处进 spec 第 5 节收尾的键盘清单（`tabIndex={-1}` 和名字，或这一行自己的键盘移动）。
- **F-9**（M6）：`Button` 轮廓一行加一句 `getButtonStyling` 的能取得焦点的调用方（M3 两个文件、M4 两个文件），在 HEAD 上重数。
- **F-10**（M7）：spec 第 5 节收尾加一条，给收尾的交接一轮（M3 设计 13.2）：M4–M8 的五份 M1 收尾交接写明"`eslint-plugin-promise(always-return)` 已由 M3/P11 全仓清零（M3/P11 spec 2.9）"；本轮不改那五份交接，照 M2 收尾的先例由收尾改。
- **F-11**（M8）：收尾的写死的英文清单删去不存在的"Label created/updated successfully."，加上"Marking as default"和标签菜单的"Edit label"、"Delete label"、"Remove from group"；每一条在 HEAD 上核对。
- **F-12**（搁置项 P1，照评审放宽）：spec 和 plan 照 HEAD 为真：spec 写出评审的轮次改了的代码（共用组件的轮廓和偏移、`CustomMenu`、`Popover` 的 `buttonAriaLabel` 和 `common.color`、`common.go_back`、没有颜色的改名、正在编辑的标签不是放下的目标、`fake-form.ts`、F-4/F-5 之后的 e2e 部分、`named`、A11 扣住的撤销、F-1–F-3 的删除）；第 3 节第 12 条的 W17 加同样的行；不真的句子（第 3 节第 12 条"原来整行都能拖"、视图的确认框是 M7、A.8 的超过 400 行的文件、6 处没有处理的 `then()`、项目下拉框的根照 T8-a、"3.2 新增六行"、A.9 的成员数、Task 6 的"待裁定"）；在 HEAD 上重数；变异表加评审加的每一行和它的层、发现它的检查；plan 的块尽量由文件生成（`mkblocks.py`），不手写，每个 `old` 的文字仍在它的块说的地方；`M3-design.md:1300` 的一句照实；M3 设计 §15 的 P11 一行留给评审记录的提交。
- **F-13**（一个观察，记下）：状态的表单打开时可以把焦点放到名称，像标签的表单那样：作为一致的候选进第 5 节收尾的键盘清单。
- 搁置项 B 的 T1、T4-e、T5 照评审的分拣留着；`problemCode()` 跨 M2–M3，只写进评审记录（第 6 节）；五份交接（F-10）和 M3 设计 §15 不在这一轮。一个实现者（不指定模型），一次分派，之后 opus 复审。没有一条是架构层面的：都是 P11 自己的文件里的删除、e2e 的 fixture 和文档。
- **F-14**（复审的 m1、m2）：四处 spec 文字改在评审记录的提交里（P10 的先例：评审的提交改正 spec 的文字）：2.3（`:62`）和第 3 节第 12 条"看不出的"一行加上 F-3 的结果（在一个项目的设置页之间移动、进状态页时不再重取状态：包装层一直挂载、持有那一个取数，标签页原来就是这样）；`:561` 引的 M3 设计的一句在第 1300 行；第 5 节收尾的写死的英文清单加上"Success!"（`project-states/create-update/create.tsx:32`、`update.tsx:30`）。代价：没有。
- **F-15**（复审的 m3）：`expectNoAddress` 放在 `e2e/fixtures/browser.ts` 比放在 485 行的 `assert/identity.ts`（核对数据库的文件）更合适：交给 M3 收尾的 `identity.ts` 拆分（spec 第 3 节第 14 条），记在本记录第 6 节。代价：到那时这个函数在一个大文件里。

## 4. 整分支评审的发现与处理

结论"修复轮之后可以合并，之后做 C9、C10"：Critical 0、Important 0、Minor 8（`$M3TMP/p11final/review.md`）。评审（opus）的范围是 `18b5ce41..0c98207d`（129 个文件，+3,253/−1,054），在 `0c98207d` 的副本上（`git archive`，`--frozen-lockfile` 安装，构建了包）跑过每一道门禁：关键词 69 条、3 个例外，没有命中；turbo 的 `check:types`、`check:lint`、`check:format`、`check:sync` 54/54，上限都恰好相等（web 303、editor 64、ui 15、propel 11、utils 7、hooks 3、constants 1），根目录有 `no-unreachable`；改到的 115 个 TS、JS 文件 15 条，正是裁定的；knip 空；vitest 16/16，web 106 个文件、965 个；`make e2e` 109/110，只有 S3（F4）；没有新的 `as`、`any`、`!`、抑制；`deadsym` 在 BASE 上 382 / 935 / 2、217 行，在 HEAD 上 362 / 981 / 2、218 行，删去的 26 行正是 A.9 的；文案没有孤儿。变异见第 2 节。它还核对了：
- 八类缺陷逐处清扫（请求体、错误、所持的值、关闭、被拒绝时留着、忙、会话、键盘），每处与 spec 相同；2.12 的 8 行与代码相同；
- 权限表：每个角色在两页和个人主页上被提供什么，对照 nerve，没有不一致，每一格有会失败的检查；
- 挂载时的请求逐页、逐个角色，与 2.11 和 S2 的 `REQUESTS` 相同；
- 共用部分的约定：`AlertModalCore` 的 13 个调用方（其余 11 个在 HEAD 上逐个读过，没有一个被拒绝之后关不掉）；`DragHandle` 的 4 个调用方的指针拖动不变；`CustomMenu` 只少了 `outline-none`；`Button`、`IconButton` 只把 `focus-visible:outline-none` 换成 `focus-visible:outline-offset-2`，没有调用方传自己的焦点样式，P11 的页面上按标记核对没有裁掉轮廓的 `overflow-hidden`；`formRefusal` 的三个使用者；`followInSession` 的 49 处；两个 getter 的读者只有两页；
- 删除：`deadsym` 没有新的死的产品代码；删掉的名字没有读者；它的盲点（每个调用方都传、没有读者）在 P11 的文件中找到 M1、M2；
- 重复：M4、M2 之外，两个 hook 的形状相同（各约十行，写成工厂是标准不许的过度设计），两个颜色选择器是 Plane 的两个组件，不算发现。

| 发现 | 处理 |
|---|---|
| **M1** `groupedStates` 从 `GroupList` 经 `GroupItem`、`StateList` 传到 `StateItem`，只在副作用的依赖里出现；`deadsym` 把依赖里的出现算作读（探查 R2 删去它，一切通过） | F-1，`c9e71088` |
| **M2** `CreateUpdateLabelInline` 留着 Plane 的显示和隐藏的接线：三个调用方都只在它为真时渲染表单，`labelForm` 总是真；每次关上设两次；`reset(defaultValues)` 在卸载之前运行；`forwardRef` 落到没有读者的 `scrollToRef` | F-2，`c9e71088` |
| **M3** 状态的根组件再取一次项目的状态（与包装层按成员关系开的取数是同一个键），这一份不经权限；标签页没有这样的取数（探查 R1 删去它，请求清单不变） | F-3，`c9e71088` |
| **M4** `sentWithin`（`settings-pages.ts`）和 `downloadedWithin`（`downloads.ts`）是同一个函数写了两遍 | F-4，`0d2061f5` |
| **M5** 侧边栏项目一行的外层按钮（pragmatic-dnd 登记的把手）和收藏夹的外层按钮在显示时照旧是 Tab 停留点，现在没有名字，前端改动清单却说不是；正在编辑的标签一行也不再是放下的目标，清单没写 | F-8，`83f30b36`：清单照实，两处进收尾的键盘清单 |
| **M6** `Button` 轮廓的 W17 一行漏了 `getButtonStyling` 的能取得焦点的调用方 | F-9，`83f30b36` |
| **M7** M4–M8 的 M1 收尾交接仍把 `always-return` 列为要清的规则 | F-10：spec 第 5 节收尾一条，交给收尾的交接一轮 |
| **M8** 收尾的写死的英文清单有不存在的"Label created/updated successfully."，漏了 P11 自己的"Marking as default"和标签菜单的三条 | F-11，`83f30b36` |
| 搁置项说得太窄：P1 不只是数目和文件表，spec 要写出评审的轮次改了的代码，第 3 节第 12 条的 W17 缺共用组件和名字的行，第 3 节第 12 条"原来整行都能拖"不真，视图是 M7，A.8 漏了 `custom-menu.tsx` 和两个 `common.json`；P3 应一并合并 M4 的两个函数；搁置项 C 缺 M5 的地方、M7 的交接、M8 的清单，"`entry.client` 照 spec 第 5 节"一句不准（第 5 节没有它；A.6 说它的拒绝在 `then` 之前已被吞下，不用交给哪个 M） | F-12（P1 放宽）；F-4；F-8、F-10、F-11 |
| 搁置项 B 的分拣：T1 留着（`error instanceof ApiError && error.problem?.code` 的写法在 M2–M3 五处，一个共用的 `problemCode()` 是整个收尾的改动）；T3 的计时器随 M4 的共用函数修好，`watchPage` 留着（函数自己看着唯一要紧的请求，`seenBy` 之后看成员的页面）；T4-e 留着（变异今天等价）；T5 留着（是故事的脆弱，不是页面的，第二个"Go back"会响亮地失败）；T7 留着，并进 M4 的函数 | 照分拣：F-4 做 T3、T7 的合并；T1 进第 6 节收尾；T4-e、T5、T7 进第 7 节 |
| 没有判断的：C9、C10；`0c98207d` 的持续集成（没有 `gh`）；M4–M8 的页面上轮廓是否被 `overflow-hidden` 裁掉（要在每一页用浏览器）；Firefox 和 Safari 的焦点（探查只在 Chromium）；plan 的文字和从 `7d6a416b` 的复现（P1 之后）；评审的轮次自己的变异没有在 HEAD 上重跑 | C9、C10 在第 2 节；持续集成在第 2 节末；轮廓的裁切和 Firefox、Safari 进第 6、7 节；plan 由 F-12 重新写出、复审重放（第 5 节）；修复轮重跑其中检查被移动的（第 2 节） |
| 不算发现的观察：Enter 打开状态的表单之后，焦点不到名称（加状态时下一次 Tab 先经过那一组的行的控件，编辑时焦点在 `body`），标签的表单把焦点放到名称；状态表单的修改发出表单的每个字段，像 P9、P10 的表单，与 spec 2.2 和 P6 :189-192 相同 | 前者 F-13 进收尾的键盘清单；后者不改 |

**修复轮**（一次分派，`0c98207d` 之上，三个提交；报告 `fix-wave-report.md`）：
- **代码**（`c9e71088`，F-1、F-2、F-3、F-6、F-7）：
  - F-1：`group-list.tsx` 不再传 `groupedStates`，`group-item.tsx`、`state-list.tsx`、`state-item.tsx` 不再收；`state-item.tsx` 的副作用的依赖是 `[draggableElementRef, state, groupKey, isDraggable, move, disabled]`。
  - F-2：`create-update-label-inline.tsx` 去掉 `labelForm`、`setLabelForm`、`forwardRef`、列表的 `scrollToRef`、`useRef` 和 `reset` 的导入；`onClose` 必传，是表单唯一的关法；新标签的颜色是 `getRandomLabelColor()`；三个调用方不再传去掉的 props；测试看 `onClose`：nerve 回答之前不调，之后调一次，被拒绝的一行读 `onClose.mock.calls`。
  - F-3：根组件的取数和导入删除；`states-labels.test.tsx` 去掉 `fetchProjectStates` 的替身和 `use-session-swr` 的 mock；S2 的挂载清单不变（包装层的取数是同一个请求）。
  - F-6：`DashedCircleProps` 没有 `radius`；started、unstarted 的图标不再传它，照旧留着自己的 `radius`（`ProgressCircle` 读它）。实现者的偏离：backlog 的图标的 `const radius = 6` 随之没有读者，一并删除（留着就是一个死的变量）。`base.tsx:143` 的说明写上"with several to pick and none picked yet, the icon of projects"。
  - F-7：`fake-form.ts` 导出 `KeptForm<V>`，三个测试的 hoisted 对象是 `KeptForm<…> & { … }`；`deadsym` 的成员 981 → 975（三个测试各自重述的 `setError`、`submit` 共 6 个没有了；裁定写的是回到 980，实际是 975），`--rows M3` 仍是 218 行。
- **e2e**（`0d2061f5`，F-4、F-5）：
  - 新文件 `e2e/fixtures/dialogs.ts`（163 行）：一个不导出的 `happenedWithin(page, ...[event, matches, act])`，`Watched` 是 `"request" | "download"` 上的元组联合，不用断言；它在 `act` 之前就听，与一秒的 `sleep`（由 `AbortController` 结束）赛跑，在 `finally` 中不再听。`sentWithin`、`downloadedWithin` 是它导出的两个名字。实现者的偏离：`happenedWithin` 不导出（knip 指出它是没有使用者的导出）；`shownWithin`、`hiddenWithin` 也移来（它们与弹窗的检查共用一个私有的 `timedOut`）。
  - 弹窗的检查和"一秒之内"的检查移出 `settings-pages.ts`（418 → 304 行），14 个故事改从 `dialogs.ts` 导入；`downloads.ts` 删除。
  - `expectNoAddress` 在 `assert/identity.ts`（472 → 485 行），W11、P9 都经它。
- **文档**（`83f30b36`，F-8–F-13）：前端改动清单的把手一句和 `getButtonStyling` 一句；spec 第 2 节、2.1–2.8、2.10、2.13，第 3 节第 1、2、9、12、14、19 条，L4 一行，第 4 节，第 5 节收尾（两个外层手柄、F-10 的交接一轮、F-11 的清单、F-13 的候选），附录 A.1、A.2、A.4、A.5、A.7–A.10；plan 由提交的代码重新写出（第 5 节）；`M3-design.md:1300` 的一句。实现者的偏离：`T4.15` 归"请求在路上和被拒绝时"一类；F-11 的清单另加"Create"、"Update"（`create.tsx:43`、`update.tsx:42`，grep 看出它们也是写死的）；第 3 节第 19 条的 W17 一句和"登记的把手"一句同样不真，收藏夹的外层按钮也不是 pragmatic-dnd 的把手（收藏夹登记它的整个元素），一并改正；`TwitterPicker` 的行号随 F-2 由 `:127` 改为 `:105`。
- 门禁（代码在 `0d2061f5`，文档的提交不改代码）：`make lint-web`（69 条规则、3 个例外、没有命中；上限 web 303、editor 64、ui 15、propel 11 都等于条数；手改的文件 0 条，除裁定的 `alert-modal.tsx`；文档的提交上再跑，turbo 54/54）；`make knip`；`make test-web`（web 106 个文件、965 个）；`make e2e` 110 个；P6、P7、P9、W11、S2、A11 的页面版本和 P8 各 3 次，23 个测试 69 次都通过。变异的重跑见第 2 节。
- 实现者的关注：成员数是 975 不是 980；F-9：评审说 M4 的两处现在显示轮廓，照 CSS 读它认为两处都不变（添加筛选的按钮的轮廓是 `CustomSearchSelect` 在 `data-focus` 上画的，`data-*` 的变体排在 `focus-visible` 之后；`dropdowns/layout.tsx` 的按钮自带 `outline-none`），文档照此写，没有在浏览器中核对；`identity.ts` 472 → 485 行（照裁定放在那里，它的拆分是 P10 的收尾一项，spec 第 3 节第 14 条记下）；plan 的 Task 5、7 是 1,674、1,832 行，超过约 1,500 行（spec 第 3 节第 1 条和 A.8 记下，两个 Task 已提交）；规则失误两次（第 1 节）。

**修复的复审**（opus，`0c98207d..83f30b36`，另有只含代码的 `review-fw-code.diff`）：F-1–F-13 都做到，Critical 0、Important 0、Minor 3；结论"可以做浏览器核对和评审记录"。
- 它跑过：`make lint-web`（上限都恰好相等）、`make knip`、修复轮改到的部分的 vitest（11 个文件）和全部的 web vitest（106 个文件、965 个）、`playwright --list` 110 个、`deadsym`（prop 362、成员 975、导出 2）。
- 它在各 Task 结束的提交上重放了 plan 的全部 440 个块：每个 `old` 的文字恰好出现一次，结果与那个 Task 的提交的文件相同（`states-labels.test.tsx` 照 spec A.10 写明的去掉了两个替身的除外）。spec 的数目与 HEAD 相同（141 个文件，类别表 98 行）。
- 三条 Minor：
  - **m1**：F-3 另去掉了在一个项目的设置页之间移动时状态的重取（包装层一直挂载；每个会话的取数在挂载时重新验证），与标签页原来一样；spec 2.2（`:62`）和 W17 的"看不出的"一行（`:182`）要写上。
  - **m2**：spec `:561` 说 M3 设计"第 1301 行"，是 1300；spec `:278` 的英文清单缺"Success!"（`create.tsx:32`、`update.tsx:30`）。
  - **m3**：`expectNoAddress` 放在 `e2e/fixtures/browser.ts` 比放在 472 行、核对数据库的 `identity.ts` 更合适，交给收尾的 `identity.ts` 拆分（spec 第 3 节第 14 条）。
- 处理：m1、m2 由 F-14 改在本记录的提交里（`:62` 是在 2.3，不是 2.2；本提交改在那里）；m3 由 F-15 交给收尾（第 6 节）。

## 5. 最终实现与 plan 的差异

写 plan 时（修订 `18b5ce41`，7,485 行）的块是原型的代码。各 Task 的修正轮、带进下一个 Task 的改动和修复轮改了原型没有的代码；执行中 plan 没有逐轮改，裁定 T2-d、T4-f、T6-d、T7-b、T10-a 把数目、变异表、文件表和落后的块都留给修复轮（搁置项 P1）。修复轮照裁定 F-12 把 plan 改为与提交的代码一致。与写 plan 时的不同都来自第 3、4 节的裁定：
- **生产代码**：
  - T1-b（`716b8351`）：`form-refusal.ts` 已被占用的码的文案经 `errorMessageKey`，不再导入 `PROBLEM_MESSAGES`；说明写明为什么留着类型标注。
  - Task 3 的修正轮（`39222646`）：状态的 `options/delete.tsx` 去掉 `focus:outline-none`。
  - Task 5 的提交（`4e7285e0`）中带进的：两种语言的 `common.json` 加 `common.color`；标签表单的颜色按钮以它为名、去掉 `focus:outline-none`，修改从 nerve 存的颜色开始（T4-a、T4-c）；`@nerve/ui` 的 `Popover` 收 `buttonAriaLabel`（`popover.tsx`、`popovers/types.ts`），状态表单的颜色按钮经它为名（T4-a）；`@nerve/ui` 的 `custom-menu.tsx` 的省略号按钮去掉 `outline-none`（T5-a）。
  - Task 5 的修正轮：propel 的 `button/helper.tsx` 去掉 `focus-visible:outline-none`（`5f178ceb`）；`button/helper.tsx`、`icon-button/helper.tsx` 的基础类加 `focus-visible:outline-offset-2`，`IconButton` 也去掉 `focus-visible:outline-none`（`9ebc375e`）；项目设置侧边栏的 `header.tsx` 的返回箭头以 `common.go_back` 为名（`baae317a`）。
  - 其余各轮不改 plan 之外的生产代码：Task 2、3、4、7、8、9 的提交照 plan 的块写生产代码，与快照在生产代码上的不同只来自之前的轮次（例如 `form-refusal.ts`）；Task 6 只改 e2e；Task 7 的评审之前的一轮和修正轮只改 A11 和 e2e 的 fixture；Task 10 和它的修正轮只改文档。
  - 修复轮（`c9e71088`）：F-1（`group-list.tsx`、`group-item.tsx`、`state-list.tsx`、`state-item.tsx`）、F-2（`create-update-label-inline.tsx` 和列表、一行、组）、F-3（`project-states/root.tsx`）、F-6（`dashed-circle.tsx` 和三个状态图标，`dropdowns/project/base.tsx` 的说明）。
- **测试和 e2e 的 fixture**：plan 之外加的：
  - Task 3（`9425050d`，带进的 T2-a–T2-c）：P6 扣住第一次设默认、核对第二次，组标题收起、展开；`fake-controls.ts:9` 的说明。修正轮（`39222646`）：`settings-pages.ts` 的 `sentWithin`；P6 核对 Edit、Delete 的轮廓和 Shift+Tab，项目成员拖不动状态；`exact: true` 两处。
  - Task 5（`4e7285e0`，带进的 T4-a–T4-d 和 Task 3 的提醒）：P6、P7 按名字"Color"找颜色按钮、核对轮廓；P7 中一次被拒绝的删除、"Remove from group"、一行的菜单和删除按钮的轮廓；行内表单的 vitest 加没有颜色的改名一行。修正轮：P7 核对 Add 的轮廓和偏移、返回箭头的名字、轮廓和偏移，标题照实；`label.store.test.ts`、`state.store.test.ts` 各一行。
  - Task 6（`fc9d55bf`）：`e2e/fixtures/auth.ts` 的 `named` 收要设的名字，P1 的两处调用传 `{ display_name }`（T6-a）。
  - Task 7（`b691d42f`）：新的 `core/lib/fake-form.ts` 代替三份替身（T7-b）；A11 的令牌名"abandoned"。评审之前的一轮（`746c1300`）：A11 扣住撤销和新建，`settings-pages.ts` 的 `closedByBackdropClick`。修正轮（`d870fe7d`）：新的 `e2e/fixtures/downloads.ts` 的 `downloadedWithin`，A11 的两个迟到的回答经它。
  - 修复轮：`c9e71088` 中行内表单的测试看 `onClose`，`states-labels.test.tsx` 去掉两个替身，三个测试以 `KeptForm` 为类型；`0d2061f5` 中新的 `dialogs.ts`（`happenedWithin` 和它的两个名字，弹窗和"一秒之内"的检查），`settings-pages.ts` 418 → 304 行，`downloads.ts` 删除，`assert/identity.ts` 的 `expectNoAddress`，16 个故事改导入或调用（14 个从 `dialogs.ts` 导入，P9、W11 经 `expectNoAddress`）。
  - 数目：vitest 在 `7d6a416b` 上 95 个文件、849 个；写 plan 时 106 个、964 个；修复轮之后 106 个、965 个（多的是 T4-c 的一行）。e2e 写 plan 时和最终都是 110 个，各轮的检查加在已有的故事里。
- **plan**（修复轮，F-12；`$M3TMP/p11tools-fw`）：块由提交的代码重新写出，不是手写：
  - `build_snaps.py` 从各 Task 结束的提交（`git archive`，P11 改到的 141 个文件）写出各 Task 的树，修复轮的改动并进最后一个改到那些文件的 Task（F-1、F-3 → Task 3，F-2 → Task 5，F-5 → Task 6，F-4、F-7 → Task 7，F-6 → Task 8，文档 → Task 10）；之后的 Task 还改的 `states-labels.test.tsx` 在 Task 3、4 的版本中同样去掉两个替身。0 个问题。
  - `mkblocks.py` 写出块（0 个问题），`compose_plan.py` 拼出 plan：8,892 行，Task 1 之前 110 行，各 Task 329、1,461、1,107、854、1,674、277、1,832、632、450、136 行；评审加的变异在各 Task 的表中。
  - `planapply.mjs check` 从 `T0` 起 440 个块（`whole` 16、`file` 19、`replace` 404、`delete` 1）都恰好应用；`replayblocks.sh` 逐 Task 应用之后与那个 Task 的树逐字节相同，Task 10 之后与修复轮之后的代码相同。块与原来相同的只有 Task 2、4、9 的。
  - 说明的文字照实改：Task 6 的第 3 节第 5 条是已裁定的，L4 的数是 6，文件表和步骤中的正则随之改。
  - **复审在各 Task 结束的提交上重放了全部 440 个块**：每个 `old` 恰好出现一次，结果与各 Task 的提交相同（写明的两个替身除外）。
  - 修复轮之后没有在副本中逐 Task 再跑门禁（brief 不要求完整的复现）；修复轮之后的代码上的门禁和反复见第 4 节。
- **文件大小**：
  - 超过约 400 行的都是原来就超过的：`tools/keywords.json` 2,387 → 2,390 行（规则的数据）；两种语言的 `common.json` 527 → 528 行（文案的数据，`common.color`）；`@nerve/ui` 的 `custom-menu.tsx` 492 行（Task 5 只改省略号按钮的一行，行数不变）；`e2e/fixtures/assert/identity.ts` 472 → 485 行（F-5 的 `expectNoAddress`；裁定 F-15：随收尾的拆分移走）；`power-k/ui/pages/context-based/work-item/commands.ts` 415 → 418 行（Task 9 的三处 `return;`，M4 拆它）。
  - P11 写的文件中最长的是 `s2-web-app.spec.ts` 375 行、`a11-api-tokens.spec.ts` 370 行、`p7-labels-page.spec.ts` 296 行、`p6-states-page.spec.ts` 289 行；新的 `dialogs.ts` 163 行。改到的已有故事中 P3 398 行、P1 396 行、W1 396 行、W3 394 行、P5 392 行；`settings-pages.ts` 304 行。
  - plan 的 Task 5、7 是 1,674、1,832 行，超过约 1,500 行（spec 第 3 节第 1 条）：Task 5 加了共用组件的焦点和名字、行内表单的改写和 P7 的核对，Task 7 加了 `fake-form.ts`、`dialogs.ts` 和改导入的 14 个故事。两个 Task 都已提交，plan 是它们的记录。
  - 变长的 Plane 文件 41 个，都是本 Phase 的对象（spec A.8）。

## 6. 移交事项

本 Phase 关闭的（spec 第 7 节）：
- M1 收尾交接的死成员、oxlint：`--rows M3` 剩下的每一行写明（spec 附录 A.9 和第 2 节），`always-return` 全仓清零，上限照新的条数；处理结果写在交接里（Task 10，`49749f5d`），状态保持 `open`，M3 的收尾改为 `closed`。
- M1-P2 交接的个人主页的页面一侧：C10 写进本记录（第 2 节），处理结果写在交接里，状态保持 `open`，由 M3 的收尾关闭。
- P10、P9、P8b、P8a 的 review 第 6 节 P11 一行；P7a、P7b 的页面一行；P9 review 收尾一行中 `confirm-workspace-member-remove.tsx` 的去处（收尾）。每一项的落点见 spec 第 3 节的交接表。

之后的里程碑和 M3 的收尾（写进它们的 spec 或收尾写的交接；spec 第 5 节各行照录，下面是执行之后的补充）：
- **M4**：
  - spec 第 5 节 M4 一行照录：工作项页面上的标签新建改经 `useLabelChanges().create`（之后可以加关键词规则）；个人主页的工作项一栏和访客看到的筛选工具栏；`work-item/commands.ts` 418 行；误归的 16 行；Task 9 的文件中 M4 的两条警告和两处复制的拒绝没有处理；两个工作项的删除确认框的 `onSubmit` 必传，改写时保持；P8a、P8b、P9、P10 交给 M4 的照旧。
  - 执行之后加的：
    - 访客的个人主页的页头（照录的一项）C10 在浏览器中看到了：布局、筛选、Display 的控件在"没有权限"之上（第 2 节）。
    - `getButtonStyling` 的两处（`rich-filters/add-filters/button.tsx:71`、`dropdowns/layout.tsx:77`）：修复轮照 CSS 推断它们的轮廓不变（前者的轮廓由 `CustomSearchSelect` 在 `data-focus` 上画，后者自带 `outline-none`），没有在浏览器中核对，C9 也没有看；M4 对接这些页面时用键盘核对。
- **M5**：spec 第 5 节没有 M5 一行。执行之后加的：`AlertModalCore` 的改动到达附件的删除确认框（请求在路上时关不掉，spec 第 3 节第 2 条的 W17），M5 的页面能运行时核对；`ButtonAvatars` 的 `showTooltip` 的调用方中有 M5 的附件（收尾一次删，见下）。
- **M6**：spec 第 5 节 M6 一行照录：`progress-stats/` 的 8 行；Task 9 的文件中 M6 的 6 条警告；迭代、模块的复制链接的拒绝没有处理（两个 quick-actions）；P10 交给 M6 的照旧。
- **M7**：spec 第 5 节 M7 一行照录：`workspace-notifications.ts` 的 17 行；`snooze/modal.tsx` 5 条；默认视图的复制链接的拒绝没有处理；P10 交给 M7 的照旧。
- **M8**：spec 第 5 节 M8 一行照录：`webhook.store.ts` 1 条；P10 交给 M8 的照旧。
- **M4–M8 共同的**（执行之后加的，整分支评审的说明和没有判断的）：
  - M4–M8 的 `AlertModalCore` 确认框随 Q2 在请求路上时关不掉，可被拒绝时它们照旧关上（不是 A-M2），不在 P11 的范围（spec 第 3 节第 2 条，整分支评审的说明）：各 M 改写它们的弹窗时照 A-M2。
  - 共用组件的焦点轮廓（`Button`、`IconButton`、`getButtonStyling`、`CustomMenu`）在 M4–M8 的页面上是否被 `overflow-hidden` 裁掉，整分支评审只在 P11 的页面上按标记核对：各 M 在自己的页面上用键盘核对。
- **契约**：P11 没有新的。
- **收尾**：
  - spec 第 5 节收尾各条照录：`ModalCore` 的重写（`AlertModalCore` 的 `tabIndex={1}` 和它的警告随之去掉，之后 P6、P7 加上删除确认框的 Tab 循环；重写保持 `isSubmitting` 时关不掉）；写死的英文（状态、标签的文件中 Plane 原有的，"Marking as default"、标签菜单的三条和"Success!"在内；`confirm-workspace-member-remove.tsx`；P9、P10 列过的）；离开工作区的入口；键盘清单（`TwitterPicker` 的输入框；没有名字的 27 处 `IconButton`；看不到轮廓或 Tab 到不了的五处和 `member-columns.tsx:72`；项目下拉框的根和 `issues/select/base.tsx` 同样的写法；侧边栏的两个外层手柄按钮；状态的表单打开时把焦点放到名称）；`ButtonAvatars` 的 `showTooltip`（跨 M3–M7 的 9 个文件 10 处，一次删）；只读工作区的 router store 的 36 个文件，含个人主页的 `layout.tsx`；误归的 41 行写进 M4、M6、M7 的交接；收尾的交接一轮（M4–M8 的五份 M1 收尾交接写明 `always-return` 已全仓清零，F-10）；P8a 修复轮的搁置项 P11（`Workspace.organization_size`）；P10 第 5 节收尾的其余各项；M3 设计 §15 的 P11 一行（本提交改）。
  - `ModalCore` 的重写和离开工作区的入口也是负责人在 M3 设计第 12 节收尾的两项（`1fa8f921`）：与上面照录的是同一项，各做一次。
  - 执行之后加的：
    - 浏览器核对看到的（C9）：状态页的组标题的文字和按钮的名字是小写的组键（"backlog"、"started"；`group-item.tsx:65` 以 CSS 的 `capitalize` 显示成大写开头，辅助技术读到的是小写的键），没有文案；与写死的英文一起定。
    - 搁置项 B 的 T1（整分支评审的分拣）：`error instanceof ApiError && error.problem?.code` 的写法在 M2–M3 的五处（`form-refusal.ts:27`、`security.tsx:96`、`use-create-workspace.ts:61`、`use-invitation-answer.ts:33`、`use-project-fetch.ts:74`），一个共用的 `problemCode()` 是整个收尾的改动，不是 P11 的。
    - F-15：`expectNoAddress` 移到 `e2e/fixtures/browser.ts`，随 `assert/identity.ts`（485 行）的拆分一起（P10 收尾的一项，spec 第 3 节第 14 条）。
    - 死代码的工具的盲点：`deadsym.mjs` 看不到每个调用方都传、却没有读者的 prop，也把副作用的依赖里的出现算作读（整分支评审的 M1、M2，Task 8 评审的 `radius`、`showTooltip` 都是读代码找到的）：与 P10 记下的 `i18norphans.py` 的盲点和 PF-L5 一起看工具。
    - 整分支评审的小处（不算发现）：`ProjectSettingLabelItem.isEditable`、`ProjectSettingLabelGroup.isEditable`、`LabelItemBlock.disabled`、`StateList.disabled` 是可选的、有默认值，没有调用方靠这个默认值。
    - 焦点只在 Chromium 中核对（整分支评审的探查、C9）：Firefox、Safari 的焦点与 P10 第 5 节收尾的 Safari 的输入法一起看。

## 7. 已知限制

- **删除确认框的 Tab 循环不核对**（spec 第 3 节第 3 条，Q3，负责人 2026-10-10）：`AlertModalCore` 的确认按钮是 `tabIndex={1}`（`alert-modal.tsx:97`，`@nerve/ui` 上限中的一条警告），到收尾的 `ModalCore` 重写之前，状态、标签的删除确认框的 Tab 循环只由评审看住（spec A.2）。经 `ModalCore` 的弹窗后面的页面对辅助技术可见（P10 的缺陷 ii，收尾的同一项任务），这两个确认框同样。
- **6 处 `then()` 的拒绝没有处理**（spec A.6）：`entry.client.tsx` 的拒绝在 `then` 之前已被吞下，`then` 之后没有能拒绝的；其余 5 处复制链接（M4 2、M6 2、M7 1）之前之后同样没有处理，`return;` 不改它们，交给各 M。
- **键盘清单中的各项**（spec 第 5 节收尾）照旧：`TwitterPicker` 的十六进制输入框只看得到光标；27 处 `IconButton` 没有名字；五处看不到轮廓或 Tab 到不了，`member-columns.tsx` 的省略号按钮被它自己的透明藏住；项目下拉框的根（M4、M6 的表单上）和 `issues/select/base.tsx` 是没有名字的 Tab 停留点，`role="presentation"` 在那里不生效；侧边栏项目一行和收藏夹的外层手柄按钮是显示时的停留点，现在没有名字，按 Enter 什么都不做；状态的表单打开时不把焦点放到名称。
- **被拒绝的移动不刷新列表**（C9 的观察）：nerve 拒绝一次标签的移动之后，页面仍照它原来知道的层级显示（C9 中 Feature 仍在顶层），直到重新载入或 SWR 的下一次重新验证。store 只写 nerve 的回答，被拒绝的修改什么都不改；nerve 的 422 是最后一道，页面说明为什么。
- **访客的个人主页的页头**（spec 第 3 节第 6 条）：照 `isIssuesTab` 渲染工作项的筛选工具栏，访客也看到布局、筛选、Display 的控件（C10 看到），它们什么都不做；同一个会话里先以成员的身份打开过的，换布局会以旧的筛选请求工作项，nerve 答 404。M4 处理。
- **访客经 PAT 离开工作区**，直到收尾的入口（Q5）。
- **`ButtonAvatars` 的 `showTooltip`** 是必传、没有读者的 prop，到收尾一次删（T8-c）。
- **每次挂载都重取、聚焦时不取**（P8a 的 F-1）。F-3 之后，在一个项目的设置页之间移动时状态和标签由项目的包装层持有，不重取（F-14）。
- **焦点只在 Chromium 中核对**；共用组件的轮廓在 M4–M8 的页面上是否被裁掉没有核对；`getButtonStyling` 在 M4 的两处只照 CSS 推断（第 6 节）。
- **一秒的窗口是界限，不是证明**（`sentWithin`、`downloadedWithin`，`dialogs.ts` 的 `happenedWithin`；搁置项 B 的 T3、T7）：项目成员的拖动、迟到的令牌的下载在一秒之后才发生的话，这两步看不到。
- **移动被 nerve 以别的字段的 422 拒绝没有测试行**（T4-e）：nerve 今天在移动上答不出别的 422，变异等价。
- **`P7.1`、`P7.2` 在 `create-token-modal.test.tsx` 的 vitest 一层存活**，由 A11 发现（Task 7 的轮次和修复轮的重跑相同）。
- **拖动把手的 vitest 自己写 React 的语义**（spec 第 6 节、第 3 节第 10 条）：React 改了 `useState`、`useRef`、`useEffect` 和提交时调用 ref 的次序时，测试和页面会不一致；P7 的页面版本照旧拖动，看住真实的那一边。
- **P7 的故事的脆弱处**（搁置项 B 的 T5，留着）：按名字找"Go back"，有第二个时会有歧义（会响亮地失败）；从 General 按 Shift+Tab 假定中间没有可聚焦的。
- **复现只到修订**：逐 Task 的门禁复现是修订时的（spec A.10）；修复轮由提交的代码重新写出块，核对和重放了 440 个块，没有再逐 Task 跑门禁。
- **超过约 400 行的文件**和 **plan 的 Task 5、7 超过约 1,500 行**（第 5 节）：`assert/identity.ts` 485 行到收尾的拆分，`work-item/commands.ts` 418 行到 M4。
- **`--rows M3` 剩 218 行**：每一行有读者，或属于 M4、M6、M7（第 2 节、spec 附录 A.9）。
- 浏览器核对看到的其余一处（小写的组键）列在第 6 节的收尾，这里不重复。
