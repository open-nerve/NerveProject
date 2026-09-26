# M2/P2 登录与会话：评审记录

| 项 | 内容 |
|---|---|
| Phase | M2/P2 `sessions` |
| 日期 | 2026-09-26 |
| 结论 | **通过**（整分支评审提出的问题已在一轮修复中处理完；一项设计缺口登记为已知限制，交负责人决定是否缓解） |
| spec / plan | [spec](../specs/P2-sessions.md) / [plan](../plans/P2-sessions.md) |
| 分支 | `worktree-m2-p2-sessions`，从 `main` 的 `fd69736`（P1 的合并）分出 |

## 1. 评审方式

- **spec 和 plan 由架构子任务产出**（`23fa061`）：
  - 先在仓库外把整个 P2 做出来（原型），测试、lint、端到端全部通过。
  - plan 中的代码由脚本从原型文件原样拼入：新文件给完整内容，改动小的文件给差异，48 个差异块都用 `patch -p1` 核对过。
  - 再从 `fd69736` 的副本出发，按 plan 逐个 Task 重放，结果与原型逐文件相同。12 个变异都被测试发现（spec 附录 A）。
- **执行前的核对**：控制者逐对核对共用文件和接口的 Task（13 行），没有发现冲突。spec 第 3 节的 19 项差异全部接受（第 3 节）。
- **逐个 Task 评审**：12 个 Task 各由一个实现子任务照 plan 写入代码，随后由独立的评审子任务检查"是否符合 spec"和"代码质量"。
  - 实现者按要求对自己的代码做变异核对。有 9 个 Task 发现 plan 自带的测试没有咬住它声称守护的性质，或者注释、文档写了代码没有的行为。控制者逐项裁定修正，修正在评审之前提交（第 3 节）。
  - Task 1 在评审之后修了一轮，修复的提交再做范围评审。
  - 下一个 Task 的实现与上一个 Task 的评审并行，同一时间只有一个实现者提交。
- **整分支评审**（`23fa061..668017b`）：结论为"修复后可合并"，有 0 个 Critical、2 个 Important、8 个 Minor，没有发现可利用的缺陷。评审另用 4 个变异核对了测试，复核了执行中的每一项裁定（都成立，其中 Task 4 带来的两处只做到一半，见 M1），并逐条分拣了逐个 Task 评审留下的 Minor。修复在一轮中完成（`668017b..0d140fe`），之后对修复的提交做了一次范围评审。
- **持续集成**：推送后通过 GitHub 公开接口确认结果，见第 2 节。

## 2. 验收标准核对（spec 第 4 节）

| # | 标准 | 结果 |
|---|---|---|
| 1 | A3–A6、A15 的接口版本通过，P1 的故事（S1–S4、A1、A2）仍然通过 | 通过：本机 `make e2e` 12 个全部通过，`stories/identity --repeat-each=5` 35 个全部通过（Task 11）；持续集成 e2e 任务通过（`668017b` 的 run 36204986788；修复轮 `0d140fe` 的 run 36207609533） |
| 2 | 登录的耗时测试通过 | 通过：本机中位数 14.1–14.8 毫秒，两组相差 1.7–3.6%（容差 25%）；持续集成的 ubuntu 机器上通过（Task 10，run 36204986788）。去掉假哈希后，未知地址只要约 0.4 毫秒，测试失败 |
| 3 | 续期的四个测试通过 | 通过：`TestRefreshRejectsWithoutRevoking`（伪造的旧代；访问令牌里的会话 id 加 `g = 0`）、`TestRefreshDetectsReuse`、`TestRefreshReuseRevokesTheSession`、`TestRefreshAfterTheSigningKeyChanged`、A5 |
| 4 | 3.10 表中除 `password_user` 外的每个桶都有测试；续期、退出只经过 `anonymous`；IPv6 前缀 | 通过：`TestRateLimitPicksTheBucketAndKey`、`TestRateLimitedRequestIs429`、失败闸门的四个测试、`TestLoginLimitsByIPAndByIPWithAddress`、`TestRegisterLimitsByIP`、`TestModuleLimitsCountByTheIPKey`（裁定加入）、`TestRefreshAndLogoutTakeNoUnitOfTheModule`、`TestIPKey`、`TestFailureGateCountsByTheIPKey`、A15 |
| 5 | 失败闸门：超额后不调用认证器；50 个并发不超额；过期、成功、故障都退回 | 通过：`TestFailureGateTurnsAwayWithoutAuthenticating`、`TestFailureGateHoldsUnderConcurrency`（认证器 3 次，3 个 401、47 个 429）、`TestFailureGateKeepsTheUnitOfAFailedCredentialOnly` |
| 6 | `platform/ratelimit` 覆盖突发、`AllowAll`、`Reserve` 的退回和闲置键的清理 | 通过：9 个测试，`-race`；"退回只算一次"由裁定加强后才真正咬住（第 3 节） |
| 7 | 每个响应都带三个安全响应头 | 通过：`httpserver` 和 `bootstrap` 的 `TestSecurityHeadersOnEveryResponse`、`TestPanicDiscardsHeadersSetBeforeIt` |
| 8 | `make lint`、`make test`、`make gen-check`、`make knip`、`make e2e` 通过 | 通过：每个 Task 提交前 `make lint-go` 两段 `0 issues.`、`make test` 通过；有生成物的 Task（5、8、9）生成物的 SHA-256 与 plan 相同，提交后 `make gen-check` 干净；Task 5、9、11 之后 `make lint-web`、`make e2e`，Task 11 之后 `make knip`；持续集成三个任务都通过 |
| 9 | 3.20 中 P2 的各行、8.7 中 P2 的 README 内容；交接按第 7 节处理 | 通过（Task 12 及其修正）：评审逐行核对了 3.20 的 7 行 P2 和 8.7 的 P2 一行，每一句都对照 HEAD 的代码 |

