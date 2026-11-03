# M5 / P4 阶段评审记录

> 预检通过，T1–T3字段/绑定/读取及聚焦回归完成；T4待实施。分支codex/m5-p4，基线main5807a0d6e369c486e98e4cb00389d8b309322221；P3已全绿归档。三份spec/15任务plan和[re-38预检](../../../reviews/re-38-M5P4预检.md)为当前依据。

## 1. 评审方式

一次fresh builder check无重要未决项；主控检查spec覆盖、类型/路径、五项Review Focus、步骤可执行性和篇幅。实施后T14按deep-review逻辑→集成→测试，另做一次fresh整分支终审；现在未进行产品终审。

## 2. 完成标准

主spec A1–A9均未勾选。生成/lint/既有迁移/架构基线通过，只证明准备环境与已有守卫，不替代P4实际功能、并发、清理、七项门禁或CI。

## 3. 规划裁定与代价

- owner事务内Lock→At→owner写→Apply，asset不接owner授权或任意闭包；代价为重读同一已锁集合，换取新项目一次绑定及统一时刻。
- 全部图片响应锁内CPU签发，账号首锁、目标资料账号不逆序锁；代价为短事务持锁，完整集合容量需T12验证。
- 有效邀请链接只签它固定workspace的logo；已签bearer期限不变，不扩大任意UUID读取。
- 四条反向FK即时NO ACTION且精确目录验证，三个owner先条件解除，再由asset删除；有限页增加少量接口协作，遗漏路径必须失败。
- builtin素材和前端控件P6，organization_size仅收紧已有响应枚举；不引入未经设计点名的依赖。

## 4. 整分支发现与处理

T1–T3已实施，整审及修复待T14。不能把builder check的无重要问题写成代码无缺陷。

## 5. 实现与plan差异

T1按计划实施。T2接口核对发现SaveAsset有意不写scope/bound_at，改为资产内部专用BindImage，保留恢复写的范围和Revision保护；不扩owner接口或放宽架构守卫。新增小端口的代价是多一个固定SQL路径，具名真实数据库反例必须覆盖。

## 6. 移交事项

F5–F7页面、29张预设素材及URL缓存属于P6；issue/comment/draft属于P5；整体故障/规模和输入交接P8。P3-M1双故障诊断、P1焦点Minor仍具名保留到P8。M5总体及输入handoff未关闭。

## 7. 限制与证据

P4预检原件为本worktree `.superpowers/sdd/P4-profile-images`，P3规划探针已入P3冻结归档；完整路径和命令/时间见re-38。当前字段与内部绑定已在分支实现，未声称P4公开API、真实图片或后续页面已经上线。


## 8. T1所属字段、纯规则与SQLC

三owner迁移00035–00037加入四个nullable UUID和cover_builtin_id；即时非deferred NO ACTION、无条件索引、精确cover-01..29及互斥CHECK。纯domain补三态Set/可空值和封面选择规则，sqlc查询/行映射保留所有现有方法签名；project INSERT携带最终封面，仍未接公开绑定API。

真实RED：旧库缺全部五列、合法cover-01失败；领域错误ID/双来源未拒绝、选择一侧未清另一侧；三个仓储遗漏读写字段。GREEN覆盖P3存量行升级不变/默认null、四FK实际23503、全部迁移Up/Down/Up、三态字段/列表/返回映射和同事务回滚。首轮全量迁移因第二次Up预期仍34失败，同步为37后通过，未放宽目录核对。

最终八包无缓存全绿（34.114s）：migrations3.574s、archtest1.696s、identity/workspace/project domain0.233/0.177/0.170s、三个postgres包6.295/13.508/5.810s；具名新测试实际运行。Go lint server/tools均0 issues（28.890s）。make gen已生成且重生成输入不变；首次gen-check因生成文件尚未提交退出2，按Makefile原规则在提交后复核，不放宽守卫。原件task-1-schema-red/cover-red/fields-red/final-green及receipt/inputs、task-1-generation、task-1-gen-check、task-1-lint均在阶段workspace。

P4整体A1–A9暂未关闭。反向清理/完整程序目录T4、公开API T5–T9及全部门禁T13仍待实施，T1不能当作阶段完整交付。

## 9. T2原子图片绑定

新增小端口ImageBindings和专用BindImage SQL，owner持有事务和父锁时一次升序锁齐全部旧新资产，统一取得At，写所属引用后再Apply。新候选先全批隐藏判定，再用途/上传/绑定状态；当前引用幂等不写，管理员可替换旧申请者的已绑定图，旧软删审计保持。new_project仅在最终绑定SQL迁入已插入的项目，SaveAsset仍不改申请者/父引用/bound_at。

应用行为RED 1.541s、真实PG RED 3.344s；SQLC nullable At首轮编译不符单独保留，不算行为证据。GREEN真实触发器在第二件绑定时失败，四件资产与用户两字段全部回滚；无事务调用拒绝、Revision/用途/状态拒绝、项目先插入再绑定、SaveAsset边界均验证。最终11项具名主测试、两包带race全通过（8.087s），完整asset/app race亦通过（16.907s，含既有local/S3合同）。lint静态布尔形式提示修正后server/tools均0 issues（17.024s），不改变规则。

四项实际行为变异（申请者、完整范围、7天、升序锁）均被具名断言检出，每次精确SHA恢复，证据task-2-mutations.json及四份日志。最终4249个源输入hash稳定；生成物提交后按原gen-check复核。证据在阶段workspace的task-2-*，公开owner接线和真实内容仍归T5–T11。

## 10. T3批量图片读取

资产端只消费owner给出的四类引用：先拒绝同ID不同宿主/用途，再去重升序按最多1000个ID取锁；全部块查询完成后才CPU签发。准备/失效/不匹配引用返回缺项，真正数据库/解码/签名错误原样返回。共享P2 contentExpiry，零UPDATE且不依赖业务时钟，不接对象I/O或owner回调。

行为RED 1.228s；8项具名主测试（6个新读取测试及2个既有内容测试）带race GREEN 2.519s。四用途、缺项/原错、1000/1001分块、重复去重、后一块失败零签名和读取审计不变通过。bound/完整范围/同ID冲突三项变异都实际断言失败，恢复SHA一致；4252源输入核对稳定。这里的Signer为编排替身，真实local/S3内容仍由T11验收，未提前宣称实物成功。原plan的ResolveContent测试名修正为实际AssetContent，避免漏跑原期限验证。
