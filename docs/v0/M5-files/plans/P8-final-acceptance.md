# M5/P8 最终验收矩阵、清理演练与交接收敛 Implementation Plan

> 2026-11-06：正式预检通过，暂不实施；一次fresh builder的6项局部问题已补实到spec§3与下列步骤，见[re-51](../../../reviews/re-51-M5P8预检.md)。分支`codex/m5-p8`，基线main`2b2c6f33e3a93536742d5f28b74eb959ba73d6d1`；P7b准确分支/main CI、归档与清理已核。

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans 逐Task内联实施；不逐Task新派实现者/审者，阶段末一次fresh whole-branch review。

**Goal:** 在最终代码上给F1–F18最终证据，完成清理/慢流演练与默认部署演示，修掉P7b交下的Minor与遗留，收敛死成员/告警/死素材，写完交接，为closeout备齐材料。
**Architecture:** 不改资产协议与权限来源；编辑器传输改为按节点身份而非节点视图持有；其余为页面层根因修复、删除与证据。
**Tech Stack:** 已锁定 React/MobX/SWR/Tiptap、Vitest/Playwright、Go bootstrap 测试与既有 local/S3 夹具；不加依赖。
**Spec:** [P8主合同](../specs/P8-final-acceptance.md)。

## Global Constraints

- 每个产品修复先有实际断言的RED再GREEN；缺模块/类型/环境失败不算行为RED。必要变异记录恢复SHA。修改文件oxlint零warning（先跑oxlint再提交）。
- 实施开始时在worktree执行`pnpm install --frozen-lockfile`（附录A的`lib/program.mjs`按锁定的TypeScript路径读取）；不新增或升级依赖。
- 第一个实现提交把`tools/keywords.json`的phase设为`M5/P8`；closeout开始时才设`M5/closeout`。
- 跨M修复（M4详情弹窗提示）单独提交、可单独回退，在阶段记录、`frontend-changes`与对应交接登记。
- 代码/生成物稳定才串行七门禁；容器套件一次一个、Go跨包`-p 1`；只清理本阶段确切拥有的临时资源，不操作既有容器。已绿后仅因改动/失败扩大或重跑。
- 新增默认E2E故事前核CI余量；三浏览器矩阵与演示用独立config，不进默认`make e2e`。

## Review Focus

1. 传输跨节点视图（T3）：拖动、包裹块等节点仍在文档中的重建期间上传/复制不丢；节点删除/换`id`/Undo撤掉插入、编辑器销毁、`images.signal`结束才取消；Redo后不静默空白、不自动重发；结果不能写到别的节点。
2. 粘贴（T2）：纯非图片、图片+文字、被拒图片批、伪造编辑器剪贴板：不吞文字、不让原生粘贴插入外部`<img>`或文件名、不越权复制。
3. 删除核验与提示（T5/T6）：不伪造204/成功；未知结果不自动再发；提示与宿主scope一致，旧scope不发提示。
4. 删除面（T7/T8）：删除的成员/素材/字段确无读者（含展开传入、第三方回调、字符串引用、样式资源）；告警修复不改运行行为。
5. 演练与矩阵（T10–T12）：精确数据而非DOM文字；邻域不受影响；缓存/跳过/重跑如实；mock不冒称真实存储。

## 文件与接口约定

| 组 | 文件责任 |
|---|---|
| 编辑器 | `web/packages/editor/src/plugins/drop.ts`、`extensions/custom-image/{paste.ts,components/use-image-transfer.ts}`；测试`src/editor-images.test.tsx` |
| 应用适配 | `core/hooks/editor/editor-assets.ts`（只在T3确需时）；`core/components/editor/rich-text/description-input/root.tsx` |
| 附件/宿主 | `core/store/asset/content-deletion.ts`、`core/components/assets/attachments/delete-dialog.tsx`、`core/components/issues/detail-host.tsx`、`issue-modal/{base,content-copy}.tsx`、链接/子项/关联弹窗表单 |
| 删除面 | `web/packages/editor/src/{types/editor.ts,helpers/editor-ref.ts}`、`core/components/core/description-versions/modal.test.tsx`、`web/apps/web/app/assets/attachment/`、`server/internal/modules/workspace/domain/workspace.go` |
| 告警 | `core/hooks/use-local-storage.tsx`、`core/lib/local-storage.ts`、`core/components/settings/helper.ts`、oxlint配置、`web/apps/web/package.json`的cap |
| 夹具 | `e2e/fixtures/{s3.ts,s3-process.ts,s3-container.ts}` |
| 演练/演示 | `server/internal/bootstrap/`下新演练测试；`e2e/playwright.m5-demo.config.ts`与`e2e/stories/files/f18-default-deployment.spec.ts` |
| 文档 | README部署节、`server/configs/`样例注释、`docs/v0/{plane-diff,frontend-changes}.md`、`app/assets/*/SOURCES.md`、七份输入交接、`docs/v0/M{6,7,8}-*/handoffs/M5-closeout.md`、阶段记录 |