实测：
- **argon2 与登录**：默认参数（m = 19456 KiB、t = 2），本机每次登录的校验约 14–15 毫秒。
- **持续集成耗时**（`668017b`）：server 61 秒（测试 22 秒，含登录耗时测试），web 90 秒，e2e 127 秒（端到端 18 秒）。

## 3. 执行中的决定

spec 第 3 节的 19 项差异全部接受：都是设计留给 P2 的细节，没有一项跨模块或改动平台与 `shared` 的边界。

执行中的裁定。前 9 项都是"plan 自带的测试或文字不成立"，由实现者的变异核对发现，在评审之前修正：

| 决定 | 原因 |
|---|---|
| Task 2：`TestReserveAndRefund` 先把桶取空，再退回两次 | 原来的测试在去掉 `sync.Once` 后仍然通过，"退回只算一次"没有被守住 |
| Task 3：`httpserver` 的安全响应头测试读 `rec.Result().Header`；注释改为 CSP 在 P4 由 `webui` 加入 | 读活的头部映射时，写出之后再设头部也能通过；注释声称 `webui` 已经设了 CSP |
| Task 4 → 5：不可信转发只警告一次的测试先发一个不带 `X-Forwarded-For` 的请求；测试的 API 配置信任 `fd00::/8`，请求元信息的测试从可信代理后面来 | 原来的测试守不住"没有转发头时不警告"和"可信代理配置真的传到了客户端 IP"；Task 4 的测试文件是过渡版本，Task 5 的差异块基于它，所以在 Task 5 的最终版本上加 |
| Task 6：刷新令牌解析的反例加上"末尾换行"，"末尾空格"保留前缀 | 原来的反例在去掉长度检查或先 `TrimSpace` 后仍然全部通过 |
| Task 7：测试替身记下并断言写入的时间；argon2 格式的反例加上严格 base64、开头的空段、`m` 的位宽（16 → 19） | 审计时间来自用例的时钟是 P1 起的约束，原来有 3 个变异把零时间传给写入仍然通过 |
| Task 8：账户行锁的测试在第一把锁出错时立即失败；改掉遮蔽外层变量的局部名 | 原来会一直挂到 `go test` 超时 |
| Task 9：加上 `TestModuleLimitsCountByTheIPKey` | 原来 `login_ip_email` 的键不带 IP（任何人都能锁住别人的邮箱），或按完整地址而不是 /64 计数，测试都通过 |
| Task 10：加上 `TestLoginThroughATrustedProxyRecordsTheClient`（从 Task 4 带来），抽出 `openPool` | 整程序的接线中可信代理配置没有被测试守住 |
| Task 12：CSP 在 M2/P4 加入；M0-P5 的引文照交接原文；M0-P2 的处理结果准确写出闸门的规则 | 文档写了代码没有的行为，或者引文不是原文 |
| Task 1（评审之后）：`numberHook` 用 float64 能精确表示的界比较；`rejectNulls` 深入列表查找空值 | 2⁶³ 这样的浮点数原来会通过 64 位整数键的检查并被截成最大值；`trusted_proxies` 中的空项原来会成为零值前缀 |
| 登录耗时测试是持续集成的已知风险 | 失败时先把两组中位数写进 review，不放宽容差。实际在持续集成上通过 |

## 4. 整分支评审的发现与处理

