# M4/P4 描述与协作实施计划

> **执行方式：** 按 Task 使用 superpowers:subagent-driven-development 或 superpowers:executing-plans；执行由 Codex 按既定方法推进，以复选框记录真实进度；P4 本轮终审按负责人新指示执行 deep-review，修复后按原合并/main CI/清理流程启动 P5。

**目标：** 完整交付M4设计§12/P4的描述、清洗、提及、版本、评论、两类表情、动态分页及对应测试/交接。

**架构：** A1已获负责人2026-10-25批准：清洗实现位于issue/adapter/content，app声明HTMLSanitizer小接口，domain保持纯净，不放宽架构守卫。所有写复用Locks.run，目录/SQL经同一只读快照，跨模块值只在bootstrap转换。原先等待外部Claude的门由负责人本轮deep-review指示替代，审查与修复通过后合并，见[本轮整分支 deep-review](../../../reviews/re-01-M4P4整体审核.md)。

**技术栈：** Go 1.27/toolchain go1.27.1、pgx v5.11.0、sqlc v1.31.1、oapi-codegen v2.8.0、PostgreSQL 18.6、Node 24、pnpm 11.10.0和仓库锁定的TipTap/vitest/Playwright；仅新增已批准bluemonday及设计点名的传递依赖。

**规格：** [P4规格](../specs/P4-issue-collaboration.md)；[上级设计](../M4-design.md)。代码基线56d575fb8ff6575880d1edef65219728a81f2b12，恢复文档HEAD 573e8eb6。本plan为紧凑可执行计划，不转录整份实现；没有整段原型或其生成SHA；实际Task验证进度见review；T1–T16独立评审通过，T15两项Minor已由T16关闭；首次分支CI的S2动画残留已修复；A2修复54866c8f已通过独立评审（C/I/M均0）及分支CI 40175286547（server/web/e2e全部成功）；系统git已通过SSH实际推送54866c8f，本轮deep-review与修复复审已完成，见[本轮整分支 deep-review](../../../reviews/re-01-M4P4整体审核.md)。

## Global Constraints

- 工作树`.codex/worktress/codex/m4-p4`、分支`codex/m4-p4`；主目录保持main。不用stash/clean/reset --hard，不动plane/refer、不在其中git，不操作已有容器，不全局安装。
- 启动提交已把keywords phase改为M4/P4，不重复。按当前会话较新的AGENTS指令，后续提交保留`feat(M4/P4):`等前缀、说明使用中文，必要时多行解释背景/改动/原因；已有英文提交不重写；push用`/usr/bin/git`。草案作者不提交，由主任务统一核对并保存准备工作。
- TDD、关键性质实际变异、独立预检、逐Task独立评审保留；本轮整分支deep-review按负责人新指示执行。接口/文件/测试/命令驱动的紧凑plan替代原型和代码逐字重放（交接02§3允许），不得伪造通过数字、哈希、生成物或杀死率。
- 预检404/403不取事务/锁/闸门；Locks.run下再读/再判，访客404先于角色/作者403；项目归档→工作项归档→无改动→依赖检查→时钟→写。新写声明503，只有bootstrap/gate映射server_busy。
- 全局表序：issues→assignees→labels→relations→links→subscribers→mentions→comments→issue_reactions→comment_reactions→activities→description_versions。同次改动同一锁下时刻、同一写者；业务描述里的“先评论后updated_at”不意味着SQL逆序。
- 角色按集合、可见性复用HiddenFrom；模块不互相import、不跨模块JOIN、SQL只经sqlc、模块SQL不写SET；端口错误原样传播，不留无消费者代码。新逻辑文件约400行以内，契约/生成物沿既有例外；`api/testdata/issue-richtext.json`作为单个Go/TS共用的固定数据契约另列规模例外，保留完整editor文档与节点/属性证据及可读排版。
- 新错误码同Task更新PROBLEM_MESSAGES和en/zh-CN errors.json并跑make test-web；每个新API当Task加入矩阵、HTTP双向契约、锁形状/闸门测试。
- 新表随首个可用API建出，同时grants/sqlc、删除连带、目录断言及furnishIssueRows；不新增临时目录豁免。迁移顺序00021→00022→00023→00024–25保持原编号。
- 实现阶段生成物只经make gen/gen-go，提交后make gen-check；server改动跑lint-go/test，契约/web/e2e改动加lint-web/knip/test-web，迁移/fixture/故事跑e2e；一次只跑一套容器套件。实际已按适用范围执行；T16无Go改动复用T14完整Go与T13 race证据，时间故事变化顺序执行Web G/e2e及最终提交后gen-check。
- 测试复用issueWorld、activityBy、workItemStamps、listIn、matrixRow.prepare、itemWrite、inBothOrdersAt、pgtest.RowWaits；e2e复用activityRow/byId/madeBy/tablesUnder/keysBelow。等待有条件、有期限，不sleep同步；逆id、同时间、两方向、两层子树、邻居/已删行让数据区分性质。


## Review Focus

1. 自定义image-component[src]不受bluemonday固定标签URL策略自动覆盖；style属性放行不等于CSS安全。T1需要真实往返及恶意src/CSS变异。
2. 管理员兼项目访客对自己的工作项可删别人评论，别人的工作项仍404；关注按被提及者自己的OwnOnly算。T3/T4/T14。
3. 动态合并“最新一行”与版本600秒滑动窗口是两套规则；同时标题/描述、不同actor、同时间逆id、600秒端点必须区分。T8/T9/T13。
4. 历史目录忽略deleted_at但必须保留is_active；过滤在LIMIT前，隐藏引用不能影响页长/cursor。T11/T12。
5. 新表与嵌入表情扩展删除图；邻居评论/子树、旧删除戳、目录与查询同快照均需要反例。T2–6/T9/T14。


---

## 文件结构、共用步骤与接口

文中的文件路径相对工作树根；仅表格使用缩写：I=`server/internal/modules/issue`，P=`server/internal/modules/project`，B=`server/internal/bootstrap`。各Task的Go包命令用完整路径。新增方法/测试名在实施时创建；目标命令必须确认有匹配测试，零测试不是通过。

| Task | 交付/主文件 | 消费前置 |
|---|---|---|
| 1 | content适配器、纯RichText、S3 editor样例 | — |
| 2 | shared/page、纯游标、ReadOnlySnapshot | — |
| 3 | comments、ProjectMembers扩展、00021、删除/furnish | 1、2 |
| 4 | comment_writes、Locks子资源检查 | 3 |
| 5 | reactions、00022、删除/furnish | 3 |
| 6 | comment_reactions、00023、内嵌/连带/furnish | 4、5 |
| 7 | 描述内部模型和已有两列SQL | 1 |
| 8 | description_activity、版本纯合并政策 | 7 |
| 9 | 描述API、mentions/versions、00024–25、删除/furnish | 3、6、7、8 |
| 10 | description_versions读API | 2、4、9 |
| 11 | 历史项目目录 | 2 |
| 12 | activities可见分页 | 8、11 |
| 13 | 交错14/15、锁后时钟 | 9、12 |
| 14 | 完整矩阵与整程序性质补证 | 3–13 |
| 15 | I11–13及I1/I2/I15/I17增长 | 14 |
| 16 | 文档/交接/评审记录 | 15、实际评审/验证证据 |

**每Task收口步骤G：** 先完成自身红绿定位命令；改server执行`make lint-go`、`make test`；改契约执行`make gen`，仅SQL执行`make gen-go`；改契约/web/e2e另跑`make lint-web`、`make knip`、`make test-web`；迁移/furnish/故事执行`make e2e`。预期lint-go两段0 issues，测试业务断言通过；环境失败单独记，不能当性质通过。代码/SQL注释英文，文档中文。变异恢复后再跑所属定位测试；独立评审/修复后只`git add`本Task精确文件并提交所列消息，生成物提交后`make gen-check`预期无差异。不得自动合并。

**有API的Task执行H：** 修改`api/modules/issue.yaml`、`api/openapi.yaml`（新增路径的根转引）、`I/adapter/http/handler.go`、`I/module.go`；测试和矩阵在该Task就加入（不是等T14）。HTTP的`TestMain`会检查全模块x-problem-codes，故HTTP定位命令用整个包`go -C server test ./internal/modules/issue/adapter/http -count=1`，不用-run漏掉别的码。所有请求security/bearer；声明requestBody的操作闭合body，无schema的GET沿既有平台语义；空数组[]、cursor空为null；错误/错误码文案同Task修改`web/apps/web/core/lib/error-messages.ts`及`web/packages/i18n/src/locales/en/errors.json`、`zh-CN/errors.json`。每个新写当Task加入`B/issue_write_locks_test.go`的issueWrites、`B/issue_writes_test.go`的事务连接测试，以及`B/gate_test.go`的503枚举；矩阵加入`B/permission_matrix_collaboration_test.go`（T3创建，后续扩展）和种子文件。每个声明的project/workspace码也用HTTP假端口走到。

**有迁移的Task执行M：** 每张表同时修改`server/sqlc.yaml`、`deploy/runtime-grants.sql`、`server/migrations/{schema,issue_schema}_test.go`；扩`I/app/{delete_issue,delete_project_rows}.go`的接口和step列表、`I/adapter/postgres/{delete_issue,delete_rows}.go`及对应`queries/*.sql`；扩对应删除单元/存储测试、`B/issue_deletion_test.go`与`e2e/fixtures/furnish.ts`；已有故事若独立构造目录行，同Task补其API前置，必要的窄夹具共用方法一并加入（T3为`e2e/fixtures/comments.ts`及I15/I17故事）。新增接口按下表；项目删除沿`RowBatch`的`id > After`、1000行键集，不能重扫或OFFSET。工作项删除沿IssueDeletion.IssueIDs；comment_reactions经comments连到issue。每表CASCADE的索引/部分唯一/CHECK/nullable依设计§4，UUIDv7只由应用给。目录夹具同Task经API造新表数据，没有临时豁免。

M的签名沿现有删除端口：工作项方法`(ctx context.Context, d IssueDeletion) error`，项目方法`(ctx context.Context, b RowBatch) ([]uuid.UUID,error)`；两者均由拥有表的Task加入步序，workspace沿既有项目删除消费，不另创跨模块SQL。

| Task/表 | 工作项删除方法 | 项目删除方法 |
|---|---|---|
| T3 comments | DeleteIssueCommentRows | DeleteProjectCommentRows |
| T5 issue_reactions | DeleteIssueReactionRows | DeleteProjectReactionRows |
| T6 comment_reactions | DeleteIssueCommentReactionRows | DeleteProjectCommentReactionRows |
| T9 mentions | DeleteIssueMentionRows | DeleteProjectMentionRows |
| T9 description_versions | DeleteIssueDescriptionVersionRows | DeleteProjectDescriptionVersionRows |

