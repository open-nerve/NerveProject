# M4/P11 详情数据层实施计划

> 给执行代理：获得实施安排后按本计划逐 Task 执行；使用 executing-plans，或按负责人选择使用 subagent-driven-development。[P11-S1](../reviews/P11-project-number-width-decision.md)已由负责人裁定A，**T1–T7已实现，继续T8**。任务按实际证据逐项勾选，阶段验收尚未完成；阶段结束做整分支 deep-review，修完局部问题后先不合并。

**目标：** 详情资源接生成接口，保持唯一工作项、权限范围读取失效和确认写入，完成按需拒绝与宿主清理。

**结构：** service 接本代 ApiClient；IssueRootStore 统一权限/队列/核心确认；各资源以 Reconciled 保存其自身 DTO/ID 与 scope；不造旧响应转换层。资源一组一组同步迁服务、store、类型和直接消费者。

**技术栈：** 仓库锁定 Node24、pnpm11.10.0、TypeScript/MobX/SWR、Vitest/Playwright、生成 openapi-fetch；无新依赖，无数据库/后端协议变更。

**Spec：** [P11 spec](../specs/P11-web-detail-data.md) 是接口与边界唯一来源；[M4设计](../M4-design.md) §3.9–3.15/§7/§9.5；[预检记录](../reviews/P11-web-detail-data-review.md)。基线 main `df5d3bbc1887486bd105fe6fc663f57f5b371992`，准备分支 `codex/m4-p11`，目录 `.codex/worktress/codex/m4-p11`。

## 全局约束

- 主目录永远 main；准备阶段不改 phase、不装依赖、不写产品原型。实施起点重核 main/远端基线与本计划是否漂移，发现漂移只重核有关内容。
- 只读 plane/refer；不动已有容器/Docker，不全局安装/增加未设计依赖，不链接别处 node_modules；禁 stash/clean/reset --hard/force，不提取 keychain 或其它凭据。push 用 /usr/bin/git。
- 每个 Task 保持可编译；旧类型只有最后实际读者同步迁走才删，不用 as any、as unknown as T 或宽 Partial 实体掩盖协议。新文件约400行，测试/表格可有记录的例外。
- 同一 canonical、权限事实和写队列。资源 held 值和 SWR key 都受权限代际约束；旧代错不提示，当前代错交回；成功修改后的读失败不能当写失败回滚。
- 请求和输入只用生成类型；query 来源 operations；显式封闭 body，不 spread Issue。通过 service 原生回答，不能给旧 URL 贴生成 DTO。
- 新错误码才更新文案表/en/zh-CN及同步测试；复用 FakeNerve/fake-root/fake-issues、fake-projects/deferred，禁止每测试复制一份假客户端。
- 局部问题自主根因修复；若需新共享协议、架构回归或高风险范围变更，先提交已完成工作再上报。详细中文提交，保留 feat/fix/docs(M4/P11) 前缀。
- 下列“新”文件均为实施目标，不把预检计划当文件已存在。C/P 路径缩写见 spec§2.1。全部生产接线在所属 Task 闭合，不延到 T14 才补。
- 数据层接线包含最小 UI 错误、确认等待与类型消费；P12–P15 完整故事保持原归属，不把这些后续页面留项写成阶段完成。
- 既有回归随真实传输迁移，不删断言迁就新服务：issue-changes.test.ts 的旧 IssueService spy/简化响应，以及 base.loading.test.tsx 的旧 Axios URL/子项与关联 fixture，在所属 T2–T4 同步换 FakeNerve/生成原生响应。保留 R31 写后 GET 不复用写前证据、拒绝后零补读、退役后零活动跟进、R32 已确认链接顺序；base.loading/base.clone 的当前输入、拒绝后重试、旧来源晚答/取消、空 HTML 与缺失 HTML、克隆身份不入 body 等性质保持。T4 删除旧服务时同时核测试 import，不只搜生产文件。

## 评审重点

1. 权限 ABA、返回 scope 不匹配、旧 GET 晚到和成功删除后的附属资源能否重新出现。
2. 批量多 ID 队列与核心更新是否共享且不重入；POST 成功、补取失败时是否重发或回滚成功写入。
3. 评论/动态最后一页“请求 cursor”是否保留；同 ID 动态移动时间、同毫秒不同纳秒、第二页失败是否正确。
4. 草稿完整 payload/PATCH→publish、跨 workspace loader、保存后 publish 被拒绝是否保留内容。
5. 共享 M7 版本 DTO 是否诚实保留；两种详情宿主同时存在、同账号切 scope 时 cleanup/late followup 是否互相干扰。

