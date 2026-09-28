# M2/收尾 closeout：设计说明（spec）

| 项 | 内容 |
|---|---|
| 里程碑 | M2 账户认证（`docs/v0/M2-auth`） |
| 日期 | 2026-09-28 |
| 基点 | `d97c513`（Merge M2/P5）；M2 从 `559c3c6`（M1 收尾的合并）开始 |
| 上级 | [M2 设计](../M2-design.md) 第 12 节"收尾 `closeout`"、13.1、13.2、第 14、15、16 节、3.20、7.9、7.10 |
| 计划 | [plans/closeout.md](../plans/closeout.md)，9 个 Task |
| 原型 | `$M2TMP/closeout/proto`（`d97c513` 的克隆），T0–T8 逐个应用计划的块，每个 Task 之后跑它的检查，最后跑全部门禁（附录 A） |

`$M2TMP` 是 `/private/tmp/claude-501/-Users-xiaoruan-project-nerve-project/99d2bc1d-fdaf-4b92-a590-29b89514572b/scratchpad/nerve-m2`。

---

## 1. 目标

收尾关闭 M2：核对 M2 承诺的都成立，关闭 M2 收到的交接，写好 M2 给 M3–M8 的交接，把 M2 标为完成。它几乎全是文档和核对，只在核对找到真实缺口的地方改代码（一处：A3 的故事，3.2）。

完成之后：

- 10 份收到的交接逐项有结论，`status: closed`；M0-P2、M0-P6 剩下的三项转交到接收的 M。
- 3.20 逐行核对过；没有同步的四处补上，各记为所在 Phase 的漏项（2.2）。差异清单按第 4 节逐列核对过（2.3）。
- M3–M8 各有一份 `handoffs/M2-closeout.md`：13.2 的每一行落在每个接收 M 的文件里，多个 M 的行拆开；13.2 之外找到的遗留也在里面（2.4、3.1）。负责人对 M4 只投递客户端的要求是 M4 交接的第 1 节。
- 长期的前端会话规则和负责人未决的 argon2 事项在总体设计里有唯一的落点（第 4 节 D2、D4）。
- 关键词守卫进入 `M2/closeout`，例外只剩跨 M 的三条（2.5）。
- 第 14 节的每一项在收尾的头上重新证明（2.6）；状态翻转由控制者在收尾 review 的提交里做（D1）。
- 7.10 的估计与实际逐项对比，写明对 M3 以后的含义（附录 B）。

---

## 2. 分诊

### 2.1 M2 收到的 10 份交接

| 交接 | `d97c513` 上的状态 | 结论 | 收尾做的 |
|---|---|---|---|
| M0-P1-sqlc-cgo | done | 两条都在 P1 完成 | 核对：`Makefile` 的 `SQLC` 是 `CGO_ENABLED=0`；五个迁移没有 `uuidv7()`；`make gen-check` 没有差异。关闭 |
| M0-P2-platform-notes | open | 第 1、3、4、6、7、8 条 P1；第 2 条认证 P1、限流 P2；第 5 条期限 P1、River 与停机 P3b。剩第 2 条的接口调用日志 | 转交 M8（M8 交接第 1 节）；停机最坏约 36 秒的部署后果转交 M8（第 3 节）。关闭 |
| M0-P3-api-codegen-notes | done | 五条在 P1、P3a 完成 | 核对：uuid 的两个测试、三个错误出口、写法检查、模块入口随 `make test` 通过。第 5 条的"一个模块一个文件"写进总体设计 6.3（D3）。关闭 |
| M0-P4-schema-conventions | done | 两条在 P1 定下，P3a 照做 | 核对：`schema_test.go` 通过；五个迁移没有 `DEFERRABLE`、`*_like`；River 的表按差异清单是原样导出。关闭 |
| M0-P5-frontend-api-notes | done | P2（安全响应头）、P4（CSP、前端改调新接口） | 核对：S2 只请求 `GET /api/v0/instance`；`TestOnlyPagesHaveAContentSecurityPolicy` 通过。关闭 |
| M0-P6-e2e-notes | open | P1–P4 完成除最后一节以外的全部 | fixture 的延伸转交：`clock.ts` → M4（第 13 节）、`storage.ts` → M5（第 5 节）、`webhook.ts` → M8（第 5 节）。关闭 |
| M1-P2-trim-content | done | P3a（接口）、P4（前端） | 核对：`Theme` 五个值；`git grep -n IUserTheme -- web` 没有输出。关闭 |
| M1-P3-trim-platform | done | P3a、P3b、P4、P5 | 按原文"收尾再核对一次"：不再读的字段、不再调用的地址不在 `api/` 里（下面的命令，退出码 1）；其余逐条核对。关闭 |
| M1-P4-router-native | done | P4 | 核对：`isValidNextPath` 的单元测试、A3 的四个不合格 `next_path`。关闭 |
| M1-closeout | done | P4、P5 | 核对：`make lint-web` 上限相等；`no-unneeded-ternary` 0 条。关闭 |

M1-P3 的核对（`d97c513`）：

```
git grep -n -i -E 'is_password_autoset|last_login_medium|has_marketing_email_consent|billing_address|has_billing_address|property_change|state_change|issue_completed|[^_]mention|provider|notification|email-check|magic|forgot|reset-password|set-password|generate-code|change-email|instance-admin|is_instance_admin|google|github|gitlab|gitea|[^0]/auth/' d97c513 -- api
→ 没有输出，退出码 1
git grep -c -E '^[[:space:]]+operationId:' d97c513 -- api/dist/openapi.yaml
→ d97c513:api/dist/openapi.yaml:15
```

15 个操作、12 个路径，都在 `/api/v0/auth/`、`/api/v0/me`、`/api/v0/api-tokens/{token_id}`、`/api/v0/instance`、`/api/v0/timezones` 下。模式里的 `[^0]/auth/` 排除 `/api/v0/auth/`，`[^_]mention` 排除别的单词中的 `_mention`。

每份交接末尾加的 `## 处理结果（M2/收尾）` 全文在计划的 Task 7。

### 2.2 3.20 逐行核对

逐行打开 3.20 写的文档和位置，看它在 `d97c513` 上是否写着这一行说的内容。行号是 `d97c513` 的。"✓"后面是找到它的位置。

