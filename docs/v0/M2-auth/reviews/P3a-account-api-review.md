# M2/P3a 账户接口：评审记录

| 项 | 内容 |
|---|---|
| Phase | M2/P3a `account-api` |
| 日期 | 2026-09-26 |
| 结论 | **通过**（整分支评审提出的问题已在一轮修复中处理完，修复的范围评审没有遗留） |
| spec / plan | [spec](../specs/P3a-account-api.md) / [plan](../plans/P3a-account-api.md) |
| 分支 | `worktree-m2-p3-account-api`，从 `main` 的 `605f367`（P2 的合并）分出 |

## 1. 评审方式

- **拆分**：P3 按负责人的裁定（2026-09-26）拆成 P3a `account-api` 和 P3b `jobs-and-admin`。命令行的"只投递"River 客户端经负责人批准推迟到 M4（M2 设计 3.15、3.17、13.2 的 M4 一行）。
- **spec 和 plan 由架构子任务产出**（`c429b11`）：
  - 先在仓库外把整个 P3a 做出来（原型），测试、lint、端到端全部通过。
  - plan 中的代码从原型原样拼入。再从 `605f367` 的副本出发，按 plan 逐个 Task 重放，结果与原型逐文件相同。
  - 原型中 215 个变异都被测试发现（spec 附录 A）。
- **执行前的核对**：
  - 控制者逐对核对共用文件和接口的 Task（20 行），没有发现冲突。
  - spec 第 3 节的 24 项差异全部接受。其中两项改了 M2 设计的文字，在控制者的提交 `33f0b4b` 中同步（第 3 节）。
  - spec 第 7 节留给控制者的一项（P1、P2 的六个测试文件读活的头部映射）加为 Task 16。
- **逐个 Task 评审**：16 个 Task（plan 的 15 个加 Task 16）各由一个实现子任务写入代码，再由独立的评审子任务检查"是否符合 spec"和"代码质量"。
  - 实现者按要求对自己的代码做变异核对。**16 个 Task 中有 14 个**发现了以下问题之一（Task 3 留下的两个变异无法区分，接受；Task 16 没有），控制者逐项裁定，修正都在评审之前提交（第 3 节）：
    - plan 自带的测试没有咬住它声称守护的性质；
    - 测试会挂住而不是失败；
    - 注释、文档写了代码没有的行为。
  - 逐个 Task 评审没有要求返工。评审提出的 Minor 由控制者分拣：顺延到后面的 Task、放进修复轮，或者接受。
  - 下一个 Task 的实现与上一个 Task 的评审并行，同一时间只有一个实现者提交。
- **整分支评审**（`605f367..c5f5ff6`）：
  - 结论为"修复后可合并"：0 个 Critical、1 个 Important、6 个 Minor。
  - 评审跑了 `go test ./...`、`make lint-go`，以及四个包的 `-race -count=3`；另用三个 overlay 探针核对了发现。
  - 修复在一轮中完成（`c5f5ff6..13d6396`，12 项）。之后对修复的提交做了一次范围评审，12 项都已处理，没有新问题。
- **持续集成**：推送后通过 GitHub 公开接口确认结果，见第 2 节。

## 2. 验收标准核对（spec 第 4 节）

