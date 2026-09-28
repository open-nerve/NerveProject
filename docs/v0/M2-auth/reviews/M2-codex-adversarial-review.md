# M2 账户认证：Codex 对抗性评审

## 1. 评审对象与实际验证

| 项 | 内容 |
|---|---|
| 范围 | `559c3c6..9ef4a4adeee84474f872d6c43ffe4b13dba0aee9`，573 个文件，145,575 行增加、4,452 行删除；包含设计评审、P1–P5、P3 拆分及收尾 |
| HEAD | `9ef4a4a`，`Merge M2 closeout; M2 is done`，与任务预期相同 |
| 日期 | 2026-09-28，Asia/Shanghai |
| 评审者 | Codex（GPT-6），主评审与后端安全、前端会话、范围闭环三个并行子任务；主评审复核证据与定级 |
| 环境 | macOS / Darwin 25.6.0，arm64；系统 Go 1.26.2，`go -C server version` 为自动选择的 Go 1.27.1；Node 24.15.0；pnpm 11.10.0；Docker Engine 29.7.2，Docker Desktop 4.87.0；PostgreSQL 18.6 |
| 工作区 | 开始时 `main` 且干净。主目录没有切换分支、创建 worktree、修改实现；变异在仓库外的临时 clone 中进行 |
| 证据口径 | “已复现”是本轮实际运行；“读代码确认”是源码与测试断言的静态核对；“推断”明确保留不确定性。历史 review 的绿灯与数字不替代本轮结果 |

### 1.1 命令与结果

| 实际命令或实验 | 本轮结果 |
|---|---|
| `git status --short`、`git branch --show-current`、`git rev-parse HEAD`、`git log --first-parent 559c3c6..HEAD`、`git diff --stat 559c3c6 HEAD` | 起点干净，分支与范围符合任务；包含收尾合并 |
| `pnpm install --frozen-lockfile` | 退出 0；使用现有 pnpm，没有执行 `corepack enable` 或全局安装 |
| `make gen-check` | 退出 0；Go、sqlc、bodyshape、打包 OpenAPI 和 TS 生成物无差异 |
| `make lint` | 首次退出 2：评审者误将它与会删除再生成文件的 `gen-check` 并行，读到暂时不存在的 instance/gen。生成结束后顺序重跑退出 0：Go 两模块各 `0 issues.`，关键词 60 规则/3 例外/零命中，前端 54 个任务成功。这是本次实验调度错误，不是仓库缺陷 |
| `make knip` | 退出 0；包含路由 typegen，配置提示也作为错误 |
| `make test` | 通过 server 全部包与嵌套 `server/tools`；使用 `-count=1`，包括真实 Postgres 集成测试 |
| `make test-web`；`TURBO_FORCE=true make test-web` | 均退出 0；后者 16 个任务、0 缓存命中，15.387 秒。六个有测试的包合计 29 个测试文件、282 个测试；web 15 文件/154，utils 5/59，i18n 2/12，constants 1/11，services 1/11，editor 5/35 |
| `make build` | 退出 0；构建真实内嵌前端的 `bin/nerve`，用于后续 HTTP 与浏览器实验 |
| `cd e2e && pnpm exec playwright install chromium` | 退出 0；只安装项目所需的浏览器缓存 |
| `/usr/bin/time -p make e2e` | 48/48 通过，Playwright 13.8 秒，总墙钟 14.81 秒；覆盖 A1–A17、S1–S5 |
| `go -C server test -race -count=1 ./internal/platform/httpserver/... ./internal/platform/ratelimit ./internal/platform/postgres` | 全部通过；没有 race 报告 |
| 后端定向测试、八项实现变异、prod 部署、迁移往返、River SQL 比对 | 见 1.2、1.4；八项变异均被现有测试发现 |
| 架构/契约/关键词的临时违例 | 见 1.3；能拦住的性质与实际盲区分别记录 |
| 三个故事的实现变异 | 见 1.2；A8、A12 被 e2e 发现，A11 的 store 原文保留由单元测试发现 |
| P4、P5 review 附录浏览器核对 | P4 选定的 8 项、P5 全部 6 项在当前构建通过；另加冻结退出、主题停用、PAT 跨账户迟应答、PATCH 到达乱序实验 |

本轮没有因 Docker、网络或缺依赖而无法运行的上述标准命令。没有重跑历史每一次 Phase 的全量 mutation，也没有重建 M1 基线：本轮用当前 HEAD 的对照与单点变异验证现有承诺。未执行真实 Caddy/公网 TLS 部署、长时间/大规模容量测试、真实浏览器后台自然冻结的穷举、全负载 36 秒停机上界测试。前端子任务初轮曾被自动审查中止，提示可能涉及网络安全风险；随后完成普通功能与一致性检查，XSS 部分限于源代码数据流和 CSP/页面运行结果，没有据此声称完成全前端渗透测试。

### 1.2 测试强度：实际变异与恢复

所有变异都在仓库外当前 HEAD 的 clone 中逐个实施，保存单点差异，运行下列既有测试，再恢复原内容（保存原始字节写回或 `git restore -- <文件>`）；新增探针文件用完删除。没有在主目录保留变异。没有把编译失败当成性质被测试守住。

| 变异 | 验证与结果 |
|---|---|
| 刷新旧代不再要求标签成立 | 未变异基线通过；后端既有伪造旧代测试在变异后失败，随后恢复原字节 |
| 续期延长绝对期限 | 既有绝对期限测试失败；见下方精准替换表 |
| 去掉账户凭证行锁 | 真实 PostgreSQL 争锁测试失败，非只靠假哈希器时序 |
| 失败闸门成功不退款 | 既有 gate 退款测试失败 |
| 默认放行未声明的操作 | 既有默认拒绝测试失败 |
| 去掉 API `Cache-Control: no-store` | 改用对应的 `TestAPIResponsesAreNotStored` 后失败；首次选 `TestSecurityHeaders` 只覆盖三个固定安全头，通过不能证明 no-store |
| JWT 验证错误直接返回底层错误，去掉固定拒绝理由 | `TestAccessTokenVerifyGivesOnlyFixedReasons` 多个分支失败；这证明固定理由约束，真实秘密编码扫描另见 1.4 |
| 跳过常见密码名单 | 现有密码策略测试失败 |
| A12：`Deactivate.deactivate` 跳过 `RevokeSessions`，仍停用账户 | `pnpm exec playwright test a12-deactivate --workers=1 --reporter=line`：2 失败、1 通过。页面与 PAT 版本都在共享 `expectAllSessionsRevoked` 断言看到 `revoked=false`、`revoke_reason=null` |
| A8：`UpdateMe.Execute` 调存储前把 `p.FirstName=nil` | 同命令运行 `a8-update-me`：3/3 失败，断言期望 `Ada`、实际空串；页面和 API 不是仅断言 200 |
| A11：把 `ApiTokenStore.createToken` 的白名单投影改为 `const listed: ApiToken = created`，让原文留在 store | `a11-api-tokens`：4/4 **仍通过**；但 `pnpm --filter web exec vitest run core/store/user/api-token.store.test.ts`：5 失败、9 通过。`expectTokenGone` 检查 DOM、表单值、URL、浏览器存储、Cookie、控制台，不检查 JS store；单元测试负责这层。不能把 e2e 通过扩大成“内存无原文”，也不能把此实验说成全部门禁假通过 |
| 恢复后三故事一起重跑 | A8/A11/A12 共 10/10 通过；副本 `git status --porcelain` 无输出 |

后端精准替换如下；命令工作目录为 `server/`，表中省略命令前缀的 `-run` 延续同包 `go test -count=1`。HTTP 包路径是 `./internal/platform/httpserver`，领域包是 `./internal/modules/identity/domain`。

| 性质 | 精准改弱 | 现有测试 / 结果 |
|---|---|---|
| 旧RT需真tag | `domain/session.go`：`token.Generation < s.Generation && tagValid`→`token.Generation < s.Generation` | `go test -count=1 ./internal/modules/identity/domain`；`TestJudgeRefresh` 失败，forged older verdict2 want3 |
| 成功认证退回失败桶 | `httpserver/api.go`：删除成功分支、调用next前的`refund()` | `go test -count=1 ./internal/platform/httpserver -run TestFailureGate`；`TestFailureGateKeepsTheUnitOfAFailedCredentialOnly/valid_credential` 失败，left2 want3 |
| 默认拒绝 | `httpserver/api.go`：`if a.public[r.Pattern]`→`if true` | `-run TestAuthenticationDeniesByDefault`；多分支204 want401/500失败 |
| no-store | `httpserver/middleware.go`：`header.Set("Cache-Control", "no-store")`→`"public"` | `-run TestAPIResponsesAreNotStored`；answer/problem/fallback/panic四分支失败。初次误选 `TestSecurityHeaders` 只测三个固定安全头；已用正确 selector 重做 |
| 常见密码名单 | `domain/password.go`：`case p.isCommon(password, email):`→`case false:` | `-run TestCommonPasswords`；14个常见/简单边缘变形例失败 |
| 会话绝对期限 | 生成的`sessions.sql.go`实际RotateSession SQL：在`generation = generation + 1, token_hash = $2`后加`, expires_at = expires_at + interval '1 day'` | `go test -count=1 ./internal/modules/identity/adapter/postgres -run 'TestRotateSession$'`；真实PG行期限变化被抓，exit1 |
| 账户行锁 | 生成的`users.sql.go`两条SQL删`FOR NO KEY UPDATE` | `-run TestTheCredentialLockBlocksLocksNotInserts -timeout 30s`；第二个锁成功，期望55P03，失败 |
| JWT日志错误固定文本 | `adapter/signing/jwt.go`：`return app.AccessClaims{}, rejection(err)`→直接`err` | `go test -count=1 ./internal/modules/identity/adapter/signing`；`TestAccessTokenVerifyGivesOnlyFixedReasons`多个分支失败 |


### 1.3 架构、契约和关键词门禁的对抗实验

| 临时输入/修改 | 命令与实际结果 | 判断 |
|---|---|---|
| 在 identity/app 加 `import _ "github.com/jackc/pgx/v5"` | `go test -count=1 ./internal/archtest -run TestRepositoryFollowsArchitectureRules` 退出 1，指出 app 纯度规则 | 守住 |
| 在 platform/clock 导入 `internal/shared` | 同测试退出 1，指出 platform 不得依赖 shared | 守住 |
| 新增生成文件引用 `openapi_types.UUID` | `go test -count=1 ./internal/archtest -run TestGeneratedCodeUsesTheStandardUUID` 退出 1，精确定位引用 | 守住，不受 golangci 默认忽略生成文件影响 |
| identity 的 sqlc 查询 `SELECT id FROM river_job LIMIT 1` | `CGO_ENABLED=0 go tool -modfile=tools/go.mod sqlc generate` 退出 1，`relation "river_job" does not exist` | 守住跨模块查询 |
| probe 模块迁移写 `ALTER TABLE users …` | `go test -count=1 ./internal/archtest -run TestSQLCSchemaScope` 退出 1，指出表属于 identity | 守住已支持语法 |
| 同处改成 `ALTER TABLE "users" …`；另一轮写 `DROP TABLE users` | 两轮均退出 0 | 已知正则扫描边界确实存在；不是本轮新发现。M3 接收条件不能当作已经实现 |
| register handler 返回未声明的 `identity.review_undeclared` | `go test -count=1 ./internal/modules/identity/adapter/http` 退出 1，`problem code … is not declared` | `CheckResponse` 正向检查真实生效；`Main` 的声明未覆盖检查也在全包运行中启用 |
| 新 TS 文件写 `CsRf`、`/auth/sign-in/`、`withCredentials: true` | `node tools/keywords.mjs` 三轮退出 1 | 字面规则有效，csrf 大小写有效 |
| 改成字符串拼接；`ERROR_CODE`、`IS_SELF_MANAGED`；`"withCredentials": true`、`withCredentials : true` | 对应各轮退出 0；用户/实例/时区/用户资产旧地址的拼接也通过 | 这是正则字面守卫，不能证明语义上不存在旧调用。格式门禁还可能拦部分非规范写法；本轮没有据此声称整个 CI 被绕过。没有发现当前 M2 可达页面因此实际调用旧地址 |

### 1.4 实际 HTTP、数据库与部署结果

手工业务实验的临时数据库只创建在允许的 `nerve-dev-db-1` 中；需要集群角色的权限实验改用 testcontainers 自己的 Postgres，不修改开发容器的角色。全部请求针对本机本次启动的 nerve。

| 性质 | 本轮证据与边界 |
|---|---|
| JWT | `alg:none`、替换算法/签名/关键头、过期、尚未生效、缺失必要期限均拒绝；签名实现锁定 EdDSA，claims 无权限。`iss/aud` 未作为 v0 契约承诺，不能凭它们缺席报漏洞 |
| 刷新令牌 | 截断、补位、换 base64 字母表、改会话/代数/secret 均拒绝；当前代只改标签可 200，这是以当前 secret 哈希证明持有且允许换签名密钥的既定规则，不是漏洞；伪造旧代 401 后原会话仍能 200；真旧代 401 后会话也 401。双续期 10 轮均 `[200,401]`，随后该会话失效，符合重复使用语义；没有出现两个都成功 |
| PAT | 库存哈希、API 列表无原文；过期/撤销/停用期间 401，恢复后 200，管理员重置后 401；浏览器一次展示和复制/下载路径见 A11 与 P5 核对 |
| 浏览器秘密存储 | `token-manager.ts:25` 的 AuthRecord 只含 RT 和 login_id，access token 在管理器内存；localStorage 中可找到刷新令牌是明确设计，XSS 风险已接受。A11 检查 PAT 不留在 DOM/表单值/URL/存储/Cookie/console；不把这等同于枚举浏览器内部全部历史记录或任意 JS 内存，本轮没有另做完整 history 数据库取证 |
| 登录枚举 | 当前相同 argon2 参数，wrong-password 与 missing-email 各 50 次：中位 14.95/14.96ms，p95 15.36/15.41ms，响应码/体相同；共用安全头映射由读代码确认，本脚本没有保存两组完整头部分布。本机分布重叠；不是对所有机器、网络或历史参数的统计证明 |
| 注册关闭 | prod 对已存在/不存在邮箱均 403 同体；用例先检查开关再查询，结合源码确认不查邮箱 |
| 客户端 IP | prod 未设可信代理，64 个不同伪造 XFF 的坏 bearer 请求得到 60×401、4×429；随后合法凭证也被同一 IP gate 挡为 429，符合已接受共享出口代价。可信代理多跳/坏项/IPv6 前缀由现有定向测试验证，未实际搭建 Caddy |
| 密码边界 | `domain/password_test.go:22` 的空、7/8/128/129 UTF-16 单位、emoji/非 ASCII 字符与常见密码变体测试通过；`domain/password.go:60`、`adapter/argon2/hasher.go:93` 读代码确认不做 NFC/NFD 归一化、不按 NUL 截断，按原密码字节哈希。本轮没有额外运行 NFC/NFD 等价字符串和含 NUL 的完整注册/登录对照，不把静态结论说成 HTTP 实测 |
| 请求体 | 超限、错误类型、未知字段、null、非法格式与边界数值有实测/测试；重复键反例见 Critical 1。非法 UTF-8 `A\xffB` 被接受并转成 `A�B`；携带 JSON 的 `text/plain`、`application/x-www-form-urlencoded` 也被接受。后两项不构成已证明的权限绕过，记录为边界宽松行为；不要把 `json.Valid` 等同于严格字节级 UTF-8 检查 |
| 头部与 CSP | API 404 带 no-store；health/ready/static 带三个固定安全头，静态资源保留自身 cache 策略。实际首页 5 个内联脚本，逐个 SHA-256 均在 CSP 中；页面核对未见 CSP 违规。413/429/panic 500 由真实中间件定向测试覆盖，响应已开始后 panic 的行为是中断连接 |
| 日志 | DEBUG 下扫描 7 个实际测试秘密（含密码、JWT、RT、PAT）的原文、hex、标准 base64、URL base64，匹配数均为 0；安全事件含 session/user/request ID。这个结论限于本次输入和既有日志测试，不扩成“任意调用者主动把秘密放进 URL/自定义头也会自动打码” |
| prod 从零 | 无签名密钥时 `nerve users create` 在配置加载阶段退出 1；临时 PKCS#8 密钥、自建空库、迁移、stdin 创建首账户、登录、`/me`、`/readyz` 成功；访问令牌 900 秒。CLI 密码不通过进程参数 |
| 迁移 | 五份 up → 五次 down → up 全部退出 0；最终五版本 applied。River 0.47.0 的锁定 `migrate-get --line main --all --exclude-version 1 --up/--down` 与 00005 相应 SQL 内容逐字节一致（318/189 行） |
| 后台/停机 | 清理、SKIP LOCKED、启动重试、就绪迁移检查、停机顺序的现有测试通过；空闲 prod SIGTERM 实测退出 0，约 4.68ms。这不能证明满负载上界；分离角色反例见 Important 2 |
| 管理员命令 | A12/A13/A16/A17 及 CLI 测试验证成功、失败退出码、状态与其他账户不受影响。命令直接使用选中的 DB 配置，没有自动识别“管理员本来想操作哪一个库”的机制；未把操作员选错连接地址报作认证漏洞 |