**M的共同反例：** 两层子树+归档子项+相邻另一工作项，创建者≠删除者，先前已删行戳不同；删除目标全部活行同删除时刻/删除者，邻居和旧删戳不变。每张新表至少一次实际变异漏删或删宽，并验证本表grants；T6在comment_reactions建出时另实际变异其经issue_comments连到issues的二层外键，不要求无该表的Task提前创建或重复既有comments变异。定位`go -C server test ./migrations ./internal/modules/issue/adapter/postgres -count=1`和`go -C server test ./internal/bootstrap -run 'Test(DeletingAWorkItem|DeletingAProject|DeletingAWorkspace|TheGrantsFile)' -count=1`，记录实际匹配的现有函数名；M最后仍跑G全量覆盖。

### HTTP参数绑定与用例归属

下表路径统一前缀`/api/v0`；UUID由生成绑定器解析，actor只取context，客户端不能指定。分页绑定query `limit *int,cursor *string`；POST/PATCH对象闭合，HTML长度≤500000 Unicode字符且整体body≤1 MiB；省略/显式null按契约区分。路径reaction使用解码后的值并经同一CheckReaction。OpenAPI operationId与Execute名称对应小驼峰，不另造别名。

| Task | 方法/路径 | 绑定到Execute | 成功 |
|---|---|---|---|
| 3 | POST /issues/{issue_id}/comments | CreateIssueComment(issue_id, body.comment_html) | 201 Comment |
| 3 | GET /issues/{issue_id}/comments | ListIssueComments(issue_id, limit, cursor) | 200 CommentPage |
| 4 | PATCH /issue-comments/{comment_id} | UpdateIssueComment(comment_id, body.comment_html) | 200 Comment |
| 4 | DELETE /issue-comments/{comment_id} | DeleteIssueComment(comment_id) | 204 |
| 5 | POST /issues/{issue_id}/reactions | AddIssueReaction(issue_id, body.reaction) | 201 IssueReaction |
| 5 | DELETE /issues/{issue_id}/reactions/{reaction} | RemoveIssueReaction(issue_id, reaction) | 204 |
| 5 | GET /issues/{issue_id}/reactions | ListIssueReactions(issue_id) | 200 IssueReactionCollection |
| 6 | POST /issue-comments/{comment_id}/reactions | AddCommentReaction(comment_id, body.reaction) | 201 CommentReaction |
| 6 | DELETE /issue-comments/{comment_id}/reactions/{reaction} | RemoveCommentReaction(comment_id, reaction) | 204 |
| 10 | GET /issues/{issue_id}/description-versions | ListDescriptionVersions(issue_id, limit, cursor) | 200 DescriptionVersionPage |
| 10 | GET /issue-description-versions/{version_id} | GetDescriptionVersion(version_id) | 200 DescriptionVersionDetail |
| 12 | GET /issues/{issue_id}/activities | ListIssueActivities(issue_id, limit, cursor) | 200 ActivityPage |

T9扩既有create/update的description_html，update的skip_activity保留是否出现；既有详情的response mapper逐个接描述并保持seen。领域/用例返回值和HTTP生成DTO分别映射，表中Comment等缩写不是额外生成类型。

### 跨Task公共值（各值由表中最早Task创建）

以下仅是接口形状，无实现体。可按职责拆文件，不另创语义相同的类型；ctx一律context.Context、ID一律标准库uuid.UUID。HTTP DTO由OpenAPI生成，不在用例返回gen类型。

```go
// T1: issue/domain/rich_text.go; app/content.go; adapter/content.NewSanitizer
type ContentIdentity struct { Text string; MentionIDs []uuid.UUID; Links []string; Checked []bool }
type RichText struct { HTML, Text string; Identity ContentIdentity; HasMention, HasImage bool }
func (r RichText) EmptyComment() bool // text空且没有mention/image
func (i ContentIdentity) Equal(other ContentIdentity) bool
type HTMLSanitizer interface { Clean(input string) domain.RichText } // app
func NewSanitizer() *Sanitizer // adapter/content，构造后只读

// T3: issue/domain/comment.go; app/comment_ports.go
type CommentReaction struct {
    ID, WorkspaceID, ProjectID, CommentID, ActorID uuid.UUID
    Reaction string
    CreatedAt time.Time
}
type Comment struct {
    ID, WorkspaceID, ProjectID, IssueID, ActorID uuid.UUID
    HTML, Text string
    EditedAt *time.Time
    CreatedAt, UpdatedAt time.Time
    Reactions []CommentReaction
}
type CommentWrite struct { Comment domain.Comment; By uuid.UUID; At time.Time } // app
type CommentDeletion struct { ID, By uuid.UUID; At time.Time } // app,T4
type CommentPage struct { Data []domain.Comment; NextCursor string }
type MemberVisibility struct { ID uuid.UUID; Role shared.Role; OwnOnly bool } // app

// T5: issue/domain/reaction.go; app/reaction_ports.go
type IssueReaction struct {
    ID, WorkspaceID, ProjectID, IssueID, ActorID uuid.UUID
    Reaction string
    CreatedAt time.Time
}
type IssueReactionWrite struct { Reaction domain.IssueReaction; By uuid.UUID; At time.Time }
type ReactionKey struct { ResourceID, ActorID uuid.UUID; Reaction string }
type ReactionDeletion struct { Key ReactionKey; By uuid.UUID; At time.Time }
// T6; CommentReaction的HTTP形状和纯类型在T3定义，表和写到T6才出现
type CommentReactionWrite struct { Reaction domain.CommentReaction; By uuid.UUID; At time.Time }

// T7: domain/description.go; app/description_rows.go
type DescriptionChange struct { Content RichText; Write, Changed bool }
func CompareDescription(old, next RichText, skip bool) DescriptionChange
type DescriptionWrite struct { IssueID uuid.UUID; HTML, Text string } // 仅两列，不写审计

// T8: domain/description_version.go; app/activity_ports.go
type DescriptionVersion struct {
    ID, WorkspaceID, ProjectID, IssueID, OwnedByID, CreatedByID, UpdatedByID uuid.UUID
    HTML, Text string
    LastSavedAt, CreatedAt, UpdatedAt time.Time
}
func CanMergeDescriptionVersion(v DescriptionVersion, actor uuid.UUID, now time.Time) bool
type ActivityRecord struct {
    ID, WorkspaceID, ProjectID, IssueID uuid.UUID
    ActorID *uuid.UUID
    Activity domain.Activity
    CreatedAt, UpdatedAt time.Time
}
type DescriptionActivityMerge struct { ID, By uuid.UUID; NewHTML string; At time.Time }
// T10/T12 app
type DescriptionVersionPage struct { Data []domain.DescriptionVersion; NextCursor string }
type ActivityPage struct { Data []ActivityRecord; NextCursor string }
// T11 app
type VisibleProject struct { ID uuid.UUID; Role shared.Role; OwnOnly bool }
```

**构造约定：** 每个公开用例`X`以`NewX(d XDeps) *X`构造；各Task的Deps字段表即完整依赖，不隐式从全局取；模块构造一个content.Sanitizer和一个Locks供全部用例共享。接口实际只嵌入用例消费的方法，下面同Task列出的store方法按读/写拆小接口；不建一个拥有全部P4方法的大Store接口。假实现继续扩已有callLog和fakes，不复制基础世界。

### Task 1: 清洗适配器、内容标识和S3真实往返

**Files:** Create `server/internal/modules/issue/domain/rich_text.go`、`rich_text_test.go`，`server/internal/modules/issue/app/content.go`，`server/internal/modules/issue/adapter/content/{sanitize,policy,projection}.go`及各自`_test.go`，`api/testdata/issue-richtext.json`，`web/packages/editor/src/sanitization-roundtrip.test.ts`；Modify `server/go.mod`/`go.sum`、`web/packages/editor/src/extensions/callout/extension-config.ts`、`web/packages/editor/src/extensions/custom-color.ts`。

**Interfaces:** Produces公共值表T1全部接口；`content.NewSanitizer() *Sanitizer`实现`app.HTMLSanitizer`。构造时建policy，Clean不修改policy；并发调用只读。T1以编译期接口满足断言连接adapter与app；T3首个消费者到来时才由`module.New`构造一次并注入，T9复用，不引入无人消费的字段、空调用、全局singleton或init。当前库实读版本v1.0.27：`AllowAttrs`、`AllowStyles`、`AllowURLSchemes`、`RequireParseableURLs`、`Sanitize`可用；不得用AllowUnsafe或全属性放行。callout现有Markdown serialize出口需转义属性/文本上下文：优先复用已有可访问的转义；当前utils的sanitizeHTML是私有函数，不为此扩公共API，若实施时仍无可复用出口，则在extension-config.ts内加私有`escapeCalloutHTML(value: string | null | undefined): string`，空值成空串，按顺序转义&、<、>、双/单引号。用于img的src/alt和icon正文，不新增通用包/依赖；URL语义校验仍归服务端适配器。

- [x] **Step 1 写红测试。** `TestCleanEditorHTML`、`TestCleanUnsafeHTML`、`TestContentIdentity`；editor测试通过真实CoreEditorExtensions，复用现有editor-interaction测试的file/mention handler配置。JSON每例保存`input,sanitized,reserialized,resanitized,expected_text,expected_mentions,expected_links,expected_checked`，callout例另存`expected_markdown`；逐边检查Go Clean(input)=sanitized、真实editor(sanitized)的getHTML=reserialized与Markdown=expected_markdown、Go Clean(reserialized)=resanitized及内容投影，不能只读固定样例互相比；raw API无块id、editor固定块id和只读模式分别有例，不由手写HTML冒充editor输出。另用sanitized构造真实editor并调用`editor.storage.markdown.getMarkdown()`走callout的实际serialize（不是仅getHTML，也不是只测独立转义函数）；data-emoji-unicode含编码引号/属性、data-icon-name含编码闭标签/标签的恶意例不得产生额外属性/标签，合法emoji/icon和安全URL须保留。把Markdown中的预期HTML片段解析后精确比较标签、属性集合及值，并固定导出文本。按真实schema枚举每个node/mark，新增schema项使覆盖测试失败；固定sanitized期望，并逐节点/属性断言保留。CSS末尾分号等合法序列化规范化单独列明，不以字节差异直接断言格式丢失，也不以宽松DOM比较掩盖属性丢失。
  ```go
  // TestContentIdentity的独立断言；夹具为保存HTML与同一内容的块id/列表重序列化。
  if !before.Identity.Equal(withIDs.Identity) { t.Fatal("block ids changed identity") }
  if before.Identity.Equal(changedHref.Identity) { t.Fatal("href change hidden") }
  if before.Identity.Equal(changedChecked.Identity) { t.Fatal("checked change hidden") }
  // 另断言重复mention保持顺序，纯格式允许相同，空p.Text==""。
  ```