## Task 1（T1）：范围、证据目录与承接映射

**Files:** 新`docs/v0/M5-files/reviews/P8-final-acceptance-review.md`（阶段记录）；读P1–P7b阶段记录、各spec、七份输入交接、re-33至re-50。

- [x] Step 1：F1–F18逐条列出现有证据（阶段、测试/故事名、收据）与本阶段最终重跑落点（T10/T11/T12）；缺口写清由哪个Task补。七份输入交接逐条款列当前事实（代码/测试路径）与P8动作（已完成待最终核/本阶段修/具名转交），再列re-50 Minor、P7b记录§9–§10的P8项与T15/T16观察。
- [x] Step 2：核目录完整：每个F、每个条款、每个承接项恰好一个落点；无产品改动。链接用本阶段工作区的`doc-links.py`（预检时由会话脚本复制，记SHA-256；覆盖M5文档、re-31起报告、M6–M8交接、frontend-changes、plane-diff、v0-design、README，并按GitHub规则核锚点），现存的re-32→P1决策§3与re-41行号锚两处断链登记给T13。
- [x] Step 3：中文提交`docs(M5/P8): 固定最终验收与交接目录`。

## Task 2（T2）：编辑器粘贴只在图片批时消费事件

**Files:** 修改`web/packages/editor/src/plugins/drop.ts`；测试`src/editor-images.test.tsx`与`src/fake-editor-images.ts`（`pasteFiles`加文字载荷参数）；`docs/v0/M5-files/specs/P7b-editor-images.md` §3与§5两处措辞；`tools/keywords.json` phase。

- [x] Step 1：改写原用例“does not swallow rejected %s files”（:37-62）并新增：①只含非图片文件且附`text/plain`的粘贴，插件不处理，文字经ProseMirror插入恰好一次、无提示；②mixed/directory/empty/svg/too-many被拒批`defaultPrevented=true`、文档不变、`role=alert`恰好一次；③图片批照旧插入；④伪造`text/nerve-editor-html`仍不越权复制；⑤拖放的被拒/普通文件仍不拦下。先`pnpm --filter @nerve/editor exec vitest run src/editor-images.test.tsx`看到①②按断言失败（①文字未插入，②未拦下）。
- [x] Step 2：实现§3第1条：`accept`返回普通/拒绝/接受三态，`handleDOMEvents.paste`按三态分支，`handlePaste`与`handleDrop`不变；改写P7b编辑器子合同两处；phase=M5/P8。变异“被拒批不preventDefault”“普通批返回true”须被杀死。
- [x] Step 3：同命令GREEN，编辑器包全部测试与类型、oxlint零warning；提交`fix(M5/P8): 编辑器粘贴只在接受或拒绝图片批时消费事件`。

## Task 3（T3）：图片传输跨节点视图重建存活

**Files:** 修改`extensions/custom-image/components/use-image-transfer.ts`、`extensions/custom-image/paste.ts`（必要时`core/hooks/editor/editor-assets.ts`）；测试`src/editor-images.test.tsx`、`core/hooks/editor/use-editor-assets.test.tsx`。

- [x] Step 1：用受控图片端口（`fake-editor-images.ts`）写用例：上传/复制进行中①拖动块越过视图匹配范围（沿`editor-images.test.tsx`现有`view.dragging`+drop技法，一次事务删加）②把块包进引用块再解除，传输不取消、新视图显示原阶段、完成后同一`id`节点得到`src`；③Undo撤掉插入：该任务结束时取消、结果不写回，Redo恢复的节点显示“传输已取消，请重新选择”且不自动重发；④节点被删除或换`id`时取消且结果不写回；⑤编辑器销毁与`images.signal`结束取消。先看到①②失败（取消被调用、新视图显示空选择）与③Redo后无提示。
- [x] Step 2：实现§3第2条：传输记录（控制器、阶段、结果）提到扩展存储（`custom-image/types.ts`的`CustomExtensionStorage`），视图卸载只解除订阅；当前任务结束时文档已无该节点才取消；完成按`id`与`src`为空定位写回；已取消id在Redo后显示可见状态（en/zh-CN）。应用适配`editor-assets.ts`只在测试证明需要时改。变异“视图销毁即取消”“按旧getPos写回”“Redo后自动重发”须被杀死。
- [x] Step 3：同命令与应用适配测试GREEN，编辑器包与`use-editor-assets`全部测试；提交`fix(M5/P8): 图片传输随节点而非节点视图存活`。

