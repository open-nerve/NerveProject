# M4/P13 筛选、归档与个人主页：预检记录

日期：2026-10-31。**当前已获准实施，T1完成，T2接续；以下§1–7保留预检快照，实施接续见§8。** [spec](../specs/P13-web-filters-archives-profile.md)、[plan](../plans/P13-web-filters-archives-profile.md)、[deep-review预检](../../../reviews/re-18-M4P13预检审核.md)。产品A1–A12和14个Task全部开放；本记录不等于阶段实施后的整分支终审。

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
