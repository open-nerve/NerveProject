# M2/P5 个人设置：评审记录

| 项 | 内容 |
|---|---|
| Phase | M2/P5 `web-account` |
| 日期 | 2026-09-28 |
| 结论 | **通过**（整分支评审提出的问题和逐个 Task 评审停放的事项已在一轮修复中处理完，修复的范围评审通过，没有留下的问题） |
| spec / plan | [spec](../specs/P5-web-account.md) / [plan](../plans/P5-web-account.md) |
| 分支 | `worktree-m2-p5-web-account`，从 `main` 的 `cec4ec9`（P4 的合并）分出 |

## 1. 评审方式

- **spec 和 plan 由架构子任务产出**（`a053e73`）：
  - 先在仓库外把整个 P5 做成原型，plan 中的代码从原型原样拼入。113 个差异块用 `patch -p1` 逐个核对。
  - 再从 `cec4ec9` 的副本出发，按 plan 逐个重放 11 个 Task，结果与原型逐文件相同（2,636 个文件）。
  - 原型中 33 个变异有 32 个被测试和故事发现，剩下 1 个被浏览器核对 C3 发现。9.6 的浏览器核对 C1–C6 在原型上全部通过。
- **执行前的核对**：
  - spec 第 3 节的 11 项差异全部接受。控制者另加了 4 项（第 12–15 条：PAT 列表的竞争、CSV 的引号、撤销确认的按钮文案、每周第一天的翻译）。设计的文字在 `f80e441` 中同步。
  - 执行前的扫描（opus，只读）：
    - 142 个块（113 个差异块、29 个整文件）；
    - 每个差异块都能在所写的行号上以零模糊应用；
    - 每个 Task 的关键词守卫、`check:*`、knip、turbo 测试都能复现，oxlint 上限等于计数。
  - 扫描提出 F1–F16，其中 F1、F2 为中。控制者逐项裁定，带进写那个文件最终版本的 Task。
- **逐个 Task 评审**：11 个 Task 各由一个实现子任务写入代码，再由独立的评审子任务（sonnet）检查"是否符合 spec"和"代码质量"。
  - 实现者按要求对自己的代码做变异核对。
  - 11 个 Task 的实现者都带着疑问交回：plan 的缺陷、测试没咬住的性质、原有的缺陷、超出 brief 的判断。控制者逐项裁定（第 3 节）。其中几次在评审之前先让实现者按裁定补改，评审看的是补改之后的状态。
  - 逐个 Task 评审要求返工的只有一次：Task 3 的 I1，关闭弹窗后 350 毫秒的清理会清掉之后又打开的弹窗里的新令牌。修复轮之后的范围评审通过。
  - 其余评审提出的 Minor 由控制者分拣：放进修复轮，或者接受。
  - 下一个 Task 的实现与上一个 Task 的评审并行，同一时间只有一个实现者提交。
- **整分支评审**（opus，`cec4ec9..926fecd`，31 个提交）：结论为"修复后可合并"，0 个 Critical、1 个 Important、9 个 Minor（第 4 节）。
  - 评审自己跑了：web 的 148 个、utils 的 59 个单元测试，oxlint（各包等于上限），knip，en 和 zh-CN 键集合的比较，以及各项计数。
  - 与它并行的 Task 11 评审（sonnet）没有发现问题。
- **修复**：
  - 一轮完成，13 个提交，`aeee615` 至 `6097b01`（基于 `926fecd`）。它包括 I1、M1–M9，以及逐个 Task 评审时停放的三项（第 4 节）。
  - 之后对 `926fecd..6097b01` 做了一次范围评审（sonnet）：通过，每一项都已关闭，没有新的问题。评审自己重算了文件数、测试数和上限，核对了 i18next 的源码（`showSupportNotice`、`createInstance`）、构建出来的 CSS 中的警告色，以及 propel 的提示对空消息的处理。
- **9.6 的浏览器核对**：由控制者在分支的代码上重跑，不是原型的代码。见第 2 节第 6 行和附录。
- **持续集成**：推送后经 GitHub 的公开接口（curl 和 jq）确认结果，见第 2 节末。

## 2. 验收标准核对（spec 第 4 节）

| # | 标准 | 结果 |
|---|---|---|
| 1 | 全部故事的页面版本和接口版本通过；此前的故事仍然通过（`make e2e` 共 48 个测试） | **通过。**<br>修复轮之后本机 `make e2e` 48 个全部通过（13.4 秒）。<br>新增和改写的页面故事：<br>- A7：每个拒绝显示在自己的字段下；密码规则两头都钉住；<br>- A8：名字的拒绝在字段下，姓可以为空，时区列表关闭，时区列表未到时按钮显示时区名；<br>- A9：键盘一步，中文星期名，被拒绝的修改，会话变化之后主题不再应用；<br>- A10：引导的拒绝在字段下；<br>- A11：CSV 按 RFC 4180 解析，弹窗关闭之后才回来的创建，两个账户，安全页的空列表；<br>- A12：停用失败，双击只发一次；<br>- S5：维护页和自动恢复。<br>每个新检查都由实现者用变异证明能失败（各 Task 的报告） |
| 2 | `domains.mjs --rows M2` 剩 24 行，每一行在附录 A.4 写明是误报，review 照录 | **通过。**<br>Task 8 实测 50 → 24，其中 50 = 开始时的 64 − Task 2、3 删掉的 15 + Task 2 给假 nerve 加的 1。24 行与 spec 附录 A.4 逐行相同，Task 8 的评审在 `c836d27` 上独立重跑得到同样的结果。<br>24 行都不是死代码：23 行是测试替身和测试中的字面量，1 行是 `OnboardingRoot` 的 `invitations`，由 M3 加回取数时传入（spec 附录 A.4 逐行说明）。<br>共用的行在 Task 8 从 257 降到 255（`TimezoneSelect` 的两个死 prop） |
| 3 | `grep -rnE "withCredentials:\s*true\|/api/users/api-tokens\|/api/timezones/" web` 没有输出 | **通过**（Task 11 在 `926fecd` 上执行，修复轮没有动这些地址）。关键词规则 `with-credentials`、`plane-api-token-urls`、`plane-timezone-urls` 看住它们（60 条规则，没有命中） |
| 4 | P4、P5 改到的文件没有 oxlint 警告；`no-unneeded-ternary` 全仓 0 条；上限 web 452、ui 19、utils 12、i18n 0 | **通过。**<br>修复轮的实现者把各包的 oxlint 结果与改到的文件取交集：P5 改到的 157 个 web 文件、P4 改到的文件都是 0 条。<br>上限：web 551 → 452，ui 25 → 19，utils 18 → 12，i18n 1 → 0。<br>utils 比 plan 的 16 低：Task 9 把 P5 改到的 `string.ts` 中 4 条原有的警告一并清掉。<br>唯一的禁用注释是新手引导资料步骤的 `autoFocus`，带理由；`DateDropdown` 外层的按键处理也保留一条带理由的（spec 第 3 节第 11 条） |
| 5 | web 单元测试 154 个、utils 59 个、i18n 12 个通过（各包 `vitest run --testTimeout=1000`） | **通过**（修复轮之后）。<br>web 从 129 到 154：PAT store 14，`store-context.test.ts` 2，有效期 3，资料和账户的应答先后 6。<br>utils 从 43 到 59：CSV 8，剪贴板 3，邮箱检查 2，密码强度 3。<br>i18n 从 10 到 12：`setLanguage` 2。 |
| 6 | 9.6 中 P5 的浏览器核对在合并前的代码上重跑（脚本对任意构建运行，见附录 A.6），写进 review | **通过。**<br>控制者在 `6097b01` 的构建上跑了 6 个核对（C1–C6），全部通过，见附录。<br>修复轮之前在 `926fecd` 上也全部通过 |
| 7 | `make lint`、`make test`、`make gen-check`、`make knip`、`make test-web`、`make e2e` 通过 | **通过**（修复轮的实现者在 `6097b01` 上执行）。<br>- `make lint-web`：54 个任务，关键词 60 条规则、3 个例外、没有命中；<br>- `make lint-go`：两段都是 `0 issues.`；<br>- `make test`：32 个包；<br>- `make gen-check`：没有差异；<br>- `make knip`、`make test-web`、`make e2e` 都通过。<br>P5 没有改 Go 代码。<br>持续集成见第 2 节末 |
| 8 | 3.20 中 P5 的一行、README 在同一次合并中写好；两份交接按第 7 节处理 | **通过。**<br>Task 11 写了 frontend-changes（`packages/services` 一行、3.1 的 M2 改为已完成、3.2 的错误处理）、README（四个标签页，令牌泄露后在哪里查看、撤销）和两份交接的处理结果；修复轮改正了其中被修复改变的句子。<br>Task 11 的评审逐项重算了计数，都与代码相符；整分支评审和修复轮的范围评审又各自核对。<br>交接见第 6 节 |
| 9 | e2e 任务的耗时写进 review（9.5、§16；附录 A.8 是本机的数） | 本机 48 个测试 13.4 秒（原型 41 个 25 秒）。<br>持续集成（ubuntu，`6097b01`）的 e2e 任务共 2 分 40 秒，其中"E2E"一步 65 秒（含 testcontainers 启动 Postgres），构建 40 秒，安装 Playwright 的浏览器 17 秒 |

