# M5/P6 资料图片上传与展示 Implementation Plan

> 实施前预检曾通过，T1–T12完成；PC2按A完成基础与页面回归，PC3按A修复并通过联合回归，T13完整门禁收尾中。**For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans 按 Task 内联实施；不用逐 Task 新派实现者/审者。完成后一次 fresh whole-branch review。本计划在 `codex/m5-p6`，初始基线c59cedf3，实施基线已noff同步 `main@f5f4ce9431e7fb360363bf169da33487cf9b864f`；[PC1](../reviews/P6-public-project-lifetime-decision.md)已批准 A并同步合同。P5准确main CI及归档清理已完成，按连续授权从T1执行。

PC2 已批准 A：[工作区更新的迟到回包](../reviews/P6-workspace-write-lifetime-decision.md)。
**Goal:** 资料图片真实上传、宿主绑定、权限/到期展示和可访问交互。
**Architecture:** AssetService 处理会话 JSON；独立 fetch 只传 File；视图持有上传，IssueRootStore 统一使生命周期失效。图片读由当前宿主 store 调和，活动来源读合并但不复制权限或实体事实。
**Tech Stack:** 已锁 React/MobX/SWR/Vitest/Playwright、File/fetch 与既有 Go local/S3 服务；无新依赖。
**Spec:** [主合同](../specs/P6-profile-upload.md)、[接口与生命周期](../specs/P6-image-lifecycle.md)、[图片消费者](../specs/P6-image-consumers.md)；[预检记录](../../../reviews/re-42-M5P6预检.md)。

## Global Constraints

- 根目录 main，codex/m5-p6 worktree；plane/refer 只读，不在其中运行 git；详细中文提交，系统 /usr/bin/git push，--no-ff 集成。
- 不操作已有容器/Docker；容器套件串行、Go 跨包 -p 1；自己不用的临时容器按精确归属回收，禁止 prune 其他项目。
- 不加依赖/后端协议/表，不改 shared ModalCore/动效；按已批准PC1补可见性签名、PC2校验工作区写、PC3复用宿主读及身份水位窄查询，IssueRootStore 仍唯一所有者，不新加平行观察器；业务 DTO 直接用生成类型。单文件约400行，既有大文件只拆本次职责，不借题重构。
- 只支持 JPEG/PNG/WebP/GIF，File.type 为空拒绝；大小正且≤当前 file_size_limit，raw PUT 15分钟有限预算、credentials omit、redirect error、no-referrer；展示阶段进度，无虚假百分比。
- URL/expiry 不持久化到业务字段；user/workspace/project/member 每种图片重新授权走宿主，当前事实和失效代次独立核对。前端撤权不声称可撤销已发出的 S3 capability。
- 新建项目一次 POST 绑定，清除 null/省略不同；固定29封面 cover-01…cover-29，自制 SVG 登记来源，无随机渲染或 Vite URL 入库。
- 旧 FileService/services 包在 P7 最后调用者迁移时删除；本阶段新控件不导入它们。M5 完成条件与关键词守卫不放宽。
- RED 是行为失败，禁止把缺类型/编译失败当产品RED。测试输入全量hash；代码/生成物稳定时才跑完整七门禁，不与Vitest并跑清dist的构建/types。
- 准确分支/main CI 分开核，W11按既有一次重跑规则；原失败/skip/缓存与完整原件保留。M5全体验收及handoff不提前勾选。

## Review Focus

1. PUT 响应丢失或 complete 已成功但响应丢失时，明确重试不能新建重复上传链或用原 key 重传；取消后迟到成功不能继续绑定（T1/T2/T12）。
2. 管理员失权又恢复、同一 URL 换账号、旧项目详情仍在、全局成员缓存含另一个工作区头像时，旧能力地址和旧请求绝不能复活（T2/T3/T10/T12）。
3. 刷新读取与资料换图/移除交错、多个头像同刻过期时，应保留最新确认状态并合并来源读，不产生反复刷新和图片错误循环（T3/T5/T10）。
4. 新建项目切换自定义/预设/移除、上传中按 Escape/Back、双击创建或未知响应时，不能半建项目、误关父弹窗或把旧文件绑定新工作区（T4/T7/T8/T12）。
5. 公共邀请与访客、后台标签页 timer 延迟、图片错误晚到和组件重复清理时，既不多取禁止接口，也不泄露旧图/保留监听或发后台删除（T3/T10/T12）。

## 文件与接口

