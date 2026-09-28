# M2/Codex 修复 codex-fixes：设计说明（spec）

| 项 | 内容 |
|---|---|
| 里程碑 | M2 账户认证（`docs/v0/M2-auth`） |
| 日期 | 2026-09-28 |
| 基点 | `37b7e9c`（在 `9ef4a4a`"Merge M2 closeout; M2 is done"之上加了 Codex 的评审报告） |
| 上级 | [M2 设计](../M2-design.md) 3.10、3.11、6.1、7.1、7.5、7.7、8.5、8.7、11.2、第 15 节、新的 17.3；[总体设计](../../v0-design.md) 3.5、4.3、7.7 |
| 输入 | [Codex 对全部 M2 的对抗性评审](../reviews/M2-codex-adversarial-review.md)（Critical 1、Important 2、Minor 3，另有第 6、7 节）；控制者的分诊（有约束力） |
| 计划 | [plans/codex-fixes.md](../plans/codex-fixes.md)，Task 0（随本 spec 提交）和 11 个 Task |
| 原型 | `$M2TMP/codexfix/proto`（本分支在 `37b7e9c` 的克隆），全部改动、先红后绿、变异和门禁都在它上面做过（附录）；计划的块由原型的文件生成，依次应用到 `37b7e9c` 的另一份干净副本上，得到的树与原型逐字节相同（附录 D） |
| 修订 | 第一稿（`f66b7a92`）经过预检（`$M2TMP/m2-codex/preflight.md`：High 2、Medium 4、Low 8），控制者接受全部发现（`preflight-rulings.md`）；这一稿按裁定修订，每一条的落点见 2.13，原型重新验证 |

---

## 1. 目标

负责人的指示：仔细处理 Codex 的报告，干净地完成 M2，不开始 M3。所以报告的每一条都在 M2 之内按根因修好，每一处修改都有能失败的回归证据（先在原型上复现、再修，变异证明测试咬得住）。修完之后 M2 仍是"已完成"：总体设计 9.4 不变，这一阶段合并时 M2 再次完成。

具体是：

1. **请求体只有一种读法**（Critical 1）：重复的成员名和不是合法 Unicode 的字符串在平台的边界上得到 400，并证明结构检查接受的请求体，生成的解码器读不出结构检查没看过的东西；检查的代价跟着请求体走，任何请求体都不能让它分配请求体的平方（预检 H1），也不能让 400 的回答随请求体变大：报出的路径最长 256 字节，同一个问题只报一次（最终评审的修复波）。
2. **令牌管理器在请求和锁回来之后重读记录**（Important 1）：修这一类，不只修退出那一行。
3. **服务角色的权限只有一个出处**（Important 2）：一个 SQL 文件，一个用真实 Postgres、恰好这些权限运行 nerve 的测试，以及权限怎样保持完整。
4. **会话的主题只在一处设置**（Minor 1）。
5. **A8–A10 的页面版本和接口版本调用同一组业务断言**（Minor 2）。
6. **文档的事实漂移改正**（Minor 3）。
7. **一个 store 的修改一个接一个发出**（报告第 6 节，Codex 的页面实验推翻了 P5 的假定）。
8. **README 和 8.5 的恢复说明完整**（报告第 7 节）。
9. **`AllowAll` 拒绝重复的桶和键**（P2 遗留，Codex 同意是潜在问题）。

---

## 2. 分诊和每项交付物的规则

### 2.1 分诊（控制者的裁定，有约束力）

| 来源 | 发现 | 裁定 | 本阶段的落点 |
|---|---|---|---|
| Critical 1 | `bodyshape` 把对象读成 map（同名成员后一个为准），生成的 handler 按结构体解码（同名对象合并，字段名不区分大小写）；Codex：`{"onboarding_step":{"WORKSPACE_INVITE":true}}` 400，追加 `"onboarding_step":{}` 后 200 并写入 `workspace_invite = true` | 在平台边界 `bodyshape` 修；重复键（按解码后的名字）和非 UTF-8 一律 400；证明不变式；不加模块的逐字段检查，不手改生成代码；`Content-Type` 保持宽松 | 2.2；Task 1、2 |
| Important 1 | 旧标签页的退出在租约过期后回来，删掉另一个标签页的新记录 | 审计令牌管理器中所有 `await` 之后写记录或改状态的路径，逐条重读；强制顺序的单元测试；每个重读都做变异 | 2.3；Task 3 |
| Important 2 | README 只写了 `goose_db_version`；少了 River 的权限时 `/readyz` 200 而 River 42501；`REINDEX … CONCURRENTLY` 要 `MAINTAIN` | 一个仓库内的 SQL 文件；testcontainers 的集成测试（就绪、River 启动并完成清理、运行角色能 `REINDEX INDEX CONCURRENTLY river_job_pkey`）；决定并证明权限怎样保持完整；M4、M8 的交接 | 2.4；Task 4、10 |
| Minor 1 | 自助停用后当前标签页保留旧账户的主题 | 会话变化时在一处设置主题（`StoreWrapper` 或 `store-context`），删除按钮上的 `setTheme` | 2.5；Task 8 |
| Minor 2 | A8–A10 共用查询，不共用业务断言 | `e2e/fixtures/assert/identity.ts` 中带参数的小断言，两个版本都调用；不建通用框架 | 2.6；Task 6 |
| Minor 3 | 11.2 的"标准库和 oapi-codegen 的运行时类型"；收尾 spec B.5 第 5 条、收尾 review 第 7 节第 5 条的"15 个操作、约 40 行" | 改正，并查找别处的重复 | 2.7；Task 9、10 |
| 第 6 节 | P5 假定 nerve 按发出的顺序处理修改；Codex 的页面实验：页面 `en`，nerve 保存 `zh-CN` | 每个 store 的修改串行，失败也放行；删除变成死代码的"丢弃较旧的应答"；审计资料和账户的每个写入者和 PAT store；单元测试和页面故事 | 2.8；Task 7 |
| 第 7 节 | README `:121` 把"逐个撤销 PAT"和 `reset-password` 并列 | 自助的路径写完整（按代码核对需要哪几步），管理员的路径仍是 `reset-password`；8.5 同改 | 2.9；Task 9、10 |
| P2 遗留 | `AllowAll` 同一个桶和键出现两次 | 改动小就作为编程错误拒绝 | 2.10；Task 5 |
| 第 6 节 | `DateDropdown` 的键盘（M4 交接第 10 节）；时区按钮的 Tab 顺序（M3 交接第 14 节） | **保持推迟，不改**：它们是共用组件，其他调用方在 M2 到不了的页面上；Codex 接受 M3、M4 作为期限；交接已有关闭条件 | 2.11 |
| 第 6 节其余 | 已知事项的其余判定 | 同意 Codex，不改 | — |

全文扫描之后没有别的架构层面的问题：`duplicate` 是 3.11 封闭集合中加的一个字段码（前后端契约只多了"重复键和非 UTF-8 是 400"，在分诊允许的范围内）；其余改动都在一个模块或一个组件之内。

### 2.2 请求体只有一种读法（Critical 1）

**规则**（M2 设计 3.11 的新条目，总体设计 3.5，差异清单第三节）：

