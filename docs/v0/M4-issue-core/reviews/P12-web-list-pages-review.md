# M4/P12 列表页面：预检与实施记录

日期：2026-10-30。**当前：预检通过后已获授权实施，T1 已完成，T2–T16 待继续。** [spec](../specs/P12-web-list-pages.md)、[plan](../plans/P12-web-list-pages.md)、[deep-review预检](../../../reviews/re-16-M4P12预检审核.md)。本记录不是P12产品完成或阶段整分支终审；阶段 A1–A12 在最终整审时逐条收口；本次不合并。

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
