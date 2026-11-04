# M5/P7a 草稿准备件恢复协议预检

> 2026-11-04：预检通过；P6准确main CI/归档/清理前置已核，当前不实施。分支codex/m5-p7a，正式main基线963636757fe5c5a8c10d5781b43d1481ba6f32a2。

## 1. 范围与批准依据

负责人已选[方案A](../v0/M5-files/reviews/P7-draft-preparation-recovery-decision.md)：补作者私有准备件发现入口，保留原发布完整性，分P7a协议/P7b页面。本次同步M5设计、P5接收合同、[P7a spec](../v0/M5-files/specs/P7a-draft-preparations.md)/[plan](../v0/M5-files/plans/P7a-draft-preparations.md)和P7b延后草案；暂不增加API或实现页面。

原P7预检[re-46](re-46-M5P7预检.md)的两问题分别由批准A和普通副本最终Issue/草稿副本最终Draft的局部编排闭环。原报告和88d5ae30上报提交保留，不改成“当时通过”。此次fresh builder只看新的P7a读取协议，不重复原前端范围；实现后整分支deep-review仍未开始。

## 2. 真实入口核对

| 约束 | 源码依据与结论 |
|---|---|
| 作者/工作区/可空项目 | bootstrap/asset_access.go→issue/app/asset_access.go→DraftLocks.read；已有Inspect可复用，无第二授权端口。guest可能先403，仅新列表按R1归一404 |
| 同一只读快照 | platform/postgres/tx.go的repeatable-read/read-only和ctx传播；asset/issue Store均消费postgres.DB(ctx)，已有asset_read_snapshot_test tracer可暂停真实HTTP |
| 未知ID确实存在 | prepare先提交再返回；duplicate先持久pending再封存、失败仅返error。新列表为唯一客户端发现入口，测试不能使用SQL/存储夹具key中的ID代替发现 |
| 精确查询与游标 | attachments.sql的null-safe父链/UUID升序模式、file_assets draft索引、shared cursor规范回编码/PageSize50/100，可复用；跨页无共享快照，不提供发布保证 |
| DTO与生成入口 | assetDTO只输出公开字段；schema.gen.ts经make gen产生，api-client已有export type *；无签名能力或手写复制DTO |
| 原删除及发布合同 | delete.go只拒绝已绑定非普通附件，draft_asset_rules仍全集合pending拒发布；查询不改写锁、事件或清理 |

## 3. 一次fresh builder与局部修正

独立审者`/root/m5_p7a_builder_check`只读源码/文档，结论 **nothing important left**。唯一实质问题：初稿将“列表看到unbound但操作时已bound”一律写为DELETE拒绝；真实普通附件仍可按权限删除。主控按已批准A保留旧写协议，收窄为描述图asset.bound、普通附件仍可删，并同步spec§3/5、planT4、P7b确认文案/测试。无需扩大为条件删除或再次请求架构决定。

主控另修正实际生成文件位置、现有export type *、stories/files目录，以及P7a/P7b实施共用守卫支持的数字phase=M5/P7；不改守卫正则、不延期例外。所有R1–R6的结论/理由/代价见[阶段记录](../v0/M5-files/reviews/P7a-draft-preparations-review.md#3-局部裁定)。原始独立消息保存在`.superpowers/sdd/P7a-draft-preparations/builder-questions.md`。

## 4. 计划覆盖与验证边界

七Task：T1游标/SQL/app，T2API/生成/接线，T3未知ID真实恢复，T4快照/权限/交错，T5七门禁，T6一次整审及最终分支CI，T7集成归档/P7b正式预检。A1→T1/T2、A2→T2/T4、A3→T3、A4→T1/T4、A5→T5/T6、A6→T7；五Review Focus均有区分度反例。

主控静态检查脚本`check-preflight.py`核main基线/产品零变化/无依赖安装/七Task/21个开放实施Step/六个开放产品验收/文档本地链接/格式；完整输入hash及结果保存在同目录`preflight-product-inputs.json`和`preflight-receipt.json`。本阶段不新增测试来复刻文档，不跑完整产品门禁；原P7准备复验的既有三子项仅证明旧发布规则，新HTTP/并发/权限全部仍待实施。

## 5. 最终前置

P6 noff main96363675的CI41833460144 attempt1三job全绿，381 E2E/14.4m；分支8e46ab55的CI41828756305也独立全绿。6,861文件归档逐hash/bytes/mode/membership核验、unknown0，正常清理分支/worktree后再次核验通过；[准确证据与归档入口](../v0/M5-files/reviews/P6-profile-upload-review.md#22-t15准确main全绿与归档清理)。正式预检基线不再悬空。

P7a预检通过，七Task/六产品验收均待实施。P7b仅延后草案，正式基线/预检等P7a集成。此次没有架构待决问题，停在用户指定的“只完成预检、暂不实施”。