## 验证协议与顺序

**G（每 Task）：** 具体行为断言→同一聚焦命令红→最小实现→绿→对生产性质作变异红→精确恢复字节/hash→绿→受影响回归/types/diff-check→独立要求与质量审查、修复复审→精确路径提交。缺文件/类型编译红另记，不算行为红；变异须使断言失败而非仅编译失败。逐轮记录 argv、cwd、源码 SHA/dirty 清单、raw、exit、测试数/skip 与恢复 hash；原件唯一前缀、禁止覆盖。

**聚焦：** 从仓库根 `pnpm --filter web exec vitest run <C/之后的core/...测试路径>`，例如 `pnpm --filter web exec vitest run core/store/issue/issue-details/issue.fetch.test.ts`。无匹配/全 skip 不算通过。类型 `pnpm --filter web check:types`；共享包改后先 `pnpm exec turbo run build --filter=web^... --force`。DOM 用现有 Vitest 单文件环境注解。新测试的确切目标与性质见每 Task。

**F（严格串行完整本地七项）：**

```sh
make gen-check
make lint-go
GOFLAGS=-p=1 make test
make lint-web
make knip
TURBO_FORCE=true make test-web
make e2e
```

每项 exit0，保留 web 各包非缓存日志，不因 reporter 只打印失败而宣称零测试；核真实任务运行和数量。E2E 用默认 chromium 全套+firefox A4，make e2e 已 build。数据库检查不并跑，不使用旧 bin。实施检查点 T4/T9/T13/T15 的产品 push 前跑 F，之后只因新产品修复/真实失败/未解风险扩大或重跑；纯文档不冒充产品复测。准确分支 SHA 的 server/web/e2e CI 另核，main CI 只在未来授权合并后核。

顺序 T1→T2→…→T15。T1 只准备立即将被 T2–T4 使用的 service；其他服务在资源 Task 原地迁移。T2 交付 root 资源失效和队列基础，T3–T9 各自兑现确认；T11 是剩余入口补齐，不能让已迁的 T2–T10 fire-and-forget 留到最后。

## Task 1：薄服务与实施守卫起点（约350–650行）

**Files：** 改 C/services/issue/issue-api.service.ts/.test.ts；新 issue-detail.service.ts/.test.ts；tools/keywords.json。搜索仅合同登记，不加未消费 wrapper/barrel。
**Interfaces：** spec§2.2 的 getByIdentifier 和9个 detail 方法；生成 body/query/response，ApiClient 注入。

- [x] 核分支基线、实际旧服务调用/21个 exact key；phase 改 M4/P11，当前无 until=M4/P11，无需给期限延期。
- [x] 请求测试逐方法核 verb/path/path params、无多余 slug/project/body、关系/链接资源 ID，204→void；422/403/404/409 和 SessionChanged 原错交回。测试 `usesNativeDetailRoutesAndBodies`、`passesTheWholeIdentifierToTheServer`、`doesNotTransformRefusals`。
- [x] 红/实现/绿：`pnpm --filter web exec vitest run core/services/issue/issue-api.service.test.ts core/services/issue/issue-detail.service.test.ts`；同时跑 api-client 既有测试与 types。服务文件不从生产 barrel 输出无人使用方法；T4前全部接通。
- [x] 变异 M01：把 deleteRelation 的路径 ID 换成对方 issue ID，应在请求断言红；G。提交 `feat(M4/P11): 准备详情资源生成服务并进入实施阶段`。

## Task 2：资源生命周期与详情读取（约900–1400行）

**Files：** C/store/issue/root.store.ts、permission-reads.ts、issue-details/issue.store.ts；新 issue-details/resource.ts/.test.ts；既有 issue.fetch.test.ts、issue-changes.test.ts、issue-modal/base.loading.test.tsx/base.clone.test.tsx 的受影响读取；app 归档详情 header.tsx 和 issueId/page.tsx（spec D3）；新 core/components/issues/detail-entry.test.tsx。
**Interfaces：** scoped resource 的 read/confirm/getter/invalidate/dispose；root 提供局部登记/失效与同队列 `enqueueIssueChanges(ids, operation)`，ID去重排序，核心单项也走同一队列集合。resource 不自己观察角色。根现有 create/update/delete/verification 语义保持。

