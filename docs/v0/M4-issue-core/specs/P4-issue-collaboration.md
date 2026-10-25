# M4/P4 描述与协作：细化规格

| 项 | 内容 |
|---|---|
| 状态 | 独立预检及六项局部修订复核通过；A1已于2026-10-25获负责人批准；T1–T14实施及独立评审完成（T1含F1、T6含I1定向修复；T9两项Minor已在T10关闭，T13两项Minor已由T14复核关闭）；T15起后续任务和阶段验收待执行 |
| 日期 | 2026-10-25 |
| 代码基线 | `56d575fb8ff6575880d1edef65219728a81f2b12`；P3 已合并；P4 启动提交已把关键词 phase 改为 `M4/P4` |
| 工作树 | `.codex/worktress/codex/m4-p4`，分支 `codex/m4-p4`；主目录保持 `main` |
| 上级 | [M4 设计](../M4-design.md) §3.4–3.8、3.12–3.14、3.23、4、5、6.5、8.2、8.6、9、12/P4、14、15；D1–D8 和 §11 不重开 |
| 输入 | [交接入口](../../claude_to_gpt/README.md)及其规定顺序；P1b/P2/P3 spec §5、review §6；本规格 §7 的交接 |
| 执行计划 | [P4 plan](../plans/P4-issue-collaboration.md)；16 个 Task，顺序不同于设计任务清单，映射见 §3 |
| 证据边界 | 本文规定阶段契约，逐Task执行证据见review §2；T1–T14已通过独立评审，T9/T13两项Minor分别在T10/T14关闭，T15起仍待实现。附录 A 保留准备阶段记录，不代表整个P4通过 |

## 1. 目标

**A1已批准（2026-10-25）：** “清洗放到 issue/adapter/content，app 声明小接口，domain 保持纯净，不放宽架构守卫。同步修订设计、spec 和 plan 后继续 P4，完成后先别合并，我安排 Claude 做整分支终审。”据此清洗由 `adapter/content` 实现，`app.HTMLSanitizer` 声明端口，domain只保留纯值与规则，module.New组合；本规格与plan已按该决定细化，独立预检及修订复核已完成，T1实施、验证、独立Task评审与F1修正复核均完成，T2–T14实施及独立评审通过，T9/T13两项Minor分别在T10/T14关闭，T15起后续实现仍待执行。

完整交付设计 §12/P4：描述及清洗、内容标识与 `skip_activity`、提及的集合同步和自动关注、描述版本及其读取、评论的增删改与分页、工作项和评论的表情、动态的分页与历史引用过滤。增加 12 个操作；修改创建/修改/详情的描述契约；不增加回复、快照、事件或版本保留任务。

所有写复用 `Locks.run`：不加锁的资源读取及权限预检 → 项目写闸门 → 事务 → 项目 `FOR NO KEY UPDATE` → 重读/重新判定 → 项目归档/工作项归档检查 → 依赖检查 → 时钟 → 写。同一次改动的全部行同一时刻、同一个写者。子资源按自身 id 寻址时，隐藏和不存在返回同一份子资源 404；访客规则先于角色拒绝和作者拒绝。普通访客可评论、回应自己可见的工作项，不能读描述历史；工作区管理员兼项目访客仍受 `OwnOnly` 约束。

P4 的完成线是 I11–I13 接口故事、I1/I2/I15/I17 的增量断言、交错 14/15、S3 往返、逐操作矩阵、目录驱动删除和实际变异证据全部成立；完整门禁与 Claude 终审由实现阶段完成。本次细化不勾选这些条件。

## 2. 交付物

### 2.1 文件与职责

以下是计划文件，不是已产生的代码清单；`*_test.go` 与拥有实现的文件同 Task。路径均相对于仓库根目录，具体文件和生成命令见 plan。

