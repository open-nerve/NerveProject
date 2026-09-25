# M2/P1 平台约定与第一个认证：评审记录

| 项 | 内容 |
|---|---|
| Phase | M2/P1 `platform-core` |
| 日期 | 2026-09-26 |
| 结论 | **通过**（整分支评审提出的问题已在一轮修复中处理完） |
| spec / plan | [spec](../specs/P1-platform-core.md) / [plan](../plans/P1-platform-core.md) |
| 分支 | `worktree-m2-design`，从 `main` 的 `7ef5f9c` 分出；M2 设计（定稿 `b253681`）与 P1 在同一分支，一起合并（M0、M1 的先例） |

## 1. 评审方式

- **spec 和 plan 由架构子任务产出**：
  - 先在仓库外把整个 P1 做出来（原型），测试和 lint 全部通过。
  - plan 中的代码由脚本从原型文件原样拼入。
  - 再从 `b253681` 的副本出发，按 plan 逐个 Task 重放。每个 Task 之后 `make lint-go`、`make test` 都通过；Task 6、Task 12 之后另跑端到端。重放的结果与原型逐文件相同（spec 附录 A）。
- **执行前的核对**：控制者逐对核对共用文件和接口的 Task（16 行），逐个核对 Task 自身的测试与代码。没有发现冲突。spec 第 3 节的 12 项差异全部接受（第 3 节）。
- **逐个 Task 评审**：13 个 Task 各由一个实现子任务照 plan 写入代码，随后由独立的评审子任务检查"是否符合 spec"和"代码质量"。
  - 下一个 Task 的实现与上一个 Task 的评审并行。同一时间只有一个实现者提交。
  - Task 1、Task 9 在评审之前由控制者裁定、修了一轮（第 3 节）。
  - Task 8、Task 10、Task 13 在评审之后各修一轮，修复的提交再做范围评审。
  - 其余 Task 一次通过。
- **整分支评审**（`4c49dc3..750e9c3`）：结论为"修复后可合并"，有 0 个 Critical、1 个 Important、7 个 Minor。评审同时复核了执行中的每一项裁定（都成立），并逐条分拣了逐个 Task 评审留下的 Minor。修复在一轮中完成（`750e9c3..98ece6b`），之后对修复的提交做了一次范围评审：六项都已处理，修复没有带来新问题。
- **持续集成**：推送后通过 GitHub 公开接口确认结果，见第 2 节。

## 2. 验收标准核对（spec 第 4 节）

