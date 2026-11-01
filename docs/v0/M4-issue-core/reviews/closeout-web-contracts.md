# M4 / 收尾 Web 合同核对

本记录是 T4 的代码与断言核对。最终 C12 三引擎视觉验收另见 [浏览器矩阵](closeout-browser-matrix.md)，最终 main 绑定仍待授权集成。原始证据根为 `.superpowers/sdd/M4-closeout/`；所有源码路径与 SHA 在 `source-read-ledger.json`，故事增长原文在 `t1-requirements.json`。

## 1. 二十二故事的实际观察

下列文件均在 `e2e/stories/issue/`。不是按题名判断通过；已读完 24 个 I-story 文件、共享 `fixtures/assert/{issue,issue-collaboration,relation}.ts` 与实际调用。PAT/API、页面、任务的适用范围不扩写。

| 故事 / 文件（省略 .spec.ts） | 实际增长断言与观察 |
|---|---|
| I1 / i1-create-issue | 默认/显式状态与负责人、完整字段、编号及项目计数；清洗/纯文本/提及/订阅/描述版本和创建动态；拒绝后数据库不变。页面填入字段后经真实 POST 和共享 SQL 断言核对 |
| I2 / i2-update-issue、i2-description | 字段差量、日期合并验证、completed_at、负责人/标签动态；描述合并和块 id 迁移不脏；真实 PATCH 保留输入/忙碌，清洗后的 HTML 与微秒级 SQL 时刻核对 |
| I3 / i3-list-groups | 原生分组/子分组结构；真实 50→51 续页，筛选后 C 分组，二维组/格总数及全部唯一 ID，不用首屏有行代替完整页 |
| I4 / i4-list-order-calendar | API 对 21 个合法排序码逐个用独立预期、limit=1 走完并核 UUID tie/count；页面表格 101 行、8 个排序方向与滚动；UTC+14 月/周边界、只拖 target_date、等待与拒绝回退。日视图另由 C12 明确核，不从故事题名推断 |
| I5 / i5-list-preferences | 两账户 11 字段偏好、date AND/me/none、PATCH 局部保留；403 不写，503 只显式重试 GET；原生 query 与独立用户设置 |
| I6 / i6-board-order | 中间/尾部 sort_order 与状态同一 PATCH；持有响应期间不乐观移动、拒绝后数据库/重新加载不变，成功仅写一次 |
| I7 / i7-quick-add | 空字段无请求、服务端 422 先说明后焦点、双 Enter 仅一个 POST；真实尾值 max(state)+10000 与数据库创建性质。API 创建沿 I1，不虚构第二版本 |
| I8 / i8-sub-issues | 无 hash 的父子双向链接；已有两个子项及新增项、父项/后代排除；等待不乐观、移动/移除后的计数和动态、键盘选取 |
| I9 / i9-relations | blocked_by 反向、单 pair、两项目动态与时刻；任一端有权可删除，隐藏项目不进入搜索；可见性不因关联扩大 |
| I10 / i10-links | URL trim/NUL/blocked 拒绝、一次 job、pending→元数据、嵌入图标；改 title 不投递、改 URL 新任务、重复不变、删除动态。危险地址零 dial 的证明来自 T3 Go，不冒称页面能观察 socket |
| I11 / i11-comments-reactions | 作者编辑、管理员只删除他人评论、独立评论/工作项表情；归档只读无 POST；IME 229 与持有响应的双新增/更新只写一次，不给评论生成工作项活动 |
| I12 / i12-activity | 101 评论+102 动态等合并为 205 条，3 页完整读取；description_id 合并/重排、last cursor 增量、重开补齐、远端过时编辑/删除、分页前隐藏引用；不是只打开首页 |
| I13 / i13-description-history | 3 版本、limit=2 两页及 20 版本保留；还原普通 PATCH、忙碌/拒绝/回焦/不额外写版本、同作者合并；真实 markdown 复制等待；访客不请求历史 |
| I14 / i14-drafts | payload 整体替换、个人归属、访客发布403、归档保留草稿；无项目51条分页、迁移字段往返；先保存再发布，409 保留且只 GET 重试，复制完整 payload 不改原稿 |
| I15 / i15-archive-issue | 归档只读控件缺席、独立本地筛选；恢复一次 POST，Back 后晚答无提示；删除全部子表、子项只读一次；访客不读归档列表但可读有权单项 |
| I16 / i16-auto-archive | PAT 只读取任务结果；跨水位旧 attempt 即使晚结束也排除，只认水位/当前进程启动之后的新 attempt；真实进程推进744小时再登录、三类对照、系统 actor 与真实恢复 actor |
| I17 / i17-delete-issue、i17-catalog | 全子树包含已归档后代；旧 tombstone 不变、新 deleter 与原作者保留；隐藏的前创建者404/访客403；catalog 新增未登记 FK 的负例确实失败 |
| I18 / i18-search | 同编号优先于标题、隐藏结果/limit、Unicode I/İ、字面%/_、中文两字；选择器五层隐藏祖先/各种关联方向；命令/顶部双宿主 late/ABA/IME；项目本地选项无额外 GET |
| I19 / i19-member-issues | A 读取 B 的三关系，A-only/C-only 对照，含 workspace admin 的 OwnOnly；局部偏好/账户及目标切换，访客桌面/窄屏无列表 GET |
| I20 / i20-guest-boundary | workspace admin+项目 guest 仍 OwnOnly；隐藏与不存在同404响应体/Content-Type/alert；允许评论写，禁止其余编辑且无隐藏内容 |
| I21 / i21-purge | 61天旧范围、近期新 tombstone/live/边界父对象完整 JSON 行；清理前记录每张 FK 的 ID，新完成 job 的时间在水位及启动之后；仅真实任务/数据库版本 |
| I22 / i22-session-switch | 创建已在 DB 提交而浏览器持有答复时，另一标签页注销/登录 B；B 新表单保持、没有 A 的 toast/跳转/行；Back 退役各宿主、Forward 不复活旧窗口。仅页面版本 |

