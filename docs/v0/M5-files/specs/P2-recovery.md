# M5 / P2：对象状态、期限与恢复

> P2 预检子合同，尚未实施。所有具体数值和实现取舍为主控在既定 M5 范围内的裁定；收益/代价见末节。既有 P1 的随机 key、immutable Seal、对象错误及传输预算保持。

## 1. 有界 metadata

`storage_metadata` 为 schema_version=1 的具名 JSON 对象，顶层仅 `schema_version, origin, staging, final, candidates`；origin 为 upload/copy。使用 JSON struct，不使用 map[string]any；读取严格拒绝未知版本/非法 key，SQL CHECK 保障基本对象形状、字段类型、候选数组长度≤3。整个编码上限 16 KiB。所有 key 必须属于本资产 ID；同一 key 不能出现在两个位置。

```go
type ObjectRecord struct {
    Key ObjectKey
    Phase ObjectPhase // active, reclaiming
    SafeAfter time.Time // real runtime clock
    Size int64
    ContentType string
    ETag, SHA256 string // opaque ETag; SHA256 is lower-case 64 hex
}
type Candidate struct {
    ObjectRecord
    LeaseUntil time.Time
}
type StorageMetadata struct {
    SchemaVersion int
    Origin Origin
    Staging *ObjectRecord
    Final *ObjectRecord
    Candidates []Candidate
}
```

尚未 Inspect 的 staging/candidate 的 ETag/SHA256 为空；已发布 final 两者必有。候选目的 key 每次新建，绝不共享固定 seal key；最多三个候选（包括 reclaiming），满额返回 503 asset.busy，不在 API 内做无界清理。metadata 修改持资产 U 锁，`storage_revision bigint` 每次递增；外部结果提交核对预期候选 key、phase、lease、scope，不靠旧整份 JSON 覆盖别人的候选。

上传申请在同一事务记录 staging、真实 `upload_expires_at` 与 `SafeAfter`，再 CPU-only 签名；commit 成功才给调用者 URL。复制申请没有 staging/上传授权，仍先提交目标 pending 行和候选，才做外部复制。失败复制不开放一个假的 upload URL；调用者可重新发起 duplicate，旧目标按准备件和候选规则回收。

## 2. 两种时钟与固定参数

业务时钟只控制：`created_at < businessNow-7d` 的 pending 或未绑定 uploaded 准备件过期；`deleted_at < businessNow-60d` 的墓碑 purge。严格小于；边界相等不删除。过期标记使用本次业务时间，父删除沿用父原 At/By；内部 metadata 回收不覆写原删除审计。

运行时钟控制：签名有效期、I/O deadline、候选 lease、安全宽限、对象/临时文件年龄、worker 预算和调度。进程内预算用 monotonic duration，持久期限保存 UTC runtime 时间，重启后重新比较；业务 offset 不得影响它们。

| 参数 | 默认 / 合法范围 | 用途 |
|---|---|---|
| files.upload_ttl | 15m / 10s–15m | 仅新上传；短值供显式配置及端到端期限验收 |
| files.operation_timeout | 10s / 1s–10s | 一次业务 complete/duplicate 的整个外部 I/O 段总预算（非每步重置） |
| candidate lease | 固定 30s | 从记录候选时起；外部段必须在 lease 内开始，最终提交也必须未过期 |
| candidate settle grace | 固定 30s | LeaseUntil+30s 之后才可认领回收候选 |
| staging SafeAfter | 签发 expiry + 本次大小的 TransferBudget + 30s | 申请时持久化；以后调小配置不缩短已有上传安全期 |
| files.cleanup_interval | 1h / 1m–24h | 周期恢复的调度间隔，RunOnStart |
| files.cleanup_batch | 100 / 1–1000 | 每页记录/对象上限，不一次读全部 metadata |
| files.cleanup_budget | 10s / 1s–30s | 每次 worker 的运行预算；每个外部操作使用剩余预算与 operation_timeout 的较小值，预算耗尽记游标后 snooze |
| orphan/temp age | 固定 24h | runtime 的实物 ModifiedAt/临时目录年龄；不受业务时钟快进影响 |

原始传输预算沿用 P1，不为加快测试缩短生产默认。请求普通 JSON 15s 期限保持；外部段使用剩余 request deadline 与 operation_timeout 较小值，存储实际可取消，不仅外层 select 后丢 goroutine。慢于 10s 的封存/检查可明确失败并留下候选，这一吞吐边界由 P3/M8 容量验收记录，不声称 100 MiB 在任意设备都可于 10s 确认。

## 3. complete / duplicate 的状态转换