- [x] 以 FakeNerve 真实 HTTP 与当前 root 写 scope、请求不匹配、较旧 Detail、父项、冷目录、byIdentifier 大小写/Unicode、当前拒绝、退役/删除/ABA 两轮 deferred；旧的“本地没有严格相等 identifier 必须零GET”断言改成服务端解析+写入前scope证明，其他 id/project preflight 断言保留。
- [x] 资源测试 `hidesHeldResourcesImmediatelyAfterPermissionChanges`、`doesNotReviveAfterPermissionABA`、`replaysConfirmationOverOlderReads`、`retiresRegistrations`；两个资源/两个 workspace 不误清。
- [x] 替换核心/父项读取，保留 canonical/verification freshness；核心及附属读取有受控 await/错误；编号 GET 应答未通过 workspace/project/current token 验证前零 canonical/附属请求。
- [x] 归档页头使用本代 detail 读取+SessionSWR；issueId 跳转不在无会话 clientLoader 里建服务，改在有 store 的组件等确认后导航，用返回project_id所指 Project.identifier 与 IssueDetail.sequence_id 拼接；不存在/拒绝时保留错误空态，按钮改到该工作区项目列表，不能保留旧 workspace-views/all-issues 地址。只做数据接线，不扩 P13/P14 UI。
- [x] 跑 resource/issue.fetch/detail-entry 新旧测试+root.store.test.ts+issue.store.test.ts+types；多 ID 队列交错与 reject 后继续测试，禁止占队列后嵌套 root.updateIssue。
- [x] M02 移除 getter 权限校验、M03 移除返回scope检查、M04 移除重放各自行为红；G。提交 `feat(M4/P11): 接通详情读取并统一资源权限失效`。

## Task 3：子项与关联（约800–1350行）

**Files：** issue-details/sub_issues.store.ts、sub_issues_filter.store.ts、relation.store.ts、root.store.ts；C/components/issues/{sub-issues,relations,issue-detail-widgets} 的真实消费者；旧 issue_relation.service.ts/types；新 relation.store.test.ts/detail-changes.test.ts、C/components/issues/relations/data.test.tsx，迁 sub_issues.fetch.test.ts。
**Interfaces：** child collection 只存ID+读取证明；relation 行有 relationId/type/targetId/createdAt；删除以 relationId；状态分布为目录派生。

- [x] 测同项目子项、不依赖 state_distribution、nullable parent、跨项目关联/可见范围、相同关系重复确认；删除请求断言关系 ID，关联目标标题来自 canonical 后更新可见。
- [x] add batch 与 child parent PATCH 串行；关联两端/批量ID有稳定队列顺序；移除父子调用普通 parent_id=null PATCH，delete child 沿 root 删除树。新父/旧父、关联两端的确认回查和列表 stale 按 spec§2.4。
- [x] 将旧 sub_issues.fetch 不可能的跨项目子项 fixture 改合同内样本；原权限旧读 ABA/拒绝性质迁到真实关联跨项目用例，不仅删失败断言。同步 issue-changes/base.loading 的子项和关联请求/fixture，保留写后新鲜度与弹窗加载回归。
- [x] 确认成功后补读403/500保留确认、不再POST；目录拒绝可观察。聚焦 sub_issues.fetch/relation.store/detail-changes+核心列表确认回归/types。
- [x] 核 relations/content、relations/helper 与弹窗的职责：由一次明确写入口提交，onSubmit收到确认实体后不得再次PATCH；去掉已无职责的回调，不发送整份Issue。`submitsOneMutationFromTheRelationEditor`验证一次编辑仅一次PATCH，拒绝留输入；删除/移除关联分别用自己的原生ID和命令。
- [x] M05 强行手加父计数、M06 多ID队列去掉共享/排序、M07 删除用目标issueId，分别行为红；G。提交 `feat(M4/P11): 迁移父子和关联的原生数据与确认流程`。

## Task 4：链接与旧核心服务收口（约450–850行）

**Files：** issue-details/link.store.ts/root.store.ts；issues/links、issue-detail/links、issue-detail-widgets/links及 create-update-link-modal 实际调用；types 的旧链接；新 link.store.test.ts、issues/links/data.test.tsx。