M3 删除故事沿既有有限轮询和相同共享断言；T3 的删除窗口已逐读/写入口核 absent 响应。未改 watchPage 的宽容范围、未清空原始错误数组、未新增忽略。

## 2. 等待后的会话与宿主

扫描取自完整 M4 范围，`t4-async-hook-candidates.json` 有 25 个 hook 候选，`t4-async-follower-files.json` 扩至 60 个调用文件。扫描是入口清单，结论来自全文及调用方。`useDetailScope` 只管本地宿主寿命，权限代际始终由 IssueRootStore 持有；登录更替退役旧 root。通用 sessionGuard/followInSession 仍是既有会话保护。

下列路径相对 `web/apps/web/core/`；同一守卫的多个调用入口分别映射实际消费者测试，并以 I22 的真实跨标签登录核 root 退役连线。

| hook / 调用方 | 守卫与具名行为证据 |
|---|---|
| hooks/use-copy-link 的 useCopyLink/useCopyText/useCopyProjectLink | 本次补 sessionGuard，成功与失败都先核原 loginId；`use-copy-link.test.ts` 三入口×两结果×1/2次账户变化，保留新会话提示，write 恰一次；collaboration-copy/detail-copy 测真实十个消费者晚答 |
| hooks/use-issues-actions | 写回归属由 root scope/session 控制；本次修归档 remove/restore/updateFilters 漏 workspaceSlug、fetchNext/useMemo 依赖；`use-issue-layout-store.test.tsx` 的 archive 三操作 rerender 到 other/B，真实请求路径及旧/新本地筛选断言；不声称所有路由都复用同一次挂载 |
| hooks/use-issue-list-mount | `use-issue-list-mount.integration.test.tsx`：mounts the list under each login and discards the prior session's late HTTP answer；x/y 真实两 root 与 HTTP，旧值不进任一新缓存；另测 guest archive/profile 零 GET、权限 ABA 与共享挂载 |
| hooks/use-project-preferences-follow | `use-project-preferences-follow.test.tsx` 的 session/route/permission ABA、unmount；`project-preferences-refresh.integration.test.tsx` 的 does not refresh the old list when session changes while saving，真实 PATCH 后不再读旧列表 |
| hooks/use-local-issue-filter-update | capture+followInSession；对应同名测试核旧拒绝静默、目标/权限 ABA、unmount/root retirement；局部筛选不另建权限 owner |
| hooks/use-group-dragndrop | 对应同名测试 actual hook and actions issue delayed PATCH; session change suppresses refusal toast；真实 root/HTTP，受拒写不污染 canonical |
| hooks/use-issue-list-refresh | 同名测试的旧 root 退役、迟到答复/旧 retry 静默；可见组才刷新、拒绝显式 retry；错误提示仍经 followInSession/current |
| hooks/use-issue-search | 请求/权限 proof 双检查，冷目录也先判权限；同名测试旧范围/权限 ABA/root retirement 无 canonical，拒绝保留可重试；I18 双宿主实际迟到读 |
| hooks/editor/use-editor-mention | 同名测试核旧 query/root/权限/卸载、current directory retry；scope adapter 不读邮箱，不回退越权目录；mentions-list-dropdown 在 debounce 起点取消旧请求，选择前重验 scope |
| components/editor/use-confirmed-autosave | 按 owner+revision 跟随；`issues/autosave.test.tsx` 的 ignores a late answer after the editor owner has been replaced、旧确认不覆盖新输入、拒绝显式重试、卸载不重送；prepareSave 由 useIssueOperations 的登录/权限保护，I22 核真实换账号入口 |
| issues/issue-detail/use-issue-operations（useDetailCommand/useIssueOperations） | run 在 command 前/后核 current+session，prepareUpdate 在发出和完成各核；`detail-commands.test.tsx` 的 does not toast or resolve an old caller successfully after unmount；restore-menu 三入口拒绝/确认/旧宿主/权限 ABA；未把没有复现的微任务间隙推测当缺陷 |
| issues/issue-detail-widgets/directory-read | 请求 owner/current root；选择器关闭/换权限后不发布目录，permissionReads 为唯一事实来源；对应 use-issue-search/editor-mention 的目录交错与 detail 控件集成覆盖 |
| issue-detail-widgets/{links,relations,sub-issues}/helper、issue-detail/issue-activity/helper | 统一 run/follow/capture，调用方 await 后由宿主决定 close；`detail-commands.test.tsx` 的四类 follows one confirmation without duplicate activity reads 与旧卸载静默，子项 loader 拒绝释放；I8/I9/I11/I12 的真实请求及无重复读取 |
| issue-detail-widgets/links/use-link-metadata-refresh | 同名测试旧根/链接替换/权限退役后不轮询、不跟随旧结果；刷新不引入新写，I10 记录实际页面元数据更新 |
| issues/issue-layouts/command-host、quick-action-dropdowns/helper | 命令 owner+当前 scope；`route-scope.test.tsx` 旧项目/旧宿主命令不能结束新宿主；restore-menu/removal 记录拒绝及确认；无事件处理器裸拒绝 |
| issues/issue-layouts/quick-add/root | `root.pending.test.tsx` retired session ignores late quick-add resolve/reject followers，A结果不能重置 B/释放B忙碌，unmount无本地效果；focus微任务再核当前会话 |
| issues/archive-issue-modal、delete-issue-modal | current+follow；`delete-issue-modal.test.tsx` does not follow an old session's deletion settlement 两种结果、同轮双提交+关闭只发一次；archive-confirmation 同一请求 owner；I15/C9 核 Back 后窗口不写 |
| issues/issue-modal/base/form、workspace-draft/delete-modal | 保存前/每次 await 后 capture+session/root；base 抛出拒绝防子表单 reset；draft delete 旧 root 不跟随；I14 存储payload/冲突/复制，I22 真账号晚答 |
| comments/comment-create、card/edit-form、core/description-versions/modal | followInSession+宿主 capture；上传后再核 root（M4 上传关闭）；历史确认后才替换编辑器，复制反馈由新 hook 保护；I11/I13/C6 及复制消费者测试 |
| labels/use-label-changes | 同名测试五类 change×success/refusal 的 does nothing on the page when it settles after another tab moved this one；无 done/字段错误/toast；M4 可选 capture 同时限制旧弹层 |
| power-k/config/preferences-commands | 同名测试真实 ProfileStore/HTTP，连续两次 switchAccount 后 success/refusal 无 theme/reload/toast；C7 真实另一标签登录后旧主题成功也不重载 |
| power-k/.../copy-commands、mutation-commands | command host/当前对象 proof；copy 复用新共享 hook；I18/C7 双搜索宿主、多选/IME/旧范围，主题等偏好等待后再核会话 |
| rich-filters/.../use-filter-select-options | 同名测试旧 config 的成功/失败及卸载不落入新选项；组件 effect owner 只隔离本地查询，不自建权限事实 |
| profile/start-of-week-preference | 沿 followInSession；偏好真实写与旧登录行为由 settings profile preferences 和 C7 共享合同保护；不改变日期模型 |

