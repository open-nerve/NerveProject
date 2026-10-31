# M4/P15 协作、草稿与命令搜索实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (- [ ]) syntax for tracking.
> 当前仅授权准备文档和预检；所有实施步骤未开始。获得实施授权后沿既有 inline 方式执行，不自行派逐 Task 实现代理。阶段完成做 deep-review，先不合并。

**Goal：** 完成 [P15 spec](../specs/P15-web-collaboration-drafts-search.md) A1–A16 与 I11–I14/I16/I18 页面、I20 评论、C6–C8/C9 草稿。
**Architecture：** 复用当前 IssueRootStore/PermissionReads/确认写与 DetailResource、useIssueSearch、原生 Draft；仅补 UI 消费者、页面与局部质量，不增加全局协调器或后端协议。
**Tech Stack：** 已锁定 React/MobX/TipTap/Headless UI/Propel/cmdk/Vitest/Playwright；不加依赖。
**Spec：** 上述 spec；基线 main `7a81ea18d8e9d80f079e4e30ae0e90b11be23ef9`，分支 `codex/m4-p15`，worktree `.codex/worktress/codex/m4-p15`。C/P/A/E 路径缩写见 spec §2.1。

## Global Constraints

- 主目录始终 main；只在 worktree 实现。plane/refer 只读；不动既有容器/Docker、不全局安装、不引新依赖、不 stash/clean/reset --hard，不提取凭据。系统 /usr/bin/git 推送，详细中文提交及 feat/fix/test/docs(M4/P15) 前缀。
- 预检保留 tools/keywords.json phase=M4/P14；T1 首个实施提交才改 M4/P15，所有例外按实际职责删除，不延期/放宽架构守卫。
- 不加 any/双重断言/Partial<完整实体> 伪造请求，不把生成 DTO 翻回 Plane 类型。新生产文件约 400 行内，单 Task 含测试约 1500 行内；超预算先按已有职责细分并同步文档。
- 组件副作用和提示依现有 session/scope；请求拒绝、写已确认而补读失败、SessionChanged 分开。Promise 显式负责，字段错误在 focus 前已可读；新文案 en/zh-CN。
- 测试复用 FakeNerve/issueStoreTab/requestedIssue/heldChange、API fixture、issue-list-page/issue-detail-page/browser/dialogs/clock；API 置前置、数据库只断言。不复制完整服务世界，不用 sleep、空匹配/编译失败作行为红。
- 已有能力先跑基线，再做有意义变异；不能故意恢复旧码伪造首次失败。失败测试/行为修复和恢复证据同 Task 提交。
- 局部问题自主根因修复；需要协议/权限所有权/跨领域业务改变时，提交已完成工作、上报等决定。M7 视图服务/版本适配/收集箱取值及 P16 终态守原边界。

## Review Focus

1. **可见与可写分离**：OwnOnly、workspace-admin+project-guest、作者编辑/ADMIN 删除他人、归档展示；T3/T4/T5 的真实组件、服务端 API 反证与页面。
2. **整份草稿和确认链**：九字段保留、clear/omit/null、PATCH→无 body POST、确认后只 GET 重试、双击/切换晚答；T6/T7 的原 store+表单+页面。
3. **搜索与命令范围**：查询立即换 owner、单 debounce、目录显式 workspace、服务端排序不被 cmdk 二次过滤、错误不伪装无结果；T8–T10/T14。
4. **真反馈与可访问性**：主题先确认、hook 内唯一复制反馈、真实 Markdown、IME、嵌套 Escape/回焦/inert、作者名字；T2–T5/T9/T13/T14。
5. **运行证据**：clock 新进程与重新登录、fresh attempt 水位、CSP/资源、三引擎显式匹配，A1–A16 逐项对应实际 executed；T12/T14/T16。

## 验证与证据协议

所有命令在 worktree 根执行。U 为 `pnpm --filter web exec vitest run`，后接任务列出的 `core/...` 精确文件；每次 exit=0、实际非零用例、无未处理拒绝才算绿。没有的文件必须先落真实测试，不能把 no test found 算红。每 Task 后 `pnpm --filter web check:types`、`git diff --check`；改 package 源码先 `pnpm exec turbo run build --filter=web^... --force`，防旧 dist 掩盖。