### 1.5 并发证据、配置口径与复现方法


`go test -count=1 ./internal/modules/identity` 和 `go test -race -count=1 ./internal/modules/identity` 均通过。`interleavings_reset_test.go:105` 的`contend`等待`pg_stat_activity`出现锁等待后才开门；`waitLimit=10s`，不是仅靠sleep推断顺序。

| 设计交错 | 实现与判断 |
|---|---|
| 1 登录与reset | `interleavings_reset_test.go:148`、`:170`；reset停在gatedPasswords（`:35`），已有用户锁、尚未执行会自动上锁的UPDATE；login停在gatedSessions（`interleavings_test.go:106`），持锁、尚未INSERT。两个方向都确实在临界区；reset先时旧密码失败，login先时新session被reset撤销 |
| 2 重新hash的登录与reset | `interleavings_reset_test.go:190`、`:213`；同一gatedPasswords，把竞争钉在写新hash之前；确认reset的hash不被覆盖 |
| 3 PAT创建与reset | `interleavings_reset_test.go:232`、`:256`；gatedTokens `:49` 在CredentialLock复核完成后、INSERT前，持用户锁；会话/PAT两种创建者各测两个方向。reset前置阻止创建，reset后置撤销新PAT |
| 4 改密码与登录 | `interleavings_test.go:201` 先把login卡在事务外假Hasher，测快照复核；本身不证明持锁。`:258`补反向，login已锁用户、INSERT前卡住，等改密码真实锁等待，再开门，补齐锁证据 |
| 5 两登录其中重hash | `interleavings_test.go:230` 卡在第一次verify，另一login先提交，再核验重新校验一次、不用旧hash覆盖新hash。本测试目的是快照/重试，闸门有意在锁外；行锁由其余测试和存储测试覆盖 |
| 6 用户锁与外键插入 | `adapter/postgres/credentials_test.go:133`；事务在`:144`真正取FOR NO KEY UPDATE后关在release channel；另事务INSERT session应成功，另一次lock应500ms lock_timeout→55P03。10s上限防挂。删锁mutation确实被捕获 |

配置的优先级由 `server/internal/platform/config/load.go:44` 和已运行的 `TestLoadAppliesLayersInOrder` 对照：内嵌基础 → 内嵌环境 → `NERVE_CONFIG_DIR` 的基础/环境 → 仅 dev 的 local 文件 → `NERVE_…__…` 环境变量。`validate_test.go:65`、`:122` 检查非法键值和跨键约束；`config_test.go:67` 检查私钥路径不进日志。prod 无密钥时 `nerve users create` 在配置加载阶段实际退出 1，非 prod 暴露监听警告仅用代码与已有测试确认，没有为此重建部署。

| 默认值 | 对照与结论 |
|---|---|
| JWT 15m；session 720h；refresh 4s；commit/rollback 2s | `server/configs/config.yaml:37`、`:39`、`:41`、`:31`；前端续期超时 8s，大于 4+2；一致 |
| HTTP header/read/write/request/shutdown | `config.yaml:10` 起分别 5s/30s/60s/15s/20s；API body 1,048,576 bytes；一致 |
| argon2 | 19,456 KiB、2 次、并行度 1、最多 4 并发、等 2s；test 为 64 KiB/1 次；`config.yaml:51`、`config.test.yaml:9` 与真实 config/argon2 测试一致 |
| 七桶（每分钟 / burst） | anonymous 600/100；auth_failure 60/60；authenticated 1200/200；login_ip 30/10；login_ip_email 10/5；register_ip 10/5；password_user 5/5。`config.yaml:63` 起与配置测试一致；test 全为 600000/100000 |
| 其他时限 | session cleanup dev/prod 1h、test 2s；jobs stop 10s 加取消收尾 1s；pool close 5s；IPv6 /64。`config.yaml:43`、`:61`、`:79`，`bootstrap/app.go:40`；一致 |

独立复现无需保存本次临时凭证：先 `make build`，在允许的开发容器用 `createdb -U nerve <独有临时库>` 建库；以 `NERVE_ENV=test`、`NERVE_DATABASE__URL` 指向它执行 `bin/nerve migrate up`，再以 `NERVE_SERVER__ADDR=127.0.0.1:0`、`NERVE_SERVER__ADDR_FILE=<外部临时路径>` 启动 `bin/nerve serve`。注册随机账户取得当前凭证，所有请求和查询只打这个库；结束时 SIGTERM 自己的 PID、等待退出，再 `dropdb` 自己的库。prod 实验另加临时 PKCS#8 Ed25519 私钥并明确 `NERVE_ENV=prod`。

- **Critical 1 复现**：对同一账户分别发送发现正文的两个原始 JSON 字符串，`Content-Type: application/json` 和相同 Bearer；先看到 400，再看到 200；查询其 `profiles.onboarding_step` 或重新 GET `/api/v0/me/profile`，确认 `workspace_invite=true`。不能先让客户端 JSON.stringify 一个 JS 对象，因为那会在请求发出前丢掉重复键。
- **Important 1 复现**：用项目 TypeScript `transpileModule` 转译原样的 `token-manager.ts`/`refresh-lock.ts`，Playwright `context.route` 提供同源 harness；每页接真实 localStorage、Date.now、leaseLock，POST 适配器发送真实 HTTP，storage 事件调用 `handleStorageChange()`。注册 X/Y，A `signIn(X)`，B start 跟随 X。拦 A logout，先 `await route.fetch()` 得真实 204、暂缓 fulfill；A `signOut()` 已到此处后以 CDP `Debugger.enable`、`Debugger.pause` 停住 A。B `signIn(Y)`，不手工改租约，让真实 10 秒租期结束。释放 A 的响应、`Debugger.resume`；两页 signed-out 且 auth 记录为空，Y 原 JWT 的 `/me` 仍 200。最后关 browser。该 harness 原样使用生产 TokenManager/leaseLock，以最小页面与 POST 适配器接真实后端，并控制调度。
- **Important 2 复现**：在专用 testcontainer 先用迁移 owner 建表，建普通 LOGIN runtime 角色；给它 public schema USAGE、四业务表 SELECT/INSERT/UPDATE/DELETE、goose_db_version SELECT；`auto_migrate=false` 用该角色启动。ready=200、River=42501。再授四 River 表的 DML 及所需 sequence USAGE/SELECT，任务可运行；以 runtime 执行 `REINDEX INDEX CONCURRENTLY river_job_pkey` 仍 42501，授 river_job MAINTAIN 后成功。本次只为实验授 `ALL SEQUENCES IN SCHEMA public`，生产建议应收窄为实际 River sequences。

P4 从现有 review 附录提取脚本执行 `unused C3 C4a C4b C4c C6c C8 C9`（另总是执行 C10）；P5 执行 `C1-four-tabs C2-dropdowns-beside C6-maintenance` 和 `C3-token-once C4-security C5-deactivate`，共 6 项。P4 未重跑 C1/C2（与现有故事重叠）、C5（真实 LAN HTTP）、C6a/C6b（服务/数据库真正停止恢复）、C7（旧维护文字）、C11（Caddy）。CLI 真 TTY 隐藏回显、实际午夜 River 中断、跨列表游标运行（当前只有 PAT 列表）、Firefox/WebKit、移动设备自然挂起未验证。额外确定性“改密码与新 PAT 创建”双向组合没有重跑；本轮依赖共同 CredentialLock、reset/PAT 交错和存储行锁测试，不能声称穷举全部排列。

七份 plan 含数万行预写实现，本轮按任务结构、全局约束、裁定/偏离/交接/完成线及对应代码块审查，没有逐字复读每段预写源码。主要设计、spec/review、十份入向、六份出向及下方逐条条款均已建立现时代码映射；被标为“读代码确认”的范围不冒充独立运行。

## 2. 总体结论

**修完列出的 Critical 后可以开始 M3。** 本轮列出 **1 个 Critical、2 个 Important、3 个 Minor**。Critical 1 是请求体校验器与最终解码器对重复键的解释不同：不只是“接受一个多余字段”，而是原本 400 的字段在重复对象后实际写入数据库。按本次任务对 Critical 的定义，这是后续所有模块会复用的错误平台保证；没有证据证明它在当前 M2 已能越权访问其他账户。

认证的主要凭证机制经实测与变异仍站得住：伪造旧刷新令牌没有撤销能力，真旧代触发撤销，绝对期限、账户锁、默认拒绝和退款测试会因对应实现变坏而失败。全量门禁与 48 个 e2e 通过不等于所有边界成立：冻结租约下退出的会话一致性、分离角色下 River 的运维权限仍有具体缺口。无需推翻 M2 的模块分层、令牌模型或每会话一个 RootStore；修复应局限在发现的根因，并给出能失败的回归证据。

已知接受事项单独复核，不冒充新发现。特别是“服务器按请求发出顺序应用 PATCH”的假定已被当前页面实验推翻，应在 M3 复用该模式前改正；它早已被 P5 明示接受，因此不计入上述数量。

## 3. 各领域评估

| 领域 | 评分（1–5） | 主要依据 | 最大风险 |
|---|---:|---|---|
| 范围闭环 | 4 | 故事、Phase、入向/出向交接均能追到代码与断言；存在断言复用及文档漂移 | 把已勾选等同绝对保证 |
| 认证与会话的安全 | 4 | JWT/RT/PAT、撤销与并发实测；八项变异被抓住 | 结构校验差异，及已接受的 PAT 持久化派生能力 |
| 前端令牌管理器与会话 | 3 | 正常双 tab 两种锁、重试、RootStore 分代与迟应答保护有效 | 旧退出清新会话；PATCH 顺序假定 |
| 后端架构与契约 | 3 | 分层、UUID、sqlc 跨表、未声明问题码门禁有反例验证 | Critical 1；sqlc 语法扫描不是完整所有权证明 |
| 数据库、迁移与后台任务 | 4 | 表/约束、迁移往返、River 锁定 SQL、清理和锁测试 | 分离运行角色漏 River 权限 |
| 质量门禁与测试 | 4 | 全量通过；变异区分 e2e 与单元测试职责 | 部分验收文字比实际断言范围宽 |
| 运维、配置与文档 | 3 | prod 首账户流程、默认值、密钥与停机说明可运行 | `/readyz` 200 不代表后台任务已能运行 |
| 代码质量 | 4 | 用例职责、消费方端口、生成类型、错误边界较清楚 | 局部全局主题补丁与错误的并发假定容易被后续照抄 |

## 4. 发现列表

### Critical 1：重复 JSON 对象键绕过结构检查，禁止字段仍可被最终解码并写库

- **类别**：契约 / 架构。
- **位置**：`server/internal/platform/httpserver/bodyshape/bodyshape.go:153`、`:189`；`server/internal/platform/httpserver/bodyshape/middleware.go:29`；`server/internal/modules/identity/adapter/http/gen/server.gen.go:1767`；`server/internal/modules/identity/domain/profile.go:69`；要求见 `docs/v0/M2-auth/M2-design.md:497`（3.11）。
- **可信度**：已复现；主评审与后端子任务分别用真实服务复现，主评审另用数据库直接查询复核。
- **证据**：检查器把对象放进 map，重复键只留下最后一项；随后把未经重写的原始 body 交给生成 handler，Go 的 struct 解码会合并重复对象、宽松匹配大小写。领域层明确相信结构已经检查。

  ```go
  var props map[string]json.RawMessage
  _ = json.Unmarshal(raw, &props)
  for name, value := range props {
      child, declared := n.Props[name]
      // …只检查 map 留下的最后一个值
  }
  ```

  对当前账户 `PATCH /api/v0/me/profile`：`{"onboarding_step":{"WORKSPACE_INVITE":true}}` 得 400 `not_allowed`；改为 `{"onboarding_step":{"WORKSPACE_INVITE":true},"onboarding_step":{}}` 得 200，响应及数据库的 `workspace_invite` 都变成 true。首对象中的 `UNKNOWN`、不允许的 `profile_complete:null` 也能通过追加重复空对象，从 400 变成 200。全量现有测试仍通过，未覆盖这一解析差异。
- **影响**：当前复现改变的是调用者自己的资料，没有证明越权。平台“未知字段/null 等在边界统一拒绝”的保证不成立，M3–M8 依赖该保证会继续放大差异，因此按任务的地基标准列 Critical。
- **处理建议（权衡）**：① **推荐，M3 开始前**在原始 JSON 结构遍历阶段统一拒绝重复对象键，按解码后的键名判重（含转义等价键），然后才做 schema 检查；补顶层/嵌套、大小写、null、转义键的真实 handler 回归。改动集中，不改生成代码，不引入第二份契约；顺带明确非法 UTF-8 的拒绝策略。② 统一解析为中间树并只把规范化后的同一份数据交给 handler：能统一语义，但引入重编码与更大维护面，没必要为当前问题扩大实现。③ 不处理：当前未见跨户权限后果，但后续领域层不能再信任结构检查，等于把同一防线复制到每个用例；不接受。

### Important 1：租约过期后，旧标签页的退出响应会删除另一标签页的新登录

- **类别**：会话 / Bug。
- **位置**：`web/apps/web/core/lib/auth/token-manager.ts:150`，尤其 `:158–160`；`web/apps/web/core/lib/auth/refresh-lock.ts:28`、`:83`；refresh 路径 `web/apps/web/core/lib/auth/token-manager.ts:242` 已有响应后的记录复核。
- **可信度**：已复现；真实 Chromium 两标签页、真实后端，未修改生产 TokenManager/leaseLock，外部 harness 转译原源码并强制调度暂停。
- **证据**：A 登录 X 后开始退出，后端已回 204，在前端消费结果前暂停 A 的 JS；超过 10 秒后 B 经实际 leaseLock 登录 Y；恢复 A 后两页均变为 signed-out，`nerve.auth=null`，Y 的凭证直接调用后端仍 200。代码在 await 前检查归属，await 后无条件删除：

  ```ts
  await this.#call("/api/v0/auth/logout", record.refresh_token);
  this.deps.storage.removeItem(AUTH_KEY);
  this.#signedOut();
  ```

  调度使用 CDP `Debugger.pause`；另一次 `cdp.send("Page.setWebLifecycleState", { state: "frozen" })` 没有阻止 8 秒定时器，未复现。不能据此声称所有自然冻结都触发。既有接受的是租约非原子导致同一会话重复续期；此处是旧操作清除**另一个 login_id**，不是同一个已知问题。
- **影响**：没有以 Y 身份执行后端 logout，也没有跨户数据写入；Y 的新登录被客户端丢弃，用户被迫再次登录。破坏 TokenManager 自身“操作只作用于发起会话”的约定。
- **处理建议（权衡）**：① **推荐，并入 M3 的第一批认证回归，接新 store 前处理**：logout await 后重新读取记录，只有同一 login_id 才删除，否则跟随新记录；用可控暂停/租期的测试覆盖。与 refresh 已有协议一致，改动小。② 去掉 HTTP 下的租约降级、要求 Web Locks：减少这类并发，但改变已批准的 HTTP 使用范围，代价过大。③ 不处理：触发条件限于租约失效或并发取得，本轮未测自然发生率；每次触发直接丢新登录，且已有同型简单机制可复用，不值得保留。

### Important 2：分离迁移和服务角色的部署说明遗漏 River 的运行及维护权限

- **类别**：运维 / 任务 / 文档。
- **位置**：`README.md:123`、`:125`；`server/internal/platform/jobs/jobs.go:81`、`:124`；`server/migrations/sql/00005_river_main_v2_to_v7.sql:1`。
- **可信度**：已复现；权限实验只在本次 testcontainer 内创建角色，没有修改共享开发库的角色。
- **证据**：迁移 owner 建表，服务角色仅有 schema USAGE、四张业务表 DML、`goose_db_version` SELECT 时，`/readyz=200`，River 报 `permission denied for table river_leader/river_queue (SQLSTATE 42501)` 并重试。补 River 表 DML 和 sequence 权限后，还缺每日 `REINDEX INDEX CONCURRENTLY` 所需权限；实测报 `permission denied for index river_job_pkey (SQLSTATE 42501)`。`GRANT MAINTAIN ON river_job TO <runtime_role>` 后同一 REINDEX 成功，后台清理完成 1 次。README 只特别列出 goose 权限，但同时启用了 River 启动和每日重建。
- **影响**：采用分离角色的部署可能只有 API 看似正常，清理任务持续不起；日后 M4 业务任务沿用会造成更实际的积压。`/readyz` 当前仅承诺数据库/迁移状态，本报告不把它冒称为已有的 jobs 健康检查。
- **处理建议（权衡）**：① **推荐，M4 第一条业务 River 任务前**补可运行的最小授权示例，明确业务表、River 表、sequence、Postgres 18 维护权限，并用真实分离角色跑“启动→任务执行→维护”smoke；变更小，不引进权限框架。② 服务进程继续用迁移 owner：最简单且本轮正常部署通过，但放大运行账户的数据库权限，不应当作默认补救。③ 不处理、留到 M8：默认单角色暂时无事，已有分离角色用户会持续丢后台能力，M4 排障成本更高，不推荐。