- **S3实施时的局部根因修复：** 真实editor测试已发现CustomColor把没有颜色属性的span也判为匹配，且其默认优先级低于TextStyle的101，分别吞掉emoji和带style的颜色标记。CustomColor仅接受自身非空data-text-color/data-background-color，缺失明确false或用精确selector；优先级102并解释拥有的解析语义。不改第三方库或全局TextStyle配置。真实往返与分别移除两项修复的变异须证明emoji、文字色/背景色（含style）保留。当前TextStyle没有任何持久属性，空mark按原生规则消失是单列的无语义规范化：精确断言schema attrs为空、文字保留、空mark消失，仍列入schema覆盖，未来增加属性必须让该例失败；不得广泛忽略mark或颜色属性。
- **颜色属性的再序列化边界：** 真实复现表明data-text-color/data-background-color中的分号和URL可在editor里变成新的style声明，表格颜色属性也有相同入口。adapter在现有唯一属性过滤边界对这些颜色值及实际使用的tr/td/th的background/textcolor收窄为安全颜色语法；保留真实COLORS_LIST键、none和合法CSS颜色，优先复用style颜色校验，拒绝注入值。不放宽style、不加依赖；共享恶意/合法样例须证明Go清洗到真实editor后不再产生超出策略的声明，删除校验的可编译变异须失败。
- [x] **Step 2 运行确认红。** `go -C server test ./internal/modules/issue/domain ./internal/modules/issue/adapter/content -run 'Test(Clean|ContentIdentity|RichText)' -count=1`；`pnpm --filter @nerve/editor exec vitest run src/sanitization-roundtrip.test.ts`。预期缺实现或确切往返/安全断言失败；新包不存在的红只说明尚未实现，补出接口后还须看到业务红。
- [x] **Step 3 实现。** 实现阶段执行`go -C server get github.com/microcosm-cc/bluemonday@v1.0.27 golang.org/x/net@v0.60.0`，核对实际go.mod/go.sum差异。x/net v0.60.0与现有crypto.57/sys.48/term.46/text.42相容；v0.61会牵动四个既有版本，故本次不用latest。预期另新增douceur v0.2.0、gorilla/css v1.0.1；若解析结果不同先核对，不无故升级/降级无关依赖。policy照Plane表枚举保留标签/属性，显式限制实际editor CSS，URL协议仅http/https/mailto/tel；自定义image-component[src]和callout[data-emoji-url]复用唯一URL值检查/规整，覆盖协议、空白/控制、危险scheme、引号/实体，并保留安全https值；后者会进入editor导出Markdown的img src，不能因属性名不是src就跳过。x/net/html在adapter中解析清洗结果，StrictPolicy纯文本实体解码并规整空白，提及仅user_mention且UUID有效；投影保留顺序/重复。Clean对任意HTML都返回安全内容，不拒绝请求。同时在callout已有serialize按上述上下文转义src/alt/icon，保留其原有合法输出语义；不把URL清洗与输出转义互相替代。
- [x] **Step 4 绿与变异。** 同命令全绿，并`go -C server test ./internal/archtest -run 'Test(RepositoryFollowsArchitectureRules|PureLayersReachNoInfrastructure)' -count=1`；实际变异分别放行onerror、javascript链接、自定义src、callout的data-emoji-url、危险CSS，删href/checked投影；另分别将callout的alt、icon、src转义改为原字符串（保持可编译），必须被真实Markdown导出断言杀死；每个被具体断言杀死后恢复绿。保持每种保留节点/mark往返，图片标签保留不等于开启上传。
- [x] **Step 5 G、独立评审及提交。** G的server/Web门禁；提交`feat(M4/P4): 通过内容适配器清洗富文本并保护导出`。样例需记实际来源/日期，未通过S3不能进入T3/T7。

### Task 2: 页大小、各自载荷与可组合的只读快照

**Files:** Create `server/internal/shared/{page.go,page_test.go}`、`server/internal/modules/issue/domain/{collaboration_cursor.go,collaboration_cursor_test.go}`、`server/internal/platform/postgres/snapshot_test.go`；Modify `server/internal/modules/identity/domain/{api_token.go,api_token_test.go}`、`server/internal/modules/identity/app/list_api_tokens.go`、`server/internal/platform/postgres/tx.go`、`server/internal/modules/issue/module.go`、`server/internal/bootstrap/{app.go,issue_writes_test.go}`。

**Interfaces:** Produces `shared.PageSize(limit *int) (int,error)`、常量DefaultPageSize=50/MaxPageSize=100；`postgres.ReadOnlySnapshot() Option`；issue.Deps.ReadTx `shared.TxManager`。三个domain类型：`CommentCursor/ActivityCursor{CreatedAt time.Time;ID uuid.UUID}`、`DescriptionVersionCursor{LastSavedAt time.Time;ID uuid.UUID}`，各自MarshalJSON/UnmarshalJSON严格两元素数组，复用shared封套，不复用identity类型。

- [x] **Step 1 红测试。** `TestPageSize`固定nil→50、1/100可用、0/101→422 limit/out_of_range；`TestCollaborationCursors`钉UTC、非零UUID、两项、null/额外项/非法时间/未知封套400、相同时间不同id不相等。`TestReadOnlySnapshotWithLockTimeout`两种Option顺序分别读回transaction_isolation=repeatable read、transaction_read_only=on、lock_timeout指定值；事务内写失败、不同连接提交后第二读不变、新事务可见、事务后参数不留。
  ```go
  size, err := shared.PageSize(nil)
  if err != nil || size != 50 { t.Fatalf("default = %d, %v", size, err) }
  // 对limit=0/101核对shared.Error全体Fields，不仅status。
  ```
- [x] **Step 2 跑红。** `go -C server test ./internal/shared ./internal/modules/identity/domain ./internal/modules/identity/app ./internal/modules/issue/domain -count=1`；`go -C server test ./internal/platform/postgres -run TestReadOnlySnapshot -count=1`（真实库）。
- [x] **Step 3 实现。** 移动PageSize并删identity副本；Options保存配置，NewTxManager在所有Option应用后组合最终BeginQuery，含READ ONLY/REPEATABLE READ/SET LOCAL lock_timeout，避免后Option覆盖前Option；写事务无ReadOnlySnapshot时保持旧语义。bootstrap构造ReadTx并注入所有issue.New测试组合。列表在独立ReadTx内完成首次目标读取、授权、所有目录和SQL；WithinTx嵌套不提升既有写事务隔离，不从写事务调用这些列表用例。
- [x] **Step 4 绿/变异。** 同命令绿；实际去掉隔离、只读、配置组合各变异必须业务失败；PAT既有分页/篡改测试保持通过。不要把写被拒绝的事务继续拿来做后续成功读取断言，应回滚再开新事务。
- [x] **Step 5 G评审/提交。** `feat(M4/P4): 共享分页限制并支持可重复读快照`；移交P5复用此接缝，不重复实现。

### Task 3: 评论创建和分页、成员可见性与00021

**Files:** Create `server/migrations/sql/00021_issue_issue_comments.sql`；`I/domain/{comment.go,comment_test.go}`，`I/app/{comments.go,comments_test.go,comment_ports.go,comment_rows.go,mentions.go,fakes_comments_test.go}`；`I/adapter/postgres/{comments.go,comments_test.go,queries/comments.sql}`；`I/adapter/http/{comments.go,comments_test.go}`；`B/{issue_comments_test.go,permission_matrix_collaboration_test.go,permission_matrix_seed_collaboration_test.go}`；Modify `I/app/ports.go`、`P/app/issue_ports.go`、`P/adapter/postgres/issue_ports.go`及`queries/issue_ports.sql`、`P/module.go`、`B/ports.go`、`B/ports_test.go`、`I/app/fakes_test.go`、`I/module.go`；新增`e2e/fixtures/comments.ts`，同步`e2e/stories/issue/{i15-archive-issue,i17-delete-issue}.spec.ts`的评论API前置；执行M/H。

**Interfaces:** Consumes T1/T2。Produces `CreateIssueComment.Execute(ctx context.Context, issueID uuid.UUID, html string) (domain.Comment,error)`，`ListIssueComments.Execute(ctx context.Context, issueID uuid.UUID, limit *int, cursor *string) (CommentPage,error)`。Deps：Create的`Locks, Comments CommentCreator, Members ProjectMembers, Sanitizer HTMLSanitizer, Clock Clock`；List的`Locks, Comments CommentLister, ReadTx shared.TxManager`。CommentCreator消费InsertComment及既有`InsertSubscribers(ctx,LinkRows)error`、`MarkChanged(ctx,ChangedRows)error`；CommentLister消费`Comments(ctx context.Context, issueID uuid.UUID, after *domain.CommentCursor, limit int) ([]domain.Comment,error)`。共享`GetComment(ctx context.Context,id uuid.UUID)(domain.Comment,bool,error)`、`InsertComment(ctx context.Context,w CommentWrite)error`。ProjectMembers新增`MentionMembers(ctx context.Context,projectID uuid.UUID,ids []uuid.UUID)([]MemberVisibility,error)`；project对应`IssueMentionMembers`返回自己的`IssueMemberVisibility`值，bootstrap转换，不改Lock返回值。此Task由`module.New`构造T1的content.NewSanitizer并注入CreateIssueComment；后续UpdateIssueComment/CreateIssue/UpdateIssue复用同一只读实例。

- [x] **Step 1 红测试。** `TestCreateComment`：空白/仅script清洗后422 comment_html/required、mention/image-only允许、500000字符允许/500001拒绝；调用者+能读被提及者关注，隐藏guest只不关注；无mentions/动态行，所有新行同刻；随机成员重复提及只插一次，现有关注id/戳不变。`TestListComments`：ASC同刻逆id、limit1/50/100、坏cursor400优先坏limit422、其他issue/已删不进页、他人合法cursor不改变target权限。
  ```go
  if got.EditedAt != nil || got.HTML != clean.HTML { t.Fatal("creation shape") }
  if !slices.Equal(subscribers, wantVisibleSortedIDs) { t.Fatal("subscription visibility") }
  if page.NextCursor == "" || len(page.Data) != 1 { t.Fatal("limit+1 page") }
  ```
- [x] **Step 2 跑红。** `go -C server test ./internal/modules/issue/app -run 'Test(CreateComment|ListComments)' -count=1`；`go -C server test ./internal/modules/project/adapter/postgres -run TestIssueMentionMembers -count=1`（新增于issue_ports_test.go）。
- [x] **Step 3 实现。** 评论请求body只含必填comment_html，创建201/list200；CommentReaction纯类型/schema在T3定义，实际表/读写到T6，当前返回[]。project同一查询读有效成员角色和project guest标志，角色为guest且标志关则OwnOnly；issue用HiddenFrom逐候选过滤订阅，排序后InsertSubscribers。写序MarkChanged→subscribers→comments，不写动态/mentions。列表ReadTx内see→Comments(limit+1)，游标仅位置，bad cursor/limit先校验；M的00021同步删除/furnish。新增issue_comment.create规则与action，list使用issue.read。
- [x] **Step 4 绿/变异。** 跑原红命令、HTTP全包、`go -C server test ./internal/bootstrap -run 'Test(Comments|IssueMentionMembers|PermissionMatrix|EachIssueWriteLocksItsProjectFirst)' -count=1`；实际删成员过滤/guest OwnOnly/issue_id/deleted_at谓词、改变写序；安全项需组合层失败。端口errDisk原样返回且后续零调用。
- [x] **Step 5 M/H/G、评审、提交。** 00021/furnish无豁免；`feat(M4/P4): 创建工作项评论并提供分页`。