| # | 标准 | 结果 |
|---|---|---|
| 1 | A7–A11 的接口版本和 S3 通过，P1、P2 的故事仍然通过 | **通过。**<br>本机 `make e2e` 的 17 个测试全部通过，覆盖 S1–S4、A1–A11、A15（Task 14 和修复轮）。<br>A7–A11 用 `--repeat-each=20` 跑，100 个全部通过。<br>持续集成的 e2e 任务通过（第 2 节末） |
| 2 | 每个需要登录的新操作都有只用 PAT 的测试 | **通过。**<br>`bootstrap` 的 `account_test.go`（资料、偏好、修改密码、停用）和 `auth_test.go`（PAT 创建 PAT、列表、撤销）。<br>A7–A11 在创建 PAT 之后全程只用 PAT |
| 3 | 整程序测试覆盖新操作：带格式的字段写错时 400 `invalid_format`；会拒绝某些字符串的参数写错时 400 | **通过。**<br>`TestBodiesThatBreakTheStructureAnswer400` 从接口描述推出每个带格式的字段的用例（即 `createApiToken` 的 `expired_at`、`updateProfile` 的 `last_workspace_id`），以及"每一种问题一起"的请求体。<br>`TestParametersThatDoNotBindAnswer400` 从接口描述推出参数（`limit`、`cursor`、`token_id`），错误信息写出参数名 |
| 4 | 3.5 的交错测试 4、5 在真实数据库上通过，`-count=5` 也通过 | **通过。** `-count=5`、`-race -count=5`、`-count=20` 都通过。<br>另加 `TestAPasswordChangeWaitsForALoginThatHoldsTheLock`（第 3 节 Task 13）：原来的两个测试从不争锁 |
| 5 | 带着 `oapi-codegen/runtime` 的 nerve 通过改写后的传递依赖测试，生成代码的 uuid 规则通过 | **通过。**<br>`TestBannedImports` 按导入的边判断。<br>`TestGeneratedCodeUsesTheStandardUUID` 覆盖 `apigen` 和各模块的 `gen`。<br>修复轮另在 depguard 中禁止手写代码导入 `runtime`、`runtime/types`（F3） |
| 6 | `onboarding_step` 的并发合并测试通过；`limit` 为 0、101（422）、`abc`（400）的测试通过 | **通过。**<br>`TestUpdateProfileMergesConcurrentSteps` 等到 `pg_stat_activity` 显示锁等待之后才放行；失败时 11 秒内失败，不会挂住。<br>页大小和参数的测试在 domain、handler 和整程序三层都有 |
| 7 | 3.10 表中的每个桶都有整程序测试；`password_user` 按账户计数 | **通过。**<br>`TestEachConfiguredBucketLimitsItsOperations`：7 个桶的突发各不相同，逐个打到上限。<br>`TestChangePasswordLimitsByAccount`：第三次改用同一账户的另一个凭证、另一个 IP。按 IP、会话或凭证计数时测试都失败。<br>关闭 P2 评审的 M7 |
| 8 | `make lint`、`make test`、`make gen-check`、`make knip`、`make e2e` 通过 | **通过。**<br>每个 Task 提交前，`make lint-go` 两段都是 `0 issues.`，`make test` 通过。<br>有生成物的 Task 核对了 SHA-256；提交后 `make gen-check` 干净。<br>改了 web 或 e2e 的 Task 跑了 `make lint-web`、`make knip`、`make e2e`。<br>持续集成三个任务都通过 |
| 9 | 3.20 中 P3a 的各行、8.7 中 P3a 的两行在同一次合并中写好；交接按第 7 节处理 | **通过**（Task 15 及其修正）。<br>评审抽查了 20 多句新写的文档，逐句对照代码、契约和 Plane 源码。<br>M0-P3 交接关闭；M3 的 `M0-P3-pagination-components` 交接随 P3a 的结果关闭 |

持续集成：
- 修复轮 `13d6396` 的分支 run 36235691569：三个任务都通过。耗时：server 159 秒（测试 26 秒），web 132 秒，e2e 134 秒（端到端 19 秒）。

## 3. 执行中的决定

spec 第 3 节的 24 项差异全部接受。它们都是设计留给 P3a 的细节，没有一项跨模块，也没有改动平台与 `shared` 的边界。其中两项改了 M2 设计的文字，在执行之前由控制者提交（`33f0b4b`）：
- **5.4**：`current_password_incorrect` 不带 `errors[]`，因为字段码是封闭的集合（第 2 项）。
- **13.2**：加一行 M3，页大小的规则在第二个分页列表出现时移到 `shared`（第 8 项）。

执行中的裁定。除特别注明的以外，都只改测试或文字，都由实现者的变异核对发现，都在评审之前修正：

