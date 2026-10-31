# M4/P14 详情、创建弹窗、父子、关联与链接实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (- [ ]) syntax for tracking.
> 本轮仅文档预检，未获实施授权。获准后沿既有 inline 执行方式从 T1 开始；不因通用工作流自行派逐 Task 实现代理。阶段结束做 deep-review，修复复审后先交负责人安排，不自动合并。

**Goal：** 完成 P14 spec A1–A16 以及 I1/I2/I8/I9/I10/I17/I20/I22 页面、S2 身份、C3 详情侧/C4/C5/C10/C11。

**Architecture：** 复用原生 IssueRootStore、PermissionReads、IssueResources、useDetailScope、确认命令与自动保存。新代码限于现有页面/表单的缺口；不改变 API、数据库、锁、权限所有权，不增加全局缓存/协调器。

**Tech Stack：** 仓库锁定的 React/MobX/Headless UI/Propel/TipTap/Vitest/Playwright；无新增依赖。

**Spec：** [P14 spec](../specs/P14-web-issue-detail.md)。基线 main `8bb4bbcb52a224ace4cb2f9d383bca9b97a022bf`；分支 `codex/m4-p14`；worktree `.codex/worktress/codex/m4-p14`。C/P/R/E 路径缩写见 spec §2.1。下文所有步骤尚未实施，新增文件在任务中明确标为“新”。

## Global Constraints

- 根目录永远 main；开发只在约定 worktree。系统 `/usr/bin/git` 推送，详细中文提交，前缀 feat/fix/test/docs(M4/P14)。不用 stash/clean/reset --hard/force，不取钥匙串或浏览器凭据。
- plane/、refer/ 只读；先按保留功能读 Plane；不改既有容器/Docker，不全局安装、不加设计未点名依赖。使用既有安装命令，不链接别处 node_modules。
- 文档预检不改 tools/keywords.json 的 M4/P13；T1 首个实施提交改 M4/P14，并按实际到期例外核验，不批量延期或放宽架构守卫。
- 实体/请求直接用生成 API 类型，不反向翻译成旧 ISearchIssueResponse，不加 any/双重断言/宽 Partial 假实体。
- 生产新文件约 400 行以内；现有超长 form 按字段/关闭职责局部拆分，不复制完整表单或 FakeNerve。必要长测试表在 review 说明。
- 拒绝、已确认写后 GET 失败、SessionChanged、旧 owner 晚答分清；组件 Promise 有归属，不吞拒绝再提示成功。所有新/修改文案同步 en、zh-CN。
- 测试复用 FakeNerve、issueStoreTab、heldChange、现有 editor/issue fixture；E2E 前置通过 API，数据库只断言。条件等待有期限，不用 sleep 造时序。
- 普通命令仍沿唯一写链，权限失效由 root 统一负责；先用真实调用/证据证明局部问题再修。新架构冲突先提交已完成工作并上报。
- 每个 Task 同步要求/质量自查；对已满足的合同补测试/变异，不故意把代码回退伪造首次红。任何空匹配、编译失败、环境超时不算行为红。
- 原型/逐 Task 产品重放按 02 §3 以静态预检替代；不预写结果、计数、生成 hash 或 CI 通过结论。

## Review Focus

1. **生命周期/关闭**：id/权限/账号 ABA 与关闭重开，旧读取/finally/命令不能污染新宿主；嵌套 Escape 只被内层消费。T1/T5/T14 的真实组件与两标签页测试。
2. **表单/编辑器**：空/长标题、项目缺失、日期倒置、迁移只补 id 与真实用户编辑混合、1.5 秒内 undo→redo；聚焦时消息已经可读。T2–T4 的 DOM、真实 Editor 与 recordFocuses。
3. **并发/确认**：多子项/跨项目关系的部分成功、重复点击、POST 确认而 GET 失败、URL A→B 后旧元数据；T6–T9 的持有请求与资源回读测试。
4. **权限/范围**：OwnOnly、管理员兼访客、隐藏父/对端，公开项目非成员 403 与隐藏项 404；T6/T7/T14 的查询/页面/DB 三层断言。
5. **运行产物/跨浏览器**：lazy 的类型/ref 与加载失败、编辑器全仓 eager 读者、CSP/外链/复制失败、无图片入口；T10–T12/T15/T16 的真实 build/Network/三浏览器，禁止仅 grep 代替运行证据。

## 验证、台账与提交协议

每 Task 五步：写/扩行为测试→确认缺口或已有绿→最小实现→绿/有效变异/恢复与受影响回归→自查/台账/精确提交。台账 `.superpowers/sdd/P14-web-issue-detail/` 保存 cwd、完整 argv、tested SHA 或 dirty 输入 hash、raw/exit、非零 executed/skip、变异补丁与恢复 hash。主项目准备目录 `.superpowers/sdd/P14-preparation-20261031/` 未来也要归档。

