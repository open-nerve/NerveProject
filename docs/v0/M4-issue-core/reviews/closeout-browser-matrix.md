# M4 / C12 最终浏览器矩阵

状态：T5–T7 候选验收完成；最终候选等价与 main 绑定仍待 T13/T14。原 P16 自动回归只作准备来源。本次对 21 组 × Chromium/Firefox/WebKit 分格记录，不把自动断言或历史成功当作人工视觉结果。

每格的细分动作、角色、入口、返回值/持久化、截图/trace/网络控制台、引擎版本及源码/bin SHA 收在 `.superpowers/sdd/M4-closeout/browser/`；控制者实际看图后写观察。任意失败或未看截图的格保持待验。最终 main 等价绑定在 T14 获准合并后补。

| 组 | 页面/入口 | 身份/故事 | 动作合同来源 | Owner | Chromium | Firefox | WebKit |
|---|---|---|---|---|---|---|---|
| B01 | 项目列表 | member；I3/I4、C1/C3；S2 admin/member/guest/non-member | 分组/排序、属性选择器、行菜单、真实右键方向键/IME Escape/回焦；精确请求与拒绝；轮廓/裁切/背景/回焦按 spec §2.2 | T5 | T5通过 | T5通过 | T5通过 |
| B02 | 看板 | member；I3/I6、C1 | 看板顺序、拖放确认/拒绝；菜单沿共用路径；轮廓/裁切/背景/回焦按 spec §2.2 | T5 | T5通过 | T5通过 | T5通过 |
| B03 | 日历 | member；I4/C1 | 日期区间、当前显示偏好；组件另核新周起始输入；轮廓/裁切/背景/回焦按 spec §2.2 | T5 | T5通过 | T5通过 | T5通过 |
| B04 | 电子表格 | member；I3/I4、C1 | 列排序/显示和原生数据；共享选择器由 C3 保护；轮廓/裁切/背景/回焦按 spec §2.2 | T5 | T5通过 | T5通过 | T5通过 |
| B05 | 快速添加 | member；I7 | Enter/错误字段先说明再回焦、服务端拒绝保留；确认写；轮廓/裁切/背景/回焦按 spec §2.2 | T5 | T5通过 | T5通过 | T5通过 |
| B06 | peek | member；C3/C4、I1/I2 | 加载中关闭/迟到读取、布局选择、标签、复制失败；回焦或旧触发器退役；轮廓/裁切/背景/回焦按 spec §2.2 | T5 | T5/T6通过 | T5/T6通过 | T5/T6通过 |
| B07 | 富筛选与显示 | member；I5/C2；single/multi en/zh-CN 兼容夹具 | 搜索/清空/选中、0/空值组件合同、日期草稿、桌面/窄屏位置、Escape/回焦；轮廓/裁切/背景/回焦按 spec §2.2 | T5 | T5通过 | T5通过 | T5通过 |
| B08 | 归档列表 | member；I15/C9；S2 admin/member | 归档/恢复、确认窗口 Tab/Escape、Back 不写、精确冷加载请求；轮廓/裁切/背景/回焦按 spec §2.2 | T5 | T5通过 | T5通过 | T5通过 |
| B09 | 个人 assigned/created/subscribed | member；I19；S2 admin/member/guest | 三关系、局部偏好、访客无列表/工具栏/旧读取；轮廓/裁切/背景/回焦按 spec §2.2 | T5 | T5通过 | T5通过 | T5通过 |
| B10 | 编号/项目详情 | member/guest/non-member；I1/I2/I20/I22、S2(detail) | 200/404/403、跨工作区错误地址、会话更替；编辑/拒绝/回焦；轮廓/裁切/背景/回焦按 spec §2.2 | T6 | T6通过 | T6通过 | T6通过 |
| B11 | 创建与更新表单 | member；I1/I2/C4/C10/C11 | 表单不误提交，日期/标签/父项内层窗口隔离，IME、确认窗 Tab、Back；轮廓/裁切/背景/回焦按 spec §2.2 | T6 | T6通过 | T6通过 | T6通过 |
| B12 | 子项/关联/链接 | member/guest；I8/I9/I10/C4/C5 | 搜索/创建/删除确认、复制失败、子项显示/筛选按钮；当前权限拒绝；轮廓/裁切/背景/回焦按 spec §2.2 | T6 | T6通过 | T6通过 | T6通过 |
| B13 | 删除 | member/guest；I17/I20/C7 | 当前工作项命令、确认窗、忙碌 Escape、写一次后离开；轮廓/裁切/背景/回焦按 spec §2.2 | T6 | T6/T7通过 | T6/T7通过 | T6/T7通过 |
| B14 | 描述编辑器 | member；I2-description、editor-loading、cleanup-interactions | 撤销重做/拒绝重试、按需加载/粘贴安全；继续输入原生 Tab、表后输入入库；轮廓/裁切/背景/回焦按 spec §2.2 | T6 | T6通过 | T6通过 | T6通过 |
| B15 | 表格行/列菜单 | member；cleanup-interactions | Tab、具名触发器/面板、IME Escape、回焦、第二行/列键盘插入且入库、应用内 Back 退役；轮廓/裁切/背景/回焦按 spec §2.2 | T6 | T6通过 | T6通过 | T6通过 |
| B16 | 评论/表情/动态 | member/guest；I11/I12/I20/C6 | 读写权限/作者、编辑取消/拒绝、分页、筛选/排序、表情；确认写与焦点；轮廓/裁切/背景/回焦按 spec §2.2 | T7 | T7通过 | T7通过 | T7通过 |
| B17 | 描述历史 | member；I13 | 首页/下一页、两版内容/恢复、当前路由验证后挂载；轮廓/裁切/背景/回焦按 spec §2.2 | T7 | T7通过 | T7通过 | T7通过 |
| B18 | 草稿创建/编辑/发布 | member、guest；I14/C9、S2 | 保存/重复提交/拒绝、复制迁移、发布、权限重载、关闭/Back；访客拒绝说明；轮廓/裁切/背景/回焦按 spec §2.2 | T7 | T7通过 | T7通过 | T7通过 |
| B19 | 命令弹窗与顶部搜索宿主 | member；I18/C7 | 搜索、键盘/IME、当前对象、双宿主标签多选、旧范围迟到响应、Back/会话退役；轮廓/裁切/背景/回焦按 spec §2.2 | T7 | T7通过 | T7通过 | T7通过 |
| B20 | 局域网 HTTP 复制 | member；C8 | 十二个实际复制入口，在无手势/非安全上下文说明失败、不假成功/写入/导航；轮廓/裁切/背景/回焦按 spec §2.2 | T7 | T7通过 | T7通过 | T7通过 |
| B21 | 设置等共享选择器消费者 | admin/member；C3、S2 | 自动化月份、项目日期筛选、API token 日历、workspace 菜单；表单隔离/回焦；轮廓/裁切/背景/回焦按 spec §2.2 | T7 | T7通过 | T7通过 | T7通过 |

