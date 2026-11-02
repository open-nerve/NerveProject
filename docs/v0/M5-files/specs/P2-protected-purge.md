# M5 / P2：受保护资产与父行物理清理合同

> 负责人已批准 A；本文件是实施合同。父清理分页与恢复在 T4 落地，真实资产 FK、已知对象回收和目录联合验证已由T5落实；各项实测状态见[阶段评审记录](../reviews/P2-assets-review.md)，不能将预检或局部验证当作全部交付。
> 来源：[裁定与独立复核](../reviews/P2-purge-contract-decision.md)、M4 §3.21/§4.16/§11.9、M5 §4.2/§7/§10。

## 1. 不可丢失的事实

`file_assets` 的 workspace/project/issue/comment/draft/user 业务父引用使用具名、即时、非 deferred 的 NO ACTION；审计 created_by/updated_by 按既有约定 SET NULL，账户本身仍不物理清理。用途 CHECK 要求的宿主字段保持；不以 SET NULL 制造无宿主却仍可访问的资产。

资产行仍有 staging、final、候选等任何待清理 key 时，不可物理删除。对象删除成功后在资产行锁下核对同一 key/代次，再移除相应记录；失败/未知结果保留原记录。其他任务不能把一次 timeout 当成不存在。旧能力或后端迟到操作可能在已完成的删除之后留下新孤立对象，由独占 namespace 的对账处理；不声称数据库和对象存储有原子提交。

绑定字段/历史内容的解除由宿主负责；资产模块不写父表。新表的全部 incoming/outgoing FK 与索引纳入目录，含 60 天内资产仍能阻止更老父墓碑提前级联清除。

## 2. 父候选如何前进

采用数据库依赖检查，不在父模块复制资产范围查询。对可能传递删除资产的父表，owner adapter 先按现有 `(id > after ORDER BY id LIMIT batch)` 选取有限候选，再用自己的 sqlc DELETE 尝试整批：

1. 整批成功：返回真实删除数与本次最大的候选 ID。
2. PostgreSQL 返回精确资产保护 FK 的 23503：该自动提交语句整体回滚；再按已选候选的 UUID 升序逐个执行相同带 cutoff 的 DELETE，各自独立短事务。
3. 单个候选再次命中同一类保护 FK：记录 blocked，继续后面的候选。它不是成功删除，也不是无对象；下一轮周期从头重看。
4. 其他错误（未知 FK、权限、连接故障、40P01、55P03 等）原样返回，保留已提交删除数，交既有重试，不用“受阻”吞掉故障。

每页输出 `Scanned`、`Deleted`、`Blocked`、`After`；下一页按扫描位置前进。只有 `Scanned == 0` 才算该表耗尽，不能以 `Deleted == 0` 提前终止。选择后被其他事务删掉或不再满足 cutoff 的候选也计入扫描位置，删除数仍按真实 RETURNING 行计。

普通子表同样区分扫描与删除：一条自有固定 SQL 用有限候选 CTE 和 DELETE CTE，返回候选数、实际删除数与候选最大 ID；DELETE 再核 cutoff。并发删除或候选时间变化不能把已扫描页说成空页，也不能删除等待后已不满足 cutoff 的行。仍为一条快速批删，没有新增父锁；代价是有限候选的物化。T4 已用实际请求 PID、SQL 与 blocker 复现并验证这两个方向。

SQL 仍由各 owner 的 sqlc 生成，只访问自己的表。基础设施仅提取 PostgreSQL 错误身份和执行上述受保护批次算法；调用方提供自身生成查询，不传跨业务模块事务闭包，不让共享层拼表名或执行业务 SQL。适配器持有的保护标识来自 bootstrap 的明确构造参数，构造后不补注入、不运行时查找服务。

## 3. 类型和构造边界

以下是 T4 已落地的基础设施合同：

```go
// platform/postgres 的基础设施值；Schema/Table/Constraint 必须全部匹配。
type ProtectedReference struct { Schema, Table, Constraint string }
type PurgePage struct {
    Scanned, Deleted, Blocked int
    After uuid.UUID
}
func ProtectedForeignKey(err error, references []ProtectedReference) bool
```

`ProtectedForeignKey` 仅接受 `*pgconn.PgError` 经 errors.As 得到的 23503，且三项身份精确命中。测试中的 `public/probe_asset/probe_asset_issue_fk` 已由真实 PostgreSQL 返回验证；生产需使用真实 `file_assets` 命名验证。不得按错误文本或仅按 SQLSTATE 分类。

asset 模块外观交出 `ProtectedParentReferences()` 的新副本（不得暴露可变全局 slice），来源为其明确的迁移约束。bootstrap 转成父 adapter 的上述消费方值。issue/project/workspace 的 Provide 构造参数新增明确的保护引用输入；app 不 import platform 类型，自己的 PurgePage 由 adapter 转换。identity 无受保护父步骤，但同样改为有限单页协议。