| Phase | 文档 | 位置 | 结果 |
|---|---|---|---|
| P1 | 总体设计 | 3.1 | ✓ `v0-design.md:149`（注册返回令牌，账户由 `GET /api/v0/me` 读取） |
| P1 | 总体设计 | 3.5 | ✓ `:200`（四个平台码）、`:201`（400 与 422 的分层）、`:202`（`FieldError` 带 `code`）、`:203`（`x-problem-codes`） |
| P1 | 总体设计 | 4.1 | ✓ `:225`（刷新令牌的格式、MAC 密钥派生、只存当前一代的哈希） |
| P1 | 总体设计 | 4.2 | ✓ `:235`（prod 默认关闭注册，第一个账户用 `nerve users create`） |
| P1 | 总体设计 | 5.5 | ✓ `:320`（审计时间列由用例的时钟写入） |
| P1 | 总体设计 | 5.6 | ✓ `:330`（改别的模块的表的迁移归被改表的模块） |
| P1 | 总体设计 | 6.2 | ✓ `:359-361`（`shared` 的内容；端口由使用方声明） |
| P1 | 总体设计 | 6.2（`module.go`） | ✓ `:387-389`（`Register(router, api)`、`PublicOperations()`、`Authenticator()`） |
| P1 | 总体设计 | 6.3 | ✓ `:407`（平台不依赖 `shared`）、`:408`（sqlc 按模块限定） |
| P1 | 总体设计 | 6.4 | ✓ `:419-430`（按路由的中间件及其顺序，参数先绑定）、`:434`（`COMMIT` 不受请求期限） |
| P1 | 总体设计 | 8.2 | ✓ `:639`（失败时的 trace、截图、日志、快照，不录像） |
| P1 | 总体设计 | 8.2（fixture 目录） | ✓ `:624-628`（`server.addr_file`、每个进程一个连接池、`assert/`、`auth.ts`） |
| P1 | M0 设计 | 3.1 | ✓ `M0-design.md:142-148`（平台不依赖 `shared`；模块入口；`internal/shared` 由 P1 建立） |
| P1 | M0 设计 | 3.3 | ✓ `:178`（按路由的中间件及其顺序） |
| P1 | M0 设计 | 3.3（中间件以外） | ✓ `:167`（`httpserver.Router`）、`:176`（健康检查的日志 DEBUG）、`:180`（平台码）、`:181`（生成代码的错误出口） |
| P1 | M0 设计 | 3.5 | ✓ `:199`（`//go:embed sql/*.sql`）、`:206`（迁移从 M2 开始，`schema_test.go`） |
| P1 | 差异清单 | 一 B | ✓ `plane-diff.md:30`（`sessions` 替换模型） |
| P1 | 差异清单 | 二·全局 | ✓ `:62-66`（`ON DELETE`、不用 `DEFERRABLE`、改名、不建 `*_like`、审计时间列、`jsonb` 的 CHECK） |
| P1 | 差异清单 | 二·按表 | ✓ `:71-82`（`users`）、`:83-93`（`profiles`）、`:102`（`auth_sessions`，按实际的列改写）；逐列见 2.3 |
| P1 | 差异清单 | 三 | ✓ `:139-140`（错误码、请求体） |
| P1 | 差异清单 | 四 | ✓ `:162-166`（密码规则、注册的前提、注册默认、未知字段、密码哈希过载） |
| P2 | 总体设计 | 3.5 | ✓ `:200`（`rate_limited`） |
| P2 | 总体设计 | 3.6 | ✓ `:207-211`（桶、突发、默认值、失败闸门） |
| P2 | 总体设计 | 4.1 | ✓ `:225`（30 天的绝对期限） |
| P2 | 总体设计 | 4.2 | ✓ `:230`（退出只结束当前会话）、`:232`（重复使用检测只在旧令牌确属这个会话时撤销） |
| P2 | 总体设计 | 6.4 | ✓ `:419`（固定链四个中间件）、`:430`（限流） |
| P2 | M0 设计 | 3.3 | ✓ `M0-design.md:177`（第四个中间件） |
| P2 | 差异清单 | 四 | ✓ `:167-169`（登录时邮箱不存在、会话的期限、限流） |
| P3a | 总体设计 | 3.1 | ✓ `:149`（修改密码、停用返回 204） |
| P3a | 总体设计 | 3.4、6.2 | ✓ `:181`（封套由 `shared` 定义，载荷由各列表定义）、`:360`（`shared` 含游标的封套） |
| P3a | 总体设计 | 4.2 | ✓ `:230`（修改密码、停用结束的会话，PAT）、`:231`（账户行锁） |
| P3a | M0 设计 | 3.2 | ✓ `M0-design.md:152` |
| P3a | M0/P3 spec | 第 7 节 | ✓ `P3-api-contract.md:419`（分页的公共组件由 M2/P3a 加入） |
| P3a | M0 设计 | 3.7（`:248`） | ✓ `:250`（google/uuid 只允许经由 `runtime`）、`:251`（生成文件不得引用 `openapi_types.UUID`） |
| P3a | M0/P3 spec | 2.8、第 7 节 | ✓ `P3-api-contract.md:289`、`:411` |
| P3a | 差异清单 | 二·按表 | ✓ `:94-101`（`api_tokens` 逐列） |
| P3a | 差异清单 | 三 | ✓ `:141`（`/api-tokens/{id}`） |
| P3a | 差异清单 | 四 | **漏（P3a）**：4.6 的"退出、修改密码、停用后旧凭证失效"登记为"修改密码、停用之后的旧凭证"（`:171`），少了"退出"；P3a spec 第 3 节第 21 条要求照登。其余 P3a 的行 ✓ `:170`、`:172`、`:175-179`。收尾改行名（T3） |
| P3b | 总体设计 | 4.2 | ✓ `:230`（恢复后 PAT 可用，重置撤销全部 PAT）；"随 M2/P3b 加入"在总体设计、差异清单、README 都已不在（`git grep` 没有输出） |
| P3b | 差异清单 | 二·按表 | ✓ `:103`（River 的表） |
| P3b | 差异清单 | 四 | ✓ `:160-161`、`:173-174` |
| P4 | 总体设计 | 4.3 | ✓ `:238`（`login_id`）、`:241`（localStorage 租约）、`:242`（只有续期 401 或重发仍 401 才结束，"会话暂不可用"） |
| P4 | 前端改动清单 | 3.2 | ✓ `frontend-changes.md:215-216`（两行已完成，M2/P4） |
| P5 | 前端改动清单 | 3.1 | ✓ `:201`（M2 一行已完成） |
| 收尾 | 总体设计 | 9.4 | **漏（设计文档的合并 `fd697367`，与 P1 同一次）**：`:691` 是"未开始"和"—"。M2 从 2026-09-25 起进行中，M0、M1 两行的格式是状态加设计文档的链接。收尾先改为"进行中"和链接（T3），"已完成"在收尾 review 的提交里（D1） |
| 收尾 | 以上全部 | — | 本表；差异清单见 2.3 |

**表外找到的漏同步**（3.20 的规则是"每处偏离，在实现它的那个 Phase 的同一次合并中同步"，下面两处是偏离，但表里没有写）：

| 位置 | 漏的 Phase | 说明 | 收尾 |
|---|---|---|---|
| M0 设计 3.7 的规则 4、6、8（`M0-design.md:243,245,247`） | P1（`448a138a`） | 架构测试的规则在 P1 改了三条（`server/internal/archtest/rules_test.go:40,42,44`）：平台也不导入 `internal/shared`；生成的代码只能被它自己的适配器导入（推广到 sqlc，M0-P2 交接第 6 条）；测试工具加上 `clocktest`。M0 设计 3.1 的包职责表同步了，3.7 的规则原文没有 | T3 改三条规则 |
| 总体设计 7.1"暂时使用"（`v0-design.md:535`） | P5 | 仍写"web 中的令牌设置页和文件工具函数依赖它；M2（PAT）和 M5（文件）对接新接口时，将它删除"。P5 删掉了令牌服务和 axios 基类，前端改动清单 1 节和 3.1 已同步，总体设计没有 | T3 改写这一条 |

**不改的**：M0/P3 spec 第 3 节差异 2（`P3-api-contract.md:346`）和第 5 节（`:389`）仍写"第一个列表接口（M3）"。它们是 P3 当时的裁定和范围，3.20 只要求改第 7 节的交接表（已改，`:419`）；照旧留作记录。

**README 的 8.7**（第 14 节第 10 项）：`d97c513` 的"部署"一节逐条都在：`NERVE_ENV=prod`（`README.md:100`），签名密钥和 `openssl`、换钥的后果（`:101-108`），迁移角色（`:123`），`trusted_proxies`（`:109`），HTTPS 与租约（`:110`），第一个账户和注册暴露邮箱（`:112`），管理命令（`:113-120`），令牌泄露后的恢复（`:121`），密钥扫描（`:122`），常见密码名单的第三方声明（`:137`）。

### 2.3 差异清单按第 4 节逐列核对

| 表 | M2 设计 | 差异清单（`d97c513`） | 结果 |
|---|---|---|---|
| `users` | 4.2：保留 10 列，逐列的类型、约束和差异；删除 30 列（Django 5、登录记录 9、砍掉的功能 10、机器人 2、旧 URL 2，另有 2 列延到 M5） | `:71-82`：保留 10 列，主键和唯一约束的名字；`email`、`password`、`first_name`/`last_name`、`display_name`、`user_timezone`、`is_active` 和审计时间列逐列；删除的 28 列分四行，头像两列写明 M5 | ✓ 逐列一致。`id` 的"不设默认值"在二·全局的"ID 改由应用生成" |
| `profiles` | 4.3：保留 11 列；删除 18 列 | `:83-93`：11 列；`user_id`、`theme`、`is_tour_completed`/`is_onboarded`、`onboarding_step`、`language`、`start_of_the_week`、审计时间列逐列；删除 2 + 7 + 9 列 | ✓。`last_workspace_id` 照搬、不是外键，没有差异，不登记（是否补外键在 M3 的交接第 4 节） |
| `api_tokens` | 4.4：保留 12 列；`token` 改 `token_hash`；删除 5 列；列表的索引 | `:94-101` | ✓ |
| `auth_sessions` | 4.5：12 列、表级约束、两个索引、旧代不存 | `:102`（12 列逐个，`auth_sessions_revoked_consistent_check`，两个索引，旧代由 HMAC 标签认出）；`:30`（一 B 替换模型） | ✓ |
| River 的表 | 3.15：主线第 2–7 版原样导出，不建 `river_migration` | `:103` | ✓ |
| 全局约定 | 3.13 | 二·全局 `:62-66` | ✓ |
| 行为差异 | 4.6 的 20 行 | 第四节 `:160-179` 的 20 行 | 19 行 ✓；"退出、修改密码、停用后旧凭证失效"的行名少了"退出"（2.2 的 P3a 漏项） |