任务中的路径是完整的 cwd 相对目标；unit 命令从 worktree 根运行 `pnpm --filter web exec vitest run <C 相对 core/... 路径>`。新测试文件与对应实现同提交，不能先运行不存在的测试把它当红。每 Task 运行 `pnpm --filter web check:types`、`git diff --check`；修改包源码先 `pnpm exec turbo run build --filter=web^... --force`，避免旧 dist 掩盖问题。

产品检查点 T4、T9、T13、最终 T16 推送前串行完整 F：

~~~sh
make gen-check
make lint-go
GOFLAGS=-p=1 make test
make lint-web
make knip
TURBO_FORCE=true make test-web
make e2e
~~~

数据库/testcontainers 套件绝不并行。默认 Firefox 仅 A4，不能冒充本段矩阵。真实页面新增到现有 I 文件或新文件后，聚焦命令 `pnpm --filter @nerve/e2e exec playwright test <stories/...> --project=chromium`，使用当次 `make build` 得到的 bin/nerve。T1–T9 新页面沿现有 expectListPageQuiet/expectQuietConsole 精确记录已有 editor 警告；不为旧接口新增失败白名单。T10 根因清理时同步删除所有既有和本段新增的 EMOJI 豁免及无用参数，最终严格零警告。T16 的新 `E/playwright.m4-detail.config.ts` 仿现有 `E/playwright.m4-filters.config.ts`，显式三 project、无 A4 grep、匹配本段页面/C 文件；逐 browser 串行运行，保留各自独立产物与实际数量。

## Task 1：入口与共同路由身份（约 350–700 行）

**Files：** spec §2.1 T1；`tools/keywords.json`；新 `C/components/browse/detail-route.test.tsx`；既有 `C/components/issues/detail-host.test.tsx`。
**Interfaces：** 沿已有 `fetchIssueCore`、`fetchIssueWithIdentifier` 与 `useSessionSWR`。不新增 endpoint；统一包含 header/body 的身份边界，key 不只放 DescriptionInput。

- [ ] Step 1：核 main 基线/干净状态与 phase 例外，读 Plane 保留入口；写 `keepsNativeIdRedirect`、`retiresHeaderAndBodyTogether`，包含 loading、拒绝、相同路由换编号及 A→B→A。核 P11 已删除项不重复记删除。
- [ ] Step 2：运行新 `core/components/browse/detail-route.test.tsx` 和既有详情 owner 回归；缺 key 的旧弹层性质须实际失败，已有原生入口通过作为基线。
- [ ] Step 3：只补共同身份/页头读取边界、必要中文文案；phase 改 M4/P14。不要在前端大小写归一编号、不要新建权限 observer。
- [ ] Step 4：M01 去共同身份失效使旧弹层测试失败，恢复；核实际 route/guard 索引、401/403/404 处理及类型/diff。
- [ ] Step 5：记录 A1 与已完成输入；提交 `feat(M4/P14): 补齐详情路由的共同身份边界`。

## Task 2：详情标题和属性、I2 属性（约 700–1100 行）

**Files：** spec T2；`C/lib/{refused-focus,form-refusal}.ts` 只复用；新 `C/components/issues/detail-fields.test.tsx`；`E/stories/issue/i2-update-issue.spec.ts`。
**Interfaces：** `IssueTitleInput` 原 props、IssueUpdate 原字段；useRefusedFocus 的 attempt 仅在失败尝试变化。Field/FieldError 的 id 描述由既有机制管理；label-select 删除 baseTabIndex 后同 Task 删实际传参。

- [ ] Step 1：写 `announcesRefusalBeforeFocus`、`doesNotMoveFocusWhileTyping` 和属性清除/双日期测试；页面通过真实输入逐项修改标题、状态、优先级、两负责人/标签及日期，记录 PATCH 与 DB。
- [ ] Step 2：运行 `core/components/issues/detail-fields.test.tsx core/components/issues/autosave.test.tsx`，失败须指向可读错误/焦点或具体属性，不以 mock 替代 FieldError 的 DOM 行为。
- [ ] Step 3：完成字段错误、标签焦点顺序、已有 dropdown 在弹窗的 modal=false/嵌套按钮缺口；同步文案与所有调用，不改变日期原子写合同。
- [ ] Step 4：I2 页面用 recordFocuses 证明空白/超长/服务端字段拒绝时焦点已经 aria-invalid+description；M02 去错误关联/M03 让错误时机提前分别红，恢复绿。
- [ ] Step 5：属性矩阵、A2 台账和精确提交 `feat(M4/P14): 完善详情字段反馈与键盘操作`。

## Task 3：描述保存与 I2 描述（约 400–850 行）

**Files：** spec T3；现有 `C/components/issues/{autosave,editor-lifecycle,editor-status}.test.tsx`；`E/stories/issue/i2-description.spec.ts`。
**Interfaces：** 保持 `DescriptionInput` 的 onSubmit/prepareSubmit 与 `useConfirmedAutosave`；只消费 `isMigrationUpdate` 生成 boolean skip_activity。