| 范围 | 文件/目录 | 职责 |
|---|---|---|
| 领域 | `server/internal/modules/issue/domain/{rich_text,description,description_version,comment,reaction,collaboration_cursor}.go`；现有 `issue.go`、`patch.go`、`activity.go`、`actions.go`、`errors.go` | 内容标识纯值、评论有效性、版本窗口、表情格式和三个游标载荷 |
| 清洗适配器 | `server/internal/modules/issue/adapter/content/{sanitize,policy,projection}.go`及测试；`issue/app/content.go` | bluemonday与x/net/html只在适配器；app.HTMLSanitizer小接口，module.New组合（A1已批准） |
| 用例 | `server/internal/modules/issue/app/{comments,comment_writes,reactions,comment_reactions,description,description_activity,description_versions,activities,mentions,visible_projects}.go及各自小端口/写入值文件`；现有 create/update/lock/删除文件 | 小端口、同一加锁路径、订阅资格、描述写入协调、分页 |
| 存储 | `server/internal/modules/issue/adapter/postgres/{comments,reactions,comment_reactions,description,description_versions,mentions,activities}.go` 及对应 `queries/*.sql` | 只访问 issue 模块表，经事务 context/sqlc；分页先过滤再截断 |
| 迁移 | `server/migrations/sql/00021_issue_issue_comments.sql` 至 `00025_issue_issue_description_versions.sql` | 五表，设计 §4.7、4.9、4.10、4.12、4.14 原样；各在首个可用接口的 Task 建出 |
| 平台/成员目录 | `server/internal/shared/page.go`；`server/internal/platform/postgres/tx.go`；project 的 `app/issue_ports.go`、`adapter/postgres/issue_ports.go` 和 `queries/issue_ports.sql` | 共享页大小、只读快照、提及成员目录、历史项目目录 |
| 边界与组合 | `api/modules/issue.yaml`、`api/openapi.yaml` 的路径转引；issue `adapter/http`、`module.go`；`server/internal/bootstrap/{ports,app}.go` 和plan列出的评论/描述/活动、矩阵、`gate_test.go` 及 `issue_collaboration_*_test.go` | 闭合请求体、生成类型、错误码、唯一组合根、矩阵、逐写503与真实库测试 |
| 模式/授权/生成 | `server/sqlc.yaml`、`deploy/runtime-grants.sql`、`server/migrations/issue_schema_test.go`；`api/dist`、issue/project 的 `gen`、TS schema | 每张新表同时纳入授权/删除；生成物只由生成器产出 |
| S3 | `api/testdata/issue-richtext.json`；`web/packages/editor/src/sanitization-roundtrip.test.ts`；`server/internal/modules/issue/adapter/content/sanitize_test.go` | 同一个样例集由 Go 与真实 TipTap editor 消费，覆盖编辑器输出与 API 输入往返 |
| 故事 | `e2e/fixtures/{furnish.ts,comments.ts,assert/issue.ts}`；`e2e/stories/issue/i{1,2,11,12,13,15,17}-*.spec.ts` | 前置数据经接口；新表同Task补公共夹具与已有故事独立构造的目录行，不为新表增加目录断言豁免 |
| 文案/文档 | `web/apps/web/core/lib/error-messages.ts`、两份 `errors.json`；总体设计/M2设计/差异清单/相关交接/P4 review | 新码与实现同 Task；交接只写已证实的处理结果 |

### 2.2 接口与依赖约定

沿用标准库 `uuid.UUID`、`shared.Actor/Grant/TxManager`、现有 `Clock`、`IssueReader`、`ProjectMembers`、`stamp`、`LinkRows`、`ChangedRows`；不引入第二套事务或角色判断。新增 Go 依赖仅设计点名的 bluemonday 及其传递依赖（`x/net`、douceur、gorilla/css）；Task 1固定bluemonday v1.0.27与x/net v0.60.0以保持既有模块组，实施时核对实际解析与许可证；本规格不声称已安装。

对外用例遵循现有 `Execute` 风格；列表参数为 `ctx context.Context, issueID uuid.UUID, limit *int, cursor *string`，评论写参数为 `ctx context.Context, resourceID uuid.UUID, html string`，表情写参数为 `ctx context.Context, resourceID uuid.UUID, reaction string`。精确签名、消费端口和返回类型在plan的公共接口表与各Task。所有Page的NextCursor为空时在HTTP映射null，Data为空映射[]。

### 2.3 清洗和内容标识（Task 1；设计 §3.7、8.2，S3）

`app.HTMLSanitizer.Clean(string) domain.RichText`由`content.NewSanitizer()`构造的适配器实现，module.New每个模块构造一次并注入需要的用例；返回 `HTML`、`Text`、内容标识和评论是否有内容。无全局可变 policy。标签与逐标签属性照只读 Plane `apps/api/plane/utils/content_validator.py`，包括 mention/image 自定义元素、任务列表、表格、代码语言、块 id；URL 仅 `http/https/mailto/tel`，脚本、事件属性和危险 URL/CSS 去掉，不因清洗而拒绝请求。不能用一条“所有属性都放行”代替枚举。bluemonday内建URL校验不会自动覆盖自定义 `image-component[src]`，必须有显式校验与恶意src变异；callout的data-emoji-url会成为editor导出Markdown的img src，也复用唯一URL值检查/规整并测试危险scheme、空白/控制、引号/实体及安全https保留。不能把允许属性当成协议校验。显式区分可保留的 style 属性与其安全 CSS 值，S3 用实际 editor 输出证明。

纯文本由清洗后 HTML 经 StrictPolicy、实体解码和空白规整得到。内容标识是纯文本、按出现顺序（保留重复）的 user mention UUID、`a[href]`、每个 `input` 的 checked 布尔；集合去重只发生在关系写入，不发生在内容标识。非 UUID/非 user mention 不参与成员同步。`image-component` 在放行表，但图片 src 的内容标识等 M5，见 §5。

