# M4/P12 列表页面：预检与实施记录

日期：2026-10-30。**当前：T1–T15及检查点CI完成；T16整审六项重要问题已修复，最终WebKit和文稿HEAD的CI待闭合。** [spec](../specs/P12-web-list-pages.md)、[plan](../plans/P12-web-list-pages.md)、[deep-review预检](../../../reviews/re-16-M4P12预检审核.md)。整分支结论见[re-17](../../../reviews/re-17-M4P12整体审核.md)，A1–A12映射见§24；保留尚未满足的验收，不合并。

## 1. 前置、身份与评审方式

负责人授权先完成P11集成、main CI与归档清理，然后仅准备P12。P11交付42afa24f已--no-ff合并main `ce3c39abef99a290157b282c23180983f8746bcb`；[准确main CI 41005896280](https://github.com/open-nerve/NerveProject/actions/runs/41005896280) attempt1三任务success，归档和清理后的再校验见[P11 review§15](P11-web-detail-data-review.md#15-授权合并main-ci与归档清理2026-10-30)。

P12分支 `codex/m4-p12`、worktree `.codex/worktress/codex/m4-p12`从该main建立，主目录始终main。原生create_worktree工具不能指定负责人规定的路径/分支，故用系统git worktree add；未安装依赖，未修改产品或phase，未运行产品测试。

使用brainstorming/writing-plans准备；原型及逐Task重放按02§3调整为静态合同、调用链、既有测试核对和独立builder-check。deep-review由主控依次执行逻辑正确性、集成安全、测试覆盖三个视角。builder-check是技能要求的一次新审者及同一审者的限定复核，未将主控结论称为多个独立审者。

原始材料位于主目录 `.superpowers/sdd/P12-preparation-20261030/`，在worktree之外，未来归档必须一起保留。`project-route-inventory.json`记录原22路径、存在性与哈希；`builder-check-01.md`为初审，`builder-check-02.md`为修订复核；`source-review-inventory.json`区分全文、合同章节和仅检索，`precheck-static.json`记录实际静态检查；最终提交身份另在`precheck-delivery.json`后置登记，避免在提交里引用自身SHA。

## 2. 验收标准逐条映射

| 标准 | 预检处理 | 产品证据状态 |
|---|---|---|
| A1–A2 搜索/权限 | 对照searchIssues生成合同、ExistingIssuesListModal三个调用、DetailResource/PermissionReads；明确空词、日期数组、for/source、scope proof、ID选择和部分成功结果 | T1待实施；不把P11资源单测当新搜索验证 |
| A3 路由 | 原22路径14存在、8已删除，明确utils目录选择及守卫扩围；已正确读props的复核，局部变量误命名不豁免守卫 | T2待实施 |
| A4–A6 列表/快建/拖放 | 沿P10b/P11确认协调，点名A∩C续页、权威泳道总数、字段取焦顺序、中间below、列尾、旧宿主跟进 | T3–T5、T13/T14待实施 |
| A7–A8 表格/日历 | 原生排序及单一偏好真值、两方向二页、browse链接、本地日期、只改target_date与多选确认/失败分开 | T6/T7/T14待实施 |
| A9–A10 控件/peek | 七下拉按两Task拆分；Calendar用Popover，离散选择用Listbox/Combobox；已有ContextMenu inert只复核；P11owner机制不重造 | T8–T12待实施 |
| A11 浏览器/请求 | C1三浏览器单独配置，testMatch筛文件+grep筛标题，覆盖默认A4限制；watchPage与既有EMOJI精确预期分开 | T13–T15待实施，不以默认Firefox A4充数 |
| A12 门禁/整审 | G、检查点F、至少15项不同性质有效变异、准确最终分支CI、完整deep-review/修复复审/归档要求明确 | 本次仅文档预检；阶段产品整审以后执行 |

## 3. 准备阶段的决定

1. P11已批准的root统一权限失效、唯一canonical和编号列宽A方案不重新选择。搜索用原生IssueCollection及现有资源读取证明，不添加第二个权限反应或旧DTO转换层。
2. 原设计15个粗项细分16Task，七个下拉分两组，预计约一万至一万七千行（含测试/删除/文档）。这是准备估算，不能当已量得diff；实施前逐Task重新量，若仍超约16Task的承载能力，先提交后上报拆分。
3. 子项/关联与日历搜索同Task迁移；SelectionSubmitResult只表达这次命令的确认ID/失败ID。UI查询防抖窗立即隐藏旧候选，同scope换词保留有权限的已选ID，换业务或权限scope清空；数据仍从canonical读。
4. DateRangeDropdown是Calendar网格，单Popover符合其交互；Listbox/Combobox用于离散选择。设计§7.5同步澄清，原props/范围写语义不变。P13 R36共享日期根因由T8实际验证后引用，完整筛选行为及R38仍归P13。
5. P12保留既有EMOJI_CHECK_WARNING的精确预期，编辑器分割/移除提示仍P14。按浏览器和真实加载次数固定文本/次数，不能观察到什么就期待什么，也不能滤掉所有warning。

以上是既定P12范围内的实现细化和文稿修正，无新公开API、后端领域协议或权限归属调整，无需另开架构决策。未对未来未实现产品预先给通过结论。

## 4. 预检发现、修订与复核

| ID | 来源/级别 | 根因与处理 | 结果 |
|---|---|---|---|
| B1 | 独立builder-check / Medium | “除主动拒绝外console全零”与P14负责移除现有Canvas提示冲突；spec§2.8/D8、plan T13/T15保留精确第三方预期，三浏览器分别固定 | 同一独立审者复核：nothing important left |
| R1 | 主控逻辑/集成 / Major | 初稿要求日历部分成功不重复写，但旧Promise<void>回调不能把成功ID交回选择器；选择所有权也不够明确 | spec§2.2/D9、plan T1/T7规定结构化UI结果及单一选择owner，T1即落部分成功测试；独立复核未发现新问题 |
| R2 | 主控集成 / Major | 七下拉按“无搜索全Listbox”会把Calendar按钮网格当成离散选项，键盘职责错误 | spec§2.6/D7、plan T9及设计§7.5明确Popover例外；公共日期载荷不变 |
| R3 | 主控覆盖 / Minor | 初稿“testMatch仅(page)”混淆文件与测试标题选择，可造成浏览器验证选择错误 | plan T15明确文件testMatch、标题grep及替换projects；每浏览器先--list核非零再运行 |

三视角顺序复核后开放发现0。完整原文保留，不以复核覆盖首轮。独立最终复核的spec SHA-256为`4d597656a0739a3863d29eef34265b5b611de867a4659af394f758fed7648ee2`，plan为`907f42b7ef7e0c7805f29cc4c7607355d941f3cec92883423cb5036ee4c85fc2`；后续若只将文首“预检中”改为“通过”，由主控静态回执登记该元数据差异，不冒称独立审者读了后来字节。

## 5. 与原粗计划的差异、覆盖和限制

实际未实现产品，因而没有“最终实现与plan差异”。16Task的拆分、取消本轮产品原型/回放、日期控件原语细化、已知warning验收归属和多选结果合同均已同步spec/plan/设计，文档没有待裁定项。

P12 spec/plan全文审查；对搜索/资源/权限/路由helper/日期/快速添加/菜单/peek等关键代码与相关既有测试做全文或明确合同段阅读，完整清单见source-review-inventory.json。全部布局消费者、所有历史交接原件、整个M4设计和整仓代码没有全文审查；原22路径的存在性/相关命中不等同22文件产品整审。未运行浏览器/编译/Vitest/E2E，不认证现有四布局行为无缺陷。产品验证仍由各Task和最终整分支审查完成。

## 6. 移交事项

- P12实施接收：T1原生搜索及三个调用方；T2原22路径；T3–T12真实布局/控件/宿主；T13–T15故事与三浏览器/F；T16整审、修复及准确CI。收到下一步实施安排后才改phase、装锁定依赖、从T1开始。
- C3逐项登记：七控件的popover/list锚点和单状态全部归P12修复/局部浏览器核验；project/select-base通过项目列表打开的创建窗，detail label-select通过列表peek，date-range/priority/state/label-dropdown通过列表行或表格即可触达。这里只验证这些控件，不预签完整创建/详情C4/C5。
- 成员八个M4调用方当前为all-properties、spreadsheet/assignee-column、peek/properties、issue-modal/default-properties、issue-detail/sidebar、relations/properties、sub-issues/properties、workspace-draft/draft-issue-properties。前三处为P12直接页面；创建及详情/关系/子项在P14完成完整页面，草稿在P15；成员基类单状态/禁用/返回焦点回归可复用但不替代页面验收。四个富筛选入口仍P13。
- CustomSelect的peek页头为P12直接验收；自动归档和通用日期筛选作为已接通M3页面的共享兼容核验，不扩到未完成的M4富筛选完整故事。P14仍补详情侧C3，C12全应用三浏览器仍收尾。
- P13–P16及M5–M7的全部剩余责任按spec§5；EMOJI提示删除和编辑器分割留P14，@nerve/ui泛型最后收口留P16，旧搜索DTO/服务最后删除留P15。输入handoff只在实施实际完成后记局部结果，当前不提前关闭。

## 7. 结论与下一步

文档预检通过，可在负责人安排后按plan从T1实施；本轮按要求停在准备阶段。只提交文档，保留codex/m4-p12及worktree，不实现、不合并P12。M4设计§14、总体设计§9.4、输入handoff顶层状态及M5–M8最终closeout没有改动。


## 8. T1 原生搜索与添加已有（2026-10-30）

三个真实调用方已原子迁移到 IssueRootStore 持有的 IssueSearchService；原生 query/IssueCollection、不存旧搜索 DTO。hook 使用宿主 DetailResource 和 root 权限证明，防抖只延后 HTTP，查询变化立即隐藏并退役旧候选；冷项目/状态目录先证实再接收 canonical。选择器只存 ID/证明，同范围跨词多选、换范围清空；窗口命名、无正 tabIndex、候选无嵌套控件。

日历等待逐项真实确认，只写 target_date，部分成功保留失败选择且重试不重复成功 PATCH；子项/关联维持原子批量确认。实际组件+FakeNerve 覆盖查询、当前拒绝/重试、迟到响应、权限 ABA、关闭重开、重复点击、批量补读失败；原有 scope/选择/拒绝性质迁移保留。删除无生产消费者的旧 callback 和预选 props；编号用现有目录的纯文本展示，避免 IdentifierText 的禁用按钮进入选项。没有修改 shared IdentifierText 或提前认领 P14。

验证：最终聚焦 10 文件 149 测试通过（0 skip），类型和格式通过；修改路径 oxlint 0 警告，web cap 随实际从 145 降为 143；关键词 76 规则/24 例外无命中，1241 翻译读取均已定义。原始 argv/cwd/源码输入哈希/dirty 状态/raw/exit 在本 worktree `.superpowers/sdd/P12-web-list-pages/` 的 `t1-final-regression`、`t1-types-final`、`t1-lint-final`、`t1-format`、`t1-keywords`、`t1-i18n` 收据。

M01 错搜索地址、M02 防抖延后失效、M03 放行其它项目三项生产变异均被行为断言捕获；原字节 SHA-256 精确恢复再绿（各 receipt/patch/red/restored 原件保留）。最初新模块不存在的导入红不计行为证明。日历提前返回、权限丢失仍 loading、候选嵌套编号按钮另有实际红→绿。jsdom 缺 ResizeObserver 与一处测试 getter 名误写属于夹具修正，未计生产缺陷或变异。T1 自查与修复复核没有未决架构问题；F/页面/三浏览器/最终整分支审查与 CI 留在既定后续任务，不能由本次单元验证替代。

## 9. T2 显式地址、目录和权限（2026-10-30）

原22路径再次逐项核实，仍为14现存/8删除，原件 `t2-route-inventory.json` 含每项当前哈希。地址正确的只复核；归档页局部变量改名；所有列表权限调用明确工作区/项目，缺地址不触发旧回退。目录投影从 utils 拆为 group-columns 单一职责模块，全部消费者显式传 scope，同步读取不发请求；profile/workspace 与项目范围分别选目录，保留 M7 getter 的其它真实消费者。

真实 route=B/store=A 夹具先复现标签/权限错误，修复后成员、访客、缺项目、跨工作区及无 HTTP 均通过。project-address-route 保留 M3 原表达式与文件并扩围到22路径及目录调用方；新增规则区分属性/解构访问与合法局部变量，不增加豁免。M04 错路由被行为断言与守卫捕获，M05 同步发 HTTP 被实际请求断言捕获；另将真实权限调用退回隐式参数，守卫确实报红。均精确恢复原字节再验证，patch/哈希/日志在实施证据目录。

最终聚焦7文件102测试通过（0 skip）；共享包强制构建10任务、类型、修改路径零 lint、关键词76规则/24例外无命中。web lint cap 按实际143→137收紧；顺带修正这批组件的回调依赖和局部同名变量。证据为 `t2-final-regression`、`t2-shared-build`、`t2-types-second`、`t2-changed-lint-final`、`t2-lint-final`、`t2-keywords-final`。初次测试中的成员夹具 ID 双前缀、标签参数遗漏及扩围正则误命中合法局部名均单记为修订过程，不冒充生产缺陷。没有新权限归属、协议或架构决定。

## 10. T3 列表分页与重试（2026-10-30）

真实列表加载控件改为原生具名按钮，未分组列表也提供手动入口；待加载按钮禁用，分页失败暂停自动观察，显式重试仍复用 binding 的原请求。实际双击复现同一续页重复 GET，现有 binding 内按正在加载的组/格对 fetchNext 和失败重试去重，没有新增数据所有者。其它组、已有行、服务器权威总数及多值数组保持。

新 List 页面组件测试使用真实 ProjectLayoutRoot、目录/偏好/生成客户端和 FakeNerve，覆盖 A 筛选/C 分组续页与同请求重试、重叠页去重、末页停止、空组显隐、部分失败不清其它组及第二页更长编号统一增宽。原 P11 identifier-width 的目录 mock 随 T2 拆模块迁到真实新路径，保留视口外/子项展开/宿主重置原断言；没有把 mock 到旧路径的失效测试当绿。

M06 续页丢 label_id、M07 计数改用已加载长度均触发行为红，精确恢复哈希再绿。最终聚焦10文件247测试（0 skip），类型、修改路径零 lint、关键词及 web cap137均通过；raw/输入哈希/退出码见 `t3-final-regression`、`t3-types`、`t3-changed-lint`、`t3-lint-cap`、`t3-keywords` 和 M06/M07 receipts。生产修复及重试协同在既有 binding 职责内，自查无新架构决定。

过程日志可见既有 DateDropdown 嵌套按钮，以及 WorkItemFilterRoot 渲染时通知 FiltersRow；前者由已定 T8 修复，后者在 T13 真实列表筛选加载时追根核验，最终页面零意外 console 标准不放宽。当前聚焦通过不等于这些后续项已关闭。

## 11. T4 快速添加、字段反馈和宿主（2026-10-30）

四布局共用 QuickAddNameField，Field/FieldControl/FieldError 保留原样式并提供名称、无效状态及错误描述；空白名称本地拒绝，422字段错误用既有文案表，403走统一拒绝提示。RHF 默认抢焦关闭，useRefusedFocus 在错误提交后取焦；输入法 Enter 由既有 isComposing 判断。首次打开和确认保存后对已挂载且已解除 pending 的实际输入框取焦，实测修复原 RHF ref 注册/reset 窗口。

调用复用 DetailScope、根权限水位和会话检查。项目/权限/分组/外部关闭使旧owner结束，晚答不弹旧toast、不重置或解锁新表单；保留重复提交的同步 busy 所有者。显式 scopeKey 区分相同预填载荷的不同组；列表/看板默认状态按 route projectId 读取，新增 route B/store A 的真实 POST 断言。列表和看板的添加入口改原生按钮，实际打开后焦点落输入框。状态/优先级/标签/负责人/none 的原生预填继续由原群组回归证明。

现有 root.pending 改为提供真实根权限事实，保留原 HTTP/重复提交/B pending 的断言，并把旧A成功toast改为安静；新 root.errors 通过真实 root writer 与 FakeNerve 覆盖空白、422、403、成功、IME、关闭/换组/权限ABA。M08错误前抢焦、M09丢宿主检查、M10去busy锁在最终源码上均实际红→精确哈希恢复→绿（`M08-final`/`M09-final`/`M10-final` receipts）；M10 明确观测到2个POST，而非只等到超时。早期引入 scopeKey 前同载荷换组仍弹toast的红、默认状态错项目及打开未取焦的红也保留。

最终聚焦10文件99测试（0 skip）通过，类型、格式、修改路径零 lint 和关键词通过，web cap137→135按实际收紧。证据为 `t4-final-regression2`、`t4-types-final2`、`t4-changed-lint-final`、`t4-format-final`、`t4-lint-final`、`t4-keywords`。第一次final聚焦揭露首次打开取焦缺口，已修复后再跑，旧失败原件保留；初次 FieldError 缺 size 的类型错误及 jsdom 缺 IntersectionObserver 只记夹具/编译过程。无未决架构问题；真实页面、三浏览器和首轮完整 F 留 T5 及后续检查点。

## 12. T5 看板、泳道和拖放（2026-10-30）

列表/看板真实卡片统一通过现有 closest-edge 产生上/下落点，指示线与请求方向一致；移除源后求邻居，中间 below 前进一位，初始目标失效不发写，列尾和空列仍由根排序协调处理。多标签/负责人保留其它成员，none 合同保持。非手工排序只允许现有跨组移动，不发送同组手工重排。

实际 hook 复用 DetailScope、会话与根权限水位，并核捕获的列表 binding；当前拒绝按码本地化，关闭/换项目/换 binding/权限 ABA 的晚答无旧 toast。确认之后才安排高亮，useDropHighlight 持有 timer 和全部动画帧，保留平滑滚动并在卸载、scope 或权限结束后停止。迁移中类型检查发现侧栏也是高亮工具的真实调用方，已同步 hook 用法，不改变项目拖动合同；日历本次仅接取消能力和原生移动端日期按钮，日期确认业务留 T7。

普通看板与泳道均有原生续页按钮，pending 禁用；错误时取消自动续页，保留显式重试。泳道头仍读取 subgroup_totals，改为可键盘折叠的按钮；隐藏行返回 null，避免缺 key 警告。真实多标签卡片显示两处但 canonical 只有一份，独立 cell cursor、失败重试与权威总数均验证。

M11（中间 below 算成 above）实际发送 2500 而非 3500；M12（相加格计数）把泳道 4 变成 6；M13（写前乐观移动）在 PATCH 等待时移走源卡片；M14（旧读跳过确认回放）把成功尾部卡片放回源组。四项均行为红、精确恢复原字节/hash 后绿，见 M11–M14 receipts 与 t5-Mxx-red/restored。旧读/补读 503/只读重试的真实 DOM 和请求断言也通过。

完整本地 F 已串行通过：gen-check、lint-go、GOFLAGS=-p=1 make test、lint-web、knip、TURBO_FORCE=true make test-web、make e2e；证据为 t5-F1-* 的 raw、argv、输入哈希与退出码。lint-web 54 项通过，test-web 18 项均强制执行无缓存，默认 E2E 169/169；这不代替 T15 的 C1 三浏览器故事。web lint cap135→133。最终聚焦11文件189测试（0 skip）通过，见 t5-final-regression；修改路径零 lint、全仓类型/格式/关键词已含 F。

自查核对了所有高亮消费者、原生落点生产者、真实 hook/根写入口、确认前后计数、同组/跨组/尾部/空列、读拒绝、独立分页与类型合同。新集成测试按拖放/泳道分文件，并共用 fake-kanban，避免复制完整假 transport。初期夹具 API 拼写/参数/数据外壳和拆文件时的 mock 导入顺序失败如实保留，不算行为红。无未决架构问题。T11 另需实测路由卸载中的全局 dragging 标志是否残留，并在原拖动层职责内处理；当前还未将这一检查宣称通过。


## 13. T6 表格确认排序、续页和编号链接（2026-10-30）

表格各排序列使用生成 IssueListOrder 的原生字段，菜单选中、箭头和 th aria-sort 统一从已确认 displayFilters 投影，移除两份 localStorage 排序状态。保留现有筛选控件的显示词汇，仅通过 projectListQuery 投影原生值；未新增 API 或依赖。修正 metadata 的图标映射及 iconKey 泄漏到 SVG 的 React 警告。

实际表格 DOM / FakeNerve 验证：PATCH 等待与 422 拒绝均保留确认箭头、偏好和行序；确认后才以新 order 发读取。同账号切项目和重挂载重新读取各自偏好，不串箭头。续页保留 order/cursor/limit、重复点击只一请求、pending 原生禁用；503 后停止自动续页，显式重试原请求，重叠页去重并在 null cursor 停止。首屏读取失败的重试入口移出加载骨架，空成功页无虚假续页。

子项列使用项目目录 identifier 和 sequence_id 的 browse 链接，包括归档工作项；没有旧 ID 地址/#sub-issues，冷目录、无效编号、零子项不造链接。共享表格容器 ref 修正为实际 div。表格 skeleton 使用固定槽位 key。T11 接续核修 issue-row 原有四条菜单键盘/静态元素警告，本 Task 该文件仅一处 ref 类型同步，无规则豁免；其它行为修改路径零 lint。web cap133→132。

M15 原生排序退回旧字段、M16 全局 storage 覆盖确认排序、M17 链接追加旧片段均实际断言红，原字节/hash 恢复后绿，见 M15–M17 receipts。最终聚焦 9 文件 161 测试（0 skip）通过，t6-final-regression；共享包强制构建 t6-shared-build2、类型 t6-types-final、关键词 t6-keywords、lint t6-lint-final / t6-changed-lint-final、格式 t6-format 与 diff-check 均通过。首轮 shared build 命令目录写错、测试查询了旧 preferences 路径/错误 DOM 文本、路由夹具中间提交及分页 fixture 顺序问题的原始失败保留，不计行为红。四种排序实际二页与三浏览器留 T14/T15，阶段验收仍开放。

自查核对所有 metadata 消费者、原生请求字段、确认状态来源、空组 loader 与首屏错误、真实目录链接及已有 filters 回归；无未决架构问题。T5 检查点 c7acea58 的远端 server/web 已通过，E2E 仍在运行，原始状态保存在忽略证据目录。


## 14. T7 日历确认日期和添加已有（2026-10-30）

真实月/周布局在 TZ=Pacific/Kiritimati（UTC+14）跨 2026/2027 年界验证当地日、日期范围和普通筛选交集，旧月成功/失败晚答均不能覆盖新月或弹旧错误。月/周范围沿用已有 store 与生成日期工具，未新增另一套日期查询。上一/下一时间段按钮补双语可访问名称。

日期拖动仍仅发 target_date：确认前原日期/计数/卡片不变，也不高亮；成功后更新目的日和计数，同日不重复写，403/422 保留原日并按码提示。根宿主复用 DetailScope/会话/当前 binding，切范围后的晚答不执行反馈；高亮由已存在的宿主 hook 在确认后安排。早于 start_date 的本地限制直接比较日期字符串，使用双语明确文案。图表显式接成员编辑权限；访客无快建/添加已有和写请求，基线已有快建保护，本次只是显式贯通只读参数（t7-guest-red 命名保留，但实际为通过，不计行为红）。

日历当天入口、真实 Modal/搜索/写/列表消费者组合覆盖：空词和输入均保留 project_id/target_date:none，有截止日期或开始日期晚于目标日的项不可选；跨词已选去重，两个 PATCH 一成功一失败后仅留下失败项，重试只发它，确认后卡片在目标日且全成功才关。关闭旧日期选择器再打开新日期，旧拒绝不关新窗/不弹错；新窗提交使用新日。T1 的部分成功处理直接复用，无第二套选择或重试缓存。

另修两个真实页面缺口：移动端当天列表重复挂载已删除；自动滚动旧 effect 读 ref.current 在加载骨架卸载后的首次容器挂载漏注册，改为跟随实际容器值并在卸载释放。对应 DOM 行数和实际 adapter 注册/清理断言先红后绿。M18 把本地日改 UTC 导致 31 日显示 1 日；M19 丢 none 搜索条件；M20 缓存首次选择导致部分重试发第 4 次 PATCH；三项有效行为红、原字节/hash 恢复绿，原始材料为 M18–M20 receipts。

最终 10 文件 197 测试（0 skip）通过，t7-final-regression2；共享强制构建 t7-shared-build、类型 t7-types-final2、关键词/i18n-keys/sync、修改路径零 lint、格式与全仓 lint 通过，web cap132→130。过程中的拆夹具导入/未用符号、同时间 ID 降序预期写反和首次 autoscroll 测试未导入计数函数均单记，不算行为红。DateDropdown 嵌套按钮与 WorkItemFilterRoot 渲染通知仍按 T8/T13 根因处理，最终不能忽略。

自查核日期来源、根 PATCH 入口、拒绝/晚答、唯一移动端容器、adapter 生命周期、selector 当前日期及重试集合，未见架构阻塞。T5 检查点 c7acea58 的准确远端 CI [41025097454](https://github.com/open-nerve/NerveProject/actions/runs/41025097454) attempt1 的 server/web/e2e 均成功；回执为 t5-ci-result/jobs.json。T6/T7 尚未作为新产品检查点推送，下一次完整 F 与推送按 T10。


## 15. T8 单状态日期下拉（2026-10-30）

DateDropdown 复用已锁定的 @nerve/propel Popover，一份受控状态承接 defaultOpen、选择关闭、Escape、离开焦点和外部点击；删除 useDropdown 与 ComboDropDown 的双状态。原生触发器始终挂载，Calendar 随面板挂载，因此 renderByDefault=false 也能由键盘打开；DropdownButton 使用已有 presentation 模式，清除改成有名称的兄弟按钮。保留 min/max、disabled、closeOnSelect、格式/tooltip 和 onClose；IME Escape 不关闭。新增日期/清除双语名称。

10 项实际 DOM 测试覆盖 Enter/Space、选择后重开、Escape 还焦、组合键、范围、disabled、外点/焦点离开、相邻日历互斥、真实 ModalCore 背景隔离和清除。实际 CreateApiTokenForm 的 CustomSelect→DateDropdown→提交验证今天不可选、明天可选以及 expired_at 保留当前本地时间；未把 mock 表单测试当共享兼容证据。真实 Tab、定位及三浏览器仍由 T15 证明。

基线 t8-date-red2 中 6 项行为/DOM 失败已修复；M21 引入独立面板状态使选后仍可见，M22 恢复 pointer-only 准备使键盘无法打开，均实际红，恢复原字节/hash 后绿。首轮新实现两个失败为 jsdom 未模拟按钮点击获焦及动画帧卸载，夹具按真实原生事件补齐，原日志保留。

最终聚焦 t8-final-regression、共享强制构建 t8-shared-build、类型 t8-types、修改路径零 lint、web cap130、关键词/i18n keys/sync 与 diff-check 通过。DateDropdown 嵌套按钮警告已消失；WorkItemFilterRoot 渲染通知仍在 T13 清单，未视为最终页面豁免。自查核所有原 public props、单一状态、真实 token 消费者与模态隔离，无架构阻塞。


## 16. T9 基础属性下拉（2026-10-30）

优先级、项目、状态复用已有 CustomSearchSelect；Headless Popover 是唯一可见开关，内部静态 Combobox 只负责搜索/选项，popper ref/styles/attributes 在实际 Panel，列表保持 modal=false，不重写该共享原语。项目的嵌套按钮、缺失目录的空白候选及多图标 key 已处理；状态控件不再用另一个懒加载按钮替换原触发器，Escape 始终还给原节点。已选数组、renderCondition、禁用及 onDropdownOpen/关闭回调保持，具名原生按钮承担 tabIndex。旧项目测试由静态嵌套按钮断言迁成真实 DOM 的单一 trigger/根无 Tab 点，保留调用者 tabIndex 合同。

DateRange 改成与单日期相同的既有 Popover/Calendar，不再将网格伪装成 Listbox。显示完全来自 value，选择或清除在请求拒绝时均保留旧值；原子 PATCH 两日期的真实 row/child 消费者回归仍绿。本地与 body portal 选择保持；默认打开时等待本地容器就绪才挂载，修复首次从 body 迁移导致焦点丢失/立即关闭。min/max、defaultOpen、disabled、占位/合并显示及键盘合同保留。

新13项 DOM覆盖三控件无悬停键盘打开、搜索焦点/过滤、全浮层定位、IME Escape/实际 Escape、关闭还焦及重开/选择、ModalCore外层背景隔离、禁用；项目多选保持数组并排除冷/过滤项；日期范围的确认显示和两种 portal。既有 detail-refusals 的当前目录拒绝、只读重试与权限 ABA 同时通过。t9-basics-red2 的6失败已修复；项目多选初次测试缺少 Headless 实际监听的 mousedown 属夹具问题，已补真实事件序列，不算产品行为红。

M23 将共享定位属性移到子节点，使实际列表失去定位；M24 将内层列表改为 modal=true，输入搜索后外层背景 aria-hidden 丢失。两项实际红，精确恢复源字节/hash、重建 UI 包后绿，见 M23/M24 receipts 及 t9-Mxx-build/red/restored；共享产品源码最终无改动。真实浏览器定位/Tab 仍留 T15。

最终 t9-final-regression 通过；类型 t9-types-final2、修改路径零 lint、格式、关键词和 i18n keys 通过，web cap130→128按实际收紧。初次测试 fixture 的 stateOf 参数、缺失翻译键和测试 lint 均已修正，保留原日志。局部复用方案已同步 D15 与设计§7.5，不改变公共接口、权限所有者或 M6/M7 协议；无架构阻塞。


## 17. T10 标签选择器和确认反馈（2026-10-30）

三个标签入口复用既有 select/base：Popover 维护唯一打开状态，静态多选 Combobox 保留搜索/选项键盘语义，实际 Panel 持定位 ref，内部 modal=false 保留外层 ModalCore 隔离。原生具名按钮是唯一 Tab 停留点，名称包含可用的当前工作项标题；表格、行、创建/收集箱表单和草稿调用同步透传，不新建数据协议。分组搜索保留命中子标签的父标题，普通选择不会误触发新建；原只读、pending、颜色/集合与统一 useLabelChanges.create 合同保持。

选择器的局部打开 owner 约束完成回调：关闭再开，旧选择/创建不能附加标签、清新查询或弹提示。当前行标签拒绝复用 useDetailScope/useRefusalToast 可见一次；详情写仍由既有 useDetailCommand 报错，避免双报。详情创建成功提示从无宿主的 root create 回调移到当前 picker 的 onCreated，已确认创建而关联拒绝仍保留已建标签和原拒绝。未改变 IssueRootStore 权限归属或服务接口。

新增 DOM 与既有拒绝测试覆盖三个真实入口的键盘、焦点、定位、ModalCore、工作项名称、关闭/重开的创建晚答和行替换晚拒绝；旧用例仅将 portal 查询范围迁到 document，未放宽断言。共享 fake-issue-properties 按两份测试实际复用拆出。t10-controls-red、t10-row-refusal-red 与 t10-closed-toast-red 的有效行为失败均已修复；computedFn 的 map 回调额外参数、空数组类型和测试 Mock 声明等实现/夹具错误另保留，不当行为变异。

M25 漏只读 disabled、M26 绕过统一 create 导致字段拒绝丢失、M27 恢复根无名 Tab 点，均被真实断言捕获，生产字节/hash 精确恢复后绿。最终 t10-final-regression 为 11 文件 160 测试（0 skip）；类型 t10-types-final2、修改路径零 lint t10-changed-lint-final、全仓 lint cap128、i18n keys/关键词及格式均通过。完整 F 首轮 knip 发现 use-dropdown/use-dropdown-key-down 两个零消费者文件，已删除，后续 knip 为零。A11 旧故事把日历当 listbox 的定位已迁为具名 dialog，保留同一定位边界、背景隔离及关闭断言；实测发现新 Popover 的默认 8px 间距不同于原 4px，DateDropdown 显式保留 sideOffset=4，未放宽几何断言。两次失败的原始截图/trace 已保留。完整本地七项最终通过：生成/Go lint/Go test 为 t10-F1 已通过且相关源码未变；web lint/knip/强制 test-web/E2E 为 t10-F4，54 项静态任务、18 项测试任务无缓存、169/169 E2E（0 skip）；列表侧浏览器矩阵仍由 T15 验证。

自查沿实际三个调用方、受控值/拒绝/关闭晚答、创建 hook 和 label root 逐条核对，当前无未决架构问题。P14 的完整详情正 tabIndex/哨兵审计仍开放，本 Task 没有据局部覆盖宣称全部关闭。


## 18. T11 菜单键盘与当前拖动宿主（2026-10-30）

沿三个实际菜单和工厂核对：两个 SubMenu 分支没有任何 nestedMenuItems 生产者，历史 copy-menu-helper 只原样返回 baseItem，因而删除两个不可达分支及 shared SubMenu、右键嵌套死链和无效 layout/duplicate 状态；没有新增子菜单或宣称其浏览器验收。D17/plan 同步实际范围，M28 改核真实菜单键盘入口。平铺菜单仍用 Headless Menu，整个 Items 接定位 ref，IME 不冒发 Enter/Escape；详情按钮用既有 customButtonClassName，删除嵌套 IconButton，并为三个入口提供当前工作项的可访问名称。

右键菜单保持关闭 inert 和 aria-hidden 的测量 DOM，打开后焦点进入首个可用按钮；方向键/Home/End 只作用于当前菜单，Enter/Space 由原生按钮执行，IME 不执行，Escape 归还当前仍存在的触发位置。移除监听 window 的全局 Enter/方向键处理，避免焦点在页面别处时误操作。列表行原本为响应式布局重复挂载两个完整 QuickActions，实测同一右键拥有两份菜单；现改一份菜单以 CSS 定位，真实行用例确认动作一次。

看板、日历、表格的菜单激活与可见样式直接跟随 aria-expanded/focus-within，删除子 div 的独立点击状态；键盘不再被 display:none 挡住。表格使用真实 Space、isComposing、实际表格行/单元格 Element，空白文本节点不被强转为可聚焦元素；只点击可用原生控制。看板拖动 beginDrag 返回当前 owner 的释放函数，源卸载/失效释放自己的删除区状态，旧 drop 不清新 drag。pending、日历 Popover/quick-add、卡片/删除区层级降至模态层下。

31 项首轮聚焦通过，后加真实列表双菜单红→绿。M28 阻断键盘、M29 漏 inert、M30 恢复嵌套 button 均有效失败，精确恢复生产字节/hash 且共享包重建后通过。相关材料为 t11-*、M28–M30 receipt。最初表格夹具缺 scrollIntoView、卡片夹具缺 ResizeObserver、测试 issueDetailOf 参数写错及路由 stub 缺符号均单独保留，不算行为红。最终受影响回归曾与 make e2e 的共享包构建重叠，导致 8 个 suite 无法解析被构建清理中的 dist；随后 lint-web 隐式执行共享包构建，又造成 10 个 suite 无法解析 dist。这两次是执行编排失败；已确认所有 turbo 命令与源码消费者必须串行，后续串行回归 13 文件 129 测试（0 skip）通过，不算产品行为失败。

修改路径零 lint 已检查，web cap128→121、ui cap12→10；类型修正为 Menu.Items 实际 HTMLElement，未增加断言或放宽规则。最终 t11-final-regression3 为 13 文件 129 测试，t11-ui-tests 为 3 文件 22 测试；全仓 lint、knip 和类型通过。完整浏览器首轮 167/169：P2 仍按 button 查找已迁为 menuitem 的右键项，已同步角色定位；S8 的角色查询读取到关闭但仍保留测量 DOM 的菜单，已给根补显式 aria-hidden 并补 DOM 断言，保留 inert 和 S8 原断言。失败 trace 已归档；t11-browser-regression2 全部 169 项通过（0 skip）。C1/C3 三浏览器仍归 T15。T10 精确提交 c65a57e8 的 CI 41036854895（attempt 1）server/web/e2e 均成功。


## 19. T12 Peek 宿主与焦点归属（2026-10-30）

既有 Peek UI owner 增加实际宿主身份；第二个仍挂载的展示宿主不能接管或清除当前选择，失权/退役/路由改变同步停止旧内容。读取子树按每次选择重建，同 ID 重开也结束旧请求回调与本地弹窗载荷，权限事实仍取自 IssueRootStore。WorkItemLevelModals provider 保留原位；清理只操作自己仍持有的选择，StrictMode 重获后旧 microtask 无效。

焦点不再由全局 issue-ID 搜索：列表、表格、看板、日历、父项、子项、关联实际入口将触发元素交 UI selection WeakMap，store 的业务选择无 DOM。Escape/明确关闭仅在当前宿主与选择下恢复仍连接且非 inert/aria-hidden 的原元素；外部点击不抢焦，子控件已经 preventDefault 的 Escape 不再关闭 Peek。延迟外部点击 timer 随监听所有者清理，换项/卸载不执行旧关闭。清掉无生产者的 duplicate modal props/state，并精确声明页头图标类型。

t12-host-red 实际 7 失败，修复后相关 10 文件 94 测试通过（t12-final-regression2，0 skip）；包括真实卡片记录触发器、双宿主、同 ID 重开、路由同步隐藏、权限失去/恢复、访客持续读取及权限 ABA 的旧错误清除、晚成功/拒绝不覆盖新项、删除/归档/恢复及焦点归还。类型、修改路径零 lint 和 cap121 通过。最初测试循环改成 it.each 时误删 finally 括号属夹具语法错误，已修复；完整成功夹具中的后续资源读取不适合单次 core 请求计数，权限 ABA 测试改为受控 core 拒绝，直接检验每次权限证明变化后的新请求和旧错误清除；保留诊断 raw，未据失败日志宣称产品已坏。

M31 首轮未失败揭示测试 StrictMode 在子树而非根，未真正执行重复挂载；现将 StrictMode 包裹渲染根，M31 去旧 owner 身份校验被实际捕获。M32 去触发器连接校验被卸载焦点用例捕获。两项 receipt2 记录变异红、原字节/hash 恢复及绿；首轮存活 receipt 不计有效性质。当前所有者/读取/焦点责任均属既有 UI 层局部修正，无架构阻塞。实际 Back/三浏览器键盘继续在 T13–T15 页面故事核验。


## 20. T13 I3/I7 真实页面故事（2026-10-30）

保留原 API 故事；新 issue-list-page 夹具由 I3/I7 两个真实页面复用，API 建独立账户/项目，真实表单登录，导航前注册 watchPage。I3 使用 53 个真实工作项，状态首 50 条与后页、A 筛选 C 标签组的 51 条交集、完整 A/C 已选标签、切看板并重载后的唯一泳道总数均核 UI 和实际 query。标签逐个 chip 都能打开完整选择器，页面明确点击 A chip，不以任意 first() 掩盖定位。I7 在指定状态真实快速创建，字段描述先于拒绝取焦，422 留输入、pending 不重发，释放实际响应后清输入；DB 状态/尾部位置/创建活动一致。

页面准备定位到 WorkItemFilterRoot 在 render 中更新 MobX 注册项，导致子组件渲染通知；StrictMode 重挂时又删除不重建。局部修复将注册/选项更新置于 layout effect，按 store/entity/key 持有实例，清理仅删除自己的实例，切 session store 同 key 也不暴露旧项。三项真实 store 测试覆盖原通知、StrictMode、实体/会话替换及卸载；不改变筛选协议或权限归属。D19 登记此根因处理。

M06-page 续页丢 label_id 使 C-only 错入 A∩C；M08-page 先取焦使记录缺少错误描述；二者均实际页面红、精确恢复 hash、当前源码重新 build 后绿，作为既有 M06/M08 的额外层级证据，不重复计独立性质。M33 恢复原 filter HOC 触发渲染警告与 StrictMode 缺注册，精确恢复后绿。早期焦点探针被 Playwright press 预先聚焦、held route 阻断低层 route 观察及双 chip 定位歧义属夹具问题，分别改用失焦后的原生表单提交、真实 request 事件计数、指名 chip；原失败 trace 均保留。

最终 t13-final-unit 为 6 文件 61 测试（0 skip），当前 make build、web/E2E 类型与修改路径零 lint 通过。t13-final-pages2 四项全绿（2 API + 2 Chromium page，0 skip）；每次实际加载的已知 EMOJI 警告精确计数，I7 仅明确期待一次 422，其他 API/旧地址/console/CSP/外域均为零。Firefox/WebKit 的文本与次数留 T15 实跑确认。证据 t13-*、M06-page/M08-page/M33 receipt。局部自查通过，无架构阻塞。


## 21. T14 表格、日历、看板与 S2（2026-10-30）

I4 保留两条 API 故事，新增表格和 UTC+14 日历页面。表格 101 个父项、真实目录与子项，标签名/负责人名/子项数/截止日期四种正反方向均由实际表头选择；每方向核确认 aria-sort，滚动分批收集全部 DOM 行顺序、唯一真实第二页 query 与末页，不排序服务端响应制造预期。预期独立取自建数元数据、同方向 UUID 和空值后置。I4 文件共 513 行：保留 240 行既有 API 及两个完整页面故事，按同一验收项集中，未新造只用一次的 fixture 拆散因果。

日历 context 明确 Pacific/Kiritimati，不修改业务/认证时钟；通过真实月份选择到当年十二月，核跨年显示范围、单日 4+1 续页和当前周 limit30。真实指针把 16 日移到 17 日，只有 target_date PATCH，确认前原位、确认后 DOM 与一条 DB 日期活动一致。添加已有空词请求准确 target_date=none，已定日期及起始日晚于目标日项不可选；选中后通过另一真实 API 修改开始日期，使实际后端返回 422，弹窗/选择保留，恢复合法开始日期后只重试该项，确认后出现。

I6 使用真实角色15成员，Todo 放到 Done 两卡之间，保持确认前原位置/计数；一次 PATCH 同时改状态与位置，DB completed_at/updated_by/唯一状态活动一致。同列中项下方只改 sort_order=250，尾部按既有排序合同为300+65535，403 拒绝有可见原因并原位，刷新后确认顺序稳定。S2 管理员/成员/访客在项目页真实切四布局，非成员保持加入界面；每次新增请求按具体路径、query、次数白名单核验，没有领域空接口豁免。

复用 dragOnto 时暴露测试辅助 transitionsEnded 把被替换/卸载的动画 AbortError 作为失败；仅将该取消视为生命周期已结束，其余错误继续抛出、无限动画仍排除。初次建数传 create 不接受的 target_date:null、同列不该重复 state 字段、尾部误用快速创建的10000间距、ModalCore 零尺寸外壳可见性、日历 query 参数顺序及 TS union 未收窄均为故事编写问题；已按既有接口/可见内容/closedWithin 改正，不修改产品迎合测试。原日志与 trace 保留，lint 的异步顺序以 reduce 保持真实用户操作顺序，无新豁免。

M11-page 的错误中项 below 导致实际请求 -65335 而非250；M18-page 的 UTC 日期偏移使所选日无法续页；M19-page 丢 none 条件被实际搜索 query 捕获。三项均真实红，精确恢复原字节/hash、重建后绿；只作为已有 M11/M18/M19 的页面证据，不重算独立变异。最终 t14-final-pages 全部16项通过（0 skip，3新 page + 保留 API/S2），t14-types-second、t14-lint-second 零错误/警告，diff-check 通过。全量 F 与三浏览器仍由 T15 执行；自查无架构阻塞。


## 22. T15 三浏览器与局部根因修复（2026-10-30）

专用配置以单对象展开默认配置后替换 projects，准确列出 Chromium/Firefox/WebKit 各11项，共33项、5个文件；没有继承默认 Firefox A4 的 grep。`t15-matrix-v2-chromium/firefox/webkit` 各11通过、0 skip，当前源码已重建；`t15-matrix-final-list` 的配置/故事输入与本轮相同。安装仅使用仓库锁定 Playwright 的本地 WebKit 2359，无全局安装。

| 实际入口/断言 | Chromium | Firefox | WebKit | 证据与边界 |
|---|---|---|---|---|
| I3 状态续页、A∩C、多标签泳道与保存布局 | 通过 | 通过 | 通过 | 同一真实API/DOM故事；布局请求 finished 后才 reload，未忽略取消请求 |
| I4 表格四种双向排序、真实第二页；UTC+14日历范围/续页/拖动/添加已有/拒绝 | 通过 | 通过 | 通过 | 两(page)故事，真实请求及DB；拖动前确实打开预览，开始拖动及放下后均关闭 |
| I6 成员跨列/中项below/尾部/确认/403回位 | 通过 | 通过 | 通过 | 原生拖放、确认顺序及活动；没有使用store调用冒充页面 |
| I7 快建字段先呈现再取焦、422/pending/双提交/状态尾部 | 通过 | 通过 | 通过 | 实际表单与DB；原API故事保留 |
| DateDropdown 与 DateRange：键盘、相邻控件、Escape焦点、完整弹层定位 | 通过 | 通过 | 通过 | 列表行；日期范围4px，外点关闭不抢新控件焦点 |
| priority/state/label-dropdown/member 的搜索、定位、当前403一次提示 | 通过 | 通过 | 通过 | 实际列表行，不以DOM单测代替几何；状态刷新保留标签宽度 |
| project/base 与 select/base | 通过 | 通过 | 通过 | 从项目列表打开创建窗，项目与标签搜索/完整Panel锚点/外层inert/关闭还焦；未提交创建，不提前认领P14完整创建 |
| detail label-select | 通过 | 通过 | 通过 | 列表peek实际标签面板；完整详情验收仍P14 |
| 菜单及关闭的右键菜单 | 通过 | 通过 | 通过 | 真实行唯一菜单、键盘箭头/Escape、右击/IME/焦点归还、关闭inert及aria-hidden |
| peek CustomSelect / 两类关闭 | 通过 | 通过 | 通过 | 当前模式真实选中、IME Enter不打开、切Modal；加载中原节点还焦/晚读不重开；完整读取退役节点不抢焦 |
| 自动归档、Created date筛选、API token自定义日期 | 通过 | 通过 | 通过 | 已接通页面的共享兼容；真实定位/焦点/外层模态隔离，等待外层动画结束再量定位，未取消动画/放宽阈值 |

三浏览器各自精确断言console：Chromium每次实际编辑器加载一条既有EMOJI_CHECK_WARNING，Firefox/WebKit没有该提示。主动403/422在Chromium/WebKit各有一条对应状态的固定resource error，Firefox没有；无其它console/API失败/pageerror/CSP/外域/旧地址豁免。Firefox曾出现的滚动警告由样式写入探针定位到日历hover preview的跟随定位，已通过拖动时关闭修复，不列入白名单。

T15局部问题及首红：`t15-inline-ime-red`/`t15-controls-red`复现行/表格未承接拒绝、IME误开、成员触发无名/替换和状态锚点宽度变化；`t15-c3-local-fixes-calendar-red`复现真实拖动预览未关闭；`t15-loading-close-red`复现加载态关闭按钮无名；`t15-two-calendars-focus-red`两项复现前一日期控件抢焦；`t15-context-return-red`复现右键返回其它控件。对应修复复用现有原语、命令与scope，无新公开协议或架构责任。最初jsdom hover未真正打开预览、配置合并导致0用例、缺WebKit、错误的测试名称/动画中测量均单列为测试/环境问题，不计产品行为红。

完整peek的初次关闭断言错误地寻找卸载后新建的同ID行。DOM removal探针与网络trace证实子项读取的父项核验会卸载列表；按spec§2.7保留核验与退役保护，分成加载中仍存/完成后退役两条页面证据，不新增全局ID回退。代价是后者交由页面正常焦点管理。成员其余创建/详情/关系/子项/草稿入口的完整页面、富筛选四入口、C12全应用三浏览器仍按原Phase归属。

局部最终验证：`t15-local-regression-final` 11文件117测试；`t15-context-return-green` 3文件42测试；`t15-context-ui-green` 3文件22测试；均0 skip。共享强制构建10任务、web/E2E类型与151修改路径oxlint零警告通过；web cap121、ui cap10均按实际保持，没有提高。完整F已通过：生成/Go lint/Go test为t15-F1，web lint/test/E2E为t15-F2；54静态任务、18强制测试任务零缓存，实际253文件2678测试（0 skip），默认E2E180/180（0 skip）。首轮knip报专用配置无消费者，已在e2e/package.json加入test:m4-list脚本；t15-knip-config-consumer零dead，t15-script-matrix-list仍准确33项，未扩大默认CI矩阵。t15-F-audit逐文件核对：服务端与矩阵仅晚加测试脚本，其余产品输入未变。33种历史变异均实际红、原字节恢复且绿，其中26种生产文件哈希仍与当前一致；其余明确列为历史证明，不能冒充最终字节的变异，详见t15-mutation-audit。检查点 `71e1863e` 的 [CI 41054578922](https://github.com/open-nerve/NerveProject/actions/runs/41054578922) attempt 1，server/web/e2e 均成功，回执 `t15-ci-jobs-final.json`。

## 23. T16 整审发现与一次修复（2026-10-30）

独立新上下文审查范围为 `ce3c39ab..71e1863e`，结论为 With fixes：Critical 0、Important 6、Minor 0；主控按逻辑、集成和测试三视角顺序复核。六项均有当前用户可触发的失败，不按文件数量机械判成跨模块设计问题：修改只在 issue UI 内恢复既有 owner/命令/读取合同，不改后端协议、权限事实、canonical 或架构守卫。逐项记录如下，最后交付结论另见整分支报告。

| 问题与根因 | 修复与边界 | 行为红 → 绿 |
|---|---|---|
| 列表/表格展开子项后，父项权威核验卸载行，行内展开意图丢失 | 布局拥有 RowExpansionProvider，跨加载骨架保留 UI 意图；沿既有 PermissionReads 的 identity 在换路由/权限/宿主时清空，不保留旧内容、不绕过核验 | project-lifecycle 的两布局实际读取/父项核验及权限失去恢复 |
| 列表首读失败后重试入口被加载骨架挡住 | IssueListRetry 放在同一布局的加载门外，重试仍只有 GET | 首次503→可达重试→确认列表 |
| 看板拖动删除吞下403、提前关窗，成功删除又使 dataId 失去当前命令身份 | 将 Promise 结果交回 DeleteIssueModal；仅持选中项的确认快照供该弹窗完成，不新增实体缓存 | 当前403留窗一次错误、204重试一次成功/关闭、双击一次DELETE、卸载晚答安静 |
| 实际 Restore 菜单把拒绝当成功、重复成功或在旧宿主提示 | helper 只承接事件 Promise；peek 的既有 operation 拥有反馈，拒绝继续传播；归档行用既有 useDetailCommand 拥有一次反馈 | 真实 view→header→menu→factory→operation 链的当前403/200及替换/卸载/权限ABA |
| 表格在 state updater 发子项 GET，StrictMode 重发且拒绝无入口 | 请求移到事件边界，承接拒绝；复用 DetailResourceError 的 loading/error/GET retry，展开按钮有名称/状态 | StrictMode一次GET、403可见、只读重试；不另造资源状态 |
| 看板自动滚动 effect 只在初次空 ref 注册 | 跟随实际容器节点注册，核验替换及卸载时释放旧节点 | 加载后注册、父项核验后重注册、宿主结束归零 |

有效红为 `t16-project-lifecycle-red5`（6项失败）与 `t16-restore-menu-red`（7项失败）；更早缺夹具/错误按钮名/权限角色等失败不计产品红。第一轮修复绿剩一项成功删除未关窗，追到 dataId 被 canonical 删除使命令身份消失，改为当前确认快照后 `t16-lifecycle-restore-green2` 13/13；补权限结束/卸载后 `t16-lifecycle-scope-green` 19/19。`t16-affected-regression` 实际10文件86测试（不存在的额外过滤路径不计执行文件）。最终全量 test-web 覆盖255文件2697测试、18任务零缓存、0 skip；修改路径 lint 零警告，完整 lint-web 54任务、knip、build、默认 E2E180/180均通过，见 `t16-F-*`/`t16-build`。服务端未改，最终交付按输入 hash 核准复用 T15 服务端三项。

最终矩阵的 Chromium/Firefox 各11/11通过；WebKit 第一次5通过/6失败，同时 Docker socket 消失、数据库连接被拒绝且 pg_dump 无法连接，原始证据为 `t16-matrix-webkit`。只读 `docker info` 确认环境不可用，已请负责人按“不动已有容器和 Docker”规则恢复服务。此轮不能计通过，也不以 T15 旧矩阵代替最终输入；保留原始证据待环境恢复重跑。文档收尾与最终准确 HEAD CI 仍在进行，当前不宣称 T16 完成。

## 24. A1–A12与写入结果的证据映射（2026-10-30）

以下按当前产品修复 `49b628fb` 核对。早期逐Task红/变异保留其发生时的输入，不把旧收据自动视作最终回归；最终全量前端、默认E2E、静态和两浏览器另跑且输入一致。A9/A11/A12因最终WebKit/文稿HEAD CI未闭合仍保留开放。

| 标准 | 可定位证据 | 当前判断 |
|---|---|---|
| A1 | issue-search.service/use-issue-search、ExistingIssuesListModal及三个调用方；t1-final-regression 149测试；旧实例/旧query检索 | 通过 |
| A2 | hook/resource/selector真实root、权限目录、当前拒绝、查询防抖窗、session/ABA/晚答；M02/M03与最终全量前端 | 通过 |
| A3 | spec§2.9原22路径14存在8已删；route-scope/group-consumers；M04/M05；关键词规则76/例外24、无新增宽泛豁免 | 通过 |
| A4 | list/view、swimlanes/list-data及I3页面，M06/M07历史及M06-page；T16首读重试/跨核验展开；当前Chromium/Firefox矩阵 | 通过；跨浏览器最终汇总另属A11 |
| A5 | quick-add root.errors/root.pending、I7实际字段描述/recordFocuses/请求/DB；M08–M10 | 通过 |
| A6 | kanban/drop与group-dragndrop/root拖动；I6一次PATCH、DB/DOM与中间below/列尾；M11–M14；T16删除/滚动生命周期 | 通过 |
| A7 | spreadsheet/view/columns/header-column、project/filter.store、I4四种排序双向真实二页；M15–M17；原排序localStorage已删除 | 通过 |
| A8 | calendar/view/selection、真实selector、UTC+14 I4、M18–M20；部分成功不重发确认项 | 通过 |
| A9 | date/basic-dropdowns/label-controls/menus、C3六条故事；api-tokens A11与settings selects；最终WebKit尚待恢复复验 | 待最终矩阵闭合 |
| A10 | detail-host/host-actions/selection/outside-click；M31/M32有效receipt2；真实C3加载中原节点/完整加载后退役，T16restore-menu | 通过 |
| A11 | 导航前watchPage；S2准确REQUESTS，默认180/180；T15矩阵33项绿是历史，T16当前Chromium/Firefox22项绿；WebKit失败不计通过 | 待WebKit复验 |
| A12 | 最终静态/knip、完整F输入核验、26项当前hash一致的独立性质、re-17六项修复；局部handoff已更新，最终文稿HEAD CI未核 | 待最终交付 |

下表按P12实际写入口分类。“成功补读失败”有些由共享确认/读取合同组合证明，没有为每个UI入口各跑同名端到端故事；不把这种组合证据写成每一入口都经历过一次页面500。

| 写入口（生成协议） | 确认 | 拒绝 | 旧scope/宿主 | 成功后的读取失败 |
|---|---|---|---|---|
| 快速添加POST issue | root.pending、project/issue.quick-create、I7：确认前输入和列表不变，确认后尾部/唯一动态 | root.errors/I7：422描述先于焦点、留输入、防重复 | root.pending/route-scope：换group/route/权限后旧答不释放B的pending或toast | issue.quick-create及root确认协调保留确认；list-data/refresh证明读取失败只GET；新项没有已打开的活动资源，不虚构此类补读 |
| 行/表格属性PATCH（含双日期/标签集合） | issue-properties、date-range-write及inline-refusals验证原生单次写/确认值 | inline-refusals实际三个消费者403，一次原码反馈、旧值保留；标签选择器自有反馈 | inline-refusals/detail-commands与权限ABA，旧宿主无提示或关闭 | issue-details/detail-collaboration的core命令：活动GET500仍成功、一次写；root authority/只读重试不回滚确认 |
| 看板状态/排序、日历日期PATCH | kanban/drop、calendar/view与I4/I6请求/DB/DOM；确认前不移动 | drop的403/422、calendar拒绝保留原位 | use-group-dragndrop/use-drop-highlight、日历scope/卸载，旧timer/scroll/反馈清理 | drop的older-read与refused-refresh实际验证503后只GET重试；canonical确认由issue-changes共享合同保证 |
| 表格排序偏好PATCH | spreadsheet/view实际菜单与project/filter.store，确认箭头和持久化值后刷新 | 两测试拒绝时保留偏好/箭头/行 | filter.store旧读/退役及view同账号换项目 | 组合证明：偏好已确认投影 + 独立列表读取错误/GET重试，未单独声称页面偏好成功后500故事；失败读不改确认偏好 |
| 日历已有项PATCH、子项/关联批量POST | selector与calendar/selection：confirmedIds/失败ID分开；detail-mutations使用原队列 | 原子批量拒绝保留全选；日历部分拒绝只留失败选择 | selector原scope证明、关闭/换词与换范围不同、权限ABA、detail-mutations旧队列 | detail-collaboration的subIssues/relation活动读取失败保持成功；日历已确认ID不在失败集合，不再PATCH |
| 新标签POST→附加标签PATCH | label-controls/issue-properties与use-label-changes统一create；受控值确认后变 | 重名字段错误、附加拒绝保留已创建标签 | T10已关闭/重新打开picker的旧创建不能附加/提示，权限/宿主归现有scope | 创建确认与附加命令分开；后续目录/活动读取失败走既有资源重试，不把已建标签当POST失败重发 |
| DELETE、archive/unarchive POST | issue-changes六种核心确认矩阵；detail-commands；T16真实删除/Restore一次反馈 | T16实际403删除留窗/Restore仍归档，无假成功；重试单次命令 | project-lifecycle/restore-menu及detail-commands换项/卸载/ABA不跟进旧UI | 核心确认投影不因列表补读失败回滚；删除后对象退役不再补读它的活动。布局重试仅GET，Restore helper不增加第二次写 |

C3逐控件落点：DateDropdown与DateRange、priority/state、member由实际列表行；project与通用label-select由创建窗；行label-dropdown由实际标签chip，详情label/select由真实peek；七浮层几何与外层pageBehindHidden、Escape/Tab/焦点均在 `list-interactions.spec.ts`。CustomSelect的peek当前模式、自动归档自定义范围、日期筛选与api-token期限兼容由同文件真实入口核验；api-token日历同时保留A11。该矩阵不等于完整创建、富筛选或M6/M7领域页面验收。

## 25. 最终输入与剩余门禁（2026-10-30）

`t16-input-audit1.json` 对照所有git输入路径的集合及SHA-256，而非只比较交集：T16静态/knip/test-web/build/默认E2E/Chromium/Firefox与当前产品字节一致；T15服务端三项后仅12个web/e2e路径变化，服务端输入不变可复用。33种有效历史变异中26种当前生产hash一致；M07/21/22/28/29/30/33后续改动仅保留历史性质证明，详见t16-mutation-audit。原件有首次失败、生产diff、恢复hash及恢复绿；无匹配、全skip、首轮存活M31、夹具编译失败不计有效性质。

修复提交 `49b628fb2807e3423f7d40ceeaa8f5f5a656cc8d` 已推送，CI [41057831211](https://github.com/open-nerve/NerveProject/actions/runs/41057831211) 的server/web已通过，e2e在核验中。最终文稿提交必须另核准确SHA，后置回执写忽略目录final-delivery.json以防自引用，不拿产品候选CI替代最终文稿CI。

WebKit最终第一次运行5通过/6失败；保留环境中断原件，不先断言所有页面失败都是同一环境原因。只读docker info确认socket不存在/服务不可连接，已请负责人恢复Docker；恢复后只对当前输入重跑该浏览器，若仍出现页面失败再按根因处理。此前通过的服务端/前端/默认E2E不因纯文稿重复重跑。

T16尚未完成，不合并、不清理worktree、不启动P13。M4§14/总体§9.4、输入handoff顶层open和最终M5–M8 closeout不提前勾选。完整执行裁定与审查范围取舍见[re-17§6](../../../reviews/re-17-M4P12整体审核.md)，没有deferred minors。
