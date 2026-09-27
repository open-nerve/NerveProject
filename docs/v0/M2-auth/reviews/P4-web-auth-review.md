# M2/P4 前端认证：评审记录

| 项 | 内容 |
|---|---|
| Phase | M2/P4 `web-auth` |
| 日期 | 2026-09-27 |
| 结论 | **通过**（整分支评审提出的问题已在一轮修复中处理完，修复的范围评审通过，留下的一个 Minor 由控制者接受） |
| spec / plan | [spec](../specs/P4-web-auth.md) / [plan](../plans/P4-web-auth.md) |
| 分支 | `worktree-m2-p4-web-auth`，从 `main` 的 `f27434c`（P3b 的合并）分出 |

## 1. 评审方式

- **负责人的决定**：新手引导删掉角色和用途两步（决策点 4，A）；退出只结束当前这一处登录；prod 默认关闭注册（决策点 2，C），登录页的注册链接跟随实例的 `signup_enabled`。
- **spec 和 plan 由架构子任务产出**（`900a3b6`）：
  - 先在仓库外把整个 P4 做出来（原型），plan 中的代码从原型原样拼入，72 个差异块用 `patch -p1` 逐个核对。
  - 再从 `f27434c` 的副本出发按 plan 逐个重放 14 个 Task，结果与原型逐文件相同（2,626 个文件）。
  - 原型中 119 个变异有 117 个被发现，没被发现的两个是等价或另有一道守卫挡住（spec 附录 A）。9.6 的浏览器核对 C1–C11 在原型上全部通过。
- **执行前的核对**：
  - spec 第 3 节的 15 项差异全部接受。改了 M2 设计文字的几项在控制者的提交 `e3cfa99` 中同步（9.4、9.6、3.20、12 节中 P4 和 P5 的关闭和 P5 的交付物 1）。
  - 执行前的扫描（opus，只读）：51 行共用文件和接口的 Task 对，14 行逐 Task 的自洽核对；72 个差异块依次都能干净地应用，oxlint 上限、关键词、单元测试数、`schema.gen.ts` 的 SHA-256、两处 `pnpm-lock.yaml` 的差异、CSP 的哈希都能复现。提出 F1–F18，其中两项是高：F1、F2（第 3 节）。控制者逐项裁定，带进对应 Task 的派发。
- **逐个 Task 评审**：14 个 Task，加上控制者在执行中加的 Task 10b，各由一个实现子任务写入代码，再由独立的评审子任务（sonnet）检查"是否符合 spec"和"代码质量"。
  - 实现者按要求对自己的代码做变异核对。15 个 Task 的实现者都带着疑问交回（plan 的缺陷、测试没咬住的性质、超出 brief 的判断），控制者逐项裁定（第 3 节）。
  - 逐个 Task 评审要求返工的有两次：Task 2（I1、I2：退出和结束会话会作用于另一个会话）、Task 5（`USE_CASES` 成了死代码而 knip 看不到）。其余评审的 Minor 由控制者分拣：当场修、放进修复轮，或者接受。
  - 下一个 Task 的实现与上一个 Task 的评审并行，同一时间只有一个实现者提交。
- **整分支评审**（opus，`f27434c..2862b63`）：结论为"修复后可合并"：0 个 Critical、1 个 Important、6 个 Minor（第 4 节）。评审跑了 web 的 116 个和 `@nerve/utils` 的 25 个单元测试、`make lint-web`、`make knip`、`make lint-go`、`webui` 和 `bootstrap` 的 Go 测试，核对了 `schema.gen.ts` 的 SHA-256，扫了被删代码留下的 `@nerve/*` 孤立导出，另在副本中跑了一个探测测试。与它并行的 Task 14 评审（sonnet）只对实现者的报告提了一个 Minor，提交的文档没有需要改的。
- **修复**：
  - 一轮完成，8 个提交，`9ff0be3` 至 `fa4579a`（基于 `2862b63`），包括 I1、M1、M2、M4–M6（M3 并入 I1）和逐个 Task 评审时停放的各项（第 4 节）。
  - 控制者另有一个文档提交 `e9ad0f1`，改正实现者报出的两处过时文字。
  - 之后对 `2862b63..e9ad0f1` 做了一次范围评审（sonnet）：通过，留下的一个 Minor 由控制者接受。
- **9.6 的浏览器核对**：由控制者在分支的代码上重跑（不是原型的代码），见第 2 节第 4 行和附录。
- **持续集成**：推送后通过 GitHub 公开接口（curl 和 jq）确认结果，见第 2 节末。

## 2. 验收标准核对（spec 第 4 节）

| # | 标准 | 结果 |
|---|---|---|
| 1 | `git grep -n -E "csrfmiddlewaretoken\|X-CSRFTOKEN" -- web` 没有输出 | **通过**（控制者在 `e9ad0f1` 上执行）。再宽一些的 `csrf`（不区分大小写）、`error_code`、`is_self_managed` 也没有；`withCredentials: true` 只剩 `web/packages/services/src/api.service.ts:23`，按 M2 设计 7.9 在 P5 随 PAT 的重写删除 |
| 2 | A1–A6、A10、A15 的页面版本（A4 有、没有 `navigator.locks` 两遍，A6 含两种切换账户）和 S2 通过；此前的故事仍然通过（`make e2e` 共 35 个测试） | **通过。**<br>修复轮之后本机 `make e2e` 35 个全部通过；页面故事和 S2 `--repeat-each=3` 48 个通过。<br>Task 13 之后 A4–A6 `--repeat-each=10` 60 个通过，页面故事和 S2 ×4 64 个通过。<br>每个页面故事都有变异证明能失败：<br>- `webLock` 不加锁，A4 有锁的一遍失败；<br>- 不取租约，A4 没有锁的一遍失败；<br>- `handleStorageChange` 忽略新的 `login_id`，A6 两种切换都失败；<br>- 漏掉一个内联脚本的哈希、所有哈希都错，S2 失败；<br>- `StoreProvider` 不订阅，5 个页面故事失败。<br>持续集成的 e2e 任务在 `2862b63` 和 `e9ad0f1` 上都通过（第 2 节末） |
| 3 | 令牌管理器、锁、中间件的单元测试通过；去掉锁、去掉写回前的 `login_id` 核对、429 或 5xx 结束会话、去掉 8 秒超时、去掉租约的锁，每个变异都让至少一个测试失败 | **通过。**<br>认证的核心 99 个单元测试：锁 9；令牌管理器 32、会话变化 14、多标签页 20；中间件 20、绑定会话 4。spec 原写 72 个，执行中的裁定加了 27 个。<br>五个变异由 Task 1、2 的实现者在分支的代码上重跑，都失败：<br>- 续期不经过锁，18 个测试失败；<br>- 写回之前不再读记录，"另一个标签页登录之后发出的请求"失败；<br>- 429 结束会话，2 个失败；<br>- 5xx 结束会话，7 个失败；<br>- 8 秒不中止，续期和退出的两个 8 秒测试失败；<br>- 租约不互斥（不管是否取到都运行任务），锁的 5 个测试失败。<br>Task 2 的评审另在副本中复现了"5xx 结束会话"（7 个失败）和 F1 的回退（2 个失败）。<br>Task 2 对令牌管理器的 26 个变异中 25 个失败，存活的一个是"假浏览器对没有变化的写入不发事件"，测不出来是预期的：它只防止以后的测试依赖浏览器不会发出的事件。<br>修复轮的 30 个变异中 29 个失败，1 个等价 |
| 4 | 9.6 中 P4 的浏览器核对在合并前的代码上重跑，写进 review；M0-P5"同一层"的关闭理由写进 review | **通过。**<br>控制者在 `e9ad0f1` 的构建上跑了 15 个核对（C1–C11，C4、C6 各三个），全部通过，见附录。修复轮之前在 `2862b63` 上也全部通过。<br>关闭理由见第 6 节 |
| 5 | `make lint`、`make test`、`make gen-check`、`make knip`、`make test-web`、`make e2e` 通过；web 的 oxlint 上限 551 | **通过。**<br>修复轮之后本机 `make lint-web`（57 条规则，3 个例外，没有命中；oxlint 551）、`make knip`、`make test-web`、`make gen-check`、`make e2e` 通过。<br>Go 从 Task 11（`a3dd116`）之后没有改动：Task 11 的 `make lint-go` 两段 `0 issues.`、`make test` 32 个包通过，整分支评审在 `2862b63` 上又跑了 `make lint-go`。<br>持续集成在 `e9ad0f1` 上跑了 `gen-check-go`、`lint-go`、`test`、`gen-check-web`、`lint-web`、`test-web`、`knip`、`build`、`e2e`，都通过 |
| 6 | 3.20 中 P4 的各行、8.7 中 P4 的 README 内容在同一次合并中写好；交接按 spec 第 7 节处理 | **通过。**<br>Task 14 写了总体设计 4.3、前端改动清单 3.1、3.2、README 和五份交接的处理结果；修复轮改正了 M2 设计 3.20 的 P4 一行，前端改动清单 3.2 改为进行中。<br>Task 14 的评审逐句抽查了约 25 处说法，都与代码相符；整分支评审和修复轮的范围评审又各自核对。<br>交接见第 6 节 |

持续集成：
- 修复轮之前，`2862b63` 的分支 run 36313441460：web、server、e2e 三个任务都通过。这是 P4 的 e2e 第一次在 ubuntu 上运行，原型只在 macOS 上跑过（spec 第 6 节的风险）。
- 修复轮之后，`e9ad0f1` 的分支 run 36317591828：三个任务都通过。
- 第一次合并（`2819aea`）之后，main 的 run 36319447105 失败，失败在 web 任务的单元测试。
  - 原因：修复轮新写的 `store-context.test.ts` 每个测试都重新加载模块。第一个测试在自己 5 秒的期限里第一次导入 stores，要编译几百个模块（本机 1.2 秒），在繁忙的 runner 上超时。超时之后它的等待还在推进假时钟，带坏了下一个测试。同一棵树此前在分支上通过了两次。
  - 修正（`3c4a183`，控制者）：一个有自己期限的 `beforeAll` 先导入一次 stores，同时替换掉 `api-client` 和 `store-context`，不运行应用代码；每个测试的 `load()` 只重新求值已编译的模块，只要几毫秒。
  - 证据：
    - 测试期限压到 500 毫秒时，原来的文件恰好失败这两个测试，新的通过（第一个测试从 1190 毫秒降到 31 毫秒）；
    - web 的 129 个单元测试在 1 秒的期限下全部通过；
    - "每次通知都换代"的变异仍让第一个测试失败。
  - 修正的范围评审（sonnet）通过。评审在副本中：
    - 用 500 毫秒的期限复现了原来的两个失败，失败信息与持续集成上的相同；
    - 原地重建、不通知两个变异都让测试失败；
    - 确认预热不建 `TokenManager` 和 `RootStore`，每个测试的 `resetModules` 丢掉预热的模块实例。
  - `root.store.test.ts` 在文件顶层导入 stores，不受这个问题影响，但原来没有写明为什么放在顶层；评审指出后加了一句注释（`fd77988`）。
  - 分支 run 36319783073 通过；再次合并后 main 的持续集成在合并时记录。

## 3. 执行中的决定

spec 第 3 节的 15 项差异在执行前全部接受，没有一项改变 M2 设计的架构、前后端契约、模块规则或负责人的决定。执行中加了第 16 条（Task 10b）和第 17 条（修复轮 FW1）。

执行前的扫描提出的问题（F1–F18）和控制者的裁定：