### 2.4 13.2 之外的遗留

按简报扫了五处：各 Phase 评审的第 6、7 节，各 Phase spec 的"不在本 Phase 范围内"，§16 中应对写明以后某个 M 的风险，Codex 的设计评审交给以后的 M 的事项，以及前面几节核对中顺带看到的事实。13.2 已有的不再列。

| 来源 | 事项 | 去处 |
|---|---|---|
| §16（失败闸门一行） | M8 做性能实测时观察同一 IP 的并发 | M8 交接第 2 节 |
| §16（请求体结构检查一行） | 请求体被解析两次，M8 实测开销 | M8 交接第 2 节 |
| §16（每个请求多一次查询一行） | M8 做性能和内存实测时一起观察 | M8 交接第 2 节 |
| §16（argon2 占用内存和 CPU 一行） | 13.2 只写了内存峰值和名单的内存，CPU 没有写 | M8 交接第 2 节（CPU 和耗时）；P1 评审第 7 节"持续集成上的 argon2 耗时没有实测"同在这一条 |
| §16 第二个 argon2 行；P2、P3a、P3b 评审第 6 节"交给负责人" | 调高参数之后登录耗时能区分休眠账户，是否加缓解由负责人以后决定 | 总体设计第 10 节新的一行（D4）；§16 这一行指向它（T8）；M8 交接第 2 节写明它不是 M8 的任务 |
| §16（生成器遇到不支持的写法一行） | 遇到它的 M 带着测试扩展生成器 | M4 交接第 4 节（第一个带数组或开放对象请求体的 M） |
| P3a 评审第 6 节"P3b 及以后的 plan" | 约 400 行的规则不适用于接口描述的模块文件 | 总体设计 6.3 规则 5（D3） |
| P3a 评审第 7 节"游标不签名" | 游标内容会影响可见范围的列表要另加检查 | M4 交接第 2 节、M7 交接第 1 节、M3 交接第 12 节（下一个分页列表） |
| P3b 评审第 7 节 | River 启动失败时 runner 无限重试、只记 WARN：模块漏注册 worker 时这是唯一的迹象 | M4 交接第 6 节（第一个加业务定时任务的 M） |
| P3b 评审第 7 节 | 数据库恢复之后真实的 River 客户端能否重新启动，只读过代码 | M8 交接第 3 节（部署核对） |
| P5 spec 第 5 节"M5" | `@nerve/services` 剩下的文件工具和整个包 | M5 交接第 3 节；同时发现地址规范化的调用方是 web 的 axios 基类（`web/apps/web/core/services/api.service.ts:10`），M6–M8 的旧调用还要它，删包时要挪进 web |
| 7.9 最后一行"P4（M5 按 Nerve 自己的协议加回上传）" | 规则 `plane-user-assets` 挡住 `USER_AVATAR`、`USER_COVER` | M5 交接第 1 节：M5 沿用这两个名字时收窄规则 |
| P4 删除 `USER_AVATAR`、`USER_COVER`（`cb2487c9`） | [M1-P3 给 M5 的交接](../../M5-files/handoffs/M1-P3-trim-platform.md)还写 8 种资源类型 | 那份交接改为 6 种（T6，D7） |
| P5 清零 `no-unneeded-ternary`（P5 评审第 2 节第 4 行） | M1 收尾给 M3–M8 的交接把它列为可以集中清掉的一类 | 六份交接写明它已清零（T6，D7） |
| 13.2 "周期下拉框的取数"写给 M4 | 周期的接口在 M6（总体设计 9.2） | M6 交接第 4 节（D5） |
| 13.2 "其余 12 个下拉框"写给 M3–M6 | 逐个看调用方：M5 没有；收集箱（`intake-state/base`）和视图（`@nerve/ui` 的 `Dropdown`，经 `dropdowns/layout.tsx` 用在 `views/form.tsx`）在 M7 | M7 交接第 3 节（D6） |
| 第 14 节第 5 项 | `/create-workspace` 挂载时不请求旧接口，只有 P4 浏览器核对 C8 看过一次，没有常设的故事 | 收尾补上 A3 的断言（3.2） |

查过、没有新的去处的：P1 评审第 6 节交给 P2、P3 的事项（P2、P3 的 spec 第 7 节已处理），P2 评审第 7 节"真实反向代理后面的客户端 IP"（P4 的 C11 关闭），P4 评审第 7 节"界面语言是全局的"（P5 spec 第 3 节第 16 条消除），P5 spec 第 5 节"M3"的 `CustomSearchSelect` 的 Tab 顺序（在 13.2 的下拉框一行，M3 交接第 14 节），Codex 设计评审"交给后续 M"一行的五项（M3 的邀请、M3/M4/M6 的物理删除图、M4/M7 的多排序游标、M5 的资产列迁移、M8 的 argon2 实测；都在 13.2），`TestWithinTxReportsAFailedRollback` 依赖内核缓冲（P1 评审第 7 节，只是以后偶发失败时的做法，不属于哪个 M），限流器在进程内和它的内存（v0 只有一个进程）。

### 2.5 关键词守卫

- `tools/keywords.json` 的 `phase` 从 `M2/P5` 改为 `M2/closeout`（T1）。`tools/keywords.mjs` 已认识 `M<n>/closeout`（M1 收尾加入）：收尾排在同一 M 的所有 Phase 之后。
- 例外只剩三条，都跨 M：`analytics`（`web/apps/web/core/services/cycle.service.ts`，`until` M6，迭代进度的接口地址）、`project-invitations`（`web/apps/web/core/services/user.service.ts`，`until` M3，`joinProject`）、`brand`（`pnpm-workspace.yaml`，`until` M9，来源说明）。P1–P5 没有留下 `until` 为 M2 的例外。
- 过期的探针（附录 A.4）：把一条例外的 `until` 改为 `M2/P5` 或 `M2/closeout`，守卫都报过期；改为 `M3` 时通过。
- 7.9 的规则：九条，都在 `tools/keywords.json`（2.6 第 5 行）。

### 2.6 第 14 节逐项

证据取自原型：`d97c513` 加上 T0–T8 的树（附录 A.5）。