### Minor 1：当前标签页自助停用后没有实际恢复默认主题

- **类别**：Bug / 会话。
- **位置**：`web/apps/web/core/lib/store-context.tsx:47`、`web/apps/web/core/lib/wrappers/store-wrapper.tsx:55`；停用入口 `web/apps/web/core/components/account/deactivate-account-modal.tsx:41`。
- **可信度**：已复现；系统浅色、账户暗色，走真实设置页自助停用。
- **证据**：回登录页后 localStorage.theme 已是 `system`，但 DOM `data-theme` 仍是 `dark`；整页 reload 后才变 light。`startSession` 只写 localStorage，当前标签页不会收到自己的 storage 事件；StoreWrapper 没有 profile 时直接跳过设主题。普通退出入口单独调用 setTheme，作为对照能够恢复。
- **影响**：页面沿用上一会话外观，没有数据或权限串户；`docs/v0/M2-auth/specs/P4-web-auth.md:163` 和 `store-context.tsx:41` 的默认主题规则没有在所有退出路径落实。
- **处理建议（权衡）**：① **推荐并入 M3**，在统一的会话/页面包装层按会话变化真正调用 setTheme，移除依赖特定退出按钮的补丁；补停用与续期 401 结束两条回归。② 每个退出入口分别补 setTheme，改得快但继续复制规则，容易漏下一入口。③ 不处理，刷新可恢复、影响轻；可短期接受，不能继续声称所有换代都立即恢复默认主题。

### Minor 2：A8–A10 共享查询函数，却没有按设计共享业务数据库断言

- **类别**：测试 / 闭环。
- **位置**：`docs/v0/M2-auth/M2-design.md:77`、`:2121`；`e2e/stories/identity/a8-update-me.spec.ts:53`、`:127`，`a9-preferences.spec.ts:101`、`:277`，`a10-onboarding-profile.spec.ts:25`、`:120`、`:137`。
- **可信度**：读代码确认；A8 变异另证明两份断言确实都会失败。
- **证据**：A8 共用 `accountOf` 的 SELECT，页面和 API 分别复制变更内容/updated_at 的 expect；A9 同样分别 expect profileOf；A10 页面 helper 与 API 单独断言，引导步骤初始差异未用参数表达。A7 的 `expectPasswordChanged` 才是要求的正例，页面保留的会话由参数表达。
- **影响**：不是没有数据库断言，也不是 PAT 功能不对等；新增字段/副作用时容易只改其中一份，§14 “同一组断言”的完成声明过满。
- **处理建议（权衡）**：① **推荐并入 M3 开始时的 e2e fixture 整理**，提取三个小业务断言、参数化期望值和初始步骤；不建通用测试框架。② 修改完成标准允许共享读取但重复 expect：改动少，但放弃负责人已选的对等约束。③ 不处理：当前结果仍受检查，短期风险有限，但长期漂移会随 M3 增大，不推荐作为后续模板。

### Minor 3：收尾后仍有两处可核实的文档事实漂移

- **类别**：文档。
- **位置**：`docs/v0/M2-auth/M2-design.md:1823`、`:2311`；`docs/v0/M2-auth/specs/closeout.md:577`。
- **可信度**：读代码确认，实际文件/操作计数确认。
- **证据**：11.2 仍称 bodyshape 运行时用“标准库和 oapi-codegen 的运行时类型”，最终包仅导入标准库，F1 同文已写明。closeout 说 identity.yaml 623 行、15 操作、约 40 行/操作；实际 identity 是 13 个操作，另 2 个属于 instance，623/13 约 48。
- **影响**：后续模块照抄架构说明、估算规模时口径错误；没有运行影响。历史 Phase 的测试数量若有明确提交/时点，不因 HEAD 增加测试而算错误。
- **处理建议（权衡）**：① **推荐并入 M3 文档同步**，直接修两处事实，保留历史 plan 的原样记录。② 给所有统计加自动生成脚本，能防部分漂移但为两个值增加长期维护，不值得。③ 不处理，影响很低，但既已定位，修正文案的代价小于继续解释冲突。


## 5. 范围与闭环矩阵

路径均相对于仓库根目录 `/Users/xiaoruan/project/nerve-project`，行号对应 `9ef4a4a`。统一缩写：`D`＝`docs/v0/M2-auth/M2-design.md`；`V`＝`docs/v0/v0-design.md`；`I`＝`server/internal/modules/identity/`；`H`＝`server/internal/platform/httpserver/`；`W`＝`web/apps/web/core/`；`E`＝`e2e/stories/identity/`；`Q`＝`server/internal/modules/identity/adapter/postgres/queries/`；`A`＝`e2e/fixtures/assert/identity.ts`。缩写后的路径直接拼接，不是另一个目录。

下文使用总报告编号：**Critical 1**＝重复 JSON 键绕过结构检查；**Important 1**＝旧标签页退出误删新会话；**Important 2**＝River 分离角色部署权限不完整；**Minor 1**＝自助停用后主题未复位；**Minor 2**＝A8–A10 未共用业务数据库断言；**Minor 3**＝bodyshape 依赖及 identity 操作数的文档漂移。

“成立”仅覆盖该行的明确要求；“部分成立”表示实现或证明仍有列出的缺口；“不成立”表示绝对保证已被反例否定。命令门禁通过不替代行为证明，门禁实测详见第1节。

### 5.1 0.1 五个目标和 0.2 八行范围

| 要求 | 结论 | 代码/测试证据与限制 |
| --- | --- | --- |
| 0.1-1 全认证、账户、偏好、PAT 能力及 PAT 对等 | 部分成立 | `api/modules/identity.yaml` 13 操作；A1–A17 覆盖下表；正常行为成立，新增重复键结构校验绕过使“所有合约禁止输入都被拒绝”不成立。 |
| 0.1-2 四表、sqlc、River 与平台约定 | 部分成立 | `server/migrations/sql/00001…00005`；`Q/sessions.sql:64` SKIP LOCKED；`I/app/cleanup_sessions.go:10` 1000；`server/internal/bootstrap/contract_test.go:25、42、67、93`；Critical 1，sqlc 正则已知边界另列。 Important 2：按 README:123 配置业务表 DML 和 goose SELECT 时 readyz 为 200，但 River 的 queue/leader 操作报42501；补 DML/sequence 后 REINDEX 仍缺 MAINTAIN。 |
| 0.1-3 令牌管理器、生成类型、不设转换层 | 部分成立 | `W/lib/auth/token-manager.ts:69`；`W/lib/auth/api-client.ts`；user/profile/PAT stores 直接持生成类型；Important 1 已证明冻结租约后旧退出误删新会话，不是类型接入缺失。 Minor 1：自助停用后登录页保留旧 dark DOM，localStorage.theme 已为 system；刷新才复位。 |
| 0.1-4 每故事页面/数据库/API PAT | 部分成立 | 17 文件、48 e2e 当前通过；A8–A10 共享的是读取，不是业务断言（Minor 2）；CLI/任务四故事按设计无页面。 |
| 0.1-5 全门禁、入向关闭、出向接收条件 | 部分成立 | 十份入向 status closed，M3–M8 50 个实质交接小节均有关闭条件；当前门禁通过仍未覆盖Critical 1和Important 1/2，§14 勾选应收回相应绝对保证。 Important 2 说明原部署角色交接仍欠 River 的运行和维护权限验收。 |
| 0.2-1 auth 范围 / workspace-invite 留 M3 | 成立 | `I/app/{register,login,refresh,logout,change_password,deactivate}.go`；onboarding 挂载不取 M3，提交创建工作区仍旧接口是明确范围边界，M3 交接§2。 |
| 0.2-2 /me、profile / assets 留 M5 | 成立 | `I/adapter/http/profile.go`；生成 User 的 avatar/cover 响应 null；A8/A9/A10；M5 交接§1 规定加列、FK与上传控件。 |
| 0.2-3 PAT / 权限、唯一管理员留 M3 | 成立 | `I/app/{create,list,revoke}_api_token.go`；`I/app/deactivate.go:42`；M3 交接§6 明确同事务终端检查与锁顺序，不在 M2 偷加权限模型。 |
| 0.2-4 instance/timezones / audit 留 M8 | 成立 | `api/modules/instance.yaml` 两操作；S3 七字段；M8 交接§1 说明 middleware 位置与所有写操作覆盖的关系。 |
| 0.2-5 五 CLI / 不做邮件、自助邮箱修改 | 成立 | `I/admin.go:40`；A13/A16/A17 和 A12 CLI；合约无邮箱自助 PATCH；README:113–120 解释恢复语义。 |
| 0.2-6 River、sqlc / 无 instance_admin | 成立 | `server/internal/platform/jobs/jobs.go`；五份迁移，无实例管理员表、操作；S1 核对迁移与 River 表。 |
| 0.2-7 六类前端页面 / Cookie 仅升级路径 | 部分成立 | A1–A12 页面覆盖；CSRF/withCredentials 守卫当前零命中；Important 1 影响令牌管理器保证，未发现 Cookie 回退。 |
| 0.2-8 e2e fixture / clock 留 M4 | 部分成立 | `e2e/fixtures/auth.ts`、`assert/identity.ts`；M4 交接§13 有 clock.ts 接收条件；A8–A10 共断言缺口。 |

### 5.2 故事逐项矩阵（页面 / 数据库 / 接口与共用函数）

页面栏的行号是测试入口，后续列给业务断言而非只给标题。认证本身 A1–A6/A15 的 API 版不使用 PAT，符合 D:76；A13/A14/A16/A17 无页面符合 D:78。

| 故事 | 判定 | 页面证据 | DB 业务断言 / API 证据 | 共用函数与结论 |
| --- | --- | --- | --- | --- |
| A1 | 成立 | `E/a1-sign-up.spec.ts:9` 注册、onboarding、Cookie/record | 同文件:44 API；A:83 session generation/hash/expiry/UA-IP，:102 account/profile defaults | 页面:41/API:52 `expectRegistered`。 |
| A2 | 成立 | `E/a2-sign-up-refused.spec.ts:12` 409、弱/常见密码、关闭注册 | API:103；两端:74/:132 `expectNothingAdded`；A:420 比账户/资料/会话集合 | prod 默认另由 `server/configs/embed_test.go:108` 验证，e2e 只测覆盖值。 |
| A3 | 成立 | `E/a3-sign-in.spec.ts:17` next_path、错误提示；:69 错密码 | API:90；A:152 新会话；页面:42/API:102 `expectSignedIn`，:87/:127 `expectNothingAdded` | 关闭路径只允许站内，query/hash 断言存在；/create-workspace 挂载守卫随收尾补。 |
| A4 | 成立 | `E/a4-refresh.spec.ts` 两种锁模式；:68 挂住第一续期 | 页面:149/API:181 `expectRefreshed`；A:166 generation、hash、last_refreshed、绝对 expiry | 于所测正常并发；API:158 用每次新令牌依次三轮；并非冻结超过租约测试，不能证明所有 lease 情况。 |
| A5 | 成立 | `E/a5-refresh-reuse.spec.ts:14` 模拟盗用后跳登录 | API:50；页面:34/API:69 `expectRevoked`；真旧代撤销、伪造旧代不撤销 | session_id 可见没有被误当作能力。 |
| A6 | 部分成立 | `E/a6-sign-out.spec.ts:56、91、127` 跨 tab 退出/直接换/退出再换 | API:163；:81/:147/:181 `expectRevoked`；旧代 logout 204 不改变 | 既有测试路径真覆盖，Important 1表明退出仍会清掉新 login_id。 见 Important 1。 |
| A7 | 成立 | `E/a7-change-password.spec.ts:17` 当前密码/规则失败、改后仍登录、PAT说明 | API:81；页面:65 传 held，API:104 不传；A:283 改hash、指定保留session、其余 reason=password_changed、PAT完整不变 | 共函数差异参数正是设计要求；API:109/:111 旧密码401、新密码200。 |
| A8 | 部分成立 | `E/a8-update-me.spec.ts:15、75` 改名、时区、刷新、拒绝输入不改库 | API:114；页面:53–55、API:127–129 独立断言内容与 updated_at | 实际行为与 PAT 流程成立，共断言要求不成立（Minor 2）。本次评审忽略 first_name 的变异使3测试失败。 见 Minor 2。 |
| A9 | 部分成立 | `E/a9-preferences.spec.ts:15、128、193` 中英定位、拒绝、换账户迟应答 | API:265；页面:101/API:277 对 profileOf 查询结果重复断言 | 功能覆盖充足，共断言要求不成立。 见 Minor 2。 |
| A10 | 部分成立 | `E/a10-onboarding-profile.spec.ts:42、56、93` 挂载静默、拒绝、注册内导航 | API:111；页面:25–31，API:120/:137 独立断言；API额外预置 workspace 步骤验证merge；:145未知键400 | 普通拼错键拒绝成立，Critical 1绕过；同函数/参数未落实。 见 Critical 1、Minor 2。 |
| A11 | 成立 | `E/a11-api-tokens.spec.ts:24、123、255` 一次展示、分页、复制失败/成功、会话变更 | API:285；:65/:296 `expectTokenStored`；A:378 hash、expiry、整行无原文；撤销 deleted_at 与401、过期401 | 正常功能范围；本次评审“将原文存整个 store”变异e2e全过但5单测失败，因此是e2e存储覆盖限制，不能称全门禁假通过。 |
| A12 | 成立 | `E/a12-deactivate.spec.ts:29、79` 自助停用、拒绝后的状态 | API:143；:57/:159/:180 `expectDeactivated`；A:307 active、password不变、sessions、profile reset、PAT不删；activate再可用 | 本次评审跳过撤销变异两测试失败；CLI停用同断言。 |
| A13 | 成立 | 无页面（设计规定） | `E/a13-reset-password.spec.ts:11、28`；A:344 hash变化、会话全撤销、PAT全撤销；API新密码可登录、旧凭证401、另一账户不变 |  |
| A14 | 成立 | 无页面/API（后台任务） | `E/a14-session-cleanup.spec.ts:9` 改expires过去，限时poll行消失，活会话仍在 | 不是sleep冒充结果，数据库定时结果真实检查。 |
| A15 | 成立 | `E/a15-sign-in-limits.spec.ts:17` 两种桶提示 | API:45；:42/:73 `expectNothingAdded`；429/Retry-After，email桶拒绝不白扣IP |  |
| A16 | 成立 | 无页面（CLI） | `E/a16-set-email.spec.ts:9、37`；A:360 email规范化、全session reason=email_changed、PAT不变；API旧邮箱401、新邮箱200、PAT/me新email |  |
| A17 | 成立 | 无页面（CLI） | `E/a17-create-user.spec.ts:10、26、31`；`expectNothingAdded`、`expectCreated`；A:146 默认account/profile、无session；API可登录 |  |
| S1 | 成立 | 不适用 | `e2e/stories/smoke/s1-server-ready.spec.ts:12` health/ready、迁移版本及River表的实际query |  |
| S2 | 成立 | `e2e/stories/smoke/s2-web-app.spec.ts:31、79、91` 登录页/未登录 | :67 静默检查只instance请求、无oldAPI、无pageerror/CSP | 覆盖挂载不是未来所有延时/用户动作。 |
| S3 | 成立 | 不适用 | `e2e/stories/smoke/s3-instance-info.spec.ts:3` 七个字段、signup/workspace creation=true,file_size_limit=5242880 | prod signup默认由configtest，不混用环境。 |
| S4/S5（§14额外） | 成立 | s4未知API；s5维护页 | `s4-unknown-api.spec.ts:5` 404problem；`s5-maintenance.spec.ts:7` 文案与可达行为 |  |

### 5.3 3.1–3.19 技术裁定