页面命令 B：先 `make build`，再 `pnpm --filter @nerve/e2e exec playwright test <stories/issue/具体文件.spec.ts> --project=chromium --grep '<指定标题>'`；只断言题名实际包含的页面，原 API 故事保留并在 owning Task/最终 F 回归。使用当次 bin/nerve、watchPage/expectListPageQuiet、recordFocuses/pageBehindHidden；不新增旧接口/CSP/警告豁免。

T5、T10、T14 和 T16 提交/推送检查点串行完整 F：
~~~sh
make gen-check
make lint-go
GOFLAGS=-p=1 make test
make lint-web
make knip
TURBO_FORCE=true make test-web
make e2e
~~~
数据库/testcontainers 套件绝不并行。失败保存原始输出，局部定位修复后按影响重跑；W11 依 06 重跑，不展开专项排查，合并 CI 重跑仍不过才报负责人。

每 Task 五步均记录在 `.superpowers/sdd/P15-web-collaboration-drafts-search/`：cwd/完整 argv、tested SHA 或 dirty 逐文件 hash、raw/exit、executed/skip、变异补丁/行为红/恢复 hash。准备证据独立在主项目 `.superpowers/sdd/P15-preparation-20261031/`。各 Task 的 Step 5 同步 spec 验收、review/plan、要求/质量自查与中文详细提交；未完成项不勾。

## Task 1：时间线与系统操作者（约 450–1000 行）

**Files：** spec T1；既有 C/components/issues/issue-detail/issue-activity/data.test.tsx、C/store/issue/issue-details/{activity.store,activity-order}.test.ts；E/stories/issue/i12-activity.spec.ts。
**Interfaces：** 沿 activity/comment fetch 与增量请求 cursor、IssueUser 原 props；已有 null/SITE_NAME 和 source 清理仅复验。

- [ ] Step 1：核基线、Plane 时间线保留行为及历史 13 实例路径，phase 改 M4/P15；扩 `reordersMergedActivityAcrossPages`、`distinguishesSystemAndMissingActor` 和 I12 (page)。至少超过两页、同毫秒不同纳秒和 kind/id 碰撞。
- [ ] Step 2：U core/components/issues/issue-detail/issue-activity/data.test.tsx core/store/issue/issue-details/activity.store.test.ts core/store/issue/issue-details/activity-order.test.ts；确认已有绿/实际缺口，不重写已完成的数据层。
- [ ] Step 3：补必要展示/本地化，首开读全页、增量按末页请求 cursor 合并；无 system 用户猜测。关闭重开清除远端删改的旧评论，保留增量窗口限制。
- [ ] Step 4：B stories/issue/i12-activity.spec.ts --grep 'I12.*page'；M01 用末页返回 null 代替请求 cursor 或仅 append 使遗漏/重复断言红，字节恢复再绿；系统 actor 三分支与 hidden reference 回归。
- [ ] Step 5：记录 A1、没有重复实现的输入；提交 `feat(M4/P15): 补齐原生时间线页面与系统操作者验证`。

## Task 2：动态过滤与键盘（约 300–650 行）

**Files：** spec T2；新 C/components/issues/issue-detail/issue-activity/controls.test.tsx；新 E/stories/issue/collaboration-interactions.spec.ts 的 C6 过滤部分。
**Interfaces：** 保留 selectedFilters/toggleFilter、sortOrder/toggleSort 合同；PopoverMenu 触发器只有一个 button，菜单选项用可键盘激活语义。

- [ ] Step 1：写 `filtersKeepOneChoiceAndReturnFocus`，测默认过滤、禁止最后一项清空、排序与持久偏好、具名按钮、Enter/Space/Escape。
- [ ] Step 2：U core/components/issues/issue-detail/issue-activity/controls.test.tsx core/components/issues/issue-detail/issue-activity/data.test.tsx；嵌套 button/静态可点击项应有具体 DOM 或键盘失败。
- [ ] Step 3：修触发器与交互语义，helper 点名 lint 已消失的逐项核零；保留源数据的原生 id/排序，不增加新的列表镜像。
- [ ] Step 4：B stories/issue/collaboration-interactions.spec.ts --grep 'C6:.*filters'；M02 破坏键盘激活或允许空过滤使对应断言红，恢复；实际 Popover 打开/选择/回焦及该文件 oxlint 零告警。
- [ ] Step 5：记录 A2；提交 `fix(M4/P15): 收口动态过滤的键盘与按钮语义`。