- [x] 原生 title=null、metadata空/favicon_url/error=blocked|failed 的展示；create/update/delete拒绝不改变集合/link_count，写成功旧GET晚到仍保留确认；修改后权威 Detail 使列表 link_count 更新，补读失败仅重试GET。
- [x] 输入只 IssueLinkCreate/IssueLinkPatch；删除204后确认集合，成功才关闭弹窗。metadata失败不转成写失败。
- [x] 核全web最后 IssueService 读者迁走（含 issue-changes 的链接 spy 与所有测试 import），删除旧 issue.service.ts/barrel导出、对应类型和 exact key；改用 FakeNerve 原生请求/响应继续核 R31/R32，service路径被新测试命中不得加入旧URL豁免。
- [x] 同步删除C/store/cycle.store.ts中已无调用的IssueService import/成员/构造；全文及全仓引用核实只有这三个位置，不替M6接新接口，不修改其余cycle行为。保留issue-layouts/removal.test.tsx的M6领域目录边界回归。
- [x] 聚焦 link.store/data + T1–T3；types/static；M08 提前写缓存、M09 元数据failed当HTTP失败；G；F并记录准确检查点。提交 `feat(M4/P11): 接通工作项链接并移除旧核心服务`。

## Task 5：评论与内嵌表情（约1000–1500行）

**Files：** C/services/issue/issue_comment.service.ts；删除issue_reaction.service.ts中的评论方法；comment/comment_reaction stores/root；C/components/comments、issue-detail/issue-activity直接消费者；P/types的旧评论类型；新 collaboration.service.test.ts/comment.store.test.ts、comments/data.test.tsx。

- [x] 先服务原生请求/错误、评论actor_id/comment_html/edited_at/reactions与评论表情actor_id，删除自己的reaction；列出全部输入/返回签名并同任务迁消费者。工作项表情留T6，不提前删除其旧类型/服务方法。
- [x] 评论 fetch 读完所有页并保存最后请求cursor；当前拒绝/第二页失败保留上轮完整值和错误，首次失败不伪完整；confirm create/edit/delete 重放。T7复用并扩增量合并性质，不先留旧 created_at__gt 请求。
- [x] 评论表情直接从 comment.reactions 读；无 GET comment-reactions，删除旧 fetch/apply 双份缓存。新增幂等按ID去重、他人同emoji不受删除影响，删除评论时旧表情答不复活评论。
- [x] UI修改等待成功后清空编辑器/结束pending，拒绝留原文字，跟进 scope 检查。对应旧类型同 Task 删除，不靠 any 迁过。
- [x] 定向服务/store/评论组件及受影响回归/types；M10恢复独立评论表情GET、M11a删除同emoji所有评论actor、M12去掉确认重放，各自行为红；G。提交 `feat(M4/P11): 迁移评论与内嵌表情并保留输入`。

## Task 6：工作项表情（约400–800行）

**Files：** C/services/issue/issue_reaction.service.ts、issue-details/reaction.store.ts/root.store.ts；issue-detail/reactions实际消费者；P/types旧工作项表情类型；新reaction.store.test.ts，扩collaboration.service.test.ts及表情组件测试。

- [x] 原生collection/add/remove服务和请求/拒绝测试；DTO仅actor_id，不复制actor_detail。删除签名不接任意userId，使用当前登录者；按id幂等确认，保留其他人同emoji。
- [x] 复用root队列及资源读取证明，确认前/拒绝不改，旧GET重放成功确认，退役/权限ABA不复活。原issue.fetch的最后旧表情fixture改真实FakeNerve，保留晚答/拒绝性质。
- [x] 真实界面受控await和按码拒绝；成员资料用目录，失败不是空成功。最后删除旧表情类型及其旧URL exact例外，不提前改M7。
- [x] 定向service/reaction/UI加已有issue.fetch/issue-changes与types；M11b删除同emoji所有工作项actor及确认重放反例；G。提交 `feat(M4/P11): 接通工作项表情的原生确认流程`。

## Task 7：动态与时间线游标（约700–1200行）

**Files：** issue_activity.service.ts、issue-details/activity.store.ts；新 issue-details/activity-order.ts/.test.ts；issue-activity/activity 展示/helpers；旧 types 动态细节；新 activity.store.test.ts，扩T5评论测试。

