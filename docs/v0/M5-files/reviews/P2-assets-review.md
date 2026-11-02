# M5 / P2 阶段评审记录

> 实施中：T1–T4完成，T5–T14待执行；没有进行整分支终审、没有合并。基线main85dc4087，预检提交72a3e58b，见[re-34](../../../reviews/re-34-M5P2预检.md)。原件在本工作树 `.superpowers/sdd/P2-assets`。

## 1. 方式

主控按executing-plans逐Task实施/TDD/自查，阶段末deep-review三视角与一次新鲜整分支审查。P2完整builder check仅为设计预检，不当产品审查。

## 2. 验收进度

A1–A12均保持未完成。T1提供用途/范围/父组合、metadata≤3候选/16KiB及受限图片校验；它不提供API、数据库或完整恢复链，不能据此勾选阶段标准。

## 3. T1已完成的验证与裁定

- domain/content 的setup编译红单列；最小零行为声明后真实性质红，最终绿。四格式实物头、完整字节hash、同长度异尾、头部≤1MiB、边长/像素限制、过短/过长流、取消均覆盖。1×1 WebP夹具另用官方完整解码确认可读。
- `task-1-mutations.json`记录4个行为变异被杀：取消候选上限、只摘要首块、只信声明图片、遗漏EOF取消。原字节hash恢复核验。
- `task-1-affected-green.log`：asset全包/archtest/config/httpserver的race通过。`task-1-lint.log`：资产相关golangci-lint 0 issues。关键词76规则/14例外无命中。
- `task-1-full-go.log/json`：实际make test完整server/tools通过，退出0；测试输入逐文件hash在task-1-tested-inputs.json，原件没有假记全项目七项已跑。
- x/image0.47的必要传递链为x/sys0.49、x/text0.43，后者要求现有x/tools0.51/x/sync0.24。go mod tidy把原本直接import的rivertype归到直接依赖，版本未变。模块图保存在task-1-dependency-graph.txt。
- validator只消费io.Reader，Close由拥有ObjectReader的app在T7验证；T1不声称证明上层Close。ETag最长1024字节且总metadata≤16KiB，超长显式拒绝不截断，P3真实provider合同继续核对。
- P1 inline类型检查复用同一RasterMediaType，既有签名读回归通过，无新类型放行。图片仅头部验证，不冒称全像素/动画安全证明。

## 4. 整分支审查

未开始；T13执行。

## 5. 与plan差异

T1确认必须的完整Go依赖传递链，补齐spec/plan；T1无ETag输入，原“ETag当摘要”变异具体化为漏算尾部字节，真正ETag版本与SHA不同职责在T7封存组合验证。每条局部裁定及代价另见台账。

## 6. 后续责任

T5开始首张资产表、删除协作与已知对象回收。P4/P5绑定、P3 S3、P6/P7页面以及P1遗留C12 Minor都未完成；输入handoff保持open。

## 7. 已知限制

当前已有基础纯规则/校验、本地有限清单/临时回收、宿主授权与锁编排、父清理单页与River持久进度；尚无资产业务路由、file_assets或资产清理worker。不能根据T1–T4通过声称F1/F3/F13/F14/F15已满足。

## 8. T2本地清单、临时目录与配置

- 实际红/绿在task-2-red.log与task-2-focused-green.log；setup编译红分开保留。10,000个真实对象以乱序创建，10页完整覆盖；最终含内部布局检查的清单读取1.027秒。目录每次读取256项、只保留最小limit+1；另外在10,000目录的3条结果页验证窗口容量≤4。不是任意规模性能承诺。
- 四个变异实际行为失败且源恢复：取消根限制、取消ctx检查、无界保留、全量枚举。年龄来自实物ModTime与调用方cutoff，严格边界/另一个root/部分进度保留已测；业务clock误接线变异需在T9真正调用者存在后完成，不冒充本Task已证明。
- 未知目录、符号链接、损坏metadata、对象内未知文件均失败且不删除；每页只读取本页metadata，非快照，后页损坏到该页失败。临时清理仅合法UUID目录中data/meta.json的已知布局，错误会携带此前已完成的游标和删除数。
- 五项生命周期配置范围、默认、溢出、未知键与日志遮盖通过；test profile继承安全默认，期限故事用显式覆盖。已有P1签名/传输与全部asset/config/configs/archtest race通过。lint首轮仅导入分组错误，修正后0 issues，原失败保留。
- 局部裁定为本页metadata读取、profile默认、对象内部未知布局及业务clock变异层级；结论、理由和代价见台账。

## 9. T3宿主授权、锁协议与父删除事实