| # | 问题 | 裁定 |
|---|---|---|
| F1（高） | 以 X 发出的请求，续期排在另一个标签页以 Y 登录之后：plan 的代码在锁下核对的是标签页此刻的会话，于是以 Y 的令牌发出 | 续期记下它为哪个会话开始；锁下读到的记录属于另一个会话时，跟随记录并以 `SessionChangedError` 结束，不续期。探测写成测试（`token-manager.session-change.test.ts`，两种锁），加变异；spec 2.5 一句 |
| F2（高） | X 的请求得到 401 时标签页已换成 Y：plan 的中间件在 Y 下续期、以 Y 的令牌重发，第二个 401 还会删掉 Y 的记录 | 中间件随请求记下 `login_id`，401 回来时会话已变就不续期、不重发（后来细化为以 `SessionChangedError` 失败，见 Task 4 一行）；测试和变异；spec 2.6 一句 |
| F3 | "会话暂不可用"的文案承诺自动重试，取账户失败的那一处并不重试 | 只有确有自动重试时才这样说（`autoRetry`）；取账户那一处用不承诺重试的文案，保留"重试"按钮 |
| F4、F11 | 总体设计 4.3 和 README 夸大了"会话暂不可用"出现的场合，漏了"重发后仍是 401 也结束会话"；README 说新手引导只剩资料一步 | Task 14 按代码改写 |
| F5 | 三处 `waitForResponse` 没有期限 | 各加 10 秒 |
| F6 | "每个写 `nerve.auth` 的测试都核对持锁"对续期的写回不成立 | 续期写回的测试断言每次写入都持锁 |
| F7 | A6 退出后查令牌的断言可能空转 | 先断言截下的令牌非空、退出前得到 200 |
| F8 | A2 的"不发请求"紧跟在横幅之后读 | 在故事结尾数注册请求的确切个数 |
| F9 | e2e 中逐字重复的逻辑（等 `PATCH`、新手引导步骤的 SQL、截取令牌） | Task 13 加第二份时抽进 fixture（`e2e/fixtures/onboarding-pages.ts`），并把 Task 12 的故事改用它们 |
| F10 | zh-CN 的标题与 spec 不同 | 用 spec 的"暂时无法连接服务器" |
| F12 | 测试标题"64 个表情"与实际不符 | 改标题 |
| F13 | "没有网络"用状态 0 的响应模拟，真实浏览器是 fetch 被拒绝 | 假 nerve 加 `fail()`；Task 2 测续期和退出、Task 4 测中间件在被拒绝的 fetch 上的行为 |
| F14 | 假浏览器对 `storage` 事件何时到达的注释不对 | 改成它实际的做法 |
| F15 | Task 6 的完成线"未登录时只请求 `/api/v0/instance`"要到 Task 7、9 才成立 | Task 6 不核对，Task 12 核对 |
| F16 | Task 14 没有跑 `make knip`、`make test-web` | 跑（Global Constraints 要求） |
| F17、F18 | spec 第 3 节第 8 条；e2e 中 `signInPath` 的独立副本 | 保持（有意为之） |

逐个 Task 的决定。除特别注明的以外，都在这个 Task 的评审之前提交：

| Task | 决定 | 原因 |
|---|---|---|
| 1 锁 | `webLock` 的测试由测试自己授予锁，记下任务运行时是否持锁 | plan 的测试在任务跑到锁外时仍然通过（实现者的变异）；改后 5 个 `webLock` 变异都被发现 |
| 2 令牌管理器 | 接受 F1、F6、F13；假浏览器对没有变化的写入不发 `storage` 事件（与浏览器一致）；`token-manager.test.ts` 514 行，与 472 行同样接受（一个类的单标签页测试，分三组，拆开要重复假实现的搭建）。**评审的 I1、I2（第二轮修正，`806ff03`–`9ed06b2`）**：`signOut` 在调用时记下 `login_id`，`endSession(loginId)` 接收被拒请求所属的会话；两者在锁下只处理自己那个会话的记录，否则跟随记录、不退出任何人。spec 2.5 把"改变会话的操作只作用于它被调用时的会话，并读当前的记录"写成一条规则 | 标签页显示 X、它的退出排在另一个标签页以 Y 登录之后时，原来的代码退出了 Y 并删掉 Y 的记录；重发被拒之后、`endSession` 拿到锁之前换成 Y 时同样。这与 F1 是同一类竞争 |
| 3 多标签页 | `handleStorageChange()` 不再接收事件的 `newValue`，读当前的记录；两处断言等到假浏览器强制送达事件之后再读 | 浏览器可能在锁已交给后来的写入者之后才送达事件：刚登录的标签页会被切回一个已不存在的会话，刚退出的会被重新登录。只有 `hold()`/`deliver()` 强制浏览器的顺序时，测试才看得出这个缺陷（去掉它们时 0/67 失败） |
| 4 中间件 | F2；**一个信号**：请求被会话变化截断时，调用方只会收到 `SessionChangedError`，不会收到另一个会话的 401；`endSession(loginId)` 返回布尔值，中间件在它为假时抛出这个错误 | 两种截断（401 之前换了会话、续期等锁时换了会话）原来一个交回 401、一个抛错；调用方会把别的会话的 401 当成当前会话失败。标签页还没听说的变化只有 `endSession` 的布尔值看得到（C4-3、C4-4 证明状态检查看不到） |
| 5 新手引导 | 删除 `USE_CASES`（评审发现）；此后每个实现者对自己删除的代码逐个检查 `@nerve/*` 的导出是否成了孤儿；A10 覆盖注册后应用内进入的一条路（交 Task 12） | `@nerve/*` 的包声明了 `"exports"`，knip 把它们所有的具名导出都当作公开接口，看不出死导出。spec 附录 A 原来把 `page.tsx` 的预取变异算作等价，只在整页加载时成立 |
| 6 实例、`ApiError` | 加"problem 没有 `detail` 时用 `title`"的测试 | 这个变异原来存活 |
| 调用方的规则 | `SessionChangedError` 从不当作认证失败：不退出、不跳转。后台加载（`fetchCurrentUser` 之类）忽略它，由重建的 stores 为新会话重新取数；用户发起的操作被它截断时可以保留错误提示（操作确实没有做） | Task 4 的"一个信号"之后，调用方需要一条统一的处理方式 |
| 7 账户 store | 停用弹窗的错误提示显示错误的消息；删除用户资源的上传路径（`uploadUserAsset` 等四个方法、`isUserAsset`、`isProfileCover`、`EFileAssetType.USER_AVATAR`/`USER_COVER`），M5 按新协议加回。**写在 `await` 之后的请求以别的账户发出**，交给 Task 10b | 按负责人"删除不用的代码"的规则。第三条：在会话 X 中开始的操作，`await` 之后发出的下一个请求在标签页跟随 Y 之后带着 Y 的令牌（资料步骤以 X 的令牌保存名字，之后的 `onboarding_step` 写进 Y）；逐处比较 `loginId` 只能补一处 |
| 8 资料 store | 完成新手引导只发一个 `PATCH`（不再同时发它取代的步骤变化）；四处 `last_workspace_id` 的写入是尽力而为，各写一句理由；修正 propel 的 `setPromiseToast`：它丢掉了 Base UI 重新拒绝的 Promise，每个失败的 Promise 提示都成为未处理的拒绝 | 两个写入并发时 `updateUserProfile` 保留最后到达的响应，`is_onboarded` 可能在本地仍为假；P4 让 `updateUserProfile` 抛错之后，所有调用方都要处理拒绝（16 个调用方逐个列出）。`invitations/page.tsx` 中没有 `return` 的 `fetchWorkspaces().then` 是 M2 设计 13.2 交给 M3 的那一类，留给 M3 |
| 9 包装 | F3、F10；`fetchCurrentUser` 吞掉 `SessionChangedError`，SWR 的键带上 `loginId` | 截断的任何时刻都不出现错误提示 |
| 10 登录页、注册页 | `PROBLEM_MESSAGES` 的注释只说它包含的内容（带令牌的操作可能答 `unauthorized`，落到通用文案） | 原注释说"nerve 回答的每一个问题码" |
| 10b（控制者加） | **每一代 stores 一个绑定会话的客户端**：`apiFor(loginId)`；中间件属于一个会话，标签页已不在这个会话时请求在发出之前以 `SessionChangedError` 失败；`UserService` 从构造函数接收客户端，模块级的 `userService` 删除；从没有会话变成有会话时也重建 stores（spec 第 3 节第 16 条） | 第 7 行的第三条，以及新手引导的创建工作区、`/create-workspace`、`/invitations`、新手引导根组件的后续写入，都是"`await` 之后以别的账户写入"。要一个机制，不逐处修补。这是 M2 设计 7.1、7.5"标签页不以错误的身份写入"的实现方式，不改架构 |
| 11 CSP | 页面以外的响应不带任何 CSP（不只是"不带页面的策略"）；内联脚本按原文取哈希（不先去空白）；S2 要能被错误的哈希弄失败（交 Task 12） | 原来的测试放过"`/api/` 带另一种 CSP"的变异；去空白的写法今天等价，但与浏览器的算法不同；`make e2e` 原来看不出所有哈希都错的构建 |
| 12 e2e 第一部分 | S2 不用 `networkidle`，等 `/api/v0/instance` 的回答和登录按钮；`expectQuietConsole`：先用一条探测消息证明控制台的监听接上了，再断言没有别的错误和警告；唯一的第三方警告（is-emoji-supported 在 Chromium 上的 `willReadFrequently` 提示）按原文列名、恰好一次 | 页面从不读回答的请求让 S2 挂到 30 秒超时；控制台警告原来记下却没人断言。未登录的深层链接在包装跳转之前加载编辑器的代码包，这是 M4 的代码拆分事项（第 6 节） |
| 13 e2e 第二部分 | `watchPage` 按状态码判断请求是否得到回答（不看 Chromium 的 `net::ERR_ABORTED`）；`expectQuietConsole` 接收 `{ warnings, errors }`，A5 列名 Chromium 对 401 的一条控制台错误；A4 的锁变异在保存的 10 秒等待处失败（重复使用检测先结束了会话），不在代数断言处 | 204 的响应体 openapi-fetch 不读，Chromium 报成被中止；A4：性质被违反就被发现，plan 的等待也是这样 |
| 14 文档 | 保留总体设计 4.3 "不以别的会话的身份发请求"一条；前端改动清单 3.2、M2 设计中过时的几行放进修复轮；本评审记下 CSP 分层的理由和在分支代码上重跑的浏览器核对 | 这一条属实，也是以后的 M 的 stores 要遵守的规则 |

控制者在过程中的其他决定：
- **修正落在写出最终版本的 Task 里**：F14 原本放在 Task 1，打乱了 Task 2 对同一文件的差异块，控制者为 Task 2 另备了调整后的差异块。此后的修正都先按 plan 的文件表找到最后改这个文件的 Task。
- **没有 DOM 的单元测试**：组件和页面的测试要 jsdom，是新的 npm 包，Global Constraints 不允许；页面由 e2e 和浏览器核对覆盖。Task 8、9 的实现者在仓库外用 jsdom 核对过（`setPromiseToast`、包装的各状态），不进仓库。
- **规则的偏离**：实现者和评审者多次用只读命令违反 shell 规则（`cd` 连用、git 的管道、`git -C`），都没有改动文件，记在进度表中。另有一次控制者自己的会话中残留了两个 `tail -f` 和一个挂住的 vitest，已停掉；此后的派发都要求结束时不留后台进程。

## 4. 整分支评审的发现与处理