S3 以真实 `CoreEditorExtensions` 生成每种保留节点/mark 的 HTML，不手写一个“看起来像 editor”的替身。块 id 采用固定测试值。API HTML 至少有无块 id 的列表、标题、嵌套 mark；样例保留 raw、sanitized、editorReserialized、再次 sanitized 及预期内容标识。只改链接、只改 checked 的两个样例必须判为普通修改。恶意输入与 Unicode/实体/空白分开测；样例未实际往返前不得填“已通过”。固定sanitized期望并检查保留节点/属性，CSS分号等合法规范化不当作格式丢失，宽松DOM比较也不能掩盖属性丢失；真实schema的node/mark增长必须触发样例覆盖检查。callout既有Markdown出口对img src/alt与icon正文作局部上下文转义，优先复用已有可访问的能力，否则仅在extension-config.ts内添加私有小函数，不新增通用包/依赖；URL语义仍由清洗适配器校验。跨语言夹具含data-emoji-unicode编码引号/属性与data-icon-name编码闭标签/标签，sanitized HTML进入真实editor后调用实际Markdown serialize，断言不能多出属性/标签且保留合法emoji/icon；仅getHTML稳定不算该出口安全证据。

S3实际执行时，允许局部修复已有CustomColor解析：仅匹配自己的非空颜色属性，优先于通用TextStyle解析带style的颜色span；真实emoji/颜色/背景往返及独立变异证明不再抢占或丢属性。不改第三方库。当前无任何持久属性的空TextStyle单列为原生无语义规范化，精确断言空schema属性、文字保留和空mark消失，仍覆盖该schema项；未来新增属性会使此断言失败，不把有意义的格式丢失归入此例。

已知颜色data属性和表格background/textcolor会被编辑器重新解释为style，适配器须对这些值执行安全颜色校验，保留实际调色板键与合法颜色，拒绝分号/URL注入；实际共享样例覆盖Go清洗后再由editor生成HTML的边界。清洗已有style不能替代该检查，也不能为往返样例放宽CSS策略。

### 2.4 页大小、载荷与读快照（Task 2）

页大小从 identity 移至 `shared.PageSize(*int) (int,error)`，默认 50，范围 1–100；越界 422 `validation_failed`/`limit`/`out_of_range`。删去 identity 的重复实现，PAT 列表行为不变。`CommentCursor`、`ActivityCursor` 是 `[created_at,id]`，`DescriptionVersionCursor` 是 `[last_saved_at,id]`，各有严格 JSON marshal/unmarshal；UTC、非零 UUID、精确两元素，复用 `shared.EncodeCursor/DecodeCursor` 封套。坏形状 400 `bad_request`，同时坏 limit 时 400 优先；合法他人位置不签名、不当作权限证明。

`postgres.ReadOnlySnapshot() Option` 与 `LockTimeout(d)` 可任意顺序组合；最终 BEGIN 同时包含 `ISOLATION LEVEL REPEATABLE READ READ ONLY` 和 `SET LOCAL lock_timeout`。列表的首次目标读取/判定、project 目录、issue 查询和评论表情批读都在独立的 `ReadTx.WithinTx` 中；嵌套既有写事务不会提高隔离级别，不能当作读快照。原有写事务保持原语义；没有模块 SQL `SET`。目录与查询快照一致、禁止写入和事务后配置不泄漏均需真实库证明。

### 2.5 评论创建和列表（Task 3；迁移 00021）

创建/列表路径为 `/api/v0/issues/{issue_id}/comments`；创建 201 `IssueComment`，列表 200 `IssueCommentPage`，默认升序 `(created_at,id)`。请求 `comment_html` 必填、非 null、≤500000 个字符；清洗后无文字且无 mention/image 节点，422 `comment_html/required`。对象只含 `id,issue_id,actor_id,comment_html,edited_at,created_at,updated_at,reactions`；不返回 stripped、不接受 access/parent_id/comment_json/attachments/description_id。

评论者与被提及且能读它的人自动关注。在已有 `ProjectMembers` 职责内增加窄方法 `MentionMembers(ctx,projectID,ids) ([]MemberVisibility,error)`，`MemberVisibility{ID uuid.UUID, Role shared.Role, OwnOnly bool}`；project 同一批读成员与 guest 标志，bootstrap 转换，不改变 `ProjectGate.Lock`。所有有效被提及者可参与资格计算，只有 `Issue.HiddenFrom(userID, OwnOnly)` 为假的能关注；管理员兼项目访客也照其项目角色。评论提及不进 `issue_mentions`。插入关注按账户 id 升序、部分唯一冲突忽略。创建写顺序为 `issues` 的 MarkChanged → `issue_subscribers` → `issue_comments`，无动态。

迁移同 Task 接 `IssueRowsDeleter`、`ProjectRowsDeleter`、两个 step 列表、sqlc、grants、存储/组合目录测试和 `furnishIssueRows`。评论表情 schema 可在本 Task 定义，因为空集合已是实际评论数据；Task 6 在有表/操作时填入真实值，期间没有表的删除豁免。

### 2.6 评论编辑与删除（Task 4）

