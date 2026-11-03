# M5/P5：工作项、评论与草稿资产

> 预检及T1–T12实现验证完成；T13本地七项已闭环，完整E2E 364/364通过，原容器20/新增遗留0。准确检查点CI、整审和集成待后续步骤。分支codex/m5-p5，准确基线main 0f93100a05304d26b074790633b5659285813156；P4 main CI 41641099230全绿且归档清理已完成。一次builder check的H1已按既有准备件隐私合同修正，见[预检](../../../reviews/re-40-M5P5预检.md)。
> 主合同：[内容与绑定](P5-content-binding.md)、[附件读取与计数](P5-attachment-reads.md)共同构成 spec。依据 M5 设计 §3、§6–8、§12 和 M4-closeout M5-01/02/03。

## 1. 目标和范围

让已实现的真实资产服务接入 issue/comment/draft：附件上传确认后进入合法宿主；内容保存验证并绑定图片；草稿发布原子迁移；读取/复制/删除遵循宿主权限；新增真实计数、排序和事件。P5 提供 API、生成类型及现有消费者的必要字段适配；上传界面/编辑器、列表的附件显示与选择控件分别在 P6/P7。P5 不提前启用图片扩展，不关闭 M5 总验收或输入 handoff。

不增加依赖、业务表、常驻副本计数、后台同步投影或跨模块 JOIN。沿用十用途、现有 file_assets、三步上传和不可变 seal。只读 Plane 根目录的 comment.py / asset/v2.py 作交互参考；不照搬确认直接置 uploaded、宽松批量改归属或吞 IntegrityError。

### 1.1 基线事实与局部决定

- asset/app/ownedAsset 和 Access.Within 目前统一要求 applicant=actor；P4 图片经独立 owner 能力读取。P5 只扩大已绑定的六种 issue/comment/draft 文件读取与附件删除，资料图片的通用入口仍不公开。
- 完成上传不等于绑定；已有 issue/comment/draft 上传实物可能 uploaded=true、bound_at=null。升级不凭历史 uploaded 批量公开未提交描述。既有附件在调用 complete 重试时按当前权限补成首次绑定，之后才进入列表/计数，不能伪造历史动作。
- 评论继续要求清洗后有正文、提及或图片，保持 M4 非空规则；不新增仅附件空评论。创建新增 attachment_ids（仅普通评论附件），描述图片身份从清洗后的 HTML 提取，两集合合并锁定。
- 评论附件采用独立分页端点，评论 DTO 不嵌无界集合。原因：评论页最大100条，嵌套附件会产生乘积；代价是展开附件需要一次分页请求。
- 附件计数由 asset 自己聚合，issue 沿现有名称目录的端口/成对数组路线消费；不增持久计数列。排序时读取授权项目的完整非零计数目录，普通返回只批量取已授权返回 ID。代价是附件排序的目录成本随有附件的工作项数增长，T12 做冷库容量证明。
- 未完成上传阻止发布；用户可删除不需要的 pending 资产后重试。已完成草稿资产整体迁移，包含从正文移除的图片；仅当前正文引用和原已绑定历史图保持绑定，从未引用的描述准备件迁移后仍未绑定、仅申请者可见和可删、继续按原创建时间7天清理。保留历史/撤销语义，代价是已绑定历史图片继续占空间。

## 2. 可观察合同

### 2.1 附件生命周期

用途 ISSUE_ATTACHMENT / COMMENT_ATTACHMENT / DRAFT_ISSUE_ATTACHMENT 在已有 issue/comment/draft scope 的 complete 或 duplicate 最终发布事务中绑定，bound_at 使用最终 held.At；new_comment 准备件只完成实物，创建评论时绑定并换为 comment scope。重复 complete 不增加绑定、事件、动态或 updated_at。并发确认胜者一次，失败候选按 P2 回收。

ISSUE_DESCRIPTION / COMMENT_DESCRIPTION / DRAFT_ISSUE_DESCRIPTION 只有被所属内容写引用才绑定；移除节点不解绑、不软删。已绑定描述通过通用 DELETE 返回 asset.bound，直至宿主删除；未绑定的准备图片仍可由申请者删除并按7天规则清理。资料图片规则不变。

