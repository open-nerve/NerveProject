---
status: open
from: M5/closeout
to: M7
created: 2026-11-06
---

# M5 交给 M7：收集箱的图片与附件、附件事件与通知、只读显示与复制为 Markdown、首个投递任务的 CLI

这是接收方尚待完成的工作。转交只表示责任和验收条件已落实到此处，不表示本页功能已经实现或通过验收。
来源为 [M5 设计 §13](../../M5-files/M5-design.md#13-输入交接与后续交接)、[P8 阶段记录](../../M5-files/reviews/P8-final-acceptance-review.md)与其中的 T12 最终矩阵；M5 的最终集成与 main CI 由 M5 收尾记录，不以本页替代。资产用途、宿主接入的六步与清理协议见 [M5 → M6 交接](../../M6-cycles-modules/handoffs/M5-closeout.md#M5-M6-01)，本页不重复。

<a id="M5-M7-01"></a>
## M5-M7-01 收集箱的图片与附件

- **现状**：收集箱的创建弹窗不启用图片（`components/inbox/modals/create-modal/issue-description.tsx`）；其测试 `create-root.files.test.tsx` 断言粘贴的图片被丢弃、没有文件请求，创建仍走 M7 未迁移的旧 `POST /api/workspaces/…/inbox-issues/`。M5/P7b 删去了这个弹窗里从未被调用的上传回调与 bulk 状态请求。收集箱详情的正文（`components/inbox/content/issue-root.tsx`）与 `IntakeDescriptionVersions` 也没有图片能力；没有图片能力时编辑器仍加载图片扩展，但拒绝上传，已有图片显示为“不可用”。前端的 `ContentHost`（`store/asset/content-view.ts`）只认 issue、new_comment、comment、draft。
- **M7 接通收集箱时**先定宿主：收集箱条目若就是工作项，复用 `ISSUE_DESCRIPTION`/`ISSUE_ATTACHMENT` 与工作项宿主的读取、权限、删除和清理；条目建成之前就要上传的，需要一个准备件范围（先例：以工作项为 id 的 `new_comment`、以工作区为 id 的 `new_project`、草稿的私有准备件与原子发布）。服务端的内容绑定只接受工作项宿主（`asset/app/content_bindings.go`），用途与放置规则只认 issue/comment（`content_binding_rules.go`），新宿主须扩展这两处；前端 `ContentHost` 与 EditorImages 能力随之扩展。正文只存资产 ID，不存签名地址。
- **不能做的**：恢复 Plane 的 `/api/assets/v2` 或 bulk 地址；为收集箱另写一条上传链；在旧 scope（换工作区、换账户、撤权）的迟到结果上写新宿主。

**何时验收：** 收集箱的图片或附件第一次开放前。上传、拒绝、取消、换宿主与迟到结果、权限与访客、删除与清理，需与 M5 的 F8–F12、F16 页面故事同等的证据（三浏览器，local 与 S3）。

<a id="M5-M7-02"></a>
## M5-M7-02 附件事件与通知

- **事件名**：`asset.attachment_added`、`asset.attachment_deleted`（`asset/domain/events.go`）。只有已绑定的普通工作项附件（`ISSUE_ATTACHMENT`）发出；评论附件、草稿附件与正文图片不发（评论附件随评论本身，草稿附件在发布时随工作项迁移并在那一刻发出 added）。
- **载荷**（`shared.Event`，`asset/app/attachment_publication.go`）：`ID`（UUIDv7）、`Type`、`OccurredAt`（绑定或删除的业务时刻）、`WorkspaceID`、`ProjectID`、`ActorID`、`Entity{Kind: "asset", ID: 资产ID}`、`Changes=[{Field: "issue", NewID: 工作项ID}]`（added）或 `OldID`（deleted）。载荷没有文件名、大小、内容类型或任何地址。
- **发出点**：确认上传（complete，含并发确认的胜者）、复制、删除、草稿发布。**事务边界**：在写入的同一事务内同步投递（`bootstrap/events.go`，按订阅注册顺序），没有 outbox；订阅者返回错误即整笔回滚。现在唯一的消费者是 issue 模块：写一条 `field=attachment`、verb 为 created/deleted 的工作项动态，并更新工作项的变更时间戳。重复确认不再发；同一附件两次 DELETE 只发一次（`bootstrap/attachment_events_test.go`）。
- **M7 的通知订阅它们（或工作项动态）时**：订阅者在写入事务内运行，不得做网络 I/O 或长时间工作；需要外发的，照 `asset.delete_scope` 的先例在同一事务投递 River 任务，由任务在事务外完成。通知只携带 ID，显示时经工作项宿主的权限重新读取；不能把签名地址或文件内容写进通知。Webhook 属 M8，同一合同见 [M5 → M8 交接](../../M8-open-release/handoffs/M5-closeout.md#M5-M8-04)。

**何时验收：** 附件通知开放时：附件增删各一次通知、重复确认与重复删除不重复、回滚不留通知或任务、无权与已离开的成员收不到、通知点开后按当前权限读取。

<a id="M5-M7-03"></a>
## M5-M7-03 只读显示与复制为 Markdown

- 通知卡片用只读的 `LiteTextEditor`（`components/workspace-notifications/sidebar/notification-card/content.tsx`），没有 EditorImages 能力；正文里的图片会显示为“不可用”。若通知或其他 M7 视图要显示带图的正文，须经该宿主的内容读取能力（按当前权限签名、到期续期一次），不能复用通知自己的地址或缓存签名地址。
- 复制为 Markdown 走 `getMarkDown` 与 `useCopyText`，图片被省略（签名地址不外泄），这是 M5 结束时的已知限制。旧的 `EditorRefApi.copyMarkdownToClipboard` 没有消费者，已在 P8 删除（`1718210a`）。M7 的视图或协作功能若需要“带图片复制”，须先设计不泄露签名地址、换宿主后不越权的方式。
- 视图表单（`components/views/form.tsx`）是纯文本，没有资产。

**何时验收：** 第一次在通知或视图中显示带图正文、或提供带图复制之前；三浏览器核对权限变化、到期与换账户。

<a id="M5-04"></a>
## M5-04 首个投递任务的 CLI（原文转交）

下面完整引用 M4 交给 M5 的 M5-04，其中又完整引用 M2 原第 1 节；相对链接按原出处解释，原文不改。可点击来源：[M4 → M5 原交接](../../M5-files/handoffs/M4-closeout.md#M5-04)。

<!-- M4-M5-04-BEGIN -->
````text
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
````
<!-- M4-M5-04-END -->

**M5 结束时的事实：** M5 仍没有投递任务的命令。CLI 的组合（`bootstrap/users.go`、`workspaces.go`、`commands.go`）只构造连接池与管理用例，不接事件发布者；`jobs.NewInserter` 唯一的非测试调用在 `bootstrap/app.go`（serve）。

**何时验收：** 首个投递任务的 CLI 提交中完成，验收同上文原文；M6、M7、M8 谁先加入这样的命令，谁完成并在另两份交接中写明。
