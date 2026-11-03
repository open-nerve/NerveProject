# M5 / P4 阶段评审记录

> 预检通过、尚未实施。分支codex/m5-p4，基线main5807a0d6e369c486e98e4cb00389d8b309322221；P3已全绿归档。三份spec/15任务plan和[re-38预检](../../../reviews/re-38-M5P4预检.md)为当前依据。

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

尚未实施，整审及修复待T14。不能把builder check的无重要问题写成代码无缺陷。

## 5. 实现与plan差异

尚无产品实现。执行中局部决定逐条写台账并回填本节；重大风险依原纪律先提交后上报。

## 6. 移交事项

F5–F7页面、29张预设素材及URL缓存属于P6；issue/comment/draft属于P5；整体故障/规模和输入交接P8。P3-M1双故障诊断、P1焦点Minor仍具名保留到P8。M5总体及输入handoff未关闭。

## 7. 限制与证据

P4预检原件为本worktree `.superpowers/sdd/P4-profile-images`，P3规划探针已入P3冻结归档；完整路径和命令/时间见re-38。当前未声称P4 API/数据库新字段、真实图片或后续页面已经上线。