未带仓库前缀的 core/ 与 helpers/ 路径均相对 `web/apps/web/`；app/ 同样相对该 web 应用，e2e/ 与 tools/ 相对仓库根。

接口只在 P6-image-lifecycle.md 声明，Task 不抄实现体。新服务 asset.service.ts；lib/asset-upload.ts；store/asset/{root.store,upload-session,scope-lifetime,profile-images}.ts；hooks/use-profile-image.ts；components/assets/{image-picker,image-upload-modal,profile-image}.tsx。正式实现时同目录新增对应 .test.ts(x)。UserStore 局部改用 Reconciled，不改变公共原语；RootStore 注入唯一会话 API。

既有显示消费者以正式预检保存的完整 rg 清单、49 项来源映射和 7 处透传调用链逐项迁移：个人设置/引导/用户菜单/侧栏、工作区菜单/设置/选择器、项目创建/设置/卡片、成员表/下拉/作者/历史/评论/资料页及其复用头像。M6/M7 尚未接通页面只换通用呈现来源，不新增它们的 API 或功能。

## Task 1（T1）：JSON 资产服务与独立 File PUT

**Files:** 新 core/services/asset.service.ts/.test.ts、core/lib/asset-upload.ts/.test.ts；tools/keywords.json；P6 review。
**Interfaces:** 产出 AssetService.prepare/complete 和 putAsset，直接消费生成 NewAsset/PreparedAsset/Asset；失败分类见子合同。

- [x] Step 1：服务测试实际记录 URL/body/signal 与 problem 原值；传输测试覆盖local根相对地址与S3绝对地址、同源/跨源、签名query原值、重复/额外headers与原File、2xx/403/500、网络、过期/非法 grant、父取消/超时竞争、redirect 拒绝。不能在 jsdom 声称已验证浏览器 Cookie，真实证据在 T11/T12。
- [x] Step 2：`pnpm --filter web exec vitest run core/services/asset.service.test.ts core/lib/asset-upload.test.ts` 记录行为 RED 后最小实现；phase=M5/P6。
- [x] Step 3：相同命令 GREEN，取消后无挂监听/计时器，错误不包含签名 query；受影响 types/lint/格式，提交 `feat(M5/P6): 接入独立文件直传与资产确认`。

## Task 2（T2）：上传会话与统一权限失效

**Files:** 新 core/store/asset/root.store.ts、upload-session.ts、scope-lifetime.ts 及测试；修改 core/store/root.store.ts/.test.ts、core/store/issue/permission-reads.ts；新增 core/store/issue/permission-reads.test.ts，回归 issue/root.store.test.ts 和 project/project.store.visibility.test.ts。
**Interfaces:** 消费 T1；产出 createUpload、UploadSession 和 root.dispose 接线；宿主写留调用页面。

- [x] Step 1：真实 deferred promise 测试四用途scope、大小/MIME拒绝、双击仅一链、逐阶段 retry、object_missing 新 prepare、旧成功/失败/进度、select 替换、scope/路由/root 退休及权限 ABA；user 不受无关 project 影响，workspace 不受别的 project 影响。实际 RootStore/ProjectStore 驱动公开非成员完整目录缺席/恢复：旧 token 永久 false、新 token current，资源 invalidate 一次；有效角色仍 undefined，受限 issue 仍拒绝，成员/guest 收窄和无变化目录/无关范围保持。
- [x] Step 2：`pnpm --filter web exec vitest run core/store/asset core/store/root.store.test.ts core/store/issue/permission-reads.test.ts core/store/issue/root.store.test.ts core/store/project`，取得断言 RED 后实现；沿 PC1 扩展现有观察器项目签名，仍只经 IssueRootStore.registerResource 接入，每个await核生命周期。不改 UserPermissionStore 或 ProjectStore 的事实算法。
- [x] Step 3：GREEN；取消/重复dispose释放 objectURL/AbortController，无 cleanup DELETE、无旧 owner callback；单改取消后继续 complete 与删除代次检查应各导致行为失败，恢复 SHA。types/lint 后提交 `feat(M5/P6): 约束上传的会话与权限生命期`。

## Task 3（T3）：宿主图片到期与读取调和

**Files:** 新 store/asset/profile-images.ts/.test.ts、hooks/use-profile-image.ts/.test.tsx、components/assets/profile-image.tsx/.test.tsx；修改 store/user/index.ts/.test.ts；root asset 生命周期接线。
**Interfaces:** 产出 ProfileImageSource、useProfileImage、ProfileImage/ProfileAvatar；消费现有 owner store 和 Reconciled，不增通用权限框架。

