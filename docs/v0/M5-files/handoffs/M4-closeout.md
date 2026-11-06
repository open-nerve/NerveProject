---
status: closed
from: M4/closeout
to: M5
created: 2026-11-02
---

# M4 交给 M5：附件、图片内容标识、删除与首个投递任务的命令

这是接收方尚待完成的工作。M4 输入的“转交关闭”只表示责任和验收条件已落实到此处，不表示本页功能已经实现或通过验收。
来源为 [M4 设计 §13](../../M4-issue-core/M4-design.md#13-交接的落点)、[逐条交接索引](../../M4-issue-core/reviews/closeout-handoffs.md)和该索引中的各 Phase spec §5 / review §6。
M4 的最终 C12、准确分支 CI 和未来 main 集成分别由[验收台账](../../M4-issue-core/reviews/closeout-acceptance.md)记录；不以阶段历史绿灯替代最终输入。

<a id="M5-01"></a>
## M5-01 文件接口与工作项附件

- 在文件服务真实可用时加入 `Issue.attachment_count`、对应排序码和显示属性“附件”；字段与真实数据一起进入 OpenAPI、生成类型、比较器和页面，不能先留虚假的零值。
- 恢复工作项详情附件区、评论附件。上传和显示经 M5 文件接口及真实预签名 PUT；编辑器只有在这条链可用后才能从 `disabledExtensions` 去掉 `image`。
- M4 不挂载附件区。工作项详情子树已按工作项身份设 key；M5 挂入确认框、上传弹窗时核换工作项、换工作区、换账户和迟到上传结果，不能把旧文件写给新宿主。
- 现有文件服务调用方仍包括 `comments/comment-create`、`inbox/modals/create-modal/create-root`、`hooks/editor/use-editor-config`。接真实接口时逐个替换旧地址和到期关键词例外，复用会话有效性与已有 `EditorRefApi`，不恢复模块级带会话实例。

<a id="M5-editor-copy"></a>
**旧复制入口的具名风险与接收条件：** `EditorRefApi.copyMarkdownToClipboard`（`web/packages/editor/src/helpers/editor-ref.ts`）仍保留为公开ref合同，但当前M4产品走`getMarkDown`与`useCopyText`。旧方法添加document copy监听后依赖copy事件自行释放；`execCommand("copy")`返回false、抛错或未派发事件时没有对应释放路径，也不提供可靠拒绝反馈。来源见[死成员复核](../../M4-issue-core/reviews/closeout-dead-members.md#editorrefapi-copy-risk)及[收尾整审F3](../../../reviews/re-30-M4收尾整体审核.md#F3)。

M5执行者在**首次复用该方法的页面合入前**，须改走现有受控复制通道，或先修复该方法的监听生命周期与失败反馈，再开放消费者。验收包括成功、无用户手势、拒绝/false/异常、宿主卸载和换编辑器：操作结束或退出后监听均释放；后续其他编辑器/页面的copy不被旧内容或旧监听截获；不得出现失败成功提示。没有完成这些条件前，不因“EditorRefApi已保留”认为此入口已经验收。本条不要求M4启用或改写当前未使用入口。

**何时验收：** M5 首次启用附件/图片的阶段。真实上传成功、拒绝、取消、过期会话、换宿主、预签名失败均有页面与持久化断言；无成功假提示、无旧作用域写入。

<a id="M5-02"></a>
## M5-02 草稿发布的同事务迁移与图片标识

- 草稿附件在发布时随新工作项迁移；创建工作项、附件归属迁移、软删草稿在同一事务中成功或回滚，保持唯一发布，不能发布后再补搬文件归属。
- 开启图片后，内容标识加入**按顺序的 `image-component[src]`**。现在的文本、提及、链接地址、勾选框规则继续保留；换图、调序必须不能借 `skip_activity=true` 绕过动态、版本和更新时间。
- 请求仍是严格布尔值；只补合法 ID 且内容标识不变才享受迁移更新。ID 去重/生成已在 P16 修正，M5 不新建另一套。
- 恢复图片上传前同时核富文本清洗、允许的图片地址和显示结果，不能只测试编辑器 JSON。

**何时验收：** 草稿附件与图片首次开放前。并发双发布、事务内故障回滚、恶意 HTML、只补 ID、换图/换序、撤销重做后发布必须有区别鲜明的反例。

<a id="M5-03"></a>
## M5-03 删除图、事件与物理清理

- `file_assets` 到工作项、评论、草稿的外键延伸 M4 §4.16；同步扩充 `deleteIssue` 的连带、`issue/adapter/postgres/delete_rows.go` 与 `furnishIssueRows`。M3/M4 的项目、工作区、工作项删除故事必须真实填充和断言新表；目录守卫不加豁免。
- 项目/工作区删除的同步事务使父范围立即不可见；M5 订阅 `project.deleted`、`workspace.deleted`，在同事务投递自己的任务，异步软删自己拥有的行，沿用原请求 At/By，重试不改审计时刻。
- 项目/工作区批量删除不逐个发 `issue.deleted`；只订阅工作项事件会遗漏这两种入口。普通删工作项对子树每项发布事件，消费者按各自边界处理。
- 60 天物理清理由 M5 自己的清理者处理文件行与存储对象，并延伸目录/外键图、有限批次和失败重试证明；不得绕开其他模块 owner 或做跨模块 JOIN。
- 附件增删是否另有事件由 M5 依 M4 §3.19 决定，并把名字、载荷、事务边界写入自己的设计和通知/Webhook 交接。

**何时验收：** 第一张附件表合入时同步延伸测试；M5 收尾前验证删除立即不可见、任务原时间/操作者、回滚不留任务、邻域不受影响、过期清理与存储重试。

<a id="M5-04"></a>
## M5-04 首个投递任务的 CLI

下面完整引用 M2 原第 1 节；其中相对链接按原出处解释，原文不改。可点击来源：[M2 → M4 原交接](../../M4-issue-core/handoffs/M2-closeout.md)。

<!-- M2-SECTION-1-BEGIN -->
```text
## 1. 命令行的"只投递"River 客户端（负责人：做好记录，以后别漏了）

- **负责人的要求**：2026-09-26 批准把它推迟到 M4 时说，"做好记录，以后别漏了"。M4 用不到它，也要把这一节原样转进下一个 M 的交接，不能随本交接关闭而丢掉。
- **做什么**：第一个"用例会投递 River 任务、又能从命令行触发"的命令加入时，`platform/jobs` 提供一个只投递的客户端（不配队列，不调用 `Start`，`river@v0.47.0/client.go:89-95`），`bootstrap` 的命令组合把它交给这些用例。
- **为什么**：命令行的组合只有连接池和 `identity` 的管理用例，没有 River 客户端（`server/internal/bootstrap/users.go:22-26`，M2 设计 3.17）。服务用的客户端（`server/internal/platform/jobs/jobs.go:70` 的 `New`）配队列、启动 worker，命令行不能用它。用例在事务里用 `InsertTx` 投递（M2 设计 3.15）；命令行上没有客户端时，任务要么没投递，要么用例在运行中失败。M2 的命令都不投递任务，所以推迟（M2 设计 3.15、3.17、13.2）。
- **可能早于 M4**：M3 给停用加端口（[M3 的交接](../../M3-workspace-project/handoffs/M2-closeout.md)第 6 节），`nerve users deactivate` 也走这个用例；端口的实现投递任务时，这一项提前到 M3。
- **建议的守法**：投递任务的端口在用例的构造参数里必填，命令的组合漏了它，在构造时就失败，而不是运行到一半；`server/internal/bootstrap/users_test.go` 这样的组合测试覆盖每个命令。
- **关闭条件**：M3 已为停用的端口加入这个客户端时，M4 的 review 核对它合这一节（没有队列、不调用 `Start`，组合测试覆盖每个命令）并写明；否则，M4 加入了这样的命令时，命令的组合有只投递的客户端，一个测试在测试库上运行这条命令，核对 `river_job` 多了这条任务、事务回滚时没有。M4 结束时仍没有这样的命令，M4 的 review 写明，并把本节原样写进下一个 M 的交接，本交接才能改为 `closed`。

```
<!-- M2-SECTION-1-END -->

**当前事实与责任：** 只投递的客户端已在 `platform/jobs`（`NewInserter`，M4/P6）；M3、M4 没有投递任务的命令，现有 CLI 组合无需构造它。**第一个投递任务的命令加入时，把它交给命令行的组合**，按 §3.20 显式声明投递的 kinds，启动时验证对应 worker。

**何时验收：** 首个投递任务的 CLI 提交中完成。真实命令经运行角色执行，验证 `river_job` 的 kind/args、提交留任务、回滚不留任务；少登记一个 worker 的负例必须失败。不能把服务端已有 NewInserter 当 CLI 已接通。

## 处理结果（M5/P2）

M5-03已落实首表同步删除图、原At/By的事务内范围投递、目录全边/真实实物夹具和NO ACTION数据库兜底；T4/T5/T9证明受保护父清理能前进并恢复收敛，T11证明五种父删除COMMIT故障整体回滚。M5-01/02的绑定、计数、内容标识与草稿迁移仍归P5/P7；P2没有新增投递任务的CLI，M5-04继续保留至首次命令或收尾转交。全部交接保持open。证据见[P2阶段评审](../reviews/P2-assets-review.md)，整审/集成结果仍单独核验。

## 处理结果（M5/P5 T13核对）

M5-01的全部Issue真实附件计数、全局排序、三类附件分页和已有store数据消费已接通；M5-02有序src、当前图片绑定、草稿完整集合换项目/发布及七处故障回滚、唯一发布、历史图片保留已验证。M5-03覆盖真实六用途资产、五种父删除、61/59天保留、物理权限故障/恢复与实际新OS进程继续River游标，目录守卫未放宽。原始证据及边界见[P5记录§20](../reviews/P5-issue-assets-review.md#20-t13验收与交接核对)。

本页保持open：P7才开放工作项/评论/编辑器图片与列表控件，撤销重做、伪造剪贴板和换宿主迟到结果仍需页面验收；P8汇总最终清理与三浏览器证据。旧copyMarkdownToClipboard没有新增消费者，也未声称已修复。P5未新增投递任务CLI，M5-04完整原文继续保留。阶段七项、整审与集成另记，不能由本段API证据代签。

## 处理结果（M5/P7b T12核对）

上文列出的旧文件服务调用方已清零：评论创建在T6改为私有准备件加一次POST，编辑器在T7–T9改用EditorImages能力（`use-editor-config`在T8删除），M7未开放的收集箱创建弹窗在T12删去从未被调用的上传回调与bulk状态请求、不启用图片。旧FileService/FileUploadService、编辑器资产store与hook、`EFileAssetType`与旧文件类型、旧文件URL工具和services包全部删除；已开放页面的行为测试在生成客户端、原始传输和旧APIService三条通道上断言没有`/api/assets/v2`、bulk、旧restore或multipart。旧copyMarkdownToClipboard没有新增消费者，复制为Markdown时图片省略（签名读取不外泄），具名风险仍归P8。本页保持open。

## 处理结果（M5/P7b T14核对）

T13真实页面在M4的browse详情路由发现并修复两处，均在路由页面层，不改通用权限/读取算法（DetailResource、IssueVerification、权限观察器）：`370aba81`——路由以`!issue || pending`门控，写入后的规范重读把Issue标为pending时整页换成骨架屏，卸载IssueDetailHost及其保留输入，改为只在路由读取本身进行时显示骨架屏；`0d093f03`——以SWR瞬时isValidating代替“本次访问的读取已证实”，温缓存首帧挂载交互详情，重连重读已证实路由并卸载宿主，改为每次进入路由（含后退/前进）为一次访问、由其自发唯一lookup、应答前不挂载交互子树，并与M5设计§9的宿主读取一样关闭隐式reconnect重读（useSessionSWR增窄选项`revalidateOnMount: false`）。原生ID入口不变。T14门禁另修一处宿主状态归属：`5ff2e4a2`——`370aba81`后browse路由与peek一样在写后规范重读期间保留IssueDetailHost，宿主里的弹窗开关跨过了正文撤回，重读后正文重挂载即以空表单重开刚确认的链接/子项/关联弹窗；现弹窗开关随规范项在/不在换新，输入状态仍只随宿主scope。写后重读期间弹窗表单卸载时，只有链接表单的成功提示被跳过（M4详情页已如此，写入与列表正确）；子项/关联经选择弹窗提交，M4起只有失败提示，此前记作“其成功toast被跳过”不准确。P8 T6②改以宿主输入状态scope（`IssueDetailInputState.current`，正文撤回期间仍存活）判定链接提示：表单被撤下后仍提示一次，宿主scope结束则不提示，写入与列表结果不变；这是M4代码的本地修复，单独提交、可回退。本页保持open。

## 处理结果（M5/P8）

M5-01：真实`attachment_count`、排序与显示属性，详情附件区与评论附件经真实上传接通（P5/P7b），T12的F8/F9在三浏览器重跑。旧复制入口：`EditorRefApi.copyMarkdownToClipboard`没有任何产品消费者，T7连同类型声明删除（`1718210a`），具名风险随方法删除而结束，不是“已修复”；复制为Markdown走`getMarkDown`与`useCopyText`，图片省略（签名读取不外泄），这一已知限制写入M7/M8交接。M5-02：草稿发布的同事务迁移、有序`image-component[src]`内容标识与历史图片保留已验证（P5/P7a/P7b，T12的F10–F12）。M5-03：删除图、原At/By的范围投递与60天物理清理由T10全链路演练（local与S3）证明，M4的两项规模测试在M5代码上重跑：`TestTheDeletionsAtTheScaleTarget`原样通过，`TestPurgeScale`按M5的有界清理协议更新后通过（T12）。M5-04：M5仍没有投递任务的CLI，本节原文整段转交M6/M7/M8的`M5-closeout`。P7b T14段已更正为只涉链接表单的成功提示，P8 T6②修复（`8be9e329`）。证据见[P8阶段记录§7](../reviews/P8-final-acceptance-review.md#7-实施记录)。除M5-04转交外条款全部完成，closeout改为closed。

## 关闭（M5/closeout，2026-11-06）

M5-01至M5-03已完成，见上面的处理结果与[P8阶段记录§7](../reviews/P8-final-acceptance-review.md#7-实施记录)；旧复制方法的风险随方法删除而结束（不是“已修复”）。M5-04（首个投递任务的CLI，含M2原第1节）在M5仍未触发，原文整段转交[M6](../../M6-cycles-modules/handoffs/M5-closeout.md#M5-04)、[M7](../../M7-collaboration/handoffs/M5-closeout.md#M5-04)、[M8](../../M8-open-release/handoffs/M5-closeout.md#M5-04)三份`M5-closeout`交接；那三份交接保持open，转交不表示该项已实现。`status`改为`closed`。原文与各阶段处理结果保留不改。