| 发现 | 级别 | 处理 |
|---|---|---|
| I1 一代 stores 没有真正隔开：`resetOnSignOut` 在唯一的 `RootStore` 上原地重建，旧一代的 store 在 `await` 之后经 `this.rootStore` 找兄弟 store 时，找到的是新一代的，带着新会话的客户端（`global-view.store` 的 `updateGlobalView`，`module`、`cycle`、`project` 的收藏写入等）；`RootStore` 不是 observable，包装之外的组件在下一次渲染之前还拿着旧一代。M2 中没有这样的请求，今天无害；但 M3 起工作区的 stores 会重现 Task 7 第三条的"写进别的账户" | Important | **已修（FW1，`9ff0be3`、`e03015c`）。** 每个会话一个 `RootStore`：新会话建新的 `RootStore(apiFor(next), 旧的)`，只沿用 `instance`、`router`、`theme`；`store-context.tsx` 的 `startSession` 设主题和语言的默认值、换进导出的活绑定 `rootStore`、通知监听者；`StoreProvider` 用 `useSyncExternalStore` 提供当前的 `RootStore`，每个用 stores 的组件都以新的一代重新渲染。旧的 stores 留着旧的 `RootStore`，它的客户端在发出之前就拒绝。`resetOnSignOut`（原本停放的改名）和 `initializeStore` 里执行不到的分支（M3）一起删除。三处没有自己 `RootStore` 的代码（`command-palette.store`、`issue.store`、`issue-layouts/utils.tsx`）只做同步读取，读活绑定。<br>控制者选择在根上改、不给 M3 立约定：Plane 的 stores 在 `await` 之后经 `this.rootStore` 找兄弟 store 的地方很多，逐处遵守的约定只是补丁。代价是会话变化时整个应用重新渲染一次。<br>测试：`store-context.test.ts` 4 个（每个测试重新加载模块，令牌管理器、中间件是真的，nerve 是假的）、`root.store.test.ts` 2 个。14 个变异都失败，包括原地重建（两种写法）、不沿用 `instance`/`router`/`theme` 之一、沿用一个账户的 store、同一会话内也换代、不通知、活绑定换成快照、换代推迟到组件之后、新的 `RootStore` 用旧会话的客户端。`StoreProvider` 不订阅的变异（只有页面看得到）让 5 个页面故事失败（A1、A3、A10 的应用内进入、A6 的两种切换） |
| M1 标签页跟随另一个会话时仍带着旧会话的退避：第一次续期得到 503 和 `Retry-After: 30` 之后跟随 Y，Y 的请求在 X 的退避结束之前不会发给 nerve | Minor | **已修（FW2，`681fa2f`）。** 换会话时清掉失败次数和下次重试的时刻（`#leaveSession`）。评审的探测写成测试；另加"Y 的第一次失败退避 1 秒而不是 2 秒"，因为只清时刻、不清次数的变异在探测下存活。`#signedOut` 不清退避的变异是等价的：退出之后什么都不读它，下一个会话由 `#switchTo` 或登录的 `#keep` 开始，两者都清 |
| M2 `UserStore.isLoading` 只写不读 | Minor | **已修（FW4，`bab2844`）。** 字段、接口中的一项和两处只写它的 `runInAction` 删除 |
| M3 `store-context.tsx` 中执行不到的分支 | Minor | **已修**，并入 FW1 |
| M4 spec 的文件数和单元测试数过时 | Minor | **已修（FW8，`fa4579a`）。** 2.1 写 plan 的表加上控制者的补充；第 4 节写 99 个 |
| M5 两处删除没有关键词规则：用户资源的上传（`cb2487c`）、Plane 当前用户的地址 `/api/users/me/` 本身 | Minor | **已修（FW7，`8b92805`）。** 新规则 `plane-user-assets`；`plane-user-urls` 加上"`/api/users/me/` 后面紧跟引号"。在 `f27434c` 的 web 源码上，`plane-user-urls` 命中 8 行（当前用户的地址正是 `user.service.ts` 的 3 个调用），`plane-user-assets` 命中 14 行；在现在的源码上没有命中。规则 56 → 57 条 |
| M6 标签页已经跟随某个会话时，`#switchTo` 仍丢掉这个会话有效的访问令牌，多续期一次 | Minor | **已修（FW3，`681fa2f`）。** 记录属于标签页此刻的会话时直接返回：留着访问令牌，不再通知。实现者的判断（控制者接受）：会话处于"暂不可用"或"启动中"时同样原样保留，不改成"已登录"：改成"已登录"会清掉自动重试的计时，而跟随自己会话的记录并不说明 nerve 已经可达；应用代码今天走不到这种情形，由测试钉住 |

评审"不作判断"的清单，控制者照评审的写法接受，每一条都有 spec 或进度表中的依据：租约不是原子的、标签页冻结超过 10 秒（spec 第 6 节，§16）；续期进行中关闭或刷新标签页（设计 3.5）；`UserStore` 用模块级的 `tokenManager`、`AuthService` 用模块级的 `publicClient`（spec 2.7、2.8）；`renew()` 已退出的分支从中间件走不到（spec 第 3 节第 4 条，有测试）；`deactivateAccount`、`signOut` 读标签页此刻的会话（Task 10b 的裁定：只差一次渲染）；`OnboardingRoot` 的邀请总是空的（2.11，M3 加回）；没有网络时登录显示通用文案（2.10）；CSP 的 `worker-src`、`frame-src` 落回 `'self'`（M4、M7）；最后一次登录替换另一个标签页的会话而不退出它（设计 7.1）；general 表单的 `maxLength={50}`（Plane 原有，P5 的页面）；改到的文件中的 oxlint 警告（P5）。其中"`navigator.locks` 之下 localStorage 在渲染进程之间的一致性"一条，按评审的建议写进 M2 设计 §16（第 7 节）。

放进修复轮的另外几项，都是逐个 Task 评审中控制者裁定要改、但不属于当时那个 Task 的文件：
- **`auth-middleware.ts` 的说明**（`7336040`，修复轮复核）：请求自己的 401 看标签页的状态，续期在锁下发现变化，重发的 401 之后由 `endSession(loginId)` 的布尔值决定，包括标签页还没听说变化的情形。只改注释。
- **表单画不出来的字段错误**（`ecbae8b`，Task 10 评审）：`needsErrorBanner(error, fields)` 在错误没有字段、或者点名了表单上没有的字段时显示表单上方的提示；3 个测试，3 个变异都失败。今天的登录、注册只点名 `email`、`password`，走不到。
- **文档**（`fa4579a`）：
  - M2 设计的 3.20 中 P4 一行（"重发后仍是 401"也结束会话）；
  - 7.1 的位置一项（`publicClient`、令牌管理器、`apiFor`），zh-CN 标题，界面表加上取账户失败的一行，结束会话一项（新的 `RootStore`）；
  - 13.1 中 S2 的一行（不用 `networkidle`，等实例的回答和登录按钮，原因）；
  - 13.2 加两行、延伸一行（第 6 节）；
  - §16 加一行；
  - 前端改动清单 3.2 改为进行中，写明 P4 做了什么、剩下什么归谁。
- **控制者补的文档**（`e9ad0f1`）：实现者报出的两处过时文字，控制者改正：M2 设计 7.9 的规则表加上修复轮的两条；13.1 中 M1-P4 的 `security.tsx` 一行和 12 节 P5 的交付物 1 写明修改密码的错误处理已在 P4 完成（Task 10 随删除 CSRF 改了这个文件），P5 做 PAT 列表。另外：13.2 的"M3 及以后"一行写明旧一代在沿用的 `router` 上注册的反应不随它释放（第 7 节）；spec 2.13 中两条规则的命中数原来写反了，改正。

修复轮之后：
- web 单元测试 116 → 129 个，其中认证的核心 99 个；
- `make lint-web`：57 条规则、3 个例外、没有命中，oxlint 仍是 551；
- `make knip`、`make test-web`、`make gen-check` 通过；
- `make e2e` 35 个通过；页面故事和 S2 `--repeat-each=3` 48 个通过；
- 实现者的 30 个变异中 29 个失败，1 个等价（上表 M1）。

修复的范围评审（sonnet，`2862b63..e9ad0f1`）：终审的每一项都已处理，没有回退或覆盖的缺口。评审在副本中自己重跑了：
- FW1 的原地重建、同一会话内也换代、不通知、活绑定换成快照四个变异，都失败；评审自己加的"从没有会话变成有会话时不换代"，也失败；
- FW2 的"不清退避"（4 个测试失败）、FW3 的"总是清掉访问令牌"（10 个失败）；
- FW7 两条规则在 `f27434c` 的 `user.service.ts`、`instance.service.ts`、`file.service.ts` 上的命中；
- 文件数（改动或新增 123 个，删除 9 个）和单元测试数（认证的核心 99 个，web 129 个）；
- `make test-web`、`make lint-web`（不用缓存）。

另抽查了 18 句新写的文字。

它只提了一个新的 Minor：`startSession` 先改记下的 `loginId`，再建新的 `RootStore`，建的时候万一抛错，就不会再重试。控制者接受，不改：20 个 store 的构造函数都不做 I/O，走不到；真抛错时应用无论哪种顺序都已出错。

## 5. 最终实现与 plan 的差异

plan 是执行记录，不回改，以 spec 和代码为准。

1. **第 3、4 节的修正让提交的代码与 plan 的代码块不同**。Task 1–4、6–13 和 10b、修复轮都有。主要的有：
   - 令牌管理器：F1，退出和结束会话的规则，`handleStorageChange()`，`endSession` 的布尔值，跟随记录时的访问令牌和退避；
   - 中间件：F2，一个信号，绑定会话；
   - stores：每个会话一个 `RootStore`，`UserService` 从构造函数接收客户端，完成新手引导的单个 `PATCH`，尽力而为的写入；
   - 包装：`autoRetry`；
   - CSP：页面以外不带，按原文取哈希；
   - e2e：`expectQuietConsole`，S2 的等待，`watchPage` 按状态码判断，抽出的 fixture。

   plan 中 Task 2、3 的代码块因此与提交的文件不同。
2. **plan 没有的新文件**：
   - `token-manager.session-change.test.ts`（F1）；
   - `auth-middleware.session.test.ts`、`store-context.test.ts`、`root.store.test.ts`（Task 10b，修复轮改写后两个）；
   - `e2e/fixtures/onboarding-pages.ts`（F9）。

   spec 2.1 按 `f27434c` 到合并前的代码列出：plan 的表加上控制者的补充。
3. **测试的数目**：
   - web 单元测试 129 个。plan 做完是 108 个，Task 10b 加 8 个，修复轮加 13 个。
   - 认证的核心 99 个：锁 9；令牌管理器 32、会话变化 14、多标签页 20；中间件 20、绑定会话 4。spec 第 4 节原写 72 个，已更正。
   - e2e 35 个：A10 两个页面测试，A4 有、没有 `navigator.locks` 两遍，A6 两种切换。
4. **生成物**：`schema.gen.ts`（911 行）与 plan 的 SHA-256 相同，两次 `pnpm-lock.yaml` 的改动与 plan 逐字相同；每个改到它们的 Task 之后 `make gen-check` 干净。
5. **生产代码中 plan 之外的改动**（都在第 3、4 节）：
   - propel 的 `setPromiseToast`（Task 8）；
   - 用户资源上传路径的删除（Task 7）；
   - `USE_CASES` 的删除（Task 5）；
   - Task 10b；
   - 修复轮。
6. **父文档同步**：
   - **执行前的 M2 设计同步**：`e3cfa99`。
   - **Task 14**（`2862b63`）：
     - 总体设计 4.3；
     - 前端改动清单 3.1、3.2；
     - README（前端一节的"登录状态""M2 中看到的页面"，端到端一节的"页面的登录状态"和 S2，部署一节的"HTTPS""CSP"）；
     - 五份交接的处理结果。
   - **修复轮**（`fa4579a`、`e9ad0f1`）：M2 设计 3.20、7.1、7.9、12 节 P5、13.1、13.2、§16，前端改动清单 3.2。
   - **spec 在执行中逐次更正**：`d927fbf`、`9ed06b2`、`554b364`、`3bd4981`、`5db5f31`、`0727a87`、`fa4579a`、`e9ad0f1`。
   - **状态**：spec 的状态和 M2 设计的进度表在本评审的提交中改为已完成。

## 6. 移交事项

| 交接 | 状态 | P4 处理的条目 | 留下的条目 |
|---|---|---|---|
| [M0-P5-frontend-api-notes](../handoffs/M0-P5-frontend-api-notes.md) | done | 前端改调 `/api/v0/instance`，认证接口在 `/api/v0/` 下，同源（Task 6、7、10）；CSP（Task 11）；"安全响应头与 CSP 放在同一层"按下面的理由关闭 | — |
| [M0-P6-e2e-notes](../handoffs/M0-P6-e2e-notes.md) | open | 页面的登录状态 `signedInPage`、S2 的接口请求和控制台和 CSP `watchPage`（Task 12、13）；新加的等待都有期限 | fixture 写法的延伸（M4、M5、M8） |
| [M1-P2-trim-content](../handoffs/M1-P2-trim-content.md) | done | 主题：`IUserTheme` 删除，用生成的 `Theme`（Task 8；spec 第 3 节第 11 条） | — |
| [M1-P3-trim-platform](../handoffs/M1-P3-trim-platform.md) | open | Cookie 会话和 CSRF；认证错误就地显示；实例字段、`is_self_managed` 和两步；新手引导挂载时的旧接口 | `@nerve/services` 中 API 令牌的旧地址（P5） |
| [M1-P4-router-native](../handoffs/M1-P4-router-native.md) | done | `next_path`（前端补上控制字符，带查询和片段）；`AuthenticationWrapper` 重写；401 不照搬恒为真的判断；`security.tsx` 的错误断言；表单不再提交到 `/auth/…` | — |
| M1-closeout | open（P5 关闭时写处理结果） | 改到的文件中的死成员随改随删（`UserStore` 的 `reset`、`isAuthenticated`、`error`、`isLoading` 等） | 其余死成员；oxlint 在改到的文件中清零、`no-unneeded-ternary`；主题下拉框（P5） |