DELETE 普通活附件要求所属工作项更新权限、评论删除权限或草稿作者；不是只检查上传者。活附件在本事务软删，重复 DELETE 对同一仍可见宿主、同一合法 actor 返回204，保留首次 At/By，零重复事件；隐藏/不存在统一asset.not_found，不能用幂等语义泄露资源。宿主已删时仍404。pending/未绑定仍仅申请者操作，重复删除也先核宿主。

### 2.2 API

- `GET /api/v0/issues/{issue_id}/attachments`、`GET /api/v0/issue-comments/{comment_id}/attachments`、`GET /api/v0/drafts/{draft_id}/attachments`：200 `{data,next_cursor}`，只对应宿主的活、已上传、已绑定普通附件；默认50，最大100，UUID升序，游标绑定版本/宿主种类/ID/最后ID。元数据无内部key、无签名 URL；内容沿现有 `/assets/{asset_id}/content` 重新授权。
- `IssueCommentCreate.attachment_ids` 可省略（等价空），非nullable、最多100、规范 UUID、uniqueItems=true；严格 bodyshape 区分结构400和语义422，错误位置稳定且不回显私有ID。PATCH 评论只修改 comment_html，后续附件通过 existing-comment 上传独立增删，避免隐藏的整体替换语义。

  T6落实：该数组项在OpenAPI用小写连字符UUID的string pattern保留原拼写；bodyshape先拒绝null/非字符串，issue HTTP适配器在调用用例前解析并拒绝非规范拼写，沿既有400字段错误协议。数量/重复留domain返回422。不改其他UUID字段的标准库解码语义，不为此改动共享bodyshape或生成器；PATCH使用独立IssueCommentUpdate合同。
- Issue 新增必需 int64 非负 `attachment_count`，所有返回 Issue 的入口同步真实填充；排序支持 attachment_count / -attachment_count，游标沿现有精确整数规则，禁止浮点转换；显示配置的合法键在 P7 页面开放时进入 UI，不先开放不可用控件。
- 新错误沿既有资产码优先复用 asset.not_found / asset.not_uploaded / asset.scope_changed / asset.bound；若确需具名字段错误同任务补 OpenAPI x-problem-codes、中文英文文案、生成 Go/TS 和全程序负例。server_busy 仅 bootstrap 映射。

### 2.3 内容与草稿

有序 image-component[src] 加入 ContentIdentity.Images []string（保留重复和顺序，缺src视空字符串）。只有规范小写连字符 UUID 进入资产引用校验；远程安全 src 原样参与标识但不取得内部授权，不能替用户抓取或在后续页面发远程请求。raw img 延续既有清洗和 HasImage，但不作为内部资产绑定通道。文本、提及、链接、checkbox、合法block ID规则不变。

正文 PATCH 与评论创建/修改：先原有可见性和写权限、归档规则，再校验当前内容涉及的资产集合；新的未绑定ID须申请者=actor、已上传、用途/完整父链匹配。已绑定同宿主图片允许有写权的其他成员保留/重新引用；别人的未绑定件、跨宿主或跨用途都拒绝。只补合法 block ID 且内容标识不变才走 skip_activity 迁移分支；换图/换序必须保留动态、版本、时间。

草稿 payload 仍按 M4 保存有界 JSON 对象，不强制它在保存时满足 IssueCreate；description_html 是字符串时用同一个 sanitizer 的投影提取图片并验证/绑定，但保留原 payload 字节语义，不另造正文标准化器。错误类型、null、未知字段和不规范字段结构仍在保存时返回400；标题长度、枚举、关联对象、日期先后等业务值可保存，发布由原严格 decoder 和创建校验拒绝。创建新草稿不能挪用别的草稿图片；正常图片上传发生于已有 draft ID。

草稿改项目在原 W(S)→旧新P(S，升序)→D(U) 边界下迁移其全部活资产父链，包括 pending，使后续确认读取当前归属；迟到的旧外部操作用 currentPlacement 拒绝，候选仍可回收。改项目失败草稿/全部资产不变。发布见绑定子合同，不能先提交新 Issue 再搬附件。

