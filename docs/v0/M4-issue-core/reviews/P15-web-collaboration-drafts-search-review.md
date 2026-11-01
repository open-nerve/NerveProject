# M4/P15 协作、草稿与命令搜索：预检、实施与交付记录

日期：2026-11-01。状态：T1–T16 实施与终审修复完成，完整本地门禁通过；最终候选 CI 按 §10 后置核准后交付，集成待办。§1–8 保留准备期快照，当前事实见 §9–10。基线 main 7a81ea18d8e9d80f079e4e30ae0e90b11be23ef9，分支 codex/m4-p15；[spec](../specs/P15-web-collaboration-drafts-search.md)、[plan](../plans/P15-web-collaboration-drafts-search.md)、[deep-review 预检](../../../reviews/re-23-M4P15预检审核.md)。

## 1. 前置与方法

P14 授权合并、main 准确 CI、I19/I3 集成测试修复及两工作树归档清理见 [P14 review §15–16](P14-web-issue-detail-review.md#16-授权集成main-ci-与归档清理2026-10-31)。本期只新建独立 worktree，主目录保持 main。原生 worktree 工具不能指定负责人要求的精确路径，沿 using-git-worktrees 使用系统 git worktree add。

按 02 §3 用源码、生成合同、消费者和测试静态核对替代产品原型及逐 Task 重放。没有安装 P15 依赖、改产品、执行 P15 测试或提前进入 phase。brainstorming 的独立 builder 全文读 spec/plan，并查评论授权、草稿发布、搜索和 clock，未发现关键未决问题；主控按 deep-review 三视角顺序核验，不是三位审者。最后路径及基线文字修订由主控负责。

## 2. 验收映射

下表均是实施后的验收落点，本次不勾选产品完成。

| 标准 | Task / 实际依据 | 状态 |
|---|---|---|
| A1 | T1，activity store/order、IssueUser、I12 | 待实施/回归 |
| A2 | T2，activity-filter/filter-root/sort-root、C6 | 待实施 |
| A3 | T3，CommentCard/QuickActions、后端评论授权、I11/I20 | 待实施 |
| A4 | T4，issue/comment reactions、picker、C6 CSP | 待实施 |
| A5 | T5，原生版本资源/restore、ModalTitle、getMarkDown、I13 | 待实施 |
| A6 | T6，draftInput、WorkspaceDraftIssues、个人草稿页、I14 | 待实施 |
| A7 | T7，P14 dirty/readiness 与 ModalCore、C9 草稿 | 待实施 |
| A8 | T8，工作项命令聚合与原生确认写、C7 | 待实施 |
| A9 | T9，theme-switcher sessionGuard、四文件 route 守卫 | 待实施 |
| A10 | T10，useIssueSearch/IssueSearchService、显式项目目录、I18 | 待实施 |
| A11 | T11，13 历史实例、WorkspaceService view 保留 | 待实施 |
| A12 | T12，ClockWorld/auto-archive fresh 水位、I16 page | 待实施 |
| A13 | T13，四复制消费者与 T5 Markdown、C8 LAN HTTP | 待实施 |
| A14 | T14，显式三个 browser、C6–C9、晚答与 Back | 待实施 |
| A15 | T15，本期 dead/lint/关键词、真实受影响文件 | 待实施 |
| A16 | T16，完整 F、整分支 deep-review、准确分支 CI | 待实施 |

## 3. 准备决定

D1–D8 详见 spec §3。命令原粗项拆为 T8 工作项动作与 T9 主题/创建路由，共 16 个 Task；其它顺延，完成线未削减。每 Task 约 300–1450 行含测试，超出先局部拆职责，不能把未接通领域挪进本期。

沿现有 root 权限所有权、确认队列与 owner；已满足的来源/actor/草稿能力只回归，版本 Markdown 用既有同步 getMarkDown，复制 hook 单独负责反馈。命令根页空 q 保留原命令，非空 q 原生搜索，项目在显式目录本地过滤。M7 服务与 P16/收尾边界不扩大。局部决定均有错判代价和测试落点；架构新发现仍先提交再上报。

## 4. 预检发现与修订

[re-23](../../../reviews/re-23-M4P15预检审核.md) 逐项记录 R1–R9：复制返回/失败、payload 整替换与发布顺序、clock 新进程凭据、M7 搜索/视图边界、查询 owner 与目录、LAN HTTP、精确路径/守卫、已有能力的真实回归、T8/T9 与 T10 搜索迁移的先后依赖。已全部修订文稿。独立 builder 未提出新的阻塞决定；主控最后修正 power-k/core/types.ts 的歧义，并全文核对 commands-list/search-menu 后把根页非空查询的页面验收明确留在 T10/T14。这两项最终修订由主控复核。

没有确认需要改变服务端协议、数据模型、权限所有权或跨模块责任的问题。这里的“通过”只代表计划可执行，不代表未来实现无缺陷。

## 5. 与设计和方法的差异

原设计 15 粗任务细化成 16 个，不改变范围；设计 §12、§15 同步。plan 提供具体 Files/Interfaces、步骤、命令、期望行为与变异，不提前伪造尚未生成代码的 hash、测试次数或整段原型。授权实现后逐 Task 保存实际输入、失败层、恢复和回归。

本期最后模块实例为 search-menu 的 WorkspaceService；此前 13 路径的实际清理归属为 P11 九处、P12 一处、P14 两处、P15 一处，不照旧分配重复施工。公开 editor ref、M7 四 view 方法、旧领域适配继续保持明确合同。

## 6. 移交事项

M1-P3、M1-P5、M2/M3 closeout 只准备本期落点，顶部 status 继续 open；本次不追加虚假的实施关闭证据。T15/T16 实施后追加各子项事实；M4 真正收尾才统一 closed、逐条勾设计 §14、改总体设计 9.4 和建立 M5–M8 的 M4-closeout。

P16 全局 dead/no-shadow、C12 和总守卫；M5 文件/图片；M6/M7 自身页面、收集箱来源/系统创建者、版本旧 DTO、M7 lazy pending discard；其它领域搜索均保留原归属。

## 7. 已知限制与待验证

评论和菜单键盘、管理员删别人、表情 CSP、完整草稿 UI、搜索乱序和 cmdk、LAN HTTP 剪贴板、三浏览器都仍待产品实现和测试。远端旧编辑/删评论超出增量窗口须关闭重开才读取；原 API 空 q 最近项不等于命令首页要列近期项。没有扩大产品承诺或将已有测试当作这些行为的替代证据。

## 8. 静态预检与准备交付

主目录 .superpowers/sdd/P15-preparation-20261031/ 留存 builder-check.md、source-review-inventory.json、path-inventory.json、document-links.json、precheck-static.json 及 raw。只核文档、路径、合同、现有基线和关闭证据；16 个 A、16 Task 共 80 步全部未勾，tools/keywords.json 仍 M4/P14。

静态预检 exit 0：本分支只改 16 份 Markdown 文档；88 个确定路径中 73 个已存在、15 个明确待新增；386 个本地链接没有新增错误。设计中原有的 M5–M8 四个未来 M4-closeout 链接保持待收尾，不伪造文件消除检查。主目录 main 与 origin/main 均为上述基线且干净，P14 准确 main CI 和清理回执及三个归档 manifest 的 hash 已核对。最终文稿复核输出另存 precheck-static-final.log。

准确文档提交和远端 HEAD 以 precheck-delivery.json 后置记录，主目录仍是通过 CI 的 main，P15 分支/工作树保留给后续 T1。最终阶段整分支审查与 CI 在实施后执行，本次不宣称已完成。

## 9. 逐任务实施记录

实施根：P15 worktree 内 `.superpowers/sdd/P15-web-collaboration-drafts-search/`。run.py 每次保留完整 argv、HEAD/dirty 文件 hash、输入摘要、raw/exit、构建程序 hash；Task 收据汇总具体命令，不以最后一次绿覆盖此前失败。用户授权连续实施、阶段末 deep-review，先不合并；完整七门禁及推送检查点为 T5/T10/T14/T16。

### T1：时间线与系统操作者

首个实施提交把 phase 改为 M4/P15；守卫 76 条、17 个例外无命中，没有延期例外。原时间线按 kind+id 和纳秒排序、末页请求 cursor 增量读、系统/null/缺失目录分支已正确，保持数据层和 actor 代码不变。新增三页与描述活动移位、同 id 评论/活动、系统同名真人及创建/归档/恢复的消费者验证。历史 13 处实例按现状只余 search-menu 的 WorkspaceService，按 T10/T11 处理，其它领域实例不计入本期清零。

I12 新页面经真实 API 准备 103 条活动和 102 条评论，默认每页 50，不拦截或伪造页面响应；首开读三页，真实编辑器确认描述保存后同一活动 id 移到评论之后，无重复/丢失，两集合都从旧末页请求 cursor 重读。远端旧评论编辑/删除在关闭再打开时完整体现。另一个页面验证隐藏历史关联不泄露，邻近可见活动仍显示；原两个 API 故事保留。

页面首轮及修改定位后的复验均为 3 通过/1 失败。首次以为是 :scope 选择器包含自身，第二次同样失败推翻了这一归因；trace 证明评论外行和内部编辑器使用同一 raw id，选择器命中两个父容器。补真实 CommentCard 消费者 DOM 身份测试，确实失败（4 通过/1 失败）；局部修复为 CommentBlock 统一持有既有公开 comment-{id} 锚点，display 继续依此滚动而不另占该 id，编辑器保留原生 id。公开评论链接/高亮合同不变。此处提前修复 T3/T13 依赖的同域展示问题，无权限/API/模型改动；代价是锚点定位完整行，由实际页面确认。未把更换选择器当作产品修复。

最终输入摘要 `f3027af0b545421036a832896804d0dbcd6c438dbc76be110ea87949eda3ec5f`：四文件 unit **37/37**；I12 Chromium **4/4**（2 API、2 page）；Web/E2E types、六个改动 TS/TSX 文件 oxlint **0 warning/0 error**、format/diff/关键词通过。m01-cursor（丢末页请求位置）与 m01-system（缺失目录误判系统）分别在真实断言层失败，源码逐字节恢复后 unit 再次绿。此前 no-await-in-loop 的两个新增测试告警改为有序 reduce/独立 Promise.all，不加豁免。精确日志/恢复 hash 见 t1-final-*.json、t1-restored-unit.json、t1-mutations.json 与 t1-receipt.json。

A1 与 T1 五步完成；三引擎矩阵、完整阶段 CI 和整分支审查仍归后续 Task，当前不声称已通过。


### T2：动态过滤与键盘

筛选触发器由嵌套 IconButton 改为图标与屏幕阅读器文本，PopoverMenu 仍唯一拥有原生按钮；选项改为原生切换按钮，以 aria-pressed 呈现选择、aria-disabled 表明最后一项不可取消。沿既有 root 的至少保留一项逻辑与 localStorage，不新增数据/状态镜像或共享库接口。

真实 root、useLocalStorage 和 Popover 的 controls 测试先在嵌套 button 断言失败（18 绿/1 红），修复后 19/19。C6 真实页面经 Enter/Space、Tab 进首选项、最后一项拒绝取消、Escape 回焦，验证筛选实际时间线、排序及刷新保留；最终 1/1。首次整页无嵌套断言发现其它动作/表情控件仍嵌套，T2 按 owning 范围验证筛选触发器内外均无嵌套，其余由 T3/T4 修复并在 T14 整体验证，没有宣称整页已清零。

M02 删除 root 最后一项保护后，按 Enter 后的 aria-pressed 真实断言由 true 变 false，测试红；逐字节恢复。Web/E2E 类型检查和四文件 oxlint 零告警（含 helper），format/diff 通过。第一次类型命令误写 check-types、提示无此脚本，已按计划 check:types 重跑通过，原输出保留。原始证据 t2-*.json、m02.patch/json；A2 与 T2 五步完成，完整七门禁仍在 T5。


### T3：评论权限与可靠输入

CommentCard 将明确的 workspaceSlug/projectId 和目录 displayName 交给菜单，管理员删除沿现有 allowPermissions；作者仍独占编辑。工作区管理员兼项目访客对可见项沿现有豁免，OwnOnly 不变。菜单只保留一个原生具名按钮，中英作者命名同步；删除等待中禁用入口。创建/编辑统一在实际发送处检查非空、ready、scope 和锁，捕获阶段挡住 IME Enter/229，输入保持既有 HTML 合同。请求自身持有 busy，防重复 handleSubmit 提前释放；局部 inert 期间不能继续改尚未确认的输入。普通拒绝按 code，字段拒绝按 field；确认后才清空/退出，旧宿主创建/编辑的晚答不触发新宿主关闭或成功反馈。

初始两组测试中先修正测试账号 id 与缺少 ResizeObserver 的环境问题，再观察 **6 红/9 绿** 的产品失败（管理员动作、创建/编辑 IME、字段拒绝）。后续连续提交使 busy 提前解除的两个红例及删除未锁入口两个红例均已局部修复。字段错误的首版 group 方案被 a11y lint 拒绝，没有豁免；改为真实编辑器属性后发现原 CoreEditorProps 浅覆盖会丢 typography，带 [editable] 的 TipTap hook 又不会在普通更新同步属性。两处均按原接口局部修复，保留默认 class/textbox 和输入实例，在 layout effect 更新字段属性后由 useRefusedFocus 回焦；范围与额外回归已同步 spec/plan/design，不改模块依赖、公共 DTO/权限 owner。对象属性组合/更新有真实 TipTap 红绿；原函数属性路径保持原样，不新增对当前 TipTap 初始化函数的支持承诺。

页面先后为 **4 绿/1 红**：先发现切 editable 重建导致回焦目标消失，再在保持实例后发现属性旧值。后一轮补充根因，没有把首轮修正冒充完整解决。最终 I11/I20 **6/6**（2 API、4 page）：真实编辑器的空/IME、422 字段提示与回焦、双击创建只一次且只有 comment_html、作者编辑确认、管理员菜单不含编辑但可删他人、普通访客及工作区管理员兼项目访客的评论/OwnOnly 都通过。受控 422 仅用于输入拒绝，成功写与其它故事均用真实服务端；各页面保持旧接口/非预期错误/CSP/站外请求守卫。

输入摘要 `49fa22798ae9107c349fcd1ce511c9bf07e00dd7503cad08748b07366a46a200`：前端六文件 **49/49**，其中本 Task 两文件 **16/16**；编辑器交互/清洗 **19/19**（产品文件与最终输入逐文件一致）；I11/I20 成功后只追加旧编辑宿主单元测试，页面输入的产品/故事 hash 保持不变。受影响 I2 描述/I12/C6 **5/5**（1 API、4 page；grep 的 page 也匹配了 I12 API 标题中的 pages，照实际记数）。Web/E2E types、14 个 TS/TSX 文件 oxlint **0 warning/0 error**。M03 去管理员删除、M04 去 IME 和去旧宿主确认保护，分别由菜单、写请求、旧回调断言捕获；三组源码 hash 全部恢复。未放宽关键词/CSP/架构守卫。

原始证据在 t3-*.json/log、m03-admin.patch/log、m04-ime.patch/log、m04-scope.patch/log、t3-mutations.json、t3-receipt.json。依赖构建时误并行 unit 造成零用例导入失败、初始共享属性函数尝试失败、默认空对象类型构建失败均保留，已修复并重新验证，不作为有效产品 RED/GREEN。T4 表情触发器的既有嵌套警告仍归 T4 收口；当前 A3 不宣称所有评论表情控件已验收。

A3 与 T3 五步完成；继续 T4。


### T4：两类表情回应与归档只读

保留原生两个 reaction owner：工作项独立集合、评论 embedded reactions，没有新 GET 或 store。评论头部和底部共用一次发送锁；计数按钮与 picker 拆为兄弟，现有 Popover 只拥有一个触发 button，等待/归档经 fieldset 禁用实际计数按钮，事件处理也检查 disabled。归档仍展示原回应；具名评论头尾按钮同时带作者，中英文一致。实际选择器用既有本站 emojibase 与码点转换，CSP 未放宽。

新增消费者测试修正最初误选外层按钮和 FakeNerve body 读取后，确认 **6 红/15 绿**：嵌套按钮、评论双击多发 POST、归档隐藏及工作项等待/只读未禁用。根因修复后，六文件 **55/55**（含评论与回应数据层、权限、输入回归）；m05-codepoints 误按 UTF-16 拆分，原始请求值断言 3 红；m05-owner 去本人条件，实际他人记录丢失断言 1 红。两次源码字节恢复一致，恢复后再次 **55/55**。

真实 Chromium I11/C6 **6/6**（1 API、5 page）：管理员/作者评论回归，两类回应均在持有 DELETE 时保留计数且只发一次，确认后仅剩另一人的记录；终态工作项归档后两个回应仍可读而零写。选择器 Enter/Space 打开、实际 searchbox 和 gridcell 选择组合表情、Escape 回焦；两个 POST 请求体均为 `128105-8205-128187`；资源含本站 `/assets/emojibase/15.3.2/en/data.json`，无第三方请求/CSP/控制台错误或新增豁免。评论单元另验证带肤色组合 `128105-127997-8205-128187`、403/409 不变计数和重试。

首轮页面使用不存在的 common.search 键，已改既有 search；后续测试修正为等待确认后入口可用再按 Space，以及创建 Done 状态满足原归档前置，不改产品业务。全页无嵌套断言还发现 3 处既有顶栏/项目切换嵌套，T4 限定评论/回应实际拥有的控件，三处仍留最终范围核对，不宣称全应用清零。两条新测试循环 lint 告警已改为独立 Promise.all。

最终产品/测试输入 `e0290810a5795fda82b0229d2f5df13deb00544a5021b3c6aa511661209a8348`；程序 `bc29a3901f4d6be0b6663e8db851bf230981dd5483711a54a43a3f1d326b3e4c`。Web/E2E 类型与 7 文件 oxlint 零告警通过；原始运行、失败、变异及恢复见 t4-*.json、m05-* 和 t4-receipt.json。A4 与 T4 五步完成，T5 接续完整七门禁检查点；尚未做阶段整审或合并。


### T5：描述历史、确认还原与 Markdown 复制

版本读资源和 restoreDescription 确认链保持；共享 modal 补真实 ModalTitle。复制改从 ready 的编辑器取 getMarkDown，交现有 useCopyText 等待并由 hook 唯一报告结果；复制等待中不能再次发送。公共 EditorRefApi 的 void 复制方法、M7 IntakeDescriptionVersions 的原 props/旧 DTO 适配与恢复包装不变，没有把 M7 页面提前验收。

真实 modal 消费者先观察 4 红/16 绿（缺少标题、提前/重复成功反馈与未 ready 复制）；局部修复后 20/20。扩版本资源和懒编辑器回归共 43/43。M06 提前关闭、旧版本替换当前选择及 M07 复制前假成功三组变异分别触发真实断言失败，源码逐字节恢复，恢复后 43/43。

I13 Chromium 4/4（原 API 1、page 3）：真实分页/选中历史、TipTap 输出精确 `# Original\n\nKeep **bold**.\n`、持有 clipboard 只写一次且确认前零反馈；403 还原保留对话和内容，随后双击只发一次普通 description_html PATCH，Escape 不关等待对话，确认后关闭且回焦。600 秒内同作者合并后版本 id 集合未增加，不承诺一次还原必新建版本。归档页列表/详情各自 500 后可重试、只读无还原；访客可读自己的可见详情，隐藏历史入口且零历史请求。仅精确预期拒绝被记录，其余网络/CSP/控制台守卫保持。

首轮页面 1 绿/3 红：51 条历史前置与 P7 保留最近 20 的 worker 竞态，最旧详情真实 404；改为 3 条真实历史、浏览器 list 请求转发合法 limit=2，所有回答仍来自真实后端，不停 worker 或造响应。该页面验证分页消费，默认上限和保留策略沿既有 API/服务端测试。另两例因为只读标题没有 textbox，扩既有 openIssueDetail 可选只读标题断言，默认可编辑断言不变。第二轮 3 绿/1 红来自 portal 根零尺寸，改核具名 dialog 唯一性、实际内容可见与焦点在内；没有改产品或加延迟。

最终聚焦输入摘要 `f957ac0ead30df123a8c33e052c58366b72a8b18338db925d17774a7cf7a49ad`，程序 `44134c4ca2909d671e482bca8cccdd40254485ba49ddcb66601ae9007048f102`。检查点七项完整门禁最终均 exit 0：Go 51 个包通过、16 个无测试包（非 verbose，逐测试 skip 未知）；Web 静态 54/54（36 缓存），knip 通过；无缓存 Vitest 286 文件/3020 项和 Node 5 项通过，默认 E2E 248/248（246 Chromium、2 Firefox，0 skip）。首轮 F4 唯一失败是警告实测 112 低于旧 cap 114，同提交下调到 112 后从 F4 继续；F1–F3 的源码输入未变，逐文件核对见 t5-gate-input-equivalence.json，没有重跑服务端或冒称同一摘要。后四门禁输入 `f7a21a156d9d0538bf51f1222a9f924b86e40f483206fc88700ae7166d2509d6`，完整 E2E 程序 `a39471da10293ddc47720e59c6a2608b50715005fca0a7070e84e7268479aada`。默认结果另存 t5-full-e2e-results 并有 manifest，防后续覆盖。所有原始输出/变异/恢复在 t5-*.json/log、m06-*/m07-*、t5-mutations.json，检查点 commit 与 CI 在 t5-receipt.json 后置记录。当前阶段整审尚未执行、不合并。

A5 与 T5 五步完成，继续 T6。准确提交及分支 CI 后置记录，不以本地验证代替远端结果。


### T6：个人草稿页面与确认发布

沿 P8/P11 的原生 Draft、整份 draftInput、队列和确认创建 owner，没有新接口或列表镜像。header/empty-state 显式按路由 workspace 判权限，root 保留 route prop；列表失败保留已加载页，retry 重取响应 total。无项目编辑器为数组提供展示空值，未编辑 payload 由 held 原值保留。右键 ContextMenu 从两套响应式 QuickActions 提到行的唯一拥有者，条目在行统一翻译；按钮复用既有具名文案。复制草稿使用私有 hasDraftSeed，使直接/关闭保存只将实际编辑合入原生种子，不能展示的未知 priority 不丢失；普通创建及现有编辑语义保持。

初次消费者 3 红来自 route 权限串用；真实页面又暴露两个右键 menu 和缺失数组导致的 value.length 异常，均根因修复。后续页面失败分别为右键非聚焦文本的焦点假设、ContextMenu 原始翻译键，改实际按钮右键并修 owner 翻译。一次 Ryuk 端口绑定超时发生在 globalSetup、零用例，原输入重跑绿，未操作既有 Docker/容器。复制两条保存测试各自因未知 priority 丢失而红，修复后绿；初版关闭测试的 mock 强制关 wrapper 已纠正，不把该 mock 失败当产品红。

最终九文件 **77/77**；实际表单覆盖空标题/九字段、项目 null、403/409/422 拒绝、PATCH 失败无 POST、先保存再无 body 发布、确认后的 GET-only retry、权限退役晚答。M08 将完整 payload 改为局部替换、M09 吞 PATCH 拒绝仍发布，均触发实际消费者断言红，源码字节恢复后七文件 55/55；随后复制修复扩大九文件 77/77。Web/E2E types 与 11 文件格式、oxlint 零告警、关键词 76 规则/17 例外无命中通过。

I14 Chromium 共 **7 个故事验证通过（3 API、4 page，分两次）**：首轮 6 绿/1 红只因复制页按钮大小写写成 Save to drafts；按实际 Save to Drafts 修正后复制 1/1，另六故事和产品输入不变。51 条真实草稿按 50+1 分页、服务端 total；无项目的九字段编辑/重开/删除、键盘/指针菜单、held 双击一次写、发布拒绝保留与确认后读重试、访客零草稿读取；复制经真实 TipTap 发整份保留输入且 DB 原记录不变。只有指定 409/500 拒绝被允许，网络/CSP/页面错误守卫保持。copy 测试的 sort 警告改 toSorted，没有加豁免。

77 单元及六故事输入 `6c6225843c85a8e8cb7a18f8fbf2db03507e0e265c7b79a707b5bd54ec9f7232`；最后复制页/静态输入 `8123471c45452c1e0ffcf96395b398b0e5418e6dbe151ff6a3591e852bc96aeb`，只改该新增故事的按钮大小写与不变排序语义；程序均 `49c97b3a58dce1d572a087e77a09b60cf488d11f531a724bc09ed80ebd58c9ec`。原始失败、变异、恢复和成功在 t6-*.json/log、t6-*-results、t6-mutations.json、t6-receipt.json。误调用不存在 make check-keywords 的命令失败如实保留，正确 node tools/keywords.mjs 已过。A6 与 T6 完成，接续 T7；跨浏览器完整矩阵仍归 T14。

T5 检查点 f6eb9a5784974d416bda08cbaa0aa3a6aecc4126 已推送；[CI 41215094874](https://github.com/open-nerve/NerveProject/actions/runs/41215094874) attempt 1 在 2026-10-31 15:52 UTC 的 server/web 为 success，E2E 仍在运行，未提前记全绿。T6 不到完整七门禁推送点，下一检查点 T10。


### T7：创建弹窗存草稿与单关闭决策

普通新建表单增加直接 Save to Drafts，沿同一个 base 提交锁和原生草稿 owner；空标题不走发布校验，描述未 ready 或加载失败不能保存，字段拒绝在提示提交后回焦。两种入口都按现有草稿 owner 的 workspace 权限；访客无直接入口/关闭存草稿提示。关闭确认通过私有 closeAfterSave 告知原提交 owner，成功才完成关闭，解决 createMore=true 时确认窗留下的问题；直接保存保持继续创建原行为，metadata 不进入 Issue/Draft 请求。

既有关闭/迁移三个文件先 27/27。新增测试看到缺少入口和“继续创建”确认后未关闭；修复后五文件 52/52。首次真实 C9 两红一绿，trace 证实未选状态发送 payload.state_id=""，服务端按 UUID 合同 400 拒绝；这不是发布语义或服务端 bug。在原 draftInput 规范化空引用/日期为清键、空 project 为 null，合法空文本/数组和未编辑字段保持。真实表单与适配器各有一个有效红，再经页面绿；未修改后端校验、权限或所有权。

最终七文件 **63/63**；包含空标题、直接/关闭保存、拒绝保留和字段先读后回焦、忙碌重复提交、缺项目草稿原回归、描述读取失败、纯迁移与混合用户编辑、guest 权限。真实 Chromium 四文件 **33/33**，其中 C9 四页覆盖真实 TipTap 直接保存/422 拒绝/空标题、继续创建选中时关闭保存、嵌套 Escape 回焦/inert、Back 退役旧命令而不关闭新工作区表单、重新加载 guest 权限后编辑器不复活、复制草稿纯 block-id 迁移不提示保存/不写 DB；其余 I14/C4/C10/C12 复验通过。权限页不声称服务端主动推送：无刷新时同 DOM 权限变化由 owner 单元验证，完整会话/ABA 矩阵仍归 T14。

M10-ready 去编辑器就绪检查使零写断言收到 1 POST；M10-escape 将内层关闭误接外层使保留 1 dialog 断言收到 0；两个变异字节恢复后 63/63。Web/E2E 类型、11 文件 oxlint 零 warning/格式、关键词 76/17 无命中。最终输入 `7b2f0517e35f382535230aa3195b51b1a26080dcda4e23bf498995c15885fc30`，程序 `6dd421fb184433a2af0f298d728c62b0f5efb79fa73a29cd0548cb6c95b7ddbb`；原始命令/失败/trace/变异在 t7-*.json/log、t7-*-results、m10-*、t7-mutations.json、t7-receipt.json。初始 portal helper、真实/假时钟 fixture 混用均已纠正并保留，不算产品红。

A7 与 T7 完成，接续 T8。T5 提交 f6eb9a57 的 [CI 41215094874](https://github.com/open-nerve/NerveProject/actions/runs/41215094874) attempt 1 已在 2026-10-31 16:00 UTC 核验 server/web/e2e 全部 success；只覆盖该检查点，不冒称尚未推送的 T6/T7 已有远端绿。下一完整七门禁与 push 为 T10。


### T8：原生命令拆分、确认队列与删除宿主

commands 入口只组合私有 context/copy/mutation hooks。显示读取与动作生命周期分离，首屏命令无需额外通知；状态/优先级使用生成 IssueUpdate，指派/标签在原 root 写队列轮到执行时再基于已确认数组切换。成员编辑与作者/管理员删除分别判定，归档只读仍允许已授权删除；拒绝按 code，旧路由/会话不提示。三复制走原 hook 的单一异步反馈，删除本文件 copy-hook 例外，没有新增队列、协议或权限 owner。

基线 16 红/10 绿，修复后暴露首屏生命周期 active 尚未提交导致命令隐藏；独立 DOM 断言 1 红，已修。初版两个 OwnOnly 夹具把 API 禁止返回的他人实体强行 acceptRead；改按 API 返回集合构造，业务 OwnOnly 仍由 I20 API/页面负例负责，不把 UI 自行过滤当成新权限事实。M11 去旧回调检查后真实多发 PATCH，1 红/26 skip，源码字节恢复。

C7 真实页面又发现：编号详情删除 204 后 canonical 消失，宿主从实体派生生命周期导致跳过返回列表。真实 root/FakeNerve 宿主回归 1 红/7 绿；改按稳定路由和原根权限水位捕获，确认删除不自我退役，换路由/权限 ABA 仍拦截。最终三文件 **37/37**；Chromium **C7 1/1**，空查询键盘进入状态子页、中文筛选/IME 不提交、确认前不投影、删除 Cancel/Escape 回焦、等待中不能关闭、204 后返回列表，零旧搜索请求。初始浏览器夹具的 db.query 数组读法及 Delete Work item 大小写两处已纠正，原失败保留。

Web/E2E 类型、9 文件 oxlint 零告警与格式、76 规则/16 例外通过；未放宽守卫。最终聚焦输入 `f04577d64635d78da80f3e16804e1b2d15c69896cb5a46af547005b4fd4083b2`，页面前只相差消费者测试的 memo 夹具，生产/页面字节未变；页面程序 `57096c7d2093bd0f53fdeb11bd9a5f794e35adc26cd2b3e021423e7e824d051d`。原始日志、hash、trace、变异恢复见 t8-*.json/log、t8-c7-*-results、t8-mutations.json、t8-receipt.json。A8/T8 完成，继续 T9；尚未阶段整审，不合并，下一 push 为 T10。


### T9：主题确认与显式工作区创建范围

主题经原 ProfileStore 排队确认、sessionGuard 检查后才应用返回的 theme 并重载；当前拒绝用原 code 文案，旧登录成功/失败均不反馈。创建命令通过 ctx 路由工作区、knownProjectIds 与既有项目权限筛选未归档可创建项目，交原弹窗 allowedProjectIds；不依赖 router getter，不改其它领域创建逻辑。四个指定文件已加入 workspace-address-route，.ts/.tsx 正例和领域/测试负例通过，content 逐字未改。

13 条新消费者在基线全红，主题提前应用、旧代仍反馈和旧工作区目录/允许项目缺失均为行为失败。修复后连同 T6 路由页回归 **17/17**。M12 提前 setTheme 造成 7/7 红，字节恢复后 17/17。初次类型检查两处夹具 activeContext="global" 不属于合同，改 null；不是产品红。Web/E2E 类型、5 TS 文件 oxlint 零告警、格式及关键词 76/16 通过。

Chromium C7 **3/3**，包括 T8 中文状态/删除回归、主题 503 原因文案且不变色不重载、真实 PATCH held 时保持主题、200 后唯一重载、另一标签页真实登录后的旧 200 不重载/变色/提示。精确允许一次 503，其余请求/CSP/控制台保持零异常。最终消费者/静态输入 `0fa087e809aeca1f43cdc4b0422de8170a2b3243b9de2a1add83ce4b22337f70`，页面输入仅相差完整 Profile 夹具/测试类型修正/守卫说明标点，生产与页面字节相同；程序 `f850f09586eaf3d197d2b50e95ab2510a06444d917427c231db3defdeb7ddc4b`。运行与变异见 t9-*.json/log、t9-c7-results、t9-mutations.json、t9-receipt.json。A9/T9 完成，接续 T10 完整七门禁检查点；不合并。

### T10：原生命令搜索、本地目录与快速重开

搜索消费者只用原 useIssueSearch 单次 500ms 延迟；query/作用域变化立即换 owner 隐藏旧结果，根非空查询只显示原生 Issue 与本地可见未归档 Project。目录复用会话 PROJECTS/id/slug 读取，错误与 loading 独立，不随 q 重取。删除六组旧映射和已无读者的额外 debounce/搜索 key；cmdk 不再过滤或重排服务端命中。空根命令及属性子页保留。

12 条新消费者基线行为红，已有搜索 hook 12 条绿；实现后 24/24。真实 I18 首轮发现关闭宿主遗留 200ms reset 能清空首次打开的新查询；独立生命周期测试 2 红，改关闭时同步清理本宿主状态后共 **26/26**。M13 去 query owner 依赖、M14 工作区仍传 project_id 分别实际 3 红/1 红，源码逐字节恢复。I18 次轮测试误把候选包含匹配当模糊匹配，按设计 §3.18 校正为合法前导零编号 0001，仍能证明结果不可客户端二次过滤；无结果断言包含真实 Clear search 名称。最终 **I18 8/8**（4 API、4 page），含跨项目排序、中文/字面/编号、目录不重复、真实错误/GET重试、晚答与两类 OwnOnly 访客。

完整 F 首轮服务端仅 identity/adapter/river 的临时容器端口查询超时（业务断言未开始）；未改 Docker/代码，完整 make test 重跑通过。随后前端 54/54、knip、强制无缓存 18/18任务通过；保留分包日志：**292 Vitest 文件、3092 用例，另 Node 5 用例**，非逐例 verbose 的 Go 结果不推测 skip 数。

全量 E2E 首轮 262 通过/1 失败：S6 仍输入命令名称，与已批准非空根查询用于工作项搜索不一致；仅改用既有 selectCommand 空根键盘选择，保留 IME、Tab/inert、子页返回/关闭、实际剪贴板成功断言，定向 2/2。S6 修改前后仅该测试文件变化，F1–3/F5–6输入未变；等价证据 t10-gate-input-equivalence-final.json，修改后 F4 与全 F7 重跑。全F7第二轮I12重开在登录页失败：两次goto隔50ms，中间refresh 200响应被文档替换丢失，下一刷新401；M2-design §3.5明确该已知代价。I12补等待中间列表可见后再重开，保留204条和远端修改/删除全部断言，定向连续 **3/3**，不改认证协议。

最终输入和七项门禁见 t10-receipt.json；原先失败、变异补丁/恢复、trace、目录及单元日志均保留。Web/E2E 类型及定向 TS lint 零告警，76规则/16例外未放宽。该任务局部决定的代价分别是关闭后立即重置视图、空根命令需方向键选择，均由页面复验。A10/T10 完成后接续 T11；阶段整审与最终CI仍待T16，不合并。

最终全 E2E **263/263**（Chromium 261、Firefox 2，0 skip）；完整七项门禁均有对应绿结果。搜索产品/消费者输入 d602e5f28e5ffaa689bb3f45d6e003df193664d6a8351c47c1cd3dbc9b0a0f72；只改 S6/I12 前置后的最终页面输入 489af980bbe74c30e61085b3187c1f9e6a49f7921d029c402e6f6b42c6294374，程序 041adf634f68a7bfb6eeff96febf90853c7f1d5afaad3e69684c3fd04e2d6d82。检查点提交后推送并单独核远端 CI，不将本地绿等同远端绿。

### T11：旧搜索合同删除与十三实例核账

删除 WorkspaceService.searchWorkspace、五个 IWorkspace*SearchResult 和 WORKSPACE_DEFAULT_SEARCH_RESULT，并删除 T10 已不需要的旧服务 mock。workspace barrel 仍导出角色、引导和导航等真实读者所需内容；没有删除整个 workspace 模块。四个 M7 view 方法从 createView 到类末尾逐字节相同，传输测试核四个路径、body、响应原值与四种原始错误传播，只证明边界保留，不代表 M7 页面已接通。

新契约测试基线 1 红/4 绿，唯一失败为已退役搜索方法仍存在；删除后与 T10 消费者共 **4 文件 31/31**。包源变化后先强制重建 10 个依赖包，再验消费者与 web types，knip、5 文件 oxlint 零告警及关键词 76 规则/16 例外通过。历史 13 路径的模块级实例全部为零，8 个剩余实例逐一列归 M5/M6/M7 或各自领域；完整路径表见 M2 handoff 子回执。M15 临时恢复 search-menu 模块实例使精确审计仅该项为 1 而失败，字节恢复后同套 31/31。不是以编译失败代替性质反证。

输入摘要 `4e7dc6e1a49f34e93b0a80176f1881a45de7572f2a46d0300654ac0dfab6dd92`；原件在本期证据目录的 t11-*.json/log、t11-view-byte-equivalence.json、t11-thirteen-audit.log、t11-mutations.json 和 t11-receipt.json。M2 §12/M3 §5 子项完成，顶层保持 open，M7 迁移和 M4 总收尾未关闭。A11/T11 完成，接续 T12。

T10 检查点 `1047507d03435366349a5cf1a9a47f3f23d7260a` 已推送，[CI 41223646422](https://github.com/open-nerve/NerveProject/actions/runs/41223646422) 最新核验 web 成功、server 仍运行；最终结论另记，不能由本地绿推断远端绿。


### T12：新时钟进程的自动归档页面

复用 ClockWorld/clockTest 与原有真实 API seed，没有改归档算法或增加 fixture owner。旧进程页面真正就绪后关闭浏览器，推进 744h 并确认旧进程退出、新进程实例不同；在新进程重新 API 登录写入控制项，再通过真实登录表单进入归档列表。业务写后数据库微秒水位与新进程 startedAt 均有独立 SQL 布尔断言，fresh completed attempt 的 attempted_at 同时晚于二者。旧终态已归档，旧非终态、新终态和刚改终态的三个控制项保持活跃。

从归档行打开 peek，再进入既有编号详情；系统动态显示 Nerve 且无用户链接，手动恢复后显示真实操作者链接，SQL 读回分别为 null/archive 和本人/restore。两张截图及 old/current URL、startedAt、水位、attempt、freshness、活动行作为附件保存；无 SQL 业务写前置。首次页面仅因误预期 archives/id 地址失败，真实 peek 全屏使用 browse 编号地址；按合同纠正测试，未改产品。

M16 改用 finalized_at 判断新任务，原 API 的 crossing 负例准确失败；M17 将 null actor 当停用用户，页面的 Nerve 断言准确失败。两处字节恢复并重建后，**I16 2/2**（API 1、page 1）通过；web/E2E 类型、单文件零告警通过。最终输入 `5fdb4b1859ee992249d343a5fb1fc5d6ed88d21e9f8a7c71f0a4d2837dbe1ac4`，程序 `bab9227b037db3634dfa637f66046683cd0f7a7845f976b4168bea47649da822`。原件见 t12-*.json/log、m16-fresh-*、m17-system-actor-*、t12-mutations.json、t12-restored-results 与 t12-receipt.json。A12/T12 完成，接续 T13；三引擎仍归 T14。


### T13：复制消费者单一反馈与真实 LAN 拒绝

创建 toast、评论 helper、布局快捷 helper 三处迁既有 useCopyLink；T8 已迁命令复制，本轮连同 T5 Markdown 复验。创建 toast 删除 copied/定时器和吞错，按钮始终可见且使用已有译文；评论保留 void 受控入口及原 scope/follow，只让 hook 负责复制反馈；布局返回 hook Promise。删除最后三个 M4 copy-hook 例外，共四处闭合，M6–M8 保留，关键词 **76 规则/13 例外**。

初次三文件 5 红/38 绿抓到创建无反馈、评论错误通道和布局拒绝逃逸；后续六红来自新测试把 fake-i18n 原样 key 误当译文，按既有测试替身合同纠正预期，真实翻译由页面核。最终五文件 **67/67**，覆盖持有/成功/拒绝、当前 origin/评论片段、无提前/重复成功、零业务写，以及 T8 命令、T5 Markdown、P14 六复制与评论原回归。M18 加外层提前成功使创建两例在等待断言准确红，字节恢复后 67/67。

C8 初轮 **12 失败**不是产品拒绝丢失：LAN 下现有 execCommand 兼容在用户手势中真实成功，测试错误假设 HTTP 必拒绝；空浏览器授权列表也不能禁止该手势兼容路径，单例仍成功，原失败截图/日志均保留。设计 C8 的“失败时提示”是条件要求，不移除 M2 兼容能力。小 helper 绑定本机非回环 IPv4 的独立 nerve 子进程、同套自有测试库，真实登录；键盘选中控件后按 navigator.userActivation.isActive 条件等待失效，再调用真实控件，由浏览器本身拒绝，未替换 clipboard/execCommand，不使用固定 sleep。与真实键盘复制成功明确区分。

最终 **Chromium C8 12/12，0 skip**：编号、详情、外链、关系、子项、peek、评论、列表、创建提示和命令 ID/标题/URL；每例附真实 origin、secureContext=false、activeGesture=false、asyncClipboard=false，唯一错误、无复制成功/导航/业务写，网络/CSP/控制台守卫不变。复用 power-k 的键盘选择部分为 highlightCommand，selectCommand 原激活保持；helper 生命周期包含 newContext/newPage 失败清理。Web/E2E types、7 TS 文件零告警与格式通过。

最终输入 `757bb23328732fcb91af61bcc701700ceef859b2350313b0bc4f6dbc0fd30e79`，程序 `ec384903fe32e0405e2871d4c155801bd98d1cd94f0b2db2f435a389fdcf8b69`；t13-*.json/log、t13-c8-*-results、m18-outer-copy-success.patch、t13-mutations.json、t13-receipt.json 保留全部前后证据。A13/T13 完成，接续 T14 三浏览器与完整检查点；不合并。


### T14：三浏览器矩阵、路由退役与快捷键

八个文件、每引擎 41 个具名页面（C6–C9 与 I11/12/13/14/16/18/20）实际运行通过，0 skip；逐行题名及证据见 [浏览器矩阵](P15-browser-matrix.md)。先列出 123 个匹配仅作选择审计，随后分别执行。C9 真实重新加载 member→guest→member 后旧编辑器与确认窗不复活；I18 持有旧范围真实 200，经 A→B→Back→A、同词重查后只能显示新结果，关闭重开为空。

两处真实产品缺口均在既有命令面板生命周期内修复：provider 原来只在工作区变化/卸载退役，同工作区换项目或编号仍留面板；现把路由 projectId/workItemIdentifier 纳入原 layout cleanup，真实 PowerKStore 三例先红后绿。Ctrl/Meta+K 关闭只 preventDefault，事件继续冒泡到文档的“打开”监听，导致面板立即重开并保留查询；wrapper 消费该快捷键的传播，接入真实 ShortcutHandler/registry 的两个消费者先红后绿。聚焦四文件 **48/48**，没有新权限 owner 或延时。

首次 Firefox 27 绿/14 红，主要是 C8 实际浏览器安全诊断与 C7 重载期间评估的布局警告，加上上述快捷键问题。C7 等待真实重载 load 再观察；C8 必须精确断言一次 LAN 密码表单诊断（本次登录 URL）及一次 execCommand 拒绝（同源且已请求的 assets JS/正行号），不是过滤任意警告；缺失、多余或外部来源都会失败。修复后 Firefox **41/41**。

首次 WebKit 34 绿/7 红：macOS Safari 遵既有 P13/P14 的 Option+Tab 到按钮；六个拒绝用例必须期待引擎实际发出的精确 Failed to load resource 状态/数量。I13 的初始 route 拦截会关闭缓存并产生未使用预加载警告，沿 P14 已证实方案先完成真实列表加载，再安装必要拒绝/分页拦截并在应用内导航，未隐藏 warning。七项定向通过后 WebKit **41/41**、Chromium 最终 **41/41**；I13 前置改变后 Firefox **3/3**，其它 38 项逐文件核输入差异仅 WebKit 条件及单测语义标记，未冒称整套再次运行。

矩阵最终输入 85b650eaa2d3796749c5d8d6a4529e9ecf9506a19b26eaf4412fb6f8c6217a39，程序 4e7df898b364b478f07ab515ab075861e389a874e521ef544c46c8df19fa3bfe。完整 F 的前四项通过；Knip 首次发现新配置未由入口脚本引用，按其它矩阵登记 e2e/package.json 的 test:m4-collaboration，从 F4 复跑，未加忽略项。只有脚本登记改变，F1–3 与页面生产输入相同，等价证据见 t14-input-equivalence.json。最终 F4–7 输入 6048df79212af9baccc1d6262f6e1fbae005b5aba26d91efc49461a8cc5570e9；七门禁全绿，前端静态 54/54、无缓存 18/18 测试任务（**294 文件/3108 Vitest，另 Node 5**），默认 E2E **277/277**（Chromium 275、Firefox 2），0 skip。报告、trace、初始失败、分包日志、构建 hash 与后置提交回执均留在 t14-*。

T10 的准确提交 1047507d03435366349a5cf1a9a47f3f23d7260a 的 [CI 41223646422](https://github.com/open-nerve/NerveProject/actions/runs/41223646422) attempt 1 已核 server/web/e2e 全部 success。A14/T14 完成，本次检查点提交后推送核独立 CI；继续 T15，不合并。


### T15：死成员与边界清账

逐文件查引用后删除十组无读者链：CommentCard 的 enableReplies/entityId（含 display 与全部调用者）、CommentBlock 的未读 ref、创建提示的 projectId（实际取 canonical issue.project_id）、未挂载 modalContainerRef 的 ResizeObserver、恒 false 的重复项弹窗状态/宽度分支及表单透传、菜单工厂无人提供的 customEditAction、两个未读取返回扩展（workItemLink 和 actionHandlers spread）、无人设置的 hideFooter。CommentCreate 的 entityId、详情真正使用的 customEditAction、M7 收集箱同名 props 与公共 EditorRefApi 全部保留。

原 M1 附录脚本用仓库已锁定 TypeScript 重跑，全仓候选 prop 372→371、member 877、export 21；这些是扫描候选，不能声称全仓死成员清零。当前生产变动只剩三个候选：命令图标的 className 为第三方 ComponentType 合同，搜索结果 issue/project 为数组元素的真实读取，全部人工核调用链保留。测试候选按 mock 组件输入、editor/ResizeObserver 运行时接口和断言记录字段核对，不把 AST 看不到的框架调用误删。手工审查抓到的已传参未读取链不依赖 AST 是否报出。

四分类记录：真实死成员删除；真实读取保留；第三方/协议合同保留；其它领域的八实例及 M7 DTO/props 按既有归属保留。初步把 handleSearchMenuClose 当成死参数的判断被全仓引用推翻：TopNavPowerK 是实际调用者，因此此任务不删除；同时发现搜索 gate 只认弹窗开关，顶部宿主需在 T16 整审确认并修复。这里不把静态引用少当作无业务用途。

15 个受影响单元文件 **132/132**，Chromium **21/21**（C6/C7/C9、I11/12/14 与评论/布局/创建 C8），类型/翻译/格式/关键词 54/54、knip 通过。首次格式检查仅 quick-add 的 JSX 换行失败，格式化后重跑通过。全分支明确列出 88 个 TS/TSX：通常规则实际扫描 87 个零告警，配置文件因根 ignorePatterns 被跳过；以相同规则、仅去 ignorePatterns 的临时配置显式扫余下 1 个，亦零告警，未冒称 --no-ignore 覆盖配置排除。文件表、三生产候选、完整扫描及前后输出均在 t15-*。

13 历史实例再核零，四 M4 copy-hook 例外为零、四 workspace-address-route 路径及正反例均保持，关键词 76/13 无命中。M15 的恢复实例反证、M12/M18 的行为反证已核原始 hash 后复用，未重复冒充新运行。本 Task 未修改守卫内容或扩大全局 no-shadow。输入 `89dfffde60176753f0678b3ae093aefc7d5605540516cf904d70580dafcff93d`，页面程序 `864e5108dc339c4f07e7c47f1f8d3b7d9495deb69d9fa417e247c663467332e4`；单位测试/AST 早一版仅相差 quick-add 格式，提交后同套再验。差异清单和四份输入子回执同步；所有 handoff 顶部 open，P16/M4 收尾未提前关闭。A15/T15 完成，进入 T16 deep-review。


### T16：整分支审查、修复与交付门

主控顺序执行 deep-review 的逻辑、集成、覆盖三个视角，范围为 main 7a81ea18 至最终候选及 dirty；没有派子代理，也未称三位独立审者。[整体审核](../../../reviews/re-24-M4P15整体审核.md) 记录 R1–R8、根因、局部裁定、失败与恢复；变动产品全文/删除基线/关联测试、实际调用者和受影响文档均纳入。

顶部搜索是另一真实宿主，原全局 modal 开关导致顶部不读；现显式透传各自 isOpen、保持服务端排序、关闭统一 context.closePalette，沿原 useIssueSearch 与目录 owner。顶部回调补展开状态依赖，多选遵 closeOnSelect，关闭恢复上下文显示，路由变化退役；实际输入/清空按钮具名。时区/每周首日/语言复用 followInSession，只向当前登录反馈并保留原生 code。全局 ShortcutHandler 忽略已消费/IME 按键，解决 document hydration 同节点监听的问题；无新事件总线、队列或权限 owner。

准确 T14 提交 792449e4702edc0128b8243cd02e1cfdfbcebb81 的 [CI 41229160601](https://github.com/open-nerve/NerveProject/actions/runs/41229160601) 是 server/web 成功、E2E 276 通过/1 失败，保留原失败。通过 GitHub 连接器取得 job 124622352597 的日志及 artifact 13053772034；ZIP SHA-256 为 8af1ef4c9dce01687de76fb97830edfb08c12f227835000b2de8054c94d732ab，没有提取凭据。trace 证实第二次 Cmd/Ctrl+K 在关闭约 83ms 后到达，退出动画保留旧 combobox，旧 wrapper 再次关闭而不是重开。wrapper 现只在 isOpen 时拥有按键和选择动作；关闭期间不误进子页/执行动作，document opener 可以重开。

新增留存退场 DOM 的 Ctrl/Meta 与动作消费者共 3 条真实行为红，C7 页面分别证实无法重开和关闭期间误进子页；修复后单元及真实三引擎均通过。测试只延长真实 CSS 退场并断言 data-leave，不用固定 sleep 或 mock 产品开关。修复前直接 inline duration 被过渡过程还原，红运行仍已观察到真实 data-leave；最终测试改独立样式精确固定这一边界。早期顶部 first-item 焦点猜测被单测绿推翻，实际 isOpen 依赖、closeOnSelect 和上下文恢复分别有页面红；两个新 E2E helper 调用错误和一次聚焦路径少匹配均明确作为夹具/命令错误保留，不冒充产品反证。

最终聚焦八文件 **89/89**。T16 完整三浏览器各 **43/43**；退场快捷键修复后各补跑 **13/13** 命令/搜索/命令复制；选择动作保护后主题退场页再各 **1/1**。未受影响的 30 项、随后未变的 12 项有路径/hash 和控制流复核，不冒称最后输入又完整执行 129 项。精确题名、每轮输入/程序摘要见 [矩阵](P15-browser-matrix.md) 和 t16-final-matrix-equivalence.json。上述通过的运行均为 0 skip，严格网络/CSP/控制台守卫保持。

本期实际 92 个 TS/TSX（含配置）零告警；按实测将 web 全仓 warning cap 112→108。新触及的 ShortcutHandler.setEnabled 零调用，删除后全仓 member 候选 876、prop 371、export 21；三个生产候选是图标 className 合同及结果对象 issue/project 真实读取，不误删。十三历史实例为零，八个其它领域实例仍按责任保留；76 条规则、13 个例外，四 M4 copy-hook 例外已零，四 route 正反例与 content 原内容均保持。

核 M01–M18 的 23 次有效反证（M15 为静态审计，其余行为断言）及一份成功 build 原件：raw hash、输入稳定、变异前后恢复一致；25 个文件记录中 18 个最终字节一致，7 个变更逐段证实未去掉被测性质，详见整审。没有重跑却记新执行，也没有将 Go 普通测试写成 race。

完整七门禁首轮的 F3 只有 workspace/adapter/postgres 因 Docker 映射端口查询超时失败，该包共 119 条测试/子测试记录都来自同一次 pgtest 初始化失败，业务断言未开始；其余 49 包通过。保留 t16-full-f3-test.log/json 和 t16-go-infrastructure-failure.json，未操作已有 Docker/容器；最终完整重跑见下一节。

F4 发现新增测试清理函数的 Node.remove 类型不匹配，局部改用 parentNode.removeChild，之后 F4–7 全部通过。修改只有测试清理，补跑 Firefox、WebKit 主题页 2/2 通过；此前单独 Firefox 在 globalSetup 因临时数据库端口绑定超时退出，页面未执行，原失败保留。产品、断言和前三门禁输入未改变，相关等价记录见 t16-full-gate-equivalence.json 与矩阵。

## 10. 最终验收与交付

状态：整审及后置 CI 的局部问题均修复复审，完整七门禁通过，A1–A16 和 T1–T16 已登记；最终候选 HEAD 按下文后置收据核准后才交付，CI 未通过时不得报告阶段完成。本期不合并，主目录 main 保持基线。

| 验收 | 对应证据 |
|---|---|
| A1 | T1 37 单元、I12 2 API/2 page、M01；最终矩阵 I12 两页 |
| A2 | T2 19 单元、M02、C6 实际筛选/排序/焦点三引擎 |
| A3 | T3 49 前端/19 编辑器、M03/M04、I11/I20 评论权限与真实输入 |
| A4 | T4 55 单元、M05、I11/C6 组合码点、仅本人、归档只读与本站 picker |
| A5 | T5 43 单元、M06/M07、I13 三页的分页/还原/真实 Markdown；M7 合同保留 |
| A6 | T6 77 单元、M08/M09、I14 四页；完整 payload、保存失败不发布、GET-only retry |
| A7 | T7 63 单元、M10、C9 四页；真实编辑器 readiness、两种保存、关闭/Back/迁移 |
| A8 | T8 37 单元、M11、C7 状态与删除；T16 多选确认/关闭复验 |
| A9 | T9 17 单元、M12、主题当前拒绝/成功/旧登录；四 route 正反例与 content 原样 |
| A10 | T10 26 单元、M13/M14、I18 API 与六个最终页面；T16 顶部/弹窗同合同 |
| A11 | T11 31 单元、M15、13 路径精确零；四 view 方法字节不变，M7 责任保留 |
| A12 | T12 I16 API/page、M16/M17；三引擎新进程/新登录/写后 attempt/系统显示 |
| A13 | T13 67 单元、M18、C8 12 页真实 LAN 拒绝；四 M4 例外零 |
| A14 | T16 三引擎各 43 页，随后 13 页/1 页按影响复验，0 skip |
| A15 | T15 十组死链、T16 setEnabled 删除、92 文件零告警；三合法候选、守卫/交接边界 |
| A16 | 完整七门禁、主控三视角 deep-review 与 R1–R8 闭环；最终候选 HEAD 的准确 CI 和证据 manifest 以本节后置收据核准 |

整审提交 `438723335779f958e8209a1ccf625f80182abbef` 的 [CI 41234416035](https://github.com/open-nerve/NerveProject/actions/runs/41234416035) attempt 1 为 server/web 成功、E2E 278 通过/1 失败。后置审查确认 cmdk 在关闭退场期间消费 Control+K，已按 R8 局部修复；真实单元和页面复现同一失败后转绿。原始 run/jobs、日志、trace 和产物摘要保留，T14 原失败也不覆写。最终候选准确 CI 由后置收据核准。

最后候选一次包含经完整验证的产品修复与验收文稿，提交前核产品/测试字节与最新完整门禁一致。最终准确 40 位 SHA、远端 ref、CI run/attempt/jobs 及证据清单摘要写入本机 `final-delivery.json`；该文件后置生成，只有三任务全绿、工作树干净且本地/远端 SHA 一致才交付，不用旧候选的 CI 代替，也不回写自身 SHA 形成提交循环。

准备期原件仍在主目录 `.superpowers/sdd/P15-preparation-20261031/`（30 份文件）；实施期与准备期均纳入 evidence-manifest.json 的逐文件摘要和空目录记录，未来获准合并后的归档不得遗漏任一处。

本机原件在本 worktree 的 `.superpowers/sdd/P15-web-collaboration-drafts-search/`。运行收据保存 HEAD、dirty 文件摘要、argv、exit、raw hash 与程序 hash；历史失败不会被新绿覆盖。最终 `evidence-manifest.json` 和 `final-delivery.json` 后置核对完整证据与本地/远端准确 SHA，归档清理待另获授权合并及 main CI 后进行。


后置 CI 修复前的完整七门禁 t16-full-retry-f1–f3 与 t16-complete-f4–f7 均 exit 0；前者输入 e14c9de376235a7f7b2ae2945213b1f4e4fd9df71df529acdd18097833d58542，后者输入 3531565df6210e93b8b8ec3e940cb7a411af9c6c48e0f40883aa84c8472ca5f5，每项执行期间输入稳定。F4 首轮仅测试样式清理的 Node.remove 类型报错，改用 parentNode.removeChild；F1–3 的服务端/生成输入不变，等价记录 t16-full-gate-equivalence.json，不重跑并不冒称同一摘要。Go 普通测试 51 个包通过、16 个包无测试文件；前端强制无缓存测试 296 个 Vitest 文件/3130 项，另 Node 5 项；默认 E2E 279/279（Chromium 277、Firefox 2，0 skip）。程序 9be9d742306019dd0a990422cce39bdd72495a32ac75c1c3c68d2ef22fa06c6d。原始日志、分包日志、完整结果快照见 t16-full-summary.json、t16-full-package-test-logs 和 t16-full-e2e-results；首轮初始化失败不覆盖。

T16 自决的理由和代价见整审及本节：宿主状态显式化仍是同一组件树合同，关闭重置查询/上下文符合已批准 spec；其它偏好沿既有登录保护；严格 lint 下调额度；测试同步真实退场/页面就绪而不加生产延迟。没有待裁定架构问题。

P16 的全局 dead/no-shadow、C12 与总守卫、M4 收尾逐条 §14/总体 9.4/M5–M8 转交均保持后续责任。输入 handoff 顶部 open；评论增量窗口以外的远端修改需关闭重开、LAN 有手势仍可能复制成功、M7 领域展示未接通等限制不扩大承诺。


## 11. 后置 CI 的 Ctrl+K 修复与最终候选

43872333 的 CI 41234416035 attempt 1：server/web 成功、E2E 278 通过/1 失败。详细日志和 trace 由 GitHub 连接器直接读取，无需负责人代取或提取凭据。失败是 C7 在 Linux 的 Control+K 被退场 cmdk 当作 Vim 导航消费；原 Mac 页面只按 ControlOrMeta→Meta，wrapper 单元的 cmdk mock 未覆盖这一分支。data-open 在退场也保留，不据它误判为已重开或共享动画库故障。证据、摘要和复审见 [整体审核 R8](../../../reviews/re-24-M4P15整体审核.md)。

移除 cmdk mock 后，Ctrl 单元真实失败（1 失败/6 通过）；页面显式 Control 同样准确复现。沿现有组件把 vimBindings 绑定 isOpen，关闭时释放键、打开时行为保持。最终 C7 在每个引擎显式验证 Control+k 和 Meta+k；89 项聚焦测试、三引擎各 13 项以及最终主题页各 1 项通过。首轮新测试的顺序循环触发 no-await-in-loop；改局部函数按顺序调用，不豁免规则、不并发操作同一页面，92 文件再次严格 lint 零告警。

局部裁定：问题限于既有命令组件内部的关闭阶段所有权；改动一个生产文件的现有库参数，无协议/领域/共享库变化。代价是退出中的列表停止 Vim 导航，避免它与全局重新打开冲突；原命令、复制、搜索和所有写的确认/拒绝合同均回归通过。

修复后的完整七门禁 `t16-ci-full-f1–f7` 全部 exit 0，在同一输入 `f6d7c5f62b1145f677d063aa745c4cb0a65b64855534d048bdab859b3993335e` 下顺序执行，期间无文件变化；程序 `329e0da55edc82fd767d0bb7bb7ede8ec104ae3f8c3c3a6b2124f522de0d3ef0`。生成检查、Go lint、Go 普通测试 51 包通过（另 16 包无测试）、前端类型/格式/lint 54/54、knip、强制无缓存前端测试 296 文件/3130 项加 Node 5 项，以及完整默认 E2E **279/279**（Chromium 277、Firefox 2，0 skip）均通过。原始收据、八包测试日志和完整页面结果已独立复制到 `t16-ci-full-summary.json`、`t16-ci-full-package-test-logs/`、`t16-ci-full-e2e-results/`；旧轮与原失败保留。

最后候选一次提交这项修复、完整验收文稿及证据入口；本地通过不代替准确最终 HEAD CI。后置 final-delivery.json 必须验本地/远端同一 SHA、三个 CI 作业全绿、主目录 main 未变、两处证据逐文件摘要，才报告阶段交付。没有合并、清理或启动 P16。

## 12. main 首次 CI 的 I13 测试修复（2026-11-01）

负责人授权后，P15 已由 de9a36fc --no-ff 合入 main b79c25f1；阶段最终 CI 41238185478 三任务全绿。首次 [main CI 41263412461](https://github.com/open-nerve/NerveProject/actions/runs/41263412461) attempt 1 则是 server/web 成功、E2E 278 通过/1 失败。I13 把编辑器初始化的格式迁移 PATCH 错当成还原而合成 403，真正还原收到 200 后弹窗正确关闭；不是已知 W11，也不是产品吞错。三个请求的原始 body/status 与 trace 摘要见主目录集成根 `i13-failure-analysis.json`。

按 02 在新的 `codex/m4-p15-ci` 工作树局部修复：先实际加载详情和编辑器、等待迁移确认，返回列表后才开启分页/拒绝拦截；拒绝仅针对普通还原；发出时记录 body，两次还原与 Original HTML 逐值相等，正常格式迁移不再算作重复提交。WebKit 真实详情预热消除测试 routing 与预加载缓存相互干扰，严格 console/CSP/网络检查保持。只改一个 E2E 文件与审查记录，产品/API/权限/队列/依赖/phase 不变。

主控顺序完成逻辑、集成、覆盖三视角的 [deep-review](../../../reviews/re-25-M4P15集成修复审核.md)，没有待决架构问题。I13 三引擎各3项，共9/9；旧总 PATCH 计数反证准确失败 expected 2 / received 3，恢复逐hash相同。早期扣住前一写却等待后一写的夹具假设被既有队列推翻，WebKit预加载警告和一次错误选择器原件保留，没有记成产品反证或隐藏警告。

完整本地七门禁在同一输入 `d82bfe2eae1187b319892b0b259b1716e2339effe38dd16a31e7df4a60240a75` 下全部 exit 0；Go普通测试51包/另16包无测试、静态54/54（36缓存）、knip、无缓存296文件/3130 Vitest加Node5、默认E2E **279/279**（Chromium277、Firefox2，0 skip）。最终程序摘要 `17ef9aeb121b49d128cf11dfe8910b8704f98ee2179a59a119db506f920a0c10`。七项运行期间输入稳定；事后仅补本节与审核的已得结果，不改变任何测试或产品字节。

修复原件在本工作树 `.superpowers/sdd/P15-ci-i13/`，`final-summary.json`、分包日志、完整页面结果和所有历史失败均保留。主目录 `.superpowers/sdd/P15-integration-20261101/` 保存原 main 失败元数据、日志与 artifact；ZIP SHA-256 `89cde27ef838dade8796238a444f87199213c03be47f5215ccfa54a40a894685`，通过 GitHub 连接器取得，没有提取凭据。修复分支准确 CI、后续 --no-ff/main CI 与归档清理由后置集成收据核准；P16 仍只准备、不实现。