**M0-P5 "安全响应头与 CSP 放在同一层"的关闭理由**（M2 设计 8.3，spec 第 4 节要求写进本评审）：交接的本意是两者都由服务端在 M2 加入。M2 有意把它们分在两层：
- `X-Content-Type-Options`、`Referrer-Policy`、`X-Frame-Options` 对接口的响应同样有意义，放在 `httpserver` 的固定链上，所有响应都带（P2）。
- CSP 只对页面有意义，而且要用 `index.html` 中内联脚本的哈希，只有 `webui` 知道这些脚本，所以由 `webui` 只加在 HTML 响应上（P4，Task 11）。

`headers_test.go` 的 `TestOnlyPagesHaveAContentSecurityPolicy` 经整个服务核对：页面带策略，资源文件、`/healthz`、`/api/v0/instance` 和 `/api/` 下的 404 都不带任何 CSP。浏览器核对 C9 在 8 个页面上没有违规，S2 在"所有哈希都错"的构建上失败（Task 12）。

Codex 对 M2 设计的评审交给 P4 的事项，都已处理：
- **I-5**（另一个标签页换了账户）和 **R4**（`login_id` 的写入竞争）：spec 2.5 的会话规则；令牌管理器、多标签页、会话变化的测试（两种锁）；A6 的两种切换；浏览器核对 C4a–C4c；Task 10b 和 FW1 让一代 stores 只以自己的会话发请求。
- **I-11**（新手引导挂载时请求 M3 的旧接口）：Task 5 删除预取；A10 的两个页面测试（整页加载、注册后应用内进入）；C8。
- **M-5**（续期非 401 的失败不退出）：只有续期得到 401、或者重发后仍是 401 才结束会话；令牌管理器和中间件的测试；C6b、C6c。
- 7.1 的"先验证 openapi-fetch 0.17.0 的中间件能重发"：spec 附录 A 的 E1。
- §16 的"重复使用检测是否频繁"：C3、C5 共 15 次页面续期全部 200，会话没有被撤销；A4 两遍各 10 次同样。
- P2 评审第 7 节"真实反向代理后面的客户端 IP"：C11（Caddy），在 P4 关闭，不留给 M8。

交给后续的事项：

| 交给 | 事项 |
|---|---|
| M2/P5 | spec 第 5 节的全部：<br>- 个人设置的 preferences（时区）、security 的 PAT 列表、api-tokens，PAT store；<br>- `@nerve/services` 删到只剩地址工具和文件工具，剩下的 Plane 类型；<br>- 主题下拉框的定位；<br>- 死成员和死 prop 的其余部分；<br>- oxlint 在改到的文件中清零（P4 改到的文件还有警告，spec 第 6 节）和 `no-unneeded-ternary`；<br>- 关键词规则 `withCredentials:\s*true`、`/api/users/api-tokens`、`/api/timezones/`；<br>- A7–A9、A11、A12 的页面版本，9.6 中 P5 的浏览器核对；<br>- 维护页的文案（M2 设计 12 节 P5 的交付物 1） |
| M3 及以后有 stores 的 M | 每个会话一个 `RootStore`：services 由这一代的 stores 用这一代的客户端建，经构造函数传入；store 只经自己的 `RootStore` 找兄弟 store；填充 stores 的 SWR 键带上 `loginId`；这些 stores 接上新接口时，一代退役要释放它在沿用的 `router` 上注册的反应（M2 设计 13.2，Task 10b、FW1） |
| M3 | 未处理的 `fetchWorkspaces()` 拒绝：`web/apps/web/app/(all)/invitations/page.tsx:88` 加入工作区之后的 `fetchWorkspaces().then(…)` 没有返回它的 Promise，也没有处理拒绝，与 13.2 中新手引导 fetcher 的一条同类，M3 对接邀请时一并改（M2 设计 13.2，Task 8） |
| M4 | 未登录的深层链接在包装跳转之前加载编辑器的代码包（tiptap 的 Emoji、`is-emoji-supported`），做代码拆分；e2e 的 `emojiCanvasWarning` 随之去掉（M2 设计 13.2，Task 12） |

P3b 评审第 6 节交给后续的事项不变：M4 的命令行"只投递"River 客户端，M8 的 `stop_grace_period`，负责人的 argon2 调参与登录耗时。

## 7. 已知限制

- **界面语言是全局的，旧的一代仍可能改它**（spec 第 3 节第 17 条）：切换之前发出的取资料请求在切换之后才回来时，旧一代的 `ProfileStore` 按旧账户的资料设语言，直到新会话的资料再设一次。只影响界面语言，不是以错误的身份写入。
- **旧的一代留在内存里**（spec 第 3 节第 17 条，M2 设计 13.2）：Plane 的 `cycle_filter`、`module_filter`、`project_filter` 在沿用的 `router` 上注册的 `reaction`，`IssueRootStore` 的 `autorun`，都不随旧的一代释放。路由变化时它们对旧的 stores 做本地的同步更新，不发请求。代价是每换一次会话留下一代 stores；原地重建时同样如此。释放交给接上这些 stores 的 M。
- **用户的操作被会话变化截断时，错误提示可能出现在新账户的页面上**：例如 C4b 中 X 的资料保存被放弃，页面已显示 Y，同时弹出"User details update failed"。按调用方的规则（第 3 节）这是有意的：操作确实没有做。
- **租约不是原子的；持有租约的标签页被冻结超过 10 秒**（spec 第 6 节，M2 设计 §16）：可能让两个标签页用同一代刷新令牌续期，触发重复使用检测，用户要重新登录。原型和分支上 A4 的租约一遍、C5 的续期全部 200，代数不重复。
- **`navigator.locks` 之下，localStorage 与锁的授予之间没有顺序**（整分支评审，M2 设计 §16）：Chromium 在渲染进程之间异步同步 localStorage，拿到锁的标签页理论上可能读到已被替换的记录，后果与上一条相同。A4 两遍各 10 次、C3、C5 的代数都干净；只能观察。
- **续期进行中关闭或刷新标签页**：轮换后的令牌丢失，下一次续期触发重复使用检测（设计 3.5 接受的代价，与响应丢失相同）。
- **生产构建下停掉 nerve，页面本身加载不了**，浏览器显示自己的错误页（spec 第 3 节第 14 条）；nerve 恢复后刷新即可。"会话暂不可用"只在 nerve 回答、续期失败时出现（C6b、C6c）。维护页的文案在 nerve 只是暂时不可达时有误导，P5 改写。
- **偏好页、api-tokens 页挂载时仍请求 Plane 的旧接口**（`/api/timezones/`、`/api/users/api-tokens/`，404，C9 记下）：P5 改写这两页；它们不在 S2、A10 的页面上。
- **未登录的深层链接加载编辑器的代码包**：`emojiCanvasWarning` 是按原文写在 e2e 中的第三方警告；依赖升级改了它时，更新 fixture。代码拆分在 M4。
- **组件和页面没有 DOM 的单元测试**（第 3 节）：它们由 e2e 和浏览器核对覆盖；`StoreProvider` 的订阅由 e2e 的变异证明被看到（第 4 节）。
- **`deactivateAccount`、`signOut` 读标签页此刻的会话**，不读这一代的会话（Task 10b 的裁定）：两者只在一次渲染的窗口里不同，而请求本身受绑定会话的客户端约束。
- **P4 改到的文件还有 oxlint 警告**：M2 设计 7.8 要求 M2 结束时清零，P5 清（spec 第 6 节）。

## 附录：9.6 的浏览器核对

按 M2 设计 9.6（沿用 M1 设计 7.5 的做法）：临时脚本不进仓库，脚本全文、假数据、运行命令和断言写在这里。浏览器核对是单元测试和端到端测试之外的证据，不代替它们。

- **对象**：合并前的代码 `e9ad0f1`（修复轮之后）。修复轮之前在 `2862b63` 上也跑过一次，15 项全部通过。架构子任务在原型上的运行记在 spec 附录 A。
- **做法**：
  - 把工作区（不含 `.git`）复制到临时目录，在那里 `make build`，脚本对那里的 `bin/nerve` 运行（`lib.mjs` 的 `P`）。
  - 每个核对在开发库 `nerve-dev-db-1` 上建一个独立的库，结束时删除。
  - C6b 另起一个 Postgres 容器，C11 另起一个 Caddy 2.10 容器，结束时都删除；C6a 另起 Vite 开发服务器（端口 3900）和监听 8080 的 nerve。
  - 运行结束后没有留下进程和容器。
- **命令**：`node checks.mjs 10.10.20.104`（本机的局域网地址，C5 用它在非安全上下文中打开页面）。
- **环境**：Node 24.15.0，Playwright 1.63.0 的 Chromium（取自 `e2e` 包的依赖），Docker 29.7.2，Go 1.27.1，macOS。
- **假数据**：
  - 每个核对用 `email(标签)` 生成带随机后缀的邮箱，密码都是 `Tr0ub4dor&3`，账户经注册接口或注册页创建。
  - C2 的 10 个 `next_path` 取值、C4b 扣住的续期、C6c 的 503 和 `Retry-After: 2`，都写在脚本中。
- **断言**：每个核对的通过条件是 `report(id, 条件, 详情)` 的第二个参数；下面"运行输出"中每行的 JSON 是详情。

### 结果（`e9ad0f1`）

| 核对 | 9.6 的条目 | 结果 |
|---|---|---|
| C1 | 登录页、注册页的就地错误 | 通过。已存在的邮箱（大写）409，表单上方"An account with this email already exists."，邮箱留在输入框；常见密码 422，字段下方提示，表单上方没有；弱密码显示规则和横幅，不发请求；错误密码、不存在的邮箱都是 401、同一句"The email or the password is wrong."；没有 CSP 违规 |
| C2 | `next_path` 的各种取值 | 通过。10 个取值都落到预期的地方：站内路径（带查询和片段）原样回来，前后的空格去掉；`//evil.example`、`/\evil.example`、`javascript:`、带制表符和 NUL 的、绝对地址都落到 `/create-workspace`；未完成新手引导的账户落到 `/onboarding`；都在同源 |
| C3 | 两个标签页的续期和退出 | 通过。8 次续期全部 200，代数 0–7 各一次，一次一个；退出后两个标签页都到 `/?next_path=%2Fonboarding`；会话的撤销原因是 `logout` |
| C4a | 切换 ①，有 `navigator.locks` | 通过。甲显示 Y；此后保存的名字写进 Y（"Yvonne"），X 的名字不变，X 的会话没有被撤销 |
| C4b | 切换 ①，没有 `navigator.locks`，正好在续期时 | 通过。nerve 把 X 从第 2 代轮换到第 3 代，第 3 代从没有写进记录；记录是 Y 的；资料步骤没有前进，两个账户的名字都没变；没有失败的接口请求。X 被放弃的保存在 Y 的页面上弹出"User details update failed"（第 7 节） |
| C4c | 切换 ②（先退出再登录） | 通过。甲随之到 `?next_path=%2Fonboarding`；乙登录 200；两个标签页都显示 Y |
| C5 | 局域网 HTTP | 通过。`isSecureContext` 为假，没有 `locks`，`getRandomValues` 可用，`randomUUID` 没有；注册 201；两个标签页都成功，7 次续期全部 200、代数 0–6 各一次，会话的代数 7，没有被撤销；启动日志有非回环地址的 WARN |
| C6a | 停掉 nerve、刷新（9.6 的字面，Vite 开发服务器提供页面） | 通过。显示维护页，地址和记录都在；nerve 启动后 4.96 秒自动回到原页面 |
| C6b | 会话暂不可用：停掉 nerve 的数据库 | 通过。"Cannot reach the server for now"和"Try again"；续期 500；地址和记录都在，没有跳到登录页；数据库启动后 1.35 秒自动恢复，回到原页面 |
| C6c | 会话暂不可用：续期 503 和 `Retry-After` | 通过。显示"会话暂不可用"；放开后 2.30 秒自己恢复，地址不变 |
| C7 | 实例请求失败时的维护页 | 通过 |
| C8 | 第一次打开 `/onboarding`、登录后的落点 | 通过。两段都没有失败的接口请求、旧接口请求、未处理的异常、控制台错误、CSP 违规；新手引导只请求实例、注册、`/me`、`/me/profile` 和两个 `PATCH`；完成了引导的账户登录后落到 `/create-workspace` |
| C9 | 登录页、新手引导页、设置页没有 CSP 违规 | 通过。8 个页面都带策略，没有违规。偏好页请求 `/api/timezones/`、api-tokens 页请求 `/api/users/api-tokens/`，都是 404（P5 改写这两页） |
| C10 | 控制台没有"navigate() 应在 useEffect 中调用"的警告 | 通过。没有这类警告；只有 6 条 Chromium 的 Canvas2D `willReadFrequently` 提示 |
| C11 | 真实反向代理后面的客户端 IP | 通过。不信任代理时会话记下代理的地址 `127.0.0.1` 并记 WARN；`server.trusted_proxies` 设为 `127.0.0.1/32` 后记下客户端的地址 `172.17.0.1`；直接访问记下 `127.0.0.1` |