1. 同一个对象里同名的成员出现两次：400 `bad_request`，`errors[]` 中一项 `{field: <这个成员的路径>, code: duplicate}`，一个名字只报一次（出现三次也是一项）。名字按**解码之后**比较：`"name"` 和把其中一个字母写成 `\u` 转义的名字是同一个名字。任何深度都算：嵌套对象、数组里的对象、开放的 map、契约没有声明的地方。
2. 字符串（成员名或值）不是合法的 Unicode：400，`{field, code: invalid_format}`。包括不是 UTF-8 的字节，以及不成对的代理项转义（高位后面不是低位、单独的低位、结尾的高位）：两者都被 `encoding/json` 换成 U+FFFD，领域拿到的不是客户端写的文字。成对的代理项、转义的反斜杠后面跟 `u`、U+FFFD 本身都合法。
3. 这两项先于结构检查：有这两种问题的请求体只得到它们。结构在请求体只有一种读法之后才检查；一个有两种读法的请求体，它的"结构"本来就没有定义。
4. 字段码：重复用新码 `duplicate`（`api/common.yaml` 的 `FieldError.code` 枚举、`shared.FieldCodes()`、前端的字段码表和中英文案同时加入；`TestFieldCodesAreTheContractsEnum` 和 `authentication.helper.test.ts` 核对它们相等）；不合法的 Unicode 用已有的 `invalid_format`（它本来就表示"这个值的写法不对"）。
5. `Content-Type` 保持宽松：`text/plain` 的请求体同样检查、同样解码。Nerve 没有 Cookie 认证，非公开操作都要 `Authorization` 头，跨站的简单请求带不上它，没有 CSRF 的风险。
6. 做法：`platform/httpserver/bodyshape/ambiguity.go`，一个只用标准库的扫描器，在 `Check` 按表检查之前把 `json.Valid` 已确认的请求体走一遍：对象里用解码后的名字计数，字符串用 `utf8.Valid` 和代理项配对判断。它就是 I-JSON（RFC 7493）的两条要求。只有成员名要解码（比较用）；值的字符串在字节上检查，不解码。
7. **代价跟着请求体走**（预检 H1）：扫描和按表检查共用 `bodyshape.go` 的 `problems`。
   - 路径只在报问题时写出。第一稿给每个值写出它的路径：祖先的路径同时留在递归的栈上，内存是深度的平方乘名字的长度。`json.Valid` 允许嵌套一万层，预检用 identity 的表量得：949,906 字节的请求体（9,999 层，名字 90 个字符）320 ms、4.4 GB；这一稿当初（路径是一个段的栈）14 ms、3.1 MB，修复波之后 11 ms、2.4 MB。
   - 一个请求体最多 16 个问题，满了就停（扫描不再往下读，按表检查不再下降）。只有懒的路径还不够：每个问题的路径可以和请求体一样长，问题可以每隔几个字节一个。预检的"300 KB 的名字下面 60,000 个不是 UTF-8 的字符串"（540 KB）在第一稿上是 60,000 个问题、每个带 300 KB 的路径，17 GB；这一稿当初 3 ms、5.7 MB，16 个问题；修复波之后 2.5 ms、0.9 MB，1 个问题（路径截短之后都相同，只报一次）。
   - **报出的路径最长 256 字节，同一个问题只报一次**（最终评审的 Critical 和 m2，修复波）。只限问题的个数也还不够：16 个问题可以都在同一个约 1 MiB 的名字下面，每个都重复这个名字。最终评审用真实的 `Check` 加上问题的编码，向匿名的 `POST /api/v0/auth/login` 发了三个约 1 MiB 的请求体（`<` 组成的名字下 16 对重复的键；同一个名字下 16 个值为 `"\xff"` 的成员；9,990 层 90 字符的名字、最底下 16 个问题），`Check` 分配 18、18、99 MB，编码分配 563、564、486 MB，回答 100.6、100.7、86 MB（`WriteProblem` 保留 Go 默认的 HTML 转义，`<` 写成 6 个字节）。修复：
     - 更长的路径截短：保留开头，在 UTF-8 字符的边界上截断，以 `…`（U+2026）结尾，合计不超过 256 字节。读的时候路径是一段只留前 257 字节的文本，另记每一段从哪里开始，读完这一段就截回去；报问题只复制这不到 257 个字节，与路径多长、多深无关。一个回答里的路径最多 16 × 256 字节（转义之后约 24 KiB）。契约里真实的路径最长几十个字节，M4 的 `tags[10].name` 远不到 256。`api/common.yaml` 的 `FieldError.field` 写明截短（`make gen`）；`WriteProblem` 不改，HTML 转义是给显示在 HTML 里的 JSON 的纵深防御，截短已经限住大小。
     - 路径和码都与已报出的相同的问题不再报，也不占上限（在最多 16 个里线性地找）。同一个名字重复时的非 UTF-8 值、重复的非 UTF-8 名字原来各报两次。两个长路径截短之后相同时只报一次，客户端本来也分不出它们。
     - 为什么路径不再是段的栈：重复的问题不占上限之后，一个请求体可以一再报同一个问题。若每次报问题都从段的栈重新写出截短的路径，9,999 层空名字（`{"":`）下重复同一个问题的约 1 MiB 请求体要 1.3 秒（每次都走一遍一万层的栈），300 字节的名字下重复 200,000 次要分配 50 MB；现在各 12 ms、0.8 MB。
     - 修复之后，三个请求体的 `Check` 各分配 2.0、2.0、1.4 MiB（`TestCheckCostsAboutTheBody`，修复之前各 18.0、18.0、96.6 MiB），回答各 1,735、1,732、1,725 字节，每个只有一个问题（`TestTheAnswerToABrokenBodyStaysSmall`，修复之前 100,626,389、100,626,341、86,474,917 字节）。
   - **按表检查也有同样的放大，同样限制。** 它的问题的路径由 schema 声明的名字和客户端写的名字组成：开放 map 的键是客户端写的，它下面的每个问题都带着它（`{"<64 KB 的键>":{8,000 个未声明的成员}}` 在第一稿上分配 1.1 GB）；递归的 schema 让路径随请求体的深度变长。所以按表检查用同一个路径、同一个上限和同一个截短。它按名字的顺序读对象的成员（map 的顺序是随机的），超过 16 个问题的请求体每次得到同样的 16 个。
   - 按表检查的另一种代价不是问题：它沿 schema 下降，每层把对象的成员 `json.Unmarshal` 进 `map[string]json.RawMessage`，复制一次，代价是请求体乘 schema 的层数。M2 的 8 个请求体 schema 最深两层（`onboarding_step`），没有数组、开放 map，也没有递归，所以是请求体的常数倍。第一个递归的请求体 schema（生成器支持）会让它变成请求体乘请求体的深度：那时按表检查改为一遍读完；第一个数组或开放 map 的请求体 schema 到来时，代价测试加上它（M4 交接第 4 节）。
   - `TestCheckCostsAboutTheBody`（不并行，`TotalAlloc` 数整个进程）：嵌套到 `json.Valid` 允许的最深的对象和数组、长名字下的 8,000 个非 UTF-8 字符串、开放 map 的 64 KB 键下的 8,000 个未声明的成员，每个 `Check` 分配不超过 64 MiB。这一稿当初各约 1.5、1.0、2.5、5.8 MiB，修复波之后各约 0.9、0.4、0.2、3.4 MiB；第一稿各 844、150、564、1130 MiB（附录 B 的 eager-paths）。修复波加了最终评审的三个请求体，每个 `Check` 分配不超过请求体的 6 倍（整条路径的 16 份约是请求体的 16 倍，读请求体约是 2 倍：长名字解码一次），和 300 字节的名字下同一个问题重复 200,000 次的一个（同样 6 倍：每次写出路径约是 50 倍）；每个请求体的回答里，路径合计不超过 16 × 256 字节。`TestCheckListsAtMostSixteenProblems`：20 个重复的名字只报前 16 个（按请求体的顺序），20 个未声明的名字（倒着写）只报按名字排在前面的 16 个；一个问题重复 20 次、再有 20 个重复的名字，那个问题只报一次（重复和非 UTF-8 各一项），不占上限，后面报 14 个名字。`TestCheckCutsALongPathShort`：256 字节的路径（顶层和嵌套的）不截，257 字节的截成 253 字节加 `…`；截断点落在 `é` 的第二个字节、`中` 的第二和第三个字节时退到字符的开头。
   - 这是"一次收集全部问题"的第二个例外（D11）；契约的 `errors` 本来就只说"请求中不合格的字段"，不改。

**不变式：结构检查接受的请求体，生成的解码器读不出结构检查没看过的东西。** 生成的 strict handler 用 `encoding/json` 把同一份字节解码进生成的结构体。Go 1.27.1 的 `encoding/json`（v1 的语义）有四种会和 map 的读法分开的行为，逐条对照：

| 解码器的行为 | 为什么在接受的请求体上不发生 |
|---|---|
| 同名成员：结构体字段合并（对象）或后一个为准；map 后一个为准 | 规则 1：每个对象里的名字按解码后比较各不相同，map 和结构体读到同一组成员 |
| 字段名先精确匹配，再**不区分大小写**地匹配 | 每个对象 schema，组件和内联的，都是 `additionalProperties: false`，结构检查接受的名字恰好是声明的名字，解码器用精确匹配就找到，退路用不上。只差大小写的名字（`Name`）是未声明的字段，400 `not_allowed`（单元测试有这一例）。前提由 `apitest` 的 `closedObject` 守住：第一稿时它只查 `components/schemas` 的组件（预检 M1），内联的对象、数组项里的对象不查；这一稿从 `schema()` 调用它，递归到每个内联的子 schema，`components()` 不再另调（同一个组件不报两次）。`rules_cases_test.go` 加两个反例（属性里的内联对象、数组项里的内联对象），现在的契约照样通过。以后要用开放的 map，先改这条规则（生成的代码把 map 的键原样读入 `map[string]T`，不区分大小写的匹配只在结构体的字段上；M4 交接第 4 节） |
| 转义：名字和值都按 JSON 转义解码 | 结构检查用同一个 `encoding/json` 解码名字（`json.Unmarshal` 进 `string`）、比较解码后的名字；格式检查本来就把原始值 `json.Unmarshal` 进映射的类型（3.11） |
| 不合法的 UTF-8、落单的代理项换成 U+FFFD | 规则 2：接受的请求体里没有这两种，解码出的文字就是客户端写的 |