- [ ] Step 1：核已有 undo/redo 性质，补 `redoReplacesPendingUndoEvenAtConfirmedValue`、普通拒绝保持内容、旧 owner 晚答测试；设计页面真实 TipTap 编辑→撤销→重做且在 1.5 秒窗口内完成。
- [ ] Step 2：运行上述三个既有 unit 文件；新增 UI 场景经真实编辑器，不能把 textarea mock 单测称为 C4 浏览器证据。
- [ ] Step 3：仅在复现缺口时修 autosave 根因；已通过部分保留。页面记录 payload 不含 description_json，迁移 true 是布尔值，内容/链接/checked 变化不能伪装迁移免审计。
- [ ] Step 4：I2 用 DB 断言 HTML/stripped、动态/版本/时间戳，普通连续编辑与纯补 id 分开；M04 保留过时待发保存、M05 改 string flag 各检出并恢复。
- [ ] Step 5：记录 A3、M1-P2 子项；提交 `test(M4/P14): 验证描述撤销重做和迁移保存`，若有根因修复用 fix 前缀并说明。

## Task 4：创建/编辑字段与迁移、I1（约 900–1400 行）

**Files：** spec T4；`C/components/issues/issue-modal/types.ts`；现有 `{base.loading,base.clone,draft}.test.tsx`；新 `form-fields.test.tsx`、`migration.test.tsx`（同目录）；`E/stories/issue/i1-create-issue.spec.ts`。
**Interfaces：** description-editor 回调增加可选 `{isMigrationUpdate?:boolean}` 或直接转发已有编辑器 options 类型；单一表单更新函数区分用户 dirty/迁移，不扩展 DraftPayload。RHF `shouldFocusError:false`，title ref 同时交给 Controller 与已有聚焦入口。

- [ ] Step 1：写 `migrationDoesNotDirtyExistingIssue`、`migrationDoesNotDirtyExistingDraft`、`migrationPreservesRealEdits`、`projectAndTitleRefusalsHaveDescriptions`。真正观察 dirty/请求/关闭提示，不只看已有草稿直接关闭这一条。
- [ ] Step 2：运行新两文件及三个原测试；确认迁移被当用户输入的行为红。API 错误通过既有 fake error 结构，不手造与 OpenAPI 不同的 body。
- [ ] Step 3：ModalTitle、FieldError/useRefusedFocus、project 提示、正 tabIndex 清除、CreateMore 单一 Switch handler；拆 form 职责时保持既有确认后消费回调边界，已创建后消费失败不能重复创建。
- [ ] Step 4：I1 c 键打开，填写标题/描述提及/属性/日期，确认 UI 编号与 DB；空标题、倒置日期、重复提交拒绝留窗。M06 将迁移标 dirty、M07 移除同步 pending 锁各红，恢复；完整 F。
- [ ] Step 5：记录 A3/A4；提交 `feat(M4/P14): 完善工作项弹窗字段和迁移语义`，完整本地门禁后推检查点。

## Task 5：嵌套关闭和焦点（约 500–850 行）

**Files：** spec T5、T4 base/form；新 `C/components/issues/issue-modal/close-policy.test.tsx`；新 `E/stories/issue/detail-interactions.spec.ts` 的 C4 场景。
**Interfaces：** 只有一个 `requestClose():void` 本地决策，读取当前 sending/readiness/dirty 并调用现有 onClose/draft discard；外层 ModalCore 与按钮复用。若抽 hook，用 `useIssueModalClose` 作为局部实现，不暴露全局注册/事件总线。

- [ ] Step 1：写 `innerEscapeLeavesDirtyOuterOpen`、`innerEscapeDoesNotOpenDiscard`、`pendingCannotCloseOrResubmit`、`returnsFocusToConnectedTrigger`；外层无改动/有改动两种，Save this draft 与父项内层各一组。
- [ ] Step 2：真实 ModalCore 嵌套 DOM 测试确认 document Escape 的缺口；不 mock 掉 Dialog/portal。复用已有 @nerve/ui modal 测试作为基础，业务消费者另验。
- [ ] Step 3：统一关闭入口，移除或严格缩限描述编辑器 Escape；补 ConfirmIssueDiscard ModalTitle 和文案。保留当前草稿保存/拒绝，不把完整保存发布搬入本段。
- [ ] Step 4：C4 此步只验 Save this draft、日期/标签 portal、Tab 环、背景隐藏、Escape/外点、返回焦点；父项的真实页面场景依赖 T6 原生查询，放 T6 Step 4 闭合，不在此放行旧 API。M08 用嵌套 ModalCore 消费者单测检出全 document Escape 回归，恢复。T16 统一三浏览器。
- [ ] Step 5：记录 A5；提交 `fix(M4/P14): 统一弹窗关闭并隔离嵌套 Escape`。