| 发现 | 级别 | 处理 |
|---|---|---|
| I1 "日志里没有秘密"的测试只核对原始字节或十六进制，slog 的 JSON 把 `[]byte` 写成 base64：往登录、重复使用的日志里加 `slog.Any("token_hash", …)` 的变异，测试照样通过 | Important | 已修（F1，`a4541b3`）：一个测试辅助函数按原文、十六进制和两种不补位的 base64（补位的形式只多了末尾的 `=`，已包含在内）查找秘密，登录、续期、注册的日志测试都用它；两个变异现在都失败 |
| I2 调高 argon2 参数之后，还没有重新登录过的账户按旧参数校验，未知地址按新参数校验，登录耗时能区分两者，直到这些账户各登录一次 | Important（设计的缺口） | 登记为已知限制（F2，`9ea1eb8`）：M2 设计 §16 加一行，3.9 和差异清单写明"存储的哈希都用当前参数时"，`config.yaml` 在 argon2 的配置项旁注明。对照：Django 默认的 PBKDF2 哈希器在密码错误时补跑缺少的迭代次数，没有这个缺口；它的 argon2 哈希器有；Plane 对未知地址直接答"用户不存在"，本来就没有耗时相同这一性质。是否在代码上缓解由负责人决定（第 6、7 节） |
| M1 `NewAPI` 接受 `IPv6PrefixLen` 为 0（所有 IPv6 客户端共用一个桶），接线也没有测试守住 | Minor | 已修（F3，`b75c17a`）：`NewAPI` 拒绝 1–128 之外的前缀长度，与缺少依赖走同一个错误；客户端 IP 的测试改用 /48。`bootstrap` 丢掉前缀长度时 20 个整程序测试失败 |
| M2 剩余的份额不到 1 纳秒时，等待截成 0，429 不带 `Retry-After` | Minor | 已修（F4，`502ce73`）：拒绝时的等待至少 1 纳秒 |
| M3 `refreshTokens` 的说明写"这个会话以前发过的令牌也会结束会话"，换签名密钥之后旧代的标签无法验证，只答 401、不撤销 | Minor | 已修（F5，`760f0f3`）：说明限定为"当前签名密钥下"，重新生成打包文件和 TS 类型 |
| M4 spec 第 3 节第 3 条依赖的 README 说明并不存在：写 `ip:端口` 的可信代理会让所有客户端悄悄落到代理的地址上 | Minor | 已修（F7，`a666558`）：README 写明代理要写纯地址；可信代理转发了解析不了的一项时，每个进程记一次 WARN，只记 `peer` |
| M5 `server.trusted_proxies` 接受 `0.0.0.0/0`、`::/0`，任何客户端都能自己选 IP | Minor | 已修（F8，`a666558`）：启动校验拒绝长度为 0 的前缀；README 写明只信任代理自己的地址 |
| M6 没有测试核对每个模块的 `Problem` 响应都声明了两个响应头 | Minor | 已修（F9，`0d140fe`）：整程序测试核对 `api/dist` 中每个操作的 `default` 响应都声明了两个响应头 |
| M7 `bootstrap` 把每个桶接到对应配置，没有测试守住 | Minor | 移交 P3：P3 加入 `password_user`，接线相同（第 6 节） |
| M8 spec 列出的测试与裁定之后的实际不一致 | Minor | 记在第 5 节 |

逐个 Task 评审留下的 Minor，整分支评审逐条分拣后：
- **本轮修掉**：限流等待截成 0（F4）；`IPv6PrefixLen` 的接线（F3）；argon2 密钥的严格 base64 没有咬住的反例（F6）；登录日志测试只核对原始字节（F1）。
- **移交**：桶与配置的接线（P3）。
- **接受**：`AllowAll` 同一个桶和键出现两次时余额成为 -1（没有调用方这样用）；404 的 `nosniff` 断言多余；`isTrusted` 中多余的 `IsValid()`；429 没有断言不带 `WWW-Authenticate`（结构上保证）；解析刷新令牌没有模糊测试（按检查的先后不会 panic）；续期每一轮都先签访问令牌（没命中很少，签名便宜）；测试替身把错 id 的写入记在"事务之外"名下（只是命名）；并发删除账户之后加锁是 500（v0 没有删除账户）；"没命中"的测试只比较 `updated_at`（每条 `UPDATE` 都写它）；`RequestMetaFrom` 每个请求取两三次；并发测试的 goroutine 中可能调用 `t.Fatal`（`wg.Wait` 保证安全）；可信代理测试读最新的会话而不按账户过滤（每个测试一个数据库）；A3 的并行请求共用调高的桶；总体设计 6.4 流程图的注释列没有重新对齐。

## 5. 最终实现与 plan 的差异

plan 是执行记录，不回改。以 spec 和代码为准。