此外，请求体是一个 JSON 值，前后只有 JSON 空白（`Check` 原有的 `json.Valid` 和去掉空白的一步），所以两边读的是同一个文档。

**回归**：`bodyshape` 的单元测试（顶层、嵌套、转义等价、数组里的对象、开放的 map、未声明的位置、非 UTF-8 的值、名字、数组项、落单的代理项，以及不该拒绝的对照；修复波加了高位代理项后面跟着不是代理项的转义，和最终评审的两个重复报出的例子）；上限、代价和截短的三个测试（第 7 条）；`closedObject` 的两个内联反例；第四个整程序测试的第 7–9 项让每个带请求体的操作（M2 有 8 个）都拒绝重复的第一个字段、重复的嵌套字段和第一个字符串字段里的非 UTF-8 字节；整程序测试 `TestTheAnswerToABrokenBodyStaysSmall`（修复波）把最终评审的三个请求体发给匿名的登录接口，回答不超过 32 KiB、每个路径不超过 256 字节；A10 的接口版本原样发 Codex 的两个请求：第一个 400 `not_allowed`，第二个 400 `[{onboarding_step, duplicate}]`，数据库中的引导步骤不变。

### 2.3 请求和锁回来之后先重读记录（Important 1）

**规则**（M2 设计 7.1，总体设计 4.3）：每个请求的 `await` 回来之后，以及锁把失败的第一次续期交回来之后，写或删 `nerve.auth`、改本标签页的状态之前，都再读一次记录；它已不是这次操作所属的会话（`login_id` 变了，或者记录没了），就不写、不删，跟随记录（`#switchTo`）。登录和注册等到锁之后写新会话的记录，不重读：按"最后登录为准"（P4）。第一稿写的是"锁内每个 `await`（请求、等锁）"，比代码宽（预检 L5）：登录在等锁之后不重读，第一次续期的重读在锁外；文件头的注释、M2 设计 7.1、总体设计 4.3 照代码改写。

**逐条审计**（`web/apps/web/core/lib/auth/token-manager.ts`）：

| 路径 | `await` 之后做什么 | 原来 | 现在 |
|---|---|---|---|
| `signIn`（注册同一条路） | 锁内没有 `await`：生成 `login_id`、写记录、设状态 | 无问题（新登录替换记录是"最后登录为准"，P4 接受） | 不变 |
| `#refresh` | `logout` 之外唯一在锁内发请求的：写回新令牌、401 时删记录 | 已在写回前重读（R4） | 不变 |
| `signOut` | `logout` 回来后删记录、设为未登录 | **不重读**：租约过期后另一个标签页的新登录被删（Codex 的发现） | 经 `#end(loginId)`：重读，仍是自己的才删，否则跟随 |
| `endSession` | 锁内没有 `await`：删记录 | 读和删之间没有 `await` | 改用同一个 `#end`，行为不变 |
| `#firstRefresh`（启动、重试） | 续期失败（429、5xx、网络）之后设为"会话暂不可用"并排定重试 | **不重读**：`navigator.locks` 把失败交回来是在之后的任务里，另一个标签页的登录可以先到，标签页先跟随了新会话，然后被设成旧会话的"暂不可用"（审计中找到的同类第二处） | 重读，已是另一个会话就跟随，不设状态、不排重试 |
| `handleStorageChange`、`#switchTo`、`#follow` | 同步，读的就是此刻的记录 | 无问题 | 不变 |
| `start`、`retry` | 只调用上面几条 | — | — |

剩下的窗口（租约不是原子的，两次读之间另一个标签页写入）是已知的租约局限（7.1、§16），不是本阶段的范围。

**回归**（`token-manager.session-change.test.ts`，用现有的假锁、假租约和 `gate` 强制顺序，不用 `sleep`）：①租约在退出的请求期间过期，另一个标签页写入新记录，退出的请求才回来：新记录留下，两个标签页都在新会话上，只发了一次 `logout`；②启动时第一次续期得到 503，`navigator.locks` 把失败交回来之前另一个标签页登录：标签页跟随新会话，状态序列是 `starting X → signed-in Y`，60 秒后也没有为旧会话重试。

### 2.4 服务角色的权限（Important 2）

**规则**（M2 设计 6.1、8.7，README"部署"）：

1. `deploy/runtime-grants.sql` 是服务角色权限的唯一出处，授权给组角色 `nerve_runtime`；服务登录用的角色加入这个组。每次 `nerve migrate up` 之后，以表的所有者执行它（幂等）。
2. 内容：`public` 的 `USAGE`；四张业务表的 `SELECT, INSERT, UPDATE, DELETE`；River 的四张表（`river_job`、`river_leader`、`river_queue`、`river_notification`）同样四种；两个序列（`river_job_id_seq`、`river_notification_id_seq`）的 `USAGE`；`river_job` 的 `MAINTAIN`（River 每天的 `REINDEX INDEX CONCURRENTLY`，PostgreSQL 17 起）；`goose_db_version` 的 `SELECT`（`/readyz`）。规则是"除 goose 的记录以外每张表读写，goose 的记录只读，视图只读，写的表的序列可用，`river_job` 可重建索引"：按表给读写（DML），不给 `TRUNCATE`、`REFERENCES`、`TRIGGER` 和 DDL（预检 L3：第一稿说"只给需要的"，而 nerve 从不删 `users`、`profiles`、`api_tokens` 的行，River 只删 `river_notification` 的；按表给 DML 是 D4 的取舍，不再细分到语句）。
   - **范围**（预检 L2）：文件列出表、视图、物化视图和序列。函数和类型不列：PostgreSQL 默认让 PUBLIC 执行函数、使用类型，River 的 `river_job_state_in_bitmask` 和 `river_job_state` 靠的就是它；迁移收回了 PUBLIC 的这些权限时，把它们加进文件。文件头和 README 写明这个范围。
   - **少了权限时是什么样**（预检 M4）：少了 River 的权限，`/readyz` 仍是 200，River 的任务（会话清理）和索引重建因 42501 失败，只记在日志里；少了业务表的权限，接口请求失败；少了 `goose_db_version` 的读，`/readyz` 是 503（`routes.go` 依次检查，迁移的检查读 goose 的表）。README 和 M2 设计 6.1 照此写，第一稿笼统地写成"`/readyz` 仍是 200，只记在日志里"。
3. **怎样保持完整：逐个列出，加一个完整性测试（推荐并采用）**，不用 `GRANT … ON ALL TABLES IN SCHEMA` 加 `ALTER DEFAULT PRIVILEGES`。比较：

   | | 逐个列出 + 测试 | `ALL TABLES` + 默认权限 |
   |---|---|---|
   | 权限的粒度 | 按表给读写（DML），不给 `TRUNCATE`、`REFERENCES`、`TRIGGER` 和 DDL；`goose_db_version` 只读 | `ALL TABLES` 给每张表同样的权限，goose 的记录也可写（写成 `GRANT ALL` 还会连 `TRUNCATE`、`REFERENCES`、`TRIGGER` 一并给出）；按表区分要另写 |
   | 漂移 | 新迁移加表而文件没跟上，CI 中的测试失败，信息指出文件 | 默认权限只对执行 `ALTER DEFAULT PRIVILEGES` 的那个角色以后建的对象生效：换了执行迁移的角色（常见于运维交接），新表悄悄没有权限，要到部署之后 River 或请求失败才发现，而且 `/readyz` 仍是 200 |
   | 运维 | 每次迁移之后执行一次（幂等），写进 README | 一次设置 |
   | 证明 | 测试在 CI 中每次跑 | 部署环境里才能验证 |

   漂移在 CI 里被抓住，比在部署之后被抓住好；多出的运维一步写进 README，与"先 `migrate up` 再 `serve`"放在一起。
4. 服务角色不是表的所有者，删不掉索引。River 的重建被停机打断、留下 `*_ccnew` 索引时，River 自己在停机时的删除也会失败，由表的所有者删除（README）。原型的探针证实：服务角色 `DROP INDEX CONCURRENTLY` 得到 `must be owner of index`（42501），所有者成功。