## 选择与签结约束

原组合配置每引擎 125 个唯一 id；新增用例仍须证明原并集完整，无 unexpected/flaky/意外 skip。S2 保留四身份，M6/M7 自身页面仍按原批准归属。LAN HTTP 用真实本机可达地址，只接隔离库；十二复制入口逐个记录安全上下文/手势和实际结果，不推断 HTTP 一概失败。

## 控制者观察

T5–T7 已完成下列观察；最终 main 绑定留待获准集成后签结。


### T5 列表筛选与局部修复

本次使用已锁定 Playwright 通过真实 API 建数据并操作页面，控制者用 `view_image` 逐张看图。浏览器 Chromium 153.0.8010.12、Firefox 155.0、WebKit 26.6；真实路径/身份/title/testId/viewport/focus/原图位置逐条保留，不从题名推断通过。

原选择 `t5-selection-original.json` 各125项，本组10文件各46项。构建收据 `t5-build-fix.json`；程序 SHA256 `4c257e8f6481656fa27065ac42a2d2741cc3f35d06ae0f9f5b0846f177f8b72a`，各后续浏览器输入相对此构建仅有E2E观察/键盘断言变化，见 `t5-coverage.json:buildEquivalence`。配置 SHA 与初始完整选择在 `t5-build-manifest.json`。