持续集成：修复轮之后，`6097b01` 的分支 run 36364148797 的 web、server、e2e 三个任务都通过。合并之后 main 的持续集成在合并时记录。

## 3. 执行中的决定

这一节列控制者在执行中做的裁定。每条写明理由，以及裁定错了要付的代价。完整的记录在执行台账中，这里只列改变了行为、结构或测试的裁定。

**执行前（spec 第 3 节，扫描 F1–F16）**
- **第 11 条**：禁用注释只用在规则不适用、又没有代码改动能消除需要的地方。
- **F1**：下拉框的打开状态只要一份。第 3 条原来的做法"跟随 Headless UI 的关闭"本身就是两份状态的同步，属于打补丁。代价：Task 1 要写原型以外的代码。
- **F5**：弹窗关闭之后才回来的创建，结果丢弃。
- **F6**：CSV 的 `#` 截断，以及 CRLF。
- **F8**：自定义日期用 `set()`，并加夏令时的测试。
- **F9**：令牌的 base64url 和去掉填充的形式，以及 `document.cookie`。
- **F14**：撤销按钮一直可见。
- **F15**：复制失败给出提示。
- **F4**：P5 的页面上不留写死的英文。

**Task 1（下拉框）**
- **F1 按字面做不到。**实现者在 Headless UI 2.2.10 的浏览器中查明了几点：
  - 用鼠标打开 `Combobox` 后，焦点不在任何控件上，Escape 不起作用；
  - 没有输入框的 `Combobox` 用键盘移动不了、也选不了选项；
  - 组件没有办法关闭 `Combobox`。
- **改为这样裁定：**
  - `CustomSelect` 用 `Listbox`，按钮上的 Enter 提交所在的表单（原生 `select` 的语义），接受；
  - `CustomSearchSelect` 只有一份状态，搜索框在打开时取得焦点；
  - `DateDropdown` 保持 plan 的两份状态，M4 重构；
  - 时区按钮不在 Tab 顺序里，交给 M3。
- **代价**：api-tokens 页的日历不能用键盘打开，这是原有的问题。
- **超出裁定、接受的两项：**
  - 单选模式下清空搜索框不再把时区设为空。这是原有的缺陷：P5 之前会发 `PATCH user_timezone: null`。
  - `onOpen` 每次打开只调一次。

**Task 2（PAT store）**
- **R-A 的做法**：取数带序号，只有最新的一次写入；取数在途中时，nerve 确认的创建和撤销按顺序记下，写入时按 id 重放。
- **plan 的缺陷，接受修复**：`createToken` 可能让同一个令牌在列表中出现两次。创建和重放共用一个去重。

**Task 3（api-tokens 页）**
- **I1（评审）**：关闭后 350 毫秒的清理不认弹窗的哪一次打开，会清掉之后又打开的弹窗里的新令牌。改为一个阶段计数，打开、关闭、离开页面都让它前进；创建的结果和这次清理都认自己的阶段。评审建议的比较方法行不通（重开时计数不变），改用这个。
- **剪贴板的回退**（`execCommand`）原来吞掉失败，纯 http 的部署上复制总是报成功。在 `@nerve/utils` 中从根上改：失败时拒绝。代价：M3、M4 中没有 `catch` 的调用方在这种失败下会有未处理的拒绝，与 `navigator.clipboard.writeText` 拒绝时相同（M2 设计 13.2）。
- **Blob 地址在点击后立即撤销**：接受，理由是 HTML 规范在点击时解析 blob 地址；只在 Chromium 中核对过。
- **CSV 的列名保持英文**：接受，它们是文件的格式。

**Task 4（A11）**
- 加上 hex(secret)。brief 的注释说有，代码里没有。
- 加上 `recordTokensShown`，一个 `MutationObserver`，记下页面上哪怕只出现过一瞬间的令牌。
- 评审的 Minor：令牌只在输入框的 `.value` 中时看不到。在 Task 5 加上"表单的值"这一处。

**Task 5（安全页）**
- **plan 的缺陷**：A7 按整页找拒绝的文字，拒绝显示在错的字段下也能通过。改为按字段所在的块找。
- 安全页没有令牌时的那一行，在 A11 中核对。

**Task 6（preferences）**
- **被拒绝的修改让页面停在 nerve 没有的状态。**
  - 主题改为在 nerve 成功之后才应用。
  - 语言：控制者先裁定"成功之后再设"，前提是会话切换之后回来的 200 会抛出 `SessionChangedError`。实现者证明这个前提不成立：中间件只在发出之前、或应答 401 时抛出。改为方案 A：界面语言是当前一代资料的派生状态，在 `StoreWrapper` 中设；stores 不再设语言，资料只取 nerve 的应答。这同时消除了 P4 spec 第 3 节第 17 条记的已知局限。代价：语言比应答晚一次渲染。
- **命令面板**：主题的提示用新的键，星期用 `i18n_label`，`START_OF_THE_WEEK_OPTIONS` 的 `label` 删除。
- 再次选中同一天会发一次相同的 `PATCH`，接受。

**Task 7（general）**
- **姓名规则只有 nerve 的一份。**客户端的 Plane 规则与 nerve 的契约不同：姓必填、50 个字、显示名不能有空格、提示是英文。结果是每个经过新手引导的账户（姓为空）都保存不了 general 页。
  - general 页和新手引导的资料步骤删掉 `validatePersonName`、`validateDisplayName`，只保留翻译过的必填。
  - nerve 的字段错误按 `code` 显示在字段下。
  - 两个校验函数和它们的正则从 `@nerve/utils` 删除。
  - 代价：名字不合规时，要等一次往返才提示。
- **新加的故事**：A8、A10、A12 失败的一步，S5。

**Task 8、9**
- `TimezoneSelect` 的三个死 prop 删除。
- 邀请卡片的按钮内只放短语内容。
- `string.ts` 的 4 条旧警告一并清掉，utils 的上限降到 12。
- 引导页剩下的英文在 Task 9 翻译。

**Task 10**
- `isFocused ||` 的变异在应用层是等价的：两个调用方都只在"非空且不合规"时显示提示。所以删掉死掉的焦点判断。
- 提示自己决定何时显示，调用方删掉重复的判断。
- 文案由调用方经共用的 hook 用 `t()` 传入，`@nerve/ui` 仍不依赖 `@nerve/i18n`。
- A2、A7 按精确的文字钉住两头。

## 4. 整分支评审的发现与处理

