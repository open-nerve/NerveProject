# M5 / P1 本地存储评审记录

> 状态：T1–T9 实施与七项本地门禁完成；T10 整审、夹具修复与复审完成，追加整包预算复验中；最终远端 CI、T11 集成尚未完成。
> T9 已按 [re-32](../../../reviews/re-32-M5文件路由方案A复核.md)落实 `/static/local`、逐段路径策略与具名开发代理；聚焦回归与完整七项本地门禁通过；阶段初审见 re-33，准确远端 CI 的唯一 server 失败按 §10 处理。
> 分支：`codex/m5-p1`；基线：`fb3e788b7be33467018385c9a67d515e51b79dad`。原件目录为本 worktree 的 `.superpowers/sdd/P1-local-storage`，集成后补归档映射。

## 1. 范围

[spec](../specs/P1-local-storage.md) 与 [plan](../plans/P1-local-storage.md) 定义 local 存储、用途 MAC、原始 HTTP、请求期限/并发、组合和测试隔离。本段不交付业务资产申请/确认、数据库表、S3 或上传页面。

## 2. 逐条验收

| 项 | 当前证据与边界 |
|---|---|
| A1 | domain/app 小端口与架构包通过；模块不互相 import，无新依赖，bootstrap 统一组合 |
| A2 | 共享合同与 app 组合的真实 PUT→Inspect→Seal→GET/HEAD；完整发布、短体/断连/取消不发布 |
| A3 | 字段、编码、用途、方法、时效变异；真实 HKDF 派生、相同 PEM 重启、新密钥拒绝；业务时钟推进不使签名过期 |
| A4 | 二次写入不覆盖、同长度源字节变化不封存、并发候选独立、u/o 公开访问约束 |
| A5 | 短预算真实 socket idle/total/cancel 与 JSON 1 MiB/15s 边界已验证；T9 默认配置 5 MiB 慢流 31.9s 成功、封存/读取字节与摘要一致，race 通过 |
| A6 | HEAD/Range/条件缓存、处置/文件名、错误 no-store、验签先于 304；中途故障截断且下一请求可用 |
| A7 | 独立 IP bucket/可信代理/IPv6、满额 503、各类退出释放；停读下载写超时和健康下载同时验证 |
| A8 | 正常 HTTP→jobs→storage→DB、后续构造失败、重复 close；同 key 跨世界不同字节，clock 重启复用 root；7 项 E2E 生命周期通过 |
| A9 | 原 namespace 回归已按最终授权修复：真实 storage 工作区页面、保留集合和原根路径守卫通过；逐段路径反例、真实 Vite 传输/取消/安全日志及构建冲突守卫通过。七项本地门禁已通过，准确测试树和日志见 §9 |
| A10 | T1–T8 局部红绿/变异留原件；七项本地门禁、独立整审及局部修复复审已通过；准确最终 branch/main CI 和归档仍未完成 |

## 3. 测试

- 每 Task 的准确基线、命令和结果在原件 `progress.md`、`task-N-tests.log` 或具名日志中，不能用后来的通过覆盖早期失败。
- T7：实际关闭行为红（0 次）、clock 复用前红（clock-1.files）；组合 race、7 项 clock E2E、架构、lint/types/format 全绿。重复关闭变异被确切计数断言杀死，恢复后再验收。
- T8：5 个路径规范化反例实得 307，位置与响应正文含签名查询；修复后为固定 404 no-store Problem。原 JSON 控制已存在，新增组合断言先核原行为，再以放宽体积/期限的变异验证敏感性，不伪称新功能首次红。
- 原 JSON 默认 15s 在真实登录查询的 context 边界采样，上下界由请求发送/完成时间给出；实际取消性质由现有 HTTP 平台测试继续验证，不用固定 sleep 同步。
- 文案先见 8 个缺项的守卫红，再补准确映射及两种语言；`make test-web` 18 任务通过（15 缓存、3 实跑）。首次组合配置缺 database.url 与一次工具 cwd 错误属于夹具/命令失误，修正后重跑，不算产品性质红。