| 组 | 具体操作与实际观察（三引擎） | 原始状态/结果 |
|---|---|---|
| B01 | 行真实右键→方向键/End/Home、IME Escape后回焦；状态/优先级/负责人/标签与起止日期、续页和拒绝。菜单蓝色键盘轮廓可见、选项/日期面板贴触发器且无裁切；错误toast可辨 | `context-keyboard/context-end`、`selector-*`、`*-calendar`、`continuation-complete`、`property-refused` |
| B02 | 实际拖动慢写时保持原卡片，拒绝回到原状态；确认后重载位置与DB一致。泳道/计数可辨，无悬空菜单 | `drag-pending/drag-refused/label-priority-lanes`，I3/I6及原C1断言 |
| B03 | 月视图每格四项与Load more、日期拖动拒绝回退；390×844选择17日展示当天清单；返回1280×720选择Week。周/月范围、API query和日期写入由原断言核对 | `month-four-per-day/date-drag-pending/add-existing-refused/mobile-selected-day/week-local-range` |
| B04 | 101项跨页的负责人、标签、子项数、截止日各升降序；编号/标题列稳定。横向滚动不同引擎位置略异，原生列与真实排序结果一致 | `table-{assignee_name,label_name,sub_issues_count,target_date}-{ascending,descending}` |
| B05 | 空名称说明先出现、拒绝保留文本、Enter慢写不重复，确认后插入。红边/错误文本与 Adding work item 均可见 | `required-name/refused-name/creation-pending` |
| B06 | peek加载骨架可关闭，迟到读不复活；模式选择/标签内层Escape只关一层。子项Display/Filters在720/500高能滚到末项，真实Tab轮廓完整、Escape回当前触发器；复制/编辑器加载留T6 | `loading-peek/mode-options/selector-Labels*`、`sub-item-*-{500,720}[-last-control]` |
| B07 | Add/multi/operator搜索/清空/键盘选择、日期草稿说明；中文优先级/运算符和single/multi en/zh-CN兼容夹具；11显示属性、桌面/390宽位置。拒绝后真实Space成功再写，保留原错误toast | `search-*`、`chinese-operator/date-draft-error/display-{1280,390}/eleven-display-properties/preference-refusal-keyboard-focus` |
| B08 | 归档确认慢写禁用，原生Tab（macOS WebKit Option-Tab）从Archive回Cancel，三引擎轮廓完整、背景隔离；Escape/Back不写，归档详情横幅只读，恢复一次 | `archive-pending/archive-tab-trap/archived-read-only` |
| B09 | assigned/created/subscribed三关系与跨项目数据/局部偏好一致；访客桌面/390宽明确拒绝，无列表/工具栏及无权取数 | `profile-{assigned,created,subscribed,guest-1280,guest-390}` |
| B11/B21（部分） | 创建标签/项目面板、项目自动化月份、项目日期范围、PAT日历可见并保留外层隔离；S2菜单四身份合同保全。整体仍待T6/T7 | `selector-{Labels,Web}`、`automation-months/project-date-filter/token-expiration-calendar` |

`browser/t5-stable-{chromium,firefox,webkit}` 原始整组结果分别46/0、45/1、46/0；Firefox失败来自错误toast遮住下一按钮中心，`t5-feedback-{chromium,firefox}` 真实键盘重验各1通过。`t5-keyboard-2-*` 每引擎2通过，补上末项真实Tab和归档回绕焦点；原固定位点 focus 在Firefox没有显示键盘轮廓，未冒充通过。最终各46唯一id通过的并集在 `t5-coverage.json`，未丢弃失败尝试。

稳定201图加键盘27图共228张，由控制者查看26张索引及必要原图；`t5-visual-review.json` 逐图含hash/路径/观察。兼容夹具截图只用于控件合同，不能代替产品页面。严格网络/console/CSP观察沿原故事，没有放宽期望错误或忽略规则。