| # | 发现 | 处理 |
|---|---|---|
| I1 | M2 自己的登录、注册、新手引导页还有写死的英文（M2 设计 7.8） | **修复**（`aeee615`）：按两个目录逐个清查，文案都走 `t()`；新手引导的失败提示按 `code` 取。新加 76 个键，中英文相同；英文的值不变，e2e 按精确文字读取。留下的英文有：产品名、示例邮箱、头像的首字母，以及第三方组件的默认文案（Banner 的 "Dismiss"，Base UI 的 "Notifications"），各自写明理由 |
| M1 | 资料和账户的 `PATCH` 应答不按先后写入 | **修复**（`ea77e39`）：两个 store 按发出的顺序给修改编号，比最后写入的那次旧的应答丢弃；新的修改被拒绝时，旧的成功仍会写入。每个 store 3 个单元测试，顺序由假 nerve 强制 |
| M2 | 主题在会话变化之后仍被应用，页面刷新 | **修复**（`15b2b13`）：在等待之前记下 `loginId`，会话变了就不应用、不刷新；A9 的第三个页面测试 |
| M3 | 停用账户的确认按钮在请求途中仍可点击 | **修复**（`650e700`）：`loading` 时禁用；A12 双击只发一次 |
| M4 | `useTranslation` 中没有调用方的第二个写语言的地方 | **删除**（`b4cf039`），连同包的索引中没有使用者的导出 |
| M5 | spec 2.10 说"两处清警告改了行为"，实际更多 | **更正**（`6097b01`）：逐项列出 |
| M6 | 13.2 没有列出重建的两个下拉框的其余调用方 | **补上**（`6097b01`）：`CustomSelect` 16 个文件、`CustomSearchSelect` 的调用方，按 M 列出 |
| M7 | 时区列表没有账户的时区，或者列表还在加载时，按钮是空的 | **修复**（`cbb83cf`）：显示时区名；删掉 Plane 的 `"Asia/Kolkata"` 默认值；A8 扣住列表核对 |
| M8 | 密码的字符规则写了两份 | **修复**（`2437d0b`）：`PASSWORD_RULES` 一份，强度由它推出；50 万个字符串新旧比较没有差异 |
| M9 | api-tokens 页的说明说令牌"连接工作区"；中文混用两个术语 | **修复**（`5dc6853`）：说明改为属于账户；中文统一为"访问令牌" |
| 停放 | `setLanguage` 在等待之后按自己的参数写 `localStorage` 和 `<html lang>`（Task 6 评审） | **修复**（`46b48a2`）：i18next 最终没有采用这个语言时，不写；2 个单元测试。随之 `instance.ts` 的一条警告清掉（`eb2baf7`，i18n 上限 1 → 0） |
| 停放 | 弱密码状态的颜色类在构建出来的 CSS 中不存在（Task 10） | **修复**（`4d527a3`）：改用设计系统的警告色；浏览器探测核对计算出的颜色 |
| 停放 | 项目成员表中没有作用的 `cursor-pointer`（Task 9 评审） | **删除**（`005edff`） |
| 停放 | A11 中 `MessageChannel` 的等待没有自己的期限（Task 3 评审） | **接受**：修复轮没有改 A11；它受 Playwright 测试期限的约束，会失败，不会挂住 |

评审列出、没有判断的事项（全应用的日期格式、CSV 的公式注入等），控制者同意评审的理由。

## 5. 最终实现与 plan 的差异

plan 的代码在原型中逐字运行过。已提交的代码在下面这些地方不同于 plan，每一处都来自第 3、4 节的裁定。

- **下拉框**：
  - `CustomSelect` 用 `Listbox`；
  - `CustomSearchSelect` 只有一份状态，打开时搜索框取得焦点，有空值守卫，`onOpen` 每次打开只调一次，另有两个文案的 prop；
  - `DateDropdown` 的禁用理由更正。
- **PAT store**：取数的序号和按 id 的重放；创建去重。
- **api-tokens 页**：
  - 用 `csvText` 和 Blob 下载；
  - 创建和清理由阶段计数控制；
  - 撤销按钮常显，确认的文案翻译；
  - 复制失败有提示；
  - 自定义日期用 `set()`。
- **A11**：
  - 令牌的 10 种形式，7 处位置；
  - CSV 的解析；
  - `holdAnswer`、`recordTokensShown`；
  - 弹窗关闭之后才回来的创建。
- **preferences**：
  - 界面语言在 `StoreWrapper` 中派生；
  - 主题在成功之后应用，并且认会话；
  - 星期、时区的文案；
  - 资料和账户的应答按先后；
  - 被拒绝的修改的测试。
- **general 和新手引导**：
  - 姓名规则只在 nerve；
  - 拒绝显示在字段下；
  - 停用的按钮在请求途中禁用；
  - S5。
- **清理**：
  - `TimezoneSelect` 的死 prop；
  - 邀请卡片的短语内容；
  - `string.ts` 的警告；
  - 密码提示自己决定何时显示，并接收翻译的文案；
  - 密码规则只有一份。
- **文档**：计数按提交之后的代码重算；13.2 加了第 6 节列出的几行。

## 6. 移交事项

| 交接 | 状态 | P5 处理的条目 | 留下的条目 |
|---|---|---|---|
| [M1-P3-trim-platform](../handoffs/M1-P3-trim-platform.md) | done | `@nerve/services` 的 API 令牌旧地址：令牌服务和 axios 基类删除，PAT 经 store 用生成的客户端（Task 2、3）；关键词规则 `plane-api-token-urls`、`with-credentials` | — |
| [M1-closeout](../handoffs/M1-closeout.md) | done | 死成员和死 prop（64 → 24，剩下的是误报）；oxlint 在改到的文件中清零，`no-unneeded-ternary` 清零，上限调低（Task 8–10、修复轮）；主题下拉框（Task 1） | — |

P4 评审交给 P5 的事项都已处理：
- 个人设置的四个标签页；
- PAT store；
- `@nerve/services`；
- 主题下拉框；
- 死成员；
- oxlint 和 `no-unneeded-ternary`；
- 三条关键词规则；
- A7–A9、A11、A12 的页面版本；
- 维护页的文案。

P4 评审交给 M3 的"邀请页 `fetchWorkspaces()` 的拒绝"也在 P5 完成（spec 第 3 节第 8 条）。

交给后续的事项，都写在 M2 设计 13.2：

| 交给 | 事项 |
|---|---|
| M3–M6（接上组件的 M） | 其余 12 个下拉框的定位和打开状态。<br>重建的 `CustomSelect`、`CustomSearchSelect` 在 P5 页面以外的调用方：`Listbox` 上 Enter 提交表单、按字母跳转、打开时页面其余部分 inert；一份打开状态、空值守卫、`onOpen` 每次打开一次 |
| M3 | 时区按钮不在 Tab 顺序里。<br>M2 走不到的运行时改动：<br>- `WorkspaceAuthWrapper` 的退出按钮；<br>- `ProfileSidebar` 经 ref 读折叠状态；<br>- 两个包装去掉 `isLoading`；<br>- `WorkspaceLogo` 改为 `<span>`；<br>- 加入工作区一步的 `<label>` 和 `Checkbox`；<br>- 加入、邀请两步和导览的中文。 |
| M4 | `DateDropdown` 的两份打开状态（键盘打不开日历，按 Tab 之后两份状态走散）。<br>`defaultOpen` 和面包屑的 `onOpen` 在浏览器中核对。<br>`cycle-options` 的取数：打开期间改了工作区或项目会重跑，重开时两个 GET 不分先后。<br>编辑器的代码分割和 `EMOJI_CHECK_WARNING`。<br>命令面板的主题命令在 `PATCH` 之前就应用、失败时不撤回 |
| M3、M4、M6、M7 | `copyTextToClipboard` 的 29 处调用中 16 处没有 `catch`，复制失败时是未处理的拒绝。各自接上页面时补上 `catch` 和提示 |

