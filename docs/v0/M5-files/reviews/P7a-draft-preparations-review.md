# M5/P7a 阶段评审记录

> 2026-11-04：按负责人已批准A同步合同，当前仅预检，产品未实施。分支codex/m5-p7a；main基线963636757fe5c5a8c10d5781b43d1481ba6f32a2，P6准确main CI41833460144及归档/清理前置已核；预检通过，产品未实施。

## 1. 范围与方式

[主spec](../specs/P7a-draft-preparations.md)与[plan](../plans/P7a-draft-preparations.md)只交付准备件查询协议及真实HTTP恢复证明。原P7内容页面预检的原始报告为[re-46](../../../reviews/re-46-M5P7预检.md)，其架构问题已经负责人选择A，普通副本Issue/Draft结果由主控局部明确。新P7a读取协议是新的设计范围，做一次fresh builder预检；实现完成后才做一次整分支deep-review，二者不冒充。

## 2. 当前验收

A1–A6均未完成；尚无新API、产品修改、依赖安装或P7a产品测试。原P7准备时只运行既有TestDraftAssetTransferPublicationStates三个子项，通过且1707个Go输入稳定；它证明原发布合同，不能当新恢复入口验证。

## 3. 局部裁定

- R1：新私有列表将本次Host.Inspect的forbidden视作asset.not_found；旧API/global mapper保持。理由、代价和精确错误边界见spec§2，T2/T4必须验证。
- R2：复用现有Asset公开DTO与默认50/最大100分页，cursor绑定draft/列表种类；不返回签名能力，不新增无用前端service或端口集合。跨页并发变化靠显式刷新与写端重验，不提供虚假一致性保证。
- R3：只读能力从已有HostAccess/ReadTx/Store取得，消费方声明Inspect-only接口，不新增模块import、owner入口或跨模块JOIN；新schema/index没有实证需求先不加。
- R4：P7b接收恢复UI和含图复制完整编排；普通副本最终Issue，草稿副本最终Draft，目标持久化后失败/取消保留并给恢复入口；本阶段不实现页面。
- R5（独立builder核得后主控收窄）：旧列表项后来绑定时，描述图片受asset.bound保护，普通附件仍可按原权限删除；查询不预留unbound状态。理由是批准A复用既有DELETE，代价是P7b确认须明确另一标签页在途/已完成的同ID也可能被丢弃，两种用途分别回归；不扩大为条件删除协议。
- R6（主控自检）：关键词守卫只接受数字phase，P7a/P7b实施共用M5/P7，不扩大PHASE_RE；实际生成位置为api-client/schema.gen.ts，由既有export type *直接暴露，无须手写别名。E2E沿既有stories/files目录。理由是遵循实际工具/包入口，代价是文档阶段名与守卫数值阶段分开记录。

## 4. 预检结论

主控已核实际事务/行查询、DraftLocks拒绝顺序、Host适配、AssetDTO、shared cursor/PageSize。新范围一次fresh builder结论nothing important left，唯一局部冲突按R5同步；主控自检通过，P6准确main CI/归档/清理已完成。最终结论：预检通过，可在获准启动后从T1实施。完整证据见[re-47](../../../reviews/re-47-M5P7a预检.md)，不将“预检通过”写成“新端点已验证”。

## 5. 后续接收

P7b承接内容附件、编辑器、恢复UI、普通/草稿复制结果、计数显示和旧services包删除；P8/closeout承接最终F1–F18、默认部署/容量、P3-M1/P4-M1、全仓warning规则/死成员、具名旧copyMarkdownToClipboard风险与所有handoffs最终关闭。输入handoffs现仍open。