**测试**（`server/internal/bootstrap/runtime_role_test.go`，`pgtest` 的 testcontainers Postgres 18，角色实验只在这里做）：`newSplitRoles` 建一个属于所有者角色的库，所有者迁移、执行文件；服务角色是 `nerve_runtime` 的成员。

- `TestTheRuntimeRoleServesWithTheGrantsFile`：以服务角色启动整个 nerve（`auto_migrate: false`）：`/readyz` 200；River 启动（日志有 `jobs started`，没有 `permission denied`），会话清理完成（过期的会话删除、未过期的留下）；`river.ReindexerIndexNamesDefault()` 的每个索引（含 `river_job_pkey`）都能以服务角色 `REINDEX INDEX CONCURRENTLY`。
- `TestTheGrantsFileCoversEveryRelationAndFunction`：`public` 里的每张表、分区表、视图、物化视图、序列和函数，服务角色的实际权限（`has_table_privilege` 的八种、`has_sequence_privilege` 的 `USAGE`/`SELECT`/`UPDATE`、`has_function_privilege` 的 `EXECUTE`）等于规则 2：表 DML（`river_job` 另有 `MAINTAIN`，goose 的记录只读），视图和物化视图 `SELECT`，序列 `USAGE`，函数 `EXECUTE`；多给、少给都失败。第一稿只查表和序列的 `USAGE`，预检的变异迁移加了一个视图，没有被发现（预检 L2）。

**交接**：M4 交接第 6 节（第一个 River 任务的一节）加"服务角色的权限"一条：M4 的表、视图和 River 升级的新表同一个提交进文件，两个测试守住；M8 交接第 3 节（部署）加"分用两个数据库角色"一条和它的关闭条件，部署核对包括以服务角色执行一次 `REINDEX INDEX CONCURRENTLY river_job_pkey`（预检 L7，Codex 报告第 8 节要的就是这一步）。

### 2.5 会话的主题（Minor 1）

**规则**（M2 设计 7.7，总体设计 7.7"页面级的状态跟随当前的会话"）：页面的主题只在 `StoreWrapper` 按会话设置：没有会话（`useSession().status === "signed-out"`：本标签页或别的标签页退出、账户停用、续期被拒）时是默认的"跟随系统"；有会话时，这个会话第一次取到资料时用资料的主题，每个会话一次（用这一代的 `UserStore` 认出会话）。之后的修改由改它的组件在 nerve 应答成功之后设置（`theme-switcher.tsx`，不变）。

- **决定是一个纯函数**（预检 M2）：`core/lib/wrappers/session-theme.ts` 的 `sessionTheme(status, session, profileTheme, themedBy)` 返回要设的主题（或不设）和现在的 `themedBy`；`StoreWrapper` 的效果只调用它、记下 `themedBy`、设主题。"每个会话一次"守的是：next-themes 0.4.6 的 `setTheme` 是 `useCallback(…, [theme])`，主题一变效果就重跑；没有这条守卫，资料过时的标签页会在别的标签页改了主题（经 next-themes 的存储同步）时把旧主题写回去，两个标签页来回改。第一稿里这条守卫没有测试：去掉它的变异在 e2e 上 24 个全过。`session-theme.test.ts` 的 5 个单元测试（顶层导入）：没有会话是 `"system"`；会话的第一份资料是它的主题；同一个会话再跑、资料的主题变了、会话暂不可用，都不设；新的 `UserStore` 是新的会话，设它的主题；资料到来之前不设。

- 删除：`store-context.tsx` 的 `startSession` 写 localStorage `theme`（它只改存储，不改 next-themes 已经在用的状态，这正是 Codex 看到的现象）；切换账户弹窗退出之后的 `setTheme("system")`（按钮上的补丁）。
- 为什么放在 `StoreWrapper` 而不是 `store-context`：`store-context` 的 `startSession` 是模块里的函数，不在 React 的渲染之内，拿不到 next-themes 的 `setTheme`；`StoreWrapper` 已经负责资料的主题和语言，按 `useSession()` 的状态设置，任何结束会话的路径（包括别的标签页、续期被拒）都经过它。
- 跟随另一个标签页的登录（X → Y）时，页面保持显示的主题，直到 Y 的资料到来再换成 Y 的（旧的可观察行为，同一账户重新登录时不闪一下）。语言仍由 `startSession` 设为默认、由 `StoreWrapper` 按资料设置，不变。

**回归**：`session-theme.test.ts`；A12 的页面版本（资料的主题是深色，页面先是 `data-theme="dark"`，停用回到登录页后是 `light`）；A5 的页面版本（续期被拒结束会话，同样）；A6 的第一个页面版本（在一个标签页退出，两个标签页都回到默认主题：守住删除按钮补丁之后的行为）。

### 2.6 共用的业务断言（Minor 2）

`e2e/fixtures/assert/identity.ts` 加三个带参数的断言，页面版本和接口版本都调用，差异由参数表达（A7 的 `expectPasswordChanged` 是范例）：

| 断言 | 内容 | 调用者 |
|---|---|---|
| `expectAccountChanged(db, email, before, change)` | 账户等于 `before` 加上 `change`，除 `updated_at` 前移以外别的都没变；返回现在的账户 | A8 的两个页面版本和接口版本 |
| `expectPreferences(db, email, preferences)` | 资料的主题、语言、每周第一天等于给出的值 | A9 的全部版本（含本阶段新加的故事），代替原来文件内的 `profileOf` |
| `expectProfileStepTaken(db, email, firstName, stepsBefore)` | 名字是 `firstName`，引导步骤是 `stepsBefore` 加上资料一步已完成，其余保持 | A10 的两个页面版本（经 `takeProfileStep`）和接口版本（`stepsBefore` 含 `workspace_join: true`：保持与重置不同） |

`onboardingStepsOf` 的返回类型从 `unknown` 改为 `OnboardingSteps`。没有通用的断言框架。比原来多守住的一点：`expectAccountChanged` 用"等于"而不是"包含"，只改名字的修改连带改了时区，页面版本会失败（变异表）。

### 2.7 文档的事实漂移（Minor 3）

- M2 设计 11.2 第 1 条：`bodyshape` 运行时只用标准库（核验 F1 之后，格式检查把原始值 `json.Unmarshal` 进映射的类型）。
- 收尾 spec 附录 B.5 第 5 条、收尾 review 第 7 节第 5 条：`identity.yaml` 623 行、13 个操作（另外 2 个在 `instance.yaml`），约每个操作 48 行。
- 查找其他重复：`grep -rn -E "15 个操作|约 40 行|oapi-codegen 的运行时类型" docs README.md`。收尾 plan 和 M1-P3 交接里的"接口描述共 15 个操作、12 个路径"说的是整个接口描述（13 + 2），是对的；M2 设计决策点 2 表中的"约 40 行"是 `users create` 命令的估计，无关。
- 改完之后的核对（计划 T10）只找三句错话本身，排除本阶段的计划（它写着这条命令，旧文本块里有原句）和同名的 spec，应当没有输出。第一稿的核对用上面这组宽的模式，预期"只剩 Codex 报告里的引文"，实际命中 13 处（更正本身、17.3、spec、计划），执行者会停下（预检 M3）。

### 2.8 一个 store 的修改一个接一个发出（第 6 节）

**规则**（M2 设计 7.5、7.7，总体设计 7.7）：`ProfileStore.updateUserProfile` 和 `UserStore.updateCurrentUser` 经 `core/lib/one-at-a-time.ts` 的 `oneAtATime()` 队列发出：每个修改在前一个有了应答或失败之后才发出，失败也放行队列；应答照旧写进 store。于是 nerve 按做出的顺序应用有了应答的修改，最后一个应答就是 nerve 保存的值。

- **只对有了应答的修改承诺顺序**（预检 L4）：没有应答的失败（请求到了 nerve 之后连接断了）仍可能在下一个修改之后才被应用；一个一直不结束的请求挡住这个 store 的队列（网页的客户端没有请求的期限），nerve 在 `server.request_timeout` 之内应答它收到的每个请求。队列的注释、M2 设计 7.7、总体设计 7.7 照此写，不再说"包括超时"。
- **只排队修改；取数不排队**（预检 L8）：会话开始时取资料（GET）的应答晚于一个修改的应答到来时，store 显示取数时的值，与以前相同（删掉的代码也没有管它）。7.7 记这一句。