## Task 3：评论权限、输入与作者命名（约 800–1400 行）

**Files：** spec T3；既有 C/components/comments/data.test.tsx；新 comments/permissions.test.tsx；E/stories/issue/{i11-comments-reactions,i20-guest-boundary}.spec.ts。
**Interfaces：** IssueCommentCreate/IssueComment 原生类型；CommentCard 明确提供项目上下文与 displayName 给 QuickActions。create/update 返回原生确认，remove 沿现有 follow；不扩回复/可见范围。

- [ ] Step 1：写 `adminDeletesButCannotEditOthersComment`、`guestCommentsOwnVisibleIssue`、`imeAndEmptyDoNotSend`、`refusalKeepsInputAndOldSuccessDoesNotClearNewHost`；覆盖作者/普通成员/管理员/管理员兼访客/归档。
- [ ] Step 2：U core/components/comments/data.test.tsx core/components/comments/permissions.test.tsx；断言真实菜单项目、名字与请求数量，真实编辑器 IME 留页面补证，不能只测传入布尔值。
- [ ] Step 3：修项目 ADMIN 删除他人、作者专有编辑、单一按钮、author-name 翻译和提交 readiness/发送锁；区分评论权限和工作项编辑权限；错误保留输入并按 code/field 显示。
- [ ] Step 4：B stories/issue/i11-comments-reactions.spec.ts stories/issue/i20-guest-boundary.spec.ts --grep 'I11.*page|I20.*comment'；M03 去 ADMIN 删除授权，M04 忽略 scope/IME 分别使权限/晚答/不发送断言红，恢复；API I11 旧权限与反应级联回归。
- [ ] Step 5：记录 A3 与实际拒绝/焦点/双击证据；提交 `feat(M4/P15): 完成评论授权与可靠提交交互`。

## Task 4：工作项与评论回应（约 550–1000 行）

**Files：** spec T4；既有 C/components/issues/issue-detail/reactions/issue.test.tsx、C/store/issue/issue-details/reaction.store.test.ts；新 C/components/comments/reaction.test.tsx；I11 页面/C6 表情。
**Interfaces：** 沿 create/removeReaction、create/removeCommentReaction，码点十进制串；评论数据只从 embedded reactions 派生，不能加假的 GET。

- [ ] Step 1：写 `removesOnlyMineAfterConfirmation`、`keepsCountsOnConflictOrRefusal`、`combinedEmojiIsOneReaction`、`archivedKeepsReadOnlyReactions`；按作者命名评论两处触发按钮。
- [ ] Step 2：U core/components/issues/issue-detail/reactions/issue.test.tsx core/components/comments/reaction.test.tsx core/store/issue/issue-details/reaction.store.test.ts；持有请求检查双击/发前不增计数。
- [ ] Step 3：补局部 pending/disabled/owner/命名缺口，归档仍显示已有回应且零写；用现有 picker/本站数据，不改 CSP 或组件数据协议。
- [ ] Step 4：B stories/issue/i11-comments-reactions.spec.ts --grep 'I11.*page' 和 C6 picker；M05 删除本人判断或拆错组合码点，断言他人反应保留/原始请求码点失败，恢复；记录 Network 无 CDN、CSP 零违规。
- [ ] Step 5：记录 A4；提交 `feat(M4/P15): 完成两类表情回应与归档只读验证`。

## Task 5：描述历史、还原与 Markdown（约 600–1100 行）

**Files：** spec T5；既有 C/components/core/description-versions/{root,modal}.test.tsx、C/components/issues/issue-detail/description-versions.test.tsx；E/stories/issue/i13-description-history.spec.ts。
**Interfaces：** 现有 readDescription/handleRestore Promise 和 restoreDescription 协调；EditorRefApi.getMarkDown():string + useCopyText(text,message):Promise<void>，不修改公共 ref。