| Task | 决定 | 原因 |
|---|---|---|
| 1 | 加 `TestValidateChecksEachBucketUnderItsOwnKey`：按 `RateLimitConfig` 的字段逐个把一个桶的一项设为 0，只应报出这个桶的键 | plan 的测试把所有桶一起清零，用错了桶的配置校验也能通过；`register_ip` 原来就有同样的缺口 |
| 2 | 三条写法规则各加一个用例：组件请求体、组件参数、路径级参数 | 去掉规则在这三处的检查，测试仍然通过 |
| 3 | 接受两个留下的变异：排序是否稳定在 13 个元素以下分不出来；删掉整条 glob，干净的树上照样通过 | 前者没有测试能分辨，结果本来就确定；后者是扫真实代码树的规则的固有性质 |
| 4 | **`DecodeCursor` 只接受 `EncodeCursor` 写出的形式**：解码后重新编码，逐字节比较（改了生产代码）。去掉三处多余的检查，保留 `null` 检查；补上换行、base64url 字母表、`bad_request` 字面值的用例 | 原来接受编码器从不写出的形式（base64 里的换行、未用的位、大写或重复的键），注释却说"其余一律不合法"。一条规则关掉整类可塑性 |
| 5 | 列表每页停在上限；更新 `last_used` 只写这一个令牌；两次读取都读回 `last_used`；**再加一个状态相反的账户**，按哈希、按 id 读凭证都必须回这个令牌自己的账户 | 连接条件写成"任意账户"时，第二个账户出现后 PAT 会拿到别人的停用状态和凭证；锁内复核信任这次读取 |
| 6 | 永不过期的令牌能认证；检查规格在开事务之前；游标行的 `created_at` 也核对 | 三个变异原来都能通过 |
| 7 | 列表钉住每个字段（一个全填、一个全空）；改正 `nullableTime` 的注释；**`TestParametersThatDoNotBind` 带合法令牌** | 原来"用例没有运行"的断言永远不会失败：不带令牌时，认证本来就拦下了请求 |
| 8 | 并发合并测试失败时释放连接（不再挂住）；未知行的测试旁边放一行真实数据；空的 PATCH 钉住 `updated_at`；两个标志写不同的值；每个文本字段都测 NUL；`validTimezone` 的注释照实写 | 101 个变异原来有 19 个存活、1 个挂住 |
| 9 | 没有调用者时不查存储；`PATCH /me` 的两行用例之一带全部字段 | `RequireActor` 被跳过、`last_name` 的映射都没有测试守住 |
| 10 | **`password_user` 跨凭证按账户计数**，第三次用同一账户的 PAT；日志测试也核对两个哈希的 base64；写入失败的测试；重试只读一次账户 | 按凭证计数时，同一账户换一个令牌就能多 5 次/分钟 |
| 11 | 顺带三项：`GET /me` 读回；`callLog` 的注释；`fakes_test.go` 按关注点拆成 4 个文件（纯移动）。再加三项：写入失败的测试、handler 答用例的错误、登录断言 `account_deactivated` | 停用的三处写入出错，以及 handler 吞掉错误，原来都没有测试发现 |
| 12 | 实例测试用两组设置，每个字段开关各一次；顺带补上 Task 3 注释缺的动词 | 把 `signup_enabled` 写死为 `true`，原来所有测试都通过，而生产的默认值是 `false` |
| 13 | **加 `TestAPasswordChangeWaitsForALoginThatHoldsTheLock`**：登录在持锁的事务里停住，等 `pg_stat_activity` 显示修改密码在等这把锁，最后登录新建的会话被撤销。测试 4、5 的注释只写它们实际守住的东西 | 测试 4、5 的闸门在 argon2 里，另一个操作总在登录的事务开始前提交完。去掉 `FOR NO KEY UPDATE`，或把锁放到事务外，两个测试都照样通过 |
| 14 | A10 先完成 `workspace_join`，要求它保留；A11 加第二个账户，撤销它的令牌得到 404，它的令牌照样能用 | 所有步骤开始都是 `false`，"合并"和"先清空再写"原来分不出；只有一个账户时，撤销别人的令牌答 204 也能通过 |
| 15 | **游标不签名**：v0 设计 3.4 和已提交的 `common.yaml`、`InvalidCursor` 的文字只写 `DecodeCursor` 实际检查的东西（改了生产代码中的文字）。README 登录耗时的一句带上 argon2 参数的条件（P2 评审 I2）。另补 v0 设计 3.1、4.2，关闭 M3 的分页交接 | M2 设计 3.12 从没要求签名，列表都按调用者过滤，改过的位置只在自己的列表里移动；brief 中的两句写了代码没有的行为 |
| 16 | 控制者加的 Task：六个测试文件（另加 `middleware_test.go:83`）读 `rec.Result().Header`；游标的往返钉住 `<>&`、`v:0`、`v:1.0`、整秒和非 UTC 的时间 | P2 发现的缺陷类别 D 在 P1、P2 留下的测试中；Task 4 的裁定依赖往返成立，需要测试钉住 |