### 脚本全文

`lib.mjs`（`P` 是分支副本的目录，`OUT` 是截图和日志的目录；路径在这里写成 `/tmp/nerve-m2/…`）：

```js
// Helpers of the M2/P4 browser checks (M2 design 9.6): nerve processes, scratch databases, API calls,
// signed-in browser contexts and page watches. Not part of the repository.
import { spawn, execFileSync } from "node:child_process";
import { closeSync, openSync, readFileSync, rmSync } from "node:fs";
import { once } from "node:events";
import { randomBytes } from "node:crypto";
import { createRequire } from "node:module";
import { setTimeout as sleep } from "node:timers/promises";

export const P = "/tmp/nerve-m2/p4branch";
export const OUT = "/tmp/nerve-m2/p4branchchecks/out";
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
  page.on("response", (res) => {
    const path = new URL(res.url()).pathname;
    if (path.startsWith("/api/") && res.status() >= 400) w.apiFailures.push(`${res.status()} ${res.request().method()} ${path}`);
  });
  page.on("requestfailed", (req) => {
    const path = new URL(req.url()).pathname;
    if (path.startsWith("/api/")) w.apiFailures.push(`${req.failure()?.errorText} ${req.method()} ${path}`);
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

export function docker(args) {
  return execFileSync("docker", args, { encoding: "utf8" }).trim();
}
```

`checks.mjs`：