## Task 6：原生父项选择与子项、I8（约 900–1400 行）

**Files：** spec T6 与 T4 选择器/context消费者；既有 `C/components/issues/parent-selection.test.tsx`；新 `parent-issues-list-modal.test.tsx`；`E/stories/issue/i8-sub-issues.spec.ts`。
**Interfaces：** `ParentIssuesListModal` value/onChange 改生成 Issue；所有生产调用同提交。读取复用 `useIssueSearch(root,workspaceSlug,query,open)` 的 ids/proof/loading/error/retry；onChange Promise 确认后才关闭，不拿已失效 proof 提交。

- [ ] Step 1：盘全读者含 M7 create-modal；写 `newIssueSearchOmitsForAndSource`、`existingParentSearchUsesSource`、`oldSearchCannotBeSelectedAfterProjectOrPermissionChange`、当前读取拒绝/retry 和 void/Promise 回调两类。
- [ ] Step 2：新 picker 测试与既有 parent-selection 运行红；核空词合法、新建无 id 不用 for，父项 id 不同于自身，至少两层后代用于排除断言。
- [ ] Step 3：原生 Issue 一路传递、目录显示、ModalTitle/输入名/无正 tabIndex/hover 链接；删除该转换器的零读者部分，不干掉 M7 仍需的其他旧搜索类型。子项复用既有命令、确认分布/计数。
- [ ] Step 4：I8 API 建前置，页面加已有两个/新建一个/换父/解除，核 query for、候选排除、DB 动态与计数；同时在 detail-interactions.spec.ts 闭合 T5 留下的父项 C4 页面：原生请求、背景隐藏、内层 Escape、外层 dirty/无改动均不受影响及焦点返回；M09 去 for 或 proof、M10 仅更新 UI 不确认父子分别检出并恢复。类型核 M7 调用能编译。
- [ ] Step 5：记录 A5 的父项页面、A6/A7 与 M7 边界；提交 `feat(M4/P14): 接通原生父项选择和子工作项页面`。

## Task 7：关联与收集箱查询适配、I9（约 650–1100 行）

**Files：** spec T7；既有 `C/components/issues/relations/data.test.tsx`；新 `C/components/inbox/modals/select-duplicate.test.tsx`；`E/stories/issue/i9-relations.spec.ts`。
**Interfaces：** ExistingIssuesListModal 原 `searchParams:Omit<IssueSearchQuery,"project_id"|"q">`、`handleOnSubmit(ids):Promise<SelectionSubmitResult>` 保持；M7 查重 `onSubmit(id):void` 保持，仅换原生读取/展示。

- [ ] Step 1：核关系 enum 的生成词汇与真实 helper；补跨项目/反向/部分确认、确认后 GET 拒绝只读重试；查重空词/快速词切换/权限失效/当前 error+retry 测试。
- [ ] Step 2：运行 relations/data 与新查重组件测试，确认旧 ProjectService 请求或未处理拒绝失败；不得用 M7 未实现的业务写作为本段测试前提。
- [ ] Step 3：关系 UI 用既有 root 命令闭合不足；查重用 useIssueSearch/Issue，移除旧实例；全仓复核最后两个读者消失后，删除 project.service.ts 整个零消费者类、project/index.ts 的导出、专用搜索类型和例外；P15 保留 WorkspaceService 清理，不重复删除这个文件。
- [ ] Step 4：I9 页面同项目 blocks、跨项目 relates_to、对端 blocked_by 与删边核 DB 两端动态；M11 去 for=relation/M12 重试已确认项分别红，恢复；不能把只测对端方向当任一端授权已验。
- [ ] Step 5：记录 A7 与零旧方法读者；提交 `feat(M4/P14): 完成关联页面和原生查重读取`。

## Task 8：链接表单与元数据、I10（约 800–1300 行）

**Files：** spec T8；新 `C/components/issues/issue-detail-widgets/links/use-link-metadata-refresh.ts` 及 `.test.tsx`；既有 `C/components/issues/links/data.test.tsx`；新 `C/components/issues/issue-detail/links/form.test.tsx`；`E/stories/issue/i10-links.spec.ts`。
**Interfaces：** 局部 `useLinkMetadataRefresh({workspaceSlug,projectId,issueId,active})` 消费已有 link resource，不存第二份数据；对外只提供当前等待/到期状态和 GET retry。单 owner 一次在途，候选 id+URL，30 次/30 秒上界；窗口不因 fetch 导致的 rerender 重置，用户 retry 才重开。