- [ ] Step 1：写 `historyRefusalStaysOpenAndLateReadCannotReplaceSelection`、`restoresViaConfirmedPatch`、`markdownCopyWaitsAndReportsRefusalOnce`；核列表续页/详情重试/无新版本承诺及 M7 展示适配原调用。
- [ ] Step 2：U core/components/core/description-versions/root.test.tsx core/components/core/description-versions/modal.test.tsx core/components/issues/issue-detail/description-versions.test.tsx；新增 ModalTitle 名字/无正 tabIndex/未 ready 不复制断言。
- [ ] Step 3：补 ModalTitle、局部交互和版本复制，删除调用 void copyMarkdown 的假成功分支；还原沿原写协调，不额外创建版本或清空未确认输入。
- [ ] Step 4：B stories/issue/i13-description-history.spec.ts --grep 'I13.*page'；M06 先关窗/错版本旧答，M07 复制前报成功分别被测试捕获，恢复；含 archive/guest/history list failure、恢复后描述确认。串行 F。
- [ ] Step 5：记录 A5、共享 M7 仅合同兼容；提交 `feat(M4/P15): 完成描述历史的可访问还原与复制`。

## Task 6：个人草稿页面与发布（约 850–1450 行）

**Files：** spec T6；既有 C/components/issues/workspace-draft/root.test.tsx、C/components/issues/issue-modal/draft.test.tsx、C/store/issue/workspace-draft/issue.store.test.ts、C/services/issue/draft-input.test.ts；新 workspace-draft/page.test.tsx；E/stories/issue/i14-drafts.spec.ts。
**Interfaces：** 沿 Draft/DraftInput、draftInput、updateIssue/deleteIssue/moveIssue。UI 保留完整 payload，顶层 project_id 与 payload clear 分开；moveIssue 本身负责 PATCH→无 body POST。

- [ ] Step 1：写 `savesUnassignedDraftWithoutLosingUntouchedFields`、`publishSavesThenSendsNoBody`、`saveFailureNeverPublishes`、`publishRefusalKeepsConfirmedDraft`、`confirmedPublishRetriesOnlyReads`；含非法 priority、失效引用、null清项目/省略保持、空串/[]/清键。
- [ ] Step 2：U core/components/issues/workspace-draft/root.test.tsx core/components/issues/workspace-draft/page.test.tsx core/components/issues/issue-modal/draft.test.tsx core/store/issue/workspace-draft/issue.store.test.ts core/services/issue/draft-input.test.ts；已有队列合同保持绿，实际 UI 缺口才修。
- [ ] Step 3：header/empty-state 显式 route slug，root 保留路由 prop；补分页/无项目/编辑/删除/发布的权限、输入与错误状态。禁止按页长猜 total、根据已解析显示表单丢未编辑字段、直接 acceptRead 替代创建确认。
- [ ] Step 4：B stories/issue/i14-drafts.spec.ts --grep 'I14.*page'；API seed 后真实输入保存、重开、删草稿、移到项目，assert DB 和列表计数。M08 payload局部覆盖、M09 PATCH失败仍POST分别实际红后恢复；OwnOnly/访客/换workspace晚答回归。
- [ ] Step 5：记录 A6 与 P8/P11 原能力复验边界；提交 `feat(M4/P15): 完成个人草稿页面与确认发布流程`。

## Task 7：创建弹窗存草稿（约 500–1000 行）

**Files：** spec T7；既有 C/components/issues/issue-modal/{draft,close-policy,migration}.test.tsx；I14 页面和 collaboration-interactions 的 C9。
**Interfaces：** 沿 CreateUpdateIssueModal discriminated props、onSubmit(formData,true)、DraftIssueLayout 单关闭决策、ModalCore 与 EditorRefApi readiness；无新全局弹窗开关。

