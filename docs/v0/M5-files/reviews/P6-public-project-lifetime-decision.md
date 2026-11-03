# P6 预检：公开项目非成员封面的读取生命期

> PC1 待负责人决策。分支 `codex/m5-p6`，基线 `c59cedf337c5c5654bef2f0cbb33407ff04ced26`。仅提交预检材料，没有修改产品；P6 未预检通过、未实现。P5 已独立 noff 集成，main CI/归档继续，不依赖此决策。

## 已确认事实

项目 API 允许工作区成员读取未加入的公开项目，返回 member_role=null；封面 URL 属于该合法项目 DTO。UserPermissionStore 对此返回 undefined，符合原工作项权限合同，不应改成 member 角色。

M4 的 PermissionReads 只在有效项目角色存在时设置项目签名。对公开项目非成员，签名在“可见→完整目录缺席→新事实恢复”三个状态中都是 undefined，因此不推进代次、不调用 IssueRootStore.registerResource 的 invalidate。实际源码 probe 在相同 observable 输入中得到原 token 三次 current=true、回调0次；member_role=15 控制组缺席和恢复均 current=false、回调1次。源码及其 SHA、完整结果归档；这是现有观察器的单元行为，不冒称浏览器图片已经泄漏或 P6 新组件有产品回归。

P6 草案要求图片旧生命期在可见性丢失后永久终止，同时仅沿既有 IssueRootStore 失效入口。不能靠渲染时再查当前存在解决 ABA：恢复后旧 token 又会通过当前存在检查。也不能让项目非成员获得工作项角色、只在图片组件额外观察一份权限、或恢复被 M4 R06 排除的旧详情。

## 建议方案 A

在既有 PermissionReads 的项目失效事实中补齐“当前项目可见，但无工作项角色”的状态。可见性仍由 ProjectStore 规范 getter 给出，按工作区匹配；只存可比较签名和水位，不缓存实体。IssueRootStore 保持唯一所有者和订阅入口，UserPermissionStore、服务端授权、ProjectStore R06 的事实选择保持原合同。有效角色仍单独判定，签名的存在只决定旧读是否过时，绝不能作工作项可读/可写证明。

先同步 M4 权限水位说明、M5 前端生命周期说明及 P6 spec/plan；T2 增加原观察器/真实 IssueRootStore 联合测试，T3 验证项目图片 ABA。完整回归包括：公开非成员的缺席/恢复/迟到响应，成员及 guest 现有角色变化、权限恢复；无变化目录/无关工作区项目不误失效；角色仍undefined且受限issue读写仍拒绝；ProjectStore全部R06正反交错；root退休释放。T12行为变异须移除非成员可见事实后被真实断言拦截。

收益：复用当前事实与唯一生命周期所有者，补足新资料消费者实际需要的失效覆盖；没有第二份权限缓存或新观察器。

风险与代价：扩展的是 M4 共享项目代次合同，可能让覆盖该项目的旧 issue/profile 读取额外失效；必须以范围隔离和完整既有回归证明不会影响无关项目，也不能把失效元数据误当授权。单纯新增一个图片组件的测试不足以验收。

## 可行替代 B

P6 暂时对公开项目非成员仅显示中性/预设封面，延后其自定义能力URL展示。这样不用扩展既有项目代次合同，但改变当前合法项目DTO的展示一致性，偏离原封面恢复范围；应明确调整F7与后续归属，不能静默降级或宣称P6原验收全部完成。

## 不采用的绕路

不新增Asset权限观察器，不把Project.member_role=null伪装成成员，不给已失效旧请求发新token，不只靠一次render中的存在判断。把所有read lifetime抽成新全局框架则扩大架构改造，当前没有必要。

## 需要决策的边界

这是对 M4 共享失效事实范围的扩展，影响所有订阅者；按负责人“跨模块协同调整先提交再上报”的纪律，在正式版本化材料后请负责人决定 A 或 B。P5预算及集成是独立工作；P6产品实现保持未开始。

## 真实 store 联合复核补证

主控随后在仓库现有 Vitest/FakeNerve 上调用实际 RootStore、ProjectStore、UserPermissionStore、IssueRootStore，以完整目录应答驱动“可见→缺席→恢复”。工作区角色是15，公开项目非成员null的旧 token 在缺席/恢复仍 true，注册资源 invalidate 0次；member_role=15 控制组均 false、invalidate 1次。按新 P6 合同断言得到1 failed/1 passed，三个失败均是预期生命期断言，而不是编译错误。非成员 assertProjectReadable 仍拒绝，证明不能用修改角色来修复此事。

p6-real-permission-probe-actual.log / -receipt.json 保存1.598秒行为复现，4434非文档输入前后不变；脚本/配置和两份结果JSON同时保留。最初独立配置误选Vitest CommonJS入口导致0 tests，原日志保留于p6-real-permission-probe.log，不算业务RED；改为既有ESM入口才得到上述有效复现。没有安装依赖、没有新增产品代码或容器，也没有修改P5已验证输入。

## 固定证据与影响边界

| 实际 store 输入 | 可见时旧 token | 完整目录缺席后 | 新事实恢复后 | invalidate 次数 |
| --- | --- | --- | --- | --- |
| 工作区 member，公开项目 member_role=null | true | true（新合同要求 false） | true（新合同要求 false） | 0（要求 1） |
| 同输入，项目 member_role=15 控制组 | true | false | false | 1 |

原件位于本 worktree `.superpowers/sdd/P6-profile-upload/precheck/`；其中 `p6-precheck-probe-evidence.json` 固定十份证据的 SHA/bytes，`source-copy-manifest.json` 核验从 P5 现场复制的全部预检材料。P5 原始归档另保留未经改写的报告和首次配置错误。实际 store 探针源码 SHA-256 为 `1dda80f7c29cb4742cf508b06105f2f4455c5f3b80795a0811d52da312ac0c57`，有效行为失败日志为 `73a5d3e3243934e18a48bf3839c710335a3da873cd60d2cd018c36933b940ecd`。

方案 A 只修改 `core/store/issue/permission-reads.ts` 的事实签名及其测试与 P6 接入；不修改 UserPermissionStore 的授权角色、不修改 ProjectStore R06 的规范来源、不修改后端项目可见规则。不是把“可见项目”升级为“可读工作项”。先修订 [M4 §3.22 权限读取水位](../../M4-issue-core/M4-design.md)、[M5 §9](../M5-design.md#9-前端与视觉)、主 spec、接口合同和 plan，再开始产品实现。