### Task 4: 评论相对权限与同一路径的子资源检查

**Files:** Create `I/app/{comment_writes.go,comment_writes_test.go}`、`I/adapter/http/{comment_writes.go,comment_writes_test.go}`、`B/issue_comment_permissions_test.go`；Modify `I/app/{lock.go,lock_test.go,comment_ports.go,comment_rows.go}`、comments存储/SQL、`I/domain/{actions,errors}.go`、`server/internal/modules/access/domain/rules.go`及其规则测试，H。

**Interfaces:** `UpdateIssueComment.Execute(ctx context.Context,commentID uuid.UUID,html string)(domain.Comment,error)`、`DeleteIssueComment.Execute(ctx context.Context,commentID uuid.UUID)error`；Deps分别为`Locks, Comments CommentUpdater, Members ProjectMembers, Sanitizer HTMLSanitizer, Clock`与`Locks, Comments CommentDeleter, Clock`。Updater=GetComment+`UpdateComment(ctx context.Context,w CommentWrite)error`+InsertSubscribers+MarkChanged；Deleter=GetComment+`DeleteComment(ctx context.Context,d CommentDeletion)error`+MarkChanged。Locks增加`write.check func(context.Context,domain.Issue,shared.Grant)error`；`seeAs(ctx context.Context,actor shared.Actor,id uuid.UUID,action shared.Action,notFound error)(domain.Issue,shared.Grant,error)`，现有see委托它以ErrIssueNotFound。write.notFound用于指名issue的预检及锁下，不再被硬编码issue错误盖掉。按设计§3.4/3.5复用动作`issue_comment.write`，project级角色集合为Admin/Member/Guest，再由check执行作者/ProjectAdmin相对收窄。

- [x] **Step 1 红测试。** `TestCommentWritePermissions`逐格：作者能改删，管理员不能改别人/能删别人，工作项创建者不等于评论作者；adminGuest自己的issue可管理员删，别人issue404；缺失/隐藏/已删comment同一404。`TestCommentWritePrecheck`非作者403不进gate/tx/clock；锁下被删、降级、移出再判。`TestUnchangedCommentWritesNothing`HTML相同无写，归档仍409；新清洗HTML不同时edited_at/主issue审计同刻。
  ```go
  if !reflect.DeepEqual(hidden, missing) { t.Fatal("child existence leaked") }
  if callsContain("Enter", "Begin", "Now") { t.Fatal("refusal acquired resources") }
  ```
- [x] **Step 2 跑红。** `go -C server test ./internal/modules/issue/app -run 'Test(CommentWrite|UnchangedComment|Locks)' -count=1`；`go -C server test ./internal/bootstrap -run TestCommentPermissions -count=1`。
- [x] **Step 3 实现。** 不加锁GetComment定位issue；run的check在seeAs/decide成功之后、gate之前执行一次，在锁下decide之后再执行一次。check每次GetComment确认未删且IssueID相同，作者/ProjectAdmin规则在其中；捕获当前评论给本次回调（同请求顺序执行）。先404后作者403再归档；普通端口错误原样透出。编辑先issue→新订阅→comment，删除先issue→comment，T6扩连带。编辑时新增可见提及者关注，删提及不退订。新码comment_not_found/not_author同Task文案。
- [x] **Step 4 绿/变异。** 原红命令及HTTP全包绿；变异允许管理员编辑、只在gate后验作者、忘记锁下重读、子资源映射漏掉隐藏issue、同HTML改edited_at分别失败。用真实组合验证预检持外部项目锁时仍即时拒绝，不能只看fake调用日志。
- [x] **Step 5 H/G评审提交。** `feat(M4/P4): 在取锁前校验评论作者权限`。

### Task 5: 工作项表情与00022

**Files:** Create `server/migrations/sql/00022_issue_issue_reactions.sql`；`I/domain/{reaction.go,reaction_test.go}`、`I/app/{reactions.go,reactions_test.go,reaction_ports.go,reaction_rows.go,fakes_reactions_test.go}`；`I/adapter/postgres/{reactions.go,reactions_test.go,queries/reactions.sql}`、`I/adapter/http/{reactions.go,reactions_test.go}`、`B/issue_reactions_test.go`；Modify actions/errors/access规则、`B/permission_matrix_targets_test.go`及`B/permission_matrix_columns_test.go`的精确路径值与自检、`I/app/lock.go`的T4旧预检注释，M/H。

**Interfaces:** `AddIssueReaction.Execute(ctx context.Context,issueID uuid.UUID,reaction string)(domain.IssueReaction,error)`；`RemoveIssueReaction.Execute(ctx context.Context,issueID uuid.UUID,reaction string)error`；`ListIssueReactions.Execute(ctx context.Context,issueID uuid.UUID)([]domain.IssueReaction,error)`。Add/Remove Deps为`Locks, Reactions IssueReactionWriter, Clock`，List为`Locks, Reactions IssueReactionLister, ReadTx`。Writer的`IssueReaction(ctx context.Context,key ReactionKey)(domain.IssueReaction,bool,error)`、`InsertIssueReaction(ctx context.Context,w IssueReactionWrite)error`、`DeleteIssueReaction(ctx context.Context,d ReactionDeletion)error`及MarkChanged；Lister的`IssueReactions(ctx context.Context,issueID uuid.UUID)([]domain.IssueReaction,error)`；domain `CheckReaction(reaction string)error`给字段reaction的422。

- [x] **Step 1 红测试。** `TestIssueReactions`、`TestCheckReaction`：128077/10084-65039合法，空/负/8位段/17段/空段/超过128非法；同actor同value重复409，其他actor同value独立；只能去掉自己的，无自己的404，删除后重新添加新id；成功Add/Remove同步issue审计且零动态/订阅；列表ASC同时间逆id/软删过滤。
  ```go
  if got.ActorID != caller || got.Reaction != "10084-65039" { t.Fatal("wrong reaction") }
  if !issue.UpdatedAt.Equal(at) || issue.UpdatedByID == nil || *issue.UpdatedByID != caller { t.Fatal("audit") }
  ```
- [x] **Step 2 跑红。** `go -C server test ./internal/modules/issue/domain ./internal/modules/issue/app -run 'Test(CheckReaction|IssueReactions)' -count=1`。
- [x] **Step 3 实现。** 正则`^[0-9]{1,7}(-[0-9]{1,7}){0,15}$`与128字符上限在domain/OpenAPI/CHECK一致；不验证emoji名录。锁下先project/issue归档，再检查活reaction存在，成功先MarkChanged后insert/delete；查询/删除绑定issue_id、actor_id、reaction、deleted_at IS NULL。新action issue_reaction.write允许Admin/Member/Guest；list用issue.read。三个API按设计路径/201/204/200，列表mapper将用例slice映射`IssueReactionCollection={data:[…]}`，空集合必须`{data:[]}`而非裸数组；HTTP契约同时断言空/非空封套，新增reaction_exists/reaction_not_found及文案。M加00022及furnish。
- [x] **Step 4 绿/变异。** 目标、存储和`go -C server test ./internal/bootstrap -run TestIssueReactions -count=1`绿；去掉actor谓词、部分唯一/CHECK、漏MarkChanged、已删仍算存在各实际失败；HTTP全包与M。
- [x] **Step 5 H/G评审提交。** `feat(M4/P4): 增加工作项表情及删除连带`。

### Task 6: 评论表情、页内嵌入和00023

**Files:** Create `server/migrations/sql/00023_issue_comment_reactions.sql`，`I/app/{comment_reactions.go,comment_reactions_test.go}`、`I/adapter/postgres/{comment_reactions.go,comment_reactions_test.go,queries/comment_reactions.sql}`、`I/adapter/http/{comment_reactions.go,comment_reactions_test.go}`、`B/issue_comment_reactions_test.go`、`server/migrations/comment_reaction_schema_test.go`、`e2e/stories/issue/i17-catalog.spec.ts`；Modify `I/app/{comment_ports.go,comment_writes.go,comment_writes_test.go,fakes_comments_test.go,comments.go}`、`I/module.go`、`I/adapter/http/comment_writes_test.go`、`B/issue_comments_test.go`及T3–5 rows/comments读取与删除，M/H；目录接线同时扩`B/workspace_deletion_catalog_test.go`、既有`workItemStamps`和`e2e/fixtures/assert/{catalog,issue}.ts`；E2E端点反例复用既有工厂，在各自临时数据库实际ALTER外键。新迁移断言独立文件复用现有基础设施，避免继续扩大约370行的issue_schema测试。

**Interfaces:** `AddCommentReaction.Execute(ctx context.Context,commentID uuid.UUID,reaction string)(domain.CommentReaction,error)`、`RemoveCommentReaction.Execute(ctx context.Context,commentID uuid.UUID,reaction string)error`；Deps `Locks, Comments CommentReader, Reactions CommentReactionWriter, Clock`。CommentReader仅GetComment。Writer的方法：`CommentReaction(ctx context.Context,key ReactionKey)(domain.CommentReaction,bool,error)`、`InsertCommentReaction(ctx context.Context,w CommentReactionWrite)error`、`DeleteCommentReaction(ctx context.Context,d ReactionDeletion)error`。新增窄`CommentReactionReader`仅含`CommentReactions(ctx context.Context,commentIDs []uuid.UUID)([]domain.CommentReaction,error)`，T3 CommentLister与T4 CommentUpdater均嵌入它；module.New向两者注入同一存储，更新相应假实现/调用日志；T4 Deleter扩`DeleteCommentReactionRows(ctx context.Context,d CommentDeletion)error`。

- [x] **Step 1 红测试。** `TestCommentReactions`复用值表但核对issue/comment的updated_at/updated_by/edited_at全部不变、无动态；隐藏comment404；重复409/只删自己同T5。`TestCommentPageIncludesReactions`两comment、多个actor、软删行、页外comment反例。`TestCommentPatchIncludesReactions`为同一评论先造两个actor的活表情和一条软删表情，分别PATCH同HTML与改HTML，均返回准确活表情/排序；同HTML仍零写/零审计变化、改HTML正常审计；不能以未知状态回答[]。`TestDeletingCommentReactions`删除评论只连带自身活表情，别人评论和旧删除戳不变。
  ```go
  if !reflect.DeepEqual(beforeIssue, afterIssue) || !reflect.DeepEqual(beforeCommentAudit, afterCommentAudit) { t.Fatal("reaction touched parent audit") }
  if len(firstPage.Data[0].Reactions) != 2 { t.Fatal("embedded reactions") }
  ```