预设资源由T7产出，T3先完成项目能力读取及空图/预设不额外GET；T9组合预设映射和能力展示，不在T3加入临时映射或前向导入。

- [x] Step 1：deferred 与受控时钟复现旧 /me read 覆盖较新 avatar/cover write；同源多头像只读一次、两个workspace同member不共享授权、失权立即无img；公开项目非成员经实际ProjectStore接受缺席/恢复后旧封面读取不能发布，新读取才可显示；ABA旧读失效、expiry/background/invalid新expiry、onError仅一次、最后引用释放、重复cleanup与新image旧error。
- [x] Step 2：`pnpm --filter web exec vitest run core/store/asset core/hooks/use-profile-image.test.tsx core/components/assets/profile-image.test.tsx core/store/user/index.test.ts core/lib/reconciled.test.ts`，RED 后实现局部调和及活动来源合并。
- [x] Step 3：GREEN；原 UserStore 全测试/退出引导、ProjectStore目录事实回归；变异绕过expiry、使用memberMap裸地址、直接旧read覆盖write均须真实失败。记录来源读取成本与无无限重试，提交 `feat(M5/P6): 按宿主权限刷新过期资料图片`。

## Task 4（T4）：通用图片选择内容与弹窗

**Files:** 新 components/assets/image-picker.tsx/.test.tsx、image-upload-modal.tsx/.test.tsx；en/zh-CN 新文案；不改 shared ModalCore。
**Interfaces:** 消费 UploadSession 与 scope 结果；产出原生 file/drop 内容组件和 ModalCore 薄包装；宿主确认busy与上传busy分开。

- [x] Step 1：键盘选择、单文件drop、多文件/目录/空MIME/超限拒绝、预览替换/revoke、可读状态/role alert、上传中Escape/背景不关、明确取消上传、owner绑定中双击/关闭拒绝、成功/拒绝/卸载焦点。
- [x] Step 2：`pnpm --filter web exec vitest run core/components/assets` RED→GREEN；原生事件，无新包，无 render 内 createObjectURL，无固定sleep或延迟reset。
- [x] Step 3：类型/格式/双语言键/修改文件零warning；提交 `feat(M5/P6): 提供可取消且可访问的图片选择控件`。

## Task 5（T5）：个人资料与引导

**Files:** settings/profile/content/pages/general/form.tsx、onboarding/steps/profile/root.tsx 及原测试；新增 workspace/sidebar/user-menu-root.test.tsx；settings/profile/sidebar/header.tsx、workspace/sidebar/user-menu-root.tsx、onboarding/switch-account-dropdown.tsx、home/widgets/empty-states/no-projects.tsx。
**Interfaces:** 单独 UserUpdate.avatar_asset_id/cover_image_asset_id 绑定或null；读统一 ProfileImageSource self。

- [x] Step 1：既有页面测试加入头像/封面独立上传/移除，名字表单不误清图片、上传不提前提交引导；绑定失败保留ready件重试不PUT，两个图片控件和并发读/写正确，换账号旧回调无toast/跳步。
- [x] Step 2：`pnpm --filter web exec vitest run core/components/settings/profile core/components/onboarding core/components/workspace/sidebar/user-menu-root.test.tsx` RED→GREEN；复用T4，移除本次变成无用的裸URL/getFileURL导入。
- [x] Step 3：当前UserStore/引导/认证回归，记录实际 UI 与 payload，提交 `feat(M5/P6): 恢复个人头像封面与引导上传`。

## Task 6（T6）：工作区图标

**Files:** workspace/settings/workspace-details.tsx/.test.tsx、workspace/logo.tsx 及测试、workspace/sidebar/{dropdown-item,workspace-menu-root}.tsx、settings/workspace/sidebar/header.tsx、power-k/menus/workspaces.tsx、settings/profile/sidebar/workspace-options.tsx；同步account/auth-forms/auth-header.tsx的public-preview直接调用形状，完整邀请行为仍归T10。
**Interfaces:** WorkspaceUpdate.logo_asset_id；workspace source UUID，不把slug作为资源身份。