| 项 | 结论 | 当前实现/断言与边界 |
| --- | --- | --- |
| 3.1 M2能到达页面边界 | 成立 | A3补/create-workspace；A10 `watch.oldApiRequests/pageErrors/apiFailures`；S2登录页；A7–A12设置页；M3交接§2明确提交后的404归M3。 |
| 3.2 提前的nullable字段、无上传 | 成立 | 合约User保留可空引用，adapter输出null；A8/A10没有上传流程；M3交接§4、M5§1规定FK和URL接入。 |
| 3.3 app ports、shared、模块入口 | 成立 | `I/app/ports.go`、`I/module.go:81、160、165、170、175`、`I/admin.go:40`；archtest rules:35–47/193/216/229；本次评审非法app→pgx、platform→shared变异均失败。 |
| 3.4 token布局与签名 | 成立 | `I/adapter/signing/jwt.go:30、47、59、83`；`signing/keys.go:17、52` HKDF；A:20解析68bytes；JWT无权限；PAT tokenhash与one-time返回A11。 |
| 3.5 轮换、绝对期限、锁、真假旧代 | 成立 | `I/app/refresh.go:49` + `Q/sessions.sql:27–31` 条件轮换无expires赋值；`I/app/credential_lock.go:29`、users.sql:32/62 NO KEY UPDATE；真实交错 `interleavings_test.go:201、230、258` / `interleavings_reset_test.go:148、170、190、213、232、256`，以及 `adapter/postgres/credentials_test.go:133` 的锁与外键插入检查。 |
| 3.6 默认拒绝、先失败闸门、事务commit | 成立 | `bootstrap/contract_test.go:25、42、67` 公开集合、注册集合、默认拒绝；`H/api_auth_test.go`、`api_ratelimit_test.go`；`platform/postgres/tx.go:60、92` WithoutCancel带期限。 |
| 3.7 密钥输入与日志 | 成立 | `I/adapter/signing/keys.go:28、46、52`；config LogValue *_file_set；`server/configs/embed_test.go:108` prod不能漏key。 |
| 3.8 密码 | 成立 | `I/domain/password.go:34、46、59、83` UTF16计长、ASCII组成规则及Unicode字母主干；blocklist数据33887/277141；密码hash实现并发4、等2s；A2常见Password1!与变体。 |
| 3.9 防枚举顺序与耗时 | 成立 | 于声明的参数前提。`I/app/register.go:55`先SignupPolicy；`login.go:64、134`假hash；现存旧参数账户可测时差已公开接受；P3b补不可用hash统一耗时。 |
| 3.10 七桶/IPv6/代理 | 成立 | `H/clientip.go`/tests；`platform/ratelimit/ratelimit_test.go:96、163、193、213`原子多桶/退款/并发/清理；A15，prod/dev与test配置分开核对。 |
| 3.11 结构400、值422与errors | 部分成立 | `H/bodyshape/bodyshape.go:153、176`、bootstrap contract:93/137；普通未知/null/缺失/格式均有断言；重复JSONkey使map校验与struct解码不一致的Critical 1 直接破坏保证；有CheckResponse未声明码mutation证据。 |
| 3.12 generation、nullable、uuid、cursor | 成立 | `server/internal/shared/cursor.go:21、41` canonical封套；PAT payload归domain；`archtest/generated_test.go:23、48、72`；本次评审runtime.UUID变异失败；date/x-go-type不支持是build时拒绝并交M4。 |
| 3.13 schema约定/clock/check/FK | 成立 | `migrations/schema_test.go:57、99、164、194–211` up-down-up/命名/CHECK反例；Q四表显式now；users10/profile11/session12/token12；删除关系图向M3/4/6/7移交。 |
| 3.14 sqlc模块边界、ALTER归属 | 部分成立 | `server/sqlc.yaml`、`archtest/sqlc_test.go:18、113`；本次评审river_job跨模块和无引号ALTER被拒，quoted ALTER/DROP绕过是既有正则边界，M3接收前必须加固；onboarding SQL合并（Q/profiles.sql:20）正确避免不同键丢更新。 |
| 3.15 River | 部分成立 | goose00005锁定River0.47.0主线2–7，未加river_migration；SKIP LOCKED1000；service client/jobs启动停机；only-insert由负责人明确延至M4，M4§1若无使用者须再转交；DB恢复实测留M8。 Important 2：River DML/sequence 与 REINDEX 的 MAINTAIN 权限未被 README 分离角色说明和现有 readiness 覆盖。 |
| 3.16 无实例管理员 | 成立 | schema/operation集没有实例管理员；CLI是本机数据库管理，不生成实例管理员角色。 |
| 3.17 五CLI的区别 | 成立 | `I/admin.go:40`无HTTP/River；create/reset/set-email/deactivate/activate用例；A12/A13/A16/A17真实DB+凭证断言。 |
| 3.18 next_path只前端 | 成立 | A3恶意next_path矩阵；URL没进登录契约或服务端redirect，M1-P4的服务端校验前提已撤销。 |
| 3.19 删除role/usecase/is_self_managed | 成立 | keywords有is-self-managed规则且零命中；profile11列；onboarding保留资料/workspace步骤不再角色用途；M1-P3关闭条逐项证据。 |

### 5.4 四个负责人决策和11.1

| 决定 | 结论 | 实现与可验收证据 |
| --- | --- | --- |
| 1 B：仅管理员改邮箱 | 成立 | `I/app/set_email.go:53`先规范化/锁/更新/撤销session，无PAT撤销；A16:37完整DB断言；README:116、120明确不是恢复方案。 |
| 2 C：prod默认关、dev/test开、CLI建首账户 | 成立 | `server/configs/embed_test.go:34、108`；A2独立signup关闭实例；A17CLI无session；S3 test默认true。 |
| 3 A：任何凭证可停用、PAT保留、activate恢复 | 成立 | `I/app/deactivate.go:42、75、101`；`activate.go:45`；A12同DB断言并恢复原PAT；不是唯一管理员检查，后者M3。 |
| 4 A：删角色用途两步/字段 | 成立 | M2字段/列与守卫；A10直接资料→workspace；不是隐藏废组件。 |
| 11.1 退出仅当前会话 | 部分成立 | backend `I/app/logout.go:27`/sessions.sql:49 当前id+generation+hash；A6正常语义成立；Important 1 已证明浏览器冻结租约后能清掉别的会话，不改变已获批后端语义。 |

### 5.5 第12节各Phase交付物

每行数字对应D该Phase的编号，不用“有spec/plan/review”替代代码验收。所有7组spec/plan/review文件均存在；plan是当时实施蓝图，task残留unchecked并不等同现时代码未做。

| Phase/交付物 | 结论 | 证据 |
| --- | --- | --- |
| P1-1 三表/sqlc/schema规则/arch扩展 | 成立 | migrations00001–00003；schema_test:57/164；sqlc_test:18；rules_test:40/229。 |
| P1-2 shared/clock/tx/http/bodyshape/tool CI/config | 部分成立 | shared/cursor后P3补；postgres/tx.go:60/92；H/api.go/bodyshape:153/176；Makefile工具模块接线；Critical 1。 |
| P1-3 合约模板/security/x-problem-codes/CheckRequest | 部分成立 | bootstrap四测试、apitest规则；未知问题码变异失败；CheckRequest与实际decoder的重复key语义不一致。 |
| P1-4 register/getMe/password/signing/authn | 成立 | app/register:55/get_me/authenticate；A1/A2；signing tests:106/123/140/240/261。 |
| P1-5 四整程序测试 | 成立 | 覆盖有限。bootstrap/contract_test.go:25、42、67、93；后来参数137、headers175。 |
| P1-6 fixture/S1/A1-A2 API | 成立 | server/auth/db/assert fixtures；S1实际版本；A1/A2共断言。 |
| P1-7 上级文档/README | 部分成立 | 下列3.20矩阵多数一致；收尾明记四个漏同步而非伪称Phase当时齐备，D:1823仍漏。 |
| P2-1 login/refresh/logout/期限 | 成立 | app/login:64/refresh:49/logout:27；conditional SQL；A3-A6 API。 |
| P2-2 限流/失败预留/IP | 成立 | ratelimit tests:96/163/193/213；H/api_ratelimit_test.go；A15双桶。 |
| P2-3 安全响应头 | 成立 | H/middleware.go + middleware_test；bootstrap/contract_test:175；CSP另webui负责。 |
| P2-4 A3-A6/A15 API、login fixture | 成立 | E对应5文件接口入口且真实DB。 |
| P2-5 上级文档/README | 成立 | V:225/230/232、README:109–112 代理/期限/枚举；3.20表可追溯。 |
| P3a-1 /me/profile/password/deactivate | 部分成立 | 五用例+profile SQL合并；正常A7-A10/A12通过；Critical 1。 |
| P3a-2 PAT迁移、create/list/revoke/auth/last_used | 成立 | migration00004、app三个用例、Q/api_tokens:13/14/21/42；A11跨页+一次展示+401。 |
| P3a-3 instance三字段/timezones/tzdata | 成立 | instance合约及S3；instance模块嵌tzdata；时区非法值领域拒绝。 |
| P3a-4 交错4/5 | 成立 | I/interleavings_test:201、230（第6在 adapter/postgres/credentials_test.go:133；:258另测改密等待持锁登录）；真实DB锁等待而非只伪造接口。 |
| P3a-5 uuid guard | 成立 | archtest/generated_test:23/48/72，runtime.UUID变异被拒。 |
| P3a-6 body格式与参数整程序 | 部分成立 | bootstrap/contract:93/137常规格式/params成立；重复key未拒。 |
| P3a-7 七桶配置接线测试 | 成立 | identity整程序limiter覆盖全部七桶，A15另实际页面/API；此前P2移交已补。 |
| P3a-8 A7-A11 API/S3 | 部分成立 | 业务断言实际存在，共函数缺口A8–A10。 |
| P3a-9 上级文档/README | 成立 | V:181/230/231、plane-diff PAT逐列、README使用PAT和CLI恢复区别。 |
| P3b-1 五CLI/NewAdmin最小组合 | 成立 | I/admin.go:40；A12/13/16/17；非serve没有job client。 |
| P3b-2 River迁移/client/启动停机/cleanup/config | 部分成立 | 分离角色启动新反例。jobs.go、runner_test；migrations00005；Q/sessions:64；A14；README:124–125解释36s和River停机。 Important 2：分离角色启动报42501；readyz200不代表worker可运行。 |
| P3b-3 交错1–3 | 成立 | I/interleavings_reset_test:148/170/190/213/232/256双顺序测试；重置与签发串行。 |
| P3b-4 不可用hash仍假hash成本 | 成立 | P3b补登录hash形式cases；app/login:64/134；旧参数时差是明确另一个已知边界。 |
| P3b-5 A12–14/16/17+停机 | 成立 | 五e2e故事；当前e2e子进程干净退出；历史毫秒数据不冒充当前满负载36s实测。 |
| P3b-6 文档/River表/README | 部分成立 | 分离角色权限缺项。plane-diff新增infra四表、README停机、命令；README:121的手动撤销分支不是完整清退（第6节）。 Important 2：README:123 只说明业务表和 goose 的权限，缺少 River DML/sequence/MAINTAIN 及其验收。 |
| P4-1 client/manager/locks/四storage事件/timeout | 部分成立 | token manager tests60/68/84/120/233/298/312/400；tabs tests84/187/222/244；Important 1。 |
| P4-2 CSRF/form/axios清理 | 成立 | 60-rule关键词基线，正文literal守住；动态拼接可绕过为scanner能力边界，不等于残留运行路径。 |
| P4-3 login/signup/errors/next_path | 成立 | A1–A3、A15；结构与值error显示；next_path攻击向量矩阵。 |
| P4-4 wrappers/user/profile/instance/types | 部分成立 | `W/store/user/index.ts`/profile/instance直接生成类型；分代stores与auth middleware；租约退出缺口。 Minor 1：`docs/v0/M2-auth/specs/P4-web-auth.md:163` 要求会话变化时重置主题；`W/lib/store-context.tsx:41–50` 只重置 localStorage，停用路径漏掉普通退出按钮的 setTheme 处理。 |
| P4-5 onboarding删步/预取/avatar | 成立 | A10首次打开无旧请求/consoleerror +资料到workspace；用户选择后M3未实现属范围。 |
| P4-6 CSP | 成立 | S2/A10 cspViolations空；webui生成内联hash；不从CDN载脚本。 |
| P4-7 keywords | 成立 | 于静态文字规则。tools/keywords.json60/3；已知semantic绕过不声称全静态安全证明。 |
| P4-8 fixture+A1–A6/A10/A15page/S2 | 部分成立 | 页面都有且当前通过；A6的Important 1反例未覆盖，A10共断言未共用。 |
| P4-9 上级文档/README | 部分成立 | V:238–242、7.7；描述lease冻结只说reuse/relogin，未揭示旧退出能清新session。 |
| P5-1 四页/PATstore/dropdowns/维护 | 部分成立 | A7/8/9/11/12及S5；功能成立；已知日期/时区键盘未完成与同页PATCH乱序由第6节已知限制评价。 Minor 1：自助停用返回登录页后仍显示旧账户暗色主题，刷新后才恢复默认。 |
| P5-2 services只留地址文件/旧类型删 | 成立 | `web/packages/services/src` 已无axios API基类；关键词plane token/timezone/user禁用；M5交接§3最终删除services。 |
| P5-3 66死项与lint清理 | 成立 | 于登记口径。P5review记录剩24=23test数据+1M3prop；运行lint上限web452/ui19/utils12/i18n0；非无意义把所有分析命中当死代码删。 |
| P5-4 keywords incl withCredentials | 成立 | 于字面。全仓零命中；withCredentials空格/quoted写法可绕过，不提供语义完备保证。 |
| P5-5 A7–9/A11/A12page | 部分成立 | 正常业务断言成立，共函数A8/A9未落实。 |
| P5-6 frontend-changes | 成立 | frontend-changes第三节已将M2改完成并说明移植范围，原M1时点保留历史不算现时冲突。 |
| closeout-1 十份入向闭合 | 部分成立 | 10closed、有逐项原条款；M0-P6共业务断言条并未完全兑现。 |
| closeout-2 3.20逐行核对 | 部分成立 | 四历史漏项诚实记录；目前还有Minor 3两处。 |
| closeout-3 M3–M8交接 | 成立 | 6文件50节有接收条件；风险交给具体M/代码/验证方式，下表。 现有接收条件本身明确；Important 2 还需追加 River 分离角色的部署验收。 |
| closeout-4 规模对比 | 部分成立 | 数据明确559c3c6..d97c513不同当前HEAD、可复测；identity操作数量错误Minor 3。 |
| closeout-5 仅跨M关键词例外 | 成立 | 60rules3exceptions，余额明确M3/M5等接手，literal变异证实守卫存在。 |
| closeout-6 总体M2状态完成 | 部分成立 | 状态确改完成，但Critical 1、Important 1/2及Minor 2要求把“完成”解释为当时验收结果，现时不满足Ready条件。 |

### 5.6 13.1 十份入向交接

所有路径位于 `docs/v0/M2-auth/handoffs/`，每份 `:2` 都是 `status: closed`，这只证明状态；以下列实现或实际移交支撑。

| 交接 | 结论 | 逐项闭合支撑 |
| --- | --- | --- |
| M0-P1-sqlc-cgo | 成立 | `server/sqlc.yaml`已限定identity四迁移，sqlc生成adapter/postgres/gen；gen-check现时成功，交接:26以后说明PG17解析及CGO工具条件。 |
| M0-P2-platform-notes | 部分成立 | River服务角色尚不完整。auth/limit已接、RequestID/Clock/TxManager实际用，密钥file_set；CLI/River、prod迁移角色README:123；接口调用日志不在M2，M8交接§1明确位置/覆盖；余三配置小问题有加载拒绝测试。 Important 2：原 goose SELECT 条款已写入，但 River 分离服务角色的完整权限尚未闭合。 |
| M0-P3-api-codegen-notes | 部分成立 | nullable/stdlibuuid/emailstring、security、模块Register/API、三错误出口（bootstrap contract:93/137）均有代码；Critical 1说明bodyshape与decoder一致性未全闭。 |
| M0-P4-schema-conventions | 成立 | 四表无DB生成UUID；app显式now；FKONDELETE/约束名/CHECK在migration与schema_test:99/164；快照差异登记。 |
| M0-P5-frontend-api-notes | 成立 | 新client+Bearerservices，安全headers在httpserver、CSP在webui依赖嵌入内容；“同一层”按依赖边界正式修订不是遗漏，S2响应/CSP验证。 |
| M0-P6-e2e-notes | 部分成立 | auth/db/assert模板、动态监听地址、停机、保留trace/screenshot/log/dbdump、S1/S2/S3；clock/storage/webhook分别转M4/M5/M8且有close条件；A8–A10并未共业务断言。 |
| M1-P2-trim-content | 成立 | Theme来自生成Profile，不残留IUserTheme映射；A9中英文/刷新/主题拒绝；P4已删旧类型。 |
| M1-P3-trim-platform | 成立 | CSRF/旧auth地址/类型删，instance三字段、角色用途删；CLIreset恢复、PAT新路径；60rules阻止已登记literal回流；M5上传字段留null合理。 |
| M1-P4-router-native | 成立 | 前端校验next_path且JSON服务器不发redirect；A3包含query/hash与恶意值；auth wrapper和password错误写新code。 |
| M1-closeout | 部分成立 | 66死项中26随后删除+15随重写删除，剩24是23test替身/字面量+1M3prop（口径起始变动见P5review）；lintcap实配452/19/12/0；主题定位已测；键盘日期/时区问题明确留后续，见已知判断。 |

### 5.7 第14节逐项完成标准