1. 按 access 子合同预检/锁下授权，pending → 增加独立 candidate(o-key, active, LeaseUntil=runtimeNow+30s, SafeAfter=lease+30s)；数据库 commit 后放开所有资源。
2. 在一个≤10s 的外部段里 Inspect 源，精确核大小/规范 MIME，Open(expectedETag) 流式校验摘要与图片头；Seal(expectedETag) 到候选；Inspect + Open 候选再次验证实际大小、摘要、格式。封存前后 SHA256 必须相同，ETag 只当不透明版本，不当 MD5。
3. 最终提交重新取账户/父/资产锁并复验权限；scope 未变、候选仍 active 且 lease 未过期、资产仍 pending，才将它移到 final、置 is_uploaded。移除该候选条目，不复制引用；其他候选保留至安全期回收，staging 也保留至 SafeAfter。
4. complete 竞争者若见已有 final：重新授权后返回已有 DTO，不能改写 final/审计。自己的败选候选仍被回收。若在外部阶段报错，也先在短授权事务重读一次已发布结果：只有另一请求确已成功时才可返回同一 DTO；否则原缺失/变化/无效内容/存储错误如实返回。
5. duplicate 最终还核源 final key/摘要及源权限；失败目标保持 pending，不暴露候选。每次 duplicate 是新资产，不提供跨请求幂等键；响应丢失后的重复可能生成额外准备件，7 天规则负责回收。

外部调用成功、DB rollback/取消、进程退出都不能忘掉 candidate。超 lease 的迟到成功不能晋升 final；即使 backend 在 deadline 后才落盘，也只是恢复记录或后续 orphan。客户端旧 PUT 永远只能写 u-key，不持有 o-key 写能力。

## 4. 回收认领与安全

先在资产事务中认领：符合期限的具体 key 从 active 改为 reclaiming；这使任何迟到 complete 都不能发布它。提交后才调用 Delete，随后再短事务核同一 key/phase 并移除记录。删除失败保留 reclaiming；失败/超时不是“不存在”。Delete 缺失幂等成功属于存储合同。任何其他路径不能把 reclaiming 恢复为 active。

- live uploaded：仅可回收安全期已过的 staging、败选候选，final 始终保留。
- live pending：仅可回收 lease+grace 已过候选；staging 保留到准备件过期或删除，不能因上传 token 过期就禁止之前已完成上传的确认。
- deleted：候选/staging 仍满足 runtime 安全期才回收；final 到严格 60 天墓碑期限后回收。bound 的描述图片保留到 owner 删除，不因本文提前加引用垃圾回收；P2 无公开 bound 入口。
- 物理资产行：严格超过 60 天且 staging/final/candidates 全空才 DELETE；SQL 自身含该条件，不能仅靠 app 判断。保护父 FK 到这一刻才解除。

资产自己的周期任务不在持资产锁之后取 owner 锁。读取一页资产 scope → owner 的只读批量删除事实（每种≤batch）→ 锁资产核仍是同一 scope/revision → 应用真实父删除 At/By。父删除不可恢复；scope/revision 已变则跳过到下一轮，不沿旧事实删新宿主。owner 端口只读自己表，包含墓碑的真实 deleted_at/updated_by，不能以 now 伪造。NO ACTION 保证对应父未被物理丢掉；无父事实是异常，不猜成“已删”。

需要新增消费方 `ParentDeletionReader.Deletions(ctx, []Host) (map[ParentKey]DeletionStamp,error)`，bootstrap 按 owner 分组，workspace/project/issue 分别提供固定查询的批量事实；父节点返回的原始删除时间取最早，已有资产删除戳永不后移。账号不物理清理；停用账号不等于业务资产父删除，失去新签发权即可。

同步删除/父范围任务按 access §4；范围任务用资产自己的 `(id > after)` 查询，每批原事务 At/By 不变。周期扫描还覆盖漏投/历史父已删仍活的资产，不能只靠事件一次投递。

## 5. 清单、孤立对象与临时目录

新增 asset/app 的消费方：

```go
type ObjectPage struct { Objects []domain.ObjectInfo; Next string }
type ObjectLister interface { List(context.Context, string, int) (ObjectPage,error) }
type TemporaryCleaner interface { SweepTemporary(context.Context,time.Time,int,string) (next string,removed int,err error) }
```

List token 为 adapter 自有、不透明、版本化的分页位置，非法 token 返回错误。local 只扫描自身 objects 下合法随机 key 的目录；`ReadDir(256)` 流式枚举，保留 after 之后最小的 limit+1 项的有限堆，内存 O(batch)，按 key 排序返回。每次枚举/读取均检查 ctx，目录/metadata 损坏、非法项或符号链接明确报错并不删除它，不扫描根外路径。不把孤立目录当正常 ObjectInfo 返回。