- 删除死代码：两个 store 的 `updatesSent`、`updateWritten` 和"比已写入的旧就丢弃"的分支；文档注释改写。
- **写入者审计**：资料和账户的修改都经这两个方法：preferences（主题、语言、每周第一天、时区）、general（名字）、新手引导（名字、步骤、`finishUserOnboarding`、`updateTourCompleted`）、主题切换（`updateUserTheme`）、命令面板的主题命令（同一个 `updateUserTheme`）。`services/user.service.ts` 的两个 PATCH 只被这两个 store 调用（`grep`）。
- **PAT store 审计**：创建和撤销不依赖 nerve 的处理顺序：撤销要用创建返回的 id，两者本来有先后；两个并发的创建只影响列表中的先后，列表"取最新"靠提交先于确认。不改。
- 范围：一个标签页之内（一个 `RootStore` 一个队列；换会话就是新的 store 和新的队列）。多个标签页、多个客户端之间仍以后写入的为准（v0 不做乐观锁）。

**回归**：两个 store 的单元测试（`FakeNerve` 由测试决定何时应答，假时钟）：第二个修改在第一个应答之前不发出，应答之后发出；第一个被拒（500）或没有应答（网络错误）也放行第二个。A9 的新页面故事就是 Codex 的实验：第一个修改（简体中文）在路由上扣住，再选 English；一个在第二次选择之后发出、已经应答的请求（`GET /api/v0/instance`）证明第二个修改没有发出；放行之后 nerve 收到的顺序是 `[zh-CN, en]`，页面、`GET /api/v0/me/profile`、数据库和刷新之后都是 English。

### 2.9 恢复说明（第 7 节）

按代码核对：
- `identity/app/change_password.go:104-122`：修改密码在锁内换哈希，`RevokeSessions(ctx, actor.UserID, actor.SessionID, …)` 撤销**除当前会话以外**的全部会话，原因 `password_changed`；PAT 不动。
- `identity/app/logout.go`：退出只在令牌是会话**当前一代**、会话还活着时结束会话；别的令牌什么都不改（3.5 续期表的第一行）。
- `identity/app/refresh.go` 和 `domain.JudgeRefresh`：拿会话发过的**旧一代**令牌来续期是重复使用，会话先被撤销（3.5）。
- 访问令牌每个请求都查会话行，会话撤销在下一个请求生效（3.5）。
- `reset_password.go:69-81`：一个事务里换哈希、撤销全部会话、撤销全部 PAT。

威胁是 README 和 8.5 自己写的：XSS 读走**这个浏览器**的 `nerve.auth`，被盗的刷新令牌就是受害者正在用的会话的。第一稿的第一步"先修改密码"在这个会话里做，而修改密码保留当前会话，对方那一份跟着保留；退出也不一定结束它：对方续期之后，本页拿的是旧的一代，退出不改变什么；直到本页下一次续期撞上重复使用的检测，会话才被撤销（页面开着时最迟在它的访问令牌到期时，默认 15 分钟；页面关了就一直不会），这期间对方仍能建 PAT（预检 H2）。所以：

1. 自助，**第一步：先退出、重新登录（或在另一个浏览器登录），在新的会话里修改密码**。为什么：修改密码结束除当前会话以外的全部会话；在新会话里改，被盗的会话不再是"当前会话"，和由它换来的访问令牌在下一个请求就失效，对方不能再用会话建 PAT。重新登录是为了换出一个不是被盗的会话；退出这一步本身不保证结束被盗的会话（上面的理由），它只是让这个浏览器回到登录页。
2. 自助，**第二步：再撤销不认识的 PAT 并重新核对列表**。为什么：修改密码不撤销 PAT；PAT 能建 PAT，撤销期间可能又出现新的，列表里不再有不认识的令牌才算完成。只撤销 PAT、不先做第一步不够：对方还握着会话，随时能再建。
3. 管理员：`nerve users reset-password`，一个事务里撤销全部会话和全部 PAT。为什么：对方不断建新令牌、自助跟不上，或者用户已无法登录时，只有它一次收回全部。

README"部署"的"令牌泄露后的恢复"、M2 设计 8.5、8.7 的对应行和 17.3 的"第 7 节"一行按此改写（第二步和 `reset-password` 一段在 README 中照旧）。

### 2.10 `AllowAll` 拒绝重复的检查

同一个桶和键在一次 `AllowAll` 里出现两次：`panic("ratelimit: AllowAll with the same bucket and key twice")`，与"别的限流器的桶"同样作为编程错误。原因：它会看两次同一份余额、扣两个单位，余额可能变成负数。现有调用方（登录的两个不同桶）不受影响。M2 设计 3.10 记一句。

### 2.11 保持推迟的两项

`DateDropdown` 的键盘（M4 交接第 10 节）和时区按钮的 Tab 顺序（M3 交接第 14 节）不改：两者是共用组件（`@nerve/ui` 的 `CustomSearchSelect`、web 的 `DateDropdown`），其他调用方在 M3、M4 的页面上，M2 改了也验证不到；Codex 接受 M3、M4 作为期限；交接中已有关闭条件。M2 设计 17.3 记下这个裁定。

### 2.12 文档

| 文档 | 改动 | Task |
|---|---|---|
| M2 设计 | 第 15 节"Codex 修复"一行（进行中，spec 和 plan 的链接） | 0 |
| M2 设计 | 3.10、3.11（一种读法、不变式、代价跟着请求体走、`Content-Type`、"一次收集全部问题"的两个例外、整程序测试第 7–9 项、字段码、候选二的"解析三遍"）、6.1、7.1、7.5、7.7、8.5、8.7、11.2、§16 的"请求体被解析三遍"一行、新的 17.3 | 9 |
| README | "部署"的"迁移"一条和 River 索引的删除（Task 4）；"令牌泄露后的恢复"（Task 10） | 4、10 |
| 总体设计 | 3.5、4.3、7.7；9.4 不变（仍是"已完成"） | 10 |
| 差异清单 | 第三节"请求体"一行 | 10 |
| M4 的收尾交接 | 第 4 节：路径排序的出处（原来的行号在 T1 之后不对了）、map 型对象（`closedObject` 查内联的）、结构检查的代价和关闭条件；第 6 节：服务角色的权限和关闭条件 | 10 |
| M8 的收尾交接 | 第 2 节：请求体解析三遍；第 3 节：分用两个角色的部署核对（含 `REINDEX`）和关闭条件 | 10 |
| 收尾 spec、收尾 review | 13 个操作、约 48 行 | 10 |

控制者的 review 提交把第 15 节这一行改为"已完成"并加 review 的链接，第 14 节第一项的"七行"改为"八行"（计划的最后一节）。

### 2.13 预检的发现和落点（控制者接受全部）

| 编号 | 发现 | 落点 |
|---|---|---|
| H1 | 扫描器给每个值写出路径，内存是深度的平方；问题不限数量，每个都带着很长的路径 | 2.2 第 7 条：路径的栈、最多 16 个问题、扫描只解码名字；按表检查同样受限，按名字的顺序读成员；复制的代价是请求体乘 schema 的层数；两个新测试；M2 设计 3.11 和 §16 的一行；M4 交接第 4 节、M8 交接第 2 节；Task 1 |
| H2 | 恢复的第一步在被盗的会话里改密码，结束不了它 | 2.9；README、M2 设计 8.5、8.7、17.3；Task 9、10 |
| M1 | 不变式②说 `closedObject` 守住它，而它只查组件 | 2.2 的不变式表；`closedObject` 从 `schema()` 调用，两个反例；Task 2、9 |
| M2 | 主题"每个会话一次"的守卫没有测试 | 2.5：纯函数 `sessionTheme` 和 5 个单元测试；Task 8 |
| M3 | T10 的核对预期达不到 | 2.7；计划 T10 第 2 步 |
| M4 | "少了权限时 `/readyz` 仍是 200"说宽了 | 2.4 规则 2；README、M2 设计 6.1；Task 4、9 |
| L1 | 生成的三个文件没有写明哈希 | 计划 T1 第 2 步 |
| L2 | 完整性测试不查视图、函数 | 2.4 规则 2 的范围和测试；Task 4 |
| L3 | "只给需要的"、`ALL TABLES` 的一句不准 | 2.4；M2 设计 6.1；README |
| L4 | 队列的注释对失败说多了 | 2.8；`one-at-a-time.ts`、7.7（两处）、第 5 节的风险 |
| L5 | "每个 `await` 之后"比代码宽 | 2.3；令牌管理器的文件头、M2 设计 7.1、总体设计 4.3 |
| L6 | 三处"一次列出全部问题"没有 D3 的例外 | 总体设计 3.5、差异清单第三节、M2 设计 3.11，连同 D11 的上限 |
| L7 | M8 的部署核对缺 `REINDEX` | M8 交接第 3 节 |
| L8 | 取数不排队 | 2.8；7.7（两处） |

---

## 3. 与设计和分诊的差异（请控制者裁定）

