# M4/P13 筛选、归档与个人主页：预检记录

日期：2026-10-31。**当前已获准实施，T1–T6完成，T7接续；以下§1–7保留预检快照，实施接续见§8。** [spec](../specs/P13-web-filters-archives-profile.md)、[plan](../plans/P13-web-filters-archives-profile.md)、[deep-review预检](../../../reviews/re-18-M4P13预检审核.md)。产品A1–A12仍待阶段验收；本记录不等于阶段实施后的整分支终审。

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

T4通过真实HeaderFilters/RootStore/FakeNerve验证11属性、同字段队列、跨字段并发及未改calendar/false/null，确认后列表拒绝只出现一次提示且重试只GET。旧权限/路由/会话ABA和卸载均结束跟进；继续复用useDetailScope与root权限水位。默认/自定义按钮均具名称、expanded、disabled、tabIndex和Escape回焦，实际Panel承接定位；七个M4入口具翻译标题。六文件38测试通过、lint-web 54任务通过；M08/M09/M10分别产生3/1/8项真实失败并恢复。M08/M09属于既有确认队列回归，不重复记作本期新实现。三浏览器几何仍待T13。

T4第一轮完整F中生成与Go lint通过，服务端TestNewAppDeleteRowsJobRuns暴露旧测试夹具竞态：61天前的软删审计行超过60天保留期，可被启动清理物理删除。显式在删除任务完成后执行purger使同一根因稳定失败（count=9）；改用配置保留期一半的旧时间后连续10次通过，保留原锁/批处理/审计断言及明确交错。仅修测试，不变服务端锁、协议或清理语义；不以它触发新架构调整。首次完整失败和红绿raw均保留。

第二轮完整F的TestDeletionInterleaving24在观测锁等待时失败。单子例20次未复现；整个24矩阵20次复现一次，临时诊断明确同一事务阻塞了关联行删除worker和项目行周期任务，旧helper把全部等待者限定为一个。新夹具明确制造第二等待者，旧helper稳定选错projects；按预期表筛选候选后仍检查稳定PID、实际SQL及锁类型，24/25全矩阵连续10轮通过。只调整测试观测，不改生产事务/锁顺序/超时；临时诊断源码已撤回，证据保留。最终验证见下段。

T4最终检查点七项均通过：生成/Go lint/Go全测取t4-full3，Web lint/knip/强制全测/E2E取t4-full4。第三轮Web全测曾有旧路由接线夹具缺少detail root的10项失败，改用真实RootStore/FakeNerve后原性质保留并补guest断言，相关23项通过。服务端源输入逐文件核验不变（t4-server-input-reuse.json），没有冒称第四轮重跑Go。最终Web 240文件2587项、全部包18任务通过，E2E 180通过；所有失败raw保留。T4完成，继续T5–T14；阶段整审、三浏览器与最终分支CI仍待执行。

T5归档原store统一规范化为list/无子分组，保留合法兄弟设置、false和列表组折叠；查询最后固定archived=true。局部更新hook复用既有宿主/权限身份，消费同步及异步拒绝，列表错误仍由binding单独提示和GET重试。实际页头与列表对guest均不挂载，viewFlags作为字段只读的单一能力来源，移除重复覆盖；恢复/删除独立留T6/T7。六文件49测试通过（含project-lifecycle原行展开性质），lint-web 54通过；删除无用代码使警告上限120降119。M11/M12/M13分别2/3/1项行为红，均准确恢复；M11为既有账号隔离回归。新增测试的动画/IntersectionObserver夹具错误及flat分页条件不符已修正，失败raw保留，不计为产品缺陷。T4远端检查点CI仍待结束，尚非最终阶段CI。

T6真实归档弹窗具名称、正常Tab顺序、同步提交锁、pending禁止dismiss和当前打开的身份快照。缺失item、item/open/权限ABA、卸载和退役均不能误关闭/提示。实际详情菜单曾吞归档拒绝并报成功，恢复也缺当前宿主跟进；修复后原命令拒绝保留，modal单独反馈，再由onArchived执行导航/关闭peek。Peek既有宿主测试保留原性质、迁移到明确确认回调，并以真实菜单新用例核一次结果。交错旧归档列表触发权威补读，拒绝仍保留一次POST确认，真实Retry只重试GET。五文件60项通过、Web静态54任务通过；M14/M15/M16分别1/5/5项行为红并恢复。M16保护P12原有恢复拒绝合同；测试夹具曾错误期待已知退出活动列表必补读及误用binding属性，全部失败raw保留，最终按真实归档交错场景验证，没有改生产协调器来满足夹具。

T7在归档layout以工作区/项目/工作项key覆盖真实页头和正文；内容维持只读，恢复及按创建者/管理员判断的删除独立授权。删除拒绝原样交弹窗一次反馈，旧完成不能导航新页面。实际路由/菜单9项与原宿主/resource回归共8文件93项通过。描述编辑器在组件测试中是只读prop探针，真实输入仍留T11。M17前两次变异未失败：快速读取卸载旧菜单，掩盖短暂复用。保持返回后的GET在途，M17-held明确旧弹窗泄漏并红、恢复后绿；只有此回执计有效性质，诊断throw和早期夹具失败不计。首轮旧菜单fixture漏useUser和翻译键错误已修，保留失败raw。T4检查点2209d237的远端CI已全绿：https://github.com/open-nerve/NerveProject/actions/runs/41105350433 （t4-ci-poll4.json）；不是最终阶段CI。

T8在route显式收窄三个TProfileViews，typed props贯穿两个profile根、BaseList/BaseKanban、mount和actions；原as TProfileViews检索为零。共用mount的日历/表格可选viewId同步机械收窄，实际非profile调用仍省略。删除晚effect设置relation；请求直接接当前typed view。profile layout的工作区/用户/view key结束页头、手机与正文owner，访客工具栏和列表同门禁，M3成员摘要保留。首红14项真实行为；最终五文件48项和Web静态54任务通过，M18/M19分别5/3项行为红并准确恢复。route测试真实layout/header/mobile/sidebar与请求链，正文用带真实列表读取和popover的探针；不冒称完整ProfileIssuesPage已验，留T9/I19。guest断言一度误把测试前置workspace GET计入页面请求，已清理观测窗口；navigate重载的测试类型错误也已修正，原始输出保留。

T9把profile内存设置键补全为workspace/target，当前路由显式取对应设置；账号仍由RootStore隔离，同目标三个view共享本地设置。保存先写内存和storage，列表失败不回读覆盖；桌面、手机与富筛选复用本地更新hook消费Promise。目录只投影已持有且仍可见的项目/标签；服务端返回的冷标签组保留行、total和cursor，名称未知显示中性占位，后续目录到达可回显。实际Page/Layout/两布局14项与相关回归共10文件134项通过，store另含storage不可用、工作区隔离。M20/M21/M22分别8/7/1项行为红并逐字节恢复；M21保护既有project_id合同，不计新实现。

T9七项完整门禁均在当前产品输入运行通过：生成、Go lint、串行Go全测、Web静态54任务、knip、强制Web全测18任务（Web 247文件2676项）、E2E 180项。回执为t9-full-*；静态warning cap由119降118。初期夹具项目ID/原生分组字段和测试输入键、bound方法及hook依赖错误均保留raw，未放宽原断言；冷标签分组与未消费Promise为真实页面根因。此前T4“11属性”实际只遍历旧目录9项，T10读入口发现缺创建/更新时间，将用独立页面枚举补验与修正，不能把旧测试称为11项完整覆盖。T10–T14及最终整审/分支CI仍待完成。
