# 04 下一步与遗留

## 1. 马上要处理的
### 1.1 main 的持续集成第 340 次红了（P3 的合并 `dfc9378b`）
- **现象**：server、web 两个任务通过，e2e 失败：125 个通过、1 个失败。
- **失败的故事**：M3 的 W11 页面版本，`e2e/stories/workspace/w11-guest-bounds.spec.ts:190`。在 `endsOnBack("ControlOrMeta+k")` 里：
  1. `moveWithinApp` 到 acme 的项目页；
  2. 等到"Add Project"可见；
  3. 按 Ctrl+K；
  4. 期望出现一个 `[role="dialog"][aria-modal="true"]`，5 秒内 14 次都是 0 个。
- **与 P3 无关的依据**：
  - P3 没有改这个故事，没有改命令面板、`ModalCore`，也没有改工作区的布局；P3 在前端只改了 `error-messages.ts`、两份 `errors.json` 和生成的 `schema.gen.ts`；
  - 第 336–339 次运行这个故事都通过。
- **本机复现不出来**：
  - 照持续集成的方式（chromium、2 个 worker、`CI=1`），整套 3 遍 372 个、I 系列 5 遍 30 个，全部通过；
  - 用 CDP 把 CPU 降速 4 倍、6 倍跑 W11，5/5 通过；
  - 照 Linux 用 Ctrl（而不是 macOS 的 Meta），5/5 通过；
  - 按键时焦点在 `BODY`。
- **读过的代码**：
  - 面板的开关在 power-K 的 store（`core/store/power-k.store.ts`）；
  - 键盘的监听在 `core/components/power-k/global-shortcuts.tsx`，用 `useEffect` 挂在 `document` 上；
  - 面板的宿主 `projects-app-provider.tsx` 卸载时关掉面板；
  - `ModalCore`（`web/packages/ui/src/modals/modal-core.tsx`）从挂载之后的那次渲染才打开。
  - 没有找到会自己关掉面板的路径。
- **还没有根因。** 需要的证据是第 340 次运行第一次尝试的 `playwright-report` 产物：里面有 trace、按键前后的 DOM 快照、两个页面的截图、nerve 的日志。在 Actions 页面的 Artifacts 下载，要负责人的权限。
- **建议**：
  1. 负责人下载这份报告，放到本机给你；
  2. 从 trace 找根因，在新分支上修（是测试的等待条件不对就改测试，是应用的竞态就改应用），合并；
  3. 若负责人选择先在页面上"Re-run failed jobs"，重跑通过只说明它是偶发的，根因仍要查，记进 M4 收尾的"不稳定的故事修正"（收尾第 2 条）。
- **注意**：本交接的提交推送之后，`main` 会再跑一次持续集成，看那一次 W11 是否通过。

### 1.2 空的工作树 `m4-p1b`
`.claude/worktrees/m4-p1b`（分支 `worktree-m4-p1b`，在 `7db78788`，没有自己的提交）是客户端重启时自动建出来的，Claude 问过负责人，还没有答复。建议删除：`git worktree remove .claude/worktrees/m4-p1b`，再 `git branch -D worktree-m4-p1b`。要负责人点头。

## 2. P4 `issue-collaboration` 的起步清单
- **设计**：第 12 节 P4 一段：16 个 Task，目标、任务、关闭、完成线都写好了。还要读它点名的 3.7（清洗、`skip_activity`）、3.12（评论、表情、关注、提及）、3.13（动态）、3.14（描述的历史，D5）、5.x、8.2（富文本）、8.6（游标的篡改），以及 D6（不做回复）、D8（项目管理员删别人的评论）。
- **此前各段交给 P4 的**：
  - P1b spec §5、评审 §6：描述、`skip_activity` 进 `IssuePatch`；动态的第 4 行（描述，合并）；I2 的描述的断言。
  - P2 spec §5、评审 §6：
    - 已归档的工作项上评论答 409 `issue.archived`；
    - 访客规则用到评论、表情、描述版本的读写；
    - M1-P3 交接里描述历史的地址；
    - 动态的字段名、评论的 `access`。
  - P3 spec §5、评审 §6：
    - `listIssueActivities` 对按标识指名别的工作项的行（第 5、20–23 行）执行访客规则；
    - `IssueDetail` 加 `description_html` 之后与 `Issue` 分开：HTTP 的 `issueDetail → issue` 转换在编译时故意断开，`apitest` 核对"`IssueDetail` 是 `Issue` 加一个字段"；分开之后的回答仍都经 `seen`；
    - M2 交接第 3 节：P4 的写在取得全部锁之后读时钟。
  - [M2 收尾交接](../M4-issue-core/handoffs/M2-closeout.md)、[M3 收尾交接](../M4-issue-core/handoffs/M3-closeout.md)里"处理结果（M4/P3）"末尾的"仍未处理"一行，落在 P4 的条目；设计 13.1 是全部交接的落点表。