P3b、P4 评审交给后续的事项不变：
- M4 的命令行"只投递" River 客户端；
- M8 的 `stop_grace_period`；
- 负责人的 argon2 调参与登录耗时；
- M3 起每个会话一个 `RootStore` 的约定。

## 7. 已知限制

- **日历不能用键盘打开**：`DateDropdown` 的限制。api-tokens 页的有效期中，预设的几项和"永不过期"可以用键盘操作。另外，在日历里按 Tab 之后，两份打开状态会走散，再点另一个下拉框时，两个列表会同时开着。M4 重构。
- **时区按钮不在 Tab 顺序里**：Headless UI 的 `Combobox.Button` 把 `tabIndex` 固定为 -1，P5 之前也是这样。用键盘打开和选择都可以（先让按钮取得焦点）。M3 处理。
- **组件没有 DOM 的单元测试**：它们由 e2e 和浏览器探测覆盖。中文文案只由探测核对，英文和中文的键集合由 `check:sync` 保证相同。
- **Blob 地址在点击后立即撤销**：只在 Chromium 中核对过。
- **资料和账户的应答先后**：假定 nerve 按发出的顺序处理这些请求（与 PAT store 的假定相同）。
- **再次选中已选的项**：会发一次相同的 `PATCH`（Headless UI 单选的行为，A9 钉住）。
- **`EMOJI_CHECK_WARNING` 按原文写在 e2e 中**：依赖升级改了它时，要更新 fixture。M4 去掉它。

## 附录：9.6 的浏览器核对

按 M2 设计 9.6（沿用 M1 设计 7.5 的做法）：临时脚本不进仓库，脚本全文、假数据、运行命令和断言写在这里。浏览器核对是单元测试和端到端测试之外的证据，不代替它们。

- **对象**：合并前的代码 `6097b01`（修复轮之后）。修复轮之前在 `926fecd` 上也跑过一次，6 项全部通过。架构子任务在原型上的运行记在 spec 附录 A.6。
- **做法**：
  - 把工作区复制到临时目录，在那里 `make build`，脚本对那里的 `bin/nerve` 运行（`lib.mjs` 的 `P`）。
  - 每个核对各自启动一个 nerve，在开发库 `nerve-dev-db-1` 上建一个独立的库，结束时删除。
  - 运行结束后没有留下进程和库。
- **命令**：`P=<分支副本> node checks.mjs`。
- **环境**：Node 24.15.0，Playwright 1.63.0 的 Chromium（取自 `e2e` 包的依赖），macOS。
- **假数据**：
  - 每个核对用 `email(标签)` 生成带随机后缀的邮箱，密码是 `lib.mjs` 的 `PASSWORD`，账户经注册接口创建并完成新手引导。
  - C3 的令牌名称、说明和有效期，C4 的错误密码和常见密码，C6 扣住的实例请求，都写在脚本中。
- **断言**：每个核对的通过条件是 `report(id, 条件, 详情)` 的第二个参数；下面"运行输出"中每行的 JSON 是详情。

### 结果（`6097b01`）

| 核对 | 9.6 中 P5 的条目 | 结果 |
|---|---|---|
| C1 | 个人设置的四个标签页 | 通过。general、preferences、security、api-tokens 都能从侧栏打开，地址正确，内容显示；安全页列出令牌；没有旧接口的请求、失败的请求和页面错误 |
| C2 | 主题下拉框在按钮旁展开（中英文各一次），以及时区、每周第一天 | 通过。英文和中文下主题、每周第一天、时区的列表都在按钮正下方 4 像素、边缘对齐；一次 Escape 关闭，下一次点击再打开；切换到中文 200，页面立即变成中文；中文下"一周的第一天"和"星期日"，没有英文标题 |
| C3 | 令牌只显示一次，CSV | 通过。创建 201；日历在按钮旁；原文显示一次，CSV 含原文，表头以 CRLF 结尾；关闭后 350 毫秒内重开显示空表单；关闭之后原文不在页面、地址、存储和控制台中；列表显示新令牌和"从未使用" |
| C4 | 安全页的字段错误和令牌列表 | 通过。当前密码错、新密码太常见都是 422，提示在各自的字段下方；列出令牌，就地撤销 204，列表变空，令牌随后得到 401 |
| C5 | 停用账户 | 通过。确认弹窗的文案；204；回到登录页（`next_path` 指回 general），浏览器里没有记录；再登录 403，"This account is deactivated."；管理员 `nerve users activate` 之后登录 200，进入 `/onboarding` |
| C6 | 维护页 | 通过。实例请求失败时显示"Nerve cannot be reached right now"和说明；恢复后约 5.3 秒自己继续，不刷新页面 |

### 脚本全文

`lib.mjs`（`P` 是分支副本的目录，`OUT` 是截图和日志的目录；路径在这里写成 `/tmp/nerve-m2/…`）：

```js
// Helpers of the M2/P5 browser checks (M2 design 9.6; P4's lib.mjs): nerve processes, scratch databases, API
// calls, signed-in browser contexts and page watches. Not part of the repository.
//
// P is the build the checks run: a checkout where `make build` has run (bin/nerve with the web app in it, and
// e2e's node modules for Playwright and pg). The environment variable P overrides it; OUT, where the logs and
// screenshots go, likewise.
import { spawn, execFileSync } from "node:child_process";
import { closeSync, mkdirSync, openSync, readFileSync, rmSync } from "node:fs";
import { once } from "node:events";
import { randomBytes } from "node:crypto";
import { createRequire } from "node:module";
import { setTimeout as sleep } from "node:timers/promises";

export const P =
  process.env.P ?? "/tmp/nerve-m2/p5proto";
export const OUT =
  process.env.OUT ?? "/tmp/nerve-m2/p5checks/out";
mkdirSync(OUT, { recursive: true });
const require = createRequire(`${P}/e2e/package.json`);
export const { chromium } = require("@playwright/test");
const { Client } = require("pg");

export const PASSWORD = "Tr0ub4dor&3";
export const DEV_DB = "postgres://nerve:nerve@localhost:55432/nerve?sslmode=disable";
export { sleep, randomBytes };

// ---- results -------------------------------------------------------------------------------------------

export const results = [];
/** Records one check: its id, whether it held, and what was seen. */
export function report(id, ok, seen) {
  results.push({ id, ok, seen });
  console.log(`${ok ? "PASS" : "FAIL"} ${id} ${JSON.stringify(seen)}`);
}

/** Runs one check; an exception is its failure. */
export async function run(id, fn) {
  const started = Date.now();
  try {
    await fn();
  } catch (error) {
    report(id, false, { error: String(error?.stack ?? error).split("\n").slice(0, 4).join(" | ") });
  }
  console.log(`  (${id}: ${Date.now() - started} ms)`);
}

// ---- databases -----------------------------------------------------------------------------------------

export async function sql(url, text, params = []) {
  const client = new Client({ connectionString: url });
  await client.connect();
  try {
    return (await client.query(text, params)).rows;
  } finally {
    await client.end();
  }
}

/** A new database in nerve-dev-db-1; dropDatabase removes it. */
export async function createDatabase(name) {
  await sql(DEV_DB, `CREATE DATABASE ${name}`);
  return DEV_DB.replace("/nerve?", `/${name}?`);
}

export async function dropDatabase(name) {
  await sql(DEV_DB, `DROP DATABASE IF EXISTS ${name} WITH (FORCE)`);
}

/** The session id and generation a refresh token carries (M2 design 3.4). */
export function tokenParts(token) {
  const raw = Buffer.from(token.slice("nrv_rt_".length), "base64url");
  const id = raw.subarray(0, 16).toString("hex");
  return {
    sessionId: `${id.slice(0, 8)}-${id.slice(8, 12)}-${id.slice(12, 16)}-${id.slice(16, 20)}-${id.slice(20)}`,
    generation: raw.readUInt32BE(16),
  };
}

export async function sessionOf(dbUrl, refreshToken) {
  const rows = await sql(
    dbUrl,
    "SELECT id, generation, ip::text AS ip, revoked_at, revoke_reason FROM auth_sessions WHERE id = $1",
    [tokenParts(refreshToken).sessionId]
  );
  return rows[0];
}

// ---- nerve ---------------------------------------------------------------------------------------------

function nerveEnv(dbUrl, extra) {
  const inherited = Object.fromEntries(Object.entries(process.env).filter(([k]) => !k.startsWith("NERVE_")));
  return { ...inherited, NERVE_ENV: "test", NERVE_DATABASE__URL: dbUrl, ...extra };
}

/** Starts bin/nerve serve on addr and waits for /readyz; its output goes to out/<name>.log. */
export async function startNerve(name, dbUrl, addr, extra = {}) {
  const addrFile = `${OUT}/${name}.addr`;
  const logFile = `${OUT}/${name}.log`;
  rmSync(addrFile, { force: true });
  const log = openSync(logFile, "a");
  const child = spawn(`${P}/bin/nerve`, ["serve"], {
    env: nerveEnv(dbUrl, { NERVE_SERVER__ADDR: addr, NERVE_SERVER__ADDR_FILE: addrFile, ...extra }),
    stdio: ["ignore", log, log],
  });
  closeSync(log);
  const deadline = Date.now() + 30_000;
  while (Date.now() < deadline) {
    if (child.exitCode !== null) throw new Error(`nerve ${name} exited (${logFile})`);
    let listen;
    try {
      listen = readFileSync(addrFile, "utf8");
    } catch {}
    if (listen) {
      const port = Number(listen.slice(listen.lastIndexOf(":") + 1));
      const local = `http://127.0.0.1:${port}`;
      const ready = await fetch(`${local}/readyz`).then((r) => r.ok, () => false);
      if (ready) {
        return {
          name,
          port,
          baseURL: local,
          logFile,
          async stop() {
            if (child.exitCode === null) {
              const exited = once(child, "exit");
              child.kill("SIGTERM");
              await exited;
            }
          },
        };
      }
    }
    await sleep(100);
  }
  child.kill("SIGKILL");
  throw new Error(`nerve ${name} did not become ready (${logFile})`);
}