## Task 4（T4）：核验保留输入不重建编辑器

**Files:** 修改`core/components/editor/rich-text/description-input/root.tsx`（`recover()`与`editable`、`onChange`守卫）；测试`core/components/issues/body-images.test.tsx`（保留输入与图片的既有用例所在）。

- [x] Step 1：用例：恢复保留输入后插入新图片（上传中），点“保存保留的输入”：编辑器实例不变、上传不取消、核验期间不可编辑且可读出忙状态，上传完成后可再保存；核验失败/422时输入仍在。先看到上传被取消（RED）。
- [x] Step 2：实现§3第3条：核验期间以`inert`包裹并在`onChange`守卫`verifying`，不切换`editable`（`use-editor.ts`以`[editable, images.signal]`重建编辑器）；变异“核验时切换editable”须被杀死。
- [x] Step 3：受影响宿主/正文测试GREEN；提交`fix(M5/P8): 核验保留输入时不重建编辑器`。

## Task 5（T5）：删除核验404出口与收据寿命

**Files:** 修改`core/store/asset/content-deletion.ts`、`core/components/assets/attachments/delete-dialog.tsx`、`core/components/issues/detail-host.tsx`；测试附件宿主`issue-detail-widgets/attachments/host.test.tsx`、`detail-host.test.tsx`；en/zh-CN文案。

- [x] Step 1：用例：DELETE回答丢失→核验得404：对话框只留关闭、说明不可见、无DELETE再发、关闭后无“删除未能确认”残留；核验得200照旧可显式再确认；收据在同宿主新命令开始或正文重挂载且焦点归还后清除、读屏只读一次。先看到再发DELETE与提示残留（RED）。
- [x] Step 2：实现§3第4条：`verify()`给出逐id结果（不可见的id离开`uncertain`），对话框据此只留关闭并说明（新增en/zh-CN键）；收据在`IssueDetailInputState.confirm()`开始新命令或`takeFocus()`归还焦点后的正文重挂载时清除。不伪造204/收据。变异“404仍留uncertain”“收据常驻”须被杀死。
- [x] Step 3：GREEN与受影响附件/宿主测试、i18n同步；提交`fix(M5/P8): 删除核验得404时结束未知状态，收据只报最近结果`。

## Task 6（T6）：副本未知发布与详情弹窗的提示

**Files:** 修改`issue-modal/base.tsx`（`finishClose`；`content-copy.tsx`仅需时）；`core/components/issues/issue-detail/links/create-update-link-modal.tsx`（成功提示改由宿主输入状态scope判定，经`useIssueDetailInput()`）；测试`issue-modal/content-copy.test.tsx`、`detail-host.test.tsx`。

- [x] Step 1：用例：①普通副本发布回答丢失、核验得已成为工作项、关闭：提示核对工作项，不提示草稿已保留；②链接创建与更新确认后规范重读撤下正文、表单卸载：成功提示恰好一次；宿主scope已结束（换工作项/路由）不提示；子项/关联保持无成功提示。先看到①错误提示、②无提示（RED）。
- [x] Step 2：实现§3第5条；①②分两个提交，②为M4本地修复，更正P7b记录与M4交接中“子项/关联提示被跳过”的说法。变异“提示随表单寿命”须被杀死。
- [x] Step 3：GREEN与受影响详情/peek测试；提交`fix(M5/P8): 未知发布已成工作项时不提示草稿已保留`、`fix(M5/P8): 链接表单在写后重读期间仍发出成功提示`。

## Task 7（T7）：删除无读者代码与素材，死成员闭环

**Files:** 删除`copyMarkdownToClipboard`（`types/editor.ts`、`helpers/editor-ref.ts`、`dist`随构建）与测试绑定；删除`web/apps/web/app/assets/attachment/`；删除`WorkspacePatch.Logo`；阶段记录死成员表。

- [x] Step 1：先证明无读者：`git grep`（含字符串/动态路径/样式）、knip、类型检查；按M1收尾计划附录A取出`deadsym.mjs`/`domains.mjs`/`lib/program.mjs`，`pnpm --filter web exec react-router typegen`后跑`--rows M5`，保存脚本hash与完整raw。
- [x] Step 2：删除上述三处；逐行处理`--rows M5`：删除只为死成员存在的代码，误报写明文件、名称与读者。`make knip`、web/editor类型、`go build ./...`与workspace包测试通过。
- [x] Step 3：受影响测试GREEN；提交`refactor(M5/P8): 删除无读者的复制方法、附件图标与工作区补丁字段`（死成员另一提交）。