```js
// M2/P4 browser checks (M2 design 9.6), against the bin/nerve that make build makes in a copy of the branch
// at the commit under review (lib.mjs: P). Scratch databases in
// nerve-dev-db-1 are created and dropped; C6b runs a Postgres container of its own and C11 a Caddy
// container of its own, both removed at the end. Usage: node checks.mjs <LAN IP> [check id ...]
import { writeFileSync } from "node:fs";
import {
  OUT,
  PASSWORD,
  P,
  alertText,
  answerTo,
  allWarnings,
  call,
  chromium,
  createDatabase,
  docker,
  dropDatabase,
  email,
  fillSignIn,
  fillSignUp,
  holdFirstRefresh,
  login,
  onboard,
  oneAtATime,
  randomBytes,
  register,
  report,
  results,
  run,
  seed,
  sessionOf,
  sleep,
  sql,
  startNerve,
  tokenParts,
  visible,
  watch,
} from "./lib.mjs";
import { spawn } from "node:child_process";
import { readFileSync } from "node:fs";

const LAN_IP = process.argv[2];
const only = new Set(process.argv.slice(3));
const wanted = (id) => only.size === 0 || only.has(id);
const tag = randomBytes(4).toString("hex");
const databases = [];
async function database(label) {
  const name = `p4check_${tag}_${label}`;
  databases.push(name);
  return createDatabase(name);
}

const browser = await chromium.launch();
const nerves = [];
async function nerve(name, dbUrl, addr = "127.0.0.1:0", extra = {}) {
  const n = await startNerve(`${tag}-${name}`, dbUrl, addr, extra);
  nerves.push(n);
  return n;
}

const shot = (page, name) => page.screenshot({ path: `${OUT}/${tag}-${name}.png`, fullPage: true });
const profileStep = (page) => page.getByText("Create your profile.");
const workspaceStep = (page) => page.getByText("Create your workspace");
const displayName = (address) => address.split("@")[0];

try {
  const mainDb = await database("main");
  const n1 = await nerve("n1", mainDb);
  const n2 = await nerve("n2", mainDb, "127.0.0.1:0", { NERVE_AUTH__ACCESS_TOKEN_TTL: "3s" });

  // C1: the sign-in and sign-up pages show refusals in place.
  if (wanted("C1"))
    await run("C1", async () => {
      const taken = email("c1-taken");
      await register(n1.baseURL, taken);
      const ctx = await browser.newContext({ baseURL: n1.baseURL });
      const page = await ctx.newPage();
      const w = await watch(page, "C1");
      await page.goto("/sign-up");
      const takenStatus = await fillSignUp(page, taken.toUpperCase());
      const takenAlert = await alertText(page);
      const takenKept = await page.getByLabel("Email", { exact: true }).inputValue();
      await shot(page, "c1-email-taken");
      const fresh = email("c1-fresh");
      const commonStatus = await fillSignUp(page, fresh, "Password1!");
      const common = await visible(page.getByText("This password is too common"));
      const commonAlert = await alertText(page);
      await shot(page, "c1-common-password");
      await page.getByLabel("Set a password", { exact: true }).fill("password");
      await page.getByLabel("Confirm password", { exact: true }).fill("password");
      const sent = w.api.length;
      await page.getByRole("button", { name: "Create account" }).click();
      const weakAlert = await alertText(page);
      const rules = await visible(page.getByText("8–128 characters"));
      const weakSent = w.api.slice(sent);
      await shot(page, "c1-weak-password");
      await page.goto("/");
      const wrongStatus = await fillSignIn(page, taken, "Wr0ng-password");
      const wrongAlert = await alertText(page);
      const unknownStatus = await fillSignIn(page, email("c1-nobody"));
      const unknownAlert = await alertText(page);
      await shot(page, "c1-sign-in-refused");
      const seen = {
        takenStatus, takenAlert, takenKept, url: new URL(page.url()).pathname, commonStatus, common, commonAlert,
        weakAlert, rules, weakSent, wrongStatus, wrongAlert, unknownStatus, unknownAlert, csp: w.csp,
      };
      report(
        "C1",
        takenStatus === 409 && takenAlert === "An account with this email already exists." &&
          takenKept === taken.toUpperCase() && commonStatus === 422 && common && commonAlert === null &&
          weakAlert === "Try setting-up a strong password to proceed" && rules && weakSent.length === 0 &&
          wrongStatus === 401 && unknownStatus === 401 && wrongAlert === "The email or the password is wrong." &&
          unknownAlert === wrongAlert && w.csp.length === 0,
        seen
      );
      await ctx.close();
    });

  // C2: next_path, every kind of value.
  if (wanted("C2"))
    await run("C2", async () => {
      const onboarded = email("c2-onboarded");
      await onboard(n1.baseURL, await register(n1.baseURL, onboarded));
      const newcomer = email("c2-new");
      await register(n1.baseURL, newcomer);
      const cases = [
        ["/settings/profile/general?tab=x#y", onboarded, "/settings/profile/general?tab=x#y"],
        [" /settings/profile/preferences", onboarded, "/settings/profile/preferences"],
        ["/%2F%2Fevil.example", onboarded, "/%2F%2Fevil.example"],
        ["//evil.example", onboarded, "/create-workspace"],
        ["/\\evil.example", onboarded, "/create-workspace"],
        ["javascript:alert(1)", onboarded, "/create-workspace"],
        ["/\t/evil.example", onboarded, "/create-workspace"],
        ["/\u0000/x", onboarded, "/create-workspace"],
        ["https://evil.example/x", onboarded, "/create-workspace"],
        ["/settings/profile/general", newcomer, "/onboarding"],
      ];
      const seen = [];
      for (const [nextPath, account, want] of cases) {
        const ctx = await browser.newContext({ baseURL: n1.baseURL });
        const page = await ctx.newPage();
        await page.goto(`/?next_path=${encodeURIComponent(nextPath)}`);
        const status = await fillSignIn(page, account);
        await page.waitForURL((url) => url.pathname !== "/", { timeout: 10_000 }).catch(() => undefined);
        await page.waitForLoadState("networkidle");
        const url = new URL(page.url());
        seen.push({ nextPath, status, landed: url.pathname + url.search + url.hash, want, origin: url.origin === n1.baseURL });
        await ctx.close();
      }
      // Signed out, a page behind the sign-in goes to the sign-in page with the whole address as next_path.
      const ctx = await browser.newContext({ baseURL: n1.baseURL });
      const page = await ctx.newPage();
      await page.goto("/settings/profile/general?tab=x#y");
      await page.waitForURL((url) => url.pathname === "/", { timeout: 10_000 });
      const redirected = new URL(page.url());
      await ctx.close();
      report(
        "C2",
        seen.every((s) => s.status === 200 && s.landed === s.want && s.origin) &&
          redirected.search === "?next_path=%2Fsettings%2Fprofile%2Fgeneral%3Ftab%3Dx%23y",
        { cases: seen, redirected: redirected.pathname + redirected.search }
      );
    });

  // C3: two tabs refresh one at a time, then sign out together.
  if (wanted("C3"))
    await run("C3", async () => {
      const address = email("c3");
      const tokens = await register(n2.baseURL, address);
      const ctx = await browser.newContext({ baseURL: n2.baseURL });
      await seed(ctx, n2.baseURL, tokens);
      const a = await ctx.newPage();
      const wa = await watch(a, "C3-a");
      await a.goto("/onboarding");
      await profileStep(a).waitFor();
      const b = await ctx.newPage();
      const wb = await watch(b, "C3-b");
      await b.goto("/onboarding");
      await profileStep(b).waitFor();
      await a.getByLabel("Name").fill("Ada");
      await b.getByLabel("Name").fill("Ada");
      await sleep(3_500); // the 3 s access tokens expire
      await holdFirstRefresh(ctx, 1_000);
      const [saved] = await Promise.all([
        Promise.all([a, b].map((tab) => answerTo(tab, "PATCH", "/api/v0/me/profile"))),
        Promise.all([a, b].map((tab) => tab.getByRole("button", { name: "Continue" }).click())),
      ]);
      const both = await Promise.all([a, b].map((tab) => visible(workspaceStep(tab))));
      await Promise.all([...wa.pending, ...wb.pending]);
      const refreshes = [...wa.refreshes, ...wb.refreshes];
      const record = JSON.parse(await a.evaluate(() => localStorage.getItem("nerve.auth")));
      const session = await sessionOf(mainDbUrl(), record.refresh_token);
      // Sign out in tab A: both tabs go to the sign-in page. The menu shows the name just saved.
      await a.getByRole("button", { name: "Ada" }).click();
      await a.getByRole("menuitem", { name: "Wrong e-mail address?" }).click();
      await a.getByRole("button", { name: "Switch account" }).click();
      await Promise.all([a, b].map((tab) => tab.waitForURL((u) => u.pathname === "/", { timeout: 10_000 })));
      const urls = [a, b].map((tab) => new URL(tab.url()).pathname + new URL(tab.url()).search);
      const after = await sessionOf(mainDbUrl(), record.refresh_token);
      const left = await b.evaluate(() => localStorage.getItem("nerve.auth"));
      report(
        "C3",
        both.every(Boolean) && saved.every((r) => r.status() === 200) && refreshes.length >= 4 && refreshes.every((r) => r.status === 200) &&
          oneAtATime(refreshes) && session.generation === refreshes.length && session.revoked_at === null &&
          urls.every((u) => u === "/?next_path=%2Fonboarding") && after.revoke_reason === "logout" && left === null,
        { both, refreshes: refreshes.length, statuses: refreshes.map((r) => r.status), oneAtATime: oneAtATime(refreshes), intervals: refreshes.map((r) => [Math.round(r.start % 100000), Math.round(r.end % 100000), r.sentGeneration, r.finishedAt % 100000]), generation: session.generation, urls, revoke: after.revoke_reason, left }
      );
      await ctx.close();
    });

  // C4a: another tab signs Y in over X without signing out (web locks): tab A goes on as Y.
  if (wanted("C4a"))
    await run("C4a", async () => {
      const x = email("c4a-x");
      const y = email("c4a-y");
      const xTokens = await register(n1.baseURL, x);
      await register(n1.baseURL, y);
      const ctx = await browser.newContext({ baseURL: n1.baseURL });
      await seed(ctx, n1.baseURL, xTokens);
      const a = await ctx.newPage();
      await a.goto("/onboarding");
      const showedX = await visible(a.getByRole("button", { name: displayName(x) }));
      const xHeld = JSON.parse(await a.evaluate(() => localStorage.getItem("nerve.auth"))).refresh_token;
      const b = await ctx.newPage();
      await b.goto("/site.webmanifest.json");
      const yRecord = { refresh_token: (await login(n1.baseURL, y)).refresh_token, login_id: randomBytes(16).toString("hex") };
      await b.evaluate(
        (text) => navigator.locks.request("nerve.auth.refresh", () => localStorage.setItem("nerve.auth", text)),
        JSON.stringify(yRecord)
      );
      const showedY = await visible(a.getByRole("button", { name: displayName(y) }));
      await a.getByLabel("Name").fill("Yvonne");
      await a.getByRole("button", { name: "Continue" }).click();
      await visible(workspaceStep(a));
      await sleep(500);
      const names = await sql(mainDbUrl(), "SELECT email, first_name FROM users WHERE email = ANY($1)", [[x, y]]);
      const xSession = await sessionOf(mainDbUrl(), xHeld);
      report(
        "C4a",
        showedX && showedY && names.find((r) => r.email === y)?.first_name === "Yvonne" &&
          names.find((r) => r.email === x)?.first_name === "" && xSession.revoked_at === null,
        { showedX, showedY, names, xRevoked: xSession.revoked_at }
      );
      await ctx.close();
    });

  // C4b: the switch lands while tab A's refresh is out (the lease, no navigator.locks): A does not write
  // X's new token back, and follows Y.
  if (wanted("C4b"))
    await run("C4b", async () => {
      const x = email("c4b-x");
      const y = email("c4b-y");
      const xTokens = await register(n2.baseURL, x);
      await register(n2.baseURL, y);
      const ctx = await browser.newContext({ baseURL: n2.baseURL });
      await ctx.addInitScript(() => {
        Reflect.deleteProperty(Navigator.prototype, "locks");
        // Every write of the record by this tab, in order.
        const setItem = Storage.prototype.setItem;
        Storage.prototype.setItem = function (key, value) {
          if (key === "nerve.auth") (window.__writes ??= []).push(value);
          if (key.startsWith("nerve.auth")) (window.__all ??= []).push(`${performance.now().toFixed(0)} set ${key} ${value.slice(0, 60)}`);
          return setItem.call(this, key, value);
        };
      });
      await seed(ctx, n2.baseURL, xTokens);
      const a = await ctx.newPage();
      const w = await watch(a, "C4b");
      await a.goto("/onboarding");
      await profileStep(a).waitFor();
      await a.getByLabel("Name").fill("Xavier");
      let release;
      let held;
      const out = new Promise((resolve) => (held = resolve));
      await ctx.route("**/api/v0/auth/refresh", async (route) => {
        if (release === undefined) {
          const go = new Promise((resolve) => (release = resolve));
          held();
          await Promise.race([go, sleep(15_000)]);
        }
        await route.continue();
      });
      await a.getByRole("button", { name: "Continue" }).click();
      await Promise.race([out, sleep(10_000).then(() => Promise.reject(new Error("no refresh went out")))]);
      const xBefore = await sessionOf(mainDbUrl(), xTokens.refresh_token);
      // Tab B signs Y in meanwhile, in the lease's race window: a new record written at once.
      const yTokens = await login(n2.baseURL, y);
      const yLogin = randomBytes(16).toString("hex");
      const b = await ctx.newPage();
      await b.goto("/site.webmanifest.json");
      await b.evaluate(
        (text) => localStorage.setItem("nerve.auth", text),
        JSON.stringify({ refresh_token: yTokens.refresh_token, login_id: yLogin })
      );
      release();
      // Y shows once Y's refresh has run, after X's held refresh let the lease go: X's has ended by then.
      const showedY = await visible(a.getByRole("button", { name: displayName(y) }));
      const xAfter = await sessionOf(mainDbUrl(), xTokens.refresh_token);
      const writes = (await a.evaluate(() => window.__writes ?? [])).map((text) => {
        const r = JSON.parse(text);
        return { ...tokenParts(r.refresh_token), login: r.login_id };
      });
      const record = JSON.parse(await a.evaluate(() => localStorage.getItem("nerve.auth")));
      const ySession = tokenParts(yTokens.refresh_token).sessionId;
      const xNewGenerationWritten = writes.some((wr) => wr.sessionId === xAfter.id && wr.generation === xAfter.generation);
      const names = await sql(mainDbUrl(), "SELECT email, first_name FROM users WHERE email = ANY($1)", [[x, y]]);
      const stayed = await visible(profileStep(a), 2_000);
      await shot(a, "c4b-after");
      console.log(JSON.stringify({ api: w.api, all: await a.evaluate(() => window.__all), text: (await a.evaluate(() => document.body.innerText)).slice(0, 300), errors: w.pageErrors, console: w.consoleErrors }));
      report(
        "C4b",
        showedY && xAfter.generation === xBefore.generation + 1 && xAfter.revoked_at === null &&
          !xNewGenerationWritten && record.login_id === yLogin && tokenParts(record.refresh_token).sessionId === ySession &&
          names.find((r) => r.email === x)?.first_name === "" && names.find((r) => r.email === y)?.first_name === "" && stayed,
        {
          showedY, xGenerationBefore: xBefore.generation, xGenerationAfter: xAfter.generation, xNewGenerationWritten,
          writes, recordIsY: record.login_id === yLogin, names, profileStepStayed: stayed, apiFailures: w.apiFailures,
        }
      );
      await ctx.close();
    });

  // C4c: another tab signs out, then signs Y in: tab A comes in as Y.
  if (wanted("C4c"))
    await run("C4c", async () => {
      const x = email("c4c-x");
      const y = email("c4c-y");
      const xTokens = await register(n1.baseURL, x);
      await register(n1.baseURL, y);
      const ctx = await browser.newContext({ baseURL: n1.baseURL });
      await seed(ctx, n1.baseURL, xTokens);
      const a = await ctx.newPage();
      await a.goto("/onboarding");
      await profileStep(a).waitFor();
      const b = await ctx.newPage();
      await b.goto("/onboarding");
      await b.getByRole("button", { name: displayName(x) }).click();
      await b.getByRole("menuitem", { name: "Wrong e-mail address?" }).click();
      await b.getByRole("button", { name: "Switch account" }).click();
      await a.waitForURL((u) => u.pathname === "/", { timeout: 10_000 });
      const aOut = new URL(a.url()).search;
      await b.waitForURL((u) => u.pathname === "/", { timeout: 10_000 });
      const status = await fillSignIn(b, y);
      await Promise.all([a, b].map((tab) => tab.waitForURL((u) => u.pathname === "/onboarding", { timeout: 10_000 })));
      const showed = await Promise.all([a, b].map((tab) => visible(tab.getByRole("button", { name: displayName(y) }))));
      report("C4c", aOut === "?next_path=%2Fonboarding" && status === 200 && showed.every(Boolean), { aOut, status, showed });
      await ctx.close();
    });

  // C5: plain HTTP at the LAN address: no secure context, no navigator.locks; the lease keeps the tabs apart.
  if (wanted("C5") && LAN_IP)
    await run("C5", async () => {
      const lanDb = await database("lan");
      const n3 = await nerve("n3-lan", lanDb, "0.0.0.0:0", { NERVE_AUTH__ACCESS_TOKEN_TTL: "3s" });
      const base = `http://${LAN_IP}:${n3.port}`;
      const ctx = await browser.newContext({ baseURL: base });
      const a = await ctx.newPage();
      const wa = await watch(a, "C5-a");
      await a.goto("/sign-up");
      const context = await a.evaluate(() => ({
        isSecureContext: window.isSecureContext,
        locks: "locks" in navigator,
        getRandomValues: typeof crypto.getRandomValues,
        randomUUID: typeof crypto.randomUUID,
      }));
      const address = email("c5");
      const signUp = await fillSignUp(a, address);
      await a.waitForURL((u) => u.pathname === "/onboarding", { timeout: 10_000 });
      await profileStep(a).waitFor();
      const loginId = JSON.parse(await a.evaluate(() => localStorage.getItem("nerve.auth"))).login_id;
      const b = await ctx.newPage();
      const wb = await watch(b, "C5-b");
      await b.goto("/onboarding");
      await profileStep(b).waitFor();
      await a.getByLabel("Name").fill("Lan");
      await b.getByLabel("Name").fill("Lan");
      await sleep(3_500);
      await holdFirstRefresh(ctx, 1_000);
      const [saved] = await Promise.all([
        Promise.all([a, b].map((tab) => answerTo(tab, "PATCH", "/api/v0/me/profile"))),
        Promise.all([a, b].map((tab) => tab.getByRole("button", { name: "Continue" }).click())),
      ]);
      const both = await Promise.all([a, b].map((tab) => visible(workspaceStep(tab))));
      await Promise.all([...wa.pending, ...wb.pending]);
      const refreshes = [...wa.refreshes, ...wb.refreshes];
      const record = JSON.parse(await a.evaluate(() => localStorage.getItem("nerve.auth")));
      const session = await sessionOf(lanDb, record.refresh_token);
      const log = readFileSync(n3.logFile, "utf8");
      const warned = log.includes("not running as prod but listening beyond this machine");
      await shot(a, "c5-lan");
      report(
        "C5",
        context.isSecureContext === false && context.locks === false && context.getRandomValues === "function" &&
          signUp === 201 && /^[0-9a-f]{32}$/.test(loginId) && both.every(Boolean) && saved.every((r) => r.status() === 200) && refreshes.length >= 4 &&
          refreshes.every((r) => r.status === 200) && oneAtATime(refreshes) && session.revoked_at === null &&
          session.generation === refreshes.length && warned &&
          [...wa.apiFailures, ...wb.apiFailures].length === 0,
        { base, context, signUp, loginId, both, refreshes: refreshes.length, sent: refreshes.map((r) => [r.sentGeneration, r.status, r.finishedAt]), generation: session.generation, revoked: session.revoked_at, warned, api: [...wa.api, ...wb.api] }
      );
      await ctx.close();
    });

  // C6a: the frontend from the Vite dev server, nerve stopped and started again (the literal steps).
  if (wanted("C6a"))
    await run("C6a", async () => {
      const devDb = await database("dev");
      let n4 = await nerve("n4-8080", devDb, "127.0.0.1:8080");
      const vite = spawn("pnpm", ["exec", "react-router", "dev", "--port", "3900", "--strictPort"], {
        cwd: `${P}/web/apps/web`,
        stdio: "ignore",
        detached: true,
      });
      try {
        const base = "http://127.0.0.1:3900";
        const deadline = Date.now() + 120_000;
        while (!(await fetch(`${base}/`).then((r) => r.ok, () => false))) {
          if (Date.now() > deadline) throw new Error("the dev server did not start");
          await sleep(500);
        }
        const address = email("c6a");
        const tokens = await register(n4.baseURL, address);
        await onboard(n4.baseURL, tokens);
        const ctx = await browser.newContext({ baseURL: base });
        await seed(ctx, base, tokens);
        const page = await ctx.newPage();
        await page.goto("/settings/profile/general", { timeout: 120_000 });
        await page.waitForLoadState("networkidle", { timeout: 120_000 });
        const before = new URL(page.url()).pathname;
        await n4.stop();
        await page.reload();
        const maintenance = page.getByText("Looks like Nerve didn't start up correctly!");
        const unavailable = page.getByText("Cannot reach the server for now");
        const shown = await Promise.race([
          maintenance.waitFor({ timeout: 30_000 }).then(() => "maintenance"),
          unavailable.waitFor({ timeout: 30_000 }).then(() => "session unavailable"),
        ]).catch(() => "neither");
        await shot(page, "c6a-nerve-stopped");
        const whileDown = { url: new URL(page.url()).pathname, record: (await page.evaluate(() => localStorage.getItem("nerve.auth"))) !== null };
        const restartAt = Date.now();
        n4 = await nerve("n4-8080-again", devDb, "127.0.0.1:8080");
        const recovered = await page
          .waitForFunction(() => !document.body.innerText.includes("Looks like Nerve") && !document.body.innerText.includes("Cannot reach the server") && document.title.includes("Profile"), undefined, { timeout: 180_000 })
          .then(() => Date.now() - restartAt, () => null);
        await shot(page, "c6a-recovered");
        report(
          "C6a",
          before === "/settings/profile/general" && shown !== "neither" && whileDown.url === before && whileDown.record && recovered !== null,
          { before, shown, whileDown, recoveredAfterMs: recovered, after: new URL(page.url()).pathname }
        );
        await ctx.close();
      } finally {
        process.kill(-vite.pid, "SIGTERM");
      }
    });

  // C6b: nerve answers, its database does not: the first refresh fails, the page waits and tries again.
  if (wanted("C6b"))
    await run("C6b", async () => {
      docker(["run", "-d", "--name", "nerve-p4-check-pg", "-e", "POSTGRES_USER=nerve", "-e", "POSTGRES_PASSWORD=nerve", "-e", "POSTGRES_DB=nerve", "-p", "127.0.0.1:55498:5432", "postgres:18.6"]);
      const pgUrl = "postgres://nerve:nerve@127.0.0.1:55498/nerve?sslmode=disable";
      const deadline = Date.now() + 60_000;
      while (!(await sql(pgUrl, "SELECT 1").then(() => true, () => false))) {
        if (Date.now() > deadline) throw new Error("the check's Postgres did not start");
        await sleep(500);
      }
      const n6 = await nerve("n6-own-pg", pgUrl);
      const address = email("c6b");
      const tokens = await register(n6.baseURL, address);
      await onboard(n6.baseURL, tokens);
      const ctx = await browser.newContext({ baseURL: n6.baseURL });
      await seed(ctx, n6.baseURL, tokens);
      const page = await ctx.newPage();
      const w = await watch(page, "C6b");
      await page.goto("/settings/profile/general");
      await page.waitForLoadState("networkidle");
      docker(["stop", "nerve-p4-check-pg"]);
      await page.reload();
      const unavailable = await visible(page.getByText("Cannot reach the server for now"), 30_000);
      const retry = await visible(page.getByRole("button", { name: "Try again" }), 1_000);
      await shot(page, "c6b-database-down");
      const whileDown = { url: new URL(page.url()).pathname, record: (await page.evaluate(() => localStorage.getItem("nerve.auth"))) !== null };
      const refreshFailures = w.apiFailures.filter((f) => f.includes("/api/v0/auth/refresh"));
      const upAt = Date.now();
      docker(["start", "nerve-p4-check-pg"]);
      const recovered = await page
        .getByText("Cannot reach the server for now")
        .waitFor({ state: "detached", timeout: 90_000 })
        .then(() => Date.now() - upAt, () => null);
      await page.waitForLoadState("networkidle");
      await shot(page, "c6b-recovered");
      const after = { url: new URL(page.url()).pathname, title: await page.title() };
      report(
        "C6b",
        unavailable && retry && whileDown.url === "/settings/profile/general" && whileDown.record && recovered !== null &&
          after.url === "/settings/profile/general",
        { unavailable, retry, whileDown, refreshFailures, recoveredAfterMs: recovered, after }
      );
      await ctx.close();
      await n6.stop();
    });

  // C6c: nerve answers the refresh 503 with Retry-After: the page waits, and comes back by itself.
  if (wanted("C6c"))
    await run("C6c", async () => {
      const address = email("c6c");
      const tokens = await register(n1.baseURL, address);
      await onboard(n1.baseURL, tokens);
      const ctx = await browser.newContext({ baseURL: n1.baseURL });
      await seed(ctx, n1.baseURL, tokens);
      const page = await ctx.newPage();
      await page.goto("/settings/profile/general");
      await page.waitForLoadState("networkidle");
      const busy = (route) =>
        route.fulfill({
          status: 503,
          headers: { "Content-Type": "application/problem+json", "Retry-After": "2" },
          body: JSON.stringify({ status: 503, code: "server_busy", title: "Service Unavailable" }),
        });
      await ctx.route("**/api/v0/auth/refresh", busy);
      await page.reload();
      const unavailable = await visible(page.getByText("Cannot reach the server for now"), 10_000);
      await ctx.unroute("**/api/v0/auth/refresh", busy);
      const at = Date.now();
      const recovered = await page
        .getByText("Cannot reach the server for now")
        .waitFor({ state: "detached", timeout: 15_000 })
        .then(() => Date.now() - at, () => null);
      report("C6c", unavailable && recovered !== null && new URL(page.url()).pathname === "/settings/profile/general", {
        unavailable,
        recoveredAfterMs: recovered,
        url: new URL(page.url()).pathname,
      });
      await ctx.close();
    });

  // C7: the instance request fails: the maintenance page.
  if (wanted("C7"))
    await run("C7", async () => {
      const ctx = await browser.newContext({ baseURL: n1.baseURL });
      await ctx.route("**/api/v0/instance", (route) => route.abort("connectionrefused"));
      const page = await ctx.newPage();
      await page.goto("/");
      const shown = await visible(page.getByText("Looks like Nerve didn't start up correctly!"), 30_000);
      await shot(page, "c7-maintenance");
      report("C7", shown, { shown });
      await ctx.close();
    });

  // C8: the first visit of /onboarding, and where a sign-in lands: nothing fails.
  if (wanted("C8"))
    await run("C8", async () => {
      const ctx = await browser.newContext({ baseURL: n1.baseURL });
      const page = await ctx.newPage();
      const w = await watch(page, "C8-onboarding");
      await page.goto("/sign-up");
      const address = email("c8");
      await fillSignUp(page, address);
      await page.waitForURL((u) => u.pathname === "/onboarding", { timeout: 10_000 });
      const profile = await visible(profileStep(page));
      await page.waitForLoadState("networkidle");
      await page.getByLabel("Name").fill("Ada");
      const [saved] = await Promise.all([
        answerTo(page, "PATCH", "/api/v0/me/profile"),
        page.getByRole("button", { name: "Continue" }).click(),
      ]);
      const next = (await visible(workspaceStep(page))) && saved.status() === 200;
      await shot(page, "c8-onboarding-next-step");
      await ctx.close();
      // An onboarded account signs in: it lands on /create-workspace.
      const onboarded = email("c8-onboarded");
      await onboard(n1.baseURL, await register(n1.baseURL, onboarded));
      const ctx2 = await browser.newContext({ baseURL: n1.baseURL });
      const page2 = await ctx2.newPage();
      const w2 = await watch(page2, "C8-landing");
      await page2.goto("/");
      const profileRead = answerTo(page2, "GET", "/api/v0/me/profile");
      await fillSignIn(page2, onboarded);
      await profileRead;
      await page2.waitForURL((u) => u.pathname !== "/", { timeout: 10_000 });
      const landed = new URL(page2.url()).pathname;
      await shot(page2, "c8-landing");
      await ctx2.close();
      const clean = (x) => ({ apiFailures: x.apiFailures, oldApi: x.oldApi, pageErrors: x.pageErrors, consoleErrors: x.consoleErrors, csp: x.csp });
      const isClean = (x) => Object.values(clean(x)).every((list) => list.length === 0);
      report("C8", profile && next && isClean(w) && landed === "/create-workspace" && isClean(w2), {
        profile, next, onboarding: clean(w), onboardingApi: w.api, landed, landing: clean(w2), landingApi: w2.api,
      });
    });

  // C9: no page of M2 violates the CSP, and every page has it.
  if (wanted("C9"))
    await run("C9", async () => {
      const address = email("c9");
      const tokens = await register(n1.baseURL, address);
      const onboardedAddress = email("c9-onboarded");
      const onboardedTokens = await register(n1.baseURL, onboardedAddress);
      await onboard(n1.baseURL, onboardedTokens);
      const pages = [];
      // Each visit signs in anew: a seeded refresh token is used once, and a second use would be a reuse.
      const visit = async (account, path) => {
        const ctx = await browser.newContext({ baseURL: n1.baseURL });
        if (account) await seed(ctx, n1.baseURL, await login(n1.baseURL, account));
        const page = await ctx.newPage();
        const w = await watch(page, `C9 ${path}`);
        const doc = await page.goto(path);
        await page.waitForLoadState("networkidle");
        await sleep(300);
        pages.push({ path, landed: new URL(page.url()).pathname, csp: w.csp, header: (doc?.headers()["content-security-policy"] ?? "").slice(0, 40), apiFailures: w.apiFailures, consoleErrors: w.consoleErrors });
        await ctx.close();
      };
      await visit(undefined, "/");
      await visit(undefined, "/sign-up");
      await visit(address, "/onboarding");
      await visit(onboardedAddress, "/create-workspace");
      for (const tab of ["general", "security", "preferences", "api-tokens"]) await visit(onboardedAddress, `/settings/profile/${tab}`);
      // Each page is the one asked for, not the sign-in page it could have been sent to.
      report("C9", pages.every((p) => p.landed === p.path && p.csp.length === 0 && p.header.startsWith("default-src 'self'; script-src 'self'")), { pages });
    });

  // C11: behind Caddy, a real reverse proxy: the client IP nerve records, with and without trusted_proxies.
  if (wanted("C11"))
    await run("C11", async () => {
      const proxyDb = await database("proxy");
      const port = 18481;
      let n7 = await nerve("n7-untrusted", proxyDb, `0.0.0.0:${port}`);
      docker(["run", "-d", "--name", "nerve-p4-check-caddy", "-p", "127.0.0.1:18480:80", "caddy:2.10-alpine", "caddy", "reverse-proxy", "--from", ":80", "--to", `host.docker.internal:${port}`]);
      try {
        const via = "http://127.0.0.1:18480";
        let deadline = Date.now() + 30_000;
        while (!(await fetch(`${via}/readyz`).then((r) => r.ok, () => false))) {
          if (Date.now() > deadline) throw new Error("Caddy did not start");
          await sleep(300);
        }
        const signUpVia = async (base, label) => {
          const ctx = await browser.newContext({ baseURL: base });
          const page = await ctx.newPage();
          await page.goto("/sign-up");
          const address = email(label);
          const status = await fillSignUp(page, address);
          await page.waitForURL((u) => u.pathname === "/onboarding", { timeout: 10_000 });
          const record = JSON.parse(await page.evaluate(() => localStorage.getItem("nerve.auth")));
          await ctx.close();
          return { status, ip: (await sessionOf(proxyDb, record.refresh_token)).ip };
        };
        const untrusted = await signUpVia(via, "c11-untrusted");
        const warnedUntrusted = readFileSync(n7.logFile, "utf8").split("\n").filter((l) => l.includes("X-Forwarded-For")).slice(0, 2);
        await n7.stop();
        // The proxy's address as nerve sees it: the session's IP without trust.
        const proxyIp = untrusted.ip.replace(/\/\d+$/, "");
        n7 = await nerve("n7-trusted", proxyDb, `0.0.0.0:${port}`, { NERVE_SERVER__TRUSTED_PROXIES: `${proxyIp}/32` });
        deadline = Date.now() + 30_000;
        while (!(await fetch(`${via}/readyz`).then((r) => r.ok, () => false))) {
          if (Date.now() > deadline) throw new Error("nerve behind Caddy did not come back");
          await sleep(300);
        }
        const trusted = await signUpVia(via, "c11-trusted");
        const direct = await signUpVia(n7.baseURL, "c11-direct");
        report("C11", untrusted.status === 201 && trusted.status === 201 && trusted.ip !== untrusted.ip, {
          untrusted, warnedUntrusted, trustedProxies: `${proxyIp}/32`, trusted, direct,
        });
      } finally {
        docker(["rm", "-f", "nerve-p4-check-caddy"]);
        await n7.stop();
      }
    });

  // C10: no page warned that navigate() should be called in useEffect.
  report("C10", !allWarnings.some((w) => /navigate\(\)|useEffect/.test(w)), {
    navigateWarnings: allWarnings.filter((w) => /navigate\(\)|useEffect/.test(w)),
    warnings: allWarnings.length,
    sample: allWarnings.slice(0, 5),
  });

  function mainDbUrl() {
    return mainDb;
  }
} finally {
  await browser.close();
  for (const n of nerves) await n.stop().catch(() => undefined);
  if (wanted("C6b")) {
    try {
      docker(["rm", "-f", "nerve-p4-check-pg"]);
    } catch {}
  }
  for (const name of databases) await dropDatabase(name).catch((e) => console.log(`drop ${name}: ${e}`));
  writeFileSync(`${OUT}/${tag}-results.json`, JSON.stringify(results, null, 2));
  console.log(`results: ${OUT}/${tag}-results.json`);
}
```

### 运行输出（`e9ad0f1`）

C4b 之前的一行是它的诊断输出（页面的请求、localStorage 的写入和页面文字）；刷新令牌只保留前缀。

```text
PASS C1 {"takenStatus":409,"takenAlert":"An account with this email already exists.","takenKept":"C1-TAKEN-MUJROZMQ-1@EXAMPLE.COM","url":"/","commonStatus":422,"common":true,"commonAlert":null,"weakAlert":"Try setting-up a strong password to proceed","rules":true,"weakSent":[],"wrongStatus":401,"wrongAlert":"The email or the password is wrong.","unknownStatus":401,"unknownAlert":"The email or the password is wrong.","csp":[]}
  (C1: 5696 ms)
