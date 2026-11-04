# M5/P7a 阶段评审记录

> 2026-11-04：按负责人已批准A同步合同并通过预检；现已获准实施，T1–T4 已完成，T5 进行中。分支codex/m5-p7a；main基线963636757fe5c5a8c10d5781b43d1481ba6f32a2，P6准确main CI41833460144及归档/清理前置已核；预检通过，实施证据见下。

## 1. 范围与方式

[主spec](../specs/P7a-draft-preparations.md)与[plan](../plans/P7a-draft-preparations.md)只交付准备件查询协议及真实HTTP恢复证明。原P7内容页面预检的原始报告为[re-46](../../../reviews/re-46-M5P7预检.md)，其架构问题已经负责人选择A，普通副本Issue/Draft结果由主控局部明确。新P7a读取协议是新的设计范围，做一次fresh builder预检；实现完成后才做一次整分支deep-review，二者不冒充。

## 2. 当前验收

A1–A4已由T1–T4验证，A5/A6待完整门禁、整审及集成闭环；原预检时无新API、产品修改、依赖安装或P7a产品测试。原P7准备时只运行既有TestDraftAssetTransferPublicationStates三个子项，通过且1707个Go输入稳定；它证明原发布合同，不能当新恢复入口验证。

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

## 7. T2 实施记录

- 新路径通过模块 YAML 和总入口 `api/openapi.yaml` 引入；生成 Go/TS/汇总契约。R8：plan 文件清单未列总入口与权限矩阵，将其按既有契约流程纳入；不改守卫或豁免。代价是新增操作必须持续保持矩阵覆盖，已由现有守卫实际检出缺行。
- HTTP 可运行空桩取得 `t2-http-red` 真实断言失败后，接入 T1 用例与严格查询参数中间件，未提交桩。`t2-http-green-matrix-red` 的唯一失败是矩阵缺行；补入新操作六角色行。首次定向矩阵命令漏选 prepare 子用例，失败属于测试选择器错误，不算行为 RED，完整日志保留后修正命令。
- HTTP/契约与矩阵分别于 `t2-http-and-contract-green-v2`、`t2-permission-matrix-green` 通过。模块 HTTP TestMain 另准确检出新端点三个声明错误码尚未经本包 CheckResponse 验证，补既有错误映射表，不豁免。架构与平台契约检查通过，源输入稳定。
- 文档预检提交 `e52d38ea` 的 CI41839754475 已全绿，仅证明旧产品树；本阶段产品尚待独立 CI。

- T2 提交 `f9ff9473`：`t2-gen-check` 输入不变且通过，`t2-final-bootstrap` 通过；HTTP adapter 全包通过；Go lint 首轮仅导入分组错误，修正后主模块/工具模块均 0 issues。T2 完成，A1 的协议/参数/DTO 验证已具备；A2/A4 的撤权与并发责任留 T4。

## 8. T3 恢复证明

- `TestDraftAssetPreparationsRecoverLostAnswer` 丢弃真实 prepare 成功回答，后续仅从新列表获 ID；两个同名准备件、另一草稿邻项、无项目草稿转项目、拒绝发布无写入、重复删除/确认无 audit/event/task 重复均有实际 HTTP 与数据库断言。
- `TestDraftAssetPreparationsRecoverFailedCopy` 在真实封存已发生后返回受控错误，调用方仅收到 503；从列表发现 pending 后才用 SQL 断言恢复记录，删除并成功发布，来源字节不变。另有 uploaded 未绑定图片可列、正文当前/历史图片不列且仍受 asset.bound 保护。
- `t3-recovery-baseline` 通过；`t3-pending-mutation` 临时漏掉 pending 后两个恢复用例均真实失败，SQL/生成物 SHA 精确恢复。E2E 在 local/S3 用同一故事验证丢失回答和 duplicate 目标已提交后图片校验失败，不依靠 SQL 或存储键发现 ID。
- R9：T3 构建同时执行既有 `make build-s3-fixture`，因为 fileStory 复用现有 local/S3 双驱动夹具；没有新依赖或生产故障入口。代价是定向 E2E 多一次本地测试辅助程序构建。格式化沿仓库锁定 oxfmt；prettier 未安装的命令失败未触发安装，不计测试失败。

- T3 提交 `aeac42f8`；最终服务端恢复回归通过，local/S3 四个 API E2E 全通过（30.3 秒），4542 个非文档输入不变，原件与 HTML 报告单独保留，不由后续运行覆盖。E2E 零新增警告，Go 主模块/工具模块 lint 均 0 issues。

## 9. T4 快照与写入交错

