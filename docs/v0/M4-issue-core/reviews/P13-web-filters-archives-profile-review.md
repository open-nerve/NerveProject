# M4/P13 筛选、归档与个人主页：实施与评审记录

日期：2026-10-31。**T1–T14 实施与终审修复完成，集成待办。** 基线 `ff89d0a1219a8d09ff924f5bd5257c486cd7e2b8`，最终产品 `de616d06ca7befeb5a39f82a6c70ff18a3f03742`。以下 §1–7 是预检历史、§8 是实施历史，当前验收以 §9–14 为准。[spec](../specs/P13-web-filters-archives-profile.md)、[plan](../plans/P13-web-filters-archives-profile.md)、[整体审核](../../../reviews/re-19-M4P13整体审核.md)。最终文稿准确 CI 必须按 §14 后置核准后才交付；未经负责人安排不合并。

## 1. 前置与评审方式

P12最终e6d1329b授权--no-ff合并main ff89d0a1219a8d09ff924f5bd5257c486cd7e2b8；准确[main CI41087022719](https://github.com/open-nerve/NerveProject/actions/runs/41087022719)三任务成功。实施/运行1786文件及准备14文件归档、删除前后哈希复验通过；P12远端/本地分支和worktree清理完成，见[P12 review§28](P12-web-list-pages-review.md#28-授权合并main-ci与归档清理2026-10-31)。

P13从该main建立codex/m4-p13、.codex/worktress/codex/m4-p13，主项目目录始终main。create_worktree原生工具不能指定负责人规定的路径/分支，按using-git-worktrees使用系统git worktree add。没有安装依赖或启动产品实现，phase仍M4/P12。

按02§3调整方法：本次用既有源码/生成合同/测试静态核对，替代原型和逐Task重放；使用brainstorming的独立builder-check及writing-plans。新审者p13_builder_precheck只读spec、设计P13/P11标签边界和相关源码，报告“nothing important left”，未修改或运行产品。主控依次完成deep-review三个视角，不称为三个独立审者。

准备根在主项目.superpowers/sdd/P13-preparation-20261031/，未来须与实施worktree材料一起归档。builder-check.md、source-review-inventory.json、consumer-inventory.json、precheck-static.json保留范围、哈希与实际静态结果；提交身份另写precheck-delivery.json，不在提交里循环引用自身SHA。

## 2. 验收映射

| 标准 | 静态核对与任务 | 产品状态 |
|---|---|---|
| A1 原生条件 | issue-filters真实转换/测试、页面allowedFilters、subscriber/project_id配置；T1/T10 | 未实施 |
| A2 四选择器 | 四消费者、loadOptions生命周期、CustomSearchSelect合同、七个M4 FiltersDropdown调用；T2/T4/T13 | 未实施 |
| A3 日期 | range及DateRangeDropdown、旧DateFilterModal和调用、开区间转换；T3/T13 | 未实施 |
| A4 项目偏好 | ProjectIssuesFilter每字段队列、确认投影、useProjectPreferencesFollow与真实HTTP测试；T4/T10 | 未实施 |
| A5 归档只读 | archived root/header/filter/issue、详情与命令权限、I15 API；T5/T7/T11 | 未实施 |
| A6 命令与owner | ArchiveIssueModal两调用、detail命令/P12 restore单反馈、归档layout/key；T6/T7/T11 | 未实施 |
| A7 typed view | route的in对象、两个布局根、IssueActions/useIssueListMount传递链；T8 | 未实施 |
| A8 三关系与访客 | profile布局桌面/手机门禁、store relation/query、I19 API授权/真实交集；T8/T9/T12 | 未实施 |
| A9 本地设置 | archived/profile helper作用域与测试、已持有labelMap/P11限制、project_id保留；T5/T9/T12 | 未实施 |
| A10 页面/浏览器 | 默认Firefox仅A4；明确新三浏览器配置、watchPage、C2/C9范围；T10–T13 | 未实施 |
| A11 验证门 | 精确聚焦argv、T4/T9/T13检查点F、M01–M22候选性质与至少15有效性质要求；T13/T14 | 未实施 |
| A12 整审/交接 | 完整基线..HEAD deep-review、修复复审、准确分支CI与对应子项；T14 | 本次仅预检，未来另做整审 |

## 3. 执行中的决定

本轮没有实施决定；准备裁定D1–D8见spec§3。重点是：

- 12个粗任务细分为14个可验收Task，不扩阶段边界。
- 项目偏好确认写与本地保存后GET失败分别处理；三view共享同目标偏好，route key按view结束UI。
- labelMap仅来自当前已持有、可见项目，不借P13重开P11已接受的M7完整工作区标签范围。
- P11归档页头旧service、P12 R36/R38已完成，P13验真实消费者，不重新实现或重复关闭。
- 归档列表AM门禁与可见guest的只读详情不同；字段只读与恢复/删除权限不能合为一个开关。

这些属于既定设计内的局部适配，没有新增架构级问题待裁定。产品实现若发现根权限/API设计缺口，仍先提交再上报。

## 4. 预检发现与处理

独立builder-check没有重要未回答问题。随后主控三视角复核修订以下文稿缺口，未改产品：

| 编号 | 级别 | 问题与根因 | 处理 |
|---|---|---|---|
| R1 | Major，局部文档 | “范围必须双端”可能把P10b合法的开区间当错误，原生反向转换实际仍用range而非独立单端操作符 | spec§2.4区分已保存开区间与编辑中的双端草稿；T3补preservesLoadedOpenRanges，保留两个空槽合同 |
| R2 | Major，局部文档 | “非法持久化不扩宽”的笼统表述与现有整字段清洗合同冲突 | 明确非法字段按P10b清除，合法日期AND/有效条件不丢，不引入新有损翻译 |
| R3 | Major，局部文档 | plan初稿写了不存在的MemberIssueFilters、两个猜测测试文件名，并把既有hook测试误写为新增 | 改为ProjectIssueFilters & Pick<MemberIssueQuery,"project_id">；日期回归用basic-dropdowns、选择器用member/base真实消费者；T4扩展既有use-project-preferences-follow.test.tsx并保留原性质 |
| R4 | Minor，局部文档 | “七调用”没有限定M4；全仓共享调用更多；聚焦命令只有模板 | 明确七个M4调用与其余共享消费者兼容边界；附录B给出T1–T9完整argv，T10–T13各自列页面命令 |
| R5 | Minor，局部文档 | 归档旧单测接受kanban/孤儿子分组，需要解释本期页面能力规范化后如何保留原测试性质 | T5要求同步迁移旧预期、保留合法兄弟字段/false与账户隔离性质；不改M6/M7共享helper语义 |

R1–R5复核均已落到spec/plan。未以“未来计划”冒称现存range、ArchiveIssueModal或访客工具栏已经修好。相应缺口是本期明确实施范围，不是本次产品交付。

## 5. 与原设计和准备方法的差异

设计§12/P13和§15同步14Task及已有成果输入；既定完成线I5/I15/I19、C2/C9和handoff落点保留。原型/逐Task产品重放未进行；无新生成hash、测试或性能数字。主控的源码阅读区分全文、指定合同章节和仅检索，不声称审完整个M4模块；详见inventory。

准备静态核对包括：14Task编号/接口/验收映射、每Task五步、A1–A12全部未勾选、相对文档链接、Markdown围栏、实际旧路径与未来新增路径分类、phase未变、产品diff为空、git diff --check。报告和回执引用的P12证据与真实归档manifest核对；具体执行结果以precheck-static.json为准。

## 6. 移交事项

下一步获得负责人实施安排后从T1开始。本轮不实现、合并P13或清理P13工作树。T14才做整分支deep-review及修复复审，完成后先交负责人安排。

输入M1/M2/M3与P5/P10b/P11/P12的本期子项见spec§7。P14接完整详情/创建/父子/关联/链接/编辑器与EMOJI警告；P15接C9草稿及评论/搜索等；P16/收尾接全局守卫/死代码与M4最终关闭；M7仍接完整工作区标签目录。

## 7. 已知限制

静态预检不能证明运行通过。选项defaultOpen、真实焦点/几何、时区和浏览器、UI owner时序以及错误恢复仍需T1起验证。无产品测试运行、无变异计数、无新接口或依赖。M4§14、总体§9.4、所有输入handoff顶层open和M5–M8最终closeout不提前关闭。

P12的CI、原始红绿/trace和归档是上一阶段证据；仅用于确认本期基线可用，不能计作P13验收。

## 8. 实施接续（2026-10-31）

负责人授权从T1实施并在阶段末deep-review，先不合并。证据在worktree的.superpowers/sdd/P13-web-filters-archives-profile/，准备证据仍保留在主目录。

T1补subscriber配置/原生入口、profile项目条件入口以及me/none选项；复用原生转换，没有伪造成员实体或增加接口。可选additionalOptions由原多选配置组合，已有消费者默认不变。起步基线77通过；新测试3个真实断言失败后修复，最终3文件81测试通过，utils包7文件62测试通过；M01移除subscriber注册导致3断言红并准确恢复。make lint-web 54任务通过（31缓存），包含关键词/翻译键/类型/格式/警告上限；没有把它称为完整F。首轮作者测试接线、profile输入与翻译键位置错误及新增测试lint告警均保留raw，修正后复验，不计作产品根因或有效行为红。

第一个产品提交将phase改为M4/P13；没有P12/P13到期例外。T2–T14及真实页面/三浏览器/完整F/最终分支CI尚待执行。

T2通过真实CustomSearchSelect覆盖空条件首次打开、静态/异步候选、配置切换的旧成功/错误、卸载、重试、合法未知ID及四消费者的中英文搜索。删除旧loadOptions，仅两处值输入共用局部读取hook；所有者含config/getOptions/重试序号，旧答不写入。T1/T2六文件106测试通过，make lint-web 54任务通过（36缓存）。M02/M03/M04分别使旧答隔离、失败反馈、首次打开的真实断言失败并恢复；M04第一次因替换锚点不唯一未执行，不计有效红。新增lint问题修复后，移除旧数组index key使warning cap由121降到120。保留全部原始失败；已有member/base测试只作接线回归，未冒称真实控件运行。T3–T14仍待完成。

T3用真实Calendar验证两步草稿、同日范围、已保存单端范围、反向/越界文字及aria关联、条件换id和外部值ABA。首次10项行为失败；外部值ABA新增断言先红后修。日期回归5文件43通过，UTC+14与UTC-7各10通过；M05/M06/M07分别去草稿、改UTC解析、去aria关联，产生9/6/5项行为失败并恢复。首次类型检查暴露toFilterArray泛型推断过宽，明确string后lint-web 54通过（36缓存）。旧DateFilterModal真实表单校验使用日历输入替身强制输入反向日期，单独验证date2和分号callback；不冒称该测试是完整页面日历。

T3局部接口调整：实际DayPicker首次选择即返回同日起止，按旧回调第一参数无法辨认两次点击。DateRangeDropdown的onSelect向后兼容增加可选selectedDay，仅传输入事实；旧调用仍消费原range，草稿归富筛选，不激活既有未使用bothRequired或增加共享缓存。同步设计/spec/plan，P11原子修改4项和原共享日期控件回归通过。

T4通过真实HeaderFilters/RootStore/FakeNerve验证显示属性确认、同字段队列、跨字段并发及未改calendar/false/null，确认后列表拒绝只出现一次提示且重试只GET。旧权限/路由/会话ABA和卸载均结束跟进；继续复用useDetailScope与root权限水位。默认/自定义按钮均具名称、expanded、disabled、tabIndex和Escape回焦，实际Panel承接定位；七个M4入口具翻译标题。六文件38测试通过、lint-web 54任务通过；M08/M09/M10分别产生3/1/8项真实失败并恢复。M08/M09属于既有确认队列回归，不重复记作本期新实现。三浏览器几何仍待T13。

T4第一轮完整F中生成与Go lint通过，服务端TestNewAppDeleteRowsJobRuns暴露旧测试夹具竞态：61天前的软删审计行超过60天保留期，可被启动清理物理删除。显式在删除任务完成后执行purger使同一根因稳定失败（count=9）；改用配置保留期一半的旧时间后连续10次通过，保留原锁/批处理/审计断言及明确交错。仅修测试，不变服务端锁、协议或清理语义；不以它触发新架构调整。首次完整失败和红绿raw均保留。

第二轮完整F的TestDeletionInterleaving24在观测锁等待时失败。单子例20次未复现；整个24矩阵20次复现一次，临时诊断明确同一事务阻塞了关联行删除worker和项目行周期任务，旧helper把全部等待者限定为一个。新夹具明确制造第二等待者，旧helper稳定选错projects；按预期表筛选候选后仍检查稳定PID、实际SQL及锁类型，24/25全矩阵连续10轮通过。只调整测试观测，不改生产事务/锁顺序/超时；临时诊断源码已撤回，证据保留。最终验证见下段。

T4最终检查点七项均通过：生成/Go lint/Go全测取t4-full3，Web lint/knip/强制全测/E2E取t4-full4。第三轮Web全测曾有旧路由接线夹具缺少detail root的10项失败，改用真实RootStore/FakeNerve后原性质保留并补guest断言，相关23项通过。服务端源输入逐文件核验不变（t4-server-input-reuse.json），没有冒称第四轮重跑Go。最终Web 240文件2587项、全部包18任务通过，E2E 180通过；所有失败raw保留。T4完成，继续T5–T14；阶段整审、三浏览器与最终分支CI仍待执行。

T5归档原store统一规范化为list/无子分组，保留合法兄弟设置、false和列表组折叠；查询最后固定archived=true。局部更新hook复用既有宿主/权限身份，消费同步及异步拒绝，列表错误仍由binding单独提示和GET重试。实际页头与列表对guest均不挂载，viewFlags作为字段只读的单一能力来源，移除重复覆盖；恢复/删除独立留T6/T7。六文件49测试通过（含project-lifecycle原行展开性质），lint-web 54通过；删除无用代码使警告上限120降119。M11/M12/M13分别2/3/1项行为红，均准确恢复；M11为既有账号隔离回归。新增测试的动画/IntersectionObserver夹具错误及flat分页条件不符已修正，失败raw保留，不计为产品缺陷。T4远端检查点CI仍待结束，尚非最终阶段CI。

T6真实归档弹窗具名称、正常Tab顺序、同步提交锁、pending禁止dismiss和当前打开的身份快照。缺失item、item/open/权限ABA、卸载和退役均不能误关闭/提示。实际详情菜单曾吞归档拒绝并报成功，恢复也缺当前宿主跟进；修复后原命令拒绝保留，modal单独反馈，再由onArchived执行导航/关闭peek。Peek既有宿主测试保留原性质、迁移到明确确认回调，并以真实菜单新用例核一次结果。交错旧归档列表触发权威补读，拒绝仍保留一次POST确认，真实Retry只重试GET。五文件60项通过、Web静态54任务通过；M14/M15/M16分别1/5/5项行为红并恢复。M16保护P12原有恢复拒绝合同；测试夹具曾错误期待已知退出活动列表必补读及误用binding属性，全部失败raw保留，最终按真实归档交错场景验证，没有改生产协调器来满足夹具。

T7在归档layout以工作区/项目/工作项key覆盖真实页头和正文；内容维持只读，恢复及按创建者/管理员判断的删除独立授权。删除拒绝原样交弹窗一次反馈，旧完成不能导航新页面。实际路由/菜单9项与原宿主/resource回归共8文件93项通过。描述编辑器在组件测试中是只读prop探针，真实输入仍留T11。M17前两次变异未失败：快速读取卸载旧菜单，掩盖短暂复用。保持返回后的GET在途，M17-held明确旧弹窗泄漏并红、恢复后绿；只有此回执计有效性质，诊断throw和早期夹具失败不计。首轮旧菜单fixture漏useUser和翻译键错误已修，保留失败raw。T4检查点2209d237的远端CI已全绿：https://github.com/open-nerve/NerveProject/actions/runs/41105350433 （t4-ci-poll4.json）；不是最终阶段CI。

T8在route显式收窄三个TProfileViews，typed props贯穿两个profile根、BaseList/BaseKanban、mount和actions；原as TProfileViews检索为零。共用mount的日历/表格可选viewId同步机械收窄，实际非profile调用仍省略。删除晚effect设置relation；请求直接接当前typed view。profile layout的工作区/用户/view key结束页头、手机与正文owner，访客工具栏和列表同门禁，M3成员摘要保留。首红14项真实行为；最终五文件48项和Web静态54任务通过，M18/M19分别5/3项行为红并准确恢复。route测试真实layout/header/mobile/sidebar与请求链，正文用带真实列表读取和popover的探针；不冒称完整ProfileIssuesPage已验，留T9/I19。guest断言一度误把测试前置workspace GET计入页面请求，已清理观测窗口；navigate重载的测试类型错误也已修正，原始输出保留。

T9把profile内存设置键补全为workspace/target，当前路由显式取对应设置；账号仍由RootStore隔离，同目标三个view共享本地设置。保存先写内存和storage，列表失败不回读覆盖；桌面、手机与富筛选复用本地更新hook消费Promise。目录只投影已持有且仍可见的项目/标签；服务端返回的冷标签组保留行、total和cursor，名称未知显示中性占位，后续目录到达可回显。实际Page/Layout/两布局14项与相关回归共10文件134项通过，store另含storage不可用、工作区隔离。M20/M21/M22分别8/7/1项行为红并逐字节恢复；M21保护既有project_id合同，不计新实现。

T9七项完整门禁均在当前产品输入运行通过：生成、Go lint、串行Go全测、Web静态54任务、knip、强制Web全测18任务（Web 247文件2676项）、E2E 180项。回执为t9-full-*；静态warning cap由119降118。初期夹具项目ID/原生分组字段和测试输入键、bound方法及hook依赖错误均保留raw，未放宽原断言；冷标签分组与未消费Promise为真实页面根因。此前T4“11属性”实际只遍历旧目录9项，T10读入口发现缺创建/更新时间，将用独立页面枚举补验与修正，不能把旧测试称为11项完整覆盖。T10–T14及最终整审/分支CI仍待完成。

T10新增I5三条页面故事：started/me/none与两个日期范围AND、11属性及布局/分组/子分组/排序/空组/子项、403拒绝和确认后503读取重试。操作由真实控件产生PATCH，核原生query、数据库三列、刷新及账号/项目隔离；日历嵌套未改字段保持。页面确认添加图标无名称、创建/更新时间未进入显示属性目录，沿原组件补翻译名称及两个既有属性键；旧T4测试改成独立枚举11项，避免随生产目录漏项。I5共4项（1 API、3页面）通过，相关组件23项通过，Web静态54任务通过。M09-page在真实构建中破坏calendar合并使保存断言失败，源码逐字节及构建恢复后4项再绿；与M09同性质，不增加独立变异计数。

T10首轮测试传Page给Locator helper、匹配双main和Assignee/Assignees名称错误属于夹具问题，不计产品缺陷；各失败trace/截图/数据库已保存。新增await循环违反原lint零告警约定，改用顺序reduce和独立Promise.all，未加豁免。此阶段只完成Chromium页面，Firefox/WebKit与最终完整F仍留T13。

T11完成5条I15页面故事和保留的API故事，Chromium共6项通过。真实操作覆盖最后一行归档的pending双击单POST/一次成功、403留窗、Back晚答不导航/不提示、归档筛选刷新保持、只读详情恢复及有权删除完整依赖图、guest可读详情而不读归档列表；普通详情与peek有子项时稳定交互且子项只读一次。页面红定位并修复非终态仍显示Archive、子项组件重复读取引起父项核验/卸载循环，以及行卸载吞掉成功提示。确认宿主移到同一布局内，导航key与既有权限身份结束旧请求；原root命令、权限来源与后端协议不变。只读页隐藏新增部件入口，部件按钮内容改为span避免嵌套button。设计/spec/plan同步记录局部职责调整。

T11相关组件首次7文件90项通过；增加真实确认宿主的最后一行和权限ABA回归后，恢复运行5文件46项通过（含15项弹窗测试），不是两轮数量相加。M23破坏宿主存续使一次成功提示断言红，源码逐字节恢复。现有T6的M14–M16仍为其原运行证据，不冒称重跑。静态检查见t11-static-final；详情加载期间父项core有页头/正文与核验，页面断言交互前后无新增读取、子项恰为一次，避免以不保证的core请求合并次数作合同。首次按钮文案/双sidebar定位/导航点击夹具错误和全部失败trace保留，未放宽产品断言。T12–T14及最终分支CI待完成。


T12新增五条I19页面故事并保留API故事。三关系、两布局、四组型逐一核对调用者可见集合；跨两页保持relation、project_id、筛选与total，刷新和同目标view共享，另目标与真实退出/登录后的账号隔离。OwnOnly项目按调用者而非目标用户筛选，隐藏/退出/归档项目不泄露；guest桌面和手机无工具栏且列表GET为零。旧面板和延迟回答在view/目标/Back后失效；已持有标签被选中后，冷刷新仍保留原限制且不补拉目录。M20-page破坏relation使真实Created页面断言红，逐字节恢复并重新构建后，I5/I15/I19共16项通过，静态检查通过。此变异复用M20性质，不增加独立性质数。首轮种子角色、控件名称、同账号切换夹具错误与报告均保留；请求记录夹具与延迟夹具统一按pathname匹配，使带查询参数的GET真正进入观测。T13三浏览器、完整F和T14整审仍待完成。


T13完成C2/C9真实可达页面及single/multi×en/zh独立浏览器兼容夹具；最新三浏览器各22项通过（t13-matrix5-*），包含13个I5/I15/I19页面故事、5个真实交互故事和4个兼容夹具，不含API故事。四选择器翻译/加载/defaultOpen/搜索/键盘/回焦、默认/自定义FiltersDropdown定位、手机Layout和归档焦点/Back均有实跑。single不在当前M4配置生成路径中，按已记录裁定用生产控件夹具验证，不伪称业务入口。

页面实跑修复手机Layout缺少名称/嵌套按钮、字段与运算符中文缺失；Firefox先红后绿证明toast入场扩大根滚动区，改fixed；项目确认相同表达式重建ID丢焦点，改由确认宿主保留DOM并推进已保存基线。实际RootStore新增焦点断言先红后绿，相关3文件51项通过。Firefox fill的IME收尾按Headless UI两帧机制同步；Mac WebKit使用原生Option-Tab，焦点陷阱允许容器自身；恢复故事等待真实详情加载后再跳页，避免用例取消仍在途GET。没有放宽watchPage或新增console忽略，所有失败trace和诊断raw保留。

最终完整F的生成、Go lint、串行Go全测使用t13-full-*（Go测试456.216秒）；前端使用t13-full2-*：lint-web 54任务、knip通过、强制全测18任务（Web247文件2679项）、默认E2E202项通过。首次全量前端3个旧夹具仍把初读归子项widget，已改经真实详情加载器并保留错误/重试、两种旧答顺序和无重复GET；聚焦3文件34项通过，未改产品或削弱原业务断言。

专项改动文件lint零告警；web cap由基线121降114，constants由1降0，未增加keywords豁免/依赖。分支检查点推送和准确CI后置核，T14尚未完成，不把本段验证当作整审或授权合并。


## 9. 最终验收映射

| 标准 | 结果 | 证据与边界 |
|---|---|---|
| A1 原生词汇/筛选 | 已核准 | filter-config、issue-filters、真实 HOC 和 I5/I19；subscriber、me/none、日期 AND、project_id、非法字段清洗。M01/M21。 |
| A2 四选择器 | 已核准 | select-inputs/options 当前性测试，C2 的添加/运算符/多选业务页面及 single/multi×en/zh 生产组件夹具；defaultOpen、加载、搜索、键盘、回焦。M02–M04。 |
| A3 日期 | 已核准 | range、本地日 UTC±、date-filter-modal/basic-dropdowns/date-range-write 回归；C2 实际日期草稿/错误名称/aria，保留原子日期写入。M05–M07。 |
| A4 项目偏好 | 已核准 | preferences-controls、list-data、follow/integration 和 I5 页面核 PATCH、数据库、确认值、拒绝/队列/旧答及 GET-only retry。M08–M10。 |
| A5 归档 | 已核准 | archive-page、archived/filter.store、restore-menu 和 I15 页面：固定 list、字段只读、有权恢复/删除；R1/R2 新红绿补最后一行与 storage 回页保留。M11–M16。 |
| A6 弹窗/详情身份 | 已核准 | archive-issue-modal、archive-detail、C9/I15：同 tick 单次写、pending、名称、焦点、Back/权限 ABA 晚答；布局宿主跨空态存续。M14–M17-held/M23。 |
| A7 typed view | 已核准 | profile-view/profile-route、list-mount、route-scope；真实路由一次收窄，非法段不读，web 中 as TProfileViews 零命中。M18。 |
| A8 个人主页权限 | 已核准 | profile-page、I19：三关系/两布局/四组型/分页与 OwnOnly 交集；访客桌面和手机无工具栏且零列表 GET。M19/M20。 |
| A9 本地设置 | 已核准 | archived/profile filter.store、store-context、local-update 和 I19：workspace/target/account 隔离、同目标共享 view、冷标签/未知 ID、project_id、storage 损坏/不可用与旧宿主。M11/M12/M21/M22，R1/R3。 |
| A10 页面与三浏览器 | 已核准 | t14-full-e2e：202 通过；t14-matrix-chromium/firefox/webkit 各 22 通过，均非零、零失败/跳过。业务页面与兼容夹具分开计数。 |
| A11 验证与清理 | 已核准 | 11 域输入匹配；生成/Go lint/Go 全测沿 T13 未变输入，Web 静态/knip/强制全测/E2E/三浏览器用 T14 新回执；117 实际文件零 lint 告警、45 候选逐读者闭合、23 个有效不同变异。最终文稿 CI 以 §14 后置回执为交付门。 |
| A12 整审和交接 | 已核准 | 独立全分支审查 + 主控修复复审，R1–R4 全处理，M1/M2/M3 只关闭本期子项；设计/计划/spec/本记录/整体报告同步，保留分支，禁止自动合并。 |

## 10. 整分支终审、修复和复审

独立审者 p13_final_review 审 ff89d0a1..863283fc：131 变更路径中 124 全文、7 历史/规则文档相关章节，另 28 个相关调用方/测试全文；root.store 仅实际写入/确认路径。按逻辑、集成、覆盖依次审查，没有把三个视角说成三名审者。0 Critical、2 Important、2 Minor；负责人要求局部问题全部修复，故 R3/R4 也在本轮处理。主控对修复提交 de616d06 的 40 路径 diff、关键生产全文与消费链复审；独立报告没有被改写成已审最终提交，没有再派第二名审者。

| 编号/等级 | 根因与修复 | 红绿与复审 |
|---|---|---|
| R1 Important | 归档重挂载无条件回读 storage，覆盖本会话选择；改为 workspace/project 首次初始化，显式跨范围 getter，账号仍由 RootStore 隔离。 | t14-r1-storage-red 两种失败明确 high→空/low；真实 A→B→A 保留条件/false/折叠后通过。新 session 恢复与退休账号拒绝由 store-context 回归。 |
| R2 Important | 恢复/删除最后一行会卸载行内反馈 owner；同布局 IssueListCommandHost 承载确认与恢复反馈，行只提交目标，保留原权限/路由身份。项目删除同根因一起修。 | t14-r2-restore-probe、t14-r2-delete-red 均行已消失但 success=[]；真实 archive-page/list-data 补一次提示、同 tick 单写、403 留窗留行并明确重试，权限 ABA 晚答静默。 |
| R3 Minor | 项目偏好重读以旧 storage 覆盖本地分组折叠；只首次初始化本地字段，服务器三个字段仍更新。 | t14-r3-collapse-red 明确折叠丢失；正常和 blocked 两分支同时核折叠保留、服务器新 filters 到达。 |
| R4 Minor | DateRangeDropdown 的无行为 Props 暗示不存在的功能；按 M1 清掉声明/无调用分支及改动范围其它真死链。 | 不伪造行为红；实际调用检索、原 deadsym、types、117 文件零告警、knip、完整 Web 与三浏览器验证兼容。 |

剩余 Critical/Important/Minor 确认项为零。不是无风险保证：没有逐帧审全部 trace、没有重放全部历史变异、没有重新设计未改的后端锁/API/schema。已接受范围限制见 §13 和整体审核；没有架构级阻塞。

### 六个写入口的四种结果

| 入口 | 确认 | 拒绝 | 旧 scope | 确认后读取失败 |
|---|---|---|---|---|
| 项目富筛选、显示设置三个偏好字段 | preferences-controls / list-data / I5：PATCH确认、队列、清理值、11属性、calendar/false | preferences-controls / I5：403保留确认数据、一次提示 | use-project-preferences-follow：权限/路由/会话ABA与卸载，list-data后续草稿不被旧确认覆盖 | preferences-controls / I5：保存后503保留确认值，Retry仅GET |
| 归档本地筛选、显示、折叠 | archived/filter.store / archive-page / I15：本地恢复、固定list、隔离actor/workspace/project、无PATCH；R1 新增 blocked reads/stale stored values 的 A→B→A 会话保留 | storage损坏/抛错保留内存；当前非读取异常由use-local-issue-filter-update一次反馈 | use-local-issue-filter-update：target/permission ABA、unmount、retire | archive-page：first/next/read-after-save拒绝只GET，原行和本地选择保留 |
| profile本地筛选、显示、折叠 | profile-page / profile/filter.store / I19：同目标共享view、workspace/target/account隔离；project_id不丢 | storage不可用保留会话内偏好；非读取拒绝由局部hook处理 | profile-route / profile-page / I19：view/target/Back/guest变化结束旧工具栏与迟到GET | profile-page桌面/手机503留本地选择、一次错误、GET重试 |
| 项目行及详情/peek归档 | archive-issue-modal / I15：同tick双击单POST、最后一行移除后仍一次反馈；DB archived_at/activity | 真实detail菜单与I15：422/403留窗、显式重试，非终态菜单不可达 | modal item/open/permission ABA、unmount/retire；I15 Back晚答不导航/提示 | modal crossing旧归档GET引发权威补读503：确认保留、一次POST，Retry只GET |
| 归档恢复 | archive-page / restore-menu / I15：真实列表最后一行一次反馈、同 tick 单 POST；详情/peek 的原确认与导航 | archive-page / restore-menu：403 留行且一次错误，明确重试后成功 | restore-menu：替换/卸载/权限ABA；detail scope原水位 | 间接证据：issue-changes.test.ts 的 sixWriteCopies unarchive late raw reads cannot undo confirmation；issue.store.test.ts 的 directConfirmImmediatelyVerifiesAcrossCopiesWithoutResendingWrite 验证共用核验拒绝、只 GET 重试。后者是 update 场景，不冒称 restore 直接页面测试。列表已证明恢复项不属归档集合时 fetchFirst 正常 resolve。 |
| 项目与归档列表删除、归档详情删除 | list-data / archive-page：最后一行删除后一次反馈、同 tick 单 DELETE；archive-detail / I15：授权与导航，完整依赖删除 stamp | archive-page / archive-detail：403 留窗留行、一次原生错误、明确重试；非创建者 member 不给入口 | archive-detail：旧删除不导航或关闭下一route确认 | 间接证据：issue.store.test.ts 的 confirmedDeleteCannotBeResurrected / confirmedDeleteEndsPendingVerificationAndOldGetCannotScheduleFollowup，issue-changes 的 sixWriteCopies delete late raw reads cannot undo confirmation。删除以 tombstone 排除实体，不要求每个确认随后再发核验 GET；列表自身读取失败沿原 binding Retry。 |

## 11. 验证、变异和修改范围清理

证据根 E=.superpowers/sdd/P13-web-filters-archives-profile/，均为实际运行回执及原始输出。

| 门禁 | 原始回执 | 最终结果与输入 |
|---|---|---|
| 生成 | t13-full-gen | 通过；API、生成脚本/配置、生成物输入未变，逐文件复用。 |
| Go lint | t13-full-lint-go | 通过；使用既有主目录 2.13.2 二进制；服务端/工具/配置未变。 |
| Go 全测 | t13-full-test-go | 通过，456.216 秒；GOFLAGS=-p=1 串行、-count=1，未冒称 T14 再跑。 |
| Web 静态 | t14-final-lint-web | 54/54，36 缓存；76 keywords 规则/24 例外零命中，1278 翻译键通过。 |
| knip | t14-final-knip | typegen 后 --treat-config-hints-as-errors 通过。 |
| Web 全测 | t14-full2-test-web | TURBO_FORCE=true，18/18、零缓存；Web 247 文件/2688 项，8 包原始日志另存 t14-web-suite-logs。 |
| 默认 E2E | t14-full-e2e | make e2e 重建实际产品，202 通过。 |
| Chromium / Firefox / WebKit | t14-matrix-{chromium,firefox,webkit} | 各 22 通过、零失败/skip；每浏览器 13 个 I5/I15/I19 页面 + 5 个 C2/C9 业务交互 + 4 个真实组件兼容夹具。 |
| 修改文件 oxlint | t14-final-changed-oxlint | 118 个候选路径，实际执行 117 文件，0 warnings/0 errors；配置排除的路径不伪计执行。 |

13 文件 170 项修复聚焦通过；真实会话 2 文件 25 项通过。第一次完整 Web 因无路由的 store-context 夹具未传 workspaceSlug，补四处明确作用域后全量通过；没有改变账号替换、退休 Root 拒绝或已保存值断言。新项目菜单测试曾缺 ResizeObserver/有效几何；这是 jsdom 接线修正，不算产品红。早期聚焦 argv 错写不存在 widget-host，实际只有 12 个文件；后续核全路径并使用真实 issue-detail-widget-modals，未用零测试充数。所有失败 raw 保留。

生成/Go 三项和最终 Web/浏览器/专项 lint 共 11 域在 t14-inputs-before-commit.json 及最终 t14-final-input-audit.json 中核运行 HEAD + dirtyInputs 与最终产品字节；记录 logSHA256，零 mismatch 才允许复用。文稿随后单独提交，不改变产品输入。测试 stderr/Node 环境警告保留于原始日志；“零 oxlint”不代表所有日志从未出现任何警告。

### 有效变异

| 性质 | 保护内容 | 实际行为红摘要 |
|---|---|---|
| M01 | 真实富筛选宿主必须注册subscriber配置并显示合法已保存条件 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 3 ⎯⎯⎯⎯⎯⎯⎯；Test Files  1 failed (1)；Tests  3 failed (3) |
| M02 | 旧配置响应不能覆盖已到达的新配置 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 1 ⎯⎯⎯⎯⎯⎯⎯；Tests  1 failed / 19 passed (20) |
| M03 | 当前读取失败必须显示可重试错误 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 3 ⎯⎯⎯⎯⎯⎯⎯；Test Files  2 failed (2)；Tests  3 failed / 17 passed (20) |
| M04 | 加载时不能消耗空条件的首次打开 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 5 ⎯⎯⎯⎯⎯⎯⎯；Tests  5 failed / 15 passed (20) |
| M05 | 首次选择必须保留本地起日草稿 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 9 ⎯⎯⎯⎯⎯⎯⎯；Test Files  1 failed (1)；Tests  9 failed / 1 passed (10) |
| M06 | 本地日不能经UTC日期解析移日 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 6 ⎯⎯⎯⎯⎯⎯⎯；Test Files  1 failed (1)；Tests  6 failed / 4 passed (10) |
| M07 | 范围错误必须与触发按钮aria关联 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 5 ⎯⎯⎯⎯⎯⎯⎯；Test Files  1 failed (1)；Tests  5 failed / 5 passed (10) |
| M08 | 偏好PATCH确认前仍显示最后确认值 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 3 ⎯⎯⎯⎯⎯⎯⎯；Test Files  1 failed (1)；Tests  3 failed / 1 passed (4) |
| M09 | 局部显示设置必须保留嵌套calendar和false | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 1 ⎯⎯⎯⎯⎯⎯⎯；Test Files  1 failed (1)；Tests  1 failed / 3 passed (4) |
| M10 | 权限ABA后不能跟进旧页面写入 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 8 ⎯⎯⎯⎯⎯⎯⎯；Test Files  1 failed (1)；Tests  8 failed / 3 passed (11) |
| M11 | 归档本地设置按 actor 隔离，不能读取另一账号 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 2 ⎯⎯⎯⎯⎯⎯⎯；Test Files  1 failed (1)；Tests  2 failed / 12 passed (14) |
| M12 | 归档富筛选写入 rich_filters 后能重新加载 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 3 ⎯⎯⎯⎯⎯⎯⎯；Test Files  1 failed (1)；Tests  3 failed / 11 passed (14) |
| M13 | 真实归档列表的字段按钮不可编辑 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 1 ⎯⎯⎯⎯⎯⎯⎯；Test Files  1 failed (1)；Tests  1 failed / 4 passed (5) |
| M14 | 归档实际确认按钮同一轮事件只提交一次 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 1 ⎯⎯⎯⎯⎯⎯⎯；Test Files  1 failed (1)；Tests  1 failed / 12 passed (13) |
| M15 | 旧归档确认不能关闭新窗口或执行导航 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 5 ⎯⎯⎯⎯⎯⎯⎯；Test Files  1 failed (1)；Tests  5 failed / 8 passed (13) |
| M16 | 恢复命令拒绝后不能再报告成功或关闭peek | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 5 ⎯⎯⎯⎯⎯⎯⎯；Test Files  1 failed (1)；Tests  5 failed / 9 passed (14) |
| M17-held | 归档详情路由 Back 结束页头和正文的浮层身份 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 1 ⎯⎯⎯⎯⎯⎯⎯；Test Files  1 failed (1)；Tests  1 failed / 8 passed (9) |
| M18 | 只有明确合法的个人主页关系进入原生读取 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 5 ⎯⎯⎯⎯⎯⎯⎯；Test Files  2 failed (2)；Tests  5 failed / 25 passed (30) |
| M19 | 工作区访客不挂载桌面或手机个人主页工具栏 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 3 ⎯⎯⎯⎯⎯⎯⎯；Test Files  1 failed (1)；Tests  3 failed / 17 passed (20) |
| M20 | 当前个人主页视图决定首页及下一页的关系条件 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 8 ⎯⎯⎯⎯⎯⎯⎯；Test Files  2 failed (2)；Tests  8 failed / 15 passed (23) |
| M21 | 修改其它筛选仍保留明确的项目限制 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 7 ⎯⎯⎯⎯⎯⎯⎯；Test Files  2 failed (2)；Tests  7 failed / 16 passed (23) |
| M22 | 标签目录未加载时不丢弃合法的已保存标签条件 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 1 ⎯⎯⎯⎯⎯⎯⎯；Tests  1 failed / 22 passed (23) |
| M23 | 归档确认移除发起行后仍保留确认反馈 | ⎯⎯⎯⎯⎯⎯⎯ Failed Tests 1 ⎯⎯⎯⎯⎯⎯⎯；Test Files  1 failed (1)；Tests  1 failed / 14 passed (15) |

共 23 个不同生产性质；M17 前两次未检出、M04 首次锚点失败不计，M09-page/M20-page 是同性质真实页面重放不再加数。productionAtExecution、逐字节 restoredAtExecution 和 currentProduction 分列于 t14-final-mutation-audit.json；M23 的旧 archive-host 现为 command-host，映射明确。不能把历史红/恢复 hash 称为最终提交上重跑。

### dead 与守卫

M1 附录 A 原始 deadsym/program/domains 提取及脚本 hash 在 dead-audit/scripts.json。真实编译扫描前为 prop390/member888/export20，修改范围64候选；清理后全仓候选 prop379/member881/export20，修改范围45候选/19文件均有实际读者或协议传输。净清19个候选，独占 operator-label/helper/barrel 链一并删除；不是全仓零 dead。45行的文件、名称、读者逐项写在[整体审核§5](../../../reviews/re-19-M4P13整体审核.md#5-修改范围的保留候选)，机器账为 t14-dead-ledger.json。web warning cap 121→114，constants 1→0，无新增豁免或抬上限；全仓 no-shadow/dead 终态仍 P16。

## 12. 本期交接子项

M1 的 typed profile view 已关闭，修改范围 dead/oxlint 具名记录；M2 §10 的四选择器/defaultOpen 已按业务入口与兼容夹具完成，§12 归档页头实例的 P11 成果只作页面回归；M3 §6 的 profile/guest/key、§7 筛选/日期/归档弹窗、§8 四选择器文案完成。三份 handoff 顶层仍 open。完整编辑器/创建/关系 P14、草稿/评论/搜索/复制 P15、C12/全仓清理/总交接 P16 与收尾不冒领。

## 13. 仍有效的边界和裁定

完整 workspace 标签目录仍 M7，冷页允许中性占位，合法条件与列表结果不能丢；single 当前无 M4 业务配置，夹具证明组件兼容，未来需业务故事。M6/M7 DateFilterModal 完整业务页未验收；本期只修共享日期边界并保留 callback。storage 无法保存时只保证本会话内保留，整页刷新持久化无法承诺。恢复/删除的“确认后读失败”按 §10 明确直接与共享合同间接证据，不能夸大为每个页面组合均直接验证。

执行过程中全部 Ruling、理由、错误代价及独立审者 Declined to judge 七项的主控决定保留在 E/progress.md，逐条全文列于[整体审核§6](../../../reviews/re-19-M4P13整体审核.md#6-执行裁定与错误代价)。无延期 Minor。

## 14. 最终交付门与保留现场

分支 codex/m4-p13；基线 main ff89d0a1219a8d09ff924f5bd5257c486cd7e2b8；最终产品 de616d06ca7befeb5a39f82a6c70ff18a3f03742。T13 检查点 863283fc 的 [CI 41127298383](https://github.com/open-nerve/NerveProject/actions/runs/41127298383) 三任务成功，仅作历史检查点，不代替本轮最后提交的 CI。

文稿提交后由主控推送，读取该精确 HEAD/branch/run/attempt，确认 server、web、e2e 三任务全部 success，生成 E/final-delivery.json（包含准确 CI URL、远端提交、输入/日志 hash 审计）。这份后置回执是 A11/A12 与 T14 完成交付的最后前提；若失败先按真实日志处理，不用正文勾选跳过。以此避免文稿反复写入自身 SHA 引起提交循环。

当前没有合并授权。主目录保持 main，工作树/分支及证据保留；主目录 .superpowers/sdd/P13-preparation-20261031/ 也必须随未来授权集成归档，不能漏掉。M4 设计 §14、总体设计 §9.4、交接顶层与 M5–M8 最终 closeout 仍待后续阶段；本次不开始 P14。