修复证据：`t5-peek-clipping-red` 在720高度真实断言失败；初次输出误放E2E非忽略目录而造成 inputUnchanged=false，全部产物原样迁移并保留 `t5-peek-clipping-red-artifact-relocation.json`（差异仅新增产物），没有修改原失败收据。局部面板高度修复后 `t5-peek-green-*`、整组及500/720高度末项补验全部通过。首次截图落在动画开始帧，帮手只等待实际有限动画，不禁用动画；`t5-dropdown-unit`零命中不计数，正确命令 `t5-dropdown-unit-2` 7项通过；错误 solid-only 轮廓断言不算产品反证。Web lint 54/54、E2E类型/lint通过，最终候选/main等价另T13/T14。

### T6 详情与编辑器验收

本组原38项/引擎均通过，另增中文编辑器1项/引擎。原始整组 `t6-{chromium,firefox,webkit}` 每组38通过；真实键盘补验 `t6-keyboard-4-chromium/firefox` 各8通过；Docker恢复后 `t6-recovered-webkit` 8通过、中文新例1失败。中文测试先修正选中单元格再次点击、再限定详情编辑器范围，`t6-chinese-2-*` 每引擎1通过。最终39个唯一测试/引擎的有效并集、所有失败尝试与等价检查在 `t6-coverage.json`，不伪称一次完整全绿。

本组唯一生产修改是链接卡片复制按钮去掉 `outline-none`，使用既有焦点轮廓。`t6-keyboard-chromium` 先业务失败，重建及六复制入口/嵌套菜单的三引擎原生Tab补验通过。最终程序SHA256 `ac3ca7dcc70ed74a5ed69e32d2e9092cd29836eb90c4effc0921a0e6b7676af6`；原整组用T5程序，差异仅该CSS，后续受影响入口全部重验。接口/权限/owner/编辑器逻辑未变。

| 组 | 实际观察与持久化/拒绝证据（三引擎） |
|---|---|
| B06 | peek复制拒绝无成功/导航，加载前后Display锚点均贴原按钮；真实Tab复制轮廓由keyboard-final补齐。 |
| B10 | 标题服务端拒绝有红边和说明；确认值已保存。guest/workspace-admin own-only隐私保持、隐藏与不存在均404；编号/id登录后真实描述保存。复制拒绝提示可见，无写。 |
| B11 | 必填/超长/服务端拒绝保留字段；创建慢写禁用，内层日期/标签/父项与确认窗保留外层背景隔离。真实Tab确认窗轮廓可见，未就绪编辑器不丢输入；旧账户迟到结果保留新表单。 |
| B12 | 跨项目关联与三子项已确认；正在添加/链接等待禁用，URL/重复拒绝指向字段。初次程序focus不作为Firefox轮廓证据；keyboard-final真实Tab/方向键补齐六复制入口，新修复链接按钮轮廓清晰。 |
| B13 | 子树删除忙碌说明含归档后代，结果只留邻项；命令删除入口仍待T7。 |
| B14 | 描述503保留文本及Retry，重试Saved；表后Continue writing蓝色轮廓完整，表后输入保存。native callout与粘贴保留文本无上传，懒加载路径有实际保存。 |
| B15 | 行/列菜单贴当前目标、背景隔离；真实Tab触发器/菜单轮廓完整，第二目标键盘插入位置由页面/DB核对。初次嵌套程序focus截图不签Firefox轮廓，keyboard-final按相邻控件Tab补齐草稿/peek并回焦。 |

原整组198图、最终键盘索引55图、中文9图，共262个观察记录由控制者逐图检查（30张比较索引）；其中最终键盘索引保留一张中途失败中文例的行菜单图，仅作已观察的部分状态，不用于判该例通过。最终中文3状态×3引擎另在 `t6-chinese-2` 完整记录。`t6-visual-review.json` 含路径/hash/观察；原生轮廓、菜单位置、背景隔离及返回焦点均可辨。中文核验范围为既有行/列触发器与面板的中文可访问名称、继续输入名称与轮廓；既有菜单项中的英文文案未冒称全部翻译。

WebKit在Docker socket丢失时的中断保留为 `t6-docker-interruption.json`，不用于签结。真实Tab按macOS WebKit的Option-Tab；程序focus不冒充键盘轮廓。I10页面仅证明blocked/失败/元数据的用户结果，危险dial仍由T3 Go合同证明。B13命令删除入口由T7续验；C12最终候选与main绑定仍待T13/T14。