// ---- API -----------------------------------------------------------------------------------------------

export async function call(base, method, path, body, token) {
  const headers = {};
  if (body !== undefined) headers["Content-Type"] = "application/json";
  if (token) headers.Authorization = `Bearer ${token}`;
  const res = await fetch(base + path, { method, headers, body: body === undefined ? undefined : JSON.stringify(body) });
  const text = await res.text();
  return { status: res.status, body: text ? JSON.parse(text) : undefined };
}

async function tokensFrom(res, want, what) {
  if (res.status !== want) throw new Error(`${what}: ${res.status} ${JSON.stringify(res.body)}`);
  return res.body;
}

export async function register(base, email) {
  return tokensFrom(await call(base, "POST", "/api/v0/auth/register", { email, password: PASSWORD }), 201, `register ${email}`);
}

export async function login(base, email) {
  return tokensFrom(await call(base, "POST", "/api/v0/auth/login", { email, password: PASSWORD }), 200, `login ${email}`);
}

export async function onboard(base, tokens) {
  const res = await call(base, "PATCH", "/api/v0/me/profile", { is_onboarded: true }, tokens.access_token);
  if (res.status !== 200) throw new Error(`onboard: ${res.status}`);
}

let counter = 0;
/** A fresh address for a check. */
export function email(label) {
  counter += 1;
  return `${label}-${Date.now().toString(36)}-${counter}@example.com`;
}

// ---- browser -------------------------------------------------------------------------------------------

/** Makes the pages of context at base start signed in with tokens, as the e2e fixture signedInPage does. */
export async function seed(context, base, tokens) {
  const record = JSON.stringify({ refresh_token: tokens.refresh_token, login_id: randomBytes(16).toString("hex") });
  await context.addInitScript(
    ({ origin, record }) => {
      if (window.location.origin !== origin || localStorage.getItem("nerve.auth.e2e-seeded") !== null) return;
      localStorage.setItem("nerve.auth.e2e-seeded", "1");
      localStorage.setItem("nerve.auth", record);
    },
    { origin: new URL(base).origin, record }
  );
  return record;
}

/** Every console warning of every watched page, for the navigate() check. */
export const allWarnings = [];

/** Watches page as the e2e fixture watchPage does: API calls, failures, errors, CSP violations. */
export async function watch(page, label) {
  const w = { label, api: [], apiFailures: [], oldApi: [], pageErrors: [], consoleErrors: [], warnings: [], csp: [], refreshes: [], pending: [] };
  page.on("request", (req) => {
    const path = new URL(req.url()).pathname;
    if (!path.startsWith("/api/")) return;
    w.api.push(`${req.method()} ${path}`);
    if (!path.startsWith("/api/v0/")) w.oldApi.push(`${req.method()} ${path}`);
  });
  // A request nerve answered is judged by its status: Chromium also reports one failed (net::ERR_ABORTED) when
  // the page leaves the body of its answer unread, as openapi-fetch does with a 204's (e2e's watchPage).
  const answered = new WeakSet();
  page.on("response", (res) => {
    answered.add(res.request());
    const path = new URL(res.url()).pathname;
    if (path.startsWith("/api/") && res.status() >= 400) w.apiFailures.push(`${res.status()} ${res.request().method()} ${path}`);
  });
  page.on("requestfailed", (req) => {
    const path = new URL(req.url()).pathname;
    if (path.startsWith("/api/") && !answered.has(req)) w.apiFailures.push(`${req.failure()?.errorText} ${req.method()} ${path}`);
  });
  // Every finished refresh, recorded once its answer is read: await Promise.all(w.pending) first.
  page.on("requestfinished", (req) => {
    if (new URL(req.url()).pathname !== "/api/v0/auth/refresh") return;
    w.pending.push(
      req.response().then((res) => {
        const t = req.timing();
        const sent = JSON.parse(req.postData() ?? "{}").refresh_token;
        w.refreshes.push({ start: t.startTime, end: t.startTime + t.responseEnd, status: res?.status(), sentGeneration: sent ? tokenParts(sent).generation : null, finishedAt: Date.now() });
      })
    );
  });
  page.on("pageerror", (e) => w.pageErrors.push(e.message));
  page.on("console", (m) => {
    if (m.type() === "error") w.consoleErrors.push(m.text());
    if (m.type() === "warning") {
      w.warnings.push(m.text());
      allWarnings.push(`${label}: ${m.text()}`);
    }
  });
  await page.exposeBinding("__cspViolation", (_s, v) => w.csp.push(v));
  await page.addInitScript(() => {
    document.addEventListener("securitypolicyviolation", (e) => window.__cspViolation(`${e.effectiveDirective} ${e.blockedURI}`));
  });
  return w;
}

/** Holds the first refresh of context from now on until another goes out or holdMs pass (as e2e A4). */
export async function holdFirstRefresh(context, holdMs) {
  let release;
  await context.route("**/api/v0/auth/refresh", async (route) => {
    if (release === undefined) {
      await new Promise((resolve) => {
        release = resolve;
        setTimeout(resolve, holdMs);
      });
    } else {
      release();
    }
    await route.continue();
  });
}

/**
 * True when no two refreshes overlapped: each sent the token the one before it got, so the generations sent
 * are 0, 1, 2 … once each (the browser's timings of a held request count from when it was let go).
 */
export function oneAtATime(refreshes) {
  const sent = refreshes.map((r) => r.sentGeneration).sort((a, b) => a - b);
  return sent.every((g, i) => g === i);
}

export async function fillSignIn(page, address, password = PASSWORD) {
  await page.getByLabel("Email", { exact: true }).clear();
  await page.getByLabel("Email", { exact: true }).fill(address);
  await page.getByLabel("Password", { exact: true }).fill(password);
  const [res] = await Promise.all([
    page.waitForResponse((r) => new URL(r.url()).pathname === "/api/v0/auth/login"),
    page.getByRole("button", { name: "Go to workspace" }).click(),
  ]);
  return res.status();
}