`PATCH/DELETE /api/v0/issue-comments/{comment_id}`；更新 200 评论，删除 204。按设计使用`issue_comment.write`动作（project级Admin/Member/Guest），再相对收窄为只有作者编辑；作者或 `Grant.ProjectAdmin` 删除。违反相对权限答 `issue.not_author`，权限预检与锁下复核均执行。作者由评论 `actor_id` 判断，不是工作项创建者。子资源/工作项隐藏与不存在统一 `issue.comment_not_found`，404 在作者 403 前。

新旧清洗后 HTML 相同时不写任何行、不设 `edited_at`；不同则 `edited_at=updated_at=now`，工作项同步 MarkChanged，新的可见提及者关注，旧关注不取消。删除保留已删行原审计，连带仍活的 comment_reactions（Task 6 补接），不触及邻居评论；不写动态。实际写表序：issues → subscribers（编辑需要时）→ comments → comment_reactions（删除需要时）。归档项目先于归档工作项，归档拒绝先于无改动。

### 2.7 工作项表情（Task 5；迁移 00022）

GET/POST `/api/v0/issues/{issue_id}/reactions`；DELETE 同路径加 `/{reaction}`。值 `^[0-9]{1,7}(-[0-9]{1,7}){0,15}$` 且 ≤128 字符，接口和数据库同一约束；不是 Unicode Emoji 名录校验。重复自己的同值 409 `issue.reaction_exists`；删除只找自己的活行，无该行 404 `issue.reaction_not_found`，不能拿其他 actor 的行顶替。集合 `(created_at,id)` 升序，不分页；HTTP为 `IssueReactionCollection={data:[…]}`，空集合 `{data:[]}`，用例slice由mapper封装。加/去成功都先 MarkChanged，再写 reaction，无动态、不自动关注。迁移同时扩展删除与 furnish。

### 2.8 评论表情（Task 6；迁移 00023）

POST/DELETE `/api/v0/issue-comments/{comment_id}/reactions[/{reaction}]`；与工作项表情同格式、重复和只删自己规则。目标隐藏/不存在答 `issue.comment_not_found`。不改工作项、评论的 updated_at/updated_by_id/edited_at；没有独立列表，在评论分页中批量读取对应页的活 reaction，按 `(created_at,id)` 排序。T6同时扩T4的CommentUpdater与module接线/假实现：复用CommentReactions窄读接口，在当前写事务内组装更新后及同HTML无改动的完整单评论应答；两种PATCH都保留多个actor的活表情，软删不出现。列表仍批读不N+1，创建新评论可确定返回[]。删除评论连带其全部 actor 的活表情；该删除与工作项/项目/工作区删除保持表序、同一时刻/删除者。迁移同时扩展目录断言和 furnish，不跨 issue 模块 JOIN。

### 2.9 描述模型及现有列读写（Task 7）

`domain.NewIssue`/`IssuePatch`/`Issue` 增加描述信息；创建默认 `<p></p>`；长度按 Unicode 字符而非字节 ≤500000，请求体仍限 1 MiB。`IssuePatch.SkipActivity *bool` 保留字段是否出现；只要出现（包括 false），就必须同时有 description_html 且无其他 patch 字段，否则 422 `skip_activity/not_allowed`。纯 HTML 相同且无 skip 的普通保存不算修改；skip=true 的保存总写清洗后 HTML；skip=false 是普通保存，清洗后HTML相同则不写（Write=false、Changed=false）。

Task 7 同时闭合app/create_issue、app/update_issue和postgres/update的值传递：创建IssueRow显式默认HTML为`<p></p>`、stripped指向空字符串（非nil），普通PATCH的IssueFields从锁下旧Issue透传两列。`description_stripped`沿既有数据库可空列用`*string`映射，保留旧NULL与明确空字符串的区别；不做COALESCE，不改迁移。存储/组合回归验证旧API默认、非默认文本、NULL和空字符串分别保持，邻居不变。Task 7 只准备内部类型、已有两列存储与规则测试，HTTP 字段在 Task 9 连同提及/版本一起出现。给后来消费者明确接口，不能把尚未接上的内部行为标作已交付接口。`GetIssue/GetIssues` 和子项/关联读取的映射一致；Task 9 后全部详情回答仍经 `seen`，列表 `Issue` 不含 description_html。

### 2.10 描述动态和版本合并决策（Task 8）

普通描述改动写动态表第 4 行；仅当工作项最新活动态（`created_at DESC,id DESC`）就是同 actor 的 description 行时原地合并，保留 old_value、id、created_by_id，更新 new_value、created_at、updated_at、updated_by_id。不是“找到最近一条 description 就合并”；不同 actor、其间其他字段动态均中断。一个 PATCH 同时改标题与描述时按设计表顺序先写标题行，再判描述合并，故新标题行会中断合并；其余字段随后写。两条业务规则分别测试，不把描述行优先写以迁就合并。

版本决策 `CanMergeDescriptionVersion(latest,actor,now)` 与动态合并独立：最新活版本按 `(last_saved_at,id) DESC`；同 owned_by_id 且 `0 <= now-last_saved_at <= 600s` 合并，窗口随每次保存滑动。600 秒恰好合并，600 秒加 1 微秒新建；更换作者新建。创建总有初版；普通描述修改后保存新内容。此 Task 实现已有动态表查询/合并以及版本纯决策，版本表和其实际写入在 Task 9。

