# M5/P7 阶段评审记录

> 2026-11-04：预检Needs Decision，未实施。main基线963636757fe5c5a8c10d5781b43d1481ba6f32a2，产品树与P6候选8e46ab55一致，main CI/归档仍待核。原始准备台账：`.superpowers/sdd/P7-content-assets/progress.md`。

## 1. 评审方式

按[主spec](../specs/P7-content-assets.md)、[内容生命周期](../specs/P7-content-lifecycle.md)、[编辑器合同](../specs/P7-editor-images.md)和[plan](../plans/P7-content-assets.md)做实际接口/调用链预检；一次fresh builder只找影响实施的未决问题。不实施产品，不安装依赖，不把P5/P6既有测试称作P7验证。阶段完成后另做一次整分支deep-review，两个scope不同。

## 2. 验收标准

A1–A8全部待实施。后端已有API/存储证据为前置，P7真实页面、计数显示、复制/撤销/失效与旧链删除尚未交付。

## 3. 准备中的局部裁定

主spec§3记录八项结论、理由与代价：唯一上传算法/用途策略、普通新建无宿主不上传、无项目草稿、只接图片、不改空评论合同、正文删除不删资产、有限主动下载、可选图片端口及未来领域边界。仍以M5设计和已批准PC1/PC2/PC3为准；新的真实架构冲突先保存现场上报。

## 4. 预检发现及处理

一次fresh builder已完成，发现2项：草稿未知ID准备件无法恢复为架构待决；普通工作项含图复制最终结果/取消恢复为局部编排。主控核对后详见[re-46](../../../reviews/re-46-M5P7预检.md)和[决策记录](P7-draft-preparation-recovery-decision.md)。本轮预检结论Needs Decision，未实施。

## 5. 实现与计划差异

尚未实施。准备阶段只读实际协议与消费者、Plane对照；没有执行整段原型重放。

## 6. 移交事项

P8：最终F1–F18、默认部署演示/容量与清理、P3-M1错误cause维护、P4-M1未用字段、旧copyMarkdownToClipboard具名风险、全仓死成员和至少一个新warning规则清零、全部handoff最终闭环。P7新增控件使用中性图标，P8仍核其他旧图标消费者。未新增投递任务CLI，原M5-04全文保持。

## 7. 已知限制

无new_issue准备范围；未保存工作项先创建或明确保存草稿后重开。签发S3能力到期前可能仍可使用，UI撤权不等于存储即时撤销。未知复制结果可能留下未绑定副本，用户明确重试前有提示，服务端原GC负责回收。下载内存受最多100MiB资产限制，P7验证取消/上限，P8做容量演练。
