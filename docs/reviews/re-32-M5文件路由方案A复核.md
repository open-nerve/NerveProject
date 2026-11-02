# Deep Review：M5 文件路由方案 A — 根因、架构与风险收益

审查日期：2026-11-02。模式：doc，目标为[命名空间决策文档 §3](../v0/M5-files/reviews/P1-storage-namespace-decision.md#3-方案-a推荐使用已保留的服务端命名空间)。分支 `codex/m5-p1`，审查 HEAD `d4a63d1fa0bd2b56df7aa196198e6c43dcf3184b`，main 基线 `fb3e788b7be33467018385c9a67d515e51b79dad`。

负责人先同意 A，随后明确暂停实施、要求独立复核。本轮只有审查文档和隔离探针，未迁移产品路由，未修改设计/spec/plan 的实施合同，未提交、推送或合并本轮报告。当前产品仍有原 `/storage` 遮蔽工作区问题，不能合并。

## 1. 结论

**A 能解决工作区命名冲突的根因，也可以保持 SOLID；但原建议比较不充分、漏了必要安全边界，不能原样认定为当前最优并立即实施。修订建议是使用 M3 已为文件预留的 `/static/local/{key}`，由 asset 独占 `/static` 传输树。**

主审按 deep-review 的逻辑、集成、测试三视角顺序核对；另按 requesting-code-review 请 `m5_p1_namespace_review` 在无先前对话上下文的情况下只读独立审查。两者结论一致：A 有条件可用，预留根方案在当前仓库中边界更清楚；无论路径选哪一个，下面的通用路径判定与开发代理日志问题都必须处理。本次是方案审查，不能替代后续实现验收和 P1 整分支 deep-review。

| 负责人关注点 | 复核判断 |
|---|---|
| 是否根因解决 | 是。根因是服务器占用了合法工作区的顶层名字；迁至已保留的名字消除两类地址重叠。仅缩小 `/storage/local` fallback 不解决顶层归属问题。 |
| 是否当前最优 | 原建议只比较 A 与工作区迁移，遗漏 M3 明示允许的 `static` 预留段。综合现有构建、缓存和命名合同，更推荐 `/static/local`。 |
| 是否符合 SOLID | 路径文字本身不能证明 SOLID。保留既有端口和组合方式、不将平台与 asset/webui 相互耦合时，A 与修订方案均符合现有分层。 |
| 是否有架构回归风险 | 有可定位的边界风险：A 与构建资源共用 `/assets`；现有通用规范化判定有漏检；新增 Vite 代理会默认记录失败请求的签名 query。 |
| 风险与收益是否平衡 | 使用已有预留名字，保留工作区地址和数据，代价可控。为尚未发布的 `/storage` 迁移工作区不划算；扩展通用路由框架或改变 `/api/` 缓存合同也没有当前收益。 |

## 2. 根因和替代方案

已保留的真实 API 证据是：创建 storage 工作区 201，读取工作区 API 200，项目/设置页面 404；只移除 asset 路由注册后页面恢复 200。新方法无关的 `/storage/` fallback 在 SPA 之前截获页面；不是数据、认证或前端动态路由出了问题。

根因代码为 [transfer.go:73](../../server/internal/modules/asset/adapter/http/transfer.go#L73)、[reserved_slugs.txt:18](../../server/internal/modules/workspace/domain/reserved_slugs.txt#L18)，既有精确守卫在 [bootstrap/reserved_test.go:55](../../server/internal/bootstrap/reserved_test.go#L55)。原件和哈希继续由原决策文档指向，不用本轮探针替代真实工作区复现。

M3 §3.10 已把 `static` 预留给“静态文件”，并明确允许启用预留段时将其从 reserved 移到 server/app。见 [M3 设计 §3.10](../v0/M3-workspace-project/M3-design.md#310-工作区名规则与保留名单m1-p2m1-p3m1-p4-交接)。这条现有规则是修订建议的依据，不是临时扩大保留名单。

| 方案 | 具体收益 | 代价与结论 |
|---|---|---|
| A：`/assets/storage/local/{key}` | 不改保留名单分类，合法工作区不迁移 | 必须划分构建资源与签名传输子树、禁止构建占用该子树，维护相邻资源的代理与缓存边界。有条件可用。 |
| 修订建议：`/static/local/{key}`，保护整个 `/static` | 使用已预留的文件名字；构建资源仍归 `/assets`，文件传输有独立根；保留名称总集合不变 | 同步 M3 分类和剩余预留名称的准确测试，约束构建产物不占 `/static`。不需更改合法工作区数据。当前更推荐。 |
| `/api/storage/local/{key}` | api 已保留，开发代理已有 `/api` | M2 §8.3 规定整个 `/api/` 回答 no-store，文件读取约定有限 private 缓存；需要改变其中一项合同。handler 技术上能覆盖响应头不等于获准改变合同。 |
| B：保留 `/storage` | 保留尚未发布的文件 URL | M3 的 slug 创建后不可改；需规划工作区迁移和旧链接处理。收益不足以覆盖兼容性代价。 |
| 仅去掉 `/storage/` fallback，保留 `/storage/local` | 当前 projects/settings 可能恢复 | 服务端仍在合法工作区树内占路径，既有顶层守卫仍失败，留下未来页面冲突，不接受。 |
| 新增未预留根、独立文件域名 | 能形成独立入口 | 前者重现同类迁移问题，后者新增部署、TLS/CORS/CSP 成本；本次没有必要。 |

`static` 分类转移时必须验证 **All 保留集合前后相同**、`CheckSlug("static")` 仍拒绝、`storage` 仍可用；保留 [server 根路径精确相等守卫](../../server/internal/bootstrap/reserved_test.go#L55)，只同步 [剩余预留名称的期望](../../server/internal/modules/workspace/domain/reserved_test.go#L33)。不能删断言、改成包含判断或增加忽略项。

## 3. Structural Issue / Major：原方案漏掉的必要边界

### R1. 通用路径保护只看原路径和最终路径，遗漏中间进入受保护树

位置：[routes.go:111](../../server/internal/platform/httpserver/routes.go#L111)，原方案 §3 第 2 项。根因是 `original` 只去掉前导重复斜杠，另一个比较值是最终 `path.Clean`，两者都可能在受保护树外。

本轮通过 Go overlay 加入审查探针，直接调用**未修改的真实 Router**，仅在探针里声明候选前缀。对于 A：

```text
GET /assets//storage/local/../../../settings?payload=review-token&tag=review-tag
实际：307
Location: /settings?payload=review-token&tag=review-tag
```

`/assets/./storage/...`、编码首段的相应变体也复现；直接位于规范 `/assets/storage/...` 的对照返回 404 no-store、没有 Location。

进一步的 `elsewhere/..` 探针在原 `/storage`、A 的 `/assets/storage`、修订候选 `/static` 三者上都返回带 query 的 307。因此它是已有通用策略的缺口，A 的多段前缀扩大了明显反例；**不能声称改用 `/static` 就修好了此问题**。

影响：违反已约定的“签名 query 不随路径规范化送到其他端点”。这里只使用假的审查标记，未证明有实际用户凭据泄露，也未证明能绕过验签读取文件。

修订要求：通用机制按规范化过程中的路径归属判断，拒绝会进入或离开已声明签名树的非规范请求；覆盖重复斜杠、点段、编码组合和进入后再退出。保持机制通用，不写 asset 名字特判，不禁止整个 `/assets` 的正常规范化，也不放宽原 API/页面规则。这项应纳入 T9 修订与实现复审。

### R2. 新增 Vite raw 代理的默认错误日志会记录签名 query

位置：原方案 §3 第 3 项，[vite.config.ts:17](../../web/apps/web/vite.config.ts#L17)，已安装 Vite 8.0.16 的 `dist/node/chunks/node.js` 第 17932–17938 行默认 proxy error handler。它打印 `res.req.url`，而 M5 §5.3/§10 禁止记录完整签名地址。

本轮启动仅本次探针拥有的临时 Vite 和本地上游，配置文件隔离、未修改应用配置。让上游不可达后，响应 502，Vite 的原始错误消息包含：

```text
http proxy error: /assets/storage/local/key?payload=review-secret-token&tag=review-secret-tag
```

这是 A 与 `/static` 新代理的共同要求。必须在开发代理接入处控制这类日志，保留可诊断的路径/错误类别，去掉 capability query；不能因为 Go 日志已经脱敏就宣称整条链完成。不能通过关闭全部错误日志掩盖失败。

### R3. 原比较遗漏现有预留段，A 的资源所有权说明也不完整

位置：[原方案:36](../v0/M5-files/reviews/P1-storage-namespace-decision.md#3-方案-a推荐使用已保留的服务端命名空间)、[webui/handler.go:12](../../server/internal/platform/webui/handler.go#L12)、[emojibase.ts:37](../../web/apps/web/emojibase.ts#L37)。

`assets` 当前是 Vite 构建资源目录；已有 emojibase 插件会输出具名子目录，不能假设未来构建永远只生成平铺的 hash 文件。现有 public/build 中未发现 `assets/storage` 或 `static` 碰撞，但这不是永久排他保证。

A 在当前 Go 路由组合中**不会自动获得公共长期缓存**：raw 路由比 webui 的 `/` 更具体，raw 读取设置 private，错误设置 no-store；只有实际由 webui 送出的构建文件才获得 immutable。A 的代价是引入并长期维护子树归属，而不是已经发生缓存泄漏。若保留 A，必须明确和验证构建/public 不得占用 `assets/storage`，同级静态文件与缺失 chunk 维持原语义。修订候选则令构建与传输分属两个根，仍应验证构建不占 `/static`。

## 4. Local Fix / Minor：实施细节不能含糊

1. **Vite 的精确匹配不是普通字符串前缀。** 实际 Vite 探针中，键 `"/assets/storage"` 把 `/assets/storage-review.js` 误送给上游；带段边界和根 query 的正则 `^/assets/storage(?:/|\?|$)` 则正确保留该静态文件。规范 raw 和根路径带 query 均到达上游。修订方案同样须区分 `/static` 与合法工作区 `/static-more`，并保持方法、字节、必要头和原 query，不先规范化签名目标。这与 [Vite 官方 proxy 规则](https://vite.dev/config/server-options#server-proxy) 相符；本轮还读了安装版本源码并运行真实服务器。
2. **路径是 adapter 的传输合同。** 当前 local 签发、HTTP 注册与精确校验分别写着前缀（token.go:43、transfer.go:75、upload.go:11、read.go:19）。迁移可用一个小型 adapter 协议定义减少漂移；不把路径常量移进 domain/app，不让 local 直接依赖 HTTP handler，不引入全局路由注册框架。机器目录与实现之间继续独立精确核对，不用共享同一错误值让测试互相证明。

## 5. SOLID 逐项核对

| 原则 | 必须维持的边界 |
|---|---|
| 单一职责 S | webui 管构建文件和 SPA；asset HTTP 管签名传输；Router 管通用匹配/规范化；bootstrap 组合。不能在 webui 中塞 asset 验签或存储读写分支。 |
| 开闭 O | 经既有路由注册和明确路径策略增加端点，通用策略修复其原本承诺；不靠一组针对 URL 的零散例外。无需新框架。 |
| 替换 L | `SignedRequest` 仍提供真实可用的目的 URL/方法/头；调用方使用返回值。未来 S3 可以返回预签名远端地址，不强迫 S3 实现本地路由。 |
| 接口隔离 I | 保持 UploadSigner、ReadSigner、ObjectReader 等小接口；本地 HTTP 的 seekable TransferReader 不进入通用存储端口。 |
| 依赖倒置 D | domain/app 不依赖 Router、webui、Vite；platform 不 import asset；跨模块仍只在 bootstrap 接线。 |

未发现 A 或修订候选必然违反上述原则的证据。它们的区别主要在路由所有权、缓存与构建的集成成本；不能把 URL 选择本身包装成 SOLID 的证明。

## 6. Missing Tests：实施前应写入 spec/plan 的必要验收

- [ ] `bootstrap/asset_namespace_test.go`：经真实 API 创建 storage 工作区、读取、直接访问 projects/settings，页面 200 HTML；原 M3 根路径守卫原样精确通过。复用已有探针和夹具。
- [ ] `platform/httpserver/routes_test.go`：`TestSignedPathNormalizationDoesNotRedirectQuery`，覆盖规范前缀、前缀内部重复斜杠/点段、编码、从树外进入后再退出；无 Location、无查询回显，邻近普通路径保留原规则。完整组合另核对实际声明的前缀。
- [ ] 现有 local/HTTP 共享合同：真实签发结果 PUT→Inspect→Seal→GET/HEAD、Range/304/到期拒绝及 private/no-store；旧 `/storage` 不再是 raw 入口，不保留重定向别名。不仅机械替换所有测试字面量。
- [ ] `web/apps/web/transfer-proxy.test.ts`：真实 Vite 与上游，检查 raw 根和子路径、相邻普通页面/资源、PUT 字节/头/query、取消和上游不可达；日志不出现签名标记。具体测试帮手沿用仓库风格，名称是建议落点，不是已新增文件。
- [ ] 完整程序和真实构建产物：JS/CSS 与 emojibase 仍可用，普通 hash 资源 immutable，缺失 chunk 404，raw 错误 no-store；构建不得占传输根/子树。
- [ ] 若采用 `/static`：保留名称集合前后完全相同，static 仅从 reserved 转入 server，剩余预留名称严格匹配；不增豁免，不修改 slug 可变性或工作区数据。
- [ ] 普通 JSON API 的认证、bodyshape、1 MiB/15 秒、no-store、publicOperations 和全程序路由目录保持；随后完成既定七项门禁、P1 整审与修复复审。

## 7. 本轮证据和边界

原件根为本 worktree `.superpowers/sdd/P1-local-storage/namespace-review/`；`evidence-manifest.json` 记录审查原件字节数和 SHA-256，完整临时 Vite 上游与服务已关闭。没有操作 Docker 或实际业务库。

| 验证 | 结果与限度 |
|---|---|
| Go overlay 的真实 Router 两组探针 | 7 个子例，1 个规范前缀对照通过、6 个安全期望失败；准确复现上述 307。只新增虚拟测试，不改产品源文件，也没有实现候选路径迁移。 |
| 真实 Vite 8.0.16，两个代理配置各 3 请求 | 普通字符串误代理同名前缀 JS；正则边界正确区分 3 个目标。该探针用隔离配置，不是应用的开发服务器整体验收。 |
| Vite 上游不可达 | 502；原始错误消息包含假的 payload/tag 标记，证实 R2。首轮探针曾遇 socket hang up，保存为 setup-failure；改用独立连接后得到上述有效结果，不把夹具失败算产品反例。 |
| 现有 workspace domain 保留名单 3 项 | `go test -count=1 -run '^(TestTheReservedListIsWellFormed|TestParseReserved|TestReservedReturnsACopy)$' ./internal/modules/workspace/domain` 通过；仅证明当前名单和解析边界，尚未执行分类迁移。 |
| 独立审阅者 | 只读完成，没有运行测试；主审另行复现路由与代理发现。摘要在 `independent-review-summary.md`，不将摘要当作执行日志。 |

审查中暂不判定：整个 P1 的存储生命周期/并发性能、尚未实施方案的全链路正确性、未来 S3 服务兼容性、实际部署的代理/CDN 缓存规则、绕过受支持创建路径写入的数据。关于不需迁移的判断限于遵守 M3 既有规则的合法工作区；未声称盘点过实际部署。

**实施继续暂停。** 本报告修正推荐，不自动把用户对原 A 的同意解释为对 `/static` 的授权，也不把方案审查通过记作 P1 完成。

### 后续授权

负责人随后明确“按照最终敲定方案，继续推进”，已授权 `/static/local` 修订方案与上述必要边界修复。本文保留审查时快照；实施状态以 P1 plan/阶段 review 为准，未将方案判断当作产品验收。
