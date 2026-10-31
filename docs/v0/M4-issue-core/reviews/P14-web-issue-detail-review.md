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
