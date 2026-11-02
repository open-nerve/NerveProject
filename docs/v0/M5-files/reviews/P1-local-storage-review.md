# M5 / P1 本地存储评审记录

> 状态：T1–T8 实施与局部复核完成；T9 完整门禁、T10 deep-review、T11 集成尚未完成。
> T9 已按 [re-32](../../../reviews/re-32-M5文件路由方案A复核.md)落实 `/static/local`、逐段路径策略与具名开发代理；聚焦回归通过，完整七项门禁及阶段整审仍未完成。
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
| A9 | 原 namespace 回归已按最终授权修复：真实 storage 工作区页面、保留集合和原根路径守卫通过；逐段路径反例、真实 Vite 传输/取消/安全日志及构建冲突守卫通过。完整七项门禁仍待核验 |
| A10 | T1–T8 局部红绿/变异留原件；完整七门禁、独立整审、准确 branch/main CI 和归档仍未完成 |

## 3. 测试

- 每 Task 的准确基线、命令和结果在原件 `progress.md`、`task-N-tests.log` 或具名日志中，不能用后来的通过覆盖早期失败。
- T7：实际关闭行为红（0 次）、clock 复用前红（clock-1.files）；组合 race、7 项 clock E2E、架构、lint/types/format 全绿。重复关闭变异被确切计数断言杀死，恢复后再验收。
- T8：5 个路径规范化反例实得 307，位置与响应正文含签名查询；修复后为固定 404 no-store Problem。原 JSON 控制已存在，新增组合断言先核原行为，再以放宽体积/期限的变异验证敏感性，不伪称新功能首次红。
- 原 JSON 默认 15s 在真实登录查询的 context 边界采样，上下界由请求发送/完成时间给出；实际取消性质由现有 HTTP 平台测试继续验证，不用固定 sleep 同步。
- 文案先见 8 个缺项的守卫红，再补准确映射及两种语言；`make test-web` 18 任务通过（15 缓存、3 实跑）。首次组合配置缺 database.url 与一次工具 cwd 错误属于夹具/命令失误，修正后重跑，不算产品性质红。

- T9 慢流使用真实默认 read=30s、API=15s、JSON=1 MiB、文件预算=55s；基于单调时间按字节节流，实际 31.914s，race 31.917s。移除逐请求读期限的定向变异在全局读期限处失败（499、约 30.6s），恢复后通过。完整七门禁只通过生成检查；其余六项因下面的命名空间回归停止，不能称完整 CI 通过。
- T9 用真实 API 创建 storage 工作区得到 201，工作区 API 为 200，但项目/设置页面都为 404 Problem；仅去掉 asset 路由注册后两页恢复 200 HTML。原 `TestTheReservedServerSlugsAreTheServersTopLevelPaths` 独立复现失败。临时探针已移回原件区，代码按原字节恢复；原件哈希和对照见待裁定方案。T8 聚焦检查没有覆盖既有工作区命名合同，预检也漏掉了该核对，不能放宽守卫迎合当前实现。

## 4. 审查问题

- T5：短体错误曾被同时取消覆盖为 499；保留具体读取失败并区分读后端故障后，实际 TCP 短体为 400、不发布，后端读取故障为 500。
- T6：给下载的后台读设置 idle 会错误结束健康下载；改为下载读侧仅总期限、写侧按实际进度，保持取消优先。新上传上限降低后，已有大文件按可信大小和最长时限读取。
- T8：ServeMux 在 handler 之前规范化路径并重定向，可能把签名 query 带入 SPA。平台提供按路径树声明的拒绝策略，由 asset 声明 `/storage`；固定错误不回显 query/path。无对象访问或其他模块权限改动。
- T8：新 raw 目录最初不参与 web 测试缓存；实际改变目录前后任务 hash 相同。加入准确输入后 hash 改变，恢复目录后 hash 回到原值，并重新运行完整前端测试。
- 阶段 deep-review 尚未进行，本节不等同于整分支终审结论。