控制者在过程中的其他决定：
- **过渡文件**：后面的 Task 还要打补丁的过渡文件，修正都写在只有这个 Task 用的测试文件里。改到过渡文件时，先用 `patch --dry-run` 确认后面的差异块仍能应用（Task 8、9、10）。
- **模糊应用**：Task 11 的 `fakes_test.go` 第 2 个块在 Task 10 加了错误检查之后，按默认模糊度应用，并核对了结果。
- **不改的项**：
  - 撤销 PAT 时不写 `updated_by_id`：P3a 只有本人撤销，移交 P3b（第 6 节）。
  - 列表索引是否部分索引，schema 测试不核对。
  - `ListTimezones` 的错误路径没有测试：修复轮 F7 之后这条路径已不存在。

## 4. 整分支评审的发现与处理

| 发现 | 级别 | 处理 |
|---|---|---|
| I1 `createApiToken` 把客户端的 `expired_at` 原样答回：可能不是 UTC，也可能是亚微秒，与存下的值不同 | Important | **已修（F1，`a148238`）。**<br>`CheckAPIToken` 返回规范化之后的规格：UTC、截到微秒。检查、写入和回答都用它。<br>domain、app、整程序三层都有测试：带偏移、7 位小数的输入，创建的回答与之后的读取相同 |
| M1 日志测试没有核对 PAT 的 32 个随机字节 | Minor | **已修（F2，`a148238`）** |
| M2 `runtime/types.UUID` 是 google `uuid.UUID` 的别名，手写代码导入它时，三道检查都发现不了 | Minor | **已修（F3，`68dfe97`）。** depguard 禁止手写代码导入 `runtime`、`runtime/types`，生成文件照 golangci-lint 的默认规则排除 |
| M3 一个游标位置有多种写法都被接受（带偏移的 UTC 时间） | Minor | **已修（F4，`efbce77`）。** `MarshalJSON` 写 UTC，每个位置只有一种写法。带 `+08:00` 的写法现在是 400 |
| M4 `identity/domain` 的包注释说"没有 I/O"，但 `validTimezone` 读宿主的时区文件 | Minor | **已修（F5，`dac4934`）** |
| M5 `api/modules/identity.yaml` 619 行，超过约 400 行 | Minor | **不改。** 设计规定每个模块一个契约文件（M0-P3 交接第 5 条），oapi-codegen 和 bodyshapegen 每个模块读一个输入。约 400 行的规则不适用于契约文件，以后的 plan 照此写明 |
| M6 用例在锁内复核时发现凭证已撤销，答的 401 不带 `error="invalid_token"`，与契约的说法不符 | Minor | **已修（F6，`f588a9e`）。** 契约的文字照实写：中间件的检查带 `invalid_token`；请求处理中发现凭证已撤销，答普通的 `Bearer`。RFC 6750 中这个属性是可选的，M2 的客户端收到任何 401 都会续期 |