| D行号/标准 | 结论 | 证据与不成立部分 |
| --- | --- | --- |
| 2120 七阶段有spec/plan/review | 成立 | specs/plans/reviews各7组文件真实存在；实施证据见上表，不以文件存在证明业务正确。 |
| 2121 A1–A17/S1–S4、PAT共断言、A7参数 | 部分成立 | 48passed、A7参数成立；A8–A10不共业务断言，Critical 1、Important 1 未覆盖。 |
| 2122 后端tests/contracts/architecture/gen | 部分成立 | 本次运行Go/gen成功及实际mutation；Critical 1、sqlc已知regex不完备，门禁成功不等于所有边界挡住。 |
| 2123 frontend/types/knip/lintcaps/tests/keywords | 成立 | 按命令准则。当前web16tasks无cache、knip成功、caps实际4个；lint执行结果见第1节；关键词60/3literal范围。 |
| 2124 无CSRF/旧接口，挂载不访问M3 | 成立 | 关键词当前零命中；S2/A3/A7–12 watch；主动提交M3表单明确不在范围。 |
| 2125 四表与River迁移/逐列差异 | 成立 | 10/11/12/12业务列；River4表+enum/function，goose独链；schema_test实际约束。 |
| 2126 四决策/11.1/总体4.2 | 部分成立 | 决策实现已核；后端退出仅当前，前端冻结旧logout清新记录破坏用户语义。 |
| 2127 9.6浏览器核对脚本全文 | 成立 | 档案完备。P4review/P5review附录确有浏览器脚本，包含C4a–c/C5局域HTTP/C6b–c；历史执行不替代本次复现，不覆盖冻结十秒新条件。 |
| 2128 无open入向/M3–M8交接 | 成立 | 于状态与转交。10closed、6出向open，各有可检关闭条件，非简单“下个M再说”。 |
| 2129 上级文档/README/总体状态同步 | 部分成立 | 3.20主要行落实；D:1823及closeout操作数遗留，README恢复分支有已知歧义。 Important 2：分离角色部署的 River 权限说明不完整。 |
| 2130 规模估计与实际 | 部分成立 | 附录B明确比较基线、分类和CIrun，identity13误写15；应修分母后再作为M3估算依据。 |

### 5.8 17.2 已吸收设计评审逐项对照

下表标识均为D:2226–2317的原编号。重复编号（I-7/N4等）分别列出但引用同一落点，不将重复计成独立成果。

| 原项 | 结论 | 当前代码或断言证据 |
| --- | --- | --- |
| I-1 并发重置清退凭证 | 成立 | `I/app/credential_lock.go:29`；reset交错真实DB六子测试:148/170/190/213/232/256；users.sql NO KEY UPDATE。 |
| I-2 真假旧refresh区分 | 成立 | `I/adapter/signing/keys.go:17、52`HKDF MAC；refresh:49判定；A5伪造不撤销/真旧代撤销；一session一行，无每代无界表。 |
| I-3 认证前失败闸门 | 成立 | H/api.go接Reserve；api_auth_test并发/失效JWT退款；ratelimit_test:163/193；桶不是认证后才执行。 |
| I-4 refresh派生永久PAT | 成立 | 为显式接受。README:121、V:725风险与恢复；未限制PAT链，这是负责人产品选择；完整恢复措辞异议见已知。 |
| I-5 跨标签账户错乱 | 部分成立 | auth-middleware.session.test、token-manager.tabs:244；A6直接换账户，A9迟应答；Important 1。 |
| I-6 拒绝契约禁止请求 | 不成立 | 绝对保证。bodyshape:176 map只查最后同名对象，而generated struct decoder合并；本次评审真实DB复现WORKSPACE_INVITE绕过。普通未知键用例通过不消除此反例。 |
| I-7 跨模块ALTER归属 | 部分成立 | sqlc.yaml/archtest.sqlc:113有所有者；无引号ALTERmutation失败；quotedALTER/DROP通过是已知M3加固项。 |
| I-8 onboarding并发merge | 成立 | Q/profiles.sql:20 JSONB双竖线合并操作；DB并发merge测试，A10 API预置别的键后仍保留。 |
| I-9 cursor不固定排序 | 成立 | shared/cursor:21、41封套任意payload；domain PAT cursor载荷，M4/M7交接要求各自排序。 |
| I-10 业务时钟/updated_at | 成立 | Q各INSERT/UPDATE用sqlc.arg(now)，固定Clock截微秒；schema CHECK/now兜底不替代业务clock。 |
| I-11 onboarding不预取M3 | 成立 | A10:35–39及注册后clientnavigation测试；onboarding旧workspace/invite预取删除。 |
| I-12 A4旧令牌用法矛盾 | 成立 | A4 API:158每次拿上一轮返回的令牌，真旧重放单独A5。 |
| M-1 limit错误落点 | 成立 | shared分页1–100；listApiTokens参数abc→400、0/101→422；bootstrap参数test:137。 |
| M-2 RetryAfter名冲突 | 成立 | shared.Error对外RetryDelay与http平台接口结构满足；项目编译/测试通过，没有字段方法同名。 |
| M-3 email/JSON CHECK | 成立 | migrations00001邮箱lower/ASCIIspace（不宣称完整emailvalidator）；00002 CASE分支包jsonb运算；schema_test:194–211含标量/数组/null反例。 |
| M-4 key路径日志 | 成立 | config LogValue只*_file_set、keysParse错误不打印PEM；signing_test:184固定reason；密钥文件路径未当成日志配置值输出。 |
| M-5 非401 refresh失败保留会话 | 成立 | token-manager.test:120/261/275/298/312；tabs timeout:288；仅retry/backoff，不默认clear。 |
| M-6 A7保留session差异 | 成立 | A7page:65 held/API:104省略，A:283同函数。 |
| M-7 前缀不自动扫描 | 成立 | README:122明确自定义规则，PAT43/refresh91 base64url字元；真实格式51/98总长。 |
| M-8 分Phase登记/非照搬措辞 | 部分成立 | plane-diff全局/四表/behavior、closeout诚实补4漏项；Minor 3尚有文字残留，不能称所有同步零误差。 |
| M-9 密码表口径 | 成立 | 数据区33887条277141bytes；全文件33904行278161bytes；不是七个#开头密码当注释丢掉。 |
| §4 四决策与11.1 | 部分成立 | 上表分别核；Important 1是行为缺陷，不是未裁定。 |
| §5 ProblemStatus取舍 | 成立 | shared.Error结构满足平台接口、不importnet/http；rules_test:193/216证明层界。 |
| §5 M0-P5同层正式关闭 | 成立 | http安全头属固定链、webui需要indexhash产CSP；M0-P5交接:33–44解释范围并关闭。 |
| §5 M0-P3参数错误出口 | 成立 | APIErrors.BadRequest绑定参数名；bootstrap/contract_test:137及list/revoke API参数反例。 |
| §5 FK关系图/M8prod | 成立 | 为移交。migrationsONDELETE实现当前表；M3/4/6/7各FKclose条件、M8§3镜像prod和停止宽限。 |
| N1 密码主干一字符绕过 | 成立 | password.go:83去两端非字母；generator相同规则；A2 Password1!~被拒。 |
| N2 timeout先后 | 成立 | 于正常调度。服务4s+commit2s小于前端8s、lease10s；冻结不会保证walltimer执行，已接受lease风险与新退出问题应分别对待。 |
| N3 anonymous/未认证输入键/暂不可用 | 成立 | 600/100，refresh/logout只匿名；IP键不取sid；token-manager.test:120等保留session退避。 |
| N4 同I-7 | 部分成立 | 所有者守卫有用但regex限制如I-7。 |
| m1 Router登记/查询参数test | 成立 | H/routes.go同时记录Handle/HandleFunc；contract_test注册集合；apitest.Target补必填参数。 |
| m2 不仅未认证才算失败 | 成立 | JWT伪签、撤销、PAT无效等落失败门；过期有效签名、DB故障/成功退回；H/api_auth_test。 |
| m3 同I-8 | 成立 | Q/profiles.sql:20及并发测试。 |
| m4 草稿结构复用 | 成立 | 为移交。M4§4明确same schema/bodyshape和publish复用，当前M2无draft不提前造实现；新duplicate缺陷须先修。 |
| m5 忘设prod | 成立 | 为缓解。dev loopback默认、warnIfExposed与configtests；README:100；M8§3镜像prod必检。 |
| m6 AllowAll原子/注册先后 | 成立 | ratelimit_test:96两个桶全扣或全不扣；register:55先signup；A2关闭已存在email同403。 |
| m7 upload与模块限额 | 成立 | 为移交。M5§2按操作放宽或独立路由必须纳入契约/显式设计，filelimit/CSP有关闭条件。 |
| R1 格式校验与decoder一致 | 部分成立 | 同raw json.Unmarshal进映射Go型对uuid/time生效；重复对象合并语义仍不同（Critical 1）。 |
| R2 NO KEY UPDATE/FK顺序 | 成立 | Q/users.sql:32/62；`I/adapter/postgres/credentials_test.go:133`真锁阻挡/插入不挡；interleavings测试。 |
| R3 存每代无上界 | 成立 | migrations00003单session12列，generation/tokenhash原位轮换，旧代MAC，无auth_refresh_tokens表。 |
| R4 login_id写入竞争 | 部分成立 | 统一锁与refresh写回核对已实现；logout冻结过租约后删除新record尚缺最后核对（Important 1）。 |
| R5 并发失败门超额 | 成立 | ratelimit_test:193和H/api_auth_test并发假认证器只允许额度内被调用。 |
| R6 IPv6前缀 | 成立 | H/clientip.go与clientip_test，config默认64、校验prefix；不是按单IPv6地址绕过。 |
| R7 commit取消/未知结果 | 成立 | postgres/tx.go:92 WithoutCancel+independenttimeout；请求cancel后commit/rollbackintegration；未知结果不会假称rollback。 |
| R8 migration角色README | 部分成立 | 原条款已写，River角色缺项。README:123 goose_db_version SELECT必须授权；不是要求服务角色使用River私迁移表。 Important 2：goose SELECT 只解决就绪检查；River queue/leader DML 与 REINDEX MAINTAIN 仍缺。 |
| R细节-重复使用限未撤销未过期 | 成立 | Q/sessions条件、domain refresh判定；A5。 |
| R细节-仅rehash并发重验一次 | 成立 | I/app/login.go和change_password.go:104；interleavings_test:230，reset_test:190/213。 |
| R细节-login与set-email | 成立 | 同account锁+变更邮箱revokesessions；A16。 |
| R细节-M3引用纠正 | 成立 | M3交接§2指实际onboarding页面及await链，不需复刻旧行号。 |
| R细节-cursor首列表才加入 | 成立 | shared/cursor现时PATlist实际使用，无孤立框架。 |
| R细节-clock微秒 | 成立 | platform/clock/clock_test:36/40，postgres审计时间相等测试。 |
| R细节-换key风险 | 成立 | 为接受。signing MAC派生同key；README:108明确旧代不能reuse识别、当时盗用仍需恢复。 |
| R细节-失败门计数范围 | 成立 | H/api_auth_test与ratelimittests；对应N3/m2。 |
| F1 bodyshapestdlib/uuid guard | 部分成立 | 代码bodyshape仅stdlib、uuidmutation失败；D:1823旧说法未清；Critical 1另外列。 |
| F2 tools进入CI | 成立 | Makefile test/lint显式server/tools；本次评审工具模块测试通过。 |
| F3 旧措辞一session/顺序 | 成立 | actualsingle-row+profiles先更新；D/A3不再第0代独行表模型。 |
| F4 COMMIT未知/ROLLBACK取消/单写 | 成立 | postgres/tx.go:60/92；单字段profile可重放但不把commit未知说成失败回滚；strictrefresh响应丢失是显式风险。 |
| F细节-key rotation失败门/盗用 | 成立 | 为接受。README:108、D§16，与MACrotation代码一致。 |
| F细节-预留并发上限是当前余额 | 成立 | ratelimit预留实际扣当前余额；README/设计不是恒可60并发。 |
| F细节-idle keys清理测试 | 成立 | platform/ratelimit/ratelimit_test.go:213 `TestIdleKeysAreDropped`。 |

### 5.9 3.20 上级文档与实际代码同步（22行）

| 3.20条款 / 上级位置 | 结论 | 实际代码/断言 |
| --- | --- | --- |
| P1注册返回token，V:149 | 成立 | A1API201 tokenpair，A:139按returnedrefresh核session；GET/me独立读取。 |
| P1错误结构和值分层，V:201 | 部分成立 | bootstrap/contract:93普通结构400/领域422；Critical 1，文字承诺需修代码兑现。 |
| P1refresh布局，V:225 | 成立 | A:20解析68bytes，signing/keys:52，sessions只currenthash。 |
| P1signupdefault，V:229 | 成立 | configs/embed_test:34/108+CLIcreate A17。 |
| P1auditclock，V:5.5 | 成立 | Q/{users,profiles,sessions,api_tokens}.sql显式now；clock_test截微秒。 |
| P1迁移所有权，V:408 | 部分成立 | sqlc.yaml归identity，archtest常规ALTER通过守卫，quoted绕过已知。 |
| P1app ports/shared，V:360/395 | 成立 | I/app/ports.go，shared仅stdlib，archtest非法依赖mutation失败。 |
| P1moduleAPI入口，V:6.2 | 成立 | I/module.go:160/165/175 PublicOperations/Authenticator/Register(router,api)。 |
| P1默认拒绝与pipeline，V:423–424 | 部分成立 | bootstrap/contract:25/42/67；bodyshape 位于正确的边界，但 Critical 1 表明结构检查仍可绕过。 |
| P1Tx commit，V:6.4 | 成立 | postgres/tx.go:92独立context，不受requestcancel。 |
| P1E2E失败留存/地址，V:8.2 | 成立 | e2e server.ts动态:0/addrfile；test fixture trace/screenshot/log/dbdump而非录像。 |
| P1差异清单全局+users/profiles/session | 成立 | plane-diff全局ONDELETE/非deferrable/命名/应用audit；四migration与schema_test约束值一致。 |
| P2绝对30d，V:225 | 成立 | Q/sessions.sql:28–31不改expires；A4 expectRefreshed验证原expiry。 |
| P2退出与真实旧代，V:230/232 | 成立 | 服务端。app/logout:27；refresh:49+MAC；A5/A6API。 |
| P2安全固定链，M0设计3.3 | 成立 | httpserver.middleware实际header链；CSP在webui其后HTML写出。 |
| P3a cursor封套，V:181 | 成立 | shared/cursor:21/41，PATownpayload；invalidcanonical游标400，limit422。 |
| P3a密码/停用/锁，V:230–231 | 成立 | change_password:58/104、deactivate:42；A7param与A12、交错。 |
| P3a uuid例外，M0设计3.7/M0P3spec2.8 | 成立 | generated_test:23/48/72允许runtime间接导入但禁止gen漏映射；mutation拒绝。 |
| P3b reset全部PAT/activate恢复，V:230 | 成立 | reset_password:57、activate:45，A13/A12。 |
| P4浏览器分代/401/retry，V:238–242、587–593 | 部分成立 | 单测与A6/A9正常分代/迟答通过，Important 1破坏“不会结束别处新会话”。 Minor 1：会话变化时主题复位的具体要求来自 `docs/v0/M2-auth/specs/P4-web-auth.md:163`；`W/lib/store-context.tsx:41–50` 只重置 localStorage，停用后 DOM 仍保留旧主题。V:593 是页面级状态处理约定，不是默认主题复位的明文承诺。 |
| P4/P5frontend-changes3.1/3.2 | 成立 | User/Profile/Token/Instance services和store已生成类型，CSRF/axios基类已移除；M1scope残留由M3–M8继续接。 |
| closeout V:9.4状态 | 部分成立 | 状态写completed事实存在；本次反例说明Ready判定应重开Critical 1和Important 1/2所涉及的修复，不以状态字段覆盖代码事实。 |

### 5.10 文档数字与实际口径

| 文档数字 | 现时计数/证据 | 判断 |
| --- | --- | --- |
| 4张业务表；users10/profile11/session12/PAT12 | migrations00001–00004逐列人工核对；profilesCHECK延续行不可误计为column 一致。 | 成立 |
| 五迁移，River主线2–7/4表，无river_migration | sql00001…00005；S1期望表集，schema migrationtest 一致。 | 成立 |
| identity 623行、15操作 | actual623行13operationId；dist878行15operationId 操作数不一致（Minor 3）。 | 部分成立 |
| blocklist33887/277141 | common_passwords.txt整文件33904行278161bytes；17行1020bytes生成说明之后才是数据 数据区一致；不能用排除所有#行的方法计数，因为密码可#开头。 | 成立 |
| 60关键词规则/3例外 | tools/keywords.json rules60/exceptions3，当前运行零hits 一致；文本guard不等于AST保证。 | 成立 |
| oxlint caps web452/ui19/utils12/i18n0 | 相应package.json的check:lint分别452/19/12/0（web:10，其余18/19） 一致。 | 成立 |
| 48e2e/17身份故事文件 | E17文件；当前48passed，另S1–S5 一致。 | 成立 |
| 7阶段各spec/plan/review | P1、P2、P3a、P3b、P4、P5、closeout七组 一致。 | 成立 |
| 十份入向、六份出向；13.2的31行 | 10closed，6open；出向M3 14节/M4 13/M5 5/M6 6/M7 6/M8 6，共50节与50关闭条件 一致；设计31行展开为50节，不是数量冲突。 | 成立 |
| 令牌98/51字元，刷新结构68bytes | 刷新7+91；PAT8+43；刷新16sid+4generation+32secret+16MAC 一致。 | 成立 |
| argon2 19456KiB×4≈76MiB；t2/p1；test64KiB/t1 | configs默认与hashadapter；domain密码8–128UTF16 一致；内存峰值还需加进程/请求/表开销，M8不应把76MiB当RSS。 | 成立 |
| 七桶dev/prod及test全部600000/100000 | 配置七名anonymous/auth_failure/authenticated/login_ip/login_ip_email/register_ip/password_user 一致于配置/接线测试；不能把test宽桶误报生产。 | 成立 |
| 最坏停机36s | HTTP20+jobs10+额外1+pool5；README:124 合理上界；空闲几毫秒不证明满载shutdown，一小时Riverrescue另说明。 | 成立 |
| closeout净变化/规模 | 明确统计559c3c6..d97c513，当前HEAD另加closeout文档/测试；后端生产net7742、测试15593、frontendnet2782中tests3712 不把基线差异误报；identity操作分母错误会影响后续估算需修。 | 部分成立 |