## Task 8（T8）：两条规则全仓清零与cap下调

**Files:** 告警命中文件（§3第7条）；oxlint配置；`web/apps/web/package.json` cap。

- [x] Step 1：保存全仓oxlint按包按规则的原始统计；删除恒假/恒真分支前后`use-local-storage`/`local-storage`/`settings/helper`的现有测试或新增小测试证明行为不变（含`window`存在时的读写）。
- [x] Step 2：先删死分支使两条规则命中为零，再在根`.oxlintrc.json`的`rules`块设两条为error（`lint-cap.mjs`遇error即失败，顺序不能反）；web cap 67→58，经各包`check:lint`核cap等于实际；M5全期改动文件（`3db22f06..HEAD`的web与tools文件）逐包oxlint零warning。
- [x] Step 3：`make lint-web`通过；提交`chore(M5/P8): valid-typeof与no-constant-binary-expression全仓清零`。

## Task 9（T9）：S3夹具保留双错误与子进程原错误

**Files:** `e2e/fixtures/{s3.ts,s3-process.ts,s3-container.ts}`；用例追加到`e2e/stories/smoke/s3-lifecycle.spec.ts`（不启动容器、毫秒级，记录默认套件时长不变）。

- [ ] Step 1：注入①`startS3Process(name, "")`使子进程启动校验失败（`s3-container.ts`的catch现只回送固定字符串）②`s3.ts`层可注入的清理函数在启动失败后也失败：断言错误含子进程原错误，双失败为同时含启动原cause与清理错误的`AggregateError`。先看到只有通用信息或只有清理错误（RED）。
- [ ] Step 2：子进程把原错误随消息回送并作为cause（`s3-container.ts`、`s3-process.ts`）；`s3.ts`与`closeProcess`的双失败用`AggregateError`保留两者（沿`e2e/fixtures/test.ts`先例）；清理函数参数默认现有`removeS3Container`；不改等待期限。
- [ ] Step 3：GREEN；S3变体故事各浏览器至少一条回归；提交`test(M5/P8): S3夹具保留启动与清理两个错误`。

## Task 10（T10）：全链路清理与慢流演练

**Files:** `server/internal/bootstrap/`新演练测试（复用既有夹具与业务时钟）；阶段记录演练节。

- [ ] Step 1：设计一次演练：工作区A（将删除）与邻域B，A中覆盖六类内容用途与资料图片、未绑定准备件、失败复制；一次>1 MiB且>15秒的真实慢流上传成功。删除A：立即拒绝新授权；任务以原At/By软删；推进业务时钟过7天与60天；清理有限批次推进；注入一次对象删除失败后重试成功。local与S3各一次。
- [ ] Step 2：精确断言每步的行（asset/引用/任务）、对象键与字节SHA；B的行与字节不变；批次与时间预算不超配置。已有阶段测试证明的性质只重跑原测试并记录名字。
- [ ] Step 3：演练GREEN并保存原件；提交`test(M5/P8): 全链路清理与慢流演练`。

## Task 11（T11）：默认部署演示

**Files:** `e2e/playwright.m5-demo.config.ts`、`e2e/stories/files/f18-default-deployment.spec.ts`；`e2e/fixtures/server.ts`必须增加显式选项，使`startNerve`不注入test环境与放宽的限速；README部署节。

- [ ] Step 1：按README“部署”节的prod步骤：独立空库上`NERVE_ENV=prod nerve migrate up`；Node `crypto`生成Ed25519 PKCS#8经`NERVE_AUTH__JWT__PRIVATE_KEY_FILE`给出；`nerve users create`建第一个账户；以基础配置默认`files`（local、默认限速）与实例独占`NERVE_FILES__LOCAL__ROOT`启动。三浏览器故事：登录→头像→工作区图标→项目封面（预设与上传）→附件上传/下载SHA→正文图片粘贴→评论文件→重启进程→全部重读一致；`watchPage`无旧路由、CSP违规、未处理异常；阶段记录列全部环境变量。
- [ ] Step 2：扫描：无`/api/assets/v2`等旧路由、无`web/packages/services`、关键词守卫无到期例外、T7死成员表闭环。
- [ ] Step 3：三浏览器GREEN并保存原件；提交`test(M5/P8): 默认部署演示`。

