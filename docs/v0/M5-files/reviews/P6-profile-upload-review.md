# M5/P6 资料图片上传与展示阶段记录

> 分支 `codex/m5-p6`，实施基线 `main@f5f4ce9431e7fb360363bf169da33487cf9b864f`。P5 已完整收尾，P6 预检提交 ab9f86b8；当前 T1–T2 完成，未关闭 A1–A8。按连续授权推进，一次整分支 deep-review 后集成；重大风险先提交上报。

## 1. 输入与范围

[主 spec](../specs/P6-profile-upload.md)、[生命周期合同](../specs/P6-image-lifecycle.md)、[49直接/7透传来源清单](../specs/P6-image-consumers.md)、[15任务 plan](../plans/P6-profile-upload.md)、[预检](../../../reviews/re-42-M5P6预检.md#7-最终基线与执行前预检通过)。PC1方案A已批准；保留原授权角色、ProjectStore R06及唯一IssueRootStore观察所有者。

## 2. 执行与证据

原件在本工作树 `.superpowers/sdd/P6-profile-upload/`；progress.md记录逐Task起点与完成、precheck/保留预检/首次失败及P5收据，run-verified.py保存每次命令全日志/exit/耗时及全量非文档输入hash。产品变化不以预检旧绿色收据代替。

T1：JSON资产prepare/complete用生成DTO及当前ApiClient；raw PUT保持原File和签名URL，拒绝协议相对/带userinfo/fragment/控制字符及重复或额外头。首个终态决定取消/15分钟超时，释放监听、定时器和响应体，不把存储错误内容带入应用错误。初始骨架46 failed/1 passed，最小实现及两个测试数据形状修正后47/47；类型检查中的表驱动union推断错误已修正，原日志保留。修改四文件oxlint零warning；完整前端第一次回归发现路由扫描把协议相对检测字面量当导航地址，改为逐字符判定并保留守卫。最终make test-web 18/18通过（17缓存、web3214项实际运行），类型与四文件格式复核通过；收据接续台账，不将首次失败删除。真实浏览器Cookie隔离和页面上传留T11/T12。

## 3. 局部裁定与代价

- 跨任务接口按预检§7的三项决定执行：T3能力/T7预设/T9组合分段交付；T6共享logo props当段同步调用方；工作区能力刷新不重置未保存资料。各项正反行为由对应Task验收。
- raw PUT用有限时长独立fetch，阶段进度不伪造字节百分比；代价是无法显示线性上传百分比。前端终止不能撤回服务端已接收写，宿主绑定仍归页面。
- 原件保留至准确main全绿和归档核验，覆盖通用skill的删除scratch建议；代价是临时证据空间。仅一次fresh整审，全部局部发现按负责人要求修复，不默认延期Minor。

## 4. 尚未完成

T3–T15及A1–A8继续；没有声称T1单元测试等于真实页面、完整P6或M5完成。

## 5. T2 上传会话与统一权限失效

实际RootStore驱动PC1的公开非成员缺席/恢复，RED的旧token仍true和invalidate0修正为永久false/一次通知；工作项角色仍undefined，受限读取仍拒绝，成员/guest和R06保留。新增RootStore.asset经IssueRootStore注册范围生命期，用户不受工作区/项目变动影响，工作区不受其他项目影响，普通工作项删除也不取消项目封面。项目写仍核管理角色与归档状态；没有增加平行权限观察器。

上传按选择代次和目标生命期检查所有await与可观察状态反馈，取消/替换/卸载/root退休释放objectURL、控制器和注册项，无cleanup DELETE。prepare失败重新准备，PUT不确定或complete失败只确认旧ID，object_missing明确新申请；非法文件/未知限额拒绝。真实ApiClient的onResponse延迟覆盖响应已经到达但交付晚于取消/新选择，补充状态观察者重入及归档后busy收尾的RED/GREEN。

两项行为变异分别取消显式cancel和删除选择代次检查，均由真实断言拦截；源码逐SHA恢复。完整make test-web 18/18通过（17缓存，web实际运行），163项受影响回归接task-done收据；类型、格式和11个修改文件oxlint零warning。第一次types发现项目ES版本没有Promise.withResolvers、联合目标拷贝丢失关联，分别使用普通Promise停车和保留联合类型的分步副本修正；不改目标平台或放宽类型。原失败和变异日志全部保留。

新增只读current/recovery明确UI重试入口，绑定后同时核readyAsset身份；选错新文件时旧选择被释放，需重新选择。路由的真正effect cleanup与宿主绑定反馈在后续页面任务联合验收，T2不冒称页面已接通。