export async function fillSignUp(page, address, password = PASSWORD) {
  await page.getByLabel("Email", { exact: true }).fill(address);
  await page.getByLabel("Set a password", { exact: true }).fill(password);
  await page.getByLabel("Confirm password", { exact: true }).fill(password);
  const [res] = await Promise.all([
    page.waitForResponse((r) => new URL(r.url()).pathname === "/api/v0/auth/register"),
    page.getByRole("button", { name: "Create account" }).click(),
  ]);
  return res.status();
}

/**
 * The answer to method path on page, up to timeout: a wait for a page's last request, e.g. the profile step's
 * PATCH /api/v0/me/profile. After a client-side navigation, waitForLoadState("networkidle") returns at once.
 */
export function answerTo(page, method, path, timeout = 10_000) {
  return page.waitForResponse((res) => res.request().method() === method && new URL(res.url()).pathname === path, { timeout });
}

export async function visible(locator, timeout = 10_000) {
  return locator.waitFor({ state: "visible", timeout }).then(() => true, () => false);
}

export async function alertText(page) {
  const alert = page.getByRole("alert");
  return (await visible(alert, 5_000)) ? (await alert.innerText()).trim() : null;
}

/** Runs nerve users with args on the database dbUrl, as the server's administrator does; returns its output. */
export function nerveUsers(dbUrl, args) {
  return execFileSync(`${P}/bin/nerve`, ["users", ...args], { env: nerveEnv(dbUrl, {}), encoding: "utf8" });
}

/**
 * Where list shows against button: the gap under or over it and the gap between the edges lined up (left or
 * right), in pixels; beside is true when the list is right under or over the button (< 8) and lined up (< 2).
 */
export async function placement(list, button) {
  const [at, by] = await Promise.all([list.boundingBox(), button.boundingBox()]);
  if (!at || !by) return { beside: false, at, by };
  const vertical = Math.min(Math.abs(at.y - (by.y + by.height)), Math.abs(by.y - (at.y + at.height)));
  const edges = Math.min(Math.abs(at.x - by.x), Math.abs(at.x + at.width - (by.x + by.width)));
  const round = (b) => ({ x: Math.round(b.x), y: Math.round(b.y), w: Math.round(b.width), h: Math.round(b.height) });
  return { beside: vertical < 8 && edges < 2, vertical: Math.round(vertical), edges: Math.round(edges), list: round(at), button: round(by) };
}

export function docker(args) {
  return execFileSync("docker", args, { encoding: "utf8" }).trim();
}
```

`checks.mjs`：

```js
// The M2/P5 browser checks (M2 design 9.6, P5): the four settings tabs; the theme's list beside its button in
// English and Chinese (and the time zones', the week's and the date's); a token shown once, with its CSV; the
// security page's field errors and token list; deactivating, back to sign-in, sign-in refused, nerve users
// activate; the maintenance page. Each check starts its own nerve on a scratch database of nerve-dev-db-1,
// dropped at the end. Screenshots and logs go to OUT.
//
// usage: node checks.mjs [check-id...]      (P=<a built checkout> to run another build)
import { readFile } from "node:fs/promises";

import {
  OUT,
  PASSWORD,
  alertText,
  answerTo,
  call,
  chromium,
  createDatabase,
  dropDatabase,
  email,
  fillSignIn,
  nerveUsers,
  onboard,
  placement,
  randomBytes,
  register,
  report,
  results,
  run,
  seed,
  sleep,
  startNerve,
  visible,
  watch,
} from "./lib.mjs";

const only = new Set(process.argv.slice(2));
const tag = randomBytes(4).toString("hex");
const dbName = `p5checks_${tag}`;
const db = await createDatabase(dbName);
const nerve = await startNerve(`${tag}-checks`, db, "127.0.0.1:0");
const base = nerve.baseURL;
const browser = await chromium.launch();

/** A signed-in, onboarded account and a page of a new context in its session. */
async function signedIn(label, viewport = { width: 1280, height: 800 }) {
  const address = email(label);
  const tokens = await register(base, address);
  await onboard(base, tokens);
  const context = await browser.newContext({ baseURL: base, viewport, acceptDownloads: true });
  await seed(context, base, tokens);
  const page = await context.newPage();
  const w = await watch(page, label);
  return { address, tokens, context, page, w };
}

/** What the watch saw that a settings page must not have: old addresses, failures, errors. */
function quiet(w, allowedFailures = []) {
  return {
    oldApi: w.oldApi,
    apiFailures: w.apiFailures.filter((f) => !allowedFailures.includes(f)),
    pageErrors: w.pageErrors,
  };
}
const isQuiet = (q) => q.oldApi.length === 0 && q.apiFailures.length === 0 && q.pageErrors.length === 0;

async function check(id, fn) {
  if (only.size > 0 && !only.has(id)) return;
  await run(id, fn);
}