- [ ] Step 1：扩 `saveToDraftsFromCreateKeepsFormOnRefusal`、`innerEscapeOnlyClosesDraftPrompt`、`migrationOnlyDoesNotPromptSave`；含缺项目/空标题、描述加载失败、混合用户编辑、无重复保存。
- [ ] Step 2：U core/components/issues/issue-modal/draft.test.tsx core/components/issues/issue-modal/close-policy.test.tsx core/components/issues/issue-modal/migration.test.tsx；对已有 P14 关闭/迁移规则先绿基线。
- [ ] Step 3：补直接 Save to drafts 和关闭确认路径的实际遗漏，拒绝保留外层和编辑内容，确认后关闭正确宿主；不向普通 Issue 请求注入 draft 标志或 skip_activity。
- [ ] Step 4：B stories/issue/collaboration-interactions.spec.ts --grep 'C9:'；真实 TipTap/portal 键盘及回焦、Back、权限切换；M10 放行未 ready 或传播内层 Escape，使错误保存/误关外层断言红后恢复。
- [ ] Step 5：记录 A7；提交 `feat(M4/P15): 收口创建弹窗的草稿保存与关闭`。

## Task 8：工作项命令按职责拆分（约 650–1150 行）

**Files：** spec T8；新 C/components/power-k/ui/pages/context-based/work-item/{copy-commands,mutation-commands}.ts、commands.test.tsx（入口 commands.ts 保留）；既有 projects-app-provider.test.tsx。
**Interfaces：** TPowerKCommandConfig[] 聚合；写请求 IssueUpdate、原生 updateIssue；删除由既有 WorkItemLevelModals 接管，复制子 hook 本任务直接用既有 useCopyText/useCopyLink。

- [ ] Step 1：写 `commandWritesOnlyNativeFieldsInCurrentScope`、`ownOnlyAdminAndArchiveUseReturnedGrant`、`oldContextCannotMutateNewIssue`；逐条覆盖状态/优先级/指派/标签/删除/三复制命令，基线命令入口实际307行，历史418行仅为交接快照。
- [ ] Step 2：U core/components/power-k/ui/pages/context-based/work-item/commands.test.tsx core/components/power-k/projects-app-provider.test.tsx；测试真实命令 action/consumer 和 FakeNerve，错误码文案/确认前不投影。
- [ ] Step 3：按复制/写命令拆分，入口只组合；类型为 IssueUpdate，路由/本代实体显式；属性命令读取轮到写时的权威值以保留队列中前次确认，删除不绕确认窗。同步删除旧文件 copy-hook 例外，T13核全部四项终态。
- [ ] Step 4：C7 先以空根查询进入工作项命令，验证键盘、禁用与删除回焦；中文输入只在 activePage 已选中的属性子页验证。根页非空中文/自由查询等 T10 迁完原生搜索后执行，不能在此提前触发旧搜索接口或豁免请求守卫。M11 去旧 scope 或确认字段保护使对应请求/晚答断言红，恢复；三复制消费者至少有真实拒绝测试，T13补全矩阵。
- [ ] Step 5：记录 A8 及命令拆分新增文件；提交 `refactor(M4/P15): 按职责拆分工作项命令并复用确认写`。

## Task 9：主题确认与工作区路由守卫（约 500–950 行）

**Files：** spec T9；新 C/components/power-k/config/{preferences-commands,creation/command}.test.tsx；T6 三 route 文件；tools/keywords.json 内置 samples 正反例及 tools/keywords.mjs。
**Interfaces：** sessionGuard + updateUserTheme；确认且同代后 setTheme/reload。useRouteWorkspace/route ctx 显式 workspace；不从 router getter推断当前范围。

- [ ] Step 1：写 `themeChangesOnlyAfterSameSessionConfirmation`、`themeRefusalAndLateSuccessDoNotReload`、`creationUsesRouteWorkspaceBeforeRouterEffect`；包含 A→B→A、旧成功/失败及 code 文案。
- [ ] Step 2：U core/components/power-k/config/preferences-commands.test.tsx core/components/power-k/config/creation/command.test.tsx；主题先应用与默认旧范围应实际红；核现有 theme-switcher 作合同样板。
- [ ] Step 3：实现主题同代确认与创建目录/权限显式范围；四文件加入 workspace-address-route，后缀 .tsx?$ 同时覆盖 command.ts，content 不变；正例覆盖四文件、负例留其它领域和 tests。
- [ ] Step 4：执行 node tools/keywords.mjs（同时校验 samples 正反例）；C7 从空根查询用键盘进入主题子页，验证真实浏览器主题拒绝/成功/旧代不重载；根页非空查询留 T10/T14。M12 先setTheme或省slug使拒绝/换工作区断言红，恢复。
- [ ] Step 5：记录 A9；提交 `fix(M4/P15): 让命令主题和创建权限跟随当前会话路由`。