`issue-layouts/utils.tsx` 的 useGroupByOptions 是同步 hook，同文件普通 async 拖放函数只返回结果；`issue-modal/fake-form.tsx` 是明确导入 Vitest 的测试夹具。`use-editor-config`、editor/use-file-upload、custom-image 的上传能力归 M5，M4 editor-capabilities 禁止上传入口。cycles/modules/inbox/views 自身页面归 M6/M7，不以范围外的旧请求扩写本期产品；M4 撤除入口由 S2 专门验。上述排除保留原路径和原因，未按文件名静默忽略。

## 3. S1/S2 与旧实例

S1 经真实 `/health`、`/ready`、连接 application_name、每个迁移文件及 max(version) 核启动。S2 每次冷加载收原始 method/path/query，requestfinished/failed 后 pending 必须为空；精确数组含合法重复数（项目偏好两个消费者的两次 GET），不是去重后比集合。

| S2 入口 | 当前可见性/请求性质 |
|---|---|
| admin/member/guest/project-nonmember | 每角色逐页冷加载，工作区/项目/状态/标签/菜单；workspace menu 开闭不取数，Escape 真卸载 |
| project list 及四布局 | 原生 GET，布局变更一 PATCH 偏好+一 GET；board state/30、table/100、calendar月范围/4、list state/50；字体就绪后才下一导航，防首屏字体被测试自己中止 |
| profile 三关系 | assigned/created/subscribed 原生查询，无多余 label/preferences；访客页面/直接tab都拒绝且不读列表 |
| archive / draft | archived=true 且本地筛选；访客无列表/创建请求；已归档项目只读，不把权限不足当空数据 |
| detail | member200/guest404/nonmember403 对应完整错误页，不泄露内容；错误数组保留所期望的精确响应 |
| M6/M7 六入口 | s2-issue-list-removal 精确允许各自尚未实现的一个旧404及对应 console URL；无 M4 列表/工具栏。归档 M6 loader 如实登记，不能签领域自身成功；请求原数组与 finally 附件保留 |