### 5.11 前端会话长期规则的逐项核对

本表把长期会话规则拆开检查，避免将“每次新建 RootStore”扩成所有全局副作用自动安全。

| 约定 | 结论 | 证据与验证等级 |
|---|---|---|
| navigator.locks 下同一浏览器续期串行 | 成立 | `web/apps/web/core/lib/auth/refresh-lock.ts:22`；P4 C3 实测两 tab 共 8 次刷新全部 200，代数不重用，退出同步；本轮全量 A4 对应路径通过 |
| 无 Web Locks 时各标签页共用同一 localStorage 租约键，并在单 tab 内排队 | 部分成立 | `web/apps/web/core/lib/auth/api-client.ts:37`–`:39`、`web/apps/web/core/lib/auth/refresh-lock.ts:52`–`:68`；P4 C4b 实测旧刷新结果不覆盖新会话；租约自身限制已登记，退出有 Important 1 |
| 续期写回前核对 login_id | 成立 | `web/apps/web/core/lib/auth/token-manager.ts:239`–`:244`；P4 C4b 实测 X 代数 2→3，但 X 新令牌没有写回 Y 记录，两账户的名字都未被旧操作改写 |
| 退出只作用于被调用时的会话 | 部分成立 | 锁获取前换账户的已有测试覆盖且通过；异步等待后租约失效的 Important 1 失败 |
| 只有续期 401 / 重发 401 自动结束会话 | 成立 | 成立（正常路径）。`web/apps/web/core/lib/auth/auth-middleware.ts:44`–`:55`、`web/apps/web/core/lib/auth/token-manager.ts:254`–`:260`；暂时失败不清记录；P4 C6c 实测 503 / Retry-After 2 秒，约 2.29 秒自动恢复。显式退出/停用是另外的用户操作 |
| 退避行为 | 成立 | 代码无 Retry-After 时 1、2、4…30 秒；有 Retry-After 时按服务端值，代码没有额外把它裁到 30 秒。设计的“最多 30 秒”适用于指数退避，不能改写为所有 Retry-After 也封顶 |
| RootStore 每个 login_id 一代，instance/router/theme 复用 | 成立 | `web/apps/web/core/lib/store-context.tsx:47`–`:59`、`web/apps/web/core/store/root.store.ts:94`–`:117`；账号、资料、PAT 都接收本代 ApiClient；本轮强制无 cache 的 test-web 通过，P4 C4a/c 和本次 PAT 双账户验证通过 |
| 旧一代成功的在途响应不能写进新一代 store | 成立 | 成功响应允许完成，但对象为旧 RootStore；旧下一步请求的客户端因 login_id 不同拒绝。不能把此约定说成所有旧 200 都抛 SessionChangedError |
| SessionChangedError 不当退出 | 成立 | `web/apps/web/core/store/user/index.ts:125`、`web/apps/web/core/store/user/api-token.store.ts:80` 忽略后台截断；auth-middleware 37/48/54 三处区别旧会话；前台提示失败不删除新会话。P4 C4b 的旧保存提示失败但 Y 会话保持 |
| SWR key 含 loginId | 成立 | `web/apps/web/core/lib/wrappers/authentication-wrapper.tsx:66`、`web/apps/web/core/components/api-token/token-list.tsx:39`；RootStore 与列表在换账户后重新取数 |
| PAT 创建原文不进入列表 store，关闭后不再显示 | 成立 | `web/apps/web/core/store/user/api-token.store.ts:99`–`:110` 明确构造无原文列表对象；P5 C3，本轮 A11，以及下述跨账户候选试验均通过 |
| 旧会话迟到的 PAT 创建不会在 B 页面显示/下载 | 成立 | 成立（本次两种真实交错）。`web/apps/web/core/lib/wrappers/authentication-wrapper.tsx:81` 返回 Loading 卸载受保护子树；`web/apps/web/core/components/api-token/modal/create-token-modal.tsx:53`–`:55` 清理推进 phase，`:88` 抛弃迟到结果。见下方两种交错对照 |
| 语言只由当前一代 profile 派生 | 成立 | 成立；同一代顺序限制见已知事项。`web/apps/web/core/lib/wrappers/store-wrapper.tsx:83`–`:85`、`web/packages/i18n/src/core/set-language.ts:16`–`:23`；不由旧 store 直接修改全局语言 |
| 新会话显示默认主题 | 部分成立 | 新资料成功后按其主题应用，旧主题修改有 inSession 检查；但只有默认存储值重置，停用后当前 DOM 不跟随，Minor 1 |
| 无刷新记录时不请求 /me | 成立 | `web/apps/web/core/lib/auth/token-manager.ts:195`–`:202`、`web/apps/web/core/store/user/index.ts:115`–`:117`；本轮 S2 通过；本次 P4 C8 新注册/登录均只请求 instance 与 M2 API |
| M2 可达页面挂载不请求 M3 旧接口 | 成立 | 成立（已跑路径）。P4 C8/C9 和 P5 C1–C5 无旧 API、无未预期 API 失败；C4 的 422、C5 的 403 属于脚本明确允许的预期拒绝；没有以此保证用户提交 M3 工作区表单也成功 |
| 四个设置页和共享下拉组件 | 部分成立 | 成立，键盘延后项除外。P5 C1–C6 全重跑；中英文 theme/week/timezone 列表均与按钮间隔 4px、边缘差 0px，Escape 一次关闭且一次点击重开 |
| next_path 保留 query/hash | 成立 | 已有 unit/e2e 和静态证据。`AuthenticationWrapper` 拼 pathname+search+hash；本轮 A3 通过；前端分工没有另跑重定向攻击载荷，不将其写成独立攻击验证 |
| 页面 CSP 正常加载与内联哈希 | 成立 | 已跑正常页面；安全效果未独立攻击验证。P4 C9 的登录、注册、onboarding、create-workspace、4个settings页均落到请求页面，有 CSP 且无违规；未新增 XSS 载荷试验 |
| XSS / 用户可控内容 | 部分成立 | 仅读代码，不给安全无缺陷结论。M2 display_name、PAT label/description、字段错误通过 React 文本与固定翻译键渲染；未发现这些路径上新增 raw HTML sink。未对整个 Plane 编辑器或遗留领域做全面 XSS 审计 |

本轮还独立否定了一个可疑点：A 创建 PAT 已在后端成功但响应扣住，B 登录且 B 的独有列表 marker 出现后释放 A；以及 A 的 PAT 已展示并下载一次后再切 B。两次真实页面实验都为 **0 次 A 原文在 B 页面出现、0 次切 B 后新增下载、0 条 A 列表项混入 B**。认证包装在新一代尚无 user/profile 时返回 Loading，旧弹窗卸载使 phase 失效。不能仅凭 modal 没有 loginId 比较就报泄漏；若未来改变这一卸载条件，须保留此跨账户回归。


### 5.12 四张业务表的逐列核对

下表逐列对照 M2 设计第4节、`docs/v0/plane-diff.md` 第二节与实际迁移；列顺序按迁移。`server/migrations/schema_test.go:57`、`:99`、`:164` 的真实 PostgreSQL 测试核对往返、约束/索引名称与非法输入反例，均在本轮通过。

| 表 / 迁移位置 | 实际列（全部） | 关键约束、索引与结论 |
|---|---|---|
| users；`server/migrations/sql/00001_identity_users.sql:5` | id, email, password, first_name, last_name, display_name, user_timezone, is_active, created_at, updated_at | 10 列；email UNIQUE、小写且不含 PostgreSQL POSIX `[[:space:]]` 匹配的空白；display_name 非空；应用生成 UUID 与审计时间。成立 |
| profiles；`server/migrations/sql/00002_identity_profiles.sql:4` | id, user_id, theme, is_tour_completed, onboarding_step, is_onboarded, last_workspace_id, language, start_of_the_week, created_at, updated_at | 11 列；user_id UNIQUE、CASCADE；theme 枚举、语言 en/zh-CN、周起点 0–6；onboarding 恰好四个布尔键；last_workspace_id 当前无 FK 是明确保留项。成立 |
| auth_sessions；`server/migrations/sql/00003_identity_auth_sessions.sql:5` | id, user_id, token_hash, generation, user_agent, ip, expires_at, last_refreshed_at, revoked_at, revoke_reason, created_at, updated_at | 12 列；user CASCADE；hash 32 bytes、generation≥0；撤销时间与原因同时空/非空；user_id、expires_at 索引。成立 |
| api_tokens；`server/migrations/sql/00004_identity_api_tokens.sql:5` | id, user_id, token_hash, label, description, expired_at, last_used, created_by_id, updated_by_id, created_at, updated_at, deleted_at | 12 列；hash UNIQUE/32 bytes；user CASCADE、两个审计用户 SET NULL；未撤销条目的 user_id/created_at DESC/id DESC 部分索引；不存原文。成立 |

业务表 `created_at/updated_at` 的 `DEFAULT now()` 仅作兜底，正常写入由用例 clock 显式传值；River 自己的表与时间不混算成业务约定。约束名遵循 PostgreSQL 自动命名或明确声明的 `auth_sessions_revoked_consistent_check`；FK 未声明 DEFERRABLE，与差异清单一致。没有发现当前查询手写绕开 sqlc 的业务路径；River 内部查询、迁移检查和测试探针不属于 identity 业务查询。

## 6. 已知事项复核


P5 的请求顺序限制不是本次新发现。`docs/v0/M2-auth/reviews/P5-web-account-review.md:136` 对修复的原描述是：“两个 store 按发出的顺序给修改编号，比最后写入的那次旧的应答丢弃；新的修改被拒绝时，旧的成功仍会写入。”同文档 `:232` 明确限定其前提：“**资料和账户的应答先后**：假定 nerve 按发出的顺序处理这些请求（与 PAT store 的假定相同）。”本评审不同意将这个前提升级为用户最后选择必定落库的保证，但按既有已知限制处理，不计入 Critical/Important/Minor 新发现数量。

README 恢复方案的异议也已有记录。`docs/v0/M2-auth/reviews/P3b-jobs-and-admin-review.md:107` 的原文是：“README 恢复步骤的第 1 步（逐个撤销不认识的 PAT）没提结束攻击者的会话：它逐字照抄 M2 设计 8.5，是设计层面的事。”这说明遗漏来自已接受的设计，不能证明第一分支能完成恢复：仍持有效刷新令牌的攻击者可以再次创建 PAT。建议在 README:121 明确手动撤销只清除所选 PAT，管理员 `reset-password` 才结束全部现存会话并撤销全部 PAT；本项不重复计为新发现。

以下均为已有记录的接受、延后或不处理项，明确不计入本报告新发现数量。Critical 1 和 Important 1 的独立因果链在发现部分论证。来源主要为D§16、P1–P5各review“接受/不改/移交”与closeout附录C；同一风险跨review重复只列一行，但逐个给理由。

| 已知项 | 判断 | 理由与推荐时点 |
|---|---|---|
| strict refresh响应丢失/页关闭/COMMIT未知要重登录 | 同意有条件接受 | 服务端无法确认客户端收到一次性新token；当前不承诺无损网络。日后加宽限需重审安全，M8先测发生率。 |
| native locks与跨渲染进程localStorage同步无强顺序 | 同意暂接受 | A4可证明常见互斥，不是跨进程原子性证明；已文档可观察。 |
| 无locks租约非原子/冻结>10s可能reuserevoke | 同意原风险，不同意吞并Important 1 | 已知后果是同一session旧refresh失效；Important 1 的旧logout删除新login_id是额外可避免的跨会话破坏，应并入 M3 第一批认证回归、接新 store 前修。 |
| auth_failure按IP的NAT连带429/并发余额上限 | 同意暂接受 | 防数据库查找洪泛与NAT公平性取舍已写清，有配置/IP归一化；M8负载测。 |
| anonymous refresh/logout同IP耗尽 | 同意暂接受 | 不能用未认证sid桶给攻击者精确定向能力；每次PK查找成本低，前端429保留session。 |
| XSS refresh可派生无限期PAT，PAT可继续派生 | 同意负责人产品裁定 | 风险不是30d内，README明确；若以后限期限，必须同时限PAT派生才能改变无界链。 |
| README手动撤销陌生PAT作为完整恢复的替代 | 不同意该表述 | 当前refresh还活可再造PAT；并入 M3 文档同步，明确 reset-password 全清退。已知异议不重算。 |
| set-email 与已验证旧邮箱密码的登录并发 | 同意既定语义 | `D:269` 的原理由是“签发给的是知道密码的人”；登录可在改邮箱后继续签新会话，set-email 不是恢复手段。源码的锁后复核只检查密码快照与 active，与明示规则一致；本轮未另强制这一交错，不把它报作凭证逃逸。 |
| PAT可deactivate、activate后PAT回生 | 同意现时裁定 | 保留Plane行为，安全恢复必须resetpassword；M3唯一管理员保护不能遗漏。 |
| 非prod暴露和ephemeralkey | 同意dev限制/M8镜像强制prod | 默认loopback+warning已经缓解，但M8部署验收必须proof，不是只写Dockerfile注释。 |
| signing keyrotation旧MAC失效、oldJWT失败门、盗用竞赛 | 同意限制 | 操作手册明确keyrotate不是凭证恢复；最小实现合理。 |
| argon参数提高后旧hash时差 | 同意负责人知情接受 | 现有保证明确仅当前参数；批量rehash无原password做不到，M8给数字后决定额外dummyhash。 |
| argon最多76MiB与登录每次DB查询 | 同意M8测 | 本机tests不代替容量测试，M8交接已经指出峰值/并发/NAT四项。 |
| bodyshape二次parse性能 | 同意M8测，但不涵盖新语义绕过 | parse成本可测；不同JSON语义不是性能接受项，M2必须修。 |
| bodyshape不支持date/x-go-type、futurearray/openobject/lexicographic errors/1e400 | 同意M4前扩展 | generator遇未支持schema失败，当前不静默错生成；M4业务采用前先补schema/test；Critical 1 与此无关。 |
| sqlc正则不覆盖quoted ALTER、DROP等 | 同意仅限当前 M2，M3 第一份跨模块迁移前需落实 | 当前四业务表同模块无跨模块DDL实际入口；本次变异已证实M3不能把守卫当完整ownerpolicy。 |
| sqlc PG17parser、River0.x锁版本 | 同意 | 版本锁和升级新migration规则明确，不改历史DDL；M8升级验证。 |
| 到workspace创建/邀请后仍404 | 同意 | M2明确只验资料步骤和挂载不请求旧接口；M3移交列出await链防未处理拒绝。 |
| P1 addrfile临时残留/全局测试collector/needsToken只认Bearer | 同意 | 测试限定临时路径、当前只有Bearer；后续authscheme加入需扩contract，但不是生产秘密泄漏。 |
| P1 sqlc checker单函数、Go/JS主干重复、warnIfExposed仅单测 | 同意 | 边界小且规则有对照测试；不为形式抽象强拆；环境warning不值得单独重启集成。 |
| P1 BodyCases只首nestedobject、DBdump含测试数据 | 同意有限覆盖，建议随新schema扩大 | contract测试不能证明所有树节点；严格边界新绕过应加明确回归。数据库fixture随机隔离无生产数据。 |
| P1 rollback FATAL57P01依赖内核缓冲 | 同意暂留 | 当前真连DB验证，不是无效测试；发生flaky再改断言/驱动注入，不删核心rollback保证。 |
| P2 AllowAll重复相同bucket/key可能负余额 | 同意当前无调用；M3新调用前拒重复或文档化 | public helper隐藏前提可致未来拒绝服务；当前生产调用组合各异，没有现时绕过路径。 |
| P2未fuzz refresh、重试重签access、account并删500 | 同意 | refresh表格/真实重放已覆盖；v0无accountdelete；fuzz可后补但不能把无fuzz当功能缺陷。 |
| P3a623行契约例外/PAT无限数量/timezones系统优先embeddedfallback | 同意 | 按模块合约不机械400行；PAT无限是显式产品风险；容器embedded防缺tzdata；submicroseconds统一截UTC有测试。 |
| P3b锁前取now导致audit时间先于实际lock/可用PAT数等待中过期 | 同意低风险，M3审计前重审 | 不形成凭证绕过，但会使审计因果不直观；以后在锁内读时钟可改。 |
| P3b Stop-before-Start panic/periodic无显式ID/CLI终端无pty自动测 | 同意现有单调用顺序 | bootstrap顺序有tests；创建worker当下唯一；终端路径手测/README，比引入复杂进程脚手架更合适。 |
| P3b backoff上限30s未突变覆盖/activate/deactivate幂等/setemail同值返回 | 同意当前 | 单元覆盖真实规则；不把所有理论mutant都当缺陷；CLI幂等是合理操作语义。 |
| P3b PHC巨大参数由DB写者控制 | 同意信任边界 | DB写者已有任意凭证/数据能力；HTTP无PHC写接口，非未认证DoS。 |
| RiverDB重启恢复只读代码、漏注册worker仅WARN | 同意M8/M4但须落实 | M8真实restart恢复；M4首异步任务测试worker已注册并处理，不把WARN等同failfast。 |
| River停机偶发ERROR/reindex遗留_ccnew/36s>Docker10s | 同意已记录行为 | exit0+无running+无连接泄漏才能叫正常；M8stop_grace与restart演练必须执行。 |
| P4旧RootStore reactions retained | 同意当前M2有限接受 | 已核剩reaction是本地filter/autorun无I/O；M3+每接一个store必须dispose与late-response测试，不能长期漏。 |
| P4旧async操作失败toast出现在新page | 同意 | 通知操作失败不写新账户数据；若globaltheme/reload则必须inSession，A9专测。 |
| P4globaltokenManager/publicClient例外、startSession先setlogin再newstore | 同意 | 可变认证authority应唯一；RootStore构造不I/O；futureconstructor不得加可能失败的副作用。 |
| P4最后登录覆盖浏览器记录但不撤旧server session | 同意 | 后端多会话是产品语义，不等同logoutall；绝对期限/cleanup防无限记录保留。 |
| P4泛化网络错误、CSPworker/frame fallbackself | 同意当前，M4/M7改前检 | 适合当前可达页；编辑器blob/iframe需要显式来源清点不可用unsafe扩开。 |
| P5DateDropdown键盘不能打开/Tab两套状态 | 不同意无限延后；接受明确M4关闭期限 | 日期选择属于M2PAT可达功能，键盘可用性债务真实；键盘行为不应只靠鼠标核对。M4公共日期组件第一项修，最好M2补。 |
| P5timezone按钮tabindex=-1 | 不同意无限延后；M3最迟 | 当前M2偏好页确受影响；M3换公共dropdown时有明确接收条件，不能每M继续转。 |
| P5CustomSelectEnter提交外层form | 同意当前验收语义 | 若选项确认立即submit不是用户期待，应在对应form新增键盘测试，当前没复现数据破坏。 |
| P5CSV英文/立即revokeBlob仅Chromium | 同意当前格式与浏览器声明 | `docs/v0/M2-auth/reviews/P5-web-account-review.md:92–93` 以点击时解析blob地址和列名属于文件格式为理由接受；只证明Chromium行为，未来扩浏览器支持时补测。 |
| P5同字段PATCH响应乱序 | 不同意作为通用最后选择保证 | 服务端最后写入与客户端最后收到不等于用户最后选择；P5 review:232 已明示假设，本次不计新发现；M3 新 store 采用前处理，实测与权衡见下。 |
| P5A11MessageChannel等全局timeout、无DOMunit | 同意当前 | Playwright是真DOM；等到测试全局timeout仍有明确失败上界，不是永挂。 |
| emoji warning精确白名单至M4 | 同意 | 已知相同上游warning被精确匹配，不是吞所有consolewarning；M4code splitting应remove。 |
| closeout A3仅mount范围/延时timer未等完 | 同意限定声明 | 设计只承诺挂载旧请求无回归，不能据此说任意用户动作都不会404；M3接入后扩行为。 |