| # | 标准 | 结果 |
|---|---|---|
| 1 | A1、A2 的接口版本和 S1–S4 在持续集成中通过；`TestBuiltInProfiles` 证明 prod 默认关闭注册 | 通过：持续集成 e2e 任务通过（`750e9c3` 的 run 36182749782，E2E 一步 18 秒；修复轮 `98ece6b` 的 run 36187332526，22 秒）。本机 `make e2e` 7/7，`--repeat-each=10` 70/70（Task 12）。`TestBuiltInProfiles` 在 `server/configs/embed_test.go` |
| 2 | 四个整程序测试和 `apitest` 的新核对通过 | 通过：`bootstrap/contract_test.go` 的 `TestPublicOperationsAreTheContractsPublicOperations`、`TestAPIRoutesAreTheContractsOperations`、`TestOperationsThatNeedATokenAnswer401WithoutOne`、`TestBodiesThatBreakTheStructureAnswer400`；`apitest` 的写法、问题码两个方向（`CheckResponse` 和 `Main`）、`CheckRequest` |
| 3 | `make gen-check` 覆盖 sqlc 和请求体结构表的输出；`TestGenerateIsDeterministic` 通过 | 通过：每个改了生成来源的 Task 之后 `make gen-check` 都干净；持续集成的 server 任务先跑 `make gen-check-go` |
| 4 | 架构测试（含 `TestSQLCSchemaScope`）和传递依赖测试通过 | 通过：`archtest` 的规则各有合成的正例和反例；`TestSQLCSchemaScope` 核对 12 种布局；`TestNerveBinaryLinksNoBannedModule`、`TestPureLayersReachNoInfrastructure` |
| 5 | 审计列等于固定时钟的集成测试、CHECK 的反例测试通过 | 通过：`identity` 的 postgres 适配器中 `TestCreateAndGetUser`、`TestCreateDefaultProfile`、`TestCreateSessionAndReadItsCredential` 核对三张表的 `created_at`、`updated_at` 等于固定时钟；`migrations/schema_test.go` 的 `TestChecksRejectCounterexamples` 25 个反例；`TestCreateUserBreakingACheckIsInternal` 证明 CHECK 被触发时答 500 |
| 6 | `TxManager` 在请求的 `context` 被取消后仍然提交、仍然回滚的集成测试通过 | 通过：`TestWithinTxCommitsAfterTheContextIsCancelled`、`TestWithinTxRollsBackAfterAFailedStatementAndCancel`。去掉 `context.WithoutCancel` 后两者都失败（Task 2）。修复轮加上"回滚失败时结果是基础设施错误"的测试（第 4 节 F3） |
| 7 | `make test`、`make lint-go` 覆盖工具模块，持续集成的日志里能看到 `ok …/server/tools/bodyshapegen` | 通过，证据是结构上的：持续集成 server 任务的 Test 一步运行 `make test`，其中第二行是 `go -C server/tools test -count=1 ./...`；Lint 一步运行 `make lint-go`，其中第二行在 `server/tools` 运行。两步都成功。日志正文要仓库管理员权限才能从接口下载（403），本机的 `make test` 输出里有 `ok …/server/tools/bodyshapegen`（Task 10、Task 11 的记录） |
| 8 | `make lint`（含 `make lint-web` 的关键词守卫和 `e2e` 的 oxlint 上限 0）、`make knip` 通过 | 通过：本机每个动了前端或 `api/` 的 Task 都跑过；持续集成 web 任务的 Lint、Unused code 两步成功 |
| 9 | 3.20 中 P1 的各行、8.7 中 P1 的 README 内容已在同一次合并中写好；交接按第 7 节处理 | 通过（Task 13 及其修复轮）：评审逐项对照代码核对了差异清单的列数和交接的状态；控制者追加的五行 3.20（M0 设计 3.1、3.3、3.5，总体设计 6.2、8.2）也已同步 |

spec 第 6 节要求写进 review 的实测：
- **argon2 默认参数**（m = 19 MiB、t = 2、p = 1）：本机（Apple M5 Max）每次 14.5–15.5 毫秒，约 19 MiB（`BenchmarkHashDefaultParams`，Task 9）。M2 设计 3.10 按每次约 40 毫秒估算的容量因此偏保守。持续集成不跑基准测试，见第 7 节。
- **持续集成耗时**（`750e9c3`）：web 130 秒，server 149 秒（其中生成代码检查 76 秒、lint 29 秒、测试 19 秒），e2e 138 秒（其中端到端 18 秒）。修复轮 `98ece6b`：web 136 秒，server 56 秒（生成代码检查 1 秒、lint 10 秒、测试 18 秒），e2e 128 秒。

## 3. 执行中的决定

spec 第 3 节的 12 项差异全部接受，理由各在该节。其中需要回写设计的，由 Task 13 的控制者补充说明一并同步（第 5 节）。