## Task 10：原生命令搜索与本地项目结果（约 850–1450 行）

**Files：** spec T10；新 C/components/power-k/ui/modal/search-menu.test.tsx；既有 C/hooks/use-issue-search.test.tsx；search-results-map.tsx 改明确原生展示或删除无读者部分；E/stories/issue/i18-search.spec.ts。
**Interfaces：** useIssueSearch(root,slug,IssueSearchQuery,open)，ids/proof/loading/error/retry；IssueSearchService 返回 {data:Issue[]}；项目读取 knownProjectIds(workspace.id)/getProjectById 并排除 archived；目录用现有 PROJECTS/id/slug 会话键与 fetchProjects({id,slug}) 复用读取，面板打开才启用，不随q重复取。

- [ ] Step 1：写 `scopeSwitchOmitsProjectAndDropsOldResultsImmediately`、`nativeSearchOrderSurvivesCommandFiltering`、`searchRefusalIsNotNoResults`、`localProjectsUseRouteWorkspace`；包含无q、中文/编号/序号/%/_/\\、同名不同项目、迟答/权限ABA。
- [ ] Step 2：U core/components/power-k/ui/modal/search-menu.test.tsx core/hooks/use-issue-search.test.tsx；实际 cmdk 结果排序/过滤用页面补证；检查唯一 useIssueSearch debounce，无组件双重500ms。
- [ ] Step 3：移除 search-menu 模块级 WorkspaceService，直接原生 Issue 与本地 Project 展示；project范围传project_id/workspace范围省略；显式 loading/error/retry，结果按服务端序，删除 M6/M7 六组旧结果映射。保留既有空输入命令模式。
- [ ] Step 4：B stories/issue/i18-search.spec.ts --grep 'I18.*page'；M13 保留旧query结果，M14 workspace仍传project_id/重新cmdk过滤，使可见范围与顺序断言红，恢复；真实错误/重试只GET。串行 F。
- [ ] Step 5：记录 A10；提交 `feat(M4/P15): 将命令搜索接入原生工作项与本地项目目录`。

## Task 11：清理旧搜索合同与十三实例（约 300–650 行）

**Files：** spec T11；C/services/workspace.service.test.ts（新，M7视图传输兼容测试）；旧专用搜索类型/常量及其所有 barrel、C/components/power-k/core/types.ts 读者；M2/M3 handoff子记录。
**Interfaces：** 只删 searchWorkspace 及专用类型/常量；WorkspaceService 四个 M7 view CRUD/list 原签名/请求保持。ProjectService已不存在，不为此重建文件。

- [ ] Step 1：列13历史路径与当前引用；rg 全仓核 IWorkspace*SearchResult、WORKSPACE_DEFAULT_SEARCH_RESULT、projectIssuesSearch、searchWorkspace 的真实读者。新增 view service 原契约对照用例 `retainsViewTransportWhileSearchIsRetired`。
- [ ] Step 2：U core/services/workspace.service.test.ts；核四 view 方法的真实路径/body/响应，不能把未实现远端响应 mock 绿声称 M7 页面已接通。类型引用变化须有具体编译或静态输入证据。
- [ ] Step 3：删除已无读者旧搜索方法/专用 DTO/常量/导出，修实际类型消费者；保留 IWorkspaceView 等 M7合同。13实例逐路径记零，另列非M4剩余实例，不用泛grep全零作虚假目标。
- [ ] Step 4：node tools/keywords.mjs；make knip 与 web check:types；M15 临时恢复 search-menu 模块实例/旧搜索调用让精确审计或消费者请求断言失败，恢复，复跑 T10 与 view契约。全仓 project.service/projectIssuesSearch零真实读者。
- [ ] Step 5：记录 A11/M2§12/M3§5子项，顶层open；提交 `refactor(M4/P15): 删除旧工作区搜索合同并保留视图边界`。

## Task 12：自动归档时钟页面（约 350–750 行）