- [ ] Step 1：补 URL 空白/无 scheme/422/409、pending关闭；元数据晚到、空 metadata、终态成功/blocked/failed、换 URL/删除/权限失效、到期/读取拒绝/retry 测试；持有 GET 证明无重叠请求。
- [ ] Step 2：运行新增 hook/form 与现有 links/data 测试；真实组件断言 `favicon_url` 永不进入 img.src，metadata.favicon 才显示。
- [ ] Step 3：FieldError/ModalTitle/refused focus、当前 owner 的有界回读、data favicon fallback。既有 IssueResources 校验晚答；如缺 token 行为，在原资源根因修，不给 hook 加假数据补丁。
- [ ] Step 4：I10 复用真实 local server/worker，证明先地址后 title/icon、改标题/URL、删除、DB job完成；M13 直接用 favicon_url、M14 去 owner/有界停止各失败，恢复。确认后的 GET 失败只读重试可数请求。
- [ ] Step 5：记录 A8 与轮询代价；提交 `feat(M4/P14): 完善链接字段和元数据回读`。

## Task 9：删除整棵子树、I17（约 500–850 行）

**Files：** spec T9；既有 `C/components/issues/delete-issue-modal.test.tsx`；`E/stories/issue/i17-delete-issue.spec.ts`、`E/fixtures/assert/issue.ts`（复用目录断言，不复制 SQL 帮手）。
**Interfaces：** `onSubmit:()=>Promise<void>` 仍必传；保留 DeleteIssueModal 当前 scope/pending/确认跟进，不改 API 204 与子树清理合同。

- [ ] Step 1：扩 `statesWholeSubtreeWithoutLeakingHiddenCounts`、`keepsDialogOnRefusal`、`doesNotCloseNewOwnerAfterOldDelete`；API 创建非删除者数据、两层子项、归档后代及跨项目关联。
- [ ] Step 2：现有 delete modal 测试与页面新增场景确认文案/时序缺口；不把 CSS 文案快照当删除目录性质。
- [ ] Step 3：文案明确子树且 en/zh-CN 完整；必要局部修复保留同步锁。禁止先客户端枚举隐藏子项或用非空断言绕过必需回调。
- [ ] Step 4：I17 真实点击确认后 expectIssueDeleted 核所有目录、同一 deleted_at/写者与留存无关项；M15 让确认前关闭或忽略拒绝须红，恢复；完整 F。
- [ ] Step 5：记录 A9；提交 `feat(M4/P14): 明确并验证子树删除确认`，完整本地门禁后推检查点。

## Task 10：编辑器按需加载与警告根因（约 650–1100 行）

**Files：** spec T10；新 `C/components/editor/lazy-editors.test.tsx`；既有 `C/components/modals/work-item-level.test.tsx`；新 `C/components/power-k/projects-app-provider.test.tsx`；新 `E/stories/issue/editor-loading.spec.ts`。
**Interfaces：** rich/lite wrapper 的公开 props、forwarded ref、onReady/上传 handler 签名保持；可导出纯 type 到相邻 types 文件，dynamic import 只加载实际实现，Suspense 包在 wrapper 边界。失败不伪装编辑器已就绪，不允许在未 ready 时误保存空描述。

- [ ] Step 1：盘全仓 editor runtime imports、isSupported/检测读者、33 个 EMOJI 命中文件；写 `preservesRefAndReadinessAcrossLazyLoad`、`settingsDoesNotMountWorkItemModals`。用 E2E 观察真实 build 的初始及登录后请求，记录编辑器 chunk 身份来源。
- [ ] Step 2：测试 lazy 未完成时不调用编辑/保存、完成后原 ref 可用、加载错误进入已有错误边界；初始匿名网络断言由真实产物失败证明，不用仅 mock import 计数。
- [ ] Step 3：按需 wrappers、settings workspace 挂载边界；核 lite-text/index.ts → toolbar.tsx → TOOLBAR_ITEMS 的第三条 eager 链，把工具栏与其真实消费者置于同一延迟边界或去无读者导出，不保留绕过 lazy 的 barrel；删除无读者 emoji 检测/死 fallback 及确实无使用者依赖；清所有 EMOJI_CHECK_WARNING 引用/定义/注释，按新严格零警告简化 expectListPageQuiet 的无用参数并同步全调用，不能保留空参兼容层。
- [ ] Step 4：包构建后 unit/受影响 settings、S2 与 editor-loading 回归；M16 恢复 eager import 或 settings host 使 C10/组件失败；M17 恢复无读者检测使严格 console 测试失败；恢复绿。保留其他实际预期错误精确匹配。
- [ ] Step 5：记录 A10、依赖读者盘点；提交 `perf(M4/P14): 按需载入编辑器并移除死表情检测`。

## Task 11：标注块、死序列化与 CSP（约 550–950 行）

**Files：** spec T11；`P/editor/src/extensions/callout/{utils.test.ts,extension-config.test.ts}`；`P/editor/src/hooks/use-editor.ts` 的死 MarkdownStorage 类型视读者删除；`E/stories/issue/editor-loading.spec.ts`。
**Interfaces：** 保持 HTML parse/render 与 getMarkDown→convertHTMLToMarkdown；去五个扩展专用 markdown.serialize，不删除 tiptap-markdown 解析能力。默认 emoji unicode=128161、URL undefined；旧合法 icon/emoji 存储兼容按现有边界。

