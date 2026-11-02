# M5 / P2 阶段评审记录

> 实施中：T1–T3完成，T4–T14待执行；没有进行整分支终审、没有合并。基线main85dc4087，预检提交72a3e58b，见[re-34](../../../reviews/re-34-M5P2预检.md)。原件在本工作树 `.superpowers/sdd/P2-assets`。

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

T4开始父清理前进与持久任务游标。P4/P5绑定、P3 S3、P6/P7页面以及P1遗留C12 Minor都未完成；输入handoff保持open。

## 7. 已知限制

当前已有基础纯规则/校验、本地有限清单/临时回收、宿主授权与锁编排；尚无资产业务路由、file_assets或新清理worker。不能根据T1–T3通过声称F1/F3/F13/F14/F15已满足。

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
