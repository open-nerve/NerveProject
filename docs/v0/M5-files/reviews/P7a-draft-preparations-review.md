# M5/P7a 阶段评审记录

> 2026-11-05：P7a T1–T7/A1–A6完成；开发基线963636757fe5c5a8c10d5781b43d1481ba6f32a2，最终候选756b90314009f00e487483729b5d19f76dbeb110、noff maincf07c9fbe663b2e2c821c5988bb06933325b8ac1的准确CI分别全绿。全量原件归档核验、分支/worktree正常清理及P7b正式预检完成。当前不实施P7b，M5总体仍未完成。

## 1. 范围与方式

[主spec](../specs/P7a-draft-preparations.md)与[plan](../plans/P7a-draft-preparations.md)只交付准备件查询协议及真实HTTP恢复证明。原P7内容页面预检的原始报告为[re-46](../../../reviews/re-46-M5P7预检.md)，其架构问题已经负责人选择A，普通副本Issue/Draft结果由主控局部明确。新P7a读取协议是新的设计范围，做一次fresh builder预检；实现完成后才做一次整分支deep-review，二者不冒充。

## 2. 当前验收

A1–A4由T1–T4验证；A5由完整门禁、整审修复及准确最终分支CI闭环，A6由noff、准确mainCI、归档清理和P7b正式预检闭环（§13）；原预检时无新API、产品修改、依赖安装或P7a产品测试。原P7准备时只运行既有TestDraftAssetTransferPublicationStates三个子项，通过且1707个Go输入稳定；它证明原发布合同，不能当新恢复入口验证。

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

## 11. T6 一次整分支审查

[re-48](../../../reviews/re-48-M5P7a整体审核.md)审查冻结范围96363675..ed1e2a49，41文件/37手写完整读取，生成物对源；未发现架构风险。唯一F1为全零draft UUID进入owner内部前置检查导致500。审者评Minor，主控按私有入口不存在同形合同列为本轮必须修复边界；当时裁定修复不改owner或全局mapper，并在正在运行的门禁结束后完成实际RED/GREEN。

R12：全零draft UUID在认证和分页参数校验后、事务前返回asset.not_found；不扩大错误捕获。理由是合法可解析但不可存在的资源ID不应成为服务端故障；代价是该入口将全零值按不存在处理，旧入口保持其原行为。最终准确分支CI合并检查点与修复后的验证，不为已知待修候选额外推送一轮CI。

Final Ruling：审者暂不判断的P7b实际页面行为继续留P7b，不计入P7a验收；已经核查API支撑及接收合同。代价是当前用户可操作的恢复界面仍待页面阶段，不能宣称F10页面完成。

首轮七门禁全部通过：gen-check 2.973秒、Go lint 1.290秒（两模块0 issues）、完整Go 998.744秒（bootstrap649.319秒）、web lint54任务/15缓存、knip通过、web test18任务/11缓存、完整E2E385/385（170.363秒）。每项4544输入稳定，raw/receipt完整保存。容器即时快照有一个临时件尚在退出，随后只读复核已自动消失：原20个全部在、新遗留0，未操作既有容器。

F1有效RED：应用层观察到非法开启事务，真实HTTP得到500；随后三行入口检查修复，两层GREEN及其他准备件回归通过。定向domain包显示no tests to run，不冒称该次运行了domain测试；全门禁负责完整覆盖。没有扩大错误捕获、改写owner、放宽守卫或新增依赖。当时最终七门禁/准确候选CI尚待后续收据；现已由§12/§13闭环，原始时间顺序保留。


## 12. 最终本地交付收据

F1修复提交`7e041b472044409105eaa9322f2e0ad079ab5ee3`后，重新串行执行全部七门禁；每项raw SHA/字节及前后4544个非文档输入与当前提交内容逐项一致。与首轮相比仅一个应用文件及两份回归测试改变，变化清单`t6-fix-input-delta.json`保留。未以首轮成功替代修复验证。

| 原件标签 | 耗时 | 退出/输入 |
|---|---|---|
| t6-gen-check | 3.192 秒 | 0；4544输入一致 |
| t6-lint-go | 20.069 秒 | 0；4544输入一致 |
| t6-test | 1005.665 秒 | 0；4544输入一致 |
| t6-lint-web | 2.768 秒 | 0；4544输入一致 |
| t6-knip | 1.976 秒 | 0；4544输入一致 |
| t6-test-web | 0.233 秒 | 0；4544输入一致 |
| t6-e2e | 167.696 秒 | 0；4544输入一致 |