- [x] listIssueActivities 的 query 使用 operation；移除 activity_type/created_at__gt。首次所有页；增量从最后请求cursor重读到next=null；记录一页和多页两种情况。
- [x] 表驱动 `reloadsTheLastRequestedPage`、`repositionsMergedDescriptionActivity`、`ordersNanosecondsBeforeIds`、`keepsTheLastCompleteValueAfterPageTwoFails`、`doesNotContinuePagingAfterRetirement`；ID覆盖而非仅尾追加，种类key区分评论/动态。
- [x] actor_id=null 系统文案、nullable字段、删除source_data/epoch/details/is_bot，并修改所有渲染分支；资料来自目录，拒绝不无声丢活动。历史无对应field的展示沿设计§3.13，不制造客户端事件。
- [x] 补齐已打开活动/评论的命令后增量读取；评论/工作项表情按服务端MarkChanged回读canonical，补读失败保留确认并只重试读。七类真实命令覆盖，评论表情继续完成前一命令留下的待核工作项。
- [x] 聚焦 activity-order/activity/comment+types；M13用next_cursor=null作增量起点、M14 append-only、M15毫秒排序各红；G。提交 `feat(M4/P11): 接通完整时间线与增量游标重读`。

## Task 8：描述版本与共享展示边界（约750–1250行）

**Files：** work_item_version.service.ts；新 issue-details/description-versions.store.ts/.test.ts；core/description-versions/{root,dropdown,dropdown-item,modal}.tsx及测试；两个M4调用方；services/inbox/intake-work_item_version.service.ts及新 intake-description-version.ts；inbox/content/issue-root.tsx；P/types旧description_version。

- [ ] 原生版本 list/get、last_saved_at/id倒序分页、列表管理员/成员允许而guest零请求、快速选版本晚答不覆盖；恢复等待 description PATCH，422/403保留弹窗/内容，无专用restore请求。
- [ ] M4两个模块级WorkItemVersionService移到本代store；共享展示组件的props/accessor与传输解耦。M7旧DTO精确迁到所属service目录，旧URL/body/nullable保持，测试捕获；不创建生成字段补丁或删M7功能。
- [ ] 改SessionSWR/权限key，关闭清理计时器和选择状态受宿主scope控制；最后删共享旧版本类型，M7本地类型明确期限M7。
- [ ] 跑新store/root测试及M7 transport相关回归/types；M16 guest发版本GET、M17恢复失败也关窗、M18旧版本答盖新选项各红；G。提交 `feat(M4/P11): 迁移描述版本并划清收集箱的旧传输类型`。

## Task 9：个人草稿与发布确认（约950–1500行）

**Files：** workspace_draft.service.ts、workspace-draft/issue.store.ts、issue/root.store.ts；issue-modal/{draft-issue-layout,base,form,modal,provider}.tsx、components/description-editor及草稿页root/block/properties/delete-modal真实消费者；P/types/workspace-draft-issues/base.ts；新 workspace-draft/issue.store.test.ts、issue-modal/draft.test.tsx，迁现有workspace-draft/root.test.tsx。
**Interfaces：** Draft/DraftPage/DraftCreate/DraftUpdate/IssueDraftPayload；按draftId的队列。root 从既有createIssue抽出带pending/列表确认的小入口供publish复用，不能在draftstore自建canonical。