### T7 协作、搜索与 LAN 验收

三引擎每组53项，另补S2工作区菜单四身份各4项。Firefox53、WebKit53和三份workspace各4项一次通过；Chromium初次44通过/9失败，LAN定向24项先22通过/2失败，命令ID/title的4项最后通过。前两次失败分别来自同一toast的标题/正文重复文本计数，以及命令文案句号未纳入匹配；修为通知dialog计数与精确文案后通过，未改产品行为。Firefox原生拒绝诊断由实际调用位置读回并核对完整文案/数量，严格console和网络检查不放宽。

完整证据并集 `t7-coverage.json`：T5 46 + T6 39 + T7 53 = **138唯一用例/引擎**，原125项/引擎全部命中；额外S2四身份与T5同id，不重复计数。最终选取无失败、flaky或skip，所有失败尝试保留。T7产品输入与 `t6-focus-build` 一致，bin SHA256仍为 `ac3ca7dcc70ed74a5ed69e32d2e9092cd29836eb90c4effc0921a0e6b7676af6`。

| 组 | 三引擎实际操作与观察 |
|---|---|
| B13 | 当前工作项命令打开删除确认，说明含后代范围；原测试确认一次写与成功离开，T6另覆盖忙碌/回焦。 |
| B16 | 作者编辑/取消与拒绝保留；管理员对他人仅复制/删除，归档表情只读；慢写保持原计数。筛选原生Tab蓝色轮廓完整，组合表情搜索框清晰；205动态三页/重开读新内容，自动归档显示Nerve、恢复显示成员。 |
| B17 | 三版本两页菜单贴触发器，历史内容/403和500说明可辨；恢复慢写禁用且Escape不关，成功回焦。归档无恢复、guest无历史请求或入口；原图的程序焦点不作为额外原生Tab证据。 |
| B18 | 51草稿可达，无项目表单保留未解析字段，复制/迁移后内容一致；拒绝与发布冲突保留输入。外内窗口背景隔离；Back后新工作区蓝色标题焦点与输入保留，权限恢复使用新表单。 |
| B19 | 中文/特殊字面/编号搜索结果与权限一致；双宿主迟到响应只留当前结果，顶部标签多选保持面板/确认勾选。高亮与弹层位置清晰，IME不误执行。 |
| B20 | 十二入口无手势均native=false且仅一个错误；真实Enter均native=true且一个复制成功提示，创建原成功提示另保留。菜单回焦轮廓可辨，无写入/导航；实际LAN origin及72项原生结果保留。 |
| B21 | 主题确认后深色可辨，旧成功不改新会话浅高对比主题；四角色工作区菜单贴触发器，权限入口符合身份；月份/日期/PAT选择器由T5观察保全。 |

240个截图记录由控制者查看27张比较索引与3张额外原图；历史403/500同状态名来自不同用例，额外原图补齐，未因比较索引重名漏看。`t7-visual-review.json`逐图含hash/路径/结论；I16六张截图取自实际page测试既有附件，不虚构DOM焦点数据。三引擎版本同T5/T6。LAN72项（12入口×2手势×3引擎）逐项记录真实HTTP origin、isSecureContext=false、navigator.clipboard不可用及native fallback返回值，不强制“HTTP复制都失败”，也没有模拟返回值。

至此21组×3的分支候选均有实际操作、成功截图、持久化/拒绝与网络控制台证据；C1–C11保持各Phase历史来源，C12使用本次新证据。T13仍须核最终产品输入，T14获准集成后才绑定main并最终签结D12。

### T13 待补验：R09 访客空结果

B01/B07 增补访客设置筛选→空结果→清除筛选→确认后恢复列表；读取偏好可写、创建工作项仍禁用。三引擎使用最终新构建，旧 T5 记录不代替本轮修复验证。

### T13 待补验：R10/R11 分组表头

B01/B02 对列表/看板实际分组标题与创建按钮做原生 Tab、Enter/Space、折叠/恢复及轮廓截图；B02 增补访客泳道无创建入口、成员正常创建入口。原 T5 观察覆盖其余路径，本项必须绑定最终构建再签。
