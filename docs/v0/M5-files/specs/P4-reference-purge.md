# M5 / P4：反向图片引用与物理清理

> P4必要子合同，预检通过、尚未实施。沿用负责人已批准的P2数据库兜底方案，不削弱任何file_assets→业务父行的保护。

## 1. 必须同时成立

新增users.avatar_asset_id/users.cover_image_asset_id/workspaces.logo_asset_id/projects.cover_image_asset_id四个反向FK，均为单列、即时、非deferrable、NO ACTION；目标file_assets.id。每个子列有无条件索引。项目builtin CHECK仍独立。FK名称分别为users_avatar_asset_fk、users_cover_image_asset_fk、workspaces_logo_asset_fk、projects_cover_image_asset_fk。

它们不是自动清空机制：asset仍有staging/final/candidate时不能删恢复行；有业务引用时也不能直接物理删资产。所属模块显式解除失效引用，不能由asset SQL UPDATE users/workspaces/projects，不能用CASCADE/SET NULL、禁用约束或目录豁免代替。

当前绑定被替换时owner已同事务移除旧引用；workspace/project软删后仍可保留墓碑中的引用，直到资产对象已清完且资产墓碑超过既有cutoff。用户账号不物理清理，头像/封面通常在替换时解除；通用清理仍覆盖两个列，避免未来删除入口遗漏留下环。

## 2. 一个有界页

app.Purger.Page的对外PurgeCursor/PurgeProgress形状不变。内部改为：

1. 无长期连接读取最多min(batch,1000)个eligible资产ID，条件仍是id>after、deleted_at<cutoff、metadata.staging/final为JSON null且candidates=[]，UUID升序。batch沿现有cleanup.purge_batch，输入仍校验1..MaxInt32；新资产引用页内部上限1000，保护候选数组/锁集合，较大配置通过更多页完成、不截断整个轮次。选择本身有限；不枚举整个owner表。
2. 空候选时Scanned=0、Done=true；非空候选记录maxID，开一个短事务。
3. bootstrap注入的具名引用端口按identity → workspace → project调用各owner的 `DetachAssetReferences(ctx, []uuid.UUID) error`。每个owner只用自己的固定sqlc SQL，先按本表id升序锁实际引用这些候选的owner行N，再按**当前字段仍等于候选ID**条件置null；users一次处理两个字段，不按输入asset顺序逐用户加锁。与清理无关的字段、原updated_at/deleted_at/by均不改。
4. 最后asset自身固定DELETE ... WHERE id=ANY(candidates)并重新核cutoff与metadata空，返回实际删除ID。FK是最后防线。提交成功后Page返回Scanned=len(candidates)、Deleted=实际数、After=maxID、Done=false，即使Deleted=0也前进。
5. 解除/DELETE/commit失败整个页回滚，保留输入cursor，不把未提交的扫描位置记为完成；错误原样交现有重试。unknown FK、40P01、连接错不当blocked或无对象。commit结果未知可重复本页，条件解除/删除均幂等。

所有对象I/O仍在P2 cleanup事务外，P4不在这个数据库页里调用存储。eligible是单向状态：公开API不能恢复deleted资产，后续确认/绑定/复制不能向它新增活对象或引用；清理只移除对象记录。若有候选已被别的purger先删，条件解除和DELETE可为0，仍按扫描位置前进。对账处理后来才出现的无记录对象沿P2/P3原协议。

锁序为本页涉及的users(N，升序) → workspaces(N，升序) → projects(N，升序) → 资产删除锁。这些是维护已失效引用的owner操作，既不重新授权也不回取缺席父锁。新请求不能绑定已删候选；如果当前owner已换成活新件，WHERE当前引用判断保留新件。用户操作仍账号第一把、W→P→asset，清理不得asset先锁再反向更新owner。

被文件故障挡住的资产不满足metadata空，不会进入eligible页；其父行按P2 ProtectedPurge继续blocked并推进其他父候选。P4不为仍有对象的资产解除可恢复事实，也不把文件故障变成全页永久停顿。