**Files：** E/stories/issue/i16-auto-archive.spec.ts、E/fixtures/{clock,test,auth,auto-archive}.ts；复用既有 archiveWatermark/completedPeriodicJobs。
**Interfaces：** ClockWorld.advance(offset):Promise<ClockRun> 返回新的 nerve/api/startedAt。clockTest从Playwright base扩展，无普通 signedInPage/api/db fixture；页面直接使用新baseURL和既有auth helper重新登录。

- [ ] Step 1：新增 `I16 (page): fresh clock process archives terminal issues and shows the system actor`；API建账号/项目/旧终态与非终态控制项，记录 old/current URL、写后水位和fresh attempt条件。
- [ ] Step 2：make build 后 B stories/issue/i16-auto-archive.spec.ts --grep 'I16.*page'；用真实服务器/worker，不mock任务，不把expiry后的旧token/cookie当登录。原API crossing/historical negative继续回归。
- [ ] Step 3：只补页面需要的复用fixture，推进744h后真实新登录/刷新，不改业务自动归档算法；等新进程启动后且水位后started的completed任务，不仅按finalized_at判断。
- [ ] Step 4：页面打开归档详情，确认SITE_NAME与对应活动，手动恢复显示用户；M16 过宽fresh判断由原API负例击穿，M17把null当停用用户由页面击穿，恢复。数据库只查询断言。
- [ ] Step 5：记录 A12，保留启动/水位/attempt/SQL读回与页面截图；提交 `test(M4/P15): 验证自动归档的新时钟进程与系统活动页面`。

## Task 13：剩余复制消费者与例外（约 500–900 行）

**Files：** spec T13；新 C/components/issues/collaboration-copy.test.tsx；T5 modal、T8 copy-commands测试；tools/keywords.json；C8 页面。
**Interfaces：** useCopyLink/useCopyText均两参数Promise<void>且catch失败；copyCommentLink若改为Promise<void>须同步TCommentsOperations及所有消费者。优先保持既有void受控入口，不为反馈增加公共返回值。

- [ ] Step 1：写 `copyWaitsForClipboardAndReportsOneOutcome` 参数化创建toast/评论/布局快捷操作/命令编号标题URL，加T5Markdown；持有/resolve/reject，检查链接origin/#comment及零业务写。
- [ ] Step 2：U core/components/issues/collaboration-copy.test.tsx core/components/power-k/ui/pages/context-based/work-item/commands.test.tsx core/components/core/description-versions/modal.test.tsx；禁止只mockhook，实际消费者触发clipboard Promise。
- [ ] Step 3：余下三个位置迁hook，T8已迁只复核；删局部copied假成功与重复toast、四M4例外清零。菜单/按钮可键盘到达；保留M6–M8例外以及公共EditorRefApi。
- [ ] Step 4：B stories/issue/collaboration-interactions.spec.ts --grep 'C8:'；C8必须在真实LAN HTTP地址核工作项/评论/编号/子项/关联入口，复用P14已有消费者；受控拒绝仅作其它分支补证，不代替LAN验证。M18去hook await或外层假成功让消费者等待/错误计数红，恢复；关键词通过。
- [ ] Step 5：记录 A13/T5联动；提交 `fix(M4/P15): 统一协作入口复制反馈并清空M4例外`。

## Task 14：三浏览器组件矩阵与会话（约 850–1400 行）

**Files：** spec T14；新 E/playwright.m4-collaboration.config.ts、E/fixtures/issue-collaboration-page.ts（仅必要小helper）；各owning Task I页面/C6–C9；扩现有会话story时只加本期行为。
**Interfaces：** 配置仿playwright.m4-detail.config.ts，显式chromium/firefox/webkit三project且各无默认A4 grep；testMatch包含本期I11/12/13/14/16/18/20和collaboration-interactions，grep严格匹配本期page/C6–9。