| # | 第 14 节 | 证据 | 收尾做的 |
|---|---|---|---|
| 1 | P1–P5 和收尾都有 spec、plan、review | 第 15 节的前六行都是"已完成"，各有三个链接；收尾一行"进行中"，有 spec 和 plan 的链接（T0） | 收尾 review 的提交补上 review 的链接（D1） |
| 2 | A1–A17 的页面和接口版本、S1–S4 | `make e2e`：48 passed（`e2e/stories/identity/` 17 个文件，`e2e/stories/smoke/` S1–S5）。页面版本和接口版本调用 `e2e/fixtures/assert/identity.ts` 的同一组断言（M0-P6 交接的处理结果，P1–P4） | — |
| 3 | 后端的各类测试、depguard、`gen-check` 的覆盖 | `make test`：32 个包 `ok`，没有 `FAIL`；`make lint-go`：两段 `0 issues.`（depguard 在 golangci-lint 里）；`make gen-check`：没有差异。交错测试在 `server/internal/modules/identity/interleavings_test.go`、`interleavings_reset_test.go`；整程序测试 `server/internal/bootstrap/contract_test.go`；规则 4 的扩展 `server/internal/archtest/rules_test.go:40`；`TestSQLCSchemaScope` 在 `sqlc_test.go`；`make gen-go` 依次跑 oapi-codegen、`bodyshapegen`、sqlc（`Makefile:77-92`），`make gen-web` 生成 TS 客户端 | — |
| 4 | 前端的类型检查、knip、oxlint、单元测试、守卫 | `make lint-web`：`keywords: 60 rules, 3 exceptions, no hits.`，`Tasks: 54 successful, 54 total`（类型检查、oxlint 等于上限、格式、中英文的键）；`make knip` 为零；`make test-web`：`Tasks: 16 successful, 16 total`。上限在 P5 调低（web 551 → 452，ui 25 → 19，utils 18 → 12，i18n 1 → 0，P5 评审第 2 节第 4 行） | — |
| 5 | CSRF 的 `grep`；7.9 的规则；挂载时不请求 M3 的旧接口 | CSRF：在原型的 `web/` 上 `grep -rn -E 'csrfmiddlewaretoken|X-CSRFTOKEN'`（不含 `node_modules` 和构建目录）没有输出，`d97c513` 上 `git grep` 退出码 1。7.9 的七行对应 `tools/keywords.json` 的九条规则：`csrf`（不分大小写，含 `X-CSRFTOKEN`、`csrfmiddlewaretoken`、`get-csrf-token`）、`plane-auth-urls`（`/auth/` 的所有写法）、`auth-error-code`、`is-self-managed`、`with-credentials`、`plane-user-urls`、`plane-api-token-urls`、`plane-timezone-urls`（三者合起来是用户、实例、令牌、时区的地址）、`plane-user-assets`。挂载时的旧接口：见下表 | T2 补上 `/create-workspace` |
| 6 | 四张业务表、River 的表、差异清单 | `server/migrations/sql/00001`–`00004` 建四张表，`00005` 是 River 的迁移（S1 核对全部已应用）；差异清单见 2.3 | — |
| 7 | 四个决策点、11.1 | 决策点 1 B：`nerve users set-email`（A16）；2 C：prod 默认关闭注册、`nerve users create`（A2、A17、S3）；3 A：自助停用和 `deactivate`、`activate`（A12）；4 A：两步删除（规则 `is-self-managed`，M2 设计 3.19）；11.1：退出只结束当前会话（A6），总体设计 4.2 第二条（`v0-design.md:230`） | — |
| 8 | 浏览器核对的脚本全文 | P4、P5 两个有页面的 Phase 的 review 附录"9.6 的浏览器核对"：脚本全文和运行输出。P4 的 C5 在局域网 HTTP 上，C4a–C4c 是两个账户两个标签页，C6b、C6c 是会话暂不可用。P1–P3b 只有后端 | — |
| 9 | `handoffs/` 没有 `open`；M3–M8 的交接 | T7 之后 `grep -l -E '^status: (open|done)$' docs/v0/M2-auth/handoffs/*.md` 没有输出；M3–M8 各有 `handoffs/M2-closeout.md`（T5、T6），13.2 每一行的接收者链接到它（T8，`rows.mjs --design`：31 行都有链接） | T5–T8 |
| 10 | 上级文档按 3.20 同步、逐行核对；README；9.4 | 2.2：逐行核对，四处漏同步由收尾补上；README 的 8.7 各条在；9.4 在收尾 review 的提交里改为"已完成" | T3；D1 |
| 11 | 收尾 review 写明 7.10 的对比 | 附录 B 是实测和测法 | 收尾 review 引用附录 B |

**第 5 项的"挂载时不请求 M3 的旧接口"由什么守着**。M2 能到达的页面是 M2 设计 3.1 列的五类。nerve 对 `/api/v0` 之外的 `/api/…` 一律答 404，所以旧接口请求既出现在 `oldApiRequests` 里，也出现在失败的接口请求里；断言后者全等同样能发现它。

| 页面 | 故事与断言（原型上的行号） |
|---|---|
| 登录页 | S2 `s2-web-app.spec.ts:72`：`apiRequests` 全等 `["GET /api/v0/instance"]` |
| 注册页 | A10 第三个页面测试：从 `/sign-up` 起 `watchPage`，由 `takeProfileStep` 断言（`a10-onboarding-profile.spec.ts:35-36`） |
| `/onboarding` | A10 的三个页面测试：第一、三个经 `takeProfileStep` 的 `:35-36`，第二个 `:85-86` |
| `/create-workspace` | **收尾补上**：A3 第一个页面测试的四个页面各 `watchPage`，表单出现后 `a3-sign-in.spec.ts:61-63` 断言旧接口请求、失败的请求、未处理的异常都为空（3.2） |
| `/settings/profile/general` | A8 `a8-update-me.spec.ts:69-70`、`:105-106`；A12 `a12-deactivate.spec.ts:134-135` |
| `/settings/profile/preferences` | A9 `a9-preferences.spec.ts:112-113`、`:180-181`、`:258-259` |
| `/settings/profile/security` | A7 `a7-change-password.spec.ts:73`：失败的请求全等两个 422 |
| `/settings/profile/api-tokens` | A11 `a11-api-tokens.spec.ts:117`、`:250`、`:280`：失败的请求为空 |

### 2.7 状态翻转和 13.2 的指向

- **第 15 节**：收尾一行在提交 spec 和计划时改为"进行中"，带两个链接（计划的 Task 0）；"已完成"和 review 的链接在收尾 review 的提交里（D1）。
- **第 14 节**：每一项的勾和证据（2.6 的证据列，收尾的头上的结果）在收尾 review 的提交里；替换的全文在计划的最后一节。
- **总体设计 9.4**：T3 把漏项改为"进行中"和链接；"已完成"在收尾 review 的提交里。
- **13.2**：保留每一行（设计的记录），接收者一格改为指向交接文件的链接；表前一句说明交接是工作副本（T8）。

---

## 3. 交付物

### 3.1 给 M3–M8 的交接

六个新文件，格式照 M1 的交接：frontmatter `status: open`、`from: M2/closeout`、`to: M<n>`、`created: 2026-09-28`；标题；一段说明来源；编号的小节，每节写做什么、为什么、代码在哪、M2 做到哪一步、**关闭条件**；来源一行。长期的规则（stores 按会话分代）只写在总体设计 7.7，交接列本 M 的范围并指向它。全文在计划的 Task 5、6。

| 文件 | 小节 |
|---|---|
| `docs/v0/M3-workspace-project/handoffs/M2-closeout.md` | 1 邀请（交负责人确认）；2 登录后的落点与新手引导的取数；3 stores 按会话分代（M3 的 19 个模块级 service、`project_filter` 的反应）；4 `last_workspace_id` 的外键；5 `workspace_creation_enabled` 的执行；6 停用的端口；7 可空的引用字段；8 sqlc 的模块边界与 `TestSQLCSchemaScope`；9 物理删除关系图（M3 的部分）；10 CSP 的表情数据；11 M2 留下的 M3 调用和类型；12 页大小的规则（和游标不签名）；13 P5 改到、M2 走不到的地方；14 下拉框和复制到剪贴板（M3 的部分） |
| `docs/v0/M4-issue-core/handoffs/M2-closeout.md` | **1 命令行的只投递 River 客户端（负责人："做好记录，以后别漏了"；M4 用不到时原样转交）**；2 游标（和不签名）；3 自动归档读 `updated_at`；4 请求体检查的两处延伸和草稿发布；5 60 天清理与删除关系图；6 定时任务的 worker 和事件订阅者；7 编辑器的代码分割；8 CSP 的 callout 表情图；9 命令面板的主题命令；10 下拉框（M4 的部分）；11 复制到剪贴板（6 处）；12 stores 按会话分代（`IssueRootStore` 的 `autorun`、5 个模块级 service、三处同步读取）和 `getUserProfileIssues`；13 `clock.ts` |
| `docs/v0/M5-files/handoffs/M2-closeout.md` | 1 头像和封面（迁移范例、`plane-user-assets` 的收窄）；2 上传与按路由的中间件、`file_size_limit`、CSP；3 删除 `@nerve/services`（地址规范化的去处）；4 stores 按会话分代（8 个 `FileService`）；5 `storage.ts` |
| `docs/v0/M6-cycles-modules/handoffs/M2-closeout.md` | 1 迭代跨模块的写入和查询；2 物理删除关系图（M6 的部分）；3 stores 按会话分代（两个反应、2 个 `CycleService`）；4 周期下拉框的取数（D5）；5 下拉框（M6 的部分）；6 复制到剪贴板（3 处） |
| `docs/v0/M7-collaboration/handoffs/M2-closeout.md` | 1 保存视图的游标（和不签名）；2 stores 按会话分代；3 下拉框（M7 的部分，D6）；4 复制到剪贴板（4 处） |
| `docs/v0/M8-open-release/handoffs/M2-closeout.md` | 1 接口调用日志挂在限流之后（M0-P2 第 2 条）；2 性能和内存的实测（13.2 和 §16 的六项；负责人的 argon2 事项在哪里）；3 部署（`NERVE_ENV=prod`、停止宽限期、River 在数据库恢复之后）；4 接口文档页不从 CDN 加载；5 `webhook.ts`；6 stores 按会话分代 |

