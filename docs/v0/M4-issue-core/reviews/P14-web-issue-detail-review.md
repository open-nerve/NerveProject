# M4/P14 详情与弹窗：实施、整审与交付记录

日期：2026-10-31。状态：**原分支已授权合入 main，集成 CI 的测试前置修复及最终 main 核准中（§15）**。基线 `8bb4bbcb52a224ace4cb2f9d383bca9b97a022bf`；[spec](../specs/P14-web-issue-detail.md)、[plan](../plans/P14-web-issue-detail.md)、[文档预检](../../../reviews/re-20-M4P14预检审核.md)。§1–7（含“未实施”表）仅保留预检历史；§8 是逐任务历史，实施验收见 §9–14，后续集成事实以 §15 起为准。phase 已进入 M4/P14。

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

检查点文档 8b1fa70e623bc9e75eda4e33b0acd7a2325c120d 已经系统 git 自动推送；T13 task-done 23/23。准确 [CI 41184198943](https://github.com/open-nerve/NerveProject/actions/runs/41184198943) attempt 1 的 server 通过、web 单测失败、e2e 未执行。已直接从本机登录浏览器读取：261 文件/2799 断言通过，但 close-policy.test.tsx 的字体检测定时器在 jsdom 销毁后访问 document，留下一个未捕获异常。原始公开元数据与 UI 日志转录 t13-ci-browser-failure.md 保留，T16 修正并重新验证；不能将本地绿或断言全过代替此失败。

### T14 访客、会话与 Back

本任务新增验证，没有改变产品权限、会话或宿主逻辑。五个组件文件含编号入口/共同身份、详情宿主、命令宿主、peek 和删除确认；新增权限 ABA、路由 ABA、旧会话成功/拒绝，连同已有同 id 重开与迟到答共 60/60。M21 去掉真实 followInSession 的会话核查，两个真实删除确认消费者测试分别因旧成功/拒绝跟进而红；M22 去掉命令宿主关闭保护，三个旧回调误关新窗测试红。两个生产文件逐字节恢复，t14-restored 60/60，不把假 root 的退役当作消费者核查。

I20 两页通过实际 API 先创建再降级，普通访客与管理员兼访客都验证 OwnOnly 列表仅自己的行、隐藏项与不存在项 404 的体/Content-Type/页面提示逐值相同；开放全部后按返回权限展示编辑或只读，数据库与零业务写断言保持。最初管理员夹具先邀请为管理员却按项目成员添加，被既有规则正确拒绝 422；参照现有 adminGuest 的 API 次序，改为先项目访客、再提升工作区角色，未修改产品规则，不计产品缺陷红。

S2 三页明确公开项目的成员 200、访客隐藏项 404、项目非成员 403，只精确放行对应拒绝。I22 真实 POST 先经 route.fetch 到 nerve 并由 DB 确认旧写者，再由第二标签页实际退出/登录另一账号，第一标签页切换工作区并打开新表单后放回旧 201；输入、地址、新账号可见数据保持，无旧成功提示，只有一次 POST。另页 Back 覆盖跨工作区、项目、工作项的创建/删除/链接/peek，Forward 不恢复旧弹层，零业务写。t14-pages-fixed 7/7 Chromium，类型、五文件 lint 0/0、格式/diff 通过；A1/A13 功能已勾，显式三浏览器与最终完整 F 仍归 T16。

### T15 六处复制反馈

编号、详情页头、链接行、关联 helper、子项 helper、peek 页头统一复用既有 useCopyText/useCopyLink；hook 等到浏览器实际完成才报告成功，拒绝给本地化错误。关联和子项 copyLink 类型及消费者改为 Promise<void>，没有新增复制协议。编号只复制文本；peek（包括已归档项）复制当前 origin 的原生编号地址，移除只用于旧归档路由的 isArchived 参数及零读者类型导出。链接/关系/子项的交互按钮带行名字，键盘打开菜单和执行复制不会触发行跳转。

detail-copy 的七个真实消费者场景（六入口加归档 peek）各持有、成功或拒绝 clipboard Promise，调用真实组件及 hook。初次 t15-copy-red 为 11 项行为失败与 5 项未处理拒绝；修复后聚焦 19/19、受影响七文件 68/68。M23 暂时去掉共用 hook 的 await，实际链接消费者的等待与拒绝断言两项变红，另捕获一项未处理拒绝；字节恢复后 68/68。测试不以 mock toast 调用替代真实消费者。新 C4 六页从 HTTP 服务加载，受控浏览器拒绝，键盘操作后一次错误、零成功、零业务写、地址不变，连同原三项 C4 共 9/9。首次子项页的定位误用了关系菜单文案，改为真实“Copy work item link”，没有修改产品来迁就测试。

定点删除六个 copy-hook 例外，关键词守卫 76 规则/17 例外零命中；类型、翻译键、12 个实际 TS 文件 oxlint 0/0 与格式/diff 通过。剩余四处仍归 M4/P15：`issues/create-issue-toast-action-items.tsx`、`issues/issue-detail/issue-activity/helper.tsx`、`issues/issue-layouts/quick-action-dropdowns/helper.tsx`、`power-k/ui/pages/context-based/work-item/commands.ts`。M2 §11 的历史六处与本段六处不完全同组，时间线和布局 helper 尚未关闭；M2/M3 顶层交接保持 open。三浏览器和 T13 CI 字体夹具修正留 T16。

### T16 收尾验证与失败诊断历史

T13 CI 的未捕获异常来自本段 fake-form 夹具保持真实项目下拉框时，也挂载了装饰 Logo 的浏览器字体检测。该第三方 hook 在 jsdom 使用 Node 定时器，卸载不取消加载；环境销毁后仍访问 document。临时诊断在真实表单打开时跟踪该库建立的定时器，原夹具确实启动外部字体轮询（t16-font-red），仅隔离装饰 Logo 后为零（t16-font-green）。两个探针均运行后字节恢复，不提交诊断断言，也不计入产品变异。项目选择器、表单、ModalCore 与关闭策略保持真实，浏览器 I1/C4 保留真实 Logo；不改第三方库、不吞异常或增加等待。六个表单文件 46/46、实际分支 166 个 TS/JS 文件 oxlint 0/0；完整门禁与新 CI 尚待运行。

显式三浏览器配置已列出每浏览器 40 项、总 120 项（13 文件），包括受 T13 定位变化影响的 C2；这是 --list 盘点，不是执行通过数。C2 内含四项既有兼容夹具，与其余真实业务页面明确区分。完整 F 后串行执行三个 project，各自保留 JSON/HTML/raw 和实际 skip。

首轮完整 F 在 9a0f32e4 的生成检查、Go lint、无缓存 Go 测试（466.758s）及 Web lint 54/54 通过后，knip 发现新矩阵配置没有命令入口。按已有 test:m4-filters 模式注册 test:m4-detail，t16-knip-entry 通过；没有新增忽略或放宽守卫。

整审发现并由主控与独立审查者共同确认一项 Major 局部问题：ready 的描述数据与尚未加载完成的编辑器实现是两种状态，requestClose 把空 ref 误当忙碌，阻止 clean 取消和 dirty 草稿确认。四个真实 C10 页面持有实际 chunk，覆盖 clean/dirty × Escape/Discard，原实现全部行为失败。只修正当前局部关闭条件，提交保护不改，已有实例 readiness=false 仍受保护；46 个表单组件测试及四页通过，类型、lint、格式/diff 通过。首轮修复后两页的失败为测试对零尺寸 Dialog 根节点判断可见，改验其真实标题后通过；没有改产品几何来迁就测试。新四页会进入三个浏览器各 44 项的最终矩阵，完整 F 在修复提交重新执行。

09525b31 的完整七项门禁全部通过：无缓存 Go 458.692 秒、Web lint 54/54、knip、无缓存 Web test 18/18 任务、完整 E2E 238/238；103 个报告原件逐 hash 保存于 t16-final-receipt.json。主控随后检查修复增量，补出同一 readiness 边界：允许进入草稿确认后，Save to Drafts 也必须等待编辑器就绪。真实 pending chunk 页面原先发出了草稿 POST 并得到 400，而没有本地就绪提示；在 handleCreateDraftIssue 加与表单相同的检查，取消仍可用。t16-lazy-draft-behavior-red 和 t16-fix-pass-pages 4/4 证明闭环，77 项受影响 Web 测试及 36 项编辑器回归通过。首次错误按钮大小写、以及对被内层 aria-hidden 的外层字段使用默认 role 定位，分别是测试错误；改用实际文案、先 Cancel 回到外层核内容再 Discard，没有改变产品焦点隔离。

按 M1 原始 deadsym 脚本盘点改动范围 118 个候选/28 文件，并逐项查真实读者。确无调用者的 DeleteIssueModal.dataId 回退、InboxIssueProperties.isVisible 恒 false 分支及其专属状态/父项 UI、测试类型里未读的 resanitized 声明已删除；五个仅文件内消费的类型收回 export。共享 JSON 中 resanitized 仍由 Go 清洗测试读取，保留；公开 EditorRefApi 按 spec §2.9 保持兼容，不能将无当前生产调用的合同方法假称已有读者。候选逐项裁定和最终复跑数随整审收据记录。T6 对 M7 父项先作类型适配的历史不改写，本次确认无入口后删除死分支，不扩大 M7 业务。

**三浏览器首轮与夹具诊断：**beda23c1 的 Web 后四项完整门禁全部通过，前面三项输入未变，分层回执 t16-closed-receipt.json；显式矩阵首轮 Chromium 44/44、Firefox 37/44、WebKit 34/44，失败原件全部保留，不能宣称三浏览器已经通过。T16 逐项定位后仅修复测试准备，未改产品行为或扩展警告白名单：

- Firefox 原生插入会丢 NUL；trace 中原请求实际是 `Invalidtitle`，服务端正确接受。用 DOM 原生 setter/input 注入准确非法内容并先核字段值，随后真实提交、422、聚焦、数据不变断言保持；三处共用 browser helper。浏览器最小探针 t16-engine-probe 同时记录实际三引擎版本。
- Firefox 的 synthetic ClipboardEvent 构造未携带 DataTransfer。三引擎统一给合成事件明确的真实剪贴板对象，继续运行真实 editor/schema/存储；区分主 .tiptap 与 ProseMirror 的临时隐藏粘贴目标，并等待前一次粘贴的真实焦点恢复后发送下一次输入。C12 详情直接登录该入口，避免原夹具刚开始列表读取就整页导航导致合法取消请求。
- Mac WebKit 默认用 Option-Tab 遍历按钮，沿已有 C2 的平台键位验证内层 Tab 环，不改产品焦点逻辑；对应 [Apple 键盘说明](https://help.apple.com/safari/mac/8.0/en.lproj/cpsh003.html)。WebKit 和 Chromium 对既有明确 422/409/404/403/503 的系统 console 消息逐项匹配，应用错误与全部 warning 仍严格为零。
- I10 的测试 PNG IDAT CRC 错误，Firefox 正确报损坏；t16-favicon-crc.json 校验并仅修 checksum，C12 同样使用有效图片输入。真实 worker 元数据、data favicon、零外连断言不变。
- 关系面板尚在裁剪展开时，测试直接 focus/click 隐藏行使 Firefox 自动滚动裁剪容器，触发 Popper 更新警告并可能丢失菜单。逐帧 probe 三轮记录了高度、scrollTop 和焦点；在现有 transitionsEnded 条件边界等行实际展开后操作。I9、关系复制、I10 三场景各重复三次全部通过，t16-visible-relations-firefox 9/9；没有延时 sleep、force click 或警告豁免。
- WebKit 的 I2 预加载警告由测试在初始载入前安装 page.route 引起：相同实际应用无 route 时单次加载且零警告，有 route 时同脚本被再次 preload，已执行的模块被报告未消费。t16-app-preload-probe-scoped / body-probe 对照保存 Network/Performance 和真实 migration body；独立小 HTML 探针未复现的记录也保留。I2 改用 request 事件观察现有普通 JSON body，仅在编辑器初始化完成后安装下一次 503 拒绝；迁移、撤销重做、重试、全部请求体及审计断言不变。WebKit 同场景连续三次通过（t16-description-webkit 3/3）。

后续 `d2ef5d0b` 的完整 Web 门禁前三项通过，默认 E2E 236/238：I10 API 的旧 PNG CRC 期望未同步；I2 Chromium 的被动 request 事件没有提供 body。两者均为测试准备错误，失败原件 141 文件已按 hash 保留。相同应用的 Chromium 双场景对照 `t16-app-body-chromium-probe` 证明：无拦截 body=null，有精确 URL fallback 拦截可得完整真实 JSON，均无预加载警告。因此 I2 只在 Chromium 开启该透传，WebKit/Firefox 保持观察；其后 I2 三引擎各 1/1 通过（`t16-body-and-favicon-*` 的标签虽含 favicon，矩阵 grep 交集实际只执行 I2，不能宣称 I10 同时通过）。I10 正确字节的 API 断言留下一轮完整 E2E 核验。没有删除请求体断言、增加重试或放宽警告。

上述定向复验完成，最终完整 F/三浏览器与最终分支 CI 仍待本次准确输入运行，不以诊断探针的通过代替阶段门禁。

## 9. 最终验收映射

最终验收输入 `b7d1f0c2daf2f6adf719e6d58744067011d58de8`。以下是产品完成事实；最终文稿的远端提交和 CI 单列 §14，不能用旧 CI 替代。

| 条款 | 最终证据 |
|---|---|
| A1 | T1 共同路由身份、T14 owner/session；矩阵 15–16、36–37，I22 实际旧 POST 先落库再跨标签换账号 |
| A2 | T2 字段拒绝/聚焦与七类属性，T6 父项；矩阵 33、40 |
| A3 | T3 同轮 undo→redo、boolean migration；T4 issue/draft dirty；矩阵 32 的实际 PATCH/审计精度 |
| A4 | T4 创建/编辑、默认状态/c 命令、FieldError、busy；矩阵 7、29 |
| A5 | T5/T6 内层 Escape/Tab/portal；整审慢加载取消和 Save to Drafts readiness；矩阵 7–9、11–14 |
| A6 | T6 原生父项所有读者、后代排除/移动/解除/部分确认；矩阵 8–9、40，M09/M10 |
| A7 | T7 原生关系/查重/反向/对端权限；矩阵 41，M11/M12；M7 自身业务仍未验收 |
| A8 | T8 URL、真实 worker 元数据/data favicon、有界 GET/晚答；矩阵 3、30，M13/M14 |
| A9 | T9 子树/必需 onSubmit、拒绝和删除后的路由跟进；矩阵 31，M15 |
| A10 | T10 lazy rich/lite/toolbar、设置边界、删除 Canvas/警告豁免；矩阵 10–17，M16/M17 |
| A11 | T11 原生 callout、五个死 serializer、14 组安全 fixture 保留、worker/CSP；矩阵 18，M18 |
| A12 | T12 M4 image/upload/附件/关注边界与 T13 原生计数地址；矩阵 19–20、38–39，M19/M20 |
| A13 | T14 OwnOnly/admin兼guest/公开项目非成员、真实换会话与 Back；矩阵 34–37、42–44，M21/M22 |
| A14 | T15 六处真实复制消费者等待/成功/拒绝、原生地址/键盘，六例外删除；矩阵 1–6，M23 |
| A15 | §10 七项分层门禁、三浏览器各 44/44；§11 有效变异 23 项、逐项 dead 与改动文件 lint |
| A16 | §12 整分支审查及作者增量复审、§13 交接边界、§14 最终文稿准确 CI 后置门；未合并 |

完整 test title/文件行号见 [P14-browser-matrix.md](P14-browser-matrix.md)。它覆盖 40 个业务页面和 4 个兼容夹具，不把夹具冒称业务页、不把默认 Firefox A4 两例冒称 P14 Firefox 验收。

## 10. 最终本地门禁与浏览器

实施原件统一在 worktree `.superpowers/sdd/P14-web-issue-detail/`；`run.py` 为每条命令保存 cwd、完整 argv、时间、HEAD、dirty diff/hash、退出码和 raw log hash。没有未提交产品输入混入最终门禁。

| 门禁 | 准确输入及结果 |
|---|---|
| gen-check / lint-go / Go test | `09525b31` 的 `t16-final-*`，三项退出 0；Go `GOFLAGS=-p=1 make test` 无缓存，458.692 秒 |
| Web lint | `b7d1f0c2` 的 `t16-ready-final-lint-web`，54/54；关键词 76 条、17 例外、零命中，1284 翻译读取均有定义 |
| knip | 同提交 `t16-ready-final-knip`，退出 0，无新忽略 |
| Web test | 同提交 `TURBO_FORCE=true make test-web`，18/18 任务、零缓存命中 |
| build + 默认 E2E | 同提交 `make e2e`，238/238（Chromium 236、既有 Firefox A4 2），68.971 秒；默认报告 103 文件逐 hash 留存 |
| 显式 P14 矩阵 | 同提交 `t16-ready-matrix-{chromium,firefox,webkit}`，各 44/44，三引擎均零 skipped/unexpected/flaky |

`t16-ready-final-receipt.json` 集合七项真实回执；`t16-ready-final-server-input-reuse.json` 证明 `09525b31..b7d1f0c2` 的 server、api、生成客户端、锁文件、Makefile、CI 和 tools 输入全无差异，因此前三项复用原始测试，不假称在最终 Web 提交重跑 Go。最后的文档提交只改 docs，仍须通过 §14 的远端准确最终 SHA CI。

三引擎实际版本为 Chromium 153.0.8010.12、Firefox 155.0、WebKit 26.6，Playwright 1.63.0；矩阵报告与产物逐 hash 已封存。历史所有红灯仍保留，包括首轮矩阵、PNG/JSON 采集夹具问题及诊断失败，不以“最终绿”覆盖历史。

## 11. 变异、dead 与守卫

`t16-mutation-audit.json` 逐项收 M01–M23 的真实生产 diff、行为 RED、精确字节恢复与恢复 GREEN；23 种有效性质，超过本段至少 16 的门槛。M06 只计准确 migration-only 的红；M05 被真实 root 运行时拒绝、预期 PATCH 缺失是行为红。字体计时器探针、测试按钮/定位/数据准备错误、空任务调用都不计变异。

按 M1 原 deadsym 脚本和锁定 TS 5.8.3，初始改动候选 118/28 文件，最终 115/26 文件。删除 `dataId` 回退、恒 false `isVisible` 分支、TS 的 `resanitized` 死声明，以及五个仅文件内类型的 export。JSON/Go 的真实二次清洗字段保留。`t16-dead-final-audit.json` 对 115 项逐条写真实 JSX/测试/动态键/框架读取或明确公共 ref 合同；公开 ref 13 项不是假装已有生产读者，遵守 spec 的兼容要求，P16 继续统一盘点。

最终变更输入共 169 个 TS/TSX 文件。默认 root 配置实际 lint 167 个（零错误/警告）；两个 config.ts 属既有 ignorePatterns，另用规则完全相同、仅不排除路径的证据配置显式 lint 两个，仍零错误/警告。原配置与守卫没有修改；`t16-ready-final-changed-lint.json`、`t16-ready-final-config-lint.json` 记录实际执行数，不能把空扫描视为通过。全局警告上限只沿任务真实净减更新。

## 12. deep-review 与修复闭环

[正式整审 re-21](../../../reviews/re-21-M4P14整体审核.md)。唯一 fresh-context 审者对 `8bb4bbcb..9a0f32e4` 读取全部 195 个净变更文件（53,706 行）、129 个去重上下文文件，72 个业务文件按调用链分组，顺序执行逻辑、集成和覆盖三个视角。原报告与阅读清单：`final-review-independent.md`、`final-review-read-inventory.json`。

确认的 Major 是 lazy import pending 时创建窗把空 editor ref 当作忙碌，阻塞 clean/dirty 的 Escape/Discard。四个真实 chunk 持有场景原实现 4/4 行为红，修复区分“未挂载”和“已挂载且忙碌”，其后发现新可达的 Save to Drafts 也须保留 readiness，已一并闭合；普通提交与 busy 保护未放松。真实页面及现有 import reject/error-boundary 单测均通过。Minor 的死分支和仅本地类型已清完，没有未处理的本段 Major/Minor。

独立首轮之后的 `9a0f32e4..e0ee0ff7` 共 22 个文件由作者复审，补读新增的 link-page/package 完整文件，并对保留控制流、调用者、公共合同及全部修改正文复核；`t16-author-increment-review.json` 和 patch 绑定输入。其后 CI 发现的 I22 前置问题另由作者完整读改动文件并复核原 host/权限/路由调用链，记录为 `t16-ci-i22-review.json`，见 §14.1；I9 就绪前置的补充复审见 `t16-i9-ready-review.json` 和 §14.2。后续测试准备修正、运行证据、最终文档均由作者负责，**不宣称独立审者已签署最终 HEAD**。没有确认需要新增权限 owner、改变 API/数据库/跨模块协议或放宽架构守卫的问题。

## 13. 交接、实际决定与保留边界

M1-P2、M1-P4、M1-closeout、M2-closeout、M3-closeout 各追加 P14 对应子项证据，顶层 status 均保持 open。M2 §8 的表情回应、§11 历史六处中未到 P14 的入口，M3 评论/历史/命令/草稿等仍归 P15；P14 的六处复制与 M2 原六处名单不是同一集合。

保留边界明确如下：链接元数据最多 30 次/30 秒，需要时显式重读；目录未就绪时子项数不能跳转；公开 EditorRefApi 保持合同；测试的 DOM 控制符/合成 paste 不宣称系统权限覆盖；增量复审没有第二位审者。M7 收集箱在 lazy pending 期间 Discard 仍按既有 readiness 提示，外层 Escape 可退出，其关闭/草稿策略没有随共享 editor 迁移统一，已追加到 M7/M3-closeout 的 §5，不能称 M7 已完整验收。描述版本 Markdown 复制及四处剩余复制入口留 P15，M6/M7 自身业务遵守 P10a 裁定 A。

全部实施裁定按台账原顺序见本文件附录；局部问题按负责人授权自行闭环。P16 的全局 dead/no-shadow、M4 设计 §14、总体设计 9.4、M5–M8 最终 M4-closeout 均留真正收尾，本期不提前勾成完成。

## 14. 最终分支 CI、证据保留与交付门

分支 `codex/m4-p14`，基线 main `8bb4bbcb52a224ace4cb2f9d383bca9b97a022bf`。本地验收输入 `b7d1f0c2` 已通过上述门禁。较早产品 `beda23c1` 的 [CI 41188324547](https://github.com/open-nerve/NerveProject/actions/runs/41188324547) attempt 1 三任务成功，仅作为历史证据，不能代替本最终文稿的 CI。

最终文稿提交之后使用系统 Git 推送，`final-delivery.json` 后置保存准确 HEAD、远端 HEAD、run/attempt、server/web/e2e 全成功、主目录仍 main 及产品输入未变证明。**该回执不存在或校验不通过时，A16/T16 和阶段交付门未通过；不得据本节预先宣称最终 CI 全绿。** 这是避免文档自指提交 SHA 的后置记录，与 P13 相同，不是降低准确提交门槛。

准备原件在主目录 `.superpowers/sdd/P14-preparation-20261031/`；实施原件在本 worktree `.superpowers/sdd/P14-web-issue-detail/`，均保留。失败 trace/截图/数据库、日志、mutation、阅读清单和回执需在以后授权合并且 main CI 通过后一起复制、逐 hash 核对，再清理分支/worktree并复验。现在不合并、不清理、不启动 P15。

### 14.1 最终文稿首轮 CI 的 I22 前置修复

`4fafe01e` 的 [CI 41191083876](https://github.com/open-nerve/NerveProject/actions/runs/41191083876) attempt 1，server/web 成功，E2E 的 I22 Back/Forward 用例在标题 fill 时发现元素 detached 并超时。这不是 W11。已通过负责人授权的 Chrome 读取日志；产物存储域被 Chrome 拦截，未绕过限制、未取得远端 trace，准确原文摘录在 `t16-ci-i22-ui-log.md`。

本地原样 10 次通过；8 倍 CPU 的三个探针全部在 Delete 菜单得到同类 detached 行为红。可见 UI 和 trace 证明 `pushState` 已改地址、React 尚未提交目标页：WEB-1 URL 对应的正文仍是列表，peek 场景的 Lists URL 正文仍是 Beta。失败点击在约第 9 秒开始，等待到第 30 秒，不是正常步骤总耗时刚好不足。`t16-ci-i22-red-analysis.json` 保留原 trace 路径、hash 与 action 时间。

局部修复仅在 I22 为每次前进/返回增加目标工作项行或详情标题的真实 UI 前置，Beta 辨识行经 API 准备；之后才打开弹窗或检验 owner 已退役。原关闭、零写入、console 断言和 30 秒超时均不改。相同 8 倍 CPU 修改后 3/3（每次约 21 秒），原两个 I22 2/2；无 CPU 探针代码进入提交。作者按逻辑/集成/覆盖三视角复审单一改动文件及调用链，无生产/架构修改。

最终验收输入 `b7d1f0c2` 已重新跑 Web 四项、默认 238/238、显式三浏览器各 44/44 和 167+2 文件严格 lint；服务端输入零差异证明扩展到该提交。新回执为 `t16-ready-final-receipt.json` / `t16-ready-matrix-*`。旧输入 e0ee0ff7 的已通过本地记录与失败远端文稿记录均保留，不相互替代；本最终文稿仍须准确提交 CI 后置核准。

### 14.2 Firefox 关联页面的加载前置

I22 修正后的首轮显式矩阵，Chromium 44/44、Firefox 43/44。I9 在重载后立即点 Relations，trace 的输入点 (573,297) 在 lazy 描述挂载时失去目标，按钮下移约 44px，最终仍折叠；失败不是权限或关系数据缺失。原 trace、帧及分析见 `t16-i9-ready-trace-analysis.json`。I9 三次打开文档均补实际 editor 挂载前置，随后 Firefox 连续 5/5；没有固定等待、重复点击或警告忽略。作者完整读单一修改文件及关系组件/store/原编辑器边界，`t16-i9-ready-review.json` 绑定补充复审。加载期间的布局跟随和退出仍由 C10 专项验证；I9 不冒称验收此时对移动按钮的指针命中。最终完整 Web 门禁、默认 E2E 及三引擎各 44 已在最终验收输入重新通过。

## 15. main CI 虚拟列表分页前置修复（2026-10-31）

负责人授权后，最终分支 ec5f38b5 经 --no-ff 合并为 main `7c978546ad8b5927a59e04c83475081df477756a`，树与分支终态相同。其 [CI 41196503733](https://github.com/open-nerve/NerveProject/actions/runs/41196503733) attempt 1 的 I19、attempt 2 的 I3 各失败一次，均 237 通过；两次 server/web 成功。这些是实际失败，不能沿用分支绿灯宣布集成完成。

按 02 §9 从 main 新建 codex/m4-p14-ci。两个浏览器下载的报告均核对页面 digest；trace、无 continuation 的请求记录和 8 倍 CPU 对照共同证明：直接点击远处页尾时，自动滚动触发虚拟行从占位切换到实际内容，按钮在 pointerdown/up 间下移 132px，click 落入列表容器。I19 原常速 5/5、限速 2/3；I3 原限速 0/3，修复后两者限速均 3/3。诊断插桩和 120 秒整例预算均精确恢复，不进入最终测试。

现有 issue-list-page fixture 新增 scrollToLastListRow，I3/I19 两处指针翻页先抵达末行真实链接，再执行原按钮 click。原数据/顺序/过滤/权限/HTTP 和错误断言不改；无固定等待、强制点击或盲目重试。I3 的键盘翻页与日历分页结构不改。逻辑、集成、覆盖三视角复审见 [re-22](../../../reviews/re-22-M4P14集成修复审核.md)。该修正不宣称改变产品虚拟化高度策略，实际覆盖边界在报告中保留。

第一轮未插桩三个测试文件输入已通过 make lint-web（54/54，36 缓存）、make knip、make e2e（238/238，build 加运行 67.489 秒）。产品、生成协议、依赖与构建配置无改动；gen/lint-go/Go test/Web test 保留 §10 的原测试输入与回执，第一轮输入复用证明为 ci-fix-input-reuse.json。P14 44 项矩阵所选故事不变，共同 fixture 仅新增未调用 helper、旧函数逐字不变，三引擎证据保留原输入，不伪称新运行。

修复提交 f93c6968 的 [CI 41199938927](https://github.com/open-nerve/NerveProject/actions/runs/41199938927) attempt 1 的分页均通过，仅 06 已知 W11 快捷键失败（237/1）；按纪律重跑后 W11 通过，但另一个 I19 冷启动标签故事失败（237/1）。实际 trace 证明描述编辑器恰在点击期间挂载、把标签按钮下推，旧坐标落到父工作项行，没有标签 GET。原用例本地 8 倍 CPU 3/3，未复现；不能虚构本地红绿。该普通筛选故事补真实 contenteditable 可见前置后仍用原鼠标 click，目录一次读取、冷启动筛选与 quiet 断言不变；加载中交互继续由 C10 覆盖。局部决定的代价是此故事只验加载后目录，不冒称加载中可点击，未改产品或守卫。

追加三视角复审和原始依据见 re-22；补前置后限速 3/3，最终无插桩静态 54/54、knip、完整 E2E 238/238（build 加运行 66.875 秒），三回执 ci-fix-label-* 绑定同一输入 a4e577d542c0ed0bb77b378c8b4c846671ee2dd65260cd4d943a3aa234ba2ce2。未变输入复用证明追加 ci-fix-label-input-reuse.json。W11 未修改，两轮失败、浏览器重跑和新标签报告均保留。

全部集成证据在主目录 `.superpowers/sdd/P14-integration-20261031/`。修复提交推送后，ci-fix-delivery.json 后置核准确修复 HEAD 与分支 CI；随后在既有授权内 --no-ff 合并 main，并以 main-ci-success-receipt.json 核准确 main。只有 main 三任务全绿、原 P14 与修复 worktree 的材料均归档逐 hash 复验后，才能正常清理两个工作树/分支。准备 P15 的文档不得作为其产品实现。

## 附录：全部实际裁定（台账原顺序）

1. 沿已批准 plan 内联执行，逐 Task 聚焦与受影响回归；T4/T9/T13/T16 才完整 F 后推送 — 遵循负责人检查点规则和避免无变化重跑 — 若有新失败或共享风险提前扩大相关验证。
2. skill 的末尾删除 workspace 不适用，证据留到授权合并、main CI 和归档复验后 — 仓库明确保留证据 — 成本为工作树占用空间。
3. 去重归自动保存当前 owner.value，移除描述渲染闭包判断；已有普通确认/失败/卸载机制不改。首次分立事件全绿不计缺陷红，同轮回调红和 M04 精确检出过时撤销写；M05 字符串被 root 运行时校验拦下，两个预期 PATCH 缺失而红，非编译红。均已字节恢复。
4. 原生 Issue + 既有搜索 proof；只读编号 span 根因修嵌套，不改权限 owner；M7 仅共用读取适配。包构建和消费测试串行，避免 dist 清理竞态。
5. 查重源是分诊项，只读 project_id/q 并保持 M7 void ID 通知，关系现有确认链只加回归；页面测试错误详见 review，不作产品红。
6. URL trim 后服务端规范化，表单局部实例与同步锁；仅展开链接区当前 id+URL owner 调度有限 GET，普通回读不重开期限，原 resource 负责晚答确认。旧 URL 在途先结束再读新 URL，元数据终态/失败均不冒充业务失败；没有全局协调器、服务端或权限规则变化。复制留 T15。
7. 编号页头生命周期使用路由证明的 projectId，不能以 canonical 实体存在作为动作 owner 的存活条件 — I17 删除成功后当前返回被误取消，真实 layout/header 测试同样检出 — 仍沿 useDetailScope 与共同 route key，换路由晚答不能导航，无全局 owner/协议修改。
8. 现有编号读取由 layout 一处发起，经 Outlet context 供正文并给页头已证明 id — 页头原来独立看全局缓存，不是当前读取结果 — 代价是正文须在本路由 layout 内，真实嵌套路由测试已验证；没有新增业务缓存或权限 owner。
9. 标题校验按后端 utf8.RuneCountInString 计 Unicode 字符，去 DOM maxlength 的 UTF-16 截断，保持 255 字符合同行为。保留原 useConfirmedAutosave/prepareSave，失败对象为既有 useRefusedFocus 的 attempt，无依赖/协议变化。
10. label-select 正 tabIndex 已由 P12 消除，复验；双日期原子清空复用已有 inline-refusals range 与 I2 API，不添加镜像测试。父项与完整模态归既定 T6/T5，A2 留待父项闭合。提前建立 T16 的小型真实 detail 入口 fixture，避免误称 list helper。
11. 描述回调转换为 load/migration/edit，resetField 仅替换未编辑描述的基线，已编辑描述保持 shouldDirty；标题等其他修改不清除 — 一个表单 owner 管理 dirty — 错误风险由 issue/draft 两种输入及混合编辑测试覆盖，不扩草稿协议。
12. 创建边界省略空 state_id，既有命令补 c 并保留 ni，确认链接展示服务端编号 — 三处均 I1 真实页面复现、设计已要求 — 不新增监听器或协议；日期拆局部 Controller，不改变共享日期控件的默认行为。
13. 聚焦回归通过后先提交产品，再在准确提交串行跑完整 F，F 完成前不推送 — 保证完整门禁有准确 tested SHA — 后续文档回执另记，不把脏树测试或文档提交冒称完整实测产品 SHA。
14. 行内计数沿现有目录生成原生 Link，普通/归档同一 browse，无目录只显示计数 — 和 P12 表格合同一致且消除键盘及旧地址根因 — 目录尚未就绪时暂不可导航，组件和真实页面覆盖该边界；不改全局 URL helper 的其他读者。
15. C2 布局漂移按组件内部缺陷修复 — 实际 lazy chunk 诊断证明按钮下移 48px、Popper 仍保留旧坐标；FiltersDropdown 的定位合同已存在，不改公开接口、领域消费者、权限根或依赖 — 代价是打开面板逐帧读一个矩形，以关闭/卸载释放、默认/自定义按钮和真实 C2/C10 回归覆盖。二次诊断证明宽泛 transition 还把定位 transform 缓动，因此动效仅保留 opacity/translate。原 C2 文件和几何 helper 未改。
16. 现有权限/会话/owner 行为已满足本段，保持产品代码，补真实消费者与页面证据 — 60 项组件与 7 项页面通过；M21 删除确认故意保留旧 fake root，只改变会话，从消费者独立检出漏跟进保护 — 代价是它不替代真实会话根退役，另以 I22 实际两个标签页、真实 route.fetch 的已确认旧响应覆盖。
17. 六个指定入口只复用既有复制 hook；归档 peek 也沿原生编号路由，去掉只驱动旧地址的参数和零读者导出 — 编号原生入口已支持归档且 spec 要求当前 origin 原生地址 — 若错会影响归档跳转，归档真实消费者复制值已覆盖，T16 核原生页面。
18. 字体 CI 异常修在本段 fake-form 的外部浏览器装饰边界，只隔离 Logo，项目选择、表单和 ModalCore 保持真实 — fontfaceobserver 的 Node 轮询超出 jsdom 生命周期，真实浏览器终止文档会清理窗口定时器 — 代价是单测不检图标，I1/C4 页面仍使用真实 Logo；字体轮询探针原红改后绿并恢复，46/46 表单验证。
19. 尚无编辑器实例不阻止取消，已建立实例明确忙碌仍阻止；提交继续要求实例就绪 — 延迟模块不能把用户困在创建窗，dirty 沿原草稿确认 — 若错可能损失编辑器未输出内容，故只允许 ref 为空且未建立编辑内容的情形，实际 chunk 持有及 busy 回归覆盖。
20. 新矩阵通过 e2e/package.json 的 test:m4-detail 注册真实命令入口 — knip 正确指出原配置未被调用 — 不改忽略清单或依赖，和既有 filters/list 入口一致，成本仅增加命令名。
21. 取消与保存分别判断 readiness，Save to Drafts 不绕过普通表单要求 — 新取消入口不能产生未就绪写入，沿同一局部组件检查，不扩 P15 草稿协议 — 代价是脚本未到达时只能保留输入或明确丢弃，不能保存草稿。
22. 原脚本候选逐项查证后删除 dataId、恒 false 的 isVisible 分支、未读测试声明与五个零外部读取的 export — 满足已批准谁改谁清，未删真实 JSON/Go 安全字段 — 若读者盘点漏项会损失入口，已核全仓实际调用并由类型、既有消费者及完整 Web 门禁再核。
23. 公开 EditorRefApi 按 spec §2.9 保持合同，即使部分方法当前无生产调用也不在本次破坏接口 — 明确的保持 ref 约束优先于自动候选，不冒称候选全为假阳性 — 成本为保留这些合同方法，P16/真正收尾再统一判定。
24. 09525b31 已完整 F 通过；最后局部修复只改 Web/测试，前三项保留准确原回执并核 server/api/生成客户端/锁与构建配置无差异，后四项从新产品提交重跑 — 完整证据按实际输入分层，不伪称重跑 Go — 若输入证明不成立须重新跑对应门禁。
25. 三浏览器输入使用同一真实非法字段和剪贴板对象，Mac Safari 采用系统 Option-Tab，预期 HTTP 拒绝按引擎精确匹配 — 首轮 trace/最小引擎探针证明原输入假设不同，业务和 DB/console 断言保持 — 代价是非法控制符由 DOM input 注入，不能冒称用户键盘可输入 NUL；粘贴仍是合成事件，不冒称系统剪贴板授权。
26. 关系菜单先等真实折叠动画揭示行，再执行用户操作；C12 直接进入被测详情 — 原测试点中裁剪内容并自动滚动、或立即导航取消前页请求，均非本场景业务动作 — 若还有揭示完成后错位/丢点击，原几何/动作/零警告断言仍失败，快速加载跟随仍由 C10 保留。
27. I2 的普通 JSON 请求用 request 事件观察，仅在实际拒绝阶段安装 route — WebKit 实际应用对照证实初始拦截引发多余 preload，观察器取得了完整真实 migration body — 代价是这个场景只适用于当前普通 JSON body，若以后改 streaming，完整 body 断言会失败，不能放空通过。
28. 独立审查只覆盖 8bb4bbcb..9a0f32e4，后续修复、dead 与运行证据由作者负责复审 — 不把作者报告冒称独立签署，全部增量另记源码与门禁输入 — 代价是增量没有第二位审者，实际回归及全量门禁承担验证。
29. M7 收集箱 pending 时 Discard 沿既有 readiness 提示，外层 Escape 仍可退出；不在 P14 改其独立关闭/草稿策略 — 已批准范围只迁共享读取与保持 editor ref，M7 自身未接通能力归 M7 — 代价为慢加载期间该按钮不可用，必须显式记录到 M7 交接，不能称 M7 完整关闭已验收。
30. 描述版本 Markdown 复制与其余四处复制沿 P15 既定范围 — P14 六消费者已验证，保持公开 copyMarkdownToClipboard ref 不等于核完历史消费者提示 — 代价是那些既存复制反馈仍未收口，M2/M3 顶层 handoff 继续 open。
31. M6/M7 自身业务不因共享 helper 已适配就提前验收，浏览器和最终构建事实只依准确产物 — 尊重 P10a 的 A 裁定和审查者只读边界 — 代价是领域自身端到端保证仍待各自里程碑，P14 不冒称全站完成。
32. I2 Chromium 用精确 URL 的 fallback 拦截开启请求体采集，WebKit/Firefox 保持 request 观察；四个完整 body 断言不变 — 同应用 Chromium 对照证明无拦截 body=null、有拦截取得实际 JSON 且无 preload 警告；WebKit 先前对照相反 — 成本是仅采集层按浏览器分支，不更改真实业务请求或放宽警告。上一轮 Web F 236/238 的另一失败为 I10 API 仍使用旧 PNG CRC 期望，已同步已修正有效图片字节。

33. I22 在每次导航及 Back/Forward 后用唯一工作项行/详情标题确认目标页面，地址栏变化不作 React 路由已提交的证明 — CI 的 detached 输入和本地 8 倍 CPU 三次 detached 菜单均指向旧页面控件；补就绪前置后同条件三次通过，保留全部关闭/零写/console 断言与原超时 — 代价是该故事验收已到达页面后的 owner 退役，不冒称覆盖路由加载未完成时继续点击旧页面的产品行为。

34. I9 的三个新文档先确认真实描述 editor 已挂载再操作关系按钮 — Firefox trace 的点击点 (573,297) 在 lazy 描述进入时失去目标，关系按钮下移约44px；这是加载后关系故事缺前置，C10另验pending布局跟随 — 代价是I9不验加载过程中对移动控件的指针命中，既有C10布局跟随与慢加载取消场景保留；不加sleep/重试点击或放宽警告。

没有以 deferred minor 名义搁置本段的局部缺陷；明确后续范围与未核验领域见 §13。