- [ ] Step 1：先核所有 getMarkdown/transformCopiedText 生产读者为零，列五处注册与专用 tests/helper；扩真实 Editor callout 插入/存储恢复/HTML 往返及 Markdown 粘贴断言，清点 P4 恶意属性 fixture 不丢。
- [ ] Step 2：运行 `pnpm --filter @nerve/editor exec vitest run src/extensions/callout/utils.test.ts src/extensions/callout/extension-config.test.ts src/sanitization-roundtrip.test.ts`；新增纯原生默认不得含 jsdelivr 的行为红。
- [ ] Step 3：删除死序列化与专用类型/helper；删除 fixture 的两种死 Markdown 投影字段，恶意 HTML cases 和 Go 清洗/roundtrip 输入保留。更新 HTML 测试的原生默认预期，不能泛化删除 URL 安全断言。
- [ ] Step 4：上述 tests、editor-interaction 与既有服务端 content 清洗测试回归；`make build` 后审 worker 创建/URL/blob 命中及真实页面 CSP；M18 恢复 CDN 默认须产生网络/CSP或属性断言红，恢复。记录实际产物路径与 hash，不仅源码零命中。
- [ ] Step 5：记录 A11、M1/P4 历史修复与本期死代码删除的区别；提交 `refactor(M4/P14): 清理死 Markdown 序列化并使用原生标注图标`。

## Task 12：M4 编辑器图片与未开放入口（约 300–650 行）

**Files：** spec T12；新 `C/components/issues/editor-capabilities.test.tsx`；`E/stories/issue/editor-loading.spec.ts`、detail interactions。
**Interfaces：** 沿 disabledExtensions 中已有 `image` 标识禁用两个图片扩展；M4 描述入口传配置，M6/M7 wrapper 默认不变。若共享 M4 专用配置可复用，必须列真实调用。

- [ ] Step 1：列详情/peek、创建/编辑/草稿描述的实际入口；写 `m4EditorsDisableImageWithoutChangingOtherEditors`；核附件/关注 UI 是否已经不挂载，已有零命中只作输入。
- [ ] Step 2：运行新 capabilities、现有 issue-modal/editor 生命周期测试；真实 Editor schema/slash 查询与上传 handler spy 证明未开放图片不可触发，不只是检查 prop 字符串。
- [ ] Step 3：在 M4 入口关闭 image，去因此零读者的局部上传 wiring；共享 M5 文件 service/API 保留真实其他读者，不扩大到全仓删除。
- [ ] Step 4：页面粘贴/拖入 image、slash 查询无上传、无附件/关注入口，允许文本输入保存；M19 开回 image 使真实交互测试红，恢复；未开放功能不能用空 service 假成功。
- [ ] Step 5：记录 A12 和 M5 开启前提；提交 `fix(M4/P14): 关闭未开放的工作项图片入口`。

## Task 13：子项计数入口与表格地址回归（约 350–700 行）

**Files：** spec T13 的表格列与 properties/all-properties.tsx；新 `C/components/issues/issue-layouts/spreadsheet/columns/sub-issue-column.test.tsx`、`C/components/issues/issue-layouts/properties/all-properties.test.tsx`；扩 I8 页面地址场景。
**Interfaces：** 用现有项目目录的 identifier 与 Issue.sequence_id，Link 直达 browse；不额外为生成 URL 请求 issue 详情。

- [ ] Step 1：写普通/归档工作项 `linksDirectlyToBrowseWithoutFragment`、目录未就绪无坏 URL；断言可见工作项与标识正确，非仅字符串包含 browse。
- [ ] Step 2：新 column test 在已由 P12 修好的表格上应绿；再在新 properties/all-properties.test.tsx 测真实子项计数点击，证明 all-properties 的 #sub-issues 残留行为红。核当前路由读取来源，不用 store 旧地址拼接。
- [ ] Step 3：保留已正确的表格链接；修 all-properties 子项计数入口的原生地址/无效片段，目录缺失不生成坏链接，同 Task 同步必要翻译、键盘可达性及行内事件隔离。
- [ ] Step 4：I8 页面从表格及行内子项计数打开该工作项的详情（含子项区）；M20 恢复无效片段使真实点击断言红，恢复；完整 F，包含 T10–T12 的包产物变更。
- [ ] Step 5：记录 A12、M1-P4 子项；提交 `fix(M4/P14): 让子工作项列直接打开原生详情`，完整本地门禁后推检查点。

## Task 14：访客、S2 身份、会话与 Back（约 900–1400 行）

**Files：** spec T14；`E/stories/issue/i20-guest-boundary.spec.ts`；新 `E/stories/issue/i22-session-switch.spec.ts`；`E/stories/smoke/s2-web-app.spec.ts`；既有 `C/components/modals/work-item-level.test.tsx` 与 `C/components/issues/peek-overview/host-actions.test.tsx`。
**Interfaces：** 复用 root 权限身份和 followInSession，不新建角色谓词。E2E 真实 route.fetch/fulfill，可持有已由 nerve 确认的响应，不能伪造成功体。