- T9 慢流使用真实默认 read=30s、API=15s、JSON=1 MiB、文件预算=55s；基于单调时间按字节节流，实际 31.914s，race 31.917s。移除逐请求读期限的定向变异在全局读期限处失败（499、约 30.6s），恢复后通过。当时完整七门禁只通过生成检查；其余六项因下面的命名空间回归停止，不能称完整 CI 通过。
- T9 用真实 API 创建 storage 工作区得到 201，工作区 API 为 200，但项目/设置页面都为 404 Problem；仅去掉 asset 路由注册后两页恢复 200 HTML。原 `TestTheReservedServerSlugsAreTheServersTopLevelPaths` 独立复现失败。临时探针已移回原件区，代码按原字节恢复；原件哈希和对照见待裁定方案。T8 聚焦检查没有覆盖既有工作区命名合同，预检也漏掉了该核对，不能放宽守卫迎合当前实现。

## 4. 审查问题

- T5：短体错误曾被同时取消覆盖为 499；保留具体读取失败并区分读后端故障后，实际 TCP 短体为 400、不发布，后端读取故障为 500。
- T6：给下载的后台读设置 idle 会错误结束健康下载；改为下载读侧仅总期限、写侧按实际进度，保持取消优先。新上传上限降低后，已有大文件按可信大小和最长时限读取。
- T8：ServeMux 在 handler 之前规范化路径并重定向，可能把签名 query 带入 SPA。平台提供按路径树声明的拒绝策略，由 asset 声明 `/storage`；固定错误不回显 query/path。无对象访问或其他模块权限改动。
- T8：新 raw 目录最初不参与 web 测试缓存；实际改变目录前后任务 hash 相同。加入准确输入后 hash 改变，恢复目录后 hash 回到原值，并重新运行完整前端测试。
- 阶段 deep-review 已由独立审阅者完成，范围、限制和初审问题见 [re-33](../../../reviews/re-33-M5P1整体审核.md)，修复复审见 §10。

## 5. plan 差异与局部裁定

- 已批准的组合边界不变：module 仅构建/注册 HTTP，bootstrap 持有存储 Close；既有 pool 工厂扩为两个明确资源工厂以验证真实所有权，不新增 DI 框架或万能 Storage。
- HTTP 定位读取由消费方另声明 `OpenTransfer`，app 的流式读取端口不扩成 seek 接口；T6 的大测试文件按 fixture/行为拆开，保持文件边界。
- raw 路由/专用码登记在 `api/storage-local.json`；Go 全路由和源码码目录、前端消息表精确并集核对。该文件不生成业务 SDK，也不增加 publicOperations 或宽泛忽略。
- T5/T6 码的同 Task 文案未同步，T8 在首次产品推送前补齐；保留偏差，不声称原 Task 已同时交付文案。
- 错了的代价：原始传输目录与业务 OpenAPI 分开维护，必须保留两侧准确并集和源码核对；路径树策略只覆盖已声明树，新增签名入口必须声明相同政策并补目录。

## 6. 交接

P2 承接准备件、确认/封存候选、业务权限和清理，并把 asset 从 actionlessModules 移到真实 Actions/规则清单；P3 承接 S3；P6/P7 承接浏览器传输与宿主入口。当前不关闭 M5 的完整故事或任何尚需业务链证明的输入交接。

## 7. 集成状态

