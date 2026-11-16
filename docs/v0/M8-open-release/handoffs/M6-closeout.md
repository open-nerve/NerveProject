---
status: open
from: M6/closeout
to: M8
created: 2026-11-07
---

# M6 交给 M8：CI 的 e2e 作业期限与间歇失败的故事、清理分页器的副本、e2e 数据库的连接余量、server 作业的分路与 lint 工具的在线下载、本机测试容器的端口冲突、e2e 作业在线安装浏览器的系统依赖、编辑器重建图片节点时中止的读取、工作项根 store 的行数、Firefox 再次载入后的字体预加载警告、出现后一帧内按下关闭的通知不再消失、两份相同的 useLocalStorage、项目页第一次读偏好失败时没有重试、上方的内容晚到时打开着的下拉列表不随按钮移动、经接口存下的按截止日期分组网页没有它的列、项目地址的守卫只按变量名认 router store 与列表的正文经 router store 读项目页的偏好、G2 页面故事在上海午夜前后的几秒会失败、默认 CI 不跑的对等与发布前的运行、自动归档遇到失败的项目即停、访客在命令面板看到项目设置的命令、冻结候选查询的规划器余量、命令面板打开子菜单时没有选中的行、富筛选的 “Remove filter” 是英文字面量

这是接收方尚待完成的工作。转交只表示责任和验收条件已落实到此处，不表示本页问题已经解决。M6 进行中发现的条目随发现写入；其余由 M6/P8 补齐（M6-M8-16 起，[P8 spec](../../M6-cycles-modules/specs/P8-final-acceptance.md) §2.13、§3），各条写明出处。延续 [M5 → M8 交接 M5-M8-06](M5-closeout.md#M5-M8-06) 中仍开放的观察。

<a id="M6-M8-01"></a>
## M6-M8-01 e2e 作业期限由 25 改为 35 分钟，C7 与 C12 再次间歇失败

- M6 设计合入的 main CI 42281169459（main `d1ec8c69`，只改文档；同一内容的分支 CI 全绿）连续两次未通过：
  - attempt 1：e2e 417 条中 1 条失败，M4 的 C7（`e2e/stories/issue/collaboration-interactions.spec.ts:422`，键盘选主题）。选 Dark 收到 503 后，第一次 `reopenDuringExit("Control+k")` 期待弹窗面板停在 30 秒的慢退场（`data-leave`），5 秒内找不到面板元素。
  - attempt 2（重跑失败的作业）：安装 Playwright 浏览器一步用了 7 分 19 秒，e2e 作业在 25 分钟期限处被取消，当时故事跑到约第 314 条；取消前 M4 的 C12（`e2e/stories/issue/review-interactions.spec.ts:83` 的 kanban 布局，见 M5-M8-06）又失败一次，错误详情因取消未打印。
- 负责人 2026-11-07 批准把 e2e 作业的 `timeout-minutes` 由 25 改为 35（`.github/workflows/ci.yml`），server 作业仍为 25。依据：绿色 e2e 作业已用 23–24 分钟、故事本身 21–22 分钟（M5-M8-06），浏览器安装偶发停滞，M6 还会加入迭代与模块的故事。
- C7 与 C12 的原因当时均未查实，均为界面时序类的间歇失败；配置中没有重试，失败即算数。
- M6/P1 检查点的分支 CI 42334013393（`c528d682`，已含 35 分钟期限）：attempt 1 的 e2e 417 条中 416 过，C12 在同一处（`review-interactions.spec.ts:83`）的 list 布局失败——新建工作项弹窗 5 秒内未关闭（`toHaveCount(0)` 收到 1，`:117`）；只重跑失败的作业，attempt 2 通过（e2e 作业 16 分钟）。
- M6/P2 候选 `784d8e0a` 的分支 CI 42380907752 attempt 1：e2e 426 条中 425 过，M5 的 F16 (page)（`e2e/stories/files/f16-content-lifetime-page.spec.ts:672`，角色改变又改回的故事）失败——展开评论的“Attachments”折叠区后，5 秒内找不到 `Download owner-note.txt`（`:698`）。这与 [M5-closeout](M5-closeout.md) 所记 F16 重连故事的“点开折叠区与晚到的版面位移竞争”是同一类（该条已建议同类辅助函数一并检查）；P2 没有改动文件与评论的前端。只重跑失败的作业，attempt 2 的 F16 通过，但 I21 因 P2 自己的测试竞争失败（P2 在 `78c7f7d3` 修复，见 [P2 阶段记录](../../M6-cycles-modules/reviews/P2-modules-backend-review.md) §5），不属本条。
- M6/P2 合入的 main CI 42399566014 两次未过（自动归档的锁探测被启动时的修剪任务骗过；S3 测试容器发布端口撞上本机连接的临时端口），按负责人指示由修复分支 `codex/m6-p2-ci` 修根因而非重跑：锁探测改为只看其行为者自己的锁与等待，测试容器在端口冲突时重建（至多三次），另修本地暴露的 S7 (page) 菜单按键竞争。M5-M8 交接所记“启动时运行的周期任务插到锁测试”一条，凡全局探测项目行锁的 bootstrap 测试已按此改写；其余仍开放的观察照旧。经过见 [P2 阶段记录](../../M6-cycles-modules/reviews/P2-modules-backend-review.md) §5。
- **C12 的根因已查实并修复**（M6/P4）：P4 合入的 main CI 42809569341 的 e2e 又在 C12 的 list 布局同一处（`:117`）失败。这次从 trace 查实：弹窗刚打开时描述编辑器尚未建成，编辑器的引用却已挂上一套操作空编辑器的 API，`isEditorReadyToDiscard()` 答 false，弹窗把 Escape 当成“编辑器忙”而拒绝关闭（错误提示 “Editor is not ready to discard changes”）。修复分支 `codex/m6-p4-ci` 改为编辑器建成之前引用为 null，经过见 [P4 阶段记录](../../M6-cycles-modules/reviews/P4-progress-transfer-review.md) §5。此前留有详情的各次 C12 失败（M5-M8-06 与本条上文）都是“弹窗打开后不久按 Escape、5 秒内未关”，与此一致。收集箱新建弹窗的同类问题交 M7（[M6-M7-01](../../M7-collaboration/handoffs/M6-closeout.md#M6-M7-01)）。
- 余量的一个数据（M6/P6a 的 T10 检查点）：分支 CI 43418051413（`82dac1ef`）全绿，e2e 作业 131867770453 用了 25 分 18 秒（E2E 一步 23 分 09 秒，452 条、2 个工作进程），比基线 main CI 43373039561 的 19 分 14 秒（E2E 一步 17 分 37 秒，453 条）多约 6 分钟。逐条比对：页面故事整体慢 36%、接口故事慢 11%，与模块无关的故事（工作项列表、文件、项目设置）同样变慢；本机同一套故事与 P5b 的最终门禁逐条持平（页面 1.00、接口 0.99、其他 1.02，总时长都是 3.4–3.5 分钟）；同一次运行中没有改 Go 的 server 两路也一快一慢。判为 runner 的速度差异，不是产品的回归；同一批故事在不同的 runner 上相差约三成，复核余量时要计入。
- **C7 的再现**（M6/P6a 合入）：main CI 43469787621（main `06b72d4b`）attempt 1 的 e2e 作业 132048379310 中 462 条有 461 过，C7 在同一个故事（`collaboration-interactions.spec.ts:422`，键盘选主题）的同一个断言失败：第二次 `reopenDuringExit("Meta+k")`（`:474`，前一次 `Control+k` 已过、按 Escape 之后）期待面板停在 `data-leave`（`:464`），面板元素还在、已 `hidden`、`data-leave` 为 null（上文 42281169459 那次是第一次 `Control+k` 时 5 秒内找不到面板元素）。P6a 在命令面板只改了模块的命令与菜单（迭代的菜单改用共用的取数门），没有动主题选择与这个故事。照本条重跑一次（只重跑失败的作业），attempt 2 的 e2e 作业 132062963902 通过（462 过，作业 25 分 00 秒）；这个时长作为 P6b 的 e2e 时长基线（[P6b spec](../../M6-cycles-modules/specs/P6b-module-list.md) §2.10），也是余量的又一个数据：同一棵树的分支 CI 43463416098 的 e2e 作业为 21 分 38 秒。
- 余量的又一个数据（M6/P6b 的 T10 检查点，2026-11-13）：分支 CI 43517662603（`3a5c3cf6`）全绿，e2e 作业 132209901127 用了 25 分 38 秒（22:28:06Z–22:53:44Z）；那时本段的页面故事还没有加入，跑的故事与 main 相同，比上一条的基线 25 分 00 秒多 38 秒，作为 P6b 推送前估算 e2e 时长的校准点（[P6b spec](../../M6-cycles-modules/specs/P6b-module-list.md) §2.10）。
- **I4 的表格排序故事超出自己的时限，已查实并修复**（M6/P6b 合入，2026-11-14）：P6b 的最终分支 CI 43545459485（`c0572192`）全绿，e2e 作业 132300181032 用了 23 分 09 秒；同一棵树合入后的 main CI 43549053092（main `954a1838`）attempt 1 的 e2e 作业 132311928038（03:14:42Z–03:42:23Z）中 469 条有 468 过，M4 的 I4 (page) 表格排序故事（`e2e/stories/issue/i4-list-order-calendar.spec.ts:267`，一条故事里依次走标签、负责人、子项数、截止日期四列各升降两向，共 8 次排序，每次都翻到真实的第二页）以 “Test timeout of 90000ms exceeded” 失败，用时 1.6 分钟。
  - trace：8 步全部通过，没有一个断言失败；准备 101 条工作项用了 4.9 秒，8 步依次 20.3、13.4、9.3、8.9、7.6、9.2、9.0、8.4 秒，最后一步在第 91 秒结束，超出这条故事的 90 秒时限（`test.setTimeout(90_000)`）。第一步里点开表头菜单等了 6.7 秒、`aria-sort` 的断言用了 5.4 秒：换排序后一百行的表格重新渲染要数秒，与 [M5-closeout](M5-closeout.md) 所记 I4 的那次是同一种慢。
  - 余量：这条故事在此前四次 CI（P6a 的 main CI 两次、P6b 的检查点与最终分支 CI）用了 1.3、1.3、1.3、1.1 分钟（90 秒时限的 73–87%），本机 16.4 秒，CI 慢 4–6 倍；这次整套故事用了 25.4 分钟（那四次 20.9–23.6 分钟），是记录中最慢的一次 runner，余量就不够了。P6b 没有改表格、排序与这条故事；同一棵树的分支 CI 上它用了 1.1 分钟。
  - 修复（分支 `codex/m6-p6b-ci`，没有重跑）：按列拆成四条故事，各自准备同样的 101 条、走升降两向，各有 90 秒；故事的正文不改（`git diff -w` 只有外层的循环）。本机每条 5.2–5.7 秒（四条合计 21.6 秒，多出的 5.2 秒是多准备三次），`--repeat-each=3` 与整个 I4 文件在 chromium 上通过；按 4–6 倍推算，CI 上每条约 21–33 秒。
  - 同一次运行的余量排查（只记录、未改）：用掉自身时限 75% 以上而通过的故事有 I19 (page) `i19-member-issues.spec.ts:340` 26.8 秒（默认 30 秒，正常的一次 19.7 秒）、F11 (page) `fixtures/file-story.ts:37` 25.7 秒、G1 (page) `g1-create-cycle-page.spec.ts:39` 25.5 秒、I19 (page) `:287` 24.4 秒、F11 `:40` 22.9 秒、P5 (page) `p5-project-members-page.spec.ts:31` 22.8 秒，都是单一场景、用 Playwright 默认的 30 秒。改默认时限或逐条加时限属 CI 预算，交下文的验收语（M8 的 CI 工作一并复核，调整须负责人同意）；在此之前若其中一条超时失败，先照本条看 trace 里各步的用时，再定修法。
- 余量的又一个数据（M6/P7a 的 T9 检查点，2026-11-14）：分支 CI 43573631430（`f1646370`）全绿，e2e 作业 132386476220 用了 19 分 32 秒（08:13:18Z–08:32:50Z）；那时本段的两个页面故事还没有加入，但迭代页、模块页的故事所在的项目开着开关，行上与表格里多了迭代、模块两项（本机门禁里 chromium 的 `stories/planning/` 各条合计由 P6b 最终门禁的 246.6 秒到 249.7 秒，9 个工作进程、只作对照）；比 P7a 的基线（修复合入后的 main CI 43557098427 的 e2e 作业 20 分 20 秒）少 48 秒，作为 P7a 推送前估算 e2e 时长的校准点（[P7a spec](../../M6-cycles-modules/specs/P7a-work-item-planning.md) §2.11）。本段的两个新页面故事（G4、G11）本机 chromium 合计 6.5 秒（预算约 20 秒）。
- 余量的又一个数据（M6/P7a 合入，2026-11-14）：P7a 推送前的估算约 20 分 25–27 秒（基线 20 分 20 秒加两个新故事），上界约 28 分 22–28 秒（以 27 分 41 秒为锚）。最终 HEAD `5d6dab04` 的分支 CI 43600392398 全绿，e2e 作业 132467556854 用了 23 分 48 秒（13:49:22Z–14:13:10Z，474 过、故事 21.8 分钟）；合入后的 main CI 43603772626（main `f42f9d64`）attempt 1 全绿，e2e 作业 132478430823 用了 23 分 43 秒（14:31:09Z–14:54:52Z，474 过、故事 21.6 分钟）。两次都比估算高约 1.16 倍、是上界的约 0.84 倍。与上一条的检查点共有的 472 条逐条比对，两次都慢 1.22 倍：接口故事 (API) 慢 13%，页面故事 (page) 与其余的故事慢 23–25%，与 P7a 无关的故事同样变慢；本机门禁里同一套故事持平（3.7、3.6、3.7 分钟）。与上文 P6a 检查点的那次同形（浏览器的故事比接口的慢得多），判为 runner 的速度差异。23 分 43 秒是 P7b 的 e2e 时长基线；P7b 的 T1 以它与这两次实测重算 P7b、P7c 合计的余量约 22.4–27.1 秒（[P7b spec](../../M6-cycles-modules/specs/P7b-list-vocabulary.md) §2.14，上界仍以 27 分 41 秒为锚）。照“同一批故事在不同的 runner 上相差约三成”（上文），23 分 43 秒若是在快的 runner 上，慢的 runner 上就会越过 30 分钟（× 1.31 约 31 分 04 秒）；这两次比检查点的 runner 已慢 1.22 倍，不像快的 runner，但 M8 复核余量时要计入。
- 余量的又一个数据（M6/P7b 的 T9 检查点，2026-11-15）：分支 CI 43629579359（`b3279330`）attempt 1 全绿，e2e 作业 132558145042 用了 20 分 25 秒（19:17:14Z–19:37:39Z）；那时本段的页面故事还没有加入，既有故事一行不改，但项目页的列表多了迭代、模块的词表（筛选、分组、拖拽），迭代页、模块页的故事所在的项目开着开关（本机门禁 474 条通过、3.7 分钟，chromium 的 `stories/planning/` 各条合计 256.2 秒，P7a 的 T9 检查点 249.7 秒，9 个工作进程、只作对照）。比 P7b 的基线（上一条的 main CI 43603772626 的 23 分 43 秒）少 3 分 18 秒（约 0.86 倍），比 P7a 的检查点 19 分 32 秒多 53 秒，作为 P7b 推送前估算 e2e 时长的校准点（[P7b spec](../../M6-cycles-modules/specs/P7b-list-vocabulary.md) §2.14）。本段的两个新页面故事本机 chromium 合计 6.9 秒（G15 (page) 3.5 秒，G4 (page) 与 G11 (page) 3.4 秒；预算 14 秒），三种引擎里 firefox 5.0 + 4.6 秒、webkit 4.2 + 4.5 秒（3 个工作进程）；P7b 的 T17 推送之前照 spec §2.14 以这些数算估算、上界与校准。
- 余量的又一个数据（M6/P7b 推送之前的估算，2026-11-15，最终候选 `16c52f41`）：本段的三个新页面故事本机 chromium 合计 6.9 秒（修复轮没有改 e2e）。估算：基线 1423 秒（P7a 的 main CI 43603772626）加 6.9 秒 × 1.67–2.25、两个工作进程摊开，约 23 分 49–51 秒；上界：S = 1702.40–1708.15 秒（P7b 的 T1）加 6.9 秒 × 3.6–4.1，约 28 分 47–56 秒；校准点：上一条的检查点 1225 秒加同一增量，约 20 分 31–33 秒；慢的 runner（× 1.31）乘在估算上约 31 分 12–14 秒，是 P7b T1 的观察、不是 spec 的停下条件（负责人已知）。P7c 的余量 (1800 − S) / f − 6.9 = 15.5–20.2 秒，不少于 P7c 的 8 秒。估算与上界都低于 30 分钟，所以推送；P7b 最终 HEAD 的分支 CI 与合入后的 main CI 的实测见下一条（[P7b 阶段记录](../../M6-cycles-modules/reviews/P7b-list-vocabulary-review.md) §7，P7c 的 T1 回填）。
- 余量的又一个数据（M6/P7b 合入，2026-11-15，P7c 的 T1 以公开的 jobs 接口与作业日志核对）：最终 HEAD `4fe308ce` 的分支 CI 43702923005 attempt 1 全绿，e2e 作业 132781366495 用了 27 分 50 秒 = 1670 秒（10:26:46Z–10:54:36Z，477 过、故事 25.7 分钟），是至今最慢的 e2e 作业（在上文 43549053092 的 27 分 41 秒之上）：上一条估算的约 1.17 倍、上界的约 0.96–0.97 倍，低于 30 分钟；是校准点约 20 分 31–33 秒的约 1.36 倍。合入后的 main CI 43706438924（main `d052213f`，树与 `4fe308ce` 相同）attempt 1 全绿、没有重跑，e2e 作业 132792042944 用了 23 分 42 秒 = 1422 秒（11:10:31Z–11:34:13Z，477 过、故事 21.7 分钟）。同一棵树、同样的 477 条逐条比对，分支 CI 慢 1.20 倍：接口故事 (API) 持平（0.999），页面故事 (page) 慢 22%、其余慢 19%，与上文 P6a、P7a 的几次同形，是 runner 的速度差异；相对校准点的 1.36 倍里，同一棵树在两个 runner 上就差 1.17 倍（main CI 是校准点的约 1.15 倍）。P7b 的三个新页面故事在这两次 CI 上合计 21.5、17.3 秒，是本机 6.9 秒的约 3.1、2.5 倍，低于上界所取的 f = 3.6–4.1。
  - 1422 秒是 P7c 的 e2e 时长基线。P7c 的 T1 照 [P7c spec](../../M6-cycles-modules/specs/P7c-palette-mobile.md) §2.12 重算：S 取 1661 + 19.3 × f（19.3 秒 = I4 拆分的 5、P7a 的 6.5、P7b 的 6.9 与 C4、C8 关系一支的等待约 0.9，后者算作加进来的故事时长）与已含这些故事的实测（1670、1422 秒）中的大者，f = 3.6–4.1 时 S = 1730.48–1740.13 秒；P7c 的余量 X ≤ (1800 − S) / f = 14.6–19.3 秒（不计那 0.9 秒即上一条的 15.5–20.2 秒），本段的新文件 ≤ 8 秒；X = 8 秒时上界 S + 8 × f = 1759.3–1772.9 秒，约 29 分 19–33 秒；估算 1422 + 8 × (1.67–2.25) / 2 = 1428.7–1431.0 秒，约 23 分 49–51 秒。慢的 runner（× 1.31）乘在 1422 秒上约 31 分 03 秒，若照这次相对校准点的 1.355–1.357 倍则约 32 分 07–10 秒；都是观察、不是 spec 的停下条件（负责人已知 e2e 作业的结构迟早要他决定）。
- 余量的又一个数据（M6/P7c 的 T10 检查点，2026-11-16）：分支 CI 43742488275（`d96ed7b2`）attempt 1 全绿、没有重跑，e2e 作业 132902462743 用了 19 分 06 秒 = 1146 秒（17:55:04Z–18:14:10Z）；那时本段的页面故事还没有加入，既有故事一行不改，但命令面板、快捷操作与三页的移动端页头已改（本机门禁 477 条中 476 条通过，F11 (page) [s3] 以 M6-M8-07 的签名失败、只重跑该文件即过；chromium 的 `stories/planning/` 61 条合计 264.0 秒，9 个工作进程、只作对照）。比 P7c 的基线 1422 秒（上一条的 main CI 43706438924）少 276 秒（约 0.81 倍），比 P7b 的检查点 20 分 25 秒少 79 秒，作为 P7c 推送前估算 e2e 时长的校准点（[P7c spec](../../M6-cycles-modules/specs/P7c-palette-mobile.md) §2.12）。本段的新页面故事 G19 (page) 的工作项一侧（`e2e/stories/planning/g19-work-item-side-page.spec.ts` 的三条：命令面板、移动端页头、“添加已有”的焦点）本机 chromium 单独跑合计 4.0 秒（2.2、1.2、0.6 秒；预算 8 秒），在三种引擎的页面配置里（3 个工作进程）chromium 4.6 秒、firefox 7.0 秒、webkit 6.4 秒；chromium `--repeat-each=3` 通过，各故事取最大是 2.3 + 1.3 + 1.0 = 4.6 秒。P7c 的 T15 推送之前照 spec §2.12 以这些数（X 取 4.6 秒）算估算、上界与校准；估算与实测见下面三条（[P7c 阶段记录](../../M6-cycles-modules/reviews/P7c-palette-mobile-review.md) §5、§7，P8 的 T1 回填）。
- **web 作业碰到 15 分钟的期限**（M6/P7c 的最终分支 CI，2026-11-16）：最终 HEAD `cf215555` 的分支 CI 43759328881 attempt 1 的 web 作业 132950361275 每一步都成功（Lint 92 秒、Unit tests 762 秒），在收尾的 Post Cache 一步碰到作业的 15 分钟期限被取消（20:39:49Z–20:54:51Z，902 秒），e2e 因此跳过，server 两路通过。web 作业随 runner 在 516–870 秒之间（P7c 的检查点 581 秒，P7b 的分支 854 秒、main 870 秒），同一步 Lint 与测试数无关、在 56–92 秒之间，这次的 92 秒与 P7b 那两次慢的 runner 相同；P7c 多出的约 140 个单测在慢的 runner 上约多 30 秒。判为慢的 runner 越过只剩约 3% 余量的期限（870 秒对 900 秒），只重跑失败的作业一次，attempt 2 的 web 作业 132955284177 用了 602 秒。web 作业的期限与 e2e 的作业结构一样是负责人的开放问题（[P8 spec](../../M6-cycles-modules/specs/P8-final-acceptance.md) §5），P7c、P8 都不改；M8 复核 CI 的预算时一并复核。
- 余量的又一个数据（M6/P7c 合入，2026-11-16，P8 的 T1 以公开的 jobs 接口与作业日志核对）：推送前的估算（P7c 的 T15，X = 4.6 秒）：基线 1422 秒加 4.6 秒 × 1.67–2.25、两个工作进程摊开，约 23 分 46–47 秒；上界 S + X × f（S = 1730.48–1740.13 秒）约 29 分 07–19 秒；校准点（上文 P7c 的 T10 检查点的 1146 秒加同一增量）约 19 分 10–11 秒；估算与上界都低于 30 分钟，所以推送。最终 HEAD `cf215555` 的分支 CI 43759328881 attempt 2（上一条）的 e2e 作业 132958271577 用了 24 分 42 秒 = 1482 秒（21:07:18Z–21:32:00Z，480 过、故事 22.3 分钟）：估算的约 1.04 倍、上界的约 0.84–0.85 倍、校准点的约 1.29 倍。合入后的 main CI 43764306705（main `ea971606`，树与 `cf215555` 相同）attempt 1 的 e2e 以 C7 失败一条（下一条），只重跑失败的作业，attempt 2 全绿：e2e 作业 132979921078 用了 27 分 44 秒 = 1664 秒（22:21:26Z–22:49:10Z，480 过、故事 25.6 分钟），估算的约 1.17 倍、上界的约 0.95 倍，低于 30 分钟；上文 43702923005 的 1670 秒仍是至今最慢的 e2e 作业。同一棵树、同样的 480 条，main 的作业比分支 CI 的慢约 1.12 倍（Playwright 汇总的故事时长 25.6 对 22.3 分钟，约 1.15 倍），是 runner 的速度差异。P7c 的三条新页面故事（G19 (page) 的工作项一侧）在分支 CI 上 13.1 秒、main CI 的 attempt 2 上 16.5 秒（attempt 1 16.8 秒），是本机 4.6 秒的约 2.8、3.6 倍（attempt 1 约 3.7 倍），没有超过上界所取的 f = 4.1。
- **C7 的再现**（M6/P7c 合入）：main CI 43764306705（main `ea971606`）attempt 1 的 e2e 作业 132970815083（21:50:35Z–22:18:22Z，27 分 47 秒）中 480 条有 479 过，C7 在同一个故事（`collaboration-interactions.spec.ts:422`，键盘选主题）的第二次 `reopenDuringExit("Meta+k")`（`:474`）失败：其中的 `selectCommand`（`:465`）看到选项先可见，约 30 毫秒后整个面板已卸下，`getAttribute` 等到故事的 30 秒期限——测试注入的 30 秒慢退场在一次反转之后的再一次退场没有挡住面板，与上文 P6a 合入那次同在第二次 `Meta+k`（那次面板还在、已 `hidden`、`data-leave` 为 null）。P7c 对命令面板的宿主只换了过滤函数（没有 keywords 的行与原来等价）与项的 key，没有动 Headless UI 的过渡，web、server 两路通过。照本条只重跑失败的作业一次，attempt 2 通过（上一条）。trace 的摘要与作业日志在 P7c 的归档 `.superpowers/sdd/M6-P7c-integration-20261116`（`ci-evidence/ci-main-43764306705/`、`ci-main/`）。
- 余量的又一个数据（M6/P8，2026-11-16）：P8 不给默认套件加故事（[P8 spec](../../M6-cycles-modules/specs/P8-final-acceptance.md) §3 第 1 条），条数照旧 480，矩阵专用的页面核对放在默认配置不跑的 `e2e/acceptance/`（`test:m6-acceptance`，[M6-M8-17](#M6-M8-17)），所以 e2e 的余量不变（P8 的 T1 不再重算）。检查点 `d94a0999` 的本机门禁（`t11-gate`）七项一次全过，build+e2e 480 条 254 秒、没有间歇失败；推送前的估算以 P7c 的 main CI attempt 2 的 1664 秒为基线、至今最慢的 1670 秒为上界，都低于 30 分钟。
- 余量的又一个数据（M6/P8 的 T11 检查点，2026-11-16）：分支 CI 43831815008（`d94a0999`）attempt 1 全绿、没有重跑，e2e 作业 133185272705 用了 20 分 36 秒 = 1236 秒（08:45:15Z–09:05:51Z，480 条），web 作业 685 秒，server 两路 865、533 秒。P8 不给默认套件加故事（480 条不变），推送前的估算取基线 1664 秒、上界取至今最慢的 1670 秒，都低于 30 分钟。
- 余量的又一个数据（M6/P8 合入，2026-11-16，M6 收尾的第一个 Task 以 P8 的归档与公开的 jobs 接口核对）：最终候选 `0ef66139` 的本机七项之后只落下一个记录提交 `352740ac`（只差五个文档文件，产品代码、测试与 e2e 未变），推送前的估算取检查点的 1236 秒、上界取至今最慢的 1670 秒，都低于 30 分钟。最终 HEAD `352740ac` 的分支 CI 43846387746 attempt 1 全绿、没有重跑：e2e 作业 133234600667 用了 20 分 34 秒 = 1234 秒（10:24:01Z–10:44:35Z），web 作业 595 秒，server 两路 825、630 秒。合入后的 main CI 43851660778（main `c4cd1880`，树与 `352740ac` 相同）attempt 1 全绿、没有重跑：e2e 作业 133252389009 用了 20 分 58 秒 = 1258 秒（10:59:12Z–11:20:10Z），web 作业 602 秒，server 两路 829、564 秒。P8 的三次 CI（检查点、最终 HEAD、main）都没有间歇失败，web 作业 685、595、602 秒，都低于 15 分钟的期限；上文 43702923005 的 1670 秒仍是至今最慢的 e2e 作业（[P8 阶段记录](../../M6-cycles-modules/reviews/P8-final-acceptance-review.md) §5、§7）。
- **web 作业又一次碰到 15 分钟的期限**（M6 收尾的首次集成，2026-11-16）：main CI 43871234997（main `9440e0fc`）attempt 1 的 web 作业 133312655892 用到 915 秒在期限处被取消，已完成的各步都成功（Lint 91 秒、Unit tests 776 秒），Build 进行到 17 秒；本次合入没有改 web 的代码。照 P7c 的先例只重跑失败的作业，attempt 2 的 web 作业 133321922950 用了 894 秒（离期限只差 6 秒），e2e 133330084017 1137 秒。收尾之前的分支 CI 43865444132 上同一作业 605 秒。web 作业的时长随 runner 在约 595–915 秒之间，慢的 runner 上已没有余量；期限与拆分仍是负责人的开放问题。
- **负责人 2026-11-17 的决定**（M6 最终报告提出的两项）：web 作业的期限由 15 改为 20 分钟；e2e 作业分两片并行（`.github/workflows/ci.yml` 的 `e2e (1)`、`e2e (2)`，`make e2e E2E_SHARD=<n>/2` 交给 Playwright 的 `--shard`，按条数平分，`s3-fixture` 每片各跑一遍），每片的期限仍为 35 分钟。按 P8 的本机时长，两片的故事约 585 与 792 秒（42/58），较长的一片在 CI 上估约 19 分钟。分片之后，“e2e 超过 30 分钟时停下报告”按每一片的作业计。

**何时验收：** M8 的 CI 工作复核每片与 35 分钟的余量（或缓存浏览器的系统依赖、调整片数，任一调整须负责人同意），并查实或修好 C7；在此之前，C7 再失败时按“重跑一次，再不过就停下报告”处理。C12 若再失败，按新的问题定位，不按本条重跑。

<a id="M6-M8-02"></a>
## M6-M8-02 三个模块逐字相同的清理分页器

`planning/app/purge.go` 是 `issue`、`project` 两个模块的同名文件的第三份逐字副本（约 100 行，M6/P1 台账的裁定：模块互不 import，沿用先例）。M6/P1 的整分支审查（[re-55](../../../reviews/re-55-M6P1整体审核.md)）指出，有三份副本时，抽成一个平台包（例如 `platform/purge`）已比重复便宜。P1 不改：它要同时改三个模块与 bootstrap 的清理接线，不属本阶段。M6 其余阶段的模块表仍归 `planning`，不再新增副本。

**何时验收：** M8 的清理工作抽出共用的分页器，三个模块改用它，`TestPurgeScale` 与清理守卫照常通过；或负责人决定保留副本，并在此写明理由。

<a id="M6-M8-03"></a>
## M6-M8-03 e2e 共用的 Postgres 连接余量

M6/P2 的本地 e2e（`make build && make e2e`）两次各有一条因 Postgres 的 `FATAL: sorry, too many clients already`（SQLSTATE 53300）失败，同一输入只重跑这一项即 426 条全过：候选 `254d3255` 上 W6（`e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts:74`）的两次注册拿不到连接而回答 500；候选 `78c7f7d3` 上 W12（`e2e/stories/workspace/w12-reactivate-member.spec.ts:143`）的 `nerve workspaces reactivate-member` 命令连不上数据库。W6 那次本机 Docker 同时运行二十余个其他项目的容器，同一候选的前一轮 `make test` 也有两个包因测试容器起不来而失败（环境失败，重跑通过）。P2 没有新增连接池或任务队列，每个进程的连接上限没有变。

连接预算（按配置核算，未逐个计数）：全部 Playwright 工作进程共用一个 `postgres:18.6` 测试容器（`e2e/fixtures/db.ts`），用默认的 `max_connections`（100）；工作进程数取 Playwright 的默认值（CPU 数的一半），本机 18 核为 9 个，CI 运行器 4 核为 2 个；每个工作进程的 Nerve 服务器连接池上限 10（`server/configs/config.yaml` 的 `database.max_conns`），测试一侧的 pg 连接池默认上限 10，另有 River 的连接。本机 9 ×（10 + 10 + River）远超 100，平时用不满，机器繁忙、各池同时涨到上限时就会用尽；CI 约 2 ×（10 + 10 + River），有余量，分支与 main CI 都没有出现过。M6 加入的模块数据、链接抓取任务与故事里的数据库轮询让本机更容易撞线。证据在 M6/P2 归档的台账（`t15-gate4-build-e2e`、`t15-gate5-build-e2e`）。

M6/P3a 的最终候选 `070762f5` 的本地门禁（`t13-gate1-build-e2e`）又一次撞上：W6（API）“a deleted invitation”一步回答 500，该测试 nerve 日志为 `FATAL: sorry, too many clients already`（53300）；只重跑这一项即 429 条全过（`t13-gate1-build-e2e-2`）。同一候选的分支 CI 不受影响。

M6/P5a 的最终候选 `5b5da6ea` 的本地门禁（`t16-gate3-build-e2e`）再一次：C12 的 LAN HTTP 一例（`e2e/stories/issue/collaboration-interactions.spec.ts:556`）页面上 `GET /api/v0/workspaces/<slug>/states` 回答 500，该测试 nerve 日志为 `FATAL: sorry, too many clients already`（53300）；只重跑这一项即 29 条全过（`t16-gate3-e2e-c12-rerun`）。按 M6-M8-01 对 C12 的要求按新问题定位，查实是本条，不是 M6-M8-01 的间歇失败。

M6/P5b 的检查点（T10，2026-11-11，`00cf6c38672c0a21a63c0dab7026f339d8185f92`）的本地门禁（`t10-gate` 的 build+e2e）又一次：445 条通过、1 条失败，W7 (API)（`e2e/stories/workspace/w7-member-management.spec.ts:44`）在测试一侧的 pg 连接池上报 `error: sorry, too many clients already`（`e2e/fixtures/db.ts:88`，故事第 107 行 `expectWrittenLastBy` 的数据库断言）。P5b 没有改这个故事；同一输入只重跑这一项即通过（`t10-gate-e2e-w7-rerun`，exit 0）。同一检查点的分支 CI 没有跑到故事（见 M6-M8-06），之后 `f4a37007` 的分支 CI 全绿。

M6/P6a 的检查点（T10，2026-11-13，`82dac1efd9dac3fe7b7d99bdcac18410cd12bb0a`）的本地门禁（`t10-gate-build-e2e`）又一次：451 条通过、1 条失败，W6 (API)（`e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts:13`）在测试一侧的 pg 连接池上报 `error: sorry, too many clients already`（`e2e/fixtures/db.ts:88`，经 `e2e/fixtures/assert/identity.ts:307` 的 `countIdentity`，故事第 57 行的数据库断言）。P6a 没有改这个故事；同一输入只重跑这一个文件即 8 条全过（`t10-gate-rerun-w6`，exit 0）。同一检查点的分支 CI 43418051413 全绿。

M6/P6a 的最终候选（T16，2026-11-14，`1f41c8ca78a3607174ba6bb3095a58a5dce87b2a`）的本地门禁（`f-final-gate-build-e2e`）又一次：461 条通过、1 条失败，W9（`e2e/stories/workspace/w9-deactivation.spec.ts:30`）第一次 `refusedBoth`（故事第 96 行，经第 91 行与 `e2e/fixtures/users.ts:44` 的 `nerveUsersFails`）的 `nerve users deactivate` 命令连不上数据库，报 `FATAL: sorry, too many clients already`（SQLSTATE 53300），与 P2 那次 W12 的命令一侧同类。P6a 没有改这个故事、`server/` 与命令；同一输入只重跑这一个文件即 7 条全过（`f-final-gate-rerun-w9`，exit 0）。最终 HEAD 的分支 CI 由 P6b 的 T1 回填到 P6a 阶段记录 §7。

M6/P7a 的检查点（T9，2026-11-14，`f1646370f1396b755be0a03c5326f7d7e8a8d4fc`）的本地门禁（`t9-gate` 的 build+e2e）又一次，这次两条同时：470 条通过、2 条失败。W6 (API)（`e2e/stories/workspace/w6-sign-up-by-invitation.spec.ts:13`）在 `e2e/fixtures/db.ts:88` 的 `countIdentity` 查询拿不到连接；W1 (page)（`e2e/stories/workspace/w1-create-workspace.spec.ts:78`）启动的第二个 nerve（关掉创建工作区的那个）自启动起 `GET /api/v0/me` 与 River 都连不上数据库（53300，见该测试的 nerve 日志），页面因此是 “Cannot reach the server for now” 而不是 “Creating workspaces is switched off”。P7a 没有改这两个故事、工作区的创建与注册；同一输入只重跑这两个文件即 14 条全过（`t9-gate-e2e-rerun-w1-w6`，exit 0，含 s3 夹具的前置项目）。同一检查点的分支 CI 43573631430 全绿。

M6/P7b 的 T17（2026-11-14，`93229770c6f02cd9af54ed09c108c839e7fe6a08`）的本地门禁（`t17-gate` 的 build+e2e）又一次：477 条中 474 条通过、3 条失败，其中 W12（`e2e/stories/workspace/w12-reactivate-member.spec.ts:143`，与 P2 那次同处）的 `nerve workspaces reactivate-member` 命令连不上数据库（53300）；同一时刻相邻的 C8 command-url（`e2e/stories/issue/collaboration-interactions.spec.ts:556`，即上文 P5a 那次记作 C12 的 LAN HTTP 一例的同一行）详情页 5 秒内没有出现标题的输入框（`e2e/fixtures/issue-detail-page.ts:22`），它的附件被随后的一次运行覆盖、nerve 日志已不在，签名无从核实，记为“很可能是本条、未核实”（P5a 时同一行查实是本条）；第三条是 F11，见 M6-M8-07。P7b 没有改这些故事、工作区的命令与 e2e 的数据库夹具；只重跑失败的三个文件一次即 42 条全过（`t17-gate-e2e-failed-rerun`，exit 0）。同一阶段的最终候选（2026-11-15，`16c52f418cdd75d76b6a26eb365b7eaf704ff46f`，`f-final-build-e2e`）再一次：476 条通过、1 条失败，W2 (API)（`e2e/stories/workspace/w2-landing.spec.ts:40`）在测试一侧的 pg 连接池上报 `error: sorry, too many clients already`（`e2e/fixtures/db.ts:88`，经 `e2e/fixtures/assert/workspace.ts` 的 `expectWorkspaceDeleted`（`:282` 调 `:249` 的查询），故事第 89 行），失败之后的 `pg_dump` 也连不上；只重跑这一个文件即 7 条全过（`f-final-e2e-w2-rerun`，exit 0）。教训：门禁的 e2e 失败时先把 `e2e/test-results` 拷进台账，再跑下一个配置。

M6/P7c 的 T15 第一次最终门禁（2026-11-16，`5087b09ee6191453ddf67b8c449109894daee459`，`t15-gate` 的 build+e2e）又一次：480 条中 479 条通过、1 条失败，S1（`e2e/stories/smoke/s1-server-ready.spec.ts:12`）的 `nerve migrate status` 连数据库（`e2e_w1`）时收到 `FATAL: sorry, too many clients already`（SQLSTATE 53300），失败之后的 `pg_dump` 也连不上；`e2e/test-results` 先存进台账（`t15-gate-test-results`）。P7c 没有改这个故事、`server/` 与 e2e 的数据库夹具；只重跑这一个文件即 6 条全过（`t15-gate-e2e-s1-rerun`，exit 0）。

M6/P8 的 T15 最终候选（2026-11-16，`0ef661393be56c1fc0d88f1c65ca6314f430e6c2`，`t15-final` 的 build+e2e）又一次：480 条中 479 条通过，P1 (page)（`e2e/stories/project/p1-create-project-page.spec.ts:63`）在 `expectProjectCreated`（`e2e/fixtures/assert/project.ts:57`）经 `e2e/fixtures/db.ts:88` 的查询收到 `sorry, too many clients already`，失败之后的 `pg_dump` 也连不上；只重跑该文件即过（`t15-final-rerun-p1`）。同阶段的检查点（`t11-gate`）与最终矩阵的默认套件（`t12-default`）没有撞上（[P8 阶段记录](../../M6-cycles-modules/reviews/P8-final-acceptance-review.md) §4.3、§5）。

**何时验收：** M8 的 CI 工作核算 e2e 的连接预算（工作进程数 × 每进程的连接上限 + 余量），按结果给测试容器设 `max_connections` 或收紧 e2e 服务器的 `database.max_conns`，并在 CI 与本地都留有余量；或负责人决定维持现状，并在此写明理由。

<a id="M6-M8-04"></a>
## M6-M8-04 server 作业：Go 测试分两路并行，lint 工具每次在线下载

- `internal/bootstrap` 包把数百个组合测试串行跑在一个测试二进制里，受 `go test -timeout=20m` 的整包期限约束（`Makefile` 的 `GO_TEST_TIMEOUT`）。[M5-closeout](M5-closeout.md) 记它在 CI 上为 756–978 秒；到 M6/P2，最近 12 次 CI 为 919–1185 秒（M6/P1、P2 各加了测试；本地约 840–860 秒，CI 运行器慢约 15–40%）。修复分支 `codex/m6-p2-ci` 的分支 CI 42413098044 中它超过 1200 秒被终止（`panic: test timed out after 20m0s`），修复分支本身没有让它在本地变慢（修复前 840–855 秒，修复后 833–848 秒）。
- 负责人 2026-11-08 批准把 CI 的 `server` 任务按矩阵分两路（`.github/workflows/ci.yml`，`make test-shard`，`tools/go-test-shard.sh`）：bootstrap 包的测试按名字排序后轮流分到两路，其他包和 `server/tools` 只在第 0 路；生成物检查与 lint 只在第 0 路。两路合起来与 `make test` 相同，脚本核对每路的 `-run` 模式恰好选中分到的测试。整包期限 20 分钟和 `server` 任务期限 25 分钟都没有改。按本地逐个测试的计时（644 个测试共 831 秒），两半分别约 450 与 382 秒。
- 本地 `make test` 照旧整包串行运行。
- 分路后 CI 报告的检查名是 `server (0)`、`server (1)`，不再有 `server`；按 [M0-P6 交接](M0-P6-release-notes.md) 设分支保护的必须检查时以此为准（那里建议的汇总任务也要依赖两路）。
- **lint 工具每次从 GitHub 下载**（M6/P3a 的观察）：`server (0)` 的 Lint 一步是 `make lint-go`，它先跑 `make tools`；CI 的运行器上没有 `./bin`，`tools` 目标每次都经 `raw.githubusercontent.com` 上的 `install.sh` 从 GitHub 下载锁定版本的 golangci-lint（`Makefile` 的 `GOLANGCI_LINT_VERSION`，现为 2.13.2；工作流只缓存 pnpm 的存储）。M6/P3a 检查点 `a37bc6370ae651118bd9cb6204279679fdd5031c` 的分支 CI 42478883539 attempt 1 中，`server (0)`（作业 128726250098）在这次下载时收到 GitHub 的 HTTP 504 而失败，lint 没有开始，与代码无关；`server (1)` 与 `web` 通过，`e2e` 随之跳过。根因是外部下载故障，只重跑失败的作业一次，attempt 2 的 `server (0)` 通过（Lint 一步 84 秒）。即 server 作业的每次运行都依赖一次在线的 GitHub 下载；按版本缓存锁定的二进制（例如以 `GOLANGCI_LINT_VERSION` 为键的 Actions 缓存）可去掉这一外部依赖。

**何时验收：** M8 的 CI 工作复核每路的余量（bootstrap 每半与 20 分钟整包期限、各路与 25 分钟任务期限）；需要时增加路数，或把 bootstrap 的测试改为可并行运行。lint 工具不再每次在线下载（例如缓存锁定版本的二进制），或负责人决定维持现状并在此写明理由。改路数、期限、作业结构或缓存，任何 CI 的改动都须负责人同意。

<a id="M6-M8-05"></a>
## M6-M8-05 本机测试容器的主机端口与 macOS 的临时端口冲突

M6/P5a 的最终候选 `5b5da6ea` 上，本机 `make test` 连续两次失败，各有包在 testcontainers 启动容器时等映射端口约 60 秒到期限（`wait until ready: mapped port … get state: GET /containers/<id>/json: context deadline exceeded`），容器内的服务已就绪：`t16-gate2` 是 `asset/adapter/s3`，`t16-gate3` 是 `platform/postgres`（16 例）与 `migrations`。Docker Desktop 的 `~/Library/Containers/com.docker.docker/Data/log/host/com.docker.backend.log` 在这些时刻记有该容器开端口转发失败：“listen tcp 0.0.0.0:56932（56934）: bind: address already in use”。

根因：Docker Desktop 的 VM 从自己的 `ip_local_port_range`（55000–65535）给容器分配随机主机端口，再在 macOS 上监听同一个端口；macOS 自己的临时端口（`net.inet.ip.portrange.first`/`last`）是 49152–65535，两者重叠，分到的端口可能已被本机的出站连接或监听占用。`make test` 并行启动大量容器、开大量连接，撞上的机会随之增加。同一警告此前只在 2026-11-10 的两个时刻出现；M6/P2（`254d3255`）与 M6/P3b（t14-gate1）的“Docker API 超时、容器拿不到映射端口”是同一签名。Linux 的 CI 没有这一层端口转发，不受影响。

负责人 2026-11-11 决定：P5a（没有 Go 改动）的服务端测试以分支 CI 为准，不改本机设置，不改测试代码，交 M8；同日延续到 P5b–P7 中不改 `server/`、`api/` 的阶段（本机 `make test` 因这一签名失败时记录，以分支 CI 的服务端两路为准）。

M6/P2 的根因修复（`server/internal/platform/postgres/pgtest/hostport.go`、`server/internal/platform/s3test/hostport.go` 的 `startPublished`/`hostPortTaken`）只认 Docker 拒绝启动容器的那一种（“programming external connectivity … address already in use”，Linux 的 Docker 与 Docker Desktop 的 VM 内分配冲突时）；Docker Desktop 在 macOS 一侧转发失败时容器照常启动，错误只表现为等映射端口超时，这层重试认不出。

M6/P5b 的 T13、T14（2026-11-12）在本机跑 e2e 的变异时，e2e 的全局准备（`e2e/global-setup.ts:13`）有三次因 testcontainers 等容器的端口映射到主机超时而失败（“Timed out after 10000ms while waiting for container ports to be bound to the host”，其中 T13 的一次在启动 reaper 时）：`t13-e2e-mut-archived-writable`、`t14-e2e-mut-create-for-guest-2`、`t14-e2e-mut-one-generation-2`。一条故事都没有跑，与所测的变异无关；各自重跑（`t13-e2e-mut-archived-writable-2`、`t14-e2e-mut-create-for-guest-3`、`t14-e2e-mut-one-generation-3`）都跑到了故事。签名与上文相同（同一台 Docker Desktop，映射端口迟迟不到主机一侧；台账没有记这几次的 Docker Desktop 后端日志），所以本条不只影响 Go 的 `make test`，也影响 e2e 的全局准备。

M6/P8 的 T8（2026-11-16）开发冻结的规模测试时（项目数临时改为 20 的本地运行，不是收据，证据在本阶段台账的 `t8-dev/`）四次撞上同一签名：Docker Desktop 的 `com.docker.backend.log` 记 “listen tcp 0.0.0.0:55210（55212、55214、55216）: bind: address already in use”，容器内的 postgres 已就绪，testcontainers 等映射端口约 60 秒到期；本机的临时端口离开这一段之后第五次通过。本段收据的运行（`t8-smoke-*`、检查点 `t11-gate` 的 `make test` 等）都没有撞上。最终矩阵的 Go 全量（`t12-go-full`，`GOFLAGS='-p=1 -v'`）撞上一次：bootstrap 包的 `TestImageStorageContract/s3` 启动 S3 测试容器时 `wait until ready: get state … context deadline exceeded`，Docker Desktop 的日志记 08:49:51Z “listen tcp4 127.0.0.1:55236: bind: address already in use”；该包其余各例都过，只重跑这一例即过（`t12-go-rerun-image-storage`）。

**何时验收：** M8 在本机门禁的说明里给出不重叠的做法并验证（例如测试容器助手自己在本机选空闲端口并显式绑定、在端口转发失败时换端口重启容器，或在 README 写明把 macOS 的临时端口范围收到 55000 以下），使本机 `make test` 不再因此失败；或负责人决定维持现状，并在此写明理由。

<a id="M6-M8-06"></a>
## M6-M8-06 e2e 作业每次在线安装浏览器的系统依赖

- e2e 作业的 “Install the Playwright browsers” 一步是 `pnpm exec playwright install --with-deps --only-shell chromium firefox`（`.github/workflows/ci.yml:120-122`）：`--with-deps` 每次经 apt 从 Ubuntu 的镜像在线安装浏览器的系统依赖，工作流只缓存 pnpm 的存储。镜像出故障时，这一步可以吃掉作业的 35 分钟期限（M6-M8-01）。
- 2026-11-11，M6/P5b 检查点 `00cf6c38672c0a21a63c0dab7026f339d8185f92` 的分支 CI 43047769115：web、server (0)、server (1) 成功；e2e 两次都被取消，原因都是 runner 解析不了 `azure.archive.ubuntu.com`，与代码无关：
  - attempt 1（作业 130626787723，15:40:38Z 开始）：apt 自 15:43:38Z 起 “Temporary failure resolving 'azure.archive.ubuntu.com'”，每个包等 20 秒超时；16:12:29Z 起 apt 反复打印 “W: Tried to start delayed item … ocl-icd-libopencl1 … but failed”，一直到作业期限到时被取消（开始后约 40 分钟），一条测试都没有跑。完整日志 2.47 GB，第 640 行之后的 1344 万行都是这一条警告。
  - attempt 2（只重跑失败的作业，作业 130653659545，16:26:19Z–17:01:39Z）：这一步 16:27:05→16:57:02 用了 30 分钟后“成功”（InRelease 在 16:31:54–16:34:01 仍解析失败，包以 35.9 kB/s 取完，“Fetched 48.8 MB in 22min 40s”），Build 53 秒，E2E 一步只剩约 3 分半即到作业期限而被取消（“Running 446 tests using 2 workers”，已跑的都通过、没有失败）。
  - 依规则只重跑一次、不改 CI 的时长或结构；该检查点的代码由下一次推送的分支 CI 覆盖。
- 下一次推送（`f4a3700732eeda348ebf160b6aa8062fb168b1f1`）的分支 CI 43070304124 全绿：这一步 28 秒（镜像已恢复），e2e 作业 130704599852 共 20 分 41 秒（E2E 一步 18 分 49 秒），距 35 分钟期限约 14 分钟。
- 可选的加固：缓存 Playwright 的系统依赖（例如以 Playwright 的版本为键缓存 apt 的包），或改用官方的 Playwright 容器镜像运行 e2e 作业。两者都改作业的结构，须负责人同意。

**何时验收：** M8 的 CI 工作让 e2e 作业不再每次在线安装浏览器的系统依赖（上面任一做法，或负责人同意的其他做法），与 M6-M8-01 的余量复核一起做；或负责人决定维持现状，并在此写明理由。在此之前，这一步再因镜像故障超时，按“环境原因、只重跑一次、记下”处理，重跑仍不过就停下报告。

<a id="M6-M8-07"></a>
## M6-M8-07 编辑器在相邻的事务里重建已有图片的节点，e2e 把中止的内容读取记为失败

M6/P5b 的 T16（2026-11-12）在 `8b11bc6381623717663ba986da74710668719494` 上串行跑本地门禁（`gates.sh t16-gate`）：build+e2e 一步 452 条通过、1 条失败——M5 的 F11 (page) [s3]（`e2e/stories/files/f11-editor-images-page.spec.ts`，失败于 `e2e/fixtures/content-files-page.ts:225` 的 finish）：页面的 `apiFailures` 多出一条 `net::ERR_ABORTED GET /api/v0/assets/{复制品}/content`。

- 经过（本阶段台账记的 trace 分析，证据在台账的 `research/t16-f11-s3-failure/trace-summary.txt`，同目录有 `trace.zip`）：两次“被拒的粘贴”（伪造的标识、源不可读）之后的撤销，各在撤销后 6–9 ms 让编辑器把正文里已有的复制品图片重读一遍：先读资产的元数据、再读内容（trace 的单调时间：撤销 137.287 s → 137.295 s / 137.309 s；撤销 137.385 s → 137.390 s / 137.409 s，元数据都 200）；第二遍的内容读取发出 9.8 ms 后未答即被 Chromium 中止——此时故事已开始下一步（普通文件的拖放与粘贴），发起它的图片元素随即被替换或移除，trace 没有记下是哪一个事务。
- 再现：同一阶段最终候选 `ebeb2cab699e15fb7cae13b9a611189c0c0c2cc8` 的门禁（`t16-final-gate`）里 F11 (page) [local] 以同一签名失败一次，过程相同（第二次撤销 26.539 s → 元数据 26.544 s、内容 26.561 s，内容读取在故事粘贴普通文本 0.5 ms 后发出、11.3 ms 后中止；证据在台账的 `research/t16-final-f11-local-failure/`）；只重跑该文件一次即通过（`t16-final-gate-e2e-f11-rerun`）。T16 的两次完整门禁都出现、分支 CI 与单独重跑从未出现。把内容读取经路由延迟 150 ms 并不能复现（`t16-red-f11-undo-race` 通过），所以不能简单归为“图片未载入完、下一次粘贴把它中止”，故事未改。`e2e/fixtures/browser.ts` 的 watch 把未答而中止的 API 请求记为失败。
- 又一次再现（M6/P6b 的 T10 检查点，2026-11-13，`3a5c3cf6c3c6651052395d830ef22d615131eeb0`）：本地门禁的 build+e2e（`t10-gate-build-e2e`）461 条通过、1 条失败，F11 (page) [local] 以同一签名失败（`e2e/fixtures/content-files-page.ts:225` 的 finish，`apiFailures` 多出一条 `net::ERR_ABORTED GET /api/v0/assets/{复制品}/content`）；P6b 没有改编辑器、资产、详情页与 F11 用到的夹具；只重跑该文件一次即 7 条全过（`t10-gate-rerun-f11`，exit 0），同一检查点的分支 CI 43517662603 全绿。照下文的验收语记下。
- 又一次再现（M6/P7b 的 T17，2026-11-14，`93229770c6f02cd9af54ed09c108c839e7fe6a08`）：本地门禁的 build+e2e（`t17-gate-build-e2e`）中 F11 (page) [s3] 以同一签名失败（`e2e/fixtures/content-files-page.ts:225` 的 finish，`apiFailures` 多出一条 `net::ERR_ABORTED GET /api/v0/assets/{复制品}/content`）；P7b 没有改编辑器、资产、详情页与 F11 用到的夹具；与同一运行中以 M6-M8-03 的签名失败的两个文件一起只重跑一次即过（`t17-gate-e2e-failed-rerun`，三个文件 42 条，exit 0）。照下文的验收语记下。
- 又一次再现（M6/P7c 的 T10 检查点，2026-11-16，`d96ed7b2443a4314339887f0dfd9ddb15ae5349e`）：本地门禁的 build+e2e（`t10-gate-build-e2e`）477 条中 476 条通过，F11 (page) [s3] 以同一签名失败（`e2e/fixtures/content-files-page.ts:225` 的 finish，`apiFailures` 多出一条 `net::ERR_ABORTED GET /api/v0/assets/{复制品}/content`），同一运行里 F11 (page) [local] 通过；不是 M6-M8-03 的签名（没有 53300 或 too many clients），没有别的失败；P7c 没有改编辑器、资产、详情页与 F11 用到的夹具；只重跑该文件一次即 7 条全过（`t10-gate-e2e-f11-rerun`：F11 的 [local] 与 [s3]，连同 s3 夹具的自检 5 例），同一检查点的分支 CI 43742488275 全绿。trace、截图与 s3 的数据库、nerve 日志随本阶段的台账归档（`t10-gate-test-results/`）。照下文的验收语记下。
- 范围：P5b 没有改编辑器、资产与详情页的代码，也没有改 F11 用到的夹具（`browser.ts` 只多导出 `consoleProbe`，watch 不变）；同一故事此前在本机与 CI 多次通过。只重跑该故事文件一次即通过（`t16-gate-e2e-f11-rerun`，exit 0：F11 的 [local] 与 [s3] 都过，连同 s3 夹具的自检共 7 例）。
- 待查：
  - 为什么相邻的事务会让已有图片的节点重建、重读——用户看到的是图片重新载入；
  - watch 是否应区别对待发起它的元素已被移除而中止的请求。

**何时验收：** M8 查明重建的原因，修正编辑器（或写明这是预期的行为），并据此决定 watch 对这类中止请求的处理；或负责人决定维持现状，并在此写明理由。在此之前，F11 再因这一签名失败，按“已查明原因、与所在阶段无关、只重跑该文件一次、记下”处理，重跑仍不过就停下报告。

<a id="M6-M8-08"></a>
## M6-M8-08 工作项根 store 超过手写文件的行数上限

观察（re-67 M10，M6/P5b 的整分支审查）：`web/apps/web/core/store/issue/root.store.ts` 有 615 行，超过手写文件约 400 行的上限。它是 M4 的文件，P5b 的基线 `0b8efcd701edf7e73a37d5bb82ad176a7795b910` 上已有 593 行；P5b 加了 22 行（迭代页的偏好筛选、列表与优先级页签三个 store 的声明与构造，`fields()` 的 `cycle_id`、`module_ids`，创建与更新放行 `cycle_id`）。P7 还要为迭代下拉框与“添加已有”往里加写入。

**何时验收：** P7 往里加写入之前先把它按职责拆开，否则由 M8 拆，拆后不超过约 400 行；或负责人决定维持现状，并在此写明理由。

**处理结果（M6/P6b）**：P6a 的 R16 把拆分提前到 P6b，P6b 在往根上加模块页的任何东西之前按职责拆开（[P6b spec](../../M6-cycles-modules/specs/P6b-module-list.md) §2.1、R1，经预检 [re-70](../../../reviews/re-70-M6P6b预检.md)）：

- 拆法：写的一侧整体移到写入单元 `IssueWrites`（`web/apps/web/core/store/issue/issue-writes.ts`，根以 `readonly writes` 建立并持有）——每个工作项一条的写队列与 `enqueueIssueChanges`、拖拽的队列、在途的写与 `pendingIDs`、确认的流水线（原 127 行的 `write()`）、`createIssue`/`confirmCreation`/`updateIssue`/`deleteIssue`/`archiveIssue`/`unarchiveIssue`；创建与更新的请求体（白名单与 `skip_activity` 的规则）和成员的快照写成纯函数（`issue-write-fields.ts`）。根留下组成、router 与目录的 getter、一代的身份与退役、读的凭据与可见性、列表与资源的登记和 `observeRead`。写入单元对根只有 `import type` 的窄接口（`Pick<IssueRootStore, …>`，根在构造时把自己传入），没有值的循环导入。调用方改经写入单元：产品代码 12 个文件 18 处、测试 21 个文件 83 处（测试只改接收者、断言不改），合入的状态里根上没有这些方法的转发。M4 的 router 退路照原样搬到写入单元（`issue-writes.ts:261`），P7 给 `updateIssue` 明确的范围、去掉它之后再把写入单元加入 `project-address-route`。
- 行数：拆前根 615 行；拆后（`691302633b51d4b35e645efe46b0ec862824a905`）根 298 行、写入单元 300 行、请求体 70 行；本段加入模块页的偏好与列表两个 store、`module_ids` 的白名单之后（`61e6075e5afd7872317020914a22a1b4e2d2fd94`）根 320 行、写入单元 300 行、请求体 75 行，都在约 400 行之内。
- 证据：提交 `c331aa9d5c044ec0c878a82d364bdde9c2111281`（拆出写入单元，产品代码改经它）、`691302633b51d4b35e645efe46b0ec862824a905`（测试改经写入单元，删去过渡的转发）；拆分前后受影响的 21 个文件 509 例与相关的 14 个文件 164 例逐个同数，`make test-web` 同数；白名单与在途的写各有能让测试失败的变异；M4 起没有能失败的测试的 `skip_activity` 规则由新单测 `issue-write-fields.test.ts` 钉住（P6b 台账 T2、T3）。
- 验收的结论：本条的条件（P7 往里加写入之前拆开，拆后不超过约 400 行）已由 P6b 满足；P7 的写入（迭代、模块下拉框与“添加已有”）照 P6b spec §5 加在写入单元、不加回根。本条保持 open 到 M6 收尾，收尾时核对 P7 之后根与写入单元仍在上限之内再关闭。

**处理结果（M6/P7a）**：本段往工作项写入加的东西都加在写入单元，没有加回根（[P7a spec](../../M6-cycles-modules/specs/P7a-work-item-planning.md) §2.1、§2.2）：`updateIssue` 的范围必给，M4 起退回 router 工作区的一支删去，写入单元随之加入 `project-address-route`（`6f543e6b`）；按变化写的命令 `changeIssue` 与逐个点名的请求体 `changeBody`（`bbbdaa80`）。行数（`wc -l`）：根 `root.store.ts` 一直是 320 行（本段没有改它）；写入单元 `issue-writes.ts` 300 → 293（T2，删去退路）→ 302（T3，加命令）；请求体 `issue-write-fields.ts` 75 → 105（T3）→ 98（T10 把增减的纯函数 `modulesAfter` 移到 `core/lib/planning-directory.ts`，表单与写入单元共用）；本段最后一个产品提交 `40cef888` 上仍是 320、302、98，都在约 400 行之内（台账 `t2-line-counts`、`t3-line-counts`、`t9-line-counts`、`t15-line-counts`；T16 在合入之前再以 `wc -l` 核对）。P7b、P7c 的写入（拖拽、“添加已有”、命令面板）同样加在写入单元、经 `changeIssue`。本条保持 open 到 M6 收尾。

**处理结果（M6/P7b）**：本段的写都经写入单元，没有加回根，根也不加 getter（[P7b spec](../../M6-cycles-modules/specs/P7b-list-vocabulary.md) §2.2、§2.7、§2.9）：拖拽的 `dropIssue`（`helpers/base-issues.store.ts`）改收变化，在写入单元的 `updateIssue` 轮到时以 `changeBody` 与所持的集合得出请求体；“添加已有”的每一项经写入单元的 `changeIssue`；列表的目录的查找取自规划 store 的 `getCycleById`、`getModuleById`。行数（`wc -l`，台账 `t16-line-counts`，HEAD `55a225c8`）：根 `root.store.ts` 320 → 319（T4：迭代页、模块页的偏好先以局部的 service 常量建立、再与本页的转换交给基类）；写入单元 `issue-writes.ts` 302、请求体 `issue-write-fields.ts` 98，本段没有改；另 T2 按职责拆开的列表绑定 `issue-list-binding.ts` 487 → 286，拆出的读 `issue-list-read.ts` 163、目录 `issue-list-catalog.ts` 148，都在约 400 行之内。T17 在最终候选 `16c52f41` 上以 `wc -l` 再核，这几件的行数同上（修复轮只改了两个分组面板与 `utils.tsx`，收据 `t17-record-line-counts`）。本条保持 open 到 M6 收尾。

**处理结果（M6/P7c）**：本段的写（命令面板的“改迭代”“改模块”）经写入单元的 `updateIssue`，请求体在工作项的写队列轮到时经 `changeBody` 以所持的集合算（`web/apps/web/core/components/power-k/ui/pages/context-based/work-item/writes.ts` 的 `useWorkItemWrites`，在命令面板既有的守卫之内），没有加回根，根也不加 getter；页头的数目的按页的读（`pageTotal`）加在项目页、归档页两个 store，不在根上（[P7c spec](../../M6-cycles-modules/specs/P7c-palette-mobile.md) §2.5、§2.8）。行数（`wc -l`，台账 `t14-line-counts`，HEAD `3b5b0a78`）：根 `root.store.ts` 319、写入单元 `issue-writes.ts` 302、请求体 `issue-write-fields.ts` 98，列表绑定 `issue-list-binding.ts` 286、读 `issue-list-read.ts` 163、目录 `issue-list-catalog.ts` 148，与 P7b 合入时相同（本段没有改这六个文件，`git diff d052213f..3b5b0a78` 对它们为空），都在约 400 行之内。本条保持 open 到 M6 收尾。

**处理结果（M6/P8）**：本段没有改这六个文件（`git diff ea971606..` 对它们为空，T13 时核对）：根 `root.store.ts` 319、写入单元 `issue-writes.ts` 302、请求体 `issue-write-fields.ts` 98、列表绑定 `issue-list-binding.ts` 286、读 `issue-list-read.ts` 163、目录 `issue-list-catalog.ts` 148 行（`wc -l`），都在约 400 行之内。本条的条件已满足，由 M6 收尾关闭。

**处理结果（M6/closeout）**：已核对，关闭。收尾在 `e1cf6f4d` 上以 `wc -l` 再核六个文件（收据 `t2-m6-m8-08-lines`）：根 `web/apps/web/core/store/issue/root.store.ts` 319、写入单元 `issue-writes.ts` 302、请求体 `issue-write-fields.ts` 98、列表绑定 `helpers/issue-list-binding.ts` 286、读 `helpers/issue-list-read.ts` 163、目录 `helpers/issue-list-catalog.ts` 148 行，都在约 400 行之内；`git diff ea971606..e1cf6f4d` 对它们为空（收据 `t2-m6-m8-08-unchanged`），P7 之后没有再改。本条关闭；本文件的其余各条照旧，`status` 仍 open。

<a id="M6-M8-09"></a>
## M6-M8-09 Firefox 在同一浏览器的再次载入后报根预加载的字体未用

观察（M6/P5b T16 的三引擎页面配置与修复轮 F2）：Firefox 在同一浏览器里任何一次再次载入文档之后（`page.reload()` 或再一次 `page.goto`，不论与前一次相隔 1.4 s 还是 13.5 s），约 10 s 报一条 JavaScript Warning：根预加载的 inter 字体 “preloaded with link preload was not used within a few seconds”。看控制台的故事若在这之后还检查，就会失败。G15 (page) 的故事一、二因此改为以页面自己的排序代替重载、以同一人另开的浏览器核对“再次载入后仍在”（修复轮 F2）；G19 的两次重载在装着扣住回答的路由时，探针 12 s 内没有这条警告；G1 的两次重载、G6 的重载、G8 的再次 `goto` 在三引擎配置里目前都通过，原因未逐一探查，同属潜在的间歇（证据在 P5b 台账的 `f-probe-g15s1-reload-a`、`f-probe-g15s1-late-reload`、`f-probe-g19-reloads`、`f2-probe-loaded-anew-12s-b` 收据）。

**何时验收：** M8 查明 Firefox 为什么认定再次载入的文档没用到预加载（例如预加载与 `@font-face` 的属性不一致，或再次载入时字体来自前一个文档的缓存），改正预加载（或写明这是 Firefox 的预期行为并让相关故事避开再次载入）；或负责人决定维持现状，并在此写明理由。

<a id="M6-M8-10"></a>
## M6-M8-10 出现后约一帧内按下关闭的通知不再消失

观察（M6/P6a 的 T13，2026-11-13）：G9 (page) 的故事二在被拒的通知出现约 7 ms 后按下它的关闭（共用帮手 `dismissToasts`），通知一直留着，故事 5 秒后失败（`--repeat-each=3` 三次中两次）。每次按下之前记下的关闭按钮都没有 `inert`、`data-starting-style`、`data-ending-style`。

- 原因（第三方，`@base-ui-components/react` 1.0.0-beta.3，propel 的 `Toast` 用它）：`ToastRoot` 在挂载时与它的 `ResizeObserver` 第一次回报时（约一帧之后）各量一次高度，每次都把这条通知的 `transitionStatus` 写回 `undefined`（`esm/toast/root/ToastRoot.js:146-157` 的 `setHeights`）；关闭把它设为 `ending` 并清掉它的计时器（`esm/toast/provider/ToastProvider.js:118-140` 的 `close`）。两次测量之间按下的关闭被第二次测量撤销，计时器又已清掉，通知不再自己退场。P5a、P5b 的故事在通知出现之后总有别的等待才关闭它，碰不到这一帧。
- 影响：真人在通知出现的一帧之内点到关闭几乎不可能，产品代码没有改。e2e 的 `dismissToasts`（`e2e/fixtures/cycle-pages.ts`）改为按下尚未退场的通知的关闭、2 秒内都消失为止，未消失就再按，至多 10 秒（`expect.toPass`）；退回只按一次的变异 `t13-mut-dismiss-once` 让 G9 故事二三次全败。证据在 P6a 台账的 `t13-probe-g9-toast`、`t13-probe-g9-toast-2` 收据。

**何时验收：** M8 升级 Base UI 时核对这一行为是否仍在（例如在通知出现后的第一帧内程序化地按关闭），仍在就向上游报告或在 propel 的 `Toast` 里绕开，并据此决定 `dismissToasts` 的重按是否还需要；或负责人决定维持现状，并在此写明理由。

<a id="M6-M8-11"></a>
## M6-M8-11 两份相同的 useLocalStorage

观察（[re-69](../../../reviews/re-69-M6P6a整体审核.md) 的观察，M6/P6a 的整分支审查）：web 里有两份行为相同（只差一行多余的重复判断、导出方式与 TODO 注释）的 `useLocalStorage`：`@nerve/hooks` 的 `web/packages/hooks/src/use-local-storage.tsx`（具名导出；第 10 行把 `typeof window === "undefined"` 判断了两次），与应用里 Plane 留下的副本 `web/apps/web/core/hooks/use-local-storage.tsx`（默认导出；它的注释写着 “TODO: Remove this once its callers use useLocalStorage from @nerve/hooks”）。模块一侧两份都用：版式与排序（`core/components/modules/use-module-display.ts`）读 `@nerve/hooks`，详情页与页头的 `module_sidebar_collapsed`、两域共用的进度页签 `core/components/core/sidebar/progress-tabs.tsx` 读应用的副本。副本在 P6a 之前就有，现有读者 8 个文件（迭代的进行中卡片 `cycles/active-cycle/cycle-stats.tsx`，迭代与模块的详情页与页头，`progress-tabs.tsx`，工作区侧栏的 `sidebar-menu-items.tsx` 与 `favorites/favorites-menu.tsx`），`@nerve/hooks` 的读者 6 个。两份读写同一个键时行为相同（都以 `local-storage:<key>` 事件互相通知），所以不是缺陷，只是重复。

**何时验收：** M8 的清理把应用副本的读者改为 `@nerve/hooks` 的 `useLocalStorage` 并删去副本，同时删去留下的 `@nerve/hooks` 一份第 10 行重复的 `|| typeof window === "undefined"`（M7 若先改侧栏的收藏，可顺手改它那一处）；或负责人决定维持现状，并在此写明理由。

<a id="M6-M8-12"></a>
## M6-M8-12 项目页第一次读偏好失败时只有说明、没有重试

观察（M6/P6b 的 T12，[P6b spec](../../M6-cycles-modules/specs/P6b-module-list.md) R4、§5 的 M8）：工作项列表的三页第一次读本页偏好失败时的样子不统一。迭代页、模块页是两域共用的 `PlanningReadFailure`（`web/apps/web/core/components/planning/read-failure.tsx`），说明原因并可重试；项目页（`web/apps/web/core/components/issues/issue-layouts/roots/project-layout-root.tsx:78`）是 M4 起的一行 `role="alert"` 文字，没有重试，要重新载入页面或离开再回来。P6b 只统一了“读到过偏好之后的重读失败”（三页都是正文沿用已读的偏好、上方一行可重试的提示 `PreferencesRereadFailure`）与页头控件出现的时机（三页都在本页偏好读到之后），项目页第一次读失败的那一行只加了“偏好还没有时”的条件，文字与无重试照旧（单测 `roots/preferences-reread.test.tsx` 钉住它照旧）。

**何时验收：** M8 随最终矩阵把项目页第一次读偏好失败改为与迭代页、模块页一致（说明原因并可重试），或写明项目页保持 M4 样子的理由；或负责人决定维持现状，并在此写明理由。

<a id="M6-M8-13"></a>
## M6-M8-13 上方的内容晚到时，打开着的下拉列表不随按钮移动

观察（M6/P7a 的 T14，2026-11-14）：工作项预览里的描述编辑器（懒加载的 `contenteditable`）在预览其余的内容显示之后约 0.22–0.28 秒才挂上，它下面的属性一栏随之整体下移约 40 像素。若在这之前打开属性一栏里的下拉框，列表停在打开时的位置，离按钮约 40 像素：列表由 react-popper 放置（共用的 `CustomSearchSelect`，`web/packages/ui/src/dropdowns/custom-search-select.tsx`），按钮随祖先的布局移动时它不重算位置。

- 发现：G11 (page) 在 m6-pages 全配置的三种引擎运行（`t14-full-pages`）中 117 条有 116 过，WebKit 一例失败：预览的模块列表离按钮 44 像素（`e2e/stories/planning/g11-issue-modules-page.spec.ts` 的 `expectListBesideButton`）。不是 M6-M8-03、M6-M8-07 的签名，没有重跑，查根因：只量位置的探针（WebKit，9 个工作进程）九次全部复现——页面安定时还没有描述的编辑器、按钮在 641 像素处，约 0.25 秒后编辑器高 28 像素、按钮到了 681 像素处（证据在 P7a 台账的 Task 14；探针文件已删）。
- 范围：不是 P7a 引入的。P7a 的迭代、模块下拉框只是共用件的调用方；以同样方式放置列表的下拉框（共用件上的状态、成员、项目，自己用 react-popper 的优先级）在上方的内容晚到时按同一机制都会这样。源头与 [M5-closeout](M5-closeout.md#M5-M8-06) 所记 F16 重连故事的“详情页挂载描述编辑器时的版面位移”相同：描述编辑器晚挂载，下面的内容下移。真人要在预览显示之后约 0.25 秒之内打开下拉框才看得到。
- 故事的处理：G4、G11 在打开列表之前先等页面第一次渲染之后才到的、位于该栏之上的内容（预览等描述的编辑器、上次编辑与创建者一行和过渡结束，详情页等侧栏的 “Created by”，弹窗等描述的编辑器与过渡结束），它们核的是列表在按钮旁，不是页面挂载的时机；之后三种引擎各 `--repeat-each=6` 共 36 条通过，m6-pages 全配置 117 条通过（`t14-full-pages-2`）。产品代码没有改。

**何时验收：** M8 消除晚到的版面位移（例如详情页与预览为描述编辑器预留它的高度，挂上时下面的内容不动），或让共用的下拉列表在按钮移动时更新位置（例如观察按钮的位置，移动时调 react-popper 的 `update`），二者做其一或都做，并在浏览器中核对：预览刚显示时打开属性一栏的下拉框，列表在按钮旁；或负责人决定维持现状，并在此写明理由。在此之前，新故事照 G4、G11 的写法先等上方的内容到齐再打开列表。

**处理结果（M6/P7b）**：本段的两个新页面故事照上面的写法，先等页面的内容到了再打开列表与弹窗：G15 (page) 在列表的组显示（组头的名字照次序到齐）之后才打开显示面板与“加筛选”，表头的菜单在换到表格的偏好答到之后才打开；G4 (page) 在空态的说明显示、G11 (page) 在列表的行显示之后才打开“添加已有”的弹窗。它们不量列表与按钮的相对位置（不用 `expectListBesideButton`），也不打开预览里属性一栏的下拉框。三种引擎各一次与 `--repeat-each=3`（台账 `t14-three-engines`、`t14-repeat3`、`t15-three-engines`、`t15-repeat3`）、m6-pages 全配置三种引擎 126 条（`t15-full-pages`）都通过，本条的现象没有再现。产品代码没有为本条改动。本条保持 open。

**处理结果（M6/P7b 的分支 CI）**：同一机制这次落在详情页、作用在点击上。P7b 推送的 HEAD `5d7adaaa` 的分支 CI 43696983860 attempt 1 中，e2e 作业 132762846784（09:11:46Z–09:37:57Z）477 条有 476 过，M4 的 C4 (page) relation（`e2e/stories/issue/detail-interactions.spec.ts:13`）在 `:54` 的 `options.focus()` 30 秒等不到 “Related target options”。不是 M6-M8-03、-07 的签名，没有重跑，从 trace 查根因：接口的回答都是 200；失败时 “Relations 1” 收着、按钮有焦点；录屏里故事在 `openIssueDetail` 看到标题之后随即点 “Relations 1”，点击开始时还没有描述的编辑器，点击结束之后的一帧里编辑器已挂上，其下的 Attachments、Relations 下移约 28 像素——按下时指针在 Relations 上（焦点移到它），抬起时指针下已是另一元素，按下与抬起不在同一元素上，不触发 click，面板没有展开。P7b 没有改详情页。

- 探针（台账 `ci-probe-*`，探针文件留在台账、不入库）：照抄 C4 与 C8/C12 的 relation 一支，只在打开详情页之前加条件，新旧两版条件相同。只以 CDP 降速（`Emulation.setCPUThrottlingRate` 4、6、8 倍，各 15 例）复现不了：降速同样拖慢 Playwright 在页面里做的可操作性检查，按下与抬起之间只有约 4–20 毫秒，编辑器落不进去（4 倍时 15 例里有 10 例按下时还没有编辑器，仍都点开了）。把按下与抬起之间放宽到 2 秒（click 的 `delay`；CI 上那次点击约 0.3 秒）后，旧故事在 4 倍降速下 15 例有 12 例在同一步失败，恰是按下时还没有编辑器的那 12 例（`ci-probe-old-rate4-press`）；再让编辑器的懒加载块晚 1 秒到（`page.route`），旧故事 15 例全在同一步失败（按下在 Relations 上、其顶 316 像素，抬起时编辑器已在、Relations 顶 344 像素、指针下是 “Attachments 0”，click 落到共同的祖先上，`ci-probe-old-hold`）；新故事在这两种条件下各 15 例全过（`ci-probe-new-rate4-press`、`ci-probe-new`）。
- 故事的处理：C4 与 C8/C12（`collaboration-interactions.spec.ts:556`）的 relation 一支在点 “Relations 1” 之前先等描述的编辑器——`e2e/fixtures/issue-detail-page.ts` 新的 `descriptionEditorMounted`，等 `[id="editor-container-<源>"] [contenteditable="true"]` 可见（I9 等故事等编辑器的写法），只用于这两处；`openIssueDetail` 不改（以只读打开、描述不可编辑的故事也用它）。两个文件里其余打开详情页的分支与故事只以键盘聚焦（与位置无关），或点的是页头的命令框与弹窗里的按钮，不在描述之下，不加等待。两个文件默认配置 48 条（`ci-after-two-files`）与 chromium `--repeat-each=3` 134 条（`ci-repeat3`）通过；多出的等待就是编辑器自己挂载的时间，每条 relation 约 0.25–0.3 秒，两个文件一次合计约 0.8 秒（103.8 → 104.6 秒）。改故事的提交是 `4fe308ce`（P7b 的最终 HEAD），它的分支 CI 43702923005 与合入后的 main CI 43706438924 都全绿（P7c 的 T1 补记）。产品代码没有改，验收语不变。本条保持 open。

<a id="M6-M8-14"></a>
## M6-M8-14 经接口存下按截止日期分组的列表，网页没有这一分组的列

观察（M6/P7b 的 T17，修复轮 F 的限定复审，2026-11-15，[re-75](../../../reviews/re-75-M6P7b整体审核.md) §4 的第三条观察）：nerve 对项目页、迭代页、模块页的列表偏好接受 `group_by: target_date`（看板的 `sub_group_by` 也接受），网页却没有这一分组：分组面板不给它，`getGroupByColumns` 也没有它的列。只能经接口存下（网页自己的面板从不写它），早于 P7b：偏好的显示词自 M4/P10b 起就认得它。

- 发现：限定复审读码时看到偏好的 `show()` 认得它，面板与列都不认得（P7b 台账 Task 17）；没有故事或使用者碰到过。主控随后只读地追了网页实际渲染什么（`16c52f41` 的树，没有运行）。
- 现状：
  - nerve：偏好的显示词由 `ListSettings.Validate` 以 `ListGroupAllowed(…, ProjectList)` 核（`server/internal/modules/issue/app/list_settings.go:49`，子分组 `:52`），项目列表的分组含 `target_date`（`server/internal/modules/issue/domain/listvocab.go:49`，子分组经 `:63-65`）；迭代页、模块页的偏好经同一个 `ListSettings`（`server/internal/modules/planning/app/list_settings.go:21-29`）。接口的分组词 `IssueListGroupField` 列着它（`api/modules/issue.yaml:1883`），偏好的 `group_by` 是字符串（`api/common.yaml:138-141`）。
  - 显示词与查询：三页共用的 `PagePreferencesFilter.show()` 把存下的 `target_date` 认作分组 `target_date`（`web/apps/web/core/store/issue/helpers/page-preferences-filter.store.ts:58`、`:75`、`:158-159`），列表与看板的查询照它发 `group_by=target_date`（`web/apps/web/core/lib/issue-filters.ts:42`、`:264-281`），nerve 按日子分组作答。
  - 列：列表与看板的宿主把它断言成 `GroupByColumnTypes` 交给 `getGroupByColumns`（`web/apps/web/core/components/issues/issue-layouts/list/base-list-root.tsx:77`、`list/default.tsx:89-96`、`kanban/default.tsx:101-121`、`kanban/swimlanes.tsx:287-302`）。`GroupByColumnTypes` 没有 `target_date`（`web/packages/types/src/issues.ts:20-29`），`TIssueGroupByOptions` 有（`web/packages/types/src/view-props.ts:15-26`），`as` 的断言让 tsc 看不见这个差。`getGroupByColumns` 的表没有这一项，从表里取到的 `columns[groupBy]` 是 `undefined`，它照样被调用（`web/apps/web/core/components/issues/issue-layouts/group-columns.tsx:45-56`），渲染时抛 `TypeError`；它回答 `undefined`（于是什么都不渲染）只在工作区、项目或权限缺时（`:33-43`）。
  - 看到的：列表有工作项时，列表与看板（看板的子分组为它时是泳道）的正文由版式的 `LayoutErrorBoundary` 接住（`web/apps/web/core/components/issues/issue-layouts/issue-layout-HOC.tsx:61`、`web/apps/web/core/components/common/layout-error-boundary.tsx:22-60`），只显示 “Something went wrong” 与 “Retry”，控制台记 “Issue layout crashed”；“Retry” 重新渲染、再抛一次。列表为空时显示空态（`issue-layout-HOC.tsx:53`、`:58-59`），不渲染列，看不出问题。表格不分组（`issue-filters.ts:269-270`），日历自己按日子分组、不读存下的 `group_by`（`:267-268`），都不受影响。页头照常：分组面板列出本页的选项而没有一项勾着（`ISSUE_GROUP_BY_OPTIONS` 没有 `target_date`，`web/packages/constants/src/issue/common.ts:61-75`；面板列出“存下的，或本页给的”，`filters/header/display-filters/group-by.tsx:40-42`、`sub-group-by.tsx:50-52`，而存下的这一项不在那张表里），选任一项即改写存下的分组，列表随之恢复。
- 范围：成员页（个人页）把分组规范到它的选项（`web/apps/web/core/store/issue/profile/filter.store.ts:87-97`），归档页的显示存在本机、不经接口，都碰不到。Plane 的同名函数以 `groupByColumnMap[groupBy]?.(…)` 回答 `undefined`、列表什么都不渲染（`plane/apps/web/core/components/issues/issue-layouts/utils.tsx:159`），它的分组选项同样没有 `target_date`；nerve 的写法自 M4 的 `4782fc6f` 起没有 `?.`。P7b 把 `cycle`、`module` 加进了这张表（`89e9b8f8`），`target_date` 是 nerve 接受而表里没有的唯一分组。

**何时验收：** M8 定下其一并在浏览器里核对：网页学会这一分组（`GroupByColumnTypes` 与 `getGroupByColumns` 给按日子的列与“无”，分组面板给出它，或至少照 P7b 的 F1 在存下时列出并勾着）；或 nerve 不再为这三种页的列表偏好接受它（`ListSettings` 拒绝它，已存下的偏好随之处理；这是 `server/`、`api/` 的变化）；或保持现状，并在此写明理由（只能经接口存下、选任一分组即恢复）。不论哪一种，表外的分组都不应让版式在渲染时抛错（例如 `getGroupByColumns` 对表外的分组回答 `undefined`，同 Plane），这一点随之定下。

<a id="M6-M8-15"></a>
## M6-M8-15 项目地址的守卫只按变量名认 router store；列表的正文经 router store 读项目页的偏好

观察（M6/P7c 的 T5，2026-11-16；T14 交 M8）：关键词规则 `project-address-route`（`tools/keywords.json`，M3/P10 立下，M4、M6 各自加入改过的文件）管的是“项目的页面从路由参数读地址，不从 router store 读”。它的内容式认 router store 的只有一式 `\brootStore\.router\b`（另几式是 `currentProjectDetails`、`projectMemberIds`、`currentWorkspace`、不带 slug 与项目的 `allowPermissions` 与几个由 router 驱动的 getter），按变量名认：把根 store 存在别的名字下的件（例如参数名是 `store`，再读 `store.router.projectId`）读 router store 的地址，规则看不见。

- 发现：P7c 的 T5 给命令面板的搜索写新 hook 时以守卫的探针核对（台账 `t5-mut-route-guard`）：参数名为 `store` 时加一行 `void store.router.projectId`，`node tools/keywords.mjs` 照样通过；参数改名为 `rootStore` 之后同一行即命中（`t5-mut-route-guard-2`）。P7c 的新件都以 `rootStore` 命名，避开了它。
- 现状（只读的扫描，台账 `t14-route-guard-scan`，HEAD `3b5b0a78`）：规则的文件表现在管 257 个文件，其中没有一处经 `rootStore` 以外的接收者读 `.router`，也没有解构出 `router`（注释与 `react-router` 的导入不算）——漏洞还没有被用到。文件表之外有经别的名字读 router store 的地址的 store：`web/apps/web/core/store/label.store.ts:114`、`state.store.ts:102`（`this.router.projectId`）、`user/index.ts:181`（`this.store.router.workspaceSlug`）、`workspace/index.ts:101`（`this.router.workspaceSlug`），它们由 router 驱动、本不在规则的范围里。
- 范围：规则由 M3 立下，文件表里有 M3、M4、M6 的件，M7 改视图的页面时还要加入；收紧它是这几个 M 共用的守卫的事，不只是 M6 的，M6 收尾之前没有命中要改，所以交 M8。

同一类的另一种读（M6/P7c 的修复轮，2026-11-16，主控裁定并入本条）：P7c 的整分支审查 [re-77](../../../reviews/re-77-M6P7c整体审核.md) 的 M2 指出，页头的控件经 `usePageIssueFilters`（`web/apps/web/core/hooks/store/use-page-issue-filters.ts:27-28`）→ `projectIssuesFilter.issueFilters`（`core/store/issue/project/filter.store.ts:29-34`，其中 `rootIssueStore.projectId` 即 `rootStore.router.projectId`，`core/store/issue/root.store.ts:66-67`）以 router store 的项目读项目页的偏好，门却按地址读：地址由项目 A 换到 B、B 的偏好已在 store 里时，在 router 落后的那一次提交里显示 A 的布局与显示。这种读守卫管不到：组件只调一个 hook，router store 藏在 store 的 getter 后面，规则的内容式（不论按不按变量名）都看不见它。

- 已修的一处（先例）：P7c 的修复轮 F3（`10ca6ae6`）把页头的 `usePageDisplay`（`core/components/issues/use-page-display.ts`）改为读 `useIssues(storeType).issuesFilter.getIssueFilters(entityId)`，三种 store 同一读法（同 `core/components/issues/issue-layouts/use-planning-page-header.ts:44`），单测以“router store 的项目是 A、地址的是 B、两者的偏好都在”钉住（`core/components/issues/mobile-header-filters.test.tsx`，变异 `f3-mut-router-read` 改回 `usePageIssueFilters` 即失败）。
- 余下的（P7c 台账 F3 的观察，早于 P7c，是 P5b 的读法）：`usePageIssueFilters` 还有 5 个产品读者，在项目页上都经 router store 的项目读本页的偏好，列表本身在那一次提交里照 A 的显示（分组、排序、显示的属性）——四种布局的根 `core/components/issues/issue-layouts/list/base-list-root.tsx:72`、`kanban/base-kanban-root.tsx:80`、`spreadsheet/base-spreadsheet-root.tsx:49`、`calendar/base-calendar-root.tsx:85`，与日历的块 `calendar/issue-block.tsx:56`。归档页的一支同形（`archivedIssuesFilter.issueFilters`，`core/store/issue/archived/filter.store.ts:66-71`）；迭代页、模块页的两支按路由参数的 `cycleId`、`moduleId` 读，不受影响。

**何时验收：** 两件都做完，或负责人对其中一件决定维持现状并在此写明理由：
- 守卫：M8 收紧规则的内容式，让它认任何接收者的 router store（例如任何 `.router` 的成员，加解构出的 `router`），样例里加一条命中（如 `store.router.projectId`）与一条不命中（如 `import { useNavigate } from "react-router";`），`node tools/keywords.mjs` 对那时的文件表无命中（有命中的先改读路由参数）；或维持现状（以命名的约定与审查防）。
- 列表的正文：上面 5 个读者在项目页（与归档页）上按地址的项目读本页的偏好（照 P7c F3 的 `usePageDisplay`：`getIssueFilters(<地址的项目>)`，不经 router store），`usePageIssueFilters` 经 router store 的两支随之去掉（hook 没有读者时删去）；单测钉住“router store 的项目是 A、地址的是 B、两者的偏好都在时，列表照 B 的显示”，并有一个改回经 router store 读即失败的变异。

<a id="M6-M8-16"></a>
## M6-M8-16 G2 页面故事在上海午夜前后的几秒会失败

观察（M6/P5a，[P5a spec](../../M6-cycles-modules/specs/P5a-web-cycles.md) §5 的 M8 一条；P8 写入）：G2 (page)（`e2e/stories/planning/g2-cycle-status-page.spec.ts`）要浏览器的日子与项目（上海，UTC+8）的日子不同，按上海此刻的小时选浏览器的时区（`anotherDayZone`，`:38-48`：上海 18 点之前选 Pago Pago，UTC−11；之后选 Kiritimati，UTC+14），并在故事开始时与读完列表之后各核一次（`:63`：浏览器的日子不是上海的；`:121`：上海的日子在读列表期间没有变）。跑的时候跨过上海的午夜（UTC 16:00），这两处会以明确的消息失败，与产品无关。

**何时验收：** M8 复核间歇项时三选一：照“已知的时间窗——失败的时刻在上海午夜前后——只重跑一次、记下”处理；或改写故事，使它不依赖跑的时刻；或负责人决定维持现状，并在此写明理由。

<a id="M6-M8-17"></a>
## M6-M8-17 默认 CI 不跑的对等与发布前的运行

M6 的页面在三种引擎上的对等、矩阵专用的核对与规模，持续集成都不跑：e2e 作业跑默认配置（`make e2e`：`e2e/stories/`，Chromium，另 A4 的页面在 Firefox），CI 不装 WebKit（`.github/workflows/ci.yml:118-122`）；`make test` 不设规模测试的环境变量，它们默认跳过（[P8 spec](../../M6-cycles-modules/specs/P8-final-acceptance.md) §2.8、§2.11、§3 第 1 条；[M6 设计](../../M6-cycles-modules/M6-design.md) §11 的“对等验收”）。发布前要在发布候选上跑：

- `pnpm --filter @nerve/e2e run test:m6-pages`（`e2e/playwright.m6-pages.config.ts`）：`e2e/stories/planning/` 的 `G<n> (page)` 故事在 Chromium、Firefox、WebKit 各一遍（P8 时 45 条 × 3 = 135 例）；默认套件只在 Chromium 里跑它们。
- `pnpm --filter @nerve/e2e run test:m6-acceptance`（`e2e/playwright.m6-acceptance.config.ts`）：`e2e/acceptance/` 六个文件 11 条 × 三种引擎 = 33 例（焦点轮廓、命令面板的行、新建表单里换项目、归档页的迭代与模块词、模块的开关；M6 收尾补的 M3 交接第 3 节三项：弹窗点外面关上、筛选按钮 Escape 之后的焦点、模块一行右键菜单的项）；默认配置的 `testDir` 是 `stories`，不跑这个目录。
- 三个规模测试（在 `server/` 下，一次一个容器套件；命令见仓库根目录 README 的“M6 的验收运行”）：`TestPurgeScale`（`NERVE_PURGE_SCALE=1`；证据目录 `NERVE_PURGE_SCALE_EVIDENCE`，不给时写进测试的临时目录、跑完即删）、`TestTheDeletionsAtTheScaleTarget`（`NERVE_MEASURE_SCALE=1`；`NERVE_SCALE_EVIDENCE`，不给时写到 `.superpowers/sdd/P9-purge/scale/<时刻>`，即 M4/P9 的台账路径）、`TestTheFreezeAtTheScaleTarget`（`NERVE_MEASURE_SCALE=1`；不给 `NERVE_SCALE_EVIDENCE` 即失败）。P8 在 T12 以 `go test -p=1 -count=1 -v ./internal/bootstrap -run '^<测试名>$'` 各跑一次（期限 30、60、30 分钟）。
- P8 的结果：`t11-m6-pages`（135 例过）、`t11-acceptance-final`（24 例过），T12 的最终矩阵与规模的数字在 [P8 阶段记录](../../M6-cycles-modules/reviews/P8-final-acceptance-review.md)。
- M4 的三引擎并集 `test:m4-cleanup`（`e2e/playwright.m4-cleanup.config.ts`，M4 P16 的组合配置）：P8 的 T12 在 `d94a0999` 上一次跑三种引擎、默认的 worker 数（本机 9 个，`t12-m4-cleanup`），510 例中 507 过；三条时序失败——Firefox 的 I4 (page) 表格按 `sub_issues_count` 换序之后 5 秒内表头没回来、WebKit 的 I11 (page) 回应的 DELETE 5 秒内没有发出、WebKit 的 C12 (review) 子工作项的 Filters 点开之后下拉没有开——在各自的引擎隔离重复 5 次都过（`t12-m4-repeat-i4-firefox`、`t12-m4-repeat-i11-webkit`、`t12-m4-repeat-c12-webkit`），判为三种引擎同时跑时的负载，不是 P8 的回归（[P8 阶段记录](../../M6-cycles-modules/reviews/P8-final-acceptance-review.md) §4.3 的间歇项）。M4 P16 的配方是每个引擎各跑一次、`--workers=2`（[P16-C12-preparation](../../M4-issue-core/reviews/P16-C12-preparation.md) §1 的命令）；发布前照那个配方跑。

**何时验收：** M8 的发布流程把这些运行写进发布前的清单，在发布候选上各跑一次并留下原件；或经负责人同意把其中一部分放进 CI（e2e 的作业结构与期限是负责人的开放问题，见 M6-M8-01）。规模测试不给证据目录时的去处（上面三者各不相同，其一指向旧台账）一并定下。

<a id="M6-M8-18"></a>
## M6-M8-18 自动归档遇到失败的项目即停

观察（M6/P8，[P8 spec](../../M6-cycles-modules/specs/P8-final-acceptance.md) §2.6、§3 第 4 条）：M4 的自动归档 `server/internal/modules/issue/app/auto_archive.go` 的 `Execute` 每个项目一个事务，一个项目的事务失败即返回（`:103` 的 `return total, err`），id 在它之后的项目这一次轮不到，River 的每次重试都在同一处失败，直到那个项目恢复；`ctx` 结束时同样返回（`:46`）。P8 修了同形的冻结任务（`server/internal/modules/planning/app/freeze_cycles.go`，`c1c0fb55`）：一个项目失败只回滚它自己的事务、接着做后面的，跑完以 `errors.Join` 交回（只留前 5 个项目的错误、各带项目 id，其余只计个数），已提交的不再是候选，`ctx` 结束即停并交回此前的失败与 `ctx` 的错误；单测在 `planning/app/freeze_cycles_failures_test.go`。自动归档属 issue 模块，跨模块，P8 不改。

**何时验收：** M8（或更早改 issue 模块自动归档的阶段）照冻结的做法改自动归档，并有同样的单测：中间一个项目失败时它前后的都提交，错误含它的原错误与 id；`ctx` 取消即停且不丢此前的失败；错误有上限。或负责人决定维持现状，并在此写明理由。

<a id="M6-M8-19"></a>
## M6-M8-19 访客在命令面板看到项目设置的命令

观察（M6/P7c 复核，[P7c 阶段记录](../../M6-cycles-modules/reviews/P7c-palette-mobile-review.md) §1；P8 写入，[P8 spec](../../M6-cycles-modules/specs/P8-final-acceptance.md) §3 第 6 条）：命令面板项目一级的“打开项目设置”`open_project_setting` 与“转到项目设置”`nav_project_settings` 只看地址里有工作区与项目（`baseProjectConditions`，`web/apps/web/core/components/power-k/config/navigation/project-commands.ts:235-236`、`:248-249`），不看角色：项目的访客也看得到这两个入口。对照：“转到归档”`nav_project_archives` 以 `hasProjectMemberLevelPermissions` 给（`:213-214`）。这是 M3 的代码，P7c 的特征测试（`power-k/config/navigation/commands.test.tsx`）钉住现状，P8 不改（不是 M6 的领域）；视图、收集箱的同类命令交 M7（[M6-M7-07](../../M7-collaboration/handoffs/M6-closeout.md#M6-M7-07)）。

**何时验收：** M8 的对等验收以 Plane 的行为为准，定访客是否该看到这两个命令：不该，就照“转到归档”的条件给，命令面板与侧栏同一规则，单测钉住访客与成员；该，在此写明理由。

<a id="M6-M8-20"></a>
## M6-M8-20 冻结候选查询的规划器余量

观察（M6/P8 的 T8，[P8 spec](../../M6-cycles-modules/specs/P8-final-acceptance.md) §2.8）：冻结任务的候选查询 `FreezeDueProjects`（`server/internal/modules/planning/adapter/postgres/queries/progress.sql:26`）靠部分索引 `cycles_freeze_due_idx`（`server/migrations/sql/00048_planning_cycles_progress_snapshot.sql:13`）。新的冻结规模测试 `TestTheFreezeAtTheScaleTarget`（`server/internal/bootstrap/planning_freeze_scale_test.go`，`NERVE_MEASURE_SCALE=1`）在 5000 个项目、每项目 40 个两周的迭代（36 个已由一次更早的运行冻结）、2500 个项目各有一个迭代到期的夹具上，以连接池同样的规划设置 `EXPLAIN (ANALYZE, BUFFERS)` store 实际发出的语句，断言计划用到这个索引：

- 冒烟 `t8-smoke-freeze-3`：Bitmap Index Scan on `cycles_freeze_due_idx`，代价 35745.38 对关掉索引扫描时的 48339.92（余量约 26%），估 17757 行、实际 2500，执行 0.748 毫秒、112 个堆页；一次运行冻结 2500 个迭代共 3.227 秒，逐项目的锁持有 p50 0.976、p95 1.109、最大 6.508 毫秒。
- 夹具先用每项目 20 个迭代、16 个冻结（冻结占 80%）时，选中的计划代价 21287.64 对 22085.27，余量只有 3.6%（`t8-smoke-freeze-2`）：规划器把 `progress_snapshot IS NULL` 与 `end_date` 不晚于今天当作互相独立，估计偏高 6–7 倍。冻结的迭代少的年轻部署里规划器会选顺序扫描；那时表小、任务每小时一次，不是性能缺陷。要估计准确需加扩展统计，是产品的迁移，不在 P8（本阶段台账 Task 8 的发现与主控的裁定）。
- 最终代码上的正式运行（T12 的 `t12-freeze-scale`）的计划与数字在 [P8 阶段记录](../../M6-cycles-modules/reviews/P8-final-acceptance-review.md)。
- 一次运行的时长上限（P8 整分支审查 re-79 m5）：P8 起冻结任务逐项目继续（[P8 spec](../../M6-cycles-modules/specs/P8-final-acceptance.md) §2.6），一次运行要走完全部候选；`FreezeCyclesWorker` 没有自己的 `Timeout`，nerve 的 River 配置（`server/internal/platform/jobs/jobs.go:71`）也没设 `JobTimeout`，取 River 的默认 1 分钟（river v0.47.0 `JobTimeoutDefault`）。按规模测试每项目约 1.4 毫秒，一分钟约容 4 万个候选；多数候选都在等锁到各自的期限时，一次运行在一分钟处以 `ctx` 的错误结束（已提交的项目保留，`errors.Join` 带上已记下的失败），下一次接着做。M8 的规模评估一并决定是否给它设自己的 `Timeout`。

**何时验收：** M8 的规模与部署评估在目标规模与年轻部署两种形状下复核这条查询的计划（以 `TestTheFreezeAtTheScaleTarget` 的 EXPLAIN 证据），决定是否加扩展统计（迁移，附加前加后的计划对照）；或负责人决定维持现状，并在此写明理由。

<a id="M6-M8-21"></a>
## M6-M8-21 命令面板打开子菜单时没有选中的行

观察（M6/P8 的 T11，矩阵专用的页面核对 `e2e/acceptance/g19-palette-rows.spec.ts`）：命令面板打开子菜单（如 “Open a cycle”）时没有选中的行——cmdk 保留根列表里所选那一行的 value，子菜单里没有它，第一次按方向键才选中第一行，按 Enter 之前要先按一次方向键。Plane 同；不在 [P8 spec](../../M6-cycles-modules/specs/P8-final-acceptance.md) §2.11 之内，P8 不改（核对先以方向键选中再断言）。

**何时验收：** M8 的对等验收决定是否在打开子菜单时选中第一行，改则以页面核对钉住；或照 Plane 维持，并在此写明理由。

<a id="M6-M8-22"></a>
## M6-M8-22 富筛选的 “Remove filter” 是英文字面量

观察（M6/P8 的 T11）：富筛选条件的移除按钮 `web/apps/web/core/components/rich-filters/filter-item/close-button.tsx:35` 的 `aria-label="Remove filter"` 是写死的英文（Plane 原样，M4 的件），中文界面下读屏也读出英文；与 P8 改为键的三处英文字面量（[P8 spec](../../M6-cycles-modules/specs/P8-final-acceptance.md) §2.3）同一类。P8 只把它的 `focus:outline-none` 换成内缩的焦点环（`ea7580d6`），文字没改。

**何时验收：** M8 的文案收尾把它改读键（en 与 zh-CN 在同一提交，过 `check:sync`），以单测钉住；或负责人决定维持现状，并在此写明理由。
