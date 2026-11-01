---
status: open
from: M4/closeout
to: M8
created: 2026-11-02
---

# M4 交给 M8：事件、出站安全、部署、锁等待与性能实数

这是接收方尚待完成的工作。M4 输入的“转交关闭”只表示责任和验收条件已落实到此处，不表示本页功能已经实现或通过验收。
来源为 [M4 设计 §13](../../M4-issue-core/M4-design.md#13-交接的落点)、[逐条交接索引](../../M4-issue-core/reviews/closeout-handoffs.md)和该索引中的各 Phase spec §5 / review §6。
M4 的最终 C12、准确分支 CI 和未来 main 集成分别由[验收台账](../../M4-issue-core/reviews/closeout-acceptance.md)记录；不以阶段历史绿灯替代最终输入。

<a id="M8-01"></a>
## M8-01 Webhook 的事件边界

事件是契约的一部分，当前共有下列 15 个工作项事件，另有 `project.deleted`、`workspace.deleted`：

| 类别 | Type |
|---|---|
| 工作项 | issue.created、issue.updated、issue.deleted、issue.archived、issue.unarchived |
| 评论 | issue.comment_created、issue.comment_updated、issue.comment_deleted |
| 关联 | issue.relation_added、issue.relation_removed |
| 表情 | issue.reaction_added、issue.reaction_removed |
| 链接 | issue.link_created、issue.link_updated、issue.link_deleted |

当前 `shared.Event` 是 ID、Type、OccurredAt、WorkspaceID、ProjectID、可空 ActorID、Entity{Kind,ID}、Changes[]{Field,Old/New *string,OldID/NewID *UUID}；真实定义与生产订阅链见 `server/internal/shared/events.go`、`server/internal/modules/issue/domain/events.go`。同步 publisher 在事务中调用订阅者，异步消费者用 InsertTx 在同事务投递，回滚不留任务；没有既存 outbox 可假定使用。Webhook 载荷、签名和投递策略由 M8 自己设计，不把内部 Event 直接冒称外部协议。

- 普通删工作项对子树每项一个 `issue.deleted`；项目/工作区批量删只发父事件，由消费者处理自己的范围，不虚构逐项工作项事件。
- 唯一没有工作项事件的标签连带是删除标签时摘下 `issue_labels`；M8 给标签增加的事件覆盖它。工作区/项目/成员其他 Webhook 事件接入现有机制，不能遗漏 M3 的原交接。
- 无变化、合法 ID 迁移和元数据爬取不因后台更新造事件；评论/表情有事件却不写工作项动态，两者不可混淆。空 actor 表示系统行为，不造假账户。

**何时验收：** 第一个 Webhook producer/consumer 合入时。覆盖 kind/args 登记、提交/回滚、批量删除入口、系统 actor、权限信息泄漏边界、重试和重复投递策略。

<a id="M8-02"></a>
## M8-02 出站安全与部署环境

Webhook 复用 `platform/outbound`：IANA 两张地址登记表、本机接口地址、DNS **全部候选**、固定 IP 拨号、每次重定向重检和禁止 URL 凭据的边界继续生效。核代理配置、重绑定、私有/保留地址和多网卡；不能用新 http.Client 绕过平台。网络 egress 仍需部署层约束端口探测等范围。

P7 元数据读取有 1 MiB 响应、64 KiB HTML 扫描、5 秒/5 跳预算与位图图片边界、不接 SVG；这些是已有用途限制，不自动成为 Webhook 的产品预算。`outbound.allow_cidrs` 仅供测试，生产不能配置为绕过规则。

**何时验收：** Webhook 可用前；部署说明和运行配置同时落地，独立危险地址/全部 DNS/每跳/无危险 dial 反例通过。

<a id="M8-03"></a>
## M8-03 pg_trgm 原部署要求与 locale

原文完整保留，含标题和来源；相对链接按原出处解释。可点击来源：[M0-P4 → M4](../../M4-issue-core/handoffs/M0-P4-pg-trgm.md)。

<!-- M0-DEPLOYMENT-BEGIN -->
```text
## 部署要求（转给 M8）

`pg_trgm` 自 PG13 起是"受信任扩展"（trusted extension）：拥有数据库 `CREATE` 权限的迁移角色就能安装它，不需要超级用户权限。M8 编写部署文档时，把这条记为部署要求：迁移账户不需要额外的超级用户权限。

来源：[M0/P4 评审记录](../../M0-foundation/reviews/P4-plane-schema-review.md)。

```
<!-- M0-DEPLOYMENT-END -->

M4 的迁移 00029 已在 PostgreSQL 18.6 验证：非 superuser、具有数据库 CREATE 的迁移角色创建受信任扩展和活工作项标题 GIN；已有扩展 Down/Up 保持 OID，Down 只删本次索引。runtime 依赖 PUBLIC EXECUTE，若运维回收则需显式授予实际使用的扩展函数，不授予额外 DDL。

实测库使用 libc / `en_US.utf8`、UTC。部署核 `LC_CTYPE` 和 locale provider，因为三元组及中文两字搜索依赖它；不能把迁移成功等同搜索语义一致，也不能保证三字符词必走索引。旧数据、升级和备份恢复均须验证。Nerve 自己设置 `jit=off`、`plan_cache_mode=force_custom_plan`；M8 在两条真实连接读回，确认池/部署没有覆盖。

**何时验收：** 部署文档与发布演练完成前；留非 owner/runtime 权限、locale、中文/英文搜索、升级/回滚和连接参数的真实证据。

<a id="M8-04"></a>
## M8-04 写闸门、任务与 CLI 的代价

- 当前每进程、每项目 K=2，等待队列 64，max_wait 默认 2 秒、write_lock_timeout 5 秒、server.request_timeout 15 秒；多个进程各自 K，不构成全局 K。
- 等闸门时没有连接和数据库锁；取得后仍可能等池。热项目数达到 `ceil(pool/K)` 时，持连接等项目锁的写仍可占满池。每次数据库等锁各受 timeout，不是整个事务合计只等一次。
- 配置 `max_wait + write_lock_timeout < request_timeout` 只保证闸门和**一次**等锁的预算；R02 已用剩余预算比较修复整数溢出。池等待/多次锁等待仍以请求期限为界；闸门/锁超时是 503 server_busy / Retry-After: 1，耗尽请求期限的写是 500，不能写成全部过载都答 503。
- `nerve users set-email` 在持续写入账户上会等外键 FOR KEY SHARE；M8 给这条 CLI 增加 lock_timeout 和可操作重试说明，M4 未新增账户物理清理。
- 删除/版本保留/purge 的周期、保留天数、批次和 worker 配置依实际 settings；Issue 删除任务的运行预算 delete_budget 默认 45 秒与请求 15 秒分开。观察失败、重试/snooze、积压和耗时；60 天 purge 每条 DELETE 至多 1000 行。自动归档在 `periodic` queue，启动运行并按配置间隔（默认 24h）调度，不写成“仅启动运行”。

**何时验收：** 发布容量方案与 CLI 加固阶段。选定连接池、K、进程数、任务并发，测正常/多热项目/队列满/请求耗尽，保留实际等待与拒绝码；不放宽锁顺序、预检或架构守卫。

<a id="M8-05"></a>
## M8-05 性能原数、已修问题与明确代价

完整场景、每个首样本及 5 个 warm 的 min/median/max、分布、SQL/EXPLAIN/原始失败在 [closeout 性能记录](../../M4-issue-core/reviews/closeout-performance.md) 及其证据根。macOS/arm64、18逻辑CPU、PG18.6；不是生产容量保证，不制造 p95。列表测模块 HTTP handler，不含网络/认证；搜索测只读 RR+Store，不冒称端到端。

| 测量 | 本次实数及限制 |
|---|---|
| 列表 | 10k/20k/100k 各 13 场景，共 39；v0 目标仍为每项目约 2 万个未删除工作项 |
| 20k 泳道空页修复 | 原 3 秒请求超时，Nested Loop 排除 101,995,000 对行；物化 cells 后请求 199.632ms，完整格数/续页/权限不变 |
| 100k 物化代价 | 首屏 SQL 中位 601.118→618.285ms（+2.9%）；空页请求 761.545→1031.342ms（+35.4%），空页 temp read/write 2876/2938→13099/16000 blocks。明确接受的本地修复代价，交 M8 评估规模路线 |
| 搜索 | 100k、13 场景；中文“登录”常见/稀少 SQL 中位 44.679/42.718ms，英文 lo 38.365ms；单项目 login 31.899ms。八个标题词场景保留顺扫成本，不能承诺三字符索引 |
| 规模删除请求 | 单项目 DELETE 6.930ms；五项目工作区 DELETE 8.683ms；各含 20k 工作项及 660k 相关行，不等于任务已经结束 |
| 删除任务 | 单项目 6.282775s；工作区五任务 14.569132–14.594108s；草稿任务 0.002235s，各 1 次 attempt |
| 邻域与闸门 | 工作区任务处理中邻域 GET 最大 66.615ms / PATCH 最大 576.006ms；单热项目闸门 78 个读最大 19.774ms；200 burst 中 134 个 503、66 个成功。保留基线与完整样本 |
| 一次物理清理 | 660,026 行、697 条查询、23 表、每批≤1000，runtime 1 attempt，观察执行 12.214602s；61天删除、59天和存活行不动、严格 cutoff；不是 SLA |

**何时验收：** M8 依据这些数决定大于 20k 项目、100k 临时文件代价、`-created_at` 单列索引查询、个人主页跨项目全工作区扫描、关系高度数及多进程容量。保留 P1b/P3/P5/P9 历史口径，不能选择最快一次或用机器误差抹去已测回归；任何新索引/架构另按负责人纪律裁定。

<a id="M8-06"></a>
## M8-06 全应用核对与文案

- 标注块默认表情已在 M4 改原生，不再第三方图片、blob worker；M8 的原 M1/M2 CSP 事项只剩全应用核验，不重新引入 CDN。
- `CustomSearchSelect` 三个文字及 `AlertModalCore` 按钮文字，等 M4–M7 所有调用方传入后改为必传；全应用核 ModalTitle、焦点/层级/IME/嵌套 Escape。
- 编辑器当前英文操作名称已可访问，但不是全应用中文化；发布期补一致翻译并验证。copy-hook 的 M4 例外已清空；局域网非安全上下文不能复制时应诚实反馈，Firefox 的已知浏览器限制不能靠假成功掩盖。
- M4 原 AST 动态死成员清账与历史 52 项、各包 lint 上限见 [closeout-dead-members](../../M4-issue-core/reviews/closeout-dead-members.md) / [同步核对](../../M4-issue-core/reviews/closeout-sync.md)；未来新读者/删除应重新数，不能永久借历史保留理由。

**何时验收：** M8 发布前，连同许可证、CSP、全应用文案、可达性和动态死成员统一收口。