- [x] PC2先行：按已批准A修复WorkspaceRootStore的调用/排队/回包发布及原始目录重放。新增store/workspace/write-lifetime.test.ts，先保留四场景复现并扩展20→5/同UUID恢复/失败迟到/root退休/排队/旧读重放/无关范围；小接口由root.store.ts接IssueRootStore的窄工作区查询，PermissionReads仍唯一观察者。updateWorkspace过时返回undefined，现有普通资料表单不显示完成toast；原有有效拒绝仍抛出。联合workspace/issue/root/reconciled回归与删除发布检查的行为变异通过后，继续下列T6页面步骤。允许本Task拆成PC2修复与页面两个提交。
- [x] Step 1：管理员替换/null清除，15/5只读无控件，失败保留原图，跨工作区/同slug新UUID/失权恢复旧回包不能复活；菜单多logo并发过期只刷新一次目录；地址刷新不重置未保存资料字段，真正切换工作区仍同步表单。
- [x] Step 2：`pnpm --filter web exec vitest run core/components/workspace core/components/settings/workspace core/store/workspace` RED→GREEN；WorkspaceLogo不保留裸URL绕过新入口，公开邀请适配在T10同时完成前不推检查点。
- [x] Step 3：工作区导航/菜单及权限回归、类型lint，提交 `feat(M5/P6): 接通管理员工作区图标与安全展示`。

## Task 7（T7）：稳定的项目预设封面

**Files:** 新 helpers/project-covers.ts/.test.ts、app/assets/project-covers/cover-01.svg…cover-29.svg/SOURCES.md；新增 components/assets/project-cover-picker.tsx/.test.tsx。
**Interfaces:** 产出稳定 PROJECT_COVERS、projectCoverURL 与只携现有DTO封面字段的选择值；本地预设不上传。

- [x] Step 1：完整29唯一ID、每个资源可构建、未知ID占位；未选择/上传/预设/显式清除状态不混淆；随机仅点击一次、rerender不变、不持久化路径、切预设取消旧上传链。
- [x] Step 2：`pnpm --filter web exec vitest run helpers/project-covers.test.ts core/components/assets/project-cover-picker.test.tsx` RED→GREEN；创建自制SVG并登记，视觉检查不同尺寸/对比。
- [x] Step 3：构建核所有29资源、无第三方标志/外部请求，提交 `feat(M5/P6): 提供稳定编号的项目预设封面`。

## Task 8（T8）：新建项目一次绑定

**Files:** projects/create/{root,use-create-project,utils}.ts(x)、project/create/{header,project-create-buttons}.tsx、project/create-project-modal.tsx 及原/新增测试。
**Interfaces:** ProjectCreationForm 显式包含 cover_image_asset_id/cover_builtin_id 的省略与null；上传scope=new_project/workspaceUUID，创建仅一次POST。

- [x] Step 1：成功按所选值一次POST零PATCH，prepare/PUT/complete失败零Project POST；绑定拒绝重试不重传，上传busy与createbusy都不能双提交/隐式关闭；自定义切预设、移除、Back/换workspace、迟到成功及未知创建响应不自动再POST。
- [x] Step 2：`pnpm --filter web exec vitest run core/components/projects/create core/components/project/create-project-modal.test.tsx` RED→GREEN；封面选择内容内嵌原弹窗，移除本地旧300ms reset，维持后续feature-selection。
- [x] Step 3：既有project创建/标识冲突/lead候选与键盘回归，类型lint，提交 `feat(M5/P6): 在项目创建请求中一次绑定封面`。

## Task 9（T9）：项目设置与卡片展示

**Files:** project/form.tsx、新form测试、project/use-update-project-details.ts/.test.ts、project/card.tsx/.test.tsx；必要的helpers/cover-image.helper.ts、common/cover-image.tsx仅清除本次过时注释/接口，不侵入其他外部URL用途。
**Interfaces:** 已有ProjectUpdate单一非null或两个null；ProjectCover使用project source与稳定映射。

- [x] Step 1：管理员真实自定义/预设/清除，普通成员/guest只读、archived禁写；省略保持其他字段、绑定拒绝旧封面保持；完整目录移除后的旧详情/迟到详情不能显示图。
- [x] Step 2：`pnpm --filter web exec vitest run core/components/project/form.test.tsx core/components/project/card.test.tsx core/components/project/use-update-project-details.test.ts core/store/project` RED→GREEN。
- [x] Step 3：卡片/表单当前配置与权限回归，提交 `feat(M5/P6): 更新项目封面并保持目录权限事实`。

## Task 10（T10）：全部既有头像消费者与邀请例外

