# M5/P7 拆分前预检与接续记录

> 2026-11-04：原预检Needs Decision已由负责人选择A关闭为“已裁定并拆分”，未实施。P7a协议的当前记录见[P7a review](P7a-draft-preparations-review.md)，P7b为延后草案。参考main963636757fe5c5a8c10d5781b43d1481ba6f32a2，原始准备台账`.superpowers/sdd/P7-content-assets/progress.md`随改名后的codex/m5-p7a保留。

## 1. 评审方式

原P7按实际接口/调用链做一次fresh builder预检，只找影响实施的未决问题。文档现迁到[P7b主spec](../specs/P7b-content-assets.md)、[内容生命周期](../specs/P7b-content-lifecycle.md)、[编辑器合同](../specs/P7b-editor-images.md)和[plan](../plans/P7b-content-assets.md)，接收恢复/副本编排后仍是延后草案。不实施产品、不安装依赖、不把P5/P6既有测试称作P7验证。P7a新协议另做一次范围明确的builder；各阶段实施后才整分支deep-review。

## 2. 验收标准

A1–A8全部待实施。后端已有API/存储证据为前置，P7真实页面、计数显示、复制/撤销/失效与旧链删除尚未交付。

## 3. 准备中的局部裁定

P7b主spec§3记录九项结论、理由与代价：原八项加恢复/普通与草稿副本最终结果。普通副本仍Issue、草稿副本仍Draft；明确提交才持久化，失败/取消保留同一目标并给恢复入口。仍以M5设计和已批准PC1/PC2/PC3为准；新的真实架构冲突先保存现场上报。

## 4. 预检发现及处理

一次fresh builder发现2项：草稿未知ID准备件恢复为架构问题，已由负责人选择A；普通工作项含图复制最终结果/取消恢复为局部编排，已在P7b§3.9/编辑器§4/T10明确。历史原报告及主控闭环见[re-46](../../../reviews/re-46-M5P7预检.md)和[决策记录](P7-draft-preparation-recovery-decision.md)。拆分后P7a先交协议，P7b正式预检等P7a最终main；本记录不把拆分前预检改写成产品已通过。

## 5. 实现与计划差异

尚未实施。准备阶段只读实际协议与消费者、Plane对照；没有执行整段原型重放。

## 6. 移交事项

P8：最终F1–F18、默认部署演示/容量与清理、P3-M1错误cause维护、P4-M1未用字段、旧copyMarkdownToClipboard具名风险、全仓死成员和至少一个新warning规则清零、全部handoff最终闭环。P7新增控件使用中性图标，P8仍核其他旧图标消费者。未新增投递任务CLI，原M5-04全文保持。

## 7. 已知限制

无new_issue准备范围；未保存工作项先创建或明确保存草稿后重开。签发S3能力到期前可能仍可使用，UI撤权不等于存储即时撤销。未知复制结果可能留下未绑定副本，草稿须提供新查询和明确丢弃恢复路径，不能只等原GC。下载内存受最多100MiB资产限制，P7b验证取消/上限，P8做容量演练。