设计基线 main 的 [CI 41414200682](https://github.com/open-nerve/NerveProject/actions/runs/41414200682) 已全绿且归档清理完成。P1 检查点 ecefb0bc 已推送，尚未合并；CI 41455598403 attempt 1 的 web 成功、server 失败、E2E skipped。详细日志已由连接器取得并保留，无需提取凭据；T10 修复及准确最终 CI 后按连续授权执行 T11，不以设计 CI 替代产品 CI。

随后 `18906f08` 修复夹具，`0189c2ab` 补验收文档。前者运行因后续提交自动取消；后者 [CI 41460165212](https://github.com/open-nerve/NerveProject/actions/runs/41460165212) 的两次 server 均因 bootstrap 整包 600s 上限失败、web 成功、E2E skipped，asset HTTP 包两次均通过。整包预算修订与最新准确 CI 接续见 §10，不以重跑本身冒称查明环境根因。

## 8. T9 命名空间修订实施证据（2026-11-02）

- `868e9a24`：asset/adapter/localprotocol 统一 `/static/local`，static 只调整保留分类，domain/app 无协议依赖。真实 storage 工作区与签发 URL 先红后绿，原根路径守卫、组合 race 和架构测试通过。
- `8e980a7e`：通用 Router 检查规范化中途进入再离开；9 个漏检子例先得 307 并泄露 query，修复后 13 个危险路径与 7 个控制通过。临时恢复二点法会失败，恢复后平台与架构 race 通过。
- 开发插件用 Node 标准库流式传输，仅分派文件树，Go 保留验签/业务预算；独占上游、取消和关闭，日志只含固定错误类别。真实 Vite/TCP 共 11 项测试通过：方法/query/二进制、Range/HEAD/304/private、部分流断连、上游中途截断、关闭、public 与晚发 bundle 冲突。日志含 query、取消失效两项变异均被杀死并恢复。
- 原件：`task-9-namespace-*.log`、`task-9-path-*.log`、`task-9-vite-*.log`、`task-9-proxy-*.log`、`task-9-combined-race.log`。类型检查使用仓库 `check:types` 先生成路由类型；直接 tsc 的缺生成物与测试 fixture 类型错误保留为初轮失败，已修正后通过。
- 夹具校正：macOS 临时目录先取 realpath，避免 Vite 构建根的符号路径差；Vite 把 port=0 当默认值，改用官方 middleware mode 配真实随机端口，避免并行端口竞争。开发服务器的 missing.js 断言明确 JavaScript Accept，生产缺失 chunk 的 404 由 Go 完整程序测试独立验证。未修改这些既有行为。

- 完整七门禁首次运行于 `b051df25258de313f6dcfac7387055c93fa7630a`：gen/gen-check 与 lint-go 通过，make test 用时 347.277s，仅 `TestBuiltInProfiles` 和 `TestEveryModuleDeclaresItsActionsOrHasNone` 失败。前者补齐 spec 中的具体默认值，后者按既有枚举机制登记 P1 没有业务动作的 asset；守卫谓词、精确全路由与角色规则不改。聚焦配置/动作/反例测试转绿，完整门禁在修复后重新执行。

- `f6cde552`：gen/gen-check、lint-go、make test 全绿（346.872s）。lint-web 的关键词检查把标准 HTTP `upgrade` 头当作付费功能；按现有精确例外机制登记一次命中与 M9 复核，不修改规则。单独 knip 揪出 fixture 内部 `listen` 的多余导出，已收回，不加忽略。后续门禁从 lint-web 继续，服务端与生成输入保持原字节，结果按输入哈希关联到最终提交。

- `07bd0781`：lint-web 54/54、knip、test-web 18/18 通过；完整构建成功，E2E 335 通过、1 个既有 C12 移动偏好故事失败。trace 证明 F6 后通知区已聚焦，4ms 后菜单延迟恢复焦点到 Layout，Close 重新 aria-hidden；原故事独立重复 10 次有 3 次同因失败。故事改为在 Escape 后等待对应 Layout/Display 按钮真正恢复焦点，再按 F6；保留次数、数据库与通知关闭断言，不加 sleep/重试，不改共用组件。修复后相同条件连续 10 次通过，lint-web 通过，完整 E2E 复验 336/336 通过；首次全量原件在 `task-9-first-e2e/`。

## 9. T9 完整本地门禁检查点

最终测试树 `9f990c5d57c96e7a9834dbe5a3ac608414f77be1`，聚合回执为原件 `task-9-9f990c5d-gates.json`。完整 E2E 336 项通过（1.8m）；没有重跑掩盖原 C12 失败，旧失败、同因 3/10 与修复后 10/10 均保留。

| 门禁 | 结果与准确来源 |
|---|---|
| gen + gen-check | `f6cde552` 通过；后续生成输入未变 |
| lint-go | `f6cde552` 通过；后续服务端输入未变 |
| make test | `f6cde552` 完整通过，346.872s，含 server/tools |
| lint-web | `07bd0781` 54/54；C12 修正后再次 54/54 |
| knip | `07bd0781` 通过，无未使用项/配置提示 |
| test-web | `07bd0781` 18/18（17 缓存、1 实跑） |
| build + e2e | `9f990c5d` 通过，336 故事，合计 111.755s |

三个 tested-tree.json 保存全部跟踪文件的 SHA-256、字节数和模式；分段回执列明原始 tested HEAD 与后续精确文件差异，未将旧结果冒记为新执行。最终检查点只补文档，产品输入等价由 `verify-task-9.py` / `task-9-verification.json` 核对。首次产品推送的准确远端记录见 §7/§10，不把本地通过写成远端通过。

## 10. T10 整审与修复复审

独立整审 [re-33](../../../reviews/re-33-M5P1整体审核.md)覆盖基线 fb3e788b 至 ecefb0bc 的 18 个提交。初审 With fixes：一项 Important 为 Linux 停读下载夹具，两个 Minor 为既有焦点竞争和状态文档。主审接受分级；当前不提前宣称最终分支或 main 通过。

- R1：原夹具用 1 KiB 接收缓冲制造背压，退出后动态放大，排空仍受原 3s 客户端期限约束。失败时 handler 已退出；修订为真实 accepted TCP 固定 32 KiB 发送缓冲、客户端固定 64 KiB 接收缓冲，额外观察底层 Close；业务代码、1500ms 退出上界及 3s 客户端期限不变。依旧要求 unexpectedEOF、非完整原字节前缀和下一上传 204，不把 timeout 视为成功。本机 race 重复 20 次通过；删除 idle 写期限的变异在 1.57s 被原退出断言拦截，原实现按字节恢复后 asset/platform/config/arch 全包 race 通过。准确 Linux CI 仍待复验，原件为 `task-10-download-{green,no-idle-mutation}.log`、`task-10-restored-race.log`。
- R2：C12 的真实焦点竞争保留为 Minor，测试等待关闭后置条件只修正故事同步，产品窗口未修。具体后续用例与三浏览器要求写入 re-33，并交后续 UI 阶段；不在 P1 修改共享焦点基础库。
- R3：设计起点、预检原停点和当前事实分开，plan/spec/review 同步准确推送及失败状态，旧失败原件不删除。
- 明确边界：P1 不承诺主机掉电后的强持久性，也未验收 P2 业务链、S3 或第三方可写文件根。

主审修复复核：18906f08 的差异仅两份测试与文档，生产输入与 T9 七门禁逐文件哈希相同；`verify-task-10-inputs.py` 验证了此边界、日志原件与文档链接。监听器仍使用真实 TCP/期限，Close 观测在底层 Close 返回后发出；退出及截断断言没有放宽。R1 本机验证通过，Linux 证明仍以最终准确 CI 为准；R3 已同步，R2 具名保留。准确 CI 收据归入本阶段原件，T11 集成后在本记录追加实际 main 与归档身份，历史失败不覆写。

R1 后续 Linux 证明：run 41460165212 的 attempt 1/2 均通过 asset HTTP 包（4.240s/4.474s）。新的 R4 为组合包累计到默认 10m，不是原传输断言再现。两次被截断用例各刚运行 0s，原件 `ci-41460165212-server.log`、`ci-41460165212-attempt-2-server.log`。同一 bootstrap 输入在旧 CI 为 448.711s、本机为 344.749s，差异核对为空。

R4 裁定：显式 Go 整包预算 15m、server 作业预算 20m，单项测试与生产期限保持。首次按既有环境规则重跑一次仍触顶后处理预算，不继续重跑碰运气。此为局部测试编排校正；完整本机 `make test` 与准确新 HEAD CI 重新验收。其余本地门禁的产品输入未变，验证回执明确列出两份测试、Makefile 和 CI 文件的变化，不把旧命令结果冒记为新命令结果。

R4 本机复验：实际 `make test` 以 `-count=1 -timeout=15m` 完整执行 server/tools，全部通过，bootstrap 349.526s；命令展开和完整输出分别为 `task-10-test-command.log`、`task-10-full-test.log`。生产源码及其他门禁输入保持原字节；最终新 HEAD 的 CI 仍须单独核验。