PASS C2 {"cases":[{"nextPath":"/settings/profile/general?tab=x#y","status":200,"landed":"/settings/profile/general?tab=x#y","want":"/settings/profile/general?tab=x#y","origin":true},{"nextPath":" /settings/profile/preferences","status":200,"landed":"/settings/profile/preferences","want":"/settings/profile/preferences","origin":true},{"nextPath":"/%2F%2Fevil.example","status":200,"landed":"/%2F%2Fevil.example","want":"/%2F%2Fevil.example","origin":true},{"nextPath":"//evil.example","status":200,"landed":"/create-workspace","want":"/create-workspace","origin":true},{"nextPath":"/\\evil.example","status":200,"landed":"/create-workspace","want":"/create-workspace","origin":true},{"nextPath":"javascript:alert(1)","status":200,"landed":"/create-workspace","want":"/create-workspace","origin":true},{"nextPath":"/\t/evil.example","status":200,"landed":"/create-workspace","want":"/create-workspace","origin":true},{"nextPath":"/\u0000/x","status":200,"landed":"/create-workspace","want":"/create-workspace","origin":true},{"nextPath":"https://evil.example/x","status":200,"landed":"/create-workspace","want":"/create-workspace","origin":true},{"nextPath":"/settings/profile/general","status":200,"landed":"/onboarding","want":"/onboarding","origin":true}],"redirected":"/?next_path=%2Fsettings%2Fprofile%2Fgeneral%3Ftab%3Dx%23y"}
  (C2: 8028 ms)