- Access 的真实性质红/绿、三个 owner 门面的红/绿、转接的红/绿各留原件；真实 PostgreSQL 的锁/父事实测试与156项角色矩阵通过。矩阵使用 API 建立宿主，实际调用各 owner 的现有权限判断，不把尚未上线的资产 HTTP 路由说成已测。
- 实际调用次序为 Inspect → 同一项目门 → Tx → credential 时钟与 CallerLock → 排序 W(S)/P(N)/D(U) → Check → 资产 U → 审计时钟 → 包内回调。错误、取消和 panic 释放资源；源目标的语义顺序独立于物理锁排序。
- PostgreSQL 测试绑定请求 PID、SQL 与 blocker，反向 ID 输入证明先持低 ID 再等高 ID；W 锁与 SHARE 兼容、D 锁与 KEY SHARE 冲突。父事实读取不额外锁父行，原时间与删除人逐值保留，创建者不同于删除者。
- `task-3-mutations.json` 六项均命中行为失败：绕过 owner、W 逆序、D 逆序、另建项目门、失去 HiddenFrom、误用审计时钟做 credential 校验。每次恢复原文件并核 SHA256。
- 局部裁定：issue 以 `AssetProjectReader` 读取已持锁项目的归档状态，避免再次锁项目；错误接线会破坏拒绝次序，真实查询及矩阵约束它。T3 提取统一 `projectWrites` 并验证资产转接使用同一实例；实际业务模块在 T5 一次构造，HTTP 交错在 T11 证明，不以转接单测替代最终接线。
- plan 原有回调示意与锁次序笔误已按 spec 和实际符号同步：Check 在父锁后、资产锁前；`AccessHeld` 携带最终锁后的时刻；`ParentKey.Kind` 使用具名 ScopeKind。业务用例仍须完成用途、绑定、父链和 revision 复查。
- `task-3-race.log`：asset/issue/project/workspace 全包、bootstrap 与 archtest 的串行 race 回归通过，bootstrap 653.318秒；源码快照核验未变。lint 首轮发现三个适配器未被使用，真实锁/矩阵改为经过这些适配器后，`task-3-lint-final.log` 为0 issues。真实 P 锁转接、错误原样传递与最终聚焦 race 通过（bootstrap 45.388秒）；没有放宽 lint 或架构守卫。阶段验收仍不勾选。

## 10. T4父清理分页与River持久进度

- 四个owner改为每次一表一页的Page，生产中不再保留无界Purge入口；原清理图、60天边界、独立提交、迟到删除与父CASCADE测试经测试专用drain迁移，未删原断言。issue/project/workspace构造时复制明确保护标识，当前生产为空；真实资产约束及精确目录在T5接上，不把空集合当保护已交付。
- `PurgeProtected`仅识别23503与schema/table/constraint全匹配。真实PG父→子→资产及自引用证明整批失败全部回滚，逐候选得到Scanned3/Deleted1/Blocked2，受阻前后邻域继续；取消/未知FK/23001/40P01保留此前独立提交进度并返回错误。
- 普通固定SQL用有限物化候选CTE和DELETE CTE分别返回扫描数、删除数与候选最大ID，DELETE再核cutoff。实际PID/SQL/blocker交错先复现候选被并发删除时错误耗尽，再复现候选被改到精确cutoff仍遭删除；两个方向修复后通过。无跨模块SQL、额外父锁或表名拼接。
- bootstrap预算使用运行时钟，单轮业务cutoff保存在全局模块/表/After游标；worker在snooze和错误前RecordOutput，编码上限16KiB，未知版本/模块/表或损坏输出明确失败。真实River在同一job的snooze后停机，换Runner且业务快进30天仍从原After/cutoff继续，完整页前进后完成；不声称每页恰好一次或计数跨重试恰好一次。
- `task-4-mutations.json`六项变异全部被杀且恢复字节hash一致：Deleted判断耗尽、只看23503、重置cutoff、丢After、漏RecordOutput、扫描数冒充删除数。失败源码/日志分别保留，不把编译错误当性质红。
- 首次受影响race暴露旧请求探针仍等待空页DELETE；新受保护路径空页不发DELETE。夹具先阻住真实前置清理、提交过期候选，再移交issues屏障；日志确认真实DeletePurgeIssues与业务MarkChanged各自PID/SQL，原项目锁/工作区锁/未提前写断言保留。聚焦RED与GREEN原件在task-4-purge-probe-*.log。
- 最终串行race覆盖platform/postgres、四owner的app/adapter及bootstrap清理/父CASCADE回归全部通过，bootstrap52.129秒；archtest1.690秒；全server lint 0 issues。1506个测试输入hash在task-4-final-inputs.json，运行后核验未变。原首次race失败及错误cwd的lint调用保留；正式结果只取final日志。
- 局部裁定及代价：单页接口让遗漏调用者编译失败；保护标识构造复制避免后注入；有限CTE保留真实扫描成本；DELETE cutoff复核避免等待后的过期事实；真实候选夹具保持原锁证明。完整理由与错误代价在台账，阶段A1–A12仍未勾选。