- **评审敏感的**：
  - HTML 的清洗（bluemonday，往返的样例 S3）；
  - 提及的集合同步和关注；
  - 描述版本的 600 秒合并；
  - 评论的权限（D8）；
  - 动态列表的访客规则；
  - 游标的载荷和篡改的测试。
- **第一个提交**把 `tools/keywords.json` 的 `phase` 改成 `M4/P4`。

## 3. P5 以后的顺序
后端：
- P5 列表引擎与显示设置；
- P6 领域事件、只投递的客户端、自动归档、测试的时钟；
- P7 链接、出站客户端与抓取；
- P8 草稿、搜索与 `link_count`；
- P9 物理清理、描述版本的保留、删除项目和工作区改为任务。

前端：
- P10a 换型与删除 M6、M7 的工作项列表；
- P10b 列表一侧的数据层；
- P11 详情一侧的数据层；
- P12 列表、看板、表格、日历；
- P13 筛选、显示设置、归档页、个人主页；
- P14 详情、创建弹窗、父子、关联、链接；
- P15 时间线、评论、表情、描述的历史、草稿、搜索、命令面板；
- P16 死成员、`no-shadow` 与守卫的终态。

最后是收尾。每段的输入照第 2 节的方法收集：设计第 12 节的条目，加上此前各段 spec §5 和评审 §6 里写给它的行。P3 交给后面的见 P3 评审 §6：
- P5：`VisibleProjects` 来了之后替换 `seeEach` 和 `ArchivedProjects`；`force_custom_plan`。
- P6：事件。
- P7：链接表在全局表序里的位置。
- P8：选择器排除自己和后代。
- P9：`DeletedAt`，交错 24、25，T7.5 和 T13.4 的组合测试。
- P14：页面只在用户改了父工作项时发 `parent_id`。

## 4. M4 收尾的清单
设计第 12 节收尾一段有 14 个 Task，第 14 节是完成标准。此外，P1a–P3 的评审还留下这些：
- P1b spec §5 "收尾"一行：测试里手写的 `55P03` 改用 `postgres.LockNotAvailable`。HEAD 上剩 13 个测试文件里的 16 处，用 `git grep '"55P03"'` 找，去掉 `tx.go` 自己、它的测试和 `gate_test.go` 造的错误。
- `workspace` 的 `deactivation_test.go` 的 `waitsFor` 是 `pgtest.RowWaits` 的近似副本，合并掉。
- M3 的几个测试世界（`project_*`、`membership_world`、`ending_world`、`interleaving_*`、`permission_matrix_*`）手拼项目成员的请求，可以收进共享的帮手。这是测试整理，可选。
- 全部交接逐条写结论，`status` 改为 `closed`。给 M5–M8 写 `handoffs/M4-closeout.md`：设计 13.2 已有四个链接指向它们，内容见那张表。
- 设计 3.23 逐行核对；[plane-diff.md](../plane-diff.md)、[frontend-changes.md](../frontend-changes.md) 定稿；README 按设计 8.8 更新。
- 浏览器核对 C12 要 Chromium、Firefox、WebKit。本机只有 Chromium，Playwright 的另外两个浏览器要不要装、装在哪，到时请负责人定。
- 性能的数字写进收尾的评审，交给 M8：列表 1 万、2 万、10 万个工作项；搜索；删除；清理。
- 设计第 15 节、总体设计 9.4 标 M4 已完成。

## 5. 和 Claude 的分工（负责人 2026-10-25 的决定）
- **Codex 推进**：每段的设计细化、实现、测试、评审记录。
- **Claude 做最终审查**：建议每段在合并进 `main` 之前，由负责人请 Claude 对整个分支做一次终审，即 [02](02-method.md) 第 5 步。
  - 交给 Claude 时说明：分支名、基线、spec 和 plan 的路径、自己的评审记录或台账。
  - 终审的发现由 Codex 用一个修复轮处理，必要时再请 Claude 复审，然后合并。
  - 负责人另有安排就按负责人的。
- **上报**：遇到架构级问题时，按 [03](03-rules-and-pitfalls.md) 第 1 节上报负责人，不由 Codex 或 Claude 自行定。