此 local 平坦布局需要 O(N) 扫描/页；deadline 限制单次工作时间而不是宣称任意 N 都能完成。P2 实测 10,000 个目录和≥3 页的有限内存、取消、结果覆盖；M8 记录目标部署规模的全轮成本。若真实目标规模导致单页持续超时，必须再设计索引/分片，不能无限延长期限或静默越过页面。P1 现存磁盘布局不在本段暗中迁移。

对每页对象，先要求真实 ModifiedAt 严格早于 runtimeNow-24h；再在 asset 自己的表按 key 所含 assetID 批量读 metadata，事务内最后核对该 key 没有被任何 active/reclaiming/final 记录引用，才在事务外删除。key 由服务端唯一生成且不能被 API 重新认领，因此“检查后新记录采用旧 key”的路径被禁止。对账不先删 DB 记录；刚落盘的迟到对象会因新 ModifiedAt 等待下一轮。

临时目录只限本 root/tmp 下 local 写入器生成的 UUID 目录；同样 runtime 24h、根约束、有限页、ctx/错误出口。不跟随符号链接，不删除未知布局，不修改别的实例根目录。没有声明多进程共享一个 local 根：M5 仍一个 Go 进程独占目录。暂停超过安全期的旧操作可能失败，不能在 lease 过期后晋升对象；后续迟到落盘由下轮扫出。

## 6. 任务游标与父清理

单次 asset sweep 依次扫描父删除/准备件过期、候选/staging、对象清单、临时目录。游标含 version、阶段、after/token、该轮 cutoff；最大 16 KiB，不含文件名、URL、正文。每页提交后记录进度；预算耗尽返回非 Complete，River adapter 用 `RecordOutput` 写入 job metadata.output 再 `JobSnooze(0)`。重启读取相同游标，正常新周期从空开始，停止后允许重放最后未持久化页，所有操作幂等。

已核仓库固定 River v0.47.0：RecordOutput 在 Work 中合法，Snooze 会保存 MetadataUpdates；不是修改 job.Args 后假定能持久化。实施须真实 worker 回归，不能只用 app 假实现证明。输出编码/版本错误原样失败；不通过 ClientFromContext 或二次 running client 绕过构造。

既有 cleanup.purge_deleted 也改为明确的单页 owner 协议：每个模块外观提供自己的 `PurgeCursor{Table,After}` 与 `PurgeProgress{Cursor,Scanned,Deleted,Blocked,Done}`，bootstrap 转为消费方值；每次只处理一个有限页。全局 cursor 保存模块名、表名、After、单轮固定 cutoff，按 asset → issue → project → workspace → identity 推进，预算复用 cleanup_budget。资产墓碑清理先执行，受保护父候选按 protected-purge 合同跳过，下一周期重看。

每页游标必须体现已经安全扫描的位置；未知 DB 错误不能越过失败候选。已独立提交的删除数与进度和 error 一同返回，River 保存进度再重试。对象故障保留记录且报错，本次 key 可在游标记为已尝试，重试继续邻域、下一周期重看；不能把故障计入 Deleted 或把整个清理写成全部成功。受保护 FK 的 blocked 是精确分类，其他错误不吞。一次 sweep/purge 未结束会阻止相同 periodic job 重叠，保留现有 uniqueness 状态集合。

## 7. 代价与证明范围

局部裁定：30s 候选 lease +30s 宽限减少请求中持锁；代价是进程停顿/极慢封存可能已完成实物却返回冲突，必须留候选且不允许迟到发布。24h 仅是未知残留宽限，已知 staging 用持久化真实 grant/预算，不靠业务 60d 保证传输安全。

局部裁定：沿用平坦目录有界内存分页，避免 P1 布局迁移；代价是每页重复扫描，容量上限需实测，不能声称无限规模。River output 是已有任务状态，不新增业务表/outbox；进程在输出落盘前退出会重放一页，靠幂等处理，不宣称 exactly-once。

必要反例：删除候选记录后外部成功/DB 失败、超 lease 晋升、业务快进61d但真实PUT未结束、调小传输配置、旧 PUT 后到、首个受阻 ID+中间可删+末尾受阻、相同文件名不同世界、目录符号链接、坏 metadata、杀进程后 cursor 恢复、周期间新低 ID/旧失败项再次进入、RecordOutput 失败不可假完成。完整永久测试及真实字节证据在 plan 分配。
