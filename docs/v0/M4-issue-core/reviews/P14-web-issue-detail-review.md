# M4/P14 详情与弹窗：准备与预检记录

日期：2026-10-31。状态：**已授权实施，逐任务推进中**。基线 `8bb4bbcb52a224ace4cb2f9d383bca9b97a022bf`；[spec](../specs/P14-web-issue-detail.md)、[plan](../plans/P14-web-issue-detail.md)、[文档预检](../../../reviews/re-20-M4P14预检审核.md)。§1–7 保留文档预检时点，实施结果从 §8 续记；A 条款待完整证据再勾选。phase 已进入 M4/P14。

## 1. 前置与方法

P13 合并、main CI、归档与清理的最终事实见 [P13 review §15](P13-web-filters-archives-profile-review.md#15-授权合并main-ci与归档清理2026-10-31)。本期从该准确 main 建 codex/m4-p14，主目录一直 main。原生 create_worktree 无法指定负责人约定的路径与分支，因此沿 using-git-worktrees 用系统 git worktree add；没有安装依赖或改产品源码。

按 02 §3 以现有源码、生成合同、测试和消费者静态核对替代原型/逐 Task 产品重放，不虚构红绿、生成 hash 或执行数量。brainstorming 的 fresh builder-check 独立读取 spec 与关键搜索/链接/编辑器合同，初次报告 nothing important left；主控随后按 deep-review 的逻辑、集成、覆盖三个视角顺序复核，不能算三个独立审者。独立补查 spec/plan 提出 R5，修订后的定点回读确认 resolved；具体阅读范围和静态结果保存在准备原件。

主项目 `.superpowers/sdd/P14-preparation-20261031/` 保存 builder-check.md、consumer-inventory.json、source-review-inventory.json、precheck-static.json、document-links.json；准确提交/远端核对另以 precheck-delivery.json 后置记录，不循环引用自己的提交 SHA。未来清理必须把本准备根与实施 worktree 的 `.superpowers/sdd/P14-web-issue-detail/` 一起归档。

## 2. 验收映射

| 标准 | 静态依据/实施任务 | 当前产品状态 |
|---|---|---|
| A1 | 原生 id/编号路由、header/body 与 detail-host/WorkItemLevelModals，T1/T14 | 未实施 |
| A2 | title-input、label-select、FieldError/useRefusedFocus、I2，T2 | 未实施 |
| A3 | 既有 confirmed autosave/boolean flag；弹窗 migration 缺口，T3/T4 | 未实施 |
| A4 | create/edit base/form、项目/标题字段和 I1，T4 | 未实施 |
| A5 | ModalCore、draft discard、document Escape，T5/T6 | 未实施 |
| A6 | issue-search 合同、原生父项全部消费者、I8，T6 | 未实施 |
| A7 | ExistingIssuesListModal、relations、M7 查重/父项适配，T6/T7 | 未实施 |
| A8 | link resource、URL 字段、metadata 空/终态、data favicon、I10，T8/T15 | 未实施 |
| A9 | 必需 delete onSubmit、子树语义/目录断言 I17，T9 | 未实施 |
| A10 | editor 三条 runtime 引入、settings host、emoji dead 检测，T10 | 未实施 |
| A11 | 五扩展死序列化、HTML fixture 与实际 build CSP，T11 | 未实施 |
| A12 | M4 image 配置、已删除入口、P12 表格链接与剩余计数入口，T12/T13 | 未实施 |
| A13 | OwnOnly、S2 403/404、真实双标签页已确认旧答，T14 | 未实施 |
| A14 | copy hooks 真实签名/反馈与六消费者，T15 | 未实施 |
| A15 | F、显式三 project、非零执行、至少16有效变异/恢复、dead/oxlint，T16 | 未实施 |
| A16 | 最终基线..HEAD deep-review、修复复审、准确分支 CI，T16 | 本次仅文档预检 |

## 3. 准备决定

D1–D8 见 spec §3，理由和代价逐项列出。核心是复用已有 root/确认/宿主能力、无源 id 的新建父项普通搜索、有源 id 才用 for=parent；元数据 UI owner 有期限回读，不增加常驻全局轮询；五处无人使用的 Markdown 序列化按 P14 既定裁定删除但保留 P4 安全 fixture；M7 仅共享选择器读取/类型适配，M5 图片和 P15 草稿/评论等边界不扩大。

T13 表格链接 P12 已完成，回归加实际剩余 all-properties 子项计数入口的片段修复。ProjectService 当前只有两处 search 读者，迁移后删除整个空类，不保留到 P15。它们是源码事实导致的任务落点校准，不是新架构或协议变更。

实施局部裁定 T13-C2：异步编辑器使已打开筛选面板的按钮移动，而共享 FiltersDropdown 只随滚动/窗口尺寸变化更新定位。修复限于此组件内部现有 Popper 实例的定位生命周期，保留公开 props、领域消费者、权限与数据链；不需要跨模块协同改协议。代价是打开面板期间逐渲染帧读取一个按钮的矩形，关闭及卸载必须释放；以组件生命周期、真实 lazy 页面和其他消费者回归核验，结果见 §8/T13。

## 4. 预检发现与修订

| 编号 | 严重度/根因 | 文稿修订与复核 |
|---|---|---|
| R1 | Major，局部计划遗漏：只列 lazy 两个 editor wrapper，lite-text barrel 的 toolbar→TOOLBAR_ITEMS 仍能提前加载 editor | spec §2.9、plan T10 明确第三条 runtime 链、同一延迟边界/删除零读者导出，并由真实构建 Network 验 C10 |
| R2 | Major，旧设计路径已过时：T13 初稿把 P12 已修好的表格列当缺陷并预期行为红 | 标记 ceb154a9 为已有输入；表格补绿回归，真正剩余 #sub-issues 在 all-properties.tsx，T13 增其真实消费者/无效片段测试，不制造假红 |
| R3 | Major，复制接口不够具体：hook 本身捕获拒绝并提示，消费者若 await 后另提示成功仍会假成功 | spec §2.10/plan T15 写两参数 Promise<void>、hook 单独负责提示；relations/sub-issues 已是 hook，操作类型和调用一起改，不留 void 假接口 |
| R4 | Minor，任务留下查路径/空服务条件：现有 detail-host/use-copy-link 测试应直接指定，ProjectService 只剩方法本身 | 所有既有/新增目标明确；不存在的 row-scope 猜测改为新增 all-properties.test；T7 删除整服务/barrel/专用类型，P15 仅回归零残留 |
| R5 | Important，独立补查发现任务顺序：T5 要验真实父项页面，但 T6 才替换打开即发出的旧搜索接口 | T5 先完成关闭策略/嵌套单测及其他原生弹窗页面；父项 C4 页面放 T6 Step 4 随原生查询闭合，A5 映射同步，不扩大旧 API 失败白名单；独立定点回读已确认 resolved |

以上只修订计划，没有改变源码。独立 builder 的 R5 是局部任务依赖，由主控修订，无需负责人重新裁定；后续读者盘点和主控三视角用于收紧可执行性。当前没有未决架构级问题。实现若发现新风险，仍按纪律先提交再上报。

## 5. 与原设计/原型方法的差异

保留 16 Task 与所有 I/C 完成线。P11 已有跳转/描述合同、P12 已有选择器/表格 browse、P13 已有归档权限分别作为输入；M4 design §12/P14 同步这些事实和实际剩余路径。EMOJI 引用数量以当前 consumer-inventory 为准备快照，历史计数不再作为未来删除门。

没有实施原型、逐 Task 重放、浏览器产品测试或七项门禁。本次静态检查只证明文档结构、路径/调用/合同、链接、phase/产品 diff 及 P13 集成证据对应正确；不把 P13 测试/CI 填为 P14 的产品通过。

静态脚本核定 16 个任务、80 个未执行步骤、16 条未勾选验收，路径区分既有文件与计划新建文件；新增/变更文档无意外失效链接，设计中 4 个原有 M5–M8 最终 closeout 链接仍明确待收尾创建。产品 diff 为空，phase 保持 M4/P13，主目录 main 与准确基线一致，P13 CI/清理回执及归档清单哈希相符。最终文件哈希和实际检查数量见 precheck-static.json，提交后远端身份见 precheck-delivery.json。

## 6. 移交与下一步

预检完成后等待负责人启动，从 T1 开始；本轮不实现。阶段最终需固定准确基线至最终 HEAD 做 deep-review，局部修复复审、准确分支 CI 全绿后先保留分支，等负责人安排集成。

M1/M2/M3 输入子项映射见 spec §7，顶层 handoff 仍 open。P15 接时间线/评论/表情/历史/草稿/搜索和余下复制，P16/收尾接全局清理/C12/设计 §14/总体设计 9.4/M5–M8 closeout。不以本期准备关闭整个 M4。

## 7. 已知限制与证据保留

P13 main 第一次 CI 为已知 W11 偶发；UI 日志与一次失败作业重跑证据保留在 P13 集成根，最终结论看 §1 指向的后置收据。P14 未执行产品测试，未声称上述潜在实现缺口已修复。公开 CI metadata 可核 SHA 与作业结论，详细日志/重跑沿已授权浏览器，不提取凭据。

## 8. 实施接续（2026-10-31）

负责人已授权完成 P14、阶段末 deep-review 和局部修复，先不合并。实施证据根为本 worktree `.superpowers/sdd/P14-web-issue-detail/`；每次 run.py 回执记录 cwd/argv、HEAD 与 dirty diff/新增文件哈希、raw/exit，准备根继续保留。

### T1 入口与共同路由身份

编号读取仍沿原 useSessionSWR 和 root；仅将这一次读取移到路由 layout，经 Outlet context 给正文、经已证明 id 给页头，取消页头自行查询编号缓存。共同 Fragment 身份含原编号、workspace、服务端解析后的项目/工作项、既有权限 identity 与 loginId。没有新权限观察器、第二缓存或后端协议。T1 将 phase 改为 M4/P14，76 规则/24 例外检查通过，无到期延期。

原生 id replace、401/403/404 拒绝、A→B→A、真实页头删除框、加载/拒绝不露缓存操作和权限 ABA 与既有详情宿主回归共 23 项；基线原宿主 13 项绿。行为红 t1-red-final 的两项分别为缓存页头操作和共同边界内局部状态残留；前期夹具/类型错误单独保留，不当性质红。M01 去共同 key 检出后逐字节恢复。共同 key 的独立局部状态用宿主探针隔离验证，实际菜单/删除框另用真实组件，不能冒称浏览器证明；页面与会话 A1 剩余归 T14/T16。

局部决定：路由上下文只传已有 SWR 读取结果，不创建业务读模型；代价是正文依赖自己的 layout，已由真实嵌套路由测试核对。编号索引仍有 WorkItemLevelModals、命令面板和导航真实读者，保留。原生 id 后的第二次 core GET 来自已有 sub-issues 父项权威核验（调用栈诊断已留存），不误当路由重复请求删除。

### T2 详情字段反馈与 I2 属性

标题保留原确认自动保存与 prepareUpdate，只在两个保存入口前做本地校验；空白或超过 255 个 Unicode 字符不发请求。服务端按 rune 计长，移除按 UTF-16 单元截断的 maxlength，以一致的字符计数展示与验证。既有 Field/FieldError、fieldErrorKeys 和 useRefusedFocus 负责描述和拒绝后聚焦，普通输入只清理旧错误，不移动焦点。复用现有英文/中文文案，无新错误协议。

| 属性/性质 | 真实 UI、请求与持久化证据 | 补充回归 |
|---|---|---|
| 标题 | I2 页面空白、256 字符、真实服务端 NUL 拒绝；recordFocuses 记录聚焦时 invalid 和描述；合法值 PATCH 后落库 | detail-fields 6 项：拒绝保留/重试、普通输入、卸载校验、255 emoji |
| 状态/优先级 | Backlog→Done、none→high，精确 PATCH 与 DB completed_at | I2 API 原活动与时间戳断言 |
| 负责人/标签 | 各选两个，再逐一清空，精确数组 PATCH 与未删除关联行断言 | 已有 issue-properties 真实标签按钮序列化和清空回归 |
| 起止日期 | 两个真实 Calendar 选值与分别清空、DB 日期与 null | 已有 inline-refusals range 测试保持一次双 null PATCH；I2 API 保留倒置拒绝与双日期原子更新 |
| 父项 | 原生选择器迁移及页面证明随 T6，A2 仍待这一项闭合 | 不在 T2 允许旧搜索请求 |

label-select 的正 tabIndex 已于 P12 消除，T2 复验当前真实详情无正 tabIndex，不伪造一次删除。日期/标签控件沿既有弹层，模态嵌套的完整证明随 T5/T6。提前添加计划 T16 中的 issue-detail-page 夹具，仅统一真实编号入口登录/观察，不拦截或伪造响应。

行为红 t2-fields-behavior-red 为四项：空白、超长、服务端聚焦、卸载校验；最初语法错误不计行为红。M02 去 FieldError 描述关联、M03 提前聚焦各准确检出空 description，均字节恢复。FieldError size 的类型缺口已修；首次页面失败为测试未计入选项头像的可访问名称，改为唯一可见姓名匹配后通过，产品无额外修改。

最终 t2-fields-final 63/63、t2-i2-complete API/页面 2/2、t2-types-final、t2-lint-clean 0 warnings/0 errors 通过；改动格式和 diff 检查通过。浏览器实际页面载入与编辑器警告仍按既有精确一条断言，T10 才删源头及所有白名单；未增加旧 API/外部请求例外。

### T3 描述确认保存、撤销重做和迁移

已有普通拒绝保留、旧确认、卸载刷新和恢复版本机制保留。新增同一回调轮内连续撤销/重做测试，定位 DescriptionInput 按闭包中的渲染值提前返回：重做回到已确认内容时，会留下刚产生的撤销待保存值。去重移到 useConfirmedAutosave 的当前 owner.value；DescriptionInput 总把变化交给它。没有新增定时器、队列或持久化协议。

首次分立 input 事件的 t3-red 实际全绿，不能计为缺陷红；同一编辑器回调轮的 t3-batched-callback-red 真实检出撤销值残留。恢复旧渲染比较的 M04 再检出错误保存一次；M05 将两个迁移 flag 变为字符串，真实 root 的运行时校验阻止 PATCH，debounce/unmount 两条预期写均未出现而失败，不是编译错误。两变异的纯补丁、输入和字节恢复哈希均归档。

真实 I2 页面用 TipTap 键盘操作，已确认内容在 1.5 秒窗口内撤销再重做后无额外 PATCH；普通连续输入遇当前 503 保留内容，仅显式 Retry 再发同一内容。API/DB 核对 HTML、stripped、合并动态/版本与保存时间。初次加载真实无块 ID 描述时只发 description_html + boolean skip_activity，HTML 保存、审计/版本/主时间戳不变。普通保存不发 description_json 或 skip_activity。既有 href/checked API 故事同时复验其不能借 skip_activity 逃过审计。

t3-restored 33/33，t3-i2-complete 真实页面和三个 API 场景 4/4，types 与改动文件 lint 0/0 通过，格式/diff 通过。首次页面断言遗漏编辑器的现有 paragraph class，已按实际输出修正测试，不计产品缺陷。M1-P2 的撤销重做子项已有处理结果；A3 中弹窗普通工作项/草稿迁移仍归 T4，未提前勾整条。

### T4 创建/编辑字段、迁移与 I1

现有表单以单一描述更新函数区分 load/migration/edit：纯迁移更新描述字段基线，不影响其他 dirty 字段；真实编辑后的迁移保留用户修改。提交元数据保留在本地函数参数，普通工作项只有纯描述迁移才添加 boolean skip_activity；草稿沿原 payload，不扩协议。RHF 关闭内置抢焦点，FieldError 与 useRefusedFocus 处理标题/项目/双日期错误；标题按 Unicode 字符校验。日期字段提取局部组件，既有 dropdown 只增加错误描述透传。移除正 tabIndex、嵌套按钮和 Create more 重复 handler。

Base 的同步 Promise 锁覆盖原生写、确认后消费及关闭。消费回调失败只报告一次，已确认的创建不重发。页面实测另外定位三个原缺口：创建命令未注册设计要求的 c（沿既有命令补 shortcut，保留 ni）；空 state_id 误发无效 UUID（创建边界省略，让服务端默认）；确认提示未展示编号（原链接显示已确认编号）。没有新增监听器、权限 owner、业务缓存或服务端变化。

验证：t4-component-final 41/41；t4-i1-verified 的真实页面/API 2/2，页面核空/长标题、原生 422、描述提及、两位负责人/两个标签、属性/日期、单次 Switch 切换、双击提交及 held 201。请求到达服务端后先核 DB，响应交给页面后核列表与提示 WEB-1；共用 expectIssueCreated/expectInitialDescription 核审计、版本、提及、关注、时间戳，另核纯文本和项目计数。页面日历拒绝选择倒置日期，真实 RHF 字段单测证明倒置输入的反馈/焦点/留窗。普通 issue 与 draft 迁移测试使用真实表单/根 store，编辑器只模拟既有回调；真实 TipTap 迁移来源另由 T3 I2 页面证明，不冒称已有草稿的浏览器测试。

行为红 t4-behavior-red 六项；M06-migration-only 将纯迁移标 dirty 后两项失败，M07 去同步锁后原生 create 被调用两次；额外日期变异检出缺失反馈。每次均字节恢复。最初 M06 同时改变 load，保留历史但以精细变异为主。空状态与提示编号另各有真实失败。夹具 import/hoisting、ModalTitle 真实上下文、data-dirty 与 Propel 属性重名、Dialog 根无自身尺寸、CSS placeholder 定位、请求路由记录顺序、Saving 状态名称和列表 id 定位错误分别修正，不记作产品行为红。首个 HTML 报告路径写在 e2e 内，已逐文件核 hash 后移到统一证据根。

t4-types-final 和 e2e types 通过；改动 lint 0 warnings/0 errors。当前先提交已验证的 T4 产品与记录，随后在该准确提交上串行完整 F；F 和检查点推送尚待回执，因此 Task 4 Step 4/5 与 A3/A4 暂未勾选。关闭决策/嵌套 Escape 留 T5，父项留 T6。

**检查点回执（替代上一段待执行状态）：**产品提交 `4d4463f760cbf10cf9ee52c4688dccb9e255324c` 的 gen-check、Go lint（两段 0 issues）、完整 Go 测试（461.8 秒、无缓存）通过。knip 发现夹具两个未使用导出，`e21a24e64c1f0eedaaeb3e6bbed078760ebceeee` 收为私有；全量 Web 测试发现 T1 编号入口旧夹具缺真实布局，`a9d4f118` 将相同拒绝/只读重试断言迁入真实嵌套路由，聚焦 18/18。后两提交只改测试，产品与服务端/生成物均未变，前面三项证据复用并保留准确 tested SHA，不冒称在 a9d4f118 重跑了 Go。

a9d4f118 的 lint-web 54/54、knip、强制无缓存 test-web 18/18 任务、E2E 205/205（Chromium 203、Firefox A4 2）全部通过；原始回执为 `t4-full-*`、`t4-full-fixed-*`、`t4-full-routed-*`。两次夹具门禁失败记录保留。A3/A4 和 Task 4 完成线已勾，检查点待此文档提交后推送；远端准确 SHA/run 后续写交付回执。后续修改仍须按 Task 和最终 T16 复验，当前不代表 P14 已完成。

### T5 嵌套关闭与焦点

沿用 DraftIssueLayout 管理 dirty 与草稿确认，让它同时拥有外层 ModalCore；同一 requestClose 检查同步写锁、编辑器就绪和 dirty，再决定关闭或显示保存草稿提示。Base 的确认完成走 finishClose，表单按钮和外点/Escape 走 requestClose；保存和发布复用原同步锁。描述编辑器不再监听全 document 的 Escape。ConfirmIssueDiscard 用 ModalTitle 和同步 en/zh-CN 文案，保留既有保存拒绝和输入。未改共享 ModalCore、权限或草稿协议。

真实嵌套组件首先复现外层无改动被一并关闭、有改动时多开保存提示，以及提示无可访问标题。M08 在实际关闭 owner 临时恢复 document Escape，两个嵌套用例均失败，随后逐字节恢复。父项单测只替换尚待 T6 迁移的旧查询服务，Dialog/portal 为真实组件；不把它冒称原生父项页面证明。

验证：t5-restored 六文件 46/46；t5-page-regression 的 C4、I1 页面/API 与 C3 共 9/9 Chromium 通过；t5-types、t5-lint（0 警告/0 错误）和 diff 检查通过。C4 证明日期/标签 portal 的 Escape 留住外层、背景隔离、内层 Tab 首尾循环、内外层外点、dirty 输入保留与焦点回还；没有意外创建请求和落库。编译使用 t5-packages / t5-build 产物。A5 仍留父项真实页面（T6）和三浏览器（T16）。

夹具修正与产品缺陷分记：real timer 前先以 fake timer 完成 FakeNerve 初始化；mock helper 必须先导入，否则先载入的 useDetailScope 指向真实默认空 store；旧断言改为翻译 key。常驻确认框需等退出动画卸载，真实页面需等外层恢复唯一 modal 且目标实际聚焦再发送下一键；首次即时断言失败保留，未以任意 sleep、重试产品写或放宽错误白名单掩盖。自查确认无新增全局协调器、无重复关闭决策。

### T6 原生父项与子项页面

ParentIssuesListModal 直接消费生成 Issue 和已有 useIssueSearch，输入变更立即隐藏旧候选，HTTP 延迟沿原 hook；新建查询只有 project_id/q，已有项带 for=parent 和源 id。选择时复核当前 proof/目录范围；权限 ABA、切项目/源/关闭重开退休旧宿主，Promise 回调成功后才关闭，拒绝和 pending 留住选择器。标题/输入有翻译名称，重试在 Options 外，删除 hover 链接和正 tabIndex。颜色与标识只读目录，不补造字段。

创建 context/provider、默认属性、父项 tag、详情与 M7 创建共用调用同批改为 Issue；表单从当前可见实体同步父项，项目切换/清除不留下旧选择。删除零生产读者的 convertWorkItemDataToSearchResponse 和专用镜像测试，保留状态目录分组的两个性质测试。M7 仅适配共用类型/展示，业务写和自身页面验收不扩；旧 ProjectService 还有 select-duplicate 读者，按 T7 删除。

局部加固：实际选项测试发现 IdentifierText 在禁止复制时仍渲染 disabled button；其只读分支改 span，所有展示消费者自然免于按钮嵌套，复制分支仍归 T15。父项详情链接直接用可见目录生成 browse 地址，缺目录不生成链接。未改后端权限、层级排除或实体协议。

验证：t6-restored-final 十二文件 87/87，包含现有子项读取/确认后 GET 拒绝、命令与全部表单；t6-page-final 的 I8 页面/API、三项 C4 和 I1 页面/API 共 7/7 Chromium 通过。I8 从真实菜单添加两个已有项与一个新项，响应扣留时 DB 先确认而 UI 不提前显示；1/2→1/3 Done、父项排除自身/当前父项/两层后代、移动/解除后的原生链接、计数和逐条审计均核对。C4 父项原生搜索空词查询仅 project_id，内层 Escape 不影响 clean/dirty 外层并回焦点。A2/A6 闭合，A5 三浏览器仍待 T16。

M09 去实际查询 for，精确参数断言红；M10 跳过实际父项命令，三种确认/拒绝测试均因过早关闭而红；均字节恢复。Web/E2E 类型通过，t6-lint-final 0/0，格式/diff 通过。另有夹具修正：FakeNerve 读取 query 而非不存在的 url；审计 helper 的 wrote 是本次写数而非累计；旧命令夹具补目录 hook。一次并发包重建造成 dist 短暂缺失，只计环境失败，改为先构建再测试；不作为行为红。后续包构建与读取这些包的验证必须串行。

T4 检查点远端回执：准确 `503b6778a3ca21ea4c04f85990d0a92be38efee3` 的 [CI 41173311219](https://github.com/open-nerve/NerveProject/actions/runs/41173311219) completed/success，原始公开元数据 t4-ci-latest.json 已保存；不是当前未推 T5/T6 的 CI 证明。

### T7 关联与收集箱查重

收集箱查重沿 Plane 的单选 ID 通知，保持 onSubmit(id):void 和 M7 自身业务写不变；读取改用已有 useIssueSearch、原生 Issue 与项目/状态目录。分诊项不是工作项源，因此查询只含 project_id/q，不伪造 for=relation；仍排除当前源 id。输入、源、权限 ABA 和重开使旧读取失效，提交前复核 proof；当前读取拒绝有 GET 重试、ModalTitle 和输入名称。移除最后的 ProjectService 类及 barrel、两个专用旧搜索类型和精确一条例外；生产零旧读者，76 条规则/23 个例外通过，knip 通过。

关系链路已有原子批量确认、跨项目作用域和确认后 GET 失败只读重试，本次保留并复验；选择器新增部分确认性质，重试只包含未确认项。M11 去真实 widget 的 for=relation，原生参数断言失败；M12 保留已确认选择项，部分确认断言失败。两项均逐字节恢复。

验证：t7-component-final 六文件 59/59；t7-i9-confirmed 页面/API 2/2 Chromium。页面真实添加同项目 Blocking、跨项目 Relates to，持有响应时先核 DB 已写且 UI 未确认；对端显示 Blocked by。另一个账号在 Web 仅访客、Ops 为成员，源项修改明确 403，仍可从 Ops 对端删掉跨项目边；两端动态、关系审计/删除者与原始边一致。没有以“反向显示”代替任一端授权证明。A7 完成。

Web/E2E 类型、变更 lint 0/0、格式/diff 均核验。初次新查重七项因仍发旧请求/未处理拒绝而红；后续测试问题（mock 路由未触发 context、POST 预期应为 201、折叠区需展开、首次搜索应答与切范围的等待、页面账号需完成引导）均修正并保留日志，不算产品缺陷或有效变异。未增加全局缓存/权限 owner、依赖或写协议，T9 完整 F 前不推送。

### T8 链接字段与元数据回读

链接表单改为按工作区/项目/项/link id/权限命名的局部实例，关闭重开退休旧命令；同一同步 sending 锁覆盖 RHF 校验、写入确认和关闭。URL 去两端空白，空白在表单拒绝；其余 scheme/唯一性归一化交给既有原生服务端，不另造客户端 URL 规则。FieldError、useFormRefusal 和 submitCount/useRefusedFocus 统一处理 422 和 409 link_exists，错误可读后才聚焦、不重复 toast；确认后读取失败仍按已成功写处理。补 ModalTitle，待发送 Escape/外点/取消无效。

新增 useLinkMetadataRefresh 只在展开的链接区调度原 link resource。作用域包含 root/宿主/既有权限身份及当前 id+URL 集合；换地址/删除结束原窗口，新链接身份可开始自己的窗口，普通 GET 的新对象或元数据变化不会重置期限。空 metadata 才等待，有字段（含 null/空值）或 error 为终态；每秒至多一 GET、30 次/30 秒内停止，一条在途请求未结束时不重叠。离页/折叠/权限失效取消定时；拒绝保留资源错误并停止自动读取，Retry 只读且打开新窗口，到期保留地址并显示双语重试入口。仍由原资源确认日志抵消 URL A 的晚答和已删除链接，无第二缓存或后端更动。

图标只读 metadata.favicon 的 data:image，favicon_url 仅来源记录，不进入 img.src；无嵌入图标/blocked/failed 使用原链接图标。初始行为红为表单/图标六项、无回读五项；M13 临时恢复外部源图标时两项红，M14 临时放宽回读预算到 300 次/300 秒时第 31 次请求检出，均字节恢复。t8-restored 六文件 48/48，包含确认后 GET 拒绝只 GET 重试、真实资源 URL 变更/删除与迟到读取。

t8-i10-confirmed 的页面/API 3/3 Chromium 通过：真实输入空白/原生 422/409 的 recordFocuses、持有创建响应前 DB 先确认且不能关闭/重复写、初始地址与真实 worker 后到 title/data icon、标题编辑零新 job、URL 编辑新 job、删除与全部动态/审计一致；无外部浏览器请求/CSP/旧 API。Web/E2E 类型和改动 lint 0/0 通过。测试修正记录保留：Problem fixture 补必需 message；PageWatch 用 requestUrls；route.fetch 持有先于旧 body recorder 时计数改用页面 request 事件；同址动态链接需限定链接区，Links 本来默认展开，写后 DOM id 会重建，定位采用当前标题的相对区域而不保存旧 id。均不是产品红。A8 的复制验证仍待 T15，三浏览器随 T16。

### T9 子树删除确认

双语确认明确删除全部子项及归档后代，按钮沿既有翻译。没有查询隐藏子树数量。保留必需 Promise 回调、权限判断和 followInSession；同步 sending 同时保护关闭事件，确认成功的 finishClose 单独执行，拒绝留窗可重试。组件先检出中英文说明缺口与同一事件周期关闭漏洞；M15 将关闭移到服务端确认前，三项断言失败且字节恢复。

I17 页面进一步复现删除后停在编号页：canonical 实体删除使 work-item-header 丢掉项目 id、卸载动作 owner，当前成功因此被当成旧宿主。局部根因修复将路由已证明的 projectId 传到页头，不再靠被删实体维持动作生命周期；useDetailScope 和共同路由 key 不变。真实编号路由单测先红，修复后证实当前删除返回列表，切到新编号后旧答仍无导航。归档和列表四组旧测试同步翻译键，不改生产权限、资源 owner 或服务端协议。

验证：t9-route-green 六文件 82/82（含原有归档/列表确认），t9-component-final 22/22；t9-i17-route 页面/API 2/2 Chromium。页面由另一作者创建两层后代、归档项和六种协作记录，并有跨项目双向关系；持有真实 DELETE 响应时先核目录中全部表的同一删除时刻/当前写者、禁止关闭/重复发送，兑现后返回列表；已删除分支和无关项逐行不变，协作创建者仍为原作者。复用 expectIssueDeleted/catalog 与 collaborationRows，无复制 SQL 目录。

类型、实际 11 个改动 TS 文件 oxlint 0/0、格式/diff 通过。首次翻译键错层被真实页面检出，单测翻译读取改为真实嵌套路径后修复；DB 确认前需等待提交、ModalCore 根无自身尺寸需断言可见 heading，两项夹具问题单独留痕。先提交本次产品，再在准确提交串行完整 F；T9 的完整门禁和推送步骤尚待回执，不将当前聚焦通过冒称全量通过。

**检查点回执（替代上一段待执行状态）：**`8a35d029` 的 gen-check、Go lint、无缓存完整 Go 测试（466.9 秒）均通过；knip 发现 T8 表单类型仅本文件读取，`8c9c7e0eb5c3bbd3b7e54a4b88ce9e16edf1702f` 收回该无用导出。服务端/生成输入无变化，前三项原始 tested SHA 保留。后者 lint-web 54/54、knip、强制无缓存 test-web 18/18 任务与 E2E 212/212（Chromium 210、Firefox A4 2）全通过。原始日志及 SHA/hash 绑定见 t9-gates-receipt.json；默认 HTML/test-results 已逐文件核 hash 复制到统一证据根。Task 9 完成线已勾，本次文档提交后推送检查点，后置远端回执另补；不代表 T10–T16 已通过。

T9 后置远端回执：提交 `0e8c912a3efd345ee4c39906aa7ea7cb0ef10cee` 的 [CI 41178590419](https://github.com/open-nerve/NerveProject/actions/runs/41178590419) server/web/e2e 均成功；原始 run/jobs 在统一证据根 t9-ci-latest.json、t9-ci-jobs.json。

### T10 编辑器加载与警告根因

富文本/轻量入口保留公开 props/ref，Suspense 内按挂载加载实现；工具栏 barrel 的第三条 eager 链无外部读取者，撤回该导出。个人设置仍有命令面板和快捷键，但无 workspace 时不挂工作项弹窗。6 项 lazy 测试覆盖等待期间不 ready、参数更新、ref/内容回调、卸载与错误进入既有边界；真实构建按代码标记识别并附 hash，两个匿名入口都不取编辑器 chunk，登录后读取、输入和数据库保存正常。未通过时没有空内容误保存。

初次删除自定义 supportMap 后，真实 C10 仍红：上游 extension-emoji 的数据导入会创建默认 Node，实际执行另一份 Canvas 检测。沿 M3 已锁定的 emojibase-data 15.3.2 直接消费纯数据，逐条比对 1,933 条名称/字符/别名/标签/文本表情完全一致（t10-emoji-data-parity.json），删除无用扩展和检测依赖、Canvas 测试垫片及 external 配置。无新依赖品种/版本，无 node_modules 修改或构建拦截。设计、spec、plan 同步记录这一局部根因处置。

全 E2E 的 EMOJI 警告常量/豁免/注释和 helper 无用参数已清零；watchPage/expectQuietConsole 保持严格。实际键盘测试另检出强制打开表情菜单后 Escape 使用空 editor 的异常，改为当前 editor owner 并安全释放定位监听；真实命令红/绿保存在 t10-emoji-command-red、t10-emoji-fixed，错误命令探针不计产品红。

M16 恢复设置页宿主使 2 项组件测试失败；M17 重新引入未使用的上游扩展使真实构建页面精确因 Canvas 警告失败，均有 patch、字节恢复 hash 和恢复绿。最终 web 5 文件 37/37、editor 编辑/安全往返 15/15、C10/S2/A7/A9 Chromium 20/20；类型、knip 通过，实际 67 个 TS 改动文件 oxlint 0/0，editor 上限按真实净减从 59 降至 54。最后仅将建议列表的类型提取恢复为原有边界，不改运行逻辑；补类型/lint 复核。全量 F 在 T13 检查点执行，A10 三浏览器归 T16。

### T11 原生标注与死序列化

标注默认及存储 fallback 只保留原生 Unicode 128161，不补 CDN 地址；既有显式 URL、原生表情、图标的解析与安全边界保留。五处 custom-color/custom-image/emoji/mentions/callout markdown.serialize 在生产无 getMarkdown 读者且 transformCopiedText=false，已删除；一并删除 callout 专用转义 helper、无读者 MarkdownStorage 类型和测试投影。提及文字仍供 renderText，图片存储仍供真实图片消费者，tiptap-markdown 的粘贴解析及 ref 的 HTML→Markdown 保持。

P4 修复的历史输入未撤销：共享 fixture 14 组仍在，input/sanitized/恶意属性/保留断言逐字段核一致，仅删 5 组死 Markdown 投影、改 3 组由默认值产生的 HTML URL。证据 t11-fixture-preservation.json；Go content 测试通过。35 项真实编辑器/安全/存储测试验证插入、旧内容读取、HTML 往返和 Markdown 粘贴。M18 恢复 CDN 默认检出 13 项失败，字节恢复后 35/35 绿。

实际构建 t11-fixed-build、C10 Chromium 4/4：两个匿名入口延迟加载、个人设置边界、原生标注创建/保存/刷新；外链、worker、CSP、pageerror、console 均严格为零。t11-build-worker-audit.json 记录 411 个 JS 产物及 hash；无 Worker/SharedWorker 构造，blob 字符串只在路由协议名单，createObjectURL 只在已有 CSV 下载，编辑器 chunk 无这两项，不扩 CSP。

改动文件 oxlint 0/0，包实际净减 6 后上限 54→48，knip 通过。构建检出公开 onChange 类型对可选对象做解构，改为命名的可选 options，字段/可选性不变；运行回调不变，复构建和页面通过。T11 完成，三浏览器统一在 T16 收尾。

T11 类型补验：页面数据库轮询按查询 API 的可空行类型读取（t11-e2e-types-fixed 通过）；轮询仍要求真实内容出现，未放宽数据库断言。

### T12 M4 图片边界

详情、peek 明确传 image 禁用；创建/编辑/草稿共用描述入口同样禁用。共享 DescriptionInput 的默认行为不变，收集箱仍有真实上传读者；没有按路径或角色偷偷改全局 editor 默认。M4 弹窗专属的上传/复制 service 调用、onAssetUpload 传递、uploadedAssetIds 和确认创建后的旧附件关联全部无读者，已删除。共享可编辑 wrapper 要求的两回调使用明确拒绝函数（Promise<never>），防止未来误接时伪造成功；正常用户入口不可触发它。

实际 owner 测试 6/6，覆盖五个 M4 宿主和共享默认；与原生命周期/表单合跑 46/46。真实 Editor 的 schema 无 image/imageComponent、slash 无图片、Nerve HTML 粘贴只保留文字且上传 spy 零调用；连同既有图片可用默认、Markdown 和编辑回归 22/22。最初详情/peek 夹具缺少成员上下文的两项错误不计产品红；三个表单入口原始缺少禁用配置是真实红。图片文件剪贴板优先按文件处理，首次页面把同一剪贴板中文本也当成文档粘贴属夹具假设；调整为分别执行文件粘贴、文件拖入、文档 HTML 粘贴和键盘输入，未修改产品粘贴规则。

C12 两页均验证实际构建：无图片菜单/附件/关注入口，图片粘贴与拖入不请求上传，HTML 内图片被 schema 排除，文字及后续输入在真实数据库保存。t12-restored-pages Chromium 2/2；同构建 C10/C4 其余 7 项先前已通过。M19 打开实际详情的 image 配置，真实页面因图片菜单仍存在而失败；字节恢复、重构建、页面恢复绿。类型、knip、实际改动文件 oxlint 0/0 通过。M5 开启图片前需接通文件服务、鉴权/身份清洗、上传后归属；本期不删除服务端图片 HTML 清洗或其他模块的真实文件读者。

### T13 子项计数地址

表格由 P12 完成，本次八项消费者回归原始全绿，不记新产品修复。行内计数的原始四项失败分别证明普通项残留片段、归档项走旧中转地址、缺少键盘链接、目录未就绪拼出 undefined 编号。改用原生 Link，沿现有可见目录 identifier/sequence_id 直达 browse；目录缺失或非法编号只呈现计数。复用已有双语子项文字作可访问名称，隔离 click/focus/Enter/Space 的行事件但不阻止浏览器默认链接行为，没有新增详情请求或导航服务。

t13-restored 两文件 16/16；t13-pages 的 I8 页面/API 4/4 Chromium。实际列表及表格分别鼠标点击、键盘 Enter 进入准确父项，核标题及两个子项，Back 回原列表且无额外 peek。M20 在真实生产 Link 恢复片段，普通/归档两项真实点击断言失败，精确字节恢复后 16/16。Web/E2E 类型、四个改动文件 oxlint 0/0、格式/diff 通过；新页面串行操作采用既有 reduce 模式，未豁免 lint。A12 功能与 M1-P4 子项已记录，handoff 顶层仍 open。完整 F 将在本次产品提交后执行，通过前不推检查点。

首次完整 F 的 c3489a1a 通过生成检查、Go lint、无缓存 Go 测试（464.877 秒）、Web lint 54/54 和 knip。test-web 检出三个既有夹具的整模块 react-router mock 没有 Link 导出，11 项在 JSX 构造时报错；未发生业务断言失败。改为保留模块真实导出并仅覆盖原有 hooks，保留日期原子写、拒绝/ABA、菜单断言；聚焦五文件 42/42、三个文件 lint 0/0、格式/diff 通过。后续仅改测试与记录，server/api/生成输入不变；前三项沿原 tested SHA，Web 四项从修正提交续跑，原失败证据保留。

0c374b13 的后四项续跑中，Web lint 54/54、knip、无缓存 test-web 18/18 任务通过；E2E 为 219 通过、1 失败。失败是既有 C2 peek 内 Display 面板偏离按钮 47.9375px；原产物 132 文件逐 hash 保存到 t13-c2-failure-*，原样单 worker 重跑十次全部相同性质失败，不能归为 W11 或偶发。

诊断 t13-c2-diagnostic 持有实际 ProseMirror chunk：加载前按钮 bottom=312.9375、面板 y≈313；释放后按钮 bottom=360.9375、面板 y=313，持续 30 帧；单独触发 resize 后面板 y=361。结合锁定 Popper 2.11.8 的 eventListeners 源码，根因是内容加载造成位置变化不触发其 scroll/resize 更新。新真实组件测试的两种按钮在没有 scroll/resize/重新渲染时应跟随下移，原实现均失败；新增 C10 持有真实 chunk、等待布局更新的页面同样失败 47.9375px。定位与生命周期修复及完整门禁的最终结果待下续，不改原 C2 断言或用等待编辑器完成规避用户交互。

局部修复在当前 Popper 实例生命周期中逐帧比较按钮矩形，仅变化时 update，实例销毁即取消；公开 props 和所有领域调用方不变。首轮组件 32/32、新 C10 通过，但两个原 C2 仍失败。t13-c2-timing 的逐帧数据证明坐标已经更新，宽泛 transition 却把定位 transform 也缓动了 200ms，开启新的过渡；因此将进出动效限定为 opacity/translate，Popper transform 直接定位。构建 CSS 证实 Tailwind 的 translate-y 使用独立 translate 属性。原 C2 文件及几何 helper 均未修改，t13-anchor-motion-pages 的 16/16（C2/C9/C10/C12）通过；组件覆盖两按钮、关闭/重开/卸载及项目成员/个人主页消费者 32/32，类型与改动文件 lint 0/0。完整 F 后四项将从此修复提交重新执行；T16 加 C2 三浏览器，不能以 Chromium 代替。

**检查点回执：**修复提交 f2448dfaf7c57e77de736f5636810bd6a5b209ed 的 lint-web 54/54、knip、强制无缓存 test-web 18/18 任务及 E2E 221/221（Chromium 219、Firefox A4 2）全通过。前三项保持 c3489a1a 的实际 tested SHA；server、api、生成客户端和锁文件从该点至本提交逐路径 diff 为空。七项分层回执与 103 个 E2E 原件的复制 hash 在 t13-full-anchor-receipt.json，不把文档更新冒称重跑。T13 完成线已勾，本次文档检查点随即推送；准确远端 CI 后置记录，T14–T16 尚未完成。

检查点文档 8b1fa70e623bc9e75eda4e33b0acd7a2325c120d 已经系统 git 自动推送，准确 [CI 41184198943](https://github.com/open-nerve/NerveProject/actions/runs/41184198943) 已启动；T13 task-done 23/23。远端终态待后置核准。

### T14 访客、会话与 Back

本任务新增验证，没有改变产品权限、会话或宿主逻辑。五个组件文件含编号入口/共同身份、详情宿主、命令宿主、peek 和删除确认；新增权限 ABA、路由 ABA、旧会话成功/拒绝，连同已有同 id 重开与迟到答共 60/60。M21 去掉真实 followInSession 的会话核查，两个真实删除确认消费者测试分别因旧成功/拒绝跟进而红；M22 去掉命令宿主关闭保护，三个旧回调误关新窗测试红。两个生产文件逐字节恢复，t14-restored 60/60，不把假 root 的退役当作消费者核查。

I20 两页通过实际 API 先创建再降级，普通访客与管理员兼访客都验证 OwnOnly 列表仅自己的行、隐藏项与不存在项 404 的体/Content-Type/页面提示逐值相同；开放全部后按返回权限展示编辑或只读，数据库与零业务写断言保持。最初管理员夹具先邀请为管理员却按项目成员添加，被既有规则正确拒绝 422；参照现有 adminGuest 的 API 次序，改为先项目访客、再提升工作区角色，未修改产品规则，不计产品缺陷红。

S2 三页明确公开项目的成员 200、访客隐藏项 404、项目非成员 403，只精确放行对应拒绝。I22 真实 POST 先经 route.fetch 到 nerve 并由 DB 确认旧写者，再由第二标签页实际退出/登录另一账号，第一标签页切换工作区并打开新表单后放回旧 201；输入、地址、新账号可见数据保持，无旧成功提示，只有一次 POST。另页 Back 覆盖跨工作区、项目、工作项的创建/删除/链接/peek，Forward 不恢复旧弹层，零业务写。t14-pages-fixed 7/7 Chromium，类型、五文件 lint 0/0、格式/diff 通过；A1/A13 功能已勾，显式三浏览器与最终完整 F 仍归 T16。