- [ ] 原生分页data/next_cursor/total，工作区A/B独立loader、完整替换payload、省略/清空project_id、owner/member权限、旧列表晚到与确认delete；请求拒绝结束loading，保留已持数据。
- [ ] 表单按生成白名单构建payload；测试只改name仍保留description/assignee/label等未编辑合法字段，空/缺字段可保存但发布按服务端422；显式清空parent/date时删除payload键、不发null或合回旧值，数组清空为[]。顶层project_id=null单独处理，草稿保存解除普通工作项分支的必选项目保护；无项目保存、清空再读取/发布均有断言。删除旧平铺Issue假形状、模块/迭代等草稿无合同字段及死方法。
- [ ] 最小消费者闭合：`editsAndDeletesAnUnassignedDraft`核无项目/无项目目录仍显示记录和编辑器、按workspace权限删除；不读取不存在的created_by，不用project admin判断个人草稿。`keepsDraftInputAndDialogAfterRefusal`核保存/删除拒绝不清输入或关窗；成功回调不重复PATCH。迁root.test的旧默认service mock与分页fixture，保留“不访问issue_local_filters”的断言。完整页面故事留P15。
- [ ] `patchesBeforePublishingWithoutABody`：deferred PATCH未完成时POST=0，PATCH拒绝POST=0，成功POST无body。409 draft_changed/403/422后草稿保留PATCH成功内容；两次保存/发布严格串行。
- [ ] publish201经root创建协调：项目/个人主页等已持列表、canonical/计数各确认一次，旧读晚到不抹掉；成功发布移除draft；新确认行进入当前显示列表后参与列宽计算，不为列宽取项目或构造next序号。补读失败不重复publish。
- [ ] workspace角色降级/退出/同账号切scope时不暴露草稿、不跟进导航；核所有WorkspaceDraftService模块实例消失。聚焦两新测试+root/issue-changes/base.loading/base.clone/types。
- [ ] M19发布前不PATCH、M20局部payload覆盖、M21publish只写canonical、M22发布成功后的列宽流程错误发出项目刷新（与M21副本确认分开）各行为红；G；F检查点。提交 `feat(M4/P11): 接通个人草稿并串行保存发布载荷`。

## Task 10：成员提及与旧 JSON 字段（约450–850行）

**Files：** hooks/editor/use-editor-mention.tsx及新test；editor/lite-text/editor.tsx、rich-text/editor.tsx、rich-text/description-input/root.tsx、issues/issue-modal/components/description-editor.tsx、inbox/modals/create-modal/issue-description.tsx；services/workspace.service.ts、相关types/props。四个searchEntity调用为lite-text、description-input、工作项表单描述、收集箱创建描述；inbox/content/issue-root的版本接线已归T8，不拿它代替收集箱创建入口。

- [ ] 逐一列四个searchEntity调用；新hook接受当前workspace/project、编辑权限和成员目录，返回既有编辑器提及展示所需值，不依赖M4列表筛选或旧search response。
- [ ] 项目有效成员、无项目有效工作区成员、inactive排除、无email、最多5条、query变化/切scope晚答、目录拒绝可观察、只读editor零请求分别断言。M7共享编辑器同改建议数据，不改M7本体API。
- [ ] 删除searchEntity方法/独占DTO、四模块实例；扫description_json/comment_json/共享JSONContent真实读者随资源换型删除，editor内部独立JSONContent不误伤；不提前删仍用于P12–P15的旧工作项搜索DTO。
- [ ] hook DOM测试+editor现有测试/types；M23 inactive被建议、M24旧query盖新词各红；G。提交 `feat(M4/P11): 从有效成员目录提供提及建议`。

## Task 11：按需拒绝与详情会话键（约450–900行）

**Files：** issue/issue-details、issue-detail-widgets、issue-modal与属性dropdown的当前读者；C/lib/use-session-swr.ts仅必要调用合同；.oxlintrc.json；新 detail-refusals.test.tsx。

- [ ] 以静态调用表列出state/member/label的await、catch及UI落点；逐项标已由T2–T10闭合或本Task修。保留全部当前拒绝入口测试，不能只测通用helper替代真实调用。
- [ ] 为核心/父项/状态、子项/关联目录、各属性下拉与版本/草稿被拒绝提供可见错误/重试、结束loader；按workspace取跨项目状态；当前错交回，SessionChanged无旧提示。
- [ ] 详情SessionSWR key同时包含loginId、业务scope与permission identity；ABA后不读旧cache，核新会话有请求。原直接SWR豁免路径随迁移删掉；M7旧协议仅由共享展示拿明确输入，不开M4豁免。
- [ ] 聚焦detail-refusals/use-session-swr/issue.fetch/sub_issues.fetch及组件拒绝回归；M25吞错误空数组、M26删permission key各红；G。提交 `fix(M4/P11): 让详情按需读取的拒绝可见并隔离缓存`。

## Task 12：弹窗与浮层宿主（约500–1000行）

**Files：** modals/work-item-level.tsx、issue-details/root.store.ts；issue-detail-widgets/issue-detail-widget-modals.tsx、issue-detail/main-content.tsx、issue-detail/parent-select.tsx、peek-overview/root.tsx/issue-detail.tsx；另按实际writer搜索纳入列表删除宿主；新 modals/work-item-level.test.tsx、issues/detail-host.test.tsx。