**Files:** 正式预检P6-image-consumers清单中全部头像/图标调用方；包括workspace/project/settings/member-columns、dropdowns/member/{avatar,base}、project/member-select、project/add-project-members-modal、profile/sidebar、comments/card/display、core/description-versions、editor/embeds/mentions/user及仍保留的通用筛选/显示；use-invitation-preview.ts/.test.ts、account/auth-forms/auth-header.tsx。
**Interfaces:** 成员source必须明确workspaceUUID/memberUUID；public邀请沿现有link key与expires_at，不需资产UUID或登录。

- [x] Step 1：成员/自身/guest与inactive原语义、两个workspace同member旧全局缓存、私有project撤权/恢复、公开邀请过期/失效/更换、多个avatar同来源合并、旧onError不能刷新新图；确保没有新增未实现M6/M7请求。
- [x] Step 2：按完整清单逐个迁移到T3显示入口，测试当前可达caller与旧props适配；历史作者沿root/dropdown传工作区UUID，编辑器mention保留捕获范围，进度统计使用成员ID而非投影裸URL。SwitcherLabel删除无调用者的logo_url分支并核两处图标caller；直接裸URL命中须逐行归零或具名登记确为非资料外部资源，不增加关键词豁免。
- [x] Step 3：`pnpm --filter web exec vitest run core/components/dropdowns/member core/components/workspace core/components/project core/components/profile core/components/comments core/components/core/description-versions core/hooks/use-invitation-preview.test.ts core/hooks/work-item-filters`及全types/knip；提交 `feat(M5/P6): 统一有权成员头像与邀请图标的到期展示`。

## Task 11（T11）：F5–F7 真实页面与双存储

**Files:** e2e/stories/files/f5-profile-images-page.spec.ts、f6-workspace-logo-page.spec.ts、f7-project-cover-page.spec.ts；复用/扩展fixtures/file-story.ts、storage.ts、auth.ts、test.ts、s3-world.ts；不复制启动/清理框架。
**Interfaces:** pageFileStory 的最小world包含api/db/nerve/baseURL/page和既有local/S3字节断言，普通函数后薄fixture。

- [x] Step 1：界面选真实File、PUT/complete/owner请求严格记录，最终DB引用、对象/磁盘hash相符；个人/工作区/项目至少各local成功、S3资料上传至少一完整页面。失败不能只看DOM文件名。
- [x] Step 2：构建后 `cd e2e && pnpm exec playwright test stories/files/f5-profile-images-page.spec.ts stories/files/f6-workspace-logo-page.spec.ts stories/files/f7-project-cover-page.spec.ts --project=chromium --workers=1`；重复确认后无双绑定，项目只有一创建，旧API请求0，真实同源Cookie探针与凭据隔离。
- [x] Step 3：保存report/trace、API/DB/bytes证据，核新增容器用完无遗留；提交 `test(M5/P6): 验证资料图片真实上传和一次绑定`。

## Task 12（T12）：F16/F17 生命周期、可访问性和三浏览器

**Files:** 新 e2e/stories/files/f16-profile-upload-lifetime.spec.ts、f17-profile-upload-accessibility.spec.ts、playwright.m5-profile.config.ts；相关受影响unit；只在P6config扩展本阶段三浏览器，不扩张默认CI全部浏览器。
**Interfaces:** 按条件等待/有期限阻塞网络，不固定sleep；复用签名能力/ownerAPI，不伪造服务端TTL配置。

- [x] Step 1：三个浏览器各真实file input/drop、rawFile传输、取消、焦点/aria；按prepare/PUT/complete/owner四边界保持真实请求暂停，换账号/Back/Forward/换workspace/项目/撤权恢复，恢复旧成功/失败后无错误绑定/旧图。
- [x] Step 2：`make build build-s3-fixture` 后 `cd e2e && pnpm exec playwright test --config=playwright.m5-profile.config.ts --workers=1`；expiry注入标明只测UI，图片每来源仅一次自动再读，manualretry只读。
- [x] Step 3：至少9个有效行为变异：带凭据、取消后complete、漏lifecheck、删除PC1非成员可见事实、旧read覆盖write、expiry失效、memberMap当权限、双POST、自动retry循环；逐SHA恢复。跑全部受影响unit/类型/格式；保存原件与容器清理，提交 `test(M5/P6): 证明上传迟到结果与可访问交互边界`。

## Task 13（T13）：完整门禁与检查点

停点：[PC3目录读取冲突](../reviews/P6-directory-read-decision.md)已复现并批准A，A3联合回归已关闭；完整Go原证据经输入hash核对复用，PC3前端门禁及42项三浏览器通过，完整E2E进行中。修复前失败原件保留。