| 编号 | 事项 | 做法 | 理由 |
|---|---|---|---|
| D1 | 字段码 | 重复用新码 `duplicate`；非 UTF-8 和落单的代理项用已有的 `invalid_format` | 重复是新的一类问题，前端要能说清"给了两次"（中英文案"Given more than once"/"重复出现"）；不合法的 Unicode 是值写错，`invalid_format` 本来就是这个意思。新码按 3.11 的规则进了每一个列出和核对它的地方 |
| D2 | 超出"非 UTF-8"的一步 | 落单的代理项转义也拒绝 | 它和非 UTF-8 的字节一样被解码器换成 U+FFFD，不拒绝它，不变式第 4 条就不成立 |
| D3 | 一次收集全部问题的例外 | 有两种读法的请求体只得到这两种问题，不再报结构问题 | 结构只对一种读法有定义；先报哪一种读法的结构都是替客户端选了一种。3.11"一次收集全部问题"对只有一种读法的请求体照旧成立 |
| D4 | 权限的完整性 | 逐个列出 + 完整性测试（2.4 第 3 条）；组角色名固定为 `nerve_runtime`；每次迁移之后重新执行文件 | 最小权限和 CI 中发现漂移；代价是运维每次迁移之后多一步（写进 README，幂等） |
| D5 | 主题的边界 | 跟随另一个标签页的登录时保持显示的主题直到资料到来；没有会话时（包括第一次打开登录页）设为"跟随系统" | 同一账户重新登录不闪；没有会话就没有账户的偏好，localStorage 中上一个会话留下的主题不再沿用 |
| D6 | A6 的主题断言 | 分诊只要求 A12 和续期被拒（A5）；另在 A6 的第一个页面版本断言两个标签页都回到默认主题 | 删掉的按钮补丁就在 A6 走的切换账户弹窗上；基点上 A6 靠补丁和 next-themes 的跨标签页同步通过，删补丁之后要有测试证明 `StoreWrapper` 接住了它（变异表 theme-no-default：A5、A6、A12 都失败） |
| D7 | A9 故事怎样证明"第二个修改还没发出" | 第二次选择之后，页面发一个请求（`GET /api/v0/instance`）并等到应答，再断言路由只见过一个修改 | 不用 `sleep`：基点上第二个修改比这个请求先到路由（基点上这一句失败，附录 A.6）；依赖的是 Chromium 按一个页面发出的先后把请求交给路由，Playwright 的路由按事件顺序处理 |
| D8 | 原型上的 `make gen-check` | 原型的改动没有提交，`gen-check` 问 git 生成物是否改过；原型上改为 `make gen` 之后逐字节比较生成物（0 处差异） | 在原型里执行 git 超出本阶段的 git 规则；真正的 `make gen-check` 在 Task 11 于提交之后的分支上执行 |
| D9 | Minor 2 的变异怎样证明 A9 | 去掉"每周第一天"的写入，两个版本都在更早的地方失败（页面的按钮、接口的应答），走不到共用断言；另用"改每周第一天时顺带把主题改回 system"证明共用断言本身咬得住 | 一条语句写入并返回，应答和数据库不会不同；共用断言多守的是"没改别的" |
| D10 | 令牌管理器的小重构 | `signOut` 和 `endSession` 共用新的 `#end(loginId)` | 两处做同一件事（重读、删除或跟随），不重复 |
| D11 | 一次收集全部问题的第二个例外（预检 H1 的裁定） | 一个请求体最多列出 16 个问题；扫描和按表检查共用这个上限，满了就停；按表检查按名字的顺序读对象的成员，同一个请求体每次得到同样的 16 个。修复波（最终评审）：每个路径最长 256 字节，更长的截短、以 `…` 结尾；同一个问题只报一次，不占上限 | 每个问题的路径可以和请求体一样长，不设上限，问题的总长是请求体的平方（2.2 第 7 条）；只设个数的上限，16 个问题仍可以各带一条约 1 MiB 的路径（回答约 100 MB），所以路径也有上限。16 个足够客户端改：正常的客户端一次写不出更多的问题，改掉之后再发会看到其余的 |

---

## 4. 验收标准（完成线）

1. `bodyshape`：24 个子测试的 `TestCheckReadsABodyOneWayOnly`（修复波之后 27 个）和 `TestADuplicateSaysWhy` 通过；`TestCheckListsAtMostSixteenProblems` 和 `TestCheckCostsAboutTheBody` 通过，第一稿的写法（eager-paths）、去掉上限（no-cap）、按 map 的顺序读成员（walk-map-order）各自让它们失败；修复波的 `TestCheckCutsALongPathShort` 和整程序测试 `TestTheAnswerToABrokenBodyStaysSmall` 通过，路径不截短（path-unclipped）、重复的问题照报（no-dedupe）各自让测试失败（附录 B）；第四个整程序测试中每个带请求体的操作都有"`<第一个字段>` twice"和"not UTF-8 in `<第一个字符串字段>`"，有嵌套对象的另有嵌套的一项（M2：17 项），全部 400 且字段和码正确；`closedObject` 的两个内联反例通过，只查组件时失败。
2. A10 的接口版本：Codex 的第一个请求 400 `not_allowed`，第二个 400 `[{onboarding_step, duplicate}]`，引导步骤不变。
3. 令牌管理器：2.3 的两个测试通过；两个重读各自去掉时，对应的测试失败。
4. `runtime_role_test.go` 的两个测试通过；缺 River 的表、缺序列、缺 `MAINTAIN`、多给权限、新迁移加表而文件没跟上、新迁移加视图、物化视图或收回了 PUBLIC 执行权的函数，各自让测试失败。
5. `session-theme.test.ts` 的 5 个测试和 A5、A6、A12 的页面版本（回到默认主题）通过；没有会话时不设默认主题，单元测试和三个页面版本都失败；去掉"每个会话一次"，单元测试失败。
6. A8、A9、A10 的页面版本和接口版本调用 2.6 的断言；文件内的 `profileOf` 删除。
7. 两个 store 的 6 个单元测试和 A9 的新故事通过；队列不等前一个、失败卡住队列，各自让测试失败；Codex 的实验在基点上复现（页面 English、nerve 简体中文），修复之后页面、接口、数据库和刷新都是 English。
8. `AllowAll` 的新测试通过，去掉检查时失败。
9. 文档按 2.12 改完；`grep -rn -E '&lt;|&gt;|&amp;'` 改过的文件没有输出；关键词守卫没有命中。
10. 七个门禁通过（附录 C）；计划的块应用到 `37b7e9c` 的干净副本得到原型的树（附录 D）。

---

## 5. 风险

| 风险 | 应对 |
|---|---|
| 客户端真的发了重复的键（例如手拼 JSON 的脚本），以前能用，现在 400 | 这正是要拒绝的：它的含义不确定（I-JSON）。400 的 `errors[]` 指出字段和 `duplicate`，改法明确。M2 的前端用 `JSON.stringify`，不会产生重复 |
| 扫描器与 `encoding/json` 对转义的理解不同 | 名字用 `json.Unmarshal` 解码，与解码器同一实现；代理项配对按 RFC 8259 的 UTF-16 规则，单元测试列出成对、落单、结尾、两个高位、转义的反斜杠 |
| 每个请求体多扫描一遍；恶意的请求体让检查的内存成平方，或者让 400 的回答随请求体变大 | 请求体上限 1 MiB；路径只在报问题时写出，最多报 16 个问题，满了就停，扫描只解码名字；报出的路径最长 256 字节，同一个问题只报一次（2.2 第 7 条）。第一稿在 1 MiB 的请求体上要 4.4 GB（深的）和 17 GB（长名字下的大量问题），公开的登录接口前面只有匿名的桶，一个请求就能耗尽容器的内存；这一稿当初各 3.1 MB、5.7 MB，修复波之后各 2.4 MB、0.9 MB。最终评审的三个约 1 MiB 的请求体在路径截短之前得到约 100、100、86 MB 的回答，`Check` 分配 18–99 MB；修复波之后回答各约 1.7 KB，`Check` 分配 1.4–2.0 MiB。`TestCheckCostsAboutTheBody` 和 `TestTheAnswerToABrokenBodyStaysSmall` 守住；按表检查复制的代价是请求体乘 schema 的层数，递归的 schema 到来时改为一遍读完（M4 交接第 4 节）；M8 实测常见请求体的耗时（M2 设计 §16"请求体被解析三遍"一行） |
| 超过 16 个问题的请求体只得到前 16 个；超过 256 字节的路径只报开头 | 正常的客户端写不出这么多问题，契约里真实的路径最长几十个字节；改掉之后再发会看到其余的；同一个请求体每次得到同样的 16 个（D11）：按固定的顺序读时遇到的前 16 个，不一定是路径最小的 16 个 |
| 运维忘了在迁移之后执行权限文件 | README 与"先 `migrate up`"写在一起；少了权限时日志有 `permission denied`；M8 的部署核对要求检查（交接第 3 节） |
| M3 的第一个迁移就会碰到完整性测试 | 这是设计：失败信息写明是哪张表、要什么权限、在哪个文件；M4 交接另有一条 |
| 修改串行之后，连续操作要等前一个应答 | 资料和账户的修改是人点出来的，一次一个；失败也放行。一个一直不结束的请求挡住这个 store 的队列：网页的客户端没有请求的期限，nerve 在 `server.request_timeout` 之内应答它收到的每个请求，挡住的只剩到不了 nerve 的请求，直到浏览器放弃连接或页面重新加载。没有应答的失败不保证顺序（2.8）。队列只在一个 store 之内 |
| A9 故事依赖 Chromium 的请求顺序 | 见 D7；Playwright 的路由按事件顺序处理，同一个页面的请求按发出的先后到达。若将来不稳，改为在页面里记录 `fetch` 的调用 |
| 主题：`status` 在"会话暂不可用"时不是 signed-out，页面保持显示的主题 | 这时会话还在（记录保留），不应换成默认；恢复之后按资料 |