**13.2 的每一行落在哪里**（`rows.mjs` 用这张表核对每个文件都有这一节）：

| 13.2 的行 | 落点 |
|---|---|
| M3 邀请 | M3 §1 |
| M3 登录后的落点与新手引导的取数 | M3 §2 |
| M3 及以后有 stores 的 M | M3 §3、M4 §12、M5 §4、M6 §3、M7 §2、M8 §6；规则在总体设计 7.7（D2） |
| M3 `last_workspace_id` | M3 §4 |
| M3 `workspace_creation_enabled` | M3 §5 |
| M3 停用的端口 | M3 §6 |
| M3 3.2 的字段规则 | M3 §7 |
| M3 sqlc 的模块边界 | M3 §8 |
| M3 `TestSQLCSchemaScope` 的正则 | M3 §8 |
| M3、M4、M6 物理删除关系图 | M3 §9、M4 §5、M6 §2 |
| M3 CSP 表情选择器 | M3 §10 |
| M3 新手引导三步、`user.service.ts` 的三个方法、`is_bot`、时区接口 | M3 §11 |
| M3 页大小的规则 | M3 §12 |
| M3 P5 改到、M2 走不到的地方 | M3 §13 |
| M4 游标 | M4 §2 |
| M4 自动归档读 `updated_at` | M4 §3 |
| M4 请求体检查的两处延伸 | M4 §4 |
| M4 草稿发布 | M4 §4 |
| M4 60 天物理清理 | M4 §5 |
| M4 只投递的 River 客户端 | M4 §1 |
| M4 `getUserProfileIssues`、事件订阅者、callout 的 CSP | M4 §12、§6、§8 |
| M4 编辑器的代码分割 | M4 §7 |
| M4 命令面板的主题命令 | M4 §9 |
| M4 周期下拉框的取数 | M6 §4（D5） |
| M3–M6 其余 12 个下拉框 | M3 §14（`member-options`、`CustomSearchSelect` 的 Tab、9 个 `CustomSelect` 调用方、3 个 `CustomSearchSelect` 调用方和面包屑）；M4 §10（`DateDropdown`、`defaultOpen`、7 个下拉框、3 个 `CustomSelect` 调用方、4 个富筛选）；M6 §5（`cycle-options`、`module-options`、3 个 `CustomSelect` 调用方、面包屑）；M7 §3（`intake-state/base`、`Dropdown`、1 个 `CustomSelect` 调用方、面包屑）（D6） |
| M3、M4、M6、M7 复制到剪贴板 | M3 §14（3 处）、M4 §11（6 处）、M6 §6（3 处）、M7 §4（4 处） |
| M5 头像和封面 | M5 §1 |
| M5 上传与按路由的中间件 | M5 §2 |
| M6 迭代的跨模块写入 | M6 §1 |
| M7 保存视图的游标 | M7 §1 |
| M8 接口调用日志、实测、文档页、`NERVE_ENV`、停止宽限期 | M8 §1、§2、§4、§3 |

每个 M 的行数：M3 16、M4 13、M5 3、M6 6、M7 4、M8 2；13.2 的 31 行都有落点。另有 M0-P2 的接口调用日志（M8 §1）、M0-P6 的三个 fixture（M4 §13、M5 §5、M8 §5）和 2.4 的表外遗留。

**12 个下拉框按"接上组件的 M"分**：取组件在哪个 M 的页面上第一次拿到真实数据。`member-options` 是项目负责人（M3 的新建项目）；`date-range`、`priority`、`project/base`、`state/base` 和三个标签的下拉框都在工作项的属性和筛选里（M4；`date-range`、`project/base` 也在周期、模块的表单里，M6 核对）；`cycle-options`、`module-options` 的数据在 M6；`intake-state/base` 是收集箱，`@nerve/ui` 的 `Dropdown` 只经 `dropdowns/layout.tsx` 用在视图的表单里，都在 M7。

**模块级 service 的数字**：`git grep -n -E '^(export )?const [A-Za-z]+ = new [A-Za-z]+Service\(' -- web/apps/web` 在 `d97c513` 上 37 行：`WorkspaceService` 14、`FileService` 8、`ProjectService` 4、`IssueService` 2、`WorkItemVersionService` 2、`CycleService` 2，`WorkspaceNotificationService`、`WorkspaceDraftService`、`TimezoneService`、`ProjectMemberService`、`IntakeWorkItemVersionService` 各 1。`TimezoneService` 只调公开操作（`publicClient`），合 7.7；其余 36 个分到 M3（19）、M4（5）、M5（8）、M6（2）、M7（2）。它们现在经 web 的 axios 基类（不带令牌）调 Plane 的旧地址。

### 3.2 A3 守住 `/create-workspace`

A3 的第一个页面测试打开四个不合格的 `next_path`，登录后都落到 `/create-workspace`（`e2e/stories/identity/a3-sign-in.spec.ts:49-58`，`d97c513`），但不看请求。收尾给这四个页面各加 `watchPage`，在表单（`#workspaceName`）出现之后断言：没有旧接口请求，没有失败的接口请求，没有未处理的异常。测试数不变（48）。

变异（附录 A.3）：在页面里放进挂载时的旧接口请求（`useEffect` 里的 `fetch`；`useSWR` 的 fetcher），改后的 A3 三遍都失败在 `oldApiRequests`；`d97c513` 的 A3 三遍都通过，没有发现。

### 3.3 文档的其余改动

| Task | 文件 | 改动 |
|---|---|---|
| T1 | `tools/keywords.json` | `phase` → `M2/closeout` |
| T3 | `docs/v0/v0-design.md` | 9.4 的 M2 一行：进行中、设计文档的链接；7.1 的 `packages/services` |
| T3 | `docs/v0/M0-foundation/M0-design.md` | 3.7 的规则 4、6、8 |
| T3 | `docs/v0/plane-diff.md` | 第四节的行名加上"退出" |
| T4 | `docs/v0/v0-design.md` | 新的 7.7（D2）；6.3 规则 5（D3）；第 10 节的校准一行指向附录 B；第 10 节新的 argon2 一行（D4） |
| T6 | 六份 `M1-closeout.md`（M3–M8）；`docs/v0/M5-files/handoffs/M1-P3-trim-platform.md` | `no-unneeded-ternary` 已清零；资源类型 6 种（D7） |
| T7 | `docs/v0/M2-auth/handoffs/` 下 10 个文件 | `## 处理结果（M2/收尾）`、`status: closed` |
| T8 | `docs/v0/M2-auth/M2-design.md` | 13.2 的指向；§16 的 argon2 一行指向总体设计第 10 节 |

---

## 4. 与设计的差异和补充（请控制者裁定）

- **D1 状态翻转放在收尾 review 的提交里。** 简报第 6 条列了三处翻转（总体设计 9.4 的"已完成"、第 15 节的收尾一行、第 14 节的勾）。M1 的收尾把它们放在收尾 review 里（M1 收尾计划"完成后"一节；时机按"控制者评审补充"对 M1 收尾 spec 第 9 节第 3 条的裁定，以收尾合并为终点），理由是那时 review 的发现都已处理，勾才成立。建议照做：T3 只把 9.4 的漏项改成"进行中"和链接，T0 把第 15 节改成"进行中"；三处的最终替换写在计划的最后一节，由控制者在收尾 review 的提交里应用（块的核对覆盖它们）。另一种做法是 T9 之后由实现者直接改，缺点是 review 还没写就打了勾。
- **D2 M2 的前端会话规则写进总体设计的新 7.7（简报的第一个问题）。** 13.2"M3 及以后有 stores 的 M"不是一次性的任务，M3–M8 都要建 stores；交接关闭之后规则还要在。候选：
  - 总体设计 7.2（职责划分）：它讲分层，加进八条会把一节撑成两件事；
  - 前端改动清单：它是逐项改动的记录，不是规则；
  - 总体设计 4.3：它讲浏览器端的行为（令牌存放、续期、401），不讲代码的写法。
  
  建议新开 7.7"stores 按会话分代（长期有效，M2 起）"，放在 7.6"前端代码质量要求（长期有效）"之后：开头一句接 4.3（行为在 4.3，代码怎样做到在 7.7），八条规则各一句，最后一条写代码和测试在哪里。全文在计划 Task 4。六份交接只写本 M 的范围（模块级 service 的个数、要释放的反应、SWR 键）并链接 7.7；13.2 那一行的接收者一格也链接 7.7。`@nerve/ui` 不依赖 `@nerve/i18n` 一条有探针证明（附录 A.4：从它导入名字，`@nerve/ui` 的类型检查失败）。