完整E2E385/385；Go使用-count=1、不使用测试结果缓存。Web任务的实际缓存数保留在各raw与.turbo日志；不把缓存命中说成全部重跑，也不从未开启-v的Go输出推断所有子测试零skip。两次全套、全部历史失败与有效变异、一次审者结论和F1 RED/GREEN均保存；未派第二同范围审者。

A1–A4已验证，F1修复闭环，没有延后minor。本地复验记录时A5仍有本次记录提交的准确最终分支CI门；T5检查点和T6最终CI合并核该候选，server/web/e2e全部成功后才在交付收据中关闭，不把旧预检CI或修复前结果作为凭据。随后持续授权下执行T7，main另跑准确CI。准确CI与清理结果在原件收据和P7b接续文档回填，避免为引用自身SHA不断生成新候选。

P7b页面、编辑器/完整副本交互、旧文件链删除与M5最终验收/handoffs继续开放。整审暂不判断的页面事项已由这个具名边界接收，P7a只交付恢复协议。

## 13. T7 集成归档与接续

2026-11-05再次获得明确授权后，准确候选`756b90314009f00e487483729b5d19f76dbeb110`的[分支CI41859931318](https://github.com/open-nerve/NerveProject/actions/runs/41859931318) attempt1三个作业均success；server126678612716、web126678613048、e2e126687761531。E2E日志为385/385，18.7分钟。GitHub Actions故障一度使run/job元数据不同步，原始状态和日志完整保存；交付校验首次拒绝且T5/T6未关闭，未据单一步骤成功合并。三作业与workflow一致后，原始七门禁日志SHA及4544输入重新核验，T5/T6关闭。

根main原基线963636757fe5c5a8c10d5781b43d1481ba6f32a2与origin一致且干净。`--no-ff`合并生成`cf07c9fbe663b2e2c821c5988bb06933325b8ac1`，双父依次为原main与最终候选，完整tree`2752264a0d7e0117344ec3af740ef8e8d2813220`与候选完全相同；系统Git推送成功。[准确main CI41869045846](https://github.com/open-nerve/NerveProject/actions/runs/41869045846) attempt1三个作业均success，原始run/all-jobs与日志单独保存；没有用分支CI替代main结果。

main E2E再次385/385通过，耗时19.0分钟；server126710759480、web126710759869、e2e126721692618的身份与必需步骤逐项核准。本次无需重跑，未改CI预算。

main CI通过后已完成下述归档清理；P7b worktree从该main创建，仅准备spec/plan及预检。原生worktree工具不接受自定义路径，按用户硬规则使用Git落在`.codex/worktress/codex/m5-p7b`，主目录未切开发分支。P7b未安装依赖、未启动T1、无产品修改。其唯一fresh builder问题由原IssueDetailHost保留本地输入的局部合同补全，不改变M4权限或通用读取算法。

归档根：`/Users/xiaoruan/project/nerve-project/.superpowers/sdd/M5-P7a-integration-20261105`。包含原P7改名前的准备台账、本阶段全部失败/变异/输入清单、两轮本地七门禁、审者报告、GitHub故障及准确branch/main CI、浏览器报告和全部.turbo原件；可重建依赖/构建产物另有明确排除清单。共2,885个文件、95,387,129字节、4,684个成员，逐SHA/bytes/mode/membership核对，unknown=0，源归档期间未改变。manifest SHA-256为`a941e831c48255c33ed9eddf6193211ae4db0e996eda9ab5d99ed36ac67521ae`。

确认本地main=origin/main=cf07c9fbe663b2e2c821c5988bb06933325b8ac1，origin/main..origin/codex/m5-p7a为0后，正常删除远端分支、旧worktree及本地分支；未使用force/stash/clean/reset-hard。清理后从归档副本再次核对所有原件，结果一致。[清理收据](/Users/xiaoruan/project/nerve-project/.superpowers/sdd/M5-P7a-integration-20261105/cleanup-receipt.json)与[清单](/Users/xiaoruan/project/nerve-project/.superpowers/sdd/M5-P7a-integration-20261105/manifest.json)保留在冻结payload之外/入口，原件不再改写。

P7b正式基线为该main，四份合同已接收真实P7a API、原DELETE边界、草稿owner寿命和输入连续性；一次fresh builder及主控局部裁定闭环，16Task/48开放Step/8开放验收、产品零变化/无依赖安装和文档链接自检通过，详见[re-49](../../../reviews/re-49-M5P7b预检.md)。T7最后完成收据在继任P7b worktree的同名P7a ledger接续登记，冻结归档保留清理前现场。P7b未启动T1，M5最终矩阵/交接继续开放。