try {
  // C1: the four tabs, each from the sidebar, each with what it is for.
  await check("C1-four-tabs", async () => {
    const { page, context, w } = await signedIn("tabs");
    await page.goto("/settings/profile/general");
    const seen = {};
    const tabs = [
      ["Profile", "/settings/profile/general", page.locator("#first_name")],
      ["Preferences", "/settings/profile/preferences", page.getByRole("button", { name: "System Preference" })],
      ["Security", "/settings/profile/security", page.getByRole("button", { name: "Change password" })],
      ["Personal Access Tokens", "/settings/profile/api-tokens", page.getByRole("button", { name: "Add access token" }).first()],
    ];
    for (const [tab, path, marker] of tabs) {
      await page.getByRole("button", { name: tab, exact: true }).click();
      const shown = await visible(marker);
      seen[tab] = { url: new URL(page.url()).pathname, shown };
      await page.screenshot({ path: `${OUT}/C1-${path.split("/").pop()}.png` });
    }
    // The security page lists the tokens under its form.
    await page.getByRole("button", { name: "Security" }).click();
    seen.securityTokens = await visible(page.getByText("Personal access tokens", { exact: true }));
    const q = quiet(w);
    report(
      "C1-four-tabs",
      tabs.every(([tab, path]) => seen[tab].url === path && seen[tab].shown) && seen.securityTokens && isQuiet(q),
      { ...seen, ...q }
    );
    await context.close();
  });

  // C2: the theme's list opens beside its button, in English and in Chinese; so do the week's and the time
  // zones' (a list with a search box) on the same page.
  await check("C2-dropdowns-beside", async () => {
    const { page, context, w, tokens } = await signedIn("theme");
    const themed = await call(base, "PATCH", "/api/v0/me/profile", { theme: "dark" }, tokens.access_token);
    if (themed.status !== 200) throw new Error(`theme: ${themed.status}`);
    await page.goto("/settings/profile/preferences");
    const seen = {};
    // Each list: where it opens; how many Escapes close it (one); whether the next click opens it again (before P5
    // an Escape closed only Headless UI's own state, and the next click did nothing).
    const open = async (key, button) => {
      console.log(`  C2: ${key}`);
      await button.click();
      const list = page.getByRole("listbox");
      await list.waitFor({ timeout: 5_000 }).catch(async (error) => {
        await page.screenshot({ path: `${OUT}/C2-${key}-failed.png` });
        throw error;
      });
      seen[key] = await placement(list, button);
      await page.screenshot({ path: `${OUT}/C2-${key}.png` });
      let escapes = 0;
      while ((await list.count()) > 0 && escapes < 3) {
        await page.keyboard.press("Escape");
        escapes += 1;
        await list.waitFor({ state: "detached", timeout: 1_000 }).catch(() => {});
      }
      seen[key].escapesToClose = escapes;
      await button.click();
      seen[key].reopensAtFirstClick = await visible(list, 2_000);
      while ((await list.count()) > 0) {
        await page.keyboard.press("Escape");
        await list.waitFor({ state: "detached", timeout: 1_000 }).catch(() => {});
      }
    };
    await open("theme-en", page.getByRole("button", { name: "Dark", exact: true }));
    await open("week-en", page.getByRole("button", { name: "Sunday" }));
    await open("timezone-en", page.getByRole("button", { name: "UTC" }));
    // Chinese, from the page's own language list.
    await page.getByRole("button", { name: "English" }).click();
    const spoken = answerTo(page, "PATCH", "/api/v0/me/profile");
    await page.getByRole("option", { name: "简体中文" }).click();
    seen.languageStatus = (await spoken).status();
    seen.chinese = await visible(page.getByText("语言和时间"));
    await open("theme-zh", page.getByRole("button", { name: "深色" }));
    await open("timezone-zh", page.getByRole("button", { name: "UTC" }));
    // The week's title and day names are translated since P5 (R-D): Chinese here, and no English left.
    seen.weekTitleZh = await visible(page.getByText("一周的第一天"), 2_000);
    seen.weekTitleEnGone = (await page.getByText("First day of the week").count()) === 0;
    await open("week-zh", page.getByRole("button", { name: "星期日" }));
    await page.screenshot({ path: `${OUT}/C2-zh-page.png` });
    const q = quiet(w);
    const keys = ["theme-en", "week-en", "timezone-en", "theme-zh", "timezone-zh", "week-zh"];
    report(
      "C2-dropdowns-beside",
      keys.every((k) => seen[k].beside && seen[k].escapesToClose === 1 && seen[k].reopensAtFirstClick) &&
        seen.chinese &&
        seen.weekTitleZh &&
        seen.weekTitleEnGone &&
        isQuiet(q),
      { ...seen, ...q }
    );
    await context.close();
  });

  // C3: a token shows once, and its CSV downloads with it; closed, it is nowhere, not even in the modal opened
  // again at once (before the modal's reset after closing, 350 ms); the list shows it without the token.
  await check("C3-token-once", async () => {
    const { page, context, w } = await signedIn("token");
    const log = [];
    page.on("console", (m) => log.push(m.text()));
    await page.goto("/settings/profile/api-tokens");
    await page.getByRole("button", { name: "Add access token" }).first().click();
    await page.getByLabel("Title").fill("deploy");
    await page.getByLabel("Description").fill("ci");
    await page.getByRole("button", { name: "Set expiration date" }).click();
    await page.getByRole("option", { name: "Custom date" }).click();
    const day = page.getByRole("button", { name: "Set date" }).first();
    await day.click();
    const calendar = await placement(page.getByRole("listbox"), day);
    await page.screenshot({ path: `${OUT}/C3-calendar.png` });
    await page.getByRole("grid").getByRole("button", { disabled: false }).last().click();
    const downloaded = page.waitForEvent("download", { timeout: 10_000 });
    const created = answerTo(page, "POST", "/api/v0/me/api-tokens");
    await page.getByRole("button", { name: "Generate token" }).click();
    const answer = await created;
    const body = await answer.json();
    const shown = await visible(page.getByText(body.token));
    await page.screenshot({ path: `${OUT}/C3-shown.png` });
    const csv = await readFile(await (await downloaded).path(), "utf8");
    // From here on, any appearance of the token in the document is recorded.
    await page.evaluate((token) => {
      window.__tokenSeen = [];
      new MutationObserver(() => {
        if (document.body.innerText.includes(token)) window.__tokenSeen.push(performance.now());
      }).observe(document.body, { subtree: true, childList: true, characterData: true });
    }, body.token);
    const closedAt = Date.now();
    await page.getByRole("button", { name: "Close" }).click();
    await page.getByText(body.token).waitFor({ state: "detached" });
    // Opened again as soon as the page lets it, within the 350 ms.
    await page.getByRole("button", { name: "Add access token" }).first().click();
    const reopenedAfterMs = Date.now() - closedAt;
    const form = await visible(page.getByLabel("Title"));
    const titleValue = await page.getByLabel("Title").inputValue();
    await page.screenshot({ path: `${OUT}/C3-reopened.png` });
    await page.getByRole("button", { name: "Cancel" }).click();
    await sleep(500);
    const seenAgain = await page.evaluate(() => window.__tokenSeen.length);
    const held = await page.evaluate(() => ({
      html: document.documentElement.outerHTML,
      url: location.href,
      local: JSON.stringify({ ...localStorage }),
      session: JSON.stringify({ ...sessionStorage }),
    }));
    const secret = body.token.slice("nrv_pat_".length);
    const where = Object.entries({ ...held, console: log.join("\n") }).filter(([, text]) => text.includes(secret)).map(([k]) => k);
    const listed = await visible(page.getByText("deploy", { exact: true }));
    const neverUsed = await visible(page.getByText("Never used"));
    await page.screenshot({ path: `${OUT}/C3-list.png` });
    const q = quiet(w);
    const seen = {
      created: answer.status(),
      calendar,
      expiresAt: body.expired_at,
      shown,
      csvHasToken: csv.includes(body.token),
      csvHeader: csv.split("\n")[0],
      reopenedAfterMs,
      form,
      titleValue,
      tokenAppearedAfterClose: seenAgain,
      tokenHeldIn: where,
      listed,
      neverUsed,
      ...q,
    };
    report(
      "C3-token-once",
      answer.status() === 201 && calendar.beside && shown && seen.csvHasToken && form && titleValue === "" && seenAgain === 0 &&
        where.length === 0 && listed && neverUsed && isQuiet(q),
      seen
    );
    await context.close();
  });

  // C4: the security page shows each refusal under its field, and lists the account's tokens, which it revokes.
  await check("C4-security", async () => {
    const { page, context, w, tokens } = await signedIn("security");
    const pat = await call(base, "POST", "/api/v0/me/api-tokens", { label: "cli" }, tokens.access_token);
    await page.goto("/settings/profile/security");
    const seen = { listed: await visible(page.getByText("cli", { exact: true })) };
    const submit = async (current, next) => {
      await page.locator("#old_password").fill(current);
      await page.locator("#new_password").fill(next);
      await page.locator("#confirm_password").fill(next);
      const answered = answerTo(page, "POST", "/api/v0/me/change-password");
      await page.getByRole("button", { name: "Change password" }).click();
      return (await answered).status();
    };
    // Under a field: the message's box starts below the field's and within 60 px of it.
    const under = async (text, field) => {
      const message = page.getByText(text, { exact: false });
      if (!(await visible(message))) return { shown: false };
      const [m, f] = await Promise.all([message.boundingBox(), page.locator(field).boundingBox()]);
      return { shown: true, gap: Math.round(m.y - (f.y + f.height)), under: m.y >= f.y + f.height - 1 && m.y - (f.y + f.height) < 60 };
    };
    seen.wrongCurrent = { status: await submit("not the password", "An0ther-good-one"), ...(await under("The current password is wrong.", "#old_password")) };
    await page.screenshot({ path: `${OUT}/C4-wrong-current.png` });
    seen.common = { status: await submit(PASSWORD, "Password1!"), ...(await under("This password is too common", "#new_password")) };
    await page.screenshot({ path: `${OUT}/C4-common.png` });
    // Revoked from the security page.
    await page.getByText("cli", { exact: true }).hover();
    await page.getByRole("button", { name: "Revoke token" }).click();
    const revoked = answerTo(page, "DELETE", `/api/v0/api-tokens/${pat.body.id}`);
    await page.getByRole("button", { name: "Delete" }).click();
    seen.revoked = (await revoked).status();
    seen.empty = await visible(page.getByText("You have no personal access tokens."));
    seen.tokenAfter = (await call(base, "GET", "/api/v0/me", undefined, pat.body.token)).status;
    await page.screenshot({ path: `${OUT}/C4-revoked.png` });
    const q = quiet(w, ["422 POST /api/v0/me/change-password"]);
    report(
      "C4-security",
      seen.listed && seen.wrongCurrent.status === 422 && seen.wrongCurrent.under && seen.common.status === 422 && seen.common.under &&
        seen.revoked === 204 && seen.empty && seen.tokenAfter === 401 && isQuiet(q),
      { ...seen, ...q }
    );
    await context.close();
  });

  // C5: deactivating: the confirmation says what happens; the page is back at sign-in, which says so and
  // refuses the account; nerve users activate lets it sign in again.
  await check("C5-deactivate", async () => {
    const { page, context, w, address } = await signedIn("deactivate");
    await page.goto("/settings/profile/general");
    await page.getByRole("button", { name: "Deactivate account" }).click();
    const seen = { confirmation: await visible(page.getByText("ask an administrator of this server to reactivate it", { exact: false })) };
    await page.screenshot({ path: `${OUT}/C5-confirm.png` });
    const deactivated = answerTo(page, "POST", "/api/v0/me/deactivate");
    await page.getByRole("button", { name: "Confirm" }).click();
    seen.status = (await deactivated).status();
    await page.waitForURL((url) => url.pathname === "/" && url.searchParams.get("next_path") === "/settings/profile/general");
    seen.url = page.url().replace(base, "");
    seen.toast = await visible(page.getByText("Your account is deactivated."));
    seen.record = await page.evaluate(() => localStorage.getItem("nerve.auth"));
    await page.screenshot({ path: `${OUT}/C5-signed-out.png` });
    seen.signIn = await fillSignIn(page, address);
    seen.alert = await alertText(page);
    await page.screenshot({ path: `${OUT}/C5-refused.png` });
    seen.activate = nerveUsers(db, ["activate", "--email", address]).trim();
    seen.signInAgain = await fillSignIn(page, address);
    await page.waitForURL("**/onboarding", { timeout: 10_000 }).catch(() => {});
    seen.after = new URL(page.url()).pathname;
    await page.screenshot({ path: `${OUT}/C5-back.png` });
    const q = quiet(w, ["403 POST /api/v0/auth/login"]);
    report(
      "C5-deactivate",
      seen.confirmation && seen.status === 204 && seen.toast && seen.record === null && seen.signIn === 403 &&
        seen.alert === "This account is deactivated." && seen.signInAgain === 200 && seen.after === "/onboarding" && isQuiet(q),
      { ...seen, ...q }
    );
    await context.close();
  });

  // C6: while nerve's instance settings fail, the page says nerve cannot be reached for now (not that it did not
  // start), and carries on by itself once they load.
  await check("C6-maintenance", async () => {
    const context = await browser.newContext({ baseURL: base });
    const page = await context.newPage();
    await page.route("**/api/v0/instance", (route) => route.fulfill({ status: 503, body: "" }));
    await page.goto("/");
    const seen = {
      title: await visible(page.getByText("Nerve cannot be reached right now")),
      text: (await page.locator("body").innerText()).split("\n").filter((l) => l.trim()).slice(0, 6),
    };
    await page.screenshot({ path: `${OUT}/C6-maintenance.png` });
    await page.unroute("**/api/v0/instance");
    const startedAt = Date.now();
    seen.carriesOn = await visible(page.getByLabel("Email", { exact: true }), 30_000);
    seen.afterMs = Date.now() - startedAt;
    await page.screenshot({ path: `${OUT}/C6-back.png` });
    report("C6-maintenance", seen.title && seen.carriesOn, seen);
    await context.close();
  });
} finally {
  await browser.close();
  await nerve.stop();
  await dropDatabase(dbName);
}