评审"不作判断"的清单，控制者逐条裁定：
- **令牌响应不带 `Cache-Control: no-store`**：
  - **已修（F10，`c39a498`）。** 固定链的安全响应头中间件给 `/api/` 下的每个响应加上 `no-store`，包括问题响应、平台的 `/api/` 404 和 panic 之后的 500。`webui` 的文件照旧由 `webui` 设置自己的缓存头。
  - 同步了 M2 设计 8.3、v0 设计 6.4、M0 设计 3.3（`13d6396`）。
  - 理由：注册、登录、续期、创建 PAT 的回答都带凭证，接口的回答又都是调用者自己的数据。
- **写 `last_used` 失败时请求答 500**：
  - **已修（F11，`c142db0`）。** 写入失败记 WARN（带令牌 id，不带令牌），认证照常成立；读取失败仍然是 500。
  - M2 设计 3.6 把 `last_used` 定为"尽力而为"，spec 2.9 原来的写法与它相反，这里以设计为准。
  - 整程序测试用触发器让写入真的失败。
- **不改，按设计或 spec**：每个账户的 PAT 没有上限（8.5，与 Plane 相同）；宿主决定哪些时区名能用（spec 第 3 节第 10 条）；读取令牌的端口有两种查法（spec 2.8）。
- **记在第 5 节**：`TestContainsURLAsPlane` 是 51 个用例（spec 写 41 个）。
- **在本评审的提交中改**：spec 和进度表中的"进行中"。

放进修复轮的另外 4 项：
- **时区在模块构建时加载一次**（F7，`9fd0e41`）：缺少的时区让启动失败，每个请求只按时钟算偏移。
- **锁等待的探测合并为一个**（F8，`299b9ef`）：`pgtest.WaitForLockWait` 由两处共用，P3b 也会用到。
- **e2e 泄露检查的注释**（F9，`7c3319d`）。
- **`httpserver/api_test.go` 按关注点拆成 4 个文件，纯移动**（F12，`44c990c`）：这个文件来自 P2，有 589 行。

## 5. 最终实现与 plan 的差异

plan 是执行记录，不回改，以 spec 和代码为准。

1. **第 3 节的测试和文字修正**：Task 1、2、5–14、16 的修正只改测试或文字。spec 列出的测试因此与实际不同：
   - `TestContainsURLAsPlane` 是 51 个用例，spec 写 41 个。
   - 各 Task 加了 spec 没有列出的测试，见第 3 节。
2. **生产代码的改动**（执行中的裁定）：
   - Task 4：`DecodeCursor` 按"重新编码后逐字节比较"判断。
   - Task 15：`common.yaml` 的 `Cursor` 说明和 `InvalidCursor` 的文字。
3. **修复轮**（`c5f5ff6..13d6396`），改了生产代码的有：
   - F1：`CheckAPIToken` 返回规范化的规格。
   - F3：depguard 规则。
   - F4：游标的时间写 UTC。
   - F6：契约的 `WWW-Authenticate` 说明。
   - F7：`instance` 的时区在构建时加载，`instance.New` 可能返回错误。
   - F10：`/api/` 响应带 `no-store`。
   - F11：写 `last_used` 失败只记 WARN。
4. **生成物**：
   - Task 5–12 的生成物与 plan 记录的 SHA-256 相同。
   - Task 15 的 `Cursor` 说明和修复轮 F6 的说明改了生成来源，新的 SHA-256 记在 Task 15 和修复轮的报告中；`make gen-check` 干净。
5. **父文档同步**：
   - **Task 15**：总体设计 3.1、3.4、3.6、4.2、6.2；M0 设计 3.2、3.7；M0/P3 spec 2.8、第 7 节；差异清单二、三、四；README 的部署一节；五份交接的处理结果。
   - **执行前后的 M2 设计同步**：
     - `c429b11`：12 节、15 节、3.15、3.17、6.1、13.2、§17；
     - `33f0b4b`：5.4、13.2；
     - 修复轮：3.12、8.3。
   - **修复轮另改**：总体设计 6.4、M0 设计 3.3（`13d6396`），spec 2.7、2.9、2.16（`65d1887`）。
   - M2 设计的进度表在本评审的提交中改为已完成。