---

## 附录 A：先红后绿（原型上复现 Codex 的每一条，再修）

"基点"指把原型中这一项的产品文件换回 `37b7e9c` 的版本、保留新的测试（`$M2TMP/codexfix/swap.sh`），跑完换回。日志在 `$M2TMP/codexfix/logs/`。

### A.1 Critical 1

- `bodyshape` 单元测试，基点的 `bodyshape.go`（扫描器换成空文件）：`TestCheckReadsABodyOneWayOnly` 的 24 个子测试中 18 个失败（其余 6 个是本来就该通过的对照：大小写不同、两个对象里同名、成对的代理项、转义的反斜杠、U+FFFD 本身等），`TestADuplicateSaysWhy` 失败。修复之后全部通过。
- 整程序测试，基点的 `bodyshape`（`red-contract.log`）：17 个新子测试全部失败（8 个操作的"twice"和"not UTF-8"，加上 `onboarding_step.profile_complete twice`），41 个原有的通过。修复之后全部通过。
- A10 的接口版本，基点的 `bodyshape` 编进 nerve（`red-e2e-dup.log`）：Codex 的第二个请求得到 **200**（`code` 为空，即返回了资料），测试在状态上失败。修复之后 400 `[{onboarding_step, duplicate}]`，步骤不变。
- 代价（预检 H1）：第一稿的 `ambiguity.go` 和 `bodyshape.go`（路径逐层写出、问题不限数量）跑两个新测试（`mut-eager-paths.log`）：代价的测试四个请求体都超过 64 MiB（844、150、564、1130 MiB），上限的测试两个子测试都失败（20 个问题全报）。这一稿各 1.5、1.0、2.5、5.8 MiB。预检的 1 MiB 请求体经 identity 的表（`$M2TMP/codexfix/probes/zz_measure_test.go.txt`，`h1-measure-after.log`）：9,999 层、90 字符的名字，949,906 字节，14 ms、3.1 MB（第一稿 320 ms、4.4 GB）；10,000 层数组，20,000 字节，1.0 MB（第一稿 150 MB）；4,000 层、100 字符的名字再加 140,000 项的数组，700,001 字节，10 ms、1.8 MB（第一稿分配 55 GB、1.2 秒）；300 KB 的名字下 60,000 个不是 UTF-8 的字符串，540,006 字节，3 ms、5.7 MB（第一稿 17 GB）。第一稿的数字是预检量的。

### A.2 Important 1

基点的 `token-manager.ts`（`red-tm.log`）：`Tests 2 failed | 14 passed (16)`，失败的正是两个新测试：①"keeps the sign-in another tab made while a sign-out's logout was out and the lease had run out"（基点删掉了 Y 的记录）；②"follows another tab's sign-in that came in before navigator.locks handed a failed first refresh back"（基点把标签页设成 X 的"暂不可用"）。修复之后 16 个全部通过。

### A.3 Important 2

用只改权限文件的三个版本跑两个测试（`$M2TMP/codexfix/red/grants/`）：

| 权限文件 | 结果 |
|---|---|
| README 原来的写法（业务表读写 + `goose_db_version` 只读） | `/readyz` 200，但 River 起不来：`permission denied for table river_queue`（42501），"jobs did not start; trying again" 一再重试，会话清理 15 秒内没有运行；完整性测试列出 River 的 4 张表和 2 个序列（`red-grants-readme.log`） |
| 去掉 `MAINTAIN` | 7 个 `REINDEX INDEX CONCURRENTLY` 全部 `permission denied for index …`（42501）；完整性测试：`river_job` 少 `MAINTAIN`（`red-grants-nomaintain.log`） |
| 去掉序列 | River 的定时任务投递失败：`permission denied for sequence river_job_id_seq`；完整性测试列出两个序列（`red-grants-noseq.log`） |

完整的文件：两个测试通过（约 3 秒）。三个版本在修订之后的测试（`TestTheGrantsFileCoversEveryRelationAndFunction`）上重跑，结果同上。

### A.4 Minor 1

基点的 `store-wrapper.tsx`、`store-context.tsx`、`switch-account-modal.tsx` 编进 nerve（`red-e2e-theme.log`）：A5 和 A12 的页面版本失败，`data-theme` 期望 `light`、实际 `dark`（Codex 看到的现象，停用和续期被拒都是）；A6 在基点上通过（切换账户弹窗的补丁设了标签页 A，next-themes 经 localStorage 同步了标签页 B），所以 A6 的断言由变异证明（附录 B）。修复之后三者通过。`session-theme.ts` 在基点上没有，它的 5 个单元测试由变异证明（附录 B 的 theme-no-default、theme-every-profile）。

### A.5 Minor 2、Minor 3

重构和文档，没有产品行为可复现；Minor 2 由附录 B 的五个后端变异证明。

### A.6 PATCH 的顺序

- 两个 store 的单元测试，基点的 store（`red-stores.log`）：`Tests 6 failed (6)`。修复之后 6 个通过。
- A9 的新故事，基点的 store 编进 nerve（`red-e2e-serial.log`）：`sent` 期望 `["zh-CN"]`，实际 `["zh-CN", "en"]`：第二个修改在第一个被扣住时就发出了。
- Codex 的实验原样（`$M2TMP/codexfix/probes/a9-codex-experiment.spec.ts`：同一个故事，去掉顺序的断言，等两个应答都到）：基点上 `sent zh-CN,en; passed on en,zh-CN`，页面是 English（"Language & Time"、`lang=en`），数据库是 `zh-CN`，测试在 `expectPreferences` 失败，与 Codex 报告第 6 节的结果相同（`red-e2e-codex-base.log`）。修复之后 `passed on zh-CN,en`，通过（`green-e2e-codex.log`）。

### A.7 恢复说明、`AllowAll`

- 恢复说明：按代码核对（2.9），没有可运行的复现。
- `AllowAll`，基点的 `ratelimit.go`（`red-ratelimit.log`）：`TestAllowAllRejectsTheSameCheckTwice` 失败（"did not panic"）。

---

## 附录 B：变异表

每个变异只改一个文件的一处（`$M2TMP/codexfix/mutants/make.py` 写出，`run.py` 跑，日志 `logs/mut-<名字>.log`），跑完换回；eager-paths 换回第一稿的两个文件，grants-new-table、grants-new-view 多放一个迁移。修订之后，改过的文件上的变异全部重写、重跑（`bodyshape`、`rules_test.go`、`session-theme.ts`、权限的测试），其余的也重跑了单元测试的那些（令牌管理器、队列、`AllowAll`）。最终评审之后的修复波在分支上做变异：path-unclipped、no-dedupe 在修复提交之后改进 worktree（`$M2TMP/fixwave/mutate.py`），跑完 `git restore`；no-cap、no-duplicate-check、no-utf8-check、no-surrogate-check 用 `go test -overlay` 在修复波的测试上重跑，不改 worktree；日志在 `$M2TMP/fixwave/`。