实际页面的顺序反例：preferences 初始为 English；选简体中文，把第一条 PATCH 扣在发到后端之前；再选 English，第二条 PATCH 先到后端并返回 200；释放第一条，它随后也返回 200。发送顺序 `[zh-CN,en]`，服务端成功顺序 `[en,zh-CN]`；页面 `html.lang=en`，真实 GET `/api/v0/me/profile` 返回 `zh-CN`，reload 后变中文。两条请求都合法，前端丢旧响应不等于服务端丢旧写入。

处理权衡：①推荐同一 store 的修改串行提交，M3 新 store 采用前先落实，失败后仍释放队列；代价是连续操作等前一请求，收益是本轮反例不再发生。②加入服务端版本/冲突控制，可覆盖多客户端，但协议与交互成本对当前偏好较大。③不处理会在普通网络重排下使页面和保存值分叉，不建议无限期接受。客户端串行只解决同一 store，不宣称多个标签页或客户端全局有序。

## 7. 对负责人裁定的意见

四个决策点及 11.1 的产品目标可以保留。仅管理员改邮箱降低邮件基础设施范围；prod 默认关闭注册与 CLI 首账户实际可用；自助停用/管理员恢复的语义明确；角色和用途步骤确实删除。Important 1 是退出实现的问题，不是要求把退出改成结束全部登录。

- **完整恢复说明需要收紧，但不改变 PAT 产品选择**。README:121 与 M2 设计 8.5 把逐个撤销陌生 PAT 和管理员 reset-password 并列。`docs/v0/M2-auth/reviews/P3b-jobs-and-admin-review.md:107` 接受的理由是“它逐字照抄 M2 设计 8.5，是设计层面的事”。这解释了来源，没有证明恢复完成：被盗 RT 或其有效会话可以再建 PAT。建议仍保留当前全权限 PAT 与 activate 恢复行为，但明确手动撤销只处理所选 PAT；清退现存会话和 PAT 用 reset-password。本轮实际验证 reset 后全部已测凭证 401。替代是引入 PAT 期限/派生约束，但必须连同 PAT→PAT 一并设计，单独缩短期限无效；当前无须扩大 M2。
- **调高 argon2 参数后休眠账户时差**：同意负责人已知情的暂缓。本轮当前参数下两组各 50 次的分布重叠，不能否定旧参数差异。旧 PHC 的 Verify 与当前参数 DummyHash 耗时不同是明确代码事实。M8 应量化升级后的分布，再选择额外 dummy 工作或继续接受；当前立即统一跑两次哈希会提高正常登录成本，缺少负载数字时不建议加入。
- **CLI 只投递 River 客户端留 M4**：同意，M2 CLI 没有业务投递使用者。M4 交接 §1 保留原话和“没有使用者则再次原样转交”的关闭条件；如果 M3 的唯一管理员检查等先产生事务内任务，首次消费者必须提前接入，不能机械等到 M4。


## 8. M3–M8 出向交接质量和前瞻

路径均相对于仓库根目录 `/Users/xiaoruan/project/nerve-project`，行号对应 `9ef4a4a`。沿用第5节缩写；本节交接路径全部完整列出。

六份出向交接均为 `status: open`，符合接收方尚未实现的状态。共50个实质小节，每节均有关闭条件：M3为14节、M4为13节、M5为5节、M6/M7/M8各6节。下列“成立”评价交接是否可执行，不表示未来里程碑已经完成。Critical 1 的修复应在 M2 完成；Important 2 的部署角色方案最迟要在 M4 引入业务队列前落实，并进入 M8 的部署验收。

| 接收M/文件 | 判定 | 质量与前瞻 |
|---|---|---|
| M3：`docs/v0/M3-workspace-project/handoffs/M2-closeout.md` | 成立 | 邀请令牌需负责人确认、已有 SignupPolicy、landing/onboarding 取数、store分代、跨模块FK、workspace创建开关、停用端口、Authorizer/sqlc守卫、CSP、旧service/type、分页及键盘问题均有具体接收条件。停用端口若投递任务，:58 明确把 only-insert 客户端从M4提前到M3；不能机械等M4。 |
| M4：`docs/v0/M4-issue-core/handoffs/M2-closeout.md` | 成立 | :19 对 only-insert 明确三种结果：M3已实现则复核、本M出现命令则实现并验证事务、仍无使用者则原样再交。cursor、审计clock、结构检查延伸、cleanup、worker、editor/CSP、公共UI、store释放与clock fixture有独立关闭条件。Critical 1修复后的同一JSON解码语义必须被后续schema复用；Important 2需增加队列角色验收。 |
| M5：`docs/v0/M5-files/handoffs/M2-closeout.md` | 成立 | avatar/cover跨模块迁移归identity、上传突破1MiB/15s默认限额同时保留文件大小限制、services包最终删除、store分代、storage fixture均可执行。不得为上传方便全局关闭认证或结构边界。 |
| M6：`docs/v0/M6-cycles-modules/handoffs/M2-closeout.md` | 成立 | planning与issue跨模块写入/查询、FK删除图、retired store reaction、cycle取数、dropdown/copy各有关闭条件；周期取数由M3/M4归到实际接入它的M6，责任明确。 |
| M7：`docs/v0/M7-collaboration/handoffs/M2-closeout.md` | 成立 | 保存视图各种排序独立cursor载荷、最近访问模块归属、store分代、dropdown/copy、FK关系图、frame-src来源清点明确，未把created_at排序当作所有列表默认。 |
| M8：`docs/v0/M8-open-release/handoffs/M2-closeout.md` | 部分成立 | 原有audit覆盖、性能四项、镜像prod/36s停止宽限、数据库恢复、离线文档、webhook fixture与store分代闭合条件明确；Important 2表明仍需补“分离迁移/服务角色可启动、投递/执行任务、运行REINDEX维护”的部署验证，readyz200不能代替。 |

### 8.1 每项关闭条件

以下直接对应交接中的关闭条件，保留具体路径与行号，以便接收方逐项关闭；不以“写进review”替代原条款要求的代码、测试或实测结果。
#### M3：`docs/v0/M3-workspace-project/handoffs/M2-closeout.md`

| 事项（标题行） | 关闭条件行 | 可验收条件 |
|---|---|---|
| 1. 邀请：凭链接中的令牌接受；关闭注册时持有邀请的人仍可注册（交负责人确认）（:12） | `docs/v0/M3-workspace-project/handoffs/M2-closeout.md:18` | 负责人的确认写进 M3 设计；只凭邮箱匹配的接受被拒绝，凭有效令牌的成功；`auth.signup_enabled = false` 时，带有效邀请令牌的注册成功、不带的得到"注册已关闭"；这些都有接口版本和页面版本的故事。 |
| 2. 登录后的落点与新手引导的取数（:20） | `docs/v0/M3-workspace-project/handoffs/M2-closeout.md:28` | 完成引导的用户登录后落到上次的工作区，没有时按 Plane 的规则落到第一个工作区或 `/create-workspace`；新手引导的工作区、邀请两步用新接口取数；A3、A10 的断言不改仍通过；取数失败时页面没有未处理的拒绝（故事或单元测试）；`currentUserSettings` 不再调 Plane 的地址（改接或删除），上面四处不再因它失败；README"前端"一节的"M2 中看到的页面"一条随之改写；M3 的设计写明 M2 设计 3.1 挂载时的规则是否延伸到 M3 能到达的页面。 |
| 3. stores 按会话分代（规则在总体设计 7.7）（:30） | `docs/v0/M3-workspace-project/handoffs/M2-closeout.md:37` | M3 合并时，上面的 `git grep` 中 M3 领域的 10 处都已消失（或是只调公开操作的 `publicClient`）；`project_filter` 的反应随退役的一代释放，有单元测试（换代之后改路由，旧一代的反应不再运行）；M3 新加的 SWR 键带 `loginId`。 |
| 4. `profiles.last_workspace_id` 的外键（:39） | `docs/v0/M3-workspace-project/handoffs/M2-closeout.md:44` | M3 设计写明补或不补和理由；补的话迁移照上面命名，`server/migrations/schema_test.go` 核对约束名，差异清单二·按表登记。 |
| 5. `workspace_creation_enabled` 的执行（:46） | `docs/v0/M3-workspace-project/handoffs/M2-closeout.md:50` | 接口在关闭时拒绝，有错误码和测试；是否提供命令写进 M3 设计。 |
| 6. 停用的端口（决策点 3 已裁定为 A）（:52） | `docs/v0/M3-workspace-project/handoffs/M2-closeout.md:59` | 三件事在停用的事务里完成，唯一管理员时停用被拒绝，接口和命令两条路都有测试；`deactivateMe` 的错误码已声明，`apitest` 两个方向的核对通过；差异清单第四节"停用账户"一行更新；M3 的写入在全局加锁顺序中的位置写进 M3 设计。 |
| 7. 可空的引用字段（M2 设计 3.2）（:61） | `docs/v0/M3-workspace-project/handoffs/M2-closeout.md:65` | 这些字段在 M3 的接口描述中可为 `null`；`IUserLite` 改用生成的类型，或已改为可空。 |
| 8. 模块边界：`Authorizer` 端口、sqlc 与 `TestSQLCSchemaScope`（:67） | `docs/v0/M3-workspace-project/handoffs/M2-closeout.md:74` | M3 设计写明第 1 件的选择；第 2 件的每种写法都有一个反例，规则漏掉它时测试失败；`Authorizer` 在 `shared` 声明，由 M3 的模块实现，`shared` 仍只依赖标准库（架构测试规则 10）。 |
| 9. 物理删除与跨模块外键的关系图（M3 的部分）（:76） | `docs/v0/M3-workspace-project/handoffs/M2-closeout.md:80` | M3 设计有延伸后的图，每条指向 `users` 的外键写明去向；项目负责人的 `CASCADE` 有结论（照搬，或改为 `SET NULL`、`RESTRICT` 并登记差异）。 |
| 10. CSP：表情选择器的数据从本站提供（:82） | `docs/v0/M3-workspace-project/handoffs/M2-closeout.md:87` | 打开表情选择器时页面不请求 `cdn.jsdelivr.net`，没有 CSP 违规（故事的 `cspViolations` 为空，或浏览器核对）；CSP 没有为它放开外部来源。 |
| 11. M2 留下的 M3 调用和类型（:89） | `docs/v0/M3-workspace-project/handoffs/M2-closeout.md:97` | 三步用新接口；这三个方法改用生成的客户端或删除；`is_bot` 删除；工作区、项目设置的时区选择用这个接口；`owner` 的类型取自生成的类型；`plane-user-urls` 已收紧（或 M3 的 review 写明还有哪个旧调用留在 `/api/users/me/` 下、交给哪个 M）。 |
| 12. 页大小的规则移到 `shared`（:99） | `docs/v0/M3-workspace-project/handoffs/M2-closeout.md:103` | 第二个分页列表合并时，页大小的规则在 `shared`，两个列表都用它；M3 没有分页列表时，M3 的 review 写明，把本节写进下一个 M 的交接。 |
| 13. P5 改到、M2 的页面走不到的地方（接上时核对）（:105） | `docs/v0/M3-workspace-project/handoffs/M2-closeout.md:115` | M3 的浏览器核对逐条写明结果。 |
| 14. 下拉框和复制到剪贴板（M3 的部分）（:117） | `docs/v0/M3-workspace-project/handoffs/M2-closeout.md:126` | M3 的 review 逐项写明：`CustomSearchSelect` 能用 Tab 到达、用键盘打开；`member-options` 的列表在按钮旁展开；列出的调用方在浏览器中核对过；3 处复制都处理失败。 |

#### M4：`docs/v0/M4-issue-core/handoffs/M2-closeout.md`