- **D3 总体设计 6.3 规则 5 加一句：接口描述的模块文件不受约 400 行的限制。** P3a 评审把它交给"P3b 及以后的 plan"，P3b、P4、P5 的计划各在 Global Constraints 里写一遍，没有长期的落点；`api/modules/identity.yaml` 已有 623 行。依据是 M0-P3 交接第 5 条的组织规则（一个模块一个文件，模块文件之间不能互相引用）。这是改 v0 的一条规则的写法，所以请裁定；不同意时删掉计划 Task 4 的第一个块，M0-P3 交接的处理结果第 5 条随之改写。
- **D4 负责人的 argon2 事项记在总体设计第 10 节（简报的第二个问题）。** 第 10 节正是"风险与留待后续确定的事项"。新的一行写明事项和"是否加缓解由负责人以后决定"，右栏指向 M8 的实测；M2 设计 §16 的那一行加一句"收尾之后记在总体设计第 10 节"（T8）。M8 的交接第 2 节在 argon2 的实测旁写一条"与负责人的事项的关系"：它不是 M8 的任务，M8 的数字供负责人参考。建议提到：负责人决定时要的正是生产参数下的耗时和内存，放在一起不会漏看。
- **D5 "周期下拉框的取数"由 M6 接，而不是 13.2 写的 M4。** 这一行要"M4 接上周期时核对"，但周期的接口和 store 在 M6（总体设计 9.2 的 M6：迭代与模块）；M4 的工作项属性里有周期下拉框，但它在 M6 之前取不到周期。13.2 的记录不改，接收者一格写明"M4；收尾改由 M6 接"及理由（T8）。不同意时，把 M6 交接第 4 节移回 M4。
- **D6 "其余 12 个下拉框"一行拆到 M3、M4、M6、M7。** 13.2 写的是"M3–M6（接上组件的 M）"，但按调用方逐个看（3.1），M5 没有，收集箱和视图的两个在 M7（同一行后半已把 `CustomSelect` 的一个调用方和面包屑的视图页头写给 M7）。接收者一格改为四个链接并写明原因（T8）。
- **D7 改两处已有交接里被 M2 改变的事实。** M1 收尾给 M3–M8 的六份交接都把 `eslint(no-unneeded-ternary)` 列为可以集中清掉的一类，M2/P5 已全仓清零，某个 M 选它就等于没清；M1-P3 给 M5 的交接写 `EFileAssetType` 剩 8 种，M2/P4 删了其中两种。建议在原文处改为现在的事实并链接证据（T6），而不是只在新交接里提一句（一个事实一个位置）。
- **D8 收尾改一个故事。** 第 14 节第 5 项要求"M2 能到达的页面挂载时不请求 M3 的旧接口"，核对发现 `/create-workspace` 没有常设的守卫（2.6），变异证明缺口是真的（3.2）。简报允许"只在核对找到真实缺口的地方改代码"，所以 T2 只改 `a3-sign-in.spec.ts`。

---

## 5. 验收标准（完成线）

- [ ] T1–T8 的每个 Task 一个提交，每个 Task 的检查都得到计划写的结果；T9 的七个门禁都通过：`make gen-check` 没有差异，`make lint-go` 两段 `0 issues.`，`make test` 32 个 `ok`、没有 `FAIL`，`make lint-web` 没有命中、54 个任务成功，`make knip` 为零，`make test-web` 16 个任务成功，`make e2e` 48 passed。
- [ ] `node tools/keywords.mjs` → `keywords: 60 rules, 3 exceptions, no hits.`，`phase` 是 `M2/closeout`，例外是 M3、M6、M9 三条；过期的探针的三行与计划相同。
- [ ] A3 守住 `/create-workspace`：两种变异各三遍都失败在 `oldApiRequests`。
- [ ] `docs/v0/M2-auth/handoffs/` 的 10 个文件都是 `status: closed`，各有 `## 处理结果（M2/收尾）`。
- [ ] M3–M8 各有 `handoffs/M2-closeout.md`，`status: open`；`paths.mjs` 只报出三个要由接收 M 新建的 fixture（`clock.ts`、`storage.ts`、`webhook.ts`）；`rows.mjs` 每个 M 都是 `0 without a section`；13.2 的 31 行都链接到交接。
- [ ] 2.2 的四处漏同步都已改，spec 记为各 Phase 的漏项。
- [ ] 总体设计有 7.7、6.3 规则 5 的一句、第 10 节的两处（D2–D4 裁定之后按裁定）。
- [ ] 本分支相对 `d97c513` 改了计划的文件表中的 29 个文件，另有本 spec 和计划。
- [ ] 收尾 review 的提交（控制者）：9.4、第 15 节、第 14 节按计划最后一节替换。

---

## 6. 风险

| 风险 | 应对 |
|---|---|
| 交接的分派错了 M（哪个 M 第一次接上某个组件或接口） | 按调用方和总体设计 9.2 的领域逐个判断（3.1 写明依据）；分派有疑问的两处列为 D5、D6 请裁定；每个交接的关闭条件都要求接收 M 在浏览器或测试里核对 |
| 交接的路径过时（接收 M 开始时代码已变） | 路径和行号取自 `d97c513`，`paths.mjs` 核对都存在；行号只写在稳定的地方，改动频繁的写函数名 |
| A3 的新断言在表单出现之前、请求还没发出时就判定 | 断言在 `#workspaceName` 可见之后；挂载时的请求在 effect 里发出，两种变异各三遍都被发现（附录 A.3）；靠计时器延后的请求不算"挂载时" |
| `make lint-web`、`make test-web` 的部分任务取自 turbo 的缓存 | turbo 按输入的哈希取缓存，输入没变的包结果相同；改到的 e2e 包重新检查了（`Cached: 36 cached, 54 total`，附录 A.2 的 T2） |
| 13.2 改了接收者一格，设计的记录变了 | 只把名字换成链接、在两行写明改派（D5、D6），事项的文字不动 |
| 2026-09-26、27 留下的 14 个 `Created` 状态的 testcontainers | 不是收尾建的；收尾不动它们，写进报告，由控制者决定是否清理 |
| 持续集成上 e2e 的时间随故事增长 | 附录 B.4：M2 的 E2E 一步从 18 秒到 56 秒；M3 起每个 M 的设计估计它，超过约 5 分钟时考虑分片 |

---

## 附录 A：原型的核对和门禁

原型：`git clone --quiet --branch worktree-m2-closeout --single-branch <worktree> $M2TMP/closeout/proto`（`d97c513`），`pnpm -C $M2TMP/closeout/proto install --frozen-lockfile`。块用 `node $M2TMP/closeout/planapply.mjs <plan> apply $M2TMP/closeout/proto <Task>` 逐个应用。命令在原型里运行时用 `node $M2TMP/closeout/at.mjs $M2TMP/closeout/proto <命令…>`（以原型为工作目录；worktree 的会话不在别的目录里跑 `cd`）。门禁用 `bash $M2TMP/closeout/gates.sh $M2TMP/closeout/proto <标签> [门禁…]`，全部输出在 `$M2TMP/closeout/logs/`。

### A.1 基线（`d97c513`，改动之前）

```
gen-check: exit 0 in 4s;
lint-go: exit 0 in 9s; 2 × '0 issues.'
test: exit 0 in 17s; 32 ok, 0 FAIL
lint-web: exit 0 in 25s; keywords: 60 rules, 3 exceptions, no hits.;  Tasks: 54 successful, 54 total
knip: exit 0 in 3s; pnpm --filter web exec react-router typegen pnpm exec knip --treat-config-hints-as-errors
test-web: exit 0 in 5s;  Tasks: 16 successful, 16 total
e2e: exit 0 in 20s;  48 passed (13.2s)
```

