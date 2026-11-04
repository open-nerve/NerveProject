# M5/P6 PC3：图片重新授权与既有目录读取冲突

日期：2026-11-04。状态：负责人已批准A，合同先行提交5c370f01；PC3已修复并通过联合回归/行为变异/三浏览器，T13完整门禁收尾中；原停点记录保留。

## 1. 已完成工作与停点

PC2 的方案 A 已完成：WorkspaceRootStore 的迟到写回与重放复用 IssueRootStore 的单调权限代次，基础及实际页面联合回归见[阶段记录 §10–11](P6-profile-upload-review.md#10-t6-pc2-基础修复与联合回归)。本次不是重开 PC2。

T1–T12 已提交到 codex/m5-p6，最近的 T12 提交 f720027dac950a3e759c5bda090cf58119d107ee。T13 的 gen-check、lint-go、完整 Go 测试通过；lint-web 的关键词检查发现本问题，七门禁尚未全部通过。A3重新开放，T14整分支终审及T15集成尚未进行。main 保持 f5f4ce9431e7fb360363bf169da33487cf9b864f。

## 2. 事实与根因

M3的工作区目录只有一个读取入口 `useWorkspacesFetch`，使用带 loginId 的 `WORKSPACES` 键；`workspaces-list-fetch` 守卫明确禁止其他组件或模块自行调用 `fetchWorkspaces`。见[M3设计§7.1/§7.10](../../M3-workspace-project/M3-design.md)、[M3/P9合同](../../M3-workspace-project/specs/P9-web-workspace-pages.md)。

P6的 `useProfileImage` 使用另一个 `profile-image` 键，经 ProfileImages / ImageReadGroup / ImageSource 再调用 WorkspaceRootStore.fetchWorkspaces。图片之间会合并，但它不与既有页面目录请求合并。Reconciled按请求代次淘汰旧读取；当页面读取比图片读取更新时，旧图片读得到undefined，即便两个HTTP回答都成功，也被ImageReadGroup视作无法获得授权证明，最终停在错误占位。

这是P6新增读取接线造成的架构回归，不是关键词误报。只增加守卫豁免、换方法名、把undefined当成功、失败后自动循环重试，都没有修复两个读取来源和证明脱节的根因，还可能破坏失权/ABA边界。

## 3. 可复现证据

探针使用实际 RootStore、ApiClient、Reconciled、SWR、useWorkspacesFetch、useProfileImage 和 DOM；只用仓库既有 fake-nerve 控制网络回复次序。未改生产代码。两个性质均得到行为断言失败：

1. 同时挂载页面目录与工作区图标：实际记录两条 `GET /api/v0/workspaces`，期望一条。
2. 先发图标读，再挂载页面目录；先交付新的目录回答，再交付旧图片回答，均为200：规范store仍含合法logo，图片URL却为null，错误为 `Image owner read was superseded or unavailable`，DOM没有img。

原件在本worktree的 `.superpowers/sdd/P6-profile-upload/`：`t13-pc3-probe.tsx`、`t13-pc3-probe-second.log`、对应 `-inputs.json` 和 `-receipt.json`。执行为临时放入 `web/apps/web/core/hooks/m5-p6-directory-probe.test.tsx` 后运行 `pnpm --filter web exec vitest run core/hooks/m5-p6-directory-probe.test.tsx`；结束后已移回证据目录，未把已知失败探针留进默认套件。第二轮2/2均为AssertionError、exit1，4526输入前后hash不变。第一轮漏设fake timers导致夹具错误，原源码和日志也保留，不算行为证据。

现有 `profile-images.test.ts` 的“superseded owner fetch”只断言安全占位和手动重试，未覆盖与页面目录读的正常协作；这解释了聚焦及浏览器正常路径通过而完整守卫仍发现回归。当前没有证据表明本问题泄露越权图片；已证明的影响是重复请求和成功后图标仍无法显示。

## 4. 方案 A：接回既有宿主读取入口（推荐）

保留 M3 的唯一工作区目录读取入口和同一个会话键，把工作区图片的重新授权接回该入口；图片层只消费当前范围的读取完成证明和规范宿主数据，不再直接发目录请求。不能把SWR缓存中的旧DTO直接当作新授权。

实施前先同步M3/M5设计与P6 spec/plan，明确小接口由图片消费侧声明、读取适配放在既有宿主读取边界；数据仍归WorkspaceRootStore/Reconciled，权限代次仍归IssueRootStore，不新建实体缓存或权限观察器。接口必须能区分“同一有效代次中已合并的在途读”“需要重新授权的新读”“已退休或被更新读替代的旧完成”。组件卸载不能取消仍被页面或其他图标使用的公共读取。

工作区先以本次两条探针RED→GREEN；同时核查self/project/member的读取与原页面是否有同一根因，纳入联合回归，不能仅靠改工作区方法名让守卫通过。若其中需要不同的上游架构改造，再提交证据上报，不在本次授权下无限扩张。

收益：保留现有页面读取模式和守卫，收敛实际重复请求及授权证明脱节。代价：需要给现有读取入口补足明确的完成/代次协作协议，测试必须覆盖挂载次序、共享读取拒绝、失权恢复与旧响应；不能只复用一个“当前有数据”的布尔值。暂时缺少有效证明时继续安全占位，不能为立即显示而接受旧读取。

## 5. 方案 B：把读取协调职责下沉到宿主 store

统一由WorkspaceRootStore拥有目录请求合并及读取完成协议，页面和图片均经这一小接口；相应修订M3的SWR职责、调用者和守卫，使守卫检查唯一宿主协调入口。不能仅添加一个转调fetchWorkspaces的别名。

收益：非React消费者也能共享同一请求；代价：改变既有M3读取职责和守卫语义，回归面更大，且需要说明SWR和store如何各自管理请求状态。当前问题有保持既有边界的方案A，暂不推荐此扩展。

## 6. 批准后的完成条件

- 同一会话的页面与多个工作区图标共用目录读取；两种挂载/回复次序、旧读取被新成功读取替代后，均不会把有效新图留在错误占位。
- 当前失败仍可见且有界重试；权限移除/恢复、同slug换UUID、账号/root退休、旧成功和旧拒绝不会复活图片；PC1/PC2继续通过，IssueRootStore仍唯一权限观察器。
- self/project/member与宿主页面读取的联合用例明确；不新增未实现接口，不增加架构守卫豁免。
- T13剩余门禁及准确分支CI通过，再按原计划做一次整分支deep-review、修复、集成与归档。未达到这些条件前不合并。

## 7. 负责人已批准A与实施界面

负责人已选择“A：复用既有读取入口，修复并继续”。本节取代页首的待决定状态，原发现和两方案保留为历史证据。

源码核查确认个人CURRENT_USER、项目PROJECT、工作区成员WORKSPACE_MEMBERS也有原页面读取，图片另用profile-image；项目搜索、编辑器mention等还会从宿主store发起合法读取。因此采用同一读取完成协议覆盖四个宿主，不仅删除工作区的一行调用。host store只发布实际read开始/完成，SWR继续协调页面请求，区别于方案B的store合并请求。

ObserveProfileRead由图片消费侧声明，RootStore组合接线；各host对结果不翻译、错误不吞掉，通用Reconciled不改。新低层self/project读取hook从原页面提取，不取项目标签/状态/成员等附加资源。工作区读取仍只有useWorkspacesFetch，原关键词守卫不改。协议细节及代价见[P6生命周期补充](../specs/P6-image-lifecycle.md#pc3共享宿主读取与完成证明负责人批准a)。

局部决定及代价：四个宿主在同一协议下发布在途完成，增加少量读取边界接线；换取页面和图片不再互相淘汰授权证明。canonical hook每次新挂载仍按现有页面语义重新授权，0窗口只取消完成后的时间缓存去重，不取消在途合并。短暂重新授权期间允许安全占位，不增加后端API、实体缓存、依赖或权限观察器。