- [x] **Step 2 跑红。** `go -C server test ./internal/modules/issue/app -run 'Test(CommentReactions|CommentPageIncludesReactions|CommentPatchIncludesReactions|DeletingCommentReactions)' -count=1`。
- [x] **Step 3 实现。** 目标定位及check复用T4的同一机制，但check仅确认comment活且归属一致，不要求作者；Authorize action=issue_reaction.write。写仅comment_reactions；删除评论实际序issues→comments→comment_reactions。ListComments在同一ReadTx取本页commentIDs后批读活reactions按created_at/id排序并分组，空页不查询，不能N+1；T4更新和无改动分支均在当前Locks.run写事务内用单个commentID调用同一窄读取接口，组装完整Comment后再返回，不能在无改动早退时漏组装，也不另开ReadTx。创建的新评论确定没有表情仍返回[]。不新增单独GET表情接口。M的二层FK删除必须经comments，不错用反应表不存在的issue_id。 两个共享reaction业务码开始用于评论时，将detail统一为资源中性的“You already have this reaction.”与“You do not have this reaction.”，同步domain文案及中英翻译；既有action的注释也涵盖评论，不新增码或动作。
- [x] **Step 4 绿/变异。** 原红命令、`go -C server test ./internal/bootstrap -run 'Test(CommentReactions|CommentPageIncludesReactions|CommentPatchIncludesReactions|DeletingCommentReactions)' -count=1`、HTTP全包/M绿；误改父审计、漏comment范围、漏已删过滤、漏二层删除各被业务断言杀死；分别将同HTML、改HTML应答的Reactions清空，两个PATCH组合/HTTP断言均须失败；读取错误原样传播，改HTML分支回滚写入。分页读取夹具在目录/查询间更改表情，证明同快照（T14扩整程序）。目录守卫精确核验comment_reactions.comment_id→issue_comments.id及issue_comments.issue_id→issues.id的实际外键列端点；把新表纳入活行、旧删戳、邻居断言，workItemStamps同样覆盖二层。未知间接表仍拒绝，补同issues父表未知链和错误列端点反例，不设目录豁免。
- [x] **Step 5 M/H/G评审提交。** furnish在新建评论上加一表情；`feat(M4/P4): 维护评论表情并补齐内嵌响应`。

### Task 7: 描述纯模型、存在性和已有列存储

**Files:** Create `I/domain/{description.go,description_test.go}`、`I/app/description_rows.go`、`I/adapter/postgres/{description.go,description_test.go,queries/description.sql}`；Modify `I/domain/{issue.go,patch.go}`及测试、`I/app/{rows.go,create_issue.go,update_issue.go}`、`I/adapter/postgres/{issues.go,update.go}`及`queries/{issues,update,sub_issues}.sql`；Create `B/issue_description_preservation_test.go`；HTTP契约不变。SubIssues沿既有生成行→GetIssueRow→issueOf转换复用新增字段，查询与生成类型同步；没有独立Go映射需要重复修改。

**Interfaces:** domain.Issue增加`DescriptionHTML string`、`DescriptionStripped *string`（保留既有NULL）；domain.NewIssue/IssuePatch增加`DescriptionHTML *string`，IssuePatch加`SkipActivity *bool`保留存在性。公共CompareDescription产出Write（要保存HTML）/Changed（普通审计变化）。app.DescriptionWrite及存储`WriteDescription(ctx context.Context,w DescriptionWrite)error`只改两列；IssueRow/IssueFields同样用`DescriptionHTML string`和`DescriptionStripped *string`保留SQL可空性；app.create_issue构造IssueRow时显式给HTML为`<p></p>`、stripped指向空字符串（非nil）；app.update_issue构造IssueFields从本次锁下旧Issue透传两列；postgres.update映射到对应sqlc参数，插入/更新主查询保持这些值。T9前create/update不消费新增HTTP输入，现有请求行为维持默认描述。

- [x] **Step 1 红测试。** `TestDescriptionChanges`：默认p、空串合法、500000 Unicode字符/500001；skip出现时无description或伴随name/assignee/date/parent等任意字段→422 skip_activity/not_allowed，包括skip=false。比较clean HTML普通相同无写（显式skip=false同样Write=false/Changed=false）、普通不同有写/审计；skip=true标识相同Write=true/Changed=false，文本/mention顺序/href/checked任一不同Changed=true。`TestWriteDescriptionPreservesAudit`存储邻居/issue审计不动。`TestExistingIssueWritesPreserveDescription`在T7末态验证旧API创建存默认两列；通过仓储准备目标与邻居的不同非默认描述，调用既有普通字段PATCH，目标两列保持、邻居完整不变。同一回归另用旧NULL与明确空字符串区分两种持久值，普通PATCH后分别原样保留；不等待T9公开字段。
  ```go
  c := domain.CompareDescription(old, withIDs, true)
  if !c.Write || c.Changed { t.Fatal("migration update was audited or lost") }
  if !domain.CompareDescription(old, changedHref, true).Changed { t.Fatal("href hidden") }
  same := domain.CompareDescription(old, old, false)
  if same.Write || same.Changed { t.Fatal("unchanged ordinary save wrote") }
  ```
- [x] **Step 2 跑红。** `go -C server test ./internal/modules/issue/domain -run 'Test(DescriptionChanges|CheckIssuePatch)' -count=1`；`go -C server test ./internal/modules/issue/adapter/postgres -run TestWriteDescriptionPreservesAudit -count=1`；`go -C server test ./internal/bootstrap -run TestExistingIssueWritesPreserveDescription -count=1`。
- [x] **Step 3 实现。** 领域只消费RichText，adapter负责解析；长度检查按utf8.RuneCountInString。T7同时接app.create_issue的显式默认、app.update_issue从锁下旧值透传、postgres.update的参数映射，主INSERT扩列后不能依赖被显式零值遮蔽的数据库默认；IssuePatch的skip结构校验与既有CheckIssuePatch聚合字段问题。读取GetIssue/GetIssues/SubIssues及readBack映射不丢描述；常规字段PATCH在T9前后都保存现有描述。公开字段到T9完整副作用就绪才进入契约。既有description_stripped可空，不用COALESCE抹平NULL/空串、不改迁移；DescriptionWrite.Text与RichText.Text仍是明确的string。skip组合校验按SetStartDate/SetTargetDate/SetParentID判断字段出现（含显式null），并覆盖其余五个普通指针字段。
- [x] **Step 4 绿/变异。** 原命令及`go -C server test ./internal/bootstrap -run TestExistingIssueWritesPreserveDescription -count=1`、原create/update/读取回归绿；新增组合测试也须先见业务红（若直接绿，用可编译变异确认后恢复）；按字节数长度、忽略false存在、漏一个可选字段的组合拒绝、WriteDescription碰updated_at各失败；把create描述改零值、普通PATCH两列改零值、漏update adapter映射分别被默认/保留值回归杀死；把NULL映射成空串也须被真实持久值断言杀死；生成go及G。T7的内部新接口由T9消费，不添加无人使用的兼容路由。
- [x] **Step 5 评审/提交。** `feat(M4/P4): 建立描述模型并保持既有写入值`。

### Task 8: 最新描述动态合并与600秒版本决策

**Files:** Create `I/domain/{description_version.go,description_version_test.go}`、`I/app/{activity_ports.go,description_activity.go,description_activity_test.go}`、`I/adapter/postgres/{description_activity.go,description_activity_test.go,queries/description_activity.sql}`；Modify `I/domain/{activity.go,activity_test.go}`。

**Interfaces:** 公共T8值；`DescriptionActivityStore`消费现有InsertActivities及`LatestActivity(ctx context.Context,issueID uuid.UUID)(ActivityRecord,bool,error)`、`MergeDescriptionActivity(ctx context.Context,m DescriptionActivityMerge)error`。app内部`writeDescriptionActivity(ctx context.Context,store DescriptionActivityStore,s stamp,change domain.Changed[string]) error`，T9消费；domain.Update增加`Description *Changed[string]`，纯`UpdateActivity`第4行的旧/新值为清洗后HTML，comment按Plane无品牌句子。

- [x] **Step 1 红测试。** `TestDescriptionActivityMerge`最新活行同actor description才合并，保留id/old_value/created_by，更新new/created_at/updated_at/updated_by；最新其他字段或其他actor不合并，即使更早有本人description；同时间id决定最新。`TestDescriptionVersionMergeWindow`599s、600s合并，600s+1µs不合并，不同owner/未来saved_at不合并，第二次以last_saved_at滑动。
  ```go
  if !domain.CanMergeDescriptionVersion(v, v.OwnedByID, v.LastSavedAt.Add(600*time.Second)) { t.Fatal("inclusive bound") }
  if domain.CanMergeDescriptionVersion(v, v.OwnedByID, v.LastSavedAt.Add(600*time.Second+time.Microsecond)) { t.Fatal("outside window") }
  ```
- [x] **Step 2 跑红。** `go -C server test ./internal/modules/issue/domain ./internal/modules/issue/app -run 'TestDescription(ActivityMerge|VersionMergeWindow)' -count=1`；存储同名TestLatestActivity在description_activity_test.go。
- [x] **Step 3 实现。** LatestActivity WHERE issue_id与deleted_at IS NULL，ORDER BY created_at DESC,id DESC LIMIT1，不能WHERE field=description提前筛。WriteDescriptionActivity仅写描述行；T9协调标题行先入库，再调用它，再写后续字段行，避免同PATCH标题被合并跨过。版本纯函数独立于动态，found=false由T9走新建；不用“最新动态”决定版本合并。
- [x] **Step 4 绿/变异。** 原命令加`go -C server test ./internal/modules/issue/adapter/postgres -run TestLatestActivity -count=1`；改成最近description、比较created_at、600改<、不更新new_value、去id tie-breaker各被对应业务断言杀死。普通UpdateActivity顺序回归通过；T9补组合和回滚证据。
- [x] **Step 5 G评审提交。** `feat(M4/P4): 合并描述动态并定义版本窗口`。

### Task 9: 描述API、提及和版本原子写入及00024–00025

**Files:** Create `server/migrations/sql/00024_issue_issue_mentions.sql`、`server/migrations/sql/00025_issue_issue_description_versions.sql`；`I/app/{description.go,description_test.go,description_ports.go,fakes_description_test.go}`；`I/adapter/postgres/{mentions.go,mentions_test.go,description_versions.go,description_versions_test.go,queries/mentions.sql,queries/description_versions.sql}`；`B/{issue_descriptions_test.go,issue_description_atomic_test.go,issue_description_skip_test.go}`、`server/internal/platform/httpserver/apitest/issue_shapes_test.go`；Modify `I/app/{create_issue,update_issue,ports}.go`、`I/domain/patch.go`、HTTP issues映射/测试及module，M/H。组合测试按详情/seen、原子链和skip规则拆分并共享issueWorld；描述在description.go独立筛提及及关注，评论mentions.go保留其必含actor语义，无需制造文件差异。

**Interfaces:** CreateIssueDeps/UpdateIssueDeps增加`Sanitizer HTMLSanitizer`；各自Issues消费既有小接口中实际用到的描述方法，Members仍为T3 ProjectMembers。新增方法：`MentionIDs(ctx context.Context,issueID uuid.UUID)([]uuid.UUID,error)`、`InsertMentions(ctx context.Context,rows LinkRows)error`、`RemoveMentions(ctx context.Context,rows UnlinkRows)error`；`LatestDescriptionVersion(ctx context.Context,issueID uuid.UUID)(domain.DescriptionVersion,bool,error)`、`InsertDescriptionVersion(ctx context.Context,v domain.DescriptionVersion)error`、`UpdateDescriptionVersion(ctx context.Context,v domain.DescriptionVersion)error`。消费T7 WriteDescription、T8 DescriptionActivityStore。公共use case Execute签名保持现有NewIssue/IssuePatch。domain.Changes加`Description *DescriptionChange`；应用层清洗比较后设置它，None仅在所有普通字段无变化且Description为nil或Write=false时为真；内容不变的skip仍需进入写分支，但不走更新审计的普通分支。