- [ ] 清点每个开关和三份数据的全部writer/宿主，用表绑定 owner 与初始值；已删除附件/isArchiveIssueModalOpen只核不恢复。
- [ ] WorkItemLevelModals保持原provider位置，对projectId??workItemIdentifier变化/卸载做layout cleanup；首页/草稿/profile及browse仍能用工作区命令。
- [ ] 详情/peek并存旧宿主卸载不关闭新owner；scope切换同步复位create/existing parent、linkData、relationKey及所拥有开关，peek宿主清peekIssue；同账号旧请求完成不关下一scope的弹窗。
- [ ] 真组件测试 `closesOwnedModalsOnScopeChange`、`keepsTheNewOwnersModalWhenTheOldHostUnmounts`、`doesNotFinishAnOldHostsActionInTheNewScope`；StrictMode挂卸/第二次cleanup安全。P13/P14真实Back故事仍后续。
- [ ] 定向两新测试及base.clone/base.loading；M27全局清所有开关、M28只在卸载清不在scope变清各红；G。提交 `fix(M4/P11): 由实际宿主结束工作项弹窗状态`。

## Task 13：错误、标签边界与统一编号列宽（约650–1100行）

**Files：** issue-modal/base.tsx；use-label-changes.ts真实五个创建调用方；use-workspace-issue-properties相关真实调用；issue-identifier、issue-layouts/list/{base-list-root,default,list-group,blocks-list,block-root,block}.tsx、issue-layouts/utils.tsx；新同目录identifier-width.tsx/.test.tsx（显示helper/局部状态，或等价职责拆分）；tools/keywords.json；新 issue-properties.test.tsx。

- [ ] 两处旧error?.error改按码文案，必要字段错误传已有Field；五处直接label创建改useLabelChanges().create，重名name字段、成功回调只本scope，拒绝不关闭/不造标签。
- [ ] 无projectId只用当前可见且已取项目的labelMap，未取标签不显示；不新增工作区目录或逐项目预取，完整恢复仍归M7。补这种限制/不越权/拒绝测试。
- [ ] 按spec§2.11落实A：List统一宽度状态，按每行合法Project.identifier+sequence_id算最大；IssueBlockRoot在RenderIfVisible外报告行宽，递归展开子项也接同一列。原calculateIdentifierWidth第二参数是实际序号，不传位数。未知项目不猜字符串，空集合/key隐藏输出0，无额外HTTP。
- [ ] 新测试 `keepsOneWidthAcrossGroupsAndLoadedRows`、`growsForPagedAndExpandedRows`、`resetsForScopeAndPermissionChanges`、`doesNotReadProjectsToSizeIdentifiers`；覆盖WEB-9/10/100、profile不同前缀、视口外已加载、过滤/删除保持、权限ABA重算、空/未知目录、两个并存布局。scope含loginId/workspace/list kind/project或profile user/view/permission identity，同scope增长，换scope同步重置；只存局部数值和注册身份，旧行cleanup不得写新owner。
- [ ] getUserProfileIssues/fetchParentStats/isArchiveIssueModalOpen 当前均已无实现，只归档删除核对，不重复开发。projectIdentifier已string|undefined保留。
- [ ] label-create-direct规则覆盖真实调用/通用允许入口，故意直接创建变异应守卫红；不设新豁免。定向identifier-width/issue-properties/base.loading/base.clone+文案sync/types/static。
- [ ] M29标签绕统一create、M30宽度固定1/取消权限scope重置（两性质分记录）、M31错误取error.error各红；G；F检查点。提交 `fix(M4/P11): 统一标签创建与详情错误和编号显示`。

## Task 14：删除过渡、完整修改矩阵与回归（约350–800行）

**Files：** 本段已改的barrel/types/旧service/关键词，C/store/issue新 detail-mutations.test.ts，e2e/stories/smoke/s2-web-app.spec.ts仅实际请求边界需调整处；不为维持绿放宽watchPage/旧URL规则。