### A.2 块的核对和每个 Task 的检查

`node planapply.mjs plan.md check $M2TMP/closeout/head`（`head/` 是 `git archive d97c513` 解开的树）：计划的 78 个替换在 `d97c513` 上各恰好一处（收尾 review 的三个替换按顺序各恰好一处），6 个新文件在 `d97c513` 上都不存在：`78 replacements, 6 new files; all ok`。全文在计划的"块的核对"。

| Task | 检查 | 结果 |
|---|---|---|
| T1 | `node tools/keywords.mjs`；例外的列表；过期的探针；`gates.sh … t1 lint-web` | `keywords: 60 rules, 3 exceptions, no hits.`；M6、M3、M9；见 A.4；`lint-web: exit 0; … Tasks: 54 successful, 54 total` |
| T2 | `gates.sh … t2 lint-web e2e`；`mount-mutation.sh` | `lint-web: exit 0 …54 successful`（`Cached: 36 cached, 54 total`，e2e 包重新检查）；`e2e: exit 0; 48 passed (13.8s)`；变异见 A.3 |
| T3 | 规则原文的 `grep`；守卫 | 三行 `:40,42,44`；没有命中 |
| T4 | `ui-i18n.sh`；守卫 | 见 A.4；没有命中 |
| T5 | `paths.mjs`、`rows.mjs M3 M4`；守卫 | M3 `59 paths, 0 missing`；M4 `45 paths, 1 missing`（`e2e/fixtures/clock.ts`，M4 新建）；`M3: 16 rows, 0 without a section`、`M4: 13 rows, 0 without a section`；没有命中 |
| T6 | `paths.mjs`、`rows.mjs M5 M6 M7 M8`；守卫 | M5 `20 paths, 1 missing`（`e2e/fixtures/storage.ts`）、M6 `20 paths, 0 missing`、M7 `15 paths, 0 missing`、M8 `6 paths, 1 missing`（`e2e/fixtures/webhook.ts`）；`M5: 3`、`M6: 6`、`M7: 4`、`M8: 2 rows, 0 without a section`；没有命中 |
| T7 | `grep -L '^status: closed$'`、`grep -l -E '^status: (open|done)$'`；守卫 | 两个都没有输出；没有命中 |
| T8 | `rows.mjs --design`；`paths.mjs --links`；守卫 | `13.2: 31 rows, 31 with a link to a handoff, 0 without`；链接的目标都存在（放入本 spec 和计划之后）；没有命中 |

### A.3 T2 的变异

`bash $M2TMP/closeout/probes/mount-mutation.sh $M2TMP/closeout/proto`，T2 之后：

```
mutation effect-fetch: a3 exit 1;  3 failed  6 passed (4.7s) ; failures quoting watch.oldApiRequests: 3; received: "GET /api/workspaces/"
mutation swr-fetch: a3 exit 1;  3 failed  6 passed (4.6s) ; failures quoting watch.oldApiRequests: 3; received: "GET /api/workspaces/"
restored: identical
rebuilt: ok
```

失败的都是 A3 的第一个页面测试（每遍在四个 `next_path` 中先失败的那一个报出，例如 `Error: //evil.example`，`+ Array [ "GET /api/workspaces/" ]`）；通过的是第二个页面测试和接口版本，各三遍。

对照：把原型的 A3 换回 `d97c513` 的版本，跑同样的两种变异：

```
mutation effect-fetch: a3 exit 0;  9 passed (4.0s) ; failures quoting watch.oldApiRequests: 0; received: none
mutation swr-fetch: a3 exit 0;  9 passed (3.7s) ; failures quoting watch.oldApiRequests: 0; received: none
restored: identical
rebuilt: ok
```

之后换回 T2 的版本（`cmp` 相同）。整套 e2e 也做了一遍对照（`bash $M2TMP/closeout/probes/control-full.sh $M2TMP/closeout/proto $M2TMP/closeout/head`：`d97c513` 的全部故事，页面里放进 `useEffect` 的变异）：

```
control-full (d97c513 stories, effect-fetch in /create-workspace): e2e exit 0;  48 passed (13.1s)
restored: identical
rebuilt: ok
```

`d97c513` 上没有哪个故事发现它；e2e 里写明落到 `/create-workspace` 的只有 A3（`a3-sign-in.spec.ts:55`）。

### A.4 两个探针

守卫的过期（T1，`node $M2TMP/closeout/at.mjs $M2TMP/closeout/proto node $M2TMP/closeout/probes/expiry.mjs`）：

```
until M2/P5: exit 1, expired exception: analytics  web/apps/web/core/services/cycle.service.ts  "analytics" is until "M2/P5", at or before phase "M2/closeout"; delete it
until M2/closeout: exit 1, expired exception: analytics  web/apps/web/core/services/cycle.service.ts  "analytics" is until "M2/closeout", at or before phase "M2/closeout"; delete it
until M3: exit 0, keywords: 60 rules, 3 exceptions, no hits.
tools/keywords.json restored: identical
```

`@nerve/ui` 不能从 `@nerve/i18n` 导入（T4，`bash $M2TMP/closeout/probes/ui-i18n.sh $M2TMP/closeout/proto`）：

```
ui imports @nerve/i18n: check:types exit 2
src/index.ts(25,55): error TS2307: Cannot find module '@nerve/i18n' or its corresponding type declarations.
restored: identical
```

第一次写成只有副作用的 `import "@nerve/i18n";`，类型检查通过（退出码 0）：TypeScript 不报解析不到的副作用导入。7.7 的写法因此是"导入的名字过不了类型检查"，探针用 `export { useTranslation as probeUseTranslation } from "@nerve/i18n";`。`web/packages/ui/node_modules/@nerve/` 下只有 `constants`、`hooks`、`propel`、`types`、`typescript-config`、`utils`。

### A.5 全部门禁（T0–T8 之后）

`bash $M2TMP/closeout/gates.sh $M2TMP/closeout/proto final`：

```
gen-check: exit 0 in 2s;
lint-go: exit 0 in 1s; 2 × '0 issues.'
test: exit 0 in 18s; 32 ok, 0 FAIL
lint-web: exit 0 in 2s; keywords: 60 rules, 3 exceptions, no hits.;  Tasks: 54 successful, 54 total
knip: exit 0 in 2s; pnpm --filter web exec react-router typegen pnpm exec knip --treat-config-hints-as-errors
test-web: exit 0 in 0s;  Tasks: 16 successful, 16 total
e2e: exit 0 in 14s;  48 passed (12.7s)
```

`make lint-web` 的 turbo 取了 38 个缓存（共 54），`make test-web` 16 个都是缓存：它们的输入在 T2 之后没有变（T3–T8 只改文档和 `tools/keywords.json`，守卫本身每次都跑）。之后 `docker ps -a --filter label=org.testcontainers=true` 列出的是 14 个 2026-09-26、27 建的 `Created` 状态的旧容器，这次运行建的都已被 Ryuk 删除。

### A.6 计划的树就是原型的树

`d97c513` 的另一份干净副本（`$M2TMP/closeout/replay`，由 `git archive` 解开），按顺序应用计划的 Task 0–8，与原型逐个文件比较（`node $M2TMP/closeout/treediff.mjs replay proto`，不含 `node_modules`、`.git`、构建产物）：`2649 and 2649 files; 0 differences`。原型相对 `d97c513`：29 个文件（23 个改、6 个新）。

---

## 附录 B：7.10 的估计与实际

### B.1 怎样量

- **范围**：`559c3c6`（M1 收尾的合并，M2 开始）到 `d97c513`（P5 的合并，M2 结束），247 个非合并提交。
- **输入**（在 worktree 根目录，一个命令一次）：
  - `git diff --numstat --find-renames 559c3c6 d97c513 -- web e2e server api tools Makefile .github > $M2TMP/closeout/measure/numstat.tsv`
  - `git diff --name-status --find-renames 559c3c6 d97c513 -- web e2e server api tools Makefile .github > $M2TMP/closeout/measure/namestatus.tsv`
  - `git archive --format=tar -o $M2TMP/closeout/m2start-559c3c6.tar 559c3c6`、`git archive --format=tar -o $M2TMP/closeout/head-d97c513.tar d97c513`，各自解开到 `m2start/`、`head/`