| 决定 | 原因 |
|---|---|
| CHECK 被触发是 500；只有 `users_email_key` 的唯一冲突映射为 `identity.email_taken`（409） | 领域先校验，CHECK 被触发说明两者不一致，是缺陷，要暴露出来，不能装成 422。M2 设计 9.2 原来写"违反唯一约束和 CHECK 时映射为领域错误"，已改 |
| 顶层 `x-problem-codes` 在 P1 只有三个码，`rate_limited` 随 P2 的限流加入 | 没有代码能答出的码是一条假的契约 |
| `ExpiredCredential()`、argon2 的 `Verify`、刷新令牌的解析和标签核对、`Actor.APITokenID`、`nrv_pat_` 分支随第一个使用者在 P2、P3 加入 | 不写用不上的代码 |
| 签名密钥在 `identity.New` 中解析，`config` 只检查 prod 是否设置 | 解析密钥是 identity 适配器的事；prod 仍在启动时失败，并指出配置项 |
| Task 1：`bodyshape.Check` 在根上只去掉 JSON 的空白（空格、`\t`、`\r`、`\n`），再逐字节检查 | plan 的代码对前面带空白的合法请求体误报 400；`bytes.TrimSpace` 又会放过解码器拒绝的 `\f`、`\v`、NBSP |
| Task 5：`apitest` 保留进程级的 `answered` 记录 | 反向核对要在 `m.Run()` 之后汇总一个包里所有测试的结果，只有进程级的状态能做到；`apitest` 只在测试中使用，加锁，只暴露记录和快照两个操作 |
| Task 8：加强测试替身，记下 `GetUser` 收到的 id 和 `Verify` 收到的时间，并断言它们 | "按调用方读取"和"业务时间来自时钟"是必须成立的性质，原来的替身守不住它们 |
| Task 9：访问令牌的 `Verify` 只报四种固定的失败原因（格式错误、签名无效、声明无效、已过期） | 伪造的令牌里字符串类型的 `exp`、`nbf`、`iat` 会让发送方写的文字经 golang-jwt 的错误进入 DEBUG 日志 |
| Task 10：注册的契约说明改为"格式正确的请求在检查邮箱和密码之前就答 `identity.signup_disabled`"，重新生成打包文件和 TS 类型 | 原来写"先于任何检查"，但请求体结构的 400、413 在它之前；公开的契约必须描述真实的行为 |
| Task 13：README 在 NCSC 的出处之外写上 SecLists 的下载地址 | NCSC 的原地址已失效，需要一个能用的来源 |
| Task 13 的控制者补充说明：同步 M2 设计 9.2（CHECK 是缺陷）、3.4 和 §17 M-7（91 个字符是 base64 部分，整个刷新令牌 98 个字符）、3.20 补五行、5.1 注册一行的措辞 | 设计文档是事实的来源，P1 的裁定要回写 |
| 整分支评审的修复范围：F1–F6（第 4 节）；其余 Minor 按阶段移交（第 6 节）或接受 | 修平台约定上的漏洞和一行的文字；其余各有落点 |

## 4. 整分支评审的发现与处理