| 事项（标题行） | 关闭条件行 | 可验收条件 |
|---|---|---|
| 1. 命令行的"只投递"River 客户端（负责人：做好记录，以后别漏了）（:12） | `docs/v0/M4-issue-core/handoffs/M2-closeout.md:19` | M3 已为停用的端口加入这个客户端时，M4 的 review 核对它合这一节（没有队列、不调用 `Start`，组合测试覆盖每个命令）并写明；否则，M4 加入了这样的命令时，命令的组合有只投递的客户端，一个测试在测试库上运行这条命令，核对 `river_job` 多了这条任务、事务回滚时没有。M4 结束时仍没有这样的命令，M4 的 review 写明，并把本节原样写进下一个 M 的交接，本交接才能改为 `closed`。 |
| 2. 列表的游标（:21） | `docs/v0/M4-issue-core/handoffs/M2-closeout.md:26` | 每种排序有自己的载荷和测试（翻页不重复、不遗漏，排序键相同时按 `id`）；改过的游标只能在调用者看得到的数据里换起点，有测试。 |
| 3. 自动归档读 `updated_at`（:28） | `docs/v0/M4-issue-core/handoffs/M2-closeout.md:31` | 写工作项的每个用例都用时钟写 `updated_at`，测试断言它等于固定时钟；自动归档的测试推进固定时钟，不等真实的时间。 |
| 4. 请求体检查的两处延伸和草稿发布（:33） | `docs/v0/M4-issue-core/handoffs/M2-closeout.md:42` | 两处延伸各有单元测试（`tags[2]` 排在 `tags[10]` 之前；开放对象里的 `1e400` 得到带路径的 400）；同一个坏的请求体，草稿发布与创建工作项给出相同的 400；`date` 的检查器已登记，不合格的日期得到带路径的 400，有测试；第一个 map 型 schema 出现时，`closedObject` 有例外分支和反例用例（或者本 M 的 review 写明没有用到）。 |
| 5. 60 天物理清理与删除关系图（M4 的部分）（:44） | `docs/v0/M4-issue-core/handoffs/M2-closeout.md:48` | 清理任务的集成测试覆盖软删除超过 60 天的 PAT 和带操作动态的工作项；M4 设计有延伸的图。 |
| 6. 定时任务和事件订阅者（:50） | `docs/v0/M4-issue-core/handoffs/M2-closeout.md:54` | 漏注册 worker 的变异让测试失败；事件订阅者的写法写进 M4 设计。 |
| 7. 编辑器的代码分割（:56） | `docs/v0/M4-issue-core/handoffs/M2-closeout.md:60` | `EMOJI_CHECK_WARNING` 和它的使用都删除，S2 和个人设置的故事仍断言控制台没有别的警告，并通过。 |
| 8. CSP：编辑器 callout 的默认表情图；`worker-src`、`frame-src`（:62） | `docs/v0/M4-issue-core/handoffs/M2-closeout.md:67` | 插入 callout、打开表情回应时没有 CSP 违规，页面不请求 `cdn.jsdelivr.net`；M4 的 review 写明编辑器是否用 `blob:` 的 worker，用的话 `csp.go` 只为它加了 `worker-src` 的来源，有测试。 |
| 9. 命令面板的主题命令（:69） | `docs/v0/M4-issue-core/handoffs/M2-closeout.md:72` | 命令面板的主题命令在应答成功之后才应用，失败时按 `code` 提示，有故事或浏览器核对。 |
| 10. 下拉框（M4 的部分）（:74） | `docs/v0/M4-issue-core/handoffs/M2-closeout.md:83` | M4 的 review 逐项写明：`DateDropdown` 能用键盘打开，只有一份状态；7 个下拉框的列表在按钮旁展开；列出的调用方在浏览器中核对过。 |
| 11. 复制到剪贴板（M4 的 6 处）（:85） | `docs/v0/M4-issue-core/handoffs/M2-closeout.md:88` | 6 处都处理失败，经 `t()` 提示，像 `api-token/modal/generated-token-details.tsx`。 |
| 12. stores 按会话分代（规则在总体设计 7.7）和 M2 留下的调用（:90） | `docs/v0/M4-issue-core/handoffs/M2-closeout.md:98` | `IssueRootStore` 的反应随退役的一代释放，有单元测试；上面的 `git grep` 中 M4 的 13 处都已消失；三处同步读取不发请求；`getUserProfileIssues` 改接或删除。 |
| 13. e2e 的 `clock.ts`（M0-P6 交接剩下的一项）（:100） | `docs/v0/M4-issue-core/handoffs/M2-closeout.md:103` | `clock.ts` 按这个写法加入，自动归档的故事用它，不等真实的时间。 |

#### M5：`docs/v0/M5-files/handoffs/M2-closeout.md`

| 事项（标题行） | 关闭条件行 | 可验收条件 |
|---|---|---|
| 1. 头像和封面（:12） | `docs/v0/M5-files/handoffs/M2-closeout.md:18` | 两列和外键照上面的两个迁移加入，差异清单二·按表更新；两个地址返回签名地址；两处上传控件用新协议；`node tools/keywords.mjs` 通过，`plane-user-assets` 仍挡住 Plane 的地址。 |
| 2. 上传与按路由的中间件（:20） | `docs/v0/M5-files/handoffs/M2-closeout.md:26` | 大于 1 MiB、耗时超过 15 秒的上传都成功（测试）；超过 `file_size_limit` 的被拒绝；整程序测试（`server/internal/bootstrap/contract_test.go`）仍通过；CSP 只为存储放开来源，有测试。 |
| 3. 删除 `@nerve/services`（:28） | `docs/v0/M5-files/handoffs/M2-closeout.md:34` | `web/packages/services` 不存在，任何 `package.json` 都不依赖 `@nerve/services`；`make knip`、`make lint-web`、`make test-web` 通过；前端改动清单 3.1 的那一行改为已完成。 |
| 4. stores 按会话分代（规则在总体设计 7.7）（:36） | `docs/v0/M5-files/handoffs/M2-closeout.md:39` | 这 8 处都已消失；M5 新加的 SWR 键带 `loginId`。 |
| 5. e2e 的 `storage.ts`（M0-P6 交接剩下的一项）（:41） | `docs/v0/M5-files/handoffs/M2-closeout.md:44` | `storage.ts` 照这个写法加入，上传的故事用它核对存储里的文件。 |

#### M6：`docs/v0/M6-cycles-modules/handoffs/M2-closeout.md`

| 事项（标题行） | 关闭条件行 | 可验收条件 |
|---|---|---|
| 1. 迭代跨模块的写入和查询（:12） | `docs/v0/M6-cycles-modules/handoffs/M2-closeout.md:16` | M6 设计列出每个跨模块的写入和查询，以及它走端口还是例外；例外在 `TestSQLCSchemaScope` 中写明理由。 |
| 2. 物理删除与跨模块外键的关系图（M6 的部分）（:18） | `docs/v0/M6-cycles-modules/handoffs/M2-closeout.md:22` | M6 设计有延伸后的图；迭代负责人的 `CASCADE` 有结论（照搬，或改为 `SET NULL`、`RESTRICT` 并登记差异）。 |
| 3. stores 按会话分代（规则在总体设计 7.7）（:24） | `docs/v0/M6-cycles-modules/handoffs/M2-closeout.md:29` | 两个反应随退役的一代释放，有单元测试；这 2 处都已消失；M6 新加的 SWR 键带 `loginId`。 |
| 4. 周期下拉框的取数（:31） | `docs/v0/M6-cycles-modules/handoffs/M2-closeout.md:37` | M6 的 review 写明三点各自的结论（保留、修正并有测试）；并发的两次取数不会让旧的回答覆盖新的。 |
| 5. 下拉框（M6 的部分）（:39） | `docs/v0/M6-cycles-modules/handoffs/M2-closeout.md:47` | M6 的 review 逐项写明：两个下拉框的列表在按钮旁展开；列出的调用方在浏览器中核对过。 |
| 6. 复制到剪贴板（M6 的 3 处）（:49） | `docs/v0/M6-cycles-modules/handoffs/M2-closeout.md:52` | 3 处都处理失败，经 `t()` 提示，像 `web/apps/web/core/components/api-token/modal/generated-token-details.tsx`。 |

#### M7：`docs/v0/M7-collaboration/handoffs/M2-closeout.md`

| 事项（标题行） | 关闭条件行 | 可验收条件 |
|---|---|---|
| 1. 保存视图的列表：每种排序一个游标载荷（:12） | `docs/v0/M7-collaboration/handoffs/M2-closeout.md:17` | 每种排序有自己的载荷和测试（翻页不重复、不遗漏）；改过的游标只能在调用者看得到的数据里换起点，有测试；跨模块的联表有结论。 |
| 2. stores 按会话分代（规则在总体设计 7.7）（:19） | `docs/v0/M7-collaboration/handoffs/M2-closeout.md:23` | 这 3 处都已消失；M7 新加的 SWR 键带 `loginId`；最近访问的操作在它自己的模块文件里。 |
| 3. 下拉框（M7 的部分）（:25） | `docs/v0/M7-collaboration/handoffs/M2-closeout.md:32` | M7 的 review 逐项写明：两个下拉框的列表在按钮旁展开；列出的调用方在浏览器中核对过。 |
| 4. 复制到剪贴板（M7 的 4 处）（:34） | `docs/v0/M7-collaboration/handoffs/M2-closeout.md:37` | 4 处都处理失败，经 `t()` 提示，像 `api-token/modal/generated-token-details.tsx`。 |
| 5. 物理删除与跨模块外键的关系图（M7 的部分）（:39） | `docs/v0/M7-collaboration/handoffs/M2-closeout.md:43` | M7 设计有延伸后的图，每条指向 `users`、工作项的外键写明去向。 |
| 6. CSP 的 `frame-src`（:45） | `docs/v0/M7-collaboration/handoffs/M2-closeout.md:48` | M7 的 review 写明收集箱或视图是否嵌入 iframe；嵌入的话，`server/internal/platform/webui/csp.go` 只为它加那个来源，有测试。 |

#### M8：`docs/v0/M8-open-release/handoffs/M2-closeout.md`

| 事项（标题行） | 关闭条件行 | 可验收条件 |
|---|---|---|
| 1. 接口调用日志挂在限流之后（:12） | `docs/v0/M8-open-release/handoffs/M2-closeout.md:17` | 中间件在限流之后，`API.Middlewares` 的顺序测试核对；被限流、认证失败的请求在它之前就结束、不进日志，M8 设计写明这与差异清单的"记录所有写请求"怎样对上。 |
| 2. 性能和内存的实测（:19） | `docs/v0/M8-open-release/handoffs/M2-closeout.md:30` | M8 的 review 有每一项的数字、测法和结论；超出预期的写明处理（调参、改默认值，或登记风险）。 |
| 3. 部署（:32） | `docs/v0/M8-open-release/handoffs/M2-closeout.md:37` | 镜像的环境里有 `NERVE_ENV=prod`；部署文件的停止宽限期不短于停机的最坏时间（或写明调小了哪些期限）；数据库重启的核对结果写进 M8 的 review。 |
| 4. 对外接口文档页不从 CDN 加载脚本（:39） | `docs/v0/M8-open-release/handoffs/M2-closeout.md:42` | 文档页不请求第三方的来源；页面带 CSP 时没有违规。 |
| 5. e2e 的 `webhook.ts`（M0-P6 交接剩下的一项）（:44） | `docs/v0/M8-open-release/handoffs/M2-closeout.md:47` | `webhook.ts` 照这个写法加入，Webhook 的故事用它核对收到的负载和签名。 |
| 6. stores 按会话分代（规则在总体设计 7.7）（:49） | `docs/v0/M8-open-release/handoffs/M2-closeout.md:52` | M8 领域没有模块级的带令牌 service 或客户端（`git grep -n -E '^(export )?const [A-Za-z]+ = new [A-Za-z]+Service\(' -- web/apps/web` 中没有 M8 的）；M8 新加的 SWR 键带 `loginId`。 |

### 8.2 后续最需要防止的四种集成遗漏

1. **M3首次跨模块DDL使正则守卫失效。** `docs/v0/M3-workspace-project/handoffs/M2-closeout.md:70–74` 已列 quoted ALTER、CREATE INDEX/TRIGGER、DROP 的反例。当前M2只有identity业务表，不能将这一接受理由延续到M3；应随第一份跨模块迁移完成守卫变异验证。
2. **CLI的事务投递缺少客户端。** `docs/v0/M4-issue-core/handoffs/M2-closeout.md:19` 要求真实测试库核对成功后有river_job、回滚后没有；M3停用端口如开始发任务，按M3交接:58提前。Important 2另要求所用服务角色具备队列DML/sequence和维护所需权限。
3. **旧store局部reaction变成跨账户I/O。** M3:37、M4:98、M6:29 都明确retired generation释放和sessionbound client；每接一个新store就验证迟应答与换账户场景，不能仅清掉模块级new Service字面命中。Minor 1也说明主题复位应统一处理；其具体要求是 `docs/v0/M2-auth/specs/P4-web-auth.md:163`，对应 `web/apps/web/core/lib/store-context.tsx:41–50`，并非总体设计7.7对默认主题的明文承诺。
4. **上传放宽破坏全局接口边界。** `docs/v0/M5-files/handoffs/M2-closeout.md:26` 要求超过1MiB/15秒的合法上传成功、超file_size_limit拒绝、整程序契约测试继续通过、CSP只放存储来源。Critical 1修复后的结构检查与认证仍应适用于新增路由。

对现有交接的新增建议应是小而明确的关闭条件：M4首业务任务前验证River分离角色；M8部署演练用该角色覆盖任务执行与REINDEX。其余长期规则继续引用总体设计7.7及模块边界章节，避免在各交接复制整份M2设计并再次漂移。

## 9. 做得好的地方

- 刷新令牌把旧代真实性与当前 secret 证明分开；真/伪旧代实际请求、并发双续期、绝对期限变异都支持核心设计，没有靠可猜的 session ID 授权撤销。
- 账户锁不是测试中的假串行：双向交错观察真实 PostgreSQL 锁等待，NO KEY UPDATE 既阻止凭证状态竞争又允许关联插入；删锁会失败。
- 平台与模块边界、UUID 生成类型、sqlc 查询范围、未声明 problem code 的门禁都有本轮失败反例。端口在使用方 app，shared 小，没有为复用把 identity 依赖倒灌平台。
- 当前 M2 store 接收本会话 ApiClient，RootStore 换对象；旧响应留在旧对象。401 重试、SessionChangedError、PAT 原文白名单投影均有能失败的测试。PAT 跨账户迟应答候选经两种真实交错没有成立。
- e2e 同时读数据库、验证真实凭证与页面；A8/A12 改坏后不是仅靠 HTTP 状态仍过关。失败工件的 trace/screenshot/log/dbdump 职责明确，后续领域可以扩展现有 fixture，无须新造框架。
- prod 首账户、五次 down 再 up、River 锁定迁移字节、CSP 内联脚本哈希都实测成立；交接列出具体文件和关闭条件，已知限制通常诚实记录。

## 10. 处理建议清单

### M3 开始前必须做

- [ ] **Critical 1**：在统一请求入口拒绝重复对象键，消除检查与最终解码语义差异；真实 handler 回归须证明本文 400→200 反例恢复为 400、数据库不变。检查转义等价键、嵌套对象、大小写和 null；明确无效 UTF-8 策略。修后再跑 gen-check、相关 Go 契约/HTTP 测试与全量既有门禁。

### 可以并入 M3

- [ ] **Important 1**：放在 M3 第一批认证回归、接新 store 前，建议尽早修。退出最后一次 await 后按原 login_id 重新核对；暂停旧 tab、让真实租约到期再登录新账户的回归必须保留新记录。剩余无 CAS 的竞争窗口仍按既有 lease 限制说明。
- [ ] **Minor 1**：集中实现当前会话的默认/资料主题应用，去掉按钮层的重复补丁；以真实 DOM 验证停用和自动会话结束。
- [ ] **Minor 2**：为 A8–A10 提取小型共享业务断言，差异用参数；保留现有 page/API 覆盖，不写通用测试框架。
- [ ] **Minor 3**：修正文档的 bodyshape 依赖和 identity 操作数，保留历史 plan。
- [ ] **已知 PATCH 顺序事项（不是新发现）**：新 store 采用前明确同一 store 的 PATCH 顺序；本轮真实语言切换已经否定“发出顺序等于服务端处理顺序”。推荐串行提交，失败也释放队列；回归同时检查页面值、GET 值和刷新后值。
- [ ] **已知恢复说明**：按 §7 说明 reset-password 与逐个撤销 PAT 的能力差异；sqlc 所有权语法守卫、停用端口中的唯一管理员检查、timezone 键盘及首次接入 store 的 disposer 按 M3 既有交接关闭。

### 随后续某个 M 处理

- [ ] **Important 2，M4 第一条业务 River 任务前**：补分离角色最小授权和真实角色 smoke，覆盖启动、任务执行、REINDEX；不要只加 goose SELECT。M8 部署验收再次验证该角色方案。
- [ ] **M4，已知交接**：CLI 只投递客户端、DateDropdown 单一打开状态和键盘、命令面板主题失败/会话保护、编辑器拆包和 CSP、clipboard 拒绝；新 schema 用到前补 bodyshape 支持，不能靠运行时忽略未知类型。
- [ ] **M5–M7，已知交接**：上传的大小/期限路由策略、排序游标载荷、跨模块事务/FK、每会话 client/disposer、全局副作用的本代校验，按 §8 接收条件做，不在 M2 预造未用模块。
- [ ] **M8，已知交接**：生产镜像强制 prod、停机宽限覆盖约 36s、River 数据库恢复和维护中断、真实代理/IPv6、哈希与 bodyshape 容量、支持浏览器的挂起/恢复与 Blob 下载实测。参数调整用数据，不凭本机短测放宽安全边界。

### 不建议做

- [ ] 不为 **Critical 1** 给每个领域补一份字段检查，或手改生成 handler；那会复制契约并掩盖平台根因。
- [ ] 不为 **Important 1** 增加选主/心跳框架，也不只为这一竞态取消已批准的 HTTP 降级。先复用现有 login_id 校验；无证据支持扩大改造。
- [ ] 不以运行账户拥有全部数据库对象替代 **Important 2** 的授权说明；也不因 readyz 当前只检查 DB/迁移就强行引入新的后台健康框架。
- [ ] 不把 **Minor 1** 继续分散修到每个退出按钮，不为 **Minor 2/3** 建通用断言或统计生成系统。
- [ ] 不把关键词正则升级成全仓任意表达式语义分析器，也不为所有后续 schema/排序提前扩平台。守卫写清能拦的字面范围；有实际新用法时补局部规则与反例。
- [ ] 不把已证伪的 PAT 跨会话候选、设计允许的当前代 RT 改标签、已登记的刷新响应丢失或旧参数耗时再次计成新漏洞。

交付清理：本次创建的 nerve/浏览器进程和临时数据库均已清理；变异副本恢复后确认干净，再删除外部临时脚本、副本和工件。没有停止、重建或修改其他项目容器，没有停止或重建 `nerve-dev-db-1`，没有触碰其他分支、worktree、`plane/`、`refer/`。最终唯一持久改动是本报告，只提交本地 `main`，不 push。
