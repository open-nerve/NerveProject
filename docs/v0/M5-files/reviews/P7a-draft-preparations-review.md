# M5/P7a 阶段评审记录

> 2026-11-04：按负责人已批准A同步合同并通过预检；现已获准实施，T1 已完成，T2 进行中。分支codex/m5-p7a；main基线963636757fe5c5a8c10d5781b43d1481ba6f32a2，P6准确main CI41833460144及归档/清理前置已核；预检通过，实施证据见下。

## 1. 范围与方式

[主spec](../specs/P7a-draft-preparations.md)与[plan](../plans/P7a-draft-preparations.md)只交付准备件查询协议及真实HTTP恢复证明。原P7内容页面预检的原始报告为[re-46](../../../reviews/re-46-M5P7预检.md)，其架构问题已经负责人选择A，普通副本Issue/Draft结果由主控局部明确。新P7a读取协议是新的设计范围，做一次fresh builder预检；实现完成后才做一次整分支deep-review，二者不冒充。

## 2. 当前验收

A1–A6待各任务闭环；原预检时无新API、产品修改、依赖安装或P7a产品测试。原P7准备时只运行既有TestDraftAssetTransferPublicationStates三个子项，通过且1707个Go输入稳定；它证明原发布合同，不能当新恢复入口验证。

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

## 6. T1 实施记录

- R7：准备件同时要求申请归属 `user_id` 与创建者 `created_by_id` 为当前作者；沿用既有私有资产所有权，不把审计字段独自当授权。代价是异常或迁移后两者不一致的记录不会暴露，仓储测试分别验证两种不一致。宿主返回的 scope/draft 必须与请求一致，错配返回既有 scope_changed，拒绝继续行查询。
- 已有游标、应用用例及实际 PostgreSQL 仓储行为 RED/GREEN；原始证据 `.superpowers/sdd/P7a-draft-preparations/`。最初缺声明/fixture 字节参数导致的编译失败不算 RED；有效 RED 为 `t1-domain-app-red-second` 与 `t1-store-red`。
- `t1-host-identity-red` 与 `t1-snapshot-mutation` 均实际断言失败并精确 SHA 恢复。首次作者变异文本匹配未命中，未运行测试，不计有效变异；重做记录另存。

- T1 提交 `bb1313a2`；关闭任务的完整三包 `go test -p 1 -count=1` 通过（37.789 秒），4538 个非文档输入前后相同，无测试缓存。`t1-full-green.json`/raw 保存；移除作者过滤使真实仓储返回多余行而失败（`t1-actor-mutation-v2`），源 SQL 与 sqlc 生成物 SHA 精确恢复。

## 7. T2 实施记录（进行中）

- 新路径通过模块 YAML 和总入口 `api/openapi.yaml` 引入；生成 Go/TS/汇总契约。R8：plan 文件清单未列总入口与权限矩阵，将其按既有契约流程纳入；不改守卫或豁免。代价是新增操作必须持续保持矩阵覆盖，已由现有守卫实际检出缺行。
- HTTP 可运行空桩取得 `t2-http-red` 真实断言失败后，接入 T1 用例与严格查询参数中间件，未提交桩。`t2-http-green-matrix-red` 的唯一失败是矩阵缺行；补入新操作六角色行。首次定向矩阵命令漏选 prepare 子用例，失败属于测试选择器错误，不算行为 RED，完整日志保留后修正命令。
- HTTP/契约与矩阵分别于 `t2-http-and-contract-green-v2`、`t2-permission-matrix-green` 通过。模块 HTTP TestMain 另准确检出新端点三个声明错误码尚未经本包 CheckResponse 验证，补既有错误映射表，不豁免。架构与平台契约检查通过，源输入稳定。
- 文档预检提交 `e52d38ea` 的 CI41839754475 已全绿，仅证明旧产品树；本阶段产品尚待独立 CI。