### 2.11 描述公开、提及与版本原子写入（Task 9；迁移 00024–00025）

`IssueCreate/IssueUpdate/IssueDetail` 加 description_html，update 加 skip_activity。`IssueDetail` 显式为 `Issue` 加一个字段，不用 allOf；修掉 P3 留的 `gen.Issue(issueDetail(i))` 同形转换，apitest 比较两个对象的字段和 required 集合。创建/修改/按编号/归档/恢复等详情都返回保存值；子项与关联的列表行不含描述；seen 仍遮蔽父 id、按可见性数子项。

创建与每次描述被写入（含还原、skip 内容不变）均从清洗后保存值提取 user mention UUID，过滤有效项目成员、去重并排序，做 issue_mentions 差集；保留行不重写，移除行软删，新增行新 UUIDv7。只改状态/标题不触及提及。普通保存（含skip=false）净化后HTML相同且其他字段无变化时不写，也不查询提及目录或补关系；skip=true总是写入，所以仍同步资格变化。被提及且可见者与既有创建者/实际负责人关注规则合并，去掉提及/负责人不取消关注。描述修改者不会仅因编辑而自动关注；评论者自动关注仍按评论规则。skip 内容不变也可以因成员资格变化同步关系行，但不改变工作项审计、不写动态/版本；关系行使用本请求锁下时刻。

内容标识不变且 skip=true：只保存清洗后 HTML/stripped 及上述关系同步，不改工作项 updated_at/updated_by_id。标识变化即普通修改（即使 flag=true），链接/checked 不能隐藏。skip=false 是普通保存。纯格式改动被 skip=true 抑制是已批准代价。P6 才实现事件；本 Phase 不加入空事件端口。

版本插入/合并同一事务；新行 owned_by_id、created_by_id、updated_by_id 都是写入者；合并保留初建时间和创建者，更新 HTML/stripped/last_saved_at/updated_at/updated_by_id。不立即截到20行（P9）。写表序：issues → assignees → labels → subscribers → mentions → activities → description_versions；所有描述/版本/提及失败都回滚主行与其他副作用。两表的 grants/删除/目录夹具同 Task 完成，`furnishIssueRows` 经描述创建或 PATCH 给每个项目造提及和版本。

### 2.12 描述版本读取（Task 10）

GET `/api/v0/issues/{issue_id}/description-versions`，降序 `(last_saved_at,id)`，Page 的每项 `{id,issue_id,owned_by_id,last_saved_at,created_at}`；GET `/api/v0/issue-description-versions/{version_id}` 加 description_html。规则 `issue_description_version.read` 为管理员/成员，仍先套工作项访客规则；单版本隐藏/不存在/已删统一 `issue.description_version_not_found`。归档资源可读。还原没有专用路由，客户端拿旧 HTML 经普通 PATCH；遵循普通描述改动、600秒合并和归档拒绝规则。

### 2.13 历史成员目录（Task 11）

在 issue/app 定义 `VisibleProjects` 的已用方法 `Historical(ctx, workspaceID, actorID uuid.UUID) ([]VisibleProject,error)`，`VisibleProject{ID uuid.UUID, Role shared.Role, OwnOnly bool}`。project 的实现只访问自己的 projects/project_members，在 workspace 范围读取 `is_active=true` 的成员关系；不看项目/成员 `deleted_at`，也不看归档，OwnOnly 按项目角色和 guest_view_all_features 计算。P5 再加普通活项目目录方法，不提前造无消费者的方法。

调用者读取目标工作项的常规授权先在相同快照中通过；历史目录只是过滤引用，不放宽目标工作项权限。被移出/主动离开造成 is_active=false 时引用不可读；删除项目只是软删除成员关系而保留 is_active 时历史可读。不存在的目录项默认不可见，端口错误原样返回，不能当成空目录。

### 2.14 动态列表（Task 12）

GET `/api/v0/issues/{issue_id}/activities`，`(created_at,id)` 升序，limit/cursor 同评论；对象字段严格照设计 §5.2，不给 comment、嵌入 detail、WORKLOG 类别。引用字段只对 parent、relates_to、duplicate、blocked_by、blocking 应用额外规则：两个非空 identifier 各自都须找到被引用 issue 及其历史项目目录，且通过该项目 OwnOnly；一个不可见则整行略去，连 old/new_value 的编号都不泄漏。引用 issue 的 deleted_at/archived_at 不过滤，已物理清理则略去；其他字段的标识是状态/标签/账户，不能错当工作项处理。

过滤写在 issue SQL 的 LIMIT/游标选择之前；引用只 JOIN issue 自己的表，项目目录通过参数传入。取 limit+1，next_cursor 指最后一条实际返回的可见行，不来自被滤行。最后一页 next_cursor=null；客户端保留产生最后一页的请求游标，刷新该页并按 id 合并，描述合并移动 created_at 会重新出现。其他客户端编辑/删除旧评论要整体重读才能更新，这是设计限制。