`t4-module-services.json` 使用仓库锁定 TypeScript AST 解析 1329 个 app 业务 TS/TSX，查模块级 variable→new *Service；M2 handoff 的原 13 路径都存在且为零。剩余8处分别为 FileService×3（M5）、CycleService×2（M6）、IntakeWorkItemVersionService×1（M7）、TimezoneService、WorkspaceNotificationService（自身领域），不等同于全仓为零。`services/workspace.service.test.ts` 精确保留 M7 四个 view transport/错误体，searchWorkspace 不再存在。

## 4. 本次修复与验证

- 复制缺少 await 后登录检查：`t4-copy-session-red` 的12个跨账号用例真实多出旧 toast；修复后四文件67项通过，原用户提示不变。外层 followInSession 无法屏蔽内部已经发出的 toast，因此根因在共享复制 hook。
- 归档回调闭包缺 workspaceSlug：`t4-archive-route-red` 的删除/恢复/折叠3项真实失败。仅补依赖及分页 callback/memo 的正确依赖；`t4-archive-route-green-2` 七文件123项通过。第一次 green 中快照工具误用 structuredClone，触发 DataCloneError；已用实际序列化值比较修正，该次不算业务反证。
- 全套类型检查找到测试自己的 void/undefined 和缺少 sub_group_by，按原类型修正。全量 Web 测试又找到两个原 SSR 渲染测试缺会话注入，复用 fake-tab 补 transport 边界；生产仍 ssr=false，不添加环境分支或放宽任何类型。

本地七门禁均已通过。源码生成5.29s、Go lint24.38s、完整Go457.38s、Web lint11.15s（54/54）、knip3.57s、Web全测39.64s（18/18任务，缓存0）、默认E2E105.80s（284通过，0失败/跳过）。三个早期Go/生成输入与最终输入仅相差四个Web测试夹具，knip仅相差两条测试边界mock；逐文件等价校验见 `t4-f-gates.json`。全部raw、失败尝试、输入SHA和默认E2E报告/trace已保留。准确检查点 `7f58273592ec0e9d488b2d87501a40b9c83b1f57` 的 [CI 41300361270](https://github.com/open-nerve/NerveProject/actions/runs/41300361270) attempt 1 三作业全部 success：web 124829832384、server 124829832486、e2e 124832479514。公开 REST 的 head_sha 与连接器 jobs 已分别保留于 `t4-ci-run-final.json` / `t4-ci-jobs-final.json`，精简收据 `t4-ci-receipt.json`。最终C12及main尚未签结。