- [ ] 按下表给每个修改列四列：确认前不改、403/422/409不改、退役/权限ABA不改/不跟进、成功后各副本及补读失败。不合并语义不同的行凑“已覆盖”；注明哪项无该拒绝码/适用性。
- [ ] 跨资源同issue排队：描述/链接/评论与批量子项交错；最后一页旧GET、删树未知后代、权限失效、不同root/workspace互不污染。核最新源码而非只引用T2绿。
- [ ] 原七个实例+前移两个逐路径零命中；is_bot旧字段/共享旧详情DTO/旧普通links地址消失；搜索与M5–M7余留按期限精确列出，不借grep整个web为零写假完成。
- [ ] keyword exact key/phase、oxlint、knip、domains/死成员都按真实命中判断；输入handoff对应行映射各Task。聚焦全detail*.test+列表root/issue-changes/drag回归；types/static与S2实际构建运行按需要。
- [ ] G；提交 `test(M4/P11): 补齐详情资源修改与权限竞态矩阵`。下一Task最终F覆盖本产品集合，不无理由再重复同一完整检查。

## Task 15：终审、证据与阶段交付（约150–400行文档，修复另计）

**Files：** 本spec/plan/review；M4-design §3.23/§15、总体设计§7.7、frontend-changes，M1/M2/M3输入handoff局部处理结果；新的整分支deep-review报告。

- [ ] spec A1–A10逐条绑定实际命令/源码SHA/raw，独立选择≥15种不同性质的生产变异，不能同性质换两个样本算两项；首失败、恢复绿和hash桥保留。
- [ ] 从实际main基线审全部分支（含准备文档/所有产品提交），deep-review逻辑→集成→覆盖，实际全文清单和遗漏如实记。发现局部问题根因修复并限定复审；结构问题先提交上报。
- [ ] 最后产品输入跑F七项，核非缓存/测试数/构建；修复后只按新的产品差异重跑必要范围。push准确候选并核server/web/e2e，不把产品候选的CI冒认为后来文档HEAD的CI。
- [ ] 删除无读者文件/导出，记录必要后续范围与局部handoff；M4§14、总体§9.4、handoff顶层与M5–M8最终closeout均保持开放。
- [ ] 文稿终审/静态核/精确提交后准确最终分支CI另核，回执放忽略证据目录避免循环提交自身SHA。汇报分支、基线、spec/plan/review/CI。**不合并、不清理worktree**，待负责人安排。
- [ ] 提交 `docs(M4/P11): 汇总详情数据层验证与整分支评审`。

## 修改矩阵与候选变异

每行均补拒绝/退役/确认/旧读晚到；下列是计划，尚未执行。

| 修改 | 确认事实与副本 | 归属 |
|---|---|---|
| 属性、描述、父项PATCH/恢复版本 | IssueDetail→canonical及全部列表；followInSession+host | T2/T3/T8 |
| 子项批量、移除父项、删除子树 | 回答子项、父计数权威补取、旧父未知stale、项目读失效 | T3 |
| 关联新增/删除 | relationId行、两端canonical权威补取、跨项目权限 | T3 |
| 链接新增/修改/删除 | IssueLink/204、link_count权威补取、crawler元数据 | T4 |
| 评论新增/编辑/删除 | IssueComment/204、时间线重放、自己的表情随评论 | T5/T7 |
| 工作项表情新增/删除 | IssueReaction/204、actor维度与幂等ID | T5 |
| 评论表情新增/删除 | CommentReaction/204、评论reactions权威源 | T5 |
| 草稿新增/更新/删除 | Draft/204、工作区页/total的确认重放 | T9 |
| 草稿发布 | PATCH后POST、创建确认入口、草稿移除/确认行显示（无列宽专用项目刷新） | T9 |
| 标签创建 | Label确认、name字段错误、会话跟进 | T13 |

M01–M31 是可选候选，不宣称31项独立安全性质；实施者记录真实失效断言，最终审查至少选15种性质（协议ID、scope、ABA、重放、队列、补读、actor、游标、重排、精度、角色、选择晚答、payload、发布顺序、列表确认、提及隔离、宿主所有权等）。不能为统计修改测试自己产生红。

## 预检边界

本计划14 Task，约一万至一万六千行的规模预估（含测试/删除/文档），不是已量得的实现规模；单Task在实施前按真实消费者和diff重估，预留拆到约16 Task的余量。预检检查依赖、合同、文件、既有回归和守卫期限；不执行实现、装依赖、编译、产品原型或逐Task代码重放。静态结果与独立builder-check/三视角deep-review另见review。P11-S1已裁定A，T9取消列宽专用项目刷新、T13明确局部显示宽度；文档预检已完成，仍由负责人另行安排T1，不据此写实现。