## 6. 移交事项

| 交接 | 交给 | 内容 |
|---|---|---|
| [M0-P3-api-codegen-notes](../handoffs/M0-P3-api-codegen-notes.md) | — | 整体关闭：google/uuid 的守卫改写，参数绑定的出口由 `listApiTokens`、`revokeApiToken` 测过 |
| [M0-P6-e2e-notes](../handoffs/M0-P6-e2e-notes.md) | M2/P3b、P4，M4、M5、M8 | P3a 处理了 PAT 对等验收、认证 fixture 的 PAT、S3 的新字段；留下页面的登录状态、S2（P4），River 停机与 fixture 的预算（P3b），fixture 写法的延伸 |
| [M1-P2-trim-content](../handoffs/M1-P2-trim-content.md) | M2/P4、P5 | P3a 处理了接口描述的主题（五个值）；留下前端的 `IUserTheme` |
| [M1-P3-trim-platform](../handoffs/M1-P3-trim-platform.md) | M2/P3b、P4 | P3a 处理了实例配置的三个字段、令牌地址；留下 Cookie 会话和 CSRF、认证错误、前端改读实例字段（P4），`set-email`（P3b） |
| [M0-P3-pagination-components](../../M3-workspace-project/handoffs/M0-P3-pagination-components.md)（M3） | — | 随 P3a 关闭：`Limit`、`Cursor`、`NextCursor` 已在 `common.yaml`。页大小的规则移到 `shared` 一事记在 M2 设计 13.2 的 M3 一行 |

P2 评审第 6 节交给 P3 的两项：
- **桶与配置的接线**（M7）：由 Task 10 关闭。
- **不可用的密码保持登录耗时**：交给 P3b（M2 设计 12 节 P3b 的交付物 4）。

交给后续的事项：

| 交给 | 事项 |
|---|---|
| P3b | 交错测试 1–3 必须在真实数据库上真的争账户行锁：闸门放在持锁的事务里，用 `pgtest.WaitForLockWait` 等到等待出现。闸门放在 argon2 里，另一个操作会先提交完，锁就没有被测到（Task 13 的教训） |
| P3b | 管理员重置密码撤销全部 PAT 时，`updated_by_id` 写谁（撤销 PAT 的查询现在不写它，P3a 只有本人撤销） |
| P3b | `nerve users activate` 加入时，重新核对 `deactivateMe` 的说明"直到管理员重新启用" |
| P3b | README 中 8.5 的恢复步骤（`nerve users reset-password`）和 A12 的端到端（spec 第 3 节第 20、24 条） |
| P3b 及以后的 plan | 约 400 行的规则不适用于契约文件（每个模块一个，M0-P3 交接第 5 条）；plan 的 Global Constraints 照此写明 |
| M3 | 第二个分页列表出现时，把页大小的规则移到 `shared`（M2 设计 13.2） |
| 负责人 | 调高 argon2 参数之后，登录耗时能区分沉睡账户（P2 评审 I2，M2 设计 §16），仍待负责人决定 |

## 7. 已知限制

- **时区名由宿主决定**（spec 第 3 节第 10 条）：`validTimezone` 先查宿主的时区文件，所以 macOS 上 `asia/shanghai` 这样的写法也能通过，宿主特有的名字（如 `posixrules`）也能通过。部署只有 Linux 容器，影响只在开发库。
- **游标不签名**：改成另一个合格的位置照样可用。列表都按调用者过滤，只能在自己的数据里换起点。总体设计 3.4 写明游标不签名；以后若有列表的游标内容会影响权限，这个列表要自己另加检查。
- **亚微秒的时间**：接口接受纳秒精度的输入，存储和回答截到微秒（F1）。
- **argon2 调参之后的登录耗时**：同 P2 评审第 7 节，仍然成立。