- [x] **Step 1 红测试。** `TestDescriptionWriteIsAtomic`：默认/显式创建都有初版，owned/created/updated均写入者；mentions只含有效成员（包括隐藏guest），只有可见者关注；重复/invalid UUID忽略、差集保留id/戳，移除软删，重加新id；仅改状态保留mentions；最后version端口失败回滚issue/assignee/label/subscriber/mention/activity。`TestSkipDescriptionPreservesAudit`包括保存带块id/列表重序列化、只格式、只href/checked变化、成员资格保持时关系行完全不重写；资格变化时只关系同步，issue审计不变。`TestIssueDescriptionAnswers`覆盖所有详情路径及seen。
  ```go
  if got.HTML != clean.HTML || version.HTML != clean.HTML { t.Fatal("saved bodies differ") }
  if !reflect.DeepEqual(beforeAudit, afterAudit) || len(newActivities) != 0 || len(newVersions) != 0 { t.Fatal("migration update left audit") }
  // href/checked样例另断言issue审计==at，动态新值正确、版本last_saved_at==at。
  ```
- [x] **Step 2 跑红。** `go -C server test ./internal/modules/issue/app -run 'Test(DescriptionWriteIsAtomic|SkipDescriptionPreservesAudit)' -count=1`；`go -C server test ./internal/bootstrap -run TestIssueDescriptionAnswers -count=1`（description尚未入契约时HTTP请求应失败，不能只靠编译红）。
- [x] **Step 3 实现。** 两表同Task建出并M全接。Create：清洗一次→项目N下资格→一个clock→issue含描述→assignees→labels→关注并集→mentions→创建动态→初版。Update：重读后清洗旧/新（无description时完全不读提及目录、不触碰mentions/版本），应用CompareDescription；普通相同HTML且其他字段无变化时零写且不读提及目录；skip同内容专用分支只WriteDescription与关系同步，正常分支先主issue+字段关系再subscribers/mentions/活动/版本。新增关注目标与实际新负责人合并去重排序，描述修改者不因编辑而额外关注，按顺序写全局表序。最新版本ORDER BY last_saved_at DESC,id DESC，活行过滤；600秒内同owner原位更新，保留id/created_at/created_by，其他情况新UUIDv7。旧工作项没有版本不伪造初版，首次实际修改时新建一版；skip同内容不新建。
- [x] **Step 4 公开完整契约并验证。** 同Task加IssueCreate/IssueUpdate/IssueDetail.description_html、IssueUpdate.skip_activity；Detail和Issue显式各映射（不再强转），apitest比较properties及required精确差一字段；create/get/by-identifier/update/archive/unarchive仍seen，子项/关联Issue不含description。定位：`go -C server test ./internal/platform/httpserver/apitest -run TestIssueDetailShape -count=1`、HTTP全包、`go -C server test ./internal/modules/issue/adapter/postgres -run 'Test(Mentions|DescriptionVersions)' -count=1`、`go -C server test ./internal/bootstrap -run 'Test(IssueDescription|DescriptionWrite|SkipDescription)' -count=1`。fixture经创建或PATCH造mentions/versions，不临时写SQL。
- [x] **Step 5 变异/G/评审提交。** 实际删mentions差集/角色过滤、skip审计保护、version写入、回滚、seen调用各由组合断言失败；另把version条件按created_at算验证滑动。M/H/G全过，独立评审这个原子交付；`feat(M4/P4): 原子保存描述提及与版本历史`。如评审判断体量不可控，先按D2重切，不提前公开缺少提及/版本的描述接口。

### Task 10: 描述版本列表与详情

**Files:** Create `I/app/{description_versions.go,description_versions_test.go}`、`I/adapter/http/{description_versions.go,description_versions_test.go}`、`B/issue_description_versions_test.go`；Modify `I/app/description_ports.go`、T9存储/SQL、actions/errors/access规则及其tableCells；H的矩阵种子、version_id归属登记。承接T9两项Minor：修改`B/issue_description_atomic_test.go`的超时前提，及`I/adapter/http/{sub_issues.go,issues_test.go,update_test.go}`的旧注释；保留通用守卫规则，不加豁免。

**Interfaces:** `ListDescriptionVersions.Execute(ctx context.Context,issueID uuid.UUID,limit *int,cursor *string)(DescriptionVersionPage,error)`；`GetDescriptionVersion.Execute(ctx context.Context,versionID uuid.UUID)(domain.DescriptionVersion,error)`。List Deps `Locks, Versions DescriptionVersionLister, ReadTx`；Get Deps `Locks, Versions DescriptionVersionReader`。Reader的`GetDescriptionVersion(ctx context.Context,id uuid.UUID)(domain.DescriptionVersion,bool,error)`，Lister的`DescriptionVersions(ctx context.Context,issueID uuid.UUID,after *domain.DescriptionVersionCursor,limit int)([]domain.DescriptionVersion,error)`。List以limit+1查询，DESC键集为`(last_saved_at,id) < (...)`。

- [x] **Step 1 红测试。** `TestDescriptionVersionPages`相同saved_at逆id、1/50/100、空页、满页无更多cursor=null，其他issue和软删不进页，改他人位置不能扩结果；`TestDescriptionVersionVisibility`guest自己403、别人404，adminGuest自己成功/别人404，归档可读；单version隐藏与不存在整个Problem相同，新码issue.description_version_not_found。
  ```go
  if !slices.Equal(ids(page.Data), wantDescendingIDs) { t.Fatal("version order") }
  if !reflect.DeepEqual(hiddenVersionReply, missingVersionReply) { t.Fatal("version leak") }
  ```
- [x] **Step 2 跑红。** `go -C server test ./internal/modules/issue/app -run 'TestDescriptionVersion(Pages|Visibility)' -count=1`。
- [x] **Step 3 实现。** 新action=issue_description_version.read，角色Admin/Member；seeAs先HiddenFrom再角色。Get先读version定位issue，经seeAs统一子资源404，不吞其他存储错误；无写事务。List首次see和全部查询在独立ReadTx内；不能先see再进入快照或在写事务里嵌套“升级”。HTTP列表不带HTML，详情带HTML，路径设计5.1；还原无路由，使用普通PATCH（可能按600秒政策合并版本）。
- [x] **Step 4 绿/变异。** 原命令、HTTP全包、`go -C server test ./internal/bootstrap -run TestDescriptionVersions -count=1`；实际反向tuple、漏issue/软删条件、guest放宽、隐藏404错误映射各失败；错误文案同Task。T9 M1须先断言latest确为同owner的过期版本，再断言新ID；实际移除600秒上限、保留owner条件后仅跑此组合测试，业务失败并恢复。M2同步列表映射与请求字段的旧注释，由HTTP整包覆盖既有行为。
- [x] **Step 5 H/G评审提交。** `feat(M4/P4): 按权限和游标读取描述版本`。

### Task 11: 活动态引用使用的历史项目目录

**Files:** Create `I/app/visible_projects.go`、`P/adapter/postgres/{historical_issue_projects.go,historical_issue_projects_test.go,queries/historical_issue_projects.sql}`、`B/issue_historical_projects_test.go`；Modify `P/app/issue_ports.go`、`P/module.go`、`I/module.go`、`B/ports.go`、`B/ports_test.go`。

**Interfaces:** issue的消费端口`VisibleProjects.Historical(ctx context.Context,workspaceID,actorID uuid.UUID)([]VisibleProject,error)`。project提供`HistoricalIssueProjects(ctx context.Context,workspaceID,actorID uuid.UUID)([]app.HistoricalIssueProject,error)`，值`ID uuid.UUID;Role shared.Role;OwnOnly bool`；bootstrap转issue值（经issue模块公开别名），I.ProjectPorts嵌入VisibleProjects；T12消费，P5再加普通目录方法。不把shared.Actor复制给其他用户逐个授权。

- [x] **Step 1 红测试。** `TestHistoricalIssueProjects`：目标workspace内active成员项目（私有/公开）均可，project/member软删或归档仍可；is_active=false、另workspace、没有membership都排除；guest_view_all关闭的普通guest/adminGuest均OwnOnly；项目不是guest则false；结果id排序；端口失败转换后errors.Is原值。
  ```go
  if !slices.Equal(ids(got), []uuid.UUID{deletedProject, liveProject}) { t.Fatal("historical scope") }
  if !gotGuest.OwnOnly || gotMember.OwnOnly { t.Fatal("project role visibility") }
  ```
- [x] **Step 2 跑红。** `go -C server test ./internal/modules/project/adapter/postgres -run TestHistoricalIssueProjects -count=1`；`go -C server test ./internal/bootstrap -run TestHistoricalProjects -count=1`。
- [x] **Step 3 实现。** project SQL仅JOIN自己projects/project_members，绑定workspaceID/actorID/is_active，不加任何两表deleted_at或archived条件；issue引用表的deleted_at由T12同样不检查。目录不用于放宽目标issue授权，目标仍按正常read规则。不是P3 ArchivedProjects或seeEach/GetIssues的替身，那些过滤软删除会改变历史语义。
- [x] **Step 4 绿/变异。** 目标绿，实际加deleted_at过滤/删is_active/漏workspace/把guest OwnOnly错误置false分别失败，最后一项由真实管理员兼guest组合用例检出，不为变异增加workspace角色依赖；make gen-go与G；目录只读，值转换在B，不跨模块import。
- [x] **Step 5 评审/提交。** `feat(M4/P4): 为动态提供历史成员可见性目录`。

### Task 12: 按可见引用分页列出动态

**Files:** Create `I/app/{activities.go,activities_test.go}`、`I/adapter/postgres/{activities.go,activities_test.go,queries/activities.sql}`、`I/adapter/http/{activities.go,activities_test.go}`、`B/issue_activities_test.go`（引用/项目删除）、`B/issue_activity_pages_test.go`（分页/增量/快照）；Modify `I/app/activity_ports.go`、H。

**Interfaces:** `ListIssueActivities.Execute(ctx context.Context,issueID uuid.UUID,limit *int,cursor *string)(ActivityPage,error)`；Deps `Locks, Activities ActivityLister, Projects VisibleProjects, ReadTx`。Lister的`Activities(ctx context.Context,issueID,actorID uuid.UUID,projects []VisibleProject,after *domain.ActivityCursor,limit int)([]ActivityRecord,error)`；存储把目录拆为全可见projectIDs/ownOnlyProjectIDs两数组，nil规范为空数组；每次请求重新从目录生成，不从cursor读项目范围。