## 3. 小接口和构造顺序

```go
// asset/app
type ImageReferenceDetacher interface {
    DetachAssetReferences(context.Context, []uuid.UUID) error
}
type AssetPurgeCandidates interface {
    PurgeAssetCandidates(context.Context, time.Time, uuid.UUID, int) ([]uuid.UUID, error)
}
type AssetPurgeDeleter interface {
    DeletePurgeAssets(context.Context, time.Time, []uuid.UUID) ([]uuid.UUID, error)
}
```

各owner的Provide返回只操作本模块表的ImageReferences端口，明确声明其拥有的引用列。bootstrap的assetImageReferences由三个具名端口组成，其DetachAssetReferences只按上面的固定顺序调用；不接收任意函数、不从表名动态选服务。

asset.Provide(pool)仍只建自有存储/绑定所需事实，不创建缺少owner依赖的Purger。新增根外观 `asset.NewPurger(Provided, ImageReferenceDetacher, shared.TxManager)` 一次构造完整app.Purger，替换原Provided.Purger；不保留旧的无解除能力构造通道或事后Setter。bootstrap在五个Provide后就可接好，不需要等owner.New。所有调用方及组合测试同时迁移，原Tables()仍只报告file_assets。

bootstrap整体purge模块顺序仍asset → issue → project → workspace → identity；只是asset墓碑页内部先解除四个引用。现有River持久cursor的Module/Table/After/Cutoff/Version不变，暂停/恢复仍有限页；重启不重新计算旧轮cutoff，也不回退已提交页的位置。P2已有父候选协议不另写一份。

## 4. 目录守卫与真实夹具

真实数据库目录独立枚举所有incoming/outgoing FK；新增四个反向边与三个owner实际提供的解除能力做精确集合对照。分别验证schema/table/constraint/column/target/action、单列id目标、非deferred、无条件索引、每owner恰有一个解除步骤及其顺序，声明返回副本。

原assetProtectionGaps仍验证file_assets六个业务父引用与全部CASCADE祖先/ProtectedTables，不能把四条反向边混入“资产父保护”的声明。purgeCatalogGaps只在上述额外证明成立时接受这四条确切NO ACTION；任意新NO ACTION/RESTRICT、漏表/漏FK/漏清理者原负例继续失败。不得按表名前缀忽略incoming FK。

真实fixture通过API绑定用户图、workspace logo、project cover，再形成父删除/对象故障/60天墓碑。泛用retention fixture、catalog/row-deletion守卫和资产恢复测试同步补反向引用；没有改变用途或绕过FK的“清库”捷径。SQL只用于读取断言和显式故障注入，不用SQL伪造F5–F7已上线。

必要反例：
- 直接DELETE有对象资产与其父，原恢复依据均保留；直接DELETE有反向引用的eligible资产仍23503。
- 删除对象失败/metadata尚有key，引用及恢复行保留；其他无关eligible页与父候选仍前进。
- 原owner引用已换活新件，清理旧ID不能置空新引用；解除后事务失败/最后DELETE错误，原引用回滚。
- 两用户/两workspace/两project反序ID和同用户两个字段，实际锁顺序与并发替换/停用不成环。
- 61天empty、59天empty、61天nonempty、活件四类同时存在；batch1分页不漏后续，cutoff与审计不漂移。
- 业务父还老、asset年轻时不能CASCADE抹除；恢复后解除→资产→父收敛；P2states/issues自引用路径仍在。
- 真实新进程恢复在一个页提交后/一个页失败后分别继续，metadata与引用不分裂。

至少五个行为/守卫变异：反向FK改SET NULL/CASCADE、漏任一引用解除、先锁asset再owner（实际PID环/超时反例）、不检查当前ref而清空、用Deleted==0结束分页。原P2父NO ACTION和未知23503负例保留；不能用编译错误算性质证明。
