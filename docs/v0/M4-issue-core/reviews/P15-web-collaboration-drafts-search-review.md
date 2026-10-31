# M4/P15 协作、草稿与命令搜索：预检、实施与交付记录

日期：2026-10-31。状态：预检通过，已授权连续实施；T1–T4 完成，T5 接续。§1–8 保留准备期快照，实施事实见 §9；阶段末审查与交付尚未执行。基线 main 7a81ea18d8e9d80f079e4e30ae0e90b11be23ef9，分支 codex/m4-p15；[spec](../specs/P15-web-collaboration-drafts-search.md)、[plan](../plans/P15-web-collaboration-drafts-search.md)、[deep-review 预检](../../../reviews/re-23-M4P15预检审核.md)。

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