1. **第 3 节的测试和文字修正**：Task 2、3、5（带来 Task 4 的两处）、6、7、8、9、10、12，都只改测试或文字，不改生产代码的行为。其中改动的测试文件都不再被后面的 Task 的差异块修改，plan 的差异块全部按原样应用。spec 列出的测试因此与实际不同（评审 M8）：
   - `TestParseRefreshTokenRejects` 是 14 个反例（spec 2.8 写 13 个）；
   - `TestVerifyRejectsAnotherFormat` 是 20 个（spec 2.9 写 16 个）；
   - spec 没有列出的新测试：`TestModuleLimitsCountByTheIPKey`（Task 9）、`TestLoginThroughATrustedProxyRecordsTheClient`（Task 10），以及修复轮加的测试（第 3 项）。
2. **Task 1 的修复轮**（`760788f`）：`numberHook` 的界、`rejectNulls` 深入列表，这两处改了生产代码。
3. **修复轮**（`668017b..0d140fe`）：
   - F1：`app` 测试的 `assertNoSecret`，登录、续期、注册的日志测试都用它。
   - F2：M2 设计 3.9、§16，差异清单，`config.yaml` 的注释（只改文档）。
   - F3：`NewAPI` 校验 `IPv6PrefixLen`；测试配置用 /48。
   - F4：限流器拒绝时的等待至少 1 纳秒。
   - F5：`refreshTokens` 的说明；重新生成 `api/dist/openapi.yaml` 和 `schema.gen.ts`。
   - F6：argon2 密钥的严格 base64 反例（`TestVerifyRejectsAnotherFormat` 共 20 个）。
   - F7、F8：可信代理转发畸形项时的一次性 WARN；拒绝 `/0`；README 两句。
   - F9：`bootstrap` 的契约测试核对两个响应头的声明。
4. **生成物**：Task 5、8、9 的生成物与 plan 记录的 SHA-256 相同，修正没有改动任何生成来源。
5. **父文档同步**：总体设计 3.5、3.6、4.1、4.2、6.4；M0 设计 3.3；差异清单四；README 的部署一节；三份交接的处理结果（Task 12）。修复轮另改了 M2 设计 3.9、§16 和差异清单的一行（F2），以及 README 的两句（F7、F8）。M2 设计的进度表在本评审的提交中改为已完成。

## 6. 移交事项

| 交接 | 交给 | 内容 |
|---|---|---|
| [M0-P2-platform-notes](../handoffs/M0-P2-platform-notes.md) | M2/P3，M8 | P2 处理了第 2 条中的限流；留下接口调用日志（M8）、River 与停机（P3） |
| [M0-P5-frontend-api-notes](../handoffs/M0-P5-frontend-api-notes.md) | M2/P4 | P2 处理了安全响应头；留下 CSP 和前端改调新接口（P4，这一项在 P4 正式关闭） |
| [M0-P6-e2e-notes](../handoffs/M0-P6-e2e-notes.md) | M2/P3、P4，M4、M5、M8 | P2 处理了认证 fixture 的登录和续期；留下 PAT、页面的登录状态、S2、S3、River 停机、fixture 写法的延伸 |

P1 评审第 6 节交给 P2 的五项都已处理（spec 第 7 节）。

整分支评审留给后续的事项：

| 交给 | 事项 |
|---|---|
| P3 | `bootstrap` 把每个限流桶接到对应的配置，没有测试守住（评审 M7）。P3 加入 `password_user` 时，用每个桶各不相同的突发值，逐个打到上限，证明接线 |
| P3 | 任何"不可用的密码"形式（例如以后的 `users create` 不设密码）都要保持登录的耗时：`errNotOurHash` 现在是很快的 500，v0 中不存在非 PHC 的哈希 |
| 负责人 | 登录耗时在调高 argon2 参数之后能区分沉睡账户（I2）。是否需要代码上的缓解，由负责人决定。可参考 Django 的 PBKDF2 哈希器：密码错误时补跑缺少的迭代次数；argon2 的成本不是一个迭代数，照搬不直接，没有便宜的做法 |

## 7. 已知限制

- **真实反向代理后面的客户端 IP**：只有单元测试、整程序测试和 README 的说明；P4 或 M8 的部署核对时再看 Caddy 后面的实际行为。
- **3.5 的六个交错测试在真实数据库上的版本**是 P3 的完成线；P2 有用例层的钩子测试和锁的集成测试。
- **大量不同键时限流器的内存**：清理有测试，规模没有压测，v0 的部署规模下不是问题。
- **限流器在进程内**：v0 只有一个进程（总体设计 3.6）。
- **调高 argon2 参数之后的登录耗时**（评审 I2）：存储的哈希都用当前参数时，已有地址和未知地址的失败耗时相同；调高参数之后，还没有重新登录过的账户与未知地址可以用耗时区分，直到这些账户各登录一次。已登记在 M2 设计 §16。