- [x] **Step 1 红测试。** `TestActivityReferenceVisibility`遍历parent/relates_to/duplicate/blocked_by/blocking，old/new两端任一隐藏整行略去；NULL标识不须查，状态/标签/用户标识不当issue；软删issue、软删project历史可见，物理清理引用略去，inactive成员不可见；隐藏父的old_value编号也不能泄漏。`TestActivityPagesFilterBeforeLimit`隐藏/可见交错、limit1、相同created_at逆id、最后页重读、他人合法cursor不能改变结果域。
  ```go
  if len(page.Data) != 1 || page.Data[0].ID != visibleAfterHidden { t.Fatal("filtered after limit") }
  if decoded.ID != page.Data[0].ID { t.Fatal("cursor names hidden or probe row") }
  ```
- [x] **Step 2 跑红。** `go -C server test ./internal/modules/issue/app -run 'Test(ActivityReferenceVisibility|ActivityPagesFilterBeforeLimit)' -count=1`；`go -C server test ./internal/modules/issue/adapter/postgres -run TestActivityVisibilityQuery -count=1`。
- [x] **Step 3 实现。** 独立ReadTx包第一次see→Historical→Activities；SQL只JOINissue模块表，所有引用条件在WHERE里先于ORDER/LIMIT，活的activity自身deleted_at仍过滤。引用issue只要求存在/目录项目可读/OwnOnly匹配CreatedBy，不看它deleted_at和archived_at；两个非NULL标识均须满足。ASC keyset>，limit+1仅为判断更多；cursor用已返回最后一行。响应严格设计字段、不含comment或detail。
- [x] **Step 4 绿/变异。** HTTP全包、原命令、`go -C server test ./internal/bootstrap -run 'Test(ActivityReferences|ActivityPages|ActivityIncremental)' -count=1`；增量从最后一页请求cursor（不是null next）再取，新增和description合并行按id重排；实际漏一端引用过滤/用GetIssues软删口径/过滤在limit后/以cursor授权各失败。
- [x] **Step 5 H/G评审提交。** `feat(M4/P4): 按历史引用可见性过滤动态分页`。

### Task 13: 交错14/15和最后一把锁后的时钟

**Files:** Create `server/internal/bootstrap/issue_collaboration_interleaving_test.go`、`issue_collaboration_clock_test.go`、`issue_collaboration_membership_test.go`；Modify `issue_writes_test.go`为既有路由构造增加测试Clock参数并保留默认入口，`issue_world_test.go`只修workItemStamps过时的固定表数注释；局部扩`issue_interleaving_test.go`与`interleaving_test.go`的等待后回调：原入口委托同一调度实现，确认锁等待后、开gate前推进测试时钟，不复制pgtest锁探针或调度流程。

**Interfaces:** Consumes现有`itemWrite`、`inBothOrdersAt`、`pgtest.RowWaits`、固定Clock及T3/T9 APIs；本Task不新增业务端口。新增测试`TestCommentAndIssueDeletionSerialize`、`TestDescriptionSavesSerialize`、`TestCollaborationReadsClockUnderLock`、`TestActivityCreatedAtDoesNotGoBack`、`TestCollaborationRechecksMembership`、`TestCommentAndReactionsSerialize`、`TestCommentWritesAndArchivalSerialize`。

- [x] **Step 1 写断言。** 交错14两顺序：评论先提交，其评论/关注随后被同删除者同刻连带；删除先提交，评论404/零新关注评论。交错15先取得项目N再读clock，后请求在锁下读取前结果；同actor两次修改动态/版本各合并、最终HTML第二个、old_value仍首次之前、created_at不减；不同actor应分行/版。锁前持者与写入者不同，项目id/时间/id都不能只选自然升序数据。另外用真实交错覆盖成员被移出/降guest后的重判、评论编辑/删除与评论表情添加/删除的两个顺序、评论写与归档的两个顺序；OwnOnly隐藏404和自己描述降guest后的403分开，副表与审计断言零写/连带结果。测试侧Clock须有同步保证，既有clocktest.Fixed本身非并发安全，不修改平台生产时钟。
  ```go
  if len(descriptionActivities) != 1 || len(windowVersions) != 1 { t.Fatal("serialized coalescing") }
  if activity.NewValue == nil || *activity.NewValue != secondHTML || *activity.OldValue != initialHTML { t.Fatal("lost description transition") }
  if secondAt.Before(firstAt) { t.Fatal("time moved before lock") }
  ```
- [x] **Step 2 运行定位。** `go -C server test ./internal/bootstrap -run 'Test(CommentAndIssueDeletionSerialize|DescriptionSavesSerialize|CollaborationReadsClockUnderLock|ActivityCreatedAtDoesNotGoBack|CollaborationRechecksMembership|CommentAndReactionsSerialize|CommentWritesAndArchivalSerialize)' -count=1`。若前序实现已使新增测试直接绿，先做针对性质可编译变异确认业务红再恢复，记录真实经过，不伪造首次红。
- [x] **Step 3 完善交错夹具/必要根因修复。** 先探测请求确实等在持者项目锁之后，再推进固定clock并释放；每次条件等待有截止期限，后台请求错误/取消可收回，不sleep同步。两个并发写的连接和事务分开；后写时钟与所有副表戳一致，不能仅断言最后HTTP状态。
- [x] **Step 4 强化验证。** 上一命令改`-race -count=5`；无40P01/业务失败。实际把clock移到Locks.run前、跳过锁下重读、用初版时间判断窗口各须触发具体断言失败；基础设施/超时不当杀死，恢复后定向绿。
- [x] **Step 5 G/独立评审/提交。** 重推HEAD全部写的gate→连接→项目N→各表锁关系；P4新写按全局表序，既有P3双端关联路径沿P3 spec§3第3条的双方项目锁证明，确认无新增等待环；`feat(M4/P4): 验证协作写入交错与锁后时钟`。

### Task 14: 逐操作矩阵、整程序快照与安全补证

**Files:** Create `server/internal/bootstrap/issue_collaboration_security_test.go`、`issue_collaboration_snapshot_test.go`、`issue_collaboration_rollback_test.go`（真实Store写后故障回滚）；Modify `issue_write_locks_test.go`的issueTables，补齐`issue_mentions`、`issue_description_versions`；局部修改`interleaving_test.go`和`issue_collaboration_clock_test.go`，承接T13两项Minor。检查并复用`permission_matrix_collaboration_test.go`、`permission_matrix_seed_collaboration_test.go`、`permission_matrix_coverage_test.go`、`permission_matrix_issue_columns_test.go`、`issue_writes_test.go`、`gate_test.go`中的既有逐操作覆盖，仅有具体缺口时修改，不为文件清单制造差异。若需拆已有文件保持职责和400行约束，不复制内容。

**Interfaces:** Consumes T3–12的12个新API；矩阵行在各Task已存在，T14补跨操作性质。复用已有matrixRow/issueWorld、事务/授权fake的失败注入能力；真实ReadTx从第一次目标读取开始，不能利用已有写事务嵌套来假装repeatable read（WithinTx嵌套不会改变隔离）。

- [x] **Step 1 断言与夹具。** `TestCollaborationSecurity`目标隐藏/随机不存在整份状态+Problem+稳定头一致；非作者update/非admin delete的403不占gate、不取得写事务连接（预检只读查询仍可短暂借用池连接）；project+issue归档同时存在先project，归档相同comment也拒绝。adminGuest自己issue有管理员相对权限，别人404；普通guest仅可见评论/表情、version403。`TestCollaborationSnapshot`分别覆盖评论页+表情批读、活动目标+历史目录+查询、版本目标+列表：首读后外连接改权限/行，当前页同快照、下一请求新状态。
  ```go
  if !reflect.DeepEqual(hiddenReply, absentReply) { t.Fatal("not-found differs") }
  if !reflect.DeepEqual(firstSnapshotRows, stillSameSnapshotRows) { t.Fatal("mixed snapshots") }
  if reflect.DeepEqual(nextRequestRows, firstSnapshotRows) { t.Fatal("next request kept old snapshot") }
  ```
- [x] **Step 2 定位。** `go -C server test ./internal/bootstrap -run 'Test(PermissionMatrix|ThePermissionMatrixCoversEveryOperation|EachIssueWriteLocksItsProjectFirst|EachWriteOnAWorkItemOfAProjectTooBusyAnswersServerBusy|TheWritesOnAWorkItemRunOnTheirTransactionsConnection|CollaborationSecurity|CollaborationSnapshot|CollaborationRollsBackPortFailures)' -count=1`。覆盖的写操作集合应与契约相等，不能用空表/无匹配通过。
- [x] **Step 3 补根因与错误边界。** 每个新写都经项目N/gate/事务context，故障每个端口都回滚、错误原值；TestCollaborationRollsBackPortFailures复用真实Store，在七个新写的写端口完成SQL后返回同一sentinel，逐项验证整份状态回滚与原错误传播，读取/授权/锁端口的既有故障证据另列；新body非法结构400、HTML/反应业务422、limit422/cursor400都经真实HTTP。删除反应合法不存在404只针对自己，不借其他人的reaction泄漏；目标归档与不存在的顺序照spec。别端修改/删除旧评论不承诺增量即时更新。
- [x] **Step 4 实际变异矩阵。** 至少运行：漏一个operation矩阵行、作者校验移gate后、主查询用pool非tx、ReadTx包晚于see、历史过滤只看一端、把cursor当scope、跳过自定义src清洗、漏新表删除。每个记录业务失败测试/层；安全/锁只有fake单元结果则补组合结果，不能靠“已覆盖”措辞。
- [x] **Step 5 G评审提交。** 对照全部12操作与5新表自查，关闭T13两项Minor：调度完成结果被held消费后清理仍能识别已结束，不重复等空channel；同时间逆id候选的合并资格不同，错误id升序必须触发具体业务断言。定向运行受影响交错（含ActivityCreatedAtDoesNotGoBack），记录一次排序变异及恢复；独立评审逐项给M1/M2结论。`feat(M4/P4): 验证协作权限与快照边界`。

### Task 15: I11–I13接口故事与旧故事增长

**Files:** Create `e2e/stories/issue/i11-comments-reactions.spec.ts`、`i12-activity.spec.ts`、`i13-description-history.spec.ts`、`i2-description.spec.ts`、`e2e/fixtures/assert/issue-collaboration.ts`；Modify `i1-create-issue.spec.ts`、`i2-update-issue.spec.ts`、`i15-archive-issue.spec.ts`、`i17-delete-issue.spec.ts`、`e2e/fixtures/furnish.ts`、`e2e/fixtures/comments.ts`（评论/表情专用API帮手）、`e2e/fixtures/issues.ts`（动态/描述版本专用API帮手）。实际为五个新文件与七个修改文件。原I2为352行、assert/issue.ts为397行，描述故事与协作断言按职责拆出；assert/issue.ts无需修改，复用已有changed/createdFor与删除目录，不另拼activity shape。furnish已有五表API前置，仅更新职责说明。

**Interfaces:** Consumes真实12 APIs与create/update描述，使用既有api fixture、activityRow/byId/madeBy/expectIssueDeleted；新增动态断言经已有activityRow，不另写一套shape。故事准备只能API，库仅用于事后断言。跨600秒边界由T8/T9固定clock测试，不在故事SQL造时间或等待600秒。