- **脚本**：`node $M2TMP/closeout/measure/measure.mjs`。7.10 只给了量的名称，没有写数法；这里的数法是：删掉的文件全算"删除"，新文件全算"新增"；改动的文件取 `min(+, −)` 为"重写"，`−` 的其余为"删除"，`+` 的其余为"新增"（所以 删除 + 重写 = −，新增 + 重写 = +，净变化 = + − −）。生成的文件（`schema.gen.ts`、server 的 `gen/`、`api/dist`）、12 个语言文件和包的清单另列。web 的文件按 7.10 的文件组归类（脚本里写明每组的路径规则），每组在 `559c3c6` 上的行数与 7.10 的口径并排，用来判断规则是否对得上：7.10 是 8,467 行，这些规则得到 8,776 行。
- **持续集成**：`bash $M2TMP/closeout/measure/ci-times.sh 559c3c62881d355616107bef68a47d49b67a6f1a d97c513a558abc98c030e4d6c4815b6d5735e6cf`，再 `bash … --run 36364148797`（公开的 GitHub API，`curl` + `jq`）。

### B.2 前端（`web/`，不含生成的文件、语言文件和清单）

| 文件组 | 7.10 的口径 | 这里的口径 | 文件 | 删除 | 重写 | 新增 | 净 | 7.10 估计的重写 |
|---|---|---|---|---|---|---|---|---|
| services | 300 | 300 | 6 | 90 | 50 | 27 | −63 | 约 250（两组合计） |
| `packages/services` | 400 | 371 | 4 | 179 | 0 | 0 | −179 | （同上） |
| stores | 1,276 | 1,276 | 8 | 345 | 180 | 139 | −206 | 约 700 |
| store 的 hooks | 203 | 176 | 3 | 12 | 22 | 14 | +2 | — |
| wrappers | 246 | 246 | 2 | 26 | 71 | 13 | −13 | 约 180 |
| 认证页和表单 | 1,122 | 1,253 | 12 | 241 | 215 | 120 | −121 | 约 450 |
| 个人设置 | 2,405 | 2,675 | 19 | 411 | 378 | 156 | −255 | 约 900 |
| 新手引导 | 2,315 | 2,315 | 16 | 451 | 176 | 20 | −431 | 约 450 |
| 类型 | 200 | 164 | 6 | 112 | 0 | 0 | −112 | — |
| 新：令牌管理器和它的同伴 | — | — | 6 | 0 | 0 | 643 | +643 | 新增约 400（令牌管理器 300、协调 100） |
| 测试 | — | — | 22 | 0 | 10 | 3,712 | +3,712 | 新增约 450 |
| 其余（跟随类型的使用方、P5 的下拉框、清理） | — | — | 102 | 347 | 503 | 152 | −195 | — |
| **合计** | 8,467 | 8,776 | **206** | **2,214** | **1,605** | **4,996** | **+2,782** | |
| 另列：生成的 `schema.gen.ts` | | | 1 | 0 | 4 | 803 | +803 | |
| 另列：语言文件 | | | 12 | 2 | 25 | 350 | +348 | |
| 另列：包的清单 | | | 8 | 1 | 7 | 3 | +2 | |

| 项 | 7.10 的估计 | 实际 | 比 |
|---|---|---|---|
| 改动的文件 | 85–100 个 | 206 个（文件组 76、新文件 6、测试 22、其余 102） | 约 2.2 倍 |
| 删除 | 约 1,050 行 | 2,214 行 | 2.1 倍 |
| 重写 | 约 3,000 行 | 1,605 行 | 0.5 倍 |
| 新增 | 约 1,000 行 | 4,996 行（测试 3,712，令牌管理器和同伴 643，其余 641） | 5 倍（不含测试 1.3 倍） |
| 净变化 | 减少约 50 行 | 增加 2,782 行；不含测试减少 930 行 | |

### B.3 后端

| 项 | 7.10 的估计 | 实际 |
|---|---|---|
| 生产代码（不含生成的代码、测试工具和名单） | 约 5,450 行 | 89 个文件，+7,866 / −124，净 7,742 行（1.4 倍） |
| 其中请求体结构的校验器、格式检查器和生成器 | 约 400 | 675（`server/internal/platform/httpserver/bodyshape` 和 `server/tools/bodyshapegen`） |
| 其中账户行锁 | 约 150 | 51（`identity/app/credential_lock.go`；加锁的 SQL 在适配器里） |
| 其中令牌桶 | 约 100 | 173（`platform/ratelimit`） |
| 其中刷新令牌的 MAC | 约 50 | 34（`identity/adapter/signing/mac.go`） |
| 测试 | 约 6,700 行 | 116 个 `*_test.go`，净 15,593 行（2.3 倍）；测试工具（`pgtest`、`apitest`、`clocktest`）另有 633 行 |
| 生成的 Go（oapi-codegen、`bodyshapegen`、sqlc） | — | 11 个文件，净 3,085 行 |
| 手写的 SQL | 约 350 行 | 336 行（查询 227，迁移 `00001`–`00004` 109） |
| River 的迁移 | 318 + 189 行 | 521 行（Up 329，Down 192；多出的是 goose 的注释和 `StatementBegin`/`End`） |
| `identity.yaml` | 约 900 行 | 623 行 |
| 常见密码名单 | 33,887 行，277 KB | 33,904 行，278,161 字节 |

### B.4 端到端和持续集成

| 项 | 7.10 的估计 | 实际 |
|---|---|---|
| fixture | 约 500 行 | 10 个文件，+1,197 / −79，净 1,118 行 |
| 17 个故事 | 约 2,150 行 | `e2e/stories/identity/` 17 个文件，2,191 行；另外 S1–S3 改、S5 新（净 +101） |
| 测试数 | — | 5 个（S1–S4）→ 48 个 |

持续集成上的 e2e 任务（`ci-times.sh` 的输出，秒）：

| 运行 | 任务 | Build 一步 | E2E 一步 |
|---|---|---|---|
| `559c3c6`，main，36104770311 | 120 | 40 | 18 |
| P5 的分支 `6097b013`，36364148797 | 160 | 40 | 65 |
| `d97c513`，main，36365103006 | 145 | 30 | 56 |

同一次运行里 web 任务 134 → 138 秒，server 任务 37 → 69 秒。本机 `make e2e` 的 48 个测试 12.7–13.8 秒。

### B.5 对 M3 以后的估算意味着什么（总体设计第 10 节要求 M2 校准）

1. **测试是新增的大头。** 前端新增的 4,996 行中 3,712 行是测试（74%），7.10 估的是 450；后端测试是生产代码的 2 倍。M3 起估新增时，测试按生产代码的 1–2 倍单独估，不并进"其余"。
2. **Plane 的代码多是删掉重写，而不是改。** 重写只有估计的一半（1,605 对 3,000），删除是两倍（2,214 对 1,050）：stores、认证页、个人设置、新手引导都是删掉旧的、写新的薄 service 和 store。M3 起把领域文件组的大部分行估为"删除"，新写的 service、store 估为"新增"，"重写"只留给组件。
3. **改动的文件数被横向的清理放大。** 206 个文件中 102 个在 7.10 的文件组之外：跟随类型的使用方，P5 的下拉框（`CustomSelect`、`CustomSearchSelect` 和它们的调用方），以及 M1 收尾交接要求的死成员、oxlint 和一整类规则（`no-unneeded-ternary` 54 处，遍布全仓）。M3 起把 M1 收尾交接的清理单列一项估，不算进领域的改动。
4. **后端的生产代码多了 40%，多在平台上。** 限流器、请求体结构检查、River 的 runner、配置和管理命令都是第一次建；M3 起复用它们，领域代码更接近按用例估的数。测试照 2 倍估。
5. **接口描述比估的短。** `identity.yaml` 623 行、15 个操作，约每个操作 40 行（含 schema）；M3 起按操作数估。
6. **端到端：故事按估计，fixture 是两倍。** 故事 2,191 对 2,150；fixture 1,118 对 500（页面的观察 `watchPage`、认证和个人设置页的帮助函数）。持续集成上 E2E 一步从 18 秒到 56 秒，多了 43 个测试，约每个测试 0.9 秒；照这个速度，M3–M8 再加 150 个测试会多约 2–3 分钟。每个 M 的设计估计它；E2E 一步超过约 5 分钟时考虑分片。