| 发现 | 级别 | 处理 |
|---|---|---|
| I1 `bodyshapegen` 不会因为数值类型而失败：`format: int8`–`int32` 生成窄整数，`uint*` 生成无符号整数，没有格式的 `number` 默认生成 `float32`；`bodyshape` 只检查整数能否解析为 int64，不检查数的范围。于是 int32 字段的 `3000000000`、uint 字段的 `-1`、number 字段的 `1e39` 都会通过边界检查，再被解码器拒绝，得到不带 `errors[]` 的笼统 400。逐个 Task 评审把整数宽度推迟，依据是"生成器遇到未知格式会失败"，这对数值类型不成立 | Important | 已修（F1，`d648b34`）：生成器按 oapi-codegen 的类型映射解析 `integer`、`number` 的 Go 类型，只接受 `int`、`int64`、`float64`；窄整数、无符号整数、`float32`、映射中没有的格式都让生成失败并说明原因。模块模板加上 `number` → `float64`。`bodyshape` 只把 float64 能表示的字面量当作数，`1e400` 在边界上就是 `invalid_format`（`TestNumbersMatchTheDecoder` 逐个对照 `json.Unmarshal`） |
| M1 经 `APIErrors.Write` 答出的 401 不带 `WWW-Authenticate` | Minor | 已修（F2，`12f9b6b`）：`Write` 给没有质询的 401 加上 `WWW-Authenticate: Bearer`，已经设置的质询保留（`TestWriteChallengesEvery401`） |
| M2 YAML 中的负数或溢出的数写进新的无符号配置项时会回绕，例如 `argon2_memory_kib: -1` 成为 4294967295 KiB | Minor | 移交 P2 的配置加固（第 6 节） |
| M3 `fn` 返回领域错误而 `ROLLBACK` 又失败时，`errors.Join` 让 `Write` 找到领域错误并答 409 之类，回滚的故障不进日志 | Minor | 已修（F3，`c2f404d`）：回滚失败时，结果是基础设施错误：包住回滚的错误，只保留领域错误的文字，`Write` 答 500 并记日志（`TestWithinTxReportsAFailedRollback`）。M2 设计 3.6 加了一句 |
| M4 没有测试核对 `shared` 的字段错误码等于 `api/common.yaml` 的枚举 | Minor | 已修（F4，`acd799f`）：`shared.FieldCodes()` 列出全部字段错误码，`TestFieldCodesListEveryFieldConstant` 核对它不漏常量；`bootstrap` 的 `TestFieldCodesAreTheContractsEnum` 与 `api/dist/openapi.yaml` 的枚举两个方向比较 |
| M5 sqlc 所有者规则用正则识别，漏掉带引号的标识符和 `ALTER` 以外的 DDL（`CREATE INDEX … ON users`、`CREATE TRIGGER`、`DROP TABLE`） | Minor | 移交 M3（M2 设计 13.2 新加一行） |
| M6 契约没有声明 `Retry-After`、`WWW-Authenticate` 响应头 | Minor | 移交 P2（`rate_limited` 同样需要 `Retry-After`） |
| M7 文字漂移：M2 设计 3.11 的 `UseNumber`；M0 设计 3.1 的 `domain` 一行；spec 第 2.4 节写错第四个失败的测试名；`configs/config.yaml` 缺少 `nerve users create` 在 P3 才有的说明 | Minor | 已修（F6，`98ece6b`）。spec 的状态在本评审的提交中改为"已完成" |

逐个 Task 评审留下的 Minor，整分支评审逐条分拣后：
- **本轮修掉**：`bodyshape.Check` 对空请求体会越界（F5，`90cba13`：`Check` 自己处理空请求体和不是一个 JSON 值的请求体，签名改为返回 `error`；中间件只调用它，判断只有一处；`\f` 的例子移到新测试中，预期是"不是 JSON"）；`identity/domain/errors.go` 中 `ErrSignupDisabled` 的注释与契约的措辞对齐（F6）；M2 设计 3.4 的 PAT 一行改为"base64url 部分 43 个字符，整个令牌 51 个字符"（F6）；M2 设计 9.2 的 `onboarding_step` 反例是八种（F6）。
- **移交**：见第 6 节。
- **接受**：`writeAddrFile` 失败时留下 `.tmp`、停机时不删地址文件（只在测试中用，端到端启动前会删）；`unanswered` 丢掉 `problemCodes` 的错误（写法测试会报格式错误的列表）；`needsToken` 把任何非空的 `security` 当作 bearer（v0 只有 bearer）；`sqlcScopeViolations` 一个函数做四项检查（短，有注释）；名单脚本的三条（目录已存在；JS 与 Go 分别实现主干规则，Go 的测试和一次 Go 重建是证据；缺少分隔行时由 `TestCommonPasswordList` 守住）；`errClaimsInvalid` 合并了几种原因（只进 DEBUG 日志，都要求签名有效）；`warnIfExposed` 的调用没有整程序测试（逻辑有单元测试，调用只有一行）；`BodyCases` 只覆盖第一个嵌套对象（符合设计的措辞）；数据库快照要调用 `docker` 命令（端到端本来就需要 Docker）。
- **不成立**：Task 10 的"中间件切片可以直接转换"（元素类型不同，不能编译）；Task 10 的"`Nullable` 零值的注释没有核实"（nullable v1.2.0 确实会写成 `""`，注释正确）。