### 2.4 精确事件/动态

真实增加/删除 ISSUE_ATTACHMENT 仅发 `asset.attachment_added` / `asset.attachment_deleted`；Entity={Kind:"asset",ID:assetID}，WorkspaceID/ProjectID/ActorID/OccurredAt为本事务事实；唯一 Change 为 Field="issue"、增加 NewID=issueID / 删除 OldID=issueID，其余值槽空。事件不带私有文件名、签名URL、key或正文。

issue 的同步 subscriber 在原事务写一条 attachment 动态并 MarkChanged；field="attachment"，verb="created"/"deleted"，新/旧 identifier=assetID，其余私有值槽为空。无账号/项目再加锁、无开启事务、无再投递相同资产事件。资料/描述/评论附件/草稿附件不冒充工作项附件动作；草稿发布迁移每个普通附件发一次 added，顺序UUID，时间和actor与创建一致。父级删除/回收无逐件用户动作，既有父删除事件原At/By不变。

通知/Webhook只记录具名事件与载荷交接至 M8；不在 P5 实现通知或新 CLI。

## 3. 证明与完成标准

每个重要性质有真实失败的定向变异；只编译失败不计 RED。API 正向数据经API，SQL限断言、可说明的故障/时间夹具。存储使用既有 local 与固定 SeaweedFS 实物，禁止假 signer/store 替代字节证明。锁测试按真实 PID/pgtest.RowWaits，不固定 sleep。

- [x] A1 内容身份/清洗/规范UUID/非法源/skip_activity/图片历史规则通过，源/生成合同无漂移。证据：[阶段记录§8、§12、§16、§18–19](../reviews/P5-issue-assets-review.md#20-t13验收与交接核对)。
- [x] A2 三类附件列表、绑定及已绑定跨成员读取/复制/删除按完整角色矩阵，资料图片和准备件不扩大。证据：[阶段记录§9–11、§17](../reviews/P5-issue-assets-review.md#20-t13验收与交接核对)。
- [x] A3 评论创建准备集合全校验同事务、无跨申请者/宿主误绑，回滚后可重试；分页有界。证据：[阶段记录§11、§13、§16](../reviews/P5-issue-assets-review.md#20-t13验收与交接核对)。
- [x] A4 草稿换项目/发布全资产、唯一发布、失败回滚、原时间/作者及迟到确认恢复得到真实交错证明。证据：[阶段记录§14、§17–19](../reviews/P5-issue-assets-review.md#20-t13验收与交接核对)。
- [x] A5 所有 Issue 出口真实附件计数，三类列表/两方向/跨页/分组/成员关系和游标精度；1000/1001及10000计数目录冷库容量证据。证据：[阶段记录§12、§15、§19](../reviews/P5-issue-assets-review.md#20-t13验收与交接核对)。
- [x] A6 事件/动态精确一次、rollback零行、父删除与60天清理/失败恢复/邻域，目录守卫保持严格。证据：[阶段记录§10、§18–19](../reviews/P5-issue-assets-review.md#20-t13验收与交接核对)。
- [x] A7 F8–F15的API/DB/字节边界及至少15个定向行为变异通过；本任务容器及时核验回收，原有容器保留。T9–T12聚焦故事、删除图与变异，加T13完整364故事复验及原20容器/新增遗留0，证据：[阶段记录§16–20](../reviews/P5-issue-assets-review.md#20-t13验收与交接核对)。
- [x] A8 最终本地七门禁、一次fresh deep-review与主控修复复核、准确最终HEAD三job CI全绿。证据：[阶段记录§21.2](../reviews/P5-issue-assets-review.md#212-最终预算复验与分支门禁闭环)。
- [ ] A9 --no-ff准确main CI全绿、全部原件归档hash/bytes/mode/membership/unknown=0并正常清理，P6预检准备完成。

## 4. 交接与代价

M4-closeout M5-01/02/03及M1/M2相关输入仍open，记录P5实际处理结果；P6/P7页面、P8整体容量/三浏览器/死成员和closeout不提前关闭。保留未知外部结果、失败重跑、缓存/skip说明与精确tested SHA；重复绿灯不能替换失败原件。