### 2.15 并发与权限（Task 13–14）

交错14覆盖发评论/删工作项两种提交顺序；评论先则连带同删除时刻，删除先则404且无关注/评论遗留。交错15同 actor 两次描述保存，后者锁下看到前者，动态/版本各合并且最终内容正确、old_value 不丢；时钟在最后一把锁后读取，created_at 不减。增加不同 actor、被移出/降访客、评论编辑/删除与表情、评论写与归档的判别数据；不会用固定 sleep 驱动先后。

扩展现有 `issueWrites`/取锁形状/闸门/事务连接测试到每个新写；矩阵每个新操作一行，作者/非作者、管理员兼访客的自己/别人、已离开、已删除、归档两种前置都要有。隐藏与不存在比较状态、Problem 全体和稳定响应头（请求 id 等非资源相关动态头先按现有帮手规整），不只比较 code。新增码同 Task 带中英文文案和 HTTP 假实现双向契约覆盖；503 只在 bootstrap/gate 映射。

### 2.16 故事与收尾文档（Task 15–16）

I11 评论与两类表情；I12 评论/动态翻页与增量新增；I13 历史读取、600秒窗口和普通 PATCH 还原。I1 加默认/显式描述、过滤后的提及和创建版本；I2 加描述动态/版本合并、skip 块 id、只改 href/checked 的审计；I15 已归档评论409；I17 父/两层子树的五表数据同删除者/时刻且邻居数据保留。所有故事前置用 API；不能用 SQL 操纵时间跨600秒，接口故事通过换作者制造新版本，边界600秒在固定时钟组合测试证明。

同步设计 §3.23/P4 各行、逐列差异、五份交接及 P4 review。review 保留 P3 §1–7 结构，所有通过数字、变异、CI 和交错证据在实际运行后记录。

## 3. 与设计的差异和补充

**A1（负责人已批准，2026-10-25）：清洗在适配器层。** 原设计点名domain/sanitize与纯层守卫冲突；最终按负责人原话采用`issue/adapter/content`、`app.HTMLSanitizer`、纯domain，保留`server/internal/archtest/rules_test.go`和`purity_test.go`的直接/间接依赖检查。基线提交573e8eb6记录了暂停，恢复时先同步文档；T1的实现/验证进度见附录与review，不代表阶段验收通过。

以下局部工程裁定已完成独立预检与修订复核；新增端口和后续协调实现仍待验证；T1验证单独记录。A1之外不改变负责人已批准的产品/架构决定。

1. **裁定：不做整段原型/逐字代码重放，采用接口、文件、测试、命令驱动的紧凑 plan。** 依据交接02 §3，这两项属于可调整做法。保留独立预检、TDD、逐Task独立评审、关键性质实际变异与负责人安排的Claude终审；代价是没有原型快照可逐字比较，执行中必须把接口变化同步回plan，不能把未运行预期当结果。
2. **裁定：迁移按首个可用API分批交付。** T3=00021，T5=00022，T6=00023，T9=00024–00025；每批同Task扩展删除及接口fixture，避免目录豁免。设计任务映射：原1→T1；原2→T3/5/6/9；原3→T2；原4→T7/8/9；原5/6→T9（政策准备T1/3/8）；原7→T10；原8→T3；原9→T4/6；原10→T5；原11→T6；原12→T11/12；原13→T13；原14→T14并随每个API长矩阵；原15→T15并随迁移长fixture；原16→T16。T7/8内部准备由T9消费，终态不得留无使用者。
3. **裁定：只读快照接缝从P5提前到P4/T2。** P4分页目录/查询同快照是设计3.4的要求；保留设计名`ReadOnlySnapshot()`，同时组合lock_timeout，避免P1b交接指出的BeginQuery盖过TxOptions。代价是提前触及平台/组合根，P5复用并删重复实现步骤。
4. **裁定：子资源相对权限纳入同一个Locks.run的两次检查。** 增加可选小回调，见plan T4；调用者无权编辑别人的评论在进闸门前就403，锁下重读后再判。`seeAs`让子资源404在源头统一，不靠HTTP层翻译所有错误。只变更同一加锁机制，不复制它。
5. **裁定：在既有ProjectMembers目录内补齐提及者可见性。** 现有Roles缺被提及者OwnOnly，调用者的Grant不能代表他；逐候选Authorizer需要代构Actor且重复读取相同项目，使用已有目录职责的窄方法批读成员角色/OwnOnly更合适。project读自己的表、bootstrap转值，issue用HiddenFrom统一过滤关注，不增顶层目录、不改变Lock值；代价是要测转换和端口失败。
6. **裁定：skip_activity出现与值分开。** false仍受“只能与描述一起、不能带其他字段”约束，只有true且标识相同才抑制审计。保存仍同步提及，确保“每次描述被写入”成立；普通相同HTML不写。设计未钉死的版本同刻选取以id降序确定，窗口600秒含端点，墙钟倒退不合并；墙钟回拨的时间线全局保证仍不在范围内。
7. **裁定：同一PATCH动态按表顺序插入/合并。** 新标题动态先于description合并检查，避免跨过刚发生的其他字段改动。动态不带时间窗口，版本有600秒滑动窗口，两者不得共用一个“最近修改”条件。
8. **裁定：分页操作显式声明400/422。** 设计5.1主要错误表未逐行列坏cursor/limit，沿identity既有契约补全；删除reaction路径值同格式检查，非法格式422，先权限预检后资源相对不存在检查，避免暴露别人表情。
9. **裁定：不宣称D2已实测满足。** 附录记录旧plan实际大小；本plan写完量文档大小，不拿短文档等同实现复杂度。T3/T9为高密度任务，独立预检若无法一次评审可先重切；仍超约16任务或约1500行plan则按D2拆P4a协作与P4b描述/动态，范围不缩水。