- [x] Step 1：逐项A1–A6、消费者清单、设计/plane-diff/handoff处理结果；P7删包责任保持open，旧未使用成员实测，不因历史数字机械删除。
- [ ] Step 2：串行 make gen-check、make lint-go、GOFLAGS=-p=1 make test、make lint-web、make knip、make test-web、make e2e（已依赖完整build及build-s3-fixture）；每项原exit/输入hash/缓存/skip记录，非仅聚焦代替全门禁。
- [ ] Step 3：详细中文检查点提交/system push，准确HEAD server/web/e2e三项全绿；失败自行取原生插件/已登录浏览器日志，按现有规则处理，不放宽超时/断言。

### T13 PC3修复步骤（负责人已批准A）

**Files:** store/asset/{profile-image-read,profile-images,image-source}.ts；hooks/use-profile-image.ts、use-workspaces-fetch.ts、use-workspace-members-fetch.ts；新增hooks/use-current-user-fetch.ts、use-project-read.ts；authentication-wrapper.tsx和use-project-fetch.ts消费提取hook；store/root.store.ts和user/workspace/project/member-workspace的读取边界；permission-reads.ts的身份窄查询及实际测试。
**Interfaces:** 图片消费侧ObserveProfileRead<T>(source, read:()=>Promise<T>):Promise<T>，source只标self/workspace/projectID/memberWorkspaceID；RootStore组合接线，结果原样返回。ProfileImages记录在途与每视图证明，不调用HTTP；ImageSource只投影规范源。四个canonical hook保留既有会话键。PermissionReads补身份水位查询供工作区图标，不改角色判定。

- [x] PC3-1：先将原两条探针转为正式行为回归，扩展四宿主与页面并发、已完成缓存/在途加入、较新成功/失败取代、共享失权ABA、卸载/root退休，记录真实RED，不能只测无关键词命中。
- [x] PC3-2：实现小读取观察接口及RootStore接线；四宿主读取边界发布真实完成，提取/复用canonical hook，删除图片独立读取方法和profile-image键。共享刷新仅合并通知，不建立新HTTP所有者。原单元夹具按新的host驱动合同更新，既有授权反例保留。
- [x] PC3-3：实际workspace/issue/project/member/user与hook联合回归GREEN；定向去掉同键复用、旧完成屏障、权限代次分别应行为失败并逐SHA恢复；三浏览器P6及受影响单元/类型/lint/格式通过，原目录守卫逐值不变。
- [ ] PC3-4：修订A3证据和阶段状态，中文提交；完成本轮七门禁及准确分支CI，再执行T14/T15。不重复无关已绿套件，复用时逐hash明确输入差异及不受影响范围。

## Task 14（T14）：整分支 deep-review

- [ ] Step 1：冻结真实基线到候选HEAD，一次fresh reviewer读完整变更手写文件、调用链、全部Review Focus、局部裁定与输入证据；生成物核源。
- [ ] Step 2：主控按实际影响分级，一次局部修复与必要RED/GREEN/受影响回归；Minor具名去向，重要架构问题先提交上报；不派重复同范围审者。
- [ ] Step 3：报告入docs/reviews下一编号，最终提交/push准确CI全绿，关闭A7。

## Task 15（T15）：main 集成归档与 P7 预检

- [ ] Step 1：核根main/W干净和祖先、准确green最终HEAD，连续授权下--no-ff，双父/树一致、system push。
- [ ] Step 2：准确main CI三项全绿后归档全部ignored原件/raw失败/浏览器/变异/审者；hash/bytes/mode/membership/unknown=0；origin/main..branch=0后正常清远端、本地和worktree，归档再次核。
- [ ] Step 3：下一codex/m5-p7独立worktree回填A8/阶段事实；P7 spec/plan一次builder预检，包含最后旧FileService/services删除、编辑器/附件与计数控件，不提前标M5完成。

## 预检自查（已通过）

A1→T1/T2/T11；A2→T2/T4/T12；A3→T3/T10/T12；A4→T5/T11；A5→T6–T9/T11；A6→T11/T12；A7→T13/T14；A8→T15。15 Tasks。源码路径/接口、角色集合、字段拼写、测试命令、builder唯一问题与跨Task接口已核；实际源码与126项宿主基线一致，P5准确main CI及归档清理前置满足。完整收据见[预检§7](../../../reviews/re-42-M5P6预检.md#7-最终基线与执行前预检通过)。该预检时点P6八项验收均未完成；实施后的当前状态以主spec和阶段记录为准。