修复轮的实现者另外找到两处同类的缺口，都没有使用者，移交（第 6 节）：
- 不限类型的节点（`{}`、`additionalProperties: true` 的开放对象、没有 `items` 的数组）不看数的范围，其中的 `1e400` 仍然得到解码器笼统的 400。
- OpenAPI 3.1 的 `contentEncoding`、`contentMediaType` 会让 oapi-codegen 把字符串生成为 `[]byte` 或 `File`，生成器的字符串检查只读 `format`，不会因此失败。

## 5. 最终实现与 plan 的差异

plan 是执行记录，不回改。以 spec 和代码为准。

1. **Task 1**：`bodyshape.Check` 和中间件的空请求体判断只去掉 JSON 的空白（`6393bb8`）。
2. **Task 8**：测试替身记下 `GetUser` 的 id 和 `Verify` 的时间，测试断言它们；过期时间由时钟的时间推出（`7262cf8`）。
3. **Task 9**：`Verify` 只报四种固定的失败原因，14 行的表格测试（`943d255`）。
4. **Task 10**：注册的契约说明（`c461fc9`）。plan 记录的 `api/dist/openapi.yaml`（`39f1119f…`）和 `schema.gen.ts`（`d3046f6d…`）的 SHA-256 因此不再成立：新值分别是 `6a2610a6…`（259 行）和 `2027056d…`（237 行），行数不变；`server.gen.go`、`bodyshape.gen.go` 不变。
5. **Task 13**：控制者补充说明的设计同步（`5166d30`）；README 写上 SecLists 的下载地址（`750e9c3`）。
6. **修复轮 F1–F6**（`750e9c3..98ece6b`）：
   - F1：`bodyshapegen` 按类型映射拒绝范围检查不到的数值类型；两个模块模板和测试数据加上 `number` → `float64`；`bodyshape` 的数只认 float64 能表示的字面量。
   - F2：`APIErrors.Write` 给没有质询的 401 加上 `WWW-Authenticate: Bearer`。
   - F3：`TxManager` 回滚失败时返回基础设施错误。
   - F4：`shared.FieldCodes()`、`apitest.Contract.Enum`，以及字段错误码与契约枚举的对照测试。
   - F5：`bodyshape.Check` 返回 `error`，负责空请求体和不是 JSON 的请求体；中间件只调用它。
   - F6：七处文字（M2 设计 3.4、3.11、9.2，M0 设计 3.1，spec 第 2.8 节，`configs/config.yaml`，`identity/domain/errors.go`）。
   - 修复轮没有改变任何生成文件，`make gen-check` 干净。
7. **父文档同步**：M2 设计 3.4、3.6、3.11、3.20、5.1、9.2、§15、§17 M-7 和 13.2；M0 设计 3.1、3.3、3.5；总体设计 3.1、3.5、4.1、4.2、5.5、5.6、6.2、6.3、6.4、8.2；差异清单。

## 6. 移交事项

| 交接 | 交给 | 内容 |
|---|---|---|
| [M0-P1-sqlc-cgo](../handoffs/M0-P1-sqlc-cgo.md) | — | **已处理（done）** |
| [M0-P4-schema-conventions](../handoffs/M0-P4-schema-conventions.md) | — | **已处理（done）** |
| [M0-P2-platform-notes](../handoffs/M0-P2-platform-notes.md) | M2/P2、P3，M8 | P1 处理的条目写在文末的处理结果中；留下限流（P2）、接口调用日志（M8）、River 与停机（P3） |
| [M0-P3-api-codegen-notes](../handoffs/M0-P3-api-codegen-notes.md) | M2/P3 | 留下 google/uuid 的守卫和参数绑定的出口 |
| [M0-P6-e2e-notes](../handoffs/M0-P6-e2e-notes.md) | M2/P2–P4，M4、M5、M8 | 留下 PAT 对等验收、认证 fixture 的登录和页面部分、S3 的新字段、S2 的断言、River 停机、fixture 写法的延伸 |