## 4. 验收标准

- [ ] S3覆盖保留节点与API输入，去除危险HTML，skip的文字/提及/href/checked四个维度有可失败断言。
- [ ] 五表结构、grants、sqlc scope、删除图及无豁免的目录fixture通过；删除子树/项目/工作区保留旧删除戳和邻居行。
- [ ] 描述与其提及、订阅、动态、版本同事务进入API；所有详情仍经seen；600秒边界和回滚通过。
- [ ] 12个新操作、分页、错误码和中英文文案、矩阵、隐藏逐值相同、作者规则与归档次序通过。
- [ ] 三种游标篡改只改变合法可见结果起点；动态历史引用软删/项目删除/is_active=false/物理清理各例通过；同一只读快照成立。
- [ ] 交错14/15与新写锁形状/闸门/事务连接测试通过；交错用`-race -count=5`无死锁。
- [ ] I11–I13及I1/I2/I15/I17增量全部通过；前置通过API。
- [ ] 关键安全、锁、游标、删除、合并性质实际变异被业务断言发现，构建失败/基础设施失败不算；只单元层覆盖的安全性质有明确补证或风险裁定。
- [ ] `make gen`、提交后`make gen-check`、`make lint-go`、`make test`、`make lint-web`、`make knip`、`make test-web`、`make e2e`及分支CI完成；main CI在负责人批准合并后记录，不提前要求合并；本机缺Firefox/已有W11偶发如实分类。
- [ ] 独立预检、逐Task评审、Claude终审和修复轮有记录；相关上级文档/差异/交接已同步；没有未经证实的通过项。

## 5. 不在P4范围内

| 接收方 | 移交 |
|---|---|
| P5 | 普通VisibleProjects方法、列表引擎、force_custom_plan；复用已提前的PageSize/ReadOnlySnapshot，勿重复实现；替换P3 seeEach/ArchivedProjects仍归P5 |
| P6 | 描述/评论/表情的领域事件与任务；skip=true且内容相同不发布事件；每次写的事件共用当前锁下时刻；不得由P4加空事件端口 |
| P9 | 每工作项最新20个版本的保留、60天物理清理、项目/工作区异步删除；P4只完成同步连带与批量语句；新表与comment FK都纳入后续清理 |
| P11 | service/store换型、description_json/comment_json删除；版本新地址、评论内嵌表情、分页契约 |
| P14 | 描述保存撤销再重做、创建/编辑弹窗isMigrationUpdate、只在用户改父项时发parent_id；I1/I2/I15/I17页面版本 |
| P15 | I11–I13页面；作者编辑/管理员删除入口；按id合并并保留最后页请求cursor；别端旧评论编辑/删除须整体重读 |
| M5（经收尾） | 图片功能打开时把按顺序image-component[src]纳入内容标识并扩充S3；保留放行标签不等于本Phase开启上传 |
| M7/M8（经收尾） | 关注只在加入瞬间可读，通知发送前再判权限；评论/表情没有动态，通知从事件；stripped供摘要/Webhook，不作搜索 |
| 收尾 | 八份交接统一closed、main CI及W11遗留按交接06；浏览器C12、整体性能测量；本Phase不修无关故事或安装全局浏览器 |

## 6. 风险

| 风险 | 判断和退路 |
|---|---|
| sanitizer与editor roundtrip不一致 | T1先实际S3再继续；不放宽危险属性换“通过”，记录差异和最小支持策略；产品格式损失无法局部解决则上报 |
| T9聚合写过大 | T7/8先固定领域/已有表接口；预检按独立评审边界审查，必要时D2拆段，不能删提及/版本或临时开放半成品描述 |
| 作者预检造成第二条写路径 | 只扩展Locks.run/seeAs；原写回归及新拒绝不占闸门由组合测试证明 |
| 历史过滤混用活目录 | 目录方法明确Historical；SQL先过滤后分页；已删除项目/未删除兄弟项目同时造数据 |
| 时间窗口/排序“假通过” | 固定时钟、同时间逆UUID、601秒和600秒边界、交替作者/其他字段；不靠自然id总升序或sleep |
| 存储读写不在同一事务 | context端口错误原样传播；真实回滚与不同连接探针，ReadOnlySnapshot组合参数读回 |
| 环境造成假红或假绿 | 执行中一次一个套件，记录基础设施失败，不算变异杀死；Firefox缺失不能写成e2e全通过 |