- [ ] Step 1：准备 OwnOnly 自己/别人及管理员兼访客、公开项目非成员；写 I20 page 与 S2 三身份。I22 两账号两标签页；DB 先确认旧写，再换账号并兑现旧答。
- [ ] Step 2：运行既有 owner/peek unit、新页面；先给已满足性质真实绿，发现局部 late follow 缺口才修，不重写权限 root 或放宽 404/403 断言。
- [ ] Step 3：闭合详情/页头/peek/WorkItemLevelModals 的必要漏点，含 Back 跨工作区/项目/id、同 id 重开、权限 ABA 和旧 finally；路由 key 不替代异步 owner 核查。
- [ ] Step 4：I20 请求/可见行/DB 与拒绝白名单精确，I22 旧响应无 toast/navigation/新页数据；M21 去会话跟进核查必须实际失败；M22 去 owner 关闭保护让晚答误关新窗须失败，恢复。
- [ ] Step 5：记录 A1/A13；提交 `test(M4/P14): 覆盖详情访客边界和跨标签页会话失效`，若有产品修复同步记录。

## Task 15：六处详情复制（约 450–800 行）

**Files：** spec T15；`C/hooks/use-copy-link.ts` 复用；新 `C/components/issues/detail-copy.test.tsx`；`E/stories/issue/detail-interactions.spec.ts`；`tools/keywords.json`。
**Interfaces：** `useCopyLink(): (link:string,copied:string)=>Promise<void>` 与 `useCopyText(): (text:string,copied:string)=>Promise<void>`；两者内部已经负责提示，不在消费者 await 后再显示成功。relations/sub-issues 已是 hook，其操作类型 copyLink 改为返回 Promise<void>，同步所有调用，不把异步函数继续标成 void。编号仅文本；链接由当前 origin/原生 browse 地址构造。

- [ ] Step 1：逐一列六消费者与 copy-hook 例外；真实按钮/菜单上 mock clipboard 的 resolve/reject，断言一次成功或一次失败，拒绝无假成功/未处理 Promise。
- [ ] Step 2：运行新 detail-copy 与现有 use-copy-link 测试；失败由当前忽略 Promise 的消费者触发，不重复写 hook 镜像测试。
- [ ] Step 3：六处迁移，交互元素有行名字/键盘 focus；同步删除这六个例外与零读者 util import，P15 的四处留下并具名记录。
- [ ] Step 4：在 HTTP 地址或受控拒绝的真实 page 触发复制失败文案，三浏览器不依赖 Chromium 专属 clipboard permission；M23 假成功/吞拒绝使消费者测试红，恢复，关键词守卫与类型通过。
- [ ] Step 5：记录 A14；提交 `fix(M4/P14): 让详情复制按实际结果反馈`。

## Task 16：组件矩阵、门禁、整审与交接（约 900–1400 行）

**Files：** 新 `E/playwright.m4-detail.config.ts`、`E/stories/issue/detail-interactions.spec.ts`、`editor-loading.spec.ts` 与本段各 page；本段 review、M4 design §12/§15、handoff 对应子项、`docs/reviews/` 下实施整审新编号。
**Interfaces：** 三个 project 的 testMatch 只含本段页面/组件目标；不能继承默认 Firefox grep。必要 page helper 在 `E/fixtures/issue-detail-page.ts`（新）复用 issue-list-page/browser/dialogs/API/assert，不能复制整套 seed/SQL/服务。

- [ ] Step 1：补齐 C3 详情七类属性、C4 全弹窗/portal/nesting/迁移、C5 父子/跨项目关系/元数据/层叠、C10 build lazy/原生 callout、C11 双账号；与 A1–A14 一一列出真实 test title，不靠空 grep 或 skip 通过。
- [ ] Step 2：完整 F 后，分别执行 `pnpm --filter @nerve/e2e exec playwright test --config=playwright.m4-detail.config.ts --project=chromium`，再 firefox，再 webkit；每次证据目录分开，收 raw、JSON executed/skip、失败截图/trace/数据库及版本。不得只把默认 A4 Firefox 两例计为本段通过。
- [ ] Step 3：M01–M23 是候选而非预填通过数，至少 16 种不同生产性质实际检出；每次保留 mutation diff/行为红/精确恢复 hash/恢复绿。遍历实际 changed files 的 dead 候选及每一读取方，删除真死成员/导出，改动文件 oxlint 零；上限只按实际净减更新，不增限盖警告。共享 helper 有真实消费者则写路径证据。
- [ ] Step 4：固定 `8bb4bbcb..最终待交付产品HEAD`，使用 deep-review 做全分支逻辑/集成/覆盖三视角，保存完整文件阅读清单。局部问题修复、必要验证和复审全部闭环；架构级先提交上报。产品修复后完整 F/受影响三浏览器补到最终产品输入；纯 docs 绑定未变输入不假称重跑。准确最终分支 CI 独立核 SHA/run/jobs。
- [ ] Step 5：逐条写 spec A 与 plan 完成证据、review 的实际决定/风险/未决，handoff 只接对应子项且顶层 open；准确提交/推送后记录后置 CI 收据。交付分支名、基线、spec/plan/review 路径，**先不合并**。提交 `docs(M4/P14): 归档详情阶段验证与整审闭环`。