## 5. plan 差异与局部裁定

- 已批准的组合边界不变：module 仅构建/注册 HTTP，bootstrap 持有存储 Close；既有 pool 工厂扩为两个明确资源工厂以验证真实所有权，不新增 DI 框架或万能 Storage。
- HTTP 定位读取由消费方另声明 `OpenTransfer`，app 的流式读取端口不扩成 seek 接口；T6 的大测试文件按 fixture/行为拆开，保持文件边界。
- raw 路由/专用码登记在 `api/storage-local.json`；Go 全路由和源码码目录、前端消息表精确并集核对。该文件不生成业务 SDK，也不增加 publicOperations 或宽泛忽略。
- T5/T6 码的同 Task 文案未同步，T8 在首次产品推送前补齐；保留偏差，不声称原 Task 已同时交付文案。
- 错了的代价：原始传输目录与业务 OpenAPI 分开维护，必须保留两侧准确并集和源码核对；路径树策略只覆盖已声明树，新增签名入口必须声明相同政策并补目录。

## 6. 交接

P2 承接准备件、确认/封存候选、业务权限和清理，并把 asset 从 actionlessModules 移到真实 Actions/规则清单；P3 承接 S3；P6/P7 承接浏览器传输与宿主入口。当前不关闭 M5 的完整故事或任何尚需业务链证明的输入交接。

## 7. 集成状态

设计基线 main 的 [CI 41414200682](https://github.com/open-nerve/NerveProject/actions/runs/41414200682) 已全绿且归档清理完成。P1 尚未首次产品推送，也未合并；T9 七门禁后推分支，T10 整审通过后按连续授权执行 T11，不以设计 CI 替代产品 CI。

## 8. T9 命名空间修订实施证据（2026-11-02）

- `868e9a24`：asset/adapter/localprotocol 统一 `/static/local`，static 只调整保留分类，domain/app 无协议依赖。真实 storage 工作区与签发 URL 先红后绿，原根路径守卫、组合 race 和架构测试通过。
- `8e980a7e`：通用 Router 检查规范化中途进入再离开；9 个漏检子例先得 307 并泄露 query，修复后 13 个危险路径与 7 个控制通过。临时恢复二点法会失败，恢复后平台与架构 race 通过。
- 开发插件用 Node 标准库流式传输，仅分派文件树，Go 保留验签/业务预算；独占上游、取消和关闭，日志只含固定错误类别。真实 Vite/TCP 共 11 项测试通过：方法/query/二进制、Range/HEAD/304/private、部分流断连、上游中途截断、关闭、public 与晚发 bundle 冲突。日志含 query、取消失效两项变异均被杀死并恢复。
- 原件：`task-9-namespace-*.log`、`task-9-path-*.log`、`task-9-vite-*.log`、`task-9-proxy-*.log`、`task-9-combined-race.log`。类型检查使用仓库 `check:types` 先生成路由类型；直接 tsc 的缺生成物与测试 fixture 类型错误保留为初轮失败，已修正后通过。
- 夹具校正：macOS 临时目录先取 realpath，避免 Vite 构建根的符号路径差；Vite 把 port=0 当默认值，改用官方 middleware mode 配真实随机端口，避免并行端口竞争。开发服务器的 missing.js 断言明确 JavaScript Accept，生产缺失 chunk 的 404 由 Go 完整程序测试独立验证。未修改这些既有行为。

- 完整七门禁首次运行于 `b051df25258de313f6dcfac7387055c93fa7630a`：gen/gen-check 与 lint-go 通过，make test 用时 347.277s，仅 `TestBuiltInProfiles` 和 `TestEveryModuleDeclaresItsActionsOrHasNone` 失败。前者补齐 spec 中的具体默认值，后者按既有枚举机制登记 P1 没有业务动作的 asset；守卫谓词、精确全路由与角色规则不改。聚焦配置/动作/反例测试转绿，完整门禁在修复后重新执行。