## 7. 已知限制、交接和关闭条件

| 输入 | P4处理 | 关闭证据（待执行） |
|---|---|---|
| P1b spec§5/review§6 | description/skip、第4行动态、I2 | T7–9/15 |
| P2 spec§5/review§6 | 评论归档409、全部子资源访客规则、版本地址、无access | T3–6/10/12/14/15 |
| P3 spec§5/review§6 | 历史引用过滤、Issue/Detail分离和seen、锁后时钟 | T9/11–14 |
| [M1-P2](../handoffs/M1-P2-trim-content.md) | 服务端skip语义以已批准3.7为准，旧字符串true不兼容 | T1/7/9/15；前端撤销再重做留P14 |
| [M1-P3](../handoffs/M1-P3-trim-platform.md) | 无access/WORKLOG等、描述历史新路径 | T3/10/12；旧前端地址留P11 |
| [M1-closeout](../handoffs/M1-closeout.md) | 服务端不定义comment_json | T3/4；前端JSONContent与comment_json一起删除留P11 |
| [M2-closeout](../handoffs/M2-closeout.md) §2/3/4/6 | 三种游标、锁后时刻、新表grants；本Phase无开放map/递归请求schema | T2/3–15；任务/保留等按原分配继续 |
| [M3-closeout](../handoffs/M3-closeout.md) §2/3/4/9 | 表序/矩阵/目录删除/共享PageSize | T2–14 |

交接只追加“处理结果（M4/P4）”与仍未处理项，status保持open，到M4收尾统一关闭。创建描述初版只覆盖P4起创建的工作项；不凭空给历史工作项伪造作者/创建时刻，已有行首次实际改描述再产生版本（局部迁移行为，T9应测试）。当前main第340次CI的W11偶发仍按交接06处理，不作为P4已通过证据。

## 附录A：验证记录与Phase大小（2026-10-25）

### A.1 已做的只读工作

已按入口顺序读取交接、设计点名章节和P1b/P2/P3交接；检查Locks、create/update/read/delete、project端口、identity分页、TxManager、契约/矩阵/fixture及Plane清洗/版本/评论源码。只读未在plane/refer执行git。准备阶段只编写spec/plan；没有另做原型实现或任务重放，后续Task按正规TDD实施。

旧plan按当前基线文本从`### Task N`到下一标题/文件末尾实测（不是旧文档附录的早期数字）：P1a 12,413行/15Task/最大1,438；P1b 9,757/15/1,619；P2 12,727/14/1,633；P3 19,914/16/1,861。旧plan包含全量代码转录，不能将其行数直接当新紧凑plan实现工作量。16Task是切分目标；T3、T9预检重点评估，修复余量按5–10%考虑。

### A.2 执行证据进度

| 项 | 状态 | 实现阶段记录位置 |
|---|---|---|
| S3真实editor往返/恶意HTML | T1已验证并通过独立评审与F1复核 | Task1日志、样例来源/版本、review §2.2 |
| 单元/存储/组合/契约/故事门禁 | T1–T14适用门禁及独立评审已完成；最近适用的T9端到端首轮125通过，I17夹具失败已定向修复验证，两项Firefox缺失仍在 | 每Task日志与提交SHA，review §2保留真实失败与修正，不将组合结果称为一轮全绿 |
| 交错14/15、锁形状、快照一致性 | T13交错/race及独立评审通过；T14安全/快照/真实回滚、完整Go及独立评审通过 | 实际项目锁等待后改变时钟/权限；-race -count=5通过，无DATA RACE或40P01 |
| 关键变异 | T1–T14共101项实际业务变异已执行并独立评审，故事性质待执行 | 分项为15/5/9/5/9/8/8/5/10/6/4/4/3/10（T14九类性质十次实验）；变异定义、业务红断言、恢复绿及无效尝试均留证，不称全面杀死率 |
| 独立预检与Claude终审 | 预检及修订复核完成；Claude终审待实施完成后由负责人安排 | review§1/3/4及分诊台账 |
| 全量CI与合并清理 | CI待执行；合并/清理等待负责人另行批准 | 实际run链接及结论 |

### A.3 变异最低集合

至少实际移除/改坏：URL协议清洗；on*属性清洗；href/checked内容标识各一；skip更新审计；600秒含端点和滑动；只合并最新动态；有效成员过滤；隐藏guest不关注；提及差集删除；作者预检；管理员只删不改；子资源404映射；仅删自己reaction；comment reaction不碰issue审计；历史is_active过滤；历史不看deleted_at；任一隐藏引用整行过滤；过滤在limit前；游标不能选定权限范围；项目N/闸门；时钟移到锁前；新表删除范围/已删除行保留；快照隔离组合。具体所属测试与层在plan各Task，不宣称已被杀死。