批次算法的基础设施消费者只接受两个小操作：`Candidates(ctx, cutoff, after, limit) ([]uuid.UUID, error)` 和 `Delete(ctx, cutoff, ids) ([]uuid.UUID, error)`。各 owner 以具名 adapter 绑定它自己的固定 sqlc 查询。不得根据外部输入选 SQL、表或列。普通子表保留单条快速批删；只为目录推导出的受保护祖先使用先选候选、批删失败再逐候选的方式。实际基础设施入口为 `PurgeProtected`，owner 的 `Purger.Page` 每次只执行一个步骤的一页；整轮循环只在有运行预算的 bootstrap 编排内，测试中的 drain 不进入产品。

阶段需要处理的祖先至少包括 `issue_comments`、`draft_issues`、`issues`（含自引用）、`states`、`projects`、`workspaces`。不是按这份手写列表豁免未来节点：目录测试独立沿 CASCADE 边反向寻找通往资产 NO ACTION 边的所有可清理祖先，再与受保护步骤声明精确对照。

## 4. 数据库保护与回收任务的职责

保护 FK 只阻止错误删除，不负责清理文件。asset 的同步删除协作、project/workspace 的原事务投递和周期恢复必须同时落实，不能把永久 23503 当作成功闭环。

- 同步 issue/comment/draft 删除在原事务标记资产，保持原 At/By，重复不覆写审计；只在真正 DeleteDraft 路径执行删除，不能让 PublishDraft 的内部软删误删将迁移的资产。
- project/workspace 删除在原事务投递 asset 的有界范围任务；权限随父范围删除立即失效，不等任务。
- asset 周期恢复经 owner 批量读取父范围的删除事实，补偿过期父仍有未处理子资产；不跨模块 JOIN。运行失败保留父行及资产恢复记录，恢复后再推进。
- `bootstrap` 的物理清理顺序为 asset → issue → project → workspace → identity。资产对象 I/O 在业务事务/连接之外；提交 metadata 变化时只锁本资产，不倒过来追父锁。
- `files` 清理任务、原 purge 和请求写的交错要用实际 PID/阻塞行证明；23503 的保护与允许 40P01 重试是不同合同，不把前者吞掉所有数据库错误。

暂时无法回收的范围不会阻止后面的无关候选。完整轮次在有限工作预算后按持久游标继续；进程退出允许重复检查已处理页，但不能丢恢复依据。具体任务游标与配置已写入 [对象恢复合同](P2-recovery.md) §2/§6；父清理游标已由 T4 验证，T5已接资产范围/已知对象任务，T9继续孤立对象和完整重启恢复。

## 5. 守卫演进

原目录测试仍从真实数据库独立读取所有表和 FK；缺清理者、重复清理者、漏 FK、过时 action、未知 NO ACTION/RESTRICT 全部失败。

新增受保护类别须同时满足：

1. FK 精确为 asset 声明的 schema/table/constraint/column/parent，实际为非 deferred NO ACTION，子列具备无条件索引；多列 FK 未设计时明确拒绝。
2. metadata 非空不能物理删除的真实 SQL/行为证据；对象删除失败时记录和父行均保留。
3. 传递祖先集合与 owner 的受保护步骤集合相等，含自引用；任意新增/遗漏祖先不能静默通过。
4. 每个受保护父步骤有“最小 ID 受阻、中间可删、最大 ID 受阻”的实际数据库用例，下一轮恢复后全部收敛。
5. 将保护 FK 改为 CASCADE、把未知 23503 当 blocked、以 Deleted==0 结束分页、跳过受保护父步骤中的任一项，至少对应一个定向反例失败。

这不是允许任意 NO ACTION，也不是 asset 表名例外。M4 旧目录与父 CASCADE 回归保留，fixture 延伸为实际资产，并明确文件依赖仍存在时的新保留语义。无资产的原行为和 59 天/活邻域保持。

## 6. 已完成的规划探针与局限

预检探针在独立测试数据库中模拟 parent → issue → asset 以及 issue 自引用；实际 NO ACTION 报错能保留叶子约束身份，整批回滚完整，逐候选检查得到 Scanned=3、Deleted=1、Blocked=2、After=3，移除恢复记录后下一轮可删除剩余两个父行。精确 schema/table/constraint/SQLSTATE 另经真实 PostgreSQL 核对。

原件位于本工作树 `.superpowers/sdd/P2-assets/`：`p2_purge_forward_probe.source.go`、`purge-forward-probe.log`（2.062s）及 `p2_purge_identity_probe.source.go`、`purge-identity-probe.log`（3.315s）。这些规划探针源执行后移出产品目录，当时没有产品实现或守卫改动；后续永久实现与测试另按 Task 留证。

这些证明 PostgreSQL 语义和所选分页策略可行；尚不证明真实 asset 表/对象删除、全部 owner 接线、锁序、规模性能或崩溃恢复已通过。实现时必须把性质转成真实模块与完整程序的永久测试，不能用探针代替阶段验收。

## 7. 局部取舍与代价

主控选择“整批先试，受保护失败才逐候选”：保持无依赖的常规批删效率，并让受阻页可前进；代价是有受阻行时多一次失败批删和最多 batch 次短事务。没有吞未知错误或放宽数据库保护，规模测试需记录这个分支的成本。

主控选择以数据库实际 FK 检查作为最终依赖判定，避免在各 owner 复制一套跨模块存在性判断和检查/删除竞态。代价是受阻属于可预期数据库失败，需要精确分类、可观察计数和恢复任务；不能仅让日志无限重复报外键错。