- 复用现有 `attachmentSnapshotTrace` 的 pauseSQL 能力和真实应用夹具，暂停于完成 Host 授权后的 ListDraftPreparations 语句前；实际 PID 的 backend_xmin/xact_start 证明快照已建立。草稿/工作区/项目删除、换项目、发布、降 guest、退出工作区、停用和注销均以第二请求提交后再放行；当前页旧事实一致，后续页重验。取消后精确 PID 不再持事务，下一读成功；其他工作区保持独立。
- 列表看过 pending 后，描述图绑定拒删与普通附件完成后可删分别验证；删除优先时迟到 PUT/complete 不复活。真实 D(U)/项目锁 PID 证明 publish 与新 prepare 的两种顺序：申请先提交使发布拒绝 pending，发布先提交使申请拒绝已消费草稿。分页间删除/新增、跨作者、跨草稿和越界结束均有断言。
- `t4-race-five`：13 个顶层用例 × 5 轮，65 次通过，零 skip/失败/data race；95.520 秒、4544 非文档输入前后一致。日志 60,754 字节，SHA `00362fa8030c707a2b7d23ffe9155917ce79216270fc46cbf3cb44fb8fda19fa`。
- 有效变异：T1 的 `t1-actor-mutation-v2`（他人归属被漏入）；T4 的 `t4-unbound-mutation`（历史绑定暴露）、`t4-split-snapshot-mutation`（实际 PID 未建立授权快照）、`t4-r1-mutation`（宿主数据库故障从500变404）、`t4-cross-draft-cursor-mutation`（跨草稿返回200而非400）。每项独立失败后精确恢复 SHA，未用编译或环境失败冒称。
- R10：任务关闭对刚完成的成功 raw/receipt/前后输入清单和当前全部非文档输入做 SHA 核验，避免仅因记录文档或提交就重跑完全相同的长测试。测试命令仍已按 plan 实际执行，任何产品/测试输入变化均须重新运行；代价是测试时 HEAD 与随后记录提交可能不同，因此保留两者并以完整输入内容证明对应，不拿其他提交的 CI 代替准确 HEAD CI。
- `t4-p5-regression` 通过：原草稿双发布、1001件迁移、失败回滚、内容保留/清理及复制回归，耗时37.419秒；`t4-lint-go` 主模块和工具模块均0 issues，耗时15.438秒。两者4544个非文档输入前后一致，与race五轮输入相同，均由receipt关闭核验；T4完成。

## 10. T5 全部门禁与接收核对

T4记录提交为`8d3313ae`；与此前成功测试的4544个非文档输入逐项相同，原始日志SHA核验通过。以下验收映射不以单元测试替代真实调用：

| 验收 | 证据与观察边界 |
|---|---|
| A1 | T1 domain/app/postgres及T2 HTTP/契约/权限矩阵；51/101+分页、规范cursor、严格参数、已有Asset DTO和错误码均有实际断言。生成物来自YAML/SQL，T5 gen-check再次通过。 |
| A2 | T2拒绝同形和T4九种宿主/会话变化；当前页一致、后续页重验；其他工作区不受影响；新列表guest404而旧draft仍403；两类数据库故障仍500。 |
| A3 | T3两个未知ID恢复链、同名邻域及历史图片；local/S3四个API E2E通过。仅从新API取得待丢弃ID，SQL只作结果断言；不领取页面完成。 |
| A4 | T4真实PID快照和写锁交错、取消释放、分页变化、65次race顶层用例及P5联合回归；五项有区分度变异均恢复原SHA。 |

依赖和副作用核对：domain仅依赖shared cursor；app声明Inspect-only和List-only小接口，组合根沿既有Hosts/ReadTx/store注入；SQL仅访问file_assets且保留完整父链、双actor、unbound/live条件。新查询没有存储依赖、写锁、事件、River任务或计数更新，HTTP复用assetDTO，不传key、metadata或签名能力。原写入口继续独立授权，不把列表当写入许可。

P7b接收以[生命周期子合同§6](../specs/P7b-content-lifecycle.md#6-草稿准备件恢复)为准：显式打开、分页、逐ID选择和确认；未知DELETE先只读核验，不因缺席断言已删；普通附件与描述图的绑定后删除行为分别说明。F10页面及旧文件链清理仍由P7b承担，M5最终验收和handoffs仍开放。

R11：T1–T4产品冻结后，让一次只读整审与T5完整门禁/CI并行；测试结束前不应用审查修复，A5仍须门禁、修复回归和准确最终CI全部通过。理由是两者独立且不改变任何产品输入；代价是审查若要求修改代码，受影响门禁须重新验证，原运行仅代表原候选。