- [x] **Step 1 写业务断言。** I11发/改/删评论、自己的和别人的反应、管理员删别人但不能编辑；检查工作项表情data封套、已有多actor表情的同HTML/改HTML PATCH均保留内嵌活表情；I12完整分页后保留产生末页的请求cursor，新增评论及描述合并后按id重排/覆盖，无遗漏；I13版本列表/detail、普通PATCH还原、归档拒绝。I1描述/提及/初版，I2普通描述动态/版本合并、skip块id及href/checked普通审计；I15评论409；I17父/两层子树五新表同删除者时刻、邻居/旧删戳保留。
  ```ts
  expect(merged.map(row => row.id)).toEqual(expectedOrderedIds);
  expect(new Set(merged.map(row => row.id)).size).toBe(merged.length);
  expect(afterIssue.updated_at).toBe(beforeIssue.updated_at); // only block-id save
  ```
- [x] **Step 2 定位运行。** `make build`，`pnpm -C e2e exec playwright test stories/issue --project=chromium`。若新增断言直接绿，按对应skip/过滤/删除变异确认业务红后恢复；不能用构建失败充当业务失败。
- [x] **Step 3 补fixture/实现。** 每项目furnish的每新表已在各迁移Task进入，这里补子树和两actor反例。I13用不同作者制造独立版本，普通还原按600秒合并政策断言，不盲数“必加一版”。一次同时写标题/描述的动态顺序也由I2抽样。
- [x] **Step 4 变异和全量门禁。** 实际故事断言至少杀死skip审计、href/checked、活动隐藏引用、删除漏一表四类；恢复绿。G的web门禁、`make e2e`串行；缺Firefox或既有W11偶发分别如实记录，不能说全绿。W11按负责人交接06，不擅修无关功能。
- [x] **Step 5 独立评审提交。** `feat(M4/P4): 覆盖评论动态和描述历史接口故事`。不计未经运行的故事数量。

### Task 16: 文档、交接与交给Claude终审

**Files:** Modify `docs/v0/v0-design.md`、`docs/v0/M2-auth/M2-design.md`、`docs/v0/plane-diff.md`、`docs/v0/M4-issue-core/M4-design.md`、本spec/plan、`handoffs/{M1-P2-trim-content,M1-P3-trim-platform,M1-closeout,M2-closeout,M3-closeout}.md`；更新已建的`reviews/P4-issue-collaboration-review.md`。仅记录有证据的结果。另承接T15 Minor M1：Create `e2e/fixtures/instant.ts`，Modify `e2e/stories/issue/{i2-description,i12-activity,i13-description-history}.spec.ts`，集中保留纳秒精度的时间排序键，供三故事复用；不引入日期依赖或改生产行为。

**Interfaces:** review沿P3§1–7（方式、验收证据、局部决定、终审发现、plan差异、移交、限制）；消费实际每Task日志/提交/变异/门禁，不用计划预期填结果。T16专属文档职责：执行者可修改Files所列跟踪文档，根任务在其实施期间冻结这些文件的并发修改；保留原有真实失败/修复和逐Task证据，修正当前状态与过时措辞，不覆盖历史。根任务随后维护独立评审及实际分支CI记录。

- [x] **Step 0 关闭T15两项Minor。** 读取task-15-review.md的M1/M2原文。以纯值检查先复现RFC3339Nano的尾零/整秒比较误判，抽取I12已用的小型定宽排序函数给I2/I12/I13，禁止改用丢微秒的Date。确认小数精度不同、无小数秒、微秒差和相同时刻后，只定位三份受影响故事。保留原颜色冲突警告，在本已需要的命令中移除冲突变量，不改全局环境、不为噪声单独重跑。T16独立评审已逐项确认M1/M2为ADDRESSED（499cf108..3c0446ea，task-16-review.md）。
- [x] **Step 1 收集验收。** 对spec§4每项指向可复核日志/提交；没有证据保持未勾选。同步A1原话与content最终放置、纯层守卫实测；填写16Task实际大小/依赖变化，不补造原型或全量重放。
- [x] **Step 2 更新上级。** 按设计3.23/P4：无回复、仅描述历史、44表/无issue_versions、20版保留交P9、PageSize共享；五表逐列差异、500000上限、skip内容标识/审计代价、空评论、表情格式、作者权限、动态分页/过滤/增量限制。未做保留任务不能写为正在运行。
- [x] **Step 3 移交。** 五份handoff追加“处理结果（M4/P4）”与余项，status仍open；P5复用ReadOnlySnapshot/历史目录，P6事件且skip相同不发布，P9保留/清理，P11新DTO/地址及comment_json，P14撤销重做/isMigrationUpdate，P15末页cursor重取，M5图片src投影经收尾交接；没有遗漏原spec§5/7。
- [x] **Step 4 核验/预备终审。** `git diff --check`，纯值定点检查及三故事定位；因承接时间断言变更，顺序执行`make lint-web`、`make knip`、`make test-web`、完整`make e2e`及提交后`make gen-check`；已通过G全量门禁的日志集中列出，新增修复只按影响复测，未有完整门禁则补齐。检查文档相对链接和新文件名。分支CI由根任务在可审查提交上使用系统git推送codex/m4-p4后核验，执行者保留未执行状态及证据落点，不虚填结论；根任务给负责人分支、基线、spec/plan/review和变异台账，当时等待负责人安排Claude整分支终审；本轮新指示及修复结果见下方“整分支终审收口”。
- [x] **Step 5 提交与停止。** `docs(M4/P4): 记录协作验收证据与后续交接`。当时按负责人“先别合并”要求停在待终审状态。负责人随后要求本轮deep-review、完整修复后启动P5；后续接续动作见下方，不改写当时停止的事实。

### CI修复轮 A2：共用动画收尾（实现、独立评审及分支CI通过）

- 依据负责人提供的run40170883949 trace和review§3/A2，负责人已选择A，允许在此修复轮统一修正现用库并验证；T1–T16既有实施/任务评审不重开。
- **已批准A的范围：** 新增现用`@headlessui/react@2.2.10`的版本修正文件到既有`patches/`，同步`pnpm-workspace.yaml`的`patchedDependencies`及锁文件；核对实际ESM/CJS入口，把已替换的动画集合重新纳入完成等待。保留取消清理、焦点/键盘和现有动画时长；不引入新的库、不直接改安装目录。
- **先证伪再修：** 在`web/packages/ui/src/`的既有Vitest/jsdom环境补真实Headless UI组件回归，受控呈现旧动画A被新动画B替代；原版本须因B完成后仍残留DOM而红，修复后在B完成才卸载。另验证过程中重新打开/再次关闭及卸载取消，回退库修正时回归重新失败；实现路径/控制方式在本修复轮细化，不把未执行预期记结果。
- **修复门禁：** 定位S2所有角色和菜单/弹窗关联故事，顺序执行`make lint-web`、`make knip`、`make test-web`、完整`make e2e`及提交后`make gen-check`；无Go变化沿已有Go证据，最新分支CI另验精确HEAD；本机Firefox限制仍单列。共享库修正需独立评审，随后纳入本轮整分支deep-review。
- **未采用B（保留取舍记录）：** 仅移除WorkspaceMenuRoot外层Transition，复用Menu.Items直接随关闭卸载的既有模式；取消该菜单100ms进入/75ms退出动效，登记Plane差异并做快速关闭/重开回归。此方案不修复其他组件的共享库路径，不能写成全局缺陷已解决。
- **实际结果：** 54866c8f；三入口回归9/9，最终原包撤回对照3失败/6通过后恢复9/9；Web54/54、knip0、test17/17、gen-check0，定向浏览器19/19、完整本地133通过/2已知Firefox缺失；独立评审C/I/M0，CI 40175286547全部成功。详见review§2.18及§4，未把A2撤回对照加进106次生产变异。
- **执行边界：** 负责人已批准A，按此范围推进，不再请求同一范围确认。Git自主推送已通过实际SSH推送验证，详情见review§4；该修复轮当时不合并、不清理，后续按本轮整分支终审收口推进。

## 自检结果与大小

16个任务与设计原任务映射在spec§3.2；12个新操作/五表全部有拥有Task，表与fixture同批交付。Task7/8内部准备由Task9消费，不提前公开描述字段；Task11历史目录由Task12消费。公共类型与方法按上面的公共表和各Interfaces定义，不依赖代码转录。Review Focus五项分别绑定实际测试和变异。作者自检报告保存于工作树的`.superpowers/sdd/P4-issue-collaboration/plan-author-report.md`（过程记录，不作为已完成测试证据）。

原plan当前基线行数实测见spec附录；本plan紧凑文本行数不等于实现工作量。T3/T9密度最高，预检重点看是否可一次独立评审；预留修复增长5–10%，若实测超过约16Task或单Task约1500行plan，先重切，仍超按已批准D2拆段并更新设计15/16，不能删scope。独立预检及六项修订复核、T1–T16实施/适用本地验证与独立Task评审均已完成；T1/F1、T6/I1及T9/T13 Minor的复核闭合见review。T16已完成授权文档交接、T15两项Minor处理及适用Web G/故事验证，独立审查C/I/M均0；完整e2e133通过/2项Firefox缺失失败。Step4已完成预备终审核验：T16的gen-check及适用Web门禁通过；首次分支CI 40170883949的S2动画残留按批准A修复，54866c8f独立评审C/I/M均0，CI 40175286547的server/web/e2e全部成功。本机缺Firefox仍单列，CI覆盖该环境。系统git经本仓库SSH推送地址已自主推送；本轮整分支deep-review与局部修复复审完成，合并后核验main CI。16Task实际实现/修复numstat及依赖变化见review §5；T16精确SHA和最终规模见执行报告，不在将提交文档中预造自己的SHA。106次生产变异不包含T16纯值边界红绿，不称全面杀死率。

T16本次验证入口：工作树 `.superpowers/sdd/P4-issue-collaboration/logs/task-16-commands.jsonl`；纯值红绿、三故事6通过、Web54/17/knip、完整e2e133通过/2环境失败及全部首次失败保留于task-16-report。T15两项Minor处理已由T16独立评审确认；表数口径为Nerve44=Plane43+新增1，五份handoff仍open，M4整体完成标准未勾。

## 整分支终审收口（2026-10-25）

- [x] 按负责人新指示，使用deep-review完成全部256文件的分区审查、跨Task接缝核对与16项既有变异原始证据抽查。
- [x] 修复gate测试共享3秒占位请求的时间窗，真实调度探针旧红/修复绿；独立修复复审通过。同步总体设计快照读与M4还原规则两处文档表述。
- [x] 8组并发测试 `-race -count=5`、完整 `make test`、两段 `make lint-go`、`make gen-check` 全部通过；文档和生成物核对见[整分支报告](../../../reviews/re-01-M4P4整体审核.md)。
- 合并与交付：本轮提交按精确SHA核验CI，`--no-ff`合并main；main CI成功后归档过程证据、清理P4分支与worktree，从main启动P5。M4整体完成标准和handoff关闭留到真正收尾。