整分支评审留给后续 Phase 的事项（P2、P3 的 spec 从这里取；M3、M4 的已写进 M2 设计 13.2）：

| 交给 | 事项 |
|---|---|
| P2 | **配置加固**：YAML 中负数或溢出的数写进无符号配置项时拒绝，不回绕（评审 M2）；YAML 中布尔配置项的空值（`signup_enabled:`）不能悄悄成为 false；随 P2 的配置项加入启动时的不等式"续期期限（4 秒）+ `commit_timeout` < 8 秒"和 `trusted_proxies` 的 CIDR 校验 |
| P2 | `NewAPI` 在构造时检查依赖不为空（P2 的 `APIConfig` 加上限流器时一起做） |
| P2 | 契约声明 `Retry-After`、`WWW-Authenticate` 响应头（评审 M6），与 `rate_limited` 一起加入 |
| P2 | `apitest` 的测试卫生：`TestCheckResponseRecordsTheAnsweredCode` 用独有的操作和码，不依赖前面的测试留下的状态；`Main` 中的循环变量 `m` 改名 |
| P2 | `TestAuthenticateTellsAnExpiredAccessToken` 的注释写"past exp"，测试实际用的是恰好等于 `exp` 的边界；P2 为 `ExpiredCredential` 改这个测试时一起改 |
| P3 | `apitest` 的 `Operation.Target()` 遇到 `content:` 形式的参数、`BodyCases()` 遇到不是对象的请求体时会 panic；P3 的第一批带参数的操作到来时扩展 |
| P3 | `bodyshapegen` 遇到 `contentEncoding`、`contentMediaType`（生成为 `[]byte`、`File`）时同样失败并说明原因；P3 是下一次改动请求体检查的 Phase（格式检查器的第一批使用者） |
| M3 | sqlc 所有者规则的正则漏洞（评审 M5）：带引号的标识符；别的模块的表上 `ALTER` 以外的 DDL |
| M4 | 字段错误的路径按字典序排序（`tags[10]` 在 `tags[2]` 之前），改为按数值排序；不限类型的节点检查数的范围（`1e400`）。两者都在第一个带数组或开放对象请求体的 M 处理，在那之前客户端不能依赖路径的顺序 |

## 7. 已知限制

- **持续集成上的 argon2 耗时没有实测**：持续集成不跑基准测试，任务日志的正文要仓库管理员权限才能下载。测试环境用 m = 64 KiB、t = 1，测试的耗时也不代表生产参数。M8 的性能实测一并测量（M2 设计 13.2 的 M8 一行）。
- **P1 到 P3 之间 prod 建第一个账户**要临时设 `NERVE_AUTH__SIGNUP_ENABLED=true`：`nerve users create` 在 P3 才有。v0 在 M2 收尾前不发布，没有实际影响（spec 第 5 节）。
- **`TestWithinTxReportsAFailedRollback` 依赖内核的缓冲**：测试断言回滚得到的是服务端的 FATAL（`57P01`）。这取决于服务端结束连接时发出的消息已经在客户端的套接字缓冲中，而不是协议的保证。本机 30/30、范围评审 3/3、持续集成的 Linux 都通过。以后如果偶发失败，把断言放宽为"回滚的错误在链中、结果不是 `ProblemError`"，生产代码不变。
- **请求体检查器的第一个使用者在 P3**：`date-time`、`uuid` 的格式检查器已有单元测试，整程序测试中逐格式的情况从 P3 起有操作可测。