## 附录 A：聚焦命令索引

以下是未来计划 argv，不是已经运行的证据；新增目标随对应 Task 创建。所有命令从 worktree 根执行。页面前先 make build；T16 矩阵配置另含 S2 的按身份详情场景，若复用大 S2 只跑其中该真实 test 或拆为同文件独立 test，不把 M7 领域页面混入。

| Task | 聚焦目标（接上述 pnpm 命令） |
|---|---|
| T1 | web vitest：`core/components/browse/detail-route.test.tsx core/components/issues/detail-host.test.tsx` |
| T2 | web vitest：`core/components/issues/detail-fields.test.tsx core/components/issues/autosave.test.tsx`；Playwright：`stories/issue/i2-update-issue.spec.ts` |
| T3 | web vitest：`core/components/issues/autosave.test.tsx core/components/issues/editor-lifecycle.test.tsx core/components/issues/editor-status.test.tsx`；Playwright：`stories/issue/i2-description.spec.ts` |
| T4 | web vitest：`core/components/issues/issue-modal/form-fields.test.tsx core/components/issues/issue-modal/migration.test.tsx core/components/issues/issue-modal/base.loading.test.tsx core/components/issues/issue-modal/base.clone.test.tsx core/components/issues/issue-modal/draft.test.tsx`；Playwright：`stories/issue/i1-create-issue.spec.ts` |
| T5 | web vitest：`core/components/issues/issue-modal/close-policy.test.tsx`；Playwright：`stories/issue/detail-interactions.spec.ts` |
| T6 | web vitest：`core/components/issues/parent-issues-list-modal.test.tsx core/components/issues/parent-selection.test.tsx`；Playwright：`stories/issue/i8-sub-issues.spec.ts` |
| T7 | web vitest：`core/components/issues/relations/data.test.tsx core/components/inbox/modals/select-duplicate.test.tsx`；Playwright：`stories/issue/i9-relations.spec.ts` |
| T8 | web vitest：`core/components/issues/issue-detail-widgets/links/use-link-metadata-refresh.test.tsx core/components/issues/issue-detail/links/form.test.tsx core/components/issues/links/data.test.tsx`；Playwright：`stories/issue/i10-links.spec.ts` |
| T9 | web vitest：`core/components/issues/delete-issue-modal.test.tsx`；Playwright：`stories/issue/i17-delete-issue.spec.ts` |
| T10 | web vitest：`core/components/editor/lazy-editors.test.tsx core/components/modals/work-item-level.test.tsx core/components/power-k/projects-app-provider.test.tsx`；Playwright：`stories/issue/editor-loading.spec.ts stories/smoke/s2-web-app.spec.ts` |
| T11 | editor vitest：任务中三个路径，加 `src/editor-interaction.test.ts`；Go：`go -C server test -count=1 ./internal/modules/issue/adapter/content`；Playwright：`stories/issue/editor-loading.spec.ts` |
| T12 | web vitest：`core/components/issues/editor-capabilities.test.tsx`；Playwright：`stories/issue/editor-loading.spec.ts stories/issue/detail-interactions.spec.ts` |
| T13 | web vitest：`core/components/issues/issue-layouts/spreadsheet/columns/sub-issue-column.test.tsx core/components/issues/issue-layouts/properties/all-properties.test.tsx`；Playwright：`stories/issue/i8-sub-issues.spec.ts` |
| T14 | web vitest：`core/components/modals/work-item-level.test.tsx core/components/issues/peek-overview/host-actions.test.tsx`；Playwright：`stories/issue/i20-guest-boundary.spec.ts stories/issue/i22-session-switch.spec.ts stories/smoke/s2-web-app.spec.ts` |
| T15 | web vitest：`core/components/issues/detail-copy.test.tsx core/hooks/use-copy-link.test.ts`；Playwright：`stories/issue/detail-interactions.spec.ts` |
| T16 | 完整 F、三浏览器各自完整 argv；实际 diff 文件 dead/oxlint、deep-review 全分支与修复复审 |

## 附录 B：准备与实施界线

本 plan 只静态预检，没有“已完成 Task”或产品测试结果。准备核对会删除不存在的旧路径、明确新增文件、验证 task/接口/调用/完成线一致，再把结果写 P14 review 与 re-20；守卫 phase 仍 M4/P13。预检后等待负责人启动 T1。