| 变异 | 改了什么 | 失败的测试 |
|---|---|---|
| no-ambiguity-call | `Check` 不调用扫描器 | 单元 18 个子测试（与基点相同）；整程序 17 个；A10 的接口版本（Codex 的请求） |
| no-duplicate-check | 不报重复 | 单元 10 个（顶层、对象、嵌套、三次、转义、两种转义、数组项、开放 map、未声明的位置、组合）；整程序 9 个"twice"。修复波重跑单元测试：另有 1 个（名字两次、值都不是 UTF-8），上限的测试 2 个（twenty names twice、一个问题重复 20 次） |
| names-as-written | 按写法而不是解码后比较名字 | 单元 2 个：转义等价、两种转义 |
| no-utf8-check | 不查 UTF-8 | 单元 5 个（值、嵌套值、数组项、名字、组合）；整程序 8 个"not UTF-8"。修复波重跑单元测试：另有 2 个（最终评审的两个重复报出的例子），上限的测试 1 个（一个问题重复 20 次） |
| no-surrogate-check | 不查代理项配对 | 单元 4 个（落单高位、结尾高位、落单低位、两个高位）；修复波重跑：5 个，另有高位后面跟着不是代理项的转义 |
| no-array-walk | 数组里的问题被丢掉 | 单元 2 个（数组项里的重复、数组项里的非 UTF-8） |
| eager-paths | 换回第一稿的 `ambiguity.go` 和 `bodyshape.go`：每个值写出路径，问题不限数量（预检 H1） | 代价的测试：四个请求体都超过 64 MiB（844、150、564、1130 MiB）；上限的测试 2 个 |
| no-cap | `full()` 总是 false：不设上限 | 上限的测试 2 个；代价的测试 2 个（长名字下的字符串 1188 MiB，map 的长键下的成员 1191 MiB）。修复波重跑：上限的测试 3 个；代价的测试不再失败：路径截短之后，长名字下的问题都相同，只报一次，上限守的只剩问题的个数 |
| walk-map-order | 按表检查按 map 的顺序读对象的成员 | 上限的测试的"twenty undeclared names"（报出的 16 个每次不同） |
| path-unclipped | 修复波：`at()` 写出整条路径（`enter` 不截名字，`clip` 什么都不做，`at` 原样返回） | 截短的测试 5 个（257 字节、嵌套的、`é`、`中` 的两种）；代价的测试：最终评审的三个请求体分配 19,487、19,481、20,795 KiB，超过请求体的 6 倍（6,143、6,142、5,562 KiB），回答的路径合计 16,770,880、16,770,880、14,545,488 字节，超过 4,096；长名字下的两个旧请求体和重复 200,000 次的一个，路径合计 1,048,630、1,048,672、4,854 字节；整程序测试 `TestTheAnswerToABrokenBodyStaysSmall` 3 个：回答 100,626,389、100,626,341、86,474,917 字节，最长的路径 1,048,180、1,048,180、909,093 字节。修复之前的代码（`4fd4ce98` 的 `bodyshape.go`）同样让这些测试失败，三个请求体的 `Check` 分配 18,468、18,468、98,938 KiB |
| no-dedupe | 修复波：`report()` 不跳过路径和码都已报出的问题 | 单元 2 个（最终评审的两个重复报出的例子）；上限的测试 1 个（一个问题重复 20 次，占满 16 个）。代价的测试和整程序测试通过：截短之后 16 个问题的路径合计 4,096 字节，回答 25,765、25,717、25,605 字节，仍在 32 KiB 之内 |
| closed-object-components-only | `closedObject` 只查组件（第一稿的写法，预检 M1） | `TestAuthoringRulesReportViolations` 的两个内联反例 |
| closed-object-twice | 组件由 `components()` 和 `schema()` 各查一次 | 同一测试的三个组件用例（报了两次） |
| signout-no-recheck | 退出回来后直接删记录（基点的写法） | 退出的那个测试 |
| first-refresh-no-recheck | 第一次续期失败后不重读 | 启动的那个测试 |
| grants: README 的写法 / 去掉 `MAINTAIN` / 去掉序列 | 附录 A.3 | 两个测试（见 A.3） |
| grants-new-table | 多一个迁移，建 `probe_things`（带序列），文件不改 | 完整性测试：`probe_things`、`probe_things_id_seq` |
| grants-new-view | 多一个迁移，建一个视图、一个物化视图和一个收回了 PUBLIC 执行权的函数，文件不改（预检 L2） | 完整性测试：`probe_counts`、`probe_emails` 要 `[SELECT]`，`probe_one()` 要 `[EXECUTE]` |
| grants-goose-insert | `goose_db_version` 多给 `INSERT` | 完整性测试：`goose_db_version has [SELECT INSERT], want [SELECT]` |
| theme-no-default | `sessionTheme` 没有会话时不设默认 | 单元 1 个（没有会话）；A5、A6、A12 的页面版本 |
| theme-every-profile | 去掉"每个会话一次"的守卫（预检 M2） | 单元 1 个（每个会话一次）。e2e 发现不了：A5–A10、A12 的 24 个全过（预检同样），所以守卫由单元测试守住 |
| a8-display-name-kept | `PATCH /me` 不写显示名（生成的查询） | A8 的三个版本：两个页面版本在 `expectAccountChanged`；接口版本更早，在应答的检查 `expect(data).toMatchObject(change)`（`e2e/stories/identity/a8-update-me.spec.ts:132`），同 D9 说的 A9 |
| a8-timezone-reset | `PATCH /me` 不带时区时把时区改成 `Europe/Berlin` | A8 的两个页面版本（第二个在 `expectAccountChanged`：只改名字，时区却变了；原来的"包含"断言发现不了） |
| a10-first-name-kept | `PATCH /me` 不写名字 | A10 的三个版本在 `expectProfileStepTaken`；A8 的三个版本：两个页面版本在 `expectAccountChanged`，接口版本在应答的检查（`a8-update-me.spec.ts:132`） |
| a10-steps-from-default | 步骤合并进默认值而不是已存的值 | A10 的接口版本在 `expectProfileStepTaken`（它的 `stepsBefore` 有 `workspace_join: true`） |
| a9-week-kept | `PATCH /me/profile` 不写每周第一天 | A9 的页面版本（按钮）和接口版本（应答），都在共用断言之前（D9） |
| a9-week-resets-theme | 改每周第一天时顺带把主题改回 `system` | A9 的页面版本，在 `expectPreferences` |
| queue-none | 队列不等前一个 | 两个 store 的 6 个单元测试；A9 的新故事 |
| queue-stuck-on-failure | 失败的修改卡住队列 | 两个 store 各 2 个（被拒、没有应答） |
| allowall-twice | 不查重复的检查 | `TestAllowAllRejectsTheSameCheckTwice` |

---

## 附录 C：门禁（原型）

`$M2TMP/codexfix/gates.sh`，在原型上（全部 Task 之后；预检之后的修订完成后重跑，结果相同）：

| 门禁 | 结果 |
|---|---|
| `make gen-check`（原型上是 `make gen` 之后逐字节比较生成物，D8） | 0 处差异 |
| `make lint-go` | 两段 `0 issues.` |
| `make test` | 32 个包 `ok`，没有 `FAIL` |
| `make lint-web` | `keywords: 60 rules, 3 exceptions, no hits.`；`Tasks: 54 successful, 54 total` |
| `make knip` | 退出码 0，没有问题 |
| `make test-web` | `Tasks: 16 successful, 16 total` |
| `make e2e` | `50 passed`（收尾时 48，加 A10、A9 各一个） |

原型的第一轮 `lint-go` 发现 `apitest/operations.go` 文档注释里两位数的列表项不合 gofmt、`lint-web` 发现 A6 的一个 `await` 在循环里（oxlint 上限 0），都已改在原型里，计划的块是改后的样子。跑完之后没有这次运行建的容器，没有留下后台进程。

## 附录 D：计划的树就是原型的树

`node $M2TMP/codexfix/planapply.mjs docs/v0/M2-auth/plans/codex-fixes.md check $M2TMP/codexfix/base`（`base/` 是 `37b7e9c` 的归档解开的树）：144 处替换、11 个整文件（8 个新文件、3 个改写的文件），全部依次恰好一处，退出码 0。在 `37b7e9c` 的另一份干净副本（`$M2TMP/codexfix/replay`，修订之后从归档重新解开）上按计划执行 Task 0–10（Task 1 之后 `make gen`），加上本 spec 和计划，`node $M2TMP/codexfix/treediff.mjs` 与原型比较：`2664 and 2664 files; 0 differences`。回放的树上再跑门禁：生成物 0 处差异、`lint-go` 两段 `0 issues.`、`make test` 32 个包 `ok`、前端 54 个检查任务、`knip`、`test-web` 16 个任务都通过；`e2e` 49 passed，S3 失败只因为回放的副本不是 git 仓库（编进的提交是 `unknown`），原型上 50 passed。输出全文在计划的"块的核对"。