PASS C3 {"both":[true,true],"refreshes":8,"statuses":[200,200,200,200,200,200,200,200],"oneAtATime":true,"intervals":[[66907,66912,0,66912],[66913,66916,1,66916],[71657,72668,4,71665],[71699,71702,6,71702],[67045,67050,2,67050],[67051,67053,3,67054],[71666,71671,5,71670],[71703,71706,7,71706]],"generation":8,"urls":["/?next_path=%2Fonboarding","/?next_path=%2Fonboarding"],"revoke":"logout","left":null}
  (C3: 5048 ms)
PASS C4a {"showedX":true,"showedY":true,"names":[{"email":"c4a-x-mujrpe46-7@example.com","first_name":""},{"email":"c4a-y-mujrpe46-8@example.com","first_name":"Yvonne"}],"xRevoked":null}
  (C4a: 796 ms)
{"api":["GET /api/v0/instance","POST /api/v0/auth/refresh","POST /api/v0/auth/refresh","GET /api/v0/me","GET /api/v0/me/profile","POST /api/v0/auth/refresh","POST /api/v0/auth/refresh","GET /api/v0/me","GET /api/v0/me/profile"],"all":["4 set nerve.auth.e2e-seeded 1","4 set nerve.auth {\"refresh_token\":\"nrv_rt_…","60 set nerve.auth.refresh_lease {\"owner\":\"190a6d1af22268cf0b7422daf5c9847f\",\"expires\":179051","167 set nerve.auth {\"refresh_token\":\"nrv_rt_…","167 set nerve.auth.refresh_lease {\"owner\":\"190a6d1af22268cf0b7422daf5c9847f\",\"expires\":179051","276 set nerve.auth {\"refresh_token\":\"nrv_rt_…","362 set nerve.auth.refresh_lease {\"owner\":\"190a6d1af22268cf0b7422daf5c9847f\",\"expires\":179051","547 set nerve.auth.refresh_lease {\"owner\":\"190a6d1af22268cf0b7422daf5c9847f\",\"expires\":179051","655 set nerve.auth {\"refresh_token\":\"nrv_rt_…"],"text":"Error\n\nUser details update failed. Please try again!\n\nC\nc4b-y-mujrpeqa-10\nCreate your profile.\n\nThis is how you will appear in Nerve.\n\nR\nName\nContinue","errors":[],"console":[]}
PASS C4b {"showedY":true,"xGenerationBefore":2,"xGenerationAfter":3,"xNewGenerationWritten":false,"writes":[{"sessionId":"01a0e2bd-9dd3-7ed9-be5b-cf4a606ed3ec","generation":0,"login":"3b9aae4a788b306bd5bd527d712e83f2"},{"sessionId":"01a0e2bd-9dd3-7ed9-be5b-cf4a606ed3ec","generation":1,"login":"3b9aae4a788b306bd5bd527d712e83f2"},{"sessionId":"01a0e2bd-9dd3-7ed9-be5b-cf4a606ed3ec","generation":2,"login":"3b9aae4a788b306bd5bd527d712e83f2"},{"sessionId":"01a0e2bd-9fe2-78d0-90fb-b9e3c7a39177","generation":1,"login":"8db96c8090006d2fb7707b6301afea44"}],"recordIsY":true,"names":[{"email":"c4b-x-mujrpeqa-9@example.com","first_name":""},{"email":"c4b-y-mujrpeqa-10@example.com","first_name":""}],"profileStepStayed":true,"apiFailures":[]}
  (C4b: 831 ms)
PASS C4c {"aOut":"?next_path=%2Fonboarding","status":200,"showed":[true,true]}
  (C4c: 535 ms)
PASS C5 {"base":"http://10.10.20.104:56896","context":{"isSecureContext":false,"locks":false,"getRandomValues":"function","randomUUID":"undefined"},"signUp":201,"loginId":"3de3406fd4a2803afae9cfaeff3a41ec","both":[true,true],"refreshes":7,"sent":[[0,200,1790510474575],[4,200,1790510479748],[6,200,1790510479972],[1,200,1790510474817],[2,200,1790510474924],[3,200,1790510479631],[5,200,1790510479861]],"generation":7,"revoked":null,"warned":true,"api":["GET /api/v0/instance","POST /api/v0/auth/register","POST /api/v0/auth/refresh","GET /api/v0/me","GET /api/v0/me/profile","POST /api/v0/auth/refresh","PATCH /api/v0/me","POST /api/v0/auth/refresh","PATCH /api/v0/me/profile","GET /api/v0/instance","POST /api/v0/auth/refresh","POST /api/v0/auth/refresh","GET /api/v0/me","GET /api/v0/me/profile","POST /api/v0/auth/refresh","PATCH /api/v0/me","POST /api/v0/auth/refresh","PATCH /api/v0/me/profile"]}
  (C5: 6005 ms)
PASS C6a {"before":"/settings/profile/general","shown":"maintenance","whileDown":{"url":"/settings/profile/general","record":true},"recoveredAfterMs":4962,"after":"/settings/profile/general"}
  (C6a: 14461 ms)
PASS C6b {"unavailable":true,"retry":true,"whileDown":{"url":"/settings/profile/general","record":true},"refreshFailures":["500 POST /api/v0/auth/refresh"],"recoveredAfterMs":1349,"after":{"url":"/settings/profile/general","title":"Profile - General settings"}}
  (C6b: 3653 ms)
PASS C6c {"unavailable":true,"recoveredAfterMs":2298,"url":"/settings/profile/general"}
  (C6c: 3080 ms)
PASS C7 {"shown":true}
  (C7: 181 ms)
PASS C8 {"profile":true,"next":true,"onboarding":{"apiFailures":[],"oldApi":[],"pageErrors":[],"consoleErrors":[],"csp":[]},"onboardingApi":["GET /api/v0/instance","POST /api/v0/auth/register","GET /api/v0/me","GET /api/v0/me/profile","PATCH /api/v0/me","PATCH /api/v0/me/profile"],"landed":"/create-workspace","landing":{"apiFailures":[],"oldApi":[],"pageErrors":[],"consoleErrors":[],"csp":[]},"landingApi":["GET /api/v0/instance","POST /api/v0/auth/login","GET /api/v0/me","GET /api/v0/me/profile"]}
  (C8: 1058 ms)
PASS C9 {"pages":[{"path":"/","landed":"/","csp":[],"header":"default-src 'self'; script-src 'self' 's","apiFailures":[],"consoleErrors":[]},{"path":"/sign-up","landed":"/sign-up","csp":[],"header":"default-src 'self'; script-src 'self' 's","apiFailures":[],"consoleErrors":[]},{"path":"/onboarding","landed":"/onboarding","csp":[],"header":"default-src 'self'; script-src 'self' 's","apiFailures":[],"consoleErrors":[]},{"path":"/create-workspace","landed":"/create-workspace","csp":[],"header":"default-src 'self'; script-src 'self' 's","apiFailures":[],"consoleErrors":[]},{"path":"/settings/profile/general","landed":"/settings/profile/general","csp":[],"header":"default-src 'self'; script-src 'self' 's","apiFailures":[],"consoleErrors":[]},{"path":"/settings/profile/security","landed":"/settings/profile/security","csp":[],"header":"default-src 'self'; script-src 'self' 's","apiFailures":[],"consoleErrors":[]},{"path":"/settings/profile/preferences","landed":"/settings/profile/preferences","csp":[],"header":"default-src 'self'; script-src 'self' 's","apiFailures":["404 GET /api/timezones/"],"consoleErrors":["Failed to load resource: the server responded with a status of 404 (Not Found)"]},{"path":"/settings/profile/api-tokens","landed":"/settings/profile/api-tokens","csp":[],"header":"default-src 'self'; script-src 'self' 's","apiFailures":["404 GET /api/users/api-tokens/"],"consoleErrors":["Failed to load resource: the server responded with a status of 404 (Not Found)"]}]}
  (C9: 8258 ms)
PASS C11 {"untrusted":{"status":201,"ip":"127.0.0.1/32"},"warnedUntrusted":["time=2026-09-27T20:01:51.407+08:00 level=WARN msg=\"ignored X-Forwarded-For from a peer that is not a trusted proxy: behind a reverse proxy, add its address to server.trusted_proxies, or every client counts as the proxy\" peer=127.0.0.1"],"trustedProxies":"127.0.0.1/32","trusted":{"status":201,"ip":"172.17.0.1/32"},"direct":{"status":201,"ip":"127.0.0.1/32"}}
  (C11: 6869 ms)
PASS C10 {"navigateWarnings":[],"warnings":6,"sample":["C6b: Canvas2D: Multiple readback operations using getImageData are faster with the willReadFrequently attribute set to true. See: https://html.spec.whatwg.org/multipage/canvas.html#concept-canvas-will-read-frequently","C6b: Canvas2D: Multiple readback operations using getImageData are faster with the willReadFrequently attribute set to true. See: https://html.spec.whatwg.org/multipage/canvas.html#concept-canvas-will-read-frequently","C9 /settings/profile/general: Canvas2D: Multiple readback operations using getImageData are faster with the willReadFrequently attribute set to true. See: https://html.spec.whatwg.org/multipage/canvas.html#concept-canvas-will-read-frequently","C9 /settings/profile/security: Canvas2D: Multiple readback operations using getImageData are faster with the willReadFrequently attribute set to true. See: https://html.spec.whatwg.org/multipage/canvas.html#concept-canvas-will-read-frequently","C9 /settings/profile/preferences: Canvas2D: Multiple readback operations using getImageData are faster with the willReadFrequently attribute set to true. See: https://html.spec.whatwg.org/multipage/canvas.html#concept-canvas-will-read-frequently"]}
results: $M2TMP/p4branchchecks/out/076619a5-results.json
```