## Task 12（T12）：最终F1–F18矩阵

**Files:** 阶段记录矩阵节。

- [ ] Step 1：在最终产品候选（T2–T11之后）上：`make build build-s3-fixture`；默认`make e2e`（含`[s3]`页面变体，Chromium）；`m5-profile`、`m5-content`（local，三浏览器）、`m5-s3`（F4，三浏览器）、`m5-demo`（三浏览器）；`GOFLAGS=-p=1 make test`。
- [ ] Step 2：每条F指向本次运行的故事/测试名、浏览器、存储变体与原件；`t13-content-local-4`等间歇项隔离重复运行至少5次并给结论。
- [ ] Step 3：提交`docs(M5/P8): 记录F1–F18最终矩阵`。

## Task 13（T13）：文档同步与交接

**Files:** README、`server/configs/`样例注释、`docs/v0/{plane-diff,frontend-changes}.md`、`app/assets/*/SOURCES.md`、七份输入交接、`docs/v0/M6-cycles-modules/handoffs/M5-closeout.md`、`docs/v0/M7-collaboration/handoffs/M5-closeout.md`、`docs/v0/M8-open-release/handoffs/M5-closeout.md`、M5设计§13/§15。

- [ ] Step 1：七份输入交接逐条写“处理结果（M5/P8）”：实际完成+证据，或具名转交目标与验收（保持`status: open`，closeout改closed）。M4-closeout的M5-04原文整段转交（若仍无投递任务的CLI），复制方法写“已删除”，P7b T14段“子项/关联提示”更正为只涉链接；M2-closeout §2的`use-file-size.ts`已删，写实际读取点（`core/hooks/editor/editor-assets.ts`读`root.instance.config.file_size_limit`）。
- [ ] Step 2：三份`M5-closeout`：资产端口与用途、宿主权限与生命周期/作用域、真实支持的存储协议与部署条件、事件名与载荷、清理/通知责任、未接入领域如何接入、仍开放的风险；M8另含部署/备份/CSP/配置。修复re-32→P1决策§3与re-41行号锚两处现存断链；`doc-links.py`零断。
- [ ] Step 3：提交`docs(M5/P8): 同步部署与差异文档并写M5交接`。

## Task 14（T14）：七门禁与准确检查点CI

- [ ] Step 1：串行`make gen-check`、`make lint-go`、`GOFLAGS=-p=1 make test`、`make lint-web`、`make knip`、`make test-web`、`make e2e`；真实exit、完整raw、cache/skip、前后输入hash、容器快照。
- [ ] Step 2：失败按根因修；环境失败同输入重跑并保留原件，不改预算。
- [ ] Step 3：中文检查点提交、system push，精确HEAD的server/web/e2e全绿。

## Task 15（T15）：一次整分支deep-review与修复

- [ ] Step 1：冻结基线..候选，一次fresh reviewer（最强档）读全部手写变更、Review Focus与所有Ruling；报告存`docs/reviews/re-52-M5P8整体审核.md`。
- [ ] Step 2：主控按效果复核；Critical/Important一次修复、各有RED/GREEN；Minor入台账；DeclinedToJudge逐条裁定。受影响的矩阵行在最终候选上重跑。
- [ ] Step 3：最终本地七门禁与修改范围对应，push准确HEAD CI全绿；阶段记录写事实、限制与closeout责任。

## Task 16（T16）：集成、归档与closeout预检

- [ ] Step 1：核main与worktree干净、准确green候选及祖先，根main用`--no-ff`，核双父/树一致，system push。
- [ ] Step 2：准确main CI三项全绿后归档全部ignored原件，核hash/bytes/mode/membership/unknown0；origin/main..branch=0后正常清理远端/本地/worktree，归档再核。
- [ ] Step 3：closeout worktree回填P8完成证据；写closeout spec/plan并一次builder预检：首次noff/main绿→§14逐条勾选与总体§9.4→第二次noff/main绿→最终归档清理；输入交接改closed。

## 预检自查

- [x] spec覆盖逐Task，A1–A8都有实现和证据落点。
- [x] 文件/名字与实际源码一致；删除项确无读者；演练与演示只用既有夹具与默认配置。
- [x] 五Review Focus均有明确反例，RED不是缺模块或编译失败，未知结果不自动重发。
- [x] 无新增依赖/协议/权限观察器/产品测试开关；默认CI套件不因矩阵增长。
- [x] 正式基于P7b最终main复核，builder问题按既有事实/局部裁定/需负责人决策分类闭环。