const failed = results.filter((r) => !r.ok);
console.log(`${results.length} checks, ${failed.length} failed${failed.length ? `: ${failed.map((r) => r.id).join(", ")}` : ""}`);
process.exit(failed.length ? 1 : 0);
```

### 运行输出（`6097b01`）

```text
PASS C1-four-tabs {"Profile":{"url":"/settings/profile/general","shown":true},"Preferences":{"url":"/settings/profile/preferences","shown":true},"Security":{"url":"/settings/profile/security","shown":true},"Personal Access Tokens":{"url":"/settings/profile/api-tokens","shown":true},"securityTokens":true,"oldApi":[],"apiFailures":[],"pageErrors":[]}
  (C1-four-tabs: 1015 ms)
  C2: theme-en
  C2: week-en
  C2: timezone-en
  C2: theme-zh
  C2: timezone-zh
  C2: week-zh
PASS C2-dropdowns-beside {"theme-en":{"beside":true,"vertical":4,"edges":0,"list":{"x":969,"y":253,"w":192,"h":181},"button":{"x":1066,"y":211,"w":95,"h":38},"escapesToClose":1,"reopensAtFirstClick":true},"week-en":{"beside":true,"vertical":4,"edges":0,"list":{"x":969,"y":509,"w":192,"h":246},"button":{"x":1073,"y":468,"w":88,"h":38},"escapesToClose":1,"reopensAtFirstClick":true},"timezone-en":{"beside":true,"vertical":4,"edges":0,"list":{"x":873,"y":373,"w":288,"h":249},"button":{"x":1092,"y":332,"w":69,"h":38},"escapesToClose":1,"reopensAtFirstClick":true},"languageStatus":200,"chinese":true,"theme-zh":{"beside":true,"vertical":4,"edges":0,"list":{"x":969,"y":253,"w":192,"h":181},"button":{"x":1069,"y":211,"w":92,"h":38},"escapesToClose":1,"reopensAtFirstClick":true},"timezone-zh":{"beside":true,"vertical":4,"edges":0,"list":{"x":873,"y":373,"w":288,"h":249},"button":{"x":1092,"y":332,"w":69,"h":38},"escapesToClose":1,"reopensAtFirstClick":true},"weekTitleZh":true,"weekTitleEnGone":true,"week-zh":{"beside":true,"vertical":4,"edges":0,"list":{"x":969,"y":509,"w":192,"h":246},"button":{"x":1080,"y":468,"w":81,"h":38},"escapesToClose":1,"reopensAtFirstClick":true},"oldApi":[],"apiFailures":[],"pageErrors":[]}
  (C2-dropdowns-beside: 1902 ms)
PASS C3-token-once {"created":201,"calendar":{"beside":true,"vertical":4,"edges":0,"list":{"x":436,"y":321,"w":308,"h":336},"button":{"x":436,"y":289,"w":83,"h":28}},"expiresAt":"2026-10-10T00:41:42Z","shown":true,"csvHasToken":true,"csvHeader":"Title,Description,Expiry,Secret key\r","reopenedAfterMs":350,"form":true,"titleValue":"","tokenAppearedAfterClose":0,"tokenHeldIn":[],"listed":true,"neverUsed":true,"oldApi":[],"apiFailures":[],"pageErrors":[]}
  (C3-token-once: 2876 ms)
PASS C4-security {"listed":true,"wrongCurrent":{"status":422,"shown":true,"gap":16,"under":true},"common":{"status":422,"shown":true,"gap":16,"under":true},"revoked":204,"empty":true,"tokenAfter":401,"oldApi":[],"apiFailures":[],"pageErrors":[]}
  (C4-security: 1269 ms)
PASS C5-deactivate {"confirmation":true,"status":204,"url":"/?next_path=%2Fsettings%2Fprofile%2Fgeneral","toast":true,"record":null,"signIn":403,"alert":"This account is deactivated.","activate":"activated deactivate-mukivh8x-5@example.com: 0 API tokens are usable again","signInAgain":200,"after":"/onboarding","oldApi":[],"apiFailures":[],"pageErrors":[]}
  (C5-deactivate: 1720 ms)
PASS C6-maintenance {"title":true,"text":["Nerve cannot be reached right now","The page could not load this server's settings. It tries again by itself and carries on once the server answers."],"carriesOn":true,"afterMs":5312}
  (C6-maintenance: 5523 ms)
6 checks, 0 failed
```