- [ ] Step 1：登记逐条 C6时间线/筛选/表情/历史，C7命令/中文搜索/主题，C8复制，C9草稿的具名矩阵；I20只选新增comment分支，原I20仍全F覆盖。每个browser必须实际非零。
- [ ] Step 2：先 --list 核匹配标题/文件不遗漏I16 clock且无默认Firefox A4；此步只是选择审计，不算执行。复用每Task页面，补两标签页成功旧答、权限ABA、Back/重开、拒绝后键盘恢复。
- [ ] Step 3：只修局部真实缺口；不得降低watchPage、CSP、旧API或console门槛。真实编辑器/portal/cmdk排序/响应持有不能用生产mock替代。新helper不复制SQL与seed整套。
- [ ] Step 4：make build；逐个串行运行 `pnpm --filter @nerve/e2e exec playwright test --config playwright.m4-collaboration.config.ts --project=chromium`、firefox、webkit（各独立output/report目录）；留实际passed/failed/skipped/trace/截图与服务hash。串行 F。
- [ ] Step 5：记录 A14与每行证据；提交 `test(M4/P15): 补齐协作草稿命令的三浏览器矩阵`。

## Task 15：本期死成员、lint与边界（约 350–850 行）

**Files：** 本期真实变动TS/TSX、tools/keywords.json；docs/v0/{frontend-changes,plane-diff}.md；阶段review与相关输入handoff子回执。
**Interfaces：** 按02死代码四分类逐个记录成员/参数/消费者；生成DTO和M7共享props不因名字相似误删，不提前扩大P16全局no-shadow任务。

- [ ] Step 1：从基线..HEAD及dirty枚举完整生产改动，读每个文件，查引用；核时间线helper、activity-filter、版本modal指定告警已零，旧未使用enableReplies等若无合法用途同任务删除全部传参。
- [ ] Step 2：对实际TS文件跑 pnpm exec oxlint --max-warnings=0 <明确文件表>，make knip、node tools/keywords.mjs、翻译键检查（沿make lint-web），记录真实文件数而非全仓警告配额。
- [ ] Step 3：根因消除本期dead/lint，必要按小职责拆分；同步差异清单、P15处理子回执及M7保留边界，顶部handoff仍open。phase只M4/P15。
- [ ] Step 4：复跑受影响unit/B与types/diff；比对13路径、四M4copy例外、四workspace-route正例；静态守卫变化有实际反例，恢复后通过。不把此Task当C12/M4closeout。
- [ ] Step 5：记录 A15；提交 `refactor(M4/P15): 清理协作页面死成员并同步边界记录`。

## Task 16：完整门禁、整分支审查与交付（约 350–950 行）

**Files：** 本spec/plan、reviews/P15-web-collaboration-drafts-search-review.md；实施期新增 docs/reviews/re-NN-M4P15整体审核.md；ignored运行与审计台账。
**Interfaces：** deep-review whole branch为7a81ea18..最终候选及未提交修改；三视角由主控顺序做，不因通用技能自行派并行实现/评审。发现架构问题遵原纪律。

- [ ] Step 1：逐 A1–A16 建映射、逐文件检查，核每项RUN/exit/输入hash/恢复hash；写所有自决理由与错判代价，不把static/precheck或上一phaseCI当产品验收。
- [ ] Step 2：在最终代码/测试输入串行完整F；T14矩阵已通过且输入未变可复用逐文件hash证据，变更影响项重跑，不靠turbo旧缓存；确保本期stories实际匹配。
- [ ] Step 3：使用deep-review完整读变更及关联测试，按逻辑/集成/覆盖分类；局部修复完整并定向复审，结构问题先提交上报。报告明确审查范围、审者/方法、发现、修复和剩余边界。
- [ ] Step 4：核最终候选相关门禁、修改后所需三浏览器，文稿静态自查；系统git提交/推送，核准确最终HEAD的server/web/e2e CI，保留失败/重跑。归档完整证据manifest，final-delivery后置验本地/远端SHA。
- [ ] Step 5：勾A与T1–T16仅在对应证据齐全后，提供分支、基线、spec/plan/review/整审路径和CI；阶段完成先不合并，等负责人安排。提交 `docs(M4/P15): 汇总协作阶段验收审查与交付证据`。

## 附录：准备与实施界线

本计划 Task 1–16 和 A1–A16 均待实施。准备阶段仅静态核路径、签名、依赖、覆盖和既有合并证据，并进行 fresh builder-check 与 deep-review 文档预检；没有产品原型、安装依赖或 P15 测试结果。后续若授权实施，从 T1 开始，不重复准备已通过的合同。
